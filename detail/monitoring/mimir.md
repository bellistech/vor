# The Mathematics of Grafana Mimir — Time Series Scalability and Storage

> *Mimir scales Prometheus horizontally via consistent hashing, tenant isolation, and block compaction. The math covers ingestion sharding, storage sizing, query splitting, compaction amplification, and cardinality limits.*

---

## 1. Ingestion Sharding (Write Path)

### The Problem

The distributor must spread incoming samples across ingesters evenly. With replication factor $R$, each sample is written to $R$ ingesters.

### The Formula

For $N$ ingesters, replication factor $R$, and total ingestion rate $I$ samples/sec:

$$I_{\text{per\_ingester}} = \frac{I \times R}{N}$$

Total write amplification:

$$W_{\text{total}} = I \times R$$

Memory per ingester (active series):

$$M_i = \frac{S_{\text{total}}}{N} \times \bar{m}_{\text{series}}$$

where $S_{\text{total}}$ is total active series and $\bar{m}_{\text{series}} \approx 8\text{-}12$ KB per active series (index + chunks in memory).

### Worked Examples

| Active Series | Ingesters ($N$) | Replication ($R$) | Series/Ingester | Memory/Ingester |
|:---:|:---:|:---:|:---:|:---:|
| 1,000,000 | 3 | 3 | 1,000,000 | ~10 GB |
| 5,000,000 | 10 | 3 | 1,500,000 | ~15 GB |
| 20,000,000 | 30 | 3 | 2,000,000 | ~20 GB |
| 100,000,000 | 100 | 3 | 3,000,000 | ~30 GB |

---

## 2. Block Storage Sizing

### The Problem

Ingesters write 2-hour TSDB blocks to object storage. Estimate storage requirements based on series count, scrape interval, and retention.

### The Formula

Bytes per sample (compressed, delta-of-delta + XOR):

$$\bar{b}_{\text{sample}} \approx 1.5 \text{ bytes}$$

Daily storage:

$$\text{Daily (bytes)} = S_{\text{total}} \times \frac{86400}{\Delta t} \times \bar{b}_{\text{sample}} \times R_{\text{dedup}}$$

where $\Delta t$ is the scrape interval in seconds and $R_{\text{dedup}} = 1$ after compactor deduplication (down from $R$ replicas).

Total storage for retention $D$ days:

$$\text{Total} = \text{Daily} \times D$$

### Worked Examples

| Active Series | Scrape Interval | Daily (GB) | 90-day (TB) | 365-day (TB) | S3 Cost/mo (365d) |
|:---:|:---:|:---:|:---:|:---:|:---:|
| 1,000,000 | 15s | 8.6 | 0.78 | 3.15 | ~$72 |
| 5,000,000 | 15s | 43.2 | 3.89 | 15.77 | ~$363 |
| 10,000,000 | 15s | 86.4 | 7.78 | 31.54 | ~$725 |
| 10,000,000 | 30s | 43.2 | 3.89 | 15.77 | ~$363 |

S3 cost at $0.023/GB/month.

---

## 3. Query Splitting and Parallelism

### The Problem

The query frontend splits large PromQL queries by time to parallelize execution. Understanding the split factor determines query latency.

### The Formula

For a query spanning time range $[t_1, t_2]$ with split interval $\delta$:

$$\text{Sub-queries} = \left\lceil \frac{t_2 - t_1}{\delta} \right\rceil$$

With query sharding on label dimensions, each sub-query is further split by $S$ shards:

$$\text{Total Parallel Jobs} = \left\lceil \frac{t_2 - t_1}{\delta} \right\rceil \times S$$

$$T_{\text{query}} \approx \frac{\text{Total Work}}{P_{\text{max}}} + T_{\text{overhead}}$$

where $P_{\text{max}}$ is `max_query_parallelism` (default 14).

### Worked Examples

| Time Range | Split Interval ($\delta$) | Shards ($S$) | Total Jobs | Parallelism | Effective Rounds |
|:---:|:---:|:---:|:---:|:---:|:---:|
| 1h | 30m | 1 | 2 | 14 | 1 |
| 24h | 30m | 1 | 48 | 14 | 4 |
| 24h | 30m | 16 | 768 | 14 | 55 |
| 7d | 1h | 16 | 2,688 | 28 | 96 |

---

## 4. Compaction Write Amplification

### The Problem

The compactor merges overlapping 2-hour blocks into larger ones. Each level of compaction reads and rewrites data, causing write amplification.

### The Formula

With compaction levels $L$ where each level merges blocks at ratio $r$:

$$\text{Write Amplification} = \sum_{l=1}^{L} 1 = L$$

For time-based compaction windows $[2h, 12h, 24h]$:

$$A_{\text{write}} = \frac{\text{Total Bytes Written to Storage}}{\text{Original Ingested Bytes}} = 1 + L$$

After deduplication (removing $R-1$ replicas):

$$\text{Effective Amplification} = \frac{1 + L}{R}$$

### Worked Examples

| Levels ($L$) | Replication ($R$) | Ingested/day (GB) | Written/day (GB) | After Dedup (GB) |
|:---:|:---:|:---:|:---:|:---:|
| 3 | 3 | 86.4 | 345.6 | 115.2 |
| 3 | 1 | 86.4 | 345.6 | 345.6 |
| 2 | 3 | 86.4 | 259.2 | 86.4 |
| 4 | 3 | 86.4 | 432.0 | 144.0 |

---

## 5. Tenant Isolation (Shuffle Sharding)

### The Problem

In multi-tenant deployments, a misbehaving tenant should not affect others. Shuffle sharding assigns each tenant a subset of ingesters.

### The Formula

With $N$ ingesters and shard size $s$ per tenant, the probability that two tenants share an ingester:

$$P(\text{overlap}) = 1 - \frac{\binom{N-s}{s}}{\binom{N}{s}}$$

For large $N$, approximated as:

$$P(\text{overlap}) \approx 1 - \left(\frac{N-s}{N}\right)^s \approx 1 - e^{-s^2/N}$$

Blast radius (fraction of tenants affected if one ingester fails):

$$B = \frac{s}{N}$$

### Worked Examples

| Ingesters ($N$) | Shard Size ($s$) | P(2 tenants overlap) | Blast Radius |
|:---:|:---:|:---:|:---:|
| 30 | 3 | 26% | 10% |
| 30 | 5 | 58% | 17% |
| 100 | 3 | 9% | 3% |
| 100 | 5 | 22% | 5% |
| 100 | 10 | 63% | 10% |

---

## 6. Cardinality and Memory Bounds

### The Problem

Each unique combination of metric name and label key-value pairs creates a new time series. Unbounded cardinality exhausts ingester memory.

### The Formula

For a metric with $L$ labels, where label $i$ has $V_i$ distinct values:

$$S_{\text{max}} = \prod_{i=1}^{L} V_i$$

Memory upper bound:

$$M_{\text{max}} = S_{\text{max}} \times \bar{m}_{\text{series}}$$

Cardinality explosion factor when adding a new label with $V_{\text{new}}$ values:

$$\text{Explosion} = V_{\text{new}} \times$$

### Worked Examples

| Metric | Labels | Values per Label | Total Series | Memory |
|:---:|:---:|:---:|:---:|:---:|
| http_requests | method(4), status(5), path(10) | - | 200 | 2 MB |
| http_requests | + user_id(10,000) | - | 2,000,000 | 20 GB |
| http_requests | + request_id(inf) | - | unbounded | OOM |
| api_latency | method(4), endpoint(50) | - | 200 | 2 MB |

---

## Prerequisites

- Prometheus TSDB internals (chunks, blocks, WAL, compaction)
- Consistent hashing and hash rings (virtual nodes, replication)
- Object storage (S3/GCS API, eventual consistency, pricing)
- PromQL query execution model (instant vs range, step alignment)
- Distributed systems consensus (memberlist gossip, CRDTs)
- Combinatorics (cardinality, label explosions)
