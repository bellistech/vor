# JunOS IS-IS Advanced — Implementation Internals, SPF Computation, and SP Scaling

> *IS-IS on JunOS operates as a native CLNP-based link-state protocol with IPv4/IPv6 carried as TLV extensions. The JunOS implementation centers on rpd's IS-IS task, which maintains the LSDB, runs Dijkstra's SPF algorithm per level and per topology, and programs the resulting routes into the kernel RIB. Understanding SPF computation logging, convergence tuning parameters, multi-topology architecture, and IS-IS scaling characteristics is essential for operating large SP networks where IS-IS is the foundation for both IP routing and MPLS signaling.*

---

## 1. JunOS IS-IS Implementation

### rpd IS-IS Task Architecture

```
┌─────────────────────────────────────────────┐
│                    rpd                       │
│  ┌────────────────────────────────────────┐  │
│  │ IS-IS Task                             │  │
│  │  ┌──────────┐  ┌──────────────────┐   │  │
│  │  │ Adjacency │  │ LSDB             │   │  │
│  │  │ Manager   │  │ (per level)      │   │  │
│  │  │           │  │ ├── Own LSP      │   │  │
│  │  │ Hello TX  │  │ ├── Peer LSPs    │   │  │
│  │  │ Hello RX  │  │ └── Purge queue  │   │  │
│  │  └─────┬─────┘  └────────┬─────────┘   │  │
│  │        │                 │              │  │
│  │  ┌─────┴─────────────────┴───────────┐  │  │
│  │  │ SPF Computation Engine             │  │  │
│  │  │ ├── Full SPF (topology change)     │  │  │
│  │  │ ├── PRC (prefix change only)       │  │  │
│  │  │ └── iSPF (incremental, future)     │  │  │
│  │  └─────────────────┬─────────────────┘  │  │
│  │                    │                     │  │
│  │  ┌─────────────────┴─────────────────┐  │  │
│  │  │ RIB Manager                        │  │  │
│  │  │ ├── Install routes in inet.0/inet6 │  │  │
│  │  │ ├── Install backup paths (TI-LFA)  │  │  │
│  │  │ └── Push FIB to PFE               │  │  │
│  │  └───────────────────────────────────┘  │  │
│  └────────────────────────────────────────┘  │
└─────────────────────────────────────────────┘
```

### PDU Types and Processing

```
PDU Type    Name                    Purpose
─────────────────────────────────────────────────────────
IIH         IS-IS Hello             Adjacency discovery/maintenance
LSP         Link State PDU          Topology and reachability info
CSNP        Complete Seq Number PDU Database synchronization (full list)
PSNP        Partial Seq Number PDU  Request/acknowledge specific LSPs

Processing pipeline:
  IIH received → adjacency state machine update
  LSP received → LSDB update → SPF trigger (if topology change)
  CSNP received → compare with local LSDB → request missing LSPs via PSNP
  PSNP received → retransmit requested LSPs
```

### Adjacency State Machine

```
States: Down → Init → Up

P2P adjacency (three-way handshake, RFC 5303):
  Down: no IIH received
  Init: IIH received, but neighbor not yet reporting our system-id
  Up:   IIH received with our system-id in the neighbor's "adjacency" TLV

  R1 → IIH (no neighbor) → R2
  R2 → IIH (neighbor=R1) → R1
  R1 → IIH (neighbor=R2) → R2
  Both: Init → Up

LAN adjacency (DIS election):
  All routers on LAN segment send IIH with priority
  Highest priority (then highest SNPA/MAC) becomes DIS
  DIS sends CSNP every 10 seconds for database sync
  Non-DIS routers form adjacency with DIS pseudonode
```

---

## 2. SPF Computation Logging

### Reading SPF Logs

```
show isis spf log

IS-IS level 2 SPF log:
  Start time           Duration  Delay    Reason
  ─────────────────────────────────────────────────
  2026-01-15 10:23:45  12ms      200ms    Link state change (ge-0/0/0.0)
  2026-01-15 10:24:01  8ms       200ms    Prefix add (10.0.1.0/24)
  2026-01-15 10:30:00  45ms      5000ms   Periodic (LSP refresh)

Fields:
  Start time: when SPF computation began
  Duration:   wall-clock time for SPF + route installation
  Delay:      time between trigger and SPF start (throttle delay)
  Reason:     what triggered SPF
```

### SPF Trigger Types

```
Full SPF triggers (recompute entire shortest path tree):
  - Adjacency up/down
  - TE metric change on any link
  - IS-IS metric change on any link
  - New node (LSP from unknown system-id)
  - Node departure (LSP purge)
  - Topology TLV change

PRC triggers (only recompute affected prefixes, not SPF tree):
  - Prefix add/delete/change (IP reachability TLV)
  - Prefix metric change (without topology change)
  - Tag/attribute change on prefix

PRC is much faster than full SPF because it skips the Dijkstra
computation and only updates the affected routes in the RIB.
```

### SPF Duration Breakdown

```
Total SPF time = Dijkstra computation + route calculation + RIB update + FIB push

For a 500-node network:
  Dijkstra:        5-15ms   (O(N log N) with binary heap)
  Route calc:      5-10ms   (prefix-to-next-hop mapping)
  RIB update:      5-20ms   (install/update/delete routes)
  FIB push:        10-50ms  (program PFE forwarding table)
  Total:           25-95ms

For a 2000-node network:
  Dijkstra:        20-60ms
  Route calc:      15-30ms
  RIB update:      20-50ms
  FIB push:        30-100ms
  Total:           85-240ms
```

---

## 3. Convergence Tuning in JunOS

### Convergence Time Components

```
Total convergence time:
  T_converge = T_detect + T_propagate + T_compute + T_program

  T_detect:     Time to detect failure
                - Physical: <50ms (LOL/LOS)
                - BFD: interval * multiplier (e.g., 300ms * 3 = 900ms)
                - IS-IS hello: hold-time (e.g., 3 * hello-interval)

  T_propagate:  Time for LSP to flood across the network
                - Depends on network diameter and flooding delay
                - Typically 10-100ms for 10-hop network

  T_compute:    Time for SPF computation
                - Depends on LSDB size and SPF throttle settings
                - Initial SPF delay + computation time

  T_program:    Time to update PFE forwarding table
                - Platform dependent: 10-100ms
```

### SPF Throttle Parameters

```
set protocols isis spf-options delay 200        # T_initial: ms before first SPF
set protocols isis spf-options holddown 5000    # T_holddown: min ms between SPFs
set protocols isis spf-options rapid-runs 3     # N_rapid: rapid SPFs before holddown

Timeline during link flap:

T=0:      Link goes down → trigger
T+200ms:  First SPF runs (initial delay)
T+400ms:  Link comes back up → trigger
T+600ms:  Second SPF runs (rapid run #2, 200ms delay)
T+800ms:  Link goes down again → trigger
T+1000ms: Third SPF runs (rapid run #3, 200ms delay)
T+1200ms: Link up again → trigger
T+6000ms: Fourth SPF runs (holddown: 1000ms + 5000ms)
          Subsequent SPFs: at holddown intervals

Aggressive tuning (fast convergence, higher CPU):
  delay: 50, holddown: 2000, rapid-runs: 5

Conservative tuning (lower CPU, slower convergence):
  delay: 500, holddown: 10000, rapid-runs: 2
```

### LSP Generation Throttle

```
When local topology changes, this router must generate a new LSP:

set protocols isis lsp-generation-interval 10   # min seconds between own LSP updates

If multiple local events happen in quick succession:
  - First event: generate LSP immediately (or after small delay)
  - Subsequent events within interval: buffer and generate one LSP
  - Reduces flooding storms during instability
```

### Overload-on-Startup

```
set protocols isis overload timeout 300

Timeline after router boot:
  T=0:      Router boots, IS-IS starts
  T=0-300s: Overload bit SET in own LSP
            → Other routers avoid using this router for transit
            → This router's own prefixes are reachable
            → Routing tables have time to fully converge
  T=300s:   Overload bit CLEARED
            → Router becomes eligible for transit traffic
            → Prevents blackholing during partial convergence
```

### Prefix Convergence Optimization

```
# JunOS can prioritize critical prefix updates in RIB installation:
set protocols isis prefix-export-limit 500     # max prefixes exported per cycle

# Install loopback (infrastructure) prefixes before customer prefixes:
# This ensures IBGP/LDP next-hops resolve before VPN routes install
# JunOS handles this implicitly — /32 loopbacks install first in SPF order
```

---

## 4. Multi-Topology IS-IS Architecture

### Why Multi-Topology

In single-topology IS-IS, IPv4 and IPv6 share the same SPF tree. Every interface must support both protocols, or the SPF tree becomes disconnected for one protocol:

```
Single topology problem:

  R1 ──── R2 ──── R3
  (v4+v6)  (v4+v6)  (v4+v6)
  │
  R4 ──── R5
  (v4+v6)  (v4 only) ← problem!

R5 does not support IPv6
In single topology: R5 is still in the SPF tree
IPv6 traffic routed through R5 → BLACKHOLED

With multi-topology:
  MT0 (IPv4): R1-R2-R3, R1-R4-R5 (R5 participates)
  MT2 (IPv6): R1-R2-R3, R1-R4 (R5 excluded from IPv6 SPF)
  IPv6 traffic avoids R5 → correct routing
```

### MT TLV Encoding

```
IS-IS Multi-Topology TLVs (RFC 5120):

MT-ID values:
  0: Default (IPv4 unicast)
  2: IPv6 unicast
  3: IPv4 multicast
  4: IPv6 multicast

TLV 229 (MT IS Reachability):
  Carries neighbor adjacency info per topology
  Includes metric per topology per neighbor

TLV 235 (MT IP Reachability):
  Carries IPv4 prefix info with MT-ID

TLV 237 (MT IPv6 Reachability):
  Carries IPv6 prefix info with MT-ID
```

### MT SPF Computation

```
With multi-topology enabled, JunOS runs separate SPF for each topology:

SPF run 1: MT-ID 0 (IPv4)
  Input: TLV 229 entries with MT-ID 0, TLV 235 prefixes
  Output: IPv4 next-hops → inet.0

SPF run 2: MT-ID 2 (IPv6)
  Input: TLV 229 entries with MT-ID 2, TLV 237 prefixes
  Output: IPv6 next-hops → inet6.0

SPF computation time doubles (two independent SPF runs)
But routing correctness is guaranteed for mixed-protocol deployments
```

---

## 5. IS-IS for Segment Routing

### SR TLVs in IS-IS

```
TLV/Sub-TLV                       Purpose
──────────────────────────────────────────────────────────
Router Capability TLV (242):
  Sub-TLV 2: SR Capability        SRGB range advertisement
  Sub-TLV 19: SR Algorithm        Supported algorithms (0, 128-255)
  Sub-TLV 22: SR Local Block      SRLB range advertisement

Extended IS Reachability (22):
  Sub-TLV 31: Adj-SID             Adjacency SID label
  Sub-TLV 32: LAN Adj-SID         LAN adjacency SID

Extended IP Reachability (135):
  Sub-TLV 3: Prefix-SID           Prefix SID index + flags

IPv6 Reachability (236):
  Sub-TLV 3: Prefix-SID           Same as IPv4 prefix-SID
```

### Prefix-SID Flags

```
Prefix-SID sub-TLV flags:
  R: Re-advertisement flag (node is not the originator)
  N: Node-SID flag (SID is for a node, not an anycast prefix)
  P: No-PHP flag (do not pop label on penultimate hop)
  E: Explicit-null flag (use explicit-null label)
  V: Value flag (SID is an absolute label, not an index)
  L: Local flag (SID has local significance)

Common combinations:
  Node-SID (loopback):     N=1, R=0, P=0, E=0 (PHP, index-based)
  Anycast-SID:             N=0, R=0, P=0 (shared across multiple nodes)
  Node-SID (no-PHP):       N=1, P=1, E=1 (explicit-null for CoS)
```

### SR SPF Computation

```
SR adds to IS-IS SPF computation:

Standard SPF output:
  Destination → Next-hop → Outgoing interface → IGP metric

SR-enhanced SPF output:
  Destination → Next-hop → Outgoing interface → IGP metric → Label operation

Label operation determination:
  If next-hop is penultimate hop for destination:
    PHP (pop label) — unless no-PHP flag set
  If next-hop is transit hop:
    SWAP (swap incoming label → next-hop's label for destination)
  If this node is ingress:
    PUSH (push label = SRGB_base + SID_index)
```

---

## 6. IS-IS Scaling in SP Networks

### LSDB Size and Memory

```
LSDB memory consumption per node:

Per-LSP overhead:
  LSP header: 27 bytes
  Average TLV payload: 200-2000 bytes (depends on adjacencies + prefixes)
  Per-node average: ~500 bytes for core router, ~2000 bytes for PE with many prefixes

Total LSDB size for N-node network:
  L2 LSDB ≈ N * avg_LSP_size

  500-node network: 500 * 1KB = 500 KB (trivial)
  5000-node network: 5000 * 1KB = 5 MB (still manageable)
  50000-node network: 50000 * 1KB = 50 MB (requires careful design)
```

### SPF Computation Scaling

```
Dijkstra's algorithm complexity: O((N + E) log N)
  N = nodes, E = edges (adjacencies)

Practical SPF times on modern JunOS platforms:
  100 nodes:    < 5ms
  500 nodes:    5-15ms
  1000 nodes:   10-30ms
  5000 nodes:   50-200ms
  10000 nodes:  200-1000ms (consider area/level design)
```

### Flooding Scaling

```
LSP flooding overhead:
  Each topology change → LSP update → flood to all neighbors → each neighbor floods further

Flooding per event:
  Worst case: N * avg_adjacencies LSP copies across network
  Mesh groups reduce: N * (avg_adjacencies - mesh_group_size + 1)

Flooding storms:
  Multiple simultaneous changes → multiple LSPs → flooding burst
  LSP generation throttle prevents: rapid-fire LSP generation
  SPF throttle prevents: CPU exhaustion from repeated SPF runs

Scale limits:
  - IS-IS comfortably handles 1000+ nodes per level
  - Beyond 2000-3000 nodes: consider splitting into areas/levels
  - L1 areas contain flooding domains, reducing per-node flooding load
```

### Area Design for Scale

```
Small SP (< 500 nodes):
  Single L2 domain, all routers L2-only
  Simple, no route leaking needed

Medium SP (500-2000 nodes):
  Core: L2 only (P routers, RRs)
  Edge: L1/L2 (PEs, aggregation)
  Access: L1 only (access routers)
  Route leaking: selective prefixes L2→L1

Large SP (2000+ nodes):
  Multiple L1 areas (regional)
  L2 backbone (inter-area)
  Route summarization at L1/L2 boundaries
  OR: flatten with single L2 + mesh groups + flooding optimization

IS-IS vs OSPF scaling comparison:
  IS-IS advantages:
  - TLV-based: extensible without protocol version changes
  - Level hierarchy simpler than OSPF area types
  - No LSA type complexity (no stub/NSSA area semantics)
  - Better flooding efficiency on P2P links (no DR/BDR overhead)
  - Native support for multi-topology
```

### Prefix Scaling

```
Per-node prefix advertisement:
  Core router: 1-5 prefixes (loopback + infrastructure)
  PE router: 10-100 prefixes (customer routes leaked into IS-IS)
  Access router: 1-10 prefixes

Total prefixes in LSDB:
  1000-node network with mix: ~5000-20000 prefixes
  PRC handles prefix changes without full SPF
  SPF only runs for topology changes (link/node up/down)

Optimization:
  - Summarize access prefixes at aggregation boundary
  - Avoid redistributing BGP into IS-IS (use BGP for internet routes)
  - Use L1 areas to contain prefix flooding to relevant domains
```

---

## 7. IS-IS Operational Best Practices

### Production IS-IS Template

```
Typical SP core router IS-IS configuration:

set protocols isis level 1 disable
set protocols isis level 2 wide-metrics-only
set protocols isis level 2 authentication-key "ENCRYPTED"
set protocols isis level 2 authentication-type md5
set protocols isis traffic-engineering
set protocols isis source-packet-routing
set protocols isis source-packet-routing node-segment ipv4-index <UNIQUE>
set protocols isis source-packet-routing srgb start-label 16000 index-range 8000
set protocols isis overload timeout 300
set protocols isis graceful-restart
set protocols isis spf-options delay 200
set protocols isis spf-options holddown 5000
set protocols isis spf-options rapid-runs 3
set protocols isis purge-originator-identification
set protocols isis backup-spf-options use-post-convergence-lfa
set protocols isis level 2 post-convergence-lfa node-protection

set protocols isis interface lo0.0 passive
set protocols isis interface ge-0/0/0.0 point-to-point
set protocols isis interface ge-0/0/0.0 level 2 metric <COST>
set protocols isis interface ge-0/0/0.0 bfd-liveness-detection minimum-interval 300
set protocols isis interface ge-0/0/0.0 bfd-liveness-detection multiplier 3
```

### Monitoring and Alerting

```
Key metrics to monitor:
  - SPF run count (show isis spf log): >10/min indicates instability
  - SPF duration: >100ms may indicate oversized LSDB
  - Adjacency flaps (show isis adjacency): any flap needs investigation
  - LSDB size (show isis database | count): unexpected growth = leak
  - Authentication failures (show isis statistics): indicates misconfiguration or attack
  - Overload bit state: unexpected overload = problem
```

## Prerequisites

- IS-IS fundamentals (PDU types, adjacency formation, LSDB), SPF algorithm (Dijkstra), MPLS label switching, IPv6 addressing, TLV encoding concepts

## Complexity

| Operation | Time | Space |
|-----------|------|-------|
| SPF computation (Dijkstra) | O((N + E) log N) | O(N + E) |
| PRC (prefix-only recomputation) | O(affected_prefixes) | O(1) per prefix |
| LSP flooding (per event) | O(N * avg_degree) | O(LSP_size) per copy |
| LSDB lookup | O(1) avg (hash by system-id) | O(N * avg_LSP_size) |
| TI-LFA backup computation | O(N log N) per protected element | O(N) per backup |
| Multi-topology SPF | O(topologies * (N + E) log N) | O(topologies * (N + E)) |

---

*IS-IS is the IGP of choice for large SP networks because of its extensibility (TLVs accommodate new features without protocol changes), its level hierarchy (simpler than OSPF area types), and its natural affinity for segment routing (IS-IS SR extensions are the most mature). The key to operating IS-IS at scale is understanding SPF computation frequency, controlling flooding with appropriate throttles and mesh groups, and designing level/area boundaries that balance routing precision against database size.*
