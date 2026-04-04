# The Mathematics of REST APIs — Idempotency, Pagination, and Rate Limiting

> *REST's architectural constraints are not just design preferences -- they encode precise mathematical properties. Idempotency is a property from algebra, pagination is a windowing function over ordered sets, and rate limiting is a token bucket problem from queueing theory. These formalisms let you reason about correctness and performance guarantees.*

---

## 1. Idempotency (Abstract Algebra)

### The Problem

An operation $f$ is idempotent if applying it multiple times produces the same result as applying it once. This property is critical for safe retries in distributed systems where network failures cause duplicate requests.

### The Formula

$$f \text{ is idempotent} \iff \forall x: f(f(x)) = f(x)$$

For HTTP methods, idempotency means the server state after $n$ identical requests is the same as after 1:

$$S_n = f^n(S_0) = f(S_0) = S_1 \quad \forall n \geq 1$$

### Worked Examples

| Method | Idempotent | Why |
|:---|:---:|:---|
| `GET /users/42` | yes | $S_n = S_0$ (no state change, read-only) |
| `PUT /users/42 {name: "Alice"}` | yes | $S_n = \{name: Alice\}$ regardless of $n$ |
| `DELETE /users/42` | yes | $S_n = \emptyset$ (gone after first, still gone after $n$th) |
| `POST /users {name: "Alice"}` | no | $S_n$ creates $n$ distinct users |
| `PATCH /users/42 {age: age+1}` | no | $S_n = \{age: age_0 + n\}$ (relative update) |

### Idempotency Key Collision Probability

For UUID v4 idempotency keys with 122 random bits, the collision probability for $n$ keys:

$$P(\text{collision}) \approx 1 - e^{-\frac{n^2}{2 \cdot 2^{122}}}$$

| Keys Generated | Collision Probability |
|:---:|:---:|
| $10^6$ | $\approx 10^{-25}$ |
| $10^{12}$ | $\approx 10^{-13}$ |
| $10^{18}$ | $\approx 10^{-1}$ |

---

## 2. Pagination Mathematics (Set Theory)

### The Problem

Pagination divides an ordered set of records into pages. Offset-based and cursor-based approaches have different consistency and performance characteristics that can be modeled precisely.

### Offset Pagination Formula

For a dataset of $N$ records, page size $L$, page number $p$ (1-indexed):

$$\text{offset} = (p - 1) \cdot L$$

$$\text{total\_pages} = \left\lceil \frac{N}{L} \right\rceil$$

$$\text{records\_on\_page}(p) = \min(L, \; N - (p-1) \cdot L)$$

### The Consistency Problem

If $k$ records are inserted before the current offset between two paginated requests:

$$\text{duplicates} = \min(k, L) \quad \text{(records shift forward, re-seen on next page)}$$

If $k$ records are deleted before the current offset:

$$\text{skipped} = \min(k, L) \quad \text{(records shift backward, missed entirely)}$$

### Cursor Pagination Formula

With cursor $c$ pointing to a specific record and stable sort key $s$:

$$\text{Page}(c, L) = \{r \in R : s(r) > s(c)\}[0:L]$$

Consistency guarantee: no duplicates or skips regardless of concurrent mutations, because the cursor anchors to a value, not a position.

### Worked Example: Offset vs Cursor Under Writes

Dataset: 100 records, page size 20, fetching page 3 (records 41-60).

Between page 2 and page 3 requests, 5 new records are inserted at positions 10-14:

| Method | Expected Records | Actual Records | Issue |
|:---|:---:|:---:|:---|
| Offset ($\text{offset}=40$) | 41-60 | 36-55 (original numbering) | 5 duplicates from page 2 |
| Cursor ($s > s_{40}$) | 41-60 | 41-60 (original numbering) | correct, stable |

---

## 3. Rate Limiting (Queueing Theory)

### The Problem

Rate limiting controls the flow of requests to protect server resources. The token bucket algorithm is the most common implementation, and its behavior can be modeled precisely.

### Token Bucket Formula

Bucket capacity $B$, refill rate $r$ tokens per second, current tokens $T(t)$:

$$T(t) = \min\left(B, \; T(t_0) + r \cdot (t - t_0)\right)$$

A request consuming $c$ tokens is allowed if:

$$T(t) \geq c$$

After allowing:

$$T(t)' = T(t) - c$$

### Burst Capacity

Maximum burst (requests in rapid succession from a full bucket):

$$\text{burst}_{max} = \left\lfloor \frac{B}{c} \right\rfloor$$

Time to recover from empty to full:

$$t_{recovery} = \frac{B}{r}$$

### Worked Example

Rate limit: 100 requests/minute ($r = 100/60 \approx 1.67$/s), burst capacity $B = 20$, cost $c = 1$:

$$\text{burst}_{max} = 20 \text{ requests instantly}$$
$$t_{recovery} = \frac{20}{1.67} \approx 12\text{s to refill from empty}$$

Sustained rate: $r = 1.67$ req/s = 100 req/min.

| Time (s) | Tokens | Request | Allowed |
|:---:|:---:|:---:|:---:|
| 0 | 20 | 20 burst | yes (T=0) |
| 1 | 1.67 | 1 | yes (T=0.67) |
| 6 | 10.67 | 10 | yes (T=0.67) |
| 7 | 2.34 | 3 | no (need 3, have 2.34) |

---

## 4. Conditional Requests and Caching (Information Theory)

### The Problem

ETags reduce bandwidth by allowing conditional requests. The ETag is a fingerprint of the response body, and its collision resistance determines correctness.

### ETag as Hash Function

For response body $B$, the ETag is:

$$\text{ETag} = H(B) \quad \text{where } H \text{ is a hash function}$$

Weak ETags (semantic equivalence):

$$W/"abc" \implies H_{weak}(B_1) = H_{weak}(B_2) \iff B_1 \equiv_{semantic} B_2$$

Strong ETags (byte-for-byte):

$$"abc" \implies H_{strong}(B_1) = H_{strong}(B_2) \iff B_1 = B_2$$

### Bandwidth Savings

For $N$ requests to a resource that changes every $k$th request, with body size $S$ and ETag/header size $h$:

$$\text{Without ETag}: N \cdot S$$

$$\text{With ETag}: \frac{N}{k} \cdot S + N \cdot h$$

$$\text{Savings} = 1 - \frac{\frac{S}{k} + h}{S} = 1 - \frac{1}{k} - \frac{h}{S}$$

### Worked Example

Resource: 50 KB JSON, changes every 10th request, header overhead 200 bytes:

$$\text{Savings} = 1 - \frac{1}{10} - \frac{0.2}{50} = 1 - 0.1 - 0.004 = 89.6\%$$

Over 1000 requests: $1000 \times 50\text{KB} = 50\text{MB}$ vs $100 \times 50\text{KB} + 1000 \times 0.2\text{KB} = 5.2\text{MB}$.

---

## 5. Content Negotiation Quality Factors (Decision Theory)

### The Problem

The HTTP `Accept` header uses quality factors ($q$-values) to express client preferences. The server must select the best representation from available options.

### The Formula

For client preferences $\{(t_i, q_i)\}$ and server-available types $A$:

$$\text{best} = \arg\max_{t \in A} q(t)$$

Where $q(t)$ is determined by matching rules with specificity ordering:

$$\text{exact match} > \text{subtype wildcard} > \text{full wildcard}$$

$$q(\text{application/json}) > q(\text{application/*}) > q(\text{*/*})$$

### Worked Example

Client: `Accept: application/json;q=0.9, application/xml;q=0.8, text/html;q=1.0`

Server supports: `{application/json, application/xml}`

| Type | Client $q$ | Available | Score |
|:---|:---:|:---:|:---:|
| text/html | 1.0 | no | skip |
| application/json | 0.9 | yes | 0.9 |
| application/xml | 0.8 | yes | 0.8 |

$$\text{Selected: application/json} \quad (q = 0.9)$$

---

## 6. HATEOAS State Machines (Automata Theory)

### The Problem

HATEOAS (Hypermedia as the Engine of Application State) means the server drives client state transitions through links. The API becomes a finite state machine where resources are states and links are transitions.

### The Formula

An API as a finite automaton $M = (Q, \Sigma, \delta, q_0, F)$:

- $Q$ = set of resource states (e.g., `order:draft`, `order:submitted`, `order:shipped`)
- $\Sigma$ = set of HTTP operations (links in responses)
- $\delta: Q \times \Sigma \to Q$ = transition function
- $q_0$ = initial state (API root)
- $F$ = terminal states (e.g., `order:delivered`, `order:cancelled`)

### Worked Example: Order Lifecycle

$$\delta(\text{draft}, \text{POST submit}) = \text{submitted}$$
$$\delta(\text{submitted}, \text{POST approve}) = \text{approved}$$
$$\delta(\text{submitted}, \text{POST cancel}) = \text{cancelled}$$
$$\delta(\text{approved}, \text{POST ship}) = \text{shipped}$$
$$\delta(\text{shipped}, \text{POST deliver}) = \text{delivered}$$

Number of valid paths from $q_0$ to any $f \in F$:

$$|\text{paths}| = \sum_{f \in F} |\{\text{walks from } q_0 \text{ to } f\}|$$

For the order FSM: 3 terminal paths (delivered, cancelled-from-submitted, cancelled-from-approved if allowed).

## Prerequisites

- http, openapi, json, networking, caching, load-balancing
