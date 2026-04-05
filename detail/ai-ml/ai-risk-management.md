# The Theory of AI Risk Management — Taxonomy, Adversarial ML, Drift Detection, and Quantification

> *AI risk management extends classical risk theory into domains where the threat surface is the model itself: adversarial machine learning provides a formal framework for evasion and poisoning attacks, statistical process control adapts to detect distributional drift in high-dimensional feature spaces, and Bayesian decision theory quantifies the expected cost of AI failures under uncertainty. These mathematical foundations transform AI governance from qualitative checklists into measurable, defensible risk postures.*

---

## 1. AI Risk Taxonomy

### Formal Risk Classification
AI risks can be formally classified along three orthogonal dimensions:

**Dimension 1: Lifecycle Stage**
```
R_lifecycle ∈ {Design, Data, Training, Evaluation, Deployment, Monitoring, Decommission}
```

**Dimension 2: Harm Type (adapted from NIST AI RMF)**
```
Harm to People:
  - Physical safety (autonomous systems, medical AI)
  - Civil liberties (surveillance, profiling)
  - Economic (credit, employment, insurance decisions)
  - Psychological (deepfakes, manipulation)

Harm to Organizations:
  - Financial loss (model failures, adversarial exploitation)
  - Reputational damage (biased outputs, hallucinations)
  - Legal liability (regulatory violations)
  - Operational disruption (model downtime, drift)

Harm to Ecosystems:
  - Market distortion (algorithmic collusion)
  - Information ecosystem damage (misinformation at scale)
  - Environmental (compute-intensive training)
  - Democratic processes (election manipulation)
```

**Dimension 3: Risk Origin**
```
Endogenous Risks (from the AI system itself):
  - Specification gaming (reward hacking)
  - Distributional shift sensitivity
  - Emergent capabilities/behaviors
  - Compounding errors in multi-step reasoning

Exogenous Risks (from the environment):
  - Adversarial actors (attackers, competitors)
  - Regulatory changes
  - Upstream data source changes
  - Infrastructure failures
```

### Risk Interaction Graph
Risks in AI systems are not independent. A directed graph $G = (V, E)$ captures causal relationships:

$$R_i \xrightarrow{p_{ij}} R_j$$

where $p_{ij}$ is the conditional probability that risk $R_i$ triggers risk $R_j$.

Example chain: Data poisoning → Biased model → Discriminatory decisions → Regulatory fine → Reputational damage.

The expected total impact of risk $R_i$ including cascading effects:

$$E[\text{Impact}(R_i)] = I_i + \sum_{j \in \text{children}(i)} p_{ij} \cdot E[\text{Impact}(R_j)]$

This requires cycle detection (risks can form feedback loops) and convergence analysis for recursive computation.

## 2. NIST AI RMF Detailed Walkthrough

### Govern Function — Deep Analysis

The Govern function establishes organizational context. It is unique in that it applies across the entire AI lifecycle and to all other functions.

**Maturity Model for AI Governance:**

| Level | Description | Characteristics |
|-------|-------------|-----------------|
| 1 — Ad Hoc | No formal AI governance | Individual decisions, no policies |
| 2 — Defined | Policies exist but inconsistently applied | Written policies, partial coverage |
| 3 — Managed | Systematic governance with metrics | KPIs tracked, regular reviews |
| 4 — Measured | Quantitative risk management | Statistical process control, benchmarks |
| 5 — Optimized | Continuous improvement, leading practice | Automated compliance, adaptive policies |

**Organizational AI Risk Appetite Framework:**

$$\text{Risk Appetite} = f(\text{Industry}, \text{Regulation}, \text{Maturity}, \text{Use Case Criticality})$$

For a given AI use case with criticality $c \in [0, 1]$:

$$\text{Acceptable Residual Risk} = \text{Base Appetite} \times (1 - c) \times \text{Regulatory Multiplier}$$

Where the regulatory multiplier increases with regulatory scrutiny (e.g., 0.5 for healthcare AI, 0.3 for financial AI, 0.8 for internal analytics).

### Map Function — Contextual Analysis

**AI Impact Assessment Framework:**

For each stakeholder group $s$ and harm type $h$:

$$\text{Impact}_{s,h} = \text{Severity}_{s,h} \times \text{Breadth}_{s,h} \times \text{Reversibility}_{s,h}^{-1}$$

Severity: magnitude of harm to an individual (1-5 scale)
Breadth: number of individuals affected (log scale)
Reversibility: how easily the harm can be undone (1 = irreversible, 5 = easily reversed)

**Use Case Risk Tiering:**

Tier 1 (Minimal Risk): Internal analytics, content recommendation for entertainment
Tier 2 (Limited Risk): Customer service chatbots, spam filtering
Tier 3 (High Risk): Credit scoring, hiring, medical diagnosis support
Tier 4 (Unacceptable Risk): Social scoring, real-time biometric surveillance (mass)

### Measure Function — Quantitative Assessment

**Trustworthiness Characteristics (NIST AI RMF):**

1. Valid and Reliable
2. Safe
3. Secure and Resilient
4. Accountable and Transparent
5. Explainable and Interpretable
6. Privacy-Enhanced
7. Fair with Harmful Bias Managed

Each characteristic requires specific metrics. A composite trustworthiness score:

$$T = \sum_{i=1}^{7} w_i \cdot t_i$$

where $w_i$ are weights reflecting organizational priorities and $t_i \in [0, 1]$ are normalized scores for each characteristic.

## 3. Adversarial ML Theory

### Evasion Attacks

**Formal Definition:**
Given a classifier $f: \mathcal{X} \to \mathcal{Y}$ and an input $x$ with true label $y$, find perturbation $\delta$ such that:

$$f(x + \delta) \neq y \quad \text{subject to} \quad \|\delta\|_p \leq \epsilon$$

**Attack Methods:**

FGSM (Fast Gradient Sign Method):
$$\delta = \epsilon \cdot \text{sign}(\nabla_x \mathcal{L}(f(x), y))$$

Single-step, fast, but suboptimal. Generates perturbation in the direction of maximum loss increase.

PGD (Projected Gradient Descent):
$$x_{t+1} = \Pi_{B_\epsilon(x)} \left( x_t + \alpha \cdot \text{sign}(\nabla_{x_t} \mathcal{L}(f(x_t), y)) \right)$$

Iterative version of FGSM. Projects back onto the $\epsilon$-ball after each step. Considered the strongest first-order attack.

C&W (Carlini-Wagner):
$$\min_\delta \|\delta\|_p + c \cdot \max(Z(x+\delta)_y - \max_{i \neq y} Z(x+\delta)_i, -\kappa)$$

Optimization-based attack that directly minimizes perturbation size. Uses logits $Z$ rather than softmax outputs. Parameter $\kappa$ controls confidence of misclassification.

**Black-Box Attacks:**
When the attacker has no access to model gradients:

1. Transfer attacks: Train surrogate model, generate adversarial examples on it, transfer to target
2. Score-based: Use output probabilities to estimate gradients (zeroth-order optimization)
3. Decision-based: Use only final class label (Boundary Attack, HopSkipJump)

### Poisoning Attacks

**Formal Definition:**
Inject malicious samples $D_p$ into training set $D$ such that the model trained on $D \cup D_p$ exhibits desired malicious behavior.

**Clean-label Poisoning:**
Perturbation applied to correctly-labeled samples. The model learns a spurious correlation without any mislabeled data:

$$x_p = \arg\min_x \|x - x_t\|^2 + \lambda \cdot \|f_\theta(x) - f_\theta(x_b)\|^2$$

where $x_t$ is the target (correctly labeled), $x_b$ is the base sample whose features we want to inject.

**Backdoor Attacks:**
Insert a trigger pattern $\Delta$ such that:
$$f(x \oplus \Delta) = y_{\text{target}} \quad \forall x$$

while $f(x) = y_{\text{correct}}$ for clean inputs. The trigger can be:
- Patch-based: small pixel pattern (e.g., 3x3 in corner)
- Blended: low-opacity overlay across entire image
- Semantic: natural feature (e.g., wearing sunglasses)
- Syntactic (NLP): specific phrase or writing style

Detection methods:
- Neural Cleanse: optimize for minimal perturbation that causes misclassification to each class
- Activation Clustering: cluster activations of final hidden layer, look for anomalous clusters
- Spectral Signatures: SVD of feature covariance, poisoned samples have large projection onto top singular vector

### Model Extraction Attacks

**Query-Based Extraction:**
Given oracle access to target model $f_T$, train surrogate model $f_S$:

$$f_S = \arg\min_g \mathbb{E}_{x \sim \mathcal{D}_q}[\mathcal{L}(g(x), f_T(x))]$$

where $\mathcal{D}_q$ is the query distribution.

**Strategies:**
1. Random query: sample inputs uniformly, label with oracle → poor efficiency
2. Active learning: select queries that maximize information gain about $f_T$
3. Jacobian-based augmentation: use $\nabla_x f_T(x)$ (if available) to generate informative queries
4. Knockoff Nets: use natural images from different domain, surprisingly effective

**Extraction Complexity:**
For a model with $d$-dimensional input and $k$ outputs:
- Linear models: $O(dk)$ queries suffice
- ReLU networks with $n$ neurons: $O(n \cdot d)$ queries for exact extraction
- Deep networks: empirically $O(10^4 - 10^6)$ queries for high-fidelity copies

### Inference Attacks

**Membership Inference:**
Determine whether a specific sample $x$ was in the training set $D$.

Attack model: Train binary classifier on shadow model outputs:
$$A(f(x), y) \to \{0, 1\} \quad \text{(member / non-member)}$$

Key insight: Models tend to be more confident on training data (overfitting). The attack exploits the gap between training and test loss distributions.

**Metric:** Membership inference advantage:
$$\text{Adv} = |P[\text{predict member} | \text{member}] - P[\text{predict member} | \text{non-member}]|$$

Values > 0.1 indicate privacy risk.

**Model Inversion:**
Reconstruct representative training samples from model access.

$$x^* = \arg\max_x f(x)_y - \lambda \cdot R(x)$$

where $R(x)$ is a regularizer encoding prior knowledge about realistic inputs (e.g., total variation for images).

## 4. Drift Detection Algorithms

### Kolmogorov-Smirnov Test
The KS test measures the maximum difference between two empirical CDFs:

$$D_{n,m} = \sup_x |F_n(x) - G_m(x)|$$

where $F_n$ is the reference CDF and $G_m$ is the current CDF.

The null hypothesis (no drift) is rejected at significance level $\alpha$ when:

$$D_{n,m} > c(\alpha) \sqrt{\frac{n + m}{n \cdot m}}$$

where $c(\alpha) = \sqrt{-\frac{1}{2}\ln(\alpha/2)}$.

**Advantages:** Non-parametric, no distributional assumptions, sensitive to location and shape changes.
**Limitations:** Univariate (must test each feature independently or use multivariate extensions), requires sufficient sample size.

### Population Stability Index (PSI)
For discretized distributions with $B$ bins:

$$\text{PSI} = \sum_{i=1}^{B} (p_i^{\text{current}} - p_i^{\text{reference}}) \cdot \ln\left(\frac{p_i^{\text{current}}}{p_i^{\text{reference}}}\right)$$

PSI is a symmetric version of KL divergence. It equals $\text{KL}(P_{\text{cur}} \| P_{\text{ref}}) + \text{KL}(P_{\text{ref}} \| P_{\text{cur}})$.

**Thresholds:**
- PSI < 0.1: No significant change
- 0.1 ≤ PSI < 0.25: Some change, investigate
- PSI ≥ 0.25: Significant change, likely retrain

**Bin Selection:** Equal-frequency (quantile) binning is preferred over equal-width to avoid empty bins and handle skewed distributions.

### ADWIN (Adaptive Windowing)
ADWIN maintains a variable-length window $W$ of recent observations and detects change by finding a split point where two sub-windows have sufficiently different means.

**Algorithm:**
1. Maintain sliding window $W = [x_1, \ldots, x_n]$
2. For each possible split $W_0 = [x_1, \ldots, x_k]$, $W_1 = [x_{k+1}, \ldots, x_n]$:
3. Compute $|\hat{\mu}_{W_0} - \hat{\mu}_{W_1}|$
4. If this exceeds threshold $\epsilon_{\text{cut}}$, declare drift and drop oldest elements

The threshold is derived from the Hoeffding bound:

$$\epsilon_{\text{cut}} = \sqrt{\frac{1}{2m} \cdot \ln\frac{4n}{\delta}}$$

where $m = \frac{1}{1/n_0 + 1/n_1}$ (harmonic mean of window sizes) and $\delta$ is the confidence parameter.

**Properties:**
- Adaptive: window shrinks during change, grows during stability
- Rigorous: false positive rate bounded by $\delta$
- Efficient: $O(\log n)$ memory using exponential histograms
- No parameters to tune beyond $\delta$

### DDM (Drift Detection Method)
Based on the binomial distribution of classification errors. Monitors the error rate $p_t$ and its standard deviation $s_t = \sqrt{p_t(1-p_t)/t}$.

**Warning level:** $p_t + s_t > p_{\min} + 2 \cdot s_{\min}$
**Drift level:** $p_t + s_t > p_{\min} + 3 \cdot s_{\min}$

where $p_{\min}$ and $s_{\min}$ are the minimum observed values of $p$ and $s$.

## 5. AI Incident Classification

### Severity Scoring Model
An AI incident severity score combines multiple factors:

$$S = w_1 \cdot H + w_2 \cdot B + w_3 \cdot D + w_4 \cdot C + w_5 \cdot R$$

where:
- $H$ = Harm magnitude (0-10): actual harm to individuals
- $B$ = Breadth (0-10): number of affected individuals (log-scaled)
- $D$ = Duration (0-10): how long before detection and containment
- $C$ = Controllability (0-10): difficulty of remediation (inverse)
- $R$ = Regulatory exposure (0-10): likelihood of regulatory action

Suggested weights for regulated industries: $w = [0.30, 0.20, 0.15, 0.15, 0.20]$

### AI Incident Taxonomy (adapted from AIAAIC Repository)

| Category | Description | Example |
|----------|-------------|---------|
| Performance failure | Model produces incorrect outputs at unacceptable rate | Medical AI misdiagnosis cluster |
| Bias/discrimination | Systematic unfairness against protected groups | Hiring AI penalizes female candidates |
| Privacy violation | Unauthorized disclosure of personal information | Chatbot reveals training data PII |
| Security breach | Adversarial exploitation of AI system | Prompt injection exfiltrates data |
| Safety incident | AI contributes to physical harm | Autonomous vehicle collision |
| Misuse | AI used for unintended harmful purpose | Deepfake for fraud |
| Reliability failure | AI system unavailable or unstable | Model serving outage |

## 6. Red Team Methodology for AI Systems

### Structured Red Team Framework

**Phase 1: Reconnaissance**
- Enumerate model capabilities and constraints
- Identify input/output modalities
- Map the application architecture
- Determine access level (API, UI, embedded)
- Review public documentation and model cards

**Phase 2: Threat Hypothesis Generation**
For each ATLAS tactic, generate specific hypotheses:

$$H_i: \text{"Attack technique } T_i \text{ can achieve objective } O_j \text{ against model } M_k \text{"}$$

Prioritize hypotheses by: likelihood of success, impact if successful, detection difficulty.

**Phase 3: Attack Execution**
Execute in order of increasing aggression:
1. Passive observation (output analysis, behavior profiling)
2. Input probing (boundary testing, format exploration)
3. Active attacks (prompt injection, adversarial examples)
4. Escalation (chained attacks, multi-step exploitation)

**Phase 4: Scoring**
For each finding, compute:

$$\text{DREAD Score} = \frac{D + R + E + A + D'}{5}$$

Adapted for AI:
- Damage: severity of successful exploitation
- Reproducibility: consistency of the attack
- Exploitability: skill and resources required
- Affected users: breadth of impact
- Discoverability: how easily an attacker could find this

## 7. AI Risk Quantification

### Value at Risk (VaR) for AI Systems
Adapting financial VaR to AI risk:

$$\text{AI-VaR}_\alpha = \inf\{l : P(\text{AI Loss} > l) \leq 1 - \alpha\}$$

The $\alpha$-level VaR is the minimum loss threshold such that the probability of exceeding it is at most $1 - \alpha$.

**Monte Carlo Simulation for AI-VaR:**
1. Model each risk factor as a random variable with estimated distribution
2. Simulate $N$ scenarios (typically $N \geq 10,000$)
3. For each scenario, compute total loss from AI failures
4. Sort losses, VaR at 95% = 500th largest loss

**Expected Shortfall (CVaR):**

$$\text{CVaR}_\alpha = E[\text{Loss} | \text{Loss} > \text{VaR}_\alpha]$$

CVaR captures the average severity of tail events beyond VaR.

### Bayesian Risk Updating
As new evidence (incidents, near-misses, test results) arrives, update risk estimates:

$$P(R_i | \text{evidence}) = \frac{P(\text{evidence} | R_i) \cdot P(R_i)}{P(\text{evidence})}$$

Prior $P(R_i)$ from initial assessment, updated with likelihood from observed data.

For rare AI risks where historical data is sparse, use conjugate priors:
- Beta-Binomial for incident rates: $\text{Beta}(\alpha + \text{incidents}, \beta + \text{non-incidents})$
- Gamma-Poisson for incident frequency: $\text{Gamma}(\alpha + \sum x_i, \beta + n)$

### Risk-Adjusted Return on AI (RA-ROAI)

$$\text{RA-ROAI} = \frac{E[\text{AI Value}] - E[\text{AI Cost}] - E[\text{AI Risk Cost}]}{E[\text{AI Investment}]}$$

Where:
- $E[\text{AI Value}]$: expected business value from AI system
- $E[\text{AI Cost}]$: operational costs (compute, maintenance, monitoring)
- $E[\text{AI Risk Cost}]$: expected loss from AI risks = $\sum_i P(R_i) \cdot I(R_i)$
- $E[\text{AI Investment}]$: total investment in development and deployment

This provides a risk-adjusted business case for AI initiatives, enabling comparison across projects with different risk profiles.
