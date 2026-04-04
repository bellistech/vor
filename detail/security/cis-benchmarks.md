# The Mathematics of CIS Benchmarks -- Compliance Scoring and Configuration Risk

> *A system is only as secure as its weakest setting; quantifying configuration risk transforms hardening from checklist compliance into measurable risk reduction.*

---

## 1. Benchmark Compliance Scoring (Weighted Pass/Fail)

### The Problem

CIS benchmarks contain hundreds of recommendations with varying security impact. A simple pass/fail percentage treats all recommendations equally, but disabling root SSH login (critical) should not carry the same weight as setting a login banner (informational). Weighted scoring enables meaningful compliance comparisons across systems and time.

### The Formula

For a benchmark with n scored recommendations, each with weight $w_i$ (derived from severity) and result $r_i \in \{0, 1\}$:

$$\text{Score} = \frac{\sum_{i=1}^{n} w_i \cdot r_i}{\sum_{i=1}^{n} w_i} \times 100\%$$

The CIS-CAT scoring model uses severity-based weights:

$$w_i = \begin{cases} 5 & \text{Critical (authentication, access control)} \\ 3 & \text{High (logging, firewall, encryption)} \\ 2 & \text{Medium (service hardening, permissions)} \\ 1 & \text{Low (banners, documentation)} \end{cases}$$

The marginal risk reduction from fixing recommendation i:

$$\Delta R_i = \frac{w_i}{\sum_{j=1}^{n} w_j} \times (1 - r_i)$$

Priority ranking for remediation: sort by $\Delta R_i / c_i$ where $c_i$ is the estimated remediation effort.

### Worked Examples

**Example 1: Linux server compliance**

200 scored recommendations. Current results: 160 pass, 40 fail.
Simple percentage: 160/200 = 80%.

Weighted analysis:
- 15 Critical fails (w=5): total weight contribution = 75
- 10 High fails (w=3): contribution = 30
- 10 Medium fails (w=2): contribution = 20
- 5 Low fails (w=1): contribution = 5
- Total weight of all recommendations: sum = 650
- Weight of passing: 650 - 130 = 520
- Weighted score: 520/650 = 80.0%

After fixing 15 Critical items: (520 + 75)/650 = 91.5% (11.5 point gain from 7.5% of recommendations).

**Example 2: Remediation prioritization**

Top 3 failing recommendations:
- R1: PermitRootLogin=yes (w=5, effort=0.5hr). Priority = 5/0.5 = 10.0
- R2: No disk encryption (w=5, effort=8hr). Priority = 5/8 = 0.625
- R3: Firewall disabled (w=5, effort=1hr). Priority = 5/1 = 5.0

Order: R1, R3, R2. Fix the quickest high-impact items first.

## 2. Configuration Drift Detection (Statistical Process Control)

### The Problem

Hardened systems drift from their baseline over time due to patches, software installations, and manual changes. Detecting statistically significant drift from the expected compliance state requires modeling normal variation (from maintenance windows and approved changes) versus anomalous regression.

### The Formula

Let $S_t$ be the compliance score at time t. The expected score follows a process:

$$S_t = S_0 - \lambda \cdot t + \sum_{k=1}^{m_t} \delta_k$$

where $\lambda$ is the natural drift rate (score degradation per day), $m_t$ is the number of remediation events, and $\delta_k$ is the score improvement from each remediation.

Using control chart methodology, the process is "in control" when:

$$\bar{S} - 3\sigma_S \leq S_t \leq \bar{S} + 3\sigma_S$$

where $\bar{S}$ and $\sigma_S$ are computed from the historical compliance score series. An out-of-control signal at time t:

$$|S_t - \bar{S}| > 3\sigma_S \implies \text{investigate}$$

The CUSUM (cumulative sum) detector is more sensitive to gradual drift:

$$C_t^+ = \max(0, C_{t-1}^+ + (S_0 - S_t) - k)$$

where k is the allowable slack (typically $0.5\sigma$). An alarm fires when $C_t^+ > h$ (decision interval, typically $4\sigma$ to $5\sigma$).

### Worked Examples

**Example 1: Monthly compliance monitoring**

Baseline score S_0 = 92%. Monthly scores: [92, 91, 90, 91, 88, 85, 84].

Mean = 88.7%, sigma = 3.15%.
Control limits: 88.7 +/- 9.45 = [79.3, 98.2].
Month 7 (84%) is within limits -- no alarm despite downward trend.

CUSUM with k = 1.5, h = 10:
- Month 1: C = max(0, 0 + (92-92) - 1.5) = 0
- Month 2: C = max(0, 0 + (92-91) - 1.5) = 0
- Month 3: C = max(0, 0 + (92-90) - 1.5) = 0.5
- Month 4: C = max(0, 0.5 + (92-91) - 1.5) = 0
- Month 5: C = max(0, 0 + (92-88) - 1.5) = 2.5
- Month 6: C = max(0, 2.5 + (92-85) - 1.5) = 8.0
- Month 7: C = max(0, 8.0 + (92-84) - 1.5) = 14.5 > 10 -- ALARM

CUSUM detects the gradual drift that the control chart missed.

**Example 2: Drift rate estimation**

Fleet of 50 servers. After 90 days without remediation, average score dropped from 94% to 87%.
- Drift rate: lambda = (94 - 87) / 90 = 0.078% per day
- Time to drop below 80% threshold: (94 - 80) / 0.078 = 180 days
- Required re-hardening interval: every 90 days to maintain >87%

## 3. Attack Surface Quantification (Combinatorial Exposure)

### The Problem

Each failing CIS recommendation potentially opens an attack path. When multiple settings fail simultaneously, they may combine to create compound vulnerabilities that are worse than the sum of individual risks. Quantifying the combinatorial attack surface helps prioritize compound remediation.

### The Formula

Let F = {f_1, f_2, ..., f_k} be the set of failing recommendations. Each failure opens an attack surface $A_i$ measured in potential attack vectors. Independent failures contribute additively:

$$A_{\text{total}} = \sum_{i=1}^{k} A_i$$

But compound failures create multiplicative exposure. For pairs of interacting failures:

$$A_{\text{compound}} = \sum_{i=1}^{k} A_i + \sum_{i<j} \alpha_{ij} \cdot A_i \cdot A_j$$

where $\alpha_{ij}$ is the interaction coefficient between failures i and j. Some combinations are particularly dangerous:

$$\alpha_{ij} = \begin{cases} 1.0 & \text{root SSH + weak password (full remote root)} \\ 0.5 & \text{no FIM + world-writable dirs (undetected modification)} \\ 0.0 & \text{independent failures (no interaction)} \end{cases}$$

The risk reduction from fixing failure f_i:

$$\Delta A_i = A_i + \sum_{j \neq i} \alpha_{ij} \cdot A_i \cdot A_j$$

### Worked Examples

**Example 1: SSH + password compound failure**

Failures: PermitRootLogin=yes (A_1 = 10 vectors), weak password policy (A_2 = 8 vectors), no fail2ban (A_3 = 5 vectors).

Interaction: alpha_12 = 0.8 (root + weak password), alpha_13 = 0.3, alpha_23 = 0.4.

A_compound = (10 + 8 + 5) + (0.8 * 10 * 8 + 0.3 * 10 * 5 + 0.4 * 8 * 5) = 23 + 64 + 15 + 16 = 118

Fixing PermitRootLogin alone: Delta_A_1 = 10 + (0.8 * 10 * 8) + (0.3 * 10 * 5) = 10 + 64 + 15 = 89.
This single fix eliminates 75% of the compound attack surface.

**Example 2: Container escape chain**

Docker failures: privileged=true (A_1=20), host network (A_2=15), no seccomp (A_3=12).
All three interacting (alpha = 1.0 for container escape chain):

A_compound = 47 + (1.0 * 20 * 15) + (1.0 * 20 * 12) + (1.0 * 15 * 12) = 47 + 300 + 240 + 180 = 767.
Individual surface: 47. Compound amplification: 16.3x.

## 4. Compliance Sampling and Audit Confidence (Hypergeometric Distribution)

### The Problem

In large environments, auditing every system against every CIS recommendation is impractical. Sampling strategies must provide statistical confidence that the fleet's compliance rate meets a target threshold. The auditor must determine how many systems to sample and how many recommendations to spot-check.

### The Formula

For a population of N systems with unknown non-compliance rate p, drawing a sample of n systems, the probability of observing exactly k non-compliant systems follows the hypergeometric distribution:

$$P(X = k) = \frac{\binom{Np}{k}\binom{N-Np}{n-k}}{\binom{N}{n}}$$

For large N, this approximates to binomial. The sample size n needed to detect non-compliance rate p with confidence $1 - \alpha$:

$$n = \frac{\ln(\alpha)}{\ln(1-p)}$$

For a two-sided confidence interval on the true compliance rate $\hat{p}$ with margin of error $\epsilon$:

$$n = \frac{z_{\alpha/2}^2 \cdot \hat{p}(1-\hat{p})}{\epsilon^2}$$

The probability that all sampled systems pass when the true fleet compliance rate is q:

$$P(\text{all pass}) = q^n$$

### Worked Examples

**Example 1: Fleet audit sampling**

Fleet: N = 1,000 servers. Target: 95% compliance. Desired confidence: 99%.

To detect if compliance is below 95% (p = 0.05 non-compliance):
n = ln(0.01) / ln(1-0.05) = -4.605 / -0.0513 = 90 systems.

Sampling 90 of 1,000 systems: if all 90 pass, 99% confidence that non-compliance < 5%.
If 5 of 90 fail: estimated non-compliance = 5.6%, 95% CI = [1.8%, 12.6%].

**Example 2: Spot-check recommendations**

200 scored recommendations per system, auditing 30 systems with 50 recommendations each.

Total checks: 30 * 50 = 1,500 out of 30 * 200 = 6,000 possible.
Coverage: 25% of recommendation-system pairs.

To estimate fleet compliance within +/- 2% with 95% confidence:
n_checks = (1.96^2 * 0.9 * 0.1) / 0.02^2 = 864 checks.
We have 1,500 > 864: sufficient for this precision.

## Prerequisites

- Descriptive statistics (weighted means, standard deviation)
- Statistical process control (control charts, CUSUM)
- Combinatorics (interaction effects, compound risk)
- Sampling theory (hypergeometric and binomial distributions)
- Confidence intervals and hypothesis testing
- Risk analysis frameworks (attack surface modeling)
- Optimization (cost-effectiveness ratios for remediation prioritization)
