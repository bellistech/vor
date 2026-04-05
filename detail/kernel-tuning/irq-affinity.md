# IRQ Affinity & Interrupt Tuning -- Deep Dive

> Interrupts are the nervous system of I/O. How you route them determines whether
> your server processes packets in 2 microseconds or 200. This document covers
> the theory, math, and architectural trade-offs behind interrupt distribution
> on multi-core Linux systems.

---

## Prerequisites

- Understanding of CPU caches (L1/L2/L3), NUMA topology, and cache coherence protocols (MESI/MOESI)
- Familiarity with Linux network stack basics (sk_buff, softirq, NAPI)
- Basic probability and queueing theory (M/M/1, Little's Law)
- Companion sheet: `sheets/kernel-tuning/irq-affinity.md`

## Complexity

| Aspect | Rating |
|--------|--------|
| Conceptual difficulty | Medium-High |
| Implementation risk | Medium (mis-pinning can halve throughput) |
| Debugging difficulty | High (symptoms are statistical, not deterministic) |
| Kernel version sensitivity | Medium (NAPI/XPS APIs stabilized ~4.x, threaded NAPI ~5.17) |

---

## 1. Interrupt Coalescing: The Latency-Throughput Trade-Off

Every hardware interrupt has a fixed cost: saving registers, flushing pipeline, taking the IDT vector, running the ISR, scheduling softirq, then restoring context. On modern x86, this costs roughly 2-5 microseconds of CPU time per interrupt.

### 1.1 The Coalescing Model

Let:

- $\lambda$ = packet arrival rate (packets/sec)
- $C_{irq}$ = CPU cost per interrupt (seconds)
- $n$ = coalescing depth (packets per interrupt)
- $\tau$ = coalescing timeout (seconds)
- $D_{coal}$ = additional latency from coalescing

Without coalescing, interrupt overhead as a fraction of one CPU core:

$$U_{irq} = \lambda \cdot C_{irq}$$

At 1 Mpps with $C_{irq} = 3\mu s$, that is $U_{irq} = 3.0$ -- three full CPU cores consumed by interrupt handling alone.

With coalescing depth $n$, the interrupt rate drops:

$$U_{irq}(n) = \frac{\lambda}{n} \cdot C_{irq}$$

But the added latency for the $k$-th packet in a coalescing window is:

$$D_k = \frac{(n - k)}{n} \cdot \tau$$

The average additional latency across all packets in a burst:

$$\bar{D}_{coal} = \frac{\tau}{2} \cdot \frac{(n-1)}{n}$$

For large $n$, this converges to $\tau / 2$.

### 1.2 Optimal Coalescing Depth

Define a cost function that weights CPU overhead and latency:

$$J(n) = \alpha \cdot \frac{\lambda \cdot C_{irq}}{n} + (1 - \alpha) \cdot \frac{\tau(n-1)}{2n}$$

where $\alpha \in [0,1]$ controls the throughput-vs-latency preference. Taking $\frac{\partial J}{\partial n} = 0$:

$$n^* = \sqrt{\frac{\alpha \cdot \lambda \cdot C_{irq} \cdot 2}{(1-\alpha) \cdot \tau}} \cdot n$$

This simplifies (assuming $\tau$ scales linearly with $n$ as $\tau = n / \lambda$) to:

$$n^* = \sqrt{\frac{2\alpha \cdot \lambda^2 \cdot C_{irq}}{(1-\alpha)}}$$

### 1.3 Adaptive Coalescing

Modern NICs (Intel ice, Mellanox mlx5) implement adaptive coalescing that dynamically adjusts $n$ and $\tau$:

- **Low load** ($\lambda < \lambda_{thresh}$): disable coalescing, interrupt per packet for minimum latency
- **High load** ($\lambda > \lambda_{thresh}$): increase $n$ proportionally, capping at a maximum timeout

The kernel's `ethtool -C` exposes this as `adaptive-rx on`. The driver maintains a moving average of inter-packet arrival times and adjusts coalescing parameters each NAPI cycle.

### 1.4 Practical Regimes

| Workload | Optimal $n$ | Timeout | Rationale |
|-----------|------------|---------|-----------|
| HFT / market data | 1 (off) | 0 | Every microsecond matters |
| Web server (10G) | 64-128 | 50-100us | Balanced CPU/latency |
| Bulk transfer (100G) | 256+ | 200us+ | CPU is the bottleneck |
| Storage (NVMe-oF) | 16-32 | 20us | IOPS-sensitive, moderate latency |

---

## 2. Cache Locality Effects of IRQ Pinning

### 2.1 The Cache Miss Cost Model

When a packet arrives, the NIC DMAs the packet into a ring buffer. The CPU that handles the interrupt must:

1. Read the ring buffer descriptor (64B, cold if on a different CPU)
2. Read packet headers for RSS/flow classification (64-128B)
3. Allocate and populate an `sk_buff` (~240B)
4. Walk protocol handler chains (code cache)

If the interrupt handler CPU differs from the application CPU, the `sk_buff` and packet data must migrate through the cache coherence protocol. On a typical Xeon:

| Transfer | Latency |
|----------|---------|
| L1 hit | ~1 ns |
| L2 hit | ~4 ns |
| L3 hit (same die) | ~12 ns |
| L3 hit (cross-die, same socket) | ~30 ns |
| Cross-socket (QPI/UPI) | ~80-120 ns |

### 2.2 Working Set Analysis

For a single flow being processed, the hot working set includes:

$$W_{flow} = S_{skb} + S_{headers} + S_{socket} + S_{proto}$$

Where:

- $S_{skb} \approx 240$ bytes (struct sk_buff)
- $S_{headers} \approx 128$ bytes (L2-L4 headers)
- $S_{socket} \approx 2048$ bytes (struct sock + tcp_sock)
- $S_{proto} \approx 4096$ bytes (protocol handler code)

Total per-flow working set: ~6.5 KB. With $F$ concurrent flows on a CPU:

$$W_{total} = F \cdot W_{flow}$$

An L1d cache (48 KB on recent Intel) can hold approximately 7 flows before thrashing. L2 (1.25 MB) can hold ~190 flows comfortably.

### 2.3 The Pinning Principle

Optimal interrupt routing minimizes the total cache miss cost:

$$C_{miss} = \sum_{f \in flows} \mathbb{1}[CPU_{irq}(f) \neq CPU_{app}(f)] \cdot P_{miss} \cdot L_{miss}$$

where $P_{miss}$ is the probability of a cache miss on cross-CPU access and $L_{miss}$ is the miss latency.

**RFS solves this directly** by steering the interrupt to the CPU running the application's `recvmsg()` syscall. Without RFS, manual IRQ pinning must approximate this by co-locating interrupt handling and application threads on the same core or at least the same L3 domain.

### 2.4 Quantifying the Benefit

Empirically, correct IRQ pinning (same core as application) vs. random placement:

- **Latency**: 15-40% reduction in p99 latency
- **Throughput**: 10-25% improvement in packets/sec per core
- **CPU efficiency**: 20-30% reduction in cycles per packet

These numbers come from the fact that cross-core `sk_buff` migration on a modern Xeon costs ~30ns, and at 1 Mpps that is 30ms of CPU time per second -- wasted purely on cache misses.

---

## 3. Multi-Queue NIC Scaling Analysis

### 3.1 RSS and the Toeplitz Hash

Receive Side Scaling (RSS) distributes incoming packets across $Q$ hardware queues using a hash of the flow tuple:

$$q = H_{Toeplitz}(src\_ip, dst\_ip, src\_port, dst\_port) \mod Q$$

The Toeplitz hash computes:

$$H = \bigoplus_{i=0}^{n-1} \begin{cases} K[i:i+32] & \text{if } input\_bit_i = 1 \\ 0 & \text{otherwise} \end{cases}$$

where $K$ is a 40-byte (320-bit) secret key and $\oplus$ is XOR. The key is programmed via `ethtool -X`.

### 3.2 Queue Imbalance

With $F$ flows distributed across $Q$ queues via a hash, the expected number of flows per queue is $F/Q$. But the actual distribution follows a balls-into-bins model. The maximum loaded queue has:

$$E[\max_q n_q] \approx \frac{F}{Q} + \sqrt{\frac{2F \ln Q}{Q}}$$

For $F = 10000$ flows and $Q = 8$ queues:

$$E[\max] \approx 1250 + \sqrt{\frac{20000 \cdot 2.08}{8}} \approx 1250 + 72 \approx 1322$$

This ~6% imbalance is acceptable. But for elephant flows (where one flow dominates bandwidth), hash-based distribution fails entirely. The heaviest queue can be 10x more loaded than average.

### 3.3 Scaling Efficiency

Define scaling efficiency $\eta$ for $Q$ queues versus a single queue:

$$\eta(Q) = \frac{T(Q)}{Q \cdot T(1)}$$

where $T(Q)$ is total throughput with $Q$ queues. In practice:

| Queues | Theoretical max | Observed efficiency | Limiting factor |
|--------|----------------|--------------------:|-----------------|
| 1 | 1.0x | 100% | baseline |
| 2 | 2.0x | 95-98% | minimal contention |
| 4 | 4.0x | 90-95% | per-queue lock contention |
| 8 | 8.0x | 85-92% | L3 cache pressure |
| 16 | 16.0x | 75-85% | memory bandwidth, NUMA effects |
| 32 | 32.0x | 60-75% | diminishing returns, cross-socket |

Beyond ~16 queues on a single socket, memory bandwidth becomes the bottleneck. The per-packet memory access pattern (DMA + descriptor read + skb alloc + protocol walk) consumes roughly 2-4 cache lines per packet, so at 10 Mpps that is 1.28-2.56 GB/s of cache-line traffic.

### 3.4 Optimal Queue Count

The optimal number of queues $Q^*$ satisfies:

$$Q^* = \min(N_{cpu\_local}, Q_{max\_hw}, Q_{effective})$$

where:

- $N_{cpu\_local}$ = CPUs on the NIC's NUMA node
- $Q_{max\_hw}$ = NIC's maximum supported queue count
- $Q_{effective}$ = throughput-optimal count (typically where $\eta(Q) > 0.85$)

Rule of thumb: use one queue per NUMA-local core, never exceed the number of local cores, and never cross NUMA boundaries for interrupt handling.

---

## 4. Busy Polling Latency Model

### 4.1 Traditional Interrupt Path

In the standard interrupt-driven path, packet delivery latency has these components:

$$D_{irq} = D_{wire} + D_{nic} + D_{dma} + D_{irq\_delivery} + D_{softirq\_sched} + D_{softirq\_run} + D_{socket\_wakeup} + D_{context\_switch}$$

Typical values:

| Component | Duration |
|-----------|----------|
| $D_{wire}$ | ~5 us (1 MTU at 10G) |
| $D_{nic}$ | 1-3 us (NIC processing) |
| $D_{dma}$ | 0.5-1 us (PCIe DMA) |
| $D_{irq\_delivery}$ | 1-2 us (APIC routing) |
| $D_{softirq\_sched}$ | 0-5 us (ksoftirqd wakeup, variable) |
| $D_{softirq\_run}$ | 2-5 us (NAPI poll, protocol handling) |
| $D_{socket\_wakeup}$ | 1-3 us (wake blocked thread) |
| $D_{context\_switch}$ | 2-5 us (restore app context) |

**Total**: ~13-30 us typical, with long-tail outliers to 50-100 us from softirq scheduling jitter.

### 4.2 Busy Polling Path

With busy polling (`SO_BUSY_POLL`), the application polls the NIC directly from `recvmsg()`:

$$D_{poll} = D_{wire} + D_{nic} + D_{dma} + D_{poll\_interval}/2 + D_{napi\_poll}$$

The key savings: no interrupt delivery, no softirq scheduling, no context switch. The cost is the polling interval -- on average, the packet sits in the DMA ring for half the poll interval before being noticed.

| Component | Duration |
|-----------|----------|
| $D_{wire}$ | ~5 us |
| $D_{nic}$ | 1-3 us |
| $D_{dma}$ | 0.5-1 us |
| $D_{poll\_interval}/2$ | 1-25 us (configurable) |
| $D_{napi\_poll}$ | 1-3 us |

**Total**: ~8-37 us, but with **near-zero jitter** -- the variance is what matters for p99.

### 4.3 Break-Even Analysis

Busy polling trades CPU cycles for latency reduction. The CPU cost of polling is:

$$C_{poll} = \frac{T_{poll}}{T_{busy\_poll}} \cdot C_{cpu}$$

where $T_{poll}$ is time spent polling (spinning), $T_{busy\_poll}$ is the configured poll duration, and $C_{cpu}$ is the cost of one CPU core.

Busy polling is justified when:

$$\frac{\Delta D_{p99} \cdot V_{latency}}{C_{cpu}} > 1$$

where $\Delta D_{p99}$ is the p99 latency improvement and $V_{latency}$ is the business value of that improvement (e.g., revenue per microsecond in HFT).

### 4.4 The Deferred IRQ Hybrid

Linux 5.17+ introduced `napi_defer_hard_irqs` combined with `gro_flush_timeout`, which provides a middle ground:

1. First packet triggers a hard IRQ
2. NAPI poll processes the ring
3. Instead of re-enabling interrupts, the kernel defers for `gro_flush_timeout` nanoseconds
4. During the deferral, a high-resolution timer fires to re-poll
5. After `napi_defer_hard_irqs` consecutive empty polls, hard IRQs are re-enabled

This gives busy-poll-like latency without dedicating a CPU core to spinning:

$$D_{deferred} \approx D_{poll} + \epsilon_{timer}$$

where $\epsilon_{timer}$ is the hrtimer scheduling jitter (typically < 5 us).

---

## 5. NUMA-Aware Interrupt Distribution Strategies

### 5.1 The NUMA Tax

Accessing memory on a remote NUMA node incurs a latency penalty:

$$L_{remote} = L_{local} \cdot R_{numa}$$

where $R_{numa}$ is the NUMA ratio (typically 1.5-2.5x depending on topology). For interrupt handling, this means:

- DMA buffer reads from remote memory: +40-80 ns per packet
- sk_buff allocation from remote node: +40-80 ns
- Socket buffer enqueue to remote-node socket: +40-80 ns
- Protocol handler code from remote node icache: +20-40 ns

Total NUMA penalty per packet: **140-280 ns**. At 1 Mpps per queue, that is 0.14-0.28 CPU-seconds wasted per second per queue.

### 5.2 The Affinity Hierarchy

Optimal interrupt placement follows a strict hierarchy:

1. **Same core as application** (ideal): zero cross-core traffic, L1/L2 warm
2. **Same L3 / CCD as application**: L3 hit (~12 ns), good enough for most workloads
3. **Same socket, different L3**: cache-to-cache transfer (~30 ns), acceptable
4. **Cross-socket**: QPI/UPI transfer (~100 ns), avoid if at all possible

### 5.3 NUMA-Aware Queue Assignment Algorithm

Given:

- $N$ NUMA nodes, each with $C_n$ cores
- $K$ NICs, each with $Q_k$ queues and attached to NUMA node $numa(k)$
- Application threads with known CPU affinity

The assignment minimizes total NUMA-crossing cost:

$$\min \sum_{k,q} \sum_{f \in flows(k,q)} cost(CPU_{irq}(k,q),\ CPU_{app}(f))$$

where:

$$cost(c_1, c_2) = \begin{cases} 0 & \text{same core} \\ \alpha & \text{same L3} \\ \beta & \text{same socket} \\ \gamma & \text{cross-socket} \end{cases}$$

with $0 < \alpha < \beta < \gamma$.

### 5.4 Practical Strategy: The Local-Node-Only Rule

The simplest effective strategy:

1. Determine NIC's NUMA node: `cat /sys/class/net/eth0/device/numa_node`
2. Set queue count to NUMA-local core count: `ethtool -L eth0 combined $N_local`
3. Pin each queue's IRQ to one NUMA-local core (1:1)
4. Set XPS to use only NUMA-local cores
5. Set RPS cpumask to NUMA-local cores
6. Bind application threads to the same NUMA node

This alone captures 80-90% of the performance benefit. The remaining 10-20% requires flow-level steering (RFS) to co-locate each flow's interrupt with its specific application thread.

### 5.5 Multi-NIC NUMA Topology

For servers with multiple NICs across NUMA nodes:

```
  NUMA Node 0           NUMA Node 1
  +----------+          +----------+
  | CPU 0-15 |--QPI/UPI--| CPU 16-31|
  | L3: 30MB |          | L3: 30MB |
  +----+-----+          +----+-----+
       |                     |
   +---+---+             +---+---+
   | NIC 0 |             | NIC 1 |
   | eth0  |             | eth1  |
   +-------+             +-------+
```

Traffic arriving on eth0 should be processed entirely within Node 0. If the application needs data from both NICs, use a message-passing or shared-memory design that minimizes cross-node data movement rather than processing remote-node interrupts.

### 5.6 Monitoring and Validation

Key metrics to validate correct NUMA-aware placement:

$$\text{NUMA miss ratio} = \frac{\text{remote\_node\_access\_count}}{\text{total\_memory\_access\_count}}$$

Measure with:

```
perf stat -e node-loads,node-load-misses -a -C 0-7 sleep 10
```

A NUMA miss ratio above 10% indicates misplaced interrupts or application threads. Target: < 5% for latency-sensitive workloads, < 10% for throughput workloads.

---

## Summary of Key Equations

| Quantity | Formula |
|----------|---------|
| IRQ CPU overhead | $U_{irq} = \lambda \cdot C_{irq}$ |
| Coalesced overhead | $U_{irq}(n) = \frac{\lambda}{n} \cdot C_{irq}$ |
| Avg coalescing latency | $\bar{D} = \frac{\tau(n-1)}{2n}$ |
| Flow working set | $W = F \cdot (S_{skb} + S_{hdr} + S_{sock} + S_{proto})$ |
| Max queue load (hashing) | $\frac{F}{Q} + \sqrt{\frac{2F \ln Q}{Q}}$ |
| NUMA penalty per packet | $\Delta L \approx 140\text{-}280\ \text{ns}$ |
| Busy poll avg wait | $D_{wait} = \frac{T_{poll\_interval}}{2}$ |
