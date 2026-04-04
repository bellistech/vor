# The Mathematics of OpenTelemetry — Sampling, Cardinality, and Pipeline Throughput

> *OpenTelemetry pipelines transform raw telemetry into actionable signals. The math governs sampling probabilities, trace completeness, cardinality explosion, Collector queue theory, and the storage cost of each dimension added to a metric.*

---

## 1. Sampling Theory (Probability)

### Head Sampling — TraceIDRatioBased

The sampler hashes the 128-bit trace ID and compares against a threshold:

$$P(\text{sampled}) = r \quad \text{where } r \in [0, 1]$$

$$\text{threshold} = r \times 2^{63}$$

$$\text{sampled} = \text{hash(traceID)} < \text{threshold}$$

### Expected Trace Volume

$$E[\text{traces stored}] = N_{\text{total}} \times r$$

$$\text{Var}[\text{traces stored}] = N_{\text{total}} \times r \times (1 - r)$$

| Total Traces/hr | Sample Rate $r$ | Expected Stored | Std Dev |
|:---:|:---:|:---:|:---:|
| 1,000,000 | 1.0 | 1,000,000 | 0 |
| 1,000,000 | 0.1 | 100,000 | 300 |
| 1,000,000 | 0.01 | 10,000 | 99.5 |
| 1,000,000 | 0.001 | 1,000 | 31.6 |

### Trace Completeness Under Independent Sampling

If services sample independently with rates $r_1, r_2, \ldots, r_k$, the probability of a complete trace across $k$ services:

$$P(\text{complete}) = \prod_{i=1}^{k} r_i$$

| Services | Per-service Rate | Complete Trace Prob |
|:---:|:---:|:---:|
| 3 | 0.5 | 0.125 |
| 5 | 0.5 | 0.031 |
| 3 | 0.1 | 0.001 |
| 10 | 0.9 | 0.349 |

This is why `ParentBased` sampling (propagating the parent decision) is critical.

---

## 2. Metric Cardinality (Combinatorics)

### Cardinality Explosion

The number of unique time series for a metric with $d$ label dimensions:

$$C = \prod_{i=1}^{d} |V_i|$$

where $|V_i|$ is the number of distinct values for label $i$.

### Worked Examples

| Metric | Labels | Values per Label | Cardinality |
|:---|:---|:---:|:---:|
| http_requests | method(4), status(5), route(50) | 4, 5, 50 | 1,000 |
| http_requests | + pod(100) | 4, 5, 50, 100 | 100,000 |
| http_requests | + user_id(10,000) | 4, 5, 50, 10,000 | 10,000,000 |

### Storage Cost per Series

$$\text{Storage/day} = C \times \frac{86400}{I} \times B$$

where $I$ = scrape interval (seconds), $B$ = bytes per sample (~2 bytes compressed).

| Cardinality | Interval | Bytes/Sample | Daily Storage |
|:---:|:---:|:---:|:---:|
| 1,000 | 15s | 2 | 11.5 MiB |
| 100,000 | 15s | 2 | 1.1 GiB |
| 10,000,000 | 15s | 2 | 114.4 GiB |

---

## 3. Collector Queue Theory (Queueing)

### Collector as M/D/1 Queue

The Collector batch processor acts as a queue with Poisson arrivals and deterministic service (batch flush):

$$\rho = \frac{\lambda}{\mu}$$

where $\lambda$ = arrival rate (spans/sec), $\mu$ = processing rate (spans/sec).

### Queue Length (Pollaczek-Khinchine)

$$L_q = \frac{\rho^2}{2(1 - \rho)}$$

$$W_q = \frac{L_q}{\lambda}$$

| Arrival (spans/s) | Capacity (spans/s) | $\rho$ | Avg Queue | Avg Wait |
|:---:|:---:|:---:|:---:|:---:|
| 5,000 | 10,000 | 0.5 | 0.25 | 50 us |
| 8,000 | 10,000 | 0.8 | 1.6 | 200 us |
| 9,500 | 10,000 | 0.95 | 9.03 | 950 us |
| 9,900 | 10,000 | 0.99 | 49.5 | 5 ms |

### Memory Limiter Threshold

$$\text{Queue Memory} = L_q \times S_{\text{avg}}$$

where $S_{\text{avg}}$ = average span size in bytes (~500-2000 bytes).

---

## 4. Batch Processor Optimization (Throughput)

### Batch Efficiency

$$\text{Batches/sec} = \frac{\lambda}{B_{\text{size}}}$$

$$\text{Network Overhead} = \text{Batches/sec} \times H$$

where $H$ = per-request header overhead (~200 bytes for gRPC).

### Optimal Batch Size

Minimize total overhead (batching latency + network overhead):

$$\text{Latency} = \frac{B_{\text{size}}}{\lambda} \quad \text{(time to fill batch)}$$

| Arrival Rate | Batch Size | Batches/sec | Fill Time | Network Overhead |
|:---:|:---:|:---:|:---:|:---:|
| 10,000 | 256 | 39.1 | 25.6 ms | 7.8 KB/s |
| 10,000 | 1,024 | 9.8 | 102 ms | 1.95 KB/s |
| 10,000 | 8,192 | 1.2 | 819 ms | 0.24 KB/s |

Larger batches reduce overhead but increase latency. The `timeout` setting caps max wait.

---

## 5. Trace ID Collision Probability (Birthday Problem)

### Collision in 128-bit Trace IDs

$$P(\text{collision}) \approx 1 - e^{-\frac{n^2}{2 \times 2^{128}}}$$

$$P \approx \frac{n^2}{2^{129}}$$

| Traces Generated | Collision Probability |
|:---:|:---:|
| $10^9$ (1 billion) | $1.47 \times 10^{-21}$ |
| $10^{12}$ (1 trillion) | $1.47 \times 10^{-15}$ |
| $10^{15}$ (1 quadrillion) | $1.47 \times 10^{-9}$ |
| $10^{18}$ | $1.47 \times 10^{-3}$ |

128-bit IDs are effectively collision-free at any practical scale.

---

## 6. Export Bandwidth (Network)

### OTLP Payload Sizing

$$\text{Bandwidth} = \lambda \times S_{\text{avg}} \times (1 - c)$$

where $c$ = gRPC compression ratio (~0.3-0.7 with gzip).

| Spans/sec | Avg Size (bytes) | Compression | Bandwidth |
|:---:|:---:|:---:|:---:|
| 1,000 | 1,000 | 0.5 | 500 KB/s |
| 10,000 | 1,000 | 0.5 | 5 MB/s |
| 100,000 | 500 | 0.6 | 20 MB/s |
| 100,000 | 1,000 | 0.7 | 30 MB/s |

---

## Prerequisites

probability, combinatorics, queueing-theory, prometheus, jaeger

## Complexity

| Operation | Time | Space |
|:---|:---:|:---:|
| Span creation | O(1) | O(k) attributes |
| Context propagation | O(1) | O(1) per header |
| TraceIDRatioBased sample | O(1) | O(1) |
| Batch flush | O(n) | O(batch_size) |
| Collector filter eval | O(s) per span | O(1) |
| Metric cardinality check | O(d) dimensions | O(C) series |
