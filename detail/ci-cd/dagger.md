# The Mathematics of Dagger — Container DAGs, Caching, and Build Optimization

> *Dagger models CI/CD as a directed acyclic graph of container operations where each node is an immutable filesystem layer. BuildKit's content-addressable caching transforms repeated builds into graph lookups, and the function-as-container abstraction enables mathematical reasoning about build reproducibility and parallelism.*

---

## 1. Content-Addressable Caching (Hash Functions)

### The Cache Model

Every Dagger operation produces a layer with a content hash:

$$h(L) = \text{SHA256}(\text{parent\_hash} \| \text{operation} \| \text{inputs})$$

### Cache Hit Decision

$$\text{CacheHit}(op) = \exists L \in \text{Cache}: h(L) = h(op)$$

### Cache Hit Rate

For a pipeline with $n$ operations and $k$ changed inputs:

$$P(\text{hit}) = \frac{n - k}{n}$$

### Worked Example: Go Build Pipeline

| Operation | Inputs Change? | Cache Hit? | Time (cold) | Time (warm) |
|:---|:---:|:---:|:---:|:---:|
| From("golang:1.24") | No | Yes | 2s | 0s |
| WithDirectory (go.mod) | No | Yes | 1s | 0s |
| go mod download | No | Yes | 30s | 0s |
| WithDirectory (source) | Yes | No | 1s | 1s |
| go build | Yes | No | 45s | 45s |
| **Total** | | | **79s** | **46s** |

$$\text{Speedup} = \frac{79}{46} = 1.72\times$$

### Optimal Layer Ordering

Place frequently-changing operations last to maximize cache hits:

$$\text{Optimal}: \text{base} \to \text{deps} \to \text{source} \to \text{build}$$
$$\text{Suboptimal}: \text{base} \to \text{source} \to \text{deps} \to \text{build}$$

Cache invalidation cascades: changing layer $i$ invalidates layers $i+1, \ldots, n$.

$$\text{Invalidated layers} = n - i$$

---

## 2. Container Operation DAG (Graph Theory)

### Function Composition as DAG

Each Dagger function call creates a chain of operations:

$$F = op_1 \circ op_2 \circ \cdots \circ op_n$$

The full pipeline forms a DAG:

$$G = (V, E) \text{ where } V = \text{operations}, E = \text{data dependencies}$$

### Parallelism Detection

Two operations are parallelizable iff they have no data dependency:

$$\text{Parallel}(op_a, op_b) \iff \nexists \text{path}(op_a, op_b) \wedge \nexists \text{path}(op_b, op_a)$$

### DAG Width and Depth

$$\text{Depth}(G) = \text{longest path} = T_{sequential}$$
$$\text{Width}(G) = \text{max antichain} = P_{max\_parallel}$$

### Amdahl's Law for Dagger Pipelines

If fraction $f$ of work is parallelizable with $P$ workers:

$$T(P) = T_{serial} + \frac{T_{parallel}}{P}$$

$$\text{Speedup} = \frac{1}{(1-f) + f/P}$$

| Serial Fraction | 2 Workers | 4 Workers | 8 Workers | Limit |
|:---:|:---:|:---:|:---:|:---:|
| 10% | 1.82x | 3.08x | 4.71x | 10x |
| 25% | 1.60x | 2.29x | 2.91x | 4x |
| 50% | 1.33x | 1.60x | 1.78x | 2x |

---

## 3. BuildKit Layer Mathematics (Filesystem Theory)

### Layer Composition

Each container operation creates an overlay filesystem layer:

$$FS_n = FS_0 \cup \Delta_1 \cup \Delta_2 \cup \cdots \cup \Delta_n$$

Where $\Delta_i$ is the diff (added/modified/deleted files) at step $i$.

### Layer Size

$$S_{layer}(\Delta_i) = \sum_{f \in \text{added}} |f| + \sum_{f \in \text{modified}} |f|$$

Deletions are represented as whiteout files (near-zero size).

### Total Image Size

$$S_{image} = S_{base} + \sum_{i=1}^{n} S_{layer}(\Delta_i)$$

### Deduplication

Content-addressable storage deduplicates identical layers:

$$S_{stored} = |\text{unique layers}| \times \bar{S}_{layer}$$

For $m$ builds sharing the same base image with $b$ common layers:

$$S_{deduplicated} = S_{base} + b \times \bar{S}_{common} + m \times \sum S_{unique}$$

$$\text{Savings} = (m-1) \times (S_{base} + b \times \bar{S}_{common})$$

---

## 4. Caching Strategy Optimization (Dynamic Programming)

### Cache Eviction

BuildKit uses LRU eviction with a size budget:

$$\text{Evict}(L) = \arg\min_{L \in \text{Cache}} \text{last\_used}(L) \quad \text{when } S_{cache} > S_{max}$$

### Cache Value Function

The value of caching layer $L$:

$$V(L) = T_{reproduce}(L) \times P(\text{future hit})(L)$$

### Optimal Cache Budget

Minimize total build time across $B$ builds:

$$T_{total}(S_{cache}) = \sum_{b=1}^{B} \sum_{op \in b} \begin{cases}
0 & \text{if cache hit} \\
T_{op} & \text{if cache miss}
\end{cases}$$

### Cache Volume Effectiveness

For Go module cache:

$$T_{without\_cache} = T_{download} + T_{compile\_deps}$$
$$T_{with\_cache} = T_{check\_cached} + T_{compile\_changed}$$

| Dependency Count | Without Cache | With Cache | Speedup |
|:---:|:---:|:---:|:---:|
| 50 | 30s | 2s | 15x |
| 200 | 120s | 5s | 24x |
| 500 | 300s | 10s | 30x |

---

## 5. Secret Security Model (Information Flow)

### Secret Isolation

Secrets in Dagger are never stored in layer caches:

$$\text{Cache key}(op) = h(\text{parent} \| \text{command}) \quad \text{(secrets excluded)}$$

$$\forall L \in \text{Cache}: \text{secret} \notin L.\text{content}$$

### Information Flow Control

Dagger enforces that secrets flow only to operations that declare them:

$$\text{Access}(secret, op) \iff secret \in op.\text{declared\_secrets}$$

### Secret Injection Model

$$\text{Container state} = \begin{cases}
\text{with secret mounted} & \text{during WithExec} \\
\text{secret removed} & \text{after WithExec (not in layer)}
\end{cases}$$

This is enforced at the BuildKit layer — secrets are tmpfs-mounted, never committed to layers.

---

## 6. Multi-Platform Build (Combinatorial Expansion)

### Build Matrix

For $P$ platforms and $V$ variants:

$$\text{Total builds} = |P| \times |V|$$

### Parallel Execution

All platform builds are independent:

$$T_{multi} = \max_{p \in P} T_{build}(p)$$

### Cross-Compilation Cost

Native vs cross-compilation:

$$\frac{T_{cross}}{T_{native}} \approx 1.0 \text{ (for Go with CGO\_ENABLED=0)}$$
$$\frac{T_{cross}}{T_{native}} \approx 2-5\times \text{ (for C/C++ with QEMU emulation)}$$

### Manifest List Size

$$S_{manifest} = |P| \times S_{per\_platform\_image} + S_{manifest\_list}$$

For amd64 + arm64 Go binary:

$$S_{total} \approx 2 \times S_{alpine+binary} + 1\text{KB}$$

---

## 7. Pipeline Reproducibility (Determinism)

### Reproducibility Conditions

A build is reproducible iff:

$$\forall t_1, t_2: \text{Build}(\text{inputs}, t_1) = \text{Build}(\text{inputs}, t_2)$$

### Sources of Non-Determinism

| Source | Mitigation | Dagger Default |
|:---|:---|:---|
| Base image tag (`:latest`) | Pin digest | Manual |
| Package versions | Lock file | Manual |
| Timestamps in binaries | `-trimpath` | Manual |
| Network fetches | Cache volumes | Automatic |
| Build time env vars | Exclude from cache | Automatic |

### Reproducibility Score

$$R = \frac{|\text{deterministic operations}|}{|\text{total operations}|}$$

Ideal: $R = 1.0$ (fully reproducible).

### Cache Correctness

If builds are reproducible:

$$h(\text{Build}(inputs)) = h(\text{Cache}(inputs)) \implies \text{Cache is correct}$$

Non-reproducible operations invalidate this guarantee:

$$\text{Stale cache risk} = 1 - R$$

---

*Dagger transforms CI/CD from imperative scripting into functional container composition. Every function call extends an immutable DAG, every cache hit is a hash table lookup, and every secret is information-flow controlled. The mathematics of content addressing, graph scheduling, and layer deduplication make builds both fast and correct.*

## Prerequisites

- Hash functions and content addressing
- Directed acyclic graphs (DAGs)
- Overlay filesystem concepts
- Amdahl's Law for parallelism

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| Cache lookup (hash) | $O(1)$ | $O(|cache|)$ |
| Layer composition | $O(|\Delta|)$ files | $O(\sum |\Delta_i|)$ |
| DAG scheduling | $O(|V| + |E|)$ | $O(|V|)$ |
| Secret mount/unmount | $O(1)$ | $O(|secret|)$ tmpfs |
| Multi-platform build | $O(\max T_p)$ | $O(|P| \times S_{image})$ |
| Cache eviction (LRU) | $O(1)$ amortized | $O(|cache|)$ |
