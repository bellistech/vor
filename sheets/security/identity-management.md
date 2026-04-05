# Identity Management (IAM, SSO, MFA, PAM, Governance)

Managing the full identity lifecycle — provisioning, authentication, authorization, and deprovisioning — across directory services, federation, privileged access, and governance frameworks.

## Identity Lifecycle

### Lifecycle Phases

```
Joiner              Mover                Leaver
(Provisioning)      (Management)         (Deprovisioning)
     |                   |                    |
     v                   v                    v
Create accounts     Transfer roles       Disable accounts
Assign roles        Update groups        Revoke access
Grant access        Recertify access     Archive data
Issue credentials   Rotate credentials   Recover licenses
Enroll MFA          Update attributes    Remove from groups
                                         Delete after retention
```

### Provisioning Checklist

| Step | System | Action |
|------|--------|--------|
| 1 | HR System (Workday, BambooHR) | Employee record created (source of truth) |
| 2 | Identity Provider (Azure AD/Entra ID) | Account provisioned via SCIM or sync |
| 3 | Active Directory | On-prem AD account created (if hybrid) |
| 4 | Email (M365, Google) | Mailbox provisioned |
| 5 | SSO/Federation | SSO profile linked |
| 6 | MFA | Enrollment initiated (TOTP, FIDO2, push) |
| 7 | PAM | Privileged accounts provisioned (if applicable) |
| 8 | Application-specific | SaaS app accounts via SCIM/JIT |

## Directory Services

### Active Directory (AD)

```powershell
# Query AD users
Get-ADUser -Filter * -Properties DisplayName, Department, Enabled |
  Select-Object SamAccountName, DisplayName, Department, Enabled

# Find disabled accounts
Get-ADUser -Filter {Enabled -eq $false} -Properties LastLogonDate |
  Select-Object SamAccountName, LastLogonDate

# Find users not logged in for 90 days
$threshold = (Get-Date).AddDays(-90)
Get-ADUser -Filter {LastLogonDate -lt $threshold -and Enabled -eq $true} \
  -Properties LastLogonDate | Select-Object SamAccountName, LastLogonDate

# List group memberships for a user
Get-ADPrincipalGroupMembership -Identity jsmith |
  Select-Object Name

# Add user to group
Add-ADGroupMember -Identity "VPN-Users" -Members jsmith

# Remove user from group
Remove-ADGroupMember -Identity "VPN-Users" -Members jsmith -Confirm:$false

# Disable an account (deprovisioning)
Disable-ADAccount -Identity jsmith

# Force password reset at next logon
Set-ADUser -Identity jsmith -ChangePasswordAtLogon $true

# Search by attribute
Get-ADUser -Filter {Department -eq "Engineering"} -Properties Department
```

### LDAP (Lightweight Directory Access Protocol)

```bash
# Search for a user
ldapsearch -H ldap://dc01.example.com -D "cn=admin,dc=example,dc=com" \
  -w "$LDAP_PASS" -b "dc=example,dc=com" "(uid=jsmith)"

# Search for all users in an OU
ldapsearch -H ldap://dc01.example.com -D "cn=admin,dc=example,dc=com" \
  -w "$LDAP_PASS" -b "ou=Engineering,dc=example,dc=com" "(objectClass=person)"

# Add a user (LDIF)
ldapadd -H ldap://dc01.example.com -D "cn=admin,dc=example,dc=com" \
  -w "$LDAP_PASS" -f new_user.ldif

# LDIF file format
cat <<'EOF' > new_user.ldif
dn: uid=jsmith,ou=People,dc=example,dc=com
objectClass: inetOrgPerson
objectClass: posixAccount
uid: jsmith
cn: John Smith
sn: Smith
givenName: John
mail: jsmith@example.com
uidNumber: 10001
gidNumber: 10000
homeDirectory: /home/jsmith
loginShell: /bin/bash
userPassword: {SSHA}hashed_password_here
EOF

# Modify a user attribute
ldapmodify -H ldap://dc01.example.com -D "cn=admin,dc=example,dc=com" \
  -w "$LDAP_PASS" <<'EOF'
dn: uid=jsmith,ou=People,dc=example,dc=com
changetype: modify
replace: mail
mail: john.smith@example.com
EOF

# Delete a user
ldapdelete -H ldap://dc01.example.com -D "cn=admin,dc=example,dc=com" \
  -w "$LDAP_PASS" "uid=jsmith,ou=People,dc=example,dc=com"

# Test authentication (bind test)
ldapwhoami -H ldap://dc01.example.com \
  -D "uid=jsmith,ou=People,dc=example,dc=com" -w "$USER_PASS"
```

### Azure AD / Entra ID

```bash
# Azure CLI: List users
az ad user list --query "[].{UPN:userPrincipalName, Display:displayName, Enabled:accountEnabled}"

# Get a specific user
az ad user show --id jsmith@example.com

# Create a user
az ad user create \
  --display-name "John Smith" \
  --user-principal-name jsmith@example.com \
  --password "TempP@ss123!" \
  --force-change-password-next-sign-in true

# List group memberships
az ad user get-member-of --id jsmith@example.com --query "[].displayName"

# Add user to group
az ad group member add --group "Engineering" --member-id <user-object-id>

# Disable a user
az ad user update --id jsmith@example.com --account-enabled false

# List enterprise applications (service principals)
az ad sp list --all --query "[].{Name:displayName, AppId:appId}" --output table

# Microsoft Graph API: List users
curl -H "Authorization: Bearer $TOKEN" \
  "https://graph.microsoft.com/v1.0/users?\$select=displayName,userPrincipalName,accountEnabled"

# Microsoft Graph API: Get sign-in logs
curl -H "Authorization: Bearer $TOKEN" \
  "https://graph.microsoft.com/v1.0/auditLogs/signIns?\$filter=userPrincipalName eq 'jsmith@example.com'"
```

## Single Sign-On (SSO)

### Protocol Comparison

| Feature | SAML 2.0 | OIDC | Kerberos |
|---------|----------|------|----------|
| Token format | XML assertion | JWT (JSON) | Kerberos ticket (binary) |
| Transport | HTTP POST/Redirect | HTTP REST | UDP/TCP port 88 |
| Use case | Enterprise SSO, SaaS | Web/mobile apps, APIs | On-prem Windows/AD |
| IdP examples | ADFS, Okta, Ping | Azure AD, Auth0, Keycloak | Active Directory KDC |
| Discovery | Metadata XML | .well-known/openid-configuration | DNS SRV records |
| Logout | Single Logout (SLO) | RP-Initiated Logout | Ticket expiration |
| Mobile friendly | Poor (XML parsing) | Good (JSON/JWT) | Poor (domain-bound) |

### SAML 2.0 Flow

```
User           Service Provider (SP)         Identity Provider (IdP)
  |                    |                              |
  |-- Access app ----->|                              |
  |                    |                              |
  |                    |-- SAML AuthnRequest -------->|
  |                    |   (HTTP Redirect or POST)    |
  |                    |                              |
  |<----- Login page (if not authenticated) ----------|
  |------ Credentials -------------------------------->|
  |                    |                              |
  |                    |<-- SAML Response ------------|
  |                    |    (Signed XML assertion)    |
  |                    |    Contains:                 |
  |                    |    - NameID (user identity)  |
  |                    |    - Attributes (groups, etc)|
  |                    |    - Conditions (time, aud.) |
  |                    |    - AuthnStatement          |
  |                    |                              |
  |                    | [Validate signature]         |
  |                    | [Check conditions]           |
  |                    | [Create session]             |
  |                    |                              |
  |<-- App session ----|                              |
```

### OIDC (OpenID Connect) Flow

```
User        Client App (RP)          Authorization Server (IdP)
  |              |                            |
  |-- Login ---->|                            |
  |              |-- /authorize ------------->|
  |              |   ?response_type=code      |
  |              |   &client_id=xxx           |
  |              |   &redirect_uri=xxx        |
  |              |   &scope=openid profile    |
  |              |   &state=random            |
  |              |                            |
  |<-- Login page ----------------------------|
  |-- Credentials ---------------------------->|
  |              |                            |
  |<-- Redirect with code --------------------|
  |-- Follow redirect -->|                    |
  |              |                            |
  |              |-- POST /token ------------>|
  |              |   grant_type=authz_code    |
  |              |   code=xxx                 |
  |              |   client_secret=xxx        |
  |              |                            |
  |              |<-- Token Response ---------|
  |              |   {                        |
  |              |     "access_token": "...", |
  |              |     "id_token": "...",     |
  |              |     "refresh_token": "..." |
  |              |   }                        |
  |              |                            |
  |              |-- GET /userinfo ---------->|
  |              |   Authorization: Bearer    |
  |              |<-- User claims ------------|
  |              |                            |
  |<-- Session --|                            |
```

### Kerberos Authentication

```
Client              KDC (Key Distribution Center)       Service
  |                    AS          TGS                      |
  |                    |           |                        |
  |-- AS-REQ --------->|           |                        |
  |   (username,       |           |                        |
  |    timestamp       |           |                        |
  |    encrypted w/    |           |                        |
  |    user's key)     |           |                        |
  |                    |           |                        |
  |<-- AS-REP ---------|           |                        |
  |   (TGT encrypted   |           |                        |
  |    w/ krbtgt key)  |           |                        |
  |                    |           |                        |
  |-- TGS-REQ ---------|---------->|                        |
  |   (TGT +           |           |                        |
  |    service SPN)    |           |                        |
  |                    |           |                        |
  |<-- TGS-REP --------|-----------|                        |
  |   (Service ticket  |           |                        |
  |    encrypted w/    |           |                        |
  |    service key)    |           |                        |
  |                    |           |                        |
  |-- AP-REQ ------------------------------------------>|
  |   (Service ticket + authenticator)                  |
  |                                                     |
  |<-- AP-REP ------------------------------------------|
  |   (Mutual auth, session key)                        |
```

## Multi-Factor Authentication (MFA)

### MFA Factor Categories

| Category | Factor | Examples |
|----------|--------|----------|
| Something you know | Password, PIN | AD password, smart card PIN |
| Something you have | Token, phone, key | TOTP app, FIDO2 key, smart card |
| Something you are | Biometric | Fingerprint, face, iris, voice |
| Somewhere you are | Location | IP geolocation, GPS, network |
| Something you do | Behavior | Typing pattern, gait, usage pattern |

### MFA Method Comparison

| Method | Phishing Resistant | User Experience | Cost | Security Level |
|--------|-------------------|-----------------|------|---------------|
| SMS OTP | No (SIM swap, SS7) | Medium | Low | Low |
| TOTP (Google Auth, Authy) | No (real-time phish) | Medium | Free | Medium |
| Push notification (Duo) | Partial (MFA fatigue) | Good | Medium | Medium-High |
| FIDO2 / WebAuthn | Yes (origin-bound) | Good | Medium (keys) | Very High |
| Smart card / PIV | Yes (certificate-bound) | Poor (reader needed) | High | Very High |
| Certificate-based (CBA) | Yes | Transparent | Medium | High |

### Cisco Duo Integration

```bash
# Duo Auth API: Verify user with push
curl -X POST "https://api-XXXXXXXX.duosecurity.com/auth/v2/auth" \
  -d "username=jsmith" \
  -d "factor=push" \
  -d "device=auto" \
  -u "$INTEGRATION_KEY:$SECRET_KEY"

# Duo Auth API: Verify passcode
curl -X POST "https://api-XXXXXXXX.duosecurity.com/auth/v2/auth" \
  -d "username=jsmith" \
  -d "factor=passcode" \
  -d "passcode=123456" \
  -u "$INTEGRATION_KEY:$SECRET_KEY"

# Duo Admin API: List users
curl "https://api-XXXXXXXX.duosecurity.com/admin/v1/users" \
  -u "$INTEGRATION_KEY:$SECRET_KEY"

# Duo Admin API: Get authentication logs
curl "https://api-XXXXXXXX.duosecurity.com/admin/v2/logs/authentication" \
  -d "mintime=$(date -d '24 hours ago' +%s)" \
  -u "$INTEGRATION_KEY:$SECRET_KEY"

# Duo Admin API: Enroll a user
curl -X POST "https://api-XXXXXXXX.duosecurity.com/admin/v1/users" \
  -d "username=jsmith" \
  -d "email=jsmith@example.com" \
  -u "$INTEGRATION_KEY:$SECRET_KEY"
```

### FIDO2 / WebAuthn Registration and Authentication

```
Registration:
  1. Server generates challenge (random bytes)
  2. Browser calls navigator.credentials.create()
  3. Authenticator generates key pair (private stays on device)
  4. Authenticator signs challenge with private key
  5. Browser returns: public key + signed challenge + attestation
  6. Server stores public key + credential ID for user

Authentication:
  1. Server generates challenge + sends allowed credential IDs
  2. Browser calls navigator.credentials.get()
  3. Authenticator signs challenge with private key
  4. Browser returns: credential ID + signed challenge + user handle
  5. Server verifies signature with stored public key
  6. Origin is bound into the signature (phishing protection)
```

## Privileged Access Management (PAM)

### PAM Architecture

```
+--------------------+
|   PAM Vault        |  Stores privileged credentials
|   (CyberArk,       |  (passwords, SSH keys, API keys, certs)
|    BeyondTrust,    |
|    Delinea)        |
+--------------------+
         |
         v
+--------------------+
|   Session Manager  |  Proxies privileged sessions
|   (Jump server,    |  Records video/keystrokes
|    PSM)            |  Enforces workflow approvals
+--------------------+
         |
    +----+----+
    |         |
    v         v
[SSH/RDP]  [Database]  [Cloud Console]  [Network Device]
```

### PAM Workflow

```
1. User requests access via PAM portal
2. PAM checks:
   - User identity (MFA required)
   - Role-based entitlement
   - Approval workflow (if required)
   - Time window (business hours only?)
   - Reason/ticket number
3. PAM checks out credential from vault
4. PAM proxies session through session manager
5. Session is recorded (video, commands, queries)
6. Credential is rotated after session ends (or on schedule)
7. Session recording stored for audit
```

### Common PAM Policies

| Policy | Description |
|--------|-------------|
| Credential rotation | Auto-rotate passwords every 24-72 hours |
| Session recording | Record all privileged SSH/RDP/DB sessions |
| Dual control | Require approval from second person |
| Time-limited access | Auto-revoke after N hours |
| Break-glass | Emergency access with post-use review |
| Command filtering | Block dangerous commands (rm -rf, DROP TABLE) |

## Identity Governance

### Access Review (Certification) Process

```
Access Certification Campaign:

1. Define scope
   - Which users? (All, department, role)
   - Which access? (All apps, sensitive apps, privileged)
   - Which reviewers? (Managers, app owners, both)

2. Generate review items
   User A --> App 1 (Role: Admin)     --> Reviewer: Manager
   User A --> App 2 (Role: ReadOnly)  --> Reviewer: App Owner
   User B --> App 1 (Role: User)      --> Reviewer: Manager
   ...

3. Reviewer actions
   - Approve (keep access)
   - Revoke (remove access)
   - Modify (change role/permissions)
   - Delegate (send to another reviewer)

4. Enforce decisions
   - Revocations auto-executed via SCIM/API
   - Audit trail of all decisions
   - Escalation for non-response
```

### Role Mining

```
Bottom-up role mining (from existing access):

User A: App1(Admin), App2(Read), App3(Write), VPN
User B: App1(Admin), App2(Read), App3(Write), VPN
User C: App1(Admin), App2(Read), App3(Write), VPN, PAM
User D: App4(Admin), App5(Read), VPN

Discovered roles:
  "Engineering Base" = App1(Admin) + App2(Read) + App3(Write) + VPN
    --> Assigned to Users A, B, C
  "Engineering Lead" = Engineering Base + PAM
    --> Assigned to User C
  "Marketing Base" = App4(Admin) + App5(Read) + VPN
    --> Assigned to User D
```

### Separation of Duties (SoD) Matrix

| Role A | Role B | Conflict |
|--------|--------|----------|
| Accounts Payable | Vendor Management | Fraud risk (create fake vendor + pay) |
| Developer | Production Admin | Code promotion without review |
| User Admin | Audit Admin | Create user + hide activity |
| Purchasing | Receiving | Theft (order goods + confirm receipt) |
| HR Admin | Payroll Admin | Ghost employee fraud |

## SCIM Provisioning

### SCIM Protocol

```bash
# SCIM 2.0: Create a user
curl -X POST "https://app.example.com/scim/v2/Users" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/scim+json" \
  -d '{
    "schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
    "userName": "jsmith@example.com",
    "name": {
      "givenName": "John",
      "familyName": "Smith"
    },
    "emails": [{"value": "jsmith@example.com", "primary": true}],
    "active": true
  }'

# SCIM 2.0: Get a user
curl "https://app.example.com/scim/v2/Users/user-id-123" \
  -H "Authorization: Bearer $TOKEN"

# SCIM 2.0: Update a user (PATCH)
curl -X PATCH "https://app.example.com/scim/v2/Users/user-id-123" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/scim+json" \
  -d '{
    "schemas": ["urn:ietf:params:scim:api:messages:2.0:PatchOp"],
    "Operations": [
      {"op": "replace", "path": "active", "value": false}
    ]
  }'

# SCIM 2.0: List users with filter
curl "https://app.example.com/scim/v2/Users?filter=userName%20eq%20%22jsmith%40example.com%22" \
  -H "Authorization: Bearer $TOKEN"

# SCIM 2.0: Delete a user
curl -X DELETE "https://app.example.com/scim/v2/Users/user-id-123" \
  -H "Authorization: Bearer $TOKEN"

# SCIM 2.0: Create a group
curl -X POST "https://app.example.com/scim/v2/Groups" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/scim+json" \
  -d '{
    "schemas": ["urn:ietf:params:scim:schemas:core:2.0:Group"],
    "displayName": "Engineering",
    "members": [
      {"value": "user-id-123"},
      {"value": "user-id-456"}
    ]
  }'
```

## Just-In-Time (JIT) Provisioning

### JIT via SAML

```
IdP sends SAML assertion with user attributes:
  <saml:Assertion>
    <saml:AttributeStatement>
      <saml:Attribute Name="email">jsmith@example.com</saml:Attribute>
      <saml:Attribute Name="firstName">John</saml:Attribute>
      <saml:Attribute Name="lastName">Smith</saml:Attribute>
      <saml:Attribute Name="groups">Engineering,VPN-Users</saml:Attribute>
    </saml:AttributeStatement>
  </saml:Assertion>

SP receives assertion --> User does not exist locally
  --> Auto-create account from assertion attributes
  --> Assign roles based on group attributes
  --> User is logged in immediately

No pre-provisioning needed. Account created on first access.
```

### JIT vs SCIM vs Manual

| Feature | JIT | SCIM | Manual |
|---------|-----|------|--------|
| Account creation | On first login | Before first login | Before first login |
| Deprovisioning | No (must handle separately) | Yes (sync deletes) | Yes (admin action) |
| Attribute sync | At each login | Continuous/scheduled | Manual |
| Pre-login access setup | No | Yes | Yes |
| Complexity | Low | Medium | High |

## Identity for IoT / OT

### IoT Identity Challenges

| Challenge | Solution |
|-----------|----------|
| No keyboard/screen for credentials | Certificate-based auth (802.1X EAP-TLS) |
| Limited compute for crypto | Lightweight crypto (ECC, ChaCha20) |
| Thousands of devices | Automated provisioning (EST, SCEP) |
| Long lifecycle (10+ years) | Certificate renewal automation |
| Diverse protocols (MQTT, CoAP) | Protocol-specific identity binding |
| Supply chain trust | Hardware root of trust (TPM, secure element) |

### Device Identity Protocols

| Protocol | Purpose | Credential Type |
|----------|---------|----------------|
| 802.1X EAP-TLS | Network access control | X.509 certificate |
| EST (Enrollment over Secure Transport) | Certificate enrollment | CSR + existing cert or HTTP auth |
| SCEP | Certificate enrollment (legacy) | Challenge password + CSR |
| DPP (Device Provisioning Protocol) | Wi-Fi onboarding | QR code + public key |
| BRSKI (RFC 8995) | Zero-touch bootstrap | Manufacturer cert (IDevID) |

## Identity Analytics

### Key Identity Metrics

| Metric | Target | Alert Threshold |
|--------|--------|----------------|
| Stale accounts (no login > 90 days) | < 5% of total | > 10% |
| Orphaned accounts (no owner) | 0 | > 0 |
| Over-privileged users | < 10% | > 20% |
| MFA enrollment | > 99% | < 95% |
| Failed auth rate | < 5% | > 15% (credential stuffing) |
| Avg time to deprovision | < 4 hours | > 24 hours |
| Access review completion | > 95% | < 80% |
| SoD violations | 0 critical | > 0 critical |

### Suspicious Identity Activity Patterns

| Pattern | Indicator | Response |
|---------|-----------|----------|
| Impossible travel | Login from NYC then London in 30 min | Block + verify |
| Credential stuffing | Many failed logins, different users, same IP | Block IP + rate limit |
| MFA fatigue | 10+ push notifications declined then accepted | Revoke session + investigate |
| Token theft | Session used from new device/IP without re-auth | Force re-authentication |
| Privilege escalation | User added to admin group outside process | Alert + auto-revert |
| Service account abuse | Interactive login on service account | Alert + disable |

## Tips

- Always use phishing-resistant MFA (FIDO2/WebAuthn) for privileged accounts and high-value targets.
- Implement JIT access for privileged roles — grant access only when needed, auto-revoke after timeout.
- Run access reviews quarterly for sensitive applications, annually for everything else.
- Integrate HR system as the authoritative source of truth for identity lifecycle — joiners/movers/leavers should trigger automated provisioning/deprovisioning.
- Deploy SCIM for SaaS provisioning to ensure deprovisioning happens automatically when accounts are disabled in the IdP.
- Monitor for MFA fatigue attacks — implement number matching (Duo Verified Push) instead of simple approve/deny.
- Enforce conditional access policies that combine identity signals: user, device, location, risk level, application sensitivity.
- Never use SMS as the only second factor — it is vulnerable to SIM swapping and SS7 interception.
- Separate admin accounts from daily-use accounts — admins should have two identities with different MFA policies.
- Use certificate-based authentication for IoT/OT devices that cannot support interactive MFA.

## See Also

- cisco-ise, oauth, jwt, pki, ssh, zero-trust, pam, radius, dot1x

## References

- [NIST SP 800-63B — Digital Identity Guidelines: Authentication](https://csrc.nist.gov/publications/detail/sp/800-63b/final)
- [NIST SP 800-63A — Enrollment and Identity Proofing](https://csrc.nist.gov/publications/detail/sp/800-63a/final)
- [FIDO2 / WebAuthn Specification](https://www.w3.org/TR/webauthn-2/)
- [RFC 7644 — SCIM Protocol](https://www.rfc-editor.org/rfc/rfc7644)
- [RFC 7643 — SCIM Core Schema](https://www.rfc-editor.org/rfc/rfc7643)
- [OASIS SAML 2.0 Specification](http://docs.oasis-open.org/security/saml/Post2.0/sstc-saml-tech-overview-2.0.html)
- [OpenID Connect Core 1.0](https://openid.net/specs/openid-connect-core-1_0.html)
- [RFC 4120 — The Kerberos Network Authentication Service (V5)](https://www.rfc-editor.org/rfc/rfc4120)
- [Cisco Duo Documentation](https://duo.com/docs)
- [Microsoft Entra ID Documentation](https://learn.microsoft.com/en-us/entra/identity/)
