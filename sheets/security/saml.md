# SAML 2.0

XML-based federated SSO protocol — OASIS standard since 2005, the workhorse of enterprise identity for two decades.

## Setup

SAML (Security Assertion Markup Language) 2.0 is an OASIS standard (March 2005) for exchanging authentication and authorization data between parties — typically an Identity Provider (IdP) and a Service Provider (SP). It uses XML for message format, XML Signature (XMLDSig) for integrity, and XML Encryption (XMLEnc) for confidentiality. SAML predates OIDC by roughly a decade and remains dominant in enterprise SSO, B2B federation, higher-education federations, and government identity.

| Aspect           | SAML 2.0                                             | OIDC                                  |
|------------------|------------------------------------------------------|---------------------------------------|
| Year             | 2005 (OASIS)                                         | 2014 (OpenID Foundation)              |
| Format           | XML                                                  | JSON / JWT                            |
| Transport        | HTTP-Redirect, HTTP-POST, SOAP, Artifact             | HTTPS only (RFC 6749 OAuth 2 base)    |
| Signing          | XMLDSig (C14N-Exclusive + RSA-SHA256)                | JWS (compact serialization)           |
| Encryption       | XMLEnc                                               | JWE                                   |
| Native client    | Awkward (ECP profile)                                | First-class (PKCE, device code)       |
| Mobile / SPA     | Awkward                                              | Native                                |
| Discovery        | Common-Domain-Cookie / metadata                      | `/.well-known/openid-configuration`   |
| Logout           | SLO (brittle)                                        | RP-Initiated / Front-Channel / Back-Channel logout |
| Token size       | 5–30 KB                                              | < 4 KB typical                        |
| Library quality  | Mature but XSW-prone                                 | Mature, simpler                       |

**When SAML wins:** existing enterprise IdP (ADFS, Shibboleth, Okta SAML), B2B SaaS where the customer's IdP is SAML-only, eduGAIN/InCommon higher-education federations, government (PIV/CAC + FICAM).

**When OIDC wins:** mobile, SPA, IoT, any greenfield deployment, anywhere you'd otherwise need ECP.

## Roles

| Role            | Acronym | Description                                                              |
|-----------------|---------|--------------------------------------------------------------------------|
| Identity Provider | IdP   | Authenticates the user, issues signed assertions                         |
| Service Provider  | SP    | Relies on IdP-issued assertion to grant access; the application          |
| User Agent        | UA    | Browser (web SSO) or ECP client                                          |
| Asserting Party   | —     | Synonym for IdP                                                          |
| Relying Party     | —     | Synonym for SP                                                           |
| Attribute Authority | AA  | (Less common) issues attribute statements, may be separate from IdP      |

Trust between IdP and SP is established **out-of-band** by exchanging metadata XML containing the partner's entityID, endpoints, signing certificate, and supported NameID formats. There is no central authority — every SAML deployment is a pairwise (or federation-mediated) trust establishment. The user-agent (browser) is **untrusted** — it carries assertions but cannot forge them because of the IdP signature.

```text
                    +-----+
                    | IdP |
                    +-----+
                       ^
                       | (3) AuthnRequest, (4) AuthnResponse
                       v
                    +-----+
                    |  UA |  (Browser)
                    +-----+
                       ^
                       | (1) GET /protected, (5) POST assertion
                       v
                    +-----+
                    |  SP |
                    +-----+
```

## Specification Stack

SAML 2.0 is not a single document — it is a stack of OASIS specifications that combine Core constructs with on-the-wire bindings, end-to-end profiles, metadata schemas, and authentication-context classes.

| Layer            | Document                                                   | Defines                                                  |
|------------------|------------------------------------------------------------|----------------------------------------------------------|
| Core             | `saml-core-2.0-os.pdf`                                     | Assertion + protocol XML schemas                         |
| Bindings         | `saml-bindings-2.0-os.pdf`                                 | HTTP-Redirect, HTTP-POST, HTTP-Artifact, SOAP, PAOS, URI |
| Profiles         | `saml-profiles-2.0-os.pdf`                                 | Web SSO, SLO, ECP, NameID Mgmt, Artifact Resolution      |
| Metadata         | `saml-metadata-2.0-os.pdf`                                 | EntityDescriptor + role descriptors                      |
| AuthnContext     | `saml-authn-context-2.0-os.pdf`                            | URI classes (Password, MFA, X509, Kerberos)              |
| Conformance      | `saml-conformance-2.0-os.pdf`                              | Operational mode targets                                 |
| Errata           | `sstc-saml-approved-errata-2.0.pdf`                        | Post-publication corrections                             |

Within Core: `<Assertion>` (the signed claim) and `<Protocol>` (request/response messages). Within Profiles: the **Web Browser SSO Profile** is what most people mean by "SAML."

## Web Browser SSO Profile

The canonical SP-initiated front-channel flow. The user starts at the SP; the SP redirects to the IdP; the IdP authenticates and POSTs a signed assertion back to the SP's Assertion Consumer Service (ACS) URL.

```text
 User      Browser              SP (sp.example.com)        IdP (idp.example.com)
  |           |                       |                            |
  |--Visit--->|--GET /protected------>|                            |
  |           |<--302 redirect-------|                             |
  |           |   Location: /sso?SAMLRequest=...&RelayState=...    |
  |           |--GET /sso?SAMLRequest=...---------------------------->|
  |           |                                              [authenticate user]
  |           |<--200 OK auto-submit form  HTTP-POST----------------|
  |           |    <input SAMLResponse=... RelayState=...>          |
  |           |--POST /acs SAMLResponse=...--->|                    |
  |           |                       [validate signature,          |
  |           |                        Audience, NotBefore/After,   |
  |           |                        Recipient, InResponseTo,     |
  |           |                        replay-check assertion ID]   |
  |           |<--302 to original URL---|                           |
```

Steps in detail:

1. User visits `https://sp.example.com/protected`.
2. SP sees no session, builds an `<AuthnRequest>`, deflates + base64-encodes it, redirects browser to IdP SSO endpoint with `SAMLRequest=` and optional `RelayState=` query parameters (HTTP-Redirect binding).
3. IdP authenticates the user (any factor).
4. IdP builds a `<Response>` containing one or more signed `<Assertion>` elements.
5. IdP returns an HTML page that auto-submits a form via `POST` to the SP's ACS URL with `SAMLResponse=` (base64) and `RelayState=`.
6. SP validates the response, creates a local session, redirects to the original URL (recovered from RelayState).

## SP-Initiated vs IdP-Initiated SSO

| Mode             | Flow                                                 | When to use                                  | Risk                                 |
|------------------|------------------------------------------------------|-----------------------------------------------|--------------------------------------|
| SP-initiated     | User starts at SP → SP issues `AuthnRequest` → IdP   | Default for web apps                         | Low — `InResponseTo` correlates request and response |
| IdP-initiated    | User starts at IdP portal → IdP POSTs unsolicited `<Response>` to SP | Portal/launcher use cases   | **Unsolicited-response** — no `InResponseTo`, attacker can replay or fixate |

**Unsolicited Response gotcha:** in IdP-initiated mode the SP cannot verify the response was a reply to a request it sent — it must trust the IdP signature alone and accept that there is no CSRF tie-in. Mitigations:

- Only enable IdP-initiated for specific tenants who need it.
- Rotate signing keys regularly.
- Strictly validate `Destination`, `Recipient`, `Audience`, `NotBefore`, `NotOnOrAfter`.
- Cache used assertion IDs (replay defense).
- Treat `RelayState` as opaque, not user input.

## AuthnRequest XML

The SP's request to authenticate. Sent via HTTP-Redirect (deflated + base64) or HTTP-POST (signed XML).

```xml
<samlp:AuthnRequest
    xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol"
    xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion"
    ID="_a1b2c3d4e5f6"
    Version="2.0"
    IssueInstant="2026-04-25T10:00:00Z"
    Destination="https://idp.example.com/sso"
    AssertionConsumerServiceURL="https://sp.example.com/saml/acs"
    ProtocolBinding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
    ForceAuthn="false"
    IsPassive="false">
  <saml:Issuer>https://sp.example.com/metadata</saml:Issuer>
  <samlp:NameIDPolicy
      Format="urn:oasis:names:tc:SAML:2.0:nameid-format:persistent"
      AllowCreate="true"/>
  <samlp:RequestedAuthnContext Comparison="exact">
    <saml:AuthnContextClassRef>
      urn:oasis:names:tc:SAML:2.0:ac:classes:PasswordProtectedTransport
    </saml:AuthnContextClassRef>
  </samlp:RequestedAuthnContext>
</samlp:AuthnRequest>
```

| Attribute / Element                  | Required | Purpose                                                              |
|--------------------------------------|----------|----------------------------------------------------------------------|
| `ID`                                 | Yes      | XS:ID, used for `InResponseTo` correlation                           |
| `Version`                            | Yes      | Always `2.0`                                                         |
| `IssueInstant`                       | Yes      | UTC timestamp                                                        |
| `Destination`                        | Yes      | Must equal the IdP SSO endpoint URL                                  |
| `AssertionConsumerServiceURL`        | No (use index) | Where the IdP will POST the response                            |
| `ProtocolBinding`                    | No       | Binding the SP wants the IdP to use for the response                 |
| `ForceAuthn`                         | No       | If `true`, IdP must reauthenticate even if a session exists          |
| `IsPassive`                          | No       | If `true`, IdP must not interact with user; only SSO if already in   |
| `<Issuer>`                           | Yes      | SP's entityID                                                        |
| `<NameIDPolicy>`                     | No       | What NameID format SP wants                                          |
| `<RequestedAuthnContext>`            | No       | What auth strength SP wants (exact / minimum / better / maximum)     |
| `<Subject>`                          | No       | If SP wants a specific user (rare)                                   |

## SAML Response XML

The IdP's reply. Always carries one or more `<Assertion>` elements; the response itself is usually signed, the assertion is always signed, or both.

```xml
<samlp:Response
    xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol"
    xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion"
    ID="_resp_001122"
    InResponseTo="_a1b2c3d4e5f6"
    Version="2.0"
    IssueInstant="2026-04-25T10:00:05Z"
    Destination="https://sp.example.com/saml/acs">
  <saml:Issuer>https://idp.example.com/metadata</saml:Issuer>
  <ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
    <!-- signature over Response -->
  </ds:Signature>
  <samlp:Status>
    <samlp:StatusCode Value="urn:oasis:names:tc:SAML:2.0:status:Success"/>
  </samlp:Status>
  <saml:Assertion
      ID="_assert_998877"
      Version="2.0"
      IssueInstant="2026-04-25T10:00:05Z">
    <saml:Issuer>https://idp.example.com/metadata</saml:Issuer>
    <ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
      <!-- signature over Assertion -->
    </ds:Signature>
    <saml:Subject>
      <saml:NameID Format="urn:oasis:names:tc:SAML:2.0:nameid-format:persistent"
                   NameQualifier="https://idp.example.com/metadata"
                   SPNameQualifier="https://sp.example.com/metadata">
        eb1d4f5a-9b2c-4a76-bf2c-1e0a8a3a8a8a
      </saml:NameID>
      <saml:SubjectConfirmation Method="urn:oasis:names:tc:SAML:2.0:cm:bearer">
        <saml:SubjectConfirmationData
            NotOnOrAfter="2026-04-25T10:05:05Z"
            Recipient="https://sp.example.com/saml/acs"
            InResponseTo="_a1b2c3d4e5f6"/>
      </saml:SubjectConfirmation>
    </saml:Subject>
    <saml:Conditions
        NotBefore="2026-04-25T09:59:35Z"
        NotOnOrAfter="2026-04-25T10:05:05Z">
      <saml:AudienceRestriction>
        <saml:Audience>https://sp.example.com/metadata</saml:Audience>
      </saml:AudienceRestriction>
      <saml:OneTimeUse/>
    </saml:Conditions>
    <saml:AuthnStatement
        AuthnInstant="2026-04-25T09:59:50Z"
        SessionIndex="_sess_0a1b2c3d"
        SessionNotOnOrAfter="2026-04-25T11:00:00Z">
      <saml:AuthnContext>
        <saml:AuthnContextClassRef>
          urn:oasis:names:tc:SAML:2.0:ac:classes:PasswordProtectedTransport
        </saml:AuthnContextClassRef>
      </saml:AuthnContext>
    </saml:AuthnStatement>
    <saml:AttributeStatement>
      <saml:Attribute Name="urn:oid:0.9.2342.19200300.100.1.3"
                      NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:uri"
                      FriendlyName="mail">
        <saml:AttributeValue>alice@example.com</saml:AttributeValue>
      </saml:Attribute>
      <saml:Attribute Name="urn:oid:1.3.6.1.4.1.5923.1.1.1.1"
                      NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:uri"
                      FriendlyName="eduPersonAffiliation">
        <saml:AttributeValue>staff</saml:AttributeValue>
        <saml:AttributeValue>employee</saml:AttributeValue>
      </saml:Attribute>
    </saml:AttributeStatement>
  </saml:Assertion>
</samlp:Response>
```

## Assertion Anatomy

An `<Assertion>` carries up to four kinds of statements. The **AuthnStatement** is mandatory for SSO; the others are optional.

| Element                | Purpose                                                                 |
|------------------------|-------------------------------------------------------------------------|
| `<Issuer>`             | EntityID of the IdP that minted the assertion                            |
| `<Signature>`          | XMLDSig over the assertion                                              |
| `<Subject>`            | Who the assertion is about; contains `<NameID>` + `<SubjectConfirmation>` |
| `<Conditions>`         | When and where the assertion is valid                                    |
| `<AuthnStatement>`     | When + how the user authenticated                                        |
| `<AttributeStatement>` | Claims about the user                                                    |
| `<AuthzDecisionStatement>` | Authorization decisions (rare; deprecated by XACML)                  |

| Subject sub-element       | Purpose                                                              |
|---------------------------|----------------------------------------------------------------------|
| `<NameID>`                | Identifier (with `Format`)                                           |
| `<SubjectConfirmation>`   | How SP confirms subject is talking; usually `Method="...:cm:bearer"` |
| `SubjectConfirmationData` | `NotOnOrAfter`, `Recipient` (must equal ACS URL), `InResponseTo`     |

| Conditions element         | Purpose                                                              |
|----------------------------|----------------------------------------------------------------------|
| `NotBefore`                | Reject if `now < NotBefore - clock_skew`                             |
| `NotOnOrAfter`             | Reject if `now >= NotOnOrAfter + clock_skew`                         |
| `<AudienceRestriction>`    | List of `<Audience>` URIs; SP entityID must appear                   |
| `<OneTimeUse>`             | Hint that SP must not cache; reject replay                           |
| `<ProxyRestriction>`       | Limits assertion forwarding (rare)                                   |

The `AuthnStatement.SessionIndex` is the cookie used to correlate this assertion with subsequent SLO requests — record it.

## NameID Formats

| Format URI (suffix on `urn:oasis:names:tc:SAML:2.0:nameid-format:`) | Description |
|---------------------------------------------------------------------|-------------|
| `persistent`        | Opaque, stable, pairwise-IdP+SP identifier; preserves user-correlation across logins; preferred for privacy |
| `transient`         | Opaque, single-session identifier; new value per login; strongest privacy, can't link sessions |
| `emailAddress`      | RFC 822 address; usually mutable; do not use as a primary key |
| `unspecified`       | Convention-specific; do not assume anything |
| `kerberos`          | `user@REALM` (RFC 1510)                                                  |
| `entity`            | EntityID (used for IdP/SP-as-subject) |
| `X509SubjectName`   | DN of an X.509 certificate (`CN=...,O=...`)                              |
| `WindowsDomainQualifiedName` | `DOMAIN\\user` (legacy)                                          |

| If you want…                       | Use NameID Format    |
|------------------------------------|----------------------|
| Stable user key for DB join        | `persistent`         |
| Anonymous one-shot assertion       | `transient`          |
| Show user a friendly identifier    | (not NameID — use AttributeStatement) |
| Federate an entity (machine)       | `entity`             |
| Smart-card user                    | `X509SubjectName`    |

`NameQualifier` (IdP entityID) and `SPNameQualifier` (SP entityID) scope the NameID; `persistent` IDs are pairwise so the same user has different IDs at different SPs.

## AttributeStatement

Attributes carry user claims. The **Name** is a URI in the `uri` NameFormat (recommended), or a short string in `basic`. Microsoft ADFS often emits `http://schemas.xmlsoap.org/ws/2005/05/identity/claims/...` (the SOAP claim names).

### eduPerson schema (InCommon, eduGAIN)

| Attribute                       | OID                                            | Use                                  |
|---------------------------------|------------------------------------------------|--------------------------------------|
| `mail`                          | `urn:oid:0.9.2342.19200300.100.1.3`            | Email address (RFC 1274)             |
| `cn`                            | `urn:oid:2.5.4.3`                              | Common name                          |
| `sn`                            | `urn:oid:2.5.4.4`                              | Surname                              |
| `givenName`                     | `urn:oid:2.5.4.42`                             | First name                           |
| `displayName`                   | `urn:oid:2.16.840.1.113730.3.1.241`            | Display name                         |
| `eduPersonPrincipalName`        | `urn:oid:1.3.6.1.4.1.5923.1.1.1.6`             | `user@scope` (eppn)                  |
| `eduPersonAffiliation`          | `urn:oid:1.3.6.1.4.1.5923.1.1.1.1`             | faculty / student / staff / member …  |
| `eduPersonScopedAffiliation`    | `urn:oid:1.3.6.1.4.1.5923.1.1.1.9`             | `staff@example.edu`                  |
| `eduPersonEntitlement`          | `urn:oid:1.3.6.1.4.1.5923.1.1.1.7`             | URN entitlement claims               |
| `eduPersonTargetedID`           | `urn:oid:1.3.6.1.4.1.5923.1.1.1.10`            | persistent NameID-style attr (deprecated by SAML2 persistent) |
| `eduPersonOrcid`                | `urn:oid:1.3.6.1.4.1.5923.1.1.1.16`            | ORCID iD                             |
| `eduPersonAssurance`            | `urn:oid:1.3.6.1.4.1.5923.1.1.1.11`            | LoA / REFEDS Assurance Framework     |
| `schacHomeOrganization`         | `urn:oid:1.3.6.1.4.1.25178.1.2.9`              | DNS-style home org                   |

### Microsoft / ADFS-style claims (NameFormat=basic or uri)

| Claim Name                                                                           | Maps to        |
|--------------------------------------------------------------------------------------|----------------|
| `http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name`                         | username       |
| `http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress`                 | email          |
| `http://schemas.xmlsoap.org/ws/2005/05/identity/claims/givenname`                    | given name     |
| `http://schemas.xmlsoap.org/ws/2005/05/identity/claims/surname`                      | surname        |
| `http://schemas.xmlsoap.org/ws/2005/05/identity/claims/upn`                          | UPN (`user@domain`) |
| `http://schemas.xmlsoap.org/claims/Group`                                            | AD group       |
| `http://schemas.microsoft.com/ws/2008/06/identity/claims/role`                       | role           |
| `http://schemas.microsoft.com/identity/claims/objectidentifier`                      | Azure AD oid   |
| `http://schemas.microsoft.com/identity/claims/tenantid`                              | Azure tenant   |

### Custom claims

Use a URI you control, e.g. `https://sp.example.com/claims/department`. Avoid colliding short names like `role` between IdPs; pick the URI namespace your SP recognizes.

```xml
<saml:Attribute Name="https://sp.example.com/claims/department"
                NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:uri">
  <saml:AttributeValue xsi:type="xs:string">engineering</saml:AttributeValue>
</saml:Attribute>
```

Multi-valued attributes use multiple `<AttributeValue>` siblings.

## Group / Role Mapping — JIT Provisioning

Map IdP-issued attributes to SP roles on first login (Just-In-Time provisioning):

| Step  | Action                                                                                |
|-------|---------------------------------------------------------------------------------------|
| 1     | Read attribute(s): `eduPersonAffiliation`, `Group`, `role`, `memberOf`, `isMemberOf`  |
| 2     | Map (case-sensitive!) to local role/permission via config table                       |
| 3     | If user does not exist: insert; populate from attributes                              |
| 4     | If user exists: update mutable attributes (email, name); leave NameID + ID unchanged  |
| 5     | Store NameID (Format + Value + NameQualifier + SPNameQualifier) as join key           |

**JIT gotchas:**

- **Persistent NameID is the join key, not email.** Email may change, persistent NameID does not.
- **Attribute mapping is case-sensitive.** `Staff != staff` for some IdPs; normalize.
- **Group removal isn't automatic.** If the user loses a group, the next assertion just doesn't list it; your SP must replace, not append.
- **First login race.** Two parallel logins of the same new user can both INSERT — use upsert with NameID-uniqueness constraint.
- **Group explosion.** Some IdPs send hundreds of AD groups; cap or filter, otherwise the assertion can exceed the URL/POST limit.
- **Don't trust `email` for identity.** Email-based JIT lets a renamed user hijack a local account if the previous owner is gone.

## Bindings

A binding maps SAML messages onto a transport.

| Binding URI suffix                                        | Direction                          | Method | Notes                                              |
|-----------------------------------------------------------|------------------------------------|--------|----------------------------------------------------|
| `urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect`      | Request (browser → IdP)            | GET    | Deflate + base64 in URL; signature is `?Signature=`; URL-length limit |
| `urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST`          | Response (IdP → SP) usually        | POST   | Auto-submit form; XML signed inside body            |
| `urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Artifact`      | Response or Request                | GET/POST | Front-channel artifact + back-channel ArtifactResolve via SOAP |
| `urn:oasis:names:tc:SAML:2.0:bindings:SOAP`               | Back-channel (SP ↔ IdP server)     | POST   | Used by Artifact Resolution, Attribute Query, SLO back-channel |
| `urn:oasis:names:tc:SAML:2.0:bindings:PAOS`               | ECP                                | reverse SOAP | Non-browser clients                          |
| `urn:oasis:names:tc:SAML:2.0:bindings:URI`                | Attribute query                    | GET    | Rare                                               |

| Use case                          | Request binding | Response binding |
|-----------------------------------|-----------------|------------------|
| Standard web SSO                  | HTTP-Redirect   | HTTP-POST        |
| Large `<AuthnRequest>` (Cond.)    | HTTP-POST       | HTTP-POST        |
| Artifact (avoid passing assertion via browser) | HTTP-Redirect | HTTP-Artifact + SOAP |
| ECP (curl, native client)         | PAOS            | PAOS             |
| Back-channel SLO                  | SOAP            | SOAP             |

### HTTP-Redirect signing

The query string is the canonical form (alphabetic order: `SAMLRequest` or `SAMLResponse`, `RelayState`, `SigAlg`), signed with the IdP/SP's private key, and the result base64-encoded in the `Signature=` parameter. This is **not XMLDSig** — it's a query-string-level signature.

### HTTP-POST signing

The XML payload itself is signed with XMLDSig before being base64-encoded into the form input.

## RelayState

`RelayState` is an opaque, application-controlled token that round-trips with the SAML message and is returned unmodified to the SP. The SP uses it to remember where the user was going.

| Rule                                          | Why                                                                  |
|-----------------------------------------------|----------------------------------------------------------------------|
| Maximum length: 80 bytes (recommended)        | HTTP-Redirect URL-length limits across user-agents                   |
| Treat as opaque — do not put state in it      | Browser tampering, CSRF, attacker-fixated values                     |
| Use it as a key into server-side state        | E.g. `RelayState=<uuid>` → DB row with original URL                  |
| URL-encode in HTTP-Redirect, raw in HTTP-POST | Both bindings have different escaping rules                          |
| Sign it (Redirect) so the browser can't swap  | Already covered by the query-string signature                        |

**Don't** put a raw return-URL in RelayState — open-redirect attack. Whitelist hosts.

## XML Signature (XMLDSig)

XMLDSig (`http://www.w3.org/2000/09/xmldsig#`) signs an XML element and embeds the signature inside it (enveloped) or outside (detached). SAML uses **enveloped signatures with Exclusive Canonicalization (C14N-Exclusive)** and RSA-SHA256.

```xml
<ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
  <ds:SignedInfo>
    <ds:CanonicalizationMethod
        Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#"/>
    <ds:SignatureMethod
        Algorithm="http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"/>
    <ds:Reference URI="#_assert_998877">
      <ds:Transforms>
        <ds:Transform Algorithm="http://www.w3.org/2000/09/xmldsig#enveloped-signature"/>
        <ds:Transform Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#"/>
      </ds:Transforms>
      <ds:DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha256"/>
      <ds:DigestValue>m1q5Q...=</ds:DigestValue>
    </ds:Reference>
  </ds:SignedInfo>
  <ds:SignatureValue>RxQp...</ds:SignatureValue>
  <ds:KeyInfo>
    <ds:X509Data>
      <ds:X509Certificate>MIIDXTCCAk...</ds:X509Certificate>
    </ds:X509Data>
  </ds:KeyInfo>
</ds:Signature>
```

| Algorithm category | Recommended URI                                                                  |
|--------------------|----------------------------------------------------------------------------------|
| Canonicalization   | `http://www.w3.org/2001/10/xml-exc-c14n#` (Exclusive C14N 1.0)                   |
| Digest             | `http://www.w3.org/2001/04/xmlenc#sha256`                                        |
| Signature          | `http://www.w3.org/2001/04/xmldsig-more#rsa-sha256`                              |
| Transform          | `http://www.w3.org/2000/09/xmldsig#enveloped-signature` then C14N-Exclusive      |
| Deprecated         | `http://www.w3.org/2000/09/xmldsig#sha1`, `...#rsa-sha1` — **do not accept**     |

C14N-Exclusive is mandatory because the assertion's namespaces may differ between issuance and verification (e.g. when wrapped inside a `<Response>`). C14N-Inclusive copies ancestor namespaces and breaks signatures across re-parenting.

## Signature Verification Algorithm

To verify an `<Assertion>` signature:

1. Locate the `<Signature>` child (must be direct child of element it signs).
2. Verify `SignedInfo/Reference/@URI` matches the enclosing element's `ID` attribute (use the schema-declared `ID` type, **not** XPath name lookup).
3. Apply the `<Transforms>` in order: enveloped-signature (remove `<Signature>`), then C14N-Exclusive.
4. Compute SHA-256 digest of canonicalized bytes; compare to `<DigestValue>`.
5. Canonicalize `<SignedInfo>` itself; verify RSA signature with the public key in `<KeyInfo>` (or out-of-band metadata).
6. Confirm the cert in `<KeyInfo>` matches the metadata-pinned cert (don't trust embedded chain alone).
7. Confirm cert is not expired and not revoked (CRL/OCSP if you require it).
8. **Use the verified element by ID**, not by document position — a signed assertion may be anywhere in the doc; ensure your code dereferences the same element that was verified (XSW defense).

```text
┌──────────────────────────────────────────────────────────┐
│  XMLDSig Verification                                     │
│  1. parse XML (no DTD; disable XXE)                      │
│  2. find <Signature> in the element to be verified       │
│  3. fetch <Reference URI="#X"> → resolve element by ID   │
│  4. apply <Transforms> (enveloped → c14n-exclusive)      │
│  5. compute SHA-256 → compare DigestValue                │
│  6. canonicalize <SignedInfo> → RSA-verify SignatureValue│
│  7. compare cert fingerprint to metadata-pinned cert     │
│  8. **process only the verified element** (by ID)        │
└──────────────────────────────────────────────────────────┘
```

## XML Signature Wrapping (XSW) Attacks

XSW: an attacker takes a legitimate signed assertion and re-arranges or duplicates XML so that the **signed** element is verified but the **processed** element is a forged copy. The seminal paper (Somorovsky et al, "On Breaking SAML", 2012) found 11/14 frameworks vulnerable, including OpenSAML, Salesforce, and IBM XS40.

| Variant                      | Attack                                                                  |
|------------------------------|-------------------------------------------------------------------------|
| XSW1 — Wrap in `<Response>`  | Original assertion stays; attacker prepends evil sibling that processor reads |
| XSW2 — Wrap in extension     | Hide signed assertion in `Object` / `Extensions`; attacker assertion in main path |
| XSW3 — Wrap in `<Object>`    | Signed copy inside `Signature/Object`; processor sees forged top-level |
| XSW4-7                       | Variants moving evil element relative to signed                         |
| XSW8                         | Cause processor to re-process descendant of wrapped node                |

### XSW1 — duplicate `<Assertion>` outside `<Response>`

Attacker copies the signed assertion, modifies the copy, and inserts both. The signature reference still points to the original via `URI="#_signed"`, but a naive processor reads the first/last `<Assertion>` it finds.

```xml
<samlp:Response>
  <saml:Assertion ID="_evil">                <!-- forged, processor reads -->
    <saml:Subject><saml:NameID>admin@victim</saml:NameID></saml:Subject>
    ...
  </saml:Assertion>
  <saml:Assertion ID="_signed">              <!-- original, signature refers to this -->
    <ds:Signature><ds:Reference URI="#_signed"/></ds:Signature>
    <saml:Subject><saml:NameID>alice@victim</saml:NameID></saml:Subject>
    ...
  </saml:Assertion>
</samlp:Response>
```

### XSW2 — wrap signed assertion inside the forged one

```xml
<samlp:Response>
  <saml:Assertion ID="_evil">                <!-- processor reads this -->
    <saml:Subject><saml:NameID>admin@victim</saml:NameID></saml:Subject>
    <saml:Assertion ID="_signed">            <!-- nested, signature still validates -->
      <ds:Signature><ds:Reference URI="#_signed"/></ds:Signature>
      ...
    </saml:Assertion>
  </saml:Assertion>
</samlp:Response>
```

### XSW3 — signed assertion inside `<ds:Object>`

```xml
<samlp:Response>
  <saml:Assertion ID="_evil">
    <saml:Subject><saml:NameID>admin@victim</saml:NameID></saml:Subject>
    <ds:Signature>
      <ds:Reference URI="#_signed"/>
      <ds:Object>
        <saml:Assertion ID="_signed">        <!-- hidden inside signature itself -->
          ...
        </saml:Assertion>
      </ds:Object>
    </ds:Signature>
  </saml:Assertion>
</samlp:Response>
```

### XSW4 — reorder so forged is first child

Signed assertion appears as later sibling; processor's `getFirstChild()` returns the evil one. Library bug class: tree-walk by index instead of ID.

### XSW5 — forged `<Response>` wrapping a signed `<Assertion>`

```xml
<samlp:Response ID="_evil_resp">
  <ds:Signature><ds:Reference URI="#_signed"/></ds:Signature>   <!-- signature copied -->
  <saml:Assertion ID="_evil">
    <saml:Subject><saml:NameID>admin@victim</saml:NameID></saml:Subject>
  </saml:Assertion>
  <saml:Assertion ID="_signed">                                 <!-- original -->
    ...
  </saml:Assertion>
</samlp:Response>
```

### XSW6 — signature wrapping with `<Extensions>`

```xml
<samlp:Response>
  <samlp:Extensions>
    <saml:Assertion ID="_signed">            <!-- hidden in Extensions -->
      <ds:Signature><ds:Reference URI="#_signed"/></ds:Signature>
    </saml:Assertion>
  </samlp:Extensions>
  <saml:Assertion ID="_evil">                <!-- processor reads -->
    <saml:Subject><saml:NameID>admin@victim</saml:NameID></saml:Subject>
  </saml:Assertion>
</samlp:Response>
```

### XSW7 — append signed `<Assertion>` after `</samlp:Response>`

Some parsers accept and process trailing siblings outside the root in lenient mode — verifier sees signature valid (against the trailing assertion), processor ingests forged top-level assertion.

### XSW8 — signature on `<Response>` but processor consumes nested `<Assertion>`

The processor verifies the outer `<Response>` signature once and trusts every inner `<Assertion>`. Attacker replaces inner `<Assertion>` with a forged one (no inner signature). Library bug: "Response signed → all assertions trusted."

### Library-specific XSW history

| Library                         | CVE                  | Vector                                                          |
|---------------------------------|----------------------|-----------------------------------------------------------------|
| OneLogin python-saml < 2.4.0    | CVE-2016-1000252     | XSW8 — outer `<Response>` signature trusted inner forged assertion |
| OneLogin ruby-saml < 0.8.18     | CVE-2017-11427       | Comment-stripping XSW (XSW comment-bypass)                      |
| node-saml passport-saml < 1.3.5 | CVE-2017-11429       | XSW comment in NameID                                           |
| Cisco AnyConnect SAML auth      | CVE-2018-0227        | XSW + signature stripping                                       |
| Citrix ADC SAML SP              | CVE-2019-19781 chain | Combined RCE + SAML XSW                                         |
| Sustainsys.Saml2 < 2.7.0        | CVE-2019-13483       | Reference URI confusion (XSW7-class)                            |
| Spring Security SAML < 1.0.10   | CVE-2018-1258        | Authentication bypass via XSW                                   |
| python3-saml < 1.10.0           | CVE-2021-30243       | XSW comment in NameID similar to ruby-saml CVE-2017-11427       |
| Shibboleth SP < 3.2.1           | CVE-2021-25392       | XSW — incorrect signature reference resolution                  |

### Defenses

- **Verify and process the same element** (by ID, not by name) — pass the verified element reference to your business logic, don't re-XPath.
- **Reject documents with multiple elements sharing an ID.**
- **Use schema validation** to forbid unexpected elements (e.g. `<Object>` containing `<Assertion>`).
- **Disable DTDs and external entities** (XXE).
- **Use C14N-Exclusive**, not Inclusive.
- **Reject sibling/trailing assertions** — exactly one `<Assertion>` per `<Response>` (or one per `<EncryptedAssertion>`).
- **Require signed assertion**, not just signed response — outer-only XSW8 then fails.
- **Reject if `<Signature>` not a direct child** of the element claimed signed.
- **Strip and reject comments** in `<NameID>` and other text nodes (xmlsec ≥ 2018 fixes).
- **Prefer libraries that have been audited post-2012**: python3-saml ≥ 1.10, OpenSAML 4, passport-saml ≥ 4, samlify ≥ 2, crewjam/saml ≥ 0.4, ruby-saml ≥ 1.12.
- **Reject if signature is missing entirely** — never accept on the basis of "issuer matches metadata."

## Encryption (XMLEnc)

XMLEnc (`http://www.w3.org/2001/04/xmlenc#`) encrypts XML elements. SAML uses it to encrypt `<Assertion>` (becoming `<EncryptedAssertion>`), `<NameID>` (becoming `<EncryptedID>`), or attributes (`<EncryptedAttribute>`).

```xml
<saml:EncryptedAssertion>
  <xenc:EncryptedData xmlns:xenc="http://www.w3.org/2001/04/xmlenc#"
                      Type="http://www.w3.org/2001/04/xmlenc#Element">
    <xenc:EncryptionMethod Algorithm="http://www.w3.org/2009/xmlenc11#aes256-gcm"/>
    <ds:KeyInfo xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
      <xenc:EncryptedKey>
        <xenc:EncryptionMethod Algorithm="http://www.w3.org/2009/xmlenc11#rsa-oaep">
          <ds:DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha256"/>
        </xenc:EncryptionMethod>
        <xenc:CipherData>
          <xenc:CipherValue>...wrapped CEK...</xenc:CipherValue>
        </xenc:CipherData>
      </xenc:EncryptedKey>
    </ds:KeyInfo>
    <xenc:CipherData>
      <xenc:CipherValue>...AES-GCM ciphertext of <Assertion>...</xenc:CipherValue>
    </xenc:CipherData>
  </xenc:EncryptedData>
</saml:EncryptedAssertion>
```

| Algorithm                      | URI                                                          | Notes                            |
|--------------------------------|--------------------------------------------------------------|----------------------------------|
| Block (recommended)            | `http://www.w3.org/2009/xmlenc11#aes256-gcm`                 | AEAD, modern                     |
| Block (legacy)                 | `http://www.w3.org/2001/04/xmlenc#aes256-cbc`                | Vulnerable to padding-oracle (Jager 2012) — avoid |
| Key wrap (recommended)         | `http://www.w3.org/2009/xmlenc11#rsa-oaep` + SHA-256         | RSA-OAEP MGF1                    |
| Key wrap (deprecated)          | `http://www.w3.org/2001/04/xmlenc#rsa-1_5`                   | Bleichenbacher-vulnerable        |

## When to Encrypt

| Scenario                                                | Encrypt? |
|---------------------------------------------------------|----------|
| Assertion travels via TLS-protected POST to SP, SP private network | No — TLS is enough     |
| Assertion contains regulated data (PHI, PII) and back-channel via Artifact + SOAP | Yes        |
| You don't trust intermediate proxies or load-balancers terminating TLS | Yes |
| Compliance requires "encrypted at rest including transit" | Yes |
| You want to hide NameID from logs                       | Encrypt only `<NameID>` (`<EncryptedID>`) |

Default: **TLS protects transit; encrypt only when policy or threat model requires.** Encrypted SAML adds key-management overhead and rules out HTTP-POST debugging.

## SAML Metadata

Metadata XML is the trust contract — both parties exchange these documents (or fetch from a federation aggregator).

```xml
<md:EntityDescriptor xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata"
                     entityID="https://idp.example.com/metadata"
                     validUntil="2026-12-31T00:00:00Z"
                     cacheDuration="PT24H">
  <md:IDPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol"
                       WantAuthnRequestsSigned="true">
    <md:KeyDescriptor use="signing">
      <ds:KeyInfo xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
        <ds:X509Data>
          <ds:X509Certificate>MIIDXTCCAk...</ds:X509Certificate>
        </ds:X509Data>
      </ds:KeyInfo>
    </md:KeyDescriptor>
    <md:KeyDescriptor use="encryption">
      <ds:KeyInfo xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
        <ds:X509Data><ds:X509Certificate>MIIDXTCCAk...</ds:X509Certificate></ds:X509Data>
      </ds:KeyInfo>
    </md:KeyDescriptor>
    <md:NameIDFormat>urn:oasis:names:tc:SAML:2.0:nameid-format:persistent</md:NameIDFormat>
    <md:NameIDFormat>urn:oasis:names:tc:SAML:2.0:nameid-format:transient</md:NameIDFormat>
    <md:SingleSignOnService
        Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect"
        Location="https://idp.example.com/sso"/>
    <md:SingleSignOnService
        Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
        Location="https://idp.example.com/sso"/>
    <md:SingleLogoutService
        Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect"
        Location="https://idp.example.com/slo"/>
  </md:IDPSSODescriptor>
</md:EntityDescriptor>
```

| Element                       | Purpose                                                              |
|-------------------------------|----------------------------------------------------------------------|
| `<EntityDescriptor>`          | Top-level; carries `entityID`, `validUntil`, `cacheDuration`         |
| `<IDPSSODescriptor>`          | IdP role: signing keys, SSO endpoints                                |
| `<SPSSODescriptor>`           | SP role: ACS, SLO, signing keys                                      |
| `<KeyDescriptor use="signing"\|"encryption">` | X.509 cert (multiple allowed for rollover)            |
| `<NameIDFormat>`              | Supported NameID format URIs                                         |
| `<SingleSignOnService>`       | (IDP) endpoint binding + URL                                         |
| `<AssertionConsumerService>`  | (SP) endpoint, can have `index`                                      |
| `<SingleLogoutService>`       | SLO endpoint                                                         |
| `<RoleDescriptor>`            | Generic, used for AttributeAuthority etc.                            |
| `<EntitiesDescriptor>`        | Wrapper containing many EntityDescriptors (federations)              |
| `<Extensions>`                | `<mdui:UIInfo>`, `<mdui:Logo>`, `<shibmd:Scope>`                     |
| `<Organization>`              | Display name and URL                                                 |
| `<ContactPerson>`             | Contact details for incident response                                |
| `<RequestedAttribute>`        | (SP) declares attributes it needs                                    |

`WantAuthnRequestsSigned="true"` (IdP) requires SP to sign requests. `AuthnRequestsSigned="true"` and `WantAssertionsSigned="true"` are SP-side. `validUntil` is the cliff after which the metadata expires; `cacheDuration` is the polling hint.

## Metadata Trust Models

| Model                       | How                                                              | Examples                            |
|-----------------------------|------------------------------------------------------------------|-------------------------------------|
| Direct exchange             | Email a metadata XML or upload via admin UI                       | Most enterprise SaaS                |
| Bilateral URL fetch         | SP polls IdP metadata URL daily; pin signing cert                 | Custom integrations                 |
| Federation                  | Trusted aggregator publishes signed master metadata               | InCommon, eduGAIN, UKAMF, ACAMP, GakuNin |
| Dynamic registration        | Rare in SAML; common in OIDC                                      | —                                   |

For federations, validate the **federation operator's signature** on the aggregated metadata, then trust the entries inside. eduGAIN aggregates ~70 national federations; InCommon aggregates US higher-ed.

## Single Logout (SLO)

SLO terminates the session at the IdP and at every SP that the IdP has issued an assertion to during the user's IdP session. **It is universally regarded as brittle.**

```text
 User clicks logout at SP1
   |
   |---LogoutRequest---->IdP
   |                      |
   |                      |--LogoutRequest-->SP2  (front- or back-channel)
   |                      |<--LogoutResponse--|
   |                      |--LogoutRequest-->SP3
   |                      |<--LogoutResponse--|
   |<--LogoutResponse-----|
```

Why brittle:

- **Front-channel** SLO requires user-agent to traverse every SP — popup-blocker, third-party-cookie, browser-back, closed-tab all break it.
- **Back-channel** SLO requires the IdP to know SP back-channel endpoints and have network reachability — usually doesn't work across NAT.
- **State synchronization** — if SP3 is offline, what does the IdP do? OASIS says: best-effort.
- **Mobile / app SP** sessions don't always expose an SLO endpoint.
- **No equivalent to OIDC's back-channel logout token verification spec until much later** (SAML SLO is from 2005).

## SLO Bindings

| Binding type    | Direction                       | Pros                            | Cons                                            |
|-----------------|---------------------------------|---------------------------------|-------------------------------------------------|
| Front-channel HTTP-Redirect | Browser-mediated      | Simple                          | Requires browser to bounce through every SP     |
| Front-channel HTTP-POST     | Browser-mediated      | Larger payload, signed body     | Same browser-bouncing brittleness               |
| Back-channel SOAP           | IdP server → SP server | No browser dependency           | NAT, firewall, TLS-trust, offline SP all break it |

Real-world advice: **build SLO if compliance demands it, but do not promise users a clean global logout.** Set short SP session timeouts as the real defense.

## SessionIndex

`<AuthnStatement SessionIndex="...">` is the IdP-issued session correlation token. The SP records `(NameID, SessionIndex)` at login. On `<LogoutRequest>` the IdP sends the same `<NameID>` + `<SessionIndex>` so the SP can identify which session(s) to terminate.

```xml
<samlp:LogoutRequest ID="_lr_001" Version="2.0" IssueInstant="..."
                     Destination="https://sp.example.com/saml/slo">
  <saml:Issuer>https://idp.example.com/metadata</saml:Issuer>
  <saml:NameID Format="urn:oasis:names:tc:SAML:2.0:nameid-format:persistent">
    eb1d4f5a-9b2c-4a76-bf2c-1e0a8a3a8a8a
  </saml:NameID>
  <samlp:SessionIndex>_sess_0a1b2c3d</samlp:SessionIndex>
</samlp:LogoutRequest>
```

If a user has two SP sessions (two logins, two SessionIndices), the IdP sends one `<LogoutRequest>` per index — the SP must terminate exactly the matching session.

## AuthnContext

`<AuthnContextClassRef>` carries a URI describing **how** the user authenticated.

| URI suffix on `urn:oasis:names:tc:SAML:2.0:ac:classes:`     | Meaning                                  |
|-------------------------------------------------------------|------------------------------------------|
| `unspecified`                                               | Not specified                            |
| `Password`                                                  | Username/password, **plaintext** (e.g. HTTP) |
| `PasswordProtectedTransport`                                | Username/password over TLS               |
| `TLSClient`                                                 | Mutual TLS                               |
| `X509`                                                      | X.509 cert (any factor)                  |
| `Smartcard`                                                 | Hardware-token cert                      |
| `SmartcardPKI`                                              | Hardware token + PKI                     |
| `Kerberos`                                                  | Windows / GSS-API Kerberos               |
| `MobileTwoFactorContract`                                   | Mobile + second factor                   |
| `PreviousSession`                                           | IdP session reused, no fresh auth        |
| `MultiFactorAuthn` (often `nist:800-63:LoA:3` or vendor)    | Multi-factor (custom URI)                |

In `<RequestedAuthnContext>` the SP says what it wants:

```xml
<samlp:RequestedAuthnContext Comparison="exact">
  <saml:AuthnContextClassRef>
    urn:oasis:names:tc:SAML:2.0:ac:classes:PasswordProtectedTransport
  </saml:AuthnContextClassRef>
</samlp:RequestedAuthnContext>
```

| `Comparison` value | Meaning                                                                    |
|--------------------|----------------------------------------------------------------------------|
| `exact`            | Authenticated context must equal one of the requested URIs                 |
| `minimum`          | Must be at least as strong (strength is IdP-defined ordering)              |
| `better`           | Must be strictly stronger                                                  |
| `maximum`          | Must be no stronger than                                                   |

The SP must **validate** the returned `<AuthnContextClassRef>` matches policy — IdPs sometimes downgrade. Don't assume that requesting MFA gets you MFA.

## Identity Provider Discovery

When a user lands at an SP that supports multiple IdPs, the SP must figure out which IdP to redirect to.

| Mechanism                          | How                                                  | Note                                |
|------------------------------------|------------------------------------------------------|-------------------------------------|
| Common Domain Cookie (CDC)         | Browser cookie on a shared domain, set by IdPs       | Profile in OASIS, mostly legacy     |
| Centralized Discovery Service      | SP redirects to a discovery URL; user picks IdP; redirected back | Shibboleth's WAYF / Discovery Service |
| `IdPList` parameter                | SP passes preferred IdPs                             | Rare                                |
| Email-domain mapping               | User types email → SP looks up domain → IdP entityID | Common in B2B SaaS                  |
| URL parameter                      | `?idp=https://idp.example.com/metadata`              | For deep links                      |

In federations like InCommon, the discovery service is centralized at `https://wayf.incommonfederation.org/`.

## Implementation Libraries

| Language | Library                          | Repo / package                              | Notes                                |
|----------|----------------------------------|---------------------------------------------|--------------------------------------|
| Python   | `python3-saml`                   | `github.com/SAML-Toolkits/python3-saml`     | OneLogin lineage, well-audited       |
| Python   | `pysaml2`                        | `github.com/IdentityPython/pysaml2`         | Used by SATOSA / Shibboleth tooling  |
| Java     | OpenSAML 4                       | `shibboleth.net/projects/opensaml`          | Reference implementation             |
| Java     | Spring Security SAML2 Login      | `spring-security-saml2-service-provider`    | SP-only, Spring-Boot-friendly        |
| Java     | Pac4j                            | `github.com/pac4j/pac4j`                    | Multi-protocol (SAML, OIDC, CAS)     |
| Node     | `passport-saml`                  | `github.com/node-saml/passport-saml`        | Passport.js strategy; SP-only        |
| Node     | `samlify`                        | `github.com/tngan/samlify`                  | SP + IdP roles                       |
| Node     | `saml2-js`                       | `github.com/Clever/saml2`                   | SP-only                              |
| Go       | `crewjam/saml`                   | `github.com/crewjam/saml`                   | SP + IdP, mature                     |
| Go       | `russellhaering/gosaml2`         | `github.com/russellhaering/gosaml2`         | SP only                              |
| .NET     | `ITfoxtec.Identity.Saml2`        | NuGet `ITfoxtec.Identity.Saml2`             | SP + IdP                             |
| .NET     | `Sustainsys.Saml2`               | `github.com/Sustainsys/Saml2`               | OWIN/Katana, ASP.NET Core            |
| Ruby     | `ruby-saml`                      | `github.com/SAML-Toolkits/ruby-saml`        | OneLogin lineage                     |
| PHP      | `SimpleSAMLphp`                  | `simplesamlphp.org`                         | Full IdP + SP, hosted apps           |
| PHP      | `php-saml`                       | `github.com/SAML-Toolkits/php-saml`         | SP-focused                           |
| Apache   | `mod_auth_mellon`                | `github.com/latchset/mod_auth_mellon`       | Apache module, no app code           |
| Apache   | `mod_shib`                       | `shibboleth.net`                            | Apache module, federation-grade      |
| nginx    | (use `mod_shib` via FastCGI)     | —                                           |                                      |

**Build vs use:** **always use a library.** Hand-rolled SAML is the single most common cause of XSW vulnerabilities. The libraries above have been audited; your code has not.

## Library Validation Checklist

When integrating any SAML library, verify it does **all** of these. If the library doesn't, replace it.

| Check                                    | Why                                                                 |
|------------------------------------------|---------------------------------------------------------------------|
| Verify XML signature on Response and/or Assertion | Required for integrity                                       |
| Pin signing cert to metadata             | KeyInfo embedded cert is attacker-controlled                        |
| Reject unsigned assertions               | "Issuer matches" is not enough                                      |
| Validate `<Audience>` includes SP entityID | Otherwise reuse-against-other-SP                                  |
| Validate `NotBefore` (with skew ≤ 5 min) | Pre-validity replay                                                 |
| Validate `NotOnOrAfter` (with skew)      | Post-validity replay                                                |
| Validate `Recipient` equals ACS URL      | Wrong-SP redirect                                                   |
| Validate `Destination` equals ACS URL    | Same                                                                |
| Validate `InResponseTo` matches sent ID  | Unsolicited / CSRF                                                  |
| Cache assertion `ID` for replay defense  | One-time use                                                        |
| Honor `<OneTimeUse>` if present          | Spec compliance                                                     |
| Use ID-based reference, not XPath name   | XSW                                                                 |
| Disable DTDs / external entities         | XXE                                                                 |
| Use C14N-Exclusive only                  | C14N-Inclusive XSW vector                                           |
| Reject SHA-1 in signatures and digests   | Collision-vulnerable                                                |
| Validate `<Issuer>` against expected IdP | Cross-IdP confusion                                                 |
| Validate `<Status>` is `Success`         | Don't accept assertions in a failed `<Response>`                    |

## Common Vulnerabilities

| CVE / Class                                  | Year | Root cause                                          | Mitigation                                          |
|----------------------------------------------|------|-----------------------------------------------------|-----------------------------------------------------|
| XML Signature Wrapping (XSW)                 | 2012 | Verified element ≠ processed element                | ID-based dereference; schema validation              |
| XXE in SAML response                         | various | DTD enabled in parser                            | Disable DTD entirely                                 |
| Padding Oracle on AES-CBC (Jager)            | 2012 | XMLEnc CBC                                          | Use AES-GCM; reject CBC if able                      |
| Bleichenbacher RSA-PKCS#1 v1.5 (XMLEnc)      | 2018 (eFAIL-style on SAML) | RSA-1.5 unwrap            | Use RSA-OAEP                                         |
| Comment-stripping bug in `xmlsec`            | 2018 | Canonicalization quirk allowed `alice<!--x-->@evil` to be read as `alice@evil` | Update libs to ≥ 2018; use SAML-Toolkits ≥ patched |
| RelayState fixation / open-redirect          | ongoing | Trusted RelayState as URL                       | Whitelist; treat as opaque key                       |
| Replay (no cache)                            | ongoing | No assertion-ID cache                           | Cache ID until `NotOnOrAfter`+skew                   |
| Issuer-only trust                            | ongoing | "If Issuer matches metadata, accept"            | Always verify signature                              |
| HTTP-Redirect signature stripping            | ongoing | Server didn't verify query-string signature     | Verify `Signature=` in HTTP-Redirect                 |
| Signed-but-not-bound binding                 | ongoing | Endpoint accepted message via wrong binding     | Reject if `ProtocolBinding` doesn't match            |

## Replay Defense

```text
On AssertionConsumerService(POST):
  1. Parse assertion (no DTD).
  2. Verify signature.
  3. Validate Conditions / Subject / Audience.
  4. Compute key = (Issuer, Assertion.ID).
  5. Atomic INSERT into table assertion_seen (key, NotOnOrAfter).
     - If conflict → HTTP 401 "assertion replay detected".
  6. Continue.
  7. Background job DELETE FROM assertion_seen WHERE NotOnOrAfter < now().
```

The cache only needs to cover until `NotOnOrAfter` plus clock skew (typically 5 min). For a load-balanced SP, the cache must be shared (Redis/DB) — not in-process.

`<OneTimeUse>` in `<Conditions>` is a hint to also cache; treat it as an unconditional "must cache this assertion ID".

## Audit + Logging

| Item                                        | Log? | Redact?                                         |
|---------------------------------------------|------|--------------------------------------------------|
| Login success                               | Yes  | NameID hashed, Issuer, AuthnContextClassRef, IP, User-Agent |
| Login failure                               | Yes  | Reason code, Issuer, IP                          |
| Assertion ID                                | Yes  | Yes (truncate)                                   |
| SessionIndex                                | Yes  | Truncate                                         |
| Email / displayName                         | DEBUG only | Yes — PII                                  |
| Full XML response                           | DEBUG only | Yes — never to INFO                       |
| Signature value                             | No   | —                                                |
| Cert fingerprint at validation time         | Yes  | —                                                |
| Logout success / failure                    | Yes  | Reason                                            |

Never log the SAMLResponse body to a centralized aggregator at INFO — they'll often appear in support tickets, spilling PII. Hash the NameID (e.g. SHA-256) for join keys in analytics.

## Common Errors (verbatim text)

| Error                                                        | Cause                                                       | Fix                                                |
|--------------------------------------------------------------|-------------------------------------------------------------|----------------------------------------------------|
| `Invalid issuer in the Assertion/Response`                   | `<Issuer>` doesn't match metadata-pinned `entityID`         | Check IdP entityID; case-sensitive trailing-slash  |
| `Could not validate timestamp: not yet valid`                | Clock skew (`now < NotBefore`)                              | NTP sync; raise tolerance to 60–300 s              |
| `Could not validate timestamp: expired`                      | Clock skew (`now >= NotOnOrAfter`) or stale assertion       | NTP sync; raise tolerance                          |
| `Recipient is not valid`                                     | `SubjectConfirmationData/@Recipient` ≠ ACS URL              | Set ACS URL exactly (https vs http, port, path)    |
| `Signature validation failed. SAML Response rejected`        | Wrong cert pin, modified assertion, wrong c14n              | Check fingerprint, C14N-Exclusive, no whitespace mods |
| `Audience restriction failed`                                | `<AudienceRestriction>` doesn't list SP entityID            | Configure IdP to emit SP entityID as audience      |
| `InResponseTo is not valid`                                  | SP didn't store sent `<AuthnRequest>` ID, or attacker forged | Persist outstanding request IDs; check on response |
| `The InResponseTo of the Logout Response is not valid`       | SLO request/response correlation                            | Same                                               |
| `Could not find AssertionConsumerService URL`                | SP metadata or AuthnRequest ACS URL doesn't match           | Align ACS URL across both                          |
| `No NameID found in the Subject of the Assertion`            | IdP didn't emit NameID                                      | Configure IdP NameID release policy                |
| `Reference validation failed`                                | XMLDSig digest mismatch — assertion modified after signing  | Don't re-pretty-print; check whitespace            |
| `The status code of the Response was not Success, was urn:oasis:names:tc:SAML:2.0:status:Responder` | IdP returned an error | Check `<StatusMessage>`                            |
| `xmlsec1: error: msg=signature verification failed`          | xmlsec1 CLI, bad cert or alg                                | Re-pin cert; check alg whitelist                   |
| `INVALID_AUDIENCE` (ADFS event log)                          | ADFS RPT identifier doesn't match SP entityID               | Edit Relying Party Trust → Identifiers             |
| `MSIS7042: The same client browser session has made N requests`        | ADFS replay protection           | Don't re-POST; restart flow                        |

## Common Gotchas

| Broken                                                                                              | Fixed                                                                                                          |
|-----------------------------------------------------------------------------------------------------|----------------------------------------------------------------------------------------------------------------|
| Trusting `<X509SubjectName>` instead of `<X509Certificate>`                                         | Pin the **full cert** (or its SHA-256 fingerprint) from metadata; SubjectName is mutable                       |
| HTTP-Redirect `AuthnRequest` exceeds 8 KB URL limit and IdP returns 414                             | Switch to HTTP-POST binding for large requests                                                                  |
| `RelayState` URL-encoded twice (Redirect) or not-at-all (POST)                                      | Encode according to binding; Redirect: percent-encode; POST: raw form-input                                    |
| Trusting `<Issuer>` because "it matches our config"                                                 | Verify XML signature against pinned cert; Issuer is just a string                                              |
| Mounting both Redirect and POST on the same endpoint and reading wrong field                        | Separate endpoints per binding, or branch on `Content-Type` and `SAMLRequest`/`SAMLResponse` location          |
| Allowing 60-minute clock skew "just in case"                                                        | Run NTP; set skew ≤ 5 min; reject otherwise                                                                    |
| Looking up element by tag name (XPath `//Assertion`) after verifying signature                      | Use `getElementById`; pass the verified node reference to processing                                           |
| Verifying `<Response>` signature but processing inner `<Assertion>` separately without re-verifying | Verify the signature on the element you actually consume                                                       |
| Ignoring `<OneTimeUse>`                                                                             | Cache assertion `ID` until `NotOnOrAfter`                                                                      |
| Trusting `KeyInfo/X509Certificate` embedded in the response                                         | Use only the cert from metadata; embedded cert can be attacker-controlled                                       |
| HTTP-Redirect signature missing `SigAlg`                                                            | Reject — required by spec                                                                                      |
| Accepting `urn:oasis:names:tc:SAML:1.0:protocol`                                                    | Reject anything not 2.0                                                                                        |
| `validUntil` past — but app cached metadata indefinitely                                            | Refresh metadata daily; refuse to accept assertions if metadata expired                                        |
| Signed `<AuthnRequest>` but IdP didn't ask for them                                                 | Set `WantAuthnRequestsSigned` and check IdP metadata flag                                                       |
| Same SP entityID across staging and production                                                      | One entityID per environment; one ACS URL per environment                                                      |
| Logging full SAML response at INFO                                                                  | DEBUG only, with PII redaction                                                                                  |

## Configuration Examples

### python3-saml (Flask SP)

```python
# settings.json
{
  "strict": true,
  "debug": false,
  "sp": {
    "entityId": "https://sp.example.com/metadata",
    "assertionConsumerService": {
      "url": "https://sp.example.com/saml/acs",
      "binding": "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
    },
    "singleLogoutService": {
      "url": "https://sp.example.com/saml/slo",
      "binding": "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect"
    },
    "NameIDFormat": "urn:oasis:names:tc:SAML:2.0:nameid-format:persistent",
    "x509cert": "-----BEGIN CERTIFICATE-----\nMIID...\n-----END CERTIFICATE-----",
    "privateKey": "-----BEGIN RSA PRIVATE KEY-----\nMIIE...\n-----END RSA PRIVATE KEY-----"
  },
  "idp": {
    "entityId": "https://idp.example.com/metadata",
    "singleSignOnService": {
      "url": "https://idp.example.com/sso",
      "binding": "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect"
    },
    "singleLogoutService": {
      "url": "https://idp.example.com/slo",
      "binding": "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect"
    },
    "x509cert": "-----BEGIN CERTIFICATE-----\nMIID...\n-----END CERTIFICATE-----"
  },
  "security": {
    "authnRequestsSigned": true,
    "wantAssertionsSigned": true,
    "wantAssertionsEncrypted": false,
    "wantNameId": true,
    "wantNameIdEncrypted": false,
    "signMetadata": true,
    "signatureAlgorithm": "http://www.w3.org/2001/04/xmldsig-more#rsa-sha256",
    "digestAlgorithm": "http://www.w3.org/2001/04/xmlenc#sha256",
    "rejectDeprecatedAlgorithm": true
  }
}
```

```python
# Flask handler
from onelogin.saml2.auth import OneLogin_Saml2_Auth

@app.route('/saml/acs', methods=['POST'])
def acs():
    auth = OneLogin_Saml2_Auth(prepare_request(request), settings)
    auth.process_response()
    if auth.get_errors():
        abort(401, auth.get_last_error_reason())
    if not auth.is_authenticated():
        abort(401, "not authenticated")
    session['nameid']        = auth.get_nameid()
    session['attrs']         = auth.get_attributes()
    session['session_index'] = auth.get_session_index()
    return redirect(request.form.get('RelayState') or '/')
```

### Spring Security 6 SAML2 (Java SP)

```yaml
spring:
  security:
    saml2:
      relyingparty:
        registration:
          example-idp:
            entity-id: https://sp.example.com/metadata
            assertingparty:
              metadata-uri: https://idp.example.com/metadata
            assertion-consumer-service:
              location: "{baseUrl}/login/saml2/sso/{registrationId}"
            signing:
              credentials:
                - private-key-location: classpath:saml/sp-key.pem
                  certificate-location: classpath:saml/sp.crt
```

```java
@Configuration
public class SecurityConfig {
  @Bean
  SecurityFilterChain chain(HttpSecurity http) throws Exception {
    http
      .authorizeHttpRequests(a -> a.anyRequest().authenticated())
      .saml2Login(s -> {})
      .saml2Logout(s -> {});
    return http.build();
  }
}
```

### passport-saml (Node SP)

```javascript
const { Strategy: SamlStrategy } = require('@node-saml/passport-saml');

passport.use(new SamlStrategy({
  callbackUrl: 'https://sp.example.com/saml/acs',
  issuer:      'https://sp.example.com/metadata',
  entryPoint:  'https://idp.example.com/sso',
  idpCert:     fs.readFileSync('idp.crt', 'utf8'),
  privateKey:  fs.readFileSync('sp.key',  'utf8'),
  publicCert:  fs.readFileSync('sp.crt',  'utf8'),
  identifierFormat: 'urn:oasis:names:tc:SAML:2.0:nameid-format:persistent',
  signatureAlgorithm: 'sha256',
  digestAlgorithm:    'sha256',
  wantAssertionsSigned: true,
  wantAuthnResponseSigned: true,
  validateInResponseTo: 'always',
  cacheProvider: redisCache,        // shared replay cache
  acceptedClockSkewMs: 5 * 60 * 1000
}, (profile, done) => done(null, {
  nameID:        profile.nameID,
  sessionIndex:  profile.sessionIndex,
  attrs:         profile.attributes
})));
```

### crewjam/saml (Go SP)

```go
package main

import (
  "crypto/rsa"
  "crypto/x509"
  "encoding/pem"
  "net/http"
  "net/url"
  "os"

  "github.com/crewjam/saml/samlsp"
)

func main() {
  keyPEM, _   := os.ReadFile("sp.key")
  certPEM, _  := os.ReadFile("sp.crt")
  keyBlock, _ := pem.Decode(keyPEM)
  certBlock,_ := pem.Decode(certPEM)

  key,  _ := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
  cert, _ := x509.ParseCertificate(certBlock.Bytes)

  idpMetadataURL, _ := url.Parse("https://idp.example.com/metadata")
  rootURL,        _ := url.Parse("https://sp.example.com")

  sp, _ := samlsp.New(samlsp.Options{
    URL:               *rootURL,
    Key:               key.(*rsa.PrivateKey),
    Certificate:       cert,
    IDPMetadataURL:    idpMetadataURL,
    SignRequest:       true,
    AllowIDPInitiated: false,
  })

  http.Handle("/", sp.RequireAccount(http.HandlerFunc(handler)))
  http.Handle("/saml/", sp)
  http.ListenAndServe(":8443", nil)
}
```

### SimpleSAMLphp (PHP IdP/SP)

```php
// authsources.php — SP role
$config = [
  'default-sp' => [
    'saml:SP',
    'entityID'        => 'https://sp.example.com/metadata',
    'idp'             => 'https://idp.example.com/metadata',
    'discoURL'        => null,
    'NameIDPolicy'    => [
      'Format'        => 'urn:oasis:names:tc:SAML:2.0:nameid-format:persistent',
      'AllowCreate'   => true,
    ],
    'sign.authnrequest'    => true,
    'WantAssertionsSigned' => true,
    'redirect.sign'        => true,
    'redirect.validate'    => true,
    'privatekey'           => 'sp.key',
    'certificate'          => 'sp.crt',
  ],
];
```

### Sustainsys.Saml2 (.NET Core SP)

```csharp
services.AddAuthentication(SignInScheme = "Cookies")
  .AddCookie("Cookies")
  .AddSaml2(opt => {
    opt.SPOptions.EntityId   = new EntityId("https://sp.example.com/metadata");
    opt.SPOptions.ServiceCertificates.Add(new X509Certificate2("sp.pfx", "pw"));
    opt.IdentityProviders.Add(new IdentityProvider(
      new EntityId("https://idp.example.com/metadata"),
      opt.SPOptions) {
        MetadataLocation       = "https://idp.example.com/metadata",
        LoadMetadata           = true,
        WantAuthnRequestsSigned = true,
      });
  });
```

## SAML 2.0 vs OIDC

| Feature                | SAML 2.0                                    | OIDC                                          |
|------------------------|---------------------------------------------|-----------------------------------------------|
| Year, body             | 2005, OASIS                                 | 2014, OpenID Foundation                       |
| Underlying             | XML / SOAP                                  | OAuth 2.0 / JSON / JWT                        |
| Token format           | XML Assertion (5–30 KB)                     | JWT id_token (1–4 KB)                         |
| Signing                | XMLDSig + C14N-Exclusive                    | JWS compact (header.payload.signature)         |
| Encryption             | XMLEnc                                      | JWE                                           |
| Discovery              | Metadata XML                                | `/.well-known/openid-configuration`           |
| Front-channel SSO      | HTTP-Redirect → HTTP-POST                   | Authorization Code + PKCE                     |
| Native client          | ECP (clunky)                                | First-class                                   |
| Mobile / SPA           | Awkward                                     | Native                                        |
| Logout                 | SLO (brittle)                               | RP-Initiated, Front-Channel, Back-Channel     |
| Session index          | `<AuthnStatement SessionIndex>`             | `sid` claim (Front-Channel/Back-Channel logout) |
| Refresh                | None (re-auth)                              | Refresh tokens                                |
| Bearer assertion       | Yes (default)                               | Yes (default)                                 |
| Holder-of-Key          | Yes (rare)                                  | Yes (DPoP, mTLS)                              |
| Federation tooling     | InCommon, eduGAIN                           | OIDC Federation 1.0 (newer)                   |
| Common in              | Enterprise, B2B, higher-ed, gov             | Mobile, SPA, IoT, social                      |
| Library complexity     | High (XML, XSW)                             | Lower (JOSE)                                  |
| Debug ergonomics       | SAML-tracer, `xmlsec1`                      | `jq` + `jwt.io`                               |

## Migration to OIDC

| Question                                                              | Decision                                       |
|-----------------------------------------------------------------------|------------------------------------------------|
| Does an existing IdP emit SAML only?                                  | Stay SAML, or run a SAML→OIDC bridge (e.g. Keycloak, Ory Hydra w/ SATOSA, Okta) |
| Is SP a mobile / native / SPA?                                        | Migrate to OIDC                                |
| Does compliance require SAML?                                         | Stay SAML (e.g. some FICAM use-cases)          |
| Are tokens > 4 KB and breaking proxies?                               | Migrate to OIDC                                |
| Greenfield SP?                                                        | OIDC                                           |
| Need fine-grained MFA AuthnContext?                                   | Either; OIDC `acr` claim equivalent            |
| Need user-attribute release governance?                               | SAML (federation) or OIDC + scopes/claims      |

**Transition strategies:**

1. **Dual-stack SP** — accept SAML and OIDC; let the IdP pick.
2. **OIDC fronting** — run an OIDC IdP that, behind the scenes, federates to a SAML IdP (Keycloak, SATOSA, Auth0).
3. **Token-broker** — central broker accepts both; downstream SPs use OIDC.
4. **App-by-app** — migrate one SP at a time; keep the SAML IdP available indefinitely.

## ECP — Enhanced Client/Proxy Profile

ECP lets a non-browser client (`curl`, mobile app, command-line tool) do SAML. The transport is **PAOS** (Reverse SOAP over HTTP). Used historically by Shibboleth and EDS for non-browser federations.

```bash
# Pseudocode — actual ECP requires PAOS headers
curl -k --user 'alice:pass' \
     -H 'Accept: text/html, application/vnd.paos+xml' \
     -H 'PAOS: ver="urn:liberty:paos:2003-08";"urn:oasis:names:tc:SAML:2.0:profiles:SSO:ecp"' \
     https://sp.example.com/protected
```

The flow:

1. Client GET to SP with PAOS headers.
2. SP returns SOAP envelope containing `<AuthnRequest>`.
3. Client POSTs SOAP to IdP ECP endpoint with HTTP Basic auth (or other).
4. IdP returns SOAP `<Response>` containing assertion.
5. Client POSTs the assertion to SP ACS.
6. SP returns the protected resource.

ECP is uncommon; most modern non-browser clients use OIDC.

## WS-Federation vs SAML 2.0

WS-Federation 1.2 (Microsoft, OASIS 2009) is a separate federation protocol that **carries SAML assertions** (or SAML 1.1 assertions) over a different request/response wire format. It was the default for ADFS until ADFS 3.0 (Server 2012R2), which speaks both.

| Aspect              | SAML 2.0                                  | WS-Federation                                |
|---------------------|-------------------------------------------|----------------------------------------------|
| Transport           | HTTP-Redirect, HTTP-POST                  | `wsignin1.0` query / form                    |
| Token format        | SAML 2.0 Assertion                        | Usually SAML 1.1 (sometimes 2.0) Assertion   |
| Signing             | XMLDSig                                   | XMLDSig                                      |
| Logout              | SLO                                       | `wsignout1.0`                                |
| Common in           | Cross-vendor                              | Microsoft stack (legacy ADFS, SharePoint)    |

If your IdP is ADFS, prefer SAML 2.0 endpoints; WS-Fed is legacy. Azure AD / Entra ID supports both.

## Tools for Debugging

| Tool                      | What                                                                                    |
|---------------------------|-----------------------------------------------------------------------------------------|
| SAML-tracer               | Firefox/Chrome extension; captures SAML messages mid-browser                             |
| samltool.com              | OneLogin's online encoder/decoder/validator (pre-prod only — never production data)      |
| `xmlsec1`                 | CLI for XMLDSig signing/verification                                                    |
| `xmllint`                 | Schema validation, pretty-print, canonicalization                                       |
| `openssl x509`            | Inspect signing certs (`openssl x509 -in idp.crt -noout -text -fingerprint -sha256`)    |
| Keycloak / Shibboleth dev IdP | Local IdP for testing                                                               |
| `aws-cli sts assume-role-with-saml` | Test AWS SAML federation                                                       |
| Wireshark + TLS keylog    | Wire-level for SOAP/back-channel                                                        |
| `samltest.id`             | Public test IdP and SP from Shibboleth                                                  |
| `mock-saml` (npm)         | Lightweight mock IdP                                                                    |

```bash
# Decode a SAMLRequest from a redirect URL
python3 -c "import sys, base64, zlib, urllib.parse; \
  q=urllib.parse.unquote(sys.argv[1]); \
  print(zlib.decompress(base64.b64decode(q), -15).decode())" \
  'fVJNb9swDP0rhu6OZcdpEi...'

# Decode a base64 SAMLResponse from POST form
python3 -c "import sys, base64; \
  print(base64.b64decode(sys.argv[1]).decode())" \
  'PHNhbWxwOlJlc3BvbnNl...'

# Pretty-print
xmllint --format response.xml

# Verify a signed assertion via xmlsec1
xmlsec1 verify --pubkey-cert-pem idp.crt assertion.xml

# Sign for testing
xmlsec1 sign --privkey-pem sp.key --output signed.xml unsigned.xml

# Inspect cert fingerprint
openssl x509 -in idp.crt -noout -fingerprint -sha256

# Compare two cert fingerprints
diff <(openssl x509 -in metadata.crt -noout -fingerprint -sha256) \
     <(openssl x509 -in pinned.crt   -noout -fingerprint -sha256)
```

```text
# Sample SAML-tracer line
POST https://sp.example.com/saml/acs
SAMLResponse=PHNhbWxwOlJlc3BvbnNlIHht...
RelayState=4f1ab2c3
```

## Idioms

| Idiom                                                                             | Why                                                          |
|-----------------------------------------------------------------------------------|--------------------------------------------------------------|
| **Use a library** — never hand-roll                                               | XSW, XXE, C14N — too easy to get wrong                       |
| **Validate Audience + NotBefore + NotOnOrAfter + Recipient + Signature**          | The minimum five checks                                      |
| **Pin signing certs from metadata, not from KeyInfo**                             | KeyInfo is attacker-controlled                               |
| **Rotate signing certs with overlap** — publish next cert in metadata before cutover | Avoid a hard failure window                              |
| **Run NTP** — clock skew is the #1 production support call                        | `chrony` or `systemd-timesyncd`                              |
| **Persist outstanding `AuthnRequest` IDs** until used or expired                  | InResponseTo defense                                         |
| **Cache assertion `ID` until `NotOnOrAfter`** in shared store                     | Replay defense                                                |
| **One entityID per environment** (staging/prod separate)                          | Prevent cross-env replay                                     |
| **Whitelist RelayState targets** — never raw-redirect                             | Open-redirect                                                |
| **Reject SHA-1**                                                                  | Collision-vulnerable                                          |
| **Reject AES-CBC, RSA-1.5** — accept GCM and OAEP only                            | Padding/Bleichenbacher                                        |
| **Disable XML DTD/external entities**                                             | XXE                                                          |
| **Schema-validate** before processing                                             | XSW catch                                                    |
| **Dereference verified element by ID**, never re-XPath                            | XSW catch                                                    |
| **Treat IdP-initiated as opt-in**                                                 | Unsolicited-response risk                                    |
| **Don't promise SLO** — set short SP session timeouts                             | SLO is brittle                                               |
| **Hash NameID for analytics joins** — don't log raw                               | PII                                                          |
| **Refresh metadata daily** — refuse expired                                       | Cert rollover, federation membership changes                 |
| **Accept clock skew ≤ 5 minutes** — not 60                                        | Long skew = long replay window                               |
| **Test with samltest.id, mock-saml, Keycloak** before contacting your enterprise IdP team | Faster iteration                                     |

## More Verbatim XML Samples

### Signed Response with embedded Assertion (full)

```xml
<samlp:Response xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol"
                xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion"
                ID="_response_a73c8d72e4f1"
                Version="2.0"
                IssueInstant="2026-04-25T10:14:23.450Z"
                Destination="https://sp.example.com/saml/acs"
                InResponseTo="_request_b62a7f81c235">
  <saml:Issuer>https://idp.example.com</saml:Issuer>
  <ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
    <ds:SignedInfo>
      <ds:CanonicalizationMethod Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#"/>
      <ds:SignatureMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"/>
      <ds:Reference URI="#_response_a73c8d72e4f1">
        <ds:Transforms>
          <ds:Transform Algorithm="http://www.w3.org/2000/09/xmldsig#enveloped-signature"/>
          <ds:Transform Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#"/>
        </ds:Transforms>
        <ds:DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha256"/>
        <ds:DigestValue>HV5jpvf6...</ds:DigestValue>
      </ds:Reference>
    </ds:SignedInfo>
    <ds:SignatureValue>q1Yvr6X...</ds:SignatureValue>
    <ds:KeyInfo>
      <ds:X509Data>
        <ds:X509Certificate>MIIDXTCCAkWgAwIBAgIJAKw...</ds:X509Certificate>
      </ds:X509Data>
    </ds:KeyInfo>
  </ds:Signature>
  <samlp:Status>
    <samlp:StatusCode Value="urn:oasis:names:tc:SAML:2.0:status:Success"/>
  </samlp:Status>
  <saml:Assertion ID="_assertion_d927ff14a3e9"
                  Version="2.0"
                  IssueInstant="2026-04-25T10:14:23.450Z">
    <saml:Issuer>https://idp.example.com</saml:Issuer>
    <saml:Subject>
      <saml:NameID Format="urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress">
        alice@example.com
      </saml:NameID>
      <saml:SubjectConfirmation Method="urn:oasis:names:tc:SAML:2.0:cm:bearer">
        <saml:SubjectConfirmationData NotOnOrAfter="2026-04-25T10:19:23.450Z"
                                      Recipient="https://sp.example.com/saml/acs"
                                      InResponseTo="_request_b62a7f81c235"/>
      </saml:SubjectConfirmation>
    </saml:Subject>
    <saml:Conditions NotBefore="2026-04-25T10:14:23.450Z"
                     NotOnOrAfter="2026-04-25T10:19:23.450Z">
      <saml:AudienceRestriction>
        <saml:Audience>https://sp.example.com</saml:Audience>
      </saml:AudienceRestriction>
      <saml:OneTimeUse/>
    </saml:Conditions>
    <saml:AuthnStatement AuthnInstant="2026-04-25T10:14:20.000Z"
                         SessionIndex="_session_8e1f2c45"
                         SessionNotOnOrAfter="2026-04-25T18:14:20.000Z">
      <saml:AuthnContext>
        <saml:AuthnContextClassRef>
          urn:oasis:names:tc:SAML:2.0:ac:classes:PasswordProtectedTransport
        </saml:AuthnContextClassRef>
      </saml:AuthnContext>
    </saml:AuthnStatement>
    <saml:AttributeStatement>
      <saml:Attribute Name="urn:oid:0.9.2342.19200300.100.1.3"
                      NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:uri"
                      FriendlyName="mail">
        <saml:AttributeValue xsi:type="xs:string"
                             xmlns:xs="http://www.w3.org/2001/XMLSchema"
                             xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
          alice@example.com
        </saml:AttributeValue>
      </saml:Attribute>
      <saml:Attribute Name="urn:oid:2.5.4.42" FriendlyName="givenName">
        <saml:AttributeValue xsi:type="xs:string">Alice</saml:AttributeValue>
      </saml:Attribute>
      <saml:Attribute Name="urn:oid:2.5.4.4" FriendlyName="sn">
        <saml:AttributeValue xsi:type="xs:string">Anderson</saml:AttributeValue>
      </saml:Attribute>
      <saml:Attribute Name="urn:oid:1.3.6.1.4.1.5923.1.1.1.1" FriendlyName="eduPersonAffiliation">
        <saml:AttributeValue xsi:type="xs:string">staff</saml:AttributeValue>
        <saml:AttributeValue xsi:type="xs:string">member</saml:AttributeValue>
      </saml:Attribute>
      <saml:Attribute Name="http://schemas.microsoft.com/ws/2008/06/identity/claims/groups">
        <saml:AttributeValue>cn=engineering,ou=groups,dc=example,dc=com</saml:AttributeValue>
        <saml:AttributeValue>cn=admins,ou=groups,dc=example,dc=com</saml:AttributeValue>
      </saml:Attribute>
    </saml:AttributeStatement>
  </saml:Assertion>
</samlp:Response>
```

### Encrypted Assertion (XML Encryption)

```xml
<samlp:Response>
  <saml:Issuer>https://idp.example.com</saml:Issuer>
  <samlp:Status><samlp:StatusCode Value="...:Success"/></samlp:Status>
  <saml:EncryptedAssertion>
    <xenc:EncryptedData xmlns:xenc="http://www.w3.org/2001/04/xmlenc#"
                        Type="http://www.w3.org/2001/04/xmlenc#Element">
      <xenc:EncryptionMethod Algorithm="http://www.w3.org/2009/xmlenc11#aes256-gcm"/>
      <ds:KeyInfo xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
        <xenc:EncryptedKey>
          <xenc:EncryptionMethod Algorithm="http://www.w3.org/2009/xmlenc11#rsa-oaep">
            <xenc:OAEPparams/>
            <ds:DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha256"/>
            <xenc11:MGF Algorithm="http://www.w3.org/2009/xmlenc11#mgf1sha1"
                        xmlns:xenc11="http://www.w3.org/2009/xmlenc11#"/>
          </xenc:EncryptionMethod>
          <ds:KeyInfo>
            <ds:X509Data><ds:X509Certificate>MII...</ds:X509Certificate></ds:X509Data>
          </ds:KeyInfo>
          <xenc:CipherData><xenc:CipherValue>encrypted-AES-key...</xenc:CipherValue></xenc:CipherData>
        </xenc:EncryptedKey>
      </ds:KeyInfo>
      <xenc:CipherData><xenc:CipherValue>encrypted-assertion-payload...</xenc:CipherValue></xenc:CipherData>
    </xenc:EncryptedData>
  </saml:EncryptedAssertion>
</samlp:Response>
```

### LogoutRequest (SP-initiated SLO)

```xml
<samlp:LogoutRequest xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol"
                     xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion"
                     ID="_logout_req_3a2b91"
                     Version="2.0"
                     IssueInstant="2026-04-25T11:00:00Z"
                     Destination="https://idp.example.com/saml/slo"
                     NotOnOrAfter="2026-04-25T11:05:00Z">
  <saml:Issuer>https://sp.example.com</saml:Issuer>
  <saml:NameID Format="urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress">
    alice@example.com
  </saml:NameID>
  <samlp:SessionIndex>_session_8e1f2c45</samlp:SessionIndex>
</samlp:LogoutRequest>
```

### LogoutResponse

```xml
<samlp:LogoutResponse ID="_logout_resp_3a2b92"
                      Version="2.0"
                      IssueInstant="2026-04-25T11:00:01Z"
                      Destination="https://sp.example.com/saml/slo"
                      InResponseTo="_logout_req_3a2b91">
  <saml:Issuer>https://idp.example.com</saml:Issuer>
  <samlp:Status>
    <samlp:StatusCode Value="urn:oasis:names:tc:SAML:2.0:status:Success"/>
  </samlp:Status>
</samlp:LogoutResponse>
```

## NameID Format Reference

| Format URI | Use case | Example value |
|---|---|---|
| `urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress` | Email as identifier | `alice@example.com` |
| `urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified` | App decides interpretation | varies |
| `urn:oasis:names:tc:SAML:2.0:nameid-format:persistent` | Opaque pairwise (per-SP) pseudonym | `9b1c8e6f-3a4d-...` |
| `urn:oasis:names:tc:SAML:2.0:nameid-format:transient` | Per-session opaque (anonymous) | `_4f8a2b...` |
| `urn:oasis:names:tc:SAML:2.0:nameid-format:entity` | SAML entity (system-to-system) | `https://idp.example.com` |
| `urn:oasis:names:tc:SAML:1.1:nameid-format:X509SubjectName` | X.509 subject DN | `CN=Alice,O=Example,C=US` |
| `urn:oasis:names:tc:SAML:1.1:nameid-format:WindowsDomainQualifiedName` | Windows DOMAIN\user | `EXAMPLE\alice` |
| `urn:oasis:names:tc:SAML:1.1:nameid-format:kerberos` | Kerberos principal | `alice@EXAMPLE.COM` |

## AuthnContext Class Refs Reference

| URI | Meaning |
|---|---|
| `urn:oasis:names:tc:SAML:2.0:ac:classes:Password` | Plain password |
| `urn:oasis:names:tc:SAML:2.0:ac:classes:PasswordProtectedTransport` | Password over TLS (most common) |
| `urn:oasis:names:tc:SAML:2.0:ac:classes:TLSClient` | Client certificate (mutual TLS) |
| `urn:oasis:names:tc:SAML:2.0:ac:classes:X509` | X.509 certificate |
| `urn:oasis:names:tc:SAML:2.0:ac:classes:Smartcard` | Smartcard |
| `urn:oasis:names:tc:SAML:2.0:ac:classes:SmartcardPKI` | Smartcard with PKI cert |
| `urn:oasis:names:tc:SAML:2.0:ac:classes:Kerberos` | Kerberos ticket |
| `urn:oasis:names:tc:SAML:2.0:ac:classes:MultiFactorContract` | MFA (any combination) |
| `urn:oasis:names:tc:SAML:2.0:ac:classes:TimeSyncToken` | TOTP / hardware token |
| `urn:oasis:names:tc:SAML:2.0:ac:classes:PreviousSession` | Existing SSO session reused |
| `urn:oasis:names:tc:SAML:2.0:ac:classes:unspecified` | IdP doesn't disclose method |

The `<saml:Conditions>` `<samlp:RequestedAuthnContext Comparison="exact|minimum|maximum|better">` element on the AuthnRequest lets the SP demand a particular factor strength.

## Library-Specific Error Catalog

### python3-saml (OneLogin)

```text
OneLogin_Saml2_Error: SAML Response not found, Only supported HTTP_POST Binding
# Cause: GET request to ACS endpoint or missing SAMLResponse parameter
# Fix: ensure HTTP-POST binding for the AssertionConsumerService

OneLogin_Saml2_Error: SAML Response could not be processed
# Cause: malformed XML, base64 decoding failed
# Fix: check intermediate URL-decoding, ensure base64 padding intact

OneLogin_Saml2_ValidationError: The response was received at X but expected Y
# Cause: Destination attribute mismatch with configured ACS URL (case-sensitive!)
# Fix: align Destination and AssertionConsumerService URL exactly

OneLogin_Saml2_ValidationError: The response has an invalid signed element
# Cause: signature wraps wrong element (XSW), or KeyInfo doesn't reference signed cert
# Fix: validate library defends against XSW (this one does); check IdP cert in trust store

OneLogin_Saml2_ValidationError: Signature validation failed. SAML Response rejected.
# Cause: cert mismatch, RSA key changed without metadata update
# Fix: refresh metadata; verify x509cert in settings.json matches IdP

OneLogin_Saml2_ValidationError: NotBefore and NotOnOrAfter conditions failed.
# Cause: clock skew between SP and IdP
# Fix: NTP both sides; allow up to 300s skew if network unreliable
```

### Sustainsys.Saml2 (.NET)

```text
Sustainsys.Saml2.SignatureValidationException: Signature didn't verify.
# Cause: cert in metadata doesn't match cert that signed the response
# Fix: ensure SPOptions.IdentityProviders has correct signing cert

Sustainsys.Saml2.SamlException: Invalid status code <Responder>
# Cause: IdP returned an error status (often AuthnRequest issue)
# Fix: inspect response StatusMessage / SecondLevelStatus

Sustainsys.Saml2.SamlException: Configured to require encryption but assertion is not encrypted.
# Cause: SP options demand encrypted assertions; IdP not configured to encrypt
# Fix: configure IdP to encrypt; or relax SP requirement
```

### Spring Security SAML

```text
org.springframework.security.saml2.Saml2Exception: Invalid status code [Responder]
org.springframework.security.saml2.Saml2Exception: Could not parse encrypted attribute
org.springframework.security.saml2.Saml2Exception: Invalid signature for object
org.springframework.security.saml2.Saml2Exception: Did not decrypt response
org.springframework.security.saml2.Saml2Exception: The destination [X] does not match
org.springframework.security.saml2.Saml2Exception: assertion contains AuthnStatement that is too old
```

### Shibboleth SP

```text
ERROR Shibboleth.SSO.SAML2 [1]: Authentication failed.
ERROR XMLTooling.TrustEngine.PKIX [1]: certificate path could not be validated
ERROR Shibboleth.NameIDPolicy [1]: format URI not supported
ERROR Shibboleth.SAML2.SSO [1]: AssertionConsumerService not found in metadata for SP
```

## XSW Attack Variants Catalog

XML Signature Wrapping (XSW) — the "signature is valid but verifier is checking the wrong element" class. There are eight named variants:

### XSW1 — duplicated Response with attacker payload

```xml
<Response>
  <Response>  <!-- attacker-injected wrapper, no signature -->
    <Assertion>...attacker's claim...</Assertion>
  </Response>
  <Signature>...covers the original (now-displaced) Response...</Signature>
  <Assertion>...legitimate but unused...</Assertion>
</Response>
```

If the verifier looks up the signed assertion by its SHA digest but processes the *first* Assertion encountered (the attacker's), authentication is bypassed.

### XSW2 — signature precedes assertion

Similar to XSW1 but signature placed before the legitimate assertion, with attacker assertion inserted afterward at sibling level.

### XSW3 — wrap with extension

Attacker injects an `<Extensions>` element containing the original signed assertion, then adds their own assertion as a sibling.

### XSW4 — wrap entire response

Attacker injects a new `<Response>` wrapping the attacker's assertion, with the original (signed) Response as its child.

### XSW5 — wrap with pre-existing extension

Like XSW3 but exploits an existing Extensions element from the IdP, hiding the malicious element among legitimate ones.

### XSW6 — signature in extension

Move the signature itself into an Extensions element so xmlsec processes it but the parser sees attacker's data first.

### XSW7 — wrap inside assertion

Attacker injects an assertion wrapper inside the original (signed) assertion.

### XSW8 — combination

Recombine multiple wrapping techniques (e.g., XSW1 + XSW7 nested) to bypass libraries that defend against single-variant XSW.

### Defense

- Validate the assertion's parent is the Response root (check `parentNode === Response`).
- Reject any document with multiple Assertion elements.
- Reject any document whose `<ds:Reference URI>` doesn't match the structurally-first Assertion ID.
- Use libraries that explicitly defend against all 8 variants (`python3-saml`, OpenSAML 4+).

## IdP Discovery Patterns

### Common Domain Cookie (legacy, deprecated)

```text
Set-Cookie: _saml_idp=https://idp.example.com; Domain=.federation.example;
            Secure; HttpOnly; SameSite=None
```

User's browser visits `wayf.federation.example` and SP redirects via this domain to read the cookie identifying user's home IdP. Deprecated due to third-party-cookie phase-out.

### WAYF Service (Where Are You From)

Federation-hosted discovery service; user picks their IdP from a list. Sample InCommon WAYF: `https://discovery.incommon.org/DS/WAYF`.

### Email Domain Mapping

Most modern: ask the user for email, parse the domain, route to the IdP for that domain.

```python
def find_idp(email: str) -> str:
    domain = email.split("@", 1)[1].lower()
    mapping = {
        "example.com": "https://idp.example.com",
        "subsidiary.example": "https://idp.subsidiary.example",
    }
    return mapping.get(domain) or DEFAULT_IDP_DISCOVERY_URL
```

### Subdomain-based

Each tenant gets `tenant.app.example`; SP introspects host header.

### Explicit IdP Selector Buttons

"Sign in with Google / Microsoft / Okta" — user picks visually. Cleanest UX, no auto-detection.

## See Also

- `owasp-auth` — broader authentication threats and defenses
- `openssl` — generating SP/IdP signing certs, key rotation
- `gpg` — out-of-band metadata signature verification
- `tls` — transport-layer protection underneath SAML
- `ssh` — orthogonal but often discussed together for service auth
- `polyglot` — XML serialization-format risks that touch SAML
- `vault` — storing SAML signing keys

## References

- OASIS, *Assertions and Protocols for the OASIS Security Assertion Markup Language (SAML) V2.0*, March 2005, `saml-core-2.0-os.pdf`
- OASIS, *Bindings for SAML V2.0*, March 2005, `saml-bindings-2.0-os.pdf`
- OASIS, *Profiles for SAML V2.0*, March 2005, `saml-profiles-2.0-os.pdf`
- OASIS, *Metadata for SAML V2.0*, March 2005, `saml-metadata-2.0-os.pdf`
- OASIS, *Authentication Context for SAML V2.0*, March 2005, `saml-authn-context-2.0-os.pdf`
- OASIS, *Conformance Requirements for SAML V2.0*, March 2005
- OASIS, *Approved Errata to OASIS Security Assertion Markup Language (SAML) V2.0*
- W3C, *XML-Signature Syntax and Processing (Second Edition)*, June 2008
- W3C, *Exclusive XML Canonicalization Version 1.0*, July 2002
- W3C, *XML Encryption Syntax and Processing Version 1.1*, April 2013
- OWASP, *SAML Security Cheat Sheet*
- OWASP, *XML External Entity (XXE) Prevention Cheat Sheet*
- Somorovsky, Mayer, Schwenk, Kampmann, Jensen, *On Breaking SAML: Be Whoever You Want to Be*, USENIX Security 2012
- Mainka, Mladenov, Schwenk, Wich, *SoK: SAML in Browser-based Single Sign-On*, IEEE EuroS&P 2017
- Jager, Schwenk, Somorovsky, *On the Security of TLS-DHE in the Standard Model*, CRYPTO 2012 (background on padding-oracle)
- IETF RFC 4346 — TLS 1.1 (transport)
- IETF RFC 6749 — OAuth 2.0 (for OIDC comparison)
- OpenID Foundation, *OpenID Connect Core 1.0* (for OIDC comparison)
- Shibboleth Consortium documentation, `shibboleth.net`
- InCommon Federation, `incommon.org`
- eduGAIN, `edugain.org`
- REFEDS Assurance Framework, `refeds.org/assurance`
- Microsoft, *AD FS Troubleshooting* MSIS event reference
- xmlsec project, `www.aleksey.com/xmlsec`
- SAML-Toolkits libraries, `github.com/SAML-Toolkits`

## Library Error Catalog (Extended)

Verbatim errors from the most-deployed SAML SP libraries. Copy-paste into a search engine if unfamiliar.

### python3-saml (OneLogin / SAML-Toolkits)

| Verbatim error | Cause | Fix |
|----------------|-------|-----|
| `OneLogin_Saml2_Error: SAML Response not found` | `process_response()` called without `SAMLResponse` in POST body | Verify HTTP-POST binding; check IdP form auto-submit |
| `Settings: invalid array... Element x must contain a y` | `settings.json` schema mismatch | `OneLogin_Saml2_Settings(settings, sp_validation_only=False)` and read errors list |
| `invalid_response: SAML Response could not be processed` | Catch-all (sig, cert, time) | Enable `debug=true`; check `errorReason` |
| `invalid_response: The Response is not signed and the SP requires it` | IdP omitted Response-level signature | Set `wantAssertionsSigned: true` OR fix IdP to sign Response |
| `invalid_response: timestamps are out of range` | Clock skew or expired assertion | Sync NTP; check `NotBefore` / `NotOnOrAfter` |
| `invalid_response: The InResponseTo of the Response... does not match` | Stale or replayed | Don't cache RequestID across browser sessions |

### ruby-saml (OneLogin)

| Verbatim error | Cause | Fix |
|----------------|-------|-----|
| `OneLogin::RubySaml::ValidationError: Current time is on or after NotOnOrAfter condition` | Assertion expired in transit | Sync NTP; raise `allowed_clock_drift` |
| `Current time is earlier than NotBefore condition` | SP clock behind IdP | Sync NTP both sides |
| `The signature method is not supported` | SHA-1 vs SHA-256 mismatch | Set `security[:signature_method]` to match IdP |
| `Digest mismatch` | XML modified post-signing (often by a proxy) | Disable XML pretty-printing on intermediaries |
| `An error occurred while loading the XML` | Malformed XML / encoding issue | Verify base64 decode; check Content-Type |

### Spring Security SAML (Java 6.x)

| Verbatim error | Cause | Fix |
|----------------|-------|-----|
| `SignatureException: Cryptographic security validation of signature could not be performed` | Wrong/expired cert in metadata | Refresh IdP metadata; check `KeyDescriptor` fingerprint |
| `MetadataResolverException: Metadata document was invalid` | Schema-invalid XML | `xmllint --schema saml-metadata-2.0.xsd metadata.xml` |
| `AuthenticationServiceException: Response is not signed when ResponseSignatureValidation is required` | SP demands Response sig, IdP only signs Assertion | Configure IdP to sign Response, OR `wantAssertionsSigned=true` |
| `Saml2AuthenticationException: invalid_destination` | `Destination` ≠ SP's ACS URL | Update IdP-side ACS URL OR fix SP entityID |

### .NET / ITfoxtec.Identity.Saml2

| Verbatim error | Cause | Fix |
|----------------|-------|-----|
| `Saml2RequestException: Signature validation failed` | Cert mismatch or modified XML | Re-import IdP signing cert; verify exclusive c14n applied |
| `Saml2ResponseException: Response is not in successful status` | StatusCode = `Responder` / `RequestDenied` | Read `<StatusMessage>` — IdP rejected the AuthnRequest |
| `Saml2RequestException: SP-initiated logout request requires NameID` | SLO without remembered NameID | Persist NameID at login; replay for SLO |

### Shibboleth SP (mod_shib / shibd, 3.x)

| Verbatim error | Cause | Fix |
|----------------|-------|-----|
| `opensaml::SecurityPolicyException: Message was signed, but signature could not be verified` | Out-of-date IdP cert | `metadata-providers.xml` reload interval; manual refetch |
| `opensaml::FatalProfileException: Unable to locate metadata for identity provider (entityID)` | EntityID typo or metadata not yet fetched | Compare exact `entityID` strings |
| `XMLObjectChildrenList::add: child cannot be added` | Malformed assertion (often XSW probe) | Patch SP; investigate as security event |
| `Session expired` | Assertion lifetime elapsed | Tune `<SessionInitiator>` lifetime + `cacheTimeout` |

### Common Recovery Sequence

When SAML breaks, this in order resolves ~90% of cases:

1. `xmllint --format response.xml` — confirm valid XML
2. `xmlsec1 --verify --pubkey-cert-pem idp.pem response.xml` — verify signature
3. `openssl x509 -in idp.pem -dates -noout` — cert not expired
4. `date -u && grep -E 'NotBefore|NotOnOrAfter' response.xml` — timestamps in range
5. `xmllint --xpath "//*[local-name()='Audience']/text()" response.xml` — `<Audience>` matches SP entityID
6. `xmllint --xpath "//*[@Destination]/@Destination" response.xml` — `<Destination>` matches ACS URL
7. Re-fetch IdP metadata; diff against cached version
8. Bump SP debug; capture next failed attempt with full XML

If 1-7 pass but auth still fails, the issue is in SP application code (session handling, attribute mapping, group-claim parsing) — not SAML itself.

## Library Compatibility Matrix

| SP library | Lang | Latest | Encrypted Assertion | SLO | Metadata | Maintenance |
|------------|------|--------|--------------------:|----:|---------:|-------------|
| python3-saml | Python | 1.16+ | ✓ | ✓ | ✓ | active |
| ruby-saml | Ruby | 1.16+ | ✓ | ✓ | ✓ | active |
| Spring Security SAML | Java | 6.x | ✓ | ✓ | ✓ | active |
| ITfoxtec.Identity.Saml2 | .NET | 4.x | ✓ | ✓ | ✓ | active |
| Shibboleth SP | C++ Apache module | 3.x | ✓ | ✓ | ✓ | active (Internet2) |
| SimpleSAMLphp | PHP | 2.x | ✓ | ✓ | ✓ | active |
| node-saml | Node | 5.x | partial | ✓ | ✓ | active |
| passport-saml | Node Passport plugin | 4.x | partial | partial | ✓ | active |
| go-saml (russellhaering) | Go | 0.4.x | ✓ | ✓ | ✓ | active |
| crewjam/saml | Go | 0.4.x | ✓ | ✓ | ✓ | active |

Errors above taken from latest stable releases. Older versions may differ — upgrade before debugging obscure errors.

## Production Hardening Checklist

Before shipping a SAML SP integration to production, every box must be ticked:

- [ ] Sign every AuthnRequest sent to the IdP (not just receive signed Responses).
- [ ] Verify the Response signature AND every Assertion signature — never one without the other.
- [ ] Reject `NameIDFormat: unspecified` unless documented and audited.
- [ ] Pin the IdP signing certificate fingerprint AND fail-closed on unexpected change (with an alert and a runbook for IdP key rotation).
- [ ] Enforce a maximum assertion lifetime ≤ 5 minutes between `IssueInstant` and `NotOnOrAfter`.
- [ ] Reject any assertion whose `InResponseTo` doesn't match a recently-sent `RequestID` from the same browser session.
- [ ] Validate `<Audience>` strictly against your registered SP `entityID`.
- [ ] Validate `<Destination>` strictly against your ACS URL.
- [ ] Track and refuse replays — store `(ID, IssueInstant)` for at least the assertion's lifetime.
- [ ] Run xmlsec/library version with the latest XSW protections; subscribe to your library's CVE feed.
- [ ] Strip XML comments before signature validation (some libraries diverge here — c.f. CVE-2017-11428).
- [ ] Enforce TLS 1.2+ for the entire login round-trip; reject the assertion if the SP-side request did not arrive over HTTPS.
- [ ] Bind the session cookie to the assertion (e.g., HMAC of NameID) so a stolen cookie alone can't replay the session.
- [ ] Set `Set-Cookie: ... HttpOnly; Secure; SameSite=Lax` on session cookies.
- [ ] Implement Single Logout (SLO) properly — both SP-init and IdP-init paths.
- [ ] Centrally log every assertion (sanitize PII), every signature failure, every clock-skew rejection.
- [ ] Add an alert for >N signature failures per minute — that's an attacker probing for XSW.
- [ ] Test failover: what happens when the IdP is down? What does the user see? Does the SP fail open or fail closed?
- [ ] Document the annual cert-rotation runbook and rehearse it in staging.
- [ ] Add a synthetic monitor that completes a real SAML login every 5 minutes and pages on failure.

A SAML deployment without all 19 boxes ticked is not production-ready, regardless of how impressive the demo looks.
