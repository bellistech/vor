# The Mathematics of Queueing Theory -- Derivations, Proofs, and Engineering Applications

> *Queueing theory transforms the intuition that "systems slow down when busy" into a precise mathematical framework. From Erlang's telephone exchange calculations in 1909 to modern cloud capacity planning, the same birth-death equations and conservation laws govern waiting in line.*

---

## 1. The M/M/1 Queue (Full Derivation)

### The Problem

Derive the steady-state distribution, mean queue length, and mean response time for a single-server queue with Poisson arrivals at rate $\lambda$ and exponential service at rate $\mu$.

### The Model

A continuous-time Markov chain on states $\{0, 1, 2, \ldots\}$ where state $n$ represents $n$ customers in the system (queue plus server).

**Transition rates:**
- State $n \to n+1$ at rate $\lambda$ (arrival)
- State $n \to n-1$ at rate $\mu$ (service completion), for $n \geq 1$

This is a birth-death process with constant birth rate $\lambda_n = \lambda$ and constant death rate $\mu_n = \mu$.

### Balance Equations

In steady state, the rate of flow into each state equals the rate of flow out. The **global balance equations** are:

For state 0:
$$\lambda p_0 = \mu p_1$$

For state $n \geq 1$:
$$(\lambda + \mu) p_n = \lambda p_{n-1} + \mu p_{n+1}$$

These simplify to **detailed balance** (rate crossing each boundary equals in both directions):

$$\lambda p_n = \mu p_{n+1} \quad \text{for all } n \geq 0$$

### Solving for the Geometric Distribution

From detailed balance, we get the recursion:

$$p_{n+1} = \frac{\lambda}{\mu} p_n = \rho \, p_n$$

where $\rho = \lambda / \mu$. Iterating:

$$p_n = \rho^n \, p_0$$

The normalization condition $\sum_{n=0}^{\infty} p_n = 1$ gives:

$$p_0 \sum_{n=0}^{\infty} \rho^n = 1$$

This geometric series converges if and only if $\rho < 1$, yielding:

$$p_0 = 1 - \rho$$

Therefore:

$$\boxed{p_n = (1 - \rho) \rho^n, \quad n = 0, 1, 2, \ldots}$$

The number of customers in the system follows a geometric distribution.

### Deriving the Mean Queue Length

$$L = E[N] = \sum_{n=0}^{\infty} n \, p_n = (1 - \rho) \sum_{n=0}^{\infty} n \rho^n$$

Using the identity $\sum_{n=0}^{\infty} n x^n = x / (1 - x)^2$ for $|x| < 1$:

$$L = (1 - \rho) \cdot \frac{\rho}{(1 - \rho)^2} = \frac{\rho}{1 - \rho}$$

$$\boxed{L = \frac{\rho}{1 - \rho} = \frac{\lambda}{\mu - \lambda}}$$

### Deriving Mean Response Time

Apply Little's Law ($L = \lambda W$):

$$W = \frac{L}{\lambda} = \frac{\rho}{\lambda(1 - \rho)} = \frac{1}{\mu - \lambda}$$

The mean wait in queue (excluding service):

$$W_q = W - \frac{1}{\mu} = \frac{1}{\mu - \lambda} - \frac{1}{\mu} = \frac{\lambda}{\mu(\mu - \lambda)} = \frac{\rho}{\mu - \lambda}$$

### Response Time Distribution

The response time $W$ of a customer in an M/M/1 queue is exponentially distributed:

$$P(W > t) = e^{-\mu(1 - \rho) t}$$

**Proof.** A customer arriving to find $n$ customers already present must wait for $n+1$ exponential service completions (the $n$ ahead plus their own). The total time is the sum of $n+1$ iid $\text{Exp}(\mu)$ random variables, which is $\text{Erlang}(n+1, \mu)$. Unconditioning over the geometric arrival-state distribution (using PASTA):

$$P(W > t) = \sum_{n=0}^{\infty} (1 - \rho) \rho^n \, P(\text{Erlang}(n+1, \mu) > t)$$

After evaluation (using the Laplace transform or direct computation), this simplifies to:

$$P(W > t) = e^{-\mu(1 - \rho) t}$$

**Percentiles.** The $p$-th percentile of response time:

$$t_p = \frac{-\ln(1 - p)}{\mu(1 - \rho)}$$

| Percentile | Formula | At $\rho = 0.8$, $\mu = 100$/s |
|---|---|---|
| 50th (median) | $0.693 / (\mu(1-\rho))$ | 34.7 ms |
| 90th | $2.303 / (\mu(1-\rho))$ | 115.1 ms |
| 95th | $2.996 / (\mu(1-\rho))$ | 149.8 ms |
| 99th | $4.605 / (\mu(1-\rho))$ | 230.3 ms |

### Variance of Queue Length

$$\text{Var}[N] = \frac{\rho}{(1 - \rho)^2}$$

Note that $\text{Var}[N] / E[N] = 1 / (1 - \rho)$, which grows without bound as $\rho \to 1$.

---

## 2. Little's Law (Proof Sketch)

### Statement

For any stable queueing system in which long-run averages exist:

$$L = \lambda W$$

where $L$ is the time-average number of customers, $\lambda$ is the long-run arrival rate, and $W$ is the mean time each customer spends in the system.

### Proof Sketch (Stidham's Approach)

Consider a system over time interval $[0, T]$. Let:

- $A(T)$ = number of arrivals in $[0, T]$
- $D(T)$ = number of departures in $[0, T]$
- $N(t)$ = number in system at time $t$, so $N(t) = A(t) - D(t)$ (assuming no customers at time 0)
- $W_i$ = time customer $i$ spends in the system

**Step 1.** The total customer-time (area under the $N(t)$ curve) can be expressed two ways:

$$\int_0^T N(t) \, dt = \sum_{i=1}^{A(T)} W_i$$

The left side counts the total "person-time" by integrating over time. The right side counts it by summing each customer's sojourn time. Both measure the same area.

**Step 2.** Divide both sides by $T$:

$$\frac{1}{T} \int_0^T N(t) \, dt = \frac{A(T)}{T} \cdot \frac{1}{A(T)} \sum_{i=1}^{A(T)} W_i$$

**Step 3.** Take $T \to \infty$. If the limits exist:

- Left side $\to L$ (time-average number in system)
- $A(T)/T \to \lambda$ (arrival rate)
- $\frac{1}{A(T)} \sum W_i \to W$ (mean sojourn time)

Therefore $L = \lambda W$. $\square$

**Generality.** The proof requires only that the three limits exist. No assumptions about arrival process, service distribution, number of servers, service discipline, or network structure.

---

## 3. Erlang C Formula (Derivation)

### The Problem

For an M/M/c queue (Poisson arrivals, exponential service, $c$ parallel servers, infinite buffer), derive the probability that an arriving customer must wait.

### The Birth-Death Chain

States $\{0, 1, 2, \ldots\}$ with rates:

- $\lambda_n = \lambda$ for all $n$ (Poisson arrivals)
- $\mu_n = \min(n, c) \cdot \mu$ (all idle servers work in parallel)

### Balance Equations

From the birth-death solution:

$$p_n = p_0 \prod_{i=0}^{n-1} \frac{\lambda_i}{\mu_{i+1}}$$

For $n \leq c$:

$$p_n = \frac{A^n}{n!} p_0, \quad \text{where } A = \lambda / \mu$$

For $n > c$ (all servers busy, so $\mu_n = c\mu$):

$$p_n = \frac{A^n}{c! \cdot c^{n-c}} p_0 = \frac{A^c}{c!} \rho^{n-c} p_0$$

where $\rho = A/c = \lambda / (c\mu)$.

### Normalization

$$1 = p_0 \left[ \sum_{k=0}^{c-1} \frac{A^k}{k!} + \frac{A^c}{c!} \sum_{j=0}^{\infty} \rho^j \right]$$

The geometric series converges iff $\rho < 1$:

$$p_0 = \left[ \sum_{k=0}^{c-1} \frac{A^k}{k!} + \frac{A^c}{c!} \cdot \frac{1}{1 - \rho} \right]^{-1}$$

### The Erlang C Probability

The probability of waiting is $P(\text{wait}) = P(N \geq c) = \sum_{n=c}^{\infty} p_n$:

$$C(c, A) = \frac{A^c}{c!} \cdot \frac{1}{1 - \rho} \cdot p_0$$

$$\boxed{C(c, A) = \frac{\dfrac{A^c}{c!} \cdot \dfrac{c}{c - A}}{\displaystyle\sum_{k=0}^{c-1} \frac{A^k}{k!} + \frac{A^c}{c!} \cdot \frac{c}{c - A}}}$$

### Performance Metrics

Given $C(c, A)$:

$$W_q = \frac{C(c, A)}{c\mu - \lambda}, \quad L_q = \lambda W_q = \frac{C(c, A) \cdot \rho}{1 - \rho}$$

$$W = W_q + \frac{1}{\mu}, \quad L = \lambda W$$

---

## 4. Pollaczek-Khinchine Mean Value Formula

### The Problem

For an M/G/1 queue (Poisson arrivals, general service distribution), derive the mean number in the system.

### Setup

Let service times have mean $E[S] = 1/\mu$, second moment $E[S^2]$, and variance $\sigma_S^2 = E[S^2] - (1/\mu)^2$. Define:

- $\rho = \lambda E[S] = \lambda / \mu$
- $C_S = \sigma_S \mu$ (coefficient of variation of service time)

### Derivation via Residual Service Time

Consider an arriving customer (by PASTA, they see a time-average snapshot). The mean waiting time in queue is:

$$W_q = E[\text{residual service of current customer}] + L_q \cdot E[S]$$

The residual (remaining) service time of the customer currently being served, seen by a random observer:

$$E[R] = \frac{E[S^2]}{2 E[S]} = \frac{\lambda E[S^2]}{2} \cdot \frac{1}{\rho}$$

Wait, let us be more precise. With probability $\rho$, the server is busy when the arrival occurs, and the expected residual service is $E[S^2] / (2E[S])$. The mean work the arrival finds ahead of it:

$$W_q = \rho \cdot \frac{E[S^2]}{2 E[S]} + L_q \cdot E[S]$$

Substituting $L_q = \lambda W_q$ (Little's Law):

$$W_q = \frac{\rho E[S^2]}{2 E[S]} + \lambda W_q E[S]$$

$$W_q = \frac{\rho E[S^2]}{2 E[S]} + \rho W_q$$

$$W_q (1 - \rho) = \frac{\rho E[S^2]}{2 E[S]}$$

$$\boxed{W_q = \frac{\lambda E[S^2]}{2(1 - \rho)}}$$

This is the **Pollaczek-Khinchine mean value formula**.

### Expressing in Terms of $C_S$

Since $E[S^2] = \sigma_S^2 + E[S]^2 = E[S]^2 (C_S^2 + 1)$:

$$W_q = \frac{\rho}{2\mu(1 - \rho)} (1 + C_S^2)$$

Mean queue length:

$$L_q = \lambda W_q = \frac{\rho^2 (1 + C_S^2)}{2(1 - \rho)}$$

Mean number in system:

$$L = L_q + \rho = \rho + \frac{\rho^2 (1 + C_S^2)}{2(1 - \rho)}$$

### Special Cases

| Model | $C_S$ | $L_q$ |
|---|---|---|
| M/D/1 | 0 | $\rho^2 / [2(1-\rho)]$ |
| M/M/1 | 1 | $\rho^2 / (1-\rho)$ |
| M/G/1, $C_S = 2$ | 2 | $5\rho^2 / [2(1-\rho)]$ |

**Key insight:** Doubling the coefficient of variation from $C_S = 1$ to $C_S = 2$ increases the mean queue length by a factor of 2.5. Reducing variability (e.g., by making service times more deterministic) is as effective as reducing load.

---

## 5. Jackson's Theorem (Product-Form Solution)

### Statement

Consider an open network of $J$ queues, each with $c_i$ servers and exponential service at rate $\mu_i$. External Poisson arrivals enter queue $i$ at rate $\gamma_i$. After service at queue $j$, a customer routes to queue $i$ with probability $r_{ji}$ or departs the network with probability $1 - \sum_i r_{ji}$.

**Theorem (Jackson, 1957).** If $\rho_i < 1$ for all $i$, the joint steady-state distribution is:

$$p(n_1, n_2, \ldots, n_J) = \prod_{i=1}^{J} p_i(n_i)$$

where $p_i(n_i)$ is the marginal distribution of an M/M/$c_i$ queue with arrival rate $\Lambda_i$ solving:

$$\Lambda_i = \gamma_i + \sum_{j=1}^{J} \Lambda_j \, r_{ji}$$

### Why This is Remarkable

Departures from an M/M/c queue are Poisson (Burke's theorem), and splitting/merging Poisson processes yields Poisson processes. Jackson's theorem shows these properties compose: despite complex feedback and routing, each queue behaves *as if* it were an independent M/M/$c_i$ queue.

### Proof Sketch

**Step 1.** Solve the traffic equations $\Lambda_i = \gamma_i + \sum_j \Lambda_j r_{ji}$ for the total arrival rates.

**Step 2.** Propose the product-form solution $\pi(\mathbf{n}) = \prod_i \pi_i(n_i)$ where each $\pi_i$ is the M/M/$c_i$ distribution with rate $\Lambda_i$.

**Step 3.** Verify this satisfies the global balance equations by substituting into:

$$\sum_{\mathbf{n'}} q(\mathbf{n}, \mathbf{n'}) \pi(\mathbf{n}) = \sum_{\mathbf{n'}} q(\mathbf{n'}, \mathbf{n}) \pi(\mathbf{n'})$$

where $q(\mathbf{n}, \mathbf{n'})$ is the transition rate from state $\mathbf{n}$ to $\mathbf{n'}$.

**Step 4.** The verification works because the detailed balance structure of each M/M/$c_i$ queue, combined with the Poisson arrival/departure properties, makes all terms cancel appropriately.

### Limitations

Jackson's theorem fails when:
- Service times are not exponential (use the BCMP theorem for phase-type distributions)
- Routing depends on queue lengths (state-dependent routing)
- Arrivals are not Poisson
- Service rates depend on the total network state (not just local queue length)

---

## 6. Heavy Traffic Approximation

### The Idea

As $\rho \to 1$, queueing behavior becomes universal: the details of arrival and service distributions wash out, and a diffusion approximation applies.

For a G/G/1 queue with arrival rate $\lambda$, service rate $\mu$, and squared coefficients of variation $C_A^2$ (arrivals) and $C_S^2$ (service):

$$W_q \approx \frac{\rho}{1 - \rho} \cdot \frac{C_A^2 + C_S^2}{2} \cdot \frac{1}{\mu}$$

This is the **Kingman approximation** (also called the VUT formula: Variability, Utilization, Time).

### Components

$$W_q \approx \underbrace{\frac{\rho}{1 - \rho}}_{\text{utilization}} \cdot \underbrace{\frac{C_A^2 + C_S^2}{2}}_{\text{variability}} \cdot \underbrace{\frac{1}{\mu}}_{\text{mean service time}}$$

**Verification against known formulas:**

- M/M/1: $C_A^2 = 1, C_S^2 = 1$, gives $W_q = \rho / (\mu(1-\rho))$ -- exact.
- M/D/1: $C_A^2 = 1, C_S^2 = 0$, gives $W_q = \rho / (2\mu(1-\rho))$ -- exact.
- M/G/1: $C_A^2 = 1$, gives $W_q = \rho(1 + C_S^2) / (2\mu(1-\rho))$ -- matches P-K formula.

---

## 7. Capacity Planning Worked Examples

### Example 1: Web Server Sizing

**Problem.** A web application receives 800 requests/second. Mean response time of the application server is 5 ms. What is the minimum number of servers to achieve 95th-percentile response time under 50 ms?

**Model.** M/M/c queue with $\lambda = 800$/s and $\mu = 200$/s (i.e., $1/\mu = 5$ ms).

**Step 1.** Offered load: $A = \lambda / \mu = 4$ Erlangs.

**Step 2.** Minimum servers for stability: $c > A = 4$, so $c \geq 5$.

**Step 3.** Evaluate M/M/c performance for each candidate $c$:

| $c$ | $\rho$ | $C(c,A)$ | $W_q$ (ms) | $W$ (ms) | 95th pct $W$ (ms) |
|---|---|---|---|---|---|
| 5 | 0.800 | 0.554 | 2.77 | 7.77 | 23.3 |
| 6 | 0.667 | 0.237 | 0.59 | 5.59 | 16.8 |
| 7 | 0.571 | 0.089 | 0.15 | 5.15 | 15.4 |

For M/M/c, the 95th percentile of response time can be approximated as:

$$t_{95} \approx W \cdot \frac{-\ln(0.05 \cdot (1 - \rho))}{\mu(1 - \rho) / \mu}$$

With $c = 5$: $t_{95} \approx 23.3$ ms, which meets the 50 ms target.

**Answer:** 5 servers suffice, but 6 provides substantial margin.

### Example 2: Database Connection Pool Sizing

**Problem.** A service makes database queries at a rate of 500/s. Each query holds a connection for an average of 8 ms. The connection pool has a maximum size $c$. Model as M/M/c. What pool size keeps mean wait time under 2 ms?

**Parameters.** $\lambda = 500$/s, $E[S] = 8$ ms, $\mu = 125$/s, $A = 4$ Erlangs.

**Step 1.** Need $c > 4$ for stability.

**Step 2.** Evaluate:

| $c$ | $\rho$ | $C(c,A)$ | $W_q$ (ms) |
|---|---|---|---|
| 5 | 0.800 | 0.554 | 4.43 |
| 6 | 0.667 | 0.237 | 0.95 |
| 7 | 0.571 | 0.089 | 0.24 |

**Answer:** Pool size of 6 gives $W_q = 0.95$ ms $< 2$ ms. Use $c = 6$ with headroom for traffic spikes.

### Example 3: Buffer Sizing with M/M/1/K

**Problem.** A network switch port has $\lambda = 900$ Mbps arrival rate and $\mu = 1000$ Mbps capacity ($\rho = 0.9$). What buffer size $K$ keeps packet loss below 0.1%?

**Model.** M/M/1/K with $\rho = 0.9$.

**Blocking probability:**

$$P_{\text{block}} = \frac{(1 - \rho)\rho^K}{1 - \rho^{K+1}}$$

We need $P_{\text{block}} \leq 0.001$. Solving numerically:

| $K$ | $P_{\text{block}}$ |
|---|---|
| 10 | 0.0648 |
| 20 | 0.0088 |
| 30 | 0.0012 |
| 40 | 0.00016 |

**Answer:** $K = 40$ gives loss rate below 0.1%. At $\rho = 0.9$, substantial buffering is needed because the system spends significant time near capacity.

### Example 4: Thread Pool via MVA

**Problem.** A closed system has $N$ worker threads cycling through a CPU (service time 2 ms) and a disk (service time 10 ms). Find throughput as a function of $N$.

**Model.** Closed network, 2 stations, $\mu_{\text{CPU}} = 500$/s, $\mu_{\text{disk}} = 100$/s.

**MVA iteration:**

| $N$ | $R_{\text{CPU}}$ (ms) | $R_{\text{disk}}$ (ms) | $X$ (req/s) | $U_{\text{CPU}}$ | $U_{\text{disk}}$ |
|---|---|---|---|---|---|
| 1 | 2.0 | 10.0 | 83.3 | 0.167 | 0.833 |
| 2 | 2.3 | 18.3 | 97.1 | 0.194 | 0.971 |
| 4 | 2.8 | 34.2 | 108.1 | 0.216 | 1.081 |
| 8 | 3.8 | 65.3 | ... | ... | ... |

The disk becomes the bottleneck. Beyond $N \approx 10$, adding threads increases queue time at the disk with minimal throughput gain.

**Optimal $N$:** The throughput knee occurs around $N = D/Z + 1$ where $D$ = total service demand and $Z$ = think time (0 here). With $D = 12$ ms: about 1-2 threads per disk service time unit, so $N \approx 6$-$8$.

---

## 8. Summary of Key Relationships

```
Model       L_q                          W_q                           Key Parameter
--------    -------------------------    ----------------------------  ------------------
M/M/1       rho^2 / (1 - rho)           rho / (mu(1 - rho))           rho = lambda/mu
M/D/1       rho^2 / (2(1 - rho))        rho / (2*mu*(1 - rho))        deterministic svc
M/G/1       rho^2(1+C_s^2)/(2(1-rho))   rho(1+C_s^2)/(2*mu*(1-rho))  C_s = CoV of svc
M/M/c       C(c,A)*rho / (1 - rho)      C(c,A) / (c*mu - lambda)     Erlang C
M/M/1/K     (finite, always stable)     (finite, always stable)       P_block = p_K
G/G/1       ~rho(C_a^2+C_s^2)/(2(1-r))  Kingman approximation         heavy traffic
```

**Conservation law:** Across all models, reducing variability (lower $C_S$, lower $C_A$) always reduces waiting. The P-K formula and Kingman approximation both show this: queue length is proportional to the sum of squared coefficients of variation.

---

## References

- Kleinrock, L. "Queueing Systems, Volume 1: Theory" (Wiley, 1975)
- Harchol-Balter, M. "Performance Modeling and Design of Computer Systems" (Cambridge, 2013)
- Gross, D. et al. "Fundamentals of Queueing Theory" (Wiley, 5th ed., 2018)
- Little, J.D.C. "A Proof for the Queuing Formula: L = lambda W" (Operations Research, 1961)
- Stidham, S. "A Last Word on L = lambda W" (Operations Research, 1974)
- Kingman, J.F.C. "The Single Server Queue in Heavy Traffic" (Proc. Cambridge Phil. Soc., 1961)
- Jackson, J.R. "Networks of Waiting Lines" (Operations Research, 1957)
- Burke, P.J. "The Output of a Queuing System" (Operations Research, 1956)
- Erlang, A.K. "The Theory of Probabilities and Telephone Conversations" (1909)
- Pollaczek, F. "Ueber eine Aufgabe der Wahrscheinlichkeitstheorie" (Math. Zeitschrift, 1930)
