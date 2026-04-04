# The Mathematics of Java -- Memory Management and Concurrency

> *The JVM is a theorem prover in disguise: every garbage collection cycle proves that certain objects are unreachable, and every happens-before edge proves that shared state is consistent.*

---

## 1. Garbage Collection Reachability (Graph Theory)

### The Problem

The JVM must determine which objects are live (reachable from GC roots) and which are garbage. This is a graph reachability problem over the object reference graph, executed under real-time constraints.

### The Formula

Given the object graph $G = (V, E)$ with GC roots $R \subseteq V$, the live set is:

$$L = \{v \in V : \exists \text{ path } r \leadsto v \text{ for some } r \in R\}$$

The dead set $D = V \setminus L$ can be reclaimed. A mark phase computes $L$ via BFS/DFS in $O(|L| + |E_L|)$ time where $E_L$ is edges within the live subgraph.

The heap occupancy after collection $k$ follows:

$$H_{k+1} = L_k + A_k - C_k$$

where $L_k$ is the surviving live set, $A_k$ is bytes allocated since collection $k$, and $C_k$ is bytes collected.

### Worked Examples

Heap size 4 GiB, live set 1.2 GiB, allocation rate 500 MiB/s, GC triggered at 90% occupancy.

Time to fill: $(4 \times 0.9 - 1.2) / 0.5 = 2.4 / 0.5 = 4.8\text{s}$ between collections.

G1 pause target 200ms: must mark and evacuate within 200ms. If marking rate is 8 GiB/s, marking 1.2 GiB takes $1.2/8 = 0.15\text{s} = 150\text{ms}$, leaving 50ms for evacuation. With 50 MiB to evacuate (average region liveness): $50/2000 = 0.025\text{s}$ at 2 GiB/s copy rate. Total: 175ms -- within budget.

---

## 2. Generational Hypothesis (Probabilistic Models)

### The Problem

Most objects die young. The generational hypothesis quantifies this observation and justifies partitioning the heap into young and old generations with different collection frequencies.

### The Formula

Object survival probability as a function of age $t$ (in allocation epochs) follows an exponential decay:

$$P(\text{survive} | \text{age} = t) = e^{-\lambda t}$$

where $\lambda$ is the mortality rate. The expected number of surviving objects from a cohort of $N_0$ objects after $t$ epochs:

$$N(t) = N_0 \cdot e^{-\lambda t}$$

The optimal young generation size $Y$ minimizes total GC work:

$$W_{\text{total}} = \frac{A}{Y} \cdot (Y + S_Y) + \frac{S_Y}{O} \cdot (O + S_O)$$

where $A$ is allocation rate, $S_Y$ is young generation survivors, $O$ is old generation size, and $S_O$ is old generation survivors.

### Worked Examples

With $\lambda = 2.0$ (aggressive mortality), after 1 epoch: $P = e^{-2} \approx 0.135$ (86.5% of objects are dead). After 3 epochs: $P = e^{-6} \approx 0.0025$ (99.75% dead).

For a cohort of 1 million objects: $N(3) = 10^6 \times 0.0025 = 2{,}500$ survivors. Only these 2,500 need to be copied to the old generation, making young GC extremely efficient.

---

## 3. Virtual Threads and Little's Law (Queueing Theory)

### The Problem

Virtual threads (Project Loom) enable millions of concurrent tasks. Understanding throughput requires queueing theory: how many concurrent requests are needed to saturate a system?

### The Formula

Little's Law relates the average number of items in a system $L$, the arrival rate $\lambda$, and the average time in the system $W$:

$$L = \lambda \cdot W$$

For a server handling requests with average latency $W$ and target throughput $\lambda$:

$$\text{required concurrency} = L = \lambda \cdot W$$

With platform threads capped at $N_{\text{threads}}$, maximum throughput is:

$$\lambda_{\max} = \frac{N_{\text{threads}}}{W}$$

With virtual threads, $N_{\text{threads}} \to \infty$ effectively, so throughput is limited only by backend resources.

### Worked Examples

A web service with $W = 200\text{ms}$ average latency (including database I/O). Target: $\lambda = 10{,}000$ req/s.

Required concurrency: $L = 10{,}000 \times 0.2 = 2{,}000$ concurrent requests.

With platform threads (max ~2,000 threads at 1 MiB stack each = 2 GiB RAM): barely achievable, uses 2 GiB just for stacks.

With virtual threads (~1 KiB stack each): $2{,}000 \times 1\text{KiB} = 2\text{MiB}$. Can scale to $L = 1{,}000{,}000$ using only 1 GiB. Throughput becomes $\lambda = 1{,}000{,}000 / 0.2 = 5{,}000{,}000$ req/s (limited by CPU and I/O, not threads).

---

## 4. Happens-Before Ordering (Partial Order Theory)

### The Problem

Java's Memory Model (JMM) defines when writes by one thread are visible to reads by another. This is formalized as a partial order called "happens-before" that prevents data races without requiring total ordering.

### The Formula

The happens-before relation $\xrightarrow{hb}$ is the transitive closure of program order and synchronization order:

$$\xrightarrow{hb} = (\xrightarrow{po} \cup \xrightarrow{so})^+$$

A read $r$ of variable $v$ sees write $w$ if:

$$w \xrightarrow{hb} r \quad \land \quad \nexists w' : w \xrightarrow{hb} w' \xrightarrow{hb} r$$

A data race exists on variable $v$ if there exist accesses $a, b$ to $v$ where at least one is a write and:

$$\neg(a \xrightarrow{hb} b) \land \neg(b \xrightarrow{hb} a)$$

A program is data-race-free (DRF) iff no execution produces a data race. The DRF guarantee states: DRF programs have sequentially consistent semantics.

### Worked Examples

Thread 1: `x = 1; lock(m); y = 2; unlock(m);`
Thread 2: `lock(m); r1 = y; unlock(m); r2 = x;`

Happens-before edges: unlock(m) in T1 $\xrightarrow{so}$ lock(m) in T2. Therefore:
- `y = 2` $\xrightarrow{hb}$ `r1 = y`, so `r1 = 2` is guaranteed.
- `x = 1` $\xrightarrow{hb}$ `r2 = x` (transitively via the lock), so `r2 = 1` is guaranteed.

Without the lock, both reads could see stale values (0) due to CPU caches and compiler reordering.

---

## 5. ZGC Pause Time Analysis (Real-Time Systems)

### The Problem

ZGC targets sub-millisecond pause times regardless of heap size. It achieves this through colored pointers and load barriers that shift GC work from stop-the-world pauses to concurrent phases.

### The Formula

ZGC pause time has three components:

$$T_{\text{pause}} = T_{\text{roots}} + T_{\text{sync}} + T_{\text{cleanup}}$$

Root scanning is $O(|R|)$ where $R$ is the GC root set (thread stacks, static fields):

$$T_{\text{roots}} = |R| \cdot c_{\text{scan}}$$

The concurrent marking overhead per mutator operation is:

$$\text{overhead} = P_{\text{barrier}} \cdot c_{\text{barrier}}$$

where $P_{\text{barrier}}$ is the probability a load hits a stale pointer (requiring barrier slow path) and $c_{\text{barrier}}$ is the barrier cost. During concurrent relocation:

$$P_{\text{barrier}} = \frac{|\text{relocating pages}|}{|\text{total pages}|}$$

### Worked Examples

Heap: 128 GiB, 500 threads, 2,000 static roots. Root set: $|R| = 500 \times 50 + 2{,}000 = 27{,}000$ roots. At $c_{\text{scan}} = 30\text{ns}$ per root:

$$T_{\text{roots}} = 27{,}000 \times 30\text{ns} = 0.81\text{ms}$$

Total pause: $0.81 + 0.05 + 0.02 = 0.88\text{ms}$ -- under 1ms even with 128 GiB heap.

Concurrent overhead: relocating 500 of 65,536 pages: $P_{\text{barrier}} = 500/65536 \approx 0.76\%$. At $c_{\text{barrier}} = 50\text{ns}$, average overhead per load is $0.0076 \times 50 = 0.38\text{ns}$ -- negligible.

---

## 6. Stream Pipeline Fusion (Algebra of Transformations)

### The Problem

Java Streams compose operations (map, filter, flatMap) into a pipeline that executes in a single pass. Understanding fusion explains why streams avoid creating intermediate collections and how short-circuiting works.

### The Formula

A stream pipeline is a composition of functions. For a source $S$ of $n$ elements, the pipeline $f_k \circ \cdots \circ f_2 \circ f_1$ with filter selectivities $\sigma_i$ processes:

$$\text{elements at stage } j = n \cdot \prod_{i=1}^{j} \sigma_i$$

where $\sigma_i = 1$ for map/flatMap and $\sigma_i < 1$ for filter. A short-circuiting terminal (e.g., `findFirst`, `limit(k)`) stops after producing $k$ results. The expected elements consumed from the source:

$$E[\text{consumed}] = \frac{k}{\prod_{i} \sigma_i}$$

For a parallel stream with $p$ threads and splittable source, the work per thread is:

$$W_{\text{thread}} \approx \frac{n \cdot C_{\text{pipeline}}}{p} + C_{\text{merge}}$$

where $C_{\text{pipeline}}$ is the per-element cost and $C_{\text{merge}}$ is the combiner cost.

### Worked Examples

Pipeline: `stream.filter(x -> x > 0).map(x -> x * 2).filter(x -> x < 100).limit(10)`

Source: 1,000,000 elements. $\sigma_1 = 0.5$ (50% positive), $\sigma_2 = 1.0$ (map), $\sigma_3 = 0.3$ (30% under 100 after doubling).

Overall selectivity: $0.5 \times 1.0 \times 0.3 = 0.15$.

Expected elements consumed to produce 10 results: $10 / 0.15 \approx 67$ elements from the source. The pipeline processes 67 elements, not 1,000,000 -- a $15{,}000\times$ reduction due to short-circuiting.

---

## Prerequisites

- Graph theory (reachability, BFS/DFS, directed graphs)
- Probability theory (exponential distribution, survival functions)
- Queueing theory (Little's Law, arrival rates, service times)
- Partial order theory (transitive closure, happens-before)
- Amortized analysis and asymptotic complexity
- Real-time systems (worst-case execution time, bounded pauses)
- Memory hierarchy (cache coherence, memory barriers)
- Function composition and algebraic transformation laws
