# The Mathematics of Flux — Reconciliation Loops and Convergence Theory

> *Flux operates as a continuous reconciliation system where desired state is declared in Git and actual state is observed in the cluster. The mathematical core is a convergence loop modeled by control theory, with dependency resolution as a DAG scheduling problem and drift detection as a set-difference operation.*

---

## 1. Reconciliation as Control Theory (Feedback Systems)

### The Problem

Flux continuously drives cluster state toward the declared desired state. This is a discrete-time feedback control loop where the error signal is the difference between desired and actual states.

### The Formula

The reconciliation error at time step $k$:

$$e_k = S_{\text{desired}} - S_{\text{actual}}(k)$$

The controller applies corrections:

$$S_{\text{actual}}(k+1) = S_{\text{actual}}(k) + \alpha \cdot e_k$$

where $\alpha \in (0, 1]$ is the convergence rate (successful apply fraction).

Convergence condition after $n$ reconciliation cycles:

$$\|e_n\| = (1 - \alpha)^n \cdot \|e_0\| \to 0 \text{ as } n \to \infty$$

### Worked Examples

With $\alpha = 0.9$ (90% of resources reconcile each cycle):

| Cycle $n$ | Remaining Error $\|e_n\| / \|e_0\|$ |
|:---:|:---:|
| 1 | 0.100 |
| 2 | 0.010 |
| 3 | 0.001 |
| 5 | 0.00001 |

Time to converge within $\epsilon$ of desired state:

$$n_{\epsilon} = \left\lceil \frac{\ln \epsilon}{\ln(1 - \alpha)} \right\rceil$$

For $\alpha = 0.9$, $\epsilon = 0.01$: $n = \lceil \ln(0.01) / \ln(0.1) \rceil = 2$ cycles.

---

## 2. Dependency DAG Scheduling (Graph Theory)

### The Problem

Flux Kustomizations have `dependsOn` fields forming a directed acyclic graph. The reconciliation order must respect topological ordering.

### The Formula

Given dependency graph $G = (V, E)$ where $V$ is the set of Kustomizations:

$$\text{reconcileOrder} = \text{toposort}(G)$$

The critical path determines minimum reconciliation time:

$$T_{\min} = \max_{P \in \text{paths}(G)} \sum_{v \in P} (I_v + R_v)$$

where $I_v$ is the reconciliation interval and $R_v$ is the apply duration for Kustomization $v$.

### Worked Examples

```
infrastructure (interval=5m, apply=30s)
  --> cert-manager (interval=5m, apply=20s)
    --> apps (interval=10m, apply=45s)
  --> monitoring (interval=10m, apply=60s)
```

| Path | Total Time |
|:---|:---:|
| infra -> cert-manager -> apps | 5:30 + 5:20 + 10:45 = 21m35s |
| infra -> monitoring | 5:30 + 10:60 = 16m30s |

Critical path: infra -> cert-manager -> apps at 21m35s.

---

## 3. Drift Detection (Set Theory)

### The Problem

Flux detects drift by comparing the set of desired resources with actual cluster resources. Pruning removes orphaned resources.

### The Formula

Let $D$ = desired resource set, $A$ = actual resource set.

Resources to create:

$$C = D \setminus A$$

Resources to prune (when `prune: true`):

$$P = A \setminus D$$

Resources to update (where content differs):

$$U = \{r \in D \cap A \mid \text{hash}(r_D) \neq \text{hash}(r_A)\}$$

Total reconciliation actions:

$$|W| = |C| + |P| + |U|$$

### Worked Examples

Given $|D| = 50$ resources, $|A| = 48$ resources, $|D \cap A| = 45$:

- Create: $|C| = 50 - 45 = 5$ new resources
- Prune: $|P| = 48 - 45 = 3$ orphaned resources
- Update: if 8 of the 45 common resources differ, $|U| = 8$
- Total actions: $|W| = 5 + 3 + 8 = 16$

---

## 4. Image Policy Semver Matching (Order Theory)

### The Problem

ImagePolicy selects the latest container image tag matching a semver range. This is a filtering and ordering problem over a partially ordered set.

### The Formula

Given image tags $T = \{t_1, t_2, \ldots, t_m\}$ and semver range constraint $R$:

$$T_{\text{valid}} = \{t \in T \mid t \models R\}$$

Selected tag:

$$t^* = \max(T_{\text{valid}}, \leq_{\text{semver}})$$

where $\leq_{\text{semver}}$ is the semver precedence ordering:

$$\text{major} \cdot 10^6 + \text{minor} \cdot 10^3 + \text{patch}$$

### Worked Examples

Tags: `[5.0.0, 5.1.2, 5.2.0, 6.0.0-rc1, 6.0.0]`, Range: `>=5.0.0 <6.0.0`:

$$T_{\text{valid}} = \{5.0.0, 5.1.2, 5.2.0\}$$

$$t^* = 5.2.0 \quad (5002000 > 5001002 > 5000000)$$

---

## 5. Notification Fan-Out (Event Routing)

### The Problem

The notification controller routes events from sources to providers. Each Alert defines a filter over event sources and severities.

### The Formula

For event $e$ with source $s_e$ and severity $\sigma_e$, and Alert $a$ with source patterns $S_a$ and severity threshold $\Sigma_a$:

$$\text{notify}(e, a) = \begin{cases} 1 & \text{if } s_e \in S_a \wedge \sigma_e \geq \Sigma_a \\ 0 & \text{otherwise} \end{cases}$$

Total notifications for event $e$ across all alerts $A$:

$$N(e) = \sum_{a \in A} \text{notify}(e, a)$$

### Worked Examples

3 alerts: error-only-slack (severity >= error, all sources), info-pagerduty (severity >= info, HelmRelease only), error-teams (severity >= error, Kustomization only).

Event: HelmRelease error:

$$N = 1 + 1 + 0 = 2 \text{ notifications (Slack + PagerDuty)}$$

Event: Kustomization info:

$$N = 0 + 0 + 0 = 0 \text{ notifications}$$

---

## 6. Multi-Tenancy Resource Isolation (Access Control)

### The Problem

Multi-tenancy requires namespace isolation with RBAC. Each tenant's ServiceAccount constrains the set of permissible operations.

### The Formula

Tenant $t$ has permission set $P_t \subseteq \mathcal{R} \times \mathcal{V} \times \mathcal{N}$ where $\mathcal{R}$ = resource types, $\mathcal{V}$ = verbs, $\mathcal{N}$ = namespaces.

Isolation property:

$$\forall t_1, t_2 \in T, \; t_1 \neq t_2 \implies P_{t_1} \cap P_{t_2} = \emptyset \text{ (on namespace dimension)}$$

The total cluster permission surface:

$$|P_{\text{total}}| = \sum_{t \in T} |P_t| + |P_{\text{platform}}|$$

### Worked Examples

3 tenants, each with 1 namespace, 5 resource types, 4 verbs:

$$|P_t| = 5 \times 4 \times 1 = 20 \text{ permissions per tenant}$$

$$|P_{\text{total}}| = 3 \times 20 + |P_{\text{platform}}|$$

---

## Prerequisites

- control-theory, graph-theory, set-theory, semver, kubernetes, gitops, helm
