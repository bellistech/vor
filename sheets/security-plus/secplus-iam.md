# Identity & Access Management (Security+ SY0-701 Objective 4.6)

> Provisioning to PAM: federation and SSO protocols (SAML vs OAuth vs OIDC vs Kerberos), the five access-control models with their exam keywords, MFA factor math, and NIST's modern password rules — 4.6 is the densest single objective on the test.

## Account Lifecycle (4.6)

```
# Provisioning — creating identities + granting entitlements when
#   someone joins/changes role. Tie to HR events; grant via groups/roles
#   (never one-off permissions); document approvals.
# De-provisioning — removing access at departure/role change. The
#   classic failure: orphaned accounts of ex-employees — prime insider/
#   takeover targets. Automate on HR termination events; disable first,
#   delete after retention.
# Permission assignments and implications — every grant is attack
#   surface. Permission creep: users accumulate rights as they change
#   roles and nobody removes the old ones. Fix: role-based assignment +
#   periodic access reviews/recertification.
# Identity proofing — verifying a human is who they claim BEFORE
#   issuing credentials (documents, knowledge checks, live selfie match).
#   Also applies at help-desk password resets — weak proofing there is
#   how attackers "recover" other people's accounts.
# Attestation — periodic formal review where owners/managers certify
#   "these people still need this access" — audit evidence, kills creep.
# Interoperability — identity systems working across platforms/orgs via
#   standards: SAML, OIDC, SCIM (System for Cross-domain Identity
#   Management — automated provisioning between IdP and apps).
```

## Federation & SSO (4.6)

```
# SSO (Single Sign-On) — authenticate once, access many apps.
# Federation — SSO stretched across ORGANIZATIONS/domains: your IdP
#   (Identity Provider) vouches for you to someone else's SP (Service
#   Provider). "Log in with your corporate account to a partner SaaS."
#
# Trust flow: user → SP → redirect to IdP → authenticate → signed
#   assertion/token back to SP → access. The SP never sees the password.
```

| Protocol | What it's for | Token format | Exam cue |
|---|---|---|---|
| LDAP (Lightweight Directory Access Protocol) | Directory lookup + auth against on-prem directory (AD) | n/a (bind) | "Query the directory", port 389/636 (LDAPS) |
| Kerberos | On-prem Windows/AD authentication | Tickets (TGT from KDC) | "Tickets", "KDC", "time-sensitive", no passwords on wire |
| SAML (Security Assertion Markup Language) | Web SSO/federation, enterprise SaaS | XML assertion | "XML", "IdP/SP", browser SSO to SaaS |
| OAuth (Open Authorization) 2.0 | AUTHORIZATION — delegated access to APIs/resources | Access token | "App accesses your data without your password", scopes |
| OIDC (OpenID Connect) | AUTHENTICATION layered on OAuth | JWT ID token | "Login with Google", mobile/modern web login |
| RADIUS | Network access AAA (VPN, Wi-Fi, 802.1X) | n/a | UDP 1812/1813, encrypts only the password |
| TACACS+ (Terminal Access Controller Access-Control System Plus) | Device administration AAA (routers/switches) | n/a | TCP 49, encrypts whole payload, per-command authorization |

```
# The three guaranteed distinctions:
# SAML vs OIDC   — both do federated web login; SAML = XML, enterprise
#                  legacy SaaS; OIDC = JSON/JWT (JSON Web Token), modern
#                  and mobile-friendly.
# OAuth vs OIDC  — OAuth AUTHORIZES (what you may access); OIDC
#                  AUTHENTICATES (who you are). "Photos app may read
#                  your cloud drive" = OAuth. "Sign in with…" = OIDC.
# RADIUS vs TACACS+ — RADIUS: network ACCESS, UDP, combined authn/authz,
#                  encrypts password only. TACACS+: device ADMIN, TCP 49,
#                  separates AAA, encrypts everything, per-command rules.
```

## Access Control Models (4.6)

| Model | Who decides | Exam keyword |
|---|---|---|
| MAC (Mandatory Access Control) | The SYSTEM, via labels + clearances | "Classification labels", "clearance", military/SELinux |
| DAC (Discretionary Access Control) | The resource OWNER | "Owner grants access", NTFS/Unix file permissions |
| RBAC (Role-Based Access Control) | Admins define roles by JOB FUNCTION | "Based on job role/department" |
| Rule-based | System-wide RULES applied to everyone | "Firewall ACL", "if condition then allow", applies regardless of identity |
| ABAC (Attribute-Based Access Control) | Policy engine evaluating ATTRIBUTES | "User dept + device health + location + time all evaluated" |

```
# Also in scope:
# Time-of-day restrictions — access only during defined hours (a simple
#   rule-based control): contractors 9–5, no logins at 3 a.m.
# Least privilege — the minimum access needed to do the job, nothing
#   more; the principle behind every model above and the answer to many
#   "BEST reduce risk" questions.
#
# Disambiguation traps:
# RBAC vs ABAC — role only? RBAC. Multiple contextual attributes
#   (device, location, time, sensitivity)? ABAC. ABAC is the engine
#   behind Zero Trust policy decisions (zero-trust sheet).
# MAC vs DAC — can the file's owner share it? DAC. Does a label/
#   clearance system forbid it regardless of the owner's wishes? MAC.
# Rule-based vs role-based — both abbreviate RBAC on sloppy quizzes;
#   SY0-701 spells them out. Rules bind to CONDITIONS, roles bind to
#   PEOPLE'S JOBS.
```

## Multifactor Authentication (4.6)

```
# Factors:
#   Something you know  — password, PIN, security questions
#   Something you have  — phone (authenticator app), smart card,
#                         hardware token, security key
#   Something you are   — biometrics: fingerprint, face, iris, voice
#   Somewhere you are   — location: GPS, network/IP, geofencing
#
# MFA = factors from DIFFERENT categories.
#   password + PIN            = ONE factor (both "know") — not MFA
#   password + authenticator  = two factors (know + have)
#   fingerprint + smart card  = two factors (are + have)
#
# Implementations:
#   Biometrics — measure FAR (False Acceptance Rate) vs FRR (False
#     Rejection Rate); crossover error rate (CER) compares systems.
#   Hard authentication token — dedicated device (key fob showing
#     codes, smart card); soft token — app-generated codes (TOTP —
#     Time-based One-Time Password, 30-second windows; HOTP —
#     HMAC-based, counter-driven).
#   Security keys — FIDO2/U2F USB/NFC devices; phishing-resistant
#     because the key cryptographically binds to the real site origin.
#
# Phishing resistance ranking (best → worst):
#   FIDO2 security key / passkey  >  authenticator app TOTP
#   >  push notification (MFA-fatigue attacks!)  >  SMS codes
#   (SIM-swap risk — NIST discourages SMS for high assurance)
```

## Password Concepts (4.6)

```
# Modern best practice (NIST SP 800-63B — the exam follows it):
#   Length — LENGTH BEATS COMPLEXITY. Long passphrases > "P@ssw0rd1!".
#   Complexity — composition rules (must have symbol/number) are no
#     longer recommended; they produce predictable substitutions.
#   Reuse — ban reuse across systems; check against breach corpora.
#   Expiration/age — routine forced rotation is OUT (it breeds
#     Password1 → Password2); rotate on evidence of compromise.
#   Screen new passwords against known-breached lists.
# Password managers — vault generating/storing unique high-entropy
#   passwords per site; one strong master secret + MFA protects all.
# Passwordless — FIDO2/WebAuthn passkeys: private key on device,
#   unlocked by biometric/PIN; nothing phishable transits the wire.
#   Exam cue: "eliminate passwords entirely" → passwordless/passkeys.
# Storage side (defender view): hash with salt + key stretching
#   (bcrypt/PBKDF2 — Password-Based Key Derivation Function 2)
#   → secplus-cryptography.
```

## Privileged Access Management (4.6)

```
# PAM (Privileged Access Management) — controlling admin/root/service
#   accounts, the highest-value targets in the org.
#
# Just-in-time (JIT) permissions — privileges granted ON REQUEST for a
#   TIME WINDOW, then auto-revoked. No standing admin rights; approval
#   + logging built in. Exam cue: "admin rights only when needed".
# Password vaulting — privileged creds live in a hardened vault;
#   admins check them out (logged, approved), vault rotates the
#   password after use. Nobody "knows" the root password anymore.
# Ephemeral credentials — short-lived, single-purpose creds/certs
#   minted on demand and expiring in minutes — nothing durable to
#   steal. Exam cue: "temporary credentials that expire automatically".
#
# Supporting practice: separate everyday account vs admin account for
# the same human; session recording on privileged access; break-glass
# accounts sealed for emergencies.
```

## Exam Cues (4.6)

```
# "ex-employee account still active"          → de-provisioning failure
# "rights accumulate over years"              → permission creep → access review
# "managers certify access quarterly"         → attestation
# "XML-based SSO to SaaS"                     → SAML
# "app reads your calendar, no password"      → OAuth
# "Sign in with Google"                       → OIDC
# "tickets, KDC, time skew breaks logins"     → Kerberos
# "per-command authorization on routers"      → TACACS+
# "Wi-Fi auth against a central server"       → RADIUS (802.1X)
# "labels and clearances"                     → MAC
# "file owner shares the file"                → DAC
# "access by department/job"                  → RBAC (role-based)
# "location + device posture + time"          → ABAC
# "password plus PIN — is it MFA?"            → no (same factor)
# "phishing-resistant MFA"                    → FIDO2 security key/passkey
# "should passwords expire every 60 days?"    → no — rotate on compromise (NIST)
# "admin access for 2 hours with approval"    → just-in-time PAM
# "credentials that expire in minutes"        → ephemeral credentials
```

## See Also

security-plus-sy0-701, secplus-security-concepts, secplus-architecture, secplus-monitoring-defense, secplus-cryptography, secplus-hardening, identity-management, access-control-models, oauth, saml, oidc, ldap, kerberos, zero-trust, vault, owasp-auth

## References

- [CompTIA Security+ SY0-701 Exam Objectives](https://www.comptia.org/certifications/security)
- [NIST SP 800-63B — Digital Identity Guidelines: Authentication and Lifecycle Management](https://pages.nist.gov/800-63-3/sp800-63b.html)
- [RFC 6749 — The OAuth 2.0 Authorization Framework](https://www.rfc-editor.org/rfc/rfc6749)
- [OpenID Connect Core 1.0](https://openid.net/specs/openid-connect-core-1_0.html)
- [OASIS SAML 2.0 Technical Overview](https://docs.oasis-open.org/security/saml/Post2.0/sstc-saml-tech-overview-2.0.html)
- [RFC 4120 — The Kerberos Network Authentication Service (V5)](https://www.rfc-editor.org/rfc/rfc4120)
- [FIDO Alliance — FIDO2/WebAuthn](https://fidoalliance.org/fido2/)
- [RFC 2865 — RADIUS](https://www.rfc-editor.org/rfc/rfc2865)
