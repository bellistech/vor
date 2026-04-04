# The Mathematics of PostgreSQL — Query Planner and Storage Internals

> *PostgreSQL's query planner uses a cost-based optimizer with explicit cost formulas. The internals cover B-tree page structure, MVCC visibility rules, WAL mechanics, and vacuum overhead.*

---

## 1. Query Planner Cost Model

### The Model

PostgreSQL's planner estimates the total cost of executing a query plan. Cost is measured in abstract units anchored to a sequential page read = 1.0.

### Cost Parameters

| Parameter | Default | Meaning |
|:---|:---:|:---|
| `seq_page_cost` | 1.0 | Cost of sequential disk page read |
| `random_page_cost` | 4.0 | Cost of random disk page read |
| `cpu_tuple_cost` | 0.01 | Cost of processing one row |
| `cpu_index_tuple_cost` | 0.005 | Cost of processing one index entry |
| `cpu_operator_cost` | 0.0025 | Cost of evaluating one operator |
| `effective_cache_size` | 4 GB | Estimated OS + PG cache size |

### Sequential Scan Cost

$$\text{Cost}_{seq} = \text{seq\_page\_cost} \times N_{pages} + \text{cpu\_tuple\_cost} \times N_{rows}$$

### Index Scan Cost

$$\text{Cost}_{idx} = \text{random\_page\_cost} \times N_{pages\_read} + \text{cpu\_index\_tuple\_cost} \times N_{index\_entries} + \text{cpu\_tuple\_cost} \times N_{rows}$$

### Worked Example

*"Table with 10,000 pages, 1,000,000 rows. Query returns 1% (10,000 rows)."*

**Sequential scan:**

$$\text{Cost} = 1.0 \times 10,000 + 0.01 \times 1,000,000 = 10,000 + 10,000 = 20,000$$

**Index scan (10,000 rows, ~100 index pages, ~1,000 heap pages via random I/O):**

$$\text{Cost} = 4.0 \times 1,000 + 0.005 \times 10,000 + 0.01 \times 10,000 = 4,000 + 50 + 100 = 4,150$$

**Planner chooses index scan** (4,150 < 20,000).

### Break-Even Point

The selectivity where index scan = sequential scan:

$$\text{random\_page\_cost} \times s \times N_{pages} = \text{seq\_page\_cost} \times N_{pages}$$

$$s = \frac{\text{seq\_page\_cost}}{\text{random\_page\_cost}} = \frac{1.0}{4.0} = 25\%$$

**Rule of thumb:** Index scans are preferred when selectivity < 25% of rows. On SSDs, set `random_page_cost = 1.1` to shift this to ~91%.

---

## 2. B-tree Index Page Structure

### The Model

PostgreSQL B-tree indexes store key-pointer pairs in 8 KiB pages.

### Page Layout

$$\text{Page Size} = 8192 \text{ bytes}$$

$$\text{Page Header} = 24 \text{ bytes}$$

$$\text{Special Area} = 16 \text{ bytes (B-tree metadata)}$$

$$\text{Usable} = 8192 - 24 - 16 = 8152 \text{ bytes}$$

### Fill Factor

$$\text{Tuples per Page} = \left\lfloor \frac{8152 \times \text{Fill Factor}}{\text{Tuple Size}} \right\rfloor$$

Default fill factor = 90% for B-tree indexes.

| Key Size | Tuple Overhead | Tuple Size | Tuples/Page (90% fill) |
|:---:|:---:|:---:|:---:|
| 4 bytes (int) | 12 bytes | 16 bytes | 458 |
| 8 bytes (bigint) | 12 bytes | 20 bytes | 366 |
| 36 bytes (uuid) | 12 bytes | 48 bytes | 152 |
| 100 bytes (text) | 12 bytes | 112 bytes | 65 |

### Index Size Estimation

$$\text{Index Pages} = \lceil \frac{N_{rows}}{\text{Tuples per Page}} \rceil \times (1 + \frac{1}{f}) \quad (\text{+internal nodes})$$

Where $f$ = fan-out per level. Approximate:

$$\text{Index Size} \approx \frac{N_{rows} \times \text{Tuple Size}}{8152 \times 0.90} \times 8192 \times 1.01$$

| Rows | Key Size | Index Size |
|:---:|:---:|:---:|
| 1M | 4 bytes (int) | 17 MiB |
| 1M | 36 bytes (uuid) | 52 MiB |
| 10M | 4 bytes (int) | 171 MiB |
| 100M | 8 bytes (bigint) | 2.1 GiB |

### B-tree Depth

$$\text{Depth} = \lceil \log_f(N_{rows}) \rceil$$

With $f \approx 400$ for integer keys:

| Rows | Depth | Lookups |
|:---:|:---:|:---:|
| 1,000 | 2 | 2 page reads |
| 1,000,000 | 3 | 3 page reads |
| 1,000,000,000 | 4 | 4 page reads |

---

## 3. MVCC — Multi-Version Concurrency Control

### The Model

Every row version has `xmin` (creating transaction) and `xmax` (deleting transaction). Visibility depends on the snapshot.

### Visibility Rules

A tuple is **visible** to transaction $T$ with snapshot $S$ if:

$$\text{Visible} = (\text{xmin} < S.\text{xmin}) \land (\text{xmin committed}) \land (\text{xmax} = 0 \lor \text{xmax} > S.\text{xmin} \lor \text{xmax aborted})$$

Simplified:
1. xmin must be a committed transaction before the snapshot
2. xmax must be either zero (not deleted), not yet committed, or aborted

### Row Overhead per Version

$$\text{Tuple Header} = 23 \text{ bytes (HeapTupleHeader)}$$

| Field | Bytes | Purpose |
|:---|:---:|:---|
| t_xmin | 4 | Creating transaction ID |
| t_xmax | 4 | Deleting transaction ID |
| t_cid | 4 | Command ID within transaction |
| t_ctid | 6 | Current tuple ID (self-link or update chain) |
| t_infomask | 2 | Visibility hint flags |
| t_infomask2 | 2 | Number of attributes, HOT flag |
| t_hoff | 1 | Offset to data |

### Transaction ID Wraparound

$$\text{XID Space} = 2^{32} = 4,294,967,296$$

$$\text{Visibility Horizon} = 2^{31} = 2,147,483,648$$

Transactions older than $2^{31}$ XIDs are considered "in the future" — data becomes invisible.

$$\text{XIDs Until Wraparound} = 2^{31} - \text{Current Age}$$

$$\text{Time Until Wraparound} = \frac{2^{31} - \text{Age}}{\text{XID Rate (txns/sec)}}$$

| XID Rate | Time to Wraparound (from zero) |
|:---:|:---:|
| 100/sec | ~248 days |
| 1,000/sec | ~24.8 days |
| 10,000/sec | ~2.5 days |

**This is why aggressive autovacuum is critical at high transaction rates.**

---

## 4. WAL — Write-Ahead Log

### The Model

Every data modification is first written to WAL before the actual heap page. This ensures crash recovery.

### WAL Segment Size

$$\text{Segment Size} = 16 \text{ MiB (default, configurable at initdb)}$$

### Checkpoint Distance

$$\text{Checkpoint Interval} = \min(\text{checkpoint\_timeout}, T_{wal\_fill})$$

$$T_{wal\_fill} = \frac{\text{max\_wal\_size}}{\text{WAL Write Rate}}$$

Default: `max_wal_size = 1 GB`, `checkpoint_timeout = 5 min`.

| WAL Write Rate | Time to Fill 1 GB | Checkpoint Trigger |
|:---:|:---:|:---|
| 1 MB/s | 1024 sec (~17 min) | Timeout (5 min) |
| 10 MB/s | 102 sec (~1.7 min) | WAL size |
| 100 MB/s | 10 sec | WAL size |

### Full Page Writes

`full_page_writes = on` (default): After each checkpoint, the first modification to each page writes the entire 8 KiB page to WAL.

$$\text{WAL Amplification}_{first\_write} = \frac{8192}{\text{Change Size}}$$

For a 100-byte row update:

$$\text{Amplification} = \frac{8192}{100} = 81.9\times$$

### WAL Size Estimation

$$\text{WAL per Transaction} \approx \text{Rows Modified} \times (\text{Row Size} + \text{WAL Header (26 bytes)})$$

$$\text{WAL Rate} = \text{TPS} \times \text{WAL per Transaction}$$

---

## 5. Table Bloat and VACUUM

### The Model

Dead tuples (from UPDATEs and DELETEs) consume space until VACUUM reclaims them.

### Bloat Ratio

$$\text{Bloat Ratio} = \frac{\text{Dead Tuples}}{\text{Live Tuples} + \text{Dead Tuples}}$$

$$\text{Wasted Space} = \text{Table Size} \times \text{Bloat Ratio}$$

### Autovacuum Trigger

$$\text{Vacuum Threshold} = \text{autovacuum\_vacuum\_threshold} + \text{autovacuum\_vacuum\_scale\_factor} \times N_{rows}$$

Default: threshold = 50, scale_factor = 0.20.

$$\text{Trigger at} = 50 + 0.20 \times N_{rows} \text{ dead tuples}$$

| Table Rows | Dead Tuples to Trigger | Dead Tuple % |
|:---:|:---:|:---:|
| 1,000 | 250 | 25% |
| 100,000 | 20,050 | 20% |
| 10,000,000 | 2,000,050 | 20% |

### VACUUM I/O Cost

$$\text{Pages Scanned} = \text{All pages with dead tuples (visibility map guided)}$$

$$T_{vacuum} = \frac{\text{Pages to Scan} \times 8 \text{ KiB}}{\text{vacuum\_cost\_limit rate}} + \text{Index Cleanup}$$

---

## 6. Connection and Memory Math

### Shared Buffers

$$\text{Recommended} = \frac{\text{RAM}}{4} \quad (\text{typical guideline})$$

### Work Memory

$$\text{Max Memory} = \text{work\_mem} \times \text{max\_connections} \times \text{sort\_operations\_per\_query}$$

| work_mem | Connections | Sorts/Query | Max Total |
|:---:|:---:|:---:|:---:|
| 4 MiB | 100 | 3 | 1.2 GiB |
| 64 MiB | 100 | 3 | 18.8 GiB |
| 256 MiB | 200 | 3 | 150 GiB |

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| $\text{pages} \times \text{cost} + \text{rows} \times \text{cpu}$ | Linear combination | Query plan cost |
| $\lceil \log_f(N) \rceil$ | Logarithmic | B-tree depth |
| $2^{31}$ | Exponential constant | XID wraparound |
| $\frac{\text{WAL Size}}{\text{Write Rate}}$ | Rate equation | Checkpoint interval |
| $50 + 0.20 \times N$ | Linear threshold | Autovacuum trigger |
| $\frac{\text{Page Size}}{\text{Change Size}}$ | Ratio | Full-page write amplification |

---

*Every `EXPLAIN ANALYZE`, `pg_stat_user_tables`, and `pg_class.relpages` query exposes these cost calculations — a mathematical optimizer that has been refined over 30+ years to turn SQL into efficient disk I/O plans.*

## Prerequisites

- SQL fundamentals (queries, joins, indexes)
- B-tree index structure concepts
- MVCC (Multi-Version Concurrency Control) basics
- Cost-based query optimization principles
- Basic statistics (histograms, selectivity estimation)

## Complexity

- **Beginner:** EXPLAIN output reading, basic index selection
- **Intermediate:** Cost model tuning, buffer pool sizing, VACUUM mechanics
- **Advanced:** Join order optimization combinatorics, selectivity estimation with histograms, TOAST page overhead modeling
