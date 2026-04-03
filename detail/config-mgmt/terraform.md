# The Mathematics of Terraform — Infrastructure as Code Theory

> *Terraform models infrastructure as a dependency graph, plans changes as set differences between desired and current state, and executes with maximum parallelism bounded by the DAG width. Its core algorithms are graph theory, state reconciliation, and constraint solving.*

---

## 1. Dependency Graph (DAG Resolution)

### The Problem

Terraform builds a **directed acyclic graph** (DAG) of all resources. The graph determines creation order, parallelism, and blast radius.

### The Resource Graph

$$G = (R, E)$$

Where:
- $R$ = set of resources, data sources, and providers
- $E$ = dependency edges (explicit `depends_on` + implicit reference-based)

### Implicit Dependency Detection

$$\text{If resource } A \text{ references } B.\text{id} \implies (B, A) \in E$$

Terraform statically analyzes HCL to extract all references — no runtime discovery needed.

### Parallelism (Graph Width)

$$\text{max\_parallel} = \text{width}(G) = \max_{l \in \text{levels}} |\{r : \text{depth}(r) = l\}|$$

With `-parallelism=P` flag:

$$\text{effective\_parallel} = \min(P, \text{width}(G))$$

Default: $P = 10$.

### Worked Example

```hcl
VPC → Subnet → Security Group → EC2 Instance → EIP
VPC → Internet Gateway
Subnet → RDS Instance
```

| Depth | Resources | Parallel |
|:---:|:---|:---:|
| 0 | VPC | 1 |
| 1 | Subnet, Internet Gateway | 2 |
| 2 | Security Group, RDS Instance | 2 |
| 3 | EC2 Instance | 1 |
| 4 | EIP | 1 |

$$T_{apply} = \sum_{l=0}^{4} \max_{r \in \text{level}(l)} T_{api}(r)$$

$$\text{Width} = 2, \quad \text{Depth} = 5$$

### Total Apply Time

$$T_{total} = \sum_{l=0}^{D} \frac{\lceil |R_l| / P \rceil}{1} \times \max_{r \in R_l} T(r)$$

Where $D$ = depth, $|R_l|$ = resources at level $l$, $P$ = parallelism.

---

## 2. Plan Algorithm (State Reconciliation)

### The Problem

`terraform plan` computes the difference between desired state (config) and current state (state file + API). This is set arithmetic.

### The Changeset Formula

$$\text{Changeset} = \text{Desired State} - \text{Current State}$$

More precisely:

$$\text{Create} = \text{Desired} \setminus \text{Current}$$
$$\text{Destroy} = \text{Current} \setminus \text{Desired}$$
$$\text{Update} = \{r : r \in \text{Desired} \cap \text{Current} \text{ AND } \text{attrs}_{desired}(r) \neq \text{attrs}_{current}(r)\}$$
$$\text{No-op} = \{r : r \in \text{Desired} \cap \text{Current} \text{ AND } \text{attrs}_{desired}(r) = \text{attrs}_{current}(r)\}$$

### Worked Example

| Resource | In Config | In State | Attributes Match | Action |
|:---|:---:|:---:|:---:|:---|
| aws_instance.web | Yes | Yes | No (size changed) | Update |
| aws_instance.api | Yes | Yes | Yes | No-op |
| aws_instance.worker | Yes | No | N/A | Create |
| aws_instance.legacy | No | Yes | N/A | Destroy |

### Plan Summary Arithmetic

$$\text{Total changes} = |\text{Create}| + |\text{Update}| + |\text{Destroy}|$$

$$\text{Plan: } 1 \text{ to add, } 1 \text{ to change, } 1 \text{ to destroy}$$

---

## 3. State Drift Detection

### The Problem

Between Terraform runs, infrastructure can change outside of Terraform (manual changes, auto-scaling, etc.). Terraform detects this as **drift**.

### Drift Detection

$$\text{Drift}(r) = \text{state}(r) \neq \text{refresh}(r)$$

Where $\text{refresh}(r)$ = current API state.

### Three-Way Comparison

$$\text{plan\_action}(r) = f(\text{config}(r), \text{state}(r), \text{refresh}(r))$$

| Config | State | Refresh (API) | Action |
|:---|:---|:---|:---|
| t2.micro | t2.micro | t2.micro | No-op |
| t2.small | t2.micro | t2.micro | Update (config change) |
| t2.micro | t2.micro | t2.large | Update (drift detected, revert) |
| t2.small | t2.micro | t2.large | Update (both changed) |

### Refresh Latency

$$T_{refresh} = \sum_{r \in \text{state}} T_{api\_read}(r) / P$$

For 200 resources at 200ms per API call, $P=10$:

$$T_{refresh} = \frac{200 \times 0.2}{10} = 4\text{ seconds}$$

---

## 4. Module Composition (Nested Graphs)

### The Problem

Modules are reusable Terraform configurations. They compose into a nested graph.

### Module Expansion

$$G_{total} = G_{root} \cup \bigcup_{m \in \text{modules}} G_m$$

### Resource Addressing

$$\text{address} = \text{module.name[index].resource\_type.name[index]}$$

### Module Count/For_Each Multiplication

$$|\text{instances}| = \text{count} \times |\text{resources in module}|$$

| Module | count/for_each | Resources/Module | Total Resources |
|:---|:---:|:---:|:---:|
| vpc | 1 | 5 | 5 |
| app_server | 3 | 8 | 24 |
| database | for_each(2) | 12 | 24 |
| monitoring | 1 | 3 | 3 |
| **Total** | | | **56** |

### State Size Growth

$$S_{state} \approx |R| \times S_{avg\_resource}$$

Where $S_{avg\_resource} \approx 1\text{-}10$ KB depending on resource complexity.

For 1,000 resources at 5 KB average: $S_{state} \approx 5$ MB.

---

## 5. Provider API Rate Limits

### The Problem

Cloud providers impose API rate limits. Terraform must operate within these constraints.

### Rate Limit Formula

$$\text{Max resources/second} = \frac{\text{API rate limit}}{R_{api\_calls\_per\_resource}}$$

| Provider | Rate Limit | Calls/Resource | Max Resources/s |
|:---|:---:|:---:|:---:|
| AWS (EC2) | 100 req/s | 2-3 | 33-50 |
| AWS (IAM) | 20 req/s | 1-2 | 10-20 |
| Azure | 200 req/5min | 2 | ~1.7 |
| GCP | 10 req/s | 2 | 5 |

### Retry with Exponential Backoff

$$T_{retry}(n) = \min(T_{base} \times 2^n + \text{jitter}, T_{max})$$

Where $n$ = retry attempt, $\text{jitter} \in [0, T_{base}]$.

| Attempt | Backoff (base=1s) | Cumulative |
|:---:|:---:|:---:|
| 1 | 2s + jitter | ~2s |
| 2 | 4s + jitter | ~6s |
| 3 | 8s + jitter | ~14s |
| 4 | 16s + jitter | ~30s |
| 5 | 32s (capped) | ~62s |

---

## 6. Workspace Isolation (State Partitioning)

### The Problem

Workspaces maintain separate state files for the same configuration. This enables environment isolation.

### State Isolation

$$\text{state}(ws_i) \cap \text{state}(ws_j) = \emptyset \quad \forall i \neq j$$

Each workspace manages completely independent infrastructure:

$$\text{resources}(ws_{dev}) \cap \text{resources}(ws_{prod}) = \emptyset$$

### Cost Multiplication

$$\text{Total resources} = |W| \times |R_{config}|$$
$$\text{Total cost} = |W| \times \text{cost}(R_{config})$$

For a config with \$500/month in resources across 4 workspaces:

$$\text{Monthly cost} = 4 \times \$500 = \$2{,}000$$

---

## 7. Import and State Surgery

### The Problem

Importing existing resources requires mapping real infrastructure to Terraform addresses.

### Import Mapping

$$\text{import}: (\text{resource\_id}_{cloud}) \rightarrow (\text{address}_{terraform})$$

### State Move (Refactoring)

$$\text{moved}: \text{address}_{old} \rightarrow \text{address}_{new}$$

Without move: Terraform sees destroy + create (dangerous).
With move: Terraform sees no change (safe).

### Blast Radius Calculation

$$\text{Blast radius}(r) = |\text{descendants}(r, G)| + 1$$

Deleting a VPC cascades to all resources that depend on it:

| Resource | Blast Radius |
|:---|:---:|
| aws_eip | 1 |
| aws_instance | 2 (instance + EIP) |
| aws_subnet | 5 (subnet + instances + EIPs) |
| aws_vpc | 15 (VPC + all children) |

---

## 8. Summary of Functions by Type

| Formula | Math Type | Domain |
|:---|:---|:---|
| $\text{Desired} \setminus \text{Current}$ | Set difference | Plan algorithm |
| $\text{width}(G)$ | Graph theory | Parallelism |
| $\text{toposort}(G)$ | Graph theory | Execution order |
| $\text{config} \neq \text{state} \neq \text{refresh}$ | Three-way compare | Drift detection |
| $|W| \times |R|$ | Multiplication | Workspace scaling |
| $T_{base} \times 2^n$ | Exponential backoff | Rate limit handling |
| $|\text{descendants}(r)| + 1$ | Tree traversal | Blast radius |

---

*Terraform's `plan` is a set difference, its `apply` is a topological sort, and its parallelism is bounded by graph width. Every `terraform apply` is a lesson in applied graph theory operating on real cloud infrastructure.*
