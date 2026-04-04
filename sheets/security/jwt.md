# JWT (JSON Web Tokens)

Compact, URL-safe token format encoding claims as a three-part Base64URL structure (Header.Payload.Signature), used for stateless authentication and authorization in web APIs.

## Token Structure

### Three Parts

```
eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.     # Header (base64url)
eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6Ikpv.  # Payload (base64url)
SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQss.  # Signature

# Concatenated with dots: HEADER.PAYLOAD.SIGNATURE
# Total: typically 800-2000 bytes
```

### Header

```json
{
  "alg": "RS256",         // Signing algorithm
  "typ": "JWT",           // Token type
  "kid": "key-2024-01"   // Key ID (for key rotation)
}
```

### Payload (Claims)

```json
{
  "iss": "https://auth.example.com",     // Issuer
  "sub": "user-12345",                   // Subject (user ID)
  "aud": "https://api.example.com",      // Audience (intended recipient)
  "exp": 1735689600,                     // Expiration (Unix timestamp)
  "nbf": 1735686000,                     // Not Before
  "iat": 1735686000,                     // Issued At
  "jti": "unique-token-id-abc",          // JWT ID (unique identifier)

  // Custom claims
  "roles": ["admin", "user"],
  "tenant_id": "org-456",
  "scope": "read write"
}
```

### Registered Claims (RFC 7519)

```
iss    # Issuer — who created the token
sub    # Subject — who the token is about
aud    # Audience — who the token is for (string or array)
exp    # Expiration Time — Unix timestamp, MUST be validated
nbf    # Not Before — token not valid before this time
iat    # Issued At — when token was created
jti    # JWT ID — unique identifier to prevent replay
```

## Signing Algorithms

### Symmetric (HMAC)

```
HS256    # HMAC-SHA256 — shared secret, both sides know key
HS384    # HMAC-SHA384
HS512    # HMAC-SHA512

# Signature = HMAC-SHA256(base64url(header) + "." + base64url(payload), secret)

# Use case: single service signs AND verifies
# Risk: secret must be shared with every verifier
```

### Asymmetric (RSA / ECDSA)

```
RS256    # RSA-SHA256 (2048+ bit keys) — most common
RS384    # RSA-SHA384
RS512    # RSA-SHA512
ES256    # ECDSA P-256 + SHA256 — smaller keys, faster
ES384    # ECDSA P-384 + SHA384
ES512    # ECDSA P-521 + SHA512
PS256    # RSA-PSS + SHA256 (probabilistic padding)
PS384    # RSA-PSS + SHA384
PS512    # RSA-PSS + SHA512

# Private key signs, public key verifies
# Use case: auth server signs, many services verify
# Preferred for microservices architectures
```

### EdDSA

```
EdDSA    # Ed25519/Ed448 — modern, fast, small signatures
         # Ed25519: 32-byte keys, 64-byte signatures
         # Recommended for new systems
```

## Key Management (JWKS)

### JSON Web Key Set

```bash
# Fetch public keys from auth server
curl https://auth.example.com/.well-known/jwks.json

# Response:
{
  "keys": [
    {
      "kty": "RSA",
      "kid": "key-2024-01",
      "use": "sig",
      "alg": "RS256",
      "n": "0vx7agoebGcQ...",    // Modulus (base64url)
      "e": "AQAB"                  // Exponent (base64url)
    },
    {
      "kty": "EC",
      "kid": "key-2024-02",
      "use": "sig",
      "alg": "ES256",
      "crv": "P-256",
      "x": "f83OJ3D2xF1Bg...",
      "y": "x_FEzRu9m36HLN..."
    }
  ]
}
```

### Key Rotation

```bash
# 1. Generate new key pair
openssl genpkey -algorithm RSA -out private-new.pem -pkeyopt rsa_keygen_bits:2048
openssl rsa -in private-new.pem -pubout -out public-new.pem

# 2. Add new key to JWKS with new kid
# 3. Start signing with new key
# 4. Keep old key in JWKS until all old tokens expire
# 5. Remove old key from JWKS

# Key rotation period: match max token lifetime + clock skew buffer
```

## Verification

### Verification Checklist

```
1. Decode header (no verification needed — it's just base64url)
2. Check "alg" — MUST match expected algorithm (prevent alg:none attack)
3. Find signing key using "kid" from JWKS endpoint
4. Verify signature using public key / shared secret
5. Check "exp" — reject if current_time > exp (with clock skew tolerance)
6. Check "nbf" — reject if current_time < nbf
7. Check "iss" — must match expected issuer
8. Check "aud" — must contain this service's identifier
9. Check custom claims (roles, scopes, tenant) as needed
```

### Command-Line Tools

```bash
# Decode JWT (no verification)
echo 'eyJhbGci...' | cut -d. -f2 | base64 -d 2>/dev/null | jq .

# Decode with jq (all parts)
jwt="eyJhbGci..."
echo $jwt | cut -d. -f1 | base64 -d 2>/dev/null | jq .   # Header
echo $jwt | cut -d. -f2 | base64 -d 2>/dev/null | jq .   # Payload

# Verify with openssl (RS256)
header_payload=$(echo -n "${jwt}" | cut -d. -f1-2)
signature=$(echo -n "${jwt}" | cut -d. -f3 | tr '_-' '/+' | base64 -d)
echo -n "$header_payload" | openssl dgst -sha256 -verify public.pem \
  -signature <(echo -n "$signature")

# Generate HS256 JWT with openssl
header='{"alg":"HS256","typ":"JWT"}'
payload='{"sub":"1234","exp":1735689600}'
h=$(echo -n "$header" | base64 | tr '/+' '_-' | tr -d '=')
p=$(echo -n "$payload" | base64 | tr '/+' '_-' | tr -d '=')
sig=$(echo -n "${h}.${p}" | openssl dgst -sha256 -hmac "secret" -binary \
  | base64 | tr '/+' '_-' | tr -d '=')
echo "${h}.${p}.${sig}"
```

### Go Verification

```go
import (
    "github.com/golang-jwt/jwt/v5"
)

// Parse and validate RS256 token
token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
    // Verify algorithm
    if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
        return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
    }
    return publicKey, nil   // *rsa.PublicKey
}, jwt.WithAudience("https://api.example.com"),
   jwt.WithIssuer("https://auth.example.com"),
   jwt.WithExpirationRequired())

if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
    sub := claims["sub"].(string)
}
```

## Common Attacks

### Algorithm Confusion

```
# Attack: change alg from RS256 to HS256
# Attacker uses public key as HMAC secret
# Prevention: ALWAYS validate alg matches expected value

# Attack: set alg to "none"
# Prevention: reject tokens with alg=none, always require signature
```

### Token Theft and Replay

```
# Prevention:
# - Short expiration (5-15 min access tokens)
# - Use jti claim with server-side tracking for one-time tokens
# - Bind tokens to client (fingerprint, DPoP)
# - Transmit only over TLS
# - Store in HttpOnly cookies (not localStorage)
```

## Tips

- Never store sensitive data in JWT payloads; they are base64url-encoded, not encrypted
- Always validate the `alg` header against expected values to prevent algorithm confusion attacks
- Use RS256 or ES256 for microservices so only the auth server needs the private key
- Keep access tokens short-lived (5-15 minutes) and use refresh tokens for session continuity
- The `aud` claim is critical in multi-service architectures to prevent token misuse across services
- Cache JWKS responses but implement a refresh mechanism when `kid` is not found (key rotation)
- Clock skew tolerance of 30-60 seconds prevents false rejections across distributed systems
- JWTs cannot be revoked without server-side state; use token blacklists or short expiry for revocation
- Prefer ES256 over RS256 for new systems: smaller tokens, faster verification, equivalent security
- Never use the `none` algorithm in production; treat it as a vulnerability
- Consider JWE (JSON Web Encryption) when payload confidentiality is required, not just integrity
- Use structured `scope` or `permissions` claims instead of broad role-based claims for fine-grained access

## See Also

oauth, tls, openssl, pki, cryptography, cors

## References

- [RFC 7519 — JSON Web Token (JWT)](https://datatracker.ietf.org/doc/html/rfc7519)
- [RFC 7515 — JSON Web Signature (JWS)](https://datatracker.ietf.org/doc/html/rfc7515)
- [RFC 7517 — JSON Web Key (JWK)](https://datatracker.ietf.org/doc/html/rfc7517)
- [RFC 7518 — JSON Web Algorithms (JWA)](https://datatracker.ietf.org/doc/html/rfc7518)
- [jwt.io — JWT Debugger](https://jwt.io/)
- [JWT Security Best Practices (RFC 8725)](https://datatracker.ietf.org/doc/html/rfc8725)
