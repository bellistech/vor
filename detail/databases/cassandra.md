# The Mathematics of Cassandra — Consistent Hashing, Quorum Arithmetic, and Compaction

> *Cassandra distributes data across a hash ring using consistent hashing and achieves tunable consistency through quorum arithmetic. The math covers token distribution, replication placement, read/write consistency proofs, compaction strategies, and tombstone lifecycle.*

---

## 1. Consistent Hashing Ring (Hash Theory)

### Token Space

Cassandra uses Murmur3 hash with a 64-bit token range:

$$T \in [-2^{63}, 2^{63} - 1]$$

$$|T| = 2^{64}$$

### Token Assignment (vnodes)

Each node owns $v$ virtual nodes (vnodes). With $N$ physical nodes:

$$\text{Total vnodes} = N \times v$$

$$\text{Expected tokens per node} = \frac{|T|}{N}$$

### Load Balance

Standard deviation of data per node:

$$\sigma = \frac{|T|}{N} \times \frac{1}{\sqrt{v}}$$

| Nodes | vnodes/node | Total vnodes | Load Std Dev (%) |
|:---:|:---:|:---:|:---:|
| 6 | 1 | 6 | 40.8% |
| 6 | 16 | 96 | 10.2% |
| 6 | 256 | 1,536 | 2.6% |
| 12 | 256 | 3,072 | 1.8% |

### Partition Placement

$$\text{node}(\text{partition\_key}) = \text{first node clockwise from } \text{murmur3}(\text{key})$$

With replication factor $RF$, data is placed on $RF$ distinct nodes clockwise on the ring.

---

## 2. Consistency Level Arithmetic (Distributed Systems)

### Read-Write Overlap

Strong consistency requires:

$$R + W > RF$$

where $R$ = read consistency level, $W$ = write consistency level, $RF$ = replication factor.

### Common Configurations

| Write CL | Read CL | RF | $R + W$ | Consistent? |
|:---:|:---:|:---:|:---:|:---:|
| ONE (1) | ONE (1) | 3 | 2 | No |
| QUORUM (2) | QUORUM (2) | 3 | 4 | Yes |
| ALL (3) | ONE (1) | 3 | 4 | Yes |
| ONE (1) | ALL (3) | 3 | 4 | Yes |
| LOCAL_QUORUM (2) | LOCAL_QUORUM (2) | 3 | 4 | Yes (local DC) |

### Quorum Formula

$$Q = \left\lfloor \frac{RF}{2} \right\rfloor + 1$$

| Replication Factor | Quorum | Nodes Tolerated Down |
|:---:|:---:|:---:|
| 1 | 1 | 0 |
| 3 | 2 | 1 |
| 5 | 3 | 2 |
| 7 | 4 | 3 |

### Availability Under Node Failures

With $f$ failed nodes out of $RF$ replicas, the probability a read/write succeeds at CL=$c$:

$$P(\text{success}) = \begin{cases} 1 & \text{if } RF - f \geq c \\ 0 & \text{if } RF - f < c \end{cases}$$

| RF | CL | 1 Node Down | 2 Nodes Down |
|:---:|:---:|:---:|:---:|
| 3 | ONE | Available | Available |
| 3 | QUORUM | Available | Unavailable |
| 3 | ALL | Unavailable | Unavailable |
| 5 | QUORUM | Available | Available |

---

## 3. Partition Sizing (Data Modeling)

### Partition Size

$$S_{\text{partition}} = N_{\text{rows}} \times \left(\sum_{i=1}^{C} S_{\text{col}_i} + O_{\text{row}}\right)$$

where $O_{\text{row}}$ = per-row overhead (~23 bytes).

### Maximum Partition Guidelines

| Max Rows | Avg Row Size | Partition Size | Recommendation |
|:---:|:---:|:---:|:---|
| 1,000 | 200 B | 200 KB | Ideal |
| 10,000 | 500 B | 5 MB | Good |
| 100,000 | 500 B | 50 MB | Acceptable |
| 1,000,000 | 500 B | 500 MB | Too large |

### Time Bucketing

For time-series data at rate $\lambda$ events/second:

$$N_{\text{rows/bucket}} = \lambda \times T_{\text{bucket}}$$

| Event Rate | Bucket (1 day) | Bucket (1 hour) | Bucket (1 min) |
|:---:|:---:|:---:|:---:|
| 1/s | 86,400 | 3,600 | 60 |
| 100/s | 8,640,000 | 360,000 | 6,000 |
| 10,000/s | 864M | 36M | 600,000 |

Choose bucket size to keep partitions under 100K rows.

---

## 4. Compaction Strategies (Storage)

### Size-Tiered Compaction (STCS)

Merges SSTables of similar size:

$$W_{\text{amp}} = O(T \times L)$$

$$\text{Temporary space} = S_{\text{data}} \quad \text{(worst case 2x)}$$

where $T$ = min threshold (default 4), $L$ = number of tiers.

### Leveled Compaction (LCS)

Fixed-size SSTables across levels:

$$W_{\text{amp}} = O(L \times T) \approx 10 \times \log_{10}(N)$$

$$\text{Read amp} = O(1) \quad \text{(at most 1 SSTable per level)}$$

### Time-Window Compaction (TWCS)

Groups SSTables by time window:

$$W_{\text{amp}} = O(T) \quad \text{(only within window)}$$

$$\text{Ideal for TTL data: no rewrite of old windows}$$

### Strategy Comparison

| Strategy | Write Amp | Read Amp | Space Amp | Best For |
|:---|:---:|:---:|:---:|:---|
| STCS | Low | High | 2x | Write-heavy |
| LCS | High (10x) | Low (1) | 1.1x | Read-heavy |
| TWCS | Very Low | Medium | 1.1x | Time-series + TTL |

---

## 5. Tombstone Lifecycle (Garbage Collection)

### Tombstone Creation

Each delete creates a tombstone marker. Tombstones persist until:

$$T_{\text{gc}} = T_{\text{delete}} + \text{gc\_grace\_seconds}$$

### Tombstone Overhead

$$\text{Tombstone size} \approx K_{\text{size}} + 16 \text{ bytes (metadata)}$$

$$\text{Read cost with tombstones} = O(n_{\text{live}} + n_{\text{tombstone}})$$

### Read Amplification from Tombstones

| Live Rows | Tombstones | Read Ratio | Warning Threshold |
|:---:|:---:|:---:|:---:|
| 100 | 10 | 1.1x | OK |
| 100 | 1,000 | 11x | Warning |
| 100 | 10,000 | 101x | Critical |
| 100 | 100,000 | 1,001x | Query failure |

Default `tombstone_warn_threshold` = 1,000, `tombstone_failure_threshold` = 100,000.

### Safe gc_grace_seconds

$$\text{gc\_grace} > \max(\text{repair\_interval}, \text{max\_hint\_window})$$

Default: `gc_grace_seconds = 864000` (10 days), requiring repair at least every 10 days.

---

## 6. Read/Write Path Latency (Performance)

### Write Path

$$T_{\text{write}} = T_{\text{commit\_log}} + T_{\text{memtable}}$$

$$T_{\text{commit\_log}} = T_{\text{fsync}} \quad (\text{sequential write})$$

$$T_{\text{memtable}} = O(\log n) \quad (\text{skiplist insert})$$

### Read Path

$$T_{\text{read}} = T_{\text{bloom}} + T_{\text{index}} + T_{\text{compression}} + T_{\text{disk}}$$

Bloom filter eliminates SSTables not containing the key:

$$P(\text{false positive}) = \left(1 - e^{-kn/m}\right)^k$$

With default 10 bits/element and 7 hash functions: $P \approx 0.82\%$.

| SSTables | Bloom FP Rate | Expected Disk Reads |
|:---:|:---:|:---:|
| 5 | 1% | 1.05 |
| 10 | 1% | 1.1 |
| 50 | 1% | 1.5 |
| 100 | 1% | 2.0 |

---

## Prerequisites

distributed-systems, hash-functions, data-structures, probability

## Complexity

| Operation | Time | Space |
|:---|:---:|:---:|
| Write (commit log + memtable) | O(log n) | O(row) |
| Point read (partition key) | O(log n) per SSTable | O(row) |
| Range scan (within partition) | O(log n + k) | O(k) rows |
| Full table scan | O(N) all partitions | O(1) streaming |
| Compaction (STCS) | O(n) merge | O(n) temp |
| Compaction (LCS) | O(n) per level | O(SSTable) |
| Repair (Merkle tree) | O(n) per range | O(tree) |
| Bloom filter lookup | O(k) hashes | O(m) bits |
