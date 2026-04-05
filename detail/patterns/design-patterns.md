# The Mathematics of Design Patterns — Coupling, Cohesion, and Complexity

> *Software design patterns are not merely aesthetic preferences — they are strategies for minimizing coupling, maximizing cohesion, and reducing cyclomatic complexity. Graph theory, information theory, and abstract algebra provide the formal underpinnings.*

---

## 1. Pattern Composition Algebra (Category Theory)

### The Problem

How do we formally reason about composing design patterns? When does applying pattern A then pattern B yield a coherent design?

### The Formula

Patterns can be modeled as morphisms in a category where objects are software designs and morphisms are pattern applications:

$$f: D_1 \rightarrow D_2$$

Composition: If $f: D_1 \rightarrow D_2$ and $g: D_2 \rightarrow D_3$, then $g \circ f: D_1 \rightarrow D_3$.

The Decorator pattern satisfies monoid laws under composition:

$$\text{id} \circ f = f = f \circ \text{id} \quad \text{(identity)}$$
$$(f \circ g) \circ h = f \circ (g \circ h) \quad \text{(associativity)}$$

Where $\text{id}$ is the identity decorator (pass-through). This means decorators can be freely stacked in any grouping:

$$\text{Logging}(\text{Auth}(\text{RateLimit}(handler)))$$

The Strategy pattern forms a set of interchangeable morphisms from a common domain — a **hom-set** $\text{Hom}(D, D')$ where each strategy is a distinct morphism with the same signature.

### Worked Examples

HTTP middleware chain in Go: each middleware has type `func(http.Handler) http.Handler`. This is an endomorphism on `http.Handler`. The set of all such middleware forms a monoid under composition with `http.HandlerFunc(func(w, r) { next.ServeHTTP(w, r) })` as identity.

Composing 3 middleware: `Logging(Auth(RateLimit(h)))`. By associativity: `(Logging . Auth) . RateLimit = Logging . (Auth . RateLimit)`. The grouping does not affect behavior.

---

## 2. Coupling and Cohesion Metrics (Package Health)

### The Problem

How do we measure whether a codebase has good structure? Intuitive notions of "clean code" need quantifiable metrics.

### The Formula

Robert C. Martin's package metrics:

**Afferent Coupling** $C_a$: Number of external packages that depend on this package (incoming dependencies).

**Efferent Coupling** $C_e$: Number of external packages this package depends on (outgoing dependencies).

**Instability**:

$$I = \frac{C_e}{C_a + C_e}$$

Where $I \in [0, 1]$. $I = 0$ means maximally stable (many dependents, hard to change). $I = 1$ means maximally unstable (depends on many, no dependents).

**Abstractness**:

$$A = \frac{N_a}{N_c}$$

Where $N_a$ = number of abstract classes/interfaces, $N_c$ = total number of classes/types.

**Distance from Main Sequence**:

$$D = |A + I - 1|$$

Ideal packages lie on the line $A + I = 1$. Packages far from this line are either:
- **Zone of Pain** (low $A$, low $I$): Concrete and stable — painful to change
- **Zone of Uselessness** (high $A$, high $I$): Abstract but nobody uses them

### Worked Examples

Package `database/sql` in Go: $C_a$ is very high (many packages depend on it), $C_e$ is low. $I \approx 0.05$ (very stable). $A$ is high (mostly interfaces: `Driver`, `Scanner`, `Valuer`). $D \approx |0.8 + 0.05 - 1| = 0.15$ — close to ideal.

Package `cmd/myapp/main`: $C_a = 0$ (nothing depends on main), $C_e$ is high. $I = 1.0$ (maximally unstable — correct for an application entry point). $A = 0$ (all concrete). $D = |0 + 1 - 1| = 0$ — on the main sequence.

---

## 3. Dependency Inversion Graph Theory (Acyclic Dependencies)

### The Problem

Circular dependencies create build problems, testing difficulties, and coupled modules. How do we formalize and detect them?

### The Formula

The dependency graph $G = (V, E)$ where vertices are packages/modules and edges are import relationships must be a **Directed Acyclic Graph** (DAG).

A cycle exists if there is a path $v_1 \rightarrow v_2 \rightarrow \ldots \rightarrow v_k \rightarrow v_1$.

**Detection**: Topological sort succeeds if and only if $G$ is a DAG. Using Kahn's algorithm:

1. Compute in-degree for each vertex
2. Enqueue all vertices with in-degree 0
3. Dequeue a vertex, remove its edges, enqueue newly zero in-degree vertices
4. If not all vertices processed, a cycle exists

**Breaking cycles** with Dependency Inversion: Extract an interface $I$ at the cycle's weakest coupling point. If $A \rightarrow B \rightarrow A$, create interface $I$ in package $A$ (or a new package), have $B$ implement $I$, and $A$ depend on $I$ instead of $B$.

Cycle count in graph: The number of distinct cycles can be exponential in $|V|$. However, the minimum number of edges to remove to break all cycles (Minimum Feedback Arc Set) is NP-hard in general.

### Worked Examples

Three packages with cycle: `auth -> user -> notification -> auth`.

Resolution: Create `notifier` interface in a shared package. `auth` depends on `notifier` interface. `notification` implements `notifier`. Dependency chain becomes: `auth -> notifier <- notification`, `user -> notification`. The cycle is broken.

---

## 4. Cyclomatic Complexity Reduction (Pattern Benefits)

### The Problem

Complex conditional logic is error-prone and hard to test. How do design patterns reduce measurable complexity?

### The Formula

McCabe's cyclomatic complexity for a control flow graph $G$:

$$M = E - N + 2P$$

Where $E$ = edges, $N$ = nodes, $P$ = connected components (usually 1).

Equivalently, $M = D + 1$ where $D$ is the number of decision points (if, case, for, while, &&, ||).

**Strategy pattern** reduces complexity by replacing conditionals with polymorphism. A function with $n$ conditional branches has $M = n + 1$. After applying Strategy:

- The dispatch function: $M = 1$ (no conditionals — dynamic dispatch)
- Each strategy implementation: $M = 1$ (single path)
- Total: $n + 1$ separate units each with $M = 1$, vs one unit with $M = n + 1$

The total McCabe number is the same, but **per-unit complexity** drops from $n+1$ to $1$, making each piece independently testable.

### Worked Examples

Before Strategy (one function):
```
func price(order Order) float64 {
    switch order.Type {        // +1
    case "standard": ...       // +1
    case "premium": ...        // +1
    case "enterprise": ...     // +1
    case "free": ...           // +1
    }
}
// M = 5
```

After Strategy: 4 implementations with $M = 1$ each, plus a factory with $M = 4$ (the switch). But the pricing logic per tier is isolated and testable. If each case had internal conditionals ($M = 3$ each), the original would be $M = 13$, while the refactored version has units of $M \leq 3$.

---

## 5. Open-Closed Principle Formalization (Extension Without Modification)

### The Problem

How do we define "open for extension, closed for modification" precisely enough to reason about it?

### The Formula

A module $M$ is **closed** for modification if its source code is not changed when adding new behavior. $M$ is **open** for extension if new behaviors can be added by creating new code that depends on $M$.

Formally, let $B(M, t)$ be the behavior of module $M$ at time $t$, and $S(M, t)$ be its source code. The OCP holds if:

$$B(M, t_2) \supset B(M, t_1) \implies S(M, t_2) = S(M, t_1)$$

New behavior is added without changing existing source. This is achieved through:

1. **Polymorphism**: Define interface $I$ in $M$. New implementations of $I$ extend behavior without modifying $M$.
2. **Plugin architecture**: $M$ discovers and loads extensions at runtime.

The **Expression Problem** (Wadler, 1998) shows this is fundamentally hard: adding new data types (rows) is easy with interfaces; adding new operations (columns) is easy with pattern matching. Doing both without modifying existing code requires advanced techniques (visitor pattern, type classes, or expression algebras).

### Worked Examples

Adding a new `PayPal` payment method to a system:

- **Closed design**: `PaymentProcessor` interface with `Process(amount)` method. Existing code calls `processor.Process(amount)`. Adding PayPal = new struct implementing `PaymentProcessor`. Zero changes to existing code.
- **Violated OCP**: `switch payment.Type { case "stripe": ..., case "paypal": ... }` — must modify existing function to add PayPal.

---

## Prerequisites

- Basic graph theory (directed graphs, cycles, DAGs)
- Familiarity with GoF design patterns
- Understanding of interfaces and polymorphism
- Elementary set theory and functions

## Complexity

| Metric | Formula | Ideal Range | Tool |
|---|---|---|---|
| Cyclomatic Complexity | $E - N + 2P$ | 1-10 per function | `gocyclo`, `radon` |
| Instability | $C_e / (C_a + C_e)$ | Varies by layer | `go-arch-lint` |
| Abstractness | $N_a / N_c$ | 0.3-0.7 for libraries | Manual analysis |
| Main Sequence Distance | $\|A + I - 1\|$ | < 0.3 | `go-arch-lint` |
| Dependency Cycle Detection | Topological sort | 0 cycles | `go vet`, `depguard` |
