# The Internals of SQL — Query Planning, Indexes, Joins, and Isolation

> *SQL is a declarative language: you describe what you want, the database decides how to compute it. Behind every query is a parser, an analyzer, a rewriter, a cost-based optimizer, and an executor. Behind every fast query is a B+tree, a hash table, or a sort. Behind every transaction is a write-ahead log, a multi-version concurrency control protocol, and a lock manager. This document walks through those internals — the "why" behind everything that makes SQL fast or slow.*

---

## 1. SQL Query Lifecycle

Every SQL statement, regardless of vendor, traverses the same canonical pipeline. Understanding each stage demystifies why a query runs the way it does — and why two superficially identical queries can produce wildly different plans.

### 1.1 The Five-Stage Pipeline

```
SQL text
  │
  ▼
[1] Parser          → Abstract Syntax Tree (AST)
  │
  ▼
[2] Analyzer        → Validated, name-resolved tree
  │                   (catalog lookups, type checks)
  ▼
[3] Rewriter        → Logical query tree (relational algebra)
  │                   (view expansion, rule rewrites)
  ▼
[4] Planner/Optimizer → Physical plan tree
  │                     (cost-based or rule-based)
  ▼
[5] Executor        → Tuples streamed to client
```

### 1.2 Stage 1 — Parser

The parser is a hand-written or generator-built (Bison/Yacc/ANTLR) recursive-descent or LALR(1) parser. It accepts the SQL grammar and produces an AST. PostgreSQL uses Bison, MySQL uses a hand-written recursive-descent parser, SQLite uses Lemon (a Bison clone written by D. Richard Hipp).

Failures here are *syntax errors* — the parser cannot recover; the query is rejected before any catalog lookup.

```sql
-- This dies in the parser; "FORM" is not a keyword
SELECT id FORM users;

-- ERROR:  syntax error at or near "users"
```

The AST mirrors the SQL grammar: nodes for `SelectStmt`, `RangeVar`, `ColumnRef`, `A_Expr`, etc. It holds *names*, not OIDs — the parser does not yet know what `users` refers to.

### 1.3 Stage 2 — Analyzer

The analyzer (Postgres calls it the "transform" stage) walks the AST, performs name resolution against the system catalog, type-checks expressions, expands `*` projections, resolves implicit casts, and produces a `Query` tree where every reference is bound to its catalog OID.

This is where errors like `column "frob" does not exist` surface. It is also where ambiguous references (`SELECT id FROM a, b WHERE id = 1`) are caught.

### 1.4 Stage 3 — Rewriter

The rewriter applies *rule-based* transformations: view expansion, row-level security predicate injection, materialized-view substitution, and subquery flattening.

```sql
-- View definition
CREATE VIEW active_users AS SELECT * FROM users WHERE active = true;

-- User query
SELECT id FROM active_users WHERE region = 'us';

-- After rewriting: the view body is inlined
SELECT id FROM users WHERE active = true AND region = 'us';
```

Rewriter rules are pattern-match transformations. They never compute cost; they always fire when their pattern matches.

### 1.5 Stage 4 — Planner / Optimizer

This is where the magic happens. The planner takes the rewritten logical tree (a relational-algebra expression: $\sigma_{p}(\pi_{c}(R \bowtie S))$) and searches for the cheapest *physical* plan that computes the same result.

Two paradigms exist:

- **Rule-based optimizer (RBO):** Apply transformation rules in a fixed priority order. Used by Oracle pre-10g and some embedded engines. Predictable but suboptimal — cannot reason about data distribution.
- **Cost-based optimizer (CBO):** Enumerate candidate plans, estimate the cost of each using table statistics, choose the cheapest. Used by PostgreSQL, MySQL 5.7+, SQL Server, Oracle 10g+, and every serious modern RDBMS.

The CBO output is a *physical plan tree*: leaves are scans (Seq Scan, Index Scan), internal nodes are operators (Hash Join, Sort, Aggregate), the root produces the final result.

### 1.6 Stage 5 — Executor

The executor walks the physical plan in a *Volcano model*: each operator implements `open()`, `next()`, `close()`. The root pulls tuples from its child, which pulls from its child, recursively. This produces a streaming, pipelined execution: a row can flow from disk to client without ever materializing the full intermediate result.

Some operators are *blocking*: Sort and HashAggregate must consume all input before producing output. Others are *streaming*: Seq Scan, Index Scan, Nested Loop.

### 1.7 The AST → Algebra → Plan Pipeline in Postgres

```bash
# See what Postgres does at each stage
psql -c "SET debug_print_parse = on;"        # AST after parser
psql -c "SET debug_print_rewritten = on;"    # After rewriter
psql -c "SET debug_print_plan = on;"         # Final plan tree
psql -c "SET client_min_messages = log;"     # Surface in psql
```

Most users never inspect these — `EXPLAIN` is the user-facing window into stage 4 + 5. But knowing the pipeline is what lets you reason about *why* a query was rewritten or planned the way it was.

---

## 2. The Cost-Based Optimizer — Statistics

The CBO is only as good as its statistics. Stale or missing stats are the #1 cause of bad plans in production.

### 2.1 What Statistics the Planner Needs

For each table, the planner stores:

| Statistic | Symbol | Meaning |
|:---|:---|:---|
| Row count | `reltuples` | Total live rows in the table |
| Page count | `relpages` | Pages on disk (8 KB in Postgres) |
| Distinct values per column | `n_distinct` | Count of unique non-null values |
| NULL fraction | `null_frac` | Proportion of rows where column is NULL |
| Most-common values (MCV) | `most_common_vals` | Top-N frequent values + their frequencies |
| Histogram | `histogram_bounds` | Equi-height buckets of remaining values |
| Average column width | `avg_width` | Bytes per row for that column |
| Correlation | `correlation` | Physical-vs-logical ordering correlation |

These are stored in `pg_statistic` (Postgres), `mysql.innodb_index_stats`, or analogous catalogs.

### 2.2 The ANALYZE Command

Statistics are not automatic. They are *sampled* — Postgres reads ~30000 rows per table by default, computes the stats, and writes them back to the catalog.

```sql
-- Postgres: refresh stats for one table
ANALYZE users;

-- Refresh stats for one column with higher accuracy
ALTER TABLE users ALTER COLUMN email SET STATISTICS 1000;
ANALYZE users (email);

-- See current stats
SELECT * FROM pg_stats WHERE tablename = 'users';
```

```sql
-- MySQL: refresh stats
ANALYZE TABLE users;

-- See current stats (InnoDB persistent stats)
SELECT * FROM mysql.innodb_table_stats WHERE table_name = 'users';
SELECT * FROM mysql.innodb_index_stats WHERE table_name = 'users';
```

The default sample size is configurable. Larger samples produce better estimates at the cost of longer ANALYZE runtime. For tables with skewed distributions, increasing `default_statistics_target` (Postgres) from 100 to 1000 often fixes plan instability.

### 2.3 Auto-Vacuum and Auto-Analyze (Postgres)

Postgres ships with an `autovacuum` daemon that combines two responsibilities:

- **Vacuum:** reclaim dead tuples (see MVCC, section 14)
- **Analyze:** refresh statistics

The daemon fires when a table has accumulated more than `autovacuum_analyze_threshold + autovacuum_analyze_scale_factor * reltuples` insert/update/delete operations since the last analyze. Defaults: threshold = 50, scale factor = 0.1 → analyze fires at 10% churn.

```sql
-- Tune per-table for hot tables
ALTER TABLE orders SET (
  autovacuum_analyze_scale_factor = 0.02,  -- analyze at 2% churn
  autovacuum_vacuum_scale_factor = 0.05    -- vacuum at 5% churn
);

-- See last analyze times
SELECT relname, last_analyze, last_autoanalyze, n_live_tup, n_dead_tup
  FROM pg_stat_user_tables;
```

### 2.4 What Happens When Stats Are Stale

The classic disaster: a bulk insert loads 10 million rows. The planner still thinks the table has 1000 rows. It picks a Nested Loop join with the now-huge table on the inner side. The query that should take 50 ms now takes 50 minutes.

```sql
-- Symptom: planner expects 1000 rows, executor returns 10000000
EXPLAIN ANALYZE SELECT * FROM orders WHERE created > NOW() - INTERVAL '1 day';
-- Seq Scan on orders  (cost=0.00..15.00 rows=1000 width=64)
--                                          ^^^^ wildly wrong
--                     (actual time=0.012..4521.293 rows=10000000)
--                                                    ^^^^^^^^ truth
```

The fix is always: `ANALYZE table_name;`. The lesson: after any bulk load, ETL job, or restore, run ANALYZE before you let production queries hit it.

---

## 3. Cardinality Estimation — Algorithms

Given the statistics, how does the planner *use* them to predict how many rows a `WHERE` clause will return? This is *cardinality estimation*, and it is the hardest problem in query optimization.

### 3.1 Selectivity

The *selectivity* of a predicate is the fraction of rows that satisfy it. If a table has 1 million rows and the predicate matches 1000, selectivity is 0.001.

The planner combines selectivity with row count to predict output cardinality:

$$\text{rows out} = \text{rows in} \times \text{selectivity}$$

### 3.2 Equi-Height Histogram

The most common shape. The histogram divides the value range into N buckets (default 100 in Postgres) such that *each bucket contains roughly the same number of rows*. The bucket *boundaries* are stored.

For a range predicate `col > x`:
- Find the bucket containing `x`
- Estimate the fraction of `x`'s bucket above `x` (linear interpolation within bucket)
- Add 1.0 for every bucket strictly above

```
column values:    1, 2, 3, 5, 8, 13, 21, 34, 55, 89  (10 rows)
3-bucket equi-height:
  bucket 1: [1, 3]   (rows 1, 2, 3)
  bucket 2: [5, 13]  (rows 5, 8, 13)
  bucket 3: [21, 89] (rows 21, 34, 55, 89)

WHERE col > 10:
  bucket 2 portion: (13 - 10) / (13 - 5) = 0.375 of bucket 2
  bucket 3:         1.0 of bucket 3
  total selectivity: (0.375 + 1.0) / 3 = 0.458
  predicted rows: 10 * 0.458 = 4.58 (actual: 4)
```

### 3.3 Equi-Width Histogram

Buckets cover equal *value ranges* but contain unequal row counts. Stores per-bucket frequency. Less common but cheaper to maintain. Fine for uniform distributions, terrible for skewed ones.

### 3.4 Most-Common-Values (MCV) List

Equi-height histograms badly mis-estimate predicates on highly-frequent values (the value sits in one tiny portion of one bucket but actually appears thousands of times). The fix: store the top-N frequent values *separately*, with their exact frequencies.

```sql
-- Postgres pg_stats inspection
SELECT attname, n_distinct, most_common_vals, most_common_freqs
  FROM pg_stats WHERE tablename = 'orders' AND attname = 'status';
-- attname:           status
-- n_distinct:        5
-- most_common_vals:  {pending, shipped, delivered, cancelled, returned}
-- most_common_freqs: {0.4, 0.3, 0.2, 0.07, 0.03}
```

For `WHERE status = 'pending'`, the planner uses MCV directly: selectivity = 0.4. No histogram involved.

For `WHERE status NOT IN ('pending', 'shipped')`: selectivity = 1 - 0.4 - 0.3 = 0.3.

### 3.5 NULL Fraction

NULLs are tracked separately. `WHERE col IS NULL` selectivity = `null_frac`. `WHERE col = x` selectivity = `(1 - null_frac) * mcv_freq` (or histogram-based).

### 3.6 Conjunction Selectivity — the Independence Assumption

For `WHERE A = a AND B = b`, the classical formula is:

$$S_{A \land B} = S_A \times S_B$$

This assumes A and B are *independent*. They almost never are in practice. Consider:

```sql
-- city='London' selectivity ~ 0.001 (1 city of 1000)
-- country='UK' selectivity ~ 0.05 (1 country of 20)
-- predicted: 0.001 * 0.05 = 0.00005 (5 rows in 100K)
-- reality:   every London row IS a UK row, so 0.001 (100 rows)
SELECT * FROM addresses WHERE city = 'London' AND country = 'UK';
```

The planner under-predicts cardinality by 20x. It then picks a Nested Loop assuming the inner side will be tiny, which it isn't.

### 3.7 Multivariate Statistics (Postgres)

Postgres 10+ added `CREATE STATISTICS` to track correlations:

```sql
-- Tell the planner that city and country are correlated
CREATE STATISTICS addr_city_country (dependencies, ndistinct, mcv)
  ON city, country FROM addresses;

ANALYZE addresses;

-- Now the planner uses joint distribution stats, not the independence assumption
```

The three kinds:

- **dependencies:** functional-dependency strength (0.0 = independent, 1.0 = perfect correlation)
- **ndistinct:** count of distinct (city, country) pairs (vs. product of distincts)
- **mcv:** joint MCV list — top-N (city, country) tuples with frequencies

This is how you fix the London-UK problem at the planner level.

### 3.8 Disjunction Selectivity

For `WHERE A = a OR B = b`, the inclusion-exclusion formula:

$$S_{A \lor B} = S_A + S_B - S_A \times S_B$$

Subject to the same independence-assumption caveats.

---

## 4. Index Internals — B-tree

The B-tree (more precisely, the B+tree variant used by all major RDBMSs) is the workhorse index structure. Understanding its shape clarifies why some queries are fast and some aren't.

### 4.1 The B+tree Shape

```
                  [ 50 | 100 ]                    ← root (internal)
                 /     |     \
          [10|30]   [60|80]   [120|150]           ← internal level
         /  |  \   /  |  \    /   |   \
       ...  ...  ...    leaf-level pages
       │←─────── linked list ─────────→│          ← sibling pointers
```

Three properties define a B+tree:
1. **All data lives in leaf pages.** Internal pages only store *separator keys* that route searches.
2. **Leaves are linked.** A doubly-linked list of leaves enables in-order scans without re-traversing the tree.
3. **Self-balancing.** All leaves are at the same depth. Splits propagate upward.

Branching factor is typically 100–500 keys per page. With 8 KB pages and ~32-byte keys, you fit ~250 keys per page. A 3-level tree holds 250³ = 15 million keys; 4-level holds 4 billion. **Almost any real-world B+tree is 3 or 4 levels deep.**

### 4.2 Lookup Cost

Lookup descends from root to leaf: O(log_b N) where b is the branching factor. With b=250, log_250(1B) ≈ 3.7. So a billion-row lookup is ~4 page reads. Each page read is ~10 µs (SSD) to ~5 ms (HDD).

### 4.3 Range Scan Cost

Range scans use the leaf-level linked list. Cost is O(log N) to find the start, plus O(pages) to walk forward. A page holds ~250 entries, so reading 10000 contiguous rows is ~40 page reads.

### 4.4 Page-Based Structure

Postgres pages are 8 KB. Each page holds:
- A 24-byte header
- An array of `ItemId` slots (4 bytes each), grouped at the page start
- Tuples themselves, packed from the page end backward
- Free space in between

This *slot-and-tuple* layout supports variable-length rows and allows `vacuum` to compact the page.

### 4.5 WAL-Protected Updates

Every page modification is logged to the Write-Ahead Log *before* the page is dirtied:

```
1. Read page into shared buffers (if not cached)
2. Lock the page
3. Generate WAL record describing the change
4. Apply change to in-memory page
5. Write WAL record to wal_buffers
6. Unlock page; dirty bit set
7. fsync(WAL) on commit
8. Background writer/checkpointer eventually flushes dirty page to disk
```

If the system crashes between steps 7 and 8, recovery replays the WAL and reconstructs the page state.

### 4.6 Clustered vs Non-Clustered (Secondary) Indexes

- **Clustered (InnoDB primary key):** the leaf pages *are* the table. Looking up by primary key gives you the row immediately. Secondary indexes store the primary key as the row pointer, requiring a second lookup.
- **Non-clustered (Postgres default, all secondary indexes):** the leaf stores a `(page, offset)` pointer to the heap. The heap is a separate file. Index lookups always require a second I/O to fetch the row.

### 4.7 Index-Only Scans and the Visibility Map

When an index *covers* every column the query needs and the planner can prove the rows are visible to the current transaction, Postgres skips the heap lookup entirely. This requires the *visibility map* — a per-page bit indicating whether all tuples on the page are visible to all current transactions.

```sql
-- Index that covers (id, name): query reads only the index
CREATE INDEX users_id_name_idx ON users (id, name);

EXPLAIN SELECT name FROM users WHERE id = 42;
--  Index Only Scan using users_id_name_idx on users
--    Heap Fetches: 0   ← zero heap reads when visibility map is good
```

`VACUUM` updates the visibility map; on a write-heavy table where vacuum lags, you may see `Heap Fetches: 1234` defeating the purpose. `VACUUM (INDEX_CLEANUP ON, ANALYZE) users;` keeps it sharp.

### 4.8 The Key Costs at a Glance

| Operation | Cost | Notes |
|:---|:---|:---|
| Equality lookup `col = x` | O(log N) ≈ 4 page reads | Plus 1 heap I/O if not index-only |
| Range scan `col BETWEEN a AND b` | O(log N + range/page_size) | Leaf list walk after descent |
| Sort by indexed col | O(N) | Scan in order, no sort |
| Insert | O(log N) | Plus possible page split |
| Delete | O(log N) | Plus possible page merge |
| Update of indexed col | O(log N) for delete + insert | Worst case for hot indexes |

### 4.9 Multi-Column (Composite) Indexes

A composite index `ON users (a, b, c)` is *ordered first by a, then by b within a, then by c within b*. The leftmost-prefix rule applies:

| Query predicate | Index usable? |
|:---|:---:|
| `a = ?` | Yes |
| `a = ? AND b = ?` | Yes |
| `a = ? AND b = ? AND c = ?` | Yes |
| `a = ? AND c = ?` | Partial (filter on c after seek) |
| `b = ?` | No (no leftmost) |
| `b = ? AND c = ?` | No |

This is why the column order in composite indexes matters more than newcomers expect.

---

## 5. Index Internals — Hash, GIN, GiST, BRIN, SP-GiST

Beyond B-trees, Postgres offers five other index access methods. Each shines for a specific access pattern.

### 5.1 Hash Index

A hash table on disk. Equality only — no range queries, no ordering. Pre-Postgres-10 hash indexes were not WAL-logged and effectively useless; modern hash indexes are crash-safe.

```sql
CREATE INDEX users_email_hash ON users USING hash (email);
SELECT * FROM users WHERE email = 'alice@example.com';
```

Smaller than B-tree on wide keys (stores hash, not full key). Slightly faster on pure equality. But B-tree is so good that hash is rarely chosen.

### 5.2 GIN — Generalized Inverted Index

For values that are *composite* — arrays, jsonb, full-text vectors — where you want to query containment.

```sql
-- Full-text search
CREATE INDEX docs_body_gin ON docs USING gin (to_tsvector('english', body));
SELECT * FROM docs WHERE to_tsvector('english', body) @@ to_tsquery('coffee & shop');

-- jsonb containment
CREATE INDEX events_data_gin ON events USING gin (data jsonb_path_ops);
SELECT * FROM events WHERE data @> '{"type": "purchase"}';

-- Array containment
CREATE INDEX posts_tags_gin ON posts USING gin (tags);
SELECT * FROM posts WHERE tags @> ARRAY['postgres', 'sql'];
```

Internal structure: a B-tree of *keys* (each token in the document, each top-level jsonb key, each array element), with a posting list per key — the row IDs that contain that key. Fast for "find rows containing X". Slow to update (every token in a row updates the index).

### 5.3 GiST — Generalized Search Tree

A framework for *any* data type with a notion of "overlap" or "distance": geometry (PostGIS), ranges, IP, fuzzy strings, nearest-neighbor.

```sql
-- Range overlap
CREATE INDEX bookings_during_gist ON bookings USING gist (during);
SELECT * FROM bookings WHERE during && '[2026-04-25, 2026-04-30)'::tsrange;

-- Geometric distance
CREATE INDEX shops_location_gist ON shops USING gist (location);
SELECT * FROM shops ORDER BY location <-> POINT(0, 0) LIMIT 10;
```

Internal structure: like a B-tree, but each internal node holds a *bounding predicate* (e.g., bounding box). Searches recursively descend into all children whose predicate could overlap the query. Lossy: index returns candidates that the executor must re-check.

### 5.4 BRIN — Block-Range Index

For *correlated* data — where physical row order matches logical order. Time-series, append-only logs, IoT data.

```sql
CREATE INDEX events_ts_brin ON events USING brin (timestamp) WITH (pages_per_range = 128);
```

Internal structure: stores the *min and max* of each contiguous block of N pages. A query for `WHERE timestamp BETWEEN x AND y` scans the BRIN, finds blocks whose [min, max] overlaps [x, y], and scans only those blocks.

Tiny — ~10 KB for a 100 GB table. Useless if data isn't physically clustered (random inserts → every block has min/max spanning the entire range).

### 5.5 SP-GiST — Space-Partitioned GiST

For non-balanced trees: quad-trees, k-d trees, radix tries. Used for IP networks, phone-number prefixes, hierarchical data with varying depth.

```sql
CREATE INDEX ips_prefix_spgist ON ip_log USING spgist (cidr_block inet_ops);
SELECT * FROM ip_log WHERE cidr_block >>= inet '192.168.1.0/24';
```

### 5.6 The Trade-Off Matrix

| Index | Best For | Worst For | Update Cost |
|:---|:---|:---|:---:|
| B-tree | Equality, range, ordering | Containment | Low |
| Hash | Pure equality | Range, ordering | Low |
| GIN | Containment in composite types | Frequent updates | Very high |
| GiST | Spatial, range overlap, KNN | Pure equality (slower than B-tree) | Medium |
| BRIN | Huge correlated tables | Random inserts | Trivial |
| SP-GiST | Hierarchical, non-balanced | General use | Medium |

Choosing the wrong index is one of the easier mistakes to make and one of the hardest to spot, because the wrong index *will be used* and the query *will work* — just slower than it should.

---

## 6. Join Algorithms — Nested Loop

A join combines rows from two relations matching a predicate. Three classic algorithms exist, each with different cost profiles.

### 6.1 The Naive Algorithm

```python
# Pseudocode for the simplest join
for outer_row in outer_relation:
    for inner_row in inner_relation:
        if predicate(outer_row, inner_row):
            yield combine(outer_row, inner_row)
```

Cost: O(N * M) where N and M are the row counts. For 1M × 1M, this is 10¹². Never run in production unless one side is tiny.

### 6.2 Block Nested Loop

A modest improvement: read the outer relation in blocks of K rows, scan the inner once per block.

```
Cost = ceil(N / K) * M + N
     ≈ N * M / K
```

Reduces I/O when the outer side fits in many fewer blocks than rows. Not a complexity win, but a constant-factor improvement of ~K.

### 6.3 Indexed Nested Loop — the Production Pattern

When the inner relation has an index on the join key, the inner-side scan becomes an index lookup: O(log M) per outer row.

```
Cost = N * log(M)
```

For N=1000, M=1M: cost = 1000 * 20 = 20000. Lightning fast. This is the optimal plan when:
- One side is small (the *outer*)
- The other side has an index on the join column
- The join is selective per outer row

```sql
-- Postgres example: look up 1000 users' orders via PK index
EXPLAIN ANALYZE
SELECT u.id, o.total
  FROM users u
  JOIN orders o ON o.user_id = u.id
 WHERE u.region = 'us'
 LIMIT 1000;

-- Nested Loop  (rows=1000)
--   ->  Seq Scan on users u  (rows=1000)
--   ->  Index Scan using orders_user_id_idx on orders o  (rows=1, cost=0.42..8.44)
--         Index Cond: (user_id = u.id)
```

### 6.4 When Nested Loop Wins

| Scenario | Why NL wins |
|:---|:---|
| Outer tiny (≤ thousands), inner indexed | log(M) per outer beats hash build |
| Outer can be filtered to <1% before join | No setup cost — straight in |
| Top-N queries with `LIMIT n` | NL streams; can stop early |
| Cross-partition joins where partition pruning shrinks outer | Tiny outer sides |

### 6.5 When Nested Loop Loses

When the outer is large *and* the inner has no useful index, nested loop becomes O(N*M) and is a disaster. The classic "wrong-plan" scenario: the planner under-estimates the outer cardinality and picks NL; at runtime the outer is huge and the query never finishes.

### 6.6 Postgres Operator Details

```
Nested Loop  (cost=0.00..1234.56 rows=100 width=64)
  ->  Seq Scan on a  (...)            ← outer
  ->  Index Scan on b  (...)          ← inner, executed once per outer row
        Index Cond: (b.x = a.x)       ← parameterized at runtime
```

The inner side is *re-executed once per outer row*, with the join key bound as a parameter. The `loops` counter in `EXPLAIN ANALYZE` shows this directly:

```
->  Index Scan on b  (rows=1) (actual rows=1 loops=1234)
```

Multiply per-row cost by loops to get total.

---

## 7. Join Algorithms — Hash Join

The workhorse for joining medium-to-large relations without useful indexes.

### 7.1 The Algorithm

```python
# Phase 1: Build
hash_table = {}
for row in build_side:           # smaller relation
    key = hash(row.join_col)
    hash_table[key].append(row)

# Phase 2: Probe
for row in probe_side:           # larger relation
    key = hash(row.join_col)
    for match in hash_table[key]:
        if row.join_col == match.join_col:  # collision check
            yield combine(row, match)
```

Cost: O(N + M) — linear in both relations. The build side must fit in memory; otherwise spill to disk.

### 7.2 The Build Side Choice

The planner builds the hash on the *smaller* (estimated) side. If estimates are wrong, the build side overflows memory.

```sql
EXPLAIN ANALYZE
SELECT * FROM orders o JOIN customers c ON o.customer_id = c.id;

-- Hash Join  (cost=...rows=1000000 width=128)
--   Hash Cond: (o.customer_id = c.id)
--   ->  Seq Scan on orders o  (rows=1000000)         ← probe
--   ->  Hash  (rows=10000 width=64)                  ← build
--         ->  Seq Scan on customers c  (rows=10000)
```

Build customers (10K rows) in memory, probe orders (1M rows) once. Total: 1.01M row touches. Beats NL (10¹⁰) by 10000x.

### 7.3 Spill to Disk

When the build side exceeds `work_mem`, Postgres falls back to a *grace hash join*:

1. Partition both inputs by `hash(join_col) mod P` into P spill files
2. For each partition pair (build_i, probe_i), repeat the standard hash join

```sql
-- See spill via EXPLAIN (ANALYZE, BUFFERS)
EXPLAIN (ANALYZE, BUFFERS)
SELECT ... ;

-- Hash Join
--   Buckets: 65536  Batches: 4  Memory Usage: 8400kB
--                                          ^^^^^^^^ exceeded work_mem
--   ->  Hash  (rows=2000000)
--         Buckets: 65536  Batches: 4  Memory Usage: 8400kB
--                                              ^^^^^^^^^^^ spilled
```

`Batches: 4` means Postgres re-partitioned into 4 batches. Each batch read+wrote to disk.

The fix is usually `SET work_mem = '256MB'` for the session — but globally raising work_mem is dangerous (many concurrent queries each can use this much).

### 7.4 Hash Requires Hashable Keys

Equi-joins only. `JOIN ON a.x = b.y`: works. `JOIN ON a.x < b.y`: hash join cannot apply; falls back to NL or Merge.

### 7.5 The Hash Anti-Pattern

```sql
-- Bad: hash on a wide composite key
SELECT * FROM big1 JOIN big2 ON big1.text_col = big2.text_col;
-- Hash builds the entire text column; memory blows up
```

Index your join keys with stable hash characteristics, or use a smaller derived hash (digest, surrogate ID).

---

## 8. Join Algorithms — Merge Join / Sort-Merge Join

The third classic. Best when both inputs are already sorted on the join key.

### 8.1 The Algorithm

```python
# Both inputs must be sorted on the join key
i, j = 0, 0
while i < len(R) and j < len(S):
    if R[i].key < S[j].key:
        i += 1
    elif R[i].key > S[j].key:
        j += 1
    else:
        # Match — emit all (i', j') with R[i'].key == S[j'].key
        ...
```

Cost: O(N + M) given sorted inputs. Plus O(N log N + M log M) sort cost if inputs are not already sorted.

### 8.2 When Merge Wins

The dominant case: *both* inputs are pre-sorted by the join key. This happens when:

- An index on the join key provides sort order (Index Scan emits sorted)
- An earlier operator (Sort) already sorted them
- The data is naturally clustered (rare)

```sql
-- Both inputs come from indexes on the join key
EXPLAIN
SELECT a.id, b.id
  FROM a JOIN b ON a.k = b.k;

-- Merge Join
--   Merge Cond: (a.k = b.k)
--   ->  Index Scan using a_k_idx on a   ← already sorted
--   ->  Index Scan using b_k_idx on b   ← already sorted
```

### 8.3 The "Let the Index Provide Sort Order" Optimization

When a query needs sorted output anyway (`ORDER BY`, `GROUP BY` without hash, window function partitioning), an Index Scan that produces sorted rows is preferred even if a Seq Scan would be cheaper for the data fetch alone.

```sql
EXPLAIN
SELECT user_id, MAX(amount) FROM orders
  GROUP BY user_id;

-- GroupAggregate  ← needs sorted input
--   Group Key: user_id
--   ->  Index Scan using orders_user_id_idx on orders  ← indexed sort
-- vs.
-- HashAggregate  (hash plan, no sort needed)
--   Group Key: user_id
--   ->  Seq Scan on orders
```

The planner picks based on data size, group cardinality, and `work_mem`.

### 8.4 Inputs Need Re-Sorting

If a Sort node sits above a Seq Scan, you've paid O(N log N) — Hash Join would have been O(N + M). Merge Join *only* wins when sorts are free.

### 8.5 The Three-Way Comparison

| Property | Nested Loop | Hash Join | Merge Join |
|:---|:---|:---|:---|
| Best case cost | O(N log M) (indexed inner) | O(N + M) | O(N + M) (pre-sorted) |
| Worst case cost | O(N * M) | O((N + M) * log P) (spill) | O((N + M) log (N + M)) (sort) |
| Equi only? | No | Yes | Yes |
| Memory | O(1) | O(build side) | O(1) streaming |
| Streams output? | Yes | No (build blocks) | Yes |
| Pre-sorted inputs? | Don't care | Don't care | Required (free) |
| Stops early on LIMIT? | Yes | No | Yes |

---

## 9. Join Algorithms — Specialty

Beyond the inner-join trio, several specialized join shapes appear in production.

### 9.1 Semi-Join — EXISTS / IN

A semi-join returns rows from the *left* side that have *at least one* match on the right. Critically, it does *not* duplicate left rows when multiple right matches exist.

```sql
-- Both forms produce a semi-join in the plan
SELECT * FROM users u WHERE EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id);
SELECT * FROM users u WHERE u.id IN (SELECT user_id FROM orders);
```

The planner short-circuits on the first match per left row. With an index on `orders.user_id`, this is ~O(N log M) and strictly cheaper than a regular join + DISTINCT.

### 9.2 Anti-Join — NOT EXISTS / NOT IN

Returns left rows with *no* match on the right.

```sql
-- The "find rows in A not in B" pattern
SELECT * FROM users u WHERE NOT EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id);

-- NOT IN behaves the same UNLESS there are NULLs in the subquery
-- NULL semantics: NOT IN returns no rows if ANY value in the subquery is NULL
SELECT * FROM users WHERE id NOT IN (SELECT user_id FROM orders);
-- ⚠ if any orders.user_id IS NULL → result is empty
```

Anti-joins also short-circuit: stop scanning the right side after the first match.

### 9.3 Left Anti Join — Idiomatic LEFT JOIN ... WHERE NULL

Another way to express "rows in A not in B":

```sql
SELECT a.*
  FROM a LEFT JOIN b ON a.k = b.k
 WHERE b.k IS NULL;
```

Some planners convert this to an explicit Anti Join; others execute it as a hash with a post-filter. Check the plan.

### 9.4 Nested-Loop Semi-Join with Index — the Workhorse

The dominant *production* shape for `EXISTS`:

```
Nested Loop Semi Join
  ->  Seq Scan on users  (rows=1M)
  ->  Index Scan on orders_user_id_idx
        Index Cond: (user_id = users.id)
        Stops after first match (loops=1M, rows=0 or 1 each)
```

For each user, look up if any order exists. Stops at the first match. Beats every alternative when an index is available.

### 9.5 Cross Join, Self Join

- **Cross join:** `FROM a CROSS JOIN b` — Cartesian product, |A| * |B| rows. Rarely intentional in production; usually a missing predicate.
- **Self join:** `FROM employees e JOIN employees m ON e.manager_id = m.id` — same table twice. Plan-wise identical to two-table join.

### 9.6 Lateral Join — Iterate-then-Join

```sql
-- For each user, the 3 most recent orders
SELECT u.id, o.order_id, o.created
  FROM users u,
       LATERAL (SELECT order_id, created
                  FROM orders
                 WHERE user_id = u.id
                 ORDER BY created DESC
                 LIMIT 3) o;
```

LATERAL makes the right side parameterized by the left. Always plans as Nested Loop. This is the canonical "top-N per group" pattern.

---

## 10. EXPLAIN Output — Reading Plan Trees

`EXPLAIN` is the primary tool for understanding what the planner chose. `EXPLAIN ANALYZE` actually runs the query and reports actuals.

### 10.1 Postgres EXPLAIN Forms

```sql
EXPLAIN <query>;                      -- Plan only, no execution
EXPLAIN ANALYZE <query>;              -- Plan + actual run; ⚠ runs DML for real!
EXPLAIN (ANALYZE, BUFFERS) <query>;   -- + page-level I/O telemetry
EXPLAIN (ANALYZE, VERBOSE) <query>;   -- + every output column listed
EXPLAIN (ANALYZE, FORMAT JSON) <query>; -- machine-readable
EXPLAIN (ANALYZE, SETTINGS) <query>;  -- non-default GUCs that affected plan
EXPLAIN (ANALYZE, WAL) <query>;       -- WAL bytes generated
```

### 10.2 The Operator Catalog

| Operator | Behavior | Cost Profile |
|:---|:---|:---|
| `Seq Scan` | Read whole table sequentially | O(pages); cheap per-page |
| `Index Scan` | Descend B-tree, fetch matching heap rows | O(matches * log N) |
| `Index Only Scan` | Index covers all needed cols; skip heap | O(matches * log N), no heap I/O |
| `Bitmap Index Scan` | Build bitmap of matching tuples | One pass over index |
| `Bitmap Heap Scan` | Read heap pages from bitmap, recheck quals | One pass over relevant pages |
| `Nested Loop` | For each outer, scan inner | O(N * M) or O(N log M) |
| `Hash Join` | Build hash, probe | O(N + M) |
| `Merge Join` | Walk two sorted inputs | O(N + M) |
| `Sort` | Quicksort or external merge sort | O(N log N), blocks |
| `Hash` | Build hash table for join | O(N), blocks |
| `Aggregate` | Single-row aggregate | O(N) |
| `GroupAggregate` | Sorted-input grouping | O(N) given sorted |
| `HashAggregate` | Hash-table grouping | O(N) memory-bounded |
| `Append` | Concatenate child outputs (UNION ALL, partitions) | O(sum of children) |
| `Materialize` | Cache child output for re-scan | O(N) memory/spill |
| `Subquery Scan` | Wrap subquery output | passthrough |
| `Function Scan` | `SELECT * FROM gen_series(...)` | passthrough |
| `Values Scan` | `VALUES (...)`, (...))` literal table | passthrough |
| `CTE Scan` | Read materialized CTE | scan |
| `WindowAgg` | Compute window functions | O(N) per partition |
| `Gather` | Collect parallel-worker output | merging |
| `Limit` | Stop after N rows | O(N), short-circuits |

### 10.3 The Cost Notation

```
Seq Scan on big  (cost=0.00..18334.00 rows=1000000 width=64) (actual time=0.025..145.872 rows=1000000 loops=1)
                       ↑       ↑          ↑          ↑                       ↑           ↑           ↑       ↑
                  startup    total      est rows   est width            actual start  actual end  actual    loops
```

- **startup cost:** abstract cost units before the first row emerges
- **total cost:** abstract cost units to emit the last row
- **rows:** estimated row count
- **width:** estimated bytes per row
- **actual time:** wall time milliseconds (start..end)
- **actual rows:** actual row count
- **loops:** how many times this node executed (multiply per-row stats by this)

Cost units are arbitrary — by default `seq_page_cost = 1.0`, `random_page_cost = 4.0`, `cpu_tuple_cost = 0.01`. Compare costs *between* plans; absolute values are meaningless.

### 10.4 Reading Estimated vs Actual

```
Seq Scan on orders  (cost=0.00..15.00 rows=1000 width=64)
                                          ^^^^ estimate
                    (actual time=0.012..4521.293 rows=10000000 loops=1)
                                                    ^^^^^^^^ actual
```

Estimate 1000 vs actual 10M = 10000x off. **This is the planner's biggest failure mode.** Investigate stats freshness, multivariate dependencies, and bind-parameter sniffing.

### 10.5 The Buffers Telemetry

```sql
EXPLAIN (ANALYZE, BUFFERS) SELECT * FROM orders WHERE id = 42;

-- Index Scan using orders_pkey on orders  (...)
--   Index Cond: (id = 42)
--   Buffers: shared hit=4 read=0 dirtied=0 written=0
```

- `shared hit`: page found in buffer cache
- `shared read`: page read from disk
- `dirtied`: page modified during scan
- `written`: page evicted and written back

Cold cache: `read` >> `hit`. Warm cache: `hit` >> `read`. If you see `read` dominating on a query that *should* be cached, you have memory pressure.

### 10.6 Spotting Bad Plans

| Symptom | Likely Cause |
|:---|:---|
| Estimate 100x off actual | Stale stats, missing multivariate stats |
| `Seq Scan` + `Filter:` on big table | Missing index |
| `Sort` consuming most time | Index could provide order |
| `Nested Loop` with high `loops` | Wrong join algorithm |
| `Hash` `Batches: > 1` | work_mem too small |
| `Materialize` above large input | Plan re-scans; check predicate hoisting |
| `Heap Fetches: many` on Index Only Scan | Vacuum lag — visibility map stale |

---

## 11. Common Plan Pitfalls

The space of bad plans is much larger than the space of good ones. A few canonical failure modes:

### 11.1 Stale Stats → Tiny-Outer Misestimate

Already covered in §2.4. The fix is `ANALYZE`. The lesson is: any large data change without `ANALYZE` is a time bomb.

### 11.2 CPU vs IO Cost Mismatch

The default cost ratios assume HDD I/O dominates. On SSDs, random I/O is ~50x faster relative to sequential. The planner over-prefers Seq Scan on SSDs.

```sql
-- Tell the planner you're on SSD
SET random_page_cost = 1.1;     -- default 4.0; SSD ~1.1
SET effective_io_concurrency = 200;  -- default 1; SSD ~200
```

This often flips Seq Scans to Index Scans on selective queries.

### 11.3 Filter-Then-Join-Then-Aggregate vs Aggregate-Then-Filter

Sometimes the optimal plan is to aggregate first, filter second. The planner usually figures this out, but complex predicates can defeat it.

```sql
-- Naive: join all rows, then aggregate, then filter the aggregate
SELECT user_id, SUM(amount)
  FROM orders
 GROUP BY user_id
HAVING SUM(amount) > 10000;

-- The planner sees HAVING as a post-aggregate filter and cannot push it down.
-- If there's a WHERE clause on a single user, push it down manually:
SELECT user_id, SUM(amount)
  FROM orders
 WHERE user_id IN (SELECT id FROM big_users)  -- predicate pushdown
 GROUP BY user_id;
```

### 11.4 Missing Index — the Seq Scan + Filter Telltale

```
Seq Scan on huge_table  (cost=0.00..1234567.00 rows=100 width=...)
  Filter: (rare_col = 'x')
  Rows Removed by Filter: 9999900
```

When `Rows Removed by Filter` is much larger than rows kept, an index on `rare_col` would help. Always.

### 11.5 Wrong Join Order

The planner enumerates join orders. For >12 tables it switches from exhaustive DP to a genetic algorithm (`geqo`). Suboptimal orders are a known failure mode; tune `join_collapse_limit` and `from_collapse_limit` if needed.

```sql
-- Force a specific join order by using subqueries / WITH
WITH small AS (SELECT * FROM users WHERE region = 'us')
SELECT s.id, o.total
  FROM small s
  JOIN orders o ON o.user_id = s.id;
```

### 11.6 Parameter Sniffing on Prepared Statements

When you `PREPARE stmt AS SELECT ... WHERE col = $1`, the planner picks a plan based on the *first* execution's parameter value. If subsequent calls pass values with different selectivities, the plan is wrong.

Postgres mitigates with the *plan cache mode*: `auto` (default) replans if the planner thinks generic plan is worse, `force_custom_plan` always replans, `force_generic_plan` never replans.

```sql
-- Force per-call plans
SET plan_cache_mode = force_custom_plan;

-- Or skip prepare entirely; embed literal:
EXECUTE 'SELECT ... WHERE col = ' || quote_literal($1);
```

### 11.7 LIMIT Pushing the Plan into NL

```sql
SELECT * FROM big1 JOIN big2 ON big1.k = big2.k LIMIT 10;
```

The planner sees LIMIT and may pick Nested Loop because it can stop early. If estimates are wrong and the join is unselective, it will scan billions of outer rows looking for 10 matches. Verify with EXPLAIN.

---

## 12. Transactions and ACID

ACID is the contract every relational database promises. Each property has a specific implementation.

### 12.1 The Four Properties

| Property | Guarantee | Implementation |
|:---|:---|:---|
| Atomicity | All-or-nothing per transaction | Undo log + WAL |
| Consistency | Constraints always hold | CHECK / FK / triggers |
| Isolation | Concurrent txns appear serial | Locks + MVCC |
| Durability | Committed = persistent | fsync(WAL) on commit |

### 12.2 Atomicity via Undo + WAL

If a transaction aborts (rollback or crash), partial changes must be undone.

- **MVCC systems (Postgres):** new versions of changed rows are simply marked invalid. The old version is still there. No undo log needed for in-place rollback; vacuum reclaims later.
- **Undo-log systems (MySQL InnoDB, Oracle):** every change is recorded in an undo log. Rollback walks the undo log in reverse, applying inverse operations.

Crash-time atomicity uses WAL: replay committed transactions from the WAL; abort uncommitted ones (their WAL records are not "commit" records, so they're discarded).

### 12.3 Consistency via Constraints

Not the database's job alone — depends on application-level invariants. The database enforces:

- `NOT NULL`, `CHECK`, `UNIQUE`, `PRIMARY KEY`, `FOREIGN KEY`
- Trigger-based invariants
- Domain types (Postgres `CREATE DOMAIN`)

If a transaction would violate a constraint, the engine errors and rolls back the entire transaction. This is part of what makes "atomic + consistent" stronger than just atomic.

### 12.4 Isolation via Locks + MVCC

The hard one. Concurrent transactions cannot interfere with each other's view of the data. Achieved via:

- **Two-phase locking (2PL):** acquire all needed locks before releasing any; classical strict-2PL underpins serializable isolation in lock-based systems
- **MVCC:** each reader sees a consistent snapshot; writers don't block readers, readers don't block writers (but writers still block writers on the same row)

Both approaches are explored in §13–§15.

### 12.5 Durability via WAL fsync

When a transaction commits, the engine must ensure committed data survives a crash. The protocol:

1. Write the WAL record describing the change to the OS buffer
2. `fsync(wal_fd)` — force the OS to flush to durable storage
3. Only *after* fsync returns, acknowledge COMMIT to the client

This is why high-throughput OLTP systems are bottlenecked on fsync rates. Group commit batches multiple commits into one fsync to amortize.

### 12.6 ACID Across Engines

| Engine | Atomicity | Isolation Default | Durability |
|:---|:---|:---|:---|
| Postgres | MVCC + WAL | Read Committed | fsync per commit (synchronous_commit) |
| MySQL InnoDB | Undo log + WAL | Repeatable Read | flush_log_at_trx_commit=1 |
| SQLite | Rollback journal or WAL | Serializable | fsync per commit |
| Oracle | Undo + Redo | Read Committed | redo log fsync |
| SQL Server | Transaction log | Read Committed | log fsync |

---

## 13. Isolation Levels — The Anomaly Catalog

The SQL standard defines four isolation levels, distinguished by which *anomalies* they permit.

### 13.1 The Four Anomalies

- **Dirty read:** see a row written by an uncommitted transaction. If the writer rolls back, you saw a value that never existed.
- **Non-repeatable read:** read a row twice in the same transaction; another transaction commits an UPDATE between your reads; you see different values.
- **Phantom read:** run `SELECT WHERE x` twice; another transaction commits an INSERT matching `x` between your reads; you see new rows.
- **Lost update / write skew:** more subtle MVCC-specific anomalies (see §13.4).

### 13.2 The Standard Levels

| Level | Dirty Read | Non-Repeatable Read | Phantom |
|:---|:---:|:---:|:---:|
| READ UNCOMMITTED | possible | possible | possible |
| READ COMMITTED | prevented | possible | possible |
| REPEATABLE READ | prevented | prevented | possible |
| SERIALIZABLE | prevented | prevented | prevented |

Postgres implements READ UNCOMMITTED as READ COMMITTED (they don't allow dirty reads under any setting). The standard is permissive: stronger is allowed.

### 13.3 Snapshot Isolation — the Common Non-Standard Level

Most modern MVCC databases offer "Snapshot Isolation" (SI): each transaction sees a consistent snapshot taken at start. Reads never block, writes conflict only on the same row.

SI prevents dirty reads, non-repeatable reads, and *most* phantoms. But it permits *write skew*:

```sql
-- T1 reads {a=10, b=10}, sees sum=20, decides to set a=-5 (still sum >= 0)
-- T2 reads {a=10, b=10}, sees sum=20, decides to set b=-5 (still sum >= 0)
-- Both commit. Final: a=-5, b=-5, sum=-10. Constraint violated!

BEGIN;  -- T1
  SELECT a + b FROM accounts WHERE id IN (1, 2);  -- 20
  UPDATE accounts SET balance = balance - 5 WHERE id = 1;
COMMIT;

-- Concurrently
BEGIN;  -- T2
  SELECT a + b FROM accounts WHERE id IN (1, 2);  -- 20 (snapshot)
  UPDATE accounts SET balance = balance - 5 WHERE id = 2;
COMMIT;
```

Postgres calls SI "REPEATABLE READ" (a label that the standard wouldn't quite endorse). True serializable in Postgres uses *Serializable Snapshot Isolation* (SSI), which detects and aborts write-skew patterns at commit time.

### 13.4 Per-Engine Defaults

| Engine | Default | Notes |
|:---|:---|:---|
| Postgres | READ COMMITTED | SET TRANSACTION ISOLATION LEVEL to upgrade |
| MySQL InnoDB | REPEATABLE READ | Different semantics: gap locking prevents phantoms |
| Oracle | READ COMMITTED | Strict snapshot per statement |
| SQL Server | READ COMMITTED | Lock-based by default; can enable SI via SET ALLOW_SNAPSHOT_ISOLATION |
| DB2 | CURSOR STABILITY ≈ READ COMMITTED | |
| SQLite | SERIALIZABLE | Single-writer simplifies things |

### 13.5 Setting Isolation in SQL

```sql
-- Per transaction
BEGIN;
  SET TRANSACTION ISOLATION LEVEL SERIALIZABLE;
  -- ... statements ...
COMMIT;

-- Per session
SET SESSION CHARACTERISTICS AS TRANSACTION ISOLATION LEVEL REPEATABLE READ;

-- Per database (Postgres GUC)
ALTER DATABASE mydb SET default_transaction_isolation = 'serializable';
```

### 13.6 The Cost of Higher Isolation

Higher isolation = more conflicts = more aborts/retries. Serializable transactions in Postgres can fail with `40001 serialization_failure`; the application must retry.

```sql
BEGIN ISOLATION LEVEL SERIALIZABLE;
  SELECT ...;
  UPDATE ...;
COMMIT;
-- ERROR: could not serialize access due to read/write dependencies among transactions
-- DETAIL: Reason code: Canceled on identification as a pivot, during commit attempt.
-- HINT: The transaction might succeed if retried.
```

The retry pattern is fundamental to writing serializable application code.

---

## 14. MVCC — Multi-Version Concurrency Control

The mechanism that makes Postgres readers wait-free.

### 14.1 The Basic Idea

Every row has a hidden header containing transaction IDs that bracket its lifetime:

- `xmin`: the txn ID that created this version
- `xmax`: the txn ID that deleted (or replaced) this version; 0 if still alive
- `cmin`, `cmax`: command IDs within the creating/deleting txn (for visibility within a txn)

Each transaction has a *snapshot*: a list of txn IDs that were active when it started. A row version is visible to txn T if:

- `xmin` is committed and not in T's active set, AND
- `xmax` is 0, OR `xmax` is uncommitted, OR `xmax` is in T's active set

```sql
-- Inspect xmin/xmax in Postgres
SELECT xmin, xmax, * FROM users WHERE id = 1;
--  xmin | xmax | id | name
-- ------+------+----+-------
--   100 |    0 |  1 | alice
```

### 14.2 UPDATE Creates a New Version

```sql
BEGIN;  -- txn 200
  UPDATE users SET name = 'Alice' WHERE id = 1;
  -- old row: xmin=100, xmax=200 (deleted by us)
  -- new row: xmin=200, xmax=0
COMMIT;
```

The old row stays — the *dead tuple*. Other readers with snapshots predating txn 200 still see it.

### 14.3 Vacuum Reclaims Dead Tuples

Once no active snapshot needs a dead tuple, vacuum can mark its space free for reuse.

```bash
# Manual vacuum (rarely needed)
psql -c "VACUUM users;"

# Aggressive vacuum reclaiming xid space
psql -c "VACUUM (FREEZE, FULL) users;"   # FULL rewrites the entire table
```

`VACUUM FULL` is exclusive-locked and rewrites the table; avoid in production. Regular `VACUUM` is online but doesn't shrink files.

### 14.4 Autovacuum Tuning

```sql
-- Per-table tuning for hot tables
ALTER TABLE orders SET (
  autovacuum_vacuum_scale_factor = 0.05,    -- vacuum at 5% dead tuples
  autovacuum_vacuum_cost_limit = 2000,       -- I/O bandwidth knob
  autovacuum_vacuum_cost_delay = 0           -- run flat-out
);

-- See bloat (dead-tuple-aware row counts)
SELECT relname, n_live_tup, n_dead_tup,
       round(100.0 * n_dead_tup / NULLIF(n_live_tup + n_dead_tup, 0), 1) AS dead_pct
  FROM pg_stat_user_tables
 ORDER BY dead_pct DESC NULLS LAST
 LIMIT 20;
```

### 14.5 The Bloat Problem

Heavy update workloads outrun vacuum. Tables accumulate dead tuples; the file grows; queries get slower (more pages to scan); indexes bloat similarly.

Symptoms:
- Table size on disk much larger than `n_live_tup * avg_row_size`
- Query latencies climb over weeks
- `pg_stat_user_tables.n_dead_tup` consistently high

Fixes:
- Tune autovacuum to be more aggressive
- Use `pg_repack` or `VACUUM (FULL)` (ouch) to compact
- Consider partitioning to bound per-partition bloat
- For write-once tables, BRIN + no updates avoids the problem entirely

### 14.6 InnoDB's Approach — Undo Log

MySQL InnoDB takes a different approach. Updates modify rows *in place* but write the old version to an *undo log*. Old readers consult the undo log to reconstruct the snapshot they need.

Trade-offs vs Postgres MVCC:

| Aspect | Postgres MVCC | InnoDB Undo |
|:---|:---|:---|
| In-place updates | No (creates new tuple) | Yes |
| Long-running readers | Block vacuum → bloat | Stretch undo log → "purge lag" |
| Index updates on UPDATE | Always (new TID) unless HOT | Only if indexed columns change |
| Rollback cost | Cheap (mark new version invalid) | Walk undo log, undo each change |
| Writer-writer conflict | Row-level lock + retry on serialization | Row lock; retry as needed |

Postgres has *Heap-Only Tuples* (HOT): if an update doesn't touch any indexed column and the new tuple fits on the same page, indexes aren't updated. This dampens bloat for narrow updates.

### 14.7 Visibility Inside a Transaction

Within one transaction, you see your own changes; outside, you see your snapshot. Postgres uses `cmin/cmax` to track per-statement visibility, which is why your second statement in a txn can see your first's changes.

---

## 15. Locking — Shared, Exclusive, Intent

MVCC handles read-vs-write concurrency. Write-vs-write still uses locks.

### 15.1 The Lock Mode Matrix

The classical hierarchy from textbook databases:

| Mode | Permits | Conflicts With |
|:---|:---|:---|
| **S** (Shared) | Other S | X, IX, SIX |
| **X** (Exclusive) | Nothing | All |
| **IS** (Intent Shared) | IS, IX, S, SIX | X |
| **IX** (Intent Exclusive) | IS, IX | S, SIX, X |
| **SIX** (Shared + Intent Exclusive) | IS | IX, S, SIX, X |

Intent locks are taken on *parents* (table) when the actual lock is taken on a *child* (row). They prevent table-level locks from being granted while row-level locks exist.

### 15.2 Postgres Lock Levels

Postgres uses 8 table-level lock modes (see `pg_locks`):

| Mode | Granted by |
|:---|:---|
| ACCESS SHARE | SELECT |
| ROW SHARE | SELECT FOR UPDATE/SHARE |
| ROW EXCLUSIVE | INSERT, UPDATE, DELETE |
| SHARE UPDATE EXCLUSIVE | VACUUM, ANALYZE, CREATE INDEX CONCURRENTLY |
| SHARE | CREATE INDEX (non-concurrent) |
| SHARE ROW EXCLUSIVE | not auto-acquired |
| EXCLUSIVE | REFRESH MATERIALIZED VIEW CONCURRENTLY |
| ACCESS EXCLUSIVE | DROP, TRUNCATE, ALTER, VACUUM FULL |

ACCESS EXCLUSIVE blocks all other access. This is why DDL on production tables is dangerous.

### 15.3 Row-Level Locks

Postgres tracks row locks via `xmax` with a "lock-only" flag. Modes:

```sql
SELECT * FROM users WHERE id = 1 FOR UPDATE;        -- exclusive row lock
SELECT * FROM users WHERE id = 1 FOR NO KEY UPDATE; -- weaker; allows FK readers
SELECT * FROM users WHERE id = 1 FOR SHARE;          -- shared row lock
SELECT * FROM users WHERE id = 1 FOR KEY SHARE;      -- weakest; FK enforcement
```

`FOR UPDATE` is the canonical "I will modify this row; nobody else may" lock, used for select-then-update sequences.

### 15.4 Deadlock Detection

A deadlock cycle: T1 holds lock A, waits for B; T2 holds B, waits for A. Neither can proceed.

Postgres runs a deadlock detector every `deadlock_timeout` (default 1s) of waiting. If a cycle is found, one transaction is aborted with `40P01 deadlock_detected`.

```sql
-- Symptom in logs:
ERROR:  deadlock detected
DETAIL: Process 12345 waits for ShareLock on transaction 100;
        blocked by process 67890.
```

### 15.5 The Retry Pattern

```python
# Application-level retry on serialization failure or deadlock
import time, random

def run_txn(conn, body):
    for attempt in range(5):
        try:
            with conn.transaction():
                return body(conn)
        except (SerializationFailure, DeadlockDetected):
            time.sleep((2 ** attempt) * 0.01 + random.random() * 0.01)
    raise RuntimeError("transaction failed after 5 retries")
```

Exponential backoff with jitter prevents thundering-herd retries.

### 15.6 lock_timeout and statement_timeout

```sql
-- Don't wait longer than 5 seconds for a lock
SET lock_timeout = '5s';

-- Don't run a single statement longer than 30 seconds
SET statement_timeout = '30s';

-- Don't keep a transaction idle longer than 60 seconds
SET idle_in_transaction_session_timeout = '60s';
```

These are essential safety knobs. An ungoverned `BEGIN; SELECT FOR UPDATE; (forever)` will hold its lock indefinitely, blocking everyone behind it.

### 15.7 FOR UPDATE NOWAIT and SKIP LOCKED

```sql
-- Fail immediately if row is already locked
SELECT * FROM jobs WHERE id = 1 FOR UPDATE NOWAIT;

-- Skip locked rows; useful for queue workers
SELECT * FROM jobs WHERE state = 'pending' FOR UPDATE SKIP LOCKED LIMIT 10;
```

`SKIP LOCKED` is the canonical pattern for SQL-as-job-queue: each worker pulls 10 unlocked jobs, locks them, processes them. No worker blocks another.

### 15.8 Inspecting Locks

```sql
-- Who's blocking whom?
SELECT blocked.pid AS blocked_pid,
       blocked.query AS blocked_query,
       blocking.pid AS blocking_pid,
       blocking.query AS blocking_query
  FROM pg_stat_activity blocked
  JOIN pg_stat_activity blocking
    ON blocking.pid = ANY(pg_blocking_pids(blocked.pid))
 WHERE blocked.wait_event_type = 'Lock';
```

Production essential. Without it, you're guessing.

---

## 16. Write-Ahead Log (WAL)

The principle: never modify data on disk before its description is durably logged.

### 16.1 The W-A-L Principle

```
For every change to durable storage:
  1. Write the change description to the WAL
  2. fsync the WAL to durable storage
  3. THEN apply the change to data pages (in memory or disk)
```

If the system crashes:
- WAL is intact up to last fsync
- Data pages may be in an inconsistent state
- Recovery: replay WAL records → data pages reach consistent state

### 16.2 fsync Semantics

`fsync(fd)` blocks until the file's bytes are physically committed to durable storage (or the storage device acknowledges, which on cheap drives can be a lie). On modern systems this is the bottleneck of high-rate transactional workloads.

```bash
# Postgres synchronous_commit options
SET synchronous_commit = on;       # default: fsync WAL on commit
SET synchronous_commit = off;      # trade durability for throughput; ~30s window
SET synchronous_commit = remote_apply;  # await replica apply (synchronous replication)
```

`synchronous_commit = off` is *not* unsafe in the traditional sense — Postgres still preserves atomicity and consistency — but a crash can lose the last few hundred ms of committed transactions. Used for non-critical workloads with high throughput.

### 16.3 Checkpoints

The data pages can drift arbitrarily far from durable WAL — recovery would replay forever. Checkpoints periodically flush all dirty pages to disk so older WAL can be discarded.

```sql
-- Tunables
SHOW checkpoint_timeout;          -- default 5min
SHOW max_wal_size;                -- default 1GB; checkpoint when exceeded
SHOW checkpoint_completion_target; -- default 0.9; spread writes over % of interval
```

Checkpoints are a tail-latency hazard: a sudden burst of dirty-page flushes can saturate I/O. `checkpoint_completion_target = 0.9` smooths the writes over 90% of the interval.

### 16.4 wal_level

```sql
SHOW wal_level;
-- minimal:   only what's needed for crash recovery
-- replica:   + enough for streaming replication & PITR (default since PG 10)
-- logical:   + row-level changes for logical decoding (CDC, replication slots)
```

Setting `wal_level = logical` enables `CREATE PUBLICATION` and logical replication slots — the basis for change data capture (CDC) tools like Debezium.

### 16.5 Logical Decoding

```sql
-- Create a replication slot
SELECT pg_create_logical_replication_slot('cdc_slot', 'pgoutput');

-- Read changes (test_decoding plugin)
SELECT * FROM pg_logical_slot_get_changes('cdc_slot', NULL, NULL);

-- Drop when done
SELECT pg_drop_replication_slot('cdc_slot');
```

A consumer that doesn't drain its slot pins WAL retention indefinitely. Disk-full incidents are common.

### 16.6 Backup From WAL — PITR

Continuous archiving plus a base backup gives you point-in-time recovery (PITR):

```bash
# Base backup
pg_basebackup -D /backup/base -X stream

# Configure archive_command in postgresql.conf
archive_mode = on
archive_command = 'cp %p /archive/%f'

# To restore to a specific time
restore_command = 'cp /archive/%f %p'
recovery_target_time = '2026-04-25 12:00:00'
```

### 16.7 Crash Recovery

On startup:
1. Read the last checkpoint location from `pg_control`
2. Replay WAL records from that location forward
3. Identify uncommitted transactions; mark their changes as aborted (in MVCC, this is just "xmin is uncommitted")
4. Open for connections

Recovery time is bounded by `max_wal_size` and the I/O bandwidth.

### 16.8 WAL on Other Engines

| Engine | Log Name | Notes |
|:---|:---|:---|
| Postgres | WAL | Single log, durable on commit |
| MySQL InnoDB | redo log + undo log | redo for forward replay; undo for rollback |
| SQLite | rollback journal *or* WAL | WAL mode is opt-in but recommended |
| Oracle | Redo log + UNDO segments | Multi-tier archive logs |

The principle is universal; the implementation details differ.

---

## 17. Query Plan Optimization — Common Wins

When EXPLAIN reveals a bad plan, the toolbox of fixes:

### 17.1 Covering Indexes — INCLUDE Clause

Make an index "cover" non-key columns the query selects, enabling Index Only Scan.

```sql
-- Before: index on (user_id), heap fetch for amount
CREATE INDEX orders_user_idx ON orders (user_id);

-- After: index covers user_id AND amount
CREATE INDEX orders_user_amt_idx ON orders (user_id) INCLUDE (amount);

EXPLAIN SELECT amount FROM orders WHERE user_id = 42;
-- Index Only Scan using orders_user_amt_idx on orders
--   Heap Fetches: 0
```

### 17.2 Partial Indexes — Subset of Rows

Index only the rows that matter; smaller index, faster scans.

```sql
-- 99% of orders are 'completed', queries always look for the 1%
CREATE INDEX orders_pending_idx ON orders (created)
  WHERE status = 'pending';

-- Tiny index, only contains pending rows
SELECT * FROM orders WHERE status = 'pending' AND created > NOW() - INTERVAL '1 day';
```

### 17.3 Expression Indexes

Index a derived value when the query filters on it.

```sql
-- Queries filter on lower(email); plain index on email doesn't help
CREATE INDEX users_email_lower_idx ON users (lower(email));

SELECT * FROM users WHERE lower(email) = 'alice@example.com';
-- Uses the expression index
```

The query *must* use the exact expression form. `lower(email)` matches; `LOWER(email)` matches (SQL is case-insensitive on functions); `lower(trim(email))` does not.

### 17.4 Index-Only Scans on Covering Indexes

Already covered (§4.7, §17.1). The combination of covering + visibility map + low write rate gives sub-millisecond reads.

### 17.5 Query Rewriting — IN to JOIN

Old wisdom: rewrite `IN (SELECT ...)` as `JOIN`. Modern planners do this automatically. Verify with EXPLAIN.

```sql
-- These plan identically in Postgres 12+
SELECT * FROM users WHERE id IN (SELECT user_id FROM orders);
SELECT u.* FROM users u JOIN (SELECT DISTINCT user_id FROM orders) o ON u.id = o.user_id;
```

### 17.6 NOT IN to NOT EXISTS

Always rewrite `NOT IN` to `NOT EXISTS` for nullable columns. The NULL semantics of `NOT IN` are catastrophic:

```sql
-- DANGEROUS: if any orders.user_id is NULL, this returns no rows
SELECT * FROM users WHERE id NOT IN (SELECT user_id FROM orders);

-- SAFE and usually faster
SELECT * FROM users u WHERE NOT EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id);
```

### 17.7 SELECT Only Needed Columns

`SELECT *` defeats Index Only Scans, increases row width, increases sort/hash memory. Always project explicitly in production queries.

### 17.8 LATERAL JOIN for Top-N Per Group

```sql
-- 3 most recent orders per user
SELECT u.id, o.order_id, o.created
  FROM users u
  CROSS JOIN LATERAL (
    SELECT order_id, created FROM orders
     WHERE user_id = u.id
     ORDER BY created DESC LIMIT 3
  ) o
 WHERE u.region = 'us';
```

With an index on `orders (user_id, created DESC)`, this is one Index Scan per user, stopping after 3 rows. Beats every window-function alternative.

### 17.9 CTE Materialization

In Postgres pre-12, every CTE was an "optimization fence" — materialized once, scanned. In 12+ they're inlined by default unless declared `MATERIALIZED`.

```sql
-- Force materialization (pre-12 default behavior)
WITH x AS MATERIALIZED (SELECT ... FROM big_table WHERE filter) ...

-- Force inlining
WITH x AS NOT MATERIALIZED (SELECT ... FROM big_table WHERE filter) ...
```

### 17.10 Parallel Query

```sql
-- Tunables
SHOW max_parallel_workers_per_gather;  -- default 2
SHOW parallel_tuple_cost;
SHOW parallel_setup_cost;

-- Force parallel for testing
SET force_parallel_mode = on;
SET min_parallel_table_scan_size = 0;
```

Seq Scans and Hash Joins on big tables benefit. Index Scans rarely do — most index scans are short.

---

## 18. Pagination Algorithms

How you page through a result set determines whether your API stays fast at page 1000.

### 18.1 The LIMIT/OFFSET Trap

```sql
-- Page 1000 of 50-row pages: skip 49950 rows
SELECT * FROM articles ORDER BY created DESC LIMIT 50 OFFSET 49950;
```

The database must *generate* and *discard* the first 49950 rows. Cost is O(N) regardless of index. Latency grows linearly with page number.

### 18.2 The Keyset / Seek Method

Use the previous page's last row as a cursor:

```sql
-- First page
SELECT id, title, created FROM articles ORDER BY created DESC LIMIT 50;
-- Returns last row with created = '2026-04-25 12:00:00', id = 12345

-- Next page: seek beyond the last cursor
SELECT id, title, created FROM articles
 WHERE (created, id) < ('2026-04-25 12:00:00', 12345)
 ORDER BY created DESC, id DESC
 LIMIT 50;
```

With an index on `(created, id)`, this is O(log N + page_size). Constant latency regardless of page depth.

### 18.3 Why the Composite Tuple

A naive seek `WHERE created < '2026-04-25 12:00:00'` is wrong: rows tied at exactly that timestamp are missed. The tuple comparison `(created, id) < (...)` orders consistently across the unique pair.

### 18.4 Cursor-Based for Stable Iteration

PostgreSQL `DECLARE CURSOR` is the database-side primitive:

```sql
BEGIN;
DECLARE my_cur CURSOR FOR
  SELECT id, title FROM articles ORDER BY created DESC;
FETCH 50 FROM my_cur;
FETCH 50 FROM my_cur;
-- ...
COMMIT;
```

The snapshot is held by the open transaction. New writes don't affect what the cursor returns. Useful for batch jobs; less useful for stateless HTTP APIs.

### 18.5 ORM Implementations

Most ORMs default to LIMIT/OFFSET because it's simple. The good ones support keyset:

- Django: `django-cursor-pagination`
- Rails: `order_query`, `pagy_keyset`
- SQLAlchemy: explicit tuple comparisons
- Hibernate: `setFirstResult` (offset) is the default; keyset requires manual writing

### 18.6 The Trade-Off Summary

| Method | Latency at deep pages | Stable across writes | Bookmarkable URL |
|:---|:---|:---|:---|
| LIMIT / OFFSET | O(N) | No (writes shift offsets) | Yes |
| Keyset | O(log N + page) | Yes | Yes |
| Cursor (txn) | O(page) | Yes (snapshot) | No |

Keyset is the right answer 90% of the time.

---

## 19. Aggregation Algorithms

`GROUP BY` and friends.

### 19.1 GroupAggregate vs HashAggregate

The two strategies:

- **GroupAggregate:** input must be sorted by group keys; aggregate as we walk. O(N log N) if sort needed; O(N) if input pre-sorted.
- **HashAggregate:** maintain a hash from group key to running aggregate. O(N) but bounded by `work_mem`.

```sql
EXPLAIN ANALYZE
SELECT user_id, SUM(amount) FROM orders GROUP BY user_id;

-- HashAggregate  (cost=18334.00..18434.00 rows=10000)
--   Group Key: user_id
--   Batches: 1  Memory Usage: 1024kB
--   ->  Seq Scan on orders
```

If the hash exceeds work_mem, it spills to disk:

```
HashAggregate
  Batches: 5  Memory Usage: 4096kB  Disk Usage: 32MB
```

### 19.2 The Streaming / Partial / Final Pattern

In distributed SQL (Citus, Spark, Trino), aggregations split into partial-then-final:

- Each worker computes partial aggregates on its shard
- A coordinator combines partials into final results

For sum: `partial_sum + partial_sum = total_sum`. For avg: workers must return `(sum, count)`, not `avg`. For median: cannot be computed from partials — the coordinator needs all data.

### 19.3 Window Functions

```sql
SELECT id, amount,
       SUM(amount) OVER (PARTITION BY user_id ORDER BY created) AS running_total
  FROM orders;
```

Execution model:
1. Sort by `(user_id, created)`
2. Walk in order; maintain running sum per partition
3. Emit each row with its computed window value

Cost: O(N log N) for the sort + O(N) for the walk. An index on `(user_id, created)` skips the sort.

Window frame styles:

| Frame | Cost per row | Notes |
|:---|:---|:---|
| `ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW` | O(1) running | Cumulative |
| `ROWS BETWEEN N PRECEDING AND N FOLLOWING` | O(N) | Sliding window |
| `RANGE BETWEEN INTERVAL '1 day' PRECEDING AND CURRENT ROW` | O(log N) per row | Time-based; needs ordering on time |

### 19.4 DISTINCT — Not Free

`SELECT DISTINCT col` is essentially `SELECT col GROUP BY col`. Same cost: hash or sort.

```sql
-- These plan identically
SELECT DISTINCT user_id FROM orders;
SELECT user_id FROM orders GROUP BY user_id;
```

### 19.5 ROLLUP / CUBE / GROUPING SETS

```sql
-- Subtotals + grand total
SELECT region, product, SUM(amount)
  FROM sales
 GROUP BY ROLLUP (region, product);

-- region        | product | sum
-- 'us'          | 'a'     | 100
-- 'us'          | 'b'     | 200
-- 'us'          |  NULL   | 300   ← subtotal for 'us'
-- 'eu'          | 'a'     | 150
-- 'eu'          |  NULL   | 150
--  NULL         |  NULL   | 450   ← grand total
```

The planner emits these as a chain of partial aggregates. Cheaper than running multiple GROUP BY queries and UNION-ing.

---

## 20. JSON Internals

Modern Postgres has two JSON types and they are not equivalent.

### 20.1 json vs jsonb

| Property | json | jsonb |
|:---|:---|:---|
| Storage | Text (verbatim) | Parsed binary tree |
| Read cost | Re-parse every access | O(log) field lookup |
| Whitespace preserved | Yes | No |
| Key order preserved | Yes | No |
| Duplicate keys allowed | Yes (last wins on parse) | No |
| Indexable | No (need expression index) | Yes (GIN) |

**Use jsonb in production.** json is for "I want to store text and validate it's JSON".

### 20.2 Indexing jsonb with GIN

Two operator classes for GIN on jsonb:

```sql
-- Default: indexes everything
CREATE INDEX events_data_gin ON events USING gin (data);

-- jsonb_path_ops: only the @> operator, much smaller and faster
CREATE INDEX events_data_path_gin ON events USING gin (data jsonb_path_ops);

-- Query
SELECT * FROM events WHERE data @> '{"type": "purchase"}';
```

### 20.3 The Operator Cheat Sheet

```sql
data -> 'key'        -- get value as jsonb
data ->> 'key'       -- get value as text
data #> '{a,b,c}'    -- get nested value as jsonb
data #>> '{a,b,c}'   -- get nested value as text
data @> '{...}'      -- contains
data <@ '{...}'      -- is contained by
data ? 'key'         -- key exists at top level
data ?| ARRAY[...]   -- any of these keys exist
data ?& ARRAY[...]   -- all of these keys exist
```

### 20.4 SQL Standard JSON

Postgres 12+ added the SQL/JSON standard operators:

```sql
SELECT JSON_VALUE(data, '$.user.name'),
       JSON_QUERY(data, '$.tags[*]' WITH WRAPPER),
       JSON_EXISTS(data, '$.user.email')
  FROM events;
```

These are vendor-portable to Oracle and DB2; the `@>` family is Postgres-specific.

### 20.5 Path Expressions

```sql
SELECT jsonb_path_query(data, '$.products[*] ? (@.price > 100).name')
  FROM orders;
```

JSONPath inside jsonb. The `?` clause filters; the path navigates. Powerful but indexable only via expression GIN.

### 20.6 The Schema-Last Trade-Off

Storing jsonb is convenient — no migrations on shape change. But:
- No type checking at write time
- Index strategy must match query patterns
- Foreign keys impossible into nested fields
- Updates to nested fields rewrite the entire jsonb (immutable internally)

Use jsonb for *truly variable* data. Use columns for *known* fields.

---

## 21. Vendor-Specific Internals

The big four engines, with their distinguishing internals.

### 21.1 PostgreSQL

- **TOAST (The Oversized Attribute Storage Technique):** values > 2 KB are compressed; > 8 KB are stored out-of-line in a TOAST table. Transparent to queries. Look for "Compression Method: pglz" or "lz4" in TOAST'd columns.
- **HOT updates:** when no indexed column changes and the new tuple fits on the same page, indexes don't get updated. See `pg_stat_user_tables.n_tup_hot_upd`.
- **Parallel workers:** Postgres can fan out Seq Scan, Hash Join, Aggregate to multiple workers. `max_parallel_workers_per_gather` controls fan-out per query.
- **Materialized views with REFRESH CONCURRENTLY:** uses a unique index to compute the diff and apply it under SHARE lock, allowing concurrent reads.

```sql
CREATE MATERIALIZED VIEW user_stats AS
  SELECT user_id, COUNT(*), SUM(amount) FROM orders GROUP BY user_id;
CREATE UNIQUE INDEX ON user_stats (user_id);
REFRESH MATERIALIZED VIEW CONCURRENTLY user_stats;
```

### 21.2 MySQL InnoDB

- **Clustered primary key:** PK *is* the row layout. PK lookups are 1 I/O; secondary index lookups are 2 I/O (secondary → PK → row).
- **Redo log + undo log:** redo for forward crash recovery, undo for rollback and snapshot reads.
- **Doublewrite buffer:** before writing a page to the data file, write it to a doublewrite area. Protects against torn writes (partial-page writes during a crash). Cost: 2x writes; can disable on hardware that guarantees atomic 16 KB writes.
- **Adaptive hash index:** InnoDB adds an internal hash index on hot pages automatically. Can be disabled.
- **Buffer pool:** InnoDB caches data + index pages. Tuning `innodb_buffer_pool_size` to ~70% of RAM is the rule.

```sql
-- See InnoDB internals
SHOW ENGINE INNODB STATUS;
```

### 21.3 SQLite

- **Embedded library, not a server:** runs in-process; one file per database; no network protocol.
- **Rollback journal vs WAL mode:** classic mode writes original pages to a journal before modifying; WAL mode writes new pages to a separate WAL and the database file is read from + WAL until checkpoint. WAL mode allows concurrent readers with one writer.
- **B-tree per table:** every table is a B-tree on the rowid (or PK if `WITHOUT ROWID`).
- **Single writer:** at most one writer at a time. Readers don't block writers (in WAL mode).

```sql
-- Enable WAL mode
PRAGMA journal_mode = WAL;

-- Check
PRAGMA journal_mode;
```

### 21.4 Oracle

- **UNDO + REDO:** UNDO segments hold pre-images for rollback and read-consistency; REDO logs hold change records for recovery and replication.
- **Flashback:** exploits UNDO to query historical states (`AS OF TIMESTAMP`).
- **Result cache:** caches query results across executions; invalidated by underlying changes.
- **PGA / SGA:** the System Global Area (shared) and Process Global Area (per-session) split working memory.

### 21.5 SQL Server

- **Transaction log:** the WAL equivalent. `BACKUP LOG` truncates.
- **Tempdb:** shared scratch space for sorts, hashes, temp tables.
- **In-Memory OLTP (Hekaton):** lock-free MVCC tables in memory with native-compiled stored procedures.
- **Columnstore indexes:** for analytical workloads. Compressed batch processing.

### 21.6 The Take-Away

The relational model is portable. The internals are not. A query optimizer that loves Postgres will hate MySQL. Learn one engine deeply, then translate to others.

---

## 22. Idioms at the Internals Depth

Patterns that exploit the internals for production wins.

### 22.1 ORDER BY Indexed Column for Sort-Skip

```sql
-- Index on (created DESC)
CREATE INDEX articles_created_idx ON articles (created DESC);

-- Query never sorts; reads index in order
SELECT id, title FROM articles ORDER BY created DESC LIMIT 50;

-- Plan:
-- Limit
--   ->  Index Scan using articles_created_idx on articles
--                                     ↑ provides sort order
```

### 22.2 Covering Index with INCLUDE

```sql
CREATE INDEX orders_user_amt_idx ON orders (user_id) INCLUDE (amount, status);

-- Index Only Scan; never touches heap
SELECT amount, status FROM orders WHERE user_id = 42;
```

The INCLUDE columns are stored in leaf pages but not in the search key — smaller index, same coverage.

### 22.3 Partial Index for Hot WHERE Patterns

```sql
-- 99% of rows are 'completed', queries always look for the 1% pending
CREATE INDEX orders_pending_idx ON orders (created) WHERE status = 'pending';

-- 100x smaller than a full index, scans only relevant rows
SELECT * FROM orders WHERE status = 'pending' AND created > NOW() - INTERVAL '1 hour';
```

### 22.4 LATERAL JOIN for Top-N Per Group

```sql
CREATE INDEX orders_user_created_idx ON orders (user_id, created DESC);

-- Top 3 orders per US user
SELECT u.id, o.order_id, o.created
  FROM users u
  CROSS JOIN LATERAL (
    SELECT order_id, created FROM orders
     WHERE user_id = u.id
     ORDER BY created DESC LIMIT 3
  ) o
 WHERE u.region = 'us';
```

### 22.5 WINDOW Function for Running Totals

```sql
SELECT id, user_id, amount,
       SUM(amount) OVER (PARTITION BY user_id ORDER BY created) AS running_sum
  FROM orders;
```

With `(user_id, created)` indexed, the sort is free; the window walk is O(N).

### 22.6 Recursive CTE for Hierarchies

```sql
-- Find all descendants of node 1 in a tree
WITH RECURSIVE descendants AS (
  SELECT id, parent_id, name FROM nodes WHERE id = 1
  UNION ALL
  SELECT n.id, n.parent_id, n.name
    FROM nodes n JOIN descendants d ON n.parent_id = d.id
)
SELECT * FROM descendants;
```

The planner runs the anchor query once, then iteratively re-runs the recursive part with the previous iteration's output, until no new rows. No native graph traversal, but works.

### 22.7 UPSERT — INSERT ON CONFLICT

```sql
INSERT INTO counters (key, value) VALUES ('hits', 1)
  ON CONFLICT (key) DO UPDATE SET value = counters.value + 1
  RETURNING value;
```

Atomic read-modify-write per row. No need for `BEGIN; SELECT FOR UPDATE; UPDATE; COMMIT;`.

### 22.8 SKIP LOCKED for Job Queues

```sql
-- Worker pulls 10 unlocked jobs, marks them processing
WITH next AS (
  SELECT id FROM jobs WHERE status = 'pending'
   ORDER BY created
   FOR UPDATE SKIP LOCKED
   LIMIT 10
)
UPDATE jobs SET status = 'processing'
 WHERE id IN (SELECT id FROM next)
 RETURNING *;
```

100% utilization with no worker blocking another.

### 22.9 Returning Generated Values

```sql
-- Get the new id without a separate SELECT
INSERT INTO orders (user_id, amount) VALUES (42, 100)
  RETURNING id, created;

-- Same for UPDATE
UPDATE orders SET status = 'shipped' WHERE id = 1
  RETURNING shipped_at;
```

### 22.10 Filtered Aggregates with FILTER

```sql
SELECT
  COUNT(*) AS total,
  COUNT(*) FILTER (WHERE status = 'completed') AS completed,
  COUNT(*) FILTER (WHERE status = 'pending') AS pending,
  AVG(amount) FILTER (WHERE status = 'completed') AS avg_completed
FROM orders;
```

Single-pass aggregate replacing what would otherwise be multiple subqueries.

### 22.11 Avoiding the N+1

```sql
-- Bad: in application code, loop over users and run a separate query
for user in users:
    orders = db.exec("SELECT * FROM orders WHERE user_id = ?", user.id)

-- Good: one query
SELECT u.*, json_agg(o.*) AS orders
  FROM users u LEFT JOIN orders o ON o.user_id = u.id
 GROUP BY u.id;
```

### 22.12 Generating Series

```sql
-- Calendar table on the fly
SELECT generate_series('2026-01-01'::date, '2026-12-31'::date, '1 day') AS day;

-- Outer-join against to fill gaps
SELECT day, COALESCE(c.cnt, 0) AS cnt
  FROM generate_series('2026-04-01'::date, CURRENT_DATE, '1 day') AS day
  LEFT JOIN (SELECT created::date AS d, COUNT(*) AS cnt FROM events GROUP BY d) c
    ON c.d = day;
```

---

## 23. Prerequisites

- Familiarity with SQL syntax (SELECT, JOIN, GROUP BY, indexes)
- Basic understanding of B-tree data structures
- Algorithm complexity (Big-O for scan, seek, sort)
- Concept of a transaction and ACID properties
- Awareness of concurrent execution and race conditions
- Comfort with `EXPLAIN` output (see §10 to develop fluency)
- A running Postgres or MySQL instance to experiment against

---

## 24. Complexity

- **Beginner:** reading EXPLAIN output, distinguishing Seq Scan from Index Scan, understanding why an index helps
- **Intermediate:** picking the right index type (B-tree vs GIN vs partial), recognizing nested-loop disasters from stale stats, choosing the right isolation level for a workload, writing keyset pagination
- **Advanced:** diagnosing parameter-sniffing regressions on prepared statements, tuning autovacuum + work_mem + random_page_cost together, implementing serializable retry loops, debugging WAL buildup from stuck replication slots, designing schemas with multivariate stats in mind
- **Expert:** reading the Postgres source to understand why a specific plan was chosen, contributing to the planner, designing distributed-SQL aggregation strategies that compose partial-then-final, building CDC pipelines on logical decoding

---

## 25. See Also

- [sql](../../sheets/databases/sql.md) — practical SQL cheat sheet (the companion to this internals deep-dive)
- [polyglot](../../sheets/languages/polyglot.md) — language-agnostic patterns that map to SQL idioms
- [postgresql](../../sheets/databases/postgresql.md) — Postgres operational and DDL cheat sheet
- [mysql](../../sheets/databases/mysql.md) — MySQL operational and DDL cheat sheet
- [sqlite](../../sheets/databases/sqlite.md) — SQLite library and pragma reference
- [redis](../../sheets/databases/redis.md) — when SQL is wrong: key-value, caching, pub-sub

---

## 26. References

- **PostgreSQL Documentation** — <https://www.postgresql.org/docs/current/> (the planner, MVCC, WAL, indexes, locks — the canonical reference for everything in this document)
- **MySQL Reference Manual** — <https://dev.mysql.com/doc/refman/> (InnoDB internals, redo/undo logs, the optimizer)
- **SQLite Documentation** — <https://www.sqlite.org/lang.html> (especially `lockingv3.html`, `wal.html`, `optoverview.html`)
- **"Database Internals" by Alex Petrov (O'Reilly, 2019)** — THE book on storage engines, indexes, transactions, distributed systems. Read this first if you read nothing else.
- **"Designing Data-Intensive Applications" by Martin Kleppmann (O'Reilly, 2017)** — broader view of databases in distributed systems; chapters 3, 5, 7 are essential
- **<https://use-the-index-luke.com/>** — Markus Winand's free book on index-aware SQL design; the canonical tutorial on B-tree internals applied to query writing
- **<https://modern-sql.com/>** — Markus Winand again; tracks recent SQL standard features (window functions, JSON, MERGE)
- **"Readings in Database Systems" 5th ed (Hellerstein, Stonebraker)** — <http://www.redbook.io/> — the classical academic reader; required if you want to understand the research lineage of every concept here
- **PostgreSQL source code** — <https://github.com/postgres/postgres>; especially `src/backend/optimizer/` and `src/backend/access/heap/`
- **"The Internals of PostgreSQL" by Hironobu Suzuki** — <https://www.interdb.jp/pg/> — free online book; chapters on MVCC, WAL, and the buffer manager are exceptional
- **Bruce Momjian's talks** — <https://momjian.us/main/presentations/internals.html> — the clearest oral exposition of Postgres internals available
