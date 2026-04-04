# The Mathematics of Kafka Streams — Stream Processing Algebra

> *Kafka Streams implements a stream processing algebra where KStreams and KTables form a duality analogous to differentials and integrals. The mathematics covers join semantics, windowed aggregation complexity, state store sizing, and exactly-once transaction costs.*

---

## 1. Stream-Table Duality (Algebraic Foundation)

### The Problem

KStream and KTable are dual representations of the same data. Understanding this duality is essential for choosing the right abstraction.

### The Formula

A KTable is the integral (aggregation) of a KStream:

$$\text{KTable}(k, t) = \bigoplus_{e \in \text{KStream}, e.key = k, e.time \leq t} e.value$$

A KStream is the differential (changelog) of a KTable:

$$\text{KStream}(t) = \text{KTable}(t) - \text{KTable}(t - 1)$$

For a count aggregation:

$$\text{KTable}[k] = \sum_{i=1}^{n_k} 1 = n_k$$

Where $n_k$ is the number of events with key $k$.

For a reduce with function $f$:

$$\text{KTable}[k] = f(f(\ldots f(v_1, v_2), v_3), \ldots, v_{n_k})$$

### Worked Examples

| KStream Events | KTable State | Operation |
|:---:|:---:|:---:|
| (A,1), (B,2), (A,3) | {A:3, B:2} | Latest value |
| (A,1), (B,2), (A,3) | {A:2, B:1} | Count |
| (A,10), (B,20), (A,30) | {A:40, B:20} | Sum |

---

## 2. Join Semantics (Relational Algebra)

### The Problem

Different join types have different semantics and performance characteristics. When does each apply?

### The Formula

KStream-KStream join (windowed, within time window $w$):

$$R = \{(a, b) : a \in S_1, b \in S_2, a.key = b.key, |a.time - b.time| \leq w\}$$

Output cardinality (worst case):

$$|R| \leq |S_1| \times |S_2|$$

Expected cardinality with uniform key distribution:

$$E[|R|] = \frac{|S_1| \times |S_2|}{K} \times \frac{2w}{T}$$

Where $K$ = number of distinct keys, $T$ = total time span.

KStream-KTable join (lookup):

$$R = \{(s, t[s.key]) : s \in S, s.key \in T\}$$

Output cardinality: $|R| = |S|$ (one output per stream event).

KTable-KTable join:

$$R = \{(a, b) : a \in T_1, b \in T_2, a.key = b.key\}$$

Output cardinality: $|R| \leq \min(|T_1|, |T_2|)$ (at most one per key).

### Worked Examples

| Join Type | Left Size | Right Size | Window | Max Output |
|:---:|:---:|:---:|:---:|:---:|
| Stream-Stream | 1M/hr | 500K/hr | 5 min | 1M * 500K * 10min/60min / K |
| Stream-Table | 1M/hr | 100K keys | - | 1M/hr |
| Table-Table | 100K keys | 50K keys | - | 50K (intersection) |

---

## 3. State Store Sizing (Storage)

### The Problem

How much local disk space does a Kafka Streams application need for its state stores?

### The Formula

State store size per instance (with $p$ partitions and $n$ instances):

$$S_{instance} = \frac{K \times (S_{key} + S_{value})}{n} \times A_{rocksdb}$$

Where:
- $K$ = total number of distinct keys
- $S_{key}$ = average serialized key size
- $S_{value}$ = average serialized value size
- $A_{rocksdb}$ = RocksDB amplification factor (typically 1.5-3x due to LSM tree levels)

Windowed store size:

$$S_{windowed} = K \times W_{retained} \times (S_{key} + S_{value} + 8) \times A_{rocksdb}$$

Where $W_{retained}$ = number of retained windows per key, and 8 bytes for the window timestamp suffix.

Changelog topic size (for fault tolerance):

$$S_{changelog} = K \times (S_{key} + S_{value}) \times R$$

Where $R$ = replication factor (typically 3).

### Worked Examples

| Keys | Key Size | Value Size | Instances | Per Instance | Changelog |
|:---:|:---:|:---:|:---:|:---:|:---:|
| 10M | 32 B | 256 B | 4 | ~1.7 GB | ~8.6 GB |
| 100M | 16 B | 128 B | 8 | ~2.7 GB | ~43 GB |
| 1M | 64 B | 1 KB | 2 | ~1.6 GB | ~3.2 GB |

---

## 4. Exactly-Once Transaction Cost (Overhead)

### The Problem

What is the performance overhead of exactly-once processing in Kafka Streams?

### The Formula

With `exactly_once_v2` (EOS v2), Kafka uses a single transaction per task commit:

$$T_{commit} = T_{flush} + T_{txn\_begin} + T_{produce} + T_{txn\_commit}$$

Throughput with EOS:

$$\text{Throughput}_{eos} = \frac{B_{commit}}{T_{commit}}$$

Where $B_{commit}$ = records per commit batch (controlled by `commit.interval.ms`).

Overhead ratio:

$$\text{Overhead} = \frac{T_{txn\_begin} + T_{txn\_commit}}{T_{commit}}$$

EOS v2 improvement over v1 (fewer producers):

$$\text{Producers}_{v1} = \text{partitions}$$
$$\text{Producers}_{v2} = \text{stream threads}$$

### Worked Examples

| Commit Interval | Records/Commit | Txn Overhead | Throughput Impact |
|:---:|:---:|:---:|:---:|
| 100 ms | 10K | 5 ms | ~5% |
| 30 ms | 3K | 5 ms | ~17% |
| 1000 ms | 100K | 5 ms | ~0.5% |

---

## 5. Repartitioning Cost (Network)

### The Problem

Operations like `selectKey()`, `groupBy()`, and `map()` that change the key trigger repartitioning. What is the cost?

### The Formula

Repartition creates an internal topic with the same partition count as the source:

$$S_{repartition} = N_{records} \times (S_{key}' + S_{value})$$

Where $S_{key}'$ is the new key size.

Network transfer for repartition (each record sent to a potentially different partition):

$$\text{Network} = S_{repartition} \times \left(1 - \frac{1}{P}\right)$$

Where $P$ = partition count. On average, $(P-1)/P$ of records go to a different partition.

End-to-end latency impact:

$$L_{repartition} = L_{produce} + L_{commit} + L_{consume}$$

Typically 10-100ms depending on configuration.

### Worked Examples

| Records/s | Record Size | Partitions | Network/s | Latency Added |
|:---:|:---:|:---:|:---:|:---:|
| 100K | 500 B | 12 | ~46 MB/s | ~20 ms |
| 1M | 200 B | 32 | ~194 MB/s | ~30 ms |
| 10K | 1 KB | 8 | ~8.75 MB/s | ~15 ms |

---

## 6. Cache Deduplication (Optimization)

### The Problem

The record cache deduplicates updates to the same key before flushing to the state store and downstream. How effective is it?

### The Formula

Deduplication ratio (fraction of updates eliminated):

$$D = 1 - \frac{K_{distinct}}{N_{updates}}$$

Where $K_{distinct}$ = distinct keys in the cache window and $N_{updates}$ = total updates.

Effective downstream rate:

$$R_{downstream} = R_{input} \times (1 - D) = R_{input} \times \frac{K_{distinct}}{N_{updates}}$$

Cache hit probability (uniform key distribution):

$$P(\text{cache hit}) = 1 - \left(1 - \frac{1}{K}\right)^{C/S_{entry}}$$

Where $C$ = cache size in bytes, $S_{entry}$ = average entry size.

### Worked Examples

| Input Rate | Distinct Keys | Cache Size | Dedup Ratio | Output Rate |
|:---:|:---:|:---:|:---:|:---:|
| 100K/s | 10K | 10 MB | 90% | 10K/s |
| 100K/s | 90K | 10 MB | 10% | 90K/s |
| 50K/s | 1K | 5 MB | 98% | 1K/s |

---

## 7. Summary of Formulas

| Formula | Type | Domain |
|---------|------|--------|
| $\text{KTable}[k] = \bigoplus e.value$ | Stream-table duality | Algebraic foundation |
| $E[\|R\|] = \|S_1\|\|S_2\| \times 2w/(KT)$ | Join cardinality | Relational algebra |
| $S = K(S_k + S_v) \times A / n$ | State store size | Storage planning |
| $\text{Overhead} = T_{txn}/T_{commit}$ | EOS cost | Transaction theory |
| $\text{Network} = S \times (1 - 1/P)$ | Repartition cost | Network analysis |
| $D = 1 - K_{distinct}/N$ | Cache dedup ratio | Optimization |

## Prerequisites

- kafka, relational-algebra, probability, information-theory, distributed-systems

## Complexity

| Operation | Time | Space |
|-----------|------|-------|
| Map/Filter (stateless) | O(1) per record | O(1) |
| GroupByKey + Count | O(1) amortized per record | O(K) keys in store |
| Windowed aggregation | O(1) per record | O(K * W) windows |
| Stream-Stream join | O(W) window scan | O(rate * W) per side |
| Stream-Table join | O(1) lookup | O(K) table state |
| Repartition | O(1) + network latency | O(N) Kafka topic |
