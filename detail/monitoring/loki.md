# The Mathematics of Loki — Label Cardinality and Log Storage Efficiency

> *Loki indexes only labels, not log content, making storage costs proportional to label cardinality rather than log volume. The mathematics cover label combinatorial explosion, chunk compression ratios, ingestion rate limiting, LogQL query cost, and retention storage planning.*

---

## 1. Label Cardinality (Combinatorial Explosion)

### The Problem

Each unique combination of label key-value pairs creates a separate stream. High-cardinality labels (e.g., user IDs, request IDs) multiply the number of streams exponentially, degrading ingestion and query performance.

### The Formula

For $k$ label keys with cardinalities $c_1, c_2, \ldots, c_k$, the maximum number of unique streams:

$$S_{\text{max}} = \prod_{i=1}^{k} c_i$$

In practice, not all combinations exist. If labels are independent:

$$S_{\text{active}} \approx \prod_{i=1}^{k} c_i \times p$$

Where $p$ is the probability a combination is active (typically $p \ll 1$).

### Worked Examples

| Labels | Cardinalities | Max Streams |
|:---:|:---:|:---:|
| {job, env} | 10, 3 | 30 |
| {job, env, host} | 10, 3, 50 | 1,500 |
| {job, env, host, path} | 10, 3, 50, 1000 | 1,500,000 |
| {job, env, host, user_id} | 10, 3, 50, 100000 | 150,000,000 |

Adding `user_id` as a label increases streams by $100{,}000\times$. This is why Loki forbids high-cardinality labels.

### Index Size

Index size scales linearly with stream count:

$$S_{\text{index}} = S_{\text{active}} \times (\bar{L}_{\text{labels}} + C_{\text{overhead}})$$

Where $\bar{L}_{\text{labels}}$ is average serialized label set size (~200 bytes) and $C_{\text{overhead}}$ ~50 bytes per entry.

---

## 2. Chunk Compression (Storage Efficiency)

### The Problem

Loki stores log lines in compressed chunks. Log data compresses well because of repetitive structure (timestamps, common prefixes, repeated patterns).

### The Formula

Compression ratio:

$$r = \frac{S_{\text{raw}}}{S_{\text{compressed}}}$$

For typical log data with gzip/snappy:

| Log Type | Raw Size | Compressed | Ratio |
|:---:|:---:|:---:|:---:|
| JSON structured | 100 MB | 8 MB | 12.5x |
| Nginx access | 100 MB | 12 MB | 8.3x |
| Syslog | 100 MB | 15 MB | 6.7x |
| Application (mixed) | 100 MB | 10 MB | 10x |

Storage cost for daily volume $V$ with retention $D$ days:

$$S_{\text{total}} = \frac{V \times D}{r}$$

### Worked Example

50 GB/day raw logs, 30-day retention, compression ratio 10x:

$$S_{\text{total}} = \frac{50 \times 30}{10} = 150 \text{ GB on object storage}$$

At S3 pricing ($0.023/GB/month):

$$C = 150 \times 0.023 = \$3.45/\text{month}$$

---

## 3. Ingestion Rate Limiting (Token Bucket)

### The Problem

Loki uses per-tenant rate limiting to prevent any single tenant from overwhelming the ingesters. The token bucket algorithm provides burst tolerance with a sustained rate cap.

### The Formula

Token bucket state:

$$\text{tokens}(t) = \min\left(B, \text{tokens}(t - \Delta t) + R \times \Delta t\right)$$

Where $R$ = `ingestion_rate_mb` (MB/s) and $B$ = `ingestion_burst_size_mb`.

Request accepted if:

$$\text{request size} \leq \text{tokens}(t)$$

### Burst Duration

Maximum burst duration at rate $b$ (burst throughput) before throttling:

$$T_{\text{burst}} = \frac{B}{b - R}$$

### Worked Example

$R = 10$ MB/s, $B = 20$ MB, burst at $b = 30$ MB/s:

$$T_{\text{burst}} = \frac{20}{30 - 10} = 1.0 \text{ second}$$

After 1 second of burst, the client is throttled to the sustained rate of 10 MB/s.

---

## 4. LogQL Query Cost (Scan Model)

### The Problem

LogQL queries scan chunks within a time range for matching streams. Query cost depends on the number of streams matched, time range, and chunk density.

### The Formula

Chunks scanned for a query matching $s$ streams over time range $T$:

$$N_{\text{chunks}} = s \times \left\lceil \frac{T}{\text{chunk duration}} \right\rceil$$

Bytes read:

$$B_{\text{read}} = N_{\text{chunks}} \times \bar{S}_{\text{chunk}}$$

Default chunk target size is ~1.5 MB compressed, duration ~1 hour.

### Worked Examples

| Streams Matched | Time Range | Chunks Scanned | Bytes Read |
|:---:|:---:|:---:|:---:|
| 10 | 1h | 10 | 15 MB |
| 10 | 24h | 240 | 360 MB |
| 1,000 | 1h | 1,000 | 1.5 GB |
| 1,000 | 24h | 24,000 | 36 GB |

Query speedup from narrow label selectors:

$$\text{Speedup} = \frac{S_{\text{total streams}}}{s_{\text{matched}}}$$

If total streams = 10,000 and query matches 10:

$$\text{Speedup} = \frac{10{,}000}{10} = 1{,}000\times$$

---

## 5. Ingester Memory Model (Write-Ahead)

### The Problem

Ingesters buffer log entries in memory before flushing to storage. Memory consumption depends on active stream count and chunk buffer sizes.

### The Formula

Memory per ingester:

$$M = S_{\text{active}} \times (M_{\text{stream}} + M_{\text{chunk buffer}})$$

Where $M_{\text{stream}} \approx 1$ KB (labels, metadata) and $M_{\text{chunk buffer}} \approx$ target chunk size uncompressed (~5 MB default).

With replication factor $r$ and $I$ ingesters, streams per ingester:

$$S_{\text{per ingester}} = \frac{S_{\text{total}} \times r}{I}$$

### Worked Example

50,000 active streams, replication factor 3, 6 ingesters:

$$S_{\text{per}} = \frac{50{,}000 \times 3}{6} = 25{,}000 \text{ streams/ingester}$$

$$M = 25{,}000 \times (1 + 5) \text{ KB} = 25{,}000 \times 6 \text{ KB} = 150 \text{ MB (chunk buffers only)}$$

With overhead (hash rings, WAL, etc.), real memory is ~3-5x:

$$M_{\text{real}} \approx 450\text{-}750 \text{ MB per ingester}$$

---

## 6. Compaction and Retention (Lifecycle)

### The Problem

The compactor merges small index files and enforces retention by deleting expired chunks. Understanding compaction frequency prevents index bloat.

### The Formula

Index files generated per day:

$$F_{\text{daily}} = \frac{S_{\text{active}}}{\text{entries per index file}} \times \frac{86{,}400}{\text{flush interval}}$$

After compaction, files reduce to:

$$F_{\text{compacted}} = \left\lceil \frac{F_{\text{daily}}}{\text{compaction ratio}} \right\rceil$$

Retention deletion rate:

$$R_{\text{delete}} = \frac{V_{\text{daily}} \times (1 + I_{\text{overhead}})}{r_{\text{compression}}}$$

### Worked Example

50 GB/day, compression 10x, 30-day retention, deletion begins on day 31:

$$R_{\text{delete}} = \frac{50}{10} = 5 \text{ GB/day of object storage freed}$$

Steady-state storage:

$$S_{\text{steady}} = \frac{50 \times 30}{10} = 150 \text{ GB}$$

---

## 7. Query Parallelism (Split and Merge)

### The Problem

The Query Frontend splits large time ranges into smaller intervals for parallel execution. The optimal split interval balances parallelism against per-query overhead.

### The Formula

For a query spanning time $T$ with split interval $I$:

$$\text{Sub-queries} = \left\lceil \frac{T}{I} \right\rceil$$

With $W$ query workers:

$$T_{\text{wall}} = \frac{\text{Sub-queries}}{W} \times T_{\text{per sub-query}}$$

Optimal split interval (minimize wall time):

$$I^* = \frac{T}{W \times k}$$

Where $k$ is the pipeline depth factor (typically 2-3 for optimal queue utilization).

### Worked Example

7-day query, 10 workers, split interval 24h:

$$\text{Sub-queries} = \lceil 7/1 \rceil = 7$$

$$T_{\text{wall}} = \frac{7}{10} \times T_{\text{sub}} = 0.7 \times T_{\text{sub}}$$

With 6h split:

$$\text{Sub-queries} = 28, \quad T_{\text{wall}} = \frac{28}{10} \times T_{\text{sub, 6h}} = 2.8 \times T_{\text{sub, 6h}}$$

Since $T_{\text{sub, 6h}} \approx 0.25 \times T_{\text{sub, 24h}}$:

$$T_{\text{wall, 6h}} = 2.8 \times 0.25 \times T_{\text{sub}} = 0.7 \times T_{\text{sub}}$$

Similar wall time, but smaller memory footprint per sub-query.

---

## Prerequisites

- combinatorics, information-theory, queuing-theory, compression-algorithms
