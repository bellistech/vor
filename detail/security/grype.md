# The Mathematics of Grype -- Vulnerability Matching and Risk Prioritization

> *Grype's vulnerability scanning pipeline is fundamentally a set-matching problem between software package inventories and CVE databases, using semantic version range containment, CVSS vector scoring for severity classification, and probabilistic models for estimating false negative rates across heterogeneous vulnerability data sources.*

---

## 1. Package-to-CVE Matching (Set Theory and Version Algebra)

### The Problem

Grype matches packages from an SBOM against vulnerability advisories. Each advisory specifies affected version ranges, and grype must determine whether a package's version falls within an affected range using semantic versioning comparison.

### The Formula

Package inventory $P = \{(n_i, v_i, t_i)\}$ where $n$ = name, $v$ = version, $t$ = type (apk, deb, npm, go, etc.).

Vulnerability database $D = \{(n_j, R_j, \text{CVE}_j, t_j)\}$ where $R_j$ is the affected version set.

Match set:

$$M = \{(p, \text{CVE}_j) \mid p = (n_i, v_i, t_i) \in P \wedge (n_j, R_j, \text{CVE}_j, t_j) \in D \wedge n_i = n_j \wedge t_i = t_j \wedge v_i \in R_j\}$$

Version range containment for semantic versions:

$$v \in R = \{v \mid v_{\text{intro}} \leq_{\text{sem}} v <_{\text{sem}} v_{\text{fix}}\}$$

For ranges without a fix version:

$$v \in R_{\text{unfixed}} = \{v \mid v_{\text{intro}} \leq_{\text{sem}} v\}$$

### Worked Examples

SBOM contains 450 packages. Database contains 280,000 advisories.

Naive comparison: $450 \times 280{,}000 = 126{,}000{,}000$ comparisons.

With name+type index (hash lookup): average $\frac{280{,}000}{50{,}000} \approx 5.6$ advisories per unique package name.

Indexed comparisons: $450 \times 5.6 = 2{,}520$ version range checks.

Speedup factor: $\frac{126{,}000{,}000}{2{,}520} = 50{,}000\times$

| Package | Version | Advisory Range | Match |
|:---|:---|:---|:---:|
| openssl | 3.1.2 | [3.0.0, 3.1.4) | Yes |
| curl | 8.5.0 | [8.0.0, 8.4.0) | No |
| zlib | 1.3.1 | [0, 1.3.1.1) | Yes |

---

## 2. CVSS Severity Classification (Threshold Functions)

### The Problem

Grype classifies vulnerabilities into severity levels (Critical, High, Medium, Low, Negligible) using CVSS base scores. The `--fail-on` flag applies a threshold function to determine whether a scan should fail.

### The Formula

Severity classification function:

$$S(\text{score}) = \begin{cases} \text{Critical} & 9.0 \leq \text{score} \leq 10.0 \\ \text{High} & 7.0 \leq \text{score} < 9.0 \\ \text{Medium} & 4.0 \leq \text{score} < 7.0 \\ \text{Low} & 0.1 \leq \text{score} < 4.0 \\ \text{Negligible} & \text{score} = 0.0 \end{cases}$$

CI gate threshold function:

$$\text{fail}(M, \tau) = \exists (p, c) \in M : S(\text{CVSS}(c)) \geq_{\text{ord}} \tau$$

where $\geq_{\text{ord}}$ is the severity ordering: Critical > High > Medium > Low > Negligible.

Count by severity:

$$N_s = |\{(p, c) \in M : S(\text{CVSS}(c)) = s\}|$$

### Worked Examples

Scan results: $|M| = 23$ matches.

| Severity | Count | Scores |
|:---|:---:|:---|
| Critical | 2 | 9.8, 9.1 |
| High | 5 | 8.4, 7.8, 7.5, 7.2, 7.0 |
| Medium | 11 | 6.5, 6.1, 5.8, ... |
| Low | 5 | 3.9, 3.2, ... |

With `--fail-on high`: $\text{fail} = \text{true}$ (2 Critical + 5 High exist).

With `--fail-on critical`: $\text{fail} = \text{true}$ (2 Critical exist).

With `--only-fixed` applied: if 1 Critical has no fix, $N_{\text{critical}} = 1$.

---

## 3. Database Coverage and False Negatives (Probability)

### The Problem

Grype's detection rate depends on the completeness of its vulnerability database. Different sources (NVD, GitHub Advisories, OS vendor advisories) have different coverage, and the union of all sources determines the overall detection probability.

### The Formula

For $k$ vulnerability data sources, each with detection probability $p_i$ for a given CVE:

$$P(\text{detect}) = 1 - \prod_{i=1}^{k}(1 - p_i)$$

False negative rate:

$$P(\text{miss}) = \prod_{i=1}^{k}(1 - p_i)$$

Expected detected vulnerabilities from true set $V$:

$$E[\text{detected}] = |V| \cdot P(\text{detect})$$

Expected missed:

$$E[\text{missed}] = |V| \cdot P(\text{miss})$$

### Worked Examples

Grype aggregates from 3 sources: NVD ($p_1 = 0.85$), GitHub Advisories ($p_2 = 0.70$), OS vendor ($p_3 = 0.90$).

$$P(\text{detect}) = 1 - (1-0.85)(1-0.70)(1-0.90) = 1 - (0.15)(0.30)(0.10) = 1 - 0.0045 = 0.9955$$

$$P(\text{miss}) = 0.0045 = 0.45\%$$

For an image with 15 true vulnerabilities:

$$E[\text{detected}] = 15 \times 0.9955 = 14.93$$

$$E[\text{missed}] = 15 \times 0.0045 = 0.07$$

Probability of missing zero out of 15:

$$P(\text{miss} = 0) = (1 - 0.0045)^{15} = 0.9955^{15} = 0.935$$

---

## 4. SBOM Dependency Depth and Transitive Risk (Graph Theory)

### The Problem

When grype scans an SBOM, the dependency graph determines how many packages are in scope. Transitive dependencies dramatically increase the attack surface, and risk propagates through dependency chains.

### The Formula

Dependency graph $G = (V, E)$ with $|V|$ total packages and depth $d$.

Total reachable packages from root:

$$|T(\text{root})| = 1 + \sum_{i=1}^{d} b^i = \frac{b^{d+1} - 1}{b - 1}$$

where $b$ is the average branching factor.

Risk propagation: vulnerability at depth $k$ reaches the root through at least one path:

$$P(\text{exploitable at root}) = 1 - (1 - p_{\text{reach}})^{|\text{paths}(v, \text{root})|}$$

where $p_{\text{reach}}$ is the probability that a given path is exercised at runtime.

### Worked Examples

Node.js project: $b = 8$ average dependencies per package, $d = 5$ depth.

$$|T| = \frac{8^6 - 1}{7} = \frac{262{,}143}{7} = 37{,}449 \text{ packages}$$

Direct dependencies: 8. Transitive: 37,441. Ratio: $\frac{37{,}441}{8} = 4{,}680\times$ more transitive.

If $P(\text{vulnerability per package}) = 0.02$:

$$E[\text{vulnerable packages}] = 37{,}449 \times 0.02 = 749$$

Direct-only scan would find: $8 \times 0.02 = 0.16$ (likely 0).

SBOM scan captures the full transitive set, revealing 749 expected vulnerabilities.

---

## 5. Ignore Rule Effectiveness (Filtering Theory)

### The Problem

Ignore rules in `.grype.yaml` suppress known-acceptable findings. The ignore configuration must balance noise reduction against the risk of masking genuine new vulnerabilities that match overly broad patterns.

### The Formula

Let $M$ be the raw match set and $I$ be the set of ignore rules.

Filtered results:

$$M_{\text{filtered}} = M \setminus \{m \in M : \exists r \in I, \text{matches}(m, r)\}$$

Noise reduction:

$$\text{noise\_reduction} = \frac{|M| - |M_{\text{filtered}}|}{|M|}$$

False suppression rate (genuine new CVEs suppressed by broad rules):

$$\text{false\_suppression} = \frac{|\{m \in M_{\text{filtered}}^c : m.\text{CVE} \notin \text{accepted\_risks}\}|}{|M_{\text{filtered}}^c|}$$

where $M_{\text{filtered}}^c = M \setminus M_{\text{filtered}}$ is the suppressed set.

### Worked Examples

Raw scan: $|M| = 45$ findings. Ignore rules: 3 specific CVEs + 1 package-level rule.

Suppressed by specific CVE rules: 3 findings.
Suppressed by package rule (all openssl CVEs): 8 findings, of which 2 are newly published.

$$|M_{\text{filtered}}| = 45 - 3 - 8 = 34$$

$$\text{noise\_reduction} = \frac{11}{45} = 24.4\%$$

$$\text{false\_suppression} = \frac{2}{11} = 18.2\%$$

Recommendation: use specific CVE ignore rules, not broad package rules, to minimize false suppression.

---

## 6. Scanner Comparison (Information Retrieval Metrics)

### The Problem

Comparing grype against trivy (or any scanner pair) requires measuring precision, recall, and F1 score against a ground truth vulnerability set.

### The Formula

For ground truth vulnerabilities $T$, scanner results $S$:

$$\text{Precision} = \frac{|S \cap T|}{|S|}, \quad \text{Recall} = \frac{|S \cap T|}{|T|}$$

$$F_1 = 2 \cdot \frac{\text{Precision} \cdot \text{Recall}}{\text{Precision} + \text{Recall}}$$

Unique findings per scanner:

$$U_{\text{grype}} = S_{\text{grype}} \setminus S_{\text{trivy}}, \quad U_{\text{trivy}} = S_{\text{trivy}} \setminus S_{\text{grype}}$$

Combined recall:

$$\text{Recall}_{\text{combined}} = \frac{|S_{\text{grype}} \cup S_{\text{trivy}} \cap T|}{|T|}$$

### Worked Examples

Ground truth: $|T| = 30$ known vulnerabilities in a test image.

| Metric | Grype | Trivy |
|:---|:---:|:---:|
| Reported ($\|S\|$) | 28 | 32 |
| True positives ($\|S \cap T\|$) | 25 | 27 |
| Precision | 89.3% | 84.4% |
| Recall | 83.3% | 90.0% |
| $F_1$ | 86.2% | 87.1% |

$$|U_{\text{grype}}| = 3, \quad |U_{\text{trivy}}| = 5$$

$$\text{Recall}_{\text{combined}} = \frac{|25 \cup 27|}{30} = \frac{30}{30} = 100\%$$

Running both scanners achieves 100% recall in this example, justifying multi-scanner strategies.

---

## Prerequisites

- set-theory, semantic-versioning, probability, graph-theory, information-retrieval, cvss
