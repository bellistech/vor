# Elliptic Curve Cryptography -- Groups, Curves, and the Discrete Logarithm Problem

> *Elliptic curves provide the richest source of finite abelian groups suitable for cryptography, achieving security equivalent to much larger algebraic structures through the apparent hardness of the elliptic curve discrete logarithm problem.*

---

## 1. The Group Law: Derivation via Chord and Tangent

### The Problem

Derive the abelian group structure on an elliptic curve from the geometric chord-and-tangent construction.

### Setup

An elliptic curve $E$ over a field $K$ (with $\text{char}(K) \neq 2, 3$) in short Weierstrass form is:

$$E: y^2 = x^3 + ax + b, \quad \Delta = -16(4a^3 + 27b^2) \neq 0$$

The set of rational points $E(K)$ includes all $(x, y) \in K^2$ satisfying the equation, plus a distinguished point at infinity $\mathcal{O}$.

### The Chord-and-Tangent Construction

By Bezout's theorem, a line $L$ in $\mathbb{P}^2$ intersects a cubic curve $E$ in exactly 3 points (counted with multiplicity, in the algebraic closure). The group law exploits this:

**Definition.** Given $P, Q \in E(K)$, draw the line $L$ through $P$ and $Q$ (or the tangent at $P$ if $P = Q$). Let $R$ be the third intersection point. Then $P + Q$ is defined as the reflection of $R$ across the $x$-axis:

$$P + Q = -R, \quad \text{where } -(x, y) = (x, -y)$$

The point at infinity $\mathcal{O}$ is the identity: it is the third intersection point of every vertical line $x = c$ with $E$.

### Point Addition Formulas (Affine Coordinates)

**Case 1: $P = (x_1, y_1)$, $Q = (x_2, y_2)$, $x_1 \neq x_2$** (addition)

The line through $P$ and $Q$ has slope:

$$\lambda = \frac{y_2 - y_1}{x_2 - x_1}$$

Substituting $y = \lambda(x - x_1) + y_1$ into $y^2 = x^3 + ax + b$ and using Vieta's formulas for the cubic in $x$:

$$x_3 = \lambda^2 - x_1 - x_2, \qquad y_3 = \lambda(x_1 - x_3) - y_1$$

**Derivation:** The substitution yields $[\lambda(x - x_1) + y_1]^2 = x^3 + ax + b$, which rearranges to $x^3 - \lambda^2 x^2 + \cdots = 0$. Since $x_1, x_2, x_3$ are the three roots, Vieta's gives $x_1 + x_2 + x_3 = \lambda^2$, so $x_3 = \lambda^2 - x_1 - x_2$. The $y$-coordinate follows from the line equation and negation.

**Case 2: $P = Q = (x_1, y_1)$, $y_1 \neq 0$** (doubling)

The tangent line at $P$ has slope obtained by implicit differentiation of $y^2 = x^3 + ax + b$:

$$\lambda = \frac{3x_1^2 + a}{2y_1}$$

The same Vieta argument gives:

$$x_3 = \lambda^2 - 2x_1, \qquad y_3 = \lambda(x_1 - x_3) - y_1$$

**Case 3:** If $x_1 = x_2$ and $y_1 = -y_2$ (or $y_1 = 0$ for doubling), then $P + Q = \mathcal{O}$.

### Proof that $(E(K), +)$ is an Abelian Group

1. **Closure:** By construction, $P + Q \in E(K) \cup \{\mathcal{O}\}$.
2. **Identity:** $P + \mathcal{O} = P$ for all $P$ (a vertical line through $P$ meets $E$ at $P$, $-P$, and $\mathcal{O}$; reflecting $-P$ gives $P$).
3. **Inverses:** $P + (-P) = \mathcal{O}$ by definition.
4. **Commutativity:** The line through $P$ and $Q$ is the same as the line through $Q$ and $P$.
5. **Associativity:** The hardest part. Proved by explicit (tedious) algebraic verification, or more elegantly via the theory of divisors on algebraic curves. The key identity is that the divisor class group $\text{Pic}^0(E)$ is isomorphic to $E(K)$ as a group.

---

## 2. Projective and Jacobian Coordinates

### The Problem

Affine addition requires a field inversion ($1/\lambda$) per operation, which is expensive (typically 20--80 times the cost of a multiplication). Projective coordinates eliminate inversions.

### Projective Coordinates (Homogeneous)

Represent $(x, y)$ as $(X : Y : Z)$ where $x = X/Z$, $y = Y/Z$. The curve equation becomes:

$$Y^2 Z = X^3 + aXZ^2 + bZ^3$$

The point at infinity is $\mathcal{O} = (0 : 1 : 0)$. Addition formulas involve only multiplications and additions in $K$. A single inversion is needed at the end to convert back to affine.

### Jacobian Coordinates

Represent $(x, y)$ as $(X : Y : Z)$ where $x = X/Z^2$, $y = Y/Z^3$. The curve equation:

$$Y^2 = X^3 + aXZ^4 + bZ^6$$

**Doubling cost (Jacobian):** For curves with $a = -3$ (like P-256), doubling takes $3M + 4S$ (multiplications + squarings), compared to $1I + 2M + 2S$ in affine.

**Mixed addition** (Jacobian + affine): When $Q$ is in affine form $(x, y, 1)$, the cost drops to $7M + 4S$. This is common in precomputed table methods.

### Operation Counts (Short Weierstrass, $a = -3$)

| Operation | Affine | Projective | Jacobian | Jacobian ($a=-3$) |
|-----------|--------|------------|----------|--------------------|
| Addition  | $1I + 2M + 1S$ | $12M + 2S$ | $11M + 5S$ | $11M + 5S$ |
| Doubling  | $1I + 2M + 2S$ | $5M + 6S$  | $1M + 8S$  | $3M + 4S$ |

Where $I$ = inversion, $M$ = multiplication, $S$ = squaring. Typically $I/M \approx 20$--$80$, $S/M \approx 0.8$.

---

## 3. Hasse's Theorem and Point Counting

### Hasse's Theorem

**Theorem (Hasse, 1933).** For an elliptic curve $E$ over $\mathbb{F}_p$:

$$|\\#E(\mathbb{F}_p) - (p + 1)| \leq 2\sqrt{p}$$

Equivalently, writing $\#E(\mathbb{F}_p) = p + 1 - t$ where $t$ is the trace of Frobenius:

$$|t| \leq 2\sqrt{p}$$

**Interpretation:** The number of points is approximately $p$, with an error bounded by $2\sqrt{p}$. For a 256-bit prime, $\#E(\mathbb{F}_p) \approx p \pm 2^{128}$.

**Proof sketch:** The Frobenius endomorphism $\phi: (x, y) \mapsto (x^p, y^p)$ satisfies the characteristic polynomial $\phi^2 - t\phi + p = 0$ in $\text{End}(E)$. The discriminant $t^2 - 4p < 0$ (since $\phi$ has no real eigenvalues), giving $|t| < 2\sqrt{p}$.

### Schoof's Algorithm

**Problem:** Compute $\#E(\mathbb{F}_p)$ exactly.

**Approach (Schoof, 1985):** Compute $t \bmod \ell$ for small primes $\ell$, then recover $t$ via CRT.

1. For each small prime $\ell$, compute the action of $\phi$ on the $\ell$-torsion subgroup $E[\ell]$.
2. The relation $\phi^2 - t\phi + p = 0$ on $E[\ell]$ becomes $\phi^2(P) + [p]P = [t]\phi(P)$ for $P \in E[\ell]$.
3. Test $t \equiv t_0 \pmod{\ell}$ for $t_0 \in \{0, 1, \ldots, \ell-1\}$ using division polynomials.
4. When $\prod \ell > 4\sqrt{p}$, recover $t$ uniquely via CRT.

**Complexity:** $O(\log^8 p)$ in the original version. The Schoof-Elkies-Atkin (SEA) improvement runs in $O(\log^4 p)$ expected time and is used in practice to count points on curves with primes of hundreds of digits.

---

## 4. Montgomery Ladder: Constant-Time Scalar Multiplication

### The Problem

Compute $kP$ in time independent of the bits of $k$, preventing timing and power analysis side channels.

### The Algorithm

The Montgomery ladder operates on $x$-coordinates only, maintaining the invariant that the two running points differ by exactly $P$:

```
Input:  k = (k_{n-1}, ..., k_1, k_0)_2, point P
Output: kP

R0 = O,  R1 = P
for i from n-1 down to 0:
    if k_i = 0:
        R1 = R0 + R1    (differential addition)
        R0 = 2 * R0
    else:
        R0 = R0 + R1    (differential addition)
        R1 = 2 * R1
return R0
```

**Key property:** At every iteration, $R_1 - R_0 = P$. This allows "differential addition" using only $x$-coordinates, since the difference is known.

### Constant-Time Implementation

The conditional swap can be implemented without branching:

$$\text{cswap}(k_i, R_0, R_1): \quad \text{mask} = -k_i; \quad \text{swap} = \text{mask} \wedge (R_0 \oplus R_1); \quad R_0 \mathrel{\oplus}= \text{swap}; \quad R_1 \mathrel{\oplus}= \text{swap}$$

This ensures identical instruction sequences for $k_i = 0$ and $k_i = 1$.

### Montgomery Curve Arithmetic

On $By^2 = x^3 + Ax^2 + x$, differential addition using only $x$-coordinates (represented as $(X : Z)$ with $x = X/Z$):

Given $x(P-Q)$, $x(P)$, $x(Q)$, compute $x(P+Q)$:

$$X_{P+Q} = Z_{P-Q} \cdot [(X_P - Z_P)(X_Q + Z_Q) + (X_P + Z_P)(X_Q - Z_Q)]^2$$
$$Z_{P+Q} = X_{P-Q} \cdot [(X_P - Z_P)(X_Q + Z_Q) - (X_P + Z_P)(X_Q - Z_Q)]^2$$

Doubling:

$$X_{2P} = (X_P + Z_P)^2 \cdot (X_P - Z_P)^2$$
$$Z_{2P} = (4X_P Z_P) \cdot [(X_P - Z_P)^2 + \tfrac{A+2}{4} \cdot 4X_P Z_P]$$

where $4X_P Z_P = (X_P + Z_P)^2 - (X_P - Z_P)^2$.

**Cost per bit:** $5M + 4S + 1 \times a_{24}$ where $a_{24} = (A+2)/4$. For Curve25519, $a_{24} = 121666$.

---

## 5. Edwards Curve Arithmetic

### Twisted Edwards Form

$$E_{a,d}: \quad ax^2 + y^2 = 1 + dx^2y^2, \quad a \neq d, \quad ad \neq 0$$

The identity is $(0, 1)$, and $-(x, y) = (-x, y)$.

### Unified Addition Formula

For $P_1 = (x_1, y_1)$ and $P_2 = (x_2, y_2)$:

$$x_3 = \frac{x_1 y_2 + x_2 y_1}{1 + d \, x_1 x_2 y_1 y_2}, \qquad y_3 = \frac{y_1 y_2 - a \, x_1 x_2}{1 - d \, x_1 x_2 y_1 y_2}$$

**Completeness:** When $a$ is a square and $d$ is a non-square in $K$, the denominators $1 \pm d \, x_1 x_2 y_1 y_2$ are never zero for points on $E_{a,d}(K)$. This means a single formula handles all cases: no special-casing for doubling, identity, or inverses.

### Extended Coordinates

Represent $(x, y)$ as $(X : Y : T : Z)$ with $x = X/Z$, $y = Y/Z$, $T = XY/Z$.

**Unified addition cost:** $8M + 1D$ (where $D$ = multiplication by the constant $d$). If $a = -1$: $8M + 1D$, or $7M + 1S + 1D$ for doubling.

### Birational Equivalence: Montgomery and Edwards

A Montgomery curve $Bv^2 = u^3 + Au^2 + u$ is birationally equivalent to the twisted Edwards curve $ax^2 + y^2 = 1 + dx^2y^2$ via:

$$u = \frac{1+y}{1-y}, \quad v = \frac{u}{x} = \frac{1+y}{x(1-y)}$$

with $a = (A+2)/B$, $d = (A-2)/B$.

Curve25519 ($A = 486662$, $B = 1$) corresponds to Ed25519 ($a = -1$, $d = -121665/121666$).

---

## 6. ECDSA Correctness Proof

### The Scheme

- **Parameters:** Curve $E/\mathbb{F}_p$, generator $G$ of prime order $n$, hash $H$.
- **Private key:** $d \in [1, n-1]$; **Public key:** $Q = dG$.
- **Sign:** Choose $k \in_R [1, n-1]$. Compute $R = kG$, $r = x_R \bmod n$, $s = k^{-1}(H(m) + rd) \bmod n$. Output $(r, s)$.
- **Verify:** Compute $w = s^{-1} \bmod n$, $u_1 = H(m) \cdot w \bmod n$, $u_2 = r \cdot w \bmod n$. Check $x(u_1 G + u_2 Q) \equiv r \pmod{n}$.

### Proof of Correctness

We must show that a validly generated signature always passes verification.

$$u_1 G + u_2 Q = u_1 G + u_2 (dG) = (u_1 + u_2 d) G$$

Substituting:

$$u_1 + u_2 d = H(m) \cdot s^{-1} + r \cdot s^{-1} \cdot d = s^{-1}(H(m) + rd)$$

From the signing equation, $s = k^{-1}(H(m) + rd)$, so $s^{-1} = k / (H(m) + rd)$. Thus:

$$u_1 + u_2 d = \frac{k}{H(m) + rd} \cdot (H(m) + rd) = k$$

Therefore $u_1 G + u_2 Q = kG = R$, and $x(u_1 G + u_2 Q) = x_R \equiv r \pmod{n}$.  $\square$

### Security Note: Nonce Reuse

If the same $k$ is used to sign two different messages $m_1, m_2$:

$$s_1 = k^{-1}(e_1 + rd), \quad s_2 = k^{-1}(e_2 + rd)$$

where $e_i = H(m_i)$. Then $s_1 - s_2 = k^{-1}(e_1 - e_2)$, giving:

$$k = (e_1 - e_2)(s_1 - s_2)^{-1} \bmod n$$

and the private key is immediately recovered:

$$d = r^{-1}(sk - e) \bmod n$$

---

## 7. The Weil Pairing and MOV Reduction

### The Weil Pairing

For an elliptic curve $E/\mathbb{F}_q$ and integer $n$ with $\gcd(n, q) = 1$, the Weil pairing is a bilinear map:

$$e_n: E[n] \times E[n] \to \mu_n \subset \overline{\mathbb{F}}_q^*$$

where $E[n] = \{P \in E(\overline{\mathbb{F}}_q) : nP = \mathcal{O}\}$ is the $n$-torsion subgroup, and $\mu_n$ is the group of $n$-th roots of unity.

**Properties:**
1. **Bilinearity:** $e_n(aP, bQ) = e_n(P, Q)^{ab}$
2. **Alternating:** $e_n(P, P) = 1$
3. **Non-degeneracy:** If $e_n(P, Q) = 1$ for all $Q \in E[n]$, then $P = \mathcal{O}$

### The MOV Attack

**Theorem (Menezes-Okamoto-Vanstone, 1993).** Let $P \in E(\mathbb{F}_q)$ have prime order $n$, and let $k$ be the embedding degree (smallest positive integer such that $n \mid q^k - 1$). Given $Q = mP$, the ECDLP can be reduced to the DLP in $\mathbb{F}_{q^k}^*$:

1. Find $R \in E(\mathbb{F}_{q^k})$ such that $e_n(P, R) \neq 1$ (a linearly independent $n$-torsion point).
2. Compute $\alpha = e_n(P, R) \in \mathbb{F}_{q^k}^*$ and $\beta = e_n(Q, R) = e_n(mP, R) = \alpha^m$.
3. Solve $\beta = \alpha^m$ in $\mathbb{F}_{q^k}^*$ using index calculus (subexponential in $q^k$).

**When it applies:** If $k$ is small (say $k \leq 6$), the DLP in $\mathbb{F}_{q^k}^*$ is much easier than the ECDLP. Supersingular curves always have $k \leq 6$. For randomly chosen curves, $k$ is typically enormous ($k \approx n$), making the reduction useless.

**Defense:** Use curves with large embedding degree. For standard curves (P-256, Curve25519), $k > 2^{200}$.

---

## 8. Curve Selection Criteria

### Security Requirements

1. **ECDLP hardness:** The group order $n$ must be prime (or have a large prime factor). This defeats the Pohlig-Hellman attack.
2. **MOV resistance:** The embedding degree $k$ must be large ($k > n / \log^2 n$ is more than sufficient).
3. **Anomalous curve resistance:** $\#E(\mathbb{F}_p) \neq p$ (otherwise Semaev-Smart-Satoh-Araki attack solves ECDLP in $O(\log p)$ time via $p$-adic lifting).
4. **Twist security:** The quadratic twist $E'$ should also have a large prime-order subgroup, protecting against invalid-curve attacks.

### Performance Requirements

5. **Efficient field arithmetic:** Special primes (Mersenne-like: $2^{255} - 19$, pseudo-Mersenne: $2^{256} - 2^{32} - 977$) enable fast modular reduction.
6. **Efficient curve arithmetic:** Small coefficients, special forms (Montgomery, Edwards) with fast formulas.
7. **Constant-time implementability:** Complete addition laws (Edwards) or $x$-only ladders (Montgomery).

### Rigidity and Verifiability

8. **Nothing-up-my-sleeve:** Curve parameters should be derived from a transparent, verifiable process. Criticism of NIST curves: the seed values used to generate P-256 are unexplained, raising concerns about potential backdoors.
9. **SafeCurves criteria** (Bernstein and Lange): a comprehensive checklist including all of the above plus: CM discriminant, rigidity, ladder support, twist security, transfer resistance.

### Comparison Table: RSA vs ECC Key Sizes

| Symmetric security | RSA modulus | ECC field size | Ratio (RSA/ECC) |
|--------------------|-------------|----------------|-----------------|
| 80 bits            | 1024 bits   | 160 bits       | 6.4x            |
| 112 bits           | 2048 bits   | 224 bits       | 9.1x            |
| 128 bits           | 3072 bits   | 256 bits       | 12x             |
| 192 bits           | 7680 bits   | 384 bits       | 20x             |
| 256 bits           | 15360 bits  | 521 bits       | 29.5x           |

The ratio grows with security level, making ECC increasingly advantageous at higher security requirements. At 256-bit security, an RSA key would be over 15,000 bits, while an ECC key is only 521 bits.

---

## 9. Curve Families: Weierstrass vs Montgomery vs Edwards

### Comparison

| Property | Short Weierstrass | Montgomery | Edwards (twisted) |
|----------|-------------------|------------|-------------------|
| Equation | $y^2 = x^3 + ax + b$ | $By^2 = x^3 + Ax^2 + x$ | $ax^2 + y^2 = 1 + dx^2y^2$ |
| Identity | Point at infinity $\mathcal{O}$ | Point at infinity | $(0, 1)$ |
| Inverse of $(x,y)$ | $(x, -y)$ | $(x, -y)$ | $(-x, y)$ |
| Addition formulas | 2 cases (add, double) | Differential only | 1 unified formula |
| Complete formulas | No | No | Yes (if $a = \square$, $d \neq \square$) |
| $x$-only ladder | Not natural | Yes (Montgomery ladder) | Possible but less natural |
| Covers all curves? | Yes (over $\text{char} \neq 2,3$) | No (needs point of order 2) | No (needs point of order 4) |

### Birational Equivalences

Not every Weierstrass curve has a Montgomery or Edwards form. The existence conditions:

- **Montgomery form** requires $E$ to have a point of order 2 over $K$, and the equation must factor appropriately.
- **Edwards form** requires $E$ to have a point of order 4 over $K$.
- Any Montgomery curve is birationally equivalent to a (twisted) Edwards curve, and vice versa.

For cryptographic purposes, starting from a Montgomery or Edwards curve and converting to Weierstrass for interoperability is common practice.

---

## Tips

- The chord-and-tangent construction is geometrically intuitive but algebraically fragile. Always verify formulas for edge cases ($P = \mathcal{O}$, $P = -Q$, $y = 0$).
- Jacobian coordinates with $a = -3$ give the fastest doubling for Weierstrass curves. P-256 was designed with $a = -3$ specifically for this reason.
- Hasse's bound is tight: for any $t$ with $|t| \leq 2\sqrt{p}$, there exists a curve $E/\mathbb{F}_p$ with $\#E(\mathbb{F}_p) = p + 1 - t$ (by the theory of complex multiplication).
- The Montgomery ladder naturally resists simple power analysis (SPA) because both branches perform the same operations. For differential power analysis (DPA), additional countermeasures (point blinding, scalar randomization) are still needed.
- The completeness of Edwards addition formulas does not automatically guarantee side-channel resistance. Implementations must still avoid variable-time modular arithmetic.
- For pairing-based cryptography, the embedding degree is a feature, not a bug. Pairing-friendly curve families (BN, BLS) are specifically designed to have manageable embedding degrees ($k = 12$ for BN254 and BLS12-381).
- The ECDSA correctness proof relies on the cancellation $s^{-1} \cdot s = 1 \bmod n$. When implementing, ensure modular arithmetic is exact -- rounding errors in floating point would be catastrophic.
- Schoof's algorithm is polynomial time, making it feasible to count points on curves over primes with thousands of bits. In practice, the SEA variant with early-abort strategies is used.

## See Also

- cryptography
- number-theory
- information-theory
- complexity-theory
- quantum-computing

## References

- Silverman, J. H. "The Arithmetic of Elliptic Curves" (2nd ed., Springer, 2009) -- the standard graduate text
- Washington, L. C. "Elliptic Curves: Number Theory and Cryptography" (2nd ed., CRC Press, 2008)
- Hankerson, D., Menezes, A., Vanstone, S. "Guide to Elliptic Curve Cryptography" (Springer, 2004)
- Bernstein, D. J. & Lange, T. "SafeCurves: choosing safe curves for elliptic-curve cryptography" (https://safecurves.cr.yp.to)
- Koblitz, N. "Elliptic Curve Cryptosystems" (1987), Mathematics of Computation, 48(177), 203--209
- Miller, V. "Use of Elliptic Curves in Cryptography" (1985), CRYPTO '85
- Menezes, A., Okamoto, T., Vanstone, S. "Reducing Elliptic Curve Logarithms to Logarithms in a Finite Field" (1993), IEEE Trans. Info. Theory
- Schoof, R. "Elliptic Curves Over Finite Fields and the Computation of Square Roots mod p" (1985), Mathematics of Computation
- Bernstein, D. J. "Curve25519: New Diffie-Hellman Speed Records" (2006), PKC 2006
- Bernstein, D. J. et al. "High-Speed High-Security Signatures" (2012), Journal of Cryptographic Engineering
- NIST FIPS 186-5: Digital Signature Standard (2023)
- RFC 7748: Elliptic Curves for Security (Langley, Hamburg, Turner, 2016)
- RFC 8032: Edwards-Curve Digital Signature Algorithm (EdDSA) (Josefsson, Liusvaara, 2017)
