# Network ACLs (Standard, Extended, and Named Access Control Lists)

> Filter traffic at layer 3/4 using ordered permit/deny rules with wildcard masks, applied to interfaces, VTY lines, and routing protocols.

## Concepts

### ACL Types and Number Ranges

```
# Standard ACLs — filter by source IP only
# Numbered range: 1-99, 1300-1999
# Applied closest to destination (limited matching)

# Extended ACLs — filter by source, destination, protocol, port
# Numbered range: 100-199, 2000-2699
# Applied closest to source (granular matching)

# Named ACLs — alphanumeric names instead of numbers
# Can be standard or extended
# Support per-entry sequence numbers for insertion/reordering
# Preferred over numbered ACLs for readability
```

### ACL Processing Order

```
# ACLs are processed top-down, first match wins
# Packet compared against each ACE (Access Control Entry) sequentially
# First matching rule is applied — remaining rules skipped
# Implicit deny all at the end of every ACL
# An empty ACL denies all traffic (implicit deny, no explicit permit)

# Processing flow:
#   Packet arrives → Match ACE 10? → yes → permit/deny → done
#                                   → no  → Match ACE 20? → yes → permit/deny → done
#                                                          → no  → ... → implicit deny
```

### Wildcard Masks

```
# Wildcard mask: inverse of subnet mask
# 0 = must match, 1 = don't care (opposite of subnet mask)

# Subnet mask → Wildcard mask conversion:
#   255.255.255.0   → 0.0.0.255       (match /24 network)
#   255.255.255.252 → 0.0.0.3         (match /30 network)
#   255.255.240.0   → 0.0.15.255      (match /20 network)
#   255.255.255.255 → 0.0.0.0         (match exact host)

# Special keywords:
#   host 10.0.0.1       = 10.0.0.1 0.0.0.0        (exact match)
#   any                 = 0.0.0.0 255.255.255.255  (match all)

# Non-contiguous wildcard masks (advanced):
#   0.0.0.254  — match even IPs only (last bit must be 0)
#   0.0.0.1    — match x.x.x.0 and x.x.x.1 only
```

## Standard ACLs

### IOS Configuration

```bash
# Numbered standard ACL
access-list 10 permit 10.0.0.0 0.0.255.255
access-list 10 deny   172.16.0.0 0.0.255.255
access-list 10 permit any

# Named standard ACL
ip access-list standard ALLOW_MGMT
 10 permit host 10.0.0.5
 20 permit 10.0.1.0 0.0.0.255
 30 deny   any

# Apply to interface (inbound)
interface GigabitEthernet0/1
 ip access-group 10 in

# Apply to interface (outbound)
interface GigabitEthernet0/2
 ip access-group ALLOW_MGMT out

# Standard ACL on VTY lines (restrict SSH/Telnet)
line vty 0 4
 access-class ALLOW_MGMT in
 transport input ssh
```

### Remark and Sequence Numbers

```bash
# Add remark for documentation
access-list 10 remark --- Management traffic ---

# Named ACL with explicit sequence numbers
ip access-list standard SERVERS
 10 permit host 10.0.0.1
 20 permit host 10.0.0.2
 30 deny   any

# Insert a new entry between existing ones
ip access-list standard SERVERS
 15 permit host 10.0.0.3

# Delete a specific entry
ip access-list standard SERVERS
 no 15

# Resequence (start at 10, increment by 10)
ip access-list resequence SERVERS 10 10
```

## Extended ACLs

### IOS Configuration

```bash
# Numbered extended ACL
access-list 100 permit tcp 10.0.0.0 0.0.0.255 host 192.168.1.100 eq 443
access-list 100 permit tcp 10.0.0.0 0.0.0.255 host 192.168.1.100 eq 80
access-list 100 deny   ip 10.0.0.0 0.0.0.255 192.168.1.0 0.0.0.255
access-list 100 permit ip any any

# Named extended ACL
ip access-list extended WEB_ACCESS
 10 permit tcp 10.0.0.0 0.0.0.255 host 192.168.1.100 eq 443
 20 permit tcp 10.0.0.0 0.0.0.255 host 192.168.1.100 eq 80
 30 permit icmp any any echo
 40 permit icmp any any echo-reply
 50 deny   ip any any log

# Apply closest to source
interface GigabitEthernet0/0
 ip access-group WEB_ACCESS in
```

### Protocol and Port Matching

```bash
# Protocol keywords: ip, tcp, udp, icmp, eigrp, ospf, gre, ahp, esp
# Port operators: eq (=), gt (>), lt (<), neq (!=), range (x y)

# TCP port matching
permit tcp any host 10.0.0.1 eq 22           # SSH
permit tcp any host 10.0.0.1 eq 443          # HTTPS
permit tcp any host 10.0.0.1 range 8080 8089 # custom range
permit tcp any any gt 1023                    # ephemeral ports

# UDP port matching
permit udp any host 10.0.0.1 eq 53           # DNS
permit udp any host 10.0.0.1 eq 161          # SNMP
permit udp any any range 33434 33534          # traceroute

# ICMP type matching
permit icmp any any echo                      # type 8 (ping request)
permit icmp any any echo-reply                # type 0 (ping reply)
permit icmp any any unreachable               # type 3
permit icmp any any time-exceeded             # type 11 (traceroute)
permit icmp any any packet-too-big            # type 3, code 4 (PMTUD)

# TCP flags
permit tcp any any established                # ACK or RST bit set
permit tcp any host 10.0.0.1 eq 80 syn        # SYN-only (new connections)
```

## Reflexive ACLs

### Stateful-Like Filtering

```bash
# Reflexive ACLs create temporary entries for return traffic
# Only available in named extended ACLs

# Outbound ACL — defines reflected traffic
ip access-list extended OUTBOUND
 permit tcp any any reflect TCP_MIRROR timeout 300
 permit udp any any reflect UDP_MIRROR timeout 120
 permit icmp any any reflect ICMP_MIRROR
 deny   ip any any

# Inbound ACL — evaluates reflected entries
ip access-list extended INBOUND
 evaluate TCP_MIRROR
 evaluate UDP_MIRROR
 evaluate ICMP_MIRROR
 deny   ip any any

interface GigabitEthernet0/0
 ip access-group OUTBOUND out
 ip access-group INBOUND in
```

## Time-Based ACLs

### Time Range Configuration

```bash
# Define a time range
time-range BUSINESS_HOURS
 periodic weekdays 08:00 to 18:00

time-range MAINTENANCE_WINDOW
 absolute start 02:00 15 March 2026 end 06:00 15 March 2026

time-range WEEKENDS
 periodic weekend 00:00 to 23:59

# Apply time range to ACL entry
ip access-list extended TIME_BASED
 10 permit tcp 10.0.0.0 0.0.0.255 any eq 443 time-range BUSINESS_HOURS
 20 permit tcp 10.0.0.0 0.0.0.255 any eq 80  time-range BUSINESS_HOURS
 30 deny   tcp 10.0.0.0 0.0.0.255 any eq 80
 40 permit ip any any
```

## Object-Group ACLs

### Object Groups (IOS)

```bash
# Network object group
object-group network SERVERS
 host 10.0.0.10
 host 10.0.0.11
 10.0.1.0 255.255.255.0

# Service object group
object-group service WEB_PORTS
 tcp eq 80
 tcp eq 443
 tcp eq 8080

object-group service MGMT_PORTS
 tcp eq 22
 tcp eq 3389
 udp eq 161

# ACL using object groups
ip access-list extended OBJ_ACL
 permit object-group WEB_PORTS 10.0.0.0 0.0.0.255 object-group SERVERS
 permit object-group MGMT_PORTS host 10.0.100.5 object-group SERVERS
 deny   ip any any log
```

### NX-OS Object Groups

```bash
# NX-OS uses slightly different syntax
object-group ip address SERVERS
 host 10.0.0.10
 host 10.0.0.11
 10.0.1.0/24

object-group ip port WEB_PORTS
 eq 80
 eq 443
 eq 8080

ip access-list OBJ_ACL
 permit tcp addrgroup SERVERS portgroup WEB_PORTS any
 deny   ip any any
```

## IPv6 ACLs

### IPv6 ACL Configuration

```bash
# IPv6 ACLs are always named and extended
# Implicit permit for NDP (neighbor discovery) at the end

ipv6 access-list V6_FILTER
 sequence 10 permit tcp 2001:db8:1::/48 any eq 443
 sequence 20 permit tcp 2001:db8:1::/48 any eq 80
 sequence 30 permit icmp any any nd-na           # Neighbor Advertisement
 sequence 40 permit icmp any any nd-ns           # Neighbor Solicitation
 sequence 50 permit icmp any any router-advertisement
 sequence 60 permit icmp any any router-solicitation
 sequence 70 deny ipv6 any any log

# Apply to interface
interface GigabitEthernet0/0
 ipv6 traffic-filter V6_FILTER in

# IPv6 ACL on VTY
line vty 0 4
 ipv6 access-class V6_FILTER in
```

## ACL for Routing Protocol Filtering

### Distribute Lists

```bash
# Filter OSPF routes using ACL
router ospf 1
 distribute-list 10 in

access-list 10 deny   10.0.99.0 0.0.0.255
access-list 10 permit any

# Filter EIGRP routes
router eigrp 100
 distribute-list 20 out GigabitEthernet0/1

access-list 20 permit 10.0.0.0 0.0.255.255
access-list 20 deny   any
```

### Prefix Lists (Preferred over ACLs for Route Filtering)

```bash
# Prefix list — more efficient than ACLs for route filtering
ip prefix-list ROUTES_IN seq 10 permit 10.0.0.0/8 le 24
ip prefix-list ROUTES_IN seq 20 deny   0.0.0.0/0 le 32

router ospf 1
 distribute-list prefix ROUTES_IN in
```

## SNMP ACL Restriction

```bash
# Restrict SNMP access with ACL
access-list 30 permit host 10.0.100.5
access-list 30 permit 10.0.100.0 0.0.0.255
access-list 30 deny   any

snmp-server community PUBLIC ro 30
snmp-server community PRIVATE rw 30
```

## ACL Logging and Monitoring

### Logging

```bash
# Log keyword — generates syslog on match (rate-limited)
access-list 100 deny tcp any any eq 23 log
access-list 100 deny ip any any log

# Log-input — includes ingress interface and MAC address
access-list 100 deny ip any any log-input

# Control ACL log rate
ip access-list log-update threshold 500

# Show ACL hit counts
show access-lists
show access-lists 100
show ip access-lists WEB_ACCESS
show ipv6 access-list V6_FILTER

# Clear ACL counters
clear access-list counters
clear access-list counters 100
```

### NX-OS ACL Statistics

```bash
# NX-OS per-entry stats
show ip access-lists WEB_ACCESS
ip access-list WEB_ACCESS statistics per-entry

# Show ACL applied to interfaces
show running-config aclmgr
show ip access-lists summary
```

## Verification and Troubleshooting

```bash
# Show all ACLs
show access-lists
show ip access-lists

# Show ACL applied to interface
show ip interface GigabitEthernet0/0 | include access
show running-config interface GigabitEthernet0/0

# Show which interfaces have ACLs
show ip interface | include line|access

# Verify VTY ACL
show line vty 0 4 | include access

# Check ACL hit counts (non-zero = matching traffic)
show access-lists 100

# Test ACL matching against a specific packet (IOS-XE)
show ip access-lists WEB_ACCESS | include 10.0.0.5

# Debug ACL (use sparingly — CPU intensive)
debug ip packet 100 detail
```

## ACL Performance Considerations

```
# Standard ACLs: fast (source-only match)
# Extended ACLs: moderate (5-tuple match)
# Object-group ACLs: expanded at install time — same TCAM use as equivalent extended ACL

# TCAM (Ternary Content-Addressable Memory)
# - Hardware-based ACL lookup on switches/routers
# - ACL entries compiled into TCAM entries
# - Lookup is O(1) regardless of ACL size
# - TCAM is finite — "show platform tcam utilization"

# Performance tips:
# - Place most-matched entries first (reduces software processing)
# - In hardware (TCAM), order doesn't affect speed
# - Merge overlapping ACLs to reduce TCAM usage
# - Use object-groups to simplify management (same TCAM footprint)
# - Turbo ACL (IOS): auto-compiled for ACLs with 3+ entries
# - Avoid log/log-input on high-traffic entries (CPU impact)
```

## Tips

- Always include a final explicit `deny any any log` to track dropped traffic and avoid silent drops.
- Apply extended ACLs closest to the source to filter traffic early and save bandwidth.
- Apply standard ACLs closest to the destination since they can only match source IPs.
- Use named ACLs over numbered for readability, editability, and sequence number support.
- Remember the implicit deny: an ACL with zero permits blocks everything.
- Use `remark` entries to document the purpose of each section in the ACL.
- Never apply an empty ACL to an interface — it will deny all traffic immediately.
- Use `show access-lists` regularly to check hit counts and identify unused or stale entries.
- Prefer prefix-lists over ACLs for BGP/OSPF/EIGRP route filtering — they are more efficient and flexible.

## See Also

- ipsec, nftables, iptables, ipv4, ipv6, ospf, eigrp, bgp, snmp, vlan, private-vlans

## References

- [RFC 3550 — IP Access Control Lists on Cisco](https://www.cisco.com/c/en/us/support/docs/security/ios-firewall/23602-confaccesslists.html)
- [Cisco IOS ACL Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/sec_data_acl/configuration/xe-16/sec-data-acl-xe-16-book.html)
- [Cisco NX-OS ACL Configuration Guide](https://www.cisco.com/c/en/us/td/docs/dcn/nx-os/nexus9000/104x/configuration/security/cisco-nexus-9000-nx-os-security-configuration-guide-104x/m-configuring-ip-acls.html)
- [Cisco IPv6 ACL Configuration](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/ipv6_basic/configuration/xe-16/ip6b-xe-16-book/ip6-access-control-lists.html)
- [Cisco TCAM and ACL Performance](https://www.cisco.com/c/en/us/td/docs/switches/lan/catalyst9300/software/release/17-6/configuration_guide/sec/b_176_sec_9300_cg/configuring_acl.html)
- [Juniper Firewall Filters (Equivalent to ACLs)](https://www.juniper.net/documentation/us/en/software/junos/routing-policy/topics/concept/firewall-filter-overview.html)
- [RFC 2827 — Network Ingress Filtering (BCP 38)](https://www.rfc-editor.org/rfc/rfc2827)
- [RFC 3704 — Ingress Filtering for Multihomed Networks (BCP 84)](https://www.rfc-editor.org/rfc/rfc3704)
