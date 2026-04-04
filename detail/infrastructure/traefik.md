# The Mathematics of Traefik — Load Balancing, Rate Limiting, and TLS Overhead

> *Traefik's routing and resilience features are grounded in classical algorithms: weighted round-robin for traffic distribution, consistent hashing for session affinity, token bucket for rate limiting, and finite state machines for circuit breaking. Quantifying these mechanisms enables precise capacity planning and SLA engineering.*

---

## 1. Weighted Round-Robin (Scheduling Theory)

### The Problem

Traefik distributes requests across backends using weighted round-robin (WRR). Each server receives traffic proportional to its weight, enabling canary deployments and heterogeneous backend capacity. The challenge is achieving smooth distribution without long bursts to any single server.

### The Formula

For $n$ servers with weights $w_1, w_2, \ldots, w_n$, the total weight:

$$W = \sum_{i=1}^{n} w_i$$

Expected request fraction for server $i$:

$$f_i = \frac{w_i}{W}$$

Over $R$ total requests, expected count for server $i$:

$$E[R_i] = R \cdot \frac{w_i}{W}$$

Smooth Weighted Round-Robin (Nginx/Traefik variant) ensures maximum deviation from ideal distribution is bounded:

$$|R_i - E[R_i]| \leq n - 1$$

### Worked Examples

Canary deployment with 3 servers (stable: 45, stable: 45, canary: 10):

| Server | Weight | Fraction | Requests per 1,000 | Max Deviation |
|:---|:---:|:---:|:---:|:---:|
| stable-1 | 45 | 0.45 | 450 | +/- 2 |
| stable-2 | 45 | 0.45 | 450 | +/- 2 |
| canary | 10 | 0.10 | 100 | +/- 2 |

Gradually shifting canary weight: at $w_{canary} = 10$, error detection requires:

$$N_{detect} = \frac{1}{f_{canary}} \cdot \frac{1}{P_{error}} = \frac{W}{w_{canary}} \cdot \frac{1}{P_{error}}$$

For 1% error rate at 10% traffic: $N_{detect} = 10 \cdot 100 = 1{,}000$ requests to detect with 95% confidence.

---

## 2. Consistent Hashing (Distributed Systems)

### The Problem

Traefik's sticky sessions use cookie-based affinity, but consistent hashing provides an alternative for stateful routing. When backends are added or removed, consistent hashing minimizes the number of remapped keys, preserving session locality.

### The Formula

For $n$ servers on a hash ring with $k$ virtual nodes each, the total ring positions:

$$P = n \cdot k$$

Fraction of keys remapped when adding one server:

$$F_{remap} = \frac{1}{n + 1}$$

Compare to naive modular hashing where adding a server remaps:

$$F_{naive} = \frac{n}{n + 1}$$

Load variance across servers with $k$ virtual nodes (keys uniformly distributed):

$$\text{Var}(L_i) = \frac{K}{n} \cdot \left(\frac{1}{k} + \frac{1}{n \cdot k}\right)$$

where $K$ is total keys. Standard deviation of load:

$$\sigma_L = \sqrt{\frac{K}{n \cdot k}}$$

### Worked Examples

With $K = 100{,}000$ sessions:

| Servers ($n$) | Virtual Nodes ($k$) | Keys Remapped (add 1) | $\sigma_L$ | Max Imbalance |
|:---:|:---:|:---:|:---:|:---:|
| 5 | 1 | 16,667 | 141 | ~28% |
| 5 | 50 | 16,667 | 20 | ~4% |
| 5 | 150 | 16,667 | 12 | ~2.3% |
| 10 | 150 | 9,091 | 8 | ~1.6% |

With $k = 150$ virtual nodes, load imbalance drops below 3%. Consistent hashing remaps only $\frac{1}{n+1}$ of keys versus $\frac{n}{n+1}$ for modular -- a $(n+1)\times$ improvement.

---

## 3. Circuit Breaker State Machine (Automata Theory)

### The Problem

Traefik's circuit breaker middleware uses a three-state machine (Closed, Open, Half-Open) to protect backends from cascading failures. The transition thresholds determine how quickly the breaker trips and recovers.

### The Formula

State transitions follow:

$$\text{Closed} \xrightarrow{\text{error ratio} > \theta} \text{Open} \xrightarrow{t > T_{timeout}} \text{Half-Open} \xrightarrow{\text{probe succeeds}} \text{Closed}$$
$$\text{Half-Open} \xrightarrow{\text{probe fails}} \text{Open}$$

Traefik's expression-based trigger:

$$\text{Trip when: } \frac{N_{error}}{N_{total}} > \theta \quad \lor \quad P_{50}(\text{latency}) > L_{max}$$

Error ratio over sliding window of $W$ seconds with request rate $\lambda$:

$$\hat{\theta} = \frac{\sum_{t=0}^{W} \mathbb{1}[\text{error at } t]}{W \cdot \lambda}$$

False positive probability (tripping when backend is healthy) with true error rate $p$ and window size $W \cdot \lambda$:

$$P_{false} = P\left(\hat{\theta} > \theta \mid p < \theta\right) = 1 - \Phi\left(\frac{(\theta - p)\sqrt{W \cdot \lambda}}{\sqrt{p(1-p)}}\right)$$

### Worked Examples

Trigger threshold $\theta = 0.30$, true error rate $p = 0.05$, $\lambda = 100$ req/sec:

| Window ($W$) | Sample Size | $P_{false}$ | Detection Time (real fault) |
|:---:|:---:|:---:|:---:|
| 1 s | 100 | $3.5 \times 10^{-10}$ | 1 s |
| 5 s | 500 | $\approx 0$ | 5 s |
| 10 s | 1,000 | $\approx 0$ | 10 s |
| 1 s | 10 (low traffic) | 0.0003 | 1 s |

With 100 req/sec, false positive probability is negligible. At low traffic (10 req/sec, 1s window), the small sample size increases false positives -- widen the window or lower the threshold.

---

## 4. Token Bucket Rate Limiting (Queuing Theory)

### The Problem

Traefik's rate limiter uses a token bucket algorithm. Tokens are added at a fixed rate (the `average`), and each request consumes one token. The `burst` parameter sets the bucket capacity, allowing short traffic spikes while enforcing long-term rate limits.

### The Formula

Token bucket state at time $t$:

$$B(t) = \min\left(B_{max}, B(t_0) + r \cdot (t - t_0)\right)$$

where $r$ is the refill rate (tokens/sec) and $B_{max}$ is the burst capacity.

Request at time $t$ is accepted iff $B(t) \geq 1$, then $B(t) \leftarrow B(t) - 1$.

Maximum burst duration at rate $\lambda_{burst}$ (assuming bucket starts full):

$$T_{burst} = \frac{B_{max}}{\lambda_{burst} - r} \quad \text{for } \lambda_{burst} > r$$

Steady-state rejection rate with arrival rate $\lambda > r$:

$$P_{reject} = 1 - \frac{r}{\lambda} \quad \text{(long-term)}$$

Time to refill from empty:

$$T_{refill} = \frac{B_{max}}{r}$$

### Worked Examples

Traefik config: `average: 100` (r = 100/sec), `burst: 200` ($B_{max} = 200$):

| Arrival Rate ($\lambda$) | Burst Duration | Rejection Rate | Refill Time |
|:---:|:---:|:---:|:---:|
| 100/sec | $\infty$ (no burst) | 0% | 2.0 s |
| 150/sec | 4.0 s | 33% (long-term) | 2.0 s |
| 200/sec | 2.0 s | 50% (long-term) | 2.0 s |
| 500/sec | 0.5 s | 80% (long-term) | 2.0 s |
| 1,000/sec | 0.22 s | 90% (long-term) | 2.0 s |

A flash crowd at 200 req/sec exhausts the burst in 2 seconds, then 50% of requests are rejected until traffic drops below 100 req/sec.

---

## 5. TLS Handshake Overhead (Networking)

### The Problem

Every new HTTPS connection to Traefik requires a TLS handshake. The handshake cost depends on the protocol version, cipher suite, and whether session resumption is available. This overhead directly impacts time-to-first-byte and connection throughput.

### The Formula

TLS 1.2 full handshake (2-RTT):

$$T_{TLS1.2} = 2 \cdot RTT + T_{kex} + T_{verify}$$

TLS 1.3 full handshake (1-RTT):

$$T_{TLS1.3} = RTT + T_{kex} + T_{verify}$$

TLS 1.3 with 0-RTT resumption:

$$T_{0RTT} = T_{kex} \quad \text{(data sent with ClientHello)}$$

Connection throughput with keep-alive and $n$ requests per connection:

$$\text{Overhead per request} = \frac{T_{handshake}}{n}$$

### Worked Examples

$RTT = 50\text{ms}$, $T_{kex} = 2\text{ms}$ (X25519), $T_{verify} = 1\text{ms}$ (ECDSA P-256):

| Protocol | Handshake Time | 1 req/conn | 10 req/conn | 100 req/conn |
|:---|:---:|:---:|:---:|:---:|
| TLS 1.2 (full) | 103 ms | 103 ms | 10.3 ms | 1.03 ms |
| TLS 1.2 (resumed) | 53 ms | 53 ms | 5.3 ms | 0.53 ms |
| TLS 1.3 (full) | 53 ms | 53 ms | 5.3 ms | 0.53 ms |
| TLS 1.3 (0-RTT) | 2 ms | 2 ms | 0.2 ms | 0.02 ms |

TLS 1.3 halves the handshake cost versus TLS 1.2. HTTP/2 multiplexing (many requests per connection) amortizes the cost further. Traefik enables TLS 1.3 by default.

---

## 6. Let's Encrypt ACME Challenge Timing (Protocol Analysis)

### The Problem

Traefik's automatic TLS uses Let's Encrypt's ACME protocol. Certificate issuance requires solving a challenge (HTTP-01, TLS-ALPN-01, or DNS-01), each with different timing characteristics. Rate limits constrain issuance volume.

### The Formula

Total certificate issuance time:

$$T_{cert} = T_{order} + T_{authz} + T_{challenge} + T_{validation} + T_{finalize}$$

For HTTP-01 challenge:

$$T_{HTTP01} = RTT_{ACME} + T_{token\_deploy} + T_{validation\_delay} + RTT_{CA \to server}$$

For DNS-01 challenge:

$$T_{DNS01} = RTT_{ACME} + T_{DNS\_propagation} + T_{TTL\_wait} + T_{validation\_delay}$$

Let's Encrypt rate limits (per registered domain):

$$R_{certs} = 50 \text{ per week}$$
$$R_{orders} = 300 \text{ per 3 hours (new orders)}$$

Renewal scheduling to avoid expiry (certificates valid for 90 days):

$$T_{renew} = T_{expiry} - T_{margin}$$

Recommended: $T_{margin} = 30$ days (Traefik default).

### Worked Examples

| Challenge Type | Typical $T_{challenge}$ | Total $T_{cert}$ | Wildcard Support |
|:---|:---:|:---:|:---:|
| HTTP-01 | 5-15 s | 10-30 s | No |
| TLS-ALPN-01 | 5-15 s | 10-30 s | No |
| DNS-01 | 30-300 s | 60-360 s | Yes |

Capacity planning for auto-provisioning $N$ domains:

| Domains ($N$) | First Issuance (HTTP-01) | Weekly Renewals (at 90d) | Within Rate Limit? |
|:---:|:---:|:---:|:---:|
| 10 | ~5 min | 0.78 | Yes |
| 50 | ~25 min | 3.9 | Yes |
| 100 | ~50 min | 7.8 | Yes |
| 500 | ~4.2 hours | 38.9 | Yes |
| 1,000 | ~8.3 hours | 77.8 | Exceeds 50/week limit |

For 1,000+ domains, use wildcard certificates with DNS-01 to consolidate under fewer certificates.

---

## Prerequisites

- scheduling-algorithms, consistent-hashing, finite-state-machines, queuing-theory, tls-protocol, acme-protocol
