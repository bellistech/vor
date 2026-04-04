# The Mathematics of Azure CLI — Cloud API Internals

> *The Azure CLI interacts with Azure Resource Manager (ARM) — a declarative API layer that manages resources as a graph. Its internals involve OAuth2 token lifecycle, ARM template evaluation, RBAC inheritance, and consumption-based pricing models.*

---

## 1. Authentication (OAuth2 Token Lifecycle)

### The Problem

Azure CLI authenticates via OAuth2, obtaining tokens with finite lifetimes. Understanding the token math prevents unexpected auth failures.

### Token Lifetime

$$T_{valid} = T_{issued} + \text{TTL}$$

| Token Type | Default TTL | Refresh |
|:---|:---:|:---|
| Access token | 60-90 min | Refreshable |
| Refresh token | 90 days | Rolling window |
| Service principal | 60 min | Client credentials grant |

### Token Refresh Timeline

$$\text{refresh\_at} = T_{issued} + \text{TTL} \times 0.75$$

The CLI proactively refreshes at 75% of TTL — for a 60-min token, at minute 45.

### Session Continuity

$$T_{session\_max} = T_{refresh\_token\_TTL} = 90 \text{ days}$$

After 90 days of inactivity, re-authentication is required.

### Concurrent Token Requests

$$\text{Tokens in flight} = \frac{N_{subscriptions} \times N_{tenants}}{1} \times N_{resources}$$

Each subscription-tenant pair requires its own token scope.

---

## 2. ARM Resource Model (Graph Structure)

### The Problem

Azure resources form a hierarchy: Management Group > Subscription > Resource Group > Resource. Understanding this graph affects scoping and permissions.

### Resource Hierarchy

$$\text{Scope} = /\text{providers}/\text{Microsoft.Compute}/\text{virtualMachines}/\text{myvm}$$

### Resource Group as Container

$$R_{group} = \{r_1, r_2, \ldots, r_n\}$$

Deleting a resource group deletes all contained resources:

$$\text{delete}(RG) \implies \forall r \in RG: \text{delete}(r)$$

### Resource Dependencies

$$G = (R, D) \quad \text{where } D = \text{dependency edges}$$

ARM resolves the DAG and creates resources in dependency order with parallelism:

$$T_{deploy} = \sum_{l=0}^{L} \max_{r \in \text{level}(l)} T_{api}(r)$$

### Deployment Modes

**Incremental** (default):

$$R_{final} = R_{existing} \cup R_{template}$$

Existing resources not in template are **preserved**.

**Complete**:

$$R_{final} = R_{template}$$

Existing resources not in template are **deleted**. This is the dangerous mode:

$$\text{Deleted} = R_{existing} \setminus R_{template}$$

---

## 3. RBAC (Role-Based Access Control)

### The Problem

Azure RBAC assigns permissions through role assignments at various scopes. Permissions inherit downward.

### Effective Permissions

$$\text{Effective}(user, scope) = \bigcup_{s \in \text{ancestors}(scope)} \text{Assigned}(user, s)$$

Permissions assigned at a parent scope flow to all children:

$$\text{Management Group} \rightarrow \text{Subscription} \rightarrow \text{Resource Group} \rightarrow \text{Resource}$$

### Deny Assignments

$$\text{Can do X} = (X \in \text{Effective Allow}) \wedge (X \notin \text{Effective Deny})$$

### Role Definition Math

$$\text{Role} = \text{Actions} \cup \text{DataActions} - \text{NotActions} - \text{NotDataActions}$$

### RBAC Limits

| Limit | Value |
|:---|:---:|
| Role assignments per subscription | 2,000 |
| Custom roles per tenant | 5,000 |
| Role definition size | 128 KB |

### Assignment Count Growth

$$\text{Assignments} = |U| \times |R| \times |S|$$

Where $U$ = users, $R$ = roles, $S$ = scopes (worst case Cartesian product). In practice, much less due to group-based assignment:

$$\text{Assignments}_{groups} = |G| \times |R| \times |S| \quad \text{where } |G| \ll |U|$$

---

## 4. Pricing Models (Consumption Math)

### VM Pricing

$$C_{VM} = T_{hours} \times R_{hourly}$$

### Reserved Instance Savings

$$\text{Savings} = 1 - \frac{R_{reserved}}{R_{paygo}}$$

| Term | Typical Savings |
|:---|:---:|
| 1-year RI | 30-40% |
| 3-year RI | 55-65% |
| Spot (low priority) | 60-90% |

### Storage Account Pricing (Tiered)

$$C_{storage} = S_{hot} \times R_{hot} + S_{cool} \times R_{cool} + S_{archive} \times R_{archive}$$

| Tier | Storage $/GB/mo | Read $/10k | Write $/10k |
|:---|:---:|:---:|:---:|
| Hot | $0.018 | $0.004 | $0.05 |
| Cool | $0.01 | $0.01 | $0.10 |
| Archive | $0.002 | $5.00 | $0.10 |

### Access Pattern Break-Even

Hot vs Cool: cheaper to use Cool if read frequency is low:

$$S \times R_{hot} + N_{read} \times R_{hot\_read} > S \times R_{cool} + N_{read} \times R_{cool\_read}$$

$$N_{read} < \frac{S \times (R_{hot} - R_{cool})}{R_{cool\_read} - R_{hot\_read}}$$

For 100 GB: $N_{read} < \frac{100 \times 0.008}{0.006} = 133$ reads per 10k — below ~1.3M reads/month, Cool wins.

---

## 5. JMESPath Query Language (CLI Output Filtering)

### The Problem

Azure CLI uses JMESPath for `--query` filtering. Understanding its complexity helps optimize large result sets.

### Query Complexity

| Operation | Complexity | Example |
|:---|:---:|:---|
| Field access | $O(1)$ | `[].name` |
| Filter | $O(N)$ | `[?location=='eastus']` |
| Sort | $O(N \log N)$ | `sort_by(@, &name)` |
| Flatten | $O(N \times M)$ | `[].subnets[]` |

### Client-Side vs Server-Side Filtering

$$T_{server\_filter} = T_{api}(filtered) \ll T_{client\_filter} = T_{api}(all) + T_{jmespath}$$

Always prefer `--query` over piping to `jq` when the API supports server-side filtering.

---

## 6. Rate Limits and Throttling

### The Problem

ARM has per-subscription, per-region rate limits.

### Rate Limit Headers

$$\text{x-ms-ratelimit-remaining-subscription-reads}: R_{remaining}$$

| Operation Type | Limit (per 5 min) |
|:---|:---:|
| Reads | 12,000 |
| Writes | 1,200 |
| Deletes | 1,200 |

### Sustained Rate

$$\text{Reads/second} = \frac{12{,}000}{300} = 40 \text{ req/s}$$
$$\text{Writes/second} = \frac{1{,}200}{300} = 4 \text{ req/s}$$

### Large Deployment Timing

For deploying 500 resources:

$$T_{min} = \frac{500}{4} = 125\text{s} \approx 2 \text{ min}$$

With parallelism and dependencies, actual time is typically 3-10 minutes.

---

## 7. Availability Zones (Fault Domain Math)

### The Problem

Azure distributes resources across fault domains and availability zones for resilience.

### Availability Calculation

Single VM SLA: 99.9% (with premium storage).

$$A_{single} = 0.999$$

Availability Set (2 fault domains):

$$A_{set} = 1 - (1 - A_{single})^2 = 1 - 0.001^2 = 0.999999$$

Availability Zones (3 zones):

$$A_{zones} = 1 - (1 - A_{zone})^3$$

With zone SLA of 99.99%:

$$A_{zones} = 1 - (0.0001)^3 = 1 - 10^{-12} \approx 99.9999999999\%$$

### Practical SLA Table

| Configuration | SLA | Downtime/Year |
|:---|:---:|:---:|
| Single VM | 99.9% | 8.76 hours |
| Availability Set | 99.95% | 4.38 hours |
| Availability Zones | 99.99% | 52.6 minutes |
| Cross-Region + Traffic Manager | 99.999% | 5.26 minutes |

---

## 8. Summary of Functions by Type

| Formula | Math Type | Domain |
|:---|:---|:---|
| $T_{issued} + \text{TTL}$ | Linear | Token lifecycle |
| $R_{existing} \cup R_{template}$ / $\setminus$ | Set operations | Deployment modes |
| $\bigcup \text{Assigned}(u, s)$ | Set union + inheritance | RBAC |
| Tiered pricing | Piecewise linear | Cost estimation |
| $1 - (1-A)^N$ | Probability | Availability |
| $12{,}000 / 300$ | Rate | Throttling |

---

*Azure CLI commands ultimately become ARM API calls — each subject to OAuth2 authentication, RBAC evaluation, rate limiting, and consumption billing. The hierarchical resource model and inheritance rules determine who can do what, where, and at what cost.*

## Prerequisites

- Azure subscription and resource group concepts
- OAuth2 and service principal authentication
- ARM (Azure Resource Manager) model
- JSON and JMESPath query syntax

## Complexity

- Beginner: login, VM creation, storage accounts, resource group management
- Intermediate: RBAC role assignments, AKS clusters, Key Vault, networking
- Advanced: ARM template evaluation, OAuth2 token lifecycle, throttling strategies, consumption billing optimization
