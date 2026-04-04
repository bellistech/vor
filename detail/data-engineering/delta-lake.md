# The Mathematics of Delta Lake — ACID Transactions, File Compaction, and Data Skipping

> *Delta Lake layers ACID semantics over Parquet files using a JSON transaction log. The math covers optimistic concurrency control, Z-ORDER locality, file compaction thresholds, data skipping selectivity, and time travel storage costs.*

---

## 1. Optimistic Concurrency Control (Transaction Theory)

### The Problem

Multiple writers may attempt concurrent commits. Delta Lake uses optimistic concurrency: each writer reads the current log version, computes changes, and attempts to commit. Conflicts are detected by checking if the read set overlaps with concurrent writes.

### The Formula

The commit succeeds if no conflicting actions occurred between read version $v_r$ and commit version $v_c$:

$$\text{conflict} = \exists\, a \in \text{actions}(v_r, v_c) : a.\text{files} \cap w.\text{read\_set} \neq \emptyset$$

With $\lambda$ concurrent writers and average transaction duration $D$, the conflict probability:

$$P_{\text{conflict}} = 1 - e^{-\lambda \cdot D \cdot f_{\text{overlap}}}$$

where $f_{\text{overlap}}$ is the fraction of the table touched by each transaction.

### Worked Examples

| Writers | TXN Duration | Table Overlap | Conflict Probability |
|:---:|:---:|:---:|:---:|
| 2 | 10s | 100% (full table) | 86.5% |
| 2 | 10s | 1% (one partition) | 1.8% |
| 10 | 30s | 1% | 26.0% |
| 10 | 30s | 0.1% | 3.0% |
| 5 | 5s | 10% | 91.8% |

Key insight: partition your writes so each writer touches disjoint partitions ($f_{\text{overlap}} \to 0$).

## 2. Z-ORDER Curve Locality (Space-Filling Curves)

### The Problem

Z-ORDER interleaves the bits of multiple column values to create a single ordering that preserves multi-dimensional locality. This enables data skipping across multiple columns simultaneously.

### The Formula

For two dimensions $(x, y)$ each with $b$ bits, the Z-value interleaves bits:

$$Z(x, y) = \sum_{i=0}^{b-1} \left( x_i \cdot 2^{2i} + y_i \cdot 2^{2i+1} \right)$$

The number of files touched by a query filtering on $d$ dimensions out of $D$ Z-ordered dimensions:

$$F_{\text{read}} \approx N_{\text{files}} \cdot \prod_{j=1}^{d} \sigma_j^{1/D}$$

where $\sigma_j$ is the selectivity on dimension $j$ (fraction of domain matched).

Without Z-ORDER (scanning all files):

$$F_{\text{baseline}} = N_{\text{files}} \cdot \prod_{j=1}^{d} \sigma_j^{1/1} = N_{\text{files}} \cdot \prod \sigma_j$$

The Z-ORDER benefit ratio:

$$\text{Speedup} = \frac{N_{\text{files}}}{ N_{\text{files}} \cdot \prod \sigma_j^{1/D}} = \prod \sigma_j^{(D-1)/D}$$

### Worked Examples

1000 files, query filters on user_id (selectivity 0.1%) and event_type (selectivity 10%):

Without Z-ORDER: scan all 1000 files (no multi-column skipping).

With Z-ORDER on 2 columns:

$$F_{\text{read}} = 1000 \times 0.001^{1/2} \times 0.1^{1/2} = 1000 \times 0.0316 \times 0.316 = 10 \text{ files}$$

| Columns Z-Ordered | Selectivities | Files Read | Speedup |
|:---:|:---:|:---:|:---:|
| 1 (user_id only) | 0.1% | 1 | 1000x |
| 2 (user_id, event_type) | 0.1%, 10% | 10 | 100x |
| 3 (+ region) | 0.1%, 10%, 20% | 27 | 37x |
| 2 (broad filters) | 10%, 20% | 141 | 7.1x |

## 3. File Compaction Analysis (Storage Optimization)

### The Problem

Streaming writes and frequent appends create many small files. OPTIMIZE compacts them into target-sized files. The overhead is reading all small files and writing consolidated output.

### The Formula

Compaction cost for $n$ small files of average size $\bar{s}$ into files of target size $T$:

$$\text{Files after} = \left\lceil \frac{n \cdot \bar{s}}{T} \right\rceil$$

$$\text{Cost} = n \cdot \bar{s} \cdot (c_{\text{read}} + c_{\text{write}})$$

The small file amplification factor (ratio of I/O operations):

$$\text{Amplification} = \frac{n}{n \cdot \bar{s} / T} = \frac{T}{\bar{s}}$$

The query performance impact of $n$ small files vs. $m$ optimal files:

$$\frac{T_{\text{query}}(n)}{T_{\text{query}}(m)} \approx 1 + \frac{(n - m) \cdot c_{\text{open}}}{m \cdot c_{\text{scan}}}$$

### Worked Examples

10,000 files of 1 MB each, target size 256 MB:

$$\text{After compaction} = \lceil 10000 / 256 \rceil = 40 \text{ files}$$

| Small Files | Avg Size | Target | After Compact | Read Amplification |
|:---:|:---:|:---:|:---:|:---:|
| 1,000 | 1 MB | 256 MB | 4 | 250x fewer files |
| 10,000 | 1 MB | 256 MB | 40 | 250x fewer files |
| 500 | 10 MB | 256 MB | 20 | 25x fewer files |
| 100 | 128 MB | 256 MB | 50 | 2x fewer files |

## 4. Data Skipping Effectiveness (Statistics Theory)

### The Problem

Delta Lake stores min/max statistics per column per file. A query filter can skip files whose min/max range does not overlap the filter predicate.

### The Formula

For a filter $a \leq x \leq b$ on a column with values uniformly distributed in $[L, H]$, the probability a file of $r$ rows is NOT skippable:

$$P_{\text{read}} = 1 - P(\max < a) - P(\min > b)$$

For uniform data in $N$ files, each with $r = N_{\text{total}}/N$ rows:

$$P_{\text{skip}} = P(\max < a) + P(\min > b) = \left(\frac{a - L}{H - L}\right)^r + \left(\frac{H - b}{H - L}\right)^r$$

When data is sorted by the filter column:

$$F_{\text{read}} = \left\lceil N \cdot \frac{b - a}{H - L} \right\rceil$$

### Worked Examples

Column with values in [0, 1000], query: $100 \leq x \leq 200$, selectivity = 10%.

| Files | Rows/File | Data Sorted | Files Read (sorted) | Files Read (random) |
|:---:|:---:|:---:|:---:|:---:|
| 100 | 10,000 | Yes | 10 | ~100 (no skipping) |
| 100 | 10,000 | Z-Ordered | 10 | 10-15 |
| 1,000 | 1,000 | Yes | 100 | ~1,000 |
| 1,000 | 1,000 | Z-Ordered | 100 | 100-150 |

Random data defeats min/max skipping because every file spans the full range.

## 5. Time Travel Storage Cost (Versioning Theory)

### The Problem

Each transaction creates a new log entry and potentially new data files. Time travel requires retaining old files. The storage cost depends on the write pattern and retention period.

### The Formula

With $w$ writes per day, each modifying fraction $f$ of the data (size $S$):

$$\text{Storage at day } d = S + d \cdot w \cdot f \cdot S \cdot (1 - r_{\text{vacuum}})$$

After VACUUM with retention $R$ days:

$$\text{Storage}_{\text{steady}} = S + R \cdot w \cdot f \cdot S$$

The overhead ratio:

$$\text{Overhead} = \frac{\text{Storage}_{\text{steady}}}{S} = 1 + R \cdot w \cdot f$$

### Worked Examples

Table size 100 GB, 1 daily write modifying 10% of data:

| Retention ($R$) | Daily Writes | Modify Fraction | Total Storage | Overhead |
|:---:|:---:|:---:|:---:|:---:|
| 7 days | 1 | 10% | 170 GB | 1.7x |
| 7 days | 1 | 100% | 800 GB | 8.0x |
| 30 days | 1 | 10% | 400 GB | 4.0x |
| 7 days | 24 (hourly) | 1% | 268 GB | 2.7x |
| 0 (vacuum 0h) | 1 | 10% | 100 GB | 1.0x |

## 6. Transaction Log Checkpointing (Log-Structured Storage)

### The Problem

Delta Lake's transaction log grows with every commit. Checkpoints consolidate log entries to avoid reading the entire history on table load.

### The Formula

Without checkpoints, table open cost for version $v$:

$$T_{\text{open}} = v \cdot c_{\text{read\_json}}$$

With checkpoints every $C$ commits (default $C = 10$):

$$T_{\text{open}} = c_{\text{read\_parquet}} + (v \bmod C) \cdot c_{\text{read\_json}}$$

The log directory size after $v$ commits with checkpoint interval $C$:

$$S_{\text{log}} = v \cdot s_{\text{json}} + \left\lfloor \frac{v}{C} \right\rfloor \cdot s_{\text{checkpoint}}$$

### Worked Examples

| Commits ($v$) | Checkpoint Interval | JSON Reads | Open Time (relative) |
|:---:|:---:|:---:|:---:|
| 1,000 | None | 1,000 | 100x |
| 1,000 | 10 | 1 + 9 = 10 | 1x |
| 10,000 | 10 | 1 + 9 = 10 | 1x |
| 10,000 | 100 | 1 + 99 = 100 | 10x |

Checkpoint overhead is negligible compared to reading thousands of JSON log entries.

## Prerequisites

- ACID transaction properties (atomicity, consistency, isolation, durability)
- Space-filling curves (Z-order, Hilbert) and bit interleaving
- Parquet columnar format and row group structure
- Min/max statistics and predicate pushdown
- Optimistic vs. pessimistic concurrency control
- Write-ahead logging and journaling concepts
