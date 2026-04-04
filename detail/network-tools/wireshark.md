# The Mathematics of Wireshark -- Packet Capture Theory and Traffic Analysis

> *Network analysis is applied statistics: every packet is a sample, every flow is a time series, and every anomaly is a deviation from the expected distribution.*

---

## 1. Capture Sizing (Storage and Bandwidth Estimation)

### The Problem

Before capturing, you must estimate disk requirements. A busy 1 Gbps link can generate over 100 MB/s of raw packet data. Capture filters reduce volume but may exclude relevant traffic. Ring buffers bound disk usage but lose old data. The administrator must balance completeness against storage.

### The Formula

Capture file size for duration $T$ seconds on a link with average utilization $U$ (fraction) at line rate $R$ (bits/second):

$$S = T \cdot U \cdot R \cdot \frac{1}{8} \cdot (1 + f_{overhead})$$

where $f_{overhead}$ accounts for pcapng framing (~0.02 or 2%).

With a capture filter that passes fraction $p$ of packets:

$$S_{filtered} = S \cdot p$$

Ring buffer sizing: for $k$ files of size $s_{max}$ each:

$$T_{retained} = \frac{k \cdot s_{max}}{U \cdot R / 8}$$

### Worked Examples

**Example 1: Sizing a 24-hour capture on a 1 Gbps link at 30% utilization.**

$$S = 86{,}400 \times 0.30 \times 10^9 \times \frac{1}{8} \times 1.02 = 3.31 \text{ TB}$$

With a BPF filter for port 25 only (estimated 0.5% of traffic):

$$S_{filtered} = 3.31 \times 0.005 = 16.5 \text{ GB}$$

Ring buffer: 10 files of 2 GB each, retaining the most recent:

$$T_{retained} = \frac{10 \times 2 \times 10^9}{0.005 \times 0.30 \times 10^9 / 8} = \frac{20 \times 10^9}{187{,}500} \approx 29.6 \text{ hours}$$

## 2. Sampling and Confidence (Statistical Packet Analysis)

### The Problem

When analyzing captured traffic for metrics (error rates, response times, protocol distribution), the capture may represent only a sample of all traffic. How many packets must be analyzed to reach a given confidence level for a proportion estimate?

### The Formula

For estimating a proportion $p$ (e.g., fraction of retransmissions) with confidence level $z_\alpha$ and margin of error $E$:

$$n = \frac{z_\alpha^2 \cdot p(1-p)}{E^2}$$

For $z_{0.95} = 1.96$ (95% confidence) and worst-case $p = 0.5$:

$$n_{max} = \frac{1.96^2 \times 0.25}{E^2} = \frac{0.9604}{E^2}$$

For time-based metrics (response time), the required sample size for a mean estimate with population standard deviation $\sigma$:

$$n = \left(\frac{z_\alpha \cdot \sigma}{E}\right)^2$$

### Worked Examples

**Example: Estimating TCP retransmission rate.**

Suppose the true retransmission rate is approximately 2%. For 95% confidence with $\pm 0.5\%$ margin:

$$n = \frac{1.96^2 \times 0.02 \times 0.98}{0.005^2} = \frac{0.0753}{0.000025} = 3{,}012 \text{ packets}$$

A 3,012-packet sample suffices to estimate a 2% retransmission rate within $\pm 0.5\%$ at 95% confidence.

**Example: DNS response time estimation.**

Mean DNS response time ~50ms, standard deviation ~30ms. For 95% confidence within $\pm 2$ms:

$$n = \left(\frac{1.96 \times 30}{2}\right)^2 = (29.4)^2 = 864 \text{ queries}$$

## 3. TCP Retransmission Detection (Sequence Number Analysis)

### The Problem

Wireshark detects retransmissions by tracking TCP sequence numbers. A retransmission occurs when a segment carries sequence numbers that have already been sent. Wireshark distinguishes retransmissions from out-of-order segments by checking whether the segment arrives after a later sequence number has been acknowledged.

### The Formula

Let $S_n$ be the sequence number of segment $n$, $L_n$ its payload length, and $A_n$ the highest ACK received before segment $n$ arrives.

Retransmission detection:

$$\text{retransmit}(n) = \begin{cases} \text{true} & \text{if } S_n + L_n \leq S_{max} \text{ and } S_n < A_{latest} \\ \text{spurious} & \text{if } S_n < A_{latest} \text{ and ACK for } S_n \text{ already received} \\ \text{out-of-order} & \text{if } S_n < S_{max} \text{ and } S_n \geq A_{latest} \end{cases}$$

where $S_{max} = \max_{k < n}(S_k + L_k)$ is the highest sequence number sent so far.

Retransmission rate:

$$r = \frac{\text{retransmitted segments}}{\text{total data segments}}$$

### Worked Examples

**Example: Analyzing a TCP stream with retransmissions.**

| Packet | Seq   | Len  | ACK   | $S_{max}$ | Classification      |
|--------|-------|------|-------|-----------|----------------------|
| 1      | 1000  | 1460 | -     | 2460      | New data             |
| 2      | 2460  | 1460 | -     | 3920      | New data             |
| 3      | 1000  | 1460 | 1000  | 3920      | Retransmission       |
| 4      | 3920  | 1460 | 2460  | 5380      | New data             |
| 5      | 2460  | 1460 | 2460  | 5380      | Spurious retransmit  |

Packet 3: $S_3 = 1000 < S_{max} = 3920$ and $S_3 < A_{latest} = 1000$, so retransmission.
Packet 5: $S_5 = 2460$, ACK 2460 already received (packet 4 implicitly ACKs it), so spurious.

## 4. Throughput Measurement (Sliding Window Analysis)

### The Problem

Wireshark's I/O Statistics graph plots throughput over time using fixed-width time buckets. The choice of bucket width (interval) affects the granularity and smoothness of the graph. Too narrow shows noise; too wide hides bursts.

### The Formula

Throughput in bucket $[t, t + \Delta t)$:

$$\text{Tput}(t) = \frac{\sum_{i : t \leq t_i < t + \Delta t} L_i}{\Delta t}$$

where $L_i$ is the frame length of packet $i$ and $t_i$ is its timestamp.

The Nyquist-Shannon theorem suggests the sampling interval should be at most half the period of the fastest variation you want to observe:

$$\Delta t \leq \frac{T_{burst}}{2}$$

For exponentially weighted moving average (EWMA) smoothing:

$$\overline{\text{Tput}}(t) = \alpha \cdot \text{Tput}(t) + (1 - \alpha) \cdot \overline{\text{Tput}}(t - \Delta t)$$

where $\alpha = \frac{2}{N+1}$ and $N$ is the number of intervals in the averaging window.

### Worked Examples

**Example: Choosing I/O graph interval for a 100 Mbps link.**

To detect 100ms microbursts: $\Delta t \leq 50$ms. At 100 Mbps, a 50ms bucket contains at most:

$$L_{max} = \frac{100 \times 10^6 \times 0.05}{8} = 625{,}000 \text{ bytes} = 625 \text{ KB}$$

For a 10-minute capture with 50ms buckets: $\frac{600}{0.05} = 12{,}000$ data points -- manageable for plotting.

With 1-second buckets on a 30% utilized link: expected $\frac{0.30 \times 100 \times 10^6}{8} = 3.75$ MB per bucket, roughly 2,500 packets. Smooth, but 100ms bursts are invisible.

## 5. Flow Entropy (Anomaly Detection)

### The Problem

Anomaly detection in network traffic uses information entropy. A diverse set of destination IPs (port scan, DDoS) or a concentrated set (normal browsing) produces different entropy values. Wireshark conversations and endpoint statistics provide the raw data; entropy calculations flag anomalies.

### The Formula

Shannon entropy of the destination IP distribution:

$$H = -\sum_{i=1}^{n} p_i \log_2 p_i$$

where $p_i = \frac{c_i}{\sum c_j}$ is the fraction of packets sent to IP $i$, and $n$ is the number of unique destination IPs.

Maximum entropy (uniform distribution over $n$ destinations):

$$H_{max} = \log_2 n$$

Normalized entropy:

$$\hat{H} = \frac{H}{H_{max}} = \frac{H}{\log_2 n}$$

$\hat{H} \to 1$ indicates uniform distribution (scan/DDoS). $\hat{H} \to 0$ indicates concentrated traffic (normal).

### Worked Examples

**Example: Detecting a port scan.**

Normal traffic from a workstation: 90% to 3 servers, 10% to 20 others.

$$H_{normal} = -(3 \times 0.30 \times \log_2 0.30 + 20 \times 0.005 \times \log_2 0.005)$$
$$H_{normal} = -(3 \times (-0.521) + 20 \times (-0.038)) = 1.563 + 0.760 = 2.32$$
$$\hat{H}_{normal} = \frac{2.32}{\log_2 23} = \frac{2.32}{4.52} = 0.51$$

During a port scan hitting 500 IPs uniformly:

$$H_{scan} = \log_2 500 = 8.97$$
$$\hat{H}_{scan} = \frac{8.97}{\log_2 500} = 1.0$$

A jump in $\hat{H}$ from 0.51 to 1.0 is a strong anomaly signal.

**Example 2: DDoS detection via source IP entropy.**

Normal inbound traffic: 500 unique source IPs, top 10 account for 80% of packets. Under a DDoS with 50,000 spoofed sources distributed uniformly:

$$H_{ddos} = \log_2 50{,}000 = 15.6 \text{ bits}$$

$$H_{normal} \approx 5.2 \text{ bits (skewed distribution)}$$

The entropy tripling from 5.2 to 15.6 bits in a single measurement window is a reliable DDoS indicator. Automated detection thresholds can be set at $\hat{H} > 0.85$ to trigger alerts.

## 6. BPF Filter Efficiency (Instruction Cost Model)

### The Problem

BPF capture filters compile to bytecode executed in the kernel for every packet. Complex filters consume CPU cycles per packet, and on high-speed links this overhead matters. Understanding the instruction cost model helps write efficient filters.

### The Formula

BPF execution cost per packet:

$$C_{filter} = \sum_{i=1}^{N} c_i$$

where $N$ is the number of BPF instructions and $c_i$ is the cost per instruction (typically 1-3 ns on modern CPUs). For a link at rate $R$ with average packet size $s$:

$$\text{Packets/sec} = \frac{R}{8 \cdot s}$$

$$\text{CPU}_\% = \frac{R \cdot C_{filter}}{8 \cdot s \cdot T_{core}}$$

where $T_{core}$ is the cycles available per second on one core.

### Worked Examples

**Example: Filter overhead on a 10 Gbps link.**

Average packet size: 800 bytes. BPF filter: 12 instructions at 2 ns each.

$$\text{Packets/sec} = \frac{10 \times 10^9}{8 \times 800} = 1{,}562{,}500$$

$$\text{CPU} = 1{,}562{,}500 \times 24 \text{ ns} = 37.5 \text{ ms/s} = 3.75\% \text{ of one core}$$

A 12-instruction BPF filter on a 10 Gbps link consumes under 4% of one CPU core -- acceptable. But a 100-instruction filter (complex OR chains) would consume 31%, potentially causing packet drops.

## Prerequisites

- Probability and statistics (confidence intervals, proportion estimation)
- Information theory (Shannon entropy, Kullback-Leibler divergence)
- TCP protocol mechanics (sequence numbers, acknowledgments, windowing)
- Signal processing (Nyquist theorem, moving averages)
- Storage estimation (bitrate, utilization, overhead calculations)
- BPF bytecode compilation and kernel packet filtering
- Computational complexity (per-packet overhead analysis)
