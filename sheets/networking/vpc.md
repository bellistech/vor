# Cisco vPC (Virtual Port Channel)

Multi-chassis EtherChannel providing loop-free topologies and active-active uplinks across a pair of Nexus switches.

## Domain and Peer Setup

### Configure vPC domain

```
! Both switches must share the same domain ID
vpc domain 100
  peer-keepalive destination 10.1.1.2 source 10.1.1.1 vrf keepalive
  role priority 1000          ! lower = primary (default 32667)
  system-priority 2000        ! lower wins (default 32667)
  peer-gateway                ! route for peer's HSRP MAC
  ip arp synchronize
  delay restore 60            ! wait before bringing up vPC after reload
  delay restore interface-vlan 45
  auto-recovery
  auto-recovery reload-delay 240
```

### Peer-keepalive link (heartbeat)

```
! Dedicated mgmt VRF recommended — NOT over peer-link
vrf context keepalive
interface mgmt0
  vrf member keepalive
  ip address 10.1.1.1/30
  no shutdown

vpc domain 100
  peer-keepalive destination 10.1.1.2 source 10.1.1.1 vrf keepalive
```

### Peer-link (data + control)

```
! Minimum 2x10G in a port-channel, dedicated interfaces recommended
interface port-channel 1
  switchport mode trunk
  switchport trunk allowed vlan 1-4094
  spanning-tree port type network
  vpc peer-link
```

## vPC Member Port-Channels

### Basic member port-channel to a downstream switch

```
! Must match on both vPC peers — same vpc number
interface port-channel 10
  switchport mode trunk
  switchport trunk allowed vlan 100-200
  vpc 10

interface ethernet 1/1
  channel-group 10 mode active
```

### vPC to a server (straight-through)

```
interface port-channel 20
  switchport mode access
  switchport access vlan 100
  vpc 20

interface ethernet 1/5
  channel-group 20 mode active
```

### Verify member port-channels

```bash
show vpc                          # overall vPC status and role
show vpc brief                    # compact summary of all vPCs
show vpc consistency-parameters   # check type-1/type-2 mismatches
show port-channel summary         # all port-channels and members
```

## Consistency Checks

### Type-1 (mandatory — suspends vPC on mismatch)

```
! These MUST match on both peers or the vPC goes down:
! - LACP mode (active/passive/on)
! - STP mode (rapid-pvst/mst)
! - STP port type (normal/network/edge)
! - MTU
! - Allowed VLANs on trunk
! - Speed/duplex of member links
! - Switchport mode (access/trunk)
! - Storm control settings

show vpc consistency-parameters global    # global params
show vpc consistency-parameters vpc 10    # per-vPC params
```

### Type-2 (advisory — logged but vPC stays up)

```
! These SHOULD match but won't suspend:
! - STP cost/priority
! - BPDU filter/guard
! - DHCP snooping
! - ARP inspection settings
! - IGMP snooping settings

show vpc consistency-parameters vlans   # VLAN-level checks
```

### Graceful consistency check

```
vpc domain 100
  graceful consistency-check   ! default: enabled
  ! Suspends secondary vPC leg only on type-1 mismatch
  ! Without this, BOTH legs suspend
```

## Failure Scenarios

### Scenario 1: Single member link failure

```
! One link in a vPC port-channel fails
! Traffic rehashes across remaining links in the port-channel
! If ALL links on one peer fail, traffic flows through peer-link
!   (peer-link acts as backup data path)

show interface port-channel 10      # check operational members
show vpc orphan-ports               # check for stranded hosts
```

### Scenario 2: Peer-link failure (keepalive UP)

```
! Secondary switch suspends ALL its vPC member ports
! Secondary keeps only non-vPC (orphan) SVIs and ports
! Primary continues forwarding normally
! Traffic from secondary-connected hosts reroutes via L3

! Recovery: restore peer-link, secondary resumes vPCs automatically
show vpc                            # check peer-link status
show vpc role                       # verify primary/secondary
```

### Scenario 3: Peer-keepalive failure (peer-link UP)

```
! No immediate impact — peer-link carries CFS heartbeats
! BUT the keepalive is the safety net: if peer-link ALSO fails,
!   both switches think they're primary = dual-active / split-brain
! Fix keepalive ASAP

show vpc peer-keepalive             # check keepalive state
```

### Scenario 4: Dual failure (peer-link + keepalive both down)

```
! SPLIT-BRAIN: both switches become primary
! Both forward traffic = duplicate packets, MAC flapping, loops
! auto-recovery helps (if configured): secondary disables vPCs
!   after a timer expires if it detects no keepalive

vpc domain 100
  auto-recovery                     ! enable auto-recovery
  auto-recovery reload-delay 240    ! seconds to wait after reload

show vpc role                       # both will show "primary" in split-brain
```

### Scenario 5: One vPC peer reloads

```
! Remaining peer continues forwarding on all vPCs
! When reloaded peer comes back:
!   1. Forms peer-keepalive
!   2. Forms peer-link
!   3. Syncs MAC/ARP tables via CFS
!   4. Waits delay-restore timer
!   5. Brings up vPC member ports

show vpc                            # watch for "peer adjacency formed"
```

## Orphan Ports

### What are orphan ports

```
! Orphan port = device single-homed to only ONE vPC peer
! Problem: if peer-link fails, secondary suspends vPCs but
!   orphan devices lose connectivity because their switch
!   can't reach the primary

! Solution: configure orphan port suspend on the secondary
!   so orphan ports also get suspended, forcing the device
!   to fail over (if dual-homed at L3)
```

### Configure orphan port suspend

```
vpc domain 100
  peer-gateway
  ! On the interface connected to the orphan device:

interface ethernet 1/10
  vpc orphan-ports suspend           ! suspend if peer-link fails
```

### Identify orphan ports

```bash
show vpc orphan-ports                 # list all orphan ports
show vpc                              # overall status
```

## Peer-Gateway

### Enable peer-gateway for HSRP/VRRP

```
! Problem: some devices (NetApp, certain servers) may route
!   packets destined to the HSRP virtual MAC to the wrong peer
! peer-gateway lets each peer accept and route traffic
!   destined to the other peer's router MAC address

vpc domain 100
  peer-gateway

! Also exclude peer-gateway traffic from peer-link to avoid loops:
interface port-channel 1
  ! (peer-link already configured)
  peer-gateway exclude-vlan <reserved-vlan>  ! optional, for specific exclusions
```

### ARP synchronization

```
vpc domain 100
  ip arp synchronize
  ! Syncs ARP table between peers via CFS over peer-link
  ! Ensures both peers can route traffic for hosts learned by the other
  ! Critical for peer-gateway to work properly

show ip arp synchronize vpc            # verify sync status
```

## Delay Restore Timers

### Configure delay restore

```
vpc domain 100
  delay restore 60                    ! vPC member ports (default 30s)
  delay restore interface-vlan 45     ! SVI interfaces (default 10s)
  delay restore orphan-port 20        ! orphan ports

! Purpose: after reload, wait for routing protocols to converge
!   before bringing up vPC ports — prevents blackholing traffic
!   while routes are still being learned
```

### Verify timers

```bash
show vpc                              # shows delay restore status
show vpc role                         # shows if in delay restore period
```

## vPC with HSRP/VRRP

### HSRP on vPC VLAN SVI

```
interface vlan 100
  ip address 10.100.0.2/24
  hsrp version 2
  hsrp 100
    ip 10.100.0.1
    priority 110                      ! primary has higher priority
    preempt
    authentication md5 key-string vpcHSRP

! On secondary peer:
interface vlan 100
  ip address 10.100.0.3/24
  hsrp version 2
  hsrp 100
    ip 10.100.0.1
    priority 100
    preempt
    authentication md5 key-string vpcHSRP
```

### Best practices for HSRP with vPC

```
! 1. Always enable peer-gateway
! 2. Always enable ip arp synchronize
! 3. Make vPC primary = HSRP active (align role priority)
! 4. Use preempt so roles stay consistent
! 5. Use delay restore interface-vlan to avoid premature SVI up
```

## vPC with FEX

### Dual-homed FEX (Active-Active to both peers)

```
! FEX connects to BOTH vPC peers via vPC port-channel
! Recommended topology for redundancy

! On primary peer:
fex 101
  pinning max-links 1
  description "Dual-homed FEX to Rack-1"

interface port-channel 101
  switchport mode fex-fabric
  fex associate 101
  vpc 101

interface ethernet 1/33
  channel-group 101

! On secondary peer:
fex 101
  pinning max-links 1
  description "Dual-homed FEX to Rack-1"

interface port-channel 101
  switchport mode fex-fabric
  fex associate 101
  vpc 101

interface ethernet 1/33
  channel-group 101
```

### Single-homed FEX (connected to one peer only)

```
! FEX connects to ONLY ONE vPC peer
! If that peer fails, FEX and all its hosts go down
! Use only when dual-homing isn't possible

! On one peer only:
fex 102
  pinning max-links 1
  description "Single-homed FEX — Rack-2"

interface port-channel 102
  switchport mode fex-fabric
  fex associate 102
  ! NO vpc command — this is NOT a vPC

interface ethernet 1/34
  channel-group 102
```

## vPC Peer-Switch (STP Optimization)

### Enable peer-switch

```
vpc domain 100
  peer-switch

! Both peers present the same bridge ID to STP
! Eliminates STP convergence when one peer reloads
! STP sees the vPC domain as a single logical switch

! Requirements:
!   - STP mode must be rapid-pvst or MST
!   - Both peers must have peer-switch enabled
!   - Both peers should be STP root for their VLANs

spanning-tree vlan 1-4094 root primary   ! on both peers
```

### Verify peer-switch

```bash
show spanning-tree summary              # bridge IDs should match
show vpc peer-switch                    # peer-switch operational status
show vpc role                           # confirms peer-switch active
```

## vPC Auto-Recovery

### Configure auto-recovery

```
vpc domain 100
  auto-recovery
  auto-recovery reload-delay 240        ! default 240 seconds

! Scenario: both peers reload simultaneously (power event)
! Without auto-recovery:
!   - Both come up, neither has keepalive/peer-link initially
!   - Both may stay operationally down waiting for the peer
! With auto-recovery:
!   - After reload-delay expires, one peer assumes primary
!   - Brings up vPCs unilaterally so traffic can flow
```

### Verify

```bash
show vpc                                # auto-recovery status
show vpc role                           # check if auto-recovery triggered
```

## CFS (Cisco Fabric Services)

### What CFS does in vPC

```
! CFS runs over the peer-link (NOT keepalive link)
! Synchronizes:
!   - MAC address tables
!   - IGMP snooping state
!   - ARP tables (if ip arp synchronize enabled)
!   - HSRP/VRRP state
!   - STP BPDU information
!   - ACL and QoS policies (Type-2)
!   - DHCP snooping bindings

show cfs status                         # CFS operational state
show cfs peers                          # peer switch info
show cfs application                    # what apps use CFS
```

## Show Commands and Troubleshooting

### Essential show commands

```bash
show vpc                                # master status command
show vpc brief                          # compact summary
show vpc role                           # primary/secondary, priority
show vpc peer-keepalive                 # keepalive link status
show vpc consistency-parameters global  # global type-1/type-2 checks
show vpc consistency-parameters vpc 10  # per-vPC consistency
show vpc statistics                     # counters, peer-link utilization
show vpc orphan-ports                   # single-homed devices
show vpc peer-switch                    # peer-switch status
```

### Port-channel diagnostics

```bash
show port-channel summary               # all port-channels with status
show port-channel database              # detailed port-channel info
show lacp counters                      # LACP PDU statistics
show lacp neighbor                      # far-end LACP info
show interface port-channel 10          # specific port-channel
```

### STP diagnostics in vPC

```bash
show spanning-tree vlan 100             # STP state for VLAN
show spanning-tree summary              # bridge IDs, root info
show spanning-tree inconsistentports    # ports in inconsistent state
```

### Common issues checklist

```
! 1. vPC not forming
!    - Check domain IDs match
!    - Check peer-keepalive reachability (ping)
!    - Check peer-link port-channel is up
!    - Check system-mac (show vpc role)

! 2. vPC member suspended
!    - show vpc consistency-parameters vpc <id>
!    - Fix type-1 mismatch (VLAN, MTU, mode, STP)
!    - Ensure LACP mode matches (active/active recommended)

! 3. Traffic blackholing after reload
!    - Increase delay restore timer
!    - Verify routing protocol convergence before vPC comes up
!    - Check ip arp synchronize is enabled

! 4. MAC flapping / duplicate frames
!    - Check for dual-active / split-brain
!    - Verify peer-link and keepalive are both up
!    - Check for L2 loops outside vPC domain

! 5. HSRP failover issues
!    - Enable peer-gateway
!    - Enable ip arp synchronize
!    - Align vPC primary with HSRP active
```

## Tips

- Always use `mode active` (LACP) for vPC member port-channels, never `mode on`
- Dedicate physical interfaces for the peer-link; never share with regular traffic
- Run the peer-keepalive over a separate VRF (mgmt or dedicated), not over the peer-link
- Align vPC role priority with HSRP/VRRP priority so primary handles both duties
- Enable `peer-gateway` and `ip arp synchronize` in every vPC domain without exception
- Set `delay restore` high enough for OSPF/BGP to converge (60-120s typical)
- Use `graceful consistency-check` (on by default) so only the secondary leg suspends on mismatch
- Monitor `show vpc statistics` for peer-link utilization; if consistently high, investigate traffic flows
- Never span a Layer-2 VLAN across vPC peers without also having the SVI on both peers
- Test failure scenarios in a maintenance window before production deployment
- Use `auto-recovery` in every deployment to handle simultaneous-reload edge cases
- Keep NX-OS versions identical on both vPC peers to avoid subtle consistency mismatches

## See Also

- port-channel
- spanning-tree
- hsrp
- lacp
- fex

## References

- Cisco Nexus 9000 vPC Configuration Guide — https://www.cisco.com/c/en/us/td/docs/switches/datacenter/nexus9000/sw/93x/vxlan/configuration/guide/b-cisco-nexus-9000-series-nx-os-vxlan-configuration-guide-93x/b-cisco-nexus-9000-series-nx-os-vxlan-configuration-guide-93x_chapter_0101.html
- Cisco vPC Design and Configuration Best Practices (CVD) — https://www.cisco.com/c/en/us/products/collateral/switches/nexus-5000-series-switches/design_guide_c07-625857.html
- Cisco Nexus 7000 vPC Operations Guide — https://www.cisco.com/c/en/us/support/docs/switches/nexus-7000-series-switches/200different-vpc-operations.html
- RFC 7348 — VXLAN (related when extending vPC with EVPN/VXLAN)
- Cisco NX-OS Verified Scalability Guide — vPC limits per platform
