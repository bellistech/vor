# The Mathematics of IGMP — Multicast Group Dynamics & Querier Convergence

> *IGMP's deceptively simple query-report cycle hides rich mathematical structure: report suppression follows coupon-collector statistics, querier election is a distributed minimum-finding algorithm, and snooping table growth scales with the product of groups and ports — a combinatorial explosion that determines switch TCAM requirements.*

---

## 1. Report Suppression and Response Timing (Uniform Random Backoff)

### The Mechanism

When a host receives an IGMP General Query with Max Response Time $T_{max}$, it selects a random delay uniformly from $[0, T_{max}]$ for each group it belongs to. If it hears another host's report for the same group before its timer expires, it suppresses its own report.

### Suppression Probability

For $n$ hosts in group $G$, each picks delay $d_i \sim U[0, T_{max}]$. The host with the minimum delay reports; all others suppress.

Probability that a specific host $k$ is the reporter:

$$P(\text{host } k \text{ reports}) = \frac{1}{n}$$

Expected number of reports per group per query (IGMPv2 with suppression):

$$E[\text{reports}] = 1$$

This holds regardless of $n$ because suppression is perfect when all hosts hear the first report. The variance in report timing:

$$E[\min(d_1, \ldots, d_n)] = \frac{T_{max}}{n+1}$$

| Hosts in Group ($n$) | $T_{max}$ = 10s | Expected First Report At |
|:---:|:---:|:---:|
| 1 | 10s | 5.00s |
| 5 | 10s | 1.67s |
| 10 | 10s | 0.91s |
| 50 | 10s | 0.20s |
| 100 | 10s | 0.10s |

### IGMPv3 Difference

IGMPv3 eliminates report suppression because reports go to 224.0.0.22 (not the group address), so hosts cannot hear each other's reports. Total reports per query:

$$E[\text{reports}_{v3}] = n$$

This increases traffic but enables accurate per-host source filtering state.

---

## 2. Querier Election Convergence (Distributed Minimum)

### The Algorithm

IGMP querier election selects the router with the lowest IP address on the subnet. All routers initially assume querier role and send General Queries. Upon receiving a query from a lower IP, a router yields.

### Convergence Time

Let $R$ routers have query interval $Q_I$ with startup query interval $Q_S = Q_I/4$ and startup query count $C_S = 2$.

Worst case: all $R$ routers send startup queries simultaneously. After one round-trip:

$$T_{converge} \leq Q_S + \delta_{prop}$$

Where $\delta_{prop}$ is the propagation delay on the LAN (microseconds for Ethernet). In practice:

$$T_{converge} \approx \frac{Q_I}{4} = \frac{125}{4} = 31.25 \text{s}$$

### Other Querier Present Interval

Non-queriers set the Other Querier Present timer:

$$T_{other} = (R_V \times Q_I) + \frac{Q_{RI}}{2}$$

Where $R_V$ = Robustness Variable (default 2), $Q_I$ = Query Interval (125s), $Q_{RI}$ = Query Response Interval (10s):

$$T_{other} = (2 \times 125) + 5 = 255 \text{s}$$

If the querier fails, re-election takes at most 255 seconds.

---

## 3. Group Membership Timeout (Exponential Decay)

### State Machine Timers

A router considers a group present on an interface as long as the Group Membership Interval has not expired:

$$T_{GMI} = (R_V \times Q_I) + Q_{RI}$$

$$T_{GMI} = (2 \times 125) + 10 = 260 \text{s}$$

### Leave Latency

When a host sends a Leave, the router sends $C_{LM}$ (Last Member Query Count, default $R_V = 2$) Group-Specific Queries at interval $T_{LM}$ (Last Member Query Interval, default 1s):

$$T_{leave} = C_{LM} \times T_{LM} = 2 \times 1 = 2 \text{s}$$

The probability that a remaining member responds within $T_{LM}$, given it picks delay $d \sim U[0, T_{LM}]$:

$$P(\text{response}) = 1 - \left(\frac{0}{T_{LM}}\right)^n = 1$$

More precisely, the probability that $n$ remaining members all fail to respond to $C_{LM}$ queries (assuming independent, uniform loss probability $p$):

$$P(\text{false prune}) = p^{n \times C_{LM}}$$

| Loss Rate $p$ | 1 Member, 2 Queries | 3 Members, 2 Queries |
|:---:|:---:|:---:|
| 0.01 | 0.0001 | $10^{-12}$ |
| 0.05 | 0.0025 | $1.5 \times 10^{-8}$ |
| 0.10 | 0.01 | $10^{-6}$ |

The Robustness Variable $R_V$ provides resilience: increasing $R_V$ from 2 to 3 cubes the false-prune probability.

---

## 4. IGMP Snooping Table Size (Combinatorial Scaling)

### Switch Memory Requirements

An IGMP snooping switch maintains a (group, port) forwarding table. In the worst case:

$$|\text{MDB entries}| = G \times P$$

Where $G$ = number of active multicast groups and $P$ = number of switch ports.

Each MDB entry requires storage for the group address (4 bytes), port bitmap, and timers. Approximate memory per entry:

$$M_{entry} \approx 4 + \lceil P/8 \rceil + 8 \text{ bytes (timer)}$$

| Switch Ports | Groups | MDB Entries (worst) | Memory (approx) |
|:---:|:---:|:---:|:---:|
| 24 | 100 | 2,400 | 43 KB |
| 48 | 500 | 24,000 | 528 KB |
| 48 | 4,096 | 196,608 | 4.3 MB |
| 96 | 10,000 | 960,000 | 24 MB |

Enterprise IPTV deployments with thousands of channels can exhaust switch TCAM. Practical limit on most access switches: 1,000--4,096 groups.

---

## 5. Multicast MAC Ambiguity (Pigeonhole Collision)

### The 32:1 Mapping Problem

IPv4 multicast maps 28 bits of group ID into 23 bits of Ethernet MAC:

$$2^{28} \text{ groups} \rightarrow 2^{23} \text{ MACs}$$

Collision ratio:

$$\frac{2^{28}}{2^{23}} = 2^5 = 32$$

The probability that two randomly chosen groups from the full multicast range collide at Layer 2:

$$P(\text{collision}) = \frac{1}{2^{23}} \approx 1.19 \times 10^{-7}$$

But for $k$ groups in use simultaneously (birthday problem variant):

$$P(\text{any collision}) \approx 1 - e^{-k(k-1)/(2 \times 2^{23})}$$

| Active Groups $k$ | $P(\text{collision})$ |
|:---:|:---:|
| 10 | 0.000005 |
| 100 | 0.0006 |
| 1,000 | 0.058 |
| 4,096 | 0.624 |

At scale, MAC-level collisions become likely, delivering unwanted traffic to hosts. IGMP snooping at Layer 3 (inspecting the IP group address) eliminates this ambiguity.

---

## 6. Query Traffic Load (Steady-State Analysis)

### Bandwidth Consumed by IGMP

In steady state with $G$ groups and query interval $Q_I$:

General Query traffic (one per $Q_I$):

$$R_{query} = \frac{S_{query}}{Q_I}$$

Where $S_{query}$ = 46 bytes (minimum Ethernet frame with IGMP query). For $Q_I = 125$s:

$$R_{query} = \frac{46 \times 8}{125} = 2.944 \text{ bps}$$

Response traffic (IGMPv2 with suppression, one report per group):

$$R_{response} = \frac{G \times S_{report}}{Q_I} = \frac{G \times 46 \times 8}{125}$$

| Active Groups $G$ | Response Traffic |
|:---:|:---:|
| 10 | 29 bps |
| 100 | 294 bps |
| 1,000 | 2.9 kbps |
| 10,000 | 29.4 kbps |

IGMP control plane traffic is negligible even at scale. The real cost is per-group state in routers and switches, not bandwidth.

---

## 7. SSM vs ASM Scalability (State Complexity)

### Any-Source Multicast (ASM)

Router state per group with PIM-SM:

$$S_{ASM} = O(|sources| \times |groups|)$$

Each (S, G) pair requires a separate forwarding entry. The RP (Rendezvous Point) must maintain state for all active groups.

### Source-Specific Multicast (SSM)

Eliminates the RP and shared tree. State is purely (S, G) with no (\*, G):

$$S_{SSM} = O(|subscriptions|)$$

Where each subscription is a unique (source, group) requested by receivers via IGMPv3 INCLUDE mode.

For IPTV with $C$ channels (each a unique source), $R$ receivers, each watching $w$ channels:

$$S_{SSM} = R \times w \times \text{sizeof(S,G entry)}$$

But the forwarding plane only needs unique (S, G) entries:

$$|\text{FIB entries}| = C$$

| Model | FIB Entries | RP Required | IGMPv3 Required |
|:---:|:---:|:---:|:---:|
| ASM (*, G) | $G$ | Yes | No (v2 sufficient) |
| ASM (S, G) | $S \times G$ | Yes (initially) | No |
| SSM | $C$ | No | Yes |

SSM scales linearly with channels, making it the standard for large IPTV deployments.

---

*IGMP's mathematics reveal a protocol carefully designed for efficiency: report suppression reduces query responses to O(1) per group, querier election converges in a single round, and SSM collapses multicast state from a source-group product to a simple channel count. The limiting factor is never IGMP bandwidth but rather the TCAM and memory capacity of intermediate switches performing snooping.*

## Prerequisites

- Probability theory (uniform distribution, birthday problem, order statistics)
- Combinatorics (pigeonhole principle, product scaling)
- Queueing and timer analysis (exponential backoff, convergence bounds)

## Complexity

- **Beginner:** Report suppression timing, basic group join/leave latency calculations
- **Intermediate:** Snooping table scaling, MAC collision probability, querier convergence analysis
- **Advanced:** SSM vs ASM state complexity, TCAM capacity planning, false-prune probability under loss
