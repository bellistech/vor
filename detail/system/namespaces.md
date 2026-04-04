# The Mathematics of Namespaces -- Set Partitioning and Resource Isolation

> *Namespaces partition the global kernel resource space into disjoint subsets,*
> *each providing processes with an independent view of system resources.*
> *This is fundamentally a problem of set theory, bijective mappings, and equivalence classes.*

---

## 1. Namespace Partitioning as Set Theory (Set Theory)

### The Problem

A namespace divides a global resource set $G$ into disjoint subsets such that processes within a subset see only their local partition. How do we formalize this?

### The Formula

Let $G$ be the global set of resources (PIDs, network interfaces, mount points). A namespace configuration $N$ partitions $G$:

$$G = N_1 \cup N_2 \cup \cdots \cup N_k, \quad N_i \cap N_j = \emptyset \; \forall \; i \neq j$$

Each process $p$ has a namespace vector:

$$\vec{N}(p) = \langle N_{\text{pid}}, N_{\text{net}}, N_{\text{mnt}}, N_{\text{user}}, N_{\text{ipc}}, N_{\text{uts}}, N_{\text{cgroup}}, N_{\text{time}} \rangle$$

Two processes $p, q$ share a resource view if and only if:

$$\text{visible}(p, q, \text{type}) \iff N_{\text{type}}(p) = N_{\text{type}}(q)$$

### Worked Examples

| Process | PID NS | NET NS | MNT NS | Isolation Level |
|---------|--------|--------|--------|-----------------|
| host-init | NS_pid_0 | NS_net_0 | NS_mnt_0 | None (root) |
| container-A | NS_pid_1 | NS_net_1 | NS_mnt_1 | Full |
| container-B | NS_pid_2 | NS_net_2 | NS_mnt_1 | Shared mounts |
| sidecar-A | NS_pid_1 | NS_net_1 | NS_mnt_2 | Shared PID+NET |

Container-A and sidecar-A share PID and NET namespaces (like Kubernetes pod model):

$$N_{\text{pid}}(\text{A}) = N_{\text{pid}}(\text{sidecar}) \implies \text{ps in sidecar sees A's processes}$$

## 2. PID Mapping Functions (Bijective Mappings)

### The Problem

PID namespaces create a hierarchy where each process has a different PID in each ancestor namespace. How does the mapping work?

### The Formula

For a process $p$ at nesting depth $d$, the PID mapping function from child namespace $C$ to parent namespace $P$:

$$f_{C \to P}: \text{PID}_C \to \text{PID}_P$$

This is an injective (one-to-one) function:

$$f_{C \to P}(\text{pid}_c) = \text{pid}_p, \quad \text{pid}_p \neq f_{C \to P}(\text{pid}_c') \; \forall \; c \neq c'$$

The reverse mapping does not exist for processes outside $C$ (isolation property):

$$f_{P \to C}(\text{pid}_p) = \begin{cases}
\text{pid}_c & \text{if } p \in C \\
\text{undefined} & \text{if } p \notin C
\end{cases}$$

### Worked Examples

A three-level PID namespace hierarchy:

```
Host (NS_0)          Container (NS_1)      Nested (NS_2)
  PID 1 (init)
  PID 1234 ──────────> PID 1 (container init)
  PID 1235 ──────────> PID 2 ─────────────> PID 1 (nested init)
  PID 1236 ──────────> PID 3 ─────────────> PID 2
```

| Process | PID in NS_0 | PID in NS_1 | PID in NS_2 |
|---------|------------|------------|------------|
| host-init | 1 | invisible | invisible |
| container-init | 1234 | 1 | invisible |
| nested-init | 1235 | 2 | 1 |
| nested-worker | 1236 | 3 | 2 |

Visibility matrix (can row see column's processes?):

| | NS_0 | NS_1 | NS_2 |
|------|------|------|------|
| NS_0 | yes | yes | yes |
| NS_1 | no | yes | yes |
| NS_2 | no | no | yes |

This forms a strict partial order: $NS_2 \subset NS_1 \subset NS_0$.

## 3. User Namespace UID Mapping (Affine Transformations)

### The Problem

User namespaces map ranges of UIDs from the parent namespace into the child. This mapping is an affine function over integer intervals.

### The Formula

A UID map entry `inside_start outside_start count` defines:

$$f(\text{uid}_{\text{inside}}) = \text{uid}_{\text{inside}} - \text{inside\_start} + \text{outside\_start}$$

$$\text{domain}: [\text{inside\_start}, \; \text{inside\_start} + \text{count} - 1]$$

$$\text{range}: [\text{outside\_start}, \; \text{outside\_start} + \text{count} - 1]$$

### Worked Examples

Typical rootless container UID map:

```
# /proc/$PID/uid_map
# inside_start  outside_start  count
         0          1000          1
         1        100000      65536
```

| Inside UID | Mapping | Outside UID |
|-----------|---------|------------|
| 0 (root) | 0 - 0 + 1000 | 1000 |
| 1 | 1 - 1 + 100000 | 100000 |
| 100 | 100 - 1 + 100000 | 100099 |
| 65535 | 65535 - 1 + 100000 | 165534 |
| 65536 | unmapped | EOVERFLOW |

Total mapped UIDs: $1 + 65536 = 65537$

Subordinate UID allocation for multiple containers:

$$\text{container}_i: [\text{base} + i \times 65536, \; \text{base} + (i+1) \times 65536 - 1]$$

| Container | Inside Range | Outside Range |
|-----------|-------------|--------------|
| 0 | 1 -- 65536 | 100000 -- 165535 |
| 1 | 1 -- 65536 | 165536 -- 231071 |
| 2 | 1 -- 65536 | 231072 -- 296607 |

## 4. Network Namespace Isolation (Graph Theory)

### The Problem

Network namespaces create isolated network stacks. Connectivity between namespaces requires explicit virtual links (veth pairs, bridges). This forms a graph.

### The Formula

Model the system as graph $G = (V, E)$ where:
- $V$ = set of network namespaces
- $E$ = set of veth pairs or bridge connections

Two namespaces $N_i, N_j$ can communicate if and only if there exists a path:

$$\text{connected}(N_i, N_j) \iff \exists \; \text{path}(N_i, N_j) \in G$$

Complete isolation: $G$ has no edges (each namespace is a disconnected vertex).

### Worked Examples

Docker bridge network topology:

```
Namespace Graph:
  host ─── docker0 (bridge)
              ├── veth_a ─── container_A
              ├── veth_b ─── container_B
              └── veth_c ─── container_C
```

Adjacency matrix:

| | host | A | B | C |
|------|------|---|---|---|
| host | 0 | 1 | 1 | 1 |
| A | 1 | 0 | 1 | 1 |
| B | 1 | 1 | 0 | 1 |
| C | 1 | 1 | 1 | 0 |

The bridge creates a fully connected subgraph. Network policies (iptables rules) prune edges:

$$G' = G \setminus \{(A, B)\} \implies A \text{ and } B \text{ cannot communicate directly}$$

## 5. Mount Namespace Propagation (Lattice Theory)

### The Problem

Mount propagation types form a hierarchy controlling how mount events flow between namespaces.

### The Formula

Mount propagation defines a partial order on namespace relationships:

$$\text{shared} \succ \text{slave} \succ \text{private} \succ \text{unbindable}$$

Propagation flow function:

$$\text{propagate}(m, N_{\text{src}}, N_{\text{dst}}) = \begin{cases}
\text{yes} & \text{if shared-shared or shared-slave (src}\to\text{dst)} \\
\text{no} & \text{if private or unbindable} \\
\text{no} & \text{if slave-shared (dst}\to\text{src)}
\end{cases}$$

### Worked Examples

| Source Type | Dest Type | Src->Dst | Dst->Src |
|------------|----------|----------|----------|
| shared | shared | yes | yes |
| shared | slave | yes | no |
| shared | private | no | no |
| slave | slave | no | no |
| private | private | no | no |

## 6. Isolation Completeness Proof (Security Analysis)

### The Problem

How many namespace types must be combined to achieve complete process isolation?

### The Formula

Define isolation score $I$ as the fraction of kernel resource categories isolated:

$$I = \frac{|\text{isolated namespaces}|}{|\text{total namespace types}|}$$

With 8 namespace types, complete isolation requires:

$$I = \frac{8}{8} = 1.0$$

### Worked Examples

| Configuration | Namespaces | Score | Security Level |
|--------------|-----------|-------|----------------|
| `unshare --pid` | 1/8 | 0.125 | Minimal |
| `unshare --pid --net --mount` | 3/8 | 0.375 | Moderate |
| Docker default | 6/8 | 0.750 | Good |
| Docker + userns | 7/8 | 0.875 | Strong |
| Full isolation | 8/8 | 1.000 | Maximum |

Docker default namespaces: PID, NET, MNT, UTS, IPC, Cgroup (no USER, no Time).

Attack surface area decreases exponentially with additional namespaces:

$$\text{attack\_surface} \propto \frac{1}{2^{|\text{namespaces}|}}$$

## 7. Namespace Creation Cost (Performance)

### The Problem

Each namespace type has different creation and memory overhead. How do we estimate the cost of namespace-heavy architectures?

### The Formula

Total overhead for $C$ containers each with $n$ namespace types:

$$\text{memory}_{\text{total}} = C \times \sum_{i=1}^{n} m_i$$

where $m_i$ is the per-namespace memory cost.

### Worked Examples

Approximate per-namespace memory overhead:

| Namespace | Memory Overhead | Creation Time |
|-----------|----------------|---------------|
| PID | ~4 KB | ~2 us |
| NET | ~80 KB (full stack) | ~100 us |
| MNT | ~8 KB (mount table) | ~5 us |
| USER | ~4 KB | ~2 us |
| IPC | ~4 KB | ~2 us |
| UTS | ~1 KB | ~1 us |
| Cgroup | ~4 KB | ~2 us |
| **Total** | **~105 KB** | **~114 us** |

For 1000 containers with full namespace isolation:

$$\text{memory} = 1000 \times 105 \, \text{KB} \approx 105 \, \text{MB}$$

$$\text{creation time} = 1000 \times 114 \, \mu s \approx 114 \, \text{ms}$$

Compare to VM overhead (~256 MB each): namespaces are 2500x more memory-efficient.

## Prerequisites

set-theory, graph-theory, linux-kernel-basics, process-management, networking-fundamentals

## Complexity

| Operation | Time | Space |
|-----------|------|-------|
| Create namespace (unshare) | O(1) | O(1) per NS type |
| Enter namespace (setns) | O(1) | O(1) |
| PID translation (child to parent) | O(depth) | O(1) |
| UID map lookup | O(n) map entries | O(1) |
| Veth pair creation | O(1) | O(1) per pair |
| Mount propagation event | O(slaves) | O(1) per mount |
| Namespace destruction | O(resources) | O(1) |
| lsns enumeration | O(processes) | O(namespaces) |
