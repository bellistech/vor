# The Mathematics of Testcontainers -- Container Lifecycle and Resource Scheduling

> *Integration testing with real containers introduces scheduling theory, resource allocation, and startup-time modeling that unit tests never face. Every container is a stochastic process with variable startup time, memory footprint, and port allocation.*

---

## 1. Container Startup Time Distribution (Queuing Theory)

### The Problem

Container startup is not deterministic. Image pull time, layer extraction, entrypoint execution, and readiness checks introduce variable delays. Modeling this variance prevents flaky timeouts.

### The Formula

Startup time $T$ follows a log-normal distribution (positive, right-skewed):

$$T \sim \text{LogNormal}(\mu, \sigma^2)$$

$$f(t) = \frac{1}{t \sigma \sqrt{2\pi}} \exp\left(-\frac{(\ln t - \mu)^2}{2\sigma^2}\right)$$

The expected startup time and variance:

$$E[T] = e^{\mu + \sigma^2/2}$$

$$\text{Var}(T) = (e^{\sigma^2} - 1) \cdot e^{2\mu + \sigma^2}$$

### Worked Examples

PostgreSQL container startup measured over 100 runs: $\mu = 1.2$ (ln-seconds), $\sigma = 0.4$:

$$E[T] = e^{1.2 + 0.08} = e^{1.28} \approx 3.6 \text{ s}$$

The 99th percentile (for timeout setting):

$$T_{99} = e^{\mu + 2.326\sigma} = e^{1.2 + 0.93} = e^{2.13} \approx 8.4 \text{ s}$$

A safe timeout should be at least $T_{99}$. In CI with cold image cache, add pull time:

$$T_{total} = T_{pull} + T_{start} \approx 15 + 8.4 = 23.4 \text{ s}$$

Setting timeout to $30$ seconds gives margin: $\frac{30 - 23.4}{23.4} = 28\%$ headroom.

---

## 2. Port Allocation and Collision Probability (Birthday Problem)

### The Problem

Testcontainers maps container ports to random ephemeral host ports. When running parallel tests, each needing containers, what is the probability of port collisions?

### The Formula

The ephemeral port range is typically $[32768, 60999]$, giving $N = 28{,}232$ ports. For $k$ containers requesting random ports, the collision probability follows the birthday problem:

$$P(\text{collision}) = 1 - \prod_{i=0}^{k-1} \frac{N - i}{N} \approx 1 - e^{-k(k-1)/(2N)}$$

### Worked Examples

Running 20 parallel tests, each with 1 container:

$$P(\text{collision}) \approx 1 - e^{-20 \times 19 / (2 \times 28232)} = 1 - e^{-0.00673} \approx 0.67\%$$

Running 50 containers:

$$P(\text{collision}) \approx 1 - e^{-50 \times 49 / 56464} = 1 - e^{-0.0434} \approx 4.2\%$$

Testcontainers avoids this by letting Docker assign ports (avoiding explicit port binding), making collisions effectively zero. The risk only appears when tests hardcode port numbers.

---

## 3. Resource Budgeting (Memory and CPU)

### The Problem

Each container consumes memory and CPU. Running too many simultaneously causes OOM kills or CPU starvation. The resource budget determines maximum parallelism.

### The Formula

Maximum parallel containers given available resources:

$$C_{max} = \min\left(\left\lfloor\frac{M_{available}}{M_{container}}\right\rfloor, \left\lfloor\frac{\text{CPU}_{available}}{\text{CPU}_{container}}\right\rfloor\right)$$

Total test suite time with $n$ container tests and $C_{max}$ parallel slots:

$$T_{suite} = \left\lceil\frac{n}{C_{max}}\right\rceil \times \max(T_{container_i})$$

### Worked Examples

CI runner: 8 GB RAM, 4 CPUs. PostgreSQL container: 256 MB RAM, 0.5 CPU. Test runner overhead: 1 GB RAM, 0.5 CPU.

$$M_{available} = 8000 - 1000 = 7000 \text{ MB}$$
$$\text{CPU}_{available} = 4 - 0.5 = 3.5$$

$$C_{max} = \min\left(\lfloor 7000 / 256 \rfloor, \lfloor 3.5 / 0.5 \rfloor\right) = \min(27, 7) = 7$$

For 21 container tests with avg $T_{container} = 5$s:

$$T_{suite} = \lceil 21 / 7 \rceil \times 5 = 3 \times 5 = 15 \text{ s}$$

Without parallelism: $21 \times 5 = 105$ s. Speedup: $7\times$.

---

## 4. Wait Strategy Polling (Convergence)

### The Problem

Wait strategies poll a readiness condition at fixed intervals until success or timeout. The expected number of polls and total wait time depend on the polling interval and the time at which the container actually becomes ready.

### The Formula

Given polling interval $\Delta$ and actual readiness time $T_r$, the number of polls before success:

$$N_{polls} = \left\lceil\frac{T_r}{\Delta}\right\rceil$$

The wasted time (polling overhead) is:

$$T_{waste} = N_{polls} \times \Delta - T_r = \left(\left\lceil\frac{T_r}{\Delta}\right\rceil \times \Delta\right) - T_r$$

The maximum waste per container is bounded by one polling interval:

$$T_{waste} < \Delta$$

### Worked Examples

Container ready at $T_r = 3.7$ s, polling every $\Delta = 1$ s:

$$N_{polls} = \lceil 3.7 \rceil = 4$$
$$T_{waste} = 4 \times 1 - 3.7 = 0.3 \text{ s}$$

With $\Delta = 0.1$ s (aggressive polling):

$$N_{polls} = \lceil 37 \rceil = 37$$
$$T_{waste} = 3.7 - 3.7 = 0.0 \text{ s}$$

But aggressive polling increases CPU load. For $n$ containers polling simultaneously:

$$\text{Polls/second} = \frac{n}{\Delta}$$

With $n = 10$, $\Delta = 0.1$: 100 polls/second. With $\Delta = 1$: 10 polls/second. The tradeoff is latency vs. CPU overhead.

---

## 5. Container Reuse Economics (Amortization)

### The Problem

Creating a fresh container per test maximizes isolation but is expensive. Reusing containers across tests amortizes startup cost but risks state leakage. The break-even point determines when reuse pays off.

### The Formula

Per-test cost with fresh containers:

$$C_{fresh} = T_{start} + T_{test} + T_{stop}$$

Per-test cost with reuse across $n$ tests:

$$C_{reuse} = \frac{T_{start} + T_{stop}}{n} + T_{test} + T_{reset}$$

where $T_{reset}$ is the cost of resetting state (truncating tables, flushing caches).

Break-even when $C_{reuse} < C_{fresh}$:

$$\frac{T_{start} + T_{stop}}{n} + T_{reset} < T_{start} + T_{stop}$$

$$T_{reset} < (T_{start} + T_{stop}) \left(1 - \frac{1}{n}\right)$$

### Worked Examples

PostgreSQL: $T_{start} = 4$ s, $T_{stop} = 1$ s, $T_{test} = 0.5$ s, $T_{reset} = 0.1$ s (TRUNCATE), $n = 30$ tests:

$$C_{fresh} = 30 \times (4 + 0.5 + 1) = 165 \text{ s}$$

$$C_{reuse} = (4 + 1) + 30 \times (0.5 + 0.1) = 5 + 18 = 23 \text{ s}$$

$$\text{Speedup} = \frac{165}{23} = 7.2\times$$

Break-even check:

$$0.1 < 5 \times (1 - 1/30) = 4.83 \quad \checkmark$$

Reuse is profitable when reset cost is less than $96.7\%$ of startup+stop cost.

---

## 6. Network Topology Complexity (Graph Theory)

### The Problem

Multi-container test setups form networks where services communicate. The number of possible communication paths grows combinatorially, affecting both test complexity and debugging difficulty.

### The Formula

For $n$ containers on a shared network, the maximum number of directed communication edges:

$$E_{max} = n(n-1)$$

The number of possible network topologies (which services can reach which):

$$T_{topologies} = 2^{n(n-1)}$$

### Worked Examples

A 4-container setup (app, db, cache, queue):

$$E_{max} = 4 \times 3 = 12 \text{ directed edges}$$

$$T_{topologies} = 2^{12} = 4{,}096$$

In practice, only a few topologies matter. A typical microservice test has:

$$E_{actual} = n - 1 \text{ (tree topology)} = 3$$

The connectivity ratio:

$$\rho = \frac{E_{actual}}{E_{max}} = \frac{3}{12} = 25\%$$

Lower connectivity means simpler debugging. Each additional edge adds a potential failure mode.

---

## Prerequisites

- Probability distributions (log-normal, birthday problem)
- Queuing theory basics (arrival rates, service times)
- Amortization and break-even analysis
- Graph theory (directed graphs, connectivity)
- Operating system resource management (memory, CPU scheduling)
- Docker networking fundamentals (bridge networks, port mapping)
