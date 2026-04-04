# The Mathematics of wrk — Latency Distributions and Coordinated Omission

> *wrk operates as a closed-loop load generator where epoll/kqueue multiplexing drives connection-level concurrency. The critical mathematics involve latency distribution analysis, coordinated omission bias, and the relationship between connection count, throughput, and tail latency under queuing constraints.*

---

## 1. Closed-Loop Model (Connection-Based Concurrency)

### Throughput Formula

With $C$ persistent connections and mean response time $\bar{T}$:

$$\text{RPS} = \frac{C}{\bar{T}}$$

Each connection completes one request-response cycle before sending the next. This is a closed-loop (synchronous) model.

$$\text{RPS}_{max} = \frac{C}{T_{min}} \quad (\text{best case, no queuing})$$

### Connection Distribution Across Threads

With $t$ threads and $C$ connections:

$$C_{per\_thread} = \left\lfloor \frac{C}{t} \right\rfloor \quad (\text{some threads get } +1)$$

Each thread runs its own epoll/kqueue event loop:

$$\text{RPS}_{total} = \sum_{i=1}^{t} \frac{C_i}{\bar{T}_i} \approx t \cdot \frac{C/t}{\bar{T}} = \frac{C}{\bar{T}}$$

| Connections ($C$) | Threads ($t$) | Avg Response | Throughput |
|:---:|:---:|:---:|:---:|
| 10 | 2 | 5 ms | 2,000 |
| 100 | 4 | 5 ms | 20,000 |
| 100 | 4 | 20 ms | 5,000 |
| 500 | 8 | 10 ms | 50,000 |
| 1000 | 8 | 50 ms | 20,000 |

---

## 2. Coordinated Omission Problem

### The Bias

In a closed-loop system, when the server slows down, the load generator sends fewer requests. Slow responses are underrepresented in the sample:

$$\text{Measured } P_{99} \ll \text{True } P_{99}$$

### Formal Definition

Let $T_i$ be the $i$-th response time. In a closed loop with $C$ connections:

$$\lambda_{actual}(t) = \frac{C}{T_{response}(t)}$$

When $T_{response}$ doubles (e.g., GC pause), $\lambda$ halves. The measurement misses all the requests that *would have arrived* during the slow period.

### Omission Magnitude

If a server pauses for $P$ seconds while serving $C$ connections at normal rate $\lambda_0$:

$$\text{Missing Samples} = (\lambda_0 - C/P) \times P = \lambda_0 P - C$$

$$\text{Missing fraction} = 1 - \frac{C}{\lambda_0 P} = 1 - \frac{T_{normal}}{P}$$

| Normal Response | Pause Duration | Missing Fraction |
|:---:|:---:|:---:|
| 5 ms | 50 ms | 90% |
| 5 ms | 500 ms | 99% |
| 10 ms | 100 ms | 90% |
| 10 ms | 1000 ms | 99% |

### wrk2 Correction

wrk2 uses a constant-rate model that schedules requests at fixed intervals:

$$t_{scheduled}(i) = t_0 + \frac{i}{\lambda_{target}}$$

$$T_{corrected}(i) = t_{completed}(i) - t_{scheduled}(i)$$

This captures the *service time experienced by users who would have arrived during the pause*.

---

## 3. Latency Distribution Analysis

### HDR Histogram Internals

wrk uses a histogram with logarithmic buckets. For range $[1\mu s, T_{max}]$ with $k$ significant digits:

$$\text{Buckets} = \left\lceil \log_2\left(\frac{T_{max}}{T_{min}}\right) \right\rceil \times 10^k$$

### Percentile Interpolation

For percentile $p$ with $n$ samples in histogram buckets:

$$\text{Target count} = \lceil p \cdot n \rceil$$

$$P_p = \text{bucket lower bound where cumulative count} \geq \text{target}$$

### Common Distribution Shapes

| Distribution | Typical Cause | Signature |
|:---|:---|:---|
| Normal | CPU-bound, stable | $P_{99}/P_{50} \approx 2-3\times$ |
| Log-normal | I/O waits, network | $P_{99}/P_{50} \approx 5-10\times$ |
| Bimodal | Cache hit/miss | Two distinct peaks |
| Heavy-tailed | GC, lock contention | $P_{99.9}/P_{99} \gg 2\times$ |

### Nines of Latency

Each additional nine captures a 10x rarer event:

$$P_{1-10^{-k}} \quad \text{requires} \quad n \geq \frac{10^{k+1}}{1 - \text{confidence}}$$

| Percentile | Samples Needed (95% CI) | Meaning |
|:---:|:---:|:---|
| p50 | 20 | Median experience |
| p90 | 200 | 1 in 10 |
| p99 | 2,000 | 1 in 100 |
| p99.9 | 20,000 | 1 in 1,000 |
| p99.99 | 200,000 | 1 in 10,000 |

---

## 4. Epoll/Kqueue Scalability

### Event Loop Model

Each wrk thread runs a single event loop handling $C/t$ connections:

$$T_{loop} = \text{epoll\_wait}() + \sum_{i=1}^{events} T_{process}(i)$$

### Scalability

epoll operates in $O(k)$ where $k$ is the number of *ready* file descriptors (not total):

$$T_{epoll} = O(k) \quad \text{where } k = \text{ready events per call}$$

Compare with select/poll at $O(C)$ per call:

| Mechanism | Cost per Call | Max FDs | wrk Usage |
|:---|:---:|:---:|:---|
| select | $O(C)$ | 1024 | Not used |
| poll | $O(C)$ | Unlimited | Not used |
| epoll (Linux) | $O(k)$ | Unlimited | Default on Linux |
| kqueue (BSD/macOS) | $O(k)$ | Unlimited | Default on macOS |

### Thread Saturation

A wrk thread saturates when processing time exceeds inter-arrival time:

$$\text{Saturated when: } \frac{C_{thread}}{T_{response}} > \frac{1}{T_{process\_per\_event}}$$

Practical limit per thread: ~50,000-100,000 RPS depending on response size.

---

## 5. HTTP Pipeline Mathematics

### Pipeline Depth

When using wrk's Lua pipeline, $d$ requests are sent per round-trip:

$$\text{Effective RPS} = \frac{C \cdot d}{\bar{T} + (d-1) \cdot T_{server}}$$

For sequential processing on the server:

$$\text{RPS}_{pipeline} = \frac{C \cdot d}{T_{RTT} + d \cdot T_{server}}$$

### Pipeline Gain

$$\text{Speedup} = \frac{\text{RPS}_{pipeline}}{\text{RPS}_{no\_pipeline}} = \frac{d \cdot (T_{RTT} + T_{server})}{T_{RTT} + d \cdot T_{server}}$$

| RTT | Server Time | Pipeline $d$ | Speedup |
|:---:|:---:|:---:|:---:|
| 1 ms | 0.1 ms | 5 | 4.1x |
| 1 ms | 0.1 ms | 10 | 5.5x |
| 10 ms | 1 ms | 5 | 3.7x |
| 0.1 ms | 0.1 ms | 5 | 1.7x |

Pipeline provides largest gains when $T_{RTT} \gg T_{server}$.

---

## 6. Throughput Capacity Detection

### Saturation Curve

As connections increase, throughput follows a saturation curve:

$$\text{RPS}(C) = \frac{C}{T_{base} + T_{queue}(C)}$$

where $T_{queue}(C)$ is the queuing delay that grows with utilization $\rho = C \cdot T_{service} / \text{workers}$:

$$T_{queue}(\rho) = \frac{\rho}{1 - \rho} \cdot T_{service} \quad (\text{M/M/1 approximation})$$

### Finding Max Throughput

The optimal connection count where throughput plateaus:

$$C_{optimal} = \text{workers} \times \frac{T_{total}}{T_{service}}$$

Beyond this, adding connections increases latency without improving throughput:

$$\frac{d(\text{RPS})}{dC} \to 0 \quad \text{as} \quad C \to C_{optimal}$$

| Server Workers | Service Time | RTT | Optimal $C$ |
|:---:|:---:|:---:|:---:|
| 4 | 5 ms | 1 ms | 4.8 |
| 16 | 10 ms | 1 ms | 17.6 |
| 64 | 5 ms | 0.5 ms | 70.4 |

---

## Prerequisites

- Queuing theory (M/M/1, M/M/c models)
- Probability distributions (log-normal, heavy-tailed)
- Operating system I/O multiplexing (epoll, kqueue)
- HTTP/1.1 pipelining and keep-alive semantics
- Histogram-based percentile estimation

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| Event loop iteration | $O(k)$ ready events | $O(C)$ connection state |
| Latency recording | $O(1)$ histogram insert | $O(B)$ histogram buckets |
| Percentile query | $O(B)$ bucket scan | $O(1)$ |
| Request generation (Lua) | $O(1)$ per call | $O(1)$ per thread |
| Summary computation | $O(B)$ histogram merge | $O(t \cdot B)$ per-thread histograms |
| Pipeline batch | $O(d)$ format | $O(d \cdot S_{req})$ buffer |
