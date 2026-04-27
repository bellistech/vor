# The Mathematics of OSPF — Algorithm, Convergence, Complexity

> *OSPF is Dijkstra's 1959 shortest-path algorithm running on every router, rebuilding the network graph in real time. The math spans graph theory (SPF computation), combinatorics (LSA flooding), Markov-chain analysis (convergence under failure), capacity planning (area design), and cryptography (authentication). Understanding the formal model is the difference between a stable IGP and a network that micro-loops on every link flap.*

---

## 0. Notation and Preliminaries

Throughout this document we adopt the following notation:

- $G = (V, E)$ — the graph induced by an OSPF area's LSDB. Vertices include routers and pseudonodes (DRs representing broadcast/NBMA segments). Edges are bidirectional adjacencies but represented as two directed edges in the SPF input.
- $V$ — the cardinality $|V|$, often interchangeably called "node count" or "router count" when no broadcast pseudonodes are present.
- $E$ — $|E|$; for IGP topologies $E$ is typically $1$-$10 \times V$.
- $w : E \to \mathbb{N}_{\geq 1}$ — non-negative integer cost on each edge. OSPF cost is constrained $1 \leq w \leq 65{,}535$ (16-bit field).
- $d(s, v)$ — shortest-path distance from source $s$ to vertex $v$.
- $\text{SPT}(s)$ — Shortest Path Tree rooted at $s$, output of Dijkstra's algorithm.
- $T_x$ — time consumed by phase $x$ (detection, flooding, SPF, FIB programming).
- $\lambda$ — failure rate (events per second) in Markov-chain analyses.
- $\mu$ — recovery rate (events per second), the reciprocal of MTTR.

OSPF defines five protocol packet types (ip-protocol 89, multicast destinations 224.0.0.5 / 224.0.0.6 for v2 and ff02::5 / ff02::6 for v3):

| Type | Name                          | Purpose                                  |
|:----:|:------------------------------|:------------------------------------------|
| 1    | Hello                         | Adjacency formation, keepalive            |
| 2    | Database Description (DBD)    | LSDB summary exchange during sync         |
| 3    | Link State Request (LSR)      | Request specific LSAs                     |
| 4    | Link State Update (LSU)       | Carry LSA payload                         |
| 5    | Link State Acknowledgment     | Confirm LSU receipt                       |

Each packet carries a 64-bit (v2) authentication field or a separate trailer (v3, RFC 7166). All OSPF reasoning ultimately reduces to: when do these five packet types fire, and how does the LSDB they synchronise feed Dijkstra's algorithm?

---

## 1. Dijkstra's Algorithm Applied to the LSDB

### 1.1 Formal Definition

Let $G = (V, E)$ be a weighted directed graph where $V$ is the set of vertices (routers and transit networks) and $E \subseteq V \times V$ is the set of edges (point-to-point or pseudonode adjacencies). Each edge $(u, v) \in E$ has a non-negative weight $w(u, v) \in \mathbb{R}_{\geq 0}$ — the OSPF interface cost.

The Single-Source Shortest Path (SSSP) problem is: given a source vertex $s \in V$, compute $d(s, v)$ for every $v \in V$, where:

$$d(s, v) = \min_{P \in \mathcal{P}(s, v)} \sum_{e \in P} w(e)$$

and $\mathcal{P}(s, v)$ is the set of all paths from $s$ to $v$. Dijkstra's algorithm solves SSSP in time bounded by the data structure used for the priority queue (PQ).

### 1.2 Pseudocode (Standard Dijkstra)

```
DIJKSTRA(G, w, s):
    for each v in V:
        d[v]   <- infinity
        pi[v]  <- nil           // predecessor pointer for SPF tree
    d[s] <- 0
    Q <- BuildPriorityQueue(V)  // keyed on d[v]
    S <- empty                  // settled set

    while Q is not empty:
        u <- ExtractMin(Q)      // smallest d[u] not yet settled
        S <- S union {u}
        for each (u, v) in Adj[u]:
            if d[u] + w(u, v) < d[v]:
                d[v]  <- d[u] + w(u, v)
                pi[v] <- u
                DecreaseKey(Q, v, d[v])
    return (d, pi)
```

The output is the shortest-path tree (SPT) rooted at $s$ — exactly the data structure OSPF installs as its routing table.

### 1.3 Time Complexity by PQ Implementation

| PQ data structure        | ExtractMin    | DecreaseKey   | Total                              |
|:-------------------------|:--------------|:--------------|:-----------------------------------|
| Adjacency matrix (array) | $O(V)$        | $O(1)$        | $O(V^2)$                           |
| Binary heap              | $O(\log V)$   | $O(\log V)$   | $O((V + E) \log V)$                |
| Pairing heap (amortized) | $O(\log V)$   | $O(\log V)$   | $O(E + V \log V)$                  |
| Fibonacci heap           | $O(\log V)$   | $O(1)$ amort. | $O(E + V \log V)$                  |
| van Emde Boas (int keys) | $O(\log\log U)$ | $O(\log\log U)$ | $O((V + E)\log\log U)$         |

In practice, OSPF implementations universally use binary heaps. Fibonacci heaps' constant factors are larger than the asymptotic improvement justifies for $V < 10^6$, and OSPF areas are bounded well below this threshold.

### 1.4 Space Complexity

- Vertex distance array $d[\cdot]$: $\Theta(V)$
- Predecessor array $\pi[\cdot]$: $\Theta(V)$
- Priority queue: $\Theta(V)$
- Adjacency list (input): $\Theta(V + E)$

Total: $\Theta(V + E)$. For 5,000 routers with average degree 4: $\sim 25{,}000$ pointers $\approx 200$ KB. A modern router with gigabytes of RAM is bounded by LSDB size, not SPT memory.

### 1.5 Why a Priority Queue Is Necessary

The naive "find min in linear scan" gives $O(V^2)$. A PQ amortizes the cost of repeatedly finding the minimum. The proof of optimality of Dijkstra requires the **monotone selection invariant**:

$$\forall u \in S, v \in V \setminus S : d(u) \leq d(v)$$

i.e. once a vertex is settled, no shorter path to it can exist. This invariant holds **only** for non-negative edge weights. OSPF cost is bounded $1 \leq \text{cost} \leq 65{,}535$, so the invariant always holds.

### 1.6 Worst-Case Behavior on Dense Graphs

For $E = \Theta(V^2)$ (complete graph):

- Adjacency matrix: $O(V^2)$ — optimal
- Binary heap: $O((V + V^2) \log V) = O(V^2 \log V)$ — *worse*

Pure data-centre fabrics are dense. Modern OSPF implementations sometimes auto-select the matrix variant when $E > V \log V / k$ for some constant $k$. FRR's ospfd uses a binary-heap implementation unconditionally.

### 1.7 Worked Example — Dijkstra on a 5-Node Graph

```
Graph weights:                  Initial state:
                                d[A]=0, all others = inf
        2                       Q = {A:0, B:inf, C:inf, D:inf, E:inf}
   A ─────── B
   │         │ 3                Iteration 1: settle A
 1 │         │                    relax (A,B): d[B] = 0 + 2 = 2
   │    4    │                    relax (A,C): d[C] = 0 + 1 = 1
   C ─────── D                  Iteration 2: settle C (d=1)
   │         │ 1                  relax (C,D): d[D] = 1 + 4 = 5
 5 │         │                    relax (C,E): d[E] = 1 + 5 = 6
   E ─────── ┘                  Iteration 3: settle B (d=2)
        6                         relax (B,D): d[D] = min(5, 2+3) = 5  (no change)
                                Iteration 4: settle D (d=5)
                                  relax (D,E): d[E] = min(6, 5+1) = 6  (no change)
                                Iteration 5: settle E (d=6)

Resulting SPT:                  Distances from A:
   A → B   (2)                    d(A)=0, d(B)=2, d(C)=1, d(D)=5, d(E)=6
   A → C → D (5)
   A → C → E (6)
```

This is exactly what OSPF computes for each router-LSA / network-LSA pair as edges in the SPF tree.

### 1.8 Two-Stage SPF — Why OSPF Computes Twice

The actual OSPF SPF runs in two stages on the LSDB:

```
Stage 1 — Intra-area SPF:
  Input:  Router-LSAs (Type 1) + Network-LSAs (Type 2) for area A
  Output: SPT_A — shortest paths to all routers and transit nets in A
Stage 2 — Inter-area SPF:
  Input:  Summary-LSAs (Type 3, 4) and SPT_A
  Output: distances to inter-area destinations via best ABR
Stage 3 — External:
  Input:  AS-External-LSAs (Type 5) and SPT for ASBRs
  Output: distances to external destinations via best ASBR
```

This staged pipeline keeps each SPF run on a smaller graph than the union of all LSDBs would produce. Without it, a single failure in one area would force every router to recompute paths to all external destinations.

### 1.9 Pseudonode Modeling

Multi-access broadcast networks (Ethernet LANs) are modeled in the SPF graph as a star: one pseudonode (the DR) plus $n$ stub edges to each attached router.

```
Without pseudonode (full mesh):       With pseudonode (star):
  edges = n * (n - 1) / 2               edges = 2 * n
  for n=10: 45 edges                   for n=10: 20 edges
  for n=50: 1225 edges                 for n=50: 100 edges
```

The pseudonode reduces the edge count from $O(n^2)$ to $O(n)$ in the SPF graph and simplifies the LSDB representation. Critically, the pseudonode is not a real router — it is a logical vertex synthesised by the DR's Network-LSA.

### 1.10 Edge Weight Asymmetry

Edge weights in OSPF can be asymmetric: $w(u, v) \neq w(v, u)$ is permitted. This is rare in practice (usually administrators configure symmetric costs), but possible. Dijkstra handles this correctly because it operates on directed edges.

A pathological asymmetric cost configuration can produce **routing micro-loops** during failures: $R_1$ chooses $R_2$ as next-hop based on $R_1$'s view, but $R_2$ chooses a different next-hop including $R_1$. While the LSDB is consistent across the area, transient FIB inconsistency during reconvergence can manifest these loops.

---

## 2. The LSA Flooding Math

### 2.1 The Reliable Flooding Protocol

OSPF flooding is the gossip protocol that propagates LSAs through an area. The algorithm is **reliable**: every received LSA is acknowledged (LSAck), every unacknowledged LSA is retransmitted (RxmtInterval, default 5s).

For each LSA received on interface $i$:

```
RECEIVE-LSA(lsa, in_interface):
    if AGE(lsa) == MaxAge:
        flush from LSDB, flood out all interfaces except in_interface
        return
    cmp = COMPARE-LS-VERSION(lsa, LSDB[lsa.id])
    if cmp == NEWER:
        install lsa in LSDB
        for each interface j != in_interface:
            queue lsa on j's flood list
        send LSAck on in_interface
    elif cmp == SAME:
        send LSAck on in_interface (implicit/explicit)
    elif cmp == OLDER:
        send our newer copy back on in_interface
```

### 2.2 Flooding Time Bound

Worst-case LSA propagation time across an area:

$$T_{flood} = N_{hops} \times (T_{prop} + T_{proc} + T_{queue}) + T_{pacing}$$

Where:
- $N_{hops}$ = diameter of the area in hops
- $T_{prop}$ = link propagation delay (1 ms per 200 km of fibre)
- $T_{proc}$ = LSA verification + LSDB lookup + checksum verify $\approx 50\text{-}500\,\mu s$
- $T_{queue}$ = output queue delay (interface-rate-dependent)
- $T_{pacing}$ = LSA group pacing (default 30s, but flooded LSAs bypass pacing)

For a typical campus area with diameter 6 hops, $T_{flood} \approx 6 \times 600\,\mu s \approx 3.6$ ms — flooding is dominated by SPF and FIB programming, not flood propagation.

### 2.3 The MaxAge Mechanism

Every LSA has a 16-bit unsigned $\text{Age}$ field, incremented every second by every router holding it.

$$\text{Age range} = [0, 3600] \text{ seconds}$$

When $\text{Age} = 3600$ (MaxAge), the LSA is flushed. The originator may also explicitly age out its own LSA by setting $\text{Age} = 3600$ and re-flooding — this is how withdrawal works.

### 2.4 LSRefresh Math

To prevent legitimate LSAs from aging out, the originator re-floods every $\text{LSRefreshTime} = 1800$ s.

$$\text{Refresh rate} = \frac{N_{LSA}}{1800} \text{ LSAs/s}$$

For $N_{LSA} = 10{,}000$: $5.56$ refreshes/s, $\approx 167$ /min. At 100 bytes/LSA (typical router-LSA encoded size), this is $\sim 4.5$ kbps continuous control-plane bandwidth. Negligible at scale.

### 2.5 LSA Sequence Numbers

The Sequence Number is a 32-bit signed integer that wraps at $0x7FFFFFFF$. The initial sequence is $0x80000001$.

$$\text{SeqNo space} = [0x80000001, 0x7FFFFFFF]$$

Comparison rules (RFC 2328 §13.1):
1. If both sequence numbers equal $0x80000000$, they are considered the same.
2. Else, larger sequence wins.
3. If sequence equal, larger checksum wins.
4. If checksum equal, lower age wins (with MaxAge-tolerance window of 15 minutes).

If a router exhausts the sequence space (must re-originate $2^{31} - 1 = 2{,}147{,}483{,}647$ times), it must wait MaxAge for the old LSA to flush before originating from $0x80000001$ again. At the LSRefreshTime of 30 minutes, this would take 122,000 years — not an operational concern.

### 2.6 LSA Checksum

Each LSA carries a 16-bit Fletcher checksum (ISO 8473 Annex C, "the IS-IS checksum"). It covers the LSA header (excluding the Age field, which mutates) and the body.

$$C_0 = \sum b_i \mod 255 \qquad C_1 = \sum C_0 \mod 255$$

The checksum is recomputed during LSA verification on every receipt. A mismatch causes the LSA to be silently discarded (no LSAck), forcing retransmission.

Why Fletcher and not CRC? Fletcher detects the same single-bit and burst errors with simpler arithmetic — sums modulo 255, no polynomial division. CPU-cheap, sufficient for distinguishing bit-flips that survived TCP/UDP-style checksums. For $L = 1500$-byte LSA, Fletcher requires $\sim 1500$ adds and 1 modulo; CRC-16 requires equivalent work.

### 2.7 Flooding Acknowledgment Bookkeeping

Each interface maintains:

- **Link-State Retransmission List**: LSAs flooded out the interface awaiting LSAck
- **Database Description List**: LSAs to send during the initial DBD exchange
- **Link-State Request List**: LSAs requested from a neighbour
- **Delayed LSAck List**: pending acks within RxmtInterval window

Flooding pseudocode:

```
SEND-LSA-ON-INTERFACE(lsa, iface):
    add lsa to iface.RxmtList
    transmit(LSU containing lsa, multicast(224.0.0.5))
    schedule retransmit at now() + RxmtInterval

ON-LSAck-RECEIVED(lsa_id, iface):
    remove lsa_id from iface.RxmtList

ON-RxmtInterval-EXPIRY(iface, lsa):
    if lsa still in RxmtList:
        unicast retransmit to neighbour
        re-arm retransmit timer
```

For a stable area with no neighbour failures, RxmtList stays nearly empty; control-plane bandwidth is dominated by LSRefresh.

### 2.8 Database Description Sequence

When two routers form an adjacency, they exchange LSDB summaries (DBD packets) to discover what each side has:

```
ExStart state:
  Master/Slave election: higher Router-ID becomes master
  Master initialises DD-Sequence-Number (32-bit)

Exchange state:
  Master sends DBD with sequence S, Slave responds with DBD-S
  Each DBD lists LSA headers (no body)
  Slave's response increments S

Loading state:
  For each missing LSA, send LSR
  Receive LSU containing requested LSAs
  Send LSAck

Full state:
  All LSAs in sync
```

Time for full LSDB sync depends on LSDB size and link bandwidth:

$$T_{sync} = \frac{|\text{LSDB}|}{\text{link-bw}} + N_{LSA} \times T_{processing}$$

For 100 MB LSDB on a 10 Gbps link: $T_{sync} \approx 100 \text{ ms} + N_{LSA} \times 100\,\mu s$. For $N_{LSA} = 10^5$: $\sim 10$ s of sync time after a router boot.

---

## 3. Convergence Time Decomposition

### 3.1 The Total Convergence Equation

$$T_{convergence} = T_{detect} + T_{originate} + T_{flood} + T_{SPF} + T_{FIB}$$

Each term is bounded by independent timers and hardware constraints.

### 3.2 Failure Detection — $T_{detect}$

| Mechanism                  | Default               | Range                   | Notes                                    |
|:---------------------------|:----------------------|:------------------------|:-----------------------------------------|
| Layer-1 link-down          | $\sim 50$ ms          | hardware                | Ethernet PHY autoneg / SFP loss-of-light |
| OSPF Hello (broadcast)     | $4 \times 10 = 40$ s  | $4 \times \text{hello}$ | Slowest; default                         |
| OSPF Hello (P2P)           | $4 \times 10 = 40$ s  | same                    | Same defaults as broadcast               |
| OSPF Hello (NBMA)          | $4 \times 30 = 120$ s | configurable            | Frame-relay legacy                       |
| OSPF fast hello            | 1 s detect            | $1$-$5$ Hz              | `ip ospf hello-multiplier`               |
| BFD single-hop             | $50$-$300$ ms         | sub-second              | RFC 5880; preferred                      |
| BFD echo                   | $\sim 30$ ms          | hardware-assisted       | Asymmetric; not all platforms            |

For modern deployments: rely on link-down for direct adjacencies, BFD for everything else. The OSPF Dead Interval is a *backstop*, not a primary failure detector.

### 3.3 LSA Origination Throttle — $T_{originate}$

OSPF rate-limits LSA origination to prevent self-induced storms. The throttle is exponential:

$$T_{originate}(n) = \min(T_{init} \cdot 2^{n}, T_{max})$$

Cisco IOS XE defaults: $T_{init} = 0$ ms, $T_{hold} = 5000$ ms, $T_{max} = 5000$ ms (controlled by `timers throttle lsa`). FRR defaults: $T_{init} = 0$ ms, $T_{hold} = 5000$ ms, $T_{max} = 5000$ ms.

### 3.4 SPF Throttle — $T_{SPF}$

After the first LSA arrives, OSPF schedules SPF after $T_{init}$. Subsequent triggers within the hold window double the wait, capped at $T_{max}$:

$$T_{SPF}(n) = \min(T_{init} \cdot 2^{n-1}, T_{max})$$

Default values vary:

| Vendor     | $T_{init}$ | $T_{hold}$ | $T_{max}$ |
|:-----------|:----------|:-----------|:----------|
| Cisco IOS  | 5000 ms   | 10000 ms   | 10000 ms  |
| IOS XR     | 50 ms     | 200 ms     | 5000 ms   |
| Juniper    | 200 ms    | 1000 ms    | 5000 ms   |
| Arista EOS | 1000 ms   | 5000 ms    | 5000 ms   |
| FRR        | 0 ms      | 50 ms      | 5000 ms   |

### 3.5 FIB Programming — $T_{FIB}$

Once SPF produces the new RIB, the FIB must be programmed in hardware:

| Hardware                | Programming rate    | $T_{FIB}$ for 100 routes  |
|:------------------------|:--------------------|:---------------------------|
| Pure software (Linux)   | $10^6$ routes/s     | 100 µs                     |
| Mid-range merchant ASIC | $10^4$-$10^5$ rt/s  | 1-10 ms                    |
| Older TCAM-based        | $10^3$-$10^4$ rt/s  | 10-100 ms                  |
| Very old CEF software   | $10^2$ rt/s         | seconds                    |

PIC (Prefix Independent Convergence) decouples convergence from prefix count by using next-hop pointers — the FIB doesn't need to be reprogrammed per-prefix when a next-hop becomes invalid.

### 3.6 Worked Example — 24-Leaf, 6-Spine Clos

```
Topology:            Failure event: leaf-3 ↔ spine-2 link down
                     Detection:     50 ms (link-down)
  spine-1 ... 6      LSA origin:    leaf-3 originates Router-LSA at t=50ms
   /  \              Flood diameter: leaf → all spines → all leaves = 2 hops
  /    \             T_flood:       2 * (1 ms + 100 µs) ≈ 2.2 ms
 leaf-1 ... 24       T_SPF:         5 ms (FRR, 100-router area)
                     T_FIB:         < 1 ms (modern broadcom Tomahawk)

Total T_convergence  = 50 ms + ~0 ms + 2.2 ms + 5 ms + 1 ms ≈ 58 ms
Without BFD          = 40 s + 0 + 2.2 ms + 5 ms + 1 ms     ≈ 40 s
```

Conclusion: BFD reduces convergence from 40 s to 60 ms — three orders of magnitude. This is why every modern data centre runs BFD on every IGP adjacency.

### 3.7 LFA / RLFA / TI-LFA

Loop-Free Alternates (RFC 5286) precompute backup paths in SPF time. When a link fails:

$$T_{convergence}^{LFA} = T_{detect} + T_{FIB-switch}$$

The flood, SPF, and re-FIB-program steps disappear from the critical path because the backup is already in the FIB. Sub-50 ms convergence is routinely achieved with LFA + BFD.

The LFA condition for backup neighbour $N$ to destination $D$ is:

$$\text{Distance}(N, D) < \text{Distance}(N, S) + \text{Distance}(S, D)$$

Where $S$ is the protecting router. This guarantees $N$ does not loop back through $S$.

#### Worked Example — LFA Coverage in a Triangle

```
        cost=10
   A ─────────── B
    \           /
   5 \         / 3
      \       /
        C
```

If link A-B fails on A:
- A's primary: A → B (cost 10)
- A's neighbours: B (down), C (cost 5)
- LFA candidate: C
- Check: distance(C, B) = 3, distance(C, A) = 5, distance(A, B) = 10
- Condition: 3 < 5 + 10 = 15. Pass.
- Therefore C is loop-free alternate; A pre-installs A → C → B as backup.

If C didn't satisfy the condition, A would have no LFA — convergence falls back to standard OSPF flood + recompute. RFC 5286 calls this "remote LFA" (RLFA) and tunnels to a PQ-node satisfying the condition.

#### LFA Coverage Probability

Empirical studies on real ISP topologies show LFA covers $\sim 70$-$80\%$ of failures. Coverage depends on topology:

| Topology                | LFA coverage |
|:------------------------|:-------------|
| Full mesh ($K_n$)       | 100%         |
| 2D torus / grid         | 95-100%      |
| Tree / hub-and-spoke    | 0% (no alternates) |
| Random scale-free       | 60-80%       |
| Real-world ISP backbone | 70-85%       |

TI-LFA achieves **100% coverage** by allowing the alternate path to traverse non-feasible neighbours via SR-encoded detours.

Topology-Independent LFA (TI-LFA, RFC 9166) extends this with segment routing — guaranteed 100% coverage by encoding an explicit detour via SR labels.

#### TI-LFA Repair Path Computation

```
TI-LFA(F):  // F is the failed link or node
    Q = post-convergence path from S avoiding F
    PQ = nodes reachable on Q with no traversal of F
    if exists a node P in PQ such that S → P uses no F:
        repair_path = (Adjacency-SID for next hop in pre-failure path)
                       ++ (Prefix-SID for P)
                       ++ (Adjacency-SID for first link on Q from P)
    else:
        recursively combine Adjacency-SIDs along Q
```

The repair path encodes an explicit forwarding sequence that survives the failure. SR's stateless nature lets every node along the path forward without coordination.

---

## 4. SPF Throttling Algorithm (RFC 4503 / Vendor-Specific)

### 4.1 The State Machine

```
        +------------+   first LSA            +------------+
        |   QUIET    | ───────────────────▶  |  WAITING   |
        |  (idle)    |    schedule SPF       | (init-delay)|
        +------------+                       +------------+
              ▲                                    │
              │ T_quiet elapsed                    │ delay expires
              │ no triggers                        ▼
        +------------+   re-trigger          +------------+
        |   HOLD     | ◀──────────────────── |  RUNNING   |
        | (back-off) |    within hold        |  (compute) |
        +------------+                       +------------+
                                                   │
                                                   ▼
                                              install RIB
```

### 4.2 Why Exponential Back-off

Consider a flapping link generating LSA every 100 ms. Without throttling:

- 10 SPFs/s, each 50 ms on a 1000-router area
- Sustained 50% CPU utilization on the SPF process
- Other processes (BGP, BFD, MPLS) are starved
- BFD timeouts $\rightarrow$ false neighbour-down events
- Cascade failure

With exponential throttling:

| Trigger # | Wait (ms) | Cumulative (ms) |
|:----------|:----------|:----------------|
| 1         | 50        | 50              |
| 2         | 100       | 150             |
| 3         | 200       | 350             |
| 4         | 400       | 750             |
| 5         | 800       | 1550            |
| 6         | 1600      | 3150            |
| 7         | 3200      | 6350            |
| 8         | 5000 cap  | 11350           |
| 9         | 5000      | 16350           |

After 5 s of quiet, the back-off resets to $T_{init}$. This bounds CPU consumption while preserving fast convergence in the common case.

### 4.3 Pseudocode

```
SPF-SCHEDULE(trigger_time):
    if state == QUIET:
        state <- WAITING
        spf_at <- now() + T_init
        n <- 1
        schedule(spf_at)
    elif state == WAITING or RUNNING:
        // already pending — do nothing
        pass
    elif state == HOLD:
        n <- n + 1
        delay <- min(T_init * 2^(n-1), T_max)
        spf_at <- now() + delay
        schedule(spf_at)

ON-SPF-COMPLETE():
    state <- HOLD
    last_spf <- now()
    arm_quiet_timer(T_quiet)

ON-QUIET-TIMER():
    if (now() - last_spf) >= T_quiet:
        state <- QUIET
        n <- 0
```

---

## 5. iSPF (Incremental SPF)

### 5.1 Mathematical Justification

A full SPF on $V$ nodes costs $O((V + E) \log V)$. If a single edge $(u, v)$ changes weight, the affected portion of the SPT is the subtree rooted at $v$ — typically $\ll V$ vertices.

Define $\text{Sub}(v)$ as the set of vertices whose shortest path passes through $v$ in the current SPT. Incremental SPF recomputes only $\text{Sub}(v)$:

$$T_{iSPF} = O(|\text{Sub}(v)| \log V)$$

For a tree with branching factor $b$ and depth $d$, a leaf affects only itself ($O(\log V)$). An internal node halfway down affects $\sim b^{d/2}$ vertices.

### 5.2 When iSPF Can Run

iSPF is correct only when the change preserves the structure of the SPT outside $\text{Sub}(v)$. Conditions:

1. **Metric-only change on existing link**: edge weight $w(u, v)$ changed but topology unchanged. iSPF safe.
2. **Edge added that does not improve any $d(s, x)$ for $x \notin \text{Sub}(v)$**: iSPF safe.
3. **Edge removed within $\text{Sub}(v)$**: iSPF safe (re-root the subtree).
4. **Edge removed outside $\text{Sub}(v)$**: must do full SPF; iSPF unsafe.
5. **Topological change at routers near root**: full SPF required.

Implementations conservatively fall back to full SPF when classification is ambiguous.

### 5.3 Cisco IOS XR Numbers

Real-world measurements (from Cisco published benchmarks):

| Area size | Full SPF | iSPF (single-link) | Speedup |
|:----------|:---------|:-------------------|:--------|
| 100 nodes | 5 ms     | 0.5 ms             | 10x     |
| 500 nodes | 25 ms    | 1 ms               | 25x     |
| 2000 nodes| 90 ms    | 2 ms               | 45x     |
| 10000 nodes| 500 ms  | 5 ms               | 100x    |

iSPF's relative speedup grows with area size. For very large IS-IS / OSPF deployments, iSPF is mandatory.

### 5.4 Pseudocode (Sketch)

```
iSPF(G, change):
    if change is metric-only on edge (u, v):
        affected <- subtree-of-v-in-SPT
        for each x in affected:
            invalidate d[x], pi[x]
        // local Dijkstra rooted at v's parent
        local-dijkstra(parent(v), affected)
    elif change adds edge (u, v) and w(u,v) < d[v] - d[u]:
        d[v]   <- d[u] + w(u, v)
        pi[v]  <- u
        propagate-relaxation-down(v)
    else:
        full-SPF()
```

### 5.5 Subtree Identification

The subtree $\text{Sub}(v)$ rooted at vertex $v$ in the current SPT can be identified in $O(|\text{Sub}(v)|)$ via a depth-first traversal of the predecessor pointers:

```
SUB(v, SPT):
    result = {v}
    stack = [v]
    while stack not empty:
        x = pop(stack)
        for each child c of x in SPT:
            result.add(c)
            stack.push(c)
    return result
```

The SPT is stored as a set of predecessor pointers $\pi[\cdot]$; the children-of relation is its inverse. For efficient inverse lookup, OSPF implementations maintain both directions explicitly.

### 5.6 Decremental SPF (the Hard Case)

When an edge weight increases or an edge is removed, paths that used that edge must be recomputed. The set of affected vertices is bounded by:

$$|\text{affected}| \leq |\text{Sub}(v)| + |\{x : \pi[x] \in \text{Sub}(v)\}|$$

The naive approach: invalidate $\text{Sub}(v)$, run partial Dijkstra to relax from boundary nodes inward. Optimal subroutines like Demetrescu-Italiano (2004) achieve incremental all-pairs shortest paths with $O(V^2)$ amortized per update, but for SSSP a simpler bounded re-Dijkstra suffices.

---

## 6. Stub / Totally Stubby / NSSA Mathematics

### 6.1 LSA Type Filtering

| Area type        | Type 1 | Type 2 | Type 3 | Type 4 | Type 5 | Type 7 | Default |
|:-----------------|:------:|:------:|:------:|:------:|:------:|:------:|:-------:|
| Backbone (0)     | yes    | yes    | yes    | yes    | yes    | no     | no      |
| Standard area    | yes    | yes    | yes    | yes    | yes    | no     | no      |
| Stub             | yes    | yes    | yes    | no     | no     | no     | yes     |
| Totally stubby   | yes    | yes    | no(*)  | no     | no     | no     | yes     |
| NSSA             | yes    | yes    | yes    | yes    | no     | yes    | optional |
| Totally NSSA     | yes    | yes    | no(*)  | no     | no     | yes    | yes     |

(*) Only the default Type-3 from ABR is permitted.

### 6.2 LSDB Size Reduction

Let:
- $R$ = routers in area
- $L_{int}$ = inter-area prefixes (Type 3 from other areas)
- $E$ = external prefixes (Type 5)
- $A$ = number of ABRs in area
- $E_{local}$ = NSSA Type-7 prefixes generated locally

$$\text{LSDB}_{\text{normal}} = O(R + N_{net} + L_{int} + A \cdot \text{ASBR-count} + E)$$

$$\text{LSDB}_{\text{stub}} = O(R + N_{net} + L_{int} + 1)$$

$$\text{LSDB}_{\text{totally-stubby}} = O(R + N_{net} + 1)$$

$$\text{LSDB}_{\text{NSSA}} = O(R + N_{net} + L_{int} + A + E_{local})$$

### 6.3 Worked Example — Internet Edge Stub Area

```
Scenario: Branch office, 5 routers, OSPF area 10.
Backbone has 800,000 redistributed BGP routes injected into OSPF as Type-5 LSAs.

Normal area:
  LSDB = 5 (Type 1) + 0 (no broadcast nets) + ~50 (inter-area) + 1 (ASBR Type-4)
       + 800,000 (Type 5)
       ≈ 800,056 LSAs
       ≈ 80 MB at 100 bytes/LSA

Stub area:
  LSDB = 5 + 50 + 1 (default route)
       ≈ 56 LSAs
       ≈ 5.6 KB

Reduction factor: ~14,000x
```

The branch-office router survives on a Pi 4; without stub, even a Cisco ASR1000 would struggle.

### 6.4 Type-7 Translation at NSSA ABR

NSSA ABRs translate Type-7 to Type-5 when leaving the area, deterministically selected by highest router-id among ABRs:

$$\text{Translator} = \arg\max_{r \in \text{ABRs}(area)} \text{RouterID}(r)$$

The non-translator ABRs do not regenerate the LSA. This eliminates duplicate Type-5 LSAs in the backbone.

---

## 7. Cost / Bandwidth Math

### 7.1 The Cost Formula

$$\text{cost} = \left\lfloor \frac{\text{ref-bw}}{\text{interface-bw}} \right\rfloor$$

Default $\text{ref-bw} = 10^8$ bps (100 Mbps) per RFC 2328. The cost is bounded:

$$1 \leq \text{cost} \leq 65535$$

Anything above 100 Mbps with default reference saturates at cost 1.

### 7.2 Worked Examples

| Interface           | Bandwidth (bps) | Cost (default 100M ref) | Cost (100G ref) |
|:--------------------|:----------------|:------------------------|:-----------------|
| Serial T1           | 1,544,000       | 64                      | 64,766           |
| 10BASE-T            | 10,000,000      | 10                      | 10,000           |
| 100BASE-T           | 100,000,000     | 1                       | 1,000            |
| 1000BASE-T          | 1,000,000,000   | 1 (capped)              | 100              |
| 10GbE               | 10,000,000,000  | 1 (capped)              | 10               |
| 40GbE               | 40,000,000,000  | 1 (capped)              | 2 (rounded)      |
| 100GbE              | 100,000,000,000 | 1 (capped)              | 1                |
| 400GbE              | 400,000,000,000 | 1 (capped)              | 1 (capped again) |

Modern recommendation: `auto-cost reference-bandwidth 1000000` (1 Tbps) on routers expecting 400G/800G interfaces, even if those don't yet exist locally — avoids reconfiguration when fabrics are upgraded.

### 7.3 ECMP Threshold

OSPF default `maximum-paths` is 4 on Cisco IOS, 16 on IOS XR / Juniper / FRR. The math constraint for ECMP installation:

$$\text{cost}_i = \min_{j \in \mathcal{P}} \text{cost}_j \quad \forall i \in \text{ECMP set}$$

i.e. only paths with exactly equal cost are installed. Unequal-cost ECMP requires EIGRP's variance feature; OSPF has no such mechanism.

---

## 8. ECMP and Load Balancing

### 8.1 Hash-Based ECMP

When $k$ equal-cost paths exist, the FIB hashes the packet's flow tuple modulo $k$ to select the egress path. Hash inputs typically:

- IPv4 5-tuple: src IP, dst IP, src port, dst port, protocol
- IPv6 6-tuple: 5-tuple + flow label
- MPLS: label stack + (entropy label if present, RFC 6790)
- Underlying inner-header inspection for tunnels (VXLAN, GENEVE, GRE)

Hash function: typically a CRC32, Toeplitz hash, or vendor-specific Robust Symmetric Hash. The hash output is reduced modulo path count:

$$\text{path}(p) = \text{hash}(\text{tuple}(p)) \mod k$$

### 8.2 Polarization

If every router in a multi-tier topology uses the same hash function with the same seed, traffic from the same flow always selects the same egress, but flows of differing tuples may all converge on one of the upstream paths, defeating load balancing.

```
Bad (polarized):                 Good (de-polarized):
  Tier-1                            Tier-1
  hash → path-A                     hash(seed=A) → path-A
  Tier-2 (same hash)                Tier-2 (different seed)
  hash → path-A again               hash(seed=B) → distributes
```

Mitigation: per-router hash seeds (Cisco's `mpls ldp explicit-null` or `mls ip cef load-sharing full simple`), CRC variation, or symmetric/asymmetric seeding.

### 8.3 Per-Flow vs Per-Packet ECMP

| Mode        | Order preservation | TCP performance | Use case                |
|:------------|:--------------------|:----------------|:------------------------|
| Per-flow    | yes                 | optimal         | default; production     |
| Per-packet  | no (reorders)       | poor (DUPACK storms) | benchmarking only |

TCP retransmission penalties from per-packet reordering: each out-of-order packet triggers 3 duplicate ACKs and a fast retransmit, halving cwnd via Reno congestion control. Production networks use per-flow ECMP exclusively.

### 8.4 Flowlet-Based ECMP

Newer fabrics (Cisco's CONGA, Microsoft Plenum) detect natural pauses in TCP flows ($> 50\,\mu s$) and re-hash at flowlet boundaries. This balances load while preserving in-order delivery within each TCP burst. Requires fabric-level hardware support; not part of OSPF itself.

---

## 9. SPF Algorithm Variants

### 9.1 Comparison Table

| Algorithm                  | Time complexity         | Space    | Implemented in OSPF? |
|:---------------------------|:------------------------|:---------|:---------------------|
| Dijkstra + binary heap     | $O((V+E)\log V)$        | $O(V)$   | yes (universal)      |
| Dijkstra + Fibonacci heap  | $O(V \log V + E)$       | $O(V)$   | no (constants)       |
| Dijkstra + pairing heap    | $O(V \log V + E)$ amort.| $O(V)$   | rare                 |
| Bidirectional Dijkstra     | $\sim O(\sqrt{V \log V \cdot E})$ effective | $O(V)$ | no (single source) |
| A* (heuristic)             | depends on heuristic    | $O(V)$   | no (no heuristic)    |
| Bellman-Ford               | $O(VE)$                 | $O(V)$   | no (slower)          |
| Floyd-Warshall (all-pairs) | $O(V^3)$                | $O(V^2)$ | no (overkill)        |
| Johnson's (all-pairs)      | $O(V^2 \log V + VE)$    | $O(V^2)$ | no                   |
| iSPF                       | $O(|\Delta| \log V)$    | $O(V)$   | yes (incremental)    |

### 9.2 Why Not Fibonacci Heaps

Fibonacci heaps achieve $O(1)$ amortized DecreaseKey. For Dijkstra:

$$T_{Fib} = O(V \log V + E)$$

For a graph with $V = 1000$, $E = 5000$:
- Binary heap: $(1000 + 5000) \times 10 = 60{,}000$ ops
- Fibonacci heap: $1000 \times 10 + 5000 = 15{,}000$ ops

But Fibonacci heap operations have constants $5\text{-}10\times$ a binary heap due to the cascading-cut bookkeeping. Real-world: binary heap wins until $V \gg 10^5$.

### 9.3 Bidirectional Dijkstra

Searches simultaneously from $s$ and from target $t$, terminating when frontiers meet:

$$T_{bidir} \approx O(\sqrt{V \log V \cdot E})$$

OSPF computes shortest paths from one source to all destinations (SSSP), so bidirectional Dijkstra (point-to-point shortest path) does not apply.

### 9.4 A* Heuristic

A* extends Dijkstra with an admissible heuristic $h(v)$ estimating distance to target:

$$f(v) = g(v) + h(v)$$

Where $g(v) = d(s, v)$ and $h(v) \leq d(v, t)$. OSPF has no natural geographic or topological heuristic — every router must compute paths to every other router. A* is irrelevant to link-state IGPs.

---

## 10. LSA Type Cardinality

### 10.1 Per-Area Counts

Let $R$ = routers in area, $N_{bcast}$ = broadcast networks, $L_{int}$ = inter-area prefixes injected, $A_{total}$ = total areas, $P_{external}$ = external prefixes.

| LSA Type | Quantity                               | Origin             |
|:---------|:----------------------------------------|:--------------------|
| Type 1 (Router)     | $R$                          | Each router         |
| Type 2 (Network)    | $N_{bcast}$                  | DR per broadcast LAN|
| Type 3 (Summary)    | $L_{int}$ per area received  | ABRs                |
| Type 4 (ASBR)       | $|\text{ASBRs}|$ per ABR     | ABRs                |
| Type 5 (External)   | $P_{external}$               | ASBRs               |
| Type 7 (NSSA)       | $P_{NSSA-local}$             | NSSA ASBRs          |
| Type 9 (Opaque link)| variable                     | per-link, RFC 5250  |
| Type 10 (Opaque area)| variable                    | per-area            |
| Type 11 (Opaque AS) | variable                     | AS-wide             |

### 10.2 Worked Example — Backbone with 100 Routers, 50 LANs, 4 Areas, 800k External Routes

```
Area 0 LSDB (assuming no stub on backbone):
  Type 1:  100 routers           = 100
  Type 2:  50 broadcast nets     = 50
  Type 3:  ~50 prefixes/area * 3 = 150  (from each non-backbone area)
  Type 4:  ~5 ASBRs              = 5
  Type 5:  800,000 external      = 800,000

Total LSAs: ~800,305
Memory:     ~80 MB at 100 bytes/LSA
Refresh BW: 800,305 / 1800 ≈ 445 LSAs/s ≈ 36 kbps continuous

Per-router CPU:
  SPF time on 100 nodes ≈ 5 ms / run
  Throttled to ≤ 1 SPF / 5 s in steady state
  CPU steady = 5 / 5000 = 0.1%
```

### 10.3 Opaque LSA (RFC 5250)

Opaque LSAs carry vendor- or extension-specific data, scoped per LSA type:
- Type 9: link-local (e.g., adjacency SID for SR)
- Type 10: area-wide (e.g., MPLS-TE link properties)
- Type 11: AS-wide (e.g., SR Mapping Server advertisements)

The 32-bit Link State ID encodes:

```
 0       7 8                    31
+--------+----------------------+
| Opaque | Opaque ID (24-bit)   |
|  Type  |                      |
+--------+----------------------+
```

Opaque Type 1 = MPLS-TE LSAs, Type 4 = Router Information LSA (RFC 7770), Type 7 = SR-related, etc.

---

## 11. The Multi-Area / Backbone Constraint

### 11.1 Why Area 0 Must Be Contiguous

OSPF is a two-level hierarchy: backbone (Area 0) and non-backbone areas. Inter-area routing is **distance-vector-like** between ABRs — a non-backbone ABR trusts Type-3 LSAs from another area's ABR only if relayed through Area 0.

Consider two areas connected only by a non-backbone path:

```
   Area 1 ─── Area 2 ─── Area 0
                ↑
      ABR-1   ABR-2     ABR-3
```

ABR-1 in Area 1 cannot reach Area 0 directly. ABR-2 sees Area-1 prefixes via Type-3, but per RFC 2328 §16.2, an ABR does **not** re-advertise Type-3 LSAs received from a non-backbone area into another non-backbone area. This prevents inter-area routing loops.

### 11.2 The Virtual Link Workaround

A virtual link is a logical Area-0 adjacency tunnelled through a transit area. Mathematically, it adds a virtual edge to Area 0's graph:

$$E_{Area0} \leftarrow E_{Area0} \cup \{(\text{ABR}_a, \text{ABR}_b)\}$$

with cost equal to the SPF cost through the transit area. This restores contiguity.

Constraints (RFC 2328 §15):
- Both endpoints must be ABRs.
- Transit area cannot be stub or NSSA (Type-3 inter-area routing must work).
- Authentication must match.
- HelloInterval and DeadInterval inflate (default 30 s / 120 s vs 10 s / 40 s for normal).

Operational guidance: virtual links are a band-aid. If you find yourself needing one permanently, you should redesign your area boundaries.

### 11.3 Multi-Topology and Address Family

OSPFv3 with RFC 5838 multi-topology allows independent SPF runs per address family (IPv4-unicast, IPv6-unicast, IPv4-multicast, IPv6-multicast). Each AF has its own LSDB and SPT:

$$\text{SPT}_{AF_i} = \text{Dijkstra}(G_{AF_i})$$

This is the formal foundation of OSPFv3's MT extension. Flex-Algo (RFC 9350) generalises further: each algorithm ID has its own SPT computed with custom metrics and constraints.

---

## 12. OSPFv2 vs OSPFv3 Wire-Format Differences

### 12.1 Common Header

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|   Version     |    Type       |       Packet Length           |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                          Router ID                            |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                           Area ID                             |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|           Checksum            | (v2: AuType / v3: Inst)| Reserved |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| (v2: Authentication 64 bits) — v3 has no auth fields here    |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

Version field: 2 for OSPFv2 (RFC 2328), 3 for OSPFv3 (RFC 5340). Type values 1-5 (Hello, DBD, LSR, LSU, LSAck) are identical.

### 12.2 Decoupling Prefixes from Topology

OSPFv2 Router-LSA carries link descriptions that include subnet IPs:

```
v2 Router-LSA Link:
  Link ID, Link Data, Type, # TOS, Metric
```

OSPFv3 Router-LSA carries only adjacency information (router IDs and metric); prefix information lives in separate **Intra-Area Prefix LSAs** (Type 9) and **Link LSAs** (Type 8):

```
v3 Router-LSA:
  Link Type, Metric, Interface ID, Neighbor Interface ID, Neighbor Router ID
v3 Intra-Area Prefix LSA:
  Referenced LS Type, Referenced LS ID, Referenced Advertising Router
  + list of prefixes (Prefix Length, Prefix Options, Metric, Address)
```

This decoupling enables:
- A single OSPFv3 instance to route multiple address families
- Re-numbering without re-flooding topology
- Smaller Router-LSAs (no addresses)

### 12.3 Per-Link Multi-Instance

OSPFv3 introduces an **Instance ID** field (8 bits) in the common header, allowing multiple independent OSPFv3 instances on the same link:

$$\text{Instances per link} = 2^8 = 256$$

Used for VRF separation, address-family separation, or multi-tenant data centres.

### 12.4 Authentication Removal from Header

OSPFv2 carried 64 bits of in-header authentication (cleartext password or MD5 trailer key-id). OSPFv3 originally relied on IPsec ESP/AH (RFC 4552) and later added an authentication trailer (RFC 7166) for environments where IPsec was operationally heavy.

---

## 13. Authentication Math

### 13.1 OSPFv2 Cleartext (RFC 2328 §D.3)

64-bit shared secret transmitted in-band. Trivially sniffed. Use only when air-gapped or in lab.

### 13.2 OSPFv2 MD5 (RFC 2328 §D.4)

```
Authentication trailer:
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|         Auth Type = 2          |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|  Key ID  | Auth Data Length=16  |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|         Cryptographic Sequence Number (32 bits)            |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

Followed by 16-byte MD5 digest at end of packet:
  digest = MD5(packet_minus_digest || pad || shared_key)
```

MD5 has been broken for collision resistance since 2004 (Wang et al.). For OSPF, the threat is forgery rather than collision, but length-extension attacks against keyed MD5 are well-documented:

$$H(K \| M) \text{ allows attacker knowing } H(K \| M) \text{ to compute } H(K \| M \| M')$$

without knowing $K$. Mitigated by using HMAC instead of plain keyed hash.

### 13.3 OSPFv2 HMAC-SHA (RFC 5709)

HMAC construction:

$$\text{HMAC}(K, M) = H((K \oplus opad) \| H((K \oplus ipad) \| M))$$

Where $ipad = 0x36$ repeated, $opad = 0x5C$ repeated, $H$ is SHA-1, SHA-256, SHA-384, or SHA-512. RFC 5709 mandates HMAC-SHA-256 minimum for new deployments.

Digest sizes:

| Algorithm     | Output bits | Output bytes |
|:--------------|:------------|:-------------|
| HMAC-SHA-1    | 160         | 20           |
| HMAC-SHA-256  | 256         | 32           |
| HMAC-SHA-384  | 384         | 48           |
| HMAC-SHA-512  | 512         | 64           |

### 13.4 OSPFv3 IPsec (RFC 4552)

OSPFv3 packets are protected with IPsec ESP (encryption + integrity) or AH (integrity only) in transport mode. Pre-shared keys with manual SA configuration:

$$\text{SA} = (\text{SPI}, \text{algorithm}, \text{key}, \text{lifetime})$$

ESP NULL encryption + HMAC-SHA-256 is common (integrity but no confidentiality, low overhead).

### 13.5 OSPFv3 Authentication Trailer (RFC 7166)

A simpler alternative: appends an authentication trailer at the end of the OSPFv3 packet, similar in spirit to OSPFv2 RFC 5709. Removes the IPsec dependency at the cost of integrating crypto code into ospfd itself.

```
Authentication Trailer:
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|     Auth Type = HMAC-SHA-x     |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|   Auth Data Len   |   Reserved  |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|    Cryptographic Seq Number (high 32 bits)      |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|    Cryptographic Seq Number (low 32 bits)       |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|    Apad (16 bytes of 0x878FE1F3 repeated)        |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|    HMAC-SHA-N digest                             |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

---

## 14. Comparison to IS-IS

### 14.1 Side-by-Side Mathematics

| Property                         | OSPF                       | IS-IS                          |
|:---------------------------------|:----------------------------|:-------------------------------|
| Algorithm                        | Dijkstra                    | Dijkstra                       |
| Encoding                         | Fixed header + LSAs         | TLV-based                      |
| Layer                            | IP (proto 89)               | OSI Layer 2 (CLNS)             |
| Multi-area                       | Areas + ABR                 | Levels (L1/L2)                 |
| Hierarchy                        | Star (Area 0 + leaves)      | Tree (L1 within L2)            |
| Address family flexibility       | Multi-instance / MT-OSPFv3  | Native multi-topology TLV-based|
| Hello                            | Multicast 224.0.0.5/6       | Multicast 01:80:c2:00:00:14/15 |
| Adjacency                        | Per-AF on v3                | Per-level                      |
| Reflood scope                    | Per-area, controlled by ABR | Per-level, controlled by L1L2  |
| LSA aging                        | 16-bit, MaxAge 3600 s       | Same numeric semantic          |
| Authentication                   | Cleartext / MD5 / HMAC-SHA  | Cleartext / HMAC-MD5 / SHA     |
| Hello interval (default)         | 10 s                        | 10 s (L2 + L1 differ)          |
| Hold time / Dead interval        | 4 × hello = 40 s            | 30 s (default L1L2)            |
| Number of areas a router can be in | 1 (or backbone) + ABR     | 1 (L1 + L2 same router)        |

### 14.2 Why IS-IS Scales Better in SP

- **TLV extensibility**: New TLVs do not require protocol-version bumps. OSPF needs new LSA types (e.g., Opaque LSAs were RFC 2370 / 5250, taking years).
- **No DR election on point-to-point**: OSPF still does P2P adjacency, but DR elections on multi-access add complexity.
- **Cleaner area model**: L1 floods only within an area; L2 floods backbone-wide. The hierarchy is exact, not virtual-link-patched.
- **Smaller flooding scope by default**: An L1L2 router only re-advertises explicit prefix leaks via ATT bit + default route or explicit redistribute.

For ISP-scale topologies (10,000+ routers across many regions), IS-IS dominates; for enterprise and DC, OSPF is more common because tooling and operator familiarity are richer.

### 14.3 Same Algorithm, Different Wire

The SPF computation is identical. Differences are in flooding scope (level vs area), encoding (TLV vs structured LSA), and adjacency (CLNS-based vs IP-based).

---

## 15. Comparison to EIGRP (Distance Vector)

### 15.1 DUAL — Diffusing Update Algorithm

EIGRP is a distance-vector protocol with loop-free guarantees via Garcia-Luna-Aceves's DUAL (1989). Each router maintains, for each destination $D$:

- **Feasible Distance** $\text{FD}(D)$: minimum advertised distance ever seen
- **Reported Distance** $\text{RD}(N, D)$: distance reported by neighbour $N$
- **Feasibility Condition** (FC): $\text{RD}(N, D) < \text{FD}(D)$

A neighbour satisfying the FC is a **Feasible Successor (FS)** — guaranteed loop-free.

```
Successor:        neighbour N with min(RD(N,D) + cost(N))
Feasible Successor: any other N where RD(N,D) < FD(D)
Active state:     no FS exists → query neighbours
Passive state:    FS exists or topology stable
```

### 15.2 Why DUAL Converges Faster Than Naive DV

Distance-vector protocols (RIP) converge in $O(\text{diameter} \times \text{period})$. With period = 30 s and diameter = 15, this is 7.5 minutes worst case. DUAL avoids this by:

1. Triggered updates (no periodic full-table flooding)
2. Loop-free path math (FC) — switch immediately when FS exists
3. Diffusing computation (Active state) only when no FS exists; bounded query/reply chain

For typical DUAL convergence:

$$T_{DUAL} = O(\text{depth of query tree})$$

Often sub-second on local failures, comparable to OSPF + LFA.

### 15.3 OSPF vs EIGRP Trade-offs

| Property             | OSPF (Link-State) | EIGRP (Distance Vector + DUAL) |
|:---------------------|:-------------------|:------------------------------|
| Convergence          | SPF + flood        | DUAL diffusing computation    |
| Topology visibility  | Full LSDB          | Only neighbour distances      |
| Memory               | $O(R + E)$         | $O(\text{neighbours} \times \text{prefixes})$ |
| Multi-vendor         | yes (IETF)         | Cisco-proprietary until 2013, then RFC 7868 |
| Unequal-cost ECMP    | no                 | yes (variance)                |
| Hierarchical design  | Areas               | Stub routers (limited)        |
| Scaling              | 100s-1000s         | 100s typically                |

---

## 16. Convergence Models

### 16.1 Markov Chain of OSPF States

Model an OSPF router's adjacency as a continuous-time Markov chain:

```
States:    Init  ↔  Two-Way  ↔  ExStart  ↔  Exchange  ↔  Loading  ↔  Full
Failures:  Down ←─ Down ←──── Down ←──── Down ←──── Down ←──── Down
```

With failure rate $\lambda$ and recovery rate $\mu$:

$$\pi_{Full} = \frac{\mu}{\mu + \lambda}$$

For $\lambda^{-1} = $ MTBF = $10^6$ s (11.5 days) and $\mu^{-1} = $ MTTR = 60 s (full re-adjacency):

$$\pi_{Full} = \frac{1/60}{1/60 + 1/10^6} \approx 0.99994$$

Availability ≈ 99.994% — five 9s requires either redundant adjacencies (ECMP) or dramatically faster MTTR.

### 16.2 Mean Time to Convergence (MTTC)

For a network of $N$ routers with Poisson failure arrivals at rate $\lambda$ per router:

$$E[T_{convergence}] = T_{detect} + T_{flood} + T_{SPF} + T_{FIB}$$

If we assume failures are independent:

$$\Pr[\text{convergence within } t] = 1 - e^{-t/E[T]}$$

For a typical large enterprise with $E[T] = 200$ ms, the probability of convergence within 1 s is $1 - e^{-5} \approx 99.3\%$.

### 16.3 Micro-Loop Probability

A micro-loop forms when two routers have inconsistent FIBs during reconvergence. The probability is bounded by the time window during which their FIBs differ:

$$\Pr[\text{loop}] \leq \frac{|t_{FIB,A} - t_{FIB,B}|}{\text{flow-arrival-period}}$$

For a 100 Gbps link with 10 µs packet inter-arrival and 50 ms FIB drift between routers, expected packets caught in a loop $\approx 5000$. Mitigations:

- Ordered FIB updates (downstream-first): RFC 6976
- TI-LFA: avoids the loop entirely by detour SR labels
- IP-FRR with explicit microloop avoidance timers

### 16.4 LSA Storm Models

Under cascading failure, LSA generation rate from $N$ routers, each with link-state thrashing at rate $r$, is bounded:

$$\text{LSA rate} \leq N \times \text{min}(r, 1/T_{originate})$$

LSA generation throttle ($T_{originate} = 5000$ ms) ensures any one router cannot exceed 0.2 LSA/s. For $N = 1000$, the area sees at most 200 LSAs/s — bounded and tractable.

### 16.5 Convergence Distribution Under Random Failures

Assume a network with $N$ routers, each with adjacency MTBF $M$ seconds. Failures arrive as a Poisson process with rate:

$$\lambda_{network} = \frac{N \times \bar{d}}{2M}$$

Where $\bar{d}$ is the average node degree. For $N = 200$, $\bar{d} = 4$, $M = 10^7$ s ($\sim$4 months):

$$\lambda_{network} = \frac{200 \times 4}{2 \times 10^7} = 4 \times 10^{-5} \text{ /s}$$

Mean time between failures across the network: $\sim 25{,}000$ s $\approx 7$ hours. With per-failure convergence time $T_{conv} = 200$ ms, the network spends:

$$\frac{T_{conv}}{1/\lambda_{network}} = \frac{0.2}{25000} = 8 \times 10^{-6}$$

i.e. $\sim 10^{-5}$ fraction of the time in transient state — about 5 nines availability before any redundancy or fast-reroute is applied.

### 16.6 Heavy-Tailed Failure Models

Real-world failure inter-arrival times are not strictly Poisson. Empirical studies (Markopoulou et al. 2008, Sigcomm) show:

- Hardware failures: Weibull distribution with shape parameter $k \approx 0.6$ (heavy tail)
- Software bugs: bursty, often clustered (correlated failures)
- Operator-induced: pareto-distributed clustering during change windows

For Weibull-distributed inter-arrival times:

$$\Pr[T < t] = 1 - e^{-(t/\eta)^k}$$

With $k < 1$, failures are clustered — periods of stability punctuated by storms. OSPF's exponential SPF throttle is designed precisely for this: bound CPU during storms, fast convergence when calm.

### 16.7 Stability of Re-convergence

A re-convergence event is **stable** if no new event arrives before SPF + FIB completes. Define:

$$P_{stable} = \Pr[\text{inter-arrival} > T_{conv}] = e^{-\lambda T_{conv}}$$

For $\lambda = 10^{-3}$ /s (one event/15 minutes) and $T_{conv} = 200$ ms: $P_{stable} \approx 0.9998$. Catastrophic instability requires $\lambda T_{conv} \gtrsim 1$, i.e. failure inter-arrival comparable to convergence time. This happens during cascade failures (data centre power events, BGP-induced LSA churn) and is the regime SPF throttle damps.

---

## 17. Segment Routing Extensions (RFC 8665)

### 17.1 SR-MPLS Label Allocation

Each router advertises an **SRGB** (Segment Routing Global Block) — a label range:

$$\text{SRGB} = [\text{SRGB-base}, \text{SRGB-base} + \text{SRGB-size} - 1]$$

A typical SRGB: $[16000, 23999]$, providing 8000 indices.

Each prefix gets a **Prefix-SID** (an index into the SRGB):

$$\text{Label}(prefix) = \text{SRGB-base}_{router} + \text{Prefix-SID-index}_{prefix}$$

Different routers can have different SRGB bases (operationally common after merger or renumbering), but the same Prefix-SID index. Each router computes its own label for the prefix using its local SRGB.

### 17.2 Adjacency-SIDs

A router advertises a unique label per adjacency. Adjacency-SIDs are local-only (not in the SRGB) and provide explicit hop-by-hop steering:

$$\text{Adj-SID-Label} \in \text{SRLB} = [\text{SRLB-base}, \text{SRLB-base + size}]$$

Typical SRLB: $[24000, 39999]$.

### 17.3 SR Extensions to Router-LSA

Carried in Opaque LSA Type 10 (Router Information LSA). New TLVs:

```
TLV 8: SID/Label Range
  - sub-TLV 1: SID/Label sub-TLV (SRGB-base, range)
TLV 9: Adjacency SID Sub-TLV (Type-10 Extended Link LSA)
TLV 10: LAN Adjacency SID Sub-TLV
```

### 17.4 Worked Example — SR Label Stack for a 3-Hop Path

```
Topology:        SRGB on each:
  R1 → R2 → R3 → R4
                 R1: [16000, 23999]
                 R2: [16000, 23999]
                 R3: [17000, 24999]   (different base!)
                 R4: [16000, 23999]

Prefix-SID indices (advertised in OSPF Opaque Type 10):
  R1.loopback: index = 1   (label on R1 = 16001, on R3 = 17001)
  R4.loopback: index = 4   (label on R1 = 16004, on R3 = 17004)

R1 wants to send to R4 via the chosen SR path:
  Label stack (top → bottom):
    [16004]   (Prefix-SID for R4 in R1's local SRGB)
  R1 swaps 16004 → 16004 → push to R2
  R2 swaps 16004 → 17004 (different base!) → push to R3
  R3 swaps 17004 → 16004 → push to R4
  R4 receives, PHP pops, delivers to loopback
```

This per-hop swap math is why every router publishes its SRGB and computes labels using its neighbours' SRGBs. OSPF carries the SRGB and Prefix-SID in Opaque Type 10 LSAs (RFC 8665).

### 17.5 Flex-Algo (RFC 9350)

An algorithm definition encoded in OSPF:

```
Flex-Algo Definition (FAD) TLV:
  Algorithm ID (128-255)
  Metric Type: IGM | min-delay | TE-metric
  Calc Type: 0 = SPF
  Priority
  Exclude Admin Group
  Include-Any Admin Group
  Include-All Admin Group
  Exclude SRLG
```

Each router runs:

$$\text{SPT}_{algo} = \text{Dijkstra}(G_{filtered}, \text{metric}_{algo})$$

Producing a per-algorithm forwarding table. Use cases:
- Disjoint paths (low-latency vs high-bandwidth)
- Compliance routing (avoid certain links/paths)
- Multi-tenant SR-TE without RSVP

---

## 18. Real-World Performance Numbers

### 18.1 SPF Runtime

| Platform               | Area size | SPF time (full) | SPF time (iSPF) |
|:-----------------------|:----------|:-----------------|:----------------|
| Cisco IOS XR ASR9000   | 10,000    | 50-100 ms        | 5-10 ms         |
| Juniper MX (Trio)      | 10,000    | 80-150 ms        | 5-15 ms         |
| Arista EOS 7280        | 5,000     | 20-50 ms         | 2-5 ms          |
| FRR on Linux x86       | 10,000    | 100-300 ms       | 5-20 ms         |
| Cisco Cat9k (campus)   | 1,000     | 5-15 ms          | 1-3 ms          |

### 18.2 LSDB Memory Footprint

| Network                              | LSDB size | Notes                                |
|:-------------------------------------|:----------|:-------------------------------------|
| Branch office (5 routers, stub)      | < 10 KB   | Trivial                              |
| Campus (200 routers, single area)    | 200-500 KB| With ~50 prefixes/router             |
| Large enterprise (1000 routers)      | 5-20 MB   | Multi-area, heavy redistribution     |
| Tier-1 ISP backbone (5000 routers)   | 50-500 MB | IGP only; BGP separate               |
| Hyperscaler DC (24,000 leaves)       | 100 MB-1 GB | Often EBGP instead of OSPF for this scale |

### 18.3 Convergence Targets

| Network type            | Target convergence | Mechanism                          |
|:------------------------|:--------------------|:-----------------------------------|
| Enterprise campus       | < 1 s               | Default OSPF + BFD                 |
| Service-provider core   | < 200 ms            | LFA + TI-LFA + BFD                 |
| Financial / HFT         | < 50 ms             | TI-LFA + hardware-assisted BFD     |
| Hyperscale DC           | < 100 ms            | EBGP / OSPF-3 + BFD echo           |
| 5G transport            | < 50 ms             | TI-LFA + 802.1ag CCM               |

### 18.4 LSA Volume in Production

Telia's IS-IS L2 (proxy for OSPF backbone scale) carried roughly 30k LSPs as of 2020. Comcast's OSPF backbone is similar. Internal data-centre OSPF rarely exceeds 5k LSAs because EBGP unnumbered handles fabric routing in most modern designs.

---

## 19. Failure Mode Analysis

### 19.1 Asymmetric Link Failure

A unidirectional failure (RX works, TX doesn't, or vice versa) prevents Hellos in one direction. Without BFD-aware OSPF or asymmetric detection:

```
Router A → Router B: Hellos arrive, neighbour seen as Two-Way
Router B → Router A: Hellos do not arrive
Router A: Dead Interval expires → adjacency torn down
Router B: until A's hellos are seen as silent → Dead Interval expires
```

OSPFv2 mitigation: Hello includes the list of seen neighbours. If A doesn't see itself in B's neighbour list, A regresses to Init state.

### 19.2 Routing Loops During Re-convergence

Two routers $R_1$ and $R_2$ both converge on an SPF result, but $R_1$ programs its FIB at $t_1$ and $R_2$ at $t_2$. For $t \in [t_1, t_2]$, traffic from $R_1$'s domain to $R_2$'s domain may loop if their FIBs disagree. Mitigations:

- **Ordered FIB**: program downstream nodes first (RFC 6976)
- **PLSN (Path Locking via SNTP)**: time-synchronised activation
- **TI-LFA**: explicit detour avoids transient inconsistency entirely

### 19.3 LSA Storm Causes

Common triggers:
- Flapping interface (port-channel imbalance, optic going bad)
- Memory pressure on router causing BGP/OSPF process restart
- Software bug re-originating LSAs every iteration
- Misconfigured route redistribution churning Type-5 LSAs

Defences:
- LSA throttle (origination)
- Damping on specific neighbours
- Operator alarms when LSA-rate > baseline + 3σ

### 19.4 LSDB Memory Exhaustion

$$\text{Memory budget} \geq \text{LSDB} + \text{RIB} + \text{FIB} + \text{kernel overhead}$$

When LSDB exceeds available memory:
- Linux/FRR: OOM kill ospfd, adjacencies drop, full-area flap on restart
- IOS: forced reload, traffic black-hole during reload
- IOS XR: graceful protocol-level back-off, partial functionality

Recommended sizing: 4-8 GB control-plane RAM for any area > 1000 routers; 16+ GB for SP backbones.

---

## 20. Tools for OSPF Analysis

### 20.1 Packet-Level Inspection

```bash
# Capture all OSPF on any interface
tcpdump -i any -n proto ospf

# Wireshark filter for Hello packets only
ospf.msg == 1

# Decode LSAs for a specific area
ospf.area_id == 0.0.0.0 and ospf.lsa.type == 1

# Find LSA with high age (potential stuck LSA)
ospf.lsa.age > 3000
```

### 20.2 FRR / Quagga Debug Commands

```
debug ospf packet all
debug ospf event
debug ospf lsa
debug ospf nsm
debug ospf zebra
show ip ospf database
show ip ospf neighbor detail
show ip ospf interface
show ip ospf statistics       # SPF runs, LSA rate
```

### 20.3 Cisco IOS XR Show Commands

```
show ospf process
show ospf summary
show ospf database router
show ospf flood-list
show ospf trace all
show ospf request-list all
show ospf retransmission-list
```

### 20.4 Looking-Glass / Topology Tools

- **BIRD Lab**: software router topology test bench
- **Quagga / FRR labs in containerlab**: Docker-based topology emulation
- **GNS3 / EVE-NG**: VM-based router emulation
- **Junosphere / Cisco VIRL** (now CML): vendor-blessed sims
- **BGPalerter / ExaBGP**: monitor and inject route events
- **YANG-based streaming telemetry**: real-time LSA-rate, SPF-time metrics
- **NetReplica / Containerlab + FRR**: precise OSPF topologies in seconds

### 20.5 Topology Visualisation

```
networkx (Python):
  Parse LSDB → build NetworkX graph → visualise with matplotlib / pygraphviz
  Compute betweenness, centrality, articulation points

batfish:
  Parse vendor configs → build full topology model
  Run "what-if" SPF computations offline
```

---

## Prerequisites

- Graph theory (vertices, edges, shortest paths, spanning trees)
- Discrete mathematics (combinatorics for flooding bounds)
- Probability / Markov chains (convergence analysis)
- Linear algebra (used minimally for SPF on adjacency matrices)
- Basic cryptographic primitives (HMAC, hash functions)
- TCP/IP fundamentals (the protocol runs on IP proto 89 / IPv6 next-header 89)

## Complexity Summary

| Operation                          | Time                  | Space           |
|:-----------------------------------|:----------------------|:----------------|
| Full SPF (Dijkstra binary heap)    | $O((V + E) \log V)$  | $O(V + E)$      |
| Incremental SPF (single-link)      | $O(|\Delta| \log V)$ | $O(V)$          |
| LSA flooding to area               | $O(N_{routers})$ msgs | $O(\text{LSDB})$|
| Best-path lookup (RIB → FIB)       | $O(\log N)$ tries    | $O(N)$          |
| LSA refresh                        | $O(L / 1800)$ /s     | $O(L)$          |
| Hello processing                   | $O(\text{neighbours})$| $O(1)$         |
| HMAC-SHA-256 on packet (1500 B)    | ~10 µs               | $O(1)$          |
| MD5 (legacy) on packet (1500 B)    | ~3 µs                | $O(1)$          |

## References

- **RFC 2328** — *OSPF Version 2* (J. Moy, 1998) — the foundational specification
- **RFC 5340** — *OSPF for IPv6* (R. Coltun, D. Ferguson, J. Moy, A. Lindem, 2008)
- **RFC 5250** — *The OSPF Opaque LSA Option* (Berger, Bryskin, Zinin, Coltun, 2008)
- **RFC 4503** / Vendor docs — SPF Throttling (initial-delay / hold-time / max-wait)
- **RFC 5709** — *OSPFv2 HMAC-SHA Cryptographic Authentication* (Bhatia, Manral, Fanto, White, Barnes, 2009)
- **RFC 5838** — *Support of Address Families in OSPFv3* (Lindem, Mirtorabi, Roy, Barnes, 2010)
- **RFC 4552** — *Authentication/Confidentiality for OSPFv3 via IPsec* (Gupta, Melam, 2006)
- **RFC 7166** — *Supporting Authentication Trailer for OSPFv3* (Bhatia, Hartman, Zhang, Lindem, 2014)
- **RFC 5286** — *Basic Specification for IP Fast Reroute (LFA)* (Atlas, Zinin, 2008)
- **RFC 6976** — *Framework for Loop-Free Convergence Using OFRR* (Shand, Bryant, Previdi, Filsfils, Atlas, 2013)
- **RFC 7770** — *Router Information LSA* (Lindem, Shen, Vasseur, Aggarwal, Shaffer, 2016)
- **RFC 7868** — *Cisco's Enhanced Interior Gateway Routing Protocol (EIGRP)* (Savage et al., 2016)
- **RFC 8665** — *OSPF Extensions for Segment Routing* (Psenak et al., 2019)
- **RFC 9166** — *TI-LFA: Topology Independent Loop-Free Alternate* (Bashandy et al., 2022)
- **RFC 9350** — *IGP Flexible Algorithm* (Psenak, Hegde, Filsfils, Talaulikar, Gulko, 2022)
- **RFC 6790** — *Use of Entropy Labels in MPLS* (Kompella, Drake, Amante, Henderickx, Yong, 2012)
- John T. Moy — *OSPF: Anatomy of an Internet Routing Protocol* (Addison-Wesley, 1998)
- Jeff Doyle, Jennifer Carroll — *Routing TCP/IP, Volume I, Second Edition* (Cisco Press, 2005)
- Russ White, Alvaro Retana — *OSPF and IS-IS: Choosing an IGP for Large-Scale Networks* (Addison-Wesley, 2005)
- E. W. Dijkstra — *A Note on Two Problems in Connexion with Graphs*, Numerische Mathematik 1, 269-271 (1959)
- Garcia-Luna-Aceves — *Loop-Free Routing Using Diffusing Computations*, IEEE/ACM ToN, 1993
- Cormen, Leiserson, Rivest, Stein — *Introduction to Algorithms (CLRS)*, Chapter 24, "Single-Source Shortest Paths"
- Tarjan — *Data Structures and Network Algorithms* (SIAM, 1983) — Fibonacci-heap analysis
- Fredman, Tarjan — *Fibonacci Heaps and Their Uses in Improved Network Optimization Algorithms*, JACM 34(3), 1987

---

*Every OSPF router is solving Dijkstra's 1959 problem hundreds of times per day, on silicon at line rate, while a Markov chain of failure events plays out beneath it. The math says the protocol converges; the timers say how fast; the LSDB says how big a network it can hold; the authentication trailer says how hard a forgery has to work. None of these are abstractions — they are the running cost of every BGP-free hop on the modern internet.*
