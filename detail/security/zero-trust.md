# The Mathematics of Zero Trust -- Continuous Verification and Risk-Adaptive Access Control

> *Zero trust architecture replaces binary perimeter trust with continuous risk scoring that combines identity confidence, device posture, behavioral signals, and environmental context into a real-time access decision function, modeled as a Bayesian inference problem with time-decaying trust and multi-factor evidence fusion.*

---

## 1. Trust Score Function (Multi-Factor Composition)

### The Problem

Zero trust requires computing a composite trust score from multiple independent signals: identity verification strength, device compliance, network context, behavioral analysis, and threat intelligence. The trust score must be computed per-request and compared against a policy threshold.

### The Formula

Trust score as weighted sum of normalized factors:

$$T = \sum_{i=1}^{n} w_i \cdot f_i(x_i) \quad \text{where} \quad \sum_{i=1}^{n} w_i = 1, \quad f_i : X_i \to [0, 1]$$

Factor functions:

$$f_{\text{identity}}(x) = \begin{cases} 1.0 & \text{FIDO2/WebAuthn} \\ 0.8 & \text{TOTP MFA} \\ 0.5 & \text{password only} \\ 0.0 & \text{unauthenticated} \end{cases}$$

$$f_{\text{device}}(x) = \frac{\text{compliant\_checks}(x)}{\text{total\_checks}}$$

Access decision:

$$\text{decision}(T) = \begin{cases} \text{allow} & T \geq \theta_{\text{allow}} \\ \text{step-up} & \theta_{\text{step-up}} \leq T < \theta_{\text{allow}} \\ \text{deny} & T < \theta_{\text{step-up}} \end{cases}$$

### Worked Examples

Weights: $w_{\text{identity}} = 0.35$, $w_{\text{device}} = 0.25$, $w_{\text{behavior}} = 0.20$, $w_{\text{network}} = 0.10$, $w_{\text{threat}} = 0.10$.

User with TOTP MFA, 8/10 device checks, normal behavior, corporate VPN, no active threats:

$$T = 0.35(0.8) + 0.25(0.8) + 0.20(0.9) + 0.10(1.0) + 0.10(1.0)$$

$$T = 0.28 + 0.20 + 0.18 + 0.10 + 0.10 = 0.86$$

Thresholds: $\theta_{\text{allow}} = 0.75$, $\theta_{\text{step-up}} = 0.50$.

$$T = 0.86 \geq 0.75 \implies \text{allow}$$

Same user, password only, unmanaged device, unusual location:

$$T = 0.35(0.5) + 0.25(0.3) + 0.20(0.9) + 0.10(0.2) + 0.10(0.7) = 0.175 + 0.075 + 0.18 + 0.02 + 0.07 = 0.52$$

$$0.50 \leq 0.52 < 0.75 \implies \text{step-up authentication required}$$

---

## 2. Time-Decaying Trust (Exponential Decay)

### The Problem

Authentication is not permanent. Trust decays over time since the last verification event. Zero trust systems must model this decay to determine when re-authentication or step-up is needed.

### The Formula

Trust decay from authentication event at time $t_0$:

$$T(t) = T_0 \cdot e^{-\lambda(t - t_0)}$$

where $\lambda$ is the decay rate and $T_0$ is the initial trust score post-authentication.

Half-life of trust:

$$t_{1/2} = \frac{\ln 2}{\lambda}$$

Time until trust drops below threshold $\theta$:

$$t_{\text{reauth}} = t_0 + \frac{1}{\lambda} \ln\left(\frac{T_0}{\theta}\right)$$

### Worked Examples

User authenticates with MFA: $T_0 = 0.95$, decay rate $\lambda = 0.0005$ per minute.

$$t_{1/2} = \frac{0.693}{0.0005} = 1{,}386 \text{ minutes} \approx 23 \text{ hours}$$

Time until trust drops below allow threshold $\theta = 0.75$:

$$t_{\text{reauth}} = \frac{1}{0.0005} \ln\left(\frac{0.95}{0.75}\right) = 2{,}000 \times \ln(1.267) = 2{,}000 \times 0.2364 = 472.8 \text{ minutes} \approx 7.9 \text{ hours}$$

Time until trust drops below step-up threshold $\theta = 0.50$:

$$t_{\text{step-up}} = \frac{1}{0.0005} \ln\left(\frac{0.95}{0.50}\right) = 2{,}000 \times 0.6419 = 1{,}283.8 \text{ minutes} \approx 21.4 \text{ hours}$$

For sensitive resources ($\theta_{\text{allow}} = 0.90$):

$$t_{\text{reauth}} = 2{,}000 \times \ln\left(\frac{0.95}{0.90}\right) = 2{,}000 \times 0.0541 = 108.2 \text{ minutes} \approx 1.8 \text{ hours}$$

---

## 3. Microsegmentation Coverage (Graph Partitioning)

### The Problem

Microsegmentation divides the network into isolated segments with explicit allow rules. The security improvement depends on the granularity of segmentation: more segments mean smaller blast radius but more policy complexity.

### The Formula

Network with $n$ workloads. Without segmentation: any compromised workload can reach all $n-1$ others.

$$\text{blast\_radius}_{\text{flat}} = n - 1$$

With $k$ segments of size $s_i$:

$$\text{blast\_radius}_{\text{segmented}} = \max_i(s_i) - 1$$

If segments are equal: $s_i = \frac{n}{k}$, blast radius $= \frac{n}{k} - 1$.

Policy complexity (number of inter-segment rules):

$$|R| = \sum_{i \neq j} r_{ij} \leq k(k-1)$$

Risk reduction factor:

$$\text{RRF} = \frac{\text{blast\_radius}_{\text{flat}}}{\text{blast\_radius}_{\text{segmented}}} = \frac{n - 1}{\frac{n}{k} - 1}$$

### Worked Examples

Data center with $n = 500$ workloads.

Flat network: blast radius = 499 workloads.

Microsegmented into $k = 50$ segments of 10 workloads each:

$$\text{blast\_radius} = 10 - 1 = 9$$

$$\text{RRF} = \frac{499}{9} = 55.4\times \text{ risk reduction}$$

Policy complexity: up to $50 \times 49 = 2{,}450$ inter-segment rule pairs.

With identity-based policies (per-workload): $k = 500$, blast radius = 0 (only the compromised workload), policy rules up to $500 \times 499 = 249{,}500$ (but most are deny-by-default).

---

## 4. Behavioral Anomaly Detection (Statistical Distance)

### The Problem

Continuous verification requires detecting anomalous user or workload behavior. The system must compare current behavior against a baseline profile and flag deviations that exceed a statistical threshold.

### The Formula

User behavior profile as distribution over features $\mathbf{x} = (x_1, \ldots, x_m)$:

$$\mathbf{\mu} = E[\mathbf{x}], \quad \Sigma = \text{Cov}(\mathbf{x})$$

Mahalanobis distance of current observation $\mathbf{x}_{\text{now}}$:

$$D_M = \sqrt{(\mathbf{x}_{\text{now}} - \mathbf{\mu})^T \Sigma^{-1} (\mathbf{x}_{\text{now}} - \mathbf{\mu})}$$

Anomaly threshold (assuming multivariate normal, significance level $\alpha$):

$$D_M^2 \sim \chi^2(m) \implies \text{anomaly if } D_M^2 > \chi^2_{1-\alpha}(m)$$

Behavioral trust factor:

$$f_{\text{behavior}} = \max\left(0, 1 - \frac{D_M}{D_{\text{max}}}\right)$$

### Worked Examples

Features: login hour, session duration, API calls per minute ($m = 3$).

Baseline: $\mathbf{\mu} = (9.5, 45, 12)$, $\Sigma = \text{diag}(4, 100, 9)$.

Current observation: $\mathbf{x}_{\text{now}} = (3.0, 120, 50)$.

$$D_M^2 = \frac{(3-9.5)^2}{4} + \frac{(120-45)^2}{100} + \frac{(50-12)^2}{9}$$

$$= \frac{42.25}{4} + \frac{5625}{100} + \frac{1444}{9} = 10.56 + 56.25 + 160.44 = 227.25$$

$$\chi^2_{0.99}(3) = 11.34$$

$$227.25 \gg 11.34 \implies \text{anomaly detected}$$

$$f_{\text{behavior}} = \max(0, 1 - \frac{\sqrt{227.25}}{20}) = \max(0, 1 - 0.754) = 0.246$$

This low behavioral score would trigger step-up authentication or access denial.

---

## 5. PEP/PDP Latency Budget (Queuing Theory)

### The Problem

The PDP must evaluate access policies on every request. The added latency must fit within the application's SLA budget. PDP response time is modeled as a queuing system under load.

### The Formula

PDP modeled as M/M/1 queue with arrival rate $\lambda$ and service rate $\mu$:

$$E[W] = \frac{1}{\mu - \lambda} \quad \text{(mean response time)}$$

Server utilization:

$$\rho = \frac{\lambda}{\mu}$$

99th percentile response time:

$$W_{99} = -\frac{\ln(1 - 0.99 \cdot \rho)}{\mu(1 - \rho)}$$

For $c$ parallel PDP instances (M/M/c):

$$P_0 = \left[\sum_{k=0}^{c-1}\frac{(\lambda/\mu)^k}{k!} + \frac{(\lambda/\mu)^c}{c!}\cdot\frac{1}{1-\rho/c}\right]^{-1}$$

### Worked Examples

Request rate: $\lambda = 5{,}000$ req/s. PDP evaluation time: $\frac{1}{\mu} = 2$ ms ($\mu = 500$ req/s per instance).

Single instance: $\rho = \frac{5{,}000}{500} = 10 > 1$ (overloaded, queue grows unbounded).

Required instances: $c \geq \frac{\lambda}{\mu} = 10$. Use $c = 12$ for headroom.

Per-instance utilization: $\rho = \frac{5{,}000}{12 \times 500} = 0.833$.

Approximate $E[W] \approx \frac{2\text{ms}}{1 - 0.833} = 12\text{ms}$.

Latency budget: if application SLA is 200ms, PDP overhead is $\frac{12}{200} = 6\%$ -- acceptable.

If $\rho = 0.95$: $E[W] = \frac{2}{1 - 0.95} = 40\text{ms}$ (20% of budget -- concerning).

---

## 6. Device Posture Scoring (Weighted Boolean Evaluation)

### The Problem

Device posture assessment evaluates multiple compliance checks with different security weights. The composite posture score determines whether the device is trusted enough for resource access.

### The Formula

Posture checks $C = \{c_1, \ldots, c_k\}$ with results $r_i \in \{0, 1\}$ and weights $w_i$:

$$P_{\text{device}} = \frac{\sum_{i=1}^{k} w_i \cdot r_i}{\sum_{i=1}^{k} w_i}$$

Critical check enforcement (must-pass):

$$P_{\text{effective}} = P_{\text{device}} \cdot \prod_{c \in \text{critical}} r_c$$

If any critical check fails, effective posture = 0 regardless of score.

### Worked Examples

| Check | Weight | Result | Contribution |
|:---|:---:|:---:|:---:|
| Disk encryption | 5 (critical) | 1 | 5 |
| OS patches current | 4 | 1 | 4 |
| EDR running | 5 (critical) | 1 | 5 |
| Firewall enabled | 3 | 0 | 0 |
| Screen lock | 2 | 1 | 2 |
| Not jailbroken | 5 (critical) | 1 | 5 |

$$P_{\text{device}} = \frac{5+4+5+0+2+5}{5+4+5+3+2+5} = \frac{21}{24} = 0.875$$

$$P_{\text{effective}} = 0.875 \times 1 \times 1 \times 1 = 0.875$$

If EDR is not running ($r_{\text{EDR}} = 0$):

$$P_{\text{device}} = \frac{16}{24} = 0.667, \quad P_{\text{effective}} = 0.667 \times 0 = 0.0$$

Device denied access despite 66.7% raw score because a critical check failed.

---

## Prerequisites

- probability, exponential-decay, graph-theory, multivariate-statistics, queuing-theory, boolean-algebra, bayesian-inference
