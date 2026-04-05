# The Mathematics of API Design — Rate Limiting, Pagination, and Compatibility

> *API design decisions have quantifiable consequences: token bucket parameters determine burst tolerance, pagination strategies affect consistency under concurrency, and versioning compatibility forms a directed graph with precise rules.*

---

## 1. Token Bucket Rate Limiting (Burst and Steady-State)

### The Problem

How do we allow short bursts of traffic while enforcing a long-term average rate? What are the mathematical properties of the token bucket algorithm?

### The Formula

A **token bucket** has two parameters:
- $r$ = refill rate (tokens per second)
- $b$ = bucket capacity (maximum burst size)

Tokens are added at rate $r$ up to maximum $b$. Each request consumes one token. If the bucket is empty, the request is rejected (or queued).

**Steady-state throughput**: $r$ requests/second.

**Maximum burst**: $b$ requests in an instant (if bucket is full).

**Time to refill from empty to full**: $T_{\text{refill}} = b / r$.

**Burst duration**: A sustained burst at rate $\lambda > r$ can last:

$$T_{\text{burst}} = \frac{b}{\lambda - r}$$

After $T_{\text{burst}}$ seconds, the bucket empties and excess requests are rejected.

**Conformance**: A traffic pattern $\lambda(t)$ conforms to the token bucket $(r, b)$ if for all intervals $[t_1, t_2]$:

$$\int_{t_1}^{t_2} \lambda(t) \, dt \leq b + r(t_2 - t_1)$$

### Worked Examples

API rate limit: $r = 100$ req/s, $b = 500$ requests.

- Steady state: 100 requests/second sustained indefinitely.
- Full burst: 500 requests instantly, then 100/s thereafter.
- Burst at 200 req/s: lasts $500 / (200 - 100) = 5$ seconds. After 5 seconds, bucket is empty and only 100 req/s is allowed.
- Refill time: $500 / 100 = 5$ seconds from empty to full.

For a mobile app that opens and makes 20 API calls rapidly, then idles: set $b \geq 20$ to accommodate the burst. If the app polls every 10 seconds: $r = 1/10 = 0.1$ req/s is sufficient for steady state, but $b = 20$ handles the burst on open.

---

## 2. Sliding Window Rate Limiting (Precision and Memory)

### The Problem

Fixed-window rate limiting has a boundary problem: a client can make $2 \times \text{limit}$ requests across the boundary of two windows. How does sliding window rate limiting fix this?

### The Formula

**Fixed window problem**: Window 1 = [0, 60s), Window 2 = [60s, 120s). Client sends $L$ requests at $t = 59$ and $L$ requests at $t = 61$. Each window allows $L$, so $2L$ requests pass in 2 seconds.

**Sliding window** approximation (Redis-style): Weight requests from the previous window by the fraction of overlap:

$$\text{count} = \text{prev\_window\_count} \times \frac{W - t_{\text{elapsed}}}{W} + \text{curr\_window\_count}$$

Where $W$ = window size, $t_{\text{elapsed}}$ = time into current window.

**Sliding window log** (exact): Store timestamp of each request. Count requests with timestamp $> t_{\text{now}} - W$. Exact but $O(n)$ space per client where $n$ = number of requests in window.

**Memory comparison**:

| Algorithm | Space per client | Precision |
|---|---|---|
| Fixed window | $O(1)$ (one counter) | Poor at boundaries |
| Sliding window counter | $O(1)$ (two counters) | Good approximation |
| Sliding window log | $O(L)$ ($L$ = limit) | Exact |
| Token bucket | $O(1)$ (count + timestamp) | Exact for bursts |

### Worked Examples

Rate limit: 100 req/min. Previous window: 80 requests. Current window: 30 requests (20 seconds in).

Sliding window estimate: $80 \times \frac{60 - 20}{60} + 30 = 80 \times 0.667 + 30 = 53.3 + 30 = 83.3$.

Since $83.3 < 100$: request allowed. If current window had 50 requests: $80 \times 0.667 + 50 = 103.3 > 100$: rejected.

---

## 3. Pagination Consistency (Concurrent Writes)

### The Problem

When data changes between paginated requests, clients may see duplicate items or miss items entirely. How do we quantify and mitigate this?

### The Formula

**Offset pagination under concurrent inserts**: If $k$ items are inserted into positions before the current offset between page requests:

- Items at positions $[\text{offset}, \text{offset}+\text{limit})$ shift by $+k$
- Client receives $k$ duplicates of items that shifted into the new page
- Client misses $0$ items (items shift right, not left)

For concurrent deletes of $d$ items before the offset:
- Items shift by $-d$
- Client misses $d$ items that shifted past the current page
- Client receives $0$ duplicates

**Cursor pagination**: A cursor encodes the last seen item's position (e.g., `id > 123`). Concurrent writes do not affect consistency because the cursor is anchored to a specific item, not a position.

**Probability of phantom reads** with offset pagination: If $\lambda$ = write rate (inserts/deletes per second), $\Delta t$ = time between page requests, and $N$ = total items:

$$P(\text{phantom}) = 1 - e^{-\lambda \Delta t}$$

Expected duplicates or missed items per page boundary: $\lambda \Delta t$.

### Worked Examples

API with 100,000 users, 10 inserts/second, client fetches pages every 2 seconds:

- Expected position shifts per page: $10 \times 2 = 20$ items
- With 50-item pages: client might see up to 20 duplicates at each page boundary
- Over 2000 pages: potentially hundreds of duplicates and missed items

Same scenario with cursor pagination: Client requests `WHERE id > 123 LIMIT 50`. Inserts with $\text{id} > 123$ appear in future pages (no duplicates). Inserts with $\text{id} < 123$ are invisible (already passed). Consistency is perfect for the client's traversal direction.

---

## 4. API Versioning Compatibility Graph (Evolution Analysis)

### The Problem

As an API evolves through versions, which transitions are safe for clients? How do we model backward and forward compatibility?

### The Formula

Model the API evolution as a directed graph $G = (V, E)$ where:
- Vertices $V = \{v_1, v_2, \ldots, v_n\}$ are API versions
- Edge $(v_i, v_j)$ exists if clients of $v_i$ can safely migrate to $v_j$

**Backward compatibility**: Version $v_j$ is backward-compatible with $v_i$ if all valid requests to $v_i$ are also valid requests to $v_j$, and responses from $v_j$ are a superset of $v_i$:

$$\text{BC}(v_i, v_j) \iff \text{Req}(v_i) \subseteq \text{Req}(v_j) \land \text{Resp}(v_i) \subseteq \text{Resp}(v_j)$$

**Forward compatibility**: Version $v_j$ is forward-compatible with $v_i$ if clients written for $v_j$ can interact with $v_i$:

$$\text{FC}(v_i, v_j) \iff \text{Req}(v_j) \subseteq \text{Req}(v_i) \land \text{Resp}(v_j) \subseteq \text{Resp}(v_i)$$

**Postel's Law** (Robustness Principle): "Be conservative in what you send, be liberal in what you accept."

Formally: A client $C$ follows Postel's Law if:
- $C$ sends only the minimal required request fields
- $C$ ignores unknown response fields

If both client and server follow Postel's Law, the compatibility surface increases dramatically — most additive changes become non-breaking.

### Worked Examples

API v1: `GET /users/123` returns `{id, name, email}`.
API v2: adds `phone` field. Returns `{id, name, email, phone}`.

- BC($v_1, v_2$): Yes. v1 requests work on v2. v2 responses have all v1 fields.
- FC($v_1, v_2$): Yes if client ignores unknown fields (Postel's Law). Client written for v2 expecting `phone` breaks on v1.

API v3: removes `email`, adds `primary_email` and `secondary_email`.

- BC($v_2, v_3$): No. Clients expecting `email` field break.
- This requires a new major version (v2 -> v3) or a migration period where both `email` and `primary_email` are returned.

---

## 5. Postel's Law Formalization (Interoperability Theory)

### The Problem

How do we formalize the robustness principle to maximize API interoperability across versions?

### The Formula

Define the **acceptance set** $A(C)$ as the set of all messages a component $C$ can process without error. Define the **production set** $P(C)$ as the set of messages $C$ may produce.

**Postel's Law**: For maximum interoperability, a well-behaved component $C$ should satisfy:

$$P(C) \subseteq S_{\text{spec}} \subseteq A(C)$$

Where $S_{\text{spec}}$ is the set of messages defined by the specification.

- $P(C) \subseteq S_{\text{spec}}$: Only send spec-compliant messages (conservative output)
- $S_{\text{spec}} \subseteq A(C)$: Accept everything the spec allows, plus more (liberal input)

**Interoperability probability**: Given $n$ independently developed components, each accepting spec-compliant messages with probability $p$ and extra-spec messages with probability $q$:

$$P(\text{interop}) = p^n \text{ (strict)} \quad \text{vs} \quad (p + q(1-p))^n \text{ (Postel)}$$

For $p = 0.95$, $q = 0.8$, $n = 5$:
- Strict: $0.95^5 = 0.774$
- Postel: $(0.95 + 0.8 \times 0.05)^5 = 0.99^5 = 0.951$

### Worked Examples

JSON API with 20 response fields. Client A expects exactly 20 fields (strict). Client B ignores unknown fields (Postel).

Server adds field 21 (new feature):
- Client A: crashes or returns error (probability of failure = 1)
- Client B: works perfectly (probability of failure = 0)

Server temporarily omits optional field 15 (bug):
- Client A: crashes if field 15 is used
- Client B: handles gracefully if field 15 is optional in its logic

Over 10 API changes per year with 50 client implementations: Postel-compliant clients require approximately 0 emergency updates, strict clients require an average of 3-5 updates per year.

---

## Prerequisites

- Basic probability for rate limiting analysis
- Set theory for compatibility definitions
- Understanding of HTTP semantics and REST principles

## Complexity

| Concept | Formula | Key Tradeoff |
|---|---|---|
| Token bucket burst | $T = b / (\lambda - r)$ | Burst size vs memory |
| Sliding window precision | $O(1)$ vs $O(L)$ space | Precision vs memory |
| Offset phantom reads | $\lambda \Delta t$ expected | Simplicity vs consistency |
| Cursor consistency | 0 phantoms | Requires stable sort key |
| Postel interop | $(p+q(1-p))^n$ | Robustness vs strictness |
