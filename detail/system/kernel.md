# The Mathematics of the Linux Kernel — Scheduling, Memory & OOM

> *The kernel is a real-time decision engine. Every context switch, every page fault, every OOM kill is governed by precise formulas balancing fairness, throughput, and survival.*

---

## 1. CFS Scheduler — Virtual Runtime (vruntime)

### The Core Formula

The Completely Fair Scheduler tracks each task's **virtual runtime** — a weighted measure of CPU consumption:

$$vruntime = \frac{wall\_time \times 1024}{weight}$$

Where:
- $wall\_time$ = actual CPU time consumed (nanoseconds)
- $weight$ = priority weight derived from nice value
- $1024$ = the weight of nice 0 (baseline)

Tasks with lower vruntime get scheduled next. Higher weight (lower nice) → slower vruntime growth → more CPU time.

### Nice Value to Weight Mapping

Each nice level changes weight by a factor of $\approx 1.25$:

$$weight(nice) = \frac{1024}{1.25^{nice}}$$

| Nice | Weight | Ratio to Nice 0 | vruntime Multiplier |
|:---:|:---:|:---:|:---:|
| -20 | 88,761 | 86.68x | 0.012x |
| -10 | 9,548 | 9.33x | 0.107x |
| -5 | 3,121 | 3.05x | 0.328x |
| 0 | 1,024 | 1.00x | 1.000x |
| 5 | 335 | 0.33x | 3.057x |
| 10 | 110 | 0.11x | 9.309x |
| 19 | 15 | 0.015x | 68.267x |

### Worked Example

Two tasks, nice 0 (weight 1024) and nice 5 (weight 335), both run for 10ms of wall time:

$$vruntime_{nice0} = \frac{10ms \times 1024}{1024} = 10ms$$

$$vruntime_{nice5} = \frac{10ms \times 1024}{335} = 30.57ms$$

The nice-5 task's vruntime grows 3x faster — it gets scheduled less often.

### Timeslice Calculation

CFS doesn't use fixed timeslices. Instead, the **ideal runtime** for a task within a scheduling period:

$$timeslice_i = \frac{weight_i}{\sum_{j} weight_j} \times sched\_period$$

Where $sched\_period$ defaults to 6ms for $\leq 8$ runnable tasks, or $0.75ms \times nr\_running$ otherwise.

**Example:** 3 tasks with nice values -5, 0, +5 (weights 3121, 1024, 335):

$$total\_weight = 3121 + 1024 + 335 = 4480$$

$$timeslice_{-5} = \frac{3121}{4480} \times 6ms = 4.18ms$$

$$timeslice_{0} = \frac{1024}{4480} \times 6ms = 1.37ms$$

$$timeslice_{+5} = \frac{335}{4480} \times 6ms = 0.45ms$$

---

## 2. CFS Red-Black Tree — O(log N) Scheduling

### Data Structure

CFS maintains runnable tasks in a **red-black tree** (self-balancing BST), keyed by vruntime.

| Operation | Complexity | Description |
|:---|:---:|:---|
| Pick next task | $O(1)$ | Leftmost node (cached) |
| Enqueue task | $O(\log N)$ | RB-tree insertion |
| Dequeue task | $O(\log N)$ | RB-tree deletion |
| Rebalance | $O(\log N)$ | At most 3 rotations |

For 1000 runnable tasks: $\log_2(1000) \approx 10$ comparisons to insert/remove.

### Min-vruntime Tracking

The scheduler maintains `cfs_rq->min_vruntime` — a monotonically increasing floor. New tasks are placed at:

$$vruntime_{new} = \max(min\_vruntime, vruntime_{parent})$$

This prevents starvation of existing tasks by newly forked processes.

---

## 3. OOM Killer — Score Calculation

### The OOM Score Formula

When the kernel runs out of memory, the OOM killer selects a victim. Each process gets a score from 0 to 1000:

$$oom\_score = \frac{RSS + swap\_usage}{total\_RAM + total\_swap} \times 1000 + oom\_score\_adj$$

Where:
- $RSS$ = Resident Set Size (physical memory used)
- $oom\_score\_adj$ = user-tunable value from -1000 to +1000

### Special Values

| oom_score_adj | Effect |
|:---:|:---|
| -1000 | **Never kill** (OOM-immune) |
| 0 | Default — pure memory-proportional |
| +1000 | **Always kill first** |

### Worked Example

System: 16 GB RAM, 4 GB swap. Process using 2 GB RSS, 500 MB swap:

$$base\_score = \frac{2048 + 512}{16384 + 4096} \times 1000 = \frac{2560}{20480} \times 1000 = 125$$

With `oom_score_adj = 200`:

$$oom\_score = 125 + 200 = 325$$

### OOM Kill Selection

The kernel iterates all processes and selects $\arg\max(oom\_score)$. Child processes of the victim are killed first if they don't share the same `mm_struct`.

---

## 4. Memory Zone Watermarks

### The Three Watermarks

The kernel maintains three watermarks per memory zone to trigger reclaim:

$$pages_{min} = \frac{min\_free\_kbytes}{page\_size \times nr\_zones}$$

$$pages_{low} = pages_{min} \times \frac{5}{4}$$

$$pages_{high} = pages_{min} \times \frac{3}{2}$$

### Zone States

| Free Pages | State | Action |
|:---|:---|:---|
| $> pages_{high}$ | Normal | No reclaim |
| $pages_{low} < free \leq pages_{high}$ | Warning | kswapd wakes |
| $pages_{min} < free \leq pages_{low}$ | Critical | kswapd aggressive reclaim |
| $\leq pages_{min}$ | Emergency | Direct reclaim (synchronous, blocks allocator) |

### Worked Example

System with `min_free_kbytes = 65536` (64 MB), page size 4 KB, 3 zones (DMA, DMA32, Normal):

$$pages_{min} = \frac{65536}{4 \times 3} = 5461 \text{ pages} \approx 21.3 \text{ MB per zone}$$

$$pages_{low} = 5461 \times 1.25 = 6826 \text{ pages} \approx 26.7 \text{ MB}$$

$$pages_{high} = 5461 \times 1.5 = 8192 \text{ pages} \approx 32.0 \text{ MB}$$

---

## 5. Page Cache Hit Rate Modeling

### Hit Rate Formula

$$hit\_rate = \frac{cache\_hits}{cache\_hits + cache\_misses}$$

The effective I/O time with caching:

$$T_{effective} = hit\_rate \times T_{memory} + (1 - hit\_rate) \times T_{disk}$$

### Worked Example

$T_{memory} \approx 100ns$, $T_{disk} \approx 10ms$ (HDD) or $100\mu s$ (NVMe SSD):

| Hit Rate | T_eff (HDD) | T_eff (NVMe) | Speedup vs No Cache |
|:---:|:---:|:---:|:---:|
| 0% | 10 ms | 100 us | 1x |
| 90% | 1.0 ms | 10.1 us | 10x |
| 99% | 100 us | 1.1 us | 100x |
| 99.9% | 10.1 us | 200 ns | ~1000x |

### Working Set Model

The page cache is most effective when the **working set** fits in RAM. If working set $W > RAM$:

$$hit\_rate \approx \frac{RAM}{W}$$

This is the **LRU steady-state** approximation. A 64 GB working set on 32 GB RAM yields $\approx 50\%$ hit rate.

---

## 6. Context Switch Cost

### Components

$$T_{switch} = T_{save\_registers} + T_{TLB\_flush} + T_{cache\_pollution}$$

| Component | Typical Cost | Notes |
|:---|:---:|:---|
| Register save/restore | 0.1-0.5 us | Fixed, ~30 registers on x86-64 |
| TLB flush | 0.5-2.0 us | Mitigated by PCID/ASID |
| Cache pollution | 1-50 us | Depends on working set overlap |
| **Total (same process)** | **1-3 us** | Threads sharing address space |
| **Total (cross process)** | **3-10 us** | Full TLB flush + cache cold start |

### Throughput Impact

With $N$ context switches per second and cost $T_{switch}$:

$$overhead = N \times T_{switch}$$

At 10,000 switches/sec with 5 us each: $overhead = 50ms/s = 5\%$ CPU lost to switching.

---

## 7. Load Average — Exponential Moving Average

### The Formula

Linux load average is an **exponentially weighted moving average** of the run queue length:

$$load(t) = load(t-1) \times e^{-\Delta t / \tau} + n \times (1 - e^{-\Delta t / \tau})$$

Where:
- $n$ = current number of runnable + uninterruptible tasks
- $\Delta t$ = sample interval (5 seconds in Linux)
- $\tau$ = time constant (60s, 300s, or 900s for 1/5/15-min averages)

### Decay Constants

The kernel precomputes $e^{-5/\tau}$ for each average:

| Average | $\tau$ | $e^{-5/\tau}$ | 63% response time |
|:---:|:---:|:---:|:---:|
| 1-min | 60s | 0.9200 | 1 minute |
| 5-min | 300s | 0.9835 | 5 minutes |
| 15-min | 900s | 0.9945 | 15 minutes |

### Worked Example

Current 1-min load = 2.0, suddenly 10 tasks become runnable:

After 5 seconds: $load = 2.0 \times 0.92 + 10 \times 0.08 = 1.84 + 0.80 = 2.64$

After 60 seconds (12 samples): $load \approx 2.0 \times 0.92^{12} + 10 \times (1 - 0.92^{12}) = 0.73 + 6.32 = 7.05$

After 300 seconds: load $\approx 9.8$ (approaching 10).

---

## 8. Summary of Kernel Mathematics

| Domain | Formula | Type |
|:---|:---|:---|
| CFS vruntime | $wall\_time \times 1024 / weight$ | Weighted linear |
| Nice→Weight | $1024 / 1.25^{nice}$ | Exponential mapping |
| Timeslice | $weight_i / \Sigma weight \times period$ | Proportional share |
| OOM score | $(RSS + swap) / (RAM + swap) \times 1000 + adj$ | Linear scoring |
| Watermarks | $min \times \{1, 5/4, 3/2\}$ | Fixed ratios |
| Page cache | $hits / (hits + misses)$ | Ratio/probability |
| Load average | $EMA$ with decay $e^{-\Delta t / \tau}$ | Exponential smoothing |
| Context switch | $\Sigma(save + TLB + cache)$ | Additive cost model |

---

*Every scheduling decision, every page reclaim, every OOM kill — the kernel is solving optimization problems thousands of times per second with these formulas.*
