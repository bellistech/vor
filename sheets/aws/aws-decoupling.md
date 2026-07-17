# AWS Decoupling & Event-Driven Architecture (SAA-C03 Task 2.1)

> Design scalable, loosely coupled architectures: SQS, SNS, EventBridge, API Gateway, Step Functions, MQ, serverless patterns, caching strategies, and container orchestration choices.

## SQS (Simple Queue Service)

```
# Fully managed pull-based queue; the default buffer between tiers.
# Standard queue
#   - Nearly unlimited throughput
#   - At-least-once delivery (duplicates possible)
#   - Best-effort ordering
# FIFO queue (name must end .fifo)
#   - Exactly-once processing, strict order per MessageGroupId
#   - 300 msg/s (3,000 with batching); high-throughput mode: 
#     ~70k msg/s per API action with many message groups
#   - Deduplication: content-based hash or explicit MessageDeduplicationId
#     within a 5-minute window
#
# Core mechanics:
# Visibility timeout (default 30 s, max 12 h) — message hidden while a
#   consumer processes; crash → message reappears (this is the retry)
# Long polling (WaitTimeSeconds up to 20 s) — fewer empty receives,
#   lower cost; ALWAYS the right answer vs short polling
# Retention 4 days default, 1 min – 14 days max
# Message size 256 KB (use S3 + Extended Client Library for larger)
# Delay queues (up to 15 min) & per-message delay (standard only)
# Dead-letter queue (DLQ) — after maxReceiveCount failures, message
#   moves to DLQ for inspection; redrive back after fixing
# Temporary credentials + SSE-KMS supported; queue policy for
#   cross-account producers/consumers
```

```bash
aws sqs create-queue --queue-name orders.fifo \
  --attributes FifoQueue=true,ContentBasedDeduplication=true
aws sqs send-message --queue-url $Q --message-body '{"id":42}' \
  --message-group-id tenant-7
aws sqs receive-message --queue-url $Q --wait-time-seconds 20 \
  --max-number-of-messages 10
aws sqs delete-message --queue-url $Q --receipt-handle $RH  # ack = delete
```

```
# Scaling pattern (exam staple): producers → SQS → Auto Scaling group
# scaling on ApproximateNumberOfMessagesVisible / backlog-per-instance
# target; or SQS → Lambda event source mapping (batch size, concurrency
# limit, partial batch responses via ReportBatchItemFailures)
```

## SNS (Simple Notification Service)

```
# Push-based pub/sub: one topic → up to 12.5M subscriptions
# Subscribers: SQS, Lambda, HTTPS, email, SMS, mobile push, Kinesis
#   Data Firehose (NOT Kinesis Data Streams)
# Standard topics (at-least-once, best-effort order) and FIFO topics
#   (order + dedup; FIFO topic → FIFO queue subscribers)
# Message filtering — subscription filter policies on message
#   attributes (or body): each subscriber receives only matching
#   messages; kills the "consumer-side if-statement" pattern
# Fan-out (THE pattern): producer → SNS topic → multiple SQS queues →
#   independent consumers at their own pace, with per-consumer DLQs.
#   "New order must trigger invoice + shipping + analytics" → fan-out
# Delivery retries with backoff for HTTPS; DLQ per subscription
# Raw message delivery — strip SNS envelope for SQS/HTTP subscribers
```

## EventBridge (The Event Router)

```
# Serverless event bus: default bus (AWS service events), custom buses,
# partner buses (SaaS: Datadog, Zendesk, Auth0...)
# Rules match event PATTERNS (JSON field matching) or run on a
#   SCHEDULE (cron/rate — EventBridge Scheduler is the newer, richer
#   scheduling service with timezones + one-off schedules)
# 20+ target types: Lambda, Step Functions, SQS, SNS, Kinesis, ECS
#   tasks, API destinations (signed HTTP to external APIs), another bus
# Cross-account/cross-Region event routing (central event bus pattern)
# Archive + replay events; schema registry with code bindings
# Input transformer reshapes the event before delivery
#
# SNS vs EventBridge:
#   SNS — mass fan-out, mobile/SMS/email, millions of subscribers,
#         lowest latency
#   EventBridge — content-based ROUTING on the event body, SaaS
#         ingestion, AWS service events, scheduling, archive/replay
# "React to AWS API calls / service state changes" → EventBridge
# (GuardDuty finding → EventBridge → Lambda remediation, etc.)
```

## API Gateway

```
# Managed front door for REST/HTTP/WebSocket APIs.
# REST API — full feature set: API keys + usage plans (throttle/quota
#   per client), request/response mapping (VTL), caching (0.5–237 GB,
#   per-stage, TTL 0–3600 s), canary deployments, WAF, resource policy
# HTTP API — ~70% cheaper, lower latency, JWT authorizers built in;
#   lacks usage plans/API keys/caching — "lowest cost API layer" answer
# WebSocket API — stateful two-way (chat, live dashboards)
#
# Integrations: Lambda proxy, HTTP backend, ANY AWS service (e.g.
#   direct PutRecord to Kinesis — no Lambda glue needed), VPC Link to
#   reach private ALB/NLB
# Endpoint types: edge-optimized (CloudFront-fronted), Regional
#   (pair with your own CloudFront/latency routing), private (invoke
#   from inside VPC via interface endpoint)
# Auth options: IAM SigV4, Cognito user pool authorizer, Lambda
#   authorizer (custom token logic), JWT (HTTP APIs), mTLS
# Throttling: default 10,000 req/s per account per Region, burst 5,000;
#   429 TooManyRequests on breach; per-method + per-client (usage plan)
#   overrides; retry with exponential backoff + jitter
# Timeout: 29 s integration max (hard) — long work goes async:
#   API GW → SQS/Step Functions, return 202 + status URL
```

## Step Functions (Workflow Orchestration)

```
# Serverless state machines (Amazon States Language JSON):
# Task, Choice, Parallel, Map (fan-out over array; distributed Map for
# millions of S3 objects), Wait, Pass, Succeed, Fail states
# Per-state retry/backoff/catch — "add retries between microservice
#   steps without code" → Step Functions
# Service integrations: Lambda, ECS/Fargate, Batch, DynamoDB, SNS/SQS,
#   Glue, EMR, SageMaker, nested workflows, ~220 services via SDK
#   integrations
# Patterns: request-response | run-a-job (.sync) | wait-for-callback
#   (.waitForTaskToken — human approval step answer)
# Standard workflows — exactly-once, up to 1 year, priced per state
#   transition; full history
# Express workflows — at-least-once, ≤5 min, priced per request+duration;
#   high-volume event processing (100k/s)
# vs SWF: SWF is legacy; only correct when "external signals + child
#   processes + code-level control" — effectively never
# Saga pattern: compensating transactions on Catch → undo steps
```

## Amazon MQ, AppFlow, AppSync

```
# Amazon MQ — managed ActiveMQ/RabbitMQ. ONLY correct when migrating
#   apps that already speak AMQP/MQTT/OpenWire/STOMP "without code
#   changes". New cloud apps → SQS/SNS. Active/standby via EFS-backed
#   brokers across AZs.
# AppFlow — no-code SaaS↔AWS data transfer (Salesforce → S3/Redshift),
#   scheduled/event/on-demand, field mapping + filtering
# AppSync — managed GraphQL: single endpoint aggregating DynamoDB/
#   Lambda/HTTP sources; real-time subscriptions (WebSocket), offline
#   sync. "GraphQL" or "real-time collaborative app data layer" → AppSync
```

## Serverless Compute Patterns

```
# Lambda essentials for architecture questions:
#   Timeout max 15 min; memory 128 MB–10 GB (CPU scales with memory);
#   /tmp 512 MB–10 GB; concurrency default 1,000/Region (soft);
#   reserved concurrency (cap + guarantee), provisioned concurrency
#   (pre-warmed — "eliminate cold starts")
#   Triggers: API GW, ALB, S3, SQS, SNS, EventBridge, Kinesis/DynamoDB
#   streams (poll-based with batch windows), cron
#   VPC Lambda: needs ENIs in subnets; reach internet via NAT; reach
#   AWS APIs via endpoints; RDS access → RDS Proxy (connection pooling)
#   SnapStart (Java/Python/.NET) cuts cold starts via snapshots
# Fargate — serverless containers for ECS/EKS: no instances to manage;
#   per-task vCPU/GB pricing; "containers without managing servers"
# Choose: event-driven, <15 min, spiky → Lambda;
#         long-running/custom runtime/steady → Fargate;
#         full node control / GPUs / daemonsets → EC2-backed
```

## Containers: ECS vs EKS (Orchestration Choice)

```
# ECS — AWS-native orchestrator: task definitions, services, ALB
#   integration, capacity providers (Fargate / Fargate Spot / EC2 ASG)
#   Simplest ops; "AWS-only, least complexity" answer
# EKS — managed Kubernetes control plane: portability, existing k8s
#   skills/manifests, hybrid (EKS Anywhere), add-ons ecosystem
#   "Kubernetes / multi-cloud portability / existing k8s tooling" → EKS
# ECR — registry (image scanning, cross-Region replication, lifecycle)
# Migration path (exam): containerize app → push to ECR → run on
#   ECS/Fargate behind ALB; stateful data OUT of the container
#   (EFS volumes mount into both ECS and EKS tasks)
# App2Container — CLI that containerizes existing .NET/Java apps
```

## Caching Strategies

```
# Where to cache (outermost first):
# CloudFront — static + dynamic edge caching, per-behavior TTLs
# API Gateway stage cache — REST APIs, per-method, TTL ≤ 3600 s
# ElastiCache Redis/Memcached — DB offload, sessions (details:
#   cs aws-databases)
# DAX — DynamoDB-native, microsecond reads, write-through
# Patterns: cache-aside (lazy load, stale risk + TTL), write-through
#   (fresh, write penalty, cold-start empty), write-behind;
#   stampede control: TTL jitter, request coalescing
# Session state exam answer: sticky sessions couple users to instances
#   (breaks scale-in) → externalize to ElastiCache/DynamoDB
```

## Loose-Coupling Litmus Table

| Requirement phrase | Answer |
|---|---|
| Buffer writes, absorb spikes, retry safely | SQS between tiers |
| Strict order + no duplicates | SQS FIFO (or Kinesis if streaming/replay) |
| One event, many independent consumers | SNS → SQS fan-out |
| Route by event content; SaaS/AWS events; cron | EventBridge |
| Multi-step process, retries, human approval | Step Functions |
| Legacy AMQP/JMS app, no code changes | Amazon MQ |
| Throttle/monetize/expose APIs | API Gateway (+usage plans) |
| Slow backend behind a 29 s API limit | API GW → SQS/Step Functions, 202 + poll |
| Replayable ordered stream, multiple readers | Kinesis Data Streams (cs aws-data-analytics) |
| Direct service-to-service sync call | Consider making it async (queue) first |

## Exam Traps

```
# SQS is PULL, SNS is PUSH — "push notification to millions" is SNS
# SQS does not push to Lambda conceptually — event source mapping
#   polls for you (still "SQS triggers Lambda" in answer options)
# Visibility timeout < processing time = duplicate processing;
#   set timeout ≈ 6× the Lambda timeout when Lambda consumes
# FIFO throughput limits are real: high-scale ordered streaming with
#   replay → Kinesis, not SQS FIFO
# SNS message filtering beats "subscriber discards 90% of messages"
# Standard queue duplicates: consumers must be IDEMPOTENT — design
#   answer even when not asked
# API Gateway 29 s hard integration timeout — no raising it
#   (long jobs → async pattern); payload cap 10 MB
# Lambda 15 min cap: video transcodes/batch jobs → Fargate/Batch
# Step Functions Standard vs Express: audit/history/1-year → Standard;
#   100k/s short events → Express
# EventBridge Scheduler vs cron rules: one-time schedules, timezones,
#   flexible windows → Scheduler (newer answer AWS prefers)
# Amazon MQ does not auto-scale like SQS; it's for protocol
#   compatibility, never for "unlimited scale"
# Sticky sessions = vertical-era thinking; externalize session state
```

## See Also

aws-saa-c03, aws-high-availability, aws-compute, aws-databases, aws-data-analytics, aws-networking, lambda, serverless-patterns, api-gateway, event-driven-architecture, microservices-patterns, kafka, rabbitmq, caching-patterns

## References

- [Amazon SQS Developer Guide](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/welcome.html)
- [Amazon SNS Developer Guide](https://docs.aws.amazon.com/sns/latest/dg/welcome.html)
- [Amazon EventBridge User Guide](https://docs.aws.amazon.com/eventbridge/latest/userguide/eb-what-is.html)
- [API Gateway Developer Guide](https://docs.aws.amazon.com/apigateway/latest/developerguide/welcome.html)
- [Step Functions Developer Guide](https://docs.aws.amazon.com/step-functions/latest/dg/welcome.html)
- [Lambda Developer Guide](https://docs.aws.amazon.com/lambda/latest/dg/welcome.html)
- [Amazon ECS Best Practices](https://docs.aws.amazon.com/AmazonECS/latest/bestpracticesguide/intro.html)
- [AWS Well-Architected — Serverless Applications Lens](https://docs.aws.amazon.com/wellarchitected/latest/serverless-applications-lens/welcome.html)
