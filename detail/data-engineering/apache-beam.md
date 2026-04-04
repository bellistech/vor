# The Mathematics of Apache Beam — Windowing, Watermarks, and Distributed Aggregation

> *Apache Beam unifies batch and streaming through a model of event-time windows, watermarks, and triggers. The math covers window assignment, watermark propagation, combiner lifting, and session window merging.*

---

## 1. Window Assignment (Temporal Partitioning)

### The Problem

Every element must be assigned to one or more windows based on its event timestamp. The window function determines which window(s) an element belongs to.

### The Formula

For fixed (tumbling) windows of duration $w$ with offset $o$:

$$\text{window}(t) = \left[ \left\lfloor \frac{t - o}{w} \right\rfloor \cdot w + o, \quad \left\lfloor \frac{t - o}{w} \right\rfloor \cdot w + o + w \right)$$

For sliding windows of duration $w$ with slide period $s$:

$$\text{windows}(t) = \left\{ \left[ k \cdot s + o, \; k \cdot s + o + w \right) : k = \left\lfloor \frac{t - o - w}{s} \right\rfloor + 1 \ldots \left\lfloor \frac{t - o}{s} \right\rfloor \right\}$$

The number of windows an element belongs to:

$$n_{\text{windows}} = \left\lceil \frac{w}{s} \right\rceil$$

### Worked Examples

Fixed windows, $w = 60$s, element at $t = 145$s:

$$\text{window}(145) = [\lfloor 145/60 \rfloor \times 60, \lfloor 145/60 \rfloor \times 60 + 60) = [120, 180)$$

Sliding windows, $w = 300$s, $s = 60$s, element at $t = 145$s:

$$n_{\text{windows}} = \lceil 300/60 \rceil = 5$$

| Window Type | Duration | Slide | Element at t=145s | Windows Assigned |
|:---:|:---:|:---:|:---:|:---:|
| Fixed | 60s | N/A | [120, 180) | 1 |
| Sliding | 300s | 60s | [0,300), [60,360), [120,420) ... | 5 |
| Sliding | 600s | 60s | 10 windows | 10 |
| Session | gap=120s | N/A | Depends on neighbors | 1 (merged) |

## 2. Watermark Propagation (Event-Time Progress)

### The Problem

The watermark $W(t)$ is the system's estimate that all events with timestamp $\leq W(t)$ have arrived. It determines when windows can be closed and results emitted.

### The Formula

For a source with maximum event-time skew (lag) $\delta$:

$$W(t) = t_{\text{processing}} - \delta$$

For a pipeline stage that buffers events with max buffer time $b$:

$$W_{\text{output}}(t) = W_{\text{input}}(t) - b$$

For a merge of $k$ inputs:

$$W_{\text{merged}}(t) = \min_{i=1}^{k} W_i(t)$$

A window $[s, e)$ is complete when:

$$W(t) \geq e + \text{allowed\_lateness}$$

### Worked Examples

Source with 30s skew, processing time = 1000s:

$$W(1000) = 1000 - 30 = 970 \text{ (event time)}$$

| Pipeline Stage | Input Watermark | Buffering | Output Watermark |
|:---:|:---:|:---:|:---:|
| Source | N/A (clock - 30s) | 0s | 970s |
| GroupByKey | 970s | 5s | 965s |
| Window merge | min(965, 960) | 0s | 960s |
| Final output | 960s | 0s | 960s |

Window [900, 960) closes when $W(t) \geq 960$: at processing time $t = 990$s.

## 3. Combiner Lifting (Distributed Aggregation)

### The Problem

GroupByKey requires shuffling all values to a single worker per key. If the aggregation is associative and commutative, partial combining (lifting) can be performed before the shuffle, reducing network transfer.

### The Formula

Without combiner lifting, shuffle data volume for key $k$ with $n_k$ values of size $s$:

$$D_{\text{shuffle}} = \sum_{k} n_k \cdot s$$

With combiner lifting, each of $W$ workers produces one partial aggregate per key:

$$D_{\text{lifted}} = \sum_{k} \min(n_k, W) \cdot s_{\text{acc}}$$

where $s_{\text{acc}}$ is the accumulator size (often constant, e.g., 16 bytes for sum/count).

Reduction factor:

$$R = \frac{D_{\text{shuffle}}}{D_{\text{lifted}}} = \frac{\sum_k n_k \cdot s}{\sum_k \min(n_k, W) \cdot s_{\text{acc}}} \approx \frac{\bar{n} \cdot s}{s_{\text{acc}}}$$

### Worked Examples

1M events, 10K distinct keys, 200 bytes per value, 16-byte accumulator, 100 workers:

Without lifting: $1{,}000{,}000 \times 200 = 200$ MB shuffled.

With lifting: $10{,}000 \times 100 \times 16 = 16$ MB shuffled.

$$R = \frac{200 \text{ MB}}{16 \text{ MB}} = 12.5\times \text{ reduction}$$

| Values | Keys | Workers | Without Lifting | With Lifting | Reduction |
|:---:|:---:|:---:|:---:|:---:|:---:|
| 1M | 10K | 10 | 200 MB | 1.6 MB | 125x |
| 1M | 10K | 100 | 200 MB | 16 MB | 12.5x |
| 1M | 1M | 100 | 200 MB | 200 MB | 1x (no benefit) |
| 10M | 1K | 50 | 2 GB | 0.8 MB | 2500x |

Key insight: combiner lifting helps most when there are many values per key (high $\bar{n}$).

## 4. Session Window Merging (Interval Graph Theory)

### The Problem

Session windows are defined by a gap duration $g$. When two elements are within $g$ of each other, their sessions merge. The merge operation is transitive: a chain of nearby events forms one session.

### The Formula

Given $n$ events sorted by timestamp $t_1 \leq t_2 \leq \ldots \leq t_n$, sessions are formed by merging consecutive events where:

$$t_{i+1} - t_i \leq g$$

The number of sessions:

$$S = 1 + \sum_{i=1}^{n-1} \mathbb{1}[t_{i+1} - t_i > g]$$

Expected session count for $n$ events uniformly distributed in $[0, T]$:

$$E[S] = n \cdot \left(1 - \frac{g}{T}\right)^{n-1} + 1 \approx n \cdot e^{-ng/T} + 1$$

Expected session length (for Poisson arrivals with rate $\lambda$):

$$E[L_{\text{session}}] = \frac{e^{\lambda g} - 1}{\lambda}$$

### Worked Examples

100 events in a 1-hour window ($T = 3600$s), gap $g = 120$s:

$$E[S] = 100 \cdot \left(1 - \frac{120}{3600}\right)^{99} + 1 = 100 \cdot 0.967^{99} + 1 \approx 100 \cdot 0.035 + 1 = 4.5$$

| Events ($n$) | Window ($T$) | Gap ($g$) | Expected Sessions |
|:---:|:---:|:---:|:---:|
| 10 | 3600s | 120s | 7.2 |
| 50 | 3600s | 120s | 3.2 |
| 100 | 3600s | 120s | 4.5 |
| 100 | 3600s | 30s | 38.5 |
| 1000 | 3600s | 120s | 1.0 (one big session) |

## 5. Late Data and Allowed Lateness (Completeness Theory)

### The Problem

In streaming, data arrives out of order. The allowed lateness parameter controls how long windows stay open after the watermark passes their end time.

### The Formula

The probability of an element being "late" (arriving after watermark passes its window end):

$$P(\text{late}) = P(d > \delta) = 1 - F_D(\delta)$$

where $d$ is the actual delay and $F_D$ is the delay CDF, and $\delta$ is the watermark skew.

With allowed lateness $L$, the probability of data being dropped:

$$P(\text{dropped}) = P(d > \delta + L) = 1 - F_D(\delta + L)$$

For exponentially distributed delays with mean $\mu$:

$$P(\text{dropped}) = e^{-(\delta + L)/\mu}$$

### Worked Examples

Delays exponentially distributed with mean $\mu = 10$s, watermark skew $\delta = 30$s:

| Allowed Lateness ($L$) | P(dropped) | Data Completeness |
|:---:|:---:|:---:|
| 0s | $e^{-3} = 4.98\%$ | 95.02% |
| 30s | $e^{-6} = 0.25\%$ | 99.75% |
| 60s | $e^{-9} = 0.012\%$ | 99.988% |
| 120s | $e^{-15} \approx 0\%$ | ~100% |

For 99.9% completeness: $L \geq \mu \cdot \ln(1000) - \delta = 10 \times 6.9 - 30 = 39$s.

## Prerequisites

- Event-time vs. processing-time semantics
- Directed acyclic graphs (DAGs) and topological ordering
- Associative and commutative operations (monoids, semigroups)
- Probability distributions (uniform, exponential, Poisson)
- Interval merging algorithms and union-find
- MapReduce paradigm and shuffle mechanics
