# The Mathematics of Concurrency -- Process Calculi, Synchronization, Correctness, and Memory Models

> *Concurrency theory provides the formal tools to reason about systems where multiple computations execute simultaneously and interact -- where the interleaving of operations introduces nondeterminism, and correctness demands rigorous proof rather than hopeful testing.*

---

## 1. CSP Formal Semantics (Traces, Failures, Divergences)

### The Problem

Define a compositional semantics for communicating processes that supports reasoning about safety (nothing bad happens), liveness (something good eventually happens), and refinement (one process correctly implements another).

### The Formula

Let $\Sigma$ be the set of all visible events. A process $P$ in CSP is characterized by three semantic models of increasing discriminating power.

**Traces Model.** The trace set $\text{traces}(P) \subseteq \Sigma^*$ is the set of all finite sequences of visible events that $P$ can perform. It must satisfy:

- $\langle\rangle \in \text{traces}(P)$ (the empty trace is always possible)
- If $s \frown t \in \text{traces}(P)$ then $s \in \text{traces}(P)$ (prefix-closure)

where $s \frown t$ denotes the concatenation of sequences $s$ and $t$.

Refinement in the traces model: $P \sqsubseteq_T Q$ iff $\text{traces}(Q) \subseteq \text{traces}(P)$. A more refined process has fewer behaviors -- it rules out more traces.

**Failures Model.** A failure is a pair $(s, X)$ where $s \in \Sigma^*$ is a trace and $X \subseteq \Sigma$ is a refusal set -- a set of events the process can refuse after performing $s$.

$$\text{failures}(P) \subseteq \Sigma^* \times \mathcal{P}(\Sigma)$$

Axioms:
1. $(s, X) \in \text{failures}(P) \Rightarrow s \in \text{traces}(P)$
2. $(s, X) \in \text{failures}(P) \wedge Y \subseteq X \Rightarrow (s, Y) \in \text{failures}(P)$ (subset-closure of refusals)

Deadlock freedom: $P$ is deadlock-free iff for all $s \in \text{traces}(P)$, $(s, \Sigma) \notin \text{failures}(P)$. That is, $P$ can never refuse everything.

**Failures-Divergences Model.** The divergence set $\text{divergences}(P) \subseteq \Sigma^*$ records traces after which $P$ may perform an infinite sequence of internal ($\tau$) actions.

Axioms for divergences:
1. $s \in \text{divergences}(P) \wedge t \in \Sigma^* \Rightarrow s \frown t \in \text{divergences}(P)$ (extension-closure)
2. $s \in \text{divergences}(P) \Rightarrow (s, X) \in \text{failures}(P)$ for all $X$ (a divergent process can refuse anything)

The full refinement ordering:

$$P \sqsubseteq_{FD} Q \iff \text{failures}(Q) \subseteq \text{failures}(P) \wedge \text{divergences}(Q) \subseteq \text{divergences}(P)$$

### Worked Example

Consider $P = (a \to b \to \text{STOP}) \sqcap (a \to c \to \text{STOP})$ where $\sqcap$ is internal choice.

Traces: $\text{traces}(P) = \{\langle\rangle, \langle a \rangle, \langle a, b \rangle, \langle a, c \rangle\}$

After trace $\langle a \rangle$, the choice has already been resolved internally. If the left branch was chosen, the process can refuse $\{c\}$ but not $\{b\}$. If the right branch was chosen, it can refuse $\{b\}$ but not $\{c\}$. Since external observation cannot determine which branch was taken:

$$(\langle a \rangle, \{b\}) \in \text{failures}(P) \quad \text{and} \quad (\langle a \rangle, \{c\}) \in \text{failures}(P)$$

But $(\langle a \rangle, \{b, c\}) \notin \text{failures}(P)$ because in every resolution at least one of $b, c$ is offered.

### The Intuition

The traces model sees only what a process does. The failures model also sees what a process refuses to do. The divergences model additionally detects livelock. Each successive model draws finer distinctions between processes, enabling stronger correctness guarantees.

---

## 2. Pi-Calculus: Syntax and Reduction

### The Problem

Model systems where the communication topology itself changes over time -- processes can learn new channel names by receiving them as messages, enabling dynamic reconfiguration.

### The Formula

The syntax of the monadic pi-calculus:

$$P, Q ::= 0 \mid \bar{x}\langle y \rangle.P \mid x(z).P \mid P \mid Q \mid (\nu x)P \mid !P \mid P + Q$$

where $\bar{x}\langle y \rangle.P$ sends name $y$ on channel $x$, $x(z).P$ receives a name on channel $x$ binding it to $z$, $(\nu x)P$ creates a fresh name $x$ private to $P$, and $!P$ denotes replication.

**Structural congruence** $\equiv$ is the smallest congruence satisfying:

$$P \mid 0 \equiv P \qquad P \mid Q \equiv Q \mid P \qquad (P \mid Q) \mid R \equiv P \mid (Q \mid R)$$
$$(\nu x)0 \equiv 0 \qquad (\nu x)(\nu y)P \equiv (\nu y)(\nu x)P$$
$$(\nu x)(P \mid Q) \equiv P \mid (\nu x)Q \quad \text{if } x \notin \text{fn}(P) \quad \text{(scope extrusion)}$$
$$!P \equiv P \mid\; !P$$

**Reduction rules:**

Communication (the only true computation step):
$$\bar{x}\langle y \rangle.P \mid x(z).Q \to P \mid Q[y/z]$$

where $Q[y/z]$ is the capture-avoiding substitution of $y$ for $z$ in $Q$.

Contextual closure:
$$\frac{P \to P'}{P \mid Q \to P' \mid Q} \qquad \frac{P \to P'}{(\nu x)P \to (\nu x)P'} \qquad \frac{P \equiv P' \quad P' \to Q' \quad Q' \equiv Q}{P \to Q}$$

### Worked Example

Model a mobile phone handoff. Let $\text{base}_1$ and $\text{base}_2$ be base stations, and $\text{phone}$ a mobile device connected to $\text{base}_1$:

$$\text{System} = (\nu \text{talk})(\text{phone}(\text{talk}) \mid \text{base}_1(\text{talk}) \mid \text{base}_2)$$

Handoff: $\text{base}_1$ sends the private channel $\text{talk}$ to $\text{base}_2$. After scope extrusion, $\text{base}_2$ can communicate on $\text{talk}$, and $\text{base}_1$ drops it. The communication topology has changed -- $\text{phone}$ now speaks to $\text{base}_2$ -- all expressed within the calculus.

### The Intuition

The pi-calculus captures mobility: the ability of processes to change their interconnection structure at runtime. By making channel names first-class values that can be communicated, it models the dynamic topologies found in real distributed systems, mobile code, and biological processes.

---

## 3. Bisimulation Equivalence

### The Problem

Define when two concurrent processes are "the same" in a way that respects their branching behavior, not just their input-output traces.

### The Formula

A **strong bisimulation** is a binary relation $R$ on processes such that whenever $(P, Q) \in R$:

1. If $P \xrightarrow{\alpha} P'$ then there exists $Q'$ such that $Q \xrightarrow{\alpha} Q'$ and $(P', Q') \in R$
2. If $Q \xrightarrow{\alpha} Q'$ then there exists $P'$ such that $P \xrightarrow{\alpha} P'$ and $(P', Q') \in R$

Two processes are **bisimilar**, written $P \sim Q$, iff there exists a strong bisimulation $R$ with $(P, Q) \in R$.

**Weak bisimulation** abstracts over internal $\tau$ actions. Define $\Rightarrow$ as the reflexive transitive closure of $\xrightarrow{\tau}$, and $\xRightarrow{\alpha}$ as $\Rightarrow \xrightarrow{\alpha} \Rightarrow$ for visible $\alpha$, or just $\Rightarrow$ for $\alpha = \tau$.

A **weak bisimulation** $R$ requires: if $P \xrightarrow{\alpha} P'$ then $Q \xRightarrow{\alpha} Q'$ with $(P', Q') \in R$, and symmetrically.

**Bisimulation vs. trace equivalence.** Let:

$$P = a.(b.0 + c.0) \qquad Q = a.b.0 + a.c.0$$

Both have the same trace set $\{\langle\rangle, \langle a \rangle, \langle a,b \rangle, \langle a,c \rangle\}$. But $P \not\sim Q$: after $P$ performs $a$, it reaches a state offering both $b$ and $c$. After $Q$ performs $a$, it reaches a state offering only $b$ or only $c$ (depending on which summand was chosen). The bisimulation game detects this difference.

### The Intuition

Trace equivalence says "the same things can happen." Bisimulation says "the same things can happen, and at each step, the same choices are available." This makes bisimulation the right equivalence for processes that interact with an environment that can observe and influence branching decisions.

---

## 4. Petri Net Reachability Analysis

### The Problem

Given a Petri net with initial marking $M_0$ and a target marking $M$, determine whether $M$ is reachable from $M_0$ via some firing sequence.

### The Formula

A Petri net $N = (P, T, F, M_0)$ has its dynamics captured by the **incidence matrix** $C \in \mathbb{Z}^{|P| \times |T|}$ where:

$$C[p, t] = F(t, p) - F(p, t)$$

The **state equation**: if $M$ is reachable from $M_0$ via firing sequence $\sigma$, and $\vec{\sigma} \in \mathbb{N}^{|T|}$ is the Parikh vector (firing count of each transition), then:

$$M = M_0 + C \cdot \vec{\sigma}$$

This is a necessary but not sufficient condition for reachability (it ignores the ordering of firings and the enabledness constraint).

**Reachability is decidable** but the problem is non-elementary in complexity. Mayr (1981) and Kosaraju (1982) proved decidability; the exact complexity was shown to be Ackermann-complete by Leroux and Schmitz (2019).

**Coverability** (is there a reachable marking $M' \geq M$?) is EXPSPACE-complete and solved by the Karp-Miller tree construction, which uses a symbol $\omega$ to represent unbounded place counts.

### Worked Example

A mutual exclusion net with places $\{s_1, s_2, \text{cs}_1, \text{cs}_2, \text{mutex}\}$, transitions $\{t_1^{\text{enter}}, t_1^{\text{exit}}, t_2^{\text{enter}}, t_2^{\text{exit}}\}$, and $M_0 = (1, 1, 0, 0, 1)$.

$t_1^{\text{enter}}$ consumes from $s_1$ and $\text{mutex}$, produces to $\text{cs}_1$. Then the marking $(0, 1, 1, 0, 0)$ is reached. Since $\text{mutex}$ is empty, $t_2^{\text{enter}}$ is not enabled. The state equation confirms that $M = (0, 0, 1, 1, ?)$ requires $\vec{\sigma}$ satisfying $? = 1 - 1 - 1 = -1 < 0$, which is infeasible. Mutual exclusion is guaranteed: the marking $(\_, \_, 1, 1, \_)$ is unreachable.

### The Intuition

The incidence matrix linearizes the Petri net dynamics, turning reachability into an integer programming question. The linear relaxation gives a fast necessary check, but the integrality and ordering constraints make the full problem extraordinarily hard.

---

## 5. Lamport's Bakery Algorithm: Correctness Proof

### The Problem

Prove that the bakery algorithm achieves mutual exclusion for $N$ processes using only regular (non-atomic) read/write registers.

### The Formula

**Mutual Exclusion.** Suppose processes $i$ and $j$ are both in the critical section. Then $\text{number}[i] \neq 0$ and $\text{number}[j] \neq 0$. Process $i$ passed the inner loop for $j$, meaning:

$$(\text{number}[j], j) \geq (\text{number}[i], i) \quad \text{at the time } i \text{ read } \text{number}[j]$$

Symmetrically, process $j$ passed the inner loop for $i$:

$$(\text{number}[i], i) \geq (\text{number}[j], j) \quad \text{at the time } j \text{ read } \text{number}[i]$$

Both inequalities together imply $(\text{number}[i], i) = (\text{number}[j], j)$, which requires $i = j$. Contradiction.

**The choosing flag.** The subtlety is that reads and writes to $\text{number}[j]$ are not atomic. While process $j$ is choosing (computing its number), process $i$ must wait. The $\text{choosing}[j]$ flag ensures that $i$ does not read a partially written $\text{number}[j]$. Formally:

Let $W_j$ be the write of $\text{number}[j]$ by process $j$, and $R_{ij}$ be the read of $\text{number}[j]$ by process $i$ in the inner loop. The protocol ensures:

$$W_j \to R_{ij}$$

in the happens-before order, because $i$ waits until $\text{choosing}[j] = \text{false}$, and $j$ sets $\text{choosing}[j] = \text{false}$ only after completing $W_j$.

**FCFS (First-Come-First-Served).** If process $i$ enters the doorway (sets $\text{choosing}[i] = \text{true}$) before process $j$, then $\text{number}[i] \leq \text{number}[j]$ (since $j$ reads $\text{number}[i]$ when computing its own number). Therefore $(number[i], i) < (number[j], j)$ (as $i$'s number is no larger), and $i$ enters the CS first.

### The Intuition

The bakery algorithm simulates a ticket-based queue using only regular registers. The choosing flag is the critical mechanism: it creates a synchronization barrier that compensates for the non-atomicity of multi-word writes. The algorithm demonstrates that mutual exclusion does not require hardware atomicity beyond single-bit reads and writes.

---

## 6. Linearizability: Definition and Examples

### The Problem

Define a correctness condition for concurrent data structures that is local (composable), nonblocking, and provides the illusion that each operation takes effect instantaneously.

### The Formula

A **history** $H$ is a sequence of invocation and response events. An operation in $H$ is a matched invocation-response pair. Operation $\text{op}_1$ **precedes** $\text{op}_2$ in $H$ (written $\text{op}_1 <_H \text{op}_2$) iff the response of $\text{op}_1$ occurs before the invocation of $\text{op}_2$.

$H$ is **linearizable** with respect to a sequential specification $S$ iff there exists a total order $\prec$ on the operations of $H$ such that:

1. **Legal:** the sequential history induced by $\prec$ is in $S$
2. **Consistent with real-time:** if $\text{op}_1 <_H \text{op}_2$ then $\text{op}_1 \prec \text{op}_2$

The **linearization point** of each operation is the instant within its invocation-response interval at which it "takes effect."

**Locality theorem** (Herlihy & Wing): $H$ is linearizable iff for each object $x$, the sub-history $H|x$ (projected onto $x$) is linearizable. This means linearizability is compositional -- objects can be verified independently.

### Worked Example

Consider a concurrent queue with the following history on a FIFO queue object:

```
Thread A:  enq(1) --------|
Thread B:       enq(2) ---|---------|
Thread A:                     deq() -> 1
Thread B:                               deq() -> 2
```

The operations $\text{enq}(1)$ and $\text{enq}(2)$ overlap. A valid linearization order is $\text{enq}(1) \prec \text{enq}(2) \prec \text{deq}() \to 1 \prec \text{deq}() \to 2$, which matches FIFO specification. This history is linearizable.

If instead $\text{deq}()$ by Thread A returned 2, we would need $\text{enq}(2) \prec \text{enq}(1)$, but then Thread B's $\text{deq}() \to 2$ would violate FIFO (1 should be dequeued before 2 in the remaining queue). No valid linearization exists: the history would not be linearizable.

### The Intuition

Linearizability bridges the gap between concurrent and sequential reasoning. By requiring each operation to appear atomic at some point during its execution, it lets programmers think sequentially about each operation while the implementation runs concurrently. The locality property makes it practical: verify each data structure in isolation.

---

## 7. Impossibility of Wait-Free Consensus (FLP Result)

### The Problem

Prove that in an asynchronous distributed system with even one potentially faulty (crash-stop) process, there is no deterministic protocol that solves consensus.

### The Formula

**Consensus** requires three properties:
- **Agreement:** all correct processes decide the same value
- **Validity:** the decision value was proposed by some process
- **Termination:** every correct process eventually decides

**Theorem (Fischer, Lynch, Paterson, 1985).** No deterministic algorithm solves consensus in an asynchronous message-passing system if even one process may crash.

**Proof structure.** The proof proceeds in two steps:

**Step 1: Bivalent initial configuration exists.** A configuration is **0-valent** (resp. **1-valent**) if all reachable decisions are 0 (resp. 1). It is **bivalent** if both decisions are reachable. By a valency argument over input vectors: consider the sequence of initial configurations $C_0, C_1, \ldots, C_n$ where $C_i$ differs from $C_{i-1}$ only in process $i$'s input. Since $C_0$ is 0-valent (all inputs 0) and $C_n$ is 1-valent (all inputs 1), there exist adjacent $C_i, C_{i+1}$ with different valencies. If process $i+1$ crashes before taking any step, the remaining processes cannot distinguish $C_i$ from $C_{i+1}$, so at least one must be bivalent.

**Step 2: Bivalent configurations persist.** From any bivalent configuration $C$, consider a pending event $e$ (a message delivery). Let $\mathcal{C}$ be the set of configurations reachable from $C$ without applying $e$, and $\mathcal{D} = e(\mathcal{C})$ (applying $e$ to each). Suppose for contradiction that $\mathcal{D}$ contains no bivalent configuration. Then every $D \in \mathcal{D}$ is univalent. There must exist $D_0$ that is 0-valent and $D_1$ that is 1-valent (otherwise $C$ would not be bivalent). There exist $C_0, C_1 \in \mathcal{C}$ such that $e(C_0) = D_0$ and $e(C_1) = D_1$, where $C_1$ is obtained from $C_0$ by applying a single event $e'$.

Case analysis on $e$ and $e'$:
- If $e$ and $e'$ involve different processes, then $e(e'(C_0)) = e'(e(C_0))$ (commutativity). But $e(e'(C_0)) = e(C_1) = D_1$ (1-valent), and $e'(e(C_0)) = e'(D_0)$ is reachable from $D_0$ (0-valent). Contradiction.
- If $e$ and $e'$ involve the same process $p$, let $\sigma$ be any deciding run from $C_0$ in which $p$ takes no steps (simulating $p$'s crash). Then $\sigma(D_0)$ decides 0 and $\sigma(D_1)$ decides 1. But $\sigma$ cannot distinguish $D_0$ from $D_1$ since both differ only in $p$'s state. Contradiction.

Therefore bivalent configurations are inescapable: the adversarial scheduler can always delay messages to prevent the system from reaching a decision. $\square$

### The Intuition

The FLP result shows that asynchrony and fault tolerance are fundamentally at odds with deterministic agreement. The adversary exploits the inability to distinguish a slow process from a crashed one. This impossibility is why practical consensus protocols (Paxos, Raft) either assume partial synchrony, use randomization, or weaken termination guarantees.

---

## 8. Memory Model Formalization

### The Problem

Specify precisely which values a read may return in a shared-memory concurrent program, accounting for hardware reorderings, compiler optimizations, and caching.

### The Formula

**Sequential Consistency (SC).** Lamport (1979) defined SC as: "the result of any execution is the same as if the operations of all processors were executed in some sequential order, and the operations of each individual processor appear in this sequence in the order specified by its program."

Formally, an execution $(E, <_{\text{po}}, <_{\text{rf}})$ (where $<_{\text{po}}$ is program order and $<_{\text{rf}}$ is the reads-from relation) satisfies SC iff there exists a total order $<_{\text{mo}}$ (memory order) on all memory operations such that:

1. $<_{\text{mo}}$ is consistent with $<_{\text{po}}$ (program order is preserved)
2. Each read $r$ of location $x$ returns the value written by the most recent write to $x$ in $<_{\text{mo}}$

**Total Store Order (TSO).** TSO relaxes SC by allowing each processor to have a FIFO store buffer. A processor's load may read from its own store buffer before the store is globally visible. Formally, TSO = SC except:

- A load may be reordered before an earlier store to a *different* address (store-load reordering)
- A load to address $x$ returns the value of the most recent store to $x$ in the local store buffer if present, otherwise the most recent store to $x$ in the global memory order

The store buffer can be modeled as: each processor $p$ has a buffer $B_p = [(x_1, v_1), (x_2, v_2), \ldots]$. A store $(x, v)$ appends to $B_p$. A load of $x$ returns $v$ from the most recent $(x, v) \in B_p$, or consults global memory if no buffered store to $x$ exists. MFENCE drains the store buffer.

**Litmus test distinguishing SC from TSO:**

```
Initially: x = 0, y = 0

Thread 1:       Thread 2:
  x = 1           y = 1
  r1 = y          r2 = x

Can r1 = 0 and r2 = 0?
  SC:  No  (at least one store must be ordered first)
  TSO: Yes (both stores buffered, both loads see 0)
```

**Relaxed models (ARM, POWER).** These allow:
- Store-store reordering (stores to different addresses may be observed out of order)
- Load-load reordering
- Load-store reordering
- No multi-copy atomicity (different processors may observe stores in different orders)

The formal framework uses **candidate executions** with relations: program order ($\text{po}$), reads-from ($\text{rf}$), coherence order ($\text{co}$), from-reads ($\text{fr} = \text{rf}^{-1}; \text{co}$). An execution is consistent if certain cycles are forbidden in the union of these relations.

**C/C++11 Memory Model.** Each atomic operation carries a memory order tag. The model defines a happens-before relation $\text{hb}$ built from:

$$\text{hb} = (\text{po} \cup \text{sw})^+$$

where $\text{sw}$ (synchronizes-with) connects a release store to an acquire load that reads from it. A non-atomic read of $x$ must read from a write $w$ such that $w \xrightarrow{\text{hb}} r$ and there is no intervening write $w'$ with $w \xrightarrow{\text{hb}} w' \xrightarrow{\text{hb}} r$. Data races on non-atomics yield undefined behavior.

### The Intuition

Memory models are contracts between hardware/compiler and programmer. Stronger models (SC) are easier to reason about but constrain optimization. Weaker models (TSO, relaxed) permit more hardware and compiler freedom but require the programmer to insert explicit fences. The formalization as constraints on execution graphs transforms an operational question ("what might my program do?") into a declarative one ("which execution graphs are valid?").

---

## References

- Hoare, C.A.R. *Communicating Sequential Processes.* Prentice Hall, 1985.
- Roscoe, A.W. *The Theory and Practice of Concurrency.* Prentice Hall, 1997.
- Milner, R. *Communicating and Mobile Systems: The Pi-Calculus.* Cambridge, 1999.
- Milner, R. *Communication and Concurrency.* Prentice Hall, 1989.
- Sangiorgi, D. and Walker, D. *The Pi-Calculus: A Theory of Mobile Processes.* Cambridge, 2001.
- Hewitt, C. et al. "A Universal Modular ACTOR Formalism for Artificial Intelligence." IJCAI, 1973.
- Petri, C.A. "Kommunikation mit Automaten." PhD thesis, 1962.
- Murata, T. "Petri Nets: Properties, Analysis and Applications." Proceedings of the IEEE, 1989.
- Leroux, J. and Schmitz, S. "Reachability in Vector Addition Systems is Ackermann-complete." FOCS, 2019.
- Lamport, L. "A New Solution of Dijkstra's Concurrent Programming Problem." CACM, 1974.
- Lamport, L. "Time, Clocks, and the Ordering of Events in a Distributed System." CACM, 1978.
- Lamport, L. "How to Make a Multiprocessor Computer That Correctly Executes Multiprocess Programs." IEEE TC, 1979.
- Dijkstra, E.W. "Solution of a Problem in Concurrent Programming Control." CACM, 1965.
- Herlihy, M. and Wing, J. "Linearizability: A Correctness Condition for Concurrent Objects." TOPLAS, 1990.
- Herlihy, M. "Wait-Free Synchronization." TOPLAS, 1991.
- Fischer, M., Lynch, N., and Paterson, M. "Impossibility of Distributed Consensus with One Faulty Process." JACM, 1985.
- Coffman, E.G. et al. "System Deadlocks." Computing Surveys, 1971.
- Adve, S. and Gharachorloo, K. "Shared Memory Consistency Models: A Tutorial." IEEE Computer, 1996.
- Batty, M. et al. "Mathematizing C++ Concurrency." POPL, 2011.
