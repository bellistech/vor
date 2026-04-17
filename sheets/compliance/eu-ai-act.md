# EU AI Act (Regulation (EU) 2024/1689)

The EU AI Act is the world's first comprehensive, horizontal regulation of artificial intelligence, establishing a risk-based framework across prohibited, high-risk, limited-risk, and minimal-risk categories, with specific rules for general-purpose AI (GPAI) models and phased enforcement from 2 February 2025 through 2 August 2027.

## Timeline of Application

### Phased Enforcement
```
02 Aug 2024  Entry into force
02 Feb 2025  Prohibitions (Art. 5) + AI literacy (Art. 4) apply
02 Aug 2025  Governance, GPAI rules, penalties, notified bodies
02 Aug 2026  Full application (most high-risk obligations)
02 Aug 2027  Annex I (product safety) high-risk systems apply
```

## Risk Categories

### Prohibited AI Practices (Art. 5)
```
- Subliminal, manipulative, or deceptive techniques causing harm
- Exploitation of vulnerabilities (age, disability, socio-economic)
- Social scoring by public authorities
- Predictive policing based solely on profiling
- Untargeted scraping of facial images for FR databases
- Emotion inference in workplace/education (except medical/safety)
- Biometric categorization by protected characteristics
- Real-time remote biometric identification in public spaces
  (narrow LE exceptions with judicial authorization)
```

### High-Risk AI Systems (Art. 6, Annex III)
```
Annex III use cases:
  1. Biometrics (categorization, emotion recognition)
  2. Critical infrastructure (water, gas, electricity, transport)
  3. Education and vocational training (grading, admissions)
  4. Employment and worker management (hiring, promotions, firing)
  5. Essential private and public services (credit scoring, benefits, triage)
  6. Law enforcement (risk assessments, evidence evaluation)
  7. Migration, asylum, border control
  8. Administration of justice and democratic processes

Annex I: AI as a safety component of regulated products (MDR, machinery,
toys, automotive — already subject to CE marking)
```

### Limited-Risk (Transparency, Art. 50)
```
- Chatbots must disclose AI interaction
- AI-generated content (deepfakes) must be labeled
- Emotion recognition / biometric categorization notification
- AI-generated text on matters of public interest disclosed
  (unless human-reviewed with editorial responsibility)
```

### Minimal-Risk
No obligations; voluntary codes of conduct encouraged.

## High-Risk System Obligations

### Provider Obligations (Art. 8-15)
```
1. Risk management system (continuous, iterative — Art. 9)
2. Data governance — relevant, representative, free of errors (Art. 10)
3. Technical documentation (Annex IV — Art. 11)
4. Record-keeping / logging (Art. 12)
5. Transparency to deployers (instructions for use — Art. 13)
6. Human oversight design (Art. 14)
7. Accuracy, robustness, cybersecurity (Art. 15)
8. Quality management system (Art. 17)
9. Post-market monitoring (Art. 72)
10. Incident reporting (Art. 73)
11. Cooperation with authorities
12. CE marking and conformity assessment (Art. 43)
13. EU database registration (Art. 71)
```

### Deployer Obligations (Art. 26)
```
- Use in accordance with instructions
- Human oversight by competent persons
- Input data representative of intended purpose
- Monitor operation and log events
- Inform provider of serious incidents
- DPIA under GDPR where applicable
- FRIA for public bodies + certain private deployers (Art. 27)
- Transparency to affected persons
```

### Fundamental Rights Impact Assessment (FRIA, Art. 27)
```
Required for:
  - Public authorities or bodies using high-risk AI
  - Private entities in essential services (banking, insurance)

FRIA content:
  - Purpose and context
  - Duration and frequency of use
  - Categories of natural persons affected
  - Specific risks of harm
  - Human oversight measures
  - Mitigation if risks materialize
```

## General-Purpose AI (GPAI) Models

### All GPAI Providers (Art. 53)
```
- Technical documentation (Annex XI)
- Information/documentation for downstream deployers (Annex XII)
- Copyright compliance policy (text and data mining)
- Summary of training data published
```

### GPAI with Systemic Risk (Art. 55)
```
Threshold: cumulative training compute > 10^25 FLOP (adjustable)

Additional obligations:
  - Model evaluation including adversarial testing
  - Assessment and mitigation of systemic risks
  - Serious incident tracking and reporting
  - Adequate cybersecurity protection (model + infra)
```

## Conformity Assessment

### Route Selection (Art. 43)
```
Annex III use cases (most):
  Internal conformity assessment
  Provider self-assesses against harmonized standards
  No notified body unless derogation

Annex I regulated products:
  Follow sectoral conformity route
  Often requires notified body

Biometric identification (narrow cases):
  Notified body required

Harmonized standards (CEN-CENELEC JTC 21):
  ISO/IEC 42001 (AI management)
  Risk management, data quality, robustness standards
  Presumption of conformity when applied
```

### CE Marking Workflow
```bash
# Conformity assessment checklist for Annex III high-risk AI
cat <<'EOF' > ce_marking_checklist.yaml
step_1_classification:
  - is_ai_system: confirm per Art. 3 definition
  - risk_category: [prohibited, high_risk, limited_risk, minimal]
  - annex_applicable: [Annex_I, Annex_III, none]

step_2_risk_management:
  - risk_management_system_established
  - hazard_identification_continuous
  - residual_risk_acceptable
  - testing_throughout_lifecycle

step_3_data_governance:
  - training_validation_testing_sets_documented
  - relevance_and_representativeness_assessed
  - bias_detection_and_mitigation
  - error_correction_procedures

step_4_technical_documentation:
  - system_description
  - design_specifications
  - development_process
  - validation_and_testing_procedures

step_5_logging:
  - automatic_recording_of_events
  - retention_period_appropriate
  - traceability_of_inputs_and_outputs

step_6_oversight:
  - human_oversight_design
  - ability_to_override
  - stop_button_equivalent

step_7_accuracy_robustness:
  - accuracy_metrics_declared
  - cybersecurity_measures_documented
  - resilience_to_errors_and_attacks

step_8_qms:
  - quality_management_system
  - change_management
  - post_market_monitoring_plan

step_9_conformity_declaration:
  - EU_declaration_of_conformity_signed
  - CE_mark_affixed
  - EU_database_registration (Art. 71)
EOF
```

## Transparency Requirements (Art. 50)

### Content Labeling
```python
# Synthetic content watermarking example (provider obligation)
from hashlib import sha256

def embed_ai_provenance(output, model_id, timestamp, prompt_hash):
    # Machine-readable marking (C2PA-compliant claim)
    provenance = {
        "ai_generated": True,
        "model": model_id,
        "timestamp": timestamp,
        "prompt_fingerprint": sha256(prompt_hash.encode()).hexdigest(),
        "art_50_eu_ai_act": True,
    }
    # Deployer must preserve marking when distributing output
    return {"content": output, "c2pa_manifest": provenance}

# Deployer obligation on deepfakes: "clearly and distinguishably"
# disclose content is artificially generated, unless:
# - evidently artistic, creative, satirical, fictional work
# - appropriate disclosure then does not hamper enjoyment
```

## Penalties (Art. 99)

### Fines
```
Prohibited practices (Art. 5):
  Up to €35,000,000 or 7% global turnover (higher)

High-risk system violations (obligations under Title III):
  Up to €15,000,000 or 3% global turnover

Incorrect/misleading info to authorities:
  Up to €7,500,000 or 1% global turnover

SMEs and start-ups: lower of the two amounts applies
EU institutions: up to €1,500,000 (prohibited) or €750,000 (other)
```

## Codes of Practice and Sandboxes

```
GPAI Code of Practice (Art. 56):
  Voluntary code convened by AI Office
  Demonstrates compliance until harmonized standards available
  First version expected 2025

AI regulatory sandboxes (Art. 57):
  Each member state establishes at least one
  Enables controlled testing of innovative AI
  Supervision-guided experimentation
  Priority access for SMEs and start-ups

Real-world testing (Art. 60):
  Before placing on market / putting into service
  Registration and ethics committee review
  Informed consent from subjects
```

## Tips
- Treat risk classification as the first engineering decision — a system that qualifies as high-risk mid-project needs retrofitted conformity assessment, which is prohibitively expensive
- Start building the Annex IV technical documentation from the first sprint; it is trivially linkable from ADRs, design docs, and model cards if you plan for it
- Log model inputs, outputs, and oversight interventions with tamper-evident storage (append-only WORM or signed logs) — Art. 12 is specific about integrity
- FRIA and GDPR DPIA overlap — build a single unified assessment template and tag sections by regulation to avoid duplicate work
- The 10^25 FLOP GPAI threshold is a training-compute ceiling the Commission can adjust; model your compliance posture against both current and next-tier thresholds
- Copyright compliance for training data is an Art. 53 obligation — document your text-and-data-mining opt-out respect policy BEFORE training, not after
- Watermark AI-generated content at generation time using C2PA-compliant provenance claims; retroactive tagging is technically and legally weak
- For chatbots, the Art. 50 disclosure must appear at first interaction, not buried in footer terms
- Use AI regulatory sandboxes in your primary member state — the supervision-guided path is faster than seeking post-hoc clarification
- Align your AI quality management system with ISO/IEC 42001 to get presumption-of-conformity against harmonized standards when they ship
- Board governance: the AI Act treats provider/deployer obligations as corporate responsibilities; management body must approve risk policy and oversee post-market monitoring

## See Also
- gdpr, nist, iso27001, dora, ai-governance, ai-risk-management, ai-ethics, ai-security-architecture, ai-privacy-trust, threat-modeling

## References
- [Regulation (EU) 2024/1689 Text](https://eur-lex.europa.eu/eli/reg/2024/1689/oj)
- [European Commission AI Act Page](https://digital-strategy.ec.europa.eu/en/policies/regulatory-framework-ai)
- [AI Office](https://digital-strategy.ec.europa.eu/en/policies/ai-office)
- [ISO/IEC 42001 AI Management System](https://www.iso.org/standard/81230.html)
- [C2PA Content Provenance](https://c2pa.org/specifications/specifications/1.3/specs/C2PA_Specification.html)
- [GPAI Code of Practice](https://digital-strategy.ec.europa.eu/en/policies/ai-code-practice)
