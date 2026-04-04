# The Mathematics of Envoy — Load Balancing, Circuit Breaking, and Queueing

> *Envoy is a probabilistic traffic router: every load balancing decision is a weighted random selection, circuit breaking implements token bucket admission control, and retry logic follows exponential backoff distributions. The math governs latency percentiles, failure isolation, and throughput bounds.*

---

## 1. Load Balancing Algorithms (Probability and Optimization)

### Round Robin

Requests distributed cyclically across $n$ endpoints:

$$\text{endpoint}(r) = r \bmod n$$

$$\text{Load per endpoint} = \frac{Q}{n}$$

Perfectly balanced only when all endpoints have equal capacity.

### Weighted Round Robin

With weights $\{w_1, \ldots, w_n\}$:

$$P(e_i) = \frac{w_i}{\sum_{j=1}^{n} w_j}$$

$$E[\text{requests to } e_i] = Q \times \frac{w_i}{W}$$

Where $W = \sum w_j$.

### Least Requests (Power of Two Choices)

Envoy's LEAST_REQUEST samples 2 random endpoints and picks the one with fewer active requests:

$$P(\text{pick least loaded}) = 1 - \left(\frac{k}{n}\right)^2$$

Where $k$ = number of overloaded endpoints.

The maximum load with $n$ endpoints reduces from:

$$O(\log n / \log \log n) \quad \text{(single random)} \to O(\log \log n) \quad \text{(two choices)}$$

| Endpoints | Max Load (random) | Max Load (2-choice) | Improvement |
|:---:|:---:|:---:|:---:|
| 10 | 3.3x avg | 1.7x avg | 2x better |
| 100 | 5.0x avg | 2.1x avg | 2.4x better |
| 1000 | 6.9x avg | 2.4x avg | 2.9x better |

### Ring Hash (Consistent Hashing)

For $n$ endpoints with $v$ virtual nodes each:

$$\text{Total ring positions} = n \times v$$

$$P(\text{endpoint } e_i) = \frac{v_i}{\sum v_j}$$

Key-to-endpoint mapping:

$$\text{endpoint}(key) = \arg\min_{p \in \text{ring}} (h(key) - p) \bmod 2^{32}$$

Standard deviation of load:

$$\sigma_{load} = \frac{1}{\sqrt{v}} \times \frac{Q}{n}$$

For $v = 150$ (Envoy default): $\sigma \approx 8.2\%$ of mean.

---

## 2. Circuit Breaking (Token Bucket Admission Control)

### Connection Circuit Breaker

$$\text{Allow}(req) = |\text{active\_connections}| < C_{max}$$

When tripped:

$$\text{Reject rate} = \frac{Q_{incoming} - C_{max} / \bar{T}_{conn}}{Q_{incoming}}$$

### Request Queue Model

Envoy models pending requests as a bounded queue:

$$Q_{pending} \leq P_{max}$$

The system operates in one of three regimes:

$$\text{Regime} = \begin{cases}
\text{Normal} & Q_{pending} < P_{max} \times 0.5 \\
\text{Degraded} & P_{max} \times 0.5 \leq Q_{pending} < P_{max} \\
\text{Open} & Q_{pending} \geq P_{max}
\end{cases}$$

### Retry Budget

Envoy limits retries as a percentage of active requests:

$$R_{allowed} = \max\left(R_{min}, \lfloor Q_{active} \times \frac{B_{percent}}{100} \rfloor\right)$$

For $B_{percent} = 20\%$, $R_{min} = 3$, $Q_{active} = 100$:

$$R_{allowed} = \max(3, \lfloor 20 \rfloor) = 20$$

### Retry Storm Prevention

Without budget, retry amplification:

$$Q_{total} = Q_{original} \times (1 + p_{fail} \times R_{max})^{hops}$$

For 3 hops, 10% failure, 3 retries:

$$Q_{total} = Q \times (1 + 0.1 \times 3)^3 = Q \times 1.3^3 = 2.197 \times Q$$

With retry budget at 20%, the multiplier is capped:

$$Q_{total} \leq Q \times 1.2^{hops}$$

---

## 3. Outlier Detection (Statistical Anomaly Detection)

### Consecutive Failure Ejection

$$\text{Eject}(e) \iff \text{consecutive\_errors}(e) \geq k_{threshold}$$

### Success Rate Ejection

Envoy computes per-endpoint success rate:

$$SR(e) = \frac{\text{success}(e)}{\text{total}(e)}$$

Ejection condition:

$$\text{Eject}(e) \iff SR(e) < \bar{SR} - k \times \sigma_{SR}$$

Where $\bar{SR}$ = average success rate across all endpoints, $\sigma_{SR}$ = standard deviation.

### Ejection Duration

$$T_{eject}(n) = \min(T_{base} \times n, T_{max})$$

Where $n$ = consecutive ejection count. Default: $T_{base} = 30s$, $T_{max} = 300s$.

### Maximum Ejection Percentage

$$|\text{ejected}| \leq \lfloor |E| \times P_{max\_eject} \rfloor$$

Default $P_{max\_eject} = 10\%$. With 20 endpoints, at most 2 can be ejected simultaneously.

---

## 4. Rate Limiting (Token Bucket Algorithm)

### Token Bucket Model

$$B(t) = \min\left(B_{max}, B(t - \Delta t) + r \times \Delta t\right)$$

A request is allowed iff $B(t) \geq 1$, then $B(t) \leftarrow B(t) - 1$.

### Burst and Sustained Rate

$$\text{Burst capacity} = B_{max}$$
$$\text{Sustained rate} = r \text{ requests/second}$$

### Fill Interval Mathematics

For `tokens_per_fill` = $T$ and `fill_interval` = $I$:

$$r = \frac{T}{I}$$

Example: 100 tokens per 60 seconds = $1.67$ req/s sustained, 100 req burst.

### Multi-Descriptor Rate Limiting

With external rate limit service, descriptors form a hierarchy:

$$\text{Rate}(req) = \min_{d \in \text{descriptors}(req)} \text{limit}(d)$$

---

## 5. Connection Pooling (Queueing Theory)

### HTTP/1.1 Connection Pool

Maximum concurrent requests per upstream:

$$Q_{max} = C_{max} \times 1 \quad \text{(one request per connection)}$$

### HTTP/2 Connection Pool

With multiplexing:

$$Q_{max} = C_{max} \times S_{max\_streams}$$

Default $S_{max\_streams} = 100$, so 1 connection supports 100 concurrent requests.

### Connection Utilization

$$\rho = \frac{Q_{active}}{Q_{max}} = \frac{Q_{active}}{C \times S}$$

### Little's Law Application

$$\bar{L} = \lambda \times \bar{W}$$

Where $\bar{L}$ = average requests in system, $\lambda$ = arrival rate, $\bar{W}$ = average response time.

If $\bar{W} = 50\text{ms}$ and $\lambda = 1000$ req/s:

$$\bar{L} = 1000 \times 0.05 = 50 \text{ concurrent requests}$$

---

## 6. Retry and Timeout Mathematics (Reliability)

### Retry with Backoff

$$T_{wait}(n) = T_{base} \times 2^n + \text{jitter}(0, T_{base} \times 2^n)$$

### Total Request Time with Retries

$$T_{total} = T_1 + \sum_{i=1}^{R} \left(T_{wait}(i) + T_{attempt}(i)\right)$$

### Success Probability with Retries

For independent failures with probability $p_f$:

$$P(\text{success}) = 1 - p_f^{R+1}$$

| Failure Rate | No Retry | 1 Retry | 2 Retries | 3 Retries |
|:---:|:---:|:---:|:---:|:---:|
| 1% | 99.0% | 99.99% | 99.9999% | ~100% |
| 5% | 95.0% | 99.75% | 99.99% | ~100% |
| 10% | 90.0% | 99.0% | 99.9% | 99.99% |
| 50% | 50.0% | 75.0% | 87.5% | 93.75% |

### Hedged Requests

Send request to $k$ endpoints simultaneously, take first response:

$$T_{hedged} = \min(T_1, T_2, \ldots, T_k)$$

For exponentially distributed latencies with rate $\mu$:

$$E[T_{hedged}] = \frac{1}{k \times \mu}$$

---

## 7. Observability Metrics (Statistical Distributions)

### Latency Percentile Estimation

Envoy computes histograms for upstream response time:

$$P_{99} = \text{value at 99th percentile of latency distribution}$$

### Histogram Buckets

Envoy uses predefined bucket boundaries:

$$\text{Buckets} = \{0.5, 1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000, 30000, 60000\} \text{ ms}$$

### Request Success Rate

$$\text{Success rate} = \frac{\sum \text{2xx responses}}{\sum \text{total responses}}$$

### Throughput Bound

$$Q_{max} = \min\left(\frac{C_{max}}{T_{avg}}, \frac{BW}{S_{avg}}, R_{rate\_limit}\right)$$

---

*Envoy's data plane is a mathematical machine: load balancing is probability, circuit breaking is admission control, retries are geometric series, and rate limiting is token bucket calculus. Understanding these models lets you tune Envoy for optimal throughput and tail latency.*

## Prerequisites

- Probability theory (distributions, expected values)
- Queueing theory (Little's Law, utilization)
- Token bucket algorithms
- Consistent hashing

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| Round robin selection | $O(1)$ | $O(n)$ endpoints |
| Least request (2-choice) | $O(1)$ | $O(n)$ endpoint state |
| Ring hash lookup | $O(\log(n \times v))$ | $O(n \times v)$ |
| Circuit breaker check | $O(1)$ | $O(1)$ counters |
| Rate limit token check | $O(1)$ | $O(1)$ bucket |
| Outlier detection | $O(n)$ per interval | $O(n)$ stats |
