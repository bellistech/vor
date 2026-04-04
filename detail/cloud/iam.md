# The Mathematics of IAM — Policy Evaluation and Access Control Theory

> *Identity and Access Management is fundamentally a problem of set theory and boolean logic: every authorization decision reduces to computing the intersection of requested actions with granted permissions, minus explicit denials, evaluated against attribute predicates.*

---

## 1. Policy Evaluation Logic (Boolean Algebra)

The core IAM decision function is a three-valued logic:

$$D(r) = \begin{cases}
\text{DENY} & \text{if } r \in \bigcup_{p} \text{Deny}(p) \\
\text{ALLOW} & \text{if } r \in \bigcup_{p} \text{Allow}(p) \setminus \bigcup_{p} \text{Deny}(p) \\
\text{IMPLICIT\_DENY} & \text{otherwise}
\end{cases}$$

Where $r$ is the request tuple $(principal, action, resource, conditions)$.

### AWS Evaluation Order

The evaluation chain processes policies in layers:

$$D_{final} = \text{SCP} \cap \text{PermBoundary} \cap (\text{Identity} \cup \text{Resource}) \setminus \text{ExplicitDeny}$$

| Layer | Type | Precedence |
|:---|:---|:---:|
| Service Control Policy | Organization | 1 (highest) |
| Permissions Boundary | Account | 2 |
| Session Policy | Session | 3 |
| Identity Policy | Principal | 4 |
| Resource Policy | Resource | 4 |
| Explicit Deny | Any | Override |

### Effective Permissions Calculation

For a principal $P$ with $n$ attached policies:

$$\text{Effective}(P) = \left(\bigcup_{i=1}^{n} A_i\right) \cap B \cap S \setminus \left(\bigcup_{i=1}^{n} D_i\right)$$

Where $A_i$ = allow set of policy $i$, $B$ = permissions boundary, $S$ = SCP allows.

---

## 2. RBAC Formal Model (Lattice Theory)

### Core RBAC Components

RBAC is defined by the NIST model as a tuple:

$$\text{RBAC} = (U, R, P, S, UA, PA, \text{user}, \text{roles})$$

- $U$ = set of users, $R$ = set of roles, $P$ = set of permissions
- $S$ = set of sessions
- $UA \subseteq U \times R$ (user-role assignment)
- $PA \subseteq P \times R$ (permission-role assignment)

### Role Hierarchy

Roles form a partial order (lattice):

$$r_1 \succeq r_2 \implies \text{permissions}(r_1) \supseteq \text{permissions}(r_2)$$

The effective permissions through hierarchy:

$$\text{auth}(r) = \bigcup_{r' \preceq r} \text{permissions}(r')$$

### Counting Complexity

| Metric | Formula | Example |
|:---|:---|:---:|
| User-role mappings | $|U| \times |R|$ | 1000 users, 50 roles = 50,000 max |
| Permission-role mappings | $|P| \times |R|$ | 5000 perms, 50 roles = 250,000 max |
| Possible role combinations | $2^{|R|}$ | $2^{50} \approx 10^{15}$ |
| Policy evaluation checks | $O(|P_{attached}| \times |S_{statements}|)$ | ~50 policies x 10 stmts |

---

## 3. ABAC Predicate Logic (First-Order Logic)

ABAC extends RBAC with attribute-based conditions expressed as predicates:

$$\text{Allow}(r) \iff \phi_1(r) \wedge \phi_2(r) \wedge \cdots \wedge \phi_k(r)$$

Where each $\phi_i$ is an attribute predicate:

$$\phi_{dept}(r) = (\text{principal.tag.dept} = \text{resource.tag.dept})$$
$$\phi_{time}(r) = (t_{current} \in [t_{start}, t_{end}])$$
$$\phi_{ip}(r) = (\text{source\_ip} \in \text{CIDR}_{allowed})$$

### Condition Evaluation

The condition block is a conjunction of condition operators:

$$C = \bigwedge_{op \in \text{operators}} \bigwedge_{key \in \text{keys}} op(key, values)$$

| Operator | Logic | Example |
|:---|:---|:---|
| StringEquals | $a = b$ | Tag match |
| StringLike | $a \sim b$ | Wildcard pattern |
| IpAddress | $ip \in CIDR$ | Source IP check |
| DateLessThan | $t < t_{max}$ | Expiry check |
| NumericLessThan | $n < n_{max}$ | Count limit |
| Bool | $b = true$ | MFA present |

### ABAC Scalability Advantage

| Approach | New user onboarding | New resource protection |
|:---|:---|:---|
| RBAC | Add to $k$ groups/roles | Update $m$ policies |
| ABAC | Set $j$ tags (O(1) policies) | Set $j$ tags (O(1) policies) |

$$\text{RBAC policy count} = O(|R| \times |P|)$$
$$\text{ABAC policy count} = O(|Attribute\_patterns|) \ll O(|R| \times |P|)$$

---

## 4. Least Privilege Quantification (Information Theory)

### Excess Permission Metric

$$E(P) = \frac{|\text{Granted}(P)| - |\text{Used}(P)|}{|\text{Granted}(P)|}$$

Where $E = 0$ is perfect least privilege and $E = 1$ means no permissions are used.

### Permission Entropy

$$H(P) = -\sum_{a \in \text{actions}} p(a) \log_2 p(a)$$

Where $p(a)$ = frequency of action $a$ relative to all actions used. High entropy = diverse usage. Low entropy = specialized role.

### Right-Sizing Formula

Given CloudTrail data over window $W$:

$$\text{Recommended}(P, W) = \{a \mid \text{count}(a, W) > 0\} \cup \text{Dependencies}(a)$$

| Window | Risk | Coverage |
|:---|:---|:---:|
| 7 days | Misses infrequent ops | ~60% |
| 30 days | Good baseline | ~85% |
| 90 days | Covers quarterly ops | ~95% |
| 365 days | Nearly complete | ~99% |

---

## 5. Cross-Account Trust (Graph Theory)

### Trust Graph

Cross-account access forms a directed graph $G = (V, E)$:

- $V$ = set of accounts/projects
- $E$ = trust relationships (role assumptions)

$$\text{Reachable}(v) = \{u \in V \mid \exists \text{ path } v \rightsquigarrow u \text{ in } G\}$$

### Transitive Trust Risk

$$\text{Blast radius}(v) = |\text{Reachable}(v)| \times \text{avg}(|\text{permissions}(e)|)$$

For $n$ accounts with full mesh trust:

$$|E_{max}| = n(n-1)$$

### Trust Chain Depth

$$\text{Max hops} = \text{longest path in } G$$

AWS limits role chaining to 1 hop (no transitive assumption), so:

$$\text{depth}_{AWS} \leq 1$$

GCP allows impersonation chains (configurable).

---

## 6. Token and Session Mathematics (Cryptography)

### Temporary Credential Lifetime

$$T_{valid} = t_{issue} + \text{duration} - t_{current}$$

$$\text{Valid} \iff T_{valid} > 0 \wedge \text{duration} \in [900, 43200] \text{ seconds}$$

### Session Token Entropy

AWS session tokens use Base64-encoded random bytes:

$$\text{Entropy} = L_{bytes} \times 8 \text{ bits}$$

For a 128-byte token: $\text{Entropy} = 1024$ bits.

Brute force time at $10^{12}$ attempts/sec:

$$T_{brute} = \frac{2^{1024}}{10^{12}} \approx 10^{296} \text{ seconds}$$

### Credential Rotation Schedule

$$\text{Risk}(t) = 1 - e^{-\lambda t}$$

Where $\lambda$ = compromise rate. For 90-day rotation:

$$\text{Risk}(90) = 1 - e^{-\lambda \times 90}$$

---

## 7. Policy Size and Evaluation Performance

### Policy Parsing Complexity

$$T_{eval} = O(|P| \times |S| \times |C|)$$

Where $|P|$ = number of policies, $|S|$ = statements per policy, $|C|$ = conditions per statement.

### AWS Policy Limits

| Limit | Value | Impact |
|:---|:---:|:---|
| Managed policy size | 6,144 chars | ~30 statements |
| Inline policy size | 2,048 chars | ~10 statements |
| Policies per principal | 10 managed | 10 eval passes |
| Trust policy size | 2,048 chars | Role assumption |
| Condition keys per statement | ~20 | AND conjunction |
| Evaluation budget | ~500ms | API latency impact |

### Total Evaluation Cost

$$T_{total} = \sum_{layer} \sum_{policy} \sum_{stmt} T_{match}(stmt, request)$$

Typical: 10 policies x 5 statements x 3 conditions = 150 predicate evaluations per request.

---

*Every API call in the cloud passes through an IAM evaluation engine that computes set intersections, boolean predicates, and attribute matches in under 500ms. Understanding the algebra of Allow, Deny, and Condition lets you write policies that are both secure and performant.*

## Prerequisites

- Set theory (unions, intersections, complements)
- Boolean algebra and predicate logic
- Graph theory basics (directed graphs, reachability)
- Lattice theory for role hierarchies

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| Policy evaluation (single) | $O(S \times C)$ | $O(1)$ |
| Full authorization check | $O(P \times S \times C)$ | $O(P)$ |
| Role hierarchy traversal | $O(|R|)$ | $O(|R|)$ |
| ABAC condition evaluation | $O(k)$ conditions | $O(1)$ |
| Trust graph reachability | $O(|V| + |E|)$ | $O(|V|)$ |
| Credential report generation | $O(|U|)$ | $O(|U|)$ |
