# CAP Theorem and Consistency Models -- Proofs, Formalizations, and Convergence Theory

> *The CAP theorem, once a folk conjecture, was formalized by Gilbert and Lynch into a precise impossibility result for asynchronous systems. The consistency models it references -- from linearizability to eventual consistency -- form a rich lattice of guarantees, each with distinct formal definitions, implementability constraints, and composability properties.*

---

## 1. The Gilbert-Lynch CAP Proof

### The Problem

Prove that no distributed system in an asynchronous network can simultaneously guarantee consistency (linearizability), availability (every non-failing node responds), and partition tolerance (correct operation despite arbitrary message loss).

### System Model

We consider an asynchronous message-passing network with $n \geq 2$ nodes. There is no shared clock. Messages may be delayed arbitrarily or lost entirely. A **partition** is a division of nodes into two non-empty sets $S_1, S_2$ such that all messages between $S_1$ and $S_2$ are lost.

Define:

- **Consistency (Atomic/Linearizable):** There exists a total order on all operations such that (1) the order is consistent with real-time precedence, and (2) every read returns the value of the most recent preceding write in this order.
- **Availability:** Every request to a non-failing node receives a non-error response (no timeout, no refusal).
- **Partition Tolerance:** The system continues to satisfy its consistency and availability guarantees even when the network is partitioned.

### The Proof

**Theorem (Gilbert-Lynch, 2002).** It is impossible for a distributed system in an asynchronous network to simultaneously provide consistency, availability, and partition tolerance.

*Proof.* By contradiction. Assume a system $\mathcal{S}$ provides all three properties. Consider two nodes $n_1$ and $n_2$ separated by a partition. The system stores a register $x$ with initial value $v_0$.

**Step 1.** A client sends a write request $w(x, v_1)$ to $n_1$. By availability, $n_1$ must acknowledge the write. Since $n_1$ and $n_2$ are partitioned, $n_1$ cannot communicate the update to $n_2$.

**Step 2.** A (possibly different) client sends a read request $r(x)$ to $n_2$. By availability, $n_2$ must return a value.

**Step 3.** By consistency (linearizability), the read must return $v_1$, since the write $w(x, v_1)$ has completed and must precede any subsequent read in the linearization order. But $n_2$ has not received any message about the write (partition), so $n_2$ can only return $v_0$.

**Step 4.** Contradiction: $n_2$ returns $v_0 \neq v_1$, violating consistency. Therefore $\mathcal{S}$ cannot exist. $\blacksquare$

### Scope and Limitations

The theorem assumes:

1. **Asynchronous model:** No bounds on message delay. In partially synchronous or synchronous models, CAP does not apply in the same form.
2. **Linearizability as "C":** Weaker consistency models (causal, eventual) may be achievable alongside A and P.
3. **Total partitions:** The proof uses a complete partition. In practice, partial partitions create more nuanced tradeoffs.

The theorem is an **impossibility result**, not a design prescription. It tells us which guarantees we must relax, not which to choose.

---

## 2. Linearizability: Formal Definition

### Histories and Specifications

A **history** $H$ is a finite sequence of invocation and response events. Each operation $op$ consists of an invocation $\text{inv}(op)$ and a response $\text{res}(op)$. An operation is **complete** if both events appear; **pending** if only the invocation appears.

A **sequential history** $S$ is a history where each invocation is immediately followed by its matching response (no interleaving).

A **sequential specification** for an object is the set of all legal sequential histories of that object. For a read/write register:

$$\text{Spec}(\text{Register}) = \{ S : \text{every read in } S \text{ returns the value of the most recent preceding write} \}$$

### The Definition (Herlihy-Wing, 1990)

A history $H$ is **linearizable** if there exists a sequential history $S$ such that:

1. **Legal:** $S$ is in the sequential specification of the object.
2. **Extension:** $S$ is obtained from a completion of $H$ (complete all pending operations or remove them).
3. **Preservation of real-time order:** If $\text{res}(op_1)$ precedes $\text{inv}(op_2)$ in $H$, then $op_1$ precedes $op_2$ in $S$.

The sequential history $S$ is called a **linearization** of $H$. The point in time at which each operation "takes effect" in $S$ is its **linearization point**, which must lie between its invocation and response.

Formally, define the real-time partial order $<_H$ on operations:

$$op_1 <_H op_2 \iff \text{res}(op_1) \text{ precedes } \text{inv}(op_2) \text{ in } H$$

Then $H$ is linearizable if and only if $<_H$ can be extended to a total order consistent with the sequential specification.

### Composability

**Theorem (Herlihy-Wing).** Linearizability is **compositional** (also called **local**): a history $H$ is linearizable if and only if, for each object $x$, the subhistory $H|x$ (restricted to operations on $x$) is linearizable.

This is a crucial property. It means we can reason about each object independently. Sequential consistency does NOT have this property.

---

## 3. Sequential Consistency vs Linearizability

### Sequential Consistency (Lamport, 1979)

A history $H$ is **sequentially consistent** if there exists a sequential history $S$ such that:

1. $S$ is legal (in the sequential specification).
2. $S$ preserves the **program order** of each process: if $op_1$ precedes $op_2$ in the same process in $H$, then $op_1$ precedes $op_2$ in $S$.

Note: there is **no real-time ordering** constraint. Operations from different processes can be reordered arbitrarily, as long as per-process order is preserved.

### Distinguishing Example

Consider two processes $P_1$ and $P_2$ operating on register $x$ (initially 0):

$$P_1: w(x, 1) \quad \text{completes at time } t_1$$
$$P_2: r(x) \to 0 \quad \text{invoked at time } t_2, \text{ where } t_2 > t_1$$

Under **linearizability**, this is illegal: $P_1$'s write completed before $P_2$'s read began, so the read must return 1.

Under **sequential consistency**, this is legal: we can construct $S = r(x) \to 0, w(x, 1)$, which reorders the operations across processes while preserving each process's internal order (trivially, since each process has one operation).

### Non-Composability of Sequential Consistency

**Claim.** Sequential consistency is not compositional.

*Proof by counterexample.* Consider two registers $x$ and $y$ (both initially 0) and two processes:

$$P_1: w(x, 1); \quad r(y) \to 0$$
$$P_2: w(y, 1); \quad r(x) \to 0$$

Restricted to $x$: the subhistory $w(x,1)$ and $r(x) \to 0$ from different processes. Sequentially consistent via $S_x = r(x) \to 0, w(x, 1)$.

Restricted to $y$: similarly, $S_y = r(y) \to 0, w(y, 1)$.

But the combined history requires: $w(x,1)$ before $r(y) \to 0$ (program order of $P_1$), meaning $w(x,1)$ before $r(y)$. And $r(y) \to 0$ means $w(y,1)$ has not happened yet, so $r(y)$ before $w(y,1)$. Similarly, $w(y,1)$ before $r(x) \to 0$ before $w(x,1)$. This gives a cycle: $w(x,1) < w(y,1) < w(x,1)$. Contradiction. The combined history is not sequentially consistent, even though each object's subhistory is. $\blacksquare$

---

## 4. Causal Consistency

### Causal Order

Define the **causal order** $\leadsto$ (also written $\rightarrow$ in Lamport's notation) on operations:

1. **Program order:** If $op_1$ precedes $op_2$ in the same process, then $op_1 \leadsto op_2$.
2. **Reads-from:** If $op_1$ is a write and $op_2$ is a read that returns the value written by $op_1$, then $op_1 \leadsto op_2$.
3. **Transitivity:** If $op_1 \leadsto op_2$ and $op_2 \leadsto op_3$, then $op_1 \leadsto op_3$.

Two operations are **concurrent** (written $op_1 \| op_2$) if neither $op_1 \leadsto op_2$ nor $op_2 \leadsto op_1$.

### Definition

A history $H$ is **causally consistent** if for each process $P_i$, there exists a legal sequential history $S_i$ that:

1. Contains all operations in $H$.
2. Respects the causal order $\leadsto$.
3. Different processes may have different $S_i$ (they may order concurrent operations differently).

### Causal Cuts and Consistent Snapshots

A **causal cut** of a distributed execution is a set of events $C$ such that if $e \in C$ and $e' \leadsto e$, then $e' \in C$. Equivalently, $C$ is downward-closed under the causal order.

A **consistent snapshot** is a causal cut. It represents a state that "could have occurred" -- no event is included without its causal predecessors.

**Theorem.** A consistent snapshot can be captured without stopping the system, using Chandy-Lamport snapshot protocol (marker-based) or vector clocks.

### Vector Clock Implementation

Each process $P_i$ maintains a vector clock $VC_i[1..n]$:

- **Local event:** $VC_i[i] \leftarrow VC_i[i] + 1$
- **Send message $m$:** attach $VC_i$ to $m$; $VC_i[i] \leftarrow VC_i[i] + 1$
- **Receive message $m$ with timestamp $T$:** $VC_i[j] \leftarrow \max(VC_i[j], T[j])$ for all $j$; then $VC_i[i] \leftarrow VC_i[i] + 1$

**Property.** $e_1 \leadsto e_2$ if and only if $VC(e_1) < VC(e_2)$ (componentwise $\leq$ with at least one strict $<$).

---

## 5. CRDT Convergence: The Semilattice Proof

### Join-Semilattice

A **join-semilattice** $(S, \sqsubseteq, \sqcup)$ is a partially ordered set where every pair of elements has a least upper bound (join):

$$\forall a, b \in S, \exists \, a \sqcup b \in S$$

The join operation satisfies:

- **Commutativity:** $a \sqcup b = b \sqcup a$
- **Associativity:** $(a \sqcup b) \sqcup c = a \sqcup (b \sqcup c)$
- **Idempotence:** $a \sqcup a = a$

### State-Based CRDT (CvRDT) Convergence

**Definition.** A CvRDT is a tuple $(S, s^0, q, u, m)$ where:

- $S$ is a join-semilattice of states
- $s^0 \in S$ is the initial state
- $q: S \to \text{Result}$ is the query function
- $u: S \times \text{Arg} \to S$ is the update function, satisfying $u(s, a) \sqsupseteq s$ (updates are inflationary)
- $m: S \times S \to S$ is the merge function, defined as $m(s_1, s_2) = s_1 \sqcup s_2$

**Theorem (Shapiro et al., 2011).** If all replicas of a CvRDT eventually receive all updates (either directly or via transitive merge), then all replicas converge to the same state.

*Proof.* Let replicas $r_1, \ldots, r_k$ have states $s_1, \ldots, s_k$ at some time. After all states have been exchanged and merged:

$$s_{\text{final}} = s_1 \sqcup s_2 \sqcup \cdots \sqcup s_k$$

By commutativity and associativity, the order of merges does not matter. By idempotence, receiving the same state multiple times has no effect. Since each update is inflationary ($u(s, a) \sqsupseteq s$), no information is lost. Therefore all replicas converge to the same $s_{\text{final}}$, and $q(s_{\text{final}})$ returns the same result at all replicas. $\blacksquare$

### G-Counter as a Semilattice

The G-Counter state space is $\mathbb{N}^n$ (vectors of natural numbers, one per node). The partial order is componentwise $\leq$:

$$(a_1, \ldots, a_n) \sqsubseteq (b_1, \ldots, b_n) \iff \forall i: a_i \leq b_i$$

The join is componentwise max:

$$(a_1, \ldots, a_n) \sqcup (b_1, \ldots, b_n) = (\max(a_1, b_1), \ldots, \max(a_n, b_n))$$

Increment at node $i$: $a_i \leftarrow a_i + 1$ (inflationary). Query: $\text{value} = \sum_i a_i$.

This forms a join-semilattice. Convergence follows from the general theorem.

---

## 6. Dynamo-Style Consistency Analysis

### Quorum Intersection Theorem

**Theorem.** In a system with $N$ replicas, if every write is acknowledged by at least $W$ replicas and every read contacts at least $R$ replicas, then:

$$R + W > N \implies \text{read and write quorum sets intersect}$$

*Proof.* The write quorum has $W$ replicas and the read quorum has $R$ replicas, drawn from the same set of $N$ replicas. If $R + W > N$, by the pigeonhole principle, at least one replica $r$ is in both quorums. Replica $r$ has received the latest write and is contacted by the read. $\blacksquare$

### Why Overlap is Necessary but Not Sufficient

Quorum overlap guarantees that the read set **contains** a replica with the latest value, but the client must **identify** which value is latest. This requires:

1. **Version vectors or timestamps** to determine recency.
2. **Read repair:** after identifying the latest value, propagate it to stale replicas.
3. **No concurrent writes:** if two writes are concurrent (neither has been fully replicated), the overlapping replica may have either value, and the system must perform conflict resolution.

**Claim.** $R + W > N$ with version vectors and read repair provides **regular register** semantics (a read concurrent with a write may return either the old or new value), but not linearizability unless writes are serialized through a coordinator.

### Sloppy Quorum Analysis

Under sloppy quorum, writes go to **any** $W$ live nodes, not necessarily the $N$ designated replicas. If a write goes to substitute node $s \notin \{r_1, \ldots, r_N\}$:

$$R + W > N \text{ still holds, but the quorum sets may not intersect}$$

This is because the read still contacts $R$ of the original $N$ replicas, while the write was acknowledged by $W$ nodes that may include substitutes outside the $N$. The quorum intersection property breaks, and consistency degrades to eventual (relying on hinted handoff for convergence).

---

## 7. Session Guarantees: Formalization

### System Model (Terry et al., 1994)

Consider a set of servers $\{S_1, \ldots, S_m\}$, each holding a replica of a database. A client session interacts with one server at a time but may switch servers (e.g., due to mobility or load balancing).

Let $\text{DB}(S_i, t)$ denote the set of writes applied at server $S_i$ by time $t$. A read at $S_i$ returns results consistent with $\text{DB}(S_i, t)$.

### The Four Guarantees

**Read Your Writes (RYW).** If session $\sigma$ performs write $w$ at server $S_j$, then a subsequent read at server $S_k$ sees $w$:

$$w \in \text{WriteSet}(\sigma, t_w) \implies w \in \text{DB}(S_k, t_r) \quad \text{for all reads at } t_r > t_w$$

**Monotonic Reads (MR).** If session $\sigma$ reads from $\text{DB}(S_j, t_1)$, then a subsequent read from $S_k$ sees at least as much:

$$\text{DB}(S_j, t_1) \subseteq \text{DB}(S_k, t_2) \quad \text{for } t_2 > t_1$$

**Monotonic Writes (MW).** If session $\sigma$ performs $w_1$ before $w_2$, then every server that applies $w_2$ has already applied $w_1$:

$$w_2 \in \text{DB}(S_k, t) \implies w_1 \in \text{DB}(S_k, t)$$

**Writes Follow Reads (WFR).** If session $\sigma$ reads from $\text{DB}(S_j, t_1)$ and then performs write $w$ at $S_k$, then any server applying $w$ has also applied all writes in $\text{DB}(S_j, t_1)$:

$$w \in \text{DB}(S_l, t) \implies \text{DB}(S_j, t_1) \subseteq \text{DB}(S_l, t)$$

### Implementation via Write Sets

Each session maintains a **read set** $RS$ (writes observed by reads) and a **write set** $WS$ (writes performed). Before issuing an operation at server $S_k$:

- **RYW:** Check $WS \subseteq \text{DB}(S_k)$. If not, wait or switch servers.
- **MR:** Check $RS \subseteq \text{DB}(S_k)$.
- **MW:** Tag each write with its causal predecessors; server applies in order.
- **WFR:** Before accepting a write, server checks $RS \subseteq \text{DB}(S_k)$.

---

## 8. The CALM Theorem

### Consistency As Logical Monotonicity (Hellerstein, 2010)

The CALM theorem connects the need for distributed coordination to a property of the computation itself.

**Definition.** A computation is **monotone** if adding new input facts never retracts previously derived output facts. Formally, if $I_1 \subseteq I_2$ (input sets), then $P(I_1) \subseteq P(I_2)$ (output sets), where $P$ is the program.

**Theorem (CALM).** A distributed program can be computed **coordination-free** (without consensus, barriers, or global synchronization) if and only if it is monotone.

Equivalently:

- Monotone programs are **eventually consistent** and **confluent** -- they converge to the correct result regardless of message ordering or delays.
- Non-monotone programs (involving negation, aggregation, or deletion) require coordination to ensure correctness.

### Connection to Datalog and CRDTs

In the Datalog stratification hierarchy:

- **Monotone Datalog** (no negation, no aggregation): coordination-free. Corresponds to the positive fragment. Examples: transitive closure, reachability, set union.
- **Stratified Datalog** (with negation): requires coordination at stratum boundaries. The negation introduces a non-monotone step that depends on having a complete view of the data.

CRDTs are a practical embodiment of CALM:

- CRDT merge functions are monotone (they can only add information to the semilattice, never retract it).
- Therefore CRDTs are coordination-free by CALM.
- Operations that require "removing" information (e.g., set removal) need careful encoding to remain monotone (e.g., tombstones in OR-Set).

### Implications for System Design

$$\text{Need coordination?} \iff \text{Computation is non-monotone}$$

This gives a **static analysis** criterion: examine the program's logic. If all operations are monotone (joins, unions, projections, selections without negation), the computation can run on an AP system without coordination. If the computation involves aggregation over a complete set, global counts, or negation, it inherently requires a coordination step.

---

## Tips

- The Gilbert-Lynch proof is deceptively simple. Its power lies in the generality of the asynchronous model: no timing assumptions means the result applies to any real network where messages can be delayed arbitrarily.
- Linearizability's composability is what makes it the "right" consistency model for concurrent object specifications. Sequential consistency's non-composability makes modular reasoning impossible.
- Vector clocks capture causal order exactly, but their size is $O(n)$ where $n$ is the number of processes. For large systems, bounded alternatives (interval tree clocks, dotted version vectors) trade precision for space.
- The CRDT semilattice proof is constructive: to design a new CRDT, define the state space as a semilattice, make updates inflationary, and merge as join. If you can do this, convergence is guaranteed.
- Sloppy quorums are an availability optimization that explicitly breaks the consistency guarantee of $R + W > N$. Use them only when availability during partitions is more valuable than read consistency.
- The CALM theorem provides a principled answer to "when do I need consensus?" If your computation is expressible as monotone Datalog, you do not need it. If you need negation or aggregation, you do.

## See Also

- database-theory
- graph-theory
- complexity-theory
- information-theory
- set-theory
- logic

## References

- Gilbert, S. & Lynch, N. "Brewer's Conjecture and the Feasibility of Consistent, Available, Partition-Tolerant Web Services" (2002), SIGACT News
- Herlihy, M. & Wing, J. "Linearizability: A Correctness Condition for Concurrent Objects" (1990), ACM TOPLAS
- Lamport, L. "How to Make a Multiprocessor Computer That Correctly Executes Multiprocess Programs" (1979), IEEE TC
- Shapiro, M., Preguica, N., Baquero, C. & Zawirski, M. "Conflict-Free Replicated Data Types" (2011), SSS
- Shapiro, M., Preguica, N., Baquero, C. & Zawirski, M. "A Comprehensive Study of Convergent and Commutative Replicated Data Types" (2011), INRIA TR 7506
- DeCandia, G. et al. "Dynamo: Amazon's Highly Available Key-Value Store" (2007), SOSP
- Terry, D. et al. "Session Guarantees for Weakly Consistent Replicated Data" (1994), PDIS
- Abadi, D. "Consistency Tradeoffs in Modern Distributed Database System Design" (2012), IEEE Computer
- Hellerstein, J. M. "The Declarative Imperative: Experiences and Conjectures in Distributed Logic" (2010), SIGMOD Record
- Alvaro, P., Conway, N., Hellerstein, J. M. & Marczak, W. R. "Consistency Analysis in Bloom: A CALM and Collected Approach" (2011), CIDR
- Mattern, F. "Virtual Time and Global States of Distributed Systems" (1988), Workshop on Parallel and Distributed Algorithms
- Chandy, K. M. & Lamport, L. "Distributed Snapshots: Determining Global States of Distributed Systems" (1985), ACM TOCS
- Attiya, H. & Welch, J. "Distributed Computing: Fundamentals, Simulations, and Advanced Topics" (2nd ed., Wiley, 2004)
- Kleppmann, M. "Designing Data-Intensive Applications" (O'Reilly, 2017), Ch. 5, 7, 9
