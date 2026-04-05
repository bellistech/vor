# Lattice-Based Cryptography (Post-Quantum Foundations)

A complete reference for lattice-based cryptographic constructions — the leading candidate family for post-quantum security, underpinning NIST's new standards for encryption and digital signatures.

## Lattice Fundamentals

### Definition

```
A lattice L is a discrete additive subgroup of R^n.

Given linearly independent vectors b1, b2, ..., bn in R^n (the basis),
the lattice they generate is:

  L(B) = { sum(z_i * b_i) : z_i in Z }

where B = [b1 | b2 | ... | bn] is the basis matrix.

The same lattice has infinitely many bases:
  B' = U * B  for any unimodular matrix U (det(U) = +/-1)
```

### Fundamental Domain and Determinant

```
The fundamental domain (parallelepiped) of basis B:

  P(B) = { sum(x_i * b_i) : 0 <= x_i < 1 }

Lattice determinant:
  det(L) = |det(B)|     (invariant across all bases)

Volume of fundamental domain = det(L)

Minkowski's theorem:
  Every n-dim lattice L with det(L) < vol(S)/2^n
  contains a nonzero point in convex symmetric body S.

  Corollary (shortest vector bound):
    lambda_1(L) <= sqrt(n) * det(L)^(1/n)
```

## Hard Lattice Problems

### Shortest Vector Problem (SVP)

```
Input:  Basis B for lattice L
Output: Nonzero vector v in L minimizing ||v||

  Exact SVP:   find v with ||v|| = lambda_1(L)
  GapSVP_gamma: distinguish lambda_1(L) <= 1 from lambda_1(L) > gamma

  Best known algorithms:
    Exact:  2^O(n) time and space (sieving)
    Approx: 2^O(n/log(gamma)) for gamma-approx (LLL gives gamma = 2^(n/2))
```

### Closest Vector Problem (CVP)

```
Input:  Basis B for lattice L, target vector t in R^n
Output: Lattice vector v in L minimizing ||v - t||

  CVP is at least as hard as SVP.
  Used directly in some cryptographic reductions.
```

### Shortest Independent Vectors Problem (SIVP)

```
Input:  Basis B for n-dimensional lattice L
Output: n linearly independent lattice vectors v1,...,vn
        minimizing max(||v_i||)

  GapSIVP_gamma: find n independent vectors of length <= gamma * lambda_n(L)
```

## Learning With Errors (LWE)

### Standard LWE (Regev, 2005)

```
Parameters: n (dimension), q (modulus), chi (error distribution, typically discrete Gaussian)

Secret:  s in Z_q^n  (chosen uniformly)

LWE samples:  (a_i, b_i)  where
  a_i <- Z_q^n  (uniform)
  b_i = <a_i, s> + e_i  (mod q)
  e_i <- chi

Search-LWE:  Given m samples, find s.
Decision-LWE: Distinguish (a_i, b_i) from uniform (a_i, u_i).

Regev's reduction (2005):
  Solving Decision-LWE is at least as hard as
  worst-case GapSVP_gamma and SIVP_gamma
  for gamma = O(n * sqrt(n) / alpha)  where alpha relates to error rate.
```

### Ring-LWE

```
Work over the ring R = Z[x]/(x^n + 1) where n is a power of 2.
Ring R_q = R/qR = Z_q[x]/(x^n + 1).

Secret:  s in R_q
Samples: (a_i, b_i) where
  a_i <- R_q (uniform)
  b_i = a_i * s + e_i  in R_q
  e_i <- chi (small polynomial)

Advantages over plain LWE:
  - Key size:    O(n log q) vs O(n^2 log q)
  - Operations:  O(n log n) via NTT vs O(n^2)

Hardness reduces to worst-case ideal lattice problems
(Lyubashevsky, Peikert, Regev 2010).
```

### Module-LWE

```
Generalization: work over R_q^k (rank-k free module over R_q).

Secret:  s in R_q^k
Samples: (a_i, b_i) where a_i in R_q^k, b_i = <a_i, s> + e_i in R_q

  k = 1  -->  Ring-LWE
  k = n  -->  (essentially) standard LWE

Module-LWE is the basis for Kyber/ML-KEM and Dilithium/ML-DSA.
Allows tuning security by adjusting k without changing polynomial degree.
```

## Worst-Case to Average-Case Reductions

### Ajtai's Breakthrough (1996)

```
First worst-case to average-case reduction in lattice cryptography.

Showed: breaking a random instance of a certain lattice problem
        is as hard as solving GapSVP in the WORST case.

Consequence: random lattice-based constructions inherit worst-case hardness.
No need to find "hard instances" — almost all instances are hard.
```

### Regev's Reduction (2005)

```
Quantum reduction: worst-case GapSVP/SIVP --> Decision-LWE

Key insight: uses a quantum algorithm to sample from
discrete Gaussian distributions on arbitrary lattices.

Classical reductions also known (Peikert 2009, Brakerski et al. 2013)
with somewhat weaker parameters.
```

## NIST Post-Quantum Standards (2024)

### ML-KEM (formerly Kyber) -- FIPS 203

```
Type:       Key Encapsulation Mechanism (KEM)
Hardness:   Module-LWE
Ring:       Z_q[x]/(x^256 + 1), q = 3329

Parameter sets:
  ML-KEM-512:   k=2, security ~AES-128
  ML-KEM-768:   k=3, security ~AES-192
  ML-KEM-1024:  k=4, security ~AES-256

Key sizes (bytes):
                pk      sk      ct
  ML-KEM-512:  800    1632     768
  ML-KEM-768:  1184   2400    1088
  ML-KEM-1024: 1568   3168    1568

Operations: KeyGen, Encaps, Decaps
Uses Fujisaki-Okamoto transform for CCA security.
```

### ML-DSA (formerly Dilithium) -- FIPS 204

```
Type:       Digital Signature
Hardness:   Module-LWE + Module-SIS (Short Integer Solution)
Ring:       Z_q[x]/(x^256 + 1), q = 8380417

Parameter sets:
  ML-DSA-44:  security ~AES-128
  ML-DSA-65:  security ~AES-192
  ML-DSA-87:  security ~AES-256

Sizes (bytes):
              pk      sk      sig
  ML-DSA-44:  1312   2560    2420
  ML-DSA-65:  1952   4032    3293
  ML-DSA-87:  2592   4896    4595

Based on Fiat-Shamir with Aborts paradigm (Lyubashevsky 2009, 2012).
```

### SLH-DSA (formerly SPHINCS+) -- FIPS 205

```
Type:       Digital Signature (hash-based, stateless)
Hardness:   Hash function security (not lattice-based)
            Included for diversity — no lattice assumptions needed.

Conservative fallback if lattice assumptions break.
Larger signatures (~7-50 KB) but minimal assumptions.
```

## Lattice-Based Encryption

### Regev's Encryption Scheme (2005)

```
Setup:
  n = security parameter
  q = prime, q >= n^2
  m = O(n log q)
  chi = discrete Gaussian with std dev alpha*q (alpha < 1/sqrt(n))

KeyGen:
  A <- Z_q^(m x n) (uniform)
  s <- Z_q^n (secret)
  e <- chi^m
  b = A*s + e (mod q)
  pk = (A, b),  sk = s

Encrypt(pk, bit mu):
  Choose random subset S of [m]
  c1 = sum(a_i for i in S) mod q    (in Z_q^n)
  c2 = sum(b_i for i in S) + mu * floor(q/2) mod q

Decrypt(sk, c1, c2):
  v = c2 - <c1, s> mod q
  output 0 if |v| < q/4, else output 1
```

## Lattice-Based Signatures

### Fiat-Shamir with Aborts (Lyubashevsky)

```
Paradigm for lattice signatures:

  1. Commit:  y <- D_sigma^n (Gaussian), w = A*y mod q
  2. Challenge: c = H(w, message)
  3. Response: z = y + c*s
  4. Abort if z reveals information about s
     (rejection sampling to ensure z is independent of s)
  5. Verify: check A*z = w + c*t mod q and ||z|| is small

The abort step is essential:
  Without it, z = y + c*s leaks s over many signatures.
  Rejection sampling ensures z follows D_sigma^n regardless of s.
```

## Fully Homomorphic Encryption (FHE)

### Gentry's Breakthrough (2009)

```
First construction of FHE — compute arbitrary functions on encrypted data.

Blueprint:
  1. Somewhat Homomorphic Encryption (SHE):
     supports limited additions and multiplications
     (noise grows with each operation)
  2. Bootstrapping: homomorphically evaluate the decryption circuit
     to "refresh" a ciphertext, reducing noise
  3. SHE + Bootstrapping = FHE (unlimited operations)

Requirement: SHE must be "bootstrappable" —
  capable of evaluating its own (augmented) decryption circuit.

Original: based on ideal lattices. Later schemes use LWE/Ring-LWE.
```

### Modern FHE Schemes

```
BGV (Brakerski-Gentry-Vaikuntanathan, 2012):
  - LWE-based, modulus switching for noise management
  - Leveled FHE (depth-bounded) without bootstrapping

BFV (Brakerski/Fan-Vercauteren, 2012):
  - Scale-invariant variant of BGV
  - Simpler noise analysis, widely implemented

CKKS (Cheon-Kim-Kim-Song, 2017):
  - Approximate arithmetic on encrypted real/complex numbers
  - Encodes messages in noise — controlled precision loss
  - Key scheme for machine learning on encrypted data

TFHE (Chillotti et al., 2016):
  - Fast bootstrapping (~13ms per gate)
  - Boolean circuit evaluation on encrypted bits
```

## Lattice Reduction

### LLL Algorithm (Lenstra-Lenstra-Lovasz, 1982)

```
Input:  Basis B = {b1, ..., bn}
Output: delta-reduced basis with short, nearly orthogonal vectors

Conditions for LLL-reduced basis:
  1. Size-reduced: |mu_{i,j}| <= 1/2 for all i > j
     where mu_{i,j} = <b_i, b*_j> / <b*_j, b*_j>
     (b*_j is Gram-Schmidt orthogonalization)
  2. Lovasz condition: ||b*_i||^2 >= (delta - mu_{i,i-1}^2) * ||b*_{i-1}||^2
     (typically delta = 3/4)

Properties:
  - Runs in polynomial time: O(n^5 * d * log^3 B) where d = dimension
  - Achieves approximation factor 2^((n-1)/2) for SVP
  - First vector satisfies: ||b1|| <= 2^((n-1)/4) * det(L)^(1/n)
```

### BKZ (Block Korkine-Zolotarev)

```
Generalization of LLL using SVP oracle on blocks of size beta.

  BKZ-beta:
    For each block of beta consecutive vectors,
    solve exact SVP and update basis.

  Approximation factor: gamma ~ beta^(n/beta)
  Running time:         2^O(beta) per block (using sieving SVP oracle)

  beta = 2:  equivalent to LLL
  beta = n:  equivalent to HKZ reduction (exact SVP)

  Practical cryptanalysis uses BKZ-2.0 with progressive sieving.
  Security estimates: "BKZ block size needed to break" = core metric.
```

## Key Figures

```
Oded Regev      - LWE, worst-case to average-case reduction (2005),
                  Regev encryption scheme. 2024 Godel Prize.
Craig Gentry    - First FHE construction (2009), bootstrapping.
Miklos Ajtai    - First worst-case/average-case reduction for lattices (1996),
                  Ajtai hash function.
Chris Peikert   - Classical LWE reductions, Ring-LWE foundations,
                  lattice trapdoors.
Vadim Lyubashevsky - Fiat-Shamir with Aborts, Ring-LWE, co-designer of
                     Dilithium/ML-DSA and Kyber/ML-KEM.
```

## See Also

- Computational Complexity
- Number Theory
- Turing Machines
- Post-Quantum Cryptography

## References

```
Regev, "On Lattices, Learning with Errors, Random Linear Codes,
  and Cryptography" (2005) — JACM 2009
Ajtai, "Generating Hard Instances of Lattice Problems" (1996) — STOC
Gentry, "Fully Homomorphic Encryption Using Ideal Lattices" (2009) — STOC
Lyubashevsky, Peikert, Regev, "On Ideal Lattices and Learning with
  Errors Over Rings" (2010) — EUROCRYPT
Peikert, "A Decade of Lattice Cryptography" (2016) — survey
NIST FIPS 203 (ML-KEM), FIPS 204 (ML-DSA), FIPS 205 (SLH-DSA) — 2024
Micciancio, Regev, "Lattice-based Cryptography" — Post-Quantum
  Cryptography textbook chapter (2009)
Lenstra, Lenstra, Lovasz, "Factoring Polynomials with Rational
  Coefficients" (1982) — Math. Ann.
```
