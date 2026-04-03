# The Mathematics of gcloud CLI — Cloud API Internals

> *The gcloud CLI interfaces with Google Cloud's resource-oriented API. Its internals involve project-based resource scoping, IAM condition evaluation, GKE cluster sizing, and the unique per-second billing model that changes cost optimization math.*

---

## 1. Resource Hierarchy (Organization Model)

### The Problem

GCP organizes resources in a strict hierarchy: Organization > Folder > Project > Resource. IAM policies inherit downward.

### Hierarchy Levels

$$\text{Org} \rightarrow \text{Folder}^* \rightarrow \text{Project} \rightarrow \text{Resource}$$

Where $\text{Folder}^*$ = zero or more nested folder levels (max depth: 10).

### Effective IAM Policy

$$\text{Effective}(r) = \text{Policy}(r) \cup \text{Policy}(\text{parent}(r)) \cup \cdots \cup \text{Policy}(\text{org})$$

IAM is **additive only** — you cannot remove inherited permissions at a lower level (unlike Azure deny assignments):

$$\text{Permissions}(r) = \bigcup_{s \in \text{ancestors}(r) \cup \{r\}} \text{Granted}(s)$$

### Project Quotas

| Limit | Value |
|:---|:---:|
| Projects per org | 50,000+ (adjustable) |
| Resources per project | Service-dependent |
| IAM bindings per policy | 1,500 |
| Folders per org | 300 (adjustable) |

---

## 2. Per-Second Billing (Cost Model)

### The Problem

GCP bills Compute Engine per second (minimum 1 minute). This changes the economics of short-lived workloads.

### Cost Formula

$$C = \max(60, T_{seconds}) \times R_{per\_second}$$

$$R_{per\_second} = \frac{R_{hourly}}{3600}$$

### Comparison: Per-Second vs Per-Hour

| VM Runtime | GCP (per-second) | AWS (per-second) | Old AWS (per-hour) |
|:---|:---:|:---:|:---:|
| 30 seconds | 1 min billed | 1 min billed | 1 hour billed |
| 5 minutes | 5 min billed | 5 min billed | 1 hour billed |
| 61 minutes | 61 min billed | 61 min billed | 2 hours billed |

### Sustained Use Discounts (Automatic)

GCP automatically applies discounts based on monthly usage:

$$\text{Discount}(u) = \begin{cases}
0\% & u \leq 25\% \\
20\% & 25\% < u \leq 50\% \\
40\% & 50\% < u \leq 75\% \\
60\% & 75\% < u \leq 100\%
\end{cases}$$

Where $u$ = fraction of month the instance was running.

### Effective Monthly Rate

$$C_{monthly} = \sum_{t=1}^{4} \min(0.25, u_t) \times R_{hourly} \times 730 \times (1 - d_t)$$

For a VM running all month:

$$C = 0.25 \times R \times 730 \times 1.0 + 0.25 \times R \times 730 \times 0.8 + 0.25 \times R \times 730 \times 0.6 + 0.25 \times R \times 730 \times 0.4$$

$$C = R \times 730 \times (0.25 + 0.20 + 0.15 + 0.10) = R \times 730 \times 0.70$$

**Effective discount: 30%** for always-on workloads.

---

## 3. IAM Conditions (CEL Expressions)

### The Problem

GCP IAM supports conditions using Common Expression Language (CEL). Conditions are boolean expressions evaluated at request time.

### Condition Evaluation

$$\text{Granted} = \text{role\_binding} \wedge \text{condition}(\text{request\_context})$$

### CEL Expressions

| Condition | CEL Expression |
|:---|:---|
| Time-limited | `request.time < timestamp("2026-06-01T00:00:00Z")` |
| Resource name | `resource.name.startsWith("projects/myproj/zones/us-central1")` |
| Resource type | `resource.type == "compute.googleapis.com/Instance"` |

### Condition Complexity

$$T_{eval} = O(\text{expression depth}) \approx O(1) \text{ for typical conditions}$$

Conditions add negligible latency to IAM evaluation (evaluated server-side).

---

## 4. API Rate Limits (Quota Model)

### The Problem

GCP uses a quota system with per-minute and per-day limits.

### Quota Formula

$$\text{Remaining} = \text{Limit} - \text{Used}_{window}$$

| API | Rate Limit | Window |
|:---|:---:|:---:|
| Compute Engine (read) | 20 req/s | Per second |
| Compute Engine (mutate) | 20 req/s | Per second |
| Cloud Storage (JSON API) | 50,000 req/s/project | Per second |
| GKE | 600 req/min | Per minute |
| IAM | 600 req/min | Per minute |

### Batch API Optimization

$$\text{Requests}_{batch} = \lceil N / B_{size} \rceil$$

Where $B_{size}$ = batch size (max 1,000 for many APIs).

For 5,000 operations: $\lceil 5000/1000 \rceil = 5$ batch requests vs 5,000 individual.

---

## 5. GKE Cluster Sizing (Node Pool Math)

### The Problem

GKE clusters size node pools to fit workloads. Understanding the allocation math prevents resource waste.

### Allocatable Resources

$$\text{Allocatable} = \text{Capacity} - \text{System Reserved} - \text{Eviction Threshold}$$

### GKE System Reserved (Memory)

| Total Memory | Reserved |
|:---|:---|
| First 4 GB | 25% = 1 GB |
| Next 4 GB | 20% = 0.8 GB |
| Next 8 GB | 10% = 0.8 GB |
| Next 112 GB | 6% |
| Over 128 GB | 2% |

**Worked Example: 16 GB node:**

$$\text{Reserved} = 1.0 + 0.8 + 0.8 = 2.6 \text{ GB}$$
$$\text{Eviction threshold} = 100 \text{ MB}$$
$$\text{Allocatable} = 16 - 2.6 - 0.1 = 13.3 \text{ GB}$$
$$\text{Efficiency} = \frac{13.3}{16} = 83.1\%$$

### GKE System Reserved (CPU)

| Total Cores | Reserved |
|:---|:---|
| First core | 6% |
| Next core | 1% |
| Next 2 cores | 0.5% |
| Over 4 cores | 0.25% |

**Worked Example: 8-core node:**

$$\text{Reserved} = 60 + 10 + 10 + 10 = 90\text{m} \quad (90 \text{ millicores})$$
$$\text{Allocatable} = 8000 - 90 = 7910\text{m}$$

### Autoscaler Formula

$$\text{Desired nodes} = \lceil \frac{\sum \text{pod requests}}{\text{Allocatable per node}} \rceil$$

---

## 6. Cloud Storage (GCS) Performance Model

### The Problem

GCS performance depends on object size, location, and access patterns.

### Throughput

$$\text{Throughput} = \min(BW_{network}, N_{parallel} \times BW_{per\_stream})$$

Single stream max: ~1.2 Gbps to a single GCS bucket.

### gsutil Parallel Upload

$$T_{upload} = \frac{S_{total}}{N_{threads} \times BW_{per\_thread}}$$

Default threads: 4 for `gsutil -m`.

### Composite Upload (Large Files)

$$S_{chunk} = S_{file} / N_{components}$$
$$T_{composite} = \max_{i} \frac{S_{chunk}}{BW_i} + T_{compose}$$

Max components: 32.

### Storage Class Cost Optimization

$$C_{optimal}(class) = \arg\min_{c} (S \times R_{storage}(c) + N_{ops} \times R_{ops}(c) + S_{retrieval} \times R_{retrieval}(c))$$

| Class | Storage $/GB/mo | Retrieval $/GB | Min Duration |
|:---|:---:|:---:|:---:|
| Standard | $0.020 | $0.00 | None |
| Nearline | $0.010 | $0.01 | 30 days |
| Coldline | $0.004 | $0.02 | 90 days |
| Archive | $0.0012 | $0.05 | 365 days |

### Early Deletion Penalty

$$C_{penalty} = R_{storage} \times (D_{min} - D_{actual})$$

Deleting a Coldline object after 30 days: charged for remaining 60 days.

---

## 7. Network Pricing (Egress Model)

### The Problem

GCP charges for egress traffic with a tiered model. Inter-zone, inter-region, and internet egress all differ.

### Egress Pricing

| Path | Cost/GB |
|:---|:---:|
| Same zone | Free |
| Cross-zone (same region) | $0.01 |
| Cross-region (US) | $0.01 |
| Cross-region (intercontinental) | $0.02-0.08 |
| Internet egress (first 1 TB) | $0.12 |
| Internet egress (1-10 TB) | $0.11 |
| Internet egress (10+ TB) | $0.08 |

### Multi-Region Application Cost

For an app with 3 regions, 100 GB cross-region traffic each:

$$C_{network} = 3 \times 100 \times 0.01 = \$3.00/\text{month}$$

Adding 1 TB internet egress:

$$C_{egress} = 1000 \times 0.12 = \$120.00/\text{month}$$

**Egress dominates** — 40x the cross-region cost.

---

## 8. Summary of Functions by Type

| Formula | Math Type | Domain |
|:---|:---|:---|
| $\bigcup \text{Granted}(ancestors)$ | Set union (additive only) | IAM inheritance |
| $\max(60, T) \times R$ | Floor + linear | Per-second billing |
| Tiered sustained discount | Piecewise | Automatic discounts |
| $\text{Cap} - \text{Reserved} - \text{Eviction}$ | Subtraction | Node allocatable |
| $\arg\min_c C(class)$ | Optimization | Storage class |
| $\lceil N/B \rceil$ | Ceiling division | Batch API |

---

*Every `gcloud` command hits a project-scoped, quota-limited API endpoint. GCP's per-second billing and automatic sustained-use discounts reward infrastructure that runs continuously, while the additive-only IAM model keeps permissions simple at the cost of less granular denial.*
