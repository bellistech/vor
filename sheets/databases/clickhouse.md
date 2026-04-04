# ClickHouse (Columnar OLAP Database)

High-performance columnar database for real-time analytics and OLAP queries, using MergeTree storage engines, vectorized query execution, and distributed tables for petabyte-scale analytical workloads.

## Connection

```bash
# CLI client
clickhouse-client
clickhouse-client --host=ch.example.com --port=9000 --user=default --password=secret
clickhouse-client --database=analytics --format=Pretty

# HTTP interface
curl 'http://localhost:8123/?query=SELECT+1'
curl 'http://localhost:8123/' --data-binary 'SELECT count() FROM events'

# With authentication
curl 'http://localhost:8123/?user=default&password=secret' \
  --data-binary 'SELECT * FROM events LIMIT 10'
```

## Table Engines

### MergeTree (primary engine)

```sql
CREATE TABLE events (
    event_date Date,
    event_time DateTime,
    user_id UInt64,
    event_type LowCardinality(String),
    properties String,
    revenue Float64
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(event_date)
ORDER BY (event_type, user_id, event_time)
TTL event_date + INTERVAL 90 DAY
SETTINGS index_granularity = 8192;
```

### ReplacingMergeTree (dedup by sort key)

```sql
CREATE TABLE users (
    user_id UInt64,
    name String,
    email String,
    updated_at DateTime
) ENGINE = ReplacingMergeTree(updated_at)
ORDER BY user_id;

-- Query with dedup (FINAL forces merge)
SELECT * FROM users FINAL WHERE user_id = 123;
```

### AggregatingMergeTree (pre-aggregated)

```sql
CREATE TABLE events_agg (
    event_date Date,
    event_type LowCardinality(String),
    count AggregateFunction(count),
    revenue AggregateFunction(sum, Float64)
) ENGINE = AggregatingMergeTree()
PARTITION BY toYYYYMM(event_date)
ORDER BY (event_date, event_type);
```

### SummingMergeTree (auto-sum numeric columns)

```sql
CREATE TABLE daily_stats (
    date Date,
    source String,
    impressions UInt64,
    clicks UInt64,
    cost Float64
) ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (date, source);
```

## Queries

```sql
-- Count with filters
SELECT count() FROM events
WHERE event_date >= '2024-01-01' AND event_type = 'purchase';

-- Aggregation
SELECT
    toStartOfHour(event_time) AS hour,
    event_type,
    count() AS cnt,
    sum(revenue) AS total_revenue,
    avg(revenue) AS avg_revenue,
    quantile(0.95)(revenue) AS p95_revenue
FROM events
WHERE event_date = today()
GROUP BY hour, event_type
ORDER BY hour, cnt DESC;

-- Window functions
SELECT
    user_id,
    event_time,
    revenue,
    runningAccumulate(sum_state) AS running_total
FROM (
    SELECT
        user_id, event_time, revenue,
        sumState(revenue) OVER (PARTITION BY user_id ORDER BY event_time) AS sum_state
    FROM events
);

-- Array operations
SELECT
    groupArray(event_type) AS types,
    groupUniqArray(user_id) AS unique_users,
    arrayJoin(splitByChar(',', tags)) AS tag
FROM events;

-- Approximate distinct count (HyperLogLog)
SELECT uniq(user_id) FROM events;
SELECT uniqExact(user_id) FROM events;   -- exact but slower
```

## Materialized Views

```sql
-- Continuous aggregation
CREATE MATERIALIZED VIEW events_hourly_mv
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(hour)
ORDER BY (hour, event_type)
AS SELECT
    toStartOfHour(event_time) AS hour,
    event_type,
    count() AS event_count,
    sum(revenue) AS total_revenue
FROM events
GROUP BY hour, event_type;

-- Materialized view with AggregatingMergeTree
CREATE MATERIALIZED VIEW uniq_users_mv
ENGINE = AggregatingMergeTree()
ORDER BY (date, source)
AS SELECT
    toDate(event_time) AS date,
    event_type AS source,
    uniqState(user_id) AS users
FROM events
GROUP BY date, source;

-- Query aggregating MV
SELECT date, source, uniqMerge(users) AS unique_users
FROM uniq_users_mv
GROUP BY date, source;
```

## Distributed Tables

```sql
-- Create local table on each shard
CREATE TABLE events_local ON CLUSTER my_cluster (
    event_date Date,
    user_id UInt64,
    event_type String,
    revenue Float64
) ENGINE = ReplicatedMergeTree('/clickhouse/tables/{shard}/events', '{replica}')
PARTITION BY toYYYYMM(event_date)
ORDER BY (event_type, user_id);

-- Create distributed table
CREATE TABLE events_distributed ON CLUSTER my_cluster
AS events_local
ENGINE = Distributed(my_cluster, default, events_local, rand());

-- Sharding key options:
-- rand()                  — random distribution
-- sipHash64(user_id)     — hash-based (same user on same shard)
-- user_id % 3            — modulo
```

## Data Ingestion

```bash
# Insert from CSV
clickhouse-client --query="INSERT INTO events FORMAT CSV" < data.csv

# Insert from TSV
clickhouse-client --query="INSERT INTO events FORMAT TabSeparated" < data.tsv

# Insert JSON
clickhouse-client --query="INSERT INTO events FORMAT JSONEachRow" <<'EOF'
{"event_date":"2024-01-01","user_id":1,"event_type":"click","revenue":0}
{"event_date":"2024-01-01","user_id":2,"event_type":"purchase","revenue":49.99}
EOF

# HTTP bulk insert
curl 'http://localhost:8123/?query=INSERT+INTO+events+FORMAT+JSONEachRow' \
  --data-binary @events.json

# Insert from S3
INSERT INTO events
SELECT * FROM s3('https://bucket.s3.amazonaws.com/data/*.parquet', 'Parquet');
```

## Administration

```sql
-- System tables
SELECT * FROM system.parts WHERE table = 'events';
SELECT * FROM system.merges;
SELECT * FROM system.mutations;
SELECT * FROM system.replicas;
SELECT * FROM system.query_log ORDER BY event_time DESC LIMIT 10;

-- Optimize (force merge)
OPTIMIZE TABLE events FINAL;
OPTIMIZE TABLE events PARTITION 202401 FINAL;

-- Alter table
ALTER TABLE events ADD COLUMN category String DEFAULT 'unknown';
ALTER TABLE events DROP COLUMN properties;
ALTER TABLE events MODIFY TTL event_date + INTERVAL 180 DAY;

-- Mutations (async updates/deletes)
ALTER TABLE events DELETE WHERE event_date < '2023-01-01';
ALTER TABLE events UPDATE revenue = revenue * 1.1 WHERE event_type = 'purchase';

-- Check mutation progress
SELECT * FROM system.mutations WHERE is_done = 0;
```

## Tips

- Design the `ORDER BY` clause to match your most common query filters; it defines the primary index
- Use `LowCardinality(String)` for columns with under ~10,000 distinct values for major compression gains
- Partition by month (`toYYYYMM`) as a default; too many partitions hurts merge performance
- Use materialized views for continuous pre-aggregation instead of querying raw data repeatedly
- Prefer approximate functions (`uniq`, `quantile`) over exact ones for interactive dashboards
- Avoid `SELECT *` in production; columnar storage means unused columns are wasted I/O
- Use `FINAL` keyword with ReplacingMergeTree sparingly; it forces merge and slows queries
- Batch inserts in blocks of at least 1,000-10,000 rows; per-row inserts create too many parts
- Monitor `system.parts` count; too many parts means merges are falling behind ingestion
- Set TTL on tables to auto-expire old data rather than running manual DELETE mutations
- Use `PREWHERE` for highly selective filters; ClickHouse can read fewer columns in the first pass

## See Also

postgresql, mongodb, victoriametrics, prometheus, cassandra

## References

- [ClickHouse Documentation](https://clickhouse.com/docs/)
- [ClickHouse MergeTree Engine](https://clickhouse.com/docs/en/engines/table-engines/mergetree-family/mergetree)
- [ClickHouse SQL Reference](https://clickhouse.com/docs/en/sql-reference)
- [ClickHouse Distributed Tables](https://clickhouse.com/docs/en/engines/table-engines/special/distributed)
- [ClickHouse Performance Tips](https://clickhouse.com/docs/en/operations/tips)
