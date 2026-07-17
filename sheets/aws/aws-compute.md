# AWS Compute Selection & Scaling (SAA-C03 Tasks 3.2 + 4.2)

> Design high-performing, elastic, cost-optimized compute: EC2 instance families and purchase options, Auto Scaling policies, Lambda sizing, containers, Batch/EMR, and the serverless-vs-managed-vs-EC2 decision.

## EC2 Instance Families (Letter Soup Decoder)

```
# General purpose  M (m7g/m7i)  — balanced; T (t3/t4g) burstable with
#   CPU credits (unlimited mode = pay for overage instead of throttle)
# Compute optimized C — batch, HPC, game servers, encoding (high
#   CPU:RAM ratio)
# Memory optimized  R — in-memory caches/DBs, big JVMs;
#   X / High Memory — SAP HANA, huge in-memory
#   z — high per-core clock + memory (licensed-per-core DBs)
# Storage optimized I (NVMe IOPS — NoSQL/OLTP), D/H (dense HDD —
#   Hadoop/file servers)
# Accelerated       P/G (GPU training/graphics), Inf/Trn (ML inference/
#   training silicon), F (FPGA)
# Suffixes: g = Graviton (ARM, ~40% better price/perf — the "reduce
#   cost, we can recompile" answer), i = Intel, a = AMD,
#   d = local NVMe disk, n = network-enhanced (up to 200 Gbps),
#   flex = mostly-cheaper M/C variants
# Nitro system — hardware-offloaded hypervisor: near-bare-metal perf,
#   required for 64k+ EBS IOPS; .metal = no hypervisor at all
# Sizing reflex: pick FAMILY by bottleneck (CPU/RAM/IO/GPU), SIZE by
#   load test; Compute Optimizer recommends from actual metrics
```

## Purchase Options (Task 4.2 Core)

```
# On-Demand         — no commitment; spiky/unknown; dev/test
# Reserved Instances (RI)
#   Standard RI     — up to 72% off; 1/3 yr; locked family/Region
#                     (size-flexible within family on Linux/shared);
#                     sellable on RI Marketplace
#   Convertible RI  — up to 66% off; exchange family/OS during term
# Savings Plans (the modern default answer)
#   Compute SP      — up to 66% off; $/hr commit; applies across EC2
#                     ANY family/Region + Fargate + Lambda
#   EC2 Instance SP — up to 72% off; locked family+Region (like RI,
#                     more flexible sizing)
# Spot              — up to 90% off spare capacity; 2-min interruption
#                     notice (EventBridge/IMDS); price = current spot
#                     (bid caps optional)
#   → fault-tolerant, stateless, checkpointable: batch, CI, rendering,
#     big data, containers. NEVER: databases, stateful single nodes
# Dedicated Instance — hardware not shared with other customers
# Dedicated Host     — physical host you control: BYOL per-socket/core
#                      licenses (Windows Server, SQL, Oracle), host
#                      affinity — "license compliance" answer
# Capacity Reservation — guarantees capacity in an AZ, NO discount
#   (combine with SP/RI for discount+guarantee); On-Demand pricing
#   while unused — DR Region capacity insurance
#
# Mix pattern (exam favorite): baseline steady load on SP/RI +
#   variable on On-Demand + opportunistic on Spot (ASG mixed-instances
#   policy with allocation strategies: price-capacity-optimized for
#   Spot is the recommended default)
```

## Placement Groups, AMIs, Fleet Mechanics

```
# Placement groups:
#   Cluster   — same rack-ish, lowest latency/highest PPS (HPC, MPI);
#               single-AZ; capacity errors → stop/start all together
#   Spread    — each instance on distinct hardware; max 7/AZ;
#               "small set of critical instances must not share
#               hardware" (HA quorum nodes)
#   Partition — groups on isolated racks (up to 7/AZ, 100s of
#               instances); HDFS/Kafka/Cassandra topology awareness
# AMI — region-scoped template (copy cross-Region for DR); golden AMIs
#   via EC2 Image Builder pipelines (build+test+distribute schedule)
# Launch template (NOT legacy launch configuration) — versioned; the
#   answer for anything ASG-config related
# Hibernate — RAM to encrypted EBS; resume with warm caches ("long
#   initialization" fix); stopped = no compute billing
# vCPU-based service quotas cap concurrent On-Demand/Spot per Region
# EC2 Fleet / Spot Fleet — mixed purchase-option fleets by target
#   capacity (ASG mixed-instances covers most exam cases)
```

## Auto Scaling Policies (Task 3.2 Core)

```
# Target tracking — hold a metric at target (CPU 50%, ALB
#   RequestCountPerTarget, custom like backlog-per-instance);
#   DEFAULT answer for "scale automatically"
# Step scaling    — add N instances per alarm breach band
# Simple scaling  — legacy single-step + cooldown
# Scheduled       — known cycles ("9am Monday surge" → pre-scale)
# Predictive      — ML forecast from history, pre-launches for daily/
#   weekly patterns ("recurring pattern + slow-warm instances")
# Cooldown (default 300 s) / warmup — prevent thrash; lifecycle hooks
#   for slow bootstraps; warm pools of pre-initialized instances
# Metrics that scale WRONG: memory (not native — needs CloudWatch
#   agent), queue depth alone (use backlog PER INSTANCE)
# Scale-in protection; termination policy (default: AZ balance →
#   oldest launch template → closest to billing hour)
# AWS Auto Scaling (the umbrella service) — scaling plans across EC2,
#   ECS, DynamoDB, Aurora replicas; Application Auto Scaling for
#   non-EC2 targets (DynamoDB, ECS services, Kinesis shards...)
```

## Lambda Sizing & Cost

```
# Memory 128 MB–10,240 MB; vCPU scales linearly (~1 vCPU @ 1,769 MB,
#   6 vCPU max) — CPU-bound? RAISE MEMORY (often faster AND cheaper:
#   billed GB-ms, shorter runtime can offset bigger GB)
# Billing: requests + GB-ms (1 ms granularity); free tier 1M req +
#   400k GB-s/month; Graviton arm64 = ~20% cheaper per GB-s
# Ephemeral /tmp 512 MB (free)–10 GB; container images to 10 GB;
#   zip 250 MB unzipped; layers for shared deps
# Concurrency: 1,000/Region default (soft); reserved concurrency =
#   cap + carve-out (also throttles to protect downstream DB);
#   provisioned concurrency = pre-warmed (+ its own hourly cost) —
#   "eliminate cold starts" answer; SnapStart = free cold-start fix
#   for Java/Python/.NET
# Power tuning (open-source state machine) finds the cheapest memory
# 15-min ceiling → longer work: Fargate, Batch, Step Functions chunks
```

## Containers & PaaS Cost Angles

```
# Fargate — per-second vCPU+GB; no idle instances; Fargate Spot (~70%
#   off, interruptible) for ECS
# ECS on EC2 — cheaper at high, steady utilization (bin-packing +
#   SP/RI/Spot on the instances); you patch/scale the fleet
# EKS — 0.10 USD/hr per cluster + workers (EC2 or Fargate);
#   Karpenter/Cluster Autoscaler for node scaling
# Elastic Beanstalk — PaaS wrapper (ALB+ASG+EC2+logs) for web apps/
#   workers: "deploy code without managing infrastructure, keep
#   instance access" — you pay only for resources; supports docker,
#   blue/green via environment swap (CNAME)
# App Runner — web-service PaaS from container/repo, scales to zero-ish
# Lightsail — fixed-price VPS bundles ("simple, predictable monthly
#   price, low ops" small-business answer)
```

## Batch, EMR, Edge Compute

```
# AWS Batch — managed batch scheduler: job queues + compute
#   environments (EC2/Spot/Fargate), array jobs, dependencies, retries;
#   picks instance types itself. "Run 100k containerized jobs
#   overnight cheaply" → Batch on Spot
# EMR — managed Hadoop/Spark/Hive/Presto/Flink clusters:
#   master + core (HDFS) + task nodes (no HDFS → PERFECT for Spot);
#   EMR on EKS / EMR Serverless variants; S3 via EMRFS as the data
#   lake ("lift existing Spark/Hadoop" → EMR; SQL-only ad hoc →
#   Athena instead)
# Outposts — AWS racks on-prem (data residency, local latency to
#   factory/hospital systems)
# Local Zones — metro-edge AZ-like zones (single-digit ms to a city)
# Wavelength — compute embedded in 5G networks (mobile edge)
# Choose by phrase: "on-premises due to regulation" → Outposts;
#   "<10 ms to users in Los Angeles" → Local Zone; "5G devices" →
#   Wavelength; "global viewers" → CloudFront (cs aws-networking)
```

## Compute Selection Table

| Requirement | Answer |
|---|---|
| Event-driven, bursty, <15 min units | Lambda |
| Containers, no server management | Fargate (ECS) |
| Kubernetes required/portability | EKS |
| Steady 24/7 high utilization | EC2 + Savings Plan (or ECS on EC2) |
| Interruptible batch/CI/rendering | Spot (ASG mixed / Batch / Fargate Spot) |
| Massive parallel job queue | AWS Batch |
| Spark/Hadoop ecosystem | EMR (task nodes on Spot) |
| Deploy web app fast, minimal ops, keep EC2 control | Elastic Beanstalk |
| Per-socket BYOL licensing | Dedicated Host |
| HPC tightly-coupled MPI | EC2 cluster placement group + EFA + FSx Lustre |
| Cut cost, workload recompiles fine | Graviton (g suffix, arm64 Lambda) |
| Guarantee DR capacity | On-Demand Capacity Reservation |

## Cost Optimization Moves (Task 4.2)

```
# 1. Right-size first — Compute Optimizer + CloudWatch (idle <40% CPU
#    → smaller size or burstable T family)
# 2. Commit second — Savings Plans on the steady floor (Cost Explorer
#    SP recommendations); Compute SP unless family-locked forever
# 3. Spot everything interruptible — mixed-instances ASG,
#    price-capacity-optimized, diversify instance types (≥10 pools)
# 4. Graviton where possible — ~40% price/perf, also Lambda/Fargate/RDS
# 5. Kill idle — stop dev/test nights+weekends (Instance Scheduler /
#    EventBridge Scheduler + Lambda); delete unattached EBS/EIPs
#    (unattached EIPs bill hourly!)
# 6. Scale to demand — ASG target tracking beats static fleets
# 7. Tiered environments — prod on SP, staging on Spot, dev burstable
# 8. Turn on Cost Explorer resource-level granularity + tags →
#    cs aws-cost-optimization for governance tooling
```

## Exam Traps

```
# Spot for stateful/database workloads = never correct
# Spot interruption gives 2 MINUTES — architectures must checkpoint
#   (to S3) or drain (Spot + ALB connection draining)
# RI/SP are BILLING constructs — they don't reserve capacity
#   (Capacity Reservations do; zonal RIs do as a legacy quirk)
# T-family burstable: sustained 100% CPU on standard mode = throttled
#   to baseline (credits gone); unlimited mode = surprise charges —
#   sustained load belongs on M/C families
# Cluster placement group = performance, single AZ (NOT HA);
#   spread = HA, 7-per-AZ cap; know which is asked
# Lambda: more memory can be CHEAPER for CPU-bound work (runs shorter)
# Provisioned concurrency costs while idle; reserved concurrency is
#   free (it's a limit, not warm capacity)
# Vertical scaling requires stop/start (EBS-backed) and has a ceiling —
#   horizontal via ASG is the scalable answer
# Hibernate ≠ stop: hibernate preserves RAM, needs encrypted root,
#   size limits; billed for EBS only while hibernated
# EMR task nodes on Spot is safe (no HDFS); core nodes on Spot risks
#   data — task=Spot, core=On-Demand is the pattern
# "Unpredictable traffic, must not manage capacity" → serverless
#   (Lambda/Fargate), not a bigger ASG
# Scheduled scaling for KNOWN spikes beats reactive target tracking
#   (metrics lag; instances take minutes to warm)
```

## See Also

aws-saa-c03, aws-decoupling, aws-high-availability, aws-storage, aws-databases, aws-cost-optimization, aws-monitoring-governance, lambda, docker, kubernetes, terraform, spark, kvm

## References

- [EC2 instance types](https://aws.amazon.com/ec2/instance-types/)
- [EC2 pricing models](https://aws.amazon.com/ec2/pricing/)
- [Savings Plans User Guide](https://docs.aws.amazon.com/savingsplans/latest/userguide/what-is-savings-plans.html)
- [Spot Instances best practices](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/spot-best-practices.html)
- [EC2 Auto Scaling User Guide](https://docs.aws.amazon.com/autoscaling/ec2/userguide/what-is-amazon-ec2-auto-scaling.html)
- [Placement groups](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/placement-groups.html)
- [Lambda memory and computing power](https://docs.aws.amazon.com/lambda/latest/operatorguide/computing-power.html)
- [AWS Batch User Guide](https://docs.aws.amazon.com/batch/latest/userguide/what-is-batch.html)
- [AWS Compute Optimizer](https://docs.aws.amazon.com/compute-optimizer/latest/ug/what-is-compute-optimizer.html)
