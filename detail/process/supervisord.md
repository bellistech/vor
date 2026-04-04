# The Mathematics of Supervisord — Restart Policies, Process Reliability & Event Queue Theory

> *Supervisord's restart behavior follows a bounded retry model with exponential backoff, its process groups form dependency-aware scheduling units, and its event system implements a producer-consumer queue. The mathematics of reliability engineering, retry theory, and queueing models explain how to configure supervisor for maximum uptime.*

---

## 1. Restart Policy as Bounded Retry (Reliability Theory)

### The Problem

When a process crashes, supervisor restarts it up to `startretries` times. Each attempt must run for at least `startsecs` seconds to count as successful. What is the probability of reaching a stable running state?

### The Formula

Let $p$ be the probability a single start attempt succeeds (runs for $\geq$ startsecs):

$$P(\text{stable}) = 1 - (1-p)^{R}$$

Where $R$ = `startretries`. The expected number of attempts before stable:

$$E[attempts] = \frac{1}{p}$$

Time to reach FATAL state (all retries exhausted):

$$T_{fatal} = \sum_{k=1}^{R} T_{attempt,k}$$

With backoff, attempt $k$ waits approximately $k$ seconds:

$$T_{fatal} = \sum_{k=1}^{R} (\min(t_k, startsecs) + k) \approx R \cdot \frac{startsecs}{2} + \frac{R(R+1)}{2}$$

### Worked Examples

**startsecs=5, startretries=3, p=0.7 per attempt:**

$$P(\text{stable}) = 1 - (0.3)^3 = 1 - 0.027 = 97.3\%$$

$$E[attempts] = 1/0.7 = 1.43$$

$$T_{fatal} = 3 \times 2.5 + (1+2+3) = 7.5 + 6 = 13.5s$$

**startsecs=10, startretries=5, p=0.5 (flaky service):**

$$P(\text{stable}) = 1 - (0.5)^5 = 96.875\%$$

$$T_{fatal} = 5 \times 5 + (1+2+3+4+5) = 25 + 15 = 40s$$

Even with a 50% per-attempt success rate, 5 retries gives 97% eventual success.

---

## 2. Process Uptime and Availability (Reliability Engineering)

### The Problem

With `autorestart=true`, supervisor restarts crashed processes. Given a mean time between failures (MTBF) and mean time to restart (MTTR), what is the process availability?

### The Formula

$$A = \frac{MTBF}{MTBF + MTTR}$$

Where $MTTR$ includes crash detection time + restart time + startsecs verification:

$$MTTR = T_{detect} + T_{restart} + T_{startsecs}$$

For a process with crash rate $\lambda$ (crashes/hour):

$$MTBF = \frac{1}{\lambda}$$

Annual downtime:

$$D_{year} = 8760 \times (1 - A) \text{ hours}$$

### Worked Examples

**Process crashes once per day, MTTR = 8 seconds:**

$$MTBF = 24 \text{ hours}, \quad MTTR = 8/3600 = 0.00222 \text{ hours}$$

$$A = \frac{24}{24.00222} = 99.9907\%$$

$$D_{year} = 8760 \times 0.0000926 = 0.81 \text{ hours} = 48.7 \text{ minutes}$$

**Process crashes 10 times per day, MTTR = 15 seconds:**

$$MTBF = 2.4 \text{ hours}, \quad MTTR = 0.00417 \text{ hours}$$

$$A = \frac{2.4}{2.40417} = 99.827\%$$

$$D_{year} = 15.2 \text{ hours}$$

Reducing MTTR (faster startsecs, lighter process) has more impact than reducing crash rate.

---

## 3. numprocs Worker Pool Reliability (Redundancy Theory)

### The Problem

Running $N$ worker instances via `numprocs` provides redundancy. If each worker has independent failure probability, what is the probability that at least $k$ workers are available?

### The Formula

For $N$ workers, each with availability $a$:

$$P(\geq k \text{ available}) = \sum_{i=k}^{N} \binom{N}{i} a^i (1-a)^{N-i}$$

Expected available workers:

$$E[available] = N \cdot a$$

### Worked Examples

**4 workers, each 99% available, need at least 2:**

$$P(\geq 2) = \sum_{i=2}^{4} \binom{4}{i}(0.99)^i(0.01)^{4-i}$$

$$= \binom{4}{2}(0.99)^2(0.01)^2 + \binom{4}{3}(0.99)^3(0.01) + (0.99)^4$$

$$= 6(0.0000980) + 4(0.009703) + 0.96060 = 0.999994$$

$$= 99.9994\% \quad \text{(six nines from four workers)}$$

**8 workers, 95% each, need at least 4:**

$$P(\geq 4) = \sum_{i=4}^{8} \binom{8}{i}(0.95)^i(0.05)^{8-i} = 99.98\%$$

Even with 5% per-worker failure rate, 8 workers needing 4 gives excellent availability.

---

## 4. Event Listener Queue Model (Queueing Theory)

### The Problem

Event listeners receive events from supervisor through a serial protocol. With `buffer_size` events buffered, what happens under high event rates?

### The Formula

Model as M/D/1 queue (Poisson arrivals, deterministic service):

$$\rho = \frac{\lambda}{mu}$$

Where $\lambda$ = event arrival rate, $\mu$ = listener processing rate.

Queue overflow probability with buffer $B$:

$$P(\text{overflow}) = \rho^B \cdot (1 - \rho) / (1 - \rho^{B+1})$$

Mean events in queue:

$$L_q = \frac{\rho^2}{2(1 - \rho)}$$

### Worked Examples

**10 events/sec arrival, listener processes 15 events/sec, buffer=10:**

$$\rho = 10/15 = 0.667$$

$$L_q = \frac{0.667^2}{2 \times 0.333} = \frac{0.444}{0.667} = 0.667 \text{ events}$$

$$P(\text{overflow}) \approx 0.667^{10} \times 0.333 / (1-0.667^{11}) = 0.006$$

**Burst: 50 events/sec for 5 seconds, buffer=10:**

$$\rho_{burst} = 50/15 = 3.33 > 1 \quad \text{(queue grows)}$$

Events accumulated: $(50 - 15) \times 5 = 175$ events backlog.

Buffer overflow after: $10 / (50 - 15) = 0.286$ seconds. Events will be lost.

This is why event listeners should be fast and `buffer_size` should be generous for bursty workloads.

---

## 5. Log Rotation Storage Model (Capacity Planning)

### The Problem

Each process logs to files with `stdout_logfile_maxbytes` and `stdout_logfile_backups`. What is the total disk usage?

### The Formula

Per process:

$$S_{process} = (1 + B) \times M$$

Where $B$ = `stdout_logfile_backups`, $M$ = `stdout_logfile_maxbytes`.

Total for all processes:

$$S_{total} = \sum_{i=1}^{P} (1 + B_i) \times M_i$$

With numprocs $N_i$:

$$S_{total} = \sum_{i=1}^{P} N_i \times (1 + B_i) \times M_i$$

### Worked Examples

**3 programs: app (50MB, 5 backups), worker x4 (20MB, 3 backups), scheduler (10MB, 3 backups):**

$$S_{app} = 1 \times (1 + 5) \times 50 = 300 \text{ MB}$$

$$S_{worker} = 4 \times (1 + 3) \times 20 = 320 \text{ MB}$$

$$S_{scheduler} = 1 \times (1 + 3) \times 10 = 40 \text{ MB}$$

$$S_{total} = 300 + 320 + 40 = 660 \text{ MB}$$

Add stderr (if not redirected): double the estimates. Total: 1.32 GB reserved for logs.

---

## 6. Process Priority Scheduling (Order Theory)

### The Problem

Process `priority` values determine startup and shutdown order. Lower priority starts first. What is the total startup time given priorities and dependencies?

### The Formula

Group processes by priority into layers $L_1, L_2, ..., L_k$ (ascending priority):

$$T_{startup} = \sum_{j=1}^{k} \max_{p \in L_j} T_{start,p}$$

Within a layer, processes start in parallel. Sequential layers wait for the previous layer's slowest process.

### Worked Examples

**nginx(priority=100, 2s), app(200, 5s), worker_0..3(300, 3s each):**

$$L_1 = \{nginx\}: T_1 = 2s$$

$$L_2 = \{app\}: T_2 = 5s$$

$$L_3 = \{w_0, w_1, w_2, w_3\}: T_3 = 3s \text{ (parallel)}$$

$$T_{startup} = 2 + 5 + 3 = 10s$$

Without priorities (all parallel): $T = \max(2, 5, 3) = 5s$.

Priority ordering adds latency but ensures nginx is ready before app connects.

---

## Prerequisites

- reliability-theory, MTBF, MTTR, availability
- binomial-distribution, redundancy
- queueing-theory, M/D/1-queue, buffer-overflow
- capacity-planning, log-rotation
- order-theory, topological-scheduling
