# The Mathematics of Jaeger — Trace Sampling and Storage Optimization

> *Distributed tracing generates enormous data volumes. The math covers sampling probability, storage growth rates, index sizing for Elasticsearch and Cassandra backends, and the statistical accuracy of latency percentiles under sampled data.*

---

## 1. Sampling Mathematics (Probability)

### Probabilistic Sampling

Each trace is sampled independently with probability $p$:

$$E[\text{sampled traces}] = N \times p$$

$$\text{SE} = \sqrt{N \times p \times (1-p)}$$

### Rate Limiting

With rate limit $R$ traces/second and arrival rate $\lambda$:

$$p_{\text{effective}} = \min\left(1, \frac{R}{\lambda}\right)$$

| Arrival Rate ($\lambda$) | Rate Limit ($R$) | Effective $p$ | Stored/hour |
|:---:|:---:|:---:|:---:|
| 100/s | 10/s | 0.1 | 36,000 |
| 1,000/s | 10/s | 0.01 | 36,000 |
| 10,000/s | 10/s | 0.001 | 36,000 |
| 5/s | 10/s | 1.0 | 18,000 |

### Multi-Service Trace Completeness

For a trace spanning $k$ services each sampling independently at rate $p_i$:

$$P(\text{complete trace}) = \prod_{i=1}^{k} p_i$$

With parent-based (propagated) sampling:

$$P(\text{complete trace}) = p_{\text{root}} \quad \text{(all children inherit)}$$

| Services | Independent $p=0.1$ | Parent-Based $p=0.1$ |
|:---:|:---:|:---:|
| 2 | 0.01 | 0.1 |
| 5 | $10^{-5}$ | 0.1 |
| 10 | $10^{-10}$ | 0.1 |

---

## 2. Storage Sizing — Elasticsearch (Capacity Planning)

### Daily Index Size

$$\text{Index Size/day} = \frac{\lambda \times p \times 86400 \times S_{\text{avg}}}{c}$$

where $S_{\text{avg}}$ = average span document size (~1-3 KB), $c$ = compression ratio (~0.5).

### Worked Examples

| Spans/sec | Sample Rate | Avg Span Size | Compression | Daily Index |
|:---:|:---:|:---:|:---:|:---:|
| 1,000 | 1.0 | 2 KB | 0.5 | 345.6 GB |
| 10,000 | 0.1 | 2 KB | 0.5 | 345.6 GB |
| 10,000 | 0.01 | 2 KB | 0.5 | 34.6 GB |
| 50,000 | 0.01 | 1.5 KB | 0.5 | 129.6 GB |

### Shard Sizing

Target 30-50 GB per shard for optimal Elasticsearch performance:

$$\text{Shards} = \left\lceil \frac{\text{Daily Index Size}}{40 \text{ GB}} \right\rceil$$

### Retention Cost

$$\text{Total Storage} = \text{Daily Size} \times D_{\text{retention}} \times (1 + R_{\text{replicas}})$$

| Daily Size | Retention | Replicas | Total Storage |
|:---:|:---:|:---:|:---:|
| 34.6 GB | 7 days | 1 | 484.4 GB |
| 34.6 GB | 14 days | 1 | 968.8 GB |
| 345.6 GB | 7 days | 1 | 4.8 TB |
| 345.6 GB | 14 days | 2 | 14.5 TB |

---

## 3. Storage Sizing — Cassandra (Wide Column)

### Write Throughput

$$\text{Writes/sec} = \lambda \times p \times W_{\text{tables}}$$

where $W_{\text{tables}}$ = number of Cassandra tables per span (typically 3: spans, service-operation index, duration index).

| Spans/sec | Sample | Tables | Cassandra Writes/sec |
|:---:|:---:|:---:|:---:|
| 5,000 | 0.1 | 3 | 1,500 |
| 50,000 | 0.01 | 3 | 1,500 |
| 50,000 | 0.1 | 3 | 15,000 |

### Disk Usage with Compaction

$$\text{Disk} = \text{Data Size} \times A_{\text{compaction}}$$

where $A_{\text{compaction}}$ = compaction space amplification (~2x for Size-Tiered, ~1.2x for Leveled).

---

## 4. Latency Percentile Accuracy Under Sampling (Statistics)

### Error Bound for Percentiles

For percentile $q$ estimated from $n$ sampled traces, the standard error:

$$\text{SE}(q) = \frac{\sqrt{q(1-q)}}{\sqrt{n}} \times \frac{1}{f(\xi_q)}$$

where $f(\xi_q)$ is the density at the quantile point.

### Minimum Samples for Accurate Percentiles

| Percentile | Required $n$ (5% error) | Required $n$ (1% error) |
|:---:|:---:|:---:|
| p50 | 384 | 9,604 |
| p90 | 346 | 8,644 |
| p95 | 183 | 4,564 |
| p99 | 38 | 956 |

### Sampling vs Percentile Accuracy

$$n_{\text{samples}} = \lambda \times p \times T_{\text{window}}$$

| Traffic | Sample Rate | 5-min Window | p99 Accuracy |
|:---:|:---:|:---:|:---:|
| 100/s | 1.0 | 30,000 | Excellent |
| 1,000/s | 0.1 | 30,000 | Excellent |
| 1,000/s | 0.01 | 3,000 | Good |
| 100/s | 0.01 | 300 | Marginal |

---

## 5. Trace DAG Properties (Graph Theory)

### Span Tree Structure

A trace is a directed acyclic graph (typically a tree) with $n$ spans:

$$\text{Edges} = n - 1 \quad \text{(tree)}$$

$$\text{Depth} = \text{critical path length}$$

$$\text{Width} = \max_{\text{level}} |\text{spans at level}|$$

### Critical Path Analysis

The trace latency is the critical path through the span DAG:

$$T_{\text{trace}} = \max_{\text{paths}} \sum_{s \in \text{path}} d_s$$

For parallel fan-out with $k$ children:

$$T_{\text{parent}} = T_{\text{self}} + \max(T_{\text{child}_1}, \ldots, T_{\text{child}_k})$$

### Trace Complexity Distribution

| Service Count | Avg Spans/Trace | Depth | Width |
|:---:|:---:|:---:|:---:|
| 3 | 5-10 | 3-4 | 2-3 |
| 10 | 20-50 | 5-8 | 5-10 |
| 50 | 100-500 | 8-15 | 10-50 |

---

## 6. Collector Throughput (Queueing)

### Collector Queue Model

$$\rho = \frac{\lambda_{\text{in}}}{\mu_{\text{write}}}$$

$$\text{Drop rate} = \max(0, \lambda_{\text{in}} - \mu_{\text{write}}) \quad \text{when queue full}$$

### Queue Sizing

$$Q_{\text{memory}} = Q_{\text{size}} \times S_{\text{avg}}$$

| Queue Size | Avg Span | Memory | Drain Time at 10K/s |
|:---:|:---:|:---:|:---:|
| 10,000 | 2 KB | 20 MB | 1 sec |
| 100,000 | 2 KB | 200 MB | 10 sec |
| 1,000,000 | 2 KB | 2 GB | 100 sec |

---

## Prerequisites

probability, graph-theory, elasticsearch, cassandra, opentelemetry

## Complexity

| Operation | Time | Space |
|:---|:---:|:---:|
| Span ingestion (collector) | O(1) | O(S) per span |
| Trace assembly (query) | O(n) spans | O(n) spans |
| Service list query (ES) | O(1) aggregation | O(s) services |
| Trace search by tag (ES) | O(log n) index | O(k) results |
| Sampling decision (head) | O(1) hash | O(1) |
| Critical path computation | O(n) DFS | O(n) stack |
