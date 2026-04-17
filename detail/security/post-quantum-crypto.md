# The Mathematics of Post-Quantum Cryptographic Migration — From Shor's Threat to FIPS-Standardized Defense

> *Quantum computers do not break cryptography by brute force — they break it by reducing integer factorization and discrete logarithms to polynomial time through Shor's algorithm. Post-quantum cryptography replaces these number-theoretic assumptions with lattice, hash, and code-based problems that remain hard under quantum models. The migration is not optional but quantitative: HNDL exposure, parameter selection, and hybrid composition all reduce to computable risk.*

---

## 1. Shor's Algorithm — Why RSA and ECC Fall

### The Problem

Classical cryptography rests on the hardness of factoring $N = pq$ (RSA) and solving $g^x = y \bmod p$ or its elliptic-curve analog (DH/ECDSA). Peter Shor (1994) showed both reduce to order-finding on a quantum computer, solvable in polynomial time.

### The Formula

For factoring $N$, pick random $a < N$ with $\gcd(a, N) = 1$. Find the order $r$ such that:

$$a^r \equiv 1 \pmod{N}$$

If $r$ is even and $a^{r/2} \not\equiv -1 \pmod{N}$:

$$\gcd(a^{r/2} - 1, N) \text{ yields a nontrivial factor of } N$$

Classical order-finding is exponential. Shor's quantum algorithm computes $r$ in $O((\log N)^3)$ by:

1. Prepare superposition $\sum_{x=0}^{Q-1} |x\rangle |a^x \bmod N\rangle$
2. Measure second register (collapses to cosets separated by $r$)
3. Quantum Fourier transform on first register reveals $r$ via frequency peaks

Required logical qubits for RSA-2048: approximately $2n + 3 \approx 4099$ where $n = 2048$. Physical qubit requirement accounting for error correction: $\sim 20$ million noisy qubits (Gidney & Ekerå, 2019).

### Worked Example

Factoring $N = 15$ with Shor's algorithm (toy example):
- Choose $a = 7$. Check $\gcd(7, 15) = 1$.
- Order: $7^1 = 7, 7^2 = 49 \equiv 4, 7^3 \equiv 13, 7^4 \equiv 1 \pmod{15}$, so $r = 4$.
- $r$ even, $a^{r/2} = 49 \equiv 4 \not\equiv -1$, so:
- $\gcd(4 - 1, 15) = \gcd(3, 15) = 3$ — nontrivial factor found.

This is polynomial in $\log N$ on a quantum computer. No classical algorithm achieves sub-exponential complexity.

### Why It Matters

Cryptographically Relevant Quantum Computers (CRQC) are estimated at 10–30 years away, but **data encrypted today** under RSA/ECDH is recorded now for decryption then. This is the Harvest-Now-Decrypt-Later threat model, and it is already active for nation-state adversaries.

---

## 2. Module Learning With Errors — The Security of ML-KEM

### The Problem

ML-KEM (FIPS 203, derived from CRYSTALS-Kyber) bases security on the Module-LWE problem: distinguishing noisy linear equations over polynomial rings from uniform randomness.

### The Formula

Let $R_q = \mathbb{Z}_q[x]/(x^n + 1)$ with $q = 3329$, $n = 256$.

Secret $\mathbf{s} \in R_q^k$ with small coefficients (centered binomial distribution).
Public matrix $\mathbf{A} \in R_q^{k \times k}$ uniformly random.
Error $\mathbf{e} \in R_q^k$ small.

Public key: $\mathbf{b} = \mathbf{A}\mathbf{s} + \mathbf{e}$

Module-LWE assumption: $(\mathbf{A}, \mathbf{A}\mathbf{s} + \mathbf{e})$ is computationally indistinguishable from uniform $(\mathbf{A}, \mathbf{u})$.

Encapsulation: sender picks $\mathbf{r}, \mathbf{e}_1, \mathbf{e}_2$ small, computes:

$$\mathbf{u} = \mathbf{A}^T \mathbf{r} + \mathbf{e}_1, \quad v = \mathbf{b}^T \mathbf{r} + e_2 + \lceil q/2 \rceil \cdot m$$

Recipient decapsulates via $v - \mathbf{s}^T \mathbf{u} \approx \lceil q/2 \rceil \cdot m$ (small noise, rounded).

### Security Parameters

| Parameter | $k$ | Classical security | Quantum security |
|-----------|---|---|---|
| ML-KEM-512 | 2 | $\approx 2^{143}$ | $\approx 2^{130}$ |
| ML-KEM-768 | 3 | $\approx 2^{207}$ | $\approx 2^{188}$ |
| ML-KEM-1024 | 4 | $\approx 2^{272}$ | $\approx 2^{247}$ |

### Worked Example

ML-KEM-768 ciphertext decapsulation failure probability $\delta \approx 2^{-164}$. For a world generating $10^{18}$ handshakes annually, expected failures:

$$E = 10^{18} \cdot 2^{-164} \approx 4.3 \cdot 10^{-32}$$

Vanishingly unlikely. IND-CCA2 security holds under the Module-LWE assumption at post-quantum parameters.

### Why It Matters

The best known quantum attack on Module-LWE (primal/dual BKZ lattice reduction with quantum Grover speedup) gives only polynomial improvement. This is the fundamental reason the lattice-based approach survives quantum adversaries.

---

## 3. Fiat-Shamir with Aborts — ML-DSA Signature Construction

### The Problem

Digital signatures from lattices must hide the secret key while providing non-repudiation. ML-DSA uses a "Fiat-Shamir with aborts" transformation that rejects signatures leaking secret information.

### The Formula

Key generation:
- Matrix $\mathbf{A} \in R_q^{k \times \ell}$
- Secrets $\mathbf{s}_1 \in R_q^\ell$, $\mathbf{s}_2 \in R_q^k$ small
- $\mathbf{t} = \mathbf{A} \mathbf{s}_1 + \mathbf{s}_2$

Signing (sketch):
1. Sample mask $\mathbf{y}$ from narrow distribution
2. Compute $\mathbf{w} = \mathbf{A} \mathbf{y}$, extract high bits $\mathbf{w}_1$
3. Challenge $c = H(\mu \| \mathbf{w}_1)$ where $\mu$ hashes message
4. $\mathbf{z} = \mathbf{y} + c \mathbf{s}_1$
5. **Rejection**: if $\|\mathbf{z}\|_\infty \geq \gamma_1 - \beta$ or $\|c \mathbf{s}_2\|_\infty \geq \gamma_2 - \beta$, **restart**

Rejection probability tuned so $\mathbf{z}$ is statistically close to uniform over a bounded set, leaking zero information about $\mathbf{s}_1$.

Verify: reconstruct $\mathbf{w}_1' = \text{HighBits}(\mathbf{A}\mathbf{z} - c\mathbf{t})$ and check $c = H(\mu \| \mathbf{w}_1')$.

### Worked Example

ML-DSA-65: typical rejection rate ~4, so each signature requires ~4 mask samples on average. At $\sigma_{\text{sign}} = 0.5$ ms per attempt on modern CPU, mean signing time $\approx 2$ ms.

### Why It Matters

Rejection sampling is the reason ML-DSA signatures are safe to publish even though they involve secret keys. Without it, every signature would leak a linear constraint on $\mathbf{s}_1$, reconstructing the full secret after $\sim \ell$ signatures.

---

## 4. Hybrid Key Exchange — Composite Security Proof

### The Problem

During migration, we need a key exchange that is secure if EITHER the classical primitive OR the PQC primitive holds. This hedges against unexpected breaks in new lattice assumptions.

### The Formula

Hybrid shared secret construction:

$$K = \text{KDF}(K_{\text{classical}} \| K_{\text{pq}} \| \text{transcript})$$

Where $K_{\text{classical}}$ comes from X25519 ECDH and $K_{\text{pq}}$ from ML-KEM-768.

Security theorem (informal): the hybrid KEM is IND-CCA2 secure if AT LEAST ONE component KEM is IND-CCA2 secure, under the assumption the KDF is a random oracle.

Proof sketch: suppose an adversary $\mathcal{A}$ distinguishes hybrid output from random with advantage $\epsilon$. Build reduction $\mathcal{B}$ against classical: on challenge $(pk, ct, K^*)$ where $K^* \in \{K_{\text{real}}, K_{\text{random}}\}$, generate PQ KEM honestly, form hybrid $K = \text{KDF}(K^* \| K_{\text{pq}} \| \text{tr})$. If $\mathcal{A}$ distinguishes with $\epsilon$, so does $\mathcal{B}$ — contradicting classical security. Symmetric argument for PQ side.

### Worked Example

TLS 1.3 handshake with X25519MLKEM768:

- X25519 contributes 32 bytes → $K_{\text{classical}}$
- ML-KEM-768 contributes 32 bytes → $K_{\text{pq}}$
- HKDF-Extract with concatenated seed:

$$\text{HKDF-Extract}(\text{salt}=0, \text{IKM} = K_{\text{classical}} \| K_{\text{pq}})$$

Handshake size overhead: ML-KEM-768 adds 1184 (pk) + 1088 (ct) = 2272 bytes to the first two flights. Typically 1–2 extra TCP packets; negligible latency impact on modern networks.

### Why It Matters

Hybrid deployments let us migrate without betting the farm on any single hard problem. If a lattice attack emerges tomorrow, hybrid connections remain secure via classical; if a CRQC arrives, hybrid connections remain secure via PQC.

---

## 5. Hash-Based Signatures — SLH-DSA Security Bounds

### The Problem

SLH-DSA (FIPS 205) relies solely on the security of cryptographic hash functions — the most conservative assumption available. Unlike lattice schemes, its security does NOT depend on any structured math problem.

### The Formula

SLH-DSA builds a hyper-tree of depth $d$ with Merkle trees of height $h/d$. One-time signatures (WOTS+) sign messages once; FORS (Forest Of Random Subsets) signs hash outputs. Overall structure is stateless.

Security reduces to (multi-target) second-preimage resistance of the hash function. For $n$-bit hash, classical security is $2^n$; quantum security via Grover is $2^{n/2}$.

Signature size:

$$|\sigma| = n \cdot (h + k \cdot (a + 1) + d \cdot \text{len}_{\text{WOTS+}})$$

Where $h, k, a, d, w$ are parameters chosen per security level.

### Parameter Comparison

| Parameter | Hash | $h$ | $d$ | $k$ | Signature size | Security |
|-----------|------|-----|-----|-----|----------------|----------|
| SLH-DSA-SHA2-128s | SHA-256 | 63 | 7 | 14 | 7856 | 128-bit |
| SLH-DSA-SHA2-128f | SHA-256 | 66 | 22 | 33 | 17088 | 128-bit |
| SLH-DSA-SHA2-256s | SHA-256 | 64 | 8 | 22 | 29792 | 256-bit |

"s" variants minimize signature size at the cost of slow signing; "f" variants are fast signing with larger signatures.

### Worked Example

A firmware image signed with SLH-DSA-SHA2-128s:
- Signature: 7856 bytes
- Verify time on ARM Cortex-M4: approximately 50 ms
- Sign time on x86 server: approximately 2 seconds

For a firmware update shipped once per month to a million devices, total verification compute = 50,000 seconds = 14 CPU-hours distributed across devices. Acceptable.

### Why It Matters

SLH-DSA is the "break glass" option. If a catastrophic attack on lattices is discovered, SLH-DSA continues to work as long as SHA-256/SHA-3 remain preimage-resistant — the most-studied assumption in all of cryptography.

---

## 6. Migration Risk Calculus — The HNDL Equation

### The Problem

Not all encrypted traffic needs PQC protection immediately. Prioritization requires quantifying exposure: what is the probability a CRQC arrives before data loses confidentiality value?

### The Formula

Mosca's theorem (Michele Mosca, 2015): if

$$X + Y > Z$$

Then you have already lost. Where:

- $X$ = years data must remain confidential
- $Y$ = years required to migrate cryptography
- $Z$ = years until CRQC arrives

Equivalently, start migrating when $X + Y \geq Z$. For most organizations today, $Y \geq 5$ years and $Z$ estimates range 10–30 years — so any data with $X \geq 5$ must migrate NOW.

Expected information loss:

$$E[L] = \sum_{t=0}^{T} P(\text{CRQC exists at } t) \cdot V(t)$$

Where $V(t)$ is the value of data still confidential at time $t$.

### Worked Example

Healthcare records: $X = 80$ (patient lifetime), $Y = 5$, $Z = 20$.

$$X + Y = 85 > Z = 20$$

Healthcare records encrypted today under RSA/ECDH are already at risk. Migration is overdue, not upcoming.

Contrast: ephemeral session tokens with $X = 0.001$ (one hour): $X + Y = 5 < Z$ — not an HNDL target, so migration can follow standard upgrade cadence.

### Why It Matters

Mosca's inequality turns executive hand-waving about "quantum threat someday" into a concrete business prioritization metric. Data classification programs can now assign HNDL priority scores directly.

---

## 7. Synthesis — The Transition as Engineering Discipline

Post-quantum migration is not a single switch but a coordinated program across layers:

| Layer | Migration action | Metric |
|-------|------------------|--------|
| Transport (TLS) | Deploy hybrid X25519MLKEM768 | % of connections PQC-hybrid |
| VPN/IPsec | Enable IKEv2 RFC 9370 + ML-KEM | % of tunnels hybrid |
| PKI | Issue ML-DSA roots & subordinates | % of chains PQC-signable |
| Code signing | SLH-DSA or ML-DSA + classical composite | % of releases composite-signed |
| Firmware | SLH-DSA for long-term roots | Device coverage |
| At-rest | AES-256 remains safe; migrate KEKs | KEK algorithm inventory |

Every classical primitive has a PQC-standardized replacement. Every replacement has quantified parameters. Every parameter choice is a trade-off between size, speed, and security level.

The organizations that treat PQC migration as a measurable multi-year program will emerge with crypto-agile infrastructure; those that wait for a "flag day" will face decade-long emergency projects under regulatory pressure. The math is clear; the clock is already running.

---
