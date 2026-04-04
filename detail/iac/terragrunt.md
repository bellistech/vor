# The Mathematics of Terragrunt — Dependency Resolution, Configuration Composition & Execution Optimization

> *Terragrunt orchestrates Terraform modules through dependency graphs, composing configuration via hierarchical includes. The mathematics of topological sorting, merge algebras, and parallel scheduling determine execution order, configuration precedence, and optimal parallelism for multi-module infrastructure deployments.*

---

## 1. Dependency Resolution (Graph Theory)

### The Problem

Terragrunt's `dependency` and `dependencies` blocks form a DAG. `run-all` must execute modules in topological order. What determines the execution order and total deployment time?

### The Formula

For a dependency graph $G = (V, E)$ with $|V| = n$ modules:

$$\text{order} = \text{TopologicalSort}(G)$$

The critical path length determines minimum deployment time:

$$T_{min} = \max_{\text{path } p \in G} \sum_{v \in p} t_v$$

With parallelism $P$, the actual time is bounded:

$$T_{actual} \geq \max\left(T_{critical}, \frac{\sum t_v}{P}\right)$$

### Worked Examples

**Dependency chain: VPC (60s) -> Subnets (30s) -> RDS (120s) -> App (45s):**

$$T_{critical} = 60 + 30 + 120 + 45 = 255s$$

No parallelism helps here -- it is a strict chain.

**Diamond: VPC (60s) -> {Subnets(30s), SecurityGroups(20s)} -> RDS (120s):**

$$T_{critical} = 60 + \max(30, 20) + 120 = 210s$$

$$T_{sequential} = 60 + 30 + 20 + 120 = 230s$$

$$Speedup = 230/210 = 1.095\times$$

**Wide tree: VPC (60s) -> 10 independent services (90s each):**

$$T_{critical} = 60 + 90 = 150s$$

$$T_{sequential} = 60 + 10 \times 90 = 960s$$

$$T_{P=3} = 60 + \lceil 10/3 \rceil \times 90 = 60 + 360 = 420s$$

$$Speedup_{P=3} = 960/420 = 2.29\times$$

---

## 2. Configuration Merge Algebra (Lattice Theory)

### The Problem

Terragrunt merges inputs from multiple sources: root config, environment config, region config, and module-level overrides. The `merge()` function combines maps with last-writer-wins semantics. What is the precedence?

### The Formula

For $n$ configuration layers $C_1, C_2, ..., C_n$ (ordered by precedence, lowest first):

$$C_{final} = C_1 \oplus C_2 \oplus ... \oplus C_n$$

Where $\oplus$ is shallow merge (right operand wins on key conflict):

$$(A \oplus B)[k] = \begin{cases} B[k] & \text{if } k \in B \\ A[k] & \text{if } k \in A \setminus B \end{cases}$$

Total unique keys:

$$|C_{final}| = |C_1 \cup C_2 \cup ... \cup C_n|$$

Overridden keys at layer $i$:

$$|Overrides_i| = |C_i \cap \bigcup_{j < i} C_j|$$

### Worked Examples

**3 layers: root (5 keys), env (3 keys, 2 overlap), module (4 keys, 1 overlap with each):**

$$|C_{final}| = |C_1 \cup C_2 \cup C_3| = 5 + 3 - 2 + 4 - 1 - 1 + 0 = 8$$

Override chain for key `region`: root="us-east-1", env="eu-west-1", module=absent.

$$C_{final}[\text{region}] = \text{"eu-west-1"} \quad \text{(env overrides root, module absent)}$$

Deep merge is needed for nested maps. Without it:

$$\{tags: \{A: 1, B: 2\}\} \oplus \{tags: \{C: 3\}\} = \{tags: \{C: 3\}\}$$

With deep merge ($\oplus_d$):

$$\{tags: \{A: 1, B: 2\}\} \oplus_d \{tags: \{C: 3\}\} = \{tags: \{A: 1, B: 2, C: 3\}\}$$

---

## 3. State Key Uniqueness (Combinatorics)

### The Problem

`path_relative_to_include()` generates state keys like `dev/vpc/terraform.tfstate`. The key must be unique across all modules. What is the collision probability?

### The Formula

Given $n$ modules with paths of structure `{env}/{service}`:

$$|Keys| = |Envs| \times |Services|$$

Collision occurs when two modules resolve to the same relative path:

$$P(\text{collision}) = 0 \text{ (by construction, if directory structure is unique)}$$

But with `generate` blocks producing files:

$$P(\text{conflict}) = P(\text{two generates write same filename with different content})$$

This is why `if_exists = "overwrite_terragrunt"` is preferred -- it makes generation idempotent.

### Worked Examples

**3 environments, 15 services each:**

$$|Keys| = 3 \times 15 = 45 \text{ unique state files}$$

**Adding regions (3 envs x 2 regions x 15 services):**

$$|Keys| = 3 \times 2 \times 15 = 90 \text{ unique state files}$$

State bucket total size estimate (1MB avg per statefile):

$$S_{bucket} = 90 \times 1MB + 90 \times H \times 1MB$$

Where $H$ = number of historical versions kept per key.

---

## 4. Parallel Execution Scheduling (Scheduling Theory)

### The Problem

`run-all` with `--terragrunt-parallelism P` must schedule $n$ modules across $P$ workers respecting dependencies. This is the bounded-width topological scheduling problem.

### The Formula

The makespan (total time) with $P$ workers:

$$T_{makespan} = \max\left(T_{critical}, \frac{W}{P}\right)$$

Where $W = \sum_{i=1}^{n} t_i$ is total work. The efficiency:

$$E = \frac{W}{P \times T_{makespan}}$$

### Worked Examples

**20 modules, total work 1800s, critical path 300s, P=5:**

$$T_{makespan} = \max(300, 1800/5) = \max(300, 360) = 360s$$

$$E = \frac{1800}{5 \times 360} = 1.0 \quad (100\% \text{ efficient})$$

**Same with P=10:**

$$T_{makespan} = \max(300, 180) = 300s$$

$$E = \frac{1800}{10 \times 300} = 0.6 \quad (60\% \text{ -- critical path bottleneck})$$

Beyond $P^* = W / T_{critical} = 1800/300 = 6$ workers, adding more workers gives no benefit.

---

## 5. DRY Factor Calculation (Information Theory)

### The Problem

Terragrunt's value proposition is reducing duplication. The DRY factor measures how much configuration is shared versus duplicated.

### The Formula

$$DRY = 1 - \frac{L_{unique}}{L_{total\_without\_tg}}$$

Where:
- $L_{unique}$ = total lines of Terragrunt config (unique content)
- $L_{total\_without\_tg}$ = total lines if each module had its own backend, provider, versions

### Worked Examples

**45 modules, each needs 30 lines of boilerplate (backend + provider + versions):**

Without Terragrunt: $45 \times 30 = 1350$ lines of boilerplate.

With Terragrunt root config: $30$ lines (one root config).

$$DRY_{boilerplate} = 1 - \frac{30}{1350} = 97.8\%$$

**45 modules, each has 20 lines of inputs, 60% shared across environments:**

Without: $45 \times 20 = 900$ input lines.
With (shared env configs): $3 \times 20 + 45 \times 8 = 420$ lines.

$$DRY_{inputs} = 1 - \frac{420}{900} = 53.3\%$$

---

## 6. Error Propagation in run-all (Reliability)

### The Problem

When a module fails during `run-all`, dependent modules cannot proceed. What is the blast radius of a single failure?

### The Formula

For a module $v$ that fails, the affected set is all descendants:

$$Affected(v) = \{u \in V : v \rightsquigarrow u\}$$

$$BlastRadius = \frac{|Affected(v)| + 1}{|V|}$$

### Worked Examples

**VPC module fails (10 dependent modules out of 15 total):**

$$BlastRadius = \frac{11}{15} = 73.3\%$$

**Monitoring module fails (0 dependents):**

$$BlastRadius = \frac{1}{15} = 6.7\%$$

**Expected blast radius for random failure, assuming uniform failure probability:**

$$E[BlastRadius] = \frac{1}{|V|} \sum_{v \in V} \frac{|Affected(v)| + 1}{|V|}$$

Modules early in the dependency chain have disproportionate blast radius. This motivates making foundational modules (VPC, IAM) extremely reliable and well-tested.

---

## Prerequisites

- topological-sorting, DAG-scheduling
- merge-operations, lattice-theory
- scheduling-theory, makespan-optimization
- information-theory, redundancy-elimination
- graph-reachability, blast-radius-analysis
