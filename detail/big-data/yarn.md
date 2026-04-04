# The Mathematics of YARN — Resource Scheduling Theory

> *YARN's scheduling problem is a variant of multi-dimensional bin packing with fairness constraints. The mathematics covers capacity allocation, dominant resource fairness, preemption cost modeling, and container placement optimization.*

---

## 1. Dominant Resource Fairness (Fair Division)

### The Problem

In a multi-resource environment (CPU, memory, GPU), how do you fairly allocate resources to competing users?

### The Formula

For user $i$ with demand vector $\vec{d}_i = (d_{i,cpu}, d_{i,mem})$ and total resources $\vec{R} = (R_{cpu}, R_{mem})$:

Dominant resource share:

$$s_i = \max_r \frac{d_{i,r}}{R_r}$$

DRF maximizes the minimum dominant share:

$$\max \min_i s_i$$

Allocation for user $i$ with $x_i$ containers:

$$x_i \times s_i \leq \text{fair share}$$

### Worked Examples

| User | CPU Demand | Mem Demand | Total CPU | Total Mem | Dominant Resource | Share |
|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| A | 2 cores | 4 GB | 100 | 400 GB | CPU (2%) | 2% |
| B | 1 core | 8 GB | 100 | 400 GB | Mem (2%) | 2% |
| C | 4 cores | 2 GB | 100 | 400 GB | CPU (4%) | 4% |

DRF equalizes dominant shares: A gets 50 containers (100 CPU, 200 GB), B gets 50 containers (50 CPU, 400 GB).

---

## 2. Capacity Scheduler Mathematics (Hierarchical Allocation)

### The Problem

How are resources distributed across a tree of queues with guaranteed and maximum capacities?

### The Formula

For queue $q$ with configured capacity $c_q$ and parent queue capacity $C_{parent}$:

$$C_q^{absolute} = c_q \times C_{parent}^{absolute}$$

Effective capacity considering maximum:

$$C_q^{effective} = \min\left(C_q^{max}, C_q^{absolute} + C_{idle}\right)$$

Where $C_{idle}$ is unused capacity from sibling queues (elasticity).

Queue utilization:

$$U_q = \frac{R_{used,q}}{C_q^{absolute}}$$

Over-capacity ratio:

$$O_q = \frac{R_{used,q}}{C_q^{absolute}} - 1, \quad O_q > 0 \text{ means using elasticity}$$

### Worked Examples

| Queue | Configured | Max | Absolute (100 nodes) | Elastic Max |
|:---:|:---:|:---:|:---:|:---:|
| root.prod | 60% | 80% | 60 nodes | 80 nodes |
| root.dev | 30% | 50% | 30 nodes | 50 nodes |
| root.test | 10% | 30% | 10 nodes | 30 nodes |
| root.prod.etl | 40% of prod | 100% of prod | 24 nodes | 48 nodes |

---

## 3. Container Packing (Bin Packing)

### The Problem

How does YARN place containers on nodes to maximize resource utilization? This is a multi-dimensional bin packing problem (NP-hard in general).

### The Formula

For node $n$ with capacity $(C_n^{cpu}, C_n^{mem})$ and existing allocations:

$$\text{Available}_{n,r} = C_{n,r} - \sum_{c \in \text{containers}_n} d_{c,r}$$

A container with demand $(d_{cpu}, d_{mem})$ can be placed on node $n$ if:

$$d_{cpu} \leq \text{Available}_{n,cpu} \land d_{mem} \leq \text{Available}_{n,mem}$$

Utilization efficiency:

$$\eta = \frac{\sum_n \sum_r \text{Used}_{n,r} / C_{n,r}}{N \times R}$$

Fragmentation (wasted resources due to dimensional mismatch):

$$F = 1 - \frac{\min_r(\text{Available}_{n,r} / d_r)}{\max_r(\text{Available}_{n,r} / d_r)}$$

### Worked Examples

| Node Capacity | Container Demand | Max Containers | Bottleneck |
|:---:|:---:|:---:|:---:|
| 64 GB, 16 cores | 4 GB, 2 cores | 8 | Both equal |
| 64 GB, 16 cores | 8 GB, 1 core | 8 | Memory |
| 64 GB, 16 cores | 2 GB, 4 cores | 4 | CPU |

---

## 4. Preemption Cost Model (Optimization)

### The Problem

When should YARN preempt containers to enforce queue guarantees, and which containers should be killed?

### The Formula

Preemption benefit (resources recovered for starved queue $q$):

$$B = \min(R_{needed,q}, R_{preempted})$$

Preemption cost (work lost):

$$C_{preempt} = \sum_{c \in \text{killed}} T_{elapsed,c} \times R_{c}$$

Where $T_{elapsed,c}$ is the time the container has been running and $R_c$ is its resource footprint.

Optimal preemption minimizes:

$$\min \sum_{c \in S} C_{preempt}(c) \quad \text{s.t.} \quad \sum_{c \in S} R_c \geq R_{needed}$$

This is a variant of the knapsack problem. YARN uses greedy heuristics:
1. Preempt youngest containers first (least work lost)
2. Preempt from most over-capacity queues first

### Worked Examples

| Container Age | Resources | Preempt Cost | Priority |
|:---:|:---:|:---:|:---:|
| 5 min | 4 GB, 2 cores | 20 GB-min | Preempt first |
| 60 min | 4 GB, 2 cores | 240 GB-min | Preempt last |
| 5 min | 16 GB, 4 cores | 80 GB-min | Preempt second |

---

## 5. Application Lifecycle Timing (Queuing Theory)

### The Problem

What determines the end-to-end latency from job submission to completion?

### The Formula

$$T_{total} = T_{queue} + T_{AM\_launch} + T_{container\_alloc} + T_{execution} + T_{cleanup}$$

Queue wait time (M/M/1 approximation for FIFO):

$$T_{queue} = \frac{\rho}{\mu(1 - \rho)}$$

Where:
- $\rho = \lambda / \mu$ = utilization
- $\lambda$ = arrival rate
- $\mu$ = service rate

Container allocation delay:

$$T_{alloc} = \frac{N_{containers}}{R_{heartbeat}} \times T_{heartbeat}$$

Where $R_{heartbeat}$ is containers allocated per heartbeat cycle (default 1s).

### Worked Examples

| Containers Needed | Heartbeat Interval | Alloc Rate | Allocation Time |
|:---:|:---:|:---:|:---:|
| 20 | 1 s | 5/heartbeat | ~4 s |
| 200 | 1 s | 20/heartbeat | ~10 s |
| 1000 | 1 s | 50/heartbeat | ~20 s |

---

## 6. Node Manager Resource Model (Linear Constraints)

### The Problem

How should NodeManager resources be partitioned between YARN and system processes?

### The Formula

$$M_{yarn} = M_{physical} - M_{os} - M_{datanode} - M_{nodemanager}$$

Rule of thumb:

$$M_{yarn} = M_{physical} \times 0.80$$

Cores:

$$V_{cores} = C_{physical} \times f_{oversubscription}$$

Where $f_{oversubscription} \in [1.0, 2.0]$. Use 1.0 for CPU-bound, up to 2.0 for I/O-bound workloads.

### Worked Examples

| Physical RAM | OS+Services | YARN Available | Container Size | Max Containers |
|:---:|:---:|:---:|:---:|:---:|
| 128 GB | 20 GB | 108 GB | 8 GB | 13 |
| 256 GB | 30 GB | 226 GB | 16 GB | 14 |
| 64 GB | 12 GB | 52 GB | 4 GB | 13 |

---

## 7. Summary of Formulas

| Formula | Type | Domain |
|---------|------|--------|
| $s_i = \max_r(d_{i,r}/R_r)$ | Dominant resource share | Fair division |
| $C_q = c_q \times C_{parent}$ | Absolute capacity | Hierarchical scheduling |
| $F = 1 - \min/\max$ available ratios | Fragmentation | Bin packing |
| $C_{preempt} = T_{elapsed} \times R$ | Preemption cost | Optimization |
| $T_{queue} = \rho / (\mu(1-\rho))$ | Wait time | Queuing theory |
| $M_{yarn} = M_{physical} \times 0.80$ | Memory budget | Resource planning |

## Prerequisites

- queuing-theory, bin-packing, fair-division, linear-programming, hadoop

## Complexity

| Operation | Time | Space |
|-----------|------|-------|
| Container scheduling decision | O(N) nodes scanned | O(1) per decision |
| Queue capacity calculation | O(Q) queue tree depth | O(Q) queue metadata |
| Preemption selection (greedy) | O(C log C) sort containers | O(C) candidate set |
| Node heartbeat processing | O(1) amortized | O(C) per node |
| Application submission | O(Q) queue lookup | O(1) app metadata |
| DRF allocation round | O(U * R) users x resources | O(U) share tracking |
