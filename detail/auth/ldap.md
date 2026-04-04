# The Mathematics of LDAP — Tree Traversal, Filter Evaluation, and Search Complexity

> *LDAP search operations traverse a hierarchical Directory Information Tree where performance depends on filter selectivity, index coverage, and subtree depth. Understanding the combinatorics of compound filters and the cost model of indexed versus unindexed searches is essential for tuning directories that serve millions of entries.*

---

## 1. DIT Traversal Complexity (Graph Theory)

### The Problem

An LDAP search with scope `sub` must visit every entry in the subtree rooted at the search base. Without indexes, the server performs a full scan. The cost depends on the branching factor and depth of the DIT.

### The Formula

For a DIT with branching factor $b$ (average children per node) and depth $d$, the total entries in a subtree rooted at level $l$ is:

$$N_{subtree}(l) = \sum_{k=0}^{d-l} b^k = \frac{b^{d-l+1} - 1}{b - 1}$$

For a full-tree search from the root ($l = 0$):

$$N_{total} = \frac{b^{d+1} - 1}{b - 1}$$

### Worked Examples

| Depth ($d$) | Branching ($b$) | Total Entries | Subtree at $l=2$ |
|:---:|:---:|:---:|:---:|
| 3 | 10 | 1,111 | 111 |
| 4 | 10 | 11,111 | 1,111 |
| 3 | 50 | 132,651 | 2,601 |
| 5 | 5 | 3,906 | 156 |

An unindexed `sub` scope search at the root of a tree with $b=50, d=3$ examines all 132,651 entries. Narrowing the base to $l=2$ reduces the scan to 2,601 entries -- a 50x improvement.

---

## 2. Search Filter Selectivity (Probability)

### The Problem

Compound LDAP filters combine equality, substring, and presence tests with Boolean operators. The directory evaluates filters entry-by-entry (or via index intersection), and the cost depends on how many entries survive each filter component.

### The Formula

For independent filter components with individual selectivities $s_i$ (fraction of entries matching):

**AND filter** -- the intersection shrinks the candidate set:

$$s_{AND} = \prod_{i=1}^{k} s_i$$

**OR filter** -- the union expands it (inclusion-exclusion):

$$s_{OR} = 1 - \prod_{i=1}^{k} (1 - s_i)$$

**NOT filter** -- the complement:

$$s_{NOT} = 1 - s_i$$

Expected result count from $N$ entries:

$$E[results] = N \cdot s_{filter}$$

### Worked Examples

Given $N = 100{,}000$ entries:

| Filter | Selectivity | Expected Results |
|:---|:---:|:---:|
| `(uid=jdoe)` | $10^{-5}$ | 1 |
| `(objectClass=person)` | 0.80 | 80,000 |
| `(ou=Engineering)` | 0.05 | 5,000 |
| `(&(objectClass=person)(ou=Engineering))` | $0.80 \times 0.05 = 0.04$ | 4,000 |
| `(&#124;(ou=Engineering)(ou=Sales))` | $1 - (0.95)(0.92) = 0.126$ | 12,600 |

### Optimal Filter Ordering

Place the most selective (lowest $s_i$) component first in AND filters. If $s_1 = 0.001$ and $s_2 = 0.5$, evaluating $s_1$ first reduces the candidate set to $0.1\%$ before testing $s_2$:

$$\text{Cost}_{s_1\text{ first}} = N + N \cdot s_1 = N(1 + s_1)$$
$$\text{Cost}_{s_2\text{ first}} = N + N \cdot s_2 = N(1 + s_2)$$

---

## 3. Index Cost Model (Information Retrieval)

### The Problem

LDAP servers maintain B-tree indexes on attributes. An indexed equality lookup is $O(\log n)$ whereas an unindexed scan is $O(n)$. The break-even point determines when adding an index is worthwhile.

### The Formula

Unindexed search cost (full scan with filter evaluation):

$$C_{scan} = N \cdot c_{eval}$$

Indexed search cost (B-tree lookup plus entry fetch):

$$C_{index} = \log_B(N) \cdot c_{io} + R \cdot c_{fetch}$$

where $B$ is the B-tree branching factor (typically 100--500), $R$ is the result count, $c_{io}$ is disk I/O cost, and $c_{fetch}$ is the per-entry retrieval cost.

Index is faster when:

$$\log_B(N) \cdot c_{io} + R \cdot c_{fetch} < N \cdot c_{eval}$$

### Worked Examples

With $N = 1{,}000{,}000$, $B = 256$, $c_{io} = c_{eval} = 1$ (normalized):

| Query | Result ($R$) | $C_{scan}$ | $C_{index}$ | Speedup |
|:---|:---:|:---:|:---:|:---:|
| `(uid=jdoe)` | 1 | 1,000,000 | $\lceil\log_{256}(10^6)\rceil + 1 = 4$ | 250,000x |
| `(ou=Engineering)` | 50,000 | 1,000,000 | $3 + 50{,}000 = 50{,}003$ | 20x |
| `(objectClass=person)` | 900,000 | 1,000,000 | $3 + 900{,}000 = 900{,}003$ | 1.1x |

High-cardinality attributes (uid, mail) benefit enormously from indexing. Low-cardinality attributes (objectClass) provide negligible improvement.

---

## 4. Connection and Bind Cost (Networking)

### The Problem

Each LDAP operation begins with a TCP connection and optionally a TLS handshake followed by a bind. Connection pooling amortizes this overhead across multiple operations.

### The Formula

Single-operation cost without pooling:

$$T_{single} = T_{tcp} + T_{tls} + T_{bind} + T_{op}$$

With connection pooling over $n$ operations:

$$T_{pooled} = \frac{T_{tcp} + T_{tls} + T_{bind}}{n} + T_{op}$$

Overhead reduction:

$$\text{Savings} = \frac{(n-1)(T_{tcp} + T_{tls} + T_{bind})}{n \cdot T_{single}} \times 100\%$$

### Worked Examples

Typical latencies: $T_{tcp} = 1\text{ms}$, $T_{tls} = 5\text{ms}$, $T_{bind} = 2\text{ms}$, $T_{op} = 1\text{ms}$:

| Operations ($n$) | Without Pooling | With Pooling | Savings |
|:---:|:---:|:---:|:---:|
| 1 | 9 ms | 9 ms | 0% |
| 10 | 90 ms | 18 ms | 80% |
| 100 | 900 ms | 108 ms | 88% |
| 1,000 | 9,000 ms | 1,008 ms | 89% |

---

## 5. Cache Hit Rate and TTL (Queuing Theory)

### The Problem

SSSD and other LDAP clients cache directory entries locally. The cache hit rate determines how many queries reach the LDAP server. Setting the TTL too low floods the server; too high serves stale data.

### The Formula

For a Zipf-distributed access pattern with exponent $\alpha$ over $N$ unique entries and cache size $C$:

$$P_{hit} \approx \frac{C^{1-\alpha}}{N^{1-\alpha}} \quad \text{for } \alpha \neq 1$$

Expected server query rate from $\lambda$ total queries/sec:

$$\lambda_{server} = \lambda \cdot (1 - P_{hit})$$

### Worked Examples

With $N = 50{,}000$ entries, $\alpha = 1.2$, $\lambda = 500$ queries/sec:

| Cache Size ($C$) | Hit Rate | Server Load (queries/sec) |
|:---:|:---:|:---:|
| 1,000 | 72.5% | 138 |
| 5,000 | 86.3% | 69 |
| 10,000 | 91.2% | 44 |
| 25,000 | 96.5% | 18 |

---

## 6. Replication Convergence (Distributed Systems)

### The Problem

Multi-master LDAP replication (389DS MMR, OpenLDAP syncrepl) must propagate changes across $n$ replicas. Convergence time depends on replication topology and change rate.

### The Formula

For a star topology with one supplier and $n-1$ consumers, replication of a single change:

$$T_{converge} = T_{detect} + T_{transfer} + T_{apply}$$

For chained replication (daisy chain), worst-case convergence through $n-1$ hops:

$$T_{chain} = (n - 1) \cdot (T_{detect} + T_{transfer} + T_{apply})$$

With change rate $\mu$ changes/sec and replication bandwidth $\beta$ changes/sec per link:

$$\text{Stable if } \mu < \beta \quad \text{(queue length } \to \infty \text{ if } \mu \geq \beta\text{)}$$

### Worked Examples

$T_{detect} = 100\text{ms}$, $T_{transfer} = 5\text{ms}$, $T_{apply} = 10\text{ms}$:

| Topology | Replicas ($n$) | Convergence |
|:---|:---:|:---:|
| Star | 4 | 115 ms |
| Chain | 4 | 345 ms |
| Star | 8 | 115 ms |
| Chain | 8 | 805 ms |

Star topology scales better -- convergence is constant regardless of replica count.

---

## Prerequisites

- tree-structures, boolean-algebra, probability, big-o-notation, btree-indexes, networking-fundamentals
