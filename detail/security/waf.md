# The Mathematics of WAF -- Anomaly Scoring and Detection Theory

> *Every HTTP request carries a score; the firewall must draw a line between the suspicious and the hostile without silencing legitimate users.*

---

## 1. Anomaly Scoring Model (Cumulative Risk)

### The Problem

A single malicious indicator in a request may be ambiguous (a parameter containing "select" could be legitimate). The anomaly scoring model assigns partial scores to each suspicious element and blocks only when the cumulative score crosses a threshold. The challenge is selecting a threshold that maximizes true positives while minimizing false positives.

### The Formula

Let a request R be inspected by n rules, each producing a binary match indicator $m_i \in \{0, 1\}$ and an assigned weight $w_i$. The anomaly score is:

$$S(R) = \sum_{i=1}^{n} m_i \cdot w_i$$

The request is blocked when $S(R) \geq \tau$ (threshold). The CRS weight assignments follow a severity mapping:

$$w_i = \begin{cases} 5 & \text{CRITICAL (SQLi, RCE)} \\ 4 & \text{ERROR (XSS, LFI)} \\ 3 & \text{WARNING (scanner)} \\ 2 & \text{NOTICE (protocol anomaly)} \end{cases}$$

For independent rule matches with individual false-match probabilities $p_i$, the expected false anomaly score on legitimate traffic is:

$$E[S_{\text{legit}}] = \sum_{i=1}^{n} p_i \cdot w_i$$

$$\text{Var}(S_{\text{legit}}) = \sum_{i=1}^{n} p_i(1-p_i) \cdot w_i^2$$

The false positive rate for threshold tau (assuming Gaussian approximation for large n by CLT):

$$P_{FP}(\tau) \approx \Phi\left(\frac{-(\ \tau - E[S_{\text{legit}}])}{\sqrt{\text{Var}(S_{\text{legit}})}}\right)$$

### Worked Examples

**Example 1: CRS with Paranoia Level 1**

Active rules n = 150. Average false-match probability p_i = 0.002. Average weight w_i = 3.5. Threshold tau = 5.

- E[S_legit] = 150 * 0.002 * 3.5 = 1.05
- Var(S_legit) = 150 * 0.002 * 0.998 * 3.5^2 = 3.66
- sigma = 1.91
- z = (5 - 1.05) / 1.91 = 2.07
- P_FP = Phi(-2.07) = 0.019 (1.9% of legitimate requests blocked)

Raising threshold to tau = 10:
- z = (10 - 1.05) / 1.91 = 4.69
- P_FP = Phi(-4.69) = 0.0000014 (1 in 714,000 requests)

**Example 2: Paranoia Level 3 impact**

Increasing paranoia activates 400 additional rules with higher p_i = 0.008.
- New E[S_legit] = 1.05 + 400 * 0.008 * 3.0 = 10.65
- With tau = 5: almost all traffic blocked. Threshold must rise to tau = 15+

## 2. Rule Evasion Probability (Encoding and Normalization)

### The Problem

Attackers use encoding transformations (URL encoding, Unicode normalization, case mixing, comment insertion) to evade pattern matching. The WAF must normalize inputs before matching, but each normalization step has computational cost and potential for recursive evasion.

### The Formula

Let the set of encoding transformations be $\mathcal{T} = \{t_1, t_2, \ldots, t_k\}$. An attack payload a can be encoded as $t_i(a)$ or composed as $t_i(t_j(a))$. If the WAF applies normalization set $\mathcal{N} \subseteq \mathcal{T}$, the evasion probability is:

$$P_{\text{evasion}} = 1 - \frac{|\{t \in \mathcal{T}^* : \exists n \in \mathcal{N}^*, n(t(a)) = a\}|}{|\mathcal{T}^*|}$$

where $\mathcal{T}^*$ denotes all compositions up to depth d. For d layers of encoding with k independent bypass techniques:

$$P_{\text{evasion}}(d) = \prod_{j=1}^{d} \left(1 - \frac{|\mathcal{N}_j|}{|\mathcal{T}_j|}\right)$$

The number of unique evasion variants grows exponentially:

$$|\text{variants}| = \sum_{i=1}^{d} k^i = \frac{k^{d+1} - k}{k - 1}$$

### Worked Examples

**Example 1: SQL injection evasion**

Evasion techniques for "UNION SELECT": case mixing (2^11 variants), comment insertion (unlimited), URL encoding (single/double/triple), Unicode normalization (fullwidth chars).

WAF normalizations: case folding, single URL decode, comment stripping.
- Case: covered (0% evasion)
- URL encoding: single decode catches %55NION but misses %2555NION (double-encoded)
- Comments: covered for /* */ but not MySQL-specific /*!50000UNION*/
- Unicode: not covered (fullwidth U+FF35 for 'U' bypasses)

P_evasion approx = 0 * 0.3 * 0.1 * 1.0 = depends on composition. With Unicode bypass: P_evasion > 0 for any threshold.

**Example 2: Recursive decoding depth**

Double URL encoding: %2527 -> %27 -> ' (single quote).
If WAF decodes once: sees %27, may or may not match.
If WAF decodes twice: sees ' and matches.
If WAF decodes 3 times: %252527 -> %2527 -> %27 -> ' requires 3 passes.

Processing cost per request with d decode passes: O(d * |R|) where |R| is request size.

## 3. Rate Limiting and Token Bucket (Traffic Shaping)

### The Problem

Rate limiting must throttle abusive clients while allowing legitimate burst traffic. The token bucket algorithm provides a flexible mechanism that permits short bursts while enforcing a long-term average rate.

### The Formula

A token bucket with capacity b (burst size) and fill rate r (tokens per second). A request consuming 1 token is admitted if the bucket has tokens available:

$$\text{tokens}(t) = \min\left(b, \text{tokens}(t_{\text{prev}}) + r \cdot (t - t_{\text{prev}})\right)$$

The maximum burst duration at rate lambda > r:

$$T_{\text{burst}} = \frac{b}{\lambda - r}$$

The long-term sustained rate is exactly r, regardless of burst patterns. The probability of a legitimate user being rate-limited, given request inter-arrival times following an exponential distribution with mean 1/lambda_user:

$$P_{\text{limited}} = P\left(\sum_{i=1}^{b+1} X_i < \frac{b}{r}\right)$$

where $X_i \sim \text{Exp}(\lambda_{\text{user}})$. Using the Erlang distribution:

$$P_{\text{limited}} = 1 - e^{-\lambda_{\text{user}} \cdot b/r} \sum_{k=0}^{b} \frac{(\lambda_{\text{user}} \cdot b/r)^k}{k!}$$

### Worked Examples

**Example 1: API rate limiting**

Bucket capacity b = 50, fill rate r = 10 req/s. Legitimate user sends 5 req/s on average.

- Burst tolerance: 50 requests instantaneously, then sustained 10/s
- Legitimate user P_limited: lambda_user = 5, b/r = 5 seconds
- P(51 requests in 5 seconds from Poisson(25)) = P(X >= 51 | lambda=25) approximately 0.0000002
- Virtually no false limiting for this user

Attacker at 100 req/s:
- Burst depleted in: 50 / (100-10) = 0.56 seconds
- After burst: 90% of requests blocked (only 10/100 admitted)

**Example 2: Distributed attack (multiple IPs)**

500 IPs each at 5 req/s = 2,500 req/s aggregate. Per-IP limit r = 10/s: no blocking triggered.
Need aggregate rate limiting: total bucket b = 500, r = 200/s.
P_limited for legitimate users under aggregate limit depends on total legitimate traffic baseline.

## 4. Virtual Patching Effectiveness (Time-to-Protect)

### The Problem

When a CVE is disclosed, the time between disclosure and application patch deployment (T_patch) creates a vulnerability window. Virtual patching via WAF rules closes this window with time T_waf << T_patch. Quantifying the risk reduction requires modeling attacker behavior during the exposure window.

### The Formula

Expected number of exploitation attempts during unprotected window:

$$E[\text{attacks}] = \lambda_a \cdot T_{\text{patch}}$$

With virtual patch deployed at time T_waf:

$$E[\text{attacks}]_{\text{protected}} = \lambda_a \cdot T_{\text{waf}} + \lambda_a \cdot (T_{\text{patch}} - T_{\text{waf}}) \cdot P_{\text{evasion}}$$

Risk reduction factor:

$$R = 1 - \frac{T_{\text{waf}} + (T_{\text{patch}} - T_{\text{waf}}) \cdot P_{\text{evasion}}}{T_{\text{patch}}}$$

For typical values (T_waf = 4 hours, T_patch = 14 days, P_evasion = 0.05):

$$R = 1 - \frac{4/24 + (14 - 4/24) \cdot 0.05}{14} = 1 - \frac{0.167 + 0.692}{14} = 0.939$$

### Worked Examples

**Example 1: Critical RCE (Log4Shell-class)**

Attack rate post-disclosure: lambda_a = 500/day. T_patch = 7 days. T_waf = 2 hours.
- Without WAF: E[attacks] = 500 * 7 = 3,500 attempts
- With WAF (P_evasion = 0.03): E = 500 * (2/24) + 500 * 6.917 * 0.03 = 41.7 + 103.8 = 145.5
- Risk reduction: R = 1 - 145.5/3500 = 95.8%

**Example 2: Low-severity information disclosure**

Attack rate: lambda_a = 10/day. T_patch = 30 days. T_waf = 24 hours.
- Without WAF: 300 attempts
- With WAF (P_evasion = 0.10): 10 * 1 + 10 * 29 * 0.10 = 10 + 29 = 39
- Risk reduction: R = 87%

## 5. Bot Detection Accuracy (Classification Theory)

### The Problem

Distinguishing automated bot traffic from legitimate human users requires classification based on behavioral signals (request timing, mouse movements, JavaScript execution, TLS fingerprints). The WAF must balance detection accuracy with user experience, as false positives (blocking real users) directly impact revenue.

### The Formula

For a binary classifier (bot vs. human) with prevalence of bots $\pi$, sensitivity (TPR) $s$, and specificity $q$, the positive predictive value (precision) is:

$$\text{PPV} = \frac{s \cdot \pi}{s \cdot \pi + (1-q)(1-\pi)}$$

The expected daily cost of the classifier with N daily requests, false positive cost $C_{FP}$ (lost revenue per blocked user), and false negative cost $C_{FN}$ (damage per unblocked bot):

$$\text{Cost} = N \cdot [(1-\pi)(1-q) \cdot C_{FP} + \pi(1-s) \cdot C_{FN}]$$

The optimal operating point minimizes total cost:

$$\frac{s}{1-q} = \frac{(1-\pi) \cdot C_{FP}}{\pi \cdot C_{FN}}$$

### Worked Examples

**Example 1: E-commerce bot blocking**

Traffic: 1M requests/day, bot prevalence pi = 15%. Classifier: sensitivity s = 0.95, specificity q = 0.99. C_FP = $0.50 (lost sale), C_FN = $0.02 (scraping cost).

PPV = (0.95 * 0.15) / (0.95 * 0.15 + 0.01 * 0.85) = 0.1425 / (0.1425 + 0.0085) = 0.944.

Daily cost = 1M * [0.85 * 0.01 * 0.50 + 0.15 * 0.05 * 0.02] = 1M * [0.00425 + 0.00015] = $4,400.
Without classifier: bot damage = 1M * 0.15 * 0.02 = $3,000. Classifier FP cost exceeds bot damage.
Need specificity q > 0.994 to break even.

## Prerequisites

- Probability theory (Bernoulli trials, Gaussian approximation, CLT)
- Statistics (hypothesis testing, false positive/negative rates)
- Queueing theory (token bucket, arrival processes)
- Combinatorics (encoding variant enumeration)
- Regular expressions and formal language theory
- HTTP protocol mechanics (methods, headers, encoding schemes)
- Set theory (transformation groups, normalization coverage)
- Classification theory (precision, recall, ROC, cost-sensitive optimization)
