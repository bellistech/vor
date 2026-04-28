# PostgreSQL — Deep Dive

> Cost-model arithmetic, MVCC tuple visibility, B-tree splits, WAL/checkpoint math, autovacuum thresholds, replication lag, and the 8-mode lock matrix — every knob defined inline, every formula written out.

---

## Query Planner Cost Model

The planner picks plans by minimising a numeric `cost` returned by `cost_*` functions in `src/backend/optimizer/path/costsize.c`. A cost is a synthetic unit anchored to "one sequential 8 KiB page read", with everything else expressed as a multiple of that anchor.

### Cost-tuple defaults (postgresql.conf)

| GUC | Default | Units | What it counts |
|:---|:---:|:---|:---|
| `seq_page_cost` | 1.0 | page | one sequentially-fetched 8 KiB page from heap |
| `random_page_cost` | 4.0 | page | one randomly-fetched 8 KiB page (heap or index) |
| `cpu_tuple_cost` | 0.01 | tuple | per-tuple CPU cost during a scan (visibility check, projection) |
| `cpu_index_tuple_cost` | 0.005 | index tuple | per-index-tuple CPU cost during an index scan |
| `cpu_operator_cost` | 0.0025 | op | one operator/function evaluation in a qual or target list |
| `parallel_tuple_cost` | 0.1 | tuple | shipping a tuple to a parallel worker |
| `parallel_setup_cost` | 1000 | per query | one-time cost to spin up workers |
| `effective_cache_size` | 4 GB | bytes | planner's *belief* about combined OS + shared_buffers cache; only used to estimate index access curves, not actually allocated |
| `min_parallel_table_scan_size` | 8 MB | bytes | smallest seq scan eligible for parallel execution |
| `min_parallel_index_scan_size` | 512 KB | bytes | smallest index scan eligible for parallel execution |
| `jit_above_cost` | 100000 | cost | turn JIT on for plans more expensive than this |
| `jit_inline_above_cost` | 500000 | cost | also inline functions in JITed code |
| `jit_optimize_above_cost` | 500000 | cost | also run LLVM `-O3` optimisations |

### When to tune

- **NVMe / cloud SSD**: lower `random_page_cost` toward `seq_page_cost` (typical 1.1–1.5). Random reads are nearly as cheap as sequential at the device.
- **Spinning rust**: keep `random_page_cost` at 4 or push to 6+. The planner must avoid index scans that fan out random heap fetches.
- **PG in containers with tight RAM**: shrink `effective_cache_size` or the planner over-uses indexes that will not actually fit in cache.
- **High `work_mem`**: hash and sort costs drop; the planner picks more hashes — expected.
- **Set per-tablespace** with `ALTER TABLESPACE ssd SET (random_page_cost = 1.1)` when you mix media.
- **Batch / OLAP**: drop `cpu_tuple_cost` slightly to bias toward parallel scans.

### Per-node cost formulas

Let `relpages = pg_class.relpages`, `reltuples = pg_class.reltuples` (last `ANALYZE` snapshot), `selectivity in [0,1]`.

**Seq Scan** (whole heap, no qual short-circuit):

```
seq_scan_cost  = relpages * seq_page_cost
               + reltuples * cpu_tuple_cost
               + reltuples * cpu_operator_cost * num_quals
```

**Index Scan** (B-tree, point or range, fetches heap tuples):

```
index_pages_visited  = ceil(selectivity * index_relpages)
heap_pages_visited   = ceil(selectivity * reltuples)        # worst case, all on different pages
correlation_factor   = 1 + (1 - correlation^2) * (heap_pages_visited - 1)

index_cost  = index_pages_visited * random_page_cost
            + index_tuples * cpu_index_tuple_cost
            + index_tuples * cpu_operator_cost
heap_cost   = correlation_factor * random_page_cost
            + matched_tuples * cpu_tuple_cost
            + matched_tuples * cpu_operator_cost * num_quals
total       = index_cost + heap_cost
```

Where `correlation = pg_stats.correlation` for the leading index column. Perfect correlation (1.0 or -1.0) collapses random heap fetches into sequential ones.

**Index-Only Scan** (visibility map all-visible bit set, all output cols in index):

```
total = index_pages_visited * random_page_cost
      + index_tuples * cpu_index_tuple_cost
      + index_tuples * cpu_operator_cost
      + (1 - all_visible_fraction) * heap_pages_visited * random_page_cost
```

`all_visible_fraction` = `pg_class.relallvisible / relpages`. When 1.0 the heap is never touched and the scan is *truly* index-only.

**Bitmap Heap Scan** (build TID bitmap from one-or-more index, sort by page, fetch heap):

```
bitmap_index_cost  = ceil(selectivity * index_relpages) * random_page_cost
                   + index_tuples * (cpu_index_tuple_cost + cpu_operator_cost)
heap_pages         = pages_touched_by_bitmap            # always <= relpages
bitmap_heap_cost   = heap_pages * cost_per_page
                     + matched_tuples * (cpu_tuple_cost + cpu_operator_cost * num_quals)
cost_per_page      = lerp(seq_page_cost, random_page_cost, density)
density            = heap_pages / relpages
```

The lerp shifts page cost from `random_page_cost` (sparse bitmap, lots of seek) toward `seq_page_cost` (dense bitmap, almost all pages touched). This is why bitmap heap scans dominate for "many tuples per page" plans.

**BitmapAnd / BitmapOr** combine two bitmap index scans:

```
bitmap_and_cost = sum(child_costs) + 0.1 * cpu_operator_cost * sum(child_tuples)
bitmap_or_cost  = sum(child_costs) + 0.1 * cpu_operator_cost * sum(child_tuples)
```

The combined `pages_touched` is then handed to bitmap heap scan.

**TID Scan** (`WHERE ctid = '(0,5)'`): one heap page fetch.

```
tid_scan_cost = num_tids * random_page_cost + num_tids * cpu_tuple_cost
```

### Join cost models

**Nested Loop**: outer side scanned once, inner side rescanned per outer tuple.

```
nestloop_cost  = outer_path_cost
               + outer_rows * inner_rescan_cost
               + outer_rows * inner_rows * cpu_tuple_cost
               + outer_rows * inner_rows * cpu_operator_cost * num_join_quals
```

If the inner is an index lookup, `inner_rescan_cost` is the index-scan cost re-evaluated per outer row — this is why selective inner indexes make nested loops scale.

**Hash Join**: build hash on inner, probe with outer.

```
hash_build_cost = inner_path_cost
                + inner_rows * cpu_operator_cost           # hash function
                + inner_rows * cpu_tuple_cost
                + spill_cost                                # if > work_mem, multi-batch
hash_probe_cost = outer_path_cost
                + outer_rows * cpu_operator_cost
                + matched_pairs * cpu_tuple_cost
total           = hash_build_cost + hash_probe_cost
```

`spill_cost = ceil(inner_size / work_mem) * (inner_size / 8KiB) * seq_page_cost` for the multi-batch tempfile dance. Set `work_mem` high enough to keep `nbatches = 1`.

**Merge Join**: both inputs pre-sorted, single linear scan.

```
merge_cost = outer_sort_cost + inner_sort_cost
           + (outer_rows + inner_rows) * cpu_tuple_cost
           + matched_pairs * cpu_operator_cost
```

If indexes provide ordering, the sort costs vanish. Merge join wins on huge equi-joins where neither side fits in `work_mem`.

### Worked EXPLAIN — Seq Scan vs Index Scan break-even

Schema: `events(id bigserial PK, ts timestamptz, payload jsonb)`, 10 M rows, `relpages=185000`, `reltuples=1e7`, B-tree on `ts`, leaf pages = 27,000.

```sql
EXPLAIN ANALYZE SELECT * FROM events WHERE ts > now() - interval '1 hour';
```

Selectivity ≈ 0.0004 (one hour out of an estimated month-long sample).

```
matched_rows         = 0.0004 * 1e7 = 4000
index_pages_visited  = ceil(0.0004 * 27000) = 11
heap_pages_visited   = 4000           # uncorrelated payload
correlation          = 0.92           # ts is monotonic insert order

correlation_factor   = 1 + (1 - 0.92^2) * (4000 - 1) ~= 626
index_cost           = 11 * 4.0  + 4000 * 0.005 + 4000 * 0.0025
                     = 44 + 20 + 10 = 74
heap_cost            = 626 * 4.0 + 4000 * 0.01 + 4000 * 0.0025
                     = 2504 + 40 + 10 = 2554
index_total          = 2628

seq_scan_cost        = 185000 * 1.0 + 1e7 * 0.01 + 1e7 * 0.0025
                     = 185000 + 100000 + 25000 = 310000
```

Planner picks index — over 100x cheaper. Crank `random_page_cost` from 4 to 16 (HDD setting) and the index plan rises to about 10500, still cheaper than seq. Drop `correlation` to 0.0 (random insert order) and `correlation_factor` becomes about 3999, index plan about 16060 — still wins, but bitmap heap scan would beat it.

### Worked EXPLAIN — Bitmap heap scan

Same table, broader query:

```sql
EXPLAIN ANALYZE SELECT id FROM events WHERE ts BETWEEN '2026-04-01' AND '2026-04-15';
```

Selectivity = 0.05, matched_rows = 500,000.

```
index_pages_visited = ceil(0.05 * 27000) = 1350
heap_pages_touched  = ~50000          # half the table by page count, since ts is correlated
density             = 50000 / 185000 = 0.27
cost_per_page       = lerp(1.0, 4.0, 0.27) = 1.81

bitmap_index_cost   = 1350 * 4.0 + 500000 * 0.0075 = 5400 + 3750 = 9150
bitmap_heap_cost    = 50000 * 1.81 + 500000 * 0.0125 = 90500 + 6250 = 96750
total_bitmap        = 105900
```

The plain index scan would cost about `correlation_factor * 4.0 + ...` ~= 200K+. Planner picks bitmap.

### Worked EXPLAIN — Hash vs nested loop

```sql
EXPLAIN ANALYZE
SELECT *
  FROM orders o
  JOIN customers c ON c.id = o.customer_id
 WHERE c.country = 'NZ';
```

`customers` 100K rows, 2K NZ. `orders` 50M rows. Index on `customers(country)` and `orders(customer_id)`.

Nested loop, NZ-customers as outer:

```
outer_path_cost   = 200          # bitmap on customers(country)
inner_rescan_cost = 5            # per-NZ-customer index lookup against orders
matched_per_outer = 500          # avg orders per customer
nestloop_cost     ~= 200 + 2000 * 5 + 2000 * 500 * 0.0125
                  = 200 + 10000 + 12500 = 22700
```

Hash join:

```
hash_build (customers NZ)  = 200 + 2000 * 0.0025 + 2000 * 0.01 = 225
hash_probe (orders SeqScan)= 50e6 * 0.01 + 50e6 * 0.0025 + relpages*1.0
                           = 500000 + 125000 + ~900000 = 1525000
```

Nested loop wins by 67x — the 50M-row probe side dominates the hash plan. Drop the `country` filter and the math flips: hash builds on 100K customers in about 3K cost, probe is unavoidable, total about 1.5M; nested loop becomes 100K * (5 + 500*0.0125) = 1.13M — still close. With matched orders/customer = 50, hash wins.

### Genetic Query Optimizer (GEQO)

When `from_collapse_limit + join_collapse_limit < num_relations` and `geqo = on`, the planner switches from exhaustive search to a genetic algorithm. Defaults: `geqo_threshold = 12`, `geqo_effort = 5`, `geqo_pool_size = 0` (auto = `2^effort`), `geqo_generations = 0` (auto = `effort * num_relations`). Set `geqo_seed = 0` for reproducibility.

### enable_* override knobs

Used when you suspect the planner is making the wrong call (or for unit tests):

```
enable_seqscan                = on
enable_indexscan              = on
enable_indexonlyscan          = on
enable_bitmapscan             = on
enable_tidscan                = on
enable_hashjoin               = on
enable_mergejoin              = on
enable_nestloop               = on
enable_hashagg                = on
enable_material               = on
enable_partitionwise_join     = off    # off by default
enable_partitionwise_aggregate = off
enable_parallel_hash          = on
enable_parallel_append        = on
enable_async_append           = on     # PG 14+, for FDWs
```

Setting `enable_seqscan = off` does NOT disable seq scans — it adds `disable_cost = 1e10` to that path. The planner still picks seq scan if every alternative is worse.

---

## MVCC Internals

### Tuple header layout

Each heap tuple (in `htup_details.h`) carries 23 bytes of header before the data:

```c
typedef struct HeapTupleHeaderData {
    union {
        HeapTupleFields t_heap;       /* xmin, xmax, t_cid (or t_xvac) */
        DatumTupleFields t_datum;
    } t_choice;
    ItemPointerData t_ctid;            /* (block, offset) - self or HOT successor */
    uint16 t_infomask2;                /* attribute count + flags */
    uint16 t_infomask;                 /* visibility flags (HEAP_XMIN_COMMITTED, ...) */
    uint8  t_hoff;                     /* header length incl. nulls bitmap */
    bits8  t_bits[FLEXIBLE_ARRAY_MEMBER];
} HeapTupleHeaderData;
```

Fields:

- **xmin** — TransactionId that inserted the tuple. The "creator" XID.
- **xmax** — TransactionId that deleted/locked the tuple, or 0 if live. On DELETE/UPDATE this becomes the deleter; on row-level lock it becomes the locker (with `HEAP_XMAX_LOCK_ONLY` set).
- **t_cid** — CommandId (within xmin's xact). Used so a single transaction sees its own statement boundaries (`cmin` for inserter, `cmax` for deleter — the union packs both into one 4-byte slot via `t_xvac`).
- **xvac** — used only by old-style VACUUM FULL (rare since 9.0).
- **ctid** — points to the tuple version itself, or to the *newer* version of an updated row (HOT chain link).

Hint bits in `t_infomask`:

| Bit | Meaning |
|:---|:---|
| `HEAP_XMIN_COMMITTED` | xmin is known committed (saves a clog lookup) |
| `HEAP_XMIN_INVALID` | xmin aborted |
| `HEAP_XMIN_FROZEN` | tuple is frozen — visible to all future xacts |
| `HEAP_XMAX_COMMITTED` | xmax is known committed |
| `HEAP_XMAX_INVALID` | xmax aborted (or never set) |
| `HEAP_XMAX_LOCK_ONLY` | xmax is a row lock, not a delete |
| `HEAP_HASNULL` / `HEAP_HASVARWIDTH` / `HEAP_HASOID_OLD` | data layout hints |
| `HEAP_HOT_UPDATED` | this tuple was HOT-updated (ctid points to newer version) |
| `HEAP_ONLY_TUPLE` | this tuple is a HOT successor (no index entries point at it) |

The first read of a tuple must cross-check xmin/xmax against pg_xact (formerly clog) to learn commit status. After the cross-check the bit gets set lazily, so subsequent reads skip the clog hit. This is why a fresh `SELECT` after big imports is unexpectedly slow — every page is dirtied just to set hint bits.

### HeapTupleSatisfiesMVCC algorithm

```
function visible(tuple, snapshot):
    xmin = tuple.xmin
    xmax = tuple.xmax

    # Step 1: was the inserter committed before this snapshot?
    if xmin == InvalidTransactionId:
        return false                       # never committed
    if xmin == GetCurrentTransactionId():
        if tuple.cmin >= snapshot.curcid:
            return false                    # later command in same xact
        if xmax == 0:
            return true
        if xmax == GetCurrentTransactionId() and tuple.cmax < snapshot.curcid:
            return false                    # we deleted it earlier in same xact
        return true
    if xid_in_progress(xmin, snapshot):
        return false
    if not committed(xmin):
        return false
    if xmin >= snapshot.xmax:
        return false                        # committed after our snapshot
    if xmin in snapshot.xip:
        return false                        # was in-progress when snapshot taken

    # Step 2: was the deleter committed before this snapshot?
    if xmax == 0:
        return true
    if xmax_is_lock_only(tuple):
        return true                         # locker, not deleter
    if xmax == GetCurrentTransactionId():
        if tuple.cmax >= snapshot.curcid:
            return true
        return false
    if xid_in_progress(xmax, snapshot):
        return true
    if not committed(xmax):
        return true
    if xmax >= snapshot.xmax or xmax in snapshot.xip:
        return true
    return false
```

The two-step test is the heart of MVCC: a tuple is visible iff its inserter is committed-and-pre-snapshot, *and* its deleter is either absent, a locker-only, or post-snapshot.

### Snapshot layout

```c
typedef struct SnapshotData {
    TransactionId xmin;        /* smallest in-progress xid at snapshot time */
    TransactionId xmax;        /* 1 + max committed xid; >= xmax is invisible */
    TransactionId *xip;        /* sorted array of in-progress xids */
    uint32 xcnt;               /* length of xip */
    CommandId curcid;          /* current command-id within this transaction */
    bool takenDuringRecovery;
    /* ... */
} SnapshotData;
```

Membership test in `xip` is binary search — O(log xcnt). With many concurrent transactions xcnt grows and the test gets slower; this is one driver behind the connection-pool gospel ("don't run with `max_connections=2000`, use pgbouncer").

`pg_export_snapshot()` serialises a snapshot into a string usable by `SET TRANSACTION SNAPSHOT 'xxx'` in another session — this is how `pg_dump --jobs=N` keeps consistent dumps across parallel workers.

### HOT updates and ctid chains

A **Heap-Only Tuple (HOT) update** mutates a row in place when:

1. None of the indexed columns change.
2. The new tuple fits on the same heap page.

Then PG writes the new version on the same page, sets `HEAP_HOT_UPDATED` on the old, sets `HEAP_ONLY_TUPLE` on the new, and links them by `ctid`. Index entries continue to point at the *original* tuple, and the index-scan code follows the chain.

`pageinspect`:

```sql
CREATE EXTENSION pageinspect;
SELECT lp, t_ctid, t_xmin, t_xmax, t_infomask::bit(16)
  FROM heap_page_items(get_raw_page('foo', 0));
```

Chain length grows under hot-spot updates; once it exceeds about 7-8 deep VACUUM will *prune* the chain (free intermediate dead tuples while keeping the index stable). The chain head is updated to point at the surviving root.

If even one indexed column changes, a normal (non-HOT) update is forced — index entries multiply, write amplification jumps. This is the math behind the "index your queries, not your fields" advice.

```
write_amplification_HOT  = 1
write_amplification_full = 1 + num_indexes
```

### Visibility map

Stored at `<relation>_vm` with **two bits per heap page**:

- `VISIBILITYMAP_ALL_VISIBLE` (bit 0) — every tuple on the page is visible to every current and future transaction.
- `VISIBILITYMAP_ALL_FROZEN` (bit 1) — every tuple has xmin frozen; aggressive VACUUM may skip the page entirely.

VM size: `ceil(relpages / 32768) * 8 KiB` (each VM page covers 32768 heap pages, 2 bits each = 8192 bytes).

Lifecycle:

1. Tuple gets inserted/updated -> page's `ALL_VISIBLE` bit cleared.
2. VACUUM scans the page, finds no dead/in-progress tuples -> sets `ALL_VISIBLE`.
3. VACUUM freezes a page (all xmins below `vacuum_freeze_min_age`) -> sets `ALL_FROZEN`.
4. Any DML on the page -> both bits cleared again.

Index-only scans depend on `ALL_VISIBLE`; without it, an "index only" scan is forced to peek at the heap to confirm visibility.

### XID wrap-around math

XIDs are 32-bit. The space is treated as a circular ordering, so any XID ahead of `current - 2^31` is "in the future" and any XID behind is "in the past". Once the current XID has advanced 2^31 past a tuple's xmin, the tuple appears in the future and disappears.

```
TOTAL_XIDS         = 2^32 = 4,294,967,296
HALF_XIDS          = 2^31 = 2,147,483,648
autovacuum_freeze_max_age default = 200,000,000     # 200 M
vacuum_freeze_min_age default     =  50,000,000     #  50 M
vacuum_freeze_table_age default   = 150,000,000     # 150 M
```

When the oldest unfrozen XID in a relation is older than `autovacuum_freeze_max_age`, autovacuum runs an **anti-wraparound** vacuum — even if it's disabled, even on a quiet table. When the oldest unfrozen XID is older than `vacuum_freeze_table_age`, a normal vacuum scans the entire heap (not just unvisited pages from the VM).

Headroom budget: `2^31 - autovacuum_freeze_max_age = 1,947,483,648` XIDs before single-user-mode lockout. At 100K tx/s sustained, that's about 5.4 hours — autovacuum must keep up.

```sql
SELECT datname,
       age(datfrozenxid)        AS xid_age,
       2^31 - age(datfrozenxid) AS xids_remaining
  FROM pg_database;
```

Wraparound shutdown: at `age(datfrozenxid) >= 2^31 - 3,000,000` the database refuses new transactions and only single-user mode (`postgres --single`) can run `VACUUM FREEZE`. PG 9.6+ widens this with multixact wraparound (separate counter; same model).

| XID rate | Time from freshly-initdb cluster to 200 M (autovacuum trigger) |
|:---:|:---:|
| 100 / sec | 23 days |
| 1 K / sec | 2.3 days |
| 10 K / sec | 5.5 hours |
| 100 K / sec | 33 minutes |

---

## B-Tree Indexes — Split Mathematics

### Page geometry

```
PAGE_SIZE             = 8192 bytes (BLCKSZ default; compile-time)
PAGE_HEADER           =   24 bytes
SPECIAL_SPACE_BTREE   =   16 bytes (BTPageOpaqueData)
ITEM_ID               =    4 bytes per pointer in the line pointer array
USABLE_PER_PAGE       = 8192 - 24 - 16 = 8152 bytes
```

A B-tree leaf entry is `IndexTupleData (8 bytes) + key + heap ctid (6 bytes)`. Round to MAXALIGN (8). Effective fanout:

```
b = floor(USABLE_PER_PAGE / (avg_index_entry_size + ITEM_ID))
T_btree = ceil(log_b(N))   where b = (8192 - 24) / (avg_index_entry_size + 8)
```

For an `int8` PK (8-byte key): entry = 8 (header) + 8 (key) + 6 (ctid) = 22 -> 24 with alignment, plus 4 ItemId = 28. `b = 8152 / 28 ~= 291`. Tree depth at 1 G rows = `ceil(log_291(1e9)) = ceil(3.55) = 4`.

| Rows | Fanout 100 (text key) | Fanout 291 (int8 key) | Fanout 580 (int4 key) |
|:---:|:---:|:---:|:---:|
| 1 K | 2 | 2 | 2 |
| 1 M | 4 | 3 | 3 |
| 100 M | 5 | 4 | 3 |
| 10 G | 6 | 5 | 4 |

### Fill factor

`CREATE INDEX ... WITH (fillfactor = 90)` controls how full leaf pages are at build time. Default = 90% for B-tree (was 100% pre-9.0), 70% for tables. Lower fillfactor reserves space for in-place inserts and HOT updates.

### Leaf split

When a leaf has insufficient free space for a new tuple, a split fires. PG implements **rightmost-page heuristics**: monotonic-insert workloads (e.g. `bigserial`) split asymmetrically — the new key always lands on the new right sibling, and the old page stays full. This minimises wasted space for append-heavy tables.

For random keys, splits are 50/50.

```
post_split_fill = 0.50 (random) or {old=1.00, new=0.10} (rightmost heuristic)
splits_per_insert ~= 1 / b      # amortised
new_root_when    = root_fanout exceeded         # height += 1
```

Tree height grows on a root split — costly because every concurrent reader must cross the new root for the next read. Use `pgstattuple` and `pgstatindex` to monitor leaf density:

```sql
CREATE EXTENSION pgstattuple;
SELECT * FROM pgstatindex('idx_events_ts');
```

Watch `leaf_fragmentation`. Above 30% suggests REINDEX (or use `REINDEX CONCURRENTLY`).

### Index-only scan eligibility

For an index `idx (a, b) INCLUDE (c)` to satisfy `SELECT a, b, c WHERE a = ?`:

- Index covers all output columns (yes).
- The matching heap pages must have `VISIBILITYMAP_ALL_VISIBLE` set (a vacuum must have run since last write).
- Predicate columns must be in the index (`a` here).

`INCLUDE` columns participate in the leaf payload but not in B-tree ordering — useful for fitting hot lookups under `idx-only` without bloating sort comparisons.

### Partial and expression indexes

Partial index:

```sql
CREATE INDEX events_recent ON events(ts) WHERE status = 'pending';
```

Stored size shrinks by selectivity. Planner uses it iff the query's predicates *imply* the index predicate (proven by the `predtest.c` boolean simplifier).

Expression index:

```sql
CREATE INDEX users_lower_email ON users (lower(email));
SELECT ... WHERE lower(email) = $1;
```

Function must be `IMMUTABLE` (or `STABLE` only with `CREATE INDEX ... IMMUTABLE` cheat; don't). Cost: every insert/update re-evaluates the expression.

### Deduplication (PG 13+)

When duplicate keys point at many heap tuples, leaf entries are de-duplicated into a single posting tuple holding a list of TIDs. Storage savings can be 5-10x on low-cardinality columns. Disable with `WITH (deduplicate_items = off)` if your inserts are already uniformised.

### B-tree page pruning vs index vacuum

Two cleanup paths:

- **Opportunistic page-level pruning** (`_bt_simpledel_pass`): an INSERT that would split the page first kills LP_DEAD-marked items, possibly saving the split.
- **Bulk index vacuum**: VACUUM walks every leaf, clearing entries whose TIDs landed in the dead-tuple list.

The first is cheap (no I/O beyond the working page) but conservative. The second is the heavy hammer but bounded by `maintenance_work_mem`.

---

## WAL & Recovery

### WAL segment geometry

```
WAL segment file size  = 16 MB (default; configurable at initdb via --wal-segsize)
WAL records            = variable length, prefixed by XLogRecord (24 bytes)
WAL block size         = 8 KiB inside the segment
LSN                    = 64-bit byte offset into the global WAL stream
```

WAL position math:

```
wal_bytes_written = pg_wal_lsn_diff(pg_current_wal_lsn(), '0/0')
segments_used     = wal_bytes_written / (16 * 1024 * 1024)
```

### Checkpoint algorithm

A checkpoint flushes every dirty buffer to its data file, then writes a new restartpoint record.

Triggers:

1. `checkpoint_timeout` elapsed (default 5 min, max 1 day).
2. WAL volume since last checkpoint exceeds `max_wal_size` (default 1 GB).
3. Manual `CHECKPOINT;` issued.
4. shutdown / pg_basebackup / pg_start_backup.

I/O smoothing — each checkpoint spreads its writes across `checkpoint_completion_target` (default 0.9, was 0.5 pre-14) of the inter-checkpoint interval:

```
inter_checkpoint  = min(checkpoint_timeout, max_wal_size / write_rate)
target_duration   = inter_checkpoint * checkpoint_completion_target
write_rate_limit  = dirty_pages / target_duration
```

Example: 1 GB max_wal_size, 80 MB/s write rate, completion target 0.9:

```
inter_checkpoint = min(300, 1024/80) = min(300, 12.8) = 12.8 s
target_duration  = 12.8 * 0.9 = 11.5 s
```

If the buffer pool is mostly dirty (say 4 GB), write rate = 4096/11.5 ~= 356 MB/s — too aggressive for typical SSDs and you'll see tail-latency spikes. Either raise `max_wal_size` (longer interval, more dirty data), or shrink `shared_buffers` (less peak dirty), or lower `bgwriter_lru_maxpages` to encourage continual background flushing.

GUCs:

| GUC | Default | Notes |
|:---|:---:|:---|
| `checkpoint_timeout` | 5 min | raise to 15-30 min on big buffer pools |
| `max_wal_size` | 1 GB | raise so timeouts trigger before volume |
| `min_wal_size` | 80 MB | keep this many recycled segments |
| `checkpoint_completion_target` | 0.9 | rarely needs tuning post-14 |
| `checkpoint_warning` | 30 s | logs "checkpoints occurring too frequently" |
| `bgwriter_lru_maxpages` | 100 | per round of bgwriter |
| `bgwriter_delay` | 200 ms | sleep between bgwriter rounds |
| `wal_compression` | off | enable lz4/zstd to shrink full-page writes (PG 15+) |
| `wal_buffers` | -1 (= 1/32 shared_buffers, max 16 MB) | rarely needs raising |
| `wal_writer_delay` | 200 ms | wal writer sleep |
| `wal_writer_flush_after` | 1 MB | force flush after this many bytes |

### Full-page writes

`full_page_writes = on` (default). The first time a page is written to WAL after a checkpoint, the whole 8 KiB page is recorded — torn-write protection. This explodes WAL volume right after a checkpoint. Heuristic: WAL volume per checkpoint cycle ~= `dirty_pages * 8 KiB` plus the row-level deltas.

```
wal_amplification_first_write = 8192 / change_size
```

For a 100-byte row update: 8192 / 100 = 81.9x amplification. Pushing the checkpoint interval out smears this over more updates per page.

### Recovery

Crash recovery:

1. Read the last checkpoint location from `pg_control`.
2. Replay WAL from there until end-of-WAL.
3. Mark database consistent.

Recovery throughput: roughly 100-500 MB/s of WAL replay on modern NVMe (single-threaded redo), dramatically faster post-PG 15 with prefetching (`recovery_prefetch = try`).

```
recovery_time ~= (latest_lsn - last_checkpoint_lsn) / replay_rate
              ~= 0 ... checkpoint_timeout * write_rate / replay_rate
```

A 5-minute checkpoint cycle at 80 MB/s write rate, 200 MB/s replay rate -> up to 2 minutes recovery. Tighten `checkpoint_timeout` to bound this.

### PITR

```
recovery.signal           # opt-in
restore_command = 'cp /archive/%f %p'
recovery_target_time = '2026-04-27 14:32:17.123456+00'
recovery_target_inclusive = true
recovery_target_action    = 'pause' | 'promote' | 'shutdown'
```

`archive_command` runs once per closed segment. Set `archive_timeout = 60s` to force a segment close on idle databases — without it, low-traffic systems can lose minutes of recoverability. Math:

```
worst_case_data_loss = max(archive_timeout, time_since_last_segment_close)
```

`pg_receivewal` streams segments live (no `archive_timeout` lag) — preferred for low-RPO setups.

### synchronous_commit modes

| Mode | Wait for | Durability if crash on primary | Latency penalty |
|:---|:---|:---|:---|
| `off` | nothing (return after WAL buffer write) | up to `wal_writer_delay` * 3 of writes lost | none |
| `local` | local flush only | all committed xacts durable | `2 * fsync_time` |
| `remote_write` | local flush + standby received | xact survives standby promotion | local flush + RTT/2 |
| `remote_apply` | local flush + standby applied | standby read consistent | local flush + RTT + apply |
| `on` (default) | local flush; if `synchronous_standby_names` set, also standby flush | both ends durable | local flush + RTT + remote flush |

Set per-transaction: `SET LOCAL synchronous_commit = remote_apply` to opt in for high-durability writes only. Bulk loads pay heavy penalty from `synchronous_commit = on`; flip to `off` for the load and `CHECKPOINT` after.

```sql
BEGIN;
SET LOCAL synchronous_commit = off;
COPY huge_table FROM '/tmp/data.csv';
COMMIT;
CHECKPOINT;
```

---

## VACUUM & Autovacuum

### Threshold formulas

Autovacuum launches a worker per database every `autovacuum_naptime` (default 1 minute). For each table:

```
n_dead          = pg_stat_all_tables.n_dead_tup
n_inserts       = pg_stat_all_tables.n_ins_since_vacuum     # PG 13+
reltuples       = pg_class.reltuples

effective_threshold = autovacuum_vacuum_threshold
                    + autovacuum_vacuum_scale_factor * reltuples
ana_threshold       = autovacuum_analyze_threshold
                    + autovacuum_analyze_scale_factor * reltuples
ins_threshold       = autovacuum_vacuum_insert_threshold
                    + autovacuum_vacuum_insert_scale_factor * reltuples   # PG 13+
```

Defaults:

| GUC | Default | Effect |
|:---|:---:|:---|
| `autovacuum_vacuum_threshold` | 50 | floor for vacuum trigger |
| `autovacuum_vacuum_scale_factor` | 0.2 | 20% of table dead tuples |
| `autovacuum_analyze_threshold` | 50 | floor for analyze trigger |
| `autovacuum_analyze_scale_factor` | 0.1 | 10% of table changed |
| `autovacuum_vacuum_insert_threshold` | 1000 | floor for insert-driven vacuum |
| `autovacuum_vacuum_insert_scale_factor` | 0.2 | 20% inserts trigger vacuum (for VM bit setting) |
| `autovacuum_freeze_max_age` | 200,000,000 | force anti-wraparound |
| `autovacuum_multixact_freeze_max_age` | 400,000,000 | multixact wraparound |
| `autovacuum_naptime` | 60 s | scan interval |
| `autovacuum_max_workers` | 3 | per cluster |

Vacuum triggers when `n_dead > effective_threshold`. Big tables suffer: a 1-billion-row table needs 200 M dead tuples before the *default* threshold fires. Set per-table:

```sql
ALTER TABLE events SET (autovacuum_vacuum_scale_factor = 0.01,
                        autovacuum_vacuum_threshold     = 1000);
```

| Table rows | Default trigger | Aggressive trigger (sf=0.01,t=1000) |
|:---:|:---:|:---:|
| 1 K | 250 dead | 1010 dead (effectively never on small tables) |
| 100 K | 20,050 | 2000 |
| 10 M | 2,000,050 | 101,000 |
| 1 B | 200,000,050 | 10,001,000 |

### Vacuum cost throttling

Autovacuum runs in cost-limited bursts to avoid I/O storms:

```
cost_per_page_hit    = vacuum_cost_page_hit    = 1
cost_per_page_miss   = vacuum_cost_page_miss   = 2  (PG 14+: was 10)
cost_per_page_dirty  = vacuum_cost_page_dirty  = 20
cost_limit           = autovacuum_vacuum_cost_limit = -1  (=> vacuum_cost_limit = 200)
delay                = autovacuum_vacuum_cost_delay = 2 ms (PG 12+; was 20 ms)

batch_cost = sum_over_pages_in_batch(weight * pages)
if batch_cost >= cost_limit:
    sleep(delay)
    batch_cost = 0
```

Throughput ceiling:

```
max_pages_per_sec = (cost_limit / cost_per_page_dirty) * (1000 / delay)
                  = (200 / 20) * (1000 / 2) = 5000 pages/sec = 39 MB/s
```

Big OLTP databases routinely raise `vacuum_cost_limit` to 1000-10000 to keep up. `autovacuum_vacuum_cost_limit = -1` inherits the foreground value.

### Freezing cliffs

```
xid_age(table) = age(pg_class.relfrozenxid)
```

- `xid_age >= vacuum_freeze_min_age` (50 M default): VACUUM may freeze tuples on visited pages.
- `xid_age >= vacuum_freeze_table_age` (150 M): VACUUM scans the whole heap (ignores VM).
- `xid_age >= autovacuum_freeze_max_age` (200 M): autovacuum forces a freeze even with autovacuum disabled.
- `xid_age >= 2,000,000,000` (2 B): emergency, single-user-mode required.

The "freezing cliff" is the moment your slow-trickling autovacuum can't keep up and the cluster does a stop-the-world anti-wraparound vacuum. Mitigation: aggressive `vacuum_freeze_table_age = 50000000`, cold partitions get one-time `VACUUM FREEZE`, watch `pg_stat_progress_vacuum`.

### pg_stat_progress_vacuum interpretation

```sql
SELECT pid, datname, relid::regclass,
       phase, heap_blks_total, heap_blks_scanned, heap_blks_vacuumed,
       index_vacuum_count, max_dead_tuples, num_dead_tuples
  FROM pg_stat_progress_vacuum;
```

Phases:

1. **initializing** — waiting for snapshot.
2. **scanning heap** — first heap pass; if `heap_blks_scanned < heap_blks_total`, you're here. ETA = `(total - scanned) / scan_rate`.
3. **vacuuming indexes** — for each index, full sequential scan pruning dead TIDs. Repeats per `index_vacuum_count` round if `maintenance_work_mem` overflows.
4. **vacuuming heap** — second heap pass to actually mark line pointers dead.
5. **cleaning up indexes** — final per-index cleanup (e.g., btree leaf merges).
6. **truncating heap** — reclaiming trailing empty pages back to OS.
7. **performing final cleanup** — pg_class updates.

`max_dead_tuples = (maintenance_work_mem * 1024 * 1024 - overhead) / 6` (each TID is 6 bytes). When `num_dead_tuples >= max_dead_tuples`, vacuum re-enters the index-cleanup round trip.

```
maintenance_work_mem = 256 MB -> ~46 M TIDs per round
table with 200 M dead -> 5 index passes -> 5x heap scan amplification on indexes
```

Set `maintenance_work_mem = 1 GB` (or higher) to compress this.

PG 17+ introduced TID-store rewrite: VACUUM uses a radix-tree-backed tid-store and the `(maintenance_work_mem - overhead)/6` formula no longer applies; one pass suffices for most tables.

### Free space map (FSM) and visibility map (VM)

Each relation has an FSM (`<relid>_fsm`) and VM (`<relid>_vm`) sibling fork.

- FSM is a tree of upper-bound free space estimates. `INSERT`s walk it to find a page with enough room.
- FSM size: about 1/3000 of heap size — for a 100 GB table, about 30 MB FSM.
- Vacuum updates FSM with actual freed bytes per page.

When FSM is wrong (vacuum hasn't run, or the table is read-only growth), inserts always hit the rightmost page even if free space exists upstream — bloat builds.

### Bloat estimation

```
expected_size = (reltuples * avg_tuple_size) / fillfactor
actual_size   = relpages * 8 KiB
bloat_ratio   = (actual_size - expected_size) / actual_size
```

`avg_tuple_size = 23 (header) + null_bitmap_bytes + sum(pg_stats.avg_width)`.

```sql
SELECT schemaname, relname,
       pg_size_pretty(pg_total_relation_size(relid)) AS total,
       n_dead_tup, n_live_tup,
       round(100.0 * n_dead_tup / NULLIF(n_live_tup + n_dead_tup, 0), 2) AS dead_pct
  FROM pg_stat_user_tables
 ORDER BY n_dead_tup DESC LIMIT 20;
```

Trigger `VACUUM (FULL, ANALYZE)` only when `bloat_ratio > 0.4` AND the wasted bytes exceed ~500 MiB (it takes `AccessExclusiveLock`). For online reclaim, prefer `pg_repack` or `pg_squeeze`.

---

## Replication — Physical & Logical

### Physical streaming replication

```
primary:                        WAL writes -> walsender -> network -> walreceiver -> WAL
                                                                                  -> replay (recovery)
```

Lag formulas:

```
write_lag  = pg_wal_lsn_diff(pg_current_wal_lsn(), sent_lsn)
flush_lag  = pg_wal_lsn_diff(sent_lsn, flush_lsn)
replay_lag = pg_wal_lsn_diff(flush_lsn, replay_lsn)
```

From `pg_stat_replication`. Bytes; convert via `pg_size_pretty()`.

```sql
SELECT application_name,
       pg_wal_lsn_diff(pg_current_wal_lsn(), sent_lsn)   AS sent_diff_bytes,
       pg_wal_lsn_diff(sent_lsn, flush_lsn)              AS flush_diff_bytes,
       pg_wal_lsn_diff(flush_lsn, replay_lsn)            AS replay_diff_bytes,
       write_lag, flush_lag, replay_lag
  FROM pg_stat_replication;
```

`*_lag` columns (PG 10+) are the round-trip elapsed time, not bytes — useful for alerting on standby that's *applying* slowly even if catch-up bandwidth is fine (e.g., busy `pg_dump` blocks recovery).

### Synchronous quorum

```
synchronous_standby_names = 'ANY 2 (s1, s2, s3)'         # quorum of any 2
synchronous_standby_names = 'FIRST 2 (s1, s2, s3)'       # priority list
synchronous_standby_names = '2 (s1, s2, s3)'             # legacy = FIRST 2
```

A commit waits until `quorum_size` standbys have acknowledged at the configured `synchronous_commit` level. With `ANY 2` of three standbys, the slowest is allowed to lag. With `FIRST 2`, s1 and s2 are mandatory; s3 is async.

Math: tail-commit latency = `quantile(quorum_size-th fastest standby latency)`. Increasing quorum from 1 to 2 across 3 standbys at p99 = 3 ms each, p99-of-min-of-2 ~= 4 ms (Erlang for k=2,n=3).

```
expected_latency(k, n, mu) ~= mu * sum_{i=n-k+1..n}(1/i)        # harmonic, exponential approx
```

For n=3,k=2,mu=1ms: `1*(1/2 + 1/3) = 0.83 ms` average vs single-standby (k=1) of 0.33 ms.

### Replication slots

Slots persist replication state on the primary, preventing WAL recycling until the consumer catches up.

```sql
SELECT slot_name,
       active,
       restart_lsn,
       pg_wal_lsn_diff(pg_current_wal_lsn(), restart_lsn) AS retained_bytes
  FROM pg_replication_slots;
```

The dangerous corollary: an inactive slot retains WAL **forever**, eventually filling the data drive. `max_slot_wal_keep_size` (PG 13+, default `-1` = unlimited) caps this; once exceeded the slot is invalidated and the consumer must rebuild.

```
slot_retention = pg_current_wal_lsn() - min(restart_lsn over all slots)
                 + active_walsender_position
```

### Logical decoding

`pg_create_logical_replication_slot('s', 'pgoutput')` runs the `pgoutput` decoder on top of WAL. Output is a stream of INSERT/UPDATE/DELETE/COMMIT messages.

Throughput model:

```
decoding_latency  = WAL_buffering + reorder_buffer_serialise + send + filter
reorder_buffer    = max(work_mem_replication, logical_decoding_work_mem)   # PG 13+
spill_threshold   = logical_decoding_work_mem (default 64 MB)
```

A long transaction holds its changes in the reorder buffer until commit; once it exceeds `logical_decoding_work_mem`, PG spills to disk under `pg_replslot/<slot>/xid-*.snap`. Big transactions blow this up; subscribe with `streaming = on` (PG 14+) to receive in-progress xacts and skip the buffer.

Logical replication conflict on subscriber: violated unique key, missing replicated row, type mismatch. The subscriber pauses; you fix manually with `ALTER SUBSCRIPTION ... DISABLE` then `pg_replication_origin_advance`.

```sql
-- on subscriber after a conflict:
SELECT * FROM pg_stat_subscription;
ALTER SUBSCRIPTION sub DISABLE;
SELECT pg_replication_origin_advance('pg_16384', '0/3F4D7A0');
ALTER SUBSCRIPTION sub ENABLE;
```

---

## Lock Modes

### The eight modes

Defined in `src/include/storage/lock.h`:

| Mode | Acquired by |
|:---|:---|
| `AccessShareLock` | SELECT |
| `RowShareLock` | SELECT FOR UPDATE / FOR SHARE |
| `RowExclusiveLock` | INSERT / UPDATE / DELETE |
| `ShareUpdateExclusiveLock` | VACUUM (non-FULL), ANALYZE, CREATE INDEX CONCURRENTLY, ALTER TABLE VALIDATE CONSTRAINT, ALTER TABLE ATTACH/DETACH PARTITION CONCURRENTLY (PG 14) |
| `ShareLock` | CREATE INDEX (non-concurrent) |
| `ShareRowExclusiveLock` | CREATE TRIGGER, some ALTER TABLE forms |
| `ExclusiveLock` | REFRESH MATERIALIZED VIEW CONCURRENTLY |
| `AccessExclusiveLock` | DROP TABLE, TRUNCATE, REINDEX (non-concurrent), CLUSTER, VACUUM FULL, most ALTER TABLE forms, REFRESH MATERIALIZED VIEW (non-concurrent) |

### Compatibility matrix

`X` = conflict (waiter blocks until holder releases). Empty = compatible (both can hold simultaneously).

| Holder \\ Requester | AccessShare | RowShare | RowExcl | ShareUpdExcl | Share | ShareRowExcl | Exclusive | AccessExcl |
|:---|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| **AccessShare**     |   |   |   |   |   |   |   | X |
| **RowShare**        |   |   |   |   |   |   | X | X |
| **RowExcl**         |   |   |   |   | X | X | X | X |
| **ShareUpdExcl**    |   |   |   | X | X | X | X | X |
| **Share**           |   |   | X | X |   | X | X | X |
| **ShareRowExcl**    |   |   | X | X | X | X | X | X |
| **Exclusive**       |   | X | X | X | X | X | X | X |
| **AccessExcl**      | X | X | X | X | X | X | X | X |

Reading: `RowExclusiveLock` (regular DML) conflicts with `ShareLock` (CREATE INDEX) and stronger. That is why a non-concurrent `CREATE INDEX` blocks all DML; switch to `CREATE INDEX CONCURRENTLY` (`ShareUpdateExclusiveLock`) to run alongside.

`AccessExclusiveLock` blocks even bare `SELECT`. Schema migrations that briefly take it (`ALTER TABLE ... SET DEFAULT`, etc.) freeze the application — read `lock_timeout = '2s'` into your migration scripts.

### Lock queue and wait math

PG keeps an ordered wait queue per lock object. New requesters that don't conflict with **all** holders *and* with all earlier waiters are granted immediately; otherwise they queue behind. This means a long-running `ShareLock` holder plus a queued `AccessExclusiveLock` waiter starves *new* `AccessShareLock` requesters because they conflict with the AccessExclusive waiter — a counter-intuitive "fairness fence".

```sql
SELECT a.pid, a.granted, a.mode, a.relation::regclass, b.pid AS blocked_by
  FROM pg_locks a
  LEFT JOIN pg_locks b
    ON a.relation = b.relation
   AND a.pid <> b.pid
   AND b.granted
   AND NOT a.granted
 WHERE a.relation IS NOT NULL
   AND NOT a.granted;
```

Pair with `pg_stat_activity` for the SQL text.

### Deadlock detection

Algorithm: each lock waiter that has been blocked > `deadlock_timeout` (default 1 s) launches a graph-cycle search across all lockers. If a cycle is found, the youngest transaction in the cycle is aborted with `40P01 deadlock detected`.

```
detection_cost = O(V + E)     V = waiters, E = wait edges
trigger period = deadlock_timeout (default 1 s)
```

Deadlock detection runs only on the *blocked* side (lazy). Tighten `deadlock_timeout` to detect faster; loosen on busy systems where transient short waits should not pay the graph-walk overhead.

Row-level deadlocks: PG serializes via `xmax`/`HEAP_XMAX_LOCK_ONLY`. Two updaters acquiring rows in different orders deadlock identically to the table-level case. Solution: always acquire row locks in a deterministic order (e.g., `ORDER BY id` before `FOR UPDATE`).

### Advisory locks

`pg_advisory_lock(key)`, `pg_try_advisory_lock(key)`. Application-level mutex; not tied to any object. `key` is a single bigint or two int4s.

```sql
SELECT pg_advisory_xact_lock(hashtext('user:42'));   -- released at commit
```

Ideal for cron-style "only one worker at a time" semantics.

---

## Indexing Cost Models — Beyond B-Tree

### GIN (Generalised Inverted Index)

For composite values: `tsvector`, `jsonb`, arrays, trigrams.

```
Index entry  = (key) -> posting list of TIDs
posting_list_format =
    if N_tids <= GIN_TUPLE_HEADER_THRESHOLD (~248 bytes): inline
    else: posting tree (mini B-tree of TIDs)
```

Insert path:

```
fastupdate = on (default)
    INSERT/UPDATE: append to pending list (cheap)
    pending list flushed by:
        - reaching gin_pending_list_limit (default 4 MB)
        - VACUUM
        - explicit pg_gin_clean_pending_list()
fastupdate = off
    INSERT/UPDATE: full GIN walk per token (expensive but consistent latency)
```

Tradeoff: `fastupdate = on` pays for ingestion latency with read-time pending-list scan. Heavy-write tables with infrequent reads keep it on; read-mostly tables turn it off.

Search cost:

```
search_cost ~= tokens_in_query * (log(N_keys) + posting_traversal)
             + pending_list_scan
```

GIN does not return tuples in any heap order; planner always wraps it in a Bitmap Heap Scan.

### GiST (Generalised Search Tree)

Lossy / overlapping bounding boxes. Used for geometry, ranges, kNN nearest-neighbour.

```
node_payload = predicate (e.g., bounding box) + child pointers
search_cost  ~= visited_pages * random_page_cost
visited_pages depends on overlap-density of predicates
```

Quality knob: `picksplit` algorithm. Each opclass implements one — `gist_box_picksplit` for boxes, `gist_range_picksplit` for ranges. A bad picksplit causes overlapping siblings, exploding visited pages.

`buffering = auto | on | off` controls bulk-load buffering. `on` cuts CREATE INDEX time on huge tables.

`KNN-GiST`: `ORDER BY pt <-> '(0,0)'::point LIMIT 5` walks the tree priority-queue style, returning sorted nearest matches. Requires opclass support (`KNN gist`).

### BRIN (Block Range INdex)

```
pages_per_range = 128  (default)
ranges          = ceil(relpages / pages_per_range)
size_per_range  = ~4 + size_of_summary
```

Each range stores `(min, max)` (or other summary). Search:

```
SELECT ... WHERE x BETWEEN a AND b
1. Walk BRIN — produce candidate range list O(ranges)
2. Bitmap Heap Scan on candidate ranges
3. Recheck per row
```

Sweet spot: monotonically inserted timestamp/serial columns. 1 TB table indexed on `ts` with `pages_per_range = 128`: 1 TB / (128 * 8 KiB) ~= 1 M ranges, ~16 MB index. A 1-day query touches 1/30 of ranges -> ~33 K pages scanned vs full-table 130 M.

`autosummarize = on` lazily summarises new ranges; otherwise stale BRIN excludes nothing (returns whole table).

### Hash index (PG 10+, fully WAL-logged)

```
buckets         = 4 (initial), doubles on overflow
hash_function   = type-specific (e.g., hashint8)
load_factor     = tuples / buckets
target_fill     = 0.75
```

Lookup: `hashfunc(key) -> bucket -> linear scan of bucket pages`. O(1) expected, O(N) worst case under collision.

Bucket split when `load_factor > target_fill`:

```
new_buckets = old_buckets * 2
rehash_keys = ~half of one bucket (incremental)
```

Use cases: equality-only lookups on huge tables where the B-tree's `O(log N)` is the bottleneck. Loses to B-tree for range queries (cannot help) and ordering (returns unsorted).

### SP-GiST (Space-Partitioned GiST)

Non-balanced trees: quad-tree, k-d tree, prefix tree, radix tree. Search cost depends on data distribution.

```
text_pattern_ops_spgist:
    insert_cost ~= O(L)        # L = key length
    search_cost ~= O(L)
```

Excellent for high-cardinality string prefix lookups (`text LIKE 'foo%'`), IPv4/IPv6 CIDR with `inet_ops`, and geometric data.

### Index type selection cheat

| Need | Pick |
|:---|:---|
| Equality + range on scalar | B-tree |
| Equality only on huge tables | Hash |
| `@>`, `<@`, full-text, jsonb keys | GIN |
| Geometry / range overlap / kNN | GiST |
| Append-mostly time-series | BRIN |
| Prefix / radix / quad | SP-GiST |

---

## TOAST — Large Value Storage

### Thresholds

```
TOAST_TUPLE_THRESHOLD = 2032 bytes (BLCKSZ / 4 - 32)         # row size that triggers TOAST attempts
TOAST_TUPLE_TARGET    = 2032 bytes                            # row size aimed for after TOAST
TOAST_MAX_CHUNK_SIZE  = 1996 bytes                            # one TOAST chunk
```

When a row exceeds `TOAST_TUPLE_THRESHOLD`, the row is rewritten with one or more attributes pushed out-of-line:

1. Try compressing each attribute marked `EXTENDED` or `MAIN`.
2. If still too big, store EXTENDED attributes out-of-line in `pg_toast_<oid>` (a separate table with `(chunk_id, chunk_seq, chunk_data)` rows).
3. If still too big after all EXTENDEDs, do the same with MAIN attributes.

Attribute storage strategies (set per column with `ALTER TABLE ... SET STORAGE`):

| Strategy | Inline OK? | Compress? | Out-of-line? |
|:---|:---:|:---:|:---:|
| `PLAIN`   | yes (forced inline) | no | no |
| `EXTERNAL`| optional | no | yes |
| `EXTENDED`| optional | yes | yes (default for varlen) |
| `MAIN`    | preferred | yes | last resort |

### Compression

```
default_toast_compression = 'pglz' | 'lz4'   # PG 14+, default 'pglz'
```

| Algorithm | Ratio (typical text) | Compress speed | Decompress speed |
|:---|:---:|:---:|:---:|
| pglz   | ~2.0x | ~150 MB/s | ~400 MB/s |
| lz4    | ~1.8x | ~600 MB/s | ~3000 MB/s |
| zstd (PG 16+ via column) | ~3.0x | ~400 MB/s | ~1500 MB/s |

PGLZ refuses to compress when expected ratio < 25%. Pre-compressed blobs (gzip files, jpeg) skip TOAST compression and just go EXTERNAL. The 25%-rule means random-looking strings (UUIDs, base64) will typically be EXTERNAL-uncompressed.

### Cost implications

- Reading TOASTed columns: extra random heap pages in `pg_toast_<oid>`. Plan an index-only scan that does NOT include the TOASTed column to avoid the de-toast.
- Updating rows that don't touch TOAST: cheap (HOT update; TOAST chunks unchanged).
- Updating TOAST: full re-TOAST of the new value, old chunks marked dead, vacuum reclaims later.

`pg_total_relation_size('foo')` includes TOAST; `pg_relation_size('foo')` does not.

```sql
SELECT pg_size_pretty(pg_total_relation_size('events'))    AS total,
       pg_size_pretty(pg_relation_size('events'))          AS heap,
       pg_size_pretty(pg_total_relation_size(reltoastrelid)) AS toast
  FROM pg_class WHERE relname = 'events';
```

---

## Configuration Math

### shared_buffers sizing

The 25%-of-RAM rule comes from two competing facts:

1. PG buffer pool uses CLOCK-sweep eviction; OS page cache uses LRU. For the same memory partitioned 50/50 between PG and OS cache, PG's smarter cache wins on hit rate.
2. Doubling `shared_buffers` does NOT double cache (OS cache shrinks correspondingly).

Empirically:

```
recommended_shared_buffers = min(0.25 * RAM, 16 GB)         # default rule
                            up to 0.40 * RAM on PG 14+ with huge pages
```

Beyond about 16 GB, PG's `BufFreelistLock` and dirty-buffer writeout become bottlenecks. Modern installs combine huge pages (`huge_pages = try`, then `vm.nr_hugepages` on Linux) with 25-40% to neutralise TLB pressure.

Buffer pool hit ratio:

```
hit_ratio = 1 - blks_read / blks_hit                    # from pg_stat_database
```

Aim above 99% for OLTP. Sub-95% means working set spills out of RAM.

### work_mem multiplier

```
peak_work_mem = max_connections * work_mem * concurrent_sort_or_hash_per_query
```

A conservative pessimistic ceiling. Each query can have multiple hash/sort/CTE operators each allocating up to `work_mem` independently — the multiplier `concurrent_sort_or_hash_per_query` is often 2-4 for analytic queries.

```
max_connections = 200, work_mem = 64 MB, ops/query = 3
peak_work_mem   = 200 * 64 MB * 3 = 38.4 GB
```

If physical RAM is 32 GB and you've already committed 8 GB to `shared_buffers`, this peak triggers swap-storms during burst load. Either drop `work_mem` to per-query expectation × 3 + safety, or use a connection pooler so `max_connections` reflects real concurrency, not client count.

PG 13+ adds `hash_mem_multiplier` (default 2.0): hash operations get `work_mem * hash_mem_multiplier`. The reasoning is hashes really need more memory than sorts; raise this to 4-8 on analytic boxes.

### effective_cache_size

```
effective_cache_size = shared_buffers + OS_page_cache_estimate
                     = shared_buffers + (free + buff/cache) seen by `top`
```

Used **only** by the planner for index-vs-seq decisions; not allocated. Lower it and the planner reluctantly picks more seq scans. Most defaults (4 GB) are too small on modern hardware; set to `0.5 * total_RAM` minimum.

### max_connections sizing

Each backend process consumes about 10 MB RSS minimum (more with pl/perl, more with replicaslot, more with cached plans). 1000 connections = 10 GB just for processes. Add `work_mem * connections * ops_per_query` and you're out of RAM at about 500 connections.

Standard pattern: `max_connections = 100-300`, with pgbouncer in `transaction` mode in front. `max_connections` also caps replication slots (`max_wal_senders + max_replication_slots <= max_connections - superuser_reserved`).

### A reality-check checklist

```
shared_buffers          = 0.25 * RAM         (capped 16 GB unless huge_pages)
effective_cache_size    = 0.5  * RAM
maintenance_work_mem    = 0.05 * RAM         (cap 2 GB; per-autovac-worker!)
work_mem                = 4..64 MB           (most workloads)
hash_mem_multiplier     = 2..8
max_wal_size            = 4 GB+              (raise for write-heavy)
checkpoint_timeout      = 15..30 min
random_page_cost        = 1.1..1.5 on SSD
effective_io_concurrency = 200..400 on NVMe
```

### Summary table

| Knob | Function | Typical range |
|:---|:---|:---|
| `shared_buffers` | buffer pool size | 0.25 * RAM |
| `effective_cache_size` | planner cache hint | 0.50 * RAM |
| `work_mem` | per-op sort/hash | 4-64 MB |
| `maintenance_work_mem` | vacuum / index build | 256 MB - 2 GB |
| `random_page_cost` | seek cost | 1.1 (NVMe) - 4 (HDD) |
| `seq_page_cost` | sequential cost | 1.0 |
| `cpu_tuple_cost` | per-tuple CPU | 0.01 |
| `max_wal_size` | checkpoint trigger | 4-64 GB |
| `checkpoint_timeout` | checkpoint interval | 15-30 min |
| `wal_buffers` | WAL ring | -1 (auto) |
| `max_connections` | backend cap | 100-300 |
| `autovacuum_vacuum_scale_factor` | dead-tup % | 0.01-0.2 |
| `vacuum_cost_limit` | vacuum throttle | 200-10000 |

---

## Performance Worked Examples

### Example 1 — Index-only scan vs heap scan break-even

Table `users(id PK, email, country, signup_ts)`, 50 M rows, 200-byte avg row, `relpages ~= 1.6 M`, `relallvisible ~= 1.59 M` (post-VACUUM). Index `idx (country) INCLUDE (id, signup_ts)` exists, leaf pages 130 K.

```sql
SELECT id, signup_ts FROM users WHERE country = 'NZ';
```

NZ-selectivity = 0.005 (250 K rows).

**Index-only scan**:

```
all_visible_fraction = 1.59M / 1.6M = 0.994
index_pages          = ceil(0.005 * 130000) = 650
heap_pages_recheck   = (1 - 0.994) * 250000 = 1500   # for the 0.6% non-visible pages
cost = 650 * 4.0 + 250000 * 0.005 + 250000 * 0.0025
     + 1500 * 4.0
     = 2600 + 1250 + 625 + 6000 = 10475
```

**Bitmap heap scan** (rebuilds via index but reads heap):

```
bitmap_index_cost = 650 * 4 + 250000 * 0.0075 = 2600 + 1875 = 4475
heap_pages        = ~50000 (NZ users randomly distributed)
density           = 50000/1600000 = 0.031
cost_per_page     = lerp(1.0, 4.0, 0.031) = 1.09
heap_cost         = 50000 * 1.09 + 250000 * 0.0125 = 54500 + 3125 = 57625
total             = 62100
```

Index-only scan wins by 6x — but only because the table was recently vacuumed. Run a flood of `UPDATE`s, `relallvisible` drops to 1.2 M, `all_visible_fraction = 0.75`, and the index-only recheck cost climbs:

```
heap_pages_recheck = 0.25 * 250000 = 62500
recheck_cost       = 62500 * 4 = 250000
total_index_only   = 4475 + 1875 + 625 + 250000 = 256975
```

Now bitmap heap wins. Lesson: index-only scan eligibility is fragile post-write; tune autovacuum to keep VM dense.

### Example 2 — HOT update chain length

Table `orders(id PK, status, updated_at)`, indexes on `id` and `status`.

Update pattern: set `updated_at = now()` on each row a few times per hour. `updated_at` is **not** indexed, but `status` is.

If the update only touches `updated_at`: HOT eligible. Chain grows in place. Vacuum prunes when reading.

If the update touches `status`: not HOT. Each update writes a new tuple AND new index entries on `status` and `id`. After 10 status changes, one row has 11 versions visible until vacuum, plus 11 index entries on each index. Bloat math:

```
without_HOT_index_writes = updates * num_indexes
write_amplification_HOT  = 1
write_amplification_full = 1 + num_indexes
```

Adding an index turns a HOT-friendly schema into a write-amplifier. Audit `pg_stat_all_tables.n_tup_hot_upd / n_tup_upd` per table:

```sql
SELECT relname,
       n_tup_upd,
       n_tup_hot_upd,
       100.0 * n_tup_hot_upd / NULLIF(n_tup_upd,0) AS hot_pct
  FROM pg_stat_user_tables
 ORDER BY n_tup_upd DESC;
```

Aim for `hot_pct > 80` on UPDATE-heavy tables.

### Example 3 — Parallel query worker count

```
max_parallel_workers_per_gather = 2 (default)
max_parallel_workers            = 8 (default)
max_worker_processes            = 8 (default)
parallel_setup_cost             = 1000
parallel_tuple_cost             = 0.1
```

A SeqScan over a 50 GB table:

```
serial_seq_cost   = 6.5e6 * seq_page_cost = 6.5M
parallel_seq_cost = serial_seq_cost / N_workers
                    + parallel_setup_cost
                    + tuples_returned * parallel_tuple_cost
```

For N=2:

```
parallel = 6.5M / 2 + 1000 + 100K * 0.1
         = 3.25M + 1000 + 10K = 3.26M       # half the time
```

For N=4 (raise both `max_parallel_workers_per_gather` and `max_parallel_workers`):

```
parallel = 6.5M/4 + 1000 + 10K = 1.63M + 11K = 1.64M
```

Diminishing returns set in once `parallel_setup_cost + tuple_shipping_cost ~= serial_seq_cost / N`. Past that, more workers = more overhead, less throughput.

```
optimal_N = sqrt(serial_cost / (parallel_setup_cost + tuples * parallel_tuple_cost))
```

### Example 4 — Partition pruning savings

Range-partitioned `events_p` by `ts` (monthly), 24 partitions, about 500 GB.

Without pruning:

```sql
SELECT count(*) FROM events_p WHERE ts >= '2026-04-01';
```

EXPLAIN shows 24 seq-scan branches -> cost about 24 * 250K = 6 M.

With pruning, the planner restricts to the 1 relevant partition:

```sql
SELECT count(*) FROM events_p WHERE ts >= '2026-04-01' AND ts < '2026-05-01';
```

Cost about 250 K (single partition). Pruning factor:

```
saving = num_partitions_pruned / total_partitions
       = 23 / 24 ~= 0.96
```

Caveats:

- `enable_partition_pruning = on` (default).
- Pruning at *plan time* requires constants in the predicate.
- Pruning at *execution time* (PG 11+) handles `$1`-style parameters and joins.
- Pruning fails on functions that aren't proven STABLE/IMMUTABLE.

```sql
EXPLAIN (ANALYZE, COSTS OFF) SELECT count(*) FROM events_p WHERE ts > now() - interval '1 day';
```

`Partitions: 1 of 24`  vs `Partitions: 24 of 24` — verify before assuming.

### Example 5 — Plan cache invalidation cascade

Prepared statement `stmt = SELECT * FROM users WHERE country = $1`. PG keeps a generic plan and a custom plan; switches to generic after 5 executions when the heuristic favours it.

```
generic_cost = generic_plan_cost
custom_cost  = avg(custom_plan_cost)
choose_generic_if generic_cost < custom_cost + (custom_planning_cost - generic_planning_cost)
```

`plan_cache_mode = auto | force_custom_plan | force_generic_plan`. Force custom when one input value (`'NZ'`) has 0.5% selectivity but another (`'US'`) has 60% — the generic plan's cost average misleads.

Now imagine `ALTER TABLE users ADD COLUMN ...` — every prepared plan referencing `users` is invalidated; next exec re-plans. A schema migration on a hot OLTP table can produce a thundering herd of re-plans, briefly spiking CPU. Mitigations:

- Migrate during off-peak.
- Use `PREPARE ... AS ... ` with `DISCARD ALL` cycle scheduling.
- Monitor `pg_stat_activity.state = 'active'` for re-plan storms.

### Example 6 — Buffer hit ratio under cold start

```
shared_buffers     = 16 GB
working_set        = 30 GB
hit_ratio_steady   = 16 / 30 = 0.53                    # fixed-distribution case
hit_ratio_zipf     = ~0.95                              # 80/20 access pattern
```

After restart, hit ratio starts at 0%. Time to warm = `working_set_active / read_bandwidth`. On a 1 GB/s NVMe with a 30 GB hot set, warm-up takes 30 seconds. Use `pg_prewarm` to skip the warmup:

```sql
CREATE EXTENSION pg_prewarm;
SELECT pg_prewarm('events', 'buffer');
```

### Example 7 — Connection pooler savings

Without pooler:

```
client_connections = 1000
max_connections    = 1000
mem_per_backend    = 12 MB
total_backend_mem  = 1000 * 12 MB = 12 GB
work_mem * connections = 1000 * 16 MB = 16 GB peak
```

With pgbouncer in transaction mode, `pool_size = 50`:

```
client_connections = 1000              # logical
backend_processes  = 50                # physical
backend_mem        = 50 * 12 MB = 600 MB
work_mem peak      = 50 * 16 MB = 800 MB
xid generation     ≈ 50 concurrent xacts at most
snapshot xip cost  = O(log 50) instead of O(log 1000)
```

Latency cost: tx-mode pooler imposes ~0.2 ms per pool checkout + ban on session-scoped state.

---

## Prerequisites

- SQL fundamentals (queries, joins, CTEs, prepared statements)
- B-tree and hash index data structure intuition
- MVCC concept (snapshot isolation, transaction IDs, vacuum)
- Cost-based query optimisation principles
- Buffer cache / OS page cache distinction
- Basic statistics (histograms, selectivity estimation, ndistinct)

## Complexity

- **Beginner**: read EXPLAIN output, identify seq scan vs index scan, know what `VACUUM ANALYZE` does
- **Intermediate**: tune `shared_buffers`, `work_mem`, `random_page_cost`; interpret `pg_stat_user_tables`; set per-table autovacuum overrides
- **Advanced**: derive cost-tuple math for multi-index plans, read MVCC source via `pageinspect`, model checkpoint I/O budgets, design partition pruning for sub-second OLAP, debug logical replication lag and reorder buffer spills, audit lock-queue starvation, plan anti-wraparound campaigns

---

## See Also

- `databases/postgresql` — operational cheat sheet, `psql` flags, common admin SQL
- `databases/sql` — SQL grammar reference (DDL/DML/DQL/DCL/TCL)
- `databases/redis` — comparison: in-memory data-structure store vs persistent OLTP
- `cs-theory/distributed-systems` — CAP, PACELC, consensus framing for replicated PG
- `cs-theory/distributed-consensus` — Raft and Paxos algorithms used by Patroni/etcd-backed HA
- `ramp-up/postgres-eli5` — narrative onramp ramp-up sheet

---

## References

- PostgreSQL Documentation — https://www.postgresql.org/docs/current/
- "PostgreSQL 17 Internals", Egor Rogov — https://postgrespro.com/community/books/internals
- "PostgreSQL: Up and Running", Regina Obe & Leo Hsu — O'Reilly
- "The Internals of PostgreSQL", Hironobu Suzuki — https://www.interdb.jp/pg/
- WAL & MVCC source: `src/backend/access/transam/`, `src/backend/storage/buffer/`, `src/backend/access/heap/heapam_visibility.c`
- Cost model source: `src/backend/optimizer/path/costsize.c`
- Lock manager source: `src/backend/storage/lmgr/lock.c`, `src/include/storage/lock.h`
- Vacuum source: `src/backend/access/heap/vacuumlazy.c`, `src/backend/postmaster/autovacuum.c`
- B-tree source: `src/backend/access/nbtree/`
- TOAST source: `src/backend/access/common/toast_internals.c`
- Logical decoding source: `src/backend/replication/logical/`
- PG release notes 14, 15, 16, 17 — track per-release planner/vacuum changes
- `pg_stat_progress_vacuum` documentation — https://www.postgresql.org/docs/current/progress-reporting.html
- `EXPLAIN` documentation — https://www.postgresql.org/docs/current/using-explain.html
- "GiST: A Generalized Search Tree for Database Systems", Hellerstein, Naughton, Pfeffer — VLDB 1995
- "Generalized Inverted Indexes", Bartunov & Sigaev — PGCon 2006
- "Postgres Indexes Under the Hood", Bruce Momjian — slides at momjian.us
- "Database Management Systems" 3rd ed., Ramakrishnan & Gehrke — hash/B-tree complexity proofs
- IETF RFC 8259 (JSON) — referenced by `jsonb` storage decisions
