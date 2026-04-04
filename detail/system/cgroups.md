# The Mathematics of Cgroups -- Resource Allocation and Quota Accounting

> *Control groups impose hard mathematical boundaries on process resource consumption.*
> *Every CPU quota, memory limit, and I/O weight reduces to arithmetic on kernel counters,*
> *turning resource management into a problem of rate limiting, proportional allocation, and accounting.*

---

## 1. CPU Quota and Period Arithmetic (Rate Limiting)

### The Problem

CFS (Completely Fair Scheduler) enforces CPU limits using a quota/period model. A cgroup receives at most `quota` microseconds of CPU time per `period` microseconds. How do we calculate the effective CPU allocation?

### The Formula

$$\text{CPU}_{\text{effective}} = \frac{\text{quota}}{\text{period}} \times 100\%$$

For multi-core allocation:

$$\text{cores}_{\text{max}} = \frac{\text{quota}}{\text{period}}$$

The `cpu.max` file in cgroups v2 takes the form `quota period` (both in microseconds).

### Worked Examples

| Scenario | Quota (us) | Period (us) | Effective CPU | Cores |
|----------|-----------|------------|---------------|-------|
| Half a core | 50,000 | 100,000 | 50% | 0.5 |
| One full core | 100,000 | 100,000 | 100% | 1.0 |
| Two cores | 200,000 | 100,000 | 200% | 2.0 |
| Quarter core | 25,000 | 100,000 | 25% | 0.25 |
| Docker `--cpus=1.5` | 150,000 | 100,000 | 150% | 1.5 |
| Kubernetes `cpu: 500m` | 50,000 | 100,000 | 50% | 0.5 |

### Throttling Calculation

When a cgroup exhausts its quota within a period, CFS throttles it. The throttled percentage:

$$\text{throttle}_\% = \frac{\text{nr\_throttled}}{\text{nr\_periods}} \times 100\%$$

From `cpu.stat`:
```
nr_periods 1000
nr_throttled 150
throttled_usec 7500000
```

$$\text{throttle}_\% = \frac{150}{1000} \times 100 = 15\%$$

Average throttle duration per event:

$$\text{avg\_throttle} = \frac{\text{throttled\_usec}}{\text{nr\_throttled}} = \frac{7{,}500{,}000}{150} = 50{,}000 \, \mu s = 50 \, ms$$

## 2. CPU Weight Proportional Sharing (Fair Division)

### The Problem

When multiple cgroups compete for CPU, shares (v1) or weights (v2) determine proportional allocation. How much CPU does each cgroup get?

### The Formula

For $n$ competing cgroups with weights $w_1, w_2, \ldots, w_n$ on a system with $C$ total cores:

$$\text{CPU}_i = \frac{w_i}{\sum_{j=1}^{n} w_j} \times C$$

### Worked Examples

Three cgroups on a 4-core machine:

| Cgroup | Weight | Share of Total | Effective Cores |
|--------|--------|---------------|-----------------|
| web | 300 | 300/500 = 60% | 2.40 |
| api | 150 | 150/500 = 30% | 1.20 |
| worker | 50 | 50/500 = 10% | 0.40 |
| **Total** | **500** | **100%** | **4.00** |

When cgroups are not all busy, idle shares redistribute:

$$\text{CPU}_i^{\text{actual}} = \frac{w_i}{\sum_{j \in \text{active}} w_j} \times C$$

If `worker` is idle:

$$\text{CPU}_{\text{web}} = \frac{300}{300 + 150} \times 4 = 2.67 \text{ cores}$$

## 3. Memory Limit Accounting (Threshold Functions)

### The Problem

Cgroups v2 defines four memory thresholds: `memory.min`, `memory.low`, `memory.high`, and `memory.max`. Each triggers different kernel behavior.

### The Formula

Memory reclaim pressure function:

$$P(\text{reclaim}) = \begin{cases}
0 & \text{if } \text{usage} \leq \text{memory.low} \\
\frac{\text{usage} - \text{memory.low}}{\text{memory.high} - \text{memory.low}} & \text{if } \text{memory.low} < \text{usage} \leq \text{memory.high} \\
1 & \text{if } \text{usage} > \text{memory.high}
\end{cases}$$

Protection hierarchy (guaranteed memory):

$$\text{effective\_min}_i = \min\left(\text{memory.min}_i, \; \frac{\text{memory.min}_i}{\sum_j \text{memory.min}_j} \times \text{parent.memory.min}\right)$$

### Worked Examples

| Threshold | Value | Behavior When Exceeded |
|-----------|-------|----------------------|
| `memory.min` | 200MB | Absolute protection; kernel will not reclaim |
| `memory.low` | 400MB | Best-effort protection; reclaim only under pressure |
| `memory.high` | 768MB | Throttle allocations; trigger direct reclaim |
| `memory.max` | 1024MB | Hard limit; OOM kill if exceeded |

For a cgroup using 600MB with low=400MB, high=768MB:

$$P(\text{reclaim}) = \frac{600 - 400}{768 - 400} = \frac{200}{368} \approx 0.543$$

The kernel applies approximately 54.3% reclaim pressure relative to other cgroups.

## 4. Memory Usage Accounting (Page Counting)

### The Problem

What counts toward a cgroup's memory usage? The kernel accounts for specific page types.

### The Formula

$$\text{memory.current} = \text{anon} + \text{file} + \text{kernel\_stack} + \text{pagetables} + \text{percpu} + \text{sock} + \text{shmem} - \text{inactive\_reclaimable}$$

From `memory.stat`:

| Stat Field | Meaning | Counted In |
|-----------|---------|-----------|
| `anon` | Anonymous pages (heap, stack, mmap private) | Yes |
| `file` | Page cache (file-backed) | Yes |
| `kernel_stack` | Kernel stack pages | Yes |
| `pagetables` | Page table entries | Yes |
| `sock` | Network socket buffers | Yes |
| `shmem` | Shared memory (tmpfs) | Yes |
| `file_mapped` | Mapped file pages | Subset of `file` |
| `inactive_anon` | Inactive anonymous pages | Reclaimable |
| `inactive_file` | Inactive file pages | Reclaimable |

### Working Set Estimation

$$\text{working\_set} \approx \text{memory.current} - \text{inactive\_file}$$

This is the metric Kubernetes uses for eviction decisions.

## 5. I/O Weight Proportional Allocation (BFQ Scheduling)

### The Problem

The BFQ (Budget Fair Queueing) I/O scheduler allocates disk bandwidth proportionally among cgroups based on weights.

### The Formula

For $n$ cgroups competing for device bandwidth $B$ (bytes/sec):

$$\text{BW}_i = \frac{w_i}{\sum_{j=1}^{n} w_j} \times B$$

With hard limits (`io.max`) applied as a ceiling:

$$\text{BW}_i^{\text{actual}} = \min\left(\text{BW}_i^{\text{proportional}}, \; \text{io.max}_i\right)$$

### Worked Examples

Disk with 500 MB/s throughput, three cgroups:

| Cgroup | Weight | io.max (rbps) | Proportional BW | Actual BW |
|--------|--------|--------------|-----------------|-----------|
| database | 500 | unlimited | 312.5 MB/s | 312.5 MB/s |
| logging | 200 | 50 MB/s | 125.0 MB/s | 50.0 MB/s |
| backup | 100 | unlimited | 62.5 MB/s | 62.5 MB/s |

When `logging` is capped, its unused bandwidth redistributes:

$$\text{BW}_{\text{database}} = \frac{500}{500 + 100} \times (500 - 50) = \frac{500}{600} \times 450 = 375 \, \text{MB/s}$$

## 6. IOPS Budget Calculation (Latency)

### The Problem

Given an IOPS limit and average I/O size, what is the effective throughput?

### The Formula

$$\text{throughput} = \text{IOPS} \times \text{avg\_io\_size}$$

$$\text{latency}_{\text{avg}} = \frac{1}{\text{IOPS}} \times 1000 \; \text{ms}$$

### Worked Examples

| IOPS Limit | Avg I/O Size | Throughput | Avg Latency |
|-----------|-------------|-----------|-------------|
| 1,000 | 4 KB | 4 MB/s | 1.0 ms |
| 1,000 | 64 KB | 64 MB/s | 1.0 ms |
| 5,000 | 4 KB | 20 MB/s | 0.2 ms |
| 500 | 128 KB | 64 MB/s | 2.0 ms |

## 7. PID Limit and Fork Depth (Combinatorics)

### The Problem

A fork bomb creates $2^n$ processes after $n$ fork generations. How quickly does a PID limit prevent system collapse?

### The Formula

$$\text{processes}(n) = 2^n$$

Time to hit pid limit $L$ starting from 1 process:

$$n_{\text{max}} = \lfloor \log_2(L) \rfloor$$

### Worked Examples

| pids.max | Fork Generations to Hit Limit | Time at 1ms/fork |
|----------|-------------------------------|-------------------|
| 64 | 6 | 6 ms |
| 100 | 6 (stops at 64, 7th would be 128) | 6 ms |
| 256 | 8 | 8 ms |
| 1,024 | 10 | 10 ms |
| 4,096 | 12 | 12 ms |

Without PID limits, a fork bomb reaches 1 million processes in ~20 generations (~20ms).

## 8. Pressure Stall Information (PSI Metrics)

### The Problem

PSI quantifies the percentage of time tasks are stalled waiting for a resource. How do we interpret and alert on these values?

### The Formula

$$\text{PSI}_{\text{some}} = \frac{\text{time any task stalled}}{\text{wall clock time}} \times 100\%$$

$$\text{PSI}_{\text{full}} = \frac{\text{time all tasks stalled}}{\text{wall clock time}} \times 100\%$$

PSI trigger threshold for proactive action:

$$\text{alert if } \text{PSI}_{\text{some}}^{\text{avg10}} > T \text{ for } d \text{ seconds}$$

Typical thresholds:

| Resource | Warning (some avg10) | Critical (full avg10) |
|----------|---------------------|----------------------|
| CPU | > 25% | > 10% |
| Memory | > 10% | > 5% |
| I/O | > 15% | > 10% |

## Prerequisites

arithmetic, linux-kernel-basics, process-scheduling, memory-management, block-io

## Complexity

| Operation | Time | Space |
|-----------|------|-------|
| Create cgroup | O(1) | O(1) per cgroup node |
| Add process to cgroup | O(1) | O(1) pointer update |
| CPU quota check (per schedule) | O(1) | O(1) per-CPU counter |
| Memory charge (per page fault) | O(log n) tree walk | O(1) per page |
| IO throttle check | O(1) | O(1) per-device counter |
| PSI calculation | O(1) amortized | O(1) per-CPU state |
| cgroup tree traversal | O(n) cgroups | O(depth) stack |
| Memory reclaim scan | O(pages) in LRU | O(1) working state |
