# Type Theory (Foundations of Type Systems and Programming Language Theory)

A compact reference for type theory -- the formal study of type systems that classify terms by their computational behavior, providing the foundation for programming language design, automated theorem proving, and the deep connection between logic and computation.

## Simple Types

```
Base types and function types -- the building blocks of all type systems:

Base types (ground types):
  Bool, Nat, Int, String, Unit, ...

Function types:
  tau ::= alpha             (base type variable)
        | tau -> tau        (function type)

  ->  is right-associative:
    A -> B -> C  =  A -> (B -> C)

Examples:
  not      : Bool -> Bool
  add      : Nat -> Nat -> Nat     (curried)
  apply    : (A -> B) -> A -> B

Type environments (contexts):
  Gamma ::= . | Gamma, x : tau

  Gamma = { x : Nat, f : Nat -> Bool }

Typing judgments:
  Gamma |- M : tau    ("under context Gamma, term M has type tau")
```

## Typing Rules (Simply Typed Lambda Calculus)

```
                  x : tau in Gamma
  Variable:      -------------------
                   Gamma |- x : tau

                 Gamma, x : sigma |- M : tau
  Abstraction:  ----------------------------
                 Gamma |- (lambda x. M) : sigma -> tau

                 Gamma |- M : sigma -> tau    Gamma |- N : sigma
  Application:  -----------------------------------------------
                          Gamma |- M N : tau

Properties:
  - Type checking is decidable
  - Type inference is decidable (Algorithm W)
  - Strongly normalizing: every well-typed term terminates
  - NOT Turing-complete (a feature, not a bug, for logic)
```

## Type Inference (Hindley-Milner)

```
Hindley-Milner (HM) type system -- basis of ML, Haskell, Rust, etc.

Key features:
  - Principal types: every typeable term has a most general type
  - Let-polymorphism: polymorphism via let-bindings
  - Complete inference: no type annotations needed

Monotypes vs. polytypes:
  tau   ::= alpha | tau -> tau | C tau_1 ... tau_n    (monotype)
  sigma ::= tau | forall alpha. sigma                 (polytype / type scheme)

Algorithm W (Damas-Milner, 1982):
  1. Assign fresh type variables to all subterms
  2. Generate constraints from typing rules
  3. Solve constraints via unification
  4. Generalize let-bound variables: forall unconstrained vars

Unification:
  unify(alpha, tau)         = [alpha := tau]  if alpha not in FV(tau)
  unify(tau1 -> tau2, tau3 -> tau4)
                            = unify(tau2[S], tau4[S]) . S
                              where S = unify(tau1, tau3)
  unify(C, C)              = identity substitution

Occurs check:
  unify(alpha, tau) fails if alpha in FV(tau)
  Prevents infinite types: alpha = alpha -> alpha
```

## Polymorphism

```
Three forms of polymorphism:

1. Parametric polymorphism (generics):
   Works uniformly for all types.
     id : forall a. a -> a
     id = lambda x. x

     map : forall a b. (a -> b) -> List a -> List b

   Parametricity theorem (Wadler): type alone constrains behavior.
     Any f : forall a. [a] -> [a] must be a permutation/selection.

2. Ad-hoc polymorphism (overloading):
   Different behavior per type.
     Type classes (Haskell):     class Eq a where (==) : a -> a -> Bool
     Traits (Rust):              trait Display { fn fmt(&self, ...) }
     Interfaces (Go, Java):     interface Comparable<T> { ... }

   Resolved at compile time via dictionary passing or monomorphization.

3. Subtype polymorphism:
   If S <: T then any term of type S can be used where T is expected.

     Subsumption rule:
                    Gamma |- M : S    S <: T
                   ---------------------------
                         Gamma |- M : T

     Function subtyping (contravariant in argument, covariant in result):
       S1 <: T1    T2 <: S2
       ---------------------
       (T1 -> T2) <: (S1 -> S2)

   Variance:
     Covariant:     List<Cat> <: List<Animal>       (read-only)
     Contravariant: Sink<Animal> <: Sink<Cat>       (write-only)
     Invariant:     MutableList<Cat> unrelated to MutableList<Animal>
```

## System F (Polymorphic Lambda Calculus)

```
System F (Girard 1972 / Reynolds 1974) -- second-order typed lambda calculus:

Syntax:
  Types:   tau ::= alpha | tau -> tau | forall alpha. tau
  Terms:   M   ::= x | lambda x:tau. M | M N
                  | Lambda alpha. M | M [tau]

  Lambda alpha. M       type abstraction (create polymorphic value)
  M [tau]               type application (instantiate)

Examples:
  id    = Lambda a. lambda x:a. x           : forall a. a -> a
  id[Nat] 5                                  : Nat

  Church booleans in System F:
    Bool  = forall a. a -> a -> a
    true  = Lambda a. lambda t:a. lambda f:a. t
    false = Lambda a. lambda t:a. lambda f:a. f

  Church naturals in System F:
    Nat   = forall a. (a -> a) -> a -> a
    zero  = Lambda a. lambda s:(a->a). lambda z:a. z
    succ  = lambda n:Nat. Lambda a. lambda s:(a->a). lambda z:a. s (n [a] s z)

Properties:
  - More expressive than HM (first-class polymorphism)
  - Type inference is UNDECIDABLE (Wells, 1999)
  - Strongly normalizing
  - Impredicative: forall a. a can be instantiated with forall a. a
```

## Dependent Types

```
Types that depend on values -- the most expressive type systems:

Pi types (dependent function types):
  Pi (x : A). B(x)     or     (x : A) -> B(x)

  When B does not depend on x, Pi x:A. B  =  A -> B (ordinary function)

  Example: a function returning a value whose TYPE depends on input
    printf : (fmt : String) -> Args(fmt) -> String

Sigma types (dependent pair types):
  Sigma (x : A). B(x)    or    (x : A) * B(x)

  A pair (a, b) where b : B(a) -- second component's type depends on first.

  When B does not depend on x, Sigma x:A. B  =  A * B (ordinary pair)

  Example: length-indexed vectors
    Vec : Type -> Nat -> Type
    nil  : Vec A 0
    cons : A -> Vec A n -> Vec A (n+1)

    head : Vec A (n+1) -> A        (cannot call on empty vector!)
    append : Vec A m -> Vec A n -> Vec A (m+n)

Languages with dependent types:
  Full:        Coq, Agda, Lean, Idris 2
  Partial:     Haskell (GADTs, type families), Rust (const generics)
```

## Curry-Howard Correspondence

```
The deep isomorphism between logic and type theory:

  Logic                    Type Theory
  -----------------------------------------------
  Proposition              Type
  Proof                    Program (term)
  Implication (A => B)     Function type (A -> B)
  Conjunction (A /\ B)     Product type (A * B)
  Disjunction (A \/ B)     Sum type (A + B)
  Truth (True)             Unit type ()
  Falsity (False)          Empty type (Void)
  Universal (forall x. P)  Pi type (Pi x:A. B)
  Existential (exists x)   Sigma type (Sigma x:A. B)
  Negation (not A)         A -> Void

  Modus ponens             Function application
  Assumption               Variable
  Lambda abstraction       Implication introduction
  Case analysis            Disjunction elimination

Proof = Program correspondence:
  To prove A -> B: write a function from A to B.
  To prove A /\ B: construct a pair (a, b).
  To prove A \/ B: inject into Left a or Right b.
  To prove False: impossible -- Void has no constructor.

  "A proof of a theorem IS a program.
   The type of a program IS a theorem."
     -- Curry, Howard, Wadler
```

## Algebraic Data Types

```
Sum types (tagged unions / coproducts):
  data Bool    = True | False                    -- 2 values
  data Maybe a = Nothing | Just a                -- 1 + a
  data Either a b = Left a | Right b             -- a + b
  data List a  = Nil | Cons a (List a)           -- mu X. 1 + a * X

  |Either a b| = |a| + |b|  (cardinality is sum)

Product types (records / tuples):
  data Pair a b = MkPair a b                     -- a * b
  data Triple a b c = MkTriple a b c             -- a * b * c

  |Pair a b| = |a| * |b|  (cardinality is product)

Why "algebraic":
  Types form a semiring under + (sum) and * (product).
    0 = Void        1 = ()
    a + 0 = a       a * 1 = a
    a + b = b + a   a * b = b * a
    a * (b + c) = a*b + a*c   (distributive)

  Exponential:  b^a = a -> b     (|a -> b| = |b|^|a|)
```

## Recursive Types

```
Types defined in terms of themselves:

Iso-recursive (explicit fold/unfold):
  mu X. F(X)

  fold   : F(mu X. F(X)) -> mu X. F(X)
  unfold : mu X. F(X) -> F(mu X. F(X))

  List a = mu X. 1 + a * X
  Tree a = mu X. a + X * X

Equi-recursive (implicit, types equal to their unfoldings):
  List a  =  1 + a * List a     (silently equated)
  Used in: OCaml (with -rectypes), most dynamically typed languages

Fixed-point of type operators:
  Fix f = f (Fix f)
  newtype Fix f = Fix { unFix :: f (Fix f) }
```

## Linear Types

```
Linear types: every value must be used exactly once.

Based on Girard's linear logic (1987):
  Multiplicative connectives:  A (*) B (tensor),  A -o B (linear implication)
  Additive connectives:        A & B (with),      A (+) B (plus)

Usage modalities:
  Linear (1):    use exactly once
  Affine:        use at most once   (Rust ownership!)
  Relevant:      use at least once
  Unrestricted:  use any number of times (standard types)

Rust ownership as affine types:
  let s = String::from("hello");     // s owns the value
  let t = s;                         // move: s consumed, t owns it
  // println!("{}", s);              // ERROR: s already moved (used)

  &T   -- shared borrow (unrestricted reads)
  &mut T -- exclusive borrow (linear write access)

  Affine = linear + allowed to drop (Rust Drop trait)

Applications:
  - Resource management (files, sockets, locks)
  - Session types (protocol compliance)
  - Memory safety without garbage collection (Rust)
  - Quantum computing (no-cloning theorem)
```

## Gradual Typing

```
Blend static and dynamic typing within one language:

  The dynamic type ? (or Any / Dynamic):
    - Any type is consistent with ?
    - Casts inserted at static/dynamic boundaries

  Consistency relation (~):
    Int ~ Int       (reflexive on ground types)
    Int ~ ?         (any type consistent with ?)
    ? ~ Bool        (symmetric)
    Int ~/~ Bool    (incompatible ground types)

  Languages: TypeScript, Python (mypy), Dart, Typed Racket, C# (dynamic)

  Benefits: incremental migration from untyped to typed code
  Cost: runtime cast failures, performance overhead at boundaries
```

## Type Soundness

```
A type system is sound if well-typed programs don't go wrong:

  "Well-typed programs cannot go wrong." -- Milner (1978)

Two lemmas (Wright & Felleisen, 1994):

  Progress:     If Gamma |- e : tau, then either e is a value
                or there exists e' such that e --> e'.
                ("Well-typed terms don't get stuck.")

  Preservation: If Gamma |- e : tau and e --> e', then Gamma |- e' : tau.
                ("Reduction preserves types.")
                Also called "subject reduction."

  Soundness = Progress + Preservation

  Strong normalization:
    Every reduction sequence terminates.
    Holds for STLC, System F, System F-omega.
    Does NOT hold for untyped lambda calculus or Turing-complete languages.
```

## Key Figures

```
Alonzo Church (1903-1995)
  - Simply typed lambda calculus (1940)
  - Foundation for all typed lambda calculi

Haskell Curry (1900-1982)
  - Curry-Howard correspondence (types = propositions)
  - Combinatory logic, currying

William Howard (b. 1926)
  - Formulae-as-types interpretation (1969, published 1980)
  - Extended Curry's observation to full intuitionistic logic

Roger Hindley (b. 1939)
  - Principal type theorem for combinatory logic (1969)
  - Independently discovered Algorithm W

Robin Milner (1934-2010)
  - Algorithm W for type inference (1978)
  - ML language, "well-typed programs cannot go wrong"
  - Turing Award 1991

Per Martin-Lof (b. 1942)
  - Intuitionistic type theory (1971, 1984)
  - Dependent types, identity types, universes
  - Foundation for Coq, Agda, Lean, HoTT

Jean-Yves Girard (b. 1947)
  - System F (1972), linear logic (1987)
  - Proof nets, geometry of interaction
```

## Tips

- The Curry-Howard correspondence is the single most important idea in type theory -- internalize it deeply
- Hindley-Milner gives you the best power-to-annotation ratio: full inference with parametric polymorphism
- Dependent types are the logical endpoint of type systems but make type checking undecidable in general
- Linear types solve resource management; Rust's borrow checker is an affine type system in practice
- System F is undecidable for inference but decidable for checking -- that is why Haskell requires annotations for rank-2+ types
- Algebraic data types are called "algebraic" because their cardinalities follow the laws of algebra
- Gradual typing is a pragmatic compromise: add types incrementally without rewriting everything
- Type soundness proofs (progress + preservation) are the gold standard for language correctness

## See Also

- `detail/cs-theory/type-theory.md` -- STLC typing rules, Algorithm W walkthrough, System F, Curry-Howard table, dependent types, Martin-Lof type theory, linear logic
- `sheets/cs-theory/lambda-calculus.md` -- untyped lambda calculus, reduction, Church encodings
- `sheets/cs-theory/category-theory.md` -- functors, monads, categorical semantics of types
- `sheets/cs-theory/automata-theory.md` -- decidability, Turing completeness, halting problem

## References

- "Types and Programming Languages" by Benjamin Pierce (MIT Press, 2002)
- "Advanced Topics in Types and Programming Languages" ed. Pierce (MIT Press, 2005)
- "Proofs and Types" by Girard, Lafont, and Taylor (Cambridge, 1989)
- "Programming in Martin-Lof's Type Theory" by Nordstrom, Petersson, Smith (Oxford, 1990)
- Milner, R. "A Theory of Type Polymorphism in Programming" (JCSS 17, 1978)
- Damas, L. and Milner, R. "Principal type-schemes for functional programs" (POPL, 1982)
- Wadler, P. "Theorems for free!" (FPCA, 1989)
- Girard, J.-Y. "Linear logic" (Theoretical Computer Science, 1987)
- Howard, W. "The formulae-as-types notion of construction" (1969, published 1980)
