# AWS Cost Optimization & Billing Governance (SAA-C03 Domain 4)

> Design cost-optimized architectures: the cost tooling suite (Cost Explorer, Budgets, CUR, Anomaly Detection), commitment strategy, tagging governance, and the per-domain cheapest-correct-answer tables.

## Cost Tooling (Pick-the-Service)

```
# Cost Explorer — interactive spend analysis: filter/group by service,
#   account, tag, Region; 13 months history + 12-month forecast;
#   RI/SP purchase RECOMMENDATIONS + utilization/coverage reports;
#   hourly + resource-level granularity opt-in
#   "Visualize/analyze historical spend, find what grew" → Cost Explorer
# AWS Budgets — thresholds + ACTIONS: cost, usage, RI/SP utilization
#   & coverage budgets; alert at forecast or actual %; actions can
#   apply an SCP or stop EC2/RDS ("alert me BEFORE we overspend" /
#   "stop dev instances at 80% of budget" → Budgets)
# Cost & Usage Report (CUR / Data Exports) — the raw hourly line-item
#   truth → S3, query with Athena/QuickSight ("most detailed billing
#   data / chargeback reports" → CUR + Athena)
# Cost Anomaly Detection — ML unusual-spend alerts (SNS/email) with
#   root-cause hints ("detect unexpected charges automatically")
# Compute Optimizer — ML right-sizing for EC2/ASG/EBS/Lambda/ECS-
#   Fargate ("identify over-provisioned instances" → Compute
#   Optimizer; Cost Explorer has a simpler EC2-only variant)
# Trusted Advisor — best-practice checks incl. cost (idle RDS, low-
#   util EC2, unassociated EIPs, RI optimization); check depth scales
#   with the SUPPORT PLAN (Business+ = all checks)
# Pricing Calculator — pre-deployment estimates
# Billing Conductor — custom rate cards for internal chargeback
# Split cost allocation — ECS/EKS pod-level cost attribution
```

## Multi-Account Billing (Organizations)

```
# Consolidated billing: one payer, aggregated usage →
#   - Volume tiering pools (S3/data transfer tiers reached faster)
#   - RI + Savings Plans DISCOUNT SHARING across accounts (disable
#     per-account if needed) — buy centrally, benefit org-wide
#   - Free tier is per ORG (one per org effectively), not per account
# Cost allocation tags — activate in Billing console (user-defined +
#   aws:createdBy generated); ONLY accrue from activation onward
# Tag governance: Organizations Tag Policies (standardize keys/values)
#   + SCP deny on missing CostCenter tag ("enforce tagging for
#   chargeback" pattern)
# Account-per-team/env is itself a cost boundary (cleanest chargeback)
# Service Catalog restricts what teams can launch (pre-approved,
#   pre-sized products) — cost guardrail by construction
```

## Commitment Strategy (When to Buy What)

```
# Order of operations (exam-approved):
# 1. Eliminate waste (idle, unattached, oversized, orphaned)
# 2. Right-size with Compute Optimizer data
# 3. Modernize pricing units: Graviton, gp3, serverless where spiky
# 4. THEN commit to the remaining steady floor:
#    Compute Savings Plan — flexible (any family/Region/Fargate/
#      Lambda) up to 66%
#    EC2 Instance SP / Standard RI — deepest (72%) when family+Region
#      is stable for 1–3 yrs
#    RDS/ElastiCache/Redshift/OpenSearch RIs; DynamoDB reserved
#      capacity — database commitments are separate purchases
# 5. Spot for anything interruptible (up to 90%) — cs aws-compute
# Utilization/coverage reports (Cost Explorer/Budgets) verify the
#   commitments stay >90% utilized — unused commitment is negative
#   savings
# No-upfront < partial < all-upfront discount depth; 3 yr > 1 yr
```

## Domain 4 Answer Tables (Condensed)

| Storage cost ask (4.1) | Answer |
|---|---|
| Unknown access pattern | S3 Intelligent-Tiering |
| Predictable cooling data | Lifecycle → IA → Glacier tiers |
| Archive, retrieval hours OK | Glacier Deep Archive |
| gp2 fleet | gp3 migration |
| Snapshot sprawl | DLM/AWS Backup retention + archive tier |
| Full detail | `cs aws-storage` |

| Compute cost ask (4.2) | Answer |
|---|---|
| Steady 24/7 | Savings Plan (Compute SP default) |
| Interruptible batch | Spot |
| Spiky/idle-most-of-day | Lambda/Fargate (scale-to-zero) |
| Oversized fleet | Compute Optimizer right-size |
| Nights/weekends idle dev | Scheduler stop/start |
| Full detail | `cs aws-compute` |

| Database cost ask (4.3) | Answer |
|---|---|
| Variable/dev-test relational | Aurora Serverless v2 / RDS stop |
| Steady production DB | RDS/ElastiCache RIs + Graviton classes |
| Spiky NoSQL | DynamoDB on-demand |
| Steady NoSQL | Provisioned + auto scaling (+ reserved) |
| Read-heavy bill | Caching (DAX/ElastiCache) before more replicas |
| Full detail | `cs aws-databases` |

| Network cost ask (4.4) | Answer |
|---|---|
| NAT processing on S3/DynamoDB traffic | Gateway VPC endpoints (free) |
| Internet egress for content | CloudFront in front |
| Chatty cross-AZ tiers | Same-AZ placement (weigh vs HA) |
| Heavy sustained hybrid transfer | Direct Connect per-GB rates |
| Many VPCs, few flows | Peering (no per-GB TGW processing) |
| Full detail | `cs aws-networking` |

## Serverless & Managed as Cost Levers

```
# TCO framing the exam rewards: managed/serverless removes the idle
# capacity AND the ops labor line:
#   EC2 cron box → EventBridge Scheduler + Lambda
#   Self-managed Kafka → MSK (or Kinesis), RabbitMQ → Amazon MQ/SQS
#   Self-managed k8s masters → EKS; VMs for containers → Fargate
#   Nightly ETL cluster → Glue (or EMR Serverless)
#   Always-on dev DB → Aurora Serverless v2 min ACU
# BUT: at high steady utilization, EC2+SP undercuts Fargate/Lambda —
#   "consistently high, predictable" flips the answer back to
#   reserved/committed EC2
```

## Waste Checklist (Free Money)

```
# Unattached EBS volumes + aged manual snapshots
# Unassociated Elastic IPs (billed idle); public IPv4 count generally
# Idle/oversized: <5% CPU instances, empty ALBs/NLBs (hourly), idle
#   NAT gateways in unused AZs, stopped-but-EBS-heavy fleets
# Old-generation families (m3→m7g), gp2→gp3, io1→gp3-if-IOPS-fit
# CloudWatch: unneeded custom metrics/dashboards/log retention =
#   set log group retention (default NEVER expires!)
# S3: incomplete multipart uploads, noncurrent versions,
#   dead buckets with versioning piles — Storage Lens finds them
# Data transfer: SDKs hitting public endpoints via NAT instead of
#   endpoints; cross-Region chatter that could be Regional
# Over-retained backups (35-day PITR when 7 needed; infinite
#   snapshots)
# Dev/test running 168 h/week for a 45 h/week team
```

## Support Plans (Occasional Exam Cameo)

```
# Basic (free) — billing support, core Trusted Advisor checks
# Developer — business-hours email, 1 contact
# Business — 24/7 phone/chat, FULL Trusted Advisor, <1 h urgent
# Enterprise On-Ramp — pool of TAM hours, <30 min critical
# Enterprise — designated TAM, Concierge billing, <15 min critical
# "Production down, need 24/7 phone + all TA checks, cheapest" →
#   Business
```

## Exam Traps

```
# Budgets ALERTS (and can act); it does not analyze history — that's
#   Cost Explorer; raw line items = CUR; surprises = Anomaly Detection
# Cost allocation tags start counting at ACTIVATION — no retroactive
#   chargeback
# RI/SP sharing is org-wide by default — "one account bought RIs,
#   another benefits" is expected behavior, not a bug
# Savings Plans don't cover RDS (RDS has its own RIs); Compute SP
#   covers EC2+Fargate+Lambda only
# Spot ≠ free tier of On-Demand: interruption handling required, and
#   never for stateful/prod-critical singletons
# "Reduce cost with NO code/architecture change" → pricing moves only
#   (SP/RI, gp3, Graviton-if-managed, scheduling) — not "rewrite to
#   Lambda"
# Trusted Advisor full checks need Business+ support — a Basic-plan
#   scenario can't lean on full TA
# Multi-account free-tier farming isn't a thing (org-level); and
#   consolidated billing alone ≠ cost REDUCTION strategy in answers —
#   it's tier pooling + discount sharing
# The cheapest option that MEETS requirements wins — cost answers
#   that violate an RTO/perf/compliance constraint in the stem are
#   the trap direction
```

## See Also

aws-saa-c03, aws-storage, aws-compute, aws-databases, aws-networking, aws-monitoring-governance, aws-high-availability, aws-cli, cloud-security, sre-fundamentals

## References

- [AWS Cost Management User Guide](https://docs.aws.amazon.com/cost-management/latest/userguide/what-is-costmanagement.html)
- [Cost Explorer](https://docs.aws.amazon.com/cost-management/latest/userguide/ce-what-is.html)
- [AWS Budgets](https://docs.aws.amazon.com/cost-management/latest/userguide/budgets-managing-costs.html)
- [Cost and Usage Reports / Data Exports](https://docs.aws.amazon.com/cur/latest/userguide/what-is-cur.html)
- [Savings Plans User Guide](https://docs.aws.amazon.com/savingsplans/latest/userguide/what-is-savings-plans.html)
- [AWS Well-Architected — Cost Optimization Pillar](https://docs.aws.amazon.com/wellarchitected/latest/cost-optimization-pillar/welcome.html)
- [Cost Anomaly Detection](https://docs.aws.amazon.com/cost-management/latest/userguide/manage-ad.html)
- [AWS Pricing Calculator](https://calculator.aws/)
