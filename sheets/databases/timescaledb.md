# TimescaleDB (Time-Series Database)

PostgreSQL extension for time-series data with automatic partitioning, continuous aggregates, compression, and real-time analytics.

## Installation

```bash
# PostgreSQL extension
CREATE EXTENSION IF NOT EXISTS timescaledb;

# Docker
docker run -d --name timescaledb -p 5432:5432 \
  -e POSTGRES_PASSWORD=password \
  timescale/timescaledb:latest-pg16

# Verify installation
SELECT extversion FROM pg_extension WHERE extname = 'timescaledb';
```

## Hypertables

```sql
-- Create a regular table first
CREATE TABLE metrics (
    time        TIMESTAMPTZ NOT NULL,
    device_id   TEXT NOT NULL,
    cpu         DOUBLE PRECISION,
    memory      DOUBLE PRECISION,
    disk_io     DOUBLE PRECISION
);

-- Convert to hypertable (auto-partitions by time)
SELECT create_hypertable('metrics', 'time');

-- Custom chunk interval (default is 7 days)
SELECT create_hypertable('metrics', 'time',
    chunk_time_interval => INTERVAL '1 day');

-- With space partitioning (hash on device_id, 4 partitions)
SELECT create_hypertable('metrics', 'time',
    partitioning_column => 'device_id',
    number_partitions => 4);

-- Show hypertable info
SELECT * FROM timescaledb_information.hypertables;
SELECT * FROM timescaledb_information.chunks
    WHERE hypertable_name = 'metrics';
```

## Inserting Data

```sql
-- Standard INSERT (same as PostgreSQL)
INSERT INTO metrics (time, device_id, cpu, memory, disk_io)
VALUES (NOW(), 'server-01', 72.5, 85.3, 120.7);

-- Batch insert
INSERT INTO metrics VALUES
    (NOW() - INTERVAL '1 hour', 'server-01', 68.2, 80.1, 95.3),
    (NOW() - INTERVAL '30 min', 'server-01', 71.0, 82.4, 102.8),
    (NOW(), 'server-01', 72.5, 85.3, 120.7);

-- COPY for bulk loading (fastest)
COPY metrics FROM '/data/metrics.csv' CSV HEADER;

-- Upsert with ON CONFLICT
INSERT INTO metrics (time, device_id, cpu, memory, disk_io)
VALUES (NOW(), 'server-01', 72.5, 85.3, 120.7)
ON CONFLICT (time, device_id) DO UPDATE
SET cpu = EXCLUDED.cpu, memory = EXCLUDED.memory;
```

## time_bucket Queries

```sql
-- Average CPU per 5-minute bucket
SELECT time_bucket('5 minutes', time) AS bucket,
       device_id,
       AVG(cpu) AS avg_cpu,
       MAX(cpu) AS max_cpu,
       MIN(cpu) AS min_cpu
FROM metrics
WHERE time > NOW() - INTERVAL '1 hour'
GROUP BY bucket, device_id
ORDER BY bucket DESC;

-- 1-hour buckets with count
SELECT time_bucket('1 hour', time) AS hour,
       COUNT(*) AS readings,
       AVG(memory) AS avg_memory
FROM metrics
WHERE time > NOW() - INTERVAL '24 hours'
GROUP BY hour
ORDER BY hour;

-- time_bucket_gapfill — fill missing intervals
SELECT time_bucket_gapfill('1 hour', time) AS hour,
       device_id,
       COALESCE(AVG(cpu), 0) AS avg_cpu,
       locf(AVG(cpu)) AS last_observation_carried_forward,
       interpolate(AVG(cpu)) AS interpolated
FROM metrics
WHERE time BETWEEN '2024-01-01' AND '2024-01-02'
GROUP BY hour, device_id
ORDER BY hour;
```

## Aggregate Functions

```sql
-- first() and last() — time-weighted values
SELECT device_id,
       first(cpu, time) AS first_cpu,
       last(cpu, time) AS last_cpu,
       last(cpu, time) - first(cpu, time) AS cpu_change
FROM metrics
WHERE time > NOW() - INTERVAL '1 hour'
GROUP BY device_id;

-- Percentile approximation
SELECT time_bucket('1 hour', time) AS hour,
       approx_percentile(0.95, percentile_agg(cpu)) AS p95_cpu,
       approx_percentile(0.99, percentile_agg(cpu)) AS p99_cpu
FROM metrics
WHERE time > NOW() - INTERVAL '24 hours'
GROUP BY hour
ORDER BY hour;

-- Histogram
SELECT time_bucket('1 hour', time) AS hour,
       histogram(cpu, 0, 100, 10) AS cpu_histogram
FROM metrics
WHERE time > NOW() - INTERVAL '6 hours'
GROUP BY hour;
```

## Continuous Aggregates

```sql
-- Create a continuous aggregate (materialized view with auto-refresh)
CREATE MATERIALIZED VIEW metrics_hourly
WITH (timescaledb.continuous) AS
SELECT time_bucket('1 hour', time) AS bucket,
       device_id,
       AVG(cpu) AS avg_cpu,
       MAX(cpu) AS max_cpu,
       MIN(cpu) AS min_cpu,
       COUNT(*) AS sample_count
FROM metrics
GROUP BY bucket, device_id
WITH NO DATA;

-- Add refresh policy (refresh hourly, cover last 3 hours)
SELECT add_continuous_aggregate_policy('metrics_hourly',
    start_offset    => INTERVAL '3 hours',
    end_offset      => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour');

-- Manual refresh
CALL refresh_continuous_aggregate('metrics_hourly',
    '2024-01-01', '2024-02-01');

-- Query the aggregate (automatic real-time aggregation)
SELECT * FROM metrics_hourly
WHERE bucket > NOW() - INTERVAL '24 hours'
ORDER BY bucket DESC;

-- Hierarchical aggregates (aggregate of aggregate)
CREATE MATERIALIZED VIEW metrics_daily
WITH (timescaledb.continuous) AS
SELECT time_bucket('1 day', bucket) AS day,
       device_id,
       AVG(avg_cpu) AS avg_cpu,
       MAX(max_cpu) AS max_cpu
FROM metrics_hourly
GROUP BY day, device_id
WITH NO DATA;
```

## Compression

```sql
-- Enable compression on hypertable
ALTER TABLE metrics SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'device_id',
    timescaledb.compress_orderby = 'time DESC'
);

-- Add compression policy (compress chunks older than 7 days)
SELECT add_compression_policy('metrics', INTERVAL '7 days');

-- Manual compression
SELECT compress_chunk(c.chunk_name)
FROM timescaledb_information.chunks c
WHERE c.hypertable_name = 'metrics'
  AND c.range_end < NOW() - INTERVAL '7 days'
  AND NOT c.is_compressed;

-- Check compression stats
SELECT hypertable_name,
       before_compression_total_bytes,
       after_compression_total_bytes,
       (1 - after_compression_total_bytes::numeric /
            before_compression_total_bytes) * 100 AS compression_ratio
FROM hypertable_compression_stats('metrics');

-- Decompress a chunk (for updates)
SELECT decompress_chunk('<chunk_name>');
```

## Retention Policies

```sql
-- Drop chunks older than 90 days automatically
SELECT add_retention_policy('metrics', INTERVAL '90 days');

-- Manual chunk drop
SELECT drop_chunks('metrics', older_than => INTERVAL '90 days');

-- Show active policies
SELECT * FROM timescaledb_information.jobs
WHERE proc_name IN ('policy_retention', 'policy_compression',
                    'policy_refresh_continuous_aggregate');

-- Remove a policy
SELECT remove_retention_policy('metrics');
```

## Data Tiering and Chunk Management

```sql
-- List all chunks with sizes
SELECT chunk_name,
       range_start,
       range_end,
       is_compressed,
       pg_size_pretty(total_bytes) AS size
FROM timescaledb_information.chunks
WHERE hypertable_name = 'metrics'
ORDER BY range_start DESC;

-- Move chunk to different tablespace
SELECT move_chunk('<chunk_name>', 'slow_tablespace');

-- Hypertable size
SELECT pg_size_pretty(hypertable_size('metrics'));
SELECT * FROM hypertable_detailed_size('metrics');

-- Approximate row count (fast)
SELECT approximate_row_count('metrics');
```

## Indexes

```sql
-- Default index on (time DESC) is created automatically
-- Add composite indexes for common query patterns
CREATE INDEX ON metrics (device_id, time DESC);

-- Partial index for recent hot data
CREATE INDEX ON metrics (time DESC)
WHERE time > NOW() - INTERVAL '7 days';

-- Expression index on time_bucket
CREATE INDEX ON metrics (time_bucket('1 hour', time), device_id);
```

## Administration

```bash
# psql connection
psql -h localhost -U postgres -d mydb

# Useful diagnostic queries
```

```sql
-- Check TimescaleDB version
SELECT timescaledb_pre_restore();
SELECT timescaledb_post_restore();

-- Job history
SELECT * FROM timescaledb_information.job_stats;

-- Chunk statistics
SELECT * FROM chunks_detailed_size('metrics');
```

## Tips

- Always set an appropriate chunk_time_interval based on your ingest rate; too small creates overhead, too large reduces pruning benefits
- Use compress_segmentby on low-cardinality columns you frequently filter by (e.g., device_id)
- Continuous aggregates with real-time aggregation give you always-fresh data without manual refresh
- Combine retention policies with continuous aggregates to keep summaries while dropping raw data
- Use time_bucket_gapfill with locf() or interpolate() for complete time series without gaps
- Batch inserts or COPY for bulk loading; individual INSERTs are slower by orders of magnitude
- Add composite indexes on (dimension_column, time DESC) for queries that filter by dimension
- Monitor chunk sizes with hypertable_detailed_size() to verify compression ratios
- Use approximate_row_count() instead of COUNT(*) for large tables
- Compression typically achieves 90-95% reduction; always set compress_orderby to time DESC
- Hierarchical continuous aggregates (daily from hourly) are more efficient than aggregating raw data
- Set chunk intervals so each chunk fits in about 25% of available memory for best performance

## See Also

- postgresql
- clickhouse
- sql

## References

- [TimescaleDB Documentation](https://docs.timescale.com/)
- [Hypertable Best Practices](https://docs.timescale.com/use-timescale/latest/hypertables/about-hypertables/)
- [Continuous Aggregates](https://docs.timescale.com/use-timescale/latest/continuous-aggregates/)
- [Compression Guide](https://docs.timescale.com/use-timescale/latest/compression/)
- [TimescaleDB GitHub](https://github.com/timescale/timescaledb)
