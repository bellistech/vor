# MPLS TE — Traffic Engineering Architecture and RSVP-TE Signaling

> *Traffic Engineering in MPLS replaces hop-by-hop shortest-path routing with constraint-based source routing, where the headend router computes an explicit path that satisfies bandwidth, affinity, and metric constraints, then signals it end-to-end using RSVP-TE. The math covers CSPF complexity, bandwidth allocation models, preemption priority combinatorics, convergence timing, and label stack overhead in TE contexts.*

---

## 1. Constraint-Based Routing — Graph Theory Foundation

### The Problem

Traditional IGP routing uses Dijkstra's algorithm on the full topology graph. CSPF (Constrained SPF) modifies this by first pruning the graph, then running Dijkstra on the reduced topology. The question is: how does constraint pruning affect path computation complexity?

### The Topology Graph

Model the network as a directed graph $G = (V, E)$ where:
- $V$ = set of vertices (routers), $|V| = n$
- $E$ = set of edges (links), $|E| = m$
- Each edge $e \in E$ has attributes: bandwidth $B(e)$, TE metric $w_{TE}(e)$, IGP metric $w_{IGP}(e)$, admin-group bitmask $A(e)$, SRLG set $S(e)$

### CSPF Pruning Phase

Given a tunnel request with constraints:
- Required bandwidth: $B_{req}$ at setup priority $p$
- Affinity: value $a$, mask $m$
- Excluded SRLGs: set $S_{excl}$

The pruned graph $G' = (V, E')$ where:

$$E' = \{ e \in E \mid B_{unrsv}(e, p) \geq B_{req} \wedge (A(e) \wedge m) = (a \wedge m) \wedge S(e) \cap S_{excl} = \emptyset \}$$

Where $B_{unrsv}(e, p)$ is the unreserved bandwidth on edge $e$ at priority $p$.

### Complexity Analysis

Standard Dijkstra: $O(m + n \log n)$ with a Fibonacci heap.

CSPF adds a pruning pass: $O(m)$ to iterate all edges and check constraints.

Total CSPF: $O(m + n \log n)$ — same asymptotic complexity as Dijkstra, since pruning is linear.

However, the pruned graph $G'$ has $|E'| \leq |E|$, so the Dijkstra phase runs on a smaller graph. In practice, aggressive constraint pruning reduces computation time.

### CSPF Tiebreaking

When multiple equal-cost paths exist after CSPF, tiebreakers are applied:

| Tiebreaker | Strategy | Formula |
|:---|:---|:---|
| Random | Uniform random selection | $P(\text{path}_i) = \frac{1}{k}$ where $k$ = equal-cost paths |
| Least-fill | Choose path with maximum minimum available bandwidth | $\arg\max_p \min_{e \in p} B_{unrsv}(e, p)$ |
| Most-fill | Choose path with minimum maximum available bandwidth | $\arg\min_p \max_{e \in p} B_{unrsv}(e, p)$ |

Least-fill spreads tunnels across the network. Most-fill packs tunnels onto already-utilized links to preserve spare capacity on other paths.

---

## 2. RSVP-TE Signaling — Message Flow and State

### Signaling Sequence

The headend (router $R_0$) signals an LSP through transit routers $R_1, R_2, \ldots, R_{k-1}$ to tailend $R_k$.

**PATH phase (downstream):**

$$R_0 \xrightarrow{\text{PATH}} R_1 \xrightarrow{\text{PATH}} R_2 \xrightarrow{\text{PATH}} \cdots \xrightarrow{\text{PATH}} R_k$$

**RESV phase (upstream):**

$$R_k \xrightarrow{\text{RESV}} R_{k-1} \xrightarrow{\text{RESV}} \cdots \xrightarrow{\text{RESV}} R_1 \xrightarrow{\text{RESV}} R_0$$

Total signaling time for initial setup:

$$T_{setup} = \sum_{i=0}^{k-1} d(R_i, R_{i+1}) + \sum_{i=k}^{1} d(R_i, R_{i-1}) + \sum_{i=0}^{k} t_{proc}(R_i)$$

Where $d(R_i, R_j)$ is the propagation delay between routers and $t_{proc}$ is per-router processing time.

For a path of $k$ hops with average link delay $\bar{d}$ and average processing time $\bar{t}$:

$$T_{setup} \approx 2k\bar{d} + (k+1)\bar{t}$$

### Worked Example

Path: 5 hops, average link delay 2ms, average processing time 1ms.

$$T_{setup} \approx 2(5)(2) + (6)(1) = 20 + 6 = 26 \text{ ms}$$

For a transcontinental path: 10 hops, average delay 15ms, processing 2ms:

$$T_{setup} \approx 2(10)(15) + (11)(2) = 300 + 22 = 322 \text{ ms}$$

### RSVP State Scaling

Each transit router maintains per-tunnel state (PATH and RESV state blocks). For a network with $T$ tunnels, each traversing an average of $h$ hops:

$$S_{total} = T \times h \times s_{per\_tunnel}$$

Where $s_{per\_tunnel}$ is the memory per tunnel per hop (typically 1-4 KB depending on implementation).

| Tunnels ($T$) | Avg Hops ($h$) | State per Hop | Total State |
|:---:|:---:|:---:|:---:|
| 100 | 5 | 2 KB | 1 MB |
| 1,000 | 5 | 2 KB | 10 MB |
| 10,000 | 5 | 2 KB | 100 MB |
| 50,000 | 8 | 2 KB | 800 MB |

RSVP state scales as $O(T \times h)$, which is why large TE deployments can stress midpoint routers.

---

## 3. Soft State and Refresh Overhead

### The Refresh Model

RSVP uses soft state: each PATH and RESV message must be periodically refreshed. If a refresh is missed for too long, state is torn down.

Cleanup timeout:

$$T_{cleanup} = (K + 0.5) \times 1.5 \times R$$

Where:
- $R$ = refresh interval (default 30 seconds)
- $K$ = allowed missed refreshes (default 3, giving $K + 0.5 = 3.5$)

$$T_{cleanup} = 3.5 \times 1.5 \times 30 = 157.5 \text{ seconds}$$

### Refresh Message Rate

Each tunnel generates 2 refresh messages per hop per refresh interval (one PATH, one RESV). For $T$ tunnels, each $h$ hops, with refresh interval $R$:

$$M_{refresh} = \frac{2 \times T \times h}{R} \text{ messages/second across the network}$$

Per-router rate for a router traversed by $T_r$ tunnels:

$$M_{router} = \frac{2 \times T_r}{R}$$

| Tunnels per Router | Refresh Interval | Messages/sec |
|:---:|:---:|:---:|
| 100 | 30s | 6.7 |
| 1,000 | 30s | 66.7 |
| 10,000 | 30s | 666.7 |
| 10,000 | 300s (Summary Refresh) | 66.7 |

### RSVP Summary Refresh (RFC 2961)

Summary Refresh bundles multiple tunnel state IDs into a single message, reducing message count by 10-100x. Instead of one PATH + one RESV per tunnel per interval, a single summary message covers $n$ tunnels.

Effective message rate with summary refresh bundling $n$ states per message:

$$M_{summary} = \frac{2 \times T_r}{R \times n}$$

For 10,000 tunnels, 30s refresh, bundles of 100: $\frac{20000}{30 \times 100} = 6.67$ messages/sec (vs 666.7 without).

---

## 4. Bandwidth Allocation and Priority Preemption

### The Priority Model

RSVP-TE defines 8 priority levels (0-7). Each link tracks unreserved bandwidth at each priority:

$$B_{unrsv}(e, p) = B_{max\_rsv}(e) - \sum_{t : \text{hold}(t) \leq p} B(t)$$

Where the sum is over all tunnels $t$ whose hold priority is numerically less than or equal to $p$ (i.e., higher or equal priority).

### Preemption

Tunnel $t_{new}$ with setup priority $p_s$ can preempt tunnel $t_{old}$ with hold priority $p_h$ if:

$$p_s < p_h$$

(Numerically lower = higher priority.)

### Priority Combinations

Setup priority must be $\leq$ hold priority (numerically: setup $\geq$ hold). Valid combinations for one tunnel:

$$C = \sum_{p_h=0}^{7} (8 - p_h) = 8 + 7 + 6 + 5 + 4 + 3 + 2 + 1 = 36$$

So there are 36 valid (setup, hold) priority pairs.

### Bandwidth Accounting Example

Link with 1 Gbps maximum reservable bandwidth. Three tunnels:

| Tunnel | Bandwidth | Setup | Hold |
|:---:|:---:|:---:|:---:|
| A | 300 Mbps | 3 | 3 |
| B | 400 Mbps | 5 | 5 |
| C | 500 Mbps | 2 | 2 |

Unreserved bandwidth at each priority:

| Priority | Tunnels Counted (hold $\leq$ p) | Reserved | Unreserved |
|:---:|:---|:---:|:---:|
| 0 | none | 0 | 1000 Mbps |
| 1 | none | 0 | 1000 Mbps |
| 2 | C (hold=2) | 500 | 500 Mbps |
| 3 | C, A (hold=2,3) | 800 | 200 Mbps |
| 4 | C, A | 800 | 200 Mbps |
| 5 | C, A, B (hold=2,3,5) | 1200 | -200 Mbps (oversubscribed at this priority) |
| 6 | C, A, B | 1200 | -200 Mbps |
| 7 | C, A, B | 1200 | -200 Mbps |

At priority 5, the link is oversubscribed. Tunnel C (setup=2) can preempt B (hold=5) since $2 < 5$.

---

## 5. CSPF with Administrative Weight (TE Metric)

### Dual Metric Model

Each link carries two independent metrics:

$$w_{IGP}(e): \text{used by OSPF/IS-IS for IP forwarding}$$
$$w_{TE}(e): \text{used by CSPF for TE tunnel path computation}$$

When TE metric is not explicitly configured: $w_{TE}(e) = w_{IGP}(e)$.

### Path Cost

For a path $P = (e_1, e_2, \ldots, e_k)$:

$$C_{TE}(P) = \sum_{i=1}^{k} w_{TE}(e_i)$$

CSPF selects: $P^* = \arg\min_P C_{TE}(P)$ subject to constraints.

### Worked Example: Divergent Metrics

Network with three paths from A to D:

**Path 1: A-B-D**
- IGP: $10 + 10 = 20$ (preferred by IGP)
- TE: $50 + 50 = 100$

**Path 2: A-C-D**
- IGP: $30 + 30 = 60$
- TE: $10 + 10 = 20$ (preferred by CSPF)

**Path 3: A-E-D**
- IGP: $15 + 15 = 30$
- TE: $25 + 25 = 50$

Result: Normal IP traffic uses Path 1 (lowest IGP cost). TE tunnels use Path 2 (lowest TE cost). This separation allows TE tunnels to prefer links that normal routing avoids (e.g., higher-capacity fiber with higher IGP cost).

---

## 6. Admin Groups — Bitmask Algebra

### The Constraint Model

Each link has a 32-bit attribute-flags field: $A(e) \in \{0, 1\}^{32}$.

Each tunnel specifies affinity $a$ and mask $m$, both 32-bit.

A link $e$ satisfies the affinity constraint iff:

$$(A(e) \wedge m) = (a \wedge m)$$

Where $\wedge$ is bitwise AND.

### Truth Table (per bit position)

| Mask bit | Affinity bit | Link bit | Result |
|:---:|:---:|:---:|:---|
| 0 | X | X | Don't care (bit ignored) |
| 1 | 0 | 0 | Match (link lacks property, tunnel excludes it) |
| 1 | 0 | 1 | No match |
| 1 | 1 | 0 | No match |
| 1 | 1 | 1 | Match (link has property, tunnel requires it) |

### Example: Link Coloring

Define colors: RED=bit0, BLUE=bit1, GREEN=bit2.

Constraint: "Include only links that are RED, don't care about other colors."

$$a = 0\text{x}00000001, \quad m = 0\text{x}00000001$$

| Link | Flags | $(A \wedge m)$ | $(a \wedge m)$ | Match? |
|:---|:---:|:---:|:---:|:---:|
| L1: RED+BLUE | 0x3 | 0x1 | 0x1 | Yes |
| L2: BLUE only | 0x2 | 0x0 | 0x1 | No |
| L3: RED+GREEN | 0x5 | 0x1 | 0x1 | Yes |
| L4: none | 0x0 | 0x0 | 0x1 | No |

### Extended Affinity (RFC 7308)

When 32 bits are insufficient, extended admin groups use an arbitrary-length bitmask. The matching formula is identical but applied to a larger bit vector. This is relevant in networks with more than 32 distinct link categories.

---

## 7. Fast Reroute — Convergence Timing

### Protection Switching Time

FRR provides local repair at the Point of Local Repair (PLR) without waiting for headend re-optimization. The failover time:

$$T_{FRR} = T_{detect} + T_{switch}$$

Where:
- $T_{detect}$: failure detection time (hardware: <10ms, BFD: 50-150ms, RSVP Hello: seconds)
- $T_{switch}$: time to activate backup path and swap labels (<10ms for pre-installed backup)

### Failure Detection Methods

| Method | $T_{detect}$ | Formula |
|:---|:---:|:---|
| Loss of Light (hardware) | 1-10 ms | Physical layer signal |
| BFD (Bidirectional Forwarding Detection) | $T_{BFD} = \text{interval} \times \text{multiplier}$ | e.g., $50\text{ms} \times 3 = 150\text{ms}$ |
| RSVP Hello | $T_{hello} = \text{interval} \times \text{miss\_count}$ | e.g., $3\text{s} \times 3.5 = 10.5\text{s}$ |
| IGP Adjacency | $T_{IGP} = \text{dead\_interval}$ | OSPF: 40s default, IS-IS: 30s default |

### Facility Backup Label Stack

During FRR with facility backup, the PLR pushes an additional label (the backup tunnel label) onto the protected packet:

Original stack at PLR: $[L_{transport}]$

After FRR activation: $[L_{backup}, L_{transport}]$

At the Merge Point (MP), the backup label is popped (via PHP or explicit pop), and the original transport label is intact. The packet continues on the original LSP as if nothing happened.

Stack depth during FRR = original stack depth + 1. MTU impact:

$$MTU_{FRR} = MTU_{original} - 4 \text{ bytes}$$

### Facility vs. Detour Scalability

For a network with $T$ tunnels and $L$ links to protect:

| Metric | Facility Backup | One-to-One Detour |
|:---|:---|:---|
| Backup LSPs needed | $L$ (one per protected link/node) | $T \times L$ (one per tunnel per protected resource) |
| State at PLR | $O(L)$ | $O(T \times L)$ |
| Bandwidth control | Shared (all tunnels share backup BW) | Per-tunnel (each detour reserves its own BW) |

Example: 1,000 tunnels, 50 protected links:
- Facility: 50 backup tunnels
- Detour: up to 50,000 detour LSPs

---

## 8. Auto-Bandwidth — Control Theory

### The Feedback Loop

Auto-bandwidth implements a feedback control loop:

1. **Measure:** Sample traffic rate on the tunnel at interval $T_s$ (e.g., every 300 seconds)
2. **Filter:** Apply adjustment threshold to suppress noise
3. **Act:** Re-signal tunnel with new bandwidth

### Adjustment Decision

Let $B_{current}$ = current reserved bandwidth, $R_{measured}$ = measured traffic rate, $\theta$ = adjustment threshold (percentage).

Resize if:

$$\frac{|R_{measured} - B_{current}|}{B_{current}} > \frac{\theta}{100}$$

New bandwidth (clamped to min/max):

$$B_{new} = \max(B_{min}, \min(B_{max}, R_{measured}))$$

### Overflow Detection

Overflow triggers immediate resize (bypassing normal interval). If traffic exceeds current bandwidth by more than overflow threshold $\phi$ for $k$ consecutive samples:

$$\text{Overflow if: } R_{measured} > B_{current} \times (1 + \frac{\phi}{100}) \text{ for } k \text{ consecutive samples}$$

### Stability Considerations

Without dampening, auto-bandwidth can oscillate between two values. The adjustment threshold $\theta$ acts as a deadband:

- Too small $\theta$ (<5%): frequent re-signaling, RSVP churn
- Too large $\theta$ (>50%): under-utilization or over-subscription for extended periods
- Practical range: 10-20%

### Worked Example

Tunnel with auto-bw: $B_{min}=10$ Mbps, $B_{max}=500$ Mbps, $\theta=10\%$, current $B=100$ Mbps.

| Measured Rate | Change | Exceeds 10%? | Action | New BW |
|:---:|:---:|:---:|:---|:---:|
| 105 Mbps | +5% | No | No change | 100 |
| 115 Mbps | +15% | Yes | Resize | 115 |
| 112 Mbps | -2.6% (from 115) | No | No change | 115 |
| 80 Mbps | -30.4% | Yes | Resize | 80 |
| 5 Mbps | -93.75% | Yes | Clamp to min | 10 |
| 600 Mbps | +5900% | Yes | Clamp to max | 500 |

---

## 9. TE Load Balancing — Traffic Distribution

### Equal-Cost TE Paths

When $k$ tunnels with equal load-share values terminate at the same destination, traffic is distributed via hashing:

$$\text{tunnel}(flow) = hash(flow\_key) \mod k$$

Where $flow\_key$ typically includes src-IP, dst-IP, protocol, src-port, dst-port.

### Weighted Distribution

With load-share values $w_1, w_2, \ldots, w_k$:

$$P(\text{tunnel}_i) = \frac{w_i}{\sum_{j=1}^{k} w_j}$$

### Example

Three tunnels: $w_1 = 5, w_2 = 3, w_3 = 2$. Total = 10.

$$P_1 = 50\%, \quad P_2 = 30\%, \quad P_3 = 20\%$$

### Hash Polarization

If the same hash function is used at multiple points in the network, traffic that was load-balanced at one hop may always select the same link at the next hop. Solutions:
- Use different hash seeds at each hop
- Include a tunnel-specific salt in the hash
- Use adaptive hashing (e.g., flowlet-based)

---

## 10. TE Database (TED) — Flooding and Convergence

### IGP-TE Flooding

When a link attribute changes (bandwidth reserved/released, link up/down, metric change), the IGP floods an updated LSA/LSP to all routers. The flooding reaches all $n$ routers in:

$$T_{flood} \approx D_{network} \times \bar{d}$$

Where $D_{network}$ is the network diameter (longest shortest path in hops) and $\bar{d}$ is average per-hop propagation + processing delay.

### TED Update Rate

Each bandwidth change triggers a flood. With $T$ tunnels and average lifetime $L$, the bandwidth event rate per link:

$$R_{events} \approx \frac{2 \times T_{link}}{L}$$

Where $T_{link}$ is the number of tunnels traversing the link and factor 2 accounts for setup + teardown.

### Flooding Dampening

To prevent excessive IGP flooding, implementations use thresholds:
- Flood only if unreserved BW changes by more than a configured percentage (e.g., 10%)
- Rate-limit floods to at most one per $T_{hold}$ seconds (e.g., 5 seconds)

This reduces flooding rate but introduces TED staleness. CSPF may compute paths based on outdated bandwidth information, leading to signaling failures (RSVP PathErr with "admission control failure"). The headend then re-computes with updated TED.

---

## 11. ERO Processing — Strict and Loose Hops

### ERO Hop Types

- **Strict hop:** The next hop must be directly connected. No intermediate routers allowed.
- **Loose hop:** Intermediate routers may exist between the current node and the specified hop. The transit router runs CSPF to reach the loose hop.

### Path Expansion

For an ERO with $s$ strict hops and $l$ loose hops, the CSPF computations needed:

- Strict hops: 0 CSPF runs (path fully specified)
- Each loose hop: 1 CSPF run at the preceding router to compute the sub-path

Total CSPF runs for a mixed ERO: $l$ (one per loose hop boundary).

### Worked Example

ERO: [strict A, strict B, loose D, strict F]

- A to B: directly connected (strict, no CSPF)
- B to D: loose hop, B runs CSPF and expands to B-C-D
- D to F: directly connected (strict, no CSPF)

Final path: A-B-C-D-F. One CSPF computation at B.

---

## 12. RSVP-TE Graceful Restart and Recovery

### Graceful Restart (RFC 3473)

When a router restarts, it must recover RSVP state without tearing down all tunnels. The restart time budget:

$$T_{restart} < T_{cleanup}$$

The restarting router must recover state from neighbors before the cleanup timer expires on those neighbors.

### Recovery Sequence

1. Router restarts, sends Hello with restart-cap bit set
2. Neighbor enters recovery mode, preserves state for $T_{recovery}$
3. Restarting router receives PATH/RESV from neighbors and reconciles with its forwarding table
4. State successfully recovered: normal operation resumes
5. State not recovered within $T_{recovery}$: tunnel torn down

### State Recovery Time

$$T_{recovery} = T_{boot} + T_{protocol\_init} + T_{state\_sync}$$

For a router with $T_r$ tunnels:

$$T_{state\_sync} \approx T_r \times t_{per\_tunnel}$$

Where $t_{per\_tunnel}$ is the time to process one tunnel's recovery (~1-10ms).

Example: 5,000 tunnels, 5ms per tunnel: $T_{state\_sync} = 25$ seconds. Add boot time (~60s) and protocol init (~5s): total ~90 seconds. This must be less than the neighbor's $T_{cleanup}$ (157.5s default).

---

## Prerequisites

- Solid understanding of MPLS fundamentals (label operations, LSP, LDP) — see the mpls detail page
- IGP knowledge (OSPF or IS-IS) for TE extensions
- Basic graph theory (Dijkstra's algorithm, shortest path)
- Understanding of the RSVP protocol model (soft state, PATH/RESV)

## References

- [RFC 3209 — RSVP-TE: Extensions to RSVP for LSP Tunnels](https://www.rfc-editor.org/rfc/rfc3209)
- [RFC 2205 — Resource ReSerVation Protocol (RSVP)](https://www.rfc-editor.org/rfc/rfc2205)
- [RFC 3630 — Traffic Engineering (TE) Extensions to OSPF Version 2](https://www.rfc-editor.org/rfc/rfc3630)
- [RFC 5305 — IS-IS Extensions for Traffic Engineering](https://www.rfc-editor.org/rfc/rfc5305)
- [RFC 4090 — Fast Reroute Extensions to RSVP-TE for LSP Tunnels](https://www.rfc-editor.org/rfc/rfc4090)
- [RFC 2961 — RSVP Refresh Overhead Reduction Extensions](https://www.rfc-editor.org/rfc/rfc2961)
- [RFC 3473 — Generalized Multi-Protocol Label Switching (GMPLS) Signaling](https://www.rfc-editor.org/rfc/rfc3473)
- [RFC 7308 — Extended Administrative Groups in MPLS-TE](https://www.rfc-editor.org/rfc/rfc7308)
- [RFC 3785 — Use of Interior Gateway Protocol Metric as a Second MPLS TE Metric](https://www.rfc-editor.org/rfc/rfc3785)
- [RFC 4875 — Extensions to RSVP-TE for Point-to-Multipoint TE LSPs](https://www.rfc-editor.org/rfc/rfc4875)
