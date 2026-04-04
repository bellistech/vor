# The Mathematics of Hive — Query Optimization and Storage Theory

> *Hive translates SQL into distributed execution plans. The mathematics covers cost-based optimization, partition pruning selectivity, columnar compression ratios, and join strategy selection based on cardinality estimates.*

---

## 1. Cost-Based Optimizer (Cardinality Estimation)

### The Problem

Given table statistics, how does Hive estimate the number of rows produced by each operator to choose the optimal join order and strategy?

### The Formula

Filter selectivity for equality predicate on column $c$:

$$\sigma_{c=v} = \frac{1}{NDV(c)}$$

Where $NDV(c)$ = number of distinct values in column $c$.

Range predicate selectivity:

$$\sigma_{c \in [a,b]} = \frac{b - a}{\max(c) - \min(c)}$$

Conjunctive predicates (independence assumption):

$$\sigma_{p_1 \land p_2} = \sigma_{p_1} \times \sigma_{p_2}$$

Output cardinality after filter:

$$|R_{out}| = |R_{in}| \times \sigma$$

Join output cardinality:

$$|R \bowtie S| = \frac{|R| \times |S|}{\max(NDV(R.k), NDV(S.k))}$$

### Worked Examples

| Table Size | NDV(join key) | Filter NDV | Filtered Rows | Join Output |
|:---:|:---:|:---:|:---:|:---:|
| R=10M, S=1M | R:100K, S:50K | R filtered to 1M | 1M | 1M*1M/100K = 10K |
| R=100M, S=10M | R:1M, S:500K | None | 100M, 10M | 100M*10M/1M = 1B |
| R=50M, S=100K | R:10K, S:10K | R filtered to 5M | 5M | 5M*100K/10K = 50M |

---

## 2. Partition Pruning (Search Space Reduction)

### The Problem

How much data does partition pruning eliminate, and what is its impact on query performance?

### The Formula

Without pruning (full scan):

$$D_{scan} = \sum_{p=1}^{P} |partition_p|$$

With pruning on predicate matching $k$ partitions:

$$D_{pruned} = \sum_{p \in \text{matched}} |partition_p|$$

Pruning ratio:

$$\eta_{prune} = 1 - \frac{k}{P}$$

Speedup (assuming I/O bound):

$$S = \frac{P}{k}$$

### Worked Examples

| Total Partitions | Matched | Data Total | Data Scanned | Speedup |
|:---:|:---:|:---:|:---:|:---:|
| 365 (daily, 1yr) | 7 (1 week) | 10 TB | 192 GB | 52x |
| 365 | 30 (1 month) | 10 TB | 822 GB | 12x |
| 8760 (hourly, 1yr) | 24 (1 day) | 50 TB | 137 GB | 365x |

---

## 3. ORC Storage Model (Columnar Compression)

### The Problem

How much storage does ORC save compared to row-oriented formats, and what determines compression ratios?

### The Formula

Row-oriented storage:

$$S_{row} = N \times \sum_{i=1}^{C} w_i$$

Where $w_i$ is the width of column $i$.

Columnar storage with compression:

$$S_{col} = \sum_{i=1}^{C} N \times w_i \times (1 - r_i)$$

Where $r_i$ is the compression ratio for column $i$ (depends on data entropy).

For dictionary-encoded columns:

$$S_{dict} = NDV \times w + N \times \lceil \log_2(NDV) \rceil / 8$$

Run-length encoding for sorted/repeated values:

$$S_{rle} = R \times (w + \lceil \log_2(L_{max}) \rceil)$$

Where $R$ = number of runs, $L_{max}$ = maximum run length.

### Worked Examples

| Column Type | Rows | Raw Size | NDV | Encoded Size | Ratio |
|:---:|:---:|:---:|:---:|:---:|:---:|
| String (country) | 100M | 800 MB | 200 | 25 MB + 100 MB idx | 6.4x |
| Int (user_id) | 100M | 400 MB | 10M | 160 MB (delta) | 2.5x |
| Double (amount) | 100M | 800 MB | 50M | 640 MB (minimal) | 1.25x |
| Boolean (flag) | 100M | 100 MB | 2 | 12.5 MB (bitmap) | 8x |

---

## 4. Join Strategy Selection (Cost Comparison)

### The Problem

When should Hive use map join (broadcast), sort-merge join, or bucket map join?

### The Formula

Map join cost:

$$C_{map} = |S| \times E + |R|$$

Where $|S|$ is the small table broadcast to $E$ executors, $|R|$ is the large table scanned.

Feasibility constraint:

$$|S| \leq M_{executor} \times f_{join}$$

Sort-merge join cost:

$$C_{smj} = |R| + |S| + (|R| + |S|) \times \log_B(|R| + |S|)$$

Bucket map join (tables bucketed on join key with same bucket count):

$$C_{bmj} = |R| + |S| \quad \text{(no shuffle, no sort)}$$

Feasibility: both tables must be bucketed by the join key into $b$ buckets where $b_R \mod b_S = 0$ or vice versa.

### Worked Examples

| Left | Right | Strategy | Shuffle Cost | Network |
|:---:|:---:|:---:|:---:|:---:|
| 500 GB | 50 MB | Map join | 0 | 50MB * 100 exec |
| 500 GB | 100 GB | Sort-merge | 600 GB shuffle | High |
| 500 GB | 100 GB | Bucket map (32b) | 0 | 0 |

---

## 5. Bucketing Theory (Hash Distribution)

### The Problem

How does bucketing distribute data, and what determines bucket count choice?

### The Formula

Bucket assignment:

$$b(key) = \text{hash}(key) \mod B$$

Expected rows per bucket (uniform distribution):

$$E[|bucket_i|] = \frac{N}{B}$$

Bucket file size:

$$S_{bucket} = \frac{S_{partition}}{B}$$

Optimal bucket count (target bucket size 128-256 MB):

$$B_{optimal} = \left\lceil \frac{S_{partition}}{S_{target}} \right\rceil$$

For bucket map join, bucket counts must be compatible:

$$B_R \mod B_S = 0 \quad \text{or} \quad B_S \mod B_R = 0$$

### Worked Examples

| Partition Size | Target Bucket | Bucket Count | Actual Bucket Size |
|:---:|:---:|:---:|:---:|
| 10 GB | 256 MB | 40 | 250 MB |
| 50 GB | 256 MB | 200 | 250 MB |
| 1 GB | 128 MB | 8 | 125 MB |

---

## 6. Predicate Pushdown Savings (I/O Reduction)

### The Problem

How much I/O does predicate pushdown save when reading ORC/Parquet files?

### The Formula

ORC stripe-level pushdown using min/max statistics:

$$\text{Stripes read} = |\{s : \min_s \leq v \leq \max_s\}|$$

$$\eta_{pushdown} = 1 - \frac{\text{Stripes read}}{\text{Total stripes}}$$

With bloom filters (false positive rate $f$):

$$P(\text{stripe read}) = \frac{k}{NDV_{stripe}} + f \times \left(1 - \frac{k}{NDV_{stripe}}\right)$$

Where $k$ = number of distinct lookup values.

### Worked Examples

| Total Stripes | Sorted Column | Lookup Values | Stripes Read | I/O Saved |
|:---:|:---:|:---:|:---:|:---:|
| 1000 | Yes (date) | 1 day | ~3 | 99.7% |
| 1000 | No (user_id) | 1 user | ~950 (bloom: ~10) | 5% (bloom: 99%) |
| 100 | Yes (timestamp) | 1 hour range | ~1 | 99% |

---

## 7. Summary of Formulas

| Formula | Type | Domain |
|---------|------|--------|
| $\sigma = 1/NDV$ | Selectivity | Cardinality estimation |
| $\|R \bowtie S\| = \|R\|\|S\|/\max(NDV)$ | Join cardinality | Query optimization |
| $S_{dict} = NDV \times w + N \log_2(NDV)/8$ | Dictionary encoding | Columnar storage |
| $C_{map} = \|S\| \times E + \|R\|$ | Map join cost | Join selection |
| $b(key) = hash(key) \mod B$ | Bucket assignment | Hash partitioning |
| $\eta = 1 - k/P$ | Pruning ratio | Partition pruning |

## Prerequisites

- information-theory, hash-functions, statistics, sql-query-planning, hadoop

## Complexity

| Operation | Time | Space |
|-----------|------|-------|
| Partition pruning | O(P) metastore lookup | O(1) |
| ORC predicate pushdown | O(S) stripe statistics | O(1) per stripe |
| Map join (broadcast) | O(R + S*E) | O(S) per executor |
| Sort-merge join | O(N log N) | O(N) shuffle |
| Bucket map join | O(R + S) | O(S/B) per task |
| Statistics collection | O(N) full scan | O(C) column stats |
