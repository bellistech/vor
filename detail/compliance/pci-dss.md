# The Mathematics of PCI DSS — Network Segmentation Validation, Risk Scoring, and Cryptographic Key Lifecycle

> *The Payment Card Industry Data Security Standard transforms qualitative security goals into quantitative requirements: network segmentation effectivenes can be measured through reachability analysis, vulnerability risk scoring follows CVSS algebra, cryptographic key strength obeys information-theoretic bounds, and the probability of cardholder data exposure follows stochastic models rooted in Markov chain theory and combinatorial attack surfaces.*

---

## 1. Network Segmentation Validation (Graph Theory)
### The Problem
PCI DSS Requirement 11.4.5 mandates penetration testing to validate network segmentation. Formally, segmentation correctness means no path exists between out-of-scope nodes and the Cardholder Data Environment. This is a graph reachability problem.

### The Formula
Model the network as a directed graph $G = (V, E)$ where $V$ is the set of hosts and $E$ represents permitted network flows. Partition $V$ into CDE nodes $C$, connected nodes $N$, and out-of-scope nodes $O$.

Segmentation is valid if and only if:

$$\forall o \in O, \; \forall c \in C: \nexists \text{ path } (o \to c) \text{ in } G$$

Equivalently, define the reachability matrix $R$ where $R_{ij} = 1$ if a path exists from $i$ to $j$:

$$R = (A + I)^{|V|} \quad (\text{Boolean matrix, transitive closure})$$

Segmentation violation count:

$$V_{seg} = \sum_{o \in O} \sum_{c \in C} R_{oc}$$

Segmentation effectiveness ratio:

$$\eta_{seg} = 1 - \frac{V_{seg}}{|O| \times |C|}$$

Target: $\eta_{seg} = 1.0$ (perfect segmentation, zero violations).

### Worked Examples
**Example**: A network with 50 hosts: 5 CDE servers, 10 connected systems, 35 out-of-scope workstations.

After computing the transitive closure, 3 out-of-scope workstations can reach 2 CDE servers through a misconfigured jump host.

$$V_{seg} = 3 \times 2 = 6$$
$$\eta_{seg} = 1 - \frac{6}{35 \times 5} = 1 - \frac{6}{175} = 1 - 0.034 = 0.966$$

While 96.6% effective, PCI requires $\eta_{seg} = 1.0$. The 6 reachable paths must be eliminated by correcting the jump host ACLs.

## 2. CVSS-Based Vulnerability Prioritization (Multi-Factor Scoring)
### The Problem
Requirements 6.3.1 and 11.3 require vulnerability identification and risk ranking. The Common Vulnerability Scoring System (CVSS v3.1) provides a standardized formula that PCI DSS references for determining scan pass/fail thresholds.

### The Formula
CVSS Base Score is computed from sub-scores. The Impact Sub-Score (ISS):

$$ISS = 1 - (1 - C)(1 - I)(1 - A)$$

where $C$, $I$, $A$ are Confidentiality, Integrity, and Availability impact values.

For Scope Unchanged:

$$\text{Impact} = 6.42 \times ISS$$

For Scope Changed:

$$\text{Impact} = 7.52 \times (ISS - 0.029) - 3.25 \times (ISS - 0.02)^{15}$$

Exploitability Sub-Score:

$$\text{Exploitability} = 8.22 \times AV \times AC \times PR \times UI$$

Base Score (if Impact > 0):

$$\text{Base} = \begin{cases} \text{Roundup}\left(\min(Impact + Exploitability, 10)\right) & \text{Scope Unchanged} \\ \text{Roundup}\left(\min(1.08 \times (Impact + Exploitability), 10)\right) & \text{Scope Changed} \end{cases}$$

ASV scan pass threshold: no vulnerability with CVSS $\geq 4.0$.

### Worked Examples
**Example**: SQL injection in a payment application — network exploitable, low complexity, no privileges needed, no user interaction, scope unchanged, high confidentiality/integrity impact, no availability impact.

Metric values: $AV = 0.85$ (Network), $AC = 0.77$ (Low), $PR = 0.85$ (None), $UI = 0.85$ (None)

$$\text{Exploitability} = 8.22 \times 0.85 \times 0.77 \times 0.85 \times 0.85 = 3.887$$

$C = 0.56$ (High), $I = 0.56$ (High), $A = 0$ (None):

$$ISS = 1 - (1 - 0.56)(1 - 0.56)(1 - 0) = 1 - 0.1936 = 0.8064$$
$$\text{Impact} = 6.42 \times 0.8064 = 5.177$$

$$\text{Base} = \text{Roundup}(\min(5.177 + 3.887, 10)) = \text{Roundup}(9.064) = 9.1$$

CVSS 9.1 (Critical). This would cause an ASV scan failure and requires immediate remediation.

## 3. Cryptographic Key Strength (Information Theory)
### The Problem
Requirements 3.6 and 3.7 mandate strong cryptography with proper key management. The strength of encryption is bounded by key length and the algorithm's effective security level, measured in bits of entropy.

### The Formula
Effective key strength in bits:

$$S = \log_2(K)$$

where $K$ is the size of the key space.

Time to brute-force at $r$ operations per second:

$$T = \frac{2^S}{r}$$

For AES with key length $n$:

$$T_{AES-n} = \frac{2^n}{r}$$

Key rotation interval based on usage volume and cryptographic wear:

$$I_{rotate} = \frac{2^{S/2}}{D \times f}$$

where $D$ is the data volume encrypted per period and $f$ is the frequency of encryption operations (birthday bound consideration: after $2^{S/2}$ operations, collision probability becomes non-negligible for some modes).

### Worked Examples
**Example**: A payment processor encrypts PAN data with AES-256. They process 10 million transactions per day, each encrypting a 16-byte PAN.

Key strength: $S = 256$ bits

Brute-force time at $10^{18}$ operations/second (theoretical exaflop):

$$T = \frac{2^{256}}{10^{18}} = \frac{1.16 \times 10^{77}}{10^{18}} = 1.16 \times 10^{59} \text{ seconds} \approx 3.67 \times 10^{51} \text{ years}$$

Birthday bound for AES-256 in CBC mode (128-bit block):

$$\text{Block collision threshold} = 2^{64} \text{ blocks} \approx 1.84 \times 10^{19} \text{ blocks}$$

At 10M transactions/day (1 block each):

$$I_{rotate} = \frac{1.84 \times 10^{19}}{10^7} = 1.84 \times 10^{12} \text{ days} \approx 5 \times 10^9 \text{ years}$$

Cryptographically, no rotation needed due to wear. However, PCI DSS requires defining a crypto-period and rotating based on risk assessment (Req 3.7.4), so annual rotation is standard practice for operational security.

## 4. Transaction Fraud Detection (Bayesian Classification)
### The Problem
Requirement 10 (logging and monitoring) and Requirement 12 (risk assessment) intersect with fraud detection. Bayesian models quantify the probability that a transaction is fraudulent given observed features, informing monitoring thresholds.

### The Formula
Posterior probability of fraud given feature vector $\mathbf{x}$:

$$P(F \mid \mathbf{x}) = \frac{P(\mathbf{x} \mid F) \cdot P(F)}{P(\mathbf{x})}$$

Using the Naive Bayes assumption (features conditionally independent):

$$P(F \mid \mathbf{x}) = \frac{P(F) \prod_{i=1}^{n} P(x_i \mid F)}{P(F) \prod_{i=1}^{n} P(x_i \mid F) + P(\neg F) \prod_{i=1}^{n} P(x_i \mid \neg F)}$$

The log-odds ratio for decision:

$$\ln \frac{P(F \mid \mathbf{x})}{P(\neg F \mid \mathbf{x})} = \ln \frac{P(F)}{P(\neg F)} + \sum_{i=1}^{n} \ln \frac{P(x_i \mid F)}{P(x_i \mid \neg F)}$$

Flag as suspicious when log-odds exceed threshold $\tau$.

### Worked Examples
**Example**: A transaction has these features: new merchant category ($P(x_1|F) = 0.4$, $P(x_1|\neg F) = 0.05$), amount > 3 standard deviations ($P(x_2|F) = 0.3$, $P(x_2|\neg F) = 0.01$), foreign country ($P(x_3|F) = 0.5$, $P(x_3|\neg F) = 0.08$).

Prior fraud rate: $P(F) = 0.001$

$$\ln \frac{P(F)}{P(\neg F)} = \ln \frac{0.001}{0.999} = -6.907$$

$$\sum \ln \frac{P(x_i|F)}{P(x_i|\neg F)} = \ln\frac{0.4}{0.05} + \ln\frac{0.3}{0.01} + \ln\frac{0.5}{0.08} = 2.079 + 3.401 + 1.833 = 7.313$$

$$\text{Log-odds} = -6.907 + 7.313 = 0.406$$

$$P(F|\mathbf{x}) = \frac{e^{0.406}}{1 + e^{0.406}} = \frac{1.501}{2.501} = 0.600$$

60% fraud probability. With a typical threshold of $\tau = 0.5$, this transaction would be flagged for review.

## 5. Scope Reduction Economics (Optimization)
### The Problem
Organizations must balance the cost of PCI compliance against the cost of scope reduction technologies (tokenization, P2PE, cloud migration). This is a constrained optimization problem.

### The Formula
Total PCI cost as a function of scope $s$ (number of in-scope systems):

$$C_{total}(s) = C_{assess}(s) + C_{controls}(s) + C_{reduction}(S_0 - s)$$

Where $S_0$ is the original scope size, and:

$$C_{assess}(s) = \alpha \cdot s + \beta \quad (\text{assessment cost, linear in scope})$$
$$C_{controls}(s) = \gamma \cdot s \quad (\text{per-system control cost})$$
$$C_{reduction}(S_0 - s) = \delta \cdot (S_0 - s)^{0.7} \quad (\text{economies of scale in reduction})$$

Optimal scope:

$$\frac{dC_{total}}{ds} = \alpha + \gamma - 0.7\delta(S_0 - s)^{-0.3} = 0$$

$$s^* = S_0 - \left(\frac{0.7\delta}{\alpha + \gamma}\right)^{10/3}$$

### Worked Examples
**Example**: A retailer has $S_0 = 200$ in-scope systems. Parameters: $\alpha = \$2{,}000$/system (assessment), $\gamma = \$5{,}000$/system (annual controls), $\delta = \$50{,}000$ (tokenization/P2PE deployment coefficient).

$$s^* = 200 - \left(\frac{0.7 \times 50{,}000}{2{,}000 + 5{,}000}\right)^{10/3} = 200 - \left(\frac{35{,}000}{7{,}000}\right)^{10/3} = 200 - 5^{3.333}$$

$$5^{3.333} = 5^3 \times 5^{0.333} = 125 \times 1.71 = 213.7$$

Since $s^*$ would be negative, this means the cost of scope reduction per system is low enough relative to compliance cost that maximum reduction is optimal. The retailer should reduce scope as aggressively as possible, targeting near-zero CDE footprint via full tokenization.

## 6. Log Anomaly Detection (Statistical Process Control)
### The Problem
Requirement 10 mandates logging and monitoring of all access to cardholder data. Detecting anomalous access patterns among millions of log entries requires statistical baselines. Control charts from statistical process control identify deviations from normal behavior.

### The Formula
For a time series of access counts $X_1, X_2, \ldots, X_n$, establish a baseline using the moving average and standard deviation:

$$\bar{X} = \frac{1}{n} \sum_{i=1}^{n} X_i, \quad s = \sqrt{\frac{1}{n-1} \sum_{i=1}^{n} (X_i - \bar{X})^2}$$

Upper and lower control limits:

$$UCL = \bar{X} + k \cdot s, \quad LCL = \max(\bar{X} - k \cdot s, \; 0)$$

where $k$ is typically 3 (for 99.7% coverage under normality).

An observation $X_t$ is anomalous if $X_t > UCL$ or $X_t < LCL$.

For the exponentially weighted moving average (EWMA), which is more sensitive to recent shifts:

$$Z_t = \lambda X_t + (1 - \lambda) Z_{t-1}$$

$$UCL_{EWMA} = \bar{X} + k \cdot s \sqrt{\frac{\lambda}{2 - \lambda}\left(1 - (1-\lambda)^{2t}\right)}$$

### Worked Examples
**Example**: A payment application logs an average of 1,200 database queries per hour during business hours, with standard deviation 150.

$$UCL = 1{,}200 + 3 \times 150 = 1{,}650$$
$$LCL = 1{,}200 - 3 \times 150 = 750$$

At 2 AM, the system logs 2,100 queries. Since $2{,}100 > 1{,}650$, this is a statistically significant anomaly ($z = \frac{2{,}100 - 1{,}200}{150} = 6.0$, well beyond the 3-sigma threshold).

Using EWMA with $\lambda = 0.2$ after a gradual increase over 5 hours (1,200, 1,300, 1,400, 1,500, 1,600):

$$Z_5 = 0.2(1{,}600) + 0.8(Z_4) = 320 + 0.8 \times [0.2(1{,}500) + 0.8 \times Z_3] \ldots$$

Computing iteratively: $Z_1 = 1{,}200$, $Z_2 = 1{,}220$, $Z_3 = 1{,}256$, $Z_4 = 1{,}305$, $Z_5 = 1{,}364$

The EWMA detects the trend before a single-point control chart would, enabling earlier investigation.

## Prerequisites
- graph-theory, linear-algebra, information-theory, bayesian-inference, optimization, combinatorics, probability-theory, cryptographic-primitives, statistical-process-control
