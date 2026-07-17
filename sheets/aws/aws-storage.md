# AWS Storage Selection & Performance (SAA-C03 Tasks 3.1 + 4.1)

> Determine high-performing, scalable, and cost-optimized storage: S3 classes and lifecycle, EBS volume types with real numbers, EFS, the four FSx flavors, Storage Gateway, and the object/file/block decision.

## Storage Type Decision (First Cut)

```
# Object (S3)  — API access (GET/PUT), unlimited scale, web/data lake,
#                static assets, backups; NOT a filesystem
# Block (EBS)  — raw device for ONE EC2 instance (io1/io2 multi-attach
#                is the narrow exception), lowest latency, databases
# File (EFS/FSx) — shared POSIX/SMB mounts, many clients simultaneously
#
# "Thousands of instances need shared access"      → EFS (Linux/NFS)
# "Windows / SMB / Active Directory integration"   → FSx for Windows
# "HPC scratch, ML training, process S3 data fast" → FSx for Lustre
# "NetApp features (snapshots, clones, SnapMirror,
#   multi-protocol NFS+SMB+iSCSI)"                 → FSx for NetApp ONTAP
# "ZFS, low-latency NFS, snapshots/clones"         → FSx for OpenZFS
# "Serve images/video to web at scale"             → S3 (+CloudFront)
# "Boot volume / transactional DB storage"         → EBS
```

## S3 Storage Classes (Know Cold)

| Class | Design | Min duration | Retrieval | Use |
|---|---|---|---|---|
| Standard | 11x9 durability, ≥3 AZ | none | instant | hot data |
| Intelligent-Tiering | auto-moves between tiers, no retrieval fees | none | instant | unknown/changing patterns — the "unpredictable access" answer |
| Standard-IA | cheaper storage, per-GB retrieval fee | 30 days | instant | monthly-ish access |
| One Zone-IA | 1 AZ only (AZ loss = data loss) | 30 days | instant | re-creatable secondary copies |
| Glacier Instant Retrieval | archive price, ms access | 90 days | instant | quarterly access archives |
| Glacier Flexible Retrieval | cheaper | 90 days | expedited 1–5 min / standard 3–5 h / bulk 5–12 h (free) | yearly archives |
| Glacier Deep Archive | cheapest (~1 USD/TB-mo) | 180 days | standard 12 h / bulk 48 h | compliance, 7–10 yr retention |
| Express One Zone | single-digit ms, directory buckets | none | instant | latency-critical hot data |

```
# Pricing levers: storage GB-month + requests + retrieval + transfer
# (cross-Region/out-to-internet; IN is free; to CloudFront free)
# Minimum billable object size in IA classes: 128 KB — millions of
#   tiny objects in IA/Glacier = trap (batch into archives first)
# Early-delete fee: deleting before min duration bills the remainder
# Intelligent-Tiering: small monthly monitoring fee per object,
#   opt-in Archive/Deep-Archive tiers; no retrieval fees ever
```

## Lifecycle, Analytics, Cost Plumbing

```json
{"Rules":[{"ID":"logs","Filter":{"Prefix":"logs/"},"Status":"Enabled",
  "Transitions":[
    {"Days":30,"StorageClass":"STANDARD_IA"},
    {"Days":90,"StorageClass":"GLACIER"},
    {"Days":365,"StorageClass":"DEEP_ARCHIVE"}],
  "NoncurrentVersionExpiration":{"NoncurrentDays":30},
  "Expiration":{"Days":2555},
  "AbortIncompleteMultipartUpload":{"DaysAfterInitiation":7}}]}
```

```
# Transition order is one-way down the cost ladder (can't lifecycle
#   INTO Standard from IA)
# Always add AbortIncompleteMultipartUpload — orphaned parts bill
#   silently (Storage Lens flags them)
# Noncurrent version expiration = the versioning cost fix
# S3 Storage Class Analysis — recommends Standard→IA ages
# S3 Storage Lens — org-wide usage/cost dashboards + anomalies
# Requester Pays — downloader pays transfer/requests (public datasets)
```

## S3 Performance

```
# Per-prefix baseline: 3,500 PUT/COPY/POST/DELETE + 5,500 GET/HEAD
#   per second PER PREFIX — parallelize across prefixes for more
#   (date-hash prefixes; no global limit)
# Multipart upload — REQUIRED >5 GB (object cap 5 TB, part 5 MB–5 GB);
#   recommended >100 MB; parallel parts + resume on failure
# Byte-range GETs — parallel downloads, partial reads
# Transfer Acceleration — upload via nearest edge → AWS backbone
#   ("scientists worldwide upload to one bucket faster")
# Multi-Region Access Points — one global endpoint, routed to nearest
#   replicated bucket, with failover
# S3 Select — SQL projection on one object (CSV/JSON/Parquet) server-
#   side; Athena for multi-object queries (cs aws-data-analytics)
# Event notifications → SQS/SNS/Lambda/EventBridge (EventBridge mode
#   adds filtering + more targets)
# Batch Operations — copy/tag/restore/invoke-Lambda over billions of
#   objects from a manifest/inventory
```

## EBS Volume Types (Numbers Matter)

| Type | Base | Max IOPS | Max MB/s | Notes |
|---|---|---|---|---|
| gp3 | 3,000 IOPS + 125 MB/s included | 16,000 | 1,000 | provision IOPS/throughput INDEPENDENT of size; ~20% cheaper than gp2 — default answer |
| gp2 | 3 IOPS/GB (min 100), burst to 3,000 | 16,000 | 250 | legacy; 5,334 GB+ = max IOPS |
| io1 | provisioned | 64,000 (Nitro) | 1,000 | 50 IOPS/GB max ratio |
| io2 Block Express | provisioned | 256,000 | 4,000 | 99.999% durability, 1,000 IOPS/GB, sub-ms — "highest performance database" answer |
| st1 | throughput HDD | 500 | 500 | big-data sequential, cheap; can't boot |
| sc1 | cold HDD | 250 | 250 | cheapest; archival on-instance; can't boot |

```
# All EBS: single AZ; attach to one instance (io1/io2 Multi-Attach up
#   to 16 Nitro instances — needs cluster-aware FS, rare correct answer)
# Snapshots: incremental to S3; Fast Snapshot Restore (FSR) removes
#   first-read latency (pay per AZ-hour); snapshot archive tier = 75%
#   cheaper, 24–72 h restore; Recycle Bin protects against deletion
# Data Lifecycle Manager (DLM) — schedule snapshot create/retain/copy
# Elastic Volumes — grow size / change type / tune IOPS live (no
#   detach); shrinking is NOT possible (new smaller volume + copy)
# Encryption: default-on setting; snapshot dance for existing volumes
#   → cs aws-data-protection
# Instance store — NVMe physically attached: highest IOPS (millions),
#   EPHEMERAL (lost on stop/terminate/hardware failure); "temp scratch,
#   buffers, caches, replicated-by-app data" only
# RAID 0 stripes volumes for IOPS beyond one volume's cap (no RAID 1
#   answer — snapshots + replication cover durability)
```

## EFS (Elastic File System)

```
# Managed NFSv4.1, Linux-only, elastic to petabytes, pay-per-GB used
# Thousands of concurrent NFS clients across AZs (mount targets per AZ);
#   also mounts into Lambda/ECS/EKS ("shared state for containers")
# Storage classes: Standard (Regional multi-AZ) / One Zone;
#   Infrequent Access (IA) + Archive classes via lifecycle management
#   (move after N days no access) — EFS Intelligent-Tiering
# Performance modes: General Purpose (default, lowest latency) |
#   Max I/O (higher aggregate, more latency; NOT on Elastic throughput)
# Throughput modes: Elastic (default, auto-scales, pay for use) |
#   Provisioned (fixed MiB/s regardless of size) |
#   Bursting (scales with stored GB — credit model)
# Access Points — per-app POSIX identity + root directory jail
# Encryption at rest (create-time) + TLS in transit (mount -o tls)
# Backup via AWS Backup; cross-Region replication (RPO minutes)
# vs EBS: many readers/writers + elastic vs one instance + provisioned
# Cost: ~3× S3 Standard per GB — don't use EFS as object dump
```

## FSx Family

```
# FSx for Windows File Server — native SMB, NTFS, AD-joined (yours or
#   AWS Managed AD), DFS namespaces, shadow copies, Multi-AZ option
#   → any "Windows file share" scenario
# FSx for Lustre — parallel POSIX FS: 100s GB/s, millions IOPS, sub-ms;
#   S3 integration (lazy-load objects as files, write back with
#   hsm_archive) — "HPC / ML training / video rendering on S3 data"
#   Deployment: scratch (temp, cheapest, no replication) vs persistent
# FSx for NetApp ONTAP — multiprotocol NFS+SMB+iSCSI, SnapMirror
#   (migrate FROM on-prem NetApp), FlexClone instant clones, dedup/
#   compression, Multi-AZ — "on-prem NetApp lift", "clone DB for test"
# FSx for OpenZFS — NFS v3/4.x, ZFS snapshots/clones, up to 1M IOPS
#   low-latency — "migrate ZFS/Linux NFS servers needing snapshots"
```

## Storage Gateway (Hybrid Bridge)

```
# On-prem VM/hardware appliance bridging to cloud storage:
# S3 File Gateway — NFS/SMB share → objects in S3 1:1; local cache of
#   hot data; "keep using file shares, store in S3, lifecycle to
#   Glacier" → this
# FSx File Gateway — low-latency on-prem cache for FSx for Windows
# Volume Gateway
#   Cached  — primary data in S3, hot blocks cached locally (expand
#             beyond local capacity)
#   Stored  — primary data local, async snapshots to S3 (low-latency
#             everything + cloud backup/DR)
# Tape Gateway — virtual tape library (VTL) for existing backup
#   software (Veeam/NetBackup) → S3/Glacier; "eliminate physical tapes
#   without changing backup workflows"
# All modes: local cache, bandwidth throttling, works over DX/VPN
```

## Data Transfer at Scale (Snow / DataSync)

```
# Rule of thumb: at sustained 100 Mbps, ~1 TB/day. Weeks of transfer
#   time or limited bandwidth → ship hardware.
# Snowball Edge Storage Optimized — ~80 TB usable/device (also
#   Compute Optimized with GPUs for edge processing)
# Snowcone — small/rugged (8–14 TB), fits a backpack; DataSync agent
#   pre-installed (ship OR sync back online)
# Snowmobile — retired; exabyte answers now = fleets of Snowballs
# DataSync — online managed transfer NFS/SMB/HDFS/S3↔(S3/EFS/FSx);
#   ~10× faster than open-source tools, schedules, includes verify,
#   bandwidth throttle, preserves metadata; works over DX
#   "one-time or scheduled bulk sync with verification" → DataSync
# Transfer Family — managed SFTP/FTPS/FTP/AS2 endpoints backed by
#   S3/EFS ("partners upload via SFTP, keep existing scripts")
# rsync/scp/aws s3 sync — the DIY distractors when DataSync is offered
```

## Cost Decision Tables (Task 4.1)

| Scenario | Cheapest correct answer |
|---|---|
| Unknown/shifting access pattern | S3 Intelligent-Tiering |
| Logs: hot 30d, audit 7y | Lifecycle Standard→IA→Glacier→Deep Archive |
| Re-creatable derivative data | One Zone-IA |
| Compliance archive, 12h retrieval OK | Glacier Deep Archive |
| Millions of tiny files to archive | tar/parquet-pack first (128 KB min + per-object overhead) |
| gp2 volumes at scale | Migrate to gp3 (~20% cheaper, decouple IOPS) |
| High IOPS temp scratch | Instance store (free with instance) |
| EBS snapshots pile-up | DLM retention + snapshot Archive tier |
| Shared FS, mostly cold | EFS IA/Archive lifecycle (or One Zone) |
| On-prem backups filling SAN | Storage Gateway (cached volumes / tape gateway) |
| Cross-Region traffic heavy | Rethink: replicate data once (CRR) vs per-request transfer |

## Exam Traps

```
# One Zone-IA loses data on AZ destruction — never for sole copies
# Glacier Instant vs Flexible: "immediate access to archives" →
#   Instant; "1–5 min OK" → Flexible expedited; "morning is fine" →
#   Deep Archive
# Retrieval FEES make IA wrong for frequently-read data — compute
#   break-even, don't reflex to "IA is cheaper"
# gp3 does not burst — it PROVISIONS; gp2's burst credits empty on
#   sustained load (mystery slowdowns = gp2 credit exhaustion)
# st1/sc1 cannot be boot volumes; instance store can't survive stop
# EBS is AZ-scoped; EFS/FSx Multi-AZ options are Regional
# EFS is Linux/NFS only — any Windows/SMB requirement → FSx
# Lustre scratch mode has NO replication — persistent for anything
#   you can't rerun
# Storage Gateway cached ≠ stored: where does the PRIMARY copy live?
#   (cached: S3; stored: on-prem)
# DataSync vs Storage Gateway: MOVE/SYNC data vs ONGOING hybrid access
# DataSync vs Snowball: days over the wire vs weeks → ship the box
# S3 Transfer Acceleration vs CloudFront: uploads from everywhere vs
#   downloads to everywhere
# 5 GB single-PUT limit: bigger objects require multipart
```

## See Also

aws-saa-c03, aws-data-protection, aws-compute, aws-databases, aws-migration, aws-cost-optimization, aws-high-availability, s3, zfs, nfs, san-storage, linux-storage-management, backup

## References

- [S3 storage classes](https://aws.amazon.com/s3/storage-classes/)
- [S3 User Guide — lifecycle](https://docs.aws.amazon.com/AmazonS3/latest/userguide/object-lifecycle-mgmt.html)
- [S3 performance guidelines](https://docs.aws.amazon.com/AmazonS3/latest/userguide/optimizing-performance.html)
- [EBS volume types](https://docs.aws.amazon.com/ebs/latest/userguide/ebs-volume-types.html)
- [EFS User Guide](https://docs.aws.amazon.com/efs/latest/ug/whatisefs.html)
- [FSx documentation hub](https://docs.aws.amazon.com/fsx/)
- [Storage Gateway User Guide](https://docs.aws.amazon.com/storagegateway/latest/userguide/WhatIsStorageGateway.html)
- [AWS DataSync User Guide](https://docs.aws.amazon.com/datasync/latest/userguide/what-is-datasync.html)
- [AWS Snow Family](https://docs.aws.amazon.com/snowball/latest/developer-guide/whatisedge.html)
