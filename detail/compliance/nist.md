# The Mathematics of NIST — Risk Quantification and Control Selection

> *The NIST cybersecurity frameworks rest on formal risk assessment methodologies that quantify threat likelihood, impact severity, and control effectiveness. From FIPS 199 categorization through SP 800-30 risk scoring to the optimization problem of selecting controls under budget constraints, the mathematics of NIST transforms subjective security judgments into defensible, reproducible decisions.*

---

## 1. Risk Scoring and Categorization (Multi-Criteria Decision Analysis)
### The Problem
FIPS 199 requires categorizing information systems by impact level (Low, Moderate, High) across three security objectives. The overall system categorization follows the high-water-mark principle.

### The Formula
System categorization for each security objective:

$$SC_{system} = \{(\text{confidentiality}, c), (\text{integrity}, i), (\text{availability}, a)\}$$

$$\text{Impact}_{overall} = \max(c, i, a)$$

Where $c, i, a \in \{L=1, M=2, H=3\}$.

The risk level from SP 800-30 combines likelihood and impact:

$$R = L \times I$$

Where $L$ = likelihood (1-100 scale) and $I$ = impact (1-100 scale), yielding:

$$R \in [1, 10000] \implies \begin{cases} \text{Very Low:} & 1 \leq R \leq 4 \\ \text{Low:} & 5 \leq R \leq 20 \\ \text{Moderate:} & 21 \leq R \leq 50 \\ \text{High:} & 51 \leq R \leq 80 \\ \text{Very High:} & 81 \leq R \leq 100 \end{cases}$$

(Using the semi-quantitative 1-10 scales from 800-30, mapped above.)

### Worked Examples
**Example 1**: A financial reporting system:
- Confidentiality: Moderate (2) — financial data is sensitive
- Integrity: High (3) — inaccurate reports cause legal liability
- Availability: Low (1) — batch processing, not real-time

$$\text{Impact}_{overall} = \max(2, 3, 1) = 3 \text{ (High)}$$

This system requires the High baseline from 800-53 (~420 controls).

**Example 2**: Risk score for a phishing threat:
- Likelihood: 8/10 (frequent, historically observed)
- Impact: 6/10 (credential theft, lateral movement)

$$R = 8 \times 6 = 48 \implies \text{Moderate}$$

## 2. Control Selection Optimization (Integer Programming)
### The Problem
Given a budget constraint, select the subset of NIST 800-53 controls that maximizes risk reduction. Each control has a cost and reduces risk to one or more threat scenarios.

### The Formula
Binary integer programming formulation:

$$\max \sum_{i=1}^{n} \sum_{j=1}^{m} r_{ij} \cdot x_i$$

Subject to:

$$\sum_{i=1}^{n} c_i \cdot x_i \leq B$$

$$x_i \in \{0, 1\}, \quad i = 1, \ldots, n$$

Where:
- $x_i$ = 1 if control $i$ is selected, 0 otherwise
- $r_{ij}$ = risk reduction from control $i$ against threat $j$
- $c_i$ = implementation cost of control $i$
- $B$ = total budget

For mandatory controls (baseline requirements):

$$x_i = 1, \quad \forall i \in \text{Baseline}$$

### Worked Examples
**Example**: Three optional controls beyond baseline, budget = $100K:

| Control | Cost | Risk Reduction (threats 1-3) | Total |
|---------|------|------------------------------|-------|
| MFA (IA-2) | $30K | [15, 20, 5] | 40 |
| SIEM (AU-6) | $80K | [10, 10, 25] | 45 |
| DLP (SC-7) | $50K | [5, 15, 20] | 40 |

Feasible combinations within $100K:
- MFA + DLP: cost=$80K, reduction=80
- MFA only: cost=$30K, reduction=40
- SIEM only: cost=$80K, reduction=45
- DLP only: cost=$50K, reduction=40

Optimal: MFA + DLP (reduction=80, cost=$80K).

## 3. Threat Likelihood Estimation (Poisson Process)
### The Problem
SP 800-30 requires estimating the likelihood of threat events. Historical incident data can be modeled as a Poisson process to predict future event frequency.

### The Formula
Probability of $k$ events in time period $t$:

$$P(X = k) = \frac{(\lambda t)^k e^{-\lambda t}}{k!}$$

Where $\lambda$ is the average rate of threat events per unit time.

Probability of at least one event:

$$P(X \geq 1) = 1 - e^{-\lambda t}$$

### Worked Examples
**Example**: Historical data shows 3 ransomware attempts per year ($\lambda = 3$).

Probability of zero attacks next year:
$$P(X=0) = e^{-3} \approx 0.0498$$

Probability of at least one attack:
$$P(X \geq 1) = 1 - 0.0498 = 0.9502$$

Probability of 5 or more attacks:
$$P(X \geq 5) = 1 - \sum_{k=0}^{4} \frac{3^k e^{-3}}{k!} = 1 - 0.8153 = 0.1847$$

## 4. Control Effectiveness Decay (Reliability Theory)
### The Problem
Controls degrade over time as threats evolve, configurations drift, and personnel change. Modeling this decay informs the frequency of reassessment.

### The Formula
Exponential decay model for control effectiveness:

$$e(t) = e_0 \cdot e^{-\mu t}$$

Where:
- $e_0$ = initial effectiveness at deployment
- $\mu$ = decay rate (dependent on threat evolution speed)
- $t$ = time since last assessment/update

Mean time to ineffectiveness (MTTI), when $e(t) < e_{threshold}$:

$$MTTI = \frac{1}{\mu} \ln\left(\frac{e_0}{e_{threshold}}\right)$$

### Worked Examples
**Example**: Firewall rules deployed with $e_0 = 0.95$, decay rate $\mu = 0.1$ per month, threshold $e_{threshold} = 0.70$.

$$MTTI = \frac{1}{0.1} \ln\left(\frac{0.95}{0.70}\right) = 10 \times \ln(1.357) = 10 \times 0.305 = 3.05 \text{ months}$$

This means firewall rules should be reviewed at least every 3 months.

After 6 months without review:
$$e(6) = 0.95 \times e^{-0.1 \times 6} = 0.95 \times 0.549 = 0.521$$

The control has fallen well below the effectiveness threshold.

## 5. Continuous Monitoring — Anomaly Detection (Hypothesis Testing)
### The Problem
ISCM (SP 800-137) requires detecting deviations from normal system behavior. Statistical hypothesis testing formalizes when an observation is anomalous.

### The Formula
For a monitored metric $X$ with historical mean $\mu_0$ and standard deviation $\sigma$:

$$H_0: \mu = \mu_0 \quad \text{(normal operation)}$$
$$H_1: \mu \neq \mu_0 \quad \text{(anomaly)}$$

Test statistic:

$$Z = \frac{\bar{X} - \mu_0}{\sigma / \sqrt{n}}$$

Reject $H_0$ (flag anomaly) if $|Z| > Z_{\alpha/2}$.

For sequential monitoring, use CUSUM (Cumulative Sum):

$$S_t = \max(0, S_{t-1} + (x_t - \mu_0) - k)$$

Alert when $S_t > h$ (decision threshold).

### Worked Examples
**Example**: Network baseline: average 500 DNS queries/min, $\sigma = 50$. Current sample of $n = 10$ minutes shows $\bar{X} = 580$.

$$Z = \frac{580 - 500}{50 / \sqrt{10}} = \frac{80}{15.81} = 5.06$$

Since $|5.06| > 1.96$ (95% confidence), this is a significant anomaly.

## 6. Supply Chain Risk Propagation (Network Risk Models)
### The Problem
NIST 800-161 addresses supply chain risk. The risk to an organization depends on the security posture of its suppliers and their suppliers (transitive risk).

### The Formula
For a supply chain graph $G = (V, E)$, the propagated risk to node $v$:

$$R_v = r_v + \sum_{u \in \text{pred}(v)} w_{uv} \cdot R_u \cdot (1 - m_{uv})$$

Where $r_v$ is intrinsic risk, $w_{uv}$ is dependency weight, and $m_{uv}$ is the mitigation effectiveness of controls on that supply chain link.

### Worked Examples
**Example**: A software vendor (intrinsic risk $r = 0.1$) depends on:
- Open-source library ($R = 0.3, w = 0.4, m = 0.8$)
- Cloud hosting ($R = 0.05, w = 0.6, m = 0.9$)

$$R_{vendor} = 0.1 + 0.4 \times 0.3 \times 0.2 + 0.6 \times 0.05 \times 0.1$$
$$= 0.1 + 0.024 + 0.003 = 0.127$$

## 7. FIPS 200 Security Requirements Coverage (Set Theory)
### The Problem
NIST 800-53 controls must cover all 17 minimum security requirements from FIPS 200. Measuring coverage completeness ensures baseline adequacy.

### The Formula
For a set of implemented controls $I$ and the set of required controls $R_b$ for baseline $b$:

$$\text{Coverage}(b) = \frac{|I \cap R_b|}{|R_b|}$$

The gap set:

$$G = R_b \setminus I$$

Weighted coverage accounting for control criticality:

$$\text{Coverage}_w = \frac{\sum_{c \in I \cap R_b} w_c}{\sum_{c \in R_b} w_c}$$

Where $w_c$ reflects the control's impact on the system's security posture (derived from NIST SP 800-60 impact mapping).

### Worked Examples
**Example**: Moderate baseline requires 325 controls. Organization has implemented 290.

$$\text{Coverage} = \frac{290}{325} = 0.892$$

Gap set: $|G| = 35$ controls.

If the 35 missing controls have average weight 1.5 (above average criticality) and implemented controls average weight 1.0:

$$\text{Coverage}_w = \frac{290 \times 1.0}{290 \times 1.0 + 35 \times 1.5} = \frac{290}{342.5} = 0.847$$

The weighted coverage is lower, indicating the gaps disproportionately affect critical control areas.

## Prerequisites
- probability, statistics, linear-programming, optimization, poisson-processes, hypothesis-testing, graph-theory, reliability-theory, set-theory
