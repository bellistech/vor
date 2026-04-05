# The Theory of AI Testing and Assurance — Statistical Validation, Fairness Impossibility, and Robustness Certification

> *AI testing theory draws on statistical hypothesis testing to validate model performance, impossibility theorems from social choice theory to navigate fairness metric trade-offs, formal verification methods to certify adversarial robustness, and distributional testing to detect dataset shift. These mathematical foundations move AI evaluation beyond empirical benchmarking toward provable guarantees and principled trade-off analysis.*

---

## 1. Statistical Validation Theory

### Hypothesis Testing for Model Evaluation

Model evaluation is fundamentally a hypothesis testing problem. When comparing model $A$ to model $B$:

$$H_0: \mu_A = \mu_B \quad \text{vs} \quad H_1: \mu_A \neq \mu_B$$

where $\mu_A, \mu_B$ are the true performance metrics on the data-generating distribution.

**McNemar's Test:**
For paired binary predictions from two models on the same test set:

| | Model B Correct | Model B Wrong |
|---|---|---|
| Model A Correct | $n_{00}$ | $n_{01}$ |
| Model A Wrong | $n_{10}$ | $n_{11}$ |

The test statistic:

$$\chi^2 = \frac{(|n_{01} - n_{10}| - 1)^2}{n_{01} + n_{10}}$$

follows a $\chi^2$ distribution with 1 degree of freedom under $H_0$. Only discordant pairs ($n_{01}, n_{10}$) contribute, which is appropriate because concordant pairs provide no evidence of difference.

**5x2 Cross-Validation Paired t-Test (Dietterich):**
More powerful than standard paired t-test for model comparison:

1. Perform 5 repetitions of 2-fold cross-validation
2. For each repetition $i$, compute differences $d_i^{(1)}$ and $d_i^{(2)}$ on each fold
3. Compute variance estimate: $s_i^2 = (d_i^{(1)} - \bar{d}_i)^2 + (d_i^{(2)} - \bar{d}_i)^2$
4. Test statistic:

$$t = \frac{d_1^{(1)}}{\sqrt{\frac{1}{5}\sum_{i=1}^{5} s_i^2}}$$

follows a $t$-distribution with 5 degrees of freedom. This avoids the inflated Type I error of standard paired tests on resampled data.

### Cross-Validation Theory

**Bias-Variance Trade-off in CV:**

For $k$-fold cross-validation, the expected error estimate has two sources of error:

$$\text{Bias}(\hat{E}_{k\text{-fold}}) = E[\hat{E}_{k\text{-fold}}] - E_{\text{true}}$$

$$\text{Var}(\hat{E}_{k\text{-fold}}) = \text{Var across different test folds}$$

As $k$ increases:
- Bias decreases (training sets are larger, closer to full dataset)
- Variance increases (test sets overlap more, estimates are correlated)
- Leave-one-out ($k=n$): nearly unbiased but high variance
- $k=5$ or $k=10$: good bias-variance trade-off empirically

**Corrected Resampled Paired t-Test (Nadeau & Bengio):**
The standard paired t-test on CV results underestimates variance because test folds share training data. The corrected variance:

$$\hat{\sigma}^2_{\text{corrected}} = \left(\frac{1}{k} + \frac{n_{\text{test}}}{n_{\text{train}}}\right) \hat{\sigma}^2$$

where $n_{\text{test}}/n_{\text{train}}$ is the ratio of test to training fold sizes.

### Confidence Intervals for Performance Metrics

**Wilson Score Interval for Accuracy:**
For $n$ test samples with $s$ successes (correct predictions):

$$\hat{p} \pm \frac{z_{\alpha/2}}{1 + z_{\alpha/2}^2/n} \sqrt{\frac{\hat{p}(1-\hat{p})}{n} + \frac{z_{\alpha/2}^2}{4n^2}}$$

This is preferred over the Wald interval (normal approximation) because it has better coverage properties near 0 and 1.

**Bootstrap Confidence Interval for Any Metric:**
1. Resample test set with replacement $B$ times (typically $B \geq 2000$)
2. Compute metric on each bootstrap sample: $\theta_1^*, \ldots, \theta_B^*$
3. BCa (bias-corrected and accelerated) interval:

$$[\theta^*_{(\alpha_1)}, \theta^*_{(\alpha_2)}]$$

where $\alpha_1, \alpha_2$ are adjusted percentiles accounting for bias and skewness.

**DeLong's Test for AUC Comparison:**
For comparing AUC-ROC of two models on the same test set, DeLong's test uses the theory of U-statistics:

$$Z = \frac{\hat{AUC}_A - \hat{AUC}_B}{\sqrt{\hat{V}(\hat{AUC}_A) + \hat{V}(\hat{AUC}_B) - 2\hat{C}(\hat{AUC}_A, \hat{AUC}_B)}}$$

where $\hat{V}$ and $\hat{C}$ are variance and covariance estimated from the Mann-Whitney U-statistic representation. This accounts for the correlation between AUC estimates evaluated on the same data.

## 2. Fairness Metric Impossibility Results

### The Impossibility Theorem (Chouldechova 2017)

**Theorem:** When base rates differ between groups ($P(Y=1|A=0) \neq P(Y=1|A=1)$), it is impossible to simultaneously satisfy:

1. **Calibration:** $P(Y=1|\hat{Y}=p, A=a) = p$ for all $a$
2. **False Positive Rate Parity:** $P(\hat{Y}=1|Y=0, A=0) = P(\hat{Y}=1|Y=0, A=1)$
3. **False Negative Rate Parity:** $P(\hat{Y}=0|Y=1, A=0) = P(\hat{Y}=0|Y=1, A=1)$

unless the classifier is perfect ($\hat{Y} = Y$) or the base rates are equal.

**Proof Sketch:**
For a calibrated classifier with score $s$ and threshold $t$:

$$\text{FPR}_a = \frac{\int_t^1 s \cdot (1-g_a(s)) \, ds}{\int_0^1 (1-g_a(s)) \, ds}$$

where $g_a(s) = P(Y=1|S=s, A=a)$. If calibration holds, $g_a(s) = s$ for both groups, so the FPR depends only on the score distribution $f_a(s)$. When base rates differ, the score distributions necessarily differ, making FPR parity impossible under calibration.

### Kleinberg-Mullainathan-Raghavan Impossibility (2016)

Three conditions that cannot all hold simultaneously (when base rates differ):

1. **Calibration within groups:** $E[Y|S=s, A=a] = s$
2. **Balance for the positive class:** $E[S|Y=1, A=0] = E[S|Y=1, A=1]$
3. **Balance for the negative class:** $E[S|Y=0, A=0] = E[S|Y=0, A=1]$

**Practical implication:** Organizations must choose which fairness properties to prioritize based on the specific context and values at stake. There is no universally "fair" classifier when groups have different base rates.

### Navigating the Impossibility

**Decision framework:**

Context → Choose primary fairness criterion:

| Context | Primary Criterion | Rationale |
|---------|-------------------|-----------|
| Criminal justice (pretrial) | FPR parity | False accusations harm liberty |
| Lending | Calibration + DIR ≥ 0.8 | Regulatory requirement (ECOA) |
| Hiring | Selection rate parity | Four-fifths rule (EEOC) |
| Medical screening | FNR parity | Missing disease is life-threatening |
| Content moderation | Precision parity | False removal harms speech |

**Pareto-optimal trade-offs:**
Given two fairness metrics $F_1$ and $F_2$, the Pareto frontier represents all classifiers where improving one metric necessarily worsens the other:

$$\mathcal{P} = \{h : \nexists h' \text{ s.t. } F_1(h') \geq F_1(h) \text{ and } F_2(h') \geq F_2(h) \text{ with at least one strict}\}$$

The choice along this frontier is a values question, not a technical one.

## 3. Adversarial Robustness Certification

### Randomized Smoothing

**Core idea:** Transform any base classifier $f$ into a provably robust classifier $g$ by averaging predictions over Gaussian perturbations.

$$g(x) = \arg\max_c P_{\delta \sim \mathcal{N}(0, \sigma^2 I)}[f(x + \delta) = c]$$

**Cohen et al. (2019) Certification:**

If $g(x) = c_A$ and the probability of the most likely class is $p_A$:

$$\text{Certified radius} = \frac{\sigma}{2}(\Phi^{-1}(p_A) - \Phi^{-1}(p_B))$$

where $p_B$ is the second most likely class probability and $\Phi^{-1}$ is the inverse normal CDF.

When $p_B$ is unknown, the simpler bound:

$$r = \sigma \cdot \Phi^{-1}(p_A)$$

guarantees that for any perturbation $\|\delta\|_2 \leq r$, the prediction $g(x + \delta) = c_A$.

**Trade-offs:**
- Larger $\sigma$: larger certified radius but lower clean accuracy
- Typical: $\sigma = 0.25$ gives ~2% accuracy drop with meaningful certified radius
- Computation: requires $N \geq 1000$ forward passes per input (Monte Carlo estimation of $p_A$)

**Limitations:**
- Only certifies $\ell_2$ robustness
- Clean accuracy degrades with $\sigma$
- Computational cost at inference time
- Does not help against semantic perturbations

### Interval Bound Propagation (IBP)

**Core idea:** Propagate interval bounds through the network to compute guaranteed output bounds for all inputs within an $\epsilon$-ball.

For input $x$ with perturbation budget $\epsilon$:

$$x \in [x - \epsilon, x + \epsilon] = [\underline{x}, \overline{x}]$$

**Linear layer** $z = Wx + b$:

$$\underline{z}_j = \sum_i \max(W_{ji}, 0) \underline{x}_i + \min(W_{ji}, 0) \overline{x}_i + b_j$$

$$\overline{z}_j = \sum_i \max(W_{ji}, 0) \overline{x}_i + \min(W_{ji}, 0) \underline{x}_i + b_j$$

**ReLU activation** $a = \max(0, z)$:

Three cases:
1. $\underline{z} \geq 0$: $[\underline{a}, \overline{a}] = [\underline{z}, \overline{z}]$ (always active)
2. $\overline{z} \leq 0$: $[\underline{a}, \overline{a}] = [0, 0]$ (always inactive)
3. $\underline{z} < 0 < \overline{z}$: $[\underline{a}, \overline{a}] = [0, \overline{z}]$ (unstable)

**Certified robustness:** If for the correct class $y$ and all other classes $c \neq y$:

$$\underline{z}_y^{(L)} > \overline{z}_c^{(L)} \quad \forall c \neq y$$

then no perturbation within $\epsilon$ can change the prediction.

**IBP Training:**
Train with IBP bounds as a regularizer:

$$\mathcal{L} = \alpha \cdot \mathcal{L}_{\text{clean}} + (1 - \alpha) \cdot \mathcal{L}_{\text{IBP}}$$

where $\mathcal{L}_{\text{IBP}}$ is the cross-entropy loss computed using the worst-case (lower bound) logits. Gradually increase $\alpha$ from 1 to target value during training.

## 4. Distributional Shift Testing

### Maximum Mean Discrepancy (MMD)

MMD measures the distance between two distributions using kernel embeddings:

$$\text{MMD}^2(P, Q) = \mathbb{E}_{x,x' \sim P}[k(x, x')] - 2\mathbb{E}_{x \sim P, y \sim Q}[k(x, y)] + \mathbb{E}_{y,y' \sim Q}[k(y, y')]$$

where $k$ is a kernel function (typically Gaussian RBF: $k(x, y) = \exp(-\|x-y\|^2 / 2\sigma^2)$).

**Hypothesis test:**
$$H_0: P = Q \quad \text{vs} \quad H_1: P \neq Q$$

The unbiased estimator of MMD$^2$ from samples $\{x_i\}_{i=1}^m \sim P$ and $\{y_j\}_{j=1}^n \sim Q$:

$$\widehat{\text{MMD}}^2 = \frac{1}{m(m-1)}\sum_{i \neq j} k(x_i, x_j) - \frac{2}{mn}\sum_{i,j} k(x_i, y_j) + \frac{1}{n(n-1)}\sum_{i \neq j} k(y_i, y_j)$$

Under $H_0$, $m \cdot \widehat{\text{MMD}}^2$ converges to a distribution expressible as a weighted sum of chi-squared random variables. In practice, permutation testing is used for the p-value.

**Advantages for ML:** Handles high-dimensional data, works with any data type via kernel choice, detects any distributional difference (not just mean shift).

### Multivariate Drift Detection

For high-dimensional feature spaces, univariate tests on individual features miss joint distribution changes. Multivariate approaches:

**1. Classifier Two-Sample Test:**
Train a binary classifier to distinguish reference from current data:

$$\text{Drift Score} = \text{AUC}(\text{classifier distinguishing reference vs. current})$$

If AUC $\approx 0.5$: no drift. If AUC $\gg 0.5$: significant drift.

Advantages: scales to high dimensions, captures complex distributional changes, interpretable (which features help distinguish the sets?).

**2. Projection-Based Tests:**
Apply PCA or random projections to reduce dimensionality, then apply univariate tests:

$$\text{Project: } x \mapsto w^T x$$
$$\text{Test: KS}(w^T X_{\text{ref}}, w^T X_{\text{cur}})$$

Bonferroni correction for multiple projections:

$$\alpha_{\text{adjusted}} = \alpha / k$$

where $k$ is the number of projections.

### Label Shift Detection

When $P(Y)$ changes but $P(X|Y)$ remains constant:

**Black Box Shift Detection (Lipton et al.):**
Using the confusion matrix $C$ of the classifier on reference data:

$$\hat{w} = C^{-1} \hat{q}$$

where $\hat{q}_c = P_{\text{current}}(\hat{Y} = c)$ and $w_c = P_{\text{current}}(Y = c) / P_{\text{ref}}(Y = c)$.

If $\hat{w}$ differs significantly from the all-ones vector, label shift has occurred.

## 5. AI Audit Standards

### IEEE 2894 — Framework Overview

IEEE 2894 provides a structured approach to evaluating AI fairness:

**Fairness Assessment Phases:**

1. **Context Definition:**
   - Identify stakeholders and their fairness concerns
   - Determine relevant protected attributes
   - Define the decision context and its stakes

2. **Metric Selection:**
   Based on context analysis, select appropriate metrics from:
   - Classification parity metrics (demographic parity, equalized odds)
   - Calibration metrics (score calibration across groups)
   - Individual fairness metrics (Lipschitz condition)
   - Causal fairness metrics (counterfactual fairness)

3. **Measurement:**
   - Compute selected metrics on evaluation data
   - Disaggregate by protected attributes and intersections
   - Report with confidence intervals
   - Compare against established thresholds

4. **Reporting:**
   - Structured fairness report
   - Metric values with statistical significance
   - Identified disparities and root causes
   - Mitigation actions taken or planned

### Assurance Levels

Different deployment contexts require different assurance levels:

| Level | Rigor | Testing Depth | Documentation | Review |
|-------|-------|---------------|---------------|--------|
| AL-1 | Basic | Standard metrics | Model card | Self-assessment |
| AL-2 | Standard | + Fairness audit | + Data doc | Internal review |
| AL-3 | Enhanced | + Robustness test | + Risk assessment | Independent review |
| AL-4 | Comprehensive | + Red team + Formal | Full compliance | Third-party audit |

AL-1: internal tools, low-risk applications
AL-2: customer-facing, moderate risk
AL-3: regulated industries, high-risk AI
AL-4: safety-critical, fundamental rights impacting

## 6. Model Interpretability Testing

### Explanation Consistency Tests

**Stability Test:**
An explanation method should produce similar explanations for similar inputs:

$$\frac{\|\phi(x) - \phi(x')\|_2}{\|x - x'\|_2} \leq L$$

where $\phi(x)$ is the explanation for input $x$ and $L$ is the Lipschitz constant of the explanation function.

High $L$: explanations are unstable (small input changes cause large explanation changes).

**Faithfulness Test (Deletion/Insertion):**
Progressively remove features in order of importance (as ranked by explanation):

$$\text{Deletion AUC} = \int_0^1 f(x_{\text{mask}(k)}) \, dk$$

where $x_{\text{mask}(k)}$ removes the top-$k$ fraction of important features. Lower deletion AUC = more faithful explanation.

$$\text{Insertion AUC} = \int_0^1 f(x_{\text{insert}(k)}) \, dk$$

where $x_{\text{insert}(k)}$ starts from baseline and adds features in importance order. Higher insertion AUC = more faithful explanation.

**Consistency Across Methods:**
Compare explanations from different methods (SHAP, LIME, Integrated Gradients):

$$\text{Rank Correlation}(\phi_{\text{SHAP}}, \phi_{\text{LIME}}) = \text{Spearman}(\text{rank}(\phi_{\text{SHAP}}), \text{rank}(\phi_{\text{LIME}}))$$

High correlation across methods increases confidence in the explanation. Low correlation suggests the explanation may be an artifact of the method rather than reflecting true model behavior.

## 7. Regression Testing for ML

### ML-Specific Regression Tests

Unlike software regression testing where outputs are deterministic, ML regression testing must account for stochastic training and continuous metric spaces.

**Performance Regression Test:**
For a new model version $v_{n+1}$, compare against production model $v_n$:

$$H_0: \mu_{v_{n+1}} \geq \mu_{v_n} - \Delta$$

where $\Delta$ is the maximum acceptable degradation (e.g., 0.5% accuracy).

This is a non-inferiority test, appropriate because we want to ensure the new model is at least as good (within tolerance) as the current model.

**Behavioral Regression Test (CheckList):**
Define test suites of invariance, directional, and minimum functionality tests:

1. **Invariance (INV):** Output should not change for certain input perturbations
   - Example: "The food was great" → "The food was really great" (same sentiment)

2. **Directional (DIR):** Output should change in a predictable direction
   - Example: Adding "not" should flip sentiment

3. **Minimum Functionality (MFT):** Simple cases that must be correct
   - Example: "I love this" → positive sentiment (always)

**Slice-Based Regression:**
Test performance on critical data slices, not just overall:

$$\forall s \in \text{Slices}: \text{metric}_{v_{n+1}}(s) \geq \text{metric}_{v_n}(s) - \Delta_s$$

where slices may include demographic groups, input types, difficulty levels, or business-critical segments.

A model that improves overall accuracy but degrades on a critical slice should fail regression testing.
