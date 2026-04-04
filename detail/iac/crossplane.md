# The Mathematics of Crossplane — Reconciliation Loops, Composition Graphs & Eventual Consistency

> *Crossplane implements the Kubernetes controller pattern to reconcile desired cloud state with actual state. The mathematics of control theory, graph composition, and eventual consistency models explain convergence behavior, composition complexity, and the guarantees platform teams can expect from declarative infrastructure.*

---

## 1. Reconciliation as Control Loop (Control Theory)

### The Problem

Each Crossplane controller watches resources and drives actual state toward desired state. The reconciliation loop runs periodically or on events. How quickly does actual state converge?

### The Formula

Model the system as a discrete-time control loop with error $e_k$ at iteration $k$:

$$e_k = S_{desired} - S_{actual,k}$$

$$S_{actual,k+1} = S_{actual,k} + \alpha \cdot e_k$$

Where $\alpha$ is the correction factor (0 = no progress, 1 = instant convergence). Convergence after $n$ iterations:

$$e_n = (1 - \alpha)^n \cdot e_0$$

Time to converge within tolerance $\epsilon$:

$$n = \frac{\ln(\epsilon / e_0)}{\ln(1 - \alpha)}$$

### Worked Examples

**Cloud API with 80% success rate ($\alpha = 0.8$), initial error = 5 resources:**

$$n = \frac{\ln(0.5 / 5)}{\ln(0.2)} = \frac{-2.303}{-1.609} = 1.43 \text{ iterations}$$

After 2 reconciliation cycles, error < 0.5 (effectively converged).

**Flaky API ($\alpha = 0.3$), 10 resources to create:**

$$n = \frac{\ln(0.5 / 10)}{\ln(0.7)} = \frac{-2.996}{-0.357} = 8.4 \text{ iterations}$$

At 30-second sync period: $8.4 \times 30 = 252$ seconds to converge.

---

## 2. Composition Graph Complexity (Graph Theory)

### The Problem

A Composition maps one XR to $N$ managed resources with patches between them. The patch graph determines execution complexity and the number of field transformations.

### The Formula

For a Composition with $R$ resources and $P$ patches:

$$Complexity_{patch} = O(P \cdot T_{transform})$$

Where $T_{transform}$ is the average cost per patch transform (map, convert, string format).

The total field resolution for a claim:

$$Fields_{resolved} = F_{claim} + \sum_{i=1}^{R} (F_{base,i} + P_i)$$

Where $F_{claim}$ = claim fields, $F_{base,i}$ = base resource fields, $P_i$ = patches applied to resource $i$.

### Worked Examples

**Database composition: 1 XR -> 4 resources (RDS, SubnetGroup, SecurityGroup, ParameterGroup), 12 patches:**

$$Fields_{resolved} = 5 + (15 + 3) + (8 + 2) + (6 + 4) + (10 + 3) = 56$$

**Full application stack: 1 XR -> 12 resources, 35 patches:**

$$Fields_{resolved} = 8 + \sum_{i=1}^{12}(F_{base,i} + P_i) \approx 8 + 12(12 + 3) = 188$$

Reconciliation time scales linearly with total fields.

---

## 3. Eventual Consistency Window (Distributed Systems)

### The Problem

After applying a claim, the infrastructure takes time to become ready. The consistency window depends on the reconciliation period, cloud API latency, and dependency chains.

### The Formula

$$T_{consistency} = T_{observe} + T_{reconcile} + T_{provision}$$

For a chain of $d$ dependent resources:

$$T_{total} = \sum_{k=1}^{d} (T_{sync,k} + T_{provision,k})$$

Where $T_{sync,k}$ is the time until the controller notices resource $k-1$ is ready (bounded by sync period $S$):

$$E[T_{sync}] = S / 2$$

### Worked Examples

**3-resource dependency chain (VPC -> Subnet -> Instance), sync period 30s, provision times 10s, 30s, 60s:**

$$T_{total} = (15 + 10) + (15 + 30) + (15 + 60) = 145 \text{ seconds}$$

**Same chain with 10s sync period:**

$$T_{total} = (5 + 10) + (5 + 30) + (5 + 60) = 115 \text{ seconds}$$

Reducing sync period from 30s to 10s saves 30s (20%) but triples API calls. The tradeoff:

$$API_{calls/hour} = \frac{R \times 3600}{S}$$

At 30s sync with 50 resources: $\frac{50 \times 3600}{30} = 6000$ calls/hour.
At 10s sync: $18000$ calls/hour. API rate limits become a concern.

---

## 4. XRD Schema Validation (Type Theory)

### The Problem

XRDs define OpenAPI schemas that constrain claim inputs. The number of valid configurations is the product of allowed values per field. How many unique configurations does an XRD allow?

### The Formula

For $F$ fields, each with domain $D_i$:

$$|Configs| = \prod_{i=1}^{F} |D_i|$$

With constraints (mutual exclusivity, conditional requirements):

$$|Valid| = |Configs| - |Violated|$$

### Worked Examples

**Database XRD: size (3 values), engine (2 values), region (4 values), ha (bool):**

$$|Configs| = 3 \times 2 \times 4 \times 2 = 48$$

**With constraint "HA only available for medium/large":**

$$|Violated| = 1 \times 2 \times 4 \times 1 = 8 \quad \text{(small + ha=true)}$$

$$|Valid| = 48 - 8 = 40$$

This determines the test matrix size for composition validation.

---

## 5. Provider Resource Coverage (Set Theory)

### The Problem

A provider covers a subset of a cloud's API surface. Platform completeness depends on provider coverage relative to required resources.

### The Formula

$$Coverage = \frac{|R_{provider} \cap R_{needed}|}{|R_{needed}|}$$

$$Gap = R_{needed} \setminus R_{provider}$$

For multi-provider platforms:

$$Coverage_{multi} = \frac{|R_{needed} \cap \bigcup_{i=1}^{P} R_{provider,i}|}{|R_{needed}|}$$

### Worked Examples

**AWS provider covers 250 resource types. Platform needs 30 types:**

$$Coverage = 28/30 = 93.3\%$$

$$Gap = \{CustomDomain, WAFv2Rule\}$$

Gaps must be filled with the Kubernetes provider (raw manifests) or custom providers.

---

## 6. Drift Detection Cost (Information Theory)

### The Problem

Crossplane detects drift by comparing desired and actual state. The comparison cost scales with resource count and field count. How much API bandwidth does drift detection consume?

### The Formula

$$B_{drift} = R \times F_{avg} \times S_{field} \times \frac{3600}{T_{sync}}$$

Where:
- $R$ = managed resource count
- $F_{avg}$ = average fields per resource
- $S_{field}$ = average bytes per field comparison
- $T_{sync}$ = sync period (seconds)

### Worked Examples

**100 managed resources, 20 fields avg, 200 bytes/field, 30s sync:**

$$B_{drift} = 100 \times 20 \times 200 \times 120 = 48 \text{ MB/hour}$$

**500 resources, 60s sync:**

$$B_{drift} = 500 \times 20 \times 200 \times 60 = 120 \text{ MB/hour}$$

The entropy of drift (expected information from a check):

$$H_{drift} = -p \log_2(p) - (1-p)\log_2(1-p)$$

At $p_{drift} = 0.01$ (1% drift rate): $H = 0.081$ bits/check. Most checks find no drift, suggesting adaptive sync periods would be more efficient.

---

## Prerequisites

- control-theory, feedback-loops, discrete-systems
- graph-theory, dependency-DAGs
- eventual-consistency, CAP-theorem
- type-theory, OpenAPI-schemas
- information-theory, entropy
