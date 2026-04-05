# Category Theory -- Composition, Abstraction, and the Mathematics of Structure

> *Category theory is the algebra of functions, formalized. It studies not the objects themselves but the morphisms between them, revealing that mathematical structure is determined by how things relate to one another -- not by what they "are."*

---

## 1. Formal Definition of a Category

### The Problem

Define the notion of a category with full axiomatic precision, establishing the foundation upon which all subsequent constructions rest.

### The Formula

A **category** $\mathcal{C}$ consists of:

1. A collection $\text{ob}(\mathcal{C})$ of **objects**.
2. For each pair of objects $A, B \in \text{ob}(\mathcal{C})$, a set $\text{Hom}_{\mathcal{C}}(A, B)$ of **morphisms** (arrows) from $A$ to $B$.
3. For each triple of objects $A, B, C$, a **composition** function:

$$\circ : \text{Hom}(B, C) \times \text{Hom}(A, B) \to \text{Hom}(A, C)$$

4. For each object $A$, an **identity morphism** $\text{id}_A \in \text{Hom}(A, A)$.

Subject to two axioms:

**Associativity.** For all $f : A \to B$, $g : B \to C$, $h : C \to D$:

$$h \circ (g \circ f) = (h \circ g) \circ f$$

**Identity.** For all $f : A \to B$:

$$f \circ \text{id}_A = f = \text{id}_B \circ f$$

### Worked Examples

**Example 1: The category $\textbf{Set}$.** Objects are sets, morphisms are total functions, composition is function composition, and $\text{id}_A$ is the identity function on $A$. Associativity and identity follow from the corresponding properties of function composition.

**Example 2: A monoid as a one-object category.** Let $(M, \cdot, e)$ be a monoid. Define a category $\mathcal{B}M$ with a single object $\star$, $\text{Hom}(\star, \star) = M$, composition $g \circ f = g \cdot f$, and $\text{id}_\star = e$. The category axioms reduce exactly to the monoid axioms.

**Example 3: A preorder as a thin category.** Let $(P, \leq)$ be a preorder. Define a category with objects $P$, a unique morphism $A \to B$ iff $A \leq B$, and no morphism otherwise. Composition follows from transitivity; identity from reflexivity.

---

## 2. Functors -- Structure-Preserving Maps

### The Problem

Define maps between categories that respect their compositional structure, generalizing the notion of homomorphism from algebra.

### The Formula

A **(covariant) functor** $F : \mathcal{C} \to \mathcal{D}$ consists of:

1. An object mapping: $A \mapsto F(A)$ for each $A \in \text{ob}(\mathcal{C})$.
2. A morphism mapping: $(f : A \to B) \mapsto (F(f) : F(A) \to F(B))$ for each morphism in $\mathcal{C}$.

Satisfying the **functor laws**:

$$F(\text{id}_A) = \text{id}_{F(A)}$$
$$F(g \circ f) = F(g) \circ F(f)$$

A **contravariant functor** $F : \mathcal{C}^{\text{op}} \to \mathcal{D}$ reverses the direction of morphisms:

$$(f : A \to B) \mapsto (F(f) : F(B) \to F(A))$$

### Worked Examples

**Example 1: The List functor in Haskell.**

The `List` type constructor defines an endofunctor on $\textbf{Hask}$:

- On objects: $A \mapsto [A]$
- On morphisms: $f \mapsto \text{map}\ f$

Verification of functor laws:

$$\text{map}\ \text{id}\ [x_1, \ldots, x_n] = [\text{id}\ x_1, \ldots, \text{id}\ x_n] = [x_1, \ldots, x_n]$$

$$\text{map}\ (g \circ f)\ [x_1, \ldots, x_n] = [(g \circ f)\ x_1, \ldots, (g \circ f)\ x_n]$$
$$= [g\ (f\ x_1), \ldots, g\ (f\ x_n)] = \text{map}\ g\ (\text{map}\ f\ [x_1, \ldots, x_n])$$

**Example 2: The Maybe functor.**

- On objects: $A \mapsto \text{Maybe}\ A$
- On morphisms: $\text{fmap}\ f\ \text{Nothing} = \text{Nothing}$, $\text{fmap}\ f\ (\text{Just}\ x) = \text{Just}\ (f\ x)$

Identity law: $\text{fmap}\ \text{id}\ (\text{Just}\ x) = \text{Just}\ (\text{id}\ x) = \text{Just}\ x$. For $\text{Nothing}$: $\text{fmap}\ \text{id}\ \text{Nothing} = \text{Nothing}$.

Composition law: $\text{fmap}\ (g \circ f)\ (\text{Just}\ x) = \text{Just}\ (g\ (f\ x)) = \text{fmap}\ g\ (\text{Just}\ (f\ x)) = \text{fmap}\ g\ (\text{fmap}\ f\ (\text{Just}\ x))$.

**Example 3: The powerset functor $\mathcal{P} : \textbf{Set} \to \textbf{Set}$.**

- On objects: $A \mapsto \mathcal{P}(A)$
- On morphisms: $(f : A \to B) \mapsto (f_* : \mathcal{P}(A) \to \mathcal{P}(B))$ where $f_*(S) = \{f(x) \mid x \in S\}$

The contravariant powerset functor maps $f$ to the preimage: $f^*(T) = \{x \in A \mid f(x) \in T\}$.

---

## 3. Natural Transformations

### The Problem

Define morphisms between functors that respect the structure of the source category, completing the "category of categories" picture.

### The Formula

Given functors $F, G : \mathcal{C} \to \mathcal{D}$, a **natural transformation** $\alpha : F \Rightarrow G$ is a family of morphisms in $\mathcal{D}$:

$$\alpha_A : F(A) \to G(A) \quad \text{for each } A \in \text{ob}(\mathcal{C})$$

such that for every morphism $f : A \to B$ in $\mathcal{C}$, the following **naturality square** commutes:

$$G(f) \circ \alpha_A = \alpha_B \circ F(f)$$

A natural transformation is a **natural isomorphism** if each $\alpha_A$ is an isomorphism.

The **functor category** $[\mathcal{C}, \mathcal{D}]$ (also written $\mathcal{D}^{\mathcal{C}}$) has functors as objects and natural transformations as morphisms.

### Worked Examples

The function $\text{safeHead} : [a] \to \text{Maybe}\ a$ defines a natural transformation from the List functor to the Maybe functor. Naturality requires:

$$\text{fmap}\ f\ (\text{safeHead}\ xs) = \text{safeHead}\ (\text{map}\ f\ xs)$$

Case $xs = []$: both sides yield $\text{Nothing}$.

Case $xs = (x:\_)$: LHS $= \text{fmap}\ f\ (\text{Just}\ x) = \text{Just}\ (f\ x)$. RHS $= \text{safeHead}\ (f\ x : \_) = \text{Just}\ (f\ x)$.

By parametricity (Wadler's "Theorems for Free"), every polymorphic function of type $\forall a.\ F\ a \to G\ a$ (where $F$ and $G$ are functors) is automatically a natural transformation.

---

## 4. Monads -- Formal Theory and Laws

### The Problem

Define monads with full categorical rigor and prove the equivalence between the endofunctor-with-natural-transformations presentation and the Kleisli-triple presentation used in programming.

### The Formula

A **monad** on a category $\mathcal{C}$ is a triple $(T, \eta, \mu)$ where:

- $T : \mathcal{C} \to \mathcal{C}$ is an endofunctor
- $\eta : \text{Id}_{\mathcal{C}} \Rightarrow T$ is a natural transformation (the **unit**)
- $\mu : T^2 \Rightarrow T$ is a natural transformation (the **multiplication**)

satisfying the **monad laws** (coherence conditions):

**Left unit:**
$$\mu_A \circ T(\eta_A) = \text{id}_{T(A)}$$

**Right unit:**
$$\mu_A \circ \eta_{T(A)} = \text{id}_{T(A)}$$

**Associativity:**
$$\mu_A \circ T(\mu_A) = \mu_A \circ \mu_{T(A)}$$

In Haskell notation, with $\text{join} = \mu$, $\text{return} = \eta$, $\text{fmap} = T$ on morphisms:

```haskell
join . fmap return  = id        -- left unit
join . return       = id        -- right unit
join . fmap join    = join . join -- associativity
```

**Equivalence with Kleisli triple.** Given $(T, \eta, \mu)$, define:

$$a \mathbin{>\!\!>\!\!=} f = \mu_B \circ T(f)\ a \quad \text{for } f : A \to T(B)$$

Conversely, given $(\mathbin{>\!\!>\!\!=}, \text{return})$, define:

$$T(f) = \mathbin{>\!\!>\!\!=}\ (\text{return} \circ f), \quad \mu_A = \mathbin{>\!\!>\!\!=}\ \text{id}_{T(A)}$$

### Monad Laws in Dual Notation

| Categorical | Kleisli (Haskell) |
|---|---|
| $\mu \circ T(\eta) = \text{id}$ | `return a >>= f  =  f a` |
| $\mu \circ \eta_T = \text{id}$ | `m >>= return  =  m` |
| $\mu \circ T(\mu) = \mu \circ \mu_T$ | `(m >>= f) >>= g  =  m >>= (\x -> f x >>= g)` |

### Verification: Maybe Monad Laws

Define $T = \text{Maybe}$, $\eta_A(x) = \text{Just}\ x$, $\mu_A(\text{Nothing}) = \text{Nothing}$, $\mu_A(\text{Just}\ \text{Nothing}) = \text{Nothing}$, $\mu_A(\text{Just}\ (\text{Just}\ x)) = \text{Just}\ x$.

**Left unit:** $\mu(\text{fmap}\ \text{return}\ (\text{Just}\ x)) = \mu(\text{Just}\ (\text{Just}\ x)) = \text{Just}\ x = \text{id}(\text{Just}\ x)$.

**Right unit:** $\mu(\text{return}\ (\text{Just}\ x)) = \mu(\text{Just}\ (\text{Just}\ x)) = \text{Just}\ x = \text{id}(\text{Just}\ x)$.

**Associativity:** $\mu(\text{fmap}\ \mu\ (\text{Just}\ (\text{Just}\ (\text{Just}\ x)))) = \mu(\text{Just}\ (\text{Just}\ x)) = \text{Just}\ x$, and $\mu(\mu(\text{Just}\ (\text{Just}\ (\text{Just}\ x)))) = \mu(\text{Just}\ (\text{Just}\ x)) = \text{Just}\ x$.

---

## 5. Kleisli Category and Associativity of Kleisli Composition

### The Problem

Define the Kleisli category associated with a monad and prove that Kleisli composition is associative, demonstrating the equivalence between the monad associativity law and the associativity of this category.

### The Formula

Given a monad $(T, \eta, \mu)$ on $\mathcal{C}$, the **Kleisli category** $\mathcal{C}_T$ is defined:

- **Objects:** $\text{ob}(\mathcal{C}_T) = \text{ob}(\mathcal{C})$
- **Morphisms:** $\text{Hom}_{\mathcal{C}_T}(A, B) = \text{Hom}_{\mathcal{C}}(A, T(B))$ (a morphism $A \to B$ in $\mathcal{C}_T$ is a Kleisli arrow $A \to T(B)$ in $\mathcal{C}$)
- **Identity:** $\eta_A : A \to T(A)$
- **Composition:** For $f : A \to T(B)$ and $g : B \to T(C)$:

$$g \circ_T f = \mu_C \circ T(g) \circ f$$

Or equivalently in Haskell notation (the "fish" operator):

$$(g \mathbin{>\!\!=\!\!>} f)(x) = f(x) \mathbin{>\!\!>\!\!=} g$$

### Proof of Associativity

**Claim:** For $f : A \to T(B)$, $g : B \to T(C)$, $h : C \to T(D)$:

$$h \circ_T (g \circ_T f) = (h \circ_T g) \circ_T f$$

**Proof.** We expand both sides.

Left side:

$$h \circ_T (g \circ_T f) = \mu_D \circ T(h) \circ (\mu_C \circ T(g) \circ f)$$
$$= \mu_D \circ T(h) \circ \mu_C \circ T(g) \circ f$$

Right side:

$$(h \circ_T g) \circ_T f = \mu_D \circ T(\mu_D \circ T(h) \circ g) \circ f$$
$$= \mu_D \circ T(\mu_D) \circ T^2(h) \circ T(g) \circ f$$

By naturality of $\mu$, we have $\mu_D \circ T(h) \circ \mu_C = \mu_D \circ \mu_{T(D)} \circ T^2(h)$. But by the monad associativity law, $\mu_D \circ \mu_{T(D)} = \mu_D \circ T(\mu_D)$. Therefore:

$$\mu_D \circ T(h) \circ \mu_C = \mu_D \circ T(\mu_D) \circ T^2(h)$$

Composing both sides on the right with $T(g) \circ f$, we get LHS $=$ RHS. $\square$

**Identity laws** follow similarly from the unit laws of the monad.

---

## 6. The Yoneda Lemma

### The Problem

State and prove the most fundamental result of category theory: the Yoneda lemma, which characterizes natural transformations out of representable functors.

### The Formula

Let $\mathcal{C}$ be a locally small category, $F : \mathcal{C} \to \textbf{Set}$ a functor, and $A \in \text{ob}(\mathcal{C})$. The **Yoneda lemma** states:

$$\text{Nat}(\text{Hom}(A, -), F) \cong F(A)$$

The bijection $\Phi$ is given by:

$$\Phi(\alpha) = \alpha_A(\text{id}_A)$$

with inverse $\Phi^{-1}$ defined for $x \in F(A)$ as:

$$\Phi^{-1}(x)_B(f) = F(f)(x) \quad \text{for } f : A \to B$$

### Proof Sketch

**Well-defined.** Given $x \in F(A)$, we must show $\Phi^{-1}(x)$ is a natural transformation. For any $g : B \to C$ and $f : A \to B$:

$$\Phi^{-1}(x)_C(g \circ f) = F(g \circ f)(x) = F(g)(F(f)(x)) = F(g)(\Phi^{-1}(x)_B(f))$$

This is exactly the naturality condition.

**$\Phi$ and $\Phi^{-1}$ are mutually inverse.**

$\Phi(\Phi^{-1}(x)) = \Phi^{-1}(x)_A(\text{id}_A) = F(\text{id}_A)(x) = \text{id}_{F(A)}(x) = x$.

$\Phi^{-1}(\Phi(\alpha))_B(f) = F(f)(\alpha_A(\text{id}_A))$. By naturality of $\alpha$: $F(f) \circ \alpha_A = \alpha_B \circ \text{Hom}(A, f)$, so $F(f)(\alpha_A(\text{id}_A)) = \alpha_B(\text{Hom}(A, f)(\text{id}_A)) = \alpha_B(f)$.

**Naturality.** The bijection is natural in both $A$ and $F$.

### Intuition

The Yoneda lemma says: to give a natural transformation from $\text{Hom}(A, -)$ to $F$, it suffices to specify a single element of $F(A)$ -- the image of $\text{id}_A$. Naturality then uniquely determines the rest.

Philosophically: an object $A$ is completely determined by how other objects map into it. This is the "probe" or "generalized element" perspective.

**Corollary (Yoneda embedding).** The functor $\text{y} : \mathcal{C} \to [\mathcal{C}^{\text{op}}, \textbf{Set}]$ defined by $\text{y}(A) = \text{Hom}(-, A)$ is fully faithful. Consequence: $A \cong B$ in $\mathcal{C}$ if and only if $\text{Hom}(-, A) \cong \text{Hom}(-, B)$.

**In programming.** The Yoneda lemma corresponds to the isomorphism:

$$\forall b.\ (a \to b) \to f\ b \;\cong\; f\ a$$

This is the basis of the "codensity" optimization and CPS (continuation-passing style) transformations. Building up composed `fmap` operations as CPS and then "lowering" is asymptotically faster for some functors.

---

## 7. Adjunctions and the Adjunction-Monad Correspondence

### The Problem

Define adjunctions precisely and prove that every adjunction gives rise to a monad, establishing the deep connection between these two fundamental concepts.

### The Formula

An **adjunction** $F \dashv G$ between categories $\mathcal{C}$ and $\mathcal{D}$ consists of functors $F : \mathcal{C} \to \mathcal{D}$ and $G : \mathcal{D} \to \mathcal{C}$ together with a natural isomorphism:

$$\text{Hom}_{\mathcal{D}}(F(A), B) \cong \text{Hom}_{\mathcal{C}}(A, G(B))$$

for all $A \in \text{ob}(\mathcal{C})$, $B \in \text{ob}(\mathcal{D})$.

Equivalently, there exist natural transformations:

$$\eta : \text{Id}_{\mathcal{C}} \Rightarrow G \circ F \quad \text{(unit)}$$
$$\varepsilon : F \circ G \Rightarrow \text{Id}_{\mathcal{D}} \quad \text{(counit)}$$

satisfying the **triangle identities** (zig-zag equations):

$$(\varepsilon_F) \circ (F\eta) = \text{id}_F$$
$$(G\varepsilon) \circ (\eta_G) = \text{id}_G$$

### Adjunction-Monad Theorem

**Theorem.** Every adjunction $F \dashv G$ induces a monad $(T, \eta, \mu)$ on $\mathcal{C}$, where:

$$T = G \circ F$$
$$\eta : \text{Id}_{\mathcal{C}} \Rightarrow T \quad \text{(the unit of the adjunction)}$$
$$\mu = G(\varepsilon_F) : T^2 = G \circ F \circ G \circ F \Rightarrow G \circ F = T$$

**Proof of monad laws.**

Left unit: $\mu \circ T(\eta) = G(\varepsilon_F) \circ G(F(\eta)) = G(\varepsilon_F \circ F(\eta)) = G(\text{id}_F) = \text{id}_T$, using the first triangle identity.

Right unit: $\mu \circ \eta_T = G(\varepsilon_F) \circ \eta_{GF} = G(\varepsilon_F) \circ \eta_{GF}$. By naturality of $\eta$ applied to $\varepsilon_F$, this equals $G(\text{id}) = \text{id}_T$, using the second triangle identity.

Associativity: follows from naturality of $\varepsilon$.

### Example: Free-Forgetful Adjunction

The **free functor** $F : \textbf{Set} \to \textbf{Mon}$ sends a set $S$ to the free monoid $S^*$ (lists over $S$). The **forgetful functor** $U : \textbf{Mon} \to \textbf{Set}$ forgets the monoid structure. Then $F \dashv U$, and the induced monad $T = U \circ F$ sends a set $S$ to $S^*$ -- the **list monad**.

The unit $\eta_S(x) = [x]$ wraps an element in a singleton list. The multiplication $\mu_S = \text{concat}$ flattens a list of lists. This is exactly `return` and `join` for the list monad in Haskell.

### Converse Direction

Not every monad arises from a unique adjunction. Given a monad $(T, \eta, \mu)$, there are two canonical factorizations:

1. **Kleisli adjunction:** $F_T \dashv G_T$ where $F_T : \mathcal{C} \to \mathcal{C}_T$ and $G_T : \mathcal{C}_T \to \mathcal{C}$.
2. **Eilenberg-Moore adjunction:** $F^T \dashv U^T$ where $U^T : \mathcal{C}^T \to \mathcal{C}$ and $\mathcal{C}^T$ is the category of $T$-algebras.

The Kleisli category is the initial factorization and the Eilenberg-Moore category is the terminal factorization, in the category of adjunctions that give rise to $T$.

---

## 8. Free Monads

### The Problem

Define free monads as a construction that separates the description of a computation from its interpretation, yielding a powerful pattern for building embedded domain-specific languages.

### The Formula

Given an endofunctor $F : \mathcal{C} \to \mathcal{C}$, the **free monad** $\text{Free}(F)$ is defined as the initial $F$-algebra of the functor $X \mapsto \text{Id} + F \circ X$, or equivalently as the fixpoint:

$$\text{Free}(F)(A) = A + F(\text{Free}(F)(A))$$

In Haskell:

```haskell
data Free f a = Pure a | Free (f (Free f a))

instance Functor f => Monad (Free f) where
    return = Pure
    Pure a >>= f = f a
    Free m >>= f = Free (fmap (>>= f) m)
```

The monad structure is:

- $\eta_A(a) = \text{Pure}(a)$ -- inject a pure value
- $\mu_A = \text{join}$ -- collapse nested `Free` layers

**Key property.** There is an adjunction between the category of endofunctors on $\mathcal{C}$ and the category of monads on $\mathcal{C}$. The free monad construction is the left adjoint to the forgetful functor that sends a monad to its underlying endofunctor.

**Interpretation.** A natural transformation $\alpha : F \Rightarrow G$ (where $G$ is a monad) extends uniquely to a monad morphism $\text{foldFree}(\alpha) : \text{Free}(F) \Rightarrow G$:

```haskell
foldFree :: Monad m => (forall x. f x -> m x) -> Free f a -> m a
foldFree _   (Pure a) = return a
foldFree phi (Free m) = phi m >>= foldFree phi
```

This separation of syntax (the free monad) from semantics (the interpreter $\alpha$) is the essence of the "free monad pattern" in functional programming.

---

## 9. Relationship to Type Theory and Logic

### The Problem

Establish the Curry-Howard-Lambek correspondence, a three-way isomorphism connecting category theory, type theory, and formal logic.

### The Formula

The **Curry-Howard-Lambek correspondence** is a three-way dictionary:

| Category Theory | Type Theory | Logic |
|---|---|---|
| Object | Type | Proposition |
| Morphism $A \to B$ | Term of type $A \to B$ | Proof of $A \Rightarrow B$ |
| Composition | Function composition | Transitivity of implication |
| Identity | Identity function | Reflexivity |
| Product $A \times B$ | Pair type $(A, B)$ | Conjunction $A \wedge B$ |
| Coproduct $A + B$ | Sum type `Either A B` | Disjunction $A \vee B$ |
| Exponential $B^A$ | Function type $A \to B$ | Implication $A \Rightarrow B$ |
| Terminal object $1$ | Unit type `()` | Truth $\top$ |
| Initial object $0$ | Void / Empty type | Falsity $\bot$ |
| $0 \to A$ | `absurd :: Void -> a` | Ex falso quodlibet |
| CCC | Simply typed $\lambda$-calculus | Intuitionistic propositional logic |
| Topos | Dependent type theory | Higher-order intuitionistic logic |

**Theorem (Lambek, 1980).** The category of simply typed $\lambda$-calculi (with $\beta\eta$-equivalence) is equivalent to the category of cartesian closed categories with chosen structure.

This means:

1. Every CCC gives rise to a simply typed $\lambda$-calculus (its internal language).
2. Every simply typed $\lambda$-calculus generates a free CCC (its syntactic category).

**Monads in the correspondence.** Under Moggi's computational interpretation (1991), a monad $T$ on a CCC represents a notion of computation:

| Monad $T$ | Computational effect |
|---|---|
| $\text{Maybe}$ | Partiality (possible failure) |
| $\text{List}$ | Nondeterminism |
| $\text{State}\ s$ | Mutable state |
| $\text{Reader}\ r$ | Environment / configuration |
| $\text{Writer}\ w$ | Logging / accumulation |
| $\text{IO}$ | General side effects |
| $\text{Cont}\ r$ | Continuations |

The Kleisli category of $T$ is then the "effectful" version of the base category: morphisms $A \to T(B)$ are "computations that take an $A$ and produce a $B$ with possible effects."

---

## 10. Cartesian Closed Categories and the Lambda Calculus

### The Problem

Show that cartesian closed categories provide the exact categorical semantics for the simply typed lambda calculus.

### The Formula

A category $\mathcal{C}$ is **cartesian closed** (CCC) if:

1. It has a **terminal object** $1$ (the product of zero objects).
2. For all $A, B \in \text{ob}(\mathcal{C})$, the **product** $A \times B$ exists.
3. For all $A, B \in \text{ob}(\mathcal{C})$, the **exponential** $B^A$ exists.

The exponential $B^A$ is characterized by a natural isomorphism (currying):

$$\text{Hom}(A \times B, C) \cong \text{Hom}(A, C^B)$$

with an **evaluation morphism** $\text{ev} : B^A \times A \to B$ that is universal.

**Interpretation of the simply typed $\lambda$-calculus in a CCC:**

- Types are objects: base types map to chosen objects; $\sigma \to \tau$ maps to $\tau^\sigma$.
- A typing context $\Gamma = x_1 : \sigma_1, \ldots, x_n : \sigma_n$ maps to the product $\sigma_1 \times \cdots \times \sigma_n$.
- A term $\Gamma \vdash M : \tau$ maps to a morphism $\llbracket \Gamma \rrbracket \to \llbracket \tau \rrbracket$.
- Lambda abstraction $\lambda x.\, M$ maps to currying.
- Application $M\; N$ maps to evaluation composed with pairing.
- Beta reduction corresponds to the equation $\text{ev} \circ \langle \text{curry}(f), g \rangle = f \circ \langle \text{id}, g \rangle$.

**Theorem.** The equational theory of $\beta\eta$-equivalence in the simply typed $\lambda$-calculus is sound and complete with respect to interpretation in CCCs.

---

## Prerequisites

- Abstract algebra (groups, monoids, homomorphisms)
- Basic topology (for motivating examples; not strictly required)
- Familiarity with Haskell or ML (for programming examples)
- Lambda calculus (for the CCC correspondence)
- Mathematical maturity with proofs by induction and diagram chasing

## Complexity

| Level | Description |
|-------|-------------|
| **Beginner** | Define categories, functors, and natural transformations. Verify functor laws for List and Maybe. Identify products and coproducts in Set. State the monad laws in Kleisli form. Use Maybe, Either, and List as monads in Haskell. |
| **Intermediate** | Prove that Kleisli composition is associative from the monad laws. State the Yoneda lemma and compute examples. Construct the free monad on a functor. Verify the triangle identities for a given adjunction. Translate between the endofunctor and Kleisli presentations of a monad. |
| **Advanced** | Prove the Yoneda lemma in full generality. Establish the adjunction-monad correspondence. Construct Eilenberg-Moore algebras for specific monads. Relate CCCs to the simply typed lambda calculus via the Lambek correspondence. Study enriched categories, 2-categories, and higher categorical structures. Explore topos-theoretic models of intuitionistic logic. |
