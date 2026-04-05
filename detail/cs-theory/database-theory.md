# The Theoretical Foundations of Database Systems -- Relations, Dependencies, Transactions, and Optimization

> *The relational model, grounded in first-order logic and set theory, provides a mathematically rigorous framework for data management whose theoretical properties -- from normalization completeness to serializability -- remain the bedrock of modern database systems.*

---

## 1. Relational Algebra: Equivalences and Optimization Rules

### The Problem

Establish the algebraic laws governing relational operators, enabling query optimizers to transform logical plans into equivalent but more efficient forms.

### The Laws

The relational algebra forms a well-defined algebra over multisets (bags) or sets of tuples. The following equivalences hold for set semantics.

**Selection equivalences:**

$$\sigma_{\theta_1 \wedge \theta_2}(R) = \sigma_{\theta_1}(\sigma_{\theta_2}(R))$$

$$\sigma_{\theta_1}(\sigma_{\theta_2}(R)) = \sigma_{\theta_2}(\sigma_{\theta_1}(R))$$

Selection is idempotent and commutative in the cascade form.

**Selection and Cartesian product / join interaction:**

$$\sigma_\theta(R \times S) = R \bowtie_\theta S$$

If $\theta$ involves only attributes of $R$:

$$\sigma_\theta(R \bowtie S) = \sigma_\theta(R) \bowtie S$$

This is the **selection pushdown** rule, the single most important optimization transformation.

**Projection cascades:**

$$\pi_L(\pi_M(R)) = \pi_L(R) \quad \text{if } L \subseteq M$$

**Join commutativity and associativity:**

$$R \bowtie S = S \bowtie R$$

$$(R \bowtie S) \bowtie T = R \bowtie (S \bowtie T)$$

These enable join reordering, the core problem in query optimization.

**Set operation equivalences:**

$$R \cup S = S \cup R, \quad R \cap S = S \cap R$$

$$(R \cup S) \cup T = R \cup (S \cup T)$$

$$\sigma_\theta(R \cup S) = \sigma_\theta(R) \cup \sigma_\theta(S)$$

$$\sigma_\theta(R - S) = \sigma_\theta(R) - S = \sigma_\theta(R) - \sigma_\theta(S)$$

### Completeness

Codd showed that $\{\sigma, \pi, \times, \cup, -\}$ forms a **relationally complete** set: every query expressible in the safe relational calculus can be expressed using these five operations. Rename ($\rho$) is sometimes included as a sixth primitive for syntactic convenience.

---

## 2. Codd's Theorem: Relational Algebra Equals Relational Calculus

### The Problem

Prove that relational algebra and the safe relational calculus have exactly the same expressive power.

### The Theorem

**Theorem (Codd, 1972).** A query $Q$ is expressible in the relational algebra if and only if $Q$ is expressible as a safe formula in the tuple relational calculus (or equivalently, the domain relational calculus).

### Proof Sketch: Algebra to Calculus

Each algebraic operator translates directly to a calculus formula:

- $\sigma_\theta(R)$: $\{t \mid R(t) \wedge \theta(t)\}$
- $\pi_{A_1,\ldots,A_k}(R)$: $\{t' \mid \exists t\, (R(t) \wedge t'.A_1 = t.A_1 \wedge \cdots \wedge t'.A_k = t.A_k)\}$
- $R \times S$: $\{t \mid \exists r \exists s\, (R(r) \wedge S(s) \wedge t = r \circ s)\}$
- $R \cup S$: $\{t \mid R(t) \vee S(t)\}$
- $R - S$: $\{t \mid R(t) \wedge \neg S(t)\}$

### Proof Sketch: Safe Calculus to Algebra

The converse uses structural induction on the calculus formula. The key insight is that **safe** formulas guarantee finite results, and each logical connective maps to an algebraic operator:

- $\wedge$ corresponds to $\bowtie$ (or $\cap$)
- $\vee$ corresponds to $\cup$
- $\neg$ (in safe context) corresponds to $-$
- $\exists$ corresponds to $\pi$

**Safety** restricts the calculus to formulas whose free variables are bounded by relations appearing in the formula, ensuring the result is always a finite set.

### Significance

Codd's theorem establishes that SQL (based on tuple relational calculus) and relational algebra (used by query optimizers internally) compute exactly the same class of queries. This separation of concerns -- declarative specification vs. procedural execution -- is the foundation of query processing.

---

## 3. Armstrong's Axioms: Soundness and Completeness

### The Problem

Prove that Armstrong's axioms are sound (every derivable FD is logically implied) and complete (every logically implied FD is derivable).

### The Axioms

Given a set of functional dependencies $F$ over a relation schema $R$:

1. **Reflexivity:** If $Y \subseteq X$, then $X \to Y$.
2. **Augmentation:** If $X \to Y$, then $XZ \to YZ$ for any $Z$.
3. **Transitivity:** If $X \to Y$ and $Y \to Z$, then $X \to Z$.

Let $F^+$ denote the closure of $F$ (all FDs derivable from $F$ using the axioms), and let $F^*$ denote the set of all FDs logically implied by $F$.

### Soundness Proof

**Theorem.** $F^+ \subseteq F^*$.

We must show each axiom preserves logical implication.

*Reflexivity:* If $Y \subseteq X$, then any two tuples agreeing on $X$ certainly agree on $Y$. So $X \to Y$ holds in every relation satisfying $F$.

*Augmentation:* Suppose $X \to Y$ holds. Consider two tuples $t_1, t_2$ with $t_1[XZ] = t_2[XZ]$. Then $t_1[X] = t_2[X]$, so by $X \to Y$, $t_1[Y] = t_2[Y]$. Combined with $t_1[Z] = t_2[Z]$, we get $t_1[YZ] = t_2[YZ]$.

*Transitivity:* If $t_1[X] = t_2[X]$, then $t_1[Y] = t_2[Y]$ (by $X \to Y$), then $t_1[Z] = t_2[Z]$ (by $Y \to Z$).

### Completeness Proof

**Theorem.** $F^* \subseteq F^+$. Equivalently, if $X \to Y \notin F^+$, then $X \to Y \notin F^*$.

*Proof.* Suppose $X \to Y$ is not derivable from $F$. Compute $X^+$, the attribute closure of $X$ under $F$. Then $Y \not\subseteq X^+$; choose some $A \in Y \setminus X^+$.

Construct a **two-tuple relation** $r$:

$$t_1 = (1, 1, \ldots, 1) \quad \text{(all 1s)}$$
$$t_2 = \begin{cases} 1 & \text{if } B \in X^+ \\ 0 & \text{if } B \notin X^+ \end{cases}$$

We claim $r$ satisfies all FDs in $F$ but violates $X \to Y$:

- $r$ satisfies $F$: For any $V \to W \in F$, if $t_1[V] = t_2[V]$, then $V \subseteq X^+$, so by the closure algorithm $W \subseteq X^+$, hence $t_1[W] = t_2[W]$.
- $r$ violates $X \to A$: $t_1[X] = t_2[X]$ (since $X \subseteq X^+$), but $t_1[A] = 1 \neq 0 = t_2[A]$ (since $A \notin X^+$).

Therefore $X \to Y$ is not logically implied by $F$. The contrapositive gives $F^* \subseteq F^+$.

---

## 4. The Chase Algorithm for Dependency Testing

### The Problem

Determine whether a decomposition is lossless (no spurious tuples) and whether functional or multivalued dependencies are implied by a given set of dependencies.

### The Algorithm

Given relation schema $R$ with dependency set $\Sigma$ and decomposition $R_1, R_2, \ldots, R_k$:

1. **Initialize** a tableau $T$ with $k$ rows (one per $R_i$) and $|R|$ columns. For row $i$, column $A_j$: set $a_j$ (a distinguished variable) if $A_j \in R_i$, otherwise $b_{ij}$ (a nondistinguished variable).

2. **Chase step for FD $X \to Y$:** If two rows agree on all attributes of $X$ but differ on some attribute $A \in Y$, equate the values:
   - If one is distinguished ($a$), replace the other with $a$.
   - If both are nondistinguished, replace one with the other.

3. **Chase step for MVD $X \twoheadrightarrow Y$:** If rows $t_1, t_2$ agree on $X$, add (if not present) rows $t_3, t_4$ where $t_3[XY] = t_1[XY]$, $t_3[R - XY] = t_2[R - XY]$ and symmetrically.

4. **Termination:** The chase terminates when no more changes can be made (or a row becomes all distinguished).

### Lossless Join Test

The decomposition $\{R_1, \ldots, R_k\}$ is lossless under $\Sigma$ if and only if, after chasing the initial tableau with $\Sigma$, some row becomes $(a_1, a_2, \ldots, a_n)$ -- all distinguished symbols.

**Special case for two-relation decomposition:** $\{R_1, R_2\}$ is lossless under FDs $F$ iff:

$$(R_1 \cap R_2) \to R_1 \quad \text{or} \quad (R_1 \cap R_2) \to R_2$$

is in $F^+$.

### Implication Test

An FD $X \to Y$ is implied by $\Sigma$ iff chasing a two-row tableau (rows agreeing on $X$, differing elsewhere) results in the rows also agreeing on $Y$.

### Properties

- The chase always terminates for FDs (finite tableau, monotone equating of variables).
- For MVDs, termination may require exponential time, but it always terminates for a finite set of MVDs and FDs.
- The chase is **sound and complete** for testing implication of FDs and MVDs.

---

## 5. Serializability Theory

### The Problem

Formalize when concurrent execution of transactions is "correct" -- equivalent to some serial execution.

### Conflict Serializability

**Definition.** Two operations conflict if they are by different transactions, access the same data item, and at least one is a write. The three conflict types are:

- Read-write ($r_i[x], w_j[x]$): unrepeatable read
- Write-read ($w_i[x], r_j[x]$): dirty read
- Write-write ($w_i[x], w_j[x]$): lost update

**Precedence graph (serialization graph).** For schedule $S$, construct $G(S) = (V, E)$:

- $V = \{T_1, T_2, \ldots, T_n\}$ (active transactions)
- $(T_i, T_j) \in E$ iff there exist conflicting operations $o_i \in T_i$, $o_j \in T_j$ with $o_i <_S o_j$

**Theorem (conflict serializability).** Schedule $S$ is conflict-serializable if and only if the precedence graph $G(S)$ is acyclic. If acyclic, any topological sort gives an equivalent serial order.

*Proof sketch.* ($\Rightarrow$) If $S$ is conflict-equivalent to serial schedule $T_{i_1}, \ldots, T_{i_n}$, then every edge $(T_a, T_b)$ must respect the serial order ($a$ before $b$), so no cycle exists. ($\Leftarrow$) If acyclic, a topological sort yields an ordering. Swapping adjacent non-conflicting operations transforms $S$ into this serial schedule, preserving all conflict orderings.

### View Serializability

**Definition.** Schedules $S$ and $S'$ are **view-equivalent** if:

1. For each data item $x$, if $T_i$ reads the initial value of $x$ in $S$, then $T_i$ reads the initial value of $x$ in $S'$.
2. For each read $r_j[x]$ reading from $w_i[x]$ in $S$, the same reads-from relationship holds in $S'$.
3. For each data item $x$, the final write on $x$ is by the same transaction in both schedules.

**Theorem.** Every conflict-serializable schedule is view-serializable, but not conversely. The additional view-serializable schedules are those involving **blind writes** (writes not preceded by reads of the same item).

**Theorem (NP-completeness).** Deciding whether a schedule is view-serializable is NP-complete (Papadimitriou, 1979). This is why practical systems use conflict serializability.

---

## 6. Two-Phase Locking: Correctness Proof

### The Problem

Prove that two-phase locking (2PL) guarantees conflict serializability.

### The Protocol

A transaction $T_i$ obeys 2PL if it never acquires a lock after releasing any lock. Formally, $T_i$ has a **lock point** $\ell_i$: all lock acquisitions occur before $\ell_i$, and all lock releases occur after $\ell_i$.

### The Proof

**Theorem.** Every schedule produced under 2PL is conflict-serializable.

*Proof.* Suppose for contradiction that the precedence graph has a cycle $T_1 \to T_2 \to \cdots \to T_k \to T_1$.

Edge $T_i \to T_{i+1}$ means there exist conflicting operations where $T_i$'s operation precedes $T_{i+1}$'s in the schedule. Since they conflict on the same data item $x$, one holds a lock on $x$ while the other waits. Specifically:

- $T_i$ must have acquired its lock on $x$ before releasing it.
- $T_{i+1}$ must acquire the conflicting lock on $x$ after $T_i$ releases it.

This implies $T_i$'s lock point precedes $T_{i+1}$'s lock point: $\ell_1 < \ell_2 < \cdots < \ell_k < \ell_1$.

This is a contradiction (a strict ordering cannot cycle). Therefore the precedence graph is acyclic, and the schedule is conflict-serializable. $\square$

### Strict 2PL

**Corollary.** Strict 2PL (exclusive locks held until commit) additionally prevents cascading aborts: if $T_i$ aborts, no other transaction has read $T_i$'s uncommitted writes, so no other transaction needs to abort.

---

## 7. ARIES Recovery Algorithm

### The Problem

Design a recovery algorithm that supports steal/no-force buffer management, fine-granularity locking, and efficient restart recovery.

### Algorithm Overview

ARIES (Algorithms for Recovery and Isolation Exploiting Semantics) uses three data structures:

1. **Log:** Sequential, append-only record of all modifications. Each entry has a unique LSN (Log Sequence Number).
2. **Dirty Page Table (DPT):** Maps page IDs to recLSN (the first LSN that dirtied the page since last flush).
3. **Active Transaction Table (ATT):** Maps transaction IDs to their state and lastLSN.

### Normal Processing

For each modification by transaction $T$:

1. Write log record $\langle \text{LSN}, T, \text{pageID}, \text{redo-info}, \text{undo-info}, \text{prevLSN} \rangle$.
2. Update pageLSN on the in-memory page to the new LSN.
3. If page is not in DPT, add it with recLSN = new LSN.

**WAL invariant:** Before a dirty page is flushed to disk, all log records up to pageLSN must be on stable storage.

**Commit:** Force all log records for $T$ to stable storage (up to $T$'s lastLSN). Write a commit record.

### Recovery: Three Phases

**Phase 1: Analysis.** Starting from the most recent checkpoint:

1. Initialize DPT and ATT from checkpoint records.
2. Scan forward through the log:
   - For each update record: if page not in DPT, add with recLSN = this LSN. Update ATT.
   - For each commit record: remove transaction from ATT.
   - For each end record: remove transaction from ATT.
3. At end of log: DPT contains all possibly dirty pages, ATT contains all uncommitted transactions.

**Phase 2: Redo (repeat history).** Starting from $\min(\text{recLSN values in DPT})$:

For each redo-able log record with LSN $\ell$ affecting page $P$:

1. If $P \notin \text{DPT}$, skip (page was already flushed).
2. If $\text{DPT}[P].\text{recLSN} > \ell$, skip (page was flushed after this update).
3. Read page $P$ from disk. If $P.\text{pageLSN} \geq \ell$, skip (update already on disk).
4. Otherwise, redo the operation and set $P.\text{pageLSN} = \ell$.

**Phase 3: Undo.** Process uncommitted transactions (those remaining in ATT):

1. Collect the lastLSN of each loser transaction.
2. Process in reverse LSN order (latest first).
3. For each update to undo: apply the undo operation, write a CLR (Compensation Log Record) with an undoNextLSN pointing to the prevLSN of the undone record.
4. When undoNextLSN is null, transaction is fully undone; write an end record.

### Key Properties

- **Correctness:** Repeating history restores the exact pre-crash state, then undo reverses uncommitted work.
- **Idempotent redo and undo:** The pageLSN check in redo and CLRs in undo ensure that recovery is idempotent -- restarting recovery mid-recovery produces the same result.
- **CLRs are redo-only:** If a crash occurs during undo, the CLRs are redone in Phase 2, and undo resumes from where it left off (via undoNextLSN).
- **Nested top actions:** CLRs can point past completed sub-operations to avoid re-undoing committed nested work.

---

## 8. Query Optimization: Join Ordering and the Selinger Optimizer

### The Problem

Find the optimal join order for $n$ relations, minimizing total query cost.

### Combinatorial Complexity

For $n$ relations, the number of possible join trees is:

- **Left-deep trees:** $n!$ orderings
- **Bushy trees:** $\frac{(2(n-1))!}{(n-1)!}$ (Catalan number $\times$ $n!$)

For $n = 10$: approximately $10! = 3.6 \times 10^6$ left-deep trees and $\sim 1.76 \times 10^{10}$ bushy trees.

### Dynamic Programming (Selinger, 1979)

The System R optimizer introduced bottom-up dynamic programming over subsets of relations.

**State space:** For each subset $S \subseteq \{R_1, \ldots, R_n\}$ with $|S| \geq 2$, compute the optimal plan.

**Recurrence:**

$$\text{OptCost}(S) = \min_{S_1 \cup S_2 = S,\, S_1 \cap S_2 = \emptyset} \left[ \text{OptCost}(S_1) + \text{OptCost}(S_2) + \text{JoinCost}(S_1, S_2) \right]$$

For left-deep trees only, the recurrence simplifies:

$$\text{OptCost}(S) = \min_{R_i \in S} \left[ \text{OptCost}(S \setminus \{R_i\}) + \text{JoinCost}(S \setminus \{R_i\}, R_i) \right]$$

**Complexity:** $O(2^n)$ subsets, each requiring $O(n)$ splits for left-deep and $O(3^n / 2^n)$ splits for bushy trees (via subset enumeration). With $n$ typically $\leq 15$, this is tractable.

### Interesting Orders

The Selinger optimizer's key insight is tracking **interesting orders**: for each subset $S$, retain not just the cheapest plan but the cheapest plan for each useful sort order (e.g., sorted on a join attribute). A plan with higher cost but a useful sort order may be globally optimal by eliminating a later sort.

**Cost with interesting orders:**

$$\text{OptPlan}(S, \text{order}) = \min_{\text{plans producing order}} \text{Cost}(\text{plan})$$

The final answer is:

$$\min_{\text{order}} \left[ \text{OptPlan}(\{R_1, \ldots, R_n\}, \text{order}) + \text{SortCost}(\text{order} \to \text{required order}) \right]$$

### Selectivity Estimation

The optimizer relies on selectivity estimates:

$$\text{sel}(A = c) = \frac{1}{V(A, R)}, \quad \text{sel}(A = B) = \frac{1}{\max(V(A,R), V(B,S))}$$

$$\text{sel}(A < c) \approx \frac{c - \min(A)}{\max(A) - \min(A)}$$

$$|R \bowtie_\theta S| \approx |R| \cdot |S| \cdot \text{sel}(\theta)$$

Errors in selectivity estimation compound multiplicatively across joins, which is why cardinality estimation remains one of the hardest problems in query optimization.

---

## 9. Datalog and Recursive Queries

### The Problem

Extend the relational model to handle recursive queries (e.g., transitive closure) that are inexpressible in basic relational algebra.

### Datalog Syntax

A Datalog program consists of rules of the form:

$$H(x_1, \ldots, x_k) \leftarrow B_1(y_1, \ldots), B_2(z_1, \ldots), \ldots$$

where $H$ is the head (an IDB -- intensional database predicate) and the $B_i$ are body literals (EDB -- extensional database, or IDB predicates).

**Example -- transitive closure (ancestor):**

$$\text{Ancestor}(x, y) \leftarrow \text{Parent}(x, y)$$
$$\text{Ancestor}(x, y) \leftarrow \text{Parent}(x, z), \text{Ancestor}(z, y)$$

### Fixed-Point Semantics

**Naive evaluation:** Repeatedly apply all rules to derive new facts until no new facts are produced. The result is the **least fixed point** of the immediate consequence operator $T_P$:

$$T_P(I) = \{ H\theta \mid (H \leftarrow B_1, \ldots, B_m) \in P,\; \theta \text{ is a substitution},\; B_1\theta, \ldots, B_m\theta \in I \}$$

Starting from $I_0 = \text{EDB}$:

$$I_{k+1} = I_k \cup T_P(I_k)$$

The sequence $I_0 \subseteq I_1 \subseteq I_2 \subseteq \cdots$ converges in at most $|D|^{\text{max arity}}$ steps (where $|D|$ is the domain size).

**Semi-naive evaluation:** Only use *newly derived* facts from the previous iteration to avoid redundant computation:

$$\Delta I_{k+1} = T_P(I_k) \setminus I_k$$

$$I_{k+1} = I_k \cup \Delta I_{k+1}$$

Each rule application uses at least one fact from $\Delta I_k$, dramatically reducing work.

### Expressive Power

**Theorem.** Datalog (without negation) expresses exactly the class of queries definable by the monotone fragment of existential fixed-point logic.

**Theorem.** Datalog cannot express:
- Set complement or difference (non-monotone operations)
- Parity ("is the number of tuples even?")
- Aggregation

**Stratified negation:** Adding negation with a stratification constraint (no recursion through negation) yields stratified Datalog, which expresses all first-order queries plus bounded recursion.

**Theorem.** Datalog with stratified negation captures exactly the inflationary fixed-point queries on ordered databases, which equals $\text{P}$ (polynomial time) on ordered structures.

---

## 10. Concurrency Control Theory: Beyond 2PL

### Timestamp Ordering

Assign each transaction $T_i$ a unique timestamp $\text{TS}(T_i)$. Maintain for each data item $x$:

- $\text{R-TS}(x)$: largest timestamp of any transaction that read $x$
- $\text{W-TS}(x)$: largest timestamp of any transaction that wrote $x$

**Read rule:** $T_i$ wants to read $x$:
- If $\text{TS}(T_i) < \text{W-TS}(x)$: $T_i$ is "too late" (would read a future write). Abort and restart with new timestamp.
- Otherwise: proceed, set $\text{R-TS}(x) = \max(\text{R-TS}(x), \text{TS}(T_i))$.

**Write rule:** $T_i$ wants to write $x$:
- If $\text{TS}(T_i) < \text{R-TS}(x)$: a "later" transaction already read the old value. Abort.
- If $\text{TS}(T_i) < \text{W-TS}(x)$: **Thomas's write rule** -- skip the write (it would be overwritten anyway). This allows more schedules than basic TO.
- Otherwise: proceed, set $\text{W-TS}(x) = \text{TS}(T_i)$.

**Theorem.** Basic timestamp ordering produces conflict-serializable schedules equivalent to the serial order defined by timestamps.

### Optimistic Concurrency Control (Kung-Robinson, 1981)

Three phases for each transaction $T_i$:

1. **Read phase:** Execute reads/writes on private copies.
2. **Validation phase:** Check for conflicts with committed or validating transactions.
3. **Write phase:** If validation succeeds, apply writes to the database.

**Validation test for $T_i$ against each $T_j$ (where $\text{TS}(T_j) < \text{TS}(T_i)$):**

One of the following must hold:

1. $T_j$ completed all three phases before $T_i$ began its read phase, or
2. $T_j$ completed its write phase before $T_i$ began its write phase, and $\text{WriteSet}(T_j) \cap \text{ReadSet}(T_i) = \emptyset$, or
3. $T_j$ completed its read phase before $T_i$ completed its read phase, and $\text{WriteSet}(T_j) \cap \text{ReadSet}(T_i) = \emptyset$ and $\text{WriteSet}(T_j) \cap \text{WriteSet}(T_i) = \emptyset$.

### Serializable Snapshot Isolation (Cahill et al., 2009)

SSI detects **dangerous structures** in the dependency graph: two consecutive anti-dependency edges ($T_1 \xrightarrow{rw} T_2 \xrightarrow{rw} T_3$) where $T_2$ is a **pivot**. If detected, abort one transaction.

**Theorem.** SSI produces serializable schedules with lower abort rates than S2PL in read-heavy workloads, while avoiding the deadlock problem of lock-based methods entirely.

---

## Tips

- Armstrong's axioms completeness proof is constructive: the two-tuple relation is the canonical counterexample for any non-implied FD. Use this technique whenever you need to show an FD does not follow from a given set.
- The chase algorithm subsumes the lossless-join test: rather than memorizing the special case for two-decomposition, apply the chase to a tableau. It also decides MVD implication.
- Conflict serializability via precedence graphs is polynomially testable; view serializability is NP-complete. This is why every practical system uses conflict serializability (or its refinements like SSI).
- The 2PL correctness proof generalizes: any protocol whose lock points define a total order on transactions produces serializable schedules.
- ARIES "repeats history" even for aborted transactions, then undoes them. This design choice dramatically simplifies the interaction between recovery and concurrency control.
- In the Selinger optimizer, interesting orders are the key to avoiding local optima. Without them, the optimizer might choose a hash join (cheaper locally) over a sort-merge join whose output order eliminates a downstream sort.
- Datalog's inability to express complement means it cannot express "find all students not enrolled in any course" without negation -- a fundamental limitation of monotone queries.

## See Also

- complexity-theory
- graph-theory
- set-theory
- logic
- distributed-systems

## References

- Codd, E. F. "A Relational Model of Data for Large Shared Data Banks" (1970), CACM
- Codd, E. F. "Relational Completeness of Data Base Sublanguages" (1972), IBM Research Report
- Armstrong, W. W. "Dependency Structures of Data Base Relationships" (1974), IFIP Congress
- Selinger, P. G. et al. "Access Path Selection in a Relational Database Management System" (1979), SIGMOD
- Mohan, C. et al. "ARIES: A Transaction Recovery Method Supporting Fine-Granularity Locking" (1992), ACM TODS
- Bernstein, P. A., Hadzilacos, V. & Goodman, N. "Concurrency Control and Recovery in Database Systems" (Addison-Wesley, 1987) -- free online
- Papadimitriou, C. H. "The Serializability of Concurrent Database Updates" (1979), JACM
- Abiteboul, S., Hull, R. & Vianu, V. "Foundations of Databases" (Addison-Wesley, 1995) -- the "Alice book," free online
- Garcia-Molina, H., Ullman, J. D. & Widom, J. "Database Systems: The Complete Book" (2nd ed., Pearson, 2008)
- Cahill, M. J., Rohm, U. & Fekete, A. D. "Serializable Isolation for Snapshot Databases" (2009), ACM TODS
