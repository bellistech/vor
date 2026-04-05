# Formal Security Models — Theory and Mathematical Foundations

> *Formal security models provide mathematically rigorous definitions of what it means for a system to be "secure." From the lattice-based confidentiality guarantees of Bell-LaPadula to the transaction integrity of Clark-Wilson, these models establish the theoretical foundations upon which all access control systems are built.*

---

## 1. Bell-LaPadula — Lattice Structure and Tranquility

### Formal Definition

The Bell-LaPadula model is defined as a state machine $(S, O, A, L, f)$ where:

- $S$ = set of subjects
- $O$ = set of objects
- $A$ = set of access modes $\{r, w, a, e\}$ (read, write, append, execute)
- $L$ = lattice of security levels
- $f$ = state function mapping subjects/objects to security levels

A state $v = (b, M, f)$ where $b$ is the set of current accesses, $M$ is the access control matrix, and $f = (f_s, f_c, f_o)$ maps subjects to maximum security level ($f_s$), current security level ($f_c$), and objects to classification level ($f_o$).

### Properties — Formal Statements

**Simple Security (ss-property):**

$$\forall (s, o, a) \in b : a \in \{r, w\} \Rightarrow f_s(s) \geq f_o(o)$$

A subject can read an object only if the subject's clearance dominates the object's classification.

**Star Property (*-property):**

$$\forall (s, o, a) \in b : a = w \Rightarrow f_c(s) \leq f_o(o)$$

$$\forall (s, o, a) \in b : a = r \Rightarrow f_o(o) \leq f_c(s)$$

A subject can only write to objects at or above their current level, and can only read at or below.

**Discretionary Security (ds-property):**

$$\forall (s, o, a) \in b : a \in M[s, o]$$

Access must also be permitted by the discretionary access control matrix.

### Lattice Structure

Security labels form a lattice $(L, \leq)$ where each label is a pair:

$$\ell = (\text{level}, \text{categories})$$

The dominance relation:

$$\ell_1 \geq \ell_2 \iff \text{level}(\ell_1) \geq \text{level}(\ell_2) \wedge \text{categories}(\ell_1) \supseteq \text{categories}(\ell_2)$$

The lattice operations:

$$\ell_1 \sqcup \ell_2 = (\max(\text{level}_1, \text{level}_2), \text{cat}_1 \cup \text{cat}_2) \quad \text{(join/LUB)}$$

$$\ell_1 \sqcap \ell_2 = (\min(\text{level}_1, \text{level}_2), \text{cat}_1 \cap \text{cat}_2) \quad \text{(meet/GLB)}$$

### Tranquility Properties

**Strong Tranquility:**

$$\forall t : f(t) = f(0)$$

Security labels never change during system operation. This simplifies security proofs but is impractical in many real systems.

**Weak Tranquility:**

Security labels may change, but never in a way that violates the security policy. Formally:

$$\forall t, t' > t : \text{secure}(v_t) \Rightarrow \text{secure}(v_{t'})$$

Subject level changes are constrained: a subject can lower their current level ($f_c$) but the system must ensure no existing accesses violate the properties.

### The Basic Security Theorem (BST)

If the initial state $v_0$ is secure, and every state transition preserves the ss-property, *-property, and ds-property, then every reachable state is secure.

$$\text{secure}(v_0) \wedge (\forall v_i, v_{i+1} : \text{secure}(v_i) \Rightarrow \text{secure}(v_{i+1})) \Rightarrow \forall v : \text{secure}(v)$$

This is proven by induction on the state sequence.

### Known Limitations

1. **No integrity protection** — BLP only addresses confidentiality
2. **Covert channels** — timing and storage channels bypass the model
3. **Trusted subjects** — must exist outside the model to perform administrative functions (declassification)
4. **Write-down problem** — subjects cannot write status reports to lower levels without violating *-property

---

## 2. Biba — The Dual of Bell-LaPadula

### Formal Duality

Biba inverts the BLP lattice ordering for integrity:

| BLP (Confidentiality) | Biba (Integrity) |
|:---|:---|
| No Read Up: $f_s(s) \geq f_o(o)$ | No Read Down: $i_s(s) \leq i_o(o)$ |
| No Write Down: $f_c(s) \leq f_o(o)$ | No Write Up: $i_s(s) \geq i_o(o)$ |
| Protects disclosure | Protects modification |

Where $i_s$ and $i_o$ are integrity levels of subjects and objects respectively.

### Integrity Level Assignment

Integrity levels reflect trustworthiness of the source:

$$i(\text{data}) = f(\text{source reliability}, \text{verification level}, \text{modification history})$$

| Integrity Level | Meaning | Example |
|:---:|:---|:---|
| Highest | Formally verified, cryptographically signed | Signed OS kernel |
| High | Tested and validated by trusted process | Production database |
| Medium | Reviewed but not formally verified | Internal documents |
| Low | Unverified, external source | User input, email attachments |

### Biba Variants

**Strict Integrity** (full Biba): both axioms enforced.

**Low-Water-Mark Policy**: subject's integrity level is lowered to the minimum of its current level and any object it reads:

$$i_s'(s) = \min(i_s(s), i_o(o)) \text{ after } s \text{ reads } o$$

This is more permissive — allows reading down but contaminates the subject. Analogous to taint tracking.

**Ring Policy**: only the write axiom is enforced (no write up). Subjects can read any integrity level but writing is restricted. Less restrictive but still prevents unauthorized modification of high-integrity data.

---

## 3. Clark-Wilson — Well-Formed Transactions

### Theoretical Foundation

Clark-Wilson models integrity through the concept of **well-formed transactions** — the idea that data can only be modified through approved procedures that maintain consistency.

### Formal Model

The system state at time $t$ is valid if:

$$\forall \text{CDI}_i : \text{IVP}_j(\text{CDI}_i) = \text{TRUE}$$

A transformation procedure $\text{TP}_k$ maintains integrity if:

$$\text{valid}(S_t) \wedge S_{t+1} = \text{TP}_k(S_t) \Rightarrow \text{valid}(S_{t+1})$$

This is the **well-formed transaction property**: every TP transforms valid states to valid states.

### Access Triple Formalization

Access is defined by the relation:

$$\text{Authorized} \subseteq U \times \text{TP} \times \text{CDI}$$

User $u$ can execute $\text{TP}_k$ on $\text{CDI}_i$ only if $(u, \text{TP}_k, \text{CDI}_i) \in \text{Authorized}$.

Separation of duties requires:

$$\forall \text{critical transaction } T = (\text{TP}_a, \text{TP}_b) : \text{user}(\text{TP}_a) \neq \text{user}(\text{TP}_b)$$

No single user can execute both steps of a critical transaction.

### Comparison with BLP/Biba

| Aspect | BLP/Biba | Clark-Wilson |
|:---|:---|:---|
| Model type | Lattice-based MAC | Transaction-based |
| Access unit | Subject-object pair | User-TP-CDI triple |
| Verification | Classification labels | IVP validation |
| Enforcement | Read/write restrictions | Authorized TPs only |
| Audit | Not inherent | Mandatory logging (C4) |
| Separation of duties | Not addressed | Core requirement (C3) |
| Application | Military/government | Commercial/business |

Clark-Wilson is strictly more expressive for commercial integrity because it captures the notion of authorized procedures and duty separation, which lattice models cannot express.

---

## 4. Chinese Wall — Conflict of Interest Dynamics

### Formal Model

Let $O$ be the set of objects, partitioned into company datasets $D_1, D_2, \ldots, D_n$ and conflict-of-interest classes $C_1, C_2, \ldots, C_m$ where each $C_j$ is a set of company datasets that compete.

The access history of subject $s$ at time $t$ is:

$$H_s(t) = \{o \in O : s \text{ has accessed } o \text{ before time } t\}$$

**Read rule** — subject $s$ can read object $o$ at time $t$ if:

$$\forall o' \in H_s(t) : [\text{company}(o') = \text{company}(o)] \vee [\text{class}(o') \neq \text{class}(o)]$$

Either $o$ is in the same company dataset as something $s$ already accessed, or $o$ is in a completely different conflict class.

**Write rule** — subject $s$ can write to object $o$ only if:

$$s \text{ can read } o \quad \wedge \quad \nexists o' \in H_s(t) : [\text{class}(o') = \text{class}(o) \wedge \text{company}(o') \neq \text{company}(o)]$$

The write rule prevents indirect information flow: a consultant who has read Bank A data cannot write to a shared resource that someone with Bank B access could read.

### Dynamic Nature

The key distinction of Brewer-Nash is that access rights change over time based on behavior:

$$\text{Permissions}(s, t_1) \supseteq \text{Permissions}(s, t_2) \quad \text{for } t_2 > t_1$$

Permissions can only decrease (or stay the same). The system becomes more restrictive as the subject accesses more objects. This is the opposite of most access control models where permissions are statically assigned.

### Maximum Access Theorem

A subject can access at most one company dataset from each conflict class. If there are $m$ conflict classes, the maximum number of company datasets a subject can access is $m$ (one per class).

---

## 5. Model Composition and Covert Channels

### Composing Multiple Models

Real systems often need both confidentiality and integrity. Lipner's combination model demonstrates that BLP and Biba can be composed:

| Entity | BLP Levels (Confidentiality) | Biba Levels (Integrity) |
|:---|:---|:---|
| System processes | System Low / System High | System High |
| Users | Depends on clearance | Depends on role |
| Production data | Based on classification | High integrity |
| Development code | Lower classification | Lower integrity |

The combined access decision:

$$\text{Allow}(s, o, a) = \text{BLP}(s, o, a) \wedge \text{Biba}(s, o, a)$$

Both models must independently permit the access.

### Covert Channel Analysis

A covert channel exists when information flows between security levels in violation of the model's intent, using mechanisms not designed for communication.

**Storage channels** — modulate a shared attribute:

$$\text{sender}_H \xrightarrow{\text{modifies shared resource}} \text{receiver}_L$$

Examples: file existence, file size, disk quota usage, error codes.

**Timing channels** — modulate observable timing:

$$\text{sender}_H \xrightarrow{\text{modulates response time}} \text{receiver}_L$$

Examples: CPU scheduling patterns, lock contention, cache hit/miss timing.

**Bandwidth estimation** for a covert channel with $n$ distinguishable states observed over time $t$:

$$C = \frac{\log_2(n)}{t} \text{ bits/second}$$

Mitigation: noise injection, resource partitioning, bandwidth reduction (add random delays).

### Formal Covert Channel Analysis Methods

| Method | Approach | Detects |
|:---|:---|:---|
| Shared Resource Matrix (SRM) | Identify shared resources between levels | Storage channels |
| Information flow analysis | Track all data dependencies | Both types |
| Noninterference testing | Verify output independence from high inputs | Both types |
| Kemmerer's method | Systematic SRM-based analysis | Storage channels |

---

## 6. Mathematical Formalization

### Category Theory View

Access control models can be expressed as categories:

- **Objects** of the category: system states
- **Morphisms**: state transitions
- The security property is a functor that must be preserved under composition of morphisms

This abstraction reveals that:
- BLP is a contravariant functor on the confidentiality lattice
- Biba is a covariant functor on the integrity lattice
- Composition corresponds to the product category

### Decidability Results

| Model | Safety Question Decidable? | Complexity |
|:---|:---:|:---|
| General access matrix (HRU) | No (undecidable) | Equivalent to Halting Problem |
| Mono-operational HRU | Yes | PSPACE-complete |
| Take-Grant | Yes | $O(n + e)$ linear time |
| BLP (static labels) | Yes | Polynomial |
| Typed access matrix (TAM) | Yes | NP-complete |
| RBAC (static roles) | Yes | Polynomial |

The fundamental result: determining whether a general access control system can reach an unsafe state is equivalent to the Halting Problem. This is why practical systems use restricted models.

### Information-Theoretic Security

A system achieves perfect security (in the information-theoretic sense) if:

$$I(H; L) = 0$$

Where $I(H; L)$ is the mutual information between high-level and low-level observables. This means low-level observations provide zero information about high-level data.

In practice, this is relaxed to computational security:

$$I(H; L) \leq \epsilon \text{ for negligible } \epsilon$$

---

## 7. Real-World Applications

### Model-to-Implementation Mapping

| Model | Real-World Implementation |
|:---|:---|
| Bell-LaPadula | SELinux MLS, CIPSO labels, classified networks |
| Biba | Windows Mandatory Integrity Control (MIC), code signing |
| Clark-Wilson | Database stored procedures, ERP workflow engines |
| Brewer-Nash | CRM access controls in consulting firms, FINRA compliance |
| Lattice | Security labels in Trusted Solaris, RSBAC |
| RBAC (practical hybrid) | Active Directory, IAM platforms, Kubernetes RBAC |

### SELinux as Multi-Model Implementation

SELinux implements aspects of multiple formal models:

```
# Type Enforcement (TE) — Clark-Wilson-like
# Domains (subjects) can only access types (objects)
# through allowed transitions (TPs)

# MLS/MCS — Bell-LaPadula lattice
# Security levels: s0-s15 (sensitivity)
# Categories: c0-c1023
# Label format: user:role:type:level

# RBAC — role-based overlay
# Users assigned to roles, roles to domains
# Separation of duties through role restrictions
```

### Model Limitations in Practice

1. **Granularity mismatch**: models reason about subjects and objects; real systems have processes, threads, files, network sockets
2. **Trusted computing base (TCB)**: every model requires some trusted component outside the model (the reference monitor)
3. **Administrative actions**: classification changes, role assignments, and policy updates must be handled by trusted subjects
4. **Usability**: strict model enforcement (especially BLP *-property) often conflicts with operational requirements

---

## 8. Summary of Formal Properties

| Model | Property Protected | Direction | Lattice-Based | Decidable |
|:---|:---|:---|:---:|:---:|
| Bell-LaPadula | Confidentiality | No Read Up, No Write Down | Yes | Yes |
| Biba | Integrity | No Read Down, No Write Up | Yes | Yes |
| Clark-Wilson | Integrity | Well-formed transactions | No | N/A |
| Brewer-Nash | Conflict of interest | Dynamic restriction | No | Yes |
| Graham-Denning | Secure operations | 8 primitive rules | No | Yes |
| HRU | General access control | Safety question | No | No (general) |
| Take-Grant | Rights propagation | Graph reachability | No | Yes |
| Noninterference | All flows | Zero mutual information | Yes | Yes |

## Prerequisites

- set theory, lattice theory, automata theory, computability theory, abstract algebra

## Complexity

| Operation | Time | Space |
|-----------|------|-------|
| BLP access decision (fixed labels) | O(1) | O(n × m) for matrix |
| Take-Grant reachability | O(n + e) | O(n + e) |
| HRU safety (mono-operational) | PSPACE | Exponential |
| Chinese Wall access check | O(k) per conflict class | O(n) access history |

---

*Formal security models are not merely academic exercises — they define the precise conditions under which a system can claim to be secure. Every mandatory access control implementation, every separation of duty policy, and every information flow restriction traces its theoretical foundation to these mathematical structures.*
