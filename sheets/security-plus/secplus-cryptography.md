# Cryptographic Solutions (Security+ SY0-701 Objective 1.4)

> Importance of appropriate cryptographic solutions: PKI, encryption levels, symmetric vs asymmetric algorithms, key exchange, TPM/HSM/enclaves, obfuscation, hashing/salting/signatures, key stretching, blockchain, and the full certificate lifecycle.

## Public Key Infrastructure (PKI) Fundamentals

```
# PKI (Public Key Infrastructure) — the entire system of hardware,
#   software, policies, and procedures for creating, distributing,
#   storing, and revoking digital certificates that bind public keys
#   to identities
#
# Public key   — shared with anyone; encrypts data TO the owner,
#                verifies signatures FROM the owner
# Private key  — never shared; decrypts data sent to the owner,
#                creates signatures; compromise = full identity theft
#
# The asymmetric contract (memorize both directions):
#   Encrypt with recipient's PUBLIC key  → only their PRIVATE key decrypts
#     (confidentiality)
#   Sign with sender's PRIVATE key       → anyone verifies with PUBLIC key
#     (integrity + authentication + non-repudiation)
#
# Key escrow — a trusted third party (or internal security office) holds
#   a copy of private keys so the organization can recover encrypted
#   data if the key holder leaves, dies, or loses the key
#   Exam cue: "organization must be able to decrypt employee data if the
#   employee is unavailable" → key escrow
#   Related: recovery agent — an authorized entity that can use escrowed
#   key material to decrypt on the org's behalf
#
# Non-repudiation — the signer cannot deny having signed; ONLY provided
#   by asymmetric digital signatures (private key is unique to signer).
#   Symmetric MACs (Message Authentication Codes, incl. HMAC) do NOT
#   provide non-repudiation — both parties hold the same key, so either
#   could have produced the tag.
```

## Symmetric vs Asymmetric Encryption

```
# Symmetric — ONE shared secret key encrypts and decrypts
#   + Fast (orders of magnitude faster), low CPU, ideal for bulk data
#   - Key distribution problem: how do both sides get the key safely?
#   - Key count scales badly: n users need n(n-1)/2 pairwise keys
#
# Asymmetric — key PAIR (public + private), mathematically related
#   + Solves key distribution (publish the public key freely)
#   + Enables digital signatures and non-repudiation
#   - Slow, CPU-heavy, small payload limits (RSA can only encrypt data
#     smaller than its modulus)
#   - n users need only n key pairs
#
# Hybrid encryption (how TLS, GPG, S/MIME actually work):
#   1. Asymmetric crypto (or DH/ECDHE key agreement) protects/derives a
#      random SYMMETRIC session key
#   2. Symmetric cipher (AES) encrypts the bulk data with that key
#   Exam cue: "combines the speed of symmetric with the key-distribution
#   convenience of asymmetric" → hybrid cryptosystem / session key
```

| Property | Symmetric | Asymmetric |
|---|---|---|
| Keys | 1 shared secret | 2 (public + private pair) |
| Speed | Fast — bulk data | Slow — small payloads only |
| Key distribution | Hard (must pre-share securely) | Easy (publish public key) |
| Keys for n users | n(n-1)/2 | n pairs (2n keys) |
| Confidentiality | Yes | Yes |
| Non-repudiation | No | Yes (digital signatures) |
| Example algorithms | AES, 3DES (deprecated), ChaCha20, Blowfish/Twofish | RSA, ECC, Diffie-Hellman, DSA/ECDSA, Ed25519 |
| Typical use | Disk/file/database encryption, TLS record layer | Key exchange, signatures, certificates |

## Algorithms and Key Lengths

| Algorithm | Type | Key lengths | Status / notes |
|---|---|---|---|
| AES (Advanced Encryption Standard) | Symmetric block (128-bit block) | 128 / 192 / 256 bits | Current standard; AES-256 for top secret; GCM mode adds authentication (AEAD) |
| 3DES (Triple DES) | Symmetric block | 112/168 effective | Deprecated (NIST disallowed after 2023) — legacy answer only |
| ChaCha20-Poly1305 | Symmetric stream + AEAD | 256 bits | Modern TLS alternative to AES-GCM, fast without AES hardware |
| RSA (Rivest–Shamir–Adleman) | Asymmetric | 2048 min, 3072/4096 stronger | 1024-bit is broken-adjacent — never acceptable; factoring-based |
| ECC (Elliptic Curve Cryptography) | Asymmetric | 256 bits ≈ RSA 3072 | Same strength with far smaller keys — mobile/IoT/constrained devices |
| DH (Diffie-Hellman) | Key exchange | 2048+ bit groups | Key AGREEMENT only — never encrypts data itself |
| ECDHE (Elliptic Curve DH Ephemeral) | Key exchange | 256-bit curves | Ephemeral = new key per session = perfect forward secrecy (PFS) |
| DSA / ECDSA / Ed25519 | Signature | 2048+ / 256 / 256 | Signing only; Ed25519 is the modern SSH/JWT favorite |

```
# Key-length guidance (NIST SP 800-57 comparable strengths):
#   Security level   Symmetric   RSA/DH      ECC        Hash
#   112-bit (legacy) 3DES        2048        224        SHA-224
#   128-bit          AES-128     3072        256        SHA-256
#   192-bit          AES-192     7680        384        SHA-384
#   256-bit          AES-256     15360       512/521    SHA-512
# Takeaway: doubling ECC key size tracks security level linearly;
#   RSA sizes balloon — that's WHY ECC wins on constrained devices.
#
# Key exchange concepts:
#   In-band  — key travels over the same channel as the data (risky
#              unless wrapped, e.g. TLS handshake)
#   Out-of-band — key shared via a different channel (phone, courier,
#              QR code) — answer for pre-shared key (PSK) scenarios
#   DH / ECDHE — both parties DERIVE the same shared secret without
#              ever transmitting it; ephemeral variants give
#              perfect forward secrecy: a stolen server private key
#              cannot decrypt PAST recorded sessions
#   Exam cue: "protect previously captured traffic even if the server
#   key is later compromised" → ephemeral keys / ECDHE / forward secrecy
#
# Longer key = more secure but slower; choose the shortest key length
# that meets the required security life of the data (crypto agility).
```

## Encryption Levels

```
# FULL-DISK ENCRYPTION (FDE)
#   Entire drive incl. OS; transparent after pre-boot auth
#   BitLocker (Windows, TPM-backed), FileVault (macOS), LUKS (Linux)
#   Self-encrypting drive (SED) — encryption in drive firmware/controller;
#     Opal is the TCG standard. Zero host CPU cost.
#   Protects: stolen/lost device at rest.  Does NOT protect: data in a
#     running, unlocked session (malware reads plaintext normally)
#   Exam cue: "laptop stolen from car — data unreadable" → FDE
#
# PARTITION encryption — one partition of a disk (e.g. LUKS on /home)
# VOLUME encryption    — a logical volume/container (VeraCrypt volume,
#   AWS EBS volume encryption); mounts as a drive when unlocked
# FILE-LEVEL encryption — individual files (EFS on NTFS, GPG a file);
#   survives copying the file elsewhere; per-user granularity
# DATABASE encryption  — TDE (Transparent Data Encryption) encrypts the
#   whole DB files at rest (SQL Server, Oracle);
#   column-level encrypts specific sensitive columns (SSNs, cards)
# RECORD-level encryption — individual rows/records encrypted with
#   possibly different keys (multi-tenant SaaS; per-customer keys)
#
# Granularity trade-off: full-disk = simplest, protects only powered-off;
#   file/record = finest control + protects data even when OS is up,
#   but more key management overhead
#
# TRANSPORT / COMMUNICATION encryption (data in transit):
#   TLS 1.2/1.3 — HTTPS, SMTPS, LDAPS...   (see tls sheet)
#   IPsec       — network-layer VPN tunnels (site-to-site-vpn sheet)
#   SSH         — secure remote shell/tunnels
#   WPA3        — wireless (SAE handshake)
# Data states: at rest (FDE/TDE), in transit (TLS/IPsec),
#   in use (secure enclave, memory encryption, homomorphic — emerging)
```

## Cryptographic Hardware and Key Management Tools

```
# TPM (Trusted Platform Module)
#   Chip ON the motherboard (or firmware-based fTPM); stores keys,
#   platform measurements (PCRs), endorsement key burned in at mfg
#   Uses: BitLocker key sealing, measured/secure boot attestation,
#   device identity. Per-DEVICE scope.
#   Exam cue: "encrypt laptop drives with hardware key storage on each
#   machine, keys released only if boot integrity intact" → TPM
#
# HSM (Hardware Security Module)
#   Dedicated NETWORK/appliance/PCIe device for an ORGANIZATION:
#   generates, stores, and uses keys without ever exposing them;
#   tamper-resistant (zeroizes on intrusion); FIPS 140-2/140-3 validated
#   Uses: CA root keys, TLS offload, payment/PIN processing, code signing
#   Exam cue: "centralized, tamper-proof key storage for the enterprise
#   CA / high-volume crypto operations" → HSM
#   TPM = one device's keys, soldered in; HSM = org-wide key vault
#   Cloud variants: AWS CloudHSM, Azure Dedicated HSM; microSD/USB
#   "micro HSMs" exist for field devices
#
# KMS (Key Management System)
#   Software/service layer for the key LIFECYCLE: generation, rotation,
#   distribution, expiration, revocation, destruction, audit
#   (AWS KMS, Azure Key Vault, HashiCorp Vault — see vault sheet)
#   Exam cue: "manage and rotate thousands of keys centrally with
#   audit logging" → key management system
#
# Secure enclave
#   Isolated co-processor/CPU region that protects keys and data IN USE
#   even from a compromised OS: Apple Secure Enclave, Intel SGX
#   (Software Guard Extensions), AMD SEV, Arm TrustZone (TEE — Trusted
#   Execution Environment)
#   Exam cue: "protect biometric/key material even if the OS kernel is
#   compromised", "data in use" → secure enclave / TEE
```

## Obfuscation: Steganography, Tokenization, Masking

```
# Obfuscation — making data unclear/hidden WITHOUT necessarily using
#   reversible-by-key encryption
#
# Steganography — HIDING data inside other data (image LSBs, audio,
#   video, whitespace in text, TCP header fields)
#   Security through obscurity — hides EXISTENCE of the message
#   Exam cue: "secret message embedded in an image posted publicly",
#   "data exfiltration hidden in cat pictures" → steganography
#
# Tokenization — replace sensitive value with a random TOKEN; the real
#   value lives only in a secure token vault; token has NO mathematical
#   relationship to the original (cannot be reversed without the vault)
#   Classic use: credit card numbers (PCI DSS scope reduction),
#   Apple Pay/Google Pay device account numbers
#   Exam cue: "reduce PCI DSS audit scope", "card number replaced by a
#   surrogate value usable only within the payment system" → tokenization
#
# Data masking — obscure PART of the data for display/test use:
#   ****-****-****-4821 ; substitute realistic fake values in dev DBs
#   (static masking = permanent copy; dynamic masking = on-the-fly per
#   query/viewer role)
#   Exam cue: "developers need production-shaped data without real PII",
#   "show only last 4 digits" → data masking
#
# Disambiguation trap:
#   Encryption   — reversible WITH key
#   Hashing      — one-way, never reversible
#   Tokenization — reversible only via vault lookup (no algorithm/key)
#   Masking      — partially hidden / substituted for display
#   Steganography— hidden inside a carrier file
```

## Hashing, Salting, HMAC, Digital Signatures

```
# Hash — one-way fixed-length digest; ANY input change avalanches the
#   output; used for integrity, password storage, signature input
#   Properties: deterministic, preimage-resistant (can't reverse),
#   collision-resistant (can't find two inputs → same digest)
#
# SHA-2 family (Secure Hash Algorithm): SHA-256 (32-byte digest),
#   SHA-384, SHA-512 — current workhorses
# SHA-3 (Keccak, sponge construction) — different internals; drop-in
#   alternative if SHA-2 ever falls
# MD5 (Message Digest 5, 128-bit) — BROKEN (collisions on demand):
#   never for signatures/passwords; forensically still seen for
#   non-adversarial file inventory alongside SHA-256
# SHA-1 — broken for collisions (SHAttered, 2017) — deprecated for
#   certs/signatures since 2017
#   Exam cue: "which algorithm should be REPLACED" → MD5 / SHA-1 / 3DES /
#   DES / WEP / SSL — the deprecated pile
#
# Salting — random per-password value concatenated before hashing:
#   defeats rainbow tables (precomputed hash lookups) and hides
#   duplicate passwords; salt is stored in plaintext next to the hash
#   (it's not a secret — its job is uniqueness)
#   Exam cue: "two users with the same password have different hashes",
#   "defend stolen hash DB against rainbow tables" → salting
#   Pepper — a SECRET site-wide value stored separately (HSM/env), added
#   to the hash input; unlike salt it is not stored with the hash
#
# Key stretching — deliberately slow/expensive hashing to make offline
#   brute force impractical:
#   PBKDF2 (Password-Based Key Derivation Function 2) — HMAC iterated
#     100k+ times; FIPS-friendly
#   bcrypt — Blowfish-based, built-in salt + cost factor
#   (scrypt / Argon2 — memory-hard, modern picks; Argon2 won the 2015
#    Password Hashing Competition)
#   Exam cue: "slow down password cracking", "derive an encryption key
#   from a password" → key stretching / PBKDF2 / bcrypt
#
# HMAC (Hash-based Message Authentication Code) — hash keyed with a
#   SHARED secret: proves integrity AND authenticity between two parties
#   (HMAC-SHA256 in API signing, JWTs, TLS record MACs)
#   No non-repudiation (both sides share the key)
#
# Digital signature = hash the message, then encrypt the DIGEST with the
#   sender's PRIVATE key; receiver decrypts with sender's public key and
#   compares digests
#   Provides: integrity + authentication + non-repudiation
#   Exam cue: "prove the message came from the sender and was not
#   altered, sender cannot deny it" → digital signature
#   Code signing = digital signature over software; driver/app trust
```

## Blockchain and Open Public Ledger

```
# Blockchain — distributed, append-only ledger; each block contains the
#   HASH of the previous block, so altering history invalidates every
#   subsequent block; consensus (proof of work/stake) replaces a
#   central authority
# Open public ledger — anyone can read/append per consensus rules
#   (permissionless: Bitcoin, Ethereum); contrast permissioned/private
#   ledgers (known participants, e.g. Hyperledger — supply chain,
#   inter-bank settlement)
# Security+ angle: integrity and availability without a trusted third
#   party; use cases = cryptocurrency, supply-chain provenance, smart
#   contracts, tamper-evident records
# Exam cue: "immutable distributed record verified by many nodes,
#   no central authority" → blockchain / open public ledger
```

## Certificates

```
# Digital certificate — X.509 structure binding a PUBLIC KEY to an
#   identity (subject), signed by an issuer
# CA (Certificate Authority) — trusted issuer; its own cert is either
#   self-signed (root) or signed by a parent (intermediate)
# Root of trust / chain of trust:
#   Root CA (offline, in an HSM, self-signed, long-lived)
#     → Intermediate/subordinate CA (does daily issuance; limits blast
#        radius if compromised)
#       → Leaf/end-entity certificate (your server)
#   Clients trust leaves because the chain terminates at a root in
#   their trust store
# RA (Registration Authority) — verifies requester identity for the CA
#   (does NOT sign certificates)
#
# CSR (Certificate Signing Request) — generated by the applicant:
#   contains the PUBLIC key + subject info, signed with the applicant's
#   private key; the PRIVATE KEY NEVER LEAVES the requester
#   Exam cue: "what do you send to the CA to get a certificate" → CSR
#
# Revocation:
#   CRL (Certificate Revocation List) — CA-published signed list of
#     revoked serial numbers; downloaded periodically → can be STALE,
#     large, slow
#   OCSP (Online Certificate Status Protocol) — client asks the CA's
#     OCSP responder about ONE cert in real time → fresher, but leaks
#     browsing to the CA and adds latency/availability dependency
#   OCSP stapling — the WEB SERVER fetches a time-stamped, CA-signed
#     OCSP response and staples it into the TLS handshake: fresh status,
#     no client→CA call, no privacy leak
#     Exam cue: "reduce client OCSP traffic / latency while proving
#     revocation status" → OCSP stapling
#   Reasons to revoke: key compromise (most urgent), CA compromise,
#     affiliation changed, superseded, cessation of operation
#
# Certificate pinning — app hard-codes the expected cert/public key and
#   rejects anything else, even if chain-valid (defeats rogue CAs /
#   corporate TLS interception); brittle at rotation time
# Certificate Transparency (CT) — public append-only logs of issued
#   certs; detects mis-issuance (required by browsers for public certs)
#
# Key fields on the exam: Subject, Issuer, Validity (notBefore/notAfter),
#   Subject Alternative Name (SAN), Key Usage / Extended Key Usage,
#   serial number, signature algorithm
# Common expiry gotcha: public TLS certs max 398 days (browser rule) —
#   "site suddenly shows warnings, nothing changed" → expired cert
```

## Certificate Types

| Type | What it covers | Exam trigger phrase |
|---|---|---|
| Self-signed | Signed by its own key — no third-party trust; free; browser warnings | "internal test system, users get trust warnings" |
| Third-party (CA-signed) | Signed by a public CA already in OS/browser trust stores | "publicly trusted, no warnings for customers" |
| Wildcard (`*.example.com`) | One host level of a single domain (`www`, `mail` — NOT `a.b.example.com`) | "many subdomains, one cert, minimize cost/management" |
| SAN / multi-domain (UCC) | Multiple explicit names, even different domains, in one cert | "example.com AND example.net AND mail.example.org on one cert" |
| Single-domain (DV — Domain Validation) | One FQDN; automated domain-control check only | "cheapest/fastest public cert" (Let's Encrypt/ACME) |
| OV / EV (Organization / Extended Validation) | CA vets the organization's legal identity (EV most rigorous) | "prove the legal entity behind the site" |
| Code-signing | Signs executables/drivers/scripts | "users warned software is from unknown publisher" |
| Machine/computer, user, email (S/MIME) | Device auth (802.1X), user auth (smart card/PIV), signed+encrypted mail | context names the subject type |
| Root / intermediate | CA certificates forming the chain | "keep the root offline; issue from intermediates" |

## Worked OpenSSL Examples

```bash
# Generate a private key + CSR (the thing you send to the CA)
openssl req -new -newkey rsa:2048 -nodes \
  -keyout server.key -out server.csr \
  -subj "/C=US/ST=TX/O=Example Corp/CN=www.example.com" \
  -addext "subjectAltName=DNS:www.example.com,DNS:example.com"
# ECC equivalent (smaller, faster):
openssl ecparam -name prime256v1 -genkey -noout -out server-ec.key
openssl req -new -key server-ec.key -out server-ec.csr

# Self-signed cert (internal/lab use — expect browser warnings)
openssl req -x509 -newkey rsa:2048 -nodes -days 365 \
  -keyout self.key -out self.crt -subj "/CN=lab.internal"

# Inspect: CSR, certificate, and a live server's cert + chain
openssl req  -in server.csr -noout -text
openssl x509 -in server.crt -noout -text            # full decode
openssl x509 -in server.crt -noout -subject -issuer -dates -ext subjectAltName
openssl s_client -connect example.com:443 -servername example.com \
  </dev/null 2>/dev/null | openssl x509 -noout -dates -issuer

# Verify a chain / match key↔cert (moduli must be identical)
openssl verify -CAfile chain.pem server.crt
openssl x509 -noout -modulus -in server.crt | openssl sha256
openssl rsa  -noout -modulus -in server.key | openssl sha256

# Check revocation via OCSP
openssl ocsp -issuer intermediate.pem -cert server.crt \
  -url http://ocsp.ca.example -resp_text

# Hashing and HMAC
sha256sum file.iso                      # GNU coreutils
openssl dgst -sha256 file.iso           # same digest, openssl syntax
openssl dgst -sha3-256 file.iso
openssl dgst -sha256 -hmac "sharedsecret" message.txt   # keyed HMAC
md5sum file.iso                         # legacy inventory only — MD5 is broken

# Symmetric file encryption with key stretching (PBKDF2 + AES-256)
openssl enc -aes-256-cbc -pbkdf2 -iter 600000 -salt \
  -in secret.txt -out secret.enc
openssl enc -d -aes-256-cbc -pbkdf2 -iter 600000 -in secret.enc

# Convert formats (PEM = Base64 text, DER = binary, PKCS#12 = key+cert bundle)
openssl x509 -in server.crt -outform der -out server.der
openssl pkcs12 -export -inkey server.key -in server.crt \
  -certfile chain.pem -out bundle.p12    # .p12/.pfx for Windows/import
```

## Exam Cues: Keyword → Answer

| Question keyword / scenario | Answer |
|---|---|
| "Encrypt bulk data quickly" | Symmetric (AES) |
| "Securely exchange a key over an untrusted network" | Diffie-Hellman / ECDHE |
| "Compromised server key must not decrypt recorded past sessions" | Perfect forward secrecy (ephemeral keys / ECDHE) |
| "Strong crypto on low-power IoT / mobile device" | ECC |
| "Prove sender identity + integrity, sender can't deny" | Digital signature |
| "Integrity + authenticity between two parties sharing a secret" | HMAC |
| "Same passwords produce different stored hashes" | Salting |
| "Slow down offline password cracking" | Key stretching (PBKDF2 / bcrypt) |
| "Recover encrypted data after employee leaves" | Key escrow / recovery agent |
| "Hardware key storage built into each laptop" | TPM |
| "Enterprise-grade tamper-resistant key appliance / CA keys" | HSM |
| "Centrally manage key lifecycle and rotation" | Key management system (KMS) |
| "Protect keys/data in use from a compromised OS" | Secure enclave / TEE |
| "Hide a message inside an image" | Steganography |
| "Replace card numbers to shrink PCI DSS scope" | Tokenization |
| "Show only last 4 digits / sanitized data for developers" | Data masking |
| "Stolen laptop, data unreadable" | Full-disk encryption |
| "Encrypt one sensitive DB column" | Column/field-level (database) encryption |
| "Immutable distributed ledger, no central authority" | Blockchain / open public ledger |
| "What is sent to the CA to obtain a cert" | CSR |
| "Real-time revocation check without client→CA traffic" | OCSP stapling |
| "Periodic downloaded list of revoked certs (may be stale)" | CRL |
| "One cert for all first-level subdomains" | Wildcard certificate |
| "One cert covering several distinct domain names" | SAN / multi-domain cert |
| "Internal-only cert triggers browser warnings" | Self-signed certificate |
| "Keep the most critical CA key offline" | Offline root CA (root of trust) |
| "App rejects valid-chain certs that aren't the expected one" | Certificate pinning |
| "MD5 / SHA-1 / DES / 3DES / SSL / WEP in an answer list" | The deprecated option — replace it |
| Distractor trap: "hashing encrypts data" | False — hashing is one-way, not encryption |
| Distractor trap: "HMAC gives non-repudiation" | False — shared key; only signatures do |
| Distractor trap: "DH encrypts the message" | False — DH only agrees on a key |

## See Also

security-plus-sy0-701, secplus-security-concepts, secplus-hardening, secplus-iam, pki, cryptography, crypto-protocols, tls, openssl, gpg, vault, zero-trust

## References

- [CompTIA Security+ SY0-701 Exam Objectives](https://www.comptia.org/certifications/security)
- [NIST SP 800-57 Part 1 Rev. 5 — Recommendation for Key Management](https://csrc.nist.gov/pubs/sp/800/57/pt1/r5/final)
- [NIST FIPS 197 — Advanced Encryption Standard (AES)](https://csrc.nist.gov/pubs/fips/197/final)
- [NIST FIPS 186-5 — Digital Signature Standard](https://csrc.nist.gov/pubs/fips/186/5/final)
- [NIST FIPS 202 — SHA-3 Standard](https://csrc.nist.gov/pubs/fips/202/final)
- [NIST SP 800-131A Rev. 2 — Transitioning Cryptographic Algorithms (3DES/SHA-1 deprecation)](https://csrc.nist.gov/pubs/sp/800/131/a/r2/final)
- [RFC 5280 — X.509 PKI Certificate and CRL Profile](https://www.rfc-editor.org/rfc/rfc5280)
- [RFC 6960 — Online Certificate Status Protocol (OCSP)](https://www.rfc-editor.org/rfc/rfc6960)
- [RFC 2104 — HMAC: Keyed-Hashing for Message Authentication](https://www.rfc-editor.org/rfc/rfc2104)
- [RFC 8018 — PKCS #5: PBKDF2](https://www.rfc-editor.org/rfc/rfc8018)
- [RFC 8446 — TLS 1.3](https://www.rfc-editor.org/rfc/rfc8446)
- [Trusted Computing Group — TPM 2.0 Specification](https://trustedcomputinggroup.org/resource/tpm-library-specification/)
- [OpenSSL Documentation](https://docs.openssl.org/)
