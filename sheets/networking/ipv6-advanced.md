# Advanced IPv6 (Extension Headers, NDP, DHCPv6, Transition, and Security)

Advanced IPv6 topics including address architecture details, extension header chaining, NDP mechanics, DHCPv6 stateful/stateless modes, prefix delegation, first-hop security, IPv6 routing protocols, and IPv4-to-IPv6 transition mechanisms.

## Address Types in Detail

### Global Unicast Address (GUA) Structure

```
+------+----------+----------+---------------------------+
| 3 b  | 45 bits  | 16 bits  |        64 bits            |
| 001  | Global   | Subnet   |    Interface ID           |
|      | Routing  |   ID     |  (EUI-64 or random)       |
|      | Prefix   |          |                           |
+------+----------+----------+---------------------------+
  2000::/3 range

# Show GUA addresses on all interfaces
ip -6 addr show scope global

# Typical allocation hierarchy
# /32  — RIR to ISP
# /48  — ISP to enterprise customer
# /56  — ISP to residential customer
# /64  — single subnet (required for SLAAC)
```

### Unique Local Address (ULA)

```
+--------+--+------------------------------------------+----------------+
| 7 bits |L | 40-bit Global ID    | 16-bit Subnet ID  | 64-bit IID     |
| fc00:: |1 | (pseudo-random)     |                    |                |
+--------+--+------------------------------------------+----------------+
  fd00::/8 in practice (L=1, locally assigned)

# Generate a compliant ULA prefix (40-bit random Global ID)
# Use current time + EUI-64 of interface, SHA-1 hash, take last 40 bits
python3 -c "
import hashlib, uuid, time, struct
eui = uuid.getnode().to_bytes(6, 'big')
ts = struct.pack('>Q', int(time.time() * 1e9))
h = hashlib.sha1(ts + eui).hexdigest()
gid = h[-10:]  # last 40 bits
print(f'fd{gid[:2]}:{gid[2:6]}:{gid[6:10]}::/48')
"
```

### Link-Local Address

```bash
# Always fe80::/10, auto-generated on every IPv6-enabled interface
# Required for NDP, OSPFv3, and other link-scoped protocols
ip -6 addr show scope link

# Link-local is ALWAYS present even if no GUA is assigned
# Scope ID (%iface) is mandatory for any operation
ping6 fe80::1%eth0
ssh fe80::1%25eth0       # URL encoding: %25 = literal %
```

### Multicast Scopes

```
Multicast address format:  ff<flags><scope>::<group>

Scope values:
  1 — Interface-local (loopback only)
  2 — Link-local (single LAN segment)
  4 — Admin-local (administratively defined)
  5 — Site-local (single site)
  8 — Organization-local (multiple sites)
  E — Global (internet-wide)

Well-known link-local multicast (ff02::):
  ff02::1     All nodes
  ff02::2     All routers
  ff02::5     OSPFv3 all routers
  ff02::6     OSPFv3 designated routers
  ff02::9     RIPng
  ff02::a     EIGRP
  ff02::d     PIM routers
  ff02::fb    mDNS
  ff02::101   NTP
  ff02::1:2   DHCPv6 all relay agents and servers
  ff02::1:3   LLMNR

Solicited-node multicast (for NDP):
  ff02::1:ff<last-24-bits-of-unicast>
  Example: 2001:db8::abcd → ff02::1:ff00:abcd
```

```bash
# Show multicast group memberships
ip -6 maddr show
ip -6 maddr show dev eth0

# Join a multicast group
ip -6 maddr add ff02::1234 dev eth0

# Verify solicited-node multicast for an address
ip -6 maddr show dev eth0 | grep "ff02::1:ff"
```

## Extension Headers

### Header Chain Order (RFC 8200)

```
IPv6 Header (Next Header field)
  └─→ Hop-by-Hop Options (NH=0)    ← must be first if present
       └─→ Destination Options (NH=60)  ← for first destination
            └─→ Routing Header (NH=43)
                 └─→ Fragment Header (NH=44)
                      └─→ Authentication Header (NH=51)
                           └─→ ESP Header (NH=50)
                                └─→ Destination Options (NH=60) ← for final dest
                                     └─→ Upper-Layer (TCP=6, UDP=17, ICMPv6=58)

# Each header has a Next Header field pointing to the next one
# Processing stops when NH indicates upper-layer protocol
```

### Extension Header Summary

| Header | NH Value | Purpose | Processed By |
|:---|:---:|:---|:---|
| Hop-by-Hop Options | 0 | Router Alert, Jumbo Payload, MLD | Every router on path |
| Routing | 43 | Source routing (Type 2 for MIPv6, SRH Type 4) | Routers listed in header |
| Fragment | 44 | Fragmentation (source only, not routers) | Destination only |
| Destination Options | 60 | Options for destination node | Destination only |
| Authentication (AH) | 51 | Integrity + authentication (IPsec) | Destination |
| ESP | 50 | Encryption + auth (IPsec) | Destination |
| No Next Header | 59 | Nothing follows | N/A |
| Mobility | 135 | Mobile IPv6 signaling | Destination |
| HIP | 139 | Host Identity Protocol | Destination |

### Fragment Header

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|  Next Header  |   Reserved    |   Fragment Offset   |Res|M|
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                        Identification                         |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

Key differences from IPv4:
  - Only the SOURCE fragments (never intermediate routers)
  - Minimum MTU is 1280 bytes (vs 576 in IPv4)
  - PMTUD is mandatory — routers send ICMPv6 Packet Too Big
  - Fragment overlap is forbidden (RFC 5722) — security hardening
```

```bash
# Check path MTU to a destination
tracepath6 2001:db8::1

# Test with specific packet size (DF is always implicit in IPv6)
ping6 -s 1452 -M do 2001:db8::1

# View cached PMTU entries
ip -6 route get 2001:db8::1 | grep mtu
```

### Segment Routing Header (SRH, Type 4)

```
IPv6 Header (DA = first segment)
  └─→ SRH (Routing Header Type 4)
       Segments Left: 2
       Segment List[0]: S3 (final destination)
       Segment List[1]: S2 (waypoint)
       Segment List[2]: S1 (first segment — current DA)

# As each segment is traversed, Segments Left decrements
# IPv6 DA is updated to the next segment in the list
# SRH enables source-routed traffic engineering in SRv6
```

## Neighbor Discovery Protocol (NDP)

### NDP Message Types (ICMPv6)

| Message | ICMPv6 Type | Purpose |
|:---|:---:|:---|
| Router Solicitation (RS) | 133 | Host asks for router presence |
| Router Advertisement (RA) | 134 | Router announces prefix, flags, lifetime |
| Neighbor Solicitation (NS) | 135 | Address resolution + DAD probe |
| Neighbor Advertisement (NA) | 136 | Response to NS |
| Redirect | 137 | Router tells host about better next-hop |

### SLAAC Process

```
1. Interface comes up
2. Generate link-local (fe80::IID)
3. Perform DAD on link-local (send NS to solicited-node multicast)
4. If no NA received → link-local is valid
5. Send RS to ff02::2 (all routers)
6. Receive RA with:
   - Prefix + prefix length (e.g., 2001:db8:1::/64)
   - A flag (autonomous) = 1 → use this prefix for SLAAC
   - L flag (on-link) = 1 → prefix is on-link
   - M flag = 0, O flag = 0 → pure SLAAC
   - Valid lifetime, preferred lifetime
   - Default router lifetime → install default route
7. Generate GUA = prefix + IID (EUI-64 or random privacy addr)
8. Perform DAD on GUA
9. Address enters preferred state
```

```bash
# Watch NDP traffic in real time
sudo tcpdump -i eth0 -n 'icmp6 and (ip6[40] >= 133 and ip6[40] <= 137)'

# Show neighbor cache (equivalent to ARP table)
ip -6 neigh show
ip -6 neigh show dev eth0

# States: INCOMPLETE, REACHABLE, STALE, DELAY, PROBE, FAILED

# Force neighbor refresh
ip -6 neigh flush dev eth0

# Manually add static neighbor
ip -6 neigh add 2001:db8::1 lladdr 00:11:22:33:44:55 dev eth0

# Send Router Solicitation manually
rdisc6 eth0

# View RA information
rdisc6 -1 eth0       # single RA, verbose
```

### Duplicate Address Detection (DAD)

```bash
# DAD sends NS for the tentative address to its solicited-node multicast
# Source = :: (unspecified), Target = tentative address
# If NA received → duplicate detected, address NOT assigned

# Check DAD configuration
sysctl net.ipv6.conf.eth0.dad_transmits    # number of DAD probes (default 1)
sysctl net.ipv6.conf.eth0.accept_dad       # 0=disable, 1=enable, 2=disable+notify

# Check for DAD failures
ip -6 addr show dadfailed
ip -6 addr show tentative

# Disable DAD (not recommended in production)
sudo sysctl -w net.ipv6.conf.eth0.dad_transmits=0

# Optimistic DAD (RFC 4429) — use address immediately, defend if challenged
sudo sysctl -w net.ipv6.conf.eth0.optimistic_dad=1
```

## DHCPv6

### Stateful vs Stateless DHCPv6

```
RA Flags determine DHCPv6 behavior:
  M=0, O=0 → SLAAC only (no DHCPv6)
  M=0, O=1 → SLAAC for address + DHCPv6 for DNS/NTP/domain (stateless)
  M=1, O=* → DHCPv6 for address assignment (stateful)

Stateless DHCPv6 (Information-Request):
  Client ──[Information-Request]──→ Server
  Server ──[Reply (DNS, NTP, etc.)]──→ Client

Stateful DHCPv6 (SARR — Solicit/Advertise/Request/Reply):
  Client ──[Solicit]──→ ff02::1:2 (all DHCP agents)
  Server ──[Advertise]──→ Client
  Client ──[Request]──→ Server
  Server ──[Reply (address + options)]──→ Client

Renew/Rebind timers: T1 (renew, default 0.5 * preferred), T2 (rebind, default 0.8 * preferred)
```

### DHCPv6 Client Configuration (Linux)

```bash
# Using dhclient for stateful DHCPv6
sudo dhclient -6 eth0

# Using dhclient for stateless (information-request only)
sudo dhclient -6 -S eth0

# Using NetworkManager
nmcli connection modify "Wired" ipv6.method dhcp        # stateful
nmcli connection modify "Wired" ipv6.method auto         # SLAAC + stateless DHCPv6

# Systemd-networkd (/etc/systemd/network/10-eth0.network)
# [Network]
# DHCP=ipv6
# IPv6AcceptRA=yes

# ISC dhcpd server config for stateful DHCPv6
# /etc/dhcp/dhcpd6.conf
# subnet6 2001:db8:1::/64 {
#   range6 2001:db8:1::100 2001:db8:1::200;
#   option dhcp6.name-servers 2001:db8::53;
#   option dhcp6.domain-search "example.com";
# }
```

### Prefix Delegation (DHCPv6-PD)

```
Requesting Router (RR) requests a prefix from Delegating Router (DR):

  RR ──[Solicit (IA_PD)]──→ DR
  DR ──[Advertise (IA_PD prefix)]──→ RR
  RR ──[Request (IA_PD)]──→ DR
  DR ──[Reply (IA_PD: 2001:db8:100::/48, lifetime)]──→ RR

RR then subnets the delegated prefix across its downstream interfaces:
  2001:db8:100:1::/64 → LAN1
  2001:db8:100:2::/64 → LAN2
  2001:db8:100:3::/64 → Guest WiFi
```

```bash
# Linux dhclient as PD client (requesting router)
# /etc/dhcp/dhclient6.conf
# interface "wan0" {
#   send dhcp6.client-id = <DUID>;
#   request;
#   also request dhcp6.name-servers;
#   send ia-pd 1;
# }

# Wide-DHCPv6 client (common on embedded routers)
# /etc/wide-dhcpv6/dhcp6c.conf
# interface wan0 {
#   send ia-pd 0;
# };
# id-assoc pd 0 {
#   prefix-interface lan0 {
#     sla-id 1;
#     sla-len 16;
#   };
# };

# Verify delegated prefix
ip -6 route show | grep "proto ra"
```

## IPv6 First-Hop Security

### RA Guard (RFC 6105)

```
# Cisco IOS — RA Guard policy
ipv6 nd raguard policy HOST-POLICY
 device-role host
!
interface GigabitEthernet0/1
 ipv6 nd raguard attach-policy HOST-POLICY

# Cisco IOS — allow RAs only from trusted router port
ipv6 nd raguard policy ROUTER-POLICY
 device-role router
 match ra prefix-list VALID-PREFIXES
!
interface GigabitEthernet0/24
 ipv6 nd raguard attach-policy ROUTER-POLICY

# Prefix list for RA Guard
ipv6 prefix-list VALID-PREFIXES permit 2001:db8:1::/48 ge 64 le 64
```

### DHCPv6 Guard

```
# Cisco IOS — DHCPv6 Guard policy
ipv6 dhcp guard policy CLIENT-POLICY
 device-role client
!
interface range GigabitEthernet0/1-23
 ipv6 dhcp guard attach-policy CLIENT-POLICY

ipv6 dhcp guard policy SERVER-POLICY
 device-role server
 match server access-list DHCP-SERVERS
!
interface GigabitEthernet0/24
 ipv6 dhcp guard attach-policy SERVER-POLICY
```

### ND Inspection and IPv6 Source Guard

```
# IPv6 ND Inspection — validates NS/NA messages
# Builds dynamic binding table (like DHCP snooping for IPv6)
ipv6 nd inspection policy ND-POLICY
 device-role host
 limit address-count 5
!
interface GigabitEthernet0/1
 ipv6 nd inspection attach-policy ND-POLICY

# IPv6 Source Guard — filters traffic based on binding table
ipv6 source-guard policy SRCGUARD-POLICY
 validate address
 deny global-autoconfig
!
interface GigabitEthernet0/1
 ipv6 source-guard attach-policy SRCGUARD-POLICY

# Show binding table
show ipv6 neighbors binding
show ipv6 nd inspection statistics
```

### IPv6 SISF (Switch Integrated Security Features)

```
# Cisco IOS-XE unified approach (replaces individual guards)
device-tracking policy TRACKING-POLICY
 security-level guard
 tracking enable
!
interface range GigabitEthernet1/0/1-48
 device-tracking attach-policy TRACKING-POLICY

show device-tracking database
show device-tracking counters
```

## IPv6 ACLs

### Cisco IOS IPv6 ACL

```
# IPv6 ACLs are always named (no numbered ACLs)
ipv6 access-list INBOUND-V6
 permit icmp any any nd-ns           ! allow Neighbor Solicitation
 permit icmp any any nd-na           ! allow Neighbor Advertisement
 permit icmp any any router-solicitation
 permit icmp any any router-advertisement
 permit icmp any any echo-request
 permit icmp any any echo-reply
 permit icmp any any packet-too-big  ! critical for PMTUD
 permit tcp any host 2001:db8::80 eq 443
 deny ipv6 any any log
!
interface GigabitEthernet0/0
 ipv6 traffic-filter INBOUND-V6 in
```

```bash
# Linux ip6tables equivalent
sudo ip6tables -A INPUT -p icmpv6 --icmpv6-type neighbor-solicitation -j ACCEPT
sudo ip6tables -A INPUT -p icmpv6 --icmpv6-type neighbor-advertisement -j ACCEPT
sudo ip6tables -A INPUT -p icmpv6 --icmpv6-type router-solicitation -j ACCEPT
sudo ip6tables -A INPUT -p icmpv6 --icmpv6-type router-advertisement -j ACCEPT
sudo ip6tables -A INPUT -p icmpv6 --icmpv6-type echo-request -j ACCEPT
sudo ip6tables -A INPUT -p icmpv6 --icmpv6-type packet-too-big -j ACCEPT
sudo ip6tables -A INPUT -p tcp -d 2001:db8::80 --dport 443 -j ACCEPT
sudo ip6tables -A INPUT -j DROP

# nftables inet family
sudo nft add rule inet filter input icmpv6 type {
  nd-neighbor-solicit, nd-neighbor-advert,
  nd-router-solicit, nd-router-advert,
  echo-request, packet-too-big
} accept
```

## IPv6 Routing Protocols

### OSPFv3

```
# Cisco IOS — OSPFv3 uses link-local as next-hop
ipv6 router ospf 1
 router-id 1.1.1.1
 passive-interface default
 no passive-interface GigabitEthernet0/0
!
interface GigabitEthernet0/0
 ipv6 ospf 1 area 0

# OSPFv3 address-family style (IOS 15.1+)
router ospfv3 1
 address-family ipv6 unicast
  router-id 1.1.1.1
  passive-interface default
  no passive-interface GigabitEthernet0/0
 !
!
interface GigabitEthernet0/0
 ospfv3 1 ipv6 area 0

# Key differences from OSPFv2:
#  - Runs over link-local addresses (not network statements)
#  - Per-link rather than per-subnet
#  - Uses IPv6 multicast ff02::5 (AllSPFRouters) and ff02::6 (AllDRouters)
#  - Built-in IPsec authentication (no plain-text auth)
#  - Instance ID field allows multiple instances per link
#  - Can carry both IPv4 and IPv6 AFs (OSPFv3 AF, RFC 5838)
```

```bash
# Linux — BIRD OSPFv3
# /etc/bird/bird6.conf
# protocol ospf v3 {
#   area 0 {
#     interface "eth0" {
#       type broadcast;
#       cost 10;
#       hello 10;
#       dead 40;
#     };
#   };
# }

# FRRouting (FRR) OSPFv3
# router ospf6
#  ospf6 router-id 1.1.1.1
#  area 0.0.0.0 range 2001:db8::/32
# !
# interface eth0
#  ipv6 ospf6 area 0.0.0.0
```

### EIGRP for IPv6

```
# Cisco IOS — EIGRP named mode with IPv6 AF
router eigrp NAMED-EIGRP
 address-family ipv6 unicast autonomous-system 100
  eigrp router-id 1.1.1.1
  af-interface default
   shutdown
  exit-af-interface
  af-interface GigabitEthernet0/0
   no shutdown
  exit-af-interface
 exit-address-family

# Classic mode
ipv6 router eigrp 100
 eigrp router-id 1.1.1.1
 no shutdown
!
interface GigabitEthernet0/0
 ipv6 eigrp 100

# Key differences from IPv4 EIGRP:
#  - Uses link-local next-hops
#  - No network statement — enabled per-interface
#  - Must explicitly enable (no shutdown) in named mode
#  - Multicast ff02::a (EIGRP routers)
```

### MP-BGP IPv6 Address Family

```
# Cisco IOS — IPv6 unicast AF in MP-BGP
router bgp 65001
 bgp router-id 1.1.1.1
 no bgp default ipv4-unicast
 neighbor 2001:db8::2 remote-as 65002
 !
 address-family ipv6 unicast
  neighbor 2001:db8::2 activate
  network 2001:db8:100::/48
 exit-address-family

# eBGP over link-local (common on IXPs)
router bgp 65001
 neighbor fe80::2%GigabitEthernet0/0 remote-as 65002
 !
 address-family ipv6 unicast
  neighbor fe80::2%GigabitEthernet0/0 activate
  neighbor fe80::2%GigabitEthernet0/0 route-map PEER-IN in
 exit-address-family

# FRRouting MP-BGP IPv6
# router bgp 65001
#  neighbor 2001:db8::2 remote-as 65002
#  address-family ipv6 unicast
#   neighbor 2001:db8::2 activate
#   network 2001:db8:100::/48
#  exit-address-family
```

## IPv4/IPv6 Transition Mechanisms

### Dual-Stack

```bash
# Most common approach — run IPv4 and IPv6 simultaneously
# Verify dual-stack operation
ip addr show dev eth0 | grep -E "inet |inet6"

# Application behavior follows RFC 6724 source address selection
# Check address selection policy
cat /etc/gai.conf

# Force IPv4 or IPv6 per-application
curl -4 https://example.com
curl -6 https://example.com

# DNS Happy Eyeballs (RFC 8305) — parallel A + AAAA queries
# Most browsers and modern apps prefer IPv6 with fast fallback to IPv4
```

### NAT64 + DNS64

```
NAT64 translates IPv6 packets to IPv4 and vice versa:

  IPv6-only client ──[IPv6]──→ NAT64 gateway ──[IPv4]──→ IPv4 server

DNS64 synthesizes AAAA records for IPv4-only destinations:
  Client queries AAAA for ipv4only.example.com
  DNS64 gets A record: 93.184.216.34
  DNS64 returns AAAA: 64:ff9b::5db8:d822 (prefix + IPv4 embedded)

NAT64 prefix options:
  Well-Known Prefix: 64:ff9b::/96 (for global use)
  Network-Specific Prefix: any /32, /40, /48, /56, /64, /96
```

```bash
# Detect NAT64 prefix via RFC 7050 (DNS-based discovery)
dig AAAA ipv4only.arpa
# If response exists, extract the prefix from the returned address

# Linux Jool (NAT64 implementation)
sudo modprobe jool
sudo jool instance add "nat64" --netfilter --pool6 64:ff9b::/96
sudo jool pool4 add --tcp 198.51.100.1 1-65535
sudo jool pool4 add --udp 198.51.100.1 1-65535
sudo jool pool4 add --icmp 198.51.100.1 0-65535

# TAYGA (stateless NAT64 userspace)
# /etc/tayga.conf
# tun-device nat64
# ipv4-addr 192.168.255.1
# prefix 64:ff9b::/96
# dynamic-pool 192.168.255.0/24

# Cisco IOS NAT64 stateful
nat64 v6v4 list ACCESS-LIST-V6 pool NAT64-POOL overload
nat64 v6v4 list ACCESS-LIST-V6 pool NAT64-POOL
nat64 prefix stateful 64:ff9b::/96

# DNS64 with BIND
# options {
#   dns64 64:ff9b::/96 {
#     clients { any; };
#     mapped { !rfc1918; any; };
#     exclude { 64:ff9b::/96; };
#   };
# };
```

### 464XLAT (RFC 6877)

```
For IPv4-only applications on IPv6-only networks:

  IPv4 app ──[IPv4]──→ CLAT ──[IPv6]──→ PLAT (NAT64) ──[IPv4]──→ IPv4 internet

CLAT (Customer-side translator): Stateless NAT46 on the client
PLAT (Provider-side translator): Stateful NAT64 in the network

# Common on mobile networks (Android, iOS)
# Android CLAT: clatd daemon creates v4-wlan0 interface
# Provides IPv4 connectivity without carrier-grade NAT44
```

### 6to4 (Deprecated, RFC 7526)

```
# Maps IPv4 address into IPv6 prefix: 2002:<IPv4>::/48
# Example: IPv4 203.0.113.1 → 2002:cb00:7101::/48
# Encapsulates IPv6 in IPv4 protocol 41
# Relies on anycast relay at 192.88.99.1
# DEPRECATED — unreliable, asymmetric routing, relay abuse
# Do not deploy — use NAT64 or dual-stack instead
```

### 6rd (IPv6 Rapid Deployment, RFC 5969)

```
# ISP-managed improvement over 6to4
# Uses ISP's own IPv6 prefix instead of 2002::/16
# Deterministic mapping: ISP prefix + IPv4 suffix → /64 prefix
# Still tunnels IPv6 in IPv4 but with ISP-controlled relays

# Example: ISP prefix 2001:db8::/32, customer IPv4 10.1.2.3
# Customer prefix: 2001:db8:0a01:0203::/64

# Cisco IOS 6rd configuration
interface Tunnel0
 ipv6 address 2001:db8:0a01:0203::1/64
 tunnel source GigabitEthernet0/0
 tunnel mode ipv6ip 6rd
 tunnel 6rd prefix 2001:db8::/32
 tunnel 6rd br 198.51.100.1
```

### ISATAP (Intra-Site Automatic Tunnel Addressing Protocol)

```
# Creates IPv6 overlay within an IPv4 site
# IID format: ::0000:5efe:<IPv4-address>
# Example: IPv4 10.1.2.3 → fe80::5efe:a01:203
# Uses potential router list (PRL) for discovery
# Cisco: tunnel mode ipv6ip isatap
# Largely superseded by native IPv6 deployment
```

### MAP-T and MAP-E (RFC 7597, RFC 7599)

```
MAP-T (Translation):
  Stateless NAT46/64 using algorithmic address mapping
  No per-flow state on the border relay
  Each CPE gets a port range based on its IPv4 sharing ratio

MAP-E (Encapsulation):
  IPv4-in-IPv6 tunneling with algorithmic port mapping
  Similar to DS-Lite but stateless

Both solve IPv4 address sharing on IPv6-only infrastructure:
  CPE ──[MAP domain]──→ Border Relay ──→ IPv4 internet

Port allocation formula:
  Ports = (1 << (16 - offset - ratio_bits))
  Each subscriber gets a deterministic port block
```

## NPTv6 (Network Prefix Translation)

```
# Stateless prefix-to-prefix translation (RFC 6296)
# Translates between ULA (internal) and GUA (external)
# 1:1 mapping — no port translation, no state table
# Preserves end-to-end reachability (modulo prefix change)
# Checksum-neutral — adjusts IID to compensate for prefix change

Internal: fd01:0203:0405::1  →  External: 2001:db8:1::xx01
  (IID adjusted so transport checksums remain valid)

# Linux netfilter NPTv6
sudo ip6tables -t mangle -A PREROUTING -d 2001:db8:1::/48 \
  -j SNPT --src-pfx 2001:db8:1::/48 --dst-pfx fd01:0203:0405::/48
sudo ip6tables -t mangle -A POSTROUTING -s fd01:0203:0405::/48 \
  -j DNPT --src-pfx fd01:0203:0405::/48 --dst-pfx 2001:db8:1::/48

# nftables NPTv6 (Linux 5.x+)
# table ip6 nat {
#   chain prerouting {
#     type filter hook prerouting priority -300;
#     ip6 daddr 2001:db8:1::/48 ip6 daddr set fd01:0203:0405::/48
#   }
#   chain postrouting {
#     type filter hook postrouting priority 300;
#     ip6 saddr fd01:0203:0405::/48 ip6 saddr set 2001:db8:1::/48
#   }
# }

# Cisco IOS NPTv6
interface GigabitEthernet0/0
 ipv6 address fd01:0203:0405::1/48
 nat66 inside
interface GigabitEthernet0/1
 ipv6 address 2001:db8:1::1/48
 nat66 outside
nat66 prefix inside fd01:0203:0405::/48 outside 2001:db8:1::/48
```

## Tips

- Never block ICMPv6 types 133-137 (NDP) or type 2 (Packet Too Big) in firewalls. Blocking these breaks IPv6 fundamentally.
- When deploying IPv6 ACLs, always include an explicit permit for NDP and PMTUD at the top of the list.
- Use RA Guard on all access ports. Rogue RAs are the IPv6 equivalent of rogue DHCP servers and can hijack default gateways.
- Prefer NAT64/DNS64 or dual-stack over tunnel-based transition mechanisms (6to4, ISATAP) which are deprecated or unreliable.
- DHCPv6 does not provide default gateway information — that always comes from RA. Even in stateful DHCPv6 environments, RA is still required.
- Privacy extensions (RFC 8981) should be enabled on client endpoints to prevent tracking via stable EUI-64 interface identifiers.
- For OSPFv3, always configure a router-id explicitly. Unlike OSPFv2, there may be no IPv4 address to auto-derive from.
- NPTv6 is not NAT — it is stateless prefix translation. It does not break end-to-end connectivity the way NAT44 does, but it does break IPsec AH and any protocol that embeds addresses in payloads.
- When troubleshooting "IPv6 works on-link but not off-link," verify that the RA includes a non-zero Router Lifetime and that the prefix has the A flag set.
- Extension header chains can cause issues with stateless ACLs and middleboxes that only inspect the first Next Header. Consider this when designing security policies.

## See Also

- ipv6, ndp, slaac, dhcpv6, ipsec, ospf, eigrp, bgp, mpls-vpn, segment-routing, nftables, ip6tables

## References

- [RFC 8200 — IPv6 Specification](https://www.rfc-editor.org/rfc/rfc8200)
- [RFC 4861 — Neighbor Discovery for IPv6](https://www.rfc-editor.org/rfc/rfc4861)
- [RFC 4862 — IPv6 SLAAC](https://www.rfc-editor.org/rfc/rfc4862)
- [RFC 8415 — DHCPv6](https://www.rfc-editor.org/rfc/rfc8415)
- [RFC 8981 — Temporary Address Extensions (Privacy)](https://www.rfc-editor.org/rfc/rfc8981)
- [RFC 6105 — RA Guard](https://www.rfc-editor.org/rfc/rfc6105)
- [RFC 7166 — DHCPv6 Guard](https://www.rfc-editor.org/rfc/rfc7610)
- [RFC 6296 — NPTv6](https://www.rfc-editor.org/rfc/rfc6296)
- [RFC 6146 — Stateful NAT64](https://www.rfc-editor.org/rfc/rfc6146)
- [RFC 6147 — DNS64](https://www.rfc-editor.org/rfc/rfc6147)
- [RFC 6877 — 464XLAT](https://www.rfc-editor.org/rfc/rfc6877)
- [RFC 5969 — 6rd](https://www.rfc-editor.org/rfc/rfc5969)
- [RFC 7597 — MAP-E](https://www.rfc-editor.org/rfc/rfc7597)
- [RFC 7599 — MAP-T](https://www.rfc-editor.org/rfc/rfc7599)
- [RFC 5838 — OSPFv3 Address Families](https://www.rfc-editor.org/rfc/rfc5838)
- [RFC 8354 — Use Case for IPv6 SRH](https://www.rfc-editor.org/rfc/rfc8354)
- [RFC 7526 — Deprecating 6to4](https://www.rfc-editor.org/rfc/rfc7526)
- [RFC 6724 — Default Address Selection](https://www.rfc-editor.org/rfc/rfc6724)
- [Jool NAT64 — Open Source Implementation](https://www.jool.mx/)
