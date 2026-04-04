# The Mathematics of dbt — DAG Scheduling, Incremental Logic, and Data Testing

> *dbt orchestrates SQL transformations as a directed acyclic graph. The math covers DAG critical path scheduling, incremental model cost analysis, test coverage probability, and warehouse slot utilization.*

---

## 1. DAG Critical Path (Graph Theory)

### The Problem

dbt builds models respecting dependencies. The total build time is bounded by the critical path (longest weighted path) through the DAG, not the sum of all model runtimes.

### The Formula

For a DAG $G = (V, E)$ where each node $v$ has execution time $w(v)$, the critical path length:

$$T_{\text{critical}} = \max_{\text{path } P \in G} \sum_{v \in P} w(v)$$

With $k$ parallel threads, the makespan (total wall time) is bounded:

$$T_{\text{makespan}} \geq \max\left(T_{\text{critical}}, \frac{\sum_{v \in V} w(v)}{k}\right)$$

The speedup from parallelism:

$$S(k) = \frac{\sum_{v \in V} w(v)}{T_{\text{makespan}}(k)} \leq \min\left(k, \frac{\sum_{v \in V} w(v)}{T_{\text{critical}}}\right)$$

### Worked Examples

A project with 50 models, total compute time 600s, critical path 120s:

| Threads ($k$) | Lower Bound | Speedup | Efficiency |
|:---:|:---:|:---:|:---:|
| 1 | 600s | 1.0x | 100% |
| 4 | 150s | 4.0x | 100% |
| 8 | 120s (critical path) | 5.0x | 62.5% |
| 16 | 120s (critical path) | 5.0x | 31.3% |

Beyond 5 threads, the critical path dominates. Adding threads yields no further improvement.

## 2. Incremental Model Cost (Optimization Theory)

### The Problem

Incremental models process only new or changed rows. The breakeven point depends on the ratio of new data to total data and the overhead of the merge operation.

### The Formula

Full refresh cost:

$$C_{\text{full}} = n \cdot c_{\text{scan}} + n \cdot c_{\text{write}}$$

Incremental cost:

$$C_{\text{incr}} = n \cdot c_{\text{scan\_src}} + \Delta n \cdot c_{\text{write}} + n_{\text{tgt}} \cdot c_{\text{merge}}$$

where $\Delta n$ is new rows, $n$ is source rows scanned (with predicate pushdown), and $n_{\text{tgt}}$ is target rows for merge lookup.

Incremental is cheaper when:

$$\frac{\Delta n}{n} < \frac{c_{\text{scan}} + c_{\text{write}} - c_{\text{scan\_src}}}{c_{\text{write}}} - \frac{n_{\text{tgt}} \cdot c_{\text{merge}}}{n \cdot c_{\text{write}}}$$

### Worked Examples

Table with 100M rows, 500K new rows daily, merge cost = 0.1x write cost:

$$\frac{C_{\text{incr}}}{C_{\text{full}}} = \frac{0.5M \cdot c_w + 100M \cdot 0.1 c_w}{100M \cdot c_w} = \frac{0.5 + 10}{100} = 10.5\%$$

| Total Rows | Daily New Rows | Full Cost | Incremental Cost | Savings |
|:---:|:---:|:---:|:---:|:---:|
| 10M | 100K | 100 units | 11 units | 89% |
| 100M | 500K | 1000 units | 105 units | 89.5% |
| 100M | 50M | 1000 units | 600 units | 40% |
| 100M | 90M | 1000 units | 1000 units | 0% |

When $\Delta n / n > 50\%$, full refresh is often faster due to merge overhead.

## 3. Test Coverage Analysis (Probability Theory)

### The Problem

dbt tests validate data quality. The probability of an undetected data quality issue depends on test coverage and the error distribution across columns and rows.

### The Formula

For a model with $C$ columns and 4 standard tests (unique, not_null, accepted_values, relationships), the coverage:

$$\text{Coverage} = \frac{T_{\text{applied}}}{C \cdot T_{\text{possible}}}$$

Probability of detecting an error with $t$ independent tests, each with detection probability $p_i$:

$$P_{\text{detect}} = 1 - \prod_{i=1}^{t} (1 - p_i)$$

For a not_null test on a column with $N$ rows and $e$ null errors:

$$P_{\text{detect\_null}} = 1 \quad \text{(deterministic, scans all rows)}$$

For a statistical test sampling $s$ rows from $N$ total, with error rate $r$:

$$P_{\text{detect}} = 1 - (1 - r)^s$$

### Worked Examples

| Error Rate | Sample Size | Detection Probability |
|:---:|:---:|:---:|
| 1% | 100 | 63.4% |
| 1% | 500 | 99.3% |
| 0.1% | 100 | 9.5% |
| 0.1% | 1000 | 63.2% |
| 0.01% | 10,000 | 63.2% |

dbt tests are exhaustive (full scan), so $P_{\text{detect}} = 1$ for any $e \geq 1$. The value of dbt tests lies in catching every single violation, not sampling.

## 4. Warehouse Slot Utilization (Scheduling Theory)

### The Problem

dbt runs compete for warehouse compute slots. Over-parallelism causes queueing; under-parallelism wastes capacity.

### The Formula

With $k$ dbt threads and $W$ warehouse slots, the utilization:

$$U = \frac{\sum_{v \in V} w(v)}{T_{\text{makespan}} \cdot \min(k, W)}$$

The optimal thread count minimizes makespan while maintaining high utilization:

$$k^* = \min\left(W, \left\lceil \frac{\sum_{v \in V} w(v)}{T_{\text{critical}}} \right\rceil\right)$$

Queue wait time when $k > W$ (M/M/c queueing model):

$$W_q = \frac{P_0 \cdot (\lambda/\mu)^W}{W! \cdot (1 - \rho/W)^2} \cdot \frac{1}{W \cdot \mu}$$

where $\rho = \lambda / \mu$ is the traffic intensity.

### Worked Examples

50 models, total compute 600s, critical path 120s, warehouse has 8 slots:

| dbt Threads | Makespan | Utilization | Queue Waits |
|:---:|:---:|:---:|:---:|
| 1 | 600s | 100% | None |
| 4 | 150s | 100% | None |
| 5 | 120s | 100% | None |
| 8 | 120s | 62.5% | None |
| 16 | 120s | 62.5% | Possible |

Optimal $k^* = \lceil 600/120 \rceil = 5$ threads.

## 5. Snapshot Storage Growth (SCD Type 2 Analysis)

### The Problem

Snapshots create a new record for every change, accumulating historical versions. Storage grows based on the change rate of source data.

### The Formula

After $d$ days with $N$ source records and daily change rate $r$:

$$\text{Rows}_d = N + N \cdot r \cdot d$$

Storage after $d$ days with average row size $s$:

$$S_d = (N + N \cdot r \cdot d) \cdot s = N \cdot s \cdot (1 + r \cdot d)$$

The ratio of snapshot size to source size:

$$\frac{S_d}{N \cdot s} = 1 + r \cdot d$$

### Worked Examples

Source table with 1M customers, 100 bytes/row:

| Change Rate ($r$) | After 30 Days | After 365 Days | After 3 Years |
|:---:|:---:|:---:|:---:|
| 0.1% | 1.03M rows | 1.365M rows | 2.1M rows |
| 1% | 1.3M rows | 4.65M rows | 11.95M rows |
| 5% | 2.5M rows | 19.25M rows | 55.75M rows |
| 10% | 4.0M rows | 37.5M rows | 110.5M rows |

With 10% daily change rate, snapshots grow to 110x the source after 3 years. Implement retention policies.

## 6. Ref Resolution and DAG Depth (Compilation Complexity)

### The Problem

dbt resolves ref() calls to build the DAG at compile time. The compilation cost grows with the number of models and the depth of the dependency graph.

### The Formula

For $M$ models with average fan-out $f$ (refs per model), the total edges:

$$E = M \cdot f$$

The DAG depth (longest dependency chain):

$$D = \max_{\text{path}} |\text{path}|$$

Compilation time scales with:

$$T_{\text{compile}} = M \cdot c_{\text{parse}} + E \cdot c_{\text{resolve}} + D \cdot c_{\text{validate}}$$

For a balanced DAG with branching factor $b$ and depth $D$:

$$M = \frac{b^D - 1}{b - 1} \approx b^D$$

### Worked Examples

| Models ($M$) | Avg Refs ($f$) | DAG Depth | Compile Time |
|:---:|:---:|:---:|:---:|
| 50 | 2 | 4 | ~2s |
| 200 | 3 | 6 | ~8s |
| 500 | 3 | 8 | ~20s |
| 2000 | 4 | 10 | ~90s |

Enable partial parsing to skip unchanged models: $T_{\text{incremental}} \approx T_{\text{full}} \cdot \frac{\Delta M}{M}$.

## Prerequisites

- Directed acyclic graphs (DAGs), topological sort, critical path method
- SQL query planning and execution cost models
- Probability theory for test coverage analysis
- Queueing theory (M/M/c models)
- Slowly changing dimensions (SCD Type 1, 2, 3)
- Data warehouse architecture (star schema, facts, dimensions)
