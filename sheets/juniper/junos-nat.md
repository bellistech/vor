# JunOS NAT (Network Address Translation)

SRX NAT translates IP addresses and ports at the security zone boundary. Three NAT types: source NAT (egress), destination NAT (ingress), static NAT (bidirectional). Rule sets define from/to zone context; rules within match and translate.

## NAT Processing Order

```
# SRX evaluates NAT in this order for EVERY packet:
#
# 1. Static NAT         (bidirectional, checked first)
# 2. Destination NAT    (ingress — before route lookup)
# 3. Route lookup        (uses translated destination)
# 4. Security policy     (uses translated addresses)
# 5. Source NAT          (egress — after policy)
# 6. Reverse translation (return traffic, automatic)
```

## Source NAT

### Interface-based source NAT (PAT — many-to-one)
```
# Translate all trust traffic to the egress interface IP
set security nat source rule-set TRUST-TO-UNTRUST from zone trust
set security nat source rule-set TRUST-TO-UNTRUST to zone untrust
set security nat source rule-set TRUST-TO-UNTRUST rule SNAT-INTERFACE match source-address 0.0.0.0/0
set security nat source rule-set TRUST-TO-UNTRUST rule SNAT-INTERFACE then source-nat interface
```

### Pool-based source NAT (many-to-many with PAT)
```
# Define a NAT pool
set security nat source pool SNAT-POOL address 203.0.113.10/32 to 203.0.113.20/32
set security nat source pool SNAT-POOL port no-translation

# Pool with port translation (PAT)
set security nat source pool SNAT-POOL-PAT address 203.0.113.10/32

# Rule referencing the pool
set security nat source rule-set TRUST-TO-UNTRUST rule SNAT-POOL match source-address 10.0.0.0/8
set security nat source rule-set TRUST-TO-UNTRUST rule SNAT-POOL then source-nat pool SNAT-POOL
```

### Pool with address range
```
set security nat source pool RANGE-POOL address 198.51.100.1/32 to 198.51.100.10/32
set security nat source pool RANGE-POOL host-address-base 10.1.1.1/32
# Maps 10.1.1.1 → 198.51.100.1, 10.1.1.2 → 198.51.100.2, etc.
```

### Persistent NAT (sticky — same source always maps to same external)
```
set security nat source pool PERSISTENT-POOL address 203.0.113.50/32
set security nat source pool PERSISTENT-POOL persistent-nat permit any-remote-host
set security nat source pool PERSISTENT-POOL persistent-nat inactivity-timeout 300
set security nat source pool PERSISTENT-POOL persistent-nat max-session-number 100

# Persistent NAT types:
#   permit target-host           — only original destination can reach back
#   permit target-host-port      — only original dest IP+port can reach back
#   permit any-remote-host       — any external host can reach the mapping
```

### Overflow pool
```
# Fallback if primary pool exhausted
set security nat source pool OVERFLOW-POOL address 203.0.113.100/32
set security nat source pool SNAT-POOL overflow-pool OVERFLOW-POOL
```

### Source NAT with no translation (exclude from NAT)
```
set security nat source rule-set TRUST-TO-UNTRUST rule NO-NAT-VPN match source-address 10.0.0.0/8
set security nat source rule-set TRUST-TO-UNTRUST rule NO-NAT-VPN match destination-address 172.16.0.0/12
set security nat source rule-set TRUST-TO-UNTRUST rule NO-NAT-VPN then source-nat off
# Place before the general SNAT rule — rules evaluated top to bottom
```

## Destination NAT

### Destination NAT with pool (port forwarding)
```
# Define destination NAT pool
set security nat destination pool WEB-SERVER address 10.1.1.100/32
set security nat destination pool WEB-SERVER address port 8080

# Rule set (from untrust to trust)
set security nat destination rule-set UNTRUST-TO-TRUST from zone untrust
set security nat destination rule-set UNTRUST-TO-TRUST rule DNAT-WEB match destination-address 203.0.113.5/32
set security nat destination rule-set UNTRUST-TO-TRUST rule DNAT-WEB match destination-port 80
set security nat destination rule-set UNTRUST-TO-TRUST rule DNAT-WEB then destination-nat pool WEB-SERVER
```

### Destination NAT pool with port range
```
set security nat destination pool APP-SERVERS address 10.1.1.0/24
set security nat destination pool APP-SERVERS address port 8000 to 8100
```

### Static destination mapping
```
set security nat destination pool STATIC-HOST address 10.2.2.50/32
set security nat destination rule-set INBOUND rule MAP-HOST match destination-address 203.0.113.50/32
set security nat destination rule-set INBOUND rule MAP-HOST then destination-nat pool STATIC-HOST
```

## Static NAT

### One-to-one bidirectional mapping
```
set security nat static rule-set STATIC-NAT from zone untrust
set security nat static rule-set STATIC-NAT rule MAP-SERVER match destination-address 203.0.113.100/32
set security nat static rule-set STATIC-NAT rule MAP-SERVER then static-nat prefix 10.1.1.100/32

# Subnet-level static NAT
set security nat static rule-set STATIC-NAT rule MAP-SUBNET match destination-address 203.0.113.0/24
set security nat static rule-set STATIC-NAT rule MAP-SUBNET then static-nat prefix 10.1.1.0/24
```

### Static NAT with port mapping
```
set security nat static rule-set STATIC-NAT rule MAP-PORT match destination-address 203.0.113.100/32
set security nat static rule-set STATIC-NAT rule MAP-PORT match destination-port 443
set security nat static rule-set STATIC-NAT rule MAP-PORT then static-nat prefix 10.1.1.100/32
set security nat static rule-set STATIC-NAT rule MAP-PORT then static-nat prefix mapped-port 8443
```

## Proxy-ARP

```
# Required when NAT pool addresses are on the same subnet as the egress interface
# SRX must respond to ARP requests for the NAT pool IPs

set security nat proxy-arp interface ge-0/0/1.0 address 203.0.113.10/32 to 203.0.113.20/32
set security nat proxy-arp interface ge-0/0/1.0 address 203.0.113.50/32
set security nat proxy-arp interface ge-0/0/1.0 address 203.0.113.100/32
```

## NAT64

```
# Translate IPv6 clients to IPv4 servers
set security nat source pool NAT64-POOL address 203.0.113.200/32
set security nat source rule-set NAT64-SNAT from zone trust-v6
set security nat source rule-set NAT64-SNAT to zone untrust-v4
set security nat source rule-set NAT64-SNAT rule V6-TO-V4 match source-address ::/0
set security nat source rule-set NAT64-SNAT rule V6-TO-V4 then source-nat pool NAT64-POOL

# Static NAT64 prefix mapping
set security nat static rule-set NAT64-STATIC from zone trust-v6
set security nat static rule-set NAT64-STATIC rule MAP64 match destination-address 64:ff9b::/96
set security nat static rule-set NAT64-STATIC rule MAP64 then static-nat prefix 0.0.0.0/0

# DNS ALG rewrites AAAA → A for NAT64
set security alg dns disable         # ensure ALG is enabled (default on)
```

## DNS ALG

```
# DNS ALG rewrites DNS responses to match NAT translations
# Enabled by default — SRX inspects DNS replies and adjusts A/AAAA records

# Disable DNS ALG (sometimes needed for DNSSEC or troubleshooting)
set security alg dns disable

# Re-enable
delete security alg dns disable

# Disable doctoring (rewriting) specifically
set security alg dns doctoring none
```

## Deterministic NAT and Port Block Allocation

```
# Deterministic NAT — predictable mapping for logging compliance
set security nat source pool DET-POOL address 203.0.113.0/24
set security nat source pool DET-POOL port deterministic-port-block-allocation block-size 128
set security nat source pool DET-POOL port deterministic-port-block-allocation max-blocks-per-address 8
set security nat source pool DET-POOL port deterministic-port-block-allocation interim-logging-interval 300

# Port block allocation (PBA) — allocate port blocks instead of individual ports
set security nat source pool PBA-POOL address 203.0.113.0/24
set security nat source pool PBA-POOL port block-allocation block-size 256
set security nat source pool PBA-POOL port block-allocation max-blocks-per-address 4
set security nat source pool PBA-POOL port block-allocation interim-logging-interval 600
```

## NAT Logging

```
# Enable session logging for NAT
set security nat source rule-set TRUST-TO-UNTRUST rule SNAT-POOL then source-nat pool SNAT-POOL
set security policies from-zone trust to-zone untrust policy ALLOW then log session-init
set security policies from-zone trust to-zone untrust policy ALLOW then log session-close

# Syslog NAT events
set system syslog file nat-log any any
set system syslog file nat-log match RT_FLOW_SESSION

# PBA/Deterministic NAT logging
set security nat source pool DET-POOL port deterministic-port-block-allocation interim-logging-interval 300
# Logs: source IP → translated IP:port-range at regular intervals
```

## NAT Rule Set Ordering and Multiple Rule Sets

```
# Multiple rule sets can exist — evaluated in configuration order
# Within a rule set, rules evaluated top to bottom, first match wins
# Use insert to reorder:
insert security nat source rule-set RS1 rule NEW-RULE before rule OLD-RULE

# Rule set context (from/to) must match the traffic:
set security nat source rule-set RS1 from zone trust
set security nat source rule-set RS1 to zone untrust
# Only packets flowing trust → untrust evaluate RS1

# From/to can also use routing-instance or interface:
set security nat source rule-set RS2 from routing-instance VR1
set security nat source rule-set RS2 to interface ge-0/0/1.0
```

## Verification Commands

```
# Show NAT configuration
show security nat source summary
show security nat destination summary
show security nat static summary

# Rule and rule-set hit counts
show security nat source rule all
show security nat destination rule all
show security nat static rule all

# Active NAT sessions
show security flow session
show security flow session nat

# NAT pool utilization
show security nat source pool all
show security nat source pool POOL-NAME

# Persistent NAT table
show security nat source persistent-nat-table all
show security nat source persistent-nat-table internal-ip 10.1.1.5

# Proxy-ARP entries
show security nat proxy-arp

# NAT64 sessions
show security flow session family inet6

# Session detail with NAT translations
show security flow session destination-prefix 10.1.1.100 extensive

# NAT resource usage
show security nat resource-usage source pool all
show security nat resource-usage destination pool all

# Clear NAT sessions
clear security nat source persistent-nat-table all
clear security flow session
```

## Tips

- Always remember the NAT processing order: static first, then destination, then source
- Security policies see translated destination IPs (post-DNAT) but original source IPs (pre-SNAT)
- Proxy-ARP is mandatory when NAT pool IPs share the egress subnet — forgetting it is the top NAT debugging issue
- Place "no NAT" rules (source-nat off) before general SNAT rules for VPN traffic exclusion
- Persistent NAT is critical for SIP, gaming, and P2P applications that need stable external mappings
- Port block allocation reduces logging volume compared to per-session NAT logging
- Deterministic NAT eliminates per-session logging entirely — mapping is computable from config
- Check pool utilization regularly: an exhausted pool silently drops new connections
- DNS ALG can break DNSSEC — disable doctoring if DNSSEC validation is required

## See Also

- junos-srx, junos-ipsec-vpn, junos-firewall-filters, iptables, nftables

## References

- [Juniper TechLibrary — Source NAT](https://www.juniper.net/documentation/us/en/software/junos/nat/topics/concept/nat-source-overview.html)
- [Juniper TechLibrary — Destination NAT](https://www.juniper.net/documentation/us/en/software/junos/nat/topics/concept/nat-destination-overview.html)
- [Juniper TechLibrary — Static NAT](https://www.juniper.net/documentation/us/en/software/junos/nat/topics/concept/nat-static-overview.html)
- [Juniper TechLibrary — Persistent NAT](https://www.juniper.net/documentation/us/en/software/junos/nat/topics/concept/nat-persistent-overview.html)
- [Juniper TechLibrary — NAT64](https://www.juniper.net/documentation/us/en/software/junos/nat/topics/concept/nat64-overview.html)
- [Juniper TechLibrary — Deterministic NAT](https://www.juniper.net/documentation/us/en/software/junos/nat/topics/concept/nat-deterministic-overview.html)
- [RFC 6146 — Stateful NAT64](https://www.rfc-editor.org/rfc/rfc6146)
- [RFC 6052 — IPv6 Addressing of IPv4/IPv6 Translators](https://www.rfc-editor.org/rfc/rfc6052)
