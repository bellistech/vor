# The Mathematics of Flink — Stream Processing Theory

> *Flink models computation as continuous dataflow graphs with well-defined semantics for time, state, and fault tolerance. The mathematics covers watermark propagation, window semantics, checkpoint barrier alignment, and state scaling.*

---

## 1. Watermark Propagation (Temporal Ordering)

### The Problem

In a distributed stream, events arrive out of order. Watermarks signal that no events with timestamp less than $W$ will arrive, enabling timely window evaluation.

### The Formula

For a source with bounded out-of-orderness $\delta$:

$$W(t) = \max_{e \in \text{seen}} e.timestamp - \delta$$

Multi-source watermark at an operator with $n$ input channels:

$$W_{op} = \min_{i=1}^{n} W_i$$

End-to-end latency for a window $[s, s+w)$:

$$T_{trigger} = s + w + \delta$$

Processing latency:

$$L = T_{trigger} - T_{wall} = (s + w + \delta) - T_{now}$$

### Worked Examples

| Window Size | Out-of-Orderness | Event Time | Trigger At | Max Latency |
|:---:|:---:|:---:|:---:|:---:|
| 5 min | 10 s | 12:00 | 12:05:10 | 5 min 10 s |
| 1 hour | 30 s | 12:00 | 13:00:30 | 60 min 30 s |
| 1 min | 5 s | 12:00 | 12:01:05 | 1 min 5 s |

---

## 2. Window Semantics (Set Theory)

### The Problem

Different window types partition the infinite event stream into finite sets. What events fall into which windows?

### The Formula

Tumbling window (non-overlapping, fixed size $w$):

$$\text{window}(e) = \left\lfloor \frac{e.ts}{w} \right\rfloor \times w$$

$$e \in [k \cdot w, (k+1) \cdot w) \quad \text{for } k = \left\lfloor e.ts / w \right\rfloor$$

Sliding window (size $w$, slide $s$):

$$e \in \text{windows} = \left\{[k \cdot s, k \cdot s + w) : k \cdot s \leq e.ts < k \cdot s + w\right\}$$

Number of windows containing event $e$:

$$n_{windows} = \left\lceil \frac{w}{s} \right\rceil$$

Session window (gap $g$):

$$\text{merge}(W_1, W_2) \iff W_2.start - W_1.end \leq g$$

### Worked Examples

| Window Type | Size | Slide/Gap | Event at 12:07 | Windows |
|:---:|:---:|:---:|:---:|:---:|
| Tumbling | 5 min | - | [12:05, 12:10) | 1 |
| Sliding | 10 min | 5 min | [12:00,12:10), [12:05,12:15) | 2 |
| Sliding | 15 min | 5 min | [11:55,12:10), [12:00,12:15), [12:05,12:20) | 3 |
| Session | - | 10 min gap | Depends on neighbors | 1 (merged) |

---

## 3. Checkpoint Barrier Alignment (Consistency)

### The Problem

Flink achieves exactly-once by injecting barriers into the stream. How does barrier alignment affect latency and buffer size?

### The Formula

Alignment time for operator with $n$ input channels:

$$T_{align} = \max_{i=1}^{n} T_{barrier,i} - \min_{i=1}^{n} T_{barrier,i}$$

Buffered records during alignment:

$$B_{align} = \sum_{i \in \text{fast}} R_i \times T_{align}$$

Where $R_i$ = record rate on channel $i$.

Unaligned checkpoints (Flink 1.11+) eliminate alignment but increase checkpoint size:

$$S_{checkpoint}^{unaligned} = S_{state} + B_{inflight}$$

$$S_{checkpoint}^{aligned} = S_{state}$$

Checkpoint interval lower bound:

$$I_{checkpoint} \geq T_{checkpoint} + T_{min\_pause}$$

### Worked Examples

| Input Channels | Rate Skew | Alignment Time | Buffered Records |
|:---:|:---:|:---:|:---:|
| 2 | 100 ms | 100 ms | 10K records |
| 4 | 500 ms | 500 ms | 200K records |
| 8 | 1 s | 1 s | 800K records |

---

## 4. State Scaling (Redistribution)

### The Problem

When changing parallelism, how does Flink redistribute keyed and operator state?

### The Formula

Key groups (Flink's unit of state redistribution):

$$G = 128 \times \max(\text{parallelism ever used})$$

Key-to-group mapping:

$$g(key) = \text{hash}(key) \mod G$$

Groups per operator instance at parallelism $p$:

$$\text{groups per instance} = \left\lfloor \frac{G}{p} \right\rfloor \text{ or } \left\lceil \frac{G}{p} \right\rceil$$

State transfer during rescale from $p_1$ to $p_2$:

$$S_{transfer} = S_{total} \times \left(1 - \frac{\min(p_1, p_2)}{\max(p_1, p_2)}\right)$$

### Worked Examples

| Old Parallelism | New Parallelism | Key Groups | Groups Moved | State Transfer |
|:---:|:---:|:---:|:---:|:---:|
| 4 | 8 | 512 | 256 | 50% |
| 8 | 4 | 1024 | 0 (merge only) | 0% (logical) |
| 10 | 15 | 1280 | ~427 | ~33% |

---

## 5. Backpressure Model (Flow Control)

### The Problem

When a downstream operator is slower than upstream, how does backpressure propagate and affect throughput?

### The Formula

Sustainable throughput of a pipeline:

$$T_{pipeline} = \min_{op \in \text{pipeline}} T_{op}$$

Backpressure ratio for operator $op$:

$$BP_{op} = 1 - \frac{T_{op}}{T_{upstream}}$$

Buffer pool utilization:

$$U_{buffer} = \frac{B_{used}}{B_{total}}$$

When $U_{buffer} \to 1$, backpressure engages. Credit-based flow control:

$$\text{Credits}_{available} = B_{total} - B_{inflight}$$

Sender is blocked when $\text{Credits}_{available} = 0$.

### Worked Examples

| Upstream Rate | Operator Capacity | Backpressure | Buffer Fill Rate |
|:---:|:---:|:---:|:---:|
| 100K evt/s | 100K evt/s | 0% | Stable |
| 100K evt/s | 80K evt/s | 20% | Growing |
| 100K evt/s | 50K evt/s | 50% | Rapid |

---

## 6. Exactly-Once Cost (Overhead Analysis)

### The Problem

What is the throughput overhead of exactly-once processing compared to at-least-once?

### The Formula

Throughput with aligned checkpoints:

$$T_{eo} = T_{raw} \times \frac{I - T_{align}}{I}$$

Where:
- $I$ = checkpoint interval
- $T_{align}$ = alignment time per checkpoint

With unaligned checkpoints:

$$T_{eo}^{unaligned} \approx T_{raw} \times \frac{I - T_{snap}}{I}$$

Where $T_{snap}$ is the snapshot time (typically much less than $T_{align}$).

Overhead percentage:

$$\text{Overhead} = \frac{T_{align}}{I} \times 100\%$$

### Worked Examples

| Checkpoint Interval | Alignment Time | Throughput Loss | With Unaligned |
|:---:|:---:|:---:|:---:|
| 60 s | 2 s | 3.3% | ~0.1% |
| 30 s | 5 s | 16.7% | ~0.5% |
| 120 s | 2 s | 1.7% | ~0.05% |

---

## 7. Summary of Formulas

| Formula | Type | Domain |
|---------|------|--------|
| $W_{op} = \min_i W_i$ | Watermark propagation | Temporal ordering |
| $n_{windows} = \lceil w/s \rceil$ | Window membership | Set partitioning |
| $B_{align} = \sum R_i \times T_{align}$ | Buffer during alignment | Checkpointing |
| $g(key) = hash(key) \mod G$ | Key group assignment | State scaling |
| $T_{pipeline} = \min_{op} T_{op}$ | Pipeline throughput | Backpressure |
| $\text{Overhead} = T_{align}/I$ | Exactly-once cost | Fault tolerance |

## Prerequisites

- distributed-systems, graph-theory, queuing-theory, event-time-processing, kafka

## Complexity

| Operation | Time | Space |
|-----------|------|-------|
| Watermark propagation | O(1) per event | O(n) input channels |
| Tumbling window assign | O(1) per event | O(W) window state |
| Session window merge | O(n log n) merges | O(sessions) |
| Checkpoint (aligned) | O(State + Alignment) | O(State) snapshot |
| Checkpoint (unaligned) | O(State + Inflight) | O(State + Buffers) |
| State rescale | O(State * transfer%) | O(State) during restore |
