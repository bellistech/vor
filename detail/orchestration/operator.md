# The Mathematics of Operators — Reconciliation Fixed Points and Controller Automata

> *A Kubernetes operator is a level-triggered control loop that drives system state toward a fixed point defined by a custom resource, where the reconciliation function is an idempotent endofunction on the cluster state space and convergence follows from monotonic progress toward the desired equilibrium.*

---

## 1. Reconciliation as Fixed-Point Iteration (Fixed-Point Theory)

### State Space

The cluster state is an element of the state space:

$$s \in \mathcal{S} = \mathcal{S}_{\text{CR}} \times \mathcal{S}_{\text{owned}}$$

Where $\mathcal{S}_{\text{CR}}$ is the custom resource spec and $\mathcal{S}_{\text{owned}}$ is the state of owned resources (StatefulSets, Services, ConfigMaps, etc.).

### Reconcile as Endofunction

The reconcile function transforms cluster state:

$$R: \mathcal{S} \rightarrow \mathcal{S}$$

The desired state $s^*$ is a fixed point:

$$R(s^*) = s^*$$

### Convergence

Starting from any state $s_0$, iterative reconciliation converges:

$$s_{n+1} = R(s_n)$$

$$\lim_{n \to \infty} s_n = s^*$$

Convergence is guaranteed if $R$ is a contraction mapping on a complete metric space:

$$d(R(s_1), R(s_2)) \leq k \cdot d(s_1, s_2) \quad \text{for some } k < 1$$

In practice, operators typically converge in 1-3 reconciliation cycles:

| Starting State | Cycles to Converge | Reason |
|:---|:---:|:---|
| CR created, no owned resources | 1-2 | Create all resources |
| Owned resource deleted | 1 | Recreate missing resource |
| CR spec changed | 1-2 | Update owned resources |
| Underlying state drifted | 1 | Correct drift |

### Idempotency Requirement

$$R(R(s)) = R(s) \quad \forall s \in \mathcal{S}$$

This means $R$ is idempotent: applying it twice has the same effect as applying it once. This is critical because the controller may reconcile multiple times for the same event.

---

## 2. Level-Triggered vs Edge-Triggered (Signal Theory)

### Edge-Triggered (Anti-Pattern)

Edge-triggered systems react to state transitions:

$$\text{action}(t) = f(\Delta s(t)) = f(s(t) - s(t - 1))$$

Problems:
- Missed events cause permanent drift
- Event ordering matters
- State reconstruction requires complete event history

### Level-Triggered (Operator Pattern)

Level-triggered systems react to current state:

$$\text{action}(t) = f(s_{\text{desired}} - s_{\text{actual}}(t))$$

Properties:
- Self-healing: any missed event is corrected on next reconcile
- Order-independent: only current state matters
- Convergent: always drives toward desired state

### Robustness Comparison

For a system with event loss probability $p$:

**Edge-triggered:**
$$P(\text{correct after } n \text{ events}) = (1-p)^n$$

For $p = 0.01$ and $n = 100$: $P = 0.366$ (63.4% chance of drift).

**Level-triggered:**
$$P(\text{correct after reconcile}) = 1 - p_{\text{reconcile\_failure}}$$

For $p_{\text{fail}} = 0.01$: $P = 0.99$ per cycle, and with retries: $P = 1 - p^k$ after $k$ attempts.

---

## 3. Watch and Informer Model (Queueing Theory)

### Event Pipeline

The controller-runtime event pipeline forms a queue:

$$\text{Watch} \xrightarrow{\text{events}} \text{EventHandler} \xrightarrow{\text{requests}} \text{WorkQueue} \xrightarrow{\text{dequeue}} \text{Reconciler}$$

### Work Queue Semantics

The work queue deduplicates by key (namespace/name):

$$Q = \text{Set}(\text{keys})$$

If the same key is enqueued multiple times before processing:

$$\text{enqueue}(k); \text{enqueue}(k) \equiv \text{enqueue}(k)$$

This ensures at-most-once-per-key processing per batch, reducing redundant reconciliation.

### Throughput Model

With $w$ concurrent workers and mean reconciliation time $\bar{T}$:

$$\lambda_{\max} = \frac{w}{\bar{T}}$$

For $w = 1$ worker and $\bar{T} = 100\text{ms}$:

$$\lambda_{\max} = 10 \text{ reconciliations/second}$$

For $w = 8$ workers:

$$\lambda_{\max} = 80 \text{ reconciliations/second}$$

### Rate Limiting

The work queue applies rate limiting to prevent thundering herd:

**Base delay:** $d_0 = 5\text{ms}$
**Max delay:** $d_{\max} = 1000\text{s}$
**Backoff:** $d_n = \min(d_0 \cdot 2^n, d_{\max})$

| Requeue | Delay | Cumulative |
|:---:|:---:|:---:|
| 1 | 5ms | 5ms |
| 2 | 10ms | 15ms |
| 3 | 20ms | 35ms |
| 10 | 2.56s | ~5.1s |
| 17 | 655s | ~1310s |
| 18+ | 1000s | capped |

---

## 4. Owner References as DAG (Graph Theory)

### Ownership Graph

Owner references form a directed acyclic graph:

$$G_{\text{own}} = (V, E)$$

Where $V$ = all resources and $E$ = owner references.

For a Database CR:

```
Database (CR)
├── StatefulSet
│   └── Pod (3 replicas)
├── Service (headless)
├── Service (client)
├── ConfigMap
└── Secret
```

$$|V| = 1 + 1 + 3 + 2 + 1 + 1 = 9$$
$$|E| = 8$$

### Garbage Collection

When the root owner is deleted, cascading deletion propagates through the DAG:

$$\text{delete}(v) \implies \forall (v, u) \in E: \text{delete}(u) \quad \text{(foreground cascading)}$$

Deletion order follows reverse topological sort:

$$\text{delete\_order} = \text{reverse}(\text{topo\_sort}(G_{\text{own}}))$$

### Adoption and Disownment

An operator must handle orphaned resources:

$$\text{orphans} = \{r \in R : \text{matches\_selector}(r) \wedge \text{owner}(r) = \emptyset\}$$

Adoption: $\text{owner}(r) \leftarrow \text{CR}$

Disownment: $\text{owner}(r) \leftarrow \emptyset$ when $r$ no longer matches the CR.

---

## 5. CRD Schema as Type System (Type Theory)

### OpenAPI v3 Schema

A CRD schema defines a type for custom resources:

$$\tau_{\text{Database}} = \{\text{spec}: \tau_{\text{spec}}, \text{status}: \tau_{\text{status}}\}$$

$$\tau_{\text{spec}} = \{\text{engine}: \text{enum}(\text{postgres}, \text{mysql}), \text{replicas}: \text{int}[1,7], \text{storage}: \tau_{\text{storage}}\}$$

### Validation as Predicate

Schema validation is a predicate:

$$\text{valid}(r, \tau) = \bigwedge_{f \in \text{fields}(\tau)} \text{type\_check}(r.f, \tau.f)$$

For structured types:

$$\text{type\_check}(v, \text{int}[a,b]) = v \in \mathbb{Z} \wedge a \leq v \leq b$$

$$\text{type\_check}(v, \text{enum}(e_1, \ldots, e_n)) = v \in \{e_1, \ldots, e_n\}$$

$$\text{type\_check}(v, \text{pattern}(p)) = v \text{ matches regex } p$$

### Version Conversion

CRD versioning requires conversion between schema versions:

$$\phi_{v1 \to v2}: \tau_{v1} \rightarrow \tau_{v2}$$

$$\phi_{v2 \to v1}: \tau_{v2} \rightarrow \tau_{v1}$$

Round-trip property (lossless conversion):

$$\phi_{v2 \to v1}(\phi_{v1 \to v2}(r)) = r \quad \text{(ideally)}$$

In practice, hub-and-spoke conversion through an internal type:

$$v1 \xrightarrow{\phi_1} \text{hub} \xrightarrow{\phi_2^{-1}} v2$$

---

## 6. Finalizer Protocol (Protocol Theory)

### State Machine with Finalizers

Finalizers extend the deletion state machine:

Without finalizer:
$$\text{exists} \xrightarrow{\text{delete}} \text{gone}$$

With finalizer:
$$\text{exists} \xrightarrow{\text{delete}} \text{deleting}(f \in F) \xrightarrow{\text{remove } f} \cdots \xrightarrow{F = \emptyset} \text{gone}$$

### Liveness Property

A finalizer blocks deletion until removed. If the controller is down:

$$F \neq \emptyset \wedge \text{controller\_down} \implies \text{resource stuck in Terminating}$$

This is a liveness hazard. The finalizer protocol must ensure:

$$\forall f \in F: \exists \text{controller that removes } f$$

### Cleanup Ordering

For external resources $E = \{e_1, e_2, \ldots, e_k\}$ with dependencies:

$$\text{cleanup\_order} = \text{reverse\_topo\_sort}(\text{dep\_graph}(E))$$

Total cleanup time:

$$T_{\text{cleanup}} = \sum_{i=1}^{k} T(e_i) \quad \text{(sequential)}$$

$$T_{\text{cleanup}} = \text{critical\_path}(\text{dep\_graph}(E)) \quad \text{(parallel)}$$

---

## 7. Status Conditions as Finite State Lattice (Lattice Theory)

### Condition Model

Status conditions form a set of independent boolean predicates:

$$\text{conditions} = \{(t_i, s_i, r_i, m_i) : t_i \in \text{Types}, s_i \in \{\text{True}, \text{False}, \text{Unknown}\}\}$$

Each condition type is a three-valued proposition:

$$s \in \{\text{True}, \text{False}, \text{Unknown}\}$$

Ordered: $\text{Unknown} \prec \text{False} \prec \text{True}$ (for positive conditions).

### Standard Condition Types

| Condition Type | True When | False When |
|:---|:---|:---|
| Available | Service is accessible | Not accessible |
| Ready | All components healthy | Some unhealthy |
| Degraded | Operating with reduced capacity | Fully operational |
| Progressing | Actively converging | Stable |
| Reconciled | Desired = actual | Drift detected |

### Aggregate Readiness

Overall readiness as conjunction:

$$\text{Ready} = \text{Available} \wedge \neg\text{Degraded} \wedge \neg\text{Progressing}$$

Mapping to phase:

$$\text{phase} = \begin{cases} \text{Running} & \text{if Ready} = \text{True} \\ \text{Degraded} & \text{if Available} \wedge \text{Degraded} \\ \text{Creating} & \text{if Progressing} \wedge \neg\text{Available} \\ \text{Failed} & \text{if } \neg\text{Available} \wedge \neg\text{Progressing} \end{cases}$$

---

## Prerequisites

fixed-point-theory, control-theory, graph-theory, queueing-theory, type-theory, lattice-theory

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| Single reconciliation | $O(R)$ — R = owned resources | $O(R)$ |
| Watch event processing | $O(1)$ — enqueue key | $O(Q)$ — queue size |
| Garbage collection (cascade) | $O(V + E)$ — ownership DAG | $O(V)$ |
| Schema validation | $O(F \times D)$ — fields x depth | $O(1)$ |
| Version conversion | $O(F)$ — field mapping | $O(F)$ |
| Finalizer cleanup | $O(k)$ — external resources | $O(1)$ |
| Owner ref lookup | $O(1)$ — indexed | $O(V)$ — index size |
