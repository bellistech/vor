# The Mathematics of Benchmarking — Statistics, Confidence, and Performance Laws

> *Benchmarking is applied statistics. This covers the mathematical foundations that make benchmark results meaningful — variance analysis, confidence intervals, the Welch t-test used by benchstat, Amdahl's law for parallelism limits, and Little's law for throughput reasoning.*

---

## 1. Variance and Measurement Noise (Descriptive Statistics)

### The Problem

A benchmark that reports "45.2 ns/op" is useless without understanding the variance. How do we characterize measurement stability and identify noise sources?

### The Formula

For $N$ benchmark samples $x_1, x_2, \ldots, x_N$:

**Sample mean**: $\bar{x} = \frac{1}{N}\sum_{i=1}^{N} x_i$

**Sample variance**: $s^2 = \frac{1}{N-1}\sum_{i=1}^{N}(x_i - \bar{x})^2$

**Coefficient of variation** (relative variability):

$$CV = \frac{s}{\bar{x}} \times 100\%$$

benchstat reports this as the $\pm$ percentage. A $CV > 5\%$ suggests noisy measurements.

**Common noise sources and their distribution signatures**:

| Source | Distribution | Typical $CV$ impact |
|--------|-------------|-------------------|
| GC pauses | Right-skewed spikes | 5-50% |
| CPU frequency scaling | Bimodal | 10-30% |
| Context switches | Right-skewed | 2-10% |
| Cache effects | Bimodal (cold/warm) | 5-20% |
| Thermal throttling | Monotone drift | 5-15% |

### Worked Examples

**Example**: 10 benchmark runs (ns/op): 42, 43, 41, 44, 43, 45, 42, 43, 44, 41.

$$\bar{x} = \frac{428}{10} = 42.8 \text{ ns/op}$$

$$s^2 = \frac{(42-42.8)^2 + (43-42.8)^2 + \cdots + (41-42.8)^2}{9} = \frac{14.16}{9} = 1.573$$

$$s = 1.254 \text{ ns/op}$$

$$CV = \frac{1.254}{42.8} \times 100\% = 2.93\%$$

benchstat would report: `42.8ns ± 3%` — a clean measurement.

## 2. Confidence Intervals and the Welch t-Test (Inferential Statistics)

### The Problem

Given two sets of benchmark measurements (before and after an optimization), how do we determine whether the difference is statistically significant? benchstat uses the Welch t-test.

### The Formula

**Confidence interval** for the mean (with $t$-distribution, $\nu = N-1$ degrees of freedom):

$$\bar{x} \pm t_{\alpha/2, \nu} \cdot \frac{s}{\sqrt{N}}$$

**Welch's t-test** for comparing two means $\bar{x}_1$, $\bar{x}_2$ with unequal variances:

$$t = \frac{\bar{x}_1 - \bar{x}_2}{\sqrt{\frac{s_1^2}{N_1} + \frac{s_2^2}{N_2}}}$$

**Welch-Satterthwaite degrees of freedom**:

$$\nu = \frac{\left(\frac{s_1^2}{N_1} + \frac{s_2^2}{N_2}\right)^2}{\frac{(s_1^2/N_1)^2}{N_1 - 1} + \frac{(s_2^2/N_2)^2}{N_2 - 1}}$$

benchstat rejects the null hypothesis ($H_0$: no difference) when $p < 0.05$ and reports the percentage change. If $p \geq 0.05$, it reports `~` (no significant change).

### Worked Examples

**Example**: Before optimization (10 runs): mean = 45.2 ns, $s_1 = 1.5$ ns. After optimization (10 runs): mean = 38.7 ns, $s_2 = 0.8$ ns.

$$t = \frac{45.2 - 38.7}{\sqrt{\frac{1.5^2}{10} + \frac{0.8^2}{10}}} = \frac{6.5}{\sqrt{0.225 + 0.064}} = \frac{6.5}{\sqrt{0.289}} = \frac{6.5}{0.5376} = 12.09$$

$$\nu = \frac{(0.225 + 0.064)^2}{\frac{0.225^2}{9} + \frac{0.064^2}{9}} = \frac{0.0835}{\frac{0.0506 + 0.0041}{9}} = \frac{0.0835}{0.00608} = 13.7 \approx 13$$

For $\nu = 13$ and $t = 12.09$: $p \ll 0.001$. The improvement is highly significant.

Percentage change: $\frac{38.7 - 45.2}{45.2} = -14.38\%$.

benchstat output: `-14.38% (p=0.000 n=10+10)`

**Minimum sample size**: benchstat requires at least $N = 5$ per group. With fewer, the confidence interval is too wide and most comparisons show `~`.

## 3. Amdahl's Law (Parallel Performance Bounds)

### The Problem

When benchmarking parallel code (like `b.RunParallel`), what is the theoretical maximum speedup? Amdahl's law sets the upper bound.

### The Formula

If fraction $f$ of a computation is parallelizable and we use $P$ processors:

$$S(P) = \frac{1}{(1 - f) + \frac{f}{P}}$$

**Maximum speedup** (as $P \to \infty$):

$$S_{max} = \frac{1}{1 - f}$$

**Gustafson's law** (alternative for scaled workloads):

$$S_G(P) = P - (1 - f)(P - 1) = 1 + f(P - 1)$$

Gustafson assumes the problem size grows with $P$, while Amdahl assumes fixed problem size.

### Worked Examples

**Example**: A request handler spends 80% of time in parallelizable database queries ($f = 0.8$) and 20% in serial JSON marshaling.

With 4 cores:
$$S(4) = \frac{1}{0.2 + \frac{0.8}{4}} = \frac{1}{0.2 + 0.2} = \frac{1}{0.4} = 2.5\times$$

With 8 cores:
$$S(8) = \frac{1}{0.2 + 0.1} = \frac{1}{0.3} = 3.33\times$$

Maximum speedup:
$$S_{max} = \frac{1}{0.2} = 5\times$$

Even with infinite cores, you cannot exceed 5x speedup. The serial 20% is the bottleneck.

**Benchmark verification**: Run `b.RunParallel` with `GOMAXPROCS=1,2,4,8` and compare actual vs Amdahl prediction:

```bash
GOMAXPROCS=1 go test -bench=BenchmarkParallel -count=10 > p1.txt
GOMAXPROCS=4 go test -bench=BenchmarkParallel -count=10 > p4.txt
benchstat p1.txt p4.txt
```

## 4. Little's Law (Throughput and Latency)

### The Problem

For server benchmarks, throughput and latency are related. Little's law provides the fundamental relationship.

### The Formula

$$L = \lambda \cdot W$$

Where:
- $L$ = average number of items in the system (concurrency)
- $\lambda$ = average arrival rate (throughput, requests/second)
- $W$ = average time in the system (latency)

**Derived forms**:

$$\lambda = \frac{L}{W} \qquad W = \frac{L}{\lambda}$$

This holds for **any** stable system regardless of arrival distribution, service distribution, or scheduling discipline.

**USL (Universal Scalability Law)** extends this for contention and coherence:

$$C(P) = \frac{P}{1 + \alpha(P - 1) + \beta P(P - 1)}$$

Where $\alpha$ is the contention coefficient and $\beta$ is the coherence (crosstalk) coefficient.

### Worked Examples

**Example**: An HTTP server benchmark shows 1000 req/s throughput with average latency of 50ms.

$$L = 1000 \times 0.050 = 50 \text{ concurrent requests}$$

If we increase concurrency to 100 (double) without changing the server:

$$W = \frac{100}{1000} = 100 \text{ ms}$$

Latency doubles. To maintain 50ms latency at 100 concurrency, we need:

$$\lambda = \frac{100}{0.050} = 2000 \text{ req/s}$$

We need to double throughput (e.g., by doubling server instances).

## 5. Measurement Methodology (Experimental Design)

### The Problem

How many samples do we need, and how should we structure the experiment to get reliable results?

### The Formula

**Required sample size** to detect a $d\%$ change with power $1-\beta$ and significance $\alpha$:

$$N \geq 2\left(\frac{z_{\alpha/2} + z_\beta}{d/\sigma}\right)^2$$

For $\alpha = 0.05$ (95% confidence) and $\beta = 0.2$ (80% power), with $z_{0.025} = 1.96$ and $z_{0.2} = 0.842$:

$$N \geq 2\left(\frac{1.96 + 0.842}{d/\sigma}\right)^2 = 2\left(\frac{2.802 \cdot \sigma}{d}\right)^2$$

### Worked Examples

**Example**: Detect a 5% improvement in a benchmark with $CV = 3\%$ ($\sigma/\mu = 0.03$).

The effect size relative to $\sigma$: $d/\sigma = 0.05/0.03 = 1.667$.

$$N \geq 2\left(\frac{2.802}{1.667}\right)^2 = 2 \times 2.826 = 5.65 \approx 6$$

6 runs per group suffices. But for a 2% improvement:

$$d/\sigma = 0.02/0.03 = 0.667$$

$$N \geq 2\left(\frac{2.802}{0.667}\right)^2 = 2 \times 17.66 = 35.3 \approx 36$$

Need 36 runs per group. This explains why benchstat sometimes shows `~` with `-count=10`.

## Prerequisites

- Descriptive statistics (mean, variance, standard deviation)
- Hypothesis testing (t-test, p-value, significance level)
- Basic queueing theory (arrival rate, service time)
- Parallel computing concepts (speedup, Amdahl's law)

## Complexity

| Computation | Time Complexity | Space Complexity |
|-------------|----------------|-----------------|
| Mean and variance | $O(N)$ | $O(1)$ |
| Welch t-test | $O(N_1 + N_2)$ | $O(1)$ |
| benchstat full analysis | $O(B \cdot N)$ | $O(B \cdot N)$ |
| USL curve fitting | $O(P \cdot I)$ iterations | $O(P)$ |

Where: $N$ = sample count, $B$ = benchmark count, $P$ = parallelism levels tested, $I$ = fitting iterations.
