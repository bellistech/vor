# The Mathematics of Nomad — Scheduling Algorithms and Bin Packing

> *Nomad's scheduler solves a variant of the multi-dimensional bin packing problem at evaluation time, placing task groups onto nodes while respecting CPU, memory, and constraint requirements. The scheduling algorithms, deployment strategies, and failure domains all have precise mathematical models that determine cluster utilization and job placement quality.*

---

## 1. Bin Packing (Combinatorial Optimization)

### The Problem

Nomad must place $k$ task groups with resource requirements $(c_i, m_i)$ for CPU and memory onto $n$ nodes with capacities $(C_j, M_j)$. This is a multi-dimensional bin packing problem, which is NP-hard. Nomad uses a scoring heuristic rather than an exact solver.

### The Formula

Nomad's bin packing score for placing task $i$ on node $j$:

$$\text{score}(i, j) = \frac{c_i}{C_j^{remaining}} \cdot w_c + \frac{m_i}{M_j^{remaining}} \cdot w_m$$

Where $w_c$ and $w_m$ are weights for CPU and memory (default equal).

The scheduler picks the node with the highest score (tightest fit), maximizing utilization:

$$j^* = \arg\max_{j \in \text{feasible}} \text{score}(i, j)$$

A node is feasible if:

$$c_i \leq C_j^{remaining} \wedge m_i \leq M_j^{remaining} \wedge \text{constraints}(i, j) = \text{true}$$

### Worked Example

3 nodes (4000 MHz CPU, 8 GB RAM each), placing a task requiring 1000 MHz CPU, 2 GB RAM:

| Node | CPU Remaining | RAM Remaining | CPU Util | RAM Util | Score |
|:---|:---:|:---:|:---:|:---:|:---:|
| A | 4000 MHz | 8 GB | 0.25 | 0.25 | 0.50 |
| B | 2000 MHz | 4 GB | 0.50 | 0.50 | 1.00 |
| C | 1500 MHz | 3 GB | 0.67 | 0.67 | 1.33 |

$$j^* = C \quad (\text{highest score, tightest fit})$$

---

## 2. Spread Scheduling (Entropy Maximization)

### The Problem

While bin packing minimizes node count, spread scheduling distributes allocations across failure domains (racks, AZs). Nomad's `spread` block maximizes allocation entropy across a target attribute.

### The Formula

For $k$ allocations distributed across $d$ failure domains, the spread score uses deviation from target weights:

$$\text{spread\_score} = 1 - \frac{\sum_{i=1}^{d} |a_i - t_i \cdot k|}{k}$$

Where $a_i$ is actual allocations in domain $i$ and $t_i$ is the target weight (summing to 1).

Perfect spread ($a_i = t_i \cdot k$ for all $i$):

$$\text{spread\_score} = 1.0$$

### Worked Example

6 allocations across 3 AZs with equal weights ($t_i = 1/3$):

| Distribution | AZ-a | AZ-b | AZ-c | Score |
|:---|:---:|:---:|:---:|:---:|
| Perfect | 2 | 2 | 2 | $1 - 0/6 = 1.0$ |
| Skewed | 4 | 1 | 1 | $1 - (2+1+1)/6 = 0.33$ |
| Worst case | 6 | 0 | 0 | $1 - (4+2+2)/6 = -0.33$ |

---

## 3. Deployment Mathematics (Reliability)

### The Problem

Rolling updates, canary deployments, and blue-green deployments trade update speed against risk. We can model the blast radius and detection probability for each strategy.

### The Formula

For a rolling update with `count = N` and `max_parallel = P`:

$$\text{batches} = \left\lceil \frac{N}{P} \right\rceil$$

$$\text{blast\_radius}(t) = \min(P \cdot t, N) \quad \text{(allocations affected after } t \text{ batches)}$$

For canary deployment with $C$ canaries:

$$\text{blast\_radius}_{canary} = C \quad \text{(until promotion)}$$

$$\text{error\_detection\_probability} = 1 - (1 - p_{error})^{C \cdot R}$$

Where $p_{error}$ is per-request error probability and $R$ is requests during canary window.

### Worked Examples

Service with $N = 12$, error rate $p_{error} = 0.01$, 100 requests during health check window:

| Strategy | Blast Radius | Detection Prob. |
|:---|:---:|:---:|
| Rolling ($P=1$) | 1 (per batch) | $1 - 0.99^{100} = 63.4\%$ |
| Rolling ($P=4$) | 4 (per batch) | $1 - 0.99^{400} = 98.2\%$ |
| Canary ($C=2$) | 2 (held) | $1 - 0.99^{200} = 86.6\%$ |
| Blue-green ($C=12$) | 12 (but instant rollback) | $1 - 0.99^{1200} = 99.999\%$ |

Time to complete ($min\_healthy = 30\text{s}$, $stagger = 10\text{s}$):

$$T_{rolling} = \left\lceil \frac{N}{P} \right\rceil \cdot (\text{min\_healthy} + \text{stagger})$$

| Strategy | Batches | Total Time |
|:---|:---:|:---:|
| $P=1$ | 12 | $12 \times 40 = 480\text{s}$ |
| $P=2$ | 6 | $6 \times 40 = 240\text{s}$ |
| $P=4$ | 3 | $3 \times 40 = 120\text{s}$ |
| Blue-green | 1 | $40\text{s}$ |

---

## 4. Resource Utilization Efficiency (Linear Programming)

### The Problem

Nomad's resource model allocates fixed CPU and memory to each task. The gap between allocated and actually used resources is waste. We can measure cluster efficiency and identify resource stranding.

### The Formula

Cluster utilization efficiency:

$$\eta_{cpu} = \frac{\sum_{i} c_i^{used}}{\sum_{j} C_j^{total}}$$

$$\eta_{mem} = \frac{\sum_{i} m_i^{used}}{\sum_{j} M_j^{total}}$$

Resource stranding occurs when one dimension is exhausted while another has capacity:

$$\text{stranded}_{cpu} = \sum_{j} C_j^{remaining} \quad \text{where } M_j^{remaining} < m_{min}$$

$$\text{stranded}_{mem} = \sum_{j} M_j^{remaining} \quad \text{where } C_j^{remaining} < c_{min}$$

### Worked Example

3 nodes (4000 MHz, 8 GB each). Tasks: Type A (1000 MHz, 4 GB), Type B (2000 MHz, 1 GB).

After packing: Node 1 has 2xA (2000 MHz used, 8 GB used), Node 2 has 2xB (4000 MHz used, 2 GB used).

| Node | CPU Remaining | RAM Remaining | Stranded? |
|:---|:---:|:---:|:---|
| Node 1 | 2000 MHz | 0 GB | 2000 MHz CPU stranded (no RAM for any task) |
| Node 2 | 0 MHz | 6 GB | 6 GB RAM stranded (no CPU for any task) |
| Node 3 | 4000 MHz | 8 GB | fully available |

$$\eta_{cpu} = \frac{6000}{12000} = 50\%, \quad \eta_{mem} = \frac{10}{24} = 41.7\%$$

---

## 5. Evaluation and Scheduling Latency (Queueing Theory)

### The Problem

Nomad processes job changes through evaluations queued in a broker. Scheduling latency depends on evaluation queue depth and processing time. We can model this as an M/M/1 queue.

### The Formula

For evaluation arrival rate $\lambda$ and processing rate $\mu$:

$$\text{utilization: } \rho = \frac{\lambda}{\mu}$$

$$\text{average queue length: } L_q = \frac{\rho^2}{1 - \rho}$$

$$\text{average wait time: } W_q = \frac{\rho}{\mu(1 - \rho)}$$

System is stable only when $\rho < 1$.

### Worked Example

Cluster with $\mu = 200$ evals/sec, varying load:

| Eval Rate ($\lambda$) | Utilization ($\rho$) | Avg Queue | Avg Wait |
|:---:|:---:|:---:|:---:|
| 50/s | 0.25 | 0.08 | 1.7ms |
| 100/s | 0.50 | 0.50 | 5.0ms |
| 150/s | 0.75 | 2.25 | 15.0ms |
| 190/s | 0.95 | 18.05 | 95.0ms |

At 95% utilization, scheduling latency spikes 56x compared to 25% utilization.

---

## 6. Multi-Region Federation (Consistency)

### The Problem

Nomad supports multi-region federation where each region has its own Raft cluster. Job forwarding across regions introduces additional latency and failure modes.

### The Formula

Cross-region job deployment time:

$$T_{deploy} = T_{local\_raft} + RTT_{region} + T_{remote\_raft} + T_{scheduling}$$

For $R$ regions deploying simultaneously:

$$T_{total} = T_{local\_raft} + \max_{r=1}^{R}(RTT_r + T_{remote\_raft_r} + T_{scheduling_r})$$

Availability: each region operates independently. Global unavailability requires all regions to fail:

$$P(\text{global\_outage}) = \prod_{r=1}^{R} P(\text{region\_r\_outage})$$

### Worked Example

3 regions, each with 99.9% availability:

$$P(\text{global\_outage}) = (0.001)^3 = 10^{-9}$$

$$\text{Availability} = 1 - 10^{-9} = 99.9999999\% \text{ (nine 9s)}$$

## Prerequisites

- consul, docker, linux-process-management, bin-packing, distributed-systems, queueing-theory
