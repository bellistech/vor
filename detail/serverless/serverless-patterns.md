# The Mathematics of Serverless Patterns -- Reliability and Consistency Under Failure

> *In distributed systems, the question is never whether things will fail, but how gracefully you recover when they do.*

---

## 1. Idempotency Window (Duplicate Detection Probability)

### The Problem

At-least-once delivery means consumers may receive duplicate events. Given a
deduplication window of duration $W$ and a retry schedule, what is the
probability that a duplicate arrives outside the window and is processed twice?

### The Formula

If retries follow exponential backoff with base delay $d$ and maximum
$R$ retries, the time of the $k$-th retry is:

$$t_k = \sum_{i=0}^{k-1} d \cdot 2^i = d(2^k - 1)$$

A duplicate escapes the window if:

$$t_k > W$$

$$k > \log_2\left(\frac{W}{d} + 1\right)$$

Probability of duplicate escaping (assuming each retry has independent
probability $p_f$ of the consumer being unavailable):

$$P_{escape} = \prod_{i=0}^{k^*-1} p_f \cdot (1 - p_f)$$

where $k^* = \lfloor \log_2(W/d + 1) \rfloor$

More simply, if all retries within the window fail and a later one succeeds:

$$P_{escape} = p_f^{k^*} \cdot (1 - p_f)$$

### Worked Examples

**Example 1:** Window $W = 24$ hours = 86,400 s, base delay $d = 60$ s.
Maximum retries within window:

$$k^* = \lfloor \log_2(86400/60 + 1) \rfloor = \lfloor \log_2(1441) \rfloor = 10$$

If consumer fails 5% of the time ($p_f = 0.05$):

$$P_{escape} = 0.05^{10} \times 0.95 = 9.75 \times 10^{-14} \times 0.95 \approx 10^{-13}$$

Effectively impossible.

**Example 2:** Short window $W = 5$ minutes = 300 s, $d = 60$ s:

$$k^* = \lfloor \log_2(300/60 + 1) \rfloor = \lfloor \log_2(6) \rfloor = 2$$

$$P_{escape} = 0.05^2 \times 0.95 = 0.002375 = 0.24\%$$

With 100,000 events/day: $100{,}000 \times 0.0024 = 240$ potential duplicates
processed. The window must be long enough to cover the retry schedule.

## 2. Circuit Breaker State Machine (Failure Rate Thresholds)

### The Problem

A circuit breaker trips after $F$ failures in $N$ requests. What is the
probability of a false trip (tripping when the service is healthy) versus
the expected detection time (when the service is actually failing)?

### The Formula

For a healthy service with error rate $\epsilon$ (transient errors), the
probability of $F$ or more failures in $N$ requests follows the binomial:

$$P_{false\_trip} = \sum_{k=F}^{N} \binom{N}{k} \epsilon^k (1-\epsilon)^{N-k}$$

For a failed service with error rate $\delta$ (close to 1), the expected
number of requests to reach $F$ failures:

$$E[N_{detect}] = \frac{F}{\delta}$$

Detection time:

$$T_{detect} = \frac{E[N_{detect}]}{\lambda} = \frac{F}{\delta \lambda}$$

### Worked Examples

**Example 1:** Threshold $F = 5$ failures in $N = 10$ requests.
Healthy error rate $\epsilon = 0.01$:

$$P_{false\_trip} = \sum_{k=5}^{10} \binom{10}{k} 0.01^k \times 0.99^{10-k}$$

$$\approx \binom{10}{5} \times 0.01^5 \times 0.99^5 = 252 \times 10^{-10} \times 0.951 = 2.4 \times 10^{-8}$$

False trip probability: $2.4 \times 10^{-8}$ (essentially zero).

**Example 2:** Service is down ($\delta = 0.95$), request rate $\lambda = 10$/s:

$$T_{detect} = \frac{5}{0.95 \times 10} = 0.526 \text{ s}$$

Detection within half a second. Tradeoff: lower $F$ = faster detection
but higher false trip risk.

## 3. Saga Compensation Cost (Expected Rollback Overhead)

### The Problem

A saga with $n$ steps has a failure probability $p_i$ at each step. When step
$k$ fails, steps $1$ through $k-1$ must be compensated. What is the expected
compensation cost?

### The Formula

Probability of failing at step $k$ (all previous succeeded):

$$P(\text{fail at } k) = p_k \prod_{i=1}^{k-1}(1 - p_i)$$

Expected compensation operations:

$$E[\text{comp}] = \sum_{k=2}^{n} (k-1) \cdot p_k \prod_{i=1}^{k-1}(1-p_i)$$

For uniform failure probability $p$ at each step:

$$E[\text{comp}] = p \sum_{k=2}^{n} (k-1)(1-p)^{k-1}$$

Using the identity $\sum_{k=1}^{n} k \cdot x^k = \frac{x(1 - (n+1)x^n + nx^{n+1})}{(1-x)^2}$:

$$E[\text{comp}] = p \cdot \frac{(1-p)(1 - n(1-p)^{n-1} + (n-1)(1-p)^n)}{p^2}$$

### Worked Examples

**Example 1:** 5-step saga, each step fails with $p = 0.02$:

$$P(\text{success}) = (1 - 0.02)^5 = 0.98^5 = 0.904$$

$$E[\text{comp}] = 0.02 \times \sum_{k=2}^{5} (k-1) \times 0.98^{k-1}$$

$$= 0.02 \times (0.98 + 2 \times 0.96 + 3 \times 0.941 + 4 \times 0.922)$$

$$= 0.02 \times (0.98 + 1.92 + 2.824 + 3.688) = 0.02 \times 9.412 = 0.188$$

On average, 0.19 compensation operations per saga execution.

**Example 2:** 10-step saga, $p = 0.05$ (less reliable services):

$$P(\text{success}) = 0.95^{10} = 0.599$$

$$E[\text{comp}] = 0.05 \times \sum_{k=2}^{10} (k-1) \times 0.95^{k-1} \approx 0.05 \times 25.7 = 1.29$$

About 1.3 compensations per saga. With a 40% failure rate, compensations
become a significant fraction of total work.

## 4. Fan-Out Completion Time (Parallel Execution with Stragglers)

### The Problem

Fan-out distributes work to $N$ parallel Lambda invocations. The fan-in waits
for all to complete. What is the expected completion time when individual
execution times have variance?

### The Formula

If each invocation has execution time drawn from distribution $F(t)$ with
CDF $F$, the completion time is the maximum of $N$ independent draws:

$$T_{fanout} = \max(T_1, T_2, \ldots, T_N)$$

For exponentially distributed execution times with mean $\mu$:

$$E[T_{max}] = \mu \cdot H_N = \mu \sum_{k=1}^{N} \frac{1}{k}$$

where $H_N$ is the $N$-th harmonic number.

For log-normally distributed times (common in practice):

$$E[T_{max}] \approx e^{\mu_{\ln} + \sigma_{\ln} \Phi^{-1}(1 - 1/N)}$$

where $\Phi^{-1}$ is the inverse standard normal CDF.

### Worked Examples

**Example 1:** 100 parallel invocations, exponential with mean 500 ms:

$$E[T_{max}] = 500 \times H_{100} = 500 \times 5.187 = 2{,}594 \text{ ms}$$

The slowest of 100 takes ~5x the average. This is the straggler problem.

**Example 2:** Same but log-normal with $\mu_{\ln} = 6.2$ (median 493 ms),
$\sigma_{\ln} = 0.5$:

$$\Phi^{-1}(1 - 1/100) = \Phi^{-1}(0.99) = 2.326$$

$$E[T_{max}] \approx e^{6.2 + 0.5 \times 2.326} = e^{7.363} = 1{,}578 \text{ ms}$$

Log-normal with moderate variance is less affected by stragglers.

## 5. Step Functions Cost Model (State Transitions vs Direct Invocation)

### The Problem

Step Functions charge per state transition. When is orchestration cheaper
than direct Lambda-to-Lambda invocation?

### The Formula

Step Functions cost for a workflow with $S$ states:

$$C_{sf} = S \times P_{transition} + \sum_{i=1}^{S} C_{lambda_i}$$

Direct Lambda chain:

$$C_{direct} = \sum_{i=1}^{S} C_{lambda_i} + (S-1) \times T_{wait} \times \mathit{mem} \times P_{compute}$$

where $T_{wait}$ is the time each Lambda spends waiting for the next one
(relevant for synchronous chains).

Break-even:

$$S \times P_{transition} = (S-1) \times T_{wait} \times \mathit{mem} \times P_{compute}$$

### Worked Examples

**Example 1:** 5-step workflow. $P_{transition} = \$0.000025$,
$T_{wait} = 0$ (async, fire-and-forget):

$$C_{sf} = 5 \times 0.000025 = \$0.000125$$

$$C_{direct} = 0 \text{ (no waiting cost for async)}$$

Step Functions costs $0.000125 per execution. At 1M executions/month:
$125/month just for orchestration.

**Example 2:** Synchronous chain, each step waits 2 seconds for the next,
mem = 256 MB:

$$C_{wait} = 4 \times 2 \times 0.256 \times 0.0000166667 = \$0.0000341$$

$$C_{sf} = \$0.000125$$

Step Functions is 3.7x more expensive but provides retry logic, error
handling, and visibility that you would otherwise build yourself.

## 6. Event Ordering Probability (Out-of-Order Delivery)

### The Problem

When events are delivered via webhooks or queues without ordering guarantees,
what is the probability that events arrive out of order, and how does this
affect state consistency?

### The Formula

If two events are produced $\Delta t$ apart and delivery times are
independent with variance $\sigma^2$, the probability of inversion:

$$P_{inversion} = \Phi\left(\frac{-\Delta t}{\sigma\sqrt{2}}\right)$$

where $\Phi$ is the standard normal CDF.

For $n$ events in sequence, the expected number of inversions:

$$E[\text{inversions}] = \sum_{i=1}^{n-1} \Phi\left(\frac{-\Delta t_i}{\sigma\sqrt{2}}\right)$$

### Worked Examples

**Example 1:** Events 100 ms apart, delivery jitter $\sigma = 50$ ms:

$$P_{inversion} = \Phi\left(\frac{-100}{50\sqrt{2}}\right) = \Phi(-1.414) = 0.079 = 7.9\%$$

For 1,000 sequential events: $999 \times 0.079 = 79$ expected inversions.

**Example 2:** Events 1 second apart, same jitter:

$$P_{inversion} = \Phi\left(\frac{-1000}{70.7}\right) = \Phi(-14.14) \approx 0$$

With sufficient spacing relative to jitter, inversions become negligible.
This justifies using timestamps for ordering rather than relying on delivery order.

## Prerequisites

- Probability distributions (Poisson, binomial, exponential, log-normal)
- Order statistics (distribution of max of $N$ random variables)
- Markov chains (circuit breaker state transitions)
- Combinatorics (binomial coefficients for failure counting)
- Harmonic series and approximations
- Cost optimization (break-even analysis, marginal cost)
- Distributed systems consistency models (eventual consistency, idempotency)
