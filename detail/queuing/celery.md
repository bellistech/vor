# The Mathematics of Celery -- Queuing Theory and Distributed Task Processing

> *A distributed task queue is fundamentally a queuing system: tasks arrive, wait in a buffer, and are served by workers. Understanding the mathematics of arrival rates, service times, and worker pools lets you right-size infrastructure, predict latency, and avoid queue saturation before it happens.*

---

## 1. Single-Queue Model (M/M/c Fundamentals)

### The Problem
A Celery deployment with $c$ workers consuming from a single queue. Tasks arrive as a Poisson process and have exponentially distributed durations. What is the expected wait time?

### The Formula
The Erlang-C formula gives the probability that an arriving task must wait:

$$P_W = \frac{\frac{(c\rho)^c}{c!} \cdot \frac{1}{1-\rho}}{\sum_{k=0}^{c-1} \frac{(c\rho)^k}{k!} + \frac{(c\rho)^c}{c!} \cdot \frac{1}{1-\rho}}$$

where $\rho = \lambda / (c\mu)$ is the utilization, $\lambda$ is the arrival rate, and $\mu = 1/\bar{w}$ is the service rate.

The expected time in queue:

$$W_q = \frac{P_W}{c\mu(1 - \rho)}$$

The expected total time (queue + service):

$$W = W_q + \frac{1}{\mu}$$

### Worked Examples
$\lambda = 20$ tasks/sec, $\bar{w} = 0.5$ sec ($\mu = 2$), $c = 12$ workers:

$$\rho = \frac{20}{12 \times 2} = 0.833$$

Computing $P_W$ for these values gives $P_W \approx 0.42$. Then:

$$W_q = \frac{0.42}{12 \times 2 \times 0.167} = \frac{0.42}{4.0} = 0.105 \text{ sec}$$

$$W = 0.105 + 0.5 = 0.605 \text{ sec}$$

At 83% utilization, average task latency is about 605 ms. Increasing to $c = 16$ drops $\rho$ to 0.625 and $W_q$ to under 20 ms.

---

## 2. Prefetch and Batching (Worker Buffer Analysis)

### The Problem
Celery's `worker_prefetch_multiplier` controls how many tasks a worker prefetches from the broker. A high multiplier improves throughput but increases head-of-line blocking. What is the optimal setting?

### The Formula
With prefetch multiplier $m$, each worker buffers up to $m$ tasks. The effective queue depth visible to other workers shrinks. If $n$ tasks are in the broker queue and $c$ workers each prefetch $m$:

$$n_{\text{visible}} = \max(0, \; n - c \cdot m)$$

The head-of-line blocking delay for a newly prefetched task:

$$D_{\text{hol}} = (m - 1) \cdot \bar{w}$$

The throughput benefit from prefetching (amortizing broker round-trips of cost $r$):

$$\text{Throughput gain} = \frac{m \cdot \bar{w}}{m \cdot \bar{w} + r} \bigg/ \frac{\bar{w}}{\bar{w} + r} = \frac{m(\bar{w} + r)}{m \cdot \bar{w} + r}$$

### Worked Examples
With $\bar{w} = 100$ ms, broker round-trip $r = 5$ ms, $m = 4$:

$$D_{\text{hol}} = 3 \times 100 = 300 \text{ ms}$$

$$\text{Gain} = \frac{4(100 + 5)}{4 \times 100 + 5} = \frac{420}{405} = 1.037$$

Only 3.7% throughput improvement but 300 ms worst-case added latency. For latency-sensitive work, $m = 1$ is correct. For batch processing with $\bar{w} = 5$ ms and $r = 5$ ms:

$$\text{Gain} = \frac{4(5 + 5)}{4 \times 5 + 5} = \frac{40}{25} = 1.6$$

60% throughput gain -- prefetching is valuable when task duration approaches broker latency.

---

## 3. Retry Backoff (Exponential Delay Analysis)

### The Problem
Tasks fail transiently and are retried with exponential backoff. How long until a task either succeeds or exhausts its retries? What is the expected number of attempts?

### The Formula
With per-attempt failure probability $p$, max retries $R$, and base delay $d$, the delay before retry $k$ is:

$$\Delta_k = d \cdot b^k$$

where $b$ is the backoff base (typically 2). The total expected time to success (given success within $R$ retries):

$$E[T] = \bar{w} + \sum_{k=1}^{R} p^k \left(\Delta_k + \bar{w}\right) \cdot \prod_{j=0}^{k-1} p$$

Simplified for geometric success:

$$E[T] = \frac{\bar{w}}{1 - p} + \frac{p \cdot d(1 - (pb)^R)}{1 - pb}$$

The probability of ultimate failure (exhausting all retries):

$$P_{\text{fail}} = p^{R+1}$$

### Worked Examples
$p = 0.1$ (10% failure rate), $R = 3$, $d = 60$ sec, $b = 2$, $\bar{w} = 5$ sec:

$$P_{\text{fail}} = 0.1^4 = 0.0001 \text{ (0.01%)}$$

Expected retries: $\sum_{k=1}^{3} p^k = 0.1 + 0.01 + 0.001 = 0.111$

Total expected retry delay contribution:

$$0.1 \times 60 + 0.01 \times 120 + 0.001 \times 240 = 6 + 1.2 + 0.24 = 7.44 \text{ sec}$$

Expected total time: $5 / 0.9 + 7.44 \approx 13.0$ sec.

With $p = 0.5$ (unreliable service): $P_{\text{fail}} = 0.5^4 = 6.25\%$ -- consider circuit breaking.

---

## 4. Chord Synchronization (Fork-Join Model)

### The Problem
A chord dispatches $n$ tasks in parallel and waits for all to complete before running a callback. What is the expected completion time when individual task durations are random?

### The Formula
For $n$ i.i.d. tasks with CDF $F(t)$ and PDF $f(t)$, the CDF of the maximum (last to finish):

$$F_{\max}(t) = [F(t)]^n$$

For exponentially distributed tasks ($\mu$):

$$E[\max(X_1, \ldots, X_n)] = \frac{1}{\mu} \sum_{k=1}^{n} \frac{1}{k} = \frac{H_n}{\mu}$$

where $H_n$ is the $n$-th harmonic number, $H_n \approx \ln(n) + \gamma$ (Euler-Mascheroni constant $\gamma \approx 0.5772$).

### Worked Examples
A chord of $n = 100$ tasks, each with mean duration $\bar{w} = 2$ sec (exponential):

$$E[\text{chord time}] = 2 \times H_{100} = 2 \times (\ln 100 + 0.5772) \approx 2 \times 5.187 = 10.37 \text{ sec}$$

The slowest of 100 tasks takes about 5x the mean. For $n = 1000$:

$$E[\text{chord time}] = 2 \times H_{1000} \approx 2 \times 7.485 = 14.97 \text{ sec}$$

This logarithmic growth means doubling the chord size adds only $2 \ln 2 \approx 1.4$ seconds.

---

## 5. Rate Limiting (Token Bucket Model)

### The Problem
Celery's rate limiting (`rate_limit='100/m'`) throttles task execution. What is the effective throughput and latency when the arrival rate exceeds the rate limit?

### The Formula
A token bucket with rate $r$ tokens/sec and bucket capacity $B$. If arrival rate $\lambda > r$, a queue builds. The delay for the $k$-th task arriving in a burst of $n$:

$$D_k = \max\left(0, \; \frac{k - B}{r}\right)$$

The steady-state queue length when $\lambda > r$:

$$L = \frac{\lambda - r}{r} \cdot B + (\lambda - r) \cdot t$$

This grows linearly -- rate limiting without backpressure causes unbounded queues.

The effective throughput:

$$\Theta_{\text{eff}} = \min(\lambda, r)$$

### Worked Examples
Rate limit $r = 100/\text{min} \approx 1.67/\text{sec}$, bucket $B = 10$, burst of $n = 50$ tasks arriving instantly:

- First 10 tasks: served immediately (bucket drains)
- Task 11: delayed $1/1.67 = 0.6$ sec
- Task 50: delayed $(50 - 10)/1.67 = 24$ sec

If sustained $\lambda = 3$/sec against $r = 1.67$/sec, the queue grows at 1.33 tasks/sec. After 10 minutes: $L = 1.33 \times 600 = 800$ tasks backed up. Backpressure or producer throttling is essential.

---

## 6. Concurrency Models (Amdahl's Law Applied)

### The Problem
Celery supports prefork (multiprocess), eventlet/gevent (green threads), and solo pools. For a task with fraction $f$ spent in I/O, which pool maximizes throughput?

### The Formula
For CPU-bound fraction $s = 1 - f$ (serial/compute portion), Amdahl's Law gives the speedup with $c$ workers:

$$S(c) = \frac{1}{s + \frac{f}{c}}$$

For prefork with $c$ processes, effective throughput on CPU-bound work:

$$\Theta_{\text{prefork}} = \frac{c}{\bar{w}}$$

For eventlet/gevent with $c$ green threads, I/O-bound speedup:

$$S_{\text{green}}(c) = \frac{1}{s + \frac{f}{c}} \approx \frac{c}{1 + (c-1) \cdot s}$$

### Worked Examples
Task is 90% I/O ($f = 0.9$, $s = 0.1$), $\bar{w} = 1$ sec:

Prefork with $c = 8$: $\Theta = 8$ tasks/sec, 8 processes consuming ~50MB each = 400MB RAM.

Eventlet with $c = 500$: $S = 1/(0.1 + 0.9/500) = 1/0.1018 = 9.82\times$ speedup per green thread "slot." Effective throughput: $500 / 1 = 500$ tasks/sec concurrently in I/O, limited by the CPU portion to ~$1/0.1 = 10$ tasks/sec of actual CPU work.

For 95% I/O tasks, eventlet/gevent at 500 concurrency handles ~$500 \times 0.95 / 1 = 475$ concurrent I/O waits while 500/20 = 25 tasks/sec complete, far exceeding prefork.

---

## Prerequisites
- queuing-theory, poisson-process, erlang-c, exponential-backoff, amdahls-law, harmonic-series, token-bucket
