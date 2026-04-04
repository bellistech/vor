# The Mathematics of VictoriaMetrics — Compression, Deduplication, and Cluster Sharding

> *VictoriaMetrics achieves extreme compression and query speed through delta-of-delta encoding, XOR float compression, and merge-tree storage. The math covers compression ratios, cluster data distribution, deduplication semantics, and query fan-out cost.*

---

## 1. Compression — Gorilla Encoding (Information Theory)

### Timestamp Compression (Delta-of-Delta)

Timestamps are compressed using double-delta encoding:

$$\delta_i = t_i - t_{i-1}$$

$$\delta\delta_i = \delta_i - \delta_{i-1}$$

For regular scrape intervals, $\delta\delta_i = 0$ most of the time, encoding in 1 bit.

| $\delta\delta$ Range | Encoding Bits | Probability (15s interval) |
|:---:|:---:|:---:|
| 0 | 1 | ~96% |
| [-63, 64] | 10 | ~3% |
| [-255, 256] | 13 | ~0.8% |
| [-2047, 2048] | 16 | ~0.15% |
| Larger | 68 | ~0.05% |

### Value Compression (XOR)

Float values compressed by XOR with previous value:

$$x_i = v_i \oplus v_{i-1}$$

If values are identical ($x_i = 0$): 1 bit. Otherwise encode leading zeros, meaningful bits, trailing zeros.

### Effective Bytes Per Sample

$$B_{\text{avg}} = \frac{B_{\text{timestamp}} + B_{\text{value}}}{8}$$

| Data Pattern | Timestamp Bits | Value Bits | Bytes/Sample |
|:---|:---:|:---:|:---:|
| Constant value, regular interval | 1 | 1 | 0.25 |
| Slowly changing, regular interval | 1 | 20-30 | 2.6-3.9 |
| Random float, regular interval | 1 | 36-50 | 4.6-6.4 |
| VictoriaMetrics average | ~2 | ~3.5 | 0.7 |
| Prometheus average | ~2 | ~10 | 1.5 |

VictoriaMetrics achieves ~0.7 bytes/sample vs Prometheus ~1.5 bytes/sample due to additional ZSTD block compression.

---

## 2. Storage Sizing (Capacity Planning)

### Disk Formula

$$\text{Disk} = \frac{S \times 86400 \times D_{\text{ret}}}{I} \times B_{\text{avg}}$$

where $S$ = active time series, $I$ = scrape interval (seconds), $D_{\text{ret}}$ = retention days.

### Worked Examples

| Time Series | Interval | Retention | Bytes/Sample | Disk |
|:---:|:---:|:---:|:---:|:---:|
| 100,000 | 15s | 30 days | 0.7 | 11.7 GiB |
| 100,000 | 15s | 365 days | 0.7 | 142.7 GiB |
| 1,000,000 | 15s | 30 days | 0.7 | 117.2 GiB |
| 1,000,000 | 15s | 365 days | 0.7 | 1,427 GiB |
| 10,000,000 | 30s | 90 days | 0.7 | 176 GiB |

### Compared to Prometheus

| Series | Retention | VM Disk (0.7B) | Prom Disk (1.5B) | Savings |
|:---:|:---:|:---:|:---:|:---:|
| 1,000,000 | 30 days | 117 GiB | 251 GiB | 53% |
| 1,000,000 | 365 days | 1,427 GiB | 3,058 GiB | 53% |

---

## 3. Cluster Sharding (Distributed Systems)

### Data Distribution

vminsert distributes time series across $N$ vmstorage nodes using consistent hashing:

$$\text{node}(\text{series}) = \text{hash}(\text{metric\_name}, \text{labels}) \mod N$$

### Replication

With replication factor $R$, each sample is written to $R$ storage nodes:

$$\text{Write amplification} = R$$

$$\text{Storage per node} = \frac{\text{Total Data} \times R}{N}$$

| Total Data | Nodes ($N$) | Replication ($R$) | Per-Node Storage |
|:---:|:---:|:---:|:---:|
| 1 TiB | 3 | 1 | 333 GiB |
| 1 TiB | 3 | 2 | 667 GiB |
| 10 TiB | 5 | 2 | 4 TiB |
| 10 TiB | 10 | 2 | 2 TiB |

### Query Fan-Out

vmselect must query all $N$ storage nodes for each request:

$$T_{\text{query}} = T_{\text{scatter}} + \max_{i=1}^{N}(T_{\text{node}_i}) + T_{\text{merge}}$$

$$T_{\text{merge}} = O(N \times k) \quad \text{where } k = \text{results per node}$$

---

## 4. Deduplication Semantics (Data Cleaning)

### Dedup Algorithm

When `dedup.minScrapeInterval=d`, samples within window $d$ are deduplicated:

$$\text{For each time series and time window } [t, t+d]:$$
$$\text{keep} = \text{sample with largest value (deterministic tiebreak)}$$

### HA Pair Write Volume

With 2 Prometheus instances scraping the same targets:

$$\text{Raw writes} = 2 \times S \times \frac{86400}{I}$$

$$\text{After dedup} = S \times \frac{86400}{I}$$

$$\text{Dedup ratio} = \frac{1}{R_{\text{replicas}}} = 0.5$$

| HA Pairs | Raw Writes/day | After Dedup | Reduction |
|:---:|:---:|:---:|:---:|
| 2 (standard HA) | 2x | 1x | 50% |
| 3 (triple) | 3x | 1x | 67% |

---

## 5. Merge Tree Operations (Data Structures)

### LSM-Like Merge Strategy

VictoriaMetrics uses a merge-tree (similar to LSM-tree):

$$\text{Write amplification} = O(L) \quad \text{where } L = \text{levels}$$

$$\text{Levels} = \left\lceil \log_T \frac{N}{M} \right\rceil$$

where $T$ = size ratio between levels, $M$ = memtable size, $N$ = total data size.

### Merge Cost

$$\text{Merge bytes} = N \times W_{\text{amp}}$$

| Data Size | Levels | Size Ratio | Write Amplification |
|:---:|:---:|:---:|:---:|
| 10 GiB | 3 | 10 | 3x |
| 100 GiB | 4 | 10 | 4x |
| 1 TiB | 5 | 10 | 5x |

### Background Merge Throughput

$$T_{\text{merge}} = \frac{N \times W_{\text{amp}}}{B_{\text{disk}}}$$

where $B_{\text{disk}}$ = disk write bandwidth.

---

## 6. Ingestion Rate Limits (Throughput)

### Maximum Ingestion

$$\lambda_{\text{max}} = \frac{B_{\text{disk}}}{B_{\text{avg}} \times W_{\text{amp}}}$$

| Disk Bandwidth | Bytes/Sample | Write Amp | Max Ingestion |
|:---:|:---:|:---:|:---:|
| 100 MB/s | 0.7 | 4 | 35.7M samples/s |
| 500 MB/s (SSD) | 0.7 | 4 | 178.6M samples/s |
| 1 GB/s (NVMe) | 0.7 | 4 | 357.1M samples/s |

### Memory for Ingestion

$$M_{\text{ingest}} \approx S_{\text{active}} \times 1 \text{ KiB}$$

| Active Series | Memory (approx) |
|:---:|:---:|
| 1,000,000 | ~1 GiB |
| 10,000,000 | ~10 GiB |
| 100,000,000 | ~100 GiB |

---

## Prerequisites

information-theory, distributed-systems, data-structures, prometheus

## Complexity

| Operation | Time | Space |
|:---|:---:|:---:|
| Sample insert | O(1) amortized | O(1) per sample |
| Merge (background) | O(n) per level | O(block) buffer |
| Point query (single ts) | O(log n) | O(1) |
| Range query | O(log n + k) | O(k) results |
| Label index lookup | O(1) hash | O(s) series set |
| Deduplication | O(w) per window | O(w) buffer |
| Cluster scatter query | O(N) fan-out | O(N * k) merge |
