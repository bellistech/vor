# The Mathematics of Grafana Tempo — Trace Storage and Query Efficiency

> *Tempo stores traces in object storage using a write-ahead log and compacted blocks. The math covers trace ID hashing, bloom filter sizing, compaction ratios, storage costs, and query fan-out complexity.*

---

## 1. Trace ID Hashing (Consistent Hashing)

### The Problem

The distributor must route spans for the same trace to the same ingester so that all spans of a trace are co-located during the ingestion window.

### The Formula

Tempo uses consistent hashing on the 128-bit trace ID. With a hash ring of $N$ ingesters and replication factor $R$:

$$h(\text{traceID}) = \text{FNV-1a}(\text{traceID}_{128}) \mod 2^{32}$$

$$\text{Target Ingesters} = \{ \text{ring}[h], \text{ring}[h+1], \ldots, \text{ring}[h+R-1] \}$$

Expected load per ingester:

$$L_i = \frac{S_{\text{total}}}{N} \pm O\left(\frac{S_{\text{total}}}{\sqrt{N \cdot V}}\right)$$

where $V$ is the number of virtual nodes per ingester and $S_{\text{total}}$ is total span throughput.

### Worked Examples

| Ingesters ($N$) | Virtual Nodes ($V$) | Replication ($R$) | Spans/sec | Spans/Ingester/sec |
|:---:|:---:|:---:|:---:|:---:|
| 3 | 128 | 1 | 50,000 | ~16,667 |
| 10 | 128 | 3 | 200,000 | ~60,000 |
| 20 | 256 | 3 | 1,000,000 | ~150,000 |

---

## 2. Bloom Filter Sizing

### The Problem

Each compacted block contains a bloom filter for fast trace ID lookups. The filter must balance false positive rate against memory and disk usage.

### The Formula

For $n$ trace IDs in a block with desired false positive rate $p$:

$$m = -\frac{n \ln p}{(\ln 2)^2}$$

$$k = \frac{m}{n} \ln 2$$

where $m$ is the number of bits in the filter and $k$ is the optimal number of hash functions.

Storage per bloom filter:

$$\text{Bloom Size (bytes)} = \frac{m}{8} = -\frac{n \ln p}{8 (\ln 2)^2}$$

### Worked Examples

| Traces/Block ($n$) | False Positive ($p$) | Bits ($m$) | Hashes ($k$) | Bloom Size |
|:---:|:---:|:---:|:---:|:---:|
| 100,000 | 0.05 | 623,527 | 4 | 76 KB |
| 100,000 | 0.01 | 958,506 | 7 | 117 KB |
| 1,000,000 | 0.01 | 9,585,059 | 7 | 1.14 MB |
| 1,000,000 | 0.001 | 14,377,588 | 10 | 1.71 MB |

---

## 3. Storage Sizing

### The Problem

Estimate object storage costs based on trace volume, span size, and retention period.

### The Formula

$$\text{Daily Storage} = S_{\text{rate}} \times 86400 \times \bar{b}_{\text{span}} \times C_{\text{ratio}}$$

where $S_{\text{rate}}$ is spans per second, $\bar{b}_{\text{span}}$ is average uncompressed span size in bytes, and $C_{\text{ratio}}$ is the compression ratio (typically 0.15-0.25 with zstd).

Total storage for retention period $D$ days:

$$\text{Total} = \text{Daily Storage} \times D$$

### Worked Examples

| Spans/sec | Avg Span (bytes) | Compression | Daily (GB) | 30-day (GB) | S3 Cost/mo |
|:---:|:---:|:---:|:---:|:---:|:---:|
| 10,000 | 500 | 0.20 | 86.4 | 2,592 | ~$60 |
| 50,000 | 500 | 0.20 | 432 | 12,960 | ~$298 |
| 100,000 | 400 | 0.15 | 518 | 15,552 | ~$358 |
| 500,000 | 300 | 0.15 | 1,944 | 58,320 | ~$1,341 |

S3 cost estimated at $0.023/GB/month.

---

## 4. Compaction Ratios

### The Problem

The compactor merges small ingester-written blocks into larger ones, reducing the number of blocks the querier must search.

### The Formula

With compaction window $W$ and compaction levels $l = 0, 1, \ldots, L$:

$$B_l = \left\lceil \frac{B_0}{F^l} \right\rceil$$

where $B_0$ is the initial number of blocks per window and $F$ is the fanout factor (typically 8-10). Total blocks after full compaction:

$$B_{\text{total}} = \sum_{l=0}^{L} B_l \approx B_0 \cdot \frac{F}{F-1} \cdot \frac{1}{F^L}$$

### Worked Examples

| Initial Blocks/day ($B_0$) | Fanout ($F$) | Levels ($L$) | Final Blocks/day |
|:---:|:---:|:---:|:---:|
| 720 | 10 | 2 | 8 |
| 1,440 | 10 | 2 | 15 |
| 720 | 8 | 3 | 2 |
| 2,880 | 10 | 3 | 3 |

---

## 5. Query Fan-out and Latency

### The Problem

A trace lookup query must check all ingesters (for recent data) and object storage blocks (for historical data). Understanding fan-out determines query latency.

### The Formula

$$T_{\text{query}} = T_{\text{ingester}} + T_{\text{backend}}$$

$$T_{\text{ingester}} = \max_{i=1}^{N} (T_{\text{rpc}_i})$$

$$T_{\text{backend}} = T_{\text{bloom}} + T_{\text{index}} + T_{\text{fetch}}$$

For a search query across time range $[t_1, t_2]$:

$$\text{Blocks to Search} = \frac{t_2 - t_1}{W_{\text{block}}} \times B_{\text{per\_window}}$$

$$T_{\text{search}} \propto \frac{\text{Blocks to Search}}{\text{Query Parallelism}}$$

### Worked Examples

| Time Range | Block Window | Blocks/Window | Total Blocks | Parallelism | Relative Cost |
|:---:|:---:|:---:|:---:|:---:|:---:|
| 1h | 2h | 2 | 2 | 10 | 1x |
| 6h | 2h | 2 | 6 | 10 | 3x |
| 24h | 2h | 2 | 24 | 10 | 12x |
| 7d | 2h | 2 | 168 | 20 | 42x |

---

## 6. Sampling Impact on Trace Completeness

### The Problem

Head sampling discards spans before ingestion. If services sample independently, traces become fragmented.

### The Formula

For a trace spanning $k$ services, each with independent sampling probability $p_i$:

$$P(\text{complete trace}) = \prod_{i=1}^{k} p_i$$

With uniform sampling rate $p$:

$$P(\text{complete trace}) = p^k$$

### Worked Examples

| Services ($k$) | Sample Rate ($p$) | P(complete) | Traces Lost (%) |
|:---:|:---:|:---:|:---:|
| 3 | 0.50 | 0.125 | 87.5% |
| 3 | 0.10 | 0.001 | 99.9% |
| 5 | 0.50 | 0.031 | 96.9% |
| 5 | 0.10 | 0.00001 | 99.999% |
| 3 | 1.00 | 1.000 | 0% |

This is why **propagated sampling** (parent-based) or **tail sampling** is critical.

---

## Prerequisites

- Distributed systems concepts (consistent hashing, replication)
- Probability (bloom filters, false positive rates)
- Object storage economics (S3/GCS pricing tiers)
- OpenTelemetry trace data model (spans, trace context propagation)
- Compression algorithms (zstd, Snappy) and their tradeoffs
