# Fabric Multicast (PIM, IGMP Snooping, and DC Multicast Routing)

Protocols and techniques for efficient one-to-many and many-to-many traffic delivery within data center fabrics, covering PIM variants, IGMP snooping, rendezvous point design, and multicast in VXLAN EVPN overlays.

## Concepts

### Multicast Addressing

- **IPv4 multicast range:** 224.0.0.0/4 (224.0.0.0 - 239.255.255.255)
- **Link-local scope:** 224.0.0.0/24 (TTL=1, never forwarded — OSPF, VRRP, IGMP)
- **SSM range:** 232.0.0.0/8 (source-specific multicast, no RP needed)
- **Admin-scoped:** 239.0.0.0/8 (private, like RFC 1918 for multicast)
- **Globally-scoped:** 224.0.1.0 - 238.255.255.255 (internet multicast)
- **L2 mapping:** IP multicast 224.x.y.z maps to MAC 0100.5e + lower 23 bits of IP
- **Overlap problem:** 32:1 address overlap (5 bits lost), so 224.1.1.1 and 225.1.1.1 share the same MAC

### Multicast Distribution Trees

- **RPT (Shared Tree / *, G):** Traffic flows from source to RP, then RP to receivers; single tree rooted at RP; suboptimal paths but conserves state
- **SPT (Shortest Path Tree / S, G):** Direct path from source to receivers; optimal path; more state per (S,G) entry
- **SPT switchover:** Receivers initially join RPT, then switch to SPT when traffic exceeds threshold (default 0 kbps = immediate switchover on Cisco)
- **SPT threshold:** `ip pim spt-threshold infinity` keeps traffic on RPT (reduces state at cost of suboptimal paths)

### PIM Sparse Mode (PIM-SM)

- **RFC 7761** — the dominant multicast routing protocol in enterprise/DC
- Uses explicit join model — receivers must request group membership
- Requires a Rendezvous Point (RP) for each group
- Builds RPT first, then can switch to SPT
- Multicast packets from source are registered at RP via PIM Register messages
- RP forwards to receivers down the shared tree
- **PIM Hello:** Sent every 30s on all PIM-enabled interfaces (holdtime 105s)
- **PIM Join/Prune:** Sent upstream hop-by-hop toward RP or source (every 60s)
- **PIM Register:** Source's DR unicasts multicast packets to RP in Register messages
- **PIM Register-Stop:** RP tells source DR to stop sending Registers (native multicast path established)
- **Assert:** Resolves duplicate traffic on multi-access segments (lower metric wins, then lower IP)

### PIM Source-Specific Multicast (PIM-SSM)

- **RFC 4607** — simplified model, no RP required
- Receivers specify exact source with IGMPv3 (S,G) join
- Builds SPT directly from source to receiver — no RPT, no RP
- Range: 232.0.0.0/8 by default
- Eliminates RP failure as single point of failure
- Ideal when sources are known (IPTV, financial feeds, live video)
- Requires IGMPv3 on receiver-facing interfaces

### PIM Dense Mode (PIM-DM)

- Flood-and-prune model — pushes traffic everywhere, then prunes branches with no receivers
- State-refresh mechanism re-floods periodically (every 60s)
- Not suitable for DC fabrics — generates excessive flooding
- No RP needed, but O(S*G) state on all routers regardless of receivers
- Largely deprecated in modern networks

### PIM Bidirectional (PIM-BiDir)

- **RFC 5015** — shared tree only, no source registration, no SPT switchover
- Traffic flows both directions on the shared tree through the RP
- RP is a "phantom" — uses Designated Forwarder (DF) election on each link
- DF election: lowest-metric path to RP wins; ties broken by highest IP
- Scales well for many-to-many applications (video conferencing, collaboration)
- No (S,G) state — only (*, G) entries — dramatically reduces MRIB size
- Trade-off: suboptimal paths (all traffic via RP tree)

### IGMP (Internet Group Management Protocol)

- **IGMPv1 (RFC 1112):** Join only, no leave — relies on timeout (basic, obsolete)
- **IGMPv2 (RFC 2236):** Adds explicit leave + group-specific query (leave latency ~3s)
- **IGMPv3 (RFC 3376):** Adds source filtering — INCLUDE/EXCLUDE mode, required for SSM
- **IGMP Query:** General query every 125s (default), group-specific on leave
- **IGMP Querier Election:** Lowest IP address on the segment wins querier role
- **IGMP Robustness Variable:** Default 2, controls retransmissions (group timeout = robustness * query interval + max response time)
- **Max Response Time:** 10s default for general query, 1s for group-specific

### IGMP Snooping

- **L2 switch optimization:** Inspects IGMP packets to build MAC-to-port-to-group table
- Without snooping: all multicast is flooded like broadcast on the VLAN
- **Snooping querier:** Switch can act as IGMP querier when no L3 querier exists on the VLAN
- **Mrouter port:** Port where the multicast router is connected; all multicast forwarded here
- **Fast-leave (immediate leave):** Port removed from group instantly on leave (only safe with single host per port)
- **Report suppression:** Switch suppresses duplicate IGMP reports (disable for IGMPv3)
- **TCN flood:** On topology change, switch floods multicast for a configurable period

### Rendezvous Point (RP) Design

- **Static RP:** Manually configured on all routers — simple but no redundancy without Anycast RP
- **Auto-RP (Cisco proprietary):** RP-Announce (224.0.1.39) + RP-Discovery (224.0.1.40); requires `ip pim autorp listener` on sparse-mode interfaces
- **BSR (Bootstrap Router) — RFC 5059:** BSR elected by priority (highest wins), distributes RP-set to all PIM routers via hop-by-hop flooding; candidate RPs announce to BSR
- **Anycast RP with MSDP — RFC 3446:** Multiple RPs share the same IP (loopback); MSDP peers sync source-active (SA) messages between RPs; provides redundancy and load sharing
- **Phantom RP (BiDir only):** RP address does not need to exist on any router — used only as a vector for DF election

### Reverse Path Forwarding (RPF)

- **Fundamental multicast check:** Packet arriving on an interface must be the interface toward the source (unicast routing table lookup)
- If RPF check fails, packet is dropped — prevents loops
- RPF interface = interface in unicast RIB pointing toward source (or RP for shared trees)
- **Static mroute:** Override RPF with `ip mroute <source> <mask> <rpf-neighbor/interface>`
- In asymmetric routing environments, RPF failures are common — use static mroutes or multicast-specific routing

### MSDP (Multicast Source Discovery Protocol)

- **RFC 3618** — used between PIM-SM domains or between Anycast RPs
- Carries Source-Active (SA) messages: (S, G, RP) tuples
- SA messages are flooded peer-to-peer via TCP (port 639)
- **SA cache:** Stores active sources learned from MSDP peers
- **Mesh-group:** All members of a mesh-group have full-mesh MSDP peering; SA messages received from a mesh-group peer are not forwarded to other mesh-group peers (reduces flooding)
- Critical for Anycast RP — without MSDP, each RP only knows about sources registered to it

## IOS Configuration

### Enable PIM Sparse Mode on Interfaces

```
! Enable multicast routing globally
ip multicast-routing

! Enable PIM-SM on all L3 interfaces participating in multicast
interface Loopback0
 ip address 10.0.0.1 255.255.255.255
 ip pim sparse-mode

interface GigabitEthernet0/0
 ip address 10.1.1.1 255.255.255.0
 ip pim sparse-mode

interface GigabitEthernet0/1
 ip address 10.1.2.1 255.255.255.0
 ip pim sparse-mode
```

### Static RP Configuration

```
! Define RP for all multicast groups
ip pim rp-address 10.0.0.100

! Define RP for a specific group range using ACL
access-list 10 permit 239.1.0.0 0.0.255.255
ip pim rp-address 10.0.0.100 10
```

### Auto-RP Configuration

```
! On the RP candidate router
ip pim send-rp-announce Loopback0 scope 16 group-list 10

! On the mapping agent (can be same or different router)
ip pim send-rp-discovery Loopback0 scope 16

! On all PIM-SM routers — required so sparse-mode interfaces
! forward Auto-RP multicast (224.0.1.39 and 224.0.1.40)
ip pim autorp listener
```

### BSR Configuration

```
! Candidate BSR — advertise this router as a potential BSR
ip pim bsr-candidate Loopback0 30 10
!                     interface   hash-mask  priority (higher wins)

! Candidate RP — announce willingness to be RP to BSR
ip pim rp-candidate Loopback0 group-list 10 priority 100
!                    interface               priority (lower wins for RP)

! Verify
show ip pim bsr-router
show ip pim rp mapping
```

### Anycast RP with MSDP

```
! Both RP routers share the same Anycast RP address on Loopback
! RP1 (Router-ID 10.0.0.1)
interface Loopback0
 ip address 10.0.0.1 255.255.255.255
 ip pim sparse-mode
interface Loopback1
 ip address 10.0.0.100 255.255.255.255
 ip pim sparse-mode

ip pim rp-address 10.0.0.100
ip msdp peer 10.0.0.2 connect-source Loopback0
ip msdp originator-id Loopback0

! RP2 (Router-ID 10.0.0.2)
interface Loopback0
 ip address 10.0.0.2 255.255.255.255
 ip pim sparse-mode
interface Loopback1
 ip address 10.0.0.100 255.255.255.255
 ip pim sparse-mode

ip pim rp-address 10.0.0.100
ip msdp peer 10.0.0.1 connect-source Loopback0
ip msdp originator-id Loopback0
```

### PIM-SSM Configuration

```
! Enable SSM for the default range (232.0.0.0/8)
ip pim ssm default

! Or specify a custom SSM range
access-list 20 permit 232.0.0.0 0.255.255.255
access-list 20 permit 239.232.0.0 0.0.255.255
ip pim ssm range 20

! Ensure IGMPv3 on receiver-facing interfaces
interface GigabitEthernet0/0
 ip igmp version 3
```

### PIM BiDir Configuration

```
ip multicast-routing

! Designate RP for BiDir groups
ip pim bidir-enable
ip pim rp-address 10.0.0.100 bidir

! Enable PIM on interfaces
interface GigabitEthernet0/0
 ip pim sparse-mode
! Note: BiDir uses sparse-mode interfaces — DF election happens automatically
```

### IGMP Snooping (IOS)

```
! IGMP snooping is enabled globally by default
ip igmp snooping

! Enable snooping querier on a VLAN (when no L3 querier exists)
ip igmp snooping vlan 100 querier
ip igmp snooping vlan 100 querier address 10.1.100.1

! Fast leave — only when one host per port
ip igmp snooping vlan 100 immediate-leave

! Limit number of IGMP groups per port
ip igmp snooping vlan 100 limit 500

! Static mrouter port
ip igmp snooping vlan 100 mrouter interface GigabitEthernet0/1

! Verify
show ip igmp snooping
show ip igmp snooping groups
show ip igmp snooping querier
```

### SPT Threshold

```
! Keep traffic on shared tree (RPT) — do not switch to SPT
ip pim spt-threshold infinity

! Apply only for specific groups
ip pim spt-threshold infinity group-list SPT-DENY
ip access-list standard SPT-DENY
 permit 239.99.0.0 0.0.255.255
```

## NX-OS Configuration

### Enable PIM on Nexus

```
! Enable PIM feature
feature pim

! Enable multicast routing for default VRF
ip pim rp-address 10.0.0.100 group-list 239.0.0.0/8

! Enable on interfaces
interface loopback0
 ip address 10.0.0.1/32
 ip pim sparse-mode

interface Ethernet1/1
 ip address 10.1.1.1/30
 ip pim sparse-mode
 no shutdown

interface Ethernet1/2
 ip address 10.1.2.1/30
 ip pim sparse-mode
 no shutdown
```

### NX-OS Anycast RP with PIM (No MSDP)

```
! NX-OS supports Anycast RP without MSDP using PIM Anycast RP set
! Spine 1
interface loopback0
 ip address 10.0.0.1/32
 ip pim sparse-mode
interface loopback100
 ip address 10.0.0.100/32
 ip pim sparse-mode

ip pim rp-address 10.0.0.100 group-list 224.0.0.0/4
ip pim anycast-rp 10.0.0.100 10.0.0.1
ip pim anycast-rp 10.0.0.100 10.0.0.2

! Spine 2
interface loopback0
 ip address 10.0.0.2/32
 ip pim sparse-mode
interface loopback100
 ip address 10.0.0.100/32
 ip pim sparse-mode

ip pim rp-address 10.0.0.100 group-list 224.0.0.0/4
ip pim anycast-rp 10.0.0.100 10.0.0.1
ip pim anycast-rp 10.0.0.100 10.0.0.2
```

### NX-OS IGMP Snooping

```
! IGMP snooping is enabled by default
! Configure querier
ip igmp snooping vlan 100 querier 10.1.100.1

! Optimized multicast flooding (OMF) — suppresses unknown multicast
ip igmp snooping vlan 100 optimise-multicast-flood

! Fast leave
ip igmp snooping vlan 100 fast-leave

! Static multicast router port
ip igmp snooping vlan 100 mrouter interface port-channel1

! Verify
show ip igmp snooping
show ip igmp snooping groups vlan 100
show ip igmp snooping statistics vlan 100
```

### PIM in Spine-Leaf Fabric

```
! Spine-Leaf PIM-SM design:
!   - Spines = RP (Anycast RP across all spines)
!   - Leaves = PIM last-hop router (DR for receivers)
!   - PIM enabled on all fabric uplinks and SVIs

! Leaf switch
feature pim
ip pim rp-address 10.0.0.100 group-list 224.0.0.0/4

interface Ethernet1/49
 description To-Spine-1
 ip address 10.1.1.1/30
 ip pim sparse-mode

interface Ethernet1/50
 description To-Spine-2
 ip address 10.1.2.1/30
 ip pim sparse-mode

interface Vlan100
 ip address 10.100.1.1/24
 ip pim sparse-mode
 ip igmp version 3
```

### Multicast with vPC

```
! vPC multicast considerations:
!   - Both vPC peers must have identical PIM and IGMP snooping config
!   - vPC primary is the PIM DR and IGMP querier (by default)
!   - PIM Hello uses the vPC virtual IP if configured
!   - Multicast traffic is forwarded by both peers but only one copy egresses

! vPC peer 1 (primary)
feature pim
feature vpc

vpc domain 1
 peer-keepalive destination 10.255.1.2 source 10.255.1.1

interface Vlan100
 ip address 10.100.1.2/24
 ip pim sparse-mode
 ip igmp version 3
 ! Lower IP wins DR election — vPC primary should be DR
 ip pim dr-priority 10

interface port-channel100
 vpc 100

! vPC peer 2 (secondary)
interface Vlan100
 ip address 10.100.1.3/24
 ip pim sparse-mode
 ip igmp version 3
 ip pim dr-priority 5

interface port-channel100
 vpc 100

! IGMP snooping must be identical on both peers
ip igmp snooping vlan 100 querier 10.100.1.2
```

### Multicast in VXLAN EVPN Fabric

```
! Two approaches for BUM traffic replication in VXLAN:
!
! 1. Ingress Replication (Head-End Replication)
!    - Source VTEP unicasts BUM to each remote VTEP
!    - No multicast in the underlay needed
!    - Scales poorly with many VTEPs (N copies per BUM frame)
!    - Simple to deploy
!
! 2. Multicast Underlay
!    - VTEPs join a multicast group per VNI (or group of VNIs)
!    - Underlay PIM replicates BUM efficiently
!    - Requires PIM in the underlay (typically PIM-SM or PIM-BiDir)
!    - Better BUM scalability

! --- Ingress Replication (NX-OS) ---
feature nv overlay
feature vn-segment-vlan-based
nv overlay evpn

interface nve1
 no shutdown
 host-reachability protocol bgp
 source-interface loopback1
 member vni 10100
  ingress-replication protocol bgp
  ! BGP EVPN distributes VTEP list — no multicast needed

! --- Multicast Underlay (NX-OS) ---
feature pim

ip pim rp-address 10.0.0.100 group-list 239.0.0.0/8
! Each VNI maps to a multicast group for BUM replication
interface nve1
 no shutdown
 host-reachability protocol bgp
 source-interface loopback1
 member vni 10100
  mcast-group 239.1.1.1
 member vni 10200
  mcast-group 239.1.1.2

! Spines run PIM-SM and serve as Anycast RP
! Leaves join multicast groups for their local VNIs
```

## Verification Commands

### IOS

```
! PIM neighbors and interface status
show ip pim neighbor
show ip pim interface
show ip pim rp mapping

! Multicast routing table
show ip mroute
show ip mroute 239.1.1.1
show ip mroute 239.1.1.1 count
show ip mroute active

! IGMP groups and interface
show ip igmp groups
show ip igmp interface GigabitEthernet0/0
show ip igmp membership

! RPF check
show ip rpf 10.1.1.100

! MSDP state
show ip msdp peer
show ip msdp sa-cache
show ip msdp summary

! PIM register state
show ip pim rp mapping
show ip pim rp-hash 239.1.1.1
```

### NX-OS

```
! PIM state
show ip pim neighbor
show ip pim interface brief
show ip pim rp
show ip pim group-range
show ip pim route
show ip pim internal event-history

! Multicast routing
show ip mroute
show ip mroute 239.1.1.1
show ip mroute summary

! IGMP
show ip igmp groups
show ip igmp interface vlan 100
show ip igmp snooping groups vlan 100

! VXLAN multicast
show nve peers
show nve vni
show nve multisite fabric-links

! RPF
show ip rpf 10.1.1.100

! vPC multicast state
show vpc consistency-parameters global
show vpc consistency-parameters interface port-channel100
```

## Troubleshooting

### Common Issues

```
! Issue: Multicast traffic not reaching receivers
! 1. Verify PIM adjacency
show ip pim neighbor
! If neighbor missing: check ip pim sparse-mode on both sides, check L3 reachability

! 2. Check RPF
show ip rpf <source-ip>
! RPF failure = packets dropped; fix routing or add static mroute
debug ip mfib pak

! 3. Verify RP mapping
show ip pim rp mapping
! Ensure all routers agree on RP for the group

! 4. Check IGMP state
show ip igmp groups
show ip igmp snooping groups vlan <id>
! No group = receiver not joining or IGMP filtered

! 5. Check mroute table for flags
show ip mroute 239.1.1.1
! Key flags: (S,G) SPT = on shortest path, (T) = forwarding
! (P) = pruned, (R) = RP-bit set, no outgoing interfaces = pruned

! Issue: Duplicate multicast on vPC
! Verify DR election and vPC consistency
show ip pim interface vlan 100
show vpc consistency-parameters global
! Both peers must have identical PIM/IGMP config

! Issue: VXLAN BUM not replicating
show nve peers
show nve vni 10100
! For multicast underlay: verify mcast-group is correct and PIM is operational
! For ingress-replication: verify BGP EVPN is distributing Type-3 routes
show bgp l2vpn evpn route-type 3
```

### Debug Commands

```
! Use with caution in production — high CPU
debug ip pim
debug ip igmp
debug ip mroute
debug ip msdp

! NX-OS event-history (non-disruptive)
show ip pim internal event-history
show ip igmp internal event-history
show ip mroute internal event-history
```

## Tips

- Always enable PIM on loopbacks used as RP or router-id — missing PIM on the RP loopback is the most common deployment mistake
- In spine-leaf, place RPs on spines with Anycast RP for redundancy — leaves should never be RPs
- Use PIM-SSM (232.0.0.0/8) whenever sources are known — eliminates RP dependency entirely
- For VXLAN EVPN, ingress replication is simpler but does not scale past ~64 VTEPs per VNI; switch to multicast underlay for large fabrics
- IGMP snooping querier is mandatory on VLANs without a PIM-enabled L3 interface — without a querier, snooping entries age out and multicast floods
- On vPC, always verify `show vpc consistency-parameters global` — PIM and IGMP mismatches cause silent failures
- Set `ip pim spt-threshold infinity` on leaf switches when you want to keep traffic on the shared tree and minimize (S,G) state in the fabric
- PIM BiDir is excellent for many-to-many workloads but poorly supported on some platforms — verify hardware support before deploying
- MSDP mesh-groups reduce SA flooding between Anycast RP peers — always configure mesh-groups when you have 3+ RPs
- MTU must accommodate VXLAN overhead (50 bytes) — set underlay MTU to at least 9050 for 9000-byte jumbo inner frames

## See Also

- VXLAN
- BGP
- OSPF
- ECMP

## References

- RFC 7761 — PIM Sparse Mode (PIM-SM)
- RFC 4607 — Source-Specific Multicast (PIM-SSM)
- RFC 5015 — Bidirectional PIM (PIM-BiDir)
- RFC 3973 — PIM Dense Mode
- RFC 3376 — IGMPv3
- RFC 2236 — IGMPv2
- RFC 4541 — IGMP and MLD Snooping Considerations
- RFC 3618 — MSDP (Multicast Source Discovery Protocol)
- RFC 5059 — BSR Mechanism for PIM
- RFC 3446 — Anycast RP using MSDP
- RFC 7432 — BGP EVPN
- RFC 8365 — VXLAN EVPN
- Cisco NX-OS Multicast Configuration Guide
- Cisco IOS IP Multicast Configuration Guide
