# PBR (Policy-Based Routing)

Forward packets based on administrator-defined policies rather than destination address alone, using route-maps to match traffic and set forwarding attributes.

## Concepts

### PBR vs Destination-Based Routing

- **Destination-based:** Router looks up destination IP in RIB/FIB; all traffic to same destination takes same path
- **Policy-based:** Route-map inspects packet attributes (source IP, protocol, length, DSCP) and overrides normal forwarding
- PBR is evaluated before the routing table lookup
- PBR applies per-interface on ingress (or locally with local policy routing)
- PBR cannot influence routing protocols; it only affects forwarding decisions for matched traffic
- If no PBR match occurs, normal destination-based forwarding resumes

### Route-Map Structure for PBR

```
route-map PBR_MAP permit 10
 match ip address ACL_MATCH
 set ip next-hop 10.0.0.1
route-map PBR_MAP permit 20
 match ip address ACL_OTHER
 set ip next-hop 10.0.0.2
route-map PBR_MAP permit 30
 ! no match — fall through to normal routing
```

- Sequences evaluated top-down; first match wins
- A permit clause without a set statement allows normal routing for matched traffic
- An implicit deny at the end drops to normal routing (not packet drop)
- Combining multiple match statements in one sequence is a logical AND

### Match Criteria

- `match ip address <ACL>` — match based on standard or extended ACL (source, destination, protocol, port)
- `match ip address prefix-list <NAME>` — match based on prefix-list
- `match length <min> <max>` — match on Layer 3 packet length (useful for separating interactive vs bulk traffic)
- `match ip next-hop <ACL>` — match based on next-hop address from routing table (rarely used in PBR)
- `match local-traffic` — match locally originated traffic (IOS-XE 17.x+)

### Set Actions

```
! Forward to specific next-hop (must be directly connected or reachable)
set ip next-hop 10.0.0.1

! Set next-hop from a list — first reachable wins
set ip next-hop 10.0.0.1 10.0.0.2

! Forward out a specific interface
set interface GigabitEthernet0/1

! Set next-hop only if no route exists in RIB for destination
set ip default next-hop 10.0.0.1

! Set default interface (used only when no explicit route exists)
set default interface GigabitEthernet0/1

! Set IP precedence (legacy QoS marking)
set ip precedence critical          ! 0-7 or name

! Set DSCP marking
set ip dscp af31                    ! DSCP value or name

! Set IP TOS byte
set ip tos 8

! Set VRF for forwarding lookup
set vrf CUSTOMER_A

! Set next-hop with recursive resolution
set ip next-hop recursive 192.168.100.1
```

### Set Action Priority Order

When multiple set actions exist, IOS applies them in this order:

1. `set ip vrf` — VRF selection first
2. `set ip next-hop` — explicit next-hop (directly connected check)
3. `set interface` — explicit output interface
4. `set ip default next-hop` — only if no RIB entry
5. `set default interface` — only if no RIB entry

## Interface PBR (Standard)

### IOS / IOS-XE

```
! Define the ACL for matching
ip access-list extended MATCH_WEB
 permit tcp any any eq 80
 permit tcp any any eq 443

! Define the route-map
route-map PBR_WEB permit 10
 match ip address MATCH_WEB
 set ip next-hop 10.1.1.1

! Apply to the ingress interface
interface GigabitEthernet0/0
 ip policy route-map PBR_WEB
```

### Verify

```bash
show route-map PBR_WEB
# Output shows match counts, set actions, and sequence numbers

show ip policy
# Output shows which interfaces have PBR applied

show route-map PBR_WEB | include matches
# Quick check if traffic is hitting the policy

debug ip policy
# Real-time PBR decision logging (use cautiously in production)
```

## Local PBR

Route traffic originated by the router itself (management traffic, generated pings, syslog, etc.).

```
! Define route-map for local traffic
route-map LOCAL_PBR permit 10
 match ip address MGMT_TRAFFIC
 set ip next-hop 10.0.0.1

! Apply globally (not on an interface)
ip local policy route-map LOCAL_PBR
```

- Local PBR applies only to traffic the router generates
- Does not affect transit traffic
- Useful for forcing management traffic through a specific ISP or VPN tunnel
- `match local-traffic` not required for local policy; all locally generated traffic is already in scope

## PBR with IP SLA Tracking

### Basic IP SLA + PBR

```
! Define IP SLA probe to primary ISP gateway
ip sla 1
 icmp-echo 203.0.113.1 source-ip 10.0.0.1
 frequency 5
ip sla schedule 1 life forever start-time now

ip sla 2
 icmp-echo 198.51.100.1 source-ip 10.0.0.1
 frequency 5
ip sla schedule 2 life forever start-time now

! Track the SLA objects
track 1 ip sla 1 reachability
track 2 ip sla 2 reachability

! Route-map with tracked next-hops
route-map DUAL_ISP permit 10
 match ip address BUSINESS_TRAFFIC
 set ip next-hop verify-availability 203.0.113.1 10 track 1
 set ip next-hop verify-availability 198.51.100.1 20 track 2

route-map DUAL_ISP permit 20
 ! All other traffic — normal routing
```

### Verify SLA and Tracking

```bash
show ip sla statistics
# Shows RTT, success/failure counts, return codes

show ip sla configuration
# Shows probe parameters

show track
# Shows track object state (Up/Down), tracked object type

show track brief
# Quick status summary of all track objects
```

## PBR for Dual-ISP / Multi-Homing

### Traffic Steering by Source

```
! Subnet A uses ISP1, Subnet B uses ISP2
ip access-list extended SUBNET_A
 permit ip 10.10.0.0 0.0.255.255 any
ip access-list extended SUBNET_B
 permit ip 10.20.0.0 0.0.255.255 any

route-map DUAL_ISP permit 10
 match ip address SUBNET_A
 set ip next-hop 203.0.113.1

route-map DUAL_ISP permit 20
 match ip address SUBNET_B
 set ip next-hop 198.51.100.1

interface GigabitEthernet0/0
 description LAN-facing
 ip policy route-map DUAL_ISP
```

### Traffic Steering by Application

```
ip access-list extended VOIP_TRAFFIC
 permit udp any any range 16384 32767
 permit udp any any eq 5060

ip access-list extended BULK_TRAFFIC
 permit tcp any any eq 443
 permit tcp any any eq 80

route-map APP_PBR permit 10
 match ip address VOIP_TRAFFIC
 set ip next-hop verify-availability 203.0.113.1 10 track 1
 set ip dscp ef

route-map APP_PBR permit 20
 match ip address BULK_TRAFFIC
 set ip next-hop 198.51.100.1
```

### Packet-Length Based Steering

```
! Small packets (interactive) via low-latency ISP
route-map LENGTH_PBR permit 10
 match length 0 500
 set ip next-hop 203.0.113.1

! Large packets (bulk transfers) via high-bandwidth ISP
route-map LENGTH_PBR permit 20
 match length 501 65535
 set ip next-hop 198.51.100.1
```

## PBR with VRF

### Inter-VRF PBR (Route Leaking Alternative)

```
! Match traffic in VRF CUSTOMER and forward via VRF INTERNET
route-map VRF_PBR permit 10
 match ip address CUSTOMER_INTERNET
 set vrf INTERNET
 set ip next-hop 10.255.0.1

! Apply on VRF-aware interface
interface GigabitEthernet0/1
 ip vrf forwarding CUSTOMER
 ip address 10.100.0.1 255.255.255.0
 ip policy route-map VRF_PBR
```

### VRF Selection PBR

```
! Classify traffic into VRFs based on source
route-map VRF_CLASSIFY permit 10
 match ip address ENGINEERING_NETS
 set vrf ENGINEERING

route-map VRF_CLASSIFY permit 20
 match ip address GUEST_NETS
 set vrf GUEST

interface GigabitEthernet0/0
 ip policy route-map VRF_CLASSIFY
```

## NX-OS PBR

### NX-OS Configuration

```
! NX-OS requires feature enablement
feature pbr

! Define route-map
route-map PBR_NXOS permit 10
 match ip address MATCH_ACL
 set ip next-hop 10.0.0.1

! Apply to interface (NX-OS uses same syntax)
interface Ethernet1/1
 ip policy route-map PBR_NXOS
```

### NX-OS PBR with Statistics

```
! Enable per-entry statistics
route-map PBR_STATS permit 10
 match ip address MATCH_ACL
 set ip next-hop 10.0.0.1
 pbr-statistics

! Verify
show route-map PBR_STATS pbr-statistics
```

### NX-OS PBR Differences from IOS

- Requires `feature pbr` to be enabled
- Supports `pbr-statistics` for per-entry hit counters
- No `set ip default next-hop` — use `set ip next-hop` with `load-share` instead
- Supports ECMP PBR with `set ip next-hop ... load-share`
- PBR applied in hardware (TCAM) on Nexus platforms
- Supports PBR for IPv6 with `match ipv6 address`
- `set ip next-hop verify-availability` supported with track objects

### NX-OS PBR Load Sharing

```
route-map PBR_ECMP permit 10
 match ip address ALL_TRAFFIC
 set ip next-hop load-share 10.0.0.1 10.0.0.2
```

## Linux PBR

### IP Rule and Multiple Routing Tables

```bash
# Linux PBR uses ip rule + ip route with named/numbered tables
# Tables defined in /etc/iproute2/rt_tables

# Add custom table
echo "100 ISP1" >> /etc/iproute2/rt_tables
echo "200 ISP2" >> /etc/iproute2/rt_tables

# Add routes to custom tables
ip route add default via 203.0.113.1 dev eth1 table ISP1
ip route add default via 198.51.100.1 dev eth2 table ISP2

# Add connected networks to each table
ip route add 10.0.0.0/24 dev eth0 table ISP1
ip route add 10.0.0.0/24 dev eth0 table ISP2

# Add rules to select tables
ip rule add from 10.10.0.0/16 table ISP1 priority 100
ip rule add from 10.20.0.0/16 table ISP2 priority 200
```

### Verify Linux PBR

```bash
# Show all rules (priority-ordered)
ip rule show
# 0:      from all lookup local
# 100:    from 10.10.0.0/16 lookup ISP1
# 200:    from 10.20.0.0/16 lookup ISP2
# 32766:  from all lookup main
# 32767:  from all lookup default

# Show routes in specific table
ip route show table ISP1
ip route show table ISP2

# Test which table a packet would use
ip route get 8.8.8.8 from 10.10.0.5
```

### Linux PBR with fwmark (iptables Integration)

```bash
# Mark packets with iptables
iptables -t mangle -A PREROUTING -p tcp --dport 80 -j MARK --set-mark 1
iptables -t mangle -A PREROUTING -p tcp --dport 443 -j MARK --set-mark 1
iptables -t mangle -A PREROUTING -p udp --dport 53 -j MARK --set-mark 2

# Route based on fwmark
ip rule add fwmark 1 table ISP1 priority 100
ip rule add fwmark 2 table ISP2 priority 200

# Add default routes to tables
ip route add default via 203.0.113.1 table ISP1
ip route add default via 198.51.100.1 table ISP2
```

### Linux PBR with nftables

```bash
# nftables marking (modern replacement for iptables)
nft add table ip mangle
nft add chain ip mangle prerouting { type filter hook prerouting priority -150 \; }
nft add rule ip mangle prerouting tcp dport { 80, 443 } meta mark set 1
nft add rule ip mangle prerouting udp dport 53 meta mark set 2

# Same ip rule/ip route as above
ip rule add fwmark 1 table ISP1 priority 100
ip rule add fwmark 2 table ISP2 priority 200
```

### Making Linux PBR Persistent

```bash
# For Debian/Ubuntu — add to /etc/network/interfaces
auto eth1
iface eth1 inet static
 address 203.0.113.2/24
 gateway 203.0.113.1
 post-up ip route add default via 203.0.113.1 table ISP1
 post-up ip rule add from 10.10.0.0/16 table ISP1 priority 100

# For RHEL/CentOS/Fedora — use /etc/sysconfig/network-scripts/rule-eth1
# from 10.10.0.0/16 table ISP1 priority 100

# For systemd-networkd — use [RoutingPolicyRule] in .network file
# [RoutingPolicyRule]
# From=10.10.0.0/16
# Table=100
# Priority=100

# For NetworkManager — use nmcli
nmcli connection modify ISP1_CONN +ipv4.routing-rules "priority 100 from 10.10.0.0/16 table 100"
```

## Troubleshooting

### PBR Not Matching Traffic

```bash
# Verify route-map is applied to the correct interface
show ip policy

# Check route-map match counters
show route-map <NAME>

# Verify ACL is matching expected traffic
show access-lists <ACL_NAME>

# Confirm next-hop is reachable
show ip route <next-hop>
ping <next-hop>

# Enable debug (use with caution)
debug ip policy
```

### PBR Next-Hop Unreachable

```bash
# If next-hop is unreachable, PBR falls back to normal routing
# Use verify-availability with track objects for controlled failover

# Check track object state
show track

# Check IP SLA probe results
show ip sla statistics

# Verify ARP entry exists for next-hop
show ip arp | include <next-hop>
```

### PBR and CEF Interaction

```
# PBR is processed in software on some older platforms
# Newer platforms (ISR 4000, Cat 9000) handle PBR in hardware via TCAM
# Verify hardware vs software switching
show ip cef <destination> detail
show platform hardware fed switch active fwd-asic resource tcam utilization
```

### Linux PBR Troubleshooting

```bash
# Verify rules are installed
ip rule show

# Trace route lookup for specific source
ip route get 8.8.8.8 from 10.10.0.5

# Check if fwmark is being set
iptables -t mangle -L -v -n

# Verify conntrack is not bypassing marks on return traffic
# (conntrack restores marks on reply packets by default)
sysctl net.ipv4.conf.all.rp_filter
# Set to 2 (loose) if PBR causes rp_filter drops
sysctl -w net.ipv4.conf.all.rp_filter=2
```

## Tips

- PBR is evaluated before the routing table; if the route-map has no match, normal routing applies (no packet drop).
- Always use `set ip next-hop verify-availability` with track objects in production to handle next-hop failures gracefully.
- PBR applied to an interface affects only ingress traffic on that interface; there is no egress PBR in IOS.
- Use `set ip default next-hop` when you want PBR to apply only when no explicit route exists (avoids overriding specific routes).
- On Nexus platforms, PBR is programmed in TCAM; check utilization with `show hardware access-list resource utilization`.
- For Linux PBR, always add connected network routes to custom tables; otherwise return traffic may use the wrong path.
- Use `ip route get` on Linux to verify which table and path a packet will actually use.
- Avoid recursive PBR next-hops unless you understand the resolution chain; prefer directly connected next-hops.
- When using PBR with VRFs, the `set vrf` action must be configured before `set ip next-hop` for correct lookup order.
- Document every PBR policy thoroughly; PBR is invisible to `show ip route` and easily forgotten during troubleshooting.
- Prefer using extended ACLs in PBR match clauses for granularity; standard ACLs only match source address.
- PBR and ECMP can coexist; PBR takes precedence for matched traffic, and unmatched traffic uses normal ECMP paths.

## See Also

- bgp, ospf, ip-sla, acl, vrf, qos, ecmp, iptables

## References

- [Cisco IOS PBR Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/iproute_pi/configuration/xe-16/irp-xe-16-book/irp-policy-based-routing.html)
- [Cisco NX-OS PBR Configuration Guide](https://www.cisco.com/c/en/us/td/docs/dcn/nx-os/nexus9000/104x/configuration/unicast-routing/cisco-nexus-9000-series-nx-os-unicast-routing-configuration-guide-104x/m-configuring-policy-based-routing.html)
- [Linux IP Rule and Route Tables (iproute2)](https://man7.org/linux/man-pages/man8/ip-rule.8.html)
- [Linux Advanced Routing & Traffic Control HOWTO — PBR](https://lartc.org/howto/lartc.rpdb.html)
- [RFC 1104 — Models of Policy Based Routing](https://www.rfc-editor.org/rfc/rfc1104)
- [RFC 7999 — BLACKHOLE Community](https://www.rfc-editor.org/rfc/rfc7999)
- [Juniper PBR (Filter-Based Forwarding)](https://www.juniper.net/documentation/us/en/software/junos/routing-policy/topics/concept/firewall-filter-option-filter-based-forwarding-overview.html)
