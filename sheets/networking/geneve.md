# Geneve (Generic Network Virtualization Encapsulation)

Network virtualization overlay protocol (RFC 8926) that encapsulates Layer 2 frames inside UDP for transport across an IP underlay, using a flexible TLV option header to carry metadata between virtual switches. Designed as the extensible successor to VXLAN and NVGRE, Geneve is the native tunnel type for Open vSwitch and the primary overlay in OpenStack and Kubernetes networking.

---

## Geneve Header Format

```
Outer Ethernet Header (14 bytes)
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|         Dst MAC (6)           |         Src MAC (6)           |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|       EtherType (0x0800 or 0x86DD)                            |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

Outer IP Header (20 bytes IPv4 / 40 bytes IPv6)
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|  ...standard IP header...  Protocol: UDP (17)                 |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

Outer UDP Header (8 bytes)
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|     Source Port (hash)        |   Dest Port = 6081            |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|     UDP Length                 |   UDP Checksum                |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

Geneve Fixed Header (8 bytes)
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|Ver(2)|Opt Len(6)|O|C| Rsvd(6)|     Protocol Type (16)        |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|             Virtual Network Identifier (VNI) (24)     | Rsvd  |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

Geneve TLV Options (variable, 0-252 bytes, 4-byte aligned)
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| Option Class (16)             |     Type (8)      | R|R|R| Len|
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                    Variable Option Data                        |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

Inner Ethernet Frame (original L2 frame)
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|  Inner Dst MAC | Inner Src MAC | Inner EtherType | Payload... |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

Field Details:
  Ver:           0 (current version)
  Opt Len:       Length of options in 4-byte multiples (0-63 = 0-252 bytes)
  O (OAM):       1 = OAM packet (control plane, not data)
  C (Critical):  1 = critical options present (must be understood)
  Protocol Type: 0x6558 = Transparent Ethernet Bridging
  VNI:           24-bit Virtual Network Identifier (16M segments)
  UDP Src Port:  Hash of inner frame headers (for ECMP)
  UDP Dst Port:  6081 (IANA assigned)
```

## Geneve vs VXLAN Comparison

```
Feature                  Geneve (RFC 8926)       VXLAN (RFC 7348)
──────────────────────   ─────────────────────   ─────────────────────
Header size (min)        50 bytes (no options)   50 bytes (fixed)
Header extensibility     TLV options (0-252B)    None (8 reserved bytes)
UDP destination port     6081                    4789
VNI size                 24 bits (16M)           24 bits (16M)
OAM flag                 Yes                     No
Critical bit             Yes                     No
Protocol type field      Yes (multi-protocol)    No (Ethernet only)
ECMP hashing             UDP src port (entropy)  UDP src port (entropy)
Control plane            Flexible (OVS, EVPN)    EVPN, flood-and-learn
Hardware offload         Growing (mlx5, i40e)    Widely supported
Inner payload            Any (via protocol type) Ethernet only

# Both use outer UDP for NAT/firewall traversal and ECMP load balancing
# Geneve's TLV options are what make it the future-proof choice
# VXLAN's reserved bytes were never standardized for metadata
```

## Linux Geneve Tunnel Setup

```bash
# Create a geneve tunnel interface
ip link add geneve100 type geneve id 100 remote 10.0.0.2 dstport 6081

# Bring it up and assign an IP
ip link set geneve100 up
ip addr add 192.168.100.1/24 dev geneve100

# On the remote side (10.0.0.2):
ip link add geneve100 type geneve id 100 remote 10.0.0.1 dstport 6081
ip link set geneve100 up
ip addr add 192.168.100.2/24 dev geneve100

# Point-to-multipoint (learning mode, no explicit remote)
ip link add geneve200 type geneve id 200 dstport 6081

# Set TTL for outer IP header (default 0 = inherit inner)
ip link add geneve100 type geneve id 100 remote 10.0.0.2 ttl 64

# Set TOS for outer IP header (inherit from inner)
ip link add geneve100 type geneve id 100 remote 10.0.0.2 tos inherit

# Enable UDP checksum on outer header (recommended)
ip link add geneve100 type geneve id 100 remote 10.0.0.2 \
    udp6zerocsumtx udp6zerocsumrx

# View geneve interface details
ip -d link show geneve100

# View geneve FDB entries (learned or static)
bridge fdb show dev geneve100

# Add static FDB entry for known destination
bridge fdb add 00:11:22:33:44:55 dev geneve100 dst 10.0.0.3

# Delete FDB entry
bridge fdb del 00:11:22:33:44:55 dev geneve100

# Check kernel geneve module is loaded
lsmod | grep geneve
modprobe geneve
```

## Geneve with Linux Bridge

```bash
# Create bridge with geneve tunnel as a port
ip link add br-geneve type bridge
ip link set br-geneve up

# Create geneve tunnel
ip link add geneve100 type geneve id 100 remote 10.0.0.2 dstport 6081
ip link set geneve100 up

# Attach geneve tunnel and local interface to bridge
ip link set geneve100 master br-geneve
ip link set eth1 master br-geneve

# Assign management IP to bridge
ip addr add 192.168.100.1/24 dev br-geneve

# Verify bridge ports
bridge link show

# Enable STP on the bridge (optional for loop prevention)
ip link set br-geneve type bridge stp_state 1

# Bridge learning + geneve = distributed L2 segment across IP fabric
```

## Open vSwitch Geneve Tunnels

```bash
# OVS uses Geneve as its default tunnel type (since OVS 2.6+)

# Create OVS bridge
ovs-vsctl add-br br-int

# Add geneve tunnel port (flow-based, VNI set per-flow)
ovs-vsctl add-port br-int geneve0 -- \
    set interface geneve0 type=geneve \
    options:remote_ip=flow \
    options:key=flow

# Add geneve tunnel port (fixed remote, fixed VNI)
ovs-vsctl add-port br-int tun-node2 -- \
    set interface tun-node2 type=geneve \
    options:remote_ip=10.0.0.2 \
    options:key=100

# Add geneve tunnel with TLV options (OVS Geneve metadata)
ovs-vsctl add-port br-int geneve0 -- \
    set interface geneve0 type=geneve \
    options:remote_ip=flow \
    options:key=flow \
    options:csum=true

# Configure Geneve TLV option mapping in OVS
# Map option class 0x0102, type 0x80, length 4 to OVS register
ovs-vsctl set Open_vSwitch . \
    other_config:geneve-option-map="{class=0x0102,type=0x80,len=4}->tun_metadata0"

# Use TLV metadata in OpenFlow rules
ovs-ofctl add-flow br-int \
    "table=0,tun_metadata0=0x1234,actions=output:LOCAL"

# Show tunnel port details
ovs-vsctl show
ovs-ofctl dump-ports-desc br-int

# Check datapath tunnel stats
ovs-dpctl show
ovs-appctl dpif/show

# Delete tunnel port
ovs-vsctl del-port br-int geneve0
```

## TLV Options and Metadata

```bash
# Geneve TLV option format:
#   Option Class (16 bits) — vendor/standards namespace
#   Type (8 bits) — option identifier within class
#   Length (5 bits) — data length in 4-byte units (0-31 = 0-124 bytes)
#   R bits (3 bits) — reserved
#   Data — variable length, padded to 4-byte boundary

# Well-known option classes:
#   0x0000-0x00FF  — IETF standardized
#   0x0100         — Linux kernel
#   0x0101         — OVS (Open vSwitch)
#   0x0102         — VMware NSX
#   0xFFFF         — Experimental

# Critical bit (C=1 in base header):
#   Set when at least one TLV has its critical flag set
#   Receiver MUST understand all critical options or drop the packet
#   Non-critical options can be safely ignored

# OAM bit (O=1 in base header):
#   Marks the packet as control/management plane
#   Transit devices may prioritize or process OAM differently
#   Data plane packets MUST set O=0

# Maximum option space: 63 * 4 = 252 bytes
# Practical limit: keep options small to avoid MTU issues
# Each option adds overhead: 4-byte header + data (4-byte aligned)
```

## MTU Considerations

```bash
# Geneve overhead calculation:
#   Outer Ethernet:     14 bytes
#   Outer IP (IPv4):    20 bytes
#   Outer UDP:           8 bytes
#   Geneve base header:  8 bytes
#   ─────────────────────────────
#   Minimum overhead:   50 bytes (no options, IPv4 underlay)

# With IPv6 underlay: 14 + 40 + 8 + 8 = 70 bytes
# With TLV options: add 4-byte header + data per option

# For 1500-byte MTU underlay:
#   Max inner frame (no options, IPv4):   1500 - 50 = 1450 bytes
#   Max inner frame (no options, IPv6):   1500 - 70 = 1430 bytes

# Recommended: use jumbo frames (9000 MTU) on the underlay
# Inner 1500 + 50 overhead = 1550 < 9000 (plenty of room)

# Set geneve interface MTU
ip link set geneve100 mtu 1450

# Verify effective MTU
ip link show geneve100

# Calculate overhead with options:
# Example: 2 TLV options, each 8 bytes (4 header + 4 data) = 16 bytes
# Total: 50 + 16 = 66 bytes overhead
# Inner MTU: 1500 - 66 = 1434 bytes

# Check path MTU to tunnel endpoint
tracepath 10.0.0.2

# Enable PMTUD for geneve tunnel
# (inner packets with DF bit will get ICMP need-frag if too large)
sysctl -w net.ipv4.ip_no_pmtu_disc=0
```

## Hardware Offload

```bash
# Check NIC offload capabilities for Geneve
ethtool -k eth0 | grep geneve
# tx-udp_tnl-segmentation: on       (TSO for Geneve)
# tx-udp_tnl-csum-segmentation: on  (TSO + checksum)
# rx-udp_tunnel-port-offload: on    (RSS/flow steering)

# Check if NIC supports Geneve Rx offload
ethtool --show-tunnels eth0
# Shows configured tunnel ports for offload

# Add Geneve port to NIC offload table
# (usually automatic when geneve interface is created)
ethtool --add-tunnel eth0 geneve port 6081

# NICs with Geneve offload support:
# - Mellanox ConnectX-5+ (mlx5_core)
# - Intel X710/XL710 (i40e)
# - Intel E810 (ice)
# - Broadcom NetXtreme-E (bnxt_en)

# Verify offload is active (check counters)
ethtool -S eth0 | grep tunnel
ethtool -S eth0 | grep geneve

# Offload impact:
# Without offload: ~3-5 Gbps (CPU-bound encap/decap)
# With offload: line rate (25/40/100 Gbps)
```

## Capturing and Debugging Geneve

```bash
# Capture Geneve traffic (UDP port 6081)
tcpdump -i eth0 -nn 'udp port 6081'

# Capture with inner packet decoding
tcpdump -i eth0 -nn -vv 'udp port 6081'

# Capture and show VNI and inner headers
tcpdump -i eth0 -nn -X 'udp port 6081' | head -100

# Filter by specific VNI (VNI is at offset 4 in Geneve header)
# Geneve header starts at UDP payload offset
# VNI occupies bytes 4-6 of the Geneve header (24 bits)
tcpdump -i eth0 -nn 'udp port 6081 and udp[12:3] = 100'

# Use tshark for detailed Geneve decode
tshark -i eth0 -f 'udp port 6081' -V -O geneve

# Filter by VNI in tshark
tshark -i eth0 -f 'udp port 6081' -Y 'geneve.vni == 100'

# Show Geneve TLV options
tshark -i eth0 -f 'udp port 6081' -Y 'geneve.options'

# OVS flow debugging
ovs-appctl ofproto/trace br-int in_port=geneve0

# Check geneve interface statistics
ip -s link show geneve100

# Check for drops (MTU issues often show as tx_dropped)
ip -s link show geneve100 | grep -E 'TX|RX|drop'

# Kernel geneve module debug
dmesg | grep geneve
```

## Geneve in Container Networking

```bash
# Geneve is the default overlay for:
#   - OVN (Open Virtual Network) — OpenStack, oVirt
#   - Cilium (Kubernetes CNI, optional)
#   - Antrea (Kubernetes CNI, default)

# Check OVN geneve tunnels
ovs-vsctl show | grep -A5 geneve

# Cilium with Geneve encapsulation
# In cilium-config ConfigMap:
#   tunnel: geneve
#   tunnel-port: 6081

# Check Cilium tunnel mode
cilium status | grep Tunnel

# Antrea default encapsulation
# In antrea-config ConfigMap:
#   tunnelType: geneve

# Kubernetes node-to-node Geneve tunnel verification
kubectl exec -n kube-system cilium-xxxx -- cilium bpf tunnel list

# Each K8s node gets a unique VNI or uses VNI 0 with flow-based routing
# Pod-to-pod traffic: inner Ethernet + outer Geneve + underlay IP
```

---

## Tips

- Geneve's TLV options are its key advantage over VXLAN. If you do not need metadata extensibility, VXLAN has broader hardware offload support and may perform better on older NICs.
- Always calculate your effective inner MTU after Geneve overhead (minimum 50 bytes with IPv4, 70 bytes with IPv6, plus any TLV options). Mismatched MTU is the most common cause of tunnel black holes.
- Use jumbo frames (MTU 9000) on the physical underlay fabric to avoid fragmenting inner 1500-byte frames. Geneve with IPv4 underlay needs only 1550 bytes of outer capacity for standard inner frames.
- The UDP source port in Geneve is a hash of the inner packet headers. This provides ECMP entropy across the underlay so that traffic for the same VNI spreads across multiple paths.
- Set the Critical bit (C=1) only when your TLV options must be processed by every hop. Unknown critical options cause packet drops, so coordinate deployments carefully.
- Hardware offload (TSO, checksum, RSS) makes orders-of-magnitude difference in Geneve performance. Check `ethtool -k` for `tx-udp_tnl-segmentation` before deploying at scale.
- OVS maps Geneve TLV options to `tun_metadata` registers in OpenFlow. This lets you carry rich policy metadata (security tags, service chain IDs) inside the tunnel header and match on it in flow rules.
- When debugging Geneve tunnels, check MTU first (`ping -M do -s 1400`), then FDB entries (`bridge fdb show`), then underlay reachability (`ping` the tunnel endpoint), then packet captures (`tcpdump udp port 6081`).
- The OAM bit (O=1) is under-used but valuable. Mark BFD-over-Geneve or tunnel health check packets as OAM so transit devices can prioritize or fast-path them.
- Geneve tunnels traversing stateful firewalls need UDP port 6081 explicitly permitted. Unlike GRE, Geneve looks like normal UDP to middleboxes, which simplifies NAT traversal.

---

## See Also

- vxlan, vlan, bridge, mtu

## References

- [RFC 8926 — Geneve: Generic Network Virtualization Encapsulation](https://www.rfc-editor.org/rfc/rfc8926)
- [RFC 7348 — VXLAN](https://www.rfc-editor.org/rfc/rfc7348)
- [IANA — Geneve Option Classes](https://www.iana.org/assignments/geneve/geneve.xhtml)
- [Open vSwitch — Tunneling](https://docs.openvswitch.org/en/latest/howto/tunneling/)
- [Linux Kernel — Geneve Driver](https://www.kernel.org/doc/html/latest/networking/geneve.html)
- [OVN Architecture](https://docs.ovn.org/en/latest/ref/ovn-architecture.7.html)
