# The Mathematics of Lattice-Based Cryptography -- From Geometry to Post-Quantum Standards

> *Lattice-based cryptography derives its security from the geometric hardness of finding short vectors in high-dimensional lattices -- problems that remain intractable even for quantum computers, making lattices the foundation of the post-quantum cryptographic transition.*

---

## 1. Lattice Geometry and Minkowski's Theorem

### The Problem

Establish the geometric foundations of lattices: what they are, how to measure them, and what fundamental bounds govern their structure.

### The Formula

A **lattice** $L \subset \mathbb{R}^n$ is a discrete additive subgroup. Given a basis $B = \{b_1, b_2, \ldots, b_n\}$ of linearly independent vectors in $\mathbb{R}^n$:

$$L(B) = \left\{ \sum_{i=1}^{n} z_i b_i : z_i \in \mathbb{Z} \right\}$$

The **determinant** (or volume) of the lattice is $\det(L) = |\det(B)|$, which is invariant across all bases of $L$. Two bases $B$ and $B'$ generate the same lattice if and only if $B' = UB$ for some unimodular matrix $U \in \mathbb{Z}^{n \times n}$ with $\det(U) = \pm 1$.

The **successive minima** $\lambda_1, \lambda_2, \ldots, \lambda_n$ of $L$ are defined as:

$$\lambda_i(L) = \min \{ r : \dim(\text{span}(L \cap \overline{B}(0, r))) \geq i \}$$

where $\overline{B}(0, r)$ is the closed ball of radius $r$.

**Minkowski's First Theorem:** For any $n$-dimensional lattice $L$:

$$\lambda_1(L) \leq \sqrt{n} \cdot \det(L)^{1/n}$$

More precisely, using the volume of the unit ball $V_n = \pi^{n/2} / \Gamma(n/2 + 1)$:

$$\lambda_1(L) \leq 2 \cdot \left( \frac{\det(L)}{V_n} \right)^{1/n}$$

**Minkowski's Second Theorem** extends this to all successive minima:

$$\left( \prod_{i=1}^{n} \lambda_i(L) \right)^{1/n} \leq \sqrt{n} \cdot \det(L)^{1/n}$$

The **Gaussian heuristic** predicts that for a "random" lattice:

$$\lambda_1(L) \approx \sqrt{\frac{n}{2\pi e}} \cdot \det(L)^{1/n}$$

### Worked Example

Consider the 2-dimensional lattice with basis $b_1 = (3, 1)$, $b_2 = (1, 3)$:

$$\det(L) = \left|\det \begin{pmatrix} 3 & 1 \\ 1 & 3 \end{pmatrix}\right| = |9 - 1| = 8$$

Minkowski bound: $\lambda_1 \leq \sqrt{2} \cdot 8^{1/2} = \sqrt{2} \cdot 2\sqrt{2} = 4$.

The actual shortest vector is $b_1 - b_2 = (2, -2)$ with $\|b_1 - b_2\| = 2\sqrt{2} \approx 2.83$, well within the bound.

### Why It Matters

Minkowski's theorem guarantees that every lattice contains short vectors, but finding them computationally is the hard problem that lattice cryptography exploits. The gap between the guaranteed existence (Minkowski) and the computational difficulty (NP-hardness of SVP) is the foundation of all lattice-based security.

---

## 2. The LLL Algorithm

### The Problem

Given an arbitrary basis for a lattice, find a "reduced" basis with short, nearly orthogonal vectors in polynomial time. This is the fundamental algorithmic tool for lattice problems.

### The Formula

Let $b_1^*, b_2^*, \ldots, b_n^*$ denote the Gram-Schmidt orthogonalization of $b_1, \ldots, b_n$ and let:

$$\mu_{i,j} = \frac{\langle b_i, b_j^* \rangle}{\langle b_j^*, b_j^* \rangle}$$

A basis is **LLL-reduced** (with parameter $\delta \in (1/4, 1]$, typically $\delta = 3/4$) if:

1. **Size reduction:** $|\mu_{i,j}| \leq 1/2$ for all $1 \leq j < i \leq n$
2. **Lovasz condition:** $\|b_i^*\|^2 \geq (\delta - \mu_{i,i-1}^2) \|b_{i-1}^*\|^2$ for all $2 \leq i \leq n$

The **LLL algorithm:**

```
Input: Basis B = {b1, ..., bn}, parameter delta
1. Compute Gram-Schmidt: b1*, ..., bn* and mu_{i,j}
2. k = 2
3. While k <= n:
   a. Size-reduce b_k:
      For j = k-1, k-2, ..., 1:
        b_k = b_k - round(mu_{k,j}) * b_j
        Update mu values
   b. If Lovasz condition holds for index k:
        k = k + 1
      Else:
        Swap b_k and b_{k-1}
        Update Gram-Schmidt
        k = max(k-1, 2)
4. Output reduced basis
```

**Output guarantees** for an LLL-reduced basis:

$$\|b_1\| \leq 2^{(n-1)/4} \cdot \det(L)^{1/n}$$

$$\|b_1\| \leq 2^{(n-1)/2} \cdot \lambda_1(L)$$

**Running time:** $O(n^5 d \log^3 M)$ where $d$ is the dimension and $M$ bounds the basis entries. Polynomial in all parameters.

### Worked Example

Starting basis: $b_1 = (1, 1, 1)$, $b_2 = (-1, 0, 2)$, $b_3 = (3, 5, 6)$.

**Step 1: Gram-Schmidt.**

$b_1^* = (1, 1, 1)$, $\|b_1^*\|^2 = 3$.

$\mu_{2,1} = \langle(-1, 0, 2), (1,1,1)\rangle / 3 = 1/3$.
$b_2^* = (-1, 0, 2) - (1/3)(1, 1, 1) = (-4/3, -1/3, 5/3)$, $\|b_2^*\|^2 = 42/9 = 14/3$.

$\mu_{3,1} = \langle(3,5,6), (1,1,1)\rangle / 3 = 14/3$.
$\mu_{3,2} = \langle(3,5,6), (-4/3, -1/3, 5/3)\rangle / (14/3) = (-12/3 - 5/3 + 30/3)/(14/3) = (13/3)/(14/3) = 13/14$.
$b_3^* = (3,5,6) - (14/3)(1,1,1) - (13/14)(-4/3, -1/3, 5/3) = \ldots$

**Step 2: Size reduction.** $|\mu_{2,1}| = 1/3 \leq 1/2$. OK.
$\mu_{3,1} = 14/3 > 1/2$. Size-reduce: $b_3 \leftarrow b_3 - \text{round}(14/3) \cdot b_1 = (3,5,6) - 5(1,1,1) = (-2, 0, 1)$.

Recompute $\mu_{3,1} = \langle(-2,0,1),(1,1,1)\rangle/3 = -1/3$. OK.
$\mu_{3,2} = \langle(-2,0,1),(-4/3,-1/3,5/3)\rangle/(14/3) = (8/3 + 0 + 5/3)/(14/3) = (13/3)/(14/3) = 13/14$. Round to 1.
$b_3 \leftarrow (-2,0,1) - 1 \cdot (-1,0,2) = (-1, 0, -1)$.

**Step 3: Check Lovasz.** After reduction, verify each pair and swap if needed. The final reduced basis has shorter, more orthogonal vectors than the input.

### Why It Matters

LLL runs in polynomial time but only achieves an exponential approximation factor $2^{O(n)}$ for SVP. Cryptographic security relies on the gap between polynomial-time algorithms (like LLL) and the exponential time needed for better approximations. If a scheme requires finding vectors shorter than what LLL can guarantee, it remains secure.

---

## 3. Regev's LWE Encryption Scheme

### The Problem

Construct a public-key encryption scheme whose security provably reduces to worst-case lattice problems, achieving the first practical connection between average-case cryptography and worst-case hardness.

### The Formula

**Parameters:** Security parameter $n$. Choose prime $q \geq n^2$, dimension $m = (1 + \epsilon) n \log q$ for small $\epsilon > 0$, and error parameter $\alpha \in (0, 1)$ with $\alpha q > 2\sqrt{n}$.

**Key Generation:**

1. Sample $A \xleftarrow{\$} \mathbb{Z}_q^{m \times n}$
2. Sample $s \xleftarrow{\$} \mathbb{Z}_q^n$
3. Sample $e \leftarrow D_{\mathbb{Z}, \alpha q}^m$ (discrete Gaussian, each coordinate)
4. Compute $b = As + e \pmod{q}$
5. Public key: $(A, b)$. Secret key: $s$.

**Encryption** of bit $\mu \in \{0, 1\}$:

1. Choose $x \xleftarrow{\$} \{0, 1\}^m$ (random binary vector)
2. $c_1 = A^T x \pmod{q} \in \mathbb{Z}_q^n$
3. $c_2 = b^T x + \mu \lfloor q/2 \rfloor \pmod{q} \in \mathbb{Z}_q$

**Decryption** of $(c_1, c_2)$:

1. Compute $v = c_2 - s^T c_1 \pmod{q}$
2. Output $0$ if $|v| < q/4$, output $1$ otherwise

### Correctness Proof

We verify that decryption recovers $\mu$:

$$v = c_2 - s^T c_1 = b^T x + \mu \lfloor q/2 \rfloor - s^T A^T x$$

$$= (As + e)^T x + \mu \lfloor q/2 \rfloor - s^T A^T x$$

$$= s^T A^T x + e^T x + \mu \lfloor q/2 \rfloor - s^T A^T x$$

$$= e^T x + \mu \lfloor q/2 \rfloor$$

The term $e^T x$ is the "noise." Since $e$ has entries of magnitude roughly $\alpha q$ and $x \in \{0,1\}^m$, we have $|e^T x| \leq \|e\|_1$. By concentration bounds on the discrete Gaussian and the choice of parameters:

$$|e^T x| \leq m \cdot \alpha q \cdot O(1) \ll q/4$$

with overwhelming probability. Therefore:

- If $\mu = 0$: $v = e^T x$, which is close to $0 \pmod{q}$, so $|v| < q/4$.
- If $\mu = 1$: $v = e^T x + \lfloor q/2 \rfloor$, which is close to $q/2 \pmod{q}$, so $|v| > q/4$.

Decryption succeeds with overwhelming probability.

### Why It Matters

Regev proved that breaking Decision-LWE (and hence this scheme) is at least as hard as solving $\widetilde{O}(n/\alpha)$-approximate versions of worst-case GapSVP and SIVP. This was the first public-key scheme with such a reduction and established LWE as the central hardness assumption in lattice cryptography.

---

## 4. Worst-Case Hardness Reductions

### The Problem

Show that the average-case hardness of LWE (and thus the security of cryptographic constructions) follows from the worst-case hardness of standard lattice problems.

### The Formula

**Ajtai's Reduction (1996):** For a uniformly random matrix $A \in \mathbb{Z}_q^{m \times n}$, finding a short nonzero vector $x$ such that $Ax = 0 \pmod{q}$ (the Short Integer Solution problem, SIS) is at least as hard as solving $\gamma$-approximate SVP in the worst case, for $\gamma = \text{poly}(n)$.

Formally, for any PPT algorithm $\mathcal{A}$ that solves random SIS instances with non-negligible probability, there exists a PPT algorithm $\mathcal{B}$ that solves worst-case $\text{GapSVP}_\gamma$ on any $n$-dimensional lattice.

**Regev's Reduction (2005):** The chain of reductions:

$$\text{worst-case GapSVP}_\gamma \leq_Q \text{BDD}_\alpha \leq \text{Decision-LWE}_{n,q,\alpha}$$

where $\leq_Q$ denotes a quantum reduction and BDD is Bounded Distance Decoding.

The quantum step: Given a GapSVP oracle target, Regev constructs a quantum algorithm that samples from discrete Gaussian distributions $D_{L,r}$ on lattice $L$ for certain widths $r$. These samples are then used to generate valid LWE instances.

**Peikert's Classical Reduction (2009):** Replaces the quantum step with a classical reduction at the cost of a lossy mode. Shows:

$$\text{worst-case GapSVP}_\gamma \leq \text{Decision-LWE}_{n,q,\alpha}$$

under a slightly stronger parameterization ($q = p^2$ for some integer $p$, enabling a "lossy mode" argument).

### The Reduction Structure

```
Worst-case lattice problem (e.g., GapSVP on ANY lattice)
   |
   | (reduction: if we could solve the easy problem,
   |  we could solve the hard problem)
   v
Average-case problem (LWE with random A, s, e)
   |
   | (construction)
   v
Cryptographic scheme (encryption, signatures, FHE)
```

### Why It Matters

Worst-case to average-case reductions are unique to lattice cryptography. In RSA or discrete-log systems, we assume specific instances are hard without any connection to worst-case complexity. Lattice schemes inherit hardness from the complexity-theoretic landscape: unless every lattice has short-vector approximation algorithms (which would collapse major complexity classes), LWE-based schemes are secure.

---

## 5. Ring-LWE and Polynomial Arithmetic

### The Problem

Reduce the key sizes and computational cost of LWE-based schemes by working over structured algebraic rings while maintaining provable security.

### The Formula

Fix $n$ a power of 2. The **cyclotomic ring** is:

$$R = \mathbb{Z}[x] / (x^n + 1)$$

and for modulus $q$:

$$R_q = \mathbb{Z}_q[x] / (x^n + 1)$$

Elements of $R_q$ are polynomials of degree $< n$ with coefficients in $\mathbb{Z}_q$. Multiplication is polynomial multiplication modulo $x^n + 1$.

**Ring-LWE distribution:** For secret $s \in R_q$ and error distribution $\chi$ over $R$:

$$(a, b = a \cdot s + e) \in R_q \times R_q$$

where $a \xleftarrow{\$} R_q$ and $e \leftarrow \chi$.

**Number Theoretic Transform (NTT):** Since $x^n + 1$ splits completely modulo $q$ when $q \equiv 1 \pmod{2n}$, multiplication in $R_q$ can be performed in $O(n \log n)$ via NTT:

1. Map $a(x) \mapsto \hat{a} = (\hat{a}_0, \ldots, \hat{a}_{n-1})$ where $\hat{a}_j = a(\omega^{2j+1})$ for primitive $2n$-th root of unity $\omega$.
2. Pointwise multiply: $\widehat{a \cdot b}_j = \hat{a}_j \cdot \hat{b}_j$.
3. Inverse NTT to recover the product.

**Key size comparison:**

| Scheme | Public key | Secret key |
|--------|-----------|------------|
| LWE ($n \times n$ matrix) | $O(n^2 \log q)$ bits | $O(n \log q)$ bits |
| Ring-LWE (single ring element) | $O(n \log q)$ bits | $O(n \log q)$ bits |

For $n = 1024$, $q \approx 2^{30}$: LWE public key $\approx 30$ GB, Ring-LWE $\approx 3.75$ KB.

### Security of Ring-LWE

Lyubashevsky, Peikert, and Regev (2010) proved that Ring-LWE is at least as hard as worst-case problems on **ideal lattices** -- lattices that correspond to ideals in the ring $R$. Specifically:

$$\text{worst-case Ideal-SVP}_\gamma \leq_Q \text{Ring-LWE}_{n,q,\alpha}$$

Ideal lattices are a special class of lattices, so this is a weaker assumption than general lattice hardness. However, decades of study have not produced algorithms that exploit the ideal structure to break Ring-LWE at the parameter sizes used in practice.

### Why It Matters

Ring-LWE makes lattice cryptography practical. Without the algebraic structure, key sizes would be gigabytes for reasonable security. The NTT enables sub-millisecond operations. Every deployed lattice scheme (Kyber, Dilithium) uses the ring or module variant.

---

## 6. ML-KEM (Kyber) -- Key Encapsulation

### The Problem

Design a CCA-secure key encapsulation mechanism suitable for replacing RSA and ECDH in TLS, SSH, and other protocols, resistant to quantum attacks.

### The Formula

**Ring:** $R_q = \mathbb{Z}_{3329}[x]/(x^{256} + 1)$, rank $k \in \{2, 3, 4\}$.

**Core CPA-secure encryption (CPAPKE):**

$\text{KeyGen}():$
1. $A \xleftarrow{\$} R_q^{k \times k}$ (derived from seed $\rho$ via XOF)
2. $s, e \leftarrow \beta_\eta^k$ (centered binomial distribution, coefficients in $\{-\eta, \ldots, \eta\}$)
3. $t = As + e$
4. Return $(\text{pk} = (t, \rho),\ \text{sk} = s)$

$\text{Encrypt}(\text{pk}, m \in \{0,1\}^{256}, \text{coins } r):$
1. $r', e_1 \leftarrow \beta_\eta^k$, $e_2 \leftarrow \beta_\eta$ (using coins $r$)
2. $u = A^T r' + e_1$
3. $v = t^T r' + e_2 + \lceil q/2 \rfloor \cdot \text{Decompress}(m)$
4. Return $(c_1, c_2) = (\text{Compress}(u, d_u), \text{Compress}(v, d_v))$

$\text{Decrypt}(\text{sk}, (c_1, c_2)):$
1. $u' = \text{Decompress}(c_1, d_u)$, $v' = \text{Decompress}(c_2, d_v)$
2. $m = \text{Compress}(v' - s^T u', 1)$

**Fujisaki-Okamoto (FO) Transform** converts CPA-PKE to CCA-KEM:

$\text{Encaps}(\text{pk}):$
1. $m \xleftarrow{\$} \{0,1\}^{256}$
2. $(K, r) = G(m \| H(\text{pk}))$ (hash to get key and encryption coins)
3. $c = \text{Encrypt}(\text{pk}, m, r)$
4. $K' = \text{KDF}(K \| H(c))$
5. Return $(c, K')$

$\text{Decaps}(\text{sk}, c):$
1. $m' = \text{Decrypt}(\text{sk}, c)$
2. $(K', r') = G(m' \| H(\text{pk}))$
3. $c' = \text{Encrypt}(\text{pk}, m', r')$
4. If $c = c'$: return $K = \text{KDF}(K' \| H(c))$
5. Else: return $K = \text{KDF}(z \| H(c))$ for random $z$ in sk (implicit rejection)

### Why It Matters

ML-KEM is NIST's primary post-quantum KEM standard (FIPS 203, finalized August 2024). It replaces ECDH in TLS 1.3 hybrid key exchange (already deployed by Google, Cloudflare, and others). The Compress/Decompress operations trade a small decryption failure probability ($< 2^{-139}$) for significantly smaller ciphertext sizes.

---

## 7. ML-DSA (Dilithium) -- Digital Signatures

### The Problem

Construct a lattice-based digital signature scheme with small signatures, fast verification, and CMA security, suitable as a drop-in replacement for RSA/ECDSA signatures.

### The Formula

**Ring:** $R_q = \mathbb{Z}_{8380417}[x]/(x^{256} + 1)$, dimensions $(k, \ell)$ varying by security level.

**Key Generation:**
1. $A \xleftarrow{\$} R_q^{k \times \ell}$ (expanded from seed $\rho$)
2. $s_1 \leftarrow S_\eta^\ell$, $s_2 \leftarrow S_\eta^k$ (secret vectors with small coefficients)
3. $t = As_1 + s_2$
4. Split $t = t_1 \cdot 2^d + t_0$ (power2round)
5. $\text{pk} = (\rho, t_1)$, $\text{sk} = (\rho, K, \text{tr}, s_1, s_2, t_0)$

**Signing** (Fiat-Shamir with Aborts):
1. $y \leftarrow S_{\gamma_1 - 1}^\ell$ (masking vector)
2. $w = Ay$
3. $w_1 = \text{HighBits}(w, 2\gamma_2)$
4. $\tilde{c} = H(\text{tr} \| \mu \| w_1)$ (challenge hash)
5. $c = \text{SampleInBall}(\tilde{c})$ (sparse polynomial, weight $\tau$)
6. $z = y + cs_1$
7. **Check 1:** $\|z\|_\infty < \gamma_1 - \beta$ (reject if too large)
8. **Check 2:** $\|\text{LowBits}(w - cs_2, 2\gamma_2)\|_\infty < \gamma_2 - \beta$ (reject if too large)
9. If either check fails, restart from step 1 (abort and retry)
10. Return $\sigma = (\tilde{c}, z, h)$ where $h$ encodes hint bits

**Verification:**
1. $w_1' = \text{UseHint}(h, Az - ct_1 \cdot 2^d, 2\gamma_2)$
2. Check $\tilde{c} = H(\text{tr} \| \mu \| w_1')$
3. Check $\|z\|_\infty < \gamma_1 - \beta$

The **rejection sampling** (abort) step is critical: without it, the distribution of $z = y + cs_1$ would depend on $s_1$, leaking the secret key over multiple signatures. With rejection, $z$ is distributed as $S_{\gamma_1 - 1}^\ell$ regardless of $s_1$.

Expected number of iterations before acceptance: approximately 4-7 depending on parameter set.

### Why It Matters

ML-DSA is NIST's primary post-quantum signature standard (FIPS 204). The Fiat-Shamir with Aborts paradigm is elegant: it converts an identification scheme into a signature scheme while the rejection sampling provides zero-knowledge, ensuring no information about $s_1$ leaks even after millions of signatures.

---

## 8. Fully Homomorphic Encryption and Bootstrapping

### The Problem

Enable computation on encrypted data: given $\text{Enc}(m_1)$ and $\text{Enc}(m_2)$, compute $\text{Enc}(f(m_1, m_2))$ for any function $f$ without ever decrypting.

### The Formula

**Somewhat Homomorphic Encryption (SHE):** An LWE-based encryption of message $m$ under secret $s$ is:

$$\text{ct} = (a, b) \quad \text{where} \quad b = \langle a, s \rangle + e + \lfloor q/p \rfloor m \pmod{q}$$

Decryption computes $b - \langle a, s \rangle = e + \lfloor q/p \rfloor m$, then recovers $m$ by rounding if $|e| < q/(2p)$.

**Homomorphic addition:** $(a_1 + a_2, b_1 + b_2)$ encrypts $m_1 + m_2$ with noise $e_1 + e_2$.

**Homomorphic multiplication** (simplified): produces encryption of $m_1 \cdot m_2$ but noise grows multiplicatively: $e_{\text{prod}} \approx e_1 \cdot e_2 \cdot q/p$. After $L$ levels of multiplication, noise is roughly $B^{2^L}$ where $B$ is initial noise bound.

Once noise exceeds $q/(2p)$, decryption fails. This limits computation depth.

**Bootstrapping (Gentry, 2009):** The key idea -- homomorphically evaluate the decryption function itself to "refresh" a noisy ciphertext:

1. Start with noisy ciphertext $\text{ct}$ under key $s$ (noise near threshold)
2. Encrypt $s$ bit-by-bit under a fresh key $s'$: publish $\text{Enc}_{s'}(s_i)$
3. Homomorphically evaluate $\text{Dec}_s(\text{ct})$ using the encrypted key
4. Result: a fresh ciphertext under $s'$ with low noise encrypting the same $m$

For bootstrapping to work, the SHE scheme must handle circuits at least as deep as its own (augmented) decryption circuit. Gentry called this **bootstrappability**.

**Modulus Switching (BGV approach):** Instead of bootstrapping after every operation, switch to a smaller modulus $q' < q$ to reduce noise proportionally:

$$e' \approx e \cdot (q'/q) + \text{rounding error}$$

This gives **leveled FHE**: support $L$ levels of multiplication by starting with modulus $q_0 \approx q^L$ and switching down after each level. No bootstrapping needed if depth is bounded.

**CKKS Approximate Arithmetic:** Encode real number $m$ as:

$$\text{ct encrypts} \quad \Delta \cdot m + e$$

where $\Delta$ is a scaling factor. After multiplication, the result is $\Delta^2 \cdot m_1 m_2 + \text{noise}$. **Rescaling** divides by $\Delta$ to keep the scale manageable, treating some noise as acceptable precision loss.

### Why It Matters

FHE enables privacy-preserving computation: cloud computing on encrypted medical records, encrypted machine learning inference, private set intersection. The BGV/BFV schemes handle exact integer arithmetic (voting, financial), while CKKS handles approximate arithmetic (statistics, ML). Practical implementations now achieve bootstrapping in seconds and leveled FHE evaluation in milliseconds per operation.

---

## 9. Security Parameter Selection

### The Problem

Choose concrete parameters $(n, q, \sigma)$ for lattice schemes such that the best known attacks require at least $2^\lambda$ operations for target security level $\lambda$ (e.g., 128, 192, 256 bits).

### The Formula

The **core-SVP methodology** estimates security by the BKZ block size $\beta$ needed to recover the secret:

1. The LWE instance defines a lattice of dimension $d$ with a short vector of length $\delta$ relative to the lattice determinant.
2. BKZ-$\beta$ finds vectors of length approximately:

$$\|b_1\| \approx \delta_\beta^{d} \cdot \det(L)^{1/d}$$

where the root Hermite factor is $\delta_\beta \approx \left(\frac{\beta}{2\pi e} (\pi \beta)^{1/\beta}\right)^{1/(2(\beta-1))}$.

3. Set $\|b_1\|$ equal to the target short vector length and solve for $\beta$.
4. The cost of BKZ-$\beta$ is dominated by SVP calls on $\beta$-dimensional sublattices. Using lattice sieving:

$$T_{\text{classical}} \approx 2^{0.292\beta + o(\beta)}$$

$$T_{\text{quantum}} \approx 2^{0.265\beta + o(\beta)} \quad \text{(using Grover-enhanced sieving)}$$

5. Set $T \geq 2^\lambda$ and solve for required $\beta$, then choose $(n, q, \sigma)$ accordingly.

**NIST security categories:**

| Category | Classical equiv. | Required BKZ $\beta$ (approx.) |
|----------|-----------------|-------------------------------|
| 1 | AES-128 | $\beta \geq 440$ |
| 3 | AES-192 | $\beta \geq 660$ |
| 5 | AES-256 | $\beta \geq 875$ |

**Parameter trade-offs:**

- Larger $n$: more security, larger keys and ciphertexts
- Larger $q$: more room for noise, but weaker security per dimension
- Larger $\sigma$ (noise): more security, higher decryption failure rate
- The ratio $q/\sigma$ largely determines security for fixed $n$

**Decryption failure rate** must satisfy $\delta_{\text{fail}} < 2^{-\lambda}$ to prevent failure-based attacks (D'Anvers et al., 2019). ML-KEM achieves $\delta_{\text{fail}} < 2^{-139}$ for ML-KEM-512.

### Why It Matters

Parameter selection is the bridge between theoretical hardness and deployed security. The core-SVP model is conservative (ignores polynomial factors, memory constraints) but is the accepted methodology. As algorithms improve -- for example, if sieving constants decrease from 0.292 -- parameters must be re-evaluated, which is why NIST standards include multiple security levels.

---

## References

- Regev, O. "On Lattices, Learning with Errors, Random Linear Codes, and Cryptography." *Journal of the ACM*, 56(6), 2009.
- Ajtai, M. "Generating Hard Instances of Lattice Problems." *STOC*, 1996.
- Gentry, C. "Fully Homomorphic Encryption Using Ideal Lattices." *STOC*, 2009.
- Lyubashevsky, V., Peikert, C., Regev, O. "On Ideal Lattices and Learning with Errors Over Rings." *EUROCRYPT*, 2010.
- Peikert, C. "Public-Key Cryptosystems from the Worst-Case Shortest Vector Problem." *STOC*, 2009.
- Lenstra, A.K., Lenstra, H.W., Lovasz, L. "Factoring Polynomials with Rational Coefficients." *Mathematische Annalen*, 261, 1982.
- Brakerski, Z., Gentry, C., Vaikuntanathan, V. "(Leveled) Fully Homomorphic Encryption without Bootstrapping." *ITCS*, 2012.
- Cheon, J.H., Kim, A., Kim, M., Song, Y. "Homomorphic Encryption for Arithmetic of Approximate Numbers." *ASIACRYPT*, 2017.
- NIST FIPS 203 (ML-KEM), FIPS 204 (ML-DSA), FIPS 205 (SLH-DSA), August 2024.
- Micciancio, D., Regev, O. "Lattice-based Cryptography." In *Post-Quantum Cryptography*, Springer, 2009.
