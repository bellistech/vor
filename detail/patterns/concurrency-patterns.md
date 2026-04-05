# The Mathematics of Concurrency — From CSP to Lock-Free Algorithms

> *Concurrent systems are governed by formal models of process interaction, temporal ordering, and the fundamental limits of parallelism. Understanding the mathematics beneath reveals why certain bugs are inevitable without proper synchronization.*

---

## 1. Communicating Sequential Processes (Formal Concurrency)

### The Problem

How do we formally reason about programs with multiple interacting processes without descending into ad-hoc analysis?

### The Formula

Hoare's CSP (1978) models processes as entities that communicate via synchronous message passing on named channels. A process $P$ is defined by the events it can engage in:

$$P = (a \rightarrow P') \mid (P_1 \parallel P_2) \mid (P_1 \square P_2) \mid STOP$$

Where:
- $a \rightarrow P'$ means "engage in event $a$, then behave as $P'$"
- $P_1 \parallel P_2$ is parallel composition
- $P_1 \square P_2$ is external choice (environment decides)
- $STOP$ is deadlock

The **traces model** defines the behavior of a process as the set of all possible sequences of events:

$$\text{traces}(P) = \{ \langle a_1, a_2, \ldots, a_n \rangle \mid P \text{ can perform } a_1 \text{ then } a_2 \ldots \}$$

A refinement relation: $P$ refines $Q$ (written $Q \sqsubseteq P$) if $\text{traces}(P) \subseteq \text{traces}(Q)$.

### Worked Examples

**Two-slot buffer** modeled in CSP:

$$\text{BUFF}_0 = \text{in} \rightarrow \text{BUFF}_1$$
$$\text{BUFF}_1 = \text{out} \rightarrow \text{BUFF}_0 \square \text{in} \rightarrow \text{BUFF}_2$$
$$\text{BUFF}_2 = \text{out} \rightarrow \text{BUFF}_1$$

This models a buffer of capacity 2. $\text{BUFF}_0$ can only accept input. $\text{BUFF}_2$ can only output. $\text{BUFF}_1$ can do either.

Go channels directly implement CSP: `ch := make(chan int)` creates a synchronous channel. `make(chan int, 2)` creates the two-slot buffer above.

---

## 2. Happens-Before Relations (Memory Ordering)

### The Problem

In concurrent systems, different threads may observe memory writes in different orders. When can we guarantee one operation's effects are visible to another?

### The Formula

The happens-before relation $\rightarrow_{hb}$ is a partial order on events defined by:

1. **Program order**: If $a$ and $b$ are in the same thread and $a$ precedes $b$, then $a \rightarrow_{hb} b$
2. **Synchronization**: If $a$ is a release (unlock, channel send) and $b$ is the corresponding acquire (lock, channel receive), then $a \rightarrow_{hb} b$
3. **Transitivity**: If $a \rightarrow_{hb} b$ and $b \rightarrow_{hb} c$, then $a \rightarrow_{hb} c$

Two events are **concurrent** if neither happens-before the other:

$$a \parallel b \iff \neg(a \rightarrow_{hb} b) \land \neg(b \rightarrow_{hb} a)$$

A **data race** exists when two concurrent events access the same memory location and at least one is a write.

### Worked Examples

Thread 1: `x = 1; unlock(mu)`
Thread 2: `lock(mu); print(x)`

By rule 1: `x=1` $\rightarrow_{hb}$ `unlock(mu)` within Thread 1.
By rule 2: `unlock(mu)` $\rightarrow_{hb}$ `lock(mu)` across threads.
By rule 1: `lock(mu)` $\rightarrow_{hb}$ `print(x)` within Thread 2.
By transitivity: `x=1` $\rightarrow_{hb}$ `print(x)`. Thread 2 sees $x = 1$.

---

## 3. Lamport Clocks (Logical Time)

### The Problem

Physical clocks are unreliable in distributed systems (drift, skew, NTP jitter). How do we establish a consistent ordering of events without synchronized clocks?

### The Formula

Each process $P_i$ maintains a counter $C_i$. Lamport's clock rules:

1. Before each event in $P_i$: $C_i := C_i + 1$
2. When $P_i$ sends message $m$: attach timestamp $t_m = C_i$
3. When $P_j$ receives $m$ with timestamp $t_m$: $C_j := \max(C_j, t_m) + 1$

**Clock condition**: If $a \rightarrow b$, then $C(a) < C(b)$.

The converse does NOT hold: $C(a) < C(b) \not\Rightarrow a \rightarrow b$. Lamport clocks provide a total order consistent with causality but cannot detect concurrency.

**Vector clocks** extend this: each process maintains a vector $V_i[1..n]$ for $n$ processes. $a \rightarrow b \iff V(a) < V(b)$ (componentwise). This captures causality exactly.

### Worked Examples

Three processes, initial clocks all 0:

- $P_1$: event $a$ at $C_1 = 1$, sends to $P_2$
- $P_2$: receives at $C_2 = \max(0, 1) + 1 = 2$, event $b$ at $C_2 = 3$, sends to $P_3$
- $P_3$: local event $c$ at $C_3 = 1$, receives from $P_2$ at $C_3 = \max(1, 3) + 1 = 4$

Total order: $a(1) < b(3) < c_{\text{recv}}(4)$. Note $c(1)$ and $a(1)$ have same timestamp but are concurrent — Lamport clocks cannot distinguish them.

---

## 4. Dining Philosophers (Deadlock Analysis)

### The Problem

Five philosophers sit at a round table, each needing two forks (shared with neighbors) to eat. What conditions produce deadlock?

### The Formula

Coffman's four necessary conditions for deadlock:

1. **Mutual exclusion**: Resources cannot be shared
2. **Hold and wait**: A process holds one resource while waiting for another
3. **No preemption**: Resources cannot be forcibly taken
4. **Circular wait**: A cycle exists in the resource allocation graph

For $n$ philosophers and $n$ forks arranged in a ring, the wait-for graph $G = (V, E)$ has a cycle of length $n$ when each philosopher holds their left fork and waits for the right:

$$P_i \rightarrow F_{(i+1) \bmod n} \rightarrow P_{(i+1) \bmod n} \rightarrow \ldots \rightarrow P_i$$

**Solution**: Break circular wait by ordering resources. Philosopher $P_{n-1}$ picks up the right fork first (lower-numbered fork), breaking the cycle.

### Worked Examples

With $n = 5$ philosophers: deadlock probability when each picks up left fork with probability 1 in lock-step is exactly 1 (deterministic deadlock). With randomized delays, expected time to deadlock grows, but it remains reachable. The resource ordering solution reduces the maximum cycle length to 0, making deadlock impossible.

---

## 5. The ABA Problem (Lock-Free Hazards)

### The Problem

Compare-and-swap (CAS) operations check if a value equals an expected value before updating. But what if the value changed from A to B and back to A? CAS sees A and succeeds, missing the intermediate modification.

### The Formula

CAS operation: $\text{CAS}(\text{addr}, \text{expected}, \text{new})$ atomically:

$$\text{CAS}(x, A, C) = \begin{cases} x := C, \text{return true} & \text{if } x = A \\ \text{return false} & \text{otherwise} \end{cases}$$

ABA sequence:
1. Thread 1 reads $x = A$
2. Thread 2 changes $x: A \rightarrow B \rightarrow A$
3. Thread 1 performs $\text{CAS}(x, A, C)$ — succeeds incorrectly

**Solutions**: Tagged pointers add a monotonic counter: $\text{CAS2}(\langle x, \text{tag} \rangle, \langle A, t \rangle, \langle C, t+1 \rangle)$. The tag never repeats (until overflow), so changed-and-restored values are detected.

### Worked Examples

Lock-free stack pop with ABA: stack is $[A, B, C]$. Thread 1 reads top $= A$, next $= B$. Thread 2 pops $A$, pops $B$, pushes $A$ back. Stack is now $[A, C]$. Thread 1's CAS succeeds setting top $= B$, but $B$ was freed — dangling pointer.

With tagged pointer: Thread 1 reads $\langle A, 0 \rangle$. After Thread 2's operations, top is $\langle A, 3 \rangle$. CAS comparing $\langle A, 0 \rangle$ fails correctly.

---

## 6. Lock-Free Queue Mathematics (Michael-Scott Queue)

### The Problem

Design a FIFO queue supporting concurrent enqueue/dequeue without locks, ensuring linearizability.

### The Formula

The Michael-Scott lock-free queue uses a linked list with head and tail pointers. Linearization points:

- **Enqueue**: The CAS that links a new node to the tail's next pointer
- **Dequeue**: The CAS that swings the head pointer forward

Correctness invariants:
1. $\text{head}$ always points to a sentinel node
2. $\text{tail}$ points to the last or second-to-last node
3. The linked list from head to tail contains all enqueued values in FIFO order

Amortized operation cost: $O(1)$ per operation. Expected number of CAS retries under contention from $k$ threads:

$$E[\text{retries}] = \sum_{i=1}^{k-1} \left(\frac{k-1}{k}\right)^i \approx k - 1$$

### Worked Examples

With 4 threads doing concurrent enqueues, expected retries per operation $\approx 3$. Each retry involves a failed CAS (cheap on modern hardware: ~10ns) versus a mutex lock/unlock (~25ns with contention). Lock-free wins when contention is moderate and operations are short.

---

## 7. Amdahl's Law (Parallel Speedup Limits)

### The Problem

Given a program where fraction $p$ can be parallelized, what is the maximum speedup with $n$ processors?

### The Formula

$$S(n) = \frac{1}{(1 - p) + \frac{p}{n}}$$

As $n \rightarrow \infty$:

$$S_{\max} = \lim_{n \to \infty} S(n) = \frac{1}{1 - p}$$

**Gustafson's Law** (1988) provides the alternative view for scaled problems:

$$S(n) = n - (1 - p)(n - 1) = 1 + p(n - 1)$$

This assumes the problem size scales with the number of processors.

### Worked Examples

Program is 95% parallelizable ($p = 0.95$):

| Processors $n$ | Speedup $S(n)$ | Efficiency $S(n)/n$ |
|---|---|---|
| 2 | 1.90x | 95.2% |
| 4 | 3.48x | 86.9% |
| 8 | 5.93x | 74.1% |
| 16 | 9.14x | 57.1% |
| 64 | 15.42x | 24.1% |
| 1024 | 19.28x | 1.9% |
| $\infty$ | 20.00x | 0% |

Maximum speedup is $1/(1-0.95) = 20\times$. Even with infinite processors, the 5% serial portion is the bottleneck. To achieve 100x speedup, you need $p \geq 0.99$.

---

## Prerequisites

- Basic understanding of threads and processes
- Familiarity with mutual exclusion and critical sections
- Understanding of atomic operations (CAS, load/store)
- Basic graph theory (cycles, partial orders)

## Complexity

| Concept | Space | Time | Key Constraint |
|---|---|---|---|
| Lamport Clock | $O(1)$ per process | $O(1)$ per event | Cannot detect concurrency |
| Vector Clock | $O(n)$ per process | $O(n)$ per event | Detects concurrency exactly |
| Lock-free queue (enqueue) | $O(1)$ amortized | $O(k)$ expected retries | $k$ = contending threads |
| Amdahl's speedup | N/A | $O(1/(1-p))$ limit | Serial fraction $1-p$ |
| Deadlock detection | $O(V + E)$ | $O(V + E)$ cycle detection | Resource allocation graph |
