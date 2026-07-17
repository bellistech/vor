# AWS Migration & Transfer Services (SAA-C03 Cross-Domain)

> Migration strategy (the 7 Rs), discovery and planning tools, MGN for servers, DMS/SCT for databases, and the online/offline data-transfer decision — the services stitched through every SAA-C03 domain.

## The 7 Rs (Strategy Vocabulary)

```
# Retire     — kill it (discovery often finds 10–20% dead apps)
# Retain     — leave on-prem for now (compliance, latency, sunk cost)
# Rehost     — lift-and-shift VMs as-is → MGN (fastest, no code change)
# Relocate   — hypervisor-level move (VMware Cloud on AWS bulk moves)
# Repurchase — drop to SaaS (Exchange → M365, CRM → Salesforce)
# Replatform — lift-and-tinker: same app, managed underpinnings
#              (self-managed MySQL → RDS, app servers → Beanstalk,
#               containers → ECS Fargate via App2Container)
# Refactor   — re-architect cloud-native (microservices, serverless,
#              DynamoDB single-table) — highest effort and payoff
# Exam decoding: "quickly with minimal changes" → rehost;
#   "reduce DB admin burden during migration" → replatform to RDS;
#   "modernize into event-driven/serverless" → refactor
```

## Discovery & Planning

```
# Application Discovery Service — inventory + dependency mapping:
#   Agentless Collector (VMware OVA — VM inventory/perf) or Discovery
#   Agent (per-server processes + NETWORK CONNECTIONS for dependency
#   grouping); data → Migration Hub (+ Athena analysis option)
# Migration Hub — single dashboard tracking discovery + migrations
#   (MGN, DMS, partner tools); groups servers into applications;
#   wave planning ("track migration status across tools centrally")
# Migration Evaluator — TCO business case from real utilization
# "Map which servers talk to which before wave planning" →
#   Discovery Agent dependency data in Migration Hub
```

## MGN (Application Migration Service)

```
# THE rehost answer (replaced SMS and CloudEndure):
# Lightweight agent → continuous BLOCK-LEVEL replication into a
#   low-cost staging area (small instances + EBS) in the target
#   account/Region; OS, apps, data all come along
# Non-disruptive TESTING: launch test instances any time from current
#   state; cutover = final sync + launch production instances
#   (minutes of downtime, RPO ~seconds)
# Launch templates map source → right-sized instance types
# Sources: physical, VMware, Hyper-V, other clouds (agentless
#   snapshot option for vCenter)
# Same engine as Elastic Disaster Recovery (DRS) — DRS = ongoing
#   DR replication with failback; MGN = one-way migration then
#   decommission (cs aws-high-availability for DRS/DR strategies)
# "Migrate 500 VMs with minimal downtime, no re-architecting" → MGN
```

## DMS + SCT (Databases)

```
# DMS — managed replication instance (or DMS Serverless) moving data:
#   Sources/targets: Oracle, SQL Server, MySQL, PostgreSQL, MariaDB,
#   SAP ASE, MongoDB, S3, Aurora, RDS, Redshift, DynamoDB, Kinesis,
#   OpenSearch, DocumentDB...
#   Task modes: full load | full load + CDC | CDC only
#   CDC (change data capture) tails source logs → near-zero-downtime
#   cutover: full load runs while source stays live, CDC drains the
#   delta, switch apps when lag ≈ 0
#   Also: continuous replication (keep on-prem + cloud in sync),
#   cross-Region DR feeds, S3 data-lake feeds, Multi-AZ replication
#   instance for the migration itself
# Homogeneous (Oracle→Oracle, MySQL→Aurora MySQL): DMS alone (or
#   native tooling: dump/restore, read-replica promotion)
#   → DMS "homogeneous migrations" now also offers native-tool
#     orchestration (pg_dump etc.) under the hood
# Heterogeneous (Oracle→Aurora PostgreSQL, SQL Server→MySQL):
#   SCT (Schema Conversion Tool) converts schema/procedures first —
#   assessment report flags manual items; then DMS moves/replicates
#   ("change database ENGINE during migration" → SCT + DMS, always)
# Babelfish alternative: SQL Server apps → Aurora PostgreSQL without
#   rewriting T-SQL (cs aws-databases)
# Large initial loads over thin pipes: SCT/DMS + Snowball Edge combo
```

## Data Transfer Decision (Files & Objects)

```
# Bandwidth math first: transfer_days ≈ TB × 8 / (Gbps × 86400 × η)
#   1 TB @ 1 Gbps ≈ 2.5 h wire-speed; @ 100 Mbps ≈ 1 day
#   Weeks-to-months or no spare bandwidth → ship hardware
#
# Online:
# DataSync — NFS/SMB/HDFS/object ↔ S3/EFS/FSx; scheduled, verified,
#   throttled, incremental ("ongoing/one-shot bulk file sync") —
#   agent on-prem, or agentless between AWS services
# Transfer Family — inbound SFTP/FTPS/FTP/AS2 endpoints onto S3/EFS
#   ("partners keep their SFTP scripts")
# S3 Transfer Acceleration — long-haul uploads to one bucket via edge
# Storage Gateway — HYBRID ACCESS (keep using on-prem protocols,
#   data lives in cloud) rather than a migration mover per se
# Kinesis/Firehose/MSK — streaming ingestion (cs aws-data-analytics)
#
# Offline (Snow family):
# Snowball Edge Storage Optimized ~80 TB usable (Compute Optimized
#   variant adds GPUs/compute at the edge)
# Snowcone 8–14 TB, rugged/tiny
# Import job: order → copy (encrypted, KMS) → ship back → AWS loads
#   S3; also EXPORT direction; OpsHub GUI; Snowball can run DataSync/
#   Lambda at edge sites
# "10 PB, 100 Mbps link, 3-month deadline" → fleet of Snowballs
```

## Other Movers

```
# VM Import/Export — OVA/VMDK/VHD → AMI (DIY, no replication;
#   MGN supersedes for fleets)
# App2Container — containerize existing .NET/Java servers → ECS/EKS
#   artifacts + pipelines
# Elastic Beanstalk / ECS / EKS as landing zones for replatformed
#   apps (cs aws-compute)
# S3 CRR/Batch Ops — bucket-to-bucket moves inside AWS
#   (cs aws-data-protection)
# Aurora/RDS snapshot copy, read-replica-promotion migrations
#   (same-engine, brief cutover)
# Mainframe Modernization, VMware Cloud on AWS — name-recognition
#   level only
```

## Cutover Patterns

```
# Blue/green: build target fully, test, flip DNS (Route 53 weighted
#   0→100 canary or hard switch; low TTL beforehand)
# Phased/strangler: route slice-by-slice (path-based ALB rules or
#   weighted records), retire legacy pieces gradually
# Bake dual-run: CDC keeps both sides consistent while comparing
# Rollback plan is part of the answer: keep source replicating
#   (MGN/DMS reverse or delayed decommission) until validation ends
# Hybrid DNS during transition → Route 53 Resolver endpoints
#   (cs aws-networking)
```

## Exam Traps

```
# "Migrate SERVERS" → MGN; "migrate DATABASES" → DMS; "migrate
#   FILES" → DataSync; "receive files from partners" → Transfer
#   Family; "keep on-prem access to cloud data" → Storage Gateway —
#   the object of the sentence picks the service
# Engine change anywhere in the stem → SCT joins DMS
# DMS CDC = minimal-downtime lever; plain dump/restore = downtime
# DataSync ≠ Storage Gateway: move vs ongoing hybrid access
# Snowball when bandwidth math fails online transfer — do the
#   arithmetic, the stem gives link speed + volume + deadline
# Snow devices encrypt with KMS and never store keys on device;
#   data is wiped after import (NIST 800-88) — security box-ticks
# MGN staging uses cheap instances; you don't pay production-size
#   until test/cutover launches
# Migration Hub tracks; it does not move anything itself
# Discovery AGENT sees network dependencies; agentless collector
#   sees VMware inventory/perf only — "dependency mapping" needs the
#   agent
# VMware Cloud on AWS = relocate whole vSphere estates; niche but
#   distinct from MGN rehost in answer lists
```

## See Also

aws-saa-c03, aws-storage, aws-databases, aws-high-availability, aws-data-analytics, aws-networking, aws-cost-optimization, rsync, mysql, postgresql, backup

## References

- [AWS migration strategies (7 Rs)](https://docs.aws.amazon.com/prescriptive-guidance/latest/large-migration-guide/migration-strategies.html)
- [Application Migration Service User Guide](https://docs.aws.amazon.com/mgn/latest/ug/what-is-application-migration-service.html)
- [DMS User Guide](https://docs.aws.amazon.com/dms/latest/userguide/Welcome.html)
- [Schema Conversion Tool User Guide](https://docs.aws.amazon.com/SchemaConversionTool/latest/userguide/CHAP_Welcome.html)
- [DataSync User Guide](https://docs.aws.amazon.com/datasync/latest/userguide/what-is-datasync.html)
- [Snow Family documentation](https://docs.aws.amazon.com/snowball/)
- [Transfer Family User Guide](https://docs.aws.amazon.com/transfer/latest/userguide/what-is-aws-transfer-family.html)
- [Application Discovery Service User Guide](https://docs.aws.amazon.com/application-discovery/latest/userguide/what-is-appdiscovery.html)
- [Migration Hub User Guide](https://docs.aws.amazon.com/migrationhub/latest/ug/whatishub.html)
