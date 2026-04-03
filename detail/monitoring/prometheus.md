# The Mathematics of Prometheus — Time Series Database Internals

> *Prometheus is a pull-based monitoring system with a custom TSDB. The math covers scrape intervals, storage sizing, query complexity, recording rules, and cardinality management.*

---

## 1. Scrape and Sample Math

### The Model

Prometheus scrapes targets at fixed intervals, collecting time series samples.

### Sample Rate

$$\text{Samples/sec} = \frac{\text{Time Series Count}}{\text{Scrape Interval (sec)}}$$

### Total Samples

$$\text{Total Samples} = \text{Time Series} \times \frac{\text{Retention Period}}{\text{Scrape Interval}}$$

### Worked Examples

| Time Series | Scrape Interval | Samples/sec | Samples in 15 days |
|:---:|:---:|:---:|:---:|
| 10,000 | 15s | 667 | 864,000,000 |
| 100,000 | 15s | 6,667 | 8,640,000,000 |
| 1,000,000 | 15s | 66,667 | 86,400,000,000 |
| 100,000 | 30s | 3,333 | 4,320,000,000 |
| 100,000 | 60s | 1,667 | 2,160,000,000 |

---

## 2. Storage Sizing — TSDB Internals

### The Model

Prometheus TSDB stores samples in compressed 2-hour blocks. Each sample is compressed to ~1-2 bytes using delta-of-delta encoding and XOR float compression.

### Storage Formula

$$\text{Disk Usage} = \text{Samples/sec} \times \text{Bytes per Sample} \times \text{Retention (sec)}$$

$$\text{Bytes per Sample} \approx 1.5 - 2.0 \text{ bytes (compressed)}$$

### Worked Examples

| Time Series | Scrape | Bytes/Sample | Retention | Storage |
|:---:|:---:|:---:|:---:|:---:|
| 10,000 | 15s | 1.5 | 15 days | 1.3 GiB |
| 100,000 | 15s | 1.5 | 15 days | 12.9 GiB |
| 100,000 | 15s | 1.5 | 90 days | 77.4 GiB |
| 1,000,000 | 15s | 2.0 | 15 days | 172.8 GiB |
| 1,000,000 | 15s | 2.0 | 90 days | 1.04 TiB |

### TSDB Block Structure

| Component | Content | Lifetime |
|:---|:---|:---|
| Head block | In-memory, last 2 hours | Active writes |
| Persistent blocks | On disk, 2-hour chunks | Compacted over time |
| WAL | Write-ahead log | Crash recovery |

### Memory Usage

$$\text{Memory} \approx \text{Active Series} \times 4 \text{ KiB (index + head samples)}$$

| Active Series | Memory |
|:---:|:---:|
| 10,000 | 40 MiB |
| 100,000 | 400 MiB |
| 1,000,000 | 3.9 GiB |
| 10,000,000 | 39 GiB |

---

## 3. Cardinality — The Explosion Problem

### The Model

Cardinality = number of unique time series. High cardinality is the primary scaling challenge.

### Cardinality Formula

$$\text{Cardinality} = \prod_{i=1}^{n} |L_i|$$

Where $|L_i|$ = number of unique values for label $i$.

### Worked Example

*"Metric `http_requests_total` with labels: method (5), status (10), path (100), instance (50)."*

$$\text{Cardinality} = 5 \times 10 \times 100 \times 50 = 250,000 \text{ time series}$$

### Cardinality Explosion Table

| Labels | Values Each | Cardinality |
|:---|:---:|:---:|
| 2 labels × 10 values | 10 | 100 |
| 3 labels × 10 values | 10 | 1,000 |
| 4 labels × 10 values | 10 | 10,000 |
| 3 labels × 100 values | 100 | 1,000,000 |
| 4 labels × 100 values | 100 | 100,000,000 |

**Adding one high-cardinality label (e.g., user_id with 1M values) can multiply cardinality by 1M.**

### Cardinality Budget

$$\text{Recommended Max} \approx 1-10 \text{ million active series}$$

$$\text{Cost per Series} = \text{Memory (4 KiB)} + \text{Disk (bytes/sample × samples)} + \text{Query Load}$$

---

## 4. PromQL Query Complexity

### Range Query Cost

$$T_{range\_query} = O(S \times \frac{R}{\text{Scrape Interval}})$$

Where:
- $S$ = series matched by selector
- $R$ = range duration

### Aggregation Cost

$$T_{agg} = O(S) \quad (\text{for sum, avg, max, min over S series})$$

### Rate/Increase Calculation

$$\text{rate}(v[d]) = \frac{v_{last} - v_{first}}{t_{last} - t_{first}}$$

$$\text{irate}(v[d]) = \frac{v_{n} - v_{n-1}}{t_{n} - t_{n-1}} \quad (\text{last two samples only})$$

### Query Performance

| Query Type | Series | Range | Samples Processed |
|:---|:---:|:---:|:---:|
| Instant vector | 1,000 | 0 | 1,000 |
| Range (5m, 15s scrape) | 1,000 | 5m | 20,000 |
| Range (1h, 15s scrape) | 1,000 | 1h | 240,000 |
| Range (24h, 15s scrape) | 10,000 | 24h | 57,600,000 |

---

## 5. Recording Rules — Pre-computation

### The Model

Recording rules pre-compute expensive queries at scrape time, trading storage for query speed.

### Cost-Benefit Analysis

$$\text{Storage Cost} = \text{Output Series} \times \frac{\text{Retention}}{\text{Evaluation Interval}} \times \text{Bytes/Sample}$$

$$\text{Query Savings} = \text{Query Rate} \times (T_{original} - T_{precomputed})$$

### Worked Example

*"Recording rule aggregates 10,000 series into 100 series."*

$$\text{Query speedup} = \frac{10,000}{100} = 100\times$$

$$\text{Extra storage} = 100 \text{ series} \times \frac{15 \text{ days}}{15 \text{s}} \times 1.5 \text{ bytes} = 12.9 \text{ MiB}$$

**12.9 MiB of storage saves 100x query time — almost always worth it.**

---

## 6. Alerting Math

### Alert Evaluation

$$\text{Pending Duration} = \text{for clause duration}$$

$$\text{Alert fires after} = \text{for} + \text{evaluation\_interval (worst case)}$$

### Missing Scrapes

$$P(\text{no data}) = P(\text{target down}) + P(\text{network issue}) + P(\text{timeout})$$

$$\text{Staleness timeout} = 5 \text{ minutes (default)}$$

A series becomes stale if no sample arrives within 5 minutes. Alerts on stale series resolve.

### Alert Grouping

$$\text{Notifications} = \lceil \frac{\text{Firing Alerts}}{\text{group\_by cardinality}} \rceil$$

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| $\frac{\text{Series}}{\text{Interval}}$ | Rate | Samples per second |
| $\text{Series} \times 4 \text{KiB}$ | Linear scaling | Memory sizing |
| $\prod |L_i|$ | Combinatorial product | Cardinality |
| $\frac{v_{last} - v_{first}}{t_{last} - t_{first}}$ | Slope | rate() function |
| $S \times \frac{R}{\text{Interval}}$ | Product | Query sample count |
| $\text{Samples/s} \times \text{Bytes} \times T$ | Triple product | Storage sizing |

---

*Every `promtool tsdb analyze`, `prometheus --storage.tsdb.retention.time`, and PromQL query reflects these internals — a monitoring system where cardinality management is the primary engineering challenge.*
