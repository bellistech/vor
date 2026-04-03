# The Mathematics of ACLs — Access Control as Set Theory

> *Access Control Lists extend Unix permissions from a fixed 3-tuple (owner, group, other) to an arbitrary set of (principal, permission) pairs. The mathematics are rooted in set theory, lattice structures, and permission inheritance algorithms.*

---

## 1. POSIX ACL Model — Extended Permission Sets

### Traditional Unix Permissions

Standard Unix: 3 principals, 3 permission bits each = $3 \times 3 = 9$ bits + 3 special bits = 12 bits total.

$$\text{Permissions} = 2^{12} = 4096 \text{ possible combinations}$$

### ACL Extension

POSIX ACLs add named user and group entries:

$$\text{ACL} = \{(p_i, \text{perms}_i) : p_i \in \text{Principals}, \text{perms}_i \subseteq \{r, w, x\}\}$$

| Entry Type | Tag | Count |
|:---|:---|:---:|
| owner | `user::` | 1 (required) |
| named user | `user:uid:` | 0-n |
| owning group | `group::` | 1 (required) |
| named group | `group:gid:` | 0-n |
| mask | `mask::` | 1 (if extended ACL) |
| other | `other::` | 1 (required) |

### Effective Permissions

The **mask** entry limits all named entries:

$$\text{effective}(p_i) = \text{perms}(p_i) \cap \text{mask}$$

This is a **set intersection** — the mask acts as an upper bound on named user/group permissions.

### Access Check Algorithm

For process with uid $u$ and groups $G = \{g_1, g_2, \ldots\}$:

$$\text{access}(u, G, \text{file}) = \begin{cases}
\text{owner perms} & \text{if } u = \text{file.owner} \\
\text{user:}u\text{ entry} \cap \text{mask} & \text{if named user entry exists} \\
\max_{g \in G}(\text{group:}g) \cap \text{mask} & \text{if any group entry matches} \\
\text{other perms} & \text{otherwise}
\end{cases}$$

---

## 2. NFSv4/Windows ACL Model

### ACE Structure

NFSv4 ACLs use Access Control Entries with richer semantics:

$$\text{ACE} = (\text{type}, \text{principal}, \text{permissions}, \text{flags})$$

| Type | Meaning |
|:---|:---|
| ALLOW | Grant permissions |
| DENY | Explicitly deny permissions |
| AUDIT | Log access attempts |
| ALARM | Trigger alarm on access |

### Permission Bits (NFSv4)

$$|\text{Permissions}| = 14 \text{ bits for files, 14 bits for directories}$$

Total permission combinations per ACE: $2^{14} = 16{,}384$

| Permission | Bit | File | Directory |
|:---|:---:|:---|:---|
| READ_DATA | 0 | Read content | List entries |
| WRITE_DATA | 1 | Write content | Create files |
| APPEND_DATA | 2 | Append | Create subdirs |
| EXECUTE | 5 | Execute | Traverse |
| DELETE | 9 | Delete file | Delete dir |
| READ_ACL | 12 | Read ACL | Read ACL |
| WRITE_ACL | 13 | Modify ACL | Modify ACL |

### Order-Dependent Evaluation

NFSv4 ACLs are evaluated **top to bottom, first match wins** for each permission bit:

$$\text{access}(p, \text{perm}) = \text{first ACE matching } (p, \text{perm}) \in \{\text{ALLOW, DENY}\}$$

This makes ACL ordering critical — a DENY before an ALLOW blocks access even if a later ALLOW would grant it.

### ACL Complexity

With $n$ ACEs and $k$ permission bits, the evaluation cost per access check:

$$T_{eval} = O(n \times k)$$

Windows NTFS ACLs can have hundreds of ACEs on deeply nested directories. The effective permission for a user across group memberships:

$$\text{effective} = \left(\bigcup_{i \in \text{ALLOW}} \text{perms}_i\right) \setminus \left(\bigcup_{j \in \text{DENY}} \text{perms}_j\right)$$

---

## 3. ACL Inheritance — Directory Propagation

### Inheritance Flags

| Flag | Meaning | Effect |
|:---|:---|:---|
| file_inherit (f) | ACE inherited by new files | Propagates to child files |
| dir_inherit (d) | ACE inherited by new dirs | Propagates to child dirs |
| inherit_only (i) | ACE not effective on this dir | Template only |
| no_propagate (n) | Don't inherit to grandchildren | Single-level inheritance |

### Inheritance as Tree Traversal

For a directory tree of depth $d$ with branching factor $b$:

Total objects inheriting an ACE:

$$N_{inherited} = \sum_{i=1}^{d} b^i = \frac{b^{d+1} - b}{b - 1}$$

| Depth | Branch Factor | Objects Affected |
|:---:|:---:|:---:|
| 3 | 10 | 1,110 |
| 5 | 10 | 111,110 |
| 3 | 100 | 1,010,100 |
| 5 | 5 | 3,905 |

Changing one inherited ACE at the root can affect millions of objects.

### Effective Permission Calculation (Windows)

$$\text{Effective}(u) = \left(\text{Explicit}(u) \cup \bigcup_{g \in \text{Groups}(u)} \text{Explicit}(g) \cup \text{Inherited}(u) \cup \bigcup_{g} \text{Inherited}(g)\right) \setminus \text{Deny}(u, G)$$

Priority order: Explicit Deny > Explicit Allow > Inherited Deny > Inherited Allow.

---

## 4. ACL Storage Overhead

### POSIX ACL Storage

Each ACL entry: 8 bytes (tag type + qualifier + permissions).

$$\text{ACL size} = 4 + 8n \text{ bytes}$$

Where $n$ is the number of entries (minimum 3: owner, group, other).

| Entries | Size | Filesystem Overhead |
|:---:|:---:|:---|
| 3 (minimal) | 28 bytes | Stored in inode |
| 10 | 84 bytes | Extended attribute |
| 50 | 404 bytes | Extended attribute |
| 100 | 804 bytes | Separate block |

### Comparison: ACL vs Traditional Permissions

| Feature | chmod (12 bits) | POSIX ACL | NFSv4 ACL |
|:---|:---:|:---:|:---:|
| Storage per file | 2 bytes | 28-800 bytes | 100-4000 bytes |
| Principals | 3 | Unlimited | Unlimited |
| Permission granularity | 3 bits/principal | 3 bits | 14 bits |
| Deny rules | No | No | Yes |
| Inheritance | umask only | Default ACL | Full flags |

---

## 5. Capability vs ACL — Theoretical Comparison

### Access Control Matrix

The access control matrix $M$ has subjects as rows, objects as columns:

$$M[s, o] = \text{permissions of subject } s \text{ on object } o$$

| Approach | Stores | Lookup Cost | Revocation |
|:---|:---|:---:|:---|
| ACL (column) | Permissions per object | $O(|ACL|)$ per check | Easy (edit object's ACL) |
| Capability (row) | Permissions per subject | $O(1)$ with token | Hard (find all copies) |

### When ACLs Win

- **Revocation**: Remove one ACL entry vs. revoking distributed capabilities
- **Audit**: Enumerate who can access object $o$ in $O(|ACL_o|)$
- **Policy change**: Update one ACL vs. updating all capability holders

### When Capabilities Win

- **Delegation**: Pass capability token without modifying ACLs
- **Scalability**: No central authority needed
- **POLA**: Minimum necessary permissions embedded in token

---

## 6. Summary of Functions by Type

| Concept | Math Type | Application |
|:---|:---|:---|
| Permission set $\{r,w,x\}$ | Power set ($2^3 = 8$) | File permissions |
| Mask intersection | Set intersection | Effective permissions |
| ACE evaluation | Ordered first-match | NFSv4 access check |
| Inheritance tree | Geometric series | Permission propagation |
| Access matrix | 2D Boolean array | ACL vs capability model |
| Effective perms | Set union minus deny | Windows NTFS |

---

*ACLs transform the simple question "can user X access file Y?" into a formal set-theoretic computation — evaluated billions of times per second across every filesystem operation on every multi-user system.*
