# Access Control Models — Formal Theory and Architecture

> *Access control is the selective restriction of access to resources. Formal models provide mathematical guarantees about security properties — confidentiality, integrity, and availability — through well-defined rules governing subject-object interactions.*

---

## 1. Formal Access Control Theory

### The Access Control Matrix (Lampson, 1971)

The foundational abstraction: a matrix $A$ where rows represent subjects $S$, columns represent objects $O$, and each cell $A[s,o]$ contains the set of access rights.

```
           File1    File2    Printer    Process2
User_A   {r,w,own}  {r}      {print}     {}
User_B     {r}    {r,w,own}  {print}    {signal}
Process1   {r}      {}       {print}    {r,w}
```

**Fundamental problem:** The matrix is sparse and grows as $|S| \times |O|$ — impractical to store directly.

**Two decompositions:**

| Approach | Storage | Perspective |
|:---|:---|:---|
| ACL (Access Control List) | Column-wise: for each object, list authorized subjects | "Who can access this object?" |
| Capability List | Row-wise: for each subject, list accessible objects | "What can this subject access?" |

### The Safety Problem (Harrison-Ruzzo-Ullman, 1976)

Given an access control system with a set of commands that can modify the matrix (create/delete subjects, create/delete objects, enter/remove rights):

**Safety question:** "Can a particular right $r$ ever appear in cell $A[s,o]$?"

**HRU Result:** The safety problem is **undecidable** in the general case. It is decidable only for mono-operational commands (single primitive operation per command).

This means no general algorithm can determine whether an arbitrary access control configuration will ever leak a particular privilege.

### Take-Grant Model (Jones, Lipton, Snyder, 1976)

Models access control as a directed graph where:
- Nodes = subjects and objects
- Edges = rights (including special `take` and `grant` rights)

**Rules:**
- **Take:** If $s_1 \xrightarrow{take} s_2 \xrightarrow{r} o$, then $s_1$ can acquire right $r$ to $o$
- **Grant:** If $s_1 \xrightarrow{grant} s_2$ and $s_1 \xrightarrow{r} o$, then $s_1$ can give $s_2$ right $r$ to $o$
- **Create/Remove:** Subject can create new objects or remove own rights

The sharing question is decidable in linear time $O(n)$ for this model.

---

## 2. Bell-LaPadula and Biba — Formal Definitions

### Bell-LaPadula (BLP) — Confidentiality Model

A state machine model where each state is a triple $(b, M, f)$:
- $b$ = current access set (subject, object, access-mode triples)
- $M$ = access matrix
- $f$ = security level function assigning clearance to subjects, classification to objects

**Security properties:**

**Simple Security Property (ss-property):** A state $(b, M, f)$ satisfies ss-property iff for every $(s, o, read) \in b$:

$$f_s(s) \geq f_o(o)$$

The subject's clearance must dominate the object's classification.

**Star Property (*-property):** For every $(s, o, write) \in b$:

$$f_o(o) \geq f_c(s)$$

The object's classification must dominate the subject's current security level. This prevents writing classified information down.

**Discretionary Security Property (ds-property):** Access must also be permitted by the access matrix $M$.

**Basic Security Theorem:** If every state transition preserves ss, *, and ds properties, and the initial state is secure, then every subsequent state is secure.

**Tranquility principle:** Security labels never change (strong) or change only according to defined rules (weak).

### Biba Integrity Model

Dual of Bell-LaPadula for integrity:

$$\text{Simple Integrity:} \quad f_i(s) \leq f_i(o) \implies s \text{ can read } o$$
$$\text{Star Integrity:} \quad f_i(s) \geq f_i(o) \implies s \text{ can write } o$$

"No read down, no write up" — prevents corruption of high-integrity data.

### Clark-Wilson Integrity Model

Focuses on well-formed transactions and separation of duty:

- **Constrained Data Items (CDI):** Objects subject to integrity controls
- **Unconstrained Data Items (UDI):** Inputs not yet validated
- **Integrity Verification Procedures (IVP):** Verify CDI consistency
- **Transformation Procedures (TP):** Only authorized operations on CDIs
- **Certification rules:** Ensure TPs maintain integrity invariants
- **Enforcement rules:** Ensure only authorized subjects execute TPs on specific CDIs

This maps directly to commercial requirements (accounting systems, databases).

### Brewer-Nash (Chinese Wall) Model

Dynamic separation of duty:
- Objects are grouped into **conflict-of-interest classes**
- Once a subject accesses data from company A, they cannot access data from competing company B in the same class
- Access history determines future access rights

$$\text{Access}(s, o) \text{ allowed iff } \forall o' \in \text{history}(s): \text{class}(o) \neq \text{class}(o') \lor \text{company}(o) = \text{company}(o')$$

---

## 3. NIST RBAC Standard (INCITS 359-2012)

### Formal Definition

**Core RBAC (RBAC0):**

- $USERS$, $ROLES$, $PRMS$ (permissions), $SESSIONS$ — finite sets
- $UA \subseteq USERS \times ROLES$ — user-to-role assignment
- $PA \subseteq PRMS \times ROLES$ — permission-to-role assignment
- $user: SESSIONS \rightarrow USERS$ — maps session to user
- $roles: SESSIONS \rightarrow 2^{ROLES}$ — active roles in session
- Constraint: $roles(s) \subseteq \{r \mid (user(s), r) \in UA\}$

**Hierarchical RBAC (RBAC1):**

- $RH \subseteq ROLES \times ROLES$ — role hierarchy (partial order)
- $r_1 \geq r_2$ means $r_1$ inherits all permissions of $r_2$
- General hierarchy: allows multiple inheritance (lattice)
- Limited hierarchy: restricted to tree structure (single parent)

**Constrained RBAC (RBAC2):**

- $SSD \subseteq 2^{ROLES} \times \mathbb{N}$ — static separation of duty
- $(rs, n) \in SSD \implies \forall t \subseteq rs: |t| \geq n \implies \neg\exists u \in USERS: \forall r \in t: (u,r) \in UA$
- "No user may be assigned to $n$ or more roles from the set $rs$"
- $DSD$ — same constraint but on $roles(s)$ (activated roles per session)

**Symmetric RBAC (RBAC3):** RBAC1 + RBAC2 combined.

### Administrative RBAC (ARBAC97)

Controls who can administer RBAC itself:
- **URA (User-Role Assignment):** Which admin roles can assign which user-roles
- **PRA (Permission-Role Assignment):** Which admin roles can assign permissions
- **RRA (Role-Role Assignment):** Which admin roles can modify the role hierarchy

Prerequisite conditions define necessary roles before assignment.

---

## 4. ABAC Architecture — XACML Deep Dive

### XACML 3.0 Policy Language

XACML (eXtensible Access Control Markup Language) is an OASIS standard for ABAC policy expression and evaluation.

**Policy structure:**

```
PolicySet
├── Policy (combining algorithm: deny-overrides)
│   ├── Target (applicability filter)
│   ├── Rule 1 (Effect: Permit)
│   │   ├── Target
│   │   └── Condition (boolean expression over attributes)
│   ├── Rule 2 (Effect: Deny)
│   └── ObligationExpressions
└── PolicySet (nested, with own combining algorithm)
```

**Combining algorithms** resolve conflicts when multiple rules/policies apply:

| Algorithm | Behavior |
|:---|:---|
| deny-overrides | Any Deny wins; Permit only if no Deny |
| permit-overrides | Any Permit wins; Deny only if no Permit |
| first-applicable | First matching rule's effect is final |
| only-one-applicable | Error if multiple policies match |
| deny-unless-permit | Deny unless at least one Permit (no NotApplicable) |
| permit-unless-deny | Permit unless at least one Deny |

**Evaluation flow:**

1. PEP intercepts access request
2. PEP creates XACML Request (subject, resource, action, environment attributes)
3. PEP sends Request to PDP
4. PDP retrieves missing attributes from PIP(s)
5. PDP locates applicable PolicySet(s) via Target matching
6. PDP evaluates Rules against Conditions
7. PDP applies combining algorithm to resolve multiple decisions
8. PDP returns Response: {Permit, Deny, NotApplicable, Indeterminate}
9. PDP includes Obligations (mandatory actions) and Advice (optional)
10. PEP enforces decision and fulfills Obligations

### Attribute Categories (XACML 3.0)

| Category | Example Attributes |
|:---|:---|
| Subject | role, department, clearance, email, group membership |
| Resource | type, classification, owner, sensitivity, location |
| Action | action-id (read, write, delete, approve) |
| Environment | current-time, ip-address, authentication-method |

### Policy Decision Complexity

XACML policy evaluation is $O(P \times R \times A)$ where:
- $P$ = number of applicable policies
- $R$ = number of rules per policy
- $A$ = number of attribute lookups per condition

Performance optimization: policy indexing by Target attributes reduces $P$ to near-constant for well-structured policies.

---

## 5. ReBAC — Zanzibar and Graph-Based Authorization

### Google Zanzibar Architecture

Zanzibar provides consistent, global authorization for Google services (Drive, YouTube, Cloud IAM) processing >10 million authorization checks per second at <10ms latency.

**Core concepts:**

**Relation tuples** form a directed graph:
$$\langle \text{object}\#\text{relation}@\text{user} \rangle$$
$$\langle \text{object}\#\text{relation}@\text{userset} \rangle$$

where a userset references another relation: `group:eng#member`

**Userset rewrite rules** define how permissions compose:

```
type document
  relations
    define owner: [user]
    define editor: [user, group#member] or owner
    define viewer: [user, group#member] or editor
    define can_share: owner
```

**Check evaluation** traverses the relation graph:

```
Check(user:alice, viewer, doc:readme)
  → Is (doc:readme#viewer@user:alice) in tuples?
  → Is user:alice an editor? (computed userset)
    → Is (doc:readme#editor@user:alice) in tuples?
    → Is user:alice an owner? (computed userset)
      → Is (doc:readme#owner@user:alice)? YES → Permit
```

### Zanzibar Consistency Model

**New Enemy Problem:** After sharing a document, a concurrent check might not see the new relation tuple due to replication lag.

**Solution:** Zookies (opaque consistency tokens):
1. Write returns a zookie encoding the write timestamp
2. Client passes zookie with subsequent Check requests
3. Zanzibar ensures the Check reads data at least as fresh as the zookie

This provides **causal consistency** without requiring global consensus on every read.

### Complexity Analysis

For a relation graph with $V$ vertices and $E$ edges:
- Simple check: $O(E)$ in worst case (graph traversal)
- With caching: amortized $O(1)$ for hot paths
- ListObjects: $O(V + E)$ (BFS/DFS from user node)

---

## 6. Capability vs ACL — Theoretical Comparison

### The Confused Deputy Problem (Hardy, 1988)

A compiler service (the "deputy") has write access to billing files and user-specified output files. A malicious user specifies the billing file as output, causing the compiler to overwrite it using its own authority.

**ACL vulnerability:** The compiler runs with ambient authority — all its permissions are always active. The ACL cannot distinguish whether access is on behalf of the user or the compiler's own need.

**Capability solution:** The compiler receives a capability (file handle) for the output file from the user. It uses its own separate capability for the billing file. The two authorities are never confused because capabilities are explicitly named.

### Principle of Least Authority (POLA)

Capabilities naturally support POLA because:
1. Authority is explicitly passed, not ambient
2. Attenuation: a capability can be wrapped to restrict operations
3. No authority by default — subjects start with no capabilities

ACLs struggle with POLA because:
1. All permissions granted to an identity are always available
2. No standard mechanism for temporary restriction
3. Permission creep is common (roles accumulate, rarely pruned)

### Confinement Problem

**Question:** Can a subject be prevented from leaking information?

**ACL approach:** Difficult — the subject has ambient authority and can use covert channels.

**Capability approach:** Possible through **membrane patterns** — a capability wrapper that mediates all access, can log, attenuate, or revoke.

### Revocation

| Aspect | ACL | Capability |
|:---|:---|:---|
| Mechanism | Modify the ACL entry | Revoke all copies of the token |
| Granularity | Per-object | Per-capability-holder |
| Immediacy | Immediate | Requires indirection layer |
| With indirection | N/A | Proxy capability; revoke the proxy |

---

## 7. Access Control in Distributed Systems

### Challenges

1. **No single reference monitor:** Multiple enforcement points across services
2. **Network partitions:** Cached authorization decisions may become stale
3. **Clock skew:** Time-based policies require synchronized clocks
4. **Token propagation:** Credentials must be securely passed between services

### Patterns

**Centralized PDP, Distributed PEP:**
- Single policy decision point (OPA, Cedar, Zanzibar)
- Each service embeds a PEP that queries the PDP
- Caching with TTL to handle PDP unavailability

**Token-based (JWT claims):**
- Authorization decisions embedded in signed tokens
- No runtime call to PDP needed
- Tradeoff: stale decisions until token expires

**Service mesh integration:**
- Istio/Envoy external authorization filter
- Authorization policy evaluated at the proxy layer
- mTLS provides service identity for service-to-service RBAC

### Least Privilege Formalization

The principle of least privilege (Saltzer & Schroeder, 1975) states:

> Every program and every user should operate using the least set of privileges necessary to complete the job.

Formally, for subject $s$ performing task $t$, the assigned permission set $P(s)$ should satisfy:

$$P(s) = P_{min}(t) = \{p \in PRMS \mid t \text{ requires } p\}$$

Enforcing this requires:
1. **Task analysis:** Enumerate required permissions per task
2. **Just-in-time provisioning:** Grant permissions only when needed
3. **Automatic de-provisioning:** Revoke when task completes
4. **Continuous monitoring:** Detect permission drift

---

## 8. Model Selection Framework

### Decision Criteria

| Criterion | Favors |
|:---|:---|
| Small team, simple resources | DAC |
| Government/military classification | MAC (BLP) |
| Enterprise with clear job functions | RBAC |
| Dynamic, context-sensitive policies | ABAC |
| Document/resource sharing workflows | ReBAC |
| Perimeter/network filtering | Rule-based |
| Regulatory compliance (SOX, HIPAA) | RBAC + SoD constraints |
| Microservices/API authorization | ABAC or ReBAC |
| Zero trust architecture | ABAC + risk-adaptive |

### Hybrid Approaches

Most production systems combine models:
- **RBAC + ABAC:** Roles provide base permissions; attributes refine with context
- **RBAC + ReBAC:** Roles for organizational access; relationships for resource sharing
- **MAC + RBAC:** Classification labels for data; roles for job functions
- **ABAC + risk-adaptive:** Attributes define policy; risk score adjusts enforcement

---

## References

- Lampson, B. "Protection" (1971) — access control matrix
- Harrison, Ruzzo, Ullman. "Protection in Operating Systems" (1976) — safety problem
- Bell, LaPadula. "Secure Computer Systems" (1973) — BLP model
- Biba, K.J. "Integrity Considerations for Secure Computer Systems" (1977)
- Clark, Wilson. "A Comparison of Commercial and Military Computer Security Policies" (1987)
- Brewer, Nash. "The Chinese Wall Security Policy" (1989)
- NIST SP 800-162: Guide to ABAC Definition and Considerations
- NIST INCITS 359-2012: RBAC Standard
- OASIS XACML 3.0 Standard
- Zanzibar: Google's Consistent, Global Authorization System (USENIX ATC 2019)
- Hardy, N. "The Confused Deputy" (1988)
- Saltzer, Schroeder. "The Protection of Information in Computer Systems" (1975)
