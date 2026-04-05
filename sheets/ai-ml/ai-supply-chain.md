# AI Supply Chain

AI Supply Chain security covers the end-to-end risk management of third-party AI components, from pre-trained models and datasets to ML libraries and cloud AI services, including model provenance verification, AI bill of materials (ML-BOM), vendor due diligence, open-source model risks, fine-tuning security, and continuous monitoring of external AI dependencies throughout their lifecycle.

## Third-Party AI Risk Assessment
### AI Vendor Risk Framework
```
Risk Assessment Dimensions:

  1. Model Risk
     ├─ Architecture transparency (open vs. closed)
     ├─ Training data provenance and documentation
     ├─ Known biases and limitations
     ├─ Performance claims and validation
     ├─ Model update frequency and versioning
     └─ Backdoor / trojan risk assessment

  2. Data Risk
     ├─ Training data sources and licensing
     ├─ PII in training data
     ├─ Data jurisdiction and sovereignty
     ├─ Data retention and deletion practices
     └─ Data sharing with third parties

  3. Operational Risk
     ├─ Service availability and SLA
     ├─ Vendor lock-in assessment
     ├─ Scalability and performance guarantees
     ├─ Incident response capabilities
     └─ Business continuity / vendor viability

  4. Security Risk
     ├─ API security (auth, encryption, rate limiting)
     ├─ Model access controls
     ├─ Data-in-transit and at-rest encryption
     ├─ Security certifications (SOC 2, ISO 27001)
     └─ Vulnerability disclosure program

  5. Compliance Risk
     ├─ Regulatory compliance (EU AI Act, sector-specific)
     ├─ Data protection (GDPR, CCPA)
     ├─ Audit rights and transparency
     ├─ Sub-processor management
     └─ Cross-border data transfer mechanisms

  6. Ethical Risk
     ├─ AI ethics policy and principles
     ├─ Bias testing and mitigation practices
     ├─ Responsible AI governance
     ├─ Dual-use risk assessment
     └─ Human rights impact assessment
```

### Risk Scoring Matrix
```
┌─────────────────────┬───────┬───────┬───────┬──────────┐
│ Risk Dimension      │ Low   │ Medium│ High  │ Critical │
│                     │ (1-2) │ (3-5) │ (6-8) │ (9-10)   │
├─────────────────────┼───────┼───────┼───────┼──────────┤
│ Model transparency  │Open   │Partial│Closed │Closed +  │
│                     │source │docs   │source │no docs   │
├─────────────────────┼───────┼───────┼───────┼──────────┤
│ Data provenance     │Full   │Partial│Minimal│None      │
│                     │docs   │docs   │docs   │          │
├─────────────────────┼───────┼───────┼───────┼──────────┤
│ Security posture    │SOC2+  │SOC2   │Basic  │No certs  │
│                     │ISO    │       │certs  │          │
├─────────────────────┼───────┼───────┼───────┼──────────┤
│ Vendor viability    │Public │Growth │Early  │Pre-      │
│                     │co.    │stage  │stage  │revenue   │
├─────────────────────┼───────┼───────┼───────┼──────────┤
│ Compliance readiness│Full   │Partial│In     │Not       │
│                     │       │       │progress│started  │
├─────────────────────┼───────┼───────┼───────┼──────────┤
│ Lock-in risk        │Open   │Std.   │Propri-│Fully     │
│                     │std.   │APIs   │etary  │locked    │
└─────────────────────┴───────┴───────┴───────┴──────────┘

Overall Score = Σ(Dimension Score × Weight) / Σ(Weights)

Risk Thresholds:
  1-3:  Acceptable — proceed with standard monitoring
  4-5:  Elevated — proceed with enhanced controls
  6-7:  High — requires risk acceptance by senior leadership
  8-10: Critical — do not proceed without mitigation
```

## Model Provenance
### Provenance Chain
```
Model Provenance = Complete lineage from data to deployed model

┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│  Data    │───→│ Training │───→│  Model   │───→│ Deploy-  │
│  Source  │    │ Process  │    │ Artifact │    │  ment    │
└────┬─────┘    └────┬─────┘    └────┬─────┘    └────┬─────┘
     │               │               │               │
  - Origin       - Code hash     - Weights hash   - Environment
  - License      - Framework     - Architecture   - Config hash
  - Hash         - Hyperparams   - Metrics        - Deploy date
  - Collection   - Hardware      - Dependencies   - Operator
    method       - Duration      - Signature      - Approval
  - PII scan     - Random seed   - Model card     - Version
  - Bias audit   - Data version  - License        - Endpoint

Verification at Each Stage:
  □ Cryptographic hash matches expected value
  □ Digital signature valid (Sigstore, GPG)
  □ Provenance metadata complete
  □ License compatible with intended use
  □ No known vulnerabilities in dependencies
  □ Audit trail unbroken from source to deployment
```

### Model Signing and Verification
```bash
# Signing model artifacts with cosign (Sigstore)
# Sign a model file
cosign sign-blob --key cosign.key \
  --output-signature model.sig \
  --output-certificate model.cert \
  model.safetensors

# Verify model signature
cosign verify-blob --key cosign.pub \
  --signature model.sig \
  model.safetensors

# Sign container image containing model
cosign sign --key cosign.key \
  registry.example.com/models/my-model:v1.0

# Verify container signature
cosign verify --key cosign.pub \
  registry.example.com/models/my-model:v1.0

# Using SLSA provenance for models
# Generate SLSA provenance attestation
slsa-verifier verify-artifact model.safetensors \
  --provenance-path model.intoto.jsonl \
  --source-uri github.com/org/model-repo \
  --source-tag v1.0.0

# Content Credentials (C2PA) for AI-generated content
# Embed provenance into AI-generated media
c2patool model_output.png \
  --manifest manifest.json \
  --output signed_output.png
```

## Pre-Trained Model Risks
### Threat Taxonomy
```
Backdoor / Trojan Attacks:
  ├─ Weight Poisoning: Malicious behavior encoded in model weights
  │   - Trigger: specific input pattern activates backdoor
  │   - Impact: misclassification, data exfiltration, harmful output
  │   - Detection: Neural Cleanse, Activation Clustering, Meta Neural Analysis
  │
  ├─ Sleeper Agent: Behavior activates under specific conditions
  │   - Time-based: activates after deployment date
  │   - Input-based: rare but specific input triggers
  │   - Context-based: specific deployment environment triggers
  │   - Detection: extremely difficult, requires exhaustive testing
  │
  └─ Supply Chain Injection:
      - Compromised model hosting platform
      - Man-in-the-middle during model download
      - Malicious model masquerading as legitimate
      - Detection: hash verification, signed artifacts

Bias Risks in Pre-Trained Models:
  ├─ Training data biases amplified through transfer learning
  ├─ Cultural and linguistic biases embedded in language models
  ├─ Representation gaps for minority groups
  ├─ Stereotyping behavior in generative models
  └─ Geographic bias (models trained primarily on Western data)

Intellectual Property Risks:
  ├─ Training data may include copyrighted material
  ├─ Model outputs may reproduce training data
  ├─ License restrictions on commercial use
  ├─ Patent claims on model architectures
  └─ Trade secret exposure through model behavior

Security Vulnerabilities:
  ├─ Serialization attacks (pickle, SafeTensors vs. PyTorch native)
  ├─ Dependency vulnerabilities in model framework
  ├─ Model file format exploits
  └─ Arbitrary code execution via model loading
```

### Model File Format Security
```
Risk Levels by Format:

  HIGH RISK — Arbitrary Code Execution:
  ┌─ Python Pickle (.pkl, .pt, .pth)
  │   - Executes arbitrary Python during deserialization
  │   - Can install malware, exfiltrate data, establish reverse shell
  │   - NEVER load untrusted pickle files
  │
  ├─ Joblib (.joblib)
  │   - Uses pickle internally, same risks
  │
  └─ ONNX (older versions with custom ops)
      - Custom operators can execute arbitrary code

  MEDIUM RISK — Limited Attack Surface:
  ├─ TensorFlow SavedModel (.pb)
  │   - Binary protobuf format
  │   - Custom ops can execute code
  │   - Safer than pickle but not immune
  │
  └─ Keras H5 (.h5)
      - HDF5 format with lambda layers risk
      - Custom objects can execute code

  LOW RISK — Data-Only Formats:
  ├─ SafeTensors (.safetensors) ← RECOMMENDED
  │   - Developed by Hugging Face
  │   - Pure tensor data, no code execution
  │   - Memory-mapped, fast loading
  │   - Designed specifically for security
  │
  ├─ ONNX (standard ops only)
  │   - Computation graph only (no custom ops)
  │   - Well-defined operator set
  │
  └─ GGUF (.gguf)
      - Binary format for quantized models
      - No code execution capability

Best Practice:
  1. Convert all models to SafeTensors before use
  2. Scan pickle files with fickling before loading
  3. Verify checksums against trusted source
  4. Run model loading in sandboxed environment
  5. Never execute pip install from model files
```

## AI SBOM (ML-BOM)
### ML Bill of Materials
```
ML-BOM Components:

  1. Model Identity
     ├─ Model name, version, hash (SHA-256)
     ├─ Architecture description
     ├─ Parameter count
     ├─ File format and size
     └─ Unique identifier (PURL or custom)

  2. Training Data
     ├─ Dataset names, versions, hashes
     ├─ Dataset licenses
     ├─ Data sources and collection methods
     ├─ Preprocessing steps applied
     └─ Known biases and limitations

  3. Training Environment
     ├─ Framework and version (PyTorch 2.x, TF 2.x)
     ├─ Language version (Python 3.x)
     ├─ Hardware (GPU type, count)
     ├─ OS and container image
     └─ Training script hash

  4. Dependencies
     ├─ Direct dependencies (with versions, hashes)
     ├─ Transitive dependencies (full tree)
     ├─ Known vulnerabilities (CVE mapping)
     ├─ License inventory
     └─ Dependency update policy

  5. Pre-Trained Components
     ├─ Base model identity and provenance
     ├─ Pre-trained weights source
     ├─ Fine-tuning methodology
     ├─ Modifications from base model
     └─ Original model license

  6. Performance Metadata
     ├─ Evaluation metrics and datasets
     ├─ Fairness assessment results
     ├─ Robustness test results
     ├─ Known failure modes
     └─ Operational constraints

ML-BOM Format (SPDX-based):
  {
    "spdxVersion": "SPDX-2.3",
    "name": "ml-bom-my-model-v1.0",
    "packages": [
      {
        "name": "my-model",
        "version": "1.0.0",
        "downloadLocation": "https://...",
        "checksums": [{"algorithm": "SHA256", "value": "abc123..."}],
        "licenseConcluded": "Apache-2.0",
        "primaryPackagePurpose": "ML-MODEL",
        "externalRefs": [
          {"referenceType": "purl",
           "referenceLocator": "pkg:ml/org/my-model@1.0.0"}
        ]
      }
    ],
    "relationships": [
      {"type": "TRAINED_ON", "relatedElement": "dataset-xyz"},
      {"type": "DEPENDS_ON", "relatedElement": "pytorch-2.1"}
    ]
  }
```

## Vendor Due Diligence for AI
### Due Diligence Checklist
```
Technical Assessment:
  □ Model architecture documented and appropriate for use case
  □ Training data sources disclosed and license-compatible
  □ Performance benchmarks provided and independently verifiable
  □ Bias/fairness testing conducted and results shared
  □ Robustness testing conducted (adversarial, edge cases)
  □ Model update/versioning policy documented
  □ API security review (auth, encryption, rate limiting)
  □ Incident history and response track record

Compliance Assessment:
  □ EU AI Act risk classification determined
  □ Conformity assessment evidence (if high-risk)
  □ GDPR compliance (DPA, data processing records)
  □ Sector-specific compliance (FDA, SEC, etc.)
  □ Audit rights included in contract
  □ Data residency and sovereignty requirements met
  □ Sub-processor transparency

Operational Assessment:
  □ SLA terms (uptime, latency, throughput)
  □ Disaster recovery and business continuity plan
  □ Scalability limits documented
  □ Monitoring and alerting capabilities
  □ Support responsiveness and expertise
  □ Exit/migration plan feasibility

Financial and Business:
  □ Vendor financial stability (funding, revenue)
  □ Insurance coverage (cyber, E&O)
  □ Pricing model transparency and predictability
  □ Lock-in assessment and switching costs
  □ Reference customers in similar industry
  □ Roadmap alignment with organizational needs
```

## Model Marketplace Security
### Platform Risk Assessment
```
Hugging Face Hub:
  Risks:
    ├─ Malicious models uploaded by anyone (open platform)
    ├─ Pickle deserialization attacks in model files
    ├─ Misleading model cards (performance claims)
    ├─ License confusion (model vs. code vs. data)
    └─ Dependency on external platform availability

  Mitigations:
    ├─ Use SafeTensors format exclusively
    ├─ Verify model author identity
    ├─ Check community activity and reviews
    ├─ Scan model files before loading (fickling)
    ├─ Pin specific model revisions (commit hash)
    └─ Mirror critical models to private registry

  Security Features:
    ├─ Malware scanning on uploads
    ├─ SafeTensors conversion tool
    ├─ Model signing (in development)
    ├─ Access tokens with scoped permissions
    └─ Private model repositories

Other Platforms:
  ┌───────────────┬──────────────┬────────────────────┐
  │ Platform      │ Openness     │ Security Controls  │
  ├───────────────┼──────────────┼────────────────────┤
  │ HuggingFace   │ Open upload  │ Scanning, SafeT.   │
  │ Kaggle Models │ Moderated    │ Platform review     │
  │ NGC (NVIDIA)  │ Curated      │ Signed, validated   │
  │ TF Hub        │ Curated      │ Google-verified     │
  │ PyTorch Hub   │ Semi-curated │ GitHub-based        │
  │ Azure AI      │ Curated      │ Enterprise controls │
  │ AWS Bedrock   │ Curated      │ AWS security        │
  │ Ollama        │ Community    │ GGUF format (safer) │
  └───────────────┴──────────────┴────────────────────┘
```

## Dataset Provenance
### Datasheets for Datasets (Gebru et al.)
```
Key Provenance Questions:

  Origin:
    ├─ Who created the dataset and why?
    ├─ What funding supported its creation?
    ├─ When was it collected/created?
    └─ What geographic/demographic scope?

  Collection:
    ├─ How was data collected? (scraping, surveys, sensors, APIs)
    ├─ Was consent obtained from data subjects?
    ├─ Were ethical review processes followed?
    ├─ What was the sampling strategy?
    └─ Are there known collection biases?

  Composition:
    ├─ What does each instance represent?
    ├─ Are there missing values or errors?
    ├─ Does it contain PII or sensitive information?
    ├─ Is it representative of the target population?
    └─ What are the label definitions and quality?

  Legal:
    ├─ What license governs the dataset?
    ├─ Were there terms of service for data sources?
    ├─ Are there copyright concerns?
    ├─ What data protection laws apply?
    └─ Are there export control restrictions?

  Maintenance:
    ├─ Who is responsible for updates?
    ├─ How are errors corrected?
    ├─ Will historical versions remain available?
    └─ Is there a deprecation plan?

Red Flags:
  ✗ No documentation of data sources
  ✗ Unclear or no licensing terms
  ✗ No bias analysis conducted
  ✗ PII present without consent evidence
  ✗ Scraping from sites prohibiting it
  ✗ No version control or change tracking
```

## Open-Source Model Risks
### Risk Categories
```
Security Risks:
  ├─ Malicious code in model files (pickle attacks)
  ├─ Backdoored weights (trojan models)
  ├─ Vulnerable dependencies (transitive risk)
  ├─ Compromised model repositories (supply chain)
  └─ No security patch SLA (community-maintained)

Quality Risks:
  ├─ Performance claims not independently verified
  ├─ Limited or no documentation
  ├─ Inconsistent quality across versions
  ├─ No formal testing or validation
  └─ Training data quality unknown

Legal Risks:
  ├─ License ambiguity (model vs. weights vs. outputs)
  ├─ Training data may include copyrighted material
  ├─ Patent infringement potential
  ├─ No indemnification or warranty
  └─ License incompatibility with proprietary use

Operational Risks:
  ├─ No vendor support or SLA
  ├─ Community may abandon project
  ├─ Breaking changes between versions
  ├─ Limited scalability guidance
  └─ No guaranteed update schedule

Mitigation Strategy:
  1. Establish approved model list (vetted open-source models)
  2. Use SafeTensors format only
  3. Verify model hashes against trusted source
  4. Run security scan (fickling, dependency scan)
  5. Conduct internal performance validation
  6. Review license compatibility
  7. Mirror models in internal registry
  8. Monitor upstream for security advisories
  9. Define fallback plan if model is discontinued
```

## Fine-Tuning Security
### Secure Fine-Tuning Practices
```
Data Security:
  ├─ Fine-tuning data encrypted at rest and in transit
  ├─ Access controls on fine-tuning datasets
  ├─ PII detection and removal before fine-tuning
  ├─ Data quality validation (no poisoned samples)
  └─ Data provenance documentation

Process Security:
  ├─ Isolated compute environment (no internet egress)
  ├─ Reproducible environment (pinned dependencies)
  ├─ Version control for fine-tuning code and configs
  ├─ Audit logging of all fine-tuning runs
  └─ Approval workflow before production deployment

Model Integrity:
  ├─ Base model hash verified before fine-tuning
  ├─ Fine-tuned model hash recorded and signed
  ├─ Performance comparison: fine-tuned vs. base model
  ├─ Safety evaluation: fine-tuning didn't degrade safety
  └─ Bias assessment: fine-tuning didn't introduce bias

Fine-Tuning Risks:
  ├─ Catastrophic forgetting of safety training
  │   Risk: Fine-tuning can override RLHF alignment
  │   Mitigation: Include safety examples in fine-tuning data
  │   Mitigation: Post-fine-tuning safety evaluation
  │
  ├─ Data poisoning through fine-tuning data
  │   Risk: Malicious examples in fine-tuning dataset
  │   Mitigation: Data curation and review
  │   Mitigation: Anomaly detection on training data
  │
  ├─ Overfitting to fine-tuning data
  │   Risk: Memorization and leakage of sensitive data
  │   Mitigation: Regularization, early stopping
  │   Mitigation: Membership inference testing
  │
  └─ License violation
      Risk: Fine-tuning may violate base model license
      Mitigation: Review license before fine-tuning
      Mitigation: Some licenses prohibit certain uses
```

## Model Licensing
### AI License Landscape
```
Permissive Licenses:
  Apache 2.0
    ├─ Commercial use: Yes
    ├─ Modification: Yes
    ├─ Distribution: Yes
    ├─ Patent grant: Yes
    ├─ Copyleft: No
    └─ Common models: Many HuggingFace models, BERT, T5

  MIT
    ├─ Commercial use: Yes
    ├─ Modification: Yes
    ├─ Distribution: Yes
    ├─ Patent grant: No (implicit)
    ├─ Copyleft: No
    └─ Common models: Various research models

Responsible AI Licenses (RAIL):
  OpenRAIL-M (Model-specific)
    ├─ Commercial use: Yes (with restrictions)
    ├─ Modification: Yes
    ├─ Distribution: Yes (must include use restrictions)
    ├─ Use restrictions: Cannot use for harm, surveillance, etc.
    ├─ Copyleft: Behavioral (restrictions propagate)
    └─ Common models: Stable Diffusion, BLOOM

  OpenRAIL-S (Source code)
    ├─ Same as OpenRAIL-M but for code
    └─ Restrictions propagate to derivatives

  CreativeML OpenRAIL-M
    ├─ Variant for creative/generative models
    └─ Common models: Stable Diffusion variants

Custom / Proprietary Licenses:
  Llama Community License (Meta)
    ├─ Commercial use: Yes (under 700M monthly active users)
    ├─ Modification: Yes
    ├─ Distribution: Yes (with license propagation)
    ├─ Revenue threshold: Must license from Meta above limit
    └─ Use restrictions: Yes (no training competing models)

  Gemma Terms of Use (Google)
    ├─ Commercial use: Yes
    ├─ Modification: Yes
    ├─ Use restrictions: Yes (prohibited uses list)
    └─ Redistribution: Must include terms

  Mistral License
    ├─ Apache 2.0 for open models
    ├─ Commercial license for API models
    └─ Different terms per model family

License Compatibility Matrix:
  ┌──────────┬──────────┬──────────┬──────────┬──────────┐
  │ Base     │ Apache   │ MIT      │ RAIL     │ Llama    │
  │ Fine-tune│ 2.0      │          │          │          │
  ├──────────┼──────────┼──────────┼──────────┼──────────┤
  │Apache 2.0│   ✓      │   ✓      │   ✓      │   ✓      │
  │MIT       │   ✓      │   ✓      │   ✓      │   ✓      │
  │RAIL      │   ✓*     │   ✓*     │   ✓      │   ✗      │
  │Llama     │   ✗      │   ✗      │   ✗      │   ✓      │
  └──────────┴──────────┴──────────┴──────────┴──────────┘
  * Must propagate RAIL use restrictions to derivative
```

## AI Vendor Contracts
### Key Contract Provisions
```
AI-Specific Contract Terms:

  Performance Guarantees:
    ├─ Minimum accuracy/performance metrics
    ├─ Maximum latency SLAs
    ├─ Availability commitments (99.9%, etc.)
    ├─ Performance degradation notification requirement
    └─ Remediation timeline for performance failures

  Transparency and Auditability:
    ├─ Right to audit AI systems and processes
    ├─ Model card / documentation requirements
    ├─ Training data disclosure obligations
    ├─ Change notification before model updates
    └─ Access to performance metrics and logs

  Data Protection:
    ├─ Data processing agreement (DPA)
    ├─ Prohibition on using customer data for training
    ├─ Data residency requirements
    ├─ Data retention and deletion obligations
    ├─ Sub-processor transparency and approval

  Liability and Indemnification:
    ├─ Liability for AI errors and biased outputs
    ├─ IP indemnification (training data copyright)
    ├─ Regulatory fine allocation
    ├─ Insurance requirements
    └─ Limitation of liability specifics

  Exit and Portability:
    ├─ Data portability upon termination
    ├─ Model portability (if custom-trained)
    ├─ Transition assistance period
    ├─ Data deletion certification
    └─ Reasonable exit timeline

  Compliance Support:
    ├─ EU AI Act conformity assessment support
    ├─ Regulatory documentation provision
    ├─ Cooperation with regulatory inquiries
    ├─ Incident notification requirements
    └─ Compliance certification sharing
```

## Continuous Monitoring of Third-Party AI
### Monitoring Framework
```
Continuous Monitoring Cadence:

  Real-Time:
    ├─ API availability and latency
    ├─ Error rates and types
    ├─ Output quality spot checks
    └─ Security event monitoring

  Daily:
    ├─ Performance metric tracking
    ├─ Data drift detection on inputs/outputs
    ├─ Cost monitoring (usage vs. budget)
    └─ Anomaly detection on model behavior

  Weekly:
    ├─ Fairness metric review
    ├─ Output quality audit (sample review)
    ├─ Vendor communication review
    └─ Dependency vulnerability scan

  Monthly:
    ├─ Comprehensive performance review
    ├─ Vendor risk score update
    ├─ License compliance check
    └─ Cost optimization review

  Quarterly:
    ├─ Full vendor risk reassessment
    ├─ Contract compliance review
    ├─ Regulatory change impact assessment
    ├─ Alternative vendor evaluation
    └─ Penetration testing of AI integration

  Annually:
    ├─ Complete due diligence refresh
    ├─ Contract renegotiation evaluation
    ├─ Strategic alignment review
    └─ Comprehensive audit (internal or third-party)

Alert Triggers:
  ├─ Model version change by vendor (undisclosed)
  ├─ Performance degradation > threshold
  ├─ Bias metric out of acceptable range
  ├─ Vendor security incident disclosure
  ├─ Vendor financial distress signals
  ├─ Regulatory enforcement action against vendor
  └─ License or terms of service change
```

## See Also
- ai-risk-management
- ai-security-architecture
- ai-compliance
- ai-testing-assurance
- open-source-licensing

## References
- SLSA Framework: https://slsa.dev/
- Sigstore (cosign): https://www.sigstore.dev/
- Hugging Face SafeTensors: https://huggingface.co/docs/safetensors/
- SPDX: https://spdx.dev/
- MITRE ATLAS Supply Chain: https://atlas.mitre.org/
- Fickling (pickle scanner): https://github.com/trailofbits/fickling
- C2PA (Content Credentials): https://c2pa.org/
- OpenRAIL License: https://www.licenses.ai/
