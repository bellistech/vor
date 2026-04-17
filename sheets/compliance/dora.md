# DORA (Digital Operational Resilience Act)

EU regulation (2022/2554) that harmonizes ICT risk management, incident reporting, resilience testing, and third-party oversight across the financial sector, in force from 17 January 2025 with binding Regulatory Technical Standards (RTS) and Implementing Technical Standards (ITS) from the ESAs.

## Scope and Who Is Covered

### Financial Entities in Scope
```
Credit institutions (banks)
Payment institutions
Electronic money institutions
Investment firms
Crypto-asset service providers (CASPs)
Central securities depositories (CSDs)
Central counterparties (CCPs)
Trading venues
Trade repositories
Managers of alternative investment funds (AIFMs)
Management companies (UCITS)
Insurance and reinsurance undertakings
Insurance intermediaries
Institutions for occupational retirement provision (IORPs)
Credit rating agencies
Statutory auditors
Administrators of critical benchmarks
Crowdfunding service providers
Securitization repositories
```

### Proportionality
```
Microenterprises (<10 staff, <=€2M turnover):
  Simplified ICT risk management framework
  Exempt from certain testing requirements

Significant entities:
  Full framework, advanced testing (TLPT)
  Board-level oversight

Critical ICT third-party providers (CTPPs):
  Designated by ESAs
  Oversight by lead overseer (EBA, EIOPA, or ESMA)
  Annual oversight fee
```

## The Five Pillars

### Pillar 1 — ICT Risk Management (Arts. 5-16)
```
Governance:
  - Management body approves framework, ultimately accountable
  - Dedicated ICT risk control function, independent from ops
  - Annual review of ICT risk management framework

Framework components:
  1. Identification of ICT-supported business functions
  2. Protection (security policies, access control, crypto)
  3. Detection (continuous monitoring, anomaly detection)
  4. Response and recovery (BCP, DR, crisis communication)
  5. Learning and evolving (post-incident review)
  6. Communication (internal and external stakeholders)

Documentation required:
  - ICT risk management framework (written)
  - Digital operational resilience strategy
  - Business continuity policy
  - ICT response and recovery plans
  - Backup policy and procedures
```

### Pillar 2 — Incident Reporting (Arts. 17-23)
```
Classification thresholds (RTS JC 2023 83):
  Duration: >= 24 hours service unavailability
  Geographical spread: >= 2 EU member states affected
  Affected clients: >= 10% of active clients OR absolute number thresholds
  Reputational impact: media coverage, regulator attention
  Data losses: confidentiality/integrity/availability breach
  Economic impact: direct + indirect costs vs thresholds

Reporting timeline (major incidents):
  Initial notification: within 4 hours of classification, max 24 hours from detection
  Intermediate report: within 72 hours, or at status change
  Final report: within 1 month of incident closure

Significant cyber threat (voluntary):
  Entities MAY report significant cyber threats
  Encouraged but not mandatory
  Separate template from incident reports

Competent authority:
  National competent authority (NCA) of home member state
  Cross-border incidents: info shared via ESAs
```

### Pillar 3 — Digital Operational Resilience Testing (Arts. 24-27)
```
Basic testing (all entities, annual):
  - Vulnerability assessments and scans
  - Open source analyses
  - Network security assessments
  - Gap analyses
  - Physical security reviews
  - Questionnaires and scanning software
  - Source code reviews where feasible
  - Scenario-based tests
  - Compatibility testing
  - Performance testing
  - End-to-end testing
  - Penetration testing

Advanced testing — TLPT (significant entities, every 3 years):
  Threat-Led Penetration Testing per TIBER-EU
  Covers live production systems
  External testers accredited or internal + external blend
  Lead overseer: competent authority of home MS
  Mutual recognition across EU
  Report to competent authority within agreed timeline

TLPT phases:
  1. Preparation (scoping, threat intel)
  2. Threat intelligence (specific threat scenarios)
  3. Red team testing (live attack simulation)
  4. Closure (remediation plan, lessons learned)
```

### Pillar 4 — ICT Third-Party Risk (Arts. 28-44)
```
Register of information (Art. 28):
  All contractual arrangements with ICT third-party providers
  Reported annually to competent authorities via ESA templates
  Includes: function supported, criticality, location, sub-outsourcing chain

Pre-contract due diligence:
  - Provider's information security standards
  - Incident history
  - Concentration risk assessment
  - Data location and transfer mechanisms
  - Exit strategy feasibility

Mandatory contractual clauses (Art. 30):
  - Full service description, locations
  - Service levels including monitoring
  - Data protection and confidentiality
  - Insolvency / resolution access rights
  - Incident reporting and cooperation
  - Audit rights (entity and regulator)
  - Termination rights and exit support
  - Cooperation with competent authorities
  - Right to subcontract only with notification

Critical providers (designated by ESAs):
  - Lead overseer performs oversight
  - General and specific investigations
  - Recommendations binding in practice
  - Financial penalties up to 1% of daily worldwide turnover
  - Non-EU providers must establish EU subsidiary
```

### Pillar 5 — Information Sharing (Art. 45)
```
Voluntary information-sharing arrangements:
  Cyber threat information and intelligence
  Tactics, techniques, procedures (TTPs)
  Indicators of compromise (IOCs)
  Must protect confidentiality and comply with GDPR
  Notified to competent authorities
```

## Implementation Workflow

### Register of Information Schema
```bash
# ESA register schema (annual submission)
cat <<'EOF' > dora_register.yaml
entity:
  lei: 549300XXXXXXXXXXX01
  name: "Example Bank SA"
  country: "FR"

ict_services:
  - contract_id: "CTR-2024-0042"
    provider_lei: "549300YYYYYYYYYYY02"
    provider_name: "CloudVendor Inc"
    service_type: "S01_cloud_compute"
    is_critical: true
    is_ccp_function: false
    data_location: ["EU-FR", "EU-DE"]
    data_transfer_outside_eu: false
    sub_outsourcing_chain:
      - tier: 1
        provider: "SubVendor Ltd"
        location: "IE"
    annual_budget_eur: 450000
    contract_start: "2024-01-15"
    exit_strategy_documented: true
    last_due_diligence: "2025-03-01"
EOF
```

### Incident Classification Engine
```python
# Map incident to DORA classification (major / non-major)
def classify_incident(incident):
    score = 0
    if incident.duration_hours >= 24:
        score += 1
    if incident.affected_member_states >= 2:
        score += 1
    if incident.affected_clients_pct >= 10:
        score += 1
    if incident.data_confidentiality_impacted:
        score += 1
    if incident.data_integrity_impacted:
        score += 1
    if incident.data_availability_impacted:
        score += 1
    if incident.economic_impact_eur >= 100_000:
        score += 1
    if incident.reputational_impact == "high":
        score += 1

    # RTS: at least 2 primary criteria or 1 primary + 2 secondary = major
    return "major" if score >= 2 else "non_major"
```

### TLPT Scope Template
```
Live production target systems:
  - Internet-facing customer portal
  - Core banking ledger
  - Payment processing gateway
  - Identity/access management

Threat scenarios (from threat intel):
  - Ransomware targeting backup systems
  - Supply chain compromise via sub-outsourcer
  - Insider threat with privileged access
  - Credential stuffing on customer portal

Red team rules of engagement:
  - No destructive actions on production data
  - Detection testing (blue team awareness level: minimal)
  - Goal flag capture within scoped systems
  - 12-week active testing window
```

## Penalties

### Administrative Measures
```
Financial entities:
  - Periodic penalty payments (up to 1% daily turnover)
  - Administrative fines per member state rules
  - Public reprimands, cease-and-desist orders
  - Withdrawal of authorization (severe cases)

Critical ICT third-party providers:
  - Lead overseer penalties up to 1% daily average worldwide turnover
  - Per-day fines for continuing non-compliance
  - Public statements of non-compliance

Criminal sanctions:
  - Member states may add criminal penalties
  - National implementation varies
```

## Tips
- Build the register of information into your procurement workflow — every new contract automatically files the required fields, never retrofitted at year-end
- Classify incidents against DORA criteria in your SOAR platform so the 4-hour clock starts automatically when a major threshold trips
- Treat sub-outsourcing as first-class in your vendor inventory; concentration risk often hides three tiers deep
- Align TLPT preparation with existing red team exercises — the TIBER-EU framework is designed to reuse mature programs, not replace them
- Document your exit strategy with executable runbooks, not aspirational prose; regulators specifically test data portability and termination feasibility
- Map DORA ICT risk controls to your existing ISO 27001 / NIST CSF controls to avoid duplicate evidence collection
- For critical ICT providers, negotiate audit and step-in rights up front; retroactive contract amendments are extremely difficult once production is live
- Incident reporting templates evolve — monitor the JC RTS and ITS publications on the ESAs websites and version-control your reporting pipelines
- Classify ICT assets by business function criticality first, then by technology layer; DORA thinks in business-function terms, not in servers
- Board-level sign-off is not a formality — the management body is personally accountable under DORA, so quarterly board packs must cover ICT risk KPIs
- If you operate cross-border, designate a single lead competent authority contact point to avoid conflicting guidance during incidents

## See Also
- gdpr, nist, iso27001, pci-dss, soc2, eu-ai-act, supply-chain-security, incident-response, threat-modeling, bcp-drp

## References
- [Regulation (EU) 2022/2554 (DORA)](https://eur-lex.europa.eu/eli/reg/2022/2554/oj)
- [EBA DORA Technical Standards](https://www.eba.europa.eu/regulation-and-policy/operational-resilience)
- [ESMA DORA Overview](https://www.esma.europa.eu/esmas-activities/digital-finance-and-innovation/digital-operational-resilience-act-dora)
- [EIOPA DORA Implementation](https://www.eiopa.europa.eu/digital-operational-resilience-act-dora_en)
- [TIBER-EU Framework](https://www.ecb.europa.eu/paym/cyber-resilience/tiber-eu/html/index.en.html)
- [Joint Committee Final Reports on RTS/ITS](https://www.esma.europa.eu/document/joint-committee-final-report-dora-rts-its)
