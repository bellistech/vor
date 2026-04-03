# The Mathematics of RIP — Distance Vector, Hop Limits, and Convergence

> *RIP is the simplest routing protocol — a pure distance-vector algorithm using hop count as metric. Its math illustrates the Bellman-Ford equation, the count-to-infinity problem, and why a 15-hop limit exists.*

---

## 1. The Bellman-Ford Equation

### The Core Formula

$$D(x, y) = \min_{v \in \text{neighbors}(x)} \{ c(x, v) + D(v, y) \}$$

Where:
- $D(x, y)$ = distance from $x$ to destination $y$
- $c(x, v)$ = cost from $x$ to neighbor $v$ (always 1 in RIP)
- $D(v, y)$ = neighbor $v$'s distance to $y$

Since RIP uses hop count ($c = 1$ for all links):

$$D(x, y) = 1 + \min_{v} D(v, y)$$

### Convergence: Iterative Updates

Each update cycle, every router sends its entire routing table to neighbors. After $k$ iterations:

$$D^{(k)}(x, y) = \min_{v} \{ 1 + D^{(k-1)}(v, y) \}$$

Convergence requires at most $V - 1$ iterations (diameter of the network).

$$T_{convergence} = (V - 1) \times T_{update}$$

With $T_{update} = 30$ seconds (default):

| Network Diameter | Iterations | Convergence Time |
|:---:|:---:|:---:|
| 3 hops | 3 | 90 sec |
| 5 hops | 5 | 150 sec |
| 10 hops | 10 | 300 sec (5 min) |
| 15 hops (max) | 15 | 450 sec (7.5 min) |

---

## 2. The 15-Hop Limit — Infinity Definition

### Why 15?

RIP defines infinity as 16 hops. A route with metric 16 is unreachable.

$$\text{Metric range} = [0, 16]$$

$$\text{Maximum usable hops} = 15$$

This was a design choice to bound the count-to-infinity problem. With infinity = 16, the worst case counting is 16 iterations:

$$T_{worst\_infinity} = 16 \times 30 = 480 \text{ seconds} = 8 \text{ minutes}$$

### Network Diameter Constraint

$$\text{Diameter} \leq 15 \text{ hops}$$

This limits RIP to small networks. For perspective:

| Network Type | Typical Diameter | RIP Viable? |
|:---|:---:|:---:|
| Small office (5 routers) | 3 | Yes |
| Campus (20 routers) | 6 | Yes |
| Enterprise (50 routers) | 12 | Marginal |
| ISP backbone | 20+ | No |

---

## 3. Count-to-Infinity — The Convergence Problem

### The Scenario

Routers A and B both reach destination D. A's path goes through B (cost 2). B has a direct link to D (cost 1). If B's link to D fails:

1. B sets $D(B, D) = \infty$ (16)
2. But before B's update arrives, A advertises: "I can reach D at cost 2"
3. B now thinks: $D(B, D) = 1 + D(A, D) = 1 + 2 = 3$
4. A updates: $D(A, D) = 1 + D(B, D) = 1 + 3 = 4$
5. Loop continues: 5, 6, 7, ... until 16

### Count Duration

$$\text{Counts} = \infty_{def} - D_{initial} = 16 - 2 = 14 \text{ iterations}$$

$$T_{count} = 14 \times 30 = 420 \text{ seconds} = 7 \text{ minutes}$$

### Mitigation Techniques

| Technique | Effect | Convergence Improvement |
|:---|:---|:---|
| Split horizon | Don't advertise route back to source | Eliminates 2-node loops |
| Poison reverse | Advertise route as 16 to source | Same, more explicit |
| Triggered updates | Send immediately on change | Reduces from 30s to ~seconds |
| Hold-down timer | Ignore worse routes for 180s | Prevents oscillation |

---

## 4. Timer Analysis

### RIP Timers

| Timer | Default | Formula |
|:---|:---:|:---|
| Update | 30 sec | Periodic full table broadcast |
| Invalid | 180 sec | $6 \times T_{update}$: no update = route invalid |
| Hold-down | 180 sec | Ignore worse routes for this period |
| Flush | 240 sec | $8 \times T_{update}$: remove from table |

### Failure Detection Time

$$T_{detect} = T_{invalid} = 180 \text{ sec} = 3 \text{ minutes}$$

Compare with OSPF (40 sec dead timer) or BFD (150 ms).

### Bandwidth Consumed by Updates

Each route entry: 20 bytes. Maximum routes per RIPv2 message: 25 (in 512-byte UDP payload).

$$\text{Messages per update} = \lceil \frac{R}{25} \rceil$$

$$\text{Bandwidth} = \lceil \frac{R}{25} \rceil \times 512 \times 8 / T_{update}$$

| Routes | Messages | Bandwidth |
|:---:|:---:|:---:|
| 25 | 1 | 137 bps |
| 100 | 4 | 546 bps |
| 500 | 20 | 2,731 bps |
| 10,000 | 400 | 54,613 bps |

At 10,000 routes (unlikely for RIP), each update consumes ~55 kbps — negligible on modern links but significant on legacy WAN connections.

---

## 5. RIPv1 vs RIPv2 vs RIPng

### Feature Comparison

| Feature | RIPv1 | RIPv2 | RIPng (IPv6) |
|:---|:---:|:---:|:---:|
| VLSM/CIDR | No | Yes | Yes |
| Authentication | No | Yes (MD5) | No (uses IPsec) |
| Multicast | No (broadcast) | 224.0.0.9 | FF02::9 |
| Max metric | 15 | 15 | 15 |
| Update size | 512 B | 512 B | Up to MTU |

### Classful vs Classless Route Entries

RIPv1 (classful): wastes address space due to class boundaries.
RIPv2 (classless): includes subnet mask, enabling VLSM.

$$\text{Routing table efficiency (RIPv2)} = \frac{\text{Specific prefixes}}{\text{Total entries}}$$

With VLSM, a single /16 might be split into 200 specific subnets — all representable in RIPv2 but collapsed to class boundary in RIPv1.

---

## 6. Load Balancing — Equal-Cost Paths

### RIP ECMP

RIP supports up to $K$ equal-cost paths (typically $K = 4$):

$$\text{Traffic per path} = \frac{1}{K}$$

Since RIP uses hop count only, two paths with the same hop count but vastly different bandwidth are treated equally:

$$\text{Path 1: } 1G \times 2 \text{ hops} = \text{cost 2}$$
$$\text{Path 2: } 56K \times 2 \text{ hops} = \text{cost 2}$$

Both get 50% of traffic — a fundamental limitation of hop-count metrics.

---

## 7. Summary of Formulas

| Formula | Math Type | Application |
|:---|:---|:---|
| $1 + \min_v D(v,y)$ | Bellman-Ford recurrence | Route calculation |
| $(V-1) \times 30$ sec | Linear | Convergence time |
| $16 - D_{initial}$ iterations | Subtraction | Count-to-infinity duration |
| $6 \times T_{update}$ | Linear multiplier | Invalid timer |
| $\lceil R/25 \rceil \times 512 \times 8 / T$ | Rate calculation | Update bandwidth |
| $\leq 15$ hops | Constraint | Network diameter limit |

---

*RIP's simplicity is both its strength and its death sentence — hop count ignores bandwidth, convergence takes minutes, and 15 hops limits network size. But Bellman-Ford is the mathematical foundation that all distance-vector protocols (including BGP) build upon.*
