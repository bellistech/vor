# The Mathematics of Container Escape — Isolation Boundaries and Privilege Geometry

> *Container isolation is built on Linux kernel primitives — namespaces, cgroups, capabilities, and seccomp filters. Each primitive defines a mathematical boundary; container escapes exploit the gaps, intersections, and misconfigurations in these boundaries. Understanding the formal models behind isolation reveals why certain combinations of privileges collapse the entire security model.*

---

## 1. Namespace Isolation (Set Theory)

### Namespace Model

Each container $C$ is defined by a tuple of namespace memberships:

$$C = (N_{\text{mnt}}, N_{\text{pid}}, N_{\text{net}}, N_{\text{ipc}}, N_{\text{uts}}, N_{\text{user}}, N_{\text{cgroup}}, N_{\text{time}})$$

The host operates in the root namespace tuple $C_{\text{host}} = (N_0, N_0, N_0, N_0, N_0, N_0, N_0, N_0)$.

### Isolation as Set Partitioning

For resource type $R$, namespace $N$ creates a partition:

$$R = R_{N_1} \sqcup R_{N_2} \sqcup \cdots \sqcup R_{N_k}, \quad R_{N_i} \cap R_{N_j} = \emptyset \text{ for } i \neq j$$

| Namespace | Isolated Resource | Kernel Object |
|:---|:---:|:---:|
| mount | Filesystem view | vfsmount tree |
| pid | Process IDs | pid_namespace |
| net | Network stack | net_namespace |
| ipc | IPC objects | ipc_namespace |
| uts | Hostname | uts_namespace |
| user | UID/GID mapping | user_namespace |
| cgroup | Cgroup hierarchy | cgroup_namespace |
| time | Boot/monotonic clocks | time_namespace |

### Namespace Escape Condition

Escape occurs when a process in container $C$ can access resources in $C_{\text{host}}$:

$$\exists R \in C_{\text{host}} : \text{process} \in C \text{ can read/write } R$$

This happens via:
- Shared namespaces: $N_i(C) = N_i(C_{\text{host}})$
- Namespace traversal: `/proc/<host_pid>/ns/*`
- Capability-mediated operations: `nsenter`, `setns()`

---

## 2. Capability Lattice (Partial Order)

### Linux Capability Model

Linux capabilities form a bounded lattice $(2^{\mathcal{C}}, \subseteq)$ where $\mathcal{C}$ is the set of all capabilities:

$$|\mathcal{C}| = 41 \text{ capabilities (as of Linux 6.x)}$$

Total possible capability sets:

$$|2^{\mathcal{C}}| = 2^{41} = 2{,}199{,}023{,}255{,}552$$

### Privilege Hierarchy

| Capability Set | Size | Escape Risk |
|:---|:---:|:---:|
| Default Docker | 14 caps | Low (restricted) |
| `--cap-add SYS_ADMIN` | 15 caps | Critical (cgroup escape) |
| `--cap-add ALL` | 41 caps | Critical (full escape) |
| `--privileged` | 41 caps + devices | Critical (trivial escape) |

### Critical Capability Combinations

Some escapes require specific capability combinations:

$$\text{cgroup\_escape} \Leftarrow \text{CAP\_SYS\_ADMIN}$$
$$\text{module\_load} \Leftarrow \text{CAP\_SYS\_MODULE}$$
$$\text{ptrace\_inject} \Leftarrow \text{CAP\_SYS\_PTRACE} \wedge \text{shared PID ns}$$
$$\text{net\_mitm} \Leftarrow \text{CAP\_NET\_ADMIN} \wedge \text{CAP\_NET\_RAW}$$

### Capability Bits Arithmetic

Effective capabilities are computed as:

$$P'(\text{eff}) = P'(\text{permitted}) \cap F(\text{inheritable})$$

For the full `--privileged` mode:

$$\text{CapEff} = \text{0x3fffffffff} = 2^{38} - 1 = 274{,}877{,}906{,}943$$

---

## 3. Cgroup Escape Mechanics (Control Flow)

### release_agent Execution Model

The cgroup `release_agent` mechanism triggers execution on the host when a cgroup becomes empty:

$$\text{notify\_on\_release} = 1 \wedge |\text{cgroup.procs}| \rightarrow 0 \implies \text{exec}(\text{release\_agent})$$

### Attack State Machine

```
S0: Container process (CAP_SYS_ADMIN)
  |
  v [mkdir cgroup]
S1: New cgroup created
  |
  v [set notify_on_release=1]
S2: Notification enabled
  |
  v [set release_agent=payload]
S3: Payload configured
  |
  v [echo $$ > cgroup.procs; exit]
S4: Cgroup becomes empty → host executes payload
  |
  v
S5: Code execution on host
```

### Conditions for Success

$$P(\text{escape}) = P(\text{CAP\_SYS\_ADMIN}) \times P(\text{cgroupfs\_writable}) \times P(\text{host\_path\_known})$$

In a `--privileged` container: all three conditions are met, so $P = 1$.

---

## 4. Seccomp Filter Complexity (Automata Theory)

### BPF Filter Model

Seccomp filters are BPF programs — a restricted deterministic finite automaton (DFA):

$$M = (Q, \Sigma, \delta, q_0, F)$$

Where:
- $Q$ = program states (up to 4096 instructions)
- $\Sigma$ = syscall arguments (system call number, arguments)
- $\delta$ = BPF instruction transitions
- $F$ = terminal states (ALLOW, KILL, ERRNO, TRAP, LOG)

### Filter Effectiveness

For $N$ total syscalls and $A$ allowed syscalls:

$$\text{Attack surface reduction} = 1 - \frac{|A|}{|N|} = 1 - \frac{|A|}{~450}$$

| Profile | Allowed Syscalls | Reduction |
|:---|:---:|:---:|
| Docker default | ~300 | 33% |
| Minimal web server | ~50 | 89% |
| Static binary | ~20 | 96% |
| `--privileged` | All (no filter) | 0% |

### Syscalls Critical for Escape

| Syscall | Purpose | Blocked by Default |
|:---|:---:|:---:|
| `mount` | Filesystem access | Yes |
| `unshare` | Namespace creation | Yes |
| `clone` (NEWUSER) | User namespace | Yes |
| `pivot_root` | Root filesystem change | Yes |
| `init_module` | Kernel module loading | Yes |
| `open_by_handle_at` | File handle access | Yes |
| `ptrace` | Process debugging | Partially |

---

## 5. Docker Socket Attack Surface (API Theory)

### Docker API as Attack Primitive

The Docker socket exposes a RESTful API. Each endpoint maps to a host operation:

$$\text{API endpoint} \xrightarrow{\text{Docker daemon}} \text{Host kernel operation}$$

### Privilege Amplification

Docker socket access provides privilege amplification factor:

$$\text{Amplification} = \frac{\text{Host privilege gained}}{\text{Container privilege held}}$$

| API Call | Effect | Amplification |
|:---|:---:|:---:|
| `POST /containers/create` (privileged) | Root on host | $\infty$ |
| `POST /containers/{id}/exec` | Command in any container | High |
| `GET /containers/{id}/logs` | Read any container logs | Medium |
| `POST /images/create` | Pull arbitrary images | Medium |
| `POST /volumes/create` | Create host-mounted volumes | High |

### Socket Exposure Probability

In real-world deployments:

| Configuration | Prevalence | Escape Difficulty |
|:---|:---:|:---:|
| Socket mounted (`-v /var/run/docker.sock`) | 15-25% | Trivial |
| TCP socket (2375/2376) | 5-10% | Trivial (if no TLS) |
| No socket exposure | 65-80% | Requires other vector |

---

## 6. Overlay Filesystem Attacks (Layer Theory)

### Union Mount Model

Container filesystems stack layers using overlay:

$$\text{View} = \text{Upper (RW)} \cup (\text{Lower}_n \cup \cdots \cup \text{Lower}_1)$$

File lookup priority: Upper $\succ$ Lower$_n$ $\succ \cdots \succ$ Lower$_1$.

### Overlay Escape Conditions

CVE-2021-3493 and CVE-2023-0386 exploit privilege confusion between layers:

$$\text{file} \in \text{Lower (nosuid)} \xrightarrow{\text{copy-up}} \text{Upper (suid preserved)}$$

The copy-up operation preserves setuid bits when it should strip them:

$$\text{Expected: } \text{mode}(\text{upper}) = \text{mode}(\text{lower}) \wedge \neg\text{S\_ISUID}$$
$$\text{Actual: } \text{mode}(\text{upper}) = \text{mode}(\text{lower}) \quad \text{(bug: SUID preserved)}$$

### Layer Count and Attack Surface

$$\text{Exposure} \propto \sum_{i=1}^{n} |\text{files}(\text{Layer}_i)| \times P(\text{vuln in Layer}_i)$$

| Base Image | Typical Layers | Files | Known CVEs (avg) |
|:---|:---:|:---:|:---:|
| Alpine | 1-3 | ~14,000 | 5-15 |
| Debian slim | 3-5 | ~30,000 | 20-50 |
| Ubuntu | 4-6 | ~50,000 | 30-80 |
| Full Ubuntu | 5-8 | ~90,000 | 50-150 |

---

## 7. Kubernetes Escape Paths (Multi-Stage)

### Pod Security Hierarchy

Kubernetes Pod Security Standards define three levels:

| Level | Capabilities | Namespaces | Escape Path |
|:---|:---:|:---:|:---:|
| Restricted | Drop ALL | Isolated | Kernel exploit only |
| Baseline | Default Docker set | Isolated | Limited vectors |
| Privileged | All / configurable | Shared possible | Multiple paths |

### RBAC Escalation Graph

Kubernetes RBAC forms a directed graph of permission escalation:

$$\text{ServiceAccount} \xrightarrow{\text{RoleBinding}} \text{Role} \xrightarrow{\text{grants}} \text{Permissions}$$

Critical escalation paths:

$$\text{pods/exec} \rightarrow \text{container shell} \rightarrow \text{node access}$$
$$\text{secrets/get} \rightarrow \text{service account tokens} \rightarrow \text{API impersonation}$$
$$\text{pods/create} + \text{privileged} \rightarrow \text{host filesystem} \rightarrow \text{node root}$$

### Attack Surface per Cluster Component

| Component | Default Port | Risk if Exposed |
|:---|:---:|:---:|
| API Server | 6443 | Full cluster control |
| Kubelet | 10250 | Node command execution |
| etcd | 2379 | All cluster secrets |
| kube-proxy | 10256 | Network manipulation |
| Dashboard | 8443 | Cluster admin (if RBAC weak) |

---

## 8. Defense-in-Depth Quantification (Risk Model)

### Layered Defense Model

Each defense layer reduces escape probability independently:

$$P(\text{escape}) = \prod_{i=1}^{n} P(\text{bypass layer}_i)$$

| Layer | Bypass Probability | Defense |
|:---|:---:|:---:|
| Unprivileged container | 0.01 | Drop capabilities |
| Read-only rootfs | 0.05 | `--read-only` |
| Seccomp filter | 0.02 | Custom profile |
| AppArmor/SELinux | 0.03 | Mandatory access control |
| User namespace | 0.01 | `--userns-remap` |
| No new privileges | 0.05 | `--security-opt=no-new-privileges` |
| Patched kernel | 0.005 | Regular updates |

Combined escape probability with all layers:

$$P = 0.01 \times 0.05 \times 0.02 \times 0.03 \times 0.01 \times 0.05 \times 0.005 = 3.75 \times 10^{-13}$$

Versus a `--privileged` container (effectively zero defense layers):

$$P(\text{escape} \mid \text{privileged}) \approx 1$$

---

*Container security is fundamentally the study of isolation boundary composition — how namespaces, capabilities, cgroups, seccomp, and filesystem layering interact to create (or fail to create) strong separation. Every container escape exploits a failure in one or more of these boundaries, and the mathematics show that defense-in-depth with independent layers compounds protection exponentially while a single misconfiguration like `--privileged` collapses all isolation to nothing.*

## Prerequisites

- Linux kernel fundamentals (namespaces, cgroups, capabilities, system calls)
- Set theory and partial orders (for understanding isolation models)
- Basic probability (independent events, conditional probability)

## Complexity

- **Beginner:** Identifying container environments, checking capabilities, understanding namespace types
- **Intermediate:** Exploiting Docker socket and cgroup release_agent, computing capability sets, evaluating seccomp filter coverage
- **Advanced:** Chaining overlay filesystem bugs with namespace confusion, building multi-stage Kubernetes escalation paths, and quantifying defense-in-depth effectiveness across all isolation primitives
