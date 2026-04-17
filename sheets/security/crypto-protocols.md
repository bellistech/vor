# Cryptographic Protocols and Algorithms

Symmetric and asymmetric algorithms, hash functions, MACs, KDFs, digital signatures, key exchange, cipher suite notation, post-quantum cryptography, and algorithm selection guidelines.

## Symmetric Encryption

### AES (Advanced Encryption Standard)

```
# AES — NIST FIPS 197, Rijndael block cipher
# Block size: 128 bits (always)
# Key sizes: 128, 192, or 256 bits
# Rounds: 10 (AES-128), 12 (AES-192), 14 (AES-256)

# AES-128: adequate for most applications
# AES-256: required for TOP SECRET (NSA Suite B / CNSA)
# AES-192: rarely used in practice (no significant advantage over 128)

# AES modes of operation:

# CBC (Cipher Block Chaining) — legacy, avoid for new designs
# - Requires padding (PKCS#7)
# - IV must be random and unpredictable
# - Susceptible to padding oracle attacks if error messages leak
# - Not parallelizable for encryption (parallelizable for decryption)

# CTR (Counter Mode) — stream cipher from block cipher
# - No padding needed
# - Parallelizable encryption and decryption
# - Nonce must never repeat with same key (catastrophic if violated)
# - No integrity protection (use with HMAC or use GCM instead)

# GCM (Galois/Counter Mode) — AEAD (recommended)
# - Provides confidentiality + integrity + authenticity
# - 128-bit authentication tag
# - Nonce: 96 bits (12 bytes) recommended
# - MUST NOT reuse nonce with same key (breaks authentication)
# - Hardware-accelerated on modern CPUs (AES-NI + CLMUL)
# - Maximum plaintext per key+nonce: 2^39 - 256 bits (~64 GB)
# - Maximum invocations per key: 2^32 (with random nonces)

# OpenSSL examples:
openssl enc -aes-256-cbc -salt -pbkdf2 -in plain.txt -out cipher.bin
openssl enc -aes-256-cbc -d -pbkdf2 -in cipher.bin -out plain.txt

# AES-GCM with OpenSSL (programmatic, not command-line enc):
# Use EVP_EncryptInit_ex with EVP_aes_256_gcm()
# Or in Go: crypto/cipher NewGCM()
# Or in Python: from cryptography.hazmat.primitives.ciphers.aead import AESGCM
```

### ChaCha20-Poly1305

```
# ChaCha20-Poly1305 — AEAD cipher (RFC 8439)
# Designed by Daniel J. Bernstein
# Used in TLS 1.3, WireGuard, SSH

# ChaCha20 stream cipher:
# - 256-bit key, 96-bit nonce, 32-bit counter
# - 20 rounds of quarter-round operations
# - No lookup tables (constant-time, immune to cache-timing attacks)
# - Fast in software on platforms without AES-NI (mobile, IoT)

# Poly1305 MAC:
# - One-time authenticator (key derived from ChaCha20)
# - 128-bit tag
# - Provably secure if key is never reused

# Performance comparison vs AES-GCM:
# Platform                 AES-256-GCM     ChaCha20-Poly1305
# ──────────────────────────────────────────────────────────────
# x86-64 with AES-NI       ~4 GB/s         ~2 GB/s
# x86-64 without AES-NI    ~200 MB/s       ~1.5 GB/s
# ARM without ARMv8 CE     ~100 MB/s       ~500 MB/s
# ARM with ARMv8 CE        ~3 GB/s         ~1 GB/s

# Choose ChaCha20-Poly1305 when:
# - Target platform lacks AES hardware acceleration
# - Constant-time operation is critical (side-channel resistance)
# - Software-only implementation required
# Choose AES-GCM when:
# - AES-NI available (most modern x86 and ARMv8)
# - Compliance requires NIST-approved algorithms (FIPS 140)
# - Hardware acceleration available
```

### 3DES (Triple DES)

```
# 3DES — three passes of DES with 2 or 3 different keys
# Block size: 64 bits
# Effective key length: 112 bits (3-key) or 80 bits (2-key)
# DEPRECATED — do not use for new designs

# Why 3DES is deprecated:
# - 64-bit block size → vulnerable to Sweet32 birthday attack
#   After 2^32 blocks (~32 GB), collisions become likely
# - Extremely slow compared to AES (~3x DES, ~10x slower than AES)
# - NIST deprecated 3DES in 2023 (disallowed after 2023)
# - PCI DSS: 3DES removed from acceptable algorithms

# Migration: replace with AES-128-GCM or AES-256-GCM everywhere
```

## Asymmetric Encryption and Key Exchange

### RSA

```
# RSA — Rivest-Shamir-Adleman
# Security basis: integer factorization problem
# Key sizes: 2048, 3072, 4096 bits (2048 minimum, 3072+ recommended)

# Usage:
# - Digital signatures (RSA-PSS recommended over PKCS#1 v1.5)
# - Key exchange (RSA key transport — NOT forward-secret)
# - Encryption (RSA-OAEP for padding)

# Key size to security level mapping:
# RSA Key Size    Equivalent Symmetric Bits    Status
# ────────────────────────────────────────────────────
# 1024            80                           BROKEN (do not use)
# 2048            112                          Acceptable until ~2030
# 3072            128                          Recommended
# 4096            ~140                         High security

# Generate RSA key pair:
openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:3072 -out rsa.key
openssl pkey -in rsa.key -pubout -out rsa.pub

# Sign with RSA-PSS:
openssl dgst -sha256 -sigopt rsa_padding_mode:pss -sign rsa.key \
  -out sig.bin message.txt

# Verify RSA-PSS signature:
openssl dgst -sha256 -sigopt rsa_padding_mode:pss -verify rsa.pub \
  -signature sig.bin message.txt

# RSA is NOT used for key exchange in TLS 1.3 (only ECDHE/DHE)
# RSA key transport (client encrypts premaster with server's RSA pubkey)
# was removed because it does not provide forward secrecy.
```

### Elliptic Curve Algorithms

```
# ECDSA — Elliptic Curve Digital Signature Algorithm
# Key sizes: P-256 (128-bit security), P-384 (192-bit), P-521 (256-bit)

# Generate ECDSA key:
openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:P-256 -out ec.key
openssl pkey -in ec.key -pubout -out ec.pub

# Sign with ECDSA:
openssl dgst -sha256 -sign ec.key -out sig.bin message.txt
openssl dgst -sha256 -verify ec.pub -signature sig.bin message.txt

# Ed25519 — Edwards-curve Digital Signature Algorithm (RFC 8032)
# Fixed curve: Curve25519 (twisted Edwards form)
# 128-bit security, 256-bit keys
# Deterministic signatures (no random nonce — no nonce-reuse risk)
# Faster than ECDSA P-256 for both signing and verification

# Generate Ed25519 key:
openssl genpkey -algorithm Ed25519 -out ed25519.key
openssl pkey -in ed25519.key -pubout -out ed25519.pub

# SSH key generation (Ed25519 preferred):
ssh-keygen -t ed25519 -C "user@example.com"

# ECDH — Elliptic Curve Diffie-Hellman (key exchange)
# Used in TLS handshake for forward-secret key agreement
# Ephemeral variant (ECDHE) generates new keys per session
# Curves: P-256, P-384, X25519 (Curve25519 for DH)

# X25519 — Curve25519 Diffie-Hellman (RFC 7748)
# The preferred key exchange in TLS 1.3 and modern protocols
# Fixed-base scalar multiplication on Montgomery curve
# 128-bit security, constant-time implementations by design
# Used in: TLS 1.3, WireGuard, Signal Protocol, Noise Protocol

# Key size comparison (equivalent security):
# Symmetric   RSA         ECC (NIST)   EdDSA/X25519
# ──────────────────────────────────────────────────
# 128 bit     3072 bit    256 bit      256 bit
# 192 bit     7680 bit    384 bit      448 bit (Ed448)
# 256 bit     15360 bit   521 bit      N/A
```

## Hash Functions

### SHA-2 Family

```
# SHA-256 — 256-bit output, 64 rounds
# The default choice for most applications
# Used in: TLS, code signing, blockchain, HMAC, password hashing (with KDF)

# SHA-384 — 384-bit output (truncated SHA-512)
# Required for some government/military applications (NSA CNSA Suite)

# SHA-512 — 512-bit output, 80 rounds
# Faster than SHA-256 on 64-bit platforms (operates on 64-bit words)
# Used when 256-bit collision resistance is needed

# Compute hashes:
sha256sum file.txt
sha384sum file.txt
sha512sum file.txt

# OpenSSL:
openssl dgst -sha256 file.txt
openssl dgst -sha512 file.txt

# Verify file integrity:
sha256sum -c checksums.txt
```

### SHA-3 (Keccak)

```
# SHA-3 — NIST FIPS 202, based on Keccak sponge construction
# NOT a replacement for SHA-2 (SHA-2 is not broken)
# Different internal structure than SHA-2 (diversity hedge)

# Variants:
# SHA3-256 — 256-bit output (128-bit collision resistance)
# SHA3-384 — 384-bit output (192-bit collision resistance)
# SHA3-512 — 512-bit output (256-bit collision resistance)
# SHAKE128 — extendable output function (XOF), 128-bit security
# SHAKE256 — extendable output function (XOF), 256-bit security

# Compute SHA-3:
openssl dgst -sha3-256 file.txt

# When to use SHA-3:
# - Compliance requires algorithm diversity (backup if SHA-2 is broken)
# - XOF capability needed (variable-length output with SHAKE)
# - Post-quantum considerations (sponge construction well-analyzed)
# - Generally, SHA-256 is preferred unless specific SHA-3 requirement
```

### BLAKE2

```
# BLAKE2 — fast cryptographic hash (RFC 7693)
# Faster than SHA-256 and SHA-3 in software
# Two variants:
#   BLAKE2b — optimized for 64-bit platforms, up to 512-bit output
#   BLAKE2s — optimized for 8-32-bit platforms, up to 256-bit output

# Features:
# - Keyed hashing (built-in MAC, no HMAC wrapper needed)
# - Personalization (domain separation)
# - Tree hashing (parallel processing)
# - Salt support

# Used in: Argon2 (password hashing), WireGuard, many crypto libraries
# NOT NIST standardized (use SHA-256/SHA-3 for compliance)

# Compute BLAKE2:
b2sum file.txt                    # BLAKE2b (512-bit default)
b2sum -l 256 file.txt             # BLAKE2b with 256-bit output
openssl dgst -blake2b512 file.txt
```

## Message Authentication Codes (MACs)

### HMAC

```
# HMAC — Hash-based Message Authentication Code (RFC 2104)
# Provides message integrity and authenticity
# HMAC-SHA256 is the standard choice

# Construction: HMAC(K, m) = H((K' xor opad) || H((K' xor ipad) || m))
# Where:
#   K' = key (padded/hashed to block size)
#   ipad = 0x36 repeated to block size
#   opad = 0x5c repeated to block size

# Compute HMAC:
openssl dgst -sha256 -hmac "secret-key" -out hmac.bin message.txt

# Verify HMAC (compare computed vs expected):
# Must use constant-time comparison to prevent timing attacks

# HMAC security:
# - HMAC-SHA256: 256-bit security
# - Key should be at least as long as the hash output (32 bytes for SHA-256)
# - Resistant to length-extension attacks (unlike raw SHA-256)

# Common uses:
# - API authentication (HMAC-SHA256 request signing)
# - JWT signature (HS256 = HMAC-SHA256)
# - IPsec integrity verification
# - Cookie signing
```

### CMAC and Poly1305

```
# CMAC — Cipher-based MAC (NIST SP 800-38B)
# Uses AES as the underlying cipher
# AES-CMAC generates 128-bit tags
# Used in: 802.11i (WPA2), EAP-AKA, some government protocols

# Poly1305 — one-time authenticator
# 128-bit tag, extremely fast
# MUST use a unique key per message (derived from ChaCha20 or AES)
# Used exclusively as part of AEAD constructions:
#   ChaCha20-Poly1305 (TLS 1.3, WireGuard)
#   AES-GCM uses GHASH (related polynomial MAC)
# NOT suitable as standalone MAC (key reuse = catastrophic break)
```

## Key Derivation Functions (KDFs)

### HKDF

```
# HKDF — HMAC-based KDF (RFC 5869)
# Two phases: Extract and Expand
# Used to derive multiple keys from a single shared secret

# Extract: PRK = HMAC(salt, input_key_material)
#   Concentrates entropy from IKM into a fixed-length PRK
#   Salt is optional but recommended (can be public)

# Expand: OKM = HMAC(PRK, info || counter)
#   Expands PRK into multiple output keys
#   info parameter provides domain separation

# Used in:
# - TLS 1.3 key schedule (deriving all session keys from shared secret)
# - Signal Protocol (Double Ratchet key derivation)
# - WireGuard handshake

# NOT for password hashing (use Argon2/scrypt/bcrypt instead)
# HKDF is designed for high-entropy inputs (DH shared secrets, random keys)
```

### Password Hashing KDFs

```
# Argon2 — winner of Password Hashing Competition (2015)
# Three variants:
#   Argon2d — data-dependent access (fastest, vulnerable to side-channel)
#   Argon2i — data-independent access (side-channel resistant)
#   Argon2id — hybrid (recommended for password hashing)

# Parameters:
#   Memory (m): minimum 64 MB for interactive, 1 GB for non-interactive
#   Iterations (t): minimum 3 for Argon2id
#   Parallelism (p): number of threads (typically 4)
#   Tag length: 32 bytes

# PBKDF2 — Password-Based Key Derivation Function 2 (RFC 8018)
# Iterative HMAC application
# Minimum iterations: 600,000 (OWASP 2023 recommendation for SHA-256)
# Weakness: trivially parallelizable on GPU (use Argon2 instead)
# Still required by some compliance standards (FIPS 140-2)

# scrypt — memory-hard KDF (RFC 7914)
# Parameters: N (CPU/memory cost), r (block size), p (parallelism)
# Recommended: N=2^17, r=8, p=1 (128 MB memory)
# Better than PBKDF2 but Argon2 is preferred for new designs

# bcrypt — Blowfish-based password hashing
# Cost factor: 10-12 for interactive, 14+ for non-interactive
# Maximum input: 72 bytes (longer passwords truncated!)
# Still widely used but Argon2 preferred for new systems
```

## Digital Signatures

### Signature Algorithms Comparison

```
# Algorithm     Key Size    Sig Size    Speed       Standard
# ─────────────────────────────────────────────────────────────
# RSA-PSS       3072 bit    384 bytes   Medium      NIST, FIPS
# ECDSA P-256   256 bit     64 bytes    Fast        NIST, FIPS
# ECDSA P-384   384 bit     96 bytes    Medium      NIST, FIPS
# Ed25519       256 bit     64 bytes    Very fast   RFC 8032
# Ed448         448 bit     114 bytes   Fast        RFC 8032

# Recommendation:
# - Ed25519 for new applications (fastest, simplest, deterministic)
# - ECDSA P-256 when NIST/FIPS compliance required
# - RSA-PSS 3072+ when RSA interoperability needed
# - Ed448 when 224-bit security level required

# Ed25519 advantages:
# - Deterministic (no random nonce → no nonce-reuse vulnerability)
# - Constant-time by design (resistant to timing side channels)
# - Small keys and signatures (32-byte public key, 64-byte signature)
# - Fast verification (important for certificate chains)

# ECDSA nonce risk:
# ECDSA requires a random nonce (k) for each signature
# If k is reused or predictable: private key is recoverable
# This is how the PS3 was broken (Sony reused k for ECDSA)
# Mitigation: RFC 6979 deterministic nonce generation
```

## Key Exchange Protocols

### Diffie-Hellman Variants

```
# Classical DH (RFC 3526 — MODP groups)
# Based on discrete logarithm problem in multiplicative group
# Group 14: 2048-bit MODP (minimum for new deployments)
# Group 15: 3072-bit MODP (recommended)
# Group 16: 4096-bit MODP (high security)

# ECDHE — Elliptic Curve Diffie-Hellman Ephemeral
# Based on ECDLP in elliptic curve groups
# Curves: P-256, P-384, X25519, X448

# X25519 (Curve25519) — preferred for key exchange
# 128-bit security, 32-byte public keys
# Constant-time, simple implementation, no special cases
# Used in TLS 1.3, WireGuard, Signal, Noise

# Forward secrecy:
# Static DH: same keys for every exchange → no forward secrecy
# Ephemeral DH (DHE/ECDHE): new keys per session → forward secrecy
# Compromise of long-term key does not reveal past session keys
# TLS 1.3 REQUIRES ephemeral key exchange (no static RSA)
```

## Cipher Suite Notation

### TLS Cipher Suite Format

```
# TLS 1.2 cipher suite notation:
# TLS_<KeyExchange>_WITH_<Cipher>_<MAC>
#
# Example: TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
#   Key Exchange: ECDHE (Ephemeral Elliptic Curve Diffie-Hellman)
#   Authentication: RSA (server cert signed with RSA)
#   Cipher: AES-256-GCM (symmetric encryption)
#   MAC: SHA384 (for PRF, GCM provides its own integrity)

# TLS 1.3 cipher suite notation (simplified):
# TLS_<Cipher>_<Hash>
# Key exchange is always ECDHE or DHE (negotiated in extensions)
# Authentication is separate (certificate type)
#
# TLS 1.3 cipher suites:
# TLS_AES_256_GCM_SHA384          (mandatory to implement)
# TLS_AES_128_GCM_SHA256          (mandatory to implement)
# TLS_CHACHA20_POLY1305_SHA256    (recommended)
# TLS_AES_128_CCM_SHA256          (IoT/constrained devices)
# TLS_AES_128_CCM_8_SHA256        (IoT, short tag)

# Check supported cipher suites:
openssl ciphers -v 'TLSv1.3' | head -20
openssl s_client -connect example.com:443 -tls1_3 2>/dev/null | \
  grep "Cipher is"

# Recommended TLS 1.2 cipher suites (ordered preference):
# TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
# TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
# TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256
# TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256
# TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256
# TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
```

### IKE/IPsec Cipher Suite Notation

```
# IKE (Internet Key Exchange) proposal format:
# <Encryption>-<Integrity>-<DH Group>

# Examples:
# aes256-sha256-modp2048     (AES-256-CBC, SHA-256, DH Group 14)
# aes256gcm16-sha384-ecp384  (AES-256-GCM-16, SHA-384, ECP-384)
# chacha20poly1305-sha256-x25519  (ChaCha20-Poly1305, SHA-256, X25519)

# ESP (child SA) proposal format:
# <Encryption>-<Integrity>[-<DH Group for PFS>]

# strongSwan notation:
# ike=aes256gcm16-sha384-ecp384!
# esp=aes256gcm16-sha384-ecp384!
# The ! means strict (no fallback to weaker proposals)

# DH Groups:
# modp2048 (Group 14)  — 112-bit security, minimum acceptable
# modp3072 (Group 15)  — 128-bit security, recommended
# modp4096 (Group 16)  — ~140-bit security
# ecp256   (Group 19)  — 128-bit security (NIST P-256)
# ecp384   (Group 20)  — 192-bit security (NIST P-384)
# x25519   (Group 31)  — 128-bit security (Curve25519)
```

## Post-Quantum Cryptography

### NIST PQC Standards (2024)

```
# NIST finalized three post-quantum algorithms in 2024:

# ML-KEM (formerly CRYSTALS-Kyber) — FIPS 203
# Key Encapsulation Mechanism (replaces RSA/ECDH for key exchange)
# Security: based on Module Learning With Errors (MLWE) lattice problem
# Variants:
#   ML-KEM-512:  128-bit security (public key: 800 bytes, ciphertext: 768 bytes)
#   ML-KEM-768:  192-bit security (public key: 1184 bytes, ciphertext: 1088 bytes)
#   ML-KEM-1024: 256-bit security (public key: 1568 bytes, ciphertext: 1568 bytes)

# ML-DSA (formerly CRYSTALS-Dilithium) — FIPS 204
# Digital Signature Algorithm (replaces RSA/ECDSA for signatures)
# Security: based on Module Learning With Errors + Module SIS
# Variants:
#   ML-DSA-44:  ~128-bit security (public key: 1312 bytes, sig: 2420 bytes)
#   ML-DSA-65:  ~192-bit security (public key: 1952 bytes, sig: 3293 bytes)
#   ML-DSA-87:  ~256-bit security (public key: 2592 bytes, sig: 4595 bytes)

# SLH-DSA (formerly SPHINCS+) — FIPS 205
# Hash-based Digital Signature (stateless)
# Security: based only on hash function security (most conservative)
# Larger signatures (~8-50 KB) but minimal security assumptions
# Backup algorithm in case lattice problems are broken

# Status for deployment:
# - TLS: hybrid key exchange (X25519 + ML-KEM-768) in Chrome and Firefox
# - Signal Protocol: PQXDH (X25519 + ML-KEM-768 hybrid)
# - SSH: hybrid key exchange drafts in progress
# - IPsec: RFC 9370 (multiple key exchanges for hybrid PQ)
```

### Hybrid Key Exchange

```
# Hybrid = classical + post-quantum combined
# Why hybrid: post-quantum algorithms are newer, less battle-tested
# If PQ algorithm is broken: classical algorithm still protects
# If classical is broken by quantum computer: PQ algorithm still protects

# TLS 1.3 hybrid key exchange (X25519Kyber768):
#   Client sends: X25519 key share + ML-KEM-768 encapsulation key
#   Server responds: X25519 key share + ML-KEM-768 ciphertext
#   Shared secret = HKDF(X25519_secret || ML-KEM_secret)

# Impact on handshake size:
# Classical TLS 1.3 ClientHello: ~300 bytes
# Hybrid PQ ClientHello: ~1500 bytes (ML-KEM public key adds ~1200 bytes)
# Minimal latency impact (1-RTT handshake preserved)

# Check if your browser supports PQ:
# Visit: https://pq.cloudflareresearch.com/
```

## Algorithm Selection Guide

### By Use Case

```
# Symmetric Encryption:
#   General purpose:     AES-256-GCM (with hardware AES-NI)
#   Mobile/embedded:     ChaCha20-Poly1305 (no AES-NI needed)
#   Disk encryption:     AES-256-XTS (full-disk), AES-256-GCM (file-level)
#   Legacy (avoid):      3DES, RC4, Blowfish, AES-CBC without HMAC

# Asymmetric / Key Exchange:
#   TLS key exchange:    X25519 (or hybrid X25519+ML-KEM-768)
#   IPsec key exchange:  ECDH P-256/P-384 or X25519
#   General DH:          X25519 (preferred), ECDH P-256 (FIPS)
#   Legacy (avoid):      RSA key transport, DH < 2048-bit

# Digital Signatures:
#   SSH keys:            Ed25519
#   TLS certificates:    ECDSA P-256 or RSA-3072 (CA ecosystem)
#   Code signing:        RSA-3072+ or ECDSA P-256
#   New protocols:       Ed25519 (or ML-DSA for post-quantum)
#   Legacy (avoid):      RSA-1024, DSA, ECDSA with random nonces

# Hashing:
#   General purpose:     SHA-256
#   High security:       SHA-384 or SHA-512
#   Performance:         BLAKE2b (non-FIPS environments)
#   Password hashing:    Argon2id (NEVER raw SHA-256 on passwords)
#   Legacy (avoid):      MD5, SHA-1

# MACs:
#   General purpose:     HMAC-SHA256
#   Within AEAD:         GCM (GHASH) or Poly1305 (handled by cipher)
#   Legacy (avoid):      HMAC-MD5, HMAC-SHA1 (still secure but deprecated)

# KDFs:
#   From DH secrets:     HKDF-SHA256
#   Password hashing:    Argon2id > scrypt > bcrypt > PBKDF2
#   Legacy (avoid):      Raw hash iterations, single HMAC pass
```

### Key Length Equivalencies

```
# Equivalent security levels across algorithm families:
#
# Security  Symmetric  RSA/DH     ECC        Hash       Post-Quantum
# (bits)    (bits)     (bits)     (bits)     (bits)     (ML-KEM)
# ──────────────────────────────────────────────────────────────────
# 80        80         1024       160        160        N/A (insecure)
# 112       112        2048       224        224        N/A
# 128       128        3072       256        256        ML-KEM-512
# 192       192        7680       384        384        ML-KEM-768
# 256       256        15360      512        512        ML-KEM-1024
#
# Source: NIST SP 800-57 Part 1, Table 2

# Minimum recommended today (2026):
# 128-bit security minimum for all new deployments
# This means: AES-128+, RSA-3072+, ECDSA P-256+, SHA-256+
```

## Tips

- Always use AEAD ciphers (AES-GCM or ChaCha20-Poly1305) for symmetric encryption. Never use CBC mode without a separate HMAC, and even then prefer AEAD.
- Never reuse a nonce with the same key in AES-GCM or ChaCha20-Poly1305. Nonce reuse in GCM reveals the GHASH authentication key.
- Use Ed25519 for SSH keys and new signature applications. It is faster, simpler, and immune to nonce-reuse attacks (deterministic signatures).
- For password hashing, use Argon2id with at least 64 MB of memory. Never use raw SHA-256 or even PBKDF2 with low iteration counts.
- RSA key transport (where the client encrypts the premaster secret with the server's RSA public key) does not provide forward secrecy. Use ECDHE for key exchange.
- When TLS 1.3 is available, use it. It removes all insecure cipher suites, mandates forward secrecy, and simplifies the handshake.
- Start planning for post-quantum migration. Use hybrid key exchange (X25519 + ML-KEM) where supported. Harvest-now-decrypt-later attacks mean data encrypted today may be vulnerable to future quantum computers.
- For FIPS 140 compliance, use NIST-approved algorithms only: AES, SHA-2/SHA-3, RSA, ECDSA (NIST curves), HMAC, HKDF, PBKDF2. Ed25519 and ChaCha20 are not FIPS-approved.
- Compare algorithm performance on your target platform. AES-GCM is faster with AES-NI; ChaCha20-Poly1305 is faster without it.

## See Also

- tls, ipsec, pki, openssl, wireguard, gpg, ssh, cryptography, post-quantum-crypto

## References

- [NIST FIPS 197 — Advanced Encryption Standard (AES)](https://csrc.nist.gov/publications/detail/fips/197/final)
- [NIST SP 800-38D — Recommendation for GCM Mode](https://csrc.nist.gov/publications/detail/sp/800-38d/final)
- [NIST SP 800-57 Part 1 — Key Management Recommendations](https://csrc.nist.gov/publications/detail/sp/800-57-part-1/rev-5/final)
- [NIST FIPS 203 — ML-KEM (Module-Lattice Key Encapsulation)](https://csrc.nist.gov/publications/detail/fips/203/final)
- [NIST FIPS 204 — ML-DSA (Module-Lattice Digital Signature)](https://csrc.nist.gov/publications/detail/fips/204/final)
- [NIST FIPS 205 — SLH-DSA (Stateless Hash-Based Digital Signature)](https://csrc.nist.gov/publications/detail/fips/205/final)
- [RFC 8439 — ChaCha20 and Poly1305 for IETF Protocols](https://www.rfc-editor.org/rfc/rfc8439)
- [RFC 8032 — Edwards-Curve Digital Signature Algorithm (EdDSA)](https://www.rfc-editor.org/rfc/rfc8032)
- [RFC 5869 — HMAC-based Extract-and-Expand Key Derivation Function (HKDF)](https://www.rfc-editor.org/rfc/rfc5869)
- [RFC 9106 — Argon2 Memory-Hard Function](https://www.rfc-editor.org/rfc/rfc9106)
- [RFC 7748 — Elliptic Curves for Security (X25519, X448)](https://www.rfc-editor.org/rfc/rfc7748)
- [RFC 8446 — The Transport Layer Security (TLS) Protocol Version 1.3](https://www.rfc-editor.org/rfc/rfc8446)
- [Bernstein, D.J. — Curve25519: New Diffie-Hellman Speed Records](https://cr.yp.to/ecdh/curve25519-20060209.pdf)
