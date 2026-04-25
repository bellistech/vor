# OpenSSL (CLI Cookbook)

The swiss-army knife for TLS, certs, keys, signatures, and crypto on the command line — every subcommand, flag, format, and error you will hit in production.

## Setup

OpenSSL ships in three meaningful forks: OpenSSL itself (mainline 3.x — what you almost always want), LibreSSL (OpenBSD's leaner fork — what macOS actually ships under the `openssl` name), and BoringSSL (Google's internal-only fork — usually not exposed as a CLI). Read the version line before running anything that depends on a specific algorithm or flag.

```bash
openssl version
# OpenSSL 3.2.1 30 Jan 2024 (Library: OpenSSL 3.2.1 30 Jan 2024)

openssl version -a
# OpenSSL 3.2.1 30 Jan 2024
# built on: Tue Jan 30 12:00:00 2024 UTC
# platform: darwin64-arm64-cc
# options:  bn(64,64)
# compiler: clang ...
# OPENSSLDIR: "/opt/homebrew/etc/openssl@3"
# ENGINESDIR: "/opt/homebrew/lib/engines-3"
# MODULESDIR: "/opt/homebrew/lib/ossl-modules"
# Seeding source: os-specific
# CPUINFO: OPENSSL_armcap=0xbd
```

Anatomy of `openssl version -a`:

- Line 1 — version string (parsed by scripts; `openssl version | awk '{print $2}'`)
- `built on` — build date
- `platform` — target triple
- `options` — bignum word size (64,64 = 64-bit)
- `compiler` — compiler + flags (useful for repro builds)
- `OPENSSLDIR` — config directory; this is where `openssl.cnf` lives
- `ENGINESDIR` — old engine plugins (1.x model, still loadable in 3.x)
- `MODULESDIR` — provider plugins (3.x model — fips, legacy, default)
- `Seeding source` — RNG seed origin
- `CPUINFO` — runtime CPU feature flags

### OpenSSL 1.1 vs 3.x — The Major-Bump Differences

The 1.1.x series ended security support on 11 September 2023. 3.0 is the current LTS. Migration pain points:

- **Provider model.** 1.x had `engines` (ENGINE API). 3.x has `providers` (loaded via config). Legacy algorithms (MD2, MD4, RC2, RC4, RC5, DES, Blowfish, CAST, IDEA, SEED, RIPEMD-160, Whirlpool) live in the **legacy** provider — disabled by default. Add `-provider legacy -provider default` (in that order).
- **Default key size for `genrsa` / `req -newkey rsa`.** 1.1.x default 2048; 3.x still 2048 — but 3.x raises the minimum security level to 112 bits which forbids RSA<2048, SHA-1 signatures on certs, and DH<2048.
- **Removed.** The `rsautl` command is deprecated → use `pkeyutl`. The legacy `dhparam`/`ecparam` interfaces are deprecated → use `genpkey -genparam` / `genpkey`.
- **Reduced trust.** TLS 1.0 and 1.1 are disabled at the default `SECLEVEL=2`.
- **Different error format.** 1.x: `error:14094410:SSL routines:ssl3_read_bytes:sslv3 alert handshake failure`. 3.x: `error:0A000410:SSL routines::sslv3 alert handshake failure`.
- **PKCS12.** 3.x writes PKCS12 with AES + PBKDF2 by default. 1.1 wrote with PBE-SHA1-3DES. Bundles created in 3.x without `-legacy` cannot be read by Java 8 / Windows XP / old `keytool`. Bundles created in 1.x cannot be read by 3.x without `-provider legacy` plus the `-legacy` flag on `pkcs12 -in`.
- **`SHA1` is gone from default signature algorithms.** Self-signing or signing CSRs without `-sha256` (etc.) produces an error in 3.x where 1.x silently used SHA-256.

### FIPS Provider in 3.x

```bash
openssl list -providers
# Providers:
#   default
#     name: OpenSSL Default Provider
#     version: 3.2.1
#     status: active

# To enable FIPS:
openssl fipsinstall -out /etc/ssl/fipsmodule.cnf -module /usr/lib/ossl-modules/fips.so
# Then activate in openssl.cnf:
#   [provider_sect]
#   fips = fips_sect
#   default = default_sect
#   [fips_sect]
#   activate = 1
```

When the FIPS provider is active, all non-FIPS-approved algorithms (MD5, RC4, DES, etc.) raise `error:1C8000D5:Provider routines::missing get params`.

### LibreSSL / BoringSSL Alternatives

- **LibreSSL.** OpenBSD fork after Heartbleed (2014). Fewer features, smaller surface area. Default on macOS, OpenBSD. Lacks: FIPS provider, providers in general, `openssl ts`, some OCSP options. Compatible with most cert/key workflows. `openssl version` reports `LibreSSL 3.x.x`.
- **BoringSSL.** Google's fork. No stable API, no CLI. You won't run it from the shell — only via curl-impersonate or other consumers.

### brew vs system openssl on macOS

macOS ships **LibreSSL** under `/usr/bin/openssl`, despite the name:

```bash
/usr/bin/openssl version
# LibreSSL 3.3.6
```

Install real OpenSSL via brew:

```bash
brew install openssl@3
# /opt/homebrew/opt/openssl@3/bin/openssl   (Apple Silicon)
# /usr/local/opt/openssl@3/bin/openssl      (Intel)
```

Make it the default in your shell:

```bash
export PATH="/opt/homebrew/opt/openssl@3/bin:$PATH"
# or for bash compatibility:
export PATH="$(brew --prefix openssl@3)/bin:$PATH"
```

Most macOS bugs in cert workflows trace back to LibreSSL silently being used instead of OpenSSL 3. **Always `which openssl` and `openssl version` before debugging.**

## Top-Level Subcommands

```bash
openssl list -commands
# asn1parse ca ciphers cmp cms crl crl2pkcs7 dgst dhparam dsa dsaparam ec ecparam
# enc engine errstr fipsinstall gendsa genpkey genrsa help info kdf list mac
# nseq ocsp passwd pkcs12 pkcs7 pkcs8 pkey pkeyparam pkeyutl prime rand rehash
# req rsa rsautl s_client s_server s_time sess_id smime speed spkac srp storeutl
# ts verify version x509

openssl list -cipher-algorithms
# Lists every symmetric cipher available (AES-256-GCM, AES-256-CBC, ChaCha20-Poly1305, ...)

openssl list -digest-algorithms
# Lists every hash (SHA256, SHA384, SHA512, SHA3-256, BLAKE2b512, ...)

openssl list -public-key-algorithms
# RSA, RSA-PSS, DSA, EC, X25519, X448, ED25519, ED448, DH, DHX, ...

openssl list -mac-algorithms
# HMAC, CMAC, GMAC, KMAC128, KMAC256, ...

openssl list -kdf-algorithms
# HKDF, PBKDF1, PBKDF2, SCRYPT, SSHKDF, SSKDF, TLS1-PRF, X942KDF-ASN1, X963KDF, ...

openssl list -providers
openssl list -signature-algorithms
openssl list -key-managers
openssl list -store-loaders
```

`openssl version` flag variations:

```bash
openssl version           # short
openssl version -v        # same as no flag (just version line)
openssl version -b        # build date
openssl version -p        # platform
openssl version -d        # OPENSSLDIR (config root)
openssl version -e        # ENGINESDIR
openssl version -m        # MODULESDIR (3.x providers)
openssl version -f        # compile flags
openssl version -o        # options (bignum size)
openssl version -r        # seed source
openssl version -c        # CPU info
openssl version -a        # everything
```

`openssl errstr` decodes hex error codes:

```bash
openssl errstr 0A0000B8
# error:0A0000B8:SSL routines::no shared cipher
```

## Generating Private Keys — RSA

Modern (3.x):

```bash
openssl genpkey -algorithm RSA \
  -pkeyopt rsa_keygen_bits:4096 \
  -pkeyopt rsa_keygen_pubexp:65537 \
  -out key.pem

# Encrypted with AES-256:
openssl genpkey -algorithm RSA \
  -pkeyopt rsa_keygen_bits:4096 \
  -aes-256-cbc \
  -out key.pem
# (prompts for passphrase)

# Pass via file/env/literal (NEVER use pass: in scripts; visible in ps):
openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:4096 \
  -aes-256-cbc -pass file:/run/secrets/keypass \
  -out key.pem

openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:4096 \
  -aes-256-cbc -pass env:KEYPASS \
  -out key.pem

openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:4096 \
  -aes-256-cbc -pass pass:hunter2 \
  -out key.pem    # bad: ps shows "pass:hunter2"
```

Legacy (still works, easier to type, deprecated):

```bash
openssl genrsa -aes256 -out key.pem 4096
openssl genrsa -out key.pem 4096                    # unencrypted
openssl genrsa -aes256 -passout pass:hunter2 -out key.pem 4096
```

PEM vs DER output:

```bash
# PEM (default — base64 with -----BEGIN PRIVATE KEY-----)
openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:4096 -out key.pem

# DER (raw binary)
openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:4096 \
  -outform DER -out key.der
```

Guidance: **3072-bit RSA** is the modern default for new keys. **4096** is the paranoid choice (slower; rarely beats 3072 cryptographically). 2048 is the practical minimum (legacy compat).

## Generating Private Keys — EC

```bash
# P-256 (the most common; alias prime256v1; same as secp256r1)
openssl genpkey -algorithm EC \
  -pkeyopt ec_paramgen_curve:P-256 \
  -out key.pem

# P-384 (for higher security level)
openssl genpkey -algorithm EC \
  -pkeyopt ec_paramgen_curve:P-384 \
  -out key.pem

# P-521 (paranoid; uncommon — many TLS stacks lack it)
openssl genpkey -algorithm EC \
  -pkeyopt ec_paramgen_curve:P-521 \
  -out key.pem

# secp256k1 (Bitcoin / Ethereum — NOT for TLS)
openssl genpkey -algorithm EC \
  -pkeyopt ec_paramgen_curve:secp256k1 \
  -out key.pem
```

Curve aliases (this trips up everyone — they are the same curve):

| OpenSSL name | NIST | SEC |
|---|---|---|
| `prime256v1` | `P-256` | `secp256r1` |
| `secp384r1` | `P-384` | `secp384r1` |
| `secp521r1` | `P-521` | `secp521r1` |
| `secp256k1` | (none) | `secp256k1` |

X25519 (key exchange only — not TLS server identity):

```bash
openssl genpkey -algorithm X25519 -out x25519.pem
openssl genpkey -algorithm X448  -out x448.pem
```

ed25519 / ed448 (signing — see next section):

```bash
openssl genpkey -algorithm ED25519 -out ed25519.pem
openssl genpkey -algorithm ED448   -out ed448.pem
```

Legacy `ecparam` route:

```bash
openssl ecparam -list_curves           # show all supported names
openssl ecparam -name prime256v1 -genkey -noout -out key.pem
openssl ecparam -name secp384r1 -genkey -noout -out key.pem
```

`-pkeyopt ec_param_enc:named_curve` is the default and you want it (instead of explicit-parameter encoding which produces non-interoperable keys).

## Generating Private Keys — Ed25519 / X25519

Ed25519 is the **modern default for new signing keys**. Tiny (32-byte private), fast (faster than RSA, comparable to ECDSA), deterministic (no RNG-failure CVE class), no curve-choice trap.

```bash
openssl genpkey -algorithm ED25519 -out signing.pem

# Public:
openssl pkey -in signing.pem -pubout -out signing.pub

# Inspect:
openssl pkey -in signing.pem -text -noout
# ED25519 Private-Key:
# priv:
#     <32 bytes hex>
# pub:
#     <32 bytes hex>
```

Ed25519 caveats for TLS use: only TLS 1.3 supports Ed25519 in the certificate signature algorithm (RFC 8446); browsers vary in support. For SSH it is universal and superior. For software signing (cosign, age, signify, minisign) it is the canonical choice.

X25519 vs Ed25519 vs Curve25519:

- **Curve25519** — the underlying elliptic curve.
- **X25519** — Diffie-Hellman key exchange on Curve25519 (RFC 7748).
- **Ed25519** — EdDSA signatures on the twisted-Edwards form of Curve25519 (RFC 8032).

You generate them as separate keys; do not reuse one for the other.

## Inspecting Private Keys

Modern unified command:

```bash
openssl pkey -in key.pem -text -noout
# Private-Key: (4096 bit, 2 primes)
# modulus:
#     00:c4:8d:7e:...
# publicExponent: 65537 (0x10001)
# privateExponent: ...
# prime1: ...
# prime2: ...
# exponent1: ...
# exponent2: ...
# coefficient: ...
```

Legacy algorithm-specific commands (still work):

```bash
openssl rsa -in key.pem -text -noout         # RSA only
openssl ec  -in key.pem -text -noout         # EC only
openssl dsa -in key.pem -text -noout         # DSA only
```

Consistency check (does the math add up?):

```bash
openssl pkey -in key.pem -check -noout
# RSA key ok

openssl rsa -in key.pem -check -noout
# (same, RSA-only)
```

Strip the passphrase (or change it):

```bash
# Remove passphrase
openssl pkey -in key.pem -out unencrypted.pem
# (prompts for the old passphrase, writes plaintext)

# Add/change to AES-256
openssl pkey -in key.pem -aes-256-cbc -out new.pem

# Non-interactive
openssl pkey -in key.pem -passin pass:old -aes-256-cbc -passout pass:new -out new.pem
```

Extract public key:

```bash
openssl pkey -in key.pem -pubout -out pub.pem
openssl rsa  -in key.pem -pubout -out pub.pem    # RSA-specific legacy
```

Get just the SHA-256 fingerprint of the public key (handy for SSHFP / pinning):

```bash
openssl pkey -in key.pem -pubout -outform DER | openssl dgst -sha256
```

## Public Key Operations

```bash
openssl pkey -in pub.pem -pubin -text -noout
# Public-Key: (4096 bit)
# Modulus: ...
# Exponent: 65537 (0x10001)
```

Convert public-key formats:

```bash
# PEM -> DER
openssl pkey -in pub.pem -pubin -outform DER -out pub.der

# DER -> PEM
openssl pkey -in pub.der -pubin -inform DER -out pub.pem

# Extract raw modulus (RSA)
openssl rsa -in pub.pem -pubin -modulus -noout
```

PKCS#8 vs SubjectPublicKeyInfo:

- Private keys today are **PKCS#8** (`-----BEGIN PRIVATE KEY-----` or `-----BEGIN ENCRYPTED PRIVATE KEY-----`). PKCS#1/SEC1 forms (`-----BEGIN RSA PRIVATE KEY-----`, `-----BEGIN EC PRIVATE KEY-----`) are legacy.
- Public keys are **SubjectPublicKeyInfo** (`-----BEGIN PUBLIC KEY-----`). RSA-specific PKCS#1 (`-----BEGIN RSA PUBLIC KEY-----`) is rare and usually a mistake.

Encode an ed25519 public key as raw 32 bytes (e.g. for libsodium interop):

```bash
openssl pkey -in ed25519.pem -pubout -outform DER | tail -c 32 | xxd -p
```

## CSR — Basic

A Certificate Signing Request (CSR) is the file you send to a CA to get a certificate. It contains your public key plus the identity (DN + SAN extensions) signed by your private key.

```bash
openssl req -new -key key.pem -out req.csr \
  -subj "/C=US/ST=CA/L=SF/O=Acme/CN=example.com"

# Non-interactive (no prompts even if config asks):
openssl req -new -key key.pem -out req.csr \
  -subj "/CN=example.com" -batch

# Generate key + CSR in one command (RSA-4096, no passphrase):
openssl req -new -newkey rsa:4096 -nodes \
  -keyout key.pem -out req.csr \
  -subj "/CN=example.com"

# Same with EC:
openssl req -new -newkey ec:<(openssl ecparam -name prime256v1) -nodes \
  -keyout key.pem -out req.csr \
  -subj "/CN=example.com"

# Or 3.x style for EC:
openssl req -new -newkey EC -pkeyopt ec_paramgen_curve:P-256 -nodes \
  -keyout key.pem -out req.csr \
  -subj "/CN=example.com"
```

Add SAN with `-addext`:

```bash
openssl req -new -key key.pem -out req.csr \
  -subj "/CN=example.com" \
  -addext "subjectAltName=DNS:example.com,DNS:www.example.com,IP:1.2.3.4"
```

`-nodes` means "no DES" — the key file is written unencrypted. Mandatory in containers, automation, and any non-interactive context.

`-batch` skips the interactive prompt for any field not specified by `-subj`.

Distinguished Name field abbreviations (per RFC 4519 / RFC 5280):

| Code | Field | Example |
|---|---|---|
| `C` | Country (2 letters) | `US` |
| `ST` | State / Province | `California` |
| `L` | Locality (city) | `San Francisco` |
| `O` | Organization | `Acme Inc` |
| `OU` | Organizational Unit | `IT` |
| `CN` | Common Name | `example.com` |
| `emailAddress` | Email | `admin@example.com` |
| `serialNumber` | Serial | `1234` |
| `street` | Street address | `123 Main St` |
| `postalCode` | ZIP | `94102` |

Browsers and modern PKI **ignore CN** — see the SAN section.

## CSR Inspection

```bash
openssl req -in req.csr -text -noout -verify
# verify OK
# Certificate Request:
#     Data:
#         Version: 1 (0x0)
#         Subject: CN = example.com
#         Subject Public Key Info:
#             Public Key Algorithm: rsaEncryption
#                 Public-Key: (4096 bit)
#                 ...
#         Requested Extensions:
#             X509v3 Subject Alternative Name:
#                 DNS:example.com, DNS:www.example.com
#     Signature Algorithm: sha256WithRSAEncryption
```

The `verify OK` line confirms the private key still owns the request (i.e. the embedded signature is valid against the embedded public key).

`-verify` failure looks like:

```bash
verify failure
4007D7BB957F0000:error:02000068:rsa routines::bad signature:...
```

…which means the CSR was tampered with after signing.

Inspect just the SAN:

```bash
openssl req -in req.csr -noout -text | grep -A1 "Subject Alternative"
```

Inspect just subject:

```bash
openssl req -in req.csr -noout -subject -nameopt sep_multiline,utf8
# subject=
#     C  = US
#     ST = California
#     L  = San Francisco
#     O  = Acme
#     CN = example.com
```

Convert CSR PEM ↔ DER:

```bash
openssl req -in req.csr -outform DER -out req.der
openssl req -in req.der -inform DER -out req.csr
```

## CSR with Config File

The `-addext` flag handles simple cases. For multi-SAN production CSRs, use a config file — it is the canonical workflow.

`csr.cnf`:

```ini
[req]
default_bits        = 4096
default_md          = sha256
prompt              = no
distinguished_name  = req_distinguished_name
req_extensions      = v3_req

[req_distinguished_name]
C  = US
ST = California
L  = San Francisco
O  = Acme Inc
OU = Engineering
CN = example.com
emailAddress = sslteam@example.com

[v3_req]
basicConstraints     = CA:FALSE
keyUsage             = digitalSignature, keyEncipherment
extendedKeyUsage     = serverAuth, clientAuth
subjectAltName       = @alt_names

[alt_names]
DNS.1 = example.com
DNS.2 = www.example.com
DNS.3 = api.example.com
DNS.4 = *.example.com
IP.1  = 192.0.2.10
IP.2  = 2001:db8::10
email.1 = admin@example.com
URI.1   = https://example.com/
```

Then:

```bash
openssl req -new -key key.pem -config csr.cnf -out req.csr

# All-in-one:
openssl req -new -newkey rsa:4096 -nodes -keyout key.pem \
  -config csr.cnf -out req.csr
```

Verify the SANs landed:

```bash
openssl req -in req.csr -noout -text | grep -A 20 "Requested Extensions"
```

`prompt = no` is the magic line that makes the DN come from the config file instead of asking interactively. Without it, OpenSSL prompts for every field.

## Self-Signed Certificate

```bash
# Two-step (existing key + CSR):
openssl req -x509 -days 365 -key key.pem -in req.csr -out cert.pem
```

One-shot (key + cert in one call, no CSR file):

```bash
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem \
  -days 365 -nodes \
  -subj "/CN=localhost" \
  -addext "subjectAltName=DNS:localhost,DNS:*.localhost,IP:127.0.0.1,IP:::1"

# EC version:
openssl req -x509 -newkey EC -pkeyopt ec_paramgen_curve:P-256 \
  -keyout key.pem -out cert.pem \
  -days 365 -nodes \
  -subj "/CN=localhost" \
  -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"

# Ed25519 self-signed (TLS 1.3 only):
openssl req -x509 -newkey ED25519 \
  -keyout key.pem -out cert.pem \
  -days 365 -nodes \
  -subj "/CN=localhost" \
  -addext "subjectAltName=DNS:localhost"
```

Public-CA-issued certs cap out at **397 days**. For private/self-signed, longer is fine — `-days 3650` for a 10-year dev CA is common.

Re-sign an existing self-signed cert (preserve key, refresh dates):

```bash
openssl x509 -in old-cert.pem -signkey key.pem -days 365 -out new-cert.pem
# But -signkey on x509 won't preserve extensions; for SAN you must regenerate via req -x509.
```

## CA Workflow

For more than one cert, set up a tiny CA. The `openssl ca` command expects a directory layout per `openssl.cnf`. The default config references `./demoCA/` but you should override.

Layout:

```
ca/
  openssl.cnf
  certs/
  crl/
  newcerts/        # one PEM per signed cert, named by serial
  private/
    ca.key.pem     # mode 0600
  index.txt        # empty file
  serial           # one-line file containing "1000"
  crlnumber        # one-line file containing "1000"
```

Bootstrap:

```bash
mkdir -p ca/certs ca/crl ca/newcerts ca/private
chmod 700 ca/private
touch ca/index.txt
echo 1000 > ca/serial
echo 1000 > ca/crlnumber
```

`ca/openssl.cnf` (the relevant sections):

```ini
[ca]
default_ca = CA_default

[CA_default]
dir              = ./ca
certs            = $dir/certs
crl_dir          = $dir/crl
new_certs_dir    = $dir/newcerts
database         = $dir/index.txt
serial           = $dir/serial
crlnumber        = $dir/crlnumber
private_key      = $dir/private/ca.key.pem
certificate      = $dir/certs/ca.cert.pem
default_md       = sha256
default_days     = 825
default_crl_days = 30
policy           = policy_strict
email_in_dn      = no
unique_subject   = no
copy_extensions  = copy

[policy_strict]
countryName             = match
stateOrProvinceName     = match
organizationName        = match
organizationalUnitName  = optional
commonName              = supplied
emailAddress            = optional

[v3_intermediate_ca]
subjectKeyIdentifier   = hash
authorityKeyIdentifier = keyid:always,issuer
basicConstraints       = critical, CA:true, pathlen:0
keyUsage               = critical, digitalSignature, cRLSign, keyCertSign

[server_cert]
basicConstraints       = CA:FALSE
nsCertType             = server
keyUsage               = critical, digitalSignature, keyEncipherment
extendedKeyUsage       = serverAuth
subjectKeyIdentifier   = hash
authorityKeyIdentifier = keyid,issuer:always
```

Generate the root CA key + cert:

```bash
openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:4096 \
  -aes-256-cbc -out ca/private/ca.key.pem
chmod 400 ca/private/ca.key.pem

openssl req -config ca/openssl.cnf -key ca/private/ca.key.pem \
  -new -x509 -days 7300 -sha256 -extensions v3_intermediate_ca \
  -out ca/certs/ca.cert.pem \
  -subj "/C=US/ST=CA/O=Acme Root CA/CN=Acme Root CA X1"
```

Sign a CSR:

```bash
openssl ca -config ca/openssl.cnf \
  -extensions server_cert \
  -days 825 -notext -md sha256 \
  -in req.csr \
  -out ca/certs/example.com.cert.pem
```

The `openssl ca` command refuses to sign a request for the same CN twice unless `unique_subject = no`. It records every issuance in `index.txt` (V = valid, R = revoked, E = expired).

Revoke a cert + regenerate CRL:

```bash
openssl ca -config ca/openssl.cnf -revoke ca/certs/example.com.cert.pem
openssl ca -config ca/openssl.cnf -gencrl -out ca/crl/ca.crl.pem
```

Intermediate CA pattern:

1. Root CA signs an intermediate CA cert (with `pathlen:0` if no further CA below).
2. Intermediate CA signs leaf certs.
3. Servers serve `leaf + intermediate` (chain). Clients trust `root` only.

For most teams: use **smallstep** (`step-ca`), **EJBCA**, or **HashiCorp Vault PKI** instead — `openssl ca` is fine for one-off and learning, painful in production.

## Certificate Inspection (`openssl x509`)

The single most useful subcommand. Every flag that matters:

```bash
openssl x509 -in cert.pem -text -noout
# Full human-readable dump.

openssl x509 -in cert.pem -noout -subject
# subject=CN = example.com

openssl x509 -in cert.pem -noout -issuer
# issuer=C = US, O = Let's Encrypt, CN = R3

openssl x509 -in cert.pem -noout -dates
# notBefore=Jan  1 00:00:00 2024 GMT
# notAfter=Apr  1 00:00:00 2024 GMT

openssl x509 -in cert.pem -noout -startdate
# notBefore=Jan  1 00:00:00 2024 GMT

openssl x509 -in cert.pem -noout -enddate
# notAfter=Apr  1 00:00:00 2024 GMT

openssl x509 -in cert.pem -noout -fingerprint
# SHA1 Fingerprint=AB:CD:EF:...
# (sha1 is default — modern usage wants -sha256)

openssl x509 -in cert.pem -noout -fingerprint -sha256
# sha256 Fingerprint=12:34:...

openssl x509 -in cert.pem -noout -fingerprint -sha384

openssl x509 -in cert.pem -noout -serial
# serial=03B5C5E3FFC421...

openssl x509 -in cert.pem -noout -issuer_hash -subject_hash
# 4f06f81d
# 9d66eb8d

# (Used for CApath rehashing — `openssl rehash dir/` creates symlinks named <hash>.0)
```

Certificate validity check (cron-friendly):

```bash
openssl x509 -in cert.pem -checkend 0
# Certificate will not expire        (exit 0)
# OR
# Certificate will expire             (exit 1)

openssl x509 -in cert.pem -checkend 2592000     # 30 days
# Certificate will not expire   (exit 0 means OK for 30 more days)
```

Show purpose:

```bash
openssl x509 -in cert.pem -noout -purpose
# Certificate purposes:
# SSL client : Yes
# SSL client CA : No
# SSL server : Yes
# SSL server CA : No
# Netscape SSL server : Yes
# S/MIME signing : No
# ...
```

Extract public key from a cert:

```bash
openssl x509 -in cert.pem -noout -pubkey > pub.pem
openssl x509 -in cert.pem -noout -pubkey -outform DER > pub.der
```

Match a key to a cert (RSA classic):

```bash
diff <(openssl rsa -in key.pem -modulus -noout) \
     <(openssl x509 -in cert.pem -modulus -noout)
# (no output = match)

# Or via hash:
openssl rsa -in key.pem -modulus -noout | openssl md5
openssl x509 -in cert.pem -modulus -noout | openssl md5
# (compare the two MD5s)
```

EC equivalent (modulus doesn't apply — compare public-point hashes):

```bash
diff <(openssl pkey -in key.pem  -pubout) \
     <(openssl x509 -in cert.pem -pubkey -noout)
```

Inspect a single extension:

```bash
openssl x509 -in cert.pem -noout -ext subjectAltName
# X509v3 Subject Alternative Name:
#     DNS:example.com, DNS:www.example.com

openssl x509 -in cert.pem -noout -ext keyUsage,extendedKeyUsage,basicConstraints
```

Format conversion:

```bash
openssl x509 -in cert.pem -outform DER -out cert.der
openssl x509 -in cert.der -inform DER -out cert.pem
```

`-nameopt`:

```bash
openssl x509 -in cert.pem -noout -subject -nameopt RFC2253
# subject=CN=example.com,O=Acme,L=SF,ST=California,C=US

openssl x509 -in cert.pem -noout -subject -nameopt multiline
# subject=
#     countryName               = US
#     stateOrProvinceName       = California
#     ...

openssl x509 -in cert.pem -noout -subject -nameopt utf8,sep_comma_plus_space,esc_2253
```

## SubjectAlternativeName (SAN) — Modern Requirement

Browsers (Chrome/Edge since 58, Firefox since 48, Safari since 2019) and most modern TLS validators **ignore the CN field for hostname matching**. SAN is mandatory.

SAN value types:

- `DNS:hostname` — most common; supports wildcards (`DNS:*.example.com` matches one label only)
- `IP:1.2.3.4` — for IP-pinned certs (rare in public PKI; common internally)
- `IP:2001:db8::1` — IPv6
- `email:user@example.com` — for S/MIME
- `URI:https://example.com/` — rare
- `otherName:1.3.6.1.4.1.311.20.2.3;UTF8:upn@example.com` — UPN for smartcards
- `RID:1.2.3.4.5` — Registered ID

Quick add via `-addext`:

```bash
-addext "subjectAltName=DNS:a.example.com,DNS:b.example.com,IP:10.0.0.1,IP:::1,email:admin@example.com,URI:https://example.com/"
```

Multi-SAN config file pattern (covered in CSR Config File section above) is more readable for >3 SANs.

Wildcard caveats: `*.example.com` covers `foo.example.com` but NOT `example.com` itself or `foo.bar.example.com`. To cover both apex and one-level subdomains: include both `DNS:example.com` and `DNS:*.example.com`.

## `openssl s_client` — The TLS Inspection Multitool

Every flag that matters:

```bash
openssl s_client -connect example.com:443 -servername example.com
# Note: -servername sends SNI. Without it, you may get the default vhost cert.
```

Connection target:

```bash
-connect host:port           # canonical
-host host -port port        # legacy split
-unix /path/to/sock          # AF_UNIX
-4 / -6                      # force IPv4 / IPv6
```

SNI — **CRITICAL**:

```bash
-servername host             # Send SNI extension. Without this, virtual-hosted servers
                             # serve the default cert which is rarely what you want.
-noservername                # Disable SNI (testing default-vhost behavior)
```

TLS version:

```bash
-tls1_3                      # only TLS 1.3
-tls1_2                      # only TLS 1.2
-tls1_1                      # only TLS 1.1 (often disabled at SECLEVEL=2)
-tls1                        # only TLS 1.0
-ssl3                        # only SSLv3 (only if compiled with enable-ssl3)
-no_tls1_3 / -no_tls1_2      # exclude
-no_tls1_1 / -no_tls1
-min_protocol TLSv1.2
-max_protocol TLSv1.3
```

Cipher control:

```bash
-cipher LIST                 # TLS 1.2 cipher allowlist (OpenSSL syntax)
                             # e.g. -cipher 'ECDHE+AES256+GCM:!SHA1'
-ciphersuites LIST           # TLS 1.3 cipher allowlist (RFC name list)
                             # e.g. -ciphersuites TLS_AES_256_GCM_SHA384
-curves LIST                 # ECDHE curve allowlist (X25519:P-256:P-384)
-sigalgs LIST                # signature algorithms (RSA+SHA256:ECDSA+SHA384:ed25519)
```

Chain / verification:

```bash
-showcerts                   # print the full chain server presents
-verify N                    # verification depth (default 9)
-verify_return_error         # exit non-zero if cert verification fails
-CAfile path/to/bundle.pem   # trust roots
-CApath dir/                 # rehashed dir of trust roots
-no-CAfile -no-CApath        # disable defaults (useful for testing)
-trusted_first               # check trusted certs first (3.x default)
-partial_chain               # accept partial chain
```

Client certificate (mTLS):

```bash
-cert client.pem
-key client.key
-keyform PEM|DER
-certform PEM|DER
-pass pass:secret            # if client key encrypted
-cert_chain chain.pem        # send extra intermediates with client cert
```

STARTTLS — opportunistic upgrade for plaintext-then-TLS protocols:

```bash
-starttls smtp               # SMTP submission (587, 25)
-starttls imap               # IMAP (143)
-starttls ftp                # FTPS via AUTH TLS (21)
-starttls pop3               # POP3 (110)
-starttls xmpp               # XMPP client-to-server (5222)
-starttls xmpp-server        # XMPP server-to-server (5269)
-starttls irc                # IRC (6667)
-starttls sieve              # ManageSieve (4190)
-starttls postgres           # PostgreSQL (5432)
-starttls mysql              # MySQL (3306) — 3.x only
-starttls nntp               # NNTP (119)
-starttls ldap               # LDAP (389)
-starttls lmtp               # LMTP (24)
-name domain                 # SMTP EHLO name etc.
```

ALPN / NPN:

```bash
-alpn h2,http/1.1            # advertise these ALPN protocols
-nextprotoneg h2,http/1.1    # legacy NPN (deprecated, but still in old code)
```

OCSP:

```bash
-status                      # request OCSP stapling response
                             # Look for "OCSP response: no response sent" or full details
```

Debugging:

```bash
-msg                         # print TLS handshake messages
-debug                       # print TLS bytes (very verbose)
-trace                       # full protocol trace (3.x; verbose)
-state                       # print SSL state transitions
-tlsextdebug                 # parse + print TLS extensions
-security_debug              # security-level debug info
-keylogfile path             # write SSLKEYLOG for Wireshark TLS decryption
```

Session resumption:

```bash
-reconnect                   # connect 5 times (resume on connections 2-5)
-sess_in file                # load session and resume
-sess_out file               # save session
-no_ticket                   # disable session tickets (force session ID)
```

Behavior:

```bash
-prexit                      # print session info on exit
-ign_eof                     # don't close on stdin EOF (servers that need keepalive)
-quiet                       # silent except data (good for scripts)
-crlf                        # translate stdin LF -> CRLF (for SMTP/HTTP testing)
-no_ign_eof
```

PSK:

```bash
-psk hex
-psk_identity name
-psk_session file
```

Verification (3.x):

```bash
-verify_hostname host
-verify_email addr
-verify_ip a.b.c.d
```

DANE:

```bash
-dane_tlsa_domain example.com
-dane_tlsa_rrdata "3 1 1 ABCDEF..."
```

## s_client Common Recipes

Quick cert dump with chain:

```bash
echo | openssl s_client -connect example.com:443 -servername example.com -showcerts 2>/dev/null
```

Just dates of the leaf cert:

```bash
echo | openssl s_client -connect example.com:443 -servername example.com 2>/dev/null \
  | openssl x509 -noout -dates -subject
```

Just expiry epoch (parseable):

```bash
echo | openssl s_client -connect example.com:443 -servername example.com 2>/dev/null \
  | openssl x509 -noout -enddate \
  | cut -d= -f2 \
  | xargs -I{} date -d "{}" +%s    # GNU date
```

Days until expiry (BSD/macOS):

```bash
end=$(echo | openssl s_client -connect example.com:443 -servername example.com 2>/dev/null \
        | openssl x509 -noout -enddate | cut -d= -f2)
echo "$(( ( $(date -j -f "%b %d %T %Y %Z" "$end" +%s) - $(date +%s) ) / 86400 )) days left"
```

Test what TLS versions a server supports:

```bash
for v in tls1 tls1_1 tls1_2 tls1_3; do
  echo -n "$v: "
  echo | openssl s_client -connect example.com:443 -servername example.com -$v 2>/dev/null \
    | grep -E "^(SSL handshake|Protocol)" | head -n1
done
```

Test ALPN negotiation:

```bash
echo | openssl s_client -connect example.com:443 -servername example.com -alpn h2,http/1.1 2>/dev/null \
  | grep "ALPN protocol"
# ALPN protocol: h2
```

OCSP stapling check:

```bash
echo | openssl s_client -connect example.com:443 -servername example.com -status 2>/dev/null \
  | grep -A 20 "OCSP response"
# OCSP response:
# ======================================
# OCSP Response Data:
#     OCSP Response Status: successful (0x0)
# ...
```

Or "no response sent" — meaning stapling isn't configured.

STARTTLS smtp 587:

```bash
openssl s_client -connect mail.example.com:587 -starttls smtp -servername mail.example.com
```

STARTTLS imap:

```bash
openssl s_client -connect imap.example.com:143 -starttls imap
```

STARTTLS postgres:

```bash
openssl s_client -connect db.example.com:5432 -starttls postgres
```

mTLS client connect:

```bash
openssl s_client -connect mtls.example.com:443 -servername mtls.example.com \
  -cert client.pem -key client.key -CAfile ca.pem
```

Test a specific cipher:

```bash
openssl s_client -connect example.com:443 -servername example.com \
  -tls1_2 -cipher 'ECDHE-RSA-AES256-GCM-SHA384'
```

Wireshark decryption with -keylogfile:

```bash
SSLKEYLOGFILE=/tmp/keys.log openssl s_client -connect example.com:443 \
  -servername example.com -keylogfile /tmp/keys.log
# Then in Wireshark: Edit > Preferences > Protocols > TLS > (Pre)-Master-Secret log file
```

## `openssl s_server` — Quick TLS Server

Tester counterpart. Useful when you need a TLS endpoint right now.

```bash
# Echo server on :4433
openssl s_server -cert cert.pem -key key.pem -accept 4433

# HTTP-ish "this is the request" responder (-www)
openssl s_server -cert cert.pem -key key.pem -accept 4433 -www
# Then: curl -k https://localhost:4433/

# HTTP file server (-WWW serves files from cwd):
openssl s_server -cert cert.pem -key key.pem -accept 4433 -WWW

# mTLS — require client cert:
openssl s_server -cert cert.pem -key key.pem -accept 4433 \
  -CAfile ca.pem -Verify 1 -www

# Verify but allow no cert:
openssl s_server -cert cert.pem -key key.pem -accept 4433 \
  -CAfile ca.pem -verify 1 -www

# Force TLS 1.3:
openssl s_server -cert cert.pem -key key.pem -accept 4433 -tls1_3 -www

# Debug a problem client:
openssl s_server -cert cert.pem -key key.pem -accept 4433 -msg -trace -www
```

`-Verify N` (capital V) = require client cert. `-verify N` (lower) = request but allow none. `N` is verification depth, usually 1 or 2.

## `openssl x509` — Certificate Manipulation

Beyond inspection, `x509` can sign CSRs (the lightweight alternative to `openssl ca`):

```bash
openssl x509 -req -in req.csr \
  -CA ca.pem -CAkey ca.key -CAcreateserial \
  -out cert.pem -days 365 -sha256

# With extensions (SAN, EKU):
openssl x509 -req -in req.csr \
  -CA ca.pem -CAkey ca.key -CAcreateserial \
  -out cert.pem -days 365 -sha256 \
  -extfile <(printf "subjectAltName=DNS:example.com,DNS:www.example.com\nextendedKeyUsage=serverAuth")

# Or with named extension section in a config:
openssl x509 -req -in req.csr \
  -CA ca.pem -CAkey ca.key -CAcreateserial \
  -out cert.pem -days 365 -sha256 \
  -extfile ext.cnf -extensions v3_ext
```

`ext.cnf` example:

```ini
[v3_ext]
basicConstraints     = CA:FALSE
keyUsage             = digitalSignature, keyEncipherment
extendedKeyUsage     = serverAuth, clientAuth
subjectAltName       = @alt_names

[alt_names]
DNS.1 = example.com
DNS.2 = www.example.com
```

`-CAcreateserial` creates `ca.srl` next to `ca.pem` containing the next serial number. Subsequent runs pass `-CAserial ca.srl` instead.

`-copy_extensions copy` (3.x): preserve extensions from CSR. By default they are stripped — a frequent source of "where did my SAN go" bugs.

```bash
openssl x509 -req -in req.csr \
  -CA ca.pem -CAkey ca.key -CAcreateserial \
  -copy_extensions copy \
  -out cert.pem -days 365 -sha256
```

`-addtrust` / `-addreject` add trust attributes:

```bash
openssl x509 -in cert.pem -addtrust serverAuth -out trusted.pem
```

## `openssl pkcs12` — PFX / P12 Bundles

PKCS#12 is the Windows / Java keystore / mobile-config format combining cert + chain + key in one (usually encrypted) file. Extension is `.p12` or `.pfx`.

Export — bundle a cert, key, and chain:

```bash
openssl pkcs12 -export \
  -out bundle.p12 \
  -inkey key.pem \
  -in cert.pem \
  -certfile chain.pem \
  -name "Production cert" \
  -passout pass:hunter2
```

`-name` sets the friendly name (visible in keystores). Without it the alias is empty.

Import — extract everything:

```bash
openssl pkcs12 -in bundle.p12 -out combined.pem -nodes
# (combined.pem will contain key + cert + any chain certs)
```

Extract just the private key:

```bash
openssl pkcs12 -in bundle.p12 -nocerts -nodes -out key.pem
```

Extract just the leaf cert:

```bash
openssl pkcs12 -in bundle.p12 -clcerts -nokeys -out cert.pem
```

Extract just the CA chain:

```bash
openssl pkcs12 -in bundle.p12 -cacerts -nokeys -out chain.pem
```

3.x compatibility for legacy P12 files (PBE-SHA1-3DES, RC2-40):

```bash
# 3.x reading 1.1-style PBE-SHA1-3DES bundle — fails by default:
openssl pkcs12 -in legacy.p12 -nodes -out out.pem
# Error: ... unsupported

# Fix:
openssl pkcs12 -legacy -in legacy.p12 -nodes -out out.pem
# Or:
openssl pkcs12 -provider legacy -provider default -in legacy.p12 -nodes -out out.pem
```

3.x writing 1.1-compatible P12 (for Java 8 / old Windows):

```bash
openssl pkcs12 -export -legacy \
  -out compat.p12 -inkey key.pem -in cert.pem
# Uses PBE-SHA1-3DES + 3DES — works with old keytool / Windows XP.
```

Modern (3.x default) bundle uses AES-256 + PBKDF2-SHA256 — strong but rejected by old tools.

Set a friendly password algorithm explicitly:

```bash
openssl pkcs12 -export -out bundle.p12 \
  -inkey key.pem -in cert.pem \
  -keypbe AES-256-CBC -certpbe AES-256-CBC \
  -macalg SHA256 -iter 200000
```

## `openssl pkcs7` — PKCS#7 / S/MIME / CMS Bundles

PKCS#7 is the cert-only bundle format used for Windows certificate chains (`.p7b`, `.p7c`). It carries no private keys.

Inspect:

```bash
openssl pkcs7 -in bundle.p7b -inform DER -print_certs -noout
# Or PEM:
openssl pkcs7 -in bundle.p7b -inform PEM -print_certs -noout
```

Convert P7B (DER) to PEM bundle:

```bash
openssl pkcs7 -in bundle.p7b -inform DER -print_certs -out chain.pem
# (chain.pem will contain each cert as a PEM block)
```

Convert PEM chain to P7B:

```bash
openssl crl2pkcs7 -nocrl -certfile chain.pem -out bundle.p7b -outform DER
```

For S/MIME signing/encryption use `openssl smime` (legacy) or `openssl cms` (preferred in 3.x):

```bash
openssl cms -sign -in msg.txt -signer cert.pem -inkey key.pem -out msg.p7s
openssl cms -verify -in msg.p7s -CAfile ca.pem
openssl cms -encrypt -in msg.txt -out msg.p7m -outform PEM cert.pem
openssl cms -decrypt -in msg.p7m -inkey key.pem -out msg.txt
```

## `openssl pkcs8` — PKCS#8 Format

PKCS#8 is the modern standard private-key container (`-----BEGIN PRIVATE KEY-----` for unencrypted, `-----BEGIN ENCRYPTED PRIVATE KEY-----` for encrypted). All new code should produce PKCS#8.

Convert legacy PKCS#1 (`-----BEGIN RSA PRIVATE KEY-----`) to PKCS#8:

```bash
openssl pkcs8 -topk8 -nocrypt -in old-key.pem -out pkcs8-key.pem
```

Encrypted PKCS#8:

```bash
openssl pkcs8 -topk8 -in old-key.pem -out enc-key.pem -v2 aes-256-cbc
# -v2 selects the PBES2 (PKCS#5 v2) algorithm. -v1 is PBE.
```

Convert PKCS#8 back to PKCS#1 (rarely needed):

```bash
openssl rsa -in pkcs8-key.pem -traditional -out pkcs1-key.pem
```

Convert PEM ↔ DER:

```bash
openssl pkcs8 -topk8 -nocrypt -in old-key.pem -outform DER -out key.der
openssl pkcs8 -inform DER -in key.der -nocrypt -out key.pem
```

## Format Conversion

Cheat table — what command for what conversion:

| From | To | Command |
|---|---|---|
| Cert PEM | Cert DER | `openssl x509 -in c.pem -outform DER -out c.der` |
| Cert DER | Cert PEM | `openssl x509 -in c.der -inform DER -out c.pem` |
| Key PEM | Key DER | `openssl pkey -in k.pem -outform DER -out k.der` |
| Key DER | Key PEM | `openssl pkey -in k.der -inform DER -out k.pem` |
| PKCS#1 RSA | PKCS#8 | `openssl pkcs8 -topk8 -nocrypt -in k.pem -out p8.pem` |
| PKCS#8 | PKCS#1 RSA | `openssl rsa -in p8.pem -traditional -out k.pem` |
| Cert + key | PKCS#12 | `openssl pkcs12 -export -inkey k.pem -in c.pem -out b.p12` |
| PKCS#12 | Cert + key | `openssl pkcs12 -in b.p12 -nodes -out combined.pem` |
| PEM bundle | PKCS#7 | `openssl crl2pkcs7 -nocrl -certfile c.pem -out b.p7b` |
| PKCS#7 | PEM bundle | `openssl pkcs7 -in b.p7b -print_certs -out c.pem` |
| CSR PEM | CSR DER | `openssl req -in r.csr -outform DER -out r.der` |
| CRL PEM | CRL DER | `openssl crl -in c.pem -outform DER -out c.der` |

PEM `BEGIN/END` headers reference:

| Header | Meaning |
|---|---|
| `BEGIN CERTIFICATE` | X.509 cert |
| `BEGIN TRUSTED CERTIFICATE` | X.509 cert with trust attributes |
| `BEGIN CERTIFICATE REQUEST` | CSR (PKCS#10) |
| `BEGIN X509 CRL` | Certificate Revocation List |
| `BEGIN PRIVATE KEY` | PKCS#8 unencrypted private key |
| `BEGIN ENCRYPTED PRIVATE KEY` | PKCS#8 encrypted |
| `BEGIN RSA PRIVATE KEY` | PKCS#1 RSA (legacy) |
| `BEGIN EC PRIVATE KEY` | SEC1 EC (legacy) |
| `BEGIN DSA PRIVATE KEY` | DSA legacy |
| `BEGIN PUBLIC KEY` | SubjectPublicKeyInfo |
| `BEGIN RSA PUBLIC KEY` | PKCS#1 RSA public (rare) |
| `BEGIN DH PARAMETERS` | DH group parameters |
| `BEGIN EC PARAMETERS` | EC named curve |
| `BEGIN PKCS7` | PKCS#7 / CMS |
| `BEGIN OCSP REQUEST` | OCSP request |
| `BEGIN OCSP RESPONSE` | OCSP response |

Quick one-liners:

```bash
# Detect format
file cert.???
# cert.pem: PEM certificate
# cert.der: data           (DER is opaque to file(1))

# Confirm PEM header
head -1 cert.???

# View any PEM block:
openssl asn1parse -in cert.pem
```

## `openssl dgst` — Hashing and Signing

Hash a file:

```bash
openssl dgst -sha256 file
# SHA2-256(file)= 9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08

openssl dgst -sha1   file        # legacy, do not use for security
openssl dgst -sha384 file
openssl dgst -sha512 file
openssl dgst -sha3-256 file
openssl dgst -sha3-512 file
openssl dgst -blake2b512 file
openssl dgst -blake2s256 file
openssl dgst -ripemd160 file     # 3.x: needs -provider legacy
openssl dgst -md5 file           # legacy

# Multiple files
openssl dgst -sha256 *.tar.gz

# Just the hex (no filename prefix)
openssl dgst -sha256 -r file
# 9f86d081...  *file               (BSD-ish format with -r)
openssl dgst -sha256 file | awk '{print $2}'
```

Streaming from stdin:

```bash
echo -n hello | openssl dgst -sha256
# SHA2-256(stdin)= 2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824
```

Binary output:

```bash
openssl dgst -sha256 -binary file > file.sha256.bin
```

HMAC:

```bash
openssl dgst -sha256 -hmac "secret" file
# HMAC-SHA2-256(file)= ...
echo -n "data" | openssl dgst -sha256 -hmac "secret"
```

3.x HMAC via `mac` (preferred):

```bash
openssl mac -digest SHA256 -macopt hexkey:736563726574 HMAC < file
# (HMAC mac with hex key = "secret")
```

Signing — RSA / ECDSA / Ed25519:

```bash
# Sign with private key
openssl dgst -sha256 -sign key.pem -out file.sig file

# Verify with public key
openssl dgst -sha256 -verify pub.pem -signature file.sig file
# Verified OK         (exit 0)
# Verification failure (exit 1)

# Ed25519 — no separate hash (the algorithm hashes internally):
openssl pkeyutl -sign -inkey ed25519.pem -rawin -in file -out file.sig
openssl pkeyutl -verify -inkey pub.pem -pubin -rawin -in file -sigfile file.sig
```

Output formats: `-binary` (raw signature bytes), default (hex-ish text). For machine consumption use `-binary`. To base64-encode:

```bash
openssl dgst -sha256 -sign key.pem file | openssl base64
```

## `openssl enc` — Symmetric Encryption

`enc` is OpenSSL's general-purpose symmetric tool. It is **legacy by 2025 standards** — for new code prefer `age`, `sops`, or `gpg --symmetric`. The historical bare-MD5 key-derivation flaw (CVE-class problem) is why `-pbkdf2` is now mandatory.

Modern PBE encryption:

```bash
openssl enc -aes-256-cbc -pbkdf2 -iter 600000 -salt \
  -in plain.txt -out cipher.bin -pass pass:hunter2

# Decrypt
openssl enc -aes-256-cbc -pbkdf2 -iter 600000 -d \
  -in cipher.bin -out plain.txt -pass pass:hunter2
```

Why `-pbkdf2 -iter 600000`:

- Without `-pbkdf2`, OpenSSL uses the broken `EVP_BytesToKey` (single-pass MD5 by default). 3.x prints `*** WARNING : deprecated key derivation used. Using -iter or -pbkdf2 would be better.`
- `-iter` sets PBKDF2 rounds. 600000 is the OWASP 2023 recommendation for PBKDF2-SHA256.
- `-salt` is on by default.

Algorithms:

```bash
openssl enc -aes-256-gcm -pbkdf2 -iter 600000 -in p -out c -pass pass:x
# Note: enc + GCM is awkward (no AAD support, format embeds tag). Prefer libsodium/age.
openssl enc -aes-256-cbc -pbkdf2 -iter 600000 ...        # most-compatible
openssl enc -chacha20    -pbkdf2 -iter 600000 ...
openssl enc -aes-128-ctr -pbkdf2 -iter 600000 ...
```

List supported ciphers:

```bash
openssl list -cipher-algorithms
openssl enc -list                      # 3.x alias
openssl enc -ciphers                   # legacy alias
```

Base64 wrapping:

```bash
# Encrypt + base64
openssl enc -aes-256-cbc -pbkdf2 -iter 600000 -salt -a \
  -in plain -out cipher.b64 -pass pass:x

# -a wraps lines at 64 chars; -A produces a single line
openssl enc -aes-256-cbc -pbkdf2 -iter 600000 -salt -A \
  -in plain -out cipher.b64 -pass pass:x

# Decrypt + un-base64
openssl enc -aes-256-cbc -pbkdf2 -iter 600000 -d -a \
  -in cipher.b64 -out plain -pass pass:x
```

Key + IV directly (advanced; you are responsible for IV uniqueness):

```bash
openssl enc -aes-256-cbc \
  -K $(openssl rand -hex 32) \
  -iv $(openssl rand -hex 16) \
  -in p -out c
```

Decrypt failure looks like:

```
bad decrypt
4007D7BB957F0000:error:1C800064:Provider routines::bad decrypt:...
```

Causes: wrong password, wrong cipher, wrong `-pbkdf2`/`-iter` combo, file corruption, or trying to decrypt a non-OpenSSL ciphertext.

## `openssl rand` — Secure Random

```bash
openssl rand -hex 16          # 128 bits as 32 hex chars
# 8a4c2d1e9f3b7a5c6d8e1f2a3b4c5d6e

openssl rand -hex 32          # 256 bits

openssl rand -base64 32       # 256 bits as ~44 chars base64
# 7XqL3...

openssl rand -base64 24       # easier to type (32 chars)

openssl rand 16 | xxd          # 16 raw bytes (binary)

openssl rand -out /tmp/rand.bin 4096   # 4 KB random file

# Print only (no newline) — useful in scripts
openssl rand -hex 16 | tr -d '\n'
```

`openssl rand` reads from the OS RNG (`/dev/urandom` on Linux, `getentropy()` on BSDs/macOS, `BCryptGenRandom` on Windows). It does NOT use legacy `RANDFILE` for actual entropy in 3.x — it just touches the file for backwards compat.

For passwords intended for humans:

```bash
openssl rand -base64 18 | tr -d '/+=' | head -c 24
# Url-safe-ish, 24-char passphrase.
```

For a strong UUID-style token:

```bash
openssl rand -hex 16
# 32-char hex, 128 bits — equivalent to a UUID's entropy.
```

Hardware RNG hint: on systems with TPM or HSM, OpenSSL 3.x can use a provider that pulls from there (`-provider tpm2-openssl` etc.). The `rand` subcommand inherits whatever the active RNG provider is.

## `openssl ts` — RFC 3161 Timestamping

Trusted timestamping proves a hash existed at a time. The `ts` subcommand creates request and verifies response from a TSA.

Build a request:

```bash
# Hash-only request (privacy-preserving; TSA never sees the file):
openssl ts -query -data file -sha256 -cert -no_nonce -out request.tsq

# -cert asks the TSA to include its signing cert in the response.
# -no_nonce omits the random nonce (some TSAs don't support nonces).

# Submit to a public TSA (e.g. freeTSA.org):
curl -s -H "Content-Type: application/timestamp-query" \
  --data-binary "@request.tsq" \
  https://freetsa.org/tsr -o response.tsr

# Inspect the response:
openssl ts -reply -in response.tsr -text
# Status info:
#     Status: Granted.
#     Status description: unspecified
# TST info:
#     Version: 1
#     Policy OID: ...
#     Hash Algorithm: sha256
#     Message data:
#         0000 - <hash bytes>
#     Time stamp: Apr  1 12:34:56.789 2024 GMT
```

Verify a response:

```bash
openssl ts -verify -data file \
  -in response.tsr \
  -CAfile freetsa-cacert.pem \
  -untrusted freetsa-tsa.pem
# Verification: OK
```

Common TSAs:

- `https://freetsa.org/tsr` — free; cert chain at https://freetsa.org/ ;
- `http://timestamp.digicert.com` — DigiCert
- `http://timestamp.sectigo.com` — Sectigo
- `http://tsa.starfieldtech.com` — GoDaddy
- `http://tss.accv.es:8318/tsa` — ACCV (Spain)

The `-policy` flag selects a policy OID if the TSA supports more than one. For most public TSAs the default is fine.

## `openssl ocsp` — OCSP Stapling and Verification

OCSP (Online Certificate Status Protocol, RFC 6960) checks revocation status without downloading a full CRL. Each cert's "Authority Information Access (AIA)" extension contains the OCSP URL.

Find the OCSP URL of a cert:

```bash
openssl x509 -in cert.pem -noout -ocsp_uri
# http://ocsp.example.com
```

Verify a cert against its OCSP responder:

```bash
openssl ocsp \
  -issuer issuer.pem \
  -cert cert.pem \
  -url http://ocsp.example.com \
  -CAfile ca-bundle.pem \
  -resp_text \
  -no_nonce
# cert.pem: good
# This Update: ...
# Next Update: ...
```

Statuses:

- `good` — not revoked, still valid
- `revoked` — revoked at the listed time
- `unknown` — responder doesn't know about the cert (treat as suspect)

`-no_nonce` is needed for many responders (especially Let's Encrypt) which don't echo nonces.

OCSP-must-staple — the cert extension (TLS Feature, RFC 7633) telling clients to fail if the server doesn't staple OCSP. Add to `ext.cnf`:

```ini
[v3_ext]
1.3.6.1.5.5.7.1.24 = DER:30:03:02:01:05
# This is the encoded value status_request(5).
```

Inspect a cert's OCSP-must-staple:

```bash
openssl x509 -in cert.pem -noout -text | grep -A1 "TLS Feature"
```

## `openssl crl` — Certificate Revocation Lists

CRLs are downloadable lists of revoked cert serials. The CDP extension on a cert points to the CRL URL.

Find a cert's CRL URL:

```bash
openssl x509 -in cert.pem -noout -text | grep -A4 "X509v3 CRL Distribution Points"
# X509v3 CRL Distribution Points:
#     Full Name:
#       URI:http://crl.example.com/ca.crl
```

Inspect a CRL:

```bash
openssl crl -in crl.pem -noout -text
# Certificate Revocation List (CRL):
#         Version 2 (0x1)
#         Signature Algorithm: sha256WithRSAEncryption
#         Issuer: ...
#         Last Update: ...
#         Next Update: ...
# Revoked Certificates:
#     Serial Number: 03ABCD...
#         Revocation Date: ...
#         CRL entry extensions:
#             X509v3 CRL Reason Code:
#                 Key Compromise
```

Verify the CRL signature against its issuer:

```bash
openssl crl -in crl.pem -CAfile ca.pem -noout
# verify OK
```

Convert formats:

```bash
openssl crl -in crl.pem -outform DER -out crl.der
openssl crl -in crl.der -inform DER -out crl.pem
```

Verify a cert against its CRL (combined):

```bash
cat ca.pem crl.pem > bundle.pem
openssl verify -crl_check -CAfile bundle.pem cert.pem
# cert.pem: OK            (or "certificate revoked")

# crl_check_all checks every cert in chain
openssl verify -crl_check_all -CAfile bundle.pem cert.pem
```

## `openssl verify` — Verify Certificate Chain

Validate a leaf cert against a trust bundle:

```bash
openssl verify -CAfile bundle.pem cert.pem
# cert.pem: OK
```

If the bundle has the root only and the leaf needs an intermediate:

```bash
openssl verify -untrusted intermediate.pem -CAfile root.pem cert.pem
```

Chain split-bundle (intermediate concatenated with leaf into chain.pem):

```bash
# server-chain.pem == leaf || intermediate
openssl verify -CAfile root.pem server-chain.pem
# WARNING: only the first cert in -in is verified — use -untrusted for the rest
```

Use system trust store explicitly:

```bash
openssl verify cert.pem
# Without -CAfile, defaults to OPENSSLDIR/certs and (3.x) /etc/ssl/certs.

openssl verify -no-CAfile -no-CApath -CAfile mybundle.pem cert.pem
# Strictly use only mybundle.pem (no system fallback).
```

Hostname check (3.x):

```bash
openssl verify -CAfile root.pem -verify_hostname example.com cert.pem
```

Common failure messages and fixes:

| Message | Cause | Fix |
|---|---|---|
| `unable to get local issuer certificate` | Trust path missing intermediate | Add `-untrusted intermediate.pem` |
| `unable to verify the first certificate` | No trust anchor | Add correct `-CAfile root.pem` |
| `certificate has expired` | `notAfter` in past | Renew |
| `certificate is not yet valid` | `notBefore` in future; clock skew | Fix clock or wait |
| `self signed certificate` | Self-signed leaf | Add `-CAfile self.pem` (trust self) |
| `self signed certificate in certificate chain` | Chain ends in root not in store | Add root to `-CAfile` |
| `certificate revoked` | Found in CRL/OCSP | Reissue |
| `unsupported certificate purpose` | Cert doesn't allow the EKU | Reissue with proper `extendedKeyUsage` |
| `EE certificate key too weak` | Key size below SECLEVEL | Increase key size or lower `-auth_level` |
| `CA signature digest algorithm too weak` | SHA-1 signed cert in 3.x | Reissue with SHA-256 |
| `Hostname mismatch` | SAN/CN doesn't match `-verify_hostname` | Reissue cert with correct SAN |

## `openssl ciphers` — Cipher Suite Inspection

```bash
openssl ciphers -v
# Show every cipher OpenSSL would offer at default settings.

openssl ciphers -v 'HIGH:!aNULL:!MD5:!3DES'
# Filter — only HIGH-strength ciphers, no anonymous, no MD5, no 3DES.

openssl ciphers -v 'ECDHE+AESGCM:ECDHE+CHACHA20'
# Mozilla "Modern" style.

openssl ciphers -s -v 'TLSv1.3'
# -s = "supported": include only currently-enabled ciphers.

openssl ciphers -V 'HIGH'
# -V = include hex codes:
# 0x13,0x02 - TLS_AES_256_GCM_SHA384
# 0xC0,0x30 - ECDHE-RSA-AES256-GCM-SHA384

openssl ciphers -tls1_3
# TLS_AES_256_GCM_SHA384:TLS_CHACHA20_POLY1305_SHA256:TLS_AES_128_GCM_SHA256

openssl ciphers -tls1_2 'HIGH:!aNULL'
```

OpenSSL cipher syntax — operators:

- `:` — separator (also `,` and ` `)
- `!CIPHER` — permanently exclude
- `-CIPHER` — exclude (can be re-added)
- `+CIPHER` — re-add at the end
- `@STRENGTH` — sort by strength
- `@SECLEVEL=2` — set security level

Aliases:

- `HIGH` — keys >= 128 bits
- `MEDIUM` — 128-bit ciphers (RC2, RC4, etc.)
- `LOW` — 64-bit (DES; off by default)
- `EXPORT` — export-grade (off)
- `aNULL` — anonymous (no auth) — always exclude
- `eNULL` — no encryption — always exclude
- `kRSA` / `kECDHE` / `kDHE` — key exchange
- `aRSA` / `aECDSA` / `aDSS` — auth
- `AES`, `AESGCM`, `CHACHA20`, `CAMELLIA`, `3DES`, `RC4`, `IDEA`, `SEED` — cipher
- `SHA1`, `SHA256`, `SHA384` — MAC
- `TLSv1.2`, `TLSv1.3` — protocol-only

Mozilla SSL Configuration Generator presets (https://ssl-config.mozilla.org/):

- **Modern** (TLS 1.3 only — `TLS_AES_256_GCM_SHA384:TLS_CHACHA20_POLY1305_SHA256:TLS_AES_128_GCM_SHA256`)
- **Intermediate** (TLS 1.2 + 1.3, recommended default)
- **Old** (TLS 1.0 + 1.1 + 1.2 + 1.3 — for legacy clients)

## `openssl version` Subcommand Variations

```bash
openssl version          # version string only
openssl version -v       # verbose (same as no flag)
openssl version -a       # all info — most useful
openssl version -d       # OPENSSLDIR — where openssl.cnf lives
openssl version -e       # ENGINESDIR
openssl version -m       # MODULESDIR — 3.x providers
openssl version -p       # platform string
openssl version -b       # build date
openssl version -f       # compile flags
openssl version -o       # options (bignum size, threading)
openssl version -r       # seed source
openssl version -c       # CPU info (CPU feature flags)
```

Use cases:

```bash
# Where does openssl.cnf live on this box?
echo "$(openssl version -d | sed 's/OPENSSLDIR: //; s/"//g')/openssl.cnf"

# Where do providers come from?
openssl version -m

# Was this binary built with TLS 1.3 enabled?
openssl version -f | grep -o 'no-tls1_3'   # if present, TLS 1.3 is disabled
```

## Engines / Providers

OpenSSL 1.x **engines** (legacy plugins): PKCS#11 for HSMs/Yubikey, GOST, AF-ALG, etc.

```bash
# 1.x or 3.x with engine compat
openssl engine -t -c
# (built-in) static engines
# (dynamic) Dynamic engine loading support
# (pkcs11) pkcs11 engine
```

OpenSSL 3.x **providers** (the new model):

```bash
openssl list -providers
# Providers:
#   default
#     name: OpenSSL Default Provider
#     version: 3.2.1
#     status: active
#   legacy
#     name: OpenSSL Legacy Provider
#     status: active

# Activate legacy at the command line:
openssl dgst -md5 file -provider legacy -provider default
# (-provider order matters: default fallback comes after legacy)

# Use FIPS:
openssl req -newkey rsa:3072 -provider fips -provider base ...
```

PKCS#11 engine for HSM / Yubikey / smart card:

```bash
# Install the libp11 / opensc package
apt install libengine-pkcs11-openssl libp11

# Inspect tokens
pkcs11-tool --module /usr/lib/x86_64-linux-gnu/pkcs11/opensc-pkcs11.so --list-slots

# Reference a key by PKCS#11 URI
openssl req -engine pkcs11 -keyform engine \
  -key "pkcs11:object=YubiKey-AUTH;type=private" \
  -new -subj "/CN=token-cert" -out token.csr
```

In 3.x, the equivalent is the **pkcs11-provider**:

```bash
openssl req -provider pkcs11 -provider default \
  -keyform PROVIDER \
  -key "pkcs11:object=YubiKey-AUTH;type=private;pin-value=123456" \
  -new -subj "/CN=token-cert" -out token.csr
```

## Configuration File

Default location: `$(openssl version -d)/openssl.cnf` (e.g. `/etc/ssl/openssl.cnf`, `/opt/homebrew/etc/openssl@3/openssl.cnf`).

Override:

```bash
OPENSSL_CONF=/tmp/my-openssl.cnf openssl req -new ...

# Bypass entirely (debug):
OPENSSL_CONF=/dev/null openssl req -new ...
```

Section anatomy:

```ini
# Top of file — controls what runs at openssl startup
openssl_conf = openssl_init

[openssl_init]
providers = provider_sect
oid_section = new_oids

[provider_sect]
default = default_sect
legacy  = legacy_sect

[default_sect]
activate = 1

[legacy_sect]
activate = 1

# req: defaults for `openssl req`
[req]
default_bits        = 2048
default_md          = sha256
prompt              = no
distinguished_name  = req_distinguished_name
req_extensions      = v3_req

[req_distinguished_name]
C  = US
ST = California
O  = Acme
CN = example.com

[v3_req]
basicConstraints     = CA:FALSE
keyUsage             = digitalSignature, keyEncipherment
extendedKeyUsage     = serverAuth
subjectAltName       = @alt_names

[alt_names]
DNS.1 = example.com
DNS.2 = www.example.com

# v3_ca: defaults for self-signed root cert
[v3_ca]
basicConstraints       = critical,CA:TRUE
subjectKeyIdentifier   = hash
authorityKeyIdentifier = keyid:always,issuer
keyUsage               = critical,digitalSignature,cRLSign,keyCertSign
```

Inspect what flags would be applied:

```bash
OPENSSL_CONF=./mycnf openssl req -newkey rsa:2048 -nodes -keyout k.pem -out r.csr
```

Debug bad config:

```bash
OPENSSL_CONF=/dev/null openssl ...    # bypass entirely; if this works, the config is broken.
```

## Common Recipes

### Inspect a Remote Cert (the canonical one-liner)

```bash
echo | openssl s_client -connect example.com:443 -servername example.com 2>/dev/null \
  | openssl x509 -noout -text -dates -fingerprint -sha256 -subject -issuer
```

Compact for monitoring:

```bash
echo | openssl s_client -connect example.com:443 -servername example.com 2>/dev/null \
  | openssl x509 -noout -dates -subject
```

### Extract Public Key from Cert

```bash
openssl x509 -in cert.pem -noout -pubkey > pub.pem
```

### Verify a Key Matches a Cert

RSA:

```bash
diff <(openssl rsa  -in key.pem  -modulus -noout) \
     <(openssl x509 -in cert.pem -modulus -noout)
# (no output = match)
```

EC / Ed25519 (no modulus — compare public key bytes):

```bash
diff <(openssl pkey -in key.pem  -pubout) \
     <(openssl x509 -in cert.pem -pubkey -noout)
```

CSR ↔ key match:

```bash
diff <(openssl rsa -in key.pem  -modulus -noout) \
     <(openssl req -in req.csr -modulus -noout)
```

### Wildcard SAN Cert

```bash
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem \
  -days 397 -nodes -sha256 \
  -subj "/CN=example.com" \
  -addext "subjectAltName=DNS:example.com,DNS:*.example.com"
```

### mTLS Server + Client Setup

```bash
# 1. CA
openssl req -x509 -newkey rsa:4096 -keyout ca.key -out ca.crt \
  -days 3650 -nodes -sha256 -subj "/CN=Internal mTLS CA"

# 2. Server cert
openssl req -newkey rsa:4096 -keyout server.key -out server.csr \
  -nodes -subj "/CN=mtls.example.com" \
  -addext "subjectAltName=DNS:mtls.example.com"
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
  -out server.crt -days 365 -sha256 -copy_extensions copy \
  -extfile <(printf "extendedKeyUsage=serverAuth\nsubjectAltName=DNS:mtls.example.com")

# 3. Client cert
openssl req -newkey rsa:4096 -keyout client.key -out client.csr \
  -nodes -subj "/CN=alice@example.com"
openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
  -out client.crt -days 365 -sha256 \
  -extfile <(printf "extendedKeyUsage=clientAuth")

# 4. Test server
openssl s_server -cert server.crt -key server.key -CAfile ca.crt \
  -accept 4433 -Verify 1 -www

# 5. Test client
openssl s_client -connect localhost:4433 -CAfile ca.crt \
  -cert client.crt -key client.key
```

### Private CA — Build Root + Intermediate + Sign Leaf

```bash
# Root
openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:4096 \
  -aes-256-cbc -out root.key
openssl req -new -x509 -key root.key -days 7300 -sha256 \
  -out root.crt -subj "/CN=Acme Root CA" \
  -extensions v3_ca

# Intermediate
openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:4096 \
  -aes-256-cbc -out int.key
openssl req -new -key int.key -out int.csr \
  -subj "/CN=Acme Intermediate CA"
openssl x509 -req -in int.csr -CA root.crt -CAkey root.key -CAcreateserial \
  -out int.crt -days 3650 -sha256 \
  -extfile <(printf "basicConstraints=critical,CA:true,pathlen:0\nkeyUsage=critical,digitalSignature,cRLSign,keyCertSign")

# Leaf (server)
openssl req -newkey rsa:3072 -nodes -keyout leaf.key -out leaf.csr \
  -subj "/CN=app.example.com" \
  -addext "subjectAltName=DNS:app.example.com"
openssl x509 -req -in leaf.csr -CA int.crt -CAkey int.key -CAcreateserial \
  -out leaf.crt -days 397 -sha256 -copy_extensions copy \
  -extfile <(printf "extendedKeyUsage=serverAuth\nsubjectAltName=DNS:app.example.com")

# Server bundle (leaf + intermediate)
cat leaf.crt int.crt > leaf-fullchain.pem

# Verify
openssl verify -CAfile root.crt -untrusted int.crt leaf.crt
```

### Convert Every Format Combination

```bash
# PEM cert -> DER
openssl x509 -in c.pem -outform DER -out c.der

# DER cert -> PEM
openssl x509 -in c.der -inform DER -out c.pem

# PEM key (PKCS#8) -> DER
openssl pkey -in k.pem -outform DER -out k.der

# Encrypted PEM key -> unencrypted PEM
openssl pkey -in k-enc.pem -out k-plain.pem

# PEM key + PEM cert -> P12
openssl pkcs12 -export -inkey k.pem -in c.pem -out b.p12

# P12 -> combined PEM
openssl pkcs12 -in b.p12 -nodes -out combined.pem

# P12 -> separate PEM key + PEM cert
openssl pkcs12 -in b.p12 -nocerts -nodes -out k.pem
openssl pkcs12 -in b.p12 -clcerts -nokeys -out c.pem

# Multiple PEM certs -> P7B chain
openssl crl2pkcs7 -nocrl -certfile chain.pem -out chain.p7b -outform DER

# P7B -> PEM chain
openssl pkcs7 -in chain.p7b -inform DER -print_certs -out chain.pem
```

## Common Errors and Fixes

EXACT error text (hex codes vary by OpenSSL version) and what to do:

```text
unable to load certificate
4007D7BB957F0000:error:0900006e:PEM routines:OPENSSL_internal:NO_START_LINE
```
Cause: file is DER, not PEM; or wrong filename; or encrypted with no `-passin`.
Fix: `file cert.???`; if DER, `openssl x509 -inform DER ...`; supply `-passin pass:...`.

```text
verify error:num=20:unable to get local issuer certificate
```
Cause: chain incomplete; the issuer of one cert in the chain isn't in your trust store.
Fix: `-CAfile root.pem` AND `-untrusted intermediate.pem`. Or download missing intermediate from cert AIA URL.

```text
verify error:num=21:unable to verify the first certificate
```
Cause: no trust anchor in store; nothing chains up to a known root.
Fix: provide correct `-CAfile`. If using system store, ensure CA bundle is installed.

```text
verify error:num=10:certificate has expired
```
Cause: `notAfter` is past.
Fix: renew the cert; or check clock skew (`date`); or `-attime` to override (testing only).

```text
verify error:num=18:self signed certificate
```
Cause: leaf is self-signed; not in trust store.
Fix: trust the leaf with `-CAfile self.pem`. Expected message for self-signed; safe to suppress when you know.

```text
no peer certificate available
```
Cause: server didn't present a cert. Possibilities: hit a non-TLS port (e.g. 80 instead of 443); server using PSK ciphers only; wrong vhost answered without cert.
Fix: confirm port; add `-servername` (SNI); check server config.

```text
wrong tag
4007D7BB957F0000:error:0688010A:asn1 encoding routines::wrong tag
```
Cause: DER vs PEM mismatch, or corrupted file.
Fix: detect format with `head -1 file`; if `-----BEGIN` it's PEM; otherwise DER. Switch `inform`/`outform`.

```text
bad decrypt
4007D7BB957F0000:error:1C800064:Provider routines::bad decrypt
```
Cause: wrong passphrase, wrong cipher, missing `-pbkdf2`, or corrupted ciphertext.
Fix: confirm pass; ensure same `-aes-256-cbc -pbkdf2 -iter N` on encrypt + decrypt; verify file integrity.

```text
Cannot operate without a private key
```
Cause: handed it a public-only key file when private was required.
Fix: use the `key.pem` (PRIVATE KEY), not `pub.pem` (PUBLIC KEY).

```text
error:0308010C:digital envelope routines::unsupported
```
Cause: 3.x default rejects legacy algorithm (MD5, RIPEMD-160, RC2, RC4, DES, 3DES, Blowfish, IDEA, SEED, CAST5, MD2, MD4, MDC-2). Most common with old PKCS#12 files (PBE-SHA1-3DES) and old `enc -des-...`.
Fix: `-provider legacy -provider default`. For `pkcs12`, also add `-legacy`. Or upgrade to a modern algorithm.

```text
Error in encrypted private key
```
Cause: file is corrupted, truncated, or in unexpected format.
Fix: `openssl asn1parse -in key.pem` to see structure; re-export from source.

```text
ssl handshake failure
4007D7BB957F0000:error:0A000410:SSL routines::sslv3 alert handshake failure
```
Cause: cipher / version / cert / SNI mismatch. Could be no shared cipher, bad client cert, no SNI, or protocol version disabled.
Fix: rerun with `-msg -trace -tlsextdebug`; explicitly set `-tls1_2`/`-tls1_3`, `-cipher`, `-servername`.

```text
unable to write 'random state'
```
Cause: `$RANDFILE` (default `~/.rnd`) not writable.
Fix: `RANDFILE=/tmp/.rnd openssl ...` or `unset RANDFILE` (3.x doesn't actually need it).

```text
DH key too small
error:0A00018A:SSL routines::dh key too small
```
Cause: server using DH params <2048 bits; client refusing per Logjam mitigation.
Fix: server: regenerate DH params at 2048+ (`openssl dhparam -out dhparam.pem 2048`) or switch to ECDHE. Client (testing): lower security level: `-cipher 'DEFAULT@SECLEVEL=1'`.

```text
shared cipher
error:0A0000B8:SSL routines::no shared cipher
```
Cause: client and server have no overlap in cipher suite list (often: client TLS 1.3-only meeting TLS 1.2-only server, or vice versa).
Fix: align protocols; check `-tls1_2`/`-tls1_3`; align cipher allowlists.

```text
certificate verify failed
verify error:num=62:Hostname mismatch
```
Cause: cert SAN/CN doesn't match the connecting hostname.
Fix: reissue cert with correct SAN; or use `-noverify_hostname` (testing only).

```text
key values mismatch
```
Cause: `pkcs12 -export` saw a key and cert that don't match each other.
Fix: confirm key and cert are paired (use the modulus / public-key compare from "Verify a Key Matches a Cert").

```text
Unsupported public key type
```
Cause: tool doesn't know the key algorithm (e.g. trying to use Ed448 with very old OpenSSL).
Fix: upgrade OpenSSL to 3.x; or switch algorithm.

```text
3.x: error:25066067:DSO support routines::could not load the shared library
```
Cause: provider/engine not found.
Fix: `openssl version -m` to find MODULESDIR; ensure provider .so is present.

## Common Gotchas (Broken + Fixed)

**1. Forgetting `-servername` (SNI).**

Broken — get default vhost cert, not the one you wanted:

```bash
openssl s_client -connect example.com:443
```

Fixed:

```bash
openssl s_client -connect example.com:443 -servername example.com
```

**2. DER vs PEM encoding mismatch.**

Broken:

```bash
openssl x509 -in cert.der -text -noout
# unable to load certificate / wrong tag
```

Fixed:

```bash
openssl x509 -in cert.der -inform DER -text -noout
```

**3. Missing `-nodes` on automated key gen.**

Broken — pipeline hangs forever waiting for passphrase:

```bash
openssl req -newkey rsa:4096 -keyout key.pem -out r.csr -subj "/CN=x"
```

Fixed:

```bash
openssl req -newkey rsa:4096 -nodes -keyout key.pem -out r.csr -subj "/CN=x"
```

**4. Legacy SSL/TLS version flag against TLS 1.3-only server.**

Broken:

```bash
openssl s_client -connect example.com:443 -tls1_1
# ... handshake failure
```

Fixed:

```bash
openssl s_client -connect example.com:443 -tls1_3
```

**5. PKCS12 created with old algorithms unreadable on 3.x.**

Broken:

```bash
openssl pkcs12 -in old.p12 -nodes -out out.pem
# error:0308010C:digital envelope routines::unsupported
```

Fixed:

```bash
openssl pkcs12 -legacy -in old.p12 -nodes -out out.pem
```

**6. Certificate vs CSR confusion.**

Broken — trying to sign a cert with `req`:

```bash
openssl req -in cert.pem -text -noout
# unable to load X509 request
```

Fixed — use `x509` for certs, `req` for CSRs:

```bash
openssl x509 -in cert.pem -text -noout
```

**7. P12 extract leaving key encrypted by accident.**

Broken — out.pem still has `ENCRYPTED PRIVATE KEY`:

```bash
openssl pkcs12 -in b.p12 -out out.pem
```

Fixed:

```bash
openssl pkcs12 -in b.p12 -out out.pem -nodes
```

**8. Signing with the wrong CA key.**

Broken — silently signs with whatever key was passed:

```bash
openssl x509 -req -in r.csr -CA other-ca.pem -CAkey our-ca.key -out c.pem
# (signature won't verify against other-ca.pem)
```

Fixed — make sure `-CA` and `-CAkey` are the same CA pair:

```bash
openssl x509 -req -in r.csr -CA our-ca.pem -CAkey our-ca.key -out c.pem
```

**9. `-CAcreateserial` race.**

Broken — two parallel `x509 -req -CAcreateserial` runs collide on `ca.srl`:

```bash
openssl x509 -req ... -CAcreateserial &  # PID 1
openssl x509 -req ... -CAcreateserial &  # PID 2 — same serial!
```

Fixed — pre-create `ca.srl`, use `-CAserial`:

```bash
echo 1000 > ca.srl
openssl x509 -req ... -CA ca.pem -CAkey ca.key -CAserial ca.srl -out c1.pem
openssl x509 -req ... -CA ca.pem -CAkey ca.key -CAserial ca.srl -out c2.pem
```

**10. `-extfile` extensions silently ignored.**

Broken — SAN missing from output cert because `-extensions` not specified:

```bash
openssl x509 -req -in r.csr -CA ca.pem -CAkey ca.key \
  -extfile ext.cnf -out c.pem
# ext.cnf has [v3_ext] section but x509 doesn't know which to use
```

Fixed:

```bash
openssl x509 -req -in r.csr -CA ca.pem -CAkey ca.key \
  -extfile ext.cnf -extensions v3_ext -out c.pem
```

**11. CSR extensions stripped during signing.**

Broken — server cert gets the DN but loses the SAN you put in the CSR:

```bash
openssl x509 -req -in r.csr -CA ca.pem -CAkey ca.key -out c.pem
# c.pem has no SAN
```

Fixed (3.x):

```bash
openssl x509 -req -in r.csr -CA ca.pem -CAkey ca.key -copy_extensions copy -out c.pem
```

**12. macOS uses LibreSSL.**

Broken — flag works on Linux but not macOS:

```bash
openssl pkcs12 -legacy -in old.p12 ...
# unknown option `-legacy'   # because /usr/bin/openssl is LibreSSL
```

Fixed — use brew openssl:

```bash
$(brew --prefix openssl@3)/bin/openssl pkcs12 -legacy -in old.p12 ...
```

**13. SHA-1 cert in 3.x fails default.**

Broken:

```bash
openssl verify -CAfile root.pem leaf.pem
# error 68 at 0 depth lookup: CA signature digest algorithm too weak
```

Fixed — reissue with SHA-256+, or temporarily lower auth level:

```bash
openssl verify -auth_level 0 -CAfile root.pem leaf.pem
```

**14. Self-signed cert without SAN in 2025.**

Broken — Chrome/Firefox refuse it; only matches `CN=`:

```bash
openssl req -x509 -newkey rsa:4096 -nodes -keyout k.pem -out c.pem \
  -subj "/CN=localhost"
```

Fixed:

```bash
openssl req -x509 -newkey rsa:4096 -nodes -keyout k.pem -out c.pem \
  -subj "/CN=localhost" \
  -addext "subjectAltName=DNS:localhost,IP:127.0.0.1,IP:::1"
```

**15. `-pass pass:secret` visible in `ps`.**

Broken:

```bash
openssl pkcs12 -export -in c.pem -inkey k.pem -out b.p12 -password pass:hunter2
# `ps fauxw` shows the password to anyone on the box.
```

Fixed:

```bash
echo -n hunter2 > /run/keypass; chmod 600 /run/keypass
openssl pkcs12 -export -in c.pem -inkey k.pem -out b.p12 -password file:/run/keypass
shred -u /run/keypass
```

**16. `enc` without `-pbkdf2`.**

Broken — `*** WARNING : deprecated key derivation used`:

```bash
openssl enc -aes-256-cbc -in p -out c -pass pass:x
```

Fixed:

```bash
openssl enc -aes-256-cbc -pbkdf2 -iter 600000 -in p -out c -pass pass:x
```

**17. Storing keys at default 600 with shared group.**

Broken — `key.pem` ends up `rw-r--r--` and committed to git:

```bash
openssl genpkey -algorithm RSA -out key.pem
git add key.pem
```

Fixed:

```bash
( umask 077 && openssl genpkey -algorithm RSA -out key.pem )
echo 'key.pem' >> .gitignore
chmod 600 key.pem
```

## Performance / Hardening Tips

- **Algorithms.** Prefer ECDSA P-256 (~10x faster verifies than RSA-2048; smaller cert; same security). Use RSA-3072+ for new keys when ECDSA isn't an option; 4096 only if a compliance audit demands. Ed25519 for SSH and software signing.
- **Cipher selection.** AES-256-GCM > AES-128-GCM > ChaCha20-Poly1305 in hardware-AES environments; ChaCha20-Poly1305 wins on mobile (no AES-NI). CBC ciphers are deprecated for new deploys (Lucky-13, padding-oracle). Avoid 3DES, RC4, IDEA, SEED, CAMELLIA-CBC, NULL, EXPORT, anonymous.
- **Protocols.** TLS 1.3 + TLS 1.2 only. TLS 1.0/1.1 forbidden by PCI DSS 3.2.1+. SSLv3 catastrophic (POODLE).
- **Forward secrecy.** Use ECDHE/DHE only (not static RSA `kRSA`). All TLS 1.3 cipher suites are forward-secret by definition.
- **OCSP stapling.** Enable on the server (nginx: `ssl_stapling on; ssl_stapling_verify on;`). Saves clients a round-trip and an OCSP responder fetch — both performance + privacy win.
- **Session resumption.** Enable session tickets with rotating ticket keys. TLS 1.3 PSK resumption is 1-RTT (or 0-RTT with replay risk).
- **HSTS + preload.** `Strict-Transport-Security: max-age=63072000; includeSubDomains; preload`.
- **Mozilla SSL Config Generator (https://ssl-config.mozilla.org/).** The single best resource for "what should my nginx/apache/haproxy config look like." Three presets: Modern (TLS 1.3 only), Intermediate (recommended default, supports IE 11+), Old (browsers older than 2014).
- **Cert lifetime.** 397 days max for public CAs (since 2020). Plan for 90-day rotation (Let's Encrypt) or shorter.
- **Key rotation.** Rotate private keys every cert renewal — `pin`-based fingerprinting catches reuse.
- **DH parameters.** Use `ffdhe2048` / `ffdhe3072` named groups (RFC 7919) instead of generated DH params; OpenSSL 3.x picks them automatically. Generated `dhparam.pem` files are no longer recommended.
- **Provider performance.** AES-NI, ARMv8 Crypto Extensions, and SHA Extensions are auto-detected; verify via `openssl speed -evp aes-256-gcm`.

```bash
openssl speed -evp aes-256-gcm    # benchmark AES
openssl speed rsa3072             # RSA sign/verify
openssl speed ecdsap256
openssl speed ed25519
```

## Idioms

### One-liner cert expiry monitor

```bash
days=$(echo | openssl s_client -connect example.com:443 -servername example.com 2>/dev/null \
  | openssl x509 -noout -enddate \
  | cut -d= -f2 \
  | { read d; echo $(( ($(date -j -f "%b %e %T %Y %Z" "$d" +%s 2>/dev/null \
                       || date -d "$d" +%s) - $(date +%s)) / 86400 )); })
echo "$days days remaining"
```

### Cert-rotation script template

```bash
#!/usr/bin/env bash
set -euo pipefail
DOMAIN="${1:?usage: $0 example.com}"
KEYDIR="/etc/ssl/private"
CERTDIR="/etc/ssl/certs"

# 1. Generate new key (don't overwrite live one yet)
( umask 077 && openssl genpkey -algorithm RSA \
    -pkeyopt rsa_keygen_bits:3072 \
    -out "$KEYDIR/$DOMAIN.new.key" )

# 2. CSR
openssl req -new -key "$KEYDIR/$DOMAIN.new.key" -sha256 \
  -subj "/CN=$DOMAIN" \
  -addext "subjectAltName=DNS:$DOMAIN,DNS:www.$DOMAIN" \
  -out "/tmp/$DOMAIN.csr"

# 3. Submit to ACME (stub — use certbot/acme.sh in real life)
# certbot certonly --csr /tmp/$DOMAIN.csr ...

# 4. Sanity-check the new cert
openssl x509 -in "/tmp/$DOMAIN.crt" -noout -checkend 0
openssl verify -CAfile /etc/ssl/certs/ca-bundle.crt "/tmp/$DOMAIN.crt"

# 5. Atomically swap
mv "/tmp/$DOMAIN.crt" "$CERTDIR/$DOMAIN.crt"
mv "$KEYDIR/$DOMAIN.new.key" "$KEYDIR/$DOMAIN.key"

# 6. Reload nginx
nginx -t && systemctl reload nginx
```

### ACME / Let's Encrypt vs raw OpenSSL

`openssl req` produces a CSR but the public-CA process needs the ACME protocol (RFC 8555) — domain validation, challenges (HTTP-01, DNS-01, TLS-ALPN-01), order, finalization. Don't reinvent it. Tools:

- **certbot** — official EFF client; complex; full-featured.
- **acme.sh** — pure shell; no deps; widely loved; more flexible than certbot for split DNS / wildcard setups.
- **lego** — Go-based; library + CLI; built-in DNS providers.
- **dehydrated** — bash; minimal.
- **Caddy** — webserver with built-in ACME; zero config for many cases.

For internal CAs use `step-ca` (smallstep) — implements ACME against your own root.

### Signing tool combinations

- **cosign** (sigstore) — signs OCI/container images. Verifies via Rekor transparency log.
- **age** (Filippo Valsorda) — file encryption, X25519 + ChaCha20-Poly1305. Replacement for `openssl enc`.
- **minisign** / **signify** — small file signers (Ed25519). Replacement for `openssl dgst -sign`.
- **gpg** — old, complex, full PGP. Use age + ssh-keygen-signing for new code.

### Quick TLS sanity check

```bash
# One-shot: connection works, cert valid, hostname matches, days left.
openssl s_client -connect example.com:443 -servername example.com \
  -verify 5 -verify_return_error -verify_hostname example.com 2>&1 \
  | grep -E "(Verify return|notAfter)"
```

### Generate a fingerprint pin (for HPKP — deprecated; or for cert pinning in apps)

```bash
openssl x509 -in cert.pem -noout -pubkey \
  | openssl pkey -pubin -outform DER \
  | openssl dgst -sha256 -binary \
  | openssl base64
# Use as the SHA-256 pin.
```

### CT log inclusion check

```bash
openssl s_client -connect example.com:443 -servername example.com -ct 2>/dev/null \
  | grep -A4 "SCTs"
```

## Tips

- **Always** specify `-servername` when using `s_client`. Hosts without it appear deprecated.
- **Always** specify `-sha256` (or stronger) on signing commands. `-sha1` defaults are still in some 1.x boxes.
- **Always** use SAN. CN-only is a 2010 idea.
- **Never** ship `.key` files in git. Combine `.gitignore` patterns: `*.key`, `*.pem`, `*.p12`, `*.pfx` plus repo-wide pre-commit hook running `git secrets`.
- **Never** put a passphrase on the command line in production. Use `-pass file:/path` or `-pass env:VAR`.
- **Never** generate DH params at runtime. Use ffdhe2048/3072 named groups (3.x default).
- **Never** write your own ASN.1 parsing. `openssl asn1parse -in foo` is the debugger.
- Use `umask 077` before generating any private key file.
- When in doubt about format, run `head -c 200 file | xxd` — PEM starts `2D 2D 2D 2D 2D 42 45 47 49 4E` (`-----BEGIN`).
- `openssl asn1parse -inform DER -in file.der` to inspect DER without writing PEM.
- For batch certificate analysis, prefer `cryptography` (Python) or `crypto/x509` (Go); `openssl x509 -text` is human-friendly but hard to parse.
- `openssl s_time -connect host:443 -new -time 5` benchmarks new-handshakes-per-second.
- `openssl speed` benchmarks the local crypto.
- The `openssl pkeyutl` family (sign/verify/encrypt/decrypt/derive) replaces the old `rsautl`/`dgst -sign` for low-level operations.
- macOS Keychain Access can import `.p12` files but expects PBE-SHA1-3DES — use `-legacy` on export.
- `openssl rand` is the safest portable random source for shell scripts; better than `$RANDOM` (16-bit, predictable).
- For TLS load testing, prefer `tlsfuzzer`, `testssl.sh`, `sslyze` over rolling your own with `s_client`.

## See Also

- ssh, gpg, tls, dns, polyglot

## References

- `man openssl` — top-level commands index
- `man openssl-req`, `man openssl-x509`, `man openssl-s_client`, `man openssl-genpkey`, `man openssl-pkey`, `man openssl-pkcs12`, `man openssl-ca`, `man openssl-dgst`, `man openssl-enc`, `man openssl-rand`, `man openssl-ts`, `man openssl-ocsp`, `man openssl-crl`, `man openssl-verify`, `man openssl-ciphers`, `man openssl-pkcs8`, `man openssl-pkcs7`, `man openssl-pkeyutl`, `man openssl-list` — per-subcommand
- `man config(5ssl)` — openssl.cnf format
- `man x509v3_config(5ssl)` — extension config syntax
- [OpenSSL Documentation](https://www.openssl.org/docs/) — official
- [OpenSSL Man Pages (3.x)](https://docs.openssl.org/master/man1/) — current
- [OpenSSL Wiki](https://wiki.openssl.org/) — community recipes
- [Mozilla SSL Configuration Generator](https://ssl-config.mozilla.org/) — Modern/Intermediate/Old presets for nginx, apache, haproxy, AWS ALB
- [Mozilla Server-Side TLS](https://wiki.mozilla.org/Security/Server_Side_TLS) — rationale behind the presets
- [SSL Labs Server Test](https://www.ssllabs.com/ssltest/) — A+ rating checklist
- [testssl.sh](https://testssl.sh/) — comprehensive TLS scanner
- [sslyze](https://github.com/nabla-c0d3/sslyze) — fast Python TLS scanner
- [Bulletproof TLS and PKI](https://www.feistyduck.com/books/bulletproof-tls-and-pki/) — Ivan Ristić, the canonical TLS theory book
- [OpenSSL Cookbook](https://www.feistyduck.com/library/openssl-cookbook/) — Ivan Ristić, free
- [Smallstep CA Tooling](https://smallstep.com/docs/step-ca/) — modern CA alternative to `openssl ca`
- [RFC 5280](https://www.rfc-editor.org/rfc/rfc5280) — X.509 PKI Certificate and CRL Profile
- [RFC 5246](https://www.rfc-editor.org/rfc/rfc5246) — TLS 1.2
- [RFC 8446](https://www.rfc-editor.org/rfc/rfc8446) — TLS 1.3
- [RFC 6960](https://www.rfc-editor.org/rfc/rfc6960) — OCSP
- [RFC 3161](https://www.rfc-editor.org/rfc/rfc3161) — RFC 3161 Time-Stamp Protocol
- [RFC 8555](https://www.rfc-editor.org/rfc/rfc8555) — ACME (Let's Encrypt protocol)
- [RFC 7468](https://www.rfc-editor.org/rfc/rfc7468) — PEM textual encoding
- [RFC 5208](https://www.rfc-editor.org/rfc/rfc5208) — PKCS#8
- [RFC 7292](https://www.rfc-editor.org/rfc/rfc7292) — PKCS#12
- [RFC 8017](https://www.rfc-editor.org/rfc/rfc8017) — PKCS#1 v2.2 (RSA)
- [RFC 8032](https://www.rfc-editor.org/rfc/rfc8032) — Ed25519 / Ed448
- [RFC 7748](https://www.rfc-editor.org/rfc/rfc7748) — X25519 / X448
- [RFC 7919](https://www.rfc-editor.org/rfc/rfc7919) — Negotiated Finite Field Diffie-Hellman (ffdhe groups)
- [RFC 7633](https://www.rfc-editor.org/rfc/rfc7633) — TLS Feature (must-staple) extension
- [LibreSSL](https://www.libressl.org/) — OpenBSD fork
- [BoringSSL](https://boringssl.googlesource.com/boringssl/) — Google fork
- [Arch Wiki — OpenSSL](https://wiki.archlinux.org/title/OpenSSL)
