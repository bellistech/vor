# Third-Party Risk, Compliance & Audits (Security+ SY0-701 Objectives 5.3 + 5.4 + 5.5)

> Vendor vetting and the agreement-type alphabet (SLA/MOU/MSA/SOW/NDA/BPA — a guaranteed question), compliance monitoring and GDPR privacy roles, then audits and the pentest taxonomy: known/partially known/unknown environments, passive vs active recon.

## Vendor Assessment (5.3)

```
# Ways to evaluate a third party BEFORE and DURING the relationship:
#   Penetration testing        — vendor allows (or supplies results of)
#                                pentests against their environment.
#   Right-to-audit clause      — contract term letting YOU (or your
#                                agent) audit the vendor's controls.
#                                No clause = no leverage later.
#   Evidence of internal audits — vendor shares their own audit results.
#   Independent assessments    — third-party reports: SOC 2 Type II,
#                                ISO 27001 certification — trusted
#                                because the assessor isn't the vendor.
#   Supply chain analysis      — mapping the vendor's OWN dependencies:
#                                their hosting, their subprocessors,
#                                their component sources. Your risk
#                                includes their fourth parties.

# Vendor selection:
#   Due diligence      — investigating before signing: financial health,
#                        security posture, references, breach history,
#                        certifications.
#   Conflict of interest — relationships that bias judgment (vendor is
#                        owned by a board member; assessor also sells
#                        remediation). Must be disclosed/avoided.

# Vendor monitoring — the relationship isn't "set and forget":
#   periodic reassessment, SLA tracking, KPI reviews, watching for
#   breaches/news, re-running questionnaires.
# Questionnaires — standardized security question sets (SIG, CAIQ)
#   vendors answer; cheap breadth, self-reported (verify the criticals).
# Rules of engagement — for any active testing of a vendor (or by one):
#   scope, timing, targets, allowed techniques, contacts — agreed IN
#   WRITING before testing starts.
```

## Agreement Types (5.3) — the disambiguation table

| Agreement | What it establishes | Binding? | Exam cue |
|---|---|---|---|
| SLA (Service-Level Agreement) | Measurable performance targets: uptime %, response time, penalties | Yes | "99.9% uptime or credits" |
| MOA (Memorandum of Agreement) | Specific cooperative terms between parties | Usually | More formal/conditional than MOU |
| MOU (Memorandum of Understanding) | Broad mutual intent to work together | Usually NOT legally binding | "Statement of intent", handshake-on-paper |
| MSA (Master Service Agreement) | Umbrella legal terms governing the whole relationship | Yes | "Overall terms; individual projects added later" |
| WO/SOW (Work Order / Statement of Work) | ONE project's scope, deliverables, timeline, price — under the MSA | Yes | "Defines this specific engagement" |
| NDA (Non-Disclosure Agreement) | Confidentiality — what shared info must stay secret | Yes | "Before sharing sensitive designs" |
| BPA (Business Partners Agreement) | Partnership mechanics: profit split, responsibilities, exit | Yes | "Two companies going to market together" |

```
# Classic pairing: MSA (relationship umbrella) + SOW per engagement.
# MOU vs MOA vs contract: MOU = intent (weakest), MOA = agreed actions,
# MSA/SOW = enforceable engagement terms.
# Pentest paperwork = rules of engagement + NDA + SOW.
```

## Compliance (5.4)

```
# Compliance reporting:
#   Internal — dashboards/attestations to management, audit committee.
#   External — filings to regulators, customers, auditors (PCI ROC/SAQ,
#              GDPR records of processing, breach notifications).

# Consequences of non-compliance:
#   Fines               — GDPR: up to €20M or 4% global revenue;
#                         HIPAA: tiered civil penalties
#   Sanctions           — restrictions imposed by regulators
#   Reputational damage — customer/market trust loss after disclosure
#   Loss of license     — losing authority to operate (banking,
#                         healthcare, payment processing)
#   Contractual impacts — breach of contract, lost deals, indemnities;
#                         card brands can revoke card acceptance

# Compliance monitoring:
#   Due diligence / due care — diligence = investigate before acting
#     (know the obligations); care = act reasonably on what you know
#     (implement/maintain controls). Exam: diligence THEN care.
#   Attestation and acknowledgement — signed statements: employees
#     acknowledge policies; executives attest control effectiveness.
#   Internal and external monitoring — self-checks + outside
#     verification (auditors, regulators, assessors).
#   Automation — compliance-as-code: continuous config checks (SCAP,
#     CIS benchmarks), automated evidence collection, drift alerts.
```

## Privacy (5.4)

```
# Legal implications stack by geography — you comply with ALL layers:
#   Local/regional — US state laws (CCPA/CPRA California), provincial
#   National       — country-level (HIPAA, GLBA sector laws)
#   Global         — GDPR's extraterritorial reach: applies to anyone
#                    processing EU residents' data, anywhere

# GDPR vocabulary (tested):
#   Data subject      — the human the data is about
#   Controller        — decides WHY/HOW data is processed
#   Processor         — processes on the controller's behalf
#     (roles detail: secplus-governance-risk)
#   Ownership         — org accountability for data it holds
#   Data inventory and retention — know WHAT personal data you hold,
#     WHERE, and keep it only as long as needed (retention schedule)
#   Right to be forgotten — data subject may demand erasure (GDPR
#     Art. 17 "right to erasure") when no lawful basis remains
```

## Audits & Assessments (5.5)

```
# Attestation — formal statement by a qualified party that controls
#   meet a standard (SOC 2 report is an attestation engagement).
#
# Internal:
#   Compliance audits  — self-check against policy/regulation
#   Audit committee    — board subcommittee overseeing audit work and
#                        independence
#   Self-assessments   — questionnaire-driven reviews by the teams
#                        themselves (cheap, least independent)
#
# External:
#   Regulatory examinations — the regulator audits you (bank exams)
#   Assessment              — third-party technical/gap evaluation
#   Independent third-party audit — certified outside auditor
#     (ISO 27001 certification body, SOC 2 CPA firm) — highest
#     independence, required by many customers
```

## Penetration Testing Taxonomy (5.5)

```
# By posture:
#   Physical   — testing doors, badges, tailgating, dumpster diving
#   Offensive  — red team: attack to find/exploit weaknesses
#   Defensive  — blue team: detect/respond to the attack
#   Integrated — purple team: red + blue cooperating, sharing findings
#                in real time

# By knowledge given to testers (new names for old colors):
#   Known environment           (was "white box") — full docs, source,
#     creds. Deepest coverage per hour; models insider knowledge.
#   Partially known environment (was "gray box")  — some info (an
#     account, network map). Balanced realism/efficiency.
#   Unknown environment         (was "black box") — nothing but the
#     target name. Most realistic external-attacker view; slowest.

# Reconnaissance:
#   Passive — gather without touching the target: OSINT, DNS records,
#     certificate transparency logs, job postings, social media.
#     Undetectable by the victim.
#   Active  — direct interaction: port scans, banner grabbing,
#     enumeration. Faster, richer — and detectable/loggable.
#
# Exam: "testers were given source code and admin creds" → known
# environment. "only the company name" → unknown. "queried public DNS
# and LinkedIn" → passive recon. "ran nmap against the perimeter" →
# active recon. "red and blue teams working together" → integrated.
```

## Exam Cues (5.3 + 5.4 + 5.5)

```
# "contract clause allowing us to audit them"   → right-to-audit
# "SOC 2 Type II provided"                      → independent assessment
# "who do THEY depend on?"                      → supply chain analysis
# "assessor profits from findings"              → conflict of interest
# "uptime guarantee with penalties"             → SLA
# "non-binding statement of intent"             → MOU
# "umbrella terms + per-project docs"           → MSA + SOW
# "protect secrets before demo"                 → NDA
# "joint venture responsibilities"              → BPA
# "scope and timing agreed before pentest"      → rules of engagement
# "4% of global revenue"                        → GDPR fine
# "no longer allowed to process cards"          → loss of license / contractual
# "investigate first, then act reasonably"      → due diligence / due care
# "signed policy acknowledgement"               → attestation & acknowledgement
# "continuous automated config compliance"      → compliance automation
# "person the data is about"                    → data subject
# "delete my data"                              → right to be forgotten
# "board group overseeing audits"               → audit committee
# "full source and docs given to testers"       → known environment
# "OSINT only, target never touched"            → passive reconnaissance
```

## See Also

security-plus-sy0-701, secplus-governance-risk, secplus-asset-vuln-management, secplus-security-awareness, secplus-monitoring-defense, secplus-incident-response, supply-chain-security, risk-management, security-governance, security-assessment, gdpr, hipaa, pci-dss, soc2, iso27001, nist, fedramp, privacy-regulations, cis-benchmarks

## References

- [CompTIA Security+ SY0-701 Exam Objectives](https://www.comptia.org/certifications/security)
- [NIST SP 800-161 Rev. 1 — Cybersecurity Supply Chain Risk Management](https://csrc.nist.gov/pubs/sp/800/161/r1/final)
- [GDPR — Article 17 Right to erasure](https://gdpr-info.eu/art-17-gdpr/)
- [GDPR — Article 83 Fines](https://gdpr-info.eu/art-83-gdpr/)
- [AICPA — SOC 2 Reports](https://www.aicpa-cima.com/topic/audit-assurance/audit-and-assurance-greater-than-soc-2)
- [ISO/IEC 27001 — Information security management systems](https://www.iso.org/standard/27001)
- [PCI Security Standards Council — Document Library](https://www.pcisecuritystandards.org/document_library/)
- [NIST SP 800-115 — Technical Guide to Information Security Testing and Assessment](https://csrc.nist.gov/pubs/sp/800/115/final)
- [PTES — Penetration Testing Execution Standard](http://www.pentest-standard.org/)
