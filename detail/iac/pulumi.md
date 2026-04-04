# The Mathematics of Pulumi — Dependency Graphs, State Convergence & Resource Diffing

> *Pulumi builds a directed acyclic graph of resource dependencies, resolves them via topological sort, and computes minimal change sets through state diffing. The mathematics of graph theory, convergence, and set operations explains deployment ordering, parallelism limits, and why certain updates require replacement.*

---

## 1. Resource Dependency Graph (Graph Theory)

### The Problem

Resources have dependencies (a subnet needs a VPC, an instance needs a subnet). Pulumi must create resources in dependency order and maximize parallelism for independent resources. This is topological sorting with parallel execution.

### The Formula

For a DAG $G = (V, E)$ with $|V| = n$ resources and $|E| = m$ dependencies:

$$T_{sequential} = \sum_{i=1}^{n} t_i$$

$$T_{parallel} = \sum_{l=1}^{L} \max_{v \in layer_l} t_v$$

Where $L$ is the number of layers in the longest-path decomposition:

$$L = \text{length of longest path in } G + 1$$

Speedup from parallelism:

$$S = \frac{T_{sequential}}{T_{parallel}} \leq \frac{n}{L}$$

### Worked Examples

**VPC (30s) -> 3 Subnets (10s each) -> 3 Instances (45s each):**

$$T_{sequential} = 30 + 3(10) + 3(45) = 195s$$

$$L = 3 \text{ layers: VPC | Subnets | Instances}$$

$$T_{parallel} = 30 + 10 + 45 = 85s$$

$$S = 195/85 = 2.29\times$$

**Star topology (1 VPC -> 10 independent subnets):**

$$T_{sequential} = 30 + 10(10) = 130s$$

$$T_{parallel} = 30 + 10 = 40s$$

$$S = 130/40 = 3.25\times$$

Maximum parallelism is bounded by the graph's width (largest antichain), given by Dilworth's theorem:

$$W = \max_l |layer_l|$$

---

## 2. State Diff Algorithm (Set Theory)

### The Problem

Given desired state $D$ and current state $C$, compute the minimal set of operations: creates, updates, deletes.

### The Formula

$$Creates = D \setminus C = \{r \in D : r \notin C\}$$

$$Deletes = C \setminus D = \{r \in C : r \notin D\}$$

$$Updates = \{r \in D \cap C : props_D(r) \neq props_C(r)\}$$

$$Unchanged = \{r \in D \cap C : props_D(r) = props_C(r)\}$$

Total operations:

$$|Ops| = |Creates| + |Deletes| + |Updates|$$

### Worked Examples

**Current: {VPC, Subnet-A, Subnet-B, SG-1}. Desired: {VPC, Subnet-A, Subnet-C, SG-1(modified)}:**

$$Creates = \{Subnet\text{-}C\}$$

$$Deletes = \{Subnet\text{-}B\}$$

$$Updates = \{SG\text{-}1\}$$

$$Unchanged = \{VPC, Subnet\text{-}A\}$$

$$|Ops| = 1 + 1 + 1 = 3$$

---

## 3. Replace vs Update Decision (Property Classification)

### The Problem

Some resource properties can be updated in-place; others force replacement (delete + create). The diff must classify each changed property.

### The Formula

For resource $r$ with changed properties $\Delta P$:

$$Action(r) = \begin{cases} \text{update} & \text{if } \Delta P \subseteq P_{mutable} \\ \text{replace} & \text{if } \Delta P \cap P_{immutable} \neq \emptyset \end{cases}$$

For replace with `deleteBeforeReplace`:

$$T_{replace} = T_{delete} + T_{create}$$

For replace with `createBeforeReplace` (default):

$$T_{replace} = T_{create} + T_{delete} \text{ (zero downtime)}$$

### Worked Examples

**EC2 instance: change `instance_type` (mutable) and `ami` (immutable):**

$$\Delta P = \{instance\_type, ami\}$$

$$\Delta P \cap P_{immutable} = \{ami\} \neq \emptyset \Rightarrow \text{replace}$$

**Cascade effect: replacement of VPC forces replacement of $k$ dependent resources:**

$$|Replacements| = 1 + |\text{descendants}(VPC)| = 1 + k$$

This is why stable resource naming and careful use of `protect` are essential.

---

## 4. Convergence Iterations (Fixed-Point Theory)

### The Problem

`pulumi up` should be idempotent: running it twice with no code changes should produce zero operations. Convergence failure indicates drift or non-deterministic providers.

### The Formula

A deployment function $f: State \rightarrow State$ converges if:

$$\exists n: f^n(S_0) = f^{n+1}(S_0)$$

The fixed point $S^* = f(S^*)$ represents the desired state. Non-convergence occurs when:

$$\forall n: f^n(S_0) \neq f^{n+1}(S_0)$$

This happens with computed defaults, timestamp fields, or random values regenerated each run.

### Worked Examples

**Convergent: Static tag values**

$$f(\{tags: \{env: "dev"\}\}) = \{tags: \{env: "dev"\}\} \quad \checkmark$$

**Non-convergent: Tag with timestamp**

$$f_1: tags.updated = "2024-01-01T00:00:00"$$

$$f_2: tags.updated = "2024-01-01T00:00:05" \neq f_1$$

Each apply changes the value, triggering another update. Use `ignoreChanges` to break the cycle.

---

## 5. Stack Reference Topology (DAG Composition)

### The Problem

Multiple stacks reference each other's outputs. Circular references create deadlocks. The inter-stack dependency graph must also be a DAG.

### The Formula

Given $S$ stacks with reference edges $E$:

$$\text{Valid} \iff G_{stacks} = (S, E) \text{ is acyclic}$$

Deployment order:

$$\text{order} = \text{TopologicalSort}(G_{stacks})$$

Maximum parallel stacks at any layer:

$$P_{max} = \max_l |layer_l(G_{stacks})|$$

### Worked Examples

**3-tier: network -> compute -> monitoring:**

$$\text{order} = [\text{network}, \text{compute}, \text{monitoring}]$$

$$P_{max} = 1 \text{ (fully serial)}$$

**Diamond: network -> {compute, database} -> app:**

$$\text{order} = [\text{network}, \{\text{compute}, \text{database}\}, \text{app}]$$

$$P_{max} = 2 \text{ (compute and database in parallel)}$$

---

## 6. Secret Encryption (Cryptography)

### The Problem

Pulumi encrypts secret values in state. The encryption uses envelope encryption: a data key encrypts the value, and a master key encrypts the data key.

### The Formula

$$C = E_{DEK}(plaintext)$$

$$C_{DEK} = E_{KEK}(DEK)$$

$$State = \{C, C_{DEK}\}$$

The security margin depends on the key size $k$:

$$\text{Brute force cost} = O(2^k)$$

For AES-256-GCM (Pulumi's default):

$$2^{256} \approx 1.16 \times 10^{77} \text{ operations}$$

At $10^{18}$ ops/sec, this takes $3.67 \times 10^{51}$ years.

### Worked Examples

**Secrets per stack with passphrase provider:**

Key derivation uses PBKDF2:

$$DEK = PBKDF2(passphrase, salt, iterations, keyLen)$$

With 100,000 iterations and a 20-character passphrase from a 95-character set:

$$\text{Passphrase space} = 95^{20} = 3.58 \times 10^{39}$$

$$\text{Effective security} = \log_2(3.58 \times 10^{39} / 100000) \approx 115 \text{ bits}$$

---

## Prerequisites

- directed-acyclic-graphs, topological-sort
- set-theory, set-operations
- fixed-point-theory, idempotence
- envelope-encryption, AES-GCM
- Dilworth-theorem, antichain
