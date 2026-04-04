# The Mathematics of Argo Rollouts — Progressive Delivery as Statistical Hypothesis Testing

> *Every canary is an experiment. Every promotion is a verdict. The mathematics decides if the new version lives or dies.*

---

## 1. Canary Weight Optimization (Risk-Bounded Exposure)

### The Problem

A canary rollout progressively shifts traffic from a stable version to a new version. Each weight step exposes more users to potential regressions. How do we choose the weight schedule to minimize blast radius while maximizing detection speed?

### The Formula

Define the cumulative user-impact as the integral of traffic weight over time. For a step schedule $\{(w_i, d_i)\}$ where $w_i$ is the weight and $d_i$ is the pause duration at step $i$:

$$\text{Impact} = \sum_{i=1}^{k} w_i \cdot d_i$$

The total rollout time:

$$T_{\text{rollout}} = \sum_{i=1}^{k} d_i$$

To minimize impact for a fixed total time $T$, the optimal strategy front-loads small weights with longer pauses:

$$\min \sum_{i=1}^{k} w_i \cdot d_i \quad \text{subject to} \quad \sum d_i = T, \quad w_k = 1$$

The blast radius at any point is bounded by the current weight:

$$\text{BlastRadius}(t) = w(t) \times N$$

where $N$ is the total user population.

### Worked Examples

**Example 1**: Compare two schedules for a 30-minute rollout to 1000 users:

Schedule A (linear): 25% for 10m, 50% for 10m, 100% for 10m

$$\text{Impact}_A = 0.25 \times 10 + 0.50 \times 10 + 1.0 \times 10 = 17.5 \text{ user-minutes per user}$$

Schedule B (conservative): 5% for 15m, 25% for 5m, 50% for 5m, 100% for 5m

$$\text{Impact}_B = 0.05 \times 15 + 0.25 \times 5 + 0.50 \times 5 + 1.0 \times 5 = 9.5 \text{ user-minutes per user}$$

Schedule B has 46% less user-impact with the same total time.

**Example 2**: Maximum affected users at any point:

Schedule A peak: $1000 \times 0.50 = 500$ users at risk during the second step.

Schedule B peak: $1000 \times 0.05 = 50$ users at risk during the longest step (15m), giving the most observation time at lowest exposure.

---

## 2. Analysis Template Design (Sequential Hypothesis Testing)

### The Problem

An AnalysisTemplate evaluates metrics to decide whether to promote or abort. This is a sequential hypothesis test: we observe metrics sample-by-sample and decide when we have enough evidence. How many samples do we need?

### The Formula

Define the null hypothesis $H_0$: "the canary is healthy" (success rate $\geq p_0$) and alternative $H_1$: "the canary is degraded" (success rate $\leq p_1$). Using the Sequential Probability Ratio Test (SPRT):

$$\Lambda_n = \prod_{i=1}^{n} \frac{P(x_i \mid H_1)}{P(x_i \mid H_0)}$$

Accept $H_0$ (promote) when $\Lambda_n \leq B$, reject $H_0$ (abort) when $\Lambda_n \geq A$:

$$A = \frac{1 - \beta}{\alpha}, \quad B = \frac{\beta}{1 - \alpha}$$

where $\alpha$ is the false abort rate and $\beta$ is the false promotion rate. The expected number of samples under $H_0$:

$$E[n \mid H_0] \approx \frac{(1 - \alpha) \ln B + \alpha \ln A}{p_0 \ln \frac{p_1}{p_0} + (1 - p_0) \ln \frac{1-p_1}{1-p_0}}$$

### Worked Examples

**Example 1**: Success rate threshold $p_0 = 0.99$, degraded threshold $p_1 = 0.95$, $\alpha = 0.05$, $\beta = 0.05$:

$$A = \frac{0.95}{0.05} = 19, \quad B = \frac{0.05}{0.95} = 0.0526$$

Expected samples under healthy conditions:

$$E[n \mid H_0] \approx \frac{0.95 \ln 0.0526 + 0.05 \ln 19}{0.99 \ln(0.95/0.99) + 0.01 \ln(0.05/0.01)}$$

$$= \frac{0.95(-2.945) + 0.05(2.944)}{0.99(-0.0412) + 0.01(1.609)} = \frac{-2.650}{-0.0248} \approx 107 \text{ samples}$$

At 1 sample per minute, the analysis runs about 107 minutes.

**Example 2**: To reduce analysis time, relax thresholds. With $p_1 = 0.90$ (detect only severe degradations):

$$E[n \mid H_0] \approx \frac{-2.650}{0.99 \ln(0.90/0.99) + 0.01 \ln(0.10/0.01)} = \frac{-2.650}{-0.0949 + 0.0230} = \frac{-2.650}{-0.0719} \approx 37 \text{ samples}$$

3x faster, but only detects drops below 90%.

---

## 3. Blue-Green Cutover (Availability During Transition)

### The Problem

Blue-green deployment maintains two full environments. During the switch, there is a brief window where DNS or service selector updates propagate. What is the availability during cutover?

### The Formula

Let $t_s$ be the switch initiation time and $\Delta$ be the propagation delay. During $[t_s, t_s + \Delta]$, a fraction $f(t)$ of requests reach the new (green) version:

$$f(t) = \frac{t - t_s}{\Delta} \quad \text{for } t \in [t_s, t_s + \Delta]$$

If the green version has an error, the availability during cutover:

$$A_{\text{cutover}} = 1 - \int_{t_s}^{t_s + \Delta} f(t) \cdot p_{\text{error}} \, \frac{dt}{\Delta} = 1 - \frac{p_{\text{error}}}{2}$$

The `scaleDownDelaySeconds` parameter $D$ ensures the old (blue) version remains available for in-flight requests:

$$P(\text{request dropped}) = P(\text{duration} > D) = e^{-D/\mu_{\text{req}}}$$

where $\mu_{\text{req}}$ is the mean request duration.

### Worked Examples

**Example 1**: Green version has a 5% error rate discovered after cutover. Propagation takes 10 seconds:

$$A_{\text{cutover}} = 1 - \frac{0.05}{2} = 0.975$$

97.5% availability during the 10-second window. Total failed requests at 1000 req/s:

$$\text{Failed} = 1000 \times 10 \times 0.05 / 2 = 250 \text{ requests}$$

**Example 2**: `scaleDownDelaySeconds: 300` with mean request duration of 30 seconds:

$$P(\text{drop}) = e^{-300/30} = e^{-10} \approx 0.0000454$$

Virtually no dropped in-flight requests. Reducing to 60 seconds:

$$P(\text{drop}) = e^{-60/30} = e^{-2} \approx 0.135$$

13.5% of long-running requests would be terminated.

---

## 4. Experiment Design (A/B Testing Power Analysis)

### The Problem

Argo Rollouts Experiments run baseline and canary variants side by side. How long must the experiment run to detect a meaningful difference with statistical confidence?

### The Formula

For a two-sample proportion test comparing success rates $p_1$ (baseline) and $p_2$ (canary), the required sample size per group:

$$n = \left(\frac{z_{\alpha/2}\sqrt{2\bar{p}(1-\bar{p})} + z_\beta\sqrt{p_1(1-p_1) + p_2(1-p_2)}}{p_1 - p_2}\right)^2$$

where $\bar{p} = (p_1 + p_2)/2$, $z_{\alpha/2}$ is the critical value for significance level $\alpha$, and $z_\beta$ is the critical value for power $1 - \beta$.

### Worked Examples

**Example 1**: Detect a drop from 99% to 97% success rate with 95% confidence and 80% power:

$$\bar{p} = 0.98, \quad z_{0.025} = 1.96, \quad z_{0.2} = 0.842$$

$$n = \left(\frac{1.96\sqrt{2 \times 0.98 \times 0.02} + 0.842\sqrt{0.99 \times 0.01 + 0.97 \times 0.03}}{0.02}\right)^2$$

$$= \left(\frac{1.96 \times 0.198 + 0.842 \times 0.200}{0.02}\right)^2 = \left(\frac{0.388 + 0.168}{0.02}\right)^2 = (27.8)^2 \approx 773$$

Each group needs 773 requests. At 100 req/s per group, the experiment needs about 8 seconds of data.

**Example 2**: Detecting a smaller difference (99% vs 98.5%) requires:

$$n = \left(\frac{1.96\sqrt{2 \times 0.9875 \times 0.0125} + 0.842\sqrt{0.0099 + 0.01478}}{0.005}\right)^2$$

$$\approx \left(\frac{0.308 + 0.132}{0.005}\right)^2 = (88.0)^2 \approx 7744$$

10x more samples needed for a difference half as large.

---

## 5. Rollback Speed (Mean Time to Recovery)

### The Problem

When a canary fails, the rollout must abort and revert traffic. The mean time to recovery (MTTR) determines how long users experience the degraded version.

### The Formula

MTTR consists of detection time $T_d$, decision time $T_\delta$, and rollback execution time $T_r$:

$$\text{MTTR} = T_d + T_\delta + T_r$$

Detection time depends on the analysis interval $I$ and the number of samples $n$ needed to trigger abort:

$$T_d = n \times I$$

The total user-impact during the incident:

$$\text{UserImpact} = w \times N \times p_{\text{error}} \times \text{MTTR}$$

### Worked Examples

**Example 1**: Canary at 10% weight, analysis interval 60s, `failureLimit: 2` (abort after 2 consecutive failures), rollback takes 30s:

$$T_d = 2 \times 60 = 120\text{s}, \quad T_\delta = 0 \text{ (automated)}, \quad T_r = 30\text{s}$$

$$\text{MTTR} = 150\text{s}$$

At 1000 total users with 50% canary error rate:

$$\text{UserImpact} = 0.10 \times 1000 \times 0.50 \times 150 = 7500 \text{ error-seconds}$$

**Example 2**: Same scenario at 50% weight:

$$\text{UserImpact} = 0.50 \times 1000 \times 0.50 \times 150 = 37500 \text{ error-seconds}$$

5x worse. This is why early canary steps should be small and analysis should run before major weight increases.

---

## Prerequisites

- Statistical hypothesis testing (null/alternative hypotheses, Type I/II errors, p-values)
- Sequential analysis (SPRT, Wald's sequential test)
- Queueing theory (request durations, draining, in-flight requests)
- Kubernetes ReplicaSet scaling and Service selector mechanics
- Traffic management concepts (weighted routing, header-based routing, virtual services)
- Prometheus query language (PromQL) for metric-based analysis
- Exponential distribution (modeling request lifetimes and failure detection)
