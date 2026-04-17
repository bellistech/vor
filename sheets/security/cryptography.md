# Applied Cryptography (Practical cryptographic primitives, protocols, and best practices)

## Symmetric Encryption

### Overview

```
Symmetric: Same key for encryption and decryption.
Fast, suitable for bulk data. Key distribution is the hard problem.

Common algorithms:
  AES-256-GCM    — Authenticated encryption, 256-bit key, recommended default
  AES-128-GCM    — Authenticated encryption, 128-bit key, still secure
  ChaCha20-Poly1305 — Authenticated encryption, fast in software (no AES-NI)
  AES-256-CBC    — Block cipher mode, requires separate MAC (use GCM instead)

Deprecated / avoid:
  DES            — 56-bit key, trivially breakable
  3DES           — Slow, 112-bit effective, deprecated by NIST (2023)
  RC4            — Stream cipher, multiple known attacks
  Blowfish       — 64-bit block, birthday bound issues
  AES-ECB        — No diffusion across blocks, leaks patterns
```

### OpenSSL Symmetric Encryption

```bash
# Encrypt a file with AES-256-GCM
openssl enc -aes-256-gcm -salt -pbkdf2 -iter 600000 \
  -in plaintext.dat -out encrypted.dat

# Decrypt
openssl enc -d -aes-256-gcm -pbkdf2 -iter 600000 \
  -in encrypted.dat -out plaintext.dat

# Encrypt with explicit key and IV (for automation, not passwords)
openssl enc -aes-256-gcm \
  -K $(openssl rand -hex 32) \
  -iv $(openssl rand -hex 12) \
  -in plaintext.dat -out encrypted.dat
```

## Asymmetric Encryption

### Key Size Recommendations (NIST SP 800-57)

```
Algorithm        Minimum Key Size    Recommended       Security Level
RSA              2048 bits           3072+ bits         112-128 bits
ECDSA/ECDH       256 bits (P-256)    384 bits (P-384)   128-192 bits
Ed25519          256 bits            256 bits           ~128 bits
Ed448            448 bits            448 bits           ~224 bits
X25519 (DH)      256 bits            256 bits           ~128 bits

Post-quantum (NIST PQC standards):
ML-KEM (Kyber)   768/1024            1024               128-256 bits
ML-DSA (Dilithium) 2/3/5             3 or 5             128-256 bits
```

### Key Generation

```bash
# Generate RSA 4096-bit key pair
openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:4096 \
  -out private.pem
openssl pkey -in private.pem -pubout -out public.pem

# Generate Ed25519 key pair (recommended for new systems)
openssl genpkey -algorithm Ed25519 -out private.pem
openssl pkey -in private.pem -pubout -out public.pem

# Generate ECDSA P-384 key pair
openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:P-384 \
  -out private.pem
openssl pkey -in private.pem -pubout -out public.pem

# Generate X25519 key pair (for key agreement)
openssl genpkey -algorithm X25519 -out private.pem
openssl pkey -in private.pem -pubout -out public.pem
```

## Hash Functions

### Current Recommendations

```
Algorithm     Output Size    Status              Use Case
SHA-256       256 bits       Recommended          General purpose, TLS, certificates
SHA-384       384 bits       Recommended          Higher security margin
SHA-512       512 bits       Recommended          Large data, some protocols
SHA-3-256     256 bits       Recommended          Alternative to SHA-2 family
SHA-3-512     512 bits       Recommended          Alternative to SHA-2 family
BLAKE2b       Up to 512      Recommended          Fast hashing, checksums
BLAKE2s       Up to 256      Recommended          Fast hashing (32-bit optimized)
BLAKE3        256 bits       Recommended          Very fast, parallelizable

Deprecated / avoid:
MD5           128 bits       Broken               Collision attacks practical
SHA-1         160 bits       Broken               Collision attacks demonstrated (SHAttered)
```

### Hashing Commands

```bash
# SHA-256 hash of a file
sha256sum file.dat
openssl dgst -sha256 file.dat

# SHA-3-256
openssl dgst -sha3-256 file.dat

# BLAKE2b-256 (via b2sum)
b2sum file.dat

# BLAKE3 (via b3sum)
b3sum file.dat

# Verify file integrity
sha256sum -c checksums.txt
```

## HMAC (Hash-Based Message Authentication Code)

```bash
# Generate HMAC-SHA256
openssl dgst -sha256 -hmac "secret_key" file.dat

# Generate HMAC with key from file
openssl dgst -sha256 -mac HMAC -macopt hexkey:$(xxd -p keyfile) file.dat

# HMAC in Python (for reference)
# import hmac, hashlib
# hmac.new(key, message, hashlib.sha256).hexdigest()
```

```
HMAC properties:
  - Provides message authentication (integrity + authenticity)
  - Resistant to length-extension attacks (unlike raw SHA-256)
  - Used in TLS, JWT (HS256), API authentication
  - Key should be at least as long as the hash output
```

## Key Derivation Functions

### Password Hashing

```
Algorithm   Parameters                          Status
Argon2id    Memory: 64MB+, Iterations: 3+,      Recommended (winner of PHC)
            Parallelism: 4+
scrypt      N=2^15+, r=8, p=1                   Recommended
bcrypt      Cost factor: 12+                     Acceptable (max 72-byte input)
PBKDF2      Iterations: 600,000+ (SHA-256)       Acceptable (NIST approved)

Avoid:
  MD5-based hashing (md5crypt)
  SHA-1 with low iterations
  Plain SHA-256/SHA-512 of passwords
  Unsalted hashes of any kind
```

```bash
# Generate Argon2id hash (via argon2 CLI)
echo -n "password" | argon2 $(openssl rand -hex 16) -id -t 3 -m 16 -p 4

# Generate bcrypt hash (via htpasswd)
htpasswd -nbBC 12 "" "password" | cut -d: -f2

# PBKDF2 with OpenSSL (key derivation, not storage)
openssl kdf -keylen 32 -kdfopt digest:SHA256 \
  -kdfopt pass:password -kdfopt salt:$(openssl rand -hex 16) \
  -kdfopt iter:600000 PBKDF2

# Verify password hash strength
# Python: pip install argon2-cffi
# from argon2 import PasswordHasher
# ph = PasswordHasher(time_cost=3, memory_cost=65536, parallelism=4)
# hash = ph.hash("password")
# ph.verify(hash, "password")
```

### Key Derivation for Encryption

```bash
# HKDF (extract-and-expand, RFC 5869)
# Used to derive multiple keys from one shared secret
# Not for passwords — use Argon2/scrypt/PBKDF2 for passwords

# HKDF via OpenSSL
openssl kdf -keylen 32 -kdfopt digest:SHA256 \
  -kdfopt key:$(openssl rand -hex 32) \
  -kdfopt info:"application-key-v1" \
  -kdfopt salt:$(openssl rand -hex 32) HKDF
```

## Digital Signatures

```bash
# Sign a file with Ed25519
openssl pkeyutl -sign -inkey private.pem \
  -in file.dat -out file.sig

# Verify signature
openssl pkeyutl -verify -pubin -inkey public.pem \
  -in file.dat -sigfile file.sig

# Sign with RSA-PSS (preferred over PKCS#1 v1.5)
openssl dgst -sha256 -sigopt rsa_padding_mode:pss \
  -sign private.pem -out file.sig file.dat

# Verify RSA-PSS signature
openssl dgst -sha256 -sigopt rsa_padding_mode:pss \
  -verify public.pem -signature file.sig file.dat

# Sign a Git commit with GPG
git commit -S -m "Signed commit"

# Verify Git commit signatures
git log --show-signature
```

## Certificate Chains and PKI

```bash
# Generate a self-signed CA certificate
openssl req -x509 -newkey rsa:4096 -sha256 -days 3650 \
  -keyout ca-key.pem -out ca-cert.pem \
  -subj "/CN=My Root CA/O=Example/C=US" -nodes

# Generate a certificate signing request (CSR)
openssl req -newkey rsa:3072 -keyout server-key.pem -out server.csr \
  -subj "/CN=server.example.com" -nodes \
  -addext "subjectAltName=DNS:server.example.com,DNS:www.example.com"

# Sign CSR with CA
openssl x509 -req -in server.csr -CA ca-cert.pem -CAkey ca-key.pem \
  -CAcreateserial -out server-cert.pem -days 365 -sha256 \
  -extfile <(printf "subjectAltName=DNS:server.example.com,DNS:www.example.com")

# View certificate details
openssl x509 -in server-cert.pem -text -noout

# Verify certificate chain
openssl verify -CAfile ca-cert.pem server-cert.pem

# Check certificate expiry
openssl x509 -in server-cert.pem -noout -enddate

# Inspect remote server certificate
openssl s_client -connect example.com:443 -servername example.com </dev/null 2>/dev/null | \
  openssl x509 -text -noout
```

## TLS Cipher Suites

### Recommended (TLS 1.3)

```
TLS 1.3 cipher suites (all are considered secure):
  TLS_AES_256_GCM_SHA384
  TLS_AES_128_GCM_SHA256
  TLS_CHACHA20_POLY1305_SHA256

TLS 1.3 removes negotiation of key exchange and signature algorithms
from the cipher suite — they are configured separately.
```

### Recommended (TLS 1.2)

```
Recommended TLS 1.2 cipher suites (forward secrecy required):
  ECDHE-ECDSA-AES256-GCM-SHA384
  ECDHE-RSA-AES256-GCM-SHA384
  ECDHE-ECDSA-CHACHA20-POLY1305
  ECDHE-RSA-CHACHA20-POLY1305
  ECDHE-ECDSA-AES128-GCM-SHA256
  ECDHE-RSA-AES128-GCM-SHA256
```

### Deprecated / Avoid

```
Avoid these cipher suites and protocols:
  SSLv2, SSLv3, TLS 1.0, TLS 1.1  — Protocol versions, all deprecated
  RC4                               — Broken stream cipher
  DES, 3DES                         — Weak/deprecated block ciphers
  CBC mode (in TLS)                 — Padding oracle attacks (POODLE, Lucky13)
  RSA key exchange (no ECDHE)       — No forward secrecy
  NULL ciphers                      — No encryption at all
  EXPORT ciphers                    — Deliberately weakened (FREAK, Logjam)
  MD5 for signatures                — Collision attacks
```

```bash
# Test server cipher suite support
nmap --script ssl-enum-ciphers -p 443 example.com

# OpenSSL cipher suite testing
openssl s_client -connect example.com:443 -tls1_3

# List available cipher suites
openssl ciphers -v 'TLSv1.3'
openssl ciphers -v 'ECDHE+AESGCM:ECDHE+CHACHA20'

# testssl.sh — comprehensive TLS testing
./testssl.sh example.com
```

## Secure Random Generation

```bash
# Generate cryptographically secure random bytes
openssl rand -hex 32        # 32 bytes as hex (64 hex chars)
openssl rand -base64 32     # 32 bytes as base64

# Read from system CSPRNG
head -c 32 /dev/urandom | xxd -p

# Generate a random UUID
uuidgen
cat /proc/sys/kernel/random/uuid   # Linux only

# Generate a random password
openssl rand -base64 24 | tr -d '/+='

# Check available system entropy (Linux)
cat /proc/sys/kernel/random/entropy_avail
```

```
Secure random sources:
  /dev/urandom    — Non-blocking, suitable for all cryptographic use
  /dev/random     — Blocking on older kernels; on Linux 5.6+ same as urandom
  getrandom()     — Preferred syscall on modern Linux
  CryptGenRandom  — Windows CSPRNG
  arc4random()    — BSD/macOS CSPRNG

Never use:
  rand() / srand()          — Predictable PRNG, not cryptographic
  Math.random() (JS)        — Not cryptographic
  time-based seeds alone    — Predictable
```

## Common Cryptographic Mistakes

```
1. Using ECB mode             — Leaks patterns in plaintext
2. Reusing nonces/IVs         — Catastrophic for GCM and stream ciphers
3. Using MD5/SHA-1            — Broken hash functions
4. Rolling your own crypto    — Use vetted libraries (libsodium, OpenSSL, Go crypto)
5. Hardcoded keys             — Keys must be managed, rotated, and protected
6. No authentication (MAC)    — Encrypt-then-MAC or use AEAD (GCM, Poly1305)
7. Weak random generation     — Always use CSPRNG
8. Short passwords with PBKDF2 and low iterations — Brute-forceable
9. Comparing MACs with ==     — Timing side-channel; use constant-time comparison
10. Ignoring key rotation     — Keys have lifetimes; plan for rotation
11. Storing keys alongside encrypted data — Defeats the purpose
12. Using RSA PKCS#1 v1.5 for encryption — Bleichenbacher attack; use OAEP
```

## Password Hashing Best Practices

```
For user password storage:
  1. Use Argon2id as first choice (memory-hard, side-channel resistant)
  2. scrypt as second choice (memory-hard)
  3. bcrypt as third choice (12+ rounds, 72-byte input limit)
  4. PBKDF2-HMAC-SHA256 if FIPS required (600,000+ iterations per OWASP 2023)

Every hash must include:
  - Unique random salt (16+ bytes)
  - Algorithm identifier in stored hash
  - Configurable work factor for future increases

Never:
  - Store passwords in plaintext or reversible encryption
  - Use a fast hash (SHA-256) directly on passwords
  - Use the same salt for multiple passwords
  - Truncate passwords before hashing (except bcrypt's 72-byte limit)
```

## Tips

- Default to AES-256-GCM or ChaCha20-Poly1305 for symmetric encryption; both provide authenticated encryption.
- Use Ed25519 for new digital signature and SSH key deployments; it is fast and has a clean security story.
- Always use AEAD (authenticated encryption with associated data); never encrypt without authenticating.
- Use HKDF for deriving multiple keys from a shared secret; use Argon2id for hashing passwords.
- For TLS, require TLS 1.3 where possible; if TLS 1.2 is needed, restrict to ECDHE cipher suites with GCM.
- Rotate keys on a defined schedule and have a documented key compromise recovery plan.
- Use constant-time comparison functions for MACs and password hashes to prevent timing attacks.
- Prefer well-audited libraries (libsodium, Go `crypto/*`, OpenSSL) over custom implementations.

## See Also

- tls, pki, openssl, gpg, ssh, post-quantum-crypto, crypto-protocols

## References

- [NIST SP 800-57 - Key Management Recommendations](https://csrc.nist.gov/publications/detail/sp/800-57-part-1/rev-5/final)
- [NIST SP 800-175B - Cryptographic Standards](https://csrc.nist.gov/publications/detail/sp/800-175b/rev-1/final)
- [OWASP Password Storage Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html)
- [OWASP Cryptographic Storage Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Cryptographic_Storage_Cheat_Sheet.html)
- [Mozilla SSL Configuration Generator](https://ssl-config.mozilla.org/)
- [testssl.sh](https://github.com/drwetter/testssl.sh)
- [libsodium Documentation](https://doc.libsodium.org/)
- [RFC 5869 - HKDF](https://datatracker.ietf.org/doc/html/rfc5869)
- [Password Hashing Competition (Argon2)](https://www.password-hashing.net/)
- [NIST Post-Quantum Cryptography](https://csrc.nist.gov/projects/post-quantum-cryptography)
