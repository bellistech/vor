# JunOS High Availability

HA mechanisms for Juniper platforms — redundant routing engines, nonstop forwarding, hitless upgrades, fast failure detection, and multi-chassis resilience. Critical for SP networks requiring five-nines uptime.

## Dual Routing Engine Configuration

### Basic dual RE setup
```
# Enable commit synchronize — keeps both REs in sync
set system commit synchronize

# Set RE0 as primary (default mastership priority)
set groups re0 system host-name ROUTER-RE0
set groups re0 interfaces fxp0 unit 0 family inet address 10.0.0.1/24

set groups re1 system host-name ROUTER-RE1
set groups re1 interfaces fxp0 unit 0 family inet address 10.0.0.2/24

set apply-groups re0
set apply-groups re1

# Mastership priority (higher = more likely to be master)
set chassis redundancy routing-engine 0 master
set chassis redundancy routing-engine 1 backup
```

### Verify dual RE status
```
show chassis routing-engine               # both RE status, uptime, memory
show chassis routing-engine 0             # specific RE
show chassis redundancy                   # mastership state
show system commit                        # last commit sync status
```

### Manual RE switchover
```
request chassis routing-engine master switch   # switch mastership to backup RE
request chassis routing-engine master acquire  # force current RE to become master
```

## Graceful Routing Engine Switchover (GRES)

### What GRES does
```
# GRES preserves:
#   - Kernel state (interfaces, forwarding table)
#   - PFE forwarding state (traffic continues during switchover)
#
# GRES does NOT preserve:
#   - Routing protocol state (BGP/OSPF/IS-IS sessions reset)
#   - Protocol daemon state (rpd restarts on new master)
#
# Result: forwarding continues during RE switchover, but routing protocols
#         re-establish sessions (graceful restart helpers assist)
```

### Enable GRES
```
set chassis redundancy graceful-switchover
set routing-options nonstop-forwarding    # companion to GRES — keep forwarding during rpd restart

# GRES requires commit synchronize
set system commit synchronize
```

### Verify GRES
```
show system switchover                    # GRES readiness
show chassis redundancy                   # shows "Graceful switchover: Configured"
```

## Nonstop Active Routing (NSR)

### What NSR does
```
# NSR preserves:
#   - All GRES benefits (kernel + PFE state)
#   - Routing protocol state (BGP, IS-IS, OSPF sessions maintained)
#   - Protocol daemon on backup RE mirrors master RE's rpd state
#
# Result: RE switchover is invisible to routing peers — no session flaps
#
# NSR is a superset of GRES
```

### Enable NSR
```
set routing-options nonstop-routing

# NSR implies GRES — also enable:
set chassis redundancy graceful-switchover
set system commit synchronize
```

### Verify NSR
```
show task replication                     # protocol replication status
show route summary                        # verify backup RE has full RIB
show bgp neighbor | match "NSR"           # NSR state per BGP peer
show isis adjacency detail | match "NSR"  # NSR state for IS-IS
```

## Nonstop Bridging (NSB)

### Enable NSB for Layer 2
```
# Preserves L2 state (MAC table, STP state) during RE switchover
set protocols layer2-control nonstop-bridging
```

## Graceful Restart

### Configure graceful restart (for use with GRES)
```
# Global graceful restart
set routing-options graceful-restart

# Per-protocol graceful restart
set protocols bgp graceful-restart
set protocols ospf graceful-restart
set protocols isis graceful-restart

# Restart time (how long neighbors wait before purging routes)
set routing-options graceful-restart restart-duration 300
```

### Verify graceful restart
```
show bgp neighbor | match "Restart"       # GR capability per peer
show ospf overview | match "Restart"      # OSPF GR state
```

## Unified ISSU (In-Service Software Upgrade)

### Prerequisites for unified ISSU
```
# Requirements:
#   - Dual RE chassis
#   - GRES enabled and operational
#   - NSR enabled (recommended)
#   - Commit synchronize active
#   - Both REs running and synchronized
#   - Compatible upgrade path (check release notes)
```

### Perform unified ISSU
```
# 1. Validate readiness
show system switchover                    # must show "Ready"
show chassis redundancy                   # both REs present
show task replication                     # NSR replication complete

# 2. Copy software to both REs
request system software add /var/tmp/junos-install-mx-x86-64-XX.X.tgz
request system software add /var/tmp/junos-install-mx-x86-64-XX.X.tgz re1

# 3. Start ISSU
request system software in-service-upgrade /var/tmp/junos-install-mx-x86-64-XX.X.tgz

# 4. Monitor progress
show system software in-service-upgrade   # ISSU state machine status
```

### ISSU process
```
# Step 1: Backup RE upgraded and rebooted with new software
# Step 2: Backup RE synchronizes state from master
# Step 3: Mastership switches to backup (now running new software)
# Step 4: Old master RE upgraded and rebooted
# Step 5: Old master comes up as backup with new software
# Step 6: Both REs running new software
#
# Traffic forwarding maintained throughout (hitless upgrade)
```

## BFD (Bidirectional Forwarding Detection)

### BFD for routing protocols
```
# BFD on OSPF
set protocols ospf area 0.0.0.0 interface ge-0/0/0 bfd-liveness-detection minimum-interval 300
set protocols ospf area 0.0.0.0 interface ge-0/0/0 bfd-liveness-detection multiplier 3
# Detects failure in: 300ms * 3 = 900ms

# BFD on IS-IS
set protocols isis interface ge-0/0/0 bfd-liveness-detection minimum-interval 300
set protocols isis interface ge-0/0/0 bfd-liveness-detection multiplier 3

# BFD on BGP
set protocols bgp group EBGP neighbor 10.0.0.2 bfd-liveness-detection minimum-interval 1000
set protocols bgp group EBGP neighbor 10.0.0.2 bfd-liveness-detection multiplier 3

# BFD on static routes
set routing-options static route 0.0.0.0/0 next-hop 10.0.0.1 bfd-liveness-detection minimum-interval 1000
set routing-options static route 0.0.0.0/0 next-hop 10.0.0.1 bfd-liveness-detection multiplier 3
```

### BFD timers
```
# minimum-interval: minimum transmit AND receive interval (milliseconds)
# minimum-receive-interval: override receive interval separately
# transmit-interval minimum-interval: override transmit interval
# multiplier: number of missed packets before declaring down
# detection time = minimum-interval * multiplier

# Aggressive BFD (sub-second detection):
set protocols ospf area 0 interface ge-0/0/0 bfd-liveness-detection minimum-interval 50
set protocols ospf area 0 interface ge-0/0/0 bfd-liveness-detection multiplier 3
# Detection time: 50ms * 3 = 150ms
```

### Multihop BFD
```
# BFD for multihop BGP sessions (eBGP multihop / loopback peering)
set protocols bgp group IBGP neighbor 10.255.0.1 bfd-liveness-detection minimum-interval 1000
set protocols bgp group IBGP neighbor 10.255.0.1 bfd-liveness-detection multiplier 3
set protocols bgp group IBGP neighbor 10.255.0.1 multihop
```

### Micro-BFD for LAG
```
# BFD per member link of an aggregate interface
set interfaces ae0 aggregated-ether-options bfd-liveness-detection minimum-interval 300
set interfaces ae0 aggregated-ether-options bfd-liveness-detection multiplier 3
set interfaces ae0 aggregated-ether-options bfd-liveness-detection neighbor 10.0.0.2

# If micro-BFD detects member link failure, link removed from bundle
# Faster than LACP timeout (LACP fast = 3 seconds, micro-BFD can be < 1 second)
```

### Verify BFD
```
show bfd session                          # all BFD sessions
show bfd session detail                   # detailed timers, state
show bfd session extensive                # full session info including errors
show bfd session neighbor 10.0.0.2        # specific neighbor
```

## VRRP on JunOS

### Basic VRRP configuration
```
set interfaces ge-0/0/0 unit 0 family inet address 10.0.0.2/24 vrrp-group 1 virtual-address 10.0.0.1
set interfaces ge-0/0/0 unit 0 family inet address 10.0.0.2/24 vrrp-group 1 priority 200
set interfaces ge-0/0/0 unit 0 family inet address 10.0.0.2/24 vrrp-group 1 preempt
set interfaces ge-0/0/0 unit 0 family inet address 10.0.0.2/24 vrrp-group 1 accept-data

# Backup router
set interfaces ge-0/0/0 unit 0 family inet address 10.0.0.3/24 vrrp-group 1 virtual-address 10.0.0.1
set interfaces ge-0/0/0 unit 0 family inet address 10.0.0.3/24 vrrp-group 1 priority 100
set interfaces ge-0/0/0 unit 0 family inet address 10.0.0.3/24 vrrp-group 1 preempt
set interfaces ge-0/0/0 unit 0 family inet address 10.0.0.3/24 vrrp-group 1 accept-data
```

### VRRP tracking
```
# Track interface — lower priority if tracked interface goes down
set interfaces ge-0/0/0 unit 0 family inet address 10.0.0.2/24 vrrp-group 1 track interface ge-0/0/1 priority-cost 120
# If ge-0/0/1 goes down, priority drops from 200 to 80 (below backup's 100)

# Track route
set interfaces ge-0/0/0 unit 0 family inet address 10.0.0.2/24 vrrp-group 1 track route 0.0.0.0/0 routing-instance default priority-cost 120
```

### VRRPv3 (IPv6 support)
```
set interfaces ge-0/0/0 unit 0 family inet6 address 2001:db8::2/64 vrrp-inet6-group 1 virtual-inet6-address 2001:db8::1
set interfaces ge-0/0/0 unit 0 family inet6 address 2001:db8::2/64 vrrp-inet6-group 1 priority 200
set interfaces ge-0/0/0 unit 0 family inet6 address 2001:db8::2/64 vrrp-inet6-group 1 preempt
```

### Verify VRRP
```
show vrrp                                 # all VRRP groups
show vrrp detail                          # detailed state, timers
show vrrp track                           # tracked objects
```

## MC-LAG (Multi-Chassis LAG)

### MC-LAG topology
```
# Two JunOS devices present a single LAG to a downstream device
#
# Device-A ──── ae0 ────┐
#                        ├──── ae0 ──── Downstream
# Device-B ──── ae0 ────┘
#
# ICCP (Inter-Chassis Control Protocol) synchronizes state between A and B
# ICL (Inter-Chassis Link) carries traffic between chassis
```

### MC-LAG configuration (Device A)
```
# 1. ICCP connection to peer
set protocols iccp local-ip-addr 10.255.0.1
set protocols iccp peer 10.255.0.2 redundancy-group-id-list 1
set protocols iccp peer 10.255.0.2 liveness-detection minimum-interval 1000
set protocols iccp peer 10.255.0.2 liveness-detection multiplier 3

# 2. Multi-chassis configuration
set multi-chassis multi-chassis-protection ge-0/0/9 interface ge-0/0/8

# 3. MC-LAG redundancy group
set interfaces ae0 aggregated-ether-options lacp active
set interfaces ae0 aggregated-ether-options lacp system-id 00:00:00:00:00:01
set interfaces ae0 aggregated-ether-options lacp admin-key 1
set interfaces ae0 aggregated-ether-options mc-ae mc-ae-id 1
set interfaces ae0 aggregated-ether-options mc-ae redundancy-group 1
set interfaces ae0 aggregated-ether-options mc-ae chassis-id 0
set interfaces ae0 aggregated-ether-options mc-ae mode active-active
set interfaces ae0 aggregated-ether-options mc-ae status-control active
set interfaces ae0 aggregated-ether-options mc-ae init-delay-time 60

# 4. Member links
set interfaces ge-0/0/0 gigether-options 802.3ad ae0
set interfaces ge-0/0/1 gigether-options 802.3ad ae0
```

### MC-LAG configuration (Device B)
```
set protocols iccp local-ip-addr 10.255.0.2
set protocols iccp peer 10.255.0.1 redundancy-group-id-list 1

set interfaces ae0 aggregated-ether-options lacp active
set interfaces ae0 aggregated-ether-options lacp system-id 00:00:00:00:00:01
set interfaces ae0 aggregated-ether-options lacp admin-key 1
set interfaces ae0 aggregated-ether-options mc-ae mc-ae-id 1
set interfaces ae0 aggregated-ether-options mc-ae redundancy-group 1
set interfaces ae0 aggregated-ether-options mc-ae chassis-id 1
set interfaces ae0 aggregated-ether-options mc-ae mode active-active
set interfaces ae0 aggregated-ether-options mc-ae status-control standby
```

### Verify MC-LAG
```
show interfaces mc-ae                     # MC-LAG status
show interfaces mc-ae id 1               # specific MC-LAG group
show iccp                                 # ICCP protocol state
show lacp interfaces ae0                  # LACP state
show multi-chassis                        # multi-chassis protection state
```

## Virtual Chassis

### Virtual Chassis configuration
```
# Convert standalone devices into a single logical device
# Typically for EX/QFX switches

# On primary device
set virtual-chassis preprovisioned
set virtual-chassis member 0 role routing-engine serial-number ABC123
set virtual-chassis member 1 role routing-engine serial-number DEF456
set virtual-chassis member 2 role line-card serial-number GHI789

# VC interconnects (VCPs)
set interfaces vcp-0 disable              # auto-detected ports
request virtual-chassis vc-port set pic-slot 1 port 0
```

### Verify Virtual Chassis
```
show virtual-chassis                      # member status, roles
show virtual-chassis status               # detailed VC state
show virtual-chassis vc-port              # VC port interconnects
```

## LACP

### Basic LACP configuration
```
# Aggregate ethernet interface
set chassis aggregated-devices ethernet device-count 5

set interfaces ae0 aggregated-ether-options lacp active
set interfaces ae0 unit 0 family inet address 10.0.0.1/24

# Add member links
set interfaces ge-0/0/0 gigether-options 802.3ad ae0
set interfaces ge-0/0/1 gigether-options 802.3ad ae0
set interfaces ge-0/0/2 gigether-options 802.3ad ae0
```

### LACP options
```
set interfaces ae0 aggregated-ether-options lacp active            # active LACP negotiation
set interfaces ae0 aggregated-ether-options lacp passive           # respond only
set interfaces ae0 aggregated-ether-options lacp periodic fast     # 1-second LACP PDUs (default slow = 30s)
set interfaces ae0 aggregated-ether-options lacp system-priority 100
set interfaces ae0 aggregated-ether-options lacp system-id 00:00:00:00:00:01  # override for MC-LAG
set interfaces ae0 aggregated-ether-options minimum-links 2        # min links for bundle up
```

### Verify LACP
```
show lacp interfaces                      # LACP state per interface
show lacp interfaces ae0 detail           # detailed LACP info
show lacp statistics interfaces ae0       # LACP PDU counters
show interfaces ae0 detail                # aggregate interface state
show interfaces ae0 extensive             # full counters and stats
```

## Commit Synchronize

### Configuration
```
set system commit synchronize             # auto-sync commits to backup RE

# Manual sync
request system configuration rescue save  # save rescue config on both REs
```

### Verify synchronization
```
show system commit                        # commit history
show system configuration rescue          # rescue config status
```

## Verification Commands Summary

### HA state
```
show chassis routing-engine               # RE status (master/backup)
show chassis redundancy                   # redundancy config and state
show system switchover                    # GRES/ISSU readiness
show task replication                     # NSR protocol replication
show bfd session                          # BFD sessions
show bfd session detail                   # BFD timers and stats
```

### Interfaces and LAG
```
show lacp interfaces                      # LACP state
show interfaces ae0                       # aggregate interface
show interfaces mc-ae                     # MC-LAG state
show iccp                                 # ICCP protocol state
```

### Protocols
```
show vrrp detail                          # VRRP state
show bgp neighbor | match "Restart|NSR"   # BGP HA state
show ospf overview | match "Restart"      # OSPF GR state
show virtual-chassis                      # VC member status
```

## Tips

- Always enable `commit synchronize` with dual RE — a desynchronized backup RE is useless during switchover
- NSR is preferred over GRES+GR because routing sessions are not disrupted at all
- GRES alone requires routing peers to support graceful restart helper mode
- BFD minimum-interval below 50ms may require hardware-assisted BFD (platform dependent)
- Micro-BFD is essential for LAG — LACP fast still takes 3 seconds to detect member failure
- VRRP `accept-data` is required for the master to respond to pings on the virtual IP
- MC-LAG LACP system-id must match on both chassis — the downstream device sees one LAG partner
- Always validate ISSU compatibility in release notes before attempting in-service upgrade
- Use `request system snapshot` before any ISSU as a rollback safety net
- BFD sessions consume memory and CPU — scale testing before deploying sub-100ms timers network-wide

## See Also

- junos-interfaces, junos-routing-fundamentals, junos-architecture, bfd, vrrp, lacp

## References

- [Juniper TechLibrary — GRES Overview](https://www.juniper.net/documentation/us/en/software/junos/high-availability/topics/concept/gres-overview.html)
- [Juniper TechLibrary — NSR Overview](https://www.juniper.net/documentation/us/en/software/junos/high-availability/topics/concept/nsr-overview.html)
- [Juniper TechLibrary — Unified ISSU](https://www.juniper.net/documentation/us/en/software/junos/high-availability/topics/concept/issu-overview.html)
- [Juniper TechLibrary — BFD](https://www.juniper.net/documentation/us/en/software/junos/high-availability/topics/concept/bfd-overview.html)
- [Juniper TechLibrary — MC-LAG](https://www.juniper.net/documentation/us/en/software/junos/mc-lag/topics/concept/mc-lag-overview.html)
- [Juniper TechLibrary — VRRP](https://www.juniper.net/documentation/us/en/software/junos/high-availability/topics/concept/vrrp-overview.html)
- [RFC 5880 — Bidirectional Forwarding Detection](https://www.rfc-editor.org/rfc/rfc5880)
- [RFC 5881 — BFD for IPv4 and IPv6](https://www.rfc-editor.org/rfc/rfc5881)
- [RFC 5882 — BFD Generic Application](https://www.rfc-editor.org/rfc/rfc5882)
- [RFC 7275 — ICCP for MC-LAG](https://www.rfc-editor.org/rfc/rfc7275)
- [RFC 5798 — VRRPv3](https://www.rfc-editor.org/rfc/rfc5798)
