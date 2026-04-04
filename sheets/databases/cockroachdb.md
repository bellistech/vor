# CockroachDB (Distributed SQL Database)

Distributed SQL database built for cloud-native applications with automatic sharding, strong consistency via Raft, and multi-region survivability.

## Installation and Startup

```bash
# Single-node for development
cockroach start-single-node --insecure --store=node1 --listen-addr=localhost:26257 \
  --http-addr=localhost:8080

# Multi-node cluster
cockroach start --insecure --store=node1 --listen-addr=localhost:26257 \
  --join=localhost:26257,localhost:26258,localhost:26259 --http-addr=localhost:8080
cockroach start --insecure --store=node2 --listen-addr=localhost:26258 \
  --join=localhost:26257,localhost:26258,localhost:26259 --http-addr=localhost:8081
cockroach start --insecure --store=node3 --listen-addr=localhost:26259 \
  --join=localhost:26257,localhost:26258,localhost:26259 --http-addr=localhost:8082

# Initialize cluster
cockroach init --insecure --host=localhost:26257

# Docker
docker run -d --name cockroach -p 26257:26257 -p 8080:8080 \
  cockroachdb/cockroach:latest start-single-node --insecure
```

## SQL Shell

```bash
cockroach sql --insecure --host=localhost:26257
cockroach sql --insecure --host=localhost:26257 --database=mydb
cockroach sql --url 'postgresql://root@localhost:26257/mydb?sslmode=disable'

# Execute SQL directly
cockroach sql --insecure -e "SELECT version()"
cockroach sql --insecure -e "SHOW DATABASES"
```

## Database and Table Operations

```sql
-- Create database
CREATE DATABASE myapp;
USE myapp;

-- Create table with UUID primary key (recommended)
CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email       STRING NOT NULL UNIQUE,
    name        STRING NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT now(),
    region      STRING NOT NULL,
    INDEX idx_users_region (region)
);

-- Serial (auto-increment) — use unique_rowid() instead of SERIAL
CREATE TABLE events (
    id          INT8 PRIMARY KEY DEFAULT unique_rowid(),
    user_id     UUID REFERENCES users(id),
    event_type  STRING NOT NULL,
    payload     JSONB,
    created_at  TIMESTAMPTZ DEFAULT now()
);

-- Hash-sharded index (prevents hot ranges on sequential writes)
CREATE TABLE logs (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ts         TIMESTAMPTZ NOT NULL DEFAULT now(),
    message    STRING,
    INDEX idx_logs_ts (ts) USING HASH
);
```

## Range and Replication

```sql
-- Show ranges for a table
SHOW RANGES FROM TABLE users;
SHOW RANGES FROM INDEX users@primary;

-- Range distribution
SELECT range_id, start_key, end_key, lease_holder, replicas
FROM [SHOW RANGES FROM TABLE users];

-- Configure replication factor
ALTER DATABASE myapp CONFIGURE ZONE USING num_replicas = 5;
ALTER TABLE users CONFIGURE ZONE USING num_replicas = 5;

-- Configure garbage collection TTL
ALTER TABLE events CONFIGURE ZONE USING gc.ttlseconds = 86400;

-- Show zone configuration
SHOW ZONE CONFIGURATION FOR TABLE users;
SHOW ALL ZONE CONFIGURATIONS;
```

## Multi-Region

```sql
-- Add regions to database
ALTER DATABASE myapp SET PRIMARY REGION "us-east1";
ALTER DATABASE myapp ADD REGION "us-west1";
ALTER DATABASE myapp ADD REGION "eu-west1";

-- Show regions
SHOW REGIONS FROM DATABASE myapp;

-- REGIONAL BY TABLE — all data in primary region
CREATE TABLE config (
    key   STRING PRIMARY KEY,
    value STRING
) LOCALITY REGIONAL BY TABLE IN PRIMARY REGION;

-- REGIONAL BY ROW — row-level region pinning
ALTER TABLE users ADD COLUMN crdb_region crdb_internal_region
    AS (CASE WHEN region = 'eu' THEN 'eu-west1'
             WHEN region = 'us-west' THEN 'us-west1'
             ELSE 'us-east1' END) STORED;
ALTER TABLE users SET LOCALITY REGIONAL BY ROW;

-- GLOBAL — low-latency reads from any region (higher write latency)
CREATE TABLE currencies (
    code STRING PRIMARY KEY,
    name STRING
) LOCALITY GLOBAL;

-- Survivability goals
ALTER DATABASE myapp SURVIVE REGION FAILURE;   -- 3+ regions required
ALTER DATABASE myapp SURVIVE ZONE FAILURE;     -- default
```

## Transactions and Contention

```sql
-- Standard transactions (serializable by default)
BEGIN;
INSERT INTO users (email, name, region) VALUES ('a@b.com', 'Alice', 'us-east');
INSERT INTO events (user_id, event_type) VALUES (
    (SELECT id FROM users WHERE email = 'a@b.com'), 'signup');
COMMIT;

-- Explicit priority
BEGIN PRIORITY HIGH;
UPDATE accounts SET balance = balance - 100 WHERE id = 1;
COMMIT;

-- AS OF SYSTEM TIME — read historical data (follower reads)
SELECT * FROM users AS OF SYSTEM TIME '-10s';

-- Follower reads for multi-region (read from nearest replica)
SELECT * FROM users AS OF SYSTEM TIME follower_read_timestamp();

-- Show contention
SELECT * FROM crdb_internal.cluster_contended_tables;
SELECT * FROM crdb_internal.cluster_contended_indexes;
```

## Change Data Capture (CDC)

```sql
-- Enable rangefeed (required for CDC)
SET CLUSTER SETTING kv.rangefeed.enabled = true;

-- Create changefeed to Kafka
CREATE CHANGEFEED FOR TABLE users, events
INTO 'kafka://broker:9092?topic_prefix=cdc_'
WITH updated, resolved = '10s',
     format = 'json',
     diff;

-- Changefeed to cloud storage
CREATE CHANGEFEED FOR TABLE users
INTO 's3://bucket/cdc?AWS_ACCESS_KEY_ID=key&AWS_SECRET_ACCESS_KEY=secret'
WITH format = 'json', resolved = '30s';

-- Changefeed to webhook
CREATE CHANGEFEED FOR TABLE users
INTO 'webhook-https://myapp.com/cdc'
WITH format = 'json', updated;

-- Monitor changefeeds
SHOW CHANGEFEED JOBS;
PAUSE JOB <job_id>;
RESUME JOB <job_id>;
CANCEL JOB <job_id>;
```

## Backup and Restore

```bash
# Full backup to cloud storage
cockroach sql --insecure -e "
BACKUP DATABASE myapp INTO 's3://bucket/backups'
WITH revision_history;"

# Incremental backup
cockroach sql --insecure -e "
BACKUP DATABASE myapp INTO LATEST IN 's3://bucket/backups';"

# Scheduled backups
cockroach sql --insecure -e "
CREATE SCHEDULE daily_backup FOR
BACKUP DATABASE myapp INTO 's3://bucket/backups'
RECURRING '@daily'
WITH SCHEDULE OPTIONS first_run = 'now';"
```

```sql
-- Restore
RESTORE DATABASE myapp FROM LATEST IN 's3://bucket/backups';

-- Point-in-time restore
RESTORE DATABASE myapp FROM LATEST IN 's3://bucket/backups'
AS OF SYSTEM TIME '2024-01-15 12:00:00';

-- Show backups
SHOW BACKUPS IN 's3://bucket/backups';
SHOW BACKUP LATEST IN 's3://bucket/backups';

-- Show scheduled jobs
SHOW SCHEDULES;
```

## Node and Cluster Management

```bash
# Node status
cockroach node status --insecure --host=localhost:26257

# Decommission a node (safely drain data)
cockroach node decommission 4 --insecure --host=localhost:26257

# Drain node for maintenance
cockroach node drain --insecure --host=localhost:26257

# Cluster settings
cockroach sql --insecure -e "SHOW ALL CLUSTER SETTINGS"
cockroach sql --insecure -e "SET CLUSTER SETTING server.time_until_store_dead = '5m'"
```

## Performance and Debugging

```sql
-- Explain query plan
EXPLAIN SELECT * FROM users WHERE region = 'us-east';
EXPLAIN ANALYZE SELECT * FROM users WHERE region = 'us-east';
EXPLAIN (DISTSQL) SELECT * FROM users WHERE region = 'us-east';

-- Statement statistics
SELECT * FROM crdb_internal.statement_statistics
ORDER BY (statistics->'statistics'->'cnt')::INT DESC
LIMIT 10;

-- Active queries
SHOW QUERIES;
CANCEL QUERY '<query_id>';

-- Active sessions
SHOW SESSIONS;

-- Table statistics
SHOW STATISTICS FOR TABLE users;
CREATE STATISTICS stats_users FROM users;

-- Index usage
SELECT * FROM crdb_internal.index_usage_statistics
WHERE table_name = 'users';

-- Ranges and leaseholders
SELECT range_id, lease_holder, array_length(replicas, 1) AS num_replicas
FROM [SHOW RANGES FROM TABLE users];
```

## Schema Changes

```sql
-- Online schema changes (non-blocking)
ALTER TABLE users ADD COLUMN phone STRING;
ALTER TABLE users DROP COLUMN phone;
CREATE INDEX CONCURRENTLY idx_name ON users(name);

-- Show running schema changes
SHOW JOBS WHEN COMPLETE (
    SELECT job_id FROM [SHOW JOBS]
    WHERE job_type = 'SCHEMA CHANGE' AND status = 'running'
);

-- Import data
IMPORT INTO users CSV DATA ('s3://bucket/users.csv')
WITH delimiter = ',', skip = '1';
```

## Tips

- Use UUID primary keys with gen_random_uuid() to distribute writes evenly across ranges
- Avoid sequential primary keys (SERIAL); they create write hot spots on a single range
- Use hash-sharded indexes for timestamp-ordered data to prevent hot ranges
- REGIONAL BY ROW gives per-row region pinning; use computed columns for automatic assignment
- Follower reads (AS OF SYSTEM TIME) reduce cross-region latency for stale-tolerant queries
- Set SURVIVE REGION FAILURE for critical databases with 3+ regions deployed
- CockroachDB is serializable by default; design for retry loops on transaction contention
- Use EXPLAIN ANALYZE to identify distributed query plans and spot full table scans
- Changefeeds with resolved timestamps enable exactly-once downstream processing
- Monitor the DB Console (port 8080) for hot ranges, slow queries, and replication lag
- Prefer BACKUP with revision_history for point-in-time restore capability

## See Also

- postgresql
- cassandra
- etcd

## References

- [CockroachDB Documentation](https://www.cockroachlabs.com/docs/)
- [Architecture Overview](https://www.cockroachlabs.com/docs/stable/architecture/overview.html)
- [Multi-Region Capabilities](https://www.cockroachlabs.com/docs/stable/multiregion-overview.html)
- [Change Data Capture](https://www.cockroachlabs.com/docs/stable/change-data-capture-overview.html)
- [CockroachDB GitHub](https://github.com/cockroachdb/cockroach)
