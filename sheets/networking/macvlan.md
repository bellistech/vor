# macvlan / ipvlan

Virtual network interfaces that allow multiple logical interfaces with distinct MAC addresses (macvlan) or shared MAC with distinct IPs (ipvlan) on a single physical parent, supporting bridge, VEPA, private, and passthru modes.

## macvlan Modes

```
Bridge Mode (default):
  Containers/VMs can communicate directly via the parent interface
  Packets between macvlans are switched locally (no external switch needed)

  ┌──────────────────────────────────┐
  │         Parent: eth0             │
  ├──────┬──────┬──────┬────────────┤
  │ mv0  │ mv1  │ mv2  │   eth0     │
  │ MAC_A│ MAC_B│ MAC_C│   MAC_0    │
  └──┬───┴──┬───┴──┬───┴─────┬─────┘
     │      │      │         │
   Can talk to each other   Cannot talk to
   directly                 parent (eth0 IP)

VEPA Mode (Virtual Ethernet Port Aggregator):
  All traffic goes to external switch, even between macvlans
  Requires 802.1Qbg-capable switch (hairpin/reflective relay)

  mv0 ──┐                 ┌── mv0
  mv1 ──┼── eth0 ── switch ──┤
  mv2 ──┘    (all out)     └── mv1 (reflected back)

Private Mode:
  macvlans are completely isolated — no inter-macvlan traffic
  Can only communicate with external hosts

  mv0 ──X── mv1    mv0 ───── external host
  mv1 ──X── mv2    mv1 ───── external host

Passthru Mode:
  Single macvlan gets exclusive access to parent
  Used for SR-IOV VF assignment and DPDK
  Only one macvlan allowed per parent
```

## Creating macvlan Interfaces

```bash
# Bridge mode (default)
ip link add mv0 link eth0 type macvlan mode bridge
ip link set mv0 up
ip addr add 192.168.1.100/24 dev mv0

# VEPA mode
ip link add mv-vepa link eth0 type macvlan mode vepa

# Private mode
ip link add mv-priv link eth0 type macvlan mode private

# Passthru mode
ip link add mv-pass link eth0 type macvlan mode passthru

# Specify MAC address
ip link add mv0 link eth0 type macvlan mode bridge
ip link set mv0 address 02:42:ac:11:00:02

# Show macvlan interfaces
ip -d link show type macvlan

# Delete
ip link del mv0
```

## macvlan in Network Namespaces

```bash
# Create namespace
ip netns add ns1

# Create macvlan and move to namespace
ip link add mv-ns1 link eth0 type macvlan mode bridge
ip link set mv-ns1 netns ns1

# Configure inside namespace
ip netns exec ns1 ip addr add 192.168.1.101/24 dev mv-ns1
ip netns exec ns1 ip link set mv-ns1 up
ip netns exec ns1 ip route add default via 192.168.1.1

# Verify
ip netns exec ns1 ip addr show
ip netns exec ns1 ping 192.168.1.1

# Multiple namespaces on same parent
for i in 1 2 3; do
  ip netns add ns$i
  ip link add mv-ns$i link eth0 type macvlan mode bridge
  ip link set mv-ns$i netns ns$i
  ip netns exec ns$i ip addr add 192.168.1.10$i/24 dev mv-ns$i
  ip netns exec ns$i ip link set mv-ns$i up
  ip netns exec ns$i ip route add default via 192.168.1.1
done
```

## ipvlan Modes

```
ipvlan L2 Mode:
  Shares parent MAC, switches at L2 (like macvlan bridge but one MAC)
  ARP/NDP handled by ipvlan driver
  All sub-interfaces share parent's MAC address

ipvlan L3 Mode:
  Routes at L3 — no ARP, no broadcast
  Parent acts as router for sub-interfaces
  External router needs static routes to ipvlan subnets

ipvlan L3S Mode:
  L3 mode + source address validation
  Works with netfilter/iptables (conntrack)
  Recommended for containers needing iptables

Comparison:
  Feature          macvlan          ipvlan
  MAC address      Unique per sub   Shared (parent MAC)
  Switch flood     One per MAC      None (one MAC)
  ARP entries      One per sub      One total
  Max per parent   Limited by MAC   Unlimited (IP only)
  Cloud compat     Often blocked    Works everywhere
  iptables         Full support     L3S only
```

## Creating ipvlan Interfaces

```bash
# ipvlan L2 (similar to macvlan bridge)
ip link add ipv0 link eth0 type ipvlan mode l2
ip link set ipv0 up
ip addr add 192.168.1.100/24 dev ipv0

# ipvlan L3 (routed, no ARP)
ip link add ipv0 link eth0 type ipvlan mode l3
ip link set ipv0 up
ip addr add 10.0.1.1/24 dev ipv0

# ipvlan L3S (L3 + netfilter source validation)
ip link add ipv0 link eth0 type ipvlan mode l3s
ip link set ipv0 up
ip addr add 10.0.1.1/24 dev ipv0

# Show ipvlan interfaces
ip -d link show type ipvlan
```

## Docker macvlan Network

```bash
# Create macvlan network in Docker
docker network create -d macvlan \
  --subnet=192.168.1.0/24 \
  --gateway=192.168.1.1 \
  -o parent=eth0 \
  my-macvlan

# Run container on macvlan network
docker run --network my-macvlan \
  --ip 192.168.1.200 \
  --name web nginx

# Container gets IP directly on the LAN — no NAT, no port mapping

# macvlan with VLAN sub-interface (802.1Q trunk)
docker network create -d macvlan \
  --subnet=10.10.100.0/24 \
  --gateway=10.10.100.1 \
  -o parent=eth0.100 \
  vlan100-net

# Multiple VLAN networks
docker network create -d macvlan \
  --subnet=10.10.200.0/24 \
  --gateway=10.10.200.1 \
  -o parent=eth0.200 \
  vlan200-net

# Docker ipvlan network
docker network create -d ipvlan \
  --subnet=192.168.1.0/24 \
  --gateway=192.168.1.1 \
  -o parent=eth0 \
  -o ipvlan_mode=l2 \
  my-ipvlan
```

## Host-to-macvlan Communication

```bash
# Problem: parent interface (eth0) CANNOT talk to its macvlan children
# Solution: create a macvlan on the host too

# Create host-side macvlan
ip link add mv-host link eth0 type macvlan mode bridge
ip addr add 192.168.1.250/32 dev mv-host
ip link set mv-host up

# Route container subnet through host macvlan
ip route add 192.168.1.200/32 dev mv-host

# Or for Docker: add route for the entire macvlan subnet
ip route add 192.168.1.192/26 dev mv-host

# Persist with systemd-networkd
# /etc/systemd/network/20-mv-host.netdev
# [NetDev]
# Name=mv-host
# Kind=macvlan
#
# [MACVLAN]
# Mode=bridge
#
# /etc/systemd/network/20-mv-host.network
# [Match]
# Name=mv-host
# [Network]
# Address=192.168.1.250/32
# [Route]
# Destination=192.168.1.192/26
```

## Kubernetes CNI with macvlan/ipvlan

```bash
# Multus CNI allows multiple network interfaces per pod
# macvlan as secondary network via NetworkAttachmentDefinition

# Install Multus
kubectl apply -f https://raw.githubusercontent.com/k8snetworkplumbingwg/multus-cni/master/deployments/multus-daemonset.yml

# NetworkAttachmentDefinition for macvlan
cat <<'EOF' | kubectl apply -f -
apiVersion: k8s.cni.cncf.io/v1
kind: NetworkAttachmentDefinition
metadata:
  name: macvlan-conf
spec:
  config: '{
    "cniVersion": "0.3.1",
    "type": "macvlan",
    "master": "eth0",
    "mode": "bridge",
    "ipam": {
      "type": "host-local",
      "subnet": "192.168.1.0/24",
      "rangeStart": "192.168.1.200",
      "rangeEnd": "192.168.1.250",
      "gateway": "192.168.1.1"
    }
  }'
EOF

# Pod with macvlan secondary interface
cat <<'EOF' | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: samplepod
  annotations:
    k8s.v1.cni.cncf.io/networks: macvlan-conf
spec:
  containers:
  - name: app
    image: alpine
    command: ["sleep", "infinity"]
EOF

# Verify — pod will have eth0 (cluster) + net1 (macvlan)
kubectl exec samplepod -- ip addr show
```

## macvlan vs bridge vs ipvlan Comparison

```
Scenario                      Recommended
─────────────────────────── ──────────────────────
Containers need LAN IPs       macvlan bridge
Cloud VM (MAC filtering)      ipvlan L2 or L3S
Maximum isolation             macvlan private
Performance (< overhead)      macvlan bridge
Many sub-interfaces (>256)    ipvlan (no MAC limit)
iptables/conntrack needed     ipvlan L3S
VM bridging with STP          Linux bridge
Container-to-host comms       Linux bridge or host macvlan
SR-IOV passthrough            macvlan passthru
```

## Tips

- macvlan containers cannot communicate with the parent host IP; create a host-side macvlan interface as a workaround
- Use ipvlan instead of macvlan in cloud environments (AWS, GCP); most cloud providers filter unknown MAC addresses
- macvlan bridge mode is fastest for container-to-container on the same host; no external switch needed
- Set the macvlan mode to private for maximum tenant isolation; containers can only reach external networks
- Docker macvlan containers get real LAN IPs with no NAT; this simplifies service discovery but requires IP planning
- Use VLAN sub-interfaces (`eth0.100`) as macvlan parents to segment traffic by VLAN on a single NIC
- ipvlan L3 mode requires static routes on the upstream router; there is no ARP for the ipvlan addresses
- Limit macvlan interfaces per parent; each adds a MAC address and some switches limit MACs per port (default ~1024)
- Enable promiscuous mode on the parent (`ip link set eth0 promisc on`) if macvlan traffic is not being received
- For Kubernetes, use Multus CNI with macvlan as a secondary network; the primary CNI handles cluster networking
- macvlan passthru mode is ideal for SR-IOV VF assignment; it gives the VM exclusive access to the hardware queue
- Monitor with `ip -s link show` on macvlan interfaces; high drop counters may indicate switch MAC table overflow

## See Also

- bridge, vlan, netns, docker, kubernetes, ip, veth

## References

- [macvlan — Kernel Documentation](https://docs.kernel.org/networking/ipvlan.html)
- [ipvlan — Kernel Documentation](https://docs.kernel.org/networking/ipvlan.html)
- [Docker macvlan Network Driver](https://docs.docker.com/network/drivers/macvlan/)
- [Multus CNI](https://github.com/k8snetworkplumbingwg/multus-cni)
- [macvlan vs ipvlan — Linux Networking](https://hicu.be/macvlan-vs-ipvlan)
- [ip-link(8) — macvlan/ipvlan](https://man7.org/linux/man-pages/man8/ip-link.8.html)
