# The Mathematics of Pyroscope — Continuous Profiling Overhead and Storage

> *Continuous profiling samples stack traces at fixed intervals. The math covers sampling theory, overhead bounds, storage sizing, profile aggregation, and the statistical significance of hot functions.*

---

## 1. Sampling Theory (CPU Profiling)

### The Problem

CPU profilers interrupt the program at regular intervals to capture stack traces. Too frequent and overhead dominates; too infrequent and rare functions are missed.

### The Formula

With sampling frequency $f$ Hz and observation window $T$ seconds:

$$N = f \times T$$

The probability of observing a function that consumes fraction $p$ of CPU time in at least one sample:

$$P(\text{observed}) = 1 - (1-p)^N$$

Minimum samples needed to observe a function at confidence level $\alpha$:

$$N_{\min} = \frac{\ln(1-\alpha)}{\ln(1-p)}$$

### Worked Examples

| CPU Fraction ($p$) | Frequency ($f$) | Window ($T$) | Samples ($N$) | P(observed) |
|:---:|:---:|:---:|:---:|:---:|
| 0.01 (1%) | 100 Hz | 10s | 1,000 | 99.996% |
| 0.01 (1%) | 100 Hz | 1s | 100 | 63.4% |
| 0.001 (0.1%) | 100 Hz | 10s | 1,000 | 63.2% |
| 0.001 (0.1%) | 100 Hz | 60s | 6,000 | 99.75% |
| 0.05 (5%) | 100 Hz | 10s | 1,000 | ~100% |

---

## 2. Profiling Overhead

### The Problem

Each sample interrupts the program to walk the stack and record the trace. Total overhead depends on sample rate, stack depth, and recording cost.

### The Formula

$$\text{Overhead} = f \times T_{\text{sample}} \times (1 + \epsilon_{\text{cache}})$$

where $T_{\text{sample}}$ is the time per sample (stack walk + storage) and $\epsilon_{\text{cache}}$ is the cache pollution factor.

For Go's pprof (100 Hz default):

$$T_{\text{sample}} \approx 1\text{-}5 \mu s$$

$$\text{Overhead} = 100 \times 3\mu s = 300\mu s/s = 0.03\%$$

For Java async-profiler with allocation profiling at interval $I$ bytes:

$$\text{Alloc Overhead} = \frac{A_{\text{rate}}}{I} \times T_{\text{sample}}$$

where $A_{\text{rate}}$ is allocation rate in bytes/sec.

### Worked Examples

| Profiler | Sample Rate | $T_{\text{sample}}$ | CPU Overhead |
|:---:|:---:|:---:|:---:|
| Go pprof | 100 Hz | 3 us | 0.03% |
| async-profiler (CPU) | 100 Hz | 5 us | 0.05% |
| py-spy | 100 Hz | 10 us | 0.10% |
| eBPF (perf events) | 99 Hz | 2 us | 0.02% |
| Go pprof | 1000 Hz | 3 us | 0.30% |

---

## 3. Storage Sizing

### The Problem

Each profile is a set of stack traces with sample counts. Storage grows with the number of services, profile types, and retention.

### The Formula

Per profile (compressed):

$$B_{\text{profile}} = U \times \bar{d} \times \bar{f}_{\text{len}} \times C_{\text{ratio}}$$

where $U$ is unique stack traces per profile, $\bar{d}$ is average stack depth, $\bar{f}_{\text{len}}$ is average function name length, and $C_{\text{ratio}}$ is compression ratio.

Daily storage per service:

$$\text{Daily} = \frac{86400}{\Delta t} \times P \times B_{\text{profile}}$$

where $\Delta t$ is the scrape/push interval and $P$ is the number of profile types.

### Worked Examples

| Services | Profile Types ($P$) | Interval ($\Delta t$) | Avg Profile Size | Daily (GB) | 30-day (GB) |
|:---:|:---:|:---:|:---:|:---:|:---:|
| 10 | 2 (CPU+mem) | 15s | 50 KB | 5.6 | 168 |
| 50 | 2 | 15s | 50 KB | 28.0 | 840 |
| 50 | 5 | 15s | 40 KB | 56.0 | 1,680 |
| 100 | 2 | 60s | 80 KB | 23.0 | 691 |

---

## 4. Profile Aggregation (Merging Flame Graphs)

### The Problem

Querying profiles over a time range requires merging $N$ individual profiles. The merged result must preserve relative sample counts.

### The Formula

For $N$ profiles, each a mapping from stack trace $s$ to sample count $c_i(s)$:

$$C_{\text{merged}}(s) = \sum_{i=1}^{N} c_i(s)$$

Self time (exclusive) for function $f$:

$$\text{Self}(f) = \sum_{s: \text{top}(s)=f} C_{\text{merged}}(s)$$

Total time (inclusive) for function $f$:

$$\text{Total}(f) = \sum_{s: f \in s} C_{\text{merged}}(s)$$

Percentage of total:

$$\%_f = \frac{\text{Total}(f)}{\sum_{s} C_{\text{merged}}(s)} \times 100$$

### Worked Examples

Given 3 profiles with stacks `main->A->B` and `main->A->C`:

| Stack | Profile 1 | Profile 2 | Profile 3 | Merged |
|:---:|:---:|:---:|:---:|:---:|
| main->A->B | 40 | 35 | 45 | 120 |
| main->A->C | 60 | 65 | 55 | 180 |
| **Total** | **100** | **100** | **100** | **300** |

- Self(B) = 120 (40%), Self(C) = 180 (60%)
- Total(A) = 300 (100%), Self(A) = 0 (0%)

---

## 5. Statistical Significance of Hot Functions

### The Problem

Is a function truly hot, or is the sample count just noise? With $N$ total samples and a function appearing $k$ times, determine confidence.

### The Formula

Under the null hypothesis that the function consumes fraction $p_0$ of CPU, the sample count $k$ follows a binomial distribution:

$$k \sim \text{Binomial}(N, p_0)$$

95% confidence interval for the true fraction:

$$\hat{p} \pm 1.96 \sqrt{\frac{\hat{p}(1-\hat{p})}{N}}$$

where $\hat{p} = k/N$.

Minimum samples for a meaningful result with margin of error $\epsilon$:

$$N_{\min} = \frac{1.96^2 \cdot p(1-p)}{\epsilon^2}$$

### Worked Examples

| Total Samples ($N$) | Function Samples ($k$) | $\hat{p}$ | 95% CI |
|:---:|:---:|:---:|:---:|
| 1,000 | 50 | 5.0% | [3.6%, 6.4%] |
| 1,000 | 10 | 1.0% | [0.4%, 1.6%] |
| 10,000 | 500 | 5.0% | [4.6%, 5.4%] |
| 10,000 | 100 | 1.0% | [0.8%, 1.2%] |
| 100 | 5 | 5.0% | [0.7%, 9.3%] |

With only 100 samples, a 5% function has a wide CI -- the result is unreliable.

---

## 6. Differential Profiling

### The Problem

Compare two profiles (e.g., before and after a deploy) to find functions that got slower or faster.

### The Formula

For function $f$ in baseline profile $A$ (total $N_A$ samples) and comparison profile $B$ (total $N_B$ samples):

$$\Delta_f = \frac{k_B}{N_B} - \frac{k_A}{N_A}$$

Z-test for significance:

$$z = \frac{\hat{p}_B - \hat{p}_A}{\sqrt{\hat{p}(1-\hat{p})\left(\frac{1}{N_A} + \frac{1}{N_B}\right)}}$$

where $\hat{p} = \frac{k_A + k_B}{N_A + N_B}$ is the pooled proportion.

### Worked Examples

| Function | Baseline ($k_A/N_A$) | New ($k_B/N_B$) | $\Delta$ | $z$-score | Significant? |
|:---:|:---:|:---:|:---:|:---:|:---:|
| parse_json | 500/10000 (5%) | 800/10000 (8%) | +3% | 8.7 | Yes |
| db_query | 200/10000 (2%) | 210/10000 (2.1%) | +0.1% | 0.5 | No |
| gc_sweep | 100/10000 (1%) | 50/10000 (0.5%) | -0.5% | 3.8 | Yes |

---

## Prerequisites

- Probability and statistics (binomial distribution, confidence intervals, hypothesis testing)
- Computer architecture (instruction sampling, hardware performance counters)
- Call stack mechanics (frame pointers, DWARF unwinding, async-safe signals)
- Compression algorithms (dictionary coding for function names, trie-based deduplication)
- Go runtime internals (goroutine scheduling, pprof signal handling)
