# GRE Tunnels (Generic Routing Encapsulation)

> Encapsulate arbitrary network-layer protocols inside IP packets for point-to-point or multipoint tunneling across routed infrastructure.

## Concepts

### GRE Header Format

```
# GRE base header: 4 bytes (mandatory)
# Optional fields add up to 12 additional bytes
#
#  0                   1                   2                   3
#  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
# +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
# |C| |K|S| Reserved0       | Ver |         Protocol Type         |
# +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
# |      Checksum (optional)      |       Reserved1 (optional)    |
# +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
# |                         Key (optional)                        |
# +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
# |                  Sequence Number (optional)                   |
# +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
#
# C (bit 0):   Checksum present
# K (bit 2):   Key present
# S (bit 3):   Sequence number present
# Ver (13-15): GRE version (0 for standard, 1 for PPTP/enhanced)
# Protocol:    EtherType of encapsulated payload
#              0x0800 = IPv4, 0x86DD = IPv6, 0x6558 = Transparent Ethernet Bridging
```

### GRE Overhead

```
# Standard GRE over IPv4:
#   Outer IP header:     20 bytes
#   GRE header (base):    4 bytes
#   Total overhead:       24 bytes
#
# With optional fields:
#   + Checksum:           4 bytes (checksum + reserved1)
#   + Key:                4 bytes
#   + Sequence number:    4 bytes
#   Max overhead:         36 bytes (20 IP + 16 GRE)
#
# GRE over IPsec (tunnel mode ESP):
#   Outer IP:            20 bytes
#   ESP header:           8 bytes
#   GRE header:           4 bytes
#   Inner IP (original): 20 bytes
#   ESP trailer:         ~18 bytes (pad + auth)
#   Total overhead:      ~70 bytes
#
# Effective MTU with 1500-byte path:
#   Plain GRE:           1500 - 24 = 1476 bytes
#   GRE + IPsec:         1500 - 70 = ~1430 bytes
```

### GRE Protocol Type (IP Protocol 47)

```
# GRE uses IP protocol number 47
# Not TCP or UDP — no port numbers
# Outer packet: [IP header, proto=47][GRE header][Inner packet]

# Common protocol types in GRE header:
#   0x0800  IPv4
#   0x86DD  IPv6
#   0x6558  Transparent Ethernet Bridging (L2 over GRE)
#   0x0806  ARP (in L2 GRE)
#   0x8847  MPLS unicast
#   0x8848  MPLS multicast
```

## IOS GRE Tunnel Configuration

### Basic Point-to-Point GRE Tunnel

```bash
# Router A (10.1.1.1) ←→ Router B (10.2.2.1)
# Tunnel network: 172.16.0.0/30

# Router A
interface Tunnel0
 ip address 172.16.0.1 255.255.255.252
 tunnel source 10.1.1.1
 tunnel destination 10.2.2.1
 tunnel mode gre ip

# Router B
interface Tunnel0
 ip address 172.16.0.2 255.255.255.252
 tunnel source 10.2.2.1
 tunnel destination 10.1.1.1
 tunnel mode gre ip

# Using interface as source (preferred — survives IP changes)
interface Tunnel0
 tunnel source GigabitEthernet0/0
 tunnel destination 10.2.2.1
```

### Tunnel Key

```bash
# Tunnel key differentiates multiple tunnels between same endpoints
# Both sides must match

# Router A
interface Tunnel0
 tunnel key 12345

# Router B
interface Tunnel0
 tunnel key 12345

# Without matching keys, packets are silently dropped
```

### GRE Keepalives

```bash
# GRE keepalives detect tunnel endpoint failure
# Sends keepalive through the tunnel; if no response, tunnel goes down

interface Tunnel0
 keepalive 10 3
 # Send keepalive every 10 seconds
 # Mark tunnel down after 3 missed keepalives (30 seconds)

# Disable keepalives (default)
interface Tunnel0
 no keepalive

# Keepalive mechanism:
# - Router sends a GRE packet to itself via the tunnel
# - Remote end forwards it back through the tunnel
# - If the packet returns, the tunnel is up
# - Requires both ends to support GRE keepalives
```

### Tunnel with Routing Protocol

```bash
# OSPF over GRE tunnel
interface Tunnel0
 ip address 172.16.0.1 255.255.255.252
 ip ospf 1 area 0
 tunnel source 10.1.1.1
 tunnel destination 10.2.2.1

router ospf 1
 network 172.16.0.0 0.0.0.3 area 0
 network 192.168.1.0 0.0.0.255 area 0

# EIGRP over GRE tunnel
interface Tunnel0
 ip address 172.16.0.1 255.255.255.252

router eigrp 100
 network 172.16.0.0 0.0.0.3
 network 192.168.1.0 0.0.0.255
```

## Recursive Routing Problem

### The Problem

```
# Recursive routing occurs when the tunnel destination
# is reachable via the tunnel interface itself
#
# Example:
#   Tunnel0 destination = 10.2.2.1
#   Route to 10.2.2.1 points to Tunnel0 (learned via OSPF over tunnel)
#
# Result: routing loop → tunnel flaps → CPU spike
#
# Packet flow in recursive routing:
#   1. Outer packet (dst=10.2.2.1) needs to be sent
#   2. Route lookup → best route is via Tunnel0
#   3. Encapsulate in GRE → new outer packet (dst=10.2.2.1)
#   4. Route lookup → best route is via Tunnel0
#   5. Loop detected → interface goes down
```

### Solutions

```bash
# Solution 1: Static route for tunnel destination via physical interface
ip route 10.2.2.1 255.255.255.255 10.1.1.2

# Solution 2: Use a separate routing table/VRF for transport
interface Tunnel0
 tunnel vrf TRANSPORT
 tunnel source 10.1.1.1
 tunnel destination 10.2.2.1

# Solution 3: Route filtering — prevent tunnel destination from being advertised over tunnel
router ospf 1
 distribute-list prefix NO_TUNNEL_DST out Tunnel0

ip prefix-list NO_TUNNEL_DST deny 10.2.2.0/24
ip prefix-list NO_TUNNEL_DST permit 0.0.0.0/0 le 32

# Solution 4: Higher administrative distance on tunnel-learned routes
interface Tunnel0
 ip ospf cost 1000

# Solution 5: Tunnel source/destination via loopback with static routes
interface Loopback0
 ip address 10.1.1.1 255.255.255.255

ip route 10.2.2.1 255.255.255.255 192.168.1.2

interface Tunnel0
 tunnel source Loopback0
 tunnel destination 10.2.2.1
```

## MTU and Fragmentation

### MTU Configuration

```bash
# Set tunnel interface MTU (accounts for GRE + IP overhead)
interface Tunnel0
 ip mtu 1476                    # 1500 - 24 (GRE overhead)

# Adjust TCP MSS to prevent fragmentation
interface Tunnel0
 ip tcp adjust-mss 1436         # 1476 - 40 (TCP/IP headers)

# With IPsec
interface Tunnel0
 ip mtu 1400                    # 1500 - ~100 (GRE + IPsec overhead)
 ip tcp adjust-mss 1360         # 1400 - 40
```

### PMTUD (Path MTU Discovery)

```bash
# Enable PMTUD on tunnel interface
interface Tunnel0
 tunnel path-mtu-discovery

# How PMTUD works with GRE:
# 1. Inner packet has DF (Don't Fragment) bit set
# 2. Outer GRE/IP packet also gets DF bit
# 3. If outer packet exceeds link MTU, ICMP "frag needed" returned
# 4. Tunnel endpoint adjusts MTU and sends ICMP back to original source
# 5. Source reduces packet size

# Tunnel DF bit behavior
interface Tunnel0
 tunnel path-mtu-discovery
 # Copies DF bit from inner to outer packet

# For double-encapsulation (GRE over IPsec):
# Ensure MSS accounts for both encapsulations
```

### Fragmentation Scenarios

```
# Scenario: 1500-byte inner packet, 1500-byte path MTU
#
# Without PMTUD:
#   Inner packet (1500) + GRE/IP (24) = 1524 bytes
#   Outer packet fragmented into two IP fragments
#   Fragment 1: 1500 bytes (with GRE header)
#   Fragment 2: 44 bytes (remaining payload + IP header)
#   Both fragments must arrive for reassembly
#
# With PMTUD:
#   ICMP "fragmentation needed" sent back
#   Source reduces packet to 1476 bytes
#   Outer packet: 1476 + 24 = 1500 bytes — fits without fragmentation
#
# With pre-fragmentation (ip mtu set):
#   Inner packet fragmented BEFORE encapsulation
#   Fragment 1: 1476 bytes + GRE/IP = 1500 bytes
#   Fragment 2: 44 bytes + GRE/IP = 68 bytes
#   Two complete GRE packets — each independently routable
```

## GRE over IPsec

### Crypto Map Approach

```bash
# Define IPsec transform set
crypto ipsec transform-set GRE_IPSEC esp-aes 256 esp-sha256-hmac
 mode transport   # Transport mode — encrypt GRE payload only

# Define crypto ACL (match GRE traffic between endpoints)
ip access-list extended GRE_TRAFFIC
 permit gre host 10.1.1.1 host 10.2.2.1

# Define crypto map
crypto map IPSEC_MAP 10 ipsec-isakmp
 set peer 10.2.2.1
 set transform-set GRE_IPSEC
 match address GRE_TRAFFIC

# Apply to physical interface (NOT tunnel interface)
interface GigabitEthernet0/0
 crypto map IPSEC_MAP
```

### IPsec Profile (VTI-Style)

```bash
# Simpler approach using IPsec profile on tunnel interface
crypto ipsec transform-set GRE_IPSEC esp-aes 256 esp-sha256-hmac
 mode transport

crypto ipsec profile GRE_PROFILE
 set transform-set GRE_IPSEC

interface Tunnel0
 ip address 172.16.0.1 255.255.255.252
 tunnel source 10.1.1.1
 tunnel destination 10.2.2.1
 tunnel mode gre ip
 tunnel protection ipsec profile GRE_PROFILE
```

## Multipoint GRE (mGRE)

### mGRE for DMVPN

```bash
# mGRE allows a single tunnel interface to connect to multiple endpoints
# Used primarily in DMVPN (Dynamic Multipoint VPN)
# NHRP resolves tunnel destinations dynamically

# Hub configuration
interface Tunnel0
 ip address 172.16.0.1 255.255.255.0
 tunnel source GigabitEthernet0/0
 tunnel mode gre multipoint          # mGRE — no static destination
 ip nhrp network-id 1
 ip nhrp map multicast dynamic       # Allow dynamic multicast mapping
 ip nhrp authentication SECRET
 ip tcp adjust-mss 1360

# Spoke configuration
interface Tunnel0
 ip address 172.16.0.2 255.255.255.0
 tunnel source GigabitEthernet0/0
 tunnel mode gre multipoint
 ip nhrp network-id 1
 ip nhrp nhs 172.16.0.1              # Hub as NHS (Next Hop Server)
 ip nhrp map 172.16.0.1 10.1.1.1     # Static mapping for hub
 ip nhrp map multicast 10.1.1.1      # Multicast to hub
 ip nhrp authentication SECRET
 ip tcp adjust-mss 1360

# With DMVPN Phase 3 (shortcut switching)
interface Tunnel0
 ip nhrp redirect                     # Hub: redirect spokes to talk directly
 ip nhrp shortcut                     # Spoke: install shortcut routes
```

## GRE with IPv6

### IPv6 over GRE (6in4-style)

```bash
# Carry IPv6 traffic over IPv4 infrastructure

# IOS configuration
interface Tunnel0
 ipv6 address 2001:db8:1::1/64
 tunnel source 10.1.1.1
 tunnel destination 10.2.2.1
 tunnel mode gre ip                   # IPv4 transport, IPv6 payload

# IPv6 routing over the tunnel
ipv6 route 2001:db8:2::/48 Tunnel0 2001:db8:1::2
```

### GRE over IPv6 Transport

```bash
# GRE tunnel using IPv6 as transport
interface Tunnel0
 ip address 172.16.0.1 255.255.255.252
 tunnel source 2001:db8:1::1
 tunnel destination 2001:db8:2::1
 tunnel mode gre ipv6                # IPv6 transport

# IPv6 transport adds 40 bytes overhead (vs 20 for IPv4)
# Effective MTU: 1500 - 40 (IPv6) - 4 (GRE) = 1456 bytes
```

## Linux GRE Configuration

### ip tunnel (iproute2)

```bash
# Create GRE tunnel
ip tunnel add gre1 mode gre remote 10.2.2.1 local 10.1.1.1 ttl 255
ip addr add 172.16.0.1/30 dev gre1
ip link set gre1 up

# Add route over tunnel
ip route add 192.168.2.0/24 dev gre1

# GRE with key
ip tunnel add gre1 mode gre remote 10.2.2.1 local 10.1.1.1 key 12345

# GRE with checksum
ip tunnel add gre1 mode gre remote 10.2.2.1 local 10.1.1.1 csum

# GRE with sequence numbers
ip tunnel add gre1 mode gre remote 10.2.2.1 local 10.1.1.1 seq

# Delete tunnel
ip tunnel del gre1

# Show tunnel parameters
ip tunnel show
ip -d link show gre1
```

### GRE TAP (L2 GRE / GRETAP)

```bash
# GRETAP: Layer 2 GRE — carries Ethernet frames
# Uses protocol type 0x6558 (Transparent Ethernet Bridging)

# Create GRETAP interface
ip link add gretap1 type gretap remote 10.2.2.1 local 10.1.1.1
ip link set gretap1 up

# Bridge the GRETAP interface (extend L2 domain)
ip link add br0 type bridge
ip link set gretap1 master br0
ip link set eth1 master br0
ip link set br0 up

# GRETAP with VLAN
ip link add gretap1 type gretap remote 10.2.2.1 local 10.1.1.1
ip link add link gretap1 name gretap1.100 type vlan id 100

# GRETAP overhead:
#   Outer IP:      20 bytes
#   GRE header:     4 bytes
#   Inner Ethernet: 14 bytes
#   Total:          38 bytes (vs 24 for L3 GRE)
```

### Persistent Linux Configuration (systemd-networkd)

```ini
# /etc/systemd/network/25-gre1.netdev
[NetDev]
Name=gre1
Kind=gre

[Tunnel]
Remote=10.2.2.1
Local=10.1.1.1
TTL=255

# /etc/systemd/network/25-gre1.network
[Match]
Name=gre1

[Network]
Address=172.16.0.1/30

[Route]
Destination=192.168.2.0/24
Gateway=172.16.0.2
```

## Verification and Troubleshooting

### IOS Commands

```bash
# Show tunnel interface status
show interface Tunnel0
show ip interface brief | include Tunnel

# Show tunnel configuration
show running-config interface Tunnel0

# Show tunnel encapsulation details
show interfaces Tunnel0 | include encaps|tunnel

# Verify tunnel endpoints
show crypto ipsec sa           # If using IPsec protection
show crypto isakmp sa          # IKE SA status

# Debug GRE
debug tunnel                   # Tunnel events
debug ip packet                # Packet-level debugging (use ACL filter)

# Check MTU
show interface Tunnel0 | include MTU
ping 172.16.0.2 size 1476 df-bit   # Test effective MTU

# NHRP verification (mGRE/DMVPN)
show ip nhrp
show ip nhrp multicast
show dmvpn                     # DMVPN-specific summary
```

### Linux Commands

```bash
# Show tunnel status
ip tunnel show
ip -d link show gre1
ip -s link show gre1           # Statistics

# Test connectivity
ping -M do -s 1448 172.16.0.2  # Test MTU (1448 + 28 ICMP/IP = 1476)

# Capture GRE traffic
tcpdump -i eth0 proto gre
tcpdump -i eth0 'ip proto 47'

# Capture inner traffic on tunnel interface
tcpdump -i gre1

# Check routing
ip route show dev gre1
ip route get 192.168.2.1

# Monitor tunnel interface
ip monitor link dev gre1
```

## Tips

- Always set `ip mtu` and `ip tcp adjust-mss` on tunnel interfaces to prevent fragmentation.
- Use `tunnel path-mtu-discovery` when possible for dynamic MTU adjustment.
- Solve recursive routing before enabling the tunnel — use static routes for tunnel endpoints.
- Prefer transport mode IPsec with GRE to avoid double IP header overhead.
- Use tunnel keys to differentiate multiple tunnels between the same pair of endpoints.
- Enable keepalives to detect remote endpoint failure and trigger routing convergence.
- For DMVPN, use mGRE with NHRP — avoid creating individual point-to-point tunnels.
- GRETAP (L2 GRE) is useful for extending VLANs across L3 boundaries but adds 14 bytes of Ethernet overhead.
- GRE is IP protocol 47, not TCP or UDP — ensure firewalls and NAT devices permit it.

## See Also

- ipsec, dmvpn, vxlan, ipv4, ipv6, mtu, ospf, eigrp, tuntap, network-acl

## References

- [RFC 2784 — Generic Routing Encapsulation (GRE)](https://www.rfc-editor.org/rfc/rfc2784)
- [RFC 2890 — Key and Sequence Number Extensions to GRE](https://www.rfc-editor.org/rfc/rfc2890)
- [RFC 1701 — Generic Routing Encapsulation (original)](https://www.rfc-editor.org/rfc/rfc1701)
- [RFC 1702 — Generic Routing Encapsulation over IPv4 Networks](https://www.rfc-editor.org/rfc/rfc1702)
- [RFC 7676 — IPv6 Support for Generic Routing Encapsulation](https://www.rfc-editor.org/rfc/rfc7676)
- [Cisco GRE Tunnel Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/interface/configuration/xe-16/ir-xe-16-book/ir-gre-tunnel.html)
- [Cisco DMVPN Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/sec_conn_dmvpn/configuration/xe-16/sec-conn-dmvpn-xe-16-book.html)
- [Linux ip-tunnel(8) man page](https://man7.org/linux/man-pages/man8/ip-tunnel.8.html)
- [Linux Kernel — GRE Tunnel Documentation](https://www.kernel.org/doc/html/latest/networking/gre.html)
