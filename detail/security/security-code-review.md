# The Mathematics of Security Code Review — Vulnerability Density and Detection Theory

> *Security code review is a search problem over a high-dimensional space of possible program behaviors. The reviewer must distinguish between the exponentially many safe execution paths and the handful of unsafe ones — a task whose difficulty is governed by vulnerability density, false-positive rates of pattern matching, and the information-theoretic limits of human attention.*

---

## 1. Vulnerability Density Models (Defect Estimation)

### Defect Rate by Language

Empirical studies measure vulnerability density as defects per thousand lines of code (KLOC):

$$\rho = \frac{|\text{vulnerabilities}|}{|\text{KLOC}|}$$

| Language | Avg. Defect Density | Memory-Safety | Logic Defects |
|:---|:---:|:---:|:---:|
| C | 0.5 - 2.0 | 40-60% | 20-30% |
| C++ | 0.5 - 1.5 | 35-50% | 25-35% |
| Java | 0.2 - 0.8 | 0% (GC) | 60-80% |
| Go | 0.1 - 0.5 | ~0% (safe) | 70-90% |
| Python | 0.2 - 0.6 | 0% | 80-95% |
| Rust | 0.05 - 0.2 | ~0% (safe) | 85-95% |

### Expected Vulnerabilities

For a codebase of $L$ KLOC: $E[\text{vulns}] = \rho \times L$

A 100 KLOC Go codebase with $\rho = 0.3$: approximately 30 vulnerabilities.

### Defect Discovery Curve

Vulnerabilities found after $t$ hours of review:

$$V(t) = V_{\text{total}} \times (1 - e^{-\lambda t})$$

| Reviewer Expertise | $\lambda$ (vulns/hour) | Hours for 90% |
|:---|:---:|:---:|
| Junior | 0.1 - 0.3 | 8 - 23 |
| Senior | 0.3 - 0.8 | 3 - 8 |
| Expert | 0.8 - 2.0 | 1 - 3 |

---

## 2. Integer Overflow — Modular Arithmetic

### Unsigned Overflow

For $n$-bit unsigned with max $M = 2^n - 1$: $a + b \bmod 2^n$. Overflow when $a > M - b$.

### Width Truncation

Casting from $n$-bit to $m$-bit ($m < n$): $\text{truncate}(x) = x \bmod 2^m$

| Source | Target | Max Loss | Example |
|:---|:---|:---:|:---|
| int64 | int32 | $2^{32}$ values | 4294967296 becomes 0 |
| int32 | int16 | $2^{16}$ values | 65536 becomes 0 |
| uint64 | int64 | sign | $2^{63}$ becomes $-2^{63}$ |

### Multiplication Overflow Probability

For random 32-bit values: $P(\text{overflow}) \approx 0.97$. Safe check: $a > 0 \land b > M/a$.

---

## 3. Race Condition Detection — Happens-Before

### Data Race Definition

A data race between accesses $a, b$ to shared $x$ if:

$$a \not\rightarrow b \land b \not\rightarrow a \land (\text{is\_write}(a) \lor \text{is\_write}(b))$$

### Race Manifestation Probability

$$P(\text{race}) = \frac{T_{\text{window}}}{T_{\text{schedule}}}$$

### Go Race Detector

| Metric | Without `-race` | With `-race` |
|:---|:---:|:---:|
| CPU overhead | 1x | 2-20x |
| Memory overhead | 1x | 5-10x |
| Detection rate | 0% | ~99% (dynamic) |
| False positives | N/A | ~0% |

The race detector uses vector clocks with $O(n)$ space per goroutine.

---

## 4. Injection Attack Surface — Formal Language Theory

### Injection as Grammar Confusion

Injection exists when: $\text{parse}(f(q, u)) \neq \text{intended\_parse}(q, u)$

where $f$ is query construction and $u$ is user input.

### Parameterization Rate

$$\text{Injection risk} = \frac{Q - P}{Q}$$

| Parameterization | Risk | Expected per 100 Queries |
|:---|:---:|:---:|
| 100% | None | 0 |
| 95% | Low | 5 |
| 80% | Medium | 20 |
| < 50% | Critical | 50+ |

---

## 5. Cryptographic Misuse — Entropy Analysis

### CSPRNG Requirement

$$\forall i: |P(b_i = 1 | b_1, \ldots, b_{i-1}) - 0.5| < \epsilon \text{ (negligible)}$$

Non-CSPRNG (math/rand): $H(\text{next} | \text{observed}) \approx 0$ (fully predictable).

| Use Case | Min Entropy (bits) | Example |
|:---|:---:|:---|
| Session token | 128 | 32 hex chars |
| Encryption key | 256 | 32 bytes |
| CSRF token | 128 | 32 hex chars |
| Nonce (GCM) | 96 | 12 bytes |

### Weak RNG Detection

Reviewing $k$ of $n$ call sites ($w$ weak): $P(\text{find weak}) = 1 - \binom{n-w}{k}/\binom{n}{k}$

For $n=20$, $w=3$, $k=10$: $P = 0.895$.

---

## 6. TOCTOU Race Window — Probability Theory

### File System TOCTOU

Between check at $t_c$ and use at $t_u$: $T_{\text{window}} = t_u - t_c$

Over $N$ attempts with window $w$ and replacement time $r$:

$$P(\text{success}) = 1 - \left(1 - \frac{w}{w + r}\right)^N$$

| Window $w$ | Replace $r$ | Single | 1000 Attempts |
|:---|:---:|:---:|:---:|
| 1 us | 10 us | 9.1% | 100% |
| 1 us | 1 ms | 0.1% | 63.2% |
| 100 ns | 1 ms | 0.01% | 9.5% |

Atomic operations: $T_{\text{window}} = 0 \implies P = 0$.

---

## 7. Review Coverage — Search Theory

### Review Effectiveness

Expected vulnerabilities found: $E = V \times (R/N) \times d$

| Review Type | Lines/Hour | Detection $d$ | $R \times d$/Hour |
|:---|:---:|:---:|:---:|
| Quick scan | 500-1000 | 0.1-0.2 | 50-200 |
| Normal review | 100-200 | 0.4-0.6 | 40-120 |
| Deep audit | 20-50 | 0.8-0.95 | 16-48 |
| Tool-assisted | 200-400 | 0.5-0.7 | 100-280 |

Tool-assisted review dominates all other strategies by 2-5x.

### Two-Reviewer Coverage

With independent reviewers at detection probability $d$:

$$P(\text{at least one finds it}) = 1 - (1-d)^2 = 2d - d^2$$

For $d = 0.5$: single reviewer 50%, two reviewers 75%.

---

## 8. Dependency Risk — Supply Chain Attack Surface

### Transitive Vulnerability Probability

If each of $n$ dependencies has probability $p$ of a known vulnerability:

$$P(\text{at least one vuln}) = 1 - (1-p)^n$$

| Dependencies ($n$) | $p = 0.01$ | $p = 0.05$ | $p = 0.10$ |
|:---|:---:|:---:|:---:|
| 10 | 9.6% | 40.1% | 65.1% |
| 50 | 39.5% | 92.3% | 99.5% |
| 100 | 63.4% | 99.4% | 100% |
| 500 | 99.3% | 100% | 100% |

### Risk Prioritization

$$\text{Risk}(i) = P(\text{vuln}_i) \times \text{reachability}(i) \times \text{impact}(i)$$

Reachability measures whether the vulnerable code path is actually invoked.

---

*The central challenge is that vulnerability density is low (0.1-2 per KLOC) while code volume is high, creating a needle-in-a-haystack search. Tool-assisted review dominates pure manual review, two independent reviewers achieve meaningful coverage improvement, and dependency chains amplify vulnerability probability exponentially. The optimal strategy combines automated scanning for known patterns with focused manual review of trust boundary crossings, authentication logic, and cryptographic operations.*

## Prerequisites

- Programming language semantics (at least one of Go, Python, C, or JavaScript)
- Common vulnerability classes (OWASP Top 10, CWE Top 25)
- Basic probability and statistics for defect estimation and coverage modeling

## Complexity

- **Beginner:** Using grep-based pattern matching, running govulncheck/npm audit, input validation review
- **Intermediate:** TOCTOU identification, integer overflow analysis, injection detection across languages
- **Advanced:** Happens-before race analysis, formal injection grammar theory, optimal review strategy calculation
