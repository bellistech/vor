# The Mathematics of OOM-Killer -- Score Calculations, Overcommit Ratios, and Memory Pressure

> *The OOM killer implements a scoring function that weighs memory consumption against*
> *administrative priority to select the optimal victim process. Behind this decision*
> *lies a constrained optimization problem: free the most memory with the least collateral damage.*

---

## 1. OOM Score Calculation (Scoring Function)

### The Problem

The kernel must select which process to kill when memory is exhausted. The OOM score determines the victim. How is it calculated?

### The Formula

The base OOM score (before adjustment) is proportional to the fraction of available memory consumed:

$$\text{oom\_score\_base} = \frac{\text{RSS} + \text{swap\_usage}}{\text{total\_available\_memory}} \times 1000$$

With adjustment applied:

$$\text{oom\_score} = \text{oom\_score\_base} + \text{oom\_score\_adj}$$

Clamped to valid range:

$$\text{oom\_score} = \max(0, \min(2000, \text{oom\_score}))$$

Special cases:
- $\text{oom\_score\_adj} = -1000 \implies \text{oom\_score} = 0$ (never kill)
- $\text{oom\_score\_adj} = 1000 \implies$ always among the first killed
- Root processes get a 3% discount: $\text{oom\_score\_base}_{\text{root}} = \text{oom\_score\_base} \times \frac{97}{100}$

### Worked Examples

System with 16 GB RAM + 4 GB swap = 20 GB available:

| Process | RSS | Swap | Base Score | Adj | Final Score |
|---------|-----|------|-----------|-----|------------|
| database | 8 GB | 0 | 400 | -500 | 0 (clamped) |
| web-server | 2 GB | 0 | 100 | 0 | 100 |
| cache | 4 GB | 1 GB | 250 | 0 | 250 |
| batch-job | 1 GB | 2 GB | 150 | 500 | 650 |
| log-agent | 512 MB | 0 | 25 | -200 | 0 (clamped) |
| sshd | 50 MB | 0 | 2 | -1000 | 0 |

**Victim selection**: batch-job (score 650) is killed first.

Memory freed by killing batch-job: $1 + 2 = 3$ GB.

### The Kernel's Select Function

When multiple processes have the same score, the kernel uses a tiebreaker:

$$\text{victim} = \arg\max_{p \in \text{processes}} \left( \text{oom\_score}(p), \; \text{RSS}(p) + \text{swap}(p) \right)$$

The process with the highest score wins. On ties, the one using more total memory is selected.

## 2. Overcommit Ratio Mathematics (Capacity Planning)

### The Problem

In overcommit mode 2, the kernel enforces a strict commit limit. How is this limit calculated, and when will `malloc()` fail?

### The Formula

$$\text{CommitLimit} = \frac{\text{overcommit\_ratio}}{100} \times \text{RAM} + \text{swap}$$

Alternative (using overcommit_kbytes):

$$\text{CommitLimit} = \text{overcommit\_kbytes} + \text{swap}$$

Allocation check for each `mmap()` / `brk()`:

$$\text{allow} = (\text{Committed\_AS} + \text{request}) \leq \text{CommitLimit}$$

### Worked Examples

System with 32 GB RAM:

| overcommit_ratio | Swap | CommitLimit | Effective RAM Multiplier |
|-----------------|------|------------|------------------------|
| 50 (default) | 0 | 16 GB | 0.50x |
| 50 | 8 GB | 24 GB | 0.75x |
| 80 | 8 GB | 33.6 GB | 1.05x |
| 100 | 8 GB | 40 GB | 1.25x |
| 100 | 0 | 32 GB | 1.00x |
| 150 | 0 | 48 GB | 1.50x |

Committed_AS growth scenario (32 GB RAM, ratio=80, 8 GB swap):

$$\text{CommitLimit} = 0.8 \times 32 + 8 = 33.6 \text{ GB}$$

| Application | Virtual Memory Requested | Committed_AS After | Status |
|------------|------------------------|-------------------|--------|
| Database | 16 GB | 16 GB | OK |
| Web server | 8 GB | 24 GB | OK |
| Cache | 6 GB | 30 GB | OK |
| Batch job | 2 GB | 32 GB | OK |
| New service | 4 GB | 36 GB | DENIED (> 33.6 GB) |

Note: Committed_AS counts virtual memory requests, not actual usage. Programs that `malloc()` large regions but use little see higher commit than RSS.

## 3. Memory Pressure Thresholds (Control Theory)

### The Problem

How do we define and measure the thresholds that predict imminent OOM events?

### The Formula

Memory utilization:

$$U = \frac{\text{MemTotal} - \text{MemAvailable}}{\text{MemTotal}} \times 100\%$$

Distance to OOM:

$$D_{\text{oom}} = \text{MemAvailable} + \text{SwapFree}$$

Time to OOM (if consumption rate is constant):

$$T_{\text{oom}} = \frac{D_{\text{oom}}}{\text{mem\_consumption\_rate}}$$

### Worked Examples

System with 16 GB RAM, 4 GB swap:

| State | MemAvailable | SwapFree | D_oom | Consumption Rate | T_oom |
|-------|-------------|----------|-------|-----------------|-------|
| Healthy | 10 GB | 4 GB | 14 GB | 100 MB/min | 143 min |
| Warning | 2 GB | 3 GB | 5 GB | 100 MB/min | 51 min |
| Critical | 500 MB | 1 GB | 1.5 GB | 100 MB/min | 15 min |
| Danger | 100 MB | 200 MB | 300 MB | 100 MB/min | 3 min |
| OOM imminent | 10 MB | 0 | 10 MB | 100 MB/min | 6 sec |

Alert thresholds:

| Severity | Condition | Action |
|----------|-----------|--------|
| Info | $U > 70\%$ | Log |
| Warning | $U > 85\%$ | Alert on-call |
| Critical | $U > 95\%$ | Scale up / shed load |
| Emergency | $D_{\text{oom}} < 256$ MB | earlyoom / manual intervention |

## 4. PSI Memory Stall Accounting (Time Series)

### The Problem

PSI (Pressure Stall Information) measures the fraction of time processes spend waiting for memory. How do we interpret these values for OOM prediction?

### The Formula

$$\text{PSI}_{\text{some}} = \frac{\Delta t_{\text{any\_task\_stalled}}}{\Delta t_{\text{wall}}} \times 100\%$$

$$\text{PSI}_{\text{full}} = \frac{\Delta t_{\text{all\_tasks\_stalled}}}{\Delta t_{\text{wall}}} \times 100\%$$

The kernel computes exponentially weighted moving averages:

$$\text{avg}_{10}(t) = \text{avg}_{10}(t-1) \times e^{-\Delta t / 10} + \text{PSI}(t) \times (1 - e^{-\Delta t / 10})$$

Similarly for avg60 and avg300 with time constants of 60 and 300 seconds.

### Worked Examples

Interpreting PSI memory values:

| some avg10 | full avg10 | Interpretation | Action |
|-----------|-----------|----------------|--------|
| 0.00 | 0.00 | No memory pressure | None |
| 5.00 | 0.00 | Light pressure, some reclaim | Monitor |
| 25.00 | 5.00 | Significant pressure | Investigate |
| 50.00 | 20.00 | Severe, throughput halved | Urgent: add memory |
| 80.00 | 60.00 | Near-OOM, system barely functional | Emergency |

PSI trigger setup for proactive OOM avoidance:

```
# /proc/pressure/memory poll trigger
# "some 150000 1000000" = trigger if any task stalled > 150ms in any 1s window
# "full 100000 1000000" = trigger if all tasks stalled > 100ms in any 1s window
```

## 5. Cgroup OOM Priority Hierarchy (Partial Ordering)

### The Problem

In a cgroup hierarchy, which cgroup's processes are killed first when the parent cgroup hits its memory limit?

### The Formula

Cgroup OOM priority is determined by protection levels:

$$\text{kill\_priority}(cg) = \frac{\text{usage}(cg) - \text{effective\_min}(cg)}{\text{usage}(cg)}$$

Higher kill_priority = killed first. Protected memory reduces the effective usage for scoring.

Effective protection in a hierarchy:

$$\text{effective\_min}(cg) = \min\left(\text{memory.min}(cg), \; \text{parent\_distributable} \times \frac{\text{memory.min}(cg)}{\sum_{\text{siblings}} \text{memory.min}(s)}\right)$$

### Worked Examples

Parent cgroup with 2 GB limit, three children:

| Child | memory.min | Usage | Kill Priority | Order |
|-------|-----------|-------|--------------|-------|
| critical-db | 1 GB | 1.2 GB | (1.2 - 1.0) / 1.2 = 0.17 | Last |
| web-app | 256 MB | 500 MB | (500 - 256) / 500 = 0.49 | Second |
| batch-job | 0 | 300 MB | (300 - 0) / 300 = 1.00 | First |

Kill order: batch-job, web-app, critical-db.

With `memory.oom.group=1` on batch-job's cgroup, ALL processes in that cgroup are killed simultaneously.

## 6. Victim Memory Liberation (Optimization)

### The Problem

The OOM killer aims to free the most memory with minimal process kills. How much memory is actually freed?

### The Formula

Memory freed by killing process $p$:

$$\text{freed}(p) = \text{RSS}_{\text{private}}(p) + \text{swap}(p) + \text{shmem}_{\text{last\_ref}}(p)$$

Shared pages are only freed if $p$ is the last reference holder:

$$\text{freed}_{\text{shared}}(p, \text{page}) = \begin{cases}
\text{page\_size} & \text{if sharers}(p, \text{page}) = 1 \\
0 & \text{if sharers}(p, \text{page}) > 1
\end{cases}$$

$$\text{freed}(p) = \sum_{\text{page} \in p} \text{freed}(p, \text{page})$$

### Worked Examples

Process with 2 GB RSS total:

| Memory Type | Size | Shared With | Actually Freed |
|------------|------|-------------|---------------|
| Private anonymous (heap) | 1.5 GB | None | 1.5 GB |
| Private file-backed | 200 MB | None | 200 MB |
| Shared libraries | 100 MB | 50 processes | ~2 MB (1/50) |
| Shared memory (IPC) | 200 MB | 3 processes | 0 (still referenced) |
| **Total RSS** | **2.0 GB** | | **~1.7 GB freed** |

Plus 500 MB swap usage freed = **2.2 GB total freed**.

The OOM killer logs report both RSS and swap freed:
```
Killed process 1234 (myapp) total-vm:8192000kB, anon-rss:1572864kB,
file-rss:204800kB, shmem-rss:204800kB, oom_score_adj:0
```

## 7. OOM Likelihood Over Time (Probabilistic Model)

### The Problem

Given memory growth patterns, what is the probability of an OOM event within a time window?

### The Formula

If memory consumption follows a random walk with drift $\mu$ (MB/hour) and volatility $\sigma$ (MB/hour):

$$P(\text{OOM before } T) = \Phi\left(\frac{\mu T - D_{\text{oom}}}{\sigma \sqrt{T}}\right)$$

where $\Phi$ is the standard normal CDF and $D_{\text{oom}}$ is the current distance to OOM.

### Worked Examples

System with $D_{\text{oom}} = 4$ GB, memory drift $\mu = 200$ MB/hr, $\sigma = 100$ MB/hr:

| Time Window | $\mu T$ | $\sigma\sqrt{T}$ | Z-score | P(OOM) |
|------------|---------|------------------|---------|--------|
| 1 hour | 200 MB | 100 MB | -38.0 | ~0% |
| 8 hours | 1.6 GB | 283 MB | -8.5 | ~0% |
| 16 hours | 3.2 GB | 400 MB | -2.0 | 2.3% |
| 20 hours | 4.0 GB | 447 MB | 0.0 | 50% |
| 24 hours | 4.8 GB | 490 MB | 1.6 | 94.5% |
| 30 hours | 6.0 GB | 548 MB | 3.6 | 99.98% |

Expected time to OOM:

$$E[T_{\text{oom}}] = \frac{D_{\text{oom}}}{\mu} = \frac{4096}{200} = 20.5 \text{ hours}$$

This model helps set monitoring alert thresholds: alert when $P(\text{OOM in 4 hours}) > 10\%$.

## Prerequisites

kernel-memory-management, process-scheduling, probability, cgroups, virtual-memory

## Complexity

| Operation | Time | Space |
|-----------|------|-------|
| OOM score calculation | O(1) per process | O(1) |
| OOM victim selection | O(processes) scan | O(1) |
| Kill and free memory | O(pages) for munmap | O(1) |
| Overcommit check (mode 2) | O(1) | O(1) counter |
| PSI calculation | O(1) amortized | O(1) per-CPU |
| Cgroup OOM traversal | O(cgroup tree depth) | O(depth) |
| earlyoom check cycle | O(processes) | O(1) |
| Memory pressure monitoring | O(1) per read | O(1) |
