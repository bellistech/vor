# The Theory of Zero-Knowledge Proofs -- From Interactive Proofs to SNARKs, STARKs, and Modern Proof Systems

> *Zero-knowledge proofs allow a prover to convince a verifier of the truth of a statement while revealing nothing beyond validity itself. Introduced by Goldwasser, Micali, and Rackoff in 1985, ZK proofs underpin modern cryptographic privacy, blockchain scalability, and verifiable computation, with practical instantiations including Groth16 SNARKs, FRI-based STARKs, and Bulletproofs.*

---

## 1. Formal Definition of Zero-Knowledge Proofs

### The Problem

Define precisely what it means for a proof system to be "zero-knowledge" -- that is, to convince a verifier of a statement's truth while provably leaking no additional information.

### The Formula

An **interactive proof system** for a language $L$ is a pair $(P, V)$ where $P$ is an unbounded prover and $V$ is a probabilistic polynomial-time verifier. They exchange messages over multiple rounds, after which $V$ outputs $\text{accept}$ or $\text{reject}$.

**Completeness.** For all $x \in L$:

$$\Pr[\langle P, V \rangle(x) = \text{accept}] \geq 1 - \text{negl}(|x|)$$

**Soundness.** For all $x \notin L$ and every (possibly cheating) prover $P^*$:

$$\Pr[\langle P^*, V \rangle(x) = \text{accept}] \leq \text{negl}(|x|)$$

**Zero-Knowledge.** $(P, V)$ is zero-knowledge if for every PPT verifier $V^*$, there exists a PPT simulator $S$ such that for all $x \in L$:

$$\text{View}_{V^*}[\langle P(x), V^*(x) \rangle] \approx S(x)$$

where $\approx$ denotes indistinguishability. The three variants are:

- **Perfect ZK:** the distributions are identical.
- **Statistical ZK:** the statistical distance is negligible: $\Delta(\text{View}, S(x)) \leq \text{negl}(|x|)$.
- **Computational ZK:** no PPT distinguisher can tell them apart with non-negligible advantage.

The simulator $S$ captures the intuition: anything $V^*$ learns from the interaction, $S$ can produce without access to the prover or the witness.

### Worked Examples

**Why the simulator must work for all $V^*$, not just honest $V$:**

Consider an honest verifier $V$ that sends truly random challenges. A simulator for $V$ alone might exploit the predictability of the challenge distribution. But a cheating $V^*$ could embed information in its challenges (e.g., set $e = H(a)$ for some function $H$). Zero-knowledge requires simulation even against such adversarial strategies.

**Distinguishing arguments from proofs:**

An interactive *proof* has soundness against unbounded provers. An interactive *argument* has soundness only against computationally bounded provers. SNARKs and STARKs are arguments, not proofs in the information-theoretic sense.

---

## 2. ZK Proof for Graph Isomorphism

### The Problem

Prove that the classic graph isomorphism zero-knowledge protocol satisfies completeness, soundness, and the zero-knowledge property.

### The Formula

Let $G_0, G_1$ be graphs on $n$ vertices. The prover knows an isomorphism $\varphi: G_0 \to G_1$.

**Protocol (single round):**

1. $P$ picks a random permutation $\pi \in S_n$, computes $H = \pi(G_0)$, sends $H$ to $V$.
2. $V$ picks $b \xleftarrow{R} \{0, 1\}$, sends $b$ to $P$.
3. $P$ computes and sends $\sigma$:
   - If $b = 0$: $\sigma = \pi$ (so $\sigma(G_0) = H$).
   - If $b = 1$: $\sigma = \pi \circ \varphi^{-1}$ (so $\sigma(G_1) = \pi(\varphi^{-1}(G_1)) = \pi(G_0) = H$).
4. $V$ checks $\sigma(G_b) = H$ and accepts.

**Completeness proof.** If the prover knows $\varphi$, then in both cases $\sigma(G_b) = H$, so $V$ always accepts.

**Soundness proof.** Suppose $G_0 \not\cong G_1$. Then $H$ is isomorphic to exactly one of $G_0, G_1$. A cheating prover can answer correctly for at most one value of $b$. Thus $\Pr[\text{cheat}] \leq 1/2$ per round. After $k$ rounds, $\Pr[\text{cheat}] \leq 2^{-k}$.

**Zero-knowledge proof (perfect ZK).** Construct simulator $S$ for any verifier $V^*$:

1. $S$ guesses $b' \xleftarrow{R} \{0, 1\}$.
2. $S$ picks random $\pi' \in S_n$, computes $H = \pi'(G_{b'})$.
3. $S$ sends $H$ to $V^*$ and receives $V^*$'s challenge $b$.
4. If $b = b'$: $S$ can respond correctly (it chose the right graph). Output the transcript.
5. If $b \neq b'$: $S$ rewinds $V^*$ and tries again from step 1.

Expected number of rewinds: 2 per successful round (since $\Pr[b = b'] = 1/2$). Conditioned on $b = b'$, the graph $H$ is a uniformly random permutation of $G_b$, exactly as in the real protocol. Hence the simulated transcript has identical distribution to the real one. This is **perfect** zero-knowledge.

### Worked Examples

For $n = 4$ with $G_0 = K_4$ (complete graph) and $G_1 = K_4$ (also complete):

$$\varphi = \begin{pmatrix} 1 & 2 & 3 & 4 \\ 3 & 1 & 4 & 2 \end{pmatrix}$$

Round: $P$ picks $\pi = (1\ 2\ 3\ 4 \mapsto 4\ 3\ 1\ 2)$, computes $H = \pi(G_0) = K_4$. Verifier sends $b = 1$. Prover computes $\sigma = \pi \circ \varphi^{-1}$:

$$\varphi^{-1} = \begin{pmatrix} 1 & 2 & 3 & 4 \\ 2 & 4 & 1 & 3 \end{pmatrix}, \quad \sigma = \pi \circ \varphi^{-1} = \begin{pmatrix} 1 & 2 & 3 & 4 \\ 3 & 2 & 4 & 1 \end{pmatrix}$$

Verifier checks: $\sigma(G_1) = H$. Since both are $K_4$, any permutation is an isomorphism. Accepts.

---

## 3. The Schnorr Protocol and Its Security Proof

### The Problem

Prove that the Schnorr identification protocol is a secure proof of knowledge for the discrete logarithm, satisfying special soundness and honest-verifier zero-knowledge.

### The Formula

**Setup.** Let $\mathbb{G}$ be a cyclic group of prime order $q$ with generator $g$. Public key $h = g^x$. The prover knows $x \in \mathbb{Z}_q$.

**Protocol:**

1. $P$ picks $r \xleftarrow{R} \mathbb{Z}_q$, sends $a = g^r$.
2. $V$ picks $e \xleftarrow{R} \mathbb{Z}_q$, sends $e$.
3. $P$ sends $z = r + ex \pmod{q}$.
4. $V$ checks $g^z = a \cdot h^e$.

**Completeness.**

$$g^z = g^{r + ex} = g^r \cdot g^{ex} = g^r \cdot (g^x)^e = a \cdot h^e$$

Holds unconditionally.

**Special Soundness.** Given two accepting transcripts $(a, e_1, z_1)$ and $(a, e_2, z_2)$ with $e_1 \neq e_2$ and the same first message $a$:

$$g^{z_1} = a \cdot h^{e_1} \quad \text{and} \quad g^{z_2} = a \cdot h^{e_2}$$

Dividing:

$$g^{z_1 - z_2} = h^{e_1 - e_2}$$

$$g^{z_1 - z_2} = g^{x(e_1 - e_2)}$$

$$x = \frac{z_1 - z_2}{e_1 - e_2} \pmod{q}$$

Since $q$ is prime and $e_1 \neq e_2$, the inverse $(e_1 - e_2)^{-1} \pmod{q}$ exists. This constitutes a **knowledge extractor**: given oracle access to a prover that succeeds on two different challenges with the same commitment, we can extract the witness $x$.

**Honest-Verifier Zero-Knowledge.** Simulator $S$:

1. Pick $z \xleftarrow{R} \mathbb{Z}_q$ and $e \xleftarrow{R} \mathbb{Z}_q$.
2. Compute $a = g^z \cdot h^{-e}$.
3. Output transcript $(a, e, z)$.

Verification: $g^z = g^z \cdot h^{-e} \cdot h^e = a \cdot h^e$. The transcript is accepting.

Distribution: In the real protocol, $r$ is uniform in $\mathbb{Z}_q$, so $a = g^r$ is uniform in $\mathbb{G}$. Challenge $e$ is uniform. Response $z = r + ex$ is uniform (since $r$ is uniform and independent of $e$). In the simulation, $z$ and $e$ are uniform, and $a$ is determined. The joint distribution $(a, e, z)$ subject to $g^z = a \cdot h^e$ is identical in both cases.

### Worked Examples

Concrete extraction with $q = 7$, $g = 3$, $x = 4$, $h = g^4 = 3^4 = 81 \equiv 4 \pmod{7}$ (in $\mathbb{Z}_7^*$, but using additive notation for $\mathbb{Z}_7$):

Working in $\mathbb{Z}_7$: two transcripts with same $a$:

- $(a, e_1 = 2, z_1 = 5)$: check $g^5 \stackrel{?}{=} a \cdot h^2$
- $(a, e_2 = 5, z_2 = 1)$: check $g^1 \stackrel{?}{=} a \cdot h^5$

Extract: $x = (z_1 - z_2)(e_1 - e_2)^{-1} = (5 - 1)(2 - 5)^{-1} = 4 \cdot (-3)^{-1} = 4 \cdot 4^{-1} \cdot (-1)^{-1}$...

More cleanly: $x = (5 - 1) \cdot (2 - 5)^{-1} \equiv 4 \cdot (-3)^{-1} \equiv 4 \cdot 4^{-1} \equiv 4 \cdot 2 \equiv 8 \equiv 1 \pmod{7}$.

(The specific values depend on the group; the algebraic extraction always works when $e_1 \neq e_2$.)

---

## 4. Fiat-Shamir in the Random Oracle Model

### The Problem

Transform interactive public-coin proofs into non-interactive arguments using a hash function, and establish security in the random oracle model.

### The Formula

**Fiat-Shamir transform.** Given a Sigma protocol $(P, V)$ with transcript $(a, e, z)$:

1. Prover computes commitment $a$.
2. Prover computes challenge $e = H(a \| x)$ where $H$ is a hash function and $x$ is the statement.
3. Prover computes response $z$.
4. Non-interactive proof $\pi = (a, z)$.

Verification: recompute $e = H(a \| x)$ and check the verification equation.

**Security in the Random Oracle Model (ROM).** Model $H$ as a uniformly random function $H: \{0,1\}^* \to \mathcal{E}$ where $\mathcal{E}$ is the challenge space.

**Soundness (ROM).** If the underlying Sigma protocol has special soundness, then the Fiat-Shamir transform produces a sound non-interactive argument in the ROM. Proof sketch:

- Suppose a PPT adversary $A$ produces a valid proof $(a^*, z^*)$ for a false statement $x^* \notin L$.
- In the ROM, $A$ must query $H$ on input $(a^* \| x^*)$ to learn the challenge.
- By the forking lemma (Pointcheval-Stern): rewind $A$ with different random oracle responses at the query point to obtain two accepting transcripts $(a^*, e_1, z_1)$ and $(a^*, e_2, z_2)$ with $e_1 \neq e_2$.
- By special soundness, extract the witness -- contradicting $x^* \notin L$.

**Zero-knowledge (ROM).** The simulator programs the random oracle:

1. $S$ picks $z, e$ at random, computes $a$ from the simulator of the Sigma protocol.
2. $S$ programs $H(a \| x) := e$.
3. $S$ outputs $\pi = (a, z)$.

Since $H$ is a random oracle, programming a single point is indistinguishable from a random function.

**Caveat.** Fiat-Shamir is *not* provably sound in the standard model (without random oracles). There exist contrived counterexamples (Goldwasser-Kalai 2003).

### Worked Examples

**Schnorr signature as Fiat-Shamir applied to Schnorr protocol:**

Sign message $m$ with secret key $x$, public key $h = g^x$:

1. Pick $r \xleftarrow{R} \mathbb{Z}_q$, compute $a = g^r$.
2. Compute $e = H(a \| m)$.
3. Compute $z = r + ex \pmod{q}$.
4. Signature: $\sigma = (e, z)$.

Verify: compute $a' = g^z \cdot h^{-e}$, check $e \stackrel{?}{=} H(a' \| m)$.

This is the Schnorr signature scheme, standardized as EdDSA (with Edwards curves).

---

## 5. Arithmetic Circuits and R1CS

### The Problem

Represent computations as arithmetic circuits over finite fields and encode them as Rank-1 Constraint Systems (R1CS), the standard intermediate representation for SNARKs.

### The Formula

An **arithmetic circuit** over a finite field $\mathbb{F}_p$ is a DAG where:

- Input nodes hold field elements (public inputs $x_1, \ldots, x_l$ and witness $w_1, \ldots, w_m$).
- Internal nodes are addition ($+$) or multiplication ($\times$) gates.
- The output wire carries the result.

**Flattening** converts an arithmetic expression into a sequence of constraints, each involving a single multiplication:

$$\text{Example: } y = x^3 + x + 5$$

Flatten:

$$s_1 = x \cdot x$$
$$s_2 = s_1 \cdot x$$
$$s_3 = s_2 + x + 5$$
$$y = s_3$$

**R1CS (Rank-1 Constraint System).** A system of $n$ constraints over a witness vector $\mathbf{s} = (1, x_1, \ldots, x_l, w_1, \ldots, w_m)$:

$$(\mathbf{a}_i \cdot \mathbf{s}) \times (\mathbf{b}_i \cdot \mathbf{s}) = (\mathbf{c}_i \cdot \mathbf{s}) \quad \text{for } i = 1, \ldots, n$$

where $\mathbf{a}_i, \mathbf{b}_i, \mathbf{c}_i$ are coefficient vectors. Each constraint captures one multiplication gate. Addition and scalar multiplication are "free" (absorbed into the linear combinations).

### Worked Examples

**$y = x^3 + x + 5$ with public input $x = 3$, output $y = 35$:**

Witness vector: $\mathbf{s} = (1, x, y, s_1, s_2) = (1, 3, 35, 9, 27)$.

Constraints:

| Constraint | $\mathbf{a}_i \cdot \mathbf{s}$ | $\mathbf{b}_i \cdot \mathbf{s}$ | $\mathbf{c}_i \cdot \mathbf{s}$ |
|---|---|---|---|
| $s_1 = x \cdot x$ | $x = 3$ | $x = 3$ | $s_1 = 9$ |
| $s_2 = s_1 \cdot x$ | $s_1 = 9$ | $x = 3$ | $s_2 = 27$ |
| $y = s_2 + x + 5$ | $s_2 + x + 5 = 35$ | $1 = 1$ | $y = 35$ |

Check: $3 \times 3 = 9$, $9 \times 3 = 27$, $35 \times 1 = 35$. All constraints satisfied.

---

## 6. QAP Construction

### The Problem

Transform an R1CS into a Quadratic Arithmetic Program (QAP), enabling efficient polynomial-based verification that is the core of pairing-based SNARKs.

### The Formula

Given R1CS with $n$ constraints and witness vector $\mathbf{s}$ of dimension $m+1$, construct polynomials over a field $\mathbb{F}_p$.

**Step 1.** Choose distinct evaluation points $r_1, \ldots, r_n \in \mathbb{F}_p$ (one per constraint).

**Step 2.** For each variable index $j \in \{0, \ldots, m\}$, define polynomials $A_j(x), B_j(x), C_j(x)$ of degree $\leq n-1$ via interpolation:

$$A_j(r_i) = a_{i,j}, \quad B_j(r_i) = b_{i,j}, \quad C_j(r_i) = c_{i,j}$$

where $a_{i,j}, b_{i,j}, c_{i,j}$ are the R1CS coefficient matrix entries.

**Step 3.** Define aggregate polynomials:

$$A(x) = \sum_{j=0}^{m} s_j \cdot A_j(x), \quad B(x) = \sum_{j=0}^{m} s_j \cdot B_j(x), \quad C(x) = \sum_{j=0}^{m} s_j \cdot C_j(x)$$

**Step 4.** The R1CS is satisfied if and only if:

$$A(x) \cdot B(x) - C(x) = H(x) \cdot Z(x)$$

where $Z(x) = \prod_{i=1}^{n} (x - r_i)$ is the **vanishing polynomial** and $H(x)$ is the quotient polynomial of degree $\leq n - 2$.

The key insight: checking $n$ constraint equations reduces to checking a single polynomial identity, which can be verified at a random point $\tau$ using polynomial commitments.

### Worked Examples

For the $y = x^3 + x + 5$ example with 3 constraints and evaluation points $r_1 = 1, r_2 = 2, r_3 = 3$:

$Z(x) = (x-1)(x-2)(x-3) = x^3 - 6x^2 + 11x - 6$

The polynomials $A_j(x)$ are degree-2 polynomials interpolating through the column $j$ values of the $\mathbf{a}$ matrix at points $1, 2, 3$. If $\mathbf{s}$ is a valid witness, then $A(\tau) \cdot B(\tau) - C(\tau)$ vanishes at $r_1, r_2, r_3$, hence is divisible by $Z(\tau)$.

---

## 7. Polynomial Commitment Schemes

### The Problem

Enable a prover to commit to a polynomial and later prove evaluations of that polynomial at chosen points, without revealing the polynomial itself. This is the cryptographic primitive at the heart of modern ZK proof systems.

### The Formula

A **polynomial commitment scheme** consists of:

- $\text{Setup}(d) \to \text{srs}$: generate structured reference string for degree-$d$ polynomials.
- $\text{Commit}(\text{srs}, p(x)) \to C$: commit to polynomial $p(x)$.
- $\text{Open}(\text{srs}, p(x), z) \to (v, \pi)$: prove $p(z) = v$ with proof $\pi$.
- $\text{Verify}(\text{srs}, C, z, v, \pi) \to \{0, 1\}$: verify the evaluation proof.

**KZG commitments (Kate-Zaverucha-Goldberg, 2010):**

Setup: trusted party generates $\text{srs} = (g, g^\tau, g^{\tau^2}, \ldots, g^{\tau^d})$ for secret $\tau$.

Commit: for $p(x) = \sum_{i=0}^{d} c_i x^i$, compute $C = g^{p(\tau)} = \prod_{i} (g^{\tau^i})^{c_i}$.

Open at $z$: compute quotient $q(x) = \frac{p(x) - p(z)}{x - z}$, proof $\pi = g^{q(\tau)}$.

Verify: check pairing equation $e(\pi, g^\tau \cdot g^{-z}) = e(C \cdot g^{-v}, g)$.

This works because $e(g^{q(\tau)}, g^{\tau - z}) = e(g^{q(\tau)(\tau - z)}, g) = e(g^{p(\tau) - v}, g)$.

**Properties:** commitment is a single group element, proof is a single group element, verification is two pairings. Requires trusted setup for $\tau$.

**Alternatives:**

| Scheme | Setup | Proof Size | Verify Time | Assumption |
|---|---|---|---|---|
| KZG | Trusted | $O(1)$ | $O(1)$ pairings | $q$-SDH |
| FRI | Transparent | $O(\log^2 d)$ | $O(\log^2 d)$ | CRHF |
| Bulletproofs IPA | Transparent | $O(\log d)$ | $O(d)$ | DLOG |
| DARK | Transparent | $O(1)$ | $O(1)$ | Strong RSA |

---

## 8. The FRI Protocol

### The Problem

Establish a transparent (no trusted setup) method for proving that a committed function is close to a low-degree polynomial, serving as the core of STARK proof systems.

### The Formula

**FRI (Fast Reed-Solomon Interactive Oracle Proof of Proximity)** proves that a function $f: D \to \mathbb{F}$ evaluated on a domain $D$ (with $|D| = n$) is $\delta$-close to a polynomial of degree $< d$.

**Setup.** Let $D$ be a multiplicative subgroup of $\mathbb{F}$ with $|D| = 2^k$ and $d < |D|$. The function $f$ is committed via a Merkle tree of its evaluations on $D$.

**FRI folding (one round):**

1. View $f(x)$ as $f(x) = f_{\text{even}}(x^2) + x \cdot f_{\text{odd}}(x^2)$, where:
   - $f_{\text{even}}(y) = \sum_{i \text{ even}} c_i \cdot y^{i/2}$
   - $f_{\text{odd}}(y) = \sum_{i \text{ odd}} c_i \cdot y^{(i-1)/2}$

2. Verifier sends random $\alpha \xleftarrow{R} \mathbb{F}$.

3. Prover computes $f'(y) = f_{\text{even}}(y) + \alpha \cdot f_{\text{odd}}(y)$.
   - $\deg(f') \leq \lfloor \deg(f)/2 \rfloor$
   - $f'$ is defined on the "squared" domain $D' = \{x^2 : x \in D\}$, with $|D'| = |D|/2$.

4. Prover commits to $f'$ on $D'$ via Merkle tree.

**Consistency checks.** Verifier queries $f$ at a random point $x_0 \in D$ and its conjugate $-x_0 \in D$, then checks:

$$f'(x_0^2) = \frac{f(x_0) + f(-x_0)}{2} + \alpha \cdot \frac{f(x_0) - f(-x_0)}{2x_0}$$

**Recursion.** Repeat for $\log_2(d)$ rounds until the polynomial is a constant. Final round: prover sends the constant, verifier checks directly.

**Soundness.** If $f$ is $\delta$-far from degree-$d$ polynomials, then with high probability over $\alpha$, the folded function $f'$ is far from degree-$\lfloor d/2 \rfloor$ polynomials. After $\log_2(d)$ rounds, either all consistency checks pass (and $f$ is close to low-degree) or the prover is caught cheating.

**Complexity:**

- Prover time: $O(n \log n)$ (dominated by $\log d$ NTT operations)
- Proof size: $O(n \cdot \log^2 d / n) = O(\log^2 d)$ (Merkle authentication paths)
- Verifier time: $O(\log^2 d)$

### Worked Examples

Simplified FRI over $\mathbb{F}_p$ with $d = 4$, domain $D$ of size 8:

Round 1: $f(x) = c_0 + c_1 x + c_2 x^2 + c_3 x^3$, degree 3. Split:
- $f_{\text{even}}(y) = c_0 + c_2 y$
- $f_{\text{odd}}(y) = c_1 + c_3 y$
- Verifier sends $\alpha_1$. Prover computes $f'(y) = (c_0 + \alpha_1 c_1) + (c_2 + \alpha_1 c_3) y$, degree 1 on $|D'| = 4$.

Round 2: $f'(y) = d_0 + d_1 y$, degree 1. Split:
- $f'_{\text{even}}(z) = d_0$
- $f'_{\text{odd}}(z) = d_1$
- Verifier sends $\alpha_2$. Prover sends constant $c = d_0 + \alpha_2 d_1$.

Verifier performs consistency checks at random query points in each round.

---

## 9. Comparison: SNARKs vs STARKs vs Bulletproofs

### The Problem

Provide a rigorous comparison of the three major zero-knowledge proof systems along all practically relevant axes.

### The Formula

| Property | Groth16 (SNARK) | STARK | Bulletproofs |
|---|---|---|---|
| **Proof size** | 128 bytes (3 $\mathbb{G}_1$ + 1 $\mathbb{G}_2$) | ~45-200 KB ($O(\log^2 n)$) | ~1-2 KB ($O(\log n)$) |
| **Prover time** | $O(n \log n)$ | $O(n \log n)$ | $O(n)$ |
| **Verifier time** | $O(1)$ (3 pairings) | $O(\log^2 n)$ | $O(n)$ (linear) |
| **Trusted setup** | Required (per-circuit) | None (transparent) | None (transparent) |
| **Post-quantum** | No (pairing/DL) | Yes (hash-based) | No (DL) |
| **Assumption** | $q$-SDH, knowledge of exponent | CRHF | DLOG |
| **Universal** | No (circuit-specific SRS) | Yes | Yes |
| **Recursion** | Via cycles of curves | Via FRI | Difficult |
| **Batch verify** | Yes (amortized pairings) | Yes | Yes (amortized) |

**When to use each:**

- **Groth16 SNARKs:** On-chain verification where gas cost dominates (Zcash, Ethereum L1 verification). Smallest proof size and fastest verification.
- **STARKs:** When trust assumptions matter (no trusted setup), post-quantum security is desired, or proofs involve very large computations (zkEVM, StarkNet). Tolerate larger proofs.
- **Bulletproofs:** Range proofs and simple statements where proof size matters more than verification speed, and no trusted setup is acceptable (Monero). Not suited for general computation at scale due to linear verification.

### Worked Examples

**Cost comparison for a circuit of $n = 2^{20}$ ($\approx 10^6$) constraints on Ethereum:**

- Groth16: proof = 128 bytes, on-chain verify = ~230K gas (~$0.50 at 20 gwei), constant regardless of $n$.
- STARK: proof $\approx$ 100 KB, on-chain verify = ~2M gas (~$5), scales with $\log^2 n$.
- Bulletproofs: proof $\approx$ 1.5 KB, on-chain verify = $O(n)$ field operations -- impractical for $n = 10^6$ on-chain.

This explains why Ethereum L1 ZK-rollups predominantly use SNARKs (or STARK proofs verified via a SNARK wrapper for smaller on-chain footprint).

---

## 10. Groth16 Protocol Overview

### The Problem

Describe the structure of the Groth16 SNARK, the most widely deployed pairing-based zero-knowledge proof system.

### The Formula

**Trusted setup.** For a QAP with polynomials $\{A_j, B_j, C_j\}_{j=0}^{m}$, vanishing polynomial $Z(x)$, and toxic waste $\tau, \alpha, \beta, \gamma, \delta \xleftarrow{R} \mathbb{F}_p$:

Generate the structured reference string (SRS):

$$\text{Proving key: } \{g_1^{\tau^i}\}, \{g_2^{\tau^i}\}, \{g_1^{\alpha}, g_1^{\beta}, g_2^{\beta}\}, \left\{g_1^{\frac{\beta A_j(\tau) + \alpha B_j(\tau) + C_j(\tau)}{\gamma}}\right\}_{j \in \text{pub}}, \left\{g_1^{\frac{\beta A_j(\tau) + \alpha B_j(\tau) + C_j(\tau)}{\delta}}\right\}_{j \in \text{priv}}, \left\{g_1^{\frac{\tau^i Z(\tau)}{\delta}}\right\}$$

$$\text{Verification key: } g_1^\alpha, g_2^\beta, g_2^\gamma, g_2^\delta, \left\{g_1^{\frac{\beta A_j(\tau) + \alpha B_j(\tau) + C_j(\tau)}{\gamma}}\right\}_{j \in \text{pub}}$$

**Proof generation.** Given witness $\mathbf{s}$ and random $r, s \xleftarrow{R} \mathbb{F}_p$:

$$\pi_A = g_1^{A(\tau) + \alpha + r\delta}, \quad \pi_B = g_2^{B(\tau) + \beta + s\delta}, \quad \pi_C = g_1^{\frac{A(\tau)B(\tau) - C(\tau)}{\delta} + \frac{H(\tau)Z(\tau)}{\delta} + s\pi_A + r\pi_B - rs\delta}$$

Proof: $\pi = (\pi_A, \pi_B, \pi_C)$ -- three group elements.

**Verification.** Given public inputs $(x_1, \ldots, x_l)$, compute:

$$L = g_1^{\sum_{j=0}^{l} x_j \cdot \frac{\beta A_j(\tau) + \alpha B_j(\tau) + C_j(\tau)}{\gamma}}$$

Check the pairing equation:

$$e(\pi_A, \pi_B) = e(g_1^\alpha, g_2^\beta) \cdot e(L, g_2^\gamma) \cdot e(\pi_C, g_2^\delta)$$

Three pairings, constant time, independent of circuit size.

### Worked Examples

The toxic waste $\tau$ must be destroyed after setup. In practice, multi-party computation (MPC) ceremonies ensure that $\tau$ is unknown to any single party:

- Zcash Sprout: 6-party ceremony (2016).
- Zcash Sapling: 90-party "Powers of Tau" ceremony (2018).
- Perpetual Powers of Tau: open, ongoing ceremony where anyone can contribute.

If any single participant destroys their share, the overall $\tau$ is unrecoverable, ensuring soundness.

---

## See Also

- computational-complexity
- cryptography
- number-theory
- graph-theory
- boolean-satisfiability

## References

```
Goldwasser, Micali, Rackoff. "The Knowledge Complexity of Interactive Proof Systems." STOC 1985; SICOMP 1989.
Goldreich, Micali, Wigderson. "Proofs that Yield Nothing But Their Validity." JACM 1991.
Schnorr. "Efficient Signature Generation by Smart Cards." J. Cryptology 4(3), 1991.
Fiat, Shamir. "How to Prove Yourself: Practical Solutions to Identification and Signature Problems." CRYPTO 1986.
Kate, Zaverucha, Goldberg. "Constant-Size Commitments to Polynomials and Their Applications." ASIACRYPT 2010.
Groth. "On the Size of Pairing-Based Non-Interactive Arguments." EUROCRYPT 2016.
Ben-Sasson, Bentov, Horesh, Riabzev. "Scalable, Transparent, and Post-Quantum Secure Computational Integrity." 2018.
Bunz, Bootle, Boneh, Poelstra, Wuille, Maxwell. "Bulletproofs: Short Proofs for Confidential Transactions and More." IEEE S&P 2018.
Thaler. "Proofs, Arguments, and Zero-Knowledge." Foundations and Trends in Privacy and Security, 2022.
Pointcheval, Stern. "Security Arguments for Digital Signatures and Blind Signatures." J. Cryptology 13(3), 2000.
```
