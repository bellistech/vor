# GDPR (General Data Protection Regulation)

The General Data Protection Regulation is the European Union's comprehensive data protection law that grants individuals eight fundamental rights over their personal data, imposes strict obligations on data controllers and processors, and carries penalties of up to 4% of global annual turnover or 20 million euros, whichever is higher.

## Data Subject Rights (Articles 12-22)
### The Eight Rights
```
1. Right of Access (Art. 15)
   - Subject can request copy of all personal data
   - Must respond within 30 days (extendable by 60)
   - Free for first request; reasonable fee for subsequent

2. Right to Rectification (Art. 16)
   - Correct inaccurate data without undue delay
   - Complete incomplete data

3. Right to Erasure / Right to be Forgotten (Art. 17)
   - Delete data when no longer necessary
   - Exceptions: legal obligations, public interest, archiving

4. Right to Restriction of Processing (Art. 18)
   - Mark data for limited processing only
   - Applied while accuracy contested or objection pending

5. Right to Data Portability (Art. 20)
   - Receive data in structured, machine-readable format
   - Transmit to another controller without hindrance

6. Right to Object (Art. 21)
   - Object to processing based on legitimate interest
   - Must stop unless compelling grounds override
   - Absolute right for direct marketing

7. Rights re: Automated Decision-Making (Art. 22)
   - Right not to be subject to solely automated decisions
   - Includes profiling producing legal effects
   - Must provide human intervention on request

8. Right to be Informed (Arts. 13-14)
   - Transparent privacy notices at collection
   - Purpose, legal basis, retention, recipients, rights
```

## Lawful Basis for Processing (Article 6)
### Six Legal Grounds
```
1. Consent (Art. 6(1)(a))
   - Freely given, specific, informed, unambiguous
   - Must be as easy to withdraw as to give
   - Cannot be bundled with T&Cs
   - Children: parental consent if under 16 (varies by member state)

2. Contractual Necessity (Art. 6(1)(b))
   - Processing necessary to perform a contract
   - Only data strictly required for the service

3. Legal Obligation (Art. 6(1)(c))
   - Tax reporting, anti-money laundering
   - Employment law requirements

4. Vital Interests (Art. 6(1)(d))
   - Life or death situations
   - Rarely applicable in practice

5. Public Interest / Official Authority (Art. 6(1)(e))
   - Government functions, public health
   - Must have basis in law

6. Legitimate Interest (Art. 6(1)(f))
   - Balancing test required (LIA)
   - Most flexible but most challenged
   - Not available to public authorities
```

### Consent Management Implementation
```python
# Django consent tracking model
from django.db import models
from django.utils import timezone

class ConsentRecord(models.Model):
    PURPOSES = [
        ('marketing', 'Marketing Communications'),
        ('analytics', 'Analytics and Personalization'),
        ('third_party', 'Third-Party Data Sharing'),
        ('profiling', 'Automated Profiling'),
    ]

    user = models.ForeignKey('auth.User', on_delete=models.CASCADE)
    purpose = models.CharField(max_length=50, choices=PURPOSES)
    granted = models.BooleanField(default=False)
    timestamp = models.DateTimeField(default=timezone.now)
    ip_address = models.GenericIPAddressField()
    source = models.CharField(max_length=100)  # e.g., "signup_form_v3"
    version = models.CharField(max_length=20)   # privacy policy version
    withdrawn_at = models.DateTimeField(null=True, blank=True)

    class Meta:
        indexes = [
            models.Index(fields=['user', 'purpose']),
            models.Index(fields=['purpose', 'granted']),
        ]

    def withdraw(self):
        self.granted = False
        self.withdrawn_at = timezone.now()
        self.save()
```

## Data Processing Agreement (DPA)
### Required Clauses (Article 28)
```
A compliant DPA must include:

1. Subject matter and duration of processing
2. Nature and purpose of processing
3. Types of personal data processed
4. Categories of data subjects
5. Controller's obligations and rights
6. Processor obligations:
   a. Process only on documented instructions
   b. Ensure confidentiality commitments from personnel
   c. Implement appropriate security measures (Art. 32)
   d. Sub-processor requirements (prior written consent)
   e. Assist with data subject rights
   f. Assist with DPIA obligations
   g. Delete or return data on termination
   h. Make available information for audits
   i. Inform if instruction infringes GDPR
```

## Data Protection Impact Assessment (DPIA)
### When Required (Article 35)
```bash
# DPIA is mandatory when processing:
cat <<'EOF' > dpia_triggers.yaml
mandatory_triggers:
  - systematic_profiling_with_legal_effects: true
  - large_scale_special_category_data: true
  - systematic_public_area_monitoring: true
  - new_technologies_with_high_risk: true
  - automated_decision_making: true
  - large_scale_processing_children_data: true
  - cross_referencing_datasets: true
  - biometric_identification: true
  - genetic_data_processing: true
  - location_tracking_at_scale: true

dpia_template:
  section_1_description:
    - nature_of_processing
    - scope_of_processing
    - context_of_processing
    - purpose_of_processing
  section_2_necessity:
    - lawful_basis
    - proportionality_assessment
    - data_minimization_check
  section_3_risks:
    - risk_to_rights_and_freedoms
    - likelihood_assessment
    - severity_assessment
  section_4_mitigations:
    - technical_measures
    - organizational_measures
    - residual_risk_assessment
  section_5_consultation:
    - dpo_opinion
    - data_subject_views
    - supervisory_authority_if_high_residual
EOF
```

## Breach Notification (Articles 33-34)
### 72-Hour Notification Process
```bash
# Breach response workflow
cat <<'EOF' > breach_response.yaml
phase_1_detection:
  timeline: "Immediate"
  actions:
    - contain_breach
    - preserve_evidence
    - activate_incident_response_team
    - notify_dpo

phase_2_assessment:
  timeline: "Within 24 hours"
  actions:
    - determine_personal_data_affected
    - count_data_subjects_impacted
    - assess_risk_to_rights_and_freedoms
    - classify_severity: [low, medium, high, critical]

phase_3_supervisory_authority:
  timeline: "72 hours from awareness"
  required_info:
    - nature_of_breach
    - categories_and_count_of_subjects
    - categories_and_count_of_records
    - likely_consequences
    - measures_taken_or_proposed
    - dpo_contact_details
  exception: "Not required if unlikely to result in risk to rights"

phase_4_data_subjects:
  timeline: "Without undue delay"
  trigger: "High risk to rights and freedoms"
  required_info:
    - clear_plain_language_description
    - dpo_contact_details
    - likely_consequences
    - measures_taken_to_mitigate
  exceptions:
    - data_rendered_unintelligible_encryption
    - subsequent_measures_eliminate_risk
    - disproportionate_effort_use_public_notice
EOF
```

## Cross-Border Transfer Mechanisms
### Transfer Tools (Chapter V)
```
Adequacy Decisions (Art. 45):
  Countries with adequate protection (as of 2024):
  Andorra, Argentina, Canada (commercial), Faroe Islands,
  Guernsey, Israel, Isle of Man, Japan, Jersey, New Zealand,
  Republic of Korea, Switzerland, UK, Uruguay,
  EU-US Data Privacy Framework

Standard Contractual Clauses (SCCs) (Art. 46(2)(c)):
  - Controller-to-Controller (Module 1)
  - Controller-to-Processor (Module 2)
  - Processor-to-Processor (Module 3)
  - Processor-to-Controller (Module 4)
  - Must conduct Transfer Impact Assessment (TIA)

Binding Corporate Rules (BCRs) (Art. 47):
  - For intra-group transfers
  - Requires DPA approval
  - 12-18 months approval timeline

Derogations (Art. 49) — limited circumstances:
  - Explicit consent (informed of risks)
  - Contract performance
  - Public interest
  - Legal claims
  - Vital interests
```

## DPO Requirements (Articles 37-39)
### When a DPO Is Mandatory
```
Mandatory DPO appointment:
  1. Public authority or body (except courts)
  2. Core activities require large-scale systematic monitoring
  3. Core activities involve large-scale special category data

DPO qualifications:
  - Expert knowledge of data protection law
  - Independent (no conflicts of interest)
  - Reports to highest management level
  - Cannot be dismissed for performing duties
  - Contact details published and notified to DPA
```

## Penalties and Enforcement
### Two-Tier Penalty Structure
```
Tier 1 (Art. 83(4)) — Up to €10M or 2% global turnover:
  - Processor obligations (Art. 25-39)
  - Certification body obligations (Art. 42-43)
  - Monitoring body obligations (Art. 41)

Tier 2 (Art. 83(5)) — Up to €20M or 4% global turnover:
  - Processing principles (Art. 5, 6, 9)
  - Data subject rights (Art. 12-22)
  - International transfers (Art. 44-49)
  - Non-compliance with DPA orders

Factors considered:
  - Nature, gravity, duration of infringement
  - Intentional or negligent character
  - Actions taken to mitigate damage
  - Degree of cooperation with DPA
  - Categories of personal data affected
  - Previous infringements
```

## Technical Implementation
### Data Subject Request API
```python
# FastAPI DSAR endpoint
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from datetime import datetime, timedelta

app = FastAPI()

class DSARRequest(BaseModel):
    email: str
    request_type: str  # access, erasure, portability, rectification
    details: str = ""

@app.post("/api/v1/dsar")
async def submit_dsar(request: DSARRequest):
    deadline = datetime.utcnow() + timedelta(days=30)
    ticket = await create_dsar_ticket(
        subject_email=request.email,
        request_type=request.request_type,
        details=request.details,
        deadline=deadline,
    )
    # Log for accountability (Art. 5(2))
    await audit_log(
        action="dsar_received",
        subject=request.email,
        request_type=request.request_type,
        ticket_id=ticket.id,
    )
    return {
        "ticket_id": ticket.id,
        "deadline": deadline.isoformat(),
        "status": "received"
    }
```

## Tips
- Always document your lawful basis for processing before you start collecting data, not after
- Implement a consent management platform (CMP) that records the exact version of the privacy notice the user agreed to
- Design your data model to support erasure from the start; retrofitting deletion across microservices is extremely costly
- Conduct a Record of Processing Activities (RoPA) inventory annually; it is required under Article 30
- Set up automated breach detection with a 72-hour countdown timer that escalates if no assessment is completed
- Use pseudonymization rather than anonymization where possible, as truly anonymous data is very difficult to achieve
- Include a Transfer Impact Assessment (TIA) for every SCC-based international transfer after Schrems II
- Train all employees who handle personal data, not just technical staff, and document the training
- Build data portability into your API from the start; JSON and CSV export capabilities satisfy Article 20
- Appoint a DPO even if not strictly required; it demonstrates accountability and simplifies DPA interactions

## See Also
- soc2, nist, fedramp, hipaa, ccpa, dora, eu-ai-act

## References
- [GDPR Full Text](https://gdpr-info.eu/)
- [EDPB Guidelines and Recommendations](https://edpb.europa.eu/our-work-tools/general-guidance/guidelines-recommendations-best-practices_en)
- [ICO Guide to GDPR](https://ico.org.uk/for-organisations/guide-to-data-protection/guide-to-the-general-data-protection-regulation-gdpr/)
- [European Commission SCCs](https://commission.europa.eu/law/law-topic/data-protection/international-dimension-data-protection/standard-contractual-clauses-scc_en)
- [CNIL DPIA Guidelines](https://www.cnil.fr/en/privacy-impact-assessment-pia)
