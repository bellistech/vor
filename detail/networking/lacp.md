# The Mathematics of LACP — Hash Distribution & Throughput Scaling

> *Link aggregation promises linear bandwidth scaling, but the reality is governed by hash distribution mathematics. A 4-link bond does not give 4x throughput to a single flow — it gives 4x aggregate capacity distributed by a hash function whose uniformity depends on traffic diversity.*

---

## 1. Hash Distribution Uniformity (Probability Theory)

### The Problem

LACP uses a hash function to assign flows to member links. How evenly are flows distributed, and what is the probability of an imbalanced assignment?

### The Formula

With $F$ flows distributed across $L$ links using a uniform hash:

$$P(\text{link } i \text{ gets } k \text{ flows}) = \binom{F}{k} \left(\frac{1}{L}\right)^k \left(\frac{L-1}{L}\right)^{F-k}$$

This is a binomial distribution with $p = 1/L$.

Expected flows per link:

$$E[k] = \frac{F}{L}$$

Standard deviation:

$$\sigma = \sqrt{F \times \frac{1}{L} \times \frac{L-1}{L}} = \sqrt{\frac{F(L-1)}{L^2}}$$

### Worked Examples (4-Link Bond)

| Total Flows $F$ | Expected/Link | $\sigma$ | P(any link gets 0 flows) |
|:---:|:---:|:---:|:---:|
| 2 | 0.5 | 0.61 | 31.6% |
| 4 | 1.0 | 0.87 | 10.5% |
| 8 | 2.0 | 1.22 | 1.0% |
| 16 | 4.0 | 1.73 | <0.01% |
| 32 | 8.0 | 2.45 | ~0% |
| 100 | 25.0 | 4.33 | ~0% |

### Maximum Link Utilization

The most loaded link determines the effective bottleneck. The expected maximum load:

$$E[\max] \approx \frac{F}{L} + \sigma \times \sqrt{2 \ln L}$$

| Flows | Links | Expected Max Load | Imbalance Ratio |
|:---:|:---:|:---:|:---:|
| 4 | 2 | 2.83 flows | 1.41x |
| 4 | 4 | 1.96 flows | 1.96x |
| 10 | 4 | 3.87 flows | 1.55x |
| 100 | 4 | 28.2 flows | 1.13x |
| 1000 | 4 | 262 flows | 1.05x |

Key insight: with only 4 flows on 4 links, you should expect the busiest link to carry nearly 2x the average. Uniform distribution requires many flows.

---

## 2. Throughput Scaling (Non-Linear Returns)

### The Problem

A bond of $L$ links of bandwidth $B$ each has aggregate capacity $L \times B$. But effective throughput depends on flow distribution. What is the actual usable throughput?

### The Formula

Effective aggregate throughput for $F$ equal-bandwidth flows:

$$\Theta_{eff} = \min\left(F \times b_f, \frac{L \times B}{\max\_load\_ratio}\right)$$

Where $b_f$ is per-flow bandwidth and $\max\_load\_ratio$ is the imbalance factor from hash distribution.

### Single-Flow Case

A single flow always uses one link:

$$\Theta_{single} = B \quad \text{(regardless of } L\text{)}$$

A 4x10G bond gives at most 10 Gbps to a single TCP connection.

### Scaling Efficiency

Scaling efficiency relative to ideal linear scaling:

$$\eta = \frac{\Theta_{eff}}{L \times B}$$

| Links $L$ | Flows $F$ | $\eta$ (efficiency) | Effective Throughput (10G links) |
|:---:|:---:|:---:|:---:|
| 2 | 1 | 50% | 10 Gbps |
| 2 | 2 | 75-100% | 15-20 Gbps |
| 2 | 10 | 95% | 19 Gbps |
| 4 | 1 | 25% | 10 Gbps |
| 4 | 4 | 55-75% | 22-30 Gbps |
| 4 | 10 | 80-90% | 32-36 Gbps |
| 4 | 100 | 97% | 39 Gbps |
| 8 | 1 | 12.5% | 10 Gbps |
| 8 | 8 | 50-65% | 40-52 Gbps |
| 8 | 100 | 95% | 76 Gbps |

### Law of Diminishing Returns

Adding links beyond a point gives diminishing returns per link:

$$\Delta \Theta = \Theta(L+1) - \Theta(L) \leq B$$

For a single flow: $\Delta \Theta = 0$ for all $L > 1$ (no benefit at all).

For $F$ flows: marginal benefit per additional link:

$$\frac{\partial \Theta}{\partial L} = B \times \left(1 - P(\text{new link gets 0 flows})\right) \approx B \times \left(1 - \left(\frac{L}{L+1}\right)^F\right)$$

---

## 3. Flow Distribution Analysis (Hash Function Quality)

### The Problem

The hash function determines distribution quality. How do different hash policies perform?

### XOR Hash Analysis (layer2)

```
hash = (src_MAC[5] XOR dst_MAC[5]) % L
```

With only 2 unique MAC pairs (e.g., server to router and back), there are only 2 possible hash values — at most 2 links used regardless of link count.

### layer3+4 Hash

```
hash = (src_IP XOR dst_IP XOR src_port XOR dst_port) % L
```

With $2^{16}$ possible source ports, this produces ~65,536 unique hash inputs. Distribution is effectively uniform for $F \gg L$.

### Hash Collision Analysis

The probability that two specific flows hash to the same link:

$$P(\text{collision}) = \frac{1}{L}$$

Expected number of flow pairs sharing a link (birthday problem analog):

$$E[\text{collisions}] = \binom{F}{2} \times \frac{1}{L} = \frac{F(F-1)}{2L}$$

| Flows $F$ | Links $L$ | Expected Collisions | % of Pairs |
|:---:|:---:|:---:|:---:|
| 4 | 4 | 1.5 | 25% |
| 10 | 4 | 11.25 | 25% |
| 10 | 8 | 5.63 | 12.5% |
| 100 | 4 | 1237.5 | 25% |
| 100 | 8 | 618.75 | 12.5% |

Collisions are expected and normal. They only matter when the colliding flows are bandwidth-heavy.

---

## 4. LACPDU Timing and Failure Detection (Timer Analysis)

### The Problem

LACP detects link failures by monitoring LACPDU reception. How quickly are failures detected?

### The Formula

LACP uses a timeout multiplier:

$$T_{detect} = 3 \times T_{LACPDU}$$

| Rate | LACPDU Interval $T_{LACPDU}$ | Detection Time $T_{detect}$ |
|:---|:---:|:---:|
| Fast | 1 second | 3 seconds |
| Slow | 30 seconds | 90 seconds |

### Comparison with MII Monitoring

```
MII monitoring: T_detect = 3 × miimon (default 300ms)
LACP fast:      T_detect = 3 seconds
LACP slow:      T_detect = 90 seconds
```

MII detects physical link failures faster, but LACP detects logical failures (misconfiguration, one-sided LACP, upstream switch issues) that MII cannot.

### Combined Detection

With both MII and LACP:

$$T_{detect} = \min(T_{MII}, T_{LACP})$$

Best practice: `miimon=100` (300ms detection) + `lacp_rate=fast` (3s LACP detection).

---

## 5. Capacity Planning with Bonds (Queueing Theory)

### The Problem

How many links do you need in a bond to achieve a target throughput with a given traffic profile?

### The Formula

Required links to achieve target throughput $\Theta_t$ with efficiency $\eta$:

$$L = \left\lceil \frac{\Theta_t}{\eta \times B} \right\rceil$$

Where $\eta$ depends on flow count (from Section 2).

### Worked Examples

Target: 30 Gbps aggregate throughput with 10G links.

| Concurrent Flows | Efficiency $\eta$ | Links Needed | Actual Capacity |
|:---:|:---:|:---:|:---:|
| 1 | 25% (for 4 links) | impossible | 10G max |
| 4 | 65% | $\lceil 30/(0.65 \times 10) \rceil = 5$ | 32.5G effective |
| 10 | 85% | $\lceil 30/(0.85 \times 10) \rceil = 4$ | 34G effective |
| 100 | 97% | $\lceil 30/(0.97 \times 10) \rceil = 4$ | 38.8G effective |

### Oversubscription Ratio

For a server farm with $S$ servers, each with a bond of $L$ links at speed $B$, connecting to uplinks of speed $U$:

$$R_{oversub} = \frac{S \times L \times B}{U}$$

| Servers | Server Bond | Uplink | Oversubscription |
|:---:|:---:|:---:|:---:|
| 20 | 2x10G | 2x100G | 2:1 |
| 48 | 2x25G | 4x100G | 6:1 |
| 48 | 4x10G | 2x100G | 9.6:1 |

Industry norm: 3:1 to 5:1 for general compute, 1:1 for storage and HPC.

---

## 6. Summary of Formulas

| Formula | Math Type | Domain |
|:---|:---|:---|
| $\binom{F}{k}(1/L)^k((L-1)/L)^{F-k}$ | Binomial distribution | Flow distribution |
| $\sqrt{F(L-1)/L^2}$ | Standard deviation | Distribution uniformity |
| $F/L + \sigma\sqrt{2\ln L}$ | Order statistics | Max link load |
| $3 \times T_{LACPDU}$ | Multiplication | Failure detection |
| $\Theta_t / (\eta \times B)$ | Division | Link count planning |
| $F(F-1)/(2L)$ | Birthday problem | Hash collisions |

## Prerequisites

- binomial distribution, hash functions, order statistics, queueing theory basics

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| Hash computation (per packet) | $O(1)$ XOR ops | $O(1)$ |
| Link selection | $O(1)$ modulo | $O(L)$ link table |
| LACPDU processing | $O(1)$ per PDU | $O(L)$ partner state |
| Failover (remove link) | $O(1)$ | $O(L)$ |
| Bond rebalance | $O(F)$ rehash (on link change) | $O(F)$ flow table |

---

*LACP's throughput scaling is fundamentally limited by hash distribution. Four links do not give 4x throughput — they give 4x capacity that is only fully utilized when traffic diversity is high enough for the hash function to spread flows evenly. The math is clear: bond for redundancy first, aggregate bandwidth second.*
