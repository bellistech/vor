# The Mathematics of Threat Hunting — Hypothesis-Driven Detection

> *Threat hunting is proactive search for adversaries that evade automated detection. It applies Bayesian reasoning to develop hypotheses, statistical analysis to identify anomalies, and graph traversal to map attack paths — converting analyst intuition into quantifiable detection probability.*

---

## 1. Hypothesis-Driven Hunting

### Bayesian Framework

A threat hunting hypothesis:

$$P(\text{threat} \mid \text{evidence}) = \frac{P(\text{evidence} \mid \text{threat}) \times P(\text{threat})}{P(\text{evidence})}$$

| Term | Meaning | Source |
|:---|:---|:---|
| $P(\text{threat})$ | Prior probability | Threat intel, industry trends |
| $P(\text{evidence} \mid \text{threat})$ | Likelihood | How often this evidence appears with this threat |
| $P(\text{evidence})$ | Base rate | How often this evidence appears normally |
| $P(\text{threat} \mid \text{evidence})$ | Posterior | Updated belief after observation |

### Worked Example: Lateral Movement Hunt

Hypothesis: "An attacker is using PsExec for lateral movement."

| Variable | Value | Rationale |
|:---|:---:|:---|
| $P(\text{threat})$ | 0.05 | 5% prior (recent sector targeting) |
| $P(\text{PsExec} \mid \text{threat})$ | 0.60 | 60% of lateral movement uses PsExec |
| $P(\text{PsExec} \mid \text{no threat})$ | 0.02 | 2% of admins use PsExec legitimately |
| $P(\text{PsExec})$ | $0.05 \times 0.6 + 0.95 \times 0.02 = 0.049$ | |

$$P(\text{threat} \mid \text{PsExec}) = \frac{0.6 \times 0.05}{0.049} = 0.612 = 61.2\%$$

Finding PsExec execution raises the probability from 5% to 61.2% — enough to investigate.

---

## 2. Anomaly Detection — Statistical Methods

### Rare Event Detection

For events following a Poisson distribution with rate $\lambda$:

$$P(X = k) = \frac{\lambda^k e^{-\lambda}}{k!}$$

If a user normally creates 2 service accounts per month ($\lambda = 2$):

$$P(X \geq 10) = 1 - \sum_{k=0}^{9} \frac{2^k e^{-2}}{k!} = 0.000083 = 0.0083\%$$

Seeing 10 service accounts created is a strong anomaly signal.

### Behavioral Baselines

| Behavior | Baseline ($\mu$) | Alert Threshold | Detection Target |
|:---|:---:|:---:|:---|
| Failed logins per user per day | 2 | > 20 | Brute force |
| PowerShell executions per host | 5 | > 50 | Living off the land |
| New scheduled tasks per week | 0.5 | > 5 | Persistence |
| Data transferred (MB) per host | 100 | > 5,000 | Exfiltration |
| Unique destinations per host | 50 | > 500 | C2 beaconing / scanning |

---

## 3. Beaconing Detection — Periodic Communication

### Beacon Characteristics

C2 beacons communicate at regular intervals with jitter:

$$t_i = T + \text{Uniform}(-j, j)$$

Where $T$ is the beacon interval and $j$ is the jitter.

### Detection via Frequency Analysis

Apply FFT (Fast Fourier Transform) to connection timestamps:

$$F(\omega) = \sum_{n=0}^{N-1} x_n \cdot e^{-2\pi i \omega n / N}$$

A strong peak at frequency $f = 1/T$ indicates periodic communication.

### Jitter-Aware Detection

| Beacon Interval | Jitter | Detection Difficulty |
|:---:|:---:|:---|
| 60s | 0% | Trivial (exact periodicity) |
| 60s | 10% (6s) | Easy (clear frequency peak) |
| 60s | 50% (30s) | Moderate (broad peak) |
| Random (30-300s) | N/A | Hard (no clear period) |

### Statistical Test for Periodicity

Standard deviation of inter-arrival times:

$$\sigma_{IAT} = \sqrt{\frac{1}{n-1}\sum_{i=1}^{n}(\Delta t_i - \overline{\Delta t})^2}$$

| Ratio $\sigma / \mu$ | Interpretation |
|:---:|:---|
| < 0.1 | Strong periodicity (likely beacon) |
| 0.1-0.3 | Moderate periodicity (possible beacon) |
| 0.3-0.5 | Weak periodicity (needs further analysis) |
| > 0.5 | Random (unlikely beacon) |

---

## 4. Hunt Maturity Model

### Maturity Levels

| Level | Name | Detection Method | Coverage |
|:---:|:---|:---|:---:|
| HM0 | Initial | Automated alerts only | 10-20% |
| HM1 | Minimal | IOC-driven searches | 20-35% |
| HM2 | Procedural | Documented hunt playbooks | 35-55% |
| HM3 | Innovative | Custom hypothesis hunts | 55-75% |
| HM4 | Leading | ML-assisted + custom analytics | 75-90% |

### Detection Gap

$$\text{Gap} = P(\text{threat exists}) \times (1 - P(\text{automated detect})) \times (1 - P(\text{hunt detect}))$$

| Automated Detection | Hunt Detection | Combined Gap |
|:---:|:---:|:---:|
| 60% | 0% (no hunting) | 40% |
| 60% | 50% | 20% |
| 60% | 80% | 8% |
| 80% | 80% | 4% |

Threat hunting halves the detection gap even at moderate maturity.

---

## 5. MITRE ATT&CK Hunt Queries

### Technique Frequency in Real Attacks

| ATT&CK ID | Technique | Frequency in APTs | Hunt Priority |
|:---|:---|:---:|:---:|
| T1059 | Command/Script Interpreter | 85% | Critical |
| T1053 | Scheduled Task/Job | 70% | High |
| T1071 | Application Layer Protocol | 65% | High |
| T1082 | System Information Discovery | 60% | Medium |
| T1055 | Process Injection | 55% | High |
| T1021 | Remote Services | 50% | High |
| T1003 | OS Credential Dumping | 45% | Critical |

### Hunt Prioritization Score

$$\text{Priority}(T) = w_1 \times \text{frequency}(T) + w_2 \times \text{impact}(T) + w_3 \times (1 - \text{detection}(T))$$

Where $w_1 + w_2 + w_3 = 1$ and detection is the current detection coverage.

High-frequency, high-impact, low-detection techniques should be hunted first.

---

## 6. Data Stacking — Finding Outliers

### The Technique

Stack counting: group by field and count occurrences. Outliers (count = 1 or very low) are suspicious.

$$\text{rarity}(v) = \frac{\text{count}(v)}{\text{total events}}$$

### Example: Process Name Stacking

| Process Name | Count | Rarity | Assessment |
|:---|:---:|:---:|:---|
| svchost.exe | 15,000 | 0.30 | Normal |
| chrome.exe | 8,000 | 0.16 | Normal |
| explorer.exe | 5,000 | 0.10 | Normal |
| csrss.exe | 3,000 | 0.06 | Normal |
| svch0st.exe | 2 | 0.00004 | **Suspicious** (typosquat) |
| update-helper.exe | 1 | 0.00002 | **Investigate** |

### Statistical Threshold

Flag values where:

$$\text{count}(v) < \mu_{counts} - 2\sigma_{counts}$$

Or using percentile: flag the bottom 1% of value frequencies.

---

## 7. Graph-Based Hunting — Lateral Movement Paths

### Authentication Graph

$$G_{auth} = (H, A) \quad \text{where } (h_i, h_j) \in A \iff h_i \text{ authenticated to } h_j$$

### Anomalous Path Detection

Normal authentication graph is sparse (users access few systems):

$$\text{Normal degree}: d(h) = 3-10$$

Compromised account: fans out to many systems:

$$\text{Anomalous}: d(h) > \mu_d + 3\sigma_d$$

### Path Length as Indicator

$$\text{Shortest path from compromise to crown jewels} = \delta(c, t)$$

| Path Length | Risk | Interpretation |
|:---:|:---|:---|
| 1 | Critical | Direct access to target |
| 2 | High | One hop away |
| 3-4 | Medium | Multi-hop required |
| $\infty$ (unreachable) | Low | Properly segmented |

### Hunt Query: Unusual Authentication Chains

Find: $h_1 \to h_2 \to h_3$ where $h_1 \to h_3$ has never been seen:

$$\text{Anomalous chain} = \{(h_1, h_2, h_3) : (h_1, h_3) \notin A_{historical}\}$$

---

## 8. Summary of Functions by Type

| Concept | Math Type | Application |
|:---|:---|:---|
| Bayes' theorem | Conditional probability | Hypothesis update |
| Poisson $P(X \geq k)$ | Discrete probability | Rare event detection |
| FFT periodicity | Frequency analysis | Beacon detection |
| $\sigma/\mu$ ratio | Coefficient of variation | Jitter classification |
| Stack counting | Frequency distribution | Outlier detection |
| Graph degree $d(h)$ | Graph theory | Lateral movement |
| Kill chain product | Independence product | Detection gap |

---

*Threat hunting is the mathematical complement to automated detection — where SIEM rules catch known patterns, hunting uses Bayesian reasoning and statistical analysis to find the threats that signatures miss, closing the gap between automated detection and actual adversary presence.*
