# The Mathematics of OPA — Policy Evaluation as Logical Inference

> *A policy is a theorem. Every request is an attempt at proof. The engine is the judge.*

---

## 1. Rego Evaluation (Datalog and Set Comprehensions)

### The Problem

Rego is a declarative query language rooted in Datalog. Every policy evaluation is a search through a logical program to determine if a query is satisfiable. How do we reason about the complexity and completeness of policy evaluation?

### The Formula

A Rego program consists of rules of the form:

$$h \leftarrow b_1 \land b_2 \land \cdots \land b_n$$

where $h$ is the head (conclusion) and $b_i$ are body literals. Evaluation is bottom-up: compute all derivable facts from the base data and rules. For a program $P$ with facts $F$, the immediate consequence operator $T_P$:

$$T_P(I) = F \cup \{ h \mid (h \leftarrow b_1, \ldots, b_n) \in P, \{b_1, \ldots, b_n\} \subseteq I \}$$

The fixed point (all derivable facts):

$$T_P^{\infty} = \bigcup_{i=0}^{\infty} T_P^i(\emptyset)$$

For stratified Rego (no recursive negation), this converges in at most $|D|$ iterations where $|D|$ is the size of the data domain.

### Worked Examples

**Example 1**: A role hierarchy policy with 3 rules and a role database of 100 users:

```
allow if { input.user.role == "admin" }
allow if { input.user.role == "editor"; input.method == "GET" }
allow if { some role in data.roles[input.user.name]; role == "admin" }
```

The third rule requires iterating over `data.roles[user]`. If each user has at most $k$ roles, evaluation cost is $O(k)$. Total cost for a single query: $O(3 + k)$. For $k = 5$, this is $O(8)$ -- near-constant.

**Example 2**: A policy that checks all containers in a pod against a deny list of $m$ images:

$$\text{Cost} = O(c \times m)$$

where $c$ is the number of containers. For $c = 5$ and $m = 200$: $O(1000)$ comparisons.

---

## 2. Partial Evaluation (Residual Programs)

### The Problem

When some inputs are known at deploy time but others arrive at runtime, partial evaluation specializes the policy for known inputs, producing a simpler residual program. How much speedup can we expect?

### The Formula

Given a program $P$ with input variables partitioned into known $\vec{k}$ and unknown $\vec{u}$, partial evaluation produces a residual program $P'$:

$$P' = \text{PE}(P, \vec{k})$$

The speedup factor depends on the fraction of the search space eliminated:

$$\text{Speedup} = \frac{|S_P|}{|S_{P'}|}$$

where $|S_P|$ is the original search space and $|S_{P'}|$ is the residual search space. If the known inputs eliminate $f$ fraction of conditions:

$$\text{Speedup} \approx \frac{1}{1 - f}$$

### Worked Examples

**Example 1**: A policy with 10 conditions, 6 depending only on the HTTP method (known at deploy time). Partial evaluation eliminates 6 conditions:

$$f = 0.6, \quad \text{Speedup} = \frac{1}{1 - 0.6} = 2.5\times$$

The residual policy only evaluates 4 runtime conditions.

**Example 2**: An RBAC policy with 50 role-path mappings. If the API path is known at compile time, partial evaluation filters to only matching rules. If 5 rules match the known path:

$$\text{Speedup} = \frac{50}{5} = 10\times$$

The `/v1/compile` endpoint returns only the residual rules as a partial evaluation result.

---

## 3. Admission Control Latency (Queuing with Deadlines)

### The Problem

Gatekeeper sits in the Kubernetes admission webhook path. Every API request waits for the policy decision. If evaluation exceeds the webhook timeout (default 10s), the request fails. What is the probability of timeout under load?

### The Formula

Model policy evaluation time as a random variable $X$ with mean $\mu_X$ and variance $\sigma_X^2$. For $n$ constraints evaluated sequentially, the total evaluation time:

$$T = \sum_{i=1}^{n} X_i$$

By the Central Limit Theorem, for large $n$:

$$T \sim \mathcal{N}(n\mu_X, n\sigma_X^2)$$

The probability of exceeding timeout $\tau$:

$$P(T > \tau) = 1 - \Phi\left(\frac{\tau - n\mu_X}{\sigma_X\sqrt{n}}\right)$$

where $\Phi$ is the standard normal CDF.

### Worked Examples

**Example 1**: 20 constraint templates, each taking $\mu_X = 5\text{ms}$ with $\sigma_X = 2\text{ms}$. Timeout $\tau = 10000\text{ms}$:

$$E[T] = 20 \times 5 = 100\text{ms}$$

$$P(T > 10000) = 1 - \Phi\left(\frac{10000 - 100}{2\sqrt{20}}\right) = 1 - \Phi(1107.4) \approx 0$$

No timeout risk. But with 500 constraints at $\mu_X = 15\text{ms}$, $\sigma_X = 10\text{ms}$:

$$E[T] = 7500\text{ms}$$

$$P(T > 10000) = 1 - \Phi\left(\frac{10000 - 7500}{10\sqrt{500}}\right) = 1 - \Phi(11.18) \approx 0$$

Still safe, but $E[T]$ is 75% of the timeout budget. At $\mu_X = 20\text{ms}$:

$$E[T] = 10000\text{ms}, \quad P(T > 10000) = 0.5$$

Half of all requests would time out.

**Example 2**: To ensure 99.9% of requests complete within timeout:

$$n\mu_X + 3.09 \cdot \sigma_X\sqrt{n} \leq \tau$$

For $\mu_X = 10\text{ms}$, $\sigma_X = 5\text{ms}$, $\tau = 10000\text{ms}$:

$$10n + 15.45\sqrt{n} \leq 10000$$

Solving: $n \leq 992$ constraints.

---

## 4. Policy Coverage (Set Cover Problem)

### The Problem

Given a set of resources to protect and a collection of policies, each covering a subset of resources, what is the minimum number of policies needed for complete coverage? This is the classic Set Cover problem.

### The Formula

Let $U$ be the universe of resource types and $S = \{S_1, S_2, \ldots, S_m\}$ be the collection of policies where $S_i \subseteq U$. The minimum set cover is:

$$\min |C| \quad \text{subject to} \quad \bigcup_{S_i \in C} S_i = U, \quad C \subseteq S$$

This is NP-hard, but the greedy algorithm achieves an approximation ratio of:

$$|C_{\text{greedy}}| \leq H(|U|) \cdot |C_{\text{opt}}|$$

where $H(n) = \sum_{k=1}^{n} 1/k \approx \ln n$ is the harmonic number.

### Worked Examples

**Example 1**: 8 resource types (Pods, Deployments, Services, Ingresses, ConfigMaps, Secrets, Namespaces, PVCs). Three candidate policies:

- $S_1$ = {Pods, Deployments, Services} (workload policy)
- $S_2$ = {Ingresses, Services, ConfigMaps} (networking policy)
- $S_3$ = {Secrets, ConfigMaps, Namespaces, PVCs} (data policy)

$S_1 \cup S_2 \cup S_3 = U$, so 3 policies cover all resources. The greedy bound:

$$|C| \leq H(8) \times |C_{\text{opt}}| = 2.72 \times |C_{\text{opt}}|$$

Since $|C_{\text{opt}}| \geq 3$ (no single pair covers all 8), greedy is optimal here.

**Example 2**: With 20 resource types and policies averaging 4 types each:

$$|C_{\text{opt}}| \geq \lceil 20/4 \rceil = 5$$

Greedy guarantee: $|C| \leq H(20) \times 5 \approx 3.0 \times 5 = 15$ policies maximum.

---

## 5. Decision Log Analysis (Anomaly Detection)

### The Problem

Decision logs generate a stream of allow/deny decisions. How do we detect anomalous patterns that indicate policy misconfiguration or attack attempts?

### The Formula

Model the deny rate as a time series $r(t)$. The expected deny rate under normal operation is $\mu_r$ with standard deviation $\sigma_r$. An anomaly is detected when:

$$|r(t) - \mu_r| > k\sigma_r$$

Using exponentially weighted moving average (EWMA) for adaptive baselines:

$$\hat{\mu}(t) = \alpha \cdot r(t) + (1 - \alpha) \cdot \hat{\mu}(t-1)$$

$$\hat{\sigma}^2(t) = \alpha \cdot (r(t) - \hat{\mu}(t))^2 + (1 - \alpha) \cdot \hat{\sigma}^2(t-1)$$

### Worked Examples

**Example 1**: Normal deny rate is 2% ($\mu_r = 0.02$, $\sigma_r = 0.005$). A spike to 15% triggers at $k = 3$:

$$|0.15 - 0.02| = 0.13 > 3 \times 0.005 = 0.015$$

Alert fires. This could indicate a policy change blocking legitimate traffic or a brute-force attack.

**Example 2**: Gradually increasing deny rate from 2% to 8% over 24 hours. With EWMA ($\alpha = 0.1$), the baseline adapts slowly:

After 10 observations at 8%: $\hat{\mu} \approx 0.02 + 0.1(0.08-0.02) \times 10 \approx 0.062$

The slow adaptation means the anomaly is detected within the first few observations, before the baseline fully adjusts.

---

## Prerequisites

- First-order predicate logic (universal/existential quantification, unification)
- Datalog semantics (stratification, fixed-point computation, negation-as-failure)
- Kubernetes admission control webhooks (validating, mutating, audit)
- Computational complexity (P, NP, NP-hard, approximation algorithms)
- Probability theory (normal distribution, CDF, Central Limit Theorem)
- Time series analysis (moving averages, anomaly detection thresholds)
- Set theory (unions, intersections, set cover)
