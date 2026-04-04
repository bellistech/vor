# The Mathematics of OCI — Container Specification Formal Models

> *The OCI specifications define containers through three interlocking formal systems: a runtime state machine over Linux namespace configurations, an image model based on content-addressed merkle DAGs, and a distribution protocol preserving integrity through cryptographic digests.*

---

## 1. Runtime State Machine (Automata Theory)

### Container Lifecycle Automaton

The OCI runtime spec defines a deterministic finite automaton:

$$M = (Q, \Sigma, \delta, q_0, F)$$

$$Q = \{\text{creating}, \text{created}, \text{running}, \text{stopped}\}$$

$$\Sigma = \{\text{create}, \text{start}, \text{kill}, \text{delete}, \text{exit}\}$$

Transition function:

$$\delta(\text{creating}, \text{create\_done}) = \text{created}$$
$$\delta(\text{created}, \text{start}) = \text{running}$$
$$\delta(\text{running}, \text{exit}) = \text{stopped}$$
$$\delta(\text{running}, \text{kill}) = \text{stopped}$$

The `delete` operation removes the container from the runtime but is not a state (it destroys the state machine instance).

### Lifecycle Hook Points

The runtime spec defines hook execution points as events on transitions:

| Hook | Trigger Transition | Timing |
|:---|:---|:---|
| `prestart` | creating $\to$ created | After namespaces, before pivot_root |
| `createRuntime` | creating $\to$ created | After create, in runtime namespace |
| `createContainer` | creating $\to$ created | After create, in container namespace |
| `startContainer` | created $\to$ running | Before user process starts |
| `poststart` | created $\to$ running | After user process starts |
| `poststop` | running $\to$ stopped | After process exits |

---

## 2. Namespace Isolation Model (Set Theory)

### Namespace as Partition

Each namespace type partitions a global resource space into isolated views:

$$N_{\text{type}}: \mathcal{R}_{\text{global}} \rightarrow \mathcal{R}_{\text{view}}$$

| Namespace | Global Resource | Isolation |
|:---|:---|:---|
| PID | Process ID space $\mathbb{Z}^+$ | Separate PID trees |
| Network | Interfaces, routes, iptables | Separate network stacks |
| Mount | Filesystem mount tree | Separate mount hierarchies |
| UTS | Hostname, domain name | Separate identifiers |
| IPC | Shared memory, semaphores | Separate IPC objects |
| User | UID/GID mappings | Separate privilege domains |
| Cgroup | Cgroup hierarchy view | Separate resource views |

### Isolation Strength

The total isolation is the product of namespace isolations:

$$I_{\text{total}} = \prod_{t \in \text{types}} I_t$$

Where $I_t$ is the isolation factor for namespace type $t$.

For $k$ namespaces enabled, the number of isolated resource dimensions:

$$\text{dimensions} = k, \quad k \leq 7$$

### User Namespace UID Mapping

The mapping function $\phi: \text{UID}_{\text{container}} \rightarrow \text{UID}_{\text{host}}$:

$$\phi(u) = \text{hostID} + (u - \text{containerID}) \quad \text{for } \text{containerID} \leq u < \text{containerID} + \text{size}$$

For mapping $(0, 100000, 65536)$:

$$\phi(u) = 100000 + u \quad \text{for } 0 \leq u < 65536$$

This is an affine transformation restricted to a bounded domain.

---

## 3. Image as Merkle DAG (Graph Theory)

### Content-Addressed Structure

An OCI image forms a merkle DAG:

$$G = (V, E)$$

$$V = \{\text{index}, \text{manifest}_1, \ldots, \text{manifest}_p, \text{config}_1, \ldots, \text{config}_p, \text{layer}_1, \ldots, \text{layer}_n\}$$

$$E = \{(\text{index}, \text{manifest}_i), (\text{manifest}_i, \text{config}_i), (\text{manifest}_i, \text{layer}_j)\}$$

Each node is identified by $\text{SHA-256}(\text{content})$.

### Structural Integrity

Any modification to a leaf node cascades digest changes upward:

$$\text{modify}(l_k) \implies \text{change}(d(\text{manifest})) \implies \text{change}(d(\text{index}))$$

Integrity verification is recursive:

$$\text{verify}(G) = \bigwedge_{(u,v) \in E} (d(v) \stackrel{?}{=} \text{ref}(u, v))$$

Total verification time:

$$T_{\text{verify}} = O(|V|) \times T_{\text{hash}}$$

### Layer Deduplication Across Images

Given a set of images $\{I_1, I_2, \ldots, I_m\}$:

$$\text{unique layers} = \left|\bigcup_{i=1}^{m} L(I_i)\right|$$

$$\text{total references} = \sum_{i=1}^{m} |L(I_i)|$$

$$\text{dedup ratio} = 1 - \frac{|\bigcup L(I_i)|}{\sum |L(I_i)|}$$

For images sharing a common base (e.g., debian:bookworm with 3 layers):

| Scenario | Images | Total Refs | Unique | Dedup |
|:---|:---:|:---:|:---:|:---:|
| Same base, 1 app layer | 10 | 40 | 13 | 67.5% |
| Same base, 3 app layers | 10 | 60 | 33 | 45.0% |
| Different bases | 10 | 40 | 40 | 0% |

---

## 4. Overlay Filesystem Algebra (Order Theory)

### Union Mount Semantics

An overlay filesystem computes the union of $n$ read-only lower layers and one read-write upper layer:

$$\text{FS}_{\text{view}} = \text{upper} \cup_{\text{overlay}} \text{lower}_n \cup_{\text{overlay}} \cdots \cup_{\text{overlay}} \text{lower}_1$$

The overlay union $\cup_{\text{overlay}}$ has precedence: upper > lower_n > ... > lower_1.

File resolution:

$$\text{resolve}(p) = \begin{cases} \text{upper}(p) & \text{if } p \in \text{upper} \\ \text{lower}_i(p) & \text{if } p \in \text{lower}_i \wedge p \notin \text{lower}_j \; \forall j > i \wedge p \notin \text{upper} \\ \text{ENOENT} & \text{otherwise} \end{cases}$$

### Copy-on-Write Cost

First write to a file in a lower layer triggers copy-up:

$$T_{\text{first\_write}}(f) = T_{\text{copy}}(f) + T_{\text{write}} = O(\text{size}(f))$$

Subsequent writes:

$$T_{\text{subsequent\_write}}(f) = T_{\text{write}} = O(\text{write size})$$

### Layer Count Impact on Lookup

File lookup traverses layers top-down:

$$T_{\text{lookup}}(p) = O(n) \quad \text{worst case (file not found)}$$

$$T_{\text{lookup}}(p) = O(k) \quad \text{where } k = \text{layer containing } p$$

Kernel optimization: directory entry caching reduces effective lookup to $O(1)$ after first access.

---

## 5. Resource Limits as Constraint Space (Linear Programming)

### Cgroup Resource Model

Container resources define a feasible region in multi-dimensional resource space:

$$\vec{r} \in \mathcal{F} = \{(c, m, p, b) : c \leq C_{\max}, m \leq M_{\max}, p \leq P_{\max}, b \leq B_{\max}\}$$

Where:
- $c$ = CPU usage (millicores or quota/period)
- $m$ = memory usage (bytes)
- $p$ = PID count
- $b$ = block I/O bandwidth

### CPU Quota Model

CPU bandwidth is allocated as:

$$\text{CPU}_{\text{fraction}} = \frac{\text{quota}}{\text{period}}$$

For `quota=50000, period=100000`:

$$\text{CPU} = \frac{50000}{100000} = 0.5 \text{ cores}$$

### Memory Limit Enforcement

When $m > M_{\max}$:

$$\text{OOM} \implies \text{kill}(\text{process with highest OOM score})$$

OOM score:

$$\text{oom\_score}(p) = \frac{\text{RSS}(p)}{\text{total RAM}} \times 1000 + \text{oom\_score\_adj}(p)$$

---

## 6. Distribution Protocol (Protocol Theory)

### Pull Protocol State Machine

$$S_0 \xrightarrow{\text{resolve tag}} S_1 \xrightarrow{\text{fetch manifest}} S_2 \xrightarrow{\text{fetch config}} S_3 \xrightarrow{\text{fetch layers}} S_4$$

### Parallel Layer Download

Layers are independent and can be downloaded in parallel:

$$T_{\text{pull}} = T_{\text{manifest}} + T_{\text{config}} + \max_{i \in [n]} T_{\text{layer}_i}$$

With bandwidth $B$ and $n$ layers of sizes $s_1, s_2, \ldots, s_n$:

**Sequential:** $T_{\text{seq}} = \sum_{i=1}^{n} s_i / B$

**Parallel ($k$ connections):** $T_{\text{par}} = \sum_{i=1}^{n} s_i / (k \cdot B)$ (idealized)

In practice, the largest layer dominates:

$$T_{\text{par}} \approx \max_i(s_i / B)$$

### Push Protocol

Push requires blob existence check + conditional upload:

$$\text{for each blob } b:$$
$$\text{HEAD } b \xrightarrow{200} \text{skip (already exists)}$$
$$\text{HEAD } b \xrightarrow{404} \text{POST (initiate)} \to \text{PUT (upload)}$$

Cross-repository mount optimization:

$$\text{POST /blobs/uploads/?mount=}\langle\text{digest}\rangle\text{\&from=}\langle\text{repo}\rangle$$

This avoids re-uploading blobs that already exist in another repository on the same registry.

---

## 7. Specification Versioning (Formal Systems)

### Spec Version Compatibility

OCI specs use semantic versioning with compatibility guarantees:

$$\text{compatible}(v_1, v_2) \iff \text{major}(v_1) = \text{major}(v_2)$$

### Extension Points

The specifications define extension points through:
- Custom annotations: $\text{key} \to \text{value}$ with reverse-domain naming
- Custom media types: for non-standard artifact types
- Platform extensions: OS-specific config fields

Annotation namespace cardinality:

$$|\text{annotations}| \leq |\text{valid keys}| \times |\text{valid values}|$$

Since keys are reverse-domain strings and values are arbitrary strings, the space is effectively unbounded.

---

## Prerequisites

automata-theory, set-theory, graph-theory, order-theory, protocol-theory

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| Container create | $O(n)$ — n = namespaces | $O(1)$ per namespace |
| File lookup (overlay) | $O(L)$ — L = layers (uncached) | $O(1)$ |
| File lookup (cached) | $O(1)$ — dentry cache | $O(n)$ — cache entries |
| Manifest verification | $O(L)$ — L = layer count | $O(1)$ |
| Image pull (parallel) | $O(S_{\max}/B)$ — largest layer | $O(S_{\text{total}})$ |
| Content digest | $O(S)$ — S = content size | $O(1)$ |
| Layer dedup check | $O(1)$ — digest comparison | $O(1)$ |
