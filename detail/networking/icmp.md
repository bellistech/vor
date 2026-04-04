# The Mathematics of ICMP — RTT Statistics, Packet Loss & Path MTU Discovery

> *ICMP is the internet's measurement protocol. Every ping carries implicit statistics: RTT distributions, loss probabilities, and MTU constraints. Understanding these numbers transforms ICMP from a simple "is it up?" tool into a precise network diagnostic instrument.*

---

## 1. RTT Statistics (Descriptive Statistics)

### The Problem

Ping reports min/avg/max/stddev of round-trip times. How should these be interpreted, and what do they reveal about the network path?

### The Formulas

Given $n$ RTT samples $R_1, R_2, \ldots, R_n$:

**Mean (average):**
$$\bar{R} = \frac{1}{n} \sum_{i=1}^{n} R_i$$

**Standard deviation:**
$$\sigma = \sqrt{\frac{1}{n-1} \sum_{i=1}^{n} (R_i - \bar{R})^2}$$

**Jitter (inter-packet delay variation):**
$$J = \frac{1}{n-1} \sum_{i=2}^{n} |R_i - R_{i-1}|$$

**Percentiles (for service-level objectives):**
- p50 (median) — typical user experience
- p95 — tail latency (5% of requests slower)
- p99 — worst-case for most users

### Worked Examples

Sample RTTs (ms): 10.2, 11.5, 10.8, 45.3, 11.1, 10.6, 10.9, 11.3, 10.7, 11.0

$$\bar{R} = \frac{143.4}{10} = 14.34 \text{ ms}$$

$$\sigma = \sqrt{\frac{\sum(R_i - 14.34)^2}{9}} \approx 10.87 \text{ ms}$$

The 45.3 ms sample is an outlier (>3$\sigma$ above median). The median (10.85 ms) is a much better measure of typical RTT than the mean (14.34 ms) because RTT distributions have long right tails.

### RTT Distribution Characteristics

| Network Type | Typical $\bar{R}$ | Typical $\sigma$ | $\sigma / \bar{R}$ (CoV) | Interpretation |
|:---|:---:|:---:|:---:|:---|
| LAN (switched) | 0.3 ms | 0.05 ms | 0.17 | Very stable |
| Metro fiber | 5 ms | 0.5 ms | 0.10 | Stable |
| Cross-continent | 60 ms | 3 ms | 0.05 | Stable (propagation dominated) |
| WiFi (loaded) | 15 ms | 20 ms | 1.33 | Highly variable |
| Cellular (4G) | 40 ms | 25 ms | 0.63 | Variable |
| Satellite (GEO) | 600 ms | 10 ms | 0.02 | Stable but slow |

A coefficient of variation (CoV = $\sigma / \bar{R}$) greater than 0.5 indicates significant jitter, typically caused by queuing delays or wireless contention.

---

## 2. Packet Loss Probability (Bernoulli Trials)

### The Problem

If we send $n$ pings and observe $k$ losses, what is the true loss rate and our confidence in that estimate?

### The Formula

Observed loss rate:

$$\hat{p} = \frac{k}{n}$$

Confidence interval (95%, normal approximation for large $n$):

$$\hat{p} \pm 1.96 \sqrt{\frac{\hat{p}(1-\hat{p})}{n}}$$

### Worked Examples

| Sent $n$ | Lost $k$ | $\hat{p}$ | 95% CI |
|:---:|:---:|:---:|:---:|
| 10 | 1 | 10.0% | [0%, 29.2%] |
| 100 | 1 | 1.0% | [0%, 2.96%] |
| 1,000 | 10 | 1.0% | [0.38%, 1.62%] |
| 10,000 | 100 | 1.0% | [0.80%, 1.20%] |
| 100 | 0 | 0.0% | [0%, 3.6%] (rule of 3) |

### Minimum Sample Size

To distinguish a loss rate of $p$ from 0 with 95% confidence:

$$n \geq \frac{3}{p} \quad \text{(rule of three)}$$

| Target Loss Rate | Min Samples | At 1 ping/sec |
|:---:|:---:|:---:|
| 10% | 30 | 30 seconds |
| 1% | 300 | 5 minutes |
| 0.1% | 3,000 | 50 minutes |
| 0.01% | 30,000 | 8.3 hours |

### Consecutive Loss Probability

If packet loss is independent with rate $p$, the probability of $k$ consecutive losses:

$$P(k \text{ consecutive}) = p^k$$

| Loss Rate $p$ | 2 in a row | 3 in a row | 5 in a row |
|:---:|:---:|:---:|:---:|
| 1% | 0.01% | 0.0001% | $10^{-10}$ |
| 5% | 0.25% | 0.0125% | $3.1 \times 10^{-7}$ |
| 10% | 1.0% | 0.1% | 0.001% |
| 20% | 4.0% | 0.8% | 0.032% |

If you observe consecutive losses more often than $p^k$ predicts, the loss is **bursty** (correlated), not random. This is common with buffer overflow events.

---

## 3. Path MTU Discovery Mathematics (Binary Search)

### The Problem

PMTUD finds the maximum packet size that traverses a path without fragmentation. How efficiently can we search for it?

### The Formula

Binary search between $MTU_{min}$ and $MTU_{max}$:

$$\text{probes} = \lceil \log_2(MTU_{max} - MTU_{min}) \rceil$$

### Worked Examples

| $MTU_{min}$ | $MTU_{max}$ | Range | Probes Needed |
|:---:|:---:|:---:|:---:|
| 68 | 1500 | 1432 | 11 |
| 1200 | 1500 | 300 | 9 |
| 1400 | 1500 | 100 | 7 |
| 1450 | 1500 | 50 | 6 |

### Total PMTUD Time

$$T_{PMTUD} = \text{probes} \times (RTT + T_{timeout})$$

The timeout is needed for probes that are silently dropped (ICMP Type 3 Code 4 might be filtered):

| RTT | Timeout | Probes | Total Time |
|:---:|:---:|:---:|:---:|
| 10 ms | 2s | 11 | 0.11s (all succeed) to 22.1s (all timeout) |
| 50 ms | 2s | 11 | 0.55s to 22.6s |
| 200 ms | 3s | 11 | 2.2s to 35.2s |

### Common MTU Breakpoints

| Path Element | MTU | Payload after IP+ICMP |
|:---|:---:|:---:|
| Ethernet | 1500 | 1472 |
| PPPoE (DSL) | 1492 | 1464 |
| IPv6 tunnel (6in4) | 1480 | 1452 |
| GRE tunnel | 1476 | 1448 |
| IPsec (ESP+tunnel) | ~1400 | ~1372 |
| WireGuard | 1420 | 1392 |
| VPN conservative | 1400 | 1372 |

---

## 4. ICMP Rate Limiting (Token Bucket)

### The Problem

The Linux kernel rate-limits ICMP error messages to prevent amplification. How does this affect diagnostic accuracy?

### The Formula

Linux uses a token bucket with:
- Rate: 1 token per `icmp_ratelimit` milliseconds (default 1000 ms)
- Burst: 1 token (no burst)

Maximum ICMP error rate:

$$R_{max} = \frac{1000}{icmp\_ratelimit} \text{ errors/sec}$$

### Impact on Diagnostics

If a host receives error-triggering packets at rate $\lambda$:

$$P(\text{error suppressed}) = \max\left(0, 1 - \frac{R_{max}}{\lambda}\right)$$

| Incoming Error Rate $\lambda$ | Rate Limit (1/s) | Errors Sent | Suppressed |
|:---:|:---:|:---:|:---:|
| 0.5 /s | 1 /s | 0.5 /s | 0% |
| 1 /s | 1 /s | 1 /s | 0% |
| 10 /s | 1 /s | 1 /s | 90% |
| 100 /s | 1 /s | 1 /s | 99% |
| 1000 /s | 1 /s | 1 /s | 99.9% |

This means during a port scan or routing flap, most ICMP errors are silently suppressed, making diagnosis harder.

---

## 5. Ping Flood Analysis (Bandwidth Calculation)

### The Problem

How much bandwidth does ICMP consume during testing, and what constitutes a denial-of-service level of traffic?

### The Formula

Each ICMP echo packet on Ethernet:

$$S_{frame} = 14_{Eth} + 20_{IP} + 8_{ICMP} + P_{payload} + 4_{FCS}$$

Default ping payload is 56 bytes (Linux) or 32 bytes (Windows):

$$S_{default} = 14 + 20 + 8 + 56 + 4 = 102 \text{ bytes}$$

Bandwidth at rate $R$ packets/sec:

$$B = R \times S_{frame} \times 8 \text{ bits/sec}$$

### Worked Examples

| Rate (pps) | Payload | Frame Size | Bandwidth | Bandwidth (bidirectional) |
|:---:|:---:|:---:|:---:|:---:|
| 1 | 56 B | 102 B | 816 bps | 1.6 kbps |
| 100 | 56 B | 102 B | 81.6 kbps | 163 kbps |
| 1,000 | 56 B | 102 B | 816 kbps | 1.6 Mbps |
| 10,000 | 1472 B | 1518 B | 121 Mbps | 243 Mbps |
| 100,000 | 1472 B | 1518 B | 1.21 Gbps | 2.43 Gbps |

A flood ping with max-size packets at 100K pps can saturate a 1 Gbps link.

---

## 6. Summary of Formulas

| Formula | Math Type | Domain |
|:---|:---|:---|
| $\bar{R} = \sum R_i / n$ | Arithmetic mean | RTT average |
| $\sigma = \sqrt{\sum(R_i - \bar{R})^2 / (n-1)}$ | Standard deviation | RTT variability |
| $\hat{p} \pm 1.96\sqrt{\hat{p}(1-\hat{p})/n}$ | Confidence interval | Loss estimation |
| $p^k$ | Geometric probability | Consecutive loss |
| $\lceil \log_2(\text{range}) \rceil$ | Binary search | PMTUD probes |
| $R \times S \times 8$ | Linear (throughput) | Flood bandwidth |
| $1000 / \text{ratelimit}$ | Reciprocal | Error rate cap |

## Prerequisites

- descriptive statistics, Bernoulli trials, binary search, confidence intervals

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| RTT calculation (per ping) | $O(1)$ | $O(1)$ |
| Running statistics (min/avg/max/stddev) | $O(1)$ per sample | $O(1)$ (Welford's) |
| PMTUD binary search | $O(\log M)$ probes | $O(1)$ |
| Loss rate estimation | $O(n)$ samples | $O(1)$ counter |
| ICMP rate limit check | $O(1)$ token bucket | $O(1)$ |

---

*ICMP's simplicity masks real statistical depth. A "1% packet loss" measured from 100 pings has a confidence interval so wide it could be anywhere from 0% to 3%. You need 30,000 samples to reliably detect 0.01% loss. The math tells you exactly how long to ping before trusting the result.*
