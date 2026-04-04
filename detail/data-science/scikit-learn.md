# The Mathematics of scikit-learn — Learning Theory, Optimization, and Statistical Estimation

> *scikit-learn implements the core mathematical machinery of statistical learning theory: empirical risk minimization drives model fitting, bias-variance decomposition explains generalization, cross-validation provides unbiased performance estimates, and regularization balances model complexity against data fidelity. Every estimator encodes an optimization problem whose solution yields the learned parameters.*

---

## 1. Empirical Risk Minimization (Statistical Learning Theory)
### The Problem
All supervised learning in scikit-learn reduces to finding parameters that minimize a loss function over training data, subject to constraints or regularization.

### The Formula
The learning problem seeks to minimize expected risk:

$$R(\hat{f}) = \mathbb{E}_{(x,y) \sim P}[L(y, \hat{f}(x))]$$

Since $P$ is unknown, minimize the empirical risk:

$$\hat{R}_n(\hat{f}) = \frac{1}{n} \sum_{i=1}^{n} L(y_i, \hat{f}(x_i))$$

With regularization (structural risk minimization):

$$\hat{f} = \arg\min_{f \in \mathcal{F}} \left[ \frac{1}{n} \sum_{i=1}^{n} L(y_i, f(x_i)) + \lambda \Omega(f) \right]$$

Where:
- $L$ = loss function (MSE, cross-entropy, hinge)
- $\Omega$ = complexity penalty (L1, L2, tree depth)
- $\lambda$ = regularization strength

### Worked Examples
**Example**: Ridge Regression ($L$ = squared error, $\Omega$ = L2 norm).

$$\hat{\beta} = \arg\min_{\beta} \left[ \frac{1}{n} \sum_{i=1}^{n} (y_i - x_i^T \beta)^2 + \alpha \|\beta\|_2^2 \right]$$

Closed-form solution:

$$\hat{\beta} = (X^TX + \alpha I)^{-1} X^Ty$$

For $X \in \mathbb{R}^{100 \times 10}$, $\alpha = 1.0$:
- Without regularization ($\alpha=0$): $\hat{\beta}$ may overfit, $\|X^TX\|$ could be ill-conditioned
- With $\alpha = 1$: condition number improves by factor $\frac{\sigma_{max}^2 + 1}{\sigma_{min}^2 + 1}$

If $\sigma_{max} = 100$, $\sigma_{min} = 0.01$:
$$\kappa_{unreg} = \frac{10000}{0.0001} = 10^8, \quad \kappa_{reg} = \frac{10001}{1.0001} \approx 10000$$

Regularization improved conditioning by $10^4$.

## 2. Bias-Variance Decomposition (Estimation Theory)
### The Problem
Model selection in scikit-learn balances underfitting (high bias) against overfitting (high variance). The bias-variance decomposition quantifies this trade-off.

### The Formula
For squared loss, the expected prediction error decomposes:

$$\mathbb{E}[(y - \hat{f}(x))^2] = \underbrace{(\mathbb{E}[\hat{f}(x)] - f(x))^2}_{\text{Bias}^2} + \underbrace{\text{Var}[\hat{f}(x)]}_{\text{Variance}} + \underbrace{\sigma^2_\epsilon}_{\text{Irreducible noise}}$$

For a model family with complexity parameter $d$:

$$\text{Bias}^2(d) \propto d^{-\alpha}, \quad \text{Variance}(d) \propto \frac{d}{n}$$

Optimal complexity minimizes total error:

$$d^* = \arg\min_d \left[ \text{Bias}^2(d) + \text{Var}(d) \right]$$

### Worked Examples
**Example**: Polynomial regression (degree $d$) on $n = 50$ data points with true function $f(x) = \sin(x)$ and noise $\sigma = 0.3$.

| Degree $d$ | Bias$^2$ | Variance | Total Error |
|-------------|----------|----------|-------------|
| 1 (linear) | 0.25 | 0.01 | 0.26 |
| 3 (cubic) | 0.04 | 0.03 | 0.07 |
| 5 | 0.01 | 0.06 | 0.07 |
| 10 | 0.005 | 0.15 | 0.155 |
| 20 | 0.001 | 0.45 | 0.451 |

Optimal: $d^* \approx 3\text{-}5$ (bias and variance are balanced).

At $d = 20$: model has 21 parameters for 50 points ($d/n = 0.42$), severe overfitting.

## 3. Cross-Validation Theory (Resampling Methods)
### The Problem
Cross-validation estimates the generalization error of a model. Understanding its statistical properties helps choose between $k$-fold, leave-one-out, and repeated methods.

### The Formula
$k$-fold CV estimate of generalization error:

$$\hat{R}_{CV} = \frac{1}{k} \sum_{j=1}^{k} L(y_{test_j}, \hat{f}_{-j}(X_{test_j}))$$

Where $\hat{f}_{-j}$ is trained on all folds except fold $j$.

Variance of CV estimate:

$$\text{Var}[\hat{R}_{CV}] = \frac{1}{k}\sigma^2 + \frac{k-1}{k}\rho\sigma^2$$

Where $\sigma^2$ is the variance of a single fold's error and $\rho$ is the correlation between fold errors (due to overlapping training sets).

For leave-one-out ($k = n$): low bias, high variance (because $\rho \approx 1$).

For 5-fold or 10-fold: moderate bias, lower variance (empirically best trade-off).

### Worked Examples
**Example**: Comparing 5-fold vs 10-fold vs LOO for $n = 200$ samples.

5-fold: each fold trains on 160, tests on 40.
- Training set overlap between any two folds: $\frac{120}{160} = 75\%$
- $\rho \approx 0.75$, effective variance reduction: moderate

10-fold: each fold trains on 180, tests on 20.
- Training set overlap: $\frac{160}{180} = 89\%$
- $\rho \approx 0.89$, less variance reduction

LOO ($k = 200$): trains on 199, tests on 1.
- Overlap: $\frac{198}{199} = 99.5\%$
- $\rho \approx 0.995$, variance barely reduces despite 200 folds

Variance comparison (relative to single test set):
$$\text{Var}_{5\text{-fold}} \approx \frac{1}{5}\sigma^2(1 + 4 \times 0.75) = 0.8\sigma^2$$
$$\text{Var}_{10\text{-fold}} \approx \frac{1}{10}\sigma^2(1 + 9 \times 0.89) = 0.9\sigma^2$$

5-fold often has lower variance despite fewer folds.

## 4. ROC Analysis and AUC (Probability Theory)
### The Problem
ROC-AUC is the most common metric for binary classification in scikit-learn. It has a precise probabilistic interpretation that informs threshold selection.

### The Formula
ROC curve plots True Positive Rate vs False Positive Rate at each threshold $\tau$:

$$TPR(\tau) = P(\hat{f}(x) \geq \tau \mid y = 1) = \frac{TP}{TP + FN}$$

$$FPR(\tau) = P(\hat{f}(x) \geq \tau \mid y = 0) = \frac{FP}{FP + TN}$$

AUC equals the probability that a random positive scores higher than a random negative:

$$AUC = P(\hat{f}(x_+) > \hat{f}(x_-))$$

Equivalently (Wilcoxon-Mann-Whitney statistic):

$$AUC = \frac{\sum_{i:y_i=1} \sum_{j:y_j=0} \mathbf{1}[\hat{f}(x_i) > \hat{f}(x_j)]}{n_+ \times n_-}$$

Standard error of AUC (DeLong's method):

$$SE(AUC) \approx \sqrt{\frac{AUC(1-AUC) + (n_+ - 1)(Q_1 - AUC^2) + (n_- - 1)(Q_2 - AUC^2)}{n_+ \times n_-}}$$

### Worked Examples
**Example**: Binary classifier on 1000 test samples (200 positive, 800 negative). AUC = 0.85.

Interpretation: given a random positive and random negative sample, the model ranks the positive higher 85% of the time.

Standard error (simplified):

$$SE \approx \sqrt{\frac{0.85 \times 0.15}{200}} \approx \sqrt{\frac{0.1275}{200}} = 0.0253$$

95% confidence interval: $0.85 \pm 1.96 \times 0.0253 = [0.800, 0.900]$

For optimal threshold using Youden's J:

$$J(\tau) = TPR(\tau) - FPR(\tau)$$

$$\tau^* = \arg\max_\tau J(\tau)$$

## 5. Regularization Paths (Optimization)
### The Problem
Lasso and ElasticNet trace a path of solutions as the regularization parameter varies. Understanding this path helps select the optimal regularization strength.

### The Formula
Lasso objective:

$$\hat{\beta}(\lambda) = \arg\min_\beta \frac{1}{2n} \|y - X\beta\|_2^2 + \lambda \|\beta\|_1$$

The subgradient optimality condition:

$$-\frac{1}{n} X_j^T(y - X\hat{\beta}) + \lambda \cdot \text{sign}(\hat{\beta}_j) = 0$$

Features enter the model at:

$$\lambda_j^{enter} = \frac{1}{n} |X_j^T y|$$

Maximum $\lambda$ (all coefficients zero):

$$\lambda_{max} = \frac{1}{n} \max_j |X_j^T y|$$

ElasticNet ($\alpha$ is L1 ratio):

$$\hat{\beta} = \arg\min_\beta \frac{1}{2n} \|y - X\beta\|_2^2 + \lambda \alpha \|\beta\|_1 + \frac{\lambda(1-\alpha)}{2} \|\beta\|_2^2$$

### Worked Examples
**Example**: 50 features, 200 samples, 10 truly relevant features.

At $\lambda_{max} = 2.5$: all coefficients zero.

Path as $\lambda$ decreases:
- $\lambda = 2.0$: 3 features selected (top 3 most correlated)
- $\lambda = 1.0$: 8 features selected
- $\lambda = 0.5$: 12 features selected (including 2 noise features)
- $\lambda = 0.1$: 25 features selected (15 noise features)

Cross-validated optimal: $\lambda^* = 0.7$ (selects 10 features, all correct).

Effective degrees of freedom at $\lambda^*$: $df(\lambda^*) = |\{j : \hat{\beta}_j \neq 0\}| = 10$.

## 6. Decision Tree Splitting Criteria (Information Theory)
### The Problem
Decision trees in scikit-learn use information-theoretic criteria (Gini impurity, entropy) to choose optimal splits. Understanding these measures explains why trees select certain features.

### The Formula
Gini impurity for a node with $K$ classes:

$$G = 1 - \sum_{k=1}^{K} p_k^2$$

Entropy (information gain criterion):

$$H = -\sum_{k=1}^{K} p_k \log_2 p_k$$

Information gain from split $s$ dividing node into left ($L$) and right ($R$):

$$IG(s) = H(\text{parent}) - \frac{n_L}{n} H(L) - \frac{n_R}{n} H(R)$$

For regression (variance reduction):

$$\Delta \text{Var}(s) = \text{Var}(\text{parent}) - \frac{n_L}{n}\text{Var}(L) - \frac{n_R}{n}\text{Var}(R)$$

### Worked Examples
**Example**: Binary classification, node with 100 samples (60 positive, 40 negative).

Parent Gini: $G = 1 - (0.6^2 + 0.4^2) = 1 - 0.52 = 0.48$

Parent entropy: $H = -0.6\log_2(0.6) - 0.4\log_2(0.4) = 0.442 + 0.529 = 0.971$

Split A: Left(50: 45+, 5-), Right(50: 15+, 35-)
$$G_L = 1 - (0.9^2 + 0.1^2) = 0.18, \quad G_R = 1 - (0.3^2 + 0.7^2) = 0.42$$
$$\Delta G_A = 0.48 - 0.5(0.18) - 0.5(0.42) = 0.48 - 0.30 = 0.18$$

Split B: Left(30: 28+, 2-), Right(70: 32+, 38-)
$$G_L = 1 - (0.933^2 + 0.067^2) = 0.124, \quad G_R = 1 - (0.457^2 + 0.543^2) = 0.496$$
$$\Delta G_B = 0.48 - 0.3(0.124) - 0.7(0.496) = 0.48 - 0.037 - 0.347 = 0.096$$

Split A is preferred ($\Delta G = 0.18 > 0.096$) because it creates a purer left child.

## Prerequisites
- statistical-learning-theory, optimization, probability, information-theory, linear-algebra, resampling-methods
