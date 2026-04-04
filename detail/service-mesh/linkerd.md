# The Mathematics of Linkerd — Lightweight Proxy Efficiency and Reliability Metrics

> *Linkerd's design philosophy is mathematical minimalism: a Rust proxy that optimizes for constant-time per-request overhead, exponentially decaying retry budgets, and statistically sound golden metrics computed from streaming histograms with bounded memory.*

---

## 1. Proxy Efficiency (Resource Complexity)

### Memory Model

The linkerd2-proxy (Rust, tokio-based) memory footprint:

$$M_{proxy} = M_{base} + M_{conn} \times C_{active} + M_{route} \times R_{routes}$$

Where:
- $M_{base} \approx 10\text{MB}$ (binary + runtime)
- $M_{conn} \approx 8\text{KB}$ per connection (vs ~50KB for Envoy)
- $M_{route} \approx 0.5\text{KB}$ per route entry

### Comparison with Envoy

| Metric | linkerd2-proxy | Envoy | Ratio |
|:---|:---:|:---:|:---:|
| Base memory | 10 MB | 50 MB | 0.2x |
| Per-connection | 8 KB | 50 KB | 0.16x |
| Per-route | 0.5 KB | 2 KB | 0.25x |
| Binary size | ~15 MB | ~40 MB | 0.38x |
| Startup time | ~50ms | ~200ms | 0.25x |

### Mesh-Wide Resource Cost

For $N$ pods with $C$ average connections each:

$$M_{mesh} = N \times (M_{base} + M_{conn} \times C)$$

| Pods | Connections/Pod | Linkerd Memory | Envoy Memory | Savings |
|:---:|:---:|:---:|:---:|:---:|
| 100 | 50 | 1.04 GB | 5.24 GB | 4.2 GB |
| 500 | 100 | 5.39 GB | 27.44 GB | 22 GB |
| 1000 | 200 | 11.56 GB | 59.60 GB | 48 GB |

---

## 2. Latency Overhead (Per-Hop Cost Model)

### Single-Hop Overhead

Each proxy hop adds:

$$T_{hop} = T_{parse} + T_{route} + T_{tls} + T_{forward}$$

| Component | linkerd2-proxy | Contribution |
|:---|:---:|:---:|
| HTTP parse | 0.05ms | Header parsing |
| Route lookup | 0.01ms | Hash map |
| TLS (session reuse) | 0.02ms | Resume |
| Forward + copy | 0.10ms | Async I/O |
| **Total** | **~0.18ms** | Per direction |

### End-to-End Overhead

For inbound + outbound through mesh:

$$T_{mesh} = 2 \times T_{hop} \approx 0.36\text{ms}$$

For a request chain of depth $d$:

$$T_{total\_overhead} = d \times 2 \times T_{hop}$$

### Latency Percentile Impact

If the application P99 is $T_{app}$:

$$T_{P99}^{meshed} = T_{app}^{P99} + d \times T_{mesh}^{P99}$$

At $T_{mesh}^{P99} \approx 1\text{ms}$ and depth $d = 3$:

$$\Delta T_{P99} = 3\text{ms}$$

---

## 3. Retry Budget Mathematics (Token Bucket with Ratio)

### Retry Budget Model

Linkerd limits retries as a ratio of successful requests:

$$R_{allowed}(t) = \max\left(R_{min}, \lfloor Q_{success}(t, \Delta t) \times r_{ratio} \rfloor\right)$$

Where:
- $r_{ratio}$ = retry ratio (default 0.2 = 20%)
- $R_{min}$ = minimum retries per second (default 10)
- $\Delta t$ = TTL window (default 10s)

### Retry Amplification Bound

Maximum total request volume:

$$Q_{total} \leq Q_{original} \times (1 + r_{ratio}) = Q_{original} \times 1.2$$

This is a hard ceiling, unlike unbounded retry policies.

### Comparison: Fixed vs Budget Retries

For $Q = 1000$ rps and $p_f = 10\%$ failure rate:

| Strategy | Retry Volume | Total Volume | Amplification |
|:---|:---:|:---:|:---:|
| No retries | 0 | 1000 | 1.0x |
| Fixed 3 retries | 300 | 1300 | 1.3x |
| Budget 20% | 200 | 1200 | 1.2x |
| Budget 20% (cascading) | capped | $\leq 1.2^d$ | Bounded |

### Recovery Dynamics

After a burst of failures, the budget refills:

$$B(t) = \min\left(B_{max}, \int_{t-\Delta t}^{t} Q_{success}(\tau) \times r_{ratio} \, d\tau\right)$$

---

## 4. Golden Metrics (Statistical Aggregation)

### The Four Golden Signals (Linkerd Subset)

Linkerd tracks three golden metrics per route:

$$\text{Success Rate} = \frac{\sum \text{2xx}}{\sum \text{total}}$$

$$\text{Request Rate} = \frac{\sum \text{requests}}{\Delta t}$$

$$\text{Latency Percentiles} = P_{50}, P_{95}, P_{99}$$

### Histogram Implementation

Linkerd uses streaming histograms with fixed bucket boundaries:

$$\text{Buckets} = \{1, 2, 3, 4, 5, 10, 20, 50, 100, 200, 300, 500, 1000, 2000, 5000, 10000\} \text{ ms}$$

Memory per histogram: $|\text{Buckets}| \times 8\text{B} = 128\text{B}$

### Percentile Estimation Error

For bucket boundaries $[b_i, b_{i+1})$:

$$\text{Error}(P_k) \leq b_{i+1} - b_i$$

Maximum estimation error at the 99th percentile in the 500-1000ms range:

$$\epsilon_{P99} \leq 500\text{ms}$$

### Aggregation Over Time Windows

For a sliding window of duration $W$ with samples arriving at rate $\lambda$:

$$n_{samples} = \lambda \times W$$

Standard error of the success rate estimate:

$$SE = \sqrt{\frac{p(1-p)}{n}} = \sqrt{\frac{0.99 \times 0.01}{\lambda W}}$$

For $\lambda = 100$ rps and $W = 30$s: $SE = 0.0018$ (0.18% precision).

---

## 5. Traffic Split Mathematics (SMI Weighted Routing)

### Weight Normalization

TrafficSplit weights are relative, not absolute:

$$P(backend_i) = \frac{w_i}{\sum_{j} w_j}$$

### Equivalent Representations

$$900:100 \equiv 90:10 \equiv 9:1 \implies P(\text{canary}) = \frac{1}{10} = 10\%$$

### Gradual Rollout Schedule

For a rollout from 0% to 100% in $n$ stages with exponential growth:

$$w_k = \min\left(W_{total}, w_0 \times 2^k\right)$$

| Stage | Canary Weight | Stable Weight | Canary % |
|:---:|:---:|:---:|:---:|
| 0 | 10 | 990 | 1% |
| 1 | 50 | 950 | 5% |
| 2 | 100 | 900 | 10% |
| 3 | 250 | 750 | 25% |
| 4 | 500 | 500 | 50% |
| 5 | 1000 | 0 | 100% |

### Rollback Decision

Rollback if canary success rate drops below threshold:

$$\text{Rollback} \iff SR_{canary} < SR_{stable} - \delta$$

With statistical significance test (z-test):

$$z = \frac{SR_{stable} - SR_{canary}}{\sqrt{\frac{p(1-p)}{n_{stable}} + \frac{p(1-p)}{n_{canary}}}}$$

Rollback if $z > z_{\alpha}$ (typically $\alpha = 0.05$, $z_\alpha = 1.645$ one-sided).

---

## 6. mTLS Certificate Lifecycle (Cryptographic Protocol)

### Certificate Rotation Timeline

$$T_{rotation} = T_{issuance} + T_{validity} \times (1 - f_{overlap})$$

Default: $T_{validity} = 24h$, $f_{overlap} = 0.25$ (6h overlap):

$$T_{rotation} = 18h$$

### Trust Anchor Rotation

Root CA validity: typically 1-10 years.

Rotation requires dual-trust period:

$$T_{dual\_trust} = T_{max\_cert\_validity} = 24h$$

### Handshake Performance (Session Resumption)

First connection (full handshake):

$$T_{first} = 2 \times RTT + T_{crypto} \approx 2\text{ms} + 1.5\text{ms} = 3.5\text{ms}$$

Resumed connection:

$$T_{resumed} = 1 \times RTT + T_{ticket} \approx 1\text{ms} + 0.1\text{ms} = 1.1\text{ms}$$

Session cache hit rate for long-lived connections:

$$P_{resume} = 1 - e^{-\lambda \times T_{session\_cache}}$$

---

## 7. Multi-Cluster Routing (Distributed Systems)

### Gateway Latency Model

Cross-cluster request via gateway:

$$T_{cross} = T_{src\_proxy} + T_{gateway\_src} + T_{network} + T_{gateway\_dst} + T_{dst\_proxy}$$

$$T_{cross} \approx 0.4\text{ms} + 0.5\text{ms} + T_{network} + 0.5\text{ms} + 0.4\text{ms}$$

### Failover Mathematics

With service mirroring, failover time:

$$T_{failover} = T_{detect} + T_{dns\_propagation} + T_{mirror\_update}$$

$$T_{failover} \approx 10\text{s} + 5\text{s} + 1\text{s} = 16\text{s}$$

### Cross-Cluster Traffic Cost

$$C_{cross} = D_{bytes} \times C_{egress} + D_{bytes} \times C_{ingress}$$

For inter-region: $C_{egress} \approx \$0.01/\text{GB}$ to $\$0.09/\text{GB}$.

---

*Linkerd achieves its "ultralight" promise through mathematical discipline: bounded memory via fixed histograms, bounded retry amplification via ratio budgets, and bounded latency overhead via a zero-allocation Rust proxy. Every design choice is an optimization constraint.*

## Prerequisites

- Probability distributions and statistical testing
- Token bucket and rate limiting algorithms
- Streaming histogram data structures
- Basic queueing theory (Little's Law)

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| Route lookup | $O(1)$ hash | $O(R)$ routes |
| Traffic split selection | $O(k)$ backends | $O(k)$ |
| Retry budget check | $O(1)$ | $O(1)$ counters |
| Histogram update | $O(\log B)$ buckets | $O(B)$ fixed |
| Percentile query | $O(B)$ scan | $O(1)$ |
| mTLS session resume | $O(1)$ ticket | $O(S)$ cache |
