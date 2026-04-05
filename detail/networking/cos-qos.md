# CoS & QoS — Token Buckets, Queuing Theory & End-to-End Traffic Engineering

> *Quality of Service is applied queuing theory: finite buffers, competing flows, and the mathematics of fairness under contention. The token bucket gives you rate control, WFQ gives you fairness, WRED gives you congestion avoidance, and the delay/jitter/loss budget tells you whether your design actually works for voice and video. Every QoS decision reduces to a tradeoff between delay, loss, and throughput.*

---

## 1. Token Bucket Algorithm (Single-Rate)

### The Problem

How does a policer or shaper decide whether a packet conforms to a rate limit? The token bucket provides a deterministic model for rate enforcement with configurable burstiness.

### Single-Rate Token Bucket (RFC 2697 srTCM)

The bucket has depth $B_c$ (committed burst size in bytes) and fills at rate $r$ (CIR in bytes/sec).

Token accumulation:

$$tokens(t) = \min\left(B_c, \; tokens(t - \Delta t) + r \cdot \Delta t\right)$$

For a packet of size $L$ bytes arriving at time $t$:

$$\text{If } tokens(t) \geq L: \quad \text{conform (green), } tokens \leftarrow tokens - L$$

$$\text{If } tokens(t) < L \text{ and } tokens_e(t) \geq L: \quad \text{exceed (yellow), } tokens_e \leftarrow tokens_e - L$$

$$\text{Otherwise:} \quad \text{violate (red), packet dropped or remarked}$$

Where $tokens_e$ is the excess burst bucket with depth $B_e$.

### Burst Duration

Maximum burst duration at line rate $C$ before tokens are exhausted:

$$T_{burst} = \frac{B_c}{C - r}$$

For $B_c = 1,500,000$ bytes, $C = 100$ Mbps, $r = 10$ Mbps:

$$T_{burst} = \frac{1,500,000}{(100 - 10) \times 10^6 / 8} = \frac{1,500,000}{11,250,000} = 133.3 \text{ ms}$$

### Minimum Burst Size

The minimum burst size must accommodate at least one maximum-sized packet:

$$B_{c,min} = MTU = 1500 \text{ bytes (Ethernet)}$$

For policing, Cisco recommends:

$$B_c = CIR \times T_c = CIR \times \frac{1}{f_{refill}}$$

Where $T_c$ is the refill interval (typically 125 ms on Cisco IOS, 1 ms on high-end platforms).

### Worked Example

Police to 10 Mbps with 125 ms refill interval:

$$B_c = \frac{10 \times 10^6}{8} \times 0.125 = 156,250 \text{ bytes}$$

At this burst size, a 10 Mbps flow can burst up to 156 KB (approximately 104 full-size packets) before being rate-limited.

---

## 2. Dual-Rate Token Bucket (trTCM, RFC 2698)

### The Problem

Single-rate policers cannot distinguish between sustained overages and short bursts. The dual-rate three-color marker adds a Peak Information Rate (PIR) for finer-grained control.

### The Model

Two independent token buckets:

$$\text{Committed bucket } C: \quad \text{depth } B_c, \quad \text{fill rate } CIR$$

$$\text{Peak bucket } P: \quad \text{depth } B_p, \quad \text{fill rate } PIR$$

Classification logic (packet size $L$):

$$\text{If } P_{tokens} < L: \quad \text{red (violate)}$$

$$\text{Else if } C_{tokens} < L: \quad \text{yellow (exceed), } P_{tokens} \leftarrow P_{tokens} - L$$

$$\text{Else:} \quad \text{green (conform), } C_{tokens} \leftarrow C_{tokens} - L, \; P_{tokens} \leftarrow P_{tokens} - L$$

### Rate Relationship

$$CIR \leq PIR \leq \text{line rate}$$

$$B_c \leq B_p \text{ (typically)}$$

### Worked Example

A customer purchases 50 Mbps CIR with 100 Mbps PIR:

| Traffic Rate | Color | Action |
|:---|:---|:---|
| $\leq$ 50 Mbps | Green | Forward, DSCP unchanged |
| 50-100 Mbps | Yellow | Forward, remark DSCP to AF12 (higher drop priority) |
| $>$ 100 Mbps | Red | Drop |

This allows bursting to 100 Mbps when the network is uncongested, while guaranteeing 50 Mbps during congestion (yellow traffic is dropped first by WRED downstream).

---

## 3. Weighted Fair Queuing — Fairness Proof

### The Problem

Why is WFQ fair, and how does it handle flows with different packet sizes?

### Generalized Processor Sharing (GPS) Model

WFQ approximates the ideal GPS model, which provides bit-by-bit round-robin service. Under GPS, the service rate for flow $i$ at time $t$ is:

$$r_i(t) = \frac{\phi_i}{\sum_{j \in A(t)} \phi_j} \cdot C$$

Where:
- $\phi_i$ = weight of flow $i$
- $A(t)$ = set of active (backlogged) flows at time $t$
- $C$ = link capacity

### WFQ Finish Time Calculation

WFQ assigns each packet a virtual finish time and serves packets in order of earliest finish time.

For packet $k$ of flow $i$, arriving at time $a_i^k$ with length $L_i^k$:

$$F_i^k = \max(F_i^{k-1}, \; V(a_i^k)) + \frac{L_i^k}{\phi_i}$$

Where $V(t)$ is the virtual time function:

$$\frac{dV}{dt} = \frac{C}{\sum_{j \in A(t)} \phi_j}$$

### Fairness Guarantee

For any interval $(\tau_1, \tau_2)$ during which both flows $i$ and $j$ are backlogged:

$$\frac{W_i(\tau_1, \tau_2)}{W_j(\tau_1, \tau_2)} \geq \frac{\phi_i}{\phi_j} - \frac{L_{max}}{W_j(\tau_1, \tau_2)}$$

Where $W_i$ is the total service (bytes) received by flow $i$. As the interval grows, the ratio converges to $\phi_i / \phi_j$, proving long-term fairness.

### WFQ vs DWRR Comparison

| Property | WFQ | DWRR |
|:---|:---|:---|
| Fairness | Near-ideal GPS | Slightly less fair |
| Complexity | $O(\log N)$ per packet (sorted finish times) | $O(1)$ per packet |
| Packet-size fairness | Yes (virtual finish time) | Yes (deficit counter) |
| Implementation | Software (complex) | Hardware-friendly |
| Latency bound | $\frac{L_{max}}{C} + \frac{L_{max}}{\phi_i / \sum \phi_j \cdot C}$ | Depends on quantum |

DWRR is preferred in hardware because it achieves $O(1)$ per-packet processing with nearly identical fairness properties, trading a slightly weaker latency bound for deterministic processing time.

---

## 4. DWRR — Deficit Counter Mathematics

### The Algorithm

Each queue $q$ has:
- Weight $w_q$ (relative share)
- Quantum $Q_q = w_q \times Q_{base}$ (bytes per round)
- Deficit counter $DC_q$ (bytes, initialized to 0)

Per round:

$$DC_q \leftarrow DC_q + Q_q$$

While $DC_q \geq L_{head}$ (size of head-of-queue packet):

$$\text{Dequeue packet, } DC_q \leftarrow DC_q - L_{head}$$

If queue is empty after draining: $DC_q \leftarrow 0$

### Bandwidth Allocation

The guaranteed bandwidth for queue $q$:

$$BW_q = \frac{w_q}{\sum_{j} w_j} \times C$$

### Worked Example (Junos Scheduler Weights)

Link speed $C = 1$ Gbps. Four queues:

| Queue | Forwarding Class | Weight | Transmit Rate |
|:---|:---|:---:|:---:|
| Q0 | best-effort | 50% | remainder |
| Q1 | expedited-forwarding | 15% | strict-high |
| Q2 | assured-forwarding | 30% | 30% |
| Q3 | network-control | 5% | 5% |

With $Q_{base} = 1500$ bytes:

$$Q_{BE} = 0.50 \times 1500 = 750 \text{ bytes}$$

$$Q_{AF} = 0.30 \times 1500 = 450 \text{ bytes}$$

$$Q_{NC} = 0.05 \times 1500 = 75 \text{ bytes}$$

The EF queue uses strict priority with a policer (not DWRR). Without the policer, worst-case EF starvation of other queues:

$$\text{Max EF delay to others} = \frac{L_{max,EF}}{C} = \frac{1500 \times 8}{10^9} = 12 \; \mu s$$

One maximum-sized EF packet delays DWRR by 12 microseconds. The policer limits total EF throughput to 15% of the link.

---

## 5. Delay, Jitter & Loss Budgets

### The Problem

Voice and video have strict requirements. How do you calculate whether a QoS design meets them?

### ITU-T G.114 / Codec Requirements

| Application | Max One-Way Delay | Max Jitter | Max Packet Loss |
|:---|:---:|:---:|:---:|
| Voice (G.711) | 150 ms | 30 ms | 1% |
| Voice (G.729) | 150 ms | 30 ms | 2% |
| Interactive Video (H.264) | 200 ms | 50 ms | 0.1% |
| Broadcast Video (MPEG) | 400 ms | 50 ms | $10^{-6}$ |
| Real-time Gaming | 100 ms | 20 ms | 1% |
| Signaling (SIP/H.323) | 500 ms | N/A | 0.1% |

### End-to-End Delay Budget

Total one-way delay:

$$D_{total} = D_{codec} + D_{packetization} + D_{serialization} + D_{propagation} + D_{queuing} + D_{dejitter}$$

#### Component Calculations

**Codec delay** (algorithm-dependent):

$$D_{codec,G.711} = 1 \text{ ms (no compression)}$$

$$D_{codec,G.729} = 25 \text{ ms (10 ms frame + 5 ms lookahead + 10 ms decode)}$$

**Packetization delay** (collecting samples into a packet):

$$D_{packetization} = \frac{\text{samples per packet}}{\text{sample rate}} = \frac{160}{8000} = 20 \text{ ms (G.711, 20ms frame)}$$

**Serialization delay** (putting bits on the wire):

$$D_{serialization} = \frac{L_{packet} \times 8}{\text{link speed}}$$

For a G.711 voice packet (218 bytes with IP/UDP/RTP headers) on various links:

| Link Speed | Serialization Delay |
|:---:|:---:|
| 64 kbps | 27.25 ms |
| 256 kbps | 6.81 ms |
| 1 Mbps | 1.74 ms |
| 10 Mbps | 0.17 ms |
| 100 Mbps | 0.017 ms |
| 1 Gbps | 0.0017 ms |

Serialization delay is only significant on links below 1 Mbps.

**Propagation delay** (speed of light in fiber):

$$D_{propagation} = \frac{d}{v} = \frac{d}{2 \times 10^8} \text{ seconds}$$

Where $d$ = distance in meters, $v$ = speed of light in fiber ($\approx 2 \times 10^8$ m/s).

| Distance | Propagation Delay |
|:---:|:---:|
| 1 km | 5 $\mu$s |
| 100 km | 0.5 ms |
| 1,000 km | 5 ms |
| 5,000 km (coast-to-coast US) | 25 ms |
| 10,000 km (US-Europe) | 50 ms |

**Queuing delay** (waiting behind other packets):

$$D_{queuing} = \frac{\sum_{i=1}^{n} L_i}{C} \quad \text{(n = packets ahead in queue)}$$

With proper QoS (strict priority for EF): $D_{queuing} \approx 0$ to $L_{max}/C$.

Without QoS (FIFO): $D_{queuing}$ is unbounded during congestion.

**Dejitter buffer** (absorbs jitter variation):

$$D_{dejitter} = 2 \times J_{max} \quad \text{(typically 40-60 ms)}$$

### Complete Budget — Worked Example

VoIP call across a 3-hop enterprise network, G.711 codec:

| Component | Delay |
|:---|:---:|
| Codec (G.711) | 1 ms |
| Packetization (20 ms frame) | 20 ms |
| Serialization (1 Gbps, 3 hops) | 0.005 ms |
| Propagation (50 km total) | 0.25 ms |
| Queuing (3 hops, strict-priority EF) | 0.036 ms |
| Dejitter buffer | 40 ms |
| **Total one-way** | **61.3 ms** |

This is well within the 150 ms budget. Even with 10 hops and transcontinental distances, strict-priority queuing keeps the queuing component negligible.

### Jitter Calculation

Jitter = variation in inter-packet delay:

$$J_n = J_{n-1} + \frac{|D_n - D_{n-1}| - J_{n-1}}{16}$$

This is the RFC 3550 smoothed jitter estimate (exponentially weighted moving average with gain 1/16).

---

## 6. DSCP-to-Queue Mapping Design

### The Problem

How do you map 64 possible DSCP values into 4 or 8 hardware queues while preserving the DiffServ class hierarchy?

### 4-Queue Design (Standard Enterprise)

| Queue | Priority | Forwarding Class | DSCP Values | Bandwidth | Buffer |
|:---:|:---|:---|:---|:---:|:---:|
| Q3 | Strict | network-control | CS6 (48), CS7 (56) | 5% | 5% |
| Q2 | Strict | expedited-forwarding | EF (46), CS5 (40) | 15% | 10% |
| Q1 | DWRR | assured-forwarding | AF1x-AF4x, CS2-CS4 | 30% | 25% |
| Q0 | DWRR | best-effort | BE (0), CS1 (8) | 50% (remainder) | 60% |

### 8-Queue Design (Service Provider / Data Center)

| Queue | Priority | DSCP | Bandwidth |
|:---:|:---|:---|:---:|
| Q7 | Strict | CS7 (56) | 1% |
| Q6 | Strict | CS6 (48) | 4% |
| Q5 | Strict | EF (46) | 10% |
| Q4 | DWRR | CS5 (40), AF4x (34,36,38) | 15% |
| Q3 | DWRR | CS4 (32), AF3x (26,28,30) | 15% |
| Q2 | DWRR | CS3 (24), AF2x (18,20,22) | 15% |
| Q1 | DWRR | CS2 (16), AF1x (10,12,14) | 15% |
| Q0 | DWRR | BE (0), CS1 (8) | 25% (remainder) |

### Queue Starvation Analysis

With strict priority queues, lower-priority DWRR queues receive service only when all strict queues are empty.

Effective bandwidth available to DWRR queues:

$$BW_{DWRR} = C - \sum_{q \in SP} \text{actual\_rate}_q$$

If strict-priority queues are not policed, worst case:

$$BW_{DWRR} = C - C = 0 \quad \text{(total starvation)}$$

This is why you must always police strict-priority queues:

$$\text{SP policer rate} \leq \text{SP allocated \%} \times C$$

For a 1 Gbps link with 15% EF allocation:

$$\text{EF policer} = 0.15 \times 10^9 = 150 \text{ Mbps}$$

$$BW_{DWRR,guaranteed} = 10^9 - 150 \times 10^6 = 850 \text{ Mbps}$$

---

## 7. WRED Mathematics — Drop Probability Functions

### The Problem

How does WRED calculate drop probability, and how do you tune min-threshold, max-threshold, and mark-probability-denominator for each drop precedence level?

### Linear Drop Probability

For queue depth $q$:

$$P_{drop}(q) = \begin{cases} 0 & \text{if } q < q_{min} \\ \frac{q - q_{min}}{q_{max} - q_{min}} \times \frac{1}{MPD} & \text{if } q_{min} \leq q \leq q_{max} \\ 1 & \text{if } q > q_{max} \end{cases}$$

Where $MPD$ = mark probability denominator (e.g., 10 means max 10% drop probability at $q_{max}$).

### Multi-Drop-Precedence Design

For an AF class with three drop precedences:

| Drop Precedence | DSCP | $q_{min}$ | $q_{max}$ | $MPD$ | Max $P_{drop}$ |
|:---|:---:|:---:|:---:|:---:|:---:|
| Low (AFx1) | AF11, AF21, AF31, AF41 | 70% | 100% | 10 | 10% |
| Medium (AFx2) | AF12, AF22, AF32, AF42 | 50% | 90% | 5 | 20% |
| High (AFx3) | AF13, AF23, AF33, AF43 | 30% | 80% | 2 | 50% |

### Average Queue Depth (EWMA)

WRED uses an exponential weighted moving average of queue depth, not instantaneous depth:

$$\bar{q}_n = (1 - w) \times \bar{q}_{n-1} + w \times q_n$$

Where $w$ is the weight factor (typically $2^{-n}$, where $n$ is configurable, often $n = 9$, so $w = 1/512$).

Small $w$: slow to react, absorbs bursts, fewer drops during short spikes
Large $w$: fast to react, drops during short bursts, closer to tail-drop behavior

### TCP Throughput Under WRED

The Mathis formula for TCP throughput under random loss:

$$T = \frac{MSS}{RTT \times \sqrt{2p/3}}$$

Where $p$ = loss probability. For MSS = 1460, RTT = 50 ms:

| Loss Rate $p$ | TCP Throughput |
|:---:|:---:|
| 0.01% | 1.27 Gbps |
| 0.1% | 402 Mbps |
| 1% | 127 Mbps |
| 5% | 56.8 Mbps |
| 10% | 40.2 Mbps |

WRED aims to keep loss rates in the 0.01-1% range, preserving high TCP throughput while avoiding the synchronization catastrophe of tail drop.

---

## 8. End-to-End QoS Architecture

### The Problem

How do you maintain consistent QoS treatment across access, distribution, core, and WAN segments with different link speeds and technologies?

### Per-Hop Behavior Consistency

Each hop must implement the same PHB semantics even if the mechanism differs:

| Segment | Link Speed | Mechanism | Notes |
|:---|:---:|:---|:---|
| Access (edge) | 1 Gbps | MF classification, DSCP marking, ingress policing | Trust boundary |
| Distribution | 10 Gbps | BA classification (trust DSCP), queuing, WRED | Aggregate policing |
| Core | 100 Gbps | BA classification, minimal queuing | Over-provisioned |
| WAN (MPLS) | 1-10 Gbps | DSCP-to-EXP map, SP+DWRR, shaping | Rate bottleneck |
| WAN (Internet) | varies | No QoS guarantees | Best effort only |

### Bandwidth Allocation Cascade

At each hop, the bandwidth allocation must be re-calculated for the local link speed:

$$BW_{class,hop} = \text{allocation\_\%} \times C_{hop}$$

For EF at 15% across hops:

| Hop | Link Speed | EF Bandwidth |
|:---:|:---:|:---:|
| Access | 1 Gbps | 150 Mbps |
| Distribution | 10 Gbps | 1.5 Gbps |
| Core | 100 Gbps | 15 Gbps |
| WAN | 1 Gbps | 150 Mbps |

The bottleneck determines the end-to-end EF capacity. In this example, the WAN link at 150 Mbps is the constraint.

### Admission Control

To prevent over-subscription of the EF class:

$$\sum_{i=1}^{N} R_{EF,i} \leq BW_{EF,bottleneck}$$

For G.711 voice calls at 80 kbps each (including headers), through a 1 Gbps WAN with 15% EF:

$$N_{max} = \frac{150 \times 10^6}{80 \times 10^3} = 1,875 \text{ simultaneous calls}$$

With a safety margin of 75% utilization:

$$N_{safe} = 0.75 \times 1,875 = 1,406 \text{ calls}$$

### DSCP Transparency

ISPs may remark or zero DSCP values. To preserve QoS across untrusted domains:

1. **MPLS tunnel**: Encapsulate in MPLS, carry DSCP in EXP bits, restore on egress
2. **GRE/IPsec tunnel**: Copy inner DSCP to outer header, ISP may remark outer only
3. **Re-classification at PE**: Ingress PE re-marks based on policy, not received DSCP

---

## 9. Junos Scheduler Weight Calculations

### The Problem

How do Junos scheduler transmit-rate percentages translate to actual bandwidth, and how do remainder and excess-rate interact?

### Transmit Rate Modes

**Percentage mode**: Guaranteed minimum bandwidth.

$$BW_{guaranteed} = \frac{\text{transmit-rate-percent}}{100} \times C_{interface}$$

**Remainder mode**: Gets all bandwidth not allocated to percentage-based schedulers.

$$BW_{remainder} = C - \sum_{q \in \text{percent}} BW_q - BW_{strict}$$

**Exact mode**: Fixed rate in bps.

$$BW_{exact} = \text{transmit-rate (configured value)}$$

### Excess Rate Distribution

When some queues are idle, their unused bandwidth is redistributed. Junos `excess-rate` controls how excess is shared:

$$BW_{excess,q} = \frac{w_{excess,q}}{\sum_{j \in active} w_{excess,j}} \times BW_{unused}$$

If `excess-rate proportional` is set, the excess weight equals the transmit-rate weight. Otherwise, excess is distributed equally.

### Worked Example

1 Gbps interface, four schedulers:

| Scheduler | transmit-rate | excess-rate | Guaranteed BW |
|:---|:---|:---|:---:|
| EF-SCHED | 15% (strict) | none | 150 Mbps |
| AF-SCHED | 30% | proportional | 300 Mbps |
| NC-SCHED | 5% | proportional | 50 Mbps |
| BE-SCHED | remainder | proportional | 500 Mbps |

If EF and NC queues are empty (650 Mbps available for AF + BE):

$$BW_{AF,actual} = 300 + \frac{30}{30 + 50} \times 200 = 300 + 75 = 375 \text{ Mbps}$$

$$BW_{BE,actual} = 500 + \frac{50}{30 + 50} \times 200 = 500 + 125 = 625 \text{ Mbps}$$

Note: BE-SCHED uses `remainder` (50%) as its proportional excess weight since remainder resolves to the remaining percentage.

### Buffer Size Calculations

Junos allocates shared memory to queues via buffer-size percentage. The total packet memory depends on the platform:

$$\text{Buffer}_{q} = \frac{\text{buffer-percent}}{100} \times M_{total}$$

For a platform with 1 MB total buffer memory:

| Queue | Buffer % | Buffer Size | Max Packets (1500B) |
|:---|:---:|:---:|:---:|
| EF | 10% | 100 KB | 66 |
| AF | 25% | 250 KB | 166 |
| NC | 5% | 50 KB | 33 |
| BE | 60% | 600 KB | 400 |

Under-sizing the EF buffer is intentional: voice packets should never queue deeply. If the EF buffer fills, it means the policer is misconfigured or there is a traffic anomaly.

---

## 10. Summary of Formulas

| Formula | Domain |
|:---|:---|
| $tokens(t) = \min(B_c, \; tokens(t-\Delta t) + r \cdot \Delta t)$ | Token bucket |
| $T_{burst} = B_c / (C - r)$ | Maximum burst duration |
| $F_i^k = \max(F_i^{k-1}, V(a_i^k)) + L_i^k / \phi_i$ | WFQ finish time |
| $r_i = (\phi_i / \sum \phi_j) \cdot C$ | GPS fair rate |
| $DC_q \leftarrow DC_q + Q_q$ | DWRR deficit counter |
| $BW_q = (w_q / \sum w_j) \times C$ | DWRR bandwidth allocation |
| $D_{total} = D_{codec} + D_{pkt} + D_{ser} + D_{prop} + D_{queue} + D_{dejit}$ | End-to-end delay budget |
| $D_{ser} = (L \times 8) / C$ | Serialization delay |
| $D_{prop} = d / (2 \times 10^8)$ | Propagation delay |
| $P_{drop} = (q - q_{min})/(q_{max} - q_{min}) \times 1/MPD$ | WRED drop probability |
| $\bar{q}_n = (1-w) \bar{q}_{n-1} + w \cdot q_n$ | WRED EWMA queue depth |
| $T = MSS / (RTT \times \sqrt{2p/3})$ | TCP throughput under loss |
| $N_{calls} = BW_{EF} / R_{call}$ | Voice call admission |

## Prerequisites

- TCP congestion control, IP header fields, 802.1Q framing, queuing theory basics, MPLS label operations

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| Token bucket check | $O(1)$ | $O(1)$ per policer |
| WFQ packet scheduling | $O(\log N)$ flows | $O(N)$ flow state |
| DWRR round processing | $O(1)$ per packet | $O(Q)$ queues |
| WRED drop decision | $O(1)$ | $O(1)$ per profile |
| MF classification (TCAM) | $O(1)$ hardware | $O(R)$ rules |
| BA classification | $O(1)$ table lookup | $O(64)$ DSCP entries |

---

*QoS is the art of making promises with finite resources. The token bucket bounds the rate, WFQ bounds the unfairness, WRED bounds the synchronization, and the delay budget bounds the design. When the math works, voice is clear, video is smooth, and bulk transfers still get their fair share. When it does not, all you have is expensive best effort.*
