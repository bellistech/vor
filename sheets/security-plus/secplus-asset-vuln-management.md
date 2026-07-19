# Asset & Vulnerability Management (Security+ SY0-701 Objectives 4.2 + 4.3)

> The asset lifecycle from procurement to certified destruction (NIST 800-88 clear/purge/destroy), then the vulnerability management loop: find it (scans, feeds, pentests, bug bounties), rate it (CVSS/CVE, false positives), fix it, and prove you fixed it.

## Asset Management Lifecycle (4.2)

```
# Acquisition/procurement process
#   Security starts BEFORE purchase: approved-vendor lists, security
#   requirements in the RFP, supply-chain vetting, licensing terms.
#   Untracked purchases become shadow IT.

# Assignment/accounting
#   Ownership      — every asset has a named owner accountable for its
#                    security and lifecycle. "Who approves changes to
#                    this server?" → the owner.
#   Classification — label assets by the sensitivity of the data they
#                    handle (public/internal/confidential/restricted) →
#                    drives the level of protection they get
#                    (secplus-data-resilience for the data side).

# Monitoring/asset tracking
#   Inventory   — the authoritative list: what you own, where it is,
#                 who owns it, what runs on it. You cannot patch or
#                 defend what you don't know exists — inventory is the
#                 prerequisite for every other control.
#   Enumeration — actively discovering what's really on the network
#                 (scans, agent check-ins) and reconciling against the
#                 inventory. Deltas = rogue or forgotten devices.

# Disposal/decommissioning
#   Sanitization  — removing data before reuse/disposal (see table).
#   Destruction   — physically destroying media when data must never
#                   be recoverable.
#   Certification — documented proof (certificate of destruction) that
#                   sanitization/destruction happened — audit evidence.
#   Data retention — legal/regulatory hold periods must be satisfied
#                   BEFORE destruction; destroying data under legal
#                   hold is spoliation.
```

## Sanitization Methods (NIST SP 800-88 mapping)

| Method | NIST level | What happens | When to use |
|---|---|---|---|
| Delete/format | (none — inadequate) | Pointers removed, data recoverable | Never for sensitive data |
| Overwrite/wipe | Clear | Every sector overwritten | Media reused internally |
| Cryptographic erase | Purge | Destroy the encryption key of a self-encrypting drive — data instantly unreadable | Fast SSD sanitization (SEDs) |
| Degaussing | Purge/Destroy | Magnetic field scrambles platters (kills the drive) | Magnetic media only — useless on SSDs |
| Shredding/incineration | Destroy | Physical destruction | Leaving org custody, highest sensitivity |

```
# Exam traps:
# SSDs — overwriting is unreliable (wear leveling hides sectors) and
#   degaussing does nothing (no magnetic storage). Answers: crypto-erase
#   or physical destruction.
# "Drives leave the building for disposal" → destruction + certificate.
# "Reuse laptops inside the company"       → clear (overwrite) is enough.
```

## Vulnerability Identification Methods (4.3)

```
# Vulnerability scan — automated check against known-vuln signatures
#   (credentialed scans see inside the host = fewer false positives;
#   non-credentialed shows the attacker's outside view).
#   Details: vulnerability-scanning sheet.

# Application security testing:
#   Static analysis (SAST — Static Application Security Testing) —
#     examines SOURCE CODE without running it; early in pipeline.
#   Dynamic analysis (DAST) — probes the RUNNING app from outside;
#     finds runtime/config issues; no source needed.
#   Package monitoring — watching third-party dependencies for new
#     CVEs (Common Vulnerabilities and Exposures) — the npm/pip supply
#     chain problem; SBOM (Software Bill of Materials) feeds this.

# Threat feeds:
#   OSINT (Open-Source Intelligence)   — free/public: blogs, advisories,
#                                        social media, paste sites.
#   Proprietary/third-party            — commercial curated intel.
#   Information-sharing organization   — ISACs (Information Sharing and
#     Analysis Centers) — sector-specific exchanges (FS-ISAC, MS-ISAC).
#   Dark web monitoring                — watching criminal markets/forums
#     for your data, creds, or planned attacks.

# Penetration testing — humans exploiting weaknesses to prove impact
#   (goes beyond scanning's "version says vulnerable").
#   Full taxonomy: secplus-third-party-compliance (objective 5.5).

# Responsible disclosure program — published rules letting outside
#   researchers report vulns safely.
#   Bug bounty program — responsible disclosure + cash rewards scaled
#   to severity.

# System/process audit — structured review of a system or process
#   against policy/standards; catches procedural gaps scanners can't.
```

## Analysis: Triage & Prioritization (4.3)

```
# Confirmation:
#   False positive — scanner says vulnerable, it isn't. Cost: wasted
#     remediation effort. Verify before ticketing.
#   False negative — scanner says clean, vulnerability exists. Cost:
#     unseen exposure — the dangerous one.
#   (True positive/negative = scanner right.)

# CVE (Common Vulnerabilities and Exposures) — the dictionary: one ID
#   per public vulnerability (CVE-2024-12345), maintained by MITRE.
# CVSS (Common Vulnerability Scoring System) — the severity math:
#   base score 0.0–10.0 from exploitability (vector, complexity,
#   privileges, user interaction) + impact (C/I/A).
#
#   CVSS v3.x bands:      0.0        None
#                         0.1–3.9    Low
#                         4.0–6.9    Medium
#                         7.0–8.9    High
#                         9.0–10.0   Critical
#
#   Example vector: CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H = 9.8
#     AV:N network-reachable, AC:L low complexity, PR:N no privileges
#     needed, UI:N no user interaction → wormable-class critical.
#
# CVSS ranks technical severity — NOT your actual risk. Prioritize with:
#   Vulnerability classification — type/category of weakness
#   Exposure factor       — how much of the asset's value is at risk
#   Environmental variables — is the vulnerable service even reachable?
#                             compensating controls in front of it?
#   Industry/organizational impact — what the loss means to YOUR business
#   Risk tolerance        — how much residual risk leadership accepts
#     (secplus-governance-risk)
# A 9.8 on an air-gapped lab box can rank below a 6.5 on the
# internet-facing payment gateway. Exploited-in-the-wild (CISA KEV
# catalog) beats theoretical severity.
```

## Response & Remediation (4.3)

```
# Patching — the default fix; test → deploy per change management
#   (secplus-security-concepts objective 1.3).
# Insurance — cyber insurance transfers financial risk when you can't
#   eliminate technical risk (risk transference).
# Segmentation — isolate the vulnerable system so the flaw is
#   unreachable (the go-to for unpatchable ICS/legacy).
# Compensating controls — alternative control achieving equivalent
#   protection when the primary fix is impossible: WAF rule in front
#   of the unpatchable app, extra monitoring, stricter ACLs.
# Exceptions and exemptions — formally documented, risk-accepted,
#   time-boxed decisions NOT to remediate — signed by the risk owner,
#   tracked in the risk register.
```

## Validation & Reporting (4.3)

```
# Validation of remediation:
#   Rescanning   — run the same scan; confirm the finding is gone.
#   Audit        — independent review that the fix (and process) stuck.
#   Verification — targeted manual confirmation (attempt the exploit,
#                  check the version/config directly).
# Reporting — metrics up and out: open critical/high counts, mean time
#   to remediate (MTTR), SLA (Service-Level Agreement) compliance per
#   severity, trend lines, exceptions in force. Reports drive the loop:
#   identify → analyze → remediate → validate → report → repeat.
```

## Exam Cues (4.2 + 4.3)

```
# "cannot defend what you don't know you have"   → asset inventory
# "discover devices actually on the network"     → enumeration
# "prove the drive was destroyed"                → certification (of destruction)
# "SSD must be sanitized quickly"                → cryptographic erase
# "degauss an SSD"                               → trick — doesn't work
# "review source code before deployment"         → static analysis (SAST)
# "test the running application externally"      → dynamic analysis (DAST)
# "alert when a dependency gets a CVE"           → package monitoring
# "sector-specific threat sharing"               → ISAC / information-sharing org
# "credentials found on criminal forum"          → dark web monitoring
# "reward outside researchers"                   → bug bounty
# "scanner flagged it but it's not exploitable"  → false positive
# "scanner missed a real vuln"                   → false negative
# "9.8 base score, internet-facing"              → remediate first (critical)
# "can't patch — what now?"                      → segmentation / compensating controls
# "business signs off on not fixing"             → exception (risk acceptance)
# "prove the patch worked"                       → rescan
```

## See Also

security-plus-sy0-701, secplus-vulnerabilities, secplus-hardening, secplus-monitoring-defense, secplus-governance-risk, secplus-third-party-compliance, secplus-incident-response, vulnerability-scanning, cve, sbom, sast-dast, asset-security, threat-hunting, risk-management, security-assessment

## References

- [CompTIA Security+ SY0-701 Exam Objectives](https://www.comptia.org/certifications/security)
- [NIST SP 800-88 Rev. 1 — Guidelines for Media Sanitization](https://csrc.nist.gov/pubs/sp/800/88/r1/final)
- [NIST SP 800-40 Rev. 4 — Guide to Enterprise Patch Management Planning](https://csrc.nist.gov/pubs/sp/800/40/r4/final)
- [FIRST — CVSS v3.1 Specification](https://www.first.org/cvss/v3.1/specification-document)
- [MITRE CVE Program](https://www.cve.org/)
- [CISA Known Exploited Vulnerabilities Catalog](https://www.cisa.gov/known-exploited-vulnerabilities-catalog)
- [National ISAC Council](https://www.nationalisacs.org/)
- [NIST SP 800-115 — Technical Guide to Information Security Testing and Assessment](https://csrc.nist.gov/pubs/sp/800/115/final)
