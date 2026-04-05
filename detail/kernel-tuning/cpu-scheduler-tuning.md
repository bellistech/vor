# CPU Scheduler Tuning -- Theory, Math, and Microarchitectural Effects

> *The Linux CPU scheduler is a runtime optimizer solving a multi-objective problem: minimize latency, maximize throughput, enforce fairness, and respect real-time guarantees -- all while navigating the physical realities of cache hierarchies, NUMA topologies, and power domains. Understanding the math beneath the tunables transforms scheduler tuning from guesswork into engineering.*

---

## 1. CFS Virtual Runtime and the Red-Black Tree

### The Problem

The Completely Fair Scheduler must allocate CPU time proportionally to task weights while maintaining $O(\log n)$ scheduling decisions. How does the virtual runtime abstraction achieve weighted fairness?

### The Model

CFS maintains a time-ordered red-black tree keyed on **virtual runtime** ($vruntime$). The leftmost node (smallest $vruntime$) is always the next task to run. The tree guarantees $O(\log n)$ insertion, deletion, and minimum extraction.

For a task $i$ with weight $w_i$, the virtual runtime advances as:

$$vruntime_i(t + \Delta t) = vruntime_i(t) + \Delta t \cdot \frac{w_0}{w_i}$$

where $w_0 = 1024$ is the weight of a nice-0 task (the reference weight) and $\Delta t$ is the wall-clock time the task actually ran.

A high-weight (low-nice) task accumulates $vruntime$ slowly, so it stays near the left of the tree and gets picked more often. A low-weight (high-nice) task accumulates $vruntime$ quickly and drifts rightward.

### Fairness Guarantee

Over a scheduling period $T$ with $n$ runnable tasks, the ideal CPU share for task $i$ is:

$$\text{share}_i = \frac{w_i}{\sum_{j=1}^{n} w_j}$$

CFS approximates this by ensuring that after every **scheduling latency period** ($sched\_latency\_ns$, default 6ms when $n \leq 8$), every runnable task has received at least one timeslice. When $n > 8$, the period extends to $n \times sched\_min\_granularity\_ns$ to prevent excessive context switching.

The actual timeslice for task $i$ in a period is:

$$\text{timeslice}_i = \max\left(\frac{w_i}{\sum_{j=1}^{n} w_j} \cdot sched\_latency, \; sched\_min\_granularity\right)$$

### Why It Works

After one full scheduling period, all tasks have approximately equal $vruntime$. The maximum $vruntime$ spread in the tree at any moment is bounded by $sched\_latency$:

$$\max_i(vruntime_i) - \min_i(vruntime_i) \leq sched\_latency$$

This spread is the **scheduling jitter bound** -- no task can be starved for longer than one scheduling period.

---

## 2. Nice-to-Weight Mapping

### The Formula

Linux maps nice values $[-20, 19]$ to weights using a multiplicative scale where each nice level represents approximately a **10% CPU share difference** (or equivalently, a factor of $\approx 1.25$ in weight ratio per nice step).

The weight array in `kernel/sched/core.c` follows:

$$w(\text{nice}) = \frac{1024}{1.25^{\text{nice}}}$$

The exact values are precomputed to avoid floating-point:

| Nice | Weight | Nice | Weight |
|------|--------|------|--------|
| -20  | 88761  | 0    | 1024   |
| -10  | 9548   | 10   | 110    |
| -5   | 3121   | 15   | 36     |
| -1   | 1277   | 19   | 15     |

### CPU Share Between Two Tasks

Given two tasks with nice values $a$ and $b$:

$$\frac{\text{CPU}_a}{\text{CPU}_b} = \frac{w(a)}{w(b)} = 1.25^{b - a}$$

For example, nice 0 vs. nice 5:

$$\frac{w(0)}{w(5)} = 1.25^5 \approx 3.05$$

The nice-0 task gets about 3x more CPU than the nice-5 task. This ratio is **independent of other tasks in the system** -- adding or removing tasks changes absolute shares but not pairwise ratios.

### Inverse Weight

CFS also precomputes inverse weights for efficient $vruntime$ calculation:

$$\text{inv\_weight}(\text{nice}) = \left\lfloor \frac{2^{32}}{w(\text{nice})} \right\rfloor$$

This allows the kernel to compute $\Delta vruntime = \Delta t \cdot w_0 / w_i$ using integer multiplication and right-shift instead of division.

---

## 3. SCHED_DEADLINE Admission Control

### The Problem

SCHED_DEADLINE implements Earliest Deadline First (EDF) scheduling with Constant Bandwidth Server (CBS) isolation. How does the kernel guarantee that accepting a new deadline task will not cause any existing deadline task to miss its deadline?

### The Admission Test

Each SCHED_DEADLINE task $\tau_i$ is described by a triple $(C_i, T_i, D_i)$:

- $C_i$ = **runtime** (worst-case execution time per period)
- $T_i$ = **period** (minimum inter-arrival time)
- $D_i$ = **relative deadline** ($\leq T_i$)

The **utilization** of task $\tau_i$ is:

$$U_i = \frac{C_i}{T_i}$$

The kernel applies the **Liu and Layland utilization bound**. A new task $\tau_{n+1}$ is admitted if and only if:

$$\sum_{i=1}^{n+1} \frac{C_i}{T_i} \leq U_{max}$$

where $U_{max}$ is typically set to $\frac{m \cdot sched\_rt\_runtime\_us}{sched\_rt\_period\_us}$ for $m$ CPUs, defaulting to approximately $0.95m$.

### EDF Optimality

For **uniprocessor** systems with implicit deadlines ($D_i = T_i$), EDF is **optimal**: if any algorithm can schedule the task set without deadline misses, EDF can too. The necessary and sufficient condition is:

$$\sum_{i=1}^{n} \frac{C_i}{T_i} \leq 1$$

For **multiprocessor** systems, EDF is no longer optimal. The kernel uses **partitioned EDF** (tasks pinned to CPUs) or **global EDF** depending on configuration. Partitioned EDF reduces to $m$ independent uniprocessor problems:

$$\forall k \in [1, m]: \sum_{i \in \text{CPU}_k} \frac{C_i}{T_i} \leq 1$$

### CBS Bandwidth Isolation

The CBS mechanism ensures that a deadline task that overruns its declared runtime $C_i$ does not steal bandwidth from other tasks. When a task exhausts its runtime budget, its deadline is **pushed forward** by one period:

$$d_i^{\text{new}} = d_i^{\text{old}} + T_i, \quad \text{budget replenished to } C_i$$

This effectively throttles the overrunning task without affecting others.

---

## 4. Queuing Theory Applied to Scheduler Latency

### The Model

Model the CPU as an M/G/1 queue (Poisson arrivals, general service times, single server). This is a simplification but provides useful bounds.

Let:
- $\lambda$ = task arrival rate (wakeups per second)
- $\mathbb{E}[S]$ = mean service time (timeslice duration)
- $\mathbb{E}[S^2]$ = second moment of service time
- $\rho = \lambda \cdot \mathbb{E}[S]$ = **utilization** (must be $< 1$ for stability)

### Pollaczek-Khinchine Formula

The mean number of tasks in the system (queue + service):

$$\mathbb{E}[L] = \rho + \frac{\rho^2 + \lambda^2 \text{Var}[S]}{2(1 - \rho)}$$

The mean waiting time (time from wakeup to first getting the CPU):

$$\mathbb{E}[W] = \frac{\lambda \mathbb{E}[S^2]}{2(1 - \rho)}$$

### Practical Implications

As utilization $\rho \to 1$, the mean waiting time $\mathbb{E}[W] \to \infty$. This is the **queueing theory explanation** for why overloaded systems have catastrophically high latencies rather than graceful degradation.

For CFS, the service time distribution depends on the timeslice allocation. If all tasks have equal weight:

$$\mathbb{E}[S] = \frac{sched\_latency}{n}, \quad \text{Var}[S] \approx 0 \text{ (deterministic)}$$

Substituting into P-K for the deterministic service time case (M/D/1):

$$\mathbb{E}[W]_{M/D/1} = \frac{\rho}{2\lambda(1 - \rho)}$$

This is exactly **half** the M/M/1 waiting time, showing that CFS's near-deterministic timeslicing reduces latency variance compared to a random scheduler.

### Tail Latency

The P99 scheduling latency in an M/G/1 system scales as:

$$W_{99} \approx \mathbb{E}[W] \cdot \ln(100) \approx 4.6 \cdot \mathbb{E}[W]$$

At $\rho = 0.8$ with 50 tasks and $sched\_latency = 6\text{ms}$:

$$\mathbb{E}[S] = 0.12\text{ms}, \quad \lambda = \frac{\rho}{\mathbb{E}[S]} = 6667/\text{s}$$

$$\mathbb{E}[W] = \frac{0.8}{2 \cdot 6667 \cdot 0.2} \approx 0.3\text{ms}, \quad W_{99} \approx 1.4\text{ms}$$

---

## 5. CPU Cache Effects on Scheduling Decisions

### The Problem

Context switches and task migrations have costs invisible to the scheduler's fairness model: cache and TLB invalidation. How do these microarchitectural effects influence optimal scheduling policy?

### Cache Warmth Model

Define the **cache footprint** $F_i$ of task $i$ as the working set size in cache. After a context switch, the new task must reload its working set, incurring a **cold-start penalty**:

$$C_{\text{switch}} = C_{\text{ctx}} + F_i \cdot \frac{1}{B} \cdot (L_{\text{mem}} - L_{\text{cache}})$$

where:
- $C_{\text{ctx}}$ = register save/restore cost ($\approx 1\text{-}5\mu\text{s}$)
- $B$ = cache line size (typically 64 bytes)
- $L_{\text{mem}}$ = main memory latency ($\approx 60\text{-}100\text{ns}$)
- $L_{\text{cache}}$ = L1/L2 cache latency ($\approx 1\text{-}10\text{ns}$)

For a task with a 2MB L2 working set:

$$C_{\text{switch}} \approx 3\mu\text{s} + \frac{2 \times 10^6}{64} \cdot 80\text{ns} \approx 3\mu\text{s} + 2.5\text{ms}$$

This cache reload penalty dwarfs the context switch cost by three orders of magnitude.

### Scheduler Implications

1. **Minimum granularity matters.** Setting $sched\_min\_granularity$ too low causes frequent context switches where tasks never amortize cache warmup costs. The optimal granularity satisfies:

$$sched\_min\_granularity \gg C_{\text{switch}}$$

2. **Cache-affine scheduling.** CFS prefers to keep tasks on the same CPU (via `wake_affine` heuristics) to preserve cache warmth. The decision compares the **migration cost** vs. the **load imbalance cost**:

$$\text{migrate if } \quad \Delta_{\text{load}} \cdot \mathbb{E}[S] > C_{\text{migration}}$$

3. **LLC scheduling domains.** The kernel groups CPUs sharing a last-level cache into a scheduling domain. Load balancing within an LLC domain is cheaper (shared cache) than across domains (cold migration).

### NUMA Migration Cost

Cross-NUMA migration adds interconnect latency on top of cache effects:

$$C_{\text{NUMA}} = C_{\text{switch}} + F_i^{\text{mem}} \cdot \frac{L_{\text{remote}} - L_{\text{local}}}{B_{\text{interconnect}}}$$

where $L_{\text{remote}} / L_{\text{local}}$ is the **NUMA ratio** (typically 1.5x--3x). Auto-NUMA balancing (`kernel.numa_balancing`) mitigates this by migrating pages to follow task placement, but incurs its own overhead from page fault scanning.

The scanning rate is controlled by:

$$\text{scan period} \in [numa\_balancing\_scan\_delay, \; numa\_balancing\_scan\_period\_max]$$

Tuning these involves a tradeoff: aggressive scanning finds misplaced pages faster but adds overhead proportional to the total memory footprint.

---

## 6. Putting It Together -- Tuning Decision Framework

### Latency-Optimized Profile

For latency-sensitive workloads (trading systems, game servers, audio processing):

$$\text{Minimize: } \mathbb{E}[W] + C_{\text{switch}} \cdot \frac{sched\_latency}{sched\_min\_granularity}$$

The first term favors short scheduling periods; the second penalizes too many switches. The optimal $sched\_min\_granularity$ balances these:

$$sched\_min\_granularity^* \approx \sqrt{sched\_latency \cdot C_{\text{switch}}}$$

With $sched\_latency = 4\text{ms}$ and $C_{\text{switch}} = 50\mu\text{s}$:

$$sched\_min\_granularity^* \approx \sqrt{4 \times 10^{-3} \cdot 5 \times 10^{-5}} \approx 450\mu\text{s}$$

### Throughput-Optimized Profile

For batch workloads (compilation, rendering, ML training):

$$\text{Maximize: } 1 - n \cdot \frac{C_{\text{switch}}}{sched\_latency}$$

This represents the fraction of CPU time doing useful work. Longer scheduling periods and higher minimum granularity reduce context switch overhead.

---

## Prerequisites

- Operating system scheduling concepts (preemptive multitasking, timeslicing)
- Basic queuing theory (arrival rates, utilization, stability)
- CPU cache hierarchy (L1/L2/LLC, cache lines, TLB)
- NUMA architecture concepts
- Red-black tree properties ($O(\log n)$ operations)

## Complexity

| Operation | Time Complexity |
|-----------|----------------|
| CFS pick-next-task | $O(1)$ (cached leftmost) |
| CFS enqueue/dequeue | $O(\log n)$ (red-black tree) |
| Load balancing (per domain) | $O(n_{\text{cpus}} \cdot n_{\text{groups}})$ |
| SCHED_DEADLINE admission | $O(n_{\text{deadline\_tasks}})$ |
| NUMA balancing scan | $O(\text{pages} / \text{scan\_rate})$ per period |
