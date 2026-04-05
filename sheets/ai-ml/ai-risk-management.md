# AI Risk Management

AI Risk Management is the systematic process of identifying, assessing, mitigating, and monitoring risks specific to artificial intelligence systems, encompassing model failures, adversarial threats, data integrity issues, and operational hazards across the AI lifecycle from development through deployment and decommissioning.

## AI-Specific Risk Categories
### Model Failure Modes
```
Hallucination / Confabulation
  - Model generates plausible but factually incorrect output
  - Risk: Decision-making based on fabricated information
  - Mitigation: Grounding, RAG, output verification, confidence scoring

Data Poisoning
  - Adversary corrupts training data to influence model behavior
  - Risk: Backdoor triggers, degraded accuracy, biased outputs
  - Mitigation: Data provenance, anomaly detection, robust training

Adversarial Attacks
  - Crafted inputs designed to fool the model
  - Types: Evasion (inference-time), poisoning (training-time)
  - Mitigation: Adversarial training, input validation, certified defenses

Model Drift
  - Data Drift: Input distribution shifts from training data
  - Concept Drift: Relationship between features and target changes
  - Risk: Silent performance degradation over time
  - Mitigation: Monitoring dashboards, automated retraining triggers

Prompt Injection (LLMs)
  - Direct: Malicious instructions in user input
  - Indirect: Injected via external data sources (web, docs, APIs)
  - Risk: Unauthorized actions, data exfiltration, guardrail bypass
  - Mitigation: Input sanitization, output filtering, privilege separation

Bias and Fairness Failures
  - Historical bias in training data propagated to predictions
  - Representation bias from non-representative datasets
  - Measurement bias from flawed data collection
  - Risk: Discriminatory outcomes, regulatory violations, reputational harm
  - Mitigation: Fairness metrics, bias audits, diverse datasets

Model Extraction / Theft
  - Adversary reconstructs model through query access
  - Risk: IP theft, enabling targeted adversarial attacks
  - Mitigation: Rate limiting, query detection, watermarking
```

## NIST AI Risk Management Framework (AI RMF 1.0)
### Four Core Functions
```
┌──────────────────────────────────────────────────────────┐
│                    NIST AI RMF 1.0                       │
├──────────────┬──────────────┬────────────┬───────────────┤
│    GOVERN    │     MAP      │  MEASURE   │    MANAGE     │
│              │              │            │               │
│ Culture &    │ Context &    │ Analyze &  │ Allocate &    │
│ oversight    │ scope risks  │ quantify   │ mitigate      │
│              │              │ risks      │ risks         │
├──────────────┼──────────────┼────────────┼───────────────┤
│ GV-1: Pol-   │ MP-1: Intended│ MS-1: AI  │ MG-1: Risk   │
│   icies &    │   context    │  risk met- │  prioritized  │
│   processes  │ MP-2: Task   │  rics used │  & treated    │
│ GV-2: Ac-    │   categorize │ MS-2: AI   │ MG-2: Risk   │
│   countable  │ MP-3: AI     │  tested    │  response     │
│   structure  │   benefits & │  before    │  implemented  │
│ GV-3: Work-  │   costs vs.  │  deploy    │ MG-3: Risk   │
│   force      │   risks      │ MS-3: Con- │  managed      │
│   diversity  │ MP-4: Risks  │  tinuous   │  post-deploy  │
│ GV-4: Org    │   mapped     │  monitor   │ MG-4: Inci-  │
│   culture    │ MP-5: Im-    │ MS-4: Feed-│  dent mgmt   │
│ GV-5: Proc-  │   pacts      │  back loop │               │
│   esses      │   documented │            │               │
│ GV-6: Risk   │              │            │               │
│   oversight  │              │            │               │
└──────────────┴──────────────┴────────────┴───────────────┘
```

### GOVERN Function Implementation
```
GV-1: Policies and Procedures
  ┌─ AI acceptable use policy
  ├─ AI ethics principles
  ├─ Model approval process (stage gates)
  ├─ Data governance for AI
  ├─ Third-party AI usage policy
  └─ AI incident response plan

GV-2: Accountability Structure
  ┌─ Chief AI Officer / AI Ethics Board
  ├─ Model Risk Management team
  ├─ AI audit function (2nd/3rd line)
  ├─ Clear RACI for AI lifecycle
  └─ Escalation paths for AI risk

GV-6: Oversight and Due Diligence
  ┌─ Regular AI risk reporting to board
  ├─ Portfolio-level AI risk view
  ├─ Third-party AI risk assessments
  └─ Regulatory horizon scanning
```

### MAP Function — Risk Identification
```
MP-1: Intended Context
  - Document intended use, users, environment
  - Identify who might be impacted
  - Define operational boundaries and constraints

MP-2: Categorize the AI Task
  - Classification / regression / generation / recommendation
  - Autonomy level: advisory → semi-autonomous → fully autonomous
  - Criticality: low → medium → high → safety-critical

MP-3: Benefits vs. Risks
  - Quantify expected value of AI system
  - Enumerate potential harms (individuals, groups, society)
  - Cost-benefit analysis with risk-adjusted projections

MP-4: Risk Mapping
  - NIST AI risk categories applied to this system
  - Attack surface analysis
  - Failure mode analysis (FMEA for AI)
```

### MEASURE Function — Risk Quantification
```
MS-1: Metrics Selection
  ┌─ Performance: accuracy, F1, AUC-ROC by subgroup
  ├─ Fairness: demographic parity, equalized odds, calibration
  ├─ Robustness: adversarial accuracy, perturbation tolerance
  ├─ Privacy: differential privacy epsilon, membership inference rate
  ├─ Explainability: fidelity of explanations, coverage
  └─ Reliability: MTBF, error rate, uptime

MS-2: Pre-deployment Testing
  ┌─ Unit tests for model components
  ├─ Integration tests for ML pipeline
  ├─ Red team / adversarial testing
  ├─ Bias / fairness audit
  ├─ Security penetration testing
  └─ Stress testing under edge cases

MS-3: Continuous Monitoring
  ┌─ Data drift detection (PSI, KS test, JS divergence)
  ├─ Concept drift detection (ADWIN, DDM, EDDM)
  ├─ Performance degradation alerts
  ├─ Fairness metric tracking
  └─ Adversarial query detection
```

## AI Threat Landscape
### OWASP Top 10 for LLM Applications (2025)
```
LLM01: Prompt Injection
  - Direct manipulation via user prompts
  - Indirect injection via external data sources
  - Defense: input/output filtering, privilege separation

LLM02: Sensitive Information Disclosure
  - Training data extraction
  - PII leakage in responses
  - Defense: output scanning, data sanitization

LLM03: Supply Chain Vulnerabilities
  - Compromised pre-trained models
  - Poisoned training datasets
  - Vulnerable ML libraries
  - Defense: model provenance, ML-BOM, dependency scanning

LLM04: Data and Model Poisoning
  - Training data manipulation
  - Fine-tuning attacks
  - Defense: data validation, anomaly detection

LLM05: Improper Output Handling
  - Unsanitized model output used in downstream systems
  - XSS, SSRF, code injection via model output
  - Defense: output encoding, sandboxing

LLM06: Excessive Agency
  - Over-permissioned plugins/tools
  - Autonomous actions without approval
  - Defense: least privilege, human-in-the-loop

LLM07: System Prompt Leakage
  - Extraction of system instructions
  - Defense: treat system prompts as non-secret, defense-in-depth

LLM08: Vector and Embedding Weaknesses
  - Poisoned embeddings in RAG systems
  - Defense: embedding validation, source filtering

LLM09: Misinformation
  - Confident generation of false information
  - Defense: grounding, fact-checking, citations

LLM10: Unbounded Consumption
  - Resource exhaustion via crafted prompts
  - Denial of wallet attacks
  - Defense: rate limiting, budget caps, timeout controls
```

### MITRE ATLAS (Adversarial Threat Landscape for AI Systems)
```
Tactic Categories:
  Reconnaissance     → ML model fingerprinting, data source ID
  Resource Development → Adversarial model training, dataset creation
  Initial Access      → Supply chain compromise, API access abuse
  ML Attack Staging   → Model poisoning, adversarial crafting
  ML Model Access     → Query API, physical env manipulation
  Exfiltration        → Model extraction, training data extraction
  Impact              → Evasion, model degradation, denial of service

Key Techniques:
  AML.T0000: ML Supply Chain Compromise
  AML.T0003: Adversarial Example Crafting (white/black box)
  AML.T0004: Data Poisoning
  AML.T0007: Model Discovery (architecture fingerprinting)
  AML.T0012: Model Extraction / Stealing
  AML.T0015: Prompt Injection
  AML.T0016: LLM Jailbreaking
  AML.T0024: Membership Inference
  AML.T0025: Model Inversion
  AML.T0040: ML-Enabled Phishing
  AML.T0043: Backdoor ML Model
  AML.T0048: Evade ML Model
```

## AI Risk Assessment
### Risk Assessment Process
```
Step 1: System Inventory
  ┌─ Catalog all AI/ML systems
  ├─ Classification: traditional ML, deep learning, generative AI
  ├─ Deployment: internal, customer-facing, embedded
  ├─ Data sensitivity: public, internal, confidential, restricted
  └─ Impact level: low, medium, high, critical

Step 2: Threat Modeling for AI
  ┌─ Identify threat actors (external, insider, supply chain)
  ├─ Map attack surfaces (training, inference, data, model)
  ├─ Apply STRIDE adapted for AI:
  │   S — Spoofing training data sources
  │   T — Tampering with model weights or training data
  │   R — Repudiation of AI decisions
  │   I — Information disclosure (model/data extraction)
  │   D — Denial of service (resource exhaustion)
  │   E — Elevation of privilege (prompt injection, jailbreaking)
  └─ Document threat scenarios and likelihood

Step 3: Impact Assessment
  ┌─ Harm to individuals (discrimination, safety, privacy)
  ├─ Financial impact (direct loss, fines, remediation)
  ├─ Reputational damage
  ├─ Regulatory consequences
  └─ Systemic / societal impact

Step 4: Risk Scoring
  Risk = Likelihood × Impact × Exposure

  Likelihood: 1 (Rare) → 5 (Almost Certain)
  Impact:     1 (Negligible) → 5 (Catastrophic)
  Exposure:   1 (Limited) → 5 (Widespread)

  Score Range:   1-25   = Low
                 26-50  = Medium
                 51-75  = High
                 76-125 = Critical
```

## Model Monitoring
### Data Drift Detection
```python
# Population Stability Index (PSI)
import numpy as np

def calculate_psi(reference, current, bins=10):
    """Calculate PSI between reference and current distributions."""
    ref_hist, bin_edges = np.histogram(reference, bins=bins, density=True)
    cur_hist, _ = np.histogram(current, bins=bin_edges, density=True)

    # Normalize to proportions
    ref_pct = ref_hist / ref_hist.sum()
    cur_pct = cur_hist / cur_hist.sum()

    # Avoid division by zero
    ref_pct = np.clip(ref_pct, 1e-6, None)
    cur_pct = np.clip(cur_pct, 1e-6, None)

    psi = np.sum((cur_pct - ref_pct) * np.log(cur_pct / ref_pct))
    return psi

# PSI interpretation:
# < 0.1  → No significant shift
# 0.1-0.2 → Moderate shift, investigate
# > 0.2  → Significant shift, retrain
```

### Concept Drift Detection
```python
# Kolmogorov-Smirnov test for drift
from scipy import stats

def detect_drift_ks(reference_preds, current_preds, alpha=0.05):
    """KS test to detect concept drift in model predictions."""
    statistic, p_value = stats.ks_2samp(reference_preds, current_preds)
    return {
        "statistic": statistic,
        "p_value": p_value,
        "drift_detected": p_value < alpha,
        "severity": "high" if statistic > 0.2 else
                    "medium" if statistic > 0.1 else "low"
    }

# ADWIN (Adaptive Windowing) for streaming drift detection
# Maintains a variable-length window of recent observations
# Detects change when two sub-windows have distinct enough means
```

### Performance Degradation Monitoring
```python
# Model monitoring pipeline
class AIModelMonitor:
    def __init__(self, model_id, thresholds):
        self.model_id = model_id
        self.thresholds = thresholds
        # defaults:
        # {"accuracy_drop": 0.05, "psi_threshold": 0.2,
        #  "latency_p99_ms": 500, "error_rate": 0.01}

    def check_performance(self, predictions, actuals, window="24h"):
        metrics = {
            "accuracy": accuracy_score(actuals, predictions),
            "precision": precision_score(actuals, predictions, average="weighted"),
            "recall": recall_score(actuals, predictions, average="weighted"),
            "f1": f1_score(actuals, predictions, average="weighted"),
        }

        alerts = []
        for metric, value in metrics.items():
            baseline = self.get_baseline(metric)
            if baseline - value > self.thresholds.get(f"{metric}_drop", 0.05):
                alerts.append({
                    "metric": metric,
                    "baseline": baseline,
                    "current": value,
                    "severity": "critical" if baseline - value > 0.1 else "warning"
                })
        return {"metrics": metrics, "alerts": alerts}

    def check_data_drift(self, reference_features, current_features):
        drift_results = {}
        for feature in reference_features.columns:
            psi = calculate_psi(
                reference_features[feature],
                current_features[feature]
            )
            drift_results[feature] = {
                "psi": psi,
                "status": "drift" if psi > self.thresholds["psi_threshold"] else "stable"
            }
        return drift_results
```

## AI Incident Response
### AI Incident Classification
```
Severity 1 — Critical
  - Safety-impacting AI failure (autonomous vehicle, medical)
  - Widespread discriminatory decisions
  - Large-scale data breach via AI system
  - Adversarial attack causing financial loss > threshold
  Response: Immediate model shutdown, war room, regulatory notification

Severity 2 — High
  - Significant bias detected in production
  - Model drift causing measurable business impact
  - Prompt injection leading to data exposure
  - Model extraction attack detected
  Response: Model rollback within 4h, incident team assembled

Severity 3 — Medium
  - Performance degradation beyond SLA thresholds
  - Minor bias detected in non-critical system
  - Adversarial probing detected but not exploited
  Response: Investigate within 24h, patch within 72h

Severity 4 — Low
  - Cosmetic AI output issues
  - Minor drift within acceptable bounds
  - False positive adversarial detection
  Response: Log, trend, address in next maintenance window
```

### AI Incident Response Playbook
```
Phase 1: Detection & Triage (0-1h)
  ┌─ Automated alert received (monitoring, user report, audit)
  ├─ Classify severity (1-4)
  ├─ Identify affected model(s) and data
  ├─ Assess blast radius (users, decisions, downstream systems)
  └─ Activate response team

Phase 2: Containment (1-4h)
  ┌─ Model rollback to last known good version
  ├─ Enable fallback (rule-based system, human review)
  ├─ Isolate compromised training data
  ├─ Block adversarial input patterns
  └─ Preserve evidence (logs, model weights, input data)

Phase 3: Analysis (4-48h)
  ┌─ Root cause analysis (data? model? infrastructure? adversary?)
  ├─ Impact assessment (who was affected? what decisions were wrong?)
  ├─ Timeline reconstruction
  ├─ Identify remediation steps for affected individuals
  └─ Determine if regulatory notification required

Phase 4: Recovery (48h-2w)
  ┌─ Retrain/fine-tune model with corrected data
  ├─ Validate fix against original failure mode
  ├─ Gradual rollout with enhanced monitoring
  ├─ Remediate affected decisions (reverse, compensate)
  └─ Update monitoring to detect similar issues

Phase 5: Post-Incident (2w-4w)
  ┌─ Blameless post-mortem
  ├─ Update AI risk register
  ├─ Improve detection capabilities
  ├─ Update incident response playbook
  └─ Share lessons learned (internal, industry)
```

## Red Teaming for AI
### Red Team Methodology
```
Scope Definition:
  ┌─ In-scope attacks: prompt injection, jailbreaking, evasion,
  │   data extraction, bias exploitation
  ├─ Test environment: production shadow, staging, isolated
  ├─ Rules of engagement: no actual data exfiltration, rate limits
  └─ Success criteria: specific failure modes to test

Test Categories:
  1. Guardrail Bypass Testing
     - Direct jailbreaking attempts
     - Indirect prompt injection via context
     - Role-playing / persona exploitation
     - Encoding tricks (base64, Unicode, leetspeak)
     - Multi-turn escalation attacks

  2. Information Extraction
     - System prompt extraction
     - Training data memorization probing
     - PII extraction attempts
     - Model architecture fingerprinting

  3. Adversarial Robustness
     - Input perturbation (noise, rotation, typos)
     - Semantic-preserving adversarial examples
     - Edge case / boundary testing
     - Out-of-distribution input handling

  4. Bias and Fairness Probing
     - Protected attribute sensitivity testing
     - Stereotyping behavior evaluation
     - Disparate treatment across demographics
     - Cultural and linguistic bias detection

  5. Abuse Scenario Testing
     - Harmful content generation
     - Misinformation creation
     - Social engineering assistance
     - Malicious code generation
```

### Red Team Report Template
```markdown
# AI Red Team Assessment Report

## Executive Summary
- System Tested: [Model/Application Name]
- Assessment Period: [Dates]
- Overall Risk Rating: [Critical/High/Medium/Low]
- Key Findings: [Count] Critical, [Count] High, [Count] Medium

## Methodology
- Framework: NIST AI RMF + MITRE ATLAS
- Tools: [Garak, ART, TextAttack, custom scripts]
- Test Categories: [List]

## Findings

### Finding RT-001: [Title]
- Category: [Prompt Injection / Evasion / Data Extraction / ...]
- Severity: [Critical/High/Medium/Low]
- ATLAS Technique: [AML.TXXXX]
- Description: [What was found]
- Attack Vector: [How the attack works]
- Impact: [What could happen in production]
- Evidence: [Screenshots, logs, prompts used]
- Recommendation: [How to fix]
- Status: [Open/In Progress/Resolved]

## Risk Summary Matrix
| Category | Tests | Pass | Fail | Critical |
|----------|-------|------|------|----------|
| Prompt Injection | X | X | X | X |
| Data Extraction | X | X | X | X |
| Bias/Fairness | X | X | X | X |
| Robustness | X | X | X | X |
```

## AI Risk Register Template
### Risk Register Format
```
┌────────┬────────────────────┬──────────┬──────────┬──────────┐
│ Risk ID│ Risk Description   │ Category │Likelihood│ Impact   │
├────────┼────────────────────┼──────────┼──────────┼──────────┤
│AIR-001 │ Training data poi- │ Data     │ Medium   │ High     │
│        │ soned via supply   │ Integrity│          │          │
│        │ chain compromise   │          │          │          │
├────────┼────────────────────┼──────────┼──────────┼──────────┤
│AIR-002 │ Model drift causes │ Model    │ High     │ Medium   │
│        │ accuracy below SLA │ Perf.    │          │          │
├────────┼────────────────────┼──────────┼──────────┼──────────┤
│AIR-003 │ Prompt injection   │ Security │ High     │ High     │
│        │ bypasses content   │          │          │          │
│        │ filtering          │          │          │          │
├────────┼────────────────────┼──────────┼──────────┼──────────┤
│AIR-004 │ Bias in hiring     │ Fairness │ Medium   │ Critical │
│        │ model discrimin-   │          │          │          │
│        │ ates against pro-  │          │          │          │
│        │ tected classes     │          │          │          │
├────────┼────────────────────┼──────────┼──────────┼──────────┤
│AIR-005 │ LLM hallucinates   │ Reliab-  │ High     │ High     │
│        │ in customer-facing │ ility    │          │          │
│        │ application        │          │          │          │
└────────┴────────────────────┴──────────┴──────────┴──────────┘

Additional Columns per Risk:
  - Risk Score (Likelihood × Impact)
  - Current Controls (what exists today)
  - Residual Risk (after controls)
  - Risk Owner (person accountable)
  - Treatment Plan (accept, mitigate, transfer, avoid)
  - Target Date (for mitigation completion)
  - Status (open, in progress, mitigated, accepted)
  - Review Frequency (quarterly, monthly, continuous)
```

## See Also
- ai-compliance
- ai-security-architecture
- ai-privacy-trust
- ai-testing-assurance
- ai-supply-chain
- nist

## References
- NIST AI RMF 1.0: https://airc.nist.gov/AI_RMF
- OWASP Top 10 for LLMs: https://owasp.org/www-project-top-10-for-large-language-model-applications/
- MITRE ATLAS: https://atlas.mitre.org/
- ISO/IEC 23894:2023 — AI Risk Management
- NIST AI 100-2e2023 — Adversarial Machine Learning
