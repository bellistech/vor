# The Theory of Lambda Calculus -- Reduction, Encoding, and the Foundations of Computation

> *The lambda calculus, a formal system of function abstraction and application, is computationally equivalent to Turing machines and provides the mathematical foundation for functional programming, type theory, and the Curry-Howard correspondence between proofs and programs.*

---

## 1. Formal Syntax (BNF and Conventions)

### The Problem

Define the lambda calculus rigorously as a formal language, establishing the grammar, binding structure, and notational conventions that underpin all subsequent theory.

### The Formula

The set $\Lambda$ of lambda terms over a countably infinite set of variables $V = \{x, y, z, x_1, x_2, \ldots\}$ is defined by the following BNF grammar:

$$M, N ::= x \mid (\lambda x.\, M) \mid (M\; N)$$

where $x \in V$. The three forms are called **variable**, **abstraction**, and **application** respectively.

**Free variables** are defined inductively:

$$\text{FV}(x) = \{x\}$$
$$\text{FV}(\lambda x.\, M) = \text{FV}(M) \setminus \{x\}$$
$$\text{FV}(M\; N) = \text{FV}(M) \cup \text{FV}(N)$$

A term $M$ is **closed** (a combinator) if $\text{FV}(M) = \emptyset$.

**Substitution** $M[x := N]$ is defined inductively:

$$x[x := N] = N$$
$$y[x := N] = y \quad \text{if } y \neq x$$
$$(\lambda x.\, M)[x := N] = \lambda x.\, M$$
$$(\lambda y.\, M)[x := N] = \lambda y.\, (M[x := N]) \quad \text{if } y \neq x \text{ and } y \notin \text{FV}(N)$$
$$(\lambda y.\, M)[x := N] = \lambda z.\, (M[y := z][x := N]) \quad \text{if } y \neq x \text{ and } y \in \text{FV}(N), z \text{ fresh}$$
$$(M_1\; M_2)[x := N] = (M_1[x := N])\; (M_2[x := N])$$

### Worked Examples

Substitution with variable capture avoidance:

$(\lambda y.\, x\; y)[x := y]$

Since $y \in \text{FV}(y)$ and the bound variable is $y$, we must rename:

$$= \lambda z.\, (x\; z)[x := y] = \lambda z.\, y\; z$$

Without renaming, we would get $\lambda y.\, y\; y$, which incorrectly captures the free $y$.

---

## 2. Reduction Rules and Confluence

### The Problem

Formalize the computation rules of lambda calculus and prove that the order of reduction does not affect the final result (when it exists).

### The Formula

The three reduction relations on $\Lambda$:

**Alpha reduction** ($\to_\alpha$): $\lambda x.\, M \to_\alpha \lambda y.\, M[x := y]$ provided $y \notin \text{FV}(M)$.

**Beta reduction** ($\to_\beta$): $(\lambda x.\, M)\; N \to_\beta M[x := N]$.

The subterm $(\lambda x.\, M)\; N$ is called a **beta redex** (reducible expression), and $M[x := N]$ is its **contractum**.

**Eta reduction** ($\to_\eta$): $\lambda x.\, M\; x \to_\eta M$ provided $x \notin \text{FV}(M)$.

We write $\twoheadrightarrow_\beta$ for the reflexive transitive closure of $\to_\beta$, and $=_\beta$ for the equivalence closure (beta convertibility).

### Step-by-Step Reduction Examples

**Example 1:** Compute $(\lambda x.\, \lambda y.\, x)\; a\; b$

$$(\lambda x.\, \lambda y.\, x)\; a\; b$$
$$\to_\beta (\lambda y.\, a)\; b \quad \text{[substitute } x := a \text{]}$$
$$\to_\beta a \quad \text{[substitute } y := b \text{, but } y \notin \text{FV}(a) \text{]}$$

**Example 2:** Self-application: $(\lambda x.\, x\; x)(\lambda y.\, y)$

$$(\lambda x.\, x\; x)(\lambda y.\, y)$$
$$\to_\beta (\lambda y.\, y)(\lambda y.\, y) \quad \text{[substitute } x := \lambda y.\, y \text{]}$$
$$\to_\beta \lambda y.\, y \quad \text{[substitute } y := \lambda y.\, y \text{]}$$

**Example 3:** Divergence: $\Omega = (\lambda x.\, x\; x)(\lambda x.\, x\; x)$

$$\Omega \to_\beta (\lambda x.\, x\; x)(\lambda x.\, x\; x) = \Omega$$

This term has no normal form -- it reduces to itself forever.

---

## 3. Church-Rosser Theorem (Confluence)

### The Problem

Prove that beta reduction is confluent: if a term can reduce in two different ways, both reduction paths can be extended to reach a common term.

### The Formula

**Church-Rosser Theorem.** If $M \twoheadrightarrow_\beta N_1$ and $M \twoheadrightarrow_\beta N_2$, then there exists a term $P$ such that $N_1 \twoheadrightarrow_\beta P$ and $N_2 \twoheadrightarrow_\beta P$.

```
           M
          / \
         /   \
        v     v
       N1     N2
        \     /
         \   /
          v v
           P
```

**Corollary 1 (Uniqueness of Normal Forms).** If $M$ has a normal form, it is unique up to alpha equivalence.

*Proof sketch.* Suppose $M \twoheadrightarrow_\beta N_1$ and $M \twoheadrightarrow_\beta N_2$ where both $N_1, N_2$ are in normal form. By Church-Rosser, there exists $P$ with $N_1 \twoheadrightarrow_\beta P$ and $N_2 \twoheadrightarrow_\beta P$. But since $N_1$ and $N_2$ are normal forms, no further reduction is possible, so $N_1 = P = N_2$ (up to $\alpha$). $\square$

**Corollary 2 (Consistency).** The lambda calculus is consistent: not all terms are beta-equivalent. In particular, $\lambda x.\, \lambda y.\, x \neq_\beta \lambda x.\, \lambda y.\, y$ (i.e., TRUE $\neq$ FALSE).

**Proof method.** The standard proof uses the Tait-Martin-Lof method of **parallel reduction** $\Rightarrow$, showing:
1. $\to_\beta \subseteq \Rightarrow \subseteq \twoheadrightarrow_\beta$
2. $\Rightarrow$ satisfies the diamond property: if $M \Rightarrow N_1$ and $M \Rightarrow N_2$ then there exists $P$ with $N_1 \Rightarrow P$ and $N_2 \Rightarrow P$

The diamond property for $\Rightarrow$ lifts to confluence for $\twoheadrightarrow_\beta$ by the Strip Lemma.

---

## 4. Encoding Arithmetic

### The Problem

Demonstrate that lambda calculus can represent natural number arithmetic, proving computational universality for this fragment.

### The Formula

**Church numerals.** The numeral $\overline{n}$ is defined as:

$$\overline{n} = \lambda f.\, \lambda x.\, f^n(x)$$

where $f^0(x) = x$ and $f^{n+1}(x) = f(f^n(x))$.

**Successor:**

$$\text{SUCC} = \lambda n.\, \lambda f.\, \lambda x.\, f\; (n\; f\; x)$$

**Addition:**

$$\text{ADD} = \lambda m.\, \lambda n.\, \lambda f.\, \lambda x.\, m\; f\; (n\; f\; x)$$

**Multiplication:**

$$\text{MUL} = \lambda m.\, \lambda n.\, \lambda f.\, m\; (n\; f)$$

**Predecessor** (the hardest basic operation -- requires pairs):

$$\text{PRED} = \lambda n.\, \lambda f.\, \lambda x.\, n\; (\lambda g.\, \lambda h.\, h\; (g\; f))\; (\lambda u.\, x)\; (\lambda u.\, u)$$

**Zero test:**

$$\text{ISZERO} = \lambda n.\, n\; (\lambda x.\, \text{FALSE})\; \text{TRUE}$$

### Worked Examples

**Verify SUCC $\overline{2}$ = $\overline{3}$:**

$$\text{SUCC}\; \overline{2}$$
$$= (\lambda n.\, \lambda f.\, \lambda x.\, f\; (n\; f\; x))\; (\lambda f.\, \lambda x.\, f\; (f\; x))$$
$$\to_\beta \lambda f.\, \lambda x.\, f\; ((\lambda f.\, \lambda x.\, f\; (f\; x))\; f\; x)$$
$$\to_\beta \lambda f.\, \lambda x.\, f\; ((\lambda x.\, f\; (f\; x))\; x)$$
$$\to_\beta \lambda f.\, \lambda x.\, f\; (f\; (f\; x))$$
$$= \overline{3}$$

**Verify ADD $\overline{2}$ $\overline{1}$ = $\overline{3}$:**

$$\text{ADD}\; \overline{2}\; \overline{1}$$
$$= (\lambda m.\, \lambda n.\, \lambda f.\, \lambda x.\, m\; f\; (n\; f\; x))\; \overline{2}\; \overline{1}$$
$$\to_\beta \lambda f.\, \lambda x.\, \overline{2}\; f\; (\overline{1}\; f\; x)$$
$$\to_\beta \lambda f.\, \lambda x.\, \overline{2}\; f\; (f\; x)$$
$$\to_\beta \lambda f.\, \lambda x.\, f\; (f\; (f\; x))$$
$$= \overline{3}$$

**Verify MUL $\overline{2}$ $\overline{3}$ = $\overline{6}$:**

$$\text{MUL}\; \overline{2}\; \overline{3} = \lambda f.\, \overline{2}\; (\overline{3}\; f)$$
$$= \lambda f.\, \overline{2}\; (\lambda x.\, f\; (f\; (f\; x)))$$
$$= \lambda f.\, \lambda x.\, (\lambda x.\, f\; (f\; (f\; x)))((\lambda x.\, f\; (f\; (f\; x)))\; x)$$
$$\to_\beta \lambda f.\, \lambda x.\, f\; (f\; (f\; (f\; (f\; (f\; x)))))$$
$$= \overline{6}$$

---

## 5. Fixed-Point Combinators and the Y Combinator

### The Problem

Show that every lambda term has a fixed point, enabling recursion in a language without built-in self-reference.

### The Formula

**Fixed-Point Theorem.** For every lambda term $F$, there exists a term $X$ such that $F\; X =_\beta X$.

**Proof.** Let $W = \lambda x.\, F\; (x\; x)$ and $X = W\; W$. Then:

$$X = W\; W = (\lambda x.\, F\; (x\; x))\; W \to_\beta F\; (W\; W) = F\; X$$

**Curry's Y combinator:**

$$Y = \lambda f.\, (\lambda x.\, f\; (x\; x))\; (\lambda x.\, f\; (x\; x))$$

satisfies $Y\; F \twoheadrightarrow_\beta F\; (Y\; F)$ for all $F$.

**Turing's fixed-point combinator** $\Theta$:

$$A = \lambda x.\, \lambda y.\, y\; (x\; x\; y)$$
$$\Theta = A\; A$$

satisfies $\Theta\; F \twoheadrightarrow_\beta F\; (\Theta\; F)$ and has the advantage that $\Theta\; F \to_\beta F\; (\Theta\; F)$ in one step (not just multi-step).

**Call-by-value variant (Z combinator):**

$$Z = \lambda f.\, (\lambda x.\, f\; (\lambda v.\, x\; x\; v))\; (\lambda x.\, f\; (\lambda v.\, x\; x\; v))$$

Required in strict languages because $Y\; F$ diverges under call-by-value evaluation.

---

## 6. De Bruijn Indices

### The Problem

Eliminate the need for alpha conversion by representing variable binding as a nameless notation where each variable reference is a natural number indicating how many binders to skip.

### The Formula

A de Bruijn index $n$ refers to the variable bound by the $n$-th enclosing $\lambda$ (counting from 0).

**Translation rules:**

| Named notation | De Bruijn notation |
|---|---|
| $\lambda x.\, x$ | $\lambda.\, 0$ |
| $\lambda x.\, \lambda y.\, x$ | $\lambda.\, \lambda.\, 1$ |
| $\lambda x.\, \lambda y.\, y$ | $\lambda.\, \lambda.\, 0$ |
| $\lambda x.\, \lambda y.\, x\; y$ | $\lambda.\, \lambda.\, 1\; 0$ |
| $\lambda f.\, \lambda x.\, f\; (f\; x)$ | $\lambda.\, \lambda.\, 1\; (1\; 0)$ |

**Beta reduction with de Bruijn indices:**

$$(\lambda.\, M)\; N \to_\beta \, \downarrow^0 (M[0 := \, \uparrow^0 N])$$

where $\uparrow^k$ (shift) increments free indices $\ge k$ by 1, and $\downarrow^k$ (unshift) decrements free indices $\ge k$ by 1.

### Worked Examples

Reducing $(\lambda.\, \lambda.\, 1\; 0)\; (\lambda.\, 0)$ (i.e., $(\lambda x.\, \lambda y.\, x\; y)\; (\lambda z.\, z)$):

1. Substitute index 0 in body $\lambda.\, 1\; 0$:
   - The inner $\lambda$ shifts the target index, so we substitute for index 1 in $1\; 0$
   - $1[1 := \uparrow^1(\lambda.\, 0)] = 1[1 := \lambda.\, 0] = \lambda.\, 0$
   - $0$ remains $0$ (bound by the inner $\lambda$)
2. Result: $\lambda.\, (\lambda.\, 0)\; 0$
3. This is $\lambda y.\, (\lambda z.\, z)\; y \to_\beta \lambda y.\, y$, confirming correctness.

---

## 7. Equivalence with Turing Machines

### The Problem

Establish that lambda calculus and Turing machines define the same class of computable functions, providing evidence for the Church-Turing thesis.

### The Formula

**Church-Turing Thesis** (informal). Every effectively computable function is lambda-definable (equivalently, Turing-computable).

**Theorem (Church 1936, Turing 1937).** A function $f : \mathbb{N} \to \mathbb{N}$ is Turing-computable if and only if it is lambda-definable.

A function $f$ is **lambda-definable** if there exists a closed lambda term $F$ such that for all $n \in \mathbb{N}$:

$$F\; \overline{n} =_\beta \overline{f(n)}$$

**Proof sketch (Turing-computable $\Rightarrow$ lambda-definable):** Encode a Turing machine configuration $(q, \text{tape}, \text{head})$ as a lambda term. The transition function $\delta$ is representable since it is finite. The step function STEP applies $\delta$ to the current configuration. The computation is simulated by iterating STEP using the Y combinator until a halting state is reached.

**Proof sketch (lambda-definable $\Rightarrow$ Turing-computable):** A Turing machine can simulate beta reduction by maintaining a representation of lambda terms on its tape, implementing substitution as a string operation, and applying the leftmost-outermost reduction strategy (which is normalizing by the Standardization Theorem).

**Consequence:** The halting problem for lambda calculus is undecidable. There is no lambda term $H$ such that:

$$H\; \ulcorner M \urcorner = \begin{cases} \text{TRUE} & \text{if } M \text{ has a normal form} \\ \text{FALSE} & \text{otherwise} \end{cases}$$

---

## 8. Scott-Curry Theorem

### The Problem

Characterize which properties of lambda terms can be decided within the calculus itself, revealing fundamental limitations analogous to Rice's theorem for Turing machines.

### The Formula

**Scott-Curry Theorem.** Let $A$ and $B$ be disjoint, non-empty sets of lambda terms that are closed under beta equivalence (i.e., if $M \in A$ and $M =_\beta N$ then $N \in A$, and likewise for $B$). Then $A$ and $B$ are **recursively inseparable**: there is no lambda term $F$ such that:

$$F\; M = \begin{cases} \text{TRUE} & \text{if } M \in A \\ \text{FALSE} & \text{if } M \in B \end{cases}$$

**Consequences:**

- No lambda term can decide whether an arbitrary term has a normal form
- No lambda term can decide whether two terms are beta-equivalent
- The set of normalizing terms and the set of non-normalizing terms are recursively inseparable

This is the lambda calculus analogue of Rice's theorem: no nontrivial extensional property of lambda terms is decidable within the calculus.

---

## 9. Curry-Howard Correspondence (Preview)

### The Problem

Reveal the deep structural isomorphism between typed lambda calculus and propositional logic, where types correspond to propositions and programs correspond to proofs.

### The Formula

| Simply Typed Lambda Calculus | Intuitionistic Propositional Logic |
|---|---|
| Type $\sigma \to \tau$ | Implication $\sigma \Rightarrow \tau$ |
| Type $\sigma \times \tau$ | Conjunction $\sigma \wedge \tau$ |
| Type $\sigma + \tau$ | Disjunction $\sigma \vee \tau$ |
| Type $\bot$ (empty) | Falsehood $\perp$ |
| Term $M : \tau$ | Proof of proposition $\tau$ |
| Function $\lambda x.\, M : \sigma \to \tau$ | Implication introduction |
| Application $M\; N$ | Modus ponens (implication elimination) |
| Beta reduction | Proof normalization (cut elimination) |
| Normal form | Cut-free proof |
| Inhabited type | Provable proposition |

**Example:** The identity function $\lambda x.\, x : \alpha \to \alpha$ corresponds to the trivial proof that $\alpha$ implies $\alpha$.

**Example:** The K combinator $\lambda x.\, \lambda y.\, x : \alpha \to \beta \to \alpha$ corresponds to the proof of the tautology $\alpha \Rightarrow (\beta \Rightarrow \alpha)$: "if $\alpha$ holds, then regardless of $\beta$, $\alpha$ still holds."

**The correspondence extends:**

| System | Logic |
|---|---|
| Simply typed $\lambda$ | Propositional logic |
| System F (polymorphic $\lambda$) | Second-order logic |
| Dependent types | Predicate logic |
| Linear types | Linear logic |

This is not merely an analogy -- it is a precise mathematical isomorphism. Every well-typed program IS a proof, and every proof IS a program.

---

## Prerequisites

- Formal language theory (BNF grammars, induction on syntax)
- Basic set theory (union, intersection, set difference)
- Mathematical induction
- Computability theory (Turing machines, halting problem, Rice's theorem)
- Propositional logic (for Curry-Howard section)

## Complexity

| Level | Description |
|-------|-------------|
| **Beginner** | Write lambda terms in the three-construct syntax. Perform single-step beta reductions. Evaluate Church numerals by hand. Identify free and bound variables. |
| **Intermediate** | Prove Church-Rosser by parallel reduction. Implement an evaluator for both call-by-name and call-by-value. Derive predecessor on Church numerals. Translate between named and de Bruijn representations. Construct recursive functions via Y combinator. |
| **Advanced** | Study Scott domains and denotational semantics. Prove the Standardization Theorem. Explore System F and polymorphic lambda calculus. Formalize the Curry-Howard correspondence for dependent types. Investigate Bohm trees and the lambda theory $\mathcal{H}^*$. |
