# The Mathematics of SIEM -- Event Correlation and Anomaly Detection

> *A billion logs arrive daily; the analyst needs three alerts that matter -- correlation is the art of finding signal in an ocean of noise.*

---

## 1. Event Correlation Windows (Temporal Logic)

### The Problem

Correlation rules must detect sequences of events within time windows. Given event streams from multiple sources with clock skew, jitter, and variable latency, the system must correctly identify related events while minimizing both false positives and missed detections.

### The Formula

For two events A and B to be correlated within window w, with clock skew delta between sources:

$$P(\text{correlated}) = P(|t_A - t_B| \leq w + \delta) = 1 - e^{-\lambda \cdot (w + \delta)}$$

where lambda is the event arrival rate. For a sequence of n ordered events within total window W:

$$P(\text{sequence in } W) = \frac{(\lambda W)^n}{n!} \cdot e^{-\lambda W}$$

The sliding window join of two event streams S_1 (rate lambda_1) and S_2 (rate lambda_2) produces expected matches:

$$E[\text{matches}] = \lambda_1 \cdot \lambda_2 \cdot w \cdot T$$

where T is the observation period and w is the correlation window.

### Worked Examples

**Example 1: Brute force followed by successful login**

Failed login rate lambda_1 = 100/hour from a specific IP. Successful login rate lambda_2 = 2/hour. Correlation window w = 5 minutes (1/12 hour).

Expected false correlations per hour:
- E[matches] = 100 * 2 * (1/12) * 1 = 16.67 false matches per hour
- Require threshold: 10+ failures before success reduces to meaningful detections

**Example 2: Multi-stage attack detection**

Three-stage chain: port scan (lambda_1 = 50/hr), exploit attempt (lambda_2 = 5/hr), reverse shell (lambda_3 = 0.1/hr). Window w = 30 min.

- P(all three from same source in 30 min, random) = (50 * 5 * 0.1) * (0.5)^2 / 6 = 10.4 expected sequences/hr
- Adding same_source_ip constraint on /24: reduces by factor of 65,536 to 0.00016/hr

## 2. Anomaly Detection Baselines (Statistical Process Control)

### The Problem

SIEM systems must distinguish abnormal behavior from normal variation. Establishing baselines requires modeling the statistical distribution of event counts, volumes, and patterns, then flagging deviations that exceed a significance threshold.

### The Formula

Using exponentially weighted moving average (EWMA) for adaptive baselining:

$$\hat{\mu}_t = \alpha \cdot x_t + (1 - \alpha) \cdot \hat{\mu}_{t-1}$$

$$\hat{\sigma}_t^2 = \alpha \cdot (x_t - \hat{\mu}_t)^2 + (1 - \alpha) \cdot \hat{\sigma}_{t-1}^2$$

An anomaly is flagged when:

$$z_t = \frac{x_t - \hat{\mu}_t}{\hat{\sigma}_t} > k$$

where k is the sensitivity parameter (typically 3 for 99.7% confidence under normality). The false positive rate for Gaussian data:

$$P_{FP} = 2 \cdot \Phi(-k) = \text{erfc}\left(\frac{k}{\sqrt{2}}\right)$$

For non-Gaussian event distributions (common in security), the Chebyshev bound gives a distribution-free guarantee:

$$P(|X - \mu| \geq k\sigma) \leq \frac{1}{k^2}$$

### Worked Examples

**Example 1: Login volume baselining**

Historical daily login count: mu = 5000, sigma = 800. Smoothing alpha = 0.1, threshold k = 3.

Day with 8500 logins:
- z = (8500 - 5000) / 800 = 4.375 > 3.0 -- anomaly flagged
- Gaussian P_FP = 2 * Phi(-4.375) = 0.000012 (1 in 83,000 days)
- Chebyshev bound: P <= 1/4.375^2 = 0.052 (conservative)

Updated baseline after anomaly: mu_new = 0.1 * 8500 + 0.9 * 5000 = 5350

**Example 2: DNS query entropy**

Normal DNS query entropy H = 3.2 bits/char, sigma_H = 0.4. DGA domains average H = 4.8 bits/char.
- z = (4.8 - 3.2) / 0.4 = 4.0
- Detection rate: P(Z > 3 | DGA) = P(z > 3 when true mean is 4.0) = Phi(4.0 - 3.0) = 0.841

## 3. Log Ingestion Capacity Planning (Queueing and Storage)

### The Problem

SIEM infrastructure must be sized to handle peak event rates without data loss while meeting retention requirements within budget. Under-provisioning causes dropped events during incidents (precisely when visibility matters most); over-provisioning wastes resources.

### The Formula

Storage capacity for retention period R days with average event size s bytes, ingest rate lambda events/second, and compression ratio c:

$$S = \frac{\lambda \cdot s \cdot 86400 \cdot R}{c}$$

For burst capacity using an M/M/1 queue model with Poisson arrivals at rate lambda and processing rate mu:

$$P(\text{queue length} > K) = \rho^{K+1}, \quad \rho = \frac{\lambda}{\mu}$$

$$W_q = \frac{\rho}{\mu(1-\rho)}$$

The required processing headroom to keep drop probability below epsilon:

$$\mu \geq \frac{\lambda}{1 - \epsilon^{1/(K+1)}}$$

### Worked Examples

**Example 1: Storage planning**

Average ingest: lambda = 15,000 EPS. Average event size: s = 800 bytes. Retention R = 365 days. Compression ratio c = 5x.

Storage = (15,000 * 800 * 86,400 * 365) / 5 = 75.7 TB

With 20% overhead for indexing: 75.7 * 1.2 = 90.8 TB

**Example 2: Burst sizing**

Normal rate lambda = 15,000 EPS. Incident burst rate lambda_peak = 60,000 EPS. Pipeline capacity mu = 50,000 EPS. Buffer K = 100,000 events.

During burst: rho = 60,000 / 50,000 = 1.2 (overloaded).
Time to fill buffer: K / (lambda_peak - mu) = 100,000 / 10,000 = 10 seconds.
Events dropped in 5-minute burst: (60,000 - 50,000) * 290 = 2,900,000.
Required capacity for zero-drop: mu >= 60,000 * 1.1 = 66,000 EPS.

## 4. Detection Coverage Metrics (Set Theory)

### The Problem

Measuring and improving detection coverage requires mapping rules to techniques systematically. Coverage gaps must be quantified, and new rule development must be prioritized by threat relevance and feasibility.

### The Formula

Let T be the set of all MITRE ATT&CK techniques relevant to the organization, D be the set of techniques with detection rules, and L be the set of techniques with available log sources:

$$\text{Coverage} = \frac{|D \cap T|}{|T|}$$

$$\text{Feasible Coverage} = \frac{|D \cap T|}{|L \cap T|}$$

$$\text{Detection Gap} = |T \setminus D|$$

For rule effectiveness, given true positive rate p and rule count n:

$$P(\text{detect attack using } k \text{ techniques}) = 1 - \prod_{i=1}^{k}(1 - p_i)$$

The expected number of alerts per real attack using m correlated rules:

$$E[\text{alerts}] = \sum_{i=1}^{m} p_i$$

### Worked Examples

**Example 1: ATT&CK coverage assessment**

Organization threat profile: |T| = 120 techniques (mapped from threat intelligence). Current rules: |D| = 45 techniques covered. Log sources available: |L| = 95 techniques observable.

- Coverage: 45/120 = 37.5%
- Feasible coverage: 45/95 = 47.4%
- Gap: 120 - 45 = 75 techniques uncovered
- Addressable gap: 95 - 45 = 50 techniques (have logs, need rules)
- Infeasible gap: 120 - 95 = 25 techniques (need new log sources)

**Example 2: Multi-rule detection probability**

Attack uses 4 techniques. Detection probabilities per technique: p = [0.6, 0.4, 0.8, 0.3].
- P(detect at least 1) = 1 - (0.4 * 0.6 * 0.2 * 0.7) = 1 - 0.0336 = 0.966
- P(detect all 4) = 0.6 * 0.4 * 0.8 * 0.3 = 0.058
- Expected alerts per attack: 0.6 + 0.4 + 0.8 + 0.3 = 2.1

## 5. Time Synchronization Error Impact (Clock Skew Analysis)

### The Problem

Correlation across distributed log sources requires aligned timestamps. Clock skew between sources causes events to appear out of order, breaking temporal correlation rules. Quantifying the impact of synchronization error on detection accuracy is essential for tuning correlation windows.

### The Formula

If two sources have independent clock offsets $\epsilon_1 \sim N(0, \sigma_1^2)$ and $\epsilon_2 \sim N(0, \sigma_2^2)$, the perceived time difference between simultaneous events is:

$$\Delta t_{\text{perceived}} = (t + \epsilon_1) - (t + \epsilon_2) = \epsilon_1 - \epsilon_2 \sim N(0, \sigma_1^2 + \sigma_2^2)$$

For a correlation window w, the probability of missing a truly correlated pair:

$$P_{\text{miss}} = P(|\Delta t_{\text{perceived}}| > w) = 2\Phi\left(\frac{-w}{\sqrt{\sigma_1^2 + \sigma_2^2}}\right)$$

The minimum window size to capture 99% of correlated events:

$$w_{\min} = 2.576 \cdot \sqrt{\sigma_1^2 + \sigma_2^2}$$

### Worked Examples

**Example 1: NTP-synchronized sources**

Two sources with NTP sync, sigma_1 = sigma_2 = 50ms.
- Combined sigma = sqrt(2500 + 2500) = 70.7ms
- With 1-second correlation window: P_miss = 2*Phi(-1000/70.7) = 2*Phi(-14.1) approximately 0
- With 100ms window: P_miss = 2*Phi(-100/70.7) = 2*Phi(-1.41) = 0.158 (15.8% missed)

Minimum window for 99% capture: w_min = 2.576 * 70.7 = 182ms.

**Example 2: Unsynchronized legacy system**

Legacy system with sigma = 30 seconds. Modern system sigma = 50ms.
- Combined sigma = sqrt(900 + 0.0025) = 30.0s
- Minimum window for 99%: 2.576 * 30 = 77.3 seconds
- A 5-minute correlation window captures 99.99%+ but increases false correlations

## Prerequisites

- Probability theory (Poisson processes, conditional probability)
- Statistics (EWMA, z-scores, hypothesis testing, Chebyshev inequality)
- Queueing theory (M/M/1, arrival and service rates)
- Set theory (coverage metrics, intersections)
- Information theory (entropy for anomaly detection)
- Time series analysis (seasonality, trend decomposition)
- Dimensional analysis for capacity planning
- Error propagation (Gaussian error combination)
