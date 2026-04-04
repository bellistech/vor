# The Mathematics of MTU — Fragmentation Cost & Path Discovery Convergence

> *MTU configuration appears trivial until the mathematics reveal its cascading effects: fragmentation doubles reassembly memory and CPU, PMTUD convergence depends on ICMP round-trip reliability, tunnel overhead stacks multiplicatively, and the throughput penalty of suboptimal MSS follows a hyperbolic curve that punishes small packets far more than large ones.*

---

## 1. Fragmentation Cost (Reassembly Overhead)

### Fragments Per Packet

When a packet of size $S$ encounters a link with MTU $M$ (and DF bit is not set):

$$F = \left\lceil \frac{S - H_{IP}}{M - H_{IP}} \right\rceil$$

Where $H_{IP} = 20$ bytes (IPv4 header, copied to each fragment). Each fragment except the last carries $M - H_{IP}$ bytes of payload; the last carries the remainder.

For $S = 4000$ bytes, $M = 1500$:

$$F = \left\lceil \frac{4000 - 20}{1500 - 20} \right\rceil = \left\lceil \frac{3980}{1480} \right\rceil = 3 \text{ fragments}$$

### Header Overhead from Fragmentation

Total bytes on the wire after fragmentation:

$$B_{frag} = F \times H_{IP} + (S - H_{IP}) = S + (F - 1) \times H_{IP}$$

Overhead ratio:

$$\eta_{frag} = \frac{(F - 1) \times H_{IP}}{S}$$

| Original Size $S$ | MTU $M$ | Fragments $F$ | Overhead Bytes | Overhead % |
|:---:|:---:|:---:|:---:|:---:|
| 1500 | 1400 | 2 | 20 | 1.3% |
| 4000 | 1500 | 3 | 40 | 1.0% |
| 9000 | 1500 | 7 | 120 | 1.3% |
| 65535 | 1500 | 45 | 880 | 1.3% |

The header overhead is small, but the real costs are CPU (per-fragment processing) and memory (reassembly buffers).

### Reassembly Memory

The receiver must buffer all fragments until the last arrives. Worst-case memory per flow with $N$ concurrent fragmented flows:

$$M_{reasm} = N \times S_{max}$$

Where $S_{max}$ is the maximum original packet size (65535 bytes for IPv4). Linux limits this:

$$M_{reasm} \leq \text{ipfrag\_high\_thresh} \text{ (default 4 MB)}$$

Maximum concurrent reassemblies:

$$N_{max} = \left\lfloor \frac{M_{reasm}}{S_{avg}} \right\rfloor$$

| Average Fragment Set Size | Max Concurrent | At 4 MB Limit |
|:---:|:---:|:---:|
| 1500 bytes | 2,730 | Adequate for most workloads |
| 9000 bytes | 455 | Tight for jumbo-frame environments |
| 65535 bytes | 62 | Easily exhausted by attack |

Fragment-based DoS attacks exploit this: sending incomplete fragment sets fills the reassembly buffer, dropping legitimate traffic.

---

## 2. PMTUD Convergence (Round-Trip Feedback Loop)

### Discovery Rounds

Classic PMTUD converges in a number of rounds equal to the number of MTU bottlenecks on the path. For a path with $k$ distinct MTU reductions:

$$R_{rounds} = k$$

Each round requires:
1. Send a packet at current PMTU estimate
2. Receive ICMP "Fragmentation Needed" with next-hop MTU
3. Reduce PMTU and retry

Total convergence time:

$$T_{PMTUD} = \sum_{i=1}^{k} (RTT_i + T_{retransmit,i})$$

Where $RTT_i$ is the round-trip time to the $i$-th bottleneck and $T_{retransmit}$ is the retransmission timeout if the ICMP is lost.

### PMTUD with ICMP Loss

If ICMP "Frag Needed" messages are dropped with probability $p$:

Expected rounds to receive one ICMP response:

$$E[attempts] = \frac{1}{1-p}$$

Expected convergence time per bottleneck:

$$E[T_i] = \frac{RTT_i + RTO \times p/(1-p)}{1}$$

Where $RTO$ is the retransmission timeout (typically 1--3 seconds for TCP).

| ICMP Loss $p$ | Expected Attempts | Added Delay per Hop |
|:---:|:---:|:---:|
| 0% | 1.0 | 0 |
| 10% | 1.11 | 0.11 RTOs |
| 50% | 2.0 | 1.0 RTOs |
| 90% | 10.0 | 9.0 RTOs |
| 100% | $\infty$ | Black hole |

At $p = 100\%$ (ICMP completely blocked), PMTUD never converges — this is the MTU black hole condition.

---

## 3. PLPMTUD Binary Search (Probe Efficiency)

### Search Algorithm

PLPMTUD (RFC 8899) uses a binary search between $MTU_{min}$ and $MTU_{max}$:

$$R_{probes} = \left\lceil \log_2 \frac{MTU_{max} - MTU_{min}}{step} \right\rceil$$

Where $step$ is the minimum resolution (typically 1 byte).

For IPv6 ($MTU_{min} = 1280$, $MTU_{max} = 9000$):

$$R_{probes} = \left\lceil \log_2 \frac{9000 - 1280}{1} \right\rceil = \left\lceil \log_2 7720 \right\rceil = 13 \text{ probes}$$

### Probe Timing

Each probe requires a round-trip plus a timeout for loss detection:

$$T_{probe} = RTT + T_{timeout}$$

Total discovery time:

$$T_{PLPMTUD} = R_{probes} \times (RTT + T_{timeout})$$

With RTT = 10ms and timeout = 200ms:

$$T_{PLPMTUD} = 13 \times 210\text{ms} = 2.73\text{s}$$

### Advantage Over Classic PMTUD

PLPMTUD does not depend on ICMP delivery:

| Metric | Classic PMTUD | PLPMTUD |
|:---|:---:|:---:|
| ICMP dependency | Required | None |
| Probes needed | $k$ (bottlenecks) | $\lceil\log_2 R\rceil$ |
| Black hole risk | Yes ($p=1$) | No |
| Layer | Network (IP) | Transport (TCP/QUIC) |
| Accuracy | Exact (from ICMP MTU) | $\pm step$ |

---

## 4. Tunnel Overhead Stacking (Nested Encapsulation)

### Single Tunnel

Effective inner MTU:

$$M_{inner} = M_{link} - O_{tunnel}$$

### Nested Tunnels

For $n$ nested tunnels with overheads $O_1, O_2, \ldots, O_n$:

$$M_{inner} = M_{link} - \sum_{i=1}^{n} O_i$$

### Common Stacking Scenarios

| Stack | Overhead Components | Total $O$ | Inner MTU (1500) | Inner MTU (9000) |
|:---|:---|:---:|:---:|:---:|
| VXLAN | 14+20+8+8 | 50 | 1450 | 8950 |
| GRE+IPsec | 24+73 | 97 | 1403 | 8903 |
| VXLAN+IPsec | 50+73 | 123 | 1377 | 8877 |
| Geneve+WireGuard | 50+60 | 110 | 1390 | 8890 |
| GRE+GRE (double) | 24+24 | 48 | 1452 | 8952 |
| VXLAN over IPv6 | 14+40+8+8 | 70 | 1430 | 8930 |

### Overhead as Percentage of Capacity

$$\eta_{overhead} = \frac{\sum O_i}{M_{link}}$$

For VXLAN+IPsec on 1500-byte MTU: $\eta = 123/1500 = 8.2\%$

On 9000-byte MTU: $\eta = 123/9000 = 1.4\%$

Jumbo frames reduce the overhead percentage by $M_{jumbo}/M_{standard} = 6\times$.

---

## 5. MSS vs Throughput (Efficiency Curve)

### Goodput Formula

TCP goodput for a flow with MSS $m$, window size $W$ (segments), RTT $R$, and loss rate $p$:

$$G = \frac{m}{R} \times \frac{1}{\sqrt{2p/3}}$$

(Mathis formula, simplified). The key insight is that goodput scales linearly with MSS.

### Header Tax Per Segment

Each TCP segment carries $H = 40$ bytes of headers (20 IP + 20 TCP). Efficiency:

$$\epsilon = \frac{m}{m + H}$$

| MSS $m$ | Efficiency $\epsilon$ | Relative to 1460 |
|:---:|:---:|:---:|
| 536 (minimum) | 93.1% | 0.95x |
| 1000 | 96.2% | 0.98x |
| 1220 (IPsec tunnel) | 96.8% | 0.99x |
| 1460 (standard) | 97.3% | 1.00x |
| 8960 (jumbo) | 99.6% | 1.02x |

The efficiency gain from jumbo frames (1460 to 8960) is only 2.3%. The real benefit of jumbo frames is reducing the packet rate (fewer interrupts, less CPU overhead per byte), not the header ratio.

### Packets Per Second

For throughput $T$ bps:

$$PPS = \frac{T}{(m + H) \times 8}$$

| Throughput | MSS 1460 | MSS 8960 | PPS Reduction |
|:---:|:---:|:---:|:---:|
| 1 Gbps | 83,333 | 13,736 | 6.1x fewer |
| 10 Gbps | 833,333 | 137,362 | 6.1x fewer |
| 100 Gbps | 8,333,333 | 1,373,626 | 6.1x fewer |

At 100 Gbps with standard MTU, the host must process 8.3M packets/second. With jumbo frames, only 1.4M — a 6x reduction in CPU interrupt load.

---

## 6. IPv4 vs IPv6 Fragmentation (Behavioral Differences)

### IPv4 Fragmentation

Any router along the path can fragment if DF=0:

$$\text{Fragment at: source or any router}$$
$$\text{Reassemble at: destination only}$$

### IPv6 Fragmentation

Only the source can fragment. Routers drop oversized packets and send ICMPv6 Packet Too Big:

$$\text{Fragment at: source only}$$
$$\text{Reassemble at: destination only}$$

### Minimum MTU Guarantee

IPv4 requires all links to support at least 68 bytes. IPv6 requires 1280 bytes:

$$M_{min}^{v4} = 68, \quad M_{min}^{v6} = 1280$$

This means IPv6 PMTUD starts from a much higher floor, reducing the search space:

$$\frac{M_{max} - M_{min}^{v4}}{M_{max} - M_{min}^{v6}} = \frac{9000 - 68}{9000 - 1280} = \frac{8932}{7720} = 1.16$$

The search space for IPv6 is only 14% smaller, but the minimum guaranteed performance (1280 bytes vs 68) is dramatically better — a 1280-byte packet carries 18.8x more payload than a 68-byte packet.

---

## 7. PMTU Cache Dynamics (Expiration and Re-probing)

### Cache Hit Rate

PMTU cache entries expire after $T_{exp}$ seconds (default 600s). If a flow communicates at interval $\tau$:

$$P(\text{cache hit}) = \begin{cases} 1 & \text{if } \tau < T_{exp} \\ 0 & \text{if } \tau \geq T_{exp} \end{cases}$$

After a cache miss, the host re-probes at full MTU, potentially triggering fragmentation or ICMP again.

### Optimal Expiration Timer

Too short: frequent re-probing wastes bandwidth and triggers unnecessary ICMP.

Too long: stale PMTU entries prevent using a higher MTU after a path change.

The probability that the path MTU has changed during interval $T_{exp}$, assuming route changes follow a Poisson process with rate $\lambda$:

$$P(\text{path changed}) = 1 - e^{-\lambda T_{exp}}$$

| Route Change Rate $\lambda$ | $T_{exp}$ = 600s | $P(\text{stale})$ |
|:---:|:---:|:---:|
| 1/hour | 600s | 15.4% |
| 1/day | 600s | 0.7% |
| 1/week | 600s | 0.1% |

For stable networks ($\lambda$ small), the default 600s expiration is conservative. For dynamic networks (cloud, SD-WAN), shorter expiration prevents using suboptimal PMTU for too long.

---

*MTU mathematics expose a protocol mechanism whose failure modes are disproportionately expensive: a single ICMP-blocking firewall creates a black hole that no amount of retransmission can fix, fragmentation consumes O(N) reassembly memory that attackers can exhaust trivially, and tunnel overhead compounds multiplicatively through nested encapsulations. The solution is deceptively simple — set jumbo frames on the underlay, clamp MSS at tunnels, and enable PLPMTUD — but knowing why requires understanding the underlying mathematics of efficiency, convergence, and failure.*

## Prerequisites

- Ceiling/floor arithmetic for fragmentation calculations
- Probability (Poisson processes for cache staleness, geometric distribution for ICMP loss)
- Logarithmic analysis (binary search convergence for PLPMTUD)

## Complexity

- **Beginner:** Fragment count calculation, tunnel overhead addition, MSS derivation from MTU
- **Intermediate:** PMTUD convergence under ICMP loss, PLPMTUD binary search efficiency, reassembly memory bounds
- **Advanced:** Nested tunnel overhead stacking, PMTU cache staleness modeling, PPS reduction analysis for jumbo frames at scale
