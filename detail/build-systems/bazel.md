# The Mathematics of Bazel -- Hermeticity and Reproducibility

> *A build is reproducible if and only if it is a pure function: given identical inputs, it must always produce bit-identical outputs.*

---

## 1. Content-Addressable Storage (Cryptographic Hashing)

### The Problem

Bazel's remote cache and execution model relies on uniquely identifying build artifacts by their content rather than by file path or timestamp. Two developers (or CI machines) must agree that a cached artifact is valid without communicating about filesystem state.

### The Formula

An action $a$ with inputs $\{i_1, \ldots, i_n\}$, command $c$, and environment $e$ produces a cache key:

$$k(a) = H(H(i_1) \| H(i_2) \| \cdots \| H(i_n) \| H(c) \| H(e))$$

where $H$ is SHA-256 and $\|$ denotes concatenation. The probability of a false cache hit (collision) is:

$$P(\text{collision}) \leq \frac{N^2}{2^{257}}$$

where $N$ is the total number of distinct actions ever executed.

### Worked Examples

A repository with 50,000 build actions executed 100 times per day for 10 years produces $N = 50{,}000 \times 100 \times 3{,}650 \approx 1.8 \times 10^{10}$ actions.

$$P(\text{collision}) \leq \frac{(1.8 \times 10^{10})^2}{2^{257}} = \frac{3.24 \times 10^{20}}{2^{257}} \approx 1.4 \times 10^{-57}}$$

This probability is negligibly small -- far less likely than hardware errors.

---

## 2. Build Graph as a Merkle DAG (Graph Theory)

### The Problem

Bazel represents the build as a Merkle DAG where each node's identity depends on its content and all its descendants. A change to any leaf propagates upward, invalidating exactly the affected subtree.

### The Formula

For a target $t$ with direct dependencies $\{d_1, \ldots, d_k\}$, the Merkle hash is:

$$M(t) = H(\text{rule}(t) \| \text{srcs}(t) \| M(d_1) \| \cdots \| M(d_k))$$

The invalidation set when a file $f$ changes is:

$$I(f) = \{t \in V : f \in \text{transitive\_srcs}(t)\}$$

The fraction of the build graph invalidated by a single-file change is:

$$\rho = \frac{|I(f)|}{|V|}$$

### Worked Examples

In a monorepo with $|V| = 20{,}000$ targets, a utility library `//lib:strings` is used by 3,000 targets (transitively). Changing a source file in `//lib:strings`:

$$\rho = \frac{3{,}000}{20{,}000} = 0.15$$

So 15% of the build graph is invalidated. If instead the change is in a leaf binary's `main.cc` with no reverse dependencies, $|I(f)| = 1$ and $\rho = 0.00005$.

This illustrates why fine-grained targets reduce rebuild cost: splitting `//lib:strings` into `//lib:strings_format` and `//lib:strings_parse` might reduce $|I(f)|$ from 3,000 to 800 for a change in format logic.

---

## 3. Diamond Dependency Resolution (Lattice Theory)

### The Problem

In large monorepos, diamond dependencies are inevitable: $A \to B \to D$ and $A \to C \to D$. Different versions of $D$ requested by $B$ and $C$ create conflicts. Bzlmod must resolve these to a single version.

### The Formula

Given a dependency graph $G$ and a version set $\{v_1^d, v_2^d, \ldots, v_m^d\}$ requested for module $d$, bzlmod applies Minimum Version Selection (MVS):

$$v_{\text{selected}}(d) = \max(v_1^d, v_2^d, \ldots, v_m^d)$$

under semantic versioning ordering. The resolved graph $G'$ satisfies:

$$\forall (u, d) \in E(G) : v_{\text{requested}}(u, d) \leq v_{\text{selected}}(d)$$

### Worked Examples

Module `app` depends on `protobuf@3.21` (via `grpc@1.60`) and `protobuf@3.23` (via `logging@2.0`).

$$v_{\text{selected}}(\text{protobuf}) = \max(3.21, 3.23) = 3.23$$

This is safe under semantic versioning since 3.23 is backwards-compatible with 3.21. If `grpc@1.60` requires `protobuf@4.0` (a major version bump), MVS cannot resolve the conflict and reports an error, forcing the developer to upgrade or fork.

---

## 4. Sandboxing and Hermeticity (Information Theory)

### The Problem

A hermetic build must be isolated from the host system. Any information leaking from the environment into the build output breaks reproducibility. Bazel's sandbox enforces this by restricting filesystem and network access.

### The Formula

Define the mutual information between the host environment $E$ and build output $O$:

$$I(E; O) = H(O) - H(O | E)$$

A perfectly hermetic build has $I(E; O) = 0$, meaning the output is conditionally independent of the environment given the declared inputs:

$$P(O | \text{inputs}, E) = P(O | \text{inputs})$$

Sources of hermeticity violation include:

$$I_{\text{leak}} = I_{\text{timestamp}} + I_{\text{hostname}} + I_{\text{PATH}} + I_{\text{undeclared\_inputs}}$$

### Worked Examples

A C++ compilation embeds `__DATE__` and `__TIME__` macros. Each invocation on a different day produces different output, contributing $I_{\text{timestamp}} \approx \log_2(86400 \times 365) \approx 25$ bits of environment information per year of possible dates.

Bazel addresses this by: (1) sandboxing the filesystem so only declared inputs are visible, (2) setting deterministic timestamps in the sandbox, (3) forbidding network access during actions unless explicitly tagged `requires-network`.

---

## 5. Remote Execution Scheduling (Distributed Computing)

### The Problem

Remote execution distributes build actions across a cluster. The scheduler must assign actions to workers while respecting platform constraints and minimizing total build latency.

### The Formula

Given $n$ actions with execution times $\{w_1, \ldots, w_n\}$ and $p$ identical workers, the optimal makespan (total build time) is bounded by:

$$T_{\text{opt}} \geq \max\left(T_{\text{critical}}, \frac{\sum_{i=1}^n w_i}{p}\right)$$

The list scheduling algorithm (greedy assignment to the least-loaded worker) achieves:

$$T_{\text{list}} \leq \left(1 + \frac{1}{p}\right) T_{\text{opt}}$$

With communication overhead $\delta$ per action (uploading inputs, downloading outputs):

$$T_{\text{remote}} = T_{\text{compute}} + n \cdot \delta$$

Remote execution is beneficial when:

$$T_{\text{local}} > T_{\text{remote}} \implies \frac{\sum w_i}{p_{\text{local}}} > \frac{\sum w_i}{p_{\text{remote}}} + n \cdot \delta$$

### Worked Examples

A build with 1,000 actions totaling 500s of compute, $\delta = 0.1$s per action:

Local (8 cores): $T_{\text{local}} = 500/8 = 62.5$s

Remote (100 workers): $T_{\text{remote}} = 500/100 + 1000 \times 0.1 = 5 + 100 = 105$s

Here remote execution is actually slower due to overhead. But with $\delta = 0.01$s (fast network):

$T_{\text{remote}} = 5 + 10 = 15$s -- now $4.2\times$ faster than local.

The crossover point is $\delta_{\text{max}} = \frac{500/8 - 500/100}{1000} = \frac{57.5}{1000} = 0.0575$s.

---

## 6. Cache Hit Rate Modeling (Probability Theory)

### The Problem

Remote caching effectiveness depends on how frequently developers build the same targets with the same inputs. Understanding cache hit rates guides infrastructure investment decisions.

### The Formula

For a target $t$ rebuilt by $m$ developers at rate $\lambda$ builds/day, with cache TTL $\tau$, the probability that a build hits the cache is:

$$P(\text{hit}) = 1 - e^{-(m-1)\lambda\tau / n_t}$$

where $n_t$ is the number of distinct action keys for target $t$ (changes per day). For a repository with $T$ targets, the overall cache hit rate is:

$$\bar{P}(\text{hit}) = \frac{\sum_{t=1}^{T} w_t \cdot P_t(\text{hit})}{\sum_{t=1}^{T} w_t}$$

where $w_t$ is the build cost (time) of target $t$.

The cost savings from caching:

$$\text{savings} = \bar{P}(\text{hit}) \cdot T_{\text{total}} \cdot C_{\text{compute}} - C_{\text{cache\_storage}}$$

### Worked Examples

Team of $m = 20$ developers, target changes once per day ($n_t = 1$), each developer builds 4 times/day ($\lambda = 4$), cache TTL $\tau = 7$ days.

$$P(\text{hit}) = 1 - e^{-19 \times 4 \times 7 / 1} = 1 - e^{-532} \approx 1.0$$

Nearly 100% hit rate for stable targets. For a rapidly changing target with $n_t = 20$ changes/day:

$$P(\text{hit}) = 1 - e^{-19 \times 4 \times 7 / 20} = 1 - e^{-26.6} \approx 1.0$$

Still very high. Cache miss rates only become significant when $n_t \gg m\lambda\tau$, which means the target changes faster than people build it -- rare in practice.

---

## Prerequisites

- Cryptographic hash functions (SHA-256, collision resistance)
- Merkle trees and content-addressable storage
- Directed acyclic graphs (DAGs) and topological sorting
- Semantic versioning and version resolution algorithms
- Information theory (mutual information, conditional independence)
- Scheduling theory (makespan, list scheduling, critical path)
- Lattice theory (partial orders, maximum/minimum operations)
- Probability theory (exponential distribution, cache models)
