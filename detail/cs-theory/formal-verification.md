# The Mathematics of Correctness -- Formal Verification from Hoare Logic to Model Checking

> *Formal verification is the application of rigorous mathematical methods to prove that a system conforms to its specification. Unlike testing, which can only show the presence of bugs, verification can prove their absence.*

---

## 1. Hoare Logic: Axiomatic Semantics

### The Problem

Given a program $C$, a precondition $P$, and a postcondition $Q$, prove that if $P$ holds before executing $C$ and $C$ terminates, then $Q$ holds afterward. This is the *partial correctness* assertion $\{P\} \; C \; \{Q\}$, known as a **Hoare triple**.

### The Axiom and Rules

**Assignment Axiom:**

$$\{Q[x / E]\} \; x := E \; \{Q\}$$

The precondition is obtained by substituting every free occurrence of $x$ in $Q$ with expression $E$. This is backward reasoning: start from what you want, compute what you need.

**Example:** To establish $\{?\} \; x := x + 1 \; \{x > 5\}$, substitute $x+1$ for $x$ in the postcondition:

$$\{x + 1 > 5\} \; x := x + 1 \; \{x > 5\}$$

which simplifies to $\{x > 4\} \; x := x + 1 \; \{x > 5\}$.

**Sequencing Rule:**

$$\frac{\{P\} \; C_1 \; \{R\} \quad \{R\} \; C_2 \; \{Q\}}{\{P\} \; C_1 ; C_2 \; \{Q\}}$$

The intermediate assertion $R$ bridges the two commands.

**Conditional Rule:**

$$\frac{\{P \land B\} \; C_1 \; \{Q\} \quad \{P \land \lnot B\} \; C_2 \; \{Q\}}{\{P\} \; \textbf{if } B \textbf{ then } C_1 \textbf{ else } C_2 \; \{Q\}}$$

**While Rule:**

$$\frac{\{I \land B\} \; C \; \{I\}}{\{I\} \; \textbf{while } B \textbf{ do } C \; \{I \land \lnot B\}}$$

where $I$ is the **loop invariant** -- a property preserved by every iteration.

**Rule of Consequence:**

$$\frac{P' \Rightarrow P \quad \{P\} \; C \; \{Q\} \quad Q \Rightarrow Q'}{\{P'\} \; C \; \{Q'\}}$$

This allows strengthening preconditions and weakening postconditions.

### Weakest Precondition Calculus (Dijkstra)

The **weakest precondition** $wp(C, Q)$ is the weakest (most general) predicate $P$ such that $\{P\} \; C \; \{Q\}$ holds. Defined recursively:

$$wp(x := E, \; Q) = Q[x/E]$$

$$wp(C_1 ; C_2, \; Q) = wp(C_1, \; wp(C_2, \; Q))$$

$$wp(\textbf{if } B \textbf{ then } C_1 \textbf{ else } C_2, \; Q) = (B \Rightarrow wp(C_1, Q)) \land (\lnot B \Rightarrow wp(C_2, Q))$$

For a while loop with invariant $I$ and variant $t$ (for total correctness):

$$wp(\textbf{while } B \textbf{ do } C, \; Q) = I$$

provided:

1. $I \land \lnot B \Rightarrow Q$ (loop exit implies postcondition)
2. $\{I \land B\} \; C \; \{I\}$ (invariant is preserved)
3. $\{I \land B \land t = z\} \; C \; \{t < z\}$ and $I \land B \Rightarrow t > 0$ (variant decreases, total correctness)

### Worked Proof: Insertion Sort Inner Loop

Consider the inner loop of insertion sort, which shifts elements rightward to insert `key` at the correct position in a sorted prefix:

```
{A[0..j-1] is sorted, key = A[j], i = j - 1}

while i >= 0 and A[i] > key do
    A[i+1] := A[i]
    i := i - 1

A[i+1] := key

{A[0..j] is sorted}
```

**Loop invariant $I$:**

$$I \equiv A[0..i] \text{ is sorted} \;\land\; A[i+2..j] \text{ is sorted} \;\land\; \forall k \in [i+2, j]: A[k] > key \;\land\; -1 \leq i < j$$

In words: elements $A[0..i]$ are in their original sorted order, elements $A[i+2..j]$ are sorted and all greater than `key`, and position $i+1$ is the "hole" where an element has been duplicated by shifting.

**Proof obligations:**

*Initialization.* Before the loop, $i = j - 1$. The range $A[0..j-1]$ is sorted (precondition). The range $A[i+2..j] = A[j+1..j]$ is empty, so trivially sorted. The universal quantifier over an empty range is vacuously true. And $-1 \leq j - 1 < j$ holds for $j \geq 0$.

*Maintenance.* Assume $I \land i \geq 0 \land A[i] > key$. After $A[i+1] := A[i]$, the element at position $i$ is copied to $i+1$, extending the shifted region. After $i := i - 1$, the new invariant holds with the hole moved one position left: $A[0..i']$ is sorted (where $i' = i - 1$), $A[i'+2..j] = A[i+1..j]$ is sorted (the shifted region grew by one), and every element in the shifted region exceeds `key` (since we entered the loop body because $A[i] > key$).

*Termination.* The variant is $t = i + 1$. Each iteration decrements $i$ by 1, so $t$ decreases by 1. The guard $i \geq 0$ ensures $t > 0$ when entering the loop body.

*Postcondition.* When the loop exits, either $i < 0$ or $A[i] \leq key$. In both cases, after $A[i+1] := key$:

- Elements $A[0..i]$ are sorted and all $\leq key$ (by the exit condition).
- $A[i+1] = key$.
- Elements $A[i+2..j]$ are sorted and all $> key$ (by the invariant).

Therefore $A[0..j]$ is sorted. $\square$

---

## 2. Temporal Logic: LTL and CTL

### Linear Temporal Logic (LTL)

LTL formulas describe properties of individual execution paths $\pi = s_0, s_1, s_2, \ldots$

**Syntax:**

$$\varphi ::= p \mid \lnot \varphi \mid \varphi_1 \land \varphi_2 \mid X\,\varphi \mid \varphi_1 \, U \, \varphi_2$$

where $p \in AP$ is an atomic proposition.

**Derived operators:**

$$F\,\varphi \equiv \top \, U \, \varphi \quad \text{(eventually/finally)}$$

$$G\,\varphi \equiv \lnot F\,\lnot\varphi \quad \text{(globally/always)}$$

$$\varphi_1 \, W \, \varphi_2 \equiv (\varphi_1 \, U \, \varphi_2) \lor G\,\varphi_1 \quad \text{(weak until)}$$

$$\varphi_1 \, R \, \varphi_2 \equiv \lnot(\lnot\varphi_1 \, U \, \lnot\varphi_2) \quad \text{(release)}$$

**Semantics.** Given an infinite path $\pi$ and position $i$:

$$\pi, i \models p \iff p \in L(s_i)$$

$$\pi, i \models X\,\varphi \iff \pi, i+1 \models \varphi$$

$$\pi, i \models \varphi_1 \, U \, \varphi_2 \iff \exists j \geq i : \pi, j \models \varphi_2 \;\land\; \forall k \in [i, j) : \pi, k \models \varphi_1$$

**Common specification patterns:**

| Pattern | LTL Formula | Meaning |
|---------|------------|---------|
| Safety | $G\,\lnot \textit{bad}$ | Nothing bad ever happens |
| Liveness | $G\,(req \Rightarrow F\,grant)$ | Every request eventually gets a grant |
| Fairness | $G\,F\,\textit{enabled} \Rightarrow G\,F\,\textit{executed}$ | If continually enabled, eventually executed |
| Precedence | $\lnot q \, U \, p$ | $p$ must precede $q$ |
| Response | $G\,(p \Rightarrow F\,q)$ | Every $p$ is eventually followed by $q$ |

### Computation Tree Logic (CTL)

CTL adds **path quantifiers** $A$ (for all paths) and $E$ (there exists a path) before every temporal operator.

**Syntax:**

$$\Phi ::= p \mid \lnot\Phi \mid \Phi_1 \land \Phi_2 \mid AX\,\Phi \mid EX\,\Phi \mid A[\Phi_1 \, U \, \Phi_2] \mid E[\Phi_1 \, U \, \Phi_2]$$

**Derived:**

$$AF\,\Phi \equiv A[\top \, U \, \Phi] \qquad EF\,\Phi \equiv E[\top \, U \, \Phi]$$

$$AG\,\Phi \equiv \lnot EF\,\lnot\Phi \qquad EG\,\Phi \equiv \lnot AF\,\lnot\Phi$$

**Semantics** (state formulas, evaluated at a state $s$ in Kripke structure $M$):

$$M, s \models AX\,\Phi \iff \forall s' : (s, s') \in R \Rightarrow M, s' \models \Phi$$

$$M, s \models EX\,\Phi \iff \exists s' : (s, s') \in R \land M, s' \models \Phi$$

$$M, s \models A[\Phi_1 \, U \, \Phi_2] \iff \text{on every path from } s, \; \Phi_1 \, U \, \Phi_2 \text{ holds}$$

### CTL vs LTL: Expressiveness

CTL and LTL are **incomparable** in expressiveness:

- $A(F\,G\,p)$ is expressible in LTL ($F\,G\,p$) but also in CTL.
- The LTL formula $F\,G\,p$ has no equivalent in CTL.
- The CTL formula $AG\,(EF\,p)$ (from every reachable state, there exists a path to a $p$-state) has no equivalent in LTL.
- **CTL\*** subsumes both LTL and CTL.

### CTL Model Checking Algorithm

The CTL model checking algorithm runs in $O(|M| \cdot |\Phi|)$ time, where $|M| = |S| + |R|$ is the size of the Kripke structure and $|\Phi|$ is the length of the formula.

For each subformula, compute the set of states satisfying it, bottom-up:

- $\text{Sat}(p) = \{s \mid p \in L(s)\}$
- $\text{Sat}(EX\,\Phi) = \{s \mid \exists s' \in \text{Sat}(\Phi) : (s, s') \in R\}$ -- one predecessor computation
- $\text{Sat}(E[\Phi_1 \, U \, \Phi_2])$ -- backward fixpoint from $\text{Sat}(\Phi_2)$, extending through $\text{Sat}(\Phi_1)$
- $\text{Sat}(EG\,\Phi)$ -- greatest fixpoint: start with $\text{Sat}(\Phi)$, iteratively remove states with no successor in the set

**LTL model checking** is PSPACE-complete in the formula size (vs. polynomial for CTL), because it requires constructing a Buchi automaton from the negated formula.

---

## 3. SPIN Model Checker

SPIN (Simple Promela Interpreter) verifies models written in the Promela language against LTL properties.

### Example: Mutual Exclusion (Peterson's Algorithm)

```promela
bool flag[2] = false;
byte turn = 0;

active [2] proctype P() {
    byte me = _pid;
    byte other = 1 - _pid;

    do
    :: true ->
        /* non-critical section */
        flag[me] = true;
        turn = other;
        (flag[other] == false || turn == me);  /* wait */
        /* critical section */
        cs: skip;
        flag[me] = false;
    od
}

ltl mutex { [] !(P[0]@cs && P[1]@cs) }
ltl liveness { [] (P[0]@cs -> <> !P[0]@cs) }
```

**Verification commands:**

```
spin -a peterson.pml          # generate verifier
gcc -o pan pan.c -DSAFETY     # compile for safety check
./pan                         # run verification
./pan -a -f                   # check liveness (acceptance cycles)
```

SPIN performs on-the-fly verification using nested depth-first search (for liveness) or standard DFS (for safety). It constructs the product automaton of the system and the negated property automaton, searching for accepting cycles.

---

## 4. TLA+ Specification

### Formal Foundations

A TLA+ specification is a temporal logic formula of the form:

$$\text{Spec} \equiv \text{Init} \land \square[\text{Next}]_{\text{vars}} \land \text{Liveness}$$

where:

- $\text{Init}$ is a state predicate characterizing initial states
- $\square[A]_v \equiv \square(A \lor v' = v)$ means: every step either satisfies action $A$ or is a **stuttering step** (no variable changes)
- Stuttering invariance enables **refinement**: a lower-level spec can take multiple steps to implement one higher-level step

### Example: Two-Phase Commit

```tla
--------------------------- MODULE TwoPhaseCommit ---------------------------
EXTENDS Naturals, FiniteSets
CONSTANT RM                          \* set of resource managers

VARIABLE rmState, tmState, tmPrepared, msgs

vars == <<rmState, tmState, tmPrepared, msgs>>

Init ==
    /\ rmState    = [r \in RM |-> "working"]
    /\ tmState    = "init"
    /\ tmPrepared = {}
    /\ msgs       = {}

TMRcvPrepared(r) ==                  \* TM receives Prepared from r
    /\ tmState = "init"
    /\ [type |-> "Prepared", rm |-> r] \in msgs
    /\ tmPrepared' = tmPrepared \cup {r}
    /\ UNCHANGED <<rmState, tmState, msgs>>

TMCommit ==                          \* TM decides to commit
    /\ tmState = "init"
    /\ tmPrepared = RM               \* all RMs prepared
    /\ tmState' = "committed"
    /\ msgs' = msgs \cup {[type |-> "Commit"]}
    /\ UNCHANGED <<rmState, tmPrepared>>

TMAbort ==                           \* TM decides to abort
    /\ tmState = "init"
    /\ tmState' = "aborted"
    /\ msgs' = msgs \cup {[type |-> "Abort"]}
    /\ UNCHANGED <<rmState, tmPrepared>>

RMPrepare(r) ==                      \* RM r prepares
    /\ rmState[r] = "working"
    /\ rmState' = [rmState EXCEPT ![r] = "prepared"]
    /\ msgs' = msgs \cup {[type |-> "Prepared", rm |-> r]}
    /\ UNCHANGED <<tmState, tmPrepared>>

RMRcvCommit(r) ==                    \* RM r receives Commit
    /\ [type |-> "Commit"] \in msgs
    /\ rmState' = [rmState EXCEPT ![r] = "committed"]
    /\ UNCHANGED <<tmState, tmPrepared, msgs>>

RMRcvAbort(r) ==                     \* RM r receives Abort
    /\ [type |-> "Abort"] \in msgs
    /\ rmState' = [rmState EXCEPT ![r] = "aborted"]
    /\ UNCHANGED <<tmState, tmPrepared, msgs>>

Next ==
    \/ TMCommit
    \/ TMAbort
    \/ \E r \in RM :
        \/ TMRcvPrepared(r)
        \/ RMPrepare(r)
        \/ RMRcvCommit(r)
        \/ RMRcvAbort(r)

Spec == Init /\ [][Next]_vars

\* Safety: no RM commits while another aborts
Consistency ==
    \A r1, r2 \in RM :
        ~(rmState[r1] = "committed" /\ rmState[r2] = "aborted")

THEOREM Spec => []Consistency
=============================================================================
```

This specification can be model-checked with TLC by instantiating $RM$ as a finite set (e.g., $\{r_1, r_2, r_3\}$). TLC explores all reachable states and verifies that `Consistency` is an invariant.

---

## 5. Separation Logic

### The Problem

Classical Hoare logic cannot naturally express properties of pointer-manipulating programs because aliasing makes frame conditions intractable. If $x$ and $y$ might alias, updating $*x$ could change $*y$, invalidating any assertion about $*y$.

### Heap Assertions

Separation logic extends Hoare logic with assertions about the heap:

$$\textbf{emp} \quad \text{(the heap is empty)}$$

$$E_1 \mapsto E_2 \quad \text{(the heap contains exactly one cell: address } E_1 \text{ with value } E_2\text{)}$$

$$P * Q \quad \text{(separating conjunction: } P \text{ and } Q \text{ hold on disjoint heap portions)}$$

$$P \mathbin{-\!\!*} Q \quad \text{(magic wand: if } P \text{ is added to the current heap, } Q \text{ holds)}$$

### Semantics

A state is a pair $(s, h)$ where $s$ is a store (variables to values) and $h$ is a heap (addresses to values, partial function).

$$s, h \models E_1 \mapsto E_2 \iff \text{dom}(h) = \{[\![E_1]\!]_s\} \text{ and } h([\![E_1]\!]_s) = [\![E_2]\!]_s$$

$$s, h \models P * Q \iff \exists h_1, h_2 : h = h_1 \uplus h_2 \land s, h_1 \models P \land s, h_2 \models Q$$

where $h_1 \uplus h_2$ denotes the union of $h_1$ and $h_2$ with disjoint domains.

### The Frame Rule

$$\frac{\{P\} \; C \; \{Q\}}{\{P * R\} \; C \; \{Q * R\}}$$

provided $C$ does not modify any free variable of $R$.

This is the key to **local reasoning**: verify each function against only the memory it accesses. The frame $R$ describes the rest of the heap, which is guaranteed untouched.

### Example: In-Place List Reversal

Linked list segment predicate:

$$\text{list}(\alpha, x) \equiv \begin{cases} x = \textbf{nil} \land \textbf{emp} & \text{if } \alpha = \epsilon \\ \exists y.\; x \mapsto (a, y) * \text{list}(\alpha', y) & \text{if } \alpha = a \cdot \alpha' \end{cases}$$

**Specification:**

$$\{\text{list}(\alpha, i)\} \; \texttt{rev}(i) \; \{\text{list}(\text{reverse}(\alpha), \textbf{ret})\}$$

**Loop invariant for iterative reversal** (with accumulator $r$ and iterator $j$):

$$\exists \alpha_1, \alpha_2.\; \alpha = \text{reverse}(\alpha_1) \cdot \alpha_2 \;\land\; \text{list}(\alpha_1, r) * \text{list}(\alpha_2, j)$$

At termination, $\alpha_2 = \epsilon$ and $\alpha_1 = \text{reverse}(\alpha)$, so $r$ points to the reversed list.

---

## 6. CEGAR: Counterexample-Guided Abstraction Refinement

### The Problem

Model checking infinite-state or very large finite-state systems requires abstraction, but coarse abstractions produce **spurious counterexamples** -- error traces in the abstract model that do not correspond to real errors.

### The CEGAR Loop

```
                    ┌──────────────────────┐
                    │  Abstract Model M_α  │
                    └──────────┬───────────┘
                               │
                        Model Check
                               │
              ┌────────────────┴────────────────┐
              │                                 │
         Property holds              Counterexample found
         (valid for concrete)              │
                                    Simulate on concrete
                                           │
                              ┌────────────┴────────────┐
                              │                         │
                         Real bug found         Spurious counterexample
                         (report)                       │
                                                 Refine abstraction
                                                        │
                                                ┌───────┴───────┐
                                                │  Refined M_α  │
                                                └───────────────┘
                                                    (repeat)
```

**Formally:**

1. **Abstract.** Construct abstract model $M_\alpha = \alpha(M)$ using abstraction function $\alpha$.
2. **Verify.** Model-check $M_\alpha \models \varphi$. If yes, the property holds on $M$ (soundness of abstraction).
3. **Validate.** If a counterexample $\sigma_\alpha$ is found, check if it is realizable in $M$. Simulate the abstract trace on the concrete model.
4. **Refine.** If $\sigma_\alpha$ is spurious, analyze why and refine $\alpha$ to eliminate the spurious trace. Common refinement strategies:
   - **Predicate abstraction refinement** (SLAM, BLAST): add new predicates that distinguish states conflated by the current abstraction.
   - **Interpolation-based refinement** (McMillan): extract Craig interpolants from infeasibility proofs to derive new predicates.

### Decidability and Complexity of Model Checking

| Problem | Complexity | Notes |
|---------|-----------|-------|
| CTL model checking | $O(\|M\| \cdot \|\Phi\|)$ | Polynomial in both model and formula |
| LTL model checking | $O(\|M\| \cdot 2^{|\Phi|})$ | Exponential in formula, polynomial in model |
| CTL* model checking | $O(\|M\| \cdot 2^{2^{|\Phi|}})$ | Doubly exponential in formula |
| LTL satisfiability | PSPACE-complete | Is there any model satisfying the formula? |
| CTL satisfiability | EXPTIME-complete | -- |
| Pushdown model checking (LTL) | EXPTIME-complete | For recursive programs (infinite state) |
| Timed automata (TCTL) | PSPACE-complete | Decidable via region/zone abstraction |
| Hybrid automata (general) | Undecidable | Reducible from halting problem |

The key insight: model checking finite-state systems against temporal logic specifications is decidable and often efficient, which is what makes it practically useful. The state explosion problem is an engineering challenge, not a theoretical barrier.

---

## 7. SAT/SMT and Bounded Model Checking

### Bounded Model Checking (BMC)

Instead of exploring the full state space, BMC unfolds the transition relation $k$ times and encodes the verification problem as a SAT/SMT query:

$$\text{BMC}(k) \equiv I(s_0) \land \bigwedge_{i=0}^{k-1} T(s_i, s_{i+1}) \land \bigvee_{i=0}^{k} \lnot P(s_i)$$

where $I$ is the initial condition, $T$ is the transition relation, and $P$ is the safety property. If the formula is satisfiable, the satisfying assignment encodes a counterexample of length at most $k$.

**Advantages over BDD-based symbolic model checking:**

- No need to compute fixpoints or BDD variable orderings.
- SAT solvers (CDCL) handle industrial-scale formulas with millions of variables.
- Naturally produces bounded counterexamples.
- Can be extended to prove properties (via $k$-induction or interpolation).

### k-Induction

To prove a property $P$ holds for all reachable states (not just up to depth $k$):

1. **Base case:** $I(s_0) \land \bigwedge_{i=0}^{k-1} T(s_i, s_{i+1}) \Rightarrow \bigwedge_{i=0}^{k} P(s_i)$
2. **Inductive step:** $\bigwedge_{i=0}^{k} P(s_i) \land \bigwedge_{i=0}^{k} T(s_i, s_{i+1}) \Rightarrow P(s_{k+1})$

If both hold, $P$ is an invariant. The parameter $k$ controls the strength of the induction hypothesis; larger $k$ can prove properties that simple ($k=0$) induction cannot.

---

## Prerequisites

- Propositional and first-order logic
- Finite automata (Buchi automata for LTL)
- Basic programming language semantics
- Graph algorithms (DFS, BFS, fixpoint computation)
- Predicate logic and proof techniques

## Complexity

| Level | Description |
|-------|-------------|
| **Beginner** | State Hoare triples for simple programs. Identify loop invariants for basic loops (summation, maximum). Distinguish safety from liveness properties. Write simple LTL formulas for mutual exclusion. |
| **Intermediate** | Prove partial and total correctness of sorting algorithms. Encode verification conditions using weakest preconditions. Model-check small Promela models with SPIN. Write TLA+ specs for simple protocols. Understand the frame rule in separation logic. |
| **Advanced** | Construct CEGAR refinement loops. Prove correctness of pointer-manipulating programs using separation logic. Encode BMC problems as SAT instances. Prove decidability results for temporal logic model checking. Design abstract domains for abstract interpretation. |
