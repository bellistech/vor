# The Theory of Networking -- Protocol Stacks, Information Theory, and Queuing

> *Networking is applied physics constrained by information theory. The math covers protocol stack design, the end-to-end principle, Shannon's channel capacity, and queuing theory for network buffers.*

---

## 1. Protocol Stack Theory

### Layered Abstraction

A protocol stack is a hierarchy of encapsulation. Each layer $L_i$ provides
services to $L_{i+1}$ and consumes services from $L_{i-1}$. The total frame
size at layer $k$ is:

$$F_k = H_k + F_{k+1}$$

where $H_k$ is the header added by layer $k$ and $F_{k+1}$ is the payload
(which is the complete frame from the layer above).

For the TCP/IP stack, expanding recursively:

$$F_{link} = H_{eth} + H_{ip} + H_{tcp} + \text{Payload}$$
$$F_{link} = 14 + 20 + 20 + \text{Payload} = 54 + \text{Payload}$$

### Protocol Overhead Ratio

The **overhead ratio** measures how much of each frame is headers vs. data:

$$\eta = \frac{\text{Payload}}{F_{link}} = \frac{\text{Payload}}{\sum_{k} H_k + \text{Payload}}$$

For a standard 1500-byte MTU Ethernet frame carrying TCP:

$$\eta = \frac{1500 - 54}{1500} = \frac{1446}{1500} \approx 96.4\%$$

For small payloads (e.g., a TCP ACK with 0 bytes of data):

$$\eta = \frac{0}{54} = 0\%$$

This is why Nagle's algorithm exists -- it coalesces small writes to avoid
sending headers with negligible payload.

### Encapsulation Taxonomy

| Layer | Header | Typical Size (bytes) | Addresses |
|:---:|:---|:---:|:---|
| 2 - Ethernet | Destination MAC, Source MAC, EtherType | 14 (+4 FCS) | 48-bit MAC |
| 3 - IPv4 | Version, IHL, TTL, Protocol, Src/Dst IP | 20-60 | 32-bit IP |
| 3 - IPv6 | Version, Flow Label, Next Header, Src/Dst IP | 40 | 128-bit IP |
| 4 - TCP | Src/Dst Port, Seq, Ack, Flags, Window | 20-60 | 16-bit Port |
| 4 - UDP | Src/Dst Port, Length, Checksum | 8 | 16-bit Port |

### Multiplexing and Demultiplexing

Each layer uses a **type field** to identify the next layer's protocol:

```
Ethernet.EtherType = 0x0800  -->  IPv4
IPv4.Protocol      = 6       -->  TCP
TCP.DstPort        = 443     -->  HTTPS application
```

The **5-tuple** (src IP, dst IP, src port, dst port, protocol) uniquely
identifies a connection. The total connection space is:

$$|C| = 2^{32} \times 2^{32} \times 2^{16} \times 2^{16} \times 256 \approx 2^{104}$$

In practice, one server with one IP listening on one port can handle at most
$2^{32} \times 2^{16} \approx 2^{48}$ concurrent connections from distinct
(src IP, src port) pairs.

---

## 2. The End-to-End Principle

### Statement

The end-to-end argument (Saltzer, Reed, Clark, 1984) states:

> Functions placed at low levels of a system may be redundant or of little
> value when compared with the cost of providing them at that low level. The
> function can be completely and correctly implemented only with the knowledge
> and help of the application standing at the endpoints.

### Formal Consequence

Let $P_{hop}$ be the probability that a single hop correctly delivers a packet.
For a path of $n$ hops:

$$P_{e2e} = P_{hop}^n$$

Even if each hop has 99.9% reliability ($P_{hop} = 0.999$), over 20 hops:

$$P_{e2e} = 0.999^{20} \approx 0.980$$

This means 2% of packets will be corrupted or lost. Therefore the endpoints
(TCP) must implement end-to-end reliability regardless of hop-level guarantees.

### Design Implications

1. **IP is best-effort by design.** Reliability belongs at the transport layer (TCP).
2. **Checksums are end-to-end.** TCP verifies data integrity at the receiver, not at every hop.
3. **Encryption is end-to-end.** TLS operates between application endpoints, not per-link.
4. **Hop-level optimizations are optional enhancements**, not substitutes for end-to-end correctness.

The principle does not say "never optimize at lower layers." It says lower-layer
optimizations cannot replace end-to-end verification.

---

## 3. Shannon's Channel Capacity

### The Theorem

Claude Shannon's noisy-channel coding theorem (1948) defines the theoretical
maximum data rate of a communication channel:

$$C = B \log_2\left(1 + \frac{S}{N}\right)$$

where:
- $C$ = channel capacity (bits/second)
- $B$ = bandwidth (Hz)
- $S/N$ = signal-to-noise ratio (linear, not dB)

### SNR Conversion

Signal-to-noise ratio in decibels:

$$\text{SNR}_{dB} = 10 \log_{10}\left(\frac{S}{N}\right)$$

$$\frac{S}{N} = 10^{\text{SNR}_{dB}/10}$$

### Worked Examples

**WiFi 802.11ac channel (80 MHz, 30 dB SNR):**

$$\frac{S}{N} = 10^{30/10} = 1000$$

$$C = 80 \times 10^6 \times \log_2(1001) \approx 80 \times 10^6 \times 9.97 \approx 798 \text{ Mbps}$$

This is the theoretical maximum. Real throughput is lower due to protocol
overhead, contention, and non-ideal coding.

**Ethernet over Cat5e (100 MHz, 25 dB SNR):**

$$\frac{S}{N} = 10^{25/10} \approx 316$$

$$C = 100 \times 10^6 \times \log_2(317) \approx 100 \times 10^6 \times 8.31 \approx 831 \text{ Mbps}$$

Gigabit Ethernet achieves this using 4 pairs with PAM-5 encoding (250 MHz
aggregate symbol rate).

**Fiber optic (C-band, 4.4 THz, 20 dB SNR):**

$$C = 4.4 \times 10^{12} \times \log_2(101) \approx 4.4 \times 10^{12} \times 6.66 \approx 29.3 \text{ Tbps}$$

Modern DWDM systems approach this limit using coherent detection and
probabilistic constellation shaping.

### Shannon Limit and Modulation

The Shannon limit sets the minimum SNR required per bit:

$$\frac{E_b}{N_0} \geq \frac{2^{C/B} - 1}{C/B}$$

As $C/B \to 0$, the limit approaches $\ln(2) \approx -1.59$ dB. No modulation
scheme can reliably communicate below this SNR.

### Practical Significance

| Technology | Spectral Efficiency | Shannon Limit | Gap |
|:---|:---:|:---:|:---:|
| WiFi 6 (1024-QAM) | ~9.6 bps/Hz | ~10.0 bps/Hz | ~0.4 dB |
| 5G NR (256-QAM) | ~7.4 bps/Hz | ~8.3 bps/Hz | ~1.0 dB |
| 400G Ethernet | ~6.2 bps/Hz | ~6.6 bps/Hz | ~0.5 dB |

Modern systems operate within 0.5-1.5 dB of the Shannon limit, thanks to
LDPC and Turbo codes (both capacity-approaching codes).

---

## 4. Queuing Theory for Network Buffers

### Why Queuing Matters

Every router, switch, and NIC has finite buffer space. When packets arrive
faster than they can be forwarded, they queue. The math of queuing determines
latency, jitter, and packet loss.

### The M/M/1 Queue

The simplest model: Poisson arrivals at rate $\lambda$, exponential service
times at rate $\mu$, one server (one output port).

**Utilization:**

$$\rho = \frac{\lambda}{\mu}$$

The system is stable only when $\rho < 1$ (arrival rate < service rate).

**Average number of packets in the system:**

$$L = \frac{\rho}{1 - \rho}$$

**Average queuing delay (time in system):**

$$W = \frac{1}{\mu - \lambda} = \frac{1}{\mu(1 - \rho)}$$

**Average time waiting in the queue (excluding service):**

$$W_q = \frac{\rho}{\mu(1 - \rho)}$$

### The Nonlinear Explosion

The key insight is that delay grows nonlinearly as utilization approaches 1:

| $\rho$ (Utilization) | $L$ (Packets in System) | $W$ (Delay, relative) |
|:---:|:---:|:---:|
| 0.1 | 0.11 | 1.1x |
| 0.5 | 1.0 | 2.0x |
| 0.7 | 2.33 | 3.3x |
| 0.9 | 9.0 | 10.0x |
| 0.95 | 19.0 | 20.0x |
| 0.99 | 99.0 | 100.0x |

This is why network engineers target 70-80% link utilization, not 95%. The
last 20% of capacity buys a 5-10x increase in latency.

### Bufferbloat

When router buffers are too large, packets queue instead of being dropped.
TCP interprets delay (not loss) as available capacity, so it keeps sending.
The result: seconds of latency with no throughput improvement.

**The bufferbloat delay:**

$$D_{buffer} = \frac{B_{size}}{C_{link}}$$

where $B_{size}$ is the buffer size in bits and $C_{link}$ is the link rate.

A 1 MB buffer on a 10 Mbps link:

$$D_{buffer} = \frac{8 \times 10^6}{10 \times 10^6} = 0.8 \text{ seconds}$$

Solutions: **CoDel** (Controlled Delay) and **fq_codel** (Fair Queuing CoDel)
actively manage queue depth by dropping packets when sojourn time exceeds a
target (typically 5ms).

### M/M/1/K -- Finite Buffer

Real buffers have finite size $K$. The M/M/1/K model adds packet loss:

**Loss probability (when buffer is full):**

$$P_{loss} = P(N = K) = \frac{(1 - \rho) \rho^K}{1 - \rho^{K+1}}$$

For $\rho = 0.9$ and $K = 100$ (100-packet buffer):

$$P_{loss} \approx \rho^{K} \approx 0.9^{100} \approx 2.66 \times 10^{-5}$$

For $\rho = 0.99$ and $K = 100$:

$$P_{loss} \approx 0.99^{100} \approx 0.366$$

At 99% utilization with a 100-packet buffer, you lose one-third of your
packets.

### Little's Law

A universal result that applies to any stable queuing system:

$$L = \lambda W$$

Average number in system = arrival rate x average time in system.

This is surprisingly powerful for back-of-envelope calculations:

- If 1000 packets/sec arrive and each spends 10ms in the router: $L = 1000 \times 0.01 = 10$ packets in the buffer on average.
- If you see 50 packets buffered at a rate of 500 pps: $W = 50/500 = 100\text{ms}$ average latency.

---

## 5. Bandwidth-Delay Product

### Definition

The **bandwidth-delay product** (BDP) is the amount of data "in flight" on a
network path:

$$BDP = B \times RTT$$

where $B$ is the bottleneck bandwidth and $RTT$ is the round-trip time.

### Significance for TCP

TCP's congestion window must be at least as large as the BDP to fully utilize
the link:

$$cwnd_{optimal} \geq BDP$$

**Example -- Transcontinental link (1 Gbps, 60ms RTT):**

$$BDP = 10^9 \times 0.060 = 60 \times 10^6 \text{ bits} = 7.5 \text{ MB}$$

TCP needs a 7.5 MB window to fill this pipe. The default Linux receive buffer
(usually 128 KB - 4 MB) may be too small. This is why `net.core.rmem_max` and
`net.ipv4.tcp_rmem` tuning matters for high-BDP paths.

**Example -- Data center (100 Gbps, 0.1ms RTT):**

$$BDP = 10^{11} \times 10^{-4} = 10^7 \text{ bits} = 1.25 \text{ MB}$$

Lower RTT in data centers means smaller windows suffice, but the sheer bandwidth
demands high packet processing rates (~8.1 million packets/sec at 1500-byte MTU).

### Long Fat Networks (LFNs)

Networks with large BDP are called LFNs. They require:

1. **Window scaling** (RFC 7323) -- TCP window field is 16 bits (64 KB max). Window scaling extends it to 1 GB.
2. **Selective acknowledgment (SACK)** -- retransmit only lost segments, not the entire window.
3. **Timestamps** -- disambiguate reordered segments in large windows.

---

## 6. Packet Loss, Retransmission, and Goodput

### TCP Throughput Model (Mathis Equation)

For long-lived TCP flows with random loss:

$$\text{Throughput} \leq \frac{MSS}{RTT \times \sqrt{p}}$$

where $p$ is the packet loss probability and $MSS$ is the maximum segment size.

**Example (MSS = 1460, RTT = 50ms, loss = 0.1%):**

$$\text{Throughput} \leq \frac{1460 \times 8}{0.050 \times \sqrt{0.001}} \approx \frac{11680}{0.00158} \approx 7.4 \text{ Mbps}$$

Even 0.1% loss on a 1 Gbps link limits TCP throughput to ~7.4 Mbps. This
explains why loss-based congestion control struggles on high-BDP paths and why
BBR (which models bandwidth and RTT directly) performs better.

### Goodput vs. Throughput

**Throughput** includes retransmissions and headers. **Goodput** is the useful
data rate received by the application:

$$\text{Goodput} = \text{Throughput} \times (1 - p) \times \eta$$

where $p$ is loss rate and $\eta$ is the protocol efficiency (payload/frame
ratio).

---

## 7. Network Calculus -- Deterministic Bounds

For real-time and safety-critical networks (TSN, AFDX), stochastic models are
insufficient. **Network calculus** provides worst-case deterministic bounds.

### Arrival and Service Curves

An arrival curve $\alpha(t)$ upper-bounds the cumulative traffic entering a node
over any interval of length $t$:

$$A(s, s+t) \leq \alpha(t) \quad \forall s$$

A service curve $\beta(t)$ lower-bounds the work done by a server:

$$D(t) \geq A \otimes \beta(t)$$

where $\otimes$ is the min-plus convolution.

### Delay and Backlog Bounds

**Maximum delay:**

$$d_{max} = \inf \{ \tau \geq 0 : \alpha(t) \leq \beta(t + \tau) \; \forall t \}$$

**Maximum backlog:**

$$q_{max} = \sup_{t \geq 0} \{ \alpha(t) - \beta(t) \}$$

These bounds are tight and hold for any traffic pattern within the arrival
curve, making them suitable for networks where probabilistic guarantees are
not sufficient.

---

## References

- Shannon, C.E. "A Mathematical Theory of Communication" (1948)
- Saltzer, J.H., Reed, D.P., Clark, D.D. "End-to-End Arguments in System Design" (1984)
- Kleinrock, L. "Queueing Systems, Volume 1: Theory" (1975)
- Mathis, M. et al. "The Macroscopic Behavior of the TCP Congestion Avoidance Algorithm" (RFC 3649)
- Le Boudec, J.-Y. & Thiran, P. "Network Calculus" (2001)
- Jacobson, V. & Karels, M. "Congestion Avoidance and Control" (1988)
- RFC 793 (TCP), RFC 5681 (TCP Congestion Control), RFC 8312 (CUBIC)
- Cardwell, N. et al. "BBR: Congestion-Based Congestion Control" (ACM Queue, 2016)
- Nichols, K. & Jacobson, V. "Controlling Queue Delay" (CoDel, ACM Queue, 2012)
- Stevens, W.R. "TCP/IP Illustrated, Volume 1" (1994)
- Tanenbaum, A.S. & Wetherall, D.J. "Computer Networks" (6th ed.)
