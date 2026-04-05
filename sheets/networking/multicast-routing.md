# Multicast Routing (PIM, IGMP, RP Design, and Multicast Troubleshooting)

IP multicast enables efficient one-to-many and many-to-many delivery by replicating packets only at branch points in the distribution tree, using PIM for inter-router signaling, IGMP for host-to-router membership, and RP mechanisms to anchor shared trees before sources trigger shortest-path switchover.

## PIM-SM Operation Flow

### Overview of PIM Sparse Mode State Machine

```
PIM-SM distribution tree types:

 Shared Tree (*,G):    Rooted at RP, all sources share one tree
 Source Tree (S,G):    Shortest path from source to receivers, per-source

State transitions:
 1. Receiver joins -> (*,G) shared tree via RP
 2. Source sends   -> First-hop router Registers with RP
 3. RP joins source -> (S,G) tree from RP to source
 4. SPT switchover  -> Last-hop router builds (S,G) direct to source
 5. RP prunes       -> RP prunes (S,G) toward source (no longer needed)
```

### Register Process (Source Registration)

```
Source                  First-Hop Router (FHR)              RP
  |                          |                               |
  |--- multicast packet ---->|                               |
  |                          |                               |
  |                          |  (no (S,G) state yet)         |
  |                          |                               |
  |                          |--- PIM Register (unicast) --->|
  |                          |    [encapsulated mcast pkt]   |
  |                          |                               |
  |                          |                               |--- de-encapsulate
  |                          |                               |    forward down (*,G) tree
  |                          |                               |
  |                          |                               |--- PIM (S,G) Join
  |                          |                               |    toward source (RPF)
  |                          |                               |
  |                          |<--- native multicast ---------|
  |                          |    (S,G) tree established     |
  |                          |                               |
  |                          |<--- Register-Stop ------------|
  |                          |    (stop encapsulating)       |
  |                          |                               |
  |--- multicast packet ---->|--- native multicast --------->|
  |                          |    (forwarded natively)       |

! Register message format:
!   PIM Type 1, unicast to RP
!   Contains full original multicast packet as payload
!   Register-Stop: PIM Type 2, unicast back to FHR
```

### Join/Prune Messages

```
PIM Join/Prune message structure:

  Type: 3 (Join/Prune)
  Upstream Neighbor: address of RPF neighbor
  Holdtime: 210 seconds (default, 3.5x hello interval)
  Number of groups: N

  For each group:
    Group address: 239.1.1.1
    Number of joined sources: J
    Number of pruned sources: P
    Joined sources:  S1, S2, ...  (or *, for (*,G) join)
    Pruned sources:  S3, S4, ...  (or (S,G,rpt) prune)

! (*,G) Join: join to RP-rooted shared tree
!   Joined source = * (wildcard)
!   Sent hop-by-hop toward RP

! (S,G) Join: join to source-rooted shortest-path tree
!   Joined source = S
!   Sent hop-by-hop toward source

! (S,G,rpt) Prune: prune source from shared tree
!   Pruned source = S with RPT bit set
!   Sent toward RP to stop receiving S via shared tree
```

### Assert Mechanism

```
Assert occurs when two routers forward the same multicast stream
onto the same LAN segment (duplicate packets detected).

Router A ----+
             |--- LAN --- Receiver
Router B ----+

Both Router A and B have (S,G) or (*,G) state for the same group.
Both forward packets onto the LAN. Receiver gets duplicates.

Assert election:
  1. Both routers detect duplicate by receiving multicast from the other
  2. Both send PIM Assert message on the LAN
  3. Assert winner determined by:
     a. Lowest metric preference (AD of the route to source/RP)
     b. If tied: lowest metric (to source for (S,G), to RP for (*,G))
     c. If tied: highest IP address on the LAN interface
  4. Loser prunes its outgoing interface for that (S,G) or (*,G)

Assert message fields:
  Group address, Source address, RPT bit
  Metric preference (AD), Metric (route metric)

! Verify:
show ip mroute
! Look for "A" flag = Assert winner
! "L" on OIL means "Assert loser, interface pruned"
```

### SPT Switchover

```
SPT switchover: last-hop router transitions from (*,G) shared tree
to (S,G) shortest-path tree for a specific source.

Default behavior (Cisco IOS):
  Switchover occurs after receiving the FIRST packet from a source
  via the shared tree (threshold = 0 kbps).

Timeline:
  1. Receiver joins (*,G) via IGMP
  2. Router sends (*,G) Join toward RP
  3. Source starts sending -> traffic arrives via shared tree (RP)
  4. Last-hop router receives first packet from S via (*,G)
  5. Router immediately sends (S,G) Join toward source (RPF)
  6. (S,G) tree builds hop-by-hop from last-hop to source
  7. Traffic arrives via both (*,G) and (S,G) briefly
  8. Router sends (S,G,rpt) Prune toward RP
  9. RP stops forwarding S's traffic down shared tree to this router
  10. Traffic now flows only via (S,G) SPT

! Configure SPT threshold:
ip pim spt-threshold 0                    ! Immediate (default)
ip pim spt-threshold infinity             ! Never switch (stay on shared tree)
ip pim spt-threshold 50                   ! Switch after 50 kbps from source

! Per-group SPT threshold:
ip pim spt-threshold infinity group-list NO-SPT-GROUPS
ip access-list standard NO-SPT-GROUPS
 permit 239.192.0.0 0.0.255.255
```

## PIM-SSM (Source-Specific Multicast)

```
PIM-SSM eliminates the RP entirely. Receivers specify both group
AND source in their IGMP membership report (IGMPv3 required).

SSM range: 232.0.0.0/8 (default)

Operation:
  1. Host sends IGMPv3 Report: JOIN (S, G) where G in 232.0.0.0/8
  2. Last-hop router creates (S,G) state immediately
  3. Router sends PIM (S,G) Join directly toward source (RPF)
  4. No RP, no shared tree, no Register process
  5. Traffic flows on (S,G) SPT from source to receiver

Advantages:
  - No RP single point of failure
  - No shared tree (simpler state)
  - No Register encapsulation overhead
  - Immune to source spoofing (receiver specifies source)
  - Lower join latency (no RP detour)

Configuration:
  ip pim ssm default                      ! Enable SSM for 232.0.0.0/8
  ip pim ssm range SSM-RANGE              ! Custom SSM range
  ip access-list standard SSM-RANGE
   permit 232.0.0.0 0.0.255.255

! Interface config (IGMPv3 required for SSM):
  interface GigabitEthernet0/1
   ip igmp version 3
   ip pim sparse-mode
```

## PIM-BiDir (Bidirectional PIM)

```
PIM-BiDir: traffic flows both up and down the shared tree.
No source registration, no (S,G) state, no SPT switchover.
Only (*,G) state exists.

Use case: many-to-many (e.g., video conferencing, trading floors)

Key concept --- Designated Forwarder (DF):
  On each link, one router is elected DF for each RP
  DF is responsible for forwarding traffic toward the RP
  DF election uses same metric comparison as Assert

Operation:
  1. Receiver joins (*,G) toward RP (same as PIM-SM)
  2. Source sends multicast
  3. On each link, DF forwards traffic toward RP
  4. RP distributes down shared tree to receivers
  5. No (S,G) state created anywhere
  6. Traffic from any source uses the same (*,G) tree

Configuration:
  ip pim bidir-enable                     ! Global
  ip pim rp-address 10.0.0.1 bidir       ! RP for bidirectional groups

Limitations:
  - Cannot use SSM (no source-specific joins)
  - No SPT switchover (always uses shared tree)
  - Higher bandwidth usage for unicast-like traffic patterns
  - Not all platforms support BiDir
```

## IGMP (Internet Group Management Protocol)

### IGMPv2 Operation

```
Host                           Router (Querier)
  |                               |
  |<-- IGMP General Query --------|  (dst: 224.0.0.1, every 60s default)
  |     (Type 0x11, Group 0.0.0.0)|
  |                               |
  |--- IGMP Membership Report --->|  (Type 0x16, Group 239.1.1.1)
  |     (dst: 239.1.1.1)          |
  |                               |
  |    [Router adds interface to OIL for 239.1.1.1]
  |                               |
  |<-- IGMP Group-Specific Query -|  (when leave received)
  |     (Type 0x11, Group=239.1.1.1, dst: 239.1.1.1)
  |                               |
  |   [No response = group pruned after timeout]

Timers (IGMPv2 defaults):
  Query interval:              60 seconds
  Query response interval:     10 seconds (max response time)
  Group membership timeout:    260 seconds (2x query + response)
  Last member query interval:  1 second
  Last member query count:     2
  Robustness variable:         2

IGMP Querier election:
  Lowest IP address on the subnet wins
  Non-querier suppresses queries
  Querier timeout: 2x query interval + 0.5x response interval
```

### IGMPv3 Operation

```
IGMPv3 adds source filtering (required for SSM):

Report types:
  INCLUDE (S1, S2):    Receive only from sources S1, S2
  EXCLUDE (S3):        Receive from all sources EXCEPT S3
  INCLUDE ():          Leave group (empty include = no sources)

IGMPv3 Membership Report format:
  Type: 0x22
  Destination: 224.0.0.22 (all IGMPv3-capable routers)
  Group records:
    Record type: IS_IN, IS_EX, TO_IN, TO_EX, ALLOW, BLOCK
    Group address: 239.1.1.1
    Source list: [10.1.1.1, 10.2.2.2]

IGMPv3 is backward compatible:
  v3 router handles v2 reports (treats as EXCLUDE {} = all sources)
  v2 router ignores v3 reports (v3 hosts fall back to v2 on that segment)
```

### IGMP Snooping

```
IGMP snooping: Layer 2 switch optimization.
Without snooping: multicast flooded to all ports in VLAN.
With snooping: multicast forwarded only to ports with receivers.

Operation:
  1. Switch intercepts IGMP Reports and Queries
  2. Builds MAC-to-port mapping for multicast groups
  3. Group 239.1.1.1 -> 01:00:5e:01:01:01 -> ports Gi0/1, Gi0/3
  4. Multicast traffic only forwarded to those ports + mrouter port

Mrouter port detection:
  - Port where PIM Hello or IGMP Query is received
  - All multicast traffic is also sent to mrouter ports

Configuration:
  ip igmp snooping                        ! Global enable (default on)
  ip igmp snooping vlan 10                ! Per-VLAN enable
  ip igmp snooping vlan 10 querier        ! Switch acts as IGMP querier
  ip igmp snooping vlan 10 mrouter interface Gi0/24  ! Static mrouter port

Troubleshooting:
  show ip igmp snooping
  show ip igmp snooping groups
  show ip igmp snooping mrouter
  show mac address-table multicast
```

## RP Design

### Auto-RP

```
Auto-RP: Cisco proprietary, uses 224.0.1.39 (Announce) and
224.0.1.40 (Discovery) to distribute RP information.

Components:
  Candidate RP:      Announces itself via 224.0.1.39
  Mapping Agent:     Listens on 224.0.1.39, distributes RP-to-group
                     mapping via 224.0.1.40

Configuration:
  ! Candidate RP:
  ip pim send-rp-announce Loopback0 scope 16

  ! Mapping Agent:
  ip pim send-rp-discovery scope 16

  ! All routers (to receive Auto-RP before PIM is fully operational):
  ip pim autorp listener                  ! Flood Auto-RP in dense mode

Auto-RP chicken-and-egg problem:
  Routers need RP to join 224.0.1.39/40, but RP info is ON those groups.
  Solution: "ip pim autorp listener" treats Auto-RP groups as dense-mode.
```

### BSR (Bootstrap Router) --- RFC 5059

```
BSR: Standards-based RP discovery.
Uses PIM Bootstrap messages (hop-by-hop flooding, not multicast).

Components:
  Candidate BSR:     Elected via priority + IP (highest wins)
  Candidate RP:      Unicasts C-RP-Adv to BSR
  BSR:               Floods RP-set in Bootstrap messages to all routers

Configuration:
  ! Candidate BSR:
  ip pim bsr-candidate Loopback0 0 100    ! hash-mask-len 0, priority 100

  ! Candidate RP:
  ip pim rp-candidate Loopback0 group-list RP-GROUPS priority 10
  ip access-list standard RP-GROUPS
   permit 239.0.0.0 0.255.255.255

  ! BSR election: highest priority wins, then highest IP
  ! RP selection: hash function maps group to RP from the RP-set

BSR vs Auto-RP:
  BSR:     Standards (RFC 5059), hop-by-hop PIM flooding
  Auto-RP: Cisco proprietary, uses multicast (224.0.1.39/40)
  BSR:     No chicken-and-egg problem (PIM messages, not multicast)
  Auto-RP: Requires "autorp listener" workaround
```

### Anycast RP (with MSDP or RFC 4610)

```
Anycast RP: Multiple routers share the same RP IP address.
Provides RP redundancy and load sharing.

Method 1: Anycast RP + MSDP
  RP1: Loopback0 = 10.0.0.100/32 (shared RP address)
       Loopback1 = 10.0.0.1/32   (unique, for MSDP peering)
  RP2: Loopback0 = 10.0.0.100/32 (same shared RP address)
       Loopback1 = 10.0.0.2/32   (unique, for MSDP peering)

  IGP advertises 10.0.0.100/32 from both RP1 and RP2.
  Sources register with nearest RP (IGP metric).
  MSDP syncs source-active (SA) state between RPs.

  ! RP1 config:
  ip pim rp-address 10.0.0.100
  ip msdp peer 10.0.0.2 connect-source Loopback1
  ip msdp originator-id Loopback1

  ! RP2 config:
  ip pim rp-address 10.0.0.100
  ip msdp peer 10.0.0.1 connect-source Loopback1
  ip msdp originator-id Loopback1

Method 2: Anycast RP with PIM (RFC 4610)
  Uses PIM Register messages between RPs instead of MSDP.
  Newer, simpler, but requires IOS support.

  ip pim rp-address 10.0.0.100
  ip pim anycast-rp 10.0.0.100 10.0.0.1
  ip pim anycast-rp 10.0.0.100 10.0.0.2
```

### MSDP (Multicast Source Discovery Protocol)

```
MSDP: TCP-based protocol (port 639) that carries Source-Active (SA)
messages between RPs in different domains or for Anycast RP.

SA message contents:
  Source address, Group address, RP address
  Optionally: encapsulated first multicast packet

MSDP peering:
  ip msdp peer 10.0.0.2 connect-source Loopback0
  ip msdp sa-filter in MSDP-FILTER        ! Filter inbound SAs
  ip msdp sa-filter out MSDP-FILTER       ! Filter outbound SAs
  ip msdp cache-sa-state                   ! Cache SAs (default on)
  ip msdp cache-rejected-sa 600            ! Cache rejected SAs for debug

Verification:
  show ip msdp peer
  show ip msdp sa-cache
  show ip msdp summary
```

## RPF Check (Reverse Path Forwarding)

```
RPF check is the fundamental loop-prevention mechanism in multicast.

Rule: A multicast packet from source S arriving on interface I is
accepted ONLY if I is the interface the router would use to route
a unicast packet BACK to S (the RPF interface).

RPF check process:
  1. Packet from S arrives on interface Gi0/1
  2. Router looks up S in unicast RIB (or MRIB if configured)
  3. Unicast route to S points to Gi0/2 (RPF interface)
  4. Gi0/1 != Gi0/2 -> RPF check FAILS -> packet DROPPED

For (*,G) state: RPF check is toward the RP
For (S,G) state: RPF check is toward the source S

RPF failure is the #1 cause of multicast black holes.

! Verify RPF:
show ip rpf 10.1.1.1                     ! RPF lookup for source 10.1.1.1
! Output:
!   RPF information for 10.1.1.1:
!     RPF interface: GigabitEthernet0/0
!     RPF neighbor:  192.168.1.1
!     RPF route/mask: 10.1.1.0/24
!     RPF type: unicast (ospf)
!     RPF recursion count: 0

! Static RPF override (when unicast and multicast topologies differ):
ip mroute 10.1.1.0 255.255.255.0 192.168.2.1
! Forces RPF for 10.1.1.0/24 through 192.168.2.1
```

## Multicast in VRF

```
! PIM must be enabled per-VRF:
ip pim vrf CUSTOMER-A rp-address 10.100.0.1
ip pim vrf CUSTOMER-A ssm default

interface GigabitEthernet0/1
 ip vrf forwarding CUSTOMER-A
 ip address 10.100.1.1 255.255.255.0
 ip pim sparse-mode

! VRF-aware IGMP:
interface GigabitEthernet0/1
 ip igmp version 3

! Verification (VRF-aware):
show ip mroute vrf CUSTOMER-A
show ip pim vrf CUSTOMER-A neighbor
show ip pim vrf CUSTOMER-A rp mapping
show ip rpf vrf CUSTOMER-A 10.100.0.1
show ip igmp vrf CUSTOMER-A groups
```

## mVPN Basics (Multicast VPN)

```
mVPN profiles (RFC 6513/6514 framework):

Profile 0:  Default MDT (GRE, PIM/SSM in core)
Profile 1:  Rosen GRE with BGP Auto-Discovery (AD)
Profile 2:  Default MDT, mLDP in core
Profile 3:  Default MDT, IR (Ingress Replication) in core
Profile 7:  Rosen GRE, BGP AD, data MDT with mLDP
Profile 12: Default MDT, IR with partitioned MDT
Profile 14: Default and data MDT, IR, BGP AD + BGP C-mcast signaling

! Profile 0 example (Rosen draft):
vrf definition CUSTOMER-A
 rd 65000:100
 route-target export 65000:100
 route-target import 65000:100
 address-family ipv4
  mdt default 239.1.1.1
  mdt data 239.1.2.0 0.0.0.255 threshold 50
  ! Default MDT: low-rate traffic, all PEs join
  ! Data MDT: high-rate sources get dedicated tree (above threshold)

! Verify:
show ip pim vrf CUSTOMER-A mdt send
show ip pim vrf CUSTOMER-A mdt receive
show ip mroute vrf CUSTOMER-A
```

## Multicast Troubleshooting Methodology

### Step-by-Step Troubleshooting

```
1. Verify source is sending:
   show ip mroute <group>
   ! Look for (S,G) entry with non-zero packet count on FHR

2. Verify RP:
   show ip pim rp mapping
   show ip pim rp mapping <group>
   ! Confirm correct RP for the group range

3. Verify RPF to source and RP:
   show ip rpf <source-ip>
   show ip rpf <rp-ip>
   ! Ensure RPF interface matches expected topology

4. Verify IGMP membership:
   show ip igmp groups
   show ip igmp membership
   ! Confirm receiver host has joined the group

5. Verify PIM neighbors:
   show ip pim neighbor
   ! Ensure PIM adjacency on all relevant interfaces

6. Verify mroute state:
   show ip mroute <group>
   show ip mroute <source> <group>
   ! Check flags, incoming interface, outgoing interface list (OIL)

7. Verify mroute counters:
   show ip mroute <group> count
   ! Check for incrementing packet/byte counters

8. Check for RPF failures:
   show ip mroute <group>
   ! "RPF nbr: 0.0.0.0" indicates RPF failure
   ! No incoming interface = broken RPF

9. Test end-to-end:
   mtrace <source> <group>
   ! Trace multicast path from source to receiver
```

### show ip mroute Flags

```
Flags in show ip mroute output:

Flag  Meaning
────────────────────────────────────────────────────────────────
T     SPT bit set (traffic flowing on shortest-path tree)
S     Sparse mode
C     Connected (directly connected receiver/source)
L     Local (router is last-hop for this group)
P     Pruned (no downstream receivers)
F     Register flag (FHR is registering with RP)
R     RP-bit set ((*,G) entry, shared tree)
J     Join SPT (transitioning from shared tree to SPT)
M     MSDP created entry
A     Assert winner (won Assert election on OIL interface)
X     Proxy Join Timer running
U     URD (URL Rendezvous Directory)
I     Received source on incoming interface (RPF passed)
Z     Multicast tunnel
E     Extranet
K     Keepalive timer running

Common flag combinations:
  (*,G) S, C, R:           Shared tree, connected receiver, RP-rooted
  (S,G) S, T:              Source tree, SPT active, traffic flowing
  (S,G) S, P:              Source tree, pruned (no receivers downstream)
  (S,G) S, F:              FHR registering source with RP
  (*,G) S, R, P:           Shared tree, no receivers (all on SPT)
```

### Common Multicast Issues

```
Problem: RPF failure
  Symptom: show ip rpf <source> shows wrong interface or "not found"
  Cause: Unicast routing doesn't match multicast topology
  Fix: Fix unicast routing or add "ip mroute <source> <mask> <rpf-nbr>"

Problem: RP unreachable
  Symptom: show ip pim rp mapping shows no RP
  Cause: Auto-RP/BSR not propagating, static RP misconfigured
  Fix: Verify RP config, check PIM neighbor on path to RP

Problem: IGMP not working
  Symptom: show ip igmp groups shows no members
  Cause: IGMP snooping blocking reports, version mismatch, host issue
  Fix: Check IGMP snooping config, verify host IGMP version

Problem: Duplicate packets
  Symptom: Receivers get 2x packets
  Cause: Assert failure, redundant paths without Assert election
  Fix: Check "show ip mroute" for Assert state, verify PIM on all links

Problem: Source not Registering
  Symptom: (S,G) on FHR has no F flag, RP shows no (S,G)
  Cause: FHR cannot unicast to RP, RP unreachable
  Fix: Verify FHR can reach RP (show ip rpf <rp>), check ACLs

Problem: SPT switchover not occurring
  Symptom: Traffic always via RP, never direct from source
  Cause: ip pim spt-threshold infinity configured
  Fix: Check spt-threshold config, set to 0 for immediate switchover

Problem: Multicast across VRF boundary fails
  Symptom: mroute exists but traffic not flowing
  Cause: Missing ip pim vrf config, RPF failure in VRF context
  Fix: Verify PIM enabled in VRF, check RPF within VRF routing table
```

### Key Debug Commands

```
debug ip pim                              ! PIM protocol messages
debug ip pim <group>                      ! PIM for specific group
debug ip igmp                             ! IGMP reports and queries
debug ip mrouting                         ! Mroute table changes
debug ip mpacket                          ! Multicast packet forwarding (verbose!)
debug ip pim auto-rp                      ! Auto-RP announcements
debug ip pim bsr                          ! BSR election and RP-set

! Always filter debugs in production:
debug ip pim group 239.1.1.1
debug condition interface GigabitEthernet0/0
```

## Tips

- Always start multicast design from the receiver side: verify IGMP membership first, then trace the tree back toward the RP and source.
- Use PIM-SSM (232.0.0.0/8) for any application where receivers know the source address --- it eliminates RP dependencies entirely.
- Deploy Anycast RP for RP redundancy in PIM-SM; use MSDP between RPs or RFC 4610 PIM Anycast RP for state synchronization.
- Set `ip pim spt-threshold infinity` on access-layer routers only if you want to keep traffic on the shared tree (reduces state but increases RP load).
- Always verify RPF with `show ip rpf <source>` before troubleshooting anything else --- 80% of multicast problems are RPF failures.
- Enable `ip igmp snooping querier` on VLANs where no Layer 3 router runs IGMP queries (otherwise snooping tables time out).
- Use `show ip mroute count` to verify traffic is actually flowing; a valid mroute entry with zero packets means something upstream is broken.
- For mVPN, start with Profile 0 (Rosen GRE) for simplicity, then migrate to mLDP or IR profiles as needed.
- Monitor `show ip pim rp mapping` regularly --- stale or conflicting RP mappings cause silent multicast failures.
- Use `mtrace` for end-to-end multicast path verification; it is the multicast equivalent of traceroute.

## See Also

- `igmp` --- IGMP protocol details, host membership, snooping configuration
- `ospf` --- OSPF multicast routing integration, DR election on multi-access
- `bgp` --- BGP multicast extensions (MBGP), mVPN address families
- `vlan` --- VLAN configuration, IGMP snooping per-VLAN settings
- `fabric-multicast` --- Data center multicast, VXLAN multicast underlay
- `ipsec` --- Multicast over IPsec (GRE encapsulation requirements)
- `mpls` --- mLDP for multicast in MPLS networks

## References

- RFC 7761 --- Protocol Independent Multicast - Sparse Mode (PIM-SM): Protocol Specification (Revised)
- RFC 3376 --- Internet Group Management Protocol, Version 3 (IGMPv3)
- RFC 2236 --- Internet Group Management Protocol, Version 2 (IGMPv2)
- RFC 4601 --- Protocol Independent Multicast - Sparse Mode (PIM-SM) (obsoleted by RFC 7761)
- RFC 3569 --- An Overview of Source-Specific Multicast (SSM)
- RFC 4607 --- Source-Specific Multicast for IP
- RFC 5015 --- Bidirectional Protocol Independent Multicast (BIDIR-PIM)
- RFC 5059 --- Bootstrap Router (BSR) Mechanism for PIM
- RFC 3618 --- Multicast Source Discovery Protocol (MSDP)
- RFC 4610 --- Anycast-RP Using Protocol Independent Multicast (PIM)
- RFC 6513 --- Multicast in MPLS/BGP IP VPNs
- RFC 6514 --- BGP Encodings and Procedures for Multicast in MPLS/BGP IP VPNs
- Cisco PIM Configuration Guide --- https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/ipmulti_pim/configuration/xe-16/imc-pim-xe-16-book.html
- Cisco IGMP Configuration Guide --- https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/ipmulti_igmp/configuration/xe-16/imc-igmp-xe-16-book.html
- "Developing IP Multicast Networks, Volume I" by Beau Williamson (Cisco Press)
