# AWS High Availability & Disaster Recovery (SAA-C03 Task 2.2)

> Design highly available and fault-tolerant architectures: global infrastructure, Route 53 routing policies, ELB family, Multi-AZ patterns per service, the four DR strategies with RPO/RTO, and single-point-of-failure elimination.

## Global Infrastructure Vocabulary

```
# Region — geographic cluster of AZs (isolated fault domain; most
#   services are Regional)
# Availability Zone — 1+ discrete data centers with independent power/
#   network; AZs in a Region are <100 km apart, linked by low-latency
#   fiber; AZ names (us-east-1a) are shuffled per account — AZ IDs
#   (use1-az4) are stable across accounts (matters for subnet sharing)
# Local Zones — compute/storage extension near metros (single-digit ms)
# Wavelength — AWS inside 5G carrier networks
# Outposts — AWS racks in your data center
# Edge locations — 400+ CloudFront/Route 53/Global Accelerator POPs
#
# HA baseline: Multi-AZ (survive AZ failure) — almost always the
# answer scope. Multi-Region = DR/latency/compliance, higher cost.
# Fault tolerance ≠ HA: fault tolerant = zero interruption (active-
# active, N+1); HA = minimal downtime (failover allowed).
```

## Route 53 (DNS-Level Resilience)

```
# Hosted zones: public (internet) / private (per-VPC internal DNS)
# ALIAS records — AWS-native pointer to ALB/CloudFront/S3 website/
#   API GW/another record IN the zone apex (CNAME can't live at apex);
#   free queries, auto-tracks target IPs → default over CNAME for AWS
#   targets
#
# Routing policies:
# Simple      — one record, no health check
# Weighted    — % split (canary/blue-green; weight 0 = drain)
# Latency     — lowest-RTT Region per user
# Failover    — primary + secondary (requires health check on primary)
# Geolocation — by user country/continent (compliance, localization)
# Geoproximity— by geography + adjustable bias (shift traffic share)
# Multivalue  — up to 8 healthy records, client-side pick (poor-man's LB)
# IP-based    — by client CIDR (known corporate egress ranges)
#
# Health checks: endpoint (HTTP/HTTPS/TCP, interval 30 s or fast 10 s,
#   ~15 global checkers, threshold default 3), calculated (boolean of
#   other checks), CloudWatch-alarm-backed (private resources — no
#   direct probing inside VPC!)
# DNS failover patterns: active-passive (failover policy), active-
#   active (latency/weighted + health checks strip the dead)
# TTL tradeoff: low TTL (30–60 s) = faster failover, more queries
# Route 53 Application Recovery Controller — readiness checks +
#   routing controls for audited Region failover (advanced answer)
```

## Elastic Load Balancing Family

```
# ALB (L7 HTTP/HTTPS)
#   Routing by path/host/header/query/method; target groups of
#   instances/IPs/Lambda/containers (dynamic ports with ECS);
#   WebSockets, HTTP/2, gRPC; authentication (Cognito/OIDC) at the LB;
#   WAF attachable; slow-start; least-outstanding-requests algorithm
#   Client IP → X-Forwarded-For (targets see LB IP)
# NLB (L4 TCP/UDP/TLS)
#   Millions of req/s, static IP / EIP PER AZ (whitelist-friendly),
#   preserves source IP, TLS passthrough possible, PrivateLink
#   backend, health checks TCP/HTTP; zonal DNS names
# GWLB (Gateway LB, L3 GENEVE 6081)
#   Inline scaling of third-party appliances (firewalls/IDS): traffic
#   hairpins through appliance fleet via GWLB endpoints
# CLB — legacy; only correct as "migrate off it"
#
# Shared behaviors:
# Cross-zone load balancing: ALB always on (free); NLB off by default
#   (inter-AZ charges when enabled)
# Connection draining = deregistration delay (default 300 s)
# Health checks mark targets; unhealthy → stop routing (ASG can also
#   REPLACE on ELB health when health-check type = ELB)
# Internal vs internet-facing schemes
# Multi-AZ by design: enable ≥2 AZs; LB nodes scale per AZ
```

## Auto Scaling as an HA Tool

```
# ASG across ≥2 AZs + ELB health checks = self-healing web tier
#   (scaling policies/economics → cs aws-compute)
# Health check types: EC2 (status checks) | ELB (app-level; catches
#   hung app with healthy VM — the "instances unhealthy but not
#   replaced" fix) | custom (SetInstanceHealth)
# Lifecycle hooks — run scripts on launch/terminate (drain, register)
# Instance refresh — rolling replace on new AMI (immutable deploys)
# Warm pools — pre-initialized stopped instances for slow-boot apps
# AZ rebalancing is automatic; min/desired/max sizing; scale-in
#   protection for stateful members
# Standby state — pull an instance for debugging without termination
```

## Multi-AZ Patterns per Service

```
# RDS Multi-AZ instance — synchronous standby, DNS failover 60–120 s,
#   standby is NOT readable; Multi-AZ DB CLUSTER — 2 READABLE standbys,
#   faster failover; read replicas are ASYNC and for scale, but can be
#   promoted (DR) — full detail cs aws-databases
# Aurora — 6 copies across 3 AZs, replica promotion <30 s,
#   reader/writer endpoints, Global Database for cross-Region (RPO ~1 s)
# ElastiCache Redis — Multi-AZ with automatic failover of primary
# EFS — Regional (multi-AZ) by default; One Zone class trades HA for cost
# S3 — 11 nines durability across ≥3 AZs (except One Zone classes);
#   CRR for Region-level protection
# DynamoDB — multi-AZ automatically; Global Tables = multi-Region
#   active-active
# SQS/SNS/Lambda/API GW — inherently multi-AZ Regional services
# EBS — single-AZ! Snapshot to S3 (Regional) or replicate at app level
# NAT gateway — single-AZ! One per AZ + per-AZ route tables, or AZ
#   failure takes out egress for every subnet pointing at it
# VPN — two tunnels; DX — order redundant circuits (SLA needs it)
```

## The Four DR Strategies (Memorize This Table)

| Strategy | RPO | RTO | Standing cost | How |
|---|---|---|---|---|
| Backup & restore | Hours | Hours–24h+ | Cheapest (storage only) | Backups/snapshots to DR Region (AWS Backup copy jobs); rebuild with IaC on disaster |
| Pilot light | Minutes | Tens of minutes–hours | Low | Data layer LIVE (replicated DB, AMIs ready); app servers OFF/zero; scale up + switch DNS on disaster |
| Warm standby | Seconds–minutes | Minutes | Medium | Full stack running SCALED-DOWN in DR Region; scale to production size + fail over |
| Multi-site active-active | ~0 | ~0 (or seconds) | 2× (highest) | Full capacity in both Regions, Route 53 latency/weighted; data layer bidirectional (Global Tables / Aurora Global write forwarding) |

```
# Definitions: RPO = max acceptable data loss (time since last sync);
#              RTO = max acceptable time to restore service
# Question decoding: match the STATED RPO/RTO to the cheapest row that
# satisfies it — paying for active-active when "RTO 4 hours" is the trap
# Pilot light vs warm standby: pilot light core is ON but serves ZERO
# traffic and app fleet is not running; warm standby serves (or can
# instantly serve) at reduced capacity
# Ingredients: AMI/snapshot copy cross-Region, AWS Backup cross-Region
# vaults, S3 CRR, RDS cross-Region read replica, Aurora Global,
# DynamoDB Global Tables, CloudFormation to re-create, Route 53
# failover + health checks, Elastic Disaster Recovery (DRS) for
# continuous block-level server replication (on-prem→AWS or
# Region→Region; RPO seconds, RTO minutes — "lowest RPO for lift-and-
# shift servers, pay only for replication until failover")
```

## Reliability Plumbing

```
# RDS Proxy — managed connection pool in front of RDS/Aurora:
#   absorbs Lambda connection storms, cuts failover time ~66% (proxy
#   re-routes without DNS wait), IAM auth, Secrets Manager creds.
#   Trigger: "too many connections", "Lambda + RDS", "reduce failover"
# Service quotas — soft limits per account/Region (EC2 vCPUs, EIPs...):
#   Service Quotas console/API for increases; DR Region quotas must be
#   PRE-RAISED or failover hits the wall (classic scenario);
#   CloudWatch usage metrics + alarms on approaching quotas
# Throttling/backoff — SDKs retry with exponential backoff + jitter;
#   architecture answer for 429/ThrottlingException storms
# Immutable infrastructure — never patch in place: new AMI → instance
#   refresh / new launch template version → roll; CloudFormation/
#   golden AMIs (EC2 Image Builder); rollback = previous version
# Workload visibility — X-Ray traces (find the failing/slow hop),
#   CloudWatch ServiceLens; synthetic canaries probing endpoints;
#   composite alarms to page on user-facing symptoms
#   (full observability stack → cs aws-monitoring-governance)
# Legacy apps that can't change (exam favorite):
#   - Can't retry/reconnect → RDS Proxy, Multi-AZ everything under it
#   - Hardcoded IP → NLB static IP / EIP; Global Accelerator static
#     anycast IPs in front of multi-Region
#   - License tied to MAC/host → Dedicated Host with host affinity
#   - Single-instance stateful → ASG min=max=1 + EIP + EBS snapshots
#     (auto-heal), or DRS replication
```

## Single Point of Failure Checklist

```
# One EC2 instance          → ASG multi-AZ behind ALB
# One NAT gateway           → NAT per AZ + per-AZ routes
# One AZ for anything       → second AZ (subnets, LB nodes, replicas)
# DB single instance        → Multi-AZ (+ read replicas for scale)
# EBS volume                → snapshots; or move state to EFS/S3
# Region                    → pick a DR row from the table above
# On-prem link              → second DX location, or DX + VPN backup
# Stateful web tier         → externalize sessions (ElastiCache/DynamoDB)
# Cron on one box           → EventBridge Scheduler + Lambda
# Hardcoded config/secrets  → Parameter Store / Secrets Manager
# DNS with static IPs       → ALIAS records, health-checked failover
```

## Exam Traps

```
# Multi-AZ ≠ multi-Region: AZ failure vs Region failure — read which
#   one the question fears
# RDS Multi-AZ standby serves NO reads (use read replicas for that);
#   Multi-AZ DB Cluster standbys DO serve reads
# Route 53 health checks cannot probe private IPs directly — use a
#   CloudWatch-alarm-backed health check
# CNAME at zone apex is invalid — ALIAS
# NLB gives static IPs; ALB does not (put Global Accelerator or NLB
#   in front when "static IP" + L7 features both demanded)
# Cross-zone LB on NLB costs inter-AZ transfer; ALB's is free
# ASG with EC2-type health checks won't catch app hangs — ELB type
# DR strategy cost order (cheap→expensive): backup&restore → pilot
#   light → warm standby → active-active; RTO order is the reverse
# Pre-provision service quotas + AMIs + capacity reservations in the
#   DR Region — failover-day quota tickets are the trap
# EBS is AZ-locked; "attach volume to instance in another AZ" =
#   snapshot → create volume in target AZ
# Aurora Global RPO ~1 s / RTO <1 min beats any self-managed
#   cross-Region database answer
# "Zero downtime AND zero data loss" → active-active + synchronous/
#   quorum data layer; nothing cheaper qualifies
```

## See Also

aws-saa-c03, aws-decoupling, aws-databases, aws-networking, aws-compute, aws-storage, aws-data-protection, aws-monitoring-governance, bcp-drp, sre-fundamentals, chaos-engineering, dns, haproxy, cloud-dns

## References

- [AWS Global Infrastructure](https://aws.amazon.com/about-aws/global-infrastructure/)
- [Route 53 Developer Guide — routing policies](https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/routing-policy.html)
- [Elastic Load Balancing User Guide](https://docs.aws.amazon.com/elasticloadbalancing/latest/userguide/what-is-load-balancing.html)
- [Disaster Recovery of Workloads on AWS (whitepaper)](https://docs.aws.amazon.com/whitepapers/latest/disaster-recovery-workloads-on-aws/disaster-recovery-workloads-on-aws.html)
- [AWS Well-Architected — Reliability Pillar](https://docs.aws.amazon.com/wellarchitected/latest/reliability-pillar/welcome.html)
- [Amazon RDS Proxy](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/rds-proxy.html)
- [AWS Elastic Disaster Recovery](https://docs.aws.amazon.com/drs/latest/userguide/what-is-drs.html)
- [Route 53 Application Recovery Controller](https://docs.aws.amazon.com/r53recovery/latest/dg/what-is-route53-recovery.html)
