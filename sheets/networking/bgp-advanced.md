# BGP Advanced (Path Attributes, Route Reflectors, Convergence, and Security)

Advanced BGP topics including path attribute manipulation, route-reflector design, confederations, ADD-PATH, PIC, FlowSpec, BGP-LS, RPKI validation, and the complete 15-step best path selection algorithm with IOS-XE and IOS-XR configuration examples.

## Path Attributes Deep-Dive

### Attribute Classification

| Attribute | Type Code | Category | Transitive | Description |
|:---|:---:|:---|:---:|:---|
| ORIGIN | 1 | Well-known mandatory | Yes | How route entered BGP (IGP/EGP/Incomplete) |
| AS_PATH | 2 | Well-known mandatory | Yes | Ordered list of ASes route has traversed |
| NEXT_HOP | 3 | Well-known mandatory | Yes | IP address of next-hop router |
| MED | 4 | Optional non-transitive | No | Multi-Exit Discriminator — hint to neighbor AS |
| LOCAL_PREF | 5 | Well-known discretionary | No (iBGP only) | Preference within local AS (higher = better) |
| ATOMIC_AGGREGATE | 6 | Well-known discretionary | Yes | Indicates route was aggregated, detail lost |
| AGGREGATOR | 7 | Optional transitive | Yes | AS and router-id that performed aggregation |
| COMMUNITY | 8 | Optional transitive | Yes | 32-bit tags (RFC 1997) |
| ORIGINATOR_ID | 9 | Optional non-transitive | No | Loop prevention for route reflectors |
| CLUSTER_LIST | 10 | Optional non-transitive | No | List of RR cluster-ids traversed |
| EXTENDED_COMMUNITY | 16 | Optional transitive | Yes | 64-bit typed communities (RT, SOO, etc.) |
| LARGE_COMMUNITY | 32 | Optional transitive | Yes | 96-bit communities (ASN:value:value) — RFC 8092 |

### COMMUNITY Attribute

```
! Standard communities (RFC 1997) — 32-bit: AA:NN
! Well-known communities:
!   no-export          (0xFFFFFF01) — do not advertise to eBGP peers
!   no-advertise       (0xFFFFFF02) — do not advertise to any peer
!   no-export-subconfed(0xFFFFFF03) — do not advertise outside confederation
!   no-peer            (0xFFFFFF04) — do not advertise to bilateral peers

! IOS-XE: Set community on routes
route-map SET-COMMUNITY permit 10
 set community 65000:100 65000:200 additive

! IOS-XR: Set community
route-policy SET-COMMUNITY
  set community (65000:100, 65000:200) additive
end-policy
```

### EXTENDED_COMMUNITY Attribute

```
! 64-bit typed communities — used heavily in MPLS VPN and EVPN
! Format: Type:Admin:Assigned
!
! Common types:
!   Route Target (RT)   — controls VRF import/export
!   Site of Origin (SOO) — prevents routing loops in multi-homed CE
!   OSPF Domain ID       — distinguishes OSPF instances across PE
!   BGP Cost Community   — additional path selection tie-breaking

! IOS-XE: Route target in VRF
vrf definition CUSTOMER-A
 rd 65000:100
 address-family ipv4
  route-target export 65000:100
  route-target import 65000:100

! IOS-XR: Extended community in route-policy
route-policy SET-RT
  set extcommunity rt (65000:100) additive
end-policy
```

### LARGE_COMMUNITY Attribute

```
! 96-bit communities (RFC 8092) — designed for 4-byte ASNs
! Format: GlobalAdmin:LocalData1:LocalData2
! Each field is 32-bit — no more AS truncation problems

! IOS-XE (17.x+):
route-map SET-LARGE-COMMUNITY permit 10
 set large-community 4200000001:100:200 additive

! IOS-XR:
route-policy SET-LARGE
  set large-community (4200000001:100:200) additive
end-policy

! Filtering on large community:
ip large-community-list standard BLOCK-LIST permit 4200000001:666:0
route-map FILTER-IN deny 10
 match large-community BLOCK-LIST
route-map FILTER-IN permit 20
```

### MED (Multi-Exit Discriminator) Manipulation

```
! MED tells external neighbor which entry point to prefer (lower = better)
! Only compared between paths from same neighbor AS by default

! IOS-XE: Set MED
route-map SET-MED-OUT permit 10
 set metric 100

router bgp 65000
 neighbor 192.168.1.2 route-map SET-MED-OUT out
 ! Compare MED across all neighbor ASes (dangerous — beware MED oscillation)
 bgp always-compare-med
 ! Use deterministic MED comparison (group by AS before comparing)
 bgp deterministic-med

! IOS-XR:
router bgp 65000
 bgp bestpath med always
 bgp bestpath med confed
 neighbor 192.168.1.2
  address-family ipv4 unicast
   route-policy SET-MED-OUT out
```

### LOCAL_PREF Manipulation

```
! LOCAL_PREF is the primary iBGP routing policy tool (higher = preferred)
! Default: 100. Only carried within iBGP (stripped at AS boundary)

! IOS-XE: Prefer routes from provider A over provider B
ip prefix-list PROVIDER-A-ROUTES permit 0.0.0.0/0 le 32
route-map FROM-PROVIDER-A permit 10
 set local-preference 200

route-map FROM-PROVIDER-B permit 10
 set local-preference 50

! IOS-XR:
route-policy FROM-PROVIDER-A
  set local-preference 200
end-policy
```

## BGP Best Path Selection — Full 15-Step Algorithm

```
Order of evaluation (first decisive match wins):

 Step  Attribute/Criterion           Rule
 ────  ─────────────────────────────  ──────────────────────────────────
  1    NEXT_HOP reachability          Next-hop must be resolvable in RIB
  2    Weight (Cisco-only)            Highest wins (local to router)
  3    LOCAL_PREF                     Highest wins (default 100)
  4    Locally originated             network/redistribute/aggregate preferred
  5    AS_PATH length                 Shortest wins (unless as-path ignore)
  6    ORIGIN type                    IGP(i) < EGP(e) < Incomplete(?)
  7    MED                            Lowest wins (same neighbor AS only*)
  8    eBGP over iBGP                 External paths preferred
  9    IGP metric to NEXT_HOP         Lowest IGP cost to reach next-hop
 10    Oldest eBGP path               Most stable/oldest external path
 11    Lowest neighbor router-id      Tiebreaker
 12    Shortest CLUSTER_LIST          Fewer RR hops preferred
 13    Lowest neighbor address        Final tiebreaker (neighbor IP)
 14    Path with lowest path-id       ADD-PATH tiebreaker
 15    DMZ-Link Bandwidth             Highest link bandwidth (IOS-XR)

 * Step 7: "bgp always-compare-med" compares MED across all ASes
 * Step 5: "bgp bestpath as-path ignore" skips AS-path length
 * Step 10: only applies if multipath not configured
```

### IOS-XR Best Path Tuning

```
router bgp 65000
 bgp bestpath as-path ignore              ! Skip step 5
 bgp bestpath compare-routerid            ! Always compare router-id
 bgp bestpath med always                  ! MED across all ASes
 bgp bestpath med confed                  ! MED in confederations
 bgp bestpath med missing-as-worst        ! Treat missing MED as 4294967295
 bgp bestpath cost-community ignore       ! Skip cost community evaluation
 address-family ipv4 unicast
  bgp bestpath aigp ignore                ! Skip AIGP in best path
```

## Route Reflectors

### Cluster Design

```
                    ┌─────────┐
                    │  RR-1   │ cluster-id 1.1.1.1
                    │ (spine) │
                    └────┬────┘
            ┌────────────┼────────────┐
       ┌────┴────┐  ┌────┴────┐  ┌────┴────┐
       │ Client  │  │ Client  │  │ Client  │
       │   R1    │  │   R2    │  │   R3    │
       └─────────┘  └─────────┘  └─────────┘

  Redundant design: Two RRs per cluster (active/backup)

       ┌─────────┐     ┌─────────┐
       │  RR-1   │─────│  RR-2   │  Same cluster-id: 1.1.1.1
       └────┬────┘     └────┬────┘
            │               │
       ┌────┴───────────────┴────┐
       │   All clients peer to   │
       │   BOTH RR-1 and RR-2   │
       └─────────────────────────┘
```

### IOS-XE Route Reflector Configuration

```
router bgp 65000
 bgp cluster-id 1.1.1.1
 ! Mark neighbors as RR clients
 neighbor 10.0.0.1 remote-as 65000
 neighbor 10.0.0.1 update-source Loopback0
 address-family ipv4 unicast
  neighbor 10.0.0.1 route-reflector-client
  ! Optional: disable client-to-client reflection (for partial mesh clients)
  no bgp client-to-client reflection
```

### IOS-XR Route Reflector Configuration

```
router bgp 65000
 bgp cluster-id 1.1.1.1
 neighbor-group RR-CLIENTS
  remote-as 65000
  update-source Loopback0
  address-family ipv4 unicast
   route-reflector-client
 !
 neighbor 10.0.0.1
  use neighbor-group RR-CLIENTS
 neighbor 10.0.0.2
  use neighbor-group RR-CLIENTS
```

### Hierarchical Route Reflectors

```
  Tier-1 RR (top):    RR-A ── RR-B     (non-client iBGP between them)
                       │        │
  Tier-2 RR (mid):   RR-C    RR-D      (clients of Tier-1; RRs for Tier-3)
                      / \      / \
  Tier-3 (leaf):    R1  R2  R3  R4     (clients of Tier-2)

  ! Tier-1 RR config:
  router bgp 65000
   neighbor 10.0.1.1 remote-as 65000     ! RR-C is client
   address-family ipv4 unicast
    neighbor 10.0.1.1 route-reflector-client
   neighbor 10.0.2.1 remote-as 65000     ! RR-B is non-client peer
```

### Optimal RR Placement

```
! Place RR on the forwarding path to prevent suboptimal routing
! Rule: RR should be between clients and exit points
! If RR is off-path, use next-hop-self on RR or add-path

! Verify RR is on forwarding path:
show bgp ipv4 unicast <prefix>
! Check: NEXT_HOP visible from all clients via IGP through the RR
```

### Virtual Route Reflector (vRR)

```
! Run RR as a VM/container — not in the forwarding path
! Advantages: decouple control plane from data plane
! Used in: large DC fabrics, SDN architectures
! Considerations:
!   - vRR only handles BGP control plane (no traffic forwarding)
!   - Must still maintain iBGP sessions with all clients
!   - Scale: single vRR can handle 500+ peers with 1M+ routes
!   - Redundancy: deploy 2 vRRs per cluster
```

## BGP Confederations

```
! Split large AS into sub-ASes for iBGP scalability
! External world sees single AS; internally each sub-AS runs iBGP full mesh

  External AS 200 ──── [ AS 65000 (public AS) ]
                        ┌──────────────────────┐
                        │  Sub-AS 65501        │
                        │  R1 ── R2            │
                        │    \  /              │
                        │ Sub-AS 65502         │
                        │  R3 ── R4            │
                        └──────────────────────┘

! IOS-XE: Sub-AS 65501 router config
router bgp 65501
 bgp confederation identifier 65000
 bgp confederation peers 65502 65503
 neighbor 10.0.0.3 remote-as 65502      ! eBGP-like to other sub-AS
 neighbor 10.0.0.2 remote-as 65501      ! iBGP within sub-AS
 address-family ipv4 unicast
  neighbor 10.0.0.3 next-hop-self       ! Important for inter-sub-AS

! IOS-XR:
router bgp 65501
 bgp confederation identifier 65000
 bgp confederation peers
  65502
  65503
 !
 neighbor 10.0.0.3
  remote-as 65502
  address-family ipv4 unicast
   next-hop-self
```

## BGP ADD-PATH

```
! Advertise multiple paths for the same prefix (RFC 7911)
! Enables better convergence and multipath without tricks

! IOS-XE:
router bgp 65000
 address-family ipv4 unicast
  ! Advertise additional paths
  bgp additional-paths select all        ! Select all paths for advertisement
  bgp additional-paths send receive      ! Enable send/receive capability
  neighbor 10.0.0.1 advertise additional-paths all
  ! Or selective: best N paths
  bgp additional-paths select best 3

! IOS-XR:
router bgp 65000
 address-family ipv4 unicast
  additional-paths receive
  additional-paths send
  additional-paths selection route-policy ADD-PATH-POLICY
 !
 neighbor 10.0.0.1
  address-family ipv4 unicast
   additional-paths receive
   additional-paths send
```

## BGP PIC (Prefix Independent Convergence)

```
! Pre-compute backup paths so failover is O(1) per prefix
! Without PIC: convergence time scales with number of prefixes
! With PIC: failover in ~50ms regardless of table size

  Normal convergence:    T = k * (number of affected prefixes)
  PIC convergence:       T = constant (~50ms)

! IOS-XE: Enable PIC Edge
router bgp 65000
 address-family ipv4 unicast
  bgp additional-paths install          ! Install backup path in RIB
  bgp bestpath prefix-sid-map install   ! For SR-enabled networks

! On CE-facing interface:
interface GigabitEthernet0/0/1
 ip bgp pic edge                        ! Per-interface PIC edge

! IOS-XR: PIC Edge (enabled by default in modern XR)
router bgp 65000
 address-family ipv4 unicast
  additional-paths install backup       ! Install backup path
 !
cef
 platform lptsp-if-null-for-pic        ! Platform-specific PIC tuning
```

## BGP Diverse Path

```
! Advertise best path + best path from different next-hop
! Less aggressive than full ADD-PATH — only 2 paths max

! IOS-XE:
router bgp 65000
 address-family ipv4 unicast
  bgp additional-paths select backup    ! Select diverse/backup path
  neighbor 10.0.0.1 advertise diverse-path backup
```

## BGP Optimal Route Reflection (ORR)

```
! RR selects best path from CLIENT's perspective, not its own
! Solves the problem of RR choosing suboptimal paths for clients

! IOS-XR:
router bgp 65000
 address-family ipv4 unicast
  optimal-route-reflection igp-metric   ! Use client's IGP metric
 !
 neighbor 10.0.0.1
  address-family ipv4 unicast
   optimal-route-reflection igp-metric
```

## BGP Graceful Restart and Graceful Shutdown

### Graceful Restart (RFC 4724)

```
! Maintain forwarding during BGP restart — stale routes kept temporarily

! IOS-XE:
router bgp 65000
 bgp graceful-restart
 bgp graceful-restart restart-time 120       ! Time to re-establish sessions
 bgp graceful-restart stalepath-time 360     ! Time to keep stale routes
 bgp graceful-restart notification           ! Restart on notification received

! IOS-XR:
router bgp 65000
 bgp graceful-restart
 bgp graceful-restart restart-time 120
 bgp graceful-restart stalepath-time 360
 bgp graceful-restart purge-time 600
```

### Graceful Shutdown (RFC 8326)

```
! Signal planned maintenance — peers reroute traffic before shutdown
! Uses well-known community GRACEFUL_SHUTDOWN (65535:0)

! IOS-XE: Shutdown a single neighbor
router bgp 65000
 neighbor 192.168.1.2 shutdown graceful 120 community 65535:0

! IOS-XE: Activate graceful shutdown for all peers
route-map GSHUT-IMPORT permit 10
 match community GSHUT
 set local-preference 0

! Receiving side should match GRACEFUL_SHUTDOWN community and lower LOCAL_PREF
ip community-list standard GSHUT permit 65535:0
route-map FROM-PEER permit 5
 match community GSHUT
 set local-preference 0
route-map FROM-PEER permit 10

! IOS-XR:
router bgp 65000
 neighbor 192.168.1.2
  graceful-maintenance
   activate
   local-preference 0
   as-prepends 3
```

## BGP FlowSpec (RFC 5575)

```
! Distribute traffic filtering rules via BGP — DDoS mitigation
! Matches on: src/dst IP, protocol, port, DSCP, fragment, packet length, ICMP

! IOS-XR: Define flowspec rule
flowspec
 address-family ipv4
  flow BLOCK-DDOS
   match destination 10.0.0.0/24
   match protocol udp
   match destination-port 53
   match packet-length 512-65535
   action traffic-rate 0              ! Drop matching traffic (rate=0)

! Apply via BGP:
router bgp 65000
 address-family ipv4 flowspec
  neighbor 10.0.0.1 activate

! Verify:
show flowspec afi-all

! Common actions:
!   traffic-rate 0              — drop
!   traffic-rate 1000000        — rate-limit to 1Mbps
!   redirect VRF:name           — redirect to scrubbing VRF
!   traffic-marking dscp 0      — remark DSCP
```

## BGP-LS (Link-State — RFC 7752)

```
! Export IGP topology (OSPF/IS-IS) into BGP NLRI for SDN controllers
! Enables controller to have complete network topology view

! IOS-XR: Distribute OSPF topology via BGP-LS
router ospf 1
 distribute link-state instance-id 100

router bgp 65000
 address-family link-state link-state
  neighbor 10.0.0.100 activate         ! SDN controller
 !

! IOS-XE:
router ospf 1
 distribute link-state

router bgp 65000
 address-family link-state
  neighbor 10.0.0.100 activate

! BGP-LS NLRI types:
!   Node    — router with attributes (SR SID, capabilities)
!   Link    — adjacency with TE attributes (BW, delay, SRLG)
!   Prefix  — reachable prefix with attributes (prefix SID)
```

## BGP RPKI/ROA Validation (RFC 6811)

```
! Route Origin Validation — protect against route hijacking
! Validates: (Prefix, MaxLength, Origin AS) against ROA database

! Validation states:
!   Valid     — ROA exists, prefix and origin AS match
!   Invalid   — ROA exists, origin AS does NOT match (hijack!)
!   NotFound  — No ROA exists for this prefix

! IOS-XE: Configure RPKI cache server
router bgp 65000
 bgp rpki server tcp 10.0.0.50 port 323 refresh 300
 ! Or with SSH transport:
 bgp rpki server ssh 10.0.0.50 port 22 username rpki refresh 300

 address-family ipv4 unicast
  ! Apply validation to best path selection
  bgp bestpath prefix-validate allow-invalid   ! Mark but don't reject
  ! Or strict: reject invalid
  bgp bestpath prefix-validate disallow-invalid

! IOS-XR:
router bgp 65000
 rpki server 10.0.0.50
  transport tcp port 323
  refresh-time 300
 !
 address-family ipv4 unicast
  bgp origin-as validation enable
  bgp origin-as validation signal ibgp

! Route-map enforcement:
route-map RPKI-FILTER deny 10
 match rpki invalid
route-map RPKI-FILTER permit 20
 match rpki valid not-found

! Verification:
show bgp rpki table
show bgp rpki servers
show bgp ipv4 unicast rpki validation
```

## BGP Dampening (RFC 2439)

```
! Suppress unstable (flapping) routes to protect convergence
! Parameters: half-life, reuse, suppress, max-suppress-time

! IOS-XE:
router bgp 65000
 address-family ipv4 unicast
  bgp dampening 15 750 2000 60
  ! half-life=15min, reuse=750, suppress=2000, max-suppress=60min
  ! Per-prefix dampening with route-map:
  bgp dampening route-map DAMPENING-POLICY

route-map DAMPENING-POLICY permit 10
 match ip address prefix-list CUSTOMER-ROUTES
 set dampening 10 500 1500 40
route-map DAMPENING-POLICY permit 20
 set dampening 15 750 2000 60

! IOS-XR:
router bgp 65000
 address-family ipv4 unicast
  bgp dampening 15 750 2000 60

! Penalty mechanics:
!   Flap event = +1000 penalty
!   Penalty decays: P(t) = P(0) * 2^(-t/half-life)
!   Route suppressed when penalty > suppress threshold
!   Route reused when penalty < reuse threshold

! Verification:
show bgp ipv4 unicast dampening dampened-paths
show bgp ipv4 unicast dampening flap-statistics
show bgp ipv4 unicast dampening parameters
```

## Show and Verification Commands

### IOS-XE

```
! Path attributes and best path detail
show bgp ipv4 unicast 10.0.0.0/24
show bgp ipv4 unicast 10.0.0.0/24 bestpath

! Route reflector clients
show bgp ipv4 unicast neighbors 10.0.0.1 | include client

! Community filtering verification
show bgp ipv4 unicast community 65000:100

! RPKI validation status
show bgp ipv4 unicast rpki validation
show bgp rpki servers

! ADD-PATH
show bgp ipv4 unicast 10.0.0.0/24 all
show bgp ipv4 unicast neighbors 10.0.0.1 | include Additional

! FlowSpec
show bgp ipv4 flowspec summary
show bgp ipv4 flowspec detail

! BGP-LS
show bgp link-state link-state summary

! Dampening
show bgp ipv4 unicast dampening dampened-paths
```

### IOS-XR

```
! Detailed path info
show bgp ipv4 unicast 10.0.0.0/24
show bgp ipv4 unicast 10.0.0.0/24 bestpath-compare

! RR cluster info
show bgp ipv4 unicast neighbors 10.0.0.1 | include cluster

! Optimal route reflection
show bgp ipv4 unicast 10.0.0.0/24 | include ORR

! RPKI
show bgp rpki summary
show bgp rpki table

! Process details
show bgp process detail
show bgp convergence
```

## Tips

- Place route reflectors on the forwarding path between clients and exit points to avoid suboptimal routing; if off-path, enable ADD-PATH or next-hop-self on the RR.
- Always assign the same `cluster-id` to redundant RRs in the same cluster; different cluster-ids create separate clusters and may cause routing loops.
- Use `bgp deterministic-med` to group paths by neighbor AS before comparing MED; without it, MED comparison order depends on arrival order and can oscillate.
- Enable RPKI validation on all eBGP sessions and at minimum log invalid routes; strict drop policies require careful ROA coverage analysis first.
- When deploying FlowSpec, test rules in monitor mode before enabling drop actions; an incorrect match can black-hole legitimate traffic.
- Use LARGE_COMMUNITY (RFC 8092) instead of standard communities for 4-byte ASN networks; the 32-bit standard community format cannot represent large ASNs natively.
- BGP PIC requires a backup path in the RIB; combine with ADD-PATH or diverse-path to ensure backup paths are available.
- For graceful shutdown, configure receiving routers to match GRACEFUL_SHUTDOWN community and set LOCAL_PREF to 0 before the maintenance window.
- Confederation sub-AS numbers should use private AS range (64512-65534 for 2-byte, 4200000000-4294967294 for 4-byte).
- Dampening is controversial in modern networks; many operators disable it on eBGP sessions because it delays convergence for legitimate topology changes.
- BGP-LS is essential for SDN/SR-TE controllers but should only be enabled toward the controller, not flooded to all BGP peers.
- When using hierarchical RRs, ensure ORIGINATOR_ID and CLUSTER_LIST are preserved end-to-end for loop prevention.

## See Also

- bgp, ospf, ospf-advanced, is-is, mpls, mpls-vpn, bfd, ecmp, rpki, segment-routing, sd-wan

## References

- [RFC 4271 — BGP-4](https://www.rfc-editor.org/rfc/rfc4271)
- [RFC 4456 — BGP Route Reflection](https://www.rfc-editor.org/rfc/rfc4456)
- [RFC 5065 — BGP Confederations](https://www.rfc-editor.org/rfc/rfc5065)
- [RFC 7911 — BGP ADD-PATH](https://www.rfc-editor.org/rfc/rfc7911)
- [RFC 5575 — BGP FlowSpec](https://www.rfc-editor.org/rfc/rfc5575)
- [RFC 7752 — BGP-LS (Link-State)](https://www.rfc-editor.org/rfc/rfc7752)
- [RFC 6811 — RPKI-Based Route Origin Validation](https://www.rfc-editor.org/rfc/rfc6811)
- [RFC 8092 — BGP Large Communities](https://www.rfc-editor.org/rfc/rfc8092)
- [RFC 4724 — Graceful Restart Mechanism for BGP](https://www.rfc-editor.org/rfc/rfc4724)
- [RFC 8326 — Graceful BGP Session Shutdown](https://www.rfc-editor.org/rfc/rfc8326)
- [RFC 2439 — BGP Route Flap Dampening](https://www.rfc-editor.org/rfc/rfc2439)
- [RFC 1997 — BGP Communities](https://www.rfc-editor.org/rfc/rfc1997)
- [RFC 4360 — BGP Extended Communities](https://www.rfc-editor.org/rfc/rfc4360)
- [Cisco IOS-XE BGP Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/iproute_bgp/configuration/xe-16/irg-xe-16-book.html)
- [Cisco IOS-XR BGP Configuration Guide](https://www.cisco.com/c/en/us/td/docs/routers/asr9000/software/bgp/configuration/guide/b-bgp-cg-asr9000.html)
