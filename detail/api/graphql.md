# The Mathematics of GraphQL — Query Complexity and Resolution

> *A GraphQL query is a tree-structured selection over a type graph. Every query has measurable complexity determined by its depth, breadth, and the cardinality of list fields. Understanding these properties formally lets you set meaningful rate limits, predict database load, and prevent denial-of-service through query abuse.*

---

## 1. Query Complexity Analysis (Graph Theory)

### The Problem

A GraphQL query is a subtree of the schema's type graph. Each field selection adds cost, and list fields multiply the cost of their children by their expected cardinality. We need a formula to assign a numeric complexity score to any query before executing it.

### The Formula

For a query tree $Q$ with nodes (field selections), the complexity is:

$$C(Q) = \sum_{f \in Q} w_f \cdot \prod_{a \in \text{ancestors}(f)} m_a$$

Where $w_f$ is the weight of field $f$ (default 1), and $m_a$ is the multiplicity (list cardinality) of each ancestor list field.

For a simple linear path with no lists: $C = \sum w_f$.

For a field nested inside a list of cardinality $n$: each child's effective cost is multiplied by $n$.

### Worked Examples

Query: `{ users(first: 10) { name posts(first: 5) { title comments(first: 20) { body } } } }`

| Field | Weight | Ancestor Multiplicities | Effective Cost |
|:---|:---:|:---:|:---:|
| `users` | 1 | - | $1$ |
| `name` | 1 | $10$ | $10$ |
| `posts` | 1 | $10$ | $10$ |
| `title` | 1 | $10 \times 5$ | $50$ |
| `comments` | 1 | $10 \times 5$ | $50$ |
| `body` | 1 | $10 \times 5 \times 20$ | $1000$ |

$$C_{total} = 1 + 10 + 10 + 50 + 50 + 1000 = 1121$$

With a complexity limit of 1000, this query would be rejected.

---

## 2. The N+1 Problem (Combinatorial Explosion)

### The Problem

Without batching, resolving a list of $N$ parent records where each has a child relation results in $1 + N$ database queries: one for the list, then one per item. With $k$ levels of nesting, this becomes exponential.

### The Formula

Without batching, for $k$ nested list fields with cardinalities $n_1, n_2, \ldots, n_k$:

$$Q_{unbatched} = 1 + n_1 + n_1 \cdot n_2 + n_1 \cdot n_2 \cdot n_3 + \cdots = \sum_{i=0}^{k} \prod_{j=1}^{i} n_j$$

With DataLoader batching (one batch query per level):

$$Q_{batched} = k + 1$$

### Worked Examples

| Nesting | Cardinalities | Unbatched Queries | Batched Queries |
|:---|:---|:---:|:---:|
| users -> posts | $n_1 = 50$ | $1 + 50 = 51$ | $2$ |
| users -> posts -> comments | $50, 10$ | $1 + 50 + 500 = 551$ | $3$ |
| users -> posts -> comments -> likes | $50, 10, 20$ | $1 + 50 + 500 + 10000 = 10551$ | $4$ |

$$\text{Reduction factor} = \frac{Q_{unbatched}}{Q_{batched}} = \frac{10551}{4} \approx 2638\times$$

---

## 3. DataLoader Batch Efficiency (Amortized Analysis)

### The Problem

DataLoader collects individual `.load(key)` calls within a single tick of the event loop and dispatches them as a single batch. The efficiency depends on the deduplication rate and batch size.

### The Formula

For $R$ total resolver calls requesting $K$ unique keys:

$$\text{Dedup ratio} = 1 - \frac{K}{R}$$

$$\text{Batch query cost} = T_{overhead} + K \cdot T_{per\_key}$$

Without batching:

$$T_{unbatched} = R \cdot (T_{overhead} + T_{per\_key})$$

With batching:

$$T_{batched} = T_{overhead} + K \cdot T_{per\_key}$$

### Worked Example

50 posts referencing 12 unique authors, $T_{overhead} = 5\text{ms}$, $T_{per\_key} = 0.1\text{ms}$:

$$T_{unbatched} = 50 \times (5 + 0.1) = 255\text{ms}$$
$$T_{batched} = 5 + 12 \times 0.1 = 6.2\text{ms}$$
$$\text{Speedup} = \frac{255}{6.2} \approx 41\times$$
$$\text{Dedup ratio} = 1 - \frac{12}{50} = 76\%$$

---

## 4. Query Depth and Rate Limiting (Resource Allocation)

### The Problem

Query depth and complexity must be bounded to prevent resource exhaustion. We need a formal model to set meaningful limits based on available server resources.

### The Formula

Maximum sustainable query rate given a complexity budget:

$$R_{max} = \frac{B_{total}}{C_{avg} \cdot T_{window}}$$

Where $B_{total}$ is total complexity budget per time window $T_{window}$, and $C_{avg}$ is average query complexity.

For a per-client rate limit with complexity weighting:

$$\text{tokens\_consumed}(q) = \max(1, C(q))$$

$$\text{allowed} \iff \sum_{q \in window} \text{tokens\_consumed}(q) \leq B_{client}$$

### Worked Example

Server capacity: 10,000 complexity units per second. Client budget: 1,000 per minute.

| Query Type | Complexity | Max Queries/Min |
|:---|:---:|:---:|
| Simple lookup | 5 | $1000 / 5 = 200$ |
| List with nested | 100 | $1000 / 100 = 10$ |
| Deep analytics | 500 | $1000 / 500 = 2$ |

---

## 5. Federation Composition (Category Theory)

### The Problem

Apollo Federation composes multiple subgraph schemas into a single supergraph. Entity resolution across subgraphs requires reference resolvers and key fields. The composition must be conflict-free.

### The Formula

For $n$ subgraphs, each providing a set of types $T_i$, the supergraph type set is:

$$T_{super} = \bigcup_{i=1}^{n} T_i$$

For an entity type $E$ with key $K$ present in subgraphs $S_1, S_2, \ldots, S_m$:

$$\text{fields}(E) = \bigcup_{i=1}^{m} \text{fields}_i(E)$$

The resolution plan for a query touching $m$ subgraphs requires:

$$\text{fetches} \leq m \quad \text{(parallel)} \quad \text{or} \quad \leq m \cdot d \quad \text{(sequential, depth } d\text{)}$$

### Worked Example

3 subgraphs: Users (id, name, email), Orders (id, userId, total), Reviews (id, userId, rating).

Query: `{ user(id: "1") { name orders { total } reviews { rating } } }`

$$\text{Fetch 1: Users subgraph} \to \{name\}$$
$$\text{Fetch 2 (parallel): Orders} \to \{total\}, \text{Reviews} \to \{rating\}$$

$$T_{total} = T_1 + \max(T_2, T_3)$$

With $T_1 = 10\text{ms}$, $T_2 = 15\text{ms}$, $T_3 = 12\text{ms}$:

$$T_{total} = 10 + 15 = 25\text{ms} \quad (\text{not } 10 + 15 + 12 = 37\text{ms})$$

---

## 6. Schema Evolution Safety (Type Theory)

### The Problem

Determining whether a schema change is backward-compatible requires checking that every valid query against the old schema remains valid against the new schema.

### The Formula

A change is safe if the new schema's query space is a superset of the old:

$$\text{safe} \iff \forall q \in \text{Valid}(S_{old}): q \in \text{Valid}(S_{new})$$

For field-level changes on output types:

$$\text{adding field: always safe (new fields are not selected by old queries)}$$
$$\text{removing field: } \text{BREAKING} \iff \exists q \text{ selecting that field}$$

For argument changes on fields:

$$\text{adding required arg: BREAKING}$$
$$\text{adding optional arg: safe}$$

### Breaking Change Summary

| Change | Output Type | Input Type |
|:---|:---:|:---:|
| Add field | safe | N/A |
| Remove field | BREAKING | safe |
| Add required arg | BREAKING | BREAKING |
| Add optional arg | safe | safe |
| Change type (narrowing) | BREAKING | safe |
| Change type (widening) | safe | BREAKING |
| Remove enum value | BREAKING (output) | safe (input) |
| Add enum value | safe (output) | BREAKING (input) |

## Prerequisites

- rest-api, openapi, graph-theory, websockets, type-systems
