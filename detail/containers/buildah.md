# The Mathematics of Buildah — Layer Algebra and Daemonless Build Optimization

> *Container image construction is a sequence of filesystem delta operations forming a chain of content-addressed layers, where build optimization reduces to minimizing total image size and layer count subject to cache reuse constraints in a directed acyclic graph.*

---

## 1. Layer Model as Chain of Diffs (Algebra)

### Filesystem State Transitions

An image is a sequence of layer transformations applied to an empty filesystem:

$$I = l_n \circ l_{n-1} \circ \cdots \circ l_2 \circ l_1$$

Each layer $l_i$ is a function:

$$l_i: \text{FS} \rightarrow \text{FS}$$

Where $\text{FS}$ is the set of all possible filesystem states.

### Layer as Delta Set

Each layer is a set of filesystem operations:

$$l_i = \{(\text{op}, \text{path}, \text{content})\}$$

Where $\text{op} \in \{\text{ADD}, \text{MODIFY}, \text{DELETE}\}$.

The final filesystem state:

$$\text{FS}_n = \bigoplus_{i=1}^{n} l_i = l_1 \oplus l_2 \oplus \cdots \oplus l_n$$

Where $\oplus$ is the overlay union operation with last-writer-wins semantics:

$$(\text{FS}_a \oplus \text{FS}_b)(p) = \begin{cases} \text{FS}_b(p) & \text{if } p \in \text{dom}(\text{FS}_b) \\ \text{FS}_a(p) & \text{if } p \in \text{dom}(\text{FS}_a) \setminus \text{dom}(\text{FS}_b) \\ \text{deleted} & \text{if } p \in \text{whiteout}(\text{FS}_b) \end{cases}$$

### Whiteout Files

Deletions are represented as whiteout files:

$$\text{DELETE}(\text{/usr/bin/old}) \mapsto \text{ADD}(\text{/usr/bin/.wh.old}, \emptyset)$$

Directory deletion (opaque whiteout):

$$\text{DELETE}(\text{/var/cache/*}) \mapsto \text{ADD}(\text{/var/cache/.wh..wh..opq}, \emptyset)$$

---

## 2. Content Addressing (Hash Theory)

### Blob Identification

Every layer, config, and manifest is identified by its cryptographic digest:

$$\text{id}(b) = \text{SHA-256}(b)$$

This provides:

**Integrity:** $P(\text{corruption undetected}) \leq 2^{-256}$

**Deduplication:** identical content shares storage:

$$\text{id}(b_1) = \text{id}(b_2) \implies b_1 = b_2 \quad \text{(with overwhelming probability)}$$

### Layer Chain Digest

The image configuration includes diff IDs (uncompressed layer digests):

$$\text{diff\_id}(l_i) = \text{SHA-256}(\text{uncompressed}(l_i))$$

The chain ID for layer $i$ enables cache lookup:

$$\text{chain\_id}(l_1) = \text{diff\_id}(l_1)$$

$$\text{chain\_id}(l_1, \ldots, l_i) = \text{SHA-256}(\text{chain\_id}(l_1, \ldots, l_{i-1}) + \text{ } + \text{diff\_id}(l_i))$$

This chain structure means a layer's identity depends on all preceding layers.

---

## 3. Build Cache as DAG (Graph Theory)

### Cache Hit Semantics

The build cache is a directed acyclic graph where nodes are layers and edges are "built upon" relationships:

$$G_{\text{cache}} = (L, E)$$

$$\text{cache\_hit}(l_i, \text{context}_i) \iff \exists l' \in L : \text{chain\_id}(l_1, \ldots, l_{i-1}, l') = \text{chain\_id}(l_1, \ldots, l_{i-1}, l_i)$$

Cache invalidation cascades: if layer $l_k$ changes, all subsequent layers $l_{k+1}, \ldots, l_n$ are invalidated:

$$\text{invalidated}(k) = \{l_i : i \geq k\}$$

$$|\text{invalidated}(k)| = n - k + 1$$

### Optimal Instruction Ordering

To maximize cache hits, order Dockerfile instructions by change frequency:

$$\text{order}: \text{rarely changed} \rightarrow \text{frequently changed}$$

Expected cache rebuild cost:

$$E[\text{cost}] = \sum_{i=1}^{n} P(\text{change at } l_i) \times \sum_{j=i}^{n} \text{time}(l_j)$$

Minimized when $P(\text{change at } l_i)$ is non-decreasing in $i$.

---

## 4. Multi-Stage Build Optimization (Optimization Theory)

### Stage Dependency Graph

A multi-stage build creates a DAG of build stages:

$$G_{\text{stages}} = (S, D)$$

Where $S = \{s_1, s_2, \ldots, s_m\}$ are stages and $D$ are `COPY --from` dependencies.

### Size Reduction

Final image size with multi-stage:

$$\text{size}_{\text{final}} = \text{size}(\text{base}_{\text{runtime}}) + \sum_{f \in F} \text{size}(f)$$

Where $F$ is the set of files copied from builder stages.

Single-stage equivalent:

$$\text{size}_{\text{single}} = \text{size}(\text{base}_{\text{build}}) + \sum_{l \in L} \text{size}(l)$$

Reduction ratio:

$$r = 1 - \frac{\text{size}_{\text{final}}}{\text{size}_{\text{single}}}$$

| Language | Build Image | Runtime Image | Reduction |
|:---|:---:|:---:|:---:|
| Go (static) | ~800MB | ~2MB (scratch) | 99.7% |
| Go (CGo) | ~800MB | ~25MB (distroless) | 96.9% |
| Java | ~500MB | ~200MB (JRE) | 60.0% |
| Node.js | ~1000MB | ~150MB (slim) | 85.0% |
| Rust | ~1200MB | ~5MB (musl static) | 99.6% |

---

## 5. Rootless Build Security Model (Security Theory)

### User Namespace Mapping

Rootless builds map container UIDs to unprivileged host UIDs:

$$f: \text{UID}_{\text{container}} \rightarrow \text{UID}_{\text{host}}$$

$$f(\text{uid}_c) = \text{uid}_{\text{host\_base}} + \text{uid}_c$$

For mapping $(0, 100000, 65536)$:

$$f(0) = 100000, \quad f(1) = 100001, \quad \ldots, \quad f(65535) = 165535$$

### Attack Surface Comparison

| Attack Vector | Rootful Build | Rootless Build |
|:---|:---:|:---:|
| Host UID 0 access | Yes | No |
| Kernel exploit surface | Full | Reduced (user NS) |
| File ownership attacks | Root files on host | Mapped UIDs only |
| Network namespace | Host network | Isolated |
| Device access | All | None |

Privilege level:

$$P_{\text{rootful}} = \text{CAP\_ALL} = 2^{41} - 1$$
$$P_{\text{rootless}} = 0 \text{ (on host)}$$

---

## 6. Squash Operation (Set Theory)

### Layer Merge

Squashing collapses $n$ layers into 1:

$$L_{\text{squashed}} = \text{flatten}(l_1, l_2, \ldots, l_n)$$

The flattened layer contains only the final state:

$$L_{\text{squashed}} = \{(p, c) : p \in \text{FS}_n \wedge c = \text{FS}_n(p)\}$$

### Size Analysis

Individual layers may contain redundant data:

$$\text{size}(l_1 + l_2 + \cdots + l_n) \geq \text{size}(\text{flatten}(l_1, \ldots, l_n))$$

The difference is waste from overwritten/deleted files:

$$\text{waste} = \sum_{i=1}^{n} \text{size}(l_i) - \text{size}(L_{\text{squashed}})$$

$$\text{waste} = \sum_{i=1}^{n-1} \sum_{p \in \text{overwritten}(l_i, l_{i+1}, \ldots, l_n)} \text{size}(l_i(p))$$

### Trade-off

Squashing eliminates waste but destroys cache reusability:

$$\text{cache\_reuse}(L_{\text{squashed}}) = 0 \quad \text{(unique digest)}$$

Decision criterion:

$$\text{squash if } \frac{\text{waste}}{\text{total size}} > \text{threshold} \wedge \text{cache importance is low}$$

---

## 7. Parallel Build Execution (Concurrency Theory)

### Build DAG Parallelism

For a multi-stage Dockerfile, the maximum parallelism is determined by the DAG width:

$$\text{width}(G) = \max_{t} |\{s \in S : s \text{ is executable at time } t\}|$$

The critical path determines minimum build time:

$$T_{\text{min}} = \max_{\text{path } p \in G} \sum_{s \in p} T(s)$$

Speedup from parallelism:

$$\text{speedup} = \frac{T_{\text{sequential}}}{T_{\text{parallel}}} = \frac{\sum_{s \in S} T(s)}{T_{\text{critical path}}}$$

For independent stages $A$ and $B$ feeding into $C$:

$$T_{\text{parallel}} = \max(T_A, T_B) + T_C$$
$$T_{\text{sequential}} = T_A + T_B + T_C$$
$$\text{speedup} = \frac{T_A + T_B + T_C}{\max(T_A, T_B) + T_C}$$

### Buildah Native Advantage

Buildah's scriptable interface enables custom parallelism:

```bash
# Parallel independent stages
stage_a() { buildah from golang:1.24; ...; } &
stage_b() { buildah from node:20; ...; } &
wait
# Combine results
```

This is not possible with standard Dockerfile syntax (sequential by default).

---

## Prerequisites

set-theory, hash-functions, graph-theory, optimization, concurrency

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| Layer creation (commit) | $O(n)$ — n = changed files | $O(n)$ |
| Content digest (SHA-256) | $O(s)$ — s = blob size | $O(1)$ |
| Cache lookup (chain ID) | $O(1)$ — hash map | $O(L)$ — L = cached layers |
| Multi-stage copy | $O(f)$ — f = files copied | $O(f)$ |
| Squash (flatten) | $O(S)$ — S = total layer size | $O(F)$ — F = final FS size |
| Rootfs mount | $O(n)$ — n = layers (overlay) | $O(1)$ — union mount |
