# The Mathematics of Network Stack Tuning --- Buffer Theory, Congestion Models & Queuing

> *A network stack is a pipeline of queues, each governed by queuing theory, feedback control, and information theory. Tuning it without understanding the math is like tuning an engine by ear --- you might get lucky, but you will never find the optimum. This page derives the formulas behind every sysctl knob.*

---

## Prerequisites

- TCP fundamentals (three-way handshake, sliding window, sequence numbers)
- Basic probability and statistics (means, distributions)
- Familiarity with sysctl, ethtool, and Linux networking tools

## Complexity

- **Mathematical depth**: Intermediate (algebra, basic calculus, probability)
- **System scope**: Kernel network stack (L3-L4), NIC hardware, queueing disciplines
- **Kernel versions**: 4.9+ (BBR), 5.x+ (advanced BPF sockmap, AF_XDP)

---

## 1. Bandwidth-Delay Product and Buffer Sizing

### The Fundamental Constraint

Every TCP connection has a maximum amount of data that can be "in flight" (sent but not yet acknowledged). This is the **bandwidth-delay product (BDP)** --- the capacity of the pipe:

$$BDP = B \times RTT$$

Where:
- $B$ = bottleneck bandwidth (bytes/sec)
- $RTT$ = round-trip time (seconds)
- $BDP$ = bytes that fill the pipe

The TCP receive window $W$ must satisfy $W \geq BDP$ for the sender to fully utilize the link. If $W < BDP$, the sender stalls waiting for ACKs and throughput drops to:

$$\text{Throughput} = \frac{W}{RTT}$$

### Utilization Formula

Link utilization as a function of window size:

$$U = \frac{W}{BDP} = \frac{W}{B \times RTT}, \quad U \leq 1$$

When $U < 1$, the achievable throughput is:

$$T = U \times B = \frac{W}{RTT}$$

### Buffer Sizing for sysctl

The kernel parameter `tcp_rmem` / `tcp_wmem` sets min/default/max TCP buffer sizes. The **max** value must accommodate the largest BDP on any connection:

$$\text{rmem\_max} \geq BDP_{max} = B_{max} \times RTT_{max}$$

For a server handling diverse clients:

$$\text{rmem\_max} \geq \max_i(B_i \times RTT_i)$$

### Worked Examples

| Scenario | Bandwidth | RTT | BDP | Minimum Buffer |
|:---|:---|:---:|:---:|:---:|
| LAN (1G) | 125 MB/s | 0.5 ms | 62.5 KB | 64 KB |
| Metro (10G) | 1.25 GB/s | 5 ms | 6.25 MB | 8 MB |
| Cross-country (10G) | 1.25 GB/s | 60 ms | 75 MB | 80 MB |
| Transatlantic (10G) | 1.25 GB/s | 80 ms | 100 MB | 128 MB |
| 100G datacenter | 12.5 GB/s | 0.5 ms | 6.25 MB | 8 MB |
| 100G cross-country | 12.5 GB/s | 40 ms | 500 MB | 512 MB |

### Memory Cost

Total memory consumed by TCP buffers for $N$ concurrent connections:

$$M_{total} = \sum_{i=1}^{N} W_i \approx N \times W_{avg}$$

With auto-tuning (`tcp_moderate_rcvbuf=1`), most connections use the default (middle value of `tcp_rmem`). Only connections with large BDP grow to the maximum. The kernel enforces a global limit via `tcp_mem` (in pages):

$$M_{limit} = \text{tcp\_mem[2]} \times \text{PAGE\_SIZE}$$

If $M_{total}$ exceeds the pressure threshold (`tcp_mem[1]`), the kernel begins shrinking windows.

---

## 2. TCP Window Scaling and Throughput Calculations

### The Window Scale Problem

The TCP header's window field is 16 bits: max value $2^{16} - 1 = 65,535$ bytes. For any link where $BDP > 65,535$, this limits throughput:

$$T_{max} = \frac{65,535}{RTT}$$

For a 60 ms RTT path: $T_{max} = \frac{65,535}{0.060} = 1.09$ MB/s = 8.7 Mbps --- unusable for a 10G link.

### Window Scale Option (RFC 7323)

The scale factor $S$ (0--14) is negotiated during the three-way handshake:

$$W_{effective} = W_{header} \times 2^S$$

Maximum effective window:

$$W_{max} = 65,535 \times 2^{14} = 1,073,725,440 \text{ bytes} \approx 1 \text{ GB}$$

### Required Scale Factor

To achieve a target window $W_{target}$:

$$S = \lceil \log_2 \left(\frac{W_{target}}{65,535}\right) \rceil$$

For 100 MB: $S = \lceil \log_2(1,525.9) \rceil = \lceil 10.57 \rceil = 11$

For 512 MB: $S = \lceil \log_2(7,812.7) \rceil = \lceil 12.93 \rceil = 13$

### Throughput Under Loss

With random packet loss rate $p$, the Mathis formula gives:

$$T \approx \frac{MSS}{RTT \times \sqrt{p}} \times C$$

Where $C \approx 1.22$ (constant from the TCP model), $MSS$ is the maximum segment size. This shows that throughput degrades as $\frac{1}{\sqrt{p}}$ --- a 1% loss rate cuts throughput by 10x compared to lossless.

For a 1500-byte MSS, 10 ms RTT, 0.1% loss:

$$T = \frac{1460}{0.010 \times \sqrt{0.001}} \times 1.22 = \frac{1460}{0.000316} \times 1.22 \approx 5.63 \text{ MB/s} \approx 45 \text{ Mbps}$$

This is why loss-based congestion control (CUBIC, Reno) struggles on lossy links --- BBR was designed to address this.

---

## 3. BBR Congestion Control Model

### The BBR Philosophy

Traditional loss-based algorithms (Reno, CUBIC) interpret **loss as congestion**. BBR (Bottleneck Bandwidth and Round-trip propagation time) instead builds an explicit model of the network path:

$$\text{Delivery Rate} = \frac{\text{Data Delivered}}{\text{Time Elapsed}}$$

BBR estimates two key quantities:
- $\hat{B}$ = bottleneck bandwidth (max recent delivery rate)
- $\hat{R}$ = propagation RTT (min recent RTT, excluding queuing delay)

### The Optimal Operating Point

BBR targets the **Kleinrock optimal**: maximum throughput at minimum delay. This occurs when the amount of data in flight equals exactly one BDP:

$$\text{Inflight}_{optimal} = \hat{B} \times \hat{R}$$

At this point:
- The bottleneck link is fully utilized ($U = 1$)
- No excess data sits in buffers (queuing delay $= 0$)
- Throughput is maximized and latency is minimized

### BBR Pacing Rate

BBR sets a **pacing rate** rather than relying on burst-and-wait:

$$\text{PacingRate} = \text{pacing\_gain} \times \hat{B}$$

During steady state (ProbeBW phase), the pacing gain cycles through:

$$g \in \{1.25, 0.75, 1, 1, 1, 1, 1, 1\}$$

The 1.25 phase probes for increased bandwidth; the 0.75 phase drains any queue built up.

### Congestion Window

BBR also sets the congestion window:

$$\text{cwnd} = \text{cwnd\_gain} \times \hat{B} \times \hat{R}$$

In steady state, $\text{cwnd\_gain} = 2$, allowing headroom for bandwidth probing.

### BBR vs. CUBIC: When to Choose

| Metric | BBR | CUBIC |
|:---|:---|:---|
| Loss sensitivity | Low (model-based) | High (loss = congestion) |
| Buffer bloat | Reduces it | Can cause it |
| Fairness with CUBIC flows | Can dominate (BBRv1) | Fair among CUBIC flows |
| High-BDP paths | Excellent | Good with sufficient buffers |
| Intra-DC (low RTT) | Overkill | Preferred |
| Best pairing | `fq` qdisc | Any qdisc |

BBRv2 (kernel 5.18+) improves fairness and loss response.

---

## 4. Interrupt Moderation Trade-offs

### The Interrupt Cost Model

Each hardware interrupt has a fixed CPU cost $C_{irq}$ (context switch, cache pollution, function call overhead). At line rate, a NIC generates:

$$I_{max} = \frac{B}{F_{avg}}$$

Where:
- $I_{max}$ = interrupts per second at line rate
- $B$ = link bandwidth (bytes/sec)
- $F_{avg}$ = average frame size (bytes)

For 10G with 64-byte frames: $I_{max} = \frac{1.25 \times 10^9}{64} = 19.5 \times 10^6$ interrupts/sec --- impossible for any CPU.

### Interrupt Coalescing

Interrupt coalescing batches $n$ frames per interrupt, reducing the interrupt rate:

$$I_{coalesced} = \frac{I_{max}}{n} = \frac{B}{n \times F_{avg}}$$

CPU overhead from interrupts:

$$\text{CPU}_{\%} = \frac{I_{coalesced} \times C_{irq}}{C_{total}}$$

Where $C_{total}$ is the total CPU cycles/sec available.

### The Latency-Throughput Trade-off

Coalescing introduces latency. With a timer-based coalescing delay $\tau$ (usecs):

$$L_{added} = \frac{\tau}{2} \quad \text{(average additional latency)}$$

The maximum additional latency is $\tau$ (worst case: packet arrives just after a coalescing interrupt fires).

With frame-count coalescing ($n$ frames per interrupt):

$$L_{max} = \frac{n \times F_{avg}}{B}$$

### Optimal Coalescing Point

The optimal coalescing parameter minimizes total cost:

$$\text{Cost} = \alpha \times \frac{B}{n \times F_{avg}} \times C_{irq} + \beta \times \frac{n \times F_{avg}}{B}$$

Where $\alpha$ weights CPU cost and $\beta$ weights latency. Taking the derivative and setting to zero:

$$\frac{d(\text{Cost})}{dn} = -\alpha \times \frac{B \times C_{irq}}{n^2 \times F_{avg}} + \beta \times \frac{F_{avg}}{B} = 0$$

$$n_{opt} = \sqrt{\frac{\alpha \times B^2 \times C_{irq}}{\beta \times F_{avg}^2}}$$

This is why **adaptive coalescing** exists --- it dynamically adjusts $n$ based on traffic patterns, effectively solving this optimization in real time.

### NAPI Polling

Linux NAPI (New API) switches between interrupt-driven and polling modes:

1. First packet triggers an interrupt
2. Driver disables interrupts and enters polling mode
3. Kernel polls for packets in a budget-limited loop (`netdev_budget`)
4. When no more packets, re-enables interrupts

The transition threshold is implicit: NAPI polls up to `netdev_budget` packets (default 300) per softirq cycle. Increasing this helps high-throughput scenarios but increases tail latency for other processes waiting for softirq time.

---

## 5. Queuing Theory for Network Buffers

### M/M/1 Queue Model

A network buffer (NIC ring buffer, socket receive queue, qdisc) can be modeled as an M/M/1 queue:

- Arrivals: Poisson process with rate $\lambda$ (packets/sec)
- Service: Exponential with rate $\mu$ (packets/sec)
- Single server (single output link)

**Utilization:**

$$\rho = \frac{\lambda}{\mu}, \quad \rho < 1 \text{ for stability}$$

**Average number of packets in system:**

$$L = \frac{\rho}{1 - \rho}$$

**Average queuing delay:**

$$W_q = \frac{\rho}{\mu(1 - \rho)} = \frac{L}{\lambda} - \frac{1}{\mu}$$

### Buffer Sizing from Queuing Theory

At high utilization ($\rho \to 1$), queue length grows unbounded. A finite buffer of size $K$ means packets are dropped when the queue is full. The drop probability for an M/M/1/K queue:

$$P_{drop} = \frac{(1 - \rho) \rho^K}{1 - \rho^{K+1}}$$

For a target drop rate $P_{target}$, the required buffer size:

$$K \geq \frac{\ln(P_{target}) - \ln(1 - \rho)}{\ln(\rho)} \quad (\text{approximate, } \rho < 1)$$

### The Stanford Model (Rule of Thumb)

Traditional buffer sizing for backbone routers:

$$B = BDP = C \times RTT$$

Where $C$ is the link capacity. For a 10G link with 250 ms RTT: $B = 312.5$ MB.

The Appenzeller-Keslassy-McKeown (2004) result shows that with $N$ independent TCP flows:

$$B = \frac{C \times RTT}{\sqrt{N}}$$

For 10,000 flows: $B = \frac{312.5 \text{ MB}}{\sqrt{10000}} = 3.125$ MB --- a 100x reduction.

### Little's Law

A universal relationship for any stable queue:

$$L = \lambda \times W$$

Where:
- $L$ = average number of items in the system
- $\lambda$ = arrival rate
- $W$ = average time in the system

This applies to every buffer in the stack:

| Buffer | $\lambda$ | $W$ | $L$ |
|:---|:---|:---|:---|
| NIC RX ring | Packet arrival rate | Time until softirq processes | Ring occupancy |
| Socket recv buffer | Segment arrival rate | Time until `recv()` call | Buffer fill level |
| TCP retransmit queue | Segment send rate | Time until ACK | Inflight segments |
| Qdisc TX queue | Packet enqueue rate | Time until NIC transmits | Queue depth |

### Tail Latency and Buffer Bloat

Excess buffering ("buffer bloat") increases latency without improving throughput. The queuing delay under load:

$$D_{queue} = \frac{Q}{C}$$

Where $Q$ is the current queue depth (bytes) and $C$ is the drain rate (bytes/sec). For a 1000-packet queue at 1G with 1500-byte packets:

$$D_{queue} = \frac{1000 \times 1500}{125 \times 10^6} = 12 \text{ ms}$$

This is why BBR + FQ (fair queuing) is effective: BBR avoids building queues, and FQ prevents a single flow from monopolizing buffer space.

### CoDel and FQ-CoDel

CoDel (Controlled Delay) actively manages queue latency by tracking the **minimum queuing delay** over a sliding window. If the minimum delay exceeds a target (default 5 ms) for an interval (default 100 ms), CoDel drops packets at increasing rates:

$$\text{drop interval} = \frac{\text{interval}}{\sqrt{\text{count}}}$$

Where count is the number of consecutive drops. This implements a square-root drop schedule that converges to the fair rate without oscillation. FQ-CoDel combines per-flow fair queuing with CoDel on each sub-queue, achieving both fairness and low latency.

---

## 6. Connection Tracking and Hash Table Sizing

### Hash Table Theory

The connection tracking table is a hash table with $H$ buckets and $N$ entries. Average chain length:

$$\bar{l} = \frac{N}{H}$$

For efficient $O(1)$ lookup, we want $\bar{l} \leq 4$. So:

$$H \geq \frac{N}{4} = \frac{\text{nf\_conntrack\_max}}{4}$$

This is exactly why the recommendation is `hashsize = conntrack_max / 4`.

### Memory Cost

Each conntrack entry consumes approximately 320 bytes (kernel-version dependent). Total memory:

$$M_{conntrack} = N \times 320 \text{ bytes}$$

For 1M entries: $M = 1,048,576 \times 320 = 320$ MB.

Hash table overhead (8 bytes per bucket pointer):

$$M_{hash} = H \times 8 \text{ bytes}$$

For 262,144 buckets: $M_{hash} = 2$ MB --- negligible.

### Timeout Optimization

Each tracked connection stays in the table for its timeout period. At steady state with connection rate $r$ (connections/sec):

$$N_{steady} = r \times \bar{T}$$

Where $\bar{T}$ is the average timeout. Reducing `nf_conntrack_tcp_timeout_established` from 432,000s (5 days) to 86,400s (1 day) reduces steady-state table occupancy by 5x for long-lived connection workloads.

---

## 7. Socket Backlog and Accept Queue Theory

### The Two-Queue Model

Linux TCP listen sockets have two queues:

1. **SYN queue** (half-open): connections in SYN_RECV state, bounded by `tcp_max_syn_backlog`
2. **Accept queue** (fully established): connections waiting for `accept()`, bounded by `min(somaxconn, backlog_arg)`

### SYN Flood Analysis

A SYN flood at rate $\lambda_{syn}$ (SYN packets/sec) fills the SYN queue in:

$$T_{fill} = \frac{Q_{syn}}{\lambda_{syn} - \lambda_{expire}}$$

Where $\lambda_{expire}$ is the rate at which entries expire (retransmit timeout). With SYN cookies enabled, the SYN queue is bypassed entirely --- the server encodes state in the SYN-ACK sequence number and reconstructs it from the client's ACK.

### Accept Queue Sizing

If the application calls `accept()` at rate $\mu$ and connections complete the handshake at rate $\lambda$:

$$P_{full} = P(\text{queue length} \geq Q_{accept})$$

For an M/M/1/K model with $\rho = \lambda / \mu$:

$$P_{full} = \frac{\rho^{Q_{accept}}(1 - \rho)}{1 - \rho^{Q_{accept} + 1}}$$

When the accept queue is full, the kernel drops the final ACK of the handshake, and the client sees a connection timeout. Setting `somaxconn` high enough to absorb bursts is critical:

$$Q_{accept} \geq \lambda_{peak} \times T_{accept}$$

Where $T_{accept}$ is the worst-case time between `accept()` calls.

---

## References

- Jacobson, V. (1988). "Congestion Avoidance and Control." SIGCOMM
- Cardwell, N. et al. (2017). "BBR: Congestion-Based Congestion Control." ACM Queue
- Appenzeller, G., Keslassy, I., McKeown, N. (2004). "Sizing Router Buffers." SIGCOMM
- Mathis, M. et al. (1997). "The Macroscopic Behavior of the TCP Congestion Avoidance Algorithm." RFC 3649
- Nichols, K. and Jacobson, V. (2012). "Controlling Queue Delay." ACM Queue (CoDel)
- RFC 7323 -- TCP Extensions for High Performance (Window Scale, Timestamps, PAWS)
- RFC 8312 -- CUBIC for Fast Long-Distance Networks
- RFC 9002 -- Using TLS to Secure QUIC (BBR references)
- Linux kernel source: `net/ipv4/tcp_bbr.c`, `net/sched/sch_fq.c`, `net/sched/sch_fq_codel.c`
- Linux kernel docs: `Documentation/networking/ip-sysctl.rst`, `Documentation/networking/scaling.rst`
