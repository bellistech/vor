# The Mathematics of MySQL — InnoDB Storage Engine Internals

> *MySQL's InnoDB engine uses a B+ tree clustered index, buffer pool caching, redo/undo logs, and MVCC. The math covers page structure, buffer pool hit ratios, redo log sizing, and query optimization.*

---

## 1. InnoDB Page Structure

### The Model

InnoDB stores data in fixed 16 KiB **pages**. Every table is a clustered B+ tree indexed by primary key.

### Page Layout

$$\text{Page Size} = 16,384 \text{ bytes (default, configurable: 4K, 8K, 16K, 32K, 64K)}$$

| Component | Size | Purpose |
|:---|:---:|:---|
| FIL Header | 38 bytes | Page number, type, checksum |
| Page Header | 56 bytes | Record count, free space pointers |
| Infimum/Supremum | 26 bytes | Boundary records |
| User Records | Variable | Actual row data |
| Free Space | Variable | Unallocated space |
| Page Directory | Variable | Slot pointers (every 4-8 records) |
| FIL Trailer | 8 bytes | Checksum verification |

$$\text{Usable Space} = 16,384 - 38 - 56 - 26 - 8 = 16,256 \text{ bytes (minus directory slots)}$$

### Rows per Page

$$\text{Rows per Page} \approx \frac{16,256 \times \text{Fill Factor}}{\text{Row Size} + \text{Record Header (5 bytes)}}$$

Default fill factor = ~93% (15/16 for B+ tree splits).

| Row Size | With Header | Rows/Page | Pages per 1M Rows |
|:---:|:---:|:---:|:---:|
| 50 bytes | 55 bytes | 274 | 3,650 |
| 100 bytes | 105 bytes | 143 | 6,993 |
| 200 bytes | 205 bytes | 73 | 13,699 |
| 500 bytes | 505 bytes | 29 | 34,483 |
| 1000 bytes | 1005 bytes | 15 | 66,667 |

---

## 2. Buffer Pool — Cache Hit Ratio

### The Model

The buffer pool caches data and index pages in RAM. Hit ratio determines performance.

### Hit Ratio Formula

$$\text{Hit Ratio} = 1 - \frac{\text{Innodb\_buffer\_pool\_reads}}{\text{Innodb\_buffer\_pool\_read\_requests}}$$

### Effective Latency

$$T_{avg} = \text{Hit Ratio} \times T_{memory} + (1 - \text{Hit Ratio}) \times T_{disk}$$

Where $T_{memory} \approx 0.0001$ ms, $T_{disk} \approx 0.1-10$ ms.

| Hit Ratio | Avg Latency (SSD) | Avg Latency (HDD) | Relative Speed |
|:---:|:---:|:---:|:---:|
| 99.9% | 0.0001 ms | 0.01 ms | Baseline |
| 99.0% | 0.001 ms | 0.1 ms | 10x slower |
| 95.0% | 0.005 ms | 0.5 ms | 50x slower |
| 90.0% | 0.01 ms | 1.0 ms | 100x slower |
| 50.0% | 0.05 ms | 5.0 ms | 500x slower |

### Buffer Pool Sizing

$$\text{Recommended} = \min(0.80 \times \text{RAM}, \text{Total Data + Index Size})$$

$$\text{Working Set Ratio} = \frac{\text{Frequently Accessed Data}}{\text{Buffer Pool Size}}$$

If Working Set Ratio > 1.0, thrashing occurs.

---

## 3. Redo Log (WAL) Sizing

### The Model

InnoDB's redo log ensures durability. Too-small logs cause frequent checkpoints; too-large logs cause slow crash recovery.

### Checkpoint Interval

$$T_{checkpoint} = \frac{\text{Total Redo Log Size}}{\text{Redo Write Rate}}$$

$$\text{Crash Recovery Time} \approx \frac{\text{Redo Log Size}}{\text{Redo Apply Rate}}$$

### Recommended Redo Log Size

$$\text{Redo Log Size} = \text{Peak Write Rate} \times \text{Desired Checkpoint Interval}$$

| Write Rate | Checkpoint Interval | Redo Log Size |
|:---:|:---:|:---:|
| 10 MB/s | 30 min | 18 GiB |
| 50 MB/s | 30 min | 90 GiB |
| 100 MB/s | 15 min | 90 GiB |
| 100 MB/s | 60 min | 360 GiB |

### Redo Log Write Amplification

$$\text{Redo per Row Update} = \text{Row Data} + \text{Undo Record} + \text{Redo Header (25 bytes)}$$

For a 100-byte row update:

$$\text{Redo} \approx 100 + 100 + 25 = 225 \text{ bytes}$$

---

## 4. B+ Tree Depth and Index Optimization

### Clustered Index Depth

$$\text{Fan-out} = \frac{16,256}{\text{Key Size} + 6 \text{ (pointer)}}$$

$$\text{Depth} = \lceil \log_f(N) \rceil + 1 \quad (\text{+1 for leaf level})$$

| PK Size | Fan-out | 1M rows | 100M rows | 1B rows |
|:---:|:---:|:---:|:---:|:---:|
| 4 (INT) | 1,625 | 2 | 3 | 3 |
| 8 (BIGINT) | 1,161 | 2 | 3 | 3 |
| 16 (UUID binary) | 738 | 2 | 3 | 4 |
| 36 (UUID varchar) | 386 | 3 | 3 | 4 |

### Secondary Index Cost

Every secondary index lookup requires a **primary key lookup** (bookmark lookup):

$$\text{Total Reads} = \text{Index Depth} + \text{Clustered Index Depth} \quad (\text{per row})$$

$$\text{Covering Index Reads} = \text{Index Depth only (no bookmark)}$$

---

## 5. MVCC and Undo Log

### Undo Space

$$\text{Undo per Transaction} = \text{Modified Rows} \times \text{Row Size}$$

$$\text{Total Undo} = \text{Concurrent Transactions} \times \text{Avg Undo per Txn}$$

### History List Length

$$\text{History Length} = \text{Unpurged Transactions}$$

$$\text{Undo Growth Rate} = \text{TPS} \times \text{Avg Rows per Txn} \times \text{Row Size}$$

$$\text{Purge Lag} = \frac{\text{History Length}}{\text{Purge Rate (txns/sec)}}$$

| Long Txn Duration | TPS | History Length | Undo Bloat |
|:---:|:---:|:---:|:---:|
| 0 (no long txns) | 1,000 | ~0 | 0 |
| 10 sec | 1,000 | 10,000 | ~1 GiB |
| 60 sec | 1,000 | 60,000 | ~6 GiB |
| 600 sec | 1,000 | 600,000 | ~60 GiB |

---

## 6. Connection and Thread Math

### Thread Pool Sizing

$$\text{Optimal Threads} = \text{CPU Cores} \times (1 + \frac{T_{wait}}{T_{compute}})$$

For I/O-bound MySQL:

$$\text{Optimal} = \text{Cores} \times (1 + \frac{0.9}{0.1}) = \text{Cores} \times 10$$

### Connection Memory

$$\text{Per-Connection Memory} = \text{sort\_buffer\_size} + \text{read\_buffer\_size} + \text{join\_buffer\_size} + \text{thread\_stack}$$

$$\text{Total} = \text{max\_connections} \times \text{Per-Connection Memory}$$

| Connections | Per-Conn Memory | Total Reserved |
|:---:|:---:|:---:|
| 100 | 2 MiB | 200 MiB |
| 500 | 2 MiB | 1 GiB |
| 1,000 | 2 MiB | 2 GiB |
| 5,000 | 2 MiB | 10 GiB |

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| $\frac{16,256}{\text{Row Size}}$ | Division | Rows per page |
| $1 - \frac{\text{misses}}{\text{requests}}$ | Ratio | Buffer pool hit rate |
| $\frac{\text{Log Size}}{\text{Write Rate}}$ | Rate equation | Checkpoint interval |
| $\lceil \log_f(N) \rceil$ | Logarithmic | B+ tree depth |
| $\text{TPS} \times \text{Duration}$ | Linear | Undo history length |
| $\text{Hit} \times T_{mem} + (1-\text{Hit}) \times T_{disk}$ | Weighted average | Effective latency |

---

*Every `SHOW ENGINE INNODB STATUS`, `EXPLAIN`, and `information_schema.INNODB_BUFFER_POOL_STATS` query reflects these internals — a storage engine where understanding page layout and buffer pool math is the difference between a fast database and a slow one.*
