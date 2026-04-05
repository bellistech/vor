# Type Theory -- From Simple Types to Dependent Types, Linear Logic, and the Curry-Howard Correspondence

> *Type theory is the mathematical study of type systems: formal calculi that classify terms according to their computational behavior. Originating in Russell's resolution of set-theoretic paradoxes and Church's simply typed lambda calculus, type theory now underpins programming language design, automated theorem proving, and the profound correspondence between proofs and programs discovered by Curry and Howard.*

---

## 1. Simply Typed Lambda Calculus (STLC)

### The Problem

The untyped lambda calculus admits non-terminating terms such as $\Omega = (\lambda x.\, x\, x)(\lambda x.\, x\, x)$. We need a discipline that rules out pathological terms while retaining enough expressive power for practical computation.

### The Formula

**Types.** Given a set of base types $\mathcal{B} = \{\text{Bool}, \text{Nat}, \ldots\}$, the set of simple types is:

$$\tau ::= \alpha \mid \tau_1 \to \tau_2$$

where $\alpha \in \mathcal{B}$ and $\to$ is right-associative: $A \to B \to C = A \to (B \to C)$.

**Typing context.** A typing context $\Gamma$ is a finite mapping from variables to types:

$$\Gamma ::= \emptyset \mid \Gamma, x : \tau$$

**Typing rules.** The typing judgment $\Gamma \vdash M : \tau$ is defined by three rules:

$$\frac{x : \tau \in \Gamma}{\Gamma \vdash x : \tau} \quad \text{(Var)}$$

$$\frac{\Gamma, x : \sigma \vdash M : \tau}{\Gamma \vdash (\lambda x.\, M) : \sigma \to \tau} \quad \text{(Abs)}$$

$$\frac{\Gamma \vdash M : \sigma \to \tau \qquad \Gamma \vdash N : \sigma}{\Gamma \vdash M\, N : \tau} \quad \text{(App)}$$

**Key properties of STLC:**

1. **Strong normalization.** Every well-typed term has a normal form, and every reduction sequence terminates. Proof: by Tait's method of logical relations (1967).

2. **Decidability.** Both type checking ("given $\Gamma$, $M$, $\tau$, does $\Gamma \vdash M : \tau$ hold?") and type inference ("given $M$, find $\Gamma$ and $\tau$") are decidable.

3. **Loss of Turing-completeness.** The Y combinator $Y = \lambda f.\, (\lambda x.\, f(x\, x))(\lambda x.\, f(x\, x))$ is not typeable in STLC. Strong normalization and Turing-completeness are mutually exclusive.

### Worked Examples

**Typing the identity function:**

We want to derive $\vdash \lambda x.\, x : A \to A$ for any type $A$.

$$\frac{x : A \in \{x : A\}}{\{x : A\} \vdash x : A} \text{(Var)}$$

$$\frac{\{x : A\} \vdash x : A}{\vdash \lambda x.\, x : A \to A} \text{(Abs)}$$

**Typing function composition:**

Let $f : B \to C$ and $g : A \to B$. We derive the type of $\lambda f.\, \lambda g.\, \lambda x.\, f\, (g\, x)$:

$$\frac{\frac{\Gamma \vdash g : A \to B \qquad \Gamma \vdash x : A}{\Gamma \vdash g\, x : B} \text{(App)} \qquad \Gamma \vdash f : B \to C}{\Gamma \vdash f\, (g\, x) : C} \text{(App)}$$

where $\Gamma = \{f : B \to C,\; g : A \to B,\; x : A\}$.

Abstracting out: $\vdash \lambda f.\, \lambda g.\, \lambda x.\, f\, (g\, x) : (B \to C) \to (A \to B) \to A \to C$.

---

## 2. Hindley-Milner Type System and Algorithm W

### The Problem

STLC requires a separate definition for every type instantiation of a polymorphic function. We want the identity function to work for all types simultaneously, with full type inference and no annotations.

### The Formula

**Types and type schemes.** Hindley-Milner distinguishes monotypes from polytypes (type schemes):

$$\tau ::= \alpha \mid \tau_1 \to \tau_2 \mid C\, \tau_1 \ldots \tau_n \qquad \text{(monotypes)}$$

$$\sigma ::= \tau \mid \forall \alpha.\, \sigma \qquad \text{(type schemes)}$$

Quantifiers $\forall$ appear only at the outermost level (prenex form). This is **predicative** polymorphism: quantified variables range only over monotypes.

**Additional typing rules** (extending STLC):

$$\frac{\Gamma \vdash M : \sigma \qquad \alpha \notin \text{FTV}(\Gamma)}{\Gamma \vdash M : \forall \alpha.\, \sigma} \quad \text{(Gen)}$$

$$\frac{\Gamma \vdash M : \forall \alpha.\, \sigma}{\Gamma \vdash M : \sigma[\alpha := \tau]} \quad \text{(Inst)}$$

$$\frac{\Gamma \vdash M : \sigma \qquad \Gamma, x : \sigma \vdash N : \tau}{\Gamma \vdash \textbf{let}\; x = M\; \textbf{in}\; N : \tau} \quad \text{(Let)}$$

The Let rule is crucial: it allows generalization of the type of $M$ before using it polymorphically in $N$.

**Algorithm W** (Damas and Milner, 1982):

Given context $\Gamma$ and expression $e$, Algorithm W returns a substitution $S$ and a type $\tau$ such that $S(\Gamma) \vdash e : \tau$ is derivable.

```
W(Gamma, x):
    if x : sigma in Gamma:
        return (id, instantiate(sigma))    -- fresh type vars for each forall

W(Gamma, lambda x. e):
    let alpha = fresh type variable
    let (S, tau) = W(Gamma + {x : alpha}, e)
    return (S, S(alpha) -> tau)

W(Gamma, e1 e2):
    let (S1, tau1) = W(Gamma, e1)
    let (S2, tau2) = W(S1(Gamma), e2)
    let alpha = fresh type variable
    let S3 = unify(S2(tau1), tau2 -> alpha)
    return (S3 . S2 . S1, S3(alpha))

W(Gamma, let x = e1 in e2):
    let (S1, tau1) = W(Gamma, e1)
    let sigma = generalize(S1(Gamma), tau1)
    let (S2, tau2) = W(S1(Gamma) + {x : sigma}, e2)
    return (S2 . S1, tau2)
```

**Unification** (Robinson, 1965):

```
unify(alpha, tau):
    if alpha == tau: return id
    if alpha in FTV(tau): FAIL (occurs check)
    return [alpha := tau]

unify(tau1 -> tau2, tau3 -> tau4):
    let S1 = unify(tau1, tau3)
    let S2 = unify(S1(tau2), S1(tau4))
    return S2 . S1

unify(C tau1..tn, C sigma1..sn):
    return unify(tau1,sigma1) . ... . unify(taun, sigman)

unify(_, _): FAIL
```

### Worked Examples

**Inferring the type of** $\lambda f.\, \lambda x.\, f\, (f\, x)$:

Step 1: Assign fresh variables. $f : \alpha_1$, $x : \alpha_2$.

Step 2: Inner application $f\, x$:
- $f : \alpha_1$, $x : \alpha_2$
- For $f\, x$ to be well-typed, unify $\alpha_1$ with $\alpha_2 \to \alpha_3$
- Substitution: $S_1 = [\alpha_1 := \alpha_2 \to \alpha_3]$
- Result type: $\alpha_3$

Step 3: Outer application $f\, (f\, x)$:
- $f : \alpha_2 \to \alpha_3$ (after $S_1$), argument: $\alpha_3$
- Unify $\alpha_2 \to \alpha_3$ with $\alpha_3 \to \alpha_4$
- $S_2 = [\alpha_2 := \alpha_3, \alpha_4 := \alpha_3]$
- Result type: $\alpha_3$

Step 4: Compose substitutions. $f : \alpha_3 \to \alpha_3$, $x : \alpha_3$.

$$\vdash \lambda f.\, \lambda x.\, f\, (f\, x) : (\alpha_3 \to \alpha_3) \to \alpha_3 \to \alpha_3$$

Generalize: $\forall a.\, (a \to a) \to a \to a$.

**Principal type theorem** (Hindley 1969, Damas-Milner 1982): If $e$ is typeable, Algorithm W finds its **principal** (most general) type. Every other valid typing is a substitution instance.

---

## 3. System F (Polymorphic Lambda Calculus)

### The Problem

Hindley-Milner restricts polymorphism to prenex position (let-bindings only). System F lifts this restriction, allowing polymorphic values to be passed as arguments and returned from functions.

### The Formula

**Syntax** (Girard 1972, Reynolds 1974):

$$\tau ::= \alpha \mid \tau_1 \to \tau_2 \mid \forall \alpha.\, \tau$$

$$M ::= x \mid \lambda x{:}\tau.\, M \mid M\, N \mid \Lambda \alpha.\, M \mid M\, [\tau]$$

Two new term forms:
- $\Lambda \alpha.\, M$ -- **type abstraction**: creates a polymorphic value
- $M\, [\tau]$ -- **type application**: instantiates a polymorphic value

**Typing rules** (extending STLC):

$$\frac{\Gamma \vdash M : \tau \qquad \alpha \notin \text{FTV}(\Gamma)}{\Gamma \vdash \Lambda \alpha.\, M : \forall \alpha.\, \tau} \quad \text{(TAbs)}$$

$$\frac{\Gamma \vdash M : \forall \alpha.\, \tau}{\Gamma \vdash M\, [\sigma] : \tau[\alpha := \sigma]} \quad \text{(TApp)}$$

**Reduction rule:**

$$(\Lambda \alpha.\, M)\, [\tau] \longrightarrow_\beta M[\alpha := \tau]$$

**Church encodings in System F** are typed:

$$\text{Bool} = \forall \alpha.\, \alpha \to \alpha \to \alpha$$

$$\text{true} = \Lambda \alpha.\, \lambda t{:}\alpha.\, \lambda f{:}\alpha.\, t$$

$$\text{Nat} = \forall \alpha.\, (\alpha \to \alpha) \to \alpha \to \alpha$$

$$\text{zero} = \Lambda \alpha.\, \lambda s{:}(\alpha \to \alpha).\, \lambda z{:}\alpha.\, z$$

$$\text{succ} = \lambda n{:}\text{Nat}.\, \Lambda \alpha.\, \lambda s{:}(\alpha \to \alpha).\, \lambda z{:}\alpha.\, s\, (n\, [\alpha]\, s\, z)$$

**Existential types** encode abstract data types:

$$\exists \alpha.\, \tau \;\equiv\; \forall \beta.\, (\forall \alpha.\, \tau \to \beta) \to \beta$$

A module with hidden representation type $\alpha$, interface $\tau$, can only be used through the interface.

**Key results:**
- System F is **strongly normalizing** (Girard's proof, 1972).
- Type inference for System F is **undecidable** (Wells, 1999).
- Type checking (given explicit annotations) is decidable.
- System F is **impredicative**: $\forall \alpha.\, \alpha$ can be instantiated with $\forall \alpha.\, \alpha$ itself.

---

## 4. Curry-Howard Correspondence

### The Problem

Logic and type theory developed independently for decades. The Curry-Howard correspondence reveals that they are the same mathematical structure viewed from two perspectives: propositions are types, and proofs are programs.

### The Formula

The correspondence, discovered progressively by Curry (1934), Howard (1969), and extended by many others, maps each concept in logic to a concept in type theory and category theory:

| **Logic** | **Type Theory** | **Category Theory** |
|---|---|---|
| Proposition $A$ | Type $A$ | Object $A$ |
| Proof of $A$ | Term $M : A$ | Morphism $1 \to A$ |
| $A \Rightarrow B$ (implication) | $A \to B$ (function type) | Exponential $B^A$ |
| $A \land B$ (conjunction) | $A \times B$ (product type) | Product $A \times B$ |
| $A \lor B$ (disjunction) | $A + B$ (sum type) | Coproduct $A + B$ |
| $\top$ (truth) | $\mathbf{1}$ (unit type) | Terminal object $1$ |
| $\bot$ (falsity) | $\mathbf{0}$ (empty type) | Initial object $0$ |
| $\forall x.\, P(x)$ | $\Pi (x : A).\, B(x)$ | Right adjoint to pullback |
| $\exists x.\, P(x)$ | $\Sigma (x : A).\, B(x)$ | Left adjoint to pullback |
| $\neg A$ | $A \to \mathbf{0}$ | $0^A$ |
| Modus ponens | Function application | Evaluation morphism |
| Hypothesis | Variable | Identity morphism |
| Cut elimination | $\beta$-reduction | Composition |
| $\Rightarrow$-introduction | $\lambda$-abstraction | Currying |
| $\land$-introduction | Pair construction | Product morphism |
| $\lor$-elimination | Case/match expression | Coproduct morphism |

**The correspondence in action:**

To prove $A \Rightarrow B \Rightarrow A$ (the K combinator):

$$\frac{\frac{a : A \in \{a : A, b : B\}}{\{a : A, b : B\} \vdash a : A}}{\frac{\{a : A\} \vdash \lambda b.\, a : B \to A}{\vdash \lambda a.\, \lambda b.\, a : A \to B \to A}}$$

The proof IS the program $\lambda a.\, \lambda b.\, a$.

**Classical vs. intuitionistic:**

The Curry-Howard correspondence holds for **intuitionistic** logic (no law of excluded middle). Adding the law of excluded middle $A \lor \neg A$ corresponds to adding **call/cc** (call with current continuation) -- Griffin (1990):

$$\text{call/cc} : \forall \alpha.\, ((\alpha \to \beta) \to \alpha) \to \alpha \quad \longleftrightarrow \quad \text{Peirce's law}$$

---

## 5. Dependent Types and Martin-Lof Type Theory

### The Problem

In simple type systems, types and terms inhabit separate universes. We cannot express properties like "a list of length $n$" or "a sorted array" in the type system. Dependent types break this barrier: types can depend on values.

### The Formula

**Pi types** (dependent function types):

$$\frac{\Gamma \vdash A : \mathcal{U} \qquad \Gamma, x : A \vdash B(x) : \mathcal{U}}{\Gamma \vdash \Pi (x : A).\, B(x) : \mathcal{U}} \quad \text{(Pi-Form)}$$

$$\frac{\Gamma, x : A \vdash M : B(x)}{\Gamma \vdash \lambda x.\, M : \Pi (x : A).\, B(x)} \quad \text{(Pi-Intro)}$$

$$\frac{\Gamma \vdash f : \Pi (x : A).\, B(x) \qquad \Gamma \vdash a : A}{\Gamma \vdash f\, a : B(a)} \quad \text{(Pi-Elim)}$$

When $B$ does not depend on $x$: $\Pi (x : A).\, B = A \to B$.

**Sigma types** (dependent pair types):

$$\frac{\Gamma \vdash A : \mathcal{U} \qquad \Gamma, x : A \vdash B(x) : \mathcal{U}}{\Gamma \vdash \Sigma (x : A).\, B(x) : \mathcal{U}} \quad \text{(Sigma-Form)}$$

$$\frac{\Gamma \vdash a : A \qquad \Gamma \vdash b : B(a)}{\Gamma \vdash (a, b) : \Sigma (x : A).\, B(x)} \quad \text{(Sigma-Intro)}$$

**Identity types** (propositional equality):

$$\frac{\Gamma \vdash a : A \qquad \Gamma \vdash b : A}{\Gamma \vdash \text{Id}_A(a, b) : \mathcal{U}} \quad \text{(Id-Form)}$$

$$\frac{\Gamma \vdash a : A}{\Gamma \vdash \text{refl}_a : \text{Id}_A(a, a)} \quad \text{(Id-Intro)}$$

**Universe hierarchy:**

$$\mathcal{U}_0 : \mathcal{U}_1 : \mathcal{U}_2 : \cdots$$

$\mathcal{U}_i$ is the type of "small" types at level $i$. This avoids Girard's paradox (the type-theoretic analogue of Russell's paradox).

### Worked Examples

**Length-indexed vectors** (the "hello world" of dependent types):

Define $\text{Vec} : \mathcal{U} \to \mathbb{N} \to \mathcal{U}$:

$$\text{nil} : \text{Vec}\, A\, 0$$

$$\text{cons} : A \to \text{Vec}\, A\, n \to \text{Vec}\, A\, (n + 1)$$

Safe head function -- cannot be called on empty vectors:

$$\text{head} : \Pi (A : \mathcal{U}).\, \Pi (n : \mathbb{N}).\, \text{Vec}\, A\, (n + 1) \to A$$

$$\text{head}\, A\, n\, (\text{cons}\, a\, \_) = a$$

Vector append with length addition in the type:

$$\text{append} : \Pi (A : \mathcal{U}).\, \Pi (m\, n : \mathbb{N}).\, \text{Vec}\, A\, m \to \text{Vec}\, A\, n \to \text{Vec}\, A\, (m + n)$$

$$\text{append}\, A\, 0\, n\, \text{nil}\, ys = ys$$

$$\text{append}\, A\, (m+1)\, n\, (\text{cons}\, x\, xs)\, ys = \text{cons}\, x\, (\text{append}\, A\, m\, n\, xs\, ys)$$

**Proof that** $1 + 1 = 2$ as a type:

$$\text{refl} : \text{Id}_{\mathbb{N}}(1 + 1,\, 2)$$

This type-checks because $1 + 1$ and $2$ are definitionally equal. The proof is trivially $\text{refl}$.

**Martin-Lof type theory (MLTT)** consists of:

1. Dependent function types ($\Pi$)
2. Dependent pair types ($\Sigma$)
3. Identity types ($\text{Id}$)
4. Finite types ($\mathbf{0}$, $\mathbf{1}$, $\mathbf{2}$, $\ldots$)
5. Natural numbers ($\mathbb{N}$) with induction
6. A universe hierarchy ($\mathcal{U}_0, \mathcal{U}_1, \ldots$)

MLTT is the foundation of Coq, Agda, Lean, and Idris, and is the basis of the Univalent Foundations program (Voevodsky) and Homotopy Type Theory (HoTT).

---

## 6. Linear Logic and Linear Types

### The Problem

In classical and intuitionistic logic, hypotheses can be used any number of times (weakening) or discarded (contraction). This is wasteful for reasoning about resources: a file handle, a network socket, or a quantum bit should be used exactly once.

### The Formula

**Girard's linear logic** (1987) removes weakening and contraction, splitting each connective into multiplicative and additive versions:

| **Linear Logic** | **Notation** | **Meaning** |
|---|---|---|
| Multiplicative conjunction | $A \otimes B$ | "I have both $A$ and $B$" |
| Multiplicative disjunction | $A \mathrel{\invamp} B$ | "Producing $A$ also produces $B$" |
| Linear implication | $A \multimap B$ | "Consuming $A$, produce $B$" |
| Additive conjunction | $A \mathbin{\&} B$ | "I can choose $A$ or $B$ (but not both)" |
| Additive disjunction | $A \oplus B$ | "One of $A$ or $B$ (I choose which)" |
| Exponential (of course) | ${!}A$ | "Unlimited supply of $A$" |
| Exponential (why not) | ${?}A$ | "Demand for $A$" |

The exponentials ${!}$ and ${?}$ reintroduce weakening and contraction in a controlled way:

$$\frac{\Gamma, A, A \vdash B}{\Gamma, {!}A \vdash B} \quad \text{(Contraction)} \qquad \frac{\Gamma \vdash B}{\Gamma, {!}A \vdash B} \quad \text{(Weakening)}$$

**Linear type system** rules:

$$\frac{}{\{x : A\} \vdash x : A} \quad \text{(Var -- uses the resource)}$$

$$\frac{\Gamma, x : A \vdash M : B}{\Gamma \vdash \lambda x.\, M : A \multimap B} \quad \text{(Abs -- $x$ used exactly once in $M$)}$$

$$\frac{\Gamma_1 \vdash M : A \multimap B \qquad \Gamma_2 \vdash N : A}{\Gamma_1, \Gamma_2 \vdash M\, N : B} \quad \text{(App -- contexts split, no sharing)}$$

Note: the context splits between premises. Each variable is used exactly once across the entire derivation.

**Usage modalities** and their structural rules:

| **Modality** | **Weakening** | **Contraction** | **Example** |
|---|---|---|---|
| Linear | No | No | Must use exactly once |
| Affine | Yes | No | Use at most once (Rust) |
| Relevant | No | Yes | Use at least once |
| Unrestricted | Yes | Yes | Standard (classical) |

### Worked Examples

**Rust ownership as affine types:**

```rust
fn consume(s: String) {
    println!("{}", s);
    // s is dropped here -- used exactly once
}

fn main() {
    let s = String::from("hello");  // s : String (affine)
    consume(s);                     // s is moved (consumed)
    // println!("{}", s);           // ERROR: s already moved
}
```

The Rust borrow checker enforces an affine type discipline:
- $\text{move}$: linear consumption (ownership transfer)
- $\&T$: shared reference (${!}$-like, unlimited reads)
- $\&\text{mut}\, T$: unique mutable reference (linear write access)
- $\text{Drop}$: weakening (allowed to discard -- making it affine, not strictly linear)

**Session types** as linear protocols:

$$S = {!}\text{Int}.\, {?}\text{Bool}.\, \text{end}$$

"Send an integer, receive a boolean, then close." The linearity ensures the protocol is followed exactly, with no skipped or duplicated steps.

---

## 7. Strong Normalization

### The Problem

Prove that every well-typed term in a given type system reaches a normal form (a term with no further reductions possible), regardless of the reduction strategy chosen.

### The Formula

**Theorem (Strong Normalization for STLC).** If $\Gamma \vdash M : \tau$ in the simply typed lambda calculus, then every reduction sequence starting from $M$ is finite.

**Proof sketch** (Tait's method of reducibility candidates, 1967):

Define for each type $\tau$ a set $\text{RED}_\tau$ of **reducible** terms:

- $\text{RED}_\alpha = \text{SN}$ (the set of strongly normalizing terms of base type $\alpha$)
- $\text{RED}_{\sigma \to \tau} = \{M \mid \forall N \in \text{RED}_\sigma.\, M\, N \in \text{RED}_\tau\}$

**Key properties** (proved by induction on $\tau$):

1. (CR1) If $M \in \text{RED}_\tau$ then $M \in \text{SN}$.
2. (CR2) If $M \in \text{RED}_\tau$ and $M \to M'$ then $M' \in \text{RED}_\tau$.
3. (CR3) If $M$ is neutral and all one-step reducts of $M$ are in $\text{RED}_\tau$, then $M \in \text{RED}_\tau$.

**Main lemma.** If $\Gamma \vdash M : \tau$ and $\gamma$ maps each $x : \sigma$ in $\Gamma$ to a term in $\text{RED}_\sigma$, then $M[\gamma] \in \text{RED}_\tau$.

Proof: by induction on the derivation of $\Gamma \vdash M : \tau$. The key case is $\lambda$-abstraction, where we use CR3 and the substitution lemma.

**Corollary.** Taking $\gamma = \text{id}$ (variables are neutral, hence in $\text{RED}$ by CR3), we get $M \in \text{RED}_\tau \subseteq \text{SN}$.

**Extensions:**

- **System F**: Strong normalization proved by Girard (1972) using a more sophisticated version of reducibility candidates (parametric over type variable interpretations).
- **System F$_\omega$**: Also strongly normalizing.
- **Martin-Lof type theory**: Normalizing (with restrictions on universes).
- **Calculus of Constructions**: Strongly normalizing (Coquand and Huet, 1988).

**Non-normalizing systems:** Adding general recursion ($\text{fix} : (A \to A) \to A$) or unrestricted recursive types destroys strong normalization and recovers Turing-completeness.

---

## 8. Type Soundness (Progress and Preservation)

### The Problem

Establish rigorously that a type system achieves its fundamental purpose: well-typed programs do not exhibit undefined behavior at runtime.

### The Formula

**Type soundness** (Wright and Felleisen, 1994) is proved via two lemmas:

**Progress.** If $\vdash e : \tau$ (in the empty context), then either:
- $e$ is a value, or
- there exists $e'$ such that $e \longrightarrow e'$.

"Well-typed closed terms are never stuck."

**Preservation** (Subject Reduction). If $\vdash e : \tau$ and $e \longrightarrow e'$, then $\vdash e' : \tau$.

"Reduction preserves types."

**Soundness follows by induction:** if $e_0 \longrightarrow e_1 \longrightarrow \cdots \longrightarrow e_n$ and $\vdash e_0 : \tau$, then by repeated application of Preservation, $\vdash e_n : \tau$, and by Progress, $e_n$ is either a value or can step further.

**Proof of Preservation** (sketch for STLC):

By induction on the derivation of $\vdash e : \tau$. The key case is $\beta$-reduction:

$$(\lambda x.\, M)\, N \longrightarrow M[x := N]$$

We need the **Substitution Lemma**: if $\Gamma, x : \sigma \vdash M : \tau$ and $\Gamma \vdash N : \sigma$, then $\Gamma \vdash M[x := N] : \tau$.

Proof of the Substitution Lemma: by induction on the derivation of $\Gamma, x : \sigma \vdash M : \tau$.

**Proof of Progress** (sketch for STLC):

By induction on $\vdash e : \tau$:
- Var: impossible (empty context).
- Abs: $\lambda x.\, M$ is already a value.
- App: $e = e_1\, e_2$. By IH, $e_1$ steps or is a value. If $e_1$ is a value of function type, it must be $\lambda x.\, M$ (by canonical forms lemma), so $e_1\, e_2 \longrightarrow M[x := e_2]$.

---

## References

- Pierce, B.C. *Types and Programming Languages* (MIT Press, 2002) -- the standard graduate textbook
- Pierce, B.C. (ed.) *Advanced Topics in Types and Programming Languages* (MIT Press, 2005)
- Girard, J.-Y., Lafont, Y., Taylor, P. *Proofs and Types* (Cambridge, 1989)
- Nordstrom, B., Petersson, K., Smith, J. *Programming in Martin-Lof's Type Theory* (Oxford, 1990)
- Milner, R. "A Theory of Type Polymorphism in Programming" (JCSS 17, 1978)
- Damas, L. and Milner, R. "Principal type-schemes for functional programs" (POPL, 1982)
- Girard, J.-Y. "Interpretation fonctionnelle et elimination des coupures" (These de doctorat, 1972)
- Reynolds, J.C. "Towards a theory of type structure" (Colloque sur la Programmation, 1974)
- Howard, W.A. "The formulae-as-types notion of construction" (1969, published 1980)
- Wadler, P. "Theorems for free!" (FPCA, 1989)
- Girard, J.-Y. "Linear logic" (Theoretical Computer Science 50, 1987)
- Wright, A. and Felleisen, M. "A Syntactic Approach to Type Soundness" (Information and Computation 115, 1994)
- Wells, J.B. "Typability and type checking in System F are equivalent and undecidable" (Annals of Pure and Applied Logic 98, 1999)
- Tait, W.W. "Intensional Interpretations of Functionals of Finite Type I" (JSL 32, 1967)
- Martin-Lof, P. *Intuitionistic Type Theory* (Bibliopolis, 1984)
- Coquand, T. and Huet, G. "The Calculus of Constructions" (Information and Computation 76, 1988)
- The Univalent Foundations Program, *Homotopy Type Theory* (Institute for Advanced Study, 2013)
