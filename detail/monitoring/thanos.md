# The Mathematics of Thanos — Downsampling, Deduplication, and Global Query Cost

> *Thanos provides a global query view over distributed Prometheus instances. The mathematics cover TSDB block compaction, downsampling resolution trade-offs, deduplication merge logic, query fan-out cost across stores, and object storage bandwidth planning.*

---

## 1. Downsampling (Resolution Reduction)

### The Problem

Raw Prometheus data at 15-second scrape intervals generates enormous storage for long-term retention. Thanos downsamples to 5-minute and 1-hour resolutions, trading precision for storage efficiency.

### The Formula

Samples per series per time window $T$:

$$N_{\text{raw}} = \frac{T}{\text{scrape interval}}$$

$$N_{\text{5m}} = \frac{T}{300}, \quad N_{\text{1h}} = \frac{T}{3600}$$

Storage reduction ratio:

$$R_{\text{5m}} = \frac{N_{\text{raw}}}{N_{\text{5m}} \times 5} = \frac{\text{scrape interval}}{300 \times 5} \times N_{\text{raw}}$$

The 5x factor for 5m resolution accounts for storing min, max, sum, count, and counter aggregates per window.

### Worked Examples

10,000 series, 15-second scrape, stored for 1 year:

| Resolution | Samples/Series/Year | Bytes/Sample | Total Storage |
|:---:|:---:|:---:|:---:|
| Raw (15s) | 2,102,400 | 2 B | 39.1 GB |
| 5m (aggregated) | 105,120 x 5 | 2 B | 9.8 GB |
| 1h (aggregated) | 8,760 x 5 | 2 B | 0.8 GB |

Total savings from tiered retention (raw 30d, 5m 180d, 1h 365d):

$$S_{\text{tiered}} = S_{\text{raw,30d}} + S_{\text{5m,180d}} + S_{\text{1h,365d}}$$

$$= \frac{30}{365} \times 39.1 + \frac{180}{365} \times 9.8 + \frac{365}{365} \times 0.8$$

$$= 3.21 + 4.83 + 0.82 = 8.86 \text{ GB}$$

vs 39.1 GB all-raw: $77\%$ reduction.

---

## 2. TSDB Block Compaction (Merge Strategy)

### The Problem

Prometheus produces 2-hour TSDB blocks. Thanos compacts these into larger blocks (up to a configurable maximum) to reduce the number of objects in storage and improve query efficiency.

### The Formula

Compaction levels follow powers of the base block duration:

$$D_{\text{level } n} = D_{\text{base}} \times 3^{n-1}$$

With $D_{\text{base}} = 2h$:

| Level | Block Duration | Blocks Merged |
|:---:|:---:|:---:|
| 0 | 2h | (raw) |
| 1 | 6h | 3 blocks |
| 2 | 18h | 3 blocks |
| 3 | 54h | 3 blocks |

Number of blocks in storage for time range $T$:

$$B(T) = \sum_{n=0}^{L} \frac{T_n}{D_n}$$

### Worked Example

30 days of data, fully compacted to level 2 (18h blocks):

$$B = \frac{30 \times 24}{18} = 40 \text{ blocks}$$

vs uncompacted:

$$B_{\text{raw}} = \frac{30 \times 24}{2} = 360 \text{ blocks}$$

Block count reduction: $89\%$. This directly reduces Store Gateway index loading time and object storage LIST operations.

---

## 3. Deduplication (Replica Merge)

### The Problem

Multiple Prometheus replicas scrape the same targets. Thanos must deduplicate overlapping time series, keeping one copy per unique set of non-replica labels.

### The Formula

For $r$ replicas of each series, deduplication reduces storage by:

$$R_{\text{dedup}} = \frac{r - 1}{r}$$

The merge algorithm selects the replica with the most complete data (fewest gaps) per time window:

$$\text{Selected}(t) = \arg\max_{i \in [1..r]} \text{coverage}(i, t)$$

Gap-filling when one replica has missing data:

$$\text{value}(t) = \begin{cases} v_1(t) & \text{if } v_1(t) \text{ exists} \\ v_2(t) & \text{if } v_1(t) \text{ missing and } v_2(t) \text{ exists} \\ \text{NaN} & \text{otherwise} \end{cases}$$

### Worked Example

2 Prometheus replicas, each with 10,000 series, 5% gaps randomly distributed:

$$P(\text{both miss same sample}) = 0.05 \times 0.05 = 0.0025$$

Effective gap rate after dedup: $0.25\%$ (down from 5%).

Storage saved:

$$S_{\text{saved}} = \frac{1}{2} \times S_{\text{total}} = 50\%$$

---

## 4. Query Fan-Out Cost (Scatter-Gather)

### The Problem

Thanos Query fans out to all registered stores (sidecars, store gateways, receivers). The query latency is bounded by the slowest store.

### The Formula

Query latency:

$$L_{\text{query}} = L_{\text{planning}} + \max_{i=1}^{n}(L_{\text{store}_i}) + L_{\text{merge}}$$

Where $n$ is the number of stores contacted.

Merge cost for $n$ stores, each returning $k$ series with $s$ samples:

$$C_{\text{merge}} = O(n \times k \times s)$$

Network transfer:

$$B_{\text{total}} = \sum_{i=1}^{n} k_i \times s_i \times 16 \text{ bytes/sample}$$

### Worked Example

5 Prometheus clusters (5 sidecars + 1 store gateway), query returns 1000 series, 3600 samples each:

$$B = 6 \times 1000 \times 3600 \times 16 = 345.6 \text{ MB}$$

With dedup (2 replicas per cluster), actual transfer:

$$B_{\text{actual}} = 3 \times 1000 \times 3600 \times 16 = 172.8 \text{ MB}$$

Query Frontend caching reduces repeated queries to near-zero cost:

$$B_{\text{cached}} \approx 0 \text{ (cache hit)}$$

---

## 5. Object Storage Cost Model

### The Problem

Thanos stores all historical metrics in object storage. Costs depend on storage volume, API operations (PUT/GET/LIST), and data retrieval.

### The Formula

Monthly cost:

$$C = S \times P_{\text{storage}} + N_{\text{PUT}} \times P_{\text{PUT}} + N_{\text{GET}} \times P_{\text{GET}} + B_{\text{egress}} \times P_{\text{egress}}$$

PUT operations (from sidecar uploads + compactor):

$$N_{\text{PUT}} = \frac{T}{D_{\text{block}}} \times \text{series count} \times \text{files per block}$$

GET operations (from queries):

$$N_{\text{GET}} = Q_{\text{rate}} \times \bar{B}_{\text{per query}} \times 30 \text{ days}$$

### Worked Example

100,000 series, 15s scrape, 1-year retention, 100 queries/hour:

Storage: ~390 GB (tiered with downsampling)

$$C_{\text{S3}} = 390 \times 0.023 + \frac{365 \times 12}{1} \times 0.000005 + 100 \times 720 \times 5 \times 0.0000004 + 50 \times 0.09$$

$$= 8.97 + 0.02 + 0.14 + 4.50 = \$13.63/\text{month}$$

---

## 6. Query Frontend Splitting (Time Partitioning)

### The Problem

The Query Frontend splits long-range queries into smaller time intervals to parallelize execution and improve cacheability.

### The Formula

For a query over time range $[t_1, t_2]$ with split interval $I$:

$$\text{Sub-queries} = \left\lceil \frac{t_2 - t_1}{I} \right\rceil$$

Cache hit rate for queries with alignment to split boundaries:

$$H = 1 - \frac{2}{N_{\text{sub}}}$$

(First and last sub-queries may be partial/unique; interior ones are fully cacheable.)

### Worked Example

30-day range, 24h split:

$$N = \lceil 30 \rceil = 30 \text{ sub-queries}$$

$$H = 1 - \frac{2}{30} = 93.3\% \text{ (after first execution)}$$

With 6h split:

$$N = 120, \quad H = 1 - \frac{2}{120} = 98.3\%$$

---

## 7. Ruler Evaluation Cost (Global Aggregation)

### The Problem

Thanos Ruler evaluates recording and alerting rules against the global view. Each rule evaluation triggers a query spanning all stores.

### The Formula

Cost per evaluation interval $e$ with $R$ rules:

$$C_{\text{eval}} = R \times C_{\text{query avg}}$$

Queries per day:

$$Q_{\text{daily}} = R \times \frac{86{,}400}{e}$$

### Worked Example

50 recording rules, 1-minute evaluation interval:

$$Q_{\text{daily}} = 50 \times \frac{86{,}400}{60} = 72{,}000 \text{ queries/day}$$

If each query takes 200 ms average:

$$\text{Query load} = \frac{72{,}000 \times 0.2}{86{,}400} = 0.167 \text{ concurrent queries (average)}$$

Peak (all rules evaluate simultaneously):

$$\text{Peak load} = 50 \times 0.2 = 10 \text{ seconds of query time per minute}$$

$$\text{Peak utilization} = \frac{10}{60} = 16.7\%$$

---

## Prerequisites

- time-series-analysis, distributed-systems, information-theory, cost-optimization
