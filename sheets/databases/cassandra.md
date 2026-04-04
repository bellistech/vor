# Cassandra (Wide-Column Database)

Distributed wide-column NoSQL database designed for high write throughput and linear scalability, using consistent hashing, tunable consistency levels, and a log-structured merge-tree storage engine.

## cqlsh (CQL Shell)

```bash
# Connect
cqlsh
cqlsh cassandra.example.com 9042
cqlsh --ssl -u cassandra -p cassandra

# Useful cqlsh commands
DESCRIBE KEYSPACES;
DESCRIBE KEYSPACE my_keyspace;
DESCRIBE TABLES;
DESCRIBE TABLE users;
CONSISTENCY QUORUM;                      # set default CL for session
TRACING ON;                              # enable request tracing
EXPAND ON;                               # vertical output format
SOURCE '/path/to/script.cql';
COPY users TO '/tmp/users.csv' WITH HEADER = TRUE;
COPY users FROM '/tmp/users.csv' WITH HEADER = TRUE;
```

## Keyspace & Tables

```sql
-- Create keyspace (replication strategy)
CREATE KEYSPACE my_app WITH replication = {
  'class': 'NetworkTopologyStrategy',
  'dc1': 3,
  'dc2': 3
};

-- SimpleStrategy (single DC only)
CREATE KEYSPACE dev WITH replication = {
  'class': 'SimpleStrategy',
  'replication_factor': 3
};

USE my_app;

-- Create table (partition key + clustering columns)
CREATE TABLE events (
    tenant_id UUID,
    event_date DATE,
    event_time TIMESTAMP,
    event_id TIMEUUID,
    event_type TEXT,
    payload TEXT,
    PRIMARY KEY ((tenant_id, event_date), event_time, event_id)
) WITH CLUSTERING ORDER BY (event_time DESC, event_id DESC)
AND compaction = {
    'class': 'TimeWindowCompactionStrategy',
    'compaction_window_size': 1,
    'compaction_window_unit': 'DAYS'
}
AND default_time_to_live = 7776000         -- 90 days
AND gc_grace_seconds = 864000;             -- 10 days

-- Static columns (one value per partition)
CREATE TABLE user_events (
    user_id UUID,
    event_id TIMEUUID,
    username TEXT STATIC,                   -- shared across partition
    event_data TEXT,
    PRIMARY KEY (user_id, event_id)
);
```

## CRUD Operations

```sql
-- Insert
INSERT INTO events (tenant_id, event_date, event_time, event_id, event_type, payload)
VALUES (uuid(), '2024-01-15', toTimestamp(now()), now(), 'click', '{"page":"/home"}');

-- Insert with TTL
INSERT INTO events (tenant_id, event_date, event_time, event_id, event_type, payload)
VALUES (uuid(), '2024-01-15', toTimestamp(now()), now(), 'click', '{}')
USING TTL 86400;                           -- expires in 24 hours

-- Insert if not exists (lightweight transaction)
INSERT INTO users (user_id, email, name)
VALUES (uuid(), 'alice@example.com', 'Alice')
IF NOT EXISTS;

-- Select (must include partition key)
SELECT * FROM events
WHERE tenant_id = 550e8400-e29b-41d4-a716-446655440000
  AND event_date = '2024-01-15'
ORDER BY event_time DESC
LIMIT 100;

-- Select with token range (for full scan)
SELECT * FROM events
WHERE token(tenant_id, event_date) > -9223372036854775808
  AND token(tenant_id, event_date) < 9223372036854775807;

-- Update
UPDATE events
SET payload = '{"page":"/updated"}'
WHERE tenant_id = 550e8400-e29b-41d4-a716-446655440000
  AND event_date = '2024-01-15'
  AND event_time = '2024-01-15 10:30:00'
  AND event_id = now();

-- Delete
DELETE FROM events
WHERE tenant_id = 550e8400-e29b-41d4-a716-446655440000
  AND event_date = '2024-01-15';

-- Batch (single partition preferred)
BEGIN BATCH
  INSERT INTO users (user_id, email) VALUES (uuid(), 'bob@example.com');
  INSERT INTO user_events (user_id, event_id, event_data) VALUES (uuid(), now(), 'signup');
APPLY BATCH;
```

## Consistency Levels

```sql
-- Per-query consistency
SELECT * FROM events WHERE ... USING CONSISTENCY QUORUM;
INSERT INTO events (...) VALUES (...) USING CONSISTENCY LOCAL_QUORUM;

-- Consistency Levels:
-- ANY         — write: at least 1 node (including hinted handoff)
-- ONE         — 1 replica
-- TWO         — 2 replicas
-- THREE       — 3 replicas
-- QUORUM      — floor(RF/2) + 1 replicas across all DCs
-- LOCAL_QUORUM — floor(RF/2) + 1 replicas in local DC
-- EACH_QUORUM — quorum in each DC (writes only)
-- ALL         — all replicas
-- LOCAL_ONE   — 1 replica in local DC
-- SERIAL      — for LWT reads (linearizable)
-- LOCAL_SERIAL — LWT reads in local DC

-- Strong consistency guarantee:
-- Write CL + Read CL > Replication Factor
-- Common: LOCAL_QUORUM write + LOCAL_QUORUM read (RF=3: 2+2>3)
```

## Indexes

```sql
-- Secondary index (use sparingly)
CREATE INDEX ON events (event_type);

-- SAI (Storage-Attached Index, Cassandra 5.0+)
CREATE CUSTOM INDEX ON events (event_type)
USING 'StorageAttachedIndex';

CREATE CUSTOM INDEX ON events (payload)
USING 'StorageAttachedIndex'
WITH OPTIONS = {'case_sensitive': 'false', 'normalize': 'true'};

-- Materialized view (auto-maintained denormalization)
CREATE MATERIALIZED VIEW events_by_type AS
    SELECT * FROM events
    WHERE event_type IS NOT NULL
      AND tenant_id IS NOT NULL
      AND event_date IS NOT NULL
      AND event_time IS NOT NULL
      AND event_id IS NOT NULL
    PRIMARY KEY (event_type, tenant_id, event_date, event_time, event_id);
```

## nodetool (Administration)

```bash
# Cluster status
nodetool status
nodetool ring
nodetool describecluster
nodetool info

# Performance
nodetool tpstats                         # thread pool stats
nodetool proxyhistograms                 # coordinator latencies
nodetool tablehistograms my_app events   # table-level latencies
nodetool tablestats my_app events        # table stats

# Repair
nodetool repair my_app                   # full repair
nodetool repair my_app events -pr        # primary range only
nodetool repair --full my_app            # full (not incremental)

# Compaction
nodetool compactionstats                 # current compactions
nodetool compact my_app events           # force compaction

# Streaming
nodetool netstats                        # network streaming status

# Flush & drain
nodetool flush my_app events             # flush memtables to disk
nodetool drain                           # flush all + stop accepting writes

# Decommission & remove
nodetool decommission                    # gracefully remove this node
nodetool removenode <host-id>            # remove a dead node

# Cleanup (after topology change)
nodetool cleanup                         # remove data no longer owned
```

## Data Modeling Patterns

```sql
-- Time series with bucketing
-- Partition key: (sensor_id, date) — bounds partition size
-- Clustering: timestamp DESC — latest first
CREATE TABLE readings (
    sensor_id UUID,
    date DATE,
    ts TIMESTAMP,
    value DOUBLE,
    PRIMARY KEY ((sensor_id, date), ts)
) WITH CLUSTERING ORDER BY (ts DESC);

-- Reverse lookup (materialized view pattern)
CREATE TABLE users_by_email (
    email TEXT,
    user_id UUID,
    name TEXT,
    PRIMARY KEY (email)
);
-- Manually maintain on write to users table
```

## Tips

- Always include the full partition key in queries; Cassandra cannot efficiently scan across partitions
- Keep partition size under 100 MB and 100,000 rows to avoid hot spots and slow reads
- Use composite partition keys with a time bucket (date, month) for time-series data to bound partition growth
- Use `LOCAL_QUORUM` for both reads and writes with RF=3 for strong consistency within a datacenter
- Prefer `TimeWindowCompactionStrategy` for time-series workloads; it reduces write amplification on TTL data
- Avoid secondary indexes on high-cardinality columns; use materialized views or manual denormalization instead
- Run `nodetool repair` regularly (at least within `gc_grace_seconds`) to prevent zombie data from tombstones
- Use `BATCH` only for atomicity across tables in the same partition; multi-partition batches are an anti-pattern
- Monitor `nodetool tpstats` for dropped mutations; this indicates the cluster is overloaded
- Set `gc_grace_seconds` to match your repair schedule; lowering it without regular repair risks data resurrection
- Use TTL on tables to auto-expire data instead of manual deletes, which create expensive tombstones

## See Also

mongodb, redis, clickhouse, etcd, postgresql

## References

- [Apache Cassandra Documentation](https://cassandra.apache.org/doc/latest/)
- [CQL Reference](https://cassandra.apache.org/doc/latest/cassandra/cql/)
- [Cassandra Data Modeling](https://cassandra.apache.org/doc/latest/cassandra/data_modeling/)
- [nodetool Reference](https://cassandra.apache.org/doc/latest/cassandra/tools/nodetool/nodetool.html)
- [Cassandra Architecture](https://cassandra.apache.org/doc/latest/cassandra/architecture/)
