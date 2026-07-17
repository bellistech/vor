# AWS Database Selection & Performance (SAA-C03 Tasks 3.3 + 4.3)

> Determine high-performing, cost-optimized database solutions: RDS vs Aurora, read replicas vs Multi-AZ, DynamoDB capacity and access design, ElastiCache, and the purpose-built database picker.

## Purpose-Built Picker (Answer by Data Shape)

| Data/access shape | Service |
|---|---|
| Relational OLTP, managed, standard engines | RDS (MySQL, PostgreSQL, MariaDB, Oracle, SQL Server, Db2) |
| Relational, cloud-native, max performance/HA | Aurora (MySQL/PostgreSQL-compatible) |
| Key-value/document, ms at any scale, serverless | DynamoDB |
| Microsecond reads on DynamoDB | DAX |
| In-memory cache/session/leaderboard/pub-sub | ElastiCache (Redis OSS / Valkey / Memcached) |
| Columnar analytics warehouse (OLAP) | Redshift |
| MongoDB-compatible document | DocumentDB |
| Graph (relationships, fraud rings, recommendations) | Neptune |
| Cassandra-compatible wide column | Keyspaces |
| Time series (IoT metrics, telemetry) | Timestream |
| Immutable verifiable ledger | QLDB (legacy — being sunset; audit trails now often DynamoDB+hash or Aurora) |
| Search/log analytics | OpenSearch Service |
| OLTP vs OLAP trap | OLTP → RDS/Aurora/DynamoDB; OLAP/reporting → Redshift/Athena, or a READ REPLICA to isolate reports |

## RDS Deep Dive

```
# Managed EC2+EBS under the hood: AWS handles patching, backups,
#   failover; you pick instance class + storage (gp3/io1/io2;
#   Provisioned IOPS for OLTP; storage autoscaling threshold)
# Multi-AZ INSTANCE — synchronous standby in second AZ; automatic DNS
#   failover 60–120 s (crash, AZ loss, patching); standby INVISIBLE to
#   reads; enable without downtime
# Multi-AZ DB CLUSTER — 1 writer + 2 READABLE standbys, semi-sync,
#   failover typically <35 s (MySQL/PostgreSQL only)
# Read replicas — ASYNC copies (eventual consistency, replica lag!):
#   up to 15 (aurora) / 5 (RDS); same-AZ, cross-AZ, or CROSS-REGION;
#   each has own endpoint (app must route reads); promotable to
#   standalone (DR or migration splits); replica of replica possible
#   (lag compounds); requires automated backups on
# Multi-AZ vs read replica (the #1 database exam question):
#   Availability/failover → Multi-AZ; read scaling → replicas;
#   both → Multi-AZ + replicas (not either/or)
# Backups: automated (PITR, retention 0–35 days, stored in S3) vs
#   manual snapshots (kept until deleted; only snapshots survive
#   instance deletion; share snapshots cross-account — encrypted ones
#   need customer managed KMS key)
# Restore = NEW instance with new endpoint (update app config/DNS)
# RDS Custom (Oracle/SQL Server) — OS/DB-level access when packaged
#   apps demand it; RDS on Outposts exists
# Blue/Green deployments (MySQL/PostgreSQL/Aurora) — staged copy with
#   replication, switchover <1 min ("upgrade with minimal downtime")
# Performance Insights — DB load by wait state/SQL ("find the slow
#   query" answer); Enhanced Monitoring = OS-level metrics
# Encryption at creation; TLS in transit; IAM DB auth (MySQL/PG) =
#   token instead of password; Secrets Manager managed rotation
# RDS Proxy — connection pooling + faster failover + IAM/Secrets:
#   "Lambda exhausts DB connections" → Proxy (cs aws-high-availability)
```

## Aurora

```
# Storage: 6 copies across 3 AZs, quorum 4/6 write 3/6 read, self-
#   healing, auto-grow to 128 TiB; only the WRITER pays sync cost —
#   replicas share the same storage (no replica lag in the RDS sense,
#   typically <100 ms)
# Endpoints: cluster (writer) / reader (load-balances all replicas) /
#   custom (subset, e.g. "analytics replicas only") — app uses reader
#   endpoint instead of juggling replica endpoints
# Failover: replica promoted <30 s typically; failover priority tiers
# Aurora Serverless v2 — fine-grained ACU autoscaling (0.5 ACU steps,
#   scales in place, Multi-AZ capable, mixes with provisioned in one
#   cluster): "variable/unpredictable load", "dev/test", "per-second
#   capacity billing"; v1 (pause-to-zero) is legacy
# Aurora Global Database — 1 primary + up to 5 secondary Regions,
#   storage-level replication, typical lag <1 s; RPO ~1 s, RTO <1 min
#   (managed failover); secondary Regions serve reads ("global reads +
#   Region DR" answer); write forwarding lets secondaries accept
#   writes logically
# Extras: Backtrack (MySQL — rewind cluster minutes/hours without
#   restore), fast database cloning (copy-on-write test copies),
#   parallel query, zero-ETL to Redshift, Babelfish (SQL Server
#   T-SQL wire compat on Aurora PostgreSQL — migration answer),
#   Aurora ML, RDS Data API (HTTP SQL for Lambda/AppSync)
# Cost note: Aurora I/O-Optimized config removes per-I/O charges
#   (I/O-heavy workloads); standard config cheaper for light I/O
```

## DynamoDB

```
# Serverless key-value/document; single-digit ms at ANY scale;
#   multi-AZ by default; unlimited items; item cap 400 KB
# Keys: partition key (hash) or partition+sort (composite);
#   design for even key distribution (hot partition = throttling)
# Capacity modes:
#   On-demand   — pay per request; instant scale; unknown/spiky —
#                 default exam answer for "unpredictable"
#   Provisioned — RCU/WCU (+ auto scaling target tracking); cheaper
#                 for steady, predictable load; reserved capacity
#                 discounts on top
#   1 RCU = 1 strongly consistent read/s ≤4 KB (2 eventual);
#   1 WCU = 1 write/s ≤1 KB; transactional = 2×
# Consistency: eventual (default) | strong (GetItem option; not on
#   GSIs) | transactions (ACID across items/tables)
# Indexes:
#   GSI — different partition+sort key, own capacity, eventual only,
#         create anytime — "query by another attribute" answer
#   LSI — same partition key, alt sort key, create AT TABLE CREATION
#         only, shares capacity, allows strong reads
# DAX — write-through in-memory cache cluster: microsecond reads,
#   API-compatible (no code rewrite) — "read-heavy, microsecond,
#   minimal code change"; ElastiCache when caching aggregates/
#   cross-service
# Streams — ordered change log (24 h) → Lambda triggers (audit,
#   replication, aggregation); Kinesis Data Streams destination option
# Global Tables — multi-Region ACTIVE-ACTIVE replication (needs
#   streams; last-writer-wins conflicts): "multi-Region ms reads and
#   writes" answer
# TTL — free auto-expiry by epoch attribute (sessions, carts);
#   PITR 35 days; on-demand backups; incremental + full export to S3
#   (no capacity consumed) → Athena over exports, not scans
# S3 large-object pattern: store blob in S3, pointer in item (400 KB!)
# Throttling fixes: better key spread, on-demand mode, DAX for hot
#   reads, write sharding (key suffix), SQS buffer for writes
```

## ElastiCache

```
# Redis OSS / Valkey — replication + Multi-AZ auto-failover, cluster
#   mode (shards), persistence (AOF/RDB), pub/sub, sorted sets
#   (leaderboards!), geo, streams; auth token + TLS; backup/restore
# Serverless option — no node sizing, per-GB+ECPU billing
# Memcached — multi-threaded plain cache, no replication/persistence/
#   failover; "simplest horizontal object cache, data loss OK"
# Choose Redis when ANY of: HA, persistence, sorted sets/pub-sub,
#   snapshots. Memcached only for "simple + multithreaded + ephemeral"
# Session store answer: ElastiCache Redis (or DynamoDB with TTL —
#   serverless flavor); caching patterns → cs aws-decoupling
# Lazy-load vs write-through tradeoffs; TTL always; cache invalidation
#   on writes for read-after-write needs
```

## Redshift (Warehouse Snapshot)

```
# Columnar MPP OLAP: leader + compute nodes; RA3 nodes separate
#   compute/storage; Serverless option (RPU billing)
# Loading: COPY from S3 (parallel), Kinesis/MSK streaming ingestion,
#   zero-ETL from Aurora/RDS; UNLOAD to S3
# Spectrum — query S3 external tables without loading
# Concurrency scaling, materialized views, sort/dist keys for perf
# Multi-AZ deployment (RA3); snapshots cross-Region for DR
# vs Athena: dedicated warehouse + BI concurrency vs serverless ad hoc
#   on the lake (cs aws-data-analytics)
```

## Capacity Planning & Migration Notes

```
# RDS/Aurora: instance class CPU/RAM + Provisioned IOPS to match
#   working set; connection limits scale with RAM — pool (RDS Proxy)
#   instead of upsizing for connections alone
# DynamoDB: model access patterns FIRST (single-table design);
#   RCU/WCU math above; adaptive capacity helps but won't save a bad
#   key
# Engine choice MySQL vs PostgreSQL: exam only expects "both are
#   open-source engines on RDS/Aurora; pick per team skills/app
#   compatibility" — deeper diffs live in cs mysql / cs postgresql
# Homogeneous migration (same engine) → native dump/replica or DMS
# Heterogeneous (Oracle→Aurora) → SCT converts schema + DMS moves/
#   replicates data (details → cs aws-migration)
```

## Database Cost Optimization (Task 4.3)

```
# Aurora Serverless v2 / RDS stop (7-day auto-restart) for dev/test
# Reserved instances: RDS/ElastiCache/Redshift 1/3-yr (up to ~69%);
#   DynamoDB reserved capacity for provisioned mode
# Graviton instance classes (db.r7g...) — cheaper per perf
# Right-size storage: gp3 over io1 unless IOPS demand it; Aurora
#   I/O-Optimized only when I/O >25% of bill
# DynamoDB: on-demand for spiky (no idle provision), provisioned+auto
#   scaling for steady; TTL to purge dead data; Standard-IA TABLE
#   CLASS for cold tables (cheaper storage, dearer requests)
# Caching = cost tool: DAX/ElastiCache absorb reads cheaper than
#   RCUs/replicas at scale
# Offload reports to a read replica instead of upsizing the writer;
#   archive cold rows to S3+Athena instead of growing the OLTP store
# Delete: unused replicas, old manual snapshots, idle dev clusters
```

## Exam Traps

```
# Multi-AZ = availability (no read scaling); replicas = read scaling
#   (async, lag); cross-Region replica = DR + local reads
# RDS Multi-AZ standby cannot be read; Aurora replicas CAN (and are
#   the failover targets)
# Read-after-write consistency needs: strong reads (DynamoDB),
#   reads from the writer (RDS), or cache invalidation — replicas may
#   serve stale data seconds behind
# LSI only at table creation; GSI anytime but eventual-only + own
#   throughput (GSI throttling back-pressures table writes!)
# DynamoDB 400 KB item limit → S3 pointer pattern
# Hot partition throttles despite ample total capacity — key design,
#   not more WCUs
# DAX ≠ ElastiCache: DynamoDB-transparent vs general cache you code
# Aurora reader endpoint balances reads; sending writes there fails
# Encrypt-existing-RDS = snapshot → copy encrypted → restore (same
#   dance as EBS, cs aws-data-protection)
# Restores create NEW endpoints — apps/DNS must repoint
# "Rewind without restore" (Aurora MySQL) = Backtrack, not PITR
# QLDB/ledger asks now usually resolve to "verifiable audit" patterns;
#   if offered vs DynamoDB Streams+audit trail, read the "immutable,
#   cryptographically verifiable" phrase → ledger
# ElastiCache Memcached has NO failover — any HA phrase kills it
# Redshift is not for OLTP; RDS is not for petabyte analytics
```

## See Also

aws-saa-c03, aws-storage, aws-compute, aws-decoupling, aws-high-availability, aws-data-analytics, aws-migration, aws-cost-optimization, dynamodb, mysql, postgresql, redis, memcached, elasticsearch, cassandra, mongodb, sql, caching-patterns

## References

- [RDS User Guide](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/Welcome.html)
- [RDS Multi-AZ deployments](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/Concepts.MultiAZ.html)
- [Aurora User Guide](https://docs.aws.amazon.com/AmazonRDS/latest/AuroraUserGuide/CHAP_AuroraOverview.html)
- [Aurora Global Database](https://docs.aws.amazon.com/AmazonRDS/latest/AuroraUserGuide/aurora-global-database.html)
- [DynamoDB Developer Guide](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/Introduction.html)
- [DynamoDB best practices](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/best-practices.html)
- [ElastiCache documentation](https://docs.aws.amazon.com/elasticache/)
- [Redshift Management Guide](https://docs.aws.amazon.com/redshift/latest/mgmt/welcome.html)
- [AWS purpose-built databases](https://aws.amazon.com/products/databases/)
