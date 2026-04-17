# The Mathematics of the EU AI Act — Risk Quantification, Model Evaluation, and Fairness Bounds

> *The AI Act's risk-based architecture is fundamentally mathematical: risk classification is a decision under uncertainty, model robustness is bounded by adversarial geometry, fairness obligations translate to statistical parity constraints, and systemic-risk thresholds (10^25 FLOPs) demarcate regulatory tiers with dimensional analysis. Compliance engineering is applied mathematics.*

---

## 1. Systemic Risk Threshold — Compute Dimensional Analysis

### The Problem

Art. 51 defines GPAI systemic risk at cumulative training compute $> 10^{25}$ FLOP. What does this number *mean* in hardware and time terms, and how far are current frontier models above or below it?

### The Formula

Total training compute:

$$C = 6 \cdot N \cdot D$$

Where $N$ is model parameters, $D$ is training tokens, and 6 is the standard forward+backward factor for transformer training (Hoffmann et al., 2022).

Hardware utilization model:

$$C = \text{FLOPS}_{\text{peak}} \cdot \eta \cdot t \cdot n_{\text{devices}}$$

Where $\eta$ is the effective MFU (model FLOP utilization, typically 0.3–0.5), $t$ is training duration, $n$ is device count.

### Worked Example

Consider a GPT-4-class model at $N = 1.8 \times 10^{12}$ parameters trained on $D = 1.3 \times 10^{13}$ tokens:

$$C = 6 \cdot 1.8 \times 10^{12} \cdot 1.3 \times 10^{13} = 1.4 \times 10^{26} \text{ FLOP}$$

This is **14×** above the AI Act systemic-risk threshold.

Hardware estimation: training on 25,000 A100 GPUs ($312$ TF peak FP16, $\eta = 0.35$):

$$t = \frac{C}{\text{FLOPS}_{\text{peak}} \cdot \eta \cdot n} = \frac{1.4 \times 10^{26}}{312 \times 10^{12} \cdot 0.35 \cdot 25000}$$

$$t \approx 5.1 \times 10^{7} \text{ seconds} \approx 590 \text{ days}$$

Realistic timelines compress this via parallelism, reducing wall-clock to ~100 days with 150k GPUs.

### Why It Matters

The $10^{25}$ threshold is not arbitrary — it corresponds to roughly 2023-era frontier training. The Commission can adjust it by delegated act. Track your own training compute budget in FLOPs, not GPU-hours, because the regulation references the former.

---

## 2. Model Robustness — Adversarial Perturbation Bounds

### The Problem

Art. 15 requires "appropriate levels of accuracy, robustness, and cybersecurity" for high-risk AI. Robustness to adversarial inputs is quantifiable through $\ell_p$-norm perturbation ball analysis.

### The Formula

For classifier $f: \mathbb{R}^d \to \{1, \ldots, K\}$ and input $x$, the adversarial robustness radius under $\ell_p$ norm is:

$$\rho_p(f, x) = \min \{ \|\delta\|_p : f(x + \delta) \neq f(x) \}$$

Certified robustness via randomized smoothing (Cohen et al., 2019): for base classifier $g$ with Gaussian noise $\mathcal{N}(0, \sigma^2 I)$, smoothed classifier $\bar{f}(x) = \arg\max_c P[g(x + \epsilon) = c]$ is certified robust within:

$$R = \frac{\sigma}{2}(\Phi^{-1}(p_A) - \Phi^{-1}(p_B))$$

Where $p_A$ is probability of the top class, $p_B$ of the runner-up, and $\Phi^{-1}$ is the inverse standard normal CDF.

Lipschitz robustness bound: for $L$-Lipschitz $f$:

$$|f(x) - f(x')| \leq L \|x - x'\|_2$$

implies certified accuracy at radius $\epsilon$ requires margin $m \geq L \epsilon$.

### Worked Example

A high-risk credit-scoring classifier with Lipschitz constant $L = 2$ must be robust to $\epsilon = 0.1$ perturbations in normalized feature space. Required decision margin:

$$m \geq 2 \cdot 0.1 = 0.2$$

If observed average margin on validation set is $0.15$, the model fails certified robustness and cannot claim Art. 15 compliance without additional mitigations.

### Why It Matters

Regulators will ask for *evidence* of robustness, not claims. Randomized smoothing with certified radii, $\ell_p$-norm PGD evaluations, and Lipschitz-constrained training produce auditable metrics.

---

## 3. Bias and Fairness — Statistical Parity Measures

### The Problem

Art. 10 requires training data "relevant, representative, free of errors" and "complete." Art. 15 implicitly extends to output fairness. Multiple incompatible fairness metrics must be navigated.

### The Formula

Let $Y$ be the true label, $\hat{Y}$ the prediction, $A$ the protected attribute.

**Demographic parity**:

$$P(\hat{Y} = 1 \mid A = 0) = P(\hat{Y} = 1 \mid A = 1)$$

**Equalized odds**:

$$P(\hat{Y} = 1 \mid A = a, Y = y) = P(\hat{Y} = 1 \mid A = a', Y = y) \quad \forall a, a', y$$

**Equal opportunity** (TPR equality):

$$P(\hat{Y} = 1 \mid A = 0, Y = 1) = P(\hat{Y} = 1 \mid A = 1, Y = 1)$$

**Impossibility theorem** (Chouldechova, Kleinberg et al.): if base rates $P(Y = 1 \mid A)$ differ across groups, no classifier can simultaneously satisfy calibration, equal FPR, and equal FNR. Regulatory compliance requires *choosing* which fairness property to prioritize and documenting the trade-off.

Disparate impact ratio (US EEOC "four-fifths rule" often used as EU benchmark):

$$\text{DI} = \frac{P(\hat{Y} = 1 \mid A = \text{minority})}{P(\hat{Y} = 1 \mid A = \text{majority})}$$

Threshold: $\text{DI} < 0.8$ triggers scrutiny.

### Worked Example

Recruitment AI screens candidates. Observed approval rates:

- $P(\hat{Y} = 1 \mid A = \text{male}) = 0.60$
- $P(\hat{Y} = 1 \mid A = \text{female}) = 0.42$

$$\text{DI} = 0.42 / 0.60 = 0.70 < 0.8 \Rightarrow \text{disparate impact}$$

Art. 10 + Art. 15 evidence package must include:

1. Demonstrated representativeness of training data
2. Measured DI and remediation actions
3. Trade-off analysis if calibration was preserved at the cost of parity

### Why It Matters

Article 10 + Article 14 (human oversight) interact: if your model has disparate impact but a human reviewer overrides edge cases, you must *quantify* oversight effectiveness — not merely assert it.

---

## 4. Risk Classification — Multi-Criteria Decision under Uncertainty

### The Problem

Classifying a system as "high-risk" under Annex III requires combining multiple soft criteria: does it fall in a listed use case, does it pose significant risk to fundamental rights, is it used in a specific deployment context?

### The Formula

Let $C_i \in \{0, 1\}$ indicate criterion $i$ holds. Annex III classification:

$$\text{HighRisk}(S) = \text{AnnexIII\_UseCase}(S) \land \neg \text{Art6(3)\_Exception}(S)$$

Art. 6(3) exceptions (added in trilogue): a system in Annex III is *not* high-risk if it performs only:

- Narrow procedural task
- Improves result of previously completed human activity
- Detects patterns/deviations without replacing human judgment
- Preparatory task for Annex III assessment

Each exception requires registration and documentation justifying the classification.

Bayesian risk uncertainty: let $\theta$ be uncertain classification factors with posterior $p(\theta | D)$. Expected regulatory risk:

$$E[R] = \int R(\theta) \cdot p(\theta | D) \, d\theta$$

Practical engineering: classify conservatively when $P(\text{high-risk} | \text{evidence}) > 0.3$, because misclassification penalties vastly exceed compliance cost.

### Worked Example

A chatbot assists loan officers by summarizing applications. Annex III point 5(b) covers credit scoring/creditworthiness assessment.

- $C_1$: Annex III listed use case → 1 (essential service)
- Art. 6(3) exception: "preparatory task"? Depends on whether officer still makes decision

If officer reviews each summary before approving, preparatory task exception may apply — but requires documented human oversight. Otherwise: high-risk, full conformity assessment.

Decision: classify as high-risk. Exception claim adds audit risk with marginal compliance cost reduction.

### Why It Matters

The penalty structure (€35M / 7% for prohibited, €15M / 3% for high-risk violations) makes under-classification the dominant risk. Economic expectation calculations favor over-classification nearly universally.

---

## 5. Post-Market Monitoring — Drift Detection

### The Problem

Art. 72 mandates post-market monitoring. Model performance drifts over time; detecting drift mathematically is essential for Art. 15 continued compliance.

### The Formula

Population Stability Index (PSI) between reference distribution $P$ and observed $Q$:

$$\text{PSI} = \sum_{i} (Q_i - P_i) \ln \frac{Q_i}{P_i}$$

Thresholds: $\text{PSI} < 0.1$ no drift; $0.1 \leq \text{PSI} < 0.25$ moderate; $\text{PSI} \geq 0.25$ significant.

KL divergence:

$$D_{\text{KL}}(Q \| P) = \sum_{i} Q_i \ln \frac{Q_i}{P_i}$$

Performance drift via confidence-interval hypothesis testing: reject $H_0: \text{AUC}_t = \text{AUC}_{\text{ref}}$ when

$$|\text{AUC}_t - \text{AUC}_{\text{ref}}| > z_{\alpha/2} \cdot \sigma_{\text{AUC}}$$

### Worked Example

Deployed high-risk fraud detector. Reference FPR = 0.02, drift threshold = 0.005.

Monthly monitoring: month 6 shows FPR = 0.028. Drift = 0.008 > 0.005.

- Trigger post-market incident assessment (Art. 72.2)
- Document in ongoing risk management system (Art. 9)
- If significant: report under Art. 73 within 15 days

### Why It Matters

DORA requires incident reporting in hours. The AI Act requires it in days. Both regulatory clocks run simultaneously once a drift event qualifies as a serious incident.

---

## 6. Synthesis — Compliance as Engineering

The AI Act is not a document to be read but a system to be instrumented. Every obligation in the regulation maps to a measurable, auditable quantity:

| Obligation | Metric | Math Tool |
|-----------|--------|-----------|
| Art. 10 data governance | Representativeness | KL divergence, KS test |
| Art. 15 accuracy | Confidence intervals | Hypothesis testing |
| Art. 15 robustness | Adversarial radius | Randomized smoothing |
| Art. 15 fairness | Disparate impact ratio | Group statistics |
| Art. 51 systemic risk | Training FLOP | Dimensional analysis |
| Art. 72 post-market | Drift | PSI, KL divergence |
| Art. 73 incidents | Severity | Bayesian scoring |

Compliance teams who treat the AI Act as a checkbox exercise will fail audits within the first enforcement cycle. Teams who instrument these quantities from day one pass with evidence packages regulators can actually consume.

---
