# Advanced OSPF Theory — SPF Algorithms, LSA Mechanics, and Convergence Optimization

> *OSPF is Dijkstra's algorithm running distributed across every router, with a flooding protocol to synchronize state, an area hierarchy to bound complexity, and a collection of convergence optimizations refined over three decades. This deep-dive covers the SPF algorithm with pseudocode, LSA flooding mechanics, external route computation, convergence tuning, MPLS VPN integration, and the LFA computation algorithm.*

---

## 1. OSPF SPF Algorithm — Pseudocode and Analysis

### Dijkstra's Algorithm Applied to OSPF

Each OSPF router builds a directed weighted graph from its Link-State Database (LSDB). The SPF algorithm computes the shortest path tree (SPT) rooted at the local router.

### Input

- Graph $G = (V, E)$ where:
  - $V$ = set of routers and transit networks (Type 1 and Type 2 LSAs)
  - $E$ = set of links with costs from Router LSAs
- Source vertex $s$ = local router

### Pseudocode

```
SPF(G, s):
    // Initialization
    for each vertex v in V:
        dist[v] = INFINITY
        prev[v] = NULL
        visited[v] = false

    dist[s] = 0
    Q = min-priority-queue containing all vertices

    // Main loop
    while Q is not empty:
        u = Q.extract-min()          // Vertex with minimum dist
        visited[u] = true

        for each neighbor v of u:
            if visited[v]:
                continue

            // Edge cost from OSPF perspective
            if u is a router and v is a transit network:
                w = interface cost from u's Router LSA
            else if u is a transit network and v is a router:
                w = 0    // Transit network to router cost is 0
            else if u is a router and v is a router (point-to-point):
                w = interface cost from u's Router LSA

            alt = dist[u] + w
            if alt < dist[v]:
                dist[v] = alt
                prev[v] = u
                Q.decrease-key(v, alt)
            else if alt == dist[v]:
                // ECMP: add u as additional parent of v
                add u to parents[v]

    return dist[], prev[], parents[]
```

### OSPF-Specific SPF Details

**Two-phase computation:**

1. **Phase 1 — Intra-area SPF**: Process Type 1 (Router) and Type 2 (Network) LSAs to build the SPT within the area. This produces routes to all routers and networks in the area.

2. **Phase 2 — Inter-area and external routes**: After the SPT is built, process Type 3 (Summary) and Type 5/7 (External) LSAs. These are not part of Dijkstra — they are simple distance-vector computations using the SPT results.

```
Phase 2 — Inter-area routes:
    for each Type 3 LSA (prefix P, advertising ABR, cost C):
        total_cost = dist[ABR] + C
        if total_cost < current_best[P]:
            install route to P via ABR with cost total_cost

Phase 2 — External routes:
    for each Type 5 LSA (prefix P, advertising ASBR, metric M, type E1/E2):
        if E1:
            total_cost = dist[ASBR] + M     // Internal + external
        if E2:
            total_cost = M                    // External only
            tiebreak = dist[ASBR]            // Use ASBR distance as tiebreak
        if total_cost < current_best[P]:
            install route to P via ASBR
```

### Complexity Analysis

| Implementation | Time Complexity | Space Complexity |
|:---|:---|:---|
| Array-based (naive) | $O(V^2)$ | $O(V)$ |
| Binary heap | $O((V + E) \log V)$ | $O(V)$ |
| Fibonacci heap | $O(V \log V + E)$ | $O(V)$ |

Modern router implementations use binary heap with optimizations. For a typical area with $V = 500$ routers and $E = 2000$ links:

$$T_{SPF} \approx (500 + 2000) \times \log_2(500) \approx 2500 \times 9 = 22{,}500 \text{ operations}$$

At $\sim 5\text{ns}$ per operation: $\sim 0.1\text{ms}$. Real-world SPF runs on 500-node areas complete in 1-5ms.

---

## 2. LSA Flooding and Aging

### Flooding Algorithm

When a router generates or receives a new LSA, it floods the LSA to all neighbors according to:

```
FloodLSA(lsa, receiving_interface):
    // Check if this LSA is newer than what we have
    existing = LSDB.lookup(lsa.type, lsa.id, lsa.adv_router)

    if existing is not NULL:
        if lsa.seq_num <= existing.seq_num:
            if lsa.seq_num == existing.seq_num:
                if lsa.checksum <= existing.checksum:
                    if lsa.age >= MaxAge AND existing.age < MaxAge:
                        // Accept MaxAge LSA for flushing
                    else:
                        // Duplicate or older — send back our copy
                        send existing to receiving_interface
                        return
        // Our copy is older — accept the new one

    // Install new LSA in LSDB
    LSDB.install(lsa)

    // Schedule SPF recalculation
    schedule_spf()

    // Flood to all interfaces EXCEPT receiving_interface
    for each interface I (I != receiving_interface):
        if I has at least one full adjacency:
            add lsa to I.retransmission_list
            send LSUpdate(lsa) on I
```

### LSA Sequence Numbers

LSA sequence numbers use a linear space with wrap-around protection:

$$\text{InitialSequenceNumber} = \texttt{0x80000001}$$
$$\text{MaxSequenceNumber} = \texttt{0x7FFFFFFF}$$

The sequence space is: $\texttt{0x80000001} \to \texttt{0x80000002} \to \ldots \to \texttt{0x7FFFFFFF}$

This provides $2^{31} - 1 \approx 2.1$ billion sequence numbers. At one LSA refresh per 30 minutes (1800 seconds), this space lasts:

$$T_{exhaust} = \frac{2^{31} - 1}{1} \times 1800 \text{ seconds} \approx 122{,}000 \text{ years}$$

### LSA Aging

Every LSA carries an age field (in seconds) that increments as the LSA propagates and ages in the LSDB:

- **LSA origination**: Age = 0
- **Transit delay**: Age += InfTransDelay (typically 1 second per hop)
- **LSDB aging**: Age increments by 1 every second
- **MaxAge**: 3600 seconds (1 hour) — LSA is flushed from LSDB
- **LSRefreshTime**: 1800 seconds (30 minutes) — originator must refresh before this

### LSA Refresh Overhead

For a network with $L$ LSAs in the LSDB, the refresh overhead is:

$$R_{refresh} = \frac{L}{1800} \text{ LSAs/second}$$

| LSDB Size | Refresh Rate | Updates/minute |
|:---:|:---:|:---:|
| 1,000 | 0.56 LSA/s | 33 |
| 5,000 | 2.78 LSA/s | 167 |
| 10,000 | 5.56 LSA/s | 333 |
| 50,000 | 27.78 LSA/s | 1,667 |

### LSA Pacing

To prevent burst flooding of refreshes, routers pace LSA transmissions:

```
LSA pacing groups refreshes into batches:
  - Group interval (default: 240 seconds on IOS)
  - All LSAs with age within the same group interval are refreshed together
  - Reduces the number of LSUpdate packets

Without pacing:  Individual LSUpdate per LSA refresh → many small packets
With pacing:     Batched LSUpdates every group interval → fewer large packets
```

---

## 3. External Route Computation — E1 vs E2 and Forwarding Address

### E1 vs E2 Route Selection

External routes come in two metric types:

**E2 (default)**: External metric only. Internal OSPF cost is used ONLY as a tiebreaker when two E2 routes have the same external metric.

$$\text{Cost}_{E2} = M_{external}$$
$$\text{Tiebreak}_{E2} = \text{OSPF cost to ASBR (forwarding address)}$$

**E1**: External metric PLUS internal OSPF cost. The full end-to-end cost is considered.

$$\text{Cost}_{E1} = M_{external} + \text{OSPF cost to ASBR (forwarding address)}$$

### Route Preference Hierarchy

When multiple external routes exist for the same prefix:

1. **Intra-area** (O) routes always preferred over inter-area
2. **Inter-area** (O IA) routes always preferred over external
3. **E1** routes preferred over E2 routes
4. Among E1: lowest total cost (external + internal) wins
5. Among E2: lowest external metric wins; if tied, lowest internal cost to ASBR wins

### Forwarding Address Selection

The forwarding address (FA) in Type 5/7 LSAs determines where traffic is sent:

```
Forwarding Address Selection Algorithm:

1. If ASBR's next-hop for the external route is on an OSPF-enabled
   broadcast/NBMA interface:
     FA = external next-hop IP address
     (Traffic goes directly to external destination, bypassing ASBR)

2. Otherwise:
     FA = 0.0.0.0
     (Traffic goes to ASBR; ASBR forwards to external destination)

Why FA matters:
  When FA != 0.0.0.0, OSPF routers compute cost to FA (not to ASBR)
  This produces optimal forwarding when multiple ASBRs share a subnet
  with the external next-hop
```

### Worked Example

```
  R1 ──(cost 10)── R2(ASBR) ──── External 10.0.0.0/8 (metric 20)
  R1 ──(cost 5)─── R3(ASBR) ──── External 10.0.0.0/8 (metric 30)

  E1 calculation from R1's perspective:
    Via R2: 20 + 10 = 30
    Via R3: 30 + 5  = 35
    Winner: R2 (cost 30)

  E2 calculation from R1's perspective:
    Via R2: metric 20, tiebreak cost 10
    Via R3: metric 30, tiebreak cost 5
    Winner: R2 (lower external metric 20)

  Now swap external metrics:
  R2: metric 30, R3: metric 30 (same)

  E2 with same metric:
    Via R2: metric 30, tiebreak cost 10
    Via R3: metric 30, tiebreak cost 5
    Winner: R3 (same metric, lower internal cost 5)
```

---

## 4. Inter-Area Path Selection

### ABR Path Selection

When an ABR advertises a Type 3 Summary LSA into a non-backbone area, it uses the cost from the backbone (Area 0). The inter-area routing rule is:

$$\text{Inter-area cost to prefix } P = \min_{ABR_i} (\text{OSPF cost to } ABR_i + \text{Type 3 cost from } ABR_i)$$

### The Split-Horizon Problem

ABRs must follow inter-area split-horizon rules:

1. An ABR MUST NOT advertise a Type 3 LSA learned from a non-backbone area back into another non-backbone area
2. All inter-area traffic MUST transit through Area 0
3. This prevents routing loops but can cause suboptimal routing when Area 0 has high-cost links

```
  Area 1 ──── ABR-A ──── Area 0 ──── ABR-B ──── Area 2
                                  (high cost)
  Area 1 ──── ABR-C ──── Area 0 ──── ABR-B ──── Area 2
                    (direct, low cost)

  Traffic from Area 1 to Area 2 MUST go through Area 0,
  even if a direct non-backbone path would be shorter.
  This is by design — it prevents routing loops.
```

### Virtual Link Impact on Path Selection

Virtual links logically extend Area 0 through a transit area. The virtual link cost equals the intra-area cost through the transit area:

$$\text{Virtual link cost} = \text{SPF cost through transit area between endpoints}$$

This cost is added to any route that traverses the virtual link, potentially making paths through the virtual link less preferred than direct Area 0 paths.

---

## 5. OSPF Convergence Optimization

### SPF Throttling

Modern OSPF implementations use exponential backoff for SPF scheduling:

```
SPF Throttling Parameters:
  spf-start    (initial delay):    typically 50ms
  spf-hold     (minimum interval): typically 200ms
  spf-max-wait (maximum interval): typically 5000ms

Behavior:
  First topology change:     SPF runs after spf-start (50ms)
  Second change within hold: SPF delayed by spf-hold (200ms)
  Subsequent rapid changes:  Delay doubles each time: 400, 800, 1600, ...
  Maximum delay:             Capped at spf-max-wait (5000ms)
  Quiet period:              After no changes for spf-max-wait, reset to spf-start
```

The delay for the $n$-th consecutive SPF trigger:

$$T_{delay}(n) = \min(\text{spf-start} \times 2^{n-1},\ \text{spf-max-wait})$$

### LSA Generation Throttling

Similar exponential backoff for LSA origination:

$$T_{lsa}(n) = \min(\text{lsa-start} \times 2^{n-1},\ \text{lsa-max-wait})$$

### LSA Pacing Optimization

LSA pacing groups refresh and flood transmissions:

- **Flood pacing**: Interval between consecutive LSUpdate packets (default 33ms on IOS)
- **Retransmission pacing**: Interval between retransmissions (default 66ms)
- **Group pacing**: Interval for batching LSA refreshes (default 240s)

### Incremental SPF (iSPF)

When only a leaf change occurs (e.g., stub network added/removed), full SPF recomputation is unnecessary. Incremental SPF only recalculates the affected portion of the tree:

$$T_{iSPF} = O(\Delta V + \Delta E) \quad \text{vs} \quad T_{full} = O((V + E) \log V)$$

Where $\Delta V$ and $\Delta E$ are the changed vertices and edges.

### Partial SPF (pSPF)

For Type 3/5/7 LSA changes (inter-area and external), only Phase 2 of the SPF needs to run — no Dijkstra recomputation required:

$$T_{pSPF} = O(L_{summary} + L_{external})$$

Where $L_{summary}$ and $L_{external}$ are the number of summary and external LSAs affected.

---

## 6. OSPF in MPLS VPN — Sham-Link Theory, DN Bit, and VPN Route Tag

### The MPLS VPN OSPF Problem

In MPLS L3VPN, the PE router runs OSPF with the CE and redistributes routes into MP-BGP for transport across the backbone. The receiving PE redistributes back into OSPF. This creates two problems:

1. **Route type change**: OSPF routes become external (Type 5) after double redistribution, losing their original intra-area or inter-area preference
2. **Routing loops**: If a CE has a backdoor link, the OSPF external route could be re-advertised back into BGP

### Solution 1: Domain ID and Route Type Preservation

```
When PE redistributes BGP → OSPF:
  If the BGP route originated from the same OSPF domain (matching Domain ID):
    Generate Type 3 Summary LSA (preserves inter-area semantics)
  If different OSPF domain:
    Generate Type 5 External LSA (normal redistribution)

Domain ID is carried in BGP Extended Community:
  Type: 0x0005 (OSPF Domain Identifier)
  Value: OSPF process/domain identifier
```

### Solution 2: DN Bit (Down Bit)

The DN bit prevents routing loops between OSPF and BGP:

```
DN Bit Behavior:
  1. PE generates Type 3 LSA from BGP route → sets DN bit
  2. CE receives Type 3 LSA with DN bit → installs route normally
  3. If CE re-advertises this LSA to another PE (backdoor):
     Second PE sees DN bit set → IGNORES the LSA
     (Prevents loop: PE→CE→CE→PE would create circular route)

DN Bit Position:
  In Type 3 LSA: Options field, bit 0x08 (DN)
  In Type 5/7 LSA: Options field, bit 0x08 (DN)
```

### Solution 3: VPN Route Tag

Additional loop prevention using a route tag:

```
VPN Route Tag Behavior:
  1. PE generates Type 5 LSA from BGP → sets tag = BGP AS number
  2. If another PE in same AS receives this Type 5 LSA via OSPF backdoor:
     PE checks tag against own BGP AS → match → IGNORES the route
```

### Solution 4: Sham Link

```
Problem scenario:
  CE-1 ─── PE-1 ═══(MPLS)═══ PE-2 ─── CE-2
    |                                    |
    └──── backdoor OSPF link (area 0) ───┘

  PE routes arrive as inter-area (Type 3) via superbackbone
  Backdoor routes are intra-area (Type 1/2) via direct OSPF
  OSPF always prefers intra-area → backdoor always wins
  MPLS VPN becomes backup only (not desired)

Sham link solution:
  Create a virtual OSPF intra-area link between PEs through the VPN
  This makes MPLS routes appear as intra-area
  Cost comparison then determines preferred path

Requirements:
  1. /32 loopback per VRF on each PE (separate from PE loopback)
  2. Loopback reachable via BGP (redistribute connected in VRF AF)
  3. Sham link cost < backdoor link cost (for MPLS to be preferred)
```

---

## 7. OSPFv3 Differences from OSPFv2

### Protocol-Level Changes

| Feature | OSPFv2 (RFC 2328) | OSPFv3 (RFC 5340) |
|:---|:---|:---|
| Network layer | IPv4 only | IPv6 native (IPv4 via AF) |
| Addressing | IPv4 addresses in packets | Link-local IPv6 addresses |
| Authentication | Built-in (MD5/simple) | IPsec AH/ESP (RFC 4552) |
| Link identification | IP address + mask | Interface ID (32-bit) |
| Flooding scope | Implicit (area/AS) | Explicit scope field in LSA header |
| LSA types | 1-7 + opaque | 1-7 + 8 (Link-LSA) + 9 (Intra-Area-Prefix) |
| Multi-instance | Limited | Full support (Instance ID field) |
| Unknown LSA handling | Drop | Store and flood (graceful extension) |

### New LSA Types in OSPFv3

**Link-LSA (Type 8)**:
- Flooding scope: link-local only
- Contains: router's link-local address, list of IPv6 prefixes on the link, router options
- Purpose: provide link-local address for next-hop, advertise prefixes for the link

**Intra-Area-Prefix-LSA (Type 9)**:
- Flooding scope: intra-area
- Contains: list of IPv6 prefixes associated with a router or transit network
- Purpose: decouple prefix information from topology (Router/Network LSAs no longer carry addresses)
- References the Router LSA or Network LSA it is associated with

### Addressing Model Difference

In OSPFv2, network addresses are embedded in Router and Network LSAs. In OSPFv3, addresses are completely separated:

```
OSPFv2 Router LSA contains:
  - Link Type, Link ID, Link Data (IP address), Metric

OSPFv3 Router LSA contains:
  - Link Type, Interface ID, Neighbor Interface ID, Neighbor Router ID, Metric
  (No IP addresses — those are in Intra-Area-Prefix LSAs)

This separation allows:
  1. Topology changes without prefix changes (and vice versa)
  2. Multiple address families on the same topology
  3. Fewer SPF runs (topology-only changes don't affect prefixes)
```

---

## 8. OSPF Graceful Restart Mechanics (RFC 3623)

### Protocol Mechanics

Graceful restart uses a Type 9 Opaque LSA (Grace-LSA) with link-local flooding scope:

```
Grace-LSA TLV Structure:
  TLV Type 1: Grace Period (seconds) — how long helpers should wait
  TLV Type 2: Graceful Restart Reason
    0 = Unknown
    1 = Software restart
    2 = Software reload/upgrade
    3 = Switch to redundant control processor
  TLV Type 3: IP Interface Address (identifies the restarting interface)
```

### State Machine

```
Restarting Router:                    Helper Router:
  1. Save forwarding table           1. Receive Grace-LSA
  2. Restart OSPF process            2. Enter helper mode for that neighbor
  3. Send Grace-LSA on all links     3. Continue advertising the neighbor's
  4. Re-form adjacencies                Router LSA as if neighbor is alive
  5. Rebuild LSDB from helpers       4. Do NOT run SPF to remove neighbor
  6. Run SPF                         5. Start grace timer
  7. Update forwarding table         6. If grace timer expires:
  8. Resume normal operation            Exit helper mode, run normal SPF
                                     7. If adjacency re-established before
                                        timer: exit helper mode gracefully
```

### Correctness Conditions

Graceful restart is correct (no forwarding loops) only if:

1. The network topology does NOT change during the restart period
2. All neighbors act as helpers (if even one does not, the restarting router may be bypassed)
3. The forwarding table remains valid throughout the restart

If topology changes during GR (e.g., a link fails elsewhere), the helper must abort:

$$\text{If } \exists \text{ LSA change (non-self)} \text{ during grace period} \implies \text{Abort GR, run SPF}$$

This is controlled by `strict-lsa-checking`. Without it, the helper continues even if topology changes, risking forwarding loops.

---

## 9. LFA Computation Algorithm (RFC 5286)

### Loop-Free Alternate Definition

For a source router $S$ protecting against failure of the link $S \to E$ (where $E$ is the primary next-hop toward destination $D$), a neighbor $N$ is a valid LFA if:

$$D_{opt}(N, D) < D_{opt}(N, S) + D_{opt}(S, D)$$

Where $D_{opt}(X, Y)$ is the shortest path distance from $X$ to $Y$.

This inequality ensures that $N$ does not route traffic back through $S$ — traffic forwarded to $N$ will reach $D$ without traversing the failed link.

### Protection Types

**Link protection** (protects against link $S \to E$ failure):

$$D_{opt}(N, D) < D_{opt}(N, S) + D_{opt}(S, D)$$

**Node protection** (protects against node $E$ failure):

$$D_{opt}(N, D) < D_{opt}(N, E) + D_{opt}(E, D)$$

Node protection is strictly stronger — a node-protecting LFA is always also link-protecting, but not vice versa.

**SRLG protection** (protects against shared risk link group failure):

$$\forall \text{ link } l \in SRLG(S \to E): D_{opt}(N, D) \text{ does not traverse } l$$

### LFA Computation Pseudocode

```
ComputeLFA(S, topology):
    // Step 1: Run Dijkstra from S to get dist_S[v] for all v
    dist_S = Dijkstra(topology, S)

    // Step 2: Run Dijkstra from every neighbor N of S
    for each neighbor N of S:
        dist_N = Dijkstra(topology, N)

    // Step 3: Run reverse Dijkstra from every neighbor
    //   (or equivalently, Dijkstra on reversed graph)
    //   to get dist_to_N[v] = dist(v, N) for all v

    // Step 4: For each destination D, find LFA
    for each destination D:
        primary_nh = next_hop(S, D)    // via SPF
        E = primary_nh                  // node to protect

        best_lfa = NULL
        for each neighbor N of S (N != E):
            // Link-protecting LFA check
            if dist_N[D] < dist_N[S] + dist_S[D]:
                // Valid link-protecting LFA
                if dist_N[D] < dist_N[E] + dist_E[D]:
                    // Also node-protecting — prefer this
                    best_lfa = N (node-protecting)
                else if best_lfa is NULL:
                    best_lfa = N (link-protecting only)

        if best_lfa is not NULL:
            install_backup(D, best_lfa)
```

### LFA Coverage Analysis

LFA coverage depends heavily on topology. For common topologies:

| Topology | Link Protection Coverage | Node Protection Coverage |
|:---|:---:|:---:|
| Ring (N nodes) | 100% | 0% |
| Full mesh | 100% | 100% |
| Square grid | ~80% | ~50% |
| Typical SP core | 70-90% | 40-70% |
| Hub-and-spoke | 0% (spokes) | 0% (spokes) |

### Remote LFA (rLFA) Extension

When no direct LFA exists, rLFA finds a "PQ node" $P$ such that:

$$D_{opt}(P, D) < D_{opt}(P, S) + D_{opt}(S, D) \quad \text{(P-space condition)}$$

AND $P$ is reachable from $S$ without traversing the failed link (Q-space condition). Traffic is tunneled from $S$ to $P$ using LDP or SR, then forwarded normally from $P$ to $D$.

### TI-LFA with Segment Routing

TI-LFA (Topology-Independent LFA) achieves 100% coverage by computing the post-convergence shortest path and encoding it as an SR label stack:

```
TI-LFA computation:
  1. Remove failed link/node from topology
  2. Run SPF on the modified topology (post-convergence SPF)
  3. Compute the post-convergence path from S to D
  4. Encode path as SR segment list:
     - If post-convergence path goes through node P: push Node-SID(P)
     - If specific link required: push Adj-SID(link)
  5. Pre-install backup with SR label stack in FIB

Result: Any single failure is repairable with at most 2-3 SR labels
Convergence: ~50ms (FIB switch only, no control plane computation)
```

---

## Prerequisites

- ospf, bgp, mpls, mpls-vpn, segment-routing, graph theory, shortest path algorithms, ipv6

## Complexity

| Operation | Time | Space |
|-----------|------|-------|
| SPF (Dijkstra, binary heap) | $O((V + E) \log V)$ | $O(V)$ |
| Incremental SPF | $O(\Delta V + \Delta E)$ | $O(V)$ |
| Partial SPF (Type 3/5 only) | $O(L)$ where $L$ = affected LSAs | $O(L)$ |
| LSA flooding per update | $O(N)$ where $N$ = neighbors | $O(1)$ |
| LFA computation (all destinations) | $O(N_S \times (V + E) \log V)$ | $O(N_S \times V)$ |
| rLFA PQ-node computation | $O(V^2 \log V)$ | $O(V^2)$ |
| TI-LFA post-convergence SPF | $O((V + E) \log V)$ per failure | $O(V)$ |
| LSA refresh overhead | $O(L / 1800)$ LSAs/second | $O(L)$ |

Where $N_S$ = number of direct neighbors of the computing router.

## References

- [RFC 2328 — OSPF Version 2](https://www.rfc-editor.org/rfc/rfc2328)
- [RFC 5340 — OSPF for IPv6 (OSPFv3)](https://www.rfc-editor.org/rfc/rfc5340)
- [RFC 5838 — OSPFv3 Address Families](https://www.rfc-editor.org/rfc/rfc5838)
- [RFC 3623 — Graceful OSPF Restart](https://www.rfc-editor.org/rfc/rfc3623)
- [RFC 3630 — TE Extensions to OSPF](https://www.rfc-editor.org/rfc/rfc3630)
- [RFC 3101 — OSPF NSSA Option](https://www.rfc-editor.org/rfc/rfc3101)
- [RFC 8665 — OSPF Extensions for Segment Routing](https://www.rfc-editor.org/rfc/rfc8665)
- [RFC 5286 — Basic Specification for IP FRR: Loop-Free Alternates](https://www.rfc-editor.org/rfc/rfc5286)
- [RFC 7490 — Remote LFA FRR](https://www.rfc-editor.org/rfc/rfc7490)
- [RFC 4552 — Authentication/Confidentiality for OSPFv3](https://www.rfc-editor.org/rfc/rfc4552)
- [RFC 4576 — Using a Link State Advertisement (LSA) Options Bit to Prevent Looping in BGP/MPLS IP VPNs](https://www.rfc-editor.org/rfc/rfc4576)
- [RFC 4577 — OSPF as PE/CE Protocol in BGP/MPLS IP VPNs](https://www.rfc-editor.org/rfc/rfc4577)
- [Dijkstra, E.W. — "A Note on Two Problems in Connexion with Graphs" (1959)](https://doi.org/10.1007/BF01386390)
- [Moy, J. — "OSPF: Anatomy of an Internet Routing Protocol" (Addison-Wesley, 1998)](https://www.ospf.org/)

---

*Every OSPF router in a 500-node area solves a graph problem that Dijkstra invented in 1956 on a napkin, running it hundreds of times per day in microseconds, rebuilding the forwarding plane on the fly while packets continue to flow. The algorithm is 70 years old. The engineering challenges of running it at scale are as fresh as ever.*
