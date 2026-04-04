# OAuth (OAuth 2.0 Authorization Framework)

OAuth 2.0 delegates authorization via access tokens, allowing third-party applications to access resources on behalf of a user without exposing credentials, using grant types tailored to different client types.

## Grant Types

### Authorization Code (Web Apps)

```
# Most secure — used by server-side web apps
# Requires client_secret stored on server

User-Agent        Client (Server)        Auth Server       Resource Server
    |--- Login Click --->|                    |                    |
    |                    |--- /authorize ---->|                    |
    |<--- 302 Login Page -------------------|                    |
    |--- Credentials --->|                    |                    |
    |                    |<--- code (302) ----|                    |
    |                    |--- POST /token --->|                    |
    |                    |    code +           |                    |
    |                    |    client_secret    |                    |
    |                    |<--- access_token ---|                    |
    |                    |--- GET /resource ---|---> API call ----->|
```

### Authorization Code + PKCE (SPAs, Mobile)

```
# No client_secret needed (public clients)
# PKCE prevents authorization code interception

1. Client generates:
   code_verifier  = random(43-128 chars, [A-Za-z0-9-._~])
   code_challenge = BASE64URL(SHA256(code_verifier))

2. Authorization request:
   GET /authorize?
     response_type=code&
     client_id=CLIENT_ID&
     redirect_uri=https://app.example.com/callback&
     scope=openid profile email&
     state=RANDOM_STATE&
     code_challenge=E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM&
     code_challenge_method=S256

3. Token exchange:
   POST /token
     grant_type=authorization_code&
     code=AUTHORIZATION_CODE&
     redirect_uri=https://app.example.com/callback&
     client_id=CLIENT_ID&
     code_verifier=dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk
```

### Client Credentials (Machine-to-Machine)

```bash
# Service-to-service, no user context
curl -X POST https://auth.example.com/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials" \
  -d "client_id=SERVICE_CLIENT_ID" \
  -d "client_secret=SERVICE_CLIENT_SECRET" \
  -d "scope=api.read api.write"

# Response
{
  "access_token": "eyJhbGciOi...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "api.read api.write"
}
```

### Device Authorization (IoT, CLI)

```bash
# For devices without browsers (RFC 8628)

# 1. Request device code
curl -X POST https://auth.example.com/device/code \
  -d "client_id=DEVICE_CLIENT_ID" \
  -d "scope=openid profile"

# Response:
# {
#   "device_code": "GmRhm...",
#   "user_code": "WDJB-MJHT",
#   "verification_uri": "https://auth.example.com/device",
#   "expires_in": 900,
#   "interval": 5
# }

# 2. User visits verification_uri, enters user_code

# 3. Device polls for token
curl -X POST https://auth.example.com/token \
  -d "grant_type=urn:ietf:params:oauth:grant-type:device_code" \
  -d "device_code=GmRhm..." \
  -d "client_id=DEVICE_CLIENT_ID"
```

## Refresh Token Flow

```bash
# Exchange refresh token for new access token
curl -X POST https://auth.example.com/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=refresh_token" \
  -d "refresh_token=REFRESH_TOKEN" \
  -d "client_id=CLIENT_ID" \
  -d "client_secret=CLIENT_SECRET"     # if confidential client

# Response includes new access_token (and optionally new refresh_token)
# Refresh token rotation: each use returns a NEW refresh token
# Old refresh token is invalidated — detects token theft
```

## Token Introspection (RFC 7662)

```bash
# Resource server validates token with auth server
curl -X POST https://auth.example.com/introspect \
  -u "resource_server:SECRET" \
  -d "token=ACCESS_TOKEN"

# Response
{
  "active": true,
  "scope": "openid profile email",
  "client_id": "CLIENT_ID",
  "username": "jdoe",
  "token_type": "Bearer",
  "exp": 1735689600,
  "iat": 1735686000,
  "sub": "user-12345",
  "aud": "https://api.example.com",
  "iss": "https://auth.example.com"
}
```

## OpenID Connect (OIDC)

### ID Token vs Access Token

```
Access Token    # Authorize API access (opaque or JWT)
ID Token        # Authenticate user identity (always JWT)
                # Contains user claims: sub, name, email, etc.

# OIDC adds to OAuth 2.0:
# - /.well-known/openid-configuration (discovery)
# - /userinfo endpoint
# - id_token response type
# - Standard claim names
```

### Discovery Endpoint

```bash
# Fetch provider metadata
curl https://auth.example.com/.well-known/openid-configuration

# Key fields:
# issuer, authorization_endpoint, token_endpoint,
# userinfo_endpoint, jwks_uri, scopes_supported,
# response_types_supported, grant_types_supported
```

### Standard OIDC Scopes

```
openid          # Required — returns sub claim in ID token
profile         # name, family_name, given_name, picture, etc.
email           # email, email_verified
address         # Postal address
phone           # phone_number, phone_number_verified
offline_access  # Request refresh token
```

## Token Storage Best Practices

```
Browser (SPA):
  - HttpOnly, Secure, SameSite=Strict cookie     # Preferred
  - NOT localStorage (XSS vulnerable)
  - NOT sessionStorage (XSS vulnerable)
  - BFF (Backend-for-Frontend) pattern            # Best

Mobile:
  - iOS Keychain / Android Keystore               # Encrypted storage
  - NOT SharedPreferences / UserDefaults

Server:
  - Environment variables or secret manager
  - Encrypted at rest
  - Short-lived access tokens (5-15 min)
  - Long-lived refresh tokens with rotation
```

## Tips

- Always use PKCE for public clients (SPAs, mobile apps, CLIs) even if the server does not require it
- Never send access tokens in URL query parameters; use the Authorization header with Bearer scheme
- Implement refresh token rotation to detect and prevent token theft
- The implicit grant (response_type=token) is deprecated; use authorization code + PKCE instead
- Validate the `state` parameter on every callback to prevent CSRF attacks
- Use short-lived access tokens (5-15 minutes) and long-lived refresh tokens for better security
- Token introspection is essential for opaque tokens; JWTs can be validated locally with the JWKS
- OIDC's `nonce` parameter in the authorization request prevents ID token replay attacks
- Always validate the `aud` (audience) claim to prevent token misuse across services
- Use the BFF (Backend-for-Frontend) pattern for SPAs to keep tokens out of the browser entirely
- Scopes should follow least-privilege: request only what the application needs
- The `offline_access` scope must be explicitly requested to receive refresh tokens with OIDC

## See Also

jwt, tls, cors, pki, vault

## References

- [RFC 6749 — OAuth 2.0 Authorization Framework](https://datatracker.ietf.org/doc/html/rfc6749)
- [RFC 7636 — PKCE](https://datatracker.ietf.org/doc/html/rfc7636)
- [RFC 7662 — Token Introspection](https://datatracker.ietf.org/doc/html/rfc7662)
- [RFC 8628 — Device Authorization Grant](https://datatracker.ietf.org/doc/html/rfc8628)
- [OpenID Connect Core 1.0](https://openid.net/specs/openid-connect-core-1_0.html)
- [OAuth 2.0 Security Best Practices (RFC 9700)](https://datatracker.ietf.org/doc/html/rfc9700)
