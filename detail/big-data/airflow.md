# The Mathematics of Airflow -- DAG Scheduling and Task Concurrency

> *Airflow's scheduler must solve a constrained topological ordering problem across time: given a DAG of dependent tasks, resource pools with finite slots, and multiple concurrent DAG runs, determine which tasks to execute and when. The underlying mathematics draws on graph theory, queuing theory, and combinatorial optimization.*

---

## 1. DAG Theory (Graph Foundations)

### The Problem
Every Airflow workflow is a directed acyclic graph. The scheduler must find a valid execution order that respects all dependency edges while maximizing parallelism within resource constraints.

### The Formula
A DAG $G = (V, E)$ has vertices $V$ (tasks) and directed edges $E$ (dependencies). A topological sort produces an ordering $\sigma$ such that:

$$\forall (u, v) \in E: \sigma(u) < \sigma(v)$$

The number of possible topological orderings of a DAG is given by:

$$|\text{TopSort}(G)| = \frac{|V|!}{\prod_{v \in V} |D(v)|!} \cdot C(G)$$

where $D(v)$ is the set of descendants of $v$ and $C(G)$ is a correction factor depending on the DAG structure. For a general DAG, computing this count is $\#P$-complete.

### Worked Examples
Consider a diamond DAG: $A \to B$, $A \to C$, $B \to D$, $C \to D$.

Valid topological orderings:
- $A, B, C, D$
- $A, C, B, D$

The critical path length is $\ell = 3$ (A-B-D or A-C-D). With unbounded parallelism, completion time equals the critical path length. With a pool of size $p = 1$, completion time is $|V| = 4$ steps.

---

## 2. Critical Path Scheduling (Execution Optimization)

### The Problem
Given task durations and the DAG structure, what is the minimum makespan (total wall-clock time) to complete all tasks?

### The Formula
Define $w(v)$ as the execution time of task $v$. The earliest start time $\text{ES}(v)$ and earliest finish $\text{EF}(v)$:

$$\text{ES}(v) = \max_{u \in \text{pred}(v)} \text{EF}(u)$$
$$\text{EF}(v) = \text{ES}(v) + w(v)$$

The makespan is:

$$M = \max_{v \in V} \text{EF}(v)$$

The slack (float) of task $v$ determines whether it is on the critical path:

$$\text{Slack}(v) = \text{LS}(v) - \text{ES}(v)$$

where $\text{LS}(v)$ is the latest start time. Tasks with $\text{Slack}(v) = 0$ form the critical path.

### Worked Examples
Pipeline: Extract (5m) -> Transform (10m) -> Load (3m), with a parallel Validate (4m) branching from Extract and merging at Load.

$$\text{ES(Extract)} = 0, \quad \text{EF(Extract)} = 5$$
$$\text{ES(Transform)} = 5, \quad \text{EF(Transform)} = 15$$
$$\text{ES(Validate)} = 5, \quad \text{EF(Validate)} = 9$$
$$\text{ES(Load)} = \max(15, 9) = 15, \quad \text{EF(Load)} = 18$$

Makespan $M = 18$ minutes. Critical path: Extract -> Transform -> Load. Validate has slack of $15 - 5 = 10$ minutes... wait, let us recalculate:

$$\text{LS(Load)} = 18 - 3 = 15$$
$$\text{LS(Transform)} = 15 - 10 = 5, \quad \text{Slack} = 5 - 5 = 0$$
$$\text{LS(Validate)} = 15 - 4 = 11, \quad \text{Slack} = 11 - 5 = 6$$

Validate has 6 minutes of slack; Transform is critical.

---

## 3. Pool Contention (Resource-Constrained Scheduling)

### The Problem
Airflow pools limit concurrent task execution against shared resources. How does pool size affect throughput and makespan?

### The Formula
Given $n$ independent tasks each requiring 1 slot from a pool of size $p$, and task durations $w_1, w_2, \ldots, w_n$, the makespan under greedy list scheduling satisfies:

$$M_p \leq \frac{1}{p} \sum_{i=1}^{n} w_i + \left(1 - \frac{1}{p}\right) w_{\max}$$

For identical task durations $w$:

$$M_p = \left\lceil \frac{n}{p} \right\rceil \cdot w$$

The throughput (tasks per unit time) approaches:

$$\Theta = \frac{p}{\bar{w}}$$

where $\bar{w}$ is the mean task duration.

### Worked Examples
A pool of size $p = 5$ with $n = 23$ tasks each taking $w = 2$ minutes:

$$M_5 = \left\lceil \frac{23}{5} \right\rceil \cdot 2 = 5 \cdot 2 = 10 \text{ minutes}$$

Throughput: $\Theta = 5 / 2 = 2.5$ tasks/minute.

If tasks have variable durations $[1, 1, 3, 3, 5]$ with $p = 2$:

$$M_2 \leq \frac{1 + 1 + 3 + 3 + 5}{2} + \left(1 - \frac{1}{2}\right) \cdot 5 = 6.5 + 2.5 = 9 \text{ minutes}$$

---

## 4. Scheduling Intervals (Temporal Semantics)

### The Problem
Airflow's `execution_date` represents the start of a data interval, not when the DAG runs. Understanding this temporal model is critical for correct backfill behavior.

### The Formula
For a DAG with schedule interval $\Delta$ and start date $t_0$, the $k$-th DAG run has:

$$\text{execution\_date}_k = t_0 + k \cdot \Delta$$
$$\text{actual\_run\_time}_k = t_0 + (k + 1) \cdot \Delta$$

The data interval for run $k$ is:

$$[t_0 + k \cdot \Delta, \; t_0 + (k + 1) \cdot \Delta)$$

For a backfill from $t_s$ to $t_e$, the number of DAG runs generated is:

$$N = \left\lfloor \frac{t_e - t_s}{\Delta} \right\rfloor$$

### Worked Examples
A daily DAG (`@daily`) with `start_date = 2024-01-01`:

- Run 0: `execution_date = 2024-01-01`, runs at `2024-01-02 00:00`, processes data for Jan 1
- Run 30: `execution_date = 2024-01-31`, runs at `2024-02-01 00:00`, processes data for Jan 31

Backfill from Jan 1 to Mar 31: $N = \lfloor 90 / 1 \rfloor = 90$ DAG runs.

With `max_active_runs = 3` and each run taking 20 minutes, minimum backfill time:

$$T_{\text{backfill}} = \left\lceil \frac{90}{3} \right\rceil \cdot 20 = 600 \text{ minutes} = 10 \text{ hours}$$

---

## 5. Executor Concurrency (Queuing Model)

### The Problem
The executor (Local, Celery, Kubernetes) determines how many tasks run simultaneously. Model the system as a queue to predict task wait times and resource utilization.

### The Formula
Modeling the executor as an $M/G/c$ queue (Poisson arrivals, general service times, $c$ worker slots), the expected wait time in queue is approximated by:

$$W_q \approx \frac{C_s^2 + 1}{2} \cdot \frac{\rho^{\sqrt{2(c+1)}}}{c(1 - \rho)} \cdot \bar{w}$$

where $\rho = \lambda \bar{w} / c$ is the utilization, $\lambda$ is the task arrival rate, $\bar{w}$ is mean service time, and $C_s$ is the coefficient of variation of service times.

Worker utilization:

$$\rho = \frac{\lambda \bar{w}}{c}$$

The system is stable when $\rho < 1$, i.e., $c > \lambda \bar{w}$.

### Worked Examples
A CeleryExecutor with $c = 16$ workers, tasks arriving at $\lambda = 10$ tasks/min, mean duration $\bar{w} = 1.2$ min, $C_s = 0.5$:

$$\rho = \frac{10 \times 1.2}{16} = 0.75$$

$$W_q \approx \frac{0.25 + 1}{2} \cdot \frac{0.75^{\sqrt{34}}}{16 \times 0.25} \cdot 1.2$$

$$W_q \approx 0.625 \cdot \frac{0.75^{5.83}}{4} \cdot 1.2 \approx 0.625 \cdot \frac{0.178}{4} \cdot 1.2 \approx 0.033 \text{ min} \approx 2 \text{ seconds}$$

At 75% utilization, queue wait is negligible. At $\rho = 0.95$ ($c = 13$), wait time increases dramatically.

---

## 6. Priority Weighting (Task Ordering)

### The Problem
When multiple tasks are eligible for execution, Airflow uses priority weights to determine order. The `weight_rule` parameter changes how weights propagate through the DAG.

### The Formula
For `weight_rule = 'downstream'` (default), the effective priority of task $v$:

$$P_{\text{down}}(v) = w(v) + \sum_{u \in \text{desc}(v)} w(u)$$

For `weight_rule = 'upstream'`:

$$P_{\text{up}}(v) = w(v) + \sum_{u \in \text{anc}(v)} w(u)$$

For `weight_rule = 'absolute'`:

$$P_{\text{abs}}(v) = w(v)$$

where $w(v)$ is the `priority_weight` of task $v$, $\text{desc}(v)$ is all descendants, and $\text{anc}(v)$ is all ancestors.

### Worked Examples
A chain $A(w=1) \to B(w=1) \to C(w=1) \to D(w=1)$ with `weight_rule='downstream'`:

$$P(A) = 1 + 1 + 1 + 1 = 4$$
$$P(B) = 1 + 1 + 1 = 3$$
$$P(C) = 1 + 1 = 2$$
$$P(D) = 1$$

Task A gets highest priority, ensuring the longest dependency chain starts first, which minimizes overall makespan.

---

## Prerequisites
- graph-theory, topological-sort, queuing-theory, critical-path-method, combinatorial-optimization
