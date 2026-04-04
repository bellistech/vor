# The Mathematics of External Secrets — Synchronization as Eventual Consistency

> *A secret known in two places is a consistency problem. A secret known in zero places is an outage.*

---

## 1. Refresh Interval Optimization (Queueing Theory)

### The Problem

Each ExternalSecret polls its provider at a fixed `refreshInterval`. With thousands of ExternalSecrets and rate-limited APIs, how do we choose intervals that balance freshness against provider capacity?

### The Formula

Model the system as an M/D/1 queue where $n$ ExternalSecrets each poll at interval $T$. The aggregate arrival rate at the provider API is:

$$\lambda = \frac{n}{T}$$

The provider has a service rate $\mu$ (requests per second). For stability, we need $\rho = \lambda / \mu < 1$. The minimum safe refresh interval:

$$T_{\min} = \frac{n}{\mu}$$

The average waiting time in an M/D/1 queue:

$$W_q = \frac{\rho}{2\mu(1 - \rho)}$$

### Worked Examples

**Example 1**: 500 ExternalSecrets polling AWS Secrets Manager. AWS rate limit is approximately 5000 requests/second ($\mu = 5000$). With $T = 3600\text{s}$ (1 hour):

$$\lambda = \frac{500}{3600} \approx 0.139 \text{ req/s}$$

$$\rho = \frac{0.139}{5000} \approx 0.0000278$$

Extremely low utilization. The minimum viable interval:

$$T_{\min} = \frac{500}{5000} = 0.1\text{s}$$

Provider capacity is not the bottleneck here; cost per API call is the real constraint.

**Example 2**: 10,000 ExternalSecrets against a self-hosted Vault instance handling $\mu = 100$ req/s. With $T = 60\text{s}$:

$$\lambda = \frac{10000}{60} \approx 166.7 \text{ req/s}$$

$$\rho = \frac{166.7}{100} = 1.67 > 1$$

The system is unstable. Minimum safe interval:

$$T_{\min} = \frac{10000}{100} = 100\text{s}$$

Setting $T = 300\text{s}$ gives $\rho = 0.33$, with average wait:

$$W_q = \frac{0.33}{2 \times 100 \times 0.67} \approx 2.5\text{ms}$$

---

## 2. Staleness Probability (Freshness Analysis)

### The Problem

A secret rotates externally at some rate. The ExternalSecret polls at a fixed interval. What is the expected staleness -- the time a Kubernetes Secret holds an outdated value?

### The Formula

Let secret rotation follow a Poisson process with rate $\alpha$ (rotations per unit time) and polling interval $T$. The expected staleness after a rotation event:

$$E[\text{staleness}] = \frac{T}{2}$$

The probability that the Kubernetes Secret is stale at any random observation time $t$:

$$P(\text{stale}) = \frac{\alpha \cdot T/2}{\alpha \cdot T/2 + 1} \approx \frac{\alpha T}{2} \quad \text{for small } \alpha T$$

The maximum staleness (worst case) is bounded by:

$$S_{\max} = T$$

### Worked Examples

**Example 1**: Database password rotates every 24 hours ($\alpha = 1/86400$), polling every 1 hour ($T = 3600\text{s}$):

$$E[\text{staleness}] = \frac{3600}{2} = 1800\text{s} = 30\text{ minutes}$$

$$P(\text{stale at observation}) \approx \frac{(1/86400) \times 3600}{2} \approx 0.021$$

About 2.1% of the time, the in-cluster secret is outdated.

**Example 2**: An API key rotates every 4 hours ($\alpha = 6/86400$) with 30-minute polling ($T = 1800\text{s}$):

$$E[\text{staleness}] = 900\text{s} = 15\text{ minutes}$$

$$P(\text{stale}) \approx \frac{(6/86400) \times 1800}{2} \approx 0.063$$

To achieve $P(\text{stale}) < 0.01$:

$$T < \frac{2 \times 0.01}{\alpha} = \frac{0.02}{6/86400} = 288\text{s} \approx 5\text{ minutes}$$

---

## 3. Multi-Provider Consistency (Vector Clocks)

### The Problem

When secrets span multiple providers (e.g., database credentials in Vault, API keys in AWS SM), the Kubernetes workload sees a composite state. How do we reason about cross-provider consistency?

### The Formula

Define a consistency vector $\vec{v} = (v_1, v_2, \ldots, v_k)$ where $v_i$ is the version of the secret from provider $i$. The system is consistent when all providers reflect the intended version set $\vec{v}^*$:

$$\text{consistent}(\vec{v}) = \begin{cases} 1 & \text{if } \vec{v} = \vec{v}^* \\ 0 & \text{otherwise} \end{cases}$$

With $k$ providers each polling at interval $T_i$ and rotation at time $t_0$, the convergence time:

$$T_{\text{converge}} = \max_{i=1}^{k} T_i$$

The window of inconsistency (some providers updated, others not):

$$W_{\text{inconsistent}} = T_{\text{converge}} - \min_{i=1}^{k} \delta_i$$

where $\delta_i$ is the actual sync delay for provider $i$ (uniformly distributed in $[0, T_i]$).

### Worked Examples

**Example 1**: A rotation event updates Vault (polled every 5m) and AWS SM (polled every 15m). The expected convergence time:

$$E[T_{\text{converge}}] = \max(E[\delta_1], E[\delta_2]) = \max(2.5, 7.5) = 7.5\text{ minutes}$$

The expected inconsistency window:

$$E[W] = E[\max(\delta_1, \delta_2)] - E[\min(\delta_1, \delta_2)]$$

For independent uniform random variables on $[0, 5]$ and $[0, 15]$:

$$E[W] \approx 7.5 - 1.67 = 5.83\text{ minutes}$$

**Example 2**: To reduce inconsistency, align polling intervals. Setting both to $T = 5\text{m}$:

$$E[W] = E[\max(\delta_1, \delta_2)] - E[\min(\delta_1, \delta_2)] = \frac{2 \times 5}{3} - \frac{5}{3} = \frac{5}{3} \approx 1.67\text{ minutes}$$

---

## 4. Secret Sprawl Entropy (Information Theory)

### The Problem

As secrets replicate across namespaces and clusters, the attack surface grows. How do we quantify the information exposure of a secret distribution topology?

### The Formula

Define the exposure entropy of a secret $s$ replicated to $m$ locations, each with independent compromise probability $p_i$:

$$H_{\text{exposure}}(s) = -\sum_{i=1}^{m} \left[ p_i \log_2 p_i + (1 - p_i) \log_2 (1 - p_i) \right]$$

The probability that the secret is compromised in at least one location:

$$P(\text{compromised}) = 1 - \prod_{i=1}^{m} (1 - p_i)$$

For uniform risk $p_i = p$:

$$P(\text{compromised}) = 1 - (1 - p)^m$$

### Worked Examples

**Example 1**: A database password exists in 8 namespaces, each with $p = 0.005$ compromise probability per year:

$$P(\text{compromised}) = 1 - (1 - 0.005)^8 = 1 - 0.995^8 \approx 0.039$$

Reducing to 2 namespaces via ClusterSecretStore:

$$P = 1 - 0.995^2 \approx 0.010$$

A 4x reduction in compromise probability.

**Example 2**: Compare two architectures -- one with 50 ExternalSecrets each syncing one key, another with 5 ExternalSecrets each syncing 10 keys:

Architecture A: 50 copies, $P_A = 1 - (1 - p)^{50}$

Architecture B: 5 copies, $P_B = 1 - (1 - p)^5$

At $p = 0.001$: $P_A \approx 0.049$, $P_B \approx 0.005$. Consolidation reduces risk by 10x.

---

## 5. Reconciliation Convergence (Fixed-Point Iteration)

### The Problem

The external-secrets operator is a Kubernetes controller that reconciles desired state (ExternalSecret spec) with actual state (Kubernetes Secret). How quickly does the system converge after a perturbation?

### The Formula

Model reconciliation as a fixed-point iteration. Let $x_n$ be the state at reconciliation cycle $n$ and $f(x)$ be the reconciliation function. Convergence requires:

$$|f'(x^*)| < 1$$

where $x^*$ is the fixed point (desired state). The convergence rate is:

$$|x_{n+1} - x^*| \leq L \cdot |x_n - x^*|$$

where $L = |f'(x^*)|$ is the Lipschitz constant. The number of cycles to reach tolerance $\epsilon$ from initial error $e_0$:

$$n \geq \frac{\ln(\epsilon / e_0)}{\ln L}$$

### Worked Examples

**Example 1**: After deleting a Kubernetes Secret, the operator must recreate it. With $L = 0$ (single-step convergence for creation), the secret is restored in exactly 1 reconciliation cycle:

$$n = 1, \quad T_{\text{restore}} = T_{\text{reconcile}} \leq T_{\text{refresh}}$$

**Example 2**: A cascading update where secret A depends on secret B (via template). If B changes, A must re-template. With reconciliation interval $T$, the worst-case cascade delay for a chain of depth $d$:

$$T_{\text{cascade}} = d \times T$$

For $d = 3$ dependencies and $T = 60\text{s}$: $T_{\text{cascade}} = 180\text{s}$.

---

## Prerequisites

- Kubernetes Secret types (Opaque, TLS, docker-registry) and encoding (base64)
- Kubernetes operator pattern and controller reconciliation loop
- Cloud provider IAM: AWS IAM/IRSA, GCP Workload Identity, Azure Managed Identity
- HashiCorp Vault architecture (secret engines, auth methods, policies)
- Go template syntax (used in ExternalSecret templating)
- Queueing theory fundamentals (arrival rate, service rate, utilization)
- Information theory (entropy, probability of compound events)
