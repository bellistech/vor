# The Arithmetic Engine of Cryptography -- Number Theory, Hardness Assumptions, and Public-Key Constructions

> *Modern public-key cryptography rests on a thin crust of number-theoretic assumptions: that certain arithmetic problems -- factoring, discrete logarithms, and their variants -- admit no efficient solution. This document develops the machinery from first principles, proves the core theorems, and maps the landscape of hardness that separates secure schemes from broken ones.*

---

## 1. The Extended Euclidean Algorithm

### The Problem

Given integers $a$ and $b$, compute $\gcd(a, b)$ and find integers $x, y$ such that $ax + by = \gcd(a, b)$ (Bezout's identity). This is the workhorse for computing modular inverses: if $\gcd(a, n) = 1$, then $x \bmod n$ is $a^{-1} \bmod n$.

### The Algorithm

Define sequences $r_i, s_i, t_i$ by:

$$r_0 = a, \quad r_1 = b, \quad s_0 = 1, \quad s_1 = 0, \quad t_0 = 0, \quad t_1 = 1$$

At each step, let $q_i = \lfloor r_{i-1} / r_i \rfloor$ and compute:

$$r_{i+1} = r_{i-1} - q_i \cdot r_i$$
$$s_{i+1} = s_{i-1} - q_i \cdot s_i$$
$$t_{i+1} = t_{i-1} - q_i \cdot t_i$$

Terminate when $r_{k+1} = 0$. Then $\gcd(a, b) = r_k$ and $a \cdot s_k + b \cdot t_k = r_k$.

### Worked Example

Compute $\gcd(240, 46)$ and find $x, y$ such that $240x + 46y = \gcd(240, 46)$.

| Step | $q_i$ | $r_i$ | $s_i$ | $t_i$ |
|------|--------|--------|--------|--------|
| 0 | -- | 240 | 1 | 0 |
| 1 | -- | 46 | 0 | 1 |
| 2 | 5 | 10 | 1 | -5 |
| 3 | 4 | 6 | -4 | 21 |
| 4 | 1 | 4 | 5 | -26 |
| 5 | 1 | 2 | -9 | 47 |
| 6 | 2 | 0 | -- | -- |

Result: $\gcd(240, 46) = 2$ and $240 \cdot (-9) + 46 \cdot 47 = -2160 + 2162 = 2$.

**Verification:** $-2160 + 2162 = 2$. Confirmed.

To find $46^{-1} \bmod 240$: since $\gcd(46, 240) = 2 \neq 1$, no inverse exists. But for $\gcd(7, 240) = 1$, we would obtain $7^{-1} \bmod 240 = 137$ since $7 \cdot 137 = 959 = 3 \cdot 240 + 239$... let us compute correctly: $7 \cdot 103 = 721 = 3 \cdot 240 + 1$, so $7^{-1} \equiv 103 \pmod{240}$.

### Complexity

The Euclidean algorithm terminates in at most $\lfloor \log_\phi(\max(a,b)) \rfloor$ steps, where $\phi = (1 + \sqrt{5})/2$ is the golden ratio. The worst case occurs for consecutive Fibonacci numbers. For $n$-bit integers, this gives $O(n)$ divisions, and the total cost with multi-precision arithmetic is $O(n^2)$ bit operations.

---

## 2. Chinese Remainder Theorem: Proof and Application to RSA-CRT

### Statement

Let $m_1, m_2, \ldots, m_k$ be pairwise coprime positive integers and let $M = \prod_{i=1}^k m_i$. Then the map

$$\mathbb{Z}/M\mathbb{Z} \to \mathbb{Z}/m_1\mathbb{Z} \times \mathbb{Z}/m_2\mathbb{Z} \times \cdots \times \mathbb{Z}/m_k\mathbb{Z}$$

defined by $x \mapsto (x \bmod m_1, x \bmod m_2, \ldots, x \bmod m_k)$ is a ring isomorphism.

### Proof

**Injectivity:** Suppose $x \equiv y \pmod{m_i}$ for all $i$. Then $m_i \mid (x - y)$ for all $i$. Since the $m_i$ are pairwise coprime, $M \mid (x - y)$, so $x \equiv y \pmod{M}$.

**Surjectivity:** Both sides have cardinality $M$, and an injective map between finite sets of equal cardinality is a bijection.

**Explicit construction:** Let $M_i = M / m_i$. Since $\gcd(M_i, m_i) = 1$, there exists $y_i$ with $M_i y_i \equiv 1 \pmod{m_i}$. Set:

$$x = \sum_{i=1}^k a_i M_i y_i$$

Then $x \equiv a_i M_i y_i \equiv a_i \pmod{m_i}$ since $M_j \equiv 0 \pmod{m_i}$ for $j \neq i$.

### Application: RSA-CRT

For RSA decryption with private key $d$ and modulus $N = pq$, the direct computation $m = c^d \bmod N$ requires exponentiation modulo an $n$-bit number. RSA-CRT computes instead:

$$m_p = c^{d_p} \bmod p, \quad m_q = c^{d_q} \bmod q$$

where $d_p = d \bmod (p-1)$ and $d_q = d \bmod (q-1)$. Then reconstruct $m \bmod N$ via CRT:

$$m = m_q + q \cdot \bigl(q^{-1} \cdot (m_p - m_q) \bmod p\bigr)$$

**Speedup:** Each exponentiation is modulo an $(n/2)$-bit number. Since modular exponentiation costs $O((\log n)^3)$, working modulo $p$ and $q$ separately gives roughly a $4\times$ speedup (two exponentiations each $8\times$ cheaper than one modulo $N$, then a cheap CRT recombination).

**Security note (Bellcore attack):** If a fault occurs during one of the two CRT exponentiations (say $m_p$ is correct but $m_q$ is faulty, yielding $\tilde{m}$), then:

$$\gcd(\tilde{m}^e - c, N) = p \quad \text{or} \quad q$$

This factors $N$. Countermeasure: verify $m^e \equiv c \pmod{N}$ before releasing the result.

---

## 3. Miller-Rabin Primality Test: Error Analysis

### Foundation

The Miller-Rabin test exploits a necessary condition for primality. Write $n - 1 = 2^s d$ with $d$ odd. If $n$ is prime and $\gcd(a, n) = 1$, then either:

$$a^d \equiv 1 \pmod{n}$$

or there exists $r \in \{0, 1, \ldots, s-1\}$ such that:

$$a^{2^r d} \equiv -1 \pmod{n}$$

A composite $n$ that satisfies one of these conditions for a given base $a$ is called a *strong pseudoprime* to base $a$, and $a$ is called a *strong liar* for $n$.

### Error Probability

**Theorem (Rabin, 1980):** For any odd composite $n$, the number of strong liars in $\{1, 2, \ldots, n-1\}$ is at most $\varphi(n)/4$.

**Corollary:** A single round of Miller-Rabin declares a composite number "probably prime" with probability at most $1/4$. After $k$ independent rounds with random bases, the error probability is at most:

$$\Pr[\text{false positive}] \leq 4^{-k}$$

**Practical bounds (Damgard-Landrock-Pomerance, 1993):** For random odd $n$ of $t$ bits, the probability that $k$ rounds of Miller-Rabin declares $n$ prime when it is composite is much smaller than $4^{-k}$. For $t \geq 600$ and $k \geq 1$:

$$p_{t,k} \leq 2^{-k} \cdot 2^{-2t/3}$$

This means that for cryptographic-size numbers (1024+ bits), even a few rounds provide overwhelming confidence.

### Deterministic Variants

Under the Generalized Riemann Hypothesis (GRH), it suffices to test all bases $a \leq 2 (\ln n)^2$ (Miller, 1976). For specific small bounds:

- $n < 3.3 \times 10^{24}$: bases $\{2, 3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37\}$ suffice.
- $n < 3.2 \times 10^{18}$: bases $\{2, 3, 5, 7, 11, 13, 17, 19, 23\}$ suffice.

---

## 4. RSA Correctness Proof

### Theorem

Let $N = pq$ with $p, q$ distinct primes. Let $e, d$ be positive integers with $ed \equiv 1 \pmod{\lambda(N)}$, where $\lambda(N) = \text{lcm}(p-1, q-1)$. Then for all $m \in \mathbb{Z}/N\mathbb{Z}$:

$$m^{ed} \equiv m \pmod{N}$$

### Proof

Write $ed = 1 + k \cdot \lambda(N)$ for some non-negative integer $k$. We must show $m^{ed} \equiv m \pmod{p}$ and $m^{ed} \equiv m \pmod{q}$; the result follows by CRT since $\gcd(p, q) = 1$.

**Case 1:** $p \mid m$. Then $m \equiv 0 \pmod{p}$ and $m^{ed} \equiv 0 \pmod{p}$. Done.

**Case 2:** $p \nmid m$. By Fermat's little theorem, $m^{p-1} \equiv 1 \pmod{p}$. Since $(p-1) \mid \lambda(N)$, write $\lambda(N) = (p-1) \cdot j$. Then:

$$m^{ed} = m^{1 + k\lambda(N)} = m \cdot (m^{p-1})^{kj} \equiv m \cdot 1^{kj} \equiv m \pmod{p}$$

By symmetry, $m^{ed} \equiv m \pmod{q}$.

By CRT, $m^{ed} \equiv m \pmod{N}$.

**Note:** Using $\lambda(N) = \text{lcm}(p-1, q-1)$ instead of $\varphi(N) = (p-1)(q-1)$ gives a smaller (and hence more efficient) private exponent $d$. Both work for correctness since $(p-1) \mid \varphi(N)$ and $(p-1) \mid \lambda(N)$.

---

## 5. Hardness Assumptions

The security of public-key cryptography rests on the assumed computational intractability of specific problems. These assumptions form a hierarchy.

### The Factoring Assumption

**FACTOR:** Given $N = pq$ for randomly chosen $n/2$-bit primes $p, q$, find $p$ and $q$.

No polynomial-time classical algorithm is known. The best known classical algorithm is the General Number Field Sieve with heuristic complexity:

$$L_N\left[\frac{1}{3}, \left(\frac{64}{9}\right)^{1/3}\right] \approx L_N\left[\frac{1}{3}, 1.923\right]$$

where $L_N[\alpha, c] = \exp\bigl(c \cdot (\ln N)^\alpha \cdot (\ln \ln N)^{1-\alpha}\bigr)$.

Shor's quantum algorithm solves FACTOR in $O((\log N)^3)$ on a quantum computer.

### The RSA Assumption

**RSA:** Given $(N, e)$ and $c = m^e \bmod N$ for random $m \in \mathbb{Z}_N^*$, find $m$.

The RSA assumption is *implied by* the factoring assumption (factoring lets you compute $d$), but it is not known whether RSA is *equivalent to* factoring. There exists an oracle relative to which RSA is easy but factoring is hard (Boneh-Venkatesan, 1998).

### The Discrete Logarithm Assumption

**DLP:** Given a cyclic group $G = \langle g \rangle$ of prime order $q$, and $h = g^x$ for random $x \in \mathbb{Z}_q$, find $x$.

Hardness depends on the group:
- In $(\mathbb{Z}/p\mathbb{Z})^*$: sub-exponential via index calculus, $L_p[1/3, 1.923]$.
- In elliptic curve groups $E(\mathbb{F}_p)$: best known attack is $O(\sqrt{q})$ (Pollard rho).

### The Computational Diffie-Hellman Assumption (CDH)

**CDH:** Given $g, g^a, g^b$ in a cyclic group $G = \langle g \rangle$, compute $g^{ab}$.

$$\text{DLP} \Rightarrow \text{CDH}$$

(Solving DLP lets you recover $a$, then compute $g^{ab} = (g^b)^a$.) The converse is not known in general, though it holds in certain groups (Maurer, 1994; den Boer, 1988, for groups where computing square roots is easy).

### The Decisional Diffie-Hellman Assumption (DDH)

**DDH:** Given $(g, g^a, g^b, Z)$ in $G$, distinguish whether $Z = g^{ab}$ or $Z$ is a random group element.

$$\text{CDH} \Rightarrow \text{DDH}$$

DDH is *strictly weaker* than CDH in some groups. For example, DDH fails in $(\mathbb{Z}/p\mathbb{Z})^*$ where $p - 1$ has small factors, because the Legendre symbol leaks whether $Z$ is a quadratic residue. DDH is believed to hold in prime-order subgroups of $(\mathbb{Z}/p\mathbb{Z})^*$ and in elliptic curve groups.

### Assumption Hierarchy

```
Factoring ------> RSA (not known if equivalent)

DLP  ===>  CDH  ===>  DDH
 (each strictly weaker going right, under current knowledge)
```

---

## 6. Index Calculus Method

### Overview

Index calculus is the most powerful technique for computing discrete logarithms in $(\mathbb{Z}/p\mathbb{Z})^*$. It has no analog in generic groups or elliptic curve groups, which is why ECC offers superior security per bit.

### Algorithm

Let $g$ be a generator of $(\mathbb{Z}/p\mathbb{Z})^*$ and let $h = g^x$. We want to find $x$.

**Step 1 (Factor base).** Choose a smoothness bound $B$ and let $\mathcal{F} = \{p_1, p_2, \ldots, p_k\}$ be all primes $\leq B$.

**Step 2 (Relation collection).** For random $r$, compute $g^r \bmod p$. If the result is $B$-smooth (all prime factors $\leq B$), record the relation:

$$g^r \equiv \prod_{i=1}^k p_i^{e_i} \pmod{p}$$

Taking discrete logs: $r \equiv \sum_{i=1}^k e_i \cdot \log_g p_i \pmod{p-1}$.

Collect slightly more than $k$ such relations.

**Step 3 (Linear algebra).** Solve the system of linear equations modulo $p - 1$ to recover $\log_g p_i$ for each $p_i \in \mathcal{F}$.

**Step 4 (Individual logarithm).** For random $s$, check if $h \cdot g^s \bmod p$ is $B$-smooth. If so:

$$h \cdot g^s \equiv \prod p_i^{f_i} \pmod{p}$$

$$x = \log_g h \equiv \sum f_i \cdot \log_g p_i - s \pmod{p-1}$$

### Complexity

With optimal choice of $B$, the complexity is $L_p[1/2, 1]$ for the basic version, and $L_p[1/3, (64/9)^{1/3}]$ for the number field sieve variant (same as factoring).

### Why Index Calculus Fails for Elliptic Curves

Index calculus requires a notion of "smoothness" -- decomposing group elements into a product of "small" elements from a factor base. In $(\mathbb{Z}/p\mathbb{Z})^*$, integers have a natural factorization into primes. Elliptic curve points have no analogous decomposition: a point $P \in E(\mathbb{F}_p)$ cannot be meaningfully written as a "product" of "small" points. This is the fundamental reason elliptic curve discrete logarithms resist sub-exponential attacks.

---

## 7. Quadratic Reciprocity

### The Legendre Symbol

For odd prime $p$ and integer $a$ with $p \nmid a$:

$$\left(\frac{a}{p}\right) = \begin{cases} 1 & \text{if } a \text{ is a quadratic residue mod } p \\ -1 & \text{if } a \text{ is a quadratic non-residue mod } p \end{cases}$$

Computed via Euler's criterion: $\left(\frac{a}{p}\right) \equiv a^{(p-1)/2} \pmod{p}$.

### The Law of Quadratic Reciprocity (Gauss, 1796)

For distinct odd primes $p$ and $q$:

$$\left(\frac{p}{q}\right) \left(\frac{q}{p}\right) = (-1)^{\frac{p-1}{2} \cdot \frac{q-1}{2}}$$

Equivalently: $\left(\frac{p}{q}\right) = \left(\frac{q}{p}\right)$ unless $p \equiv q \equiv 3 \pmod{4}$, in which case $\left(\frac{p}{q}\right) = -\left(\frac{q}{p}\right)$.

### Supplementary Laws

$$\left(\frac{-1}{p}\right) = (-1)^{(p-1)/2} = \begin{cases} 1 & p \equiv 1 \pmod{4} \\ -1 & p \equiv 3 \pmod{4} \end{cases}$$

$$\left(\frac{2}{p}\right) = (-1)^{(p^2-1)/8} = \begin{cases} 1 & p \equiv \pm 1 \pmod{8} \\ -1 & p \equiv \pm 3 \pmod{8} \end{cases}$$

### The Jacobi Symbol

The Jacobi symbol generalizes the Legendre symbol to composite moduli. For odd $n = p_1^{e_1} \cdots p_k^{e_k}$:

$$\left(\frac{a}{n}\right) = \prod_{i=1}^k \left(\frac{a}{p_i}\right)^{e_i}$$

The Jacobi symbol can be computed in $O(\log^2 n)$ time using reciprocity, without factoring $n$. However, $\left(\frac{a}{n}\right) = 1$ does not imply $a$ is a QR mod $n$ when $n$ is composite.

### Cryptographic Applications

- **Solovay-Strassen primality test:** If $n$ is prime, then $\left(\frac{a}{n}\right) \equiv a^{(n-1)/2} \pmod{n}$ for all $a$. An Euler pseudoprime is a composite satisfying this for some base $a$.
- **Goldwasser-Micali encryption:** Based on the quadratic residuosity assumption: given $N = pq$ and $z \in \mathbb{Z}_N^*$ with $\left(\frac{z}{N}\right) = 1$, it is hard to decide whether $z$ is a QR mod $N$.
- **Blum integers:** $N = pq$ with $p \equiv q \equiv 3 \pmod{4}$. Every QR mod $N$ has exactly four square roots, and exactly one is itself a QR. This enables the Blum-Blum-Shub pseudorandom generator.

---

## 8. Finite Field Arithmetic

### Prime Fields $\mathbb{F}_p$

Arithmetic in $\mathbb{F}_p = \mathbb{Z}/p\mathbb{Z}$:

| Operation | Method | Cost (bit ops for $n$-bit $p$) |
|-----------|--------|-------------------------------|
| Addition | Integer add, reduce mod $p$ | $O(n)$ |
| Subtraction | Integer subtract, reduce | $O(n)$ |
| Multiplication | Integer multiply, reduce mod $p$ | $O(n^2)$ naive, $O(n \log n \log \log n)$ FFT |
| Inversion | Extended Euclidean algorithm | $O(n^2)$ |
| Exponentiation | Square-and-multiply | $O(n)$ multiplications $= O(n^3)$ naive |

**Montgomery multiplication:** Replaces expensive division by $p$ with division by a power of 2 (which is a bit shift). Convert to Montgomery form $\tilde{a} = a \cdot R \bmod p$ where $R = 2^n$. Multiplication in Montgomery form avoids trial division by $p$, replacing it with:

$$\text{MonPro}(\tilde{a}, \tilde{b}) = \tilde{a} \cdot \tilde{b} \cdot R^{-1} \bmod p$$

computed without division by using a precomputed $p' = -p^{-1} \bmod R$.

### Extension Fields $\mathbb{F}_{2^n}$

Elements are polynomials $a(x) = a_{n-1}x^{n-1} + \cdots + a_1 x + a_0$ with $a_i \in \{0, 1\}$, represented as $n$-bit strings.

| Operation | Method | Cost |
|-----------|--------|------|
| Addition | Bitwise XOR | $O(n)$ |
| Multiplication | Polynomial multiply mod $f(x)$ | $O(n^2)$ schoolbook, $O(n \log n \log \log n)$ via FFT |
| Squaring | $a(x)^2 = a(x^2)$ (insert zeros), reduce | $O(n)$ with precomputation |
| Inversion | Extended Euclidean or Itoh-Tsujii | $O(n^2)$ or $O(\log n)$ multiplications |
| Frobenius $\phi: a \mapsto a^2$ | Squaring (a linear map in $\mathbb{F}_{2^n}$) | $O(n)$ |

**Choosing irreducible polynomials:** For efficiency, use trinomials $x^n + x^k + 1$ or pentanomials. For AES, $\mathbb{F}_{2^8}$ uses the irreducible polynomial $x^8 + x^4 + x^3 + x + 1$.

**Itoh-Tsujii inversion:** Computes $a^{-1} = a^{2^n - 2}$ using the identity:

$$a^{-1} = a^{2^n - 2} = \left(a^{2^{n-1} - 1}\right)^2$$

By expressing $2^{n-1} - 1$ in a binary addition chain, this requires only $O(\log n)$ multiplications and $O(n)$ squarings (which are cheap in characteristic 2).

---

## Prerequisites

- Modular arithmetic fundamentals (division with remainder, congruences)
- Group theory basics (cyclic groups, order of an element, Lagrange's theorem)
- Elementary probability (for probabilistic primality testing)
- Linear algebra over finite fields (for index calculus)
- Asymptotic notation and algorithm analysis

## Complexity

| Level | Description |
|-------|-------------|
| **Beginner** | Compute gcd via Euclidean algorithm. Apply Fermat's little theorem. Perform RSA encryption/decryption on small examples. State the CRT and apply it to solve simultaneous congruences. |
| **Intermediate** | Prove RSA correctness via Euler's theorem. Analyze Miller-Rabin error bounds. Implement extended Euclidean algorithm. Explain the CDH/DDH hierarchy. Compute in $\mathbb{F}_{2^8}$ (AES field). Perform RSA-CRT decryption. |
| **Advanced** | Describe the number field sieve at the algorithmic level. Explain why index calculus fails for elliptic curves. Prove quadratic reciprocity (via Gauss sums or Eisenstein's proof). Analyze Bellcore fault attacks on RSA-CRT. Construct reductions between hardness assumptions. Implement Montgomery multiplication. |
