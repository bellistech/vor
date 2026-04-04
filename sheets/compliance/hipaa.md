# HIPAA (Health Insurance Portability and Accountability Act)

The federal law establishing national standards for protecting individuals' medical records and personal health information, applying to covered entities and business associates, with civil penalties ranging from $100 to $50,000 per violation and criminal penalties up to $250,000 with imprisonment.

## Protected Health Information (PHI)
### The 18 HIPAA Identifiers
```
PHI is any individually identifiable health information held or
transmitted by a covered entity or its business associate.

The 18 identifiers that make health data "individually identifiable":

 1. Names
 2. Geographic data smaller than a state
 3. Dates (except year) related to an individual
 4. Phone numbers
 5. Fax numbers
 6. Email addresses
 7. Social Security numbers
 8. Medical record numbers
 9. Health plan beneficiary numbers
10. Account numbers
11. Certificate/license numbers
12. Vehicle identifiers and serial numbers
13. Device identifiers and serial numbers
14. Web URLs
15. IP addresses
16. Biometric identifiers
17. Full-face photographs
18. Any other unique identifying number or code

De-identification methods (Section 164.514):
  Expert Determination: Statistician certifies re-identification risk
                        is "very small" (Safe Harbor threshold < 0.04)
  Safe Harbor:          Remove all 18 identifiers + no actual knowledge
                        that residual info could identify an individual
```

## Privacy Rule (45 CFR Part 160 and Subparts A, E of Part 164)
### Core Provisions
```
Who it applies to (Covered Entities):
  1. Health Plans (insurers, HMOs, Medicare, Medicaid)
  2. Health Care Clearinghouses
  3. Health Care Providers who transmit electronically

Key provisions:
  - Minimum Necessary Standard: Use/disclose only the minimum PHI
    needed for the intended purpose
  - Patient rights: access, amendment, accounting of disclosures,
    restriction requests, confidential communications
  - Notice of Privacy Practices (NPP): Must be provided at first
    encounter; describes uses, disclosures, and patient rights
  - Marketing restrictions: Prior written authorization required
    for most marketing uses of PHI
  - Fundraising: Opt-out right must be provided

Permitted disclosures WITHOUT patient authorization:
  - Treatment, Payment, Healthcare Operations (TPO)
  - Public health activities
  - Victims of abuse, neglect, domestic violence
  - Judicial/administrative proceedings
  - Law enforcement purposes
  - Decedents (coroners, funeral directors)
  - Organ donation
  - Research with IRB/Privacy Board waiver
  - Serious threat to health/safety
  - Workers' compensation
```

## Security Rule (45 CFR Part 160 and Subparts A, C of Part 164)
### Administrative Safeguards (Section 164.308)
```bash
# Administrative safeguards — the largest and most detailed section
cat <<'EOF' > hipaa_admin_safeguards.yaml
administrative_safeguards:
  security_management_process:
    - risk_analysis: "Required — identify threats to ePHI"
    - risk_management: "Required — reduce risks to reasonable level"
    - sanction_policy: "Required — discipline for violations"
    - information_system_review: "Required — audit logs review"

  assigned_security_responsibility:
    - security_officer: "Required — single designated individual"

  workforce_security:
    - authorization: "Addressable — access based on role"
    - clearance_procedures: "Addressable — background checks"
    - termination_procedures: "Addressable — revoke access on departure"

  information_access_management:
    - access_authorization: "Addressable — policies for granting access"
    - access_establishment: "Addressable — provisioning process"
    - access_modification: "Addressable — re-evaluation procedures"

  security_awareness_training:
    - security_reminders: "Addressable — periodic updates"
    - malicious_software: "Addressable — protection procedures"
    - login_monitoring: "Addressable — detect failed attempts"
    - password_management: "Addressable — create/change/safeguard"

  security_incident_procedures:
    - response_and_reporting: "Required — identify and respond"

  contingency_plan:
    - data_backup: "Required — retrievable exact copies of ePHI"
    - disaster_recovery: "Required — restore lost data"
    - emergency_mode: "Required — critical operations during crisis"
    - testing: "Addressable — periodic plan testing"
    - criticality_analysis: "Addressable — prioritize systems"

  evaluation:
    - periodic_assessment: "Required — ongoing compliance evaluation"

  baa_contracts:
    - written_contract: "Required for all business associates"
EOF
```

### Physical Safeguards (Section 164.310)
```bash
cat <<'EOF' > hipaa_physical_safeguards.yaml
physical_safeguards:
  facility_access_controls:
    - contingency_operations: "Addressable"
    - facility_security_plan: "Addressable"
    - access_control_validation: "Addressable"
    - maintenance_records: "Addressable"

  workstation_use:
    - policies: "Required — specify proper functions and access"

  workstation_security:
    - physical_restrictions: "Required — restrict access to authorized"

  device_and_media_controls:
    - disposal: "Required — sanitize media before disposal"
    - media_reuse: "Required — remove ePHI before reuse"
    - accountability: "Addressable — track hardware/media"
    - data_backup: "Addressable — backup before moving equipment"
EOF
```

### Technical Safeguards (Section 164.312)
```bash
cat <<'EOF' > hipaa_technical_safeguards.yaml
technical_safeguards:
  access_control:
    - unique_user_id: "Required — assign unique identifier"
    - emergency_access: "Required — obtain ePHI in emergency"
    - automatic_logoff: "Addressable — terminate idle sessions"
    - encryption_decryption: "Addressable — encrypt ePHI at rest"

  audit_controls:
    - audit_logs: "Required — record and examine access"

  integrity:
    - mechanism_to_authenticate: "Addressable — verify ePHI not altered"

  person_or_entity_authentication:
    - authentication: "Required — verify identity before access"

  transmission_security:
    - integrity_controls: "Addressable — protect ePHI in transit"
    - encryption: "Addressable — encrypt ePHI in transit"
EOF
```

## Required vs Addressable Implementation
```
Required: Must be implemented as specified. No exceptions.

Addressable: Must perform one of the following:
  1. Implement the specification as written
  2. Implement an equivalent alternative measure
  3. Document why it is not reasonable and appropriate
     (this is NOT optional — you must still protect ePHI)

Common mistake: "Addressable" does NOT mean "optional."
Every addressable specification MUST be evaluated and documented.
```

## Breach Notification Rule (45 CFR Sections 164.400-414)
### Notification Requirements
```bash
# Breach notification workflow and timelines
cat <<'EOF' > breach_notification.yaml
definition:
  breach: "Unauthorized acquisition, access, use, or disclosure
           of unsecured PHI that compromises security or privacy"
  exceptions:
    - unintentional_good_faith_access_by_workforce
    - inadvertent_disclosure_within_same_organization
    - good_faith_belief_recipient_cannot_retain_info

risk_assessment_factors:
  1. nature_and_extent_of_PHI: "Types of identifiers and likelihood of re-identification"
  2. unauthorized_person_involved: "Who accessed or received the PHI"
  3. was_PHI_actually_acquired_or_viewed: "Whether data was accessed"
  4. extent_of_risk_mitigation: "Steps taken to reduce harm"

notification_timelines:
  individuals:
    method: "First-class mail or email (if consented)"
    deadline: "Without unreasonable delay, no later than 60 days"
    content:
      - description_of_breach
      - types_of_information_involved
      - steps_to_protect_from_harm
      - what_entity_is_doing_to_investigate
      - contact_information

  hhs_secretary:
    fewer_than_500: "Annual log submitted within 60 days of year end"
    500_or_more: "Without unreasonable delay, no later than 60 days"

  media:
    trigger: "500+ individuals in a single state or jurisdiction"
    deadline: "Without unreasonable delay, no later than 60 days"
    method: "Prominent media outlet in the state/jurisdiction"

unsecured_PHI:
  note: "PHI rendered unusable via encryption (NIST standards)
         or destruction is NOT unsecured and does NOT trigger
         breach notification requirements"
EOF
```

## Business Associate Agreements (BAA)
### Required Elements
```
BAA must contractually require the business associate to:

  1. Use/disclose PHI only as permitted by the agreement
  2. Implement appropriate safeguards
  3. Report breaches and security incidents
  4. Ensure subcontractors agree to same restrictions
  5. Make PHI available for patient access rights
  6. Make PHI available for amendment
  7. Provide accounting of disclosures
  8. Make internal practices available to HHS
  9. Return or destroy PHI on termination
 10. Authorize termination if BA violates terms

Business Associate examples:
  - Cloud service providers storing ePHI
  - IT contractors with ePHI access
  - Billing companies
  - EHR vendors
  - Shredding companies
  - Consultants reviewing PHI
  - Attorneys with PHI access

NOT business Associates:
  - Conduits (USPS, ISPs, couriers) with transient access
  - Covered entity workforce members
  - Plan sponsors under group health plan
```

## HITECH Act Enhancements
```
HITECH Act (2009) + Omnibus Rule (2013):
  - Extended Security Rule directly to business associates
  - Established breach notification requirements
  - Increased civil penalties (tiered structure)
  - Authorized state attorneys general to enforce HIPAA
  - Modified breach standard to "low probability of compromise"
  - Expanded business associate definition
  - Genetic information treated as PHI under HIPAA
```

## Penalty Structure
### Civil Money Penalties (CMPs)
```
Tier 1 — Did Not Know:
  Per violation: $100 - $50,000
  Annual cap:   $25,000 (same provision)

Tier 2 — Reasonable Cause:
  Per violation: $1,000 - $50,000
  Annual cap:   $100,000 (same provision)

Tier 3 — Willful Neglect (Corrected):
  Per violation: $10,000 - $50,000
  Annual cap:   $250,000 (same provision)

Tier 4 — Willful Neglect (Not Corrected):
  Per violation: $50,000
  Annual cap:   $1,500,000 (same provision)

Note: Inflation adjustments apply. As of 2024, maximums are higher.
      State attorneys general can also pursue actions (HITECH).

Criminal Penalties (DOJ prosecution):
  Tier 1 — Knowingly obtain/disclose: Up to $50K + 1 year
  Tier 2 — Under false pretenses:     Up to $100K + 5 years
  Tier 3 — For profit/malicious harm: Up to $250K + 10 years
```

## Risk Assessment (Section 164.308(a)(1)(ii)(A))
```
10-step process: 1) Scope all ePHI systems, 2) Document data flows,
3) Identify threats, 4) Identify vulnerabilities, 5) Document controls,
6) Assess likelihood, 7) Assess impact, 8) Assign risk levels,
9) Document everything, 10) Review periodically

Common OCR audit findings:
  - No risk analysis conducted at all
  - Incomplete scope (missed systems)
  - No follow-up risk management plan
  - No documentation of addressable decisions
```

## Tips
- Conduct a thorough risk analysis annually and document it completely; the most common OCR audit finding is an absent or incomplete risk analysis
- Treat "addressable" specifications as "required unless you document why not" and always implement an equivalent alternative if the specification itself is not adopted
- Establish a formal BAA with every vendor, cloud provider, and subcontractor that could access ePHI, even if their access seems incidental
- Encrypt all ePHI at rest and in transit using NIST-approved standards; encrypted data is "secured" and exempt from breach notification
- Implement automatic session timeout and unique user IDs across all systems that access ePHI to satisfy technical safeguard requirements
- Train all workforce members on HIPAA policies within 30 days of hire and retrain annually, documenting all sessions
- Maintain audit logs of all access to ePHI for at least six years (the HIPAA documentation retention requirement) and review them regularly
- Apply the minimum necessary standard rigorously: role-based access controls should limit each user to only the PHI required for their job function
- Develop and test your contingency plan (backup, disaster recovery, emergency mode) at least annually with documented results
- When a breach occurs, perform the four-factor risk assessment immediately and document the determination whether notification is required
- Keep a current inventory of all systems and devices that store, process, or transmit ePHI, including mobile devices and removable media

## See Also
- gdpr, soc2, nist, fedramp, pci-dss, iso27001

## References
- [HHS HIPAA Home Page](https://www.hhs.gov/hipaa/index.html)
- [HIPAA Privacy Rule (45 CFR Part 164 Subpart E)](https://www.ecfr.gov/current/title-45/subtitle-A/subchapter-C/part-164/subpart-E)
- [HIPAA Security Rule (45 CFR Part 164 Subpart C)](https://www.ecfr.gov/current/title-45/subtitle-A/subchapter-C/part-164/subpart-C)
- [HHS Breach Notification Rule](https://www.hhs.gov/hipaa/for-professionals/breach-notification/index.html)
- [NIST SP 800-66 Rev. 2 — HIPAA Security Rule Implementation Guide](https://csrc.nist.gov/publications/detail/sp/800-66/rev-2/final)
- [HHS Guidance on Encryption](https://www.hhs.gov/hipaa/for-professionals/breach-notification/guidance/index.html)
- [OCR Audit Protocol](https://www.hhs.gov/hipaa/for-professionals/compliance-enforcement/audit/protocol/index.html)
- [HITECH Act Text](https://www.congress.gov/111/plaws/publ5/PLAW-111publ5.htm)
