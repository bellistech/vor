# AI Governance (Frameworks, Risk Management, and Lifecycle Controls)

Practical reference for AI governance frameworks, operating models, risk management, model lifecycle governance, documentation requirements, and compliance with emerging AI regulations.

## AI Governance Frameworks

### NIST AI Risk Management Framework (AI RMF 1.0)

```text
Four Core Functions:

  GOVERN -- establish and maintain governance structures
    - Define AI risk management roles and responsibilities
    - Establish organizational AI policies and procedures
    - Align AI risk management with enterprise risk management
    - Foster an organizational culture of responsible AI
    - Engage diverse stakeholders in AI governance

  MAP -- identify and contextualize AI risks
    - Categorize AI systems by risk level and context
    - Document intended purposes and known limitations
    - Identify stakeholders and potential impacts
    - Map data requirements and provenance
    - Assess legal and regulatory requirements

  MEASURE -- analyze and monitor AI risks
    - Quantify identified risks with appropriate metrics
    - Track performance and bias metrics over time
    - Conduct regular model evaluations and audits
    - Implement monitoring for drift and degradation
    - Benchmark against organizational risk tolerances

  MANAGE -- prioritize and respond to AI risks
    - Prioritize risks based on impact and likelihood
    - Plan and implement risk treatment measures
    - Allocate resources for risk management
    - Communicate risks to stakeholders
    - Document decisions and rationale
```

### EU AI Act Risk Classification

```text
Risk Level      Examples                           Requirements
Unacceptable    Social scoring, real-time           Prohibited
                biometric ID in public spaces,
                subliminal manipulation,
                exploitation of vulnerabilities

High-risk       Critical infrastructure, education, Conformity assessment,
                employment, credit scoring,         risk management system,
                law enforcement, migration,         data governance,
                biometric identification            technical documentation,
                                                    human oversight,
                                                    accuracy/robustness

Limited risk    Chatbots, emotion recognition,      Transparency obligations
                deep fakes, AI-generated content    (inform users of AI
                                                    interaction)

Minimal risk    Spam filters, AI in video games,    No specific obligations
                inventory management                (voluntary codes of practice)

General Purpose AI (GPAI):
  All GPAI:     Technical documentation, training data summary,
                copyright compliance, EU representative
  Systemic risk: Safety evaluation, adversarial testing, incident reporting,
                 cybersecurity measures, energy consumption reporting
```

### Other Frameworks

```text
Framework                  Organization    Focus Area
OECD AI Principles         OECD            International policy guidance
IEEE 7000 Series           IEEE            Ethical design processes
ISO/IEC 42001              ISO             AI Management System (AIMS)
ISO/IEC 23894              ISO             AI Risk Management
Singapore FEAT             MAS             Fairness, Ethics, Accountability, Transparency
Canada AIDA                Canada          Algorithmic impact assessment
ALTAI                      EU              Assessment List for Trustworthy AI
Blueprint for AI Bill      White House     Rights-based AI framework
  of Rights (US)
```

## Governance Operating Model

### AI Governance Structure

```text
Board / Executive Level:
  ├── AI Strategy Committee    -- strategic direction, investment priorities
  ├── Chief AI Officer (CAIO)  -- overall AI program ownership
  └── AI Ethics Board          -- ethical review, policy guidance

Management Level:
  ├── AI Governance Office     -- policy, standards, compliance monitoring
  ├── AI Risk Committee        -- risk assessment, tolerance setting
  └── Model Risk Management    -- model validation, ongoing monitoring

Operational Level:
  ├── AI Development Teams     -- build and deploy AI systems
  ├── Data Engineering         -- data quality, lineage, governance
  ├── MLOps / Platform         -- infrastructure, CI/CD for models
  └── AI Audit / Compliance    -- internal audit, regulatory reporting

RACI Matrix:
  Activity                   CAIO  Ethics Board  Dev Team  Risk  Legal
  AI policy definition       A     C             I         C     C
  Model risk assessment      A     C             R         A     C
  Ethical review             C     A             R         I     C
  Model deployment approval  A     C             R         A     I
  Incident response          A     I             R         C     C
  Regulatory reporting       A     I             I         C     A

  R=Responsible, A=Accountable, C=Consulted, I=Informed
```

### AI Ethics Board

```text
Composition:
  - Chief AI Officer or delegate (chair)
  - Legal/compliance representative
  - Data protection / privacy officer
  - Domain subject matter experts (rotating)
  - External ethics advisor(s)
  - Employee representative
  - Community/user representative (for high-impact systems)

Charter:
  Scope:        Review high-risk AI use cases, set ethical guidelines
  Authority:    Advisory (recommend) or binding (approve/reject) -- define clearly
  Frequency:    Monthly meetings + ad-hoc for urgent cases
  Quorum:       Majority of members
  Escalation:   Unresolved issues escalate to executive sponsor
  Independence: Members must declare conflicts of interest

Review Triggers:
  - New AI system deployment (high-risk classification)
  - Material change to existing AI system
  - Ethical concern raised by any stakeholder
  - Adverse outcome or incident involving AI system
  - Regulatory inquiry or audit finding
  - Annual review of all high-risk AI systems
```

## AI Risk Categories

### Risk Taxonomy

```text
Technical Risks:
  Model performance     -- accuracy degradation, concept/data drift
  Robustness            -- adversarial attacks, edge cases, distribution shift
  Reliability           -- inconsistent outputs, hallucinations, confabulation
  Scalability           -- performance degradation at scale
  Security              -- model extraction, data poisoning, prompt injection

Ethical Risks:
  Bias/Fairness         -- discriminatory outcomes across protected groups
  Transparency          -- inability to explain decisions to stakeholders
  Privacy               -- PII leakage, re-identification, inference attacks
  Autonomy              -- undermining human agency or decision-making
  Manipulation          -- using AI to deceive or manipulate behavior

Operational Risks:
  Dependency            -- vendor lock-in, single model dependency
  Data quality          -- training data errors, label noise, staleness
  Integration           -- failure modes in production pipeline
  Human oversight       -- inadequate human-in-the-loop controls
  Skill gap             -- insufficient expertise to manage AI systems

Legal/Regulatory Risks:
  Compliance            -- violation of AI regulations (EU AI Act, sector-specific)
  Liability             -- unclear accountability for AI-caused harm
  IP/Copyright          -- training data copyright, generated content ownership
  Contractual           -- vendor AI terms, data usage restrictions
  Cross-border          -- jurisdictional variation in AI regulation

Societal Risks:
  Job displacement      -- automation of human roles
  Environmental         -- energy consumption, carbon footprint of training
  Information integrity -- deepfakes, misinformation generation
  Power concentration   -- market dominance through AI capabilities
```

## Model Risk Management

### Model Lifecycle

```text
Stage           Governance Controls
Ideation        - Business case with AI justification
                - Ethical pre-screening
                - Regulatory applicability check

Design          - AI Impact Assessment
                - Data requirements and sourcing plan
                - Fairness metrics definition
                - Risk classification (high/limited/minimal)

Development     - Data quality validation
                - Bias testing during training
                - Model documentation (model card)
                - Code review and version control
                - Experiment tracking (MLflow, W&B)

Validation      - Independent model validation (challenger models)
                - Bias and fairness audit
                - Robustness testing (adversarial, edge cases)
                - Performance benchmarking
                - Explainability assessment

Deployment      - Deployment approval (governance gate)
                - Canary/staged rollout
                - Monitoring setup (performance, bias, drift)
                - Human oversight mechanisms
                - Rollback plan documented

Monitoring      - Continuous performance monitoring
                - Drift detection (data and concept drift)
                - Bias monitoring on live data
                - Incident tracking
                - Periodic revalidation (quarterly/annual)

Retirement      - Retirement decision criteria (performance, relevance)
                - Transition plan (replacement model or manual process)
                - Data retention/deletion per policy
                - Documentation archive
                - Stakeholder notification
```

### Model Validation

```text
Three Lines of Defense:
  1st Line: AI development team  -- self-testing, unit tests, integration tests
  2nd Line: Model risk management -- independent validation, bias audit
  3rd Line: Internal audit       -- process compliance, governance effectiveness

Validation Tests:
  Performance:
    - Accuracy, precision, recall, F1 on holdout set
    - Performance across subgroups (sliced evaluation)
    - Comparison to baseline/champion model
    - Out-of-distribution performance

  Fairness:
    - Demographic parity across protected groups
    - Equalized odds analysis
    - Calibration across groups
    - Disparate impact ratio (80% rule / four-fifths rule)

  Robustness:
    - Adversarial example testing
    - Input perturbation sensitivity
    - Missing data handling
    - Edge case behavior

  Explainability:
    - Feature importance analysis (SHAP, LIME)
    - Decision boundary visualization
    - Counterfactual explanations
    - Explanation fidelity testing
```

## AI Inventory / Registry

### AI System Registry

```text
Registry Fields:
  System Identification:
    - System name and unique ID
    - Version number
    - Owner (team and individual)
    - Business unit / department
    - Deployment date

  Classification:
    - Risk level (unacceptable / high / limited / minimal)
    - Use case category (recommendation, decision-support, autonomous)
    - Regulatory applicability (EU AI Act, sector-specific)
    - Data sensitivity level

  Technical Details:
    - Model type (LLM, classification, regression, etc.)
    - Training data description and sources
    - Model architecture summary
    - Infrastructure and deployment platform
    - Third-party components (APIs, models, datasets)

  Governance:
    - Approval status and date
    - Ethics board review status
    - DPIA / AI Impact Assessment status
    - Monitoring plan reference
    - Last validation date
    - Next scheduled review date

  Performance:
    - Key metrics and current values
    - Fairness metrics and thresholds
    - Known limitations and failure modes
    - Incident history

Registry Platforms:
  - Custom internal database/wiki
  - Model registry tools (MLflow Model Registry, Weights & Biases)
  - GRC platforms with AI module (ServiceNow, OneTrust)
  - Dedicated AI governance platforms (Credo AI, Holistic AI, IBM OpenPages)
```

## AI Impact Assessment

### Assessment Framework

```text
AI Impact Assessment (AIA) Template:

Section 1: System Description
  - Purpose and business objective
  - Intended users and affected populations
  - Input data and output decisions
  - Level of automation (advisory, semi-autonomous, autonomous)
  - Deployment context and environment

Section 2: Risk Assessment
  - Risk classification (high/limited/minimal)
  - Identified risks (technical, ethical, legal, societal)
  - Risk likelihood and impact scoring
  - Vulnerable populations affected

Section 3: Fairness Analysis
  - Protected attributes relevant to use case
  - Fairness metrics selected and rationale
  - Bias testing results
  - Mitigation measures for identified bias

Section 4: Transparency and Explainability
  - Explainability approach for the system
  - User-facing explanations (format, content)
  - Technical documentation completeness
  - Audit trail capabilities

Section 5: Human Oversight
  - Human-in-the-loop design
  - Override mechanisms
  - Escalation procedures
  - Operator training requirements

Section 6: Data Governance
  - Data sources and quality assessment
  - Data lineage and provenance
  - Privacy considerations (PII, consent, purpose limitation)
  - Data retention and deletion plan

Section 7: Mitigation Plan
  - Controls for each identified risk
  - Implementation timeline
  - Residual risk assessment
  - Monitoring and review schedule

Section 8: Approval
  - Stakeholder sign-offs (technical, legal, ethics, business)
  - Conditions or restrictions on deployment
  - Review schedule (date of next assessment)
```

## Documentation Requirements

### Model Cards

```text
Model Card Template (Mitchell et al., 2019):

  Model Details:
    - Model name, version, date
    - Developers and contact
    - Model type and architecture
    - License and terms of use
    - Citation and references

  Intended Use:
    - Primary intended uses
    - Primary intended users
    - Out-of-scope use cases (explicitly state)

  Training Data:
    - Datasets used (name, size, source)
    - Data preprocessing steps
    - Annotation process
    - Known biases in training data

  Evaluation Data:
    - Datasets used for evaluation
    - Motivation for dataset selection
    - Preprocessing for evaluation

  Metrics:
    - Overall performance metrics
    - Performance disaggregated by subgroup
    - Decision thresholds and rationale
    - Confidence intervals

  Ethical Considerations:
    - Sensitive use cases
    - Risks and harms
    - Use cases to avoid
    - Fairness considerations

  Caveats and Recommendations:
    - Known limitations
    - Known failure modes
    - Recommended monitoring
    - Conditions for retraining
```

### Datasheets for Datasets

```text
Datasheet Template (Gebru et al., 2021):

  Motivation:
    - Why was this dataset created?
    - Who created it and on whose behalf?
    - Who funded it?

  Composition:
    - What does each instance represent?
    - How many instances total?
    - Does it contain sensitive information?
    - Is it a sample? If so, what population?

  Collection Process:
    - How was data collected?
    - Who was involved in collection?
    - What timeframe?
    - Were ethical review processes used?
    - Was consent obtained?

  Preprocessing:
    - What preprocessing was applied?
    - Was raw data retained?
    - What software was used?

  Uses:
    - What tasks has this been used for?
    - What should it NOT be used for?
    - Are there regulatory restrictions?

  Distribution:
    - How is it distributed?
    - License and terms?
    - Export control restrictions?

  Maintenance:
    - Who maintains the dataset?
    - How often is it updated?
    - How can errors be reported?
    - Will older versions remain available?
```

## AI Policy Templates

### Key Policies

```text
1. Acceptable Use of AI Policy
   - Approved AI tools and platforms
   - Prohibited uses (sensitive data in public AI, autonomous decisions)
   - Data handling requirements for AI tools
   - Approval process for new AI tools
   - Incident reporting procedures

2. AI Development Policy
   - Development lifecycle requirements
   - Documentation standards (model cards, datasheets)
   - Testing requirements (performance, fairness, security)
   - Approval gates for deployment
   - Open source AI usage guidelines

3. AI Ethics Policy
   - Ethical principles (fairness, transparency, accountability, safety)
   - Ethics review triggers and process
   - Prohibited applications
   - Stakeholder engagement requirements
   - Whistleblower protections

4. AI Risk Management Policy
   - Risk classification criteria
   - Risk assessment methodology
   - Risk tolerance levels
   - Monitoring and reporting requirements
   - Incident response procedures

5. AI Vendor Management Policy
   - Vendor AI assessment criteria
   - Contractual requirements (transparency, audit rights, data handling)
   - Third-party model evaluation requirements
   - Ongoing monitoring of vendor AI systems
   - Exit strategy and data portability
```

## See Also

- AI Ethics
- Privacy Regulations
- Supply Chain Security
- Security Awareness
- eu-ai-act
- ai-risk-management

## References

- NIST AI RMF 1.0 (January 2023): https://www.nist.gov/artificial-intelligence/ai-risk-management-framework
- EU AI Act: Regulation (EU) 2024/1689
- ISO/IEC 42001:2023 AI Management System
- OECD AI Principles (2019): https://oecd.ai/en/ai-principles
- IEEE 7000-2021: Standard for Ethical Design
- Mitchell et al. (2019): Model Cards for Model Reporting
- Gebru et al. (2021): Datasheets for Datasets
