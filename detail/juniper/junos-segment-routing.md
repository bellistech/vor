# JunOS Segment Routing — Source Routing Paradigm, TI-LFA Computation, and SRv6 Architecture

> *Segment Routing replaces per-hop signaling protocols (LDP, RSVP-TE) with source-routed label stacks distributed by the IGP. Each node advertises segment identifiers (SIDs) for its prefixes and adjacencies, and the ingress node encodes the complete forwarding path as an ordered list of SIDs. This eliminates per-flow state in the core, enables topology-independent protection, and provides the foundation for SDN-controlled traffic engineering.*

---

## 1. The Source Routing Paradigm

### Traditional vs Segment Routing

```
Traditional MPLS (LDP/RSVP):
  - Each router independently decides the next hop
  - LDP: distributed label binding, follows IGP shortest path
  - RSVP-TE: per-tunnel signaling, per-flow state at every hop
  - Core routers maintain per-flow state (thousands of LSPs)

  Ingress PE → P1 → P2 → P3 → Egress PE
  (LDP binding)  (LDP binding)  (LDP binding)
  State at P1: knows about this flow
  State at P2: knows about this flow
  State at P3: knows about this flow

Segment Routing:
  - Ingress router encodes the ENTIRE path in the packet header
  - Core routers just follow instructions (SID stack)
  - No per-flow state anywhere except ingress
  - IGP distributes SIDs (no LDP, no RSVP signaling)

  Ingress PE → P1 → P2 → P3 → Egress PE
  Label stack: [SID-P2, SID-P3, SID-Egress]
  State at P1: only knows SID-P2 → forward to P2 (no flow state)
  State at P2: only knows SID-P3 → forward to P3
  State at P3: only knows SID-Egress → forward to Egress
```

### State Reduction

In a network with N nodes and F traffic flows:

```
LDP state:    O(N * N) label bindings (full mesh of prefix labels)
RSVP-TE:      O(F) tunnel states per transit node
Segment Routing: O(N) prefix-SIDs + O(adjacencies) adj-SIDs per node

For a 100-node SP core with 10,000 traffic flows:
  LDP:     ~10,000 label bindings per node
  RSVP-TE: ~10,000 tunnel states per transit node
  SR:      ~100 prefix-SIDs + ~4-8 adj-SIDs per node ≈ 108 entries
```

This 100x reduction in state is the primary scaling advantage of SR.

---

## 2. MPLS Label Allocation for SR

### SRGB (Segment Routing Global Block)

The SRGB is a contiguous range of MPLS labels reserved for prefix-SIDs. Each node in the SR domain should configure the same SRGB for operational simplicity (though different SRGBs are supported — the IGP handles translation).

```
Default SRGB: 16000-23999 (8000 labels)

Label calculation:
  Label(prefix-SID) = SRGB_base + SID_index

Example:
  SRGB base = 16000
  Node R1: SID index 1 → Label 16001
  Node R2: SID index 2 → Label 16002
  Node R3: SID index 3 → Label 16003

When SRGB differs between nodes:
  R1 (SRGB 16000-23999): R2's SID = 16000 + 2 = 16002
  R3 (SRGB 20000-27999): R2's SID = 20000 + 2 = 20002
  IGP advertisements include the SRGB range, so each node
  computes the correct label for each peer's SRGB
```

### SRLB (Segment Routing Local Block)

The SRLB is a label range for locally significant SIDs (adjacency-SIDs):

```
SRLB: 100000-100999 (default varies by platform)

Adjacency-SIDs are local to the allocating node:
  R1 ge-0/0/0 → R2: adj-SID 100001 (only meaningful at R1)
  R1 ge-0/0/1 → R3: adj-SID 100002 (only meaningful at R1)
  R2 ge-0/0/0 → R1: adj-SID 100001 (R2's local space — different from R1's)
```

### Label Stack Operations

```
Prefix-SID behavior at each hop:

PHP (Penultimate Hop Popping):
  Penultimate router pops the top label before forwarding to destination
  This is the DEFAULT behavior for prefix-SIDs in JunOS

  Stack: [16003, 16005] at R1
  R1 → R2: swap 16003 → (R2's label for next hop), forward
  R4 → R5: pop 16005 (PHP), forward as IP

No-PHP (Explicit-Null):
  Set with explicit-null flag on the SID advertisement
  Useful when egress node needs to see the label (e.g., for CoS)
```

---

## 3. TI-LFA Computation Algorithm

### The Problem with Traditional LFA

Loop-Free Alternate (RFC 5286) computes backup next-hops that avoid the failed link. But traditional LFA cannot protect all topologies — some failures have no loop-free alternate via a single next-hop.

```
Example where LFA fails:

     R1 ──── R2 ──── R3
      │               │
      R4 ──── R5 ──── R6

R1 → R3 primary path: R1-R2-R3
If R1-R2 fails:
  LFA checks: can R4 reach R3 without going through R1?
  R4 path: R4-R5-R6-R3 — does not go through R1 ✓
  LFA works here.

But in ring topology:
  R1 ── R2 ── R3
  │              │
  R6 ── R5 ── R4

R1 → R3 via R2. If R2 fails:
  R6 path to R3: R6-R5-R4-R3 — OK
  But R6 might also route R6-R1-R2-R3 (loop through R1)
  LFA may not find a valid alternate.
```

### TI-LFA Algorithm

TI-LFA (Topology-Independent LFA) solves this by computing backup paths using segment routing label stacks. The algorithm:

```
Step 1: Compute post-convergence SPF
  Run SPF on the topology WITH the protected element removed
  (link failure: remove the link; node failure: remove the node)
  Result: post-convergence shortest path tree

Step 2: Find the P-node (Point of Local Repair)
  The P-node is the node on the post-convergence path that is
  reachable from the PLR (Point of Local Repair) without traversing
  the failed element.

  P-node = first node on post-convergence path reachable from PLR

Step 3: Find the Q-node
  The Q-node is the node on the post-convergence path from which
  the destination is reachable on the post-convergence tree.

  Q-node = last node on post-convergence path before destination
  that is in the PLR's post-convergence SPF tree

Step 4: Construct repair label stack
  If P-node == Q-node:
    Stack = [prefix-SID of P/Q-node]
  If P-node != Q-node:
    Stack = [prefix-SID of P-node, adj-SID from Q-node toward destination]
  If P-node not found (rare, complex topology):
    Stack = [adj-SID sequence to reach Q-node]
```

### TI-LFA Worked Example

```
Topology:
            10          10          10
  R1 ────────── R2 ────────── R3 ────────── R4
  │                                         │
  │ 10                                      │ 10
  │                                         │
  R5 ────────────────── R6 ─────────────── R7
            20                    20

Prefix: R4 loopback (SID index 4)
Primary path from R1: R1→R2→R3→R4 (cost 30)
Protected link: R1→R2

Step 1: Post-convergence SPF (remove R1-R2 link)
  R1→R4 via: R1→R5→R6→R7→R4 (cost 60)

Step 2: P-node
  From R1 (PLR), reachable without R1-R2: R5
  R5 is on the post-convergence path

Step 3: Q-node
  R7 is the last node on post-convergence path before R4

Step 4: Repair stack
  P-node (R5) ≠ Q-node (R7)
  Stack: [prefix-SID(R5)=16005, prefix-SID(R7)=16007, prefix-SID(R4)=16004]
  Or simplified if R5→R7 path is deterministic:
  Stack: [prefix-SID(R7)=16007] (if R5 routes to R7 without R1-R2)

Pre-installed backup next-hop at R1:
  For destination R4: via R5, push label stack [16007, 16004]
```

### Coverage Analysis

TI-LFA provides 100% coverage for any single link or node failure in any topology, given:
1. The network is 2-connected (has at least 2 disjoint paths)
2. Segment routing is enabled on all nodes
3. Prefix-SIDs and adjacency-SIDs are available

```
Coverage comparison:
  LFA (RFC 5286):           ~40-80% coverage depending on topology
  Remote LFA (RFC 7490):    ~90-95% coverage
  TI-LFA:                   100% coverage (topology-independent)
```

---

## 4. Flex-Algo Constraints

### Flex-Algo Concept

Flex-algo allows multiple SPF computations on the same topology, each with different constraints:

```
Algorithm 0: Default IGP SPF (standard shortest path)
Algorithms 128-255: User-defined flex-algos

Each flex-algo defines:
  1. Metric type: IGP metric, TE metric, or delay metric
  2. Constraints: admin-group include/exclude, SRLG avoidance
  3. Calculation type: SPF (default)
```

### Flex-Algo Participation

```
Definition: One or more nodes DEFINE the flex-algo (algorithm parameters)
Participation: Nodes that SUPPORT the flex-algo compute separate SPF

Node roles:
  Definer: advertises flex-algo definition in IGP
  Participant: has a prefix-SID for the algo + supports the constraints
  Non-participant: ignores the algo, does not install algo-specific routes

If a node does not participate in algo 128:
  - It does not advertise a SID for algo 128
  - It is excluded from algo 128's topology
  - Traffic using algo 128 is routed AROUND this node
```

### Flex-Algo Use Cases

```
Algorithm 128: Low-latency path
  metric-type: min-unidirectional-link-delay
  constraint: include-any admin-group LOW-LATENCY
  Use: VoIP, real-time trading

Algorithm 129: High-bandwidth path
  metric-type: te-metric (where TE metric reflects available BW)
  constraint: include-any admin-group HIGH-BW
  Use: video streaming, bulk transfer

Algorithm 130: Disjoint path (avoiding specific SRLGs)
  metric-type: igp
  constraint: exclude-srlg FIBER-BUNDLE-A
  Use: diverse routing for resilience
```

### Flex-Algo and Prefix-SIDs

Each flex-algo gets its own prefix-SID per node:

```
Node R1:
  Algorithm 0:   SID index 1   → Label 16001
  Algorithm 128: SID index 101 → Label 16101
  Algorithm 129: SID index 201 → Label 16201

Traffic to R1 via low-latency path: push label 16101
Traffic to R1 via high-BW path: push label 16201
Traffic to R1 via default IGP: push label 16001
```

---

## 5. SRv6 Implementation in JunOS

### SRv6 Addressing Architecture

```
SRv6 SID format:
┌──────────────────────────────────────────────────┐
│          Locator           │     Function         │
│  (identifies the node)     │  (identifies action)  │
├────────────────────────────┼─────────────────────┤
│  2001:db8:1::/48           │  ::1 = End           │
│  2001:db8:1::/48           │  ::10 = End.DT4      │
│  2001:db8:1::/48           │  ::20 = End.DT6      │
└────────────────────────────┴─────────────────────┘

Full SID: 2001:db8:1::1 (locator + function)
         ├── Locator: 2001:db8:1::/48 (routable IPv6 prefix)
         └── Function: ::1 (End behavior)
```

### SRv6 Packet Format

```
Original IPv6 packet:
┌────────────┬─────────┐
│ IPv6 Header│ Payload │
└────────────┴─────────┘

SRv6 encapsulated packet:
┌──────────────────┬───────────────────┬────────────┬─────────┐
│ Outer IPv6 Header│ SRH (Segment      │ Inner IPv6 │ Payload │
│ DA = active SID  │  Routing Header)  │ Header     │         │
│ SA = source      │ SID list + ptr    │ (original) │         │
└──────────────────┴───────────────────┘────────────┴─────────┘

SRH fields:
  Segments Left (SL): pointer to current active SID in the list
  SID List: ordered list of SIDs (bottom-up, SID[0] = last hop)

Processing:
  1. Forward to DA (current active SID)
  2. At SID endpoint: decrement SL, update DA to SID[SL]
  3. Process Function (End, End.DT4, etc.)
  4. Forward to new DA
```

### SRv6 vs SR-MPLS Comparison

| Aspect | SR-MPLS | SRv6 |
|:---|:---|:---|
| SID encoding | MPLS label (20 bits) | IPv6 address (128 bits) |
| SID capacity | 2^20 (~1M) per SRGB | 2^128 per locator |
| Stack depth | Limited by MTU (~10-15 labels) | Limited by MTU (~5-8 SIDs) |
| SID overhead | 4 bytes per label | 16 bytes per SID |
| Forwarding plane | MPLS | IPv6 |
| Network dependencies | MPLS-enabled core | IPv6-enabled core |
| Hardware support | Universal | Newer platforms required |
| Service functions | Limited (VPN labels) | Rich (End.DT4, End.DT6, End.B6...) |
| Network programming | Label stacking | SRH + functions |

### SRv6 Micro-SID

Micro-SID compresses SRv6 SIDs to reduce overhead:

```
Standard SRv6: 16 bytes per SID
Micro-SID:     Packs multiple SIDs into a single 128-bit container

Format: Locator (32 bits) + uSID1 (16 bits) + uSID2 (16 bits) + ... + uSID6 (16 bits)

Example:
  fcbb:bb01:0100:0200:0300:0400::
  ├── Locator block: fcbb:bb01 (32 bits)
  ├── uSID1: 0100 (node 1)
  ├── uSID2: 0200 (node 2)
  ├── uSID3: 0300 (node 3)
  ├── uSID4: 0400 (node 4)
  └── Remaining: padding

Result: 4 SIDs in 16 bytes instead of 64 bytes
```

---

## 6. SR vs LDP/RSVP Comparison

### Feature Comparison

| Feature | LDP | RSVP-TE | SR-MPLS |
|:---|:---|:---|:---|
| Signaling protocol | LDP (TCP 646) | RSVP (protocol 46) | None (IGP extension) |
| Path computation | Follows IGP SPF | Explicit (CSPF or manual) | IGP SPF + SR-TE |
| Per-flow state | No | Yes (per LSP) | No |
| Fast reroute | LFA/RLFA | Facility backup | TI-LFA (100% coverage) |
| Traffic engineering | No (follows IGP) | Yes (full TE) | Yes (SR-TE policies) |
| Bandwidth reservation | No | Yes (RSVP admission) | No (but policers/CoS) |
| Control plane overhead | Moderate | High (soft state refresh) | Low (IGP only) |
| Scaling | Good | Poor (state * transit nodes) | Excellent |
| SDN compatibility | Poor | Moderate | Excellent |
| Migration from | N/A | Complex (per-tunnel) | Easy (coexistence) |

### Migration Path

```
Phase 1: Enable SR alongside LDP
  - Configure SRGB and prefix-SIDs
  - Both LDP and SR labels installed in inet.3
  - LDP preferred by default (lower preference)
  - No traffic impact

Phase 2: Prefer SR over LDP
  - Set SR preference lower than LDP
  - Traffic shifts to SR-MPLS labels
  - LDP still active as backup
  - Verify SR forwarding with traceroute mpls

Phase 3: Remove LDP
  - Disable LDP on all interfaces
  - Remove LDP configuration
  - SR-MPLS is sole label distribution
  - Monitor for any LDP-dependent features (L2VPN, etc.)
```

---

## 7. SR Scaling Advantages

### Control Plane Scaling

```
Network: N nodes, A average adjacencies per node, F traffic flows

LDP control plane:
  Label bindings: N * N = N^2 per node (for all prefixes)
  Sessions: A per node (one per LDP neighbor)
  Total state: O(N^2 * N) across network = O(N^3)

RSVP-TE control plane:
  LSP state: F per transit node
  Refresh messages: F * refresh_rate per transit node
  Total state: O(F * N) across network
  With 10,000 flows and 100 nodes: 1,000,000 states

SR control plane:
  SID advertisements: N prefix-SIDs + A adj-SIDs per node
  Carried in existing IGP LSAs/LSPs (no additional sessions)
  Total state: O(N + A) per node
  With 100 nodes, 4 adj each: ~500 SIDs total
```

### Data Plane Scaling

```
LDP forwarding table:
  One label per prefix per node = O(N) entries
  No per-flow entries

RSVP-TE forwarding table:
  One entry per transit LSP = O(F) entries
  Scales linearly with traffic flows
  10,000 LSPs = 10,000 forwarding entries per transit node

SR forwarding table:
  One entry per prefix-SID = O(N) entries
  One entry per adj-SID = O(A) entries
  Total: O(N + A) entries, independent of flow count
  100 nodes: ~108 forwarding entries per node
```

## Prerequisites

- MPLS label switching fundamentals, IGP operation (IS-IS/OSPF), IPv6 addressing, traffic engineering concepts, LFA/FRR fundamentals

## Complexity

| Operation | Time | Space |
|-----------|------|-------|
| Prefix-SID label lookup | O(1) | O(N) per node |
| TI-LFA backup computation | O(N log N) per protected link | O(N) per backup path |
| Flex-algo SPF computation | O(N log N) per algorithm | O(N) per algorithm |
| SRv6 SRH processing | O(segments_left) per hop | O(SID_list_length) per packet |
| SR policy installation | O(1) | O(segment_list_length) |

---

*Segment Routing represents a paradigm shift from distributed hop-by-hop signaling to centralized source routing. The elimination of per-flow state in the core, combined with topology-independent protection and native SDN compatibility, makes it the clear successor to LDP and RSVP-TE in modern SP networks. The transition from SR-MPLS to SRv6 further unifies the data plane by eliminating the MPLS layer entirely, though hardware maturity remains a gating factor.*
