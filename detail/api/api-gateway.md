# The Mathematics of API Gateways -- Rate Limiting Algorithms and Traffic Shaping

> *The gateway's purpose is not to say yes or no, but to say "not yet" at precisely the right moment.*

---

## 1. Token Bucket Algorithm (Steady-State and Burst Analysis)

### The Problem

The token bucket allows bursts while enforcing an average rate. Given a bucket
capacity $C$ and refill rate $r$ tokens per second, what is the maximum burst
size, and how long does it take to recover from a burst?

### The Formula

Tokens available at time $t$ after a burst at $t_0$ that consumed $B$ tokens:

$$T(t) = \min\left(C, \; C - B + r(t - t_0)\right)$$

Recovery time (time to refill from empty to full):

$$t_{recovery} = \frac{C}{r}$$

Time until next request can be served after a full burst:

$$t_{next} = \frac{1}{r}$$

Maximum burst duration at request rate $\lambda > r$:

$$t_{burst} = \frac{C}{\lambda - r}$$

Requests served during burst:

$$N_{burst} = C + r \cdot t_{burst} = C + \frac{r \cdot C}{\lambda - r} = \frac{C \lambda}{\lambda - r}$$

### Worked Examples

**Example 1:** Bucket capacity $C = 100$, refill rate $r = 10$/s. Client sends
50 requests instantly:

$$T(0^+) = 100 - 50 = 50 \text{ tokens remaining}$$

Recovery time: $t_{recovery} = \frac{50}{10} = 5$ seconds to refill.

**Example 2:** Same bucket, client sends at $\lambda = 20$/s sustained.
How long before throttling begins?

$$t_{burst} = \frac{100}{20 - 10} = 10 \text{ seconds}$$

Total requests served before throttling:

$$N_{burst} = \frac{100 \times 20}{20 - 10} = 200 \text{ requests}$$

After 10 seconds, the client is limited to $r = 10$/s.

## 2. Sliding Window Rate Limiting (Precision vs Memory)

### The Problem

Sliding window counters provide more precise rate limiting than fixed windows
but require more memory. What are the space-time tradeoffs?

### The Formula

**Fixed window**: Simple counter per window. Memory:

$$M_{fixed} = K \times (S_{key} + S_{counter}) \text{ per window}$$

where $K$ is the number of unique clients.

Problem: boundary burst allows $2L$ requests across a window boundary.

**Sliding window log**: Store timestamp of every request. Memory:

$$M_{log} = K \times L \times S_{timestamp}$$

where $L$ is the rate limit.

**Sliding window counter** (approximation): Combine current and previous
window counts:

$$count \approx count_{prev} \times (1 - \frac{t_{elapsed}}{W}) + count_{curr}$$

Memory: $M_{approx} = K \times 2S_{counter}$

Error bound:

$$|error| \leq count_{prev} \times \frac{t_{elapsed}}{W}$$

### Worked Examples

**Example 1:** 1M unique API keys, rate limit 100/min, 8-byte timestamps:

$$M_{log} = 10^6 \times 100 \times 8 = 800 \text{ MB}$$

$$M_{approx} = 10^6 \times 2 \times 4 = 8 \text{ MB}$$

$$\text{Space ratio} = \frac{800}{8} = 100\times$$

The approximate approach uses 100x less memory.

**Example 2:** Fixed window boundary attack. Limit: 100 requests per 60s.
Client sends 100 requests at $t = 59$ and 100 at $t = 61$:

$$\text{Actual rate} = \frac{200}{2 \text{ s}} = 100 \text{ req/s}$$

The fixed window allows 200 requests in 2 seconds, effectively 100x
the intended 100/minute rate. Sliding window prevents this.

## 3. JWT Validation Cost (Cryptographic Overhead)

### The Problem

Every request through the gateway requires JWT validation. What is the
CPU cost of RSA vs HMAC vs ECDSA verification, and how does it affect
gateway throughput?

### The Formula

Gateway throughput limited by JWT validation:

$$R_{max} = \frac{N_{cores}}{T_{verify}}$$

Verification times (typical, single core):

| Algorithm | $T_{verify}$ | Operations/s/core |
|-----------|-------------|-------------------|
| HMAC-SHA256 | ~1 $\mu s$ | ~1,000,000 |
| RSA-2048 | ~30 $\mu s$ | ~33,000 |
| RSA-4096 | ~100 $\mu s$ | ~10,000 |
| ECDSA P-256 | ~80 $\mu s$ | ~12,500 |
| Ed25519 | ~60 $\mu s$ | ~16,700 |

With caching (cache JWT validation result for TTL $t_c$):

$$R_{effective} = \frac{R_{max}}{1 - h} + \frac{h}{T_{cache\_lookup}}$$

where $h$ is the cache hit ratio.

### Worked Examples

**Example 1:** 8-core gateway, RSA-2048 JWTs, no caching:

$$R_{max} = \frac{8}{30 \times 10^{-6}} = 266{,}667 \text{ req/s}$$

Sufficient for most APIs, but becomes a bottleneck at scale.

**Example 2:** Same gateway with 90% cache hit ratio:

$$R_{effective} = \frac{266{,}667}{0.1} = 2{,}666{,}670 \text{ req/s (cache-miss limited)}$$

Plus cache hits at ~10M req/s. Effective throughput jumps 10x.

**Example 3:** Switch to HMAC-SHA256 (shared secret, no public key crypto):

$$R_{max} = \frac{8}{1 \times 10^{-6}} = 8{,}000{,}000 \text{ req/s}$$

30x improvement over RSA, but HMAC requires shared secrets (less secure
for multi-party systems).

## 4. Cache Hit Ratio (Gateway Response Caching)

### The Problem

API gateway caching reduces backend load but consumes memory. What is the
optimal cache size, and how does hit ratio vary with cache capacity?

### The Formula

For Zipfian request distribution (typical for APIs) with parameter $s \approx 1$,
the probability of requesting the $k$-th most popular resource:

$$P(k) = \frac{1/k^s}{\sum_{i=1}^{N} 1/i^s} = \frac{1/k^s}{H_{N,s}}$$

Cache hit ratio for cache size $C$ out of $N$ unique resources:

$$h(C) = \frac{\sum_{k=1}^{C} 1/k^s}{H_{N,s}} = \frac{H_{C,s}}{H_{N,s}}$$

For $s = 1$:

$$h(C) \approx \frac{\ln C + \gamma}{\ln N + \gamma}$$

where $\gamma \approx 0.5772$ is the Euler-Mascheroni constant.

### Worked Examples

**Example 1:** 100,000 unique API endpoints, cache for 1,000 ($s = 1$):

$$h(1000) \approx \frac{\ln 1000 + 0.577}{\ln 100000 + 0.577} = \frac{6.908 + 0.577}{11.513 + 0.577} = \frac{7.485}{12.09} = 61.9\%$$

61.9% of requests served from cache with only 1% of endpoints cached.

**Example 2:** Double the cache to 2,000 endpoints:

$$h(2000) \approx \frac{\ln 2000 + 0.577}{12.09} = \frac{7.601 + 0.577}{12.09} = \frac{8.178}{12.09} = 67.6\%$$

Doubling cache size yields only 5.7 percentage points improvement.
Zipfian distributions exhibit diminishing returns from cache expansion.

## 5. Load Balancing Fairness (Request Distribution)

### The Problem

Round-robin load balancing assumes equal backend capacity. When backends
have different capacities or response times, how does unfairness accumulate?

### The Formula

For weighted round-robin with weights $w_i$ and capacities $c_i$:

$$\text{Load ratio}_i = \frac{w_i / \sum w_j}{c_i / \sum c_j}$$

A value of 1.0 is perfectly balanced. The Jain's fairness index:

$$J = \frac{\left(\sum_{i=1}^{n} x_i\right)^2}{n \sum_{i=1}^{n} x_i^2}$$

where $x_i$ is the load ratio per backend. $J = 1$ is perfectly fair,
$J = 1/n$ is maximally unfair.

For least-connections balancing with Poisson arrivals at rate $\lambda$
and exponential service times with rate $\mu_i$:

$$E[\text{queue}_i] = \frac{\rho_i}{1 - \rho_i}$$

where $\rho_i = \lambda_i / \mu_i$.

### Worked Examples

**Example 1:** 3 backends, round-robin (equal weights), capacities 100, 80, 60 req/s.
Each gets $\lambda/3$ of traffic:

$$\text{Load ratios} = \frac{1/3}{100/240}, \frac{1/3}{80/240}, \frac{1/3}{60/240} = 0.8, 1.0, 1.33$$

$$J = \frac{(0.8 + 1.0 + 1.33)^2}{3 \times (0.64 + 1.0 + 1.77)} = \frac{9.80}{10.23} = 0.958$$

Not terrible, but backend 3 is overloaded by 33%.

**Example 2:** Weighted round-robin with $w = (100, 80, 60)$:

$$\text{Load ratios} = \frac{100/240}{100/240}, \frac{80/240}{80/240}, \frac{60/240}{60/240} = 1.0, 1.0, 1.0$$

$$J = 1.0 \text{ (perfectly fair)}$$

Correct weights eliminate imbalance entirely.

## 6. Canary Routing Risk (Blast Radius Calculation)

### The Problem

Canary deployments route a percentage of traffic to a new version. What is the
expected number of users affected by a bug in the canary, and how quickly
should the rollback trigger?

### The Formula

Expected affected users in time window $T$:

$$U_{affected} = \lambda \cdot T \cdot p_{canary} \cdot p_{bug}$$

where $p_{canary}$ is the canary traffic percentage and $p_{bug}$ is the
probability of the bug manifesting per request.

Time to detect with error rate monitoring (assuming binomial test):

$$N_{detect} = \frac{z^2 \cdot p_{bug}(1 - p_{bug})}{E^2}$$

where $z$ is the z-score for desired confidence and $E$ is the acceptable
error margin.

$$T_{detect} = \frac{N_{detect}}{\lambda \cdot p_{canary}}$$

### Worked Examples

**Example 1:** 1,000 req/s, 5% canary, bug affects 10% of requests.
Users affected in 1 minute:

$$U_{affected} = 1000 \times 60 \times 0.05 \times 0.10 = 300 \text{ users}$$

**Example 2:** Time to detect the 10% error rate with 95% confidence
($z = 1.96$), margin $E = 0.03$:

$$N_{detect} = \frac{1.96^2 \times 0.1 \times 0.9}{0.03^2} = \frac{0.3457}{0.0009} = 384 \text{ requests}$$

$$T_{detect} = \frac{384}{1000 \times 0.05} = 7.68 \text{ seconds}$$

Users affected before detection:

$$U_{before\_detect} = 1000 \times 7.68 \times 0.05 \times 0.10 = 38.4 \approx 39 \text{ users}$$

With 1% canary instead of 5%: $T_{detect} = 38.4$ s but only $U = 38$ users.
Same blast radius, just slower detection.

## Prerequisites

- Token bucket and leaky bucket algorithms
- Probability distributions (Zipfian, Poisson, binomial)
- Cryptographic algorithm performance characteristics
- Cache replacement policies (LRU, LFU) and hit ratio modeling
- Queueing theory (M/M/1, M/M/c for backend load modeling)
- Statistical hypothesis testing (z-test for anomaly detection)
- Jain's fairness index for load distribution analysis
