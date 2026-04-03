# The Mathematics of Podman — Daemonless Container Internals

> *Podman runs containers without a daemon — each container is a direct child of the calling process. This architecture eliminates a single point of failure and enables rootless containers through user namespace UID mapping.*

---

## 1. Daemonless Architecture (Process Model)

### The Problem

Docker routes all operations through a central daemon. Podman eliminates this — containers are direct children of the calling process. What are the implications?

### Process Tree Comparison

**Docker:**

$$\text{User} \rightarrow \text{docker CLI} \rightarrow \text{dockerd} \rightarrow \text{containerd} \rightarrow \text{shim} \rightarrow \text{container}$$

$$\text{Depth} = 5, \quad \text{SPOF} = \text{dockerd}$$

**Podman:**

$$\text{User} \rightarrow \text{podman} \rightarrow \text{conmon} \rightarrow \text{container}$$

$$\text{Depth} = 3, \quad \text{SPOF} = \text{none}$$

### Process Count Formula

For $N$ containers:

| Runtime | Daemon Processes | Per-Container | Total |
|:---|:---:|:---:|:---:|
| Docker | 2 (dockerd + containerd) | 1 (shim) | $2 + N$ |
| Podman | 0 | 1 (conmon) | $N$ |

### Reliability Model

Mean time between failures for the daemon model:

$$\text{MTBF}_{docker} = \min(\text{MTBF}_{dockerd}, \text{MTBF}_{containerd})$$

$$\text{MTBF}_{podman} = \text{MTBF}_{conmon} \quad \text{(per container, independent)}$$

If dockerd crashes, all container management stops. If one conmon crashes, only that container is affected:

$$\text{Blast radius}_{docker} = N \quad \text{(all containers)}$$
$$\text{Blast radius}_{podman} = 1 \quad \text{(one container)}$$

---

## 2. User Namespace UID Mapping (Rootless Containers)

### The Problem

Rootless Podman maps container UIDs to unprivileged host UIDs. This mapping is defined in `/etc/subuid`.

### The Mapping Function

$$\text{UID}_{host} = \text{UID}_{container} + \text{offset}$$

For user `alice` with subuid entry `alice:100000:65536`:

$$f(\text{UID}_{container}) = \text{UID}_{container} + 100{,}000$$

| Container UID | Host UID | Identity |
|:---:|:---:|:---|
| 0 (root) | 100,000 | Unprivileged on host |
| 1 (daemon) | 100,001 | Unprivileged on host |
| 1000 (user) | 101,000 | Unprivileged on host |
| 65535 | 165,535 | Last mapped UID |

### Security Boundary

$$\text{Container root (UID 0)} \rightarrow \text{Host UID 100,000} \rightarrow \text{NO host privileges}$$

Even if an attacker escapes the container as root, they land as an unprivileged user on the host.

### Subordinate ID Range Capacity

$$C_{containers} = \frac{\text{subuid range}}{65{,}536}$$

With default range of 65,536 UIDs: $C = 1$ non-overlapping container UID space.

For multiple non-overlapping ranges:

$$\text{subuid}: \text{alice:100000:655360} \implies C = \frac{655{,}360}{65{,}536} = 10 \text{ containers}$$

---

## 3. Podman Pod Model (Shared Namespaces)

### The Problem

A Podman pod groups containers sharing network and IPC namespaces — the same model as Kubernetes pods.

### Namespace Sharing Matrix

| Namespace | Shared in Pod | Private per Container |
|:---|:---:|:---:|
| Network | Yes | No |
| IPC | Yes | No |
| PID | Optional | Default |
| Mount | No | Yes |
| UTS | Optional | Default |
| User | No | Yes |
| Cgroup | No | Yes |

### Pod Networking

All containers in a pod share one network namespace:

$$\text{IP}(c_1) = \text{IP}(c_2) = \cdots = \text{IP}(c_n) = \text{IP}_{pod}$$

Port conflicts within a pod:

$$\forall i \neq j: \text{ports}(c_i) \cap \text{ports}(c_j) = \emptyset \quad \text{(required)}$$

### Infra Container Overhead

Each pod has an infra container (like Kubernetes pause):

$$\text{Containers}_{total} = \sum_{p \in \text{pods}} (1 + C_p)$$

Where $C_p$ = user containers in pod $p$, and $+1$ = infra container.

Memory overhead of infra container: approximately 1 MB (it just calls `pause()`).

---

## 4. Image Storage (containers/storage)

### The Problem

Podman uses the containers/storage library with overlay or VFS drivers. How does storage scale?

### Overlay Layer Sharing

Identical to Docker's overlay2 model:

$$S_{total} = S_{shared\_layers} + \sum_{i=1}^{N} S_{writable_i}$$

### Rootless Storage Path

Rootless containers store images in `$HOME/.local/share/containers/storage/`, which means:

$$S_{user\_home} \geq S_{images} + S_{containers} + S_{volumes}$$

### Storage Comparison: Root vs Rootless

| Aspect | Root | Rootless |
|:---|:---|:---|
| Storage location | `/var/lib/containers/` | `~/.local/share/containers/` |
| Filesystem | Typically XFS/ext4 (dedicated) | User's home FS |
| Overlay support | Native | Requires fuse-overlayfs |
| Performance | Native kernel overlay | FUSE overhead: $1.2\text{-}2\times$ slower |
| Quota | Filesystem quota | User disk quota |

### FUSE Overhead (Rootless)

$$T_{rootless} = T_{root} \times k_{fuse}$$

Where $k_{fuse} \approx 1.2\text{-}2.0$ depending on operation:

| Operation | Root (overlay) | Rootless (fuse-overlayfs) | Ratio |
|:---|:---:|:---:|:---:|
| File create | 5 us | 8 us | 1.6x |
| File read (cached) | 0.5 us | 0.6 us | 1.2x |
| Directory listing (1000 files) | 200 us | 380 us | 1.9x |

---

## 5. Systemd Integration (Container Lifecycle)

### The Problem

Podman integrates with systemd for container auto-start and lifecycle management via `podman generate systemd`.

### Restart Policy as State Machine

$$\text{States} = \{\text{inactive}, \text{activating}, \text{active}, \text{failed}, \text{restarting}\}$$

Restart with backoff:

$$T_{restart}(n) = \min(T_{base} \times 2^n, T_{max})$$

Where $n$ = consecutive failure count.

| Failure # | Backoff (base=100ms, max=5min) |
|:---:|:---:|
| 1 | 200 ms |
| 2 | 400 ms |
| 3 | 800 ms |
| 5 | 3.2 s |
| 10 | 102.4 s |
| 15 | 300 s (capped) |

### Quadlet (Declarative Systemd Units)

Quadlet files in `~/.config/containers/systemd/` define containers declaratively. The dependency graph:

$$\text{After}=\text{network-online.target} \implies T_{start} \geq T_{network\_ready}$$

---

## 6. Networking Modes

### The Problem

Podman rootless networking uses slirp4netns or pasta for user-space network stack. Performance varies significantly.

### Network Stack Performance

$$\text{Throughput} = BW_{theoretical} \times \eta_{stack}$$

| Mode | Mechanism | $\eta$ (Efficiency) | Typical Throughput |
|:---|:---|:---:|:---:|
| Root bridge (CNI) | Kernel netfilter | 0.95 | 9.5 Gbps |
| Root host | No isolation | 1.0 | 10 Gbps |
| Rootless slirp4netns | Userspace TCP/IP | 0.10-0.30 | 1-3 Gbps |
| Rootless pasta | Userspace splice | 0.50-0.80 | 5-8 Gbps |

### Port Forwarding Overhead

Rootless port forwarding requires a userspace proxy:

$$T_{packet} = T_{kernel} + 2 \times T_{context\_switch} + T_{userspace\_copy}$$

The two context switches (kernel → user → kernel) add approximately 5-15 us per packet.

---

## 7. Compatibility Layer (Docker API Emulation)

### The API Socket Model

Podman emulates the Docker API via a socket:

$$\text{podman system service} \rightarrow \text{unix:///run/podman/podman.sock}$$

Compatibility coverage:

$$\text{Coverage} = \frac{|\text{Implemented endpoints}|}{|\text{Docker API endpoints}|} \approx 0.92$$

The 8% gap is mostly: Swarm endpoints, build cache introspection, and some plugin APIs.

---

## 8. Summary of Functions by Type

| Formula | Math Type | Domain |
|:---|:---|:---|
| $\text{UID}_{container} + \text{offset}$ | Linear mapping | User namespaces |
| Blast radius = 1 vs $N$ | Fault isolation | Architecture |
| $T_{base} \times 2^n$ | Exponential backoff | Restart policy |
| $BW \times \eta$ | Efficiency ratio | Network performance |
| $\text{range} / 65536$ | Division | Subordinate ID capacity |
| $1 + C_p$ per pod | Counting | Pod overhead |

---

*Podman proves that containers don't need a daemon — by making each container a direct child process with user namespace isolation, it achieves better security (rootless by default) and reliability (no SPOF) at the cost of slightly more complex networking.*
