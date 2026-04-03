# The Mathematics of nice — Priority Weights, CFS Scheduling & I/O Class Interaction

> *nice values are logarithmic priority weights. Each step of 1 changes CPU share by ~10%, and the exponential mapping from nice to weight is the foundation of the CFS proportional-share scheduler.*

---

## 1. Nice Value to CFS Weight Mapping

### The Exponential Formula

Each nice level changes the weight by a factor of $\approx 1.25$:

$$weight(nice) = \frac{1024}{1.25^{nice}}$$

Where:
- $nice \in [-20, 19]$ (40 levels)
- $weight(0) = 1024$ (baseline)
- Higher weight = more CPU time

### Complete Weight Table

| Nice | Weight | Ratio to Nice 0 | CPU Share (2 tasks) |
|:---:|:---:|:---:|:---:|
| -20 | 88,761 | 86.68x | 98.86% |
| -15 | 29,154 | 28.47x | 96.61% |
| -10 | 9,548 | 9.33x | 90.31% |
| -5 | 3,121 | 3.05x | 75.29% |
| -1 | 1,277 | 1.25x | 55.49% |
| 0 | 1,024 | 1.00x | 50.00% |
| 1 | 820 | 0.80x | 44.47% |
| 5 | 335 | 0.33x | 24.66% |
| 10 | 110 | 0.11x | 9.70% |
| 15 | 36 | 0.035x | 3.39% |
| 19 | 15 | 0.015x | 1.44% |

### The 10% Rule

Adjacent nice values differ by $\approx 10\%$ in CPU share:

$$\frac{weight(n)}{weight(n+1)} = 1.25$$

$$\frac{CPU(n)}{CPU(n+1)} \approx 1.25$$

This means each nice increment gives ~10% less CPU relative to the adjacent level.

---

## 2. CPU Share Calculation — Proportional Sharing

### Two-Task Model

With tasks at nice $a$ and $b$:

$$CPU_a = \frac{weight(a)}{weight(a) + weight(b)} \times 100\%$$

**Example:** Nice 0 vs nice 5:

$$CPU_0 = \frac{1024}{1024 + 335} = \frac{1024}{1359} = 75.3\%$$

$$CPU_5 = \frac{335}{1359} = 24.7\%$$

### Multi-Task Model

With $n$ tasks at various nice values:

$$CPU_i = \frac{weight(nice_i)}{\sum_{j=1}^{n} weight(nice_j)} \times 100\%$$

**Example:** Three tasks at nice -5, 0, +10:

$$\Sigma = 3121 + 1024 + 110 = 4255$$

| Task | Nice | Weight | CPU Share |
|:---:|:---:|:---:|:---:|
| A | -5 | 3121 | 73.4% |
| B | 0 | 1024 | 24.1% |
| C | 10 | 110 | 2.6% |

### Relative Share Between Two Specific Tasks

$$\frac{CPU_a}{CPU_b} = \frac{weight(a)}{weight(b)} = 1.25^{b-a}$$

Nice 0 vs nice 10:

$$ratio = 1.25^{10} = 9.31$$

Task at nice 0 gets 9.31x more CPU than nice 10.

---

## 3. Virtual Runtime (vruntime) Impact

### vruntime Growth Rate

$$\frac{d(vruntime)}{d(wall\_time)} = \frac{1024}{weight(nice)}$$

| Nice | Weight | vruntime Rate | Meaning |
|:---:|:---:|:---:|:---|
| -20 | 88,761 | 0.012x | vruntime barely advances |
| 0 | 1,024 | 1.000x | Baseline rate |
| 10 | 110 | 9.309x | vruntime advances 9x faster |
| 19 | 15 | 68.27x | vruntime races ahead |

### Worked Example

Two tasks run for 10 ms of wall time:

$$vruntime_{nice0} = 10ms \times \frac{1024}{1024} = 10ms$$

$$vruntime_{nice10} = 10ms \times \frac{1024}{110} = 93.1ms$$

CFS picks the task with lowest vruntime. After nice-0 runs 10ms, nice-10 must wait until its vruntime (growing faster) falls below nice-0's. In practice, nice-10 gets ~9x less CPU.

---

## 4. Timeslice Calculation

### CFS Targeted Latency

CFS doesn't use fixed timeslices. Instead:

$$timeslice_i = \frac{weight_i}{\sum weight_j} \times sched\_period$$

Where $sched\_period$:

$$sched\_period = \begin{cases} 6ms & \text{if } nr\_running \leq 8 \\ nr\_running \times 0.75ms & \text{if } nr\_running > 8 \end{cases}$$

### Minimum Granularity

$$timeslice_{min} = 0.75ms$$

No task gets less than this, regardless of nice value.

### Worked Example

20 runnable tasks, all nice 0 except one at nice -10:

$$sched\_period = 20 \times 0.75ms = 15ms$$

$$\Sigma weights = 19 \times 1024 + 9548 = 29004$$

$$timeslice_{nice-10} = \frac{9548}{29004} \times 15ms = 4.94ms$$

$$timeslice_{nice0} = \frac{1024}{29004} \times 15ms = 0.53ms$$

The nice -10 task gets ~9.3x the timeslice of each nice-0 task.

---

## 5. ionice — I/O Scheduling Priority

### I/O Classes

$$class \in \{0 (none), 1 (realtime), 2 (best\text{-}effort), 3 (idle)\}$$

### CFQ/BFQ Weight Mapping

Within best-effort class, 8 priority levels (0=highest, 7=lowest):

$$io\_weight(class2, prio) = \frac{8 - prio}{8} \times base\_weight$$

### Nice to ionice Default Mapping

Without explicit ionice, I/O priority derives from CPU nice:

$$io\_prio = \frac{nice + 20}{5}$$

| Nice Range | io_prio | I/O Weight |
|:---:|:---:|:---:|
| -20 to -17 | 0 (highest) | 100% |
| -16 to -12 | 1 | 87.5% |
| -11 to -7 | 2 | 75% |
| -6 to -2 | 3 | 62.5% |
| -1 to 3 | 4 | 50% |
| 4 to 8 | 5 | 37.5% |
| 9 to 13 | 6 | 25% |
| 14 to 19 | 7 (lowest) | 12.5% |

### I/O Idle Class

`ionice -c 3` (idle): process gets I/O only when no other process needs the disk:

$$IO_{idle} = \begin{cases} available & \text{if } IO_{others} = 0 \\ 0 & \text{if } IO_{others} > 0 \end{cases}$$

**Use case:** Backups, indexing — zero impact on interactive performance.

---

## 6. Real-Time Priority vs Nice

### Priority Domains

$$priority\_space = \underbrace{[0, 99]}_{real\text{-}time} \cup \underbrace{[100, 139]}_{normal\ (nice)}$$

Real-time always preempts normal:

$$\forall rt, normal: CPU(rt) \text{ takes priority}$$

### Mapping

| System | Range | Method |
|:---|:---:|:---|
| SCHED_FIFO | rt_prio 1-99 | Preemptive, no timeslice |
| SCHED_RR | rt_prio 1-99 | Round-robin, 100ms default timeslice |
| SCHED_OTHER (nice) | nice -20 to 19 | CFS proportional share |
| SCHED_BATCH | nice -20 to 19 | CFS, longer timeslice |
| SCHED_IDLE | none | Only when CPU idle |

### Why Not Use Real-Time?

$$risk = P(rt\_task\_infinite\_loop) \times cost(system\_hang)$$

A SCHED_FIFO task at priority 99 that doesn't yield will **starve everything** including the kernel's worker threads (on non-preemptible kernels).

---

## 7. Autogroup — Per-TTY Nice Groups

### The Autogroup Model (CONFIG_SCHED_AUTOGROUP)

Each TTY session gets its own scheduling group:

$$CPU_{tty} = \frac{weight_{autogroup}}{\sum weight_{all\_groups}}$$

$$CPU_{task \in tty} = \frac{weight_{task}}{\sum weight_{tasks\_in\_tty}} \times CPU_{tty}$$

### Impact on Interactive Performance

Without autogroup: `make -j$(nproc)` spawns $N$ tasks that compete equally with your terminal.

With autogroup: compilation gets one group's share, terminal gets another:

$$CPU_{interactive} \approx \frac{1}{2} \text{ (50\% regardless of compilation thread count)}$$

Without autogroup: $CPU_{interactive} \approx \frac{1}{N+1}$ where $N$ = compilation threads.

---

## 8. Summary of nice Mathematics

| Concept | Formula | Type |
|:---|:---|:---|
| Nice to weight | $1024 / 1.25^{nice}$ | Exponential |
| CPU share | $w_i / \sum w_j$ | Proportional |
| Adjacent ratio | $1.25$ (10% difference) | Geometric |
| vruntime rate | $1024 / weight$ | Inverse proportion |
| Timeslice | $w_i / \sum w_j \times period$ | Proportional |
| Nice to ionice | $(nice + 20) / 5$ | Linear mapping |
| Weight ratio | $1.25^{\Delta nice}$ | Exponential in difference |

---

*nice is a logarithmic dial on the CPU scheduler. Each step of 1 changes your share by ~10%, and the exponential weight mapping ensures that the difference between nice 0 and nice 10 is a 9:1 CPU ratio — not an absolute amount, but a proportional share of whatever CPU is available.*
