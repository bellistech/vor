# The Mathematics of JWT — Signature Schemes and Claims Verification Logic

> *A JWT is a triple of Base64URL-encoded structures whose integrity rests on digital signature theory. Verification is a predicate logic conjunction over cryptographic validity and temporal claim bounds, while algorithm selection determines the trust model topology.*

---

## 1. Token Structure as Concatenated Encoding (Coding Theory)

### Base64URL Encoding

Each JWT segment uses Base64URL (RFC 4648 Section 5):

$$\text{Base64URL}: \{0,1\}^* \rightarrow \{A\text{-}Z, a\text{-}z, 0\text{-}9, \text{-}, \_\}^*$$

Encoding expansion ratio:

$$r = \frac{|\text{encoded}|}{|\text{input}|} = \frac{4}{3} \approx 1.333$$

For a JWT with header $h$ bytes, payload $p$ bytes, and signature $s$ bytes:

$$|JWT| = \lceil \frac{4h}{3} \rceil + 1 + \lceil \frac{4p}{3} \rceil + 1 + \lceil \frac{4s}{3} \rceil$$

### Typical JWT Sizes

| Algorithm | Header | Payload (10 claims) | Signature | Total |
|:---|:---:|:---:|:---:|:---:|
| HS256 | 36 bytes | ~200 bytes | 43 bytes | ~370 chars |
| RS256 | 50 bytes | ~200 bytes | 342 bytes | ~790 chars |
| ES256 | 50 bytes | ~200 bytes | 86 bytes | ~450 chars |
| EdDSA | 50 bytes | ~200 bytes | 86 bytes | ~450 chars |

---

## 2. HMAC Signature Scheme (Symmetric Cryptography)

### HS256 Construction

HMAC-SHA256 follows the construction from RFC 2104:

$$\text{HMAC}(K, m) = H((K \oplus \text{opad}) \| H((K \oplus \text{ipad}) \| m))$$

Where:
- $K$ = secret key (padded to block size $B = 64$ bytes)
- $\text{ipad}$ = 0x36 repeated $B$ times
- $\text{opad}$ = 0x5C repeated $B$ times
- $H$ = SHA-256, $m$ = `header.payload`

### Security Properties

**Unforgeability:** For an attacker making $q$ verification queries:

$$\text{Adv}^{\text{MAC}}_{\text{HMAC-SHA256}}(t, q) \leq \frac{q}{2^{256}} + \epsilon_{\text{PRF}}(t)$$

**Key length requirements:**

$$|K| \geq 256 \text{ bits (to match hash output length)}$$

If $|K| < 256$ bits, brute force time:

$$T_{\text{brute}} = \frac{2^{|K|}}{R}$$

Where $R$ is the verification rate. At $R = 10^9$/s and $|K| = 128$:

$$T = \frac{2^{128}}{10^9} \approx 10^{29} \text{ seconds}$$

---

## 3. RSA Signature Scheme (Asymmetric Cryptography)

### RS256 (RSASSA-PKCS1-v1_5)

Signing with private key $(n, d)$:

$$\sigma = m^d \bmod n$$

Where $m = \text{EMSA-PKCS1-v1\_5-ENCODE}(H(\text{header.payload}), |n|)$

Verification with public key $(n, e)$:

$$m' = \sigma^e \bmod n$$

$$\text{valid} \iff m' = \text{EMSA-PKCS1-v1\_5-ENCODE}(H(\text{header.payload}), |n|)$$

### RSA Key Size and Security

| Key Size (bits) | Security Level (bits) | Status |
|:---:|:---:|:---|
| 1024 | ~80 | Broken (factorable) |
| 2048 | ~112 | Minimum acceptable |
| 3072 | ~128 | Recommended |
| 4096 | ~140 | High security |

Factoring difficulty (GNFS complexity):

$$T_{\text{factor}}(n) = \exp\left(\left(\frac{64}{9}\right)^{1/3} (\ln n)^{1/3} (\ln \ln n)^{2/3}\right)$$

### Performance Comparison

| Operation | RS256 (2048-bit) | ES256 (P-256) | EdDSA (Ed25519) |
|:---|:---:|:---:|:---:|
| Sign | ~1.5 ms | ~0.1 ms | ~0.05 ms |
| Verify | ~0.05 ms | ~0.3 ms | ~0.1 ms |
| Signature size | 256 bytes | 64 bytes | 64 bytes |
| Key size (public) | 256 bytes | 64 bytes | 32 bytes |

---

## 4. ECDSA Signature Scheme (Elliptic Curve Cryptography)

### ES256 on P-256

The curve P-256 (secp256r1):

$$y^2 = x^3 - 3x + b \pmod{p}$$

Where $p = 2^{256} - 2^{224} + 2^{192} + 2^{96} - 1$.

**Signing** with private key $d$:
1. Choose random $k \xleftarrow{R} [1, n-1]$
2. Compute $(x_1, y_1) = k \cdot G$
3. $r = x_1 \bmod n$
4. $s = k^{-1}(H(m) + r \cdot d) \bmod n$
5. Signature $= (r, s)$

**Verification** with public key $Q = d \cdot G$:
1. $w = s^{-1} \bmod n$
2. $(x_1, y_1) = H(m) \cdot w \cdot G + r \cdot w \cdot Q$
3. Valid $\iff r \equiv x_1 \pmod{n}$

### Nonce Criticality

If nonce $k$ is reused for two messages $m_1, m_2$:

$$s_1 = k^{-1}(H(m_1) + r \cdot d) \bmod n$$
$$s_2 = k^{-1}(H(m_2) + r \cdot d) \bmod n$$

Then:

$$k = \frac{H(m_1) - H(m_2)}{s_1 - s_2} \bmod n$$

$$d = \frac{s_1 \cdot k - H(m_1)}{r} \bmod n$$

The private key is recoverable from a single nonce reuse (as happened with Sony PlayStation 3 signing key).

---

## 5. Claims Verification as Predicate Logic (Formal Logic)

### Verification Predicate

JWT verification is a conjunction of predicates:

$$\text{valid}(T) = P_{\text{sig}}(T) \wedge P_{\text{exp}}(T) \wedge P_{\text{nbf}}(T) \wedge P_{\text{iss}}(T) \wedge P_{\text{aud}}(T) \wedge P_{\text{alg}}(T)$$

Each predicate:

$$P_{\text{sig}}(T) \iff \text{Verify}(\text{key}, \text{header.payload}, \text{signature})$$

$$P_{\text{exp}}(T) \iff t_{\text{now}} \leq T.\text{exp} + \epsilon_{\text{skew}}$$

$$P_{\text{nbf}}(T) \iff t_{\text{now}} \geq T.\text{nbf} - \epsilon_{\text{skew}}$$

$$P_{\text{iss}}(T) \iff T.\text{iss} \in \mathcal{I}_{\text{trusted}}$$

$$P_{\text{aud}}(T) \iff T.\text{aud} \cap \mathcal{A}_{\text{self}} \neq \emptyset$$

$$P_{\text{alg}}(T) \iff T.\text{alg} \in \mathcal{G}_{\text{allowed}}$$

### Clock Skew Tolerance

With skew tolerance $\epsilon$ (typically 30-60 seconds):

$$\text{valid window} = [T.\text{nbf} - \epsilon, \; T.\text{exp} + \epsilon]$$

$$\text{window length} = (T.\text{exp} - T.\text{nbf}) + 2\epsilon$$

For 15-minute tokens with 60s skew:

$$\text{effective lifetime} = 900 + 120 = 1020 \text{ seconds} = 17 \text{ minutes}$$

---

## 6. Algorithm Confusion Attack (Attack Theory)

### The alg:none Attack

Attacker modifies header:

$$\text{header} = \{\text{"alg": "none", "typ": "JWT"}\}$$

If the verifier accepts `alg=none`:

$$P_{\text{sig}}(T) = \text{true} \quad \forall T$$

The verification degenerates to claim-only checking.

### RSA/HMAC Confusion

If server expects RS256 (asymmetric) but accepts HS256:

1. Attacker obtains public key $K_{\text{pub}}$ (often from JWKS)
2. Creates token with `alg: HS256`
3. Signs with $\text{HMAC}(K_{\text{pub}}, \text{header.payload})$
4. Server uses $K_{\text{pub}}$ as HMAC secret (since alg says HS256)
5. Verification succeeds

Prevention: strict algorithm allowlisting per key:

$$\text{key\_to\_alg}: \text{KeyID} \rightarrow \{\text{allowed algorithms}\}$$

$$P_{\text{alg}}(T) \iff T.\text{alg} \in \text{key\_to\_alg}(T.\text{kid})$$

---

## 7. Revocation Without State (Distributed Systems)

### The Stateless Revocation Problem

JWTs are designed to be stateless, but revocation requires state. This is a fundamental tension:

$$\text{stateless verification}: O(1) \text{ time, } O(0) \text{ server state}$$
$$\text{revocation check}: O(1) \text{ time, } O(n) \text{ server state}$$

### Probabilistic Revocation (Bloom Filters)

A Bloom filter for revoked JTIs:

$$P(\text{false positive}) = \left(1 - e^{-kn/m}\right)^k$$

Where $k$ = hash functions, $n$ = revoked tokens, $m$ = filter bits.

For $n = 10^4$ revoked tokens, $P_{\text{fp}} = 0.01$:

$$m = -\frac{n \ln P_{\text{fp}}}{(\ln 2)^2} \approx 95,851 \text{ bits} \approx 12 \text{ KB}$$

$$k = \frac{m}{n} \ln 2 \approx 7$$

This allows $O(1)$ revocation checks with 12 KB of state instead of storing all revoked JTIs.

### Short-Lived Token Strategy

Maximum damage window from a stolen token:

$$W = \min(L, t_{\text{detect}})$$

Where $L$ = token lifetime and $t_{\text{detect}}$ = time to detect theft.

Expected loss: $E[\text{loss}] = \lambda \cdot W$ where $\lambda$ = damage rate.

Optimal lifetime minimizes $E[\text{total cost}]$:

$$L^* = \sqrt{\frac{C_{\text{refresh}}}{\lambda}} $$

---

## Prerequisites

base64-encoding, hmac, rsa-cryptography, elliptic-curves, predicate-logic, bloom-filters

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| HS256 sign/verify | $O(n)$ — n = payload size | $O(1)$ — 32-byte digest |
| RS256 sign | $O(k^3)$ — k = key bits | $O(k)$ — key storage |
| RS256 verify | $O(k^2)$ — exponent is small | $O(k)$ |
| ES256 sign | $O(k^2)$ — scalar multiply | $O(k)$ |
| ES256 verify | $O(k^2)$ — two point multiplies | $O(k)$ |
| Claims validation | $O(c)$ — c = claim count | $O(1)$ |
| Bloom filter check | $O(k)$ — k = hash functions | $O(m)$ — m = filter bits |
