# AWS Monitoring, Governance & IaC (SAA-C03 Cross-Domain)

> CloudWatch, CloudTrail, Config, X-Ray, Systems Manager, Trusted Advisor, CloudFormation, and the multi-account governance stack — the observability and control plane behind every SAA-C03 domain.

## The Who/What/When Triad (Most-Tested Distinction)

```
# CloudWatch — HOW IS IT PERFORMING  (metrics, logs, alarms)
# CloudTrail — WHO DID WHAT, WHEN     (API audit trail)
# Config     — WHAT DOES IT LOOK LIKE / DID IT COMPLY (resource state
#              history + rules)
# "CPU spiked"→CloudWatch; "who deleted the bucket"→CloudTrail;
# "was this SG ever open to 0.0.0.0/0, when did it change"→Config
```

## CloudWatch

```
# Metrics — namespaced time series; EC2 basic = 5 min free, detailed
#   = 1 min paid; custom metrics via PutMetricData (standard 1 min,
#   high-resolution 1 s)
#   NOT default on EC2: memory %, disk space — need CloudWatch AGENT
#   (the "monitor RAM" trap); agent config via SSM Parameter Store
# Alarms — threshold/anomaly-band on a metric; states OK/ALARM/
#   INSUFFICIENT_DATA; actions: SNS, ASG scale, EC2 stop/terminate/
#   RECOVER (auto-recover on host failure keeps IP/volumes — classic
#   answer), Systems Manager
#   Composite alarms — AND/OR of alarms to cut pager noise
# Logs — log groups/streams; agents+services push in; RETENTION IS
#   NEVER-EXPIRE BY DEFAULT (set retention = cost answer);
#   metric filters → alarm on log patterns ("alert on ERROR rate")
#   Logs Insights — query language over log groups
#   Subscription filters → Lambda/Kinesis/Firehose (stream to
#   OpenSearch/S3 in near-real-time); export-to-S3 (batch) for archive
# Dashboards — cross-Region/cross-account panes
# Synthetics canaries — scripted probes of endpoints/UI flows
#   ("detect user-facing breakage before users do")
# RUM / Evidently / ServiceLens / Container-Lambda Insights —
#   know-they-exist level
# EventBridge (ex CloudWatch Events) handles the event routing side
#   → cs aws-decoupling
```

## CloudTrail

```
# Records API calls (console, CLI, SDK, service-to-service):
#   identity, time, source IP, parameters, response
# 90-day event history free per account; a TRAIL persists to S3
#   (+ optional CloudWatch Logs for alarming on events)
# Management events (control plane, default) vs DATA events (S3
#   object-level GetObject/PutObject, Lambda Invoke — high volume,
#   opt-in, extra cost): "log who read objects in this bucket" →
#   data events enabled (or S3 server access logs as the cheap/loose
#   alternative)
# Organization trail — one trail, every member account, delivered to
#   a central (locked-down) S3 bucket — the audit answer
# Integrity: log file validation (SHA-256 digest chain), bucket
#   policy + Object Lock for tamper-proofing
# CloudTrail Insights — anomaly detection on API call rates
# CloudTrail Lake — SQL queryable event store (else: Athena on the
#   S3 trail)
```

## AWS Config

```
# Configuration recorder → timeline of every resource's settings
#   (what changed, when, relationships) — forensic/state history
# Config RULES evaluate compliance continuously:
#   managed rules (encrypted-volumes, restricted-ssh,
#   s3-bucket-public-read-prohibited, required-tags...) or custom
#   (Lambda / Guard policies)
# REMEDIATION: auto-run SSM Automation documents on NON_COMPLIANT
#   ("automatically close public SGs" pattern — detect via Config,
#   fix via SSM, or route through EventBridge)
# Conformance packs — rule bundles (CIS, PCI) deployed org-wide
# Aggregators — multi-account/Region compliance view
# Config ≠ prevention: it DETECTS after the fact; SCPs prevent
#   (cs aws-iam); it also underpins Control Tower detective guardrails
# Not free: per configuration item + rule evaluation — scope wisely
```

## X-Ray & Application Visibility

```
# X-Ray — distributed tracing: SDK/agent instruments app; service
#   map, per-request timelines, annotations; finds WHICH hop is slow/
#   failing in microservices ("trace requests across services" →
#   X-Ray; infra metrics → CloudWatch; API audit → CloudTrail)
# Integrations: Lambda (checkbox), API GW, ALB adds X-Amzn-Trace-Id,
#   ECS/EKS daemon
# ADOT (AWS Distro for OpenTelemetry) — the OTel-standard path into
#   X-Ray/CloudWatch/Prometheus-managed
# Amazon Managed Grafana / Managed Service for Prometheus — for
#   k8s-native monitoring stacks (aware level)
```

## Systems Manager (SSM — The Ops Toolbox)

```
# Agent-based fleet management (EC2 + on-prem "managed instances"):
# Session Manager — shell/port-forward without SSH/bastion/inbound
#   ports; IAM-gated; logged (cs aws-network-security)
# Run Command — run scripts across the fleet, no SSH
# Patch Manager — OS patch baselines + maintenance windows +
#   compliance reporting ("patch 500 instances on schedule")
# State Manager — enforce desired config (agent versions, software)
# Automation — runbooks (documents) for ops workflows: AMI golden
#   builds, restart sequences, Config remediations
# Inventory — installed-software census; Explorer/OpsCenter — ops
#   dashboards/items
# Parameter Store — config/secret strings (vs Secrets Manager —
#   cs aws-network-security)
# Fleet prerequisites: SSM agent + instance role + (VPC endpoints or
#   egress) — "Session Manager can't connect" checklist
# Maintenance Windows — scheduled task slots across the fleet
```

## Account Health & Advice

```
# Trusted Advisor — best-practice checks: cost, performance,
#   security, fault tolerance, service limits (full set needs
#   Business+ support — cs aws-cost-optimization)
# AWS Health Dashboard — YOUR account's service events + planned
#   changes (EventBridge-able: "notify when AWS schedules maintenance
#   affecting my resources"); Service Health Dashboard = global public
#   status
# Well-Architected Tool — self-review workloads against the pillars;
#   Service Quotas — view/raise limits + CloudWatch usage alarms
#   (cs aws-high-availability)
```

## CloudFormation (IaC on the Exam)

```
# Declarative JSON/YAML stacks; the assumed IaC in most answers
#   (Terraform is absent from this exam — cs terraform for real life)
# Core: template → stack → change sets (preview!) → update; rollback
#   on failure (default); drift detection (who hand-edited?);
#   stack policies protect resources during updates
# StackSets — deploy one template to MANY accounts/Regions
#   (Organizations-integrated auto-deploy to new accounts) — the
#   "roll baseline to every account" answer
# Nested stacks for reuse; exports/imports between stacks;
#   parameters + Mappings + conditions; cfn helper scripts / cloud-init
#   user data for instance bootstrap
# DeletionPolicy: Retain/Snapshot on stateful resources (don't lose
#   the DB with the stack)
# Custom resources (Lambda-backed) fill gaps
# Elastic Beanstalk/SAM/CDK/Amplify generate CloudFormation under the
#   hood (CDK = code → CFN; SAM = serverless shorthand)
# Service Catalog — publish approved CFN templates as self-service
#   products with constraints (governed provisioning for teams)
# Immutable/infra patterns → cs aws-high-availability; golden AMIs →
#   EC2 Image Builder (cs aws-compute)
```

## Multi-Account Governance Recap

```
# Organizations — accounts/OUs/SCPs + consolidated billing
#   (mechanics → cs aws-iam; billing → cs aws-cost-optimization)
# Control Tower — landing zone: Account Factory, preventive (SCP) +
#   detective (Config) guardrails, centralized logging accounts
# Delegated administrator accounts for security tooling (GuardDuty,
#   Security Hub, Config org aggregation) — keep management account
#   empty of workloads
# License Manager — track/enforce BYOL entitlements (pairs with
#   Dedicated Hosts — cs aws-compute)
# Resource Groups + Tag Editor — operate on tagged sets
# Standard governance answer shape: Organizations + Control Tower +
#   org CloudTrail + Config conformance packs + Security Hub +
#   IAM Identity Center
```

## Exam Traps

```
# Memory/disk metrics need the CloudWatch agent — no agent, no RAM
#   graphs; the metric namespace is CWAgent
# CloudWatch Logs default retention = forever = growing bill; set
#   retention or export/lifecycle to S3
# EC2 auto-RECOVER alarm (StatusCheckFailed_System) preserves
#   instance ID/IP/EBS — distinct from ASG replacing the instance
# S3 object-read auditing needs CloudTrail DATA events (paid,
#   opt-in) — management events alone won't show GetObject
# Config detects, SCP prevents, remediation fixes — match the verb
# Config is Regional: enable per Region (aggregators for the view)
# X-Ray vs CloudWatch: trace path vs measure metrics; "which
#   microservice adds latency" is X-Ray every time
# Session Manager failures = agent/role/endpoint triad, not SSH keys
# StackSets vs stacks: many accounts/Regions vs one
# Change sets preview; drift detection audits manual edits — pick by
#   whether the change already happened
# DeletionPolicy Retain on databases in CFN answers about "safely
#   delete the stack"
# Trusted Advisor "service limit" checks vs Service Quotas console:
#   TA warns, Quotas raises
# Health Dashboard events can trigger EventBridge automation —
#   "auto-open ticket when AWS plans maintenance" pattern
```

## See Also

aws-saa-c03, aws-iam, aws-network-security, aws-cost-optimization, aws-high-availability, aws-compute, aws-decoupling, terraform, prometheus, grafana, opentelemetry, cloud-security, siem, sre-fundamentals

## References

- [CloudWatch User Guide](https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/WhatIsCloudWatch.html)
- [CloudTrail User Guide](https://docs.aws.amazon.com/awscloudtrail/latest/userguide/cloudtrail-user-guide.html)
- [AWS Config Developer Guide](https://docs.aws.amazon.com/config/latest/developerguide/WhatIsConfig.html)
- [X-Ray Developer Guide](https://docs.aws.amazon.com/xray/latest/devguide/aws-xray.html)
- [Systems Manager User Guide](https://docs.aws.amazon.com/systems-manager/latest/userguide/what-is-systems-manager.html)
- [CloudFormation User Guide](https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/Welcome.html)
- [Control Tower User Guide](https://docs.aws.amazon.com/controltower/latest/userguide/what-is-control-tower.html)
- [Trusted Advisor](https://docs.aws.amazon.com/awssupport/latest/user/trusted-advisor.html)
