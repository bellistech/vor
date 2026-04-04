# The Mathematics of Capabilities — Privilege Lattices and Access Control Algebra

> *Linux capabilities decompose monolithic root privilege into a lattice of fine-grained permissions, where capability set transformations across execve() follow precise algebraic rules governing privilege flow through process hierarchies.*

---

## 1. Capability Set Algebra (Abstract Algebra)

### Capability Universe

The set of all Linux capabilities forms a finite universe:

$$\mathcal{C} = \{c_0, c_1, c_2, \ldots, c_{n-1}\}$$

Where $n = $ `CAP_LAST_CAP + 1` (currently $n = 41$ as of Linux 6.x). Each capability is a single bit in a 64-bit bitmask:

$$c_i \mapsto 2^i \quad \text{for } 0 \leq i < 64$$

A capability set is an element of the power set:

$$S \in \mathcal{P}(\mathcal{C}), \quad |\mathcal{P}(\mathcal{C})| = 2^{41} \approx 2.2 \times 10^{12}$$

### Set Operations on Capability Sets

Capability manipulation uses standard set operations implemented as bitwise logic:

| Operation | Set Theory | Bitwise | Meaning |
|:---|:---|:---|:---|
| Union | $A \cup B$ | `A \| B` | Grant additional caps |
| Intersection | $A \cap B$ | `A & B` | Restrict to common caps |
| Complement | $\overline{A}$ | `~A` | All caps not in A |
| Difference | $A \setminus B$ | `A & ~B` | Remove specific caps |
| Subset test | $A \subseteq B$ | `(A & B) == A` | A is within B |

### Privilege Ordering

Capability sets form a partial order under subset inclusion:

$$A \preceq B \iff A \subseteq B$$

This defines a Boolean lattice $(\mathcal{P}(\mathcal{C}), \subseteq)$:
- Join (least upper bound): $A \vee B = A \cup B$
- Meet (greatest lower bound): $A \wedge B = A \cap B$
- Top element: $\top = \mathcal{C}$ (all capabilities = root)
- Bottom element: $\bot = \emptyset$ (no capabilities = unprivileged)

---

## 2. Execve Transformation Rules (Transformation Theory)

### The Five Sets

Each thread maintains five capability sets, forming a state vector:

$$\vec{T} = (E, P, I, B, A)$$

Where:
- $E$ = Effective (checked by kernel)
- $P$ = Permitted (upper bound for E)
- $I$ = Inheritable (preserved across exec)
- $B$ = Bounding (absolute ceiling)
- $A$ = Ambient (auto-elevated on exec)

### File Capability Sets

Each executable may have three capability attributes:

$$\vec{F} = (F_P, F_I, F_E)$$

Where $F_E \in \{0, 1\}$ is the effective bit (scalar, not set).

### Transformation Equations

On `execve()`, the new thread capabilities are computed:

$$P' = (P \cap B) \cup (F_P \cap B) \cup (F_I \cap I)$$

$$E' = \begin{cases} P' & \text{if } F_E = 1 \\ A' & \text{if } F_E = 0 \end{cases}$$

$$I' = I$$

$$A' = (A \cap I) \cap B$$

In matrix-like notation, this is a non-linear transformation due to the conditional on $F_E$:

$$\vec{T'} = \Phi(\vec{T}, \vec{F}, B)$$

### Invariant: Monotonic Restriction of Bounding Set

The bounding set can only shrink:

$$B' \subseteq B$$

This is enforced by `prctl(PR_CAPBSET_DROP, cap)` which is irreversible:

$$B_{\text{new}} = B_{\text{old}} \setminus \{c\}$$

No operation can add capabilities to $B$, making it a monotonically decreasing function over process lifetime.

---

## 3. Ambient Capability Flow (Graph Theory)

### Privilege Inheritance Graph

Without ambient capabilities, unprivileged programs cannot inherit capabilities through `execve()` unless the binary has file capabilities. Ambient capabilities create a path in the privilege flow graph:

$$G = (V, E) \quad \text{where } V = \text{processes}, \; E = \text{exec transitions}$$

For a capability $c$ to flow from parent to child via ambient:

$$c \in A' \iff c \in A \wedge c \in I \wedge c \in B$$

This triple-intersection requirement creates a three-factor gate:

$$\text{flow}(c) = \mathbb{1}[c \in A] \cdot \mathbb{1}[c \in I] \cdot \mathbb{1}[c \in B]$$

### Reachability

A capability $c$ is reachable by a descendant process at depth $d$ if and only if:

$$c \in \bigcap_{k=0}^{d} (I_k \cap B_k)$$

Since $B$ is monotonically non-increasing and $I$ is stable across exec:

$$\text{reachable}(c, d) \implies \text{reachable}(c, d-1)$$

---

## 4. Docker Capability Model (Set Theory)

### Default Grant Set

Docker grants a specific subset $D \subset \mathcal{C}$:

$$D = \{\text{AUDIT\_WRITE}, \text{CHOWN}, \text{DAC\_OVERRIDE}, \text{FOWNER}, \ldots\}$$

$$|D| = 14, \quad |D| / |\mathcal{C}| = 14/41 \approx 34\%$$

### Container Effective Capabilities

With `--cap-drop` set $R$ and `--cap-add` set $G$:

$$E_{\text{container}} = (D \setminus R) \cup G$$

The recommended pattern `--cap-drop=ALL --cap-add=NEEDED`:

$$E = (\emptyset) \cup G = G$$

Privilege reduction ratio:

$$r = 1 - \frac{|G|}{|\mathcal{C}|} = 1 - \frac{|G|}{41}$$

| Configuration | $\|E\|$ | Reduction $r$ |
|:---|:---:|:---:|
| Default Docker | 14 | 65.9% |
| `--cap-drop=ALL --cap-add=NET_BIND` | 1 | 97.6% |
| `--cap-add=ALL` | 41 | 0% |
| `--cap-drop=ALL` | 0 | 100% |

---

## 5. Privilege Escalation Graph (Graph Theory)

### Capability Dependency Graph

Some capabilities enable acquisition of others, forming a directed dependency graph:

$$G_{\text{esc}} = (\mathcal{C}, E_{\text{esc}})$$

Where $(c_i, c_j) \in E_{\text{esc}}$ means having $c_i$ can lead to acquiring $c_j$:

| Source Capability | Reachable Capabilities | Path |
|:---|:---|:---|
| `CAP_SYS_ADMIN` | Nearly all | Mount, namespace, BPF |
| `CAP_SYS_PTRACE` | `CAP_SYS_ADMIN` | Trace privileged process |
| `CAP_DAC_OVERRIDE` | File caps | Write to any file |
| `CAP_SETUID` | All (via root) | setuid(0) |
| `CAP_SYS_MODULE` | All (kernel) | Load kernel module |
| `CAP_NET_ADMIN` | Network caps | iptables, routing |

### Transitive Closure

The full escalation potential is the transitive closure:

$$\text{reach}(c) = \{c\} \cup \bigcup_{(c,c') \in E_{\text{esc}}} \text{reach}(c')$$

For `CAP_SYS_ADMIN`:

$$|\text{reach}(\text{SYS\_ADMIN})| \approx |\mathcal{C}| - 2 \approx 39$$

This makes `CAP_SYS_ADMIN` effectively equivalent to full root in terms of escalation potential.

---

## 6. Security Metrics (Information Theory)

### Capability Entropy

The entropy of a capability configuration measures its privilege diversity:

$$H(E) = -\sum_{c \in \mathcal{C}} p(c) \log_2 p(c)$$

For uniform capability usage (each cap equally likely to be exercised):

$$H_{\max} = \log_2 |E|$$

### Attack Surface Metric

The capability attack surface combines count with impact:

$$\text{ASM} = \sum_{c \in E} w(c)$$

Where $w(c)$ is the risk weight of capability $c$:

| Capability | Risk Weight $w(c)$ | Rationale |
|:---|:---:|:---|
| `CAP_SYS_ADMIN` | 10.0 | Near-equivalent to root |
| `CAP_SYS_MODULE` | 9.5 | Kernel code execution |
| `CAP_SYS_PTRACE` | 8.5 | Process memory access |
| `CAP_NET_ADMIN` | 7.0 | Network reconfiguration |
| `CAP_NET_RAW` | 6.0 | Packet injection |
| `CAP_NET_BIND_SERVICE` | 2.0 | Privileged port only |
| `CAP_CHOWN` | 3.0 | File ownership |

### Minimum Privilege Score

$$\text{MPS} = \frac{\text{ASM}_{\text{actual}}}{\text{ASM}_{\text{minimum\_required}}}$$

Ideal: $\text{MPS} = 1.0$ (no excess privilege). Docker default:

$$\text{MPS}_{\text{Docker}} = \frac{\sum_{c \in D} w(c)}{\sum_{c \in R} w(c)}$$

Where $R$ is the set actually required by the workload.

---

## 7. Kubernetes Pod Security Algebra (Formal Verification)

### Policy as Constraint

Pod Security Standards define capability constraints as set predicates:

**Restricted:**
$$E \subseteq \{\text{NET\_BIND\_SERVICE}\} \wedge \text{drop} = \mathcal{C}$$

**Baseline:**
$$E \cap \mathcal{D} = \emptyset$$

Where $\mathcal{D}$ is the set of dangerous capabilities.

### Admission Control Decision

The admission controller computes:

$$\text{admit}(\text{pod}) = \bigwedge_{c \in \text{containers}} \text{compliant}(c.\text{securityContext}, \text{policy})$$

$$\text{compliant}(sc, p) = (sc.\text{caps.add} \subseteq p.\text{allowed}) \wedge (\mathcal{C} \setminus sc.\text{caps.drop} \subseteq p.\text{maxSet})$$

---

## Prerequisites

set-theory, boolean-algebra, lattice-theory, graph-theory, information-theory

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| Capability check (kernel) | $O(1)$ — bitmask test | $O(1)$ — 64-bit word |
| execve() transformation | $O(1)$ — 5 bitwise ops | $O(1)$ — 5 bitmasks |
| Bounding set drop | $O(1)$ — clear bit | $O(1)$ |
| getcap file scan | $O(n)$ — n files | $O(1)$ per file |
| Escalation reachability | $O(\|\mathcal{C}\|^2)$ — transitive closure | $O(\|\mathcal{C}\|^2)$ — adjacency matrix |
| Policy compliance check | $O(\|E\|)$ — set comparison | $O(\|\mathcal{C}\|)$ — bitmask |
