# OpenID Connect (OIDC Authentication)

Complete reference for OIDC — flows (authorization code + PKCE, client credentials), ID tokens, JWT structure, standard claims, discovery, token validation, provider configuration, session management, and logout.

## Overview

```
OIDC = OAuth 2.0 + Identity Layer

OAuth 2.0: "Can this app access this resource on behalf of the user?"
OIDC:      "Who is this user?" (adds ID token, UserInfo, standard claims)

OIDC is built ON TOP of OAuth 2.0, not a replacement.
```

### Key Terminology

| Term | Description |
|------|-------------|
| OP (OpenID Provider) | The identity server (Keycloak, Auth0, Okta, Google) |
| RP (Relying Party) | Your application — relies on the OP for authentication |
| ID Token | JWT containing user identity claims |
| Access Token | Token for accessing protected resources (API) |
| Refresh Token | Long-lived token to obtain new access tokens |
| UserInfo Endpoint | API to get additional user claims |
| Scope | Permission requested (openid, profile, email, etc.) |

## Flows

### Authorization Code + PKCE (Recommended)

This is the recommended flow for all public clients (SPAs, mobile apps, CLIs) and confidential clients.

```
1. RP generates code_verifier (random 43-128 chars)
2. RP computes code_challenge = BASE64URL(SHA256(code_verifier))

3. RP redirects user to OP:
   GET /authorize?
     response_type=code&
     client_id=my-app&
     redirect_uri=https://app.example.com/callback&
     scope=openid profile email&
     state=abc123&
     nonce=xyz789&
     code_challenge=E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM&
     code_challenge_method=S256

4. User authenticates at OP

5. OP redirects to RP with authorization code:
   GET /callback?code=AUTH_CODE&state=abc123

6. RP exchanges code for tokens (back-channel):
   POST /token
     grant_type=authorization_code&
     code=AUTH_CODE&
     redirect_uri=https://app.example.com/callback&
     client_id=my-app&
     code_verifier=dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk

7. OP returns:
   {
     "access_token": "eyJ...",
     "id_token": "eyJ...",
     "refresh_token": "eyJ...",
     "token_type": "Bearer",
     "expires_in": 3600
   }
```

### Client Credentials (Machine-to-Machine)

No user involved. Service authenticates as itself.

```
POST /token
  grant_type=client_credentials&
  client_id=my-service&
  client_secret=SECRET&
  scope=api.read api.write

Response:
{
  "access_token": "eyJ...",
  "token_type": "Bearer",
  "expires_in": 3600
}
```

No ID token is returned (no user identity).

### Implicit Flow (DEPRECATED)

```
DO NOT USE. Tokens are exposed in URL fragments.
Replaced by Authorization Code + PKCE.
```

### Hybrid Flow

```
response_type=code id_token
Returns both an authorization code and an ID token from the
authorization endpoint. Used when the RP needs immediate identity
verification before the back-channel token exchange.
```

## PKCE (Proof Key for Code Exchange)

### Generation

```go
// Go implementation
func generatePKCE() (verifier, challenge string, err error) {
    // code_verifier: 43-128 characters from [A-Z, a-z, 0-9, -, ., _, ~]
    buf := make([]byte, 32)
    if _, err := rand.Read(buf); err != nil {
        return "", "", err
    }
    verifier = base64.RawURLEncoding.EncodeToString(buf)

    // code_challenge: BASE64URL(SHA256(code_verifier))
    h := sha256.Sum256([]byte(verifier))
    challenge = base64.RawURLEncoding.EncodeToString(h[:])

    return verifier, challenge, nil
}
```

```python
# Python implementation
import hashlib, base64, secrets

code_verifier = secrets.token_urlsafe(32)  # 43 chars
code_challenge = base64.urlsafe_b64encode(
    hashlib.sha256(code_verifier.encode()).digest()
).rstrip(b'=').decode()
```

```bash
# CLI implementation
CODE_VERIFIER=$(openssl rand -base64 32 | tr -d '=/+' | head -c 43)
CODE_CHALLENGE=$(echo -n "$CODE_VERIFIER" | openssl dgst -sha256 -binary | base64 | tr -d '=' | tr '/+' '_-')
```

## ID Token (JWT)

### Structure

```
eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6InJzYTEifQ
.
eyJpc3MiOiJodHRwczovL2F1dGguZXhhbXBsZS5jb20iLCJzdWIiOiJ1c2VyXzEyMyIsImF1ZCI6Im15LWFwcCIsImV4cCI6MTcwOTEyMDAwMCwiaWF0IjoxNzA5MTE2NDAwLCJub25jZSI6Inh5ejc4OSIsImVtYWlsIjoiYWxpY2VAZXhhbXBsZS5jb20ifQ
.
SIGNATURE

Header.Payload.Signature (base64url-encoded, separated by dots)
```

### Header

```json
{
  "alg": "RS256",       // signing algorithm
  "typ": "JWT",         // token type
  "kid": "rsa1"         // key ID (matches JWKS)
}
```

### Standard Claims

```json
{
  "iss": "https://auth.example.com",   // issuer — MUST match expected
  "sub": "user_123",                    // subject — unique user ID
  "aud": "my-app",                      // audience — MUST match client_id
  "exp": 1709120000,                    // expiration (unix timestamp)
  "iat": 1709116400,                    // issued at
  "auth_time": 1709116380,             // when user actually authenticated
  "nonce": "xyz789",                    // MUST match sent nonce (replay prevention)
  "at_hash": "fUHyO2r2Z3DZ53EsNrWBb0xWXoaNy59IiKCAqksmQEo",
  "acr": "urn:mace:incommon:iap:silver", // authentication context class
  "amr": ["pwd", "mfa"],               // authentication methods
  "azp": "my-app",                      // authorized party

  // Standard scopes add these claims:
  "name": "Alice Smith",                // scope: profile
  "given_name": "Alice",                // scope: profile
  "family_name": "Smith",              // scope: profile
  "email": "alice@example.com",        // scope: email
  "email_verified": true,              // scope: email
  "phone_number": "+1-555-0123",       // scope: phone
  "address": {                         // scope: address
    "formatted": "123 Main St"
  }
}
```

### Standard Scopes

| Scope | Claims Added |
|-------|-------------|
| `openid` | `sub` (required for OIDC) |
| `profile` | `name`, `family_name`, `given_name`, `middle_name`, `nickname`, `preferred_username`, `profile`, `picture`, `website`, `gender`, `birthdate`, `zoneinfo`, `locale`, `updated_at` |
| `email` | `email`, `email_verified` |
| `address` | `address` |
| `phone` | `phone_number`, `phone_number_verified` |
| `offline_access` | Requests a refresh token |

## Discovery Endpoint

### .well-known/openid-configuration

```bash
curl https://auth.example.com/.well-known/openid-configuration | jq .
```

```json
{
  "issuer": "https://auth.example.com",
  "authorization_endpoint": "https://auth.example.com/authorize",
  "token_endpoint": "https://auth.example.com/token",
  "userinfo_endpoint": "https://auth.example.com/userinfo",
  "jwks_uri": "https://auth.example.com/.well-known/jwks.json",
  "registration_endpoint": "https://auth.example.com/register",
  "scopes_supported": ["openid", "profile", "email", "address", "phone"],
  "response_types_supported": ["code", "id_token", "code id_token"],
  "grant_types_supported": ["authorization_code", "client_credentials", "refresh_token"],
  "subject_types_supported": ["public", "pairwise"],
  "id_token_signing_alg_values_supported": ["RS256", "ES256"],
  "token_endpoint_auth_methods_supported": ["client_secret_basic", "client_secret_post", "private_key_jwt"],
  "code_challenge_methods_supported": ["S256"],
  "claims_supported": ["sub", "iss", "aud", "exp", "iat", "name", "email"]
}
```

## Token Validation

### Required Checks

```go
func ValidateIDToken(tokenString string, config *OIDCConfig) (*Claims, error) {
    // 1. Parse and verify signature using JWKS
    token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
        kid := token.Header["kid"].(string)
        return getPublicKeyFromJWKS(config.JWKSURL, kid)
    })
    if err != nil {
        return nil, fmt.Errorf("signature verification failed: %w", err)
    }

    claims := token.Claims.(jwt.MapClaims)

    // 2. Verify issuer
    if claims["iss"] != config.Issuer {
        return nil, fmt.Errorf("invalid issuer: %s", claims["iss"])
    }

    // 3. Verify audience
    if !claims.VerifyAudience(config.ClientID, true) {
        return nil, fmt.Errorf("invalid audience")
    }

    // 4. Verify expiration
    if !claims.VerifyExpiresAt(time.Now().Unix(), true) {
        return nil, fmt.Errorf("token expired")
    }

    // 5. Verify issued-at is not in the future
    if !claims.VerifyIssuedAt(time.Now().Unix(), true) {
        return nil, fmt.Errorf("token issued in the future")
    }

    // 6. Verify nonce (if sent in auth request)
    if config.ExpectedNonce != "" && claims["nonce"] != config.ExpectedNonce {
        return nil, fmt.Errorf("invalid nonce")
    }

    // 7. Verify authorized party (if aud contains multiple values)
    aud, _ := claims["aud"].([]interface{})
    if len(aud) > 1 {
        if claims["azp"] != config.ClientID {
            return nil, fmt.Errorf("invalid authorized party")
        }
    }

    return parseClaims(claims), nil
}
```

### JWKS Fetching

```go
type JWKS struct {
    Keys []JWK `json:"keys"`
}

type JWK struct {
    Kty string `json:"kty"` // key type: RSA, EC
    Kid string `json:"kid"` // key ID
    Use string `json:"use"` // sig (signature)
    N   string `json:"n"`   // RSA modulus
    E   string `json:"e"`   // RSA exponent
    Crv string `json:"crv"` // EC curve: P-256
    X   string `json:"x"`   // EC x coordinate
    Y   string `json:"y"`   // EC y coordinate
}

func getPublicKeyFromJWKS(jwksURL, kid string) (interface{}, error) {
    resp, err := http.Get(jwksURL)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var jwks JWKS
    json.NewDecoder(resp.Body).Decode(&jwks)

    for _, key := range jwks.Keys {
        if key.Kid == kid {
            return parseJWK(key)
        }
    }
    return nil, fmt.Errorf("key %s not found in JWKS", kid)
}
```

## UserInfo Endpoint

```bash
curl -H "Authorization: Bearer ACCESS_TOKEN" \
    https://auth.example.com/userinfo

{
  "sub": "user_123",
  "name": "Alice Smith",
  "email": "alice@example.com",
  "email_verified": true,
  "picture": "https://example.com/alice.jpg"
}
```

## Session Management and Logout

### RP-Initiated Logout

```
GET /logout?
  id_token_hint=eyJ...&
  post_logout_redirect_uri=https://app.example.com/logged-out&
  state=abc123
```

### Back-Channel Logout

The OP sends a logout token to the RP's back-channel endpoint:

```
POST /backchannel-logout
Content-Type: application/x-www-form-urlencoded

logout_token=eyJ...
```

The logout token contains:
```json
{
  "iss": "https://auth.example.com",
  "sub": "user_123",
  "aud": "my-app",
  "iat": 1709116400,
  "jti": "unique-token-id",
  "events": {
    "http://schemas.openid.net/event/backchannel-logout": {}
  },
  "sid": "session_abc"
}
```

### Front-Channel Logout

The OP renders an iframe pointing to the RP's logout URL:

```html
<iframe src="https://app.example.com/front-channel-logout?sid=session_abc&iss=https://auth.example.com"></iframe>
```

## Provider Examples

### Keycloak

```bash
# Discovery
curl https://keycloak.example.com/realms/myrealm/.well-known/openid-configuration

# Common endpoints
# Authorize: /realms/{realm}/protocol/openid-connect/auth
# Token:     /realms/{realm}/protocol/openid-connect/token
# UserInfo:  /realms/{realm}/protocol/openid-connect/userinfo
# Logout:    /realms/{realm}/protocol/openid-connect/logout
# JWKS:      /realms/{realm}/protocol/openid-connect/certs
```

### Auth0

```bash
# Discovery
curl https://YOUR_DOMAIN.auth0.com/.well-known/openid-configuration

# Authorize: https://YOUR_DOMAIN.auth0.com/authorize
# Token:     https://YOUR_DOMAIN.auth0.com/oauth/token
# UserInfo:  https://YOUR_DOMAIN.auth0.com/userinfo
# JWKS:      https://YOUR_DOMAIN.auth0.com/.well-known/jwks.json
```

### Google

```bash
# Discovery
curl https://accounts.google.com/.well-known/openid-configuration

# Authorize: https://accounts.google.com/o/oauth2/v2/auth
# Token:     https://oauth2.googleapis.com/token
# JWKS:      https://www.googleapis.com/oauth2/v3/certs
```

## Tips

- Always use Authorization Code + PKCE — even for confidential clients (defense in depth)
- Never store tokens in localStorage — use httpOnly secure cookies or in-memory
- Validate ALL required fields on the ID token — issuer, audience, expiry, nonce
- Cache JWKS responses with appropriate TTL (5-15 minutes) to avoid latency on every request
- Use the `nonce` parameter to prevent replay attacks in implicit/hybrid flows
- The `state` parameter prevents CSRF — generate a random value and verify on callback
- `sub` is the only stable user identifier — `email` can change
- Use `offline_access` scope to request refresh tokens
- Implement token refresh logic before tokens expire (use `exp` claim minus buffer)
- Back-channel logout is more reliable than front-channel (no browser dependency)

## See Also

- `detail/auth/oidc.md` — JWT signature mathematics, token lifetime optimization
- `sheets/quality/twelve-factor.md` — Factor III (Config) for managing OIDC secrets

## References

- https://openid.net/specs/openid-connect-core-1_0.html — OIDC Core Specification
- https://datatracker.ietf.org/doc/html/rfc7636 — PKCE (RFC 7636)
- https://datatracker.ietf.org/doc/html/rfc7519 — JWT (RFC 7519)
- https://openid.net/specs/openid-connect-discovery-1_0.html — OIDC Discovery
- https://openid.net/specs/openid-connect-backchannel-1_0.html — Back-Channel Logout
