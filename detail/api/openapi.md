# The Mathematics of OpenAPI — Schema Composition and Constraint Validation

> *An OpenAPI specification is fundamentally a formal grammar for HTTP APIs. Its schema composition operators (allOf, oneOf, anyOf) map directly to set-theoretic operations, while validation constraints define membership functions over value spaces. Understanding these mathematical foundations reveals why certain schema designs fail and how to reason about API compatibility.*

---

## 1. Schema Set Theory (Combinatorics)

### The Problem

OpenAPI schemas define value spaces -- sets of valid JSON values. The composition keywords `allOf`, `oneOf`, and `anyOf` combine schemas using set-theoretic intersection, exclusive-or, and union. Misunderstanding these operations leads to impossible schemas or overly permissive validation.

### The Formulas

For schemas $A$ and $B$, each defining a set of valid values:

$$\text{allOf}(A, B) = A \cap B$$

$$\text{anyOf}(A, B) = A \cup B$$

$$\text{oneOf}(A, B) = (A \setminus B) \cup (B \setminus A) = A \triangle B$$

The `not` keyword is set complement:

$$\text{not}(A) = U \setminus A$$

where $U$ is the universal set of all JSON values.

### Worked Examples

| Composition | Schema A (integers 1-10) | Schema B (even integers) | Result Set |
|:---|:---|:---|:---|
| `allOf(A, B)` | $\{1..10\}$ | $\{2,4,6,...\}$ | $\{2,4,6,8,10\}$ |
| `anyOf(A, B)` | $\{1..10\}$ | $\{2,4,6,...\}$ | $\{1,2,3,...,10,12,14,...\}$ |
| `oneOf(A, B)` | $\{1..10\}$ | $\{2,4,6,...\}$ | $\{1,3,5,7,9,12,14,16,...\}$ |

### The Empty Schema Trap

When `allOf` combines contradictory constraints:

$$\text{allOf}(\{type: string\}, \{type: integer\}) = \emptyset$$

This schema is valid YAML but matches no possible value. No validator will warn you at author time.

---

## 2. Constraint Validation (Predicate Logic)

### The Problem

Each schema keyword is a predicate function. A value is valid when the conjunction of all predicates holds. For numeric constraints, this defines a bounded interval.

### The Formula

For a numeric schema with constraints:

$$V(x) = (x \geq \text{minimum}) \wedge (x \leq \text{maximum}) \wedge (x \mod \text{multipleOf} = 0)$$

The number of valid integers in a bounded `multipleOf` constraint:

$$N = \left\lfloor \frac{\text{maximum}}{\text{multipleOf}} \right\rfloor - \left\lceil \frac{\text{minimum}}{\text{multipleOf}} \right\rceil + 1$$

### Worked Examples

| Constraint | minimum | maximum | multipleOf | Valid Count |
|:---|:---:|:---:|:---:|:---:|
| Page size | 10 | 100 | 10 | $\lfloor 100/10 \rfloor - \lceil 10/10 \rceil + 1 = 10$ |
| Batch size | 1 | 1000 | 50 | $\lfloor 1000/50 \rfloor - \lceil 1/50 \rceil + 1 = 20$ |
| Port range | 1024 | 65535 | 1 | $65535 - 1024 + 1 = 64512$ |

### String Pattern Cardinality

For a string with `minLength: m`, `maxLength: M`, and alphabet size $|\Sigma|$:

$$|S| = \sum_{k=m}^{M} |\Sigma|^k = |\Sigma|^m \cdot \frac{|\Sigma|^{M-m+1} - 1}{|\Sigma| - 1}$$

A `format: uuid` has exactly $|\Sigma| = 16$, effective length $k = 32$ hex digits:

$$|UUID_{v4}| = 2^{122} \approx 5.3 \times 10^{36}$$

---

## 3. API Compatibility (Partial Orders)

### The Problem

When evolving an API across versions, we need a formal definition of backward compatibility. A new schema version $B$ is backward-compatible with $A$ if every value valid under $A$ is also valid under $B$.

### The Formula

Backward compatibility as a subset relation:

$$A \subseteq B \implies B \text{ is backward-compatible with } A$$

For request bodies (contravariant -- server must accept at least what old clients send):

$$\text{compatible}(v_{new}) \iff V_{old}^{request} \subseteq V_{new}^{request}$$

For response bodies (covariant -- new server output must be understandable by old clients):

$$\text{compatible}(v_{new}) \iff V_{new}^{response} \subseteq V_{old}^{response}$$

### Breaking Change Detection Rules

| Change | Request Body | Response Body |
|:---|:---:|:---:|
| Add required field | BREAKING | safe |
| Remove required field | safe | BREAKING |
| Add optional field | safe | safe (if clients ignore unknown) |
| Narrow enum values | safe | BREAKING |
| Widen enum values | BREAKING | safe |
| Tighten `maximum` | BREAKING | safe |
| Loosen `maximum` | safe | BREAKING |

### Worked Example: Enum Evolution

Old schema: `status: enum [active, inactive]`
New schema: `status: enum [active, inactive, archived]`

$$V_{old} = \{\text{active}, \text{inactive}\}$$
$$V_{new} = \{\text{active}, \text{inactive}, \text{archived}\}$$

In response: $V_{new} \not\subseteq V_{old}$ -- BREAKING (old clients see unknown `archived`).
In request: $V_{old} \subseteq V_{new}$ -- safe (old clients send values new server accepts).

---

## 4. Discriminator Routing (Decision Theory)

### The Problem

Polymorphic schemas with `oneOf` and a `discriminator` create a decision function mapping a property value to exactly one sub-schema. The discriminator must be a bijection from tag values to schemas.

### The Formula

For discriminator property $d$ with $n$ sub-schemas:

$$f: D \to \{S_1, S_2, \ldots, S_n\}, \quad |D| = n, \quad f \text{ is bijective}$$

Validation cost without discriminator (must try all schemas):

$$T_{no\_disc} = O(n \cdot C_{validate})$$

Validation cost with discriminator (direct lookup):

$$T_{disc} = O(1 + C_{validate})$$

### Worked Example

A `Shape` with 5 sub-schemas, each taking 2ms to validate against:

$$T_{no\_disc} = 5 \times 2\text{ms} = 10\text{ms (worst case, tries all)}$$
$$T_{disc} = 0.01\text{ms (lookup)} + 2\text{ms} = 2.01\text{ms}$$

$$\text{Speedup} = \frac{10}{2.01} \approx 4.97\times$$

---

## 5. Code Generation Complexity (Graph Theory)

### The Problem

An OpenAPI generator must resolve all `$ref` pointers, build a dependency graph of schemas, and generate types in topological order. Circular references create cycles that require special handling (pointers, lazy initialization).

### The Formula

For a spec with $S$ schemas and $R$ ref edges, the generation graph $G = (S, R)$ must be checked for cycles:

$$\text{Cycle detection: DFS in } O(|S| + |R|)$$

If the graph is a DAG (directed acyclic graph), topological sort yields generation order:

$$\text{Topo sort: } O(|S| + |R|)$$

Total generated types for a spec with $P$ paths, $O$ operations, $S$ schemas:

$$T_{types} = S + O_{request} + O_{response} + P_{params}$$

### Worked Example

A spec with 50 schemas, 30 operations, each with request + response:

$$T_{types} = 50 + 30 + 30 + 15 = 125 \text{ generated types}$$

Lines of code estimate (Go, ~40 lines per type):

$$LOC \approx 125 \times 40 = 5000 \text{ lines}$$

---

## 6. Spec Size and Performance (Information Theory)

### The Problem

Large OpenAPI specs affect tooling performance -- parsing time, memory consumption, and IDE responsiveness. We can model spec complexity to predict when splitting becomes necessary.

### The Formula

Spec complexity metric:

$$C_{spec} = P \cdot \bar{O}_P \cdot \bar{S}_O$$

Where $P$ = number of paths, $\bar{O}_P$ = average operations per path, $\bar{S}_O$ = average schemas referenced per operation.

YAML parse time (empirical, linear in document size):

$$T_{parse} \approx 0.5\text{ms/KB} \times S_{KB}$$

### Worked Examples

| Spec | Paths | Ops/Path | Schemas/Op | Complexity | Parse Time |
|:---|:---:|:---:|:---:|:---:|:---:|
| Small API | 10 | 2 | 3 | 60 | ~5ms |
| Medium API | 50 | 3 | 5 | 750 | ~50ms |
| Large API (Stripe) | 200 | 2.5 | 8 | 4000 | ~500ms |

Rule of thumb: split specs when $C_{spec} > 2000$ or file size exceeds 500 KB.

## Prerequisites

- json-schema, rest-api, yaml, http-status-codes, api-gateway
