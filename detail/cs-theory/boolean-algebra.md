# Boolean Algebra and Logic -- From Lattice Theory to SAT Solving

> *Boolean algebra provides the algebraic foundation for digital logic and formal reasoning. Its deep connections to lattice theory, circuit complexity, and computational hardness make it a cornerstone of theoretical computer science.*

---

## 1. Boolean Algebra as a Lattice

### The Problem

Establish the algebraic structure of Boolean algebras and their relationship to lattice theory.

### The Formula

A **Boolean algebra** is a complemented distributive lattice $(B, \land, \lor, \lnot, 0, 1)$ satisfying:

1. $(B, \land, \lor)$ is a lattice: both $\land$ and $\lor$ are associative, commutative, and satisfy absorption ($a \land (a \lor b) = a$ and $a \lor (a \land b) = a$).
2. Distributivity: $a \land (b \lor c) = (a \land b) \lor (a \land c)$ and dually.
3. Complementation: for every $a \in B$, there exists $\lnot a$ with $a \land \lnot a = 0$ and $a \lor \lnot a = 1$.

The partial order is defined by $a \leq b$ iff $a \land b = a$ (equivalently, $a \lor b = b$).

### Key Properties

Every finite Boolean algebra is isomorphic to a power set algebra. Specifically, if $|B| = 2^n$, then $B \cong (\mathcal{P}(S), \cap, \cup, \complement, \emptyset, S)$ for some set $S$ with $|S| = n$.

**Proof sketch.** Let $\text{At}(B)$ be the set of atoms of $B$ (elements $a > 0$ with no $b$ satisfying $0 < b < a$). In a finite Boolean algebra, every element is the join of the atoms below it:

$$x = \bigvee \{a \in \text{At}(B) : a \leq x\}$$

The map $x \mapsto \{a \in \text{At}(B) : a \leq x\}$ is an isomorphism $B \to \mathcal{P}(\text{At}(B))$.

---

## 2. Stone's Representation Theorem

### The Problem

Characterize all Boolean algebras in terms of topological spaces.

### The Formula

**Stone's Theorem (1936).** Every Boolean algebra $B$ is isomorphic to the algebra of clopen (closed and open) subsets of a compact, totally disconnected Hausdorff space (a *Stone space*).

The Stone space of $B$ is $S(B) = \{\text{ultrafilters of } B\}$, topologized by taking as basis the sets:

$$\hat{a} = \{U \in S(B) : a \in U\}$$

for each $a \in B$.

### Construction

An **ultrafilter** $U$ on $B$ is a proper filter (upward-closed, closed under $\land$, not containing $0$) such that for every $a \in B$, either $a \in U$ or $\lnot a \in U$.

The representation map $\phi: B \to \text{Clopen}(S(B))$ is:

$$\phi(a) = \hat{a} = \{U \in S(B) : a \in U\}$$

This is an isomorphism of Boolean algebras:
- $\phi(a \land b) = \hat{a} \cap \hat{b}$
- $\phi(a \lor b) = \hat{a} \cup \hat{b}$
- $\phi(\lnot a) = S(B) \setminus \hat{a}$

**Significance.** Stone duality establishes a contravariant equivalence between the category of Boolean algebras and the category of Stone spaces, providing a bridge between algebra and topology.

---

## 3. Shannon's Switching Circuit Thesis

### The Problem

Show that Boolean algebra provides a complete mathematical framework for the analysis and synthesis of relay and switching circuits.

### The Formula

In his 1937 master's thesis at MIT, Claude Shannon established the correspondence:

| Circuit Element | Boolean Algebra |
|---|---|
| Switch closed (current flows) | 1 (true) |
| Switch open (no current) | 0 (false) |
| Series connection | AND ($a \land b$) |
| Parallel connection | OR ($a \lor b$) |
| Normally-closed relay | NOT ($\lnot a$) |

### Shannon Expansion (Boole Expansion Theorem)

Any Boolean function $f(x_1, \ldots, x_n)$ can be decomposed with respect to variable $x_i$:

$$f(x_1, \ldots, x_n) = x_i \cdot f(x_1, \ldots, x_{i-1}, 1, x_{i+1}, \ldots, x_n) \lor \lnot x_i \cdot f(x_1, \ldots, x_{i-1}, 0, x_{i+1}, \ldots, x_n)$$

Or more compactly:

$$f = x_i \cdot f_{x_i} \lor \lnot x_i \cdot f_{\overline{x_i}}$$

where $f_{x_i}$ (the positive cofactor) and $f_{\overline{x_i}}$ (the negative cofactor) are the restrictions of $f$ with $x_i$ set to 1 and 0 respectively.

**Applications.** Shannon expansion is the basis for:
- Binary Decision Diagrams (each node performs one expansion)
- Recursive circuit decomposition (Shannon-based synthesis)
- Sensitivity and influence analysis of Boolean functions

---

## 4. Functional Completeness and NAND Sufficiency

### The Problem

Prove that the single gate NAND is sufficient to express every Boolean function.

### The Formula

A set of Boolean operations $S$ is **functionally complete** if every Boolean function $f: \{0,1\}^n \to \{0,1\}$ can be expressed as a composition of operations in $S$.

**Post's Lattice Theorem (Emil Post, 1941).** The lattice of all clones (composition-closed sets of Boolean functions containing all projections) over $\{0,1\}$ is fully classified. A set of Boolean functions is functionally complete if and only if it is not contained in any of the five maximal clones:

1. $T_0$: functions preserving 0 ($f(0, \ldots, 0) = 0$)
2. $T_1$: functions preserving 1 ($f(1, \ldots, 1) = 1$)
3. $S$: self-dual functions ($f(\lnot x_1, \ldots, \lnot x_n) = \lnot f(x_1, \ldots, x_n)$)
4. $M$: monotone functions ($x \leq y \Rightarrow f(x) \leq f(y)$)
5. $L$: affine/linear functions ($f = a_0 \oplus a_1 x_1 \oplus \cdots \oplus a_n x_n$)

### NAND Sufficiency Proof

Define NAND as $\uparrow$: $a \uparrow b = \lnot(a \land b)$.

Check NAND against Post's five clones:

1. $T_0$: $\text{NAND}(0, 0) = 1 \neq 0$. NAND does not preserve 0.
2. $T_1$: $\text{NAND}(1, 1) = 0 \neq 1$. NAND does not preserve 1.
3. $S$: $\text{NAND}(0, 0) = 1$ but $\lnot \text{NAND}(1, 1) = \lnot 0 = 1$. We need $\text{NAND}(\lnot 0, \lnot 0) = \text{NAND}(1,1) = 0 \neq \lnot 1 = 0$. Actually check: self-duality requires $f(\bar{a}, \bar{b}) = \overline{f(a,b)}$. We have $\text{NAND}(1,1) = 0$ and $\overline{\text{NAND}(0,0)} = \overline{1} = 0$. And $\text{NAND}(1,0) = 1$ while $\overline{\text{NAND}(0,1)} = \overline{1} = 0$. Since $1 \neq 0$, NAND is not self-dual.
4. $M$: $\text{NAND}(0, 1) = 1 > \text{NAND}(1, 1) = 0$, violating monotonicity.
5. $L$: NAND is not affine (its truth table $\{1,1,1,0\}$ does not satisfy the XOR parity condition).

Since $\{\text{NAND}\}$ lies outside all five maximal clones, it is functionally complete by Post's theorem.

### Constructive Proof

$$\lnot a = a \uparrow a$$

$$a \land b = (a \uparrow b) \uparrow (a \uparrow b)$$

$$a \lor b = (a \uparrow a) \uparrow (b \uparrow b)$$

Since $\{\lnot, \land\}$ is functionally complete and both are expressible via NAND, $\{\text{NAND}\}$ is functionally complete.

---

## 5. Completeness of Resolution

### The Problem

Prove that propositional resolution is refutation-complete: if a set of clauses is unsatisfiable, iterated resolution derives the empty clause.

### The Formula

**Resolution Principle.** Given clauses $C_1 = \{p\} \cup A$ and $C_2 = \{\lnot p\} \cup B$, the resolvent is $C = A \cup B$.

**Ground Resolution Theorem.** A set of propositional clauses $\Sigma$ is unsatisfiable if and only if the empty clause $\square$ can be derived from $\Sigma$ by iterated resolution.

### Proof of Completeness (Semantic Argument)

The proof proceeds by strong induction on the number of propositional variables $n$.

**Base case ($n = 0$).** The only unsatisfiable set of clauses with no variables is $\{\square\}$, which already contains the empty clause.

**Inductive step.** Assume the theorem holds for fewer than $n$ variables. Let $\Sigma$ be an unsatisfiable set of clauses over variables $\{p_1, \ldots, p_n\}$. Pick variable $p_n$ and define:

$$\Sigma_{p_n} = \{C \setminus \{\lnot p_n\} : C \in \Sigma, p_n \notin C\}$$

This is $\Sigma$ with $p_n$ set to true (clauses containing $p_n$ are satisfied and removed; $\lnot p_n$ is deleted from remaining clauses). Similarly define $\Sigma_{\lnot p_n}$.

Both $\Sigma_{p_n}$ and $\Sigma_{\lnot p_n}$ are unsatisfiable (since $\Sigma$ is unsatisfiable regardless of $p_n$'s value) and contain fewer than $n$ variables. By the induction hypothesis, the empty clause is derivable from each.

Tracing the derivation back through the original clauses, we obtain either $\square$ directly, or clauses $\{p_n\}$ and $\{\lnot p_n\}$ as resolvents. Resolving these two yields $\square$.

---

## 6. The DPLL Algorithm for SAT

### The Problem

Design a sound and complete backtracking algorithm for the Boolean satisfiability problem.

### The Algorithm

The Davis-Putnam-Logemann-Loveland (DPLL, 1962) algorithm is a refinement of the Davis-Putnam (1960) procedure that uses splitting (backtracking search) instead of variable elimination.

```
function DPLL(F, alpha):
    // F: formula in CNF, alpha: partial assignment

    // Unit propagation
    while F contains a unit clause {l}:
        alpha := alpha + {l = true}
        F := simplify(F, l)

    // Pure literal elimination
    for each literal l appearing in F but not ~l:
        alpha := alpha + {l = true}
        F := simplify(F, l)

    // Termination
    if F is empty (all clauses satisfied):
        return SAT, alpha
    if F contains the empty clause:
        return UNSAT

    // Branching (splitting rule)
    choose an unassigned variable x
    if DPLL(F + {x}, alpha + {x = true}) = SAT:
        return SAT
    else:
        return DPLL(F + {~x}, alpha + {x = false})
```

where $\text{simplify}(F, l)$ removes all clauses containing $l$ and deletes $\lnot l$ from all remaining clauses.

### Complexity

DPLL runs in $O(2^n)$ time in the worst case, where $n$ is the number of variables. However, unit propagation and pure literal elimination often prune the search tree dramatically.

### From DPLL to CDCL

Modern SAT solvers extend DPLL with **Conflict-Driven Clause Learning** (CDCL):

1. **Implication graph.** Track the reason for each assignment. When a conflict arises, analyze the implication graph to find a **conflict clause** -- a new clause that is logically implied by the original formula but blocks the current partial assignment pattern.

2. **Non-chronological backtracking.** Instead of undoing only the last decision, backtrack to the earliest decision level involved in the conflict.

3. **Learned clause.** Add the conflict clause to the formula, pruning future search.

4. **Restarts.** Periodically restart the search from scratch while retaining learned clauses, to escape poor variable ordering decisions.

The combination of these techniques enables CDCL solvers to handle industrial SAT instances with millions of variables.

---

## 7. BDD Construction and Operations

### The Problem

Construct canonical representations of Boolean functions that support efficient manipulation.

### The Formula

An **Ordered Binary Decision Diagram (OBDD)** for a function $f(x_1, \ldots, x_n)$ with variable ordering $x_1 < x_2 < \cdots < x_n$ is a rooted directed acyclic graph where:

- Terminal nodes are labeled 0 or 1.
- Each internal node is labeled with a variable $x_i$ and has two outgoing edges: $\text{low}$ (for $x_i = 0$) and $\text{high}$ (for $x_i = 1$).
- Along every path from root to terminal, variables appear in the fixed order.

A **Reduced OBDD (ROBDD)** satisfies two additional reduction rules:
1. **No redundant tests:** If both edges of a node lead to the same child, eliminate the node.
2. **No duplicate subgraphs:** Merge nodes with the same variable, low child, and high child.

**Canonicity Theorem (Bryant, 1986).** For a fixed variable ordering, the ROBDD of a Boolean function is unique. Two functions are equal if and only if their ROBDDs are identical (pointer-equal after reduction).

### The Apply Algorithm

The core BDD operation is **Apply**, which computes $f \diamond g$ for any binary Boolean operation $\diamond$:

```
function Apply(op, f, g):
    // Memoization via computed table
    if (op, f, g) in cache:
        return cache[(op, f, g)]

    // Terminal cases
    if f and g are both terminals:
        return op(value(f), value(g))

    // Recursive Shannon expansion
    let x = topmost variable among f, g
    low  = Apply(op, f|_{x=0}, g|_{x=0})
    high = Apply(op, f|_{x=1}, g|_{x=1})

    // Reduction: skip redundant node
    if low == high:
        return low

    // Unique table lookup (hash consing)
    result = find_or_create_node(x, low, high)
    cache[(op, f, g)] = result
    return result
```

**Complexity.** Apply runs in $O(|f| \cdot |g|)$ time, where $|f|$ and $|g|$ are the sizes (number of nodes) of the input BDDs.

### Variable Ordering

The size of an ROBDD depends critically on the variable ordering. For the function $f = x_1 x_2 \lor x_3 x_4 \lor x_5 x_6$:

- Ordering $x_1 < x_2 < x_3 < x_4 < x_5 < x_6$: $O(n)$ nodes (linear).
- Ordering $x_1 < x_3 < x_5 < x_2 < x_4 < x_6$: $O(2^{n/2})$ nodes (exponential).

Finding the optimal variable ordering is itself NP-hard. In practice, heuristics (sifting, symmetric sifting) yield good orderings.

---

## 8. Boolean Functions and Circuit Complexity

### The Problem

Relate the complexity of Boolean functions to the size and depth of circuits computing them.

### The Formula

A **Boolean circuit** over a basis $\Omega$ (e.g., $\{\land, \lor, \lnot\}$) is a directed acyclic graph where:
- Input nodes are labeled with variables or constants.
- Internal nodes (gates) are labeled with operations from $\Omega$.
- One node is designated as the output.

The **circuit complexity** $C(f)$ of a Boolean function $f$ is the minimum number of gates in any circuit computing $f$.

**Shannon's Counting Argument (1949).** Almost all Boolean functions $f: \{0,1\}^n \to \{0,1\}$ require circuits of size $\Omega(2^n / n)$.

**Proof.** There are $2^{2^n}$ Boolean functions on $n$ variables. A circuit with $s$ gates over a basis of $b$ binary operations can be specified by at most $(s \cdot \log_2(b \cdot (s+n)^2))$ bits (choosing each gate's operation and two inputs). For $s < 2^n / (2n)$, the number of distinct circuits is less than $2^{2^n}$, so not all functions can be computed.

### The $P/\text{poly}$ Connection

A language $L$ is in $P/\text{poly}$ if there exists a polynomial-size circuit family $\{C_n\}$ such that $C_n$ decides $L$ restricted to inputs of length $n$.

$$P \subseteq P/\text{poly}}$$

If $\text{NP} \not\subseteq P/\text{poly}$, then $P \neq NP$. Proving super-polynomial circuit lower bounds for an NP problem would resolve the $P$ vs. $NP$ question. The best known lower bound for an explicit function in NP is $\Omega(n)$ -- embarrassingly far from the $2^n/n$ Shannon bound.

---

## 9. Shannon Expansion and Influence of Variables

### The Problem

Analyze how individual variables affect a Boolean function using the Shannon expansion.

### The Formula

Recall the Shannon expansion:

$$f = x_i \cdot f_{x_i} \lor \overline{x_i} \cdot f_{\overline{x_i}}$$

The **influence** of variable $x_i$ on function $f$ is:

$$\text{Inf}_i(f) = \Pr_{x}[f_{x_i}(x) \neq f_{\overline{x_i}}(x)]$$

where the probability is over uniform random assignment of all variables except $x_i$.

Equivalently, the influence is the fraction of inputs where flipping $x_i$ changes the output.

### Total Influence

The **total influence** (also called average sensitivity) is:

$$I(f) = \sum_{i=1}^{n} \text{Inf}_i(f)$$

**Poincare inequality for Boolean functions.** For any Boolean function $f: \{0,1\}^n \to \{0,1\}$:

$$\text{Var}(f) \leq I(f)$$

where $\text{Var}(f) = \mathbb{E}[f] \cdot (1 - \mathbb{E}[f])$.

### KKL Inequality

**Kahn-Kalai-Linial Theorem (1988).** For any balanced Boolean function $f$ (i.e., $\mathbb{E}[f] = 1/2$), there exists a variable $x_i$ with influence:

$$\text{Inf}_i(f) \geq \Omega\left(\frac{\log n}{n}\right)$$

This is tight: the tribes function achieves this bound.

---

## 10. Worked Example: BDD Construction

### Setup

Construct the ROBDD for $f(x_1, x_2, x_3) = x_1 x_2 \lor x_3$ with ordering $x_1 < x_2 < x_3$.

### Shannon Expansion

$$f = x_1 \cdot f_{x_1} \lor \overline{x_1} \cdot f_{\overline{x_1}}$$

$$f_{x_1} = x_2 \lor x_3, \quad f_{\overline{x_1}} = x_3$$

Expand $f_{x_1}$:

$$f_{x_1} = x_2 \cdot (f_{x_1})_{x_2} \lor \overline{x_2} \cdot (f_{x_1})_{\overline{x_2}} = x_2 \cdot 1 \lor \overline{x_2} \cdot x_3$$

Expand $f_{\overline{x_1}}$:

$$f_{\overline{x_1}} = x_2 \cdot x_3 \lor \overline{x_2} \cdot x_3 = x_3$$

Since $f_{\overline{x_1}}$ does not depend on $x_2$, the $x_2$ node on that branch is redundant and is eliminated.

### Resulting ROBDD

```
         x1
        /    \
      x3      x2
     / \     /  \
    0   1   x3    1
           / \
          0   1
```

The ROBDD has 3 internal nodes ($x_1$, $x_2$, $x_3$) and 2 terminals (0, 1), with the left subtree sharing the $x_3$ node. This is canonical for the ordering $x_1 < x_2 < x_3$.

### Verification via Truth Table

| $x_1$ | $x_2$ | $x_3$ | $f$ | Path in BDD |
|---|---|---|---|---|
| 0 | 0 | 0 | 0 | $x_1 \to_L x_3 \to_L 0$ |
| 0 | 0 | 1 | 1 | $x_1 \to_L x_3 \to_H 1$ |
| 0 | 1 | 0 | 0 | $x_1 \to_L x_3 \to_L 0$ |
| 0 | 1 | 1 | 1 | $x_1 \to_L x_3 \to_H 1$ |
| 1 | 0 | 0 | 0 | $x_1 \to_H x_2 \to_L x_3 \to_L 0$ |
| 1 | 0 | 1 | 1 | $x_1 \to_H x_2 \to_L x_3 \to_H 1$ |
| 1 | 1 | 0 | 1 | $x_1 \to_H x_2 \to_H 1$ |
| 1 | 1 | 1 | 1 | $x_1 \to_H x_2 \to_H 1$ |

All 8 rows match $f = x_1 x_2 \lor x_3$.

---

## Tips

- Stone's representation theorem means every abstract Boolean algebra question can be translated into a concrete question about sets. Use this for intuition.
- Shannon expansion is the conceptual backbone of BDDs, decision trees, and recursive circuit decomposition. Master it thoroughly.
- NAND sufficiency follows immediately from Post's lattice theorem, but the constructive proof (building NOT, AND, OR from NAND) is what matters for circuit design.
- Resolution is refutation-complete but not generatively complete. To prove a formula valid, negate it and derive the empty clause.
- DPLL is the skeleton; CDCL is the muscle. Modern SAT solvers owe their power primarily to clause learning and non-chronological backtracking.
- BDD variable ordering is an art. For circuits with independent subcircuits, interleave the variables of each subcircuit. For symmetric functions, any ordering works equally well.
- Shannon's counting argument is non-constructive. Proving explicit super-polynomial circuit lower bounds is one of the deepest open problems in complexity theory.
- The KKL inequality has profound consequences: no Boolean function can be simultaneously balanced and have all variables with negligible influence.

## See Also

- complexity-theory
- automata-theory
- information-theory
- lambda-calculus
- turing-machines

## References

- Boole, G. "An Investigation of the Laws of Thought" (1854)
- Stone, M. H. "The Theory of Representations for Boolean Algebras" (1936), Trans. AMS
- Shannon, C. E. "A Symbolic Analysis of Relay and Switching Circuits" (1937), master's thesis, MIT
- Davis, M. & Putnam, H. "A Computing Procedure for Quantification Theory" (1960), JACM
- Davis, M., Logemann, G. & Loveland, D. "A Machine Program for Theorem-Proving" (1962), CACM
- Cook, S. A. "The Complexity of Theorem-Proving Procedures" (1971), STOC
- Post, E. L. "The Two-Valued Iterative Systems of Mathematical Logic" (1941), Annals of Math Studies
- Bryant, R. E. "Graph-Based Algorithms for Boolean Function Manipulation" (1986), IEEE Trans. Computers
- Kahn, J., Kalai, G. & Linial, N. "The Influence of Variables on Boolean Functions" (1988), FOCS
- Biere, A. et al. "Handbook of Satisfiability" (2nd ed., IOS Press, 2021)
- Knuth, D. E. "The Art of Computer Programming, Vol. 4A: Combinatorial Algorithms" (Addison-Wesley, 2011)
