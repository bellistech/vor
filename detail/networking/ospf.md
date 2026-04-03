# The Mathematics of OSPF — Dijkstra, Flooding, and Area Scaling

> *OSPF is Dijkstra's shortest path algorithm running on every router, rebuilding the network graph in real time. The math spans graph theory (SPF computation), combinatorics (LSA flooding), and capacity planning (area design).*

---

## 1. Dijkstra's SPF Algorithm — Complexity Analysis

### The Problem

Every OSPF router builds a complete graph of the network (the Link-State Database) and runs Dijkstra's algorithm to compute shortest paths to all destinations.

### The Algorithm Complexity

**Naive implementation (adjacency matrix):**

$$O(V^2)$$

**With binary heap (adjacency list):**

$$O((V + E) \log V)$$

**With Fibonacci heap:**

$$O(V \log V + E)$$

Where:
- $V$ = number of vertices (routers/networks)
- $E$ = number of edges (links)

### Worked Examples

| Network Size | Links | $O(V^2)$ ops | $O((V+E)\log V)$ ops | Ratio |
|:---:|:---:|:---:|:---:|:---:|
| 50 routers | 150 | 2,500 | 1,131 | 2.2x |
| 200 routers | 800 | 40,000 | 7,644 | 5.2x |
| 500 routers | 2,000 | 250,000 | 22,477 | 11.1x |
| 1,000 routers | 5,000 | 1,000,000 | 59,932 | 16.7x |
| 5,000 routers | 20,000 | 25,000,000 | 368,082 | 67.9x |

The heap-based implementation becomes dramatically better as the network grows.

### SPF Run Time in Practice

Modern routers compute SPF in approximately:

$$T_{SPF} \approx k \times V \quad \text{(roughly linear in practice)}$$

Where $k \approx 1$-$5$ microseconds per node. For 1,000 nodes: $\sim 1$-$5$ ms.

---

## 2. OSPF Metric and Path Cost

### The Formula

$$\text{Cost} = \frac{\text{Reference Bandwidth}}{\text{Interface Bandwidth}}$$

Default reference bandwidth: $10^8$ bps (100 Mbps).

### Worked Examples

| Interface | Bandwidth | Cost |
|:---|:---:|:---:|
| T1 (1.544 Mbps) | 1,544,000 | 64 |
| FastEthernet | 100,000,000 | 1 |
| GigabitEthernet | 1,000,000,000 | 1 (capped!) |
| 10GigE | 10,000,000,000 | 1 (same!) |

### The Problem with Default Reference

At 100 Mbps reference, everything >= 100 Mbps has cost 1. Solution: increase reference bandwidth.

With reference = $10^{10}$ (10 Gbps):

| Interface | Cost |
|:---|:---:|
| FastEthernet | 100 |
| GigabitEthernet | 10 |
| 10GigE | 1 |
| 40GigE | 1 (still capped) |

**Rule:** Set reference bandwidth to your fastest link speed or higher.

### Path Cost Calculation

Total path cost = sum of all outgoing interface costs along the path:

$$\text{Path Cost} = \sum_{i=1}^{n} \text{Cost}(e_i)$$

Dijkstra selects the path with minimum total cost. When two paths have equal cost, OSPF performs **Equal-Cost Multi-Path (ECMP)** load balancing.

---

## 3. LSA Flooding Math

### The Problem

When a link changes state, the originating router floods an LSA (Link-State Advertisement) to all other routers. How many LSA copies traverse the network?

### Flooding in a Full Mesh

In a broadcast segment with $N$ routers, the DR (Designated Router) reduces flooding:

**Without DR:** Each router sends to all others:

$$\text{LSA copies} = N \times (N - 1) = N^2 - N$$

**With DR/BDR:** Each router sends to DR, DR re-floods to all:

$$\text{LSA copies} = (N - 1) + (N - 1) = 2(N - 1)$$

| Routers | Without DR | With DR | Reduction |
|:---:|:---:|:---:|:---:|
| 5 | 20 | 8 | 60% |
| 10 | 90 | 18 | 80% |
| 25 | 600 | 48 | 92% |
| 50 | 2,450 | 98 | 96% |

### LSDB Size

The Link-State Database size scales with the number of LSAs in an area:

$$\text{LSDB size} \approx R \times L_{avg}$$

Where:
- $R$ = number of routers in the area
- $L_{avg}$ = average number of LSAs originated per router (typically 3-8: router LSA + network LSAs + summary LSAs)

For a 500-router area with 5 LSAs each: $500 \times 5 = 2,500$ LSAs. At ~100 bytes each: $\sim 250$ KB of LSDB.

---

## 4. Area Design — Scaling Math

### The Problem

A single OSPF area doesn't scale beyond ~500 routers. Areas partition the network to contain SPF computation and flooding.

### Area Scaling Model

**Single area:** SPF runs on all $V$ vertices:

$$T_{SPF} \propto V^2 \quad \text{(or } V \log V + E \text{)}$$

**With $A$ areas, each containing $V/A$ routers:**

$$T_{SPF\_per\_area} \propto \left(\frac{V}{A}\right)^2$$

**Improvement factor:**

$$\frac{V^2}{(V/A)^2} = A^2$$

Splitting into 4 areas reduces SPF computation by $16\times$.

### Area Design Table

| Total Routers | Areas | Routers/Area | SPF Improvement |
|:---:|:---:|:---:|:---:|
| 200 | 1 | 200 | 1x (baseline) |
| 200 | 4 | 50 | 16x |
| 200 | 10 | 20 | 100x |
| 1,000 | 10 | 100 | 100x |
| 1,000 | 20 | 50 | 400x |

### ABR Summary LSA Count

An Area Border Router (ABR) injects summary LSAs between areas. For $A$ areas with $P$ prefixes each, the backbone (Area 0) carries:

$$\text{Backbone LSAs} = A \times P$$

This is why route summarization at ABRs is critical — without it, the backbone becomes a bottleneck.

---

## 5. Hello/Dead Timer Math

### The Formulas

$$T_{dead} = k \times T_{hello}$$

Default: $k = 4$, so $T_{dead} = 4 \times T_{hello}$

| Network Type | Hello | Dead | Detection Time |
|:---|:---:|:---:|:---:|
| Broadcast | 10 sec | 40 sec | 40 sec |
| Point-to-point | 10 sec | 40 sec | 40 sec |
| NBMA | 30 sec | 120 sec | 120 sec |
| Fast timers | 1 sec | 3 sec | 3 sec |
| BFD-assisted | 50 ms | 150 ms | 150 ms |

### SPF Scheduling Timers (Throttling)

OSPF uses exponential backoff for SPF scheduling to prevent CPU overload during instability:

$$T_{wait}(n) = \min\left(T_{init} \times 2^n, T_{max}\right)$$

Typical defaults: $T_{init} = 50$ ms, $T_{max} = 5,000$ ms.

| SPF Run # | Wait Time | Cumulative |
|:---:|:---:|:---:|
| 1 | 50 ms | 50 ms |
| 2 | 100 ms | 150 ms |
| 3 | 200 ms | 350 ms |
| 4 | 400 ms | 750 ms |
| 5 | 800 ms | 1,550 ms |
| 6 | 1,600 ms | 3,150 ms |
| 7 | 3,200 ms | 6,350 ms |
| 8 | 5,000 ms (capped) | 11,350 ms |

After a quiet period ($T_{quiet}$, typically 5 sec), the backoff resets.

---

## 6. LSA Refresh and Aging

### The Model

Each LSA has a 16-bit age field (seconds). LSAs are refreshed before aging out:

$$T_{refresh} = 1800 \text{ sec (30 min)}$$
$$T_{maxage} = 3600 \text{ sec (60 min)}$$

An LSA must be refreshed every 30 minutes or it ages out and is flushed. For $L$ LSAs in the LSDB:

$$\text{Refresh rate} = \frac{L}{T_{refresh}} = \frac{L}{1800} \text{ LSAs/sec}$$

| LSDB Size | Refresh Rate | Refreshes/min |
|:---:|:---:|:---:|
| 1,000 LSAs | 0.56/sec | 33 |
| 5,000 LSAs | 2.78/sec | 167 |
| 10,000 LSAs | 5.56/sec | 333 |
| 50,000 LSAs | 27.78/sec | 1,667 |

At scale, LSA refresh traffic itself becomes significant overhead.

---

## 7. Stub Area Optimization

### The Math

Stub areas replace external routes (Type-5 LSAs) with a single default route:

$$\text{LSDB reduction} = E_{external}$$

For a network with 100,000 external routes redistributed into OSPF, a stub area saves ~100,000 LSAs per non-backbone area.

| Area Type | Type-3 (Summary) | Type-5 (External) | Type-7 (NSSA) | Total Savings |
|:---|:---:|:---:|:---:|:---:|
| Normal | All | All | N/A | 0 |
| Stub | All | None (default only) | N/A | $E$ LSAs |
| Totally Stub | Default only | None | N/A | $S + E$ LSAs |
| NSSA | All | None | Allowed | $E - E_{local}$ |

---

## 8. Summary of Formulas

| Formula | Math Type | Application |
|:---|:---|:---|
| $O(V^2)$ / $O(V \log V + E)$ | Algorithmic complexity | SPF computation |
| $\text{Ref BW} / \text{BW}$ | Division / inverse | Interface cost |
| $\sum \text{Cost}(e_i)$ | Summation | Path cost |
| $N^2 - N$ vs $2(N-1)$ | Quadratic vs linear | Flooding with/without DR |
| $(V/A)^2$ | Quadratic partition | Area scaling |
| $T_{init} \times 2^n$ | Exponential backoff | SPF throttling |
| $L / 1800$ | Rate calculation | LSA refresh overhead |
| $4 \times T_{hello}$ | Linear multiplier | Dead interval |

---

*Every OSPF router in your network is solving the shortest path problem hundreds of times per day — Dijkstra's 1956 algorithm running on silicon at line rate, rebuilding the forwarding table in milliseconds.*
