# AWS SAA-C03 (AWS Certified Solutions Architect — Associate Exam Guide)

> Blueprint, logistics, in-scope service catalog, and question-decoding strategy for the SAA-C03 exam — the anchor sheet for the `aws/` category; every domain has a dedicated deep-dive sheet.

## Exam Logistics

```
# Format
# 65 questions total — 50 scored + 15 unscored (unscored are unmarked,
#   used by AWS to evaluate future questions; answer everything)
# 130 minutes (add 30 min free via "ESL +30" accommodation if English
#   is your second language — request BEFORE scheduling)
# Question types:
#   Multiple choice   — 1 correct answer, 3 distractors
#   Multiple response — 2+ correct out of 5+ options; no partial credit
# No penalty for guessing — never leave a question blank
# Flag-and-review is supported; unanswered = wrong

# Scoring
# Scaled score 100–1000, pass = 720
# Compensatory model: pass the exam overall, no per-domain minimum
# Score report shows section-level performance bands (informational)

# Delivery and cost
# 150 USD (associate tier) — Pearson VUE test center or online proctored
# Certification valid 3 years; recertify by passing SAA again or any
#   Professional-level exam (SAP-C02 renews SAA automatically)

# Target candidate
# ≥1 year hands-on with AWS compute, networking, storage, databases
# The exam validates design per the AWS Well-Architected Framework:
#   secure, resilient, high-performing, cost-optimized architectures
```

## Content Domains and Weights

```
# Domain 1: Design Secure Architectures            30%
#   1.1 Design secure access to AWS resources       → cs aws-iam
#   1.2 Design secure workloads and applications    → cs aws-network-security
#   1.3 Determine appropriate data security controls → cs aws-data-protection
#
# Domain 2: Design Resilient Architectures          26%
#   2.1 Design scalable, loosely coupled architectures → cs aws-decoupling
#   2.2 Design highly available / fault-tolerant       → cs aws-high-availability
#
# Domain 3: Design High-Performing Architectures    24%
#   3.1 High-performing / scalable storage          → cs aws-storage
#   3.2 High-performing and elastic compute         → cs aws-compute
#   3.3 High-performing database solutions          → cs aws-databases
#   3.4 High-performing / scalable networks         → cs aws-networking
#   3.5 Data ingestion and transformation           → cs aws-data-analytics
#
# Domain 4: Design Cost-Optimized Architectures     20%
#   4.1 Cost-optimized storage      → cs aws-storage + cs aws-cost-optimization
#   4.2 Cost-optimized compute      → cs aws-compute + cs aws-cost-optimization
#   4.3 Cost-optimized databases    → cs aws-databases + cs aws-cost-optimization
#   4.4 Cost-optimized networks     → cs aws-networking + cs aws-cost-optimization
#
# Cross-domain topics with dedicated sheets:
#   Migration & transfer services   → cs aws-migration
#   Monitoring, governance, multi-account → cs aws-monitoring-governance

# Weight math (50 scored questions):
# D1 ≈ 15 q, D2 ≈ 13 q, D3 ≈ 12 q, D4 ≈ 10 q
```

## The Well-Architected Framework (Exam Lens)

```
# Six pillars — the exam explicitly validates designing against them
# 1. Operational Excellence — run/monitor systems, continuous improvement
#      (IaC, small reversible changes, anticipate failure, game days)
# 2. Security — protect data/systems/assets
#      (strong identity foundation, least privilege, defense in depth,
#       encrypt in transit + at rest, traceability, automate security)
# 3. Reliability — recover from failure, scale horizontally
#      (test recovery, auto-recover, stop guessing capacity, manage
#       change through automation)
# 4. Performance Efficiency — use resources efficiently
#      (democratize advanced tech via managed services, go global in
#       minutes, serverless first, experiment often, mechanical sympathy)
# 5. Cost Optimization — avoid unneeded cost
#      (consumption pricing, measure efficiency, stop paying for
#       undifferentiated heavy lifting, analyze spend)
# 6. Sustainability — minimize environmental impact
#      (right-size, managed services, efficient regions, reduce idle)
#
# Exam mapping: D1 = Security pillar, D2 = Reliability,
#               D3 = Performance Efficiency, D4 = Cost Optimization
# AWS Well-Architected Tool (free) — review workloads against pillars
```

## Question-Decoding Strategy

```
# 1. Find the constraint keyword — it selects the answer:
#    "MOST cost-effective"      → cheapest option that still works
#                                  (Spot, S3 lifecycle, Graviton, serverless)
#    "LEAST operational overhead" → most managed option
#                                  (serverless > managed > self-hosted on EC2)
#    "MOST highly available"    → Multi-AZ minimum, multi-Region if offered
#    "best performance"         → provisioned/dedicated options, caching, edge
#    "minimize latency"         → CloudFront, Global Accelerator, ElastiCache,
#                                  DAX, read replicas near users
#    "near-real-time"           → Kinesis / Data Firehose (60 s), not batch
#    "MOST secure"              → least privilege, no long-lived credentials,
#                                  roles > keys, private subnets, endpoints
# 2. Eliminate answers that are wrong on facts (unsupported combos,
#    e.g. "SQS FIFO fan-out to multiple consumers via SNS standard" —
#    check the service sheets for validity tables)
# 3. Two answers both "work" → the constraint keyword breaks the tie
# 4. Distractor patterns:
#    - Right service, wrong feature tier (S3 Standard-IA for hourly access)
#    - Oversized answer (multi-Region active-active when Multi-AZ suffices)
#    - Deprecated-pattern answer (self-managed NAT instance vs NAT gateway)
#    - Cross-purpose service (Macie for threat detection — it's data
#      classification; GuardDuty is threat detection)
# 5. Time: 130 min / 65 q = 2 min/question. First pass ≤90 s each,
#    flag long scenario questions, second pass with remaining time.
```

## Keyword → Service Reflex Table

| Scenario keyword | Reflex answer |
|---|---|
| Object storage, static website, data lake | S3 (`cs aws-storage`) |
| Block storage for one EC2 instance | EBS |
| Shared POSIX file system, Linux, 1000s of clients | EFS |
| Windows file share / SMB / Active Directory | FSx for Windows File Server |
| HPC / ML training file system, S3-linked | FSx for Lustre |
| Decouple producer/consumer, buffer bursts | SQS |
| Fan-out one event to many consumers | SNS → SQS fan-out |
| Ordered, exactly-once queue | SQS FIFO |
| SaaS/AWS-service events, schedule, event bus, routing rules | EventBridge |
| Serverless workflow, human approval, retries between steps | Step Functions |
| Real-time streaming, replayable, multiple consumers, shards | Kinesis Data Streams |
| Stream → S3/Redshift/OpenSearch with zero admin | Amazon Data Firehose |
| SQL on S3 data in place, pay-per-query | Athena |
| Serverless ETL, data catalog, crawlers | Glue |
| Global users, static+dynamic content acceleration, edge cache | CloudFront |
| Non-HTTP TCP/UDP global acceleration, static anycast IPs | Global Accelerator |
| Relational, managed, Multi-AZ failover | RDS |
| MySQL/PostgreSQL-compatible, 5x throughput, 15 replicas, global | Aurora |
| Single-digit ms at any scale, key-value, serverless | DynamoDB |
| Microsecond reads on DynamoDB | DAX |
| Sub-ms cache, sessions, leaderboards (sorted sets) | ElastiCache for Redis |
| Petabyte data warehouse, columnar, BI | Redshift |
| Threat detection from logs (VPC Flow, DNS, CloudTrail) | GuardDuty |
| Discover/classify PII in S3 | Macie |
| EC2/ECR vulnerability scanning | Inspector |
| SQLi/XSS/rate limiting on ALB, CloudFront, API Gateway | WAF |
| DDoS protection (L3/4 free, advanced paid) | Shield / Shield Advanced |
| Store DB passwords with automatic rotation | Secrets Manager |
| Free config strings / license keys / plain params | SSM Parameter Store |
| User sign-up/sign-in for web/mobile apps | Cognito |
| Workforce SSO into AWS accounts + CLI | IAM Identity Center |
| Encrypt anything with managed keys, audit key use | KMS |
| Dedicated single-tenant HSM, FIPS 140-2 L3 | CloudHSM |
| Private connectivity to AWS service, no IGW/NAT | VPC endpoint / PrivateLink |
| Connect 100s of VPCs + on-prem hub-and-spoke | Transit Gateway |
| Dedicated fiber, consistent bandwidth to AWS | Direct Connect |
| Encrypted tunnel over internet in minutes | Site-to-Site VPN |
| DNS, health checks, routing policies, failover | Route 53 |
| Lift-and-shift server migration (block replication) | Application Migration Service (MGN) |
| Database migration, heterogeneous engines | DMS (+ SCT for schema) |
| Petabytes offline transfer / edge compute | Snow Family |
| NFS/SMB/S3 sync over network, on-prem ↔ AWS | DataSync |
| SFTP/FTPS/FTP endpoint backed by S3/EFS | Transfer Family |
| On-prem apps need cloud storage (cached iSCSI/NFS/VTL) | Storage Gateway |
| Multi-account guardrails, OU structure | Organizations + SCPs + Control Tower |
| Record resource config changes, compliance rules | Config |
| Who called which API when | CloudTrail |
| Metrics, alarms, dashboards, log aggregation | CloudWatch |
| Distributed request tracing | X-Ray |

## In-Scope Services (Study Checklist)

```
# Analytics
#   Athena, AWS Data Exchange, EMR, Glue, Kinesis Data Streams,
#   Amazon Data Firehose, Lake Formation, MSK (Managed Kafka),
#   OpenSearch Service, QuickSight (rebranding to Amazon Quick Suite),
#   Redshift
# Application Integration
#   AppFlow, AppSync, EventBridge, MQ, SNS, SQS, Step Functions
# Cost Management
#   Budgets, Cost and Usage Report (CUR), Cost Explorer, Savings Plans
# Compute
#   Batch, EC2, EC2 Auto Scaling, Elastic Beanstalk, Outposts,
#   Local Zones, Wavelength
# Containers
#   ECR, ECS, EKS, Fargate (also listed under Serverless)
# Database
#   Aurora (+ Aurora Serverless v2), DocumentDB, DynamoDB, ElastiCache,
#   Keyspaces (Cassandra), Neptune (graph), RDS, Timestream (time series)
# Developer Tools / Front-End
#   X-Ray, Amplify, API Gateway, AppSync
# Machine Learning (know the use case, not the internals)
#   Comprehend (NLP/sentiment), Forecast, Fraud Detector, Kendra (search),
#   Lex (chatbots), Polly (text→speech), Rekognition (image/video),
#   SageMaker, Textract (OCR + forms), Transcribe (speech→text), Translate
# Management and Governance
#   CloudFormation, CloudTrail, CloudWatch, AWS CLI, Compute Optimizer,
#   Config, Control Tower, Health Dashboard, License Manager, Console,
#   Organizations, Service Catalog, Systems Manager, Trusted Advisor,
#   Well-Architected Tool
# Media
#   Elastic Transcoder (legacy), Kinesis Video Streams
# Migration and Transfer
#   Application Discovery Service, Application Migration Service (MGN),
#   DMS, DataSync, Migration Hub, Snow Family, Transfer Family
# Networking and Content Delivery
#   CloudFront, Direct Connect, Global Accelerator, PrivateLink,
#   Route 53, Transit Gateway, VPC, Site-to-Site VPN, Client VPN
# Security, Identity, Compliance
#   ACM, Artifact (compliance reports), Audit Manager, CloudHSM, Cognito,
#   Detective, Directory Service, Firewall Manager, GuardDuty, IAM,
#   IAM Identity Center, Inspector, KMS, Macie, Network Firewall,
#   RAM (Resource Access Manager), Secrets Manager, Security Hub,
#   Shield, WAF
# Serverless
#   Fargate, Lambda (+ SAM at awareness level)
# Storage
#   AWS Backup, EBS, EFS, Elastic Disaster Recovery (DRS), FSx (Windows,
#   Lustre, NetApp ONTAP, OpenZFS), S3, S3 Glacier tiers, Storage Gateway
```

## Out-of-Scope (Do Not Study for This Exam)

```
# Whole categories excluded: AR/VR, Blockchain (Managed Blockchain),
# Game Tech (GameLift), IoT (all IoT services), Quantum (Braket),
# Satellite (Ground Station), RoboMaker
# Also excluded: developer-pipeline detail (CodeCommit/Build/Deploy/
# Pipeline internals — DevOps exam), Device Farm, Pinpoint internals,
# deep ML (that's the ML Specialty), SAP-specific, Mainframe Modernization
# If an answer option names an out-of-scope service, it is almost
# always a distractor — but "almost": Organizations/Control Tower
# questions may mention member services casually
```

## Study Plan (Using This Category)

```
# Pass 1 — breadth (1–2 weeks):
#   cs aws-saa-c03        (this sheet — memorize the reflex table)
#   cs aws-iam            cs aws-network-security   cs aws-data-protection
#   cs aws-decoupling     cs aws-high-availability
#   cs aws-storage        cs aws-compute            cs aws-databases
#   cs aws-networking     cs aws-data-analytics
#   cs aws-cost-optimization  cs aws-migration
#   cs aws-monitoring-governance
# Pass 2 — depth: every "Exam Traps" section in the sheets above
# Pass 3 — practice exams: AWS Skill Builder official practice set;
#   score ≥80% consistently before booking
# Hands-on floor: build a VPC from scratch (subnets, NAT, endpoints),
#   an ASG behind an ALB, an S3 lifecycle policy, an SQS→Lambda pipeline,
#   and one RDS Multi-AZ failover test — the exam rewards muscle memory
# Free tier + AWS Skill Builder "Solutions Architect – Associate"
#   official 20-question practice set is free
```

## Booking and Day-Of

```
# 1. aws.training → Certification account (links to Pearson VUE)
# 2. Request "ESL +30" accommodation BEFORE scheduling if applicable
# 3. Online proctored: room scan, no notes/second monitor/talking;
#    test center: two IDs, nothing else
# 4. Results: no on-screen pass/fail at submission — results arrive by
#    email within 24 h (usually same evening); score + section bands
#    appear in the Certification account
# 5. Passing unlocks 50% discount voucher for the next exam (use it on
#    SAP-C02 Professional within 3 years to auto-renew this cert)
```

## See Also

aws-iam, aws-network-security, aws-data-protection, aws-decoupling, aws-high-availability, aws-storage, aws-compute, aws-databases, aws-networking, aws-data-analytics, aws-cost-optimization, aws-migration, aws-monitoring-governance, aws-cli, cloud-security, iam, s3, vpc

## References

- [AWS Certified Solutions Architect — Associate (SAA-C03) Exam Guide](https://d1.awsstatic.com/training-and-certification/docs-sa-assoc/AWS-Certified-Solutions-Architect-Associate_Exam-Guide.pdf)
- [SAA-C03 official exam page](https://aws.amazon.com/certification/certified-solutions-architect-associate/)
- [AWS Well-Architected Framework](https://docs.aws.amazon.com/wellarchitected/latest/framework/welcome.html)
- [AWS Skill Builder — official practice question sets](https://skillbuilder.aws/)
- [AWS Certification recertification policy](https://aws.amazon.com/certification/recertification/)
- [Pearson VUE — AWS testing](https://home.pearsonvue.com/aws)
- [AWS Architecture Center — reference architectures](https://aws.amazon.com/architecture/)
