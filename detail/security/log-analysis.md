# The Mathematics of Log Analysis — Pattern Recognition and Statistical Anomaly Detection

> *Log analysis is applied statistics: event frequencies follow distributions, anomalies are deviations from baselines, and correlation across sources transforms individual events into attack narratives. The mathematics span information theory, time series analysis, and graph-based event correlation.*

---

## 1. Log Volume and Throughput

### Events Per Second (EPS)

$$\text{EPS} = \sum_{s \in \text{sources}} R_s$$

| Source | Typical EPS | Peak EPS |
|:---|:---:|:---:|
| Firewall | 100-10,000 | 100,000 |
| Web server (access log) | 50-5,000 | 50,000 |
| Auth/login events | 1-100 | 1,000 |
| DNS resolver | 500-50,000 | 500,000 |
| Endpoint agent | 10-100 per host | 1,000 |
| Database audit | 10-1,000 | 10,000 |

### Storage Estimation

$$V_{daily} = \text{EPS}_{avg} \times S_{avg} \times 86400$$

| EPS | Avg Event Size | Daily Volume | 90-Day Retention |
|:---:|:---:|:---:|:---:|
| 100 | 500 bytes | 4.3 GB | 387 GB |
| 1,000 | 500 bytes | 43 GB | 3.87 TB |
| 10,000 | 500 bytes | 430 GB | 38.7 TB |
| 100,000 | 500 bytes | 4.3 TB | 387 TB |

### Compression

Log data compresses well (repetitive text):

$$S_{compressed} = S_{raw} \times (1 - r) \quad \text{where } r \approx 0.85-0.92$$

Typical 10:1 compression ratio. 430 GB/day becomes ~43 GB/day compressed.

---

## 2. Baseline Modeling

### Normal Distribution Baseline

For event count $X$ in a time window:

$$X \sim N(\mu, \sigma^2)$$

Estimated from historical data:

$$\mu = \frac{1}{n}\sum_{i=1}^{n} x_i, \quad \sigma = \sqrt{\frac{1}{n-1}\sum_{i=1}^{n}(x_i - \mu)^2}$$

### Anomaly Threshold

$$\text{Alert if } |x - \mu| > k\sigma$$

| $k$ | Normal Traffic Flagged | Detection Sensitivity |
|:---:|:---:|:---:|
| 2 | 4.6% | High (many false positives) |
| 3 | 0.27% | Balanced |
| 4 | 0.006% | Low (may miss subtle attacks) |

### Time-of-Day Baseline

Many log patterns are periodic (24-hour cycle):

$$\mu(t) = a_0 + \sum_{k=1}^{K} a_k \cos\left(\frac{2\pi k t}{T}\right) + b_k \sin\left(\frac{2\pi k t}{T}\right)$$

Where $T = 86400$ seconds (24 hours) and $K = 3$ captures daily/weekly patterns.

An event at 3 AM matching a pattern typical for 2 PM is anomalous even if the absolute count is normal.

---

## 3. Shannon Entropy for Anomaly Detection

### Request Entropy

For a distribution of request types $\{p_1, p_2, \ldots, p_n\}$:

$$H = -\sum_{i=1}^{n} p_i \log_2(p_i)$$

| Traffic Pattern | Entropy | Interpretation |
|:---|:---:|:---|
| Normal web traffic | 3.5-5.0 | Diverse request types |
| Brute force attack | 0.5-1.0 | Repetitive (same endpoint) |
| Web scan | 5.0-7.0 | Many different URLs probed |
| DDoS | 0.1-0.5 | Nearly identical requests |

### Worked Example

Normal: 40% GET /, 30% GET /api, 20% POST /login, 10% other:

$$H = -0.4\log_2(0.4) - 0.3\log_2(0.3) - 0.2\log_2(0.2) - 0.1\log_2(0.1) = 1.85$$

Under brute force: 95% POST /login, 5% other:

$$H = -0.95\log_2(0.95) - 0.05\log_2(0.05) = 0.29$$

Entropy drop from 1.85 to 0.29 signals an anomaly.

---

## 4. Correlation Rules — Event Chaining

### Temporal Correlation

Events $e_1, e_2$ are correlated if:

$$|t_{e_1} - t_{e_2}| \leq w \quad \text{AND} \quad \text{common field}(e_1, e_2) \neq \emptyset$$

Common fields: source IP, user, session ID, process ID.

### Attack Chain Detection

A SIEM correlation rule detects ordered event sequences:

$$\text{Alert if } e_1 \xrightarrow{<t_1} e_2 \xrightarrow{<t_2} e_3$$

Example: Failed logins followed by success followed by data access:

| Step | Event | Window | Source |
|:---:|:---|:---:|:---|
| 1 | 5+ failed logins (same user) | 10 min | Auth log |
| 2 | Successful login (same user) | 5 min after step 1 | Auth log |
| 3 | Access to sensitive file | 30 min after step 2 | File audit |

### False Positive Rate of Chains

If each event has independent FP rate $p_i$:

$$P(\text{chain FP}) = \prod_{i=1}^{n} p_i \times P(\text{temporal match})$$

Longer chains have exponentially lower false positive rates:

| Chain Length | Individual FP Rate | Chain FP Rate |
|:---:|:---:|:---:|
| 1 | 1% | 1% |
| 2 | 1% | 0.01% |
| 3 | 1% | 0.0001% |
| 4 | 1% | 0.000001% |

This is why multi-event correlation dramatically improves precision.

---

## 5. Log Parsing — Regular Expression Performance

### Regex Complexity

| Pattern Type | Example | Matching Complexity |
|:---|:---|:---:|
| Fixed string | `"Failed password"` | $O(n)$ |
| Simple alternation | `(ssh\|sshd)` | $O(n)$ |
| Character class | `[0-9]{1,3}\.[0-9]{1,3}` | $O(n)$ |
| Backreference | `(\w+) \1` | $O(n^2)$ possible |
| Catastrophic backtracking | `(a+)+$` | $O(2^n)$ pathological |

### Parsing Throughput

$$T_{parse} = \frac{n_{events} \times n_{patterns} \times T_{regex}}{n_{threads}}$$

| Events/s | Patterns | Threads | CPU Required |
|:---:|:---:|:---:|:---:|
| 1,000 | 50 | 4 | 0.1 cores |
| 10,000 | 100 | 4 | 2.5 cores |
| 100,000 | 100 | 8 | 12.5 cores |
| 1,000,000 | 200 | 16 | 125 cores |

At high EPS, structured logging (JSON) with field extraction is 10-100x faster than regex parsing.

---

## 6. SIEM Retention and Search

### Index Size

$$S_{index} = V_{raw} \times r_{index}$$

Where $r_{index}$ is the index ratio:

| SIEM | Index Ratio | 100 GB Raw = Index |
|:---|:---:|:---:|
| Elasticsearch | 1.0-1.5x | 100-150 GB |
| Splunk | 0.5-0.7x | 50-70 GB |
| Loki (log-only) | 0.05-0.1x | 5-10 GB |

### Search Performance

Full-text search across $n$ events:

$$T_{search} = \frac{n}{R_{search}}$$

| Technology | Search Rate | 1B Events |
|:---|:---:|:---:|
| grep (raw files) | 100K/s | 2.8 hours |
| Elasticsearch | 10M/s | 100 seconds |
| Columnar (ClickHouse) | 100M/s | 10 seconds |

### Retention Cost

$$C_{retain} = V_{daily} \times T_{days} \times C_{storage}$$

| Retention | Volume (10K EPS) | S3 Cost/Month | Hot Storage Cost/Month |
|:---:|:---:|:---:|:---:|
| 30 days | 1.3 TB | $30 | $200 |
| 90 days | 3.9 TB | $90 | $600 |
| 365 days | 15.7 TB | $360 | $2,400 |

Hot tier for search, cold/frozen tier for compliance — tiered storage reduces cost by 80%+.

---

## 7. Threat Intelligence Matching

### IOC Matching Rate

$$\text{Hit rate} = \frac{|\text{events matching IOC}|}{|\text{total events}|}$$

Typical IOC match rates:

| IOC Type | Items in Feed | Match Rate | True Positive Rate |
|:---|:---:|:---:|:---:|
| IP addresses | 100K | 0.01-0.1% | 30-60% |
| Domains | 500K | 0.001-0.01% | 40-70% |
| File hashes | 1M | 0.0001-0.001% | 80-95% |
| URLs | 200K | 0.001-0.01% | 50-80% |

### IOC Decay

Threat intelligence ages — IOCs become stale:

$$\text{Relevance}(t) = e^{-\lambda t}$$

| IOC Age | Relevance (IP) | Relevance (Hash) |
|:---:|:---:|:---:|
| 1 day | 95% | 99% |
| 7 days | 70% | 95% |
| 30 days | 30% | 85% |
| 90 days | 5% | 60% |
| 365 days | ~0% | 20% |

IP indicators decay fast (infrastructure rotates); file hashes remain useful longer.

---

## 8. Summary of Functions by Type

| Concept | Math Type | Application |
|:---|:---|:---|
| $\mu \pm k\sigma$ threshold | Normal distribution | Anomaly detection |
| Shannon entropy $H$ | Information theory | Traffic pattern analysis |
| $\prod p_i$ chain FP | Probability product | Correlation precision |
| Fourier $a_k \cos + b_k \sin$ | Time series decomposition | Time-of-day baseline |
| $V \times T \times C$ | Linear scaling | Retention cost |
| $e^{-\lambda t}$ decay | Exponential | IOC relevance |

## Prerequisites

- statistics (baseline, standard deviation), regular expressions, entropy, time series

## Complexity

| Operation | Time | Space |
|-----------|------|-------|
| Pattern search (regex) | O(n * m) | O(m) |
| Baseline computation | O(n) | O(w) |
| Entropy calculation | O(n) | O(k) |

---

*Log analysis transforms raw event streams into security intelligence — the mathematics of baselining, entropy, and correlation convert millions of events per day into the handful of alerts that matter, separating signal from noise at machine speed.*
