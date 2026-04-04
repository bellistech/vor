# The Mathematics of Cloud DNS — Resolution, Routing, and Availability

> *DNS is a distributed database with probabilistic routing: weighted records become discrete probability distributions, latency-based routing solves nearest-neighbor problems, and health-check failover models system reliability as a Markov chain with absorbing states.*

---

## 1. DNS Resolution Chain (Graph Traversal)

### Recursive Resolution as Tree Walk

A DNS query traverses a hierarchy of nameservers:

$$T_{resolve} = T_{cache\_check} + \sum_{i=1}^{d} T_{ns_i} \times \mathbb{1}[\text{cache miss at level } i]$$

Where $d$ is the delegation depth (typically 3-4: root, TLD, authoritative, possible CNAME).

### Cache Hit Probability

$$P(\text{cache hit}) = 1 - e^{-\lambda \times TTL}$$

Where $\lambda$ = query rate for that record. For a popular domain at 100 queries/sec with TTL=300:

$$P(\text{hit}) = 1 - e^{-100 \times 300} \approx 1.0$$

For a rare domain at 0.001 queries/sec:

$$P(\text{hit}) = 1 - e^{-0.001 \times 300} = 1 - e^{-0.3} \approx 0.26$$

### Effective Resolution Latency

$$\bar{T}_{resolve} = P_{hit} \times T_{cache} + (1 - P_{hit}) \times T_{full}$$

| TTL | Query Rate | Cache Hit Rate | Avg Latency |
|:---:|:---:|:---:|:---:|
| 60s | 10/s | 99.998% | ~0.1ms |
| 300s | 1/s | 99.99% | ~0.1ms |
| 300s | 0.01/s | 95.0% | ~2.6ms |
| 60s | 0.001/s | 5.8% | ~47ms |

---

## 2. Weighted Routing (Discrete Probability)

### Traffic Distribution

Given records $\{r_1, \ldots, r_k\}$ with weights $\{w_1, \ldots, w_k\}$:

$$P(r_i) = \frac{w_i}{\sum_{j=1}^{k} w_j}$$

### Expected Traffic per Endpoint

For total queries $Q$:

$$E[\text{queries to } r_i] = Q \times P(r_i) = Q \times \frac{w_i}{\sum w_j}$$

### Canary Deployment Example

Weights: primary=90, canary=10:

$$P(\text{canary}) = \frac{10}{100} = 0.10$$

For 1,000,000 daily queries:

$$E[\text{canary queries}] = 100{,}000$$
$$\text{Std dev} = \sqrt{Q \times p \times (1-p)} = \sqrt{10^6 \times 0.1 \times 0.9} = 300$$

### Confidence Interval

95% of the time, canary receives:

$$100{,}000 \pm 1.96 \times 300 = [99{,}412, 100{,}588]$$

---

## 3. Latency-Based Routing (Nearest Neighbor)

### Optimization Problem

Latency-based routing minimizes client-to-region latency:

$$\text{Region}(c) = \arg\min_{r \in \text{Regions}} L(c, r)$$

Where $L(c, r)$ = measured latency from client $c$'s network to region $r$.

### Latency Measurement Model

AWS measures latency from resolver networks to regions:

$$L(c, r) = L_{network}(c.resolver, r) + \epsilon$$

Where $\epsilon$ accounts for measurement noise.

### Expected Global Latency

$$\bar{L} = \sum_{c \in \text{clients}} \frac{Q_c}{Q_{total}} \times \min_{r} L(c, r)$$

### Comparison: Geolocation vs Latency

| Metric | Geolocation | Latency-Based |
|:---|:---|:---|
| Decision basis | Client IP geography | Measured RTT |
| Optimality | Approximate | Near-optimal |
| Update frequency | Static mapping | Dynamic probing |
| Failure mode | Mislocated clients | Stale measurements |

---

## 4. Health Check Reliability (Markov Chains)

### Health State Model

A health check has two states: Healthy (H) and Unhealthy (U).

Transition probabilities per check interval:

$$P(H \to U) = p_{fail}$$
$$P(U \to H) = p_{recover}$$

### Failure Detection Time

With failure threshold $k$ (consecutive failures needed):

$$E[T_{detect}] = k \times T_{interval}$$

For $k=3$ and $T_{interval}=10s$:

$$E[T_{detect}] = 30\text{s}$$

### False Positive Probability

Probability of $k$ consecutive false positives:

$$P(\text{false alarm}) = p_{false}^k$$

For $p_{false} = 0.01$ and $k = 3$:

$$P(\text{false alarm}) = 0.01^3 = 10^{-6}$$

### Multi-Region Availability

With $n$ regions, each with availability $A$, and failover routing:

$$A_{system} = 1 - \prod_{i=1}^{n}(1 - A_i)$$

| Regions | Per-Region Availability | System Availability | Downtime/Year |
|:---:|:---:|:---:|:---:|
| 1 | 99.9% | 99.9% | 8.76 hours |
| 2 | 99.9% | 99.9999% | 31.5 seconds |
| 3 | 99.9% | 99.9999999% | 0.03 seconds |

---

## 5. TTL Optimization (Cache Theory)

### The TTL Trade-Off

$$\text{Stale probability} = P(\text{change during TTL}) = 1 - e^{-\mu \times TTL}$$

Where $\mu$ = rate of record changes.

### Query Volume vs TTL

Authoritative query load:

$$Q_{auth} = Q_{total} \times (1 - P_{cache\_hit})$$

$$Q_{auth} \approx Q_{total} \times e^{-\lambda \times TTL}$$

| TTL | Cache Hit Rate | Auth Queries (per 1M) | Propagation Delay |
|:---:|:---:|:---:|:---:|
| 30s | 95% | 50,000 | 30s max |
| 60s | 97% | 30,000 | 60s max |
| 300s | 99.5% | 5,000 | 5 min max |
| 3600s | 99.97% | 300 | 1 hour max |

### Optimal TTL

Minimize cost function:

$$C(TTL) = \alpha \times Q_{auth}(TTL) + \beta \times T_{stale}(TTL)$$

$$\frac{dC}{d(TTL)} = -\alpha \lambda Q_{total} e^{-\lambda \cdot TTL} + \beta \mu e^{-\mu \cdot TTL} = 0$$

---

## 6. DNSSEC Validation (Cryptographic Chain)

### Chain of Trust

DNSSEC forms a cryptographic chain from root to leaf:

$$\text{Valid}(R) = \text{Verify}(R.RRSIG, R.data, ZSK) \wedge \text{Verify}(ZSK, DNSKEY.RRSIG, KSK) \wedge \text{DS}(KSK) \in \text{Parent}$$

### Signature Verification Cost

Each RRSIG verification requires one public key operation:

$$T_{verify} = T_{ECDSA} \approx 0.1\text{ms} \quad \text{(P-256)}$$
$$T_{verify} = T_{RSA} \approx 0.5\text{ms} \quad \text{(2048-bit)}$$

### Response Size Impact

| Record Type | Without DNSSEC | With DNSSEC | Overhead |
|:---|:---:|:---:|:---:|
| A record | ~44 bytes | ~300 bytes | ~7x |
| AAAA record | ~56 bytes | ~312 bytes | ~5.6x |
| MX record | ~60 bytes | ~400 bytes | ~6.7x |

### Key Rollover Schedule

$$T_{rollover} = T_{propagation} + T_{TTL_{max}} + T_{safety}$$

Typical ZSK rollover: 90 days. KSK rollover: 1-2 years.

---

## 7. DNS Query Cost Model (Cloud Economics)

### AWS Route 53 Pricing

$$C_{monthly} = C_{zone} \times N_{zones} + C_{query} \times Q_{millions}$$

$$C_{monthly} = \$0.50 \times N + \$0.40 \times Q_{std} + \$0.60 \times Q_{latency/geo}$$

### Cost Optimization via TTL

Higher TTLs reduce query volume to authoritative servers:

$$Q_{billed} = Q_{total} \times (1 - P_{cache\_hit})$$

Savings from doubling TTL:

$$\Delta C = C_{query} \times Q_{total} \times (e^{-\lambda \cdot TTL} - e^{-\lambda \cdot 2 \cdot TTL})$$

| Monthly Queries | TTL=60s | TTL=300s | Savings |
|:---:|:---:|:---:|:---:|
| 100M | $40.00 | $8.00 | $32.00 |
| 1B | $400.00 | $80.00 | $320.00 |

---

*Cloud DNS transforms name resolution into a programmable traffic management layer. Weighted distributions become probability functions, latency routing solves optimization problems, and health-check failover achieves five-nines availability through redundancy mathematics.*

## Prerequisites

- Probability distributions (discrete, exponential)
- Markov chains and state transitions
- Basic optimization (minimizing cost functions)
- Cryptographic signature verification concepts

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| DNS cache lookup | $O(1)$ hash | $O(n)$ entries |
| Recursive resolution | $O(d)$ hops | $O(1)$ |
| Weighted selection | $O(k)$ records | $O(k)$ |
| Health check evaluation | $O(1)$ per check | $O(n)$ state |
| DNSSEC validation | $O(d)$ signatures | $O(d)$ keys |
| TTL expiry scan | $O(n)$ entries | $O(n)$ |
