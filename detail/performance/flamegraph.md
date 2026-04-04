# The Mathematics of Flame Graphs — Sampling Theory and Stack Trace Analysis

> *Flame graphs transform sampled stack traces into a visual hierarchy. The math covers sampling accuracy, stack collapse algorithms, differential statistics, and the relationship between sample count and confidence.*

---

## 1. Sampling and Accuracy

### The Problem

A profiler samples the call stack $N$ times. Each sample captures the currently executing function. How many samples are needed to identify hot functions with a given accuracy?

### The Formula

If function $f$ consumes fraction $p$ of total execution time, the expected number of samples hitting $f$ is:

$$E[k] = N \cdot p$$

Standard deviation:

$$\sigma_k = \sqrt{N \cdot p \cdot (1-p)}$$

Relative error (coefficient of variation):

$$\text{CV} = \frac{\sigma_k}{E[k]} = \sqrt{\frac{1-p}{N \cdot p}}$$

For target relative error $\epsilon$:

$$N_{\min} = \frac{1-p}{p \cdot \epsilon^2}$$

### Worked Examples

| Target Function ($p$) | Desired Error ($\epsilon$) | Samples Needed ($N_{\min}$) | At 99 Hz (seconds) |
|:---:|:---:|:---:|:---:|
| 50% | 5% | 400 | 4s |
| 10% | 5% | 3,600 | 36s |
| 1% | 10% | 9,900 | 100s |
| 1% | 5% | 39,600 | 400s |
| 0.1% | 10% | 99,900 | 1,010s |

---

## 2. Stack Collapse Algorithm

### The Problem

Raw profiler output contains one stack trace per sample, with frames ordered from leaf to root. The collapse step merges identical stacks and counts occurrences.

### The Formula

Given $N$ samples producing stack traces $s_1, s_2, \ldots, s_N$, the collapsed output is a multiset:

$$C = \{ (s, c(s)) \mid s \in \text{unique}(s_1, \ldots, s_N), \; c(s) = |\{i : s_i = s\}| \}$$

Number of unique stacks $U$:

$$U = |\text{unique}(s_1, \ldots, s_N)| \leq N$$

In practice, $U \ll N$ because programs execute the same code paths repeatedly.

Compression ratio:

$$R_{\text{compress}} = \frac{N}{U}$$

### Worked Examples

| Samples ($N$) | Unique Stacks ($U$) | Compression ($R$) | Typical Application |
|:---:|:---:|:---:|:---:|
| 3,000 | 150 | 20x | Simple HTTP server |
| 30,000 | 800 | 37x | Microservice |
| 100,000 | 2,500 | 40x | Monolith |
| 1,000,000 | 5,000 | 200x | Long-running daemon |

---

## 3. Flame Graph Geometry

### The Problem

Each frame in the SVG has a width proportional to its inclusive sample count. Calculate frame widths and the minimum detectable function.

### The Formula

For a flame graph of pixel width $W$ and total samples $N$, a function with $k$ inclusive samples has width:

$$w_f = W \times \frac{k}{N} \text{ pixels}$$

Minimum visible function (at least 1 pixel wide):

$$p_{\min} = \frac{1}{W} = \frac{N_{\min}}{N}$$

$$k_{\min} = \left\lceil \frac{N}{W} \right\rceil$$

With `--minwidth=0.1` (hide frames below 0.1%):

$$k_{\text{threshold}} = \left\lceil N \times 0.001 \right\rceil$$

### Worked Examples

| SVG Width ($W$) | Total Samples ($N$) | Min Visible ($p_{\min}$) | $k_{\min}$ | At 0.1% threshold |
|:---:|:---:|:---:|:---:|:---:|
| 1,200 px | 10,000 | 0.083% | 9 | 10 |
| 1,200 px | 100,000 | 0.083% | 84 | 100 |
| 1,200 px | 1,000 | 0.083% | 1 | 1 |
| 800 px | 10,000 | 0.125% | 13 | 10 |

---

## 4. Differential Flame Graphs

### The Problem

Given two collapsed profiles (before and after), quantify which functions got slower or faster and whether the difference is statistically significant.

### The Formula

For function $f$ in baseline (total $N_A$ samples, $k_A$ hits) and comparison (total $N_B$ samples, $k_B$ hits):

Normalized difference:

$$\Delta_f = \frac{k_B}{N_B} - \frac{k_A}{N_A}$$

The `difffolded.pl` script computes per-stack deltas:

$$\delta(s) = \frac{c_B(s)}{N_B} - \frac{c_A(s)}{N_A}$$

Color mapping in the SVG:

$$\text{color}(s) = \begin{cases} \text{red (intensity} \propto \delta) & \text{if } \delta > 0 \text{ (regression)} \\ \text{blue (intensity} \propto |\delta|) & \text{if } \delta < 0 \text{ (improvement)} \\ \text{white} & \text{if } \delta = 0 \end{cases}$$

Chi-squared test for significance:

$$\chi^2 = \sum_{f} \frac{(O_f - E_f)^2}{E_f}$$

where $O_f = k_B$ and $E_f = N_B \cdot (k_A / N_A)$.

### Worked Examples

| Function | Before ($k_A/N_A$) | After ($k_B/N_B$) | $\Delta$ | Color |
|:---:|:---:|:---:|:---:|:---:|
| json_parse | 800/10000 (8%) | 1200/10000 (12%) | +4% | Red |
| db_connect | 500/10000 (5%) | 200/10000 (2%) | -3% | Blue |
| gc_mark | 300/10000 (3%) | 310/10000 (3.1%) | +0.1% | White |

---

## 5. Off-CPU vs On-CPU Time Decomposition

### The Problem

A thread's wall-clock time splits into on-CPU (executing) and off-CPU (blocked/sleeping). Flame graphs can visualize each independently.

### The Formula

$$T_{\text{wall}} = T_{\text{on-CPU}} + T_{\text{off-CPU}}$$

From perf scheduler events:

$$T_{\text{off-CPU}} = \sum_{i} (t_{\text{wake}_i} - t_{\text{sleep}_i})$$

Off-CPU percentage:

$$\%_{\text{off}} = \frac{T_{\text{off-CPU}}}{T_{\text{wall}}} \times 100$$

For I/O-bound workloads, $\%_{\text{off}} > 90\%$, meaning CPU flame graphs reveal less than 10% of where time is spent.

### Worked Examples

| Workload Type | $T_{\text{on-CPU}}$ | $T_{\text{off-CPU}}$ | $\%_{\text{off}}$ | CPU Flame Graph Reveals |
|:---:|:---:|:---:|:---:|:---:|
| CPU-bound (compute) | 95% | 5% | 5% | 95% of bottleneck |
| Mixed (web server) | 40% | 60% | 60% | 40% of bottleneck |
| I/O-bound (database) | 5% | 95% | 95% | 5% of bottleneck |
| Lock-heavy (contention) | 20% | 80% | 80% | 20% of bottleneck |

---

## 6. Safepoint Bias (Java/JVM)

### The Problem

JVM-based profilers that use `GetStackTrace` can only sample at safepoints, biasing results toward functions that reach safepoints frequently. async-profiler avoids this.

### The Formula

Let $f_{\text{true}}(x)$ be the true CPU fraction for function $x$ and $f_{\text{biased}}(x)$ the measured fraction:

$$f_{\text{biased}}(x) = f_{\text{true}}(x) \times \frac{r_{\text{sp}}(x)}{\bar{r}_{\text{sp}}}$$

where $r_{\text{sp}}(x)$ is the safepoint rate in function $x$ and $\bar{r}_{\text{sp}}$ is the average safepoint rate.

Bias factor:

$$B(x) = \frac{r_{\text{sp}}(x)}{\bar{r}_{\text{sp}}}$$

Functions with tight loops and no safepoints have $B(x) \to 0$ (invisible to the profiler), while functions with frequent method calls have $B(x) > 1$ (over-represented).

### Worked Examples

| Function | True CPU% | Safepoint Rate | Bias Factor | Measured CPU% |
|:---:|:---:|:---:|:---:|:---:|
| tight_loop (no calls) | 30% | 0.1x | 0.1 | 3% |
| process_request | 20% | 1.5x | 1.5 | 30% |
| parse_json | 15% | 1.0x | 1.0 | 15% |
| gc_collect | 10% | 2.0x | 2.0 | 20% |

async-profiler uses `AsyncGetCallTrace` + perf events, eliminating safepoint bias entirely.

---

## Prerequisites

- Operating system scheduling (context switches, voluntary vs involuntary preemption)
- Statistics (binomial distribution, confidence intervals, chi-squared test)
- Call stack mechanics (frame pointers, return addresses, DWARF CFI)
- Hardware performance counters (PMU, perf_event_open, sampling interrupts)
- SVG rendering (XML structure, coordinate systems, interactive JavaScript)
