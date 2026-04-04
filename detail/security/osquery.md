# The Mathematics of osquery -- Query Optimization and Endpoint Telemetry

> *The operating system is a database; every process, socket, and file is a row waiting to be queried -- but careless queries cost more than the threats they seek.*

---

## 1. Query Cost Estimation (Relational Algebra)

### The Problem

osquery tables vary dramatically in cost. Some tables (like `processes`) read kernel state in O(n) where n is the number of processes. Others (like `file` or `hash`) perform disk I/O per row, making unbounded queries potentially catastrophic on production systems. Estimating query cost before execution is essential for safe scheduling.

### The Formula

The cost of a query Q joining tables $T_1, T_2, \ldots, T_k$ with filter selectivity $\sigma_i$ on each table:

$$C(Q) = \sum_{i=1}^{k} |T_i| \cdot c_i + \sum_{(i,j) \in J} |T_i| \cdot \sigma_i \cdot |T_j| \cdot \sigma_j \cdot c_{join}$$

where $|T_i|$ is the cardinality of table i, $c_i$ is the per-row cost (CPU for virtual tables, I/O for disk-backed), and $c_{join}$ is the join cost. The per-row cost varies by table type:

$$c_i = \begin{cases} c_{\text{kernel}} \approx 1\mu s & \text{(processes, mounts)} \\ c_{\text{disk}} \approx 100\mu s & \text{(file, hash)} \\ c_{\text{network}} \approx 10ms & \text{(curl, yara)} \end{cases}$$

Filter pushdown optimization reduces cost when the WHERE clause constrains the virtual table's key column:

$$C_{\text{optimized}} = |T_i| \cdot \sigma_i \cdot c_i \quad \text{vs.} \quad C_{\text{naive}} = |T_i| \cdot c_i$$

### Worked Examples

**Example 1: Process-to-socket join**

Query: `SELECT * FROM processes p JOIN process_open_sockets s ON p.pid = s.pid`

Assume 500 processes, 2000 open sockets, kernel read cost = 1 us/row.
- Scan processes: 500 * 1 us = 0.5 ms
- Scan sockets: 2000 * 1 us = 2 ms
- Hash join: 500 entries in hash table, 2000 probes = 2000 * 0.1 us = 0.2 ms
- Total: approximately 2.7 ms

**Example 2: Unbounded file hash query**

Query: `SELECT path, sha256 FROM hash WHERE directory = '/usr/'` (recursive)

Assume /usr/ contains 50,000 files, SHA-256 cost = 500 us per average file.
- Total: 50,000 * 500 us = 25 seconds
- With filter `WHERE path LIKE '/usr/bin/%'` (2,000 files): 2,000 * 500 us = 1 second
- Cost reduction: 25x improvement from filter pushdown

## 2. Differential Query Results (Set Difference)

### The Problem

osquery's scheduled queries can run in differential mode, reporting only changes between consecutive executions. This dramatically reduces log volume but requires efficient computation of set differences. Understanding the expected change rate helps in capacity planning and anomaly detection.

### The Formula

Given snapshots $S_t$ and $S_{t+1}$ of a table at consecutive intervals, the differential result is:

$$\Delta_{\text{added}} = S_{t+1} \setminus S_t$$
$$\Delta_{\text{removed}} = S_t \setminus S_{t+1}$$

The change rate r over interval delta_t:

$$r = \frac{|\Delta_{\text{added}}| + |\Delta_{\text{removed}}|}{|S_t|}$$

For log volume estimation, if snapshot cardinality is N and change rate is r per interval:

$$V_{\text{snapshot}} = N \cdot s \quad \text{(bytes per interval)}$$
$$V_{\text{differential}} = N \cdot r \cdot s \quad \text{(bytes per interval)}$$

The compression ratio of differential vs. snapshot:

$$\text{ratio} = \frac{V_{\text{differential}}}{V_{\text{snapshot}}} = r$$

For a Poisson process model of changes with rate lambda:

$$P(|\Delta| = k) = \frac{(\lambda \cdot \Delta t)^k \cdot e^{-\lambda \cdot \Delta t}}{k!}$$

### Worked Examples

**Example 1: Process table differential**

Average running processes N = 500. Process creation rate lambda = 20/minute. Query interval delta_t = 5 minutes.

- Expected changes per interval: lambda * delta_t = 100 (new + terminated)
- Snapshot log volume: 500 * 200 bytes = 100 KB per interval
- Differential log volume: 100 * 200 bytes = 20 KB per interval
- Compression ratio: r = 100/500 = 0.20 (80% reduction)

**Example 2: Installed packages**

Packages N = 1,500. Change rate lambda = 0.5/day. Query interval delta_t = 1 hour.

- Expected changes: 0.5/24 = 0.021 per interval
- P(0 changes) = e^(-0.021) = 0.979
- 97.9% of intervals generate zero log entries
- Differential saves: 1,500 * 200 * 24 = 7.2 MB/day vs near-zero

## 3. Fleet Query Distribution (Scheduling Theory)

### The Problem

When managing thousands of endpoints, all hosts running the same query at the same instant creates thundering herd problems at the log aggregation layer. Schedule splaying distributes query execution over a window, but must balance load smoothing against detection latency.

### The Formula

With n hosts, query interval I, and splay percentage p, each host's actual execution time is:

$$t_{\text{exec}}(h) = t_{\text{scheduled}} + U(0, I \cdot p)$$

where $U(0, I \cdot p)$ is a uniform random delay. The peak arrival rate at the log collector without splay:

$$\lambda_{\text{peak}} = n \cdot \frac{1}{I}$$

With splay, the arrival rate smooths to:

$$\lambda_{\text{splayed}} = \frac{n}{I \cdot p}$$

The reduction in peak rate:

$$\text{reduction} = 1 - \frac{1}{p} \cdot \frac{1}{n}$$

For worst-case detection latency with splay:

$$T_{\text{detect,max}} = I + I \cdot p = I(1 + p)$$

### Worked Examples

**Example 1: 10,000 host fleet**

Query interval I = 300s, splay p = 0.10.

Without splay: 10,000 results arrive within 1 second.
- Peak rate: 10,000 results/second

With splay: results spread over 30 seconds.
- Smoothed rate: 10,000/30 = 333 results/second
- 30x reduction in peak collector load
- Worst-case detection latency: 300 * 1.10 = 330 seconds

**Example 2: Critical security query**

Compromise detection query, interval I = 60s, fleet n = 5,000.

With p = 0.05 (minimal splay): spread over 3 seconds.
- Smoothed rate: 5,000/3 = 1,667/s
- Max detection latency: 63 seconds
- Acceptable for near-real-time detection

## 4. FIM Change Detection (Information-Theoretic Approach)

### The Problem

File integrity monitoring generates events for every file change in monitored directories. In active directories (log files, temp directories, caches), the event rate overwhelms analysts. Quantifying the "information content" of a change helps prioritize alerts by unexpectedness.

### The Formula

The self-information (surprisal) of a file change event, given the historical change frequency f(path) over observation period T:

$$I(\text{change at path}) = -\log_2 P(\text{change}) = -\log_2 \frac{f(\text{path})}{T}$$

High surprisal indicates an unusual change (more informative). For a monitored directory with N files, the entropy of the change distribution:

$$H = -\sum_{i=1}^{N} p_i \log_2 p_i$$

where $p_i = f_i / \sum f_j$. Files that have never changed in the observation window get maximum surprisal:

$$I_{\text{max}} = \log_2 T \quad \text{(bits, assuming unit-time resolution)}$$

The expected number of "interesting" events (surprisal above threshold theta) per day:

$$E[\text{interesting}] = \sum_{i=1}^{N} \mathbb{1}[I_i > \theta] \cdot \lambda_i$$

### Worked Examples

**Example 1: /etc/ monitoring**

1,000 files monitored for 90 days. `/etc/resolv.conf` changes 5 times. `/etc/shadow` has never changed.

- resolv.conf surprisal: -log_2(5/90) = -log_2(0.056) = 4.17 bits
- shadow change surprisal: -log_2(1/90) = 6.49 bits (using Laplace smoothing: 1 observed in 90 days)
- Actual zero-history surprisal: log_2(90) = 6.49 bits (maximum for this window)

If `/etc/shadow` suddenly changes: 6.49 bits vs. system-log at 0.1 bits. Shadow change is 64x more informative.

**Example 2: Alert prioritization**

1,000 FIM events/day from /etc/. Entropy H = 3.2 bits. Threshold theta = 5 bits.
- Events above threshold: approximately 1000 * 2^(-5) = 31 events/day
- These 31 events cover the rarely-changing files that warrant investigation
- 96.9% noise reduction compared to raw event feed

## Prerequisites

- Relational algebra (joins, projections, selection, cost estimation)
- Set theory (set difference, symmetric difference)
- Probability theory (Poisson processes, uniform distribution)
- Information theory (entropy, self-information, surprisal)
- Queueing theory (arrival rates, thundering herd)
- Computational complexity (I/O vs. CPU cost models)
- Operating system internals (procfs, file descriptors, kernel state)
