# The Mathematics of CVE — Vulnerability Scoring Theory and Risk Quantification

> *CVSS scoring is a multi-dimensional vector space mapping vulnerability characteristics to a severity scalar through non-linear composition functions, while EPSS applies logistic regression to predict exploitation probability, enabling risk-optimal patch prioritization.*

---

## 1. CVSS v3.1 Base Score Computation (Multi-Variable Calculus)

### Impact Sub-Score

The impact metrics form a 3-dimensional vector in $[0, 1]^3$:

$$\vec{I} = (I_C, I_I, I_A)$$

Where $I_C$, $I_I$, $I_A$ are the confidentiality, integrity, and availability impact values:

| Metric Value | Weight |
|:---|:---:|
| None (N) | 0.0 |
| Low (L) | 0.22 |
| High (H) | 0.56 |

The **Impact Sub-Score Base (ISS)** combines these:

$$\text{ISS} = 1 - (1 - I_C)(1 - I_I)(1 - I_A)$$

This is the probability-theoretic complement of all-none: at least one dimension has impact.

For Scope Unchanged ($S = U$):

$$\text{Impact} = 6.42 \times \text{ISS}$$

For Scope Changed ($S = C$):

$$\text{Impact} = 7.52 \times [\text{ISS} - 0.029] - 3.25 \times [\text{ISS} - 0.02]^{15}$$

The $[\cdot]^{15}$ term creates a sigmoidal suppression at low ISS values.

### Exploitability Sub-Score

$$\text{Exploitability} = 8.22 \times AV \times AC \times PR \times UI$$

Where each factor is a metric weight:

| Metric | Values | Weights |
|:---|:---|:---|
| Attack Vector (AV) | N, A, L, P | 0.85, 0.62, 0.55, 0.20 |
| Attack Complexity (AC) | L, H | 0.77, 0.44 |
| Privileges Required (PR, S=U) | N, L, H | 0.85, 0.62, 0.27 |
| Privileges Required (PR, S=C) | N, L, H | 0.85, 0.68, 0.50 |
| User Interaction (UI) | N, R | 0.85, 0.62 |

### Final Base Score

$$\text{Base Score} = \begin{cases} 0 & \text{if Impact} = 0 \\ \text{roundup}(\min(f(I, E), 10)) & \text{otherwise} \end{cases}$$

For Scope Unchanged:

$$f(I, E) = \text{Impact} + \text{Exploitability}$$

For Scope Changed:

$$f(I, E) = 1.08 \times (\text{Impact} + \text{Exploitability})$$

The `roundup` function: round up to nearest 0.1.

---

## 2. Score Distribution Analysis (Statistics)

### CVSS Score Space

The number of distinct CVSS v3.1 base vectors:

$$|V| = |AV| \times |AC| \times |PR| \times |UI| \times |S| \times |C| \times |I| \times |A|$$

$$|V| = 4 \times 2 \times 3 \times 2 \times 2 \times 3 \times 3 \times 3 = 2592$$

But many map to the same score. Distinct score values: approximately 76.

### NVD Score Distribution (Empirical)

| Severity | Score Range | % of CVEs | Cumulative |
|:---|:---:|:---:|:---:|
| Critical | 9.0 - 10.0 | ~15% | 15% |
| High | 7.0 - 8.9 | ~40% | 55% |
| Medium | 4.0 - 6.9 | ~35% | 90% |
| Low | 0.1 - 3.9 | ~10% | 100% |

Mean CVSS score across NVD: $\mu \approx 7.1$, $\sigma \approx 1.8$.

This right-skewed distribution means CVSS alone is insufficient for prioritization (most CVEs score high).

---

## 3. EPSS — Exploit Prediction Scoring (Logistic Regression)

### Model

EPSS (Exploit Prediction Scoring System) predicts the probability a CVE will be exploited in the wild within 30 days:

$$P(\text{exploit} | \vec{x}) = \frac{1}{1 + e^{-(\beta_0 + \sum_{i=1}^{n} \beta_i x_i)}}$$

Where $\vec{x}$ includes features like:
- Days since publication
- CVSS base score
- Vendor/product
- CWE type
- Presence in exploit databases
- Social media mentions
- References to exploit code

### EPSS vs CVSS Decision Matrix

| | EPSS High (> 0.1) | EPSS Low (< 0.1) |
|:---|:---|:---|
| **CVSS Critical** | Patch immediately | Schedule within sprint |
| **CVSS High** | Patch within 24h | Next patch cycle |
| **CVSS Medium** | Investigate context | Backlog |
| **CVSS Low** | Monitor | Accept risk |

### Coverage and Efficiency

For a given EPSS threshold $\theta$:

$$\text{Coverage}(\theta) = \frac{|\{v : \text{EPSS}(v) \geq \theta \wedge v \text{ exploited}\}|}{|\{v : v \text{ exploited}\}|}$$

$$\text{Efficiency}(\theta) = \frac{|\{v : \text{EPSS}(v) \geq \theta \wedge v \text{ exploited}\}|}{|\{v : \text{EPSS}(v) \geq \theta\}|}$$

At $\theta = 0.1$: Coverage $\approx 80\%$, Efficiency $\approx 30\%$.
At $\theta = 0.5$: Coverage $\approx 50\%$, Efficiency $\approx 70\%$.

---

## 4. Vulnerability Growth Rate (Time Series Analysis)

### CVE Publication Rate

Annual CVE publications follow approximately exponential growth:

$$N(t) = N_0 \cdot e^{rt}$$

| Year | CVEs Published | Growth Rate |
|:---:|:---:|:---:|
| 2017 | 14,714 | baseline |
| 2019 | 17,382 | 8.5% |
| 2021 | 20,171 | 7.7% |
| 2023 | 29,065 | 20.0% |
| 2024 | ~35,000 | ~20.4% |

Compound annual growth rate:

$$r = \left(\frac{N_{2024}}{N_{2017}}\right)^{1/7} - 1 \approx 13.2\%$$

### Mean Time to Exploit (MTTE)

For vulnerabilities that are exploited:

$$\text{MTTE} = \frac{1}{|\mathcal{E}|} \sum_{v \in \mathcal{E}} (t_{\text{exploit}}(v) - t_{\text{publish}}(v))$$

Empirical data shows:

$$\text{Median MTTE} \approx 14 \text{ days}$$
$$\text{Mean MTTE} \approx 37 \text{ days}$$
$$P(\text{exploit within 7 days} | \text{exploited}) \approx 0.35$$

---

## 5. Patch Prioritization as Optimization (Operations Research)

### Resource-Constrained Patching

Given $n$ vulnerabilities and patching capacity $K$ per time period:

$$\text{maximize} \sum_{i=1}^{n} x_i \cdot \text{risk}(v_i)$$

$$\text{subject to} \sum_{i=1}^{n} x_i \cdot \text{cost}(v_i) \leq K, \quad x_i \in \{0, 1\}$$

This is a 0-1 knapsack problem (NP-hard in general, but practical sizes are solvable).

### Risk Score

$$\text{risk}(v) = P(\text{exploit}) \times \text{impact}(v) \times \text{exposure}(v)$$

Where:
- $P(\text{exploit})$ = EPSS score
- $\text{impact}(v)$ = CVSS impact sub-score or business impact
- $\text{exposure}(v)$ = number of affected assets / internet exposure

### Greedy Approximation

Sort by risk-to-cost ratio:

$$\text{priority}(v) = \frac{\text{risk}(v)}{\text{cost}(v)}$$

Patch in decreasing order of priority until budget exhausted.

Approximation ratio: $\frac{\text{greedy}}{\text{optimal}} \geq \frac{1}{2}$ for the knapsack relaxation.

---

## 6. SBOM and Dependency Graph (Graph Theory)

### Dependency Vulnerability Propagation

A software project's dependency graph $G = (V, E)$ where:
- $V$ = packages (direct + transitive)
- $E$ = dependency relations

A vulnerability in package $p$ affects all dependents:

$$\text{affected}(p) = \{v \in V : p \in \text{reachable}(v)\}$$

### Transitive Dependency Risk

For a project with $d$ direct and $t$ transitive dependencies:

$$P(\text{any vuln}) = 1 - \prod_{i=1}^{d+t} (1 - p_i)$$

Where $p_i$ is the probability dependency $i$ has a known vulnerability.

If $p_i = p$ (uniform):

$$P(\text{any vuln}) = 1 - (1-p)^{d+t}$$

For $p = 0.05$ and $d + t = 200$:

$$P = 1 - 0.95^{200} = 1 - 0.0000358 \approx 99.996\%$$

### Median Dependencies

| Ecosystem | Median Direct | Median Transitive | Total |
|:---|:---:|:---:|:---:|
| npm | 12 | 683 | 695 |
| Maven | 8 | 145 | 153 |
| PyPI | 6 | 42 | 48 |
| Go modules | 5 | 38 | 43 |

---

## 7. Scanner False Positive Analysis (Statistical Testing)

### Binary Classification Metrics

| | Vulnerable (actual) | Not Vulnerable (actual) |
|:---|:---:|:---:|
| **Flagged** | True Positive (TP) | False Positive (FP) |
| **Not Flagged** | False Negative (FN) | True Negative (TN) |

$$\text{Precision} = \frac{TP}{TP + FP}$$

$$\text{Recall} = \frac{TP}{TP + FN}$$

$$\text{F1} = 2 \cdot \frac{\text{Precision} \cdot \text{Recall}}{\text{Precision} + \text{Recall}}$$

### Reachability-Aware Scanning

Tools like `govulncheck` filter by call graph reachability:

$$\text{Reachable CVEs} = \{v : \exists \text{path in call graph from main to } v.\text{function}\}$$

$$\text{Reduction} = 1 - \frac{|\text{Reachable CVEs}|}{|\text{All CVEs in deps}|}$$

Empirical reduction: typically 60-80% fewer alerts.

---

## Prerequisites

multi-variable-functions, logistic-regression, time-series, graph-theory, optimization, statistics

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| CVSS base score computation | $O(1)$ — fixed formula | $O(1)$ — 8 metrics |
| EPSS model inference | $O(f)$ — f = features | $O(1)$ |
| Knapsack prioritization | $O(n \cdot K)$ — DP | $O(K)$ |
| Dependency graph traversal | $O(V + E)$ — BFS/DFS | $O(V)$ |
| SBOM vulnerability lookup | $O(V \cdot \log D)$ — D = DB size | $O(V)$ |
| Call graph reachability | $O(V + E)$ — graph traversal | $O(V)$ |
