# The Mathematics of Lambda -- Cold Start Probability and Invocation Cost Optimization

> *In a world where you pay per millisecond, the difference between 128 MB and 256 MB is not memory -- it is time.*

---

## 1. Cold Start Probability (Container Lifecycle)

### The Problem

Lambda maintains warm containers for a window after each invocation. Given an
invocation rate and warm window duration, what is the probability that a given
invocation hits a cold start?

### The Formula

Model invocations as a Poisson process with rate $\lambda$ (requests/second).
The warm window is $W$ seconds (typically 5-15 minutes). A cold start occurs
when no invocation arrived during the previous $W$ seconds.

Probability of cold start:

$$P_{cold} = P(\text{no arrival in } W) = e^{-\lambda W}$$

Expected cold starts per hour:

$$N_{cold} = 3600\lambda \cdot e^{-\lambda W}$$

### Worked Examples

**Example 1:** 1 request per minute ($\lambda = 1/60$), warm window $W = 600$ s (10 min):

$$P_{cold} = e^{-\frac{1}{60} \times 600} = e^{-10} = 0.0000454 = 0.005\%$$

Cold starts per hour: $60 \times 0.0000454 = 0.003$ (essentially never).

**Example 2:** 1 request per hour ($\lambda = 1/3600$), same warm window:

$$P_{cold} = e^{-\frac{1}{3600} \times 600} = e^{-0.167} = 0.846 = 84.6\%$$

Cold starts per hour: $1 \times 0.846 = 0.85$ (nearly every invocation is cold).

**Example 3:** Critical threshold -- what rate eliminates 99% of cold starts?

$$0.01 = e^{-\lambda \times 600}$$

$$\lambda = \frac{-\ln(0.01)}{600} = \frac{4.605}{600} = 0.00767 \text{ req/s} = 0.46 \text{ req/min}$$

Approximately 1 request every 2 minutes is sufficient.

## 2. Memory-CPU Scaling (Performance-Cost Tradeoff)

### The Problem

Lambda allocates CPU proportionally to memory. At 1,769 MB, you get 1 full
vCPU. How does memory allocation affect execution time and cost?

### The Formula

CPU allocation:

$$CPU(\mathit{mem}) = \frac{\mathit{mem}}{1769}$$

For CPU-bound workloads, execution time scales inversely with CPU:

$$T(\mathit{mem}) = T_{base} \times \frac{1769}{\mathit{mem}}$$

Cost per invocation:

$$C(\mathit{mem}) = \mathit{mem} \times T(\mathit{mem}) \times P_{per\_GB\_ms}$$

For CPU-bound: $C(\mathit{mem}) = \mathit{mem} \times T_{base} \times \frac{1769}{\mathit{mem}} \times P = T_{base} \times 1769 \times P$

This is constant! For purely CPU-bound workloads, cost is independent of
memory allocation.

For I/O-bound workloads, execution time has a floor:

$$T(\mathit{mem}) = \max\left(T_{io}, T_{base} \times \frac{1769}{\mathit{mem}}\right)$$

### Worked Examples

**Example 1:** CPU-bound function takes 10 seconds at 128 MB. Time at 1769 MB:

$$T(1769) = 10 \times \frac{1769}{1769} = 10 \text{ s... wait}$$

$$T(1769) = 10 \times \frac{128}{1769} = 0.724 \text{ s}$$

Cost comparison ($P = \$0.0000166667$ per GB-second):

$$C(128) = 0.128 \times 10 \times 0.0000166667 = \$0.0000213$$

$$C(1769) = 1.769 \times 0.724 \times 0.0000166667 = \$0.0000213$$

Identical cost but 13.8x faster. Always maximize memory for CPU-bound work.

**Example 2:** I/O-bound function waiting 5 seconds for an API call, 200 ms CPU:

$$T(128) = \max(5, 0.2 \times \frac{1769}{128}) = \max(5, 2.76) = 5 \text{ s}$$

$$T(1769) = \max(5, 0.2) = 5 \text{ s}$$

$$C(128) = 0.128 \times 5 \times P = 0.64 \cdot P$$

$$C(1769) = 1.769 \times 5 \times P = 8.845 \cdot P$$

For I/O-bound: use minimum memory to minimize cost ($13.8\times$ cheaper at 128 MB).

## 3. Concurrency and Throttling (Little's Law)

### The Problem

Lambda has an account-level concurrency limit (default 1,000). Given request
rate and duration, will the function be throttled?

### The Formula

Little's Law gives the average concurrent executions:

$$L = \lambda \times W$$

where $\lambda$ is the arrival rate (req/s) and $W$ is the average execution
duration (seconds).

Throttling occurs when:

$$L > C_{limit}$$

$$\lambda_{max} = \frac{C_{limit}}{W}$$

With reserved concurrency $C_r$ for a specific function:

$$\lambda_{max} = \frac{C_r}{W}$$

Burst capacity: Lambda allows an initial burst of 3,000 concurrent executions,
then scales at 500/minute.

### Worked Examples

**Example 1:** 100 req/s, each taking 2 seconds:

$$L = 100 \times 2 = 200 \text{ concurrent executions}$$

Well within 1,000 limit. Maximum rate before throttling:

$$\lambda_{max} = \frac{1000}{2} = 500 \text{ req/s}$$

**Example 2:** Batch processing: 10,000 SQS messages, each taking 30 seconds:

$$L = \frac{10{,}000}{30} \times 30 = 10{,}000 \text{ ... not quite}$$

With SQS batch size 10 and 5 concurrent pollers initially:

Concurrent = min(messages/batch_size, concurrency_limit) = min(1000, 1000) = 1000

But burst limit applies: starts at min(3000, messages) and scales.

## 4. Cost Optimization (Provisioned vs On-Demand)

### The Problem

Provisioned concurrency eliminates cold starts but costs more. At what
invocation rate does provisioned concurrency become cost-effective?

### The Formula

On-demand cost per hour:

$$C_{od} = N \times (T_{warm} \times (1 - P_{cold}) + (T_{warm} + T_{cold}) \times P_{cold}) \times \mathit{mem} \times P_{compute}$$

Provisioned cost per hour (always-on):

$$C_{prov} = C_{od}(P_{cold}=0) + n_{prov} \times 3600 \times \mathit{mem} \times P_{provisioned}$$

Break-even when:

$$C_{od} = C_{prov}$$

$$N \times P_{cold} \times T_{cold} \times \mathit{mem} \times P_{compute} = n_{prov} \times 3600 \times \mathit{mem} \times P_{provisioned}$$

### Worked Examples

**Example 1:** $N = 100$/hour, $T_{cold} = 2$ s, $P_{cold} = 0.1$, mem = 512 MB,
$P_{compute} = \$0.0000166667$/GB-s, $P_{provisioned} = \$0.0000041667$/GB-s:

Cold start penalty cost per hour:

$$C_{penalty} = 100 \times 0.1 \times 2 \times 0.512 \times 0.0000166667 = \$0.000171$$

Provisioned cost per hour (1 instance):

$$C_{prov} = 1 \times 3600 \times 0.512 \times 0.0000041667 = \$0.00768$$

Provisioned is 45x more expensive than the cold start penalty. Not worth it.

**Example 2:** $N = 10{,}000$/hour, $P_{cold} = 0.3$, $T_{cold} = 5$ s (Java):

$$C_{penalty} = 10{,}000 \times 0.3 \times 5 \times 0.512 \times 0.0000166667 = \$0.128/\text{hour}$$

$$C_{prov}(10) = 10 \times 3600 \times 0.512 \times 0.0000041667 = \$0.0768/\text{hour}$$

Provisioned concurrency saves $0.051/hour ($37/month). Worth it for latency-sensitive Java functions.

## 5. Event Source Mapping (Batch Size Optimization)

### The Problem

SQS and DynamoDB Streams use batch processing. What batch size minimizes cost
while meeting latency requirements?

### The Formula

Cost per message with batch size $B$:

$$C_{msg} = \frac{T_{fixed} + B \times T_{per\_msg}}{B} \times \mathit{mem} \times P$$

$$C_{msg} = \left(\frac{T_{fixed}}{B} + T_{per\_msg}\right) \times \mathit{mem} \times P$$

As $B \to \infty$: $C_{msg} \to T_{per\_msg} \times \mathit{mem} \times P$

Latency per message (worst case, last message in batch waits for batch window):

$$L_{msg} = W_{batch} + T_{fixed} + B \times T_{per\_msg}$$

### Worked Examples

**Example 1:** $T_{fixed} = 50$ ms (init), $T_{per\_msg} = 10$ ms, mem = 256 MB:

At $B = 1$: $C_{msg} = (50 + 10) \times 0.256 \times P = 15.36 \times P$

At $B = 10$: $C_{msg} = (5 + 10) \times 0.256 \times P = 3.84 \times P$

At $B = 100$: $C_{msg} = (0.5 + 10) \times 0.256 \times P = 2.688 \times P$

Cost reduction from $B=1$ to $B=10$: $\frac{15.36 - 3.84}{15.36} = 75\%$

**Example 2:** Latency constraint of 5 seconds, batch window = 1 second:

$$5 = 1 + 0.05 + B \times 0.01$$

$$B_{max} = \frac{5 - 1.05}{0.01} = 395$$

Optimal batch size: $B = \min(395, B_{max\_lambda}) = 10$ (SQS limit)

## 6. ARM vs x86 Cost Efficiency

### The Problem

Graviton2 (ARM64) Lambda is 20% cheaper per GB-second. When combined with
potential performance differences, what is the total cost impact?

### The Formula

$$C_{arm} = T_{arm} \times \mathit{mem} \times 0.8P$$

$$C_{x86} = T_{x86} \times \mathit{mem} \times P$$

Break-even performance ratio:

$$\frac{T_{arm}}{T_{x86}} = \frac{1}{0.8} = 1.25$$

ARM is cheaper unless it is more than 25% slower than x86.

### Worked Examples

**Example 1:** Function runs in 100 ms on both architectures:

$$\text{Savings} = 1 - 0.8 = 20\%$$

**Example 2:** Function runs 10% slower on ARM (110 ms vs 100 ms):

$$C_{arm} = 110 \times 0.8 = 88$$

$$C_{x86} = 100 \times 1.0 = 100$$

$$\text{Savings} = 1 - \frac{88}{100} = 12\%$$

Still 12% cheaper on ARM despite being slower.

## Prerequisites

- Poisson processes and exponential inter-arrival times
- Little's Law ($L = \lambda W$) for concurrency modeling
- Microeconomics of cloud pricing (per-millisecond billing granularity)
- CPU scheduling and proportional share allocation
- Queueing theory (M/M/c queues for concurrent Lambda modeling)
- Batch processing optimization (fixed vs marginal cost decomposition)
- Cold start mechanics (container initialization, JVM warmup, runtime loading)
