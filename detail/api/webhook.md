# The Mathematics of Webhooks -- Delivery Reliability and Retry Optimization

> *A webhook that fires once and forgets is a notification. A webhook that retries until acknowledged is a contract.*

---

## 1. Retry Schedule Optimization (Exponential Backoff Analysis)

### The Problem

A webhook provider retries failed deliveries with exponential backoff. Given
a base delay $d$, backoff multiplier $b$, maximum retries $R$, and per-attempt
success probability $p$, what is the expected delivery time and the probability
of eventual success?

### The Formula

Delay before the $k$-th retry:

$$\Delta_k = d \cdot b^{k-1}$$

Total time to the $k$-th retry:

$$T_k = d \sum_{i=0}^{k-1} b^i = d \cdot \frac{b^k - 1}{b - 1}$$

Probability of success by the $k$-th attempt (assuming independent failures):

$$P(\text{success by } k) = 1 - (1-p)^k$$

Probability of total failure (all $R+1$ attempts fail):

$$P_{fail} = (1-p)^{R+1}$$

Expected delivery time (conditioned on success):

$$E[T | \text{success}] = \frac{\sum_{k=0}^{R} T_k \cdot p(1-p)^k}{1 - (1-p)^{R+1}}$$

### Worked Examples

**Example 1:** $d = 60$ s, $b = 2$, $R = 5$ retries, $p = 0.9$ success rate.

Total time to last retry:

$$T_5 = 60 \times \frac{2^5 - 1}{2 - 1} = 60 \times 31 = 1{,}860 \text{ s} = 31 \text{ min}$$

Probability of total failure:

$$P_{fail} = (1 - 0.9)^6 = 0.1^6 = 10^{-6}$$

One in a million deliveries fail. Expected delivery time:

$$E[T] = \frac{0 \times 0.9 + 60 \times 0.09 + 180 \times 0.009 + \ldots}{1 - 10^{-6}}$$

$$\approx 0 + 5.4 + 1.62 + 0.42 + 0.10 + 0.02 = 7.56 \text{ s}$$

Most deliveries succeed on the first try, keeping expected time low.

**Example 2:** Degraded consumer ($p = 0.3$), same schedule:

$$P_{fail} = 0.7^6 = 0.118 = 11.8\%$$

$$E[T] = \frac{0(0.3) + 60(0.21) + 180(0.147) + 420(0.103) + 900(0.072) + 1860(0.050)}{0.882}$$

$$= \frac{0 + 12.6 + 26.5 + 43.3 + 64.8 + 93.0}{0.882} = \frac{240.2}{0.882} = 272.3 \text{ s} \approx 4.5 \text{ min}$$

## 2. HMAC Verification Cost (Timing Attack Resistance)

### The Problem

HMAC signature verification must be constant-time to prevent timing attacks.
What is the information leakage from non-constant-time comparison, and
how many requests does an attacker need to exploit it?

### The Formula

For a byte-by-byte comparison that short-circuits on the first mismatch,
the timing difference between a correct and incorrect byte:

$$\Delta t = T_{compare} \approx 1\text{--}10 \text{ ns per byte}$$

An attacker guessing byte $i$ of an $n$-byte HMAC observes:

$$T(\text{correct prefix of length } i) = T_{base} + i \cdot \Delta t$$

Bytes to crack sequentially: for each of $n$ bytes, try all 256 values.
Total attempts:

$$A_{timing} = 256 \times n$$

For HMAC-SHA256 ($n = 32$ bytes):

$$A_{timing} = 256 \times 32 = 8{,}192 \text{ requests}$$

Versus brute force:

$$A_{brute} = 256^{32} = 2^{256} \approx 1.16 \times 10^{77}$$

The timing attack reduces the search space by a factor of:

$$\frac{A_{brute}}{A_{timing}} = \frac{2^{256}}{8{,}192} = 2^{243}$$

With constant-time comparison (`hmac.compare_digest`), $\Delta t = 0$ and
the attack is impossible.

### Worked Examples

**Example 1:** Attacker sends 8,192 requests over network with 1 ms jitter.
Signal: 5 ns timing difference. Signal-to-noise ratio:

$$SNR = \frac{5 \times 10^{-9}}{1 \times 10^{-3}} = 5 \times 10^{-6}$$

To detect with confidence, need $N$ samples per byte:

$$N = \left(\frac{z \cdot \sigma}{\Delta t}\right)^2 = \left(\frac{3 \times 10^{-3}}{5 \times 10^{-9}}\right)^2 = 3.6 \times 10^{11}$$

Infeasible over network. But on localhost ($\sigma = 100$ ns):

$$N = \left(\frac{3 \times 100 \times 10^{-9}}{5 \times 10^{-9}}\right)^2 = 3{,}600$$

Total requests: $3{,}600 \times 256 \times 32 = 29{,}491{,}200$ -- feasible in hours.

**Example 2:** Constant-time comparison (`hmac.compare_digest`):

$$\Delta t = 0 \quad \Rightarrow \quad SNR = 0 \quad \Rightarrow \quad N = \infty$$

No amount of sampling reveals information. Always use constant-time comparison.

## 3. Delivery Probability Over Time (Survival Analysis)

### The Problem

Given a retry schedule and time-varying consumer availability, what is the
probability of successful delivery as a function of elapsed time?

### The Formula

Model consumer availability as a function $a(t) \in [0, 1]$ representing the
probability the consumer is healthy at time $t$.

The survival function (probability of NOT having been delivered by time $t$):

$$S(t) = \prod_{k: T_k \leq t} (1 - a(T_k))$$

The delivery probability by time $t$:

$$D(t) = 1 - S(t) = 1 - \prod_{k: T_k \leq t} (1 - a(T_k))$$

For constant availability $a$:

$$D(t) = 1 - (1-a)^{|\{k : T_k \leq t\}|}$$

### Worked Examples

**Example 1:** Consumer has 99% availability. Retry schedule: 0, 1, 5, 30, 120,
600 minutes. Delivery probability after each attempt:

| Attempt $k$ | Time (min) | $D(T_k)$ |
|-------------|------------|-----------|
| 0 | 0 | $1 - 0.01^1 = 0.990$ |
| 1 | 1 | $1 - 0.01^2 = 0.9999$ |
| 2 | 5 | $1 - 0.01^3 = 0.999999$ |
| 3 | 30 | $1 - 0.01^4 \approx 1$ |

After just 2 retries, delivery is virtually certain.

**Example 2:** Consumer experiences a 4-hour outage starting at $t = 0$.
$a(t) = 0$ for $t < 240$ min, $a(t) = 0.95$ for $t \geq 240$:

| Attempt | Time (min) | $a(T_k)$ | Running $S$ |
|---------|-----------|-----------|-------------|
| 0 | 0 | 0 | 1.0 |
| 1 | 1 | 0 | 1.0 |
| 2 | 5 | 0 | 1.0 |
| 3 | 30 | 0 | 1.0 |
| 4 | 120 | 0 | 1.0 |
| 5 | 600 | 0.95 | 0.05 |

$$D(600) = 1 - 0.05 = 0.95$$

The event is delivered on the 6th attempt (after 10 hours) with 95% probability.
This shows why providers like Stripe retry for up to 3 days.

## 4. Idempotency Window Sizing (Storage vs Risk Tradeoff)

### The Problem

Idempotency keys must be stored long enough to deduplicate all retries. How
large must the storage be, and what is the risk of evicting a key too early?

### The Formula

Storage required for idempotency records with TTL $W$, event rate $\lambda$:

$$M = \lambda \cdot W \cdot S_{record}$$

where $S_{record}$ is the size per idempotency record.

Risk of premature eviction (key expires before last retry):

$$P_{premature} = P(T_{last\_retry} > W) = \begin{cases} 0 & \text{if } W \geq T_R \\ 1 & \text{if } W < T_R \end{cases}$$

For variable retry schedules with jitter:

$$P_{premature} = P\left(\sum_{k=0}^{R} (\Delta_k + J_k) > W\right)$$

where $J_k$ is the jitter added to each retry.

### Worked Examples

**Example 1:** 10,000 events/hour, 24-hour TTL, 256 bytes per record:

$$M = 10{,}000 \times 24 \times 256 = 61{,}440{,}000 \text{ bytes} = 58.6 \text{ MB}$$

**Example 2:** Stripe-style retries (up to 3 days). Safe TTL:

$$W_{safe} = 3 \times 24 \times 3600 + \text{margin} = 259{,}200 + 86{,}400 = 345{,}600 \text{ s} = 4 \text{ days}$$

At 100,000 events/hour:

$$M = 100{,}000 \times 96 \times 256 = 2.46 \text{ GB}$$

With DynamoDB at $1.25/GB/month, this costs $3.07/month. Cheap insurance.

## 5. Payload Size Economics (Fat vs Thin Payloads)

### The Problem

Should webhook payloads include the full event data ("fat") or just an ID
requiring the consumer to fetch the data ("thin")? What are the bandwidth
and latency tradeoffs?

### The Formula

**Fat payload** cost per event:

$$C_{fat} = S_{payload} \times C_{bandwidth} + T_{serialize} \times C_{compute}$$

**Thin payload** cost per event:

$$C_{thin} = S_{id} \times C_{bandwidth} + T_{api\_call} \times C_{compute} + T_{rtt}$$

The break-even payload size:

$$S_{break} = S_{id} + \frac{T_{api\_call} \times C_{compute} + T_{rtt} \times C_{compute}}{C_{bandwidth}}$$

Consumer-side latency comparison:

$$L_{fat} = T_{delivery} + T_{parse}$$

$$L_{thin} = T_{delivery} + T_{parse\_id} + T_{rtt\_api} + T_{parse\_response}$$

### Worked Examples

**Example 1:** Fat payload (2 KB) vs thin (100 bytes + API call with 50 ms RTT).
At 1M events/day:

$$BW_{fat} = 10^6 \times 2{,}048 = 2.048 \text{ GB/day}$$

$$BW_{thin} = 10^6 \times 100 = 100 \text{ MB/day (delivery)} + 10^6 \times 2{,}048 = 2.048 \text{ GB/day (API)}$$

Total thin bandwidth is actually higher (2.148 GB) because the consumer still
fetches the data, plus the overhead of an HTTP request/response cycle.

**Example 2:** Consumer latency at p99 ($T_{delivery} = 100$ ms):

$$L_{fat} = 100 + 1 = 101 \text{ ms}$$

$$L_{thin} = 100 + 0.1 + 50 + 1 = 151.1 \text{ ms}$$

Fat payloads are 33% faster for consumers but increase provider bandwidth.
The right choice depends on whether the consumer always needs the full data.

## 6. Dead Letter Queue Accumulation (Failure Modeling)

### The Problem

When a consumer endpoint is down for an extended period, events accumulate
in the dead letter queue. What is the DLQ growth rate and the time to
process the backlog when the consumer recovers?

### The Formula

DLQ growth rate during outage:

$$\frac{dQ}{dt} = \lambda \cdot P_{exhaust}(t)$$

where $P_{exhaust}(t)$ is the probability an event has exhausted all retries
by time $t$.

For events produced during the outage window $[0, D]$:

$$Q(D) = \int_0^D \lambda \cdot P_{exhaust}(D - \tau) \, d\tau$$

If the retry schedule completes in time $T_R$, then $P_{exhaust}(t) = 1$ for
$t > T_R$:

$$Q(D) = \lambda \cdot \max(0, D - T_R)$$

Recovery time (processing backlog at rate $\mu$):

$$T_{recovery} = \frac{Q(D)}{\mu - \lambda}$$

(Requires $\mu > \lambda$ to drain the backlog.)

### Worked Examples

**Example 1:** 100 events/min, retry schedule completes in 30 min, outage lasts
2 hours:

$$Q(120) = 100 \times (120 - 30) = 9{,}000 \text{ events}$$

If consumer processes at 200 events/min post-recovery:

$$T_{recovery} = \frac{9{,}000}{200 - 100} = 90 \text{ minutes}$$

Total impact duration: 2 hours (outage) + 1.5 hours (recovery) = 3.5 hours.

**Example 2:** Same scenario but consumer can only process 120 events/min:

$$T_{recovery} = \frac{9{,}000}{120 - 100} = 450 \text{ minutes} = 7.5 \text{ hours}$$

With only 20% headroom, recovery takes much longer. This motivates
provisioning consumers with at least 2x burst capacity.

## Prerequisites

- Geometric series and exponential backoff analysis
- Cryptographic hash functions and HMAC construction
- Timing attack theory and constant-time comparison
- Survival analysis and reliability theory
- Queueing theory (arrival rates, service rates, backlog draining)
- Information theory (bits of security, brute force complexity)
- Cost-benefit analysis for system design tradeoffs
