# The Mathematics of SAST & DAST -- Taint Analysis, Fuzzing Coverage, and Detection Theory

> *Static analysis performs abstract interpretation and taint propagation over control flow graphs to find source-to-sink vulnerabilities, while dynamic analysis applies fuzzing coverage metrics and attack grammar generation to discover exploitable flaws at runtime, with their complementary detection capabilities modeled through precision-recall tradeoffs.*

---

## 1. Taint Analysis (Dataflow on Control Flow Graphs)

### The Problem

SAST tools like Semgrep and CodeQL track data from untrusted sources (user input) through the program to security-sensitive sinks (SQL queries, file writes, command execution). A taint propagation vulnerability exists when there is a path in the control flow graph from a tainted source to an unsanitized sink.

### The Formula

Control flow graph $G = (N, E)$ with nodes $N$ (statements) and edges $E$ (control flow).

Taint set at node $n$:

$$\text{Taint}(n) = \text{gen}(n) \cup (\text{Taint}_{\text{in}}(n) \setminus \text{kill}(n))$$

$$\text{Taint}_{\text{in}}(n) = \bigcup_{p \in \text{pred}(n)} \text{Taint}(p)$$

Source nodes: $S = \{n : \text{gen}(n) \neq \emptyset\}$ (user input, file reads, network).

Sink nodes: $K = \{n : \text{is\_sink}(n) = \text{true}\}$ (SQL, exec, eval, write).

Vulnerability: $\exists n \in K : \text{Taint}_{\text{in}}(n) \cap \text{tainted\_vars}(n) \neq \emptyset$.

Sanitizers: kill taint along a path. $\text{kill}(n) = \text{sanitized\_vars}(n)$.

### Worked Examples

Code: `user_input -> process() -> query(sql)`.

CFG nodes: $n_1$ (read input), $n_2$ (process), $n_3$ (build SQL), $n_4$ (execute query).

$$\text{Taint}(n_1) = \{x\}, \quad \text{Taint}(n_2) = \{x, y\}, \quad \text{Taint}(n_3) = \{x, y, q\}$$

At sink $n_4$: $\text{Taint}_{\text{in}}(n_4) = \{x, y, q\}$, $q$ used in SQL query.

$$\text{tainted\_vars}(n_4) \cap \text{Taint}_{\text{in}}(n_4) = \{q\} \neq \emptyset \implies \text{SQL injection}$$

With sanitizer at $n_{2.5}$ (parameterized query builder): $\text{kill}(n_{2.5}) = \{x, y\}$.

$$\text{Taint}(n_3) = \emptyset, \quad \text{Taint}_{\text{in}}(n_4) = \{q_{\text{safe}}\} \implies \text{no vulnerability}$$

---

## 2. Abstract Interpretation (Lattice Theory)

### The Problem

SAST tools use abstract interpretation to reason about all possible program executions without running the code. Program states are abstracted into a lattice, and fixpoint computation determines properties at each program point.

### The Formula

Abstract domain lattice $(L, \sqsubseteq, \sqcup, \sqcap, \bot, \top)$.

Abstract transfer function for statement $s$:

$$\hat{f}_s : L \to L$$

Fixpoint iteration:

$$X_n^{(0)} = \bot, \quad X_n^{(k+1)} = \hat{f}_n\left(\bigsqcup_{p \in \text{pred}(n)} X_p^{(k)}\right)$$

Convergence at fixpoint:

$$X_n^* = X_n^{(k)} \text{ when } X_n^{(k+1)} = X_n^{(k)} \quad \forall n$$

Widening operator $\nabla$ for guaranteed termination on infinite-height lattices:

$$X_n^{(k+1)} = X_n^{(k)} \nabla \hat{f}_n\left(\bigsqcup_{p \in \text{pred}(n)} X_p^{(k)}\right)$$

### Worked Examples

Integer range analysis: intervals $[a, b]$ where $a, b \in \mathbb{Z} \cup \{-\infty, +\infty\}$.

For `if x > 0 && x < 100: buf[x] = 1`:

At branch $x \in [1, 99]$: $[1, 99] \sqsubseteq [0, 99]$ (valid indices) $\implies$ safe.

At branch $x \in [100, +\infty]$: $[100, +\infty] \not\sqsubseteq [0, 99]$ $\implies$ buffer overflow reported.

---

## 3. Fuzzing Coverage (Code Coverage Metrics)

### The Problem

DAST fuzzing tools measure code coverage to guide input generation. Higher coverage correlates with higher vulnerability detection probability. Coverage metrics quantify how thoroughly the fuzzer has explored the target application.

### The Formula

Statement coverage:

$$C_{\text{stmt}} = \frac{|\text{executed\_statements}|}{|\text{total\_statements}|}$$

Branch coverage:

$$C_{\text{branch}} = \frac{|\text{taken\_branches}|}{|\text{total\_branches}|}$$

Path coverage (exponential in branches):

$$|\text{paths}| \leq 2^{|\text{branches}|}$$

$$C_{\text{path}} = \frac{|\text{explored\_paths}|}{|\text{total\_paths}|}$$

Expected bugs found as function of coverage:

$$E[\text{bugs}] = B \cdot (1 - (1 - C_{\text{branch}})^{\alpha})$$

where $B$ is the total bug count and $\alpha$ is a discovery exponent (typically 1.5-3).

### Worked Examples

Application with 10,000 statements, 2,500 branches.

After 1 hour of fuzzing: 7,500 statements executed, 1,800 branches taken.

$$C_{\text{stmt}} = \frac{7{,}500}{10{,}000} = 75\%$$

$$C_{\text{branch}} = \frac{1{,}800}{2{,}500} = 72\%$$

Total paths: $2^{2{,}500} \approx 10^{753}$ (exhaustive path coverage is impossible).

Expected bugs ($B = 20$, $\alpha = 2$):

$$E[\text{bugs}] = 20 \times (1 - (1 - 0.72)^2) = 20 \times (1 - 0.0784) = 20 \times 0.9216 = 18.4$$

At 50% branch coverage:

$$E[\text{bugs}] = 20 \times (1 - 0.50^2) = 20 \times 0.75 = 15.0$$

Diminishing returns: 72% coverage finds 18.4 bugs, but reaching 90% coverage would find $20 \times (1 - 0.01) = 19.8$.

---

## 4. SAST vs DAST Detection Complementarity (Set Theory)

### The Problem

SAST and DAST find overlapping but distinct vulnerability classes. SAST excels at code-level logic flaws visible in source, while DAST finds configuration and deployment issues only visible at runtime. Combining both maximizes total detection.

### The Formula

Total vulnerability set $V$. Detected by SAST: $D_S$. Detected by DAST: $D_D$.

Combined detection:

$$D_{\text{total}} = D_S \cup D_D$$

Unique to each:

$$U_S = D_S \setminus D_D, \quad U_D = D_D \setminus D_S$$

Overlap:

$$O = D_S \cap D_D$$

Coverage improvement from adding DAST to SAST:

$$\Delta_D = \frac{|D_{\text{total}}| - |D_S|}{|V|} = \frac{|U_D|}{|V|}$$

Recall of combined approach:

$$R_{\text{combined}} = \frac{|D_S \cup D_D|}{|V|}$$

### Worked Examples

Application with $|V| = 40$ known vulnerabilities (from manual audit).

| Category | SAST Finds | DAST Finds | Both |
|:---|:---:|:---:|:---:|
| SQL injection | 5 | 3 | 3 |
| XSS | 4 | 6 | 3 |
| SSRF | 2 | 1 | 1 |
| Hardcoded secrets | 8 | 0 | 0 |
| Misconfig (headers) | 0 | 5 | 0 |
| Auth bypass | 1 | 4 | 1 |
| **Total** | **20** | **19** | **8** |

$$|D_S| = 20, \quad |D_D| = 19, \quad |O| = 8$$

$$|D_{\text{total}}| = 20 + 19 - 8 = 31$$

$$R_S = \frac{20}{40} = 50\%, \quad R_D = \frac{19}{40} = 47.5\%, \quad R_{\text{combined}} = \frac{31}{40} = 77.5\%$$

Combined recall (77.5%) exceeds either tool alone by 27.5 and 30 percentage points respectively.

---

## 5. Secrets Scanning (Pattern Matching and Entropy)

### The Problem

Secrets scanners must distinguish true secrets (API keys, passwords, tokens) from benign strings that match similar patterns. Entropy measurement helps differentiate random secrets from non-secret strings that match regex patterns.

### The Formula

Shannon entropy of string $s$ of length $n$:

$$H(s) = -\sum_{c \in \text{alphabet}} \frac{f(c)}{n} \log_2 \frac{f(c)}{n}$$

where $f(c)$ is the frequency of character $c$ in $s$.

Maximum entropy for alphabet size $|\Sigma|$:

$$H_{\text{max}} = \log_2 |\Sigma|$$

Normalized entropy:

$$H_{\text{norm}} = \frac{H(s)}{H_{\text{max}}}$$

Secret detection rule:

$$\text{is\_secret}(s) = \text{regex\_match}(s) \wedge (H_{\text{norm}}(s) > \theta_H) \wedge (|s| \geq \ell_{\text{min}})$$
### Worked Examples

String 1: `AKIAIOSFODNN7EXAMPLE` (AWS access key, 20 chars, base-36).

$$H = -\sum \frac{f(c)}{20} \log_2 \frac{f(c)}{20}$$

Distinct chars: A(2), K(1), I(2), O(2), S(1), F(1), D(1), N(2), 7(1), E(2), X(1), M(1), P(1), L(1).

$$H \approx 3.52 \text{ bits}, \quad H_{\text{max}} = \log_2 36 = 5.17$$

$$H_{\text{norm}} = \frac{3.52}{5.17} = 0.681$$

With threshold $\theta_H = 0.5$: String 1 ($H_{\text{norm}} = 0.681$) flagged; `aaaaaaa...` ($H_{\text{norm}} = 0$) ignored.

---

## 6. Shift-Left Economics (Cost Functions)

### The Problem

The cost to fix a vulnerability increases exponentially with the stage at which it is discovered. Shift-left integrates security testing earlier to minimize total remediation cost.

### The Formula

Cost multiplier at stage $s$:

$$C(s) = C_0 \cdot k^s$$

where $C_0$ is the base cost (fixing during coding), $k$ is the escalation factor (typically 5-10x per stage), and $s$ is the stage index.

Stages: $s = 0$ (coding), $s = 1$ (build/CI), $s = 2$ (testing), $s = 3$ (staging), $s = 4$ (production).

Total remediation cost:

$$C_{\text{total}} = \sum_{s=0}^{4} N_s \cdot C_0 \cdot k^s$$

where $N_s$ is the number of vulnerabilities first detected at stage $s$.

### Worked Examples

$C_0 = \$100$ (developer fixes during coding), $k = 6$ (industry average).

| Stage | Multiplier | Cost/Fix |
|:---|:---:|:---:|
| Coding (SAST pre-commit) | $6^0 = 1$ | $100 |
| CI (SAST in pipeline) | $6^1 = 6$ | $600 |
| Testing (DAST) | $6^2 = 36$ | $3,600 |
| Staging (pentest) | $6^3 = 216$ | $21,600 |
| Production (incident) | $6^4 = 1{,}296$ | $129,600 |

Without shift-left: 20 bugs in production = $20 \times \$129{,}600 = \$2{,}592{,}000$.

With shift-left (15 coding, 3 CI, 1 testing, 1 staging):

$$C_{\text{shifted}} = 15(100) + 3(600) + 1(3{,}600) + 1(21{,}600) = \$28{,}500$$

Savings: $\$2{,}592{,}000 - \$28{,}500 = \$2{,}563{,}500$ (98.9% reduction).

---

## Prerequisites

- graph-theory, lattice-theory, set-theory, information-theory, probability, cost-optimization, abstract-interpretation
