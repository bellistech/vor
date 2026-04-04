# The Mathematics of CMake -- Dependency Graph Resolution

> *A build system is a compiler of compilers: it transforms a declarative dependency graph into a minimal, correct execution schedule.*

---

## 1. Dependency Graphs as DAGs (Graph Theory)

### The Problem

CMake constructs a directed acyclic graph (DAG) where nodes are targets and edges are `target_link_libraries` relationships. The build system must determine a valid execution order that respects all dependencies and maximizes parallelism.

### The Formula

A topological sort produces a linear ordering of vertices such that for every directed edge $(u, v)$, vertex $u$ appears before $v$. The number of valid topological orderings of a DAG is bounded by:

$$\frac{n!}{\prod_{v \in V} |D(v)|!}$$

where $D(v)$ is the set of descendants of vertex $v$ and $n = |V|$.

The critical path length determines the minimum possible build time under unlimited parallelism:

$$T_{\min} = \max_{p \in \text{paths}(G)} \sum_{v \in p} w(v)$$

where $w(v)$ is the compilation time of target $v$.

### Worked Examples

Consider a project with targets: `app -> libcore -> libutil`, `app -> libnet -> libutil`.

```
app
 |--- libcore --- libutil
 |--- libnet  --- libutil
```

The topological orderings are:
1. libutil, libcore, libnet, app
2. libutil, libnet, libcore, app

Critical path: libutil (2s) + libcore (5s) + app (3s) = 10s. With 2 cores, libcore and libnet compile in parallel after libutil, so actual time is 2 + 5 + 3 = 10s (libnet at 4s finishes during libcore's 5s).

---

## 2. Visibility Propagation (Lattice Theory)

### The Problem

CMake's `PRIVATE`, `INTERFACE`, and `PUBLIC` keywords control how compile definitions, include paths, and link dependencies propagate through the target graph. This forms a lattice of visibility levels.

### The Formula

Define the visibility lattice $\mathcal{L} = \{\bot, P, I, U\}$ where $\bot$ (none) $\leq P$ (PRIVATE) $\leq U$ (PUBLIC) and $\bot \leq I$ (INTERFACE) $\leq U$. The propagation rule for a chain $A \xrightarrow{v_1} B \xrightarrow{v_2} C$ is:

$$v_{\text{effective}}(A \to C) = v_1 \sqcap v_2$$

where the meet operation is defined as:

$$\text{PRIVATE} \sqcap x = \bot \quad \forall x$$
$$\text{INTERFACE} \sqcap \text{PUBLIC} = \text{INTERFACE}$$
$$\text{PUBLIC} \sqcap \text{PUBLIC} = \text{PUBLIC}$$

### Worked Examples

Given: `app --PUBLIC--> libcore --PRIVATE--> libutil`:

$$v_{\text{effective}}(\text{app} \to \text{libutil}) = \text{PUBLIC} \sqcap \text{PRIVATE} = \bot$$

App does NOT see libutil's headers. Now with `app --PUBLIC--> libcore --PUBLIC--> libutil`:

$$v_{\text{effective}}(\text{app} \to \text{libutil}) = \text{PUBLIC} \sqcap \text{PUBLIC} = \text{PUBLIC}$$

App sees libutil's headers and links against it.

---

## 3. Incremental Build Correctness (Hash-Based Change Detection)

### The Problem

CMake (via the underlying build tool) must detect which targets need rebuilding when source files change. False negatives cause incorrect builds; false positives waste time.

### The Formula

A target $t$ needs rebuilding if and only if:

$$\exists f \in \text{deps}(t) : h(f) \neq h_{\text{cached}}(f)$$

where $h$ is a hash function (typically modification timestamp or content hash). The probability of a false negative with a $k$-bit hash is:

$$P(\text{collision}) = 1 - e^{-n^2 / 2^{k+1}}$$

For $n = 10{,}000$ files and SHA-256 ($k = 256$):

$$P \approx \frac{10^8}{2^{257}} \approx 10^{-69}$$

### Worked Examples

Timestamp-based detection (Make): file `util.cpp` modified at $t_1 = 1712000000$, object `util.o` built at $t_0 = 1711999000$. Since $t_1 > t_0$, rebuild is triggered. Clock skew of $\delta$ seconds creates a danger zone where $|t_1 - t_0| < \delta$ may produce incorrect results.

Content-hash detection (Ninja): Even if the timestamp changes, if $\text{SHA256}(\text{util.cpp}_{\text{new}}) = \text{SHA256}(\text{util.cpp}_{\text{old}})$, no rebuild is needed. This avoids unnecessary recompilation after `git checkout` that restores identical content with newer timestamps.

---

## 4. Generator Expression Evaluation (Conditional Compilation Algebra)

### The Problem

Generator expressions are evaluated at generate time, not configure time. They form a Boolean algebra over build configuration, platform, and compiler properties.

### The Formula

A generator expression $G$ maps a build context $\mathcal{C} = (\text{config}, \text{platform}, \text{compiler}, \ldots)$ to a string:

$$G : \mathcal{C} \to \Sigma^*$$

The conditional expression $\langle \text{IF}:\phi, s \rangle$ evaluates as:

$$\langle \text{IF}:\phi, s \rangle(\mathcal{C}) = \begin{cases} s & \text{if } \phi(\mathcal{C}) = \text{true} \\ \epsilon & \text{otherwise} \end{cases}$$

Compound expressions compose: $\text{AND}(\phi_1, \phi_2) = \phi_1 \wedge \phi_2$, enabling:

$$\langle \text{IF}:\text{AND}(\text{CONFIG}=\text{Debug}, \text{PLATFORM}=\text{Linux}), \text{-fsanitize=address} \rangle$$

### Worked Examples

Expression: `$<$<AND:$<CONFIG:Debug>,$<CXX_COMPILER_ID:GNU>>:-fno-omit-frame-pointer>`

Context $\mathcal{C}_1 = (\text{Debug}, \text{Linux}, \text{GNU})$: evaluates to `-fno-omit-frame-pointer`.
Context $\mathcal{C}_2 = (\text{Release}, \text{Linux}, \text{GNU})$: evaluates to empty string $\epsilon$.
Context $\mathcal{C}_3 = (\text{Debug}, \text{Windows}, \text{MSVC})$: evaluates to $\epsilon$.

---

## 5. Build Parallelism and Amdahl's Law (Performance Theory)

### The Problem

Given a dependency graph, what is the maximum speedup achievable by adding more CPU cores to the build?

### The Formula

Let $s$ be the fraction of the build that is inherently sequential (the critical path ratio). Amdahl's Law gives the maximum speedup with $p$ processors:

$$S(p) = \frac{1}{s + \frac{1-s}{p}}$$

The critical path ratio for a build DAG $G$ is:

$$s = \frac{T_{\text{critical}}}{T_{\text{total}}} = \frac{\max_{p \in \text{paths}} \sum_{v \in p} w(v)}{\sum_{v \in V} w(v)}$$

### Worked Examples

A project has total compilation work $T_{\text{total}} = 120\text{s}$ and critical path $T_{\text{critical}} = 30\text{s}$, so $s = 0.25$.

With 4 cores: $S(4) = \frac{1}{0.25 + 0.75/4} = \frac{1}{0.4375} = 2.29\times$

With 8 cores: $S(8) = \frac{1}{0.25 + 0.75/8} = \frac{1}{0.34375} = 2.91\times$

With $\infty$ cores: $S(\infty) = \frac{1}{0.25} = 4.0\times$

The build can never be faster than 30s regardless of parallelism. This is why breaking large translation units into smaller ones (reducing $s$) matters more than adding cores beyond a certain point.

---

## 6. Configuration Space Explosion (Combinatorics)

### The Problem

CMake presets, generator expressions, and platform variables create a combinatorial explosion of possible build configurations. Testing all combinations is infeasible; understanding the growth rate helps prioritize CI matrices.

### The Formula

Given $d$ configuration dimensions with cardinalities $c_1, c_2, \ldots, c_d$, the total configuration space is:

$$|C| = \prod_{i=1}^{d} c_i$$

A pairwise coverage strategy (testing all 2-combinations of values across dimensions) requires:

$$N_{\text{pairwise}} \geq \max_{i,j} (c_i \cdot c_j)$$

test configurations, typically $O(\max(c_i)^2 \cdot \log d)$ by covering array theory.

### Worked Examples

Dimensions: build type (3: Debug, Release, RelWithDebInfo), compiler (3: GCC, Clang, MSVC), platform (3: Linux, macOS, Windows), C++ standard (3: C++17, C++20, C++23), shared/static (2).

Full space: $3 \times 3 \times 3 \times 3 \times 2 = 162$ configurations.

Pairwise coverage: a covering array $\text{CA}(N; 2, 5, \{3,3,3,3,2\})$ requires approximately $N \approx 9\text{-}12$ configurations (known bounds from NIST covering array tables). This tests every pair of dimension values with 93% fewer builds than exhaustive testing.

---

## Prerequisites

- Directed acyclic graphs (DAGs) and topological sorting
- Lattice theory (partial orders, meet/join operations)
- Hash functions and collision probability
- Boolean algebra and conditional evaluation
- Amdahl's Law and parallel speedup analysis
- Basic graph theory (paths, critical path method)
- Combinatorics (Cartesian products, covering arrays)
