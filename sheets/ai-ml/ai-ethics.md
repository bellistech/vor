# AI Ethics (Fairness, Bias, Explainability, and Responsible AI)

Practical reference for AI ethics principles, bias detection and mitigation, explainability methods, transparency reporting, and ethical AI design processes.

## AI Ethics Principles

### Core Principles

```text
Principle           Description                           Implementation
Fairness            Equitable treatment across groups     Bias testing, fairness constraints
Transparency        Openness about AI capabilities/limits Model cards, system documentation
Accountability      Clear responsibility for outcomes     Ownership, audit trails, oversight
Privacy             Protection of personal information    Data minimization, differential privacy
Safety              Prevention of harm                    Testing, monitoring, kill switches
Beneficence         AI should benefit individuals/society Impact assessment, stakeholder input
Non-maleficence     AI should not cause harm              Risk assessment, prohibited use policies
Autonomy            Preserve human agency and choice      Human-in-the-loop, opt-out options
Justice             Fair distribution of benefits/burdens Equitable access, inclusive design
Dignity             Respect for human dignity             No deception, manipulation, or exploitation
```

### Responsible AI Framework

```text
Responsible AI Lifecycle:

  Define    →  What problem are we solving? Who benefits? Who could be harmed?
  Design    →  What data, model, and UX choices minimize harm?
  Develop   →  Implement fairness constraints, bias testing, documentation
  Deploy    →  Human oversight, monitoring, staged rollout
  Monitor   →  Continuous fairness, performance, and drift tracking
  Improve   →  Feedback loops, incident response, model updates
  Retire    →  Graceful decommission, impact assessment of removal

At Each Stage Ask:
  - Who are the stakeholders affected?
  - What are the potential harms?
  - What are the fairness implications?
  - Is the system sufficiently transparent?
  - Are accountability mechanisms in place?
  - Is there meaningful human oversight?
```

## Bias Types

### Bias Taxonomy

```text
Data Bias:

  Selection Bias        -- training data does not represent target population
    Example:            Hiring model trained only on tech industry resumes
    Detection:          Compare training distribution to population demographics
    Mitigation:         Stratified sampling, oversampling, data augmentation

  Measurement Bias      -- features or labels measured differently across groups
    Example:            Credit scores systematically lower for minorities
                        due to historical discrimination
    Detection:          Analyze feature distributions across groups
    Mitigation:         Use proxy-free features, audit measurement process

  Representation Bias   -- underrepresentation of certain groups in training data
    Example:            Face recognition trained mostly on lighter-skinned faces
    Detection:          Demographic breakdown of training data
    Mitigation:         Balanced datasets, targeted data collection

  Historical Bias       -- training data reflects past societal discrimination
    Example:            Word embeddings encoding "nurse=female, doctor=male"
    Detection:          Bias probes on embeddings, association tests
    Mitigation:         Debiasing techniques, counterfactual data augmentation

  Label Bias            -- systematic errors in ground truth labels
    Example:            Recidivism labels reflect biased policing, not actual crime
    Detection:          Inter-annotator agreement, label audit
    Mitigation:         Multiple annotators, bias-aware labeling guidelines

Algorithmic Bias:

  Aggregation Bias      -- model fails to account for subgroup differences
    Example:            Single diabetes model for all ethnicities
                        (HbA1c thresholds differ by race)
    Detection:          Subgroup performance analysis
    Mitigation:         Subgroup-specific models, stratified evaluation

  Learning Bias         -- model amplifies patterns in biased training data
    Example:            Image captioning models amplifying gender stereotypes
    Detection:          Compare bias metrics in data vs model outputs
    Mitigation:         Regularization, adversarial debiasing, constrained learning

  Evaluation Bias       -- evaluation dataset not representative of deployment
    Example:            Benchmark accuracy high but fails on underrepresented users
    Detection:          Disaggregated evaluation across demographics
    Mitigation:         Representative evaluation datasets, real-world testing

Deployment Bias:

  Automation Bias       -- users over-trust AI recommendations
    Example:            Doctors accepting AI diagnosis without independent review
    Detection:          User studies measuring over-reliance
    Mitigation:         Confidence display, mandatory human review thresholds

  Feedback Loop Bias    -- biased outputs influence future training data
    Example:            Predictive policing → more police → more arrests → confirms model
    Detection:          Monitor for self-reinforcing patterns
    Mitigation:         Break feedback loops, use external data sources

  Population Drift Bias -- model deployed on different population than training
    Example:            US-trained model deployed in India without adaptation
    Detection:          Monitor input distribution vs training distribution
    Mitigation:         Local adaptation, drift monitoring, retraining
```

## Bias Detection Metrics

### Group Fairness Metrics

```text
Notation:
  A = protected attribute (e.g., race, gender)
  Y = true outcome (ground truth label)
  Y_hat = predicted outcome (model prediction)
  S = model score (probability or risk score)

Demographic Parity (Statistical Parity):
  Definition:  P(Y_hat=1 | A=0) = P(Y_hat=1 | A=1)
  Meaning:     Positive prediction rate is equal across groups
  Limitation:  Ignores actual outcome rates (may conflict with accuracy)
  Use when:    Outcomes themselves may be biased (e.g., hiring decisions)

Equalized Odds:
  Definition:  P(Y_hat=1 | Y=1, A=0) = P(Y_hat=1 | Y=1, A=1)  AND
               P(Y_hat=1 | Y=0, A=0) = P(Y_hat=1 | Y=0, A=1)
  Meaning:     Equal TPR and FPR across groups
  Limitation:  Requires unbiased ground truth labels
  Use when:    Ground truth is reliable, want equal error rates

Equal Opportunity:
  Definition:  P(Y_hat=1 | Y=1, A=0) = P(Y_hat=1 | Y=1, A=1)
  Meaning:     Equal true positive rate (sensitivity) across groups
  Limitation:  Does not constrain false positive rate
  Use when:    False negatives are the primary harm (e.g., loan approval)

Predictive Parity:
  Definition:  P(Y=1 | Y_hat=1, A=0) = P(Y=1 | Y_hat=1, A=1)
  Meaning:     Equal precision (PPV) across groups
  Limitation:  May not equalize error rates
  Use when:    Positive predictions trigger interventions (e.g., medical screening)

Calibration:
  Definition:  P(Y=1 | S=s, A=0) = P(Y=1 | S=s, A=1) for all scores s
  Meaning:     Same score means same probability regardless of group
  Limitation:  Can be calibrated but still discriminatory in other metrics
  Use when:    Score interpretation must be group-independent (e.g., risk scores)

Disparate Impact Ratio (Four-Fifths Rule):
  Definition:  min(P(Y_hat=1|A=0), P(Y_hat=1|A=1)) /
               max(P(Y_hat=1|A=0), P(Y_hat=1|A=1))
  Threshold:   Ratio < 0.8 indicates adverse impact (EEOC guideline)
  Use when:    Employment, credit, housing decisions (US regulatory context)
```

### Individual Fairness Metrics

```text
Individual Fairness (Dwork et al., 2012):
  Definition:  Similar individuals should receive similar predictions
  Formal:      d(f(x_i), f(x_j)) <= L * d(x_i, x_j)
               where d = distance metric, f = model, L = Lipschitz constant
  Challenge:   Defining appropriate similarity metric is domain-specific

Counterfactual Fairness (Kusner et al., 2017):
  Definition:  Prediction does not change if protected attribute were different
  Formal:      P(Y_hat_A=a | X=x, A=a) = P(Y_hat_A=a' | X=x, A=a)
  Meaning:     Same prediction in a counterfactual world with different A
  Challenge:   Requires causal model of data generating process
```

## Bias Mitigation

### Pre-Processing (Data-Level)

```text
Technique              Description                        When to Use
Resampling             Over/under-sample to balance        Simple class/group imbalance
Reweighting            Assign weights to equalize          Cannot change data, can weight
Data augmentation      Generate synthetic minority data    Insufficient minority examples
Feature transformation Remove protected info from features When features proxy for protected
Relabeling             Correct labels near decision        Label bias suspected
Causal debiasing       Remove causal effect of protected   Causal structure is known
```

### In-Processing (Algorithm-Level)

```text
Technique                Description                       Library/Implementation
Adversarial debiasing    Train adversary to predict A       AIF360, Fairlearn
                         from predictions; model learns
                         to be uninformative about A
Constrained optimization Add fairness constraint to loss    Fairlearn (ExponentiatedGradient)
Prejudice remover        Add discrimination-aware           AIF360
                         regularization to loss function
Fair representation      Learn fair latent representation   AIF360 (LFR)
Meta-learning            Learn to be fair across tasks      Research implementations
```

### Post-Processing (Output-Level)

```text
Technique              Description                        Library/Implementation
Threshold adjustment   Different decision thresholds       AIF360, Fairlearn
                       per group to equalize metric        (ThresholdOptimizer)
Calibrated equalized   Adjust scores to achieve            AIF360 (CalibratedEqOdds)
  odds                 equalized odds post-hoc
Reject option          Defer to human when model is        AIF360 (RejectOptionClassifier)
                       uncertain near decision boundary
Score transformation   Map scores to achieve fairness      Custom implementation
```

### Fairness Libraries

```bash
# Fairlearn (Microsoft) -- Python
pip install fairlearn
# Key classes:
#   MetricFrame -- disaggregated metrics by group
#   ExponentiatedGradient -- in-processing constrained optimization
#   ThresholdOptimizer -- post-processing threshold adjustment
#   GridSearch -- sweep over fairness-constrained models

# AI Fairness 360 (IBM) -- Python
pip install aif360
# Key classes:
#   BinaryLabelDataset -- fairness-aware dataset
#   DisparateImpactRemover -- pre-processing
#   PrejudiceRemover -- in-processing
#   CalibratedEqOddsPostprocessing -- post-processing

# What-If Tool (Google) -- TensorFlow / notebook
pip install witwidget
# Interactive visualization of fairness metrics

# Aequitas (U Chicago) -- Python
pip install aequitas
# Bias audit toolkit with group metrics
```

## Explainability Methods

### LIME (Local Interpretable Model-agnostic Explanations)

```text
How LIME Works:
  1. Select instance to explain
  2. Generate perturbed samples around the instance
  3. Get model predictions for perturbed samples
  4. Weight samples by proximity to original instance
  5. Fit interpretable model (linear regression) on weighted samples
  6. Interpretable model coefficients = feature importances

Strengths:
  - Model-agnostic (works with any black-box model)
  - Local fidelity (accurate near the instance)
  - Human-interpretable output (feature weights)

Limitations:
  - Explanations can be unstable (run-to-run variation)
  - Perturbation approach may not respect feature correlations
  - Local ≠ global (explanation may not generalize)
  - No causal claims (correlational only)
```

```bash
# LIME usage (Python)
pip install lime

# Tabular explanation:
# from lime.lime_tabular import LimeTabularExplainer
# explainer = LimeTabularExplainer(X_train, feature_names=features)
# explanation = explainer.explain_instance(x_test[0], model.predict_proba)
# explanation.show_in_notebook()

# Text explanation:
# from lime.lime_text import LimeTextExplainer
# explainer = LimeTextExplainer(class_names=['neg', 'pos'])
# explanation = explainer.explain_instance(text, model.predict_proba)
```

### SHAP (SHapley Additive exPlanations)

```text
How SHAP Works:
  Based on Shapley values from cooperative game theory.
  Each feature's contribution = average marginal contribution
  across all possible feature coalitions.

  Shapley value for feature i:
    phi_i = sum over S (subset of features without i):
      |S|! * (|F|-|S|-1)! / |F|! * [f(S union {i}) - f(S)]

  where F = all features, f = model prediction function

SHAP Variants:
  KernelSHAP     -- model-agnostic, uses weighted regression (like LIME)
  TreeSHAP       -- exact, fast for tree-based models (XGBoost, RF, LightGBM)
  DeepSHAP       -- approximation for deep learning (DeepLIFT + Shapley)
  LinearSHAP     -- exact for linear models
  PartitionSHAP  -- hierarchical feature grouping

Strengths:
  - Solid theoretical foundation (Shapley axioms: efficiency, symmetry, etc.)
  - Local AND global explanations (aggregate local → global)
  - Consistent (if feature impact increases, SHAP value does not decrease)
  - Interaction effects detectable

Limitations:
  - Computationally expensive for many features (KernelSHAP)
  - Assumes feature independence (can be addressed with conditional SHAP)
  - Explanation complexity grows with feature count
```

```bash
# SHAP usage (Python)
pip install shap

# TreeSHAP for XGBoost:
# import shap
# explainer = shap.TreeExplainer(xgb_model)
# shap_values = explainer.shap_values(X_test)
# shap.summary_plot(shap_values, X_test)         # global importance
# shap.force_plot(explainer.expected_value,       # single prediction
#                 shap_values[0], X_test.iloc[0])
# shap.dependence_plot("feature_name", shap_values, X_test)  # interaction
```

### Other Explainability Methods

```text
Counterfactual Explanations:
  Question:    "What minimal change to input would change the prediction?"
  Example:     "If your income were $5K higher, the loan would be approved"
  Advantage:   Actionable, intuitive for end users
  Libraries:   DiCE (Microsoft), Alibi

Attention Visualization (for Transformers):
  Question:    "Which input tokens did the model attend to?"
  Methods:     Attention weight visualization, attention rollout
  Caveat:      Attention ≠ explanation (attention may not indicate causal importance)
  Libraries:   BertViz, Captum (PyTorch)

Integrated Gradients:
  Question:    "How much does each feature contribute vs a baseline?"
  Method:      Accumulate gradients along path from baseline to input
  Advantage:   Satisfies sensitivity and implementation invariance axioms
  Libraries:   Captum (PyTorch), tf-explain

Concept-Based Explanations (TCAV):
  Question:    "Does the model use human-understandable concepts?"
  Method:      Test with Concept Activation Vectors
  Example:     "Model uses 'striped' concept to classify zebras"
  Libraries:   TCAV (Google)

Anchors:
  Question:    "What sufficient conditions guarantee this prediction?"
  Method:      Find minimal feature conditions that anchor the prediction
  Example:     "If income > $50K AND employed > 2 years → approved (95% confidence)"
  Libraries:   Alibi (SeldonIO)
```

## Transparency Reporting

### AI Transparency Report Template

```text
1. System Overview
   - System name and purpose
   - Deployment scope (users, geography, scale)
   - Level of automation (advisory, semi-autonomous, autonomous)

2. Capabilities and Limitations
   - What the system can reliably do
   - Known failure modes and edge cases
   - Domains where the system should NOT be used
   - Confidence levels and uncertainty communication

3. Data Summary
   - Training data sources (high-level, not proprietary details)
   - Demographic representation in training data
   - Data freshness and update frequency
   - Privacy protections applied

4. Performance Metrics
   - Overall accuracy/performance metrics
   - Performance disaggregated by relevant subgroups
   - Comparison to human performance (if applicable)
   - Trend over time

5. Fairness Assessment
   - Fairness metrics evaluated
   - Results across protected groups
   - Mitigation measures applied
   - Remaining disparities and rationale for acceptance

6. Human Oversight
   - How humans interact with the system
   - Override and appeal mechanisms
   - Escalation procedures
   - Training provided to operators

7. Incident Report Summary
   - Number and nature of incidents in reporting period
   - Corrective actions taken
   - Changes to system based on incidents
```

## Human Oversight

### Human-in-the-Loop Design Patterns

```text
Pattern                  Description                    Use Case
Human-in-the-loop (HITL) Human approves every decision  High-stakes: medical, criminal justice
Human-on-the-loop (HOTL) Human monitors and can         Medium-stakes: content moderation,
                         intervene                      loan pre-screening
Human-over-the-loop      Human designs constraints,     Lower-stakes: recommendations,
                         AI operates within bounds      search ranking

Override Mechanisms:
  Kill switch            -- immediately disable AI system
  Score threshold        -- human review when score is uncertain (e.g., 0.4-0.6)
  Random audit           -- sample N% of decisions for human review
  Appeal process         -- affected individuals can request human review
  Escalation             -- complex/edge cases auto-escalated to human
  Time-limited autonomy  -- AI operates for X hours, then human checkpoint
```

## Ethical AI Design Process

### Ethics by Design Methodology

```text
Phase 1: Ethical Pre-Assessment
  [ ] Define the problem -- is AI the right solution?
  [ ] Identify stakeholders (direct users, affected populations, society)
  [ ] Map potential harms (individual, group, societal)
  [ ] Check against prohibited use cases
  [ ] Conduct preliminary fairness analysis
  [ ] Document ethical considerations and decisions

Phase 2: Inclusive Design
  [ ] Diverse team composition (gender, race, discipline, background)
  [ ] Stakeholder consultation (especially affected communities)
  [ ] Adversarial persona development ("who could be harmed?")
  [ ] Accessibility review (disability, language, literacy)
  [ ] Cultural sensitivity review

Phase 3: Development with Safeguards
  [ ] Bias testing at each development milestone
  [ ] Fairness constraints in model training
  [ ] Explainability implemented and tested
  [ ] Privacy by design (minimization, anonymization)
  [ ] Security testing (adversarial robustness)
  [ ] Documentation: model card, datasheet

Phase 4: Pre-Deployment Review
  [ ] Ethics board review (if high-risk)
  [ ] AI impact assessment completed
  [ ] Red team / adversarial testing
  [ ] User testing with diverse participants
  [ ] Monitoring plan defined (metrics, thresholds, alerts)
  [ ] Rollback plan documented

Phase 5: Continuous Ethical Monitoring
  [ ] Fairness metrics monitored in production
  [ ] Feedback channels for affected individuals
  [ ] Incident response process for ethical issues
  [ ] Regular re-assessment (quarterly minimum)
  [ ] Public transparency reporting (where appropriate)
```

## AI Impact on Society

### Key Societal Considerations

```text
Labor Market:
  - Job displacement in automatable roles
  - New job creation in AI-adjacent fields
  - Skill transition and reskilling needs
  - Wage impacts and inequality effects
  - Gig economy and algorithmic management

Information Ecosystem:
  - Deepfakes and synthetic media
  - Automated disinformation at scale
  - Content moderation challenges
  - Filter bubbles and polarization
  - Trust erosion in media and institutions

Power Dynamics:
  - Concentration of AI capabilities in few organizations
  - Surveillance capabilities and civil liberties
  - Digital divide (who benefits, who is harmed)
  - Global AI governance gaps
  - Environmental cost of large-scale AI training

Autonomy and Agency:
  - Behavioral nudging and manipulation
  - Algorithmic decision-making in consequential domains
  - Erosion of human skills through over-reliance
  - Informed consent for AI interactions
  - Right to human alternative
```

## See Also

- AI Governance
- Privacy Regulations
- Security Awareness

## References

- Mehrabi, N. et al. (2021): A Survey on Bias and Fairness in Machine Learning
- Ribeiro, M. et al. (2016): "Why Should I Trust You?" Explaining the Predictions of Any Classifier (LIME)
- Lundberg, S. & Lee, S. (2017): A Unified Approach to Interpreting Model Predictions (SHAP)
- Dwork, C. et al. (2012): Fairness Through Awareness
- Kusner, M. et al. (2017): Counterfactual Fairness
- Mitchell, M. et al. (2019): Model Cards for Model Reporting
- AI Fairness 360 (IBM): https://aif360.mybluemix.net
- Fairlearn (Microsoft): https://fairlearn.org
- NIST AI RMF 1.0 (January 2023)
