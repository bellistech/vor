# The Mathematics of Chaos Engineering — MTTR, Reliability, and Cascading Failures

> *Chaos engineering is applied reliability theory. This explores the mathematics of mean time to recovery under fault injection, the probability models behind failure cascading, the composition of nines for system reliability, and queuing theory predictions under degraded conditions.*

---

## 1. MTTR Modeling Under Chaos (Reliability Theory)

### The Problem

Chaos experiments measure the system's ability to recover from faults. Mean Time To Recovery (MTTR) is the key metric. How do we model and optimize it?

### The Formula

**Availability** from MTBF and MTTR:

$$A = \frac{MTBF}{MTBF + MTTR}$$

For a target of 99.9% availability:

$$MTTR \leq \frac{MTBF \times (1 - 0.999)}{0.999} = \frac{MTBF}{999}$$

**MTTR decomposition**:

$$MTTR = T_{detect} + T_{diagnose} + T_{resolve} + T_{verify}$$

Each phase can be modeled:

| Phase | Distribution | Typical values |
|-------|-------------|---------------|
| $T_{detect}$ | Exponential (monitoring interval) | 30s - 5min |
| $T_{diagnose}$ | Log-normal (human cognition) | 5min - 2hr |
| $T_{resolve}$ | Exponential (automation) or Log-normal (manual) | 1min - 1hr |
| $T_{verify}$ | Deterministic (health check interval) | 30s - 5min |

**Expected MTTR with automation**:

$$E[MTTR_{auto}] = E[T_{detect}] + E[T_{resolve,auto}] + E[T_{verify}]$$

$$E[MTTR_{manual}] = E[T_{detect}] + E[T_{diagnose}] + E[T_{resolve,manual}] + E[T_{verify}]$$

Automation eliminates $T_{diagnose}$ and reduces $T_{resolve}$.

### Worked Examples

**Example**: Pre-chaos vs post-chaos MTTR for a pod failure.

Before chaos program:
- $T_{detect} = 5$ min (manual observation)
- $T_{diagnose} = 30$ min (log diving)
- $T_{resolve} = 15$ min (manual restart)
- $T_{verify} = 5$ min
- $MTTR = 55$ min

After chaos-driven improvements:
- $T_{detect} = 30$ sec (automated alerting)
- $T_{diagnose} = 0$ (auto-remediation)
- $T_{resolve} = 45$ sec (Kubernetes auto-restart)
- $T_{verify} = 30$ sec (readiness probe)
- $MTTR = 1.75$ min

Availability improvement (assuming $MTBF = 720$ hours):
- Before: $A = \frac{720}{720 + 0.917} = 99.873\%$
- After: $A = \frac{720}{720 + 0.029} = 99.996\%$

From 3 nines to nearly 5 nines, just by reducing MTTR.

## 2. Failure Injection Probability (Experimental Design)

### The Problem

How often should we inject faults, and what injection rate provides statistically meaningful results without excessive risk?

### The Formula

**Injection rate** $\lambda_{inject}$: faults injected per unit time. Natural failure rate: $\lambda_{natural}$.

**Effective failure rate** during chaos experiment:

$$\lambda_{effective} = \lambda_{natural} + \lambda_{inject}$$

**Error budget consumption**: with error budget $E_b$ (in downtime-minutes per month):

$$E_b = (1 - A_{target}) \times T_{month}$$

For 99.9% availability: $E_b = 0.001 \times 43200 = 43.2$ minutes/month.

**Safe chaos budget**: allocate fraction $f$ of error budget to chaos:

$$T_{chaos} \leq f \times E_b$$

With $f = 0.1$ (10% of error budget): $T_{chaos} \leq 4.32$ minutes/month.

**Number of experiments per month** with expected impact duration $d$:

$$N_{experiments} \leq \frac{T_{chaos}}{d}$$

### Worked Examples

**Example**: 99.95% availability target, expected chaos impact of 30 seconds per experiment.

$$E_b = 0.0005 \times 43200 = 21.6 \text{ min/month}$$

$$T_{chaos} \leq 0.1 \times 21.6 = 2.16 \text{ min/month}$$

$$N_{experiments} \leq \frac{2.16}{0.5} = 4.32$$

At most 4 chaos experiments per month, each expected to cause 30 seconds of degradation.

## 3. Cascading Failure Graph Analysis (Network Reliability)

### The Problem

When one component fails, it can trigger failures in dependent components. How do we model and predict cascading failures?

### The Formula

Model the system as a directed graph $G = (V, E)$ where $V$ is the set of services and $E$ represents dependencies. Edge $(u, v) \in E$ means $v$ depends on $u$.

**Single failure cascade**: when service $u$ fails, the set of affected services:

$$\text{cascade}(u) = \{v \in V : \exists \text{ path from } u \text{ to } v \text{ in } G\}$$

**Cascade probability**: if service $u$ fails with probability $p_u$, and each dependency propagates failure with probability $q_{uv}$:

$$P(\text{cascade reaches } v | u \text{ fails}) = 1 - \prod_{\text{paths } u \to v}\left(1 - \prod_{(a,b) \in \text{path}} q_{ab}\right)$$

For a simple chain $A \to B \to C$ with equal propagation probability $q$:

$$P(C \text{ fails} | A \text{ fails}) = q^2$$

For fan-out topology ($A \to B, A \to C, A \to D$):

$$P(\text{any downstream fails} | A \text{ fails}) = 1 - (1-q)^3$$

**Circuit breaker** reduces $q$: when the circuit is open, $q_{uv} \approx 0$ for the affected edge.

### Worked Examples

**Example**: Microservices topology:

```
API Gateway -> Auth Service -> User DB
API Gateway -> Product Service -> Product DB
Product Service -> Inventory Service -> Inventory DB
```

If User DB fails ($p = 0.001$) with propagation $q = 0.8$:

- Auth Service failure: $P = 0.001 \times 0.8 = 0.0008$
- API Gateway failure (via Auth): $P = 0.001 \times 0.8 \times 0.8 = 0.00064$
- Product Service: unaffected ($P = 0$, no dependency path)

With circuit breaker on Auth Service ($q \to 0.05$ when open):
- API Gateway failure: $P = 0.001 \times 0.05 \times 0.8 = 0.00004$

16x reduction in cascade probability.

## 4. Reliability Mathematics — Nines Composition (System Reliability)

### The Problem

Each component has its own availability. How do we compute system-level availability from component availabilities?

### The Formula

**Series system** (all components must work):

$$A_{series} = \prod_{i=1}^{n} A_i$$

**Parallel system** (at least one component must work):

$$A_{parallel} = 1 - \prod_{i=1}^{n}(1 - A_i)$$

**k-of-n system** (at least $k$ of $n$ components must work):

$$A_{k/n} = \sum_{i=k}^{n}\binom{n}{i}A^i(1-A)^{n-i}$$

(assuming identical component availability $A$).

**Nines arithmetic** (useful shortcut):

For components in series with availabilities expressed as "nines":
- 99.9% + 99.9% in series $\approx$ 99.8% (nines subtract logarithmically)
- 99.9% + 99.9% in parallel $\approx$ 99.9999% (nines add)

### Worked Examples

**Example 1**: Request path through 4 services, each with 99.9% availability.

$$A_{system} = 0.999^4 = 0.996 = 99.6\%$$

Four nines components in series give less than three nines.

**Example 2**: Database with primary (99.95%) and replica (99.95%) in active-passive:

$$A_{db} = 1 - (1 - 0.9995)^2 = 1 - 0.0005^2 = 1 - 0.00000025 = 99.999975\%$$

Nearly 7 nines — redundancy dramatically improves availability.

**Example 3**: 3-node cluster, needs 2 of 3 for quorum (each 99.9%):

$$A_{2/3} = \binom{3}{2}(0.999)^2(0.001) + \binom{3}{3}(0.999)^3$$
$$= 3 \times 0.998 \times 0.001 + 0.997 = 0.002994 + 0.997 = 0.99999$$

99.999% — five nines from three-nines components.

## 5. Queuing Theory Under Fault Injection (Performance Degradation)

### The Problem

When a fault reduces capacity, how do latency and queue depth change? Queuing theory predicts the nonlinear relationship between utilization and response time.

### The Formula

**M/M/1 queue** (single server, exponential arrivals and service):

$$W = \frac{1}{\mu - \lambda} = \frac{1}{\mu(1 - \rho)}$$

where $\rho = \lambda / \mu$ is utilization, $\lambda$ is arrival rate, $\mu$ is service rate.

**Average queue length**:

$$L = \frac{\rho}{1 - \rho}$$

**When a fault reduces capacity** by fraction $f$ (e.g., losing 1 of 3 servers, $f = 1/3$):

$$\mu' = \mu(1 - f)$$

$$\rho' = \frac{\lambda}{\mu'} = \frac{\rho}{1 - f}$$

$$W' = \frac{1}{\mu' - \lambda} = \frac{1}{\mu(1-f) - \lambda}$$

**Critical threshold**: system becomes unstable when $\rho' \geq 1$, i.e., when:

$$f \geq 1 - \rho$$

### Worked Examples

**Example**: 3-server system with $\lambda = 200$ rps, $\mu = 100$ rps per server ($\mu_{total} = 300$). Utilization $\rho = 200/300 = 0.667$.

Normal response time (M/M/c approximation):
$$W \approx \frac{1}{\mu_{total} - \lambda} = \frac{1}{100} = 10\text{ms}$$

Chaos: kill 1 server ($f = 1/3$, $\mu' = 200$ rps):
$$\rho' = 200/200 = 1.0$$

**System is at saturation.** Response times approach infinity. Queue depth grows without bound.

Chaos: kill 1 server but shed 20% traffic ($\lambda' = 160$):
$$\rho' = 160/200 = 0.8$$
$$W' = \frac{1}{200 - 160} = 25\text{ms}$$

Response time increases 2.5x, but system remains stable. This demonstrates why load shedding is critical during partial outages.

## Prerequisites

- Probability theory (exponential distribution, Bernoulli trials)
- Graph theory (directed graphs, reachability, paths)
- Queuing theory (M/M/1, utilization, stability conditions)
- Reliability engineering (MTBF, MTTR, availability)

## Complexity

| Analysis | Time Complexity | Space Complexity |
|----------|----------------|-----------------|
| Cascade reachability (BFS/DFS) | $O(|V| + |E|)$ | $O(|V|)$ |
| Series/parallel availability | $O(n)$ | $O(1)$ |
| k-of-n availability | $O(n)$ | $O(1)$ |
| Cascade probability (all paths) | $O(|V|! / (|V|-d)!)$ worst case | $O(|V|)$ |
| Queuing analysis (M/M/c) | $O(c)$ | $O(1)$ |

Where: $|V|$ = services, $|E|$ = dependencies, $n$ = redundant components, $c$ = servers, $d$ = path depth.
