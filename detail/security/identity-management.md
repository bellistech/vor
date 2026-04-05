# The Theory of Identity Management — Perimeter Dissolution, Protocol Mechanics, and the Zero-Trust Identity Model

> *Identity has replaced the network perimeter as the primary security boundary. Understanding authentication protocol internals, access control models, and identity attack surfaces is essential for designing systems that survive in a world where credentials are the #1 attack vector.*

---

## 1. Identity as the New Perimeter

### The Perimeter Shift

```
Traditional Model (Castle-and-Moat):
  [Internet] --> [Firewall] --> [Trusted Network]
  - Inside the firewall = trusted
  - VPN = extend the trust boundary
  - Identity = secondary (you're already "inside")

Modern Model (Identity-Centric):
  [Users] --> [Identity Provider] --> [Policy Engine] --> [Resources]
  - No trusted network (zero trust)
  - Every access decision requires identity verification
  - Identity = THE security boundary
  - Network location is just one signal among many
```

### Why Identity Won

| Factor | Network Perimeter | Identity Perimeter |
|--------|------------------|-------------------|
| Cloud migration | Cannot protect SaaS | Controls access regardless of location |
| Remote work | VPN does not scale | SSO works from anywhere |
| Lateral movement | Flat network = game over | Each resource requires re-auth |
| Supply chain | Third-party VPN = risk | Federation with granular control |
| IoT proliferation | Cannot firewall everything | Device identity + certificates |

### The Identity Security Stack

```
Layer 5: Governance     Access reviews, SoD, compliance reporting
Layer 4: Authorization  RBAC, ABAC, policy engines (OPA, Cedar)
Layer 3: Authentication SSO, MFA, passwordless, continuous auth
Layer 2: Directory      AD, LDAP, Entra ID, user store
Layer 1: Identity       Provisioning, lifecycle, SCIM, HR integration
Layer 0: Trust Root     PKI, hardware keys, biometric enrollment
```

---

## 2. Authentication Protocol Internals

### Kerberos Ticket Structure

```
TGT (Ticket-Granting Ticket):
+------------------------------------------+
| Encrypted with krbtgt password hash:     |
|   - Client principal name                |
|   - Client realm                         |
|   - Session key (client <-> TGS)         |
|   - Auth time                            |
|   - Start time                           |
|   - End time (default: 10 hours)         |
|   - Renew till (default: 7 days)         |
|   - Client IP addresses (optional)       |
|   - Flags (forwardable, renewable, etc.) |
+------------------------------------------+

Service Ticket:
+------------------------------------------+
| Encrypted with service account hash:     |
|   - Client principal name                |
|   - Client realm                         |
|   - Session key (client <-> service)     |
|   - Auth time                            |
|   - Start time / End time                |
|   - Flags                                |
|   - Authorization data (PAC)             |
+------------------------------------------+

PAC (Privilege Attribute Certificate):
+------------------------------------------+
| - User SID                               |
| - Group SIDs (security group memberships)|
| - Logon info (domain, logon server)      |
| - Signature (KDC key)                    |
| - Signature (server key)                 |
+------------------------------------------+
```

### Kerberos Cryptographic Operations

| Operation | Key Used | Algorithm (Modern AD) |
|-----------|----------|----------------------|
| AS-REQ pre-auth | User's password hash | AES-256 (RC4 legacy) |
| TGT encryption | krbtgt account hash | AES-256 |
| Service ticket encryption | Service account hash | AES-256 |
| Authenticator | Session key | AES-256 |
| PAC signature | KDC key + server key | HMAC-SHA1 |

### SAML Assertion Anatomy

```xml
<saml:Assertion Version="2.0" ID="_abc123" IssueInstant="2026-04-05T10:00:00Z">

  <!-- WHO issued the assertion -->
  <saml:Issuer>https://idp.example.com</saml:Issuer>

  <!-- Digital signature (XML-DSIG) -->
  <ds:Signature>
    <ds:SignedInfo>
      <ds:CanonicalizationMethod Algorithm="exc-c14n"/>
      <ds:SignatureMethod Algorithm="rsa-sha256"/>
      <ds:Reference URI="#_abc123">
        <!-- References the assertion being signed -->
        <ds:DigestMethod Algorithm="sha256"/>
        <ds:DigestValue>base64_hash_of_assertion</ds:DigestValue>
      </ds:Reference>
    </ds:SignedInfo>
    <ds:SignatureValue>base64_rsa_signature</ds:SignatureValue>
  </ds:Signature>

  <!-- WHO the assertion is about -->
  <saml:Subject>
    <saml:NameID Format="emailAddress">jsmith@example.com</saml:NameID>
    <saml:SubjectConfirmation Method="bearer">
      <saml:SubjectConfirmationData
        NotOnOrAfter="2026-04-05T10:05:00Z"
        Recipient="https://sp.example.com/saml/acs"
        InResponseTo="_req456"/>
    </saml:SubjectConfirmation>
  </saml:Subject>

  <!-- WHEN the assertion is valid -->
  <saml:Conditions NotBefore="2026-04-05T09:59:00Z"
                   NotOnOrAfter="2026-04-05T10:05:00Z">
    <saml:AudienceRestriction>
      <saml:Audience>https://sp.example.com</saml:Audience>
    </saml:AudienceRestriction>
  </saml:Conditions>

  <!-- HOW the user authenticated -->
  <saml:AuthnStatement AuthnInstant="2026-04-05T10:00:00Z"
                       SessionIndex="_sess789">
    <saml:AuthnContext>
      <saml:AuthnContextClassRef>
        urn:oasis:names:tc:SAML:2.0:ac:classes:PasswordProtectedTransport
      </saml:AuthnContextClassRef>
    </saml:AuthnContext>
  </saml:AuthnStatement>

  <!-- WHAT attributes the user has -->
  <saml:AttributeStatement>
    <saml:Attribute Name="groups">
      <saml:AttributeValue>Engineering</saml:AttributeValue>
      <saml:AttributeValue>VPN-Users</saml:AttributeValue>
    </saml:Attribute>
    <saml:Attribute Name="department">
      <saml:AttributeValue>Engineering</saml:AttributeValue>
    </saml:Attribute>
  </saml:AttributeStatement>

</saml:Assertion>
```

### SAML Security Validation Checklist

| Check | Attack Prevented |
|-------|-----------------|
| Verify XML signature against IdP certificate | Assertion forgery |
| Validate NotBefore / NotOnOrAfter | Replay attacks |
| Validate Audience matches SP entity ID | Assertion confusion |
| Validate InResponseTo matches original request | CSRF |
| Validate Recipient matches ACS URL | Assertion redirect |
| Check for signature wrapping (signed element = asserted element) | XML signature wrapping |
| Enforce single use (track assertion IDs) | Replay |

### OIDC Token Structure

```
ID Token (JWT):
  Header: {"alg": "RS256", "kid": "key-id-1"}
  Payload: {
    "iss": "https://idp.example.com",      // Issuer
    "sub": "user-id-123",                   // Subject (unique user ID)
    "aud": "client-app-id",                 // Audience (client ID)
    "exp": 1743850800,                      // Expiration (Unix timestamp)
    "iat": 1743847200,                      // Issued at
    "nonce": "random-nonce",                // Replay prevention
    "auth_time": 1743847190,                // When user authenticated
    "acr": "urn:mfa",                       // Auth context (MFA level)
    "amr": ["pwd", "otp"],                  // Auth methods used
    "azp": "client-app-id",                 // Authorized party
    "email": "jsmith@example.com",
    "groups": ["Engineering", "VPN-Users"]
  }
  Signature: RS256(header.payload, private_key)

Access Token:
  - Opaque string OR JWT (implementation-specific)
  - Sent to resource server in Authorization header
  - Validated by resource server (introspection or JWT verification)
  - Scoped to specific permissions (scope claim)

Refresh Token:
  - Opaque, long-lived
  - Used to obtain new access tokens without re-authentication
  - Stored securely (server-side or secure storage)
  - Should be rotated on each use (refresh token rotation)
```

### Protocol Security Comparison

| Attack | SAML Mitigation | OIDC Mitigation | Kerberos Mitigation |
|--------|----------------|-----------------|---------------------|
| Replay | NotOnOrAfter, assertion ID tracking | Token expiration, nonce | Ticket expiration, replay cache |
| Man-in-middle | TLS + signed assertions | TLS + PKCE | Mutual auth, session key |
| Token theft | Short validity, audience restriction | Short-lived access tokens, refresh rotation | Ticket forwarding restrictions |
| Phishing | N/A (phishable) | N/A (phishable without FIDO2) | N/A (password-based AS-REQ) |
| Privilege escalation | Signed attributes from IdP | Signed JWT claims | PAC validation, PAC signing |

---

## 3. MFA Factor Categories and Security Analysis

### Factor Security Hierarchy

From weakest to strongest:

$$\text{SMS OTP} < \text{TOTP} < \text{Push} < \text{Push + Number Match} < \text{FIDO2/WebAuthn}$$

### Why SMS is Weak

| Attack | Mechanism | Difficulty |
|--------|-----------|------------|
| SIM swapping | Social engineer carrier to port number | Low (documented attacks) |
| SS7 interception | Exploit SS7 protocol to intercept SMS | Medium (state-level, organized crime) |
| Malware on phone | Intercept SMS via Android malware | Medium |
| Voicemail hack | Redirect voice OTP to compromised voicemail | Low |

### Why FIDO2/WebAuthn is Strong

FIDO2 binds authentication to the origin (domain), making phishing impossible:

```
Phishing scenario:

Real site:  https://bank.example.com
Phish site: https://bank-example.com (attacker-controlled)

With passwords + TOTP:
  User enters credentials on phish site
  Attacker replays credentials + TOTP to real site
  Result: Account compromised

With FIDO2/WebAuthn:
  Authenticator receives challenge bound to: bank-example.com
  Authenticator signs with key registered for: bank.example.com
  Origins do not match --> Authenticator uses DIFFERENT key (or none)
  Attacker gets a signature that real site rejects
  Result: Attack fails
```

The cryptographic binding:

$$\text{Signature} = \text{Sign}(K_{private}, H(\text{origin} \| \text{challenge} \| \text{client\_data}))$$

Where origin includes the full domain. A phishing site's origin will never match, so the signature is useless to the attacker.

### MFA Fatigue Attack and Countermeasures

```
Attack flow:
  1. Attacker obtains valid username + password (phishing, breach)
  2. Attacker triggers login repeatedly (automated)
  3. User receives dozens of push notifications
  4. User approves one to make them stop --> Account compromised

Countermeasures:
  - Number matching: User must type a number shown on login screen
    (Attacker cannot see the screen --> cannot tell user the number)
  - Rate limiting: Block after 3 denied push attempts
  - Anomaly detection: Alert on unusual push volume
  - Phishing-resistant MFA: FIDO2 (no push to approve)
  - Risk-based auth: Step-up only when risk score warrants it
```

---

## 4. PAM Just-In-Time Access Theory

### The Problem with Standing Privileges

Standing (permanent) privileged access creates a large attack surface:

$$\text{Risk} = P(\text{compromise}) \times \text{Impact} \times \text{Duration of exposure}$$

Standing access means Duration = always, maximizing risk.

### JIT Access Model

```
Traditional (Standing Access):
  Admin account --> Always has root/admin --> Always at risk
  Duration: 24/7/365
  Attack surface: Maximum

JIT (Just-In-Time):
  User --> Request access --> Approval --> Grant for N hours --> Auto-revoke
  Duration: Only when needed (hours, not years)
  Attack surface: Minimized

JIT reduces risk by:
  Risk_JIT / Risk_standing = T_jit / T_total
  If admin needs 2 hours/week of access:
  Risk_reduction = 2 / (7 * 24) = 1.2% of standing risk
  = 98.8% risk reduction
```

### JIT Implementation Patterns

| Pattern | Description | Example |
|---------|-------------|---------|
| Elevation on request | User requests, approver grants, auto-revoke | CyberArk JIT, Azure PIM |
| Ephemeral credentials | Short-lived credentials generated per session | HashiCorp Vault dynamic secrets |
| Ephemeral accounts | Temporary admin account created, destroyed after | Cloud IAM temporary roles |
| Session broker | PAM creates session, user never sees credential | CyberArk PSM, BeyondTrust |

---

## 5. RBAC vs ABAC vs ReBAC

### Role-Based Access Control (RBAC)

```
Model:
  Users --> Roles --> Permissions

  User: jsmith
    Role: "Engineer"
      Permissions: read:code, write:code, read:docs
    Role: "On-Call"
      Permissions: read:production-logs, restart:services

Evaluation:
  Can jsmith read code?
  --> jsmith has role "Engineer"
  --> "Engineer" has permission "read:code"
  --> ALLOW

Strengths:
  - Simple to understand and audit
  - Well-supported by all systems
  - Easy to implement

Weaknesses:
  - Role explosion: N users * M resources * K actions = many roles
  - No context awareness (time, location, device, risk)
  - Static: Cannot express dynamic policies
```

### Attribute-Based Access Control (ABAC)

```
Model:
  Policy = f(Subject attributes, Resource attributes, Action, Environment)

  Subject attributes: department=Engineering, clearance=Secret, location=US
  Resource attributes: classification=Confidential, owner=Engineering
  Action: read
  Environment: time=BusinessHours, device=Managed, network=Corporate

Policy example (XACML-like):
  IF subject.department == resource.owner
  AND subject.clearance >= resource.classification
  AND environment.time IN BusinessHours
  AND environment.device == Managed
  THEN ALLOW

Strengths:
  - Extremely flexible and context-aware
  - No role explosion (attributes compose naturally)
  - Dynamic policies based on real-time context

Weaknesses:
  - Complex to design, implement, and debug
  - Difficult to audit ("why was this allowed?")
  - Requires rich attribute infrastructure
  - Policy conflicts harder to detect
```

### Relationship-Based Access Control (ReBAC)

```
Model:
  Access based on relationships between entities in a graph

  Graph:
    org:acme --member--> user:jsmith
    org:acme --owner--> folder:engineering-docs
    folder:engineering-docs --parent--> doc:design-spec
    doc:design-spec --viewer--> user:jsmith (inherited from org membership)

  Policy:
    user can view doc IF:
      user is viewer of doc
      OR user is editor of doc
      OR user is member of org that owns folder that is parent of doc

  Evaluation (graph traversal):
    Can jsmith view design-spec?
    --> jsmith --member--> acme --owner--> engineering-docs --parent--> design-spec
    --> Relationship path exists --> ALLOW

Strengths:
  - Natural for hierarchical resources (files, folders, orgs)
  - Handles sharing and delegation elegantly
  - Powers Google Drive, GitHub, Notion permissions

Weaknesses:
  - Requires graph database infrastructure
  - Complex relationship resolution (graph traversal)
  - Harder to understand than RBAC for simple cases

Implementations:
  - Google Zanzibar (paper) --> SpiceDB, Ory Keto, Authzed
  - AWS Cedar (combines ABAC + ReBAC)
  - OpenFGA (Auth0/Okta)
```

### Comparison Matrix

| Dimension | RBAC | ABAC | ReBAC |
|-----------|------|------|-------|
| Granularity | Coarse (role-level) | Fine (attribute-level) | Fine (relationship-level) |
| Scalability | Role explosion at scale | Scales with attributes | Scales with graph |
| Context-aware | No | Yes | Partial (relationship context) |
| Audit clarity | High (who has what role) | Low (complex policies) | Medium (trace relationships) |
| Implementation | Simple | Complex | Medium-Complex |
| Standards | NIST RBAC | XACML, ALFA | Zanzibar, Cedar |
| Best for | Enterprise apps, OS/DB | Healthcare, government, ABAC | Document systems, SaaS, social |

---

## 6. Identity Governance Maturity Model

### Maturity Levels

| Level | Name | Characteristics |
|-------|------|----------------|
| 1 | Ad-hoc | Manual provisioning, no reviews, spreadsheet tracking |
| 2 | Defined | Processes documented, periodic reviews, basic reporting |
| 3 | Managed | Automated provisioning (SCIM), regular reviews, SoD checks |
| 4 | Optimized | Risk-based reviews, ML-driven anomaly detection, continuous compliance |
| 5 | Autonomous | Self-healing identities, predictive access, zero standing privileges |

### Maturity Assessment Criteria

| Capability | Level 1 | Level 3 | Level 5 |
|------------|---------|---------|---------|
| Provisioning | Manual tickets | SCIM + HR sync | Self-service + auto-approve based on role |
| Deprovisioning | Manual (days-weeks) | Automated (< 4 hours) | Real-time (< 1 minute) |
| Access reviews | None or annual | Quarterly, manager-driven | Continuous, risk-based, ML-assisted |
| SoD | Not checked | Static rules | Dynamic detection + auto-remediation |
| MFA | Optional | Required for VPN/cloud | Passwordless, continuous, risk-adaptive |
| Privileged access | Shared passwords | PAM vault + recording | JIT + ephemeral + zero standing |
| Visibility | Manual audit logs | SIEM integration | Identity analytics platform |

---

## 7. Zero-Trust Identity Verification

### Continuous Authentication

Traditional authentication is a binary event (login = trusted). Zero trust requires continuous verification:

```
Traditional:
  [Login] --> [Trusted for 8 hours] --> [Logout]

  Problem: If session is hijacked after login, attacker has full access

Zero Trust (Continuous):
  [Login] --> [Verify] --> [Verify] --> [Verify] --> [Logout]
              every        every        every
              request      5 min        risk change

  Signals evaluated continuously:
  - Device health (is AV running? OS patched? disk encrypted?)
  - Network location (corporate, VPN, public Wi-Fi, TOR?)
  - User behavior (normal access pattern? impossible travel?)
  - Application sensitivity (public wiki vs financial system)
  - Time of day (business hours vs 3 AM)
```

### Risk Score Computation

$$\text{Risk} = \sum_{i} w_i \cdot S_i$$

Where $S_i$ are individual signal scores and $w_i$ are learned weights:

| Signal | Low Risk (0) | Medium (0.5) | High Risk (1.0) |
|--------|-------------|--------------|-----------------|
| Device | Managed, compliant | BYOD, compliant | Unknown, non-compliant |
| Network | Corporate | VPN | Public Wi-Fi, TOR |
| Location | Home/office (known) | Domestic travel | Foreign, impossible travel |
| Behavior | Normal pattern | Slightly unusual | Never-before-seen pattern |
| Auth age | < 1 hour | 1-4 hours | > 8 hours |
| App sensitivity | Public | Internal | Restricted/regulated |

Policy action based on aggregate risk:

$$\text{Action} = \begin{cases}
\text{Allow} & \text{if Risk} < 0.3 \\
\text{Step-up MFA} & \text{if } 0.3 \leq \text{Risk} < 0.7 \\
\text{Block + Alert} & \text{if Risk} \geq 0.7
\end{cases}$$

---

## 8. Identity Attack Surface

### Credential Stuffing

```
Attack:
  1. Attacker obtains breached credential database
     (e.g., Collection #1: 773 million emails + passwords)
  2. Automated tool tries each credential pair against target site
  3. Password reuse rate: ~65% of users reuse passwords
  4. Success rate: 0.1-2% of attempts succeed

Defense layers:
  Layer 1: Rate limiting (fail2ban, WAF rules)
  Layer 2: CAPTCHA after N failures
  Layer 3: Breached password detection (HaveIBeenPwned API)
  Layer 4: MFA (even if password is correct, 2nd factor needed)
  Layer 5: Passwordless (no password to stuff)

Detection signals:
  - High volume of failed logins from single IP
  - Many different usernames from same source
  - Login attempts at inhuman speed
  - Geographic anomaly (bulk attempts from unusual region)
```

### Token Theft

```
Techniques:
  1. Session hijacking (XSS to steal cookies)
  2. Token extraction from memory (Mimikatz for Kerberos)
  3. OAuth token theft (redirect URI manipulation)
  4. Refresh token theft (insecure storage)
  5. PRT (Primary Refresh Token) theft (Azure AD device token)

Mitigations:
  - Token binding (bind token to TLS connection or device)
  - Short-lived access tokens (5-15 min expiration)
  - Refresh token rotation (single use)
  - Certificate-based authentication (token bound to cert)
  - Continuous access evaluation (CAE) — revoke in real time
  - Sender-constrained tokens (DPoP, mTLS)
```

### Golden Ticket Attack (Kerberos)

```
Prerequisites:
  - Attacker has compromised the krbtgt account hash
  - krbtgt is the Kerberos TGT encryption key in Active Directory

Attack:
  1. Extract krbtgt hash (requires domain admin or DC compromise)
  2. Forge a TGT with arbitrary:
     - Username (can impersonate any user, even non-existent)
     - Group memberships (Domain Admins, Enterprise Admins)
     - Lifetime (can set to 10 years)
  3. Use forged TGT to request service tickets for any service
  4. Full domain compromise with persistent access

Detection:
  - TGT lifetime exceeding domain policy
  - TGT without corresponding AS-REQ in DC logs
  - Account name in ticket does not exist in AD
  - Event ID 4769 with anomalous encryption type

Remediation:
  - Reset krbtgt password TWICE (old + current hash)
  - Wait for full replication between all DCs
  - All existing TGTs are invalidated (business disruption)
  - Investigate root cause (how was krbtgt compromised?)
```

### Silver Ticket Attack

```
Prerequisites:
  - Attacker has the NTLM hash of a service account

Attack:
  1. Forge a service ticket (no need for TGT or KDC)
  2. Ticket is valid because it is encrypted with the service's key
  3. Access ONLY the specific service (not domain-wide like golden ticket)
  4. Does not contact KDC (harder to detect)

Detection:
  - Service ticket without corresponding TGS-REQ at KDC
  - PAC validation failures (if service validates PAC with KDC)
  - Anomalous service access patterns

Remediation:
  - Reset the compromised service account password
  - Enable PAC validation on services
  - Use managed service accounts (gMSA) with automatic rotation
```

### Pass-the-Hash / Pass-the-Ticket

```
Pass-the-Hash:
  - Extract NTLM hash from memory (Mimikatz: sekurlsa::logonpasswords)
  - Use hash directly for NTLM authentication (no need to crack password)
  - Works because NTLM is a challenge-response using the hash

Pass-the-Ticket:
  - Extract Kerberos tickets from memory (Mimikatz: sekurlsa::tickets)
  - Inject ticket into another session
  - Use stolen TGT to request service tickets as the victim

Defense:
  - Credential Guard (Windows): Isolates hashes in secure enclave
  - Disable NTLM where possible (use Kerberos only)
  - Protected Users security group (disables NTLM, delegation)
  - LAPS (Local Administrator Password Solution): Unique local admin per machine
  - Tiered administration: Separate admin accounts per tier
  - Limit credential caching on endpoints
```

---

## See Also

- cisco-ise, oauth, jwt, pki, ssh, zero-trust, pam, radius, dot1x

## References

- [NIST SP 800-63 — Digital Identity Guidelines](https://csrc.nist.gov/publications/detail/sp/800-63/4/final)
- [NIST SP 800-207 — Zero Trust Architecture](https://csrc.nist.gov/publications/detail/sp/800-207/final)
- [RFC 4120 — The Kerberos Network Authentication Service (V5)](https://www.rfc-editor.org/rfc/rfc4120)
- [OASIS SAML 2.0 Core](http://docs.oasis-open.org/security/saml/v2.0/saml-core-2.0-os.pdf)
- [OpenID Connect Core 1.0](https://openid.net/specs/openid-connect-core-1_0.html)
- [FIDO2 / WebAuthn Level 2](https://www.w3.org/TR/webauthn-2/)
- [RFC 7644 — SCIM Protocol](https://www.rfc-editor.org/rfc/rfc7644)
- [Google Zanzibar Paper](https://research.google/pubs/pub48190/)
- [MITRE ATT&CK — Credential Access](https://attack.mitre.org/tactics/TA0006/)
- [Mimikatz Documentation](https://github.com/gentilkiwi/mimikatz/wiki)
