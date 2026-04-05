# AI Privacy and Trust

AI Privacy and Trust covers the privacy-enhancing technologies (PETs) that protect sensitive data throughout the AI lifecycle, the explainability methods that make model decisions interpretable, and the documentation and governance frameworks that build justified confidence in AI systems, from differential privacy and federated learning to model cards and trust calibration.

## Privacy-Enhancing Technologies (PETs) for AI
### PETs Landscape
```
┌────────────────────────────────────────────────────────────┐
│              Privacy-Enhancing Technologies                 │
├────────────────────┬───────────────────────────────────────┤
│ Data Minimization  │ Computation Protection                │
├────────────────────┼───────────────────────────────────────┤
│ Anonymization      │ Differential Privacy                  │
│ Pseudonymization   │ Secure Multi-Party Computation        │
│ Synthetic Data     │ Homomorphic Encryption                │
│ Data Masking       │ Trusted Execution Environments        │
│ Aggregation        │ Federated Learning                    │
│ k-Anonymity        │ Secure Enclaves                       │
│ l-Diversity        │ Zero-Knowledge Proofs                 │
│ t-Closeness        │ Private Information Retrieval         │
└────────────────────┴───────────────────────────────────────┘

Selection Matrix:
  Need                          → Recommended PET
  ─────────────────────────────────────────────────
  Aggregate statistics          → Differential Privacy
  Distributed training          → Federated Learning + DP
  Third-party model inference   → Homomorphic Encryption / TEE
  Multi-org data collaboration  → Secure MPC
  Dataset sharing               → Synthetic Data + k-Anonymity
  Model serving privacy         → TEE / Encrypted Inference
  Compliance (GDPR Art. 25)     → DP + Data Minimization
```

## Differential Privacy Implementation
### Practical DP Deployment
```python
# Differentially Private Query System
import numpy as np

class DPQueryEngine:
    """Answer aggregate queries with differential privacy guarantees."""

    def __init__(self, epsilon_budget, delta=1e-6):
        self.epsilon_budget = epsilon_budget
        self.epsilon_spent = 0.0
        self.delta = delta

    def dp_count(self, data, predicate, epsilon=None):
        """Count records matching predicate with DP noise."""
        epsilon = epsilon or self._auto_epsilon()
        true_count = sum(1 for d in data if predicate(d))
        sensitivity = 1  # Adding/removing one record changes count by 1
        noise = np.random.laplace(0, sensitivity / epsilon)
        self.epsilon_spent += epsilon
        return max(0, round(true_count + noise))

    def dp_mean(self, values, lower, upper, epsilon=None):
        """Compute mean with DP noise. Requires known bounds."""
        epsilon = epsilon or self._auto_epsilon()
        clipped = np.clip(values, lower, upper)
        n = len(values)
        true_mean = np.mean(clipped)
        sensitivity = (upper - lower) / n
        noise = np.random.laplace(0, sensitivity / epsilon)
        self.epsilon_spent += epsilon
        return true_mean + noise

    def dp_histogram(self, values, bins, epsilon=None):
        """Compute histogram with DP noise on each bin."""
        epsilon = epsilon or self._auto_epsilon()
        hist, edges = np.histogram(values, bins=bins)
        sensitivity = 1  # One record affects one bin by 1
        noise = np.random.laplace(0, sensitivity / epsilon, size=len(hist))
        self.epsilon_spent += epsilon
        return np.maximum(0, hist + noise).astype(int), edges

    def remaining_budget(self):
        return self.epsilon_budget - self.epsilon_spent

    def _auto_epsilon(self):
        """Allocate 10% of remaining budget per query."""
        return max(0.01, self.remaining_budget() * 0.1)
```

### DP-SGD for Model Training
```python
# DP-SGD Training with Opacus
import torch
from opacus import PrivacyEngine
from opacus.validators import ModuleValidator

# Step 1: Prepare model (replace incompatible layers)
model = ModuleValidator.fix(model)  # BatchNorm → GroupNorm, etc.
errors = ModuleValidator.validate(model, strict=False)
assert len(errors) == 0, f"Model not DP-compatible: {errors}"

# Step 2: Configure privacy parameters
MAX_GRAD_NORM = 1.0       # Clip individual gradients
EPSILON = 3.0             # Total privacy budget
DELTA = 1e-5              # Failure probability (< 1/n)
EPOCHS = 10
BATCH_SIZE = 256

# Step 3: Attach privacy engine
privacy_engine = PrivacyEngine()
model, optimizer, train_loader = privacy_engine.make_private_with_epsilon(
    module=model,
    optimizer=optimizer,
    data_loader=train_loader,
    epochs=EPOCHS,
    target_epsilon=EPSILON,
    target_delta=DELTA,
    max_grad_norm=MAX_GRAD_NORM,
)
# Noise multiplier is automatically calculated

# Step 4: Train normally
for epoch in range(EPOCHS):
    for batch in train_loader:
        optimizer.zero_grad()
        loss = criterion(model(batch.x), batch.y)
        loss.backward()
        optimizer.step()

    eps = privacy_engine.get_epsilon(delta=DELTA)
    print(f"Epoch {epoch}: epsilon={eps:.2f}, delta={DELTA}")
    if eps > EPSILON:
        print("Privacy budget exhausted!")
        break

# Key trade-offs:
# Lower epsilon → more noise → more privacy → less accuracy
# Larger batch size → better privacy/utility trade-off
# More epochs → higher epsilon (budget consumed)
# Lower max_grad_norm → more clipping → more bias but less noise needed
```

## Federated Learning
### FL Architecture and Protocols
```
FedAvg Protocol (McMahan et al.):

Round t:
  1. Server sends global model w_t to selected clients
  2. Each client k:
     a. Initializes local model: w_k = w_t
     b. Trains on local data for E epochs:
        w_k ← w_k - η∇L(w_k; D_k)
     c. Sends update Δw_k = w_k - w_t to server
  3. Server aggregates:
     w_{t+1} = w_t + (1/K) Σ_k (n_k/n) Δw_k
     where n_k = size of client k's data, n = total

Communication Pattern:
  ┌─────────────────────────────────────────────┐
  │  Round 1    Round 2    Round 3    ...        │
  │  ┌─┐        ┌─┐        ┌─┐                  │
  │  │S│──→C    │S│──→C    │S│──→C               │
  │  │ │←──C    │ │←──C    │ │←──C               │
  │  │ │──→C    │ │──→C    │ │──→C               │
  │  │ │←──C    │ │←──C    │ │←──C               │
  │  │ │──→C    │ │──→C    │ │──→C               │
  │  │ │←──C    │ │←──C    │ │←──C               │
  │  └─┘        └─┘        └─┘                  │
  │  S=Server, C=Client                          │
  └─────────────────────────────────────────────┘

FL Variants:
  Cross-Device FL:
    - Millions of mobile/IoT devices
    - Highly heterogeneous data (non-IID)
    - Intermittent connectivity
    - Example: keyboard prediction, health monitoring

  Cross-Silo FL:
    - Few organizations (2-100)
    - Reliable connections
    - Regulatory motivation (data sovereignty)
    - Example: multi-hospital medical AI, financial fraud detection
```

### FL Privacy Enhancements
```
Secure Aggregation:
  Purpose: Server learns only aggregate, not individual updates
  Protocol:
    1. Each pair of clients (i,j) agrees on a random mask s_ij
    2. Client i sends: Δw_i + Σ_{j>i} s_ij - Σ_{j<i} s_ji
    3. When server sums all masked updates, masks cancel out
    4. Server gets Σ_i Δw_i without seeing any individual Δw_i

  Handles client dropout via Shamir secret sharing of masks

Local Differential Privacy in FL:
  Each client adds noise to their update before sending:
    Δw_i^noisy = clip(Δw_i, C) + N(0, σ²C²I)

  where C is the clipping bound and σ is calibrated for target ε

  Total privacy guarantee per round:
    (ε, δ)-DP for each client's contribution

Compression + Privacy:
  - Gradient compression reduces communication AND attack surface
  - Top-k sparsification: send only largest k% of gradient entries
  - Quantization: reduce precision (32-bit → 1-8 bit)
  - Random rotation: compress in random subspace
```

## Secure Multi-Party Computation
### SMPC for ML Inference
```
Two-Party Computation (2PC) for Neural Network Inference:

Scenario: Client has private input x, Server has private model f
Goal: Client learns f(x), neither party learns the other's data

Protocol (Garbled Circuits + Secret Sharing):
  1. Linear layers: Beaver triple-based multiplication on shares
  2. ReLU/activation: Garbled circuits for comparison
  3. Final output: reconstruct only at client

Performance (approximate):
  ┌────────────────────┬───────────┬──────────┬──────────┐
  │ Model              │ Plaintext │ 2PC Time │ Overhead │
  ├────────────────────┼───────────┼──────────┼──────────┤
  │ Logistic Regression│ <1ms      │ ~50ms    │ 50x      │
  │ Small CNN (MNIST)  │ ~5ms      │ ~5s      │ 1000x    │
  │ ResNet-50          │ ~30ms     │ ~5min    │ 10000x   │
  │ BERT-base          │ ~50ms     │ ~30min   │ 36000x   │
  │ GPT-2 (117M)       │ ~200ms   │ hours    │ 10000x+  │
  └────────────────────┴───────────┴──────────┴──────────┘

  Current state: practical for small models, research-stage for LLMs
```

## Homomorphic Encryption for ML
### HE Schemes for AI
```
Partially Homomorphic Encryption (PHE):
  - RSA: multiplication only (m1 · m2)
  - Paillier: addition only (m1 + m2)
  - ElGamal: multiplication only
  - Use case: simple aggregation, voting, summation

Somewhat Homomorphic Encryption (SHE):
  - Limited number of multiplications + unlimited additions
  - Sufficient for low-degree polynomial computations
  - Use case: polynomial approximation of ML inference

Fully Homomorphic Encryption (FHE):
  - Arbitrary computation on encrypted data
  - Schemes: BFV (integers), BGV (integers), CKKS (approximate/floats)
  - Use case: encrypted ML training and inference
  - Major limitation: performance overhead

CKKS Scheme (best for ML):
  - Supports approximate arithmetic on encrypted real numbers
  - Operations: addition, multiplication, rotation
  - Noise grows with computation depth → bootstrapping required
  - Typical parameters: ring dimension 2^15, 128-bit security

Practical Encrypted Inference:
  Libraries: Microsoft SEAL, HElib, Lattigo, OpenFHE, Concrete
  Frameworks: Concrete-ML (Zama), TenSEAL, EVA
```

## Synthetic Data Generation
### Methods and Quality
```
Generation Methods:
  1. Statistical Methods
     - Copula-based: model marginal + dependency structure
     - Bayesian networks: learn conditional distributions
     - CTGAN: conditional tabular GAN
     - TVAE: tabular variational autoencoder

  2. Deep Learning Methods
     - GANs (Generative Adversarial Networks)
     - VAEs (Variational Autoencoders)
     - Diffusion Models
     - LLM-based generation (for text)

  3. Rule-Based Methods
     - Domain-specific rules + random sampling
     - Template-based with variable injection
     - Most controllable, least realistic

Quality Assessment:
  ┌──────────────────────┬──────────────────────────────────┐
  │ Metric               │ Description                      │
  ├──────────────────────┼──────────────────────────────────┤
  │ Statistical Fidelity │ Distribution similarity (KS, JS) │
  │ ML Utility           │ Train on synthetic, test on real │
  │ Privacy              │ Nearest-neighbor distance ratio  │
  │ Coverage             │ % of real data modes captured    │
  │ Novelty              │ % of synthetic not in real data  │
  │ Diversity            │ Variety within synthetic dataset │
  └──────────────────────┴──────────────────────────────────┘

Privacy Risk in Synthetic Data:
  - Synthetic data is NOT automatically private
  - GANs can memorize rare records
  - Must validate with membership inference tests
  - Best practice: combine with differential privacy
    (DP-GAN, PATE-GAN, DP-CTGAN)
```

## Data Anonymization for ML
### Anonymization Techniques
```
Technique          Privacy Level    Utility Impact    Reversible?
──────────────────────────────────────────────────────────────
Pseudonymization   Low (still PD)   Minimal          Yes (with key)
k-Anonymity        Medium           Moderate         Partially
l-Diversity        Medium-High      Moderate-High    No
t-Closeness        High             High             No
Differential Priv. Formal guarantee Variable         No
Data Masking       Medium           Variable         Depends
Tokenization       Medium           Minimal (if     Yes (with vault)
                                    format-pres.)
Generalization     Medium           Moderate         No
Suppression        High             High             No

For ML Specifically:
  - Anonymize training data BEFORE model training
  - Model itself may memorize and leak original data
  - Evaluate re-identification risk post-training
  - Consider: anonymized data may have reduced utility for ML
  - Best practice: anonymize + DP during training for defense-in-depth
```

## Explainability for Trust
### LIME (Local Interpretable Model-agnostic Explanations)
```
How LIME Works:
  1. Select instance x to explain
  2. Generate perturbed samples around x
  3. Get model predictions for perturbations
  4. Weight perturbations by proximity to x
  5. Fit interpretable model (linear, decision tree) on weighted samples
  6. Explanation = coefficients of interpretable model

Implementation:
  from lime.lime_tabular import LimeTabularExplainer

  explainer = LimeTabularExplainer(
      training_data=X_train,
      feature_names=feature_names,
      class_names=class_names,
      mode='classification'
  )

  explanation = explainer.explain_instance(
      data_row=x_test[0],
      predict_fn=model.predict_proba,
      num_features=10,
      num_samples=5000
  )
  explanation.show_in_notebook()

Output: Feature contributions (positive/negative) for the prediction
  e.g., "age > 50: +0.32, income < 30K: +0.28, employed=No: +0.15"
```

### SHAP (SHapley Additive exPlanations)
```
How SHAP Works:
  Based on Shapley values from cooperative game theory:
  - Each feature is a "player" in a "game" (prediction)
  - Shapley value = fair allocation of the prediction to features
  - Considers all possible feature coalitions

  φ_i = Σ_{S⊆N\{i}} [|S|!(|N|-|S|-1)! / |N|!] ×
        [f(S ∪ {i}) - f(S)]

  where S = subset of features, N = all features

SHAP Variants:
  ┌────────────────────┬─────────────────────────────────┐
  │ Variant            │ Best For                        │
  ├────────────────────┼─────────────────────────────────┤
  │ KernelSHAP         │ Any model (model-agnostic)      │
  │ TreeSHAP           │ Tree-based (XGBoost, RF, etc.)  │
  │ DeepSHAP           │ Deep learning models            │
  │ LinearSHAP         │ Linear models (exact, fast)     │
  │ PartitionSHAP      │ Text / image (hierarchical)     │
  └────────────────────┴─────────────────────────────────┘

Implementation:
  import shap

  # TreeSHAP for XGBoost
  explainer = shap.TreeExplainer(model)
  shap_values = explainer.shap_values(X_test)

  # Summary plot (global feature importance)
  shap.summary_plot(shap_values, X_test, feature_names=feature_names)

  # Force plot (single prediction explanation)
  shap.force_plot(explainer.expected_value, shap_values[0], X_test[0])

  # Dependence plot (feature interaction)
  shap.dependence_plot("age", shap_values, X_test)
```

### Attention Visualization (Transformers)
```
Methods:
  1. Raw Attention Weights
     - Visualize attention matrix from each head/layer
     - Shows which tokens attend to which
     - Caveat: attention ≠ explanation (may not reflect causal importance)

  2. Attention Rollout
     - Combine attention across layers multiplicatively
     - Better captures information flow through network
     - rollout = Π_{l} (0.5 * I + 0.5 * A_l)

  3. Gradient-weighted Attention
     - Weight attention by gradient of output w.r.t. attention
     - Better identifies causally relevant attention patterns

  4. Integrated Gradients
     - Model-agnostic, axiomatically justified
     - Interpolate from baseline to input, integrate gradients
     - IG_i = (x_i - x'_i) × ∫₀¹ (∂F(x' + α(x-x'))/∂x_i) dα

Implementation:
  # Using Captum for PyTorch
  from captum.attr import LayerIntegratedGradients

  lig = LayerIntegratedGradients(model, model.embedding_layer)
  attributions = lig.attribute(
      inputs=input_ids,
      baselines=baseline_ids,
      target=predicted_class
  )
  # attributions[i] = importance score for token i
```

## Model Documentation
### Model Cards (Mitchell et al. 2019)
```
Required Sections:
  1. Model Details
     - Organization, date, version, type, license
     - References to papers, documentation

  2. Intended Use
     - Primary intended uses
     - Primary intended users
     - Out-of-scope uses (explicitly stated)

  3. Factors
     - Relevant factors (demographics, environmental)
     - Evaluation factors (what was tested)

  4. Metrics
     - Model performance measures
     - Decision thresholds
     - Variation across factors (disaggregated metrics)

  5. Evaluation Data
     - Datasets used for evaluation
     - Preprocessing steps
     - Motivation for dataset selection

  6. Training Data
     - Datasets used for training
     - Preprocessing steps
     - Data collection methodology

  7. Quantitative Analyses
     - Unitary results (overall performance)
     - Intersectional results (across subgroups)

  8. Ethical Considerations
     - Sensitive use cases
     - Known biases and mitigations
     - Human life impact assessment

  9. Caveats and Recommendations
     - Known limitations
     - Ideal deployment conditions
     - Monitoring recommendations
```

### Datasheets for Datasets (Gebru et al. 2021)
```
Seven Question Categories:

1. Motivation
   - Why was the dataset created?
   - Who funded it?
   - What tasks was it designed for?

2. Composition
   - What are the instances? (images, text, records)
   - How many instances?
   - Does it contain confidential data?
   - Is it a sample? What's the sampling strategy?

3. Collection Process
   - How was data collected? (APIs, surveys, sensors)
   - Who collected it?
   - Was consent obtained?
   - What timeframe?

4. Preprocessing/Cleaning/Labeling
   - Was preprocessing applied?
   - Who did the labeling?
   - What quality control was used?

5. Uses
   - What has the dataset been used for?
   - What are appropriate uses?
   - What should it NOT be used for?

6. Distribution
   - How is the dataset distributed?
   - Under what license?
   - Are there export controls?

7. Maintenance
   - Who maintains the dataset?
   - How often is it updated?
   - How can errors be reported?
   - Will older versions be available?
```

## AI Trust Frameworks
### Trust Dimensions
```
Trustworthiness Pillars:

  1. Technical Robustness & Safety
     - Resilience to attack and error
     - Fallback plans and graceful degradation
     - Accuracy and reliability

  2. Privacy & Data Governance
     - Data protection (GDPR compliance)
     - Data quality and integrity
     - Access control and minimization

  3. Transparency
     - Traceability of decisions
     - Explainability of outputs
     - Communication about AI use

  4. Diversity, Non-discrimination & Fairness
     - Avoidance of unfair bias
     - Accessibility and universal design
     - Stakeholder participation

  5. Societal & Environmental Well-being
     - Environmental sustainability
     - Social impact assessment
     - Democratic values alignment

  6. Accountability
     - Auditability
     - Minimization of negative impact
     - Reporting and redress mechanisms

  7. Human Agency & Oversight
     - Human-in-the-loop capability
     - Ability to override AI decisions
     - User autonomy preservation
```

### Trust Calibration
```
Appropriate Trust = f(AI Capability, Context, Stakes)

Over-Trust (Automation Bias):
  - Accepting AI output without verification
  - Risk: errors propagated without catch
  - Mitigation: mandatory human review for high-stakes
  - Mitigation: show confidence intervals, not just predictions
  - Mitigation: occasional "challenge" exercises

Under-Trust (Algorithm Aversion):
  - Rejecting AI assistance even when beneficial
  - Risk: foregone value, inefficiency
  - Mitigation: demonstrate track record with metrics
  - Mitigation: gradual deployment (advisory → augmented → automated)
  - Mitigation: user education on AI capabilities/limitations

Calibrated Trust:
  - Trust proportional to demonstrated reliability
  - Different trust levels for different use cases
  - Regular recalibration as system evolves
  - Trust verified through ongoing monitoring

Trust Metrics:
  ┌─ User trust survey scores (pre/post deployment)
  ├─ Override rate (how often humans reject AI recommendations)
  ├─ Appropriate override rate (overrides that were correct)
  ├─ Reliance rate (AI recommendations followed)
  ├─ Task performance with vs. without AI
  └─ Time to decision with vs. without AI
```

## Transparency Reporting
### AI Transparency Report Template
```
Annual AI Transparency Report

1. AI System Inventory
   - Number and types of AI systems deployed
   - Risk classification of each system
   - Changes from previous reporting period

2. Performance Summary
   - Accuracy metrics by system
   - Fairness metrics by system and demographic group
   - Trend analysis (improving/degrading)

3. Incident Summary
   - Number and severity of AI incidents
   - Root causes and corrective actions
   - Lessons learned

4. Privacy Metrics
   - Privacy budget consumption (if using DP)
   - Data access requests fulfilled
   - Data deletion requests processed

5. Human Oversight Statistics
   - Human review rates
   - Override rates and reasons
   - Escalation patterns

6. Bias and Fairness
   - Bias audit results
   - Disparate impact analysis
   - Corrective actions taken

7. Third-Party AI
   - Vendor AI systems in use
   - Risk assessments performed
   - Monitoring findings

8. Forward-Looking Statements
   - Planned AI deployments
   - Known risks and mitigations
   - Regulatory compliance roadmap
```

## User Consent for AI
### Consent Framework
```
Consent Requirements by Jurisdiction:
  GDPR (Article 22):
    - Right not to be subject to solely automated decision-making
    - Explicit consent required for automated decisions with legal effects
    - Must provide: meaningful information about logic, significance, consequences
    - Right to human intervention, express views, contest decision

  EU AI Act (Article 50):
    - Must inform users they are interacting with AI
    - Must mark AI-generated content as such
    - No requirement for consent to AI use generally
    - Transparency is the primary obligation

Consent Design Patterns:
  ┌─ Clear disclosure: "This decision is assisted by AI"
  ├─ Purpose: "AI is used to [specific purpose]"
  ├─ Opt-out: ability to request human-only processing
  ├─ Explanation: right to meaningful explanation of AI decision
  ├─ Contest: ability to challenge and appeal AI decisions
  ├─ Granularity: per-purpose consent, not blanket
  └─ Withdrawal: easy withdrawal of consent with fallback

Implementation:
  □ Consent management platform integration
  □ Consent state checked before AI inference
  □ Fallback path for non-consenting users
  □ Consent records retained for audit
  □ Consent withdrawal triggers data/model review
```

## See Also
- ai-security-architecture
- ai-risk-management
- ai-compliance
- ai-testing-assurance
- gdpr
- llm-fundamentals

## References
- Dwork & Roth: The Algorithmic Foundations of Differential Privacy
- Mitchell et al.: Model Cards for Model Reporting (2019)
- Gebru et al.: Datasheets for Datasets (2021)
- NIST Privacy Framework: https://www.nist.gov/privacy-framework
- EU Ethics Guidelines for Trustworthy AI: https://digital-strategy.ec.europa.eu/en/library/ethics-guidelines-trustworthy-ai
- Opacus: https://opacus.ai/
- SHAP: https://shap.readthedocs.io/
- LIME: https://github.com/marcotcr/lime
