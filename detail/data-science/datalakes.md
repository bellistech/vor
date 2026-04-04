# The Mathematics of Data Lakes — Storage Optimization, Partitioning Theory, and Query Planning

> *Modern data lake architectures encode deep mathematical principles: partitioning strategies optimize for information-theoretic pruning efficiency, compaction algorithms solve bin-packing problems to minimize file fragmentation, and query planners use cost models rooted in statistics and combinatorial optimization to select execution strategies across distributed storage layers.*

---

## 1. Partition Pruning Efficiency (Information Theory)
### The Problem
Partitioning divides data into segments so queries only read relevant partitions. The effectiveness depends on how well the partition key correlates with query predicates. Information theory quantifies this correlation.

### The Formula
Partition selectivity for a predicate $P$ on a table with $K$ partitions:

$$S = \frac{|\{k : k \in K, P(k) = \text{true}\}|}{|K|}$$

The information gain from partition pruning:

$$IG = H(\text{data}) - H(\text{data} \mid \text{partition})$$

$$H(\text{data}) = \log_2(N) \quad \text{(bits to identify a row among } N \text{)}$$

$$H(\text{data} \mid \text{partition}) = \sum_{k=1}^{K} \frac{n_k}{N} \log_2(n_k)$$

Where $n_k$ is the number of rows in partition $k$.

Data skipped by pruning:

$$\text{Bytes skipped} = (1 - S) \times \text{Total bytes}$$

### Worked Examples
**Example**: 10TB event table, 365 daily partitions, roughly 27.4GB per day. Query filters to 7 days.

$$S = \frac{7}{365} = 0.0192$$

$$\text{Bytes read} = 0.0192 \times 10\text{TB} = 192\text{GB}$$

$$\text{Bytes skipped} = 9.808\text{TB} \quad \text{(98.1\% reduction)}$$

Without partitioning: full 10TB scan.

Information gain per row: $\log_2(N) - \log_2(N/365) = \log_2(365) \approx 8.5$ bits.

Adding a secondary bucket partition on `user_id` with 16 buckets:

$$S_{composite} = \frac{7}{365} \times \frac{1}{16} = 0.0012$$

$$\text{Bytes read} = 0.0012 \times 10\text{TB} = 12\text{GB} \quad \text{(99.88\% reduction)}$$

## 2. File Compaction as Bin Packing (Combinatorial Optimization)
### The Problem
Small files degrade query performance due to metadata overhead and reduced I/O efficiency. Compaction merges small files into target-sized files, which is a variant of the bin-packing problem.

### The Formula
Given $n$ small files with sizes $s_1, s_2, \ldots, s_n$ and target file size $T$, minimize the number of output files $m$:

$$\min m \quad \text{subject to} \quad \sum_{j \in B_i} s_j \leq T, \quad i = 1, \ldots, m$$

$$\bigcup_{i=1}^{m} B_i = \{1, \ldots, n\}, \quad B_i \cap B_j = \emptyset$$

Lower bound:

$$m \geq \left\lceil \frac{\sum_{i=1}^{n} s_i}{T} \right\rceil$$

Query overhead from small files (metadata cost per file is $c$):

$$\text{Overhead} = n \times c$$

After compaction:

$$\text{Overhead}_{new} = m \times c \approx \frac{\sum s_i}{T} \times c$$

### Worked Examples
**Example**: 10,000 small files averaging 1MB each (streaming ingestion). Target: 128MB files.

$$m_{lower} = \left\lceil \frac{10{,}000 \times 1}{128} \right\rceil = 79 \text{ files}$$

First-Fit Decreasing heuristic typically achieves $m \leq 1.22 \times m_{lower} + 1 \approx 97$ files.

Metadata overhead reduction: $\frac{10{,}000}{79} \approx 127\times$ improvement.

If metadata cost is 10ms per file per query:
- Before: $10{,}000 \times 10\text{ms} = 100$ seconds
- After: $79 \times 10\text{ms} = 0.79$ seconds

## 3. Cost-Based Query Optimization (Statistics)
### The Problem
Query engines choose execution plans (scan order, join strategy, predicate pushdown) based on table statistics. The optimizer estimates the cost of each plan using cardinality estimation.

### The Formula
Cardinality estimation for a filter predicate on column $C$ with value $v$:

$$\hat{n} = N \times \text{sel}(C = v)$$

Selectivity estimates:
- Equality: $\text{sel}(C = v) = \frac{1}{\text{NDV}(C)}$ (uniform assumption)
- Range: $\text{sel}(a \leq C \leq b) = \frac{b - a}{\max(C) - \min(C)}$ (uniform)
- With histogram (equi-depth, $B$ buckets): $\text{sel}(C = v) = \frac{1}{f_v}$ if $v$ is a frequent value, else $\frac{N - \sum f_i}{B \times \text{NDV}_{non\_freq}}$

Join cardinality (inner join on key $k$):

$$|R \bowtie_k S| \approx \frac{|R| \times |S|}{\max(\text{NDV}_R(k), \text{NDV}_S(k))}$$

Total query cost:

$$C_{total} = C_{scan} + C_{filter} + C_{join} + C_{agg} + C_{sort}$$

### Worked Examples
**Example**: Table with 100M rows, column `status` has NDV=5, column `amount` ranges [0, 10000].

Query: `WHERE status = 'active' AND amount > 5000`

$$\text{sel}(\text{status}) = \frac{1}{5} = 0.2$$
$$\text{sel}(\text{amount} > 5000) = \frac{10000 - 5000}{10000 - 0} = 0.5$$

Assuming independence:
$$\hat{n} = 100M \times 0.2 \times 0.5 = 10M \text{ rows}$$

If `status` is skewed (80% are 'active'):
$$\hat{n}_{corrected} = 100M \times 0.8 \times 0.5 = 40M$$

The uniform assumption underestimates by 4x, leading to suboptimal plan selection.

## 4. Schema Evolution Compatibility (Type Theory)
### The Problem
Table formats must determine when a schema change is backward-compatible (readers with old schema can read new data) and forward-compatible (readers with new schema can read old data).

### The Formula
Define a subtyping relation $\leq$ on schemas:

$$S_1 \leq S_2 \iff \text{every valid instance of } S_1 \text{ is also valid under } S_2$$

Compatibility rules for schema operations:

$$\text{Add optional column}: S \leq S' \quad (\text{backward compatible})$$
$$\text{Drop optional column}: S' \leq S \quad (\text{forward compatible})$$
$$\text{Widen type } T_1 \to T_2: T_1 \leq T_2 \quad (\text{e.g., int} \to \text{long})$$
$$\text{Rename column}: \text{incompatible without ID mapping}$$

Iceberg uses column IDs to handle renames:

$$\text{compat}(S_1, S_2) = \forall \text{id} \in S_1 \cap S_2: \text{type}_{S_1}(\text{id}) \leq \text{type}_{S_2}(\text{id})$$

### Worked Examples
**Example**: Schema evolution sequence:

$S_1$: `{order_id: long, amount: int, status: string}`

$S_2$: `{order_id: long, amount: long, status: string, discount: double?}`

Changes: widen `amount` (int -> long), add optional `discount`.

$$\text{int} \leq \text{long} \quad \checkmark$$
$$\text{add optional} \quad \checkmark$$

$S_1 \leq S_2$: backward compatible. Old readers can read $S_2$ data by ignoring `discount` and narrowing `amount` (with overflow check).

$S_3$: `{order_id: long, total: long, status: string, discount: double?}`

Rename `amount` -> `total`: without column IDs, this is a breaking change (column `amount` disappears, `total` appears). With Iceberg column IDs (both map to id=2), it is compatible.

## 5. Time Travel via Snapshot Isolation (Database Theory)
### The Problem
Table formats implement time travel through snapshot isolation, maintaining a log of immutable snapshots. The space-time trade-off determines how long historical queries are possible versus storage cost.

### The Formula
Storage cost of $n$ snapshots with data change rate $\delta$ per snapshot:

$$\text{Storage}(n) = S_0 + \sum_{i=1}^{n} \delta_i$$

For copy-on-write (COW): $\delta_i = |R_i| \times \bar{s}$ where $R_i$ is rewritten files.

For merge-on-read (MOR): $\delta_i = |D_i| \times \bar{s}_{log}$ where $D_i$ is delta/log files.

Space amplification factor:

$$A = \frac{\text{Total storage (all snapshots)}}{\text{Latest snapshot size}} = \frac{S_0 + \sum \delta_i}{S_0 + \sum \delta_i - \text{garbage}}$$

With snapshot expiration after $k$ versions:

$$\text{Storage}_{bounded} \leq S_0 + k \times \max(\delta_i)$$

### Worked Examples
**Example**: 500GB table, daily batch updates affecting 5% of data (COW).

Daily delta: $\delta = 0.05 \times 500\text{GB} = 25\text{GB}$

After 30 days (retaining all snapshots):
$$\text{Storage} = 500 + 30 \times 25 = 1250\text{GB}$$

Space amplification: $A = 1250/500 = 2.5\times$

With snapshot expiration at 7 days:
$$\text{Storage} \leq 500 + 7 \times 25 = 675\text{GB} \quad (A = 1.35\times)$$

For MOR (delta files are ~10% of COW rewrites):
$$\delta_{MOR} = 0.1 \times 25 = 2.5\text{GB}$$

After 30 days: $500 + 30 \times 2.5 = 575\text{GB}$ ($A = 1.15\times$).

Trade-off: MOR saves storage but incurs read-time merge cost.

## 6. Z-Ordering and Space-Filling Curves (Computational Geometry)
### The Problem
Z-ordering (used by Delta Lake's OPTIMIZE ZORDER) interleaves bits of multiple column values to create a linear ordering that preserves multi-dimensional locality, improving predicate pushdown for multi-column filters.

### The Formula
The Z-value for a point $(x, y)$ with $b$-bit coordinates:

$$Z(x, y) = \sum_{i=0}^{b-1} (x_i \cdot 2^{2i} + y_i \cdot 2^{2i+1})$$

The Hilbert curve provides better locality:

$$\text{Locality ratio: } \frac{L_{Hilbert}}{L_{Z-order}} \approx 0.7 \text{ (Hilbert is 30\% better)}$$

For $d$ dimensions and $N$ total points, the expected number of files touched by a range query covering fraction $f$ of each dimension:

$$\text{Files}_{Z-order} \approx N^{1/d} \times f^{(d-1)/d}$$
$$\text{Files}_{no\_order} \approx N \times f^d$$

### Worked Examples
**Example**: 1000 Parquet files, 2 filter dimensions (date, region), each covering 10% ($f = 0.1$).

Without Z-ordering:
$$\text{Files} = 1000 \times 0.1^2 = 10 \text{ files}$$

With Z-ordering:
$$\text{Files} \approx 1000^{1/2} \times 0.1^{1/2} = 31.6 \times 0.316 = 10 \text{ files}$$

For 3 dimensions (date, region, category), $f = 0.1$:

Without: $1000 \times 0.1^3 = 1$ file (already very selective)

With Z-order: $1000^{1/3} \times 0.1^{2/3} = 10 \times 0.215 = 2.15$ files

Z-ordering provides the most benefit when selectivity is moderate (10-50% per dimension) and the number of dimensions is 2-4.

## Prerequisites
- information-theory, combinatorial-optimization, statistics, type-theory, database-theory, computational-geometry, space-filling-curves
