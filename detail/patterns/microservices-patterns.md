# The Mathematics of Microservices — Failure, Retries, and Distributed Consensus

> *Microservice patterns are engineering responses to mathematical certainties: failures will cascade, retries will amplify load, and distributed consensus has provable limits. Understanding the formulas reveals optimal configurations.*

---

## 1. Circuit Breaker State Machine (Formal Automaton)

### The Problem

How do we formally model circuit breaker behavior and prove that it prevents cascading failures?

### The Formula

A circuit breaker is a Mealy machine $M = (Q, \Sigma, \Delta, \delta, \lambda, q_0)$ where:

- $Q = \{\text{Closed}, \text{Open}, \text{HalfOpen}\}$
- $\Sigma = \{\text{success}, \text{failure}, \text{timeout}\}$ (inputs)
- $\Delta = \{\text{forward}, \text{reject}, \text{probe}\}$ (outputs)

Transition function $\delta$ and output function $\lambda$:

| State | Input | Next State | Output |
|---|---|---|---|
| Closed | success | Closed | forward |
| Closed | failure ($n < T$) | Closed | forward |
| Closed | failure ($n \geq T$) | Open | reject |
| Open | any request | Open | reject |
| Open | timeout | HalfOpen | probe |
| HalfOpen | success ($m \geq S$) | Closed | forward |
| HalfOpen | success ($m < S$) | HalfOpen | probe |
| HalfOpen | failure | Open | reject |

Where $T$ = failure threshold, $S$ = success threshold, $n$ = failure count, $m$ = half-open success count.

**Failure amplification without circuit breaker**: If service $A$ calls service $B$ with timeout $t$ and $B$ is down, $A$ blocks for $t$ per request. With $\lambda$ requests/second arriving at $A$, blocked threads accumulate at rate $\lambda \cdot t$. Thread pool of size $P$ exhausts in $P / \lambda$ seconds. With circuit breaker, rejection is $O(1)$ — no thread blocking.

### Worked Examples

Service receives 1000 req/s, each call to downstream has 5s timeout, thread pool = 200.

Without circuit breaker: Pool exhausts in $200 / 1000 = 0.2$ seconds. All requests fail.

With circuit breaker (threshold = 5, timeout = 30s): After 5 failures (at most 25 thread-seconds consumed), circuit opens. Remaining 995 req/s get instant rejection. After 30s, one probe request tests downstream. If it succeeds, 3 more probes (success threshold), then full traffic resumes — total downtime approximately 31 seconds instead of indefinite cascade.

---

## 2. Exponential Backoff Mathematics (Retry Analysis)

### The Problem

What is the expected total wait time and expected number of retries before success? How does jitter prevent thundering herds?

### The Formula

**Basic exponential backoff**: delay after attempt $k$ is $d_k = b \cdot 2^k$ where $b$ is the base delay.

**Total wait time** after $n$ retries:

$$T_n = \sum_{k=0}^{n-1} b \cdot 2^k = b(2^n - 1)$$

**With full jitter** (uniform random in $[0, d_k]$): expected delay at attempt $k$ is $E[d_k] = b \cdot 2^k / 2 = b \cdot 2^{k-1}$.

Expected total wait:

$$E[T_n] = \sum_{k=0}^{n-1} b \cdot 2^{k-1} = \frac{b(2^n - 1)}{2}$$

**Success probability**: If each attempt succeeds independently with probability $p$, the probability of success within $n$ attempts:

$$P(\text{success within } n) = 1 - (1-p)^n$$

**Expected attempts to first success**: $E[K] = 1/p$ (geometric distribution).

**Thundering herd**: Without jitter, $N$ clients all retry at the same moment (synchronized failure). With full jitter over interval $[0, d_k]$, the expected number of clients retrying in any window of size $\epsilon$ is approximately $N \cdot \epsilon / d_k$ — spreading the load evenly.

### Worked Examples

Base delay $b = 100\text{ms}$, max 5 retries, success probability per attempt $p = 0.8$:

| Attempt $k$ | Max delay | Expected delay (jitter) | Cumulative expected wait |
|---|---|---|---|
| 0 | 100ms | 50ms | 50ms |
| 1 | 200ms | 100ms | 150ms |
| 2 | 400ms | 200ms | 350ms |
| 3 | 800ms | 400ms | 750ms |
| 4 | 1600ms | 800ms | 1550ms |

$P(\text{success within 5}) = 1 - 0.2^5 = 0.99968$.

With 1000 clients and no jitter: all 1000 hit the server at exactly 100ms, then 200ms, then 400ms — spike pattern. With full jitter: at the 400ms retry level, clients spread uniformly over [0, 400ms] — peak load = $1000/400 \approx 2.5$ req/ms instead of instantaneous 1000.

---

## 3. Saga Rollback DAG Analysis (Compensation Ordering)

### The Problem

When a saga step fails, compensating transactions must execute in reverse order. What are the constraints on this rollback, and when is correct compensation impossible?

### The Formula

A saga is a sequence of transactions $T_1, T_2, \ldots, T_n$ with compensating transactions $C_1, C_2, \ldots, C_n$. The compensation graph $G_C = (V, E)$ has edges representing ordering constraints.

For a linear saga, if $T_k$ fails, compensations execute as $C_{k-1}, C_{k-2}, \ldots, C_1$.

**Commutativity requirement**: If compensations $C_i$ and $C_j$ are independent (no shared state), they may execute in parallel: $C_i \parallel C_j$.

**Semantic compensation**: Unlike ACID rollback, saga compensations are semantic — they apply a correcting action, not an undo. The compensated state $S'$ may differ from the original state $S_0$:

$$S' = C_1(C_2(\ldots C_{k-1}(S_k))) \neq S_0$$

For example, a "refund" compensation leaves a refund record — the state is not identical to "never charged."

**Idempotency requirement**: Each $C_i$ must be idempotent because the orchestrator may retry failed compensations:

$$C_i(C_i(S)) = C_i(S)$$

### Worked Examples

Order saga: $T_1$ = reserve inventory, $T_2$ = charge payment, $T_3$ = create shipment.

If $T_3$ fails: execute $C_2$ (refund) then $C_1$ (release inventory). Since refund and release are independent: $C_2 \parallel C_1$ is safe, reducing compensation time from $\text{latency}(C_2) + \text{latency}(C_1)$ to $\max(\text{latency}(C_2), \text{latency}(C_1))$.

---

## 4. CAP Theorem Proof Sketch (Brewer/Gilbert-Lynch)

### The Problem

Can a distributed system simultaneously provide Consistency, Availability, and Partition tolerance?

### The Formula

**Theorem** (Gilbert & Lynch, 2002): It is impossible for a distributed system to simultaneously provide all three:

1. **Consistency** (C): Every read receives the most recent write
2. **Availability** (A): Every request receives a response (no timeouts)
3. **Partition tolerance** (P): The system operates despite network partitions

**Proof sketch**: Consider two nodes $N_1$ and $N_2$ with a network partition between them.

1. Client writes value $v_1$ to $N_1$.
2. Due to partition, $N_2$ cannot receive the update.
3. Client reads from $N_2$.
4. System must choose:
   - Return stale data (sacrifice C, preserve A)
   - Block/error (sacrifice A, preserve C)
   - This situation cannot occur (sacrifice P — but partitions are inevitable)

Since network partitions are unavoidable in distributed systems, the practical choice is between CP and AP:

- **CP systems**: Block during partitions (HBase, MongoDB with majority reads, etcd)
- **AP systems**: Return possibly stale data (Cassandra, DynamoDB, CouchDB)

**PACELC** extension: If there is a Partition, choose A or C; Else (normal operation), choose Latency or Consistency.

### Worked Examples

Two-datacenter deployment, network link fails:

- **CP choice** (banking): Reject writes to the minority partition. Users in one datacenter see errors. Data is always correct.
- **AP choice** (social media): Both datacenters accept writes. After partition heals, merge conflicts using last-write-wins or application-specific resolution. Users always get a response, but may see stale data.

---

## 5. Cascading Failure Probability (Dependency Chains)

### The Problem

When services form dependency chains, how does the probability of end-to-end failure grow?

### The Formula

For a serial chain of $n$ independent services, each with availability $a_i$:

$$A_{\text{chain}} = \prod_{i=1}^{n} a_i$$

For $n$ services each at 99.9% availability:

$$A = 0.999^n$$

**Failure probability** of the chain:

$$P(\text{failure}) = 1 - \prod_{i=1}^{n} a_i$$

For parallel redundancy with $k$ replicas:

$$A_{\text{redundant}} = 1 - (1 - a)^k$$

**Fan-out amplification**: If a service fans out to $m$ downstream services (all required for response):

$$A_{\text{fanout}} = a_{\text{self}} \cdot \prod_{j=1}^{m} a_j$$

### Worked Examples

Request path: API Gateway (99.99%) -> Auth (99.9%) -> Order (99.9%) -> Payment (99.95%) -> Inventory (99.9%):

$$A = 0.9999 \times 0.999 \times 0.999 \times 0.9995 \times 0.999 = 0.9964$$

That is 99.64% availability = 31.5 hours of downtime per year, despite each service being "three nines" or better.

With circuit breakers and fallbacks reducing the dependency chain from serial to partially parallel (graceful degradation when payment is down):

$$A_{\text{degraded}} = 0.9999 \times 0.999 \times 0.999 \times (1 - (1-0.9995)^2) \times 0.999 = 0.9974$$

Adding a second payment provider replica brings the chain to 99.74%.

---

## Prerequisites

- Basic probability and independence
- Finite state machines / automata theory
- Geometric series and exponential functions
- Directed acyclic graphs

## Complexity

| Concept | Key Metric | Formula | Impact |
|---|---|---|---|
| Circuit breaker rejection | Latency | $O(1)$ vs $O(\text{timeout})$ | Prevents thread exhaustion |
| Exponential backoff total wait | Time | $b(2^n - 1)/2$ with jitter | Bounds retry duration |
| Chain availability | Probability | $\prod a_i$ | Multiplies failure risk |
| Saga compensation | Time | $\max(C_i)$ if parallelizable | Reduces rollback latency |
| Jitter load spreading | Throughput | $N \cdot \epsilon / d_k$ per window | Prevents thundering herd |
