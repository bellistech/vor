# The Mathematics of Ansible — Configuration Management Theory

> *Ansible applies desired state to infrastructure through SSH, using idempotent operations, fact-based set algebra, and parallel execution strategies. Its mathematical foundations are in function theory, graph execution, and connection multiplexing.*

---

## 1. Idempotency (The Core Axiom)

### The Problem

Every Ansible module must be **idempotent** — applying the same operation twice produces the same result as applying it once.

### The Formal Definition

$$f(f(x)) = f(x)$$

Where $f$ = module execution, $x$ = system state. Running a playbook once or ten times must produce identical final state.

### Idempotency in Practice

| Module | Operation | First Run | Second Run | Idempotent? |
|:---|:---|:---|:---|:---:|
| `copy` | Copy file | Changed | OK (no change) | Yes |
| `user` | Create user | Changed | OK (exists) | Yes |
| `command` | Run script | Changed | Changed (always) | **No** |
| `shell` | Run shell cmd | Changed | Changed (always) | **No** |
| `apt` | Install pkg | Changed | OK (installed) | Yes |

### The `creates` Guard (Making Non-Idempotent Modules Safe)

$$f_{guarded}(x) = \begin{cases}
f(x) & \text{if } \text{creates\_path} \notin \text{filesystem} \\
x & \text{if } \text{creates\_path} \in \text{filesystem}
\end{cases}$$

This transforms a non-idempotent operation into an idempotent one.

### Changed Count as Convergence Metric

$$\text{Converged} \iff \text{changed} = 0$$

A fully converged system reports zero changes on every run. This is the operational definition of desired state.

---

## 2. Execution Strategies (Parallelism Models)

### The Problem

Ansible supports different execution strategies that control how tasks are distributed across hosts.

### Linear Strategy (Default)

All hosts execute task $k$ before any host starts task $k+1$:

$$T_{linear} = \sum_{k=1}^{K} \max_{h \in H} T_k(h)$$

The slowest host at each task becomes the bottleneck.

### Free Strategy

Each host proceeds independently through all tasks:

$$T_{free} = \max_{h \in H} \sum_{k=1}^{K} T_k(h)$$

Total time = the slowest host's total, not the sum of per-task maxima.

### Comparison

For $K$ tasks, $N$ hosts with varying speeds:

| Strategy | Parallelism | Total Time | Use Case |
|:---|:---:|:---|:---|
| linear | Per-task | $\sum \max T_k$ | Rolling updates |
| free | Per-host | $\max \sum T_k$ | Independent config |
| host_pinned | Per-host (ordered) | $\max \sum T_k$ | Ordered convergence |

### Worked Example

3 hosts, 3 tasks. Task times in seconds:

| | Task 1 | Task 2 | Task 3 | Host Total |
|:---|:---:|:---:|:---:|:---:|
| Host A | 2 | 5 | 1 | 8 |
| Host B | 3 | 2 | 4 | 9 |
| Host C | 1 | 8 | 2 | 11 |
| **Max per task** | **3** | **8** | **4** | |

$$T_{linear} = 3 + 8 + 4 = 15 \text{ seconds}$$
$$T_{free} = \max(8, 9, 11) = 11 \text{ seconds}$$
$$\text{Speedup} = \frac{15}{11} = 1.36\times$$

---

## 3. SSH Connection Multiplexing

### The Problem

Each task requires an SSH connection. Without multiplexing, the overhead is enormous.

### Connection Cost Without Multiplexing

$$T_{connections} = K \times N \times T_{ssh\_handshake}$$

Where $T_{ssh\_handshake} \approx 200\text{-}500$ ms (TCP + key exchange + auth).

For 20 tasks on 50 hosts: $T = 20 \times 50 \times 0.3 = 300$ seconds of pure connection overhead.

### Connection Multiplexing (ControlMaster)

With `ssh_args = -o ControlMaster=auto -o ControlPersist=60s`:

$$T_{connections} = N \times T_{ssh\_handshake} + (K-1) \times N \times T_{mux}$$

Where $T_{mux} \approx 1\text{-}5$ ms (reuse existing socket).

$$\text{Savings} = 1 - \frac{N \times T_{handshake} + (K-1) \times N \times T_{mux}}{K \times N \times T_{handshake}}$$

For same example: $T = 50 \times 0.3 + 19 \times 50 \times 0.003 = 15 + 2.85 = 17.85$ seconds.

$$\text{Speedup} = \frac{300}{17.85} = 16.8\times$$

### Pipelining

With `pipelining = True`, Ansible sends module code over the existing SSH connection instead of SCP:

$$T_{task} = T_{send} + T_{execute} \quad \text{(no extra SSH round-trip for file transfer)}$$

Saves approximately 1 round-trip per task.

---

## 4. Fact Gathering as Set Operations

### The Problem

Ansible gathers facts about each host, creating a key-value universe. Inventory groups and host patterns operate as set algebra.

### Group Membership

$$\text{hosts}(G) = \{h : h \in G\}$$

### Pattern Operations

| Pattern | Set Operation | Meaning |
|:---|:---|:---|
| `webservers` | $G_{web}$ | All hosts in webservers |
| `webservers:dbservers` | $G_{web} \cup G_{db}$ | Union |
| `webservers:&staging` | $G_{web} \cap G_{staging}$ | Intersection |
| `webservers:!phoenix` | $G_{web} \setminus G_{phoenix}$ | Difference |
| `all:!windows` | $U \setminus G_{windows}$ | Complement |

### Worked Example

$$G_{web} = \{a, b, c, d\}, \quad G_{db} = \{c, d, e\}, \quad G_{staging} = \{b, d, e, f\}$$

| Pattern | Result | Count |
|:---|:---|:---:|
| `webservers:dbservers` | $\{a, b, c, d, e\}$ | 5 |
| `webservers:&staging` | $\{b, d\}$ | 2 |
| `webservers:!dbservers` | $\{a, b\}$ | 2 |
| `all` | $\{a, b, c, d, e, f\}$ | 6 |

---

## 5. Variable Precedence (22-Level Hierarchy)

### The Problem

Ansible has 22 levels of variable precedence. This is the most complex merge hierarchy of any configuration management tool.

### Precedence (Lowest to Highest)

| Level | Source | Overrides |
|:---:|:---|:---|
| 1 | command line values (not vars) | — |
| 2 | role defaults (`defaults/main.yml`) | Level 1 |
| 3 | inventory file/script group vars | Level 2 |
| 4 | inventory `group_vars/all` | Level 3 |
| 5 | playbook `group_vars/all` | Level 4 |
| 6 | inventory `group_vars/*` | Level 5 |
| 7 | playbook `group_vars/*` | Level 6 |
| ... | ... | ... |
| 20 | `set_fact` / `register` | Level 19 |
| 21 | role params (include_role) | Level 20 |
| 22 | extra vars (`-e`) | **Always wins** |

### The Rule

$$V_{final}(key) = V_{\max(level)}(key)$$

Extra vars (`-e`) are level 22 — they always win. This is an architectural guarantee.

---

## 6. Forks and Serial (Batch Processing)

### The Problem

`forks` controls SSH parallelism. `serial` controls rolling update batch size.

### Fork Saturation

$$\text{Effective parallelism} = \min(\text{forks}, |H|)$$

Default forks = 5. For 100 hosts:

$$\text{Batches} = \lceil 100 / 5 \rceil = 20 \text{ batches per task}$$

### Serial Rolling Updates

$$\text{Batches} = \lceil |H| / \text{serial} \rceil$$

$$T_{rolling} = \text{Batches} \times T_{playbook\_per\_batch}$$

| Hosts | Serial | Batches | Downtime per Batch |
|:---:|:---:|:---:|:---|
| 100 | 10 | 10 | 10% capacity loss |
| 100 | 25 | 4 | 25% capacity loss |
| 100 | 50 | 2 | 50% capacity loss |
| 100 | 1 | 100 | 1% (safest) |

### max_fail_percentage

$$\text{fail\_threshold} = \lfloor |H_{batch}| \times \frac{\text{max\_fail\_pct}}{100} \rfloor$$

For 10-host batch with `max_fail_percentage: 30`:

$$\text{fail\_threshold} = \lfloor 10 \times 0.3 \rfloor = 3$$

If 4+ hosts fail in a batch, the play aborts.

---

## 7. Role Dependency Graph

### The Model

Role dependencies form a DAG:

$$G = (R, D) \quad \text{where } R = \text{roles}, D = \text{dependency edges}$$

### Deduplication

By default, a role runs only once even if depended on by multiple roles:

$$\text{executions}(r) = 1 \quad \text{(unless } \texttt{allow\_duplicates: true}\text{)}$$

### Execution Order

$$\text{order} = \text{toposort}(G) \quad \text{(dependencies before dependents)}$$

---

## 8. Summary of Functions by Type

| Formula | Math Type | Domain |
|:---|:---|:---|
| $f(f(x)) = f(x)$ | Idempotency | Module design |
| $\sum \max T_k$ vs $\max \sum T_k$ | Min-max optimization | Strategy selection |
| $G_1 \cup G_2$, $G_1 \cap G_2$, $G_1 \setminus G_2$ | Set algebra | Host patterns |
| $N \times T_{handshake}$ vs $N \times T_{mux}$ | Linear cost | SSH multiplexing |
| $\lceil H / \text{serial} \rceil$ | Ceiling division | Rolling updates |
| $V_{\max(level)}$ | Priority ordering | Variable precedence |

---

*Ansible's power is in its simplicity — SSH, idempotent modules, and YAML. But underneath, it's applying set theory to inventory, function theory to convergence, and graph algorithms to role dependencies.*

## Prerequisites

- SSH key-based authentication
- Linux system administration (packages, services, files)
- YAML syntax
- Basic Python (for custom modules and filters)

## Complexity

- Beginner: ad-hoc commands, simple playbooks, static inventory
- Intermediate: roles, Jinja2 templates, vault, dynamic inventory, handlers
- Advanced: parallel execution tuning, connection multiplexing, custom modules, idempotency proofs
