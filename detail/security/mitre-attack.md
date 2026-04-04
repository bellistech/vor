# The Mathematics of MITRE ATT&CK -- Coverage Optimization and Threat Modeling

> *With hundreds of techniques and finite resources, the defender must solve an optimization problem: which detections yield the greatest reduction in adversary freedom of action?*

---

## 1. Detection Coverage Optimization (Set Cover Problem)

### The Problem

Given a budget for N detection rules and a universe of ATT&CK techniques used by relevant threat groups, which rules should be developed first to maximize coverage? Each detection rule may cover one or more techniques, and each technique may be detectable by multiple data sources. This maps to a weighted set cover problem.

### The Formula

Let U be the universe of techniques to cover, and let $R = \{r_1, r_2, \ldots, r_m\}$ be candidate detection rules where each $r_i$ covers a subset $S_i \subseteq U$ with development cost $c_i$. The optimization problem is:

$$\min \sum_{i=1}^{m} c_i \cdot x_i \quad \text{subject to} \quad \sum_{i: t \in S_i} x_i \geq 1 \quad \forall t \in U, \quad x_i \in \{0, 1\}$$

This is NP-hard, but the greedy algorithm achieves a $\ln|U| + 1$ approximation. The greedy step selects the rule with the best cost-effectiveness ratio:

$$r^* = \arg\max_{r_i} \frac{|S_i \cap U_{\text{remaining}}|}{c_i}$$

Weighted by threat group prevalence, the value of covering technique t:

$$v(t) = \sum_{g \in G} w_g \cdot \mathbb{1}[t \in T_g]$$

where $w_g$ is the threat group weight (based on targeting relevance) and $T_g$ is the set of techniques used by group g.

### Worked Examples

**Example 1: Prioritizing detection development**

Threat profile: 3 relevant groups with weights w = [0.5, 0.3, 0.2].

Group techniques:
- G1 (w=0.5): {T1059, T1053, T1003, T1566, T1071}
- G2 (w=0.3): {T1059, T1003, T1078, T1048, T1071}
- G3 (w=0.2): {T1059, T1566, T1078, T1027, T1105}

Technique values:
- T1059: 0.5 + 0.3 + 0.2 = 1.0 (highest priority -- used by all groups)
- T1003: 0.5 + 0.3 = 0.8
- T1071: 0.5 + 0.3 = 0.8
- T1566: 0.5 + 0.2 = 0.7
- T1078: 0.3 + 0.2 = 0.5

Greedy order: T1059 first, then T1003/T1071, then T1566.

**Example 2: Rule cost-effectiveness**

Rule A: covers {T1059.001, T1059.003, T1027} with cost 2 days. Effectiveness = 3/2 = 1.5.
Rule B: covers {T1003.001} with cost 0.5 days. Effectiveness = 1/0.5 = 2.0.
Rule C: covers {T1053.005, T1053.003, T1543.003, T1543.002} with cost 3 days. Effectiveness = 4/3 = 1.33.

Greedy selection: B first (ratio 2.0), then A (1.5), then C (1.33).

## 2. Attack Graph Analysis (Graph Theory)

### The Problem

ATT&CK techniques are not independent; attackers chain them in sequences from Initial Access through Impact. Modeling these chains as a directed graph enables identification of critical chokepoints where a single detection or mitigation disrupts the most attack paths.

### The Formula

Let G = (V, E) be the attack graph where vertices are techniques and edges represent "enables" relationships. The betweenness centrality of technique v:

$$C_B(v) = \sum_{s \neq v \neq t} \frac{\sigma_{st}(v)}{\sigma_{st}}$$

where $\sigma_{st}$ is the total number of shortest paths from s to t, and $\sigma_{st}(v)$ is the number passing through v. High betweenness centrality indicates a chokepoint.

The number of distinct attack paths from Initial Access to Impact through an n-layer kill chain with $k_i$ technique options per layer:

$$|\text{paths}| = \prod_{i=1}^{n} k_i$$

Blocking a technique at layer j with $k_j$ options reduces paths by:

$$\Delta = \frac{1}{k_j} \cdot \prod_{i=1}^{n} k_i = \frac{|\text{paths}|}{k_j}$$

The fraction of paths disrupted by detecting technique t used in layer j:

$$f_{\text{disrupted}}(t) = \frac{1}{k_j}$$

### Worked Examples

**Example 1: Kill chain chokepoint**

Simple 5-layer attack: Initial Access (3 options), Execution (5 options), Persistence (8 options), Credential Access (4 options), Exfiltration (3 options).

Total paths: 3 * 5 * 8 * 4 * 3 = 1,440.

Blocking one Execution technique: eliminates 1,440/5 = 288 paths (20%).
Blocking one Persistence technique: eliminates 1,440/8 = 180 paths (12.5%).

Highest impact: block at the layer with fewest options (Initial Access or Exfiltration: 33% per technique).

**Example 2: Betweenness centrality in practice**

T1059 (Command and Scripting Interpreter) appears in paths from nearly every Initial Access technique to downstream tactics. If it connects 12 of 15 possible attack chains:
- Relative centrality: 12/15 = 0.80
- Detecting T1059 with high fidelity disrupts 80% of modeled attack paths
- Compare to T1027 (Obfuscated Files) with centrality 0.30

## 3. Threat Intelligence Scoring (Bayesian Inference)

### The Problem

When multiple threat intelligence sources report different confidence levels about a threat group's use of a technique, how should these be combined into a single probability estimate? Bayesian inference provides a principled framework for updating belief as evidence accumulates.

### The Formula

Prior probability that group G uses technique T, based on base rate:

$$P(T|G) = \frac{\text{groups using } T}{\text{total groups}}$$

Given evidence E (a new intelligence report with reliability r):

$$P(T|G, E) = \frac{P(E|T, G) \cdot P(T|G)}{P(E|T, G) \cdot P(T|G) + P(E|\neg T, G) \cdot P(\neg T|G)}$$

For a source with true positive rate (reliability) r and false positive rate f:

$$P(E|T, G) = r, \quad P(E|\neg T, G) = f$$

$$P(T|G, E) = \frac{r \cdot P(T|G)}{r \cdot P(T|G) + f \cdot (1 - P(T|G))}$$

For multiple independent sources $E_1, E_2, \ldots, E_n$ with reliabilities $r_1, \ldots, r_n$:

$$\text{odds}(T|G, E_1, \ldots, E_n) = \text{odds}(T|G) \cdot \prod_{i=1}^{n} \frac{r_i}{f_i}$$

### Worked Examples

**Example 1: Single intel report**

Base rate for T1566.001 (Spearphishing Attachment): used by 45/140 groups = P(T) = 0.321. Intel report claims APT-X uses it, source reliability r = 0.8, false positive rate f = 0.1.

P(T | report) = (0.8 * 0.321) / (0.8 * 0.321 + 0.1 * 0.679)
= 0.257 / (0.257 + 0.068) = 0.791

One reliable report raises confidence from 32% to 79%.

**Example 2: Conflicting sources**

Source A (r=0.9, f=0.05) says YES. Source B (r=0.7, f=0.15) says NO. Prior P(T) = 0.3.

Prior odds = 0.3/0.7 = 0.429.
Source A (positive): multiply by 0.9/0.05 = 18. Odds = 7.71.
Source B (negative): multiply by (1-0.7)/(1-0.15) = 0.353. Odds = 2.72.
P(T | A yes, B no) = 2.72 / (1 + 2.72) = 0.731.

Despite one source disagreeing, the higher-reliability positive report dominates.

## 4. Detection Confidence Scoring (Receiver Operating Characteristic)

### The Problem

Each detection rule has a sensitivity (true positive rate) and specificity (true negative rate) that depend on the detection threshold. Quantifying detection quality across all possible thresholds enables objective comparison between rules and informs tuning decisions.

### The Formula

For a detection rule with score distribution $f_+(s)$ for true attacks and $f_-(s)$ for benign events, at threshold tau:

$$\text{TPR}(\tau) = \int_{\tau}^{\infty} f_+(s) \, ds$$

$$\text{FPR}(\tau) = \int_{\tau}^{\infty} f_-(s) \, ds$$

The Area Under the ROC Curve:

$$\text{AUC} = \int_0^1 \text{TPR}(\text{FPR}^{-1}(x)) \, dx = P(S_+ > S_-)$$

where $S_+$ and $S_-$ are scores drawn from the positive and negative distributions. AUC = 1.0 is perfect separation; AUC = 0.5 is random guessing. For Gaussian score distributions with equal variance:

$$\text{AUC} = \Phi\left(\frac{\mu_+ - \mu_-}{\sigma\sqrt{2}}\right)$$

The optimal threshold (minimizing Bayes risk with equal costs):

$$\tau^* = \frac{\sigma^2 \ln\frac{P(-)}{P(+)} + \frac{\mu_+^2 - \mu_-^2}{2}}{\mu_+ - \mu_-}$$

### Worked Examples

**Example 1: PowerShell detection rule**

Benign score distribution: mu_- = 2.0, sigma = 1.5. Attack score: mu_+ = 6.0, sigma = 1.5.

AUC = Phi((6.0 - 2.0) / (1.5 * sqrt(2))) = Phi(1.886) = 0.970.
This is a high-quality rule (AUC > 0.95).

With attack prevalence P(+) = 0.001:
tau* = (1.5^2 * ln(999) + (36 - 4)/2) / (6 - 2) = (2.25 * 6.907 + 16) / 4 = (15.54 + 16) / 4 = 7.89

At tau = 7.89: TPR = Phi((6.0 - 7.89)/1.5) = Phi(-1.26) = 0.104 (low sensitivity).
At tau = 4.0 (equal-error): TPR = 0.909, FPR = 0.091 (better for security use).

**Example 2: Comparing two detection rules**

Rule A: AUC = 0.85. Rule B: AUC = 0.92. At FPR = 0.01:
- Rule A TPR approximately 0.45
- Rule B TPR approximately 0.72

Rule B detects 60% more true attacks at the same false positive rate. Investing in Rule B tuning yields better returns.

## Prerequisites

- Combinatorial optimization (set cover, greedy algorithms, NP-hardness)
- Graph theory (directed graphs, betweenness centrality, path enumeration)
- Bayesian statistics (prior/posterior, likelihood ratios, odds form)
- Signal detection theory (ROC curves, AUC, sensitivity/specificity)
- Probability distributions (Gaussian, Poisson for base rates)
- Linear programming (for exact set cover solutions)
- Threat intelligence analysis methodology
