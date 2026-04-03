# The Mathematics of Kafka — Distributed Streaming Internals

> *Apache Kafka is a distributed commit log. The math covers partition throughput, consumer group rebalancing, exactly-once semantics, storage sizing, and replication latency.*

---

## 1. Partition Throughput — The Parallelism Unit

### The Model

Kafka's parallelism is determined by partition count. Each partition is an ordered, immutable sequence of records.

### Throughput Formula

$$\text{Cluster Throughput} = \text{Partitions} \times \text{Per-Partition Throughput}$$

$$\text{Per-Partition Throughput} = \min(\text{Producer Rate}, \text{Broker Disk I/O}, \text{Consumer Rate})$$

### Partition Count Sizing

$$\text{Partitions} = \max\left(\frac{T_{target}}{T_{partition}}, C_{max}\right)$$

Where:
- $T_{target}$ = target throughput
- $T_{partition}$ = throughput per partition (~10 MB/s producer, ~30 MB/s consumer per partition typical)
- $C_{max}$ = maximum consumer instances (partitions >= consumers)

### Worked Examples

| Target Throughput | Per-Partition | Partitions Needed | With Consumers |
|:---:|:---:|:---:|:---:|
| 50 MB/s | 10 MB/s | 5 | max(5, consumers) |
| 500 MB/s | 10 MB/s | 50 | max(50, consumers) |
| 2 GB/s | 10 MB/s | 200 | max(200, consumers) |

### Partition Overhead

Each partition has a cost on the broker:

$$\text{Memory per Partition} \approx \text{index cache} + \text{log segment metadata} \approx 10 \text{ KiB}$$

$$\text{File Descriptors per Partition} = 2 \times \text{Segment Count (index + log)}$$

| Partitions | Memory Overhead | File Descriptors (10 segments each) |
|:---:|:---:|:---:|
| 100 | 1 MiB | 2,000 |
| 1,000 | 10 MiB | 20,000 |
| 10,000 | 100 MiB | 200,000 |
| 100,000 | 1 GiB | 2,000,000 |

---

## 2. Consumer Group Rebalancing

### Assignment Strategies

**Range Assignment:**

$$\text{Partitions per Consumer} = \lfloor \frac{P}{C} \rfloor$$

$$\text{Consumers with Extra} = P \mod C$$

**Round-Robin Assignment:**

$$\text{Consumer for Partition } i = i \mod C$$

### Worked Example

*"12 partitions, 5 consumers."*

**Range:** $\lfloor 12/5 \rfloor = 2$ base, $12 \mod 5 = 2$ extra.

| Consumer | Range Assignment | Round-Robin |
|:---:|:---|:---|
| C0 | P0, P1, P2 (3) | P0, P5, P10 (3) |
| C1 | P3, P4, P5 (3) | P1, P6, P11 (3) |
| C2 | P6, P7 (2) | P2, P7 (2) |
| C3 | P8, P9 (2) | P3, P8 (2) |
| C4 | P10, P11 (2) | P4, P9 (2) |

### Rebalance Downtime

$$T_{rebalance} = T_{revoke} + T_{assign} + T_{rejoin}$$

$$T_{revoke} \approx \text{session.timeout.ms (default 10s)} + \text{processing time}$$

### Cooperative Rebalancing

With incremental cooperative protocol:

$$\text{Partitions Disrupted} = \text{Only those changing assignment}$$

vs eager (stop-the-world):

$$\text{Partitions Disrupted} = \text{All partitions}$$

---

## 3. Exactly-Once Semantics (EOS)

### Idempotent Producer

Each producer gets a Producer ID (PID) and sequence number per partition:

$$\text{Dedup Key} = (\text{PID}, \text{Partition}, \text{Sequence Number})$$

$$\text{Sequence Numbers:} \quad 0, 1, 2, \ldots \quad (\text{monotonically increasing per partition})$$

Broker rejects duplicates if:

$$\text{Incoming Seq} \leq \text{Last Committed Seq for (PID, Partition)}$$

### Transactional Writes

Atomic writes across multiple partitions:

$$\text{Transaction} = \{(\text{Topic}_1, \text{Partition}_a, \text{Record}), (\text{Topic}_2, \text{Partition}_b, \text{Record}), \ldots\}$$

$$\text{Transaction Overhead} = \text{Transaction Marker Records (2 per partition)} + \text{Coordinator RPC}$$

### Transaction Throughput Impact

$$\text{TPS}_{transactional} \approx \frac{\text{TPS}_{non-txn}}{1 + \frac{T_{commit}}{T_{batch}}}$$

| Batch Interval | Commit Overhead | Throughput vs Non-Txn |
|:---:|:---:|:---:|
| 100 ms | 10 ms | 91% |
| 50 ms | 10 ms | 83% |
| 10 ms | 10 ms | 50% |
| 1 ms | 10 ms | 9% |

---

## 4. Storage Sizing and Retention

### Storage Formula

$$\text{Storage} = \text{Throughput} \times \text{Retention Period} \times \text{Replication Factor}$$

### Worked Examples

| Throughput | Retention | Replication | Storage Needed |
|:---:|:---:|:---:|:---:|
| 10 MB/s | 7 days | 3 | 18.1 TiB |
| 100 MB/s | 3 days | 3 | 77.8 TiB |
| 1 GB/s | 1 day | 3 | 259.2 TiB |
| 10 MB/s | 30 days | 3 | 77.8 TiB |

### Log Segment Sizing

$$\text{Segment Roll Time} = \min\left(\frac{\text{segment.bytes}}{\text{Write Rate}}, \text{segment.ms}\right)$$

Default: `segment.bytes = 1 GiB`, `segment.ms = 7 days`.

| Write Rate | Time to Fill 1 GiB Segment |
|:---:|:---:|
| 1 MB/s | 17 minutes |
| 10 MB/s | 102 seconds |
| 100 MB/s | 10 seconds |

### Compacted Topic Size

$$\text{Compacted Size} = \text{Unique Keys} \times \text{Avg Record Size}$$

$$\text{Compaction Ratio} = \frac{\text{Unique Keys}}{\text{Total Records}}$$

---

## 5. Replication and Durability

### ISR (In-Sync Replicas)

$$\text{Durability} = \text{min.insync.replicas (acks=all)}$$

$$P(\text{data loss}) = P(\text{all ISR members fail before sync})$$

### Replication Lag

$$\text{Lag (bytes)} = \text{Leader Log End Offset} - \text{Follower Log End Offset}$$

$$\text{Lag (time)} = \frac{\text{Lag (bytes)}}{\text{Produce Rate}}$$

### Replication Bandwidth

$$\text{Replication BW} = \text{Produce Rate} \times (\text{Replication Factor} - 1)$$

| Produce Rate | RF | Replication BW | Total Disk Write |
|:---:|:---:|:---:|:---:|
| 100 MB/s | 2 | 100 MB/s | 200 MB/s |
| 100 MB/s | 3 | 200 MB/s | 300 MB/s |
| 500 MB/s | 3 | 1 GB/s | 1.5 GB/s |

---

## 6. Producer Batching and Compression

### Batch Efficiency

$$\text{Effective Throughput} = \frac{\text{Batch Size}}{\text{Linger Time} + T_{network} + T_{broker}}$$

### Compression Ratios

$$\text{Network Savings} = 1 - \frac{1}{\text{Compression Ratio}}$$

| Codec | Typical Ratio (JSON) | CPU Cost | Network Savings |
|:---|:---:|:---:|:---:|
| None | 1.0x | None | 0% |
| Snappy | 2-3x | Very low | 50-67% |
| LZ4 | 2-4x | Low | 50-75% |
| GZIP | 4-8x | High | 75-88% |
| ZSTD | 4-8x | Medium | 75-88% |

### End-to-End Latency

$$T_{e2e} = T_{linger} + T_{batch\_fill} + T_{compress} + T_{network} + T_{broker\_write} + T_{replication} + T_{consumer\_poll}$$

| Component | Typical | Optimized |
|:---|:---:|:---:|
| Linger | 0-100 ms | 0-5 ms |
| Network | 0.1-1 ms | 0.1 ms |
| Broker write | 1-5 ms | 0.5-1 ms |
| Replication | 1-10 ms | 1-3 ms |
| Consumer poll | 0-500 ms | 0-100 ms |
| **Total** | **2-616 ms** | **1.6-109 ms** |

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| $\text{Partitions} \times T_{per}$ | Linear scaling | Cluster throughput |
| $\lfloor P/C \rfloor$ | Integer division | Consumer assignment |
| $(\text{PID}, \text{Partition}, \text{Seq})$ | Tuple dedup | Exactly-once |
| $T \times R \times \text{RF}$ | Triple product | Storage sizing |
| $\frac{\text{Batch}}{\text{Linger} + T}$ | Rate equation | Producer throughput |
| $1 - \frac{1}{\text{Ratio}}$ | Fraction | Compression savings |

---

*Every `kafka-topics --describe`, `kafka-consumer-groups --describe`, and broker metric reflects these internals — a distributed commit log where partition count is the fundamental unit of parallelism.*
