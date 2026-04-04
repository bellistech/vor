# The Mathematics of TLS — Cryptographic Protocol Engineering

> *TLS 1.3 (RFC 8446) is a state machine driven by finite field arithmetic, elliptic curve cryptography, and authenticated encryption. Every handshake is a precisely choreographed exchange of mathematical proofs.*

---

## 1. TLS 1.3 Handshake State Machine

### State Transitions (RFC 8446 Section 4)

The TLS 1.3 handshake is a deterministic finite automaton with exactly **1-RTT** for full handshake and **0-RTT** for resumption.

```
Client                                Server

ClientHello          -------->
  + key_share                         ServerHello
  + supported_versions                + key_share
                                      {EncryptedExtensions}
                                      {CertificateRequest*}
                                      {Certificate*}
                                      {CertificateVerify*}
                     <--------        {Finished}
{Certificate*}
{CertificateVerify*}
{Finished}           -------->
[Application Data]   <------->        [Application Data]
```

### Message Count Comparison

| Version | Full Handshake RTTs | Messages | Cipher Negotiation |
|:---:|:---:|:---:|:---|
| TLS 1.0 | 2 | 9 | Client proposes list |
| TLS 1.2 | 2 | 7-9 | Client proposes list |
| TLS 1.3 | 1 | 5-7 | Client sends key_share |
| TLS 1.3 0-RTT | 0 | 3-5 | Pre-shared key |

---

## 2. Elliptic Curve Diffie-Hellman (ECDHE) — Forward Secrecy

### The Mathematical Foundation

TLS 1.3 mandates ephemeral key exchange. The default curve is **x25519** (Curve25519).

Curve25519 is defined over $\mathbb{F}_p$ where $p = 2^{255} - 19$:

$$y^2 = x^3 + 486662x^2 + x \pmod{p}$$

### Key Exchange Protocol

1. Client generates random scalar $a \in [1, n-1]$, computes $A = a \cdot G$
2. Server generates random scalar $b \in [1, n-1]$, computes $B = b \cdot G$
3. Both compute shared secret: $S = a \cdot B = b \cdot A = ab \cdot G$

Where $G$ is the generator point and $n$ is the group order:

$$n = 2^{252} + 27742317777372353535851937790883648493$$

### Security: Elliptic Curve Discrete Logarithm Problem (ECDLP)

Given points $P$ and $Q = k \cdot P$, finding $k$ requires:

$$O(\sqrt{n}) \text{ operations (Pollard's rho)}$$

For Curve25519: $\sqrt{2^{252}} = 2^{126}$ operations — equivalent to **126-bit security**.

### Forward Secrecy Guarantee

Because $a$ and $b$ are ephemeral (generated per session and discarded):

- Compromising the server's long-term private key does **not** reveal past session keys
- Each session has a unique shared secret $S = ab \cdot G$
- Breaking one session requires solving ECDLP for that specific $(a, b)$ pair

---

## 3. Cipher Suite Security Levels

### TLS 1.3 Mandatory Cipher Suites

TLS 1.3 reduced cipher suites from ~300 to exactly **5**:

| Cipher Suite | AEAD | Hash | Security Level |
|:---|:---|:---|:---:|
| TLS_AES_128_GCM_SHA256 | AES-128-GCM | SHA-256 | 128-bit |
| TLS_AES_256_GCM_SHA384 | AES-256-GCM | SHA-384 | 256-bit |
| TLS_CHACHA20_POLY1305_SHA256 | ChaCha20-Poly1305 | SHA-256 | 256-bit |
| TLS_AES_128_CCM_SHA256 | AES-128-CCM | SHA-256 | 128-bit |
| TLS_AES_128_CCM_8_SHA256 | AES-128-CCM-8 | SHA-256 | 128-bit |

### What "128-bit Security" Means

An attacker must perform $2^{128}$ operations to break the cipher:

$$\text{Time} = \frac{2^{128}}{\text{operations/second}}$$

| Attacker Capability | Time to Break AES-128 | Time to Break AES-256 |
|:---|:---|:---|
| $10^9$ ops/sec (PC) | $1.08 \times 10^{22}$ years | $3.67 \times 10^{60}$ years |
| $10^{12}$ ops/sec (cluster) | $1.08 \times 10^{19}$ years | $3.67 \times 10^{57}$ years |
| $10^{18}$ ops/sec (nation-state) | $1.08 \times 10^{13}$ years | $3.67 \times 10^{51}$ years |

Universe age: $\approx 1.38 \times 10^{10}$ years. AES-128 requires ~1000x the age of the universe even for a nation-state.

---

## 4. HKDF — Key Derivation (RFC 5869)

### The Key Schedule

TLS 1.3 derives all keys from the shared secret using HMAC-based Key Derivation Function:

**Extract phase:**

$$\text{PRK} = \text{HMAC-Hash}(\text{salt}, \text{IKM})$$

**Expand phase:**

$$T(1) = \text{HMAC-Hash}(\text{PRK}, \text{info} \| 0x01)$$
$$T(i) = \text{HMAC-Hash}(\text{PRK}, T(i-1) \| \text{info} \| i)$$

### Key Schedule Tree

```
PSK (or 0) -----> HKDF-Extract = Early Secret
                        |
                  Derive-Secret(., "derived", "")
                        |
                        v
(EC)DHE ---------> HKDF-Extract = Handshake Secret
                        |
                  Derive-Secret(., "derived", "")
                        |
                        v
0 ---------------> HKDF-Extract = Master Secret
```

### Derived Key Sizes

| Key | Length (AES-128) | Length (AES-256) |
|:---|:---:|:---:|
| Client handshake traffic key | 16 bytes | 32 bytes |
| Server handshake traffic key | 16 bytes | 32 bytes |
| Client application traffic key | 16 bytes | 32 bytes |
| Server application traffic key | 16 bytes | 32 bytes |
| IV (nonce) | 12 bytes | 12 bytes |

---

## 5. Certificate Chain Validation

### Chain as Directed Graph

A certificate chain forms a **directed acyclic graph** (DAG):

$$\text{Leaf} \xrightarrow{\text{signed by}} \text{Intermediate}_1 \xrightarrow{\text{signed by}} \cdots \xrightarrow{\text{signed by}} \text{Root}$$

### Validation Algorithm (RFC 5280 Section 6)

For each certificate $C_i$ in the chain $[C_0, C_1, \ldots, C_n]$:

1. **Signature verification:** $\text{Verify}(C_i.\text{signature}, C_{i+1}.\text{publicKey}) = \text{true}$
2. **Validity period:** $\text{notBefore}_i \leq \text{now} \leq \text{notAfter}_i$
3. **Name chaining:** $C_i.\text{issuer} = C_{i+1}.\text{subject}$
4. **Basic constraints:** If $i < n$, then $C_i.\text{isCA} = \text{true}$ and $C_i.\text{pathLen} \geq n - i - 1$
5. **Key usage:** $C_i$ must have keyCertSign for intermediate certs
6. **Trust anchor:** $C_n$ must be in the local trust store

### Chain Length vs. Verification Cost

$$T_{verify} = \sum_{i=0}^{n} T_{sig}(C_i) + T_{revocation}(C_i)$$

| Chain Length | RSA-2048 Verify | ECDSA-P256 Verify |
|:---:|:---:|:---:|
| 2 (leaf + root) | ~0.4 ms | ~0.6 ms |
| 3 (+ intermediate) | ~0.6 ms | ~0.9 ms |
| 4 (+ cross-signed) | ~0.8 ms | ~1.2 ms |

ECDSA verification is slower per-operation but uses smaller certificates (savings in bandwidth).

---

## 6. AEAD — Authenticated Encryption with Associated Data

### AES-GCM Construction

For each TLS record:

$$(\text{ciphertext}, \text{tag}) = \text{AES-GCM}(K, \text{nonce}, \text{plaintext}, \text{AAD})$$

**Nonce construction** (critical for security):

$$\text{nonce} = \text{IV} \oplus \text{sequence\_number}$$

The nonce is 96 bits. The sequence number increments per record. The XOR ensures unique nonces without explicit transmission.

### Nonce Reuse Catastrophe

If two records use the same nonce with the same key:

$$C_1 \oplus C_2 = P_1 \oplus P_2$$

This leaks the XOR of plaintexts — a complete break of confidentiality. TLS 1.3's sequential nonce construction prevents this by design.

### GCM Authentication Tag

The 128-bit authentication tag provides:

$$P(\text{forgery}) = \frac{1}{2^{128}} \approx 2.94 \times 10^{-39}$$

An attacker has a 1-in-$2^{128}$ chance of forging a valid ciphertext per attempt.

---

## 7. 0-RTT Resumption — Speed vs. Security Tradeoff

### Pre-Shared Key (PSK) Mode

0-RTT allows sending application data in the first flight:

$$\text{Latency}_{0\text{-RTT}} = 0 \text{ round trips (data sent with ClientHello)}$$

### The Replay Attack Problem

0-RTT data has **no forward secrecy** and is **replayable**:

- An attacker who captures the ClientHello + 0-RTT data can replay it
- The server sees a valid PSK and processes the data again
- Mitigation: servers must implement anti-replay (strike registers or time windows)

### Anti-Replay Window

$$P(\text{replay accepted}) = \begin{cases} 0 & \text{if } t_{replay} > t_{window} \\ \text{possible} & \text{if } t_{replay} \leq t_{window} \end{cases}$$

Typical window: 10 seconds. This is why 0-RTT should only carry idempotent requests.

---

## 8. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| $y^2 = x^3 + ax^2 + x$ | Elliptic curve algebra | Key exchange (ECDHE) |
| $O(\sqrt{n})$ | Complexity theory | ECDLP security bound |
| $2^{128}$ brute force | Exponential | Cipher strength |
| HMAC chain | Hash composition | Key derivation (HKDF) |
| $\text{IV} \oplus \text{seq}$ | Bitwise XOR | Nonce construction |
| $1/2^{128}$ | Probability | Forgery resistance |
| DAG validation | Graph theory | Certificate chain |

## Prerequisites

- elliptic curve algebra, finite fields, modular arithmetic, hash functions, probability

---

*Every HTTPS connection on the internet executes this exact mathematics — the 1-RTT handshake completes in under 100ms while establishing 128+ bits of security.*
