# The Mathematics of Kustomize — Overlay Algebra and YAML Merge Semantics

> *Kustomize implements a compositional algebra over Kubernetes YAML resources, where overlays are morphisms in a category of configurations, strategic merge patches follow structural recursion over typed schemas, and transformer pipelines form a monoid of endofunctions on resource sets.*

---

## 1. Resource Set Algebra (Set Theory)

### Resource Identity

Each Kubernetes resource is uniquely identified by a tuple:

$$r = (\text{apiVersion}, \text{kind}, \text{namespace}, \text{name})$$

This tuple forms the **resource ID** in Kustomize's internal model. A kustomization output is a set of resources:

$$K = \{r_1, r_2, \ldots, r_n\}$$

### Base and Overlay Composition

The overlay operation combines a base with modifications:

$$O(B) = T_m \circ T_{m-1} \circ \cdots \circ T_1(B \cup R_{\text{additional}})$$

Where:
- $B$ = base resource set
- $R_{\text{additional}}$ = overlay-specific resources
- $T_i$ = transformers (patches, labels, namespace, images, etc.)

### Multi-Base Composition

When an overlay references multiple bases:

$$K = T\left(\bigcup_{i=1}^{k} B_i \cup R_{\text{overlay}}\right)$$

Resource conflict resolution: if two bases contain the same resource ID, Kustomize reports an error (no implicit merge).

---

## 2. Strategic Merge Patch (Structural Recursion)

### Schema-Aware Merge

Strategic merge patches follow the Kubernetes API schema to determine merge behavior. For each field, the schema defines a merge strategy:

$$\text{merge}(a, b, s) = \begin{cases} b & \text{if } s = \text{replace} \\ a \cup b & \text{if } s = \text{merge (by key)} \\ f_{\text{recursive}}(a, b) & \text{if } s = \text{object (recursive)} \end{cases}$$

### List Merge Strategies

Kubernetes lists have three merge strategies determined by the schema:

| Strategy | Key | Behavior |
|:---|:---|:---|
| `merge` on key | `name`, `port`, etc. | Match by key, merge matched items |
| `replace` | none | Replace entire list |
| `atomic` | none | Replace entire list (no partial merge) |

For `spec.containers` (merge key: `name`):

$$\text{merge}(L_{\text{base}}, L_{\text{patch}}) = \{c_{\text{merged}} : c \in L_{\text{base}} \cup L_{\text{patch}}\}$$

Where items with matching `name` are merged, new items are added, and items not in the patch are preserved.

### Formal Merge Algorithm

For objects $A$ (base) and $B$ (patch):

$$\text{merge}(A, B) = \begin{cases} B & \text{if } B \text{ is scalar or null} \\ \{k: \text{merge}(A[k], B[k]) \mid k \in \text{keys}(A) \cup \text{keys}(B)\} & \text{if } A, B \text{ are objects} \\ \text{listMerge}(A, B, \text{mergeKey}) & \text{if } A, B \text{ are lists with mergeKey} \\ B & \text{if } A, B \text{ are lists without mergeKey} \end{cases}$$

---

## 3. JSON Patch as Sequence (Algebraic Operations)

### Patch Operations

JSON Patch (RFC 6902) defines atomic operations on a JSON document:

$$p: D \rightarrow D'$$

Each operation is a function:

| Operation | Function | Type |
|:---|:---|:---|
| `add` | $D' = D \cup \{(\text{path}, \text{value})\}$ | Insertion |
| `remove` | $D' = D \setminus \{(\text{path}, \_)\}$ | Deletion |
| `replace` | $D' = (D \setminus \{(\text{path}, \_)\}) \cup \{(\text{path}, \text{value})\}$ | Update |
| `move` | $D' = \text{add}(\text{remove}(D, \text{from}), \text{path}, D[\text{from}])$ | Relocation |
| `copy` | $D' = \text{add}(D, \text{path}, D[\text{from}])$ | Duplication |
| `test` | $D' = D$ if $D[\text{path}] = \text{value}$, else error | Assertion |

### Patch Composition

A patch is a sequence of operations applied left to right:

$$P = [p_1, p_2, \ldots, p_k]$$

$$\text{apply}(D, P) = p_k(\cdots(p_2(p_1(D)))\cdots)$$

Composition is associative but not commutative:

$$\text{apply}(D, P_1 \cdot P_2) = \text{apply}(\text{apply}(D, P_1), P_2)$$

$$P_1 \cdot P_2 \neq P_2 \cdot P_1 \quad \text{(in general)}$$

### Path Notation

JSON Pointer (RFC 6901):

$$\text{path} = / \text{segment}_1 / \text{segment}_2 / \cdots / \text{segment}_n$$

Array indexing: `/spec/containers/0/env/-` (append to array).

---

## 4. Transformer Pipeline as Monoid (Category Theory)

### Transformer Definition

Each Kustomize transformer is an endofunction on the resource set:

$$T: \mathcal{P}(\text{Resources}) \rightarrow \mathcal{P}(\text{Resources})$$

### Transformer Types

| Transformer | Effect |
|:---|:---|
| `commonLabels` | $T_L(R) = \{r \oplus \text{labels} : r \in R\}$ |
| `namespace` | $T_N(R) = \{r[\text{ns} \mapsto N] : r \in R, r \text{ namespaced}\}$ |
| `namePrefix` | $T_P(R) = \{r[\text{name} \mapsto p + r.\text{name}] : r \in R\}$ |
| `images` | $T_I(R) = \{r[\text{image} \mapsto I(r.\text{image})] : r \in R\}$ |
| `patches` | $T_{\text{patch}}(R) = \{\text{merge}(r, \text{patch}) : r \in R \text{ if matched}\}$ |

### Monoid Structure

The set of transformers forms a monoid under composition:

$$(T, \circ, \text{id})$$

- **Closure:** $T_1 \circ T_2$ is a transformer
- **Associativity:** $(T_1 \circ T_2) \circ T_3 = T_1 \circ (T_2 \circ T_3)$
- **Identity:** $\text{id}(R) = R$

The Kustomize pipeline applies transformers in a fixed order:

$$K = T_{\text{replacements}} \circ T_{\text{patches}} \circ T_{\text{images}} \circ T_{\text{namespace}} \circ T_{\text{labels}} \circ T_{\text{prefix}} \circ T_{\text{generators}}$$

---

## 5. ConfigMap Hash Suffix (Hash Theory)

### Content-Based Naming

Kustomize appends a hash suffix to generated ConfigMaps and Secrets:

$$\text{name}' = \text{name} + \text{"-"} + \text{hash}(content)[:10]$$

The hash function:

$$h = \text{FNV-32a}(\text{sorted(data entries)})$$

Encoded as a base-36 string (lowercase alphanumeric).

### Rolling Update Trigger

When content changes, the hash changes, which changes the name:

$$\text{content}_1 \neq \text{content}_2 \implies h_1 \neq h_2 \implies \text{name}_1 \neq \text{name}_2$$

All referencing Deployments see a new ConfigMap name in their PodSpec, triggering a rolling update:

$$\Delta(\text{ConfigMap name in PodSpec}) \implies \text{RollingUpdate}$$

### Collision Probability

FNV-32a produces 32-bit hashes, but only 10 base-36 characters are used (~51.7 bits of representation from 32 bits of hash):

$$P(\text{collision}) = \frac{1}{2^{32}} \approx 2.3 \times 10^{-10} \text{ per pair}$$

For $n$ ConfigMaps (birthday problem):

$$P(\text{any collision}) \approx \frac{n^2}{2^{33}}$$

At $n = 1000$: $P \approx 1.2 \times 10^{-4}$ (negligible in practice).

---

## 6. Overlay DAG and Diamond Dependency (Graph Theory)

### Kustomization Dependency Graph

Kustomizations form a DAG through `resources` references:

$$G = (K, E)$$

Where $K$ = kustomization directories and $E$ = `resources` references.

### Diamond Dependency Problem

```
     overlay-prod
    /            \
  base-app    base-monitoring
    \            /
     shared-config
```

If `shared-config` is included through two paths, Kustomize resolves this by:

1. Each path produces its own copy of resources
2. Resource ID conflicts are detected
3. Error if same resource ID appears with different content

Resolution: use `components` for shared functionality instead of double-including bases.

### DAG Depth and Build Time

Build time is proportional to the DAG traversal:

$$T_{\text{build}} = O(|K| + |E| + \sum_{k \in K} |R_k| \times |T_k|)$$

Where $|R_k|$ = resources in kustomization $k$ and $|T_k|$ = transformers applied.

---

## 7. Component Algebra (Module Theory)

### Components as Mixins

Kustomize Components are reusable transformation modules:

$$C: \mathcal{P}(\text{Resources}) \rightarrow \mathcal{P}(\text{Resources})$$

Components can add resources and apply patches:

$$C(R) = T_{\text{patches}}(R \cup R_{\text{component}})$$

### Composition Independence

Components should be independently composable:

$$C_1 \circ C_2 = C_2 \circ C_1 \quad \text{(ideally commutative)}$$

This holds when components modify disjoint resource fields:

$$\text{fields}(C_1) \cap \text{fields}(C_2) = \emptyset \implies C_1 \circ C_2 = C_2 \circ C_1$$

When components overlap, application order matters:

$$\text{fields}(C_1) \cap \text{fields}(C_2) \neq \emptyset \implies \text{order-dependent}$$

### Component Cardinality

For $n$ available components, the number of possible overlay configurations:

$$|\text{configurations}| = 2^n \quad \text{(each component included or not)}$$

For 5 components (monitoring, logging, security, ingress, autoscaling):

$$|\text{configurations}| = 32$$

---

## Prerequisites

set-theory, algebraic-structures, graph-theory, hash-theory, category-theory

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| Strategic merge patch | $O(d \times b)$ — d = depth, b = breadth | $O(S)$ — result size |
| JSON patch application | $O(k)$ — k = operations | $O(S)$ — document size |
| Transformer pipeline | $O(R \times T)$ — resources x transformers | $O(R)$ |
| Hash suffix computation | $O(S)$ — S = content size | $O(1)$ |
| DAG build resolution | $O(K + E)$ — kustomizations + edges | $O(R_{\text{total}})$ |
| Resource ID conflict check | $O(R \log R)$ — sort + scan | $O(R)$ |
