# AI Compliance

AI Compliance encompasses the regulatory frameworks, standards, and organizational practices required to develop, deploy, and operate artificial intelligence systems lawfully, covering the EU AI Act's risk-based classification system, emerging global AI regulations, sector-specific requirements, and international standards like ISO/IEC 42001 for AI management systems.

## EU AI Act
### Risk Categories
```
┌─────────────────────────────────────────────────────────────┐
│                    EU AI Act Risk Pyramid                    │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│                    ┌───────────┐                            │
│                    │PROHIBITED │ ← Article 5                │
│                    │  (Banned) │   Effective: Feb 2025      │
│                    └─────┬─────┘                            │
│                  ┌───────┴────────┐                         │
│                  │   HIGH RISK    │ ← Annex III + Art. 6    │
│                  │ (Strict Rules) │   Effective: Aug 2026   │
│                  └───────┬────────┘                         │
│             ┌────────────┴─────────────┐                    │
│             │      LIMITED RISK        │ ← Art. 50          │
│             │ (Transparency Obligations)│  Effective: Aug 2026│
│             └────────────┬─────────────┘                    │
│        ┌─────────────────┴──────────────────┐               │
│        │          MINIMAL RISK              │ ← No specific │
│        │      (No obligations, voluntary)    │   obligations │
│        └────────────────────────────────────┘               │
└─────────────────────────────────────────────────────────────┘
```

### Prohibited AI Practices (Article 5)
```
Banned outright (no exceptions unless noted):

1. Social Scoring by Public Authorities
   - Government evaluation of citizens based on social behavior
   - Leading to detrimental treatment in unrelated contexts

2. Subliminal / Manipulative / Deceptive Techniques
   - AI that manipulates beyond conscious awareness
   - Causing significant harm to the person or others

3. Exploitation of Vulnerabilities
   - Targeting age, disability, social/economic situation
   - To materially distort behavior causing significant harm

4. Real-Time Remote Biometric Identification (Public Spaces)
   - Law enforcement use in publicly accessible spaces
   - Exceptions: targeted victim search, imminent terrorist threat,
     serious criminal suspects (requires judicial authorization)

5. Biometric Categorization (Sensitive Attributes)
   - Inferring race, political opinions, trade union membership,
     religious beliefs, sex life, sexual orientation
   - Exception: labeling/filtering lawfully acquired datasets

6. Untargeted Facial Recognition Scraping
   - Building facial recognition databases from internet/CCTV scraping

7. Emotion Recognition in Workplace/Education
   - Except for medical or safety reasons

8. Predictive Policing (Individual)
   - Based solely on profiling or personality traits
   - Exception: augmenting human assessment based on objective facts
```

### High-Risk AI Systems (Annex III)
```
Category 1: Biometrics
  - Remote biometric identification (non-real-time)
  - Biometric categorization (non-prohibited)
  - Emotion recognition (non-prohibited contexts)

Category 2: Critical Infrastructure
  - Safety components of critical infrastructure
  - Road traffic, water/gas/heating/electricity supply

Category 3: Education & Vocational Training
  - Admission/assignment to educational institutions
  - Assessment of learning outcomes
  - Monitoring of prohibited behavior during exams

Category 4: Employment & Workers Management
  - Recruitment: CV screening, interview evaluation
  - Promotion, termination, task allocation decisions
  - Performance monitoring and evaluation

Category 5: Essential Services Access
  - Credit scoring / creditworthiness assessment
  - Life and health insurance risk assessment
  - Emergency services dispatch prioritization

Category 6: Law Enforcement
  - Individual risk assessment (recidivism)
  - Polygraphs and similar tools
  - Evidence reliability assessment
  - Profiling during investigations

Category 7: Migration, Asylum, Border Control
  - Risk assessment for irregular migration
  - Application processing assistance
  - Identification of persons

Category 8: Justice & Democratic Processes
  - Assisting judicial authorities in fact-finding
  - Influencing outcome of elections/referendums
```

### High-Risk Requirements (Articles 8-15)
```
Requirement                    Article  Key Obligations
──────────────────────────────────────────────────────────────
Risk Management System         Art. 9   Continuous lifecycle risk mgmt
                                        Identify, analyze, evaluate,
                                        treat risks. Residual risk
                                        must be acceptable.

Data Governance                Art. 10  Training data must be relevant,
                                        representative, free of errors.
                                        Bias examination required.
                                        Privacy-compliant datasets.

Technical Documentation        Art. 11  Detailed system description,
                                        design choices, training data,
                                        testing methodology, performance
                                        metrics. Before market placement.

Record-Keeping                 Art. 12  Automatic event logging.
                                        Traceability throughout lifecycle.
                                        Logs retained per applicable law.

Transparency & Information     Art. 13  Instructions for use to deployers.
                                        Capabilities and limitations.
                                        Intended purpose clearly stated.

Human Oversight                Art. 14  Designed for effective human
                                        oversight. Ability to intervene,
                                        override, or stop the system.

Accuracy, Robustness, Security Art. 15  Appropriate accuracy levels.
                                        Resilient to errors/faults.
                                        Cybersecurity measures.
```

### Transparency Obligations (Article 50)
```
All AI Systems (Limited Risk):
  ├─ AI-generated content must be marked as such
  │   (deepfakes, synthetic text, synthetic audio/video)
  ├─ Users must be informed they are interacting with AI
  │   (chatbots, emotion recognition systems)
  └─ AI-generated content must be machine-detectable
      (watermarking, metadata, C2PA)

Deployers of High-Risk AI:
  ├─ Must inform individuals subject to AI decisions
  ├─ Must provide meaningful explanation of the decision
  └─ Must inform of right to human review
```

### Conformity Assessment (Article 43)
```
Path A: Internal Control (Self-Assessment)
  - Available for most Annex III high-risk systems
  - Provider conducts own conformity assessment
  - Documents compliance with all Chapter III requirements
  - Affixes CE marking and registers in EU database

Path B: Third-Party Assessment (Notified Body)
  - Required for biometric identification systems (Category 1)
  - Required for critical infrastructure safety components
  - Notified Body reviews technical documentation
  - May conduct testing and auditing
  - Issues certificate of conformity

Steps for Conformity Assessment:
  1. Establish quality management system (Art. 17)
  2. Prepare technical documentation (Art. 11)
  3. Conduct risk management (Art. 9)
  4. Verify data governance (Art. 10)
  5. Test accuracy, robustness, cybersecurity (Art. 15)
  6. Ensure human oversight mechanisms (Art. 14)
  7. Perform conformity assessment (internal or third-party)
  8. Draw up EU declaration of conformity
  9. Affix CE marking
  10. Register in EU database (Art. 49)
```

### EU AI Act Fines (Article 99)
```
Violation Type                              Maximum Fine
────────────────────────────────────────────────────────────
Prohibited AI practices (Art. 5)            €35M or 7% global
                                            annual turnover

High-risk non-compliance                    €15M or 3% global
(Arts. 8-15 requirements)                   annual turnover

Incorrect information to authorities        €7.5M or 1.5% global
                                            annual turnover

SME/Startup reduced fines:
  - Fines capped at lower of amount or %
  - Proportionality principle applies

Factors in determining fines:
  - Nature, gravity, duration of infringement
  - Intentional or negligent character
  - Actions taken to mitigate harm
  - Degree of responsibility / technical measures
  - Previous infringements
  - Degree of cooperation with authorities
  - Manner in which infringement became known
```

## Global AI Regulatory Landscape
### Jurisdiction Comparison
```
┌────────────┬──────────────┬──────────────┬──────────────────┐
│ Jurisdiction│ Approach     │ Key Law/     │ Status           │
│            │              │ Framework    │                  │
├────────────┼──────────────┼──────────────┼──────────────────┤
│ EU         │ Risk-based   │ AI Act       │ In force (phased │
│            │ regulation   │ Reg. 2024/   │ 2025-2027)       │
│            │              │ 1689         │                  │
├────────────┼──────────────┼──────────────┼──────────────────┤
│ US         │ Sector-      │ EO 14110     │ Active (Biden    │
│            │ specific +   │ (Oct 2023)   │ EO; Congress     │
│            │ Executive    │ NIST AI RMF  │ considering      │
│            │ action       │ State laws   │ legislation)     │
├────────────┼──────────────┼──────────────┼──────────────────┤
│ UK         │ Pro-         │ AI Regulation│ White paper      │
│            │ innovation,  │ White Paper  │ (2023), sector   │
│            │ sector-led   │ (2023)       │ regulators lead  │
├────────────┼──────────────┼──────────────┼──────────────────┤
│ Canada     │ Rights-      │ AIDA (C-27)  │ Proposed (died   │
│            │ based        │              │ on order paper)  │
├────────────┼──────────────┼──────────────┼──────────────────┤
│ China      │ Use-case     │ Algorithmic  │ In force (2022+) │
│            │ specific     │ Recomm. Reg. │ Deep synthesis,  │
│            │ regulation   │ Deep Synth.  │ generative AI    │
│            │              │ GenAI Reg.   │ regulations      │
├────────────┼──────────────┼──────────────┼──────────────────┤
│ Singapore  │ Voluntary    │ Model AI Gov.│ Framework v2     │
│            │ governance   │ Framework    │ (2020), AI Verify│
│            │ framework    │ AI Verify    │ testing tool     │
├────────────┼──────────────┼──────────────┼──────────────────┤
│ Brazil     │ Rights-      │ PL 2338/2023 │ Under            │
│            │ based        │              │ consideration    │
├────────────┼──────────────┼──────────────┼──────────────────┤
│ Japan      │ Soft law     │ AI Guidelines│ Voluntary,       │
│            │              │ Social       │ industry-led     │
│            │              │ Principles   │                  │
└────────────┴──────────────┴──────────────┴──────────────────┘
```

### US AI Executive Order 14110 (Oct 2023)
```
Key Requirements:
  Safety & Security:
    ├─ Dual-use foundation models: safety testing before release
    ├─ Red-teaming requirements for large models
    ├─ Report training runs using > 10^26 FLOP (or 10^23 for bio)
    ├─ NIST to develop AI safety guidelines and standards
    └─ Watermarking / content authentication for AI-generated content

  Civil Rights & Equity:
    ├─ Prevent AI discrimination in housing, employment, credit
    ├─ Algorithmic discrimination protections
    └─ Bias testing in federal AI use

  Privacy:
    ├─ Support privacy-preserving techniques
    ├─ Federal data minimization
    └─ Evaluate commercial data broker practices

  Federal Government AI Use:
    ├─ Chief AI Officers in each agency
    ├─ AI governance boards
    ├─ Inventory of AI use cases
    └─ Impact assessments for rights-impacting AI

  Workforce & Innovation:
    ├─ AI talent immigration reform
    ├─ AI R&D funding
    └─ Small business AI support
```

### China AI Regulations
```
Algorithmic Recommendation Regulation (2022):
  - Transparency: Users must be informed of algorithmic recommendations
  - User control: Opt-out of personalized recommendations
  - No price discrimination via algorithms
  - Algorithm filing with CAC (Cyberspace Administration of China)

Deep Synthesis Regulation (2023):
  - Deepfake/synthetic content must be labeled
  - Provider must verify real identity of users
  - Watermarking requirements
  - Content moderation obligations

Generative AI Regulation (2023):
  - Training data must be lawful
  - Must respect IP and personal data rights
  - Content must adhere to socialist core values
  - Security assessment before public release
  - Filing with CAC for public-facing GenAI services
  - User real-name verification
```

## Audit Readiness Checklist
### Pre-Audit Preparation
```
Documentation:
  □ AI system inventory with risk classification
  □ Technical documentation per Art. 11 (high-risk)
  □ Data governance documentation (data sources, quality, bias analysis)
  □ Risk management records (assessments, mitigations, residual risks)
  □ Model cards for all production models
  □ Training/validation/test dataset documentation
  □ Human oversight procedures and escalation paths
  □ Incident response plan specific to AI
  □ Change management records for model updates
  □ Third-party AI vendor assessments

Governance:
  □ AI ethics policy approved by board/leadership
  □ Roles and responsibilities documented (RACI)
  □ AI ethics committee / review board minutes
  □ Staff AI training records
  □ Whistleblower/reporting mechanisms for AI concerns

Technical:
  □ Model performance metrics (current + historical)
  □ Bias/fairness testing results
  □ Adversarial robustness test results
  □ Data drift monitoring reports
  □ Security assessment reports
  □ Privacy impact assessments
  □ Explainability/interpretability outputs
  □ Automated logging and audit trail evidence

Compliance Mapping:
  □ Regulation → control mapping matrix
  □ Gap analysis against applicable regulations
  □ Remediation plan with timelines
  □ Previous audit findings and closure evidence
```

## Continuous Compliance Monitoring
### Monitoring Framework
```
Real-Time Monitoring:
  ┌─ Model performance vs. declared accuracy (Art. 15)
  ├─ Fairness metrics vs. thresholds (bias detection)
  ├─ Input/output logging completeness (Art. 12)
  ├─ Human oversight trigger rates (Art. 14)
  └─ Security event monitoring (Art. 15)

Periodic Reviews:
  ┌─ Monthly: Drift analysis, performance trends
  ├─ Quarterly: Bias audit, risk reassessment
  ├─ Semi-annual: Regulatory horizon scan, gap analysis
  ├─ Annual: Full conformity re-assessment
  └─ Triggered: Post-incident, post-update, regulation change

Automated Compliance Checks:
  ┌─ CI/CD pipeline gates for model deployment
  │   ├─ Fairness threshold check
  │   ├─ Performance regression check
  │   ├─ Documentation completeness check
  │   └─ Security scan pass
  ├─ Continuous logging validation
  ├─ Data governance policy enforcement
  └─ Automated regulatory change tracking
```

## AI Documentation Requirements
### Model Card Template (Based on Mitchell et al.)
```markdown
# Model Card: [Model Name]

## Model Details
- Developer: [Organization]
- Model Type: [Architecture]
- Version: [X.Y.Z]
- Date: [YYYY-MM-DD]
- License: [License Type]
- Contact: [Email]

## Intended Use
- Primary Use Cases: [List]
- Out-of-Scope Uses: [List]
- Users: [Intended audience]

## Training Data
- Dataset: [Name, size, source]
- Preprocessing: [Steps applied]
- Known Limitations: [Biases, gaps]

## Evaluation Data
- Dataset: [Name, size, source]
- Motivation: [Why this dataset]

## Performance Metrics
| Metric | Overall | Group A | Group B | Group C |
|--------|---------|---------|---------|---------|
| Accuracy | X% | X% | X% | X% |
| F1 Score | X | X | X | X |
| FPR | X% | X% | X% | X% |

## Ethical Considerations
- [Bias analysis results]
- [Fairness constraints applied]
- [Potential harms identified]

## Limitations and Risks
- [Known failure modes]
- [Distribution shift sensitivity]
- [Adversarial vulnerability assessment]
```

## Sector-Specific AI Regulation
### Healthcare AI
```
US (FDA):
  - AI/ML-based SaMD (Software as Medical Device)
  - 510(k), De Novo, or PMA pathway depending on risk
  - Predetermined Change Control Plan for adaptive AI
  - Good Machine Learning Practice (GMLP) principles
  - Real-world performance monitoring post-market

EU (MDR + AI Act):
  - Medical device AI classified under MDR 2017/745
  - Additionally subject to EU AI Act high-risk requirements
  - Notified Body assessment for Class IIb+ devices
  - Clinical evaluation including AI-specific considerations
  - Post-market surveillance with AI monitoring

Key Compliance Points:
  □ Clinical validation (not just technical validation)
  □ Intended use clearly scoped
  □ Human oversight by qualified clinician
  □ Explainability appropriate for clinical context
  □ Bias testing across patient demographics
  □ Adverse event reporting for AI failures
```

### Financial Services AI
```
US:
  - Fair lending laws (ECOA, FHA) apply to AI credit decisions
  - SR 11-7 Model Risk Management (Fed/OCC)
  - CFPB enforcement on algorithmic discrimination
  - SEC/FINRA guidance on AI in trading

EU:
  - AI Act high-risk for credit scoring
  - DORA (Digital Operational Resilience Act) covers AI
  - EBA guidelines on ML in credit risk
  - MiFID II suitability for AI-driven advice

Key Compliance Points:
  □ Adverse action notice with specific reasons (not "AI said no")
  □ Disparate impact testing across protected classes
  □ Model Risk Management (MRM) framework with 3 lines of defense
  □ Explainability sufficient for regulators and consumers
  □ Stress testing AI models under economic scenarios
  □ Ongoing monitoring for fair lending compliance
```

## ISO/IEC 42001 — AI Management System
### AIMS Structure
```
ISO/IEC 42001:2023 — AI Management System (AIMS)

Clause 4: Context of the Organization
  - Interested parties and their AI-related needs
  - Scope of the AIMS
  - AI system lifecycle considerations

Clause 5: Leadership
  - AI policy statement
  - Roles, responsibilities, and authorities
  - Management commitment to responsible AI

Clause 6: Planning
  - AI risk assessment and treatment
  - AI objectives and planning to achieve them
  - AI impact assessment methodology

Clause 7: Support
  - Resources (compute, data, expertise)
  - Competence and awareness
  - Communication
  - Documented information management

Clause 8: Operation
  - Operational planning and control
  - AI risk assessment execution
  - AI system development lifecycle
  - Data management
  - Third-party and supplier management

Clause 9: Performance Evaluation
  - Monitoring, measurement, analysis
  - Internal audit
  - Management review

Clause 10: Improvement
  - Nonconformity and corrective action
  - Continual improvement

Annex A: AI Controls (39 controls in 9 domains)
Annex B: Implementation guidance
```

### Annex A Control Domains
```
A.2  AI Policies                  (3 controls)
A.3  Internal Organization        (3 controls)
A.4  Resources for AI Systems     (5 controls)
A.5  Assessing Impacts of AI      (4 controls)
A.6  AI System Lifecycle          (9 controls)
A.7  Data for AI Systems          (5 controls)
A.8  Information for Interested   (4 controls)
     Parties
A.9  Use of AI Systems            (4 controls)
A.10 Third-Party & Customer       (2 controls)
     Relationships
```

## See Also
- ai-risk-management
- gdpr
- nist
- soc2
- iso27001
- ai-security-architecture

## References
- EU AI Act (Regulation 2024/1689): https://eur-lex.europa.eu/eli/reg/2024/1689
- NIST AI RMF: https://airc.nist.gov/AI_RMF
- ISO/IEC 42001:2023: https://www.iso.org/standard/81230.html
- US Executive Order 14110: https://www.whitehouse.gov/briefing-room/presidential-actions/2023/10/30/executive-order-on-ai/
- Singapore Model AI Governance Framework: https://www.pdpc.gov.sg/help-and-resources/2020/01/model-ai-governance-framework
