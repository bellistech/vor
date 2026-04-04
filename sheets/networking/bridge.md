# Linux Bridge

Software Layer 2 switch in the Linux kernel that forwards frames between attached interfaces based on MAC address learning, supporting STP/RSTP, VLAN filtering, and integration with containers, VMs, and network namespaces.

## Bridge Basics

```bash
# Create a bridge with ip link
ip link add br0 type bridge
ip link set br0 up

# Add interfaces to the bridge
ip link set eth0 master br0
ip link set eth1 master br0

# Assign IP to the bridge (for management / L3 routing)
ip addr add 192.168.1.1/24 dev br0

# Remove interface from bridge
ip link set eth0 nomaster

# Delete bridge
ip link del br0

# Show bridge info
ip link show type bridge
ip -d link show br0

# Show bridge ports
bridge link show
```

## brctl (Legacy bridge-utils)

```bash
# Create bridge (deprecated — use ip link)
brctl addbr br0

# Add interface
brctl addif br0 eth0

# Show bridges
brctl show

# Show MAC address table (FDB)
brctl showmacs br0

# Set STP on/off
brctl stp br0 on
brctl stp br0 off

# Set bridge parameters
brctl setfd br0 4          # Forward delay (seconds)
brctl sethello br0 2       # Hello interval (seconds)
brctl setmaxage br0 20     # Max age (seconds)

# Set port priority
brctl setportprio br0 eth0 32

# Set bridge priority (lower = more likely to be root)
brctl setbridgeprio br0 8192

# Delete bridge
brctl delbr br0
```

## MAC Address Table (FDB)

```bash
# Show FDB (forwarding database)
bridge fdb show br br0

# Show specific interface FDB
bridge fdb show dev eth0

# Add static FDB entry
bridge fdb add 00:11:22:33:44:55 dev eth0 master static

# Delete FDB entry
bridge fdb del 00:11:22:33:44:55 dev eth0 master

# Flush all dynamic FDB entries
bridge fdb flush dev br0

# Set aging time (seconds, 0 = no aging)
ip link set br0 type bridge ageing_time 300
```

## STP / RSTP Configuration

```bash
# Enable STP on bridge
ip link set br0 type bridge stp_state 1

# Bridge priority (0-65535, default 32768)
ip link set br0 type bridge priority 8192

# Forward delay (default 15 sec, min 2 sec)
ip link set br0 type bridge forward_delay 400  # centiseconds

# Hello time (default 2 sec)
ip link set br0 type bridge hello_time 200  # centiseconds

# Max age (default 20 sec)
ip link set br0 type bridge max_age 2000  # centiseconds

# Port cost (lower = preferred path)
bridge link set dev eth0 cost 100

# Port priority (0-63, default 32)
bridge link set dev eth0 priority 16

# Show STP state
bridge link show dev eth0
# state: disabled, listening, learning, forwarding, blocking

# Show bridge STP topology
cat /sys/class/net/br0/bridge/root_id
cat /sys/class/net/br0/bridge/bridge_id
cat /sys/class/net/br0/bridge/topology_change
```

## VLAN Filtering

```bash
# Enable VLAN filtering on bridge
ip link set br0 type bridge vlan_filtering 1

# Add VLAN to port (tagged)
bridge vlan add vid 100 dev eth0

# Add VLAN as untagged (access port)
bridge vlan add vid 100 dev eth0 pvid untagged

# Add VLAN to bridge itself (for bridge-local traffic)
bridge vlan add vid 100 dev br0 self

# Add a range of VLANs
bridge vlan add vid 100-200 dev eth0

# Delete VLAN from port
bridge vlan del vid 100 dev eth0

# Show VLAN configuration
bridge vlan show

# Show per-port VLAN info
bridge vlan show dev eth0

# Set default PVID for all new ports
ip link set br0 type bridge vlan_default_pvid 1

# Trunk port (multiple tagged VLANs)
bridge vlan add vid 100 dev eth0
bridge vlan add vid 200 dev eth0
bridge vlan add vid 300 dev eth0
```

## bridge-nf-call (Netfilter Integration)

```bash
# Bridge packets pass through iptables when enabled
# This is needed for Docker/K8s but can cause issues

# Check current settings
sysctl net.bridge.bridge-nf-call-iptables
sysctl net.bridge.bridge-nf-call-ip6tables
sysctl net.bridge.bridge-nf-call-arptables

# Enable (required for Kubernetes)
sysctl -w net.bridge.bridge-nf-call-iptables=1
sysctl -w net.bridge.bridge-nf-call-ip6tables=1

# Disable (pure L2 bridge without netfilter overhead)
sysctl -w net.bridge.bridge-nf-call-iptables=0

# Persist
echo "net.bridge.bridge-nf-call-iptables = 1" >> /etc/sysctl.d/bridge.conf
echo "net.bridge.bridge-nf-call-ip6tables = 1" >> /etc/sysctl.d/bridge.conf
sysctl -p /etc/sysctl.d/bridge.conf

# Load br_netfilter module (required for sysctl to work)
modprobe br_netfilter
echo "br_netfilter" > /etc/modules-load.d/br_netfilter.conf
```

## Docker Bridge Networking

```bash
# Default docker0 bridge
ip link show docker0
bridge link show master docker0
docker network inspect bridge

# Create custom bridge network
docker network create --driver bridge \
  --subnet 172.20.0.0/16 \
  --gateway 172.20.0.1 \
  --opt com.docker.network.bridge.name=br-custom \
  my-network

# Run container on custom bridge
docker run --network my-network --name web nginx

# Connect container to additional bridge
docker network connect my-network existing-container

# Inspect bridge for a network
docker network inspect my-network | jq '.[0].Options'

# Docker bridge internals
# - Each container gets a veth pair: one end in container, one on bridge
# - iptables NAT for outbound (MASQUERADE)
# - iptables FORWARD rules for inter-container traffic
iptables -t nat -L POSTROUTING -v -n | grep docker
iptables -L DOCKER -v -n

# Disable inter-container communication
docker network create --driver bridge \
  --opt com.docker.network.bridge.enable_icc=false \
  isolated-net
```

## KVM/libvirt Bridged Networking

```bash
# Create bridge for VMs
ip link add virbr-ext type bridge
ip link set virbr-ext up
ip link set enp3s0 master virbr-ext

# Move IP from physical NIC to bridge
ip addr del 192.168.1.100/24 dev enp3s0
ip addr add 192.168.1.100/24 dev virbr-ext
ip route add default via 192.168.1.1 dev virbr-ext

# libvirt bridge definition (XML)
# /etc/libvirt/qemu/networks/bridged.xml
# <network>
#   <name>bridged</name>
#   <forward mode="bridge"/>
#   <bridge name="virbr-ext"/>
# </network>

virsh net-define /etc/libvirt/qemu/networks/bridged.xml
virsh net-start bridged
virsh net-autostart bridged

# Attach VM to bridge
virsh attach-interface --domain myvm --type bridge \
  --source virbr-ext --model virtio --persistent

# netplan bridge config (Ubuntu)
# /etc/netplan/01-bridge.yaml
# network:
#   version: 2
#   ethernets:
#     enp3s0:
#       dhcp4: false
#   bridges:
#     br0:
#       interfaces: [enp3s0]
#       addresses: [192.168.1.100/24]
#       routes:
#         - to: default
#           via: 192.168.1.1
#       nameservers:
#         addresses: [8.8.8.8, 8.8.4.4]
```

## Bridge with Network Namespaces

```bash
# Create bridge and namespaces
ip link add br0 type bridge
ip link set br0 up

# Create namespace and veth pair
ip netns add ns1
ip link add veth-ns1 type veth peer name veth-br1

# Move one end to namespace, other to bridge
ip link set veth-ns1 netns ns1
ip link set veth-br1 master br0

# Configure addresses
ip netns exec ns1 ip addr add 10.0.0.2/24 dev veth-ns1
ip netns exec ns1 ip link set veth-ns1 up
ip link set veth-br1 up
ip addr add 10.0.0.1/24 dev br0

# Verify connectivity
ip netns exec ns1 ping 10.0.0.1
```

## Tips

- Use `ip link` and `bridge` commands over `brctl`; bridge-utils is deprecated and lacks VLAN support
- Enable VLAN filtering (`vlan_filtering 1`) for proper 802.1Q trunk/access port behavior on Linux bridges
- Set `bridge-nf-call-iptables=1` and load `br_netfilter` for Kubernetes; CNI plugins depend on this
- Lower the bridge priority (e.g., 8192) if you want a specific bridge to become the STP root
- Reduce `forward_delay` to 2-4 seconds if STP convergence time is critical and your topology is simple
- Assign an IP address to the bridge interface itself (not its member ports) for management access
- Watch out for MAC flapping in bridge FDB logs; it usually indicates a loop or miscabled redundant path
- Use `bridge monitor` to watch FDB, VLAN, and link state changes in real time
- For Docker custom bridges, always specify `--subnet` to avoid IP conflicts with other Docker networks
- When bridging a physical NIC for VMs, move the IP and default route to the bridge before adding the NIC
- Set `ageing_time 0` on bridges used for monitoring/SPAN ports to keep all MAC entries permanent
- Use `bridge vlan show` regularly to audit which VLANs are allowed on each port

## See Also

- vlan, ip, macvlan, netns, docker, kvm, iptables, stp

## References

- [bridge(8) man page](https://man7.org/linux/man-pages/man8/bridge.8.html)
- [ip-link(8) — bridge type](https://man7.org/linux/man-pages/man8/ip-link.8.html)
- [Linux Bridge — Kernel Documentation](https://docs.kernel.org/networking/bridge.html)
- [IEEE 802.1D — STP](https://standards.ieee.org/ieee/802.1D/5765/)
- [IEEE 802.1Q — VLAN Bridges](https://standards.ieee.org/ieee/802.1Q/10323/)
- [Docker Bridge Networking](https://docs.docker.com/network/drivers/bridge/)
