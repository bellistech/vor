# The Mathematics of Automata Theory -- Formal Languages, Grammars, and the Limits of Computation

> *Every computational model defines a frontier: the class of languages it recognizes partitions all possible problems into the decidable and the undecidable, the tractable and the intractable, giving rise to the deepest structure in theoretical computer science.*

---

## 1. Formal Definitions of Finite Automata

### The Problem

Define DFA and NFA precisely, establish their equivalence, and characterize the gap between determinism and nondeterminism for finite-state machines.

### The Formula

A **Deterministic Finite Automaton** is a 5-tuple $M = (Q, \Sigma, \delta, q_0, F)$ where:

- $Q$ is a finite set of states
- $\Sigma$ is a finite input alphabet
- $\delta : Q \times \Sigma \to Q$ is the transition function
- $q_0 \in Q$ is the start state
- $F \subseteq Q$ is the set of accept states

The **extended transition function** $\hat{\delta} : Q \times \Sigma^* \to Q$ is defined inductively:

$$\hat{\delta}(q, \epsilon) = q$$
$$\hat{\delta}(q, wa) = \delta(\hat{\delta}(q, w), a)$$

$M$ accepts $w$ if and only if $\hat{\delta}(q_0, w) \in F$.

A **Nondeterministic Finite Automaton** is a 5-tuple $N = (Q, \Sigma, \delta, q_0, F)$ where:

$$\delta : Q \times (\Sigma \cup \{\epsilon\}) \to \mathcal{P}(Q)$$

The NFA accepts $w$ if there exists at least one computation path ending in a state $q \in F$.

### The Epsilon-Closure

For a set of states $S \subseteq Q$:

$$\text{ECLOSE}(S) = \{ q \mid q \text{ is reachable from some } s \in S \text{ via zero or more } \epsilon\text{-transitions} \}$$

Computed by BFS/DFS on epsilon-edges alone.

---

## 2. NFA-to-DFA Subset Construction

### The Problem

Given an NFA $N = (Q_N, \Sigma, \delta_N, q_0, F_N)$, construct an equivalent DFA $D = (Q_D, \Sigma, \delta_D, d_0, F_D)$.

### The Construction

$$Q_D = \mathcal{P}(Q_N) \quad \text{(each DFA state is a set of NFA states)}$$
$$d_0 = \text{ECLOSE}(\{q_0\})$$
$$\delta_D(S, a) = \text{ECLOSE}\left(\bigcup_{q \in S} \delta_N(q, a)\right)$$
$$F_D = \{ S \in Q_D \mid S \cap F_N \neq \emptyset \}$$

### Worked Example

NFA over $\{0, 1\}$ recognizing strings ending in $01$:

```
States: {q0, q1, q2}
Start: q0, Accept: {q2}
Transitions:
  delta(q0, 0) = {q0, q1}    delta(q0, 1) = {q0}
  delta(q1, 1) = {q2}
  (no epsilon transitions)
```

Subset construction:

```
DFA states (subsets of {q0, q1, q2}):

  {q0}       --0--> {q0, q1}
  {q0}       --1--> {q0}
  {q0, q1}   --0--> {q0, q1}
  {q0, q1}   --1--> {q0, q2}    <-- accept (contains q2)
  {q0, q2}   --0--> {q0, q1}
  {q0, q2}   --1--> {q0}

Reachable DFA states: {q0}, {q0,q1}, {q0,q2}
Accept states: {q0,q2}
```

Renaming $A = \{q0\}$, $B = \{q0, q1\}$, $C = \{q0, q2\}$:

```
     0     1
A -> B     A
B -> B     C*
C -> B     A

Start: A, Accept: {C}
```

**Worst-case blowup:** An NFA with $n$ states can require a DFA with $2^n$ states. This bound is tight (consider the language "the $n$-th symbol from the end is 1").

---

## 3. The Myhill-Nerode Theorem

### The Problem

Characterize regular languages algebraically without reference to any automaton, and prove that the minimal DFA is unique.

### The Formula

Define the **Myhill-Nerode equivalence relation** $\equiv_L$ on $\Sigma^*$:

$$x \equiv_L y \iff (\forall z \in \Sigma^*)[xz \in L \iff yz \in L]$$

**Theorem (Myhill-Nerode).** A language $L$ is regular if and only if $\equiv_L$ has finitely many equivalence classes. Moreover, the number of equivalence classes equals the number of states in the minimal DFA for $L$.

### Application

For $L = \{a^n b^n \mid n \ge 0\}$, consider the strings $a, a^2, a^3, \ldots$

For $a^i$ and $a^j$ with $i \neq j$, take $z = b^i$. Then $a^i b^i \in L$ but $a^j b^i \notin L$. So $a^i \not\equiv_L a^j$ for all $i \neq j$, giving infinitely many equivalence classes. Therefore $L$ is not regular.

This is strictly stronger than the pumping lemma: every non-regular language is detected by Myhill-Nerode, but some non-regular languages satisfy the pumping lemma vacuously.

---

## 4. Pushdown Automata -- Formal Definition

### The Problem

Define the pushdown automaton and establish its equivalence with context-free grammars.

### The Formula

A **Pushdown Automaton** is a 6-tuple $M = (Q, \Sigma, \Gamma, \delta, q_0, F)$ where:

- $Q$ is a finite set of states
- $\Sigma$ is a finite input alphabet
- $\Gamma$ is a finite stack alphabet
- $\delta : Q \times (\Sigma \cup \{\epsilon\}) \times (\Gamma \cup \{\epsilon\}) \to \mathcal{P}(Q \times (\Gamma \cup \{\epsilon\}))$ is the transition function
- $q_0 \in Q$ is the start state
- $F \subseteq Q$ is the set of accept states

A configuration (instantaneous description) is a triple $(q, w, \gamma) \in Q \times \Sigma^* \times \Gamma^*$.

The transition $(q', b) \in \delta(q, a, t)$ means: in state $q$, reading input $a$ (or $\epsilon$), with $t$ on top of stack (or $\epsilon$ for no pop), move to $q'$ and push $b$ (or $\epsilon$ for no push).

**Equivalence Theorem:** A language is context-free if and only if some PDA recognizes it.

**Acceptance modes:**
- Accept by final state: $L(M) = \{ w \mid (q_0, w, Z_0) \vdash^* (q_f, \epsilon, \gamma) \text{ for some } q_f \in F \}$
- Accept by empty stack: $N(M) = \{ w \mid (q_0, w, Z_0) \vdash^* (q, \epsilon, \epsilon) \}$

Both modes are equivalent in power.

---

## 5. Closure Properties

### The Problem

For each level of the Chomsky hierarchy, determine which operations preserve membership in that language class.

### Closure Properties Table

| Operation | Regular | CFL | CSL | RE |
|-----------|---------|-----|-----|-----|
| Union | Yes | Yes | Yes | Yes |
| Intersection | Yes | **No** | Yes | Yes |
| Complement | Yes | **No** | Yes | **No** |
| Concatenation | Yes | Yes | Yes | Yes |
| Kleene Star | Yes | Yes | Yes | Yes |
| Intersection with Regular | Yes | Yes | Yes | Yes |
| Homomorphism | Yes | Yes | **No** | Yes |
| Inverse Homomorphism | Yes | Yes | Yes | Yes |
| Reversal | Yes | Yes | Yes | Yes |

Key consequences:

- CFLs are closed under intersection with a regular language: if $L$ is a CFL and $R$ is regular, then $L \cap R$ is a CFL. This is extremely useful for proofs.
- The class of deterministic CFLs (DCFLs) is closed under complement but not under union or intersection.

---

## 6. Decidability Properties

### The Problem

Determine which decision problems are decidable for each class of automata.

### Decidability Table

| Problem | DFA/NFA | DPDA | PDA/CFG | LBA | TM |
|---------|---------|------|---------|-----|-----|
| Membership ($w \in L$?) | $O(n)$ | $O(n)$ | $O(n^3)$ CYK | Decidable | **Undecidable** |
| Emptiness ($L = \emptyset$?) | Decidable | Decidable | Decidable | Decidable | **Undecidable** |
| Finiteness ($|L| < \infty$?) | Decidable | Decidable | Decidable | Decidable | **Undecidable** |
| Equivalence ($L_1 = L_2$?) | Decidable | Decidable | **Undecidable** | **Undecidable** | **Undecidable** |
| Inclusion ($L_1 \subseteq L_2$?) | Decidable | Decidable | **Undecidable** | **Undecidable** | **Undecidable** |
| Universality ($L = \Sigma^*$?) | Decidable | Decidable | **Undecidable** | **Undecidable** | **Undecidable** |
| Regularity (Is $L$ regular?) | Trivial | Decidable | **Undecidable** | -- | **Undecidable** |

---

## 7. Pumping Lemma Proofs

### The Problem

Use the pumping lemmas to formally prove that specific languages fall outside a given language class.

### Proof: $L = \{0^n 1^n \mid n \ge 0\}$ Is Not Regular

**Claim:** $L$ is not regular.

**Proof.** Suppose for contradiction that $L$ is regular. Let $p$ be the pumping length from the pumping lemma.

Choose $w = 0^p 1^p \in L$. Then $|w| = 2p \ge p$.

By the pumping lemma, $w = xyz$ where $|y| > 0$, $|xy| \le p$, and $xy^i z \in L$ for all $i \ge 0$.

Since $|xy| \le p$, both $x$ and $y$ consist entirely of $0$s. Write $y = 0^k$ for some $k \ge 1$.

Pump with $i = 0$: $xy^0 z = 0^{p-k} 1^p$. Since $k \ge 1$, we have $p - k < p$, so the number of $0$s differs from the number of $1$s. Thus $xy^0 z \notin L$. Contradiction. $\blacksquare$

### Proof: $L = \{a^n b^n c^n \mid n \ge 0\}$ Is Not Context-Free

**Claim:** $L$ is not context-free.

**Proof.** Suppose for contradiction that $L$ is context-free. Let $p$ be the pumping length.

Choose $w = a^p b^p c^p \in L$. Then $|w| = 3p \ge p$.

By the CFL pumping lemma, $w = uvxyz$ where $|vy| > 0$, $|vxy| \le p$, and $uv^i xy^i z \in L$ for all $i \ge 0$.

Since $|vxy| \le p$, the substring $vxy$ cannot span all three symbols $a$, $b$, $c$ simultaneously. Therefore $v$ and $y$ together touch at most two of the three symbols.

Pump with $i = 2$: $uv^2 xy^2 z$ increases the count of at most two of the three symbols while leaving the third unchanged. The three counts are no longer equal, so $uv^2 xy^2 z \notin L$. Contradiction. $\blacksquare$

---

## 8. The CYK Parsing Algorithm

### The Problem

Given a context-free grammar $G$ in Chomsky Normal Form and a string $w$, determine whether $w \in L(G)$ in $O(n^3 |R|)$ time.

### The Algorithm

The Cocke-Younger-Kasami algorithm uses dynamic programming. Let $w = a_1 a_2 \cdots a_n$.

Define $T[i, j]$ = set of nonterminals $A$ such that $A \Rightarrow^* a_i a_{i+1} \cdots a_j$.

**Base case** ($j = i$, substrings of length 1):

$$T[i, i] = \{ A \mid (A \to a_i) \in R \}$$

**Inductive case** ($j > i$, substrings of length $\ell = j - i + 1$):

$$T[i, j] = \{ A \mid \exists k, i \le k < j : (A \to BC) \in R, B \in T[i, k], C \in T[k+1, j] \}$$

**Accept** if $S \in T[1, n]$.

### Worked Example

Grammar in CNF:

```
S -> AB | BC
A -> BA | a
B -> CC | b
C -> AB | a
```

Parse $w = baaba$:

```
Table T[i,j]:          j=1    j=2    j=3    j=4    j=5
                       'b'    'a'    'a'    'b'    'a'
i=1 (b)              {B}    {S,A}  {B}    --     {S,A}
i=2 (a)                     {A,C}  {B}    {S,A}  --
i=3 (a)                            {A,C}  {S,C}  {B}
i=4 (b)                                   {B}    {S,A}
i=5 (a)                                          {A,C}

Fill order: length 1, then 2, then 3, then 4, then 5.

Length 1: T[1,1]={B}, T[2,2]={A,C}, T[3,3]={A,C}, T[4,4]={B}, T[5,5]={A,C}

Length 2: T[1,2]: k=1: A->BA? B in T[1,1], A in T[2,2] -> yes, A.
                         S->AB? A in T[1,1]? no. S->BC? B in T[1,1], C in T[2,2] -> yes, S.
                  T[1,2] = {S, A}

         T[2,3]: k=2: S->AB? A in T[2,2], B in T[3,3]? no. ...
                       B->CC? C in T[2,2], C in T[3,3] -> yes, B.
                  T[2,3] = {B}

         T[3,4]: k=3: S->AB? A in T[3,3], B in T[4,4] -> yes, S.
                       C->AB? A in T[3,3], B in T[4,4] -> yes, C.
                  T[3,4] = {S, C}

         T[4,5]: k=4: S->AB? no (A not in T[4,4]).
                       S->BC? B in T[4,4], C in T[5,5] -> yes, S.
                       A->BA? B in T[4,4], A in T[5,5] -> yes, A.
                  T[4,5] = {S, A}

Length 3-5: (continue filling by considering all split points)

S in T[1,5]? Yes -> w is in L(G).
```

**Complexity:** $O(n^3 \cdot |R|)$ time, $O(n^2)$ space.

---

## 9. Kleene's Theorem and Regular Expression Equivalence

### The Problem

Prove that regular expressions, DFAs, and NFAs all describe exactly the same class of languages.

### The Three Directions

**RE $\to$ NFA (Thompson's Construction):**

For each regular expression operator, construct a small NFA fragment:

```
Base: single character 'a'
  -->(q0)--a-->(q1)

Union: R1 | R2
  Create new start, epsilon to both sub-NFA starts.
  Both sub-NFA accepts epsilon to new single accept.

Concatenation: R1 R2
  Accept of NFA1 epsilon-transitions to start of NFA2.

Kleene star: R*
  New start (also accept) epsilon to sub-NFA start.
  Sub-NFA accept epsilon back to new start.
```

**NFA $\to$ DFA:** Subset construction (Section 2).

**DFA $\to$ RE (State Elimination):**

Iteratively remove states from the DFA, replacing transition labels with regular expressions that account for the removed paths. When only start and accept remain, the label on the remaining edge is the regular expression.

---

## 10. Turing Machines and Undecidability

### The Problem

Define Turing machines formally and establish the fundamental undecidability results.

### The Formula

A **Turing Machine** is a 7-tuple $M = (Q, \Sigma, \Gamma, \delta, q_0, q_{\text{accept}}, q_{\text{reject}})$ where:

$$\delta : Q \times \Gamma \to Q \times \Gamma \times \{L, R\}$$

**Church-Turing Thesis:** Every effectively computable function is Turing-computable. This is not a theorem but a widely accepted hypothesis.

### The Halting Problem

**Theorem.** $\text{HALT}_{\text{TM}} = \{ \langle M, w \rangle \mid M \text{ is a TM that halts on input } w \}$ is undecidable.

**Proof (diagonalization).** Assume $H$ decides $\text{HALT}_{\text{TM}}$. Construct $D$: on input $\langle M \rangle$, run $H(\langle M, \langle M \rangle \rangle)$. If $H$ accepts (M halts on its own encoding), loop forever. If $H$ rejects, accept.

Now run $D(\langle D \rangle)$:
- If $D$ halts on $\langle D \rangle$, then $H$ accepts, so $D$ loops. Contradiction.
- If $D$ loops on $\langle D \rangle$, then $H$ rejects, so $D$ accepts (halts). Contradiction.

Therefore $H$ cannot exist. $\blacksquare$

### Rice's Theorem

**Theorem.** Every non-trivial semantic property of Turing machines is undecidable.

Formally: let $P$ be a property of recognizable languages (not of machines). If $P$ is non-trivial (some TMs have it, some do not), then $\{ \langle M \rangle \mid L(M) \text{ has property } P \}$ is undecidable.

This immediately implies undecidability of: "Is $L(M)$ empty?", "Is $L(M)$ regular?", "Is $L(M) = \Sigma^*$?", etc.

---

## Tips

- The subset construction always works but can produce exponentially many states. In practice, use lazy construction (only build reachable states).
- CYK is the standard $O(n^3)$ algorithm for CFL membership, but Earley's algorithm handles arbitrary CFGs (not just CNF) and runs in $O(n^3)$ worst case, $O(n^2)$ for unambiguous grammars, $O(n)$ for most LR grammars.
- The pumping lemma is a necessary but not sufficient condition for regularity. Myhill-Nerode is both necessary and sufficient.
- When proving non-regularity or non-context-freeness, choose your "adversarial" string carefully. The string must be in the language and long enough to pump.
- Deterministic PDAs (DPDAs) are strictly weaker than nondeterministic PDAs. The language $\{ww^R \mid w \in \{0,1\}^*\}$ is context-free but not deterministic context-free.

## See Also

- Computational Complexity
- Decidability and Computability
- Compiler Design (lexing and parsing phases)
- Formal Verification

## References

- Sipser, M. *Introduction to the Theory of Computation*, 3rd ed., Cengage (2012), Chapters 1-5
- Hopcroft, J., Motwani, R., Ullman, J. *Introduction to Automata Theory, Languages, and Computation*, 3rd ed., Addison-Wesley (2006)
- Chomsky, N. "Three models for the description of language." IRE Transactions on Information Theory, 2(3):113-124 (1956)
- Rabin, M. & Scott, D. "Finite automata and their decision problems." IBM Journal of Research and Development, 3(2):114-125 (1959)
- Kleene, S. "Representation of events in nerve nets and finite automata." Automata Studies, Princeton (1956)
- Cocke, J. & Schwartz, J. *Programming Languages and Their Compilers*, Courant Institute (1970)
- Myhill, J. "Finite automata and the representation of events." WADD TR 57-624 (1957)
- Nerode, A. "Linear automaton transformations." Proceedings of the AMS, 9(4):541-544 (1958)
