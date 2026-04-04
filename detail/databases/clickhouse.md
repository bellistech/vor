# The Mathematics of ClickHouse — Columnar Compression, MergeTree, and Vectorized Execution

> *ClickHouse achieves extreme analytical query speed through columnar storage, sparse primary indexes, SIMD-vectorized execution, and merge-tree compaction. The math covers compression ratios, index granularity, merge amplification, and query scan optimization.*

---

## 1. Columnar Compression (Information Theory)

### Column vs Row Storage

For a table with $C$ columns, a query touching $c$ columns:

$$\text{Row store I/O} = N \times \sum_{i=1}^{C} S_i$$

$$\text{Column store I/O} = N \times \sum_{i=1}^{c} S_i$$

$$\text{I/O reduction} = 1 - \frac{c}{C} \times \frac{\bar{S}_{\text{queried}}}{\bar{S}_{\text{all}}}$$

| Total Columns | Queried Columns | I/O Reduction |
|:---:|:---:|:---:|
| 50 | 3 | 94% |
| 100 | 5 | 95% |
| 200 | 10 | 95% |
| 20 | 10 | 50% |

### Compression Ratios by Data Type

| Column Type | Encoding | Typical Ratio | Bytes/Value (compressed) |
|:---|:---|:---:|:---:|
| UInt8 (enum-like) | Dictionary + LZ4 | 50:1 - 200:1 | 0.04 - 0.16 |
| UInt64 (sorted ID) | Delta + LZ4 | 10:1 - 50:1 | 0.16 - 0.8 |
| Float64 (metric) | Gorilla + LZ4 | 5:1 - 15:1 | 0.53 - 1.6 |
| DateTime | Delta + LZ4 | 20:1 - 100:1 | 0.04 - 0.2 |
| String (high cardinality) | LZ4 | 2:1 - 5:1 | varies |
| LowCardinality(String) | Dictionary + LZ4 | 20:1 - 100:1 | 0.08 - 0.4 |

### Storage Sizing

$$\text{Compressed size} = \sum_{i=1}^{C} \frac{N \times S_i}{R_i}$$

| Rows | Columns | Avg Raw Size | Avg Compression | Compressed Size |
|:---:|:---:|:---:|:---:|:---:|
| 1 billion | 20 | 8 B/col | 10:1 | 16 GB |
| 1 billion | 50 | 8 B/col | 10:1 | 40 GB |
| 10 billion | 20 | 8 B/col | 10:1 | 160 GB |
| 100 billion | 50 | 8 B/col | 15:1 | 2.67 TB |

---

## 2. Primary Index — Sparse Granularity (Data Structures)

### Index Granularity Model

ClickHouse stores one index entry per $G$ rows (default $G = 8192$):

$$\text{Index entries} = \left\lceil \frac{N}{G} \right\rceil$$

$$\text{Index size} = \frac{N}{G} \times K_{\text{avg}}$$

### Granules Scanned per Query

For a point query on the primary key:

$$\text{Granules scanned} = 1 \quad \text{(binary search to exact granule)}$$

For a range query matching fraction $f$ of data:

$$\text{Granules scanned} = \left\lceil f \times \frac{N}{G} \right\rceil$$

$$\text{Rows scanned} = \text{Granules} \times G$$

| Total Rows | Granularity | Index Entries | Index Size (32B key) |
|:---:|:---:|:---:|:---:|
| 1 million | 8,192 | 122 | 3.9 KB |
| 1 billion | 8,192 | 122,070 | 3.8 MB |
| 100 billion | 8,192 | 12,207,031 | 381 MB |

### Skip Index (Data Skipping)

Bloom filter or min/max index per granule group:

$$P(\text{false positive}) = \left(1 - e^{-kn/m}\right)^k$$

where $k$ = hash functions, $m$ = bits, $n$ = elements.

---

## 3. MergeTree Compaction (Write Amplification)

### Part Lifecycle

Each INSERT creates a new part. Background merges combine parts:

$$\text{Parts after } W \text{ inserts} = W \quad \text{(before merge)}$$

$$\text{Parts after merge} \approx \log_T(W) \quad \text{(T = size ratio)}$$

### Write Amplification

$$W_{\text{amp}} = \sum_{l=0}^{L} T = \frac{T^{L+1} - 1}{T - 1}$$

For default settings ($T \approx 10$):

| Data Size | Levels | Write Amplification |
|:---:|:---:|:---:|
| 1 GB | 2 | ~11x |
| 100 GB | 3 | ~111x |
| 10 TB | 4 | ~1,111x |

### Merge Throughput Requirement

$$\text{Merge bandwidth} = \lambda_{\text{ingest}} \times W_{\text{amp}}$$

| Ingest Rate | Write Amp | Required Merge I/O |
|:---:|:---:|:---:|
| 100 MB/s | 10x | 1 GB/s |
| 500 MB/s | 10x | 5 GB/s |
| 1 GB/s | 10x | 10 GB/s |

---

## 4. Vectorized Query Execution (CPU Optimization)

### Block Processing

ClickHouse processes data in blocks of $B$ rows (default 65,536):

$$\text{Function calls} = \frac{N}{B} \quad \text{(vs } N \text{ for row-at-a-time)}$$

### SIMD Speedup

With SIMD width $W$ (e.g., AVX2 = 256 bits = 4 doubles):

$$\text{SIMD speedup} = \min(W, \text{pipeline\_depth})$$

| Operation | Scalar | SIMD (AVX2) | Speedup |
|:---|:---:|:---:|:---:|
| Filter UInt64 | 1 elem/cycle | 4 elem/cycle | 4x |
| Sum Float64 | 1 elem/cycle | 4 elem/cycle | 4x |
| String compare | 1 byte/cycle | 32 bytes/cycle | 32x |
| Hash computation | 1 elem/cycle | 4 elem/cycle | 4x |

### Query Scan Rate

$$\text{Scan rate} = \frac{\text{Memory bandwidth}}{\text{Bytes per row (compressed)}} \times D_{\text{decompression}}$$

| Mem Bandwidth | Compressed Row | Decompression Overhead | Scan Rate |
|:---:|:---:|:---:|:---:|
| 50 GB/s | 1 B | 1.2x | 41.7 B rows/s |
| 50 GB/s | 4 B | 1.2x | 10.4 B rows/s |
| 50 GB/s | 20 B | 1.5x | 1.67 B rows/s |

---

## 5. Distributed Query Cost (Fan-Out)

### Scatter-Gather

For a distributed table with $S$ shards:

$$T_{\text{query}} = T_{\text{plan}} + \max_{i=1}^{S}(T_{\text{shard}_i}) + T_{\text{merge}}$$

### Data Transfer

$$\text{Network transfer} = S \times \frac{R_{\text{partial}}}{S} = R_{\text{partial}}$$

For aggregation queries, partial results are small:

$$R_{\text{partial}} = G \times S_{\text{row}} \quad \text{(G = group count)}$$

### Shard Imbalance

$$\text{Slowdown} = \frac{\max(T_{\text{shard}_i})}{\text{avg}(T_{\text{shard}_i})}$$

| Shards | Data Skew | Slowdown Factor |
|:---:|:---:|:---:|
| 3 | None | 1.0x |
| 3 | 2:1:1 | 1.5x |
| 10 | Uniform | 1.0x |
| 10 | Hot shard (3x) | 2.1x |

---

## 6. Approximate Algorithms (Probabilistic)

### HyperLogLog (uniq)

$$\text{Relative error} = \frac{1.04}{\sqrt{m}}$$

where $m$ = number of registers (default 17,408 in ClickHouse).

$$\text{Memory} = m \times 5 \text{ bits} \approx 10 \text{ KB}$$

| True Cardinality | Error (%) | Memory |
|:---:|:---:|:---:|
| 1,000 | 0.79% | 10 KB |
| 1,000,000 | 0.79% | 10 KB |
| 1,000,000,000 | 0.79% | 10 KB |

### Quantile (t-digest)

$$\text{Relative error at quantile } q \approx \frac{\delta}{n} \times q(1-q)$$

Memory: ~5-10 KB per quantile state.

---

## Prerequisites

information-theory, data-structures, distributed-systems, sql

## Complexity

| Operation | Time | Space |
|:---|:---:|:---:|
| Point lookup (primary key) | O(log(N/G)) | O(G) granule |
| Range scan (fraction f) | O(f * N / G) | O(G) per block |
| Full aggregation | O(N) vectorized | O(groups) |
| Insert (new part) | O(n log n) sort | O(n) part |
| Merge (background) | O(n) per merge | O(n) output |
| Distributed query | O(S * local) | O(S * partial) |
| uniq (HyperLogLog) | O(N) | O(10 KB) |
| Bloom filter check | O(k) hashes | O(m) bits |
