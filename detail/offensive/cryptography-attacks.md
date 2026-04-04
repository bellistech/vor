# The Mathematics of Cryptography Attacks — Information Theory and Algebraic Exploitation

> *Every cryptographic attack exploits a gap between a cipher's theoretical security model and its real-world instantiation. Padding oracles transform a single-bit information leak into full plaintext recovery through adaptive chosen-ciphertext queries. Compression side channels correlate entropy reduction with secret content. Timing attacks extract key bits from microarchitectural costs. The mathematics is always the same: information leakage compounds across observations.*

---

## 1. Padding Oracle — Adaptive Chosen-Ciphertext Decryption

### CBC Mode Decryption

For AES-CBC, decryption of block $C_i$:

$$P_i = D_K(C_i) \oplus C_{i-1}$$

The attacker modifies $C_{i-1}$ to control the intermediate value. For valid padding with last byte = `0x01`:

$$D_K(C_i)[B] = C_{i-1}'[B] \oplus 0x01$$

### Query Complexity

| Step | Queries per Byte | Total for Block ($B=16$) |
|:---|:---:|:---:|
| Last byte ($k=1$) | $\leq 256$ | 256 |
| Second-to-last ($k=2$) | $\leq 256$ | 512 |
| Full block | $\leq 256 \times 16$ | 4,096 |
| $N$-block message | $4{,}096 \times N$ | $4{,}096N$ |

Expected queries with uniform distribution: $128 \times B \times N$ per message.

---

## 2. Compression Side-Channel — CRIME and BREACH

### Compression Ratio as Oracle

For compression function $Z$ and message $M = \text{secret} \| \text{attacker\_input}$:

$$|Z(M)| < |Z(M')| \iff \text{attacker\_input shares substrings with secret}$$

For a 32-character CSRF token over hex alphabet:

$$\text{Queries} = 32 \times 16 = 512$$

| Attack | Layer | TLS Version | Mitigated By |
|:---|:---:|:---:|:---|
| CRIME | TLS compression | TLS 1.0-1.2 | Disable TLS compression |
| BREACH | HTTP gzip | Any | Length hiding, SameSite cookies |
| HEIST | HTTP/2 | HTTP/2 | Frame padding |

The minimum detectable compression difference: $\Delta\ell_{\min} \geq 1$ byte (deflate granularity).

---

## 3. Timing Side-Channels — Statistical Extraction

### Non-Constant-Time Comparison

A byte-by-byte comparison exits on first mismatch:

$$T(\text{compare}(a, b)) = c_0 + c_1 \times |\text{matching prefix}(a, b)|$$

Required samples for statistical significance (two-sample t-test):

$$n \geq \left(\frac{z_{\alpha/2} \cdot \sigma}{\Delta t}\right)^2$$

| $\Delta t$ | $\sigma$ | Confidence 95% | Required Samples |
|:---|:---:|:---:|:---:|
| 10 ns | 1 ms | $z = 1.96$ | 38,416,000 |
| 100 ns | 100 us | $z = 1.96$ | 3,842 |
| 1 us | 100 us | $z = 1.96$ | 39 |

### Constant-Time Implementation

$$\text{result} = \bigoplus_{i=0}^{n-1} (a_i \oplus b_i), \quad \text{return } \text{result} == 0$$

No early exit regardless of input content.

---

## 4. RSA Attacks — Number Theory

### Bleichenbacher (PKCS#1 v1.5)

PKCS#1 v1.5 padding: $\text{EB} = 0x00 \| 0x02 \| \text{PS} \| 0x00 \| M$

The oracle reveals whether $m = c^d \bmod n$ satisfies $2B \leq m < 3B$ where $B = 2^{8(k-2)}$.

After each conformant ciphertext $c' = (s^e \cdot c) \bmod n$, the interval narrows:

$$m \in \left[\frac{2B + rn}{s}, \frac{3B - 1 + rn}{s}\right]$$

Total queries for 2048-bit RSA: approximately $2^{20}$ (~1,000,000).

### Hastad's Broadcast Attack

Same message $m$ encrypted with $e$ different keys $(n_1, \ldots, n_e)$. By CRT: $c = m^e \bmod \prod n_i$. Since $m^e < \prod n_i$: $m = \sqrt[e]{c}$.

---

## 5. Block Cipher Mode Analysis

### ECB Determinism

$$P_i = P_j \iff C_i = C_j$$

Any ciphertext block collisions confirm ECB mode.

### CTR/GCM Nonce Reuse

CTR nonce reuse: $C_1 \oplus C_2 = P_1 \oplus P_2$ — recover via crib dragging.

GCM nonce reuse: authentication key $H$ is recoverable as root of polynomial over $\text{GF}(2^{128})$. Once $H$ is known, arbitrary forgeries are possible.

### Birthday Bound

After $q$ encryptions with $b$-bit block cipher:

$$P(\text{collision}) \approx \frac{q^2}{2^{b+1}}$$

| Block Size | 50% Collision | Practical Limit |
|:---|:---:|:---:|
| 64-bit (3DES) | $2^{32}$ blocks = 32 GB | ~4 GB |
| 128-bit (AES) | $2^{64}$ blocks | Effectively unlimited |

---

## 6. Hash Function Cryptanalysis

### Length Extension Attack

For Merkle-Damgard hash $H$: the digest $H(\text{secret} \| m)$ reveals internal state $s_{\text{final}}$. The attacker computes:

$$H(\text{secret} \| m \| \text{pad} \| m_{\text{ext}}) = H_{\text{continue}}(s_{\text{final}}, m_{\text{ext}})$$

| Hash | Merkle-Damgard | Length Extension |
|:---|:---:|:---:|
| MD5, SHA-1, SHA-256 | Yes | Vulnerable |
| SHA-3 (Keccak) | No (sponge) | Immune |
| BLAKE2 | No | Immune |
| HMAC (any hash) | Nested | Immune |

---

## 7. Meet-in-the-Middle Complexity

For double encryption $C = E_{K_2}(E_{K_1}(P))$ with $n$-bit keys:

$$\text{Brute force}: T = O(2^{2n}), \quad S = O(1)$$
$$\text{MITM}: T = O(2^{n+1}), \quad S = O(2^n)$$

The MITM tradeoff trades memory for time: encrypt P with all $K_1$ into a hash table, then decrypt C with all $K_2$ and look up matches.

| Scheme | Brute Force | MITM | Effective Security |
|:---|:---:|:---:|:---:|
| Double DES | $2^{112}$ | $2^{57}$ | ~57 bits |
| Triple DES (3-key) | $2^{168}$ | $2^{112}$ | 112 bits |
| Triple DES (2-key) | $2^{112}$ | $2^{56}$ | ~80 bits |

For $r$-fold encryption, optimal MITM split:

$$T = O(2^{n \cdot \lceil r/2 \rceil}), \quad S = O(2^{n \cdot \lfloor r/2 \rfloor})$$

---

## 8. Protocol Downgrade — POODLE Analysis

Per-byte CBC padding recovery:

$$P(\text{correct pad byte}) = \frac{1}{256}$$

Expected requests per byte: 256. For $L$-byte secret: $256L$ total requests.

$$P(\text{POODLE}) = P(\text{SSLv3 supported}) \times P(\text{fallback triggered}) \times P(\text{pad correct})$$

The attack requires a network position (MITM) and the ability to trigger repeated connections with attacker-chosen plaintext.

### Downgrade Attack Success Factors

| Factor | POODLE | FREAK | Logjam |
|:---|:---:|:---:|:---:|
| Server vulnerability | SSLv3 enabled | Export RSA | Export DHE |
| Queries per byte | 256 | N/A (key recovery) | N/A (key recovery) |
| Key strength attacked | CBC padding | 512-bit RSA | 512-bit DH |
| Factoring time | N/A | Hours (2015) | Minutes (precomp) |

---

*The recurring theme across all cryptographic attacks is the amplification of small information leaks: a single bit distinguishing valid from invalid padding becomes full plaintext recovery in $O(B \times N \times 256)$ queries, a one-byte compression difference recovers secrets in $O(L \times |\Sigma|)$ queries, and a nanosecond timing difference yields key material given sufficient statistical samples. Secure cryptographic engineering is fundamentally about eliminating every channel through which information about secrets can escape.*

## Prerequisites

- Modular arithmetic, group theory, and finite fields (GF($2^n$))
- Probability theory and statistical hypothesis testing
- Understanding of block cipher modes (ECB, CBC, CTR, GCM) and hash constructions

## Complexity

- **Beginner:** Understanding ECB vs CBC, detecting padding oracles, using hashcat
- **Intermediate:** Implementing padding oracle attacks, analyzing TLS configurations, birthday bounds
- **Advanced:** Bleichenbacher interval narrowing, GCM nonce-reuse key recovery, novel timing channels
