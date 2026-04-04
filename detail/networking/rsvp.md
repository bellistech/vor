# The Mathematics of RSVP — Token Buckets, Admission Control, and Bandwidth Allocation

> *RSVP translates QoS requirements into mathematical guarantees. The token bucket model quantifies traffic shape, admission control solves a bin-packing optimization, and soft state refresh creates a self-healing distributed system governed by exponential decay.*

---

## 1. Token Bucket Model (Traffic Shaping)

### The Token Bucket Formula

RSVP uses the token bucket model to describe traffic. A sender generates tokens at rate $r$ into a bucket of depth $b$. A packet of size $L$ is conformant if:

$$L \leq b + r \times t$$

Where $t$ is the time since the last packet. The burst size over any interval $[t_1, t_2]$:

$$A(t_1, t_2) \leq b + r \times (t_2 - t_1)$$

This is the **arrival curve** — an affine function bounding the maximum data that can arrive in any interval of length $\Delta t = t_2 - t_1$:

$$\sigma(\Delta t) = b + r \times \Delta t$$

### Token Bucket Parameters

| Parameter | Symbol | Unit | Description |
|:---|:---:|:---:|:---|
| Token rate | $r$ | bytes/sec | Average sustainable rate |
| Bucket depth | $b$ | bytes | Maximum burst size |
| Peak rate | $p$ | bytes/sec | Maximum instantaneous rate ($p \geq r$) |
| Min policed unit | $m$ | bytes | Smallest counted packet |
| Max datagram size | $M$ | bytes | Largest allowed packet ($M \leq b$) |

### Burst Duration

The maximum burst duration at peak rate before the bucket empties:

$$T_{burst} = \frac{b}{p - r}$$

For $b = 10{,}000$ bytes, $r = 100{,}000$ bytes/sec, $p = 1{,}000{,}000$ bytes/sec:

$$T_{burst} = \frac{10{,}000}{1{,}000{,}000 - 100{,}000} = \frac{10{,}000}{900{,}000} = 11.1 \text{ ms}$$

---

## 2. Guaranteed Service Delay Bound (Network Calculus)

### End-to-End Delay

For Guaranteed Service (RFC 2212), the worst-case end-to-end delay through $K$ hops is bounded by:

$$D_{total} = \sum_{k=1}^{K} \left(\frac{b_k + C_k}{R_k} + \frac{D_k}{R_k}\right) + \sum_{k=1}^{K} d_k$$

Where at hop $k$:
- $b_k$ = token bucket depth
- $C_k$ = rate-dependent error term (accounts for packetization)
- $D_k$ = rate-independent delay error (propagation, processing)
- $R_k$ = reserved rate
- $d_k$ = propagation delay

### Simplified Single-Hop Delay

For a single hop with reserved rate $R \geq r$:

$$D = \frac{b}{R} + \frac{M + C}{R} + d$$

Where:
- $\frac{b}{R}$ = time to drain the burst at reserved rate
- $\frac{M+C}{R}$ = serialization delay plus error term
- $d$ = propagation delay

| Reserved Rate ($R$) | Burst ($b$) | Max Packet ($M$) | Delay Bound |
|:---:|:---:|:---:|:---:|
| 1 Mbps | 10 KB | 1500 B | 91.5 ms |
| 10 Mbps | 10 KB | 1500 B | 9.2 ms |
| 100 Mbps | 10 KB | 1500 B | 0.9 ms |
| 1 Gbps | 10 KB | 1500 B | 0.09 ms |

### Slack Term

The slack term $S$ allows trading delay for bandwidth efficiency:

$$S = D_{requested} - D_{minimum}$$

If $S > 0$, intermediate routers may reduce the reserved rate $R$ while still meeting the delay requirement, freeing bandwidth for other flows.

---

## 3. Admission Control (Bin Packing)

### Bandwidth Admission Decision

At each router, admission control checks if accepting a new reservation would exceed available bandwidth:

$$\sum_{i=1}^{n} R_i + R_{new} \leq BW_{available}$$

Where $R_i$ is the reserved rate for existing flow $i$, and $BW_{available}$ is the RSVP-configured interface bandwidth.

### Utilization After Admission

$$\rho = \frac{\sum_{i=1}^{n+1} R_i}{BW_{total}}$$

| Interface BW | Reserved Flows | Total Reserved | Utilization |
|:---:|:---:|:---:|:---:|
| 1 Gbps | 10 x 100 Mbps | 1,000 Mbps | 100% |
| 10 Gbps | 50 x 100 Mbps | 5,000 Mbps | 50% |
| 10 Gbps | 100 x 100 Mbps | 10,000 Mbps | 100% |
| 100 Gbps | 200 x 100 Mbps | 20,000 Mbps | 20% |

### Overbooking Factor

In practice, not all flows use their full reservation. With overbooking factor $\alpha > 1$:

$$\sum_{i=1}^{n} R_i \leq \alpha \times BW_{available}$$

$$\alpha = \frac{1}{\bar{u}}$$

Where $\bar{u}$ is the average utilization of reserved bandwidth. If flows typically use 50% of their reservation, $\alpha = 2$ doubles effective capacity.

---

## 4. Soft State Refresh and Expiry (Exponential Decay)

### Refresh Mechanism

RSVP state is maintained by periodic refresh messages. If no refresh arrives within the cleanup timeout, state expires:

$$T_{cleanup} = (K + 0.5) \times 1.5 \times R$$

Where:
- $K$ = number of missed refreshes before cleanup (default 3)
- $R$ = refresh period (default 30 seconds)

$$T_{cleanup} = (3 + 0.5) \times 1.5 \times 30 = 3.5 \times 45 = 157.5 \text{ s}$$

In practice, the commonly cited value is $\approx 3.5 \times R = 105$ s (simplified).

### Probability of State Loss

If each refresh message has independent loss probability $p$, the probability of losing $K$ consecutive refreshes:

$$P(\text{state loss}) = p^K$$

| Loss Rate ($p$) | $K=3$ | $K=4$ | $K=5$ |
|:---:|:---:|:---:|:---:|
| 1% | 0.0001% | 0.000001% | $10^{-8}$% |
| 5% | 0.0125% | 0.000625% | 0.00003% |
| 10% | 0.1% | 0.01% | 0.001% |
| 20% | 0.8% | 0.16% | 0.032% |

Even with 10% packet loss, the probability of spurious state deletion is only 0.1% per refresh cycle.

### Refresh Overhead

Total refresh messages per second across all interfaces:

$$\text{Refreshes/sec} = \frac{N_{flows}}{R}$$

For 10,000 flows with $R = 30$ s: $\frac{10{,}000}{30} = 333$ messages/sec.

With summary refresh (RFC 2961), this reduces to approximately:

$$\text{Refreshes/sec} \approx \frac{N_{flows}}{R \times S_{bundle}}$$

Where $S_{bundle}$ is the average bundle size (typically 50-100 state IDs per message).

---

## 5. RSVP-TE Bandwidth Allocation Models

### Maximum Allocation Model (MAM)

Bandwidth is partitioned into Class Types (CTs). Each CT has a strict bandwidth constraint:

$$\sum_{i \in CT_j} R_i \leq BC_j, \quad \forall j$$

Where $BC_j$ is the bandwidth constraint for class type $j$.

### Russian Doll Model (RDM)

Bandwidth constraints are hierarchical — higher classes can use lower-class bandwidth:

$$\sum_{k=j}^{N} \text{Reserved}_{CT_k} \leq BC_j$$

| Constraint | MAM | RDM |
|:---|:---|:---|
| $BC_0$ | CT0 only: 7 Gbps | CT0+CT1+CT2: 10 Gbps |
| $BC_1$ | CT1 only: 2 Gbps | CT1+CT2: 5 Gbps |
| $BC_2$ | CT2 only: 1 Gbps | CT2 only: 2 Gbps |
| **Total** | **10 Gbps** | **10 Gbps** |

RDM allows higher-priority classes to borrow unused bandwidth from lower-priority pools, improving utilization.

---

## 6. Preemption Priority Mathematics

### Setup and Hold Priority

Each LSP has a setup priority $P_s$ and hold priority $P_h$ (0 = highest, 7 = lowest):

$$P_s \leq P_h \text{ (always, by convention)}$$

An LSP with setup priority $P_s^{new}$ can preempt an existing LSP with hold priority $P_h^{existing}$ if:

$$P_s^{new} < P_h^{existing}$$

### Preemption Decision

When bandwidth is insufficient, the router selects LSPs to preempt using:

$$\text{Minimize: } \sum_{i \in \text{preempted}} R_i$$

$$\text{Subject to: } \sum_{i \in \text{preempted}} R_i \geq R_{new} - BW_{free}$$

$$\forall i \in \text{preempted}: P_h^{(i)} > P_s^{new}$$

This is a variant of the knapsack problem (NP-hard in general, but practical for small numbers of LSPs per interface).

---

## 7. Fast Reroute Recovery Time

### FRR Switchover Analysis

Fast Reroute targets 50 ms recovery. The total recovery time:

$$T_{recovery} = T_{detect} + T_{switch} + T_{propagate}$$

Where:
- $T_{detect}$ = failure detection time (BFD or physical layer)
- $T_{switch}$ = time to activate backup path
- $T_{propagate}$ = time for first packet on new path

| Detection Method | $T_{detect}$ | $T_{switch}$ | $T_{propagate}$ | Total |
|:---|:---:|:---:|:---:|:---:|
| Physical (fiber cut) | 1-10 ms | 1-5 ms | 0-5 ms | 2-20 ms |
| BFD (10ms interval) | 30-50 ms | 1-5 ms | 0-5 ms | 31-60 ms |
| RSVP Hello (default) | 9-15 s | 1-5 ms | 0-5 ms | 9-15 s |
| Refresh timeout | 105 s | 1-5 ms | 0-5 ms | 105 s |

### Packets Lost During Switchover

$$\text{Packets lost} = \frac{T_{recovery} \times \text{Rate}}{L_{avg}}$$

For a 50 ms switchover on a 1 Gbps tunnel with 500-byte average packets:

$$\frac{0.050 \times 1{,}000{,}000{,}000 / 8}{500} = 12{,}500 \text{ packets}$$

---

*RSVP transforms the abstract notion of "quality of service" into concrete mathematical guarantees. Token buckets bound traffic arrival, network calculus proves delay bounds, and admission control solves optimization problems at line rate. The elegance of soft state turns an unreliable network into a self-healing reservation system where the mathematics of probability ensures robustness against message loss.*

## Prerequisites

- calculus (rates and integrals), probability theory, combinatorial optimization, network calculus fundamentals

## Complexity

- **Beginner:** Token bucket calculations and admission control arithmetic
- **Intermediate:** Guaranteed service delay bounds and soft state probability analysis
- **Advanced:** Network calculus arrival/service curves and preemption optimization
