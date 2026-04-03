# The Mathematics of Redis — Data Structure and Memory Internals

> *Redis is an in-memory data structure server. The math covers data structure complexity, jemalloc memory overhead, eviction policies, replication bandwidth, and persistence trade-offs.*

---

## 1. Data Structure Complexity

### Core Structures and Big-O

| Structure | Implementation | Insert | Lookup | Delete | Range |
|:---|:---|:---:|:---:|:---:|:---:|
| String | SDS (Simple Dynamic String) | O(1) | O(1) | O(1) | N/A |
| List | Quicklist (ziplist + linked list) | O(1) ends | O(n) | O(n) | O(S+n) |
| Set | Hashtable or intset | O(1) | O(1) | O(1) | O(n) |
| Sorted Set | Skiplist + hashtable | O(log n) | O(log n) | O(log n) | O(log n + m) |
| Hash | Hashtable or ziplist | O(1) | O(1) | O(1) | O(n) |
| Stream | Radix tree + listpacks | O(1) | O(log n) | N/A | O(n) |

### Skiplist — Sorted Set Internals

The skiplist has $\log_2(n)$ expected levels. Each node has ~1.33 pointers on average (p=0.25):

$$E[\text{levels}] = \frac{1}{1 - p} = \frac{1}{0.75} = 1.33 \text{ pointers per node}$$

$$E[\text{max level}] = \log_{1/p}(n) = \log_4(n)$$

| Elements | Expected Max Level | Search Comparisons |
|:---:|:---:|:---:|
| 1,000 | 5 | ~10 |
| 1,000,000 | 10 | ~20 |
| 1,000,000,000 | 15 | ~30 |

### Ziplist — Compact Encoding

For small hashes/sorted sets (< `hash-max-ziplist-entries` = 128):

$$\text{Ziplist Lookup} = O(n) \quad (\text{linear scan})$$

$$\text{Ziplist Memory} = 11 + \sum_{i=1}^{n} (\text{prevlen}_i + \text{encoding}_i + \text{data}_i)$$

The crossover point where hashtable becomes faster than ziplist:

$$n_{crossover} \approx 128 \text{ entries (default threshold)}$$

---

## 2. Memory Overhead — jemalloc Allocation Classes

### The Model

Redis uses jemalloc, which allocates in fixed **size classes**. Every allocation is rounded up to the next class.

### jemalloc Size Classes (selected)

| Requested | Allocated | Waste |
|:---:|:---:|:---:|
| 1-8 bytes | 8 bytes | 0-7 bytes |
| 9-16 bytes | 16 bytes | 0-7 bytes |
| 17-32 bytes | 32 bytes | 0-15 bytes |
| 33-48 bytes | 48 bytes | 0-15 bytes |
| 49-64 bytes | 64 bytes | 0-15 bytes |
| 65-80 bytes | 80 bytes | 0-15 bytes |
| 81-96 bytes | 96 bytes | 0-15 bytes |
| 97-128 bytes | 128 bytes | 0-31 bytes |

### Redis Object Overhead

Every Redis value is wrapped in a `robj` (redisObject):

$$\text{redisObject} = 16 \text{ bytes}$$

| Field | Bytes | Purpose |
|:---|:---:|:---|
| type | 4 bits | String, List, Set, ZSet, Hash |
| encoding | 4 bits | Raw, Int, Ziplist, Skiplist, etc. |
| lru | 24 bits | LRU clock or LFU counters |
| refcount | 4 bytes | Reference count |
| ptr | 8 bytes | Pointer to actual data |

### SDS (Simple Dynamic String) Overhead

$$\text{SDS Overhead} = \text{header (3-17 bytes)} + \text{string data} + 1 \text{ (null terminator)}$$

| SDS Type | Header Size | Max Length |
|:---|:---:|:---:|
| sdshdr5 | 1 byte | 31 bytes |
| sdshdr8 | 3 bytes | 255 bytes |
| sdshdr16 | 5 bytes | 65,535 bytes |
| sdshdr32 | 9 bytes | 4 GiB |
| sdshdr64 | 17 bytes | 2^63 bytes |

### Total Memory per Key-Value Pair

$$\text{Memory per KV} = \text{dictEntry (24 bytes)} + \text{Key robj (16)} + \text{Key SDS} + \text{Value robj (16)} + \text{Value Data} + \text{jemalloc rounding}$$

### Worked Example

*"10 million keys, each key = 20-byte string, value = 100-byte string."*

| Component | Raw Size | jemalloc Class | Allocated |
|:---|:---:|:---:|:---:|
| dictEntry | 24 bytes | 32 bytes | 32 |
| Key robj | 16 bytes | 16 bytes | 16 |
| Key SDS (20 + 3 + 1) | 24 bytes | 32 bytes | 32 |
| Value robj | 16 bytes | 16 bytes | 16 |
| Value SDS (100 + 3 + 1) | 104 bytes | 128 bytes | 128 |
| **Per KV total** | | | **224 bytes** |

$$\text{Total} = 10,000,000 \times 224 = 2.09 \text{ GiB}$$

**Actual data:** 10M × 120 bytes = 1.12 GiB. **Overhead: 87%.**

---

## 3. Eviction Policies — LRU and LFU

### LRU Sampling

Redis doesn't implement true LRU (which requires a doubly-linked list of all keys). Instead, it samples $k$ random keys and evicts the least recently used:

$$P(\text{evict optimal}) = 1 - \left(1 - \frac{1}{N}\right)^k$$

Default sample size: `maxmemory-samples = 5`.

| Sample Size | Accuracy vs True LRU | CPU Cost |
|:---:|:---:|:---:|
| 1 | ~60% | Lowest |
| 5 (default) | ~90% | Low |
| 10 | ~95% | Medium |
| 50 | ~99% | High |

### LFU Decay

LFU (Least Frequently Used) uses a logarithmic frequency counter with time decay:

$$\text{counter} = \min(255, \text{counter} + \frac{1}{\text{counter} \times \text{lfu-log-factor} + 1})$$

$$\text{Decay: counter} = \max(0, \text{counter} - \frac{\text{minutes elapsed}}{\text{lfu-decay-time}})$$

Default: `lfu-log-factor = 10`, `lfu-decay-time = 1`.

| Access Count | Counter Value (factor=10) |
|:---:|:---:|
| 1 | 1 |
| 10 | 8 |
| 100 | 18 |
| 1,000 | 131 |
| 10,000 | 223 |
| 100,000 | 250 |
| 1,000,000 | 255 (max) |

### Eviction Policy Comparison

| Policy | Evicts | Best For |
|:---|:---|:---|
| volatile-lru | Least recent with TTL | Cache with expiry |
| allkeys-lru | Least recent any key | General cache |
| volatile-lfu | Least frequent with TTL | Frequency-biased cache |
| allkeys-lfu | Least frequent any key | Hot/cold workloads |
| volatile-random | Random with TTL | Unknown access patterns |
| allkeys-random | Random any key | Uniform access |
| volatile-ttl | Shortest TTL | Expire-first strategy |
| noeviction | Error on full | Data store (not cache) |

---

## 4. Persistence — RDB and AOF Trade-offs

### RDB (Snapshotting)

$$T_{rdb} = \frac{\text{Memory Used}}{\text{Disk Write Speed}}$$

$$\text{Fork Memory} = \text{Copy-on-Write pages} \times 4 \text{ KiB}$$

$$\text{COW Pages} \approx \text{Write Rate} \times T_{rdb} \times \frac{1}{\text{Page Size}}$$

| Memory | Disk Speed | RDB Time | COW at 1% write/sec |
|:---:|:---:|:---:|:---:|
| 1 GiB | 500 MB/s | 2 sec | ~10 MiB |
| 10 GiB | 500 MB/s | 20 sec | ~100 MiB |
| 50 GiB | 500 MB/s | 100 sec | ~500 MiB |
| 100 GiB | 1 GB/s | 100 sec | ~1 GiB |

### AOF (Append-Only File)

$$\text{AOF Size} \geq \text{Sum of all write commands} \times \text{Protocol Overhead}$$

$$\text{Protocol Overhead} \approx 2-3\times \text{ (RESP encoding of commands)}$$

| AOF fsync Policy | Data Loss Window | Write Latency |
|:---|:---:|:---:|
| always | 0 | +0.5-2 ms per write |
| everysec (default) | 1 second | +0 ms |
| no | OS-dependent | +0 ms |

### AOF Rewrite

$$\text{Rewrite Size} \approx \text{Memory Used} \times 1.5 \quad (\text{RESP command encoding})$$

$$\text{Compaction Ratio} = \frac{\text{Old AOF Size}}{\text{Rewritten AOF Size}}$$

---

## 5. Replication Bandwidth

### Full Sync (RDB Transfer)

$$T_{full\_sync} = T_{rdb\_generate} + \frac{\text{RDB Size}}{\text{Network BW}}$$

### Partial Resync (Replication Backlog)

$$\text{Backlog Required} = \text{Write Rate} \times \text{Max Disconnect Time}$$

Default backlog: 1 MiB. For a replica that disconnects for 60 seconds:

$$\text{Backlog Needed} = \text{Write Rate (MB/s)} \times 60$$

| Write Rate | 10 sec disconnect | 60 sec | 300 sec |
|:---:|:---:|:---:|:---:|
| 1 MB/s | 10 MiB | 60 MiB | 300 MiB |
| 10 MB/s | 100 MiB | 600 MiB | 3 GiB |
| 100 MB/s | 1 GiB | 6 GiB | 30 GiB |

---

## 6. Pipeline and Throughput Math

### Round-Trip Overhead

$$\text{Ops/sec (no pipeline)} = \frac{1}{\text{RTT} + T_{command}}$$

$$\text{Ops/sec (pipeline of } k\text{)} = \frac{k}{\text{RTT} + k \times T_{command}}$$

| RTT | No Pipeline | Pipeline 10 | Pipeline 100 |
|:---:|:---:|:---:|:---:|
| 0.1 ms (local) | 10,000 | 91,000 | 500,000 |
| 1 ms (LAN) | 1,000 | 9,100 | 91,000 |
| 10 ms (WAN) | 100 | 990 | 9,900 |

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| O(log n) skiplist | Logarithmic | Sorted set operations |
| $16 + \text{SDS} + \text{jemalloc}$ | Additive overhead | Memory per object |
| $1 - (1-1/N)^k$ | Probability | LRU sampling accuracy |
| $\frac{k}{\text{RTT} + kT}$ | Throughput | Pipeline performance |
| $\text{Memory} / \text{Disk BW}$ | Rate equation | RDB snapshot time |
| $\text{Rate} \times T_{disconnect}$ | Linear | Replication backlog |

---

*Every `INFO memory`, `OBJECT ENCODING`, and `MEMORY USAGE` command exposes these internals — a system where understanding memory allocation classes can save you 50% of your RAM bill.*
