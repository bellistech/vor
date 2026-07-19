# Security Governance & Risk Management (Security+ SY0-701 Objectives 5.1 + 5.2)

> Policy vs standard vs procedure vs guideline (a guaranteed question), governance roles from owner to custodian, and the quantitative risk math — SLE = AV × EF, ALE = SLE × ARO — with worked numbers, plus RTO/RPO/MTTR/MTBF untangled.

## Governance Document Hierarchy (5.1)

| Document | Nature | Binding? | Example |
|---|---|---|---|
| Policy | High-level statement of management intent — the WHAT and WHY | Mandatory | "All remote access requires MFA" |
| Standard | Specific uniform requirement implementing policy — the WITH WHAT | Mandatory | "MFA must use FIDO2 keys or TOTP; SMS prohibited" |
| Procedure | Step-by-step instructions — the HOW | Mandatory | "To enroll a security key: step 1…" |
| Guideline | Recommended best practice — flexibility allowed | Optional | "Prefer passphrases of four random words" |

```
# Exam pattern: quote a document, ask what it is.
#   Mandatory + broad intent            → policy
#   Mandatory + specific technology     → standard
#   Numbered steps                      → procedure
#   "Should/recommended", not required  → guideline

# Policies named in the objectives:
#   AUP (Acceptable Use Policy) — what users may/may not do with org
#     systems; signed at onboarding; the document HR cites at discipline.
#   Information security policies — the umbrella program documents.
#   Business continuity + disaster recovery — keep operating / restore
#     after disruption (bcp-drp sheet).
#   Incident response — authority + process when incidents hit
#     (secplus-incident-response).
#   SDLC (Software Development Lifecycle) — security gates in dev work
#     (sdlc-security).
#   Change management — how changes are approved/rolled back
#     (secplus-security-concepts objective 1.3).
#
# Standards commonly required: password, access control, physical
#   security, encryption (approved algorithms/key lengths →
#   secplus-cryptography).
# Procedures commonly required: change management, onboarding/
#   offboarding, playbooks (step-by-step response to a scenario).
```

## External Considerations & Governance Structures (5.1)

```
# External drivers that shape policy (know the categories):
#   Regulatory — HIPAA (health), PCI DSS (payment cards), SOX (finance)
#   Legal      — breach-notification statutes, contract law, e-discovery
#   Industry   — sector norms/frameworks (NIST CSF, ISO 27001, CIS)
#   Local/regional — state privacy laws (CCPA)
#   National   — federal law, critical-infrastructure rules
#   Global     — GDPR (General Data Protection Regulation) reaches any
#                org processing EU residents' data

# Monitoring and revision — governance is a loop: review policies on a
#   cycle and after incidents/regulatory change; version and re-approve.

# Governance structures:
#   Boards        — board of directors; ultimate accountability, sets
#                   risk appetite
#   Committees    — steering/security committees; cross-functional
#                   oversight, approve policy
#   Government entities — regulators and agencies imposing requirements
#   Centralized   — one governance body decides for all (consistent,
#                   slower, single choke point)
#   Decentralized — business units decide locally (fast, risk of
#                   inconsistency)
```

## Roles & Responsibilities for Systems and Data (5.1)

| Role | Responsibility | Exam cue |
|---|---|---|
| Owner | Accountable for the asset/data; sets classification, approves access | "Ultimately accountable", senior/business role |
| Controller | Determines WHY and HOW personal data is processed (GDPR term) | "Decides purposes and means of processing" |
| Processor | Processes data ON BEHALF OF the controller | "Cloud provider handling data per contract" |
| Custodian/steward | Day-to-day care: backups, permissions, integrity | "Implements controls the owner mandates", IT staff |

```
# Controller vs processor is GDPR language and a favorite question:
#   the retailer deciding to collect customer emails = controller;
#   the email-marketing SaaS sending on its behalf = processor.
# Owner vs custodian: owner DECIDES (classification, who gets access),
#   custodian EXECUTES (applies the ACLs, runs the backups).
```

## Risk Management Process (5.2)

```
# Risk identification — enumerate what can go wrong: threat × 
#   vulnerability × asset. Inputs: asset inventory, threat intel,
#   assessments, audits.
#
# Risk assessment cadence:
#   Ad hoc      — triggered by an event/question ("we just acquired a
#                 company — what did we inherit?")
#   Recurring   — scheduled (annual/quarterly)
#   One-time    — for a specific project/change
#   Continuous  — ongoing, automated (dashboards, KRIs, scanner feeds)
```

## Risk Analysis: Qualitative vs Quantitative (5.2)

```
# Qualitative — subjective scales (low/medium/high), heat maps, expert
#   judgment. Fast, no dollar data needed; output is a ranked matrix of
#   likelihood × impact.
# Quantitative — dollars and math. Terms (MEMORIZE):
#
#   AV  (Asset Value)                 — what the asset is worth ($)
#   EF  (Exposure Factor)             — % of value lost in one incident
#   SLE (Single Loss Expectancy)      = AV × EF
#   ARO (Annualized Rate of Occurrence) — expected events per year
#   ALE (Annualized Loss Expectancy)  = SLE × ARO
#
# Worked example 1:
#   Web server AV = $200,000; a defacement costs 25% of its value
#   (EF = 0.25) → SLE = 200,000 × 0.25 = $50,000.
#   Expected twice a year (ARO = 2) → ALE = 50,000 × 2 = $100,000/yr.
#   A $60,000/yr control that prevents it is worth buying
#   (ALE reduction > control cost).
#
# Worked example 2:
#   Laptop fleet: AV = $2,000/laptop, theft loses full value (EF = 1.0)
#   → SLE = $2,000. History says 15 thefts/year (ARO = 15)
#   → ALE = $30,000/yr. Full-disk encryption doesn't stop theft but
#   slashes the DATA exposure factor — recompute EF with data value in
#   AV and the control justifies itself.
#
# Worked example 3 (reverse question): ALE = $40,000 and ARO = 0.5
#   (once every 2 years) → SLE = ALE / ARO = $80,000.
#
# Probability vs likelihood — probability is the statistical number;
#   likelihood is the (often qualitative) chance rating. Impact = size
#   of harm when it happens.
```

## Risk Register, Tolerance & Appetite (5.2)

```
# Risk register — the living inventory of identified risks. Columns:
#   description, owner, likelihood, impact, score, response, status.
#   KRI (Key Risk Indicator) — metric that warns a risk is rising
#     (e.g. % unpatched criticals, phishing click rate).
#   Risk owner — the accountable person per risk.
#   Risk threshold — the line that triggers action/escalation.
#
# Risk tolerance — how much deviation from the appetite the org will
#   absorb in practice.
# Risk appetite — the amount/type of risk leadership WANTS to take:
#   Expansionary — take more risk chasing growth (startups)
#   Conservative — minimize risk (banks, healthcare)
#   Neutral      — balanced
```

## Risk Management Strategies (5.2)

| Strategy | Meaning | Scenario cue |
|---|---|---|
| Transfer | Shift financial consequence to another party | "Purchased cyber insurance", outsourcing with liability terms |
| Accept | Proceed knowingly; document it | "Cost of fixing exceeds the loss" |
| — exemption | Approved deviation: control never applies to this system | Legacy box formally excluded from policy |
| — exception | Approved TEMPORARY deviation, with expiry | "90-day waiver while migrating" |
| Avoid | Eliminate the activity entirely | "Discontinued the feature/service" |
| Mitigate | Apply controls to reduce likelihood/impact | "Deployed WAF and patched" — the default |

```
# Residual risk = risk remaining AFTER controls. It never reaches zero;
# leadership formally accepts what's left (sign-off = accountability).
# Risk reporting — registers, heat maps, KRI dashboards up to the board;
# the output that makes governance (5.1) work.
```

## Business Impact Analysis (5.2)

```
# BIA (Business Impact Analysis) — determines which functions are
#   critical and what downtime/data loss actually costs; feeds BCP/DRP.
#
#   RTO (Recovery Time Objective)  — max tolerable DOWNTIME: how long
#     until the service must be back.
#   RPO (Recovery Point Objective) — max tolerable DATA LOSS, measured
#     backwards in time: how old may the restored data be. Drives
#     backup frequency: RPO 1h → back up (at least) hourly.
#   MTTR (Mean Time To Repair)     — average time to fix a failure.
#   MTBF (Mean Time Between Failures) — average uptime between
#     failures; reliability predictor for components.
#
#   Timeline:
#     ---last backup----X(failure)--------repaired---------
#            |<--RPO-->|         |<-MTTR->|
#                                |<---- must be < RTO ---->|
#
# Trap: RTO is about TIME DOWN, RPO is about DATA LOST. "We can lose at
# most 15 minutes of transactions" → RPO. "Site must be up within 4
# hours" → RTO. MTBF is a hardware-reliability stat, not an objective.
```

## Exam Cues (5.1 + 5.2)

```
# "mandatory, high-level management intent"     → policy
# "specific required technology/config"         → standard
# "step-by-step"                                → procedure
# "recommended, flexible"                       → guideline
# "document users sign about system usage"      → AUP
# "decides why data is processed"               → controller
# "processes on the controller's behalf"        → processor
# "performs backups per the owner's rules"      → custodian
# "SLE × ARO"                                   → ALE
# "AV × EF"                                     → SLE
# "bought insurance"                            → transfer
# "documented decision to live with it"         → accept
# "temporary approved deviation"                → exception
# "stopped offering the risky service"          → avoid
# "risk remaining after controls"               → residual risk
# "board's desired level of risk"               → risk appetite
# "metric that flags rising risk"               → KRI
# "maximum tolerable downtime"                  → RTO
# "maximum tolerable data loss"                 → RPO
```

## See Also

security-plus-sy0-701, secplus-security-concepts, secplus-asset-vuln-management, secplus-third-party-compliance, secplus-data-resilience, secplus-incident-response, secplus-security-awareness, risk-management, security-governance, bcp-drp, gdpr, hipaa, pci-dss, iso27001, nist, soc2

## References

- [CompTIA Security+ SY0-701 Exam Objectives](https://www.comptia.org/certifications/security)
- [NIST SP 800-30 Rev. 1 — Guide for Conducting Risk Assessments](https://csrc.nist.gov/pubs/sp/800/30/r1/final)
- [NIST SP 800-37 Rev. 2 — Risk Management Framework](https://csrc.nist.gov/pubs/sp/800/37/r2/final)
- [NIST SP 800-34 Rev. 1 — Contingency Planning Guide (BIA, RTO/RPO)](https://csrc.nist.gov/pubs/sp/800/34/r1/upd1/final)
- [ISO/IEC 27005 — Information security risk management](https://www.iso.org/standard/80585.html)
- [ISO 31000 — Risk management guidelines](https://www.iso.org/iso-31000-risk-management.html)
- [GDPR — Article 4 definitions (controller/processor)](https://gdpr-info.eu/art-4-gdpr/)
