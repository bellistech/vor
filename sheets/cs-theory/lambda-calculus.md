# Lambda Calculus (Foundations of Computation and Functional Programming)

A compact reference for lambda calculus -- the minimal universal model of computation invented by Alonzo Church, underlying every functional programming language and serving as the theoretical backbone of type theory, proof theory, and programming language semantics.

## Syntax

### The Three Constructs

```
Every lambda term is built from exactly three forms:

  Variable:      x
  Abstraction:   (lambda x. M)      -- function definition
  Application:   (M N)              -- function application

BNF Grammar:
  <term> ::= <variable>
           | (lambda <variable>. <term>)
           | (<term> <term>)

Convention: application is left-associative, abstraction extends as far right as possible.
  lambda x. lambda y. x y  =  lambda x. (lambda y. (x y))
  M N P                    =  ((M N) P)
```

### Free and Bound Variables

```
In (lambda x. M):
  x is bound in M
  Any variable not bound by an enclosing lambda is free

FV(x)            = {x}
FV(lambda x. M)  = FV(M) \ {x}
FV(M N)          = FV(M) union FV(N)

Example:
  lambda x. x y       FV = {y}       (x is bound, y is free)
  lambda x. lambda y. x y z   FV = {z}
```

## Reduction Rules

### Alpha Reduction (Renaming)

```
Rename bound variables to avoid capture:

  lambda x. M  -->alpha  lambda y. M[x := y]

provided y is not free in M.

Example:
  lambda x. x  -->alpha  lambda y. y
```

### Beta Reduction (Computation)

```
Apply a function to its argument:

  (lambda x. M) N  -->beta  M[x := N]

Substitution M[x := N] replaces all free occurrences of x in M with N,
renaming bound variables as needed to avoid capture.

Examples:
  (lambda x. x) y               -->  y
  (lambda x. x x)(lambda y. y)  -->  (lambda y. y)(lambda y. y)  -->  lambda y. y
  (lambda x. lambda y. x) a b   -->  (lambda y. a) b  -->  a
```

### Eta Reduction (Extensionality)

```
Remove redundant abstraction:

  lambda x. M x  -->eta  M

provided x is not free in M.

Eta expresses extensionality: two functions are equal if they
produce the same output for every input.
```

## Normal Forms

```
A term is in normal form if no beta reduction can be applied.

Beta normal form:     No beta redexes remain
Head normal form:     lambda x1...xn. y M1 ... Mk    (head is a variable)
Weak head normal form: lambda x. M  or  x M1 ... Mk  (no top-level redex)

Not all terms have a normal form:
  Omega = (lambda x. x x)(lambda x. x x)  -->beta  itself (diverges)
```

## Evaluation Strategies

```
Given a term with multiple redexes, which to reduce first?

Call-by-name (leftmost outermost):
  - Reduce the leftmost outermost redex first
  - Arguments are NOT evaluated before substitution
  - Normalizing: finds normal form if one exists (by Normalization Theorem)
  - Haskell (lazy evaluation is memoized call-by-name)

Call-by-value (leftmost innermost):
  - Evaluate arguments to values before substitution
  - May diverge even when a normal form exists
  - Most languages: OCaml, Scheme, ML, JavaScript

Example where strategy matters:
  (lambda x. lambda y. x) a Omega
  Call-by-name:  --> (lambda y. a) Omega  --> a     (terminates)
  Call-by-value: tries to evaluate Omega first       (diverges)
```

## Church Numerals

```
Natural numbers encoded as higher-order functions:

  0 = lambda f. lambda x. x
  1 = lambda f. lambda x. f x
  2 = lambda f. lambda x. f (f x)
  3 = lambda f. lambda x. f (f (f x))
  n = lambda f. lambda x. f^n x       (apply f n times)

Successor:
  SUCC = lambda n. lambda f. lambda x. f (n f x)

Addition:
  ADD = lambda m. lambda n. lambda f. lambda x. m f (n f x)

Multiplication:
  MUL = lambda m. lambda n. lambda f. m (n f)

Exponentiation:
  EXP = lambda m. lambda n. n m
```

## Church Booleans and Control Flow

```
TRUE  = lambda t. lambda f. t
FALSE = lambda t. lambda f. f

AND = lambda p. lambda q. p q FALSE
OR  = lambda p. lambda q. p TRUE q
NOT = lambda p. p FALSE TRUE

IF-THEN-ELSE = lambda p. lambda a. lambda b. p a b

Example:
  IF TRUE  a b  -->  (lambda t. lambda f. t) a b  -->  a
  IF FALSE a b  -->  (lambda t. lambda f. f) a b  -->  b

Pairs:
  PAIR  = lambda x. lambda y. lambda f. f x y
  FST   = lambda p. p TRUE
  SND   = lambda p. p FALSE
```

## Y Combinator (Fixed-Point)

```
Recursion in lambda calculus via fixed-point combinators:

  Y = lambda f. (lambda x. f (x x))(lambda x. f (x x))

Property: Y F  -->  F (Y F)
  Y F = (lambda x. F (x x))(lambda x. F (x x))
      -->beta  F ((lambda x. F (x x))(lambda x. F (x x)))
      = F (Y F)

Call-by-value variant (Z combinator / Curry's paradoxical combinator):
  Z = lambda f. (lambda x. f (lambda v. x x v))(lambda x. f (lambda v. x x v))

Factorial via Y:
  FACT = Y (lambda f. lambda n. IF (ISZERO n) 1 (MUL n (f (PRED n))))
```

## Simply Typed Lambda Calculus

```
Add types to prevent nontermination and paradoxes:

Types:
  tau ::= alpha             (base type)
        | tau -> tau        (function type)

Typing rules:
                 x : tau in Gamma
  Variable:     -------------------
                  Gamma |- x : tau

                Gamma, x : sigma |- M : tau
  Abstraction: ----------------------------
                Gamma |- (lambda x. M) : sigma -> tau

                Gamma |- M : sigma -> tau    Gamma |- N : sigma
  Application: -----------------------------------------------
                         Gamma |- M N : tau

Properties:
  - Strong normalization: every well-typed term reduces to a normal form
  - Decidable type checking and type inference
  - NOT Turing-complete (cannot express Y combinator)
```

## Connection to Functional Programming

```
Lambda Calculus          Functional Programming
--------------------------------------------------
Abstraction              Anonymous function / closure
Application              Function call
Beta reduction           Evaluation / execution
Free variables           Variables from enclosing scope
Church numerals          Peano-style natural numbers
Y combinator             Recursive let bindings
Alpha equivalence        Variable shadowing
Call-by-name             Lazy evaluation (Haskell)
Call-by-value            Strict evaluation (ML, Scheme)
Simply typed LC          Hindley-Milner type systems
Curry-Howard             Types as propositions, programs as proofs
```

## Key Figures

```
Alonzo Church (1903-1995)
  - Invented lambda calculus (1930s)
  - Proved the undecidability of the Entscheidungsproblem
  - Church-Turing thesis: lambda calculus = Turing machines in power
  - PhD advisor to Alan Turing, Stephen Kleene, and others

Haskell Curry (1900-1982)
  - Curry-Howard correspondence (types = propositions)
  - Combinatory logic (S, K, I combinators)
  - Currying named after him (Church independently discovered it)
  - The Haskell language is named in his honor

Dana Scott (b. 1932)
  - Denotational semantics for lambda calculus (Scott domains, 1969)
  - Proved that untyped lambda calculus has nontrivial models
  - Scott-Curry theorem on undefinability
  - Scott topology and continuous lattices
```

## Tips

- Lambda calculus has only three constructs but is Turing-complete -- complexity emerges from composition
- When reducing, track free variables carefully to avoid variable capture during substitution
- The Y combinator only works under call-by-name; use the Z combinator for call-by-value languages
- Church numerals are slow (unary) -- real languages use them conceptually, not literally
- Simply typed lambda calculus trades Turing-completeness for guaranteed termination
- Understanding beta reduction deeply is the key to understanding closures in any language
- De Bruijn indices eliminate alpha equivalence issues entirely (see detail page)

## See Also

- `detail/cs-theory/lambda-calculus.md` -- Church-Rosser theorem, de Bruijn indices, Curry-Howard correspondence
- `sheets/cs-theory/computability-theory.md` -- Turing machines, decidability, Church-Turing thesis
- `sheets/cs-theory/type-theory.md` -- dependent types, System F, polymorphism
- `sheets/cs-theory/category-theory.md` -- functors, monads, categorical semantics

## References

- "Lambda-Calculus and Combinators: An Introduction" by Hindley & Seldin (Cambridge, 2008)
- "The Lambda Calculus: Its Syntax and Semantics" by Barendregt (North-Holland, 1984)
- "Types and Programming Languages" by Benjamin Pierce (MIT Press, 2002)
- Church, A. "An Unsolvable Problem of Elementary Number Theory" (American J. of Mathematics, 1936)
- Turing, A. "Computability and lambda-definability" (Journal of Symbolic Logic, 1937)
