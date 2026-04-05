# AI Testing and Assurance

AI Testing and Assurance encompasses the validation, verification, and continuous evaluation strategies for AI systems, covering model performance metrics, bias and fairness testing, adversarial robustness assessment, production monitoring, and third-party audit frameworks that together provide evidence-based confidence in AI system behavior across its operational lifecycle.

## AI Testing Strategies
### Testing Pyramid for AI
```
                    ┌──────────┐
                    │  System  │  End-to-end integration tests
                    │  Tests   │  User acceptance testing
                    ├──────────┤  A/B tests in production
                   ╱            ╲
                  ╱  Integration  ╲  Pipeline tests
                 ╱    Tests        ╲ Data flow validation
                ╱                    ╲ API contract tests
               ├──────────────────────┤
              ╱                        ╲
             ╱     Component Tests      ╲  Model unit tests
            ╱                            ╲ Feature engineering tests
           ╱                              ╲ Data validation tests
          ├────────────────────────────────┤

Additional AI-Specific Testing Layers:
  ┌─ Data Tests: Schema, distribution, quality, bias
  ├─ Model Tests: Performance, fairness, robustness
  ├─ Infrastructure Tests: Serving latency, throughput, failover
  ├─ Monitoring Tests: Alert accuracy, drift detection coverage
  └─ Compliance Tests: Documentation, logging, oversight
```

### Test Matrix
```
┌──────────────────┬──────────┬──────────┬──────────┬──────────┐
│ Test Type        │ Pre-Train│ Post-    │ Pre-     │ Post-    │
│                  │          │ Train    │ Deploy   │ Deploy   │
├──────────────────┼──────────┼──────────┼──────────┼──────────┤
│ Data quality     │    ✓     │          │          │    ✓     │
│ Data bias        │    ✓     │          │          │    ✓     │
│ Unit tests       │    ✓     │    ✓     │    ✓     │          │
│ Performance      │          │    ✓     │    ✓     │    ✓     │
│ Fairness         │          │    ✓     │    ✓     │    ✓     │
│ Robustness       │          │    ✓     │    ✓     │          │
│ Security         │          │    ✓     │    ✓     │    ✓     │
│ Integration      │          │          │    ✓     │          │
│ A/B test         │          │          │          │    ✓     │
│ Stress test      │          │          │    ✓     │          │
│ Drift monitoring │          │          │          │    ✓     │
│ Red team         │          │    ✓     │    ✓     │    ✓     │
└──────────────────┴──────────┴──────────┴──────────┴──────────┘
```

## Model Validation
### Train/Test Split Strategies
```
Holdout Split:
  ┌──────────────────────────┬──────────┬──────────┐
  │     Training (70%)       │Val (15%) │Test (15%)│
  └──────────────────────────┴──────────┴──────────┘
  - Simplest approach
  - Risk: high variance if dataset is small
  - Use when dataset is large (>100K samples)

K-Fold Cross-Validation:
  Fold 1: [Test] [Train] [Train] [Train] [Train]
  Fold 2: [Train] [Test] [Train] [Train] [Train]
  Fold 3: [Train] [Train] [Test] [Train] [Train]
  Fold 4: [Train] [Train] [Train] [Test] [Train]
  Fold 5: [Train] [Train] [Train] [Train] [Test]

  - K=5 or K=10 is standard
  - Better estimate of generalization performance
  - Computationally expensive (K × training cost)
  - Final metric = mean ± std across folds

Stratified K-Fold:
  - Preserves class distribution in each fold
  - Essential for imbalanced datasets
  - Each fold has same class ratio as full dataset

Time-Series Split:
  Fold 1: [Train    ] [Test]
  Fold 2: [Train         ] [Test]
  Fold 3: [Train              ] [Test]
  - Never use future data to predict past
  - Expanding window or sliding window
  - Critical for temporal data (finance, forecasting)

Group K-Fold:
  - Ensures all samples from one group are in same fold
  - Prevents data leakage across correlated samples
  - Example: all images from one patient in same fold
```

### Validation Best Practices
```
Common Pitfalls:
  ✗ Data leakage: test data influences training
    ├─ Fitting scaler on full dataset, then splitting
    ├─ Feature selection using test labels
    └─ Temporal leakage (using future to predict past)

  ✗ Distribution mismatch: test ≠ production
    ├─ Test set not representative of real-world inputs
    ├─ Sampling bias in data collection
    └─ Domain shift between development and deployment

  ✗ Overfitting to test set: repeated evaluation
    ├─ Tuning hyperparameters on test set
    ├─ Cherry-picking best test result
    └─ Fix: use separate validation and test sets

Best Practices:
  ✓ Pipeline all preprocessing (fit on train only)
  ✓ Use separate holdout test set (never tune on it)
  ✓ Report confidence intervals, not just point estimates
  ✓ Validate on realistic data (not cleaned/curated)
  ✓ Test on adversarial and edge case data
  ✓ Document all validation procedures
```

## Performance Metrics
### Classification Metrics
```
Confusion Matrix:
                    Predicted
                 Positive  Negative
  Actual  Pos    TP         FN
          Neg    FP         TN

Core Metrics:
  Accuracy  = (TP + TN) / (TP + TN + FP + FN)
  Precision = TP / (TP + FP)        — "of predicted positive, how many correct?"
  Recall    = TP / (TP + FN)        — "of actual positive, how many found?"
  F1 Score  = 2 × (Precision × Recall) / (Precision + Recall)
  Specificity = TN / (TN + FP)      — "of actual negative, how many correct?"

When to Use Which:
  Metric        Best When
  ─────────────────────────────────────────────────────────
  Accuracy      Classes are balanced
  Precision     False positives are costly (spam, fraud alerts)
  Recall        False negatives are costly (disease, security)
  F1            Balance between precision and recall needed
  AUC-ROC       Threshold-independent evaluation
  AUC-PR        Imbalanced datasets (positive class is rare)

Multi-Class Extensions:
  Macro Average:    Mean of per-class metrics (equal weight per class)
  Weighted Average: Mean weighted by class frequency
  Micro Average:    Global TP/FP/FN (equivalent to accuracy for multi-class)
```

### AUC-ROC and AUC-PR
```
ROC Curve (Receiver Operating Characteristic):
  - X-axis: False Positive Rate (1 - Specificity)
  - Y-axis: True Positive Rate (Recall)
  - Each point = different classification threshold
  - AUC-ROC = area under this curve
  - Perfect: AUC = 1.0, Random: AUC = 0.5

  Interpretation:
    0.9-1.0: Excellent
    0.8-0.9: Good
    0.7-0.8: Fair
    0.6-0.7: Poor
    0.5-0.6: Fail (near random)

PR Curve (Precision-Recall):
  - X-axis: Recall
  - Y-axis: Precision
  - More informative for imbalanced datasets
  - AUC-PR is sensitive to performance on minority class
  - Random baseline = proportion of positive class

Implementation:
  from sklearn.metrics import (
      accuracy_score, precision_score, recall_score,
      f1_score, roc_auc_score, average_precision_score,
      confusion_matrix, classification_report
  )

  print(classification_report(y_true, y_pred, digits=4))
  roc_auc = roc_auc_score(y_true, y_prob)
  pr_auc = average_precision_score(y_true, y_prob)
```

### Regression Metrics
```
  MAE  = (1/n) Σ|y_i - ŷ_i|          — Mean Absolute Error
  MSE  = (1/n) Σ(y_i - ŷ_i)²        — Mean Squared Error
  RMSE = √MSE                        — Root Mean Squared Error
  MAPE = (1/n) Σ|y_i - ŷ_i|/|y_i|   — Mean Abs. Percentage Error
  R²   = 1 - Σ(y_i - ŷ_i)²/Σ(y_i - ȳ)²  — Coefficient of Determination

  MAE:  Robust to outliers, interpretable in original units
  MSE:  Penalizes large errors more heavily
  RMSE: Same units as target, penalizes large errors
  MAPE: Scale-independent, but undefined when y_i = 0
  R²:   Proportion of variance explained, 1.0 = perfect
```

## Bias Testing
### Fairness Metrics
```
Group Fairness Metrics:

  Demographic Parity (Statistical Parity):
    P(Ŷ=1 | A=0) = P(Ŷ=1 | A=1)
    "Equal positive prediction rates across groups"

  Equalized Odds:
    P(Ŷ=1 | Y=1, A=0) = P(Ŷ=1 | Y=1, A=1)  — Equal TPR
    P(Ŷ=1 | Y=0, A=0) = P(Ŷ=1 | Y=0, A=1)  — Equal FPR
    "Equal error rates across groups"

  Equal Opportunity:
    P(Ŷ=1 | Y=1, A=0) = P(Ŷ=1 | Y=1, A=1)
    "Equal TPR only (recall parity)"

  Predictive Parity:
    P(Y=1 | Ŷ=1, A=0) = P(Y=1 | Ŷ=1, A=1)
    "Equal precision across groups"

  Calibration:
    P(Y=1 | Ŷ=p, A=0) = P(Y=1 | Ŷ=p, A=1) = p
    "Predicted probabilities mean the same across groups"

Individual Fairness:
  d(f(x_1), f(x_2)) ≤ L · d(x_1, x_2)
  "Similar individuals receive similar predictions"
```

### Disparate Impact and Four-Fifths Rule
```
Disparate Impact Ratio:
  DIR = P(Ŷ=1 | A=protected) / P(Ŷ=1 | A=non-protected)

Four-Fifths (80%) Rule (EEOC):
  If DIR < 0.8, there is adverse impact
  (selection rate of protected < 80% of non-protected)

Example:
  Male applicants selected:    60/100 = 60%
  Female applicants selected:  40/100 = 40%
  DIR = 40% / 60% = 0.667 < 0.8 → ADVERSE IMPACT

Implementation:
  from fairlearn.metrics import (
      demographic_parity_difference,
      equalized_odds_difference,
      MetricFrame
  )

  # Compute disparate impact ratio
  def disparate_impact_ratio(y_true, y_pred, sensitive):
      protected = y_pred[sensitive == 1].mean()
      non_protected = y_pred[sensitive == 0].mean()
      return protected / non_protected if non_protected > 0 else 0

  dir = disparate_impact_ratio(y_true, y_pred, sensitive_feature)
  print(f"Disparate Impact Ratio: {dir:.3f}")
  print(f"Four-Fifths Rule: {'PASS' if dir >= 0.8 else 'FAIL'}")

  # Disaggregated metrics
  metric_frame = MetricFrame(
      metrics={"accuracy": accuracy_score, "selection_rate": selection_rate},
      y_true=y_true, y_pred=y_pred,
      sensitive_features=sensitive_feature
  )
  print(metric_frame.by_group)
  print(f"Difference: {metric_frame.difference()}")
```

### Bias Audit Workflow
```
Step 1: Identify Protected Attributes
  ┌─ Race/ethnicity
  ├─ Gender/sex
  ├─ Age
  ├─ Disability status
  ├─ Religion
  ├─ National origin
  └─ Other jurisdiction-specific attributes

Step 2: Disaggregate Performance Metrics
  ┌─ Compute accuracy, precision, recall, F1 per group
  ├─ Compute selection rates per group
  ├─ Compute error rates per group
  └─ Compute intersectional metrics (e.g., Black women)

Step 3: Apply Fairness Tests
  ┌─ Four-fifths rule (disparate impact)
  ├─ Demographic parity difference
  ├─ Equalized odds difference
  ├─ Calibration across groups
  └─ Statistical significance tests

Step 4: Root Cause Analysis
  ┌─ Training data representation by group
  ├─ Label bias (are labels themselves biased?)
  ├─ Feature bias (proxy features correlated with protected attributes)
  ├─ Measurement bias (different data quality by group)
  └─ Historical bias (real-world patterns reflected in data)

Step 5: Mitigation
  Pre-processing: Resampling, reweighting, data augmentation
  In-processing: Fairness constraints during training
  Post-processing: Threshold adjustment per group

Step 6: Document and Report
  ┌─ Metrics by group in model card
  ├─ Fairness constraints applied
  ├─ Remaining disparities and justification
  └─ Ongoing monitoring plan
```

## Robustness Testing
### Adversarial Robustness Testing
```
Test Categories:
  1. Perturbation Robustness
     - Add noise to inputs (Gaussian, salt-and-pepper)
     - Measure accuracy degradation vs. perturbation magnitude
     - Report: accuracy at ε = {0.01, 0.05, 0.1, 0.3}

  2. Adversarial Example Testing
     - Generate adversarial examples (FGSM, PGD, C&W)
     - Measure robust accuracy (% correct under attack)
     - Report: clean accuracy vs. robust accuracy gap

  3. Semantic Robustness
     - Apply meaning-preserving transformations
     - Text: typos, paraphrasing, synonym substitution
     - Image: rotation, scaling, brightness change
     - Report: accuracy under each transformation type

  4. Out-of-Distribution Detection
     - Feed inputs from different distribution
     - Model should indicate low confidence or abstain
     - Report: OOD detection AUROC

Tools:
  ┌─ ART (Adversarial Robustness Toolbox) — IBM
  ├─ Foolbox — comprehensive attack library
  ├─ TextAttack — NLP adversarial attacks
  ├─ CleverHans — reference implementations
  └─ RobustBench — adversarial robustness benchmarks
```

### Stress Testing
```
Stress Test Scenarios:

  Volume Stress:
    ├─ 10x normal inference load
    ├─ Sustained high throughput for 1 hour
    ├─ Measure: latency degradation, error rates
    └─ Pass criteria: p99 latency < 2x normal, error rate < 0.1%

  Input Stress:
    ├─ Maximum length inputs
    ├─ Minimum length inputs (empty/null)
    ├─ Unusual character encodings
    ├─ Very large numeric values
    └─ Pass criteria: graceful handling, no crashes

  Data Quality Stress:
    ├─ Missing features (10%, 30%, 50%)
    ├─ Corrupted values (noise injection)
    ├─ Shifted distributions (future data)
    └─ Pass criteria: documented degradation curve

  Infrastructure Stress:
    ├─ Model serving node failure (kill 1 of N)
    ├─ GPU memory exhaustion
    ├─ Network partition
    └─ Pass criteria: automatic failover, no data loss
```

## A/B Testing for Models
### Experiment Design
```
A/B Test Setup:
  Control (A): Current production model
  Treatment (B): New candidate model

  Traffic Allocation:
    Phase 1 (1-2 days):  1% to treatment (canary)
    Phase 2 (3-7 days):  10% to treatment (validation)
    Phase 3 (1-2 weeks): 50% to treatment (full test)
    Phase 4: 100% rollout or rollback

  Key Metrics:
    Primary: The business metric you're optimizing
      (CTR, conversion, revenue, accuracy, user satisfaction)
    Guardrail: Metrics that must not degrade
      (latency, error rate, fairness metrics)
    Secondary: Additional metrics for insight
      (engagement, time-on-task, support tickets)

  Statistical Requirements:
    Significance level (α): 0.05 (5% false positive rate)
    Power (1-β): 0.80 (80% chance of detecting real effect)
    MDE (Min. Detectable Effect): Define before experiment
    Sample size: n = (Z_α/2 + Z_β)² × 2σ² / MDE²

  Decision Framework:
    ┌─ Primary metric significantly better → SHIP
    ├─ Primary metric neutral, guardrails OK → EXTEND test
    ├─ Any guardrail metric degraded → DO NOT SHIP
    └─ Primary metric significantly worse → ROLLBACK
```

### Shadow Testing (Dark Launch)
```
Shadow Mode:
  - New model receives production traffic
  - Responses NOT served to users
  - Responses logged for offline comparison
  - Zero risk to users during evaluation

  ┌──────┐     ┌───────────┐     ┌──────────┐
  │ User │────→│ Router    │────→│ Model A  │──→ Response
  │      │     │           │     │ (prod)   │     (served)
  └──────┘     │           │     └──────────┘
               │           │     ┌──────────┐
               │           │────→│ Model B  │──→ Response
               │           │     │ (shadow) │     (logged)
               └───────────┘     └──────────┘

Benefits:
  - Test on real production traffic
  - No user impact
  - Catch issues before exposure
  - Build confidence with real data
```

## Model Monitoring in Production
### Monitoring Dashboard
```
Key Monitoring Signals:

  Performance Metrics (real-time):
    ├─ Prediction accuracy (if labels available)
    ├─ Prediction confidence distribution
    ├─ Prediction latency (p50, p95, p99)
    ├─ Throughput (requests/second)
    └─ Error rate

  Data Quality (hourly/daily):
    ├─ Feature distribution statistics
    ├─ Missing value rates
    ├─ Data drift scores (PSI per feature)
    ├─ Schema violations
    └─ Input volume patterns

  Fairness (daily/weekly):
    ├─ Disparate impact ratio by group
    ├─ Error rate parity
    ├─ Selection rate by group
    └─ Intersectional metrics

  Operational (real-time):
    ├─ GPU/CPU utilization
    ├─ Memory usage
    ├─ Queue depth
    ├─ Model version deployed
    └─ Serving infrastructure health

Alert Thresholds:
  ┌─ Accuracy drops > 5% from baseline → WARNING
  ├─ Accuracy drops > 10% from baseline → CRITICAL
  ├─ PSI > 0.2 for any feature → WARNING
  ├─ PSI > 0.5 for any feature → CRITICAL
  ├─ Fairness metric out of range → CRITICAL
  ├─ p99 latency > 2x SLA → WARNING
  └─ Error rate > 1% → CRITICAL
```

## AI Auditing Frameworks
### Audit Standards
```
IEEE 2894-2024: Guide for AI and Automated Decision Fairness
  - Covers fairness measurement, mitigation, documentation
  - Applicable to classification and recommendation systems
  - Provides fairness metric selection guidance

NIST AI RMF 1.0: AI Risk Management Framework
  - Four functions: Govern, Map, Measure, Manage
  - Trustworthiness characteristics as audit criteria
  - AI risk assessment methodology

ISO/IEC 42001:2023: AI Management System
  - Certifiable standard for AI governance
  - 39 Annex A controls across 9 domains
  - Compatible with ISO 27001, ISO 9001

ISO/IEC TR 24027:2021: Bias in AI Systems
  - Taxonomy of bias sources
  - Measurement approaches
  - Mitigation techniques

ALTAI (Assessment List for Trustworthy AI):
  - Self-assessment questionnaire from EU HLEG
  - 7 key requirements with specific questions
  - Designed for organizational self-evaluation
```

### Third-Party AI Assessment
```
Assessment Scope:
  ┌─ Model governance and documentation review
  ├─ Training data audit (provenance, bias, quality)
  ├─ Performance evaluation (independent test data)
  ├─ Fairness assessment (disaggregated metrics)
  ├─ Robustness testing (adversarial, perturbation)
  ├─ Security assessment (adversarial ML, API security)
  ├─ Privacy assessment (membership inference, extraction)
  ├─ Explainability evaluation (explanation fidelity)
  └─ Compliance mapping (regulation → evidence)

Assessor Qualifications:
  ├─ Domain expertise (ML, statistics, domain-specific)
  ├─ Independence (no conflicts of interest)
  ├─ Methodological rigor (documented procedures)
  └─ Regulatory knowledge (applicable laws/standards)

Assessment Report:
  1. Executive summary with risk rating
  2. Scope and methodology
  3. Findings by category with severity
  4. Evidence documentation
  5. Recommendations with priority
  6. Management response and remediation plan
  7. Follow-up timeline
```

## Continuous Validation
### Production Validation Pipeline
```python
# Continuous validation framework
class ContinuousValidator:
    def __init__(self, model_id, config):
        self.model_id = model_id
        self.validators = [
            PerformanceValidator(
                min_accuracy=config.min_accuracy,
                max_degradation=config.max_degradation
            ),
            FairnessValidator(
                protected_attributes=config.protected_attrs,
                max_disparity=config.max_disparity,
                metric="equalized_odds"
            ),
            DriftValidator(
                reference_data=config.reference_data,
                psi_threshold=config.psi_threshold,
                ks_alpha=config.ks_alpha
            ),
            RobustnessValidator(
                perturbation_budget=config.epsilon,
                min_robust_accuracy=config.min_robust_accuracy
            ),
        ]

    def validate(self, predictions, actuals, features, sensitive_attrs):
        results = {}
        overall_pass = True
        for validator in self.validators:
            result = validator.check(
                predictions, actuals, features, sensitive_attrs
            )
            results[validator.name] = result
            if result.status == "FAIL":
                overall_pass = False
                self.alert(validator.name, result)
        return ValidationReport(
            model_id=self.model_id,
            timestamp=datetime.utcnow(),
            overall_pass=overall_pass,
            results=results
        )

    def alert(self, validator_name, result):
        """Send alert for validation failure."""
        severity = "CRITICAL" if result.severity == "high" else "WARNING"
        send_alert(
            channel="ai-monitoring",
            severity=severity,
            message=f"{self.model_id}: {validator_name} failed - {result.message}"
        )
```

## See Also
- ai-risk-management
- ai-compliance
- ai-security-architecture
- ai-privacy-trust
- ai-supply-chain

## References
- Fairlearn: https://fairlearn.org/
- AIF360 (AI Fairness 360): https://aif360.mybluemix.net/
- ART (Adversarial Robustness Toolbox): https://adversarial-robustness-toolbox.readthedocs.io/
- IEEE 2894-2024: https://standards.ieee.org/ieee/2894/10854/
- NIST AI RMF: https://airc.nist.gov/AI_RMF
- Google ML Testing Guide: https://research.google/pubs/pub46555/
- Scikit-learn Metrics: https://scikit-learn.org/stable/modules/model_evaluation.html
