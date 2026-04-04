# The Mathematics of iperf -- Bandwidth Measurement and TCP Dynamics

> *Measuring network performance is measuring a stochastic process: TCP throughput is bounded by physics, shaped by congestion control, and limited by the weakest link in a chain of buffers.*

---

## 1. Bandwidth-Delay Product (The Fundamental Limit)

### The Problem

TCP throughput on a high-latency link is bounded not by the link speed but by the product of bandwidth and round-trip time. The sender can only have a finite number of unacknowledged bytes "in flight." If the TCP window is smaller than the bandwidth-delay product, the link cannot be fully utilized regardless of its capacity.

### The Formula

The Bandwidth-Delay Product (BDP):

$$\text{BDP} = B \cdot \text{RTT}$$

where $B$ is the bottleneck bandwidth (bytes/second) and RTT is the round-trip time (seconds).

Maximum achievable throughput with window size $W$:

$$\text{Tput}_{max} = \frac{W}{\text{RTT}}$$

Link utilization:

$$U = \frac{W}{\text{BDP}} = \frac{W}{B \cdot \text{RTT}} \quad (\text{capped at } 1.0)$$

For full utilization: $W \geq \text{BDP}$.

### Worked Examples

**Example 1: Transatlantic 10 Gbps link.**

RTT from New York to London: 70ms. Bottleneck bandwidth: 10 Gbps.

$$\text{BDP} = \frac{10 \times 10^9}{8} \times 0.070 = 87{,}500{,}000 \text{ bytes} = 83.4 \text{ MB}$$

With default Linux TCP buffer (net.ipv4.tcp_rmem max = 6 MB):

$$U = \frac{6 \times 10^6}{87.5 \times 10^6} = 6.86\%$$

Actual throughput: $10 \times 0.0686 = 686$ Mbps. Tuning to $W = 88$ MB yields full 10 Gbps.

**Example 2: Parallel streams as a workaround.**

With $P$ parallel streams, each with window $W$:

$$\text{Tput}_{total} = P \cdot \frac{W}{\text{RTT}}$$

For the example above with $W = 6$ MB and $P = 14$ streams:

$$\text{Tput} = 14 \times \frac{6 \times 10^6}{0.070} = 14 \times 85.7 \text{ Mbps} = 1.2 \text{ Gbps}$$

Still not full utilization, but significantly better than a single stream.

## 2. TCP Throughput with Loss (Mathis Formula)

### The Problem

Packet loss forces TCP to reduce its congestion window. Under steady-state conditions with random loss, TCP CUBIC and Reno exhibit predictable throughput as a function of loss rate. This is critical when interpreting iperf3 results that show retransmissions.

### The Formula

The Mathis formula for TCP Reno throughput:

$$\text{Tput} = \frac{\text{MSS}}{RTT} \cdot \frac{C}{\sqrt{p}}$$

where MSS is the Maximum Segment Size (typically 1460 bytes), $p$ is the packet loss probability, and $C$ is a constant ($C \approx \sqrt{3/2} \approx 1.22$ for TCP Reno).

For TCP CUBIC (default on Linux):

$$\text{Tput}_{CUBIC} \approx \frac{0.8 \cdot \left(\frac{3}{2(1+32p^2)}\right)^{1/4}}{RTT \cdot p^{3/4}}$$

In practice, CUBIC is less sensitive to loss than Reno on high-BDP paths.

### Worked Examples

**Example: Impact of 0.1% loss on a 1 Gbps link.**

MSS = 1460 bytes, RTT = 10ms, $p = 0.001$.

TCP Reno:

$$\text{Tput} = \frac{1460 \times 8}{0.010} \times \frac{1.22}{\sqrt{0.001}}$$

$$= 1{,}168{,}000 \times \frac{1.22}{0.0316} = 1{,}168{,}000 \times 38.6 = 45.1 \text{ Mbps}$$

A 1 Gbps link with only 0.1% loss delivers just 45 Mbps with TCP Reno -- a 95.5% throughput reduction.

TCP CUBIC on the same link achieves approximately 200-400 Mbps due to its cubic window growth function, but still far below wire rate.

**This is why iperf3 retransmission counts matter.** Even small loss rates devastate throughput.

## 3. UDP Jitter and Loss (Statistical Measurement)

### The Problem

iperf3 UDP mode measures jitter and packet loss. Jitter is defined as the inter-packet delay variation (IPDV), computed as a running average of consecutive packet delay differences. Loss is the ratio of missing to expected datagrams.

### The Formula

One-way delay for packet $i$: $d_i = R_i - S_i$ where $R_i$ is receive time and $S_i$ is send time.

Inter-packet delay variation (jitter) per RFC 3550:

$$J_i = J_{i-1} + \frac{|d_i - d_{i-1}| - J_{i-1}}{16}$$

This is an exponentially weighted moving average with $\alpha = 1/16$.

Packet loss rate:

$$L = \frac{N_{sent} - N_{received}}{N_{sent}}$$

Confidence interval for loss rate (binomial proportion):

$$L \pm z_\alpha \sqrt{\frac{L(1-L)}{N_{sent}}}$$

### Worked Examples

**Example: Analyzing a 60-second UDP test at 100 Mbps.**

Results: 728,300 datagrams sent, 725,100 received, reported jitter 0.45ms.

$$L = \frac{728{,}300 - 725{,}100}{728{,}300} = \frac{3{,}200}{728{,}300} = 0.439\%$$

95% confidence interval:

$$0.439\% \pm 1.96 \sqrt{\frac{0.00439 \times 0.99561}{728{,}300}} = 0.439\% \pm 0.048\%$$

Loss rate: $0.39\% - 0.49\%$ with 95% confidence.

A jitter of 0.45ms is acceptable for VoIP (< 1ms threshold) but the 0.44% loss exceeds the 0.1% target for real-time media.

## 4. Parallel Streams and Fairness (Jain's Fairness Index)

### The Problem

When running multiple parallel streams (`-P`), each competes for bandwidth at shared bottlenecks. Ideally, all streams receive equal bandwidth. Jain's fairness index quantifies how equitably bandwidth is distributed. This matters when comparing single-stream versus multi-stream results.

### The Formula

Jain's fairness index for $n$ streams with throughputs $x_1, x_2, \ldots, x_n$:

$$F = \frac{\left(\sum_{i=1}^{n} x_i\right)^2}{n \cdot \sum_{i=1}^{n} x_i^2}$$

Properties:
- $F = 1$: perfectly fair (all streams equal)
- $F = 1/n$: maximally unfair (one stream gets everything)

Aggregate throughput with $n$ streams vs single stream:

$$\text{Gain} = \frac{\sum_{i=1}^{n} x_i}{x_{single}}$$

### Worked Examples

**Example: 4 parallel TCP streams on a 1 Gbps link.**

Stream throughputs: 245 Mbps, 252 Mbps, 248 Mbps, 240 Mbps.

$$\sum x_i = 985, \quad \sum x_i^2 = 245^2 + 252^2 + 248^2 + 240^2 = 242{,}573$$

$$F = \frac{985^2}{4 \times 242{,}573} = \frac{970{,}225}{970{,}292} = 0.99993$$

Nearly perfect fairness. Total throughput: 985 Mbps vs a single stream's 943 Mbps.

$$\text{Gain} = \frac{985}{943} = 1.045 \quad (4.5\% \text{ improvement})$$

The marginal gain from parallelism on a low-latency link is small. On high-latency links (where single-stream is window-limited), the gain is much larger.

## 5. Zero-Copy and CPU Efficiency (Overhead Model)

### The Problem

At 10 Gbps and above, CPU becomes the bottleneck before the network. Each byte copied between kernel and user space consumes CPU cycles. iperf3's `-Z` flag enables `sendfile()` (zero-copy), bypassing the user-space buffer and reducing CPU overhead.

### The Formula

CPU cycles per byte with standard `send()`:

$$C_{copy} = C_{syscall} + C_{memcpy} + C_{checksum} + C_{DMA}$$

With zero-copy `sendfile()`:

$$C_{zero} = C_{syscall} + C_{checksum} + C_{DMA}$$

Savings: $\Delta C = C_{memcpy}$, where memory copy cost:

$$C_{memcpy} \approx \frac{B}{BW_{mem}} \cdot f_{CPU}$$

For a 3 GHz CPU with 20 GB/s memory bandwidth, copying 10 Gbps of data:

$$C_{memcpy} = \frac{10 \times 10^9 / 8}{20 \times 10^9} \times 3 \times 10^9 = 187.5 \times 10^6 \text{ cycles/s}$$

That is approximately 6.25% of a single core at 3 GHz.

### Worked Examples

**Example: CPU overhead comparison at 25 Gbps.**

Without zero-copy (standard send):
- Memory copies: 2 (user-to-kernel, kernel-to-NIC DMA staging)
- CPU per copy: $\frac{25 \times 10^9 / 8}{20 \times 10^9} = 15.6\%$ of memory bandwidth per copy
- Total CPU: ~1.2 cores consumed by memory operations alone

With zero-copy (sendfile):
- Memory copies: 0 (kernel reads directly from page cache to NIC via DMA)
- CPU: ~0.3 cores (syscall + checksum offload)

In iperf3 testing, enabling `-Z` on a 25G link typically increases measured throughput by 15-30% because the measurement tool itself was the bottleneck.

**Key insight:** when iperf3 shows less than expected throughput on 10G+ links, try `-Z` before blaming the network.

## Prerequisites

- TCP congestion control fundamentals (slow start, congestion avoidance, fast retransmit)
- Bandwidth-delay product and TCP windowing
- Basic statistics (confidence intervals, moving averages)
- UDP protocol mechanics (datagrams, no flow control)
- Operating system internals (zero-copy I/O, DMA, socket buffers)
- Fairness metrics and game theory (Nash equilibrium in bandwidth sharing)
