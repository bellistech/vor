# The Mathematics of GPG — Web of Trust and Hybrid Encryption

> *GnuPG implements the OpenPGP standard (RFC 4880) using hybrid encryption: asymmetric cryptography for key exchange and symmetric cryptography for bulk data. Its trust model replaces centralized PKI with a decentralized web of trust based on transitive signature chains.*

---

## 1. Hybrid Encryption — The OpenPGP Message Format

### Encryption Process

To encrypt message $M$ for recipient with public key $K_{pub}$:

1. Generate random session key $s$ (256 bits for AES-256)
2. Encrypt: $C_{sym} = \text{AES-256-CFB}(s, M)$
3. Encrypt session key: $C_{key} = \text{RSA}(K_{pub}, s)$ or $C_{key} = \text{ECDH}(K_{pub}, s)$
4. Output: $C_{key} \| C_{sym}$

### Why Hybrid?

| Property | RSA Only | AES Only | Hybrid |
|:---|:---:|:---:|:---:|
| Key distribution | Solved | Unsolved | Solved |
| Speed (1 MB) | ~10 seconds | ~0.3 ms | ~0.5 ms |
| Message size overhead | 16x expansion | ~0% | ~0.5% |

RSA can only encrypt data up to key size minus padding:

$$|M_{RSA}| \leq |n| - 42 \text{ bytes (PKCS\#1 v1.5)} = 256 - 42 = 214 \text{ bytes for RSA-2048}$$

The session key (~32 bytes) fits easily; the message does not.

---

## 2. Key Types and Security Levels

### GPG Key Algorithms

| Algorithm | Type | Key Size | Security Level | Speed |
|:---|:---|:---:|:---:|:---|
| RSA | Sign + Encrypt | 2048-4096 | 112-128 bit | Moderate |
| DSA | Sign only | 2048-3072 | 112-128 bit | Fast sign |
| ElGamal | Encrypt only | 2048-4096 | 112-128 bit | Moderate |
| ECDSA (P-256) | Sign only | 256 | 128 bit | Fast |
| ECDH (Cv25519) | Encrypt only | 256 | 128 bit | Fast |
| EdDSA (Ed25519) | Sign only | 256 | 128 bit | Fastest |

### Key Size vs Performance

$$T_{sign} \propto |k|^2, \quad T_{verify} \propto |k|^2 \text{ (RSA)}$$

| Operation | RSA-2048 | RSA-4096 | Ed25519 |
|:---|:---:|:---:|:---:|
| Sign | 1.5 ms | 8 ms | 0.05 ms |
| Verify | 0.1 ms | 0.3 ms | 0.15 ms |
| Encrypt (session key) | 0.1 ms | 0.3 ms | 0.04 ms |
| Decrypt (session key) | 1.5 ms | 8 ms | 0.04 ms |

Ed25519 is 30-160x faster than RSA-4096 for signing.

---

## 3. Web of Trust — Transitive Trust Graph

### Trust Model

The web of trust is a directed graph:

$$G_{trust} = (K, S) \quad \text{where } (k_i, k_j) \in S \iff k_i \text{ has signed } k_j$$

### Trust Levels

| Level | Meaning | Weight |
|:---:|:---|:---:|
| Unknown | No information | 0 |
| None | Explicitly untrusted | 0 |
| Marginal | Somewhat trusted introducer | 1 |
| Full | Fully trusted introducer | 3 |
| Ultimate | Your own key | $\infty$ |

### Validity Calculation

A key is **valid** if:

$$\text{Valid}(k) \iff \exists \text{ path from ultimate key to } k \text{ with sufficient trust}$$

Specifically:
- 1 fully trusted signature, OR
- 3 marginally trusted signatures (default `completes-needed = 3`)

### Trust Path Length

Default maximum trust path: `max-cert-depth = 5`

$$\text{Valid}(k) \iff \exists \text{ path of length} \leq 5 \text{ from your key to } k$$

### Small World Effect

In a web of trust with $n$ keys and average signatures per key $d$:

$$\text{Average path length} \approx \frac{\ln n}{\ln d}$$

| Keys | Avg Signatures | Expected Path Length |
|:---:|:---:|:---:|
| 1,000 | 5 | 4.3 |
| 10,000 | 5 | 5.7 |
| 100,000 | 10 | 5.0 |
| 1,000,000 | 10 | 6.0 |

The "strong set" of the PGP web of trust (mutually connected keys) historically contained ~50,000 keys with average path length ~6.

---

## 4. Digital Signatures

### RSA Signature (PKCS#1 v1.5)

$$\text{sig} = (\text{DigestInfo}(H(M)))^d \pmod{n}$$

Where DigestInfo includes the hash algorithm OID (ASN.1 encoded).

### Signature Verification

$$\text{DigestInfo}(H(M)) \stackrel{?}{=} \text{sig}^e \pmod{n}$$

### Signature Packet Format

| Field | Size | Content |
|:---|:---:|:---|
| Version | 1 byte | 4 (current) |
| Signature type | 1 byte | 0x00 (binary), 0x01 (text), 0x10-0x13 (key sigs) |
| Hash algorithm | 1 byte | SHA-256 (8), SHA-512 (10) |
| Key algorithm | 1 byte | RSA (1), DSA (17), EdDSA (22) |
| Key ID | 8 bytes | Signing key fingerprint (last 8 bytes) |
| Hashed subpackets | variable | Timestamp, policy URI, notation |
| Signature value | key-dependent | RSA: 256-512 bytes, EdDSA: 64 bytes |

---

## 5. Key Fingerprints

### Fingerprint Computation

$$\text{Fingerprint} = \text{SHA-1}(0x99 \| \text{len} \| \text{pubkey packet})$$

20 bytes (160 bits) displayed as 10 groups of 4 hex characters:

```
D8FC 66D2 9C85 4A43 92F0  8E19 5B40 4ED7 A1E1 3498
```

### Fingerprint Collision Probability

Birthday bound for SHA-1 (160 bits):

$$P(\text{collision among } k \text{ keys}) \approx \frac{k^2}{2^{161}}$$

For 100 million keys: $P \approx \frac{10^{16}}{2^{161}} = 3.4 \times 10^{-33}$ — negligible.

However, SHA-1 is broken for intentional collisions ($2^{63}$ complexity). OpenPGP is migrating to SHA-256 fingerprints (v5 keys, 32 bytes).

### Key ID Collision

Short key IDs (8 hex chars = 32 bits) are trivially collided:

$$P(\text{collision at 32 bits}) \approx \frac{k^2}{2^{33}} \rightarrow 50\% \text{ at } k \approx 77{,}000 \text{ keys}$$

**Always use full fingerprints for key verification.** Short key IDs were exploited in the "Evil32" attack.

---

## 6. Symmetric Cipher Selection

### Cipher Preference

GPG selects the strongest cipher supported by ALL recipients:

$$\text{Selected cipher} = \max_{c \in \bigcap_{r} \text{prefs}(r)} \text{strength}(c)$$

### Available Ciphers

| Cipher | Key Size | Block Size | Security | Status |
|:---|:---:|:---:|:---:|:---|
| IDEA | 128 | 64 | 128-bit | Legacy |
| 3DES | 168 | 64 | 112-bit | Mandatory (fallback) |
| CAST5 | 128 | 64 | 128-bit | Default (legacy GPG) |
| AES-128 | 128 | 128 | 128-bit | Recommended |
| AES-256 | 256 | 128 | 256-bit | Recommended |
| Twofish | 256 | 128 | 256-bit | Available |
| Camellia-256 | 256 | 128 | 256-bit | Available |

### Mode of Operation

OpenPGP uses CFB (Cipher Feedback) mode with a modification:

$$C_i = P_i \oplus E_K(\text{shift register}_i)$$

The shift register provides a form of IV. Unlike GCM, CFB does not provide authentication — OpenPGP adds a Modification Detection Code (MDC):

$$\text{MDC} = \text{SHA-1}(\text{plaintext} \| 0xD3 \| 0x14)$$

---

## 7. Keyring Storage and Subkeys

### Key Structure

A GPG key is actually a set of subkeys:

$$\text{Key} = \{k_{master}, k_{sign}, k_{encrypt}, k_{auth}\}$$

| Subkey | Usage | Stored On | Rotation |
|:---|:---|:---|:---|
| Master (certify) | Sign other keys, create subkeys | Offline backup | Never (or rarely) |
| Signing | Sign messages/code | Daily machine | Yearly |
| Encryption | Decrypt messages | Daily machine | Yearly |
| Authentication | SSH auth | Daily machine | Yearly |

### Subkey Revocation

If a subkey is compromised:

$$\text{Impact} = \begin{cases} \text{All trust relationships lost} & \text{if master key compromised} \\ \text{Only that subkey's operations affected} & \text{if subkey compromised} \end{cases}$$

The master key should be stored offline (air-gapped, hardware token) — compromise of the master key is catastrophic.

### Keyring Scalability

Keyring size:

$$S_{keyring} = \sum_{k \in \text{keys}} (|k_{pub}| + |\text{signatures on } k|)$$

| Keys | Avg Signatures | Keyring Size |
|:---:|:---:|:---:|
| 100 | 3 | ~500 KB |
| 1,000 | 5 | ~8 MB |
| 10,000 | 10 | ~150 MB |

---

## 8. Summary of Functions by Type

| Concept | Math Type | Application |
|:---|:---|:---|
| Hybrid encryption | Composition of ciphers | Message confidentiality |
| $M^d \bmod n$ | Modular exponentiation | RSA signing |
| Trust graph $G = (K, S)$ | Directed graph traversal | Web of trust |
| $\ln n / \ln d$ | Small-world property | Trust path length |
| $k^2 / 2^{161}$ | Birthday probability | Fingerprint collisions |
| $\bigcap \text{prefs}$ | Set intersection | Cipher negotiation |
| CFB + MDC | Stream mode + integrity | Symmetric encryption |

## Prerequisites

- modular arithmetic, prime factorization, hash functions, web of trust (graph theory)

---

*GPG replaces centralized certificate authorities with a mathematical web of trust — transitive signature chains where each user makes independent trust decisions, creating a resilient, decentralized identity system backed by the same cryptographic primitives as TLS.*
