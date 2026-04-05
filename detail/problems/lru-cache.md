# The Mathematics of LRU Caching -- Eviction Policies and Amortized Guarantees

> *An LRU cache is a bounded dictionary that forgets strategically: every access refreshes an entry's lease on life, while the entry untouched longest is the first to be sacrificed. The elegance lies in achieving constant-time guarantees for both access and eviction through the marriage of hashing and linked structure.*

---

## 1. The Data Structure Invariant

An LRU cache of capacity $k$ maintains two co-operating structures over a key-value universe $U$:

1. **Hash map** $H : K \to \text{Node}^*$ mapping keys to node references.
2. **Doubly linked list** $L$ of nodes ordered by recency, with sentinel head $s_h$ and sentinel tail $s_t$.

At all times the following invariant holds:

$$|H| = |L| \leq k$$

where $|L|$ counts only data nodes (excluding sentinels). The list encodes a total order $\prec$ on cached keys such that:

$$\text{key}_1 \prec \text{key}_2 \iff \text{key}_1 \text{ was accessed more recently than } \text{key}_2$$

The most-recently-used node sits at $s_h.\text{next}$ and the least-recently-used at $s_t.\text{prev}$.

**Sentinel advantage.** Let $n$ be a data node with predecessor $p$ and successor $q$ in $L$. Removal is:

$$p.\text{next} \leftarrow q, \quad q.\text{prev} \leftarrow p$$

Without sentinels, both $p$ and $q$ may be null, requiring four conditional branches. With sentinels, the assignments are unconditional. This eliminates $O(1)$-constant but error-prone branching and is the same technique used in the Linux kernel's `list.h`.

### Operation Costs

**Get(key):**

$$T_{\text{get}} = T_{\text{hash-lookup}} + T_{\text{list-remove}} + T_{\text{list-insert-front}} = O(1) + O(1) + O(1) = O(1)$$

**Put(key, value):**

For the insertion case with eviction ($|H| = k$):

$$T_{\text{put}} = T_{\text{hash-lookup}} + T_{\text{evict}} + T_{\text{hash-delete}} + T_{\text{node-create}} + T_{\text{hash-insert}} + T_{\text{list-insert-front}}$$

Each term is $O(1)$, giving:

$$T_{\text{put}} = O(1)$$

For the update case (key exists), there is no eviction or allocation:

$$T_{\text{put-update}} = T_{\text{hash-lookup}} + T_{\text{list-remove}} + T_{\text{list-insert-front}} = O(1)$$

## 2. Competitive Analysis and the Cache Hit Ratio

Let $\sigma = (r_1, r_2, \ldots, r_n)$ be a sequence of $n$ requests from a universe of $N$ distinct keys, served by a cache of size $k$. Define the **hit ratio**:

$$h(\sigma, k) = \frac{|\{i : r_i \in C_i\}|}{n}$$

where $C_i$ is the cache state just before request $r_i$.

**Theorem (Sleator & Tarjan, 1985).** For any request sequence $\sigma$, the number of cache misses under LRU with cache size $k$ satisfies:

$$\text{LRU}_k(\sigma) \leq k \cdot \text{OPT}_k(\sigma)$$

where $\text{OPT}_k$ is the optimal offline algorithm (Belady's MIN). That is, LRU is **$k$-competitive**.

Under the **independent reference model** (IRM) where each request is drawn i.i.d. from a probability distribution $\{p_1, p_2, \ldots, p_N\}$ with $p_1 \geq p_2 \geq \cdots \geq p_N$, the characteristic time $t_c(k)$ -- the expected number of requests between consecutive accesses to the $k$-th most popular item -- satisfies:

$$t_c(k) = \frac{1}{\sum_{i=1}^{k} p_i}$$

For a **Zipf distribution** with parameter $\alpha$ where $p_i \propto i^{-\alpha}$, the miss rate for large $N$ scales as:

$$1 - h \sim \begin{cases} k^{1-\alpha} & \text{if } \alpha > 1 \\ (\ln k)^{-1} & \text{if } \alpha = 1 \\ \text{const} & \text{if } \alpha < 1 \end{cases}$$

This explains why LRU caches are effective in practice: real-world access patterns (web pages, database rows, file blocks) typically follow Zipf with $\alpha \in [0.8, 1.2]$, concentrating most requests on a small fraction of keys.

### Comparison with Other Eviction Policies

How does LRU compare to alternative online policies? Let $\text{MISS}_k^P(\sigma)$ denote the number of misses for policy $P$ with cache size $k$ on sequence $\sigma$.

| Policy | Competitive Ratio | Description |
|--------|-------------------|-------------|
| LRU | $k$ | Evict least recently used |
| FIFO | $k$ | Evict oldest inserted (ignores re-access) |
| CLOCK | $k$ | Approximation of LRU via circular buffer + use bit |
| LFU | $\Omega(k)$ | Evict least frequently used; poor on distribution shifts |
| Random | $k$ | Evict uniformly at random |
| OPT (Belady) | $1$ (offline) | Evict the key whose next access is furthest in the future |

All deterministic online policies have competitive ratio at least $k$ (Borodin et al., 1995). Randomised marking algorithms achieve $O(\log k)$ competitive ratio against an oblivious adversary:

$$\mathbb{E}[\text{MISS}_k^{\text{RAND-MARK}}(\sigma)] \leq 2 H_k \cdot \text{OPT}_k(\sigma)$$

where $H_k = \sum_{i=1}^{k} \frac{1}{i} \approx \ln k + \gamma$ is the $k$-th harmonic number and $\gamma \approx 0.5772$ is the Euler-Mascheroni constant.

### The Stack Distance Property

LRU has a unique monotonicity property called **inclusion** or the **stack distance property**: for any sequence $\sigma$, if $C_k$ is the set of items cached by LRU with cache size $k$, then:

$$C_k(\sigma) \subseteq C_{k+1}(\sigma) \quad \text{for all } k$$

This means increasing cache size never increases misses. FIFO does **not** have this property -- it suffers from **Belady's anomaly** where a larger cache can produce more misses on certain sequences. Concretely, consider $\sigma = (1, 2, 3, 4, 1, 2, 5, 1, 2, 3, 4, 5)$ with cache size 3 vs. 4 under FIFO: size 3 yields 9 misses while size 4 yields 10 misses.

The stack distance $d(r_i)$ of a request $r_i$ is its position in the LRU stack at the time of the request ($\infty$ if not present). A request hits if and only if $d(r_i) \leq k$. The distribution of stack distances fully characterises LRU performance:

$$h(k) = \Pr[d \leq k] = \sum_{j=1}^{k} f(j)$$

where $f(j) = \Pr[d = j]$ is the stack distance probability mass function.

## 3. Amortized Space and the Slot-Reuse Optimisation

In the naive implementation, each `put` of a new key allocates a node and each eviction discards one. Over $m$ operations the total allocations are bounded by $m$, but the garbage collector (or manual free) must reclaim evicted nodes.

The **slot-reuse optimisation** (used in the Rust solution above) avoids allocation after the cache is full. Nodes live in a contiguous array $A[0..k+1]$ (indices 0 and 1 are sentinels). On eviction, the victim's slot is overwritten in place:

$$A[\text{lru\_idx}].\text{key} \leftarrow \text{new\_key}, \quad A[\text{lru\_idx}].\text{val} \leftarrow \text{new\_val}$$

This gives an **amortized allocation cost** of:

$$T_{\text{alloc}}^{\text{amortized}} = \frac{k + 2}{m} \to 0 \text{ as } m \to \infty$$

After the initial $k$ insertions fill the cache, zero further heap allocations occur. This is critical in systems programming (OS page caches, database buffer pools) where allocation latency is unacceptable.

**Memory layout.** With slot reuse, all nodes occupy a contiguous `Vec<Node>` of exactly $k + 2$ elements. Each node is:

$$\text{sizeof(Node)} = \text{sizeof(key)} + \text{sizeof(val)} + 2 \cdot \text{sizeof(usize)}$$

Total cache memory (excluding the hash map) is:

$$M_{\text{list}} = (k + 2) \cdot \text{sizeof(Node)}$$

The hash map adds $O(k)$ entries of size $\text{sizeof(key)} + \text{sizeof(usize)}$ each, so total space is $\Theta(k)$.

### Hash Map Resizing and Worst-Case Bounds

The $O(1)$ time bound on `get` and `put` is **amortized** due to hash map resizing. When the load factor $\lambda = n / m$ (where $n$ is the number of entries and $m$ is the number of buckets) exceeds a threshold (typically $\lambda_{\max} = 0.75$), the map doubles its bucket array and rehashes all entries in $O(n)$ time.

Over a sequence of $n$ insertions starting from an empty map with initial bucket count $m_0$, the total rehashing cost is:

$$\sum_{i=0}^{\lfloor \log_2(n/m_0) \rfloor} 2^i \cdot m_0 = m_0 \cdot (2^{\lfloor \log_2(n/m_0) \rfloor + 1} - 1) \leq 2n$$

So the amortized cost per insertion remains $O(1)$. For an LRU cache, the hash map never grows beyond $k$ entries, so at most $\lfloor \log_2 k \rfloor$ resizes occur during the cache warm-up phase, and none thereafter.

For applications requiring **worst-case** $O(1)$ per operation, cuckoo hashing or linear probing with Robin Hood hashing can be used. Alternatively, pre-allocating the hash map to capacity $k$ at construction time eliminates all resizes:

$$m_0 = \lceil k / \lambda_{\max} \rceil$$

This is the approach used in production systems like memcached and Redis.

### Cache-Oblivious Considerations

The linked list structure has poor spatial locality: nodes may be scattered across the heap, causing cache line misses on traversal. While LRU only ever accesses the head and tail of the list (not interior nodes), the slot-reuse approach with a contiguous array improves hardware cache behaviour. For a cache line of size $B$ bytes:

$$\text{Cache lines touched per operation} = O(1) \quad \text{(slot reuse, contiguous array)}$$

versus potentially $O(1)$ distinct cache lines with pointer-based nodes -- the same asymptotic count, but with higher constant factors due to indirection and TLB pressure.

---

## Prerequisites

- **Hash maps:** average-case $O(1)$ lookup, insert, delete; understanding of load factor, chaining vs. open addressing, and amortized resize cost.
- **Doubly linked lists:** $O(1)$ insert and remove given a direct node pointer/reference; sentinel node pattern.
- **Pointer/reference semantics:** or index-based simulation for ownership-strict languages (Rust's borrow checker forbids naive pointer-based doubly linked lists without `unsafe`).
- **Competitive analysis:** online algorithms, competitive ratio, adversarial sequences, the distinction between oblivious and adaptive adversaries.
- **Probability:** independent reference model, Zipf/power-law distributions, harmonic numbers $H_k$.
- **Amortized analysis:** aggregate method, accounting method, or potential method for bounding sequences of operations.

## Complexity

| Aspect | Bound | Notes |
|--------|-------|-------|
| `get` time | $O(1)$ average | Hash lookup + two pointer swaps |
| `put` time | $O(1)$ average | Hash lookup + possible eviction + two pointer swaps |
| `put` time (worst case) | $O(n)$ | Hash map resize; amortized $O(1)$ over $n$ operations |
| Space | $\Theta(k)$ | $k$ = capacity; nodes + hash map entries |
| Allocations after warm-up | $0$ | With slot-reuse optimisation |
| Competitive ratio vs. OPT | $k$ | Sleator-Tarjan bound; tight |
| Miss rate (Zipf $\alpha > 1$) | $O(k^{1-\alpha})$ | Decreases polynomially with cache size |
| Hash map resizes (lifetime) | $\leq \lfloor \log_2 k \rfloor$ | Only during warm-up; zero after cache full |
| Competitive ratio (randomised) | $O(\log k)$ | Randomised marking; $2 H_k \cdot \text{OPT}$ |
| Stack distance hit condition | $d(r_i) \leq k$ | Monotone: larger $k$ never hurts (inclusion property) |
| Belady's anomaly | Does not apply | LRU satisfies the inclusion property; FIFO does not |
