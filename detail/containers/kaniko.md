# The Mathematics of kaniko — Layer Caching and Build Graph Optimization

> *kaniko builds container images by executing Dockerfile instructions as a sequence of filesystem snapshots. The efficiency of builds depends on cache hit rates modeled as probability distributions over instruction changes, snapshot diff algorithms as set operations, and multi-stage builds as DAGs with materialization choices.*

---

## 1. Layer Cache Hit Probability (Probability Theory)

### The Problem

kaniko caches each Dockerfile instruction as a layer. A cache hit requires the instruction text, base image digest, and all input files to be unchanged. The probability of a cache hit depends on how frequently each layer's inputs change.

### The Formula

For instruction $i$ with change probability $p_i$ per build:

$$P(\text{hit}_i) = 1 - p_i$$

Since layers are ordered and a miss invalidates all subsequent layers:

$$P(\text{hit}_i \mid \text{hit}_{i-1}) = \begin{cases} 1 - p_i & \text{if instruction } i-1 \text{ hit cache} \\ 0 & \text{otherwise} \end{cases}$$

Expected cached layers (from top of Dockerfile):

$$E[\text{cached}] = \sum_{i=1}^{n} \prod_{j=1}^{i} (1 - p_j)$$

Expected build time savings:

$$\Delta T = \sum_{i=1}^{n} T_i \cdot \prod_{j=1}^{i} (1 - p_j)$$

### Worked Examples

5-layer Dockerfile with change probabilities:

| Layer | Instruction | $p_i$ | $\prod(1-p_j)$ | Time $T_i$ |
|:---:|:---|:---:|:---:|:---:|
| 1 | FROM golang:1.22 | 0.01 | 0.990 | 0s (pull) |
| 2 | COPY go.mod go.sum | 0.10 | 0.891 | 5s |
| 3 | RUN go mod download | 0.10 | 0.802 | 30s |
| 4 | COPY . . | 0.95 | 0.040 | 2s |
| 5 | RUN go build | 0.95 | 0.002 | 60s |

$$E[\text{cached}] = 0.990 + 0.891 + 0.802 + 0.040 + 0.002 = 2.725 \text{ layers}$$

$$\Delta T = 0 + 0.891 \times 5 + 0.802 \times 30 + 0.040 \times 2 + 0.002 \times 60 = 28.6\text{s saved}$$

---

## 2. Filesystem Snapshot Diff (Set Theory)

### The Problem

After each Dockerfile instruction, kaniko snapshots the filesystem to produce a layer. The snapshot algorithm must compute the diff between the previous and current filesystem states efficiently.

### The Formula

For filesystem states $F_{i-1}$ and $F_i$:

Created files:

$$C_i = \{f \in F_i \mid f \notin F_{i-1}\}$$

Deleted files (whiteouts):

$$D_i = \{f \in F_{i-1} \mid f \notin F_i\}$$

Modified files:

$$M_i = \{f \in F_i \cap F_{i-1} \mid \text{hash}(f_i) \neq \text{hash}(f_{i-1})\}$$

Layer content:

$$L_i = C_i \cup M_i \cup \text{whiteout}(D_i)$$

Layer size:

$$S(L_i) = \sum_{f \in C_i \cup M_i} \text{size}(f) + |D_i| \cdot \epsilon$$

where $\epsilon$ is the whiteout marker size (~negligible).

### Worked Examples

`RUN apt-get install -y curl` changes:

| Category | Count | Size |
|:---|:---:|:---:|
| Created ($C$) | 142 files | 4.2 MB |
| Modified ($M$) | 8 files (dpkg database) | 0.3 MB |
| Deleted ($D$) | 3 files (temp) | 0 MB |

$$S(L) = 4.2 + 0.3 + 0 = 4.5 \text{ MB}$$

Snapshot scan time with `redo` mode (tracks only touched paths):

$$T_{\text{redo}} = O(|C| + |M| + |D|) = O(153)$$

vs. `full` mode (scans entire filesystem):

$$T_{\text{full}} = O(|F_i|) = O(15000) \text{ files}$$

---

## 3. Multi-Stage Build DAG (Graph Theory)

### The Problem

Multi-stage Dockerfiles create a DAG of build stages. kaniko must decide which stages to build and which can use cached results. The `--target` flag prunes unreachable stages.

### The Formula

Build DAG $G = (S, E)$ where $S$ = stages and $E$ = COPY --from dependencies.

For target stage $t$, the required stage set:

$$R(t) = \{t\} \cup \bigcup_{s \in \text{deps}(t)} R(s)$$

Stages to skip:

$$\text{skip} = S \setminus R(t)$$

Build time reduction:

$$\Delta T = \sum_{s \in \text{skip}} T_s$$

### Worked Examples

```
Stage A (builder): FROM golang:1.22
Stage B (test):    COPY --from=A /app /app
Stage C (assets):  FROM node:20 (independent)
Stage D (final):   COPY --from=A /app/server
                   COPY --from=C /dist
```

DAG: $A \to B$, $A \to D$, $C \to D$.

Target D: $R(D) = \{D, A, C\}$. Stage B is skipped.

Target B: $R(B) = \{B, A\}$. Stages C and D are skipped.

---

## 4. Cache Repository Storage (Information Theory)

### The Problem

The cache repository stores one image per cacheable layer. Total cache storage depends on the number of unique instruction+input combinations encountered over build history.

### The Formula

For $b$ builds over time, each with $n$ layers, and layer change probability $p_i$:

Expected unique cache entries for layer $i$ after $b$ builds:

$$U_i(b) = 1 + (b-1) \cdot p_i \cdot \prod_{j=1}^{i-1}(1 - p_j)$$

Total cache entries:

$$U_{\text{total}} = \sum_{i=1}^{n} U_i(b)$$

Cache storage:

$$S_{\text{cache}} = \sum_{i=1}^{n} U_i(b) \cdot \bar{S}_i$$

### Worked Examples

100 builds, 5 layers with $p = [0.01, 0.10, 0.10, 0.95, 0.95]$:

Layer 1: $U_1 = 1 + 99 \times 0.01 = 1.99 \approx 2$ unique entries
Layer 2: $U_2 = 1 + 99 \times 0.10 \times 0.99 = 10.8 \approx 11$
Layer 3: $U_3 = 1 + 99 \times 0.10 \times 0.891 = 9.82 \approx 10$
Layer 4: $U_4 = 1 + 99 \times 0.95 \times 0.802 = 76.4 \approx 76$
Layer 5: not independently cached (depends on L4)

$$U_{\text{total}} \approx 2 + 11 + 10 + 76 = 99 \text{ cache entries}$$

With average layer sizes [50MB, 5MB, 30MB, 2MB]:

$$S_{\text{cache}} = 2(50) + 11(5) + 10(30) + 76(2) = 100 + 55 + 300 + 152 = 607 \text{ MB}$$

---

## 5. Build Time Modeling (Optimization Theory)

### The Problem

Total kaniko build time includes context download, layer execution, snapshot computation, cache lookup, and push. Optimizing Dockerfile instruction order minimizes expected build time.

### The Formula

Total build time:

$$T = T_{\text{context}} + \sum_{i=1}^{n} \left[(1 - H_i)(T_{\text{exec},i} + T_{\text{snap},i}) + H_i \cdot T_{\text{pull},i}\right] + T_{\text{push}}$$

where $H_i$ is the cache hit indicator.

Optimal instruction ordering minimizes:

$$\min_{\sigma \in \text{Perm}} E\left[\sum_{i=1}^{n} (1 - H_{\sigma(i)}) \cdot T_{\text{exec},\sigma(i)}\right]$$

subject to dependency constraints between instructions.

### Worked Examples

Reordering: move `COPY . .` (high change, low cost) after `RUN go mod download` (low change, high cost):

Before (bad order): COPY . . (p=0.95, 2s) then go mod download (p=0.10, 30s):

$$E[T] = 0.95 \times (2 + 30) + 0.05 \times 0.10 \times 30 = 30.4 + 0.15 = 30.55\text{s}$$

After (good order): go mod download (p=0.10, 30s) then COPY . . (p=0.95, 2s):

$$E[T] = 0.10 \times 30 + 0.90 \times 0.95 \times 2 + 0.90 \times 0.05 \times 0 = 3.0 + 1.71 = 4.71\text{s}$$

Savings: $30.55 - 4.71 = 25.84$s per build on average.

---

## 6. Reproducible Build Determinism (Hash Theory)

### The Problem

`--reproducible` mode strips timestamps and non-deterministic metadata so identical inputs produce identical output digests. The determinism property is a function from build inputs to output hash.

### The Formula

Reproducible build function:

$$f: (\text{Dockerfile}, \text{Context}, \text{Args}) \to \text{digest}$$

Determinism property:

$$\forall t_1, t_2: \; f_{t_1}(D, C, A) = f_{t_2}(D, C, A)$$

Non-deterministic sources eliminated:

$$\text{stripped} = \{\text{timestamps}, \text{build\_host}, \text{layer\_ids}, \text{created\_date}\}$$

Remaining non-determinism (not controlled by kaniko):

$$\text{residual} = \{\text{apt/apk mirror content}, \text{pip versions}, \text{network fetches}\}$$

### Worked Examples

Without `--reproducible`: same Dockerfile built at $t_1$ and $t_2$:

$$\text{digest}_{t_1} = \text{sha256:abc...} \neq \text{sha256:def...} = \text{digest}_{t_2}$$

With `--reproducible` and pinned dependencies:

$$\text{digest}_{t_1} = \text{sha256:abc...} = \text{sha256:abc...} = \text{digest}_{t_2}$$

---

## Prerequisites

- probability-theory, set-theory, graph-theory, information-theory, optimization-theory, hash-functions
