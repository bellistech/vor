# The Mathematics of Turing Machines -- Computability, Undecidability, and the Limits of Formal Systems

> *A Turing machine is the simplest possible device that captures the full power of algorithmic computation, yet even this power has absolute, provable boundaries — boundaries intimately connected to the logical incompleteness discovered by Godel.*

---

## 1. Formal Definition (The 7-Tuple)

### The Problem

Precisely define the Turing machine as a mathematical object, establishing the foundation for all results in computability theory.

### The Formula

A Turing machine is a 7-tuple $M = (Q, \Sigma, \Gamma, \delta, q_0, q_{accept}, q_{reject})$ where:

- $Q$ is a finite set of **states**
- $\Sigma$ is the **input alphabet** ($\text{blank} \notin \Sigma$)
- $\Gamma$ is the **tape alphabet** ($\Sigma \subset \Gamma$, $\text{blank} \in \Gamma$)
- $\delta: Q \times \Gamma \to Q \times \Gamma \times \{L, R\}$ is the **transition function**
- $q_0 \in Q$ is the **start state**
- $q_{accept} \in Q$ is the **accept state**
- $q_{reject} \in Q$ is the **reject state** ($q_{reject} \ne q_{accept}$)

A **configuration** of $M$ is a triple $(q, w, i)$ where $q \in Q$ is the current state, $w \in \Gamma^*$ is the tape contents, and $i$ is the head position. We write configurations as $uqv$ where $u$ is the tape content to the left of the head, $q$ is the current state, and $v$ is the tape content from the head position onward.

The **yields** relation $C_1 \vdash C_2$ holds when configuration $C_2$ follows from $C_1$ in one step of $M$.

$M$ **accepts** $w$ if there exists a sequence of configurations $C_0, C_1, \ldots, C_k$ where:
1. $C_0 = q_0 w$ (start configuration)
2. $C_i \vdash C_{i+1}$ for each $i$
3. $C_k$ contains $q_{accept}$

### Worked Example

A TM that decides $L = \{0^n 1^n : n \ge 0\}$:

```
States: {q0, q1, q2, q3, q4, q_accept, q_reject}
Sigma: {0, 1}
Gamma: {0, 1, X, Y, _}

Transitions:
  (q0, _) -> (q_accept, _, R)     -- empty string accepted
  (q0, Y) -> (q3, Y, R)           -- skip Ys to verify end
  (q0, 0) -> (q1, X, R)           -- mark a 0, go find matching 1
  (q1, 0) -> (q1, 0, R)           -- skip 0s
  (q1, Y) -> (q1, Y, R)           -- skip Ys
  (q1, 1) -> (q2, Y, L)           -- mark matching 1, go back
  (q2, 0) -> (q2, 0, L)           -- scan left past 0s
  (q2, Y) -> (q2, Y, L)           -- scan left past Ys
  (q2, X) -> (q0, X, R)           -- found marked 0, restart
  (q3, Y) -> (q3, Y, R)           -- skip remaining Ys
  (q3, _) -> (q_accept, _, R)     -- all matched
```

Step-by-step trace on input `0011`:

```
Step   Configuration          Action
----   ---------------------- --------------------------
  0    q0 [0] 0 1 1           Read 0, write X, move R
  1    X q1 [0] 1 1           Read 0, skip right
  2    X 0 q1 [1] 1           Read 1, write Y, move L
  3    X q2 [0] Y 1           Read 0, scan left
  4    q2 [X] 0 Y 1           Read X, move R
  5    X q0 [0] Y 1           Read 0, write X, move R
  6    X X q1 [Y] 1           Read Y, skip right
  7    X X Y q1 [1]           Read 1, write Y, move L
  8    X X q2 [Y] Y           Read Y, scan left
  9    X q2 [X] Y Y           Read X, move R
 10    X X q0 [Y] Y           Read Y, skip right (q3)
 10    X X q3 [Y] Y           Read Y, skip right
 11    X X Y q3 [Y]           Read Y, skip right
 12    X X Y Y q3 [_]         Read blank -> q_accept
```

Result: **Accept**. The string $0011 = 0^2 1^2$ is in $L$.

---

## 2. The Halting Problem (Undecidability by Diagonalization)

### The Problem

Prove that no Turing machine can decide whether an arbitrary Turing machine halts on a given input.

### The Formula

Define the language:

$$HALT = \{ \langle M, w \rangle : M \text{ is a TM and } M \text{ halts on input } w \}$$

**Theorem.** $HALT$ is undecidable.

**Proof.** Assume for contradiction that a TM $H$ decides $HALT$:

$$H(\langle M, w \rangle) = \begin{cases} \text{accept} & \text{if } M \text{ halts on } w \\ \text{reject} & \text{if } M \text{ loops on } w \end{cases}$$

Construct a new TM $D$ as follows. On input $\langle M \rangle$:

1. Run $H(\langle M, \langle M \rangle \rangle)$
2. If $H$ accepts (meaning $M$ halts on $\langle M \rangle$), then **loop forever**
3. If $H$ rejects (meaning $M$ loops on $\langle M \rangle$), then **halt and accept**

Now consider the behavior of $D$ on input $\langle D \rangle$:

- **Case 1:** $D$ halts on $\langle D \rangle$. Then $H(\langle D, \langle D \rangle \rangle)$ accepts, so $D$ loops. Contradiction.
- **Case 2:** $D$ loops on $\langle D \rangle$. Then $H(\langle D, \langle D \rangle \rangle)$ rejects, so $D$ halts. Contradiction.

Both cases lead to contradiction. Therefore, $H$ cannot exist. $\square$

### The Diagonalization Argument Visualized

```
              Input to machine:
              <M1>   <M2>   <M3>   <M4>   ...
Machine M1: [ halt   loop   halt   halt   ... ]
Machine M2: [ loop   halt   loop   halt   ... ]
Machine M3: [ halt   halt   loop   loop   ... ]
Machine M4: [ loop   loop   halt   halt   ... ]
  ...

Diagonal:   [ halt   halt   loop   halt   ... ]
                |      |      |      |
D flips:    [ loop   loop   halt   loop   ... ]

D differs from EVERY machine on the diagonal.
D cannot appear anywhere in the table.
But D is a valid TM -- contradiction with the assumption
that H could fill in this table.
```

This is structurally identical to Cantor's proof that $\mathbb{R}$ is uncountable.

---

## 3. The Church-Turing Thesis and Its Implications

### The Problem

Characterize the class of functions computable by any algorithmic process and understand why all reasonable models of computation are equivalent.

### The Formula

**Church-Turing Thesis (1936):** A function $f: \mathbb{N} \to \mathbb{N}$ is computable by an effective procedure if and only if it is computable by a Turing machine.

This is a **thesis**, not a theorem, because "effective procedure" is an informal notion. Evidence for the thesis:

| Model | Year | Equivalent to TM? |
|-------|------|--------------------|
| $\mu$-recursive functions (Godel-Herbrand) | 1931 | Yes |
| $\lambda$-calculus (Church) | 1936 | Yes |
| Turing machines (Turing) | 1936 | Yes |
| Post systems (Post) | 1936 | Yes |
| Markov algorithms | 1951 | Yes |
| Register machines | 1963 | Yes |
| Game of Life (Conway) | 1970 | Yes (Turing-complete) |
| RAM model | 1973 | Yes |
| Cellular automata (Rule 110) | 2004 | Yes (proved universal) |

**Implication 1 — Robustness:** Adding features to TMs (multiple tapes, nondeterminism, two-way infinite tape, multi-dimensional tape) does not change the class of computable functions.

**Implication 2 — Universality:** There exist universal machines that can simulate any other machine, given its description. This is the theoretical foundation for stored-program computers.

**Implication 3 — Absolute limits:** If a function cannot be computed by a Turing machine, it cannot be computed by ANY algorithmic means — regardless of programming language, hardware, or cleverness.

---

## 4. Godel's Incompleteness Theorems and Undecidability

### The Problem

Trace the deep connection between Godel's results on the limitations of formal systems and Turing's results on the limitations of computation.

### The Formula

**Godel's First Incompleteness Theorem (1931):** For any consistent formal system $F$ capable of expressing basic arithmetic, there exists a sentence $G_F$ such that:

$$F \nvdash G_F \quad \text{and} \quad F \nvdash \neg G_F$$

The sentence $G_F$ is true (in the standard model $\mathbb{N}$) but unprovable in $F$.

**Godel's Second Incompleteness Theorem:** No consistent system $F$ capable of expressing basic arithmetic can prove its own consistency:

$$F \nvdash \text{Con}(F)$$

**Connection to the Halting Problem:**

Godel's incompleteness can be derived from the undecidability of the halting problem. Consider a formal system $F$ and the following decision procedure: enumerate all proofs in $F$, checking if any proves "$M$ halts on $w$" or "$M$ does not halt on $w$." If $F$ were complete for halting statements, this enumerator would decide $HALT$. Since $HALT$ is undecidable, $F$ must be incomplete.

More precisely, Godel numbering provides the mechanism:

```
Godel numbering: assigns unique natural number to each:
  - Symbol:    '(' -> 1,  ')' -> 2,  '0' -> 3, ...
  - Formula:   product of primes raised to symbol codes
  - Proof:     product of primes raised to formula codes

This encoding lets a formal system "talk about itself."
A TM description <M> is essentially a Godel number.

Historical flow:
  1931  Godel: Formal systems cannot prove all truths
                (incompleteness)
  1936  Church: Lambda calculus cannot solve
                Entscheidungsproblem
  1936  Turing: No algorithm can solve the halting problem
                (undecidability)
  Both: formalized "what computation cannot do"
```

---

## 5. Equivalence of Computational Models

### The Problem

Prove that multi-tape TMs, nondeterministic TMs, and single-tape TMs all recognize exactly the same class of languages, establishing the robustness of the computability boundary.

### The Formula

**Theorem 1:** For every $k$-tape TM $M$ running in time $t(n)$, there exists a single-tape TM $S$ such that $L(S) = L(M)$ and $S$ runs in time $O(t(n)^2)$.

**Proof sketch:** $S$ stores all $k$ tapes on a single tape separated by delimiters. Each simulated step requires scanning the entire active portion of the tape (length $\le t(n)$) to find all $k$ head positions, then updating. Total: $t(n)$ steps, each costing $O(t(n))$ scan time.

```
Single tape simulating 3 tapes:

  # a b [c] d # X [Y] Z # _ [_] _ #
       tape 1      tape 2    tape 3

  [x] = head position marker (dotted symbol in Sipser notation)
  #   = tape delimiter
```

**Theorem 2:** For every nondeterministic TM $N$, there exists a deterministic TM $D$ such that $L(D) = L(N)$.

**Proof sketch:** $D$ performs a breadth-first search of $N$'s computation tree. If $N$ has at most $b$ choices at each step (branching factor), the tree at depth $d$ has at most $b^d$ nodes. $D$ systematically explores all paths using a 3-tape simulation: tape 1 = input, tape 2 = simulation tape, tape 3 = path address.

**Time complexity of simulation:** If $N$ runs in nondeterministic time $t(n)$, then $D$ runs in time $O(b^{t(n)} \cdot t(n))$ — exponential blowup.

**Theorem 3 (Equivalence summary):**

$$\text{Single-tape TM} \equiv \text{Multi-tape TM} \equiv \text{NTM} \equiv \lambda\text{-calculus} \equiv \mu\text{-recursive}$$

All recognize exactly the class of **recursively enumerable** (Turing-recognizable) languages.
All decide exactly the class of **recursive** (decidable) languages.

---

## 6. Complexity of Simulation and the Simulation Hierarchy

### The Problem

Analyze the computational overhead of simulating one model of computation with another, and establish the fundamental time and space hierarchies.

### The Formula

**Simulation costs:**

| Simulated Model | Simulator | Time Overhead |
|-----------------|-----------|---------------|
| $k$-tape TM ($t(n)$ steps) | 1-tape TM | $O(t(n)^2)$ |
| NTM ($t(n)$ steps, branching $b$) | Deterministic TM | $O(b^{t(n)})$ |
| 2-stack PDA | TM | $O(n)$ (direct) |
| TM ($t(n)$ steps) | RAM model | $O(t(n) \log t(n))$ |
| RAM ($t(n)$ steps) | TM | $O(t(n)^3)$ |

**Time Hierarchy Theorem:** For time-constructible $f$,

$$DTIME(o(f(n))) \subsetneq DTIME(f(n) \cdot \log^2 f(n))$$

There exist problems solvable in time $f(n)$ that cannot be solved in significantly less time. This gives us the proper containment:

$$P \subsetneq EXP$$

More time genuinely lets you solve more problems.

**Space Hierarchy Theorem:** For space-constructible $f$,

$$DSPACE(o(f(n))) \subsetneq DSPACE(f(n))$$

The space hierarchy is tighter (no log factor) because space can be reused.

**Universal TM simulation overhead:** A UTM simulating a TM $M$ with $s$ states and $g$ tape symbols on input of length $n$ runs in time $O(t(n) \cdot \log t(n))$ using Hennie-Stearns simulation (2-tape UTM). A single-tape UTM incurs $O(t(n)^2)$.

---

## 7. Rice's Theorem (Full Statement and Proof)

### The Problem

Prove that no algorithm can determine any non-trivial semantic property of the language recognized by a Turing machine.

### The Formula

**Definition.** A property $P$ of Turing-recognizable languages is a set of languages. $P$ is **non-trivial** if there exist TMs $M_1, M_2$ such that $L(M_1) \in P$ and $L(M_2) \notin P$.

**Rice's Theorem.** For every non-trivial property $P$:

$$L_P = \{ \langle M \rangle : L(M) \in P \}$$

is undecidable.

**Proof.** Assume WLOG that $\emptyset \notin P$ (if $\emptyset \in P$, consider $\overline{P}$). Since $P$ is non-trivial, there exists a TM $T$ with $L(T) \in P$.

Reduce $A_{TM}$ to $L_P$. Given $\langle M, w \rangle$, construct TM $M_w$:

```
M_w on input x:
  1. Simulate M on w
  2. If M accepts w, simulate T on x and output T's answer
  3. If M rejects w, reject
```

Analysis:
- If $M$ accepts $w$: $M_w$ simulates $T$, so $L(M_w) = L(T) \in P$, thus $\langle M_w \rangle \in L_P$
- If $M$ does not accept $w$: $L(M_w) = \emptyset \notin P$, thus $\langle M_w \rangle \notin L_P$

Therefore $\langle M, w \rangle \in A_{TM} \iff \langle M_w \rangle \in L_P$, which is a valid reduction from $A_{TM}$ to $L_P$. Since $A_{TM}$ is undecidable, $L_P$ is undecidable. $\square$

---

## Prerequisites

- Finite automata (DFA, NFA) and regular languages
- Context-free grammars and pushdown automata
- Basic set theory and proof techniques (contradiction, induction)
- Mathematical logic fundamentals (first-order logic, formal systems)
- Cantor's diagonalization argument

## Complexity

| Level | Description |
|-------|-------------|
| **Beginner** | Define TMs formally. Trace execution on simple inputs. Understand the difference between decidable and recognizable. State the halting problem and its significance. |
| **Intermediate** | Reproduce the halting problem proof. Perform mapping reductions between languages. Apply Rice's theorem to classify properties as decidable or undecidable. Simulate multi-tape TMs on a single tape. |
| **Advanced** | Prove equivalence of computational models. Connect Godel's incompleteness to undecidability. Analyze simulation overhead and hierarchy theorems. Construct oracle TMs and study the arithmetical hierarchy. Explore hypercomputation proposals and their relationship to the Church-Turing thesis. |
