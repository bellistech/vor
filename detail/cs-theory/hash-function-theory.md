# The Mathematics of Hash Functions -- Cryptographic Security, Universal Hashing, and Provable Constructions

> *A hash function compresses the infinite into the finite, and the art of cryptographic hashing lies in ensuring that this compression destroys all exploitable structure -- that the output is, for all practical purposes, indistinguishable from randomness.*

---

## 1. Birthday Bound -- Collision Probability

### The Problem

Quantify the probability that a random hash function with $n$-bit output produces a collision after $q$ queries, establishing the fundamental security limit for any hash function.

### The Derivation

Model $H$ as a random function $H : \mathcal{M} \to \{0,1\}^n$. Let $N = 2^n$. After $q$ queries with distinct inputs $x_1, \ldots, x_q$, the outputs $H(x_1), \ldots, H(x_q)$ are independent and uniform over $\{0, \ldots, N-1\}$.

The probability of **no collision** among $q$ outputs:

$$P(\text{no collision}) = \prod_{i=1}^{q-1} \left(1 - \frac{i}{N}\right)$$

Using the inequality $1 - x \leq e^{-x}$ for $x \geq 0$:

$$P(\text{no collision}) \leq \prod_{i=1}^{q-1} e^{-i/N} = e^{-\sum_{i=1}^{q-1} i/N} = e^{-q(q-1)/(2N)}$$

Therefore:

$$P(\text{collision}) \geq 1 - e^{-q(q-1)/(2N)}$$

Setting $P(\text{collision}) = 1/2$ and solving:

$$q \approx \sqrt{2N \ln 2} = 1.1774 \cdot \sqrt{N} = 1.1774 \cdot 2^{n/2}$$

### The Exact Form

For small $q$ relative to $N$, a tighter approximation uses the Taylor expansion of $\ln(1 - x) \approx -x - x^2/2 - \cdots$:

$$P(\text{collision}) \approx 1 - e^{-q^2/(2N)} \approx \frac{q^2}{2N} \quad \text{for } q \ll \sqrt{N}$$

This is the **birthday bound**: an $n$-bit hash function cannot provide more than $n/2$ bits of collision resistance against generic attacks.

### Multi-collision Generalization (Joux)

For an $r$-collision (finding $r$ messages with the same hash), the expected number of queries is:

$$q \sim (r!)^{1/r} \cdot N^{(r-1)/r}$$

For $r = 2$ (standard collision), $q \sim \sqrt{2N}$, recovering the birthday bound. The key insight (Joux 2004) is that for iterated hash functions, multicollisions are far cheaper than expected -- see Section 8.

---

## 2. Merkle-Damgard Security Proof

### The Problem

Prove that if the compression function $f$ is collision-resistant, then the Merkle-Damgard iteration $H$ is collision-resistant.

### The Construction

Let $f : \{0,1\}^n \times \{0,1\}^b \to \{0,1\}^n$ be a compression function. Define $H$ on a padded message $M = m_1 \| m_2 \| \cdots \| m_k$ (where $m_k$ includes the length encoding) by:

$$h_0 = IV$$
$$h_i = f(h_{i-1}, m_i) \quad \text{for } i = 1, \ldots, k$$
$$H(M) = h_k$$

### The Proof (by contrapositive)

Suppose we have a collision for $H$: two distinct messages $M \neq M'$ with $H(M) = H(M')$.

Let $M = m_1 \| \cdots \| m_k$ and $M' = m'_1 \| \cdots \| m'_{k'}$.

**Case 1: $k \neq k'$.** Since Merkle-Damgard strengthening appends $|M|$ in the final block, $m_k \neq m'_{k'}$ (they encode different lengths). But $f(h_{k-1}, m_k) = f(h'_{k'-1}, m'_{k'})$. If $(h_{k-1}, m_k) \neq (h'_{k'-1}, m'_{k'})$, we have a collision in $f$. Since $m_k \neq m'_{k'}$, the inputs differ, so this is indeed a collision in $f$.

**Case 2: $k = k'$.** Walk backward from the final block. We have $f(h_{k-1}, m_k) = f(h'_{k-1}, m'_k)$. Either:
- $(h_{k-1}, m_k) \neq (h'_{k-1}, m'_k)$: collision in $f$, done.
- $(h_{k-1}, m_k) = (h'_{k-1}, m'_k)$: then $h_{k-1} = h'_{k-1}$ and $m_k = m'_k$. Recurse on position $k-1$.

Since $M \neq M'$ and $k = k'$, there exists some index $j$ where $m_j \neq m'_j$. The backward walk must reach this index, yielding a collision in $f$ at some step.

**Conclusion**: A collision in $H$ implies a collision in $f$. Equivalently, collision resistance of $f$ implies collision resistance of $H$. $\square$

### Length Extension Attack

The same iterative structure enables length extension. Given $H(M)$ and $|M|$ (but not $M$ itself), an attacker computes:

$$H(M \| \text{pad}(|M|) \| M') = f(\cdots f(H(M), m'_1) \cdots, m'_{k'})$$

This works because $H(M)$ is the internal state after processing $M$, and the attacker can continue the iteration. This is a structural weakness of Merkle-Damgard, not a collision attack -- it breaks domain separation properties expected of a hash function.

**Mitigation**: Use HMAC, apply a finalization step (as in SHA-3's sponge), or use a wide-pipe design (internal state wider than output).

---

## 3. Sponge Construction -- Security Proof Sketch

### The Problem

Establish the security of the sponge construction used by SHA-3, relating it to the security of the underlying permutation.

### The Construction

A sponge operates on a state of $b = r + c$ bits, where $r$ is the **rate** (bits absorbed/squeezed per step) and $c$ is the **capacity** (security parameter).

Let $f : \{0,1\}^b \to \{0,1\}^b$ be a public random permutation. The sponge $S[f, r]$ operates as:

**Absorbing**: Starting from the all-zero state, XOR each $r$-bit message block into the first $r$ bits of the state, then apply $f$.

**Squeezing**: Output the first $r$ bits of the state, apply $f$, repeat until sufficient output is produced.

### The Security Bound

**Theorem** (Bertoni-Daemen-Peeters-Van Assche 2008): The sponge construction $S[f, r]$ with a random permutation $f$ is indifferentiable from a random oracle up to $\mathcal{O}(2^{c/2})$ queries.

The proof uses the **indifferentiability framework** (Section 9). The key argument is:

1. Model $f$ as a random permutation (ideal permutation model).
2. Construct a simulator $\mathcal{S}$ that, given access to a random oracle $\mathcal{R}$, simulates $f$ such that no distinguisher $\mathcal{D}$ making at most $q$ queries can tell apart $(\text{Sponge}^f, f)$ from $(\mathcal{R}, \mathcal{S}^{\mathcal{R}})$.
3. The simulator must maintain consistency: for any absorb-squeeze path through the sponge, the outputs must match $\mathcal{R}$.
4. An inconsistency arises only if a distinguisher's query to $f$ or $f^{-1}$ causes a **state collision** in the $c$-bit capacity portion. Since the capacity bits are never directly accessible, each query has at most a $2^{-c}$ probability of hitting a specific capacity value.
5. By a union bound over $q$ queries, the distinguishing advantage is:

$$\text{Adv} \leq \frac{q(q+1)}{2 \cdot 2^c} \approx \frac{q^2}{2^{c+1}}$$

This becomes non-negligible when $q \approx 2^{c/2}$.

### Implications for SHA-3

For SHA-3-256: $c = 512$, so security up to $2^{256}$ permutation queries -- matching the $256/2 = 128$-bit collision resistance and $256$-bit preimage resistance expected of a $256$-bit hash.

The sponge avoids length extension attacks structurally: the capacity bits are never output, so the full internal state cannot be recovered from the hash output.

---

## 4. HMAC Security Proof

### The Problem

Prove that HMAC is a secure pseudorandom function (PRF) when the underlying compression function is a PRF.

### The Construction

$$\text{HMAC}_K(M) = H\bigl((K \oplus \text{opad}) \| H((K \oplus \text{ipad}) \| M)\bigr)$$

where $H$ is a Merkle-Damgard hash, $\text{opad} = \texttt{0x5C}\ldots\texttt{5C}$, $\text{ipad} = \texttt{0x36}\ldots\texttt{36}$.

### The Proof Strategy (Bellare 2006)

Let $f : \{0,1\}^n \times \{0,1\}^b \to \{0,1\}^n$ be the compression function of $H$.

**Step 1: Decompose HMAC.** Define the **inner function**:

$$\text{NMAC}_K(M) = f_{K_1}^*(f_{K_2}^*(M))$$

where $f_k^*$ denotes the iterated application of $f$ with initial value $k$, and $K = (K_1, K_2)$.

HMAC is a specific instantiation of NMAC where $K_1 = f(IV, K \oplus \text{opad})$ and $K_2 = f(IV, K \oplus \text{ipad})$.

**Step 2: NMAC is a PRF if $f$ is a PRF.** The inner hash $f_{K_2}^*(M)$ is a PRF over variable-length messages (by the cascade construction -- a PRF-based analogue of the Merkle-Damgard collision resistance proof). The outer application $f_{K_1}(\cdot)$ compresses the output of the inner PRF through a single application of the PRF $f$.

**Step 3: Security bound.** For any adversary $\mathcal{A}$ making at most $q$ queries of total length at most $\sigma$ blocks:

$$\text{Adv}_{\text{HMAC}}^{\text{PRF}}(\mathcal{A}) \leq \text{Adv}_f^{\text{PRF}}(q, \sigma) + \text{Adv}_f^{\text{PRF}}(q, q)$$

The first term bounds the inner PRF distinguishing advantage; the second bounds the outer application (which processes $q$ single-block messages).

### Key Insight

HMAC does **not** require collision resistance of $H$. If $f$ is a PRF but $H$ has collisions (as with MD5), HMAC remains secure. The PRF property of $f$ is a weaker assumption than collision resistance of the iterated hash.

---

## 5. Universal Hashing -- Collision Probability Bounds

### The Problem

Establish the collision probability guarantees of universal hash families and their application to hash tables and MACs.

### Carter-Wegman Universal Hashing (1979)

A family $\mathcal{H} = \{h : U \to [m]\}$ is **universal** (or 2-universal) if for all $x \neq y \in U$:

$$\Pr_{h \leftarrow \mathcal{H}}[h(x) = h(y)] \leq \frac{1}{m}$$

It is **strongly universal** (pairwise independent) if for all $x \neq y$ and all $a, b \in [m]$:

$$\Pr_{h \leftarrow \mathcal{H}}[h(x) = a \text{ and } h(y) = b] = \frac{1}{m^2}$$

### The Linear Construction

For a prime $p \geq |U|$, the family:

$$h_{a,b}(x) = ((ax + b) \bmod p) \bmod m$$

with $a \in \{1, \ldots, p-1\}$, $b \in \{0, \ldots, p-1\}$ is strongly universal. The collision probability satisfies:

$$\Pr[h_{a,b}(x) = h_{a,b}(y)] \leq \frac{1}{m} + \frac{1}{p} \leq \frac{2}{m}$$

for $p \geq m$.

### Expected Performance in Hash Tables

With $n$ keys hashed into $m$ buckets using a universal family:

- **Expected maximum chain length**: $O(\sqrt{n/m \cdot \log m})$ for 2-universal families; $O(\log n / \log \log n)$ for $O(\log n)$-wise independence.
- **Expected total collisions**: For a universal family, the expected number of colliding pairs is at most $\binom{n}{2} / m$.
- **Chernoff-style bounds** require higher independence. With $k$-wise independence ($k \geq 4$), the probability that any bucket has more than $c \cdot n/m$ keys decreases polynomially in $c$.

### Tabulation Hashing

Tabulation hashing (Patrascu-Thorup 2012) achieves strong concentration bounds (comparable to full independence for many applications) using only 3-wise independent tables:

$$h(x) = T_1[x_1] \oplus T_2[x_2] \oplus \cdots \oplus T_c[x_c]$$

where $x = (x_1, \ldots, x_c)$ is the key split into $c$ characters, and each $T_i$ is a random lookup table. Simple tabulation is only 3-wise independent but behaves like full independence for linear probing and cuckoo hashing.

---

## 6. Carter-Wegman MAC

### The Problem

Construct an information-theoretically secure message authentication code from universal hashing.

### The Construction

Let $\mathcal{H} = \{h_k : \mathcal{M} \to \{0,1\}^n\}$ be a universal hash family. Let $E$ be a PRF (or one-time pad). The Carter-Wegman MAC is:

$$\text{CW-MAC}_{(k, r)}(M) = h_k(M) \oplus E_r(\text{nonce})$$

The universal hash compresses the message, and the encryption masks the hash output.

### Security Bound

For any adversary making $q$ MAC queries:

$$\Pr[\text{forgery}] \leq \frac{1}{2^n} + \text{Adv}_E^{\text{PRF}}(q)$$

The $1/2^n$ term comes from the universal hash collision probability. Since the nonce ensures the pad is fresh for each message, the adversary learns nothing about $h_k$ from previous MAC values.

### Polynomial Evaluation MAC (GHASH/Poly1305)

A practical instantiation uses polynomial evaluation over $\text{GF}(2^{128})$:

$$h_r(m_1, \ldots, m_\ell) = m_1 r^\ell + m_2 r^{\ell-1} + \cdots + m_\ell r$$

This is $\epsilon$-almost-universal with $\epsilon = \ell / 2^{128}$ (the probability that a degree-$\ell$ polynomial has a root at a random point). Used in AES-GCM (GHASH) and ChaCha20-Poly1305.

---

## 7. Merkle Tree Proof Verification

### The Problem

Prove that a Merkle tree proof (authentication path) guarantees integrity of a leaf with security equivalent to the underlying hash function.

### The Construction

A Merkle tree over $n = 2^d$ data blocks $D_0, \ldots, D_{n-1}$:

- **Leaves**: $L_i = H(0x00 \| D_i)$ (domain-separated with a leaf prefix)
- **Internal nodes**: $N_{i,j} = H(0x01 \| N_{i+1, 2j} \| N_{i+1, 2j+1})$ (internal prefix)
- **Root**: $R = N_{0,0}$

The proof of inclusion for leaf $D_j$ consists of the $d = \log_2 n$ sibling hashes along the path from $L_j$ to the root.

### Verification

Given data $D_j$, authentication path $(s_0, s_1, \ldots, s_{d-1})$, and root $R$:

1. Compute $v_0 = H(0x00 \| D_j)$
2. For $i = 0, \ldots, d-1$:
   - If $j$'s $i$-th bit is 0: $v_{i+1} = H(0x01 \| v_i \| s_i)$
   - If $j$'s $i$-th bit is 1: $v_{i+1} = H(0x01 \| s_i \| v_i)$
3. Accept if $v_d = R$

### Security

**Theorem**: If $H$ is collision-resistant, no polynomial-time adversary can produce a valid Merkle proof for $D'_j \neq D_j$ under the same root $R$.

**Proof sketch**: A forged proof implies that either:
- Two different leaf values produce the same leaf hash (collision in $H$), or
- Two different inputs to an internal node produce the same output (collision in $H$).

The domain separation prefixes ($0x00$ for leaves, $0x01$ for internal nodes) prevent **second preimage attacks** where an attacker reinterprets an internal node as a leaf or vice versa.

**Proof size**: $d \cdot n_{\text{hash}} = \log_2(n) \cdot n_{\text{hash}}$ bits, where $n_{\text{hash}}$ is the hash output length. For $n = 2^{20}$ leaves with SHA-256: $20 \times 256 = 5120$ bits $= 640$ bytes.

---

## 8. Generic Attacks on Iterated Hash Functions

### Joux Multicollision Attack (2004)

**Theorem** (Joux): For any $n$-bit Merkle-Damgard hash, a $2^t$-way multicollision (a set of $2^t$ distinct messages all hashing to the same value) can be found in time $t \cdot 2^{n/2}$ -- only $t$ times the cost of a single collision.

**Procedure**:
1. Starting from $IV$, find a collision pair $(m_0, m'_0)$ with $f(IV, m_0) = f(IV, m'_0) = h_1$. Cost: $2^{n/2}$.
2. Starting from $h_1$, find a collision pair $(m_1, m'_1)$ with $f(h_1, m_1) = f(h_1, m'_1) = h_2$. Cost: $2^{n/2}$.
3. Repeat $t$ times.
4. Any combination of $m_i$ or $m'_i$ at each step yields a valid message, giving $2^t$ messages that all hash to $h_t$.

**Total cost**: $t \cdot 2^{n/2}$ for a $2^t$-multicollision.

**Implication**: Cascading two independent $n$-bit Merkle-Damgard hashes $H_1(M) \| H_2(M)$ does NOT give $2n$ bits of collision resistance. An attacker finds $2^{n/2+1}$ messages colliding under $H_1$ (cost $(n/2+1) \cdot 2^{n/2}$), then searches for a collision under $H_2$ among these messages (cost $2^{n/4}$ by birthday on $2^{n/2+1}$ messages). Total: roughly $2^{n/2}$, not $2^n$.

### Herding Attack (Kelsey-Schneier 2006)

A herding attack enables a **chosen-target-forced-prefix preimage**: commit to a hash value $h$ first, then given a challenge prefix $P$, find a suffix $S$ such that $H(P \| S) = h$.

**Procedure**:
1. **Offline phase**: Build a diamond structure -- a $2^k$-node tree of intermediate hash values that all converge to a single final state. Cost: $2^{(n+k)/2+2}$ compression function evaluations.
2. **Online phase**: Given prefix $P$, compute $h_P = H'(IV, P)$ (the intermediate state after processing $P$), then find a one-block linking message from $h_P$ to any node in the diamond. Cost: $2^{n-k}$.

**Optimal $k$**: Setting $k = n/3$ gives total cost $\approx 2^{2n/3}$, beating the $2^n$ cost of a brute-force preimage attack.

---

## 9. Indifferentiability Framework (Maurer-Renner-Holenstein 2004)

### The Problem

Formalize when a hash construction $C^f$ (using an ideal primitive $f$) can securely replace a random oracle $\mathcal{R}$ in any cryptographic application.

### The Definition

A construction $C^f$ is **indifferentiable** from $\mathcal{R}$ if there exists a simulator $\mathcal{S}$ (with access to $\mathcal{R}$) such that for any distinguisher $\mathcal{D}$:

$$\left| \Pr[\mathcal{D}^{C^f, f} = 1] - \Pr[\mathcal{D}^{\mathcal{R}, \mathcal{S}^{\mathcal{R}}} = 1] \right| \leq \epsilon$$

In the **real world**, $\mathcal{D}$ interacts with the construction $C^f$ and the underlying primitive $f$. In the **ideal world**, $\mathcal{D}$ interacts with a random oracle $\mathcal{R}$ and a simulator $\mathcal{S}$ that mimics $f$ consistently.

### The Composition Theorem

**Theorem** (MRH 2004): If $C^f$ is indifferentiable from $\mathcal{R}$, then for any cryptosystem $\Pi$ secure in the random oracle model, the instantiation $\Pi^{C^f}$ is secure in the $f$-ideal model.

This is strictly stronger than mere collision resistance or PRF security. It captures all "hash function-like" properties simultaneously.

### Applications

| Construction | Ideal Primitive | Indifferentiable from RO? |
|-------------|----------------|--------------------------|
| Merkle-Damgard | Ideal compression function | No (length extension) |
| HMAC / NMAC | Ideal compression function | No (related keys) |
| Sponge | Ideal permutation | Yes, up to $2^{c/2}$ queries |
| Chop-MD (truncated) | Ideal compression function | Yes, up to $2^{(b-n)/2}$ queries |
| Enveloped MD: $H_2(H_1(M))$ | Ideal compression function | Yes (under some conditions) |

### Why Merkle-Damgard Fails

The distinguisher for Merkle-Damgard is straightforward:

1. Query $\mathcal{O}_1$ (the hash/RO) on a message $M$ to get $y$.
2. Compute $\text{pad}(|M|)$.
3. Query $\mathcal{O}_2$ (the primitive/simulator) as $f(y, m')$ for an arbitrary block $m'$.
4. Query $\mathcal{O}_1$ on $M \| \text{pad}(|M|) \| m'$ to get $y'$.
5. Check if $y' = f(y, m')$.

In the real world, this always holds (by construction). In the ideal world with a random oracle, $y'$ is random and independent of $f(y, m')$, so the check fails with overwhelming probability.

---

## Tips

- The birthday bound is tight for generic attacks on hash functions. Any construction must have output at least $2s$ bits for $s$-bit collision security.
- Length extension is not a theoretical curiosity -- it has broken real protocols (e.g., Flickr API signature scheme, 2009).
- The sponge's capacity $c$ is the security parameter, not the output length. SHA-3-256 has $c = 512$ bits, giving $256$-bit security against generic attacks.
- HMAC's security proof shows it is safe to use HMAC-MD5 and HMAC-SHA1 as PRFs, even though the underlying hash functions have known collisions. The relevant property is compression function PRF security, not collision resistance.
- Joux multicollisions show that concatenating hash functions ($H_1(M) \| H_2(M)$) provides far less security than expected. Use a single hash with sufficient output length instead.
- The indifferentiability framework is the gold standard for hash function security proofs. A construction that is indifferentiable from a random oracle can safely replace a random oracle in any application -- a property that plain collision resistance does not guarantee.

## See Also

- Cryptography Fundamentals
- Computational Complexity
- Information Theory
- Number Theory
- Data Structures

## References

- Merkle, R. "A certified digital signature." CRYPTO '89, LNCS 435, pp. 218-238 (1989)
- Damgard, I. "A design principle for hash functions." CRYPTO '89, LNCS 435, pp. 416-427 (1989)
- Bellare, M., Canetti, R., Krawczyk, H. "Keying hash functions for message authentication." CRYPTO '96, LNCS 1109 (1996)
- Bellare, M. "New proofs for NMAC and HMAC: security without collision resistance." J. Cryptology 28(4):844-878 (2015), originally CRYPTO '06
- Bertoni, G., Daemen, J., Peeters, M., Van Assche, G. "On the indifferentiability of the sponge construction." EUROCRYPT 2008, LNCS 4965, pp. 181-197 (2008)
- Maurer, U., Renner, R., Holenstein, C. "Indifferentiability, impossibility results on reductions, and applications to the random oracle methodology." TCC 2004, LNCS 2951, pp. 21-39 (2004)
- Carter, L. & Wegman, M. "Universal classes of hash functions." JCSS 18(2):143-154 (1979)
- Joux, A. "Multicollisions in iterated hash functions." CRYPTO 2004, LNCS 3152, pp. 306-316 (2004)
- Kelsey, J. & Schneier, B. "Second preimages on n-bit hash functions for much less than $2^n$ work." EUROCRYPT 2005, LNCS 3494 (2005)
- Black, J., Rogaway, P., Shrimpton, T. "Black-box analysis of the block-cipher-based hash-function constructions from PGV." CRYPTO 2002, LNCS 2442 (2002)
- Patrascu, M. & Thorup, M. "The power of simple tabulation hashing." JACM 59(3):1-50 (2012)
- Fredman, M., Komlos, J., Szemeredi, E. "Storing a sparse table with O(1) worst case access time." JACM 31(3):538-544 (1984)
- Canetti, R., Goldreich, O., Halevi, S. "The random oracle methodology, revisited." JACM 51(4):557-594 (2004)
- Coron, J.-S., Dodis, Y., Malinaud, C., Puniya, P. "Merkle-Damgard revisited: how to construct a hash function." CRYPTO 2005, LNCS 3621 (2005)
