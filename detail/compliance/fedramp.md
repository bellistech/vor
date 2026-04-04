# The Mathematics of FedRAMP — Authorization Risk and Compliance Scoring

> *FedRAMP authorization rests on a quantitative foundation that transforms subjective security assessments into reproducible risk scores. From FIPS 199 impact categorization through vulnerability severity scoring via CVSS, to the statistical sampling of controls during 3PAO assessments, the program applies formal mathematical models to determine whether a cloud service meets the threshold for federal use.*

---

## 1. FIPS 199 Security Categorization (Lattice Theory)
### The Problem
Every system seeking FedRAMP authorization must be categorized by impact level. This categorization determines which baseline (Low, Moderate, High) applies and consequently how many controls must be implemented.

### The Formula
The security category is a three-tuple over a partially ordered set:

$$SC = \{(C, impact_C), (I, impact_I), (A, impact_A)\}$$

Where $impact \in \{L=1, M=2, H=3\}$ forms a total order $L < M < H$.

The overall system impact applies the join (supremum) operation:

$$Impact_{system} = \bigvee_{o \in \{C, I, A\}} impact_o = \max(impact_C, impact_I, impact_A)$$

This high-water-mark principle means one High-impact objective escalates the entire system.

### Worked Examples
**Example 1**: A cloud email system for a civilian agency:
- Confidentiality: Moderate (email contains CUI)
- Integrity: Moderate (message tampering is a concern)
- Availability: Low (brief outages are tolerable)

$$Impact_{system} = \max(2, 2, 1) = 2 \implies \text{Moderate baseline (325 controls)}$$

**Example 2**: A law enforcement case management system:
- Confidentiality: High (criminal investigation data)
- Integrity: High (evidence chain of custody)
- Availability: Moderate (real-time access needed but not life-safety)

$$Impact_{system} = \max(3, 3, 2) = 3 \implies \text{High baseline (420 controls)}$$

## 2. CVSS Vulnerability Scoring (Weighted Metric Composition)
### The Problem
FedRAMP requires vulnerability remediation within strict SLAs tied to CVSS severity. Understanding the CVSS formula helps predict which findings will be flagged as critical.

### The Formula
CVSS v3.1 Base Score computation:

$$ISS = 1 - [(1 - C_{impact}) \times (1 - I_{impact}) \times (1 - A_{impact})]$$

Where $C_{impact}, I_{impact}, A_{impact} \in \{0, 0.22, 0.56\}$ for None, Low, High.

If scope is unchanged:

$$Impact = 6.42 \times ISS$$

If scope is changed:

$$Impact = 7.52 \times [ISS - 0.029] - 3.25 \times [ISS - 0.02]^{15}$$

$$Exploitability = 8.22 \times AV \times AC \times PR \times UI$$

$$BaseScore = \begin{cases} 0 & \text{if } Impact \leq 0 \\ \lceil \min(Impact + Exploitability, 10) \rceil_{0.1} & \text{otherwise} \end{cases}$$

### Worked Examples
**Example**: Remote code execution with network access, low complexity, no privileges, no user interaction, scope changed, full CIA impact.

$$ISS = 1 - [(1 - 0.56)(1 - 0.56)(1 - 0.56)] = 1 - [0.44^3] = 1 - 0.0852 = 0.9148$$

$$Impact = 7.52 \times (0.9148 - 0.029) - 3.25 \times (0.9148 - 0.02)^{15}$$
$$= 7.52 \times 0.8858 - 3.25 \times 0.8948^{15}$$
$$= 6.661 - 3.25 \times 0.1944 = 6.661 - 0.632 = 6.029$$

$$Exploitability = 8.22 \times 0.85 \times 0.77 \times 0.85 \times 0.85 = 3.887$$

$$BaseScore = \lceil \min(6.029 + 3.887, 10) \rceil_{0.1} = \lceil 9.916 \rceil_{0.1} = 10.0$$

Result: Critical (9.0-10.0). FedRAMP SLA: remediate within 30 days.

## 3. 3PAO Sampling Methodology (Stratified Sampling)
### The Problem
3PAOs cannot test every instance of every control. They must select representative samples from the population of system components, users, and transactions.

### The Formula
For attribute sampling with desired confidence $c$ and tolerable error rate $e$:

$$n = \frac{Z_c^2 \times p(1-p)}{e^2}$$

For stratified sampling across system tiers:

$$n_{total} = \sum_{h=1}^{H} n_h, \quad n_h = n_{total} \times \frac{N_h \sigma_h}{\sum_{k=1}^{H} N_k \sigma_k}$$

Where $N_h$ is the stratum size and $\sigma_h$ is the stratum standard deviation.

### Worked Examples
**Example**: A CSP has 3 tiers of infrastructure:
- Tier 1: 10 production servers (high criticality, $\sigma = 0.9$)
- Tier 2: 50 staging servers (medium, $\sigma = 0.5$)
- Tier 3: 200 dev instances (low, $\sigma = 0.2$)

Total weighted: $10 \times 0.9 + 50 \times 0.5 + 200 \times 0.2 = 9 + 25 + 40 = 74$

With total sample $n = 30$:
- Tier 1: $30 \times 9/74 = 3.6 \approx 4$
- Tier 2: $30 \times 25/74 = 10.1 \approx 10$
- Tier 3: $30 \times 40/74 = 16.2 \approx 16$

## 4. Risk Exposure Quantification (Expected Loss Model)
### The Problem
The FedRAMP Risk Exposure Table aggregates individual finding risks into an overall risk posture. This determines whether the risk is acceptable for authorization.

### The Formula
Total risk exposure:

$$RE_{total} = \sum_{i=1}^{n} w_i \times L_i \times I_i \times (1 - M_i)$$

Where:
- $w_i$ = weight of the affected control area
- $L_i$ = likelihood of exploitation (0-1)
- $I_i$ = impact if exploited (1-10)
- $M_i$ = mitigation factor from compensating controls (0-1)

Authorization threshold:

$$\text{Authorize if } RE_{total} < RE_{threshold} \text{ AND no unmitigated Critical/High findings}$$

### Worked Examples
**Example**: Three open findings in a Moderate system:

| Finding | Weight | Likelihood | Impact | Mitigation | Risk |
|---------|--------|------------|--------|------------|------|
| Missing patches | 0.8 | 0.6 | 7 | 0.5 | 1.68 |
| Weak passwords | 0.7 | 0.4 | 5 | 0.8 | 0.28 |
| No log review | 0.6 | 0.3 | 4 | 0.3 | 0.504 |

$$RE_{total} = 1.68 + 0.28 + 0.504 = 2.464$$

If $RE_{threshold} = 5.0$ for Moderate, the system is within acceptable risk.

## 5. Continuous Monitoring SLA Compliance (Survival Analysis)
### The Problem
FedRAMP mandates strict remediation timelines. Measuring SLA compliance over time uses survival analysis to model the probability of remediating a finding within the allowed window.

### The Formula
Survival function for remediation time $T$:

$$S(t) = P(T > t) = e^{-\lambda t}$$

Where $\lambda$ is the remediation rate. The probability of meeting the SLA deadline $d$:

$$P(\text{SLA met}) = 1 - S(d) = 1 - e^{-\lambda d}$$

Hazard function (instantaneous remediation rate):

$$h(t) = \frac{f(t)}{S(t)} = \lambda$$

For the Weibull model (non-constant remediation rate):

$$S(t) = e^{-(t/\eta)^{\beta}}$$

### Worked Examples
**Example**: Historical data shows average remediation time for High findings is 20 days ($\lambda = 1/20 = 0.05$). FedRAMP SLA is 30 days.

$$P(\text{SLA met}) = 1 - e^{-0.05 \times 30} = 1 - e^{-1.5} = 1 - 0.2231 = 0.7769$$

Only 77.7% of High findings are remediated within SLA. To achieve 95% compliance:

$$0.95 = 1 - e^{-\lambda \times 30} \implies \lambda = \frac{-\ln(0.05)}{30} = \frac{2.996}{30} = 0.0999$$

Required average remediation time: $1/0.0999 \approx 10$ days.

## 6. Authorization Decision Modeling (Decision Theory)
### The Problem
The Authorizing Official must decide whether to grant ATO based on aggregate risk, compensating controls, and operational necessity. This is a decision under uncertainty.

### The Formula
Expected utility of authorization decision:

$$EU(\text{authorize}) = p_{safe} \times U_{benefit} + (1 - p_{safe}) \times U_{breach}$$

$$EU(\text{deny}) = U_{status\_quo}$$

Authorize if:

$$EU(\text{authorize}) > EU(\text{deny})$$

$$p_{safe} \times U_{benefit} + (1 - p_{safe}) \times U_{breach} > U_{status\_quo}$$

Break-even safety probability:

$$p^* = \frac{U_{status\_quo} - U_{breach}}{U_{benefit} - U_{breach}}$$

### Worked Examples
**Example**: A cloud migration decision:
- $U_{benefit} = 100$ (operational improvement from cloud)
- $U_{breach} = -500$ (cost of security incident)
- $U_{status\_quo} = 20$ (current on-prem operations)
- System safety probability after FedRAMP controls: $p_{safe} = 0.95$

$$EU(\text{authorize}) = 0.95 \times 100 + 0.05 \times (-500) = 95 - 25 = 70$$
$$EU(\text{deny}) = 20$$

Since $70 > 20$, authorization is the rational decision.

Break-even: $p^* = \frac{20 - (-500)}{100 - (-500)} = \frac{520}{600} = 0.867$

Authorization is justified as long as $p_{safe} > 86.7\%$.

## Prerequisites
- probability, statistics, decision-theory, survival-analysis, sampling-theory, lattice-theory, cvss-scoring
