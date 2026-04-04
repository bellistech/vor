# The Mathematics of SQL -- Query Optimization and Index Theory

> *Every query plan is a proof that a particular physical execution strategy satisfies the logical specification of the query, and the optimizer's job is to find the proof with minimal cost.*

---

## 1. B-Tree Index Complexity (Data Structures)

### The Problem

B-tree indexes are the workhorse of relational databases. Understanding their structure explains why indexed lookups are fast, why column order in composite indexes matters, and when a sequential scan wins.

### The Formula

A B-tree of order $m$ with $n$ keys has height:

$$h = \lceil \log_m(n + 1) \rceil$$

The number of disk I/Os for a point lookup is $h + 1$ (traverse tree + fetch heap tuple). For a range scan returning $k$ rows:

$$\text{I/O}_{\text{range}} = h + \lceil k / f \rceil$$

where $f$ is the number of keys per leaf page (fanout). The crossover point where a sequential scan beats an index scan occurs when the selectivity $\sigma$ exceeds:

$$\sigma_{\text{crossover}} \approx \frac{c_{\text{random}}}{c_{\text{seq}}} \cdot \frac{1}{n_{\text{pages}}}$$

where $c_{\text{random}}/c_{\text{seq}}$ is the random-to-sequential I/O cost ratio (typically 4-10 for SSDs, 50-100 for HDDs).

### Worked Examples

Table `orders` with $n = 10{,}000{,}000$ rows, page size 8 KiB, 100 keys per leaf page ($f = 100$), B-tree order $m = 200$.

Height: $h = \lceil \log_{200}(10^7) \rceil = \lceil 7/2.3 \rceil = \lceil 3.04 \rceil = 4$

Point lookup: 4 + 1 = 5 I/Os. At 0.1ms per SSD random read: 0.5ms.

Range scan for 1,000 rows: $4 + \lceil 1000/100 \rceil = 4 + 10 = 14$ I/Os = 1.4ms.

Crossover: with $c_{\text{random}}/c_{\text{seq}} = 5$ and $n_{\text{pages}} = 100{,}000$: $\sigma_{\text{crossover}} = 5/100{,}000 = 0.005\%$. So for queries returning more than 500 rows (0.005% of 10M), sequential scan may win -- but only on HDD. On NVMe, the ratio drops and indexes remain effective at higher selectivities.

---

## 2. Window Function Evaluation (Partition Algebra)

### The Problem

Window functions compute values across sets of rows related to the current row. The database must partition, sort, and compute frame-relative aggregates efficiently.

### The Formula

For a window function $f$ with partition key $P$ and order key $O$, the computation over $n$ rows with $k$ distinct partitions of average size $\bar{p} = n/k$:

$$T_{\text{window}} = T_{\text{sort}} + T_{\text{scan}}$$

$$T_{\text{sort}} = O(n \log n) \quad \text{(global sort by } P, O\text{)}$$

For cumulative aggregates (ROWS UNBOUNDED PRECEDING), each partition requires $O(\bar{p})$ incremental computation. For sliding windows of size $w$:

$$T_{\text{sliding}} = O(n \cdot \min(w, \log w))$$

The $\log w$ factor applies when using a balanced tree to maintain the window (for non-invertible aggregates like MAX). For invertible aggregates (SUM, COUNT), the window slides in $O(1)$ per row:

$$T_{\text{invertible}} = O(n)$$

### Worked Examples

Query: 7-day moving average over 1,000,000 daily records across 100 departments.

Sort: $O(10^6 \log 10^6) = O(10^6 \times 20) = 2 \times 10^7$ comparisons.

Sliding SUM (invertible): $O(10^6)$ -- add new day, subtract day leaving window.

If instead computing sliding MEDIAN (non-invertible): need two heaps or order-statistic tree: $O(10^6 \log 7) \approx 2.8 \times 10^6$ operations.

Total: sort dominates at $O(n \log n)$. Using a pre-sorted index on `(department, date)` eliminates the sort entirely, reducing to $O(n)$.

---

## 3. Cost-Based Query Optimization (Dynamic Programming)

### The Problem

Given a query joining $k$ tables, the optimizer must find the join order and physical operators (hash join, nested loop, merge join) that minimize estimated cost. This is equivalent to finding the optimal parenthesization of a chain of operations.

### The Formula

The number of possible join orderings for $k$ tables without cross-joins is the Catalan number:

$$C_k = \frac{1}{k+1}\binom{2k}{k} = \frac{(2k)!}{(k+1)! \cdot k!}$$

With commutativity (left/right swap), the search space doubles: $2^k \cdot C_k$.

The optimizer estimates cost using cardinality estimates. For a join $R \bowtie_\theta S$:

$$|R \bowtie S| = \frac{|R| \cdot |S|}{\max(\text{ndistinct}(R.\text{key}), \text{ndistinct}(S.\text{key}))}$$

The independence assumption gives selectivity of conjunctive predicates:

$$\sigma(p_1 \wedge p_2) = \sigma(p_1) \cdot \sigma(p_2)$$

### Worked Examples

5-table join: $C_5 = 42$ orderings, with commutativity: $2^5 \times 42 = 1{,}344$ plans. Feasible for exhaustive search.

10-table join: $C_{10} = 16{,}796$, with commutativity: $2^{10} \times 16{,}796 \approx 17$ million plans. PostgreSQL uses genetic algorithm (GEQO) above `geqo_threshold` (default 12 tables).

Cardinality estimate: `orders` (1M rows, 50K distinct user_ids) JOIN `users` (50K rows):

$$|R \bowtie S| = \frac{10^6 \times 5 \times 10^4}{\max(5 \times 10^4, 5 \times 10^4)} = \frac{5 \times 10^{10}}{5 \times 10^4} = 10^6$$

This makes sense: each user has ~20 orders on average, join preserves all orders.

---

## 4. Recursive CTE Termination (Fixed-Point Theory)

### The Problem

Recursive CTEs compute a fixed point: they iterate until no new rows are produced. Proving termination requires showing the intermediate result sets form a monotonically growing chain that reaches a fixed point in finite steps.

### The Formula

A recursive CTE computes the least fixed point of operator $T$:

$$R_0 = \text{base query}$$
$$R_{i+1} = R_i \cup T(R_i)$$

The computation terminates when $R_{i+1} = R_i$ (fixed point). For a graph with $|V|$ vertices and $|E|$ edges, the maximum depth is:

$$d_{\max} = |V| - 1 \quad \text{(longest simple path)}$$

The total work for a transitive closure is:

$$W = \sum_{i=0}^{d_{\max}} |R_i \setminus R_{i-1}| \cdot |E| / |V|$$

In the worst case (complete graph): $O(|V|^2)$ output rows, $O(|V|^3)$ work.

### Worked Examples

Organization hierarchy: 10,000 employees, maximum management depth of 8 levels. The recursive CTE produces at most 8 iterations.

$R_0$: 1 row (CEO). $R_1$: ~10 VPs. $R_2$: ~50 directors. ... $R_8$: cumulative 10,000 rows.

Total work: $\sum_{i=0}^{8} |R_i \setminus R_{i-1}|$ new rows per iteration, each requiring a join against `employees`. With an index on `manager_id`: $O(n \log n)$ total. Without index: $O(n \times d_{\max} \times n) = O(n^2 d)$ -- catastrophic for large $n$.

For graph traversal (social network, 1M users, average 150 edges): without cycle detection, the recursion diverges. The `WHERE target != ALL(path)` guard limits paths to simple paths, ensuring termination in at most $|V|$ iterations.

---

## 5. Partition Pruning (Set Theory)

### The Problem

Partitioned tables divide data into disjoint subsets. The optimizer must determine which partitions can be skipped based on query predicates, reducing I/O proportionally.

### The Formula

For a table $T$ partitioned into $\{P_1, P_2, \ldots, P_k\}$ by range on column $c$, where $P_i$ covers range $[l_i, u_i)$, a query with predicate $c \in [a, b]$ touches:

$$\text{partitions scanned} = |\{P_i : [l_i, u_i) \cap [a, b] \neq \emptyset\}|$$

The I/O reduction factor is:

$$\text{speedup} = \frac{k}{|\text{partitions scanned}|}$$

For hash partitioning with $k$ partitions and equality predicate $c = v$:

$$\text{partition} = h(v) \mod k$$

Only 1 partition is scanned, giving speedup $= k$.

### Worked Examples

Table `events` with 2 billion rows, partitioned by month (24 partitions over 2 years). Query: `WHERE created_at BETWEEN '2024-07-01' AND '2024-09-30'`.

Partitions scanned: July, August, September = 3 out of 24.

Speedup: $24/3 = 8\times$. Each partition has $\approx 83M$ rows, so the query scans 250M rows instead of 2B.

If further sub-partitioned by region (4 regions), and the query adds `AND region = 'us-east'`: $3 \times 1 = 3$ sub-partitions out of $24 \times 4 = 96$. Speedup: $96/3 = 32\times$, scanning ~62.5M rows.

---

## 6. Cardinality Estimation Error Propagation (Statistics)

### The Problem

Query optimizer decisions depend on cardinality estimates. Errors compound through multiple joins, causing exponentially worse plan choices. Understanding error propagation explains why the optimizer sometimes picks catastrophically bad plans.

### The Formula

If the true cardinality of a single-table predicate is $C$ and the estimate has multiplicative error $e$, so $\hat{C} = C \cdot e$, then after $k$ joins with independent estimation errors $e_1, e_2, \ldots, e_k$:

$$\hat{C}_{\text{final}} = C_{\text{true}} \cdot \prod_{i=1}^{k} e_i$$

The log-error is additive:

$$\log(\hat{C}_{\text{final}} / C_{\text{true}}) = \sum_{i=1}^{k} \log(e_i)$$

If each $\log(e_i)$ is drawn from a distribution with mean $\mu$ and variance $\sigma^2$, the expected log-error after $k$ joins:

$$E[|\log \text{error}|] = O(\sqrt{k} \cdot \sigma) \quad \text{(if } \mu = 0\text{)}$$

### Worked Examples

A 5-table join where each cardinality estimate is off by a factor of 3 (either $3\times$ over or $3\times$ under):

Worst case: all errors in the same direction: $3^5 = 243\times$ error.

The optimizer thinks a join produces 100 rows but it actually produces 24,300. It picks a nested loop join (optimal for 100 rows) instead of a hash join, causing the query to run 100x slower than the optimal plan.

This is why `ANALYZE` (updating statistics) and multi-column statistics (`CREATE STATISTICS`) are critical for multi-join queries.

---

## Prerequisites

- B-tree data structures and balanced tree properties
- Computational complexity (Big-O notation, amortized analysis)
- Combinatorics (Catalan numbers, permutations)
- Fixed-point theory (monotone operators, Kleene's theorem)
- Set theory (partitions, intersection, disjointness)
- Probability and statistics (selectivity estimation, independence assumption, error propagation)
- Dynamic programming (optimal substructure in query planning)
