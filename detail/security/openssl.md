# The Mathematics of OpenSSL — Applied Cryptographic Operations

> *OpenSSL is the reference implementation of TLS and a general-purpose cryptographic library. Understanding it means understanding the mathematics of key generation, certificate signing, symmetric encryption, and random number generation as concrete operations.*

---

## 1. RSA Key Generation Internals

### Prime Generation

OpenSSL generates RSA keys by finding large primes using **Miller-Rabin primality testing**.

For an $n$-bit RSA key, each prime $p, q$ is $n/2$ bits. The density of primes near $N$:

$$\pi(N) \approx \frac{N}{\ln N}$$

Expected trials to find an $n/2$-bit prime:

$$E[\text{trials}] \approx \frac{n}{2} \times \ln 2 \approx 0.347n$$

| Key Size | Prime Size | Expected Trials | Generation Time |
|:---:|:---:|:---:|:---:|
| RSA-2048 | 1024 bits | ~355 | 0.1-0.5 s |
| RSA-3072 | 1536 bits | ~532 | 0.5-2 s |
| RSA-4096 | 2048 bits | ~710 | 1-5 s |
| RSA-8192 | 4096 bits | ~1420 | 10-60 s |

### Miller-Rabin Primality Test

Probability of a composite passing $k$ rounds of Miller-Rabin:

$$P(\text{false prime}) \leq 4^{-k}$$

| Rounds ($k$) | Error Probability | OpenSSL Default |
|:---:|:---:|:---:|
| 10 | $9.5 \times 10^{-7}$ | Minimum |
| 20 | $9.1 \times 10^{-13}$ | Standard |
| 40 | $8.3 \times 10^{-25}$ | High security |
| 64 | $2.9 \times 10^{-39}$ | FIPS mode |

---

## 2. Random Number Generation (CSPRNG)

### Entropy Sources

OpenSSL's DRBG (Deterministic Random Bit Generator) requires seed entropy:

$$H_{seed} \geq 256 \text{ bits (for 128-bit security)}$$

| Platform | Entropy Source | Entropy Rate |
|:---|:---|:---:|
| Linux | `/dev/urandom` (ChaCha20 DRNG) | Unlimited (after seeding) |
| Linux | `getrandom()` syscall | Blocks until 256-bit seed |
| macOS | `getentropy()` | 256 bits per call |
| Hardware | RDRAND/RDSEED (Intel) | ~800 MB/s |

### DRBG Reseeding

OpenSSL reseeds the DRBG after:

$$\text{Reseed interval} = 2^{48} \text{ generate requests OR } 2^{48} \text{ bytes generated}$$

This prevents state compromise from revealing future outputs.

---

## 3. Certificate Operations

### CSR Signature Process

When generating a Certificate Signing Request:

$$\text{CSR} = \text{Sign}_{d}(\text{Subject} \| \text{PublicKey} \| \text{Extensions})$$

The signature proves possession of the private key corresponding to the public key in the CSR.

### X.509 Certificate Fields

| Field | ASN.1 Type | Size (RSA-2048) | Size (ECDSA-P256) |
|:---|:---|:---:|:---:|
| Serial number | INTEGER | 20 bytes | 20 bytes |
| Signature algorithm | OID | 13 bytes | 10 bytes |
| Issuer DN | SEQUENCE | ~100 bytes | ~100 bytes |
| Validity | SEQUENCE | 32 bytes | 32 bytes |
| Subject DN | SEQUENCE | ~100 bytes | ~100 bytes |
| Public key | BIT STRING | 294 bytes | 91 bytes |
| Signature | BIT STRING | 256 bytes | 72 bytes |
| **Total (typical)** | | **~1200 bytes** | **~600 bytes** |

### DER vs PEM Encoding

PEM adds Base64 encoding overhead:

$$\text{PEM size} = \frac{4}{3} \times \text{DER size} + \text{headers} + \text{newlines}$$

$$\text{Overhead} \approx 33\% + 50 \text{ bytes (headers)}$$

---

## 4. Symmetric Encryption Performance

### Algorithm Benchmarks

$$\text{Throughput} = \frac{\text{data size}}{\text{encryption time}}$$

Typical OpenSSL benchmarks (`openssl speed`):

| Algorithm | Key | Block | Throughput (1 core) |
|:---|:---:|:---:|:---:|
| AES-128-GCM | 128 | 128 | 4-6 GB/s (AES-NI) |
| AES-256-GCM | 256 | 128 | 3-5 GB/s (AES-NI) |
| AES-128-CBC | 128 | 128 | 1-2 GB/s (AES-NI) |
| ChaCha20-Poly1305 | 256 | stream | 2-4 GB/s (AVX2) |
| AES-128-GCM (no AES-NI) | 128 | 128 | 200-400 MB/s |
| 3DES-CBC | 168 | 64 | 50-80 MB/s |

### AES-NI Speedup

Hardware AES instructions provide:

$$\text{Speedup}_{AES\text{-}NI} = \frac{T_{software}}{T_{hardware}} \approx 10\text{-}20\times$$

### Encryption Overhead Per TLS Record

$$\text{Overhead per record} = \text{IV/nonce} + \text{auth tag} + \text{padding} + \text{record header}$$

| Mode | IV | Tag | Header | Total Overhead |
|:---|:---:|:---:|:---:|:---:|
| AES-GCM | 8 bytes (explicit) | 16 bytes | 5 bytes | 29 bytes |
| ChaCha20-Poly1305 | 0 (implicit) | 16 bytes | 5 bytes | 21 bytes |
| AES-CBC + HMAC-SHA256 | 16 bytes | 32 bytes | 5 bytes | 53+ bytes |

---

## 5. Elliptic Curve Operations

### Supported Curves

| Curve | Field Size | Security | Key Gen Time | Sign Time |
|:---|:---:|:---:|:---:|:---:|
| P-256 (prime256v1) | 256 bits | 128-bit | 0.05 ms | 0.08 ms |
| P-384 (secp384r1) | 384 bits | 192-bit | 0.15 ms | 0.20 ms |
| P-521 (secp521r1) | 521 bits | 256-bit | 0.40 ms | 0.50 ms |
| X25519 (DH only) | 255 bits | 128-bit | 0.04 ms | N/A |
| Ed25519 (sign only) | 255 bits | 128-bit | N/A | 0.04 ms |

### ECDH Shared Secret Computation

$$S = x\text{-coordinate of } (d_A \cdot Q_B) = x\text{-coordinate of } (d_B \cdot Q_A)$$

Where $d$ is the private scalar and $Q$ is the public point.

Scalar multiplication cost: $O(\log n)$ point additions via double-and-add.

For P-256: ~256 point doublings + ~128 point additions (on average).

---

## 6. Password-Based Key Derivation

### PBKDF2 (RFC 8018)

$$DK = T_1 \| T_2 \| \ldots \| T_{\lceil dkLen/hLen \rceil}$$
$$T_i = U_1 \oplus U_2 \oplus \cdots \oplus U_c$$
$$U_1 = \text{PRF}(P, S \| \text{INT}(i))$$
$$U_j = \text{PRF}(P, U_{j-1})$$

Where $P$ = password, $S$ = salt, $c$ = iteration count.

### Iteration Count Recommendations

$$T_{derive} = c \times T_{PRF}$$

| Year | OWASP Recommended $c$ | Derive Time (SHA-256) |
|:---:|:---:|:---:|
| 2023 | 600,000 | ~300 ms |
| 2025 | 1,000,000 | ~500 ms |

### Comparison: PBKDF2 vs scrypt vs Argon2

| KDF | CPU Hard | Memory Hard | GPU Resistant |
|:---|:---:|:---:|:---:|
| PBKDF2 | Yes | No | No |
| scrypt | Yes | Yes | Moderate |
| Argon2id | Yes | Yes | Yes |

OpenSSL 3.x supports all three via the EVP_KDF API.

---

## 7. ASN.1/DER Encoding Mathematics

### Tag-Length-Value (TLV) Structure

Every ASN.1 element:

$$\text{Encoded} = \text{Tag}(1 \text{ byte}) \| \text{Length}(1\text{-}5 \text{ bytes}) \| \text{Value}$$

Length encoding:

$$\text{Length} = \begin{cases} L & \text{if } L \leq 127 \text{ (short form, 1 byte)} \\ 0x80 | n \| L_{\text{bytes}} & \text{if } L > 127 \text{ (long form, } 1+n \text{ bytes)} \end{cases}$$

### Certificate Size Breakdown

| Component | ASN.1 Overhead | Data | Total |
|:---|:---:|:---:|:---:|
| RSA-2048 public key | 24 bytes | 256 bytes | 280 bytes |
| RSA-2048 signature | 10 bytes | 256 bytes | 266 bytes |
| ECDSA-P256 public key | 26 bytes | 65 bytes | 91 bytes |
| ECDSA-P256 signature | 8 bytes | ~70 bytes | ~78 bytes |
| Subject DN (typical) | 20 bytes | 80 bytes | 100 bytes |

---

## 8. Summary of Functions by Type

| Concept | Math Type | Application |
|:---|:---|:---|
| Prime density $N/\ln N$ | Number theory | RSA key generation |
| Miller-Rabin $4^{-k}$ | Probabilistic testing | Primality verification |
| Base64 $4/3$ expansion | Encoding ratio | PEM format overhead |
| AES-NI $10\text{-}20\times$ | Hardware acceleration | Encryption throughput |
| PBKDF2 iterations | Linear cost function | Password-based keys |
| ASN.1 TLV | Structural encoding | Certificate format |
| ECDH scalar multiply | Elliptic curve algebra | Key agreement |

---

*OpenSSL translates abstract cryptographic mathematics into concrete byte operations — every `openssl genrsa`, `openssl req`, and `openssl s_client` invocation executes the exact algorithms described here.*
