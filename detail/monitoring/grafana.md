# The Mathematics of Grafana — Dashboard and Visualization Internals

> *Grafana is a visualization platform that queries time series backends. The math covers query resolution, panel rendering, data point reduction, and capacity planning for dashboard performance.*

---

## 1. Time Resolution — Data Point Density

### The Model

Grafana requests data from backends at a resolution determined by the panel width and time range.

### Resolution Formula

$$\text{Step} = \max\left(\frac{\text{Time Range}}{\text{Max Data Points}}, \text{Min Step}\right)$$

Where:
- Max Data Points defaults to panel pixel width (typically 600-1920)
- Min Step = scrape interval of the data source

### Effective Resolution

$$\text{Points Returned} = \frac{\text{Time Range}}{\text{Step}}$$

### Worked Examples

| Time Range | Panel Width (px) | Step | Points | Resolution |
|:---:|:---:|:---:|:---:|:---:|
| 1 hour | 1200 | 3s | 1,200 | High |
| 6 hours | 1200 | 18s | 1,200 | Good |
| 24 hours | 1200 | 72s | 1,200 | Medium |
| 7 days | 1200 | 504s (~8m) | 1,200 | Low |
| 30 days | 1200 | 2,160s (~36m) | 1,200 | Very Low |

### Data Loss from Downsampling

If scrape interval = 15s:

$$\text{Samples Skipped} = \frac{\text{Step}}{\text{Scrape Interval}} - 1$$

| Time Range | Step | Scrape (15s) | Samples Skipped per Point |
|:---:|:---:|:---:|:---:|
| 1 hour | 3s | 15s | 0 (shows all) |
| 24 hours | 72s | 15s | 3.8 (shows 1 in 5) |
| 7 days | 504s | 15s | 32.6 (shows 1 in 34) |
| 30 days | 2,160s | 15s | 143 (shows 1 in 144) |

**Key insight:** Short spikes (<Step) become invisible at long time ranges.

---

## 2. Query Load — Backend Impact

### The Model

Each panel generates one or more queries to the data source. Dashboard load = sum of all panel queries.

### Query Load Formula

$$\text{Dashboard Queries} = \sum_{i=1}^{P} Q_i$$

Where $P$ = panels, $Q_i$ = queries per panel.

### Load from Viewers

$$\text{Total Query Rate} = \text{Dashboard Queries} \times \text{Viewers} \times \frac{1}{\text{Refresh Interval}}$$

### Worked Examples

*"Dashboard with 20 panels, 3 queries each, 10s refresh, 50 viewers."*

$$\text{Queries/sec} = 20 \times 3 \times 50 \times \frac{1}{10} = 300 \text{ queries/sec}$$

| Panels | Queries/Panel | Viewers | Refresh | Total QPS |
|:---:|:---:|:---:|:---:|:---:|
| 10 | 2 | 5 | 30s | 3.3 |
| 20 | 3 | 10 | 10s | 60 |
| 20 | 3 | 50 | 10s | 300 |
| 50 | 5 | 100 | 5s | 5,000 |

### Query Caching

$$\text{Cache Hit Ratio} = 1 - \frac{\text{Unique Queries}}{\text{Total Queries}}$$

If all viewers see the same dashboard:

$$\text{Unique Queries} = \text{Panels} \times \text{Queries/Panel}$$

$$\text{Cache Hit} = 1 - \frac{60}{300} = 80\%$$

---

## 3. Panel Rendering — Client-Side Performance

### The Model

Grafana renders panels in the browser. Performance depends on data point count and panel complexity.

### Rendering Time

$$T_{render} \approx \text{Data Points} \times T_{per\_point} + T_{layout}$$

| Data Points | Render Time (graph) | Render Time (table) |
|:---:|:---:|:---:|
| 100 | ~5 ms | ~2 ms |
| 1,000 | ~20 ms | ~10 ms |
| 10,000 | ~100 ms | ~50 ms |
| 100,000 | ~500 ms | ~200 ms |
| 1,000,000 | ~5,000 ms | ~2,000 ms |

### Total Dashboard Render

$$T_{dashboard} = T_{query\_max} + \sum_{i=1}^{P} T_{render\_i}$$

Queries run in parallel; rendering is mostly parallel (browser paint).

### Memory per Dashboard

$$\text{Browser Memory} \approx \text{Total Data Points} \times 16 \text{ bytes (timestamp + value)}$$

| Total Points | Memory |
|:---:|:---:|
| 10,000 | 156 KiB |
| 100,000 | 1.5 MiB |
| 1,000,000 | 15.3 MiB |
| 10,000,000 | 153 MiB |

---

## 4. Alert Rule Evaluation

### The Model

Grafana alerting evaluates rules at fixed intervals, querying backends and applying conditions.

### Evaluation Load

$$\text{Alert QPS} = \frac{\text{Alert Rules}}{\text{Evaluation Interval}}$$

### Alert State Machine

| State | Transition | Condition |
|:---|:---|:---|
| Normal | Normal -> Pending | Condition true |
| Pending | Pending -> Alerting | Condition true for `for` duration |
| Pending | Pending -> Normal | Condition false before `for` |
| Alerting | Alerting -> Normal | Condition false |
| NoData | Any -> NoData | No data for evaluation |

### Notification Volume

$$\text{Notifications/hour} = \text{Firing Alerts} \times \frac{60}{\text{Repeat Interval (min)}}$$

| Firing Alerts | Repeat Interval | Notifications/hour |
|:---:|:---:|:---:|
| 10 | 60 min | 10 |
| 10 | 5 min | 120 |
| 100 | 60 min | 100 |
| 100 | 5 min | 1,200 |

---

## 5. Variable Queries — Template Expansion

### The Model

Dashboard variables generate queries to populate dropdowns. High-cardinality variables multiply load.

### Variable Query Cost

$$\text{Variable Queries} = \text{Variable Count} \times \text{Refresh Frequency}$$

### Multi-Value Expansion

$$\text{Panel Queries with Multi-Value} = |V_1| \times |V_2| \times \ldots \times |V_k| \times Q_{base}$$

### Worked Example

*"2 multi-value variables: region (5 values), service (20 values), 3 queries per panel."*

$$\text{Queries per Panel} = 5 \times 20 \times 3 = 300$$

For 10 panels: $10 \times 300 = 3,000$ queries per dashboard load.

**This is the most common Grafana performance antipattern.** Use `$__all` or regex to reduce.

---

## 6. Data Source Connection Pooling

### The Model

Grafana maintains connection pools to data sources.

### Pool Sizing

$$\text{Connections Needed} = \frac{\text{Concurrent Queries}}{\text{Queries per Connection}}$$

$$\text{Concurrent Queries} = \text{Viewers} \times \text{Panels per Dashboard} \times P(\text{simultaneous})$$

| Viewers | Panels | Concurrent Factor | Connections |
|:---:|:---:|:---:|:---:|
| 10 | 20 | 0.1 | 20 |
| 50 | 20 | 0.1 | 100 |
| 100 | 30 | 0.1 | 300 |

### Rate Limiting

$$\text{Max QPS per Source} = \text{Connections} \times \frac{1}{T_{avg\_query}}$$

| Connections | Avg Query Time | Max QPS |
|:---:|:---:|:---:|
| 10 | 100 ms | 100 |
| 50 | 100 ms | 500 |
| 100 | 50 ms | 2,000 |

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| $\frac{\text{Range}}{\text{Width}}$ | Division | Query step/resolution |
| $P \times Q \times V \times \frac{1}{R}$ | Product | Total query rate |
| $\text{Points} \times T_{per}$ | Linear | Render time |
| $\prod |V_i| \times Q$ | Combinatorial | Multi-value expansion |
| $1 - \frac{\text{Unique}}{\text{Total}}$ | Ratio | Cache hit rate |
| $\frac{\text{Alerts}}{\text{Interval}}$ | Rate | Alert evaluation QPS |

---

*Every Grafana dashboard load triggers these calculations — a visualization layer where the math of resolution, caching, and cardinality determines whether your monitoring dashboard loads in milliseconds or times out.*
