# The Mathematics of Distributed Systems — Impossibility, Correctness, and Convergence

> *Distributed systems operate under fundamental impossibility results that constrain what can be built. The mathematics of consensus, consistency, and convergence define the boundaries within which all practical systems must operate.*

---

## 1. FLP Impossibility Theorem (Consensus Limits)

### The Problem

Can a deterministic consensus protocol guarantee termination in an asynchronous system where even one process may crash?

### The Formula

**Theorem** (Fischer, Lynch, Paterson, 1985): No deterministic consensus protocol can guarantee agreement in an asynchronous distributed system if even one process may fail by crashing.

**Key definitions**:
- **Asynchronous**: No bound on message delay or process speed
- **Consensus**: All correct processes agree on the same value
- **Validity**: The decided value was proposed by some process
- **Termination**: Every correct process eventually decides

**Proof sketch**: The proof shows that from any **bivalent configuration** (one where both decision values 0 and 1 are still possible), there exists a valid execution that leads to another bivalent configuration. Therefore, the protocol can be kept in an undecided state indefinitely.

A configuration $C$ is **bivalent** if there exist two executions from $C$: one leading to decision 0, another to decision 1. An **initial bivalent configuration** must exist (by a partition argument on initial states).

Given any bivalent configuration and any pending message $m$, the proof constructs a new bivalent configuration reachable by delivering $m$ — the adversary always has a strategy to delay decision.

**Practical implication**: All real consensus protocols (Raft, Paxos) use one or more escape hatches:
- **Randomization**: Probability-1 termination (Ben-Or protocol)
- **Partial synchrony**: Assume eventual message delivery within bounds
- **Failure detectors**: Oracle that eventually accurately detects crashes ($\diamond S$)

### Worked Examples

Raft circumvents FLP by assuming partial synchrony: election timeouts assume messages arrive within a bounded time. If the network is truly asynchronous (messages delayed arbitrarily), Raft can fail to elect a leader. In practice, networks are "mostly synchronous" so Raft works reliably.

---

## 2. Raft Log Replication Correctness (Safety Proof)

### The Problem

How do we prove that Raft's log replication guarantees all committed entries are durable and applied in the same order across all nodes?

### The Formula

**Leader Completeness Property**: If a log entry is committed in a given term, that entry will be present in the logs of all leaders for all higher-numbered terms.

**Proof** (by contradiction): Suppose entry $e$ is committed at index $i$ in term $T$, but leader $L'$ in term $T' > T$ does not have $e$ at index $i$.

1. $e$ was replicated to a majority $S_1$ in term $T$
2. $L'$ received votes from a majority $S_2$ in term $T'$
3. By quorum intersection: $|S_1 \cap S_2| \geq 1$. Call this voter $V$.
4. $V$ has $e$ at index $i$ (member of $S_1$) and voted for $L'$ (member of $S_2$)
5. Vote granting requires $L'$'s log to be "at least as up-to-date" as $V$'s
6. "At least as up-to-date" means: higher last term, or same term with longer/equal log
7. Since $V$ has entry $e$ from term $T$, $L'$ must have an entry at index $i$ from term $\geq T$

**Log Matching Property**: If two logs contain an entry with the same index and term, then:
1. They store the same command
2. All preceding entries are identical

This follows from: (a) a leader creates at most one entry per index per term, and (b) AppendEntries consistency check ensures followers match the leader's log.

### Worked Examples

5-node cluster, entry $e$ committed at index 7, term 3. Nodes A,B,C have $e$ (majority). Node D starts election for term 4, needs 3 votes. Must get a vote from at least one of {A,B,C}. That voter has $e$ at index 7 term 3. D's log must have index 7 with term $\geq 3$ to win the vote. Therefore D has $e$ (or a later entry at that index, which by Leader Completeness was built on top of $e$).

---

## 3. Consistent Hashing Ring Analysis (Load Variance)

### The Problem

How many virtual nodes are needed to achieve acceptable load balance in consistent hashing?

### The Formula

With $n$ physical nodes and $k$ virtual nodes each, there are $N = nk$ points on the ring. Each virtual node "owns" an arc of the ring proportional to $1/N$.

The **arc length** for each virtual node follows approximately an exponential distribution with mean $1/N$. The total load on a physical node is the sum of $k$ arc lengths.

By the Central Limit Theorem, for large $k$, a physical node's total arc length is approximately normal:

$$\text{Load}_i \sim \mathcal{N}\left(\frac{1}{n}, \frac{1}{n^2 k}\right)$$

**Coefficient of variation** (relative standard deviation):

$$CV = \frac{\sigma}{\mu} = \frac{1/\sqrt{n^2 k}}{1/n} = \frac{1}{\sqrt{k}}$$

For the load to be within $\pm 10\%$ of the mean with 95% probability:

$$1.96 \cdot \frac{1}{\sqrt{k}} \leq 0.10 \implies k \geq 384$$

### Worked Examples

| Virtual nodes $k$ | CV | 95% load range |
|---|---|---|
| 10 | 31.6% | [0.38x, 1.62x] mean |
| 50 | 14.1% | [0.72x, 1.28x] mean |
| 100 | 10.0% | [0.80x, 1.20x] mean |
| 200 | 7.1% | [0.86x, 1.14x] mean |
| 500 | 4.5% | [0.91x, 1.09x] mean |

With 10 physical nodes and 150 virtual nodes each (1500 ring points), the maximum overload on any node is approximately 16% above the mean (with 95% confidence). Amazon Dynamo uses 150+ virtual nodes.

---

## 4. CRDT Merge Lattice Theory (Join-Semilattice)

### The Problem

How do we guarantee that replicas of a data structure converge regardless of the order in which updates are merged?

### The Formula

A **join-semilattice** is a partially ordered set $(S, \sqsubseteq)$ where every pair of elements has a least upper bound (join):

$$\forall a, b \in S: \exists a \sqcup b \in S$$

The join operation $\sqcup$ must satisfy:
1. **Commutativity**: $a \sqcup b = b \sqcup a$
2. **Associativity**: $(a \sqcup b) \sqcup c = a \sqcup (b \sqcup c)$
3. **Idempotency**: $a \sqcup a = a$

A **state-based CRDT** (CvRDT) defines:
- State space $S$ forming a join-semilattice
- Update operations that are inflationary: $\forall s, u: s \sqsubseteq u(s)$
- Merge function $= \sqcup$ (join)

**Convergence theorem**: If all updates are eventually delivered to all replicas, all replicas converge to the same state, regardless of delivery order.

**Proof**: By commutativity and associativity, the order of merges does not matter. By idempotency, duplicate deliveries are harmless. Since updates are inflationary, the state monotonically increases in the lattice.

### Worked Examples

**G-Counter lattice**: State = vector of natural numbers. $a \sqsubseteq b \iff \forall i: a[i] \leq b[i]$. Join: $(a \sqcup b)[i] = \max(a[i], b[i])$.

Node A: $[3, 0, 0]$, Node B: $[1, 2, 0]$, Node C: $[1, 0, 4]$.

Any merge order converges to $[3, 2, 4]$:
- $A \sqcup B = [3, 2, 0]$, then $\sqcup C = [3, 2, 4]$
- $B \sqcup C = [1, 2, 4]$, then $\sqcup A = [3, 2, 4]$

Counter value: $3 + 2 + 4 = 9$ regardless of merge order.

---

## 5. Byzantine Fault Tolerance (The 3f+1 Bound)

### The Problem

How many total nodes are needed to tolerate $f$ Byzantine (arbitrarily malicious) failures?

### The Formula

**Theorem** (Lamport, Shostak, Pease, 1982): Byzantine agreement requires at least $n \geq 3f + 1$ nodes to tolerate $f$ Byzantine failures.

**Proof sketch** (for $n = 3, f = 1$): Three generals: A, B, C. Suppose C is Byzantine.

- A sends "attack" to B and C
- C sends B a message claiming "A said retreat"
- B receives "attack" from A and "A said retreat" from C
- B cannot distinguish: is A lying or C?

With $n = 3f + 1 = 4$: A sends "attack" to B, C, D. Even if one is Byzantine, the other two honest nodes confirm A's message. Majority of $3f + 1 - f = 2f + 1$ honest nodes always outvote $f$ Byzantine nodes.

**PBFT complexity**: $O(n^2)$ message complexity per consensus round ($n$ = total nodes). For $f = 1$: need 4 nodes, $O(16)$ messages per round.

### Worked Examples

Blockchain context with $f = 1$: Need 4 validators. If validator C is malicious and sends conflicting votes, the 3 honest validators (A, B, D) form a $2/3$ supermajority and can reach consensus. With only 3 validators ($n = 3f = 3$), one Byzantine node can prevent consensus by sending different messages to each honest node.

---

## 6. Dynamo-Style Quorum Mathematics (Sloppy Quorums)

### The Problem

How do read/write quorums interact with replication to provide tunable consistency, and what happens during node failures?

### The Formula

**Strict quorum**: With $N$ replicas, read quorum $R$, write quorum $W$:

$$R + W > N \implies \text{strong consistency}$$

The overlap guarantees at least one node in the read set has the latest write.

**Probability of stale read** with $R + W \leq N$: If a write completes on $W$ of $N$ nodes and a read contacts $R$ of $N$ nodes:

$$P(\text{stale read}) = \frac{\binom{N-W}{R}}{\binom{N}{R}}$$

**Sloppy quorum** (Dynamo): During failures, writes go to "next available" nodes on the hash ring (hinted handoff). This preserves availability but breaks the quorum intersection property — reads may miss recent writes until hinted handoffs complete.

**Anti-entropy**: Merkle tree comparison between replicas detects and repairs inconsistencies. For a keyspace of $K$ keys, Merkle tree comparison is $O(\log K)$ to identify differing subtrees.

### Worked Examples

$N = 3, W = 2, R = 2$: $R + W = 4 > 3$, strong consistency.

$N = 3, W = 1, R = 1$ (fast but weak):

$$P(\text{stale read}) = \frac{\binom{2}{1}}{\binom{3}{1}} = \frac{2}{3} = 66.7\%$$

After a write to node A, reading from B or C (probability 2/3) returns stale data until async replication completes. $R = 2$ reduces this to:

$$P(\text{stale}) = \frac{\binom{2}{2}}{\binom{3}{2}} = \frac{1}{3} = 33.3\%$$

Only if both selected read nodes are the non-written ones. With $R = 2, W = 2$: $P(\text{stale}) = 0$.

---

## Prerequisites

- Partial orders, lattices, and semilattices
- Basic probability and combinatorics
- Graph theory fundamentals
- Understanding of distributed system failure models (crash, Byzantine)

## Complexity

| Result | Bound | Significance |
|---|---|---|
| FLP Impossibility | No deterministic consensus in async + 1 crash | Fundamental limit |
| Byzantine tolerance | $n \geq 3f + 1$ nodes | Minimum for $f$ malicious |
| Quorum intersection | $R + W > N$ | Strong consistency condition |
| Consistent hash balance | $CV = 1/\sqrt{k}$ | Virtual nodes for load balance |
| Gossip convergence | $O(\log N)$ rounds | Epidemic dissemination |
| PBFT messages | $O(n^2)$ per round | Byzantine consensus cost |
