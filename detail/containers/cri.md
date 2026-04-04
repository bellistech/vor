# The Mathematics of CRI — Runtime Abstraction Layers and Pod Lifecycle Automata

> *The Container Runtime Interface is a formally defined gRPC service contract that maps Kubernetes pod semantics to OCI runtime operations through a layered abstraction hierarchy, where pod sandbox lifecycle follows a deterministic state machine with well-defined transition guards.*

---

## 1. CRI as Abstraction Layer (Category Theory)

### Morphism Between Domains

The CRI defines a functor from Kubernetes domain objects to OCI runtime primitives:

$$F: \textbf{K8s} \rightarrow \textbf{OCI}$$

| K8s Concept | CRI Mapping | OCI Primitive |
|:---|:---|:---|
| Pod | PodSandbox | Namespaces + cgroups |
| Container | Container | OCI config.json + rootfs |
| Image | Image | OCI Image Manifest |
| Volume | Mount | Bind mount / tmpfs |
| ResourceQuota | Resources | Cgroup limits |

The composition:

$$\text{kubelet} \xrightarrow{F_{\text{CRI}}} \text{containerd/CRI-O} \xrightarrow{F_{\text{OCI}}} \text{runc/kata/gVisor}$$

### Interface Cardinality

The CRI RuntimeService defines 18 RPCs and ImageService defines 5 RPCs:

$$|CRI_{\text{RPCs}}| = 23$$

Each RPC maps a high-level operation to potentially many low-level syscalls:

$$\text{fan-out}(\text{RunPodSandbox}) \approx 50\text{-}200 \text{ syscalls}$$

$$\text{fan-out}(\text{CreateContainer}) \approx 30\text{-}100 \text{ syscalls}$$

---

## 2. Pod Sandbox State Machine (Automata Theory)

### Formal Definition

The pod sandbox lifecycle is a deterministic finite automaton:

$$M = (Q, \Sigma, \delta, q_0, F)$$

Where:
- $Q = \{\text{INIT}, \text{READY}, \text{NOT\_READY}, \text{REMOVED}\}$
- $\Sigma = \{\text{run}, \text{stop}, \text{remove}, \text{error}\}$
- $q_0 = \text{INIT}$
- $F = \{\text{REMOVED}\}$

Transition function:

$$\delta(\text{INIT}, \text{run}) = \text{READY}$$
$$\delta(\text{READY}, \text{stop}) = \text{NOT\_READY}$$
$$\delta(\text{NOT\_READY}, \text{remove}) = \text{REMOVED}$$
$$\delta(\text{READY}, \text{error}) = \text{NOT\_READY}$$
$$\delta(\text{INIT}, \text{error}) = \text{NOT\_READY}$$

### Container State Machine (Nested)

Each container within a pod has its own automaton:

$$M_c = (Q_c, \Sigma_c, \delta_c, q_{c0}, F_c)$$

$$Q_c = \{\text{CREATED}, \text{RUNNING}, \text{EXITED}, \text{REMOVED}, \text{UNKNOWN}\}$$

State dependency: container states are bounded by pod sandbox state:

$$\text{state}(c) = \text{RUNNING} \implies \text{state}(\text{pod}(c)) = \text{READY}$$

### Composite State Space

For a pod with $n$ containers, the composite state space:

$$|S_{\text{pod}}| = |Q| \times |Q_c|^n = 4 \times 5^n$$

For a typical pod with 2 containers:

$$|S| = 4 \times 25 = 100 \text{ possible states}$$

With constraints (container cannot run if pod not ready):

$$|S_{\text{valid}}| \ll 100$$

---

## 3. Shim Process Tree (Graph Theory)

### Containerd v2 Shim Model

The process tree forms a forest (collection of trees):

$$T = (V, E) \text{ where } V = \text{processes}, E = \text{parent-child}$$

```
containerd (PID 1234)
├── shim-runc-v2 (PID 2001) ── Pod A
│   ├── pause (PID 2010)
│   ├── container-1 (PID 2011)
│   └── container-2 (PID 2012)
├── shim-runc-v2 (PID 2002) ── Pod B
│   ├── pause (PID 2020)
│   └── container-3 (PID 2021)
```

Process count per node:

$$P_{\text{total}} = 1_{\text{containerd}} + \sum_{i=1}^{p} \left(1_{\text{shim}_i} + 1_{\text{pause}_i} + n_i\right)$$

Where $p$ = number of pods, $n_i$ = containers in pod $i$.

### v1 vs v2 Shim Overhead

| Model | Shim Processes | Memory Overhead |
|:---|:---:|:---|
| v1 (per-container) | $\sum n_i$ | $\sim 10\text{MB} \times \sum n_i$ |
| v2 (per-pod) | $p$ | $\sim 10\text{MB} \times p$ |
| Savings | $\sum n_i - p$ | $\sim 10\text{MB} \times (\sum n_i - p)$ |

For a node with 100 pods, average 3 containers each:

$$\text{v1}: 300 \text{ shims} \times 10\text{MB} = 3\text{GB}$$
$$\text{v2}: 100 \text{ shims} \times 10\text{MB} = 1\text{GB}$$
$$\text{savings} = 2\text{GB}$$

---

## 4. Image Pull as Content-Addressed DAG (Graph Theory)

### Layer Deduplication

An OCI image is a directed acyclic graph of content-addressed blobs:

$$G_{\text{image}} = (B, D)$$

Where $B$ = blobs (layers, config, manifest) and $D$ = digest references.

Two images sharing base layers:

$$\text{shared}(I_1, I_2) = L(I_1) \cap L(I_2)$$

Storage savings from deduplication:

$$\text{savings} = \sum_{l \in \text{shared}} \text{size}(l) \times (\text{refcount}(l) - 1)$$

### Pull Bandwidth

For an image with $n$ layers, total pull size:

$$S_{\text{pull}} = \sum_{i=1}^{n} \text{size}(l_i) \times \mathbb{1}[l_i \notin \text{local cache}]$$

With layer cache hit rate $h$:

$$E[S_{\text{pull}}] = (1 - h) \times \sum_{i=1}^{n} \text{size}(l_i)$$

Typical cache hit rates on warm nodes: $h \approx 0.7\text{-}0.9$.

---

## 5. gRPC Channel Performance (Queueing Theory)

### CRI Request Latency

The CRI gRPC channel acts as a single-server queue. For Poisson arrivals with rate $\lambda$ and exponential service time with rate $\mu$:

$$\text{M/M/1}: \quad W = \frac{1}{\mu - \lambda}$$

### Latency Breakdown by Operation

| CRI Operation | Typical Latency | Bottleneck |
|:---|:---:|:---|
| RunPodSandbox | 200-500ms | Network namespace + CNI |
| CreateContainer | 50-200ms | Rootfs setup |
| StartContainer | 100-300ms | Process creation |
| PullImage | 1-60s | Network I/O |
| StopContainer | 0-30s | Graceful shutdown |
| RemoveContainer | 10-50ms | Filesystem cleanup |

### Throughput Limit

Maximum pod creation rate limited by RunPodSandbox latency:

$$\lambda_{\max} = \frac{1}{T_{\text{RunPodSandbox}}} \approx \frac{1}{0.3\text{s}} \approx 3.3 \text{ pods/s}$$

With parallelism $k$ (kubelet workers):

$$\lambda_{\max} = \frac{k}{T_{\text{RunPodSandbox}}} = \frac{8}{0.3} \approx 26.7 \text{ pods/s}$$

---

## 6. RuntimeClass Selection (Decision Theory)

### Runtime Decision Matrix

RuntimeClass maps workload requirements to runtimes:

$$R: \text{Workload} \rightarrow \text{Runtime}$$

| Criterion | runc | gVisor | Kata | Score Formula |
|:---|:---:|:---:|:---:|:---|
| Startup time | 50ms | 200ms | 2s | $s_t = 1/t$ |
| Memory overhead | 2MB | 50MB | 256MB | $s_m = 1/m$ |
| Syscall overhead | 0% | 20-50% | 5-10% | $s_p = 1/(1+o)$ |
| Isolation strength | Medium | High | Very High | $s_i$ = ordinal |
| Compatibility | 100% | ~95% | ~98% | $s_c$ = fraction |

### Optimization

For a workload with weight vector $\vec{w} = (w_t, w_m, w_p, w_i, w_c)$:

$$\text{score}(r) = \vec{w} \cdot \vec{s}(r)$$

$$r^* = \arg\max_r \text{score}(r)$$

Typical mappings:
- Trusted workloads: $r^* = \text{runc}$ (maximize performance)
- Untrusted code: $r^* = \text{gVisor}$ (balance isolation/performance)
- Multi-tenant: $r^* = \text{Kata}$ (maximize isolation)

---

## 7. Pod Scheduling and Runtime Binding (Constraint Satisfaction)

### Constraint Model

Pod placement with RuntimeClass is a constraint satisfaction problem:

$$\text{find } n \in N \text{ such that:}$$

$$\text{RuntimeClass}(\text{pod}) \in \text{installed\_runtimes}(n)$$
$$\text{resources}(\text{pod}) \leq \text{available}(n)$$
$$\text{nodeSelector}(\text{pod}) \subseteq \text{labels}(n)$$
$$\text{tolerations}(\text{pod}) \supseteq \text{taints}(n)$$

### Overhead Accounting

RuntimeClass specifies resource overhead:

```yaml
overhead:
  podFixed:
    cpu: "250m"
    memory: "120Mi"
```

Total resource request for scheduling:

$$R_{\text{total}} = R_{\text{containers}} + R_{\text{overhead}}$$

$$R_{\text{total}}^{\text{cpu}} = \sum_{c \in \text{containers}} R_c^{\text{cpu}} + R_{\text{overhead}}^{\text{cpu}}$$

For Kata containers with 256Mi overhead on a 4Gi node:

$$\text{max pods} = \left\lfloor \frac{4096\text{Mi} - \text{system reserved}}{R_{\text{pod}} + 256\text{Mi}} \right\rfloor$$

---

## Prerequisites

category-theory, finite-automata, graph-theory, queueing-theory, optimization

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| Pod state transition | $O(1)$ — single state update | $O(1)$ |
| Container create (runc) | $O(L)$ — L = layers | $O(S)$ — S = rootfs size |
| Image pull (cached) | $O(1)$ — manifest check | $O(1)$ |
| Image pull (uncached) | $O(S)$ — S = image size | $O(S)$ |
| Layer deduplication check | $O(1)$ — digest lookup | $O(n)$ — n = stored layers |
| RuntimeClass scheduling | $O(N)$ — N = nodes | $O(R)$ — R = runtimes |
