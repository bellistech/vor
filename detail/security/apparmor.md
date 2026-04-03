# The Mathematics of AppArmor — Path-Based Mandatory Access Control

> *AppArmor confines programs by defining allowed filesystem paths, capabilities, and network access as a whitelist policy. Its path-based model trades the completeness of SELinux's label-based approach for simplicity — profiles are glob-pattern matching functions evaluated at every system call.*

---

## 1. Profile as a Permission Function

### Profile Model

An AppArmor profile defines a function from paths to permissions:

$$\text{profile}(p) : \text{Path} \rightarrow \mathcal{P}(\{\text{r, w, x, m, k, l, ix, px, ux, cx}\})$$

Where $\mathcal{P}$ denotes the power set (all subsets).

| Permission | Meaning | Risk Level |
|:---|:---|:---:|
| r | Read | Low |
| w | Write | Medium |
| x (ix/px/ux/cx) | Execute (inherit/profile/unconfined/child) | High |
| m | Memory map executable | Medium |
| k | Lock (file locking) | Low |
| l | Link | Medium |

### Profile Modes

$$\text{Mode} = \begin{cases} \text{enforce} & \text{deny + log violations} \\ \text{complain} & \text{log only (learning mode)} \\ \text{unconfined} & \text{no restrictions (disabled)} \end{cases}$$

---

## 2. Glob Pattern Matching

### Pattern Language

AppArmor paths use glob-style patterns:

| Pattern | Matches | Expansion |
|:---|:---|:---|
| `/etc/passwd` | Exact file | 1 path |
| `/etc/*` | One level | All files in /etc |
| `/etc/**` | Recursive | All files under /etc |
| `/tmp/app-*.log` | Wildcard | app-1.log, app-foo.log, etc. |
| `/home/*/Documents/**` | Per-user recursive | All docs for all users |

### Pattern Specificity Ordering

More specific patterns take precedence:

$$\text{specificity}(p) = |\text{literal characters in } p| - |\text{wildcards in } p|$$

When multiple patterns match, the most specific wins. This creates a **partial order** on rules.

### Pattern Match Complexity

For a profile with $n$ rules and a path of length $L$:

$$T_{match} = O(n \times L) \text{ (linear scan with glob matching)}$$

AppArmor compiles profiles to a DFA (deterministic finite automaton), reducing runtime to:

$$T_{match}^{DFA} = O(L) \text{ (independent of rule count)}$$

---

## 3. Profile Compilation — DFA Construction

### From Globs to Automata

Each glob pattern is converted to a regular expression, then all patterns are merged into a single DFA:

$$\text{DFA states} = O(2^{n_{patterns}}) \text{ worst case}$$

In practice, AppArmor's DFA is much smaller due to pattern structure:

| Profiles | Patterns | DFA States | Compiled Size |
|:---:|:---:|:---:|:---:|
| 10 | 200 | ~5,000 | ~200 KB |
| 50 | 2,000 | ~50,000 | ~2 MB |
| 100 | 5,000 | ~120,000 | ~5 MB |
| 200 | 15,000 | ~400,000 | ~16 MB |

### Compilation Time

$$T_{compile} = O(n \times |\Sigma|^k)$$

Where $|\Sigma| = 256$ (byte alphabet) and $k$ depends on pattern complexity.

Large profiles (e.g., snap packages with deep paths) can take 10-30 seconds to compile.

### Runtime Performance

DFA match per syscall: ~0.5-2 microseconds regardless of profile size.

$$\text{Overhead per syscall} \approx 1 \mu s$$

At 100,000 syscalls/second: ~100 ms total CPU overhead per second (0.01% on modern hardware).

---

## 4. Capability Confinement

### Linux Capabilities as a Bitmask

AppArmor controls which Linux capabilities a confined process may use:

$$\text{cap\_mask} = \sum_{i=0}^{40} b_i \times 2^i$$

Where $b_i = 1$ if capability $i$ is allowed.

| CAP Value | Capability | Risk |
|:---:|:---|:---|
| 0 | CAP_CHOWN | Change file ownership |
| 1 | CAP_DAC_OVERRIDE | Bypass file permissions |
| 7 | CAP_SETUID | Set UID |
| 12 | CAP_NET_RAW | Raw sockets |
| 21 | CAP_SYS_ADMIN | Superpower (mount, bpf, etc.) |
| 25 | CAP_SYS_PTRACE | Trace processes |

### Capability Reduction

A well-confined application should need minimal capabilities:

| Application | Required Caps | Typical Default Caps |
|:---|:---:|:---:|
| Web server (nginx) | 2 (NET_BIND, DAC_OVERRIDE) | 41 (all) |
| DNS resolver | 1 (NET_BIND_SERVICE) | 41 |
| Container runtime | 14 | 41 |
| Unprivileged app | 0 | 41 |

$$\text{Cap reduction ratio} = 1 - \frac{|\text{allowed caps}|}{|\text{total caps}|} = 1 - \frac{2}{41} = 95.1\%$$

---

## 5. Network Access Control

### Network Rules

$$\text{net\_access} \subseteq \text{Family} \times \text{Type} \times \text{Direction}$$

| Family | Type | Permission |
|:---|:---|:---|
| inet (IPv4) | stream (TCP) | send, receive, connect, accept |
| inet6 (IPv6) | dgram (UDP) | bind, listen |
| unix | seqpacket | create, shutdown |

### Network Confinement Effectiveness

| Profile Rule | Blocked Attacks |
|:---|:---|
| `deny network raw` | Packet sniffing, ping floods |
| `deny network inet dgram` | DNS tunneling, UDP exfil |
| `network inet stream` (allow only) | Only TCP connections |
| `network unix` (deny inet) | No network access at all |

---

## 6. Stacking — Multiple Profiles

### Profile Intersection

When AppArmor stacking is used (e.g., in containers), multiple profiles apply simultaneously:

$$\text{effective}(p) = \text{profile}_1(p) \cap \text{profile}_2(p) \cap \cdots \cap \text{profile}_n(p)$$

The effective permissions are the **intersection** of all active profiles — the most restrictive combination.

### Container Isolation with Stacking

| Layer | Profile | Purpose |
|:---:|:---|:---|
| 1 | Container runtime default | Base container restrictions |
| 2 | Application-specific profile | App-level confinement |
| 3 | Snap/Flatpak profile | Package restrictions |

$$\text{Effective permissions} = P_{runtime} \cap P_{app} \cap P_{snap}$$

Each layer can only **remove** permissions, never add them.

---

## 7. Profile Generation — Statistical Learning

### aa-logprof Workflow

Learning mode (`complain`) records all accesses:

$$\text{Profile}_{learned} = \{(p_i, \text{perms}_i) : \text{access to } p_i \text{ observed during training}\}$$

### Coverage Problem

$$\text{Coverage} = \frac{|\text{paths exercised during training}|}{|\text{paths used in production}|}$$

| Training Duration | Typical Coverage | Risk of Deny in Production |
|:---:|:---:|:---:|
| 1 hour | 40-60% | High |
| 1 day | 70-85% | Medium |
| 1 week | 85-95% | Low |
| 1 month | 95-99% | Very low |

Rare code paths (error handlers, backup routines, leap-year logic) may only trigger after weeks or months — incomplete training leads to production denials.

### Profile Minimality

The optimal profile minimizes permissions while allowing all legitimate access:

$$\text{minimize} \sum_{i} |\text{perms}(p_i)| \quad \text{subject to: all legitimate accesses allowed}$$

This is a **set cover problem** when glob patterns are involved — NP-hard in general but tractable for typical profile sizes.

---

## 8. AppArmor vs SELinux — Quantitative Comparison

| Metric | AppArmor | SELinux |
|:---|:---:|:---:|
| Policy model | Path-based (glob) | Label-based (type enforcement) |
| Confinement granularity | File path | Security context |
| Profile compilation | DFA (seconds) | AV hash table (seconds) |
| Runtime overhead per check | ~1 $\mu$s | ~0.5 $\mu$s |
| Policy size (typical) | 2-16 MB | 2-15 MB |
| File rename handling | May break policy | Labels follow inode |
| Default policy coverage | Per-application | System-wide |
| Learning mode | complain (aa-logprof) | permissive (audit2allow) |

### The Rename Problem

If file `/etc/shadow` is renamed to `/tmp/shadow`:
- **AppArmor:** `/etc/shadow` rules no longer apply (path changed)
- **SELinux:** `shadow_t` label follows the inode (still protected)

This is the fundamental tradeoff: path-based is intuitive but fragile under rename; label-based is robust but complex.

---

## 9. Summary of Functions by Type

| Concept | Math Type | Application |
|:---|:---|:---|
| Profile function | Path → permission set | Access control |
| Glob matching | Regular language (DFA) | Pattern evaluation |
| $2^n$ DFA states | Exponential (worst case) | Profile compilation |
| Capability bitmask | Binary integer | Privilege restriction |
| Profile stacking $\cap$ | Set intersection | Container isolation |
| Coverage $\%$ | Statistical sampling | Training completeness |
| Set cover | Combinatorial optimization | Profile minimality |

---

*AppArmor transforms the question "should this program access this file?" into a DFA state transition — compiled once at profile load, evaluated in microseconds at every system call, enforcing the principle of least privilege through mathematical pattern matching.*
