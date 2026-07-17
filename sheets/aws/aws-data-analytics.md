# AWS Data Ingestion & Analytics (SAA-C03 Task 3.5)

> Determine high-performing data ingestion and transformation solutions: Kinesis vs Firehose vs MSK, Glue ETL and the Data Catalog, Athena, Lake Formation, QuickSight/Quick Suite, OpenSearch, and data lake design on S3.

## Ingestion Decision (Streaming Trio + Batch)

```
# Kinesis Data Streams — real-time (sub-second), REPLAYABLE stream:
#   retention 24 h default → 365 days; multiple consumers re-read;
#   ordered per shard (partition key)
#   Capacity: provisioned shards (1 MB/s or 1k rec/s in, 2 MB/s out
#   per shard; resharding is manual) or ON-DEMAND mode (auto-scales —
#   "unpredictable throughput" answer)
#   Producers: SDK/KPL/agent; PutRecords batching
#   Consumers: Lambda (event source mapping), KCL apps, Firehose,
#   Managed Flink; enhanced fan-out = dedicated 2 MB/s per consumer
#   (push) when many consumers contend
#   1 MB record cap; ProvisionedThroughputExceeded → more shards /
#   better key spread / on-demand
#
# Amazon Data Firehose (ex Kinesis Data Firehose) — ZERO-ADMIN
#   delivery pipe: buffers (size MBs / interval seconds — near-real-
#   time, not sub-second) → S3, Redshift, OpenSearch, Splunk, HTTP,
#   Snowflake, Iceberg; inline transform via Lambda; format conversion
#   JSON→Parquet/ORC; compression; dynamic partitioning of S3 keys;
#   auto-scales, pay-per-GB — no shards, NO replay, single destination
#   per stream
#   "Easiest way to land streaming data in S3/OpenSearch" → Firehose
#
# MSK (Managed Streaming for Kafka) — Kafka API compatibility:
#   existing Kafka apps/ecosystem (Connect, Schema Registry), very
#   high throughput, >1 MB messages possible, MSK Serverless option
#   "Migrating Kafka / needs Kafka ecosystem" → MSK; otherwise Kinesis
#
# Batch ingestion: DataSync (files), DMS (databases, CDC), AppFlow
#   (SaaS), Transfer Family (SFTP), Snow (offline) → cs aws-migration
# Frequency framing: real-time (Streams/MSK) vs near-real-time
#   (Firehose 60 s+) vs scheduled batch (Glue/EMR jobs) — match the
#   question's freshness requirement, don't over-engineer
```

## Kinesis Family Cheat Rows

| Need | Pick |
|---|---|
| Sub-second processing, multiple apps replay stream | Kinesis Data Streams |
| Land stream into S3/Redshift/OpenSearch, no code, no admin | Data Firehose |
| SQL/Flink on streams (windows, aggregations) | Managed Service for Apache Flink |
| Kafka protocol/ecosystem compatibility | MSK |
| Video/camera streams for ML | Kinesis Video Streams |
| Simple decoupling, no replay, per-message consume | SQS (cs aws-decoupling) |

## Glue (Serverless ETL + Catalog)

```
# Data Catalog — Hive-compatible metastore for the lake: databases →
#   tables (schema, location, format, partitions); shared by Athena,
#   Redshift Spectrum, EMR, Lake Formation — THE central schema store
# Crawlers — scan S3/JDBC/DynamoDB, infer schema + partitions,
#   populate catalog on schedule ("new files appear daily; make them
#   queryable" → crawler + Athena)
# ETL jobs — serverless Spark (Python/Scala; DPU-billed) or lighter
#   Python shell; visual job editor (Glue Studio); job bookmarks =
#   process only NEW data between runs; triggers/workflows chain jobs
# Streaming ETL — micro-batch from Kinesis/Kafka into the lake
# DataBrew — no-code data prep (250+ transforms) for analysts
# Schema Registry — Avro/JSON schema evolution for streams
# Glue Data Quality — rule-based DQ checks on datasets/pipelines
# "Serverless ETL / catalog the lake / convert CSV→Parquet nightly"
#   → Glue; heavy custom Spark with cluster control → EMR
```

## Athena (Serverless SQL on S3)

```
# Presto/Trino engine; query data IN PLACE in S3; pay ~5 USD/TB
#   SCANNED — every optimization is about scanning less:
#   1. Columnar formats (Parquet/ORC) — read only needed columns
#   2. Compress (snappy/zstd) + right-size files (128 MB–1 GB;
#      millions of tiny files = slow + expensive)
#   3. PARTITION by query filters (date=2026-07-01/) + partition
#      projection (skip metastore lookups for predictable layouts)
#   4. SELECT columns, never SELECT *
# Reads schemas from Glue Catalog; CTAS materializes results (and
#   converts formats); federated queries (Lambda connectors) reach
#   RDS/DynamoDB/on-prem; UNLOAD writes Parquet results
# Workgroups — per-team query isolation: enforced result location,
#   per-query/workgroup scan LIMITS (cost guardrails), CloudWatch
#   metrics, capacity reservations (DPU) for predictable latency
# Results land in S3 (encrypt result location too)
# vs Redshift: ad-hoc/serverless/lake vs dedicated warehouse + BI
#   concurrency + complex joins at scale (cs aws-databases)
# "Query ALB/VPC Flow/CloudTrail logs in S3 with SQL, no infra"
#   → Athena (the single most repeated analytics answer)
```

## Lake Formation & Data Lake Layout

```
# Lake Formation — permissions + governance layer over S3 + Glue
#   Catalog: database/table/column/row/cell-level grants (LF-tags for
#   attribute-based grants at scale), cross-account sharing, blueprints
#   for ingestion, governed tables (ACID)
#   Replaces "bucket-policy spaghetti per analyst" — "fine-grained
#   (column-level) access control on the data lake" → Lake Formation
# Canonical lake zones (S3 prefixes/buckets):
#   raw/ (as-landed) → cleaned/ (validated, Parquet) → curated/
#   (business aggregates) — lifecycle each zone independently
#   (cs aws-storage for classes)
# Registered locations: LF manages credentials to S3 via its service
#   role; consumers (Athena/Redshift Spectrum/EMR) get vended,
#   scoped-down access
# Iceberg/table formats: modern lakes use Iceberg tables (ACID,
#   schema evolution) via Glue/Athena — aware-level for the exam
```

## QuickSight / Quick Suite (BI)

```
# Serverless BI: dashboards, embedded analytics, ML insights
#   (anomalies, forecasts), Q natural-language queries; per-user or
#   capacity pricing; readers cheap
# SPICE — in-memory engine: cache datasets for fast dashboards and
#   to stop hammering Athena/Redshift per view
# Sources: Athena, Redshift, RDS/Aurora, S3, OpenSearch, SaaS
# Note: AWS is rebranding QuickSight into Amazon Quick Suite (2026
#   guide wording: "Amazon QuickSuite") — same exam role: THE
#   dashboards/visualization answer
# Row-level security for multi-tenant dashboards; VPC connections
#   for private sources
# "Business users need dashboards on lake/warehouse data, serverless"
#   → QuickSight (never EC2+Grafana in this exam's answers)
```

## OpenSearch Service

```
# Managed OpenSearch (Elasticsearch fork): full-text search, log
#   analytics (with Dashboards), k-NN vector search; Serverless option
# Ingest via Data Firehose (the classic logs pipeline:
#   CloudWatch Logs/apps → Firehose → OpenSearch → Dashboards),
#   OpenSearch Ingestion pipelines, or clients
# Multi-AZ with dedicated masters for prod; UltraWarm/cold tiers for
#   aging indices (cost); fine-grained access control + Cognito auth
#   for Dashboards
# "Search product catalog / analyze logs interactively / autocomplete"
#   → OpenSearch; SQL over static files → Athena instead
```

## Reference Pipelines (Exam Composites)

```
# Clickstream real-time dashboard:
#   producers → Kinesis Data Streams → Managed Flink (windows/aggs)
#   → Firehose → OpenSearch Dashboards (+ raw branch → S3 via Firehose)
# Serverless lake analytics:
#   sources → Firehose (Parquet conversion, dynamic partitioning)
#   → S3 raw/ → Glue job → S3 curated/ → Glue Catalog → Athena →
#   QuickSight (SPICE) ... governance by Lake Formation
# Log analytics with SQL:
#   ALB/VPC Flow/CloudTrail → S3 → (crawler) → Athena partitioned
#   queries → QuickSight
# DB offload for reports:
#   Aurora zero-ETL → Redshift → QuickSight (no pipeline code), or
#   DMS CDC → S3 lake for engine-agnostic analytics
# IoT telemetry:
#   devices → Kinesis (on-demand) → Firehose → S3 + Timestream for
#   recent-window queries
```

## Exam Traps

```
# Streams vs Firehose: replay/multiple consumers/sub-second = Streams;
#   zero-management delivery to S3/OpenSearch = Firehose; Firehose is
#   NEAR-real-time (buffering ≥ ~60 s or size threshold)
# Firehose can read FROM a Kinesis stream, not vice versa
# Shard math still shows up: required MB/s ÷ 1 MB/s = min shards
#   (or say "on-demand mode" when sizing is the pain point)
# Hot shard = bad partition key (device_id of one chatty device) —
#   fix the key, not the shard count
# SNS/SQS vs Kinesis: work distribution/decoupling vs ordered
#   replayable ANALYTICS stream
# Athena cost/perf = Parquet + partitions + compression, in that
#   order; "queries slow/expensive on CSV" → convert with Glue/CTAS
# Athena cost CONTROL = workgroup scan limits
# Crawler keeps NEW partitions queryable (or partition projection);
#   "yesterday's data missing in Athena" → un-crawled partition
# Column-level security on the lake → Lake Formation, not IAM alone
#   (IAM/S3 policies stop at object level)
# Glue vs EMR: serverless managed ETL vs cluster control/custom
#   frameworks/long-running (EMR task nodes on Spot for cost —
#   cs aws-compute)
# OpenSearch is not a data lake query engine; Athena is not a
#   search engine — match tool to access pattern
# Kinesis Data Analytics is now Managed Service for Apache Flink
#   (old name may appear in answers — same thing)
```

## See Also

aws-saa-c03, aws-storage, aws-databases, aws-decoupling, aws-compute, aws-migration, aws-cost-optimization, kafka, spark, hadoop, flink, elasticsearch, datalakes, airflow, dbt, parquet, avro

## References

- [Kinesis Data Streams Developer Guide](https://docs.aws.amazon.com/streams/latest/dev/introduction.html)
- [Amazon Data Firehose Developer Guide](https://docs.aws.amazon.com/firehose/latest/dev/what-is-this-service.html)
- [AWS Glue Developer Guide](https://docs.aws.amazon.com/glue/latest/dg/what-is-glue.html)
- [Athena User Guide](https://docs.aws.amazon.com/athena/latest/ug/what-is.html)
- [Athena performance tuning](https://docs.aws.amazon.com/athena/latest/ug/performance-tuning.html)
- [Lake Formation Developer Guide](https://docs.aws.amazon.com/lake-formation/latest/dg/what-is-lake-formation.html)
- [QuickSight User Guide](https://docs.aws.amazon.com/quicksight/latest/user/welcome.html)
- [MSK Developer Guide](https://docs.aws.amazon.com/msk/latest/developerguide/what-is-msk.html)
- [OpenSearch Service Developer Guide](https://docs.aws.amazon.com/opensearch-service/latest/developerguide/what-is.html)
