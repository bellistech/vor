# The Mathematics of Threat Modeling — Risk Quantification and Attack Graph Theory

> *Threat modeling is the application of structured reasoning to adversarial systems. At its core, it transforms qualitative security concerns into quantitative risk assessments through combinatorial analysis of attack paths, probabilistic estimation of exploit likelihood, and decision-theoretic optimization of defensive resource allocation.*

---

## 1. STRIDE as Predicate Logic (Formal Threat Classification)

### Security Property Violations

Each STRIDE category corresponds to the negation of a security property:

$$\text{Spoofing} \iff \lnot\text{Authentication}, \quad \text{Tampering} \iff \lnot\text{Integrity}$$
$$\text{Info Disclosure} \iff \lnot\text{Confidentiality}, \quad \text{Elevation} \iff \lnot\text{Authorization}$$

### Threat Count per System

| Element Type | Applicable STRIDE | Max Threats |
|:---|:---|:---:|
| External Entity | S, R | 2 |
| Process | S, T, R, I, D, E | 6 |
| Data Store | T, R, I, D | 4 |
| Data Flow | T, I, D | 3 |

For $p$ processes, $d$ data stores, $f$ flows, $x$ external entities:

$$|\text{Threats}| \leq 6p + 4d + 3f + 2x$$

---

## 2. DREAD Scoring — Bayesian Risk Assessment

### Score Aggregation

$$\text{DREAD} = \frac{D_{\text{damage}} + R_{\text{repro}} + E_{\text{exploit}} + A_{\text{affected}} + D_{\text{discover}}}{5}$$

### Converting to Risk Matrix

$$\text{Likelihood} = \frac{R + E + D_{\text{discover}}}{3}, \quad \text{Impact} = \frac{D_{\text{damage}} + A}{2}$$

$$\text{Risk} = \text{Likelihood} \times \text{Impact}$$

### Inter-Rater Reliability

For $k$ independent raters, the intraclass correlation coefficient (ICC):

| ICC Range | Agreement | Reliability |
|:---|:---:|:---|
| 0.0 - 0.5 | Poor | Scores too subjective; recalibrate |
| 0.5 - 0.75 | Moderate | Acceptable for initial triage |
| 0.75 - 0.9 | Good | Reliable for prioritization |
| 0.9 - 1.0 | Excellent | Consensus achieved |

---

## 3. Attack Trees — Boolean Algebra and Optimization

### Formal Definition

An attack tree: $T = (V, E, \text{type}: V \to \{\text{AND}, \text{OR}, \text{LEAF}\}, \text{cost}: \text{LEAF} \to \mathbb{R}^+)$

### Aggregation Rules

| Metric | OR Node | AND Node |
|:---|:---|:---|
| Min Cost | $\min(c_1, \ldots, c_n)$ | $\sum c_i$ |
| Min Time | $\min(t_i)$ | $\max(t_i)$ parallel, $\sum t_i$ serial |
| Probability | $1 - \prod(1 - p_i)$ | $\prod p_i$ |
| Skill Level | $\min(s_i)$ | $\max(s_i)$ |

### Worked Example

Exfiltrate DB (OR):
- SQLi: $\$500$, $p=0.7$
- Phishing + Lateral (AND): $\$200 + \$800 = \$1000$, $p = 0.4 \times 0.3 = 0.12$
- Insider: $\$5000$, $p=0.05$

$$\text{Cheapest} = \$500, \quad P(\text{root}) = 1 - (0.3)(0.88)(0.95) = 0.749$$

---

## 4. Data Flow Analysis — Graph Theory

### DFD as Graph

A DFD: $G = (V, E)$ where $V = V_{\text{ext}} \cup V_{\text{proc}} \cup V_{\text{store}}$.

Trust zones $Z_1, \ldots, Z_k$ partition $V$. Attack surface:

$$A = \{(u,v) \in E : \text{zone}(u) \neq \text{zone}(v)\}$$

| System Type | Components | Data Flows | Attack Surface |
|:---|:---:|:---:|:---:|
| Monolith | 5-10 | 10-20 | 5-10 |
| Microservices | 20-50 | 50-200 | 30-100 |
| Cloud-native | 30-100 | 100-500 | 50-200 |

---

## 5. PASTA — Probabilistic Attack Path Analysis

### Path Probability

For path $p = (v_0, v_1, \ldots, v_k)$ with independent exploitation:

$$P(p) = \prod_{i=0}^{k-1} P(\text{exploit}(v_i, v_{i+1}))$$

### Expected Loss

$$\text{Expected Loss} = \sum_{p \in \text{Paths}} P(p) \times I(p)$$

### Optimal Mitigation Selection

Mitigation $m$ reducing edge probability by factor $r_m$:

$$\Delta\text{Risk} = \sum_{p \ni (u,v)} P(p) \times I(p) \times (1 - r_m)$$

Optimal mitigation maximizes $\Delta\text{Risk} / \text{cost}(m)$.

---

## 6. Defense Optimization — Resource Allocation

### Knapsack Formulation

Given budget $B$, mitigations with costs $c_i$ and risk reduction $v_i$:

$$\max \sum_{i=1}^{n} v_i x_i \quad \text{s.t.} \quad \sum c_i x_i \leq B, \quad x_i \in \{0, 1\}$$

Greedy by efficiency ratio $v_i / c_i$ provides a 2-approximation.

| Mitigation | Cost | Risk Reduction | Efficiency |
|:---|:---:|:---:|:---:|
| Input validation | \$5K | 0.30 | 0.060 |
| MFA | \$10K | 0.35 | 0.035 |
| Security training | \$8K | 0.15 | 0.019 |
| WAF deployment | \$20K | 0.25 | 0.013 |
| Encryption at rest | \$15K | 0.20 | 0.013 |

Greedy order: Input validation, MFA, training, WAF/encryption.

---

## 7. Threat Coverage — Set Theory

Coverage against threat universe $U$:

$$\text{Coverage} = \frac{|M \cap U|}{|U|}$$

| Methodology | Typical Coverage |
|:---|:---:|
| STRIDE (manual) | 70-90% of OWASP Top 10 |
| STRIDE (tooling) | 85-95% |
| ATT&CK mapping | 40-60% |
| Combined | 80-95% |

Gap severity: $\text{Gap Risk} = \sum_{t \in U \setminus M} P(t) \times I(t)$

### Multi-Methodology Improvement

Combining methodologies $A$ and $B$ with independent coverage $c_A$ and $c_B$:

$$c_{A \cup B} = c_A + c_B - c_A \times c_B$$

For STRIDE ($c = 0.85$) and ATT&CK ($c = 0.50$):

$$c_{\text{combined}} = 0.85 + 0.50 - 0.425 = 0.925$$

---

## 8. Residual Risk — Convergence Analysis

### Iterative Reduction

After $k$ rounds with effectiveness $e_i \approx e_1 \alpha^{i-1}$ ($\alpha \approx 0.7$):

$$R_k = R_0 \prod_{i=1}^{k} (1 - e_1 \alpha^{i-1})$$

| Round | Effectiveness | Cumulative Reduction | Residual |
|:---|:---:|:---:|:---:|
| 1 | 50.0% | 50.0% | 50.0% |
| 2 | 35.0% | 67.5% | 32.5% |
| 3 | 24.5% | 75.5% | 24.5% |
| 4 | 17.2% | 79.7% | 20.3% |
| 5 | 12.0% | 82.1% | 17.9% |

Residual risk asymptotically approaches a floor set by threat landscape evolution.

---

*Defenders face a covering problem (protect all paths) while attackers face a shortest-path problem (find the cheapest route). This fundamental asymmetry means that quantitative threat modeling and efficient resource allocation are not optional but essential for any system facing a motivated adversary.*

## Prerequisites

- Graph theory (directed graphs, paths, trees, cuts)
- Basic probability theory (independence, conditional probability, Bayes' theorem)
- Optimization theory (knapsack problem, greedy algorithms)

## Complexity

- **Beginner:** Applying STRIDE to components, building simple DFDs, using DREAD scoring
- **Intermediate:** Attack tree construction, PASTA workflow, coverage analysis against ATT&CK
- **Advanced:** Probabilistic attack path analysis, defense optimization as knapsack, residual risk modeling
