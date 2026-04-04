# The Mathematics of Sidekiq -- Thread Concurrency and Retry Backoff

> *Sidekiq's threaded concurrency model trades the memory overhead of process-per-worker for the complexity of shared-state threading. Its retry system implements a polynomial backoff curve that spreads retries over days. Understanding these mathematical foundations helps tune concurrency, predict retry windows, and size Redis memory.*

---

## 1. Thread Pool Concurrency (Little's Law)

### The Problem
Sidekiq runs $c$ threads in a single Ruby process. Each thread picks a job from Redis, executes it, and returns. Given an arrival rate and mean job duration, how many threads do we need to avoid growing queue depth?

### The Formula
Little's Law relates the average number of jobs in the system $L$, the arrival rate $\lambda$, and the average time in system $W$:

$$L = \lambda W$$

For a stable system with $c$ threads, utilization $\rho$:

$$\rho = \frac{\lambda \bar{w}}{c}$$

The system is stable when $\rho < 1$, meaning:

$$c > \lambda \bar{w}$$

Average queue length (M/M/c model):

$$L_q = \frac{P_W \cdot \rho}{1 - \rho}$$

where $P_W$ is the Erlang-C wait probability. Average wait time:

$$W_q = \frac{L_q}{\lambda}$$

### Worked Examples
$\lambda = 50$ jobs/sec, $\bar{w} = 0.4$ sec, $c = 25$ threads:

$$\rho = \frac{50 \times 0.4}{25} = 0.8$$

$$L = 50 \times 0.4 = 20 \text{ jobs in system on average}$$

Of those 20, about $c \cdot \rho = 20$ are being processed and $L_q \approx 3.2$ are waiting. With $c = 25$, wait time is roughly $W_q = 3.2/50 = 64$ ms.

Increasing to $c = 30$: $\rho = 0.667$, $L_q \approx 0.7$, $W_q \approx 14$ ms. But this requires 30 database connections.

---

## 2. Retry Backoff Curve (Polynomial Delay)

### The Problem
Sidekiq's default retry formula is $f(n) = n^4 + 15 + \text{rand}(10) \cdot (n + 1)$, where $n$ is the retry count. How does total retry time accumulate over 25 retries?

### The Formula
The deterministic component of retry delay at attempt $n$:

$$\Delta(n) = n^4 + 15$$

The jitter component has expected value:

$$E[\text{jitter}(n)] = 4.5 \cdot (n + 1)$$

Total expected delay for retry $n$:

$$E[\Delta(n)] = n^4 + 15 + 4.5(n + 1)$$

Cumulative time through $R$ retries:

$$T(R) = \sum_{n=1}^{R} E[\Delta(n)] = \sum_{n=1}^{R} n^4 + 15R + 4.5 \sum_{n=1}^{R}(n+1)$$

Using the identity $\sum_{n=1}^{R} n^4 = \frac{R(R+1)(2R+1)(3R^2+3R-1)}{30}$:

$$T(R) = \frac{R(R+1)(2R+1)(3R^2+3R-1)}{30} + 15R + 4.5 \cdot \frac{(R+1)(R+2)}{2} - 4.5$$

### Worked Examples
For the default $R = 25$:

$$\sum_{n=1}^{25} n^4 = \frac{25 \times 26 \times 51 \times (3 \times 625 + 75 - 1)}{30} = \frac{25 \times 26 \times 51 \times 1949}{30}$$

$$= \frac{25 \times 26 \times 51 \times 1949}{30} = 2,153,645$$

$$15 \times 25 = 375$$

$$4.5 \times \frac{26 \times 27}{2} - 4.5 = 4.5 \times 351 - 4.5 = 1575.0$$

$$T(25) = 2,153,645 + 375 + 1575 = 2,155,595 \text{ sec} \approx 24.9 \text{ days}$$

Sidekiq spreads 25 retries over approximately 25 days. The last retry alone:

$$\Delta(25) = 25^4 + 15 + 4.5 \times 26 = 390,625 + 15 + 117 = 390,757 \text{ sec} \approx 4.5 \text{ days}$$

---

## 3. Memory Sizing (Redis Capacity)

### The Problem
Every enqueued Sidekiq job is a JSON blob stored in Redis. How much Redis memory is needed for a given queue depth and job payload size?

### The Formula
Each job in Redis has a base overhead $h$ (Redis list entry overhead, ~100 bytes) plus the serialized JSON payload $s$ bytes. For $n$ jobs across $q$ queues:

$$M_{\text{jobs}} = n \cdot (h + s)$$

The retry set and scheduled set use sorted sets. Each entry has additional overhead $h_z$ (~120 bytes for the sorted set node):

$$M_{\text{retry}} = n_r \cdot (h_z + s)$$

Total Redis memory:

$$M_{\text{total}} = M_{\text{jobs}} + M_{\text{retry}} + M_{\text{scheduled}} + M_{\text{dead}} + M_{\text{overhead}}$$

Redis overhead for data structures and fragmentation (typically 1.2-2x):

$$M_{\text{actual}} \approx 1.5 \times M_{\text{total}}$$

### Worked Examples
Average job payload $s = 500$ bytes, $h = 100$ bytes, $n = 1{,}000{,}000$ queued jobs:

$$M_{\text{jobs}} = 1{,}000{,}000 \times 600 = 600 \text{ MB}$$

With 50,000 in retry set and 10,000 dead:

$$M_{\text{retry}} = 50{,}000 \times 620 = 31 \text{ MB}$$
$$M_{\text{dead}} = 10{,}000 \times 620 = 6.2 \text{ MB}$$

$$M_{\text{actual}} \approx 1.5 \times 637 \approx 956 \text{ MB}$$

Rule of thumb: 1 KB per job (with overhead). 1M jobs needs about 1 GB of Redis. Keep job args small.

---

## 4. Queue Weight Scheduling (Weighted Random Selection)

### The Problem
Sidekiq supports weighted queue priorities (`[critical, 6], [default, 4], [low, 2]`). How does this translate to job throughput per queue?

### The Formula
With weights $w_1, w_2, \ldots, w_q$ for queues $1, 2, \ldots, q$, Sidekiq generates a random permutation weighted by frequency. The probability of checking queue $i$ first in a given cycle:

$$P(i \text{ first}) = \frac{w_i}{\sum_{j=1}^{q} w_j}$$

The expected fraction of processing time devoted to queue $i$ (when all queues have work):

$$F_i = \frac{w_i}{\sum_{j=1}^{q} w_j}$$

When queue $i$ is empty, its allocation redistributes proportionally to other queues.

### Worked Examples
Weights: critical=6, default=4, low=2. Total = 12.

$$F_{\text{critical}} = 6/12 = 50\%$$
$$F_{\text{default}} = 4/12 = 33.3\%$$
$$F_{\text{low}} = 2/12 = 16.7\%$$

With $c = 24$ threads and all queues busy: critical gets ~12 threads, default ~8, low ~4.

If critical empties: default gets $4/6 = 66.7\%$ and low gets $2/6 = 33.3\%$ of all 24 threads.

Note: this is not strict priority. Low-priority jobs still execute even when critical has work. For strict priority, list queues without weights: `queues: [critical, default, low]`.

---

## 5. Throughput Under GVL (Global VM Lock)

### The Problem
Ruby's GVL (Global VM Lock, formerly GIL) serializes CPU-bound Ruby code. With $c$ Sidekiq threads, what is the actual throughput for jobs with a mix of CPU and I/O work?

### The Formula
A job spends fraction $f_{\text{io}}$ in I/O (GVL released) and $f_{\text{cpu}} = 1 - f_{\text{io}}$ in CPU (GVL held). Effective parallelism:

$$P_{\text{eff}}(c) = \frac{1}{f_{\text{cpu}} + \frac{f_{\text{io}}}{c}}$$

This is Amdahl's Law where the "serial" portion is CPU work (GVL-bound). Maximum speedup:

$$\lim_{c \to \infty} P_{\text{eff}}(c) = \frac{1}{f_{\text{cpu}}}$$

Throughput with $c$ threads:

$$\Theta(c) = \frac{P_{\text{eff}}(c)}{\bar{w}} = \frac{1}{\bar{w} \cdot f_{\text{cpu}} + \bar{w} \cdot f_{\text{io}} / c}$$

### Worked Examples
Job: $\bar{w} = 200$ ms, $f_{\text{io}} = 0.8$ (80% database/HTTP), $f_{\text{cpu}} = 0.2$:

With $c = 25$ threads:

$$P_{\text{eff}} = \frac{1}{0.2 + 0.8/25} = \frac{1}{0.232} = 4.31$$

$$\Theta = \frac{4.31}{0.2} = 21.5 \text{ jobs/sec}$$

Theoretical max ($c \to \infty$): $1/0.2 = 5\times$ speedup, $\Theta_{\max} = 25$ jobs/sec.

Diminishing returns: going from 25 to 50 threads yields $P_{\text{eff}} = 1/(0.2 + 0.016) = 4.63$, only 7% improvement. The GVL caps CPU-bound throughput regardless of thread count.

For 95% I/O jobs ($f_{\text{cpu}} = 0.05$): $P_{\text{eff}}(25) = 1/(0.05 + 0.038) = 11.4$, and threads scale much better.

---

## 6. Batch Completion Probability (Pro Feature)

### The Problem
A Sidekiq Pro batch launches $n$ jobs. Each has independent success probability $p$. What is the probability the batch fully succeeds, and how does the `on(:complete)` callback timing depend on the slowest job?

### The Formula
Probability all $n$ jobs succeed (including retries with max $R$):

$$P_{\text{batch}} = \left(1 - (1-p)^{R+1}\right)^n$$

For per-job failure rate $q = 1 - p$ and single-attempt success:

$$P_{\text{batch}} = p^n$$

Expected number of failures in the batch:

$$E[\text{failures}] = n \cdot (1-p)^{R+1}$$

### Worked Examples
$n = 1000$ jobs, per-attempt $p = 0.99$, $R = 5$ retries:

Per-job ultimate failure: $(0.01)^6 = 10^{-12}$

$$P_{\text{batch}} = (1 - 10^{-12})^{1000} \approx 1 - 10^{-9}$$

Virtually guaranteed success. But with $p = 0.9$ and $R = 3$:

Per-job ultimate failure: $(0.1)^4 = 0.0001$

$$E[\text{failures}] = 1000 \times 0.0001 = 0.1$$

$$P_{\text{batch}} = (1 - 0.0001)^{1000} = 0.9048$$

Only 90.5% chance of full batch success. Increasing $R$ to 5: $(0.1)^6 = 10^{-6}$, $P_{\text{batch}} = 0.999$, much safer.

---

## Prerequisites
- littles-law, amdahls-law, queuing-theory, polynomial-growth, sorted-sets, erlang-c, weighted-random
