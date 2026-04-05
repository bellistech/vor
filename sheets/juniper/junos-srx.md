# JunOS SRX Platform (Security Services Gateway)

SRX is Juniper's next-gen firewall: flow-based stateful inspection, zones, policies, NAT, VPN, UTM, IDP, and AppSecure. Chassis cluster for HA. Data plane (SPU/NPU) handles forwarding; control plane (RE) runs JunOS.

## SRX Architecture

### Data plane vs control plane
```
Control Plane (RE)                    Data Plane (SPU/NPU)
├── JunOS kernel + daemons            ├── Flow processing engine
├── Routing protocols (rpd)           ├── Session table
├── CLI / NETCONF / REST API          ├── NAT translations
├── Policy compilation                ├── IPsec encryption/decryption
├── Logging and management            ├── UTM / IDP inspection
└── Commits → pushes to data plane    └── Hardware-accelerated forwarding

# SPU  = Services Processing Unit (high-end SRX5000 line)
# NPU  = Network Processing Unit (branch SRX300/1500/4x00)
# IOC  = I/O Card (SRX5000 — interface connectivity)
# SPC  = Services Processing Card (SRX5000 — SPU + memory)
```

### SRX series comparison
```
Branch SRX (SRX300, SRX320, SRX340, SRX345, SRX380)
├── Fixed-form factor, 1U
├── Single RE + integrated NPU
├── 64K–375K sessions
├── Typical: branch office, retail, small campus
└── Packet mode available

Mid-range SRX (SRX1500, SRX4100, SRX4200, SRX4600)
├── 1U–2U, higher throughput
├── Dedicated NPU, hardware crypto
├── 2M–10M sessions
├── Typical: campus edge, data center perimeter
└── Packet mode available

High-end SRX (SRX5400, SRX5600, SRX5800)
├── Chassis-based, modular
├── Multiple SPCs (SPU per card)
├── 10M–60M+ sessions
├── Typical: service provider, large DC, carrier-grade
└── Central point architecture with fabric interconnect
```

## Security Zones

### Configure zones
```
set security zones security-zone trust
set security zones security-zone trust host-inbound-traffic system-services ssh
set security zones security-zone trust host-inbound-traffic system-services ping
set security zones security-zone trust host-inbound-traffic protocols ospf
set security zones security-zone trust interfaces ge-0/0/0.0

set security zones security-zone untrust
set security zones security-zone untrust screen untrust-screen
set security zones security-zone untrust interfaces ge-0/0/1.0

set security zones security-zone dmz
set security zones security-zone dmz interfaces ge-0/0/2.0
set security zones security-zone dmz host-inbound-traffic system-services http
```

### Functional zones
```
# Management zone — dedicated out-of-band management
set security zones functional-zone management interfaces fxp0.0
set security zones functional-zone management host-inbound-traffic system-services ssh
set security zones functional-zone management host-inbound-traffic system-services https
```

### View zones
```
show security zones
show security zones security-zone trust
show security zones security-zone trust interfaces
```

## Security Policies

### Basic zone-to-zone policy
```
# Policy: trust → untrust (allow web traffic)
set security policies from-zone trust to-zone untrust policy ALLOW-WEB match source-address any
set security policies from-zone trust to-zone untrust policy ALLOW-WEB match destination-address any
set security policies from-zone trust to-zone untrust policy ALLOW-WEB match application junos-http
set security policies from-zone trust to-zone untrust policy ALLOW-WEB match application junos-https
set security policies from-zone trust to-zone untrust policy ALLOW-WEB then permit
set security policies from-zone trust to-zone untrust policy ALLOW-WEB then log session-close
```

### Policy actions
```
# permit    — allow and create session
# deny      — drop silently (no session created)
# reject    — drop and send TCP RST or ICMP unreachable

set security policies from-zone untrust to-zone trust policy BLOCK-ALL match source-address any
set security policies from-zone untrust to-zone trust policy BLOCK-ALL match destination-address any
set security policies from-zone untrust to-zone trust policy BLOCK-ALL match application any
set security policies from-zone untrust to-zone trust policy BLOCK-ALL then deny
set security policies from-zone untrust to-zone trust policy BLOCK-ALL then log session-init
```

### Policy with count
```
set security policies from-zone trust to-zone untrust policy ALLOW-WEB then count
```

### Policy ordering
```
# Policies evaluated top to bottom within a zone pair — first match wins
# Reorder with insert:
insert security policies from-zone trust to-zone untrust policy NEW-RULE before policy ALLOW-WEB
```

## Address Books

### Zone-based address book (legacy)
```
set security zones security-zone trust address-book address SERVERS 10.1.1.0/24
set security zones security-zone trust address-book address ADMIN-PC 10.1.1.50/32
set security zones security-zone trust address-book address-set INTERNAL address SERVERS
set security zones security-zone trust address-book address-set INTERNAL address ADMIN-PC
```

### Global address book (modern — preferred)
```
set security address-book global address WEB-SERVER 10.1.1.100/32
set security address-book global address DB-SERVER 10.1.1.200/32
set security address-book global address PUBLIC-RANGE 203.0.113.0/24
set security address-book global address-set DMZ-SERVERS address WEB-SERVER
set security address-book global address-set DMZ-SERVERS address DB-SERVER

# DNS-based address entry
set security address-book global address CLOUD-SVC dns-name api.example.com
```

## Applications and Application Sets

### Custom application
```
set applications application CUSTOM-APP protocol tcp
set applications application CUSTOM-APP destination-port 8443

set applications application CUSTOM-APP-RANGE protocol tcp
set applications application CUSTOM-APP-RANGE destination-port 8000-8100

set applications application CUSTOM-UDP protocol udp
set applications application CUSTOM-UDP destination-port 5060-5061
set applications application CUSTOM-UDP inactivity-timeout 120
```

### Application set
```
set applications application-set WEB-APPS application junos-http
set applications application-set WEB-APPS application junos-https
set applications application-set WEB-APPS application CUSTOM-APP
```

### Predefined applications
```
# JunOS includes hundreds of predefined applications:
# junos-http, junos-https, junos-ssh, junos-dns-tcp, junos-dns-udp,
# junos-bgp, junos-ospf, junos-icmp-all, junos-ftp, junos-smtp, etc.

show applications
show applications application junos-ssh
```

## Global Policies

```
# Global policies match traffic regardless of zone pair
# Evaluated AFTER zone-pair policies (lower priority)
set security policies global policy GLOBAL-DENY-MALWARE match source-address any
set security policies global policy GLOBAL-DENY-MALWARE match destination-address any
set security policies global policy GLOBAL-DENY-MALWARE match application any
set security policies global policy GLOBAL-DENY-MALWARE then deny

# Global policy with application services
set security policies global policy GLOBAL-LOG match source-address any
set security policies global policy GLOBAL-LOG match destination-address any
set security policies global policy GLOBAL-LOG match application any
set security policies global policy GLOBAL-LOG then permit
set security policies global policy GLOBAL-LOG then log session-init
set security policies global policy GLOBAL-LOG then log session-close
```

## Policy Scheduling

```
# Time-based policies — active only during configured windows
set schedulers scheduler BUSINESS-HOURS start-date 2024-01-01.00:00 stop-date 2030-12-31.23:59
set schedulers scheduler BUSINESS-HOURS monday start-time 08:00:00 stop-time 18:00:00
set schedulers scheduler BUSINESS-HOURS tuesday start-time 08:00:00 stop-time 18:00:00
set schedulers scheduler BUSINESS-HOURS wednesday start-time 08:00:00 stop-time 18:00:00
set schedulers scheduler BUSINESS-HOURS thursday start-time 08:00:00 stop-time 18:00:00
set schedulers scheduler BUSINESS-HOURS friday start-time 08:00:00 stop-time 18:00:00

# Apply schedule to policy
set security policies from-zone trust to-zone untrust policy ALLOW-SOCIAL match source-address any
set security policies from-zone trust to-zone untrust policy ALLOW-SOCIAL match destination-address any
set security policies from-zone trust to-zone untrust policy ALLOW-SOCIAL match application junos-https
set security policies from-zone trust to-zone untrust policy ALLOW-SOCIAL then permit
set security policies from-zone trust to-zone untrust policy ALLOW-SOCIAL scheduler-name BUSINESS-HOURS
```

## Packet Mode vs Flow Mode

```
# Flow mode (default): stateful inspection, session-based
# Packet mode: stateless per-packet forwarding (like a router)

# Enable packet mode globally
set security forwarding-options family inet6 mode packet-based
set security forwarding-options family mpls mode packet-based

# Selective packet mode (per-interface, requires reboot)
set security forwarding-process application-services disable-flow-on-interface ge-0/0/3.0

# Check current mode
show security flow status
```

## Chassis Cluster (HA)

### Basic cluster setup
```
# On BOTH nodes (console required for initial setup):
set chassis cluster cluster-id 1 node 0 reboot    # node 0
set chassis cluster cluster-id 1 node 1 reboot    # node 1

# Cluster interfaces (after reboot, from node 0)
set interfaces fab0 fabric-options member-interfaces ge-0/0/4    # node 0 fabric
set interfaces fab1 fabric-options member-interfaces ge-7/0/4    # node 1 fabric

# Control link (dedicated — typically fxp1)
# Configured automatically via cluster-id assignment

# Redundancy group 0 (RE — control plane HA)
set chassis cluster redundancy-group 0 node 0 priority 200
set chassis cluster redundancy-group 0 node 1 priority 100

# Redundancy group 1+ (data plane — interface failover)
set chassis cluster redundancy-group 1 node 0 priority 200
set chassis cluster redundancy-group 1 node 1 priority 100
set chassis cluster redundancy-group 1 preempt
set chassis cluster redundancy-group 1 interface-monitor ge-0/0/0 weight 255
set chassis cluster redundancy-group 1 interface-monitor ge-7/0/0 weight 255
```

### Redundant Ethernet interfaces (reth)
```
set interfaces reth0 redundant-ether-options redundancy-group 1
set interfaces reth0 unit 0 family inet address 10.1.1.1/24
set interfaces ge-0/0/0 gigether-options redundant-parent reth0    # node 0
set interfaces ge-7/0/0 gigether-options redundant-parent reth0    # node 1

set chassis cluster reth-count 4    # max number of reth interfaces
```

### Session failover
```
# Enable session synchronization between nodes
set chassis cluster redundancy-group 1 gratuitous-arp-count 4
set security flow tcp-session no-syn-check        # accept mid-stream sessions after failover
set security flow tcp-session no-sequence-check   # relax sequence checking post-failover
```

### Cluster verification
```
show chassis cluster status
show chassis cluster interfaces
show chassis cluster statistics
show chassis cluster information
show chassis cluster data-plane interfaces
```

## AppSecure

### AppID (Application Identification)
```
# Enabled by default when AppSecure license is active
# Identifies applications via signatures, context, and heuristics

# Use in security policy
set security policies from-zone trust to-zone untrust policy BLOCK-TORRENTS match application junos:BITTORRENT
set security policies from-zone trust to-zone untrust policy BLOCK-TORRENTS then deny
```

### AppFW (Application Firewall)
```
# Application firewall rule set
set security application-firewall rule-sets SOCIAL-MEDIA rule BLOCK-FACEBOOK match dynamic-application junos:FACEBOOK-ACCESS
set security application-firewall rule-sets SOCIAL-MEDIA rule BLOCK-FACEBOOK then deny

set security application-firewall rule-sets SOCIAL-MEDIA rule ALLOW-REST match dynamic-application any
set security application-firewall rule-sets SOCIAL-MEDIA rule ALLOW-REST then permit

# Apply to security policy
set security policies from-zone trust to-zone untrust policy WEB-POLICY then permit application-services application-firewall rule-set SOCIAL-MEDIA
```

### AppTrack (Application Tracking)
```
# Track application usage per zone
set security application-tracking
set security zones security-zone trust application-tracking

# View tracked applications
show security application-tracking counters
show security application-tracking session
```

### AppQoS (Application Quality of Service)
```
# Rate-limit or prioritize by application
set security application-firewall rule-sets QOS-RULES rule LIMIT-VIDEO match dynamic-application junos:YOUTUBE
set security application-firewall rule-sets QOS-RULES rule LIMIT-VIDEO then permit
# Combine with CoS rate limiting
```

## UTM and IDP Integration

```
# Attach UTM profile to a security policy
set security policies from-zone trust to-zone untrust policy ALLOW-WEB then permit application-services utm-policy UTM-POLICY

# Attach IDP policy to a security policy
set security policies from-zone trust to-zone untrust policy ALLOW-WEB then permit application-services idp-policy IDP-POLICY

# Both can be applied simultaneously
set security policies from-zone trust to-zone untrust policy FULL-INSPECT then permit application-services utm-policy UTM-POLICY
set security policies from-zone trust to-zone untrust policy FULL-INSPECT then permit application-services idp-policy IDP-POLICY
```

## Screens (IDS)

```
# Screens protect against common attacks at the zone level
set security screen ids-option untrust-screen icmp ping-death
set security screen ids-option untrust-screen icmp flood threshold 500
set security screen ids-option untrust-screen ip source-route-option
set security screen ids-option untrust-screen ip tear-drop
set security screen ids-option untrust-screen tcp syn-flood attack-threshold 500
set security screen ids-option untrust-screen tcp syn-flood alarm-threshold 256
set security screen ids-option untrust-screen tcp syn-flood source-threshold 100
set security screen ids-option untrust-screen tcp syn-flood timeout 20
set security screen ids-option untrust-screen tcp land
set security screen ids-option untrust-screen tcp winnuke

# Apply screen to zone
set security zones security-zone untrust screen untrust-screen

# View screen counters
show security screen statistics zone untrust
show security screen ids-option untrust-screen
```

## Verification Commands

```
# Security policies
show security policies
show security policies from-zone trust to-zone untrust
show security policies hit-count
show security policies detail

# Sessions
show security flow session
show security flow session summary
show security flow session count
show security flow session destination-prefix 10.1.1.0/24
show security flow session application-firewall

# Zones
show security zones
show security zones security-zone trust

# Address books
show security address-book global

# Applications
show security application-tracking counters
show security application-tracking session

# Chassis cluster
show chassis cluster status
show chassis cluster interfaces
show chassis cluster statistics

# General platform
show chassis hardware
show chassis routing-engine
show system processes extensive
show security monitoring
```

## Tips

- Always create an explicit deny-all policy as the last rule in each zone pair — easier to audit than implicit deny
- Use global address books instead of per-zone address books — they work across all zone pairs
- Policy hit counts are your best friend for auditing — find unused rules with `show security policies hit-count`
- Chassis cluster requires identical hardware and JunOS versions on both nodes
- After cluster failover, TCP sessions may need `no-syn-check` and `no-sequence-check` to survive
- Packet mode bypasses all security services — use only for transit routing segments
- AppID needs a few packets to identify the application — initial packets are allowed before AppFW can block
- Screens are evaluated before security policies — they are your first line of defense
- `reject` in a policy sends RST/ICMP back, which leaks zone existence info to attackers — use `deny` on untrust-facing rules

## See Also

- junos-nat, junos-ipsec-vpn, junos-utm, junos-ids-ips, junos-firewall-filters, junos-high-availability

## References

- [Juniper TechLibrary — SRX Series Documentation](https://www.juniper.net/documentation/us/en/software/junos/security-services/index.html)
- [Juniper TechLibrary — Security Policies](https://www.juniper.net/documentation/us/en/software/junos/security-policies/topics/concept/security-policy-overview.html)
- [Juniper TechLibrary — Security Zones](https://www.juniper.net/documentation/us/en/software/junos/security-policies/topics/concept/security-zone-overview.html)
- [Juniper TechLibrary — Chassis Cluster](https://www.juniper.net/documentation/us/en/software/junos/chassis-cluster-security-devices/topics/concept/chassis-cluster-overview.html)
- [Juniper TechLibrary — AppSecure](https://www.juniper.net/documentation/us/en/software/junos/application-identification/topics/concept/application-identification-overview.html)
- [Juniper TechLibrary — SRX Series Comparison](https://www.juniper.net/us/en/products/security/srx-series.html)
