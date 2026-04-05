# The Mathematics of OIDC — JWT Signatures, Token Entropy, and Session Security

> *OpenID Connect relies on cryptographic foundations at every layer. This explores the mathematics of JWT signature algorithms (RS256, ES256), the entropy requirements for nonces and PKCE verifiers, token lifetime optimization as a security-usability tradeoff, and the probability theory behind session fixation prevention.*

---

## 1. JWT Signature Mathematics (Cryptography)

### The Problem

JWT signatures ensure that tokens have not been tampered with and were issued by a trusted party. What are the mathematical operations behind RS256 and ES256?

### The Formula

**RS256 (RSA-PKCS1-v1.5 with SHA-256)**:

RSA key generation:
1. Choose two large primes $p, q$ (typically 1024 bits each for RSA-2048)
2. Compute $n = p \cdot q$ (the modulus)
3. Compute $\phi(n) = (p-1)(q-1)$
4. Choose $e$ (public exponent, typically 65537)
5. Compute $d \equiv e^{-1} \pmod{\phi(n)}$ (private exponent)

Signing (private key operation):
$$s = m^d \bmod n$$

where $m = \text{PKCS1-pad}(\text{SHA256}(\text{header.payload}))$.

Verification (public key operation):
$$m' = s^e \bmod n$$

Token is valid iff $m' = \text{PKCS1-pad}(\text{SHA256}(\text{header.payload}))$.

**ES256 (ECDSA with P-256 and SHA-256)**:

Key generation on curve P-256 ($y^2 = x^3 + ax + b \bmod p$):
1. Private key: random integer $d \in [1, n-1]$ where $n$ is the curve order
2. Public key: $Q = d \cdot G$ (point multiplication on the curve)

Signing:
1. Compute $h = \text{SHA256}(\text{header.payload})$, truncated to curve bit length
2. Choose random $k \in [1, n-1]$ (ephemeral key — **CRITICAL**: must be unique per signature)
3. Compute $(x_1, y_1) = k \cdot G$
4. $r = x_1 \bmod n$ (if $r = 0$, choose new $k$)
5. $s = k^{-1}(h + r \cdot d) \bmod n$ (if $s = 0$, choose new $k$)
6. Signature is $(r, s)$

Verification:
1. Compute $u_1 = h \cdot s^{-1} \bmod n$
2. Compute $u_2 = r \cdot s^{-1} \bmod n$
3. Compute $(x_1, y_1) = u_1 \cdot G + u_2 \cdot Q$
4. Valid iff $x_1 \equiv r \pmod{n}$

### Worked Examples

**Security levels**:

| Algorithm | Key Size | Security Level | Signature Size |
|-----------|----------|---------------|----------------|
| RS256 | 2048 bits | 112 bits | 256 bytes |
| RS256 | 4096 bits | 128 bits | 512 bytes |
| ES256 | 256 bits (P-256) | 128 bits | 64 bytes |
| ES384 | 384 bits (P-384) | 192 bits | 96 bytes |

ES256 achieves 128-bit security with 256-bit keys (vs 3072+ bits for equivalent RSA). JWT signature size: ES256 = 64 bytes vs RS256 = 256 bytes — significant for tokens in HTTP headers.

**ECDSA $k$-reuse vulnerability**: if the same $k$ is used for two different messages $h_1, h_2$:

$$s_1 = k^{-1}(h_1 + r \cdot d) \bmod n$$
$$s_2 = k^{-1}(h_2 + r \cdot d) \bmod n$$

Subtracting: $s_1 - s_2 = k^{-1}(h_1 - h_2) \bmod n$

$$k = \frac{h_1 - h_2}{s_1 - s_2} \bmod n$$

Then: $d = \frac{s_1 \cdot k - h_1}{r} \bmod n$

Private key recovered. This is why deterministic ECDSA (RFC 6979) is recommended.

## 2. Token Lifetime Optimization (Security-Usability Tradeoff)

### The Problem

Short-lived tokens are more secure (smaller window of compromise) but require more frequent refresh (worse UX, more server load). What is the optimal lifetime?

### The Formula

**Risk exposure** from a compromised token with lifetime $L$:

$$R(L) = P(\text{compromise}) \times E[\text{damage}(t)] \times L$$

If damage accumulates linearly at rate $\delta$ per second:

$$R(L) = P_c \times \delta \times \frac{L^2}{2}$$

(Quadratic because both the probability of being in a compromised state AND the duration contribute.)

**Refresh cost** with $n$ users and token lifetime $L$:

$$C(L) = \frac{n \times c_{refresh}}{L}$$

where $c_{refresh}$ is the cost of a single refresh operation.

**Total cost** (risk + operational):

$$T(L) = P_c \times \delta \times \frac{L^2}{2} + \frac{n \times c_{refresh}}{L}$$

Minimizing $T(L)$ by taking $\frac{dT}{dL} = 0$:

$$P_c \times \delta \times L - \frac{n \times c_{refresh}}{L^2} = 0$$

$$L^* = \left(\frac{n \times c_{refresh}}{P_c \times \delta}\right)^{1/3}$$

### Worked Examples

**Example**: 10,000 users, $P_c = 10^{-6}$ per second (probability of token being compromised), $\delta = 1$ damage unit/second, $c_{refresh} = 0.01$ cost units.

$$L^* = \left(\frac{10000 \times 0.01}{10^{-6} \times 1}\right)^{1/3} = \left(\frac{100}{10^{-6}}\right)^{1/3} = (10^8)^{1/3} = 464 \text{ seconds} \approx 8 \text{ minutes}$$

Common industry settings:

| Token Type | Typical Lifetime | Use Case |
|-----------|-----------------|----------|
| Access Token | 5-30 minutes | API authorization |
| ID Token | 5-60 minutes | Authentication proof |
| Refresh Token | 7-30 days | Session continuation |
| Authorization Code | 30-600 seconds | One-time exchange |

## 3. Nonce and PKCE Entropy Requirements (Information Theory)

### The Problem

The `nonce` parameter prevents replay attacks, and the PKCE `code_verifier` prevents authorization code interception. How much entropy is needed for these to be secure?

### The Formula

**Nonce entropy requirement**: the nonce must be unguessable by an attacker who observes $q$ authentication requests.

Collision probability for random nonces of $b$ bits after $q$ uses (birthday attack):

$$P(\text{collision}) \approx 1 - e^{-q^2 / 2^{b+1}}$$

For $P < 2^{-32}$ (negligible collision probability) with $q = 2^{32}$ requests:

$$2^{-32} \geq 1 - e^{-2^{64} / 2^{b+1}}$$

$$b \geq 128 \text{ bits}$$

**PKCE code_verifier** (RFC 7636): must have at least 256 bits of entropy.

From the RFC: code_verifier is 43-128 characters from `[A-Z, a-z, 0-9, -, ., _, ~]` (66 characters).

Entropy per character: $\log_2(66) = 6.04$ bits.

For 43 characters: $43 \times 6.04 = 259.8$ bits $\geq 256$ bits. (This is why 43 is the minimum length.)

**Guessing probability** for 256-bit entropy:

$$P(\text{guess in } q \text{ attempts}) = \frac{q}{2^{256}}$$

For $q = 2^{80}$ attempts (massive computational effort):

$$P = \frac{2^{80}}{2^{256}} = 2^{-176} \approx 0$$

### Worked Examples

**Example**: Comparing nonce generation methods:

| Method | Entropy | Secure? |
|--------|---------|---------|
| `Math.random()` (JS) | ~52 bits (double mantissa) | NO |
| UUID v4 | 122 bits | Marginal |
| `crypto.randomBytes(16)` | 128 bits | YES |
| `crypto.randomBytes(32)` | 256 bits | YES (recommended) |
| `secrets.token_urlsafe(32)` (Python) | 256 bits | YES |

**State parameter**: same entropy requirements as nonce. Additionally, must be bound to the user's session to prevent CSRF.

## 4. Session Fixation Prevention (Attack Graph Analysis)

### The Problem

Session fixation attacks occur when an attacker sets a known session ID before the user authenticates. OIDC mitigates this through several mechanisms. How effective are they?

### The Formula

**Session fixation attack probability** without mitigation:

$$P(\text{attack}) = P(\text{attacker sets session}) \times P(\text{user authenticates with fixed session})$$

With mitigation (session regeneration on authentication):

$$P(\text{attack}) = P(\text{attacker sets session}) \times P(\text{user authenticates}) \times P(\text{session not regenerated})$$

OIDC's `nonce` parameter provides implicit session fixation prevention:

1. RP generates random nonce, stores it in session
2. Nonce is included in the ID token
3. RP verifies ID token nonce matches stored nonce
4. Attacker cannot predict the nonce value

**Probability of successful session fixation with OIDC nonce** ($b$-bit nonce):

$$P(\text{fixation}) = P(\text{set session}) \times \frac{1}{2^b}$$

For $b = 128$: $P(\text{fixation}) = P(\text{set session}) \times 2^{-128} \approx 0$.

**Combined defenses** (defense in depth):

| Defense | Reduction Factor |
|---------|-----------------|
| Nonce verification | $2^{-128}$ |
| State parameter (CSRF) | $2^{-128}$ |
| Session regeneration | Eliminates fixed session |
| PKCE | Prevents code interception |

The combined probability of bypassing all defenses:

$$P(\text{bypass all}) \leq 2^{-128} \times 2^{-128} = 2^{-256}$$

### Worked Examples

**Example**: Without OIDC protections (basic session cookie auth):

- $P(\text{set session via XSS}) = 0.01$ (depends on XSS presence)
- $P(\text{user authenticates}) = 0.5$ (50% of visitors log in)
- $P(\text{session fixation}) = 0.01 \times 0.5 = 0.005$

With OIDC nonce (128-bit):
- $P(\text{session fixation}) = 0.005 \times 2^{-128} \approx 1.47 \times 10^{-41}$

Effectively zero.

## 5. Token Refresh Timing (Optimization)

### The Problem

When should the client refresh an access token? Too early wastes resources; too late causes failed requests.

### The Formula

Let $L$ be the token lifetime, $T_r$ be the refresh request latency, and $T_c$ be the clock skew between client and server.

**Safe refresh window**: refresh when remaining lifetime $\leq T_r + T_c + \epsilon$ (safety margin):

$$t_{refresh} = t_{issued} + L - T_r - T_c - \epsilon$$

**Proactive refresh probability**: if requests arrive at rate $\lambda$, the probability of needing the token during the refresh window:

$$P(\text{need token during refresh}) = 1 - e^{-\lambda \cdot T_r}$$

For $\lambda = 10$ req/s and $T_r = 0.5$s: $P = 1 - e^{-5} = 0.993$.

This means you should refresh **before** the token expires, not in response to a 401.

**Optimal refresh strategy**: maintain two tokens (current + next) and refresh when:

$$t_{remaining} \leq \max(2T_r, \frac{L}{10})$$

### Worked Examples

**Example**: Access token lifetime $L = 300$s, refresh latency $T_r = 0.5$s, clock skew $T_c = 5$s, safety margin $\epsilon = 10$s.

$$t_{refresh} = t_{issued} + 300 - 0.5 - 5 - 10 = t_{issued} + 284.5\text{s}$$

Refresh at 95% of token lifetime. More aggressive: $L/10 = 30$s buffer, refresh at $t_{issued} + 270$s (90% of lifetime).

## Prerequisites

- Modular arithmetic (RSA, ECDSA operations)
- Elliptic curve cryptography (point multiplication, curve groups)
- Probability theory (birthday problem, collision analysis)
- Information theory (entropy, randomness requirements)

## Complexity

| Operation | Time Complexity | Space Complexity |
|-----------|----------------|-----------------|
| RSA-2048 signing | $O(k^3)$ where $k$ = key bits | $O(k)$ |
| RSA-2048 verification | $O(k^2)$ (small exponent) | $O(k)$ |
| ECDSA-P256 signing | $O(k^3)$ field operations | $O(k)$ |
| ECDSA-P256 verification | $O(k^3)$ field operations | $O(k)$ |
| JWKS cache lookup | $O(1)$ hash table | $O(n_{keys})$ |
| Token validation (full) | $O(k^2 + c)$ where $c$ = claims | $O(t)$ token size |

Where: $k$ = key size in bits, $n_{keys}$ = keys in JWKS, $c$ = number of claims, $t$ = token size in bytes.
