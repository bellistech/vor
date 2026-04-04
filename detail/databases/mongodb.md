# The Mathematics of MongoDB — B-Tree Indexes, Sharding, and Aggregation Cost

> *MongoDB stores BSON documents with B-tree indexes, distributes data via consistent hashing or range partitioning, and processes queries through a cost-based optimizer. The math covers index selectivity, shard distribution, replication lag, and aggregation pipeline complexity.*

---

## 1. B-Tree Index Performance (Data Structures)

### B-Tree Lookup

MongoDB uses B+ trees (WiredTiger) with branching factor $B$:

$$\text{Height} = \lceil \log_B(n) \rceil$$

$$\text{Disk reads per lookup} = h + 1 \quad \text{(traverse + leaf)}$$

### Index Size

$$\text{Index Size} = n \times (K_{\text{avg}} + P + O)$$

where $K_{\text{avg}}$ = average key size, $P$ = pointer size (8 bytes), $O$ = overhead (~16 bytes).

| Documents | Avg Key Size | Branching | Height | Index Size |
|:---:|:---:|:---:|:---:|:---:|
| 1,000,000 | 20 B | 200 | 3 | ~44 MB |
| 10,000,000 | 20 B | 200 | 4 | ~440 MB |
| 100,000,000 | 50 B | 100 | 5 | ~7.4 GB |
| 1,000,000,000 | 20 B | 200 | 5 | ~44 GB |

### Compound Index Selectivity

For a compound index on fields $(a, b, c)$:

$$\text{Selectivity} = \frac{1}{|V_a|} \times \frac{1}{|V_b|} \times \frac{1}{|V_c|}$$

$$\text{Documents scanned} = n \times \text{Selectivity}$$

| Field Values | Selectivity | Docs Scanned (1M total) |
|:---:|:---:|:---:|
| a=100 | 1/100 = 0.01 | 10,000 |
| a=100, b=50 | 1/5000 | 200 |
| a=100, b=50, c=10 | 1/50000 | 20 |

---

## 2. Sharding Distribution (Hash Theory)

### Hashed Shard Key

$$\text{shard}(\text{doc}) = \text{hash}(\text{shard\_key}) \mod C$$

where $C$ = number of chunks. Chunks are assigned to shards.

### Chunk Distribution

With $C$ chunks and $S$ shards, ideal distribution:

$$\text{Chunks per shard} = \frac{C}{S}$$

Standard deviation (random assignment):

$$\sigma = \sqrt{\frac{C \times (S-1)}{S^2}} \approx \sqrt{\frac{C}{S}}$$

| Chunks | Shards | Ideal per Shard | Std Dev |
|:---:|:---:|:---:|:---:|
| 100 | 3 | 33.3 | 5.4 |
| 1,000 | 5 | 200 | 12.6 |
| 10,000 | 10 | 1,000 | 30.0 |

### Range Shard Key Hot Spots

With monotonically increasing keys (timestamps), all writes go to one shard:

$$\text{Write throughput} = \frac{\mu_{\text{single\_shard}}}{1} \quad \text{(no parallelism)}$$

vs hashed keys:

$$\text{Write throughput} = S \times \mu_{\text{single\_shard}}$$

---

## 3. WiredTiger Cache and Compression (Memory)

### Working Set in Cache

$$\text{Cache hit rate} = \frac{W_{\text{hot}}}{W_{\text{total}}}$$

where $W_{\text{hot}}$ = frequently accessed data, $W_{\text{total}}$ = total data.

### Recommended Cache Size

$$\text{WiredTiger cache} = \max(256 \text{ MB}, 0.5 \times (\text{RAM} - 1 \text{ GB}))$$

### Compression Ratios

| Compression | Ratio | CPU Cost | Use Case |
|:---|:---:|:---:|:---|
| snappy (default) | 2:1 - 4:1 | Low | General workloads |
| zstd | 3:1 - 8:1 | Medium | High compression needs |
| zlib | 3:1 - 6:1 | High | Maximum compression |
| none | 1:1 | None | Low-latency reads |

$$\text{Disk usage} = \frac{N \times D_{\text{avg}}}{R_{\text{compression}}} + I_{\text{total}}$$

---

## 4. Replication Lag (Distributed Systems)

### Oplog Model

$$\text{Lag} = T_{\text{apply}} - T_{\text{primary}}$$

$$T_{\text{apply}} = T_{\text{network}} + T_{\text{write}}$$

### Oplog Window

$$T_{\text{window}} = \frac{S_{\text{oplog}}}{\lambda_{\text{writes}} \times B_{\text{avg\_op}}}$$

| Oplog Size | Write Rate | Avg Op Size | Window |
|:---:|:---:|:---:|:---:|
| 1 GB | 100 ops/s | 500 B | 5.8 hours |
| 5 GB | 100 ops/s | 500 B | 29 hours |
| 5 GB | 1,000 ops/s | 1 KB | 1.4 hours |
| 50 GB | 10,000 ops/s | 500 B | 2.9 hours |

### Write Concern Latency

$$T_{w} = \begin{cases} T_{\text{local}} & w=1 \\ \max(T_{\text{node}_1}, \ldots, T_{\text{node}_{w-1}}) & w > 1 \\ \max(T_{\text{node}_1}, \ldots, T_{\text{node}_{\lceil n/2 \rceil}}) & w=\text{"majority"} \end{cases}$$

---

## 5. Aggregation Pipeline Complexity (Query Processing)

### Stage Costs

| Stage | Time | Space | Notes |
|:---|:---:|:---:|:---|
| $match (indexed) | O(log n + k) | O(k) | Uses index |
| $match (scan) | O(n) | O(1) | Full collection scan |
| $sort (indexed) | O(k) | O(1) | Covered by index |
| $sort (in-memory) | O(k log k) | O(k) | 100 MB RAM limit |
| $group | O(n) | O(g) groups | Hash aggregation |
| $lookup | O(n * m) | O(n * j) | Nested loop join |
| $unwind | O(n * a) | O(n * a) | a = avg array length |
| $project | O(n) | O(1) | Per-document transform |

### $lookup Join Cost

$$T_{\text{lookup}} = n \times (T_{\text{index\_lookup}} + j \times T_{\text{fetch}})$$

where $j$ = average join matches per document.

With index on foreign field:

$$T_{\text{lookup}} = n \times O(\log m) \times j$$

Without index:

$$T_{\text{lookup}} = n \times O(m)$$

---

## 6. Document Size and BSON Encoding (Storage)

### BSON Overhead

$$\text{BSON size} = 4 + \sum_{i=1}^{k} (1 + |\text{key}_i| + 1 + |\text{value}_i|) + 1$$

where 4 = document size header, 1 = type byte, 1 = null terminator per key.

### Type Sizes

| BSON Type | Size | Notes |
|:---|:---:|:---|
| int32 | 4 bytes | |
| int64 / double | 8 bytes | |
| ObjectId | 12 bytes | |
| boolean | 1 byte | |
| date | 8 bytes | |
| string | 4 + len + 1 | length prefix + null |
| embedded doc | recursive | |

### Document Count per Page

$$\text{Docs per 4KB page} = \left\lfloor \frac{4096}{D_{\text{avg}}} \right\rfloor$$

| Avg Doc Size | Docs/Page | Docs in 1 GB Cache |
|:---:|:---:|:---:|
| 200 B | 20 | 5,368,709 |
| 1 KB | 4 | 1,048,576 |
| 4 KB | 1 | 262,144 |
| 16 KB | 0.25 | 65,536 |

---

## Prerequisites

data-structures, hash-functions, distributed-systems, sql

## Complexity

| Operation | Time | Space |
|:---|:---:|:---:|
| Find by _id | O(log n) B-tree | O(1) |
| Find by indexed field | O(log n + k) | O(k) results |
| Find (full scan) | O(n) | O(1) |
| Insert | O(log n) per index | O(d) document |
| Update by _id | O(log n) | O(d) |
| Aggregate ($group) | O(n) | O(g) groups |
| Aggregate ($sort) | O(n log n) | O(n) or disk |
| Shard scatter-gather | O(S * log n) | O(S * k) |
