# The Mathematics of Kyverno — Policy Compliance as Constraint Satisfaction

> *Every resource is a candidate. Every policy is a constraint. Admission is the proof that all constraints are satisfied.*

---

## 1. Pattern Matching (Structural Unification)

### The Problem

Kyverno validates resources by matching them against YAML patterns. This is a form of structural unification: every field in the pattern must unify with the corresponding field in the resource. How do we formalize the matching semantics and reason about pattern complexity?

### The Formula

Define a pattern $P$ and a resource $R$ as nested key-value structures. The match function $M(P, R)$ is recursive:

$$M(P, R) = \bigwedge_{k \in \text{keys}(P)} \begin{cases}
M(P[k], R[k]) & \text{if } P[k] \text{ is a map} \\
\text{glob}(P[k], R[k]) & \text{if } P[k] \text{ is a string} \\
P[k] = R[k] & \text{if } P[k] \text{ is a scalar} \\
\forall i.\, \exists j.\, M(P[k][i], R[k][j]) & \text{if } P[k] \text{ is a list}
\end{cases}$$

The complexity of matching a pattern of depth $d$ with list elements of size $l$ against a resource with list sizes $n$:

$$T(P, R) = O\left(\prod_{i=1}^{d} n_i \cdot l_i\right)$$

For typical Kubernetes resources where nesting is bounded and lists are small, this is effectively polynomial.

### Worked Examples

**Example 1**: A pattern requiring resource limits on all containers. The pattern has depth 4 (spec.containers[].resources.limits) and matches against a Pod with 3 containers:

$$T = O(3 \times 1 \times 1 \times 2) = O(6) \text{ comparisons}$$

Each container is checked for the existence of `memory` and `cpu` fields.

**Example 2**: A pattern with conditional anchors `=(hostPID): false`. The anchor means "if key exists, must equal false." This adds a conditional branch:

$$M_{\text{anchor}}(P, R) = \begin{cases}
P[k] = R[k] & \text{if } k \in \text{keys}(R) \\
\text{true} & \text{if } k \notin \text{keys}(R)
\end{cases}$$

The conditional anchor avoids false positives on resources that legitimately omit the field.

---

## 2. Mutation Ordering (Partial Order on Transformations)

### The Problem

When multiple mutate policies apply to the same resource, the final result depends on application order. Some mutations commute (order-independent), while others conflict. How do we determine a safe ordering?

### The Formula

Define mutations as functions $m_i: R \to R$ on the resource space. Two mutations commute if:

$$m_i(m_j(R)) = m_j(m_i(R)) \quad \forall R$$

For non-commuting mutations, define a dependency relation $\prec$ where $m_i \prec m_j$ means $m_i$ must apply before $m_j$. A valid ordering exists if and only if the dependency graph is a DAG:

$$\text{valid ordering} \iff \nexists \text{ cycle in } (M, \prec)$$

The number of valid orderings is the number of topological sorts of the DAG. For $n$ independent mutations:

$$|\text{orderings}| = n!$$

For a total order (full dependency chain):

$$|\text{orderings}| = 1$$

### Worked Examples

**Example 1**: Three mutations:
- $m_1$: Add label `team=backend`
- $m_2$: Add resource requests `cpu=100m`
- $m_3$: Inject sidecar container

These all modify different paths and commute: $|\text{orderings}| = 3! = 6$, all producing the same result.

**Example 2**: Two conflicting mutations:
- $m_1$: Set `replicas=3` (from policy "min-replicas")
- $m_2$: Set `replicas=5` (from policy "ha-replicas")

These do not commute: $m_1(m_2(R)).\text{replicas} = 3$ but $m_2(m_1(R)).\text{replicas} = 5$. Kyverno applies policies in alphabetical order by name, so "ha-replicas" applies first, then "min-replicas" overwrites it. The result is $\text{replicas} = 3$.

---

## 3. Generate Rule Synchronization (Eventual Consistency Convergence)

### The Problem

Generate rules create derived resources when trigger resources are created or updated. With `synchronize: true`, changes to the trigger propagate to generated resources. This is an eventual consistency problem. How long does convergence take?

### The Formula

Let $T_w$ be the webhook processing time and $T_r$ be the reconciliation interval. For a single-depth generation (trigger creates one resource):

$$T_{\text{converge}} = T_w + T_r$$

For cascading generations of depth $d$ (generated resource triggers another generation):

$$T_{\text{converge}} = d \times (T_w + T_r)$$

The consistency gap (time when generated resource is out of sync):

$$G = T_r + T_{\text{processing}}$$

The probability that a read during the gap observes stale data:

$$P(\text{stale}) = \frac{G}{\text{mean time between updates}}$$

### Worked Examples

**Example 1**: A namespace creation triggers generation of a NetworkPolicy and a ResourceQuota. Both are independent (depth 1). With $T_w = 100\text{ms}$ and $T_r = 15\text{s}$:

$$T_{\text{converge}} = 100\text{ms} + 15\text{s} = 15.1\text{s}$$

Both resources converge in parallel since they are independent.

**Example 2**: A namespace with label `team=X` triggers a ResourceQuota, which in turn (hypothetically) triggers an alert ConfigMap. Depth $d = 2$:

$$T_{\text{converge}} = 2 \times 15.1\text{s} = 30.2\text{s}$$

If the namespace label changes every 4 hours on average:

$$P(\text{stale}) = \frac{15.1}{14400} \approx 0.001$$

0.1% chance of observing stale generated resources.

---

## 4. Image Verification Cost (Cryptographic Verification Complexity)

### The Problem

Image verification rules check cosign signatures and attestations for every container image in admitted pods. Each verification involves cryptographic operations and potentially network calls to transparency logs. What is the total verification cost?

### The Formula

For a pod with $c$ containers, each referencing an image that requires $s$ signature verifications and $a$ attestation checks:

$$T_{\text{verify}} = \sum_{i=1}^{c} \left( s_i \cdot T_{\text{sig}} + a_i \cdot T_{\text{attest}} \right) + c \cdot T_{\text{registry}}$$

where $T_{\text{sig}}$ is the signature verification time (CPU-bound), $T_{\text{attest}}$ is the attestation check time, and $T_{\text{registry}}$ is the registry round-trip for fetching signature metadata.

For ECDSA-P256 (cosign default):

$$T_{\text{sig}} \approx 0.5\text{ms (CPU)} + T_{\text{network}}$$

The webhook timeout constraint:

$$T_{\text{verify}} \leq T_{\text{webhook}} = 10\text{s (default)}$$

### Worked Examples

**Example 1**: A pod with 3 containers, each requiring 1 signature verification. Registry latency is 200ms:

$$T_{\text{verify}} = 3 \times (1 \times 0.5\text{ms} + 0) + 3 \times 200\text{ms} = 1.5\text{ms} + 600\text{ms} = 601.5\text{ms}$$

Well within the 10-second timeout. Parallelizing registry calls reduces to approximately 201.5ms.

**Example 2**: A pod with 5 containers, each requiring 2 signatures and 1 attestation. Registry latency 500ms (cross-region):

$$T_{\text{verify}} = 5 \times (2 \times 0.5 + 1 \times 2) + 5 \times 500 = 15\text{ms} + 2500\text{ms} = 2515\text{ms}$$

Serial execution uses 2.5s of the 10s budget. With 3 such pods in a Deployment rollout, the total verification across all admission calls: $3 \times 2515 = 7545\text{ms}$. Each pod is a separate admission call, so they do not stack, but bursty deployments may slow down.

---

## 5. Policy Report Completeness (Coverage Metrics)

### The Problem

Background scanning evaluates all existing resources against policies, producing policy reports. How do we measure the completeness and compliance rate of a cluster?

### The Formula

Define the compliance rate $C$ as the fraction of (resource, policy) pairs that pass:

$$C = \frac{\sum_{r \in R} \sum_{p \in P} \mathbb{1}[\text{pass}(r, p)]}{|R| \times |P_r|}$$

where $P_r \subseteq P$ is the set of policies applicable to resource $r$ (based on match/exclude criteria).

The total number of evaluations:

$$E = \sum_{r \in R} |P_r|$$

The false sense of security metric (policies that match nothing):

$$\text{FSS} = |\{p \in P : \forall r \in R, r \notin \text{match}(p)\}|$$

### Worked Examples

**Example 1**: A cluster with 500 pods, 50 deployments, and 20 namespaces. Policies:
- P1: require-labels (matches Pods, Deployments) -> 550 evaluations
- P2: disallow-latest (matches Pods) -> 500 evaluations
- P3: require-limits (matches Pods) -> 500 evaluations

Total evaluations: $E = 550 + 500 + 500 = 1550$

If 30 pods fail P2 and 45 pods fail P3:

$$C = \frac{1550 - 30 - 45}{1550} = \frac{1475}{1550} \approx 0.952$$

95.2% compliance rate.

**Example 2**: A policy requiring `PodDisruptionBudget` for all Deployments matches 50 deployments. If 35 have PDBs:

$$C_{\text{pdb}} = \frac{35}{50} = 0.70$$

70% compliance. The policy report shows 15 violations, each identifying the specific non-compliant Deployment.

---

## Prerequisites

- Kubernetes admission webhooks (validating, mutating, webhook configuration)
- YAML structure and strategic merge patch semantics
- Container image registries (OCI distribution spec, image manifests, tags vs digests)
- Cosign signature format and Sigstore transparency logs
- Partial orders and directed acyclic graphs (for mutation ordering)
- Set theory (match/exclude as set intersection and difference)
- Basic cryptography (ECDSA signature verification, certificate chains)
