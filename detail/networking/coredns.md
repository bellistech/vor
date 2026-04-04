# The Mathematics of CoreDNS — Plugin Chains and Caching Theory

> *CoreDNS processes DNS queries through an ordered plugin chain. The mathematics cover plugin execution as a pipeline, cache hit ratio optimization, TTL decay, DNS query latency modeling, and the probabilistic analysis of prefetching strategies.*

---

## 1. Plugin Chain Execution (Pipeline Model)

### The Problem

CoreDNS processes each query through an ordered sequence of plugins. Each plugin can handle the query, pass it to the next, or return immediately. The execution order is fixed at compile time.

### The Formula

For a chain of $k$ plugins, the probability that plugin $i$ handles the query:

$$P_i = p_i \times \prod_{j=1}^{i-1}(1 - p_j)$$

Total expected processing time:

$$E[T] = \sum_{i=1}^{k} P_i \times \left(\sum_{j=1}^{i} t_j\right)$$

Where $t_j$ is the processing time of plugin $j$ and $p_i$ is the probability plugin $i$ handles the query.

### Worked Example

Chain: log (pass-through, 0.1 ms) -> cache (hit rate 80%, 0.05 ms) -> kubernetes (handles 60% of remaining, 2 ms) -> forward (handles rest, 20 ms):

$$E[T] = 0.1 + 0.8 \times 0.05 + 0.2 \times 0.6 \times 2.0 + 0.2 \times 0.4 \times 20.0$$

$$E[T] = 0.1 + 0.04 + 0.24 + 1.6 = 1.98 \text{ ms}$$

Without cache ($p_{\text{cache}} = 0$):

$$E[T_{\text{no cache}}] = 0.1 + 0.6 \times 2.0 + 0.4 \times 20.0 = 9.3 \text{ ms}$$

Speedup from caching:

$$S = \frac{9.3}{1.98} = 4.7\times$$

---

## 2. DNS Cache Theory (Temporal Locality)

### The Problem

The `cache` plugin stores responses by query name, type, and class. Cache effectiveness depends on query distribution and TTL values.

### The Formula

Cache hit ratio under Zipf-distributed queries with parameter $\alpha$ and cache size $C$ out of $N$ unique domains:

$$H(C, N, \alpha) = \frac{\sum_{i=1}^{C} i^{-\alpha}}{\sum_{i=1}^{N} i^{-\alpha}}$$

For the generalized harmonic number $H_N^{(\alpha)} = \sum_{i=1}^N i^{-\alpha}$:

$$H(C, N, \alpha) = \frac{H_C^{(\alpha)}}{H_N^{(\alpha)}}$$

### Worked Examples

DNS queries typically follow Zipf with $\alpha \approx 0.9$:

| Cache Size | Unique Domains | Hit Ratio ($\alpha = 0.9$) |
|:---:|:---:|:---:|
| 100 | 10,000 | 0.52 |
| 1,000 | 10,000 | 0.78 |
| 5,000 | 10,000 | 0.93 |
| 9,984 | 10,000 | 0.998 |

The default CoreDNS cache size of 9,984 entries captures virtually all benefit for typical query distributions.

---

## 3. TTL Management (Decay and Staleness)

### The Problem

Cached DNS records have a Time-To-Live that decreases with age. CoreDNS decrements TTL in responses and must decide when to evict or refresh.

### The Formula

Remaining TTL for a cached record:

$$\text{TTL}_{\text{remaining}} = \text{TTL}_{\text{original}} - (t_{\text{now}} - t_{\text{cached}})$$

Record validity:

$$\text{Valid}(r) = \begin{cases} \text{true} & \text{if } \text{TTL}_{\text{remaining}} > 0 \\ \text{stale} & \text{if } 0 \geq \text{TTL}_{\text{remaining}} > -T_{\text{stale}} \\ \text{expired} & \text{otherwise} \end{cases}$$

With `serve_stale 1h`, $T_{\text{stale}} = 3600$ seconds.

### Expected Cache Refresh Rate

For $n$ cached records with average TTL $\bar{\tau}$:

$$R_{\text{refresh}} = \frac{n}{\bar{\tau}} \text{ queries/second}$$

| Cached Records | Avg TTL | Refresh Rate |
|:---:|:---:|:---:|
| 1,000 | 300s | 3.3 q/s |
| 5,000 | 300s | 16.7 q/s |
| 10,000 | 60s | 166.7 q/s |

---

## 4. Prefetch Optimization (Predictive Caching)

### The Problem

The `prefetch` directive triggers an upstream refresh before TTL expiry for frequently accessed records. The goal is to prevent cache misses for popular domains.

### The Formula

Prefetch condition: record is refreshed if it received at least $h$ hits in window $w$ and TTL is below fraction $f$:

$$\text{Prefetch}(r) = (\text{hits}(r, w) \geq h) \land \left(\frac{\text{TTL}_{\text{remaining}}}{\text{TTL}_{\text{original}}} \leq f\right)$$

Expected upstream query reduction:

$$\Delta Q = \sum_{r \in \text{popular}} \frac{1}{\text{TTL}_r} \times P(\text{miss without prefetch})$$

### Worked Example

With `prefetch 10 1h 20%`: a record with TTL=300s, queried 50 times/hour:

$$\text{Queries per TTL window} = 50 \times \frac{300}{3600} = 4.17$$

Since 4.17 < 10 (threshold), this record would NOT be prefetched. A record queried 200 times/hour:

$$\text{Queries per TTL window} = 200 \times \frac{300}{3600} = 16.7 \geq 10 \implies \text{prefetch}$$

Prefetch triggers at 20% remaining TTL = 60s remaining, saving one miss every 300s.

---

## 5. Kubernetes DNS Query Volume (Service Discovery)

### The Problem

In Kubernetes, CoreDNS resolves service names, pod IPs, and headless service endpoints. Query volume scales with cluster size and inter-service communication.

### The Formula

Expected DNS queries per second for a cluster with $s$ services, $p$ pods, and $r$ requests/pod/sec:

$$Q_{\text{total}} = p \times r \times (1 - H_{\text{cache}}) \times D$$

Where $D$ is the DNS-to-HTTP ratio (typically 0.3-0.5 due to connection reuse and local caching).

### Worked Example

Cluster: 500 pods, 100 services, 50 req/pod/sec, client-side cache hit 70%, DNS ratio 0.4:

$$Q = 500 \times 50 \times (1 - 0.7) \times 0.4 = 3,000 \text{ q/s}$$

CoreDNS pod sizing (each instance handles ~30,000 q/s):

$$\text{Replicas} = \left\lceil \frac{Q}{30{,}000} \times \text{safety factor} \right\rceil = \left\lceil \frac{3{,}000}{30{,}000} \times 2 \right\rceil = 1$$

---

## 6. Forwarding Latency (Upstream Selection)

### The Problem

The `forward` plugin distributes queries across upstream servers using configurable policies. Selection strategy affects both latency and reliability.

### The Formula

Round-robin expected latency with $u$ upstreams, latencies $L_1, L_2, \ldots, L_u$:

$$E[L_{\text{rr}}] = \frac{1}{u} \sum_{i=1}^{u} L_i$$

With health checking (failed servers removed), effective latency improves:

$$E[L_{\text{healthy}}] = \frac{1}{|U_{\text{healthy}}|} \sum_{i \in U_{\text{healthy}}} L_i$$

### Availability

With $u$ independent upstreams, each with availability $a$:

$$A_{\text{system}} = 1 - (1 - a)^u$$

| Upstreams | Individual Availability | System Availability |
|:---:|:---:|:---:|
| 1 | 99.9% | 99.9% |
| 2 | 99.9% | 99.9999% |
| 3 | 99.9% | 99.9999999% |

---

## 7. Zone Transfer Cost (AXFR/IXFR)

### The Problem

The `file` plugin serves authoritative zones and supports zone transfers. Full (AXFR) and incremental (IXFR) transfers have different bandwidth costs.

### The Formula

AXFR transfer size:

$$S_{\text{AXFR}} = \sum_{i=1}^{n} s_i + 2 \times s_{\text{SOA}}$$

Where $n$ is the number of records and $s_i$ is the wire format size of record $i$.

IXFR transfer size (for $d$ changed records):

$$S_{\text{IXFR}} = 2 \times s_{\text{SOA}} + \sum_{j=1}^{d} (s_{j,\text{old}} + s_{j,\text{new}})$$

Bandwidth savings:

$$\text{Savings} = 1 - \frac{S_{\text{IXFR}}}{S_{\text{AXFR}}} = 1 - \frac{2d}{n}$$

### Worked Example

Zone with 10,000 records (avg 100 bytes each), 50 records changed:

$$S_{\text{AXFR}} = 10{,}000 \times 100 = 1 \text{ MB}$$
$$S_{\text{IXFR}} \approx 50 \times 200 = 10 \text{ KB}$$
$$\text{Savings} = 1 - \frac{10}{1{,}000} = 99\%$$

---

## Prerequisites

- queuing-theory, probability, graph-theory, dns-fundamentals
