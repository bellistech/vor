# The Mathematics of Social Engineering — Persuasion Modeling, Campaign Optimization, and Defense Quantification

> *Social engineering exploits predictable patterns in human cognition: the probability of a user clicking a phishing link follows a logistic model driven by urgency, authority, and personalization features; campaign effectiveness can be optimized using Bayesian A/B testing; organizational risk aggregates as a series system where each employee is a potential failure point; and the economics of awareness training obey diminishing returns curves that determine optimal investment levels.*

---

## 1. Phishing Click Probability (Logistic Regression)
### The Problem
The probability that a target clicks a phishing link depends on measurable features of the email and the target. A logistic model quantifies how urgency, authority cues, personalization, and target characteristics combine to predict click likelihood.

### The Formula
The probability of a click given feature vector $\mathbf{x}$:

$$P(\text{click} \mid \mathbf{x}) = \frac{1}{1 + e^{-(\beta_0 + \sum_{i=1}^{n} \beta_i x_i)}}$$

Key features and typical coefficients (from simulated campaign data):

$$\ln\frac{P(\text{click})}{1 - P(\text{click})} = \beta_0 + \beta_1 \cdot \text{urgency} + \beta_2 \cdot \text{authority} + \beta_3 \cdot \text{personalization} + \beta_4 \cdot \text{training\_recency} + \beta_5 \cdot \text{workload}$$

Where:
- $\beta_0 \approx -2.5$ (baseline log-odds)
- $\beta_1 \approx 1.2$ (urgency increases odds by $e^{1.2} \approx 3.3\times$)
- $\beta_2 \approx 0.9$ (authority cue, $e^{0.9} \approx 2.5\times$)
- $\beta_3 \approx 0.7$ (personalization, $e^{0.7} \approx 2.0\times$)
- $\beta_4 \approx -0.8$ (recent training decreases odds)
- $\beta_5 \approx 0.5$ (high workload increases odds)

### Worked Examples
**Example**: Spear phishing email targeting a finance employee. Features: high urgency ($x_1 = 1$), CEO impersonation ($x_2 = 1$), uses target's name and project ($x_3 = 1$), training was 8 months ago ($x_4 = 0$, coded as not recent), moderate workload ($x_5 = 0.5$).

$$\text{logit} = -2.5 + 1.2(1) + 0.9(1) + 0.7(1) + (-0.8)(0) + 0.5(0.5) = 0.55$$

$$P(\text{click}) = \frac{1}{1 + e^{-0.55}} = \frac{1}{1 + 0.577} = 0.634$$

63.4% click probability. If training had been recent ($x_4 = 1$):

$$\text{logit} = 0.55 + (-0.8)(1) = -0.25$$

$$P(\text{click}) = \frac{1}{1 + e^{0.25}} = \frac{1}{1 + 1.284} = 0.438$$

Training reduces click probability from 63.4% to 43.8% — a 19.6 percentage point reduction.

## 2. Organizational Risk Aggregation (Series System Reliability)
### The Problem
In a social engineering attack, each employee is a potential entry point. The organization is compromised if any single employee falls for the attack. This is a series system where one failure causes system failure.

### The Formula
If each employee $i$ has independent probability $p_i$ of falling for a social engineering attack, the organizational survival probability is:

$$P(\text{no breach}) = \prod_{i=1}^{N} (1 - p_i)$$

For homogeneous employees with click probability $p$:

$$P(\text{no breach}) = (1 - p)^N$$

$$P(\text{at least one breach}) = 1 - (1 - p)^N$$

The expected number of compromised employees:

$$E[\text{compromised}] = N \cdot p$$

The probability that exactly $k$ employees are compromised follows the binomial distribution:

$$P(X = k) = \binom{N}{k} p^k (1-p)^{N-k}$$

### Worked Examples
**Example**: A company has 500 employees. After training, the average click rate is 5% ($p = 0.05$).

$$P(\text{at least one click}) = 1 - (1 - 0.05)^{500} = 1 - 0.95^{500}$$

$$= 1 - e^{500 \ln(0.95)} = 1 - e^{-25.65} \approx 1 - 7.2 \times 10^{-12} \approx 1.0$$

Even at 5% per-person click rate, organizational breach is essentially certain. Expected compromised: $500 \times 0.05 = 25$ employees.

To achieve 50% organizational survival against a targeted campaign:

$$(1 - p)^{500} = 0.5 \implies p = 1 - 0.5^{1/500} = 1 - 0.99861 = 0.00139$$

Each employee must have only a 0.14% click rate — demonstrating why defense-in-depth (technical controls + training + process) is essential; training alone cannot reduce click rates enough.

## 3. Awareness Training ROI (Diminishing Returns Model)
### The Problem
Security awareness training reduces click rates but with diminishing returns. Each additional hour of training produces less incremental benefit. The optimal investment level balances training cost against breach risk reduction.

### The Formula
Click rate as a function of training investment $t$ (hours per year):

$$p(t) = p_0 \cdot e^{-\lambda t} + p_{min}$$

where $p_0$ is the untrained click rate, $\lambda$ is the learning decay rate, and $p_{min}$ is the irreducible minimum (social engineering will always fool some people).

Expected annual breach cost:

$$C_{breach}(t) = L \cdot [1 - (1 - p(t))^N]$$

where $L$ is the expected loss from a successful breach and $N$ is the number of employees.

Total cost (training + residual risk):

$$C_{total}(t) = c \cdot N \cdot t + C_{breach}(t)$$

Optimal training hours:

$$\frac{dC_{total}}{dt} = c \cdot N - L \cdot N \cdot \lambda \cdot p_0 \cdot e^{-\lambda t} \cdot (1 - p(t))^{N-1} = 0$$

### Worked Examples
**Example**: 1,000 employees, untrained click rate $p_0 = 0.25$, $p_{min} = 0.02$, $\lambda = 0.3$/hour, training cost $c = \$50$/hour/employee, breach loss $L = \$2{,}000{,}000$.

After 4 hours of annual training:

$$p(4) = 0.25 \cdot e^{-0.3 \times 4} + 0.02 = 0.25 \times 0.301 + 0.02 = 0.095$$

Training cost: $50 \times 1{,}000 \times 4 = \$200{,}000$

After 8 hours:

$$p(8) = 0.25 \cdot e^{-2.4} + 0.02 = 0.25 \times 0.091 + 0.02 = 0.043$$

Training cost: $50 \times 1{,}000 \times 8 = \$400{,}000$

After 12 hours:

$$p(12) = 0.25 \cdot e^{-3.6} + 0.02 = 0.25 \times 0.027 + 0.02 = 0.027$$

Diminishing returns: hours 0-4 reduced click rate by 15.5 pp, hours 4-8 by 5.2 pp, hours 8-12 by only 1.6 pp. The marginal benefit of training beyond 8 hours is minimal.

## 4. Campaign A/B Optimization (Bayesian Testing)
### The Problem
Phishing simulation programs must determine which email templates, timing, and pretexts are most effective at testing (and training) employees. Bayesian A/B testing compares campaign variants while accounting for uncertainty in small sample sizes.

### The Formula
Model the click rate of variant $j$ as a Beta distribution:

$$\theta_j \sim \text{Beta}(\alpha_j + s_j, \; \beta_j + n_j - s_j)$$

where $s_j$ is the number of clicks, $n_j$ is the number of recipients, and $\alpha_j, \beta_j$ are prior parameters.

The probability that variant A has a higher click rate than variant B:

$$P(\theta_A > \theta_B) = \int_0^1 \int_0^{\theta_A} f(\theta_A) f(\theta_B) \, d\theta_B \, d\theta_A$$

For decision-making, use the expected loss:

$$E[\text{loss} \mid \text{choose A}] = E[\max(\theta_B - \theta_A, 0)]$$

### Worked Examples
**Example**: Two phishing templates tested on 100 employees each.

- Template A (urgency-based): 22 clicks out of 100
- Template B (authority-based): 18 clicks out of 100

Using uniform priors $\text{Beta}(1, 1)$:

$$\theta_A \sim \text{Beta}(23, 79), \quad \mu_A = \frac{23}{102} = 0.225$$
$$\theta_B \sim \text{Beta}(19, 83), \quad \mu_B = \frac{19}{102} = 0.186$$

Standard deviations:

$$\sigma_A = \sqrt{\frac{23 \times 79}{102^2 \times 103}} = 0.041, \quad \sigma_B = \sqrt{\frac{19 \times 83}{102^2 \times 103}} = 0.038$$

Approximate $P(\theta_A > \theta_B)$ using normal approximation:

$$Z = \frac{\mu_A - \mu_B}{\sqrt{\sigma_A^2 + \sigma_B^2}} = \frac{0.225 - 0.186}{\sqrt{0.041^2 + 0.038^2}} = \frac{0.039}{0.056} = 0.696$$

$$P(\theta_A > \theta_B) \approx \Phi(0.696) = 0.757$$

75.7% probability that Template A (urgency) produces a higher click rate than Template B (authority). Not yet conclusive — need more data or accept the uncertainty for training template selection.

## 5. Pretext Effectiveness Decay (Temporal Model)
### The Problem
Social engineering pretexts lose effectiveness as awareness spreads through organizations. After a publicized incident or training session, employees become temporarily resistant to specific attack patterns. This decay can be modeled as a time-dependent process.

### The Formula
Pretext effectiveness as a function of time since last exposure/training:

$$E(t) = E_{base} + (E_{peak} - E_{base})(1 - e^{-\gamma t})$$

where $E_{base}$ is the effectiveness immediately after training, $E_{peak}$ is the fully-decayed (untrained) effectiveness, and $\gamma$ is the awareness decay rate.

For an organization with periodic training at interval $T$, the average effectiveness over a cycle:

$$\bar{E} = \frac{1}{T} \int_0^T E(t) \, dt = E_{base} + (E_{peak} - E_{base})\left(1 - \frac{1}{\gamma T}(1 - e^{-\gamma T})\right)$$

### Worked Examples
**Example**: After a phishing awareness session, click rate drops from $E_{peak} = 0.25$ to $E_{base} = 0.05$. Awareness decay rate $\gamma = 0.15$ per month.

At month 3 after training:

$$E(3) = 0.05 + (0.25 - 0.05)(1 - e^{-0.15 \times 3}) = 0.05 + 0.20(1 - 0.638) = 0.05 + 0.072 = 0.122$$

At month 6:

$$E(6) = 0.05 + 0.20(1 - e^{-0.9}) = 0.05 + 0.20(1 - 0.407) = 0.05 + 0.119 = 0.169$$

With quarterly training ($T = 3$ months), average click rate:

$$\bar{E} = 0.05 + 0.20\left(1 - \frac{1}{0.45}(1 - e^{-0.45})\right) = 0.05 + 0.20\left(1 - \frac{0.362}{0.45}\right) = 0.05 + 0.20 \times 0.196 = 0.089$$

With monthly training ($T = 1$):

$$\bar{E} = 0.05 + 0.20\left(1 - \frac{1}{0.15}(1 - e^{-0.15})\right) = 0.05 + 0.20\left(1 - \frac{0.139}{0.15}\right) = 0.05 + 0.20 \times 0.073 = 0.065$$

Monthly training achieves 6.5% vs quarterly at 8.9%. The 2.4pp improvement costs 4x the training investment — whether it is justified depends on breach cost modeling from Section 3.

## Prerequisites
- logistic-regression, probability-theory, bayesian-inference, reliability-theory, optimization, exponential-decay, series-systems, combinatorics
