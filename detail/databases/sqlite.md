# The Mathematics of SQLite — Embedded Database Internals

> *SQLite is a serverless, file-based relational database. The math covers B-tree page structure, WAL mode performance, file locking overhead, and the unique constraints of an embedded engine.*

---

## 1. B-tree Page Structure

### The Model

SQLite stores all data in a single file organized as a collection of B-tree pages. Default page size = **4,096 bytes** (configurable: 512 to 65,536).

### Page Layout

$$\text{Page Usable} = \text{Page Size} - \text{Reserved Space (default 0)}$$

| Component | Size | Purpose |
|:---|:---:|:---|
| Page header | 8-12 bytes | Page type, cell count, pointers |
| Cell pointer array | 2 × cells bytes | Offsets to cell data |
| Free space | Variable | Unallocated |
| Cell content | Variable | Actual row data |

### Cells per Page

$$\text{Max Cells} = \frac{\text{Usable} - \text{Header}}{\text{Cell Size} + 2 \text{ (pointer)}}$$

For a table with 100-byte rows:

$$\text{Cells} \approx \frac{4096 - 12}{100 + 2 + 4} \approx 38 \text{ rows per page}$$

### B-tree Depth

$$\text{Interior Fan-out} = \frac{\text{Page Usable} - 12}{\text{Key Size} + 4 \text{ (child pointer)}}$$

For 8-byte integer keys:

$$f = \frac{4084}{12} = 340$$

| Rows | Depth | Page Reads per Lookup |
|:---:|:---:|:---:|
| 100 | 1 | 1 |
| 10,000 | 2 | 2 |
| 1,000,000 | 3 | 3 |
| 100,000,000 | 4 | 4 |

---

## 2. File Size and Overhead

### Database File Size

$$\text{File Size} = \text{Page Count} \times \text{Page Size}$$

$$\text{Page Count} = \text{Data Pages} + \text{Index Pages} + \text{Internal Pages} + \text{Free Pages}$$

### Row Overhead

$$\text{Row Storage} = \text{Header} + \text{Type Codes} + \text{Data}$$

| Component | Size |
|:---|:---|
| Record header length | 1-9 bytes (varint) |
| Column type codes | 1-9 bytes each (varint) |
| Column data | Varies by type |

### SQLite Type Sizes

| Type | Storage | Bytes |
|:---|:---|:---:|
| NULL | No storage | 0 |
| INTEGER (0) | Stored as type code | 0 |
| INTEGER (1 byte) | 8-bit signed | 1 |
| INTEGER (2 bytes) | 16-bit big-endian | 2 |
| INTEGER (4 bytes) | 32-bit big-endian | 4 |
| INTEGER (8 bytes) | 64-bit big-endian | 8 |
| REAL | 64-bit IEEE float | 8 |
| TEXT | UTF-8 encoded | Length |
| BLOB | Raw bytes | Length |

### Worked Example — Database Sizing

*"1 million rows: id (INT), name (avg 20 chars), email (avg 30 chars), age (INT)."*

| Column | Type Code | Data | Total |
|:---|:---:|:---:|:---:|
| id | 1 byte | 4 bytes | 5 bytes |
| name | 1 byte | 20 bytes | 21 bytes |
| email | 1 byte | 30 bytes | 31 bytes |
| age | 1 byte | 1 byte | 2 bytes |
| Header | 1 byte | - | 1 byte |
| **Row total** | | | **60 bytes** |

$$\text{Rows per page} = \frac{4084}{60 + 2} \approx 65$$

$$\text{Pages needed} = \frac{1,000,000}{65} = 15,385$$

$$\text{File size} = 15,385 \times 4,096 = 60 \text{ MiB}$$

With one index on email (avg 30 bytes per key):

$$\text{Index pages} = \frac{1,000,000}{4084 / (30 + 6)} \approx 8,800 \text{ pages} = 34.4 \text{ MiB}$$

$$\text{Total} \approx 94 \text{ MiB}$$

---

## 3. WAL Mode — Write-Ahead Logging

### The Model

WAL mode separates readers from writers. Writes go to the WAL file; readers see a consistent snapshot.

### WAL vs Rollback Journal

| Aspect | Rollback Journal | WAL Mode |
|:---|:---|:---|
| Writer blocks readers | Yes | No |
| Multiple readers | Yes | Yes |
| Multiple writers | No | No |
| Write location | Main DB file | WAL file |
| Read path | Main DB | Main DB + WAL |

### WAL Checkpoint

$$\text{WAL Size} = \text{Writes Since Last Checkpoint} \times \text{Avg Page Size Written}$$

$$T_{checkpoint} = \frac{\text{WAL Size}}{\text{Disk Write Speed}}$$

### Checkpoint Thresholds

$$\text{Auto-checkpoint at} = 1000 \text{ pages (default)} = 4 \text{ MiB (at 4K pages)}$$

| Checkpoint Threshold | WAL Max Size | Checkpoint Time (SSD) |
|:---:|:---:|:---:|
| 100 pages | 400 KiB | ~0.001 sec |
| 1,000 pages (default) | 4 MiB | ~0.008 sec |
| 10,000 pages | 40 MiB | ~0.08 sec |
| 100,000 pages | 400 MiB | ~0.8 sec |

### WAL Read Overhead

Each read must check the WAL for modified pages:

$$T_{read} = T_{wal\_index\_lookup} + T_{page\_read}$$

WAL index (shm file) uses a hash table: $O(1)$ lookup per page.

---

## 4. Locking Model

### Lock States

| Lock | Allows Concurrent | Held During |
|:---|:---|:---|
| UNLOCKED | Everything | No transaction |
| SHARED | Other SHARED | SELECT (reading) |
| RESERVED | SHARED only | First write in transaction |
| PENDING | Nothing new | Waiting for SHARED to clear |
| EXCLUSIVE | Nothing | Actual writing |

### Lock Escalation Path

$$\text{UNLOCKED} \rightarrow \text{SHARED} \rightarrow \text{RESERVED} \rightarrow \text{PENDING} \rightarrow \text{EXCLUSIVE}$$

### Busy Timeout Math

$$\text{Retries} = \frac{\text{busy\_timeout (ms)}}{\text{Retry Interval}}$$

Default retry uses exponential backoff:

$$T_{total} = \sum_{i=0}^{n} \min(T_{base} \times 2^i, T_{max})$$

### Concurrency Throughput (Rollback Mode)

$$\text{Write TPS} = \frac{1}{T_{fsync} + T_{write}} \approx \frac{1}{T_{fsync}}$$

| Storage | fsync Time | Max Write TPS |
|:---|:---:|:---:|
| HDD | 5-15 ms | 67-200 |
| SATA SSD | 0.1-0.5 ms | 2,000-10,000 |
| NVMe SSD | 0.02-0.05 ms | 20,000-50,000 |

### WAL Mode Write TPS

$$\text{WAL TPS} = \frac{1}{T_{wal\_append}} \approx \frac{1}{T_{sequential\_write}}$$

WAL is sequential, so typically 5-10x faster than rollback journal (which requires random I/O):

| Storage | Rollback TPS | WAL TPS | Speedup |
|:---|:---:|:---:|:---:|
| HDD | 100 | 500 | 5x |
| SATA SSD | 5,000 | 30,000 | 6x |
| NVMe | 25,000 | 100,000 | 4x |

---

## 5. Query Planner

### Cost Model

SQLite's planner estimates cost as number of disk reads + CPU operations:

$$\text{Cost} \approx \text{Pages Read} + 0.01 \times \text{Rows Processed}$$

### Table Scan vs Index

$$\text{Full Scan Cost} = \text{Table Pages}$$

$$\text{Index Scan Cost} = \text{Index Depth} \times \text{Matching Rows} + \text{Matching Rows (heap lookups)}$$

### Covering Index Advantage

$$\text{Covering Scan Cost} = \text{Index Depth} \times \text{Matching Rows} \quad (\text{no heap lookup})$$

### ANALYZE Statistics

SQLite stores 24 histogram samples per index for selectivity estimation:

$$\text{Estimated Selectivity} = \frac{\text{Matching Samples}}{24}$$

---

## 6. Maximum Limits

### SQLite Compile-Time Limits

| Limit | Default | Maximum |
|:---|:---:|:---:|
| Max database size | 281 TiB | $2^{46}$ bytes |
| Max page count | $2^{32} - 2$ | 4,294,967,294 |
| Max row size | 1 GiB | 1 GiB |
| Max columns per table | 2,000 | 32,767 |
| Max SQL length | 1 MiB | 1 GiB |
| Max attached databases | 10 | 125 |

### Page Count and Size Relationship

$$\text{Max DB Size} = \text{Max Pages} \times \text{Page Size}$$

| Page Size | Max Pages | Max DB Size |
|:---:|:---:|:---:|
| 512 bytes | 4,294,967,294 | 2 TiB |
| 4,096 bytes | 4,294,967,294 | 16 TiB |
| 65,536 bytes | 4,294,967,294 | 281 TiB |

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| $\frac{\text{Usable}}{\text{Row Size}}$ | Division | Rows per page |
| $\lceil \log_f(N) \rceil$ | Logarithmic | B-tree depth |
| $\frac{1}{T_{fsync}}$ | Reciprocal | Max write TPS |
| $\text{Pages} \times \text{Size}$ | Linear | Database file size |
| $\text{Writes} \times \text{Page Size}$ | Linear | WAL file size |
| $\text{Depth} \times \text{Rows}$ | Multiplicative | Index scan cost |

---

*Every `.schema`, `EXPLAIN QUERY PLAN`, and `PRAGMA page_count` reflects these internals — the world's most deployed database (trillions of instances) running on a single-file B-tree engine that fits in 700 KiB of code.*

## Prerequisites

- SQL fundamentals (queries, indexes, transactions)
- B-tree data structure concepts
- File I/O and page-based storage
- WAL (Write-Ahead Logging) basics

## Complexity

- **Beginner:** Database creation, basic queries, page size selection
- **Intermediate:** WAL mode tuning, page cache sizing, VACUUM mechanics
- **Advanced:** B-tree cell packing analysis, overflow page chains, WAL checkpoint frame accounting
