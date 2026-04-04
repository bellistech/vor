# The Mathematics of Gateway API — Traffic Routing as Graph Theory

> *Every packet is a traveler. Every route is a decision. The gateway is the crossroads where topology meets intent.*

---

## 1. Route Matching Priority (Lexicographic Ordering)

### The Problem

When multiple HTTPRoutes match an incoming request, the Gateway must select the most specific one. The Gateway API specification defines a precise priority ordering based on hostname specificity, path length, header count, and method matching. How do we formalize this as a total order?

### The Formula

Define a match specificity vector for each rule $r$ as a tuple:

$$\vec{s}(r) = (h(r), p(r), n_h(r), n_q(r))$$

where:
- $h(r) \in \{2, 1, 0\}$: hostname specificity (exact=2, wildcard=1, none=0)
- $p(r) \in \mathbb{N}$: path length (longer = more specific)
- $n_h(r) \in \mathbb{N}$: number of header matches
- $n_q(r) \in \mathbb{N}$: number of query parameter matches

The priority ordering is lexicographic:

$$r_1 \succ r_2 \iff \vec{s}(r_1) >_{\text{lex}} \vec{s}(r_2)$$

For two rules with equal specificity vectors, the tie-breaking falls to:

$$\text{tiebreak}(r_1, r_2) = \text{oldest}(\text{creationTimestamp})$$

### Worked Examples

**Example 1**: Three routes for `api.example.com`:
- $r_1$: hostname=`api.example.com`, path=`/api/v1/users`, 0 headers
- $r_2$: hostname=`*.example.com`, path=`/api/v1/users`, 1 header
- $r_3$: hostname=`api.example.com`, path=`/api`, 0 headers

Specificity vectors:
$$\vec{s}(r_1) = (2, 15, 0, 0)$$
$$\vec{s}(r_2) = (1, 15, 1, 0)$$
$$\vec{s}(r_3) = (2, 4, 0, 0)$$

Ordering: $r_1 \succ r_3 \succ r_2$ (exact hostname beats wildcard, then longer path wins).

**Example 2**: Two routes with identical hostnames and paths:
- $r_1$: 2 header matches, created 10:00
- $r_2$: 1 header match + 1 query match, created 09:00

$$\vec{s}(r_1) = (2, 10, 2, 0), \quad \vec{s}(r_2) = (2, 10, 1, 1)$$

$r_1 \succ r_2$ because header count (component 3) breaks the tie.

---

## 2. Traffic Splitting (Weighted Random Selection)

### The Problem

HTTPRoute supports weighted backend references for canary deployments and traffic shifting. Each request is independently assigned to a backend with probability proportional to its weight. What is the variance in actual traffic distribution?

### The Formula

For $k$ backends with weights $w_1, w_2, \ldots, w_k$, the probability of routing to backend $i$:

$$p_i = \frac{w_i}{\sum_{j=1}^{k} w_j}$$

Over $N$ requests, the number of requests to backend $i$ follows a Binomial distribution:

$$X_i \sim \text{Binomial}(N, p_i)$$

$$E[X_i] = Np_i, \quad \text{Var}(X_i) = Np_i(1-p_i)$$

The coefficient of variation (relative error):

$$\text{CV}_i = \frac{\sqrt{Np_i(1-p_i)}}{Np_i} = \sqrt{\frac{1-p_i}{Np_i}}$$

### Worked Examples

**Example 1**: A 90/10 canary split over 1000 requests:

$$p_{\text{stable}} = 0.9, \quad p_{\text{canary}} = 0.1$$

$$E[X_{\text{canary}}] = 100, \quad \text{Var}(X_{\text{canary}}) = 90$$

$$\text{CV}_{\text{canary}} = \sqrt{\frac{0.9}{100}} = 0.095$$

95% confidence interval for canary traffic: $100 \pm 1.96\sqrt{90} = 100 \pm 18.6$, so between 81 and 119 requests (8.1% to 11.9%).

**Example 2**: At only 100 total requests with a 5/95 split:

$$E[X_{\text{canary}}] = 5, \quad \text{Var} = 4.75$$

$$\text{CV} = \sqrt{\frac{0.95}{5}} = 0.436$$

43.6% relative error. The canary might receive 0-10 requests. Meaningful analysis requires at least:

$$N \geq \frac{z^2(1-p)}{p \cdot \epsilon^2} = \frac{1.96^2 \times 0.95}{0.05 \times 0.01} = 7299 \text{ requests}$$

for 10% relative precision at 95% confidence.

---

## 3. Listener Multiplexing (Hostname Partitioning)

### The Problem

A single Gateway can have multiple listeners on the same port, differentiated by hostname via SNI (Server Name Indication). This partitions the request space. How do we reason about conflicts and coverage?

### The Formula

Define the hostname space $H$ as the set of all possible hostnames. Each listener $l_i$ covers a subset $H_i \subseteq H$:

- Exact hostname `api.example.com`: $H_i = \{\text{api.example.com}\}$
- Wildcard `*.example.com`: $H_i = \{x.\text{example.com} : x \in \Sigma^+\}$

Two listeners conflict if their hostname sets overlap:

$$\text{conflict}(l_i, l_j) \iff H_i \cap H_j \neq \emptyset \land i \neq j$$

The total coverage of a Gateway:

$$H_{\text{covered}} = \bigcup_{i=1}^{n} H_i$$

Requests to uncovered hostnames: $H \setminus H_{\text{covered}}$ receive no match (404 or connection refused).

### Worked Examples

**Example 1**: Three listeners on port 443:
- $l_1$: `api.example.com` (exact)
- $l_2$: `*.example.com` (wildcard)
- $l_3$: `app.example.com` (exact)

$H_1 \cap H_2 = \{\text{api.example.com}\}$ -- conflict exists but is resolved by specificity (exact wins over wildcard). Effective routing:
- `api.example.com` -> $l_1$ (exact match wins)
- `app.example.com` -> $l_3$ (exact match wins)
- `anything-else.example.com` -> $l_2$ (wildcard catches rest)

**Example 2**: Two wildcards on the same port:
- $l_1$: `*.example.com`
- $l_2$: `*.example.com`

$H_1 = H_2$ -- complete overlap. This is a conflict. The Gateway API spec resolves by listener order (first listed wins) and sets a `Conflicted` condition on the second listener.

---

## 4. Cross-Namespace Security (Reference Grant as Capability Model)

### The Problem

ReferenceGrant controls which namespaces can reference resources in other namespaces. This is a capability-based access control model. How do we analyze the security properties of a ReferenceGrant topology?

### The Formula

Model the cluster as a directed graph $G = (N, E)$ where nodes are namespaces and edges represent granted references. A ReferenceGrant in namespace $B$ allowing references from namespace $A$ creates edge $(A, B)$:

$$E = \{(A, B) : \exists \text{ReferenceGrant in } B \text{ allowing from } A\}$$

The blast radius of a compromised namespace $A$ is the set of namespaces reachable via granted references:

$$\text{BlastRadius}(A) = \{B : (A, B) \in E\}$$

The maximum blast radius across all namespaces:

$$R_{\max} = \max_{A \in N} |\text{BlastRadius}(A)|$$

A secure topology minimizes $R_{\max}$ while maintaining necessary connectivity.

### Worked Examples

**Example 1**: A cluster with 4 namespaces: `apps`, `gateway`, `database`, `monitoring`.

ReferenceGrants:
- `gateway` grants from `apps` (routes reference gateway)
- `database` grants from `apps` (backend services)
- `monitoring` grants from all namespaces

$$\text{BlastRadius}(\text{apps}) = \{\text{gateway}, \text{database}, \text{monitoring}\} \rightarrow 3$$
$$\text{BlastRadius}(\text{gateway}) = \{\text{monitoring}\} \rightarrow 1$$
$$R_{\max} = 3$$

**Example 2**: Without ReferenceGrants (all cross-namespace denied):

$$R_{\max} = 0$$

Every namespace is isolated. Routes can only reference Services in their own namespace. This is the default posture -- ReferenceGrants explicitly open holes.

---

## 5. Gateway Scaling (Connection Capacity Planning)

### The Problem

A Gateway proxies all traffic entering the cluster. Connection limits, memory per connection, and CPU for TLS termination constrain the maximum throughput. How do we size the Gateway infrastructure?

### The Formula

Each active connection consumes memory $m_c$ and CPU for TLS handshake $c_h$. For $C$ concurrent connections with TLS termination:

$$\text{Memory} = C \times m_c + M_{\text{base}}$$

$$\text{CPU}_{\text{handshake}} = R_{\text{new}} \times c_h$$

where $R_{\text{new}}$ is the rate of new connections per second. For HTTP/2 with connection reuse, the effective connection count:

$$C_{\text{eff}} = \frac{C_{\text{clients}}}{k}$$

where $k$ is the multiplexing factor (streams per connection).

The maximum requests per second:

$$\text{RPS}_{\max} = \min\left(\frac{\text{CPU}_{\text{total}} - R_{\text{new}} \times c_h}{c_r}, \frac{\text{Memory}_{\text{total}} - M_{\text{base}}}{m_c}\right)$$

where $c_r$ is CPU per request for routing/proxying.

### Worked Examples

**Example 1**: An Envoy-based Gateway with 4 CPU cores and 8GB RAM. Per-connection memory: 32KB, TLS handshake: 1ms CPU, per-request routing: 0.1ms CPU. Base memory: 512MB. Connection rate: 1000 new/s.

$$\text{CPU available for routing} = 4000\text{ms/s} - 1000 \times 1\text{ms} = 3000\text{ms/s}$$

$$\text{RPS}_{\max}^{\text{CPU}} = \frac{3000}{0.1} = 30000 \text{ req/s}$$

$$C_{\max}^{\text{memory}} = \frac{(8192 - 512) \times 1024}{32} = 245760 \text{ connections}$$

CPU is the bottleneck at 30,000 RPS.

**Example 2**: With HTTP/2 and multiplexing factor $k = 100$, 10,000 clients create only:

$$C_{\text{eff}} = \frac{10000}{100} = 100 \text{ connections}$$

$$\text{Memory} = 100 \times 32\text{KB} + 512\text{MB} \approx 515\text{MB}$$

HTTP/2 multiplexing dramatically reduces connection overhead, shifting the bottleneck to per-request CPU.

---

## Prerequisites

- HTTP/1.1 and HTTP/2 protocol mechanics (connection management, multiplexing, TLS)
- Kubernetes Service networking (ClusterIP, endpoints, DNS resolution)
- TLS and SNI (Server Name Indication for hostname-based routing)
- Kubernetes RBAC and namespace isolation model
- Probability distributions (Binomial for traffic splitting analysis)
- Graph theory (directed graphs, reachability, partitioning)
- Capacity planning (Little's Law, resource utilization modeling)
