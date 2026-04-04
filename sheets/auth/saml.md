# SAML (Security Assertion Markup Language)

Implement browser-based single sign-on using SAML 2.0 with SP-initiated and IdP-initiated flows, assertion validation, metadata exchange, HTTP POST and Redirect bindings, RelayState management, and X.509 certificate lifecycle across Shibboleth, ADFS, Okta, and Keycloak.

## SSO Flow

### SP-Initiated Flow

```
User → SP: GET /protected
SP → User: 302 Redirect to IdP (AuthnRequest)
User → IdP: GET /sso?SAMLRequest=<base64>&RelayState=/protected
IdP → User: Login form (if no session)
User → IdP: POST credentials
IdP → User: HTML form with SAMLResponse (auto-submit POST)
User → SP: POST /acs (Assertion Consumer Service)
SP → User: 302 Redirect to /protected (session established)
```

### SP-Initiated AuthnRequest

```xml
<samlp:AuthnRequest
    xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol"
    xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion"
    ID="_a1b2c3d4e5f6"
    Version="2.0"
    IssueInstant="2026-04-03T10:00:00Z"
    Destination="https://idp.example.com/sso"
    AssertionConsumerServiceURL="https://sp.example.com/acs"
    ProtocolBinding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST">
  <saml:Issuer>https://sp.example.com/metadata</saml:Issuer>
  <samlp:NameIDPolicy
      Format="urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"
      AllowCreate="true"/>
</samlp:AuthnRequest>
```

### IdP-Initiated Flow

```
User → IdP: Navigate to IdP portal
IdP → User: Select application
IdP → User: HTML form with SAMLResponse (no prior AuthnRequest)
User → SP: POST /acs
SP → User: Redirect to default landing page
```

## SAML Assertions

### Full Response Structure

```xml
<samlp:Response
    xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol"
    ID="_resp_789"
    InResponseTo="_a1b2c3d4e5f6"
    Version="2.0"
    IssueInstant="2026-04-03T10:00:05Z"
    Destination="https://sp.example.com/acs">
  <saml:Issuer>https://idp.example.com</saml:Issuer>
  <ds:Signature><!-- XML Signature over Response --></ds:Signature>
  <samlp:Status>
    <samlp:StatusCode Value="urn:oasis:names:tc:SAML:2.0:status:Success"/>
  </samlp:Status>
  <saml:Assertion ID="_assert_456" Version="2.0"
      IssueInstant="2026-04-03T10:00:05Z">
    <saml:Issuer>https://idp.example.com</saml:Issuer>
    <saml:Subject>
      <saml:NameID Format="urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress">
        jdoe@example.com
      </saml:NameID>
      <saml:SubjectConfirmation Method="urn:oasis:names:tc:SAML:2.0:cm:bearer">
        <saml:SubjectConfirmationData
            InResponseTo="_a1b2c3d4e5f6"
            NotOnOrAfter="2026-04-03T10:05:05Z"
            Recipient="https://sp.example.com/acs"/>
      </saml:SubjectConfirmation>
    </saml:Subject>
    <saml:Conditions
        NotBefore="2026-04-03T09:59:55Z"
        NotOnOrAfter="2026-04-03T10:05:05Z">
      <saml:AudienceRestriction>
        <saml:Audience>https://sp.example.com/metadata</saml:Audience>
      </saml:AudienceRestriction>
    </saml:Conditions>
    <saml:AuthnStatement
        AuthnInstant="2026-04-03T10:00:03Z"
        SessionIndex="_sess_012">
      <saml:AuthnContext>
        <saml:AuthnContextClassRef>
          urn:oasis:names:tc:SAML:2.0:ac:classes:PasswordProtectedTransport
        </saml:AuthnContextClassRef>
      </saml:AuthnContext>
    </saml:AuthnStatement>
    <saml:AttributeStatement>
      <saml:Attribute Name="email">
        <saml:AttributeValue>jdoe@example.com</saml:AttributeValue>
      </saml:Attribute>
      <saml:Attribute Name="groups">
        <saml:AttributeValue>engineering</saml:AttributeValue>
        <saml:AttributeValue>devops</saml:AttributeValue>
      </saml:Attribute>
    </saml:AttributeStatement>
  </saml:Assertion>
</samlp:Response>
```

## Metadata XML

### SP Metadata

```xml
<md:EntityDescriptor
    xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata"
    entityID="https://sp.example.com/metadata">
  <md:SPSSODescriptor
      AuthnRequestsSigned="true"
      WantAssertionsSigned="true"
      protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <md:KeyDescriptor use="signing">
      <ds:KeyInfo>
        <ds:X509Data>
          <ds:X509Certificate>MIIDpDCCA...base64cert...</ds:X509Certificate>
        </ds:X509Data>
      </ds:KeyInfo>
    </md:KeyDescriptor>
    <md:KeyDescriptor use="encryption">
      <ds:KeyInfo>
        <ds:X509Data>
          <ds:X509Certificate>MIIDqDCCA...base64cert...</ds:X509Certificate>
        </ds:X509Data>
      </ds:KeyInfo>
    </md:KeyDescriptor>
    <md:SingleLogoutService
        Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect"
        Location="https://sp.example.com/slo"/>
    <md:AssertionConsumerService
        Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
        Location="https://sp.example.com/acs"
        index="0" isDefault="true"/>
  </md:SPSSODescriptor>
</md:EntityDescriptor>
```

### IdP Metadata

```xml
<md:EntityDescriptor
    xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata"
    entityID="https://idp.example.com">
  <md:IDPSSODescriptor
      WantAuthnRequestsSigned="true"
      protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <md:KeyDescriptor use="signing">
      <ds:KeyInfo>
        <ds:X509Data>
          <ds:X509Certificate>MIIDrDCCA...base64cert...</ds:X509Certificate>
        </ds:X509Data>
      </ds:KeyInfo>
    </md:KeyDescriptor>
    <md:SingleSignOnService
        Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect"
        Location="https://idp.example.com/sso"/>
    <md:SingleSignOnService
        Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
        Location="https://idp.example.com/sso"/>
  </md:IDPSSODescriptor>
</md:EntityDescriptor>
```

## Bindings

### HTTP-Redirect Binding

```bash
# AuthnRequest sent via query parameter (deflated + base64 + URL-encoded)
# Used for small messages (AuthnRequest, LogoutRequest)
GET https://idp.example.com/sso?\
  SAMLRequest=fZJNb4JAEIZ...&\
  RelayState=%2Fprotected&\
  SigAlg=http%3A%2F%2Fwww.w3.org%2F...rsa-sha256&\
  Signature=base64sig...

# Decode a SAMLRequest for debugging
echo "fZJNb4JAEIZ..." | base64 -d | python3 -c "import zlib,sys; print(zlib.decompress(sys.stdin.buffer.read(),-15).decode())"
```

### HTTP-POST Binding

```html
<!-- IdP returns auto-submitting form (used for SAMLResponse) -->
<html>
<body onload="document.forms[0].submit()">
  <form method="POST" action="https://sp.example.com/acs">
    <input type="hidden" name="SAMLResponse" value="PHNhbWxw...base64..."/>
    <input type="hidden" name="RelayState" value="/protected"/>
    <noscript><input type="submit" value="Continue"/></noscript>
  </form>
</body>
</html>
```

### Artifact Binding

```bash
# Step 1: IdP sends artifact reference (not full assertion)
# 302 Location: https://sp.example.com/acs?SAMLart=AAQAAMh48...

# Step 2: SP resolves artifact via back-channel SOAP call
# SP → IdP: ArtifactResolve (SOAP over HTTPS)
# IdP → SP: ArtifactResponse containing full assertion
```

## RelayState

```bash
# RelayState preserves the original URL across the SSO redirect
# Max length: 80 bytes (per SAML spec, though many IdPs allow more)

# SP sets RelayState when redirecting to IdP
RelayState=/dashboard?tab=overview

# IdP echoes RelayState back in the POST to ACS
# SP redirects user to RelayState value after assertion validation
```

## Certificate Management

### Generate SP Certificates

```bash
# Generate signing key and certificate (self-signed, 3 years)
openssl req -x509 -newkey rsa:2048 -nodes \
  -keyout sp-signing.key -out sp-signing.crt \
  -days 1095 -subj "/CN=sp.example.com"

# Generate encryption key and certificate
openssl req -x509 -newkey rsa:2048 -nodes \
  -keyout sp-encrypt.key -out sp-encrypt.crt \
  -days 1095 -subj "/CN=sp.example.com"

# View certificate details
openssl x509 -in sp-signing.crt -text -noout

# Check certificate expiry
openssl x509 -in sp-signing.crt -enddate -noout

# Extract public cert for metadata (base64, no headers)
openssl x509 -in sp-signing.crt -outform DER | base64
```

### Certificate Rotation

```bash
# 1. Generate new certificate
openssl req -x509 -newkey rsa:2048 -nodes \
  -keyout sp-signing-new.key -out sp-signing-new.crt \
  -days 1095 -subj "/CN=sp.example.com"

# 2. Add new cert to SP metadata (keep old cert too)
# 3. Upload updated metadata to IdP
# 4. Wait for IdP metadata cache refresh (24-48h)
# 5. Switch SP to sign with new key
# 6. Remove old cert from metadata after grace period
```

## SAML vs OIDC Comparison

```
Feature           SAML 2.0                  OIDC (OAuth 2.0)
─────────────────────────────────────────────────────────────
Token format      XML assertions            JWT (JSON)
Transport         Browser redirect/POST     Browser redirect + back-channel
Signature         XML DSIG (enveloped)      JWS (JOSE)
Encryption        XML Encryption            JWE (optional)
Discovery         Metadata XML              .well-known/openid-configuration
Mobile support    Poor (XML heavy)          Native (JSON/JWT lightweight)
Complexity        High                      Moderate
Maturity          2005 (enterprise)         2014 (modern apps)
Logout            SLO (unreliable)          RP-initiated (session mgmt)
```

## Tips

- Always validate `NotBefore`, `NotOnOrAfter`, `Audience`, `Recipient`, `InResponseTo`, and `Destination` on every assertion -- skipping any one of these checks opens signature wrapping or replay attacks
- Require signed assertions (`WantAssertionsSigned="true"`) even when the response itself is signed -- response-level signatures alone are vulnerable to XML wrapping attacks
- Use HTTP-POST binding for responses (not Redirect) because assertions with signatures and attributes exceed URL length limits in browsers
- Set clock skew tolerance to 2-3 minutes maximum on the SP -- wider windows increase replay attack risk
- Implement certificate rotation with overlap periods (dual certificates in metadata) to avoid hard cutover downtime
- Store RelayState server-side and pass only a reference token in the SAML flow to prevent open redirect vulnerabilities
- Validate XML schema and canonicalization before signature verification -- XML parser differences (e.g., comment handling) are a common attack surface
- Use Artifact binding for high-security environments where assertions should never pass through the browser
- Enable encrypted assertions (`<EncryptedAssertion>`) when attribute values contain sensitive data like group memberships or PII
- Log assertion IDs and enforce one-time use to prevent replay attacks within the NotOnOrAfter window
- Test IdP-initiated flow separately -- it lacks `InResponseTo` validation, making it inherently less secure than SP-initiated

## See Also

- kerberos, ldap, oauth, oidc, x509, openssl, jwt

## References

- [OASIS SAML 2.0 Specification](http://docs.oasis-open.org/security/saml/v2.0/)
- [SAML 2.0 Technical Overview](http://docs.oasis-open.org/security/saml/Post2.0/sstc-saml-tech-overview-2.0.html)
- [RFC 7522 — SAML 2.0 Profile for OAuth 2.0](https://datatracker.ietf.org/doc/html/rfc7522)
- [Shibboleth Documentation](https://shibboleth.atlassian.net/wiki/spaces/IDP5/overview)
- [Keycloak SAML Guide](https://www.keycloak.org/docs/latest/server_admin/#saml)
- [SAML Security Cheat Sheet (OWASP)](https://cheatsheetseries.owasp.org/cheatsheets/SAML_Security_Cheat_Sheet.html)
