# RSVP-TE (Resource Reservation Protocol — Traffic Engineering)

Signaling protocol that establishes MPLS TE tunnels with explicit paths and bandwidth reservations, supporting fast reroute, make-before-break, and point-to-multipoint LSPs.

## Concepts

### Core Signaling

- **PATH message:** Sent downstream (ingress to egress) carrying ERO, label request, sender TSpec
- **RESV message:** Sent upstream (egress to ingress) carrying label assignment, FlowSpec, RRO
- **PathTear:** Tears down an LSP from ingress toward egress
- **ResvTear:** Tears down reservation from egress toward ingress
- **PathErr:** Error notification sent upstream (does not tear down state)
- **ResvErr:** Error notification sent downstream

### Key Objects

| Object | Direction | Purpose |
|--------|-----------|---------|
| ERO (Explicit Route Object) | PATH | Specifies hop-by-hop path (strict/loose hops) |
| RRO (Record Route Object) | RESV/PATH | Records actual path taken (labels and addresses) |
| Label Request | PATH | Requests downstream label allocation |
| Label | RESV | Carries allocated label upstream |
| Sender TSpec | PATH | Describes traffic parameters (bandwidth) |
| FlowSpec | RESV | Confirms reserved bandwidth |
| Session Attribute | PATH | Carries setup/hold priority, SE style, FRR flags |
| DETOUR | PATH | Signals one-to-one backup path |
| RRO (in PATH) | PATH | Records ingress-to-current path for loop detection |

### Soft-State Refresh

- RSVP-TE is soft-state: PATH and RESV messages must be refreshed periodically
- Default refresh interval: **30 seconds**
- State times out after missing **3 consecutive refreshes** (configurable)
- Summary Refresh (RFC 2961) reduces overhead by bundling refresh state into MESSAGE_ID ACK exchanges
- Bundle messages group multiple RSVP messages into a single IP packet

## Tunnel Setup

### IOS-XE Configuration

```
! Enable MPLS TE globally
mpls traffic-eng tunnels

! Enable TE on OSPF (or IS-IS)
router ospf 1
 mpls traffic-eng router-id Loopback0
 mpls traffic-eng area 0

! Enable TE on interfaces
interface GigabitEthernet0/0/0
 mpls traffic-eng tunnels
 ip rsvp bandwidth 1000000
 ! Allocatable bandwidth in kbps

! Create TE tunnel
interface Tunnel0
 ip unnumbered Loopback0
 tunnel mode mpls traffic-eng
 tunnel destination 10.0.0.5
 tunnel mpls traffic-eng autoroute announce
 tunnel mpls traffic-eng bandwidth 50000
 tunnel mpls traffic-eng priority 3 3
 ! Setup priority 3, hold priority 3
 tunnel mpls traffic-eng path-option 1 explicit name STRICT-PATH
 tunnel mpls traffic-eng path-option 2 dynamic
```

### IOS-XR Configuration

```
mpls traffic-eng
 interface GigabitEthernet0/0/0/0
  bandwidth 1000000
 !
!

interface tunnel-te0
 ipv4 unnumbered Loopback0
 destination 10.0.0.5
 signalled-bandwidth 50000
 priority 3 3
 autoroute announce
 path-option 1 explicit name STRICT-PATH
 path-option 2 dynamic
!

explicit-path name STRICT-PATH
 index 1 next-address strict ipv4 unicast 10.1.1.2
 index 2 next-address strict ipv4 unicast 10.2.2.2
 index 3 next-address strict ipv4 unicast 10.0.0.5
!
```

## Explicit Path

### IOS-XE Explicit Path

```
! Strict hops — exact path, all hops specified
ip explicit-path name STRICT-PATH enable
 next-address strict 10.1.1.2
 next-address strict 10.2.2.2
 next-address strict 10.3.3.2

! Loose hops — intermediate routing allowed between hops
ip explicit-path name LOOSE-PATH enable
 next-address loose 10.1.1.2
 next-address loose 10.3.3.2

! Mixed strict and loose
ip explicit-path name MIXED-PATH enable
 next-address strict 10.1.1.2
 next-address loose 10.3.3.2

! Exclude address — avoid a node
ip explicit-path name AVOID-NODE enable
 exclude-address 10.2.2.2
```

### CSPF (Constrained SPF)

- RSVP-TE uses CSPF to compute paths that satisfy constraints (bandwidth, affinity)
- CSPF prunes links that lack required bandwidth, then runs SPF on remaining topology
- Affinity bits (link attributes / admin groups) allow include/exclude of link colors
- If CSPF fails, the tunnel stays down unless a dynamic fallback path-option exists

```
! IOS-XE affinity configuration
interface Tunnel0
 tunnel mpls traffic-eng affinity 0x0000000A mask 0x0000000F
 ! Requires links with bit 1 and bit 3 set

! Set link attribute on interface
interface GigabitEthernet0/0/0
 mpls traffic-eng attribute-flags 0x0000000A
```

## Bandwidth Reservation and Preemption

### Setup and Hold Priority

- Range: **0 (highest) to 7 (lowest)**
- **Setup priority:** Priority used when establishing the LSP (can it preempt others?)
- **Hold priority:** Priority used when defending the LSP (can it be preempted?)
- Rule: setup priority should be >= hold priority (numerically; lower number = higher priority)
- A new LSP with setup priority N can preempt existing LSPs with hold priority > N

### Bandwidth Model

```
! IOS-XE: per-interface reservable bandwidth
interface GigabitEthernet0/0/0
 ip rsvp bandwidth 1000000 1000000
 ! First value: interface bandwidth pool
 ! Second value: max per-flow reservation

! IOS-XR: sub-pool (BC1) for priority traffic
mpls traffic-eng
 interface GigabitEthernet0/0/0/0
  bandwidth 1000000 sub-pool 200000
```

### Preemption Example

| Tunnel | Bandwidth | Setup | Hold | Result |
|--------|-----------|-------|------|--------|
| T1 | 500 Mbps | 4 | 4 | Established first |
| T2 | 600 Mbps | 2 | 2 | Preempts T1 (setup 2 < hold 4) |
| T3 | 300 Mbps | 5 | 5 | Cannot preempt T2 (setup 5 > hold 2) |

## Fast Reroute (FRR)

### Facility Backup (Bypass Tunnel)

- Pre-computed backup tunnel around a potential failure point
- PLR (Point of Local Repair) detects failure and reroutes traffic into bypass tunnel
- MP (Merge Point) is where traffic re-enters the original path
- Label stack: outer = bypass tunnel label, inner = original tunnel label (nested LSP)
- **NHOP bypass:** Protects against link failure (next-hop backup)
- **NNHOP bypass:** Protects against node failure (next-next-hop backup)

```
! IOS-XE: Enable FRR on the protected tunnel
interface Tunnel0
 tunnel mpls traffic-eng fast-reroute

! Create bypass tunnel on the PLR
interface Tunnel100
 ip unnumbered Loopback0
 tunnel mode mpls traffic-eng
 tunnel destination 10.2.2.2
 tunnel mpls traffic-eng path-option 1 explicit name BYPASS-PATH
 ! Mark as backup tunnel
 mpls traffic-eng backup-path Tunnel100

! Assign bypass to protected interface
interface GigabitEthernet0/0/0
 mpls traffic-eng backup-path Tunnel100
```

```
! IOS-XR FRR
interface tunnel-te0
 fast-reroute
!

interface tunnel-te100
 ipv4 unnumbered Loopback0
 destination 10.2.2.2
 path-option 1 explicit name BYPASS-PATH
 backup-bw global-pool 500000
!

mpls traffic-eng
 interface GigabitEthernet0/0/0/0
  backup-path tunnel-te 100
```

### One-to-One Backup (Detour)

- Each protected LSP gets its own dedicated backup path
- PLR signals a DETOUR object in the PATH message
- More resource intensive than facility backup (each LSP has its own detour)
- Provides per-LSP protection granularity

```
! IOS-XE: one-to-one backup
interface Tunnel0
 tunnel mpls traffic-eng fast-reroute protect node
 ! 'protect node' enables NNHOP protection
```

### FRR Switchover

- BFD or RSVP Hello detects failure in **milliseconds**
- PLR immediately switches traffic to backup path
- Ingress re-signals the primary LSP via a new path (global repair)
- Once primary re-converges, traffic moves back from backup

## Make-Before-Break (MBB)

### Concepts

- Used during re-optimization or bandwidth change without traffic disruption
- New LSP is signaled **before** the old one is torn down
- Both LSPs share the same session (Shared Explicit / SE style)
- SE style allows both old and new LSPs to share bandwidth on common links
- Sequence: signal new path, verify it is up, move traffic, tear down old path

```
! IOS-XE: MBB is the default re-optimization behavior
interface Tunnel0
 tunnel mpls traffic-eng path-option 1 explicit name NEW-PATH
 tunnel mpls traffic-eng path-option 2 explicit name OLD-PATH
 ! Re-optimization timer (seconds)
 tunnel mpls traffic-eng auto-bw
  frequency 300
  min-bw 10000
  max-bw 900000
  adjustment-threshold 10
```

### IOS-XR Re-optimization

```
mpls traffic-eng
 reoptimize timers frequency 300
!

interface tunnel-te0
 auto-bw
  bw-limit min 10000 max 900000
  adjustment-threshold 10 percent
  application 15
 !
!
```

## RSVP-TE Hello Protocol

- Lightweight keepalive between RSVP-TE neighbors
- Detects neighbor failure independent of routing protocol
- Default hello interval: **5 seconds**, miss count: **3** (15-second detection)
- Faster detection than relying on PATH/RESV refresh timeout (90 seconds)

```
! IOS-XE: RSVP hello
ip rsvp signalling hello
 ! Global enable
ip rsvp signalling hello graceful-restart
 ! Enable GR via hello

! Per-interface tuning
interface GigabitEthernet0/0/0
 ip rsvp signalling hello
```

```
! IOS-XR: RSVP hello
rsvp
 signalling hello graceful-restart
  interface GigabitEthernet0/0/0/0
   hello interval 3000
   hello missed 4
  !
!
```

## RSVP-TE Graceful Restart

### Concepts

- Allows a restarting router to preserve forwarding state during control plane restart
- Neighbors maintain reservation state for the restarting node
- GR helper: neighbor that preserves state on behalf of the restarting router
- Recovery time: period during which the restarting router re-learns state
- Restart time: maximum time the helper waits before tearing down state

```
! IOS-XE
ip rsvp signalling hello graceful-restart mode help-neighbor
ip rsvp signalling hello graceful-restart send recovery-time 120000
ip rsvp signalling hello graceful-restart send restart-time 180000

! IOS-XR
rsvp
 signalling hello graceful-restart
  restart-time 180
  recovery-time 120
 !
!
```

## Summary Refresh (RFC 2961)

```
! IOS-XE: enable summary refresh
ip rsvp signalling refresh reduction

! IOS-XR
rsvp
 signalling refresh reduction
  summary max-size 1500
  bundle-messages max-size 4096
 !
!
```

- MESSAGE_ID and MESSAGE_ID_ACK replace per-message refresh
- Reduces RSVP control plane load from O(n) messages to O(1) summary per refresh interval
- Reliable delivery via ACK/NACK mechanism
- SRefresh message contains list of MESSAGE_IDs that are still valid

## P2MP RSVP-TE

### Concepts

- Point-to-Multipoint LSPs: single ingress, multiple egress (leaf) nodes
- Used for multicast replication in MPLS networks
- Sub-LSPs branch at bifurcation points
- Signaled per-leaf: each leaf has its own S2L (Source-to-Leaf) sub-LSP
- Grafting: adding a new leaf without disrupting existing leaves
- Pruning: removing a leaf without affecting others

```
! IOS-XR P2MP tunnel
interface tunnel-mte0
 ipv4 unnumbered Loopback0
 destination 10.0.0.5
  path-option 1 dynamic
 !
 destination 10.0.0.6
  path-option 1 dynamic
 !
 destination 10.0.0.7
  path-option 1 dynamic
 !
 signalled-bandwidth 100000
!
```

## Verification and Troubleshooting

### IOS-XE

```
! Show all TE tunnels and their state
show mpls traffic-eng tunnels

! Show detailed tunnel info including ERO, RRO, bandwidth
show mpls traffic-eng tunnels tunnel 0 detail

! Show RSVP reservations
show ip rsvp reservation

! Show RSVP sender (PATH state)
show ip rsvp sender

! Show RSVP interface bandwidth usage
show ip rsvp interface

! Show RSVP neighbors and hello state
show ip rsvp hello instance

! Show FRR protection status
show mpls traffic-eng fast-reroute database

! Show explicit paths
show ip explicit-paths

! Debug RSVP signaling (use with caution)
debug ip rsvp
debug mpls traffic-eng tunnels signaling
```

### IOS-XR

```
! Show tunnel status
show mpls traffic-eng tunnels

! Show RSVP sessions
show rsvp session

! Show RSVP neighbors
show rsvp neighbors

! Show RSVP interface
show rsvp interface

! Show FRR status
show mpls traffic-eng fast-reroute database

! Show P2MP tunnels
show mpls traffic-eng tunnels p2mp
```

## Tips

- Always enable summary refresh (RFC 2961) in production to reduce RSVP control plane overhead.
- Use NNHOP bypass tunnels for node protection; NHOP bypass only protects against link failure.
- Set setup priority >= hold priority (numerically) to prevent preemption oscillation.
- BFD integration with RSVP-TE provides sub-second failure detection for FRR switchover.
- Make-before-break is the default for re-optimization; do not manually tear down tunnels for path changes.
- CSPF failures are silent from the data plane perspective; monitor tunnel state changes.
- P2MP RSVP-TE scales poorly with many leaves; consider mLDP or SR-based multicast for large deployments.
- Auto-bandwidth adjusts reservations based on measured traffic; set min/max bounds to prevent over/under provisioning.
- RSVP-TE state is per-hop; every transit router maintains PATH/RESV state for every LSP, which limits scalability.

## See Also

- mpls, ospf, is-is, bgp, bfd, vxlan

## References

- [RFC 3209 — RSVP-TE: Extensions to RSVP for LSP Tunnels](https://www.rfc-editor.org/rfc/rfc3209)
- [RFC 4090 — Fast Reroute Extensions to RSVP-TE](https://www.rfc-editor.org/rfc/rfc4090)
- [RFC 2961 — RSVP Refresh Overhead Reduction](https://www.rfc-editor.org/rfc/rfc2961)
- [RFC 3473 — GMPLS Signaling RSVP-TE Extensions](https://www.rfc-editor.org/rfc/rfc3473)
- [RFC 4875 — Extensions to RSVP-TE for P2MP TE LSPs](https://www.rfc-editor.org/rfc/rfc4875)
- [RFC 5063 — Extensions to GMPLS RSVP Graceful Restart](https://www.rfc-editor.org/rfc/rfc5063)
- [RFC 3471 — Generalized MPLS Signaling Functional Description](https://www.rfc-editor.org/rfc/rfc3471)
- [Cisco IOS-XE MPLS TE Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/mp_te_path/configuration/xe-16/mp-te-path-xe-16-book.html)
- [Cisco IOS-XR MPLS TE Configuration Guide](https://www.cisco.com/c/en/us/td/docs/routers/asr9000/software/mpls/configuration/guide/b-mpls-cg-asr9000.html)
