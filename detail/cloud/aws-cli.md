# The Mathematics of AWS CLI — Cloud API Internals

> *The AWS CLI is a wrapper around the AWS SDK, translating commands into signed HTTP requests against regional API endpoints. Understanding request signing, pagination models, rate limiting, and cost estimation formulas gives you control over the largest cloud platform.*

---

## 1. Request Signing (SigV4 Algorithm)

### The Problem

Every AWS API request must be cryptographically signed using Signature Version 4. This is a multi-step HMAC chain.

### The Signing Chain

$$\text{DateKey} = \text{HMAC-SHA256}(\text{"AWS4"} \| K_{secret}, \text{date})$$
$$\text{RegionKey} = \text{HMAC-SHA256}(\text{DateKey}, \text{region})$$
$$\text{ServiceKey} = \text{HMAC-SHA256}(\text{RegionKey}, \text{service})$$
$$\text{SigningKey} = \text{HMAC-SHA256}(\text{ServiceKey}, \text{"aws4\_request"})$$
$$\text{Signature} = \text{HMAC-SHA256}(\text{SigningKey}, \text{StringToSign})$$

### String to Sign

$$\text{StringToSign} = \text{Algorithm} \| \text{Timestamp} \| \text{CredentialScope} \| \text{Hash}(\text{CanonicalRequest})$$

### Why This Matters

Each HMAC is $O(n)$ where $n$ = input length. For bulk operations (10,000 API calls), signing overhead:

$$T_{signing} = 10{,}000 \times T_{hmac} \approx 10{,}000 \times 0.01\text{ms} = 100\text{ms}$$

Negligible — the network round trip dominates.

---

## 2. Pagination Model (Token-Based Iteration)

### The Problem

AWS APIs return paginated results. Understanding pagination math prevents incomplete data and excess API calls.

### The Pagination Formula

$$\text{API calls} = \lceil N_{total} / P_{size} \rceil$$

Where $P_{size}$ = page size (varies by API).

| API | Default Page Size | Max Page Size |
|:---|:---:|:---:|
| ec2 describe-instances | 1,000 | 1,000 |
| s3 list-objects-v2 | 1,000 | 1,000 |
| iam list-users | 100 | 1,000 |
| dynamodb scan | 1 MB of data | 1 MB |

### Worked Example: Listing 15,000 S3 Objects

$$\text{API calls} = \lceil 15{,}000 / 1{,}000 \rceil = 15$$

$$T_{total} = 15 \times T_{api} \approx 15 \times 100\text{ms} = 1.5\text{s}$$

### `--paginator` Auto-Pagination

With `--no-paginate`: returns first page only ($P_{size}$ results).
With default (auto): iterates all pages sequentially.

$$T_{auto} = \lceil N / P \rceil \times T_{api}$$

### DynamoDB Scan Pagination (Data-Based)

DynamoDB paginates by data volume, not item count:

$$\text{Pages} = \lceil S_{total} / 1\text{MB} \rceil$$

$$\text{RCU consumed} = \lceil S_{total} / 4\text{KB} \rceil \times 0.5 \quad \text{(eventually consistent)}$$

---

## 3. Rate Limiting and Throttling

### The Problem

AWS APIs have rate limits per account per region. Exceeding them triggers HTTP 429 (Throttling).

### Rate Limit Model

$$\text{Requests remaining} = \min(\text{bucket\_max}, \text{bucket} + \text{refill\_rate} \times \Delta t) - 1$$

This is a **token bucket** algorithm:

$$B(t) = \min(B_{max}, B(t-1) + r \times \Delta t)$$

| Service | Bucket Size | Refill Rate | Sustained Rate |
|:---|:---:|:---:|:---:|
| EC2 (describe) | 100 | 20/s | 20 req/s |
| EC2 (mutate) | 50 | 5/s | 5 req/s |
| S3 (GET) | 5,500 | 5,500/s/prefix | 5,500/s |
| S3 (PUT) | 3,500 | 3,500/s/prefix | 3,500/s |
| IAM | 100 | 20/s | 20 req/s |

### Retry with Exponential Backoff

AWS CLI automatic retry:

$$T_{wait}(n) = \min(2^n \times 100\text{ms} \times (1 + \text{rand}()), 20\text{s})$$

| Retry | Base Wait | With Jitter (avg) | Cumulative |
|:---:|:---:|:---:|:---:|
| 1 | 200 ms | ~300 ms | 0.3s |
| 2 | 400 ms | ~600 ms | 0.9s |
| 3 | 800 ms | ~1.2s | 2.1s |
| 4 | 1.6s | ~2.4s | 4.5s |
| 5 | 3.2s | ~4.8s | 9.3s |

Default max retries: 2 (standard mode), 5 (adaptive mode).

---

## 4. S3 Transfer Performance (Multipart Upload)

### The Problem

Large file transfers use multipart upload. Part size and concurrency affect throughput.

### Multipart Upload Formula

$$\text{Parts} = \lceil S_{file} / S_{part} \rceil$$

$$T_{upload} = \frac{\text{Parts}}{C_{concurrent}} \times \frac{S_{part}}{BW}$$

### Constraints

$$S_{part} \in [5\text{MB}, 5\text{GB}]$$
$$\text{Parts} \leq 10{,}000$$
$$\therefore S_{file} \leq 10{,}000 \times 5\text{GB} = 50\text{TB}$$

### Optimal Part Size

$$S_{part}^{opt} = \max\left(5\text{MB}, \lceil S_{file} / 10{,}000 \rceil\right)$$

### Worked Example: 10 GB Upload

With 100 Mbps bandwidth:

$$S_{part} = \max(5\text{MB}, \lceil 10\text{GB} / 10{,}000 \rceil) = 5\text{MB}$$
$$\text{Parts} = \lceil 10{,}000 / 5 \rceil = 2{,}000$$

Sequential: $T = 2{,}000 \times \frac{5 \times 8}{100} = 800\text{s}$

Parallel ($C=10$): $T = \frac{2{,}000}{10} \times 0.4 = 80\text{s}$

$$\text{Speedup} = 10\times$$

---

## 5. Cost Estimation Formulas

### EC2 Instance Cost

$$\text{Monthly cost} = \text{hourly rate} \times 730 \text{ hours}$$

### S3 Storage Cost

$$C_{S3} = S_{stored} \times R_{per\_GB} + N_{requests} \times R_{per\_request} + S_{transferred} \times R_{per\_GB\_out}$$

### Data Transfer Cost (Tiered)

| Tier | Volume/Month | Rate (us-east-1) |
|:---|:---:|:---:|
| First 10 TB | 0 - 10 TB | $0.09/GB |
| Next 40 TB | 10 - 50 TB | $0.085/GB |
| Next 100 TB | 50 - 150 TB | $0.07/GB |
| Over 150 TB | 150+ TB | $0.05/GB |

### Worked Example: Monthly S3 Bill

- Storage: 500 GB Standard = $500 \times 0.023 = \$11.50$
- PUT requests: 100,000 = $100{,}000 \times 0.000005 = \$0.50$
- GET requests: 1,000,000 = $1{,}000{,}000 \times 0.0000004 = \$0.40$
- Data out: 100 GB = $100 \times 0.09 = \$9.00$

$$C_{total} = 11.50 + 0.50 + 0.40 + 9.00 = \$21.40$$

---

## 6. IAM Policy Evaluation (Boolean Logic)

### The Problem

IAM evaluates policies using a deterministic algorithm. The result is Allow, Deny, or Implicit Deny.

### Evaluation Algorithm

$$\text{Decision} = \begin{cases}
\text{Deny} & \text{if any explicit Deny matches} \\
\text{Allow} & \text{if any Allow matches AND no Deny} \\
\text{Implicit Deny} & \text{otherwise (default)}
\end{cases}$$

### Effective Permissions

$$\text{Effective} = \left(\bigcup_{p \in \text{policies}} \text{Allow}(p)\right) \setminus \left(\bigcup_{p \in \text{policies}} \text{Deny}(p)\right)$$

Explicit Deny always wins — it's a logical AND-NOT:

$$\text{Can do X} = (\exists \text{Allow for X}) \wedge (\nexists \text{Deny for X})$$

### Policy Size Limits

| Limit | Value |
|:---|:---:|
| Managed policy size | 6,144 chars |
| Inline policy size | 2,048 chars |
| Policies per user | 10 |
| Policies per role | 10 |
| Roles per account | 1,000 (default) |

---

## 7. CloudWatch Metrics (Statistical Aggregation)

### The Problem

CloudWatch stores metrics as datapoints and aggregates them over periods.

### Aggregation Functions

$$\text{Average}(p) = \frac{\sum_{d \in \text{period}} v_d}{|d|}$$
$$\text{Sum}(p) = \sum_{d \in \text{period}} v_d$$
$$\text{Maximum}(p) = \max_{d \in \text{period}} v_d$$
$$\text{p99}(p) = \text{99th percentile of } \{v_d\}$$

### Metric Resolution

| Resolution | Datapoint Interval | Retention |
|:---|:---:|:---:|
| Standard | 60s | 15 days (1-min), 63 days (5-min), 455 days (1-hr) |
| High-res | 1s | 3 hours (1-sec), then standard |

### Alarm Evaluation

$$\text{ALARM} \iff \text{metric}(stat, period) \geq \text{threshold} \text{ for } N \text{ of } M \text{ periods}$$

---

## 8. Summary of Functions by Type

| Formula | Math Type | Domain |
|:---|:---|:---|
| HMAC-SHA256 chain | Cryptographic | Request signing |
| $\lceil N/P \rceil$ | Ceiling division | Pagination |
| $B_{max}$, token refill | Token bucket | Rate limiting |
| $2^n \times 100\text{ms}$ | Exponential backoff | Retry logic |
| $\text{Allow} \setminus \text{Deny}$ | Set difference | IAM evaluation |
| Tiered pricing | Piecewise linear | Cost estimation |

---

*Every `aws` command becomes an HMAC-signed HTTPS request subject to pagination, rate limits, and IAM evaluation. The CLI abstracts this, but the math determines your throughput, cost, and access control.*

## Prerequisites

- AWS account and IAM fundamentals (users, roles, policies)
- HTTP/HTTPS and REST API concepts
- JSON and JMESPath query syntax
- Understanding of cloud regions and availability zones

## Complexity

- Beginner: configure profiles, S3 operations, EC2 listing
- Intermediate: IAM policy authoring, STS role assumption, CloudFormation, JMESPath queries
- Advanced: SigV4 signing internals, rate limit handling, pagination strategies, cost optimization
