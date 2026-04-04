# The Mathematics of CORS — Origin Algebra and Preflight Decision Theory

> *CORS is a policy enforcement automaton where the browser evaluates origin tuples against server-declared access sets, with preflight caching as a TTL-bounded memoization layer over the decision function that maps request characteristics to allow/deny outcomes.*

---

## 1. Origin as Algebraic Tuple (Set Theory)

### Origin Definition

An origin is a 3-tuple from RFC 6454:

$$O = (\text{scheme}, \text{host}, \text{port})$$

Two origins are same-origin if and only if:

$$O_1 =_{\text{origin}} O_2 \iff s_1 = s_2 \wedge h_1 = h_2 \wedge p_1 = p_2$$

This is an equivalence relation (reflexive, symmetric, transitive) that partitions all URLs into origin equivalence classes.

### Origin Space

The cardinality of the origin space:

$$|\mathcal{O}| = |\text{Schemes}| \times |\text{Hosts}| \times |\text{Ports}|$$

With practical bounds:
- $|\text{Schemes}| \approx 3$ (http, https, ws/wss)
- $|\text{Hosts}| \leq 2^{253}$ (DNS name limit: 253 characters)
- $|\text{Ports}| = 65536$

$$|\mathcal{O}| \leq 3 \times 2^{253} \times 2^{16} \approx 2^{271}$$

### Opaque Origins

Some contexts produce opaque (null) origins with no tuple representation:

$$O_{\text{opaque}} = \text{null}$$

$$O_{\text{opaque}} \neq_{\text{origin}} O_{\text{opaque}}$$

Opaque origins are never equal to anything, including themselves (irreflexive).

---

## 2. CORS Decision Function (Decision Theory)

### Request Classification

The browser classifies each cross-origin request using a predicate:

$$\text{simple}(R) = P_{\text{method}}(R) \wedge P_{\text{headers}}(R) \wedge P_{\text{type}}(R)$$

Where:

$$P_{\text{method}}(R) \iff R.\text{method} \in \{\text{GET}, \text{HEAD}, \text{POST}\}$$

$$P_{\text{headers}}(R) \iff R.\text{headers} \subseteq \mathcal{H}_{\text{safe}}$$

$$P_{\text{type}}(R) \iff R.\text{Content-Type} \in \{\text{text/plain}, \text{multipart/form-data}, \text{application/x-www-form-urlencoded}\}$$

The safe header set:

$$\mathcal{H}_{\text{safe}} = \{\text{Accept}, \text{Accept-Language}, \text{Content-Language}, \text{Content-Type}\}$$

### Decision Tree

$$D(R, O_{\text{req}}, O_{\text{target}}) = \begin{cases} \text{ALLOW} & \text{if } O_{\text{req}} =_{\text{origin}} O_{\text{target}} \\ \text{SIMPLE\_CORS} & \text{if } O_{\text{req}} \neq O_{\text{target}} \wedge \text{simple}(R) \\ \text{PREFLIGHT\_CORS} & \text{if } O_{\text{req}} \neq O_{\text{target}} \wedge \neg\text{simple}(R) \end{cases}$$

---

## 3. Preflight as Memoized Decision (Caching Theory)

### Preflight Cache Model

The browser maintains a preflight cache $C$ mapping request signatures to allow decisions:

$$C: (\text{origin}, \text{url}, \text{method}, \text{headers}) \rightarrow (\text{allowed}, \text{expiry})$$

Cache lookup:

$$\text{hit}(R) \iff \exists e \in C : e.\text{key} = \text{sig}(R) \wedge t_{\text{now}} < e.\text{expiry}$$

### Cache TTL Bounds

The `Access-Control-Max-Age` header sets the TTL:

$$\text{TTL} = \min(\text{Max-Age}_{\text{server}}, \text{Max-Age}_{\text{browser}})$$

Browser-imposed caps:

| Browser | Max TTL | Default (no header) |
|:---|:---:|:---:|
| Chrome | 7200s (2h) | 5s |
| Firefox | 86400s (24h) | 5s |
| Safari | 604800s (7d) | 5s |

### Cache Efficiency

For a client making $\lambda$ requests/second to $n$ distinct endpoints:

$$P(\text{cache hit}) = 1 - \frac{n}{\lambda \cdot \text{TTL}}$$

For $\lambda = 10$ req/s, $n = 5$ endpoints, TTL = 3600s:

$$P(\text{hit}) = 1 - \frac{5}{36000} \approx 99.99\%$$

Preflight overhead as fraction of total requests:

$$\text{overhead} = \frac{n}{\lambda \cdot \text{TTL} + n} \approx \frac{n}{\lambda \cdot \text{TTL}}$$

---

## 4. Header Matching as Set Operations (Set Theory)

### Allow-Headers Validation

Server declares allowed headers: $\mathcal{H}_{\text{allowed}}$

Request declares needed headers: $\mathcal{H}_{\text{request}}$

Preflight succeeds if:

$$\mathcal{H}_{\text{request}} \subseteq \mathcal{H}_{\text{allowed}}$$

### Wildcard Semantics

Without credentials:

$$\text{"*"} \equiv \mathcal{H}_{\text{all}} \setminus \{\text{Authorization}\}$$

Note: even wildcard does not include `Authorization` in the Fetch spec.

With credentials (`Access-Control-Allow-Credentials: true`):

$$\text{"*"} \equiv \text{invalid (rejected by browser)}$$

All four wildcard-capable headers must enumerate explicitly:

$$\text{Allow-Origin} \neq *, \quad \text{Allow-Methods} \neq *, \quad \text{Allow-Headers} \neq *, \quad \text{Expose-Headers} \neq *$$

### Expose-Headers Filtering

Without `Access-Control-Expose-Headers`, JavaScript can only read CORS-safelisted response headers:

$$\mathcal{H}_{\text{visible}} = \mathcal{H}_{\text{safelisted}} \cup \mathcal{H}_{\text{exposed}}$$

$$\mathcal{H}_{\text{safelisted}} = \{\text{Cache-Control}, \text{Content-Language}, \text{Content-Length}, \text{Content-Type}, \text{Expires}, \text{Pragma}\}$$

---

## 5. Cache Poisoning Attack (Security Theory)

### Vary Header Necessity

When dynamically reflecting the Origin:

$$\text{response}(O) = \text{Access-Control-Allow-Origin}: O$$

Without `Vary: Origin`, an intermediate cache stores:

$$\text{cache}[\text{url}] = (\text{body}, \{\text{ACAO}: O_1\})$$

A subsequent request from $O_2$ receives the cached response with $O_1$, failing CORS:

$$O_2 \neq O_1 \implies \text{CORS blocked}$$

Or worse, if the cache serves `ACAO: *` to a credentialed request:

$$\text{wildcard} \wedge \text{credentials} \implies \text{browser rejects}$$

### Correct Cache Behavior

With `Vary: Origin`, the cache key becomes:

$$\text{key} = (\text{url}, \text{Origin header value})$$

Cache entries per URL: $|\text{entries}| = |\{O : O \text{ makes requests}\}|$

Cache storage: $O(|\text{URLs}| \times |\text{Origins}|)$

---

## 6. Credential Modes as Trust Levels (Lattice Theory)

### Trust Ordering

CORS defines three credential modes forming a total order:

$$\text{omit} \prec \text{same-origin} \prec \text{include}$$

| Mode | Cookies Sent | ACAO Wildcard | Trust Level |
|:---|:---:|:---:|:---:|
| `omit` | Never | Allowed | Lowest |
| `same-origin` | Same-origin only | Allowed | Medium |
| `include` | Always (cross-origin) | Forbidden | Highest |

### Security Constraint Matrix

The interaction between credentials and wildcards forms a constraint:

$$\text{valid}(C, W) = \neg(C \wedge W)$$

| Allow-Credentials | Allow-Origin: * | Allow-Methods: * | Allow-Headers: * | Result |
|:---:|:---:|:---:|:---:|:---|
| false | OK | OK | OK | Valid |
| true | FAIL | FAIL | FAIL | Must enumerate |
| true | specific | specific | specific | Valid |

---

## 7. Private Network Access (Extension Theory)

### Network Tiers

Private Network Access extends CORS with a network tier hierarchy:

$$\text{public} \prec \text{private} \prec \text{local}$$

Cross-tier requests (e.g., public website accessing localhost) require additional preflight:

$$\text{preflight required} \iff \text{tier}(O_{\text{source}}) \prec \text{tier}(O_{\text{target}})$$

Additional header required:

$$\text{Access-Control-Allow-Private-Network: true}$$

### Tier Classification

| Network | Tier | CIDR |
|:---|:---|:---|
| Internet | public | not RFC 1918 |
| 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16 | private | RFC 1918 |
| 127.0.0.0/8, ::1 | local | Loopback |

Cross-tier attack surface:

$$\text{ASM}_{\text{cross-tier}} = \sum_{i < j} |\text{endpoints}(T_j)| \times |\text{methods}|$$

---

## Prerequisites

set-theory, tuple-algebra, caching-theory, decision-theory, lattice-theory

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| Origin comparison | $O(1)$ — three string comparisons | $O(1)$ |
| Simple request classification | $O(h)$ — h = header count | $O(1)$ |
| Preflight cache lookup | $O(1)$ — hash map | $O(n)$ — n = cached entries |
| Header set validation | $O(h)$ — subset check | $O(h)$ |
| Origin allowlist check | $O(1)$ — hash set | $O(k)$ — k = allowed origins |
| Vary-aware cache | $O(1)$ lookup | $O(u \times o)$ — URLs x origins |
