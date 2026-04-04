# The Mathematics of TCP — Window Calculus & Congestion Control

> *TCP is a state machine wrapped in a feedback control system. Every packet carries implicit math: sequence arithmetic modulo 2^32, bandwidth-delay products, exponential backoff, and the AIMD control law that keeps the internet from collapsing.*

---

## 1. The Bandwidth-Delay Product (Capacity Planning)

### The Problem

How much data can be "in flight" between sender and receiver? This determines the optimal TCP window size — set it too small and you waste bandwidth; too large and you cause congestion.

### The Formula

$$BDP = B \times RTT$$

Where:
- $BDP$ = bandwidth-delay product (bytes)
- $B$ = bottleneck bandwidth (bytes/sec)
- $RTT$ = round-trip time (seconds)

### Worked Examples

| Link | Bandwidth | RTT | BDP | Window Needed |
|:---|:---|:---:|:---:|:---:|
| LAN (1G) | 125 MB/s | 0.5 ms | 62,500 B | 64 KB |
| Metro (10G) | 1.25 GB/s | 5 ms | 6,250,000 B | ~6 MB |
| Cross-country (1G) | 125 MB/s | 60 ms | 7,500,000 B | ~7.5 MB |
| Transatlantic (10G) | 1.25 GB/s | 80 ms | 100,000,000 B | ~100 MB |
| Satellite (100M) | 12.5 MB/s | 600 ms | 7,500,000 B | ~7.5 MB |

### Why 65,535 Bytes Breaks

The original TCP header has a 16-bit window field: $2^{16} - 1 = 65,535$ bytes max. For any link where $BDP > 65,535$, the pipe is underutilized. RFC 7323 **Window Scale** option adds a shift count $S$ (0-14):

$$W_{effective} = W_{header} \times 2^S$$

Max effective window: $65,535 \times 2^{14} = 1,073,725,440$ bytes (~1 GB).

**Utilization without scaling:**

$$U = \frac{W_{max}}{BDP} = \frac{65,535}{BDP}$$

For the transatlantic link: $U = \frac{65,535}{100,000,000} = 0.065\%$ — catastrophic.

---

## 2. Sequence Number Arithmetic (Modular Arithmetic)

### The Problem

TCP sequence numbers are 32-bit unsigned integers that wrap around. How long until they wrap, and does this cause ambiguity?

### The Formula

$$T_{wrap} = \frac{2^{32}}{B}$$

Where:
- $T_{wrap}$ = time to exhaust sequence space (seconds)
- $B$ = throughput in bytes/sec

### Worked Examples

| Speed | Throughput | $T_{wrap}$ | Risk |
|:---|:---:|:---:|:---|
| 10 Mbps | 1.25 MB/s | 3,436 sec (~57 min) | None |
| 1 Gbps | 125 MB/s | 34.4 sec | Moderate |
| 10 Gbps | 1.25 GB/s | 3.44 sec | High |
| 100 Gbps | 12.5 GB/s | 0.34 sec | Critical |

At 10 Gbps+, sequence numbers wrap faster than the Maximum Segment Lifetime (MSL = 120 sec). Solution: **TCP Timestamps** (RFC 7323) add a 32-bit timestamp to disambiguate — called PAWS (Protection Against Wrapped Sequences).

### Sequence Comparison

TCP uses modular comparison. Sequence $A$ is "less than" $B$ if:

$$0 < (B - A) \mod 2^{32} < 2^{31}$$

This splits the 32-bit space into two halves: the "past" and the "future" relative to any reference point.

---

## 3. RTT Estimation — Jacobson/Karels Algorithm

### The Problem

TCP must estimate round-trip time to set retransmission timeouts. Too short = spurious retransmissions. Too long = wasted time on real losses.

### The Formulas (RFC 6298)

On each new RTT sample $R$:

$$SRTT_{new} = (1 - \alpha) \times SRTT_{old} + \alpha \times R$$

$$RTTVAR_{new} = (1 - \beta) \times RTTVAR_{old} + \beta \times |SRTT_{old} - R|$$

$$RTO = SRTT + \max(G, 4 \times RTTVAR)$$

Where:
- $\alpha = 1/8$ (SRTT smoothing factor)
- $\beta = 1/4$ (variance smoothing factor)
- $G$ = clock granularity (typically 1 ms)
- $RTO$ = Retransmission Timeout

### Worked Example

Starting with $SRTT = 100$ ms, $RTTVAR = 10$ ms, new sample $R = 130$ ms:

$$SRTT = \frac{7}{8}(100) + \frac{1}{8}(130) = 87.5 + 16.25 = 103.75 \text{ ms}$$

$$RTTVAR = \frac{3}{4}(10) + \frac{1}{4}|100 - 130| = 7.5 + 7.5 = 15 \text{ ms}$$

$$RTO = 103.75 + 4(15) = 163.75 \text{ ms}$$

### Exponential Backoff on Timeout

When RTO fires without an ACK:

$$RTO_{n} = RTO_0 \times 2^n$$

| Attempt | RTO (starting 200 ms) |
|:---:|:---:|
| 1 | 200 ms |
| 2 | 400 ms |
| 3 | 800 ms |
| 4 | 1.6 sec |
| 5 | 3.2 sec |
| 6 | 6.4 sec |

Capped at 60 seconds (RFC 6298). Total time before giving up (typically 15 retries): ~13.6 minutes.

---

## 4. Congestion Control — AIMD, Cubic, and BBR

### 4a. Classic AIMD (Reno)

**Additive Increase:** Each RTT, grow window by 1 MSS:

$$W_{n+1} = W_n + 1$$

**Multiplicative Decrease:** On loss, halve the window:

$$W_{n+1} = \frac{W_n}{2}$$

### Steady-State Throughput (Mathis Formula)

$$Throughput = \frac{MSS}{RTT \times \sqrt{p}}$$

Where $p$ = packet loss rate. This is the **TCP throughput equation**.

| Loss Rate $p$ | RTT = 20 ms, MSS = 1460 | RTT = 100 ms, MSS = 1460 |
|:---:|:---:|:---:|
| 0.01% (10^-4) | 730 Mbps | 146 Mbps |
| 0.1% (10^-3) | 231 Mbps | 46 Mbps |
| 1% (10^-2) | 73 Mbps | 14.6 Mbps |
| 5% | 32.7 Mbps | 6.5 Mbps |

### 4b. TCP Cubic

Cubic uses a cubic function of time since last loss:

$$W(t) = C(t - K)^3 + W_{max}$$

Where:
- $C = 0.4$ (scaling constant)
- $K = \sqrt[3]{\frac{W_{max} \times \beta}{C}}$ where $\beta = 0.7$ (multiplicative decrease factor)
- $W_{max}$ = window size at last loss event

**Key property:** Growth is independent of RTT, making it fairer across paths with different latencies. Near $W_{max}$, growth slows (concave), then accelerates past it (convex) — probing for new capacity.

### 4c. BBR (Bottleneck Bandwidth and RTT)

BBR estimates two parameters:

$$\hat{B} = \max\left(\frac{delivered}{interval}\right) \quad \text{(max bandwidth)}$$

$$\hat{RTT}_{min} = \min(RTT_{samples}) \quad \text{(min RTT)}$$

Target operating point: $BDP = \hat{B} \times \hat{RTT}_{min}$

BBR cycles through 4 phases: Startup ($2/\ln 2$ gain), Drain, ProbeBW (8 RTT cycles), ProbeRTT.

---

## 5. Slow Start — Exponential Growth Phase

### The Formula

$$W_n = W_0 \times 2^n$$

Where $n$ = number of RTTs. Window doubles each RTT (exponential growth).

### Time to Fill a Pipe

$$n = \lceil \log_2\left(\frac{BDP}{MSS}\right) \rceil$$

| BDP | MSS | RTTs to fill | At 50 ms RTT |
|:---:|:---:|:---:|:---:|
| 64 KB | 1,460 | 6 | 300 ms |
| 1 MB | 1,460 | 10 | 500 ms |
| 10 MB | 1,460 | 13 | 650 ms |
| 100 MB | 1,460 | 17 | 850 ms |

---

## 6. Three-Way Handshake — State Machine Timing

### Connection Establishment

$$T_{connect} = 1.5 \times RTT$$

(SYN → SYN-ACK → ACK, but the ACK can carry data, so it's effectively 1 RTT + processing).

### Connection with TLS 1.3

$$T_{TLS1.3} = 2 \times RTT \quad \text{(1 RTT TCP + 1 RTT TLS)}$$

$$T_{TLS1.3\_0RTT} = 1 \times RTT \quad \text{(resumed session)}$$

### Nagle + Delayed ACK Interaction

Nagle's algorithm: don't send small segments while ACKs are outstanding.
Delayed ACK: wait up to 200 ms (or 2nd segment) before ACKing.

**Worst case latency for small writes:**

$$T_{Nagle+DelACK} = T_{DelACK\_timer} = 200 \text{ ms}$$

This is why interactive protocols (SSH, gaming) disable Nagle (`TCP_NODELAY`).

---

## 7. Summary of Formulas

| Formula | Math Type | Domain |
|:---|:---|:---|
| $BDP = B \times RTT$ | Linear (product) | Capacity planning |
| $2^{32} / B$ | Inverse proportion | Sequence wrap time |
| $(1-\alpha) \cdot SRTT + \alpha \cdot R$ | EWMA (exponential weighted moving average) | RTT estimation |
| $RTO \times 2^n$ | Exponential growth | Backoff timing |
| $MSS / (RTT \sqrt{p})$ | Inverse square root | Throughput modeling |
| $C(t-K)^3 + W_{max}$ | Cubic polynomial | Congestion control |
| $W_0 \times 2^n$ | Exponential growth | Slow start |
| $(1-\beta) \cdot V + \beta \cdot |S - R|$ | Variance estimation | Jitter tracking |

## Prerequisites

- sequence arithmetic, exponential functions, sliding window algorithms, feedback control

---

*TCP's math is a live feedback control system — every connection on your machine is running these equations in real time, adjusting windows, estimating RTTs, and probing for bandwidth thousands of times per second.*
