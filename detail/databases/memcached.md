# The Mathematics of Memcached — Hashing, Memory Allocation, and Cache Theory

> *Memcached is a distributed hash table with a slab allocator. The math covers consistent hashing ring placement, slab class sizing, LRU eviction probability, cache hit ratio modeling, and memory fragmentation analysis.*

---

## 1. Consistent Hashing (Distributed Systems)

### The Problem

When distributing keys across $N$ servers, naive modular hashing $h(k) \bmod N$ causes massive key redistribution when a server is added or removed. With $K$ total keys, $K \cdot \frac{N-1}{N}$ keys must move.

### The Formula

Consistent hashing maps both servers and keys onto a ring of size $2^{32}$. Each server is placed at $V$ virtual nodes. When a server is added or removed, only $\frac{K}{N}$ keys are expected to move:

$$E[\text{keys moved}] = \frac{K}{N}$$

The load balance factor with $V$ virtual nodes per server:

$$\text{max load} \approx \frac{K}{N} \cdot \left(1 + \frac{c}{\sqrt{V}}\right)$$

where $c \approx 1.6$ for 95th percentile.

### Worked Examples

| Servers | Virtual Nodes | Total Keys | Keys Moved on Add | Max Server Load |
|:---:|:---:|:---:|:---:|:---:|
| 10 | 1 | 1,000,000 | 100,000 | ~260,000 |
| 10 | 50 | 1,000,000 | 100,000 | ~122,600 |
| 10 | 150 | 1,000,000 | 100,000 | ~113,100 |
| 10 | 500 | 1,000,000 | 100,000 | ~107,200 |

With 150 virtual nodes: $\frac{1{,}000{,}000}{10} \cdot (1 + \frac{1.6}{\sqrt{150}}) \approx 113{,}100$ max keys per server.

## 2. Slab Allocator (Memory Management)

### The Problem

Naive malloc/free causes memory fragmentation. Memcached pre-allocates fixed-size chunks in slab classes to eliminate fragmentation at the cost of internal waste.

### The Formula

Slab classes follow a geometric progression. Given growth factor $f$ and minimum chunk size $s_0$:

$$s_i = s_0 \cdot f^i$$

The internal fragmentation (wasted bytes) for an item of size $x$ stored in slab class $i$:

$$\text{waste}(x) = s_i - x \quad \text{where } s_i = \min\{s_j : s_j \geq x\}$$

The expected waste ratio with growth factor $f$:

$$E\left[\frac{\text{waste}}{s_i}\right] = \frac{f - 1}{2f}$$

### Worked Examples

| Growth Factor ($f$) | Expected Waste Ratio | Waste for 200-byte item |
|:---:|:---:|:---:|
| 1.25 | 10.0% | Class=208B, waste=8B (4.0%) |
| 1.50 | 16.7% | Class=216B, waste=16B (7.4%) |
| 2.00 | 25.0% | Class=256B, waste=56B (21.9%) |

Default $f = 1.25$ with $s_0 = 96$ bytes produces classes: 96, 120, 152, 192, 240, 304, ...

Each slab page is 1 MB. Chunks per page for class $i$:

$$\text{chunks}_i = \left\lfloor \frac{1{,}048{,}576}{s_i} \right\rfloor$$

## 3. Cache Hit Ratio (Probability Theory)

### The Problem

Predicting cache effectiveness requires modeling the probability that a requested item is in the cache, given finite memory and item access patterns.

### The Formula

Under an LRU eviction policy with Zipf-distributed access patterns (exponent $\alpha$), the hit ratio for a cache of size $C$ with $N$ total items:

$$H(C, N, \alpha) = \frac{\sum_{i=1}^{C} \frac{1}{i^\alpha}}{\sum_{i=1}^{N} \frac{1}{i^\alpha}}$$

For the common case where $\alpha \approx 1$ (web workloads):

$$H \approx \frac{\ln(C) + \gamma}{\ln(N) + \gamma}$$

where $\gamma \approx 0.5772$ is the Euler-Mascheroni constant.

### Worked Examples

| Cache Size ($C$) | Total Items ($N$) | Zipf $\alpha$ | Hit Ratio |
|:---:|:---:|:---:|:---:|
| 1,000 | 1,000,000 | 0.8 | 62.3% |
| 1,000 | 1,000,000 | 1.0 | 50.0% |
| 10,000 | 1,000,000 | 1.0 | 66.7% |
| 100,000 | 1,000,000 | 1.0 | 83.4% |
| 100,000 | 1,000,000 | 1.2 | 91.8% |

## 4. LRU Eviction Analysis (Queueing Theory)

### The Problem

Under LRU, an item is evicted when the number of distinct items accessed since its last request exceeds the cache capacity. The time-to-eviction depends on the working set size.

### The Formula

The characteristic time $T_C$ is the time window during which exactly $C$ distinct items are accessed. An item with access rate $\lambda_i$ survives eviction if:

$$P(\text{hit}_i) = 1 - e^{-\lambda_i \cdot T_C}$$

For the overall miss rate with $N$ items and Poisson arrivals:

$$\text{Miss Rate} = \frac{\sum_{i=1}^{N} \lambda_i \cdot e^{-\lambda_i \cdot T_C}}{\sum_{i=1}^{N} \lambda_i}$$

### Worked Examples

With total request rate $\Lambda = 10{,}000$ req/s, cache size $C = 50{,}000$, and item $i$ accessed at $\lambda_i = 2$ req/s:

$$T_C \approx \frac{C}{\Lambda} \cdot \frac{1}{1 - H} \approx 10 \text{ seconds}$$

$$P(\text{hit}_i) = 1 - e^{-2 \cdot 10} \approx 1.0$$

An item accessed only $\lambda_j = 0.01$ req/s:

$$P(\text{hit}_j) = 1 - e^{-0.01 \cdot 10} = 1 - e^{-0.1} \approx 0.095$$

## 5. Memory Overhead (Systems Engineering)

### The Problem

Each cached item has metadata overhead beyond the stored key-value pair. Understanding the true cost per item determines effective cache capacity.

### The Formula

Per-item overhead in memcached:

$$\text{item\_size} = 48 + \text{key\_len} + \text{value\_len} + 2 \quad \text{(CAS disabled)}$$

$$\text{item\_size} = 56 + \text{key\_len} + \text{value\_len} + 2 \quad \text{(CAS enabled)}$$

The 48-byte header includes: next/prev pointers (16B), hash chain pointer (8B), flags/exptime/nbytes/nsuffix/clsid (16B), refcount + slabs_clsid (8B).

Effective capacity:

$$\text{items}_{\max} = \frac{M}{\bar{s}_{\text{chunk}}}$$

where $M$ is total memory and $\bar{s}_{\text{chunk}}$ is the average chunk size used.

### Worked Examples

| Memory | Avg Key | Avg Value | Overhead | Items Stored |
|:---:|:---:|:---:|:---:|:---:|
| 1 GB | 32 B | 200 B | 50 B | ~3,800,000 |
| 1 GB | 32 B | 1 KB | 50 B | ~980,000 |
| 1 GB | 100 B | 10 KB | 50 B | ~105,000 |
| 8 GB | 32 B | 200 B | 50 B | ~30,400,000 |

## 6. Multi-Get Optimization (Network Analysis)

### The Problem

Individual get requests incur per-request round-trip overhead. Multi-get batching amortizes network latency.

### The Formula

Time for $n$ individual gets with round-trip time $R$ and per-key processing time $p$:

$$T_{\text{individual}} = n \cdot (R + p)$$

Time for a single multi-get of $n$ keys:

$$T_{\text{multi}} = R + n \cdot p + \frac{n \cdot \bar{v}}{B}$$

where $\bar{v}$ is average value size and $B$ is bandwidth. Speedup:

$$S = \frac{T_{\text{individual}}}{T_{\text{multi}}} \approx \frac{n \cdot R}{R + n \cdot p}$$

### Worked Examples

With $R = 0.5$ ms, $p = 0.01$ ms, $n = 100$ keys:

$$S = \frac{100 \cdot 0.5}{0.5 + 100 \cdot 0.01} = \frac{50}{1.5} = 33.3\times$$

| Batch Size | Individual Time | Multi-Get Time | Speedup |
|:---:|:---:|:---:|:---:|
| 10 | 5.1 ms | 0.6 ms | 8.5x |
| 50 | 25.5 ms | 1.0 ms | 25.5x |
| 100 | 51.0 ms | 1.5 ms | 34.0x |
| 500 | 255.0 ms | 5.5 ms | 46.4x |

## 7. Connection Overhead (Resource Modeling)

Each connection consumes memory for buffers and thread state:

$$M_{\text{conn}} = n_{\text{conn}} \cdot (s_{\text{read\_buf}} + s_{\text{write\_buf}} + s_{\text{thread}})$$

With default 2 KB read + 2 KB write buffers and 8 KB thread overhead:

$$M_{\text{conn}} = n_{\text{conn}} \times 12 \text{ KB}$$

| Connections | Memory Overhead | Fraction of 1 GB Cache |
|:---:|:---:|:---:|
| 100 | 1.2 MB | 0.1% |
| 1,000 | 12 MB | 1.2% |
| 10,000 | 120 MB | 11.7% |
| 50,000 | 600 MB | 58.6% |

At 50,000 connections, over half the memory goes to connection overhead rather than cached data.

## Prerequisites

- Hash functions and modular arithmetic
- Probability distributions (Zipf, Poisson)
- Geometric series and logarithmic approximations
- LRU cache replacement fundamentals
- Network latency and round-trip time concepts
- Memory allocation strategies (slab vs. buddy vs. malloc)
