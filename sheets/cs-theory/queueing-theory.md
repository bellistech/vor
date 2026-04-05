# Queueing Theory (Arrival Processes, Service Models, and Performance Laws)

A practitioner's reference for queueing theory: Kendall notation, fundamental laws, classical queue models (M/M/1, M/M/c, M/M/1/K, M/G/1, M/D/1), network models, and engineering applications in capacity planning and system sizing.

## Kendall Notation

### The Six-Part Descriptor: A/S/c/K/N/D

```
A  Arrival process distribution
S  Service time distribution
c  Number of servers (parallel)
K  System capacity (queue + servers); omit = infinite
N  Population size (calling source); omit = infinite
D  Service discipline; omit = FIFO

Common arrival/service symbols:
  M   Markovian (memoryless) -- exponential inter-arrivals or service times
  D   Deterministic (constant)
  G   General (arbitrary distribution)
  Ek  Erlang-k
  PH  Phase-type

Examples:
  M/M/1       single server, Poisson arrivals, exponential service, infinite queue
  M/M/c       c parallel servers, infinite queue
  M/M/1/K     single server, finite buffer of size K
  M/G/1       single server, general service distribution
  M/D/1       single server, deterministic (constant) service
  G/G/1       general arrivals and service, single server (hardest to analyze)
```

## Arrival Processes

### Poisson Process

```
The Poisson process is the canonical arrival model.

P(N(t) = k) = (lambda * t)^k * e^(-lambda * t) / k!

Properties:
  - Inter-arrival times are exponential: f(t) = lambda * e^(-lambda * t)
  - Memoryless: P(X > s + t | X > s) = P(X > t)
  - Merging: sum of independent Poisson processes is Poisson
  - Splitting: random thinning of Poisson process yields Poisson
  - Mean inter-arrival time: 1/lambda
  - Variance of inter-arrival time: 1/lambda^2
```

### PASTA Property

```
PASTA: Poisson Arrivals See Time Averages

If arrivals are Poisson, the fraction of arrivals that find the system
in state j equals the long-run fraction of time the system is in state j.

  a_j = p_j    (arrival-time probability = time-average probability)

This does NOT hold for non-Poisson arrivals.

Consequence: for M/M/1, an arriving customer sees the same distribution
of queue length as a random observer at a random time.
```

## Service Time Distributions

```
Exponential (memoryless):
  f(t) = mu * e^(-mu * t),  t >= 0
  Mean: 1/mu
  Variance: 1/mu^2
  Coefficient of variation: C_s = 1

Deterministic (constant):
  Service time = 1/mu exactly
  Variance: 0
  C_s = 0

Erlang-k:
  Sum of k iid exponentials, each with rate k*mu
  Mean: 1/mu
  Variance: 1/(k * mu^2)
  C_s = 1/sqrt(k)

General:
  Mean: E[S] = 1/mu
  Variance: Var[S] = sigma_s^2
  C_s = sigma_s * mu  (coefficient of variation)
```

## Fundamental Laws

### Little's Law

```
L = lambda * W

  L       mean number of customers in the system
  lambda  mean arrival rate
  W       mean time a customer spends in the system

Variants:
  L_q = lambda * W_q       (queue only, excluding service)
  L_s = lambda * E[S]      (in service only)

Properties:
  - Distribution-free: holds for ANY queueing system in steady state
  - Requires only: system is stable and long-run averages exist
  - No assumptions about arrival process or service distribution
```

### Utilization Law

```
rho = lambda / (c * mu)

  rho     server utilization (fraction of time a server is busy)
  lambda  arrival rate
  mu      service rate per server
  c       number of servers

Stability condition:  rho < 1  (for infinite-capacity queues)
```

## M/M/1 Queue

### Steady-State Results

```
Parameters:
  lambda  arrival rate (Poisson)
  mu      service rate (exponential)
  rho     = lambda / mu  (utilization, must be < 1)

State probabilities (geometric distribution):
  p_n = (1 - rho) * rho^n     n = 0, 1, 2, ...

Mean number in system:        L   = rho / (1 - rho)
Mean number in queue:         L_q = rho^2 / (1 - rho)
Mean time in system:          W   = 1 / (mu - lambda)
Mean wait in queue:           W_q = rho / (mu - lambda)
Mean response time:           W   = 1 / (mu - lambda)  = W_q + 1/mu

Variance of number in system: Var[N] = rho / (1 - rho)^2
```

### Response Time Distribution

```
P(W > t) = e^(-mu * (1 - rho) * t)

Response time is exponentially distributed with rate mu*(1-rho).

Percentiles:
  t_p = -ln(1 - p) / (mu * (1 - rho))

Example: 95th percentile response time
  t_95 = -ln(0.05) / (mu * (1 - rho))
       = 2.996 / (mu * (1 - rho))
```

## M/M/c Queue

### Erlang C Formula

```
The probability an arriving customer must wait (all c servers busy):

C(c, A) = P(wait > 0) = [A^c / c! * c/(c - A)] / [sum_{k=0}^{c-1} A^k/k! + A^c/c! * c/(c-A)]

where A = lambda/mu (offered load in Erlangs)

Performance metrics:
  L_q = C(c, A) * rho / (1 - rho)
  W_q = C(c, A) / (c * mu - lambda)
  W   = W_q + 1/mu
  L   = lambda * W

Stability: rho = lambda / (c * mu) < 1
```

## M/M/1/K Queue (Finite Buffer)

```
Finite system capacity K (queue + server).

State probabilities:
  If rho != 1:  p_n = (1 - rho) * rho^n / (1 - rho^(K+1))   n = 0, ..., K
  If rho  = 1:  p_n = 1 / (K + 1)

Blocking probability (system full):
  P_block = p_K = (1 - rho) * rho^K / (1 - rho^(K+1))

Effective arrival rate:
  lambda_eff = lambda * (1 - P_block)

Mean number in system:
  L = rho/(1-rho) - (K+1)*rho^(K+1) / (1 - rho^(K+1))

Stability: always stable (arrivals are rejected when full).
No requirement that rho < 1.
```

## M/G/1 Queue

### Pollaczek-Khinchine Formula

```
Mean number in system:
  L = rho + rho^2 * (1 + C_s^2) / (2 * (1 - rho))

where:
  rho = lambda / mu = lambda * E[S]
  C_s = sigma_s / E[S]  (coefficient of variation of service time)

Mean queue length:
  L_q = rho^2 * (1 + C_s^2) / (2 * (1 - rho))

Mean wait in queue (P-K mean value formula):
  W_q = lambda * E[S^2] / (2 * (1 - rho))

Mean time in system:
  W = W_q + E[S]

Key insight: variance of service time matters.
  C_s = 0 (deterministic) gives half the queue length of C_s = 1 (exponential).
```

## M/D/1 Queue

```
Special case of M/G/1 with deterministic service (C_s = 0).

L_q = rho^2 / (2 * (1 - rho))       (exactly half of M/M/1)
W_q = rho / (2 * mu * (1 - rho))
L   = rho + rho^2 / (2 * (1 - rho))
W   = W_q + 1/mu

M/D/1 has the smallest queue length of any M/G/1 queue with the same load.
```

## Birth-Death Processes

```
A continuous-time Markov chain on states {0, 1, 2, ...} where:
  - Transitions only occur between adjacent states
  - lambda_n = birth rate in state n (rate of n -> n+1)
  - mu_n     = death rate in state n (rate of n -> n-1)

Balance equations:
  lambda_0 * p_0 = mu_1 * p_1
  (lambda_n + mu_n) * p_n = lambda_{n-1} * p_{n-1} + mu_{n+1} * p_{n+1}

Solution:
  p_n = p_0 * prod_{i=0}^{n-1} (lambda_i / mu_{i+1})
  p_0 = 1 / [1 + sum_{n=1}^{inf} prod_{i=0}^{n-1} (lambda_i / mu_{i+1})]

M/M/1 is a birth-death process with lambda_n = lambda, mu_n = mu for all n.
M/M/c is a birth-death process with lambda_n = lambda, mu_n = min(n, c)*mu.
```

## Queueing Networks

### Jackson Networks (Open)

```
A network of M/M/c_i queues with external Poisson arrivals and
probabilistic (Markov) routing.

Jackson's Theorem:
  The joint steady-state distribution is the PRODUCT of individual
  queue distributions:

  p(n_1, n_2, ..., n_J) = prod_{i=1}^{J} p_i(n_i)

  Each queue i behaves as an independent M/M/c_i queue with
  total arrival rate Lambda_i (sum of external + internal routing).

Traffic equations:
  Lambda_i = gamma_i + sum_{j=1}^{J} Lambda_j * r_{ji}

  gamma_i = external arrival rate to queue i
  r_{ji}  = routing probability from queue j to queue i

Stability: rho_i = Lambda_i / (c_i * mu_i) < 1 for all i.
```

### Open vs Closed Networks

```
Open network:
  - Customers arrive from outside, eventually depart
  - Population varies over time
  - Stability requires rho_i < 1 at every node
  - Example: web requests flowing through load balancer -> app -> DB

Closed network:
  - Fixed population N circulates forever (no arrivals, no departures)
  - No stability condition needed
  - Solved by mean value analysis (MVA) or convolution algorithm
  - Example: thread pool with fixed number of worker threads

Gordon-Newell theorem: closed analogue of Jackson's theorem.
Product-form solution with normalization constant.
```

### Mean Value Analysis (MVA)

```
Iterative algorithm for closed queueing networks.
No need to compute normalization constant.

For N customers, at each station i:
  1. Mean queue length seen by arriving customer (by arrival theorem):
     L_i(N-1) = mean number at station i with N-1 customers

  2. Mean response time:
     R_i(N) = (1/mu_i) * (1 + L_i(N-1))    (single server)

  3. Throughput:
     X(N) = N / sum_i (V_i * R_i(N))
     where V_i = visit ratio to station i

  4. Mean queue length:
     L_i(N) = X(N) * V_i * R_i(N)

Start with L_i(0) = 0 and iterate N = 1, 2, ..., target population.
```

## Applications

### Capacity Planning

```
Web server sizing (M/M/c model):
  Given: request rate lambda, mean service time 1/mu, target W_q
  Find minimum c such that W_q <= target

Load balancer sizing:
  Model each backend as M/M/1.
  With c backends: effective rate per backend = lambda/c
  rho per backend = lambda / (c * mu)
  W = 1 / (mu - lambda/c)

Thread pool sizing:
  Closed network with N threads.
  Too few threads: low throughput (CPU underutilized)
  Too many threads: excessive context switching
  Use MVA to find optimal N.

Buffer sizing (M/M/1/K):
  Choose K to keep P_block below target.
  K = ceil(ln(P_block * (1 - rho) / (1 - rho)) / ln(rho))
  Rule of thumb: K >= 2 * L for M/M/1 gives P_block < 5%

Database connection pool:
  Model as M/M/c/c (Erlang B -- no queue, pure blocking)
  Size pool so blocking probability < 1%
```

### Quick Sizing Rules

```
At rho = 0.5:  L = 1,    W = 2/mu
At rho = 0.8:  L = 4,    W = 5/mu
At rho = 0.9:  L = 9,    W = 10/mu
At rho = 0.95: L = 19,   W = 20/mu
At rho = 0.99: L = 99,   W = 100/mu

The "hockey stick": response time explodes as rho -> 1.
Keep rho <= 0.7-0.8 for interactive systems.
Keep rho <= 0.9 for batch systems.
```

## Key Figures

| Name | Contribution | Year |
|------|-------------|------|
| Agner Krarup Erlang | Founded queueing theory, Erlang B/C formulas for telephone networks | 1909 |
| David George Kendall | Kendall notation (A/S/c/K/N/D) for classifying queues | 1953 |
| John D.C. Little | Little's Law (L = lambda * W), distribution-free | 1961 |
| Felix Pollaczek | Pollaczek-Khinchine formula for M/G/1 queues | 1930 |
| Aleksandr Khintchine | Co-developed P-K formula independently | 1932 |
| James R. Jackson | Jackson networks, product-form solutions for open networks | 1957 |
| W. J. Gordon & G. F. Newell | Closed queueing network product-form theorem | 1967 |

## Tips

- Little's Law is your most powerful tool -- it applies to any stable system, no distributional assumptions needed
- The M/M/1 model overestimates queue length when service is less variable than exponential (use M/D/1 or M/G/1)
- Always check rho < 1 before applying infinite-buffer formulas -- finite buffers change everything
- PASTA only applies to Poisson arrivals -- bursty or correlated arrivals see worse performance
- For real systems, measure the coefficient of variation of service time and use the P-K formula
- Response time percentiles matter more than means -- use the M/M/1 response time distribution
- Jackson's theorem requires Poisson external arrivals and exponential service -- violations break product form
- Heavy traffic approximation (rho -> 1): W_q approx rho / (mu * (1 - rho)) regardless of distribution details

## See Also

- `detail/cs-theory/queueing-theory.md` -- M/M/1 derivation, Little's Law proof, Erlang C derivation, capacity planning worked examples
- `sheets/cs-theory/probability-theory.md` -- exponential distribution, Poisson process, Markov chains
- `sheets/cs-theory/information-theory.md` -- entropy, source coding, channel capacity
- `sheets/cs-theory/algorithm-analysis.md` -- amortized analysis, asymptotic notation

## References

- "Queueing Systems, Volume 1: Theory" by Leonard Kleinrock (Wiley, 1975)
- "Queueing Systems, Volume 2: Computer Applications" by Leonard Kleinrock (Wiley, 1976)
- "Performance Modeling and Design of Computer Systems" by Mor Harchol-Balter (Cambridge University Press, 2013)
- "Fundamentals of Queueing Theory" by Donald Gross, John Shortle, James Thompson, and Carl Harris (Wiley, 5th ed., 2018)
- Erlang, A.K., "The Theory of Probabilities and Telephone Conversations" (Nyt Tidsskrift for Matematik B, 1909)
- Little, J.D.C., "A Proof for the Queuing Formula: L = lambda W" (Operations Research, 1961)
- Kendall, D.G., "Stochastic Processes Occurring in the Theory of Queues" (Annals of Mathematical Statistics, 1953)
- Jackson, J.R., "Networks of Waiting Lines" (Operations Research, 1957)
- Pollaczek, F., "Ueber eine Aufgabe der Wahrscheinlichkeitstheorie" (Mathematische Zeitschrift, 1930)
