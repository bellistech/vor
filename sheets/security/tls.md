# TLS (Transport Layer Security)

> The cryptographic protocol that secures HTTPS, SMTP-over-TLS, IMAPS, FTPS, MQTT-TLS, gRPC, and most modern client/server traffic on the public Internet.

## Setup

TLS = Transport Layer Security, the protocol. SSL (Secure Sockets Layer) = its predecessor (deprecated; everything called "SSL" since ~1999 is actually TLS, but the name stuck for marketing). Provides three guarantees over a TCP-like (or QUIC for TLS 1.3 over UDP) byte stream: confidentiality (no eavesdropping), integrity (no tampering), and authentication (you reached the server you intended).

Version timeline at a glance:

```bash
# Dead / banned in browsers and almost all servers:
SSL 1.0          # Internal at Netscape, never released — known broken
SSL 2.0 (1995)   # RFC 6176 prohibits — banned
SSL 3.0 (1996)   # POODLE killed it (2014) — banned in browsers since 2015

# End-of-life (still in some embedded/legacy):
TLS 1.0 (1999)   # RFC 2246 — EOL March 2020 (PCI-DSS dropped it 2018)
TLS 1.1 (2006)   # RFC 4346 — EOL March 2020

# Live:
TLS 1.2 (2008)   # RFC 5246 — still common, secure when configured well
TLS 1.3 (2018)   # RFC 8446 — modern preferred; 1-RTT handshake; 0-RTT optional
```

Quick test (does the server answer TLS at all?):

```bash
openssl s_client -connect example.com:443 -servername example.com </dev/null
# Look at the "Protocol :" line (TLSv1.2 or TLSv1.3) and the "Cipher :" line
```

Quick negotiate-down (what is the *minimum* the server still accepts?):

```bash
openssl s_client -connect example.com:443 -servername example.com -tls1_1 </dev/null 2>&1 | grep -E '^Protocol|^Cipher|alert'
# A server that returns "alert protocol version" for tls1_1 is correctly hardened.
```

What the rest of this sheet covers: the byte-level handshake of TLS 1.2 and 1.3, every cipher-suite component, certificate-chain validation, SNI/ALPN/OCSP/CT/CAA, mTLS, ACME and Let's Encrypt, every common error message and fix, and the openssl/testssl.sh diagnostic toolkit.

## Protocol Versions

### SSL 2.0 (1995)

Wire-incompatible with everything modern. RFC 6176 (2011) explicitly prohibits SSL 2.0. Banned because:

- No protection against truncation attacks (the TCP FIN equivalent in the cipher stream).
- Same key for MAC and encryption.
- Vulnerable to cipher-suite rollback (an MITM forces the weakest agreed cipher).
- Vulnerable to DROWN (2016) when a server even accepts SSLv2 — that alone leaks TLS 1.2 RSA sessions.

How to detect it on a server (you should never see this in 2026):

```bash
openssl s_client -connect host:443 -servername host -ssl2 </dev/null
# Modern OpenSSL won't even compile SSLv2 in. If the server still speaks it,
# use a docker image of an older openssl to test:
docker run --rm -it instrumentisto/openssl:1.0.2 \
  s_client -connect host:443 -ssl2 </dev/null
```

### SSL 3.0 (1996)

Killed by POODLE (Padding Oracle On Downgraded Legacy Encryption, 2014). The CBC padding scheme leaks one byte of plaintext per ~256 requests when an attacker can force protocol downgrade. Browsers disabled SSL 3.0 across 2014-2015. RFC 7568 (2015) "Deprecating SSLv3" is unambiguous: do not use.

Detect:

```bash
openssl s_client -connect host:443 -servername host -ssl3 </dev/null
# Expected on a hardened server: "ssl handshake failure" or "no ssl/tls supported"
```

### TLS 1.0 (RFC 2246, 1999)

The renaming of SSL 3.1 to "TLS" so Microsoft would adopt it (Netscape and IETF politics). EOL since March 2020 by mainstream browsers. PCI-DSS dropped it in June 2018. Vulnerable to BEAST (CBC IV predictability) without a TLS 1.1+ workaround. Still seen on:

- Embedded/IoT devices with frozen firmware
- Old industrial control systems
- Legacy enterprise mainframes

Detect:

```bash
openssl s_client -connect host:443 -servername host -tls1 </dev/null 2>&1 | head -20
```

### TLS 1.1 (RFC 4346, 2006)

Fixed BEAST by using explicit IVs. Otherwise marginal improvements over 1.0. EOL same date as TLS 1.0 (March 2020). Almost no software intentionally targets 1.1 — devices either support 1.0+1.1 (legacy) or 1.2+1.3 (modern).

### TLS 1.2 (RFC 5246, 2008)

Still the most-deployed version because:

- Adds AEAD ciphers (GCM and CCM modes).
- Adds SHA-256 / SHA-384 in PRF and signature algorithms.
- Lets the client and server independently signal preferred hash and signature combos via the `signature_algorithms` extension.
- All browsers, every cloud LB, every modern OpenSSL/BoringSSL/LibreSSL/wolfSSL/mbedTLS supports it.

Secure when configured with: ECDHE key exchange, AEAD cipher (GCM or CHACHA20-POLY1305), and TLS 1.2's RSA key transport disabled.

### TLS 1.3 (RFC 8446, 2018)

Ground-up redesign. Drops every legacy primitive:

- No RSA key transport — only (EC)DHE, so all sessions are forward-secret.
- No CBC, no RC4, no 3DES, no MD5, no SHA-1.
- Only AEAD ciphers.
- 1-RTT handshake by default; 0-RTT optional with PSK.
- Encrypted handshake from `EncryptedExtensions` onward (server identity is hidden from passive observers, modulo SNI).
- Removed compression (CRIME/BREACH won), removed renegotiation (replaced by post-handshake auth).

### Negotiation: who picks the version?

The client offers a max version in `ClientHello.legacy_version` (always `TLS 1.2` for compatibility) and the *real* max version in the `supported_versions` extension. The server picks the highest both support and echoes it in `ServerHello.supported_versions`.

```bash
# Force a specific version client-side:
openssl s_client -connect host:443 -servername host -tls1_2 </dev/null
openssl s_client -connect host:443 -servername host -tls1_3 </dev/null

# What versions does this server even offer?
nmap --script ssl-enum-ciphers -p 443 host
```

## TLS 1.2 Handshake — Step by Step

The full ECDHE-RSA TLS 1.2 handshake (the "modern" 1.2 case). Time ordering left-to-right; arrows show direction.

```
Client                                                 Server
  |                                                       |
  |---- 1. ClientHello -------------------------------->  |
  |                                                       |
  |<--- 2. ServerHello ------------------------------------|
  |<--- 3. Certificate ------------------------------------|
  |<--- 4. ServerKeyExchange ------------------------------|
  |<--- 5. CertificateRequest [optional, mTLS] -----------|
  |<--- 6. ServerHelloDone --------------------------------|
  |                                                       |
  |---- 7. Certificate [optional, mTLS] ---------------->  |
  |---- 8. ClientKeyExchange --------------------------->  |
  |---- 9. CertificateVerify [optional, mTLS] --------->  |
  |----10. ChangeCipherSpec ---------------------------->  |
  |----11. Finished ------------------------------------>  |
  |                                                       |
  |<---12. ChangeCipherSpec ------------------------------|
  |<---13. Finished --------------------------------------|
  |                                                       |
  |==== Application Data (encrypted, AEAD) =============  |
```

### Step 1: ClientHello

The client sends:

- `legacy_version` = `0x0303` (TLS 1.2, byte literal).
- `random`: 32 bytes of cryptographic randomness (nonce; first 4 bytes used to be a timestamp by convention, modern implementations randomize all 32).
- `legacy_session_id`: 0–32 bytes for session resumption (see TLS 1.2 session-id mechanism).
- `cipher_suites`: ordered list of supported ciphers (typically 10–30).
- `legacy_compression_methods`: must contain only `null` (`0x00`); compression killed by CRIME.
- `extensions`: ordered list. Common ones:
  - `server_name` (SNI) — the requested hostname.
  - `supported_groups` — curves the client can do ECDHE on (`x25519`, `secp256r1`, `secp384r1`, `secp521r1`).
  - `ec_point_formats` — usually just `uncompressed`.
  - `signature_algorithms` — what {hash, signature} pairs the client accepts on the cert.
  - `application_layer_protocol_negotiation` (ALPN) — `h2`, `http/1.1`.
  - `status_request` — request OCSP stapling.
  - `signed_certificate_timestamp` — request CT proofs.
  - `renegotiation_info` — secure-renegotiation indicator.

Inspect raw bytes with:

```bash
openssl s_client -connect host:443 -servername host -msg -trace </dev/null 2>&1 | head -100
```

### Step 2: ServerHello

The server picks one cipher suite and one compression method (must be `null`), echoes back its own 32-byte `random`, and possibly a session-id. Includes its own extensions chosen from the client's offered set.

Critical: the server MUST pick from the client's offered cipher list — it cannot invent one. If no overlap, the server returns `handshake_failure` alert.

### Step 3: Certificate

The server sends an `ASN.1 DER`-encoded certificate chain in order: leaf first, then intermediates. The root MAY be omitted (and SHOULD be, since the client must already trust it independently).

Inspect:

```bash
openssl s_client -connect host:443 -servername host -showcerts </dev/null
# Look for "-----BEGIN CERTIFICATE-----" blocks; first is leaf, rest intermediates.
```

### Step 4: ServerKeyExchange

Sent only if the chosen cipher needs additional key material the certificate doesn't carry. For ECDHE: the server picks an ephemeral EC keypair, sends the public key, and signs `ClientHello.random || ServerHello.random || params` with the private key from its certificate (the `signature_algorithms` extension dictates which {hash, signature} pair).

Not sent for static-RSA key exchange (which TLS 1.3 removed entirely).

### Step 5: CertificateRequest (optional)

Used for mutual TLS (mTLS). The server tells the client "I want your cert too, and here are the CAs I'll trust for it."

### Step 6: ServerHelloDone

Empty handshake message — "I'm done sending; your turn." Just a signal.

### Step 7: Certificate (client, optional)

Only if mTLS was requested. Client sends its certificate chain.

### Step 8: ClientKeyExchange

For ECDHE: client sends *its* ephemeral EC public key. Both sides now compute the shared secret via ECDH:

```
shared_secret = ECDH(client_ephemeral_priv, server_ephemeral_pub)
              = ECDH(server_ephemeral_priv, client_ephemeral_pub)
```

For static RSA (legacy): client generates a 48-byte `pre_master_secret`, encrypts it under the server's RSA public key from the cert. Server decrypts with its long-term RSA private key. **No forward secrecy** — if the server's RSA private key ever leaks, every captured past session is decryptable.

### Step 9: CertificateVerify (optional)

mTLS only: client signs the handshake transcript so far with its private key, proving possession of the key matching the cert it just sent.

### Step 10: ChangeCipherSpec (client)

A 1-byte message that says "everything I send after this is encrypted with the negotiated keys." Not technically part of the handshake protocol — it's a separate "ChangeCipherSpec" sub-protocol — but it sits in the same byte stream.

The pre_master_secret + both randoms feed into the PRF (Pseudo-Random Function) to derive:

- `client_write_MAC_key` (for non-AEAD ciphers)
- `server_write_MAC_key` (for non-AEAD ciphers)
- `client_write_key`
- `server_write_key`
- `client_write_IV`
- `server_write_IV`

### Step 11: Finished (client)

The first encrypted message. Contains an HMAC of all preceding handshake messages keyed with the master secret. Proves to the server that the client computed the same keys.

### Step 12: ChangeCipherSpec (server)

Server switches to encryption.

### Step 13: Finished (server)

Server's HMAC over the handshake including the client's Finished. Both sides have now proven they agree on every byte. If either Finished MAC fails, the connection aborts with `decrypt_error`.

Application data flows after step 13. Total: 2 round trips before any HTTP request is sent.

### Why the order matters

If the attacker tampered with any handshake byte, the Finished MAC won't match and the connection aborts. Hence "TLS handshake authenticates the entire prior handshake retroactively." This is why `triple-handshake` and similar attacks before the `extended_master_secret` extension (RFC 7627) were so dangerous.

## TLS 1.3 Handshake — The Simplification

Goals: 1-RTT (one round trip from `ClientHello` to first application data), encrypt as much of the handshake as possible, drop legacy ciphers, simplify the state machine.

```
Client                                                    Server
  |                                                          |
  |---- 1. ClientHello ----------------------------------->  |
  |     + key_share         (precomputed for likely curves)  |
  |     + signature_algorithms                               |
  |     + supported_versions (TLS 1.3)                       |
  |     + psk_key_exchange_modes [if resuming]               |
  |     + pre_shared_key   [if resuming, last extension]     |
  |                                                          |
  |<---- 2. ServerHello -----------------------------------|
  |     + key_share                                        |
  |     [from here, all messages encrypted]               |
  |<---- 3. EncryptedExtensions ---------------------------|
  |<---- 4. CertificateRequest [optional, mTLS] -----------|
  |<---- 5. Certificate -----------------------------------|
  |<---- 6. CertificateVerify -----------------------------|
  |<---- 7. Finished --------------------------------------|
  |                                                          |
  |---- 8. Certificate [optional, mTLS] ----------------->  |
  |---- 9. CertificateVerify [optional, mTLS] ----------->  |
  |----10. Finished ------------------------------------->  |
  |                                                          |
  |===== Application Data (encrypted, AEAD) ===============  |
```

### Step 1: ClientHello (TLS 1.3 version)

Client preemptively sends one or more `key_share` entries. Each entry = "for group X, here is my ephemeral public key." The client guesses which curve(s) the server will pick. Defaults: `x25519` and `secp256r1` together is ~99% coverage.

If the server picks a group the client did NOT prepopulate, it returns `HelloRetryRequest` (HRR), and the client tries again with the right share — costing an extra round trip. Hence "preemptively send shares for likely curves."

`legacy_version` is still `0x0303` (TLS 1.2) for middlebox compatibility. The actual version is in the `supported_versions` extension.

### Step 2: ServerHello

Server picks one cipher suite (only AEAD options in 1.3 — see below), picks one of the offered key_shares, and sends its own ephemeral public key for that group. From this point both sides can derive `handshake_traffic_secret` and switch to encrypting *every subsequent message*.

### Step 3: EncryptedExtensions

The server's chosen extensions (ALPN, server_name acknowledgement, max_fragment_length, etc.) — moved out of `ServerHello` because they were now sensitive enough to encrypt.

### Step 4: CertificateRequest (optional)

mTLS, same as 1.2 but moved into the encrypted phase.

### Step 5: Certificate

The server's X.509 chain, encrypted.

### Step 6: CertificateVerify

Server signs the handshake transcript hash with the private key matching its cert. Replaces the implicit signing-via-ServerKeyExchange of TLS 1.2.

### Step 7: Finished (server)

HMAC over the transcript, keyed with `server_finished_key`. Closes the server's flight.

### Step 8-9: Client Certificate / CertificateVerify (optional, mTLS)

### Step 10: Finished (client)

Done. The client can actually piggyback application data with this Finished — that's the canonical "1-RTT".

### 0-RTT (Early Data)

If the client has a previously-saved PSK (pre-shared key) from a prior session ticket, it can:

```
ClientHello
  + key_share
  + pre_shared_key (with binder)
  + early_data extension
[encrypted with PSK-derived early_traffic_secret]
HTTP request (Early Data)
```

The server can read and act on the early data *before* completing the handshake. Saves a round trip but:

- **Replayable.** A passive attacker can capture the early data and replay it. The server cannot tell a fresh request from a replay.
- Only safe for idempotent requests. POSTs and anything stateful must NOT be sent as 0-RTT.
- Browsers/HTTPx clients gate 0-RTT to GET requests by default.

To opt out server-side: nginx `ssl_early_data off;` (this is the default).

## Cipher Suites — Anatomy

A cipher suite specifies four primitives bundled together. Wire format is a 2-byte identifier; humans read the mnemonic name.

### TLS 1.2 cipher-suite name format

```
TLS_<KX>_<AUTH>_WITH_<BULK>_<MAC>

         |------------------|         |------------|
         key exchange + auth          symmetric+MAC
```

Example breakdown:

```
TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
└── TLS              fixed prefix
    └── ECDHE        key exchange = Elliptic-Curve Diffie-Hellman Ephemeral (forward secret)
        └── RSA      signature on the cert / ServerKeyExchange = RSA
            └── AES_256_GCM   bulk cipher = AES-256 in GCM mode (AEAD)
                  └── SHA384  PRF + Finished MAC hash; for AEAD this is NOT the bulk MAC
                              (GCM is its own MAC) but is still used in PRF and HKDF
```

### KX — Key Exchange

| Code | Meaning | Forward Secret? | Status |
|---|---|---|---|
| `RSA` | Static RSA key transport (client encrypts pre-master to server pubkey) | NO | Banned in TLS 1.3, avoid in 1.2 |
| `DHE` | Diffie-Hellman Ephemeral over finite-field group | YES | Acceptable but slower than ECDHE |
| `ECDHE` | Elliptic-Curve Diffie-Hellman Ephemeral | YES | Modern default |
| `DH` | Static Diffie-Hellman (cert binds DH public key) | NO | Effectively unused |
| `ECDH` | Static Elliptic-Curve DH | NO | Effectively unused |
| `PSK` | Pre-shared key | YES (with -DHE variant) | Common in IoT |
| `ECDHE_PSK` | ECDHE plus PSK | YES | Common in resumption |

### AUTH — Authentication / Signature

| Code | Meaning | Notes |
|---|---|---|
| `RSA` | RSA signature on cert (and optionally ServerKeyExchange) | Most common |
| `ECDSA` | Elliptic-Curve Digital Signature Algorithm | Smaller certs, faster verify |
| `EdDSA` | Ed25519 signatures | TLS 1.3+ only (RFC 8422) |
| `DSS` | DSA signatures | Effectively dead; DSA keys obsolete |

### BULK — Symmetric Cipher

| Code | Mode | AEAD? | Notes |
|---|---|---|---|
| `AES_128_GCM` | Galois/Counter Mode | YES | Most common, hardware-accelerated (AES-NI) |
| `AES_256_GCM` | Galois/Counter Mode | YES | Higher security margin, slightly slower |
| `CHACHA20_POLY1305` | ChaCha20 stream + Poly1305 MAC | YES | No AES-NI dependency; fast on mobile/ARM |
| `AES_128_CCM` | Counter with CBC-MAC | YES | Common in IoT / DTLS |
| `AES_128_CCM_8` | CCM with 8-byte tag | YES | Even more constrained IoT |
| `AES_128_CBC` | Cipher Block Chaining + separate HMAC | NO | LEGACY — padding-oracle attacks (Lucky13, BEAST) |
| `AES_256_CBC` | CBC + HMAC | NO | LEGACY |
| `3DES_EDE_CBC` | Triple-DES | NO | DEAD — Sweet32 birthday attack |
| `RC4_128` | Stream cipher | NO | DEAD — biased keystream |

### MAC — Message Authentication Code (only relevant for non-AEAD)

| Code | Hash | Notes |
|---|---|---|
| `SHA256` | SHA-2/256 | OK |
| `SHA384` | SHA-2/384 | OK; required for some certs |
| `SHA` | SHA-1 | DEPRECATED for new certs and signatures |
| `MD5` | MD5 | DEAD |

For AEAD ciphers (GCM, CCM, ChaCha20-Poly1305), the MAC is integrated into the cipher; the suite name's hash field is used in the PRF and HKDF, not as an HMAC over the record.

### TLS 1.3 cipher suites — the entire (minimal) list

TLS 1.3 simplified to AEAD-only and removed the KX and AUTH from the suite name (those are negotiated separately via `supported_groups` + `signature_algorithms`).

```
TLS_AES_128_GCM_SHA256          (mandatory-to-implement)
TLS_AES_256_GCM_SHA384
TLS_CHACHA20_POLY1305_SHA256
TLS_AES_128_CCM_SHA256          (rare, IoT-flavored)
TLS_AES_128_CCM_8_SHA256        (rare, IoT-flavored)
```

That's it. Five total. Everything else (curve, signature) is negotiated via separate extensions.

### Reading what was actually negotiated

```bash
openssl s_client -connect host:443 -servername host </dev/null 2>&1 | grep -E '^(Protocol|Cipher|Server Temp Key|Peer signing digest|Peer signature type)'
# Example output:
#   Protocol  : TLSv1.3
#   Cipher    : TLS_AES_256_GCM_SHA384
#   Server Temp Key: X25519, 253 bits
#   Peer signing digest: SHA256
#   Peer signature type: ECDSA
```

### What you'd see for old TLS 1.2 ECDHE-RSA-AES256-GCM:

```
Protocol  : TLSv1.2
Cipher    : ECDHE-RSA-AES256-GCM-SHA384
Server Temp Key: X25519, 253 bits
```

Note OpenSSL prints TLS 1.2 suites in the dashed form (`ECDHE-RSA-AES256-GCM-SHA384`) but writes them in IANA form (`TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384`) in the IANA registry — same thing, two notations.

## Forward Secrecy

The property: if the server's long-term private key is later compromised, recorded past sessions remain undecryptable. The eavesdropper has the recorded ciphertext + the private key, but the per-session key is gone.

How it's achieved: ephemeral key exchange. The session key is derived from a Diffie-Hellman exchange where both parties generate fresh keypairs per connection and discard them at the end.

### The killer comparison

| Scheme | Forward Secret? | Why |
|---|---|---|
| `RSA` key transport (TLS 1.2 only) | NO | The pre_master_secret is encrypted under the server's long-term RSA pubkey. With the cert's RSA private key, decrypt the past session's pre_master and derive every key. |
| `DHE_RSA` | YES | DH params are ephemeral; private key signs the params for authentication only. |
| `ECDHE_RSA` | YES | Same idea, EC for speed. |
| `ECDHE_ECDSA` | YES | Same, ECDSA for the signature. |
| TLS 1.3 (any suite) | YES (mandatory) | Static RSA key transport is removed; (EC)DHE always used. |

### Verifying forward secrecy is on

```bash
# Look at the negotiated cipher; if it begins with ECDHE or DHE, you're forward-secret.
openssl s_client -connect host:443 -servername host </dev/null 2>&1 | grep '^Cipher'

# To force *non*-forward-secret (RSA-only) and confirm the server refuses:
openssl s_client -connect host:443 -servername host -tls1_2 -cipher 'AES256-SHA' </dev/null
# Expected: handshake_failure on a hardened server.
```

### Why static RSA was banned in TLS 1.3

Beyond the lack of forward secrecy: the RSA key-transport mechanism is also vulnerable to the Bleichenbacher 1998 attack and its 2017 sequel ROBOT. Even if you patch the padding oracle, removing the entire mechanism is the engineering win.

## Authenticated Encryption (AEAD)

AEAD = Authenticated Encryption with Associated Data. One algorithm produces ciphertext + integrity tag in a single pass. Replaces the older "encrypt-then-MAC" or "MAC-then-encrypt" CBC+HMAC compositions, which were the source of an entire generation of padding-oracle attacks (Lucky13, POODLE).

### The three TLS AEADs

#### AES-GCM

- AES (128 or 256 key) in counter mode, with a Galois-field MAC over ciphertext + AAD.
- Hardware acceleration on modern x86 (AES-NI) and ARM (Cryptography Extensions).
- 96-bit nonce (in TLS, derived deterministically from the sequence number XORed with a per-direction salt).
- Tag length: 128 bits.
- Pitfall: nonce reuse with the same key catastrophically breaks it. TLS's nonce derivation prevents reuse within a session; outside TLS, never reuse.

#### ChaCha20-Poly1305

- ChaCha20 stream cipher (RFC 8439) + Poly1305 MAC.
- Pure software, no hardware acceleration needed; faster than AES on ARM cores without crypto extensions.
- Originally designed by Google and DJB; adopted because it gave mobile clients without AES-NI a non-CBC option.
- Same nonce-reuse caveat.

#### AES-CCM

- AES in Counter mode with CBC-MAC.
- Two variants: 128-bit tag (`CCM`) and 64-bit tag (`CCM_8`).
- Mostly for IoT (smaller code footprint than GCM).
- Slightly slower than GCM in software due to two AES passes.

### Why AEAD wins over CBC+HMAC

- Single pass — no leaking timing differences between "decrypt fails" and "MAC fails."
- No padding — no padding oracle.
- Cleaner key schedule.
- The MAC covers the AAD (e.g., the TLS record header), so attacker can't tamper with metadata.

### Visualizing a TLS 1.3 record (AEAD)

```
+-----------------+----------------------------+--------+
| Record header   | Ciphertext (plaintext +    | Auth   |
| (5 bytes,       | inner type + padding,      | Tag    |
| AAD)            | AEAD-encrypted)            | 16B    |
+-----------------+----------------------------+--------+
type=0x17 (app data, opaque)
version=0x0303 (legacy)
length=...
```

The record's `type` field on the wire is always `0x17` ("application_data") in TLS 1.3 — the *real* type (handshake, alert, app data) is in the inner plaintext, after decryption. This hides the handshake/data boundary from passive observers.

## Key Exchange — RSA vs ECDHE vs DHE

### Static RSA key transport (TLS 1.2 only, banned in 1.3)

Wire flow:

1. Server cert contains `pubkey_RSA`.
2. Client picks 48-byte random `pre_master_secret`.
3. Client computes `enc = RSA-PKCS1-v1.5-encrypt(pubkey_RSA, pre_master_secret)`.
4. Client sends `enc` as `ClientKeyExchange.exchange_keys`.
5. Server decrypts with its long-term RSA private key.
6. Both derive the master_secret and traffic keys.

Why it's bad:

- No forward secrecy.
- PKCS#1 v1.5 padding is the source of Bleichenbacher / ROBOT.
- Inflicts CPU cost on the server (RSA decrypt is expensive).

### DHE — Diffie-Hellman Ephemeral

Wire flow:

1. Server picks ephemeral DH keypair `(p, g, x_s, g^x_s mod p)`.
2. Server signs `(p, g, g^x_s)` with cert's private key. Sends in `ServerKeyExchange`.
3. Client picks ephemeral `x_c`, computes `g^x_c mod p`. Sends in `ClientKeyExchange`.
4. Both compute shared `g^(x_s * x_c) mod p`.

Bad params (small `p`, weak `g`) lead to LOGJAM. Use ≥2048-bit groups; RFC 7919 defines fixed groups (`ffdhe2048`, `ffdhe3072`, etc.) clients can validate without trusting the server's choice.

### ECDHE — Elliptic-Curve Diffie-Hellman Ephemeral

Same idea, EC group. Common groups:

- `x25519` — Curve25519 in Edwards form (RFC 7748). 128-bit security. Modern default.
- `secp256r1` (a.k.a. P-256 / `prime256v1`) — NIST curve, 128-bit security.
- `secp384r1` — P-384, 192-bit security.
- `secp521r1` — P-521, 256-bit security.
- `x448` — Curve448, 224-bit security.

Why ECDHE beat DHE:

- 256-bit EC keys ≈ 3072-bit RSA/DH security. Smaller wire size, faster math.
- `x25519` is constant-time and side-channel-resistant by construction.

### What the server picks

The server picks from the client's `supported_groups` extension, then picks the highest-priority one it supports. Usually clients send `[x25519, secp256r1, x448, secp521r1, secp384r1]`.

Inspect:

```bash
openssl s_client -connect host:443 -servername host </dev/null 2>&1 | grep 'Server Temp Key'
# Output like:  Server Temp Key: X25519, 253 bits
```

## Authentication — Certificates and Signatures

The handshake establishes a shared key, but it doesn't tell you who the peer is. Authentication is bolted on via X.509 certificates signed by a trusted CA.

### X.509 v3 fields that matter

```
Version:          3
Serial Number:    issuer-unique
Signature Alg:    e.g. sha256WithRSAEncryption, ecdsa-with-SHA384
Issuer:           CN=Let's Encrypt R3, O=Let's Encrypt, C=US
Validity:         notBefore + notAfter
Subject:          CN=example.com (deprecated for hostname binding)
Subject Public Key Info:
                  algorithm + bytes
Extensions:
  Subject Alternative Name (SAN):  DNS:example.com, DNS:www.example.com
  Basic Constraints:               CA:FALSE  (for end-entity)
  Key Usage:                       digitalSignature, keyEncipherment
  Extended Key Usage:              TLS Web Server Authentication
  Authority Key Identifier:        keyid:...
  Subject Key Identifier:          ...
  CRL Distribution Points:         URI:http://...crl
  Authority Information Access:    OCSP - URI:http://...ocsp
                                   CA Issuers - URI:http://...crt
  CT Precertificate SCTs:          [embedded SCT bytes]
Signature:        bytes signed by issuer's private key over the TBSCertificate
```

### Signature algorithms

Modern (preferred):

- `RSA-PSS` — Probabilistic Signature Scheme (RFC 8017). Provably secure padding.
- `ECDSA` with P-256, P-384, P-521.
- `Ed25519`, `Ed448` — EdDSA (RFC 8032). Deterministic, fast.

Legacy still common:

- `RSA-PKCS1-v1.5` — older padding, supported in cert chains for compatibility, not for TLS 1.3 handshakes (signature_algorithms forbids it for TLS 1.3 handshake signatures, but it's still allowed in cert chain signatures).

Dead:

- `MD5withRSA` — collision-broken.
- `SHA1withRSA` / `SHA1withECDSA` — collision-broken; major browsers stopped accepting in 2017.

Inspect:

```bash
openssl x509 -in cert.pem -noout -text | grep -E 'Signature Algorithm|Public Key Algorithm'
```

### CertificateVerify (TLS 1.3) and what it proves

The server's `CertificateVerify` is a signature over the transcript hash, made with the private key matching the cert's public key. This proves the server actually has the private key — the cert alone, without this signature, doesn't prove anything (it's just public bytes anyone could replay).

In TLS 1.2, the equivalent proof is implicit: `ServerKeyExchange` is signed in ECDHE/DHE suites, or `ClientKeyExchange` round-trips against the cert's pubkey in static-RSA.

## Certificate Chain Construction

### The three layers

```
Root CA cert        (self-signed, in client trust store)
   |  signs
   v
Intermediate cert   (issued by Root, signs leaf certs)
   |  signs
   v
Leaf cert           (issued for specific domain)
```

### What the server sends — and what it must NOT send

The server's `Certificate` message must contain:

- The **leaf** certificate (mandatory, first).
- Every **intermediate** between the leaf and a publicly trusted root.
- It MUST NOT include the **root** — clients already have it; sending it wastes ~2 KB and breaks if the client trusts a different root in a cross-signed chain.

### The "untrusted certificate" failure mode

The #1 cause of the dreaded "ERR_CERT_AUTHORITY_INVALID" / "self-signed certificate in certificate chain" / "unable to get local issuer certificate" is:

> The server has a valid leaf cert, but never installed (or never sends) the intermediate.

Browsers like Chrome and Firefox have AIA-fetching that can transparently download missing intermediates from the URL in the leaf's `Authority Information Access` extension — but `curl`, `openssl s_client`, Java HTTPS client, Go's TLS client, and many corporate proxies do NOT. So the site "works in Chrome but breaks in monitoring tools."

### Building a valid chain file

```bash
# Leaf and intermediate concatenated. NEVER include the root.
cat leaf.pem intermediate.pem > fullchain.pem

# In Let's Encrypt's certbot output:
ls /etc/letsencrypt/live/example.com/
# cert.pem        leaf only
# chain.pem       intermediate only
# fullchain.pem   leaf + intermediate (use this for nginx/apache ssl_certificate)
# privkey.pem     private key
```

Common server config:

```nginx
# nginx
ssl_certificate     /etc/letsencrypt/live/example.com/fullchain.pem;
ssl_certificate_key /etc/letsencrypt/live/example.com/privkey.pem;
```

```apache
# Apache 2.4.8+
SSLCertificateFile     /etc/letsencrypt/live/example.com/fullchain.pem
SSLCertificateKeyFile  /etc/letsencrypt/live/example.com/privkey.pem
# Older Apache 2.x:
SSLCertificateFile     /etc/letsencrypt/live/example.com/cert.pem
SSLCertificateChainFile /etc/letsencrypt/live/example.com/chain.pem
```

### Diagnosing chain issues

```bash
# Show every cert the server sends:
openssl s_client -connect example.com:443 -servername example.com -showcerts </dev/null 2>/dev/null

# Count them:
openssl s_client -connect example.com:443 -servername example.com -showcerts </dev/null 2>/dev/null \
  | grep -c 'BEGIN CERTIFICATE'

# Verify chain against a CA bundle:
openssl s_client -connect example.com:443 -servername example.com -showcerts </dev/null 2>/dev/null \
  | openssl crl2pkcs7 -nocrl -certfile /dev/stdin \
  | openssl pkcs7 -print_certs -noout

# Online tool that catches the missing-intermediate problem:
#   https://www.ssllabs.com/ssltest/   (Qualys)
#   https://crt.sh/?q=example.com      (CT logs - find every cert ever)
#   https://decoder.link/sslchecker/example.com/443
```

## Certificate Validation

When a client receives the server's chain, it MUST do every step or the chain is invalid.

### 1. Build the path to a trusted root

Walk leaf → intermediate → ... → root. Stop when you hit a cert in the local trust store. If you can't reach one, it's untrusted.

### 2. Verify each signature

For every link `(child, parent)`:

- Verify `parent.signs(child)` using the parent's public key.
- The signature must use a non-deprecated algorithm (no SHA-1, no MD5).

### 3. Validity period

For every cert in the chain:

- `now >= notBefore` (cert isn't from the future)
- `now <= notAfter` (cert isn't expired)

Modern leaf certs are 90 days (Let's Encrypt) or up to 397 days (commercial CA, browser-imposed cap since Sep 2020). Apple shortened it further to 47 days starting 2027.

### 4. Hostname matching

The leaf cert's `Subject Alternative Name` (SAN) extension MUST contain the hostname being connected to. Modern browsers (since Chrome 58, 2017) **ignore the CN field entirely** — only SAN counts.

Wildcards: `*.example.com` matches `foo.example.com` but NOT `example.com` and NOT `bar.foo.example.com` (single label depth only).

```bash
# Inspect SANs:
openssl s_client -connect example.com:443 -servername example.com </dev/null 2>/dev/null \
  | openssl x509 -noout -ext subjectAltName

# Output:
#   X509v3 Subject Alternative Name:
#       DNS:example.com, DNS:www.example.com
```

### 5. Revocation check

Two mechanisms (both have weaknesses):

- **CRL** (Certificate Revocation List): periodic, signed list of revoked serials. Slow, large, often stale. Largely obsolete.
- **OCSP** (Online Certificate Status Protocol): real-time query to a CA-run responder. Online, but blocking and a privacy leak.
- **OCSP Stapling**: the server itself fetches the OCSP response and includes it in the handshake. Solves both issues (see OCSP Stapling section below).

Browsers default to **soft-fail** OCSP: if the responder is unreachable, treat the cert as valid. This means revocation can be effectively bypassed by an attacker who blocks the OCSP responder. Counter-measure: **OCSP must-staple** extension makes a cert require a stapled response; absence = fail.

### 6. Key Usage and Extended Key Usage

The leaf must have:

- `Key Usage: digitalSignature` (and `keyEncipherment` for static-RSA suites)
- `Extended Key Usage: serverAuth (1.3.6.1.5.5.7.3.1)` — for HTTPS servers
- `Extended Key Usage: clientAuth (1.3.6.1.5.5.7.3.2)` — for mTLS clients

Intermediates must have:

- `Basic Constraints: CA:TRUE`
- `Key Usage: keyCertSign, cRLSign`

### 7. Constraints on intermediates

`Name Constraints` extension may restrict an intermediate to issuing only for specific domains. `Path Length Constraint` caps how many further sub-CAs can chain from this one.

## SNI — Server Name Indication

Without SNI, a TLS server reachable on a single IP can only serve one certificate (because the server has to pick a cert before knowing which hostname the client wants — the request body is encrypted by then).

SNI fixes this: the client sends the desired hostname in plaintext as a `ClientHello` extension. The server reads it, picks the matching cert, and proceeds.

### Wire format

```
struct {
    NameType name_type;        // 0 = host_name (only defined value)
    HostName host_name;        // length-prefixed UTF-8 (typically ASCII)
} ServerName;

struct {
    ServerName server_name_list<1..2^16-1>;
} ServerNameList;
```

### Why it matters

- Enables shared-IP virtual hosting (multiple HTTPS sites behind one IP).
- The cloud / CDN business model (Cloudflare, Fastly, AWS CloudFront) relies on it.
- IPv4 exhaustion + the demand for HTTPS made SNI mandatory.

### The privacy problem

In TLS 1.2 and most TLS 1.3 deployments the SNI is **plaintext on the wire**. Anyone on the path (ISPs, employer, censor) sees `Host: example.com` even though everything else is encrypted.

### ECH (Encrypted ClientHello)

Encrypts the SNI (and most of the rest of the ClientHello) under a key the server publishes via DNS (`HTTPS` / `SVCB` records). State of deployment:

- Chrome and Firefox have it behind flags.
- Cloudflare implements it server-side.
- Most other servers and CDNs are years behind.

Test ECH support of a client:

```bash
# Cloudflare's ECH test:
curl -vsko /dev/null https://crypto.cloudflare.com/cdn-cgi/trace 2>&1 | grep -i ech
# 'sni=encrypted' in the trace body means ECH succeeded.
```

### Diagnosing missing SNI

```bash
# WITHOUT -servername, openssl sends no SNI; you may get the wrong cert (or the "default" cert).
openssl s_client -connect example.com:443 </dev/null 2>&1 | grep '^subject='
# Compare to:
openssl s_client -connect example.com:443 -servername example.com </dev/null 2>&1 | grep '^subject='
```

If the two differ, the server is virtual-hosting — always pass `-servername` when debugging.

## ALPN — Application-Layer Protocol Negotiation

ALPN lets the client and server agree on the post-TLS protocol (`http/1.1`, `h2`, `h3` — though `h3` is QUIC, separate transport). The client sends a list, the server picks one. Decided during the TLS handshake, so HTTP/2 doesn't need an extra upgrade.

### Wire format (extension)

```
opaque ProtocolName<1..2^8-1>;

struct {
    ProtocolName protocol_name_list<2..2^16-1>;
} ProtocolNameList;
```

Common values: `h2`, `http/1.1`, `imap`, `pop3`, `smtp`, `irc`, `dot` (DNS-over-TLS), `doq` (DNS-over-QUIC), `acme-tls/1` (ACME TLS-ALPN-01).

### Diagnosing

```bash
# Tell the server I prefer h2 then http/1.1:
openssl s_client -connect example.com:443 -servername example.com -alpn h2,http/1.1 </dev/null 2>&1 | grep -i 'alpn'
# Expected:
#   ALPN protocol: h2

# What if you ask for something the server doesn't speak?
openssl s_client -connect example.com:443 -servername example.com -alpn nonsense/1 </dev/null 2>&1 | grep -i 'alpn'
# Expected:
#   No ALPN negotiated
```

### nginx + ALPN for HTTP/2

```nginx
listen 443 ssl http2;          # The 'http2' keyword turns on ALPN h2.
ssl_certificate     ...;
ssl_certificate_key ...;
```

## OCSP — Online Certificate Status Protocol

When the client wants to know "is this leaf cert still valid, or has the CA revoked it?", it can send an OCSP request to the responder URL listed in the cert's `Authority Information Access` extension.

### Wire flow (without stapling)

```
Client -> OCSP Responder: "Status of cert serial 0xABCDEF, issued by CN=R3?"
OCSP Responder -> Client: "good" / "revoked" / "unknown" + signature + thisUpdate + nextUpdate
```

### Issues

1. **Latency**: extra round trip (DNS + TCP + HTTPS to responder).
2. **Privacy**: the responder learns which sites the user visits.
3. **Availability**: if the responder is down, fail-open (browser default = soft-fail, attack vector) or fail-closed (rare, breaks for users when CA infra has outages).

### Inspect a cert's OCSP URL

```bash
openssl x509 -in cert.pem -noout -ocsp_uri
# http://r3.o.lencr.org

# Make a manual OCSP query (verbose because OCSP is fiddly):
openssl ocsp -issuer chain.pem -cert cert.pem \
  -url http://r3.o.lencr.org \
  -header "Host=r3.o.lencr.org"
# Expected:
#   cert.pem: good
#   This Update: ...
#   Next Update: ...
```

### OCSP must-staple

A cert with the `tls-feature: status_request` extension declares "if I'm presented in a TLS handshake without a stapled OCSP response, the client MUST treat me as invalid." This converts soft-fail into hard-fail for that cert.

```bash
# Look for tls-feature OID 1.3.6.1.5.5.7.1.24 in the cert:
openssl x509 -in cert.pem -noout -text | grep -A1 'TLS Feature'
```

Drawback: if the server forgets to staple (config error), the cert is unusable until you fix it. Major browsers gradually backed off must-staple support after deployment pain.

## OCSP Stapling

The server pre-fetches its own OCSP response and includes ("staples") it in the handshake. The client trusts the staple if its signature checks out and the cert's serial matches.

### Wire format

The client sends `status_request` extension in `ClientHello`. The server includes the OCSP response in `CertificateStatus` in TLS 1.2, or in the `Certificate` message's per-entry extensions in TLS 1.3.

### Why it wins

- **No client-side OCSP request.** Saves a round trip.
- **No privacy leak.** The responder talks to the server, not to every visitor.
- **Hard-fail compatible.** If the server doesn't staple, the must-staple extension makes the cert unusable.

### Server config — nginx

```nginx
ssl_stapling           on;
ssl_stapling_verify    on;
resolver               1.1.1.1 8.8.8.8 valid=300s;     # nginx needs DNS to fetch OCSP
resolver_timeout       5s;

# Optional: pre-trusted issuer chain (speeds first staple)
ssl_trusted_certificate /etc/letsencrypt/live/example.com/chain.pem;
```

### Server config — apache

```apache
SSLUseStapling           on
SSLStaplingCache         "shmcb:/var/run/ocsp(128000)"
SSLStaplingResponderTimeout 5
SSLStaplingReturnResponderErrors off
```

### Server config — haproxy

```
global
    ssl-default-bind-options no-sslv3 no-tls-tickets
frontend https
    bind *:443 ssl crt /etc/haproxy/certs/example.com.pem
    # haproxy needs the staple response fetched externally and dropped at:
    #   /etc/haproxy/certs/example.com.pem.ocsp
```

### Verify that stapling is working

```bash
openssl s_client -connect example.com:443 -servername example.com -status </dev/null 2>&1 | grep -A 5 'OCSP response'
# Expected when stapling works:
#   OCSP response:
#   ======================================
#   OCSP Response Data:
#       OCSP Response Status: successful (0x0)
#       Response Type: Basic OCSP Response
#       ...
#       Cert Status: good
```

If output says `OCSP response: no response sent` — the server isn't stapling.

## Certificate Transparency

CT = a public, append-only Merkle-tree log of every certificate ever issued by participating CAs. Operated by Google, Cloudflare, DigiCert, Sectigo, Let's Encrypt, and others.

### Why it exists

Pre-CT, a rogue or compromised CA could issue a cert for `gmail.com` to an attacker, and Google might never know until end-users started complaining. CT makes mass-misissuance detectable: anyone can monitor the logs for unexpected certs in their domain.

### How it integrates into TLS

Browsers require leaf certs to be accompanied by 2+ Signed Certificate Timestamps (SCTs) from approved logs. The cert can carry SCTs as an X.509 extension, the server can deliver them via TLS extension `signed_certificate_timestamp`, or via OCSP stapling.

### Find every cert ever issued for your domain

```bash
# crt.sh — the canonical free CT search:
curl -s 'https://crt.sh/?q=example.com&output=json' | jq -r '.[].name_value' | sort -u

# Filter by date:
curl -s 'https://crt.sh/?q=example.com&output=json' \
  | jq -r '.[] | select(.not_before > "2026-01-01") | "\(.not_before)\t\(.name_value)"'
```

### Attack-surface enumeration

Pen-testers commonly use CT to enumerate subdomains:

```bash
# Get every subdomain that has ever had a cert:
curl -s 'https://crt.sh/?q=%25.example.com&output=json' \
  | jq -r '.[].name_value' | tr ',' '\n' | sort -u
```

(See the `polyglot` sheet for more web-scraping idioms.)

### Verify a cert has SCTs

```bash
openssl x509 -in cert.pem -noout -text | grep -A 3 'CT Precertificate SCTs'
```

## CAA — Certification Authority Authorization

DNS records that restrict which CAs can issue certs for your domain. RFC 8659. CAs MUST check CAA before issuing (RFC 8659 §3); if the CAA list excludes them, they must refuse.

### Record format

```
example.com.  IN  CAA  0 issue     "letsencrypt.org"
example.com.  IN  CAA  0 issuewild "letsencrypt.org"
example.com.  IN  CAA  0 iodef     "mailto:security@example.com"
```

- `issue` — which CA may issue normal certs.
- `issuewild` — which CA may issue wildcards.
- `iodef` — where to report violations.
- `0` is the flag byte; `128` would mean "critical" (CA must refuse if it doesn't understand the property).

### Common idioms

```
# Allow only Let's Encrypt:
example.com.  IN  CAA  0 issue "letsencrypt.org"

# Forbid issuance entirely (paranoid, breaks renewal):
example.com.  IN  CAA  0 issue ";"

# Restrict to specific account at LE:
example.com.  IN  CAA  0 issue "letsencrypt.org; accounturi=https://acme-v02.api.letsencrypt.org/acme/acct/12345"

# Multiple CAs (any can issue):
example.com.  IN  CAA  0 issue "letsencrypt.org"
example.com.  IN  CAA  0 issue "digicert.com"
```

### Query CAA

```bash
dig +short example.com CAA
# Expected:
# 0 issue "letsencrypt.org"
```

(See the `dig` sheet for full DNS query syntax.)

### Why it matters

A CAA record on its own doesn't stop a malicious CA — they can ignore CAA. But it shifts blame: a CA caught violating CAA loses browser trust. So CAA is a deterrent + a violation marker, not a hard barrier.

## Let's Encrypt and ACME

Let's Encrypt = the free, automated, browser-trusted CA, run by ISRG. ACME (Automated Certificate Management Environment, RFC 8555) = the protocol for requesting and renewing certs.

### Lifecycle

1. Generate an ACME account keypair (one-time).
2. Submit an order: list of domain names (identifiers).
3. CA returns a list of authorizations — one per identifier.
4. Each authorization has multiple challenge options (`http-01`, `dns-01`, `tls-alpn-01`).
5. Solve one challenge per identifier, prove control of the domain.
6. CA issues the cert.
7. Cert is valid 90 days.
8. Renew at ~60 days (rule of thumb: renew at 1/3 of remaining life).

### Challenge types

- **HTTP-01**: CA fetches `http://<domain>/.well-known/acme-challenge/<token>`. Server replies with `<token>.<account-key-thumbprint>`. Easy; works for non-wildcard certs only; needs port 80.
- **DNS-01**: CA looks up TXT record `_acme-challenge.<domain>` for a specific value. Required for wildcard certs (`*.example.com`). Useful when port 80 is blocked.
- **TLS-ALPN-01**: CA opens TLS to port 443 with ALPN `acme-tls/1` and a specific SNI; server presents a special cert containing the challenge. Useful when only port 443 is reachable.

### certbot — the canonical client

```bash
# HTTP-01, certbot runs its own webserver on port 80:
sudo certbot certonly --standalone -d example.com -d www.example.com

# HTTP-01, with nginx integration:
sudo certbot --nginx -d example.com

# DNS-01 for wildcard, with Cloudflare DNS plugin:
sudo certbot certonly --dns-cloudflare \
  --dns-cloudflare-credentials ~/.secrets/cloudflare.ini \
  -d 'example.com' -d '*.example.com'

# Renew (cron-friendly):
sudo certbot renew --quiet

# Force renewal for testing:
sudo certbot renew --force-renewal --cert-name example.com

# Dry-run (no real cert; uses staging server):
sudo certbot renew --dry-run
```

### acme.sh — alternative pure-shell client

```bash
curl https://get.acme.sh | sh
~/.acme.sh/acme.sh --issue -d example.com -w /var/www/html
~/.acme.sh/acme.sh --install-cert -d example.com \
  --key-file /etc/nginx/ssl/example.com.key \
  --fullchain-file /etc/nginx/ssl/example.com.crt \
  --reloadcmd "systemctl reload nginx"
```

### lego — Go-based, single binary

```bash
lego --email you@example.com --domains example.com --http run
lego --email you@example.com --domains example.com --http renew
```

### Renewal cron

Conventional wisdom: try to renew twice a day, no-op if not yet due. Even better, add jitter to avoid the 02:00 thundering herd:

```bash
# /etc/cron.d/certbot
0 0,12 * * * root sleep $((RANDOM % 3600)) && certbot renew --quiet --post-hook "systemctl reload nginx"
```

(See the `bash` sheet for `RANDOM` semantics.)

### Staging vs production

ALWAYS test against the ACME staging endpoint first. Production has aggressive rate limits; staging has none.

```bash
certbot register --server https://acme-staging-v02.api.letsencrypt.org/directory ...
```

### Rate limits (production, current)

- 50 certs / week / registered domain.
- 5 duplicate certs / week.
- 5 failed validations / hour / hostname.
- 300 new orders / 3 hours / account.

Hit a limit, you wait. There's no escape hatch.

## mTLS — Mutual TLS Authentication

Both sides authenticate via certs, not just the server. Common uses:

- Service-to-service auth in a service mesh (Istio, Linkerd, Consul Connect).
- API gateways requiring client certs from partners.
- VPN-like client cert auth on web portals.

### Wire flow (TLS 1.3)

```
ClientHello                  -->
                             <-- ServerHello + EncryptedExtensions + CertificateRequest
                             <-- Certificate + CertificateVerify + Finished
Certificate                  -->
CertificateVerify            -->
Finished                     -->
```

The `CertificateRequest` carries the list of CAs the server will accept the client cert from. The client picks any cert it has whose issuer is in that list (or sends an empty `Certificate` message if none — server then decides whether to fail or fall back).

### Server config — nginx mTLS

```nginx
server {
    listen 443 ssl http2;
    ssl_certificate     /etc/nginx/ssl/server.crt;
    ssl_certificate_key /etc/nginx/ssl/server.key;

    ssl_client_certificate /etc/nginx/ssl/client-ca-bundle.crt;
    ssl_verify_client      on;            # require client cert
    ssl_verify_depth       2;

    # Pass cert details to upstream via headers:
    location / {
        proxy_set_header X-Client-Cert-Subject $ssl_client_s_dn;
        proxy_set_header X-Client-Cert-Verify  $ssl_client_verify;
        proxy_pass http://backend;
    }
}
```

### Server config — apache mTLS

```apache
SSLVerifyClient   require
SSLVerifyDepth    2
SSLCACertificateFile /etc/apache2/ssl/client-ca-bundle.crt
```

### Test from the client side

```bash
# With curl:
curl --cert client.crt --key client.key \
     --cacert server-ca.crt \
     https://api.example.com/

# With openssl s_client:
openssl s_client -connect api.example.com:443 \
  -servername api.example.com \
  -cert client.crt -key client.key \
  -CAfile server-ca.crt
```

### Generating a client cert with a private CA

```bash
# 1. CA key + self-signed CA cert
openssl genrsa -out ca.key 4096
openssl req -x509 -new -nodes -key ca.key -sha256 -days 3650 \
  -out ca.crt -subj "/CN=Internal Root CA"

# 2. Client key
openssl genrsa -out client.key 2048

# 3. Client CSR
openssl req -new -key client.key -out client.csr -subj "/CN=alice@example.com"

# 4. Sign with CA, with clientAuth EKU:
cat > client.ext <<EOF
extendedKeyUsage = clientAuth
EOF
openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
  -out client.crt -days 365 -sha256 -extfile client.ext

# Verify:
openssl x509 -in client.crt -noout -text | grep -A 2 'Extended Key Usage'
# Expected: TLS Web Client Authentication
```

### Operational reality

Private CAs introduce operational burden:

- Cert lifecycle management (issuance, renewal, revocation).
- Key distribution.
- CA root rotation when the root expires (every 10–20 years).
- HSM for the CA private key (recommended).

Hashicorp Vault, smallstep CA, Boulder (Let's Encrypt's), and Cloudflare CFSSL all aim at this niche.

## Session Resumption — Tickets and PSK

Full TLS handshake = 2-RTT (TLS 1.2) or 1-RTT (TLS 1.3) plus several KB of cert chain. If a client reconnects shortly after, do you really need to do all that again? No.

### TLS 1.2 — Session ID

Server assigns a session ID in `ServerHello`, caches the master secret keyed by ID. Client stores the ID. On reconnect, client sends `ClientHello` with the ID; if server still has it cached, the handshake collapses to 1-RTT and the cert chain isn't sent.

Drawback: requires server-side state. Cluster of servers needs a shared cache (memcached, etc.).

### TLS 1.2 — Session Tickets (RFC 5077)

Server encrypts the master secret + connection params under a server-side key, hands the ticket to the client. Client presents it on reconnect; server decrypts. Stateless server-side.

Drawback: if the ticket key leaks, every old ticket can be decrypted — breaking forward secrecy. Best practice: rotate the ticket key every few hours.

```nginx
# nginx — disable tickets if you can't rotate keys, or rotate them yourself:
ssl_session_tickets on;
ssl_session_ticket_key /etc/nginx/ssl_ticket.key;
# Generate: openssl rand 80 > /etc/nginx/ssl_ticket.key
```

### TLS 1.3 — PSK (Pre-Shared Keys)

Server sends a `NewSessionTicket` post-handshake. The ticket includes a PSK identity + ticket lifetime. On reconnect, client sends the PSK identity in `ClientHello`'s `pre_shared_key` extension. Server resumes with the matching PSK.

Combined with `key_share`, the resumption is 1-RTT (or 0-RTT with early data).

### 0-RTT replay caveat

0-RTT data is encrypted with a key derivable purely from the PSK and `ClientHello`'s nonce. An attacker who captures the 0-RTT request can replay it (the nonce isn't bound to a server-acknowledged token). Servers need anti-replay defenses:

- Single-use ticket (ticket binding to a one-time `obfuscated_ticket_age` window).
- `Anti-Replay` ServerHello extension (still draft in many places).
- Application-level idempotency.

Production rule: only allow 0-RTT for safe, idempotent requests (HTTP GET, conditional fetches).

```nginx
ssl_early_data off;     # default; turn on only if you've thought through replays
```

## Diagnosing TLS — openssl s_client

The Swiss Army knife. (See the `openssl` sheet for the whole `openssl` family.)

### Canonical commands

```bash
# Full chain dump:
openssl s_client -connect host:443 -servername host -showcerts </dev/null

# Force TLS 1.2:
openssl s_client -connect host:443 -servername host -tls1_2 </dev/null

# Force TLS 1.3:
openssl s_client -connect host:443 -servername host -tls1_3 </dev/null

# Test ALPN:
openssl s_client -connect host:443 -servername host -alpn h2,http/1.1 </dev/null 2>&1 | grep -i ALPN

# Force a specific cipher (TLS 1.2):
openssl s_client -connect host:443 -servername host -cipher 'ECDHE-RSA-AES256-GCM-SHA384' </dev/null

# Force a specific cipher (TLS 1.3):
openssl s_client -connect host:443 -servername host -ciphersuites 'TLS_AES_256_GCM_SHA384' </dev/null

# Quick expiry check:
echo | openssl s_client -connect host:443 -servername host 2>/dev/null \
  | openssl x509 -noout -dates

# Full subject:
echo | openssl s_client -connect host:443 -servername host 2>/dev/null \
  | openssl x509 -noout -subject -issuer

# All SANs:
echo | openssl s_client -connect host:443 -servername host 2>/dev/null \
  | openssl x509 -noout -ext subjectAltName

# OCSP stapling test:
openssl s_client -connect host:443 -servername host -status </dev/null 2>&1 | grep -A 5 'OCSP'

# Verbose protocol-level debug:
openssl s_client -connect host:443 -servername host -msg -trace </dev/null

# Show only the negotiated bits:
openssl s_client -connect host:443 -servername host </dev/null 2>&1 \
  | grep -E '^(Protocol|Cipher|Server Temp Key|Peer signature)'

# Test STARTTLS for SMTP submission:
openssl s_client -connect smtp.example.com:587 -starttls smtp

# Test STARTTLS for IMAP:
openssl s_client -connect imap.example.com:143 -starttls imap

# Test STARTTLS for POP3:
openssl s_client -connect pop.example.com:110 -starttls pop3

# Test STARTTLS for FTP:
openssl s_client -connect ftp.example.com:21 -starttls ftp

# Test STARTTLS for XMPP (jabber):
openssl s_client -connect xmpp.example.com:5222 -starttls xmpp

# Test STARTTLS for LDAP:
openssl s_client -connect ldap.example.com:389 -starttls ldap

# DTLS (UDP-based TLS):
openssl s_client -connect host:4443 -dtls
```

### What `</dev/null` does

By default, `openssl s_client` keeps the connection open and reads stdin for application data. Piping `/dev/null` (empty input) makes it close cleanly after handshake.

### Connection lifecycle to look for

```
CONNECTED(00000003)
depth=2 ...
verify return:1
depth=1 ...
verify return:1
depth=0 ...
verify return:1
---
Certificate chain
 0 s:CN=example.com
   i:CN=R3, O=Let's Encrypt, C=US
 1 s:CN=R3, O=Let's Encrypt, C=US
   i:CN=ISRG Root X1, O=Internet Security Research Group, C=US
---
SSL handshake has read 4321 bytes and written 678 bytes
---
New, TLSv1.3, Cipher is TLS_AES_256_GCM_SHA384
Server public key is 2048 bit
Verify return code: 0 (ok)
---
```

If `Verify return code: 0 (ok)` — chain validates. Anything else is the reason.

### Mac OS LibreSSL note

macOS ships LibreSSL as `/usr/bin/openssl`. Some flags (`-msg -trace`, modern cipher names) differ. Install Homebrew openssl:

```bash
brew install openssl
/opt/homebrew/opt/openssl/bin/openssl s_client ...
```

## Diagnosing TLS — testssl.sh and SSL Labs

`testssl.sh` is the comprehensive scanner. Open source, runs locally, no data sent to third parties.

### Install

```bash
git clone --depth 1 https://github.com/drwetter/testssl.sh.git
cd testssl.sh
./testssl.sh --version
```

### Full audit

```bash
./testssl.sh https://example.com
# Or:
./testssl.sh example.com:443

# Just protocols:
./testssl.sh -p https://example.com

# Just ciphers, by category:
./testssl.sh -E https://example.com

# Just cert details:
./testssl.sh -S https://example.com

# Vulnerabilities:
./testssl.sh -U https://example.com

# Quick / less verbose:
./testssl.sh --fast https://example.com

# JSON output for CI:
./testssl.sh --jsonfile out.json https://example.com

# Nicer terminal colors:
./testssl.sh --color 3 https://example.com
```

### What `-U` covers

- POODLE (SSLv3 padding oracle)
- BEAST (TLS 1.0 CBC)
- CRIME (TLS compression)
- BREACH (HTTP compression)
- FREAK (export-grade RSA)
- LOGJAM (export-grade DH)
- Sweet32 (3DES birthday)
- Heartbleed (OpenSSL CVE-2014-0160)
- Ticketbleed
- ROBOT (RSA padding oracle)
- LUCKY13 (CBC timing)
- BLEICHENBACHER attacks
- DROWN (cross-protocol SSLv2)
- WINSHOCK / SChannel issues

### SSL Labs (web equivalent)

```
https://www.ssllabs.com/ssltest/analyze.html?d=example.com&hideResults=on&latest
```

Grades A+ down to F. The `&hideResults=on` flag keeps your scan out of the public archive.

### Mozilla Observatory

```
https://observatory.mozilla.org/analyze/example.com
```

Goes beyond TLS into HTTP security headers (HSTS, CSP, frame options, etc.).

## Common Vulnerabilities — Historical

Each one is a sub-genre of attack. None should be live in 2026 with modern config; they all motivate a config knob to turn off.

### POODLE — SSL 3.0 Padding Oracle (Oct 2014)

CBC-mode SSLv3 lets an attacker decrypt one byte per ~256 forced-downgrade requests. Killed SSL 3.0. Fix: disable SSL 3.0 entirely.

```nginx
ssl_protocols TLSv1.2 TLSv1.3;     # never list SSLv3
```

### BEAST — TLS 1.0 CBC IV Predictability (2011)

CBC mode in TLS 1.0 chained the IV from the previous record's last block — predictable. Attacker injects controlled plaintext (via JS in the browser) to recover unknown bytes. Workaround: TLS 1.1+ uses explicit per-record IVs. Real fix: don't use CBC; use AEAD ciphers (TLS 1.2 GCM).

### CRIME — TLS Compression Side Channel (2012)

If TLS-level compression is on, attacker observing ciphertext length can binary-search the contents (e.g., a session cookie). Fix: disable TLS compression. Modern OpenSSL doesn't even compile it in.

### BREACH — HTTP Body Compression (2013)

Same idea as CRIME but at the HTTP layer (gzip Content-Encoding). Hard to disable globally because gzip is essential for performance. Fixes: random padding on sensitive endpoints, mask CSRF tokens per-request, length-hide via random length headers.

### Heartbleed — OpenSSL CVE-2014-0160

A specific OpenSSL bug in the heartbeat extension let an attacker read up to 64 KB of server-process memory (private keys, session data) per request. Patched in OpenSSL 1.0.1g (April 2014). Fix: upgrade OpenSSL; rotate every key/cert that was on a vulnerable server.

```bash
# Detect:
nmap --script ssl-heartbleed -p 443 host
```

### FREAK — Factoring RSA Export Keys (2015)

Servers still supporting `EXPORT_RSA_*` ciphers (legacy 512-bit RSA from US export controls) could be downgraded. The 512-bit RSA keys factor in hours on a cloud cluster. Fix: disable export ciphers.

### LOGJAM — Export DH (2015)

Same idea, DH side. 512-bit DH groups precomputable. Fix: disable export ciphers, use ≥2048-bit DH.

```nginx
# Generate strong DH params:
openssl dhparam -out /etc/nginx/dhparam.pem 2048
ssl_dhparam /etc/nginx/dhparam.pem;
```

### Sweet32 — 3DES Birthday (2016)

3DES has a 64-bit block. After ~2^32 blocks (about 32 GB) under one key, birthday collisions reveal plaintext. Modern HTTP keeps connections open long enough to hit this. Fix: disable 3DES.

### DROWN — Cross-protocol SSLv2 (2016)

A server still accepting SSLv2 with the same cert/key as TLS 1.2 leaks TLS 1.2 sessions. Fix: turn off SSLv2 EVERYWHERE on every port using that key (mail server, etc., not just HTTPS).

### ROBOT — Bleichenbacher 1998 Returned (2017)

PKCS#1 v1.5 padding oracle in TLS RSA key transport. Attacker can decrypt past sessions or forge signatures, given enough queries. Fix: prefer ECDHE so RSA key transport isn't used; patch OpenSSL/F5/Citrix loadbalancers.

### LUCKY13 — CBC Timing (2013)

Padding error vs MAC error timing differences in TLS 1.0/1.1/1.2 CBC. Fixable with constant-time MAC verification. Real fix: AEAD ciphers.

## Modern Best-Practice TLS Config

Use the Mozilla SSL Configuration Generator. Pick "Modern", "Intermediate", or "Old" based on what clients you must support.

```
https://ssl-config.mozilla.org/
```

### Modern (TLS 1.3 only)

Supports: Firefox 63+, Android 10+, Chrome 70+, iOS 12.2+, Java 11+, OpenSSL 1.1.1+, Opera 57+, Safari 12.1+. NOT IE 11.

```nginx
server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;

    ssl_certificate     /path/to/fullchain.pem;
    ssl_certificate_key /path/to/privkey.pem;

    ssl_protocols TLSv1.3;
    ssl_prefer_server_ciphers off;

    add_header Strict-Transport-Security "max-age=63072000" always;

    # OCSP stapling
    ssl_stapling on;
    ssl_stapling_verify on;
    ssl_trusted_certificate /path/to/chain.pem;

    resolver 1.1.1.1 8.8.8.8 valid=300s;
}
```

### Intermediate (TLS 1.2 + 1.3)

Supports nearly everything since 2013. The current sane default.

```nginx
server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;

    ssl_certificate     /path/to/fullchain.pem;
    ssl_certificate_key /path/to/privkey.pem;

    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:DHE-RSA-AES128-GCM-SHA256:DHE-RSA-AES256-GCM-SHA384:DHE-RSA-CHACHA20-POLY1305;
    ssl_prefer_server_ciphers off;

    ssl_dhparam /path/to/dhparam.pem;

    add_header Strict-Transport-Security "max-age=63072000" always;

    ssl_stapling on;
    ssl_stapling_verify on;
    ssl_trusted_certificate /path/to/chain.pem;

    resolver 1.1.1.1 8.8.8.8 valid=300s;
}
```

### Old (legacy clients)

Only if you must support clients like Windows XP IE6. You shouldn't.

### apache 2.4 intermediate

```apache
SSLEngine on
SSLCertificateFile      /path/to/fullchain.pem
SSLCertificateKeyFile   /path/to/privkey.pem

SSLProtocol             all -SSLv3 -TLSv1 -TLSv1.1
SSLCipherSuite          ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:DHE-RSA-AES128-GCM-SHA256:DHE-RSA-AES256-GCM-SHA384
SSLHonorCipherOrder     off
SSLSessionTickets       off

SSLOpenSSLConfCmd DHParameters "/path/to/dhparam.pem"

SSLUseStapling          On
SSLStaplingCache        "shmcb:logs/ssl_stapling(32768)"

Header always set Strict-Transport-Security "max-age=63072000"
```

### haproxy intermediate

```
global
    ssl-default-bind-ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:DHE-RSA-AES128-GCM-SHA256:DHE-RSA-AES256-GCM-SHA384
    ssl-default-bind-ciphersuites TLS_AES_128_GCM_SHA256:TLS_AES_256_GCM_SHA384:TLS_CHACHA20_POLY1305_SHA256
    ssl-default-bind-options ssl-min-ver TLSv1.2 no-tls-tickets

frontend https
    bind *:443 ssl crt /path/to/fullchain-and-key.pem alpn h2,http/1.1
    http-response set-header Strict-Transport-Security max-age=63072000
```

## Cipher Suite Cookbook

The modern TLS 1.2 allowlist (in OpenSSL dashed-syntax). Order matters — first match wins on most servers.

```
# 12 cipher suites, ordered: AES-GCM > ChaCha20 > prefer ECDSA over RSA
ECDHE-ECDSA-AES128-GCM-SHA256
ECDHE-ECDSA-AES256-GCM-SHA384
ECDHE-RSA-AES128-GCM-SHA256
ECDHE-RSA-AES256-GCM-SHA384
ECDHE-ECDSA-CHACHA20-POLY1305
ECDHE-RSA-CHACHA20-POLY1305
DHE-RSA-AES128-GCM-SHA256
DHE-RSA-AES256-GCM-SHA384
DHE-RSA-CHACHA20-POLY1305
```

For TLS 1.3 the suite list is fixed and minimal:

```
TLS_AES_128_GCM_SHA256
TLS_AES_256_GCM_SHA384
TLS_CHACHA20_POLY1305_SHA256
```

### What to NEVER include

```
*RC4*           # broken in TLS by 2015
*3DES*          # Sweet32
*EXP*           # export ciphers, FREAK/LOGJAM
*MD5            # collision-broken
*-CBC-SHA       # CBC + SHA-1 = Lucky13 + deprecated hash
NULL*           # unencrypted, debugging only
ADH-*           # anonymous DH, no auth
```

### Test what your server actually negotiates

```bash
# Ordered list of every cipher this server speaks:
nmap --script ssl-enum-ciphers -p 443 host

# Force one suite at a time to find what's enabled:
for c in $(openssl ciphers 'ECDHE+AESGCM:ECDHE+CHACHA20' | tr ':' '\n'); do
  echo -n "$c: "
  echo | openssl s_client -connect host:443 -servername host -cipher "$c" 2>&1 | grep -E 'Cipher|alert' | head -1
done
```

## HSTS — HTTP Strict Transport Security

The `Strict-Transport-Security` response header tells browsers "for this domain, always use HTTPS, never accept HTTP, even if the user typed it." Defends against SSL-stripping MITM.

### Header format

```
Strict-Transport-Security: max-age=63072000; includeSubDomains; preload
```

- `max-age=N` — seconds the policy persists. `63072000` = 2 years (browser preload list requires ≥1 year).
- `includeSubDomains` — applies to every subdomain too.
- `preload` — opts the domain into the browser-bundled HSTS preload list.

### Preload list

Submit at https://hstspreload.org/. Once accepted, every shipped browser hard-codes the policy — even a brand-new device that's never seen your site enforces HTTPS. Removing yourself from the list takes weeks/months and may have already shipped to phones with frozen update channels.

### Server-side

```nginx
# nginx
add_header Strict-Transport-Security "max-age=63072000; includeSubDomains; preload" always;
# 'always' = include even on error responses
```

```apache
# apache
Header always set Strict-Transport-Security "max-age=63072000; includeSubDomains; preload"
```

### Gotcha: includeSubDomains

If you set `includeSubDomains` on `example.com` and you have a legacy HTTP-only sub `legacy.example.com`, that sub instantly breaks. Test:

```bash
# Roll out staged: first max-age=300 (5 min), bump to 3600, then 86400, then 31536000.
add_header Strict-Transport-Security "max-age=300" always;
```

Once you're sure, ramp.

## HPKP — Deprecated

HTTP Public Key Pinning (RFC 7469) let a server tell the browser "only accept these specific public keys for me, even if a different valid cert is presented." Nice in theory; in practice:

- Caused too many self-inflicted outages — admins pinned the leaf, then renewal changed the key, every visitor got a hard error for `max-age` seconds.
- Replaced by Certificate Transparency + monitoring.

Chrome removed support in 2018. Don't use HPKP. Don't even mention it except to tell people not to use it. The successor `Expect-CT` is also deprecated as of mid-2023; CT enforcement is now built into the cert validation path itself.

## Common Errors and Fixes

Each row has the exact text an admin/user is likely to see, the cause, and the fix.

### Browser errors

```
ERR_CERT_AUTHORITY_INVALID
"Your connection is not private"
"net::ERR_CERT_AUTHORITY_INVALID"
"NET::ERR_CERT_AUTHORITY_INVALID"
"self signed certificate"
"self signed certificate in certificate chain"
"unable to get local issuer certificate"
```

Cause: the leaf is signed by an intermediate the client doesn't trust, OR the server didn't send the intermediate. Fix: install the full chain (`fullchain.pem`, not `cert.pem` alone) on the server.

```
ERR_CERT_DATE_INVALID
"Your connection is not private"
"net::ERR_CERT_DATE_INVALID"
"certificate has expired"
"NotAfter: <past date>"
```

Cause: the leaf or an intermediate has expired (or system clock is wrong). Fix: renew the cert; check NTP if the clock is off.

```
ERR_CERT_COMMON_NAME_INVALID
"subject alternative name mismatch"
"hostname doesn't match certificate"
```

Cause: the cert's SAN list doesn't include the hostname the client connected to. Fix: reissue the cert with the correct SANs (e.g., add `www.example.com` to a cert with only `example.com`).

```
ERR_SSL_PROTOCOL_ERROR
"net::ERR_SSL_PROTOCOL_ERROR"
"unsupported protocol"
"alert protocol version"
"version too low / too high"
```

Cause: client and server can't agree on a TLS version (one wants TLS 1.0 only, the other wants 1.2+). Fix: align — typically modernize the client.

```
ERR_SSL_VERSION_INTERFERENCE
```

Cause: a TLS-aware middlebox (SSL inspection appliance, school proxy, corporate AV) is messing with the handshake. Fix: install the middlebox's CA cert into the client's trust store, or remove the middlebox.

```
SSL_ERROR_HANDSHAKE_FAILURE
"alert handshake failure"
"no shared cipher"
```

Cause: no overlap between client and server cipher suite lists. Fix: enable a modern cipher on at least one side.

```
ERR_SSL_BAD_RECORD_MAC_ALERT
"alert bad_record_mac"
"alert decrypt_error"
```

Cause: a record's MAC didn't verify — either bit-flip on the wire (rare with TCP), buggy cipher implementation, or a man-in-the-middle. Often indicates flaky NIC/middlebox.

```
DH KEY too small
"dh key too small"
"ssl_choose_client_version:no protocols available"
```

Cause: server uses <2048-bit DH parameters; modern OpenSSL refuses by default. Fix: regenerate `dhparam.pem` at 2048+ bits, or switch to ECDHE-only.

```
no peer certificate available
```

Cause: server didn't present a cert (rare for HTTPS — usually means you connected to a non-TLS port, or the server is misconfigured to not send the cert). Check `openssl s_client -connect host:port` output.

```
Bad TLS Hostname
"sslv3 alert handshake failure" (when server wants SNI but you didn't send it)
```

Cause: SNI mismatch or SNI not sent on a virtual-hosted server. Fix: pass `-servername host` to openssl, or fix the client to send SNI.

```
tlsv1 alert internal error
"alert internal_error"
```

Cause: server-side fault. Common in our experience: missing intermediate, an HSM that just lost connection, or a misbehaving plugin. Check server logs.

```
tlsv1 alert protocol version
"alert protocol_version"
```

Cause: explicit version-mismatch alert from server. Server doesn't speak the version the client demanded. Fix: verify with `nmap --script ssl-enum-ciphers`.

```
tlsv1 alert decode error
"alert decode_error"
```

Cause: mid-handshake the receiver couldn't parse a message (malformed ASN.1, bad length, etc.). Often a buggy implementation on one side.

### OpenSSL command errors

```
"unable to load certificate"
"PEM routines:get_name:no start line"
```

Cause: file isn't PEM, or has Windows line endings, or has UTF-8 BOM. Fix: `dos2unix cert.pem; head -1 cert.pem` should be `-----BEGIN CERTIFICATE-----`.

```
"139934217...:error:..:lib(20):func(...):reason(...)"
```

The hex prefix (e.g. `139934217`) is a thread ID. The interesting parts are `lib`, `func`, and `reason`. Look up the reason code with `openssl errstr 0xN` if needed.

### nginx errors

```
"SSL_do_handshake() failed (SSL: error:14094412:SSL routines:SSL3_READ_BYTES:sslv3 alert bad certificate) while SSL handshaking, client: ..."
```

mTLS: the client cert chain doesn't validate against `ssl_client_certificate`. Check the client cert's issuer is in the configured CA bundle.

```
"SSL_do_handshake() failed (SSL: error:1417C0C7:SSL routines:tls_process_client_hello:peer did not return a certificate)"
```

mTLS: client didn't send a cert. Make sure the client config actually includes the cert.

### Apache errors

```
"AH02572: Failed to configure at least one certificate and key for example.com:443"
```

Cause: the path in `SSLCertificateFile` or `SSLCertificateKeyFile` is wrong, or the cert and key don't match. Verify match:

```bash
openssl rsa -in privkey.pem -modulus -noout | openssl md5
openssl x509 -in cert.pem -modulus -noout | openssl md5
# Same hash = matching pair.
```

## Common Gotchas

Catalogue of broken patterns and the fix.

### Serving leaf without intermediates

```nginx
# Bad:
ssl_certificate /etc/letsencrypt/live/example.com/cert.pem;          # leaf only

# Good:
ssl_certificate /etc/letsencrypt/live/example.com/fullchain.pem;     # leaf + intermediate
```

Most browsers will quietly fetch the missing intermediate via AIA, but `curl`, Java, Go, monitoring tools, and corporate proxies will fail.

### Cert expiring unnoticed

You don't notice until users call. Always have monitoring:

```bash
# Cron-friendly expiry checker:
host=example.com
days=$(echo | openssl s_client -connect "$host":443 -servername "$host" 2>/dev/null \
  | openssl x509 -noout -enddate \
  | sed 's/notAfter=//' \
  | xargs -I{} date -d {} +%s)
now=$(date +%s)
echo $(( (days - now) / 86400 )) days remaining
```

Plug into Prometheus' `blackbox_exporter` `tls_cert_not_after` metric, or use a SaaS like Detectify, Better Uptime, etc.

### Supporting TLS 1.0 / 1.1 in 2026

Don't. Audit and disable:

```nginx
ssl_protocols TLSv1.2 TLSv1.3;
```

If you have a hardware client that truly cannot do TLS 1.2, you have a security problem you should be paying down, not perpetuating.

### Forgot to renew

Use ACME automation:

```
# /etc/cron.d/certbot
0 0,12 * * * root sleep $((RANDOM % 3600)) && certbot renew --quiet --post-hook "systemctl reload nginx"
```

### Cipher list copied from 2014 stackoverflow

Old answers list `RC4-SHA`, `DES-CBC3-SHA`, etc. as "for compatibility." Don't. Use Mozilla's generator output instead.

### SNI not set on s_client when virtual hosting

```bash
# Bad (server returns "default" cert, may be wrong domain):
openssl s_client -connect host:443

# Good:
openssl s_client -connect host:443 -servername host
```

This is the #1 source of "the cert looks fine in browsers but my Go HTTP client says hostname mismatch" — the Go client correctly sends SNI; your debug command doesn't.

### HSTS preload + includeSubDomains breaking subdomains

If `legacy.example.com` is HTTP-only and you preload `example.com` with `includeSubDomains`, every browser that's seen the preload will refuse legacy. Fix BEFORE you preload:

1. Inventory every subdomain.
2. Make every subdomain HTTPS (free with Let's Encrypt).
3. Roll HSTS with short `max-age` first, ramp.
4. Only THEN add `includeSubDomains; preload`.

### Long-lived session tickets

If your nginx ticket key never rotates, an attacker who compromises that key can decrypt every session that used a ticket since the key was generated — no forward secrecy. Rotate via cron, or just disable tickets if you don't need them.

### Mixed-content warnings

A page served over HTTPS that includes `http://...` resources — browser blocks/warns. Fix every `<img src="http://..." />`, `<script src="http://..." />`, etc. Use `https://` or scheme-relative `//host/path`.

### CN-only certs

Some old CA-issued certs have only CN, no SAN. Modern browsers reject them outright. Reissue with proper SANs.

### Wildcard cert behavior

```
*.example.com   matches    foo.example.com
                NOT MATCH  example.com         (apex)
                NOT MATCH  bar.foo.example.com (two-level)
```

Buy / issue both `example.com` AND `*.example.com` if you serve both apex and subs.

### Path MTU + handshake fragmentation

Some legacy middleboxes drop fragmented handshake records. Symptom: handshake hangs on networks with smaller MTUs. Workaround: smaller cert chain, or jumbo support along the path.

## Performance — TLS Optimization

### Session resumption

- Drops handshake to 1-RTT (TLS 1.2 with session ID/ticket; TLS 1.3 with PSK).
- 0-RTT (TLS 1.3 only) drops to ~0-RTT for the first request, replay-restricted.
- Configure session cache sizes generously: `ssl_session_cache shared:SSL:50m;` in nginx ≈ 200,000 sessions.

### OCSP stapling

- Saves the client an extra RTT to the OCSP responder.
- Saves the responder bandwidth.
- See OCSP Stapling section above for config.

### HTTP/2

- Multiplexes multiple HTTP requests over one TLS connection — one handshake amortized over many requests.
- nginx `listen 443 ssl http2;`
- Apache `Protocols h2 http/1.1`

### HTTP/3 (QUIC)

- Combines transport + TLS 1.3 in 1-RTT (or 0-RTT).
- Eliminates head-of-line blocking at the transport layer.
- nginx 1.25+ supports `listen 443 quic reuseport;`
- HAProxy 2.6+, Caddy 2+, LiteSpeed have stable support.

### ECC certs over RSA

- ECDSA-P256 leaf cert is ~50% smaller than RSA-2048.
- ECDSA verify is ~3-10x faster than RSA-2048 verify.
- Negotiating `ECDHE-ECDSA-*` ciphers shaves bytes on the wire.

```bash
# Generate ECDSA keypair + CSR:
openssl ecparam -genkey -name secp256r1 -out ec.key
openssl req -new -key ec.key -out ec.csr -subj "/CN=example.com"
```

### Hybrid certs (RSA + ECDSA)

Modern servers can present both — clients negotiate via `signature_algorithms`. nginx:

```nginx
ssl_certificate     /path/rsa-fullchain.pem;
ssl_certificate_key /path/rsa.key;
ssl_certificate     /path/ecdsa-fullchain.pem;
ssl_certificate_key /path/ecdsa.key;
```

### Connection reuse / keep-alive

- HTTP/1.1 keep-alive amortizes one handshake over many requests on the same connection.
- HTTP/2 multiplexing is even better.
- Too-aggressive keep-alive timeout wastes server FDs; balance.

### TLS False Start (deprecated)

A short-lived idea to send app data after just one round trip in TLS 1.2; obsoleted by TLS 1.3's native 1-RTT.

### Hardware acceleration

- AES-NI on x86 → AES-GCM is essentially free.
- ARMv8 Cryptography Extensions → same on ARM.
- CHACHA20-POLY1305 if neither is available (mobile in 2014-2018).

### Profiling a TLS handshake

```bash
# curl times each phase:
curl -w 'dns:%{time_namelookup}s connect:%{time_connect}s tls:%{time_appconnect}s ttfb:%{time_starttransfer}s total:%{time_total}s\n' \
  -s -o /dev/null https://example.com

# tcpdump the handshake bytes:
tcpdump -i any -s0 -w tls.pcap port 443 and host host
# Open in Wireshark, filter `tls`. (Wireshark decodes if you have keylog.)
```

### Wireshark TLS decryption

```bash
# Tell Chrome / curl to log keys:
SSLKEYLOGFILE=/tmp/tls-keys.log curl https://example.com

# In Wireshark: Edit > Preferences > Protocols > TLS > (Pre)-Master-Secret log filename
# Set to /tmp/tls-keys.log. Now Wireshark decrypts the captured packets.
```

## Idioms

### "I want a free, auto-renewing cert on this server."

```bash
sudo apt install certbot python3-certbot-nginx
sudo certbot --nginx -d example.com -d www.example.com
# certbot edits nginx config, schedules renewal cron, done.
```

### "I want an A+ on SSL Labs."

1. Mozilla "Modern" or "Intermediate" config.
2. HSTS with `max-age=31536000; includeSubDomains` (preload optional).
3. OCSP stapling on.
4. Drop SSL/TLS <1.2.
5. Strong DH params (≥2048) or just disable DHE in favor of ECDHE-only.
6. Test with `https://www.ssllabs.com/ssltest/`.

### "I want to see which sites actually use TLS 1.3."

```bash
for host in $(cat hosts.txt); do
  ver=$(openssl s_client -connect "$host":443 -servername "$host" </dev/null 2>&1 | grep '^Protocol' | awk '{print $3}')
  echo "$host $ver"
done
```

### "I want a private CA for my microservices."

Use `smallstep`:

```bash
brew install step
step ca init                  # interactive
step ca certificate svc.example.com svc.crt svc.key
```

Or HashiCorp Vault's PKI engine, or cert-manager in Kubernetes (with a `ClusterIssuer`).

### "I want to issue mTLS certs to engineers."

Look at `step-ca` with OIDC provisioner — engineers `step ca certificate $(whoami) ~/.tls/cert ~/.tls/key` and the CA verifies their Google/Okta SSO session.

### "I want HTTPS in front of a localhost dev server."

```bash
brew install mkcert
mkcert -install                    # add a trusted local CA to your trust store
mkcert localhost 127.0.0.1 ::1     # issues localhost.pem + localhost-key.pem
```

Now `https://localhost` works without warnings.

### "I want to inspect traffic from a mobile app."

Use `mitmproxy`. Trust its CA on the phone, point the phone's Wi-Fi at the proxy, watch decrypted traffic. Doesn't work for cert-pinned apps.

### "Cert-manager in Kubernetes."

```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: ops@example.com
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
    - http01:
        ingress:
          class: nginx
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: example-com
spec:
  secretName: example-com-tls
  issuerRef:
    name: letsencrypt-prod
    kind: ClusterIssuer
  dnsNames:
  - example.com
  - www.example.com
```

### "I keep getting rate-limited by Let's Encrypt."

Switch to staging while iterating, then production for the real cert:

```bash
certbot --staging ...           # rate-limit-free playground
certbot ...                     # then real
```

## Tips

- Always `-servername` when debugging with `openssl s_client`. Modern servers virtual-host.
- Never put the root cert in `fullchain.pem`. The client already has it; sending it wastes bytes and breaks cross-signed chains.
- Renew at 60 days for a 90-day cert. Daily cron with idempotent renew is bulletproof.
- For high-availability behind a load balancer, terminate TLS at the LB, not at every backend.
- Don't put the CA private key on a webserver. Use an HSM or a dedicated CA server.
- For TLS 1.3 only deployments, you can ditch DH params entirely (`ssl_dhparam` is unnecessary).
- `0-RTT` is for idempotent reads only. The application must NOT mutate state on a 0-RTT request.
- Watch for OCSP responder downtime — it can break stapling silently. Monitor for unstapled handshakes.
- Avoid `ssl_session_tickets on;` without a key-rotation policy. Rotated tickets every 4-12 hours preserve forward secrecy.
- `openssl ciphers -v 'HIGH:!aNULL:!MD5'` lists what your local OpenSSL will negotiate. Helps build a cipher allowlist.
- For `curl` with custom CA: `--cacert ca.pem`. For Go's TLS client: set `tls.Config.RootCAs`.
- The `iodef` CAA record can be a webhook URL (`https://...`); CAs that observe a CAA violation can POST a structured report there.
- ECH support is partial; it's still good practice to deploy. The privacy gain is worth the marginal risk of unexpected client behavior.
- Don't pin certs in clients. Pin the CA / use CT log monitoring.
- Don't trust `Verify return code: 0 (ok)` from `openssl s_client` if you didn't pass `-CAfile` or have a populated default trust store.

## See Also

- openssl, ssh, dns, dig, polyglot, bash

## References

- [RFC 5246 — TLS 1.2](https://www.rfc-editor.org/rfc/rfc5246)
- [RFC 8446 — TLS 1.3](https://www.rfc-editor.org/rfc/rfc8446)
- [RFC 6066 — TLS Extensions: Extension Definitions (SNI)](https://www.rfc-editor.org/rfc/rfc6066)
- [RFC 7301 — TLS ALPN Extension](https://www.rfc-editor.org/rfc/rfc7301)
- [RFC 7525 — Recommendations for Secure Use of TLS and DTLS](https://www.rfc-editor.org/rfc/rfc7525)
- [RFC 6797 — HTTP Strict Transport Security (HSTS)](https://www.rfc-editor.org/rfc/rfc6797)
- [RFC 6960 — OCSP](https://www.rfc-editor.org/rfc/rfc6960)
- [RFC 6962 — Certificate Transparency](https://www.rfc-editor.org/rfc/rfc6962)
- [RFC 8555 — Automatic Certificate Management Environment (ACME)](https://www.rfc-editor.org/rfc/rfc8555)
- [RFC 8659 — DNS Certification Authority Authorization (CAA)](https://www.rfc-editor.org/rfc/rfc8659)
- [RFC 8740 — Using TLS 1.3 with HTTP/2](https://www.rfc-editor.org/rfc/rfc8740)
- [RFC 8470 — TLS 1.3 Early Data and HTTP](https://www.rfc-editor.org/rfc/rfc8470)
- [RFC 7568 — Deprecating SSLv3](https://www.rfc-editor.org/rfc/rfc7568)
- [RFC 8996 — Deprecating TLS 1.0 and 1.1](https://www.rfc-editor.org/rfc/rfc8996)
- [RFC 8446bis (TLS 1.3 errata + clarifications)](https://datatracker.ietf.org/doc/draft-ietf-tls-rfc8446bis/)
- [RFC 9001 — TLS for QUIC](https://www.rfc-editor.org/rfc/rfc9001)
- [Mozilla SSL Configuration Generator](https://ssl-config.mozilla.org/)
- [Mozilla Server Side TLS Guidelines](https://wiki.mozilla.org/Security/Server_Side_TLS)
- [Mozilla Web Security](https://developer.mozilla.org/en-US/docs/Web/Security)
- [testssl.sh — Testing TLS/SSL Encryption](https://testssl.sh/)
- [Qualys SSL Labs — SSL Server Test](https://www.ssllabs.com/ssltest/)
- [crt.sh — Certificate Transparency search](https://crt.sh/)
- [Let's Encrypt documentation](https://letsencrypt.org/docs/)
- [certbot user guide](https://eff-certbot.readthedocs.io/en/stable/)
- ["Bulletproof TLS and PKI" by Ivan Ristić, Feisty Duck Press](https://www.feistyduck.com/books/bulletproof-tls-and-pki/)
- [Ivan Ristić's blog](https://blog.ivanristic.com/)
- [The TLS 1.3 illustrated guide (Michael Driscoll)](https://tls13.xargs.org/)
- [The TLS 1.2 illustrated guide (Michael Driscoll)](https://tls12.xargs.org/)
- [Cloudflare blog: TLS posts](https://blog.cloudflare.com/tag/tls/)
- [Hashicorp Vault PKI engine](https://developer.hashicorp.com/vault/docs/secrets/pki)
- [smallstep CA](https://smallstep.com/docs/step-ca/)
- [cert-manager (Kubernetes)](https://cert-manager.io/)
