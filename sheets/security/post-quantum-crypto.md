# Post-Quantum Cryptography (FIPS 203/204/205)

Operational guide to NIST-standardized post-quantum algorithms ML-KEM (FIPS 203), ML-DSA (FIPS 204), SLH-DSA (FIPS 205), and the deprecation timeline for classical RSA/ECC — with hybrid migration patterns, parameter selection, and deployment guidance for TLS, SSH, VPN, and code signing.

## The NIST PQC Standards

### Finalized Standards (August 2024)
```
FIPS 203 — ML-KEM (Module-Lattice-Based Key Encapsulation)
  Based on: CRYSTALS-Kyber
  Purpose: key establishment / key encapsulation
  Parameters: ML-KEM-512, ML-KEM-768, ML-KEM-1024

FIPS 204 — ML-DSA (Module-Lattice-Based Digital Signatures)
  Based on: CRYSTALS-Dilithium
  Purpose: digital signatures (primary recommendation)
  Parameters: ML-DSA-44, ML-DSA-65, ML-DSA-87

FIPS 205 — SLH-DSA (Stateless Hash-Based Signatures)
  Based on: SPHINCS+
  Purpose: digital signatures (conservative backup)
  Parameters: SLH-DSA-SHA2-128s/128f, 192s/192f, 256s/256f
             SLH-DSA-SHAKE-128s/128f, 192s/192f, 256s/256f
             (s = small signatures, f = fast signing)

FIPS 206 — FN-DSA (FALCON-based, draft 2025)
  Based on: FALCON
  Purpose: signatures when small size is critical
  Parameters: FN-DSA-512, FN-DSA-1024
```

## Parameter Sizes (Bytes)

### ML-KEM Sizes
| Parameter | Security | Public Key | Ciphertext | Secret Key | Shared Secret |
|-----------|----------|------------|------------|------------|---------------|
| ML-KEM-512 | Level 1 (AES-128) | 800 | 768 | 1632 | 32 |
| ML-KEM-768 | Level 3 (AES-192) | 1184 | 1088 | 2400 | 32 |
| ML-KEM-1024 | Level 5 (AES-256) | 1568 | 1568 | 3168 | 32 |

### ML-DSA Sizes
| Parameter | Security | Public Key | Signature |
|-----------|----------|------------|-----------|
| ML-DSA-44 | Level 2 | 1312 | 2420 |
| ML-DSA-65 | Level 3 | 1952 | 3309 |
| ML-DSA-87 | Level 5 | 2592 | 4627 |

### SLH-DSA Sizes (selected)
| Parameter | Security | Public Key | Signature | Sign Speed |
|-----------|----------|------------|-----------|-----------|
| SLH-DSA-SHA2-128s | Level 1 | 32 | 7856 | slow |
| SLH-DSA-SHA2-128f | Level 1 | 32 | 17088 | fast |
| SLH-DSA-SHA2-256s | Level 5 | 64 | 29792 | slow |
| SLH-DSA-SHA2-256f | Level 5 | 64 | 49856 | fast |

## Migration Timeline

### NIST Deprecation Schedule (SP 800-131A Rev 3 draft)
```
2025 — PQC standards finalized; begin migration planning
2030 — RSA/ECDSA/ECDH below 112-bit classical security DEPRECATED
2035 — RSA, ECDSA, ECDH, DH, FFDHE at any size DISALLOWED for USG
2030 — CNSA 2.0 requires PQC for National Security Systems in new designs
```

### Cryptographic Inventory (First Step)
```bash
# Find TLS endpoints and their algorithms
nmap --script ssl-enum-ciphers -p 443 example.com

# Inventory certificates and their algorithms
openssl s_client -connect example.com:443 -showcerts </dev/null 2>/dev/null \
  | openssl x509 -noout -text | grep -E "Signature Algorithm|Public Key Algorithm"

# Find SSH host keys
ssh-keyscan -t rsa,ecdsa,ed25519 host.example.com

# Inventory GPG keys
gpg --list-keys --with-colons | awk -F: '/^pub/ {print $4, $5}'

# Scan codebase for crypto primitives (starting point)
rg -t go -t python -t rust \
  "(RSA|ECDSA|ECDH|DH|rsa\.|ecdsa\.|x509\.|crypto/rsa|crypto/ecdsa)" \
  --stats
```

## Hybrid Key Exchange

### X25519 + ML-KEM-768 (RFC 9794 / draft-ietf-tls-hybrid-design)
```
Shared secret = X25519(sk_A, pk_B) || ML-KEM.Decaps(sk_A^pq, ct_B)
              = classical_ss || pq_ss

Rationale: breaks ONLY if BOTH classical AND PQC primitive break.
Defense-in-depth for the transition period.

TLS 1.3 named groups (codepoints pending):
  X25519MLKEM768   — hybrid (recommended transition)
  SecP256r1MLKEM768 — NIST-curve hybrid
  MLKEM512 / MLKEM768 / MLKEM1024 — PQ-only (post-transition)
```

### OpenSSL 3.5+ Hybrid Groups
```bash
# Check OpenSSL PQC support
openssl list -kem-algorithms
openssl list -signature-algorithms

# Generate ML-KEM keypair
openssl genpkey -algorithm ML-KEM-768 -out mlkem768.key

# Hybrid TLS server (OpenSSL 3.5+ with oqs provider or built-in)
openssl s_server -key server.key -cert server.crt \
  -groups X25519MLKEM768:X25519:SecP256r1MLKEM768 \
  -tls1_3

# Verify client-server negotiated a hybrid group
openssl s_client -connect example.com:443 -groups X25519MLKEM768 \
  -tls1_3 2>&1 | grep "Negotiated"
```

## Signatures

### ML-DSA for Code Signing
```bash
# OpenSSL 3.5+ signature with ML-DSA-65
openssl genpkey -algorithm ML-DSA-65 -out signer.key
openssl pkey -in signer.key -pubout -out signer.pub

# Sign a binary
openssl pkeyutl -sign -rawin -in binary -inkey signer.key -out binary.sig

# Verify
openssl pkeyutl -verify -rawin -in binary -sigfile binary.sig -pubin -inkey signer.pub
```

### Composite (Hybrid) Signatures
```
X.509 composite signatures (draft-ietf-lamps-pq-composite-sigs):
  composite = Sign_classical(m) || Sign_pq(m)

Applications:
  - Code signing during transition
  - Firmware update authentication
  - Document signing (long-term archival)
```

### SSH Host Keys with ML-DSA
```bash
# OpenSSH 10.0+ (with PQC patches or upstream support)
ssh-keygen -t ml-dsa-65 -f ~/.ssh/id_ml_dsa_65

# Hybrid host key
# /etc/ssh/sshd_config
HostKey /etc/ssh/ssh_host_ed25519_key
HostKey /etc/ssh/ssh_host_ml_dsa_65_key
HostKeyAlgorithms ssh-ed25519,ml-dsa-65
```

## IPsec / VPN Migration

### IKEv2 Hybrid Post-Quantum (RFC 9370)
```
IKEv2 supports multiple additional key exchanges (AdditionalKE1..7)
allowing a classical DH followed by one or more PQC KEMs:

IKE_SA_INIT: classical DH (X25519 or P-256)
IKE_INTERMEDIATE (RFC 9242): ML-KEM-768 KEM exchange
  (optionally) IKE_INTERMEDIATE: additional PQC KEM

strongSwan configuration (6.0+):
  ike = aes256gcm16-prfsha384-x25519-mlkem768
  esp = aes256gcm16
```

## Code Migration Patterns

### Algorithm Abstraction Layer
```go
// Go: wrap signing behind an interface so the PQ switch is a one-line change
type Signer interface {
    Sign(message []byte) ([]byte, error)
    Verify(message, signature []byte) bool
    PublicKey() []byte
    Algorithm() string
}

// Swap implementations: ECDSASigner -> MLDSASigner -> CompositeSigner
// without touching call sites
```

### Python with liboqs
```python
import oqs

# Signature
with oqs.Signature("ML-DSA-65") as signer:
    public_key = signer.generate_keypair()
    signature = signer.sign(b"my message")
    # verify
    with oqs.Signature("ML-DSA-65") as verifier:
        assert verifier.verify(b"my message", signature, public_key)

# KEM
with oqs.KeyEncapsulation("ML-KEM-768") as kem:
    public_key = kem.generate_keypair()
    ciphertext, shared_secret_send = kem.encap_secret(public_key)
    shared_secret_recv = kem.decap_secret(ciphertext)
    assert shared_secret_send == shared_secret_recv
```

## PKI and Certificates

### X.509 with PQC (RFC 9764 / draft-ietf-lamps-dilithium-certificates)
```bash
# Generate ML-DSA-65 CA (OpenSSL 3.5+)
openssl req -x509 -newkey ml-dsa-65 -keyout ca.key -out ca.crt \
  -days 3650 -nodes -subj "/CN=PQC-CA"

# Issue end-entity certificate with ML-DSA-65
openssl req -new -newkey ml-dsa-65 -keyout server.key -out server.csr \
  -nodes -subj "/CN=server.example.com"
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
  -out server.crt -days 365

# Inspect
openssl x509 -in server.crt -noout -text | head -30
```

## Harvest-Now-Decrypt-Later (HNDL) Threat

### What Must Migrate First
```
Data with long confidentiality lifetime (10+ years):
  - Health records
  - Financial records (tax retention)
  - Intellectual property
  - Classified material
  - Legal documents
  - DRM-protected content

Communications that adversaries can store now for future decryption:
  - Bulk traffic intercepts
  - VPN tunnels carrying long-life secrets
  - Certificate private keys (for MITM later)
  - Encrypted backups

Migration priority = confidentiality_lifetime - years_until_CRQC
  Values < 0 are actively leaking in quantum sense today
```

## Tips
- Begin with a cryptographic inventory — you cannot migrate what you cannot see; inventory reveals hardcoded RSA/ECDSA lurking in scripts, firmware, and protobuf files
- Prioritize by confidentiality lifetime minus estimated years until a cryptographically relevant quantum computer — high-lifetime secrets transiting public networks today are the HNDL victims
- Adopt hybrid (classical + PQC) for key exchange during transition; breaking hybrid requires breaking BOTH halves, so defense-in-depth is effectively free
- For signatures, prefer ML-DSA-65 as the primary workhorse; reserve SLH-DSA for long-term archival where conservative hash-based security outweighs size
- Design algorithm abstraction layers early so that swapping Ed25519 for ML-DSA-65 is a one-line change, not a refactor
- Budget for the size increase — ML-DSA-65 signatures are 3309 bytes vs 64 for Ed25519; path MTU, certificate chains, firmware slots, and bandwidth all feel this
- For constrained devices, evaluate FN-DSA (FALCON) when it finalizes — smaller signatures but floating-point complexity needs careful side-channel review
- Do not roll your own PQC — use NIST reference implementations, liboqs, or mainlined OpenSSL 3.5+; side-channel vulnerabilities are subtle
- Track draft IETF specs: hybrid TLS (RFC 9794), composite signatures, X.509 profiles, IKEv2 (RFC 9370), SSH — codepoints are still being finalized
- Test interoperability across vendors early; the PQC ecosystem has subtle encoding differences (seed-based vs expanded keys, domain separation)
- Re-issue CA certificates first, then leaf certificates — CAs typically have the longest validity periods and highest HNDL exposure
- Run liboqs benchmarks on your target hardware; PQC signatures verify faster than ECDSA on modern CPUs but ML-DSA signing is CPU-intensive

## See Also
- tls, crypto-protocols, cryptography, pki, openssl, ssh, ipsec, wireguard, supply-chain-security, sbom, lattice-crypto

## References
- [FIPS 203 — ML-KEM](https://csrc.nist.gov/pubs/fips/203/final)
- [FIPS 204 — ML-DSA](https://csrc.nist.gov/pubs/fips/204/final)
- [FIPS 205 — SLH-DSA](https://csrc.nist.gov/pubs/fips/205/final)
- [NIST PQC Project](https://csrc.nist.gov/projects/post-quantum-cryptography)
- [NSA CNSA 2.0](https://media.defense.gov/2022/Sep/07/2003071834/-1/-1/0/CSA_CNSA_2.0_ALGORITHMS_.PDF)
- [IETF Hybrid Key Exchange in TLS 1.3](https://datatracker.ietf.org/doc/draft-ietf-tls-hybrid-design/)
- [Open Quantum Safe (liboqs)](https://openquantumsafe.org/)
- [OpenSSL 3.5 PQC Release Notes](https://github.com/openssl/openssl/blob/master/CHANGES.md)
