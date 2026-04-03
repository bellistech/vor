# The Mathematics of GitLab CI — Pipeline Execution Theory

> *GitLab CI uses a stage-based pipeline model with DAG extensions, parallel job splitting, and a sophisticated runner autoscaling system. Its execution model involves stage synchronization barriers, merge train queuing, and coverage metrics.*

---

## 1. Stage-Based Execution Model

### The Problem

GitLab CI organizes jobs into stages. All jobs in a stage run in parallel; stages run sequentially.

### Stage Barrier Model

$$T_{pipeline} = \sum_{s=1}^{S} \max_{j \in \text{stage}(s)} T_j$$

Each stage is a **synchronization barrier** — the slowest job in each stage determines stage duration.

### Worked Example

| Stage | Jobs | Job Times | Stage Time |
|:---|:---|:---|:---:|
| build | build-linux, build-macos | 3 min, 8 min | 8 min |
| test | unit, integration, lint | 5 min, 12 min, 2 min | 12 min |
| deploy | deploy-staging | 3 min | 3 min |
| **Total** | | | **23 min** |

### DAG Override (`needs` keyword)

With `needs`, jobs can skip the stage barrier:

$$T_{job} = T_{start\_after\_needs} = \max_{n \in \text{needs}(j)} T_{complete}(n) + T_j$$

$$T_{pipeline}^{DAG} \leq T_{pipeline}^{stages}$$

**Example:** If `deploy` only needs `build-linux` (3 min), not the full test stage:

$$T_{pipeline}^{DAG} = \max(3 + 3, 8 + 12) = \max(6, 20) = 20 \text{ min}$$
$$\text{Savings} = 23 - 20 = 3 \text{ min}$$

---

## 2. Parallel Job Splitting

### The Problem

The `parallel` keyword splits a single job into $N$ parallel instances. Combined with test splitting, this dramatically reduces test time.

### Parallel Split Formula

$$T_{parallel} = \frac{T_{serial}}{N} + T_{overhead}$$

$$\text{Speedup} = \frac{T_{serial}}{T_{parallel}} = \frac{T_{serial}}{T_{serial}/N + T_{overhead}}$$

### Amdahl's Law Applied

If fraction $p$ of work is parallelizable:

$$\text{Speedup}(N) = \frac{1}{(1-p) + p/N}$$

For test suites, $p \approx 0.95$ (5% setup overhead):

| Parallel | Theoretical Speedup | With 30s Overhead | Actual Speedup |
|:---:|:---:|:---:|:---:|
| 2 | 1.90x | On 10-min suite | 1.85x |
| 4 | 3.48x | | 3.16x |
| 8 | 5.93x | | 4.83x |
| 16 | 8.77x | | 6.21x |

### Parallel Matrix

$$\text{parallel: matrix:}$$

$$|J| = \prod_{i} |A_i|$$

Same Cartesian product as GitHub Actions.

---

## 3. Runner Autoscaling (Fleet Sizing)

### The Problem

GitLab Runner with Docker Machine autoscaler scales the runner fleet based on demand. The scaling math determines cost and queue wait time.

### Scaling Parameters

$$\text{IdleCount} = R_{idle} \quad \text{(pre-warmed runners)}$$
$$\text{IdleTime} = T_{idle} \quad \text{(seconds before scale-down)}$$
$$\text{MaxInstances} = R_{max}$$

### Queue Wait Time

$$T_{queue} = \begin{cases}
0 & \text{if } R_{idle} > 0 \\
T_{provision} & \text{if } R_{idle} = 0 \text{ and } R_{current} < R_{max} \\
T_{wait\_for\_free} & \text{if } R_{current} = R_{max}
\end{cases}$$

Where $T_{provision} \approx 30\text{-}120$ seconds for cloud VMs.

### Cost Model

$$C_{runners} = R_{idle} \times T_{total} \times R_{hourly} + \sum_{\text{jobs}} T_j \times R_{hourly}$$

The first term is the **idle cost** (paying for pre-warmed capacity).

### Optimal Idle Count

$$R_{idle}^{opt} = \lceil \lambda \times T_{provision} \rceil$$

Where $\lambda$ = average job arrival rate. If jobs arrive at 2/min and provisioning takes 1 min:

$$R_{idle}^{opt} = \lceil 2 \times 1 \rceil = 2$$

---

## 4. Merge Trains (Queue Theory)

### The Problem

Merge trains serialize merge requests to ensure each MR is tested on top of all previously queued MRs. This prevents "broken main" but adds latency.

### Merge Train Pipeline

For MR at position $k$ in the train:

$$\text{Base}(MR_k) = \text{main} + MR_1 + MR_2 + \cdots + MR_{k-1}$$

### Pipeline Time

Without merge trains:

$$T_{total} = \max_k T_k \quad \text{(all test in parallel)}$$

With merge trains (sequential):

$$T_{total} = \sum_{k=1}^{N} T_k$$

With merge trains (parallel pipelines):

$$T_{total} = T_1 + (N-1) \times 0 \quad \text{(if all start simultaneously)}$$

But if $MR_j$ fails, all subsequent MRs must restart:

$$T_{restart} = T_{pipeline} \times \text{expected restarts}$$

### Failure Impact

If each MR has probability $p$ of pipeline failure:

$$P(\text{no failure in train of } N) = (1-p)^N$$

$$E[\text{restarts}] = \sum_{k=1}^{N} p \times (N - k)$$

For $N=5$, $p=0.1$:

$$P(\text{clean run}) = 0.9^5 = 0.590$$

41% chance at least one restart is needed.

---

## 5. Coverage Tracking (Statistical Metrics)

### The Problem

GitLab tracks code coverage percentage from CI output. Understanding the metric helps set meaningful thresholds.

### Coverage Formula

$$\text{Coverage} = \frac{L_{covered}}{L_{total}} \times 100\%$$

### Coverage Delta (MR metric)

$$\Delta C = C_{MR} - C_{base}$$

### The Diminishing Returns Problem

Going from 80% to 90% coverage requires covering 50% of remaining uncovered lines:

$$\text{New lines to cover} = L_{total} \times (C_{target} - C_{current})$$

| Current | Target | Lines to Cover (1000 total) | Effort Multiple |
|:---:|:---:|:---:|:---:|
| 0% → 50% | 50% | 500 | 1x (baseline) |
| 50% → 75% | 75% | 250 | 1x |
| 75% → 90% | 90% | 150 | 1.2x (harder code) |
| 90% → 95% | 95% | 50 | 2-3x (edge cases) |
| 95% → 99% | 99% | 40 | 5-10x (error paths) |

### Coverage as Quality Gate

$$\text{Pipeline passes} \iff C_{MR} \geq C_{threshold} \wedge \Delta C \geq \Delta_{min}$$

Typical thresholds: $C_{threshold} = 80\%$, $\Delta_{min} = -1\%$ (no more than 1% regression).

---

## 6. Artifact and Cache Management

### The Problem

GitLab CI differentiates artifacts (passed between stages) and cache (persisted across pipelines).

### Artifact Transfer Cost

$$T_{pipeline\_overhead} = \sum_{\text{stages}} (T_{upload} + T_{download})$$

$$T_{transfer} = \frac{S_{artifact}}{BW}$$

### Cache Key Strategy

$$\text{hit} = \begin{cases}
\text{exact} & \text{if } \text{key} \in \text{cache\_store} \\
\text{partial} & \text{if } \text{key\_prefix} \text{ matches} \\
\text{miss} & \text{otherwise}
\end{cases}$$

### Distributed Cache (S3 Backend)

$$T_{cache} = T_{S3\_download} + T_{extract} = \frac{S_{cache}}{BW_{S3}} + \frac{S_{cache}}{BW_{disk}}$$

| Cache Size | S3 Download | Extract | Total |
|:---:|:---:|:---:|:---:|
| 50 MB | 0.5s | 0.2s | 0.7s |
| 500 MB | 5s | 2s | 7s |
| 2 GB | 20s | 8s | 28s |

---

## 7. Rules Evaluation (Boolean Logic)

### The Problem

GitLab CI `rules` determine whether a job runs. Rules are evaluated as an ordered list of conditions.

### Evaluation Model

$$\text{action}(job) = \text{first matching rule's action (include or exclude)}$$

### Rule Conditions

$$\text{rule matches} = \text{if\_condition} \wedge \text{changes\_condition} \wedge \text{exists\_condition}$$

### `changes` Path Matching

$$\text{changes\_match} = \exists f \in \text{changed\_files}: f \sim \text{glob\_pattern}$$

### Pipeline Filtering Efficiency

$$\text{Jobs skipped} = |J_{total}| - |J_{matched}|$$

$$\text{Time saved} = \sum_{j \in J_{skipped}} T_j$$

For a monorepo with 20 services, changing 1 service should run ~5% of jobs:

$$\text{Savings} = 1 - 0.05 = 95\%$$

---

## 8. Summary of Functions by Type

| Formula | Math Type | Domain |
|:---|:---|:---|
| $\sum \max T_j$ per stage | Sum of maxima | Stage execution |
| $\prod |A_i|$ | Cartesian product | Matrix/parallel |
| $1/((1-p) + p/N)$ | Amdahl's law | Parallel speedup |
| $(1-p)^N$ | Probability | Merge train reliability |
| $L_{covered}/L_{total}$ | Ratio | Coverage |
| $\lceil \lambda \times T_{provision} \rceil$ | Queueing theory | Runner autoscaling |

---

*GitLab CI's stage model is a synchronization barrier, its parallel keyword is Amdahl's law in action, and merge trains are a queueing theory problem. Understanding these formulas lets you design pipelines that are fast, reliable, and cost-effective.*
