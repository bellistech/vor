# Category Theory (Abstract Algebra of Composition and Structure)

A compact reference for category theory -- the branch of mathematics that formalizes structure-preserving mappings between mathematical objects, providing a unifying language for algebra, topology, logic, and the foundations of functional programming.

## Categories

### Definition

```
A category C consists of:

  Objects:      A collection ob(C) of objects
  Morphisms:    For each pair A, B in ob(C), a set Hom(A, B) of morphisms (arrows)
  Composition:  For f : A -> B and g : B -> C, a composite g . f : A -> C
  Identity:     For each object A, a morphism id_A : A -> A

Axioms:
  Associativity:   h . (g . f) = (h . g) . f
  Identity:        f . id_A = f = id_B . f    for f : A -> B
```

### Examples of Categories

```
Set         Objects: sets            Morphisms: functions
Grp         Objects: groups          Morphisms: group homomorphisms
Top         Objects: topological sp. Morphisms: continuous maps
Vect_k      Objects: vector spaces   Morphisms: linear maps
Pos         Objects: posets          Morphisms: monotone maps
Hask        Objects: Haskell types   Morphisms: functions (a -> b)

A monoid is a category with exactly one object.
A preorder is a category with at most one morphism between any two objects.
```

## Functors

### Covariant Functor

```
A functor F : C -> D maps:
  Objects:    A  |->  F(A)
  Morphisms:  (f : A -> B)  |->  (F(f) : F(A) -> F(B))

Preserving:
  Identity:     F(id_A) = id_{F(A)}
  Composition:  F(g . f) = F(g) . F(f)

In programming: a functor is a type constructor with a lawful map/fmap operation.
  fmap id      = id
  fmap (g . f) = fmap g . fmap f
```

### Contravariant Functor

```
A contravariant functor F : C^op -> D reverses arrows:
  (f : A -> B)  |->  (F(f) : F(B) -> F(A))

In programming: Contravariant typeclass
  contramap :: (a -> b) -> f b -> f a

Example: Predicate a = a -> Bool
  contramap f pred = pred . f
```

### Endofunctors

```
A functor F : C -> C from a category to itself.

In Haskell, every Functor is an endofunctor on Hask:
  List   : Hask -> Hask       fmap f [x, y, z] = [f x, f y, f z]
  Maybe  : Hask -> Hask       fmap f (Just x)  = Just (f x)
  IO     : Hask -> Hask       fmap f action    = action >>= return . f
```

## Natural Transformations

```
Given functors F, G : C -> D, a natural transformation alpha : F => G is:
  A family of morphisms  alpha_A : F(A) -> G(A)  for each object A in C

such that for every morphism f : A -> B:
  alpha_B . F(f) = G(f) . alpha_A       (naturality square commutes)

       F(A) --F(f)--> F(B)
        |               |
    alpha_A         alpha_B
        |               |
        v               v
       G(A) --G(f)--> G(B)

In programming:
  safeHead :: [a] -> Maybe a       -- natural transformation List => Maybe
  listToMaybe, maybeToList         -- between List and Maybe
  reverse :: [a] -> [a]            -- natural transformation List => List

Natural transformations correspond to polymorphic functions
(parametric polymorphism guarantees the naturality condition for free).
```

## Products and Coproducts

### Products

```
The product of A and B is an object A x B with projections:
  pi_1 : A x B -> A
  pi_2 : A x B -> B

Universal property: for any C with f : C -> A, g : C -> B,
  there exists a unique h : C -> A x B such that
    pi_1 . h = f   and   pi_2 . h = g

In programming: tuples (a, b) with fst and snd
In Set: cartesian product
In logic: conjunction (A AND B)
```

### Coproducts

```
The coproduct of A and B is an object A + B with injections:
  inl : A -> A + B
  inr : B -> A + B

Universal property: for any C with f : A -> C, g : B -> C,
  there exists a unique h : A + B -> C such that
    h . inl = f   and   h . inr = g

In programming: Either a b with Left and Right
In Set: disjoint union
In logic: disjunction (A OR B)
```

## Monads

### Definition (Three Equivalent Forms)

```
1. Kleisli triple (T, return, >>=):
   T       : a type constructor
   return  : a -> T a
   (>>=)   : T a -> (a -> T b) -> T b

2. Endofunctor + natural transformations (T, eta, mu):
   T   : C -> C                 (endofunctor)
   eta : Id => T                (unit / return)
   mu  : T . T => T             (multiplication / join)

3. The two are related by:
   join   = (>>= id)
   m >>= f = join (fmap f m)
```

### Monad Laws

```
Kleisli form:
  Left identity:    return a >>= f       =  f a
  Right identity:   m >>= return         =  m
  Associativity:    (m >>= f) >>= g      =  m >>= (\x -> f x >>= g)

Endofunctor form:
  mu . T(eta)   = id_T        (left unit)
  mu . eta_T    = id_T        (right unit)
  mu . T(mu)    = mu . mu_T   (associativity)
```

### Common Monads in Programming

```
Maybe / Option:
  return x = Just x
  Nothing >>= f = Nothing
  Just x  >>= f = f x
  Use: computations that may fail

Either e:
  return x = Right x
  Left e  >>= f = Left e
  Right x >>= f = f x
  Use: computations with error information

List []:
  return x = [x]
  xs >>= f = concatMap f xs
  Use: nondeterministic computation

IO:
  return x = pure value in IO context
  action >>= f = sequence effects
  Use: side effects, I/O

State s:
  return x = \s -> (x, s)
  m >>= f  = \s -> let (a, s') = m s in f a s'
  Use: stateful computation

Reader r:
  return x = \_ -> x
  m >>= f  = \r -> f (m r) r
  Use: shared environment / configuration
```

## Kleisli Category

```
Given a monad (T, return, >>=) on category C, the Kleisli category C_T has:
  Objects:    same as C
  Morphisms:  a Kleisli arrow A -> T(B) in C becomes A -> B in C_T
  Composition: (g <=< f) x = g =<< f x    -- fish operator
               equivalently: f >=> g = \x -> f x >>= g
  Identity:   return : A -> T(A)

Kleisli composition is associative (follows from monad associativity law).
```

## Adjunctions

```
An adjunction F -| G between categories C and D consists of:
  Functors:  F : C -> D  (left adjoint)
             G : D -> C  (right adjoint)

Natural isomorphism:
  Hom_D(F(A), B) ~ Hom_C(A, G(B))      for all A in C, B in D

Equivalently, natural transformations:
  eta : Id_C => G . F          (unit)
  epsilon : F . G => Id_D      (counit)

satisfying the triangle identities:
  (epsilon . F) . (F . eta) = id_F
  (G . epsilon) . (eta . G) = id_G

Every adjunction gives rise to a monad: T = G . F, with
  unit   = eta
  join   = G(epsilon_F)
```

## Yoneda Lemma

```
For a locally small category C, functor F : C^op -> Set, and object A:

  Nat(Hom(A, -), F)  ~  F(A)

(Natural transformations from the representable functor Hom(A, -) to F
 are in bijection with elements of F(A).)

The bijection sends alpha : Hom(A, -) => F to alpha_A(id_A).

Corollary (Yoneda embedding): C embeds fully and faithfully into Set^{C^op}.
  Two objects are isomorphic iff their representable functors are.

In programming: forall b. (a -> b) -> f b  ~  f a
  (CPS / continuation-passing is a computational form of Yoneda)
```

## Cartesian Closed Categories

```
A category C is cartesian closed (CCC) if it has:
  1. A terminal object 1
  2. Binary products A x B for all A, B
  3. Exponential objects B^A for all A, B

Exponentials satisfy:
  Hom(A x B, C) ~ Hom(A, C^B)        (currying / uncurrying)

CCCs model the simply typed lambda calculus:
  Types        <->  Objects
  Terms        <->  Morphisms
  Functions    <->  Exponentials
  Tuples       <->  Products
  Unit         <->  Terminal object
  Evaluation   <->  eval : B^A x A -> B
```

## Toposes (Brief)

```
A topos is a category that behaves like a generalized universe of sets:
  1. Has all finite limits (including products, equalizers, terminal object)
  2. Has exponential objects (is cartesian closed)
  3. Has a subobject classifier Omega with true : 1 -> Omega

Set is the canonical topos. Sheaf categories Sh(X) over topological
spaces are toposes. Every topos has an internal logic (intuitionistic).

Key idea: replace "true/false" with a richer truth-value object Omega.
In Set, Omega = {0, 1}. In a general topos, Omega can be a Heyting algebra.
```

## Connection to Programming

```
Category Theory          Programming
---------------------------------------------------
Category                 Type system
Object                   Type
Morphism                 Function
Functor                  Type constructor with fmap
Natural transformation   Polymorphic function
Product                  Tuple (a, b) / struct
Coproduct                Either a b / tagged union
Exponential              Function type a -> b
Monad                    Computation with effects
return / eta             Wrapping a pure value
join / mu                Flattening nested effects
>>= / bind              Sequencing effectful computations
Kleisli arrow            a -> m b (effectful function)
Adjunction               Free/forgetful functor pair
Yoneda                   CPS transform / defunctionalization
CCC                      Simply typed lambda calculus
Topos                    Intuitionistic type theory
```

## Key Figures

```
Saunders Mac Lane (1909-2005)
  - Co-invented category theory (1945, with Eilenberg)
  - Authored "Categories for the Working Mathematician" (1971)
  - Developed monad theory (with Beck)
  - "I did not invent category theory to talk about functors.
    I invented it to talk about natural transformations."

Samuel Eilenberg (1913-1998)
  - Co-invented category theory (1945, with Mac Lane)
  - Pioneer of homological algebra and algebraic topology
  - Eilenberg-Moore and Eilenberg-Zilber constructions
  - The original paper: "General Theory of Natural Equivalences"

Philip Wadler (b. 1956)
  - Brought monads into functional programming (1992)
  - "Monads for functional programming" -- foundational paper
  - Key contributor to Haskell's type system
  - Theorems for free (parametricity and naturality)
  - "The essence of functional programming" (1992)
```

## Tips

- A monad is just a monoid in the category of endofunctors (Mac Lane) -- this is literally true, not just a joke
- Think of functors as structure-preserving maps: they carry objects AND arrows, respecting composition
- Natural transformations are "the right notion of morphism between functors" -- they are why category theory was invented
- Every adjunction gives a monad; not every monad comes from a unique adjunction
- The Yoneda lemma is arguably the most important result -- "you are completely determined by how others see you"
- Products are AND, coproducts are OR, exponentials are IMPLIES -- the Curry-Howard-Lambek correspondence
- Start with concrete categories (Set, Hask) before attempting abstract diagram chasing

## See Also

- `detail/cs-theory/category-theory.md` -- formal axioms, monad laws proofs, Yoneda lemma, adjunction-monad correspondence
- `sheets/cs-theory/lambda-calculus.md` -- the computational system that CCCs model
- `sheets/cs-theory/type-theory.md` -- types as objects, Curry-Howard-Lambek
- `sheets/cs-theory/computability-theory.md` -- alternative foundations of computation

## References

- "Categories for the Working Mathematician" by Saunders Mac Lane (Springer, 2nd ed., 1998)
- "Category Theory for Programmers" by Bartosz Milewski (2019, also available online)
- Eilenberg, S. and Mac Lane, S. "General Theory of Natural Equivalences" (Trans. AMS, 1945)
- Wadler, P. "Monads for functional programming" (Advanced Functional Programming, 1995)
- Moggi, E. "Notions of computation and monads" (Information and Computation, 1991)
- Awodey, S. "Category Theory" (Oxford Logic Guides, 2nd ed., 2010)
- Mac Lane, S. and Moerdijk, I. "Sheaves in Geometry and Logic: A First Introduction to Topos Theory" (Springer, 1994)
