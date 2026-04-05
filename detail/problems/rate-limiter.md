# The Mathematics of Rate Limiting -- Sliding Windows and Queueing Theory

> *A rate limiter enforces a moving constraint on event density: at every instant, the count of events in the trailing window must not exceed a threshold -- a deceptively simple invariant with deep connections to queueing theory and distributed consensus.*

---

## 1. The Sliding Window Invariant (Algorithm Design)

### The Problem

Formalize the sliding window rate limiter as a constraint over a continuous time domain.

### The Formula

Let $T = \{t_1, t_2, \ldots\}$ be the sequence of accepted request timestamps (monotonically
non-decreasing). The rate limiter enforces:

$$\forall t: |\{t_i \in T \mid t - W < t_i \le t\}| \le N$$

where $W$ is the window size in seconds and $N$ is the maximum request count.

A new request at time $t$ is **allowed** if and only if:

$$|\{t_i \in T \mid t - W < t_i \le t\}| < N$$

If allowed, $t$ is appended to $T$. The window $(t - W, t]$ is half-open: a request
at time $t_0$ expires at exactly $t_0 + W$.

### Worked Examples

$N = 3$, $W = 10$:

| Request time | Window $(t-10, t]$ | Count in window | Allowed? | $T$ after |
|:---:|:---:|:---:|:---:|:---:|
| 1 | (-9, 1] | 0 | Yes | [1] |
| 2 | (-8, 2] | 1 | Yes | [1, 2] |
| 3 | (-7, 3] | 2 | Yes | [1, 2, 3] |
| 4 | (-6, 4] | 3 | No | [1, 2, 3] |
| 11 | (1, 11] | 2 | Yes | [2, 3, 11] |

At $t = 11$: timestamp 1 falls outside $(1, 11]$, so only $\{2, 3\}$ remain. Count is 2 < 3, so allowed.

---

## 2. Amortized Cost of Eviction (Amortized Analysis)

### The Problem

Prove that the sliding window log achieves amortized O(1) per operation despite
O(k) evictions.

### The Formula

Use the **accounting method**. Assign each timestamp 2 credits when inserted:
1 credit for the insertion itself, 1 credit saved for its future eviction.

- **Insert:** costs 1 actual work + stores 1 credit. Total charge: 2.
- **Evict:** each evicted timestamp pays for its removal with its stored credit. Cost: 0 amortized.

Over $m$ operations, the total work is at most $2m$ (each timestamp is inserted once and
evicted once). Therefore amortized cost per operation is $O(1)$.

**Worst case per call:** O(N) when all $N$ timestamps expire simultaneously. But this
cannot happen on consecutive calls (nothing left to evict after a full flush).

### Worked Examples

100 requests in quick succession, then silence for $W$ seconds, then 1 request:

- Requests 1--100: each takes O(1) insert. Total: 100.
- Request 101 at time $t + W$: evicts all 100. Cost: 100.
- Amortized over 101 calls: $200 / 101 \approx 2$ per call.

---

## 3. Fixed Window vs. Sliding Window (System Design)

### The Problem

Quantify the burst vulnerability of fixed windows and show how sliding windows eliminate it.

### The Formula

**Fixed window:** Divide time into intervals $[kW, (k+1)W)$. Each interval has an
independent counter. Maximum burst at a window boundary:

$$\text{burst}_{\text{fixed}} = 2N$$

A client sends $N$ requests at $t = kW - \epsilon$ and $N$ at $t = kW + \epsilon$.
Both windows allow $N$, but $2N$ arrive within $2\epsilon$ seconds.

**Sliding window:** At any point $t$, the window $(t - W, t]$ contains at most $N$
requests. Maximum burst:

$$\text{burst}_{\text{sliding}} = N$$

The sliding window counter approximation interpolates:

$$\text{estimated} = C_{\text{prev}} \cdot \left(1 - \frac{t - kW}{W}\right) + C_{\text{curr}}$$

This reduces the boundary burst to approximately $\frac{3N}{2}$ in the worst case.

### Worked Examples

$N = 100$, $W = 60$ seconds:

- Fixed window: 100 requests at $t = 59.9$, 100 at $t = 60.1$ -- 200 in 0.2 seconds.
- Sliding window: at $t = 60.1$, window $(0.1, 60.1]$ contains all 200 -- only 100 allowed.
- Sliding counter at $t = 60.1$: $\text{estimated} = 100 \cdot (1 - 0.1/60) + 0 \approx 99.8$ -- blocks at 100.

---

## 4. Distributed Rate Limiting (Distributed Systems)

### The Problem

Extend the single-node rate limiter to a distributed system where requests arrive at
multiple servers.

### The Formula

With $S$ servers sharing a global limit of $N$ requests per window:

**Approach 1 -- Local partitioning:** Each server enforces $N/S$ requests. Simple but
wastes capacity when load is uneven.

**Approach 2 -- Centralized store (Redis):** All servers query a shared sorted set.

```
MULTI
ZREMRANGEBYSCORE key 0 (t - W)
ZADD key t t
ZCARD key
EXEC
```

If `ZCARD` result $\le N$, allow. The Lua script ensures atomicity:

$$\text{allow} \iff |\{t_i \in \text{ZSET} \mid t_i > t - W\}| < N$$

**Approach 3 -- Gossip-based:** Servers periodically share local counts. Convergence
delay allows temporary over-admission of up to $\delta \cdot \lambda$ extra requests,
where $\delta$ is gossip interval and $\lambda$ is arrival rate.

### Worked Examples

3 servers, $N = 100$, $W = 60$. Partitioned: each allows 33. If Server A gets 90% of
traffic, it rejects at 33 while B and C are idle. Centralized Redis allows all 100 to
go to Server A.

---

## 5. Token Bucket Equivalence (Queueing Theory)

### The Problem

Compare the sliding window log to the token bucket algorithm and analyze their
behavioral differences.

### The Formula

**Token bucket:** A bucket of capacity $B$ refills at rate $r = N/W$ tokens per second.
Each request consumes 1 token. If the bucket is empty, the request is rejected.

$$\text{tokens}(t) = \min(B, \text{tokens}(t_{\text{last}}) + r \cdot (t - t_{\text{last}}))$$

**Behavioral difference:** The token bucket allows bursts of up to $B$ requests
instantaneously (draining the bucket), then enforces a steady rate of $r$. The sliding
window always enforces exactly $N$ in any $W$-second span.

**Equivalence:** Setting $B = N$ and $r = N/W$, the token bucket approximately matches
the sliding window for uniform traffic. Under bursty traffic, the token bucket is more
permissive (allows burst then recovers), while the sliding window is strict.

### Worked Examples

$N = 10$, $W = 10$ ($r = 1$/sec), idle for 10 seconds:

- Token bucket: 10 tokens accumulated. Burst of 10 allowed, then 1/sec.
- Sliding window: 10 requests in first second allowed. 11th in same second rejected.
  But after 1 more second, window shifts and 1 more allowed.

Both allow 10 in 10 seconds, but the token bucket permits all 10 at $t = 0$ while the
sliding window also permits all 10 at $t = 0$ (same in this case).

---

## Prerequisites

- Deque/queue data structures
- Mutex and thread safety
- Amortized analysis (accounting method)
- Basic queueing theory (optional)

## Complexity

| Level | Description |
|-------|-------------|
| **Beginner** | Implement sliding window log with a list/deque. Test basic allow/reject behavior. Understand the eviction cutoff. |
| **Intermediate** | Add thread safety with a mutex. Implement the sliding window counter approximation. Compare fixed vs. sliding window burst behavior. Analyze amortized complexity. |
| **Advanced** | Implement distributed rate limiting with Redis sorted sets. Compare token bucket vs. sliding window under various traffic patterns. Implement adaptive rate limiting that adjusts limits based on server load. Study leaky bucket and GCRA algorithms. |
