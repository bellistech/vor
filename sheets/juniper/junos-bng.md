# JunOS BNG -- Broadband Network Gateway (MX Series)

Quick reference for Junos BNG subscriber management, PPPoE, IPoE, AAA, dynamic profiles, QoS, and verification on MX Series routers.

## BNG Architecture Overview

```
Subscriber CPE
    |
    v
Access Network (DSLAM/OLT via VLAN/S-VLAN)
    |
    v
MX Series BNG (MX240/MX480/MX960/MX2010/MX2020)
    |-- PPPoE termination
    |-- IPoE (DHCP) termination
    |-- RADIUS AAA
    |-- Per-subscriber QoS
    |-- Dynamic profiles
    |-- Address assignment
    |
    v
Core / Internet

Key components on the MX:
  - Memory Type Buffer (MTB) line cards for subscriber density
  - Memory Enhanced Type 5 (MPC7E/8E/9E) for high-scale BNG
  - Memory Enhanced Type 11 (MPC11E) for next-gen scale
  - Memory Type Compact (MPC10E) for cost-optimized BNG
  - Memory Enhanced Type 14 (MPC14E) for disaggregated platforms
  - Memory Buffer Processor (MBP) for per-sub queuing
```

## PPPoE Configuration

### Access Profile (AAA Binding)

```
set access profile PPPoE-ACCESS-PROFILE authentication-order radius
set access profile PPPoE-ACCESS-PROFILE radius authentication-server 10.0.0.100
set access profile PPPoE-ACCESS-PROFILE radius accounting-server 10.0.0.100
set access profile PPPoE-ACCESS-PROFILE radius options nas-identifier "MX-BNG-01"
set access profile PPPoE-ACCESS-PROFILE radius options nas-port-type Ethernet
set access profile PPPoE-ACCESS-PROFILE accounting order radius
set access profile PPPoE-ACCESS-PROFILE accounting accounting-stop-on-failure
set access profile PPPoE-ACCESS-PROFILE accounting accounting-stop-on-access-deny
```

### PPPoE Underlying Interface

```
# Configure the access-facing interface for PPPoE
set interfaces ge-1/0/0 flexible-vlan-tagging
set interfaces ge-1/0/0 auto-configure vlan-ranges dynamic-profile PPPoE-DYNAMIC
set interfaces ge-1/0/0 auto-configure vlan-ranges accept pppoe
set interfaces ge-1/0/0 auto-configure vlan-ranges ranges vlan-range 100-4000
set interfaces ge-1/0/0 encapsulation flexible-ethernet-services

# Stacked VLAN (S-VLAN + C-VLAN)
set interfaces ge-1/0/0 flexible-vlan-tagging
set interfaces ge-1/0/0 auto-configure stacked-vlan-ranges dynamic-profile PPPoE-DYNAMIC
set interfaces ge-1/0/0 auto-configure stacked-vlan-ranges accept pppoe
set interfaces ge-1/0/0 auto-configure stacked-vlan-ranges ranges vlan-range 100-200 inner-vlan-range 1-4094
```

### PPPoE Dynamic Profile

```
set dynamic-profiles PPPoE-DYNAMIC interfaces pp0 unit "$junos-interface-unit"
set dynamic-profiles PPPoE-DYNAMIC interfaces pp0 unit "$junos-interface-unit" ppp-options chap
set dynamic-profiles PPPoE-DYNAMIC interfaces pp0 unit "$junos-interface-unit" ppp-options pap
set dynamic-profiles PPPoE-DYNAMIC interfaces pp0 unit "$junos-interface-unit" pppoe-options underlying-interface "$junos-underlying-interface"
set dynamic-profiles PPPoE-DYNAMIC interfaces pp0 unit "$junos-interface-unit" pppoe-options server
set dynamic-profiles PPPoE-DYNAMIC interfaces pp0 unit "$junos-interface-unit" family inet unnumbered-address lo0.0
set dynamic-profiles PPPoE-DYNAMIC interfaces pp0 unit "$junos-interface-unit" family inet negotiate-address
```

### PPPoE with PAP/CHAP Authentication

```
# CHAP only
set dynamic-profiles PPPoE-DYNAMIC interfaces pp0 unit "$junos-interface-unit" ppp-options chap

# PAP only
set dynamic-profiles PPPoE-DYNAMIC interfaces pp0 unit "$junos-interface-unit" ppp-options pap

# Both (CHAP preferred, PAP fallback)
set dynamic-profiles PPPoE-DYNAMIC interfaces pp0 unit "$junos-interface-unit" ppp-options chap
set dynamic-profiles PPPoE-DYNAMIC interfaces pp0 unit "$junos-interface-unit" ppp-options pap

# Set the access profile for authentication
set dynamic-profiles PPPoE-DYNAMIC interfaces pp0 unit "$junos-interface-unit" pppoe-options access-concentrator BNG-01
```

## IPoE Configuration (DHCP)

### DHCP Local Server

```
# Define address pool
set access address-assignment pool POOL-RESIDENTIAL family inet network 10.100.0.0/16
set access address-assignment pool POOL-RESIDENTIAL family inet range RANGE-1 low 10.100.0.10
set access address-assignment pool POOL-RESIDENTIAL family inet range RANGE-1 high 10.100.255.254
set access address-assignment pool POOL-RESIDENTIAL family inet dhcp-attributes router 10.100.0.1
set access address-assignment pool POOL-RESIDENTIAL family inet dhcp-attributes name-server 8.8.8.8
set access address-assignment pool POOL-RESIDENTIAL family inet dhcp-attributes name-server 8.8.4.4
set access address-assignment pool POOL-RESIDENTIAL family inet dhcp-attributes lease-time 86400

# Configure DHCP local server
set system services dhcp-local-server group DHCP-GROUP interface ge-1/0/0.0
set system services dhcp-local-server group DHCP-GROUP access-profile IPoE-ACCESS-PROFILE
set system services dhcp-local-server pool-match-order ip-address-first
```

### IPoE Dynamic Profile

```
set dynamic-profiles IPoE-DYNAMIC interfaces demux0 unit "$junos-interface-unit"
set dynamic-profiles IPoE-DYNAMIC interfaces demux0 unit "$junos-interface-unit" demux-options underlying-interface "$junos-underlying-interface"
set dynamic-profiles IPoE-DYNAMIC interfaces demux0 unit "$junos-interface-unit" family inet unnumbered-address lo0.0
set dynamic-profiles IPoE-DYNAMIC interfaces demux0 unit "$junos-interface-unit" family inet address "$junos-subscriber-ip-address/$junos-subscriber-netmask"
```

### IPoE with Auto-Configure (VLAN-Based)

```
set interfaces ge-1/0/0 flexible-vlan-tagging
set interfaces ge-1/0/0 auto-configure vlan-ranges dynamic-profile IPoE-DYNAMIC
set interfaces ge-1/0/0 auto-configure vlan-ranges accept dhcp
set interfaces ge-1/0/0 auto-configure vlan-ranges ranges vlan-range 100-4000
set interfaces ge-1/0/0 encapsulation flexible-ethernet-services
```

### DHCP Relay

```
# DHCP relay to external server
set forwarding-options dhcp-relay group RELAY-GROUP interface ge-1/0/0.0
set forwarding-options dhcp-relay group RELAY-GROUP active-server-group DHCP-SERVERS
set forwarding-options dhcp-relay group RELAY-GROUP access-profile RELAY-PROFILE

set forwarding-options dhcp-relay server-group DHCP-SERVERS 10.0.0.200
set forwarding-options dhcp-relay server-group DHCP-SERVERS 10.0.0.201

# Relay options
set forwarding-options dhcp-relay group RELAY-GROUP relay-option-82 circuit-id
set forwarding-options dhcp-relay group RELAY-GROUP relay-option-82 remote-id
```

### DHCPv6 (Dual-Stack)

```
set system services dhcp-local-server dhcpv6 group DHCPv6-GROUP interface ge-1/0/0.0
set access address-assignment pool POOLv6-PD family inet6 prefix 2001:db8::/32
set access address-assignment pool POOLv6-PD family inet6 range RANGE-PD prefix-length 56

set dynamic-profiles DUAL-STACK interfaces demux0 unit "$junos-interface-unit" family inet6 address "$junos-subscriber-ipv6-address"
set dynamic-profiles DUAL-STACK interfaces demux0 unit "$junos-interface-unit" family inet6 dhcpv6-options prefix-delegation "$junos-subscriber-ipv6-prefix"
```

## AAA / RADIUS

### RADIUS Server Configuration

```
set access radius-server 10.0.0.100 port 1812
set access radius-server 10.0.0.100 accounting-port 1813
set access radius-server 10.0.0.100 secret "$9$encrypted-secret"
set access radius-server 10.0.0.100 timeout 5
set access radius-server 10.0.0.100 retry 3
set access radius-server 10.0.0.100 source-address 192.168.1.1

# Redundant RADIUS
set access radius-server 10.0.0.101 port 1812
set access radius-server 10.0.0.101 accounting-port 1813
set access radius-server 10.0.0.101 secret "$9$encrypted-secret"
```

### RADIUS Authentication Attributes

```
Common RADIUS attributes used in Junos BNG:

Attribute                      | Use
-------------------------------|--------------------------------------------
User-Name (1)                  | PPPoE username / MAC for IPoE
User-Password (2)              | PAP password
CHAP-Password (3)              | CHAP response
NAS-IP-Address (4)             | BNG loopback IP
NAS-Port (5)                   | Subscriber interface index
Framed-IP-Address (8)          | IP assigned to subscriber
Framed-IP-Netmask (9)          | Mask for assigned IP
Framed-Protocol (7)            | PPP = 1
Framed-Pool (88)               | Local pool name from RADIUS
Session-Timeout (27)           | Max session duration (seconds)
Acct-Session-Id (44)           | Unique accounting session ID
Acct-Status-Type (40)          | Start(1)/Stop(2)/Interim-Update(3)

Juniper VSAs (Vendor 2636):
  Juniper-Local-User-Name       | Maps to local user template
  Juniper-Primary-Dns           | Primary DNS for subscriber
  Juniper-Secondary-Dns         | Secondary DNS for subscriber
  Juniper-Switching-Filter      | Dynamic firewall filter
  Juniper-Cos-Parameter         | CoS profile name
  Juniper-Ingress-Policy-Name   | Ingress firewall filter
  Juniper-Egress-Policy-Name    | Egress firewall filter
```

### RADIUS Accounting

```
set access profile PPPoE-ACCESS-PROFILE accounting order radius
set access profile PPPoE-ACCESS-PROFILE accounting accounting-stop-on-failure
set access profile PPPoE-ACCESS-PROFILE accounting accounting-stop-on-access-deny
set access profile PPPoE-ACCESS-PROFILE accounting immediate-update
set access profile PPPoE-ACCESS-PROFILE accounting update-interval 10

# Interim accounting updates every 10 minutes
# accounting-stop-on-failure sends Stop on auth failure
# immediate-update sends update when service is activated
```

### RADIUS Change of Authorization (CoA)

```
# Enable CoA listener
set access radius-server 10.0.0.100 dynamic-request-port 3799

# CoA operations:
#   Disconnect-Request   -- terminate subscriber session
#   CoA-Request          -- modify subscriber attributes (QoS, filter, etc.)

# CoA can push:
#   - New QoS profile (Juniper-Cos-Parameter)
#   - New firewall filter (Juniper-Switching-Filter)
#   - Service activation/deactivation
#   - Session-Timeout change
```

## Dynamic Profiles

### Dynamic Profile Variables

```
Junos predefined variables (resolved at subscriber login):

$junos-interface-unit              Dynamically assigned unit number
$junos-underlying-interface        Physical/VLAN interface beneath the subscriber
$junos-subscriber-ip-address       IP assigned to the subscriber
$junos-subscriber-netmask          Netmask for the subscriber IP
$junos-subscriber-ipv6-address     IPv6 address
$junos-subscriber-ipv6-prefix      Delegated prefix (DHCPv6-PD)
$junos-cos-scheduler-map           Scheduler map from RADIUS
$junos-cos-shaping-rate            Shaping rate from RADIUS
$junos-routing-instance            Routing instance name
$junos-input-filter                Input firewall filter
$junos-output-filter               Output firewall filter
```

### Dynamic Profile with QoS and Firewall Filter

```
set dynamic-profiles SUB-FULL interfaces demux0 unit "$junos-interface-unit"
set dynamic-profiles SUB-FULL interfaces demux0 unit "$junos-interface-unit" demux-options underlying-interface "$junos-underlying-interface"
set dynamic-profiles SUB-FULL interfaces demux0 unit "$junos-interface-unit" family inet unnumbered-address lo0.0
set dynamic-profiles SUB-FULL interfaces demux0 unit "$junos-interface-unit" family inet address "$junos-subscriber-ip-address/$junos-subscriber-netmask"
set dynamic-profiles SUB-FULL interfaces demux0 unit "$junos-interface-unit" family inet filter input "$junos-input-filter"
set dynamic-profiles SUB-FULL interfaces demux0 unit "$junos-interface-unit" family inet filter output "$junos-output-filter"

set dynamic-profiles SUB-FULL class-of-service traffic-control-profiles "$junos-cos-traffic-control-profile"
set dynamic-profiles SUB-FULL class-of-service interfaces demux0 unit "$junos-interface-unit" output-traffic-control-profile "$junos-cos-traffic-control-profile"
```

### Service Profiles (Stacked Dynamic Profiles)

```
# Base profile for subscriber session
set dynamic-profiles BASE-SUBSCRIBER interfaces demux0 unit "$junos-interface-unit"
set dynamic-profiles BASE-SUBSCRIBER interfaces demux0 unit "$junos-interface-unit" family inet unnumbered-address lo0.0

# Service profile layered on top (activated via RADIUS CoA)
set dynamic-profiles SERVICE-TURBO class-of-service traffic-control-profiles TURBO-TCP
set dynamic-profiles SERVICE-TURBO class-of-service interfaces demux0 unit "$junos-interface-unit" output-traffic-control-profile TURBO-TCP
set dynamic-profiles SERVICE-TURBO firewall family inet filter TURBO-FILTER term ALLOW then accept

# Activate via RADIUS:
#   Juniper-Service-Activate = "SERVICE-TURBO"
# Deactivate via CoA:
#   Juniper-Service-Deactivate = "SERVICE-TURBO"
```

## Address Assignment

### Local Address Pool

```
set access address-assignment pool POOL-100M family inet network 10.200.0.0/16
set access address-assignment pool POOL-100M family inet range R1 low 10.200.0.2
set access address-assignment pool POOL-100M family inet range R1 high 10.200.127.254
set access address-assignment pool POOL-100M family inet dhcp-attributes router 10.200.0.1
set access address-assignment pool POOL-100M family inet dhcp-attributes name-server 8.8.8.8

# Named pool referenced in RADIUS via Framed-Pool = "POOL-100M"
```

### RADIUS Framed-IP-Address

```
# RADIUS returns Framed-IP-Address for static assignment:
#   Framed-IP-Address = 10.200.1.50
#   Framed-IP-Netmask = 255.255.255.255

# Or RADIUS returns pool name:
#   Framed-Pool = "POOL-100M"

# Junos resolves the pool locally and assigns from it.
```

### IPv6 Address Assignment

```
# SLAAC (RA-based)
set dynamic-profiles DUAL-STACK interfaces demux0 unit "$junos-interface-unit" family inet6 address "$junos-subscriber-ipv6-address"

# DHCPv6 Prefix Delegation
set access address-assignment pool PD-POOL family inet6 prefix 2001:db8::/32
set access address-assignment pool PD-POOL family inet6 range PD-RANGE prefix-length 56

# DHCPv6 IA-NA (individual address)
set access address-assignment pool NA-POOL family inet6 prefix 2001:db8:a000::/48
set access address-assignment pool NA-POOL family inet6 range NA-RANGE prefix-length 128
```

## Per-Subscriber QoS

### Traffic Control Profiles

```
# Define shaping rate per subscriber tier
set class-of-service traffic-control-profiles TCP-50M shaping-rate 50m
set class-of-service traffic-control-profiles TCP-50M scheduler-map SUB-SCHED-MAP
set class-of-service traffic-control-profiles TCP-50M guaranteed-rate 10m

set class-of-service traffic-control-profiles TCP-100M shaping-rate 100m
set class-of-service traffic-control-profiles TCP-100M scheduler-map SUB-SCHED-MAP
set class-of-service traffic-control-profiles TCP-100M guaranteed-rate 20m

set class-of-service traffic-control-profiles TCP-1G shaping-rate 1g
set class-of-service traffic-control-profiles TCP-1G scheduler-map SUB-SCHED-MAP
set class-of-service traffic-control-profiles TCP-1G guaranteed-rate 100m
```

### Scheduler Maps for Subscribers

```
# Define schedulers
set class-of-service schedulers SCHED-BE transmit-rate remainder
set class-of-service schedulers SCHED-BE buffer-size remainder
set class-of-service schedulers SCHED-BE priority low

set class-of-service schedulers SCHED-EF transmit-rate percent 30
set class-of-service schedulers SCHED-EF buffer-size percent 10
set class-of-service schedulers SCHED-EF priority strict-high

set class-of-service schedulers SCHED-AF transmit-rate percent 20
set class-of-service schedulers SCHED-AF buffer-size percent 20
set class-of-service schedulers SCHED-AF priority medium-high

set class-of-service schedulers SCHED-NC transmit-rate percent 5
set class-of-service schedulers SCHED-NC buffer-size percent 5
set class-of-service schedulers SCHED-NC priority medium-high

# Build the scheduler map
set class-of-service scheduler-maps SUB-SCHED-MAP forwarding-class best-effort scheduler SCHED-BE
set class-of-service scheduler-maps SUB-SCHED-MAP forwarding-class expedited-forwarding scheduler SCHED-EF
set class-of-service scheduler-maps SUB-SCHED-MAP forwarding-class assured-forwarding scheduler SCHED-AF
set class-of-service scheduler-maps SUB-SCHED-MAP forwarding-class network-control scheduler SCHED-NC
```

### Hierarchical CoS (H-CoS)

```
# H-CoS enables per-subscriber queuing with aggregate port shaping

# Level 1: port-level shaping
set class-of-service traffic-control-profiles PORT-LEVEL shaping-rate 10g

# Level 2: per-subscriber shaping
set class-of-service traffic-control-profiles SUB-LEVEL shaping-rate 100m
set class-of-service traffic-control-profiles SUB-LEVEL scheduler-map SUB-SCHED-MAP

# Bind to interface hierarchy
set class-of-service interfaces ge-1/0/0 output-traffic-control-profile PORT-LEVEL
set dynamic-profiles SUB-FULL class-of-service interfaces demux0 unit "$junos-interface-unit" output-traffic-control-profile SUB-LEVEL
```

### RADIUS-Driven QoS Activation

```
# RADIUS returns VSA to select traffic-control-profile:
#   Juniper-Cos-Traffic-Control-Profile = "TCP-100M"

# Or via CoA for mid-session upgrade/downgrade:
#   CoA-Request with Juniper-Cos-Traffic-Control-Profile = "TCP-1G"

# Dynamic profile references the variable:
set dynamic-profiles SUB-FULL class-of-service interfaces demux0 unit "$junos-interface-unit" output-traffic-control-profile "$junos-cos-traffic-control-profile"
```

## ANCP (Access Node Control Protocol)

```
# ANCP enables BNG to learn DSL line rates from DSLAM
set protocols ancp neighbor 10.50.0.1 auto-configure
set protocols ancp neighbor 10.50.0.1 tcp-port 6068

# ANCP-learned rate is used for downstream shaping:
#   - Actual sync rate reported by DSLAM
#   - BNG adjusts traffic-control-profile shaping-rate dynamically

# Verification
show ancp neighbor
show ancp subscriber
show ancp statistics
```

## Subscriber Redundancy

### Subscriber Replication (Stateful Failover)

```
# On both primary and backup BNG:
set system services subscriber-management enable

# Configure redundancy
set unified-edge gateways gateway BNG-GW system-id BNG-PRIMARY
set unified-edge gateways gateway BNG-GW redundancy-options peer 10.0.0.2

# Subscriber state replication ensures session persistence during failover
# Replicated state includes:
#   - Session state (PPPoE, IPoE)
#   - IP address assignments
#   - CoS profiles
#   - Firewall filters
#   - Accounting state
```

### ISSU (In-Service Software Upgrade)

```
# ISSU allows hitless Junos upgrade on dual-RE MX chassis
# Subscribers remain active during RE switchover

# Prerequisites:
#   - Dual RE (Routing Engine)
#   - NSR (Nonstop Active Routing) enabled
#   - GRES (Graceful Routing Engine Switchover) enabled

set chassis redundancy graceful-switchover
set routing-options nonstop-routing

# Perform ISSU
request system software in-service-upgrade /var/tmp/junos-install-mx-x86-64-XX.X.tgz
```

## CLI Verification Commands

### Subscriber Commands

```
# All active subscribers
show subscribers

# Summary by type
show subscribers summary

# Detailed subscriber info
show subscribers user-name "user@example.com" detail
show subscribers interface pp0.1073741824 detail

# Filter by type
show subscribers protocol pppoe
show subscribers protocol dhcp

# Filter by port/VLAN
show subscribers port ge-1/0/0
show subscribers vlan-id 100

# Subscriber accounting
show subscribers accounting-statistics user-name "user@example.com"

# Subscriber count by interface
show subscribers summary port ge-1/0/0
```

### PPPoE Commands

```
show pppoe interfaces
show pppoe interfaces pp0.1073741824 detail
show pppoe lockout
show pppoe statistics
show pppoe version
show ppp interface pp0.1073741824 extensive
```

### DHCP Commands

```
show dhcp server binding
show dhcp server binding detail
show dhcp server statistics
show dhcp relay binding
show dhcp relay statistics
show dhcpv6 server binding
show system services dhcp-local-server binding
```

### RADIUS Commands

```
show network-access aaa statistics radius
show network-access aaa statistics authentication
show network-access aaa statistics accounting
show network-access requests pending
show network-access radius-servers
```

### QoS / CoS Commands

```
show class-of-service interface demux0.1073741824
show class-of-service traffic-control-profile TCP-100M
show class-of-service scheduler-map SUB-SCHED-MAP
show subscribers user-name "user@example.com" detail | find cos
```

### Address Pool Commands

```
show network-access address-assignment pool POOL-100M
show network-access address-assignment pool POOL-100M usage
show network-access address-assignment address-pool POOL-100M statistics
```

### Troubleshooting

```
# Trace PPPoE negotiation
set protocols pppoe traceoptions file pppoe-trace
set protocols pppoe traceoptions flag all

# Trace RADIUS
set access radius-server 10.0.0.100 traceoptions file radius-trace
set access radius-server 10.0.0.100 traceoptions flag all

# Trace DHCP
set system services dhcp-local-server traceoptions file dhcp-trace
set system services dhcp-local-server traceoptions flag all

# Trace dynamic profiles
set dynamic-profiles traceoptions file dynprof-trace
set dynamic-profiles traceoptions flag all

# Check for subscriber session errors
show subscribers error

# Clear a stuck subscriber session
clear subscribers interface pp0.1073741824
clear subscribers user-name "user@example.com"
```

## Tips

- Always test dynamic profiles with `show dynamic-profiles` before committing to production.
- Use `commit confirmed 5` when making BNG changes -- an accidental lockout reverts in 5 minutes.
- PPPoE sessions use the `pp0` interface; IPoE sessions use `demux0` -- never mix them.
- RADIUS Framed-Pool must exactly match the local pool name (case-sensitive).
- When debugging PPPoE, check `show pppoe statistics` for PADI/PADO/PADR/PADS/PADT counters.
- H-CoS requires MPC line cards with per-subscriber queuing support (MPC7E or later).
- ANCP shaping rates override static traffic-control-profile rates when configured.
- For dual-stack, the subscriber needs both an IPoE/PPPoE IPv4 session and a DHCPv6 session for prefix delegation.
- Subscriber interface unit numbers are dynamically assigned -- do not hardcode them.
- CoA requires the BNG to listen on port 3799 and the RADIUS server to know the BNG IP.

## See Also

- JunOS Interfaces
- JunOS Routing Fundamentals
- JunOS Firewall Filters
- JunOS Class of Service (CoS)
- RADIUS and AAA
- PPPoE Protocol
- DHCP
- ISP Edge Architecture

## References

- Juniper TechLibrary: Subscriber Management and Services
- Juniper TechLibrary: Broadband Subscriber Sessions
- Juniper TechLibrary: Dynamic Profiles
- Juniper TechLibrary: DHCP Local Server and Relay
- Juniper TechLibrary: Class of Service for Subscriber Interfaces
- Juniper TechLibrary: ANCP Configuration
- RFC 2516 -- A Method for Transmitting PPP Over Ethernet (PPPoE)
- RFC 2865 -- Remote Authentication Dial In User Service (RADIUS)
- RFC 2866 -- RADIUS Accounting
- RFC 5176 -- Dynamic Authorization Extensions to RADIUS (CoA/DM)
- RFC 6320 -- Protocol for Access Node Control Mechanism in Broadband Networks (ANCP)
