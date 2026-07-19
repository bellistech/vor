# CompTIA Security+ SY0-701 (Exam Guide — Security+ SY0-701 Objectives 1.1–5.6)

> Blueprint, logistics, domain map, question-decoding strategy, and the high-yield acronym list for the Security+ SY0-701 exam — the anchor sheet for the `security-plus/` category; every objective has a dedicated deep-dive sheet.

## Exam Logistics

```
# Format
# Maximum 90 questions — a mix of:
#   Multiple choice           — single answer or multiple response
#   Performance-based (PBQ)   — simulations: drag-and-drop matching,
#     firewall rule ordering, log analysis, matching attacks to
#     controls; usually 3–5 PBQs and they appear FIRST
# 90 minutes total — the PBQs eat time; budget them (see strategy below)
# No penalty for guessing — never leave a question blank
# Flag-and-review supported; unanswered = wrong

# Scoring
# Scaled score 100–900, pass = 750
# Compensatory model — pass overall, no per-domain minimum
# "Maximum 90" because CompTIA mixes question types; you may see fewer

# Delivery and cost
# Pearson VUE test center or online proctored (OnVUE)
# Exam code SY0-701 (launched Nov 2023; SY0-601 retired Jul 2024)
# Voucher ≈ 404 USD list (bundles/academic discounts common)

# Recommended experience (not required — no formal prerequisites)
# CompTIA Network+ plus 2 years of experience in IT administration
#   with a security focus

# Certification lifecycle
# Valid 3 years from pass date
# Renew via the Continuing Education (CE) program: 50 CEUs over the
#   3-year cycle (training, higher certs like CySA+/PenTest+/CASP+
#   auto-renew it, CertMaster CE course), OR retake the current exam
# Security+ is ISO/ANSI 17024 accredited and on the DoD 8140/8570
#   approved list (IAT Level II) — the reason it is an HR checkbox
```

## Content Domains and Weights

```
# Domain 1: General Security Concepts                          12%
#   Security controls, fundamental concepts (CIA, AAA, Zero Trust,
#   physical security, deception tech), change management,
#   cryptographic solutions
# Domain 2: Threats, Vulnerabilities, and Mitigations          22%
#   Threat actors and motivations, attack surfaces and vectors,
#   vulnerability types, indicators of malicious activity,
#   mitigation techniques
# Domain 3: Security Architecture                              18%
#   Architecture models (cloud, IaC, serverless, ICS/SCADA, IoT),
#   securing enterprise infrastructure, data protection strategies,
#   resilience and recovery
# Domain 4: Security Operations                                28%
#   Hardening, asset management, vulnerability management,
#   monitoring/alerting, enterprise security capabilities, IAM,
#   automation, incident response, log data sources
# Domain 5: Security Program Management and Oversight          20%
#   Governance, risk management, third-party risk, compliance,
#   audits and assessments, security awareness
#
# Weight math (~90 questions):
# D1 ≈ 11 q, D2 ≈ 20 q, D3 ≈ 16 q, D4 ≈ 25 q, D5 ≈ 18 q
# Domain 4 (Security Operations) is the biggest slice — weight your
# study time accordingly; D2 + D4 = half the exam
```

## Objective → Sheet Map

```
# Domain 1 — General Security Concepts (12%)
#   1.1 Compare security control categories/types     → cs secplus-security-concepts
#   1.2 Fundamental concepts (CIA, AAA, Zero Trust,
#       physical security, deception/disruption)      → cs secplus-security-concepts
#   1.3 Change management processes and impact        → cs secplus-security-concepts
#   1.4 Cryptographic solutions (PKI, encryption,
#       hashing, certificates, blockchain)            → cs secplus-cryptography
#
# Domain 2 — Threats, Vulnerabilities, Mitigations (22%)
#   2.1 Threat actors and motivations                 → cs secplus-threat-landscape
#   2.2 Threat vectors and attack surfaces            → cs secplus-threat-landscape
#   2.3 Vulnerability types                           → cs secplus-vulnerabilities
#   2.4 Indicators of malicious activity              → cs secplus-attack-indicators
#   2.5 Mitigation techniques                         → cs secplus-hardening
#
# Domain 3 — Security Architecture (18%)
#   3.1 Architecture/infrastructure concepts          → cs secplus-architecture
#   3.2 Apply security principles to infrastructure   → cs secplus-architecture
#   3.3 Data protection concepts and strategies       → cs secplus-data-resilience
#   3.4 Resilience and recovery                       → cs secplus-data-resilience
#
# Domain 4 — Security Operations (28%)
#   4.1 Apply common security techniques (hardening)  → cs secplus-hardening
#   4.2 Asset management (acquisition→disposal)       → cs secplus-asset-vuln-management
#   4.3 Vulnerability management                      → cs secplus-asset-vuln-management
#   4.4 Alerting and monitoring tools/concepts        → cs secplus-monitoring-defense
#   4.5 Enterprise capabilities to enhance security   → cs secplus-monitoring-defense
#   4.6 Identity and access management                → cs secplus-iam
#   4.7 Automation and orchestration                  → cs secplus-incident-response
#   4.8 Incident response activities                  → cs secplus-incident-response
#   4.9 Use data sources to support investigations    → cs secplus-incident-response
#
# Domain 5 — Security Program Management (20%)
#   5.1 Security governance elements                  → cs secplus-governance-risk
#   5.2 Risk management processes                     → cs secplus-governance-risk
#   5.3 Third-party risk assessment and management    → cs secplus-third-party-compliance
#   5.4 Security compliance                           → cs secplus-third-party-compliance
#   5.5 Audits and assessments                        → cs secplus-third-party-compliance
#   5.6 Security awareness practices                  → cs secplus-security-awareness
```

## Question-Decoding Strategy

```
# 1. Find the qualifier — it selects among multiple "correct" answers:
#    "BEST"        → several options work; pick the most complete/most
#                    directly targeted one (not the broadest)
#    "MOST likely" → probability, not possibility — pick the common
#                    cause, not the exotic one
#    "FIRST"       → order-of-operations question; usually the IR
#                    lifecycle (preparation→detection→analysis→
#                    containment→eradication→recovery→lessons learned)
#                    or change management (approval BEFORE work) or
#                    forensics (order of volatility: RAM before disk)
#    "MOST cost-effective" → cheapest control that still mitigates
#                    (often the managerial/awareness answer, not tech)
#    "LEAST"       → invert your reading; slow down and re-read
# 2. Identify the exam's lens: is this a technical, managerial,
#    operational, or physical question? The answer will match the lens.
#    "Which POLICY..." → managerial answer, never a firewall rule.
# 3. Eliminate factually wrong options first (wrong layer, wrong
#    protocol, expanded acronym doesn't fit the scenario)
# 4. Distractor patterns:
#    - Right family, wrong member (IDS offered where IPS is needed —
#      "detect" vs "prevent/block" wording decides it)
#    - Sound-alike acronyms (DLP vs DRP, MDM vs MOU, EDR vs EDM)
#    - Preventive control offered when the question asks how to DETECT
#    - Over-engineered answer when the scenario says "small business"
#      or "limited budget"
#    - Yesterday's tech (WEP, DES, SHA-1, Telnet) — only correct when
#      the question asks what to REPLACE
# 5. Scenario questions: the last sentence contains the actual ask;
#    read it FIRST, then mine the scenario for the constraint
```

## PBQ (Performance-Based Question) Tactics

```
# PBQs appear at the START of the exam — do not sink 20 minutes there
# Skip strategy: flag every PBQ immediately, do all multiple choice
#   (fast points), return with remaining time — PBQs are NOT worth
#   proportionally more per minute spent
# Known PBQ styles for SY0-701:
#   - Drag attack names onto log/scenario descriptions
#   - Order firewall/ACL rules (remember: first match wins,
#     implicit deny at the bottom, most-specific rules first)
#   - Match ports to protocols (22 SSH, 53 DNS, 80/443 HTTP/S,
#     389/636 LDAP/LDAPS, 3389 RDP, 1812 RADIUS, 49 TACACS+,
#     25/587 SMTP/submission, 143/993 IMAP/S, 161/162 SNMP)
#   - Configure wireless security (choose WPA3-Enterprise, 802.1X,
#     RADIUS server fields)
#   - Match control types/categories to examples
#   - Interpret a vulnerability scan / CVSS output
# Partial credit is generally awarded on PBQs — always attempt
# The simulation "Reset" button restores the initial state if you
#   tangle yourself; answers are saved when you click Next
```

## Keyword → Answer Reflex Table

| Scenario keyword | Reflex answer |
|---|---|
| Unskilled attacker using downloaded tools | Script kiddie / unskilled attacker |
| State-sponsored, long-term, stealthy | Nation-state actor / APT (advanced persistent threat) |
| Employee stealing data, has legit access | Insider threat |
| Attack for political/social cause | Hacktivist |
| Verify integrity of downloaded file | Hash comparison (SHA-256) |
| Prove sender cannot deny sending | Non-repudiation → digital signature |
| Encrypt data, share key securely | Hybrid: asymmetric wraps symmetric key |
| Stop laptop data theft after loss | FDE (full-disk encryption) + TPM |
| Malware that encrypts files for payment | Ransomware |
| Credential reuse from another breach | Credential stuffing (vs spraying: one password, many accounts) |
| Fake site harvesting credentials, urgent email | Phishing (voice=vishing, SMS=smishing) |
| Attack targeting a specific executive | Whaling / spear phishing |
| Compromise a site the victims already visit | Watering-hole attack |
| Rogue interception between two parties | On-path attack (formerly man-in-the-middle) |
| Overwhelm service from many sources | DDoS (distributed denial of service) |
| Injection into database query | SQLi → parameterized queries/prepared statements |
| Script executes in other users' browsers | XSS (cross-site scripting) → output encoding, CSP |
| Forged request using victim's session | CSRF (cross-site request forgery) → anti-CSRF tokens |
| Vulnerability with no patch available yet | Zero-day → compensating controls, segmentation |
| Single authentication point for partners | Federation (SAML/OIDC), IdP |
| One login for many internal apps | SSO (single sign-on) |
| Verify device health before net access | NAC (network access control), 802.1X posture |
| Detect-only network sensor | IDS; inline blocking = IPS |
| Aggregate + correlate logs, alerts | SIEM (security information and event management) |
| Automate response playbooks/runbooks | SOAR (security orchestration, automation, and response) |
| Malware analysis in isolation | Sandbox |
| Decoy server to study attackers | Honeypot (network of them = honeynet) |
| Stop sensitive data exfiltration | DLP (data loss prevention) |
| Manage/wipe corporate mobile devices | MDM (mobile device management) |
| Broker security between users and SaaS | CASB (cloud access security broker) |
| Filter web app traffic, block SQLi/XSS | WAF (web application firewall) |
| Time-boxed money impact of downtime | RTO; max tolerable data loss = RPO |
| Annualized loss = SLE × ARO | ALE (annualized loss expectancy) |
| Agreement: measurable service levels | SLA (service level agreement) |
| Agreement: intent, not binding details | MOU (memorandum of understanding) |
| Hardware root of trust on motherboard | TPM (trusted platform module) |
| Network device for centralized key storage | HSM (hardware security module) |
| Records who did what for later audit | Logging/auditing → accounting (the third A of AAA) |

## High-Yield Acronyms (Grouped by Theme)

```
# ---- Core concepts ----
# CIA    — Confidentiality, Integrity, Availability (the triad)
# AAA    — Authentication, Authorization, Accounting
# ACL    — Access Control List (ordered rules, implicit deny last)
# CVE    — Common Vulnerabilities and Exposures (vuln catalog IDs)
# CVSS   — Common Vulnerability Scoring System (0.0–10.0 severity)
# SCAP   — Security Content Automation Protocol (machine-readable
#          compliance/vuln content: OVAL, XCCDF, CPE)
# OSINT  — Open-Source Intelligence (public-source recon)

# ---- Cryptography and PKI ----
# AES    — Advanced Encryption Standard (symmetric, 128/192/256-bit)
# PKI    — Public Key Infrastructure (CAs, certs, trust chains)
# CSR    — Certificate Signing Request (public key + identity → CA)
# CRL    — Certificate Revocation List (batch revocation download)
# OCSP   — Online Certificate Status Protocol (per-cert live check;
#          stapling puts the response in the TLS handshake)
# HSM    — Hardware Security Module (tamper-resistant key custody)
# TPM    — Trusted Platform Module (per-device chip: keys, attestation)
# FDE    — Full-Disk Encryption (BitLocker, LUKS; SED = self-encrypting)
# PFS    — Perfect Forward Secrecy (ephemeral DH; past sessions safe
#          even if the long-term private key later leaks)
# PBKDF2 — Password-Based Key Derivation Function 2 (slow hash +
#          salt for password storage; bcrypt/scrypt/Argon2 same role)
# GPG    — GNU Privacy Guard (OpenPGP implementation)

# ---- Network security ----
# IPSec  — Internet Protocol Security; ESP = Encapsulating Security
#          Payload (encrypts), AH = Authentication Header (integrity
#          only, no confidentiality); tunnel vs transport mode
# NGFW   — Next-Generation Firewall (app-aware, IPS, TLS inspection)
# UTM    — Unified Threat Management (all-in-one SMB firewall box)
# WAF    — Web Application Firewall (L7: SQLi, XSS, OWASP Top 10)
# NAC    — Network Access Control (posture check before admission)
# EAP    — Extensible Authentication Protocol (802.1X framework;
#          EAP-TLS = cert both sides, PEAP = TLS tunnel + inner auth)
# CHAP   — Challenge-Handshake Authentication Protocol (challenge/
#          response, password never crosses the wire)
# CCMP   — Counter mode CBC-MAC Protocol (AES-based WPA2 encryption)
# WPA3   — Wi-Fi Protected Access 3 (SAE handshake replaces PSK
#          4-way; resists offline dictionary attacks)
# SASE   — Secure Access Service Edge (SD-WAN + cloud security stack)
# SD-WAN — Software-Defined Wide Area Network
# RADIUS — Remote Authentication Dial-In User Service (UDP 1812/1813,
#          encrypts only the password field; combines authn+authz)
# TACACS+— Terminal Access Controller Access-Control System Plus
#          (TCP 49, encrypts full payload, separates AAA — device
#          administration answer; RADIUS = network access answer)

# ---- Email security ----
# SPF    — Sender Policy Framework (DNS TXT: which IPs may send)
# DKIM   — DomainKeys Identified Mail (crypto signature on headers)
# DMARC  — Domain-based Message Authentication, Reporting and
#          Conformance (policy tying SPF+DKIM: none/quarantine/reject)

# ---- Identity and access ----
# IdP    — Identity Provider (asserts identity to service providers)
# SAML   — Security Assertion Markup Language (XML federation/SSO)
# PAM    — Privileged Access Management (vault, checkout, session
#          recording for admin accounts) — NOT Pluggable Auth Modules here
# HOTP   — HMAC-based One-Time Password (counter-based; valid until used)
# TOTP   — Time-based One-Time Password (30–60 s window)
# MDM    — Mobile Device Management (enroll, enforce, remote wipe)

# ---- Operations and monitoring ----
# SIEM   — Security Information and Event Management (collect,
#          normalize, correlate, alert, retain logs)
# SOAR   — Security Orchestration, Automation, and Response
#          (playbooks/runbooks acting on SIEM alerts)
# EDR    — Endpoint Detection and Response (host telemetry + response)
# XDR    — Extended Detection and Response (EDR + network + cloud +
#          identity correlated in one platform)
# DLP    — Data Loss Prevention (content inspection at endpoint,
#          email, network egress, cloud)
# CASB   — Cloud Access Security Broker (visibility/policy for SaaS)
# RAT    — Remote Access Trojan (covert remote control malware)
# XSS    — Cross-Site Scripting; CSRF — Cross-Site Request Forgery

# ---- Risk, BC/DR, and agreements ----
# SLE    — Single Loss Expectancy = asset value × exposure factor
# ARO    — Annualized Rate of Occurrence (expected events/year)
# ALE    — Annualized Loss Expectancy = SLE × ARO
# MTBF   — Mean Time Between Failures (reliability of repairable asset)
# MTTR   — Mean Time To Repair/Recover (average fix time)
# RTO    — Recovery Time Objective (max tolerable downtime)
# RPO    — Recovery Point Objective (max tolerable data loss, time)
# MOA/MOU— Memorandum of Agreement/Understanding (intent; MOA more
#          formal than MOU; usually not legally binding)
# MSA    — Master Service Agreement (umbrella terms for ongoing work)
# SOW    — Statement of Work (specific deliverables under an MSA)
# NDA    — Non-Disclosure Agreement (confidentiality obligation)
# BPA    — Business Partners Agreement (profit/obligation split)
# SLA    — Service Level Agreement (measurable targets + penalties)

# ---- Privacy and compliance ----
# GDPR   — General Data Protection Regulation (EU; controller vs
#          processor, DPO, 72-hour breach notification, right to
#          erasure)
```

## Study Plan (Using This Category)

```
# Pass 1 — breadth, in objective order (2–3 weeks):
#   cs security-plus-sy0-701        (this sheet — memorize the reflex table)
#   cs secplus-security-concepts    cs secplus-cryptography
#   cs secplus-threat-landscape     cs secplus-vulnerabilities
#   cs secplus-attack-indicators    cs secplus-hardening
#   cs secplus-architecture         cs secplus-data-resilience
#   cs secplus-asset-vuln-management cs secplus-monitoring-defense
#   cs secplus-iam                  cs secplus-incident-response
#   cs secplus-governance-risk      cs secplus-third-party-compliance
#   cs secplus-security-awareness
# Pass 2 — depth: every "Exam Cues" section in the sheets above +
#   the acronym list here until every expansion is automatic
# Pass 3 — practice exams: score ≥85% consistently before booking
#   (the real exam feels harder than most practice sets)
# Hands-on floor: run nmap + a vulnerability scan on a lab VM, read a
#   CVSS vector string, configure 802.1X in theory, walk one tabletop
#   incident-response exercise, hash-and-verify a file, and inspect a
#   real TLS certificate chain — PBQs reward muscle memory
# Deep-dive neighbours already in vor: cs pki, cs tls, cs siem,
#   cs ids-ips, cs incident-response, cs risk-management, cs zero-trust
```

## Booking and Day-Of

```
# 1. Buy voucher / schedule at CompTIA store → Pearson VUE
# 2. Test center: two forms of ID, nothing else at the desk
#    OnVUE online: system test in advance, room scan, no notes,
#    no second monitor, no one entering the room, webcam always on
# 3. Time budget: flag PBQs first, sweep multiple choice, return
# 4. Result: provisional pass/fail on screen at submission; official
#    score report in your Pearson VUE / CompTIA account shortly after
# 5. Post-pass: certification appears in CompTIA CertMaster/Certmetrics;
#    start logging CEUs immediately — 50 CEUs over 3 years renews it,
#    or stack CySA+/PenTest+/SecurityX (CASP+) to auto-renew
```

## See Also

secplus-security-concepts, secplus-cryptography, secplus-threat-landscape, secplus-vulnerabilities, secplus-attack-indicators, secplus-hardening, secplus-architecture, secplus-data-resilience, secplus-asset-vuln-management, secplus-monitoring-defense, secplus-iam, secplus-incident-response, secplus-governance-risk, secplus-third-party-compliance, secplus-security-awareness, zero-trust, pki, siem, incident-response, risk-management, nist

## References

- [CompTIA Security+ SY0-701 official exam page](https://www.comptia.org/certifications/security)
- [SY0-701 Exam Objectives (PDF)](https://www.comptia.org/training/resources/exam-objectives)
- [SY0-701 Acronym List (within the objectives PDF)](https://www.comptia.org/training/resources/exam-objectives)
- [CompTIA Continuing Education program](https://www.comptia.org/continuing-education)
- [Pearson VUE — CompTIA testing](https://home.pearsonvue.com/comptia)
- [NIST SP 800-53 Rev. 5 — Security and Privacy Controls](https://csrc.nist.gov/pubs/sp/800/53/r5/upd1/final)
- [NIST Cybersecurity Framework 2.0](https://www.nist.gov/cyberframework)
- [NIST SP 800-61 Rev. 2 — Computer Security Incident Handling Guide](https://csrc.nist.gov/pubs/sp/800/61/r2/final)
