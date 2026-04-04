# The Mathematics of STP — Convergence Timing & Graph Theory

> *Spanning Tree Protocol is applied graph theory: given a cyclic graph of bridges and links, find a minimum spanning tree rooted at the bridge with the lowest identifier. The timers that govern convergence are not arbitrary — they derive from worst-case diameter calculations and propagation bounds.*

---

## 1. Convergence Timing (Timer Algebra)

### The Problem

STP (802.1D) has three timers: Hello Time, Forward Delay, and Max Age. How do these interact, and why does convergence take so long?

### The Formulas

$$T_{convergence}^{STP} = T_{max\_age} + 2 \times T_{forward\_delay}$$

Default values:
- $T_{hello} = 2$ seconds
- $T_{forward\_delay} = 15$ seconds
- $T_{max\_age} = 20$ seconds

$$T_{convergence}^{default} = 20 + 2(15) = 50 \text{ seconds}$$

### What Each Timer Does

**Max Age ($T_{max\_age}$):** How long a switch waits after losing BPDUs from root before declaring root down. A blocking port must wait $T_{max\_age}$ before considering itself a candidate for root port.

**Forward Delay ($T_{forward\_delay}$):** Time spent in each transitional state (Listening, Learning). Prevents loops during topology change by ensuring all switches have converged before forwarding.

**Hello Time ($T_{hello}$):** Interval between BPDU transmissions from root bridge. All other switches relay root BPDUs on their designated ports.

### RSTP Convergence

RSTP (802.1w) eliminates the listening and forward delay timers for most transitions:

$$T_{convergence}^{RSTP} = 3 \times T_{hello} + T_{handshake}$$

Where $T_{handshake}$ is the proposal/agreement exchange (one RTT per hop):

$$T_{handshake} = H \times RTT_{link}$$

For a 7-hop diameter network with 1 ms link RTT:

$$T_{convergence}^{RSTP} = 3(2) + 7(0.001) = 6.007 \text{ seconds}$$

In practice, RSTP converges in 1-3 seconds for typical campus networks.

### Timer Relationship Constraint

RFC dictates:

$$T_{max\_age} \geq 2 \times (T_{hello} - 1)$$
$$T_{max\_age} \leq 2 \times (T_{forward\_delay} - 1)$$

These ensure:
1. At least 2 missed hellos before declaring root dead
2. Forward delay is long enough for BPDUs to propagate

---

## 2. Network Diameter Limits (Graph Diameter)

### The Problem

STP has a maximum network diameter. What determines this limit, and how is it calculated?

### The Formula

Each switch increments the Message Age field in BPDUs by 1. When Message Age reaches Max Age, the BPDU is discarded:

$$D_{max} = T_{max\_age} / \text{increment} = T_{max\_age}$$

With default Max Age = 20, the theoretical maximum diameter is 20 hops. But the standard recommends:

$$D_{recommended} = 7 \text{ hops (switches from root to furthest leaf)}$$

### Why 7 Hops?

The constraint comes from the convergence formula. With larger diameters, the forward delay must increase to ensure BPDUs propagate before ports transition:

$$T_{forward\_delay} \geq \frac{D}{2} \times T_{hello} + \delta$$

For $D = 7$, $T_{hello} = 2$:

$$T_{forward\_delay} \geq \frac{7}{2} \times 2 + 1 = 8 \text{ seconds}$$

The default 15 seconds provides comfortable margin for 7 hops. Beyond 7 hops:

| Diameter $D$ | Min Forward Delay | Min Max Age | Total Convergence |
|:---:|:---:|:---:|:---:|
| 5 | 6s | 10s | 22s |
| 7 | 8s | 14s | 30s |
| 10 | 11s | 20s | 42s |
| 15 | 16s | 30s | 62s |
| 20 | 21s | 40s | 82s |

---

## 3. Spanning Tree as Graph Theory

### The Problem

STP finds a spanning tree of a bridged network. What is the formal graph-theoretic problem, and how does STP's algorithm relate to known spanning tree algorithms?

### Formal Definition

Given an undirected graph $G = (V, E)$ where:
- $V$ = set of bridges (switches)
- $E$ = set of links between bridges
- $w(e)$ = cost (inverse bandwidth) of edge $e$
- $r$ = root vertex (bridge with lowest ID)

STP finds a **minimum-cost spanning tree rooted at $r$**, which is a subgraph $T = (V, E')$ where:
- $E' \subseteq E$
- $T$ is connected
- $T$ is acyclic (tree)
- $|E'| = |V| - 1$
- The sum of edge costs on the path from any vertex to $r$ is minimized

### Relationship to Known Algorithms

STP is a **distributed** Bellman-Ford shortest path algorithm:

$$d_v = \min_{(u,v) \in E} (d_u + w(u,v))$$

Each bridge $v$ computes its distance to root $r$ by:
1. Receiving root path cost from neighbors ($d_u$)
2. Adding link cost $w(u,v)$
3. Selecting the neighbor with minimum total cost as root port

### Convergence Comparison

| Algorithm | Type | Convergence | Messages |
|:---|:---|:---|:---|
| STP (802.1D) | Distributed Bellman-Ford | $O(D \times T_{hello})$ | $O(|V| \times |E|)$ |
| RSTP (802.1w) | Distributed with sync | $O(D \times RTT)$ | $O(|E|)$ |
| Prim's (centralized) | Greedy | $O(|E| \log |V|)$ | N/A (centralized) |
| Kruskal's (centralized) | Greedy | $O(|E| \log |E|)$ | N/A (centralized) |

STP is slower than centralized algorithms but works without global knowledge — each bridge only sees its directly connected neighbors' BPDUs.

---

## 4. BPDU Propagation and Message Age (Wave Analysis)

### The Problem

How long does it take for a topology change to propagate across the entire network?

### The Formula

A BPDU from the root propagates one hop per hello interval:

$$T_{propagation} = D \times T_{hello}$$

Where $D$ is the network diameter. But since Message Age increments at each hop:

$$\text{Message Age at hop } h = h$$

The BPDU is discarded at hop $h$ if:

$$h > T_{max\_age}$$

### Topology Change Propagation

When a topology change occurs:
1. Detecting switch sends TCN BPDU toward root
2. Root sets TC flag in BPDUs for $T_{forward\_delay} + T_{max\_age}$ seconds
3. All switches shorten MAC address table aging to $T_{forward\_delay}$

Total TC notification time:

$$T_{TC} = D \times T_{hello} + T_{forward\_delay} + T_{max\_age}$$

With defaults and $D = 7$:

$$T_{TC} = 7 \times 2 + 15 + 20 = 49 \text{ seconds}$$

### MAC Table Flush Impact

During topology change, MAC table ages at $T_{forward\_delay}$ (15s) instead of default 300s. This causes:

$$\text{Flooding increase} = \frac{T_{normal\_aging}}{T_{forward\_delay}} = \frac{300}{15} = 20\times$$

A 20x increase in unknown unicast flooding for 15 seconds after topology change.

---

## 5. Redundancy and Failure Scenarios (Reliability)

### The Problem

STP provides redundancy through blocking ports. What is the failover time for different failure scenarios?

### Failure Scenarios

| Failure Type | Detection | Recovery Time (STP) | Recovery Time (RSTP) |
|:---|:---|:---|:---|
| Direct link failure | Physical | $T_{fd} \times 2 = 30$s | <1s (alternate port) |
| Indirect link failure | Max Age | $T_{ma} + T_{fd} \times 2 = 50$s | ~6s (3 hello) |
| Root bridge failure | Max Age | $T_{ma} + T_{fd} \times 2 = 50$s | ~6s + election |
| Unidirectional failure | Max Age (if loop guard) | $T_{ma} + T_{fd} \times 2 = 50$s | ~6s |

### Availability Calculation

With STP failover time $T_f$ and mean time between failures $MTBF$:

$$A = \frac{MTBF}{MTBF + T_f}$$

| MTBF | Failover (STP) | Failover (RSTP) | Availability (STP) | Availability (RSTP) |
|:---:|:---:|:---:|:---:|:---:|
| 1 year | 50s | 3s | 99.99984% | 99.99999% |
| 1 month | 50s | 3s | 99.9981% | 99.99988% |
| 1 week | 50s | 3s | 99.9917% | 99.99950% |
| 1 day | 50s | 3s | 99.942% | 99.9965% |

The difference matters for SLAs: 50 seconds of downtime is 2.6 minutes per month with weekly failures (STP) vs 0.2 minutes (RSTP).

---

## 6. BPDU Rate and Bandwidth (Overhead Analysis)

### The Problem

How much bandwidth does STP consume, and does it scale?

### The Formula

BPDU size on Ethernet:

$$S_{BPDU} = 14_{Eth} + 35_{STP} + 8_{LLC} = 57 \text{ bytes (minimum frame: 64 bytes)}$$

BPDUs per second from root:

$$R_{BPDU} = \frac{1}{T_{hello}} = 0.5 \text{ per second per port}$$

Total BPDU bandwidth per port:

$$B_{BPDU} = R_{BPDU} \times 64 \times 8 = 256 \text{ bps}$$

### PVST+ Scaling

With $V$ VLANs (PVST+), each VLAN runs a separate STP instance:

$$B_{PVST+} = V \times R_{BPDU} \times 64 \times 8$$

| VLANs | BPDUs/sec | Bandwidth per trunk |
|:---:|:---:|:---:|
| 10 | 5 | 2.56 kbps |
| 100 | 50 | 25.6 kbps |
| 500 | 250 | 128 kbps |
| 1000 | 500 | 256 kbps |
| 4094 | 2047 | 1.05 Mbps |

Bandwidth is negligible, but CPU processing of 2000+ BPDUs/sec on low-end switches can be significant. MSTP reduces this to a few instances regardless of VLAN count.

---

## 7. Summary of Formulas

| Formula | Math Type | Domain |
|:---|:---|:---|
| $T_{ma} + 2 \times T_{fd}$ | Addition | STP convergence |
| $3 \times T_{hello} + H \times RTT$ | Linear | RSTP convergence |
| $\min(d_u + w(u,v))$ | Bellman-Ford relaxation | Root path cost |
| $D \times T_{hello}$ | Multiplication | BPDU propagation |
| $MTBF / (MTBF + T_f)$ | Ratio | Availability |
| $V \times R \times S \times 8$ | Product | PVST+ bandwidth |

## Prerequisites

- graph theory (spanning trees, cycles), Bellman-Ford algorithm, timer analysis, availability math

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| Root bridge election | $O(D \times T_{hello})$ | $O(|V|)$ BPDUs |
| Port role calculation | $O(|E|)$ per bridge | $O(|E|)$ port states |
| STP convergence | $O(D \times T_{hello} + 2T_{fd})$ | $O(|V|)$ |
| RSTP convergence | $O(D \times RTT)$ | $O(|V|)$ |
| MSTP per-instance | Same as RSTP | $O(|V|)$ per instance |
| PVST+ all VLANs | $O(V \times D \times T)$ | $O(V \times |V|)$ |

---

*STP is a distributed graph algorithm running in real time on your switches. The 50-second convergence of classic STP is not a bug — it is the mathematically necessary time for Bellman-Ford to propagate across a 7-hop diameter at 2-second intervals with safety margins. RSTP slashes this by replacing passive timers with active handshakes.*
