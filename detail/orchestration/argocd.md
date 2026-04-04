# The Mathematics of Argo CD — GitOps Convergence and Sync Wave Topology

> *Argo CD implements a convergence controller whose reconciliation loop drives cluster state toward the Git-declared desired state, with sync waves imposing a topological ordering on resource application and health checks providing a boolean lattice of application readiness.*

---

## 1. GitOps Convergence Model (Control Theory)

### State Space

Define the system state spaces:

$$S_{\text{desired}} \in \mathcal{M} \quad \text{(manifests from Git)}$$
$$S_{\text{live}} \in \mathcal{M} \quad \text{(current cluster state)}$$

The GitOps controller drives:

$$S_{\text{live}}(t) \rightarrow S_{\text{desired}}(t)$$

### Drift Detection

At each reconciliation cycle, Argo CD computes the diff:

$$\Delta(t) = S_{\text{desired}}(t) \ominus S_{\text{live}}(t)$$

Where $\ominus$ is the structured diff operation on Kubernetes resources.

Sync status:

$$\text{status}(t) = \begin{cases} \text{Synced} & \text{if } \Delta(t) = \emptyset \\ \text{OutOfSync} & \text{if } \Delta(t) \neq \emptyset \end{cases}$$

### Convergence Guarantee

With `selfHeal: true`, the controller applies corrections:

$$S_{\text{live}}(t + \epsilon) = \text{apply}(S_{\text{live}}(t), \Delta(t))$$

Convergence time:

$$T_{\text{converge}} = T_{\text{detect}} + T_{\text{sync}} + T_{\text{health}}$$

Where:
- $T_{\text{detect}}$ = polling interval (default 3 minutes) or webhook latency (~seconds)
- $T_{\text{sync}}$ = `kubectl apply` time for all resources
- $T_{\text{health}}$ = time for health checks to pass

### Self-Heal as Feedback Loop

The self-heal mechanism is a negative feedback controller:

$$u(t) = K_p \cdot \Delta(t)$$

Where $K_p = 1$ (full correction per cycle). This is a proportional controller with unity gain, guaranteeing convergence in one sync cycle if the apply succeeds:

$$\|\Delta(t + 1)\| = 0 \quad \text{if apply succeeds}$$

---

## 2. Sync Waves as Topological Sort (Graph Theory)

### Wave Ordering

Sync waves define a partial order on resources:

$$r_i \prec r_j \iff \text{wave}(r_i) < \text{wave}(r_j)$$

Resources within the same wave are applied concurrently (unordered):

$$\text{wave}(r_i) = \text{wave}(r_j) \implies r_i \| r_j \quad \text{(parallel)}$$

### Dependency DAG

The sync process creates a DAG from wave annotations:

$$G_{\text{sync}} = (R, E)$$

$$E = \{(r_i, r_j) : \text{wave}(r_i) < \text{wave}(r_j)\}$$

This is a layered graph where each layer is a wave number:

$$L_w = \{r \in R : \text{wave}(r) = w\}$$

### Execution Model

Sync proceeds wave by wave:

$$\text{for } w = w_{\min} \text{ to } w_{\max}:$$
$$\quad \text{apply}(L_w)$$
$$\quad \text{wait\_healthy}(L_w)$$
$$\quad \text{if } \neg \text{healthy}(L_w): \text{abort}$$

Total sync time:

$$T_{\text{sync}} = \sum_{w=w_{\min}}^{w_{\max}} \left(\max_{r \in L_w} T_{\text{apply}}(r) + \max_{r \in L_w} T_{\text{healthy}}(r)\right)$$

### Wave Assignment Strategy

| Wave | Resources | Rationale |
|:---:|:---|:---|
| -3 | Namespaces | Must exist first |
| -2 | CRDs | Required before CRs |
| -1 | RBAC, ServiceAccounts | Required by workloads |
| 0 | ConfigMaps, Secrets | Configuration (default wave) |
| 1 | Migrations (Jobs) | Schema must be ready |
| 2 | Deployments, StatefulSets | Application workloads |
| 3 | Services, Ingress | Networking |
| 4 | HPA, PDB | Scaling and disruption policies |

---

## 3. Health Assessment Lattice (Lattice Theory)

### Health Status Values

Health statuses form a lattice ordered by "goodness":

$$\text{Missing} \prec \text{Degraded} \prec \text{Suspended} \prec \text{Progressing} \prec \text{Healthy}$$

### Aggregate Health

Application health is the meet (infimum) of all resource healths:

$$H_{\text{app}} = \bigwedge_{r \in R} H(r)$$

This means a single Degraded resource makes the entire application Degraded:

$$\exists r : H(r) = \text{Degraded} \implies H_{\text{app}} = \text{Degraded}$$

### Resource Health Functions

Built-in health assessment for common resource types:

| Resource | Healthy When | Progressing When |
|:---|:---|:---|
| Deployment | `availableReplicas == replicas` | `updatedReplicas < replicas` |
| StatefulSet | `readyReplicas == replicas` | `currentRevision != updateRevision` |
| DaemonSet | `numberReady == desiredNumberScheduled` | Rolling update in progress |
| Job | `succeeded >= completions` | Active pods > 0 |
| Pod | Phase = Running, all containers ready | Phase = Pending |
| PVC | Phase = Bound | Phase = Pending |
| Ingress | Has at least one IP/hostname | No address assigned |

### Health Check as Boolean Function

Each health check is a predicate:

$$h_r: \text{ResourceState} \rightarrow \{\text{Healthy}, \text{Progressing}, \text{Degraded}, \text{Missing}, \text{Unknown}\}$$

Custom health checks (Lua) extend this function:

$$h_{\text{custom}}: \text{ResourceState} \rightarrow (\text{status}, \text{message})$$

---

## 4. Retry with Exponential Backoff (Probability Theory)

### Backoff Sequence

Argo CD's retry mechanism uses exponential backoff:

$$T_i = \min(D \cdot F^i, T_{\max})$$

Where:
- $D$ = initial duration (e.g., 5s)
- $F$ = backoff factor (e.g., 2)
- $T_{\max}$ = maximum duration (e.g., 3m)

For $D = 5, F = 2, T_{\max} = 180$:

| Attempt | Delay | Cumulative |
|:---:|:---:|:---:|
| 1 | 5s | 5s |
| 2 | 10s | 15s |
| 3 | 20s | 35s |
| 4 | 40s | 75s |
| 5 | 80s | 155s |
| 6+ | 180s (capped) | 335s+ |

### Expected Time to Success

If each attempt succeeds with probability $p$:

$$E[\text{attempts}] = \frac{1}{p}$$

$$E[\text{time}] = \sum_{i=0}^{E[\text{attempts}]-1} T_i$$

For $p = 0.8$ (transient failures):

$$E[\text{attempts}] = 1.25$$
$$E[\text{time}] \approx 5 + 0.2 \times 10 = 7\text{s}$$

---

## 5. ApplicationSet Generators (Combinatorics)

### Generator Cardinality

Each generator produces a set of parameter tuples:

$$G: \text{Config} \rightarrow \mathcal{P}(\text{ParamTuples})$$

| Generator | Output Size | Parameters |
|:---|:---|:---|
| Git (directories) | $\|D\|$ = directories | `path`, `path.basename` |
| Git (files) | $\|F\|$ = files | file content fields |
| Cluster | $\|C\|$ = clusters | `name`, `server`, labels |
| List | $\|L\|$ = list entries | arbitrary key-value |
| Pull Request | $\|PR\|$ = open PRs | `branch`, `number` |
| Matrix | $\|G_1\| \times \|G_2\|$ | cross product |
| Merge | $\|G_1 \bowtie G_2\|$ | inner join |

### Matrix Generator

Produces the Cartesian product of two generators:

$$G_{\text{matrix}} = G_1 \times G_2 = \{(p_1, p_2) : p_1 \in G_1, p_2 \in G_2\}$$

For 3 clusters and 5 apps:

$$|G_{\text{matrix}}| = 3 \times 5 = 15 \text{ Applications}$$

### Merge Generator

Produces the inner join of generators on shared keys:

$$G_{\text{merge}} = G_1 \bowtie_{\text{key}} G_2 = \{p_1 \cup p_2 : p_1 \in G_1, p_2 \in G_2, p_1[\text{key}] = p_2[\text{key}]\}$$

---

## 6. RBAC Policy as Access Matrix (Access Control Theory)

### Policy Model

Argo CD RBAC policies define an access control matrix:

$$A: \text{Subject} \times \text{Resource} \times \text{Action} \rightarrow \{\text{allow}, \text{deny}\}$$

Policy rules:

$$p(\text{subject}, \text{resource}, \text{action}, \text{object}) = \text{effect}$$

### Permission Resolution

For a subject $s$ with roles $\{r_1, r_2, \ldots\}$:

$$A(s, \text{res}, \text{act}) = \bigvee_{r \in \text{roles}(s)} A(r, \text{res}, \text{act})$$

Deny takes precedence (deny-override):

$$\text{denied}(s) \implies A(s) = \text{deny}$$

### Project Scope Restriction

AppProject constrains the access space:

$$\text{valid}(\text{app}) = (\text{app.source} \in P.\text{sourceRepos}) \wedge (\text{app.dest} \in P.\text{destinations})$$

The project defines a security boundary:

$$\mathcal{A}_{\text{project}} = \mathcal{A}_{\text{global}} \cap \mathcal{A}_{\text{project\_constraints}}$$

---

## 7. Diff Algorithm Complexity (Algorithm Theory)

### Three-Way Diff

Argo CD performs a three-way diff:

$$\Delta_3(\text{desired}, \text{live}, \text{last-applied})$$

This distinguishes:
- Fields added by the user (in desired, not in last-applied)
- Fields added by controllers (in live, not in last-applied)
- Fields explicitly removed (in last-applied, not in desired)

### Diff Complexity

For resources with $n$ fields at depth $d$:

$$T_{\text{diff}} = O(n \times d) \quad \text{per resource}$$

For an application with $R$ resources:

$$T_{\text{total\_diff}} = O(R \times n \times d)$$

### Normalized Diff

Argo CD normalizes resources before diffing to ignore:
- Default values injected by admission controllers
- Managed fields (server-side apply metadata)
- Annotation ordering

The normalization function:

$$N(r) = \text{sort}(\text{strip\_defaults}(\text{strip\_managed}(r)))$$

$$\Delta = N(r_{\text{desired}}) \ominus N(r_{\text{live}})$$

---

## Prerequisites

control-theory, graph-theory, lattice-theory, probability, combinatorics, access-control

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| Drift detection (per app) | $O(R \times n \times d)$ — resources x fields x depth | $O(R)$ |
| Sync (sequential waves) | $O(W \times R_w)$ — waves x resources per wave | $O(R)$ |
| Health aggregation | $O(R)$ — one check per resource | $O(1)$ |
| ApplicationSet generation | $O(\|G_1\| \times \|G_2\|)$ — matrix | $O(\|G_1\| \times \|G_2\|)$ |
| RBAC evaluation | $O(P)$ — P = policy rules | $O(1)$ |
| Git manifest render | $O(S)$ — S = repo size | $O(M)$ — M = manifest count |
