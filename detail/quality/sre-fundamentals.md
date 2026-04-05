# The Mathematics of Reliability — Nines, Queues, and Failure Rates

> *Reliability engineering is fundamentally a quantitative discipline. The mathematics of availability, queuing, and failure probability transform vague reliability goals into precise operational targets with measurable consequences.*

---

## 1. The Nines Table (Availability Mathematics)

### The Problem

What does "five nines" actually mean in terms of permitted downtime? How do availability targets translate to operational constraints?

### The Formula

Availability $A$ is expressed as a percentage. The "number of nines" $n$ corresponds to:

$$A = 1 - 10^{-n}$$

Permitted downtime per period $T$:

$$D = T \times (1 - A) = T \times 10^{-n}$$

### Worked Examples

| Nines | Availability | Downtime/year | Downtime/month | Downtime/week |
|---|---|---|---|---|
| 1 nine (90%) | 90.000% | 36.53 days | 73.05 hours | 16.80 hours |
| 2 nines (99%) | 99.000% | 3.65 days | 7.31 hours | 1.68 hours |
| 3 nines (99.9%) | 99.900% | 8.77 hours | 43.83 min | 10.08 min |
| 3.5 nines | 99.950% | 4.38 hours | 21.92 min | 5.04 min |
| 4 nines (99.99%) | 99.990% | 52.60 min | 4.38 min | 1.01 min |
| 4.5 nines | 99.995% | 26.30 min | 2.19 min | 30.24 sec |
| 5 nines (99.999%) | 99.999% | 5.26 min | 26.30 sec | 6.05 sec |
| 6 nines | 99.9999% | 31.56 sec | 2.63 sec | 0.60 sec |

**Cost of each additional nine**: Achieving each additional nine roughly requires 10x the engineering investment. Going from 99.9% to 99.99% is far harder than going from 99% to 99.9%.

---

## 2. Error Budget Burn Rate Algebra (Alerting Thresholds)

### The Problem

How fast is the error budget being consumed, and when should we alert?

### The Formula

Let $B = 1 - \text{SLO}$ be the total error budget fraction. Over window $W$ (e.g., 30 days), the ideal burn rate is:

$$\text{burn\_rate}_{\text{ideal}} = \frac{B}{W}$$

The actual burn rate over measurement window $w$:

$$\text{burn\_rate}_{\text{actual}} = \frac{e_w}{w}$$

Where $e_w$ = fraction of errors in window $w$.

The burn rate multiplier $\beta$:

$$\beta = \frac{\text{burn\_rate}_{\text{actual}}}{\text{burn\_rate}_{\text{ideal}}} = \frac{e_w \cdot W}{w \cdot B}$$

$\beta = 1$: budget consumed at exactly the sustainable rate.
$\beta = 14.4$: budget will exhaust in $W / 14.4 \approx 2$ days (for 30-day window).

**Time to budget exhaustion**:

$$T_{\text{exhaust}} = \frac{B_{\text{remaining}}}{\text{burn\_rate}_{\text{actual}}}$$

**Multi-window alerting** (Google SRE recommendation):

| Alert | Long window | Short window | $\beta$ | Budget consumed |
|---|---|---|---|---|
| Page (fast) | 1 hour | 5 min | 14.4 | 2% in 1h |
| Page (slow) | 6 hours | 30 min | 6.0 | 5% in 6h |
| Ticket | 3 days | 6 hours | 1.0 | 10% in 3d |

### Worked Examples

SLO = 99.9%, Window = 30 days. Budget $B = 0.001$.

Current error rate = 0.5% over the last hour. $\beta = \frac{0.005 \times 30 \times 24}{1 \times 0.001} = 3600$. This is an extreme burn rate — the budget would exhaust in $30 \times 24 / 3600 = 0.2$ hours (12 minutes). Immediate page.

Current error rate = 0.002% over the last 6 hours. $\beta = \frac{0.00002 \times 720}{6 \times 0.001} = 2.4$. Moderate burn — budget would exhaust in $720/2.4 = 300$ hours (12.5 days). Create a ticket.

---

## 3. Composite SLO Calculation (Dependent Services)

### The Problem

If service A depends on services B and C, what is the composite SLO? How do dependent service SLOs combine?

### The Formula

**Serial dependency** (A requires both B and C):

$$A_{\text{composite}} = A_A \times A_B \times A_C$$

**Parallel dependency** (A requires at least one of B or C):

$$A_{\text{composite}} = A_A \times (1 - (1 - A_B)(1 - A_C))$$

**Partial dependency** (A degrades but functions without B):

$$A_{\text{composite}} = A_A \times (A_B + (1 - A_B) \times A_{\text{degraded}})$$

Where $A_{\text{degraded}}$ is the availability of A's degraded mode.

### Worked Examples

API Gateway (99.99%) depends on:
- Auth Service (99.9%)
- User Service (99.9%)
- Cache (99.99%) — degraded mode available without cache

Serial path (auth required): $A = 0.9999 \times 0.999 = 0.9989 \approx 99.89\%$

Adding user service: $A = 0.9999 \times 0.999 \times 0.999 = 0.9979 \approx 99.79\%$

Three services at 99.9% serial = $0.999^3 = 99.7\%$. Two nines and change from three separate three-nines services.

With cache degradation (95% availability in degraded mode):
$A = 0.9979 \times (0.9999 + (1 - 0.9999) \times 0.95) = 0.9979 \times 0.999995 \approx 99.79\%$

Cache's high availability and graceful degradation means it barely affects the composite.

---

## 4. Queuing Theory (Little's Law and M/M/1)

### The Problem

How do we predict queue depths, wait times, and throughput for request-driven services?

### The Formula

**Little's Law** (the most useful result in queuing theory):

$$L = \lambda W$$

Where:
- $L$ = average number of items in the system (queue + being served)
- $\lambda$ = average arrival rate (items/second)
- $W$ = average time an item spends in the system

This holds for any stable queueing system regardless of distribution.

**M/M/1 queue** (Poisson arrivals, exponential service, 1 server):

- Arrival rate: $\lambda$, Service rate: $\mu$
- Utilization: $\rho = \lambda / \mu$ (must be $< 1$ for stability)
- Average items in system: $L = \rho / (1 - \rho)$
- Average wait time: $W = 1 / (\mu - \lambda)$
- Average queue length: $L_q = \rho^2 / (1 - \rho)$
- Average time in queue: $W_q = \rho / (\mu - \lambda)$

**Key insight**: As $\rho \rightarrow 1$, $L \rightarrow \infty$ and $W \rightarrow \infty$. Systems near saturation exhibit exponentially growing latency, not linear. At 90% utilization, $L = 9$ items waiting. At 95%, $L = 19$.

### Worked Examples

API server: $\lambda = 900$ req/s, $\mu = 1000$ req/s.

$\rho = 0.9$. $L = 0.9/0.1 = 9$ requests in system. $W = 1/(1000-900) = 10$ ms average.

If traffic increases 5% to $\lambda = 945$: $\rho = 0.945$. $L = 0.945/0.055 = 17.2$. $W = 1/55 = 18.2$ ms. A 5% traffic increase caused an 82% increase in latency.

If traffic increases to $\lambda = 990$: $\rho = 0.99$. $L = 99$. $W = 100$ ms. Near saturation is catastrophic.

**Little's Law application**: If we observe $L = 50$ requests in flight and $\lambda = 1000$ req/s, then $W = L/\lambda = 50/1000 = 50$ ms average latency. No need to measure latency directly.

---

## 5. MTTR, MTTF, MTBF Relationships (Failure Analysis)

### The Problem

How do we measure and relate the key reliability metrics: time to repair, time between failures, and time to failure?

### The Formula

**Definitions**:
- **MTTF** (Mean Time To Failure): Average time a system operates before failure (non-repairable systems)
- **MTTR** (Mean Time To Repair): Average time to restore service after failure
- **MTBF** (Mean Time Between Failures): Average time between consecutive failures (repairable systems)

**Relationship**:

$$\text{MTBF} = \text{MTTF} + \text{MTTR}$$

**Availability as a function of MTBF and MTTR**:

$$A = \frac{\text{MTBF}}{\text{MTBF} + \text{MTTR}} = \frac{\text{MTTF}}{\text{MTTF} + \text{MTTR}}$$

**Improving availability**: Can either increase MTTF (prevent failures) or decrease MTTR (recover faster).

Reducing MTTR is usually more cost-effective:
- Improving MTTF from 100h to 200h with MTTR=1h: $A = 100/101 \rightarrow 200/201 = 99.01\% \rightarrow 99.50\%$
- Reducing MTTR from 1h to 0.1h with MTTF=100h: $A = 100/101 \rightarrow 100/100.1 = 99.01\% \rightarrow 99.90\%$

**Failure rate** $\lambda_f = 1/\text{MTTF}$. For $n$ independent components in series:

$$\lambda_{\text{system}} = \sum_{i=1}^{n} \lambda_i$$

$$\text{MTTF}_{\text{system}} = \frac{1}{\sum 1/\text{MTTF}_i}$$

### Worked Examples

Service with MTTF = 720 hours (30 days) and MTTR = 2 hours:

$$A = \frac{720}{720 + 2} = 99.72\%$$

To reach 99.9%: need $\text{MTTF}/(\text{MTTF} + \text{MTTR}) \geq 0.999$.

Option A: Increase MTTF to 2000 hours. $A = 2000/2002 = 99.90\%$.
Option B: Reduce MTTR to 0.72 hours (43 min). $A = 720/720.72 = 99.90\%$.

Option B is usually easier: invest in better monitoring, runbooks, and automated remediation rather than preventing all failures.

Two services in series, each with MTTF = 720h, MTTR = 1h:

$$\text{MTTF}_{\text{system}} = \frac{1}{1/720 + 1/720} = 360 \text{ hours}$$

$$A_{\text{system}} = \frac{360}{360 + 1} = 99.72\%$$

---

## 6. Availability as f(MTBF, MTTR) — Operational Targets

### The Problem

Given a target availability, what operational constraints does that impose on failure frequency and recovery speed?

### The Formula

From $A = \text{MTBF} / (\text{MTBF} + \text{MTTR})$:

$$\text{MTTR} = \text{MTBF} \times \frac{1 - A}{A}$$

For a given availability target, the relationship between permissible failure frequency and recovery time:

| Target $A$ | If MTBF = 30 days | Required MTTR |
|---|---|---|
| 99.0% | 30 days | 7.27 hours |
| 99.9% | 30 days | 43.2 minutes |
| 99.95% | 30 days | 21.6 minutes |
| 99.99% | 30 days | 4.32 minutes |
| 99.999% | 30 days | 25.9 seconds |

### Worked Examples

Target: 99.95% availability. Current MTBF = 14 days, MTTR = 45 minutes.

Current: $A = \frac{14 \times 24 \times 60}{14 \times 24 \times 60 + 45} = \frac{20160}{20205} = 99.78\%$

Gap: Need 99.95%, have 99.78%.

Path 1 (improve MTTF): Need $\text{MTBF} = \text{MTTR} \times A / (1-A) = 45 \times 0.9995/0.0005 = 89,955$ min $= 62.5$ days.

Path 2 (improve MTTR): Need $\text{MTTR} = \text{MTBF} \times (1-A)/A = 20160 \times 0.0005/0.9995 = 10.1$ min.

Path 2 requires reducing MTTR from 45 min to 10 min. Achievable with automated rollback and better detection. Path 1 requires doubling MTBF from 14 to 62.5 days — much harder.

---

## Prerequisites

- Basic probability and statistics
- Exponential distribution fundamentals
- Understanding of service architectures and dependencies
- Familiarity with SLI/SLO/SLA concepts

## Complexity

| Metric | Formula | Key Insight |
|---|---|---|
| Nines to downtime | $D = T \times 10^{-n}$ | Each nine is 10x harder |
| Burn rate | $\beta = e_w \cdot W / (w \cdot B)$ | Multi-window for accuracy |
| Serial availability | $\prod A_i$ | Multiplies failure risk |
| Little's Law | $L = \lambda W$ | Universal, distribution-free |
| M/M/1 latency | $W = 1/(\mu - \lambda)$ | Nonlinear near saturation |
| MTBF availability | $\text{MTBF}/(\text{MTBF}+\text{MTTR})$ | Reduce MTTR over increase MTTF |
