# The Mathematics of Trivy — Vulnerability Scoring and Risk Quantification

> *Trivy's scanning engine maps package inventories against vulnerability databases using set intersection, prioritizes findings via CVSS scoring which combines base, temporal, and environmental metrics in a nonlinear formula, and SBOM analysis is fundamentally a dependency graph traversal for transitive vulnerability discovery.*

---

## 1. Vulnerability Matching (Set Theory)

### The Problem

Trivy enumerates installed packages in an image and matches them against known vulnerabilities. This is a set intersection between the package inventory and the vulnerability database, filtered by version ranges.

### The Formula

For image packages $P = \{(p_i, v_i)\}$ and vulnerability database entries $V = \{(p_j, R_j, \text{CVE}_j)\}$ where $R_j$ is the affected version range:

$$\text{vulns}(P) = \{(p_i, \text{CVE}_j) \mid (p_i, v_i) \in P \wedge (p_i, R_j, \text{CVE}_j) \in V \wedge v_i \in R_j\}$$

Version range check:

$$v \in R = [v_{\min}, v_{\text{fix}}) \iff v_{\min} \leq_{\text{semver}} v <_{\text{semver}} v_{\text{fix}}$$

Total vulnerability count:

$$N_{\text{vuln}} = |\text{vulns}(P)|$$

### Worked Examples

Image has 150 packages. Database has 200,000 CVEs.

| Package | Version | CVEs Matched | Reason |
|:---|:---|:---:|:---|
| openssl | 3.1.2 | 3 | versions < 3.1.4 affected |
| curl | 8.4.0 | 1 | versions 8.0.0-8.4.0 affected |
| glibc | 2.38 | 2 | versions < 2.38-5 affected |
| busybox | 1.36.1 | 0 | no known CVEs for this version |

$$N_{\text{vuln}} = 3 + 1 + 2 + 0 = 6$$

With `--ignore-unfixed` filtering: if 2 of 6 have no fix available:

$$N_{\text{actionable}} = 6 - 2 = 4$$

---

## 2. CVSS Scoring (Nonlinear Algebra)

### The Problem

Each vulnerability has a CVSS v3.1 base score computed from attack vector, complexity, privileges required, user interaction, scope, and CIA impact metrics using a specific nonlinear formula.

### The Formula

Base Score computation:

$$\text{ISS} = 1 - [(1 - C_I)(1 - I_I)(1 - A_I)]$$

where $C_I, I_I, A_I \in \{0, 0.22, 0.56\}$ (None, Low, High) for Confidentiality, Integrity, Availability.

Impact sub-score (scope unchanged):

$$\text{Impact} = 6.42 \times \text{ISS}$$

Impact sub-score (scope changed):

$$\text{Impact} = 7.52 \times [\text{ISS} - 0.029] - 3.25 \times [\text{ISS} - 0.02]^{15}$$

Exploitability:

$$\text{Exploit} = 8.22 \times AV \times AC \times PR \times UI$$

Base score:

$$\text{Base} = \begin{cases} 0 & \text{if Impact} \leq 0 \\ \min(10, \lceil \text{Impact} + \text{Exploit} \rceil_{0.1}) & \text{scope unchanged} \\ \min(10, \lceil 1.08 \times (\text{Impact} + \text{Exploit}) \rceil_{0.1}) & \text{scope changed} \end{cases}$$

where $\lceil x \rceil_{0.1}$ rounds up to nearest 0.1.

### Worked Examples

CVE with: AV=Network(0.85), AC=Low(0.77), PR=None(0.85), UI=None(0.68), Scope=Unchanged, C=High(0.56), I=High(0.56), A=High(0.56):

$$\text{ISS} = 1 - [(1-0.56)(1-0.56)(1-0.56)] = 1 - 0.0850 = 0.9150$$

$$\text{Impact} = 6.42 \times 0.9150 = 5.874$$

$$\text{Exploit} = 8.22 \times 0.85 \times 0.77 \times 0.85 \times 0.68 = 3.114$$

$$\text{Base} = \min(10, \lceil 5.874 + 3.114 \rceil_{0.1}) = \min(10, 9.0) = 9.0 \text{ (Critical)}$$

---

## 3. Transitive Dependency Vulnerability (Graph Theory)

### The Problem

SBOM-based scanning must discover transitive vulnerabilities through the dependency graph. A direct dependency may be safe, but its transitive dependencies may be vulnerable.

### The Formula

Dependency graph $G = (N, E)$ where $N$ = packages and $E$ = dependency edges.

Transitive closure of direct dependencies $D$:

$$T(D) = D \cup \bigcup_{d \in D} T(\text{deps}(d))$$

Transitive vulnerability exposure:

$$V_{\text{transitive}} = \text{vulns}(T(D)) \setminus \text{vulns}(D)$$

Exposure depth:

$$\text{depth}(v) = \min_{p \in \text{path}(\text{root}, \text{pkg}(v))} |p|$$

### Worked Examples

App depends on framework A, which depends on library B, which depends on crypto C:

```
myapp -> framework-A (safe)
           -> lib-B (safe)
              -> crypto-C v1.2 (CVE-2024-1234, CRITICAL)
```

$$|D| = 1, \quad |T(D)| = 3$$

$$V_{\text{transitive}} = \{\text{CVE-2024-1234}\}$$

$$\text{depth}(\text{CVE-2024-1234}) = 3$$

Direct scan of myapp finds 0 vulnerabilities. SBOM scan finds 1 critical at depth 3.

---

## 4. Risk Aggregation (Statistics)

### The Problem

An image may have dozens of vulnerabilities. Aggregate risk must consider both the count and severity distribution to produce a meaningful risk score.

### The Formula

Aggregate risk score:

$$R = \sum_{v \in V} w(\text{severity}(v)) \cdot \text{CVSS}(v)$$

where severity weights:

$$w(s) = \begin{cases} 1.0 & s = \text{CRITICAL} \\ 0.6 & s = \text{HIGH} \\ 0.3 & s = \text{MEDIUM} \\ 0.1 & s = \text{LOW} \end{cases}$$

Normalized risk (0-100 scale):

$$R_{\text{norm}} = \min\left(100, \frac{R}{N_{\text{max}}} \times 100\right)$$

where $N_{\text{max}}$ is a calibration constant (e.g., 50 for a baseline threshold).

### Worked Examples

Image with: 2 CRITICAL (9.8, 9.1), 3 HIGH (7.5, 7.2, 8.1), 5 MEDIUM (5.4, 4.9, 6.2, 5.8, 5.0):

$$R = 1.0(9.8 + 9.1) + 0.6(7.5 + 7.2 + 8.1) + 0.3(5.4 + 4.9 + 6.2 + 5.8 + 5.0)$$

$$R = 18.9 + 13.68 + 8.49 = 41.07$$

$$R_{\text{norm}} = \min(100, \frac{41.07}{50} \times 100) = 82.1$$

Risk classification: High (threshold > 70).

---

## 5. Database Freshness and False Negatives (Probability)

### The Problem

Trivy depends on vulnerability databases that lag behind CVE disclosure. The probability of missing a vulnerability depends on database update frequency and CVE publication rate.

### The Formula

For database update interval $\Delta t$ and CVE publication rate $\lambda$ (CVEs/day):

Expected missed CVEs at any point:

$$E[\text{missed}] = \lambda \cdot \frac{\Delta t}{2}$$

Probability of missing at least one critical CVE (fraction $f_c$ of CVEs are critical):

$$P(\text{miss\_critical}) = 1 - e^{-\lambda \cdot f_c \cdot \Delta t}$$

### Worked Examples

$\lambda = 80$ CVEs/day, $f_c = 0.05$ (5% critical), $\Delta t = 1$ day:

$$E[\text{missed}] = 80 \times 0.5 = 40 \text{ CVEs}$$

$$P(\text{miss\_critical}) = 1 - e^{-80 \times 0.05 \times 1} = 1 - e^{-4} = 0.982$$

With $\Delta t = 6$ hours (0.25 days):

$$E[\text{missed}] = 80 \times 0.125 = 10$$

$$P(\text{miss\_critical}) = 1 - e^{-1} = 0.632$$

---

## 6. IaC Misconfiguration Detection (Policy Logic)

### The Problem

IaC scanning evaluates configurations against security policies expressed as logical rules. Each policy is a predicate over the configuration structure.

### The Formula

Policy $\pi$ is a function:

$$\pi: \text{Config} \to \{\text{PASS}, \text{FAIL}, \text{SKIP}\}$$

For configuration $c$ and policy set $\Pi$:

$$\text{findings}(c) = \{(\pi, \text{FAIL}) \mid \pi \in \Pi \wedge \pi(c) = \text{FAIL}\}$$

Compliance score:

$$\text{compliance} = \frac{|\{\pi \in \Pi \mid \pi(c) = \text{PASS}\}|}{|\{\pi \in \Pi \mid \pi(c) \neq \text{SKIP}\}|}$$

### Worked Examples

Terraform config scanned against 50 applicable policies:

| Result | Count |
|:---|:---:|
| PASS | 38 |
| FAIL | 9 |
| SKIP | 3 |

$$\text{compliance} = \frac{38}{38 + 9} = \frac{38}{47} = 0.809 = 80.9\%$$

By severity: 2 CRITICAL fails, 3 HIGH, 4 MEDIUM.

Weighted compliance: $\frac{38 \times 1.0}{38 + 2 \times 4.0 + 3 \times 2.0 + 4 \times 1.0} = \frac{38}{56} = 67.9\%$

---

## Prerequisites

- set-theory, nonlinear-algebra, graph-theory, statistics, probability, predicate-logic, cvss
