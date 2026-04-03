# The Mathematics of Cryptography — From Algebra to Algorithms

> *Modern cryptography rests on three pillars: symmetric ciphers (AES), asymmetric ciphers (RSA/ECC), and hash functions. Each is grounded in different branches of mathematics — finite field arithmetic, number theory, and elliptic curves.*

---

## 1. AES — Advanced Encryption Standard (Rijndael)

### Block Cipher Structure

AES operates on a $4 \times 4$ matrix of bytes (128-bit block) called the **state**. Each round applies four transformations:

| Step | Operation | Mathematical Basis |
|:---:|:---|:---|
| 1 | SubBytes | Multiplicative inverse in $GF(2^8)$ + affine transform |
| 2 | ShiftRows | Cyclic permutation |
| 3 | MixColumns | Matrix multiplication in $GF(2^8)$ |
| 4 | AddRoundKey | XOR with round key |

### Round Counts by Key Size

| Key Size | Rounds | Block Size | Key Schedule Words |
|:---:|:---:|:---:|:---:|
| AES-128 | 10 | 128 bits | 44 |
| AES-192 | 12 | 128 bits | 52 |
| AES-256 | 14 | 128 bits | 60 |

### SubBytes — The S-Box

Each byte $b$ is replaced by its multiplicative inverse in $GF(2^8)$ (the Galois field with $2^8 = 256$ elements), then an affine transformation is applied:

$$b' = M \cdot b^{-1} + c$$

Where $M$ is a fixed $8 \times 8$ binary matrix and $c = \text{0x63}$.

The irreducible polynomial defining $GF(2^8)$:

$$m(x) = x^8 + x^4 + x^3 + x + 1$$

### MixColumns — Matrix Multiplication

Each column is multiplied by a fixed matrix over $GF(2^8)$:

$$\begin{bmatrix} 2 & 3 & 1 & 1 \\ 1 & 2 & 3 & 1 \\ 1 & 1 & 2 & 3 \\ 3 & 1 & 1 & 2 \end{bmatrix} \begin{bmatrix} s_0 \\ s_1 \\ s_2 \\ s_3 \end{bmatrix} = \begin{bmatrix} s'_0 \\ s'_1 \\ s'_2 \\ s'_3 \end{bmatrix}$$

Multiplication by 2 in $GF(2^8)$: left-shift and conditional XOR with $\text{0x1B}$ if the high bit was set.

### Brute Force Complexity

$$\text{AES-128 keyspace} = 2^{128} = 3.4 \times 10^{38} \text{ keys}$$

Best known attack on AES-128: **biclique** reduces to $2^{126.1}$ — still computationally infeasible.

---

## 2. RSA — Rivest-Shamir-Adleman

### Key Generation

1. Choose two large primes $p, q$ (each ~1536 bits for RSA-3072)
2. Compute $n = p \cdot q$ (the modulus)
3. Compute $\phi(n) = (p-1)(q-1)$ (Euler's totient)
4. Choose $e$ such that $1 < e < \phi(n)$ and $\gcd(e, \phi(n)) = 1$ (typically $e = 65537$)
5. Compute $d = e^{-1} \pmod{\phi(n)}$ (extended Euclidean algorithm)

**Public key:** $(n, e)$, **Private key:** $(n, d)$

### Encryption and Decryption

$$C = M^e \pmod{n} \quad \text{(encrypt)}$$
$$M = C^d \pmod{n} \quad \text{(decrypt)}$$

**Correctness proof** (Euler's theorem): Since $ed \equiv 1 \pmod{\phi(n)}$:

$$C^d = (M^e)^d = M^{ed} = M^{k\phi(n) + 1} = (M^{\phi(n)})^k \cdot M \equiv 1^k \cdot M \equiv M \pmod{n}$$

### Factoring Difficulty

The security of RSA depends on the hardness of factoring $n = pq$.

Best known classical algorithm: **General Number Field Sieve (GNFS)**:

$$T_{GNFS} = O\left(\exp\left(1.923 \cdot (\ln n)^{1/3} \cdot (\ln \ln n)^{2/3}\right)\right)$$

### Worked Example (Small)

Let $p = 61, q = 53$:

$$n = 61 \times 53 = 3233$$
$$\phi(n) = 60 \times 52 = 3120$$
$$e = 17 \quad (\gcd(17, 3120) = 1)$$
$$d = 17^{-1} \pmod{3120} = 2753$$

Encrypt $M = 65$: $C = 65^{17} \pmod{3233} = 2790$

Decrypt: $M = 2790^{2753} \pmod{3233} = 65$ ✓

---

## 3. Elliptic Curve Cryptography (ECC)

### Curve Definition

An elliptic curve over a prime field $\mathbb{F}_p$:

$$y^2 \equiv x^3 + ax + b \pmod{p}, \quad 4a^3 + 27b^2 \neq 0$$

### Point Addition

Given two points $P = (x_1, y_1)$ and $Q = (x_2, y_2)$ on the curve, $R = P + Q = (x_3, y_3)$:

$$\lambda = \frac{y_2 - y_1}{x_2 - x_1} \pmod{p}$$
$$x_3 = \lambda^2 - x_1 - x_2 \pmod{p}$$
$$y_3 = \lambda(x_1 - x_3) - y_1 \pmod{p}$$

### Point Doubling ($P = Q$)

$$\lambda = \frac{3x_1^2 + a}{2y_1} \pmod{p}$$

### Scalar Multiplication

$$k \cdot P = \underbrace{P + P + \cdots + P}_{k \text{ times}}$$

Efficient via **double-and-add** algorithm: $O(\log k)$ operations instead of $O(k)$.

### NIST Key Length Equivalences

| Symmetric | RSA/DH | ECC | Security Level |
|:---:|:---:|:---:|:---:|
| AES-128 | RSA-3072 | ECC-256 | 128-bit |
| AES-192 | RSA-7680 | ECC-384 | 192-bit |
| AES-256 | RSA-15360 | ECC-521 | 256-bit |

### Why ECC Wins

Key size for equivalent security:

$$\text{RSA key size} \approx \frac{(\text{ECC key size})^2}{2}$$

RSA-3072 certificate: ~400 bytes. ECC-256 certificate: ~130 bytes. 3x bandwidth savings.

---

## 4. Hash Functions — One-Way Compression

### Properties (Formal)

| Property | Definition | Complexity |
|:---|:---|:---:|
| Preimage resistance | Given $h$, find $m$: $H(m) = h$ | $O(2^n)$ |
| Second preimage | Given $m_1$, find $m_2 \neq m_1$: $H(m_1) = H(m_2)$ | $O(2^n)$ |
| Collision resistance | Find any $m_1 \neq m_2$: $H(m_1) = H(m_2)$ | $O(2^{n/2})$ |

The collision bound $O(2^{n/2})$ comes from the **birthday paradox**:

$$P(\text{collision among } k \text{ hashes}) \approx 1 - e^{-k^2/(2 \cdot 2^n)}$$

For 50% collision probability: $k \approx 1.177 \sqrt{2^n} = 1.177 \cdot 2^{n/2}$

### Hash Comparison

| Algorithm | Output | Collision Resistance | Status |
|:---|:---:|:---:|:---|
| MD5 | 128 bits | $2^{18}$ (broken) | Deprecated |
| SHA-1 | 160 bits | $2^{63}$ (broken) | Deprecated |
| SHA-256 | 256 bits | $2^{128}$ | Current standard |
| SHA-3-256 | 256 bits | $2^{128}$ | Alternative standard |
| BLAKE3 | 256 bits | $2^{128}$ | Fastest secure hash |

---

## 5. Modes of Operation — Block to Stream

### ECB Failure (Why Modes Matter)

ECB encrypts identical plaintext blocks to identical ciphertext blocks:

$$C_i = E_K(P_i)$$

This leaks patterns. The famous "ECB penguin" demonstrates this visually.

### CBC — Cipher Block Chaining

$$C_i = E_K(P_i \oplus C_{i-1}), \quad C_0 = \text{IV}$$

Requires random IV. Sequential (not parallelizable). Vulnerable to padding oracle attacks (POODLE).

### CTR — Counter Mode

$$C_i = P_i \oplus E_K(\text{nonce} \| i)$$

Turns block cipher into stream cipher. Fully parallelizable. No padding needed.

### GCM — Galois/Counter Mode (AEAD)

Combines CTR encryption with GHASH authentication:

$$\text{Tag} = \text{GHASH}_H(A, C) \oplus E_K(\text{nonce} \| 0^{31} \| 1)$$

Where $H = E_K(0^{128})$ is the hash key and GHASH is polynomial evaluation over $GF(2^{128})$.

---

## 6. Key Space and Entropy

### Entropy Formula

$$H = \log_2(\text{keyspace})$$

| Key Type | Keyspace | Entropy (bits) |
|:---|:---|:---:|
| AES-128 | $2^{128}$ | 128 |
| RSA-2048 (primes) | $\approx 2^{1024}$ per prime | ~2048 total |
| 256-bit ECC private key | $2^{256}$ | 256 |
| 8-char password (printable ASCII) | $95^8 = 6.63 \times 10^{15}$ | 52.6 |
| 12-word BIP39 mnemonic | $2^{128}$ | 128 |

### Grover's Quantum Threat

A quantum computer running Grover's algorithm reduces symmetric key security by half:

$$\text{Quantum search} = O(2^{n/2})$$

| Classical | Post-Quantum Effective |
|:---:|:---:|
| AES-128 → 64-bit | Insufficient |
| AES-256 → 128-bit | Still secure |

This is why NIST recommends AES-256 for quantum-resistant symmetric encryption.

---

## 7. Diffie-Hellman Key Exchange

### Classic DH (Finite Field)

Over a prime $p$ with generator $g$:

1. Alice picks $a$, sends $A = g^a \pmod{p}$
2. Bob picks $b$, sends $B = g^b \pmod{p}$
3. Shared secret: $S = B^a = A^b = g^{ab} \pmod{p}$

Security relies on the **Discrete Logarithm Problem (DLP)**:

Given $g, p, A = g^a \pmod{p}$, finding $a$ requires:

$$O\left(\exp\left(1.923 \cdot (\ln p)^{1/3} \cdot (\ln \ln p)^{2/3}\right)\right)$$

### Minimum Key Sizes (NIST SP 800-57)

| Security Level | DH/DSA | RSA | ECC |
|:---:|:---:|:---:|:---:|
| 80-bit (legacy) | 1024 | 1024 | 160 |
| 112-bit | 2048 | 2048 | 224 |
| 128-bit | 3072 | 3072 | 256 |
| 192-bit | 7680 | 7680 | 384 |
| 256-bit | 15360 | 15360 | 521 |

---

## 8. Summary of Functions by Type

| Concept | Math Type | Core Operation |
|:---|:---|:---|
| AES S-Box | Galois field inverse | $b^{-1}$ in $GF(2^8)$ |
| RSA | Modular exponentiation | $M^e \pmod{n}$ |
| ECC | Elliptic curve algebra | Point addition/doubling |
| Hash (birthday) | Combinatorial probability | $O(2^{n/2})$ collisions |
| DH | Discrete logarithm | $g^a \pmod{p}$ |
| Key entropy | Information theory | $H = \log_2(|\mathcal{K}|)$ |
| GCM tag | Polynomial over $GF(2^{128})$ | GHASH authentication |

---

*Every encrypted message, signed certificate, and secure connection on the internet relies on these exact mathematical structures — finite fields, modular arithmetic, and elliptic curves running at billions of operations per second.*
