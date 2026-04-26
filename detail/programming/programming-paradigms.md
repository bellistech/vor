# Programming Paradigms — Theory

A deep dive into the formal foundations of programming paradigms — lambda calculus, type theory, semantic models, and the mathematical structures that underlie the patterns. The user-facing pattern catalog lives in `sheets/programming/programming-paradigms.md`; this page is the theory companion.

## Setup

A programming paradigm is not merely a coding style or a set of conventions imposed by a community. It is a *theoretical model of computation* — a mathematical commitment about what programs *are*, how they execute, and what equational reasoning is sound. The paradigm shapes what the type system can say, what optimisations the compiler may apply, and what proofs the programmer can write about programs.

The four classical paradigms — imperative, functional, logic, object-oriented — correspond to four answers to the question "what is a program?":

- **Imperative** — a program is a sequence of state-modifying instructions, executed in order on an abstract Turing-style machine. Its mathematical model is the *transition system*: configurations connected by labelled edges.
- **Functional** — a program is an expression that *denotes a value*. Its model is the lambda calculus or, equivalently, a Cartesian-closed category. Execution is *evaluation*: rewriting an expression to its normal form.
- **Logic** — a program is a set of facts and rules from which the runtime *derives* answers by resolution. Its model is first-order logic (Horn clause fragment) and its execution is proof search.
- **Object-oriented** — a program is a population of communicating agents, each holding state and responding to messages. Its model is the actor calculus or the π-calculus, sometimes specialised to a record-of-functions semantics.

Modern languages mix these models. Haskell is functional with a typed effect layer (IO, ST, STM monads). Erlang is functional outside actors and message-passing inside. Rust is imperative but with linear-type-discipline borrowing rules that give a sound mutation calculus. The point of paradigm theory is not to label languages but to give the *semantic vocabulary* in which their behaviour is defined and provable.

There are three orthogonal axes that paradigm theory keeps revisiting:

- **What is a value?** A bitstring? An element of a domain (CPO)? A proof? A wave function?
- **What is a function?** A black box? A morphism in a category? A logical implication?
- **What is execution?** Rewriting? Search? Message dispatch? Unitary evolution?

The remainder of this page traces the formal answers from Church's lambda calculus through dependent types, monads, the Curry-Howard correspondence, and the calculi of concurrency.

```text
                  Operational    Denotational   Axiomatic
                  semantics      semantics      semantics
                       \             |              /
                        \            |             /
                         +-----------+------------+
                                     |
                          A program "means" ...
                                     |
                +--------------------+---------------------+
                |             |             |              |
            Imperative   Functional      Logic        Concurrent
                |             |             |              |
            Turing       Lambda calc    Horn clauses   π-calculus
            machine      / CCC          / SLD-res      / actors
```

## The Lambda Calculus (Church 1936)

Alonzo Church introduced the lambda calculus in 1932-1936 as a formal system for the foundations of mathematics, three years before Turing's machine. It is the simplest *non-trivial* universal model of computation: only three syntactic forms, three rewrite rules, and yet expressively complete.

**Syntax** — terms M, N are built from:

```
M, N ::= x          -- variable
       | λx. M      -- abstraction (anonymous function)
       | M N        -- application
```

That's it. Numbers, booleans, lists, pairs — all are encoded.

**Free and bound variables** — In `λx. M`, the binder `λx` *binds* the variable `x` in the body `M`. A variable is *free* if it is not bound by any enclosing `λ`. The set FV(M) of free variables is computed inductively:

```
FV(x)        = { x }
FV(λx. M)    = FV(M) \ { x }
FV(M N)      = FV(M) ∪ FV(N)
```

A *closed term* (or *combinator*) is one with FV(M) = ∅.

**Three reduction rules:**

**α-conversion** — bound names are placeholders. Renaming a bound variable does not change meaning, provided the new name does not collide with a free variable:

```
λx. x   ≡α   λy. y
λx. (x y) ≡α λz. (z y)    -- x → z is fine, x → y would capture
```

α-equivalence is the smallest congruence respecting the renaming rule. We work modulo α throughout.

**β-reduction** — *the* computation rule. Substitution of the argument into the body:

```
(λx. M) N   →β   M[x := N]
```

with the proviso that capture-avoiding substitution `M[x := N]` α-renames bound variables of `M` to avoid capturing free variables of `N`.

**η-conversion** — function extensionality. If `M` does not contain `x` free, then:

```
λx. (M x)   →η   M
```

η reflects the principle that a function that just calls another function is *equal* to that function. Some semantic frameworks include η, others (e.g. operational evaluation under call-by-value) omit it.

**β-redex** — a sub-term of the form `(λx. M) N`; β-reduction *contracts* a redex.

**Normal form** — a term with no β-redex. A term is *strongly normalising* (SN) if every reduction sequence reaches a normal form; *weakly normalising* (WN) if some reduction sequence does. The Church-Rosser theorem (a.k.a. *confluence*) states:

> If `M →* N₁` and `M →* N₂`, then there exists `P` with `N₁ →* P` and `N₂ →* P`.

Confluence implies that *if* a term has a normal form, that normal form is unique up to α. The order of reductions does not change the answer — only whether you find one.

## Untyped Lambda Calculus

The untyped λ-calculus is *Turing-complete*: any partial computable function is definable. The proof goes through Kleene's recursion theorem, but constructively the magic ingredient is the **Y combinator**, which makes recursion definable from anonymous abstraction alone.

**The Y combinator:**

```
Y = λf. (λx. f (x x)) (λx. f (x x))
```

Verify the fixed-point property:

```
Y f
  = (λx. f (x x)) (λx. f (x x))             -- by def
  →β f ((λx. f (x x)) (λx. f (x x)))        -- β at outer redex
  = f (Y f)
```

So `Y f =β f (Y f)`. Any function `f` of one argument has the property that `Y f` is a fixed point of `f`. To define recursion, you write the *body* of your recursive function as `f`, and `Y f` is the recursive function itself. Example: factorial.

```
fact-step = λr. λn. if (iszero n) 1 (mul n (r (pred n)))
fact      = Y fact-step
```

`Y` is one of many fixed-point combinators. Turing's variant is:

```
Θ = (λx y. y (x x y)) (λx y. y (x x y))
```

with the property `Θ f →β f (Θ f)` (slightly cleaner reduction).

**Church numerals** — natural numbers as iteration. The numeral `n` takes a function and a base case and applies the function `n` times:

```
0   = λf. λx. x
1   = λf. λx. f x
2   = λf. λx. f (f x)
3   = λf. λx. f (f (f x))
n   = λf. λx. fⁿ x
```

with arithmetic:

```
succ  = λn. λf. λx. f (n f x)
add   = λm. λn. λf. λx. m f (n f x)
mul   = λm. λn. λf. m (n f)
exp   = λm. λn. n m                         -- m to the power n
```

(Predecessor is famously trickier; Church spent a year before Kleene found `pred`. See Pierce TAPL §5.2.)

**Church booleans** — branching as selection. A boolean is a 2-argument function picking either its first or second:

```
true   = λx. λy. x
false  = λx. λy. y
if     = λb. λt. λe. b t e         -- == b t e on its own
not    = λb. b false true
and    = λp. λq. p q false
or     = λp. λq. p true q
```

**Pairs and projections:**

```
pair   = λx. λy. λs. s x y
fst    = λp. p (λx. λy. x)
snd    = λp. p (λx. λy. y)
```

With pairs, lists, and Y, we can encode any total or partial recursive function. Untyped λ-calculus is the bedrock of every functional language.

## The Halting Problem in Lambda Calculus

Turing's halting problem has a direct λ-calculus analogue: the **normalisation problem**. Given a closed term `M`, does there exist a normal form `N` with `M →* N`?

**Theorem (Church 1936)** — The normalisation problem is undecidable.

The proof mirrors Turing's diagonalisation. Suppose `H : Λ → {true, false}` decides whether a term has a normal form. Build a term:

```
B = λx. if (H (x x)) Ω x
```

where `Ω = (λy. y y)(λy. y y)` is the canonical non-terminating term. Now consider `B B`:

- If `B B` *would* have a normal form, then `H (B B)` returns `true`, so `B B →β Ω`, which has no normal form — contradiction.
- If `B B` would not have a normal form, then `H (B B)` returns `false`, so `B B →β B`, which is a normal form — contradiction.

Hence no such `H` exists. Equivalently, *β-equivalence is undecidable*: there is no algorithm deciding `M =β N`.

This is the *foundational price* of Turing-completeness. To regain decidability, you must restrict the calculus — which the simply-typed λ-calculus does.

## Simply-Typed Lambda Calculus (STLC)

Adding types to the lambda calculus changes the picture dramatically. Church's STLC of 1940 introduces:

```
Types τ, σ ::= ι              -- base types (e.g. Nat, Bool)
             | τ → σ          -- arrow type (function from τ to σ)

Terms M, N ::= x
             | λx:τ. M
             | M N
```

Typing judgement `Γ ⊢ M : τ` is given by three rules (Γ is a context, a partial map from variables to types):

```
                        Γ, x:τ ⊢ M : σ
   ─────────────  Var   ─────────────────  Abs
   Γ, x:τ ⊢ x : τ       Γ ⊢ λx:τ. M : τ→σ

         Γ ⊢ M : τ → σ      Γ ⊢ N : τ
         ─────────────────────────────  App
                  Γ ⊢ M N : σ
```

A term is *well-typed* if the rules derive a typing for it. Type checking and type inference for STLC are *decidable* and *linear* in the size of the term.

**Strong normalisation theorem (Tait 1967)** — Every well-typed STLC term is strongly normalising. The proof uses *reducibility candidates* (a.k.a. *logical relations*): a predicate Red(τ) on terms is defined by induction on τ such that

- Red(ι) = SN terms of type ι
- Red(τ → σ) = terms `M : τ → σ` such that for every `N` in Red(τ), `M N` is in Red(σ)

One shows by induction that every well-typed term is in Red of its type, hence SN.

**Loss of expressiveness** — STLC is *not* Turing-complete. The Y combinator is *not typable*: `λx. f (x x)` requires `x : τ` and `x : τ → σ` simultaneously, which has no STLC solution. Total functional languages (Coq, Agda, Idris in Total mode) embrace this loss to gain decidability and consistency.

**STLC as a metric** — every modern type system is "STLC plus stuff": polymorphism, recursion, subtyping, dependent types, effects. The base case is what you fall back to when the extensions are stripped.

## Hindley-Milner Type System (1969 / 1978)

Independently developed by J. Roger Hindley (1969) for combinatory logic and Robin Milner (1978) for ML, the *Hindley-Milner* type system extends STLC with **let-polymorphism**: a `let` binding can have a *polymorphic* type that is freshly instantiated at each use site.

**Types and schemes:**

```
Types       τ ::= α | ι | τ → σ
Type schemes σ ::= τ | ∀α. σ
```

A *scheme* `∀α. τ` is a polymorphic type — a type with universally quantified variables. Variables in let-bindings get schemes; lambda-bound variables get *types*.

**Generalisation and instantiation:**

```
Γ ⊢ M : τ      α ∉ FV(Γ)
─────────────────────────  Gen
   Γ ⊢ M : ∀α. τ

Γ ⊢ M : ∀α. τ
─────────────  Inst
Γ ⊢ M : τ[α := σ]
```

The classical example:

```
let id = λx. x in (id 3, id "hi")
```

In STLC this is ill-typed: `id` would have to choose between `Int → Int` and `String → String`. In HM, `id : ∀α. α → α` is generalised, then instantiated separately at each use.

**Algorithm W (Milner)** — a constraint-generating type-inference procedure. Walking the term, W collects equality constraints between *type variables* and solves them by *unification*. The output is a *principal type scheme* — the most general type any extension could derive. HM's *principal types theorem* says:

> Every well-typed HM term has a unique most-general type scheme up to α-renaming.

Algorithm W in pseudocode:

```
W(Γ, x)        = let ∀α₁..αₙ. τ = Γ(x) in (id, τ[αᵢ := βᵢ fresh])
W(Γ, λx. M)    = let β fresh, (S, τ) = W(Γ ∪ {x:β}, M) in (S, S(β) → τ)
W(Γ, M N)      = let (S₁, τ₁) = W(Γ, M),
                     (S₂, τ₂) = W(S₁(Γ), N),
                     β fresh,
                     S₃ = mgu(S₂(τ₁), τ₂ → β)
                 in (S₃ ∘ S₂ ∘ S₁, S₃(β))
W(Γ, let x = M in N)
               = let (S₁, τ₁) = W(Γ, M),
                     σ = generalise(S₁(Γ), τ₁),
                     (S₂, τ₂) = W(S₁(Γ) ∪ {x:σ}, N)
                 in (S₂ ∘ S₁, τ₂)
```

`mgu` is Robinson's most-general unifier; `generalise` quantifies free type variables not appearing in the context.

**The let-vs-lambda asymmetry** — HM does *not* support polymorphism on lambda-bound variables (sometimes called "system F's first-class polymorphism"); the limitation is what makes inference decidable. To recover lambda-polymorphism you need rank-N types, at which point inference becomes undecidable (rank-3 and beyond) or partial (rank-2).

HM is the type system of ML, OCaml, F#, and the surface fragment of Haskell (extended with type classes, GADTs, type families, etc.).

## System F (Polymorphic Lambda Calculus)

System F was introduced by Girard (1972) for proof theory and by Reynolds (1974) for parametric polymorphism in programming. Where HM has *let*-polymorphism, System F has *first-class* polymorphism.

**Syntax:**

```
Types τ ::= α | τ → σ | ∀α. τ
Terms M ::= x | λx:τ. M | M N | Λα. M | M [τ]
```

`Λα. M` is *type abstraction*; `M [τ]` is *type application*. The β-rules now include:

```
(λx:τ. M) N   →β   M[x := N]
(Λα. M) [σ]   →β   M[α := σ]            -- type-β
```

**Typing rules:**

```
Γ, x:τ ⊢ M : σ                    Γ ⊢ M : ∀α. τ
─────────────────  Abs            ──────────────────  TyApp
Γ ⊢ λx:τ. M : τ→σ                 Γ ⊢ M [σ] : τ[α := σ]

       Γ ⊢ M : τ      α ∉ FV(Γ)
       ──────────────────────────  TyAbs
            Γ ⊢ Λα. M : ∀α. τ
```

Polymorphic identity:

```
id  : ∀α. α → α
id  = Λα. λx:α. x
id [Int] 5         = 5
id [Bool] true     = true
```

**Strong normalisation theorem (Girard 1972)** — Every well-typed System F term is strongly normalising. The proof requires *impredicative* logical relations and is one of the deepest results in type theory.

**Type inference for System F is undecidable** (Wells 1994). Surface languages restrict to rank-1 (HM) or rank-2 polymorphism, or require explicit type annotations. Haskell's `RankNTypes` extension exposes System F directly with manual annotations.

**Parametricity (Reynolds 1983)** — *Theorems for free*. A polymorphic type constrains the function so much that the function's behaviour is determined by its type up to a parametric relation. For example, the only function of type `∀α. α → α` is the identity. The only function of type `∀α. α → α → α` is one of two projections. This *parametricity* is the formal underpinning of Haskell's "free theorems" and an important guide for API design.

## Dependent Types (Martin-Löf 1972)

Per Martin-Löf's *intuitionistic type theory* extends STLC by allowing types to *depend on values*. The fundamental new constructs are:

**Π-types (dependent function)** — `Π(x:A). B(x)` is the type of functions taking `x:A` and returning a value of type `B(x)`. When `B` does not depend on `x`, this collapses to the ordinary arrow `A → B`.

```
Vec : Nat → Type → Type        -- length-indexed lists
nil  : Vec 0 A
cons : A → Vec n A → Vec (n+1) A

append : Π(n m : Nat) (A : Type). Vec n A → Vec m A → Vec (n+m) A
```

**Σ-types (dependent pair)** — `Σ(x:A). B(x)` is the type of pairs `(a, b)` where `a:A` and `b:B(a)`. Generalises both the product type and existential quantification.

```
Pair-of-vec-and-its-length : Σ(n : Nat). Vec n Int
```

**Identity types** — `Id A x y` (or `x ≡ y` informally) is the type of *proofs* that `x` and `y` (both of type `A`) are equal. The constructor is `refl : Π(x:A). Id A x x`.

**Universe hierarchy** — `Type₀ : Type₁ : Type₂ : ...` to avoid Russell-style paradoxes. Calling all of them just `Type` (impredicative) breaks consistency, as Girard's paradox shows.

**Why it matters** — types can express specifications. The type `Π(n:Nat). Π(v: Vec (n+1) Int). { x : Int | x ∈ v }` is the type of a *partial function* that, given a non-empty vector, returns an element of it; the implementation cannot lie about its specification because the type-checker rejects everything else.

This is the **Curry-Howard correspondence** in full strength: programs *are* proofs.

## Curry-Howard Correspondence

Discovered independently by Haskell Curry (1934, in combinatory logic) and W.A. Howard (1969, for the typed λ-calculus), the correspondence equates:

| Logic                  | Type theory                |
| ---------------------- | -------------------------- |
| Proposition            | Type                       |
| Proof                  | Program (closed term)      |
| Implication `A → B`    | Function type `A → B`      |
| Conjunction `A ∧ B`    | Product type `A × B`       |
| Disjunction `A ∨ B`    | Sum type `A + B`           |
| True (⊤)               | Unit type `()`             |
| False (⊥)              | Empty type `⊥` / `Void`    |
| Universal `∀x. P(x)`   | Π-type `Π(x:A). P(x)`      |
| Existential `∃x. P(x)` | Σ-type `Σ(x:A). P(x)`      |
| Modus ponens           | Function application       |
| Implication intro      | λ-abstraction              |

A *closed term* of type `T` is, equivalently, a constructive proof of the proposition that `T` denotes. β-reduction is *cut-elimination* in the corresponding sequent calculus. Strong normalisation of STLC is *consistency* of intuitionistic propositional logic.

**Practical consequence — proof assistants:**

- **Coq** (Inria, Calculus of Inductive Constructions, 1989-) — used to verify the four-colour theorem, the Feit-Thompson theorem, the CompCert C compiler.
- **Agda** (Norell, 2007-) — pure dependent type theory; *the* teaching language for dependent types.
- **Lean** (de Moura, MS Research, 2013-; Lean 4 is now mainstream) — used in Mathlib, the largest formalised mathematics library.
- **Idris** (Brady, 2011-) — dependently typed *programming* language with effects-as-monads (Idris 1) or algebraic effects (Idris 2).

In each, you write a program whose type is a theorem; the type-checker certifies the proof. The same compiler that checks `merge_sort_returns_a_sorted_permutation : ∀l. is_sorted (merge_sort l) ∧ is_perm l (merge_sort l)` *also* runs `merge_sort` on real inputs.

## Operational Semantics (Plotkin 1981)

Operational semantics defines program meaning by *how it executes*. Plotkin's *Structural Operational Semantics* (SOS) is the dominant style; the alternative *natural semantics* (Kahn 1987) is also called *big-step*.

**Small-step semantics** — define a relation `M → M'` ("M reduces in one step to M'"). For STLC with call-by-value:

```
        N → N'
─────────────────────  ξ-app-1
M N  →  M N'

V is a value
       M → M'
─────────────────────  ξ-app-2
V M  →  V M'

V is a value
─────────────────────────  β-cbv
(λx. M) V  →  M[x := V]
```

The *evaluation contexts* style (Felleisen) writes the same rules more compactly:

```
E[(λx. M) V]  →  E[M[x := V]]    -- E is an evaluation context
```

**Big-step semantics** — `M ⇓ V` ("M evaluates to value V"). For the same calculus:

```
                 M ⇓ λx. M'    N ⇓ V'    M'[x := V'] ⇓ V
─────────  Var   ──────────────────────────────────────────  App
V ⇓ V            M N ⇓ V
```

Big-step is closer to a recursive interpreter; small-step is closer to a virtual machine. Big-step *cannot* distinguish *non-termination* from *undefinedness*; small-step can (an *infinite* reduction sequence vs *no* reducing rule applicable).

**Reduction strategies:**

- **Normal-order** — leftmost outermost redex first. Finds a normal form if one exists.
- **Applicative-order** — leftmost innermost. The standard call-by-value strategy.
- **Call-by-name** — argument passed unevaluated; substitutes into body and reduces.
- **Call-by-value** — argument fully evaluated before substitution.
- **Call-by-need (lazy)** — call-by-name with sharing: each thunk evaluated at most once. Haskell's strategy.

Plotkin's *Call-by-value, call-by-name and the λ-calculus* (1975) proved each strategy corresponds to a sound distinct calculus and gave the *CPS-translation* mapping CBV programs to CBN.

## Denotational Semantics (Scott-Strachey)

Denotational semantics, developed by Christopher Strachey and Dana Scott in the late 1960s, gives meaning to programs as *mathematical objects* — typically *continuous functions* on *complete partial orders* (CPOs).

**The challenge of recursion** — the equation `f = λx. if x = 0 then 1 else x * f (x-1)` defines factorial recursively. As a mathematical equation it has multiple solutions (e.g. partial functions extending factorial); the *intended* solution is the *least defined* one. Domain theory makes this precise.

**Domain theory:**

- A **CPO** is a poset `(D, ⊑)` with a least element `⊥` ("undefined") and *least upper bounds* (lubs) for all directed subsets.
- A function `f : D → E` is **continuous** if it preserves directed lubs: `f(⊔X) = ⊔ f(X)` for all directed `X`.
- The **Knaster-Tarski / Kleene fixed-point theorem** — every continuous `f : D → D` has a *least fixed point* `fix(f) = ⊔ₙ fⁿ(⊥)`.

**Denotation of a recursive function** — `[[ μf. F[f] ]] = fix([[ F ]])` where `[[ F ]] : (D → D) → (D → D)` is the continuous functional defined by the body. Factorial is the least fixed point of:

```
F : (Nat → Nat⊥) → (Nat → Nat⊥)
F(g)(n) = if n = 0 then 1 else n * g(n-1)
```

(`Nat⊥` is naturals plus a `⊥` element representing non-termination.)

**Adequacy theorem** — for STLC plus `fix`, the denotational semantics agrees with operational: `M ⇓ V` iff `[[M]] = [[V]] ≠ ⊥`. *Full abstraction* is the converse — denotational equality matches *contextual* operational equivalence — and is famously hard for languages with side effects (the *full abstraction problem* for PCF).

**Why use denotational semantics:**

- *Compositional* — `[[M N]]` is defined in terms of `[[M]]` and `[[N]]`. Any program-equivalence is automatically a congruence.
- *Mathematical* — once you have `[[ ]]`, you can prove things by ordinary set-theoretic / order-theoretic arguments.
- *Optimisation justification* — replacing a sub-term by another with equal denotation is *always* sound.

The Scott-Strachey programme was the inspiration for the development of category-theoretic semantics, monadic semantics (Moggi 1991) and, eventually, the move from *denotation as set-theoretic function* to *denotation as morphism in a categorical model*.

## Axiomatic Semantics (Hoare 1969)

Tony Hoare's *An Axiomatic Basis for Computer Programming* (1969) defined program meaning through a *deductive system* over **Hoare triples**:

```
{P}  S  {Q}
```

Read: "if precondition `P` holds before executing `S`, and `S` terminates, then postcondition `Q` holds after." This is the *partial-correctness* triple; the *total-correctness* triple `[P] S [Q]` additionally asserts that `S` terminates.

**Hoare logic rules** (for a simple imperative language):

```
                          {P}  S₁  {Q}    {Q}  S₂  {R}
──────────────  Skip      ────────────────────────────  Seq
{P}  skip  {P}            {P}  S₁; S₂  {R}

──────────────────────────  Assign      (note: substitution in P)
{P[x := E]}  x := E  {P}

{P ∧ B}  S₁  {Q}    {P ∧ ¬B}  S₂  {Q}
──────────────────────────────────────  If
{P}  if B then S₁ else S₂  {Q}

         {P ∧ B}  S  {P}
──────────────────────────────  While   -- P is the loop invariant
{P}  while B do S  {P ∧ ¬B}

P' ⇒ P    {P}  S  {Q}    Q ⇒ Q'
─────────────────────────────────  Consequence
       {P'}  S  {Q'}
```

The **assignment axiom** is the trickiest. To compute "what is true *after* `x := E`?", substitute *backwards*: anything provable about `x` *after* must hold for `E` *before*.

**Loop invariants** — the heart of imperative correctness proofs. A loop's invariant is a property that:

1. Holds before the loop starts.
2. Is preserved by each iteration.
3. With the loop guard's negation, implies the desired postcondition.

Finding the right invariant is a creative act; tools like Dafny, Why3, Frama-C, and Verifast assist by checking annotations.

**Weakest precondition (Dijkstra 1975)** — `wp(S, Q)` is the *weakest* precondition guaranteeing `Q` after `S`:

```
wp(skip, Q)              = Q
wp(x := E, Q)            = Q[x := E]
wp(S₁; S₂, Q)            = wp(S₁, wp(S₂, Q))
wp(if B then S₁ else S₂, Q) = (B ⇒ wp(S₁, Q)) ∧ (¬B ⇒ wp(S₂, Q))
wp(while B do S, Q)      = least fixed point of F(X) = (B ⇒ wp(S, X)) ∧ (¬B ⇒ Q)
```

A program's correctness reduces to checking `P ⇒ wp(S, Q)` — a verification condition discharged by a theorem prover.

## Imperative Programming

Imperative programming is the paradigm of the *von Neumann machine*: a stored-program computer where state lives in memory cells and execution is a sequence of state-modifying instructions. The model is:

```
state  =  memory × registers × program counter
step    : state → state
program : state → state*  (sequence of states)
```

Variables are *names for memory cells*; assignment overwrites; branching changes the program counter; loops repeat. The fundamental imperative operations are *sequence* (`S₁; S₂`), *branch* (`if B then S₁ else S₂`), and *loop* (`while B do S`).

**Properties:**

- **Mutable state** — same expression evaluated at different points may yield different values. `x` after `x := 5` is `5`; after `x := x + 1` it is `6`.
- **Sequence dependence** — `S₁; S₂` and `S₂; S₁` generally differ. Reordering requires *data dependence analysis*.
- **Side effects** — an expression's value is not its only observable: it may also mutate the heap, perform I/O, raise an exception.

**Hoare logic is *the* axiomatic semantics for imperative languages** because it directly axiomatises assignment. Loop invariants substitute for the structural induction available in functional languages.

**The "side effect = mutation" view** — in imperative languages, *time* is built in. The history of a memory cell is a sequence of values; reasoning involves *temporal logics* (LTL, CTL) to express "always", "eventually", "until". Tools like SPIN, NuSMV, TLA+ are designed for this temporal world.

**Dominant languages** — C, C++, Java, Python (despite higher-order functions), Go, Rust (with linear typing), JavaScript (despite closures). Almost every "industrial" language has a fundamentally imperative semantics, because that's what the machine is.

## Functional Programming

Functional programming takes the *opposite* metaphysical commitment: a program is an expression *denoting a value*; execution is *evaluation*. There is no built-in notion of time, sequence, or mutable state. The model is the lambda calculus.

**Properties:**

- **First-class functions** — functions are values: assignable, returnable, storable. Higher-order functions `(a → b) → (c → d)` compose the program structure.
- **Immutability** — once bound, a name retains its value. `x = 5` does not "make `x` equal `5`"; it *defines* `x` to mean `5`.
- **Referential transparency** — replacing an expression by its value never changes the program's meaning. This is *the* enabling property for equational reasoning, fearless refactoring, and compiler optimisation.

**Equational reasoning:**

```
map f (xs ++ ys)  =  map f xs ++ map f ys           -- map distributes over ++
map (f . g)       =  map f . map g                  -- map preserves composition
foldr f z (xs ++ ys) = foldr f (foldr f z ys) xs    -- foldr fusion law
```

These are *theorems*, provable by induction on lists, holding for all `f`, `g`, `xs`, `ys` (in a pure language).

**No-time view** — `let x = expensive() in x + x` is the same as `expensive() + expensive()` *denotationally*; *operationally*, a smart compiler shares (call-by-need). The programmer reasons about *what* the answer is; the compiler decides *how* to compute it.

**Strict vs lazy:**

- **Strict** (ML, OCaml, F#, Lisp by default) — arguments evaluated before function call. Predictable space; awkward with infinite data.
- **Lazy** (Haskell, Miranda) — arguments evaluated only when needed; results memoised. Allows infinite structures (`[1..]`); exposes *space leak* hazards.

**Strict evaluation is closer to mathematics** in the sense that termination of `f a b` requires `a` and `b` to be defined. Lazy evaluation is closer to the *denotational* picture: every well-defined expression has a meaning even if its sub-expressions diverge.

## Pure-Functional vs Impure

A *pure* function:

1. Returns the same output for the same input.
2. Has no observable side effects (no I/O, no mutation visible outside the function).

Most "functional" languages (ML, OCaml, Lisp, Scheme, Clojure) are *impure*: they support pure programming but allow side effects via dedicated constructs (`ref`, `set!`, `print`). Haskell is *pure*: every function is pure; side effects are *values*, encoded as monadic terms.

**The IO monad in Haskell:**

```haskell
main :: IO ()
main = do
  name <- getLine             -- :: IO String
  putStrLn ("Hello, " ++ name) -- :: IO ()
```

`main` is a *value of type `IO ()`* — a *recipe* describing the I/O actions to perform. The runtime evaluates this recipe; the *language* itself is purely functional. From inside the language, an `IO` action is just data; from outside, the runtime interprets it.

**The "world threading" interpretation** — informally, `IO a` ≅ `World → (a, World)`. Each I/O action transforms the world; sequencing actions threads the world through:

```
return    : a → IO a
return x  ≅ λw. (x, w)

(>>=)     : IO a → (a → IO b) → IO b
m >>= k   ≅ λw. let (x, w') = m w in k x w'
```

The world cannot be inspected, copied, or duplicated, so this *linear* threading is enforced abstractly. (Linear types make this exact, e.g. in Linear Haskell or Idris 2.)

**Why purity** — the type `Int → Int` *means* "function from Int to Int" — not "function that *might* print to stdout, mutate global state, or launch missiles". Reasoning is local. Concurrency is safe by construction (no data races on immutable data). Optimisation is unrestricted.

**The cost of purity** — performance-critical code that *needs* mutation (in-place updates, hash tables) must be carefully encoded via monads (`ST`, `IORef`, `MVar`, `STM`) or arrays. Haskell's `Data.Vector.Mutable` and the `ST` monad provide *locally* impure code with a *globally* pure interface.

## Monads

A **monad** is a *type constructor* `m` (e.g. `Maybe`, `List`, `IO`) plus two operations:

```
return : a → m a
(>>=)  : m a → (a → m b) → m b      -- "bind"
```

satisfying three laws.

**The three monad laws:**

```
-- Left identity
return x >>= f         ≡  f x

-- Right identity
m >>= return           ≡  m

-- Associativity
(m >>= f) >>= g        ≡  m >>= (λx. f x >>= g)
```

**Categorical roots — Kleisli triples.** In category theory, a monad on a category `C` is a triple `(T, η, μ)` where:

- `T : C → C` is an endofunctor.
- `η : Id ⇒ T` is the *unit* natural transformation.
- `μ : T² ⇒ T` is the *multiplication* natural transformation.

with the coherence laws:

```
μ ∘ T(μ) = μ ∘ μ_T            -- associativity
μ ∘ T(η) = id_T = μ ∘ η_T     -- unit
```

The Haskell `return` is `η`; `μ` is `join : m (m a) → m a`; bind is `m >>= f = join (fmap f m)`. The *Kleisli category* `C_T` has the same objects as `C`, but a morphism from `A` to `B` in `C_T` is a morphism `A → T B` in `C`. Bind is *Kleisli composition*.

Equivalently, a Kleisli triple on `C` is a function on objects `T`, an arrow `η_A : A → T A` for each object, and a *lifting* `f* : T A → T B` for each arrow `f : A → T B`. Eilenberg-Moore showed the two formulations are equivalent.

**Why monads matter for programming.** Eugenio Moggi's *Notions of computation and monads* (1991) showed that diverse computational features — partiality, exceptions, state, continuations, non-determinism — share the same algebraic structure. Wadler popularised the construction in functional languages. A *monadic effect* is composable: code abstract over `m a` works for any monad.

**Do-notation:**

```haskell
do x <- m
   y <- f x
   g x y
```

desugars to:

```haskell
m >>= λx. f x >>= λy. g x y
```

Sequencing of monadic actions reads like an imperative program; the type system tracks that they are *values* of monadic type, not statements.

## Common Monads

A tour of the most-used monads in functional programming, each capturing a different *notion of computation*.

**Maybe (Option) — partiality.**

```haskell
data Maybe a = Nothing | Just a

return x          = Just x
Just x  >>= f     = f x
Nothing >>= _     = Nothing
```

Sequencing fails as soon as any step is `Nothing`. The program "computes a value, or doesn't".

**Either — failure with reason.**

```haskell
data Either e a = Left e | Right a

return x           = Right x
Right x  >>= f     = f x
Left e   >>= _     = Left e
```

`Left` carries an *error value*; the rest of the chain short-circuits.

**List — non-determinism.**

```haskell
return x           = [x]
xs >>= f           = concatMap f xs    -- == concat (map f xs)
```

A computation returns *zero or more* answers; bind explores all combinations. Pythagorean triples in three lines:

```haskell
triples = do a <- [1..20]
             b <- [a..20]
             c <- [b..20]
             guard (a*a + b*b == c*c)
             return (a, b, c)
```

**State — mutable state, purely.**

```haskell
newtype State s a = State { runState :: s -> (a, s) }

return x         = State $ \s -> (x, s)
m >>= f          = State $ \s -> let (x, s') = runState m s
                                 in runState (f x) s'

get   = State $ \s -> (s, s)
put s = State $ \_ -> ((), s)
modify f = State $ \s -> ((), f s)
```

Pure function from input state to (result, output state). Sequencing threads state transparently. *Monad transformers* (`StateT`) layer state on other effects.

**IO — side effects.** As above; conceptually `World → (a, World)`.

**Reader — dependency injection.**

```haskell
newtype Reader r a = Reader { runReader :: r -> a }

return x  = Reader $ \_ -> x
m >>= f   = Reader $ \r -> runReader (f (runReader m r)) r

ask       = Reader id
local f m = Reader $ \r -> runReader m (f r)
```

A computation that has *read-only* access to an environment `r`. The `Reader r a ≅ r → a`, but giving it a monadic structure lets it compose in `do`-notation with other effects.

**Writer — logging / accumulation.**

```haskell
newtype Writer w a = Writer { runWriter :: (a, w) }
-- requires Monoid w

return x          = Writer (x, mempty)
Writer (x, w) >>= f =
  let Writer (y, w') = f x
  in Writer (y, w <> w')

tell w = Writer ((), w)
```

Each step contributes to a monoidal log. With `w = [String]`, you get accumulating trace messages.

**Continuation (Cont).**

```haskell
newtype Cont r a = Cont { runCont :: (a -> r) -> r }
return x = Cont $ \k -> k x
m >>= f  = Cont $ \k -> runCont m (\x -> runCont (f x) k)
```

Programs in CPS form. Generalises every other monad — *callCC* gives first-class continuations; useful for early exit, generators, exceptions, coroutines.

## Applicative Functors

Introduced by McBride and Paterson (*Applicative programming with effects*, 2008), applicative functors are a *weaker* abstraction than monads:

```
class Functor f => Applicative f where
  pure  :: a -> f a
  (<*>) :: f (a -> b) -> f a -> f b
```

with laws:

```
pure id  <*> v          = v                                -- identity
pure (.) <*> u <*> v <*> w = u <*> (v <*> w)               -- composition
pure f   <*> pure x     = pure (f x)                       -- homomorphism
u        <*> pure y     = pure ($ y) <*> u                 -- interchange
```

**Why weaker is sometimes better:**

- *Static structure* — in `f <*> a <*> b`, the *shape* of the effect is fixed before any value is observed. This enables *batched* evaluation, *parallel* execution, *static analysis* of the effect.
- *Parsing combinators* — Applicative-only parsers can be analysed *statically* (e.g. for emptiness, follow sets), but cannot make parsing decisions based on previously-parsed values. Monadic parsers can, at the cost of static analysis.

```haskell
liftA2 :: Applicative f => (a -> b -> c) -> f a -> f b -> f c
liftA2 f a b = f <$> a <*> b

-- safeDiv combining Maybe:
safeDiv :: Int -> Int -> Maybe Int
liftA2 (+) (safeDiv 10 2) (safeDiv 6 3)        -- Just 7
liftA2 (+) (safeDiv 10 0) (safeDiv 6 3)        -- Nothing
```

Every monad is an applicative (`<*> = ap`); not every applicative is a monad — `ZipList`, `Const`, validation applicatives that *accumulate* errors (whereas Either-monad short-circuits) are pure-applicative.

## Functors

The most basic abstraction in this hierarchy:

```
class Functor f where
  fmap :: (a -> b) -> f a -> f b
```

with laws:

```
fmap id        = id                          -- identity
fmap (g . h)   = fmap g . fmap h             -- composition
```

**Categorically** — a functor between categories `C` and `D` is a mapping of objects and arrows preserving identity and composition. Haskell's `Functor` is an *endofunctor* on the category `Hask` of types and total functions.

**Examples** — `Maybe`, `[]`, `Either e`, `IO`, `(->) r`, `Map k`, `Tree`, `Vector` — all admit `fmap`. Many libraries provide derived combinators built on `fmap`:

```
(<$)  :: a -> f b -> f a
(<&>) :: f a -> (a -> b) -> f b
void  :: f a -> f ()
```

The hierarchy `Functor → Applicative → Monad → MonadFix → MonadIO ...` is one of the cleanest cases of *progressive disclosure* in language design: you reach for the *weakest* abstraction that solves your problem, gaining the most generality.

## Logic Programming

Logic programming flips the imperative *how* into a *what*. A program is a set of *facts* and *rules*; queries trigger *proof search* using **resolution**, the inference rule of propositional and first-order logic.

**Horn clauses** — restricted clauses with *at most one* positive literal:

```
A :- B₁, B₂, ..., Bₙ.        -- A holds if all Bᵢ hold
A.                           -- fact (n = 0)
:- B₁, B₂, ..., Bₙ.          -- goal (no positive literal)
```

Restricting to Horn clauses gives a tractable fragment with a complete proof procedure (SLD-resolution).

**Prolog example:**

```prolog
parent(tom, bob).
parent(bob, ann).
parent(ann, jen).

ancestor(X, Y) :- parent(X, Y).
ancestor(X, Y) :- parent(X, Z), ancestor(Z, Y).

?- ancestor(tom, jen).         % yes
?- ancestor(X, ann).           % X = tom ; X = bob
```

**SLD resolution** (Selective Linear Definite-clause resolution) — start with the goal, repeatedly *resolve* it against clause heads, **unifying** to bind variables, until the empty goal is derived (success) or no clause matches (failure / backtrack).

**Prolog's strategy:**

- Clauses tried in *source order* (top to bottom).
- Goals selected *left to right*.
- *Depth-first* search with chronological *backtracking*.

The strategy is incomplete in general (depth-first can loop on infinite search trees with finite proofs). Pure logic programming would use *fair* search; Prolog trades completeness for predictable, stack-friendly execution.

**Beyond Prolog:**

- **Datalog** — Horn clauses without function symbols; bottom-up evaluation; strict termination guarantee. Used in static analysis (Datomic, Soufflé).
- **Mercury** — declarative logic with *modes* and *determinism* annotations; comparable performance to imperative languages.
- **λProlog** — higher-order Horn clauses; meta-programming, theorem-prover construction.
- **Curry** — fuses functional and logic programming; *narrowing* generalises pattern matching.

## Unification

Unification is the *engine* of logic programming and the *mechanism* of HM type inference. It solves the equation:

```
M = N
```

over terms with variables, by finding a *substitution* σ such that σM ≡ σN.

**Robinson's unification algorithm (1965):**

```
unify(t₁, t₂) =
  | t₁ is a variable x:
      | t₂ ≡ x:       return id
      | x occurs in t₂: fail (occurs-check)
      | otherwise:    return [x := t₂]
  | t₂ is a variable: unify(t₂, t₁)
  | t₁ = f(a₁..aₙ), t₂ = g(b₁..bₘ):
      | f ≠ g or n ≠ m: fail
      | otherwise:
          σ = id
          for i = 1..n:
            σ' = unify(σ aᵢ, σ bᵢ)
            σ = σ' ∘ σ
          return σ
```

**Most-general unifier (MGU)** — if a unifier exists, Robinson's algorithm finds the *most general* one — every other unifier is an instance. MGUs are unique up to variable renaming.

**Occur-check** — `unify(x, f(x))` should fail (no finite term satisfies `x = f(x)`). Without occur-check, the algorithm produces a *cyclic* term — useful in some settings (rational tree unification, Prolog's `=` by default omits occur-check for performance), dangerous in others (HM type inference would loop).

**Higher-order unification** — unifying terms of typed λ-calculus is *undecidable* in general. *Pattern unification* (Miller 1991) is a decidable fragment used in λProlog, Twelf, and modern type-checkers.

## Constraint Programming

Constraint programming generalises both logic and search programming: a *constraint store* accumulates relations among variables; *propagation* removes inconsistent values from variable domains; *search* explores remaining possibilities.

**Core concepts:**

- **Variable** — has a *domain* of possible values.
- **Constraint** — a relation over variables (e.g. `x + y = z`, `all-different(a, b, c)`).
- **Constraint store** — the conjunction of currently-imposed constraints.
- **Propagation** — apply consistency rules to *prune* domains.
- **Search** — when propagation cannot decide, *split* the problem (e.g. by variable assignment) and recurse.

**Levels of consistency:**

- **Node consistency** — every value in `D(x)` satisfies unary constraints on `x`.
- **Arc consistency** — for every binary constraint `c(x, y)`, every value in `D(x)` has *some* compatible value in `D(y)`.
- **Path consistency** — extends to triples; AC-3, AC-4 algorithms enforce arc consistency in `O(ed²)` and `O(ed²)` respectively.
- **k-consistency** — extends to k-tuples.
- **Global consistency** — full enumeration; intractable.

Real systems strike balances: CHIP, ECLᵢPSᵉ, Gecode, Choco, MiniZinc.

**Constraint Logic Programming (CLP)** — Prolog with a *constraint solver* attached. CLP(FD) handles finite domains; CLP(R) handles real arithmetic; CLP(B) handles booleans; CLP(Q) handles rationals.

```prolog
:- use_module(library(clpfd)).

queens(N, Qs) :-
    length(Qs, N),
    Qs ins 1..N,
    safe(Qs),
    label(Qs).

safe([]).
safe([Q|Qs]) :- noattack(Q, Qs, 1), safe(Qs).
```

The N-queens problem in 5 lines, with built-in propagation and backtracking.

## Object-Oriented Programming

OOP traces to *Simula 67* (Dahl and Nygaard, Norwegian Computing Center, 1967), the first language with *classes*, *objects*, *inheritance*, and *virtual procedures*. *Smalltalk-72* (Kay, Goldberg, Ingalls at Xerox PARC) made everything an object, message-passing the universal mechanism.

**The four pillars (post-hoc rationalisation):**

1. **Encapsulation** — bundle data with operations; hide representation behind an interface.
2. **Inheritance** — subclasses extend / specialise superclasses; code reuse via class hierarchies.
3. **Polymorphism** — uniform interface, varying implementation; enables substitution.
4. **Abstraction** — the type signature is the contract; clients depend on it, not on the implementation.

**Encapsulation as an algebraic concept** — an object is an *abstract data type* (ADT). Its public methods are the *signature* of an algebra; its private state is the *carrier set*; equational laws (often implicit) are the *equations*. Two implementations are *behaviourally equivalent* if no client can distinguish them — exactly the data-refinement notion of programming-language theory.

**Inheritance vs subtyping** — these are *separate* concepts. *Inheritance* is a code-reuse mechanism (a subclass copies-and-extends the superclass's table of methods). *Subtyping* is a substitution principle (an `S` may stand wherever a `T` is expected). Liskov and Wing (1994) made the difference precise.

**Dynamic dispatch** — when calling `obj.method()`, the implementation chosen depends on `obj`'s *runtime* class. This is implemented via *vtables* (a per-class table of method pointers); `obj.method()` indexes into the vtable. Dynamic dispatch implements *single dispatch*: only the *receiver* (`obj`) influences resolution.

**Object models in semantics:**

- **Records of functions** — Cardelli's calculus (1984): an object is a labelled tuple of methods, each closing over a self-reference; subtyping is structural.
- **Classes as templates** — Java/C++: a class is a template specifying field layout and method tables; subtyping is nominal.
- **Prototypes** — JavaScript / Self: no classes; objects directly delegate to other objects.

The *object calculus* (Abadi-Cardelli, *A Theory of Objects*, 1996) is the canonical formal foundation: a tiny calculus where objects, methods, method override, and inheritance are all primitives. It is *not* reducible to STLC + records — the recursion through `self` is essential.

## The Liskov Substitution Principle

Barbara Liskov's 1987 invited keynote and the 1994 paper with Jeannette Wing formalised what subtyping must mean to support code reuse:

> Let `φ(x)` be a property provable about objects `x` of type `T`. Then `φ(y)` should be true for objects `y` of type `S` where `S` is a subtype of `T`.

In design-pattern parlance: *subtypes must obey the contracts of their supertypes*.

**Concrete obligations on a subtype `S` of `T`:**

- **Preconditions cannot be strengthened** — if `T.method` accepts inputs satisfying `P`, then `S.method` must accept all such inputs (it may accept *more*).
- **Postconditions cannot be weakened** — if `T.method` guarantees `Q`, then `S.method` must guarantee `Q` (it may guarantee *more*).
- **Invariants must be preserved** — class invariants of `T` continue to hold for `S`.
- **History constraint** — the *sequence* of states `S` can transit through is a subset of those `T` can transit through. (E.g. an immutable subtype of a mutable type would violate this.)
- **No new exceptions** that `T`'s clients are unprepared for.

**Variance:**

- *Covariant* return types — `S.method` may return a *more specific* type than `T.method`. (Java since 1.5; C++ via override.) Permitted because clients expecting `T`'s result type accept any subtype.
- *Contravariant* parameter types — `S.method` may accept a *more general* parameter type than `T.method`. Most languages disallow this on syntactic grounds; functional-style records-of-closures do allow it.
- *Invariant* generic parameters — Java's `List<Number>` is *not* a subtype of `List<Object>`, because methods like `add` use the parameter contravariantly while `get` uses it covariantly. Use-site variance (`? extends T`, `? super T`) recovers expressiveness.

The classical *square-rectangle* example illustrates the trap: `Square extends Rectangle` may *seem* sound, but `setWidth(3)` on a square implicitly changes the height, breaking a client who expected `setWidth` to leave the height alone.

## Multimethods (CLOS)

In Common Lisp's CLOS, Dylan, Julia, Clojure (via `defmulti`), and Cecil, dispatch is on *all* arguments, not just the receiver — *multiple dispatch* / *multimethods*.

```lisp
(defmethod collide ((a Asteroid) (b Ship)) ...)
(defmethod collide ((a Ship)     (b Asteroid)) ...)
(defmethod collide ((a Asteroid) (b Asteroid)) ...)
(defmethod collide ((a Ship)     (b Ship)) ...)
```

Calling `(collide x y)` selects the most specific applicable method according to the *class precedence list* (linearised multiple inheritance, e.g. via the C3 algorithm).

**Why multimethods help** — binary operations (collision, drawing, conversion, comparison) are inherently *symmetric*; forcing them onto a single receiver is awkward (the *visitor pattern* and *double dispatch* are workarounds in single-dispatch languages).

```julia
# Julia: dispatch on all argument types
+(a::Int, b::Int) = ...
+(a::Float, b::Int) = ...
+(a::Vector, b::Vector) = ...
+(a::Matrix, b::Vector) = ...
```

Julia's entire numerics tower is structured around multimethods.

**Cost** — *method resolution* must inspect *all* argument types; static analysis is harder. Type *generalisation* (multimethods as *generic functions* in their own right) gives a clean denotational semantics: a generic function is a *family* of functions indexed by argument types, with a precedence order picking the active member at call time.

## Aspect-Oriented Programming

Gregor Kiczales's *aspect-oriented programming* (1997, AspectJ 2001) targets *cross-cutting concerns* — logging, security checks, transactions, persistence — that resist *modularisation* in pure OOP because they pervade many classes.

**Vocabulary:**

- **Joinpoint** — a well-defined point in program execution (method call, field access, exception handler).
- **Pointcut** — a *predicate* selecting a set of joinpoints (e.g. "all calls to setters in package `model`").
- **Advice** — code that runs *at* a joinpoint matching a pointcut. Variants:
  - **Before** — runs before the joinpoint.
  - **After** — runs after, regardless of outcome.
  - **After-returning** — runs after a normal return.
  - **After-throwing** — runs after an exception.
  - **Around** — wraps the joinpoint; may proceed, replace, or skip.
- **Aspect** — module bundling related pointcuts and advice.
- **Weaving** — the process of combining base code with aspects, either *statically* (compile-time, AspectJ) or *dynamically* (runtime, Spring AOP).

```java
@Aspect
public class LoggingAspect {
  @Pointcut("execution(* com.example.service..*(..))")
  public void serviceLayer() {}

  @Before("serviceLayer()")
  public void logEntry(JoinPoint jp) {
    System.out.println("Entering " + jp.getSignature());
  }
}
```

**Theoretical perspectives** — joinpoint shadows can be modelled as a *labelled* operational semantics where each transition is tagged with the joinpoint it represents; pointcuts are predicates over the label-stream; advice composes via the labelled trace. Less polished than monad theory, but rigorous foundations exist (Wand, Kiczales, Dutchyn 2004).

**Critique** — *implicit* control flow makes program comprehension harder; aspects can unintentionally clobber invariants. The pendulum has swung partly back: Spring's AOP is heavily used; AspectJ less so. Algebraic effects (below) cover much of the same design space with cleaner reasoning.

## Concurrent Programming Models

Concurrency is a separate axis from sequential programming: how do multiple threads of execution share state, communicate, synchronise? Three influential calculi:

**CSP — Communicating Sequential Processes (Hoare 1978)** — processes communicate via *synchronous* channels:

```
P, Q ::= STOP                       -- deadlocked process
       | a -> P                     -- engage event a, then P
       | P □ Q                      -- external choice
       | P ⊓ Q                      -- internal choice
       | P || Q                     -- parallel
       | P[a/b]                     -- renaming
```

CSP's denotational semantics uses *traces*, *failures*, and *divergences* to give precise refinement notions. The FDR refinement checker tools the model. Go's goroutines + channels are *inspired by* CSP (though without the algebraic tool support).

**CCS — Calculus of Communicating Systems (Milner 1980)** — processes synchronise on *named* actions and their *co-actions*; `bisimulation` is the equivalence:

```
P ~ Q  iff  for every transition P -a-> P', exists Q -a-> Q' with P' ~ Q'
                          and vice versa
```

Bisimulation is the *finest* equivalence respecting the labelled-transition structure — the right notion of "same observable behaviour" for processes.

**Actor model (Hewitt 1973)** — each *actor* has a *mailbox*; on receiving a message it may:

1. Send messages to other actors.
2. Create new actors.
3. Choose its behaviour for the next message.

No shared state; messages are immutable; each actor is sequential internally. Erlang, Akka, Pony, Elixir all build on this model.

## The π-Calculus

Milner, Parrow, and Walker's *π-calculus* (1992) extends CCS with *mobility*: channel names can be passed as messages. Syntax:

```
P, Q ::= 0                          -- inactivity
       | x⟨y⟩.P                     -- send y over x, then P
       | x(y).P                     -- receive on x, bind to y, then P
       | P | Q                      -- parallel composition
       | (νx) P                     -- create fresh name x in P
       | !P                         -- replication (≅ P | P | P | ...)
```

**Why mobility matters** — capabilities can be *passed*. A server can send a *fresh* private channel to a client for further interaction. The π-calculus is *the* foundation for distributed and mobile computing.

**Reduction:**

```
x⟨y⟩.P | x(z).Q   →   P | Q[z := y]
```

**Typed π-calculi** assign *channel types* tracking what messages a channel may carry; *session types* (Honda 1993, Honda-Vasconcelos-Kubo 1998) encode *protocols* as types, ensuring deadlock-free communication patterns at compile time.

```
Session type: !Int.?Bool.end           -- send Int, receive Bool, done
```

A communicating program is type-correct iff its actual interactions match its declared session type. Languages: Sing#, SILL, the *Effekt* / *MiniML* with sessions, F* with session types.

## The Actor Model

The actor model is the engineering descendant of Hewitt's 1973 paper and its operational realisation in Erlang (Armstrong, Virding, Williams at Ericsson, 1987-) and OTP.

**Properties:**

- **Encapsulated state** — each actor's state is private; mutated only by its own message-handling code.
- **Asynchronous messages** — sending does not block; the message lands in the recipient's mailbox.
- **Location transparency** — local and remote actors look the same; messages may cross machine boundaries.
- **No shared memory** — therefore no data races; concurrency safety is by construction.
- **Supervision** — a *supervisor* actor monitors children and decides what to do on failure: restart, escalate, give up. The "let it crash" philosophy embraces failure as normal.

**Erlang/OTP idioms:**

- `gen_server` — generic server with cast (async), call (sync with reply), and state.
- `gen_statem` — finite state machine.
- `supervisor` — restart strategy: one-for-one, one-for-all, rest-for-one.
- `application` — packaged unit of supervision.

```erlang
loop(State) ->
    receive
        {From, {get, Key}} ->
            From ! {ok, maps:get(Key, State, undefined)},
            loop(State);
        {set, Key, Value} ->
            loop(State#{Key => Value});
        stop ->
            ok
    end.
```

Each actor is a tail-recursive function over its state; each `receive` blocks until a matching message arrives.

**Modern actor frameworks:** Akka (Scala/Java/JVM), Pony (with capability-based safety), Orleans (.NET, "virtual actors"), Microsoft Service Fabric.

## Software Transactional Memory (STM)

STM brings *database-style transactions* to in-memory data. A *transaction* is a sequence of reads and writes that appears *atomic* (all-or-nothing) and *isolated* (no other transaction sees intermediate state).

**Haskell's STM (Harris, Marlow, Peyton Jones, Herlihy 2005):**

```haskell
atomically :: STM a -> IO a
retry      :: STM a
orElse     :: STM a -> STM a -> STM a
newTVar    :: a -> STM (TVar a)
readTVar   :: TVar a -> STM a
writeTVar  :: TVar a -> a -> STM ()
```

```haskell
transfer :: TVar Int -> TVar Int -> Int -> STM ()
transfer from to amount = do
  bal <- readTVar from
  when (bal < amount) retry
  writeTVar from (bal - amount)
  modifyTVar to (+ amount)
```

`retry` blocks until any read variable changes; `orElse` composes alternatives. The system guarantees *linearisability* of transactional operations: an external observer sees them in some serial order consistent with real-time order.

**Composability** — *the* unique advantage of STM. Two locks can deadlock when composed; two transactions cannot. `atomically (transfer a b 100) >> atomically (transfer b c 50)` is two atomic operations; `atomically (transfer a b 100 >> transfer b c 50)` is *one*. Try writing the latter with locks.

**Cost** — read/write tracking, retry on conflict, no I/O inside transactions (or you'd need to undo it on rollback). Practical performance varies; STM shines under low contention, suffers under high.

**Implementations:** Haskell `Control.Concurrent.STM`, Clojure's `dosync` / `ref`, .NET's deprecated STM.NET, hardware TM on Intel TSX (controversial), Multicore OCaml's algebraic-effect-based STM.

## Reactive Programming

Conal Elliott's *Functional Reactive Programming* (FRP, 1997) modelled GUIs and animation declaratively. Two primitives:

```
Behavior a   ≅   Time → a              -- continuous time-varying value
Event a      ≅   [(Time, a)]            -- discrete sequence of timestamped values
```

**Combinators:**

```
constant   : a -> Behavior a
time       : Behavior Time
($$)       : Behavior (a -> b) -> Behavior a -> Behavior b   -- applicative

never      : Event a
once       : Time -> a -> Event a
filterE    : (a -> Bool) -> Event a -> Event a
mergeE     : Event a -> Event a -> Event a
foldp      : (a -> s -> s) -> s -> Event a -> Behavior s     -- "fold over time"
sample     : Behavior a -> Event b -> Event a                -- snapshot on each event
```

**Pure FRP** has *push-pull* implementations to be efficient: changes propagate *forward*; time queries *pull*. Engineering pure FRP is hard; many "reactive" libraries cheat.

**Approximations:**

- **RxJS / Rx.NET / RxSwift** — observable streams + operators (`map`, `filter`, `merge`, `combineLatest`, `flatMap`). Push-only; no first-class behaviour.
- **Reactor (Java)** — `Flux<T>` (zero-or-many), `Mono<T>` (zero-or-one); back-pressure protocol via reactive-streams.
- **Elm** — stripped-down FRP; the *signal* model evolved into the *model-update-view* (MVU) architecture, the model for Redux.
- **React / SolidJS / Vue** — UI-frame FRP-ish: state changes invalidate views; reconciliation chooses what to re-render.

The mathematical idea — *programs as functions over time* — survives even in approximations as a *cleaner* mental model than callback graphs.

## Dataflow Programming

In dataflow, a program is a *graph* of *nodes* connected by *edges*; values *flow* along edges; a node fires when its inputs are ready, producing outputs that feed downstream nodes.

**Synchronous dataflow** — Lustre (Caspi, Halbwachs, Pilaud, Plaice 1987) — used in safety-critical systems (airbus, nuclear plants):

```
node Counter(reset, inc : bool) returns (n : int);
let
  n = 0 -> if reset then 0 else (pre n) + (if inc then 1 else 0);
tel
```

Every variable is a *stream*; `pre n` is the previous value; `->` is "first then". Programs compile to *bounded-memory* state machines.

**Other dataflow languages:**

- **Esterel** — synchronous reactive, imperative-style.
- **SIGNAL** — relational dataflow.
- **LabVIEW** — visual dataflow for instrumentation.
- **TensorFlow 1.x** — *deferred* dataflow graphs for ML; replaced by eager mode + autograph in 2.x.
- **Apache Beam, Storm, Flink** — dataflow for stream processing.

**Theoretical roots** — Kahn networks (1974) — networks of deterministic processes connected by FIFO channels; produce the *same* output regardless of scheduling, given the network is *closed*. The Kahn principle underlies modern stream-processing semantics.

## Differentiable Programming

The newest paradigm to gain traction (~2018 onwards): a program *is* a differentiable function, and gradients are first-class. Originated in deep-learning frameworks; generalised by Yann LeCun's call to elevate "differentiable programming" alongside imperative/functional/logic.

**Automatic differentiation:**

- **Forward-mode AD** — propagate dual numbers `a + b·ε` (where `ε² = 0`); efficient when the function has few inputs and many outputs.
- **Reverse-mode AD** (a.k.a. *backpropagation*) — record a *tape* of operations; replay backwards multiplying Jacobians. Efficient when the function has many inputs and few outputs (the typical ML case).

**Frameworks:**

- **TensorFlow** — graph-mode (1.x) and eager (2.x).
- **PyTorch** — eager from day one; *autograd* engine builds dynamic tape.
- **JAX** — pure functional; `grad`, `jit`, `vmap`, `pmap` as composable transforms.
- **Zygote.jl** (Julia) — source-to-source AD; differentiates *arbitrary* Julia code, not just a sub-DSL.

**Mathematical foundation** — *categories of differentiable functions* (Cockett, Cruttwell, Gallagher 2014) generalise smooth maps. *Categorical-AD* (Elliott 2018, *The simple essence of automatic differentiation*) gives a clean compositional account.

The deep claim: machine learning, control, optimisation, and graphics are all *differentiable programming*; the paradigm shift is treating differentiation as a *language feature*, not a library service.

## Quantum Programming

A quantum program transforms *qubits* — vectors in a complex Hilbert space:

```
|ψ⟩ = α|0⟩ + β|1⟩          where α, β ∈ ℂ, |α|² + |β|² = 1
```

A multi-qubit state is the *tensor product* of individual qubits — except for *entangled* states, which cannot be written as a tensor product.

**Quantum gates** are *unitary* matrices acting on the Hilbert space:

```
Hadamard:   H = (1/√2) [[1,  1],
                        [1, -1]]

Pauli-X:    X = [[0, 1],
                 [1, 0]]                -- the "NOT" gate

CNOT (2-qubit):  flips target if control is |1⟩
```

A *quantum circuit* is a sequence of gates; *measurement* collapses the state to a classical bit (probabilistically, with probability `|α|²` of `0` and `|β|²` of `1`).

**Key principles:**

- **Superposition** — a qubit can be in a *combination* of `|0⟩` and `|1⟩`; an n-qubit register lives in a `2ⁿ`-dimensional Hilbert space.
- **Entanglement** — joint states cannot be factored; measuring one qubit instantly determines the other (Bell-state correlations).
- **No-cloning theorem** — there is no quantum operation `U` with `U(|ψ⟩|0⟩) = |ψ⟩|ψ⟩` for arbitrary `|ψ⟩`. Hence quantum information is fundamentally different: it can be *teleported* but not *copied*.
- **Reversibility** — every quantum gate is unitary, hence reversible. Classical computation can be embedded reversibly using ancilla bits.

**Quantum programming languages:**

- **Q#** (Microsoft) — operations on qubits with classical control flow.
- **Qiskit** (IBM) — Python-embedded; circuits + algorithms + simulators.
- **Cirq** (Google) — Python; targets NISQ-era hardware.
- **Quipper** — Haskell-embedded DSL; the original "scalable" quantum language.
- **Silq** (Bichsel et al. 2020) — *automatic uncomputation* of garbage qubits — a high-level quantum language.

**Type-theoretic foundations** — *linear* type systems are natural for quantum: no-cloning means qubit references must be used *exactly once*. *Quantum lambda calculi* (van Tonder 2003, Selinger-Valiron 2006) give operational semantics for higher-order quantum programs.

## Type Theory & Programming Languages

Benjamin Pierce's *Types and Programming Languages* (TAPL, 2002) is the canonical textbook. Its taxonomy of type-theoretic features:

**Subtyping:**

- *Width subtyping* — a record type `{l₁:τ₁, ..., lₙ:τₙ, lₙ₊₁:σ}` is a subtype of `{l₁:τ₁, ..., lₙ:τₙ}` (more fields).
- *Depth subtyping* — `{l:S}` ≤ `{l:T}` if `S` ≤ `T` (covariant in field types). Combined with mutability this is unsound; needs invariance for mutable fields.
- *Nominal* (Java, C#) vs *structural* (TypeScript, OCaml object types) — does the *name* of the type matter, or only its *structure*?

**Polymorphism:**

- *Parametric* — same code, any type. `id : ∀α. α → α`.
- *Ad-hoc / overloading* — different code, named the same. Haskell's type classes, Rust's traits.
- *Subtype polymorphism* — code accepting `T` works for any subtype of `T`.

**Row polymorphism** — generalises record subtyping: `{l:τ | r}` is a record with field `l:τ` and *some other fields* (*row variable* `r`). Functions can require certain fields without forbidding others.

```ocaml
let get_x : {x:int; ..ρ} -> int = fun r -> r.x
```

works on any record with an `x:int` field. OCaml's polymorphic methods, PureScript's row types, Elm's extensible records all use row polymorphism.

**Refinement types** — types refined by *predicates*:

```
{x : Int | x > 0}              -- positive integers
{xs : List Int | sorted xs}    -- sorted lists
```

LiquidHaskell, Refined Rust, F* all support refinement types — the type-checker discharges side-conditions to an SMT solver.

**Linear and substructural types** — *linear* values must be used *exactly once*; *affine* values *at most once*; *relevant* values *at least once*. The basis for Rust's borrow checker, Linear Haskell, Mezzo, Pony's reference capabilities. Mathematically they correspond to *linear logic* (Girard 1987) — a refinement of intuitionistic logic where "use" is metered.

**Session types** as discussed in the π-calculus section.

**Gradual typing** (Siek-Taha 2006) — interleaves typed and untyped code with runtime checks at the boundaries. Used in TypeScript, Typed Racket, mypy.

## Effect Systems

A *type-and-effect system* (Lucassen-Gifford 1988) extends type judgements to track *what side effects* a computation can perform, e.g. `Γ ⊢ M : τ ! ε` where `ε` is an effect set.

**Algebraic effects + handlers** (Plotkin-Power 2003, Plotkin-Pretnar 2009):

- An *effect signature* is a set of operations with input and output types: `Get : Unit → Int`, `Put : Int → Unit`.
- An *effectful program* invokes operations: `do x ← Get(); Put(x + 1)`.
- A *handler* gives a semantic interpretation, mapping operations to ordinary continuations. The same effectful program runs with different handlers (state-passing, logging, exception, concurrency).

**Languages:**

- **Eff** (Bauer-Pretnar) — research language; the canonical algebraic-effect calculus.
- **Koka** (Leijen, Microsoft) — algebraic effects with row-typed effect rows.
- **Multicore OCaml** — runtime effects (lightweight threads, async/await, generators) built atop the same machinery.
- **Frank** — *abilities* + handlers, more conservative than Eff.

```koka
fun safediv(x : int, y : int) : exn int
  if y == 0 then throw("div by zero") else x / y
```

`exn` is an effect, not a type — the function's type is `(int, int) → exn int`.

**The "monads as design pattern, effects as language feature" position** — algebraic effects achieve much of what monad transformers achieve, without the monad-stacking-and-lifting boilerplate. They compose freely; handlers are first-class. Major work by Bauer, Pretnar, Plotkin, Power, Leijen. The bridge to monads: *every* algebraic effect can be modelled by a *free monad* over its signature, with handlers as *fold*s over the free monad structure.

## Category Theory in CS

Category theory provides the *unifying language* for many of the structures above.

**Definitions:**

- A **category** `C` consists of *objects* and *morphisms* (arrows) between objects, with identity and associative composition.
- A **functor** `F : C → D` maps objects and arrows, preserving identity and composition.
- A **natural transformation** `η : F ⇒ G` is an arrow `η_A : F A → G A` for each object `A`, naturally — i.e. commuting with the action of arrows.

**Cartesian-closed categories (CCCs)** — categories with:

- *Terminal object* `1` (think: unit type).
- *Binary products* `A × B` with projections (think: pairs).
- *Exponentials* `A ⇒ B` with *evaluation* `eval : (A ⇒ B) × A → B` and *currying* — for any `f : C × A → B`, a unique `λf : C → A ⇒ B` with `eval ∘ (λf × id) = f`.

A CCC is a *model* of typed lambda calculus: types ≅ objects, terms ≅ morphisms, β-reduction matches the eval / currying laws.

**Monoidal categories** — categories with a *tensor product* `⊗` and a *unit object* `I`, satisfying associativity / unit *coherence laws*. Models of *linear* logic; basis for *string diagrams* used in quantum computing, applied category theory, and graphical programming languages.

**Limits and colimits** — universal constructions including products, coproducts (sums), pullbacks, pushouts, equalisers, coequalisers. Many programming-language constructs (sums, products, fixed points) are colimits or limits.

**Initial algebras and final coalgebras** — the categorical setting for *folds* and *unfolds* (recursive and corecursive computation):

- An *F-algebra* is `(A, α : F A → A)` for an endofunctor `F`.
- The *initial* F-algebra is the unique up to iso `(μF, in)` such that for any algebra `(A, α)` there is a unique homomorphism `(|α|) : μF → A`.
- This is *exactly* a structural fold (`foldr` for lists is the catamorphism for the list functor).

Dually, the *final coalgebra* gives *unfold* (`anamorphism`).

## The Curry-Howard-Lambek Correspondence

The deepest unification combines logic, computation, and category theory:

| Logic              | Type theory       | Category theory                     |
| ------------------ | ----------------- | ----------------------------------- |
| Proposition        | Type              | Object in CCC                       |
| Proof              | Closed term       | Morphism                            |
| `A → B`            | `A → B`           | Exponential `A ⇒ B`                 |
| `A ∧ B`            | `A × B`           | Product `A × B`                     |
| `A ∨ B`            | `A + B`           | Coproduct `A + B`                   |
| ⊤ (true)           | unit `()`         | Terminal object `1`                 |
| ⊥ (false)          | empty type        | Initial object `0`                  |
| Modus ponens       | Function app      | Composition with `eval`             |
| Implication intro  | λ-abstraction     | Currying                            |
| Cut elimination    | β-reduction       | Equational reasoning                |

This is the *Curry-Howard-Lambek correspondence*: typed λ-calculi correspond to constructive logics correspond to Cartesian-closed categories. Lambek (1972) showed the categorical side; Howard's manuscript circulated 1969 and was published 1980; Curry's combinatory-logic version was 1934.

**Practical consequence** — categorical foundations underwrite *every* sound program transformation:

- *Refactoring* by η-expansion and β-reduction is *literally* the equational theory of CCCs.
- *Optimisation* via category-theoretic laws (foldr fusion, map-fusion) holds *for all* concrete instances.
- *Compilers* like *Compiling to Categories* (Elliott 2017) literally compile lambda terms to CCC operations — a category being a different *target language* (CUDA, Verilog, the autodiff category, etc.).

The correspondence is more than a curiosity. It is the *reason* the lambda-calculus-with-types is the right level of abstraction for programming-language theory: it sits at the unique fixed point of three apparently distinct disciplines.

The whole picture in one diagram:

```
        Logic                Type theory             Category theory
                                                              
  Constructive logic   ⟷    Typed λ-calculus    ⟷    Cartesian-closed
  Intuitionistic              dependent types          categories
  Linear logic                linear types             monoidal cats
  Modal logics                effect types             monads
  Classical (CPS)             continuations            *-autonomous cats

     Curry & Howard         (1969-1980)         Lambek (1972)
                  +─────────── = ──────────+
                              C-H-L
```

The deeper you go in any of the three columns, the deeper you go in the others. Programming-paradigm theory, in the end, is the study of this triple equivalence — and the design choices each language makes about which corner it inhabits.

## References

- Church, A. (1936). *An unsolvable problem of elementary number theory*. American Journal of Mathematics 58(2), 345–363.
- Church, A. (1940). *A formulation of the simple theory of types*. Journal of Symbolic Logic 5(2), 56–68.
- Curry, H. B. (1934). *Functionality in combinatory logic*. Proc. NAS USA 20, 584–590.
- Howard, W. A. (1980). *The formulae-as-types notion of construction*. In *To H.B. Curry: Essays on Combinatory Logic*, pp. 479–490. Academic Press. (Manuscript circulated 1969.)
- Hindley, R. (1969). *The principal type-scheme of an object in combinatory logic*. Trans. AMS 146, 29–60.
- Milner, R. (1978). *A theory of type polymorphism in programming*. JCSS 17(3), 348–375.
- Damas, L., Milner, R. (1982). *Principal type-schemes for functional programs*. POPL 82.
- Reynolds, J. C. (1974). *Towards a theory of type structure*. Programming Symposium, LNCS 19.
- Girard, J.-Y. (1972). *Interprétation fonctionnelle et élimination des coupures de l'arithmétique d'ordre supérieur*. PhD thesis, Université Paris VII.
- Wells, J. B. (1994). *Typability and type checking in System F are equivalent and undecidable*. LICS 94.
- Reynolds, J. C. (1983). *Types, abstraction and parametric polymorphism*. IFIP Congress.
- Martin-Löf, P. (1972). *An intuitionistic theory of types*. Notes; published in *Twenty-five Years of Constructive Type Theory* (1998).
- Coquand, T., Huet, G. (1988). *The calculus of constructions*. Information and Computation 76.
- Lambek, J. (1972). *Deductive systems and categories*. Lecture Notes in Mathematics 274.
- Plotkin, G. (1981). *A structural approach to operational semantics*. Aarhus tech report DAIMI FN-19. Reprinted JLAP 60–61, 2004.
- Plotkin, G. (1975). *Call-by-name, call-by-value and the λ-calculus*. TCS 1, 125–159.
- Kahn, G. (1987). *Natural semantics*. STACS.
- Felleisen, M., Hieb, R. (1992). *The revised report on the syntactic theories of sequential control and state*. TCS 103.
- Scott, D., Strachey, C. (1971). *Toward a mathematical semantics for computer languages*. PRG-6.
- Scott, D. (1982). *Domains for denotation semantics*. ICALP 82.
- Gunter, C. (1992). *Semantics of Programming Languages: Structures and Techniques*. MIT Press.
- Hoare, C. A. R. (1969). *An axiomatic basis for computer programming*. CACM 12(10), 576–580.
- Dijkstra, E. W. (1975). *Guarded commands, nondeterminacy and formal derivation of programs*. CACM 18(8), 453–457.
- Liskov, B., Wing, J. (1994). *A behavioral notion of subtyping*. ACM TOPLAS 16(6), 1811–1841.
- Cardelli, L. (1984). *A semantics of multiple inheritance*. Semantics of Data Types, LNCS 173.
- Abadi, M., Cardelli, L. (1996). *A Theory of Objects*. Springer.
- Kiczales, G. et al. (1997). *Aspect-oriented programming*. ECOOP, LNCS 1241.
- Hoare, C. A. R. (1978). *Communicating sequential processes*. CACM 21(8), 666–677.
- Hoare, C. A. R. (1985). *Communicating Sequential Processes*. Prentice Hall.
- Milner, R. (1980). *A Calculus of Communicating Systems*. Springer LNCS 92.
- Milner, R. (1989). *Communication and Concurrency*. Prentice Hall.
- Milner, R., Parrow, J., Walker, D. (1992). *A calculus of mobile processes, parts I and II*. Information and Computation 100.
- Milner, R. (1999). *Communicating and Mobile Systems: the π-calculus*. Cambridge University Press.
- Honda, K. (1993). *Types for dyadic interaction*. CONCUR.
- Honda, K., Vasconcelos, V. T., Kubo, M. (1998). *Language primitives and type discipline for structured communication-based programming*. ESOP.
- Hewitt, C., Bishop, P., Steiger, R. (1973). *A universal modular ACTOR formalism for artificial intelligence*. IJCAI.
- Armstrong, J. (2003). *Making reliable distributed systems in the presence of software errors*. PhD thesis, KTH.
- Harris, T., Marlow, S., Peyton Jones, S., Herlihy, M. (2005). *Composable memory transactions*. PPoPP.
- Elliott, C., Hudak, P. (1997). *Functional reactive animation*. ICFP.
- Elliott, C. (2009). *Push-pull functional reactive programming*. Haskell Symposium.
- Caspi, P., Halbwachs, N., Pilaud, D., Plaice, J. (1987). *LUSTRE: a declarative language for programming synchronous systems*. POPL.
- Kahn, G. (1974). *The semantics of a simple language for parallel programming*. IFIP Congress.
- Wadler, P. (1992). *The essence of functional programming*. POPL.
- Moggi, E. (1991). *Notions of computation and monads*. Information and Computation 93(1), 55–92.
- McBride, C., Paterson, R. (2008). *Applicative programming with effects*. JFP 18(1).
- Mac Lane, S. (1971). *Categories for the Working Mathematician*. Springer.
- Pierce, B. (1991). *Basic Category Theory for Computer Scientists*. MIT Press.
- Pierce, B. (2002). *Types and Programming Languages*. MIT Press.
- Pierce, B. (ed.) (2005). *Advanced Topics in Types and Programming Languages*. MIT Press.
- Wadler, P. (2015). *Propositions as types*. CACM 58(12), 75–84.
- Bauer, A., Pretnar, M. (2015). *Programming with algebraic effects and handlers*. JLAMP 84(1).
- Plotkin, G., Power, J. (2003). *Algebraic operations and generic effects*. Applied Categorical Structures 11.
- Plotkin, G., Pretnar, M. (2009). *Handlers of algebraic effects*. ESOP.
- Leijen, D. (2017). *Type directed compilation of row-typed algebraic effects*. POPL.
- Robinson, J. A. (1965). *A machine-oriented logic based on the resolution principle*. JACM 12(1).
- Lloyd, J. (1987). *Foundations of Logic Programming*. Springer.
- Apt, K. (2003). *Principles of Constraint Programming*. Cambridge University Press.
- Rossi, F., van Beek, P., Walsh, T. (2006). *Handbook of Constraint Programming*. Elsevier.
- Bichsel, B., Baader, M., Gehr, T., Vechev, M. (2020). *Silq: a high-level quantum language with safe uncomputation and intuitive semantics*. PLDI.
- Selinger, P., Valiron, B. (2006). *A lambda calculus for quantum computation with classical control*. MSCS 16(3).
- Nielsen, M., Chuang, I. (2010). *Quantum Computation and Quantum Information*. Cambridge University Press, 10th anniversary edition.
- Elliott, C. (2017). *Compiling to categories*. ICFP.
- Elliott, C. (2018). *The simple essence of automatic differentiation*. ICFP.
- Cockett, R., Cruttwell, G., Gallagher, J. (2014). *Differential restriction categories*. TAC 25.
- Girard, J.-Y. (1987). *Linear logic*. TCS 50.
- Wadler, P. (1990). *Linear types can change the world!*. Programming Concepts and Methods.
- Siek, J., Taha, W. (2006). *Gradual typing for functional languages*. Scheme Workshop.
- Vytiniotis, D., Peyton Jones, S., Schrijvers, T., Sulzmann, M. (2011). *OutsideIn(X) modular type inference with local assumptions*. JFP.
- Tait, W. W. (1967). *Intensional interpretations of functionals of finite type I*. Journal of Symbolic Logic 32(2).
- Coq Development Team. *The Coq proof assistant reference manual*. https://coq.inria.fr/.
- Norell, U. (2007). *Towards a practical programming language based on dependent type theory*. PhD, Chalmers.
- Lean Mathlib contributors. *Mathematical components for Lean 4*. https://leanprover-community.github.io/.
- Brady, E. (2017). *Type-driven Development with Idris*. Manning.
- Harper, R. (2016). *Practical Foundations for Programming Languages*. Cambridge University Press, 2nd ed.
- Winskel, G. (1993). *The Formal Semantics of Programming Languages*. MIT Press.
- See Also: `sheets/programming/programming-paradigms.md` — the user-facing pattern catalog.
- See Also: `sheets/programming/lambda-calculus.md`, `sheets/programming/type-theory.md`, `sheets/programming/category-theory.md` (if present in the corpus) — companion sheets.
