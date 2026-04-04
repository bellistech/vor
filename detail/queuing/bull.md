# The Mathematics of BullMQ -- Job Scheduling and Redis Queue Mechanics

> *BullMQ implements a distributed job queue atop Redis sorted sets and lists. Its delayed job system, rate limiter, and flow orchestration all have precise mathematical underpinnings rooted in scheduling theory, token bucket algorithms, and DAG execution. Understanding these models enables predictable scaling and correct capacity planning.*

---

## 1. Delayed Job Scheduling (Sorted Set Mechanics)

### The Problem
BullMQ stores delayed jobs in a Redis sorted set keyed by their target timestamp. A polling loop moves jobs from the delayed set to the waiting list when their time arrives. What is the scheduling precision and overhead?

### The Formula
Jobs are stored with score $s = t_{\text{now}} + \Delta$ where $\Delta$ is the delay in milliseconds. The scheduler polls every $d$ ms. The actual execution time $t_{\text{exec}}$ satisfies:

$$t_{\text{target}} \leq t_{\text{exec}} \leq t_{\text{target}} + d + \epsilon$$

where $\epsilon$ is Redis command latency. The expected jitter:

$$E[\text{jitter}] = \frac{d}{2} + \bar{\epsilon}$$

For $n$ delayed jobs, the `ZRANGEBYSCORE` operation to find ready jobs has complexity:

$$O(\log N + k)$$

where $N$ is the total sorted set size and $k$ is the number of jobs being moved.

### Worked Examples
With poll interval $d = 5000$ ms (BullMQ default `stalledInterval`), Redis latency $\bar{\epsilon} = 1$ ms:

$$E[\text{jitter}] = 2500 + 1 = 2501 \text{ ms} \approx 2.5 \text{ sec}$$

For time-sensitive delays, reduce the scheduler interval. With $d = 1000$ ms:

$$E[\text{jitter}] = 500 + 1 = 501 \text{ ms}$$

A sorted set with $N = 1{,}000{,}000$ delayed jobs, finding the 50 that are ready:

$$O(\log 1{,}000{,}000 + 50) = O(20 + 50) = O(70) \text{ operations}$$

Redis executes this in microseconds -- sorted set size barely affects performance.

---

## 2. Rate Limiting (Token Bucket)

### The Problem
BullMQ's worker-level rate limiter allows `max` jobs per `duration` milliseconds. This is a sliding-window token bucket. How does it behave under varying load?

### The Formula
Token bucket parameters: capacity $B = \text{max}$, refill rate $r = B / T$ tokens per ms (where $T = \text{duration}$). At time $t$, available tokens:

$$\text{tokens}(t) = \min\left(B, \; \text{tokens}(t_{\text{last}}) + r \cdot (t - t_{\text{last}})\right)$$

A job can execute when $\text{tokens}(t) \geq 1$. After execution: $\text{tokens} \leftarrow \text{tokens} - 1$.

Sustained throughput under saturation:

$$\Theta_{\text{max}} = \frac{B}{T}$$

Burst capacity: $B$ jobs can execute instantly, then the system throttles to $r$ jobs/ms.

The delay for the $k$-th job in a burst of $n > B$ arriving at $t = 0$:

$$D(k) = \begin{cases} 0 & \text{if } k \leq B \\ \frac{k - B}{r} & \text{if } k > B \end{cases}$$

### Worked Examples
Config: `{ max: 10, duration: 1000 }` -- 10 jobs per second.

$B = 10$, $T = 1000$ ms, $r = 0.01$ jobs/ms.

Burst of 25 jobs at $t = 0$:
- Jobs 1-10: execute immediately
- Job 11: delayed $1/0.01 = 100$ ms
- Job 15: delayed $5/0.01 = 500$ ms
- Job 25: delayed $15/0.01 = 1500$ ms

Sustained arrival at 15 jobs/sec: queue grows at $15 - 10 = 5$ jobs/sec. After 1 minute: 300 jobs backed up. After 1 hour: 18,000 jobs waiting.

$$W_q(t) = \frac{(\lambda - \Theta_{\text{max}}) \cdot t}{\lambda} = \frac{5t}{15} = \frac{t}{3}$$

Wait time grows linearly. At $t = 60$ sec, average wait is 20 seconds.

---

## 3. Exponential Backoff (Retry Strategy)

### The Problem
BullMQ supports exponential backoff for retries: the delay before retry $k$ is $d \cdot b^{k-1}$ where $d$ is the base delay and $b$ is the multiplier (default 2). What is the total time window for retries?

### The Formula
Delay before retry $k$ (1-indexed):

$$\Delta_k = d \cdot b^{k-1}$$

Total delay through $R$ retry attempts:

$$T(R) = d \cdot \sum_{k=0}^{R-1} b^k = d \cdot \frac{b^R - 1}{b - 1}$$

With jitter factor $j \in [0, 1]$ (randomized to avoid thundering herd):

$$\Delta_k^{(\text{jitter})} = d \cdot b^{k-1} \cdot (1 - j + j \cdot U)$$

where $U \sim \text{Uniform}(0, 1)$. Expected total:

$$E[T(R)] = d \cdot \frac{b^R - 1}{b - 1} \cdot \left(1 - \frac{j}{2}\right)$$

### Worked Examples
Default: $d = 1000$ ms, $b = 2$, $R = 3$ attempts:

$$\Delta_1 = 1000, \quad \Delta_2 = 2000, \quad \Delta_3 = 4000$$

$$T(3) = 1000 \cdot \frac{8 - 1}{1} = 7000 \text{ ms} = 7 \text{ sec}$$

With $R = 10$: $T(10) = 1000 \cdot 1023 = 1{,}023{,}000$ ms $\approx 17$ minutes.

With $R = 20$: $T(20) = 1000 \cdot (2^{20} - 1) = 1{,}048{,}575{,}000$ ms $\approx 12.1$ days.

For gentler backoff with $b = 1.5$, $R = 10$:

$$T(10) = 1000 \cdot \frac{1.5^{10} - 1}{0.5} = 1000 \cdot \frac{57.67 - 1}{0.5} = 113{,}330 \text{ ms} \approx 1.9 \text{ min}$$

---

## 4. Flow Execution (DAG Scheduling)

### The Problem
BullMQ flows define parent-child job dependencies forming a DAG. A parent job only executes once all children complete. What is the expected flow completion time?

### The Formula
For a flow tree with depth $d$ and branching factor $b$, total jobs:

$$N = \frac{b^{d+1} - 1}{b - 1}$$

At each level $\ell$ (0 = leaves, $d$ = root), there are $b^{d - \ell}$ jobs. The completion time of level $\ell$ depends on the maximum completion time of level $\ell - 1$.

For i.i.d. job durations with mean $\bar{w}$, the expected completion of $m$ parallel jobs at one level (max of $m$ exponentials):

$$E[\max_m] = \bar{w} \cdot H_m$$

Total flow completion time through $d + 1$ levels with $c$ concurrent workers:

$$T_{\text{flow}} = \sum_{\ell=0}^{d} E\left[\max\left(1, \frac{b^{d-\ell}}{c}\right)\right] \cdot \bar{w}$$

When $c \geq b^d$ (enough workers for all leaves): $T_{\text{flow}} \approx (d + 1) \cdot \bar{w} \cdot H_{b^d / c + 1}$.

### Worked Examples
A CI/CD flow: root (deploy), 2 children (build, migrate), build has 3 children (unit-test, integration-test, lint). Total depth $d = 2$, $N = 7$ jobs.

With $\bar{w} = 30$ sec and $c = 10$ workers:

- Level 0 (leaves): 3 test jobs run in parallel. $E[\max_3] = 30 \times (1 + 1/2 + 1/3) = 55$ sec.
- Level 1: build and migrate run in parallel. $E[\max_2] = 30 \times 1.5 = 45$ sec.
- Level 2: deploy runs alone. $E = 30$ sec.

$$T_{\text{flow}} = 55 + 45 + 30 = 130 \text{ sec} \approx 2.2 \text{ min}$$

Sequential execution would take $7 \times 30 = 210$ sec. The flow saves 38% wall-clock time.

---

## 5. Concurrency and Throughput (Worker Pool Model)

### The Problem
A BullMQ worker with `concurrency: c` processes up to $c$ jobs simultaneously using the Node.js event loop (or $c$ sandboxed processes). How do you size $c$ for maximum throughput?

### The Formula
For event-loop workers (async I/O), the effective concurrency approaches $c$ for I/O-bound jobs. Throughput:

$$\Theta = \frac{c}{\bar{w}}$$

For sandboxed processors (CPU-bound), limited by available CPU cores $p$:

$$\Theta_{\text{sandbox}} = \frac{\min(c, p)}{\bar{w}_{\text{cpu}}}$$

Queue stability requires:

$$\Theta > \lambda \implies c > \lambda \cdot \bar{w}$$

Expected queue depth at utilization $\rho = \lambda \bar{w} / c$ (M/M/c approximation):

$$L_q \approx \frac{\rho^{c+1}}{(1-\rho)^2} \cdot \frac{1}{c \cdot c!}$$

### Worked Examples
I/O-bound API calls: $\bar{w} = 200$ ms, $\lambda = 100$ jobs/sec.

Required: $c > 100 \times 0.2 = 20$. Set $c = 30$ for headroom.

$$\rho = \frac{100 \times 0.2}{30} = 0.667$$

$$\Theta = 30 / 0.2 = 150 \text{ jobs/sec capacity}$$

CPU-bound image processing via sandboxed workers: $\bar{w} = 2$ sec, 8-core machine.

$$\Theta_{\text{sandbox}} = \frac{8}{2} = 4 \text{ jobs/sec}$$

Setting $c = 16$ (double the cores) gives no benefit since the CPU is the bottleneck. Each sandbox process consumes a core, so $c = p = 8$ is optimal.

---

## 6. Repeatable Job Drift (Cron Precision)

### The Problem
BullMQ repeatable jobs use cron expressions or fixed intervals. Over time, does execution drift from the intended schedule? How does the system prevent duplicate repeatable jobs?

### The Formula
For a fixed-interval repeatable job with period $P$ ms, the $k$-th execution target:

$$t_k = t_0 + k \cdot P$$

Actual execution includes processing delay $\delta_k$ and scheduling jitter $\epsilon_k$:

$$t_k^{(\text{actual})} = t_0 + k \cdot P + \epsilon_k$$

BullMQ anchors each repeat to the previous target (not actual completion), preventing drift:

$$t_{k+1} = t_k + P \quad (\text{not } t_k^{(\text{actual})} + P)$$

The deduplication key for a repeatable job:

$$\text{key} = \text{hash}(\text{name}, \text{queue}, \text{pattern}, \text{tz}, \text{endDate})$$

This ensures at most one instance of each repeatable configuration exists in the sorted set.

Drift accumulation over $n$ periods with anchor-based scheduling:

$$\text{drift}_n = \epsilon_n \quad (\text{bounded, does not accumulate})$$

versus completion-based scheduling:

$$\text{drift}_n = \sum_{k=1}^{n} \epsilon_k \quad (\text{random walk, grows as } \sqrt{n})$$

### Worked Examples
Job repeats every $P = 60{,}000$ ms (1 minute), jitter $\epsilon \sim \text{Uniform}(-500, 500)$ ms.

After 1440 iterations (24 hours) with anchor-based scheduling: max drift is still bounded at $|\epsilon| \leq 500$ ms. The 1440th execution fires within 500 ms of the exact 24-hour mark.

With completion-based scheduling (hypothetical): $\text{std}(\text{drift}_{1440}) = 500/\sqrt{3} \times \sqrt{1440} \approx 10{,}950$ ms $\approx 11$ seconds of accumulated drift. BullMQ's anchor approach avoids this entirely.

For cron-based jobs, the next execution time is computed from the cron expression, providing exact alignment with wall-clock times (accounting for DST transitions via the `tz` option).

---

## Prerequisites
- token-bucket-algorithm, sorted-sets, dag-scheduling, exponential-backoff, event-loop-concurrency, cron-expressions, random-walk
