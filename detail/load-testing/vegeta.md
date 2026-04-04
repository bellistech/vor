# The Mathematics of Vegeta — Constant-Rate Load Generation and Statistical Analysis

> *Vegeta implements an open-loop load generator with precise timer-based request scheduling, HDR histogram latency recording, and rate-independent throughput measurement that eliminates coordinated omission bias from performance analysis.*

---

## 1. Open-Loop Model (Constant Arrival Rate)

### Request Scheduling

Vegeta schedules requests at fixed intervals regardless of server response time:

$$t_i = t_0 + \frac{i}{\lambda} \quad \text{for } i = 0, 1, 2, \ldots$$

where $\lambda$ is the target rate (requests/second).

Inter-arrival time is constant:

$$\Delta t = \frac{1}{\lambda}$$

| Rate ($\lambda$) | Inter-arrival Time | Requests in 1 min |
|:---:|:---:|:---:|
| 10/s | 100 ms | 600 |
| 100/s | 10 ms | 6,000 |
| 1,000/s | 1 ms | 60,000 |
| 10,000/s | 100 $\mu$s | 600,000 |
| 50,000/s | 20 $\mu$s | 3,000,000 |

### Workers and Backpressure

Each request requires a worker (goroutine). Maximum concurrent workers needed:

$$W_{max} = \lambda \times T_{response,max}$$

If $W_{max} > W_{configured}$, requests queue at the worker pool:

$$T_{measured} = T_{queue} + T_{server}$$

$$T_{queue} = \max\left(0, \frac{W_{active} - W_{max}}{W_{max}} \times T_{server}\right)$$

| Rate | Max Response | Workers Needed | With 100 Workers |
|:---:|:---:|:---:|:---:|
| 100/s | 100 ms | 10 | OK |
| 500/s | 200 ms | 100 | At limit |
| 1000/s | 500 ms | 500 | Queuing (5x overload) |
| 1000/s | 50 ms | 50 | OK |

---

## 2. Latency Measurement (Response Time Accounting)

### True Response Time

Vegeta measures latency from scheduled send time, not actual send time:

$$T_{latency}(i) = t_{response}(i) - t_{scheduled}(i)$$

This captures queuing delay when the server falls behind:

$$T_{latency} = T_{queue} + T_{network} + T_{server} + T_{network}$$

### Comparison with Closed-Loop Measurement

In a closed loop (wrk-style):

$$T_{measured}^{closed} = t_{response} - t_{sent}$$

This misses the wait time for requests that could not be sent during server pauses.

| Scenario | Closed-Loop P99 | Open-Loop P99 | True User P99 |
|:---|:---:|:---:|:---:|
| Steady 10ms response | 15 ms | 15 ms | 15 ms |
| 10ms + 100ms GC pause | 100 ms | 110 ms | 110 ms |
| 10ms + 1s GC pause | 1,000 ms | 1,010 ms | 1,010 ms |
| 10ms + periodic 500ms stalls | 500 ms | 510 ms | 510 ms |

The open-loop measurement matches user experience because users do not stop arriving when the server is slow.

---

## 3. Rate Accuracy (Timer Precision)

### Go Timer Granularity

Vegeta uses `time.Ticker` for scheduling. The minimum reliable interval:

$$\Delta t_{min} \approx 1 \text{ ms on Linux}, \quad 1 \text{ ms on macOS}$$

$$\lambda_{max} = \frac{1}{\Delta t_{min}} = 1,000 \text{ RPS per worker group}$$

For higher rates, Vegeta uses multiple worker groups:

$$\lambda_{effective} = G \times \lambda_{per\_group}$$

### Jitter Analysis

Timer jitter follows approximately normal distribution:

$$\Delta t_{actual} = \Delta t_{target} + \mathcal{N}(0, \sigma_j^2)$$

$$\sigma_j \approx 50-200 \mu s \quad (\text{typical OS scheduler jitter})$$

Rate accuracy:

$$\text{Rate Error} = \frac{\sigma_j}{\Delta t_{target}}$$

| Target Rate | $\Delta t$ | Jitter ($\sigma_j = 100\mu s$) | Error |
|:---:|:---:|:---:|:---:|
| 100/s | 10 ms | 100 $\mu$s | 1.0% |
| 1,000/s | 1 ms | 100 $\mu$s | 10% |
| 10,000/s | 100 $\mu$s | 100 $\mu$s | 100% (needs batching) |

---

## 4. Statistical Report Computation

### Metrics Aggregation

For $n$ results $\{(t_i, s_i, d_i, b_i)\}$ (timestamp, status, duration, bytes):

$$\text{Rate} = \frac{n}{t_n - t_0}$$

$$\text{Throughput} = \frac{n_{success}}{t_n - t_0}$$

$$\text{Success Ratio} = \frac{|\{i : s_i \in [200, 400)\}|}{n}$$

### Latency Percentiles

Sorted durations $d_{(1)} \leq d_{(2)} \leq \ldots \leq d_{(n)}$:

$$P_p = d_{(\lceil p \cdot n \rceil)}$$

$$\text{Mean} = \frac{1}{n}\sum_{i=1}^{n} d_i$$

### Histogram Bucketing

For user-defined bucket boundaries $[b_0, b_1, b_2, \ldots, b_k]$:

$$\text{Count}_j = |\{i : b_{j-1} \leq d_i < b_j\}|$$

$$\text{Percentage}_j = \frac{\text{Count}_j}{n} \times 100$$

### Standard Error of Percentiles

$$\text{SE}(P_p) = \frac{1}{f(P_p)} \sqrt{\frac{p(1-p)}{n}}$$

For the 99th percentile with $n = 10,000$ and estimated density $f$:

$$\text{SE}(P_{99}) \approx \frac{\sigma}{f(P_{99})} \sqrt{\frac{0.0099}{10000}}$$

---

## 5. Throughput Saturation Curve

### Finding Server Capacity

Running rate sweeps produces a throughput-vs-rate curve:

$$\text{Throughput}(\lambda) = \begin{cases} \lambda & \text{if } \lambda < C \\ C & \text{if } \lambda \geq C \end{cases}$$

where $C$ is the server's maximum capacity (RPS).

### Latency Under Load

Using M/M/c queuing model with $c$ server workers:

$$\rho = \frac{\lambda}{c \cdot \mu}$$

$$E[T] = \frac{1}{\mu} \cdot \left(1 + \frac{C(c, \rho)}{c(1-\rho)}\right)$$

where $C(c, \rho)$ is the Erlang C probability of queuing.

### Rate Sweep Analysis

| Offered Rate | Throughput | P50 | P99 | Status |
|:---:|:---:|:---:|:---:|:---|
| 100/s | 100/s | 5 ms | 12 ms | Under capacity |
| 500/s | 500/s | 8 ms | 25 ms | Approaching limit |
| 1000/s | 980/s | 15 ms | 80 ms | Near saturation |
| 2000/s | 1100/s | 45 ms | 500 ms | Overloaded |
| 5000/s | 1050/s | 200 ms | 2000 ms | Severely overloaded |

The knee of the curve at the saturation point:

$$\frac{d(\text{Throughput})}{d(\lambda)} \to 0, \quad \frac{d(P_{99})}{d(\lambda)} \to \infty$$

---

## 6. Binary Encoding and Result Composition

### Result Record Structure

Each result in Vegeta's binary format:

$$\text{Record} = \underbrace{8}_{\text{timestamp}} + \underbrace{2}_{\text{status}} + \underbrace{8}_{\text{latency}} + \underbrace{8}_{\text{bytes\_out}} + \underbrace{8}_{\text{bytes\_in}} + \underbrace{2+n}_{\text{error}} = 36 + n \text{ bytes}$$

### Storage Requirements

$$S_{results} = n \times (36 + \bar{e}) \text{ bytes}$$

where $\bar{e}$ is the average error string length (0 for successes).

| Duration | Rate | Results | File Size (no errors) |
|:---:|:---:|:---:|:---:|
| 30s | 100/s | 3,000 | 105 KiB |
| 60s | 1,000/s | 60,000 | 2.1 MiB |
| 300s | 5,000/s | 1,500,000 | 51 MiB |
| 3600s | 10,000/s | 36,000,000 | 1.2 GiB |

### Result Composition

Vegeta results are concatenable — Unix pipes can merge attacks:

$$\text{Combined} = R_1 \| R_2 \| \ldots \| R_k$$

$$\text{Report}(\text{Combined}) = f\left(\bigcup_{i=1}^{k} R_i\right)$$

This enables ramp-up simulation:

$$\lambda(t) = \lambda_1 \text{ for } [0, D_1), \quad \lambda_2 \text{ for } [D_1, D_1+D_2), \quad \ldots$$

---

## Prerequisites

- Open-loop vs closed-loop load generation models
- Queuing theory (arrival rates, service times)
- Percentile estimation and statistical significance
- Timer precision and OS scheduling jitter
- Binary serialization and streaming aggregation

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| Attack (per request) | $O(1)$ schedule + send | $O(W)$ active workers |
| Result recording | $O(1)$ append | $O(1)$ per result |
| Report computation | $O(n \log n)$ sort for percentiles | $O(n)$ all results |
| Histogram bucketing | $O(n \cdot k)$ for $k$ buckets | $O(k)$ counters |
| Plot generation | $O(n)$ scan | $O(n)$ data points |
| Encode/decode | $O(n)$ stream | $O(1)$ per record |
