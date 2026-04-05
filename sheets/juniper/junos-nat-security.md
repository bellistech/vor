# JunOS NAT Security Considerations

NAT on SRX platforms integrates with security policies, ALGs, VPN, and HA. NAT processing occurs after route lookup but interacts with zone-based policy enforcement — understanding the ordering is critical for correct rule design.

## NAT and Security Policy Interaction

### Processing order
```
Ingress packet
  └→ Route lookup (ingress zone determined)
     └→ Destination NAT (DNAT) applied BEFORE policy lookup
        └→ Security policy lookup (uses ORIGINAL source, TRANSLATED destination)
           └→ Source NAT (SNAT) applied AFTER policy match
              └→ Forwarding

# Key insight:
#   - DNAT happens BEFORE the security policy check
#   - SNAT happens AFTER the security policy check
#   - Policy must match: original-src + translated-dst + original-dst-port (if DNAT changes port)
```

### Policy with destination NAT
```
# DNAT rule: external 203.0.113.10 → internal 10.1.1.100
set security nat destination pool WEB-SERVER address 10.1.1.100/32
set security nat destination rule-set INBOUND from zone untrust
set security nat destination rule-set INBOUND rule WEB-DNAT match destination-address 203.0.113.10/32
set security nat destination rule-set INBOUND rule WEB-DNAT match destination-port 80
set security nat destination rule-set INBOUND rule WEB-DNAT then destination-nat pool WEB-SERVER

# Security policy must reference the TRANSLATED destination
set security policies from-zone untrust to-zone trust policy ALLOW-WEB match source-address any
set security policies from-zone untrust to-zone trust policy ALLOW-WEB match destination-address 10.1.1.100/32
set security policies from-zone untrust to-zone trust policy ALLOW-WEB match application junos-http
set security policies from-zone untrust to-zone trust policy ALLOW-WEB then permit
```

### Policy with source NAT
```
# SNAT rule: internal 10.1.1.0/24 → PAT via interface
set security nat source rule-set OUTBOUND from zone trust
set security nat source rule-set OUTBOUND to zone untrust
set security nat source rule-set OUTBOUND rule PAT-OUT match source-address 10.1.1.0/24
set security nat source rule-set OUTBOUND rule PAT-OUT then source-nat interface

# Security policy uses ORIGINAL source address (before SNAT)
set security policies from-zone trust to-zone untrust policy ALLOW-OUT match source-address 10.1.1.0/24
set security policies from-zone trust to-zone untrust policy ALLOW-OUT match destination-address any
set security policies from-zone trust to-zone untrust policy ALLOW-OUT match application any
set security policies from-zone trust to-zone untrust policy ALLOW-OUT then permit
```

## Twice NAT (Source + Destination NAT)

### Overlapping address spaces
```
# Both sides use 10.1.1.0/24 — NAT both source and destination
# Destination NAT: translate dst 192.168.1.100 → 10.1.1.100
set security nat destination pool REMOTE-SERVER address 10.1.1.100/32
set security nat destination rule-set TWICE-DST from zone untrust
set security nat destination rule-set TWICE-DST rule DNAT-OVERLAP match destination-address 192.168.1.100/32
set security nat destination rule-set TWICE-DST rule DNAT-OVERLAP then destination-nat pool REMOTE-SERVER

# Source NAT: translate src 10.1.1.0/24 → 172.16.1.0/24
set security nat source pool MAPPED-SRC address 172.16.1.1/32 to 172.16.1.254/32
set security nat source rule-set TWICE-SRC from zone trust
set security nat source rule-set TWICE-SRC to zone untrust
set security nat source rule-set TWICE-SRC rule SNAT-OVERLAP match source-address 10.1.1.0/24
set security nat source rule-set TWICE-SRC rule SNAT-OVERLAP match destination-address 10.1.1.0/24
set security nat source rule-set TWICE-SRC rule SNAT-OVERLAP then source-nat pool MAPPED-SRC
```

## NAT Pools

### Source NAT pool with port translation
```
set security nat source pool SNAT-POOL address 203.0.113.20/32 to 203.0.113.30/32
set security nat source pool SNAT-POOL port no-translation    # 1:1 NAT (no PAT)

# With port translation (PAT) — default behavior
set security nat source pool PAT-POOL address 203.0.113.50/32
# PAT uses ports 1024-65535 per IP — ~64,000 sessions per address
```

### Pool sizing
```
# Sessions per IP with PAT: ~64,000 (ports 1024-65535)
# Required IPs = peak_concurrent_sessions / 64000
# Example: 250,000 concurrent sessions → 250000/64000 = 4 IPs minimum

# Overflow handling
set security nat source pool SNAT-POOL overflow-pool interface   # fall back to interface NAT
set security nat source pool SNAT-POOL address-persistent        # same src always maps to same NAT IP
```

### Destination NAT pool with port mapping
```
set security nat destination pool HTTPS-SERVER address 10.1.1.100/32
set security nat destination pool HTTPS-SERVER address port 8443

# Maps external port 443 → internal port 8443
set security nat destination rule-set INBOUND rule HTTPS match destination-port 443
set security nat destination rule-set INBOUND rule HTTPS then destination-nat pool HTTPS-SERVER
```

## Persistent NAT

### Modes
```
# any-remote-host: same src IP:port always maps to same NAT IP:port
# ANY external host can reach the mapped address (most permissive)
set security nat source pool PERSIST-POOL address 203.0.113.40/32
set security nat source pool PERSIST-POOL persistent-nat permit any-remote-host

# target-host: same src+dst pair maps to same NAT IP:port
# Only the original target host can send return traffic
set security nat source pool PERSIST-POOL persistent-nat permit target-host

# target-host-port: same src+dst+port tuple maps to same NAT IP:port
# Only original target host AND port can send return traffic (most restrictive)
set security nat source pool PERSIST-POOL persistent-nat permit target-host-port
```

### Persistent NAT configuration
```
set security nat source pool VoIP-POOL address 203.0.113.60/32
set security nat source pool VoIP-POOL persistent-nat permit any-remote-host
set security nat source pool VoIP-POOL persistent-nat inactivity-timeout 300
set security nat source pool VoIP-POOL persistent-nat max-session-number 100

set security nat source rule-set OUTBOUND rule VoIP-NAT match source-address 10.1.2.0/24
set security nat source rule-set OUTBOUND rule VoIP-NAT match application junos-sip
set security nat source rule-set OUTBOUND rule VoIP-NAT then source-nat pool VoIP-POOL
```

## NAT ALGs (Application Layer Gateways)

### ALG overview
```
# ALGs inspect payload to rewrite embedded IP addresses/ports
# Required when application-layer data contains addressing info

Enabled by default:    SIP, FTP, DNS, TFTP, RSH, RTSP, PPTP, talk/ntalk
Disabled by default:   H.323, SCCP, MGCP, MSRPC, Sun-RPC, SQL, IKE-ESP

show security alg status           # show which ALGs are enabled
```

### SIP ALG
```
# SIP embeds IPs in SDP body (media negotiation) and Via/Contact headers
# ALG rewrites: Contact, Via, SDP c= (connection), SDP m= (media port), Route/Record-Route

set security alg sip enable
set security alg sip application-screen protect deny
set security alg sip retain-hold-resource
set security alg sip inactive-media-timeout 120

# Disable SIP ALG (common when SIP proxy handles NAT traversal)
set security alg sip disable
```

### FTP ALG
```
# FTP PORT/PASV commands contain IP:port for data channel
# ALG rewrites PORT command IP and opens pinhole for data connection

set security alg ftp enable
set security alg ftp ftps-extension allow-encrypted     # handle FTPS (explicit TLS)

# FTP data channel pinhole: ALG opens temporary security policy for PORT-negotiated data
```

### DNS ALG
```
# DNS ALG inspects responses for embedded addresses (A/AAAA records)
# Rewrites DNS responses when DNAT is in play (DNS doctoring)

set security alg dns enable
set security alg dns maximum-message-length 8192
set security alg dns disable-on-tcp                     # only process UDP DNS
```

### H.323 ALG
```
# H.323 suite: H.225 (call signaling), H.245 (media control), RAS
# ALG handles dynamic port negotiation for media streams

set security alg h323 enable
set security alg h323 application-screen message-flood threshold 1000
```

### RTSP ALG
```
# RTSP controls media streaming — embeds transport ports in SETUP response
# ALG rewrites Transport header and opens pinhole for RTP/RTCP

set security alg rtsp enable
```

## NAT Traversal for VPN

### IPsec NAT-T
```
# NAT-T encapsulates ESP in UDP port 4500 to traverse NAT devices
# SRX auto-detects NAT via IKE vendor ID payloads

set security ike gateway VPN-GW address 203.0.113.100
set security ike gateway VPN-GW ike-policy IKE-POL
set security ike gateway VPN-GW external-interface ge-0/0/0
set security ike gateway VPN-GW nat-keepalive 20       # keepalive interval (seconds)

# NAT-T is enabled by default on SRX — IKE floats from 500 to 4500 when NAT detected
# Verify:
show security ike security-associations detail | match nat
```

### Exempting VPN traffic from NAT
```
# Source NAT exemption for VPN-bound traffic (common pattern)
set security nat source rule-set OUTBOUND rule NO-NAT-VPN match source-address 10.1.0.0/16
set security nat source rule-set OUTBOUND rule NO-NAT-VPN match destination-address 10.2.0.0/16
set security nat source rule-set OUTBOUND rule NO-NAT-VPN then source-nat off

# Place this rule BEFORE the general PAT rule — first match wins
```

## NAT with HA (Chassis Cluster)

### Session synchronization
```
# NAT sessions are synchronized via RTO (real-time objects) over the fabric link
# On failover, NAT mappings survive — active sessions continue without re-establishment

# Verify NAT session sync
show chassis cluster status
show security nat source summary
show security nat source persistent-nat-table all

# Persistent NAT table is synced — external hosts can still reach mapped addresses after failover
```

### Considerations
```
# Interface-based SNAT: uses reth interface IP (shared between nodes)
# Pool-based SNAT: pool addresses shared — both nodes use same pool
# Node-specific NAT: avoid — breaks on failover
# ARP for NAT pool addresses: handled by reth interface or proxy-arp on active node
```

## NAT Monitoring and Logging

### Session monitoring
```
show security nat source summary                        # source NAT rule hit counts
show security nat destination summary                    # destination NAT rule hit counts
show security nat static summary                         # static NAT rule hit counts
show security nat source pool all                        # pool utilization (used/total)
show security nat source persistent-nat-table all        # persistent NAT mappings
show security nat source rule all                        # detailed rule statistics
show security nat destination rule all                   # detailed DNAT rule statistics
show security flow session nat                           # active sessions with NAT info
```

### Logging
```
# Enable NAT session logging (uses security log infrastructure)
set security nat source rule-set OUTBOUND rule PAT-OUT then source-nat pool SNAT-POOL
set security log mode stream
set security log source-address 10.1.1.1
set security log stream NAT-LOG host 10.5.0.10
set security log stream NAT-LOG host port 514
set security log stream NAT-LOG format sd-syslog
set security log stream NAT-LOG category all

# Session-based logging captures:
#   - Session init: original 5-tuple + translated 5-tuple
#   - Session close: same + bytes/packets transferred + duration
```

### SNMP monitoring
```
set snmp community public authorization read-only
set snmp view ALL oid .1 include
# NAT MIB: jnxJsNatObjects (enterprise .1.3.6.1.4.1.2636.3.39.1.7)
# Traps for pool exhaustion
set security nat source pool SNAT-POOL pool-utilization-alarm raise-threshold 80
set security nat source pool SNAT-POOL pool-utilization-alarm clear-threshold 60
```

## NAT Troubleshooting

### Methodology
```
Step 1: Verify NAT rules are matching
  show security nat source rule all
  show security nat destination rule all
  # Check "Translation hits" — if 0, rule is not matching traffic

Step 2: Verify security policy is correct
  show security policies hit-count
  # Remember: policy uses original-src + translated-dst

Step 3: Check session table
  show security flow session source-prefix 10.1.1.0/24
  show security flow session nat
  # Verify session shows correct translated addresses

Step 4: Check NAT pool utilization
  show security nat source pool all
  # Pool exhaustion = new connections fail silently

Step 5: Trace NAT processing
  set security flow traceoptions file nat-trace
  set security flow traceoptions flag all
  set security flow traceoptions packet-filter TRACE-NAT source-prefix 10.1.1.0/24
  # WARNING: traceoptions is CPU-intensive — use specific filters, disable when done

Step 6: Verify ALG behavior
  show security alg status
  show security flow session application sip
  # If ALG is mangling packets incorrectly, try disabling it
```

### Common issues
```
# NAT rule not matching:
#   - Check rule-set from-zone / to-zone (or from-interface / to-interface)
#   - Check match conditions against actual packet 5-tuple
#   - Rule ordering — earlier rule may be matching first

# Sessions failing despite NAT rule match:
#   - Security policy references pre-NAT destination (should use post-DNAT address)
#   - Missing return route for NAT pool addresses
#   - Pool exhaustion (show security nat source pool all)

# ALG issues:
#   - Encrypted payload (ALG cannot inspect TLS)
#   - ALG disabled for required protocol
#   - ALG and application incompatibility — try disabling ALG

# HA failover NAT issues:
#   - Persistent NAT table not synced (check fabric link)
#   - ARP not updating for NAT pool addresses on new active node
```

## Tips

- DNAT is evaluated before security policy; SNAT is evaluated after — this is the most common source of NAT+policy misconfigurations
- Always place VPN NAT exemption rules before general PAT rules — first match wins
- Persistent NAT is required for protocols where external hosts initiate connections to a NATted address (VoIP, gaming, P2P)
- `any-remote-host` persistent NAT is the least secure — any host can reach the mapping; use `target-host` or `target-host-port` where possible
- SIP ALG is often more trouble than it is worth — if a SIP proxy handles NAT traversal, disable the ALG
- Monitor pool utilization and set alarm thresholds before exhaustion causes silent failures
- NAT session logging is essential for compliance (PCI DSS, audit trails) — log both session-init and session-close
- In HA, always use reth interfaces or shared pools — node-specific NAT addresses break on failover
- Twice NAT is the solution for overlapping address spaces — design the mapping table carefully before implementation
- DNS ALG (DNS doctoring) rewrites A records in responses — verify it is not mangling responses for external domains

## See Also

- junos-security-policies, junos-ha-security, junos-screens, ipsec

## References

- [Juniper TechLibrary — Source NAT](https://www.juniper.net/documentation/us/en/software/junos/nat/topics/concept/nat-source-overview.html)
- [Juniper TechLibrary — Destination NAT](https://www.juniper.net/documentation/us/en/software/junos/nat/topics/concept/nat-destination-overview.html)
- [Juniper TechLibrary — Persistent NAT](https://www.juniper.net/documentation/us/en/software/junos/nat/topics/concept/nat-persistent-overview.html)
- [Juniper TechLibrary — NAT ALGs](https://www.juniper.net/documentation/us/en/software/junos/nat/topics/concept/nat-alg-overview.html)
- [Juniper TechLibrary — NAT with Chassis Cluster](https://www.juniper.net/documentation/us/en/software/junos/chassis-cluster-security/topics/concept/chassis-cluster-nat.html)
- [RFC 3022 — Traditional IP Network Address Translator](https://www.rfc-editor.org/rfc/rfc3022)
- [RFC 4787 — NAT Behavioral Requirements for UDP](https://www.rfc-editor.org/rfc/rfc4787)
- [RFC 3947 — Negotiation of NAT-Traversal in the IKE](https://www.rfc-editor.org/rfc/rfc3947)
