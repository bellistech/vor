# Kolmogorov Complexity (Algorithmic Information Theory)

A complete reference for Kolmogorov complexity, the theory of shortest descriptions, incompressibility, and algorithmic randomness — the bridge between computation, information, and randomness.

## Kolmogorov Complexity Definition

### K(x) -- Shortest Description Length

```
K(x) = min { |p| : U(p) = x }

  U    — a fixed universal Turing machine (UTM)
  p    — a program (binary string) that U executes
  |p|  — the length of p in bits
  x    — the string being described

K(x) is the length of the shortest program that produces x on U and halts.
```

### Intuition

```
Short K(x)  =>  x has a short description  =>  x has structure/pattern
Large K(x)  =>  x has no short description  =>  x is "random"

Examples:
  x = 000...0 (n zeros)    K(x) <= log2(n) + c       (compressible)
  x = 01010101...01         K(x) <= log2(n) + c       (compressible)
  x = random binary string  K(x) >= n - c  (w.h.p.)   (incompressible)

  c is a constant depending on the UTM
```

## Invariance Theorem

### Statement

```
For any two universal Turing machines U1 and U2:

  |K_U1(x) - K_U2(x)| <= c

where c depends on U1 and U2 but NOT on x.

Consequence: K(x) is defined up to an additive constant.
             The choice of UTM does not matter asymptotically.
```

### Why It Works

```
U1 can simulate U2 with a fixed-length prefix (the simulator).
If p is the shortest program for x on U2, then:

  K_U1(x) <= |p| + |simulator| = K_U2(x) + c

Symmetrically for the other direction.
```

## Incomputability of K(x)

### K(x) Is Not Computable

```
Theorem: There is no algorithm that, given x, outputs K(x).

Proof sketch (Berry paradox):
  Suppose K were computable.
  Consider: "output the first string x with K(x) > n"
  This program has length ~log2(n) + c.
  But it produces x with K(x) > n.
  So K(x) <= log2(n) + c < n for large n.    Contradiction.

Consequence: We can approximate K(x) from above (by compression)
             but never know the true value.
```

### Upper Semicomputability

```
K(x) is upper semicomputable (enumerable from above):
  - Run all programs of length 1, 2, 3, ... in parallel
  - When program p outputs x, record |p| as an upper bound
  - The infimum of these bounds equals K(x)

But K(x) is NOT lower semicomputable => not computable.
```

## Incompressible Strings

### Counting Argument

```
Definition: x is c-incompressible if K(x) >= |x| - c.
            x is incompressible (random) if K(x) >= |x|.

Counting:
  Strings of length n:                     2^n
  Programs of length < n:    sum_{i=0}^{n-1} 2^i = 2^n - 1

  At most 2^n - 1 strings can have K(x) < n.
  So at least ONE string of length n has K(x) >= n.

  More generally: at least 2^n - 2^(n-c) + 1 strings have K(x) >= n - c.
  Fraction of c-incompressible strings >= 1 - 2^(-c).

  c=1: at least half of all strings are 1-incompressible
  c=10: at least 99.9% are 10-incompressible
```

### Properties of Incompressible Strings

```
If x is incompressible (length n):
  - x has roughly n/2 zeros and n/2 ones
  - Every substring pattern appears with expected frequency
  - x passes all polynomial-time statistical tests
  - x looks "random" by every effective test
```

## Conditional Complexity

### K(x|y) -- Complexity Given Side Information

```
K(x|y) = min { |p| : U(p, y) = x }

The length of the shortest program producing x when given y as input.

Properties:
  K(x|y) <= K(x) + c             (extra info never hurts much)
  K(x|y) = 0 + c  if y encodes x (x is computable from y)
  K(x|x) = O(1)                  (x given itself is trivial)
```

## Chain Rule for Kolmogorov Complexity

### Statement

```
K(x, y) = K(x) + K(y|x) + O(log(K(x, y)))

Equivalently:
  K(x, y) = K(y) + K(x|y) + O(log(K(x, y)))

The joint complexity of (x, y) equals the complexity of one
plus the conditional complexity of the other, up to log terms.

Compare with Shannon: H(X, Y) = H(X) + H(Y|X)
```

## Prefix-Free Complexity

### Definition

```
K_prefix(x) uses a prefix-free universal TM:
  - No valid program is a prefix of another
  - The program must be self-delimiting

K_prefix(x) >= K(x)  (prefix-free programs carry their own length)

Advantage: K_prefix satisfies the chain rule EXACTLY:
  K_prefix(x, y) = K_prefix(x) + K_prefix(y|x) + O(1)

Also known as: self-delimiting complexity (Chaitin, Levin)
```

### Kraft Inequality Connection

```
sum over all x: 2^{-K_prefix(x)} <= 1

This is the Kraft inequality — prefix-free codes form a valid
probability distribution (the universal semimeasure).
```

## Algorithmic Randomness (Martin-Lof Randomness)

### Definition

```
An infinite sequence w is Martin-Lof random if it passes
all effective statistical tests.

Formally: w is ML-random if w is not in any constructive null set.

A constructive null set (Martin-Lof test) is a sequence {U_n}
of uniformly r.e. open sets with measure(U_n) <= 2^{-n}.

Equivalently (Schnorr-Levin theorem):
  w is ML-random  <=>  K_prefix(w_1..n) >= n - O(1) for all n

  (the initial segments are all nearly incompressible)
```

### Key Results

```
ML-random sequences:
  - Satisfy the law of large numbers
  - Satisfy the law of the iterated logarithm
  - Are normal in every base
  - Have measure 1 (almost every sequence is ML-random)
  - But no specific ML-random sequence can be constructed
```

## Berry Paradox Connection

### The Paradox

```
"The smallest positive integer not definable in under sixty letters."

This sentence defines a number in under sixty letters,
contradicting the claim that the number is not so definable.

Formalized: The Berry paradox IS the proof that K(x) is incomputable.
  If K were computable, a short program could find strings with
  high complexity, producing a description shorter than K(x) says is possible.
```

## Connections to Shannon Entropy

### K(x) vs H(X)

```
Shannon entropy H(X): average information per symbol (distribution-based)
Kolmogorov complexity K(x): information in a specific string (object-based)

Key relationship (for i.i.d. random variable X):
  E[K(X_1...X_n)] = n * H(X) + O(log n)

  The expected Kolmogorov complexity of a random string
  equals the Shannon entropy rate times the string length.

H(X) is computable (given the distribution).
K(x) is not computable.
H(X) is a property of the source.
K(x) is a property of the individual string.
```

## Kolmogorov Structure Function

### Definition

```
h_x(alpha) = min { log |S| : x in S, K(S) <= alpha }

Given a "model complexity" budget alpha, find the smallest set S
containing x that can be described in alpha bits.

Captures the tradeoff between model complexity and data-to-model code.
Used in MDL and model selection.
```

## Minimum Description Length (MDL) Principle

### Idea

```
Best model M for data D minimizes:

  L(M) + L(D|M)

  L(M)   — description length of the model
  L(D|M) — description length of data given the model

Rooted in Kolmogorov complexity but made practical
by using computable code lengths instead of K(x).

Prevents overfitting: complex models have large L(M).
Prevents underfitting: poor models have large L(D|M).
```

## Solomonoff Induction

### Universal Prior

```
Solomonoff's universal prior for prediction:

  m(x) = sum over { p : U(p) = x } 2^{-|p|}

This is the probability that a random program produces x.
Dominates every computable probability distribution.

For prediction: given x_1..x_n, predict x_{n+1} by:
  P(x_{n+1} | x_1..x_n) = m(x_1..x_{n+1}) / m(x_1..x_n)

Properties:
  - Converges to the true distribution (if computable)
  - Optimal in a strong sense (dominance)
  - Incomputable (like K(x))
```

## Chaitin's Omega (Halting Probability)

### Definition

```
Omega = sum over { p : U(p) halts } 2^{-|p|}

Properties:
  - 0 < Omega < 1, well-defined real number
  - Omega is ML-random (its binary expansion is incompressible)
  - Omega is enumerable from below but not computable
  - Knowing the first n bits of Omega decides the halting problem
    for all programs of length <= n
  - Omega is "maximally unknowable" — the most random r.e. real
```

## Key Figures

```
Andrey Kolmogorov (1965)  — defined algorithmic complexity independently
Ray Solomonoff (1960)     — first to propose algorithmic probability/induction
Gregory Chaitin (1966)    — independent definition, Omega, incompleteness
Per Martin-Lof (1966)     — algorithmic randomness via constructive null sets
Leonid Levin (1973)       — prefix complexity, universal distribution
Ming Li & Paul Vitanyi    — definitive textbook, applications across CS
```

## See Also

- Turing Machines
- Information Theory
- Computability
- Entropy
- Data Compression
- Bayesian Inference

## References

```
Li, M. & Vitanyi, P. "An Introduction to Kolmogorov Complexity
  and Its Applications" (4th ed., Springer, 2019)
Kolmogorov, A.N. "Three approaches to the quantitative definition
  of information" (1965)
Solomonoff, R. "A formal theory of inductive inference" (1964)
Chaitin, G. "On the length of programs for computing finite
  binary sequences" (1966)
Grunwald, P. "The Minimum Description Length Principle" (MIT Press, 2007)
Cover, T. & Thomas, J. "Elements of Information Theory" (Chapter 14)
Downey, R. & Hirschfeldt, D. "Algorithmic Randomness and Complexity"
  (Springer, 2010)
```
