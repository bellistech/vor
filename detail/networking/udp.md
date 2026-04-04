# The Mathematics of UDP — Datagram Sizing, Throughput, and Loss Modeling

> *UDP is the minimal transport — 8 bytes of header, no state, no guarantees. The math covers throughput efficiency at small packet sizes, loss probability under load, jitter buffering for real-time applications, and the checksum arithmetic that catches bit flips.*

---

## 1. Header Efficiency — Overhead Analysis

### UDP Header

| Field | Size | Purpose |
|:---|:---:|:---|
| Source Port | 2 bytes | Sender identification |
| Destination Port | 2 bytes | Demultiplexing |
| Length | 2 bytes | Header + payload size |
| Checksum | 2 bytes | Integrity (optional in IPv4) |
| **Total** | **8 bytes** | |

### Comparison with TCP

| Transport | Header Size | Overhead at 1472B payload | Overhead at 64B payload |
|:---|:---:|:---:|:---:|
| UDP | 8 B | 0.54% | 11.1% |
| TCP (min) | 20 B | 1.34% | 23.8% |
| TCP (timestamps) | 32 B | 2.13% | 33.3% |

### Throughput Efficiency

$$\eta = \frac{L_{payload}}{L_{payload} + H_{UDP} + H_{IP} + H_{L2}}$$

| Payload | UDP+IPv4+Eth | Total | Efficiency |
|:---:|:---:|:---:|:---:|
| 1 byte | 8+20+14 = 42 B | 43 B | 2.3% |
| 64 bytes | 42 B | 106 B | 60.4% |
| 512 bytes | 42 B | 554 B | 92.4% |
| 1,472 bytes (max in 1500 MTU) | 42 B | 1,514 B | 97.2% |

### Maximum Datagram Size

$$L_{max} = 2^{16} - 1 - H_{UDP} = 65,535 - 8 = 65,527 \text{ bytes}$$

In practice, limited by IP fragmentation concerns to $MTU - H_{IP} - H_{UDP}$.

---

## 2. Packet Loss Modeling

### Loss Under Load

When arrival rate $\lambda$ exceeds service rate $\mu$, a queue fills and drops occur. For an M/M/1/K queue (finite buffer):

$$P_{loss} = \frac{(1 - \rho) \rho^K}{1 - \rho^{K+1}}$$

Where $\rho = \lambda / \mu$ and $K$ = buffer size in packets.

### Simplified: Random Loss Model

If each packet has independent loss probability $p$:

$$P(\text{n consecutive received}) = (1 - p)^n$$

$$P(\text{at least 1 loss in n packets}) = 1 - (1 - p)^n$$

| Loss Rate $p$ | 10 packets | 100 packets | 1,000 packets |
|:---:|:---:|:---:|:---:|
| 0.01% | 0.1% | 1.0% | 9.5% |
| 0.1% | 1.0% | 9.5% | 63.2% |
| 1% | 9.6% | 63.4% | 99.996% |

### Application-Level Redundancy (FEC)

Forward Error Correction adds $R$ redundant packets per $N$ data packets:

$$P_{recovery} = 1 - \binom{N+R}{> R \text{ losses}} \times p^{R+1}$$

**Reed-Solomon FEC example (N=10, R=3):** Tolerates up to 3 lost packets out of 13.

$$\text{Overhead} = \frac{R}{N} = \frac{3}{10} = 30\%$$

---

## 3. Jitter Buffer Sizing — Real-Time Applications

### The Problem

VoIP and video require steady playout. Network jitter means packets arrive at variable intervals. A jitter buffer smooths this.

### Buffer Depth

$$B_{depth} = J_{max} + M$$

Where:
- $J_{max}$ = maximum expected jitter (ms)
- $M$ = safety margin (ms)

### Delay-Loss Tradeoff

Larger buffer = more delay but fewer underruns. Smaller buffer = less delay but more audible gaps.

$$P_{underrun} = P(jitter > B_{depth})$$

If jitter is normally distributed with mean $\mu_j$ and standard deviation $\sigma_j$:

$$P_{underrun} = 1 - \Phi\left(\frac{B_{depth} - \mu_j}{\sigma_j}\right)$$

| Buffer (ms) | $\sigma$ coverage | Underrun probability |
|:---:|:---:|:---:|
| $\mu + 1\sigma$ | 68.3% | 15.9% |
| $\mu + 2\sigma$ | 95.4% | 2.3% |
| $\mu + 3\sigma$ | 99.7% | 0.15% |
| $\mu + 4\sigma$ | 99.99% | 0.003% |

### VoIP Sizing Example

Codec: G.711 at 20 ms frames. Measured jitter: $\mu = 5$ ms, $\sigma = 10$ ms.

For 1% underrun: $B = 5 + 2.33 \times 10 = 28.3$ ms → round to 30 ms (1.5 frames).

Total one-way delay: codec (20 ms) + buffer (30 ms) + network (50 ms) = 100 ms. Under the 150 ms ITU-T G.114 recommendation.

---

## 4. Checksum — Ones' Complement Arithmetic

### The Algorithm

UDP checksum uses 16-bit ones' complement sum:

1. Create a pseudo-header (src IP, dst IP, protocol, UDP length)
2. Sum all 16-bit words using ones' complement addition
3. Take the ones' complement of the result

### Ones' Complement Addition

$$a \oplus b = (a + b) + \text{carry}$$

Carry bits wrap around and are added to the LSB.

### Error Detection Capability

The 16-bit checksum detects:
- All single-bit errors
- All double-bit errors
- Any odd number of bit errors

$$P_{undetected} \leq 2^{-16} = \frac{1}{65,536} \approx 0.0015\%$$

For random corruption, the probability of a checksum collision.

### IPv4 vs IPv6 Checksum

| Version | UDP Checksum | Required? |
|:---|:---|:---:|
| IPv4 | Ones' complement | Optional (0 = disabled) |
| IPv6 | Ones' complement | **Mandatory** (RFC 8200) |

IPv6 removed the IP header checksum, so UDP's checksum is the only integrity check.

---

## 5. UDP Throughput Limits — Kernel and NIC

### Packets per Second (Small Packets)

The practical PPS limit for UDP is constrained by:

$$PPS_{max} = \min\left(\frac{BW}{(L + H) \times 8}, PPS_{kernel}, PPS_{NIC}\right)$$

| Component | Typical PPS Limit |
|:---|:---:|
| 10G NIC (64B frames) | 14.88 Mpps |
| Linux kernel (per core) | ~1-2 Mpps |
| Linux kernel (with XDP) | ~10-24 Mpps |
| DPDK (per core) | ~30-60 Mpps |

### Line-Rate PPS by Frame Size

$$PPS_{line} = \frac{BW}{(L_{frame} + IFG + Preamble) \times 8} = \frac{BW}{(L + 20) \times 8}$$

Where IFG (12 bytes) + Preamble (8 bytes) = 20 bytes of Ethernet overhead.

| Frame Size | 1G Line Rate | 10G Line Rate | 100G Line Rate |
|:---:|:---:|:---:|:---:|
| 64 B | 1.49 Mpps | 14.88 Mpps | 148.8 Mpps |
| 128 B | 0.84 Mpps | 8.45 Mpps | 84.5 Mpps |
| 512 B | 0.23 Mpps | 2.35 Mpps | 23.5 Mpps |
| 1,518 B | 0.08 Mpps | 0.81 Mpps | 8.1 Mpps |

---

## 6. Multicast and Broadcast — UDP's Strength

### Multicast Efficiency

Sending to $N$ receivers:

| Method | Packets Sent | Bandwidth Used |
|:---|:---:|:---|
| Unicast UDP | $N$ copies | $N \times S_{packet}$ |
| Multicast UDP | 1 copy (replicated by network) | $1 \times S_{packet}$ (on each link) |
| Broadcast UDP | 1 copy | $1 \times S_{packet}$ (flooded everywhere) |

$$\text{Bandwidth savings} = \frac{N - 1}{N} = 1 - \frac{1}{N}$$

For 1,000 receivers: 99.9% bandwidth reduction on the sender's link.

---

## 7. Summary of Formulas

| Formula | Math Type | Application |
|:---|:---|:---|
| $L / (L + H)$ | Ratio | Payload efficiency |
| $(1-p)^n$ | Geometric probability | Consecutive success |
| $1 - \Phi((B - \mu)/\sigma)$ | Normal distribution | Jitter buffer underrun |
| $2^{-16}$ | Error detection bound | Checksum collision |
| $BW / ((L + 20) \times 8)$ | Division | Line-rate PPS |
| $R/N$ | Fraction | FEC overhead |
| $(N-1)/N$ | Ratio | Multicast savings |

## Prerequisites

- checksum arithmetic, binary operations, basic probability

---

*UDP's simplicity is its math — 8 bytes of header and zero state. It doesn't guarantee delivery, ordering, or congestion control, which makes it the fastest transport on the wire and the foundation for DNS, VoIP, gaming, QUIC, and every protocol that prefers speed over safety.*
