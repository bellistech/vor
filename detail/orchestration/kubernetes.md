# The Mathematics of Kubernetes — Orchestration Theory

> *Kubernetes is a distributed systems problem solver. Its scheduler uses scoring algorithms, its control plane relies on Raft consensus, its autoscaler applies feedback control theory, and its resource model enforces QoS through algebraic classification.*

---

## 1. Scheduler Scoring (Filtering + Scoring Pipeline)

### The Problem

Given a pod to schedule and $N$ nodes, the scheduler must select the optimal node. This is a two-phase algorithm: **filter** (eliminate infeasible), then **score** (rank remaining).

### Phase 1: Filtering (Predicate Functions)

$$\text{Feasible} = \{n \in N : \forall p \in \text{Predicates}, p(n, \text{pod}) = \text{true}\}$$

Key predicates:

| Predicate | Condition |
|:---|:---|
| PodFitsResources | $\text{requested} + \text{allocated} \leq \text{capacity}$ |
| PodFitsHost | $\text{pod.nodeName} = n.\text{name}$ or unset |
| NoDiskConflict | $\text{volumes}(\text{pod}) \cap \text{volumes}(\text{node}) = \emptyset$ |
| MatchNodeSelector | $\text{pod.nodeSelector} \subseteq n.\text{labels}$ |
| PodToleratesNodeTaints | $\forall t \in n.\text{taints}: t \in \text{pod.tolerations}$ |

### Phase 2: Scoring (Priority Functions)

Each priority function scores nodes 0-10. The final score:

$$\text{Score}(n) = \sum_{i=1}^{K} w_i \times f_i(n)$$

Where $w_i$ = weight of priority function $i$, $f_i(n) \in [0, 10]$.

### Key Priority Functions

**LeastRequestedPriority** (spread pods across nodes):

$$f_{LRP}(n) = \frac{(\text{capacity} - \text{requested})}{\text{capacity}} \times 10$$

Computed separately for CPU and memory, then averaged:

$$f_{LRP} = \frac{f_{cpu} + f_{mem}}{2}$$

**BalancedResourceAllocation** (prefer balanced CPU/memory ratios):

$$f_{BRA}(n) = 10 - 10 \times |R_{cpu} - R_{mem}|$$

Where $R_{cpu} = \text{requested\_cpu}/\text{capacity\_cpu}$ and likewise for memory.

### Worked Example

3-node cluster, scheduling a pod requesting 500m CPU, 256Mi memory:

| Node | CPU (capacity/used) | Mem (capacity/used) | LRP CPU | LRP Mem | LRP Score |
|:---|:---:|:---:|:---:|:---:|:---:|
| A | 4000m / 2000m | 8Gi / 4Gi | $(4000-2500)/4000 \times 10 = 3.75$ | $(8192-4352)/8192 \times 10 = 4.69$ | 4.22 |
| B | 4000m / 1000m | 8Gi / 6Gi | $(4000-1500)/4000 \times 10 = 6.25$ | $(8192-6400)/8192 \times 10 = 2.19$ | 4.22 |
| C | 8000m / 3000m | 16Gi / 2Gi | $(8000-3500)/8000 \times 10 = 5.63$ | $(16384-2304)/16384 \times 10 = 8.59$ | 7.11 |

**Winner: Node C** with score 7.11.

---

## 2. etcd Raft Consensus

### The Problem

The Kubernetes control plane stores all state in etcd, which uses the Raft consensus algorithm. Understanding quorum math is essential for cluster sizing.

### Quorum Formula

$$Q = \lfloor N/2 \rfloor + 1$$

### Fault Tolerance

$$F = N - Q = \lfloor (N-1)/2 \rfloor$$

| etcd Nodes ($N$) | Quorum ($Q$) | Tolerates ($F$) | Availability |
|:---:|:---:|:---:|:---:|
| 1 | 1 | 0 | No HA |
| 3 | 2 | 1 | Standard |
| 5 | 3 | 2 | High |
| 7 | 4 | 3 | Very high |

### Why Not Even Numbers?

$N=4$: $Q=3$, $F=1$. Same fault tolerance as $N=3$ but more nodes to maintain. Even-numbered clusters waste resources.

### Leader Election Timeout

$$T_{election} \in [\text{electionTimeout}, 2 \times \text{electionTimeout}]$$

Default: 1000ms. Randomized to prevent split votes. The probability of split vote decreasing with each round:

$$P(\text{split vote in round } k) \leq \left(\frac{1}{N}\right)^k$$

### Log Replication Latency

$$T_{commit} = \text{max}(T_{\text{quorum fastest responses}})$$

For $N=5$: send to 4 followers, wait for 2 responses (quorum = 3 including leader). Commit latency is the 2nd-fastest follower response.

---

## 3. Pod Resource QoS Classification

### The Problem

Kubernetes classifies pods into QoS classes that determine eviction priority. The classification is purely algebraic.

### Classification Rules

$$\text{QoS}(pod) = \begin{cases}
\text{Guaranteed} & \text{if } \forall c \in \text{containers}: \text{req} = \text{lim} \neq \emptyset \text{ for CPU and memory} \\
\text{BestEffort} & \text{if } \forall c \in \text{containers}: \text{req} = \text{lim} = \emptyset \text{ for all resources} \\
\text{Burstable} & \text{otherwise}
\end{cases}$$

### Eviction Priority

When a node is under memory pressure:

$$\text{Eviction order}: \text{BestEffort} \prec \text{Burstable} \prec \text{Guaranteed}$$

Within Burstable, pods using more than their request are evicted first:

$$\text{priority}(p) = \text{usage}(p) - \text{request}(p)$$

Higher excess = evicted first.

### Worked Examples

| Pod | CPU req/lim | Mem req/lim | QoS Class |
|:---|:---:|:---:|:---|
| A | 500m/500m | 256Mi/256Mi | Guaranteed |
| B | 250m/500m | 128Mi/256Mi | Burstable |
| C | none/none | none/none | BestEffort |
| D | 100m/none | 64Mi/128Mi | Burstable |

---

## 4. Horizontal Pod Autoscaler (HPA)

### The Scaling Formula

$$\text{desiredReplicas} = \lceil \text{currentReplicas} \times \frac{\text{currentMetricValue}}{\text{desiredMetricValue}} \rceil$$

### Worked Examples

**Example 1: CPU-based scaling**
- Current replicas: 3
- Current CPU utilization: 80%
- Target CPU utilization: 50%

$$\text{desired} = \lceil 3 \times \frac{80}{50} \rceil = \lceil 3 \times 1.6 \rceil = \lceil 4.8 \rceil = 5$$

**Example 2: Scale down**
- Current replicas: 10
- Current CPU utilization: 20%
- Target CPU utilization: 50%

$$\text{desired} = \lceil 10 \times \frac{20}{50} \rceil = \lceil 10 \times 0.4 \rceil = \lceil 4.0 \rceil = 4$$

### Stabilization Window

The HPA applies a sliding window to prevent flapping:

$$\text{recommendation}(t) = \begin{cases}
\max(\text{desired}[t-w_{down}, t]) & \text{for scale-down} \\
\min(\text{desired}[t-w_{up}, t]) & \text{for scale-up}
\end{cases}$$

Default windows: $w_{down} = 300\text{s}$, $w_{up} = 0\text{s}$.

### Tolerance Band

No scaling occurs if the ratio is within tolerance (default 10%):

$$|\frac{\text{current}}{\text{desired}} - 1.0| \leq 0.1 \implies \text{no change}$$

---

## 5. Service Load Balancing (iptables Probability Chains)

### The Problem

Kubernetes Services distribute traffic to pods using iptables probability rules. How does it achieve uniform distribution?

### The iptables Probability Chain

For $N$ pods, iptables creates $N$ rules with decreasing probabilities:

$$P_k = \frac{1}{N - k + 1} \quad \text{for } k = 1, 2, \ldots, N$$

| Rule | Probability | Effective Probability |
|:---:|:---:|:---:|
| 1 of 3 | $1/3 = 0.333$ | 0.333 |
| 2 of 3 | $1/2 = 0.500$ | $(1 - 0.333) \times 0.5 = 0.333$ |
| 3 of 3 | $1/1 = 1.000$ | $(1 - 0.667) \times 1.0 = 0.333$ |

Each pod receives exactly $\frac{1}{N}$ of traffic. The general proof:

$$P(\text{pod } k) = \prod_{i=1}^{k-1}\left(1 - \frac{1}{N-i+1}\right) \times \frac{1}{N-k+1} = \frac{1}{N}$$

### iptables vs IPVS Scaling

| Endpoints | iptables Rule Updates | IPVS Hash Lookup |
|:---:|:---:|:---:|
| 10 | $O(N)$ = fast | $O(1)$ |
| 1,000 | $O(N)$ = slow | $O(1)$ |
| 10,000 | $O(N)$ = very slow (5s+) | $O(1)$ |

At scale, IPVS wins: $O(1)$ lookup vs $O(N)$ iptables chain traversal.

---

## 6. Resource Requests and Limits (Bin Packing)

### The Problem

Scheduling pods onto nodes is a variant of the **bin packing problem** (NP-hard in general).

### Allocatable Capacity

$$\text{Allocatable} = \text{Capacity} - \text{kube-reserved} - \text{system-reserved} - \text{eviction-threshold}$$

**Worked Example:** 16 GB node:

$$\text{Allocatable} = 16384 - 1024 - 512 - 100 = 14{,}848 \text{ Mi}$$

### Packing Efficiency

$$\eta = \frac{\sum \text{pod requests}}{\text{Allocatable}} \times 100\%$$

| Scenario | Pod Sizes | Pods Packed | Efficiency |
|:---|:---|:---:|:---:|
| Uniform | 10 x 1Gi | 14 | 94.3% |
| Mixed | 2x4Gi + 6x1Gi | 8 (14Gi) | 94.3% |
| Wasteful | 3 x 5Gi | 2 (10Gi) | 67.3% |

The 5Gi pods leave 4.8Gi stranded — classic bin packing fragmentation.

---

## 7. Network Policy (Graph-Based Access Control)

### The Model

Network policies define a directed graph of allowed traffic:

$$G = (P, E) \text{ where } P = \text{pods}, E = \text{allowed connections}$$

Default (no policy): $E = P \times P$ (full mesh).

With policies: $E = \{(s, d) : \exists \text{ policy allowing } s \rightarrow d\}$

### Policy Evaluation

$$\text{allowed}(s, d) = \begin{cases}
\text{true} & \text{if no policy selects } d \text{ (default allow)} \\
\text{true} & \text{if } \exists \text{ ingress policy on } d \text{ allowing } s \\
\text{false} & \text{otherwise}
\end{cases}$$

---

## 8. Summary of Functions by Type

| Formula | Math Type | Domain |
|:---|:---|:---|
| $\sum w_i \times f_i(n)$ | Weighted sum | Scheduler scoring |
| $\lfloor N/2 \rfloor + 1$ | Floor / quorum | etcd Raft |
| $\lceil R \times C/D \rceil$ | Ceiling / ratio | HPA scaling |
| $\prod(1 - 1/(N-i+1))$ | Product series | iptables probability |
| req = lim classification | Piecewise | QoS classes |
| $(cap - req)/cap \times 10$ | Normalization | LeastRequestedPriority |

---

*Kubernetes is a distributed systems textbook running in production — Raft consensus, bin packing heuristics, feedback control loops, and probabilistic load balancing all working together to keep your pods running.*
