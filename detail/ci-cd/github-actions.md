# The Mathematics of GitHub Actions — CI/CD Pipeline Theory

> *GitHub Actions executes workflows as directed acyclic graphs of jobs with matrix expansion, conditional evaluation, and concurrency controls. Its execution model involves Cartesian product matrix strategies, artifact transfer costs, and runner pool scheduling.*

---

## 1. Matrix Strategy (Cartesian Product Expansion)

### The Problem

Matrix strategies generate job permutations from multiple dimensions. The total jobs are the Cartesian product of all matrix axes.

### The Formula

$$|J| = \prod_{i=1}^{D} |A_i|$$

Where $D$ = number of dimensions, $|A_i|$ = values in dimension $i$.

### Worked Examples

| Matrix Definition | Dimensions | Total Jobs |
|:---|:---:|:---:|
| `os: [ubuntu, macos]` x `node: [16, 18, 20]` | 2 x 3 | 6 |
| `os: [ubuntu, macos, windows]` x `node: [18, 20]` x `arch: [x64, arm64]` | 3 x 2 x 2 | 12 |
| `go: [1.22, 1.23, 1.24]` x `db: [postgres, mysql, sqlite]` | 3 x 3 | 9 |

### With Exclusions

$$|J_{effective}| = \prod |A_i| - |\text{exclude}| + |\text{include}|$$

**Example:** 3 OS x 3 Node = 9, exclude `{macos, 16}`, include `{ubuntu, 21-nightly}`:

$$|J| = 9 - 1 + 1 = 9$$

### Execution Time

With unlimited runners:

$$T_{matrix} = \max_{j \in J} T_j$$

With $R$ available runners:

$$T_{matrix} = \lceil |J| / R \rceil \times \max_{batch} T_{batch}$$

### Cost Calculation

$$C_{matrix} = \sum_{j \in J} T_j \times R_{per\_minute}(os_j)$$

| Runner OS | Rate/Minute |
|:---|:---:|
| Linux | $0.008 |
| Windows | $0.016 |
| macOS | $0.08 |

**6-job matrix (2 Linux, 2 Windows, 2 macOS) each 10 min:**

$$C = 2 \times 10 \times 0.008 + 2 \times 10 \times 0.016 + 2 \times 10 \times 0.08 = 0.16 + 0.32 + 1.60 = \$2.08$$

macOS is $10\times$ Linux cost — this dominates.

---

## 2. Job Dependency Graph (DAG Execution)

### The Problem

Jobs with `needs` form a DAG. Execution order and total pipeline time depend on the critical path.

### Critical Path

$$T_{pipeline} = \max_{\text{paths}} \sum_{j \in \text{path}} T_j$$

The critical path is the longest path through the DAG — no amount of parallelism can reduce it below this.

### Worked Example

```yaml
jobs:
  lint:      # 2 min
  test:      # 8 min, needs: lint
  build:     # 5 min, needs: lint
  deploy:    # 3 min, needs: [test, build]
```

| Path | Total Time |
|:---|:---:|
| lint → test → deploy | 2 + 8 + 3 = 13 min |
| lint → build → deploy | 2 + 5 + 3 = 10 min |

$$T_{pipeline} = 13 \text{ min (critical path)}$$

### Parallelism Savings

$$\text{Sequential}: T = 2 + 8 + 5 + 3 = 18 \text{ min}$$
$$\text{Parallel}: T = 13 \text{ min}$$
$$\text{Savings} = 1 - 13/18 = 27.8\%$$

---

## 3. Concurrency Controls (Queue Theory)

### The Problem

The `concurrency` key prevents multiple workflow runs from executing simultaneously. This is a mutex/semaphore model.

### Concurrency Group

$$\text{group}(run) = \text{template string with context variables}$$

Runs in the same group are serialized:

$$\text{At any time}: |\{r \in \text{group} : r.\text{status} = \text{running}\}| \leq 1$$

### With `cancel-in-progress: true`

$$\text{New run arrives} \implies \text{cancel running run in same group}$$

This is a "last-write-wins" model — only the most recent run completes.

### Queue Depth

Without cancellation, pending runs queue up:

$$\text{Queue depth} = N_{pushes\_during\_run} = \frac{T_{run}}{T_{push\_interval}}$$

For a 10-minute run with pushes every 2 minutes:

$$\text{Queue depth} = 10 / 2 = 5 \text{ pending runs}$$

With `cancel-in-progress`, queue depth is always 0 or 1.

---

## 4. Artifact Transfer (Upload/Download Cost)

### The Problem

Artifacts transferred between jobs consume storage and bandwidth.

### Transfer Time

$$T_{artifact} = T_{compress} + \frac{S_{compressed}}{BW_{upload}} + \frac{S_{compressed}}{BW_{download}}$$

### Storage Limits

| Plan | Storage | Retention |
|:---|:---:|:---:|
| Free | 500 MB | 90 days |
| Pro | 1 GB | 90 days |
| Enterprise | 50 GB | 90 days |

### Artifact Size Impact

$$T_{job\_gap} = T_{upload}(producer) + T_{scheduling} + T_{download}(consumer)$$

| Artifact Size | Upload | Download | Gap Overhead |
|:---:|:---:|:---:|:---:|
| 10 MB | 1s | 1s | ~30s (scheduling dominates) |
| 100 MB | 5s | 5s | ~40s |
| 1 GB | 30s | 30s | ~90s |

### Optimization: Minimize Inter-Job Artifacts

$$T_{optimized} < T_{artifact} \iff \text{combine jobs when artifact transfer > time saved by parallelism}$$

---

## 5. Caching (Hit Rate and Savings)

### The Problem

Action caches (`actions/cache`) store and restore build dependencies. Cache hit rate determines pipeline speedup.

### Cache Hit Savings

$$T_{with\_cache} = T_{restore} + T_{build\_incremental}$$
$$T_{without\_cache} = T_{build\_full}$$
$$\text{Speedup} = \frac{T_{without}}{T_{with}}$$

### Worked Examples

| Language | Full Build | Cached Build | Speedup |
|:---|:---:|:---:|:---:|
| Node.js (npm install) | 60s | 5s (restore) + 2s (verify) = 7s | 8.6x |
| Go (mod download) | 30s | 3s + 0s = 3s | 10x |
| Rust (cargo build) | 300s | 5s + 45s = 50s | 6x |
| Python (pip install) | 45s | 3s + 1s = 4s | 11.3x |

### Cache Key Strategy

$$\text{hit} = \begin{cases}
\text{exact} & \text{if } \text{key} \in \text{cache} \\
\text{partial} & \text{if } \exists k \in \text{restore-keys}: k \text{ prefix matches} \\
\text{miss} & \text{otherwise}
\end{cases}$$

### Cache Eviction

$$\text{Eviction}: \text{LRU when total > 10 GB per repo}$$

---

## 6. Workflow Billing (Minute Calculation)

### The Problem

GitHub Actions bills by the minute with OS multipliers and rounding.

### Billing Formula

$$C_{workflow} = \sum_{j \in \text{jobs}} \lceil T_j \rceil \times M_{os}$$

Where $\lceil T_j \rceil$ = job time rounded up to the next minute, $M_{os}$ = OS multiplier.

| OS | Multiplier |
|:---|:---:|
| Linux | 1x |
| macOS | 10x |
| Windows | 2x |

### Monthly Budget Calculation

Free tier: 2,000 minutes (Linux equivalent).

$$\text{Budget}_{remaining} = 2000 - \sum_{workflows} \sum_{jobs} \lceil T_j \rceil \times M_{os}$$

**Worked Example:** 100 workflow runs/month, each with 3 Linux jobs (5 min) and 1 macOS job (5 min):

$$C = 100 \times (3 \times 5 \times 1 + 1 \times 5 \times 10) = 100 \times 65 = 6{,}500 \text{ minutes}$$

$$\text{Overage} = 6{,}500 - 2{,}000 = 4{,}500 \text{ minutes at } \$0.008 = \$36$$

---

## 7. Reusable Workflows (Composition)

### The Problem

Reusable workflows (`workflow_call`) can be nested up to 4 levels deep and called from matrix jobs.

### Nesting Depth

$$\text{Max depth} = 4 \quad \text{(caller → reusable → reusable → reusable)}$$

### Fan-Out with Matrix + Reusable

$$|J_{total}| = |J_{caller\_matrix}| \times |J_{reusable\_matrix}|$$

A caller matrix of 3 x reusable matrix of 4 = 12 total jobs from a single workflow file.

### Job Limit

$$|J_{total}| \leq 256 \quad \text{per workflow run}$$

---

## 8. Summary of Functions by Type

| Formula | Math Type | Domain |
|:---|:---|:---|
| $\prod |A_i|$ | Cartesian product | Matrix expansion |
| $\max \sum T_j$ over paths | Critical path | Pipeline duration |
| $\lceil T \rceil \times M_{os}$ | Rounding + multiplier | Billing |
| $T_{full} / T_{cached}$ | Ratio | Cache speedup |
| Queue depth = $T_{run}/T_{push}$ | Rate division | Concurrency |
| $|J_{caller}| \times |J_{reusable}|$ | Multiplication | Workflow composition |

---

*GitHub Actions turns YAML into a distributed job scheduler — matrix products generate the test matrix, DAG analysis determines the critical path, and OS multipliers determine the bill. Understanding these formulas is the difference between a 5-minute pipeline and a 50-minute one.*

## Prerequisites

- Git fundamentals (branches, tags, pull requests)
- YAML syntax
- Basic CI/CD concepts (build, test, deploy)
- GitHub repository structure and permissions

## Complexity

- Beginner: simple workflows with checkout and run steps, trigger events
- Intermediate: matrix builds, artifacts, secrets, environments, reusable workflows
- Advanced: Cartesian product optimization, critical path analysis, composite actions, self-hosted runners, cost modeling
