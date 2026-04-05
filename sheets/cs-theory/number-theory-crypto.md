# Number Theory for Cryptography (Primes, Modular Arithmetic, and Public-Key Foundations)

A practitioner's reference for the number-theoretic foundations underpinning modern public-key cryptography: modular arithmetic, primality testing, factoring, discrete logarithms, RSA, and finite fields.

## Modular Arithmetic

### Congruences

```
a ≡ b (mod n)   means   n | (a - b)

Properties:
  If a ≡ b and c ≡ d (mod n), then:
    a + c ≡ b + d (mod n)
    a * c ≡ b * d (mod n)
    a^k ≡ b^k (mod n)       for k >= 0
```

### Modular Inverse

```
a^(-1) mod n exists  iff  gcd(a, n) = 1

Finding the inverse:
  Extended Euclidean Algorithm    O(log n)
  Euler's theorem: a^(phi(n)-1)   when gcd(a,n)=1
  Fermat's little theorem: a^(p-2) mod p   when p is prime
```

### Chinese Remainder Theorem (CRT)

```
Given pairwise coprime moduli m1, m2, ..., mk and any a1, ..., ak:

  x ≡ a1 (mod m1)
  x ≡ a2 (mod m2)
  ...
  x ≡ ak (mod mk)

has a unique solution modulo M = m1 * m2 * ... * mk.

Solution: x = sum( ai * Mi * yi )  mod M
  where Mi = M / mi,  yi = Mi^(-1) mod mi
```

| Operation | Condition | Complexity |
|---|---|---|
| a mod n | -- | O(1) (fixed precision) |
| gcd(a, b) | -- | O(log min(a,b)) |
| Extended GCD | -- | O(log min(a,b)) |
| Modular inverse | gcd(a,n) = 1 | O(log n) |
| Modular exponentiation | -- | O(log e * (log n)^2) |
| CRT reconstruction | pairwise coprime | O(k * (log M)^2) |

## Euler's Totient Function

```
phi(n) = |{ a : 1 <= a <= n, gcd(a, n) = 1 }|

Key formulas:
  phi(p)       = p - 1                         (p prime)
  phi(p^k)     = p^(k-1) * (p - 1)
  phi(m * n)   = phi(m) * phi(n)               when gcd(m,n) = 1
  phi(n)       = n * prod(1 - 1/p)             over all prime p | n
```

### Examples

```
phi(12)  = 12 * (1 - 1/2) * (1 - 1/3) = 4
phi(100) = 100 * (1 - 1/2) * (1 - 1/5) = 40
phi(p*q) = (p-1)(q-1)    (RSA modulus, p,q distinct primes)
```

## Fermat's Little Theorem

```
If p is prime and gcd(a, p) = 1:

  a^(p-1) ≡ 1 (mod p)

Equivalently:  a^p ≡ a (mod p)   for all a

Application:  Modular inverse via a^(p-2) mod p
Warning:      Converse is FALSE — Carmichael numbers are composites
              that satisfy a^(n-1) ≡ 1 for all gcd(a,n)=1
              (e.g., 561 = 3 * 11 * 17)
```

## Euler's Theorem

```
If gcd(a, n) = 1:

  a^phi(n) ≡ 1 (mod n)

Generalizes Fermat (phi(p) = p-1).
Foundation of RSA correctness.
```

## Prime Numbers and Primality Testing

### Prime Distribution

```
Prime counting function:  pi(n) ~ n / ln(n)
Prime number theorem:     density of primes near n is 1/ln(n)
Probability a random k-bit number is prime:  ~1/(k * ln 2)
```

### Primality Tests

| Algorithm | Type | Complexity | Notes |
|---|---|---|---|
| Trial division | Deterministic | O(sqrt(n)) | Only for small n |
| Fermat test | Probabilistic | O(k * log^2 n) | Fooled by Carmichael numbers |
| Miller-Rabin | Probabilistic | O(k * log^2 n) | Error <= 4^(-k) for k rounds |
| Solovay-Strassen | Probabilistic | O(k * log^2 n) | Error <= 2^(-k) |
| AKS | Deterministic | O(log^6 n) (improved) | First poly-time deterministic test (2002) |
| ECPP | Probabilistic/cert | Heuristic poly | Produces a certificate |

### Miller-Rabin Test (Pseudocode)

```
Input: n (odd, > 3), k (security parameter)
Write n - 1 = 2^s * d  (d odd)
Repeat k times:
  Pick random a in [2, n-2]
  x = a^d mod n
  if x == 1 or x == n-1: continue
  for r = 1 to s-1:
    x = x^2 mod n
    if x == n-1: continue outer
  return COMPOSITE
return PROBABLY PRIME
```

## Integer Factoring

| Algorithm | Complexity | Type |
|---|---|---|
| Trial division | O(sqrt(n)) | Deterministic |
| Pollard's rho | O(n^(1/4)) expected | Randomized |
| Pollard's p-1 | O(B * log n) | Deterministic (smooth) |
| Quadratic sieve (QS) | L(1/2, 1) | Sub-exponential |
| Number field sieve (NFS) | L(1/3, 1.923) | Sub-exponential |
| Shor's algorithm | O((log n)^3) | Quantum |

```
L-notation: L(alpha, c) = exp( c * (ln n)^alpha * (ln ln n)^(1-alpha) )
  alpha = 0: polynomial
  alpha = 1: exponential
  0 < alpha < 1: sub-exponential
```

### Factoring Hardness and Key Sizes

| RSA Key Size | Security Level (bits) | Status (2026) |
|---|---|---|
| 1024 | ~80 | Deprecated, factorable by well-funded adversaries |
| 2048 | ~112 | Minimum recommended |
| 3072 | ~128 | NIST recommended through 2030 |
| 4096 | ~152 | Conservative choice |

## Discrete Logarithm Problem (DLP)

```
Given a cyclic group G = <g> of order n, and h in G:
  Find x such that g^x = h

Hardness depends on the group:
  Z/pZ*     — sub-exponential (index calculus): L(1/3, 1.923)
  Elliptic curves — no sub-exponential attack known: O(sqrt(n))
```

| DLP Algorithm | Group | Complexity |
|---|---|---|
| Baby-step giant-step | Any | O(sqrt(n)) time and space |
| Pollard's rho (DLP) | Any | O(sqrt(n)) time, O(1) space |
| Pohlig-Hellman | Any (smooth order) | O(sum sqrt(pi)) |
| Index calculus | Z/pZ* | L(1/3, c) |

## Diffie-Hellman Key Exchange

```
Public parameters: prime p, generator g of Z/pZ*

Alice                           Bob
  a <-- random in [1, p-2]       b <-- random in [1, p-2]
  A = g^a mod p  ----------->
            <-----------  B = g^b mod p
  s = B^a mod p                  s = A^b mod p
  (= g^(ab) mod p)              (= g^(ab) mod p)

Security relies on:
  CDH:  Given g, g^a, g^b, compute g^(ab)  (believed hard)
  DDH:  Distinguish (g^a, g^b, g^ab) from (g^a, g^b, g^c)
```

## RSA Cryptosystem

### Key Generation

```
1. Choose large primes p, q  (|p| ≈ |q| ≈ n/2 bits)
2. Compute N = p * q
3. Compute phi(N) = (p-1)(q-1)
4. Choose e with gcd(e, phi(N)) = 1  (common: e = 65537)
5. Compute d = e^(-1) mod phi(N)

Public key:  (N, e)
Private key: (N, d)   [or (p, q, d)]
```

### Encryption / Decryption

```
Encrypt:  c = m^e mod N       (m < N)
Decrypt:  m = c^d mod N

Correctness: c^d = m^(ed) = m^(1 + k*phi(N)) = m * (m^phi(N))^k ≡ m (mod N)
  (by Euler's theorem, when gcd(m, N) = 1)
```

### RSA Security Considerations

```
- Textbook RSA is NOT CPA-secure (deterministic)
- Use OAEP padding (PKCS#1 v2) for encryption
- Use PSS padding for signatures
- d must be large (Wiener's attack if d < N^0.25 / 3)
- p and q must not be close (Fermat factoring)
- p - 1 and q - 1 must have large prime factors (Pollard p-1)
- Never reuse N with different e values (common modulus attack)
```

## Finite Fields

### GF(p) -- Prime Fields

```
GF(p) = Z/pZ = {0, 1, 2, ..., p-1}
  Addition and multiplication modulo p
  Every nonzero element has a multiplicative inverse
  Multiplicative group GF(p)* is cyclic of order p-1
```

### GF(2^n) -- Binary Extension Fields

```
GF(2^n) = F2[x] / f(x)   where f(x) is irreducible of degree n
  Elements are polynomials of degree < n with binary coefficients
  Addition = XOR (coefficient-wise mod 2)
  Multiplication = polynomial multiply mod f(x)

Used in: AES (GF(2^8)), elliptic curves over binary fields, CRC
```

| Field | Order | Characteristic | Used In |
|---|---|---|---|
| GF(p), p prime | p | p | RSA, DH, DSA, ECDSA (prime curves) |
| GF(2^n) | 2^n | 2 | AES, binary ECC, CRC |
| GF(p^n), p odd | p^n | p | Pairing-based crypto |

## Quadratic Residues

```
a is a quadratic residue mod p  iff  exists x : x^2 ≡ a (mod p)

Euler's criterion:
  a^((p-1)/2) ≡  1 (mod p)  =>  a is a QR
  a^((p-1)/2) ≡ -1 (mod p)  =>  a is a QNR

Legendre symbol:  (a/p) = a^((p-1)/2) mod p   (= 0, 1, or -1)

Exactly (p-1)/2 nonzero elements of GF(p) are QRs.

Applications:
  - Goldwasser-Micali encryption (QR assumption)
  - Tonelli-Shanks algorithm for modular square roots
  - Quadratic reciprocity (Gauss): connects (p/q) and (q/p)
```

## Key Figures

| Name | Contribution | Era |
|---|---|---|
| Leonhard Euler | Totient function, Euler's theorem, quadratic reciprocity (partial) | 1736-1783 |
| Pierre de Fermat | Fermat's little theorem, Fermat primes | 1640 |
| Carl Friedrich Gauss | Quadratic reciprocity (Disquisitiones, 1801), congruence notation | 1801 |
| Ron Rivest | RSA cryptosystem (co-inventor) | 1977 |
| Adi Shamir | RSA cryptosystem (co-inventor) | 1977 |
| Leonard Adleman | RSA cryptosystem (co-inventor), number field sieve | 1977 |
| Whitfield Diffie | Public-key cryptography, Diffie-Hellman key exchange | 1976 |
| Martin Hellman | Diffie-Hellman key exchange | 1976 |
| Gary Miller | Miller-Rabin primality test (deterministic under GRH) | 1976 |
| Michael Rabin | Miller-Rabin randomized primality test | 1980 |
| Manindra Agrawal | AKS deterministic primality test (with Kayal, Saxena) | 2002 |

## Tips

- Always use cryptographically secure random number generators for key generation
- RSA key generation: ensure |p - q| is large to prevent Fermat factoring
- Miller-Rabin with k = 40 rounds gives error probability < 2^(-80) -- sufficient for most applications
- For modular exponentiation, use square-and-multiply (binary method)
- CRT speeds up RSA private-key operations by ~4x (compute mod p and mod q separately)
- The number field sieve is the fastest known classical factoring algorithm for large n
- Elliptic curve groups offer equivalent security at much smaller key sizes than Z/pZ*

## See Also

- `detail/cs-theory/number-theory-crypto.md` -- extended Euclidean algorithm, RSA correctness proof, hardness assumptions
- `sheets/cs-theory/complexity-theory.md` -- computational complexity classes, P vs NP
- `sheets/cs-theory/information-theory.md` -- entropy, information-theoretic security

## References

- "A Computational Introduction to Number Theory and Algebra" by Victor Shoup (Cambridge, 2009)
- "Introduction to Modern Cryptography" by Katz and Lindell (3rd ed., 2020)
- Rivest, Shamir, Adleman, "A Method for Obtaining Digital Signatures and Public-Key Cryptosystems" (CACM, 1978)
- Diffie and Hellman, "New Directions in Cryptography" (IEEE IT, 1976)
- Agrawal, Kayal, Saxena, "PRIMES is in P" (Annals of Mathematics, 2004)
- NIST SP 800-57 Part 1 Rev. 5 -- Recommendation for Key Management (2020)
- Menezes, Oorschot, Vanstone, "Handbook of Applied Cryptography" (CRC Press, 1996)
