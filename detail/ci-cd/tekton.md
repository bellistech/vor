# The Mathematics of Tekton — DAG Scheduling and Pipeline Optimization

> *Tekton pipelines are directed acyclic graphs (DAGs) where tasks are nodes and dependencies are edges. Pipeline execution is a topological sort problem, workspace sharing is a resource allocation problem, and trigger processing follows event-driven queueing models.*

---

## 1. Pipeline DAG Scheduling (Graph Theory)

### Pipeline as DAG

A Tekton Pipeline is a DAG $G = (V, E)$:

- $V = \{t_1, t_2, \ldots, t_n\}$ = set of Tasks
- $E = \{(t_i, t_j) \mid t_j.\text{runAfter includes } t_i\}$

### Topological Sort

Execution order is determined by topological sort:

$$\text{order} = \text{toposort}(G) = [v_1, v_2, \ldots, v_n] \text{ where } (v_i, v_j) \in E \implies i < j$$

### Critical Path

The minimum pipeline execution time:

$$T_{pipeline} = \max_{\text{path } P \in G} \sum_{t \in P} T_t$$

This is the longest path in the weighted DAG (critical path method).

### Worked Example

```
clone (30s) --> test (120s) --> build-image (60s) --> deploy (20s)
           \-> version (5s) -/
```

| Path | Duration |
|:---|:---:|
| clone -> test -> build-image -> deploy | 230s |
| clone -> version -> build-image -> deploy | 115s |

$$T_{critical} = 230\text{s}$$

### Parallelism Degree

Maximum parallel tasks at any time:

$$P_{max} = \max_{t} |\{v \in V : \text{start}(v) \leq t < \text{end}(v)\}|$$

The width of the DAG provides an upper bound:

$$P_{max} \leq W(G) = \text{max antichain size}$$

---

## 2. Step Execution (Sequential Pipeline)

### Within-Task Scheduling

Steps within a Task execute sequentially in a single Pod:

$$T_{task} = \sum_{i=1}^{s} T_{step_i} + \sum_{i=1}^{s-1} T_{transition_i}$$

Where $T_{transition}$ is the container start/stop overhead (~100ms).

### Step Container Overhead

Each step runs in its own container within the pod:

$$T_{overhead} = T_{pull} + T_{start} + T_{init}$$

| Component | Cold (first pull) | Warm (cached) |
|:---|:---:|:---:|
| Image pull | 5-60s | 0s |
| Container start | 0.5s | 0.5s |
| Runtime init | 0.1-1s | 0.1-1s |
| **Total** | **6-62s** | **0.6-1.5s** |

### Task Pod Efficiency

For $s$ steps sharing the same image:

$$T_{efficient} = T_{pull\_once} + s \times (T_{step} + T_{transition})$$
$$T_{naive} = s \times (T_{pull} + T_{step} + T_{transition})$$

$$\text{Speedup} = \frac{s \times T_{pull} + s \times T_{step}}{T_{pull} + s \times T_{step}}$$

---

## 3. Workspace Storage Mathematics (Resource Allocation)

### PVC Sizing

Required storage for a workspace:

$$S_{workspace} = S_{source} + S_{build\_artifacts} + S_{cache} + S_{overhead}$$

### VolumeClaimTemplate Lifecycle

With auto-provisioned PVCs:

$$\text{PVCs active} = |\text{running PipelineRuns}|$$

$$S_{total} = |\text{concurrent runs}| \times S_{workspace}$$

### Storage I/O Impact

Sequential task access to shared PVC:

$$T_{IO} = \frac{S_{read} + S_{write}}{BW_{PV}}$$

| Storage Class | Read BW | Write BW | IOPS |
|:---|:---:|:---:|:---:|
| gp3 (EBS) | 125 MB/s | 125 MB/s | 3,000 |
| pd-ssd (GCE) | 100 MB/s | 100 MB/s | 6,000 |
| premium (Azure) | 150 MB/s | 150 MB/s | 5,000 |

### EmptyDir vs PVC Trade-off

$$T_{emptyDir} = T_{local\_IO} \quad \text{(node-local, fast)}$$
$$T_{PVC} = T_{network\_IO} \quad \text{(network-attached, slower)}$$

But EmptyDir is ephemeral — data lost between Tasks.

---

## 4. Results and Data Flow (Information Passing)

### Result Size Constraints

Tekton results are stored in the termination message:

$$|result| \leq 4096 \text{ bytes (per result)}$$
$$\sum |results| \leq 12288 \text{ bytes (per task)}$$

### Result as DAG Edge Labels

Results create implicit data dependencies:

$$E_{data} = \{(t_i, t_j) \mid t_j \text{ references } t_i.\text{results}\}$$

$$E_{total} = E_{runAfter} \cup E_{data}$$

### Data Flow Complexity

For $n$ tasks with $r$ results each:

$$|\text{possible references}| = n \times r \times (n - 1)$$

Maximum data edges in a pipeline:

$$|E_{data}| \leq \frac{n(n-1)}{2} \times r$$

---

## 5. Trigger Throughput (Queueing Theory)

### Event Processing Model

The EventListener processes incoming webhooks as a queue:

$$\text{Arrival rate} = \lambda \text{ events/second}$$
$$\text{Service rate} = \mu \text{ events/second}$$

### M/M/1 Queue Model

$$\rho = \frac{\lambda}{\mu} \quad \text{(utilization)}$$

$$\bar{W} = \frac{1}{\mu - \lambda} \quad \text{(average wait time)}$$

$$\bar{L} = \frac{\rho}{1 - \rho} \quad \text{(average queue length)}$$

### Stability Condition

$$\lambda < \mu \quad \text{(arrival rate must be less than service rate)}$$

### Throughput Limits

| Component | Processing Rate | Bottleneck |
|:---|:---:|:---|
| EventListener HTTP | ~1000 events/s | Network/CPU |
| Interceptor (webhook validation) | ~500 events/s | HMAC computation |
| TriggerTemplate instantiation | ~100 PipelineRuns/s | K8s API writes |
| Pipeline scheduling | ~50 PipelineRuns/s | Pod scheduling |

### Burst Handling

For a burst of $B$ events in $\Delta t$:

$$\text{Queue depth} = B - \mu \times \Delta t$$

$$\text{Drain time} = \frac{B - \mu \times \Delta t}{\mu}$$

---

## 6. Resource Consumption (Cluster Capacity)

### Per-PipelineRun Resources

$$R_{run} = \sum_{t \in \text{tasks}} \max_{s \in \text{steps}(t)} R_s$$

Because steps within a task share a pod, the pod request is the max of its steps.

### Concurrent Run Capacity

$$N_{max\_runs} = \min\left(\frac{R_{cluster}}{R_{run}}, \frac{S_{storage}}{S_{workspace}}\right)$$

### Resource Efficiency

$$\eta = \frac{\sum_t \sum_s T_s \times R_s}{T_{pipeline} \times R_{peak}}$$

Typical efficiency for sequential pipelines: 30-50% (resources allocated but idle between steps).

### Cost Model

$$C_{pipeline} = T_{pipeline} \times \frac{R_{cpu} \times C_{cpu/hr} + R_{mem} \times C_{mem/hr}}{3600}$$

| Pipeline Duration | CPU Request | Memory Request | Cost (on-demand) |
|:---:|:---:|:---:|:---:|
| 5 min | 2 cores | 4 GB | ~$0.01 |
| 30 min | 4 cores | 8 GB | ~$0.10 |
| 120 min | 8 cores | 16 GB | ~$0.80 |

---

## 7. Pipeline Optimization (DAG Transformations)

### Minimizing Critical Path

Given tasks with dependencies and durations, minimize:

$$T_{opt} = \min_{\text{valid schedules}} T_{makespan}$$

This is the makespan minimization problem (NP-hard in general, but tractable for small DAGs).

### Task Fusion

Combining sequential tasks with same image:

$$T_{fused} = T_a + T_b + T_{transition}$$
$$T_{separate} = T_a + T_{scheduling} + T_{pull} + T_b$$

$$\text{Savings} = T_{scheduling} + T_{pull} - T_{transition} \approx 5\text{s}$$

### Parallelization Opportunities

For independent tasks with no shared state:

$$T_{parallel} = \max(T_a, T_b)$$
$$T_{sequential} = T_a + T_b$$

$$\text{Speedup} = \frac{T_a + T_b}{\max(T_a, T_b)} \leq 2$$

### Pipeline Width vs Depth Trade-off

$$T_{makespan} = f(\text{depth}(G))$$
$$R_{peak} = f(\text{width}(G))$$

Wider pipelines finish faster but use more concurrent resources.

---

*Tekton transforms CI/CD into a graph scheduling problem where tasks are nodes, dependencies are edges, and the critical path determines build time. Understanding DAG theory, queueing models, and resource allocation helps you design pipelines that are both fast and resource-efficient.*

## Prerequisites

- Directed acyclic graphs (DAGs) and topological sorting
- Critical path method (CPM)
- Queueing theory (M/M/1 model)
- Container resource management

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| Topological sort | $O(|V| + |E|)$ | $O(|V|)$ |
| Critical path | $O(|V| + |E|)$ | $O(|V|)$ |
| DAG width (antichain) | $O(|V|^2)$ | $O(|V|)$ |
| Trigger processing | $O(1)$ per event | $O(Q)$ queue |
| Result propagation | $O(|E_{data}|)$ | $O(|V| \times |R|)$ |
| Resource scheduling | $O(|V|)$ tasks | $O(|V|)$ pods |
