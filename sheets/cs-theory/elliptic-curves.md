# Elliptic Curve Cryptography (Groups, Curves, and Discrete Logarithms)

A practitioner's reference for elliptic curve cryptography -- from the group law and scalar multiplication to ECDH, ECDSA, EdDSA, curve selection, and attacks.

## Elliptic Curve Definitions

### Weierstrass Form (General)

```
y^2 + a1*x*y + a3*y = x^3 + a2*x^2 + a4*x + a6

  - Defined over a field K (typically F_p or F_{2^m})
  - Must be non-singular (discriminant != 0)
```

### Short Weierstrass Form

```
y^2 = x^3 + a*x + b      (char(K) != 2, 3)

  - Discriminant: Delta = -16(4a^3 + 27b^2) != 0
  - Most common form in standards (P-256, secp256k1)

Example -- secp256k1:
  y^2 = x^3 + 7   (a = 0, b = 7, over F_p where p = 2^256 - 2^32 - 977)

Example -- P-256 (NIST):
  y^2 = x^3 - 3x + b   (a = -3, b = specific 256-bit constant)
```

### Montgomery Form

```
B*y^2 = x^3 + A*x^2 + x

  - Enables efficient x-coordinate-only arithmetic (Montgomery ladder)
  - Naturally constant-time: key for side-channel resistance
  - Example: Curve25519 (A = 486662, B = 1, p = 2^255 - 19)
```

### Edwards Form

```
x^2 + y^2 = 1 + d*x^2*y^2      (twisted Edwards: a*x^2 + y^2 = 1 + d*x^2*y^2)

  - Complete addition law: single formula for all point pairs (no special cases)
  - Example: Ed25519 uses the twisted Edwards curve
    -x^2 + y^2 = 1 - (121665/121666)*x^2*y^2  over F_{2^255 - 19}
```

## Group Law

### The Group (E(K), +)

```
Elements: points (x, y) on the curve, plus the point at infinity O
Identity: O (the point at infinity)
Inverse:  -(x, y) = (x, -y)      [short Weierstrass]
Order:    #E(F_p) = number of points including O

Closure:  P + Q is always a point on the curve (or O)
```

### Point Addition (P != Q)

```
Given P = (x1, y1), Q = (x2, y2), with P != Q:

  lambda = (y2 - y1) / (x2 - x1)        (slope of the secant line)
  x3 = lambda^2 - x1 - x2
  y3 = lambda * (x1 - x3) - y1

  Result: P + Q = (x3, y3)

Special case: if x1 = x2 and y1 = -y2, then P + Q = O
```

### Point Doubling (P = Q)

```
Given P = (x1, y1), compute 2P:

  lambda = (3*x1^2 + a) / (2*y1)         (slope of the tangent line)
  x3 = lambda^2 - 2*x1
  y3 = lambda * (x1 - x3) - y1

  Result: 2P = (x3, y3)

Special case: if y1 = 0, then 2P = O
```

## Scalar Multiplication

### Double-and-Add Algorithm

```
Input:  scalar k (n bits), point P
Output: Q = k*P

Algorithm (left-to-right binary):
  Q = O
  for i from n-1 down to 0:
      Q = 2*Q                    (double)
      if bit i of k is 1:
          Q = Q + P              (add)
  return Q

Cost: ~n doublings + ~n/2 additions on average
WARNING: naive implementation leaks k via timing (branch on secret bits)
```

### Montgomery Ladder (Constant-Time)

```
Input:  scalar k (n bits), point P (x-coordinate only)
Output: x(k*P)

  R0 = O, R1 = P
  for i from n-1 down to 0:
      if bit i of k is 0:
          R1 = R0 + R1
          R0 = 2*R0
      else:
          R0 = R0 + R1
          R1 = 2*R1
  return R0

  - Same operations per loop iteration regardless of bit value
  - Side-channel resistant: constant-time by construction
  - Used by Curve25519 / X25519
```

## The Elliptic Curve Discrete Logarithm Problem (ECDLP)

```
Given: points P and Q = k*P on E(F_p)
Find:  the scalar k

  - Best known classical algorithms: O(sqrt(n)) where n = ord(P)
  - No subexponential classical algorithm known (unlike factoring or finite-field DLP)
  - This is what makes ECC attractive: smaller keys for equivalent security

Security comparison (symmetric-equivalent bits):
  ECC key size   RSA key size    Security level
  160 bits       1024 bits       80 bits
  224 bits       2048 bits       112 bits
  256 bits       3072 bits       128 bits
  384 bits       7680 bits       192 bits
  521 bits       15360 bits      256 bits
```

## ECDH (Elliptic Curve Diffie-Hellman)

```
Setup: agree on curve E over F_p, generator G of order n

Alice:                          Bob:
  a <- random in [1, n-1]        b <- random in [1, n-1]
  A = a*G  -->                   B = b*G  -->
             <-- exchange -->
  shared = a*B = a*b*G          shared = b*A = b*a*G

  - Shared secret: x-coordinate of a*b*G (or hash thereof)
  - X25519: ECDH on Curve25519, x-coordinate-only, constant-time
  - Security: equivalent to the Computational Diffie-Hellman (CDH) problem on E
```

## ECDSA (Elliptic Curve Digital Signature Algorithm)

```
Parameters: curve E, generator G of order n, hash function H

Key generation:
  private key:  d <- random in [1, n-1]
  public key:   Q = d*G

Signing message m:
  1. e = H(m)                              (hash the message)
  2. k <- random in [1, n-1]               (per-signature nonce, MUST be unique)
  3. (x1, y1) = k*G
  4. r = x1 mod n                          (if r = 0, restart)
  5. s = k^{-1} * (e + r*d) mod n          (if s = 0, restart)
  Signature: (r, s)

Verification of (r, s) on message m:
  1. e = H(m)
  2. w = s^{-1} mod n
  3. u1 = e*w mod n,   u2 = r*w mod n
  4. (x1, y1) = u1*G + u2*Q
  5. Accept iff r = x1 mod n

CRITICAL: nonce k must NEVER be reused or predictable.
  - Reuse of k reveals private key d:
    d = (s1*k - e1) * r^{-1} mod n
  - Use RFC 6979 (deterministic k from private key + message hash)
```

## EdDSA / Ed25519

```
EdDSA: Schnorr-like signature scheme on twisted Edwards curves

Ed25519 parameters:
  - Curve: twisted Edwards over F_{2^255 - 19}
  - Base point: specific point of order L = 2^252 + 27742...
  - Hash: SHA-512
  - Cofactor: h = 8

Advantages over ECDSA:
  - Deterministic nonces (no random k needed)
  - Faster verification (batch-verifiable)
  - Complete addition formulas (no edge cases)
  - Constant-time by design
  - Single-pass signing (no need to hash message twice)

Ed448: similar scheme on Edwards448-Goldilocks curve (448-bit, 224-bit security)
```

## Standard Curves and Parameters

```
Curve          Form          Field size   Security   Cofactor   Use case
-----------    -----------   ----------   --------   --------   --------
secp256k1      Weierstrass   256 bits     128 bits   1          Bitcoin, Ethereum
P-256/prime256v1 Weierstrass 256 bits     128 bits   1          TLS, NIST standard
P-384          Weierstrass   384 bits     192 bits   1          Government, high security
P-521          Weierstrass   521 bits     256 bits   1          Maximum NIST security
Curve25519     Montgomery    255 bits     128 bits   8          X25519 key exchange
Ed25519        Edwards       255 bits     128 bits   8          Signatures (EdDSA)
Ed448          Edwards       448 bits     224 bits   4          High-security EdDSA
BN254          Weierstrass   254 bits     ~100 bits  1          Pairings, zkSNARKs
BLS12-381      Weierstrass   381 bits     128 bits   varies     Pairings, Ethereum 2.0
```

## Cofactor and Twist Security

### Cofactor

```
#E(F_p) = h * n     where n = prime order of the main subgroup, h = cofactor

  - h = 1: every point (except O) has prime order (simplest, safest)
  - h > 1: small-subgroup attacks possible if not mitigated
  - Mitigation: multiply received points by h (cofactor clearing)
  - Curve25519: h = 8, so always multiply by 8 or validate points

Ristretto: technique to build a prime-order group from a cofactor-8 curve
  - Eliminates cofactor-related pitfalls for Curve25519/Ed25519
```

### Twist Security

```
The quadratic twist E' of E over F_p:
  E:  y^2 = x^3 + ax + b
  E': y^2 = x^3 + a*u^2*x + b*u^3    (u a non-square in F_p)

  - #E(F_p) + #E'(F_p) = 2p + 2
  - Invalid-curve attacks send points on E' instead of E
  - Twist-secure: both E and E' have nearly-prime order
  - Curve25519 is twist-secure; many NIST curves are not
```

## Attacks on ECDLP

### Pohlig-Hellman Attack

```
If n = ord(P) = p1^e1 * p2^e2 * ... * pr^er:
  1. Solve ECDLP in each subgroup of order pi^ei
  2. Combine via CRT (Chinese Remainder Theorem)

Cost: O(sum sqrt(pi^ei)) instead of O(sqrt(n))
Defense: use curves with n prime (or nearly prime with large prime factor)
```

### Pollard's Rho for ECDLP

```
  - Birthday-paradox random walk on the group
  - Expected cost: O(sqrt(n)) group operations
  - Memory: O(1) with Floyd's cycle detection or distinguished points
  - Best generic attack; ~2^128 operations for a 256-bit curve
  - Parallelizable: r processors give O(sqrt(n)/sqrt(r)) speedup
```

### MOV Attack (Menezes-Okamoto-Vanstone)

```
  - Reduces ECDLP to DLP in F_{p^k} via the Weil or Tate pairing
  - k = embedding degree (smallest k such that n | p^k - 1)
  - If k is small, finite-field DLP is subexponential -> curve is weak
  - Defense: ensure k is large (k >= 20 for typical security levels)
  - Supersingular curves have small k (k <= 6); avoid for plain ECC
  - Pairing-friendly curves intentionally have moderate k (BN, BLS families)
```

## Pairing-Based Cryptography

```
Bilinear pairing: e: G1 x G2 -> GT
  - e(aP, bQ) = e(P, Q)^{ab}       (bilinearity)
  - e(P, Q) != 1 for generating P, Q (non-degeneracy)

Applications:
  - BLS signatures (short signatures, aggregatable)
  - Identity-based encryption (IBE)
  - Zero-knowledge proofs (zkSNARKs: Groth16, PLONK)
  - Attribute-based encryption

Curves: BN254, BLS12-381, BLS12-377
Tradeoff: pairing-friendly curves sacrifice some ECDLP hardness for pairing structure
```

## Key Figures

```
| Name              | Contribution                                              | Year |
|-------------------|-----------------------------------------------------------|------|
| Neal Koblitz       | Proposed ECC independently (elliptic curves for crypto)  | 1985 |
| Victor Miller      | Proposed ECC independently (same idea, same year)        | 1985 |
| Daniel Bernstein   | Curve25519, Ed25519, ChaCha20, constant-time design      | 2006 |
| Andrew Wiles       | Proved Fermat's Last Theorem (modularity of elliptic curves) | 1995 |
| Hendrik Lenstra    | Elliptic curve factoring method (ECM)                    | 1987 |
| Alfred Menezes     | MOV attack, co-authored SEC standards                    | 1993 |
| Scott Vanstone     | MOV attack, Certicom / SEC curve standardization         | 1993 |
| Tatsuaki Okamoto   | MOV attack co-author                                     | 1993 |
| Peter Shor         | Quantum algorithm that breaks ECDLP in polynomial time   | 1994 |
```

## Tips

- ECC gives equivalent security to RSA at much smaller key sizes: 256-bit ECC is roughly 3072-bit RSA.
- Always use constant-time implementations. Timing side channels on scalar multiplication leak the private key.
- Never reuse ECDSA nonces. A single nonce reuse reveals the private key (this broke the PS3 signing key in 2010).
- Use RFC 6979 for deterministic ECDSA nonces, or prefer EdDSA which is deterministic by design.
- For new protocols, prefer Curve25519 (X25519 for key exchange) and Ed25519 (for signatures) over NIST P-256.
- Validate all received public keys: check that they lie on the curve and in the correct subgroup.
- Cofactor matters: for Curve25519 (h=8), use cofactor clearing or the Ristretto abstraction.
- Supersingular curves are weak for plain ECC (small embedding degree) but useful for pairings.
- Post-quantum: ECC is broken by Shor's algorithm. Plan migration to lattice-based or hash-based schemes.

## See Also

- cryptography
- number-theory
- information-theory
- complexity-theory
- quantum-computing

## References

- Koblitz, N. "Elliptic Curve Cryptosystems" (1987), Mathematics of Computation
- Miller, V. "Use of Elliptic Curves in Cryptography" (1985), CRYPTO proceedings
- Bernstein, D. J. "Curve25519: New Diffie-Hellman Speed Records" (2006), PKC
- Bernstein, D. J. et al. "High-Speed High-Security Signatures" (2012) -- Ed25519 paper
- Hankerson, D., Menezes, A., Vanstone, S. "Guide to Elliptic Curve Cryptography" (Springer, 2004)
- NIST FIPS 186-5: Digital Signature Standard (DSS), 2023
- SEC 2: Recommended Elliptic Curve Domain Parameters (Certicom, 2010)
- RFC 7748: Elliptic Curves for Security (X25519, X448)
- RFC 8032: Edwards-Curve Digital Signature Algorithm (EdDSA)
- RFC 6979: Deterministic Usage of the Digital Signature Algorithm (DSA) and ECDSA
