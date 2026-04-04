# The Mathematics of SELinux — Mandatory Access Control as Type Theory

> *SELinux implements mandatory access control through a formal type enforcement model. Every access decision is a lookup in a policy matrix indexed by (subject type, object type, object class, permission) — a four-dimensional Boolean function evaluated millions of times per second.*

---

## 1. Security Context — The Four-Tuple

### Context Structure

Every process and file in SELinux has a security context:

$$\text{context} = (\text{user}, \text{role}, \text{type}, \text{level})$$

Example: `system_u:system_r:httpd_t:s0`

| Field | Domain | Cardinality (typical) |
|:---|:---|:---:|
| User | SELinux users | 5-20 |
| Role | RBAC roles | 10-30 |
| Type | Type enforcement labels | 3,000-5,000 |
| Level | MLS/MCS sensitivity | 1,024 categories |

### Total Context Space

$$|\mathcal{C}| = |U| \times |R| \times |T| \times |L|$$

For a typical targeted policy: $10 \times 20 \times 4000 \times 1024 = 819{,}200{,}000$ possible contexts.

In practice, only a small subset is valid (constrained by user-role-type mappings).

---

## 2. Type Enforcement — The Access Matrix

### Policy as a Boolean Function

$$\text{allow}(s, t, c, p) \in \{0, 1\}$$

Where:
- $s$ = source type (subject/process)
- $t$ = target type (object/file)
- $c$ = object class (file, socket, process, etc.)
- $p$ = permission (read, write, execute, etc.)

### Access Vector Table Dimensions

| Dimension | Count (targeted policy) | Examples |
|:---|:---:|:---|
| Source types | ~4,000 | httpd_t, sshd_t, init_t |
| Target types | ~4,000 | httpd_log_t, shadow_t |
| Object classes | ~80 | file, dir, socket, process |
| Permissions | ~40 per class | read, write, open, getattr |

### Policy Size

Total possible rules:

$$|\text{Rules}| = |S| \times |T| \times |C| \times |P| = 4000 \times 4000 \times 80 \times 40 = 5.12 \times 10^{10}$$

Actual rules in targeted policy: ~400,000 (0.0008% of the space is allowed).

**Default deny principle:** The ratio of denied to allowed:

$$\frac{\text{denied}}{\text{allowed}} \approx \frac{5.12 \times 10^{10}}{400{,}000} = 128{,}000 : 1$$

---

## 3. Policy Compilation — Binary Representation

### AV Hash Table

The compiled policy uses a hash table for O(1) lookups:

$$\text{bucket} = \text{hash}(s, t, c) \pmod{n_{buckets}}$$

| Policy Version | Compiled Size | Rules | Load Time |
|:---:|:---:|:---:|:---:|
| Targeted (minimal) | ~2 MB | ~100K | 0.1 s |
| Targeted (full) | ~8 MB | ~400K | 0.5 s |
| MLS (strict) | ~15 MB | ~800K | 1.0 s |

### Access Vector Cache (AVC)

The kernel caches recent decisions to avoid repeated policy lookups:

$$\text{AVC hit rate} = \frac{\text{cache hits}}{\text{total lookups}} \approx 99.5\%$$

AVC size: typically 512-1024 entries. With a hot cache:

$$T_{decision} = \begin{cases} O(1) & \text{AVC hit (< 1 } \mu\text{s)} \\ O(\log n) & \text{AVC miss, policy lookup (5-20 } \mu\text{s)} \end{cases}$$

### Performance Impact

$$\text{Overhead}_{SELinux} = (1 - h) \times T_{miss} + h \times T_{hit}$$

Where $h$ is the hit rate.

With 99.5% hit rate: $0.005 \times 15\mu s + 0.995 \times 0.5\mu s = 0.57\mu s$ average per access check.

On a system performing 100,000 access checks/second: ~57 ms total CPU time per second (<0.006% overhead).

---

## 4. Multi-Level Security (MLS) — Lattice Model

### Bell-LaPadula Model

MLS implements the Bell-LaPadula confidentiality model:

**No Read Up (Simple Security):**
$$\text{read allowed} \iff L(s) \geq L(o)$$

**No Write Down (*-Property):**
$$\text{write allowed} \iff L(s) \leq L(o)$$

Where $L(s)$ is the subject's clearance and $L(o)$ is the object's classification.

### Sensitivity Levels

$$\text{level} = (s, C) \quad \text{where } s \in \text{sensitivities}, C \subseteq \text{categories}$$

Dominance relation (partial order):

$$(s_1, C_1) \geq (s_2, C_2) \iff s_1 \geq s_2 \land C_1 \supseteq C_2$$

### Category Combinations

With $n$ categories, the number of possible compartment sets:

$$|\mathcal{P}(\text{categories})| = 2^n$$

| Categories | Compartment Sets | Use Case |
|:---:|:---:|:---|
| 10 | 1,024 | Small deployment |
| 256 | $1.16 \times 10^{77}$ | Standard (container isolation) |
| 1024 | $1.80 \times 10^{308}$ | Maximum |

MCS (Multi-Category Security) in container runtimes uses categories to isolate containers — each container gets a unique category pair $(c_i, c_j)$, providing $\binom{1024}{2} = 523{,}776$ unique isolation labels.

---

## 5. Type Transition Rules

### Automatic Transitions

When a process of type $s$ creates an object of class $c$ in a directory of type $t$:

$$\text{type\_transition } s \; t : c \rightarrow t'$$

Example: `type_transition httpd_t var_log_t : file httpd_log_t`

When httpd_t creates a file in a var_log_t directory, the file gets type httpd_log_t.

### Domain Transition (Process)

When a process of type $s$ executes a binary of type $t$:

$$\text{type\_transition } s \; t : \text{process} \rightarrow s'$$

Three conditions must all be met:

1. `allow s t : file execute` — source can execute the binary
2. `allow s s' : process transition` — source can transition to new domain
3. `allow s' t : file entrypoint` — binary type is an entrypoint for new domain

This forms a **directed graph** of allowed domain transitions:

$$G_{transition} = (T_{domains}, E_{transitions})$$

Where $|E| \ll |T|^2$ — only a small fraction of transitions are allowed.

---

## 6. Boolean Conditionals

### Policy Booleans

SELinux booleans toggle rules at runtime without recompiling:

$$\text{effective}(r) = \text{allow}(r) \land \text{bool}(r)$$

| Boolean | Default | Effect When Toggled |
|:---|:---:|:---|
| httpd_can_network_connect | off | Apache can connect to any port |
| httpd_enable_cgi | off | CGI script execution |
| allow_user_mysql_connect | off | Users can connect to MySQL |
| ftp_home_dir | off | FTP access to home dirs |

### Combinatorial Impact

With $b$ booleans, there are $2^b$ possible policy configurations:

| Booleans | Configurations | Audit Effort |
|:---:|:---:|:---|
| 10 | 1,024 | Manageable |
| 50 | $1.13 \times 10^{15}$ | Intractable to test all |
| 100 | $1.27 \times 10^{30}$ | Requires formal analysis |

RHEL ships with ~300 booleans — exhaustive testing is impossible, requiring risk-based analysis.

---

## 7. Audit Log Analysis

### Denial Message Format

```
avc: denied { read } for pid=1234 comm="httpd"
  name="index.html" dev="sda1" ino=5678
  scontext=system_u:system_r:httpd_t:s0
  tcontext=unconfined_u:object_r:user_home_t:s0
  tclass=file permissive=0
```

### Denial Rate as Policy Fitness Metric

$$\text{Denial rate} = \frac{\text{AVC denials per minute}}{\text{Total access checks per minute}}$$

| Denial Rate | Interpretation | Action |
|:---:|:---|:---|
| 0 | Policy complete or permissive mode | Verify enforcing mode |
| < 0.01% | Well-tuned policy | Monitor |
| 0.01-1% | Policy gaps exist | Investigate + fix |
| > 1% | Significant policy mismatch | Major policy work needed |

### audit2allow Workflow

Denials → Policy rules: $O(n)$ where $n$ = unique denial patterns.

Each unique $(s, t, c, p)$ tuple generates one allow rule. Batch processing:

```
audit2allow -M mymodule < /var/log/audit/audit.log
semodule -i mymodule.pp
```

---

## 8. Summary of Functions by Type

| Concept | Math Type | Application |
|:---|:---|:---|
| Context 4-tuple | Cartesian product | Security labeling |
| allow(s,t,c,p) | 4D Boolean function | Access decision |
| AVC cache | Hash table lookup | Performance optimization |
| Bell-LaPadula | Lattice partial order | Confidentiality model |
| $2^n$ categories | Power set | MCS container isolation |
| Domain transitions | Directed graph | Process confinement |
| $2^b$ booleans | Exponential configuration | Policy tunability |

## Prerequisites

- type theory, Boolean algebra, state machines, set theory

---

*SELinux evaluates millions of access decisions per second through a compiled Boolean policy — it's a type system for the operating system, where every process and file has a type, and the policy defines exactly which types can interact.*
