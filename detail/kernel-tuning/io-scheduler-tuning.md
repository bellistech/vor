# I/O Scheduler Tuning --- Theory, Algorithms, and Performance Modeling

> Deep dive into Linux block layer scheduling algorithms, queue theory, bandwidth allocation math, and the trade-offs between latency, throughput, and fairness across rotational and solid-state storage.

## Prerequisites

- Familiarity with Linux block device layer (`/sys/block/`, blk-mq)
- Basic understanding of disk I/O (sectors, blocks, seeks, queues)
- Comfort with queueing theory concepts (arrival rate, service time)
- See: `kernel`, `iostat`, `io-scheduler-tuning` (cheatsheet)

## Complexity

- **Conceptual:** Moderate -- requires understanding of scheduling algorithms and queueing theory
- **Operational:** Low to moderate -- most tuning is sysfs writes and udev rules
- **Mathematical:** Moderate -- Little's Law, proportional share allocation, IOPS modeling

## 1. The Elevator Algorithm and Seek Optimization

The classical disk scheduling problem arises from the mechanical nature of HDDs: a read/write head must physically move (seek) to the correct track and wait for the platter to rotate to the correct sector. The cost function for an I/O operation on rotational media is:

$$T_{io} = T_{seek} + T_{rotation} + T_{transfer}$$

where:
- $T_{seek}$ is the time to move the head to the target track (0.5--15 ms typical)
- $T_{rotation}$ is the rotational latency, on average half a revolution: $\frac{1}{2} \cdot \frac{60}{RPM}$ seconds
- $T_{transfer}$ is the time to read/write the data once positioned

For a 7200 RPM drive:

$$T_{rotation} = \frac{1}{2} \cdot \frac{60}{7200} = 4.17 \text{ ms}$$

### 1.1 SCAN (Elevator) Algorithm

The SCAN algorithm moves the head in one direction, servicing all requests along the way, then reverses. This is analogous to an elevator. If the head is at position $h$ moving in direction $d \in \{+, -\}$, and $R = \{r_1, r_2, \ldots, r_n\}$ is the set of pending requests sorted by logical block address:

$$\text{Next request} = \begin{cases} \min\{r \in R : r \geq h\} & \text{if } d = + \\ \max\{r \in R : r \leq h\} & \text{if } d = - \end{cases}$$

The total seek distance for $n$ uniformly distributed requests across $L$ tracks is bounded by:

$$D_{SCAN} \leq 2L$$

regardless of $n$, because the head traverses the disk at most twice. Compare with FCFS (First Come First Served):

$$E[D_{FCFS}] = \frac{n \cdot L}{3}$$

For $n = 100$ requests on $L = 10{,}000$ tracks, SCAN reduces total seek distance by a factor of roughly $\frac{n}{6} \approx 16\times$.

### 1.2 C-SCAN and Deadline Variants

C-SCAN (Circular SCAN) only services requests in one direction, then returns to the beginning without servicing. This provides more uniform wait times. The `mq-deadline` scheduler builds on this by adding FIFO expiry: if a request has waited longer than its deadline ($T_{read} = 500$ ms, $T_{write} = 5000$ ms by default), it is promoted regardless of position.

The dispatch decision in mq-deadline is:

$$\text{dispatch}(r) = \begin{cases} r_{FIFO} & \text{if } \text{age}(r_{FIFO}) > T_{deadline} \\ r_{sorted} & \text{otherwise (C-SCAN order)} \end{cases}$$

This guarantees bounded latency: no request waits longer than its deadline class, at the cost of occasional seeks out of order. The `fifo_batch` parameter controls how many sorted-order requests are dispatched between deadline checks --- higher values improve throughput (fewer direction changes) but risk deadline violations under load.

### 1.3 Why Seek Optimization is Irrelevant for SSDs

SSDs and NVMe devices have no mechanical head. Access time is constant regardless of logical block address:

$$T_{io,SSD} = T_{access} + T_{transfer}$$

where $T_{access}$ is typically 25--100 us (NAND flash) or 10--20 us (NVMe with DRAM cache). Since $T_{seek} = 0$, the entire premise of elevator algorithms collapses. The scheduler degenerates to a pure queueing discipline, which is why `none` (passthrough) is optimal for NVMe.

## 2. BFQ Weight-Based Bandwidth Allocation

BFQ (Budget Fair Queueing) implements proportional-share scheduling. Each process (or cgroup) is assigned a weight $w_i$, and BFQ guarantees that the fraction of disk bandwidth allocated to process $i$ converges to:

$$B_i = \frac{w_i}{\sum_{j=1}^{n} w_j} \cdot B_{total}$$

where $B_{total}$ is the aggregate disk bandwidth and the sum is over all $n$ active processes.

### 2.1 Budget Assignment

BFQ does not allocate time slices like CFQ. Instead, it assigns a budget (in sectors) to each process queue. The budget for process $i$ is:

$$\text{budget}_i = \frac{w_i}{w_{max}} \cdot \text{max\_budget}$$

where $w_{max}$ is the maximum weight among active queues and `max_budget` is a tunable (default: auto-calculated from device throughput). A process can dispatch up to its budget in sectors before being preempted.

### 2.2 Fairness Guarantee

Consider three cgroups with weights $w_A = 500$, $w_B = 300$, $w_C = 200$:

$$B_A = \frac{500}{500 + 300 + 200} = 50\%$$
$$B_B = \frac{300}{1000} = 30\%$$
$$B_C = \frac{200}{1000} = 20\%$$

If the disk delivers 200 MB/s total:
- $A$ gets 100 MB/s
- $B$ gets 60 MB/s
- $C$ gets 40 MB/s

When $C$ goes idle, bandwidth is redistributed proportionally among active queues:

$$B_A' = \frac{500}{500 + 300} = 62.5\% = 125 \text{ MB/s}$$
$$B_B' = \frac{300}{800} = 37.5\% = 75 \text{ MB/s}$$

This is work-conserving: idle bandwidth is never wasted.

### 2.3 Low-Latency Mode

When `low_latency=1`, BFQ boosts interactive processes. A process is classified as interactive if its think time (gap between I/O completions and new submissions) exceeds a threshold. Interactive processes receive a weight boost:

$$w_i^{boosted} = w_i \cdot \frac{B_{total}}{B_{threshold}}$$

This allows a terminal or GUI application issuing small sporadic I/O to preempt a bulk sequential writer, dramatically improving perceived responsiveness. The trade-off is reduced throughput for background workloads.

### 2.4 BFQ vs. CFQ

BFQ is the multi-queue successor to CFQ (Completely Fair Queueing), which was removed in Linux 5.0. Key differences:

| Property | CFQ | BFQ |
|---|---|---|
| Queue architecture | Single-queue | Multi-queue (blk-mq) |
| Fairness unit | Time slices | Sector budgets |
| Idling | Always (hurts SSDs) | Selective (only when beneficial) |
| Interactive boost | Basic | Sophisticated heuristic |
| cgroup support | blkcg v1 | blkcg v1 + v2 |

## 3. Queue Depth, Latency, and Little's Law

Little's Law relates the average number of items in a queueing system to the arrival rate and average time spent:

$$L = \lambda \cdot W$$

where:
- $L$ = average number of requests in the system (queue + service)
- $\lambda$ = average arrival rate (requests per second)
- $W$ = average time a request spends in the system (queueing + service)

### 3.1 Application to I/O Scheduling

For a storage device, $L$ corresponds to the queue depth (the `nr_requests` parameter plus in-flight I/O). If we want to saturate a device capable of $\text{IOPS}_{max}$ with average service time $\bar{s}$:

$$L_{optimal} = \text{IOPS}_{max} \cdot \bar{s}$$

Example: an NVMe SSD with $\text{IOPS}_{max} = 500{,}000$ and $\bar{s} = 0.1$ ms:

$$L_{optimal} = 500{,}000 \cdot 0.0001 = 50$$

A queue depth of 50 is needed to saturate this device. Below this, the device is underutilized (requests complete faster than new ones arrive). Above this, excess requests queue and latency grows linearly:

$$W = \frac{L}{\lambda} = \frac{L}{\text{IOPS}_{max}} \quad \text{(at saturation)}$$

### 3.2 The Latency-Throughput Trade-off

Below saturation, increasing queue depth improves throughput because the device can begin servicing the next request immediately (no idle gaps). At saturation, additional queue depth only adds latency:

$$W_{total} = \bar{s} + W_{queue}$$

where $W_{queue}$ grows with offered load. For an M/M/1 queue model:

$$W_{queue} = \frac{\rho}{1 - \rho} \cdot \bar{s}$$

where $\rho = \lambda / \mu$ is the utilization factor and $\mu = 1/\bar{s}$ is the service rate. As $\rho \to 1$, $W_{queue} \to \infty$ -- the system becomes unstable.

This is why reducing `nr_requests` can dramatically improve tail latency at the cost of peak throughput. For latency-sensitive workloads (databases, key-value stores), operating at $\rho \approx 0.5\text{--}0.7$ is the sweet spot.

### 3.3 Kyber's Latency-Based Throttling

Kyber implements this trade-off directly. It maintains two token pools (read and write), and each submitted I/O must acquire a token. If the observed latency exceeds the target:

$$\text{if } \bar{W}_{observed} > T_{target}: \text{reduce tokens (throttle)}$$
$$\text{if } \bar{W}_{observed} < T_{target}: \text{increase tokens (allow more)}$$

This is a feedback control loop. Kyber uses exponentially weighted moving averages of completion latencies:

$$\bar{W}_{t} = \alpha \cdot W_t + (1 - \alpha) \cdot \bar{W}_{t-1}$$

where $\alpha$ is the smoothing factor. The token count is adjusted by domain (reads vs. writes), allowing Kyber to prioritize read latency over write latency, which matches most workload requirements.

## 4. IOPS Modeling for Different Workload Patterns

### 4.1 Random I/O on Rotational Media

For random 4KB reads on a 7200 RPM HDD:

$$\text{IOPS}_{random} = \frac{1}{T_{seek,avg} + T_{rotation,avg} + T_{transfer}}$$

With $T_{seek,avg} = 8$ ms, $T_{rotation,avg} = 4.17$ ms, $T_{transfer} \approx 0.05$ ms:

$$\text{IOPS}_{random} = \frac{1}{0.008 + 0.00417 + 0.00005} \approx 82 \text{ IOPS}$$

This is the fundamental limit of rotational media for random access. No scheduler can exceed this; the scheduler can only reduce wasted seeks.

### 4.2 Sequential I/O on Rotational Media

For sequential 128KB reads (no seek, minimal rotational latency after first block):

$$\text{Throughput}_{seq} = \frac{\text{block size}}{T_{rotation,partial} + T_{transfer}}$$

With sustained transfer rate of 150 MB/s (outer tracks, modern drive):

$$\text{IOPS}_{seq,128K} = \frac{150 \text{ MB/s}}{128 \text{ KB}} = 1{,}200 \text{ IOPS}$$

The scheduler's job for sequential workloads is to maintain contiguity -- this is where elevator algorithms excel and where `read_ahead_kb` has the most impact.

### 4.3 IOPS on NVMe

NVMe IOPS are dominated by NAND flash latency and controller parallelism:

$$\text{IOPS}_{NVMe} = \frac{N_{channels} \cdot N_{dies/channel}}{T_{page,read}}$$

A typical NVMe SSD with 8 channels, 8 dies per channel, and 50 us page read time:

$$\text{IOPS}_{NVMe} = \frac{8 \cdot 8}{0.00005} = 1{,}280{,}000 \text{ IOPS (theoretical)}$$

Practical IOPS are lower due to controller overhead, garbage collection, and FTL (Flash Translation Layer) lookups. Marketed figures of 500K--1M random read IOPS at QD=256 are typical.

### 4.4 Mixed Workload Modeling

For a mixed workload with read fraction $f_r$ and write fraction $f_w = 1 - f_r$:

$$\text{IOPS}_{mixed} = \frac{1}{\frac{f_r}{\text{IOPS}_{read}} + \frac{f_w}{\text{IOPS}_{write}}}$$

This is the harmonic mean weighted by the read/write ratio. For an SSD with 500K read IOPS and 200K write IOPS at a 70/30 mix:

$$\text{IOPS}_{mixed} = \frac{1}{\frac{0.7}{500{,}000} + \frac{0.3}{200{,}000}} = \frac{1}{1.4 \times 10^{-6} + 1.5 \times 10^{-6}} = \frac{1}{2.9 \times 10^{-6}} \approx 345{,}000 \text{ IOPS}$$

## 5. NVMe vs. Rotational Scheduling Considerations

### 5.1 Queue Architecture Comparison

| Property | HDD (SATA) | SSD (SATA) | NVMe |
|---|---|---|---|
| Hardware queues | 1 (NCQ, depth 32) | 1 (NCQ, depth 32) | 1--65535 (typically 32--128) |
| Seek penalty | 3--15 ms | 0 | 0 |
| Service time | 5--15 ms | 0.05--0.2 ms | 0.01--0.1 ms |
| IOPS (random 4K) | 75--200 | 10K--100K | 100K--1M+ |
| Benefits from elevator | Yes (major) | No | No |
| Benefits from merging | Yes (reduces seeks) | Marginal | Minimal |
| Optimal scheduler | mq-deadline / bfq | mq-deadline / kyber / none | none |

### 5.2 CPU Cost of Scheduling

On NVMe devices, the scheduler itself becomes a bottleneck. At 1M IOPS, each I/O passes through the scheduler in both submission and completion paths. The CPU cost per I/O for each scheduler (measured on Linux 5.x):

| Scheduler | CPU ns/IO (submit) | CPU ns/IO (complete) |
|---|---|---|
| none | ~50 | ~30 |
| mq-deadline | ~150 | ~100 |
| kyber | ~200 | ~120 |
| bfq | ~500 | ~300 |

At 1M IOPS, `bfq` consumes approximately 800 us per millisecond (80% of one CPU core) just for scheduling overhead. This is why `none` is strongly recommended for high-performance NVMe workloads.

### 5.3 Multi-Queue Topology

NVMe exposes multiple hardware submission and completion queues, typically one per CPU core. The blk-mq layer maps software queues to hardware queues:

$$\text{Software queues (ctx)} \xrightarrow{\text{scheduler}} \text{Hardware dispatch queues (hctx)} \xrightarrow{\text{driver}} \text{NVMe SQ/CQ pairs}$$

With `none`, this mapping is direct -- I/O submitted on CPU $k$ goes to hardware queue $k$ with no cross-CPU contention. With a scheduler, requests may be reordered across CPUs, introducing lock contention and cache bouncing:

$$T_{contention} \propto N_{CPUs} \cdot \text{IOPS}_{per\_CPU}$$

This is another reason `none` scales better on many-core systems with NVMe.

### 5.4 Decision Framework

The scheduler choice follows a simple decision tree:

1. **NVMe?** Use `none`. Hardware manages queues, scheduling adds only overhead.
2. **SSD in a VM guest?** Use `none`. Host handles scheduling.
3. **SSD, latency-critical?** Use `kyber`. Auto-tunes queue depth to hit latency targets.
4. **SSD, general purpose?** Use `mq-deadline`. Simple, low overhead, prevents starvation.
5. **HDD, mixed workloads or QoS needed?** Use `bfq`. Weight-based fairness, interactive boost.
6. **HDD, general purpose?** Use `mq-deadline`. Seek optimization with deadline guarantees.

For any device, measure with `fio` under your actual workload before committing to a scheduler. Synthetic benchmarks can mislead -- a scheduler that wins at QD=1 may lose at QD=256, and vice versa.
