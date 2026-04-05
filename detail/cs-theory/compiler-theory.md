# The Theory of Compilation -- From Formal Languages to Executable Code

> *A compiler is a constructive proof that the source language can be faithfully embedded into the target language, with every phase corresponding to a well-founded transformation on formal structures: regular sets, context-free grammars, type judgments, lattice-theoretic fixed points, and graph-theoretic colorings.*

---

## 1. FIRST and FOLLOW Set Computation

### The Problem

Given a context-free grammar $G = (V, T, P, S)$, compute the FIRST and FOLLOW sets needed for predictive parsing. These sets determine which production to apply given the current nonterminal and lookahead token.

### FIRST Set Algorithm

**Definition.** $\text{FIRST}(\alpha)$ for a string $\alpha \in (V \cup T)^*$ is the set of terminals that can begin strings derived from $\alpha$, plus $\epsilon$ if $\alpha \Rightarrow^* \epsilon$.

**Algorithm (iterative fixed point):**

Initialize $\text{FIRST}(a) = \{a\}$ for all terminals $a$, and $\text{FIRST}(A) = \emptyset$ for all nonterminals $A$.

Repeat until no changes:

For each production $A \to Y_1 Y_2 \cdots Y_k$:

$$\text{FIRST}(A) \gets \text{FIRST}(A) \cup \left(\text{FIRST}(Y_1) \setminus \{\epsilon\}\right)$$

If $\epsilon \in \text{FIRST}(Y_1)$:

$$\text{FIRST}(A) \gets \text{FIRST}(A) \cup \left(\text{FIRST}(Y_2) \setminus \{\epsilon\}\right)$$

Continue for $Y_3, Y_4, \ldots$ as long as each preceding $Y_i$ derives $\epsilon$.

If $\epsilon \in \text{FIRST}(Y_i)$ for all $1 \le i \le k$, then add $\epsilon$ to $\text{FIRST}(A)$.

**Extension to strings:** For $\alpha = X_1 X_2 \cdots X_n$:

$$\text{FIRST}(X_1 X_2 \cdots X_n) = \begin{cases} (\text{FIRST}(X_1) \setminus \{\epsilon\}) \cup \text{FIRST}(X_2 \cdots X_n) & \text{if } \epsilon \in \text{FIRST}(X_1) \\ \text{FIRST}(X_1) & \text{otherwise} \end{cases}$$

### FOLLOW Set Algorithm

**Definition.** $\text{FOLLOW}(A)$ for nonterminal $A$ is the set of terminals that can appear immediately to the right of $A$ in some sentential form. $\$ \in \text{FOLLOW}(S)$ always.

**Algorithm (iterative fixed point):**

Initialize $\text{FOLLOW}(S) = \{\$\}$ and $\text{FOLLOW}(A) = \emptyset$ for all other nonterminals.

Repeat until no changes:

For each production $A \to \alpha B \beta$:

$$\text{FOLLOW}(B) \gets \text{FOLLOW}(B) \cup \left(\text{FIRST}(\beta) \setminus \{\epsilon\}\right)$$

If $\epsilon \in \text{FIRST}(\beta)$ or $\beta = \epsilon$ (i.e., $B$ is at the end):

$$\text{FOLLOW}(B) \gets \text{FOLLOW}(B) \cup \text{FOLLOW}(A)$$

### Worked Example

Grammar:

$$E \to T E' \qquad E' \to + T E' \mid \epsilon \qquad T \to F T' \qquad T' \to * F T' \mid \epsilon \qquad F \to ( E ) \mid \text{id}$$

**FIRST sets:**

$$\text{FIRST}(F) = \{(, \text{id}\}$$
$$\text{FIRST}(T') = \{*, \epsilon\}$$
$$\text{FIRST}(T) = \text{FIRST}(F) = \{(, \text{id}\}$$
$$\text{FIRST}(E') = \{+, \epsilon\}$$
$$\text{FIRST}(E) = \text{FIRST}(T) = \{(, \text{id}\}$$

**FOLLOW sets:**

$$\text{FOLLOW}(E) = \{\$, )\}$$

From $E \to T E'$: $\text{FOLLOW}(E') = \text{FOLLOW}(E) = \{\$, )\}$

From $E \to T E'$: $\text{FOLLOW}(T) \supseteq (\text{FIRST}(E') \setminus \{\epsilon\}) \cup \text{FOLLOW}(E) = \{+, \$, )\}$

From $T \to F T'$: $\text{FOLLOW}(T') = \text{FOLLOW}(T) = \{+, \$, )\}$

From $T \to F T'$: $\text{FOLLOW}(F) \supseteq (\text{FIRST}(T') \setminus \{\epsilon\}) \cup \text{FOLLOW}(T) = \{*, +, \$, )\}$

---

## 2. LL(1) Parse Table Construction

### The Problem

Construct a predictive parsing table $M[A, a]$ that determines, for nonterminal $A$ and lookahead terminal $a$, which production to apply. The grammar is LL(1) if and only if no cell contains more than one production.

### The Construction

For each production $A \to \alpha$:

1. For each terminal $a \in \text{FIRST}(\alpha)$, set $M[A, a] = A \to \alpha$.
2. If $\epsilon \in \text{FIRST}(\alpha)$, then for each terminal $b \in \text{FOLLOW}(A)$, set $M[A, b] = A \to \alpha$.
3. If $\epsilon \in \text{FIRST}(\alpha)$ and $\$ \in \text{FOLLOW}(A)$, set $M[A, \$] = A \to \alpha$.

### LL(1) Condition

A grammar is LL(1) if and only if for every pair of productions $A \to \alpha \mid \beta$:

$$\text{FIRST}(\alpha) \cap \text{FIRST}(\beta) = \emptyset$$

and if $\epsilon \in \text{FIRST}(\alpha)$, then:

$$\text{FIRST}(\beta) \cap \text{FOLLOW}(A) = \emptyset$$

### Worked Example (continued)

Parse table for the expression grammar:

```
         id        +         *         (         )         $
E     E->TE'                        E->TE'
E'              E'->+TE'                      E'->eps   E'->eps
T     T->FT'                        T->FT'
T'              T'->eps   T'->*FT'            T'->eps   T'->eps
F     F->id                         F->(E)
```

**Parsing "id + id * id":**

```
Stack              Input              Action
$E                 id + id * id $     E -> T E'
$E'T               id + id * id $     T -> F T'
$E'T'F             id + id * id $     F -> id
$E'T'              + id * id $        T' -> eps
$E'                + id * id $        E' -> + T E'
$E'T               id * id $          T -> F T'
$E'T'F             id * id $          F -> id
$E'T'              * id $             T' -> * F T'
$E'T'F             id $               F -> id
$E'T'              $                  T' -> eps
$E'                $                  E' -> eps
$                  $                  ACCEPT
```

---

## 3. LR(1) Parsing Automaton Construction

### The Problem

Construct the canonical LR(1) parsing automaton: a DFA whose states are sets of LR(1) items, each item being a production with a dot position and a lookahead terminal.

### LR(1) Items

An LR(1) item is a pair $[A \to \alpha \cdot \beta, a]$ where:

- $A \to \alpha \beta$ is a production
- The dot indicates how much of the RHS has been matched
- $a \in T \cup \{\$\}$ is the lookahead

### Closure Operation

Given a set of items $I$:

$$\text{CLOSURE}(I) = I \cup \left\{ [B \to \cdot \gamma, b] \;\middle|\; [A \to \alpha \cdot B \beta, a] \in \text{CLOSURE}(I), \; B \to \gamma \in P, \; b \in \text{FIRST}(\beta a) \right\}$$

The key insight: the lookahead $b$ for the new item $[B \to \cdot \gamma, b]$ comes from $\text{FIRST}(\beta a)$, where $\beta$ is whatever follows $B$ in the original item and $a$ is the original lookahead.

### Goto Operation

$$\text{GOTO}(I, X) = \text{CLOSURE}\left(\left\{ [A \to \alpha X \cdot \beta, a] \;\middle|\; [A \to \alpha \cdot X \beta, a] \in I \right\}\right)$$

### Automaton Construction

1. Start with $I_0 = \text{CLOSURE}(\{[S' \to \cdot S, \$]\})$ where $S' \to S$ is the augmented start production.
2. For each state $I_i$ and each grammar symbol $X$, compute $\text{GOTO}(I_i, X)$. If this set is nonempty and not already a state, add it.
3. Repeat until no new states are generated.

### Action and Goto Tables

For state $I_i$:

- If $[A \to \alpha \cdot a \beta, b] \in I_i$ and $a$ is a terminal: $\text{ACTION}[i, a] = \text{shift } j$ where $I_j = \text{GOTO}(I_i, a)$.
- If $[A \to \alpha \cdot, a] \in I_i$ and $A \ne S'$: $\text{ACTION}[i, a] = \text{reduce } A \to \alpha$.
- If $[S' \to S \cdot, \$] \in I_i$: $\text{ACTION}[i, \$] = \text{accept}$.
- If $\text{GOTO}(I_i, A) = I_j$ for nonterminal $A$: $\text{GOTO}[i, A] = j$.

### LALR(1) from LR(1)

LALR(1) states are obtained by merging LR(1) states that share the same LR(0) core (same items ignoring lookaheads). The lookahead sets are unioned. This can introduce reduce/reduce conflicts not present in LR(1), but never shift/reduce conflicts.

### Worked Example

Grammar: $S \to C C$, $C \to c C \mid d$.

Augmented: $S' \to S$.

**State $I_0$:**

```
[S' -> .S,   $]
[S  -> .CC,  $]
[C  -> .cC,  c/d]
[C  -> .d,   c/d]
```

**State $I_1$ = GOTO($I_0$, S):**

```
[S' -> S.,   $]        --> accept
```

**State $I_2$ = GOTO($I_0$, C):**

```
[S  -> C.C,  $]
[C  -> .cC,  $]
[C  -> .d,   $]
```

**State $I_3$ = GOTO($I_0$, c):**

```
[C  -> c.C,  c/d]
[C  -> .cC,  c/d]
[C  -> .d,   c/d]
```

**State $I_4$ = GOTO($I_0$, d):**

```
[C  -> d.,   c/d]      --> reduce C -> d on c or d
```

**State $I_5$ = GOTO($I_2$, C):**

```
[S  -> CC.,  $]        --> reduce S -> CC on $
```

**State $I_6$ = GOTO($I_2$, c):**

```
[C  -> c.C,  $]
[C  -> .cC,  $]
[C  -> .d,   $]
```

**State $I_7$ = GOTO($I_2$, d):**

```
[C  -> d.,   $]        --> reduce C -> d on $
```

**State $I_8$ = GOTO($I_3$, C):**

```
[C  -> cC.,  c/d]      --> reduce C -> cC on c or d
```

**State $I_9$ = GOTO($I_6$, C):**

```
[C  -> cC.,  $]        --> reduce C -> cC on $
```

Note: GOTO($I_3$, c) = $I_3$, GOTO($I_3$, d) = $I_4$, GOTO($I_6$, c) = $I_6$, GOTO($I_6$, d) = $I_7$.

States $I_3$ and $I_6$ have the same LR(0) core but different lookaheads. In LALR(1), they merge into a single state with lookahead $\{c, d, \$\}$. Similarly $I_4$ and $I_7$ merge, and $I_8$ and $I_9$ merge.

---

## 4. SSA Construction Algorithm

### The Problem

Transform a program in conventional IR into Static Single Assignment form, where every variable is assigned exactly once and phi-functions merge values at control flow join points.

### Dominance

A node $d$ **dominates** node $n$ (written $d \;\text{dom}\; n$) if every path from the entry node to $n$ passes through $d$. The **immediate dominator** $\text{idom}(n)$ is the closest strict dominator of $n$. The dominator tree has edges from $\text{idom}(n)$ to $n$.

**Cooper-Harvey-Kennedy algorithm** for computing dominators in almost-linear time:

```
for all nodes n: doms[n] = undefined
doms[entry] = entry
changed = true
while changed:
    changed = false
    for each node b in reverse postorder (except entry):
        new_idom = first processed predecessor of b
        for each other predecessor p of b:
            if doms[p] != undefined:
                new_idom = intersect(p, new_idom)
        if doms[b] != new_idom:
            doms[b] = new_idom
            changed = true

intersect(b1, b2):
    finger1 = b1, finger2 = b2
    while finger1 != finger2:
        while finger1 < finger2: finger1 = doms[finger1]
        while finger2 < finger1: finger2 = doms[finger2]
    return finger1
```

### Dominance Frontiers

The **dominance frontier** of node $d$, written $\text{DF}(d)$, is:

$$\text{DF}(d) = \{ n \mid \exists \text{ predecessor } p \text{ of } n \text{ such that } d \;\text{dom}\; p \text{ but } d \not\!\;\text{sdom}\; n \}$$

where $\text{sdom}$ means strict dominance ($d \;\text{sdom}\; n$ iff $d \;\text{dom}\; n$ and $d \ne n$).

Intuitively: $\text{DF}(d)$ is the "border" where $d$'s dominance ends. These are exactly the nodes where phi-functions for variables defined at $d$ might be needed.

**Algorithm (Cytron et al.):**

```
for each node n:
    DF[n] = {}

for each node n:
    if n has multiple predecessors:
        for each predecessor p of n:
            runner = p
            while runner != idom[n]:
                DF[runner] = DF[runner] + {n}
                runner = idom[runner]
```

### Phi-Function Placement (Iterated Dominance Frontier)

For each variable $v$, let $\text{Defs}(v)$ be the set of nodes that contain a definition of $v$. Phi-functions for $v$ must be placed at the **iterated dominance frontier**:

$$\text{DF}^+(S) = \lim_{i \to \infty} \text{DF}^i(S)$$

where $\text{DF}^1(S) = \bigcup_{n \in S} \text{DF}(n)$ and $\text{DF}^{i+1}(S) = \text{DF}^1(S \cup \text{DF}^i(S))$.

**Algorithm:**

```
for each variable v:
    worklist = Defs(v)
    ever_on_worklist = Defs(v)
    while worklist is not empty:
        remove some node n from worklist
        for each node d in DF[n]:
            if d does not already have a phi for v:
                insert "v = phi(v, v, ..., v)" at top of d
                    (one argument per predecessor of d)
                if d not in ever_on_worklist:
                    ever_on_worklist = ever_on_worklist + {d}
                    add d to worklist
```

### Variable Renaming

After phi placement, rename all variables so each definition gets a unique subscript. This is done by a preorder walk of the dominator tree:

```
counter[v] = 0 for all v
stack[v] = [] for all v

rename(block):
    for each phi-function "v = phi(...)" in block:
        i = counter[v]++
        replace LHS with v_i
        push i onto stack[v]

    for each instruction in block:
        for each use of variable v:
            replace with v_{top(stack[v])}
        for each definition of variable v:
            i = counter[v]++
            replace with v_i
            push i onto stack[v]

    for each successor s of block:
        j = index of block in predecessors of s
        for each phi "v_k = phi(...)" in s:
            set j-th argument to v_{top(stack[v])}

    for each child c of block in dominator tree:
        rename(c)

    pop all items pushed onto stacks in this call
```

---

## 5. Data Flow Analysis Framework

### The Problem

Define a general framework for computing program properties (reaching definitions, available expressions, liveness, etc.) as fixed points on lattices.

### The Lattice-Theoretic Framework

A **data flow analysis** is a tuple $(L, \sqcap, F, \iota)$ where:

- $(L, \sqsubseteq)$ is a complete lattice with meet $\sqcap$ and join $\sqcup$
- $F : L \to L$ is a monotone transfer function ($x \sqsubseteq y \implies F(x) \sqsubseteq F(y)$)
- $\iota \in L$ is the initial value (boundary condition)

The analysis computes $\text{OUT}[B]$ (or $\text{IN}[B]$) for each basic block $B$ by solving:

**Forward analysis:**

$$\text{IN}[B] = \bigsqcap_{P \in \text{pred}(B)} \text{OUT}[P]$$
$$\text{OUT}[B] = F_B(\text{IN}[B])$$

**Backward analysis:**

$$\text{OUT}[B] = \bigsqcap_{S \in \text{succ}(B)} \text{IN}[S]$$
$$\text{IN}[B] = F_B(\text{OUT}[B])$$

### Transfer Functions

For a basic block $B$ with gen set $\text{gen}(B)$ and kill set $\text{kill}(B)$:

$$F_B(x) = \text{gen}(B) \cup (x \setminus \text{kill}(B))$$

| Analysis | Direction | Meet | Lattice | Gen | Kill |
|----------|-----------|------|---------|-----|------|
| Reaching Definitions | Forward | $\cup$ | $(\mathcal{P}(\text{Defs}), \subseteq)$ | Defs in $B$ | Defs killed by $B$ |
| Available Expressions | Forward | $\cap$ | $(\mathcal{P}(\text{Exprs}), \supseteq)$ | Exprs computed in $B$ | Exprs invalidated by $B$ |
| Live Variables | Backward | $\cup$ | $(\mathcal{P}(\text{Vars}), \subseteq)$ | Vars used before def in $B$ | Vars defined in $B$ |
| Very Busy Expressions | Backward | $\cap$ | $(\mathcal{P}(\text{Exprs}), \supseteq)$ | Exprs used before killed | Exprs killed in $B$ |

### Fixed-Point Iteration (Chaotic Iteration)

```
Initialize: OUT[entry] = iota
            OUT[B] = top for all other B  (or bottom, depending on direction)

Repeat:
    for each block B in some order:
        IN[B] = meet over all predecessors
        new = F_B(IN[B])
        if new != OUT[B]:
            OUT[B] = new
            mark successors for revisiting

Until no changes (fixed point reached).
```

**Theorem (Kam-Ullman).** If the lattice has finite height and all transfer functions are monotone, chaotic iteration converges to the **Maximum Fixed Point (MFP)**. If transfer functions are additionally distributive ($F(x \sqcap y) = F(x) \sqcap F(y)$), then the MFP equals the **Meet Over all Paths (MOP)** solution, which is the ideal solution.

### Worked Example: Reaching Definitions

```
Block B1: d1: x = 5       gen={d1}, kill={d2,d4}
          d2: y = 1
Block B2: d3: z = x + y   gen={d3}, kill={}
Block B3: d4: x = z + 1   gen={d4}, kill={d1}
          goto B2

CFG: B1 -> B2 -> B3 -> B2

Iteration 0: OUT[B1]={d1,d2}, OUT[B2]={}, OUT[B3]={}

Iteration 1:
  IN[B2] = OUT[B1] + OUT[B3] = {d1,d2}
  OUT[B2] = {d3} + ({d1,d2} - {}) = {d1,d2,d3}
  IN[B3] = OUT[B2] = {d1,d2,d3}
  OUT[B3] = {d4} + ({d1,d2,d3} - {d1}) = {d2,d3,d4}

Iteration 2:
  IN[B2] = {d1,d2} + {d2,d3,d4} = {d1,d2,d3,d4}
  OUT[B2] = {d3} + {d1,d2,d3,d4} = {d1,d2,d3,d4}
  IN[B3] = {d1,d2,d3,d4}
  OUT[B3] = {d4} + ({d1,d2,d3,d4} - {d1}) = {d2,d3,d4}

Iteration 3: no changes -> fixed point.
```

---

## 6. Chaitin's Graph Coloring Register Allocation

### The Problem

Map an unbounded number of virtual registers (temporaries) to a fixed number $K$ of physical machine registers, inserting spill code (loads and stores) where necessary.

### The Interference Graph

The **interference graph** $G = (V, E)$ has:

- $V$ = set of virtual registers (live ranges)
- $E$ = $\{(u, v) \mid u \text{ and } v \text{ are simultaneously live at some program point}\}$

Two virtual registers interfere if and only if their live ranges overlap. Interfering registers cannot share the same physical register.

### Chaitin's Algorithm

**Phase 1: Build** -- Compute liveness information and construct the interference graph.

**Phase 2: Simplify** -- Repeatedly remove nodes with degree $< K$ from the graph, pushing them onto a stack. Removing a node cannot increase the degree of remaining nodes, so this preserves colorability.

**Phase 3: Potential Spill** -- If all remaining nodes have degree $\ge K$, select a node to mark as a potential spill. Heuristics for selection:

$$\text{priority}(v) = \frac{\text{spill cost}(v)}{\text{degree}(v)}$$

where spill cost accounts for the number of uses/definitions weighted by loop nesting depth:

$$\text{spill cost}(v) = \sum_{\text{def/use of } v} 10^{\text{loop depth}}$$

Choose the node with minimum priority (cheapest to spill relative to graph reduction benefit). Push it onto the stack and continue simplification.

**Phase 4: Select** -- Pop nodes from the stack and assign colors (physical registers). For each node, choose a color not used by any already-colored neighbor. If no color is available for a potential spill node, it becomes an **actual spill**.

**Phase 5: Spill Code** -- For each actual spill, insert store instructions after each definition and load instructions before each use. This splits the live range into many short segments. Then restart the entire algorithm (rebuild, simplify, select) with the new code.

### Convergence

Spilling shortens live ranges, reducing interference. In the worst case, every live range is length 1 (a single instruction), which always admits a $K$-coloring for $K \ge 1$. Chaitin's algorithm therefore always terminates.

### Coalescing (Chaitin-Briggs Enhancement)

**Move coalescing** attempts to eliminate register-to-register copies by merging the source and destination into a single node. Merge $u$ and $v$ (connected by a copy) if:

**Briggs criterion:** The merged node $uv$ has fewer than $K$ neighbors of significant degree ($\ge K$).

**George criterion:** Every neighbor $t$ of $u$ either already interferes with $v$ or has degree $< K$.

If coalescing succeeds, the copy instruction is eliminated.

### Worked Example

Suppose $K = 3$ (three physical registers: r0, r1, r2) and the interference graph is:

```
    a --- b
    |   / |
    |  /  |
    | /   |
    c --- d
    |
    e
```

Degrees: $a=2, b=3, c=4, d=2, e=1$.

**Simplify:** Remove $e$ (degree 1 < 3). Remove $a$ (degree 1 < 3 after removing $e$'s effect... actually $a$ was degree 2 and shares no edge with $e$, so $a$ still has degree 2 < 3). Remove $d$ (degree 1 after $a$ removed). Remove $b$ (degree 1). Remove $c$ (degree 0).

Stack (top to bottom): $c, b, d, a, e$.

**Select:** Pop $c$: assign r0. Pop $b$: neighbors $\{c=\text{r0}\}$ in colored graph, assign r1. Pop $d$: neighbors $\{b=\text{r1}, c=\text{r0}\}$, assign r2. Pop $a$: neighbors $\{b=\text{r1}, c=\text{r0}\}$, assign r2. Pop $e$: neighbors $\{c=\text{r0}\}$, assign r1.

Result: $a \to \text{r2}, b \to \text{r1}, c \to \text{r0}, d \to \text{r2}, e \to \text{r1}$.

No spills needed.

---

## 7. Partial Evaluation and the Futamura Projections

### The Problem

Given a program $p$ and its input divided into **static** (known at compile time) and **dynamic** (known only at run time) parts, specialize $p$ with respect to the static input to produce a faster **residual program**.

### Partial Evaluation

A **partial evaluator** (also called a **specializer**) is a program $\text{mix}$ such that:

$$\text{mix}(p, s) = p_s$$

where $p_s$ is the residual program satisfying:

$$\forall d.\; p(s, d) = p_s(d)$$

The partial evaluator evaluates all computations depending only on $s$ and generates code for computations depending on $d$.

### The Three Futamura Projections

Let $\text{int}$ be an interpreter for language $L$ written in language $S$, and let $\text{src}$ be a source program in $L$.

**First Futamura Projection -- Compilation:**

$$\text{mix}(\text{int}, \text{src}) = \text{target}$$

Specializing the interpreter with respect to the source program produces a target program (compiled code). The interpreter's dispatch loop is unfolded with respect to the known source, eliminating interpretation overhead.

$$\forall d.\; \text{int}(\text{src}, d) = \text{target}(d)$$

**Second Futamura Projection -- Compiler Generation:**

$$\text{mix}(\text{mix}, \text{int}) = \text{compiler}$$

Specializing the partial evaluator with respect to the interpreter produces a compiler. This compiler, when applied to any source program, produces target code.

$$\forall \text{src}.\; \text{compiler}(\text{src}) = \text{mix}(\text{int}, \text{src}) = \text{target}$$

**Third Futamura Projection -- Compiler Generator Generation:**

$$\text{mix}(\text{mix}, \text{mix}) = \text{cogen}$$

Specializing the partial evaluator with respect to itself produces a compiler generator. Given any interpreter, it produces a compiler.

$$\forall \text{int}.\; \text{cogen}(\text{int}) = \text{mix}(\text{mix}, \text{int}) = \text{compiler}$$

### The Binding-Time Analysis

Before specialization, a **binding-time analysis (BTA)** classifies each variable and expression as either:

- **Static (S):** value known from the static input alone -- evaluate at specialization time
- **Dynamic (D):** depends on dynamic input -- generate code in the residual program

Rules (abstract interpretation over $\{S, D\}$):

$$S \;\text{op}\; S = S \qquad S \;\text{op}\; D = D \qquad D \;\text{op}\; S = D \qquad D \;\text{op}\; D = D$$

Conditional on a static test: both branches evaluated. Conditional on a dynamic test: both branches residualized, values at join points become dynamic.

### Significance

The Futamura projections demonstrate that compilation, compiler generation, and compiler-generator generation are all instances of a single concept: partial evaluation. This provides a deep unifying principle connecting interpreters and compilers, showing that the boundary between "interpretation" and "compilation" is a matter of staging rather than a fundamental distinction.

---

## 8. Advanced Topics in Instruction Selection

### Tree Pattern Matching (Maximal Munch)

Instruction selection by dynamic programming on expression trees. Each IR tree node is annotated with the cheapest covering:

$$\text{cost}(n) = \min_{\text{rule } r \text{ matching at } n} \left( \text{cost}(r) + \sum_{i} \text{cost}(\text{child}_i(n, r)) \right)$$

where $\text{child}_i(n, r)$ denotes the subtrees left uncovered by rule $r$ at node $n$.

**BURS (Bottom-Up Rewrite System):** Precompute all possible matches at each node bottom-up, then select the minimum-cost cover top-down. Runs in $O(n)$ time on the tree size.

### Peephole Optimization

Post-code-generation local optimization over sliding windows of instructions:

```
Pattern                         Replacement
-------                         -----------
LOAD r1, [addr]                 (eliminated if r1 not used
STORE [addr], r1                 before next store to addr)

MOV r1, r2                      (eliminated -- use r2 directly)
... (r1 not redefined) ...       substitute r2 for r1

ADD r1, r1, #0                  (eliminated -- identity)
MUL r1, r1, #1                  (eliminated -- identity)
MUL r1, r1, #2                  SHL r1, r1, #1
```

---

## References

- Aho, Lam, Sethi, Ullman. *Compilers: Principles, Techniques, and Tools* (2nd ed.), Pearson, 2006.
- Appel. *Modern Compiler Implementation in ML*, Cambridge University Press, 1998.
- Cooper, Torczon. *Engineering a Compiler* (3rd ed.), Morgan Kaufmann, 2022.
- Muchnick. *Advanced Compiler Design and Implementation*, Morgan Kaufmann, 1997.
- Cytron, Ferrante, Rosen, Wegman, Zadeck. "Efficiently Computing Static Single Assignment Form and the Control Dependence Graph." *ACM TOPLAS* 13(4):451--490, 1991.
- Chaitin. "Register Allocation and Spilling via Graph Coloring." *SIGPLAN Notices* 17(6):98--105, 1982.
- Briggs, Cooper, Torczon. "Improvements to Graph Coloring Register Allocation." *ACM TOPLAS* 16(3):428--455, 1994.
- Knuth. "On the Translation of Languages from Left to Right." *Information and Control* 8(6):607--639, 1965.
- Futamura. "Partial Evaluation of Computation Process -- An Approach to a Compiler-Compiler." *Systems, Computers, Controls* 2(5):45--50, 1971.
- Jones, Gomard, Sestoft. *Partial Evaluation and Automatic Program Generation*, Prentice Hall, 1993.
- Kam, Ullman. "Monotone Data Flow Analysis Frameworks." *Acta Informatica* 7:305--317, 1977.
- Cooper, Harvey, Kennedy. "A Simple, Fast Dominance Algorithm." *Software Practice and Experience*, 2001.
