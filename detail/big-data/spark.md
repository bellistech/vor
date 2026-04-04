# The Mathematics of Spark — DAG Scheduling and Memory Models

> *Spark's execution model is a directed acyclic graph of stages separated by shuffle boundaries. Understanding partition sizing, memory allocation fractions, and shuffle cost functions is essential for tuning jobs that process terabytes efficiently.*

---

## 1. Partition Sizing (Data Distribution)

### The Problem

How many partitions should a Spark job use to balance parallelism against overhead?

### The Formula

$$P_{optimal} = \max\left(\frac{D}{T_{partition}}, E \times C\right)$$

Where:
- $D$ = total data size in bytes
- $T_{partition}$ = target partition size (128-256 MB typical)
- $E$ = number of executors
- $C$ = cores per executor

Partition count after shuffle:

$$P_{shuffle} = \texttt{spark.sql.shuffle.partitions}$$

With AQE:

$$P_{aqe} = \left\lceil \frac{S_{shuffle}}{T_{target}} \right\rceil$$

### Worked Examples

| Data Size | Target Part. | Executors x Cores | Optimal Partitions |
|:---:|:---:|:---:|:---:|
| 100 GB | 200 MB | 20 x 4 | max(512, 80) = 512 |
| 1 TB | 256 MB | 50 x 4 | max(4096, 200) = 4096 |
| 10 GB | 128 MB | 10 x 4 | max(80, 40) = 80 |

---

## 2. Unified Memory Model (Resource Allocation)

### The Problem

How is executor memory divided between storage (caching), execution (shuffles, joins, sorts), and user data structures?

### The Formula

Total usable memory (per executor):

$$M_{usable} = (M_{executor} - M_{reserved}) \times f_{fraction}$$

Where:
- $M_{executor}$ = `spark.executor.memory`
- $M_{reserved}$ = 300 MB (Spark internal overhead)
- $f_{fraction}$ = `spark.memory.fraction` (default 0.6)

Storage vs. execution split:

$$M_{storage} = M_{usable} \times f_{storage}$$
$$M_{execution} = M_{usable} \times (1 - f_{storage})$$

Where $f_{storage}$ = `spark.memory.storageFraction` (default 0.5).

The boundary is soft: execution can evict cached data if needed, but not vice versa.

$$M_{user} = (M_{executor} - M_{reserved}) \times (1 - f_{fraction})$$

### Worked Examples

| Executor Mem | Fraction | Storage Frac | Usable | Storage | Execution | User |
|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| 8 GB | 0.6 | 0.5 | 4.62 GB | 2.31 GB | 2.31 GB | 3.08 GB |
| 16 GB | 0.6 | 0.5 | 9.42 GB | 4.71 GB | 4.71 GB | 6.28 GB |
| 4 GB | 0.75 | 0.3 | 2.78 GB | 0.83 GB | 1.94 GB | 0.93 GB |

---

## 3. Shuffle Cost Analysis (I/O Complexity)

### The Problem

Shuffles are the most expensive operation in Spark. What determines the cost?

### The Formula

Shuffle write volume:

$$S_{write} = \sum_{i=1}^{P_{in}} |partition_i|$$

Shuffle read volume (with possible data expansion from serialization):

$$S_{read} = S_{write} \times (1 + \epsilon_{serde})$$

Sort-merge join cost:

$$T_{smj} = O\left(S_{total} \log \frac{S_{total}}{B}\right) + T_{network}$$

Broadcast join cost (when one side is small):

$$T_{bhj} = O(|R_{small}| \times E) + O(|R_{large}|)$$

Broadcast is cheaper when:

$$|R_{small}| \times E < S_{write}(R_{small}) + S_{write}(R_{large})$$

### Worked Examples

| Left Table | Right Table | Join Type | Shuffle Volume | Broadcast Cost |
|:---:|:---:|:---:|:---:|:---:|
| 500 GB | 50 MB | Broadcast | 0 (no shuffle) | 50 MB x 20 exec = 1 GB |
| 500 GB | 100 GB | Sort-Merge | ~600 GB | N/A (too large) |
| 50 GB | 2 GB | Broadcast | 0 | 2 GB x 20 = 40 GB |

---

## 4. DAG Stage Analysis (Graph Theory)

### The Problem

How does Spark decompose a job into stages, and what determines the critical path?

### The Formula

A job's DAG $G = (V, E)$ where vertices are RDD partitions and edges are dependencies:

- **Narrow dependency** (map, filter): one parent partition to one child
- **Wide dependency** (shuffle): all parent partitions to all child partitions

Stage boundaries occur at every wide dependency:

$$|Stages| = |ShuffleBoundaries| + 1$$

Critical path length:

$$T_{job} \geq \max_{path \in G} \sum_{stage \in path} T_{stage}$$

Stage execution time:

$$T_{stage} = \max_{task \in stage} T_{task} + T_{scheduler}$$

Skew factor:

$$\text{Skew} = \frac{T_{max\_task}}{T_{median\_task}}$$

A skew factor > 3 indicates significant data skew requiring mitigation.

### Worked Examples

| Operations | Narrow Deps | Wide Deps | Stages |
|:---:|:---:|:---:|:---:|
| read -> filter -> map -> write | 3 | 0 | 1 |
| read -> groupBy -> map -> join -> write | 1 | 2 | 3 |
| read -> repartition -> agg -> sort -> write | 1 | 3 | 4 |

---

## 5. Caching Efficiency (Memory Economics)

### The Problem

When does caching a DataFrame provide net benefit versus recomputation?

### The Formula

Caching is beneficial when:

$$T_{recompute} \times (N_{uses} - 1) > T_{cache\_write} + M_{cache} \times C_{memory}$$

Where:
- $T_{recompute}$ = time to recompute the DataFrame from source
- $N_{uses}$ = number of times the DataFrame is accessed
- $T_{cache\_write}$ = time to serialize and store in memory
- $M_{cache}$ = memory consumed by cached data
- $C_{memory}$ = opportunity cost of memory (evictions of other cached data)

Memory amplification factor (deserialized vs. on-disk):

$$A = \frac{M_{deserialized}}{M_{serialized}}$$

Typically $A \approx 2\text{-}5$ for JVM objects due to object headers, pointers, and boxing.

### Worked Examples

| DataFrame Size (disk) | Deserialized | Recompute Time | Uses | Cache Benefit |
|:---:|:---:|:---:|:---:|:---:|
| 10 GB | 35 GB | 120 s | 5 | 480 s saved |
| 10 GB | 35 GB | 120 s | 1 | -120 s (waste) |
| 1 GB | 3 GB | 5 s | 10 | 45 s saved |

---

## 6. Broadcast Variable Sizing (Network)

### The Problem

What is the network cost of broadcasting a variable to all executors?

### The Formula

BitTorrent-style broadcast in Spark:

$$T_{broadcast} = O\left(\frac{|V|}{BW} \times \log_2 E\right)$$

Where:
- $|V|$ = size of the broadcast variable
- $BW$ = network bandwidth per node
- $E$ = number of executors

Total memory consumed across the cluster:

$$M_{total} = |V| \times E$$

### Worked Examples

| Variable Size | Executors | Bandwidth | Broadcast Time | Cluster Memory |
|:---:|:---:|:---:|:---:|:---:|
| 100 MB | 50 | 1 Gbps | ~0.6 s | 5 GB |
| 1 GB | 100 | 10 Gbps | ~0.7 s | 100 GB |
| 10 MB | 20 | 1 Gbps | ~0.04 s | 200 MB |

---

## 7. Summary of Formulas

| Formula | Type | Domain |
|---------|------|--------|
| $P = \max(D/T, E \times C)$ | Partition sizing | Data distribution |
| $M_{usable} = (M - 300MB) \times 0.6$ | Memory model | Resource allocation |
| $T_{smj} = O(S \log(S/B))$ | Join cost | Shuffle analysis |
| $\|Stages\| = \|Shuffles\| + 1$ | Stage count | DAG theory |
| $\text{Skew} = T_{max}/T_{median}$ | Skew detection | Performance tuning |
| $T_{broadcast} = O((V/BW) \log E)$ | Broadcast cost | Network analysis |

## Prerequisites

- graph-theory, linear-algebra, probability, hadoop, jvm-memory-model

## Complexity

| Operation | Time | Space |
|-----------|------|-------|
| Map/Filter (narrow) | O(N) per partition | O(1) streaming |
| GroupByKey | O(N log N) sort | O(N) shuffle buffer |
| Sort-merge join | O(N log N) | O(N) shuffle |
| Broadcast join | O(N + M*E) | O(M) per executor |
| Cache (deserialized) | O(N) write | O(N * amplification) |
| Repartition | O(N) shuffle | O(N) network transfer |
