# The Mathematics of TimescaleDB — Chunk Partitioning, Compression, and Time-Series Aggregation

> *TimescaleDB partitions time-series data into chunks for locality and pruning. The math covers chunk sizing, compression entropy, continuous aggregate staleness, time-bucket alignment, and interpolation error bounds.*

---

## 1. Chunk Interval Optimization (Partitioning Theory)

### The Problem

Choosing the chunk time interval is a trade-off: too small creates excessive chunk overhead and planning time, too large reduces partition pruning and bloats individual chunks beyond memory.

### The Formula

The optimal chunk interval $I$ should produce chunks that fit in 25% of available memory $M$:

$$I = \frac{0.25 \cdot M}{R \cdot \bar{s}}$$

where $R$ is the ingest rate (rows/second) and $\bar{s}$ is the average row size in bytes (including indexes).

The number of active chunks at any time:

$$N_{\text{active}} = \left\lceil \frac{W}{I} \right\rceil \cdot P$$

where $W$ is the query window and $P$ is the number of space partitions.

### Worked Examples

| Memory | Ingest Rate | Row Size | Optimal Interval | Chunk Size |
|:---:|:---:|:---:|:---:|:---:|
| 16 GB | 10,000 rows/s | 200 B | ~5.8 hours | ~4 GB |
| 16 GB | 1,000 rows/s | 200 B | ~58 hours | ~4 GB |
| 64 GB | 50,000 rows/s | 500 B | ~9.3 hours | ~16 GB |
| 8 GB | 500 rows/s | 150 B | ~77 hours | ~2 GB |

For 16 GB memory, 10,000 rows/s, 200 bytes/row:

$$I = \frac{0.25 \times 16 \times 10^9}{10{,}000 \times 200} = \frac{4 \times 10^9}{2 \times 10^6} = 2{,}000 \text{ s} \approx 5.8 \text{ hours}$$

## 2. Compression Ratio (Information Theory)

### The Problem

TimescaleDB uses column-oriented compression with delta-of-delta encoding for timestamps and Gorilla encoding for floating-point values. The compression ratio depends on data regularity.

### The Formula

For timestamps with constant interval $\delta$, delta-of-delta encoding compresses $n$ timestamps to:

$$S_{\text{time}} = 64 + (n - 1) \cdot b_{\text{dod}}$$

where $b_{\text{dod}}$ is bits per delta-of-delta (often 0-1 bits for regular intervals).

For floating-point values using Gorilla XOR encoding, consecutive similar values compress to:

$$S_{\text{float}} = 64 + \sum_{i=2}^{n} b_{\text{xor}}(v_i, v_{i-1})$$

where $b_{\text{xor}}$ is the meaningful bits in the XOR of consecutive values (typically 0-20 bits).

Overall compression ratio:

$$CR = \frac{n \cdot (s_{\text{time}} + k \cdot s_{\text{value}})}{S_{\text{time}} + \sum_{j=1}^{k} S_{\text{col}_j} + S_{\text{meta}}}$$

### Worked Examples

1000 rows, regular 10-second interval, 4 float columns with slowly changing values:

Uncompressed: $1000 \times (8 + 4 \times 8) = 40{,}000$ bytes

Compressed timestamps: $8 + 999 \times 0.125 \approx 133$ bytes (0-bit delta-of-delta with framing)

Compressed floats (avg 12 bits/value): $4 \times (8 + 999 \times 1.5) \approx 6{,}026$ bytes

$$CR = \frac{40{,}000}{133 + 6{,}026 + 200} \approx 6.3\times$$

Typical ratios: 10-20x for regular sensor data, 5-10x for irregular event data.

## 3. time_bucket Alignment (Modular Arithmetic)

### The Problem

time_bucket snaps timestamps to bucket boundaries. Understanding the alignment is critical for correct aggregation and gap-filling.

### The Formula

For a bucket width $w$ and timestamp $t$ (as Unix epoch):

$$\text{bucket}(t, w) = \left\lfloor \frac{t - o}{w} \right\rfloor \cdot w + o$$

where $o$ is the origin offset (default: Unix epoch, 2000-01-01 for PostgreSQL).

The number of buckets in a time range $[t_1, t_2]$:

$$N_{\text{buckets}} = \left\lfloor \frac{t_2 - o}{w} \right\rfloor - \left\lfloor \frac{t_1 - o}{w} \right\rfloor + 1$$

### Worked Examples

For 5-minute buckets ($w = 300$ s) and timestamp 14:07:23:

$$\text{bucket}(14{:}07{:}23, 300) = \left\lfloor \frac{50843}{300} \right\rfloor \times 300 = 169 \times 300 = 50700 \rightarrow 14{:}05{:}00$$

| Timestamp | 5-min Bucket | 1-hour Bucket | 1-day Bucket |
|:---:|:---:|:---:|:---:|
| 14:07:23 | 14:05:00 | 14:00:00 | 00:00:00 |
| 14:59:59 | 14:55:00 | 14:00:00 | 00:00:00 |
| 15:00:00 | 15:00:00 | 15:00:00 | 00:00:00 |

## 4. Interpolation Error (Numerical Analysis)

### The Problem

When gap-filling with linear interpolation, the error depends on the curvature of the true signal and the gap size.

### The Formula

For linear interpolation between points $(t_1, v_1)$ and $(t_2, v_2)$ at time $t$:

$$\hat{v}(t) = v_1 + \frac{v_2 - v_1}{t_2 - t_1} \cdot (t - t_1)$$

The maximum interpolation error for a signal with bounded second derivative $|f''| \leq M$:

$$|\epsilon_{\max}| = \frac{M \cdot (t_2 - t_1)^2}{8}$$

### Worked Examples

For a temperature signal with $|f''| \leq 0.01$ degrees/min^2:

| Gap Duration | Max Error |
|:---:|:---:|
| 5 min | 0.03 degrees |
| 15 min | 0.28 degrees |
| 60 min | 4.50 degrees |
| 240 min | 72.0 degrees |

This shows why short gaps are safe to interpolate but long gaps should use LOCF or NULL.

## 5. Continuous Aggregate Staleness (Refresh Theory)

### The Problem

Continuous aggregates refresh on a schedule. Between refreshes, the aggregate may be stale. Real-time aggregation merges materialized data with recent unmaterialized data.

### The Formula

Without real-time aggregation, maximum data staleness:

$$\text{staleness}_{\max} = \text{schedule\_interval} + \text{end\_offset}$$

Query cost with real-time aggregation (merging materialized + raw):

$$T_{\text{query}} = T_{\text{mat}} + T_{\text{raw}}(\Delta t)$$

where $\Delta t = \text{now} - \text{last\_refresh\_end}$ is the unmaterialized window.

$$T_{\text{raw}}(\Delta t) = \frac{\Delta t \cdot R \cdot s_{\text{scan}}}{\text{chunk\_prune\_factor}}$$

### Worked Examples

With hourly refresh, 1-hour end_offset, 10,000 rows/s ingest:

| Scenario | Staleness | Raw Rows to Scan |
|:---:|:---:|:---:|
| No real-time, just refreshed | 0 | 0 |
| No real-time, worst case | 2 hours | N/A (stale) |
| Real-time, just refreshed | 0 | ~36M (1 hr) |
| Real-time, mid-cycle | 0 | ~54M (1.5 hr) |

## 6. Space Partitioning Hash Distribution (Hash Theory)

### The Problem

Space partitioning distributes data across $P$ partitions using a hash function on a dimension column. Skewed distributions cause hot partitions.

### The Formula

For $P$ partitions and $D$ distinct dimension values, expected rows per partition:

$$E[\text{rows}_p] = \frac{N}{P}$$

With Zipf-distributed dimension access ($\alpha$ skewness), the load on the busiest partition:

$$\text{max\_load} \approx \frac{N}{P} + \frac{N}{D^\alpha \cdot H_{D,\alpha}} \cdot \sqrt{2 \cdot P \cdot \ln(P)}$$

where $H_{D,\alpha}$ is the generalized harmonic number.

### Worked Examples

| Dimensions ($D$) | Partitions ($P$) | Skew ($\alpha$) | Max Partition Overhead |
|:---:|:---:|:---:|:---:|
| 1,000 | 4 | 0.0 (uniform) | ~2% |
| 1,000 | 4 | 0.5 | ~8% |
| 100 | 4 | 1.0 | ~25% |
| 10 | 4 | 1.0 | ~60% |

High skew with few dimensions makes space partitioning counterproductive.

## Prerequisites

- PostgreSQL fundamentals (tables, indexes, EXPLAIN)
- Partition pruning and query planning
- Modular arithmetic and floor/ceiling functions
- Information theory basics (entropy, encoding)
- Linear interpolation and error analysis
- Hash functions and uniform distribution
