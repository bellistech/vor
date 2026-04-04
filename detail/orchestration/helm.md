# The Mathematics of Helm — Kubernetes Package Management

> *Helm is the package manager for Kubernetes — it renders Go templates into manifests, manages release versioning with a state machine, and resolves chart dependencies as a DAG. The template engine, revision model, and dependency solver all have precise formulations.*

---

## 1. Template Rendering (String Substitution Algebra)

### The Problem

Helm charts are Go templates with values. The rendering function transforms templates + values into Kubernetes manifests.

### The Rendering Function

$$\text{render}: T \times V \rightarrow M$$

Where:
- $T$ = set of template files
- $V$ = values (merged hierarchy)
- $M$ = set of rendered manifests

### Values Merge Hierarchy

Values are merged with a strict precedence:

$$V_{final} = V_{defaults} \triangleleft V_{parent} \triangleleft V_{user\_values} \triangleleft V_{set\_flags}$$

Where $\triangleleft$ means "right overrides left for matching keys."

### Merge Depth

For nested values, the merge is recursive:

$$V_{final}[a.b.c] = \text{last defined}(V_{defaults}[a.b.c], V_{parent}[a.b.c], V_{user}[a.b.c], V_{set}[a.b.c])$$

### Worked Example

```yaml
# values.yaml (defaults)        # user-values.yaml           # --set flag
replicaCount: 1                  replicaCount: 3              image.tag=v2.0
image:                           image:
  tag: latest                      repository: myrepo
  repository: nginx
```

| Key | Default | User | --set | Final |
|:---|:---:|:---:|:---:|:---:|
| replicaCount | 1 | 3 | — | 3 |
| image.repository | nginx | myrepo | — | myrepo |
| image.tag | latest | — | v2.0 | v2.0 |

---

## 2. Release Revision Model (State Machine)

### The Problem

Each Helm release maintains a revision history. Operations transition releases through defined states.

### Release States

$$\text{States} = \{\text{pending-install}, \text{deployed}, \text{pending-upgrade}, \text{pending-rollback}, \text{superseded}, \text{failed}, \text{uninstalling}\}$$

### State Transitions

$$\text{install} \rightarrow \text{pending-install} \xrightarrow{\text{success}} \text{deployed}$$
$$\text{install} \rightarrow \text{pending-install} \xrightarrow{\text{failure}} \text{failed}$$
$$\text{upgrade} \rightarrow \text{pending-upgrade} \xrightarrow{\text{success}} \text{deployed} \quad (\text{previous} \rightarrow \text{superseded})$$
$$\text{rollback}(r) \rightarrow \text{pending-rollback} \xrightarrow{\text{success}} \text{deployed}$$

### Revision Numbering

$$\text{rev}_{install} = 1$$
$$\text{rev}_{upgrade} = \text{rev}_{current} + 1$$
$$\text{rev}_{rollback} = \text{rev}_{current} + 1 \quad \text{(new revision, not revert)}$$

### History Growth

After $U$ upgrades and $R$ rollbacks:

$$\text{Total revisions} = 1 + U + R$$

With `--history-max=M`:

$$\text{Stored revisions} = \min(1 + U + R, M)$$

---

## 3. Chart Dependency Resolution (DAG)

### The Problem

Charts can depend on other charts (subcharts). Dependencies form a DAG that must be resolved.

### Dependency DAG

$$G = (C, D) \text{ where } C = \text{charts}, D = \text{dependency edges}$$

### Resolution Order

$$\text{install\_order} = \text{reverse\_toposort}(G)$$

Dependencies are installed bottom-up: leaf charts first, root chart last.

### Dependency Conditions

Charts can conditionally include subcharts:

$$\text{include}(subchart) = \begin{cases}
\text{true} & \text{if } V[\text{condition\_key}] = \text{true} \\
\text{true} & \text{if no condition specified} \\
\text{false} & \text{if } V[\text{condition\_key}] = \text{false}
\end{cases}$$

### Version Constraint Solving

Dependencies specify version ranges using semantic versioning:

$$\text{constraint}: \texttt{>= 1.2.0, < 2.0.0}$$

$$\text{satisfies}(v, c) = v \in [\text{lower}(c), \text{upper}(c))$$

The solver finds the latest version satisfying all constraints:

$$v_{selected} = \max\{v \in \text{available} : \text{satisfies}(v, c)\}$$

### Worked Example: Dependency Tree

```
my-app (v1.0.0)
├── postgresql (>= 12.0.0, < 13.0.0) → 12.8.3
├── redis (>= 17.0.0) → 17.3.14
└── common (>= 2.0.0) → 2.2.5
    └── (no sub-dependencies)
```

| Chart | Constraint | Available | Selected |
|:---|:---|:---|:---:|
| postgresql | >= 12.0.0, < 13.0.0 | 12.1.0, 12.8.3, 13.0.1 | 12.8.3 |
| redis | >= 17.0.0 | 17.0.0, 17.3.14, 18.0.0 | 17.3.14 |
| common | >= 2.0.0 | 2.0.0, 2.2.5 | 2.2.5 |

---

## 4. Hook Execution Model (Weighted Ordering)

### The Problem

Helm hooks run at specific lifecycle points. Multiple hooks at the same point are ordered by weight.

### Hook Weights

$$\text{execution\_order} = \text{sort\_by}(\text{weight}, \text{ascending})$$

$$w \in \mathbb{Z} \quad \text{(any integer, default = 0)}$$

### Hook Lifecycle Points

| Phase | Runs | Use Case |
|:---|:---|:---|
| pre-install | Before any resources | DB migration |
| post-install | After all resources | Load seed data |
| pre-upgrade | Before upgrade | Backup |
| post-upgrade | After upgrade | Cache warm |
| pre-delete | Before deletion | Drain connections |
| post-delete | After deletion | Cleanup external |
| test | On `helm test` | Integration tests |

### Hook Deletion Policies

$$\text{delete} = \begin{cases}
\text{before-hook-creation} & \text{delete previous hook resource before new one} \\
\text{hook-succeeded} & \text{delete after success} \\
\text{hook-failed} & \text{delete after failure}
\end{cases}$$

---

## 5. Three-Way Merge (Upgrade Strategy)

### The Problem

Helm 3 uses a three-way strategic merge for upgrades:

### The Three Sources

$$\text{Merge}(\text{old\_chart}, \text{live\_state}, \text{new\_chart}) \rightarrow \text{patch}$$

### Merge Logic

For each field:

| Old Chart | Live State | New Chart | Action |
|:---|:---|:---|:---|
| A | A | A | No change |
| A | A | B | Update to B |
| A | B | A | Keep B (manual edit preserved) |
| A | B | B | Keep B |
| A | B | C | Update to C |
| — | — | A | Add A |
| A | A | — | Delete |
| A | B | — | Delete (but warn) |

### Why Three-Way Matters

Two-way merge (Helm 2) would overwrite manual changes. Three-way preserves them:

$$\text{manual\_edits} = \text{live} - \text{old\_chart}$$
$$\text{chart\_changes} = \text{new\_chart} - \text{old\_chart}$$
$$\text{result} = \text{old\_chart} + \text{manual\_edits} + \text{chart\_changes}$$

Conflicts occur when both sides change the same field: $\text{manual\_edits} \cap \text{chart\_changes} \neq \emptyset$.

---

## 6. Chart Size and Rendering Performance

### The Problem

Large charts with many templates and values have non-trivial rendering costs.

### Rendering Complexity

$$T_{render} = O\left(\sum_{t \in T} |t| \times D_t\right)$$

Where $|t|$ = template size and $D_t$ = depth of value lookups.

### Manifest Count Growth

$$|\text{manifests}| = |T_{base}| + \sum_{s \in \text{subcharts}} |T_s| \times \mathbb{1}[\text{enabled}(s)]$$

### Real-World Chart Sizes

| Chart | Templates | Subcharts | Total Manifests | Render Time |
|:---|:---:|:---:|:---:|:---:|
| nginx | 5 | 0 | 5 | 20 ms |
| wordpress | 12 | 2 | 25 | 80 ms |
| airflow | 35 | 4 | 60 | 200 ms |
| istio | 80+ | 5 | 150+ | 500+ ms |

---

## 7. OCI Registry Storage (Chart Distribution)

### The Problem

Helm 3 supports storing charts in OCI registries alongside container images.

### Chart Archive Size

$$S_{chart} = S_{templates} + S_{values} + S_{crds} + S_{metadata}$$

Typical chart sizes:

| Component | Size Range | Compresses To |
|:---|:---:|:---:|
| Templates | 10-500 KB | 3-100 KB |
| Values | 2-50 KB | 1-15 KB |
| CRDs | 0-5 MB | 0-1 MB |
| Chart.yaml + README | 5-20 KB | 2-8 KB |

### Pull Efficiency

$$T_{pull} = \frac{S_{chart}}{BW} + T_{auth} + T_{decompress}$$

Charts are typically 10-100 KB compressed — orders of magnitude smaller than container images.

---

## 8. Summary of Functions by Type

| Formula | Math Type | Domain |
|:---|:---|:---|
| $V_{def} \triangleleft V_{user} \triangleleft V_{set}$ | Merge algebra | Values resolution |
| $1 + U + R$ | Linear counting | Revision history |
| $\text{reverse\_toposort}(G)$ | Graph theory | Dependency order |
| $\text{old} + \text{edits} + \text{changes}$ | Three-way merge | Upgrade strategy |
| $\max\{v : v \in c\}$ | Constraint solving | Version selection |
| $O(\Sigma |t| \times D_t)$ | Complexity | Render performance |

---

*Helm transforms Kubernetes from "apply a pile of YAML" to "install a versioned, configurable, rollbackable package" — the template algebra, three-way merge, and dependency DAG make this possible.*

## Prerequisites

- Kubernetes fundamentals (resources, namespaces, kubectl)
- Go template syntax
- Semantic versioning
- YAML proficiency

## Complexity

- Beginner: installing charts, overriding values, basic upgrades
- Intermediate: chart authoring, dependencies, hooks, release management
- Advanced: three-way merge strategy, OCI registry distribution, library charts, custom template functions
