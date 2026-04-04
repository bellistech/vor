# The Mathematics of Jenkins — CI/CD Pipeline Theory

> *Jenkins is a distributed build system with a controller-agent architecture. Its internals involve executor pool scheduling, pipeline DAG execution, distributed workspace management, and queue theory for job prioritization.*

---

## 1. Executor Pool Model (Controller-Agent Architecture)

### The Problem

Jenkins distributes jobs across agents with a fixed number of executors. Understanding the pool math is essential for sizing.

### Executor Capacity

$$E_{total} = E_{controller} + \sum_{a \in \text{agents}} E_a$$

### Utilization

$$U = \frac{E_{busy}}{E_{total}}$$

### Queue Wait Time (Little's Law)

$$L = \lambda \times W$$

Where:
- $L$ = average queue length
- $\lambda$ = job arrival rate (jobs/min)
- $W$ = average wait time

Rearranging:

$$W = \frac{L}{\lambda}$$

### Worked Example

- 10 executors, average job duration 5 min
- Job arrival rate: 3 jobs/min
- System throughput capacity: $10 / 5 = 2$ jobs/min

Since $\lambda = 3 > \mu = 2$, the queue grows without bound. **Undersized.**

With 20 executors: $\mu = 20/5 = 4 > 3 = \lambda$. Queue is stable:

$$W_{avg} = \frac{1}{\mu - \lambda} = \frac{1}{4 - 3} = 1 \text{ min}$$

### M/M/c Queue Model

For $c$ executors, arrival rate $\lambda$, service rate $\mu$ per executor:

$$\rho = \frac{\lambda}{c \times \mu}$$

$$P(\text{wait}) = \frac{(c\rho)^c}{c!(1-\rho)} \times \left[\sum_{k=0}^{c-1}\frac{(c\rho)^k}{k!} + \frac{(c\rho)^c}{c!(1-\rho)}\right]^{-1}$$

| Executors | $\rho$ (utilization) | Avg Wait |
|:---:|:---:|:---:|
| 4 ($\lambda=3, \mu=1$) | 0.75 | 2.2 min |
| 6 | 0.50 | 0.3 min |
| 8 | 0.375 | 0.1 min |
| 10 | 0.30 | 0.03 min |

---

## 2. Pipeline DAG (Declarative and Scripted)

### The Problem

Jenkins Pipeline stages can run in parallel, forming a DAG. The critical path determines minimum pipeline time.

### Stage Model

```groovy
stage('Build') { ... }          // 3 min
stage('Test') {
    parallel {
        stage('Unit') { ... }    // 8 min
        stage('Integration') { ... }  // 12 min
        stage('Lint') { ... }    // 2 min
    }
}
stage('Deploy') { ... }         // 5 min
```

$$T_{pipeline} = T_{build} + \max(T_{unit}, T_{integration}, T_{lint}) + T_{deploy}$$

$$T = 3 + 12 + 5 = 20 \text{ min}$$

### Sequential vs Parallel Comparison

$$T_{sequential} = 3 + 8 + 12 + 2 + 5 = 30 \text{ min}$$
$$T_{parallel} = 3 + 12 + 5 = 20 \text{ min}$$
$$\text{Savings} = \frac{30 - 20}{30} = 33.3\%$$

### Nested Parallelism

$$T_{nested} = \sum_{\text{sequential stages}} \max_{\text{parallel branches}} T_{branch}$$

Maximum parallelism:

$$P_{max} = \max_{\text{stage}} |\text{parallel branches in stage}|$$

---

## 3. Distributed Build (Workspace Management)

### The Problem

Each build occupies a workspace on the agent. Workspace management affects disk usage and build isolation.

### Workspace Size

$$S_{workspace} = S_{checkout} + S_{build\_artifacts} + S_{dependencies}$$

### Disk Pressure Formula

$$S_{agent\_used} = \sum_{e=1}^{E} S_{workspace_e} + S_{tool\_cache}$$

$$T_{until\_full} = \frac{S_{disk} - S_{used}}{R_{growth}}$$

### Workspace Cleanup Strategies

| Strategy | Disk Usage | Build Time | Safety |
|:---|:---:|:---:|:---:|
| Clean before build | $1 \times S_{workspace}$ per executor | Slow (full clone) | High |
| Incremental | $1 \times S_{workspace}$ + history | Fast (git pull) | Medium |
| Per-build directory | $N \times S_{workspace}$ | Fast | High |

### Checkout Time

$$T_{checkout} = \begin{cases}
\frac{S_{repo}}{BW} & \text{full clone} \\
\frac{S_{delta}}{BW} & \text{incremental (git fetch)} \\
\frac{S_{shallow}}{BW} & \text{shallow clone (depth=1)}
\end{cases}$$

| Repo Size | Full Clone | Shallow Clone | Speedup |
|:---:|:---:|:---:|:---:|
| 100 MB | 10s | 2s | 5x |
| 1 GB | 100s | 5s | 20x |
| 10 GB | 1000s | 10s | 100x |

---

## 4. Plugin Ecosystem (Dependency Graph)

### The Problem

Jenkins has 1,800+ plugins with complex interdependencies. Plugin updates can cascade failures.

### Dependency Depth

$$\text{Total deps}(p) = |\text{transitive\_closure}(\text{deps}(p))|$$

Typical plugin dependency depths:

| Plugin | Direct Deps | Transitive Deps |
|:---|:---:|:---:|
| Git | 5 | 15 |
| Pipeline | 8 | 30 |
| Blue Ocean | 12 | 60+ |
| Docker | 4 | 20 |

### Startup Time Impact

$$T_{startup} = T_{core} + \sum_{p \in \text{plugins}} T_{init}(p)$$

| Plugins Installed | Typical Startup | Memory Impact |
|:---:|:---:|:---:|
| 20 | 30s | +200 MB |
| 50 | 60s | +500 MB |
| 100 | 120s | +1 GB |
| 200 | 300s | +2 GB |

### Update Risk

$$P(\text{breakage}) = 1 - \prod_{p \in \text{updated}} (1 - p_{break}(p))$$

For 10 plugin updates, each with 2% break probability:

$$P = 1 - 0.98^{10} = 1 - 0.817 = 18.3\%$$

---

## 5. Build Queue Priority (Scheduling Algorithm)

### The Problem

When multiple jobs compete for executors, Jenkins uses a priority system.

### Default Priority

$$\text{priority}(j) = T_{queued}(j) \quad \text{(FIFO — longer wait = higher priority)}$$

### With Priority Sorter Plugin

$$\text{priority}(j) = w_{group} \times P_{group}(j) + w_{job} \times P_{job}(j) + w_{wait} \times T_{queued}(j)$$

### Label Matching

Jobs request specific agent labels:

$$\text{Eligible}(j) = \{a : \text{labels}(j) \subseteq \text{labels}(a)\}$$

$$\text{Schedulable}(j) = \{a \in \text{Eligible}(j) : E_{free}(a) > 0\}$$

### Starvation Prevention

Without priority aging, low-priority jobs starve:

$$\text{priority}(j, t) = P_{base}(j) + \alpha \times (t - T_{queued}(j))$$

Where $\alpha$ = aging factor, increasing priority linearly with wait time.

---

## 6. Distributed Builds (Network Overhead)

### The Problem

Controller-agent communication adds overhead to every build step.

### JNLP/Remoting Protocol

$$T_{step} = T_{dispatch} + T_{execute} + T_{return}$$

Where:
- $T_{dispatch}$ = serialize + send command (~5-20ms)
- $T_{execute}$ = actual work
- $T_{return}$ = serialize + send result (~5-20ms)

### Overhead per Step

$$O_{step} = T_{dispatch} + T_{return} \approx 10\text{-}40 \text{ ms}$$

For a pipeline with 200 steps:

$$O_{total} = 200 \times 30 = 6{,}000 \text{ ms} = 6\text{s}$$

### Agent Connectivity

| Connection Type | Latency | Suitable For |
|:---|:---:|:---|
| LAN (<1ms) | 10-20 ms/step | Standard |
| VPN (10-50ms) | 30-100 ms/step | Acceptable |
| WAN (100ms+) | 120-240 ms/step | Problematic |

For WAN at 200ms round trip, 200 steps:

$$O_{total} = 200 \times 400 = 80{,}000 \text{ ms} = 80\text{s}$$

---

## 7. Jenkinsfile Shared Libraries (Code Reuse)

### The Problem

Shared libraries reduce duplication across pipelines. The loading cost and cache behavior affect build time.

### Library Loading Time

$$T_{library} = T_{git\_fetch} + T_{compile\_groovy}$$

First build: $T \approx 5\text{-}30$s (clone + compile).
Subsequent: $T \approx 1\text{-}5$s (cached).

### Code Reuse Metric

$$\text{Reuse ratio} = \frac{L_{shared}}{L_{shared} + L_{per\_pipeline}}$$

| Org Size | Pipelines | Shared Library Lines | Per-Pipeline Lines | Reuse |
|:---:|:---:|:---:|:---:|:---:|
| Small | 10 | 500 | 200 each | 20% |
| Medium | 50 | 2,000 | 100 each | 29% |
| Large | 200 | 5,000 | 50 each | 33% |

---

## 8. Summary of Functions by Type

| Formula | Math Type | Domain |
|:---|:---|:---|
| $L = \lambda \times W$ | Little's law | Queue sizing |
| $\lambda / (c \times \mu)$ | M/M/c queue | Executor utilization |
| $\sum \max T_{parallel}$ | Critical path | Pipeline duration |
| $1 - \prod(1-p)$ | Probability | Plugin update risk |
| $P_{base} + \alpha \times \Delta t$ | Linear aging | Priority scheduling |
| $S_{repo}/BW$ vs $S_{shallow}/BW$ | Ratio | Checkout optimization |

---

*Jenkins is a distributed job scheduler with 20+ years of evolution. Its controller-agent model is a classic M/M/c queue, its pipelines are DAGs, and its plugin ecosystem is a dependency management problem. The math hasn't changed — just the scale.*

## Prerequisites

- Groovy syntax fundamentals (for Jenkinsfile)
- CI/CD concepts (build, test, deploy pipelines)
- Java runtime environment (JRE) basics
- SCM fundamentals (Git integration)

## Complexity

- Beginner: declarative pipelines, basic stages, credentials
- Intermediate: parallel stages, shared libraries, Docker agents, parameters
- Advanced: executor pool sizing (queueing theory), scripted pipelines, distributed workspaces, plugin dependency management
