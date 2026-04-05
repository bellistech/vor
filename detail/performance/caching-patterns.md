# The Mathematics of Caching — Hit Ratios, Eviction, and Probabilistic Structures

> *Caching performance is governed by access pattern distributions, working set theory, and the mathematics of eviction policies. Understanding these foundations enables precise cache sizing, optimal TTL selection, and probabilistic techniques that dramatically reduce miss rates.*

---

## 1. Cache Hit Ratio Mathematics (Cold Start to Steady State)

### The Problem

What hit ratio can we expect from a cache of size $C$ serving $N$ unique items with access frequency distribution $f(i)$?

### The Formula

The **hit ratio** $h$ is the probability that a requested item is in the cache:

$$h = \sum_{i=1}^{C} f(i)$$

Where items are ranked by access frequency: $f(1) \geq f(2) \geq \ldots \geq f(N)$, and $\sum_{i=1}^{N} f(i) = 1$.

For a perfectly optimal cache (always evicts the least popular item), the hit ratio equals the sum of the top-$C$ access frequencies.

**Cold start**: Initially the cache is empty ($h = 0$). After $k$ requests with $m$ unique items seen, the expected hit ratio grows as:

$$h(k) \approx 1 - \frac{E[\text{new items at request } k]}{1}$$

For uniform random access over $N$ items after $k$ requests:

$$E[\text{items seen}] = N \left(1 - \left(1 - \frac{1}{N}\right)^k\right)$$

$$h(k) \approx 1 - \frac{N(1 - 1/N)^k}{1} \quad \text{(probability next request is new)}$$

**Steady state**: Reached when the cache is full and access patterns stabilize. For LRU with Zipf-distributed access:

$$h_{\text{LRU}} \approx 1 - \left(\frac{C}{N}\right)^{1 - 1/\alpha}$$

Where $\alpha$ is the Zipf exponent (typically 0.8-1.2 for web workloads).

### Worked Examples

$N = 100{,}000$ unique items, cache size $C = 10{,}000$, Zipf distribution with $\alpha = 1.0$:

Optimal hit ratio: $h = \sum_{i=1}^{10000} \frac{1/i}{\sum_{j=1}^{100000} 1/j} = \frac{H_{10000}}{H_{100000}} = \frac{9.21}{11.51} \approx 80.0\%$

With only $C = 1{,}000$: $h = H_{1000} / H_{100000} = 6.91 / 11.51 \approx 60.0\%$

10x the cache size improved hit rate from 60% to 80% — not a proportional improvement, because the "long tail" of infrequent items contributes progressively less.

---

## 2. LRU and LFU Eviction Analysis (Policy Comparison)

### The Problem

When should you use LRU (Least Recently Used) vs LFU (Least Frequently Used)? What are their optimal workloads?

### The Formula

**LRU** evicts the item with the oldest last-access time. LRU is a stack algorithm (increasing cache size never decreases hit ratio — Belady's inclusion property).

LRU **competitive ratio**: LRU is $k$-competitive with the optimal offline algorithm (OPT), where $k = C$ (cache size). This means:

$$\text{misses}_{\text{LRU}} \leq C \cdot \text{misses}_{\text{OPT}} + C$$

**LFU** evicts the item with the fewest accesses. LFU excels when access frequencies are stable. However, pure LFU suffers from **cache pollution**: once-popular items accumulate high counts and persist even when they become cold.

**Comparison**:

| Workload | LRU | LFU | Winner |
|---|---|---|---|
| Recency-biased (web browsing) | Good | Poor | LRU |
| Frequency-biased (database hot set) | Moderate | Good | LFU |
| Scan-resistant | Poor (scans evict hot items) | Good | LFU |
| Mixed/bursty | Good | Poor (stale counts) | LRU |

**Modern policies** (TinyLFU, W-TinyLFU used in Caffeine/Ristretto): Combine a small LRU "window" with a frequency-based main cache. New items enter the window; only items that prove popular are promoted to the main cache. This provides:
- LRU-like recency handling
- LFU-like frequency awareness
- Scan resistance

### Worked Examples

Database query cache with stable hot set (20% of queries = 80% of traffic):

LRU: A full table scan reads 100,000 keys sequentially, evicting the hot set. After the scan, hit ratio drops to 0% and must rebuild. Recovery time = $C / \lambda$ where $\lambda$ = request rate for hot keys.

LFU: Scan reads have low frequency counts. Hot keys with high counts survive the scan. Hit ratio barely affected.

W-TinyLFU: Scan items enter the small window, never make it to the main cache (low frequency). Hot keys remain. Best of both worlds.

---

## 3. Working Set Theory (Denning's Model)

### The Problem

How much cache do we need to hold the "working set" of data that a system actively uses?

### The Formula

Denning's **working set** $W(t, \tau)$ at time $t$ with window $\tau$ is the set of distinct items accessed in the interval $(t - \tau, t]$.

The **working set size** $|W(t, \tau)|$ is the number of distinct items in the working set.

For stationary random access with $N$ total items and access probability $p_i$ for item $i$:

$$E[|W(t, \tau)|] = \sum_{i=1}^{N} \left(1 - (1 - p_i)^{\tau \lambda}\right)$$

Where $\lambda$ is the request rate (requests per unit time).

**Cache sizing rule**: Set $C \geq E[|W(t, \tau)|]$ where $\tau$ is the TTL. This ensures the cache can hold all items that will be accessed within one TTL period.

**Thrashing threshold**: If $C < |W|$, the system thrashes — constantly evicting items that will soon be needed again. Hit ratio drops sharply below a critical cache size.

### Worked Examples

E-commerce product catalog: $N = 500{,}000$ products, $\lambda = 10{,}000$ req/s, TTL $\tau = 300$s.

Top 1000 products account for 60% of traffic ($p_i \approx 0.0006$ each). Items 1001-10000 account for 30% ($p_i \approx 0.000033$ each). Remaining 490,000 account for 10%.

$E[|W|] \approx 1000 \times (1 - (1-0.0006)^{3{,}000{,}000}) + 9000 \times (1 - (1-0.000033)^{3{,}000{,}000}) + 490{,}000 \times (1 - (1-0.0000002)^{3{,}000{,}000})$

$\approx 1000 \times 1.0 + 9000 \times 1.0 + 490{,}000 \times 0.451 = 1000 + 9000 + 220{,}990 = 230{,}990$ items.

Cache should hold approximately 231,000 items. At 2 KB per item: approximately 450 MB.

---

## 4. Zipf Distribution in Access Patterns (Power Law)

### The Problem

Real-world access patterns follow power laws. How does this affect cache design?

### The Formula

**Zipf's Law**: The frequency of the $i$-th most popular item is:

$$f(i) = \frac{1/i^\alpha}{\sum_{j=1}^{N} 1/j^\alpha} = \frac{1/i^\alpha}{H_{N,\alpha}}$$

Where $\alpha$ is the Zipf exponent and $H_{N,\alpha}$ is the generalized harmonic number.

For $\alpha = 1$ (common in web workloads):
- The most popular item gets $1/H_N$ fraction of all requests
- The top $k$ items get $H_k / H_N$ fraction
- $H_N = \ln(N) + \gamma \approx \ln(N) + 0.5772$

**Cache hit ratio under Zipf** with cache size $C$ and $N$ items:

$$h(C) = \frac{H_{C,\alpha}}{H_{N,\alpha}} \approx \frac{\ln(C)}{\ln(N)} \quad \text{(for } \alpha = 1 \text{)}$$

This logarithmic relationship means doubling cache size provides diminishing returns.

### Worked Examples

Web application: $N = 1{,}000{,}000$ pages, $\alpha = 1.0$.

| Cache size $C$ | Fraction of $N$ | Hit ratio $H_C / H_N$ |
|---|---|---|
| 100 | 0.01% | 5.19 / 14.39 = 36.1% |
| 1,000 | 0.1% | 6.91 / 14.39 = 48.0% |
| 10,000 | 1% | 9.21 / 14.39 = 64.0% |
| 100,000 | 10% | 11.51 / 14.39 = 80.0% |
| 500,000 | 50% | 13.07 / 14.39 = 90.8% |

Key insight: Caching 10% of items captures 80% of requests. The 80/20 rule (Pareto principle) is a manifestation of Zipf with $\alpha \approx 1$.

Higher $\alpha$ (more skewed): smaller cache achieves higher hit ratio. Lower $\alpha$ (more uniform): need larger cache for same hit ratio.

---

## 5. Probabilistic Early Expiration — XFetch Algorithm (Stampede Prevention)

### The Problem

When a cached item expires, multiple concurrent requests trigger simultaneous recomputation. How does probabilistic early refresh prevent this?

### The Formula

**XFetch**: Each request computes whether to refresh based on:

$$\text{refresh if } \Delta \cdot \beta \cdot \ln(\text{rand}()) + \text{age} > \text{TTL}$$

Where:
- $\Delta$ = time to recompute the value (last measured)
- $\beta$ = tuning parameter (default 1.0)
- $\text{rand}()$ = uniform random in $(0, 1)$
- $\text{age}$ = time since last refresh
- $\text{TTL}$ = expiration time

Since $\ln(\text{rand}()) \leq 0$, the expression $\Delta \cdot \beta \cdot \ln(\text{rand}())$ is always negative. As age approaches TTL, the overall expression becomes positive — triggering refresh.

**Expected time of first refresh** with $\lambda$ requests/second:

$$E[T_{\text{refresh}}] \approx \text{TTL} - \frac{\Delta \cdot \beta}{\ln(\lambda \cdot \Delta \cdot \beta)}$$

The higher the request rate $\lambda$, the earlier (relative to TTL) the first refresh occurs — automatically scaling the early-refresh window with popularity.

### Worked Examples

TTL = 300s, $\Delta$ = 0.5s (recompute time), $\beta$ = 1.0, $\lambda$ = 100 req/s.

At age = 295s (5s before expiry):
$P(\text{refresh per request}) = P(0.5 \times 1.0 \times \ln(U) + 295 > 300) = P(\ln(U) > 10) \approx 0$ (extremely unlikely since $\max(\ln(U)) = 0$).

Wait — this means at age = 295, $0.5 \times 1.0 \times \ln(U) + 295 > 300$, so $\ln(U) > 10$ which is impossible. The formula triggers when age is very close to TTL.

Correcting: $0.5 \cdot \ln(U) + 299.5 > 300 \implies \ln(U) > 1$ which is impossible. So at age 299.5: $0.5 \cdot \ln(U) + 299.5 > 300 \implies \ln(U) > 1$, still impossible.

At age = 300 exactly: $0.5 \cdot \ln(U) + 300 > 300 \implies \ln(U) > 0$, impossible. TTL expired, all requests refresh.

At age = 300.2: $0.5 \cdot \ln(U) + 300.2 > 300 \implies \ln(U) > -0.4 \implies U > 0.67$. $P \approx 33\%$. With 100 req/s, expected first refresh within ~30ms. Only one request actually recomputes.

---

## 6. Bloom Filter for Negative Caching (False Positive Rate)

### The Problem

When most lookups are for items that do not exist (negative lookups), checking the cache and database for every miss is wasteful. Can we efficiently filter out non-existent keys?

### The Formula

A **Bloom filter** is a space-efficient probabilistic data structure that can test set membership with:
- False positives: possible (reports "maybe in set" when it is not)
- False negatives: impossible (never reports "not in set" when it is)

For $n$ items inserted into a Bloom filter with $m$ bits and $k$ hash functions:

**False positive rate**:

$$p = \left(1 - e^{-kn/m}\right)^k$$

**Optimal number of hash functions**:

$$k_{\text{opt}} = \frac{m}{n} \ln 2 \approx 0.693 \frac{m}{n}$$

**Required bits per element** for target false positive rate $p$:

$$\frac{m}{n} = -\frac{\ln p}{(\ln 2)^2} \approx -1.44 \ln p$$

### Worked Examples

Database with 10 million valid user IDs. Cache receives queries for arbitrary IDs — 90% of lookups are for non-existent IDs.

Target false positive rate: 1%.

$m/n = -1.44 \times \ln(0.01) = -1.44 \times (-4.605) = 6.63$ bits per element.

Total memory: $10{,}000{,}000 \times 6.63 / 8 \approx 8.3$ MB.

$k_{\text{opt}} = 0.693 \times 6.63 = 4.6 \approx 5$ hash functions.

Without Bloom filter: 10,000 req/s, 90% negative = 9,000 DB queries for non-existent keys per second.

With Bloom filter: 9,000 negative queries filtered, only $9{,}000 \times 0.01 = 90$ false positives reach DB. Reduction: 99%.

**Counting Bloom filter**: Uses counters instead of bits, supporting deletions. Each counter = 4 bits (to handle overflow): $m_{\text{counting}} = 4 \times m_{\text{standard}}$.

---

## 7. Cache Coherence Protocols (Distributed Invalidation)

### The Problem

With multiple cache tiers (L1 per-process, L2 distributed), how do we keep caches coherent when data changes?

### The Formula

**Write-invalidate protocol**: On write, invalidate all cached copies.

Invalidation propagation time for $n$ caches with network delay $d$:
- **Sequential notification**: $T = n \times d$
- **Parallel notification**: $T = d$ (single round)
- **Gossip notification**: $T = O(\log n) \times d$

**Staleness window**: Time between data update and cache invalidation completion. During this window, some caches serve stale data.

$$P(\text{stale read}) = 1 - e^{-\lambda_r \times T_{\text{invalidation}}}$$

Where $\lambda_r$ = read rate during invalidation period.

**Consistency models for distributed caches**:
- **Strong**: Write waits for all invalidations to complete before returning. High latency, no stale reads.
- **Eventual**: Write returns immediately, invalidations propagate asynchronously. Low latency, possible stale reads during propagation.
- **Bounded staleness**: TTL ensures maximum staleness = TTL. No active invalidation needed.

### Worked Examples

10 application servers, each with L1 cache. Data update at $t=0$.

Parallel invalidation ($d = 2\text{ms}$): All caches invalidated by $t = 2\text{ms}$. At 10,000 req/s per server, approximately $10 \times 10{,}000 \times 0.002 = 200$ stale reads.

Gossip invalidation ($d = 2\text{ms}$, $\log_2 10 \approx 3.3$ rounds): Completed by $t = 6.6\text{ms}$. Approximately 660 stale reads. But less network overhead (each node contacts $O(\log n)$ peers instead of all peers).

TTL-based (TTL = 60s, no active invalidation): Maximum staleness = 60 seconds. At 100,000 total req/s, up to 6,000,000 stale reads in the worst case (all reads during TTL). Trade-off: zero invalidation infrastructure, but higher staleness.

---

## Prerequisites

- Probability distributions (Zipf, exponential, uniform)
- Logarithmic and harmonic series
- Basic understanding of hash functions
- Familiarity with caching architectures (L1/L2/L3)

## Complexity

| Concept | Space | Time | Key Formula |
|---|---|---|---|
| LRU cache | $O(C)$ | $O(1)$ get/put | $k$-competitive with OPT |
| Bloom filter | $O(n)$ bits | $O(k)$ per lookup | $p = (1 - e^{-kn/m})^k$ |
| Zipf hit ratio | N/A | N/A | $h = H_C / H_N$ |
| Working set | $O(\|W\|)$ | N/A | $\sum (1 - (1-p_i)^{\tau\lambda})$ |
| XFetch refresh | $O(1)$ extra | $O(1)$ per check | $\Delta\beta\ln(U) + \text{age} > \text{TTL}$ |
