# AI Security Architecture

AI Security Architecture encompasses the design principles, technical controls, and defensive strategies required to protect AI systems across the entire lifecycle, from securing training pipelines and model artifacts to defending inference endpoints against adversarial inputs, prompt injection, and data exfiltration, while preserving model utility through privacy-preserving computation techniques.

## Secure AI Design Principles
### Defense-in-Depth for AI Systems
```
┌──────────────────────────────────────────────────────────────┐
│                  AI Security Architecture                    │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐    │
│  │ Layer 1: Network & Infrastructure Security           │    │
│  │   Firewalls, VPN, mTLS, network segmentation         │    │
│  │ ┌──────────────────────────────────────────────────┐  │    │
│  │ │ Layer 2: API & Application Security              │  │    │
│  │ │   Auth, rate limiting, input validation           │  │    │
│  │ │ ┌──────────────────────────────────────────────┐ │  │    │
│  │ │ │ Layer 3: Model Security                      │ │  │    │
│  │ │ │   Access control, watermarking, monitoring    │ │  │    │
│  │ │ │ ┌──────────────────────────────────────────┐ │ │  │    │
│  │ │ │ │ Layer 4: Data Security                   │ │ │  │    │
│  │ │ │ │   Encryption, DLP, provenance, PETs      │ │ │  │    │
│  │ │ │ └──────────────────────────────────────────┘ │ │  │    │
│  │ │ └──────────────────────────────────────────────┘ │  │    │
│  │ └──────────────────────────────────────────────────┘  │    │
│  └──────────────────────────────────────────────────────┘    │
│                                                              │
│  Cross-cutting: Logging, Monitoring, Incident Response       │
└──────────────────────────────────────────────────────────────┘
```

### Core Principles
```
1. Least Privilege for AI Components
   - Models access only necessary data at inference time
   - Training pipelines use scoped credentials
   - Model serving has no access to training data
   - Tools/plugins get minimal permissions

2. Defense in Depth
   - Multiple independent security layers
   - No single control failure leads to compromise
   - Input validation at every boundary

3. Zero Trust for AI
   - Never trust model inputs (even from internal systems)
   - Never trust model outputs (always validate downstream)
   - Verify model integrity before serving
   - Authenticate and authorize every API call

4. Fail Secure
   - Model errors → safe default (not AI decision)
   - Authentication failure → deny access
   - Monitoring failure → alert and restrict
   - Graceful degradation to rule-based fallback

5. Assume Breach
   - Model weights may be extracted
   - Training data may be partially inferred
   - System prompts will be discovered
   - Plan for containment, not just prevention
```

## Model Protection
### Model Encryption
```
At Rest:
  ├─ Encrypt model weights with AES-256-GCM
  ├─ Key management via HSM or cloud KMS
  ├─ Separate keys per model version
  └─ Key rotation schedule (90 days)

In Transit:
  ├─ TLS 1.3 for all model transfer
  ├─ mTLS between training and serving infrastructure
  └─ Signed model artifacts (GPG or Sigstore)

In Use:
  ├─ Trusted Execution Environments (TEE)
  │   ├─ Intel SGX / TDX enclaves
  │   ├─ AWS Nitro Enclaves
  │   ├─ Azure Confidential Computing
  │   └─ GCP Confidential VMs
  ├─ Encrypted inference (homomorphic encryption)
  └─ Secure model loading (verify hash before load)
```

### Model Access Control
```
Role-Based Access:
  ┌─ Model Developer: train, evaluate, version
  ├─ ML Engineer: deploy, monitor, rollback
  ├─ Data Scientist: query, experiment (staging only)
  ├─ Application: inference only (production endpoint)
  ├─ Auditor: read metrics, logs, documentation
  └─ Admin: full access, key management

Access Control Implementation:
  ┌─ Authentication: API keys, OAuth 2.0, mTLS certs
  ├─ Authorization: RBAC with model-level granularity
  ├─ Rate Limiting: per-user, per-application, global
  ├─ Quotas: token/request limits per consumer
  └─ Audit: log all model access with caller identity

Model Registry Security:
  ┌─ Signed model artifacts (hash + signature)
  ├─ Immutable model versions (no overwrites)
  ├─ Access control on registry (push/pull)
  ├─ Vulnerability scanning of model dependencies
  └─ Provenance metadata (who, when, what data, what code)
```

### Model Watermarking
```
Purpose: Detect unauthorized model copies / prove ownership

Techniques:
  1. Weight-based Watermarking
     - Embed secret pattern in model weights
     - Survives fine-tuning and pruning
     - Verification: provide trigger inputs, check outputs

  2. Output-based Watermarking (Text)
     - Modify token sampling to embed statistical signal
     - Detectable with statistical test on output text
     - Examples: modifying logit bias for specific token sequences

  3. Backdoor-based Watermarking
     - Specific trigger input → specific output (benign backdoor)
     - Ownership proof: only owner knows trigger
     - Risk: can be detected/removed by determined adversary

  4. Fingerprinting
     - Use model's unique behavior on crafted inputs
     - Decision boundary differences between models
     - Conference/adversarial examples as fingerprints
```

## Training Pipeline Security
### Data Provenance
```
Data Supply Chain:
  Source → Collection → Storage → Processing → Training → Model

At Each Stage:
  ┌─ Cryptographic hash of data (SHA-256)
  ├─ Timestamp and actor identity
  ├─ Transformation applied (code version, parameters)
  ├─ Upstream source verification
  └─ Immutable audit log (append-only)

Data Provenance Standards:
  - W3C PROV for general provenance
  - SLSA (Supply-chain Levels for Software Artifacts) adapted for data
  - C2PA (Coalition for Content Provenance and Authenticity) for media

Implementation:
  ┌─ DVC (Data Version Control) for dataset versioning
  ├─ ML Metadata (MLMD) for pipeline provenance
  ├─ Sigstore/cosign for signing data artifacts
  └─ Content-addressable storage for immutability
```

### Secure Computation for Training
```
Threat: Unauthorized access to training data during computation

Protections:
  1. Trusted Execution Environments (TEE)
     - Training inside SGX/TDX enclave
     - Data decrypted only inside enclave
     - Attestation proves code integrity
     - Limitation: performance overhead, memory limits

  2. Secure Multi-Party Computation (SMPC)
     - Multiple parties contribute data without revealing it
     - Secret sharing: data split across parties
     - Computation on shares produces encrypted result
     - No single party sees complete dataset

  3. Federated Learning
     - Data stays on local devices/servers
     - Only gradients/updates sent to aggregator
     - Secure aggregation prevents gradient inspection
     - Differential privacy on updates for formal guarantees

  4. Differential Privacy in Training
     - DP-SGD: clip gradients + add calibrated noise
     - Privacy budget (epsilon) tracks cumulative exposure
     - Trades model utility for privacy guarantees
```

### Training Infrastructure Security
```
Compute Environment:
  ┌─ Isolated training VPC / network segment
  ├─ No internet egress from training nodes
  ├─ GPU/TPU cluster with hardware attestation
  ├─ Ephemeral training environments (destroy after use)
  └─ Encrypted scratch storage (auto-delete on completion)

Code Integrity:
  ┌─ Signed training code (git commit signing)
  ├─ Reproducible builds (pinned dependencies, containers)
  ├─ Code review for training scripts
  ├─ SBOM for training environment
  └─ Immutable container images (digest-based references)

Secrets Management:
  ┌─ No credentials in training code
  ├─ Vault/KMS for API keys, database credentials
  ├─ Short-lived tokens for data access
  ├─ Rotate credentials after each training run
  └─ Audit all secret access
```

## Inference Security
### Input Validation
```
Pre-Processing Pipeline:
  Raw Input → Schema Validation → Content Filtering →
  Sanitization → Feature Extraction → Model Inference

Schema Validation:
  ┌─ Type checking (text, image, structured data)
  ├─ Length limits (max tokens, image dimensions)
  ├─ Format validation (encoding, file type)
  ├─ Range checking (numeric bounds)
  └─ Required field validation

Content Filtering (Pre-Inference):
  ┌─ Known malicious pattern detection (regex, ML classifier)
  ├─ Prompt injection detection
  │   ├─ Instruction-hierarchy classifier
  │   ├─ Semantic similarity to known injection patterns
  │   └─ Perplexity-based anomaly detection
  ├─ PII detection and redaction
  ├─ Profanity/toxicity pre-screening
  └─ File type verification (magic bytes, not just extension)

Sanitization:
  ┌─ Unicode normalization (prevent homoglyph attacks)
  ├─ Control character removal
  ├─ HTML/script tag stripping
  ├─ Encoding normalization (UTF-8)
  └─ Whitespace normalization
```

### Output Filtering
```
Post-Processing Pipeline:
  Model Output → Content Safety Check → PII Scan →
  Format Validation → Sanitization → Response

Content Safety:
  ┌─ Toxicity classifier on output
  ├─ Harmful content detection (violence, self-harm, CSAM)
  ├─ Factuality check (for grounded applications)
  ├─ Bias detection on output
  └─ Refusal detection (ensure refusals are appropriate)

PII Protection:
  ┌─ Named entity recognition for PII
  ├─ Pattern matching (SSN, credit card, phone, email)
  ├─ Training data memorization detection
  ├─ Regex-based redaction
  └─ Configurable PII categories per application

Output Sanitization:
  ┌─ Code output sandboxing (no execution without review)
  ├─ URL validation (no internal URLs, SSRF prevention)
  ├─ Structured output schema validation
  ├─ Maximum output length enforcement
  └─ Encoding for downstream consumption (HTML escape, etc.)
```

## LLM Security
### Prompt Injection Defense
```
Defense Layers:

1. Architecture-Level
   ├─ Instruction hierarchy (system > user)
   ├─ Separate privileged and unprivileged contexts
   ├─ Tool/function calling with explicit schemas
   ├─ Output constrained to structured formats where possible
   └─ Principle of least privilege for tools/actions

2. Input-Level
   ├─ Prompt injection classifiers (fine-tuned detection models)
   ├─ Input segmentation (delimiters between system/user/data)
   ├─ Canary tokens in system prompt (detect extraction)
   ├─ Input length limits
   └─ Multi-turn context window management

3. Processing-Level
   ├─ Constitutional AI / RLHF alignment
   ├─ Instruction-following fine-tuning
   ├─ Output refusal training
   └─ Adversarial training against injection patterns

4. Output-Level
   ├─ Output validation against expected schema
   ├─ Action confirmation (human-in-the-loop for side effects)
   ├─ Tool call validation (allowed tools, allowed parameters)
   ├─ Response consistency checking
   └─ Hallucination detection

5. Monitoring-Level
   ├─ Log all prompts and responses
   ├─ Anomaly detection on prompt patterns
   ├─ Alert on known injection signatures
   ├─ Track refusal rates and patterns
   └─ Red team continuously
```

### Guardrails Implementation
```python
# Example guardrails pipeline
class LLMGuardrails:
    def __init__(self, config):
        self.input_filters = [
            PromptInjectionDetector(threshold=0.85),
            PII_Detector(categories=["SSN", "CREDIT_CARD", "EMAIL"]),
            ToxicityFilter(threshold=0.9),
            LengthValidator(max_tokens=4096),
        ]
        self.output_filters = [
            ContentSafetyClassifier(categories=config.blocked_categories),
            PIIRedactor(mode="mask"),
            HallucinationDetector(grounding_docs=config.knowledge_base),
            OutputSchemaValidator(schema=config.output_schema),
        ]

    def process_input(self, user_input, system_prompt):
        context = {"user_input": user_input, "system_prompt": system_prompt}
        for filter in self.input_filters:
            result = filter.check(context)
            if result.blocked:
                return BlockedResponse(
                    reason=result.reason,
                    filter=filter.__class__.__name__
                )
        return PassedInput(sanitized=context)

    def process_output(self, model_output, context):
        for filter in self.output_filters:
            result = filter.check(model_output, context)
            if result.blocked:
                return BlockedResponse(reason=result.reason)
            if result.modified:
                model_output = result.sanitized_output
        return ValidatedOutput(content=model_output)
```

### Content Filtering Architecture
```
┌────────────────────────────────────────────────────────────┐
│                  Content Filtering Pipeline                 │
│                                                            │
│  User Input                                                │
│      │                                                     │
│      ▼                                                     │
│  ┌──────────┐  Block  ┌────────────┐                      │
│  │ Injection │───────→│  Rejection  │                      │
│  │ Detector  │        │  Response   │                      │
│  └────┬─────┘        └────────────┘                      │
│       │ Pass                                               │
│       ▼                                                    │
│  ┌──────────┐  Block  ┌────────────┐                      │
│  │ Content  │───────→│  Rejection  │                      │
│  │ Policy   │        │  Response   │                      │
│  └────┬─────┘        └────────────┘                      │
│       │ Pass                                               │
│       ▼                                                    │
│  ┌──────────┐                                              │
│  │   LLM    │                                              │
│  │  Model   │                                              │
│  └────┬─────┘                                              │
│       │                                                    │
│       ▼                                                    │
│  ┌──────────┐  Block  ┌────────────┐                      │
│  │ Output   │───────→│  Safe       │                      │
│  │ Safety   │        │  Fallback   │                      │
│  └────┬─────┘        └────────────┘                      │
│       │ Pass                                               │
│       ▼                                                    │
│  ┌──────────┐                                              │
│  │  PII     │                                              │
│  │ Redactor │                                              │
│  └────┬─────┘                                              │
│       │                                                    │
│       ▼                                                    │
│  Response                                                  │
└────────────────────────────────────────────────────────────┘
```

## MLOps Security
### Secure ML Pipeline
```
┌─────────┐    ┌──────────┐    ┌─────────┐    ┌──────────┐
│  Data    │───→│ Training │───→│  Model  │───→│ Serving  │
│  Store   │    │ Pipeline │    │Registry │    │ Endpoint │
└────┬────┘    └────┬─────┘    └────┬────┘    └────┬─────┘
     │              │               │               │
  Encrypted     Isolated         Signed          Auth'd
  at rest       compute          artifacts       access
  Access        No egress        Immutable       Rate
  logged        Ephemeral        Versioned       limited

Security Controls at Each Stage:
  Data Store:
    ├─ Encryption (AES-256)
    ├─ Access control (RBAC + attribute-based)
    ├─ Data loss prevention (DLP)
    ├─ Audit logging
    └─ Data classification labels

  Training Pipeline:
    ├─ Container image scanning
    ├─ Dependency vulnerability scanning
    ├─ Reproducible builds
    ├─ Training code review
    └─ Experiment tracking with integrity

  Model Registry:
    ├─ Cryptographic signing (Sigstore/cosign)
    ├─ SBOM generation (ML-BOM)
    ├─ Vulnerability scanning
    ├─ Approval workflows
    └─ Immutable storage

  Serving Endpoint:
    ├─ Authentication (OAuth 2.0 / API key)
    ├─ Authorization (scope-based)
    ├─ Rate limiting and quotas
    ├─ Input/output validation
    └─ Monitoring and alerting
```

### CI/CD Security Gates for ML
```
Pre-Merge Gates:
  □ Code review approved (2 reviewers for training code)
  □ Unit tests pass
  □ Static analysis (bandit, semgrep) clean
  □ Dependency vulnerability scan (no critical/high)
  □ License compliance check

Pre-Training Gates:
  □ Data quality checks pass
  □ Data bias scan within thresholds
  □ Training configuration reviewed
  □ Compute budget approved
  □ Privacy budget (epsilon) within allocation

Pre-Deployment Gates:
  □ Model performance meets SLA
  □ Fairness metrics within thresholds
  □ Adversarial robustness test pass
  □ Security scan of model artifact
  □ Model signed and registered
  □ Canary deployment successful
  □ Rollback plan documented
  □ Monitoring dashboards configured
```

## Model Serving Security
### API Gateway Configuration
```
API Gateway for Model Serving:

Authentication:
  ├─ OAuth 2.0 with JWT validation
  ├─ API key authentication (for service-to-service)
  ├─ mTLS for internal services
  └─ Token introspection for fine-grained access

Rate Limiting:
  ├─ Global: 10,000 req/min across all consumers
  ├─ Per-user: 100 req/min (adjustable by tier)
  ├─ Per-application: 1,000 req/min
  ├─ Burst: 2x sustained rate for 10 seconds
  └─ Token-based limits for LLMs (tokens/min, tokens/day)

Request Validation:
  ├─ Content-Type enforcement
  ├─ Payload size limits (10MB default)
  ├─ Schema validation (JSON Schema / protobuf)
  ├─ Header validation
  └─ Query parameter sanitization

Security Headers:
  ├─ X-Request-ID (tracing)
  ├─ X-RateLimit-Remaining
  ├─ Strict-Transport-Security
  ├─ Content-Security-Policy
  └─ X-Content-Type-Options: nosniff
```

### Model Serving Infrastructure
```
Production Architecture:

  ┌─────────────────────────────────────────────┐
  │              Load Balancer (L7)              │
  │         (TLS termination, WAF rules)        │
  └──────────────────┬──────────────────────────┘
                     │
  ┌──────────────────▼──────────────────────────┐
  │              API Gateway                     │
  │    (Auth, rate limit, input validation)      │
  └──────────────────┬──────────────────────────┘
                     │
  ┌──────────────────▼──────────────────────────┐
  │          Guardrails Service                  │
  │   (Pre-processing, content safety)          │
  └──────────────────┬──────────────────────────┘
                     │
  ┌──────────────────▼──────────────────────────┐
  │          Model Serving Cluster               │
  │   (Isolated network, GPU nodes, no egress)  │
  └──────────────────┬──────────────────────────┘
                     │
  ┌──────────────────▼──────────────────────────┐
  │          Post-Processing Service             │
  │   (Output filtering, PII redaction)         │
  └──────────────────┬──────────────────────────┘
                     │
                     ▼
               Response to Client
```

## Adversarial Robustness
### Defense Strategies
```
Training-Time Defenses:
  1. Adversarial Training
     - Generate adversarial examples during training
     - Include in training batch (50% clean, 50% adversarial)
     - Increases robustness but may reduce clean accuracy
     - PGD-AT: use PGD to generate strong adversarial examples

  2. Certified Defenses
     - Randomized Smoothing: provable robustness via noise
     - Interval Bound Propagation: formal verification
     - Provides mathematical guarantee on adversarial radius

  3. Input Transformation
     - JPEG compression (removes high-frequency perturbations)
     - Spatial smoothing (blurs adversarial noise)
     - Feature squeezing (reduce input precision)
     - Limitation: can be bypassed by adaptive attacks

Inference-Time Defenses:
  1. Ensemble Methods
     - Multiple models vote on prediction
     - Adversarial examples rarely fool all models
     - Increases compute cost

  2. Input Preprocessing
     - Random resizing/padding
     - Denoising autoencoder
     - Neural network-based purification

  3. Detection
     - Statistical tests on input features
     - Neural network-based adversarial detectors
     - Confidence calibration (adversarial inputs often high-entropy)
```

## Privacy-Preserving ML
### Technique Comparison
```
┌──────────────────┬────────────┬─────────────┬──────────────┐
│ Technique        │ Privacy    │ Utility     │ Compute      │
│                  │ Guarantee  │ Impact      │ Overhead     │
├──────────────────┼────────────┼─────────────┼──────────────┤
│ Differential     │ Formal     │ Moderate    │ Low          │
│ Privacy (DP-SGD) │ (epsilon)  │ (2-10% acc) │ (1.1-2x)    │
├──────────────────┼────────────┼─────────────┼──────────────┤
│ Federated        │ Data stays │ Minor       │ Communication│
│ Learning         │ local      │ (1-5% acc)  │ overhead     │
├──────────────────┼────────────┼─────────────┼──────────────┤
│ Secure Multi-    │ Crypto-    │ None        │ Very high    │
│ Party Comp.      │ graphic    │ (exact)     │ (1000-10000x)│
├──────────────────┼────────────┼─────────────┼──────────────┤
│ Homomorphic      │ Crypto-    │ None        │ Very high    │
│ Encryption       │ graphic    │ (exact)     │ (10000x+)    │
├──────────────────┼────────────┼─────────────┼──────────────┤
│ Trusted Exec.    │ Hardware   │ None        │ Moderate     │
│ Environments     │ isolation  │ (exact)     │ (1.5-3x)    │
├──────────────────┼────────────┼─────────────┼──────────────┤
│ Synthetic Data   │ Statistical│ Variable    │ Training     │
│ Generation       │ (no formal)│ (5-20% acc) │ cost only    │
└──────────────────┴────────────┴─────────────┴──────────────┘
```

### Differential Privacy Implementation
```python
# DP-SGD Training Loop (simplified)
from opacus import PrivacyEngine

# Attach privacy engine to model training
privacy_engine = PrivacyEngine()
model, optimizer, data_loader = privacy_engine.make_private_with_epsilon(
    module=model,
    optimizer=optimizer,
    data_loader=train_loader,
    epochs=epochs,
    target_epsilon=3.0,       # Privacy budget
    target_delta=1e-5,        # Failure probability
    max_grad_norm=1.0,        # Gradient clipping bound
)

# Training proceeds normally — Opacus handles:
# 1. Per-sample gradient computation
# 2. Gradient clipping to max_grad_norm
# 3. Calibrated Gaussian noise addition
# 4. Privacy budget accounting (RDP → (ε,δ)-DP conversion)

for batch in data_loader:
    optimizer.zero_grad()
    loss = criterion(model(batch.x), batch.y)
    loss.backward()
    optimizer.step()

# Check spent privacy budget
epsilon = privacy_engine.get_epsilon(delta=1e-5)
print(f"Spent epsilon: {epsilon:.2f}")
```

### Federated Learning Setup
```
Federated Architecture:

  ┌──────────┐  ┌──────────┐  ┌──────────┐
  │ Client 1 │  │ Client 2 │  │ Client N │
  │ Local    │  │ Local    │  │ Local    │
  │ Data     │  │ Data     │  │ Data     │
  │ Training │  │ Training │  │ Training │
  └────┬─────┘  └────┬─────┘  └────┬─────┘
       │              │              │
       │   Encrypted  │   Encrypted  │
       │   Gradients  │   Gradients  │
       ▼              ▼              ▼
  ┌──────────────────────────────────────┐
  │         Secure Aggregator            │
  │  (aggregates without seeing          │
  │   individual updates)                │
  └──────────────────┬───────────────────┘
                     │
                     ▼
              Global Model Update
              (broadcast to clients)

Security Controls:
  ├─ Secure aggregation (MPC-based, no plaintext gradients)
  ├─ Differential privacy on local updates
  ├─ Byzantine-robust aggregation (median, Krum, trimmed mean)
  ├─ Client authentication and attestation
  ├─ Communication encryption (TLS 1.3)
  └─ Gradient compression (reduces attack surface + bandwidth)
```

## See Also
- ai-risk-management
- ai-privacy-trust
- ai-testing-assurance
- ai-supply-chain
- ai-compliance
- nist

## References
- OWASP LLM Top 10: https://owasp.org/www-project-top-10-for-large-language-model-applications/
- MITRE ATLAS: https://atlas.mitre.org/
- NIST AI 100-2e2023 — Adversarial ML: https://csrc.nist.gov/pubs/ai/100/2/e2023/final
- Opacus (DP-SGD): https://opacus.ai/
- Google Secure AI Framework (SAIF): https://safety.google/cybersecurity-advancements/saif/
- Microsoft AI Security Risk Assessment: https://learn.microsoft.com/en-us/security/ai-red-team/
