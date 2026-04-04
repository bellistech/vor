# The Mathematics of Docker — Container Internals

> *Docker containers aren't virtual machines — they're isolated process groups built on Linux namespaces, cgroups, and union filesystems. Every resource limit, layer operation, and isolation boundary has precise mathematical underpinnings.*

---

## 1. Union Filesystem Layer Math (Overlay2)

### The Problem

Docker images are composed of stacked read-only layers. Understanding the storage cost requires modeling layer deduplication and copy-on-write overhead.

### Image Size Formula

For a multi-stage build, the final image size is only the layers from the final stage:

$$S_{final} = \sum_{i \in \text{final stage}} S_{\text{layer}_i}$$

Not the sum of all build layers. This is the key insight of multi-stage builds.

### Worked Example: Multi-Stage Build

| Stage | Layers | Total Size |
|:---|:---|:---:|
| Build stage (golang:1.24) | Go compiler, source, deps, binary | 1.2 GB |
| Final stage (alpine:3.19) | Alpine base + COPY binary | 15 MB |

$$S_{build} = 1.2 \text{ GB} \quad \text{(discarded)}$$
$$S_{final} = 12\text{ MB (alpine)} + 3\text{ MB (binary)} = 15\text{ MB}$$

**Savings:** $1 - \frac{15}{1200} = 98.75\%$

### Layer Deduplication Across Containers

If $C$ containers share the same base image with $L$ layers of total size $S_{base}$:

$$S_{total} = S_{base} + \sum_{i=1}^{C} S_{writable_i}$$

Not $C \times S_{base}$. For 100 containers sharing a 200 MB base image:

$$S_{naive} = 100 \times 200 = 20{,}000 \text{ MB}$$
$$S_{actual} = 200 + 100 \times S_{writable} \approx 200 + 100 \times 5 = 700 \text{ MB}$$

### Copy-on-Write Cost

When a container modifies a file in a read-only layer, the entire file is copied up:

$$\text{CoW cost} = S_{file} \quad \text{(not } S_{modified\_bytes}\text{)}$$

Modifying 1 byte of a 100 MB log file copies all 100 MB. This is why you should never modify large files in lower layers.

---

## 2. Cgroup v2 Resource Accounting

### CPU Limits: The Quota/Period Model

$$\text{cpu.max} = \frac{\text{quota}}{\text{period}}$$

Where:
- quota = microseconds of CPU time allowed per period
- period = scheduling period (default: 100,000 us = 100 ms)

### The CPU Shares to Cores Mapping

$$\text{Effective CPUs} = \frac{\text{quota}}{\text{period}}$$

| Docker Flag | Quota | Period | Effective CPUs |
|:---|:---:|:---:|:---:|
| `--cpus=0.5` | 50,000 | 100,000 | 0.5 |
| `--cpus=1` | 100,000 | 100,000 | 1.0 |
| `--cpus=2.5` | 250,000 | 100,000 | 2.5 |
| `--cpus=4` | 400,000 | 100,000 | 4.0 |

### CPU Shares (Relative Weighting)

When containers compete for CPU, shares determine proportional allocation:

$$\text{CPU}_i = \frac{W_i}{\sum_{j=1}^{N} W_j} \times C_{available}$$

Where $W_i$ = container $i$'s shares (default: 1024).

**Worked Example:** 3 containers with shares 1024, 512, 256 on a 4-core host:

$$\text{CPU}_1 = \frac{1024}{1024 + 512 + 256} \times 4 = \frac{1024}{1792} \times 4 = 2.29 \text{ cores}$$
$$\text{CPU}_2 = \frac{512}{1792} \times 4 = 1.14 \text{ cores}$$
$$\text{CPU}_3 = \frac{256}{1792} \times 4 = 0.57 \text{ cores}$$

### Memory Limits

$$\text{memory.max} = L \text{ (hard limit in bytes)}$$

When usage exceeds $L$, the OOM killer activates. The OOM score:

$$\text{oom\_score} = \frac{\text{RSS}_{process}}{\text{memory.max}} \times 1000$$

Higher score = killed first.

---

## 3. Namespace Isolation Model

### The Seven Namespaces

Docker uses 7 Linux namespaces to create process isolation. Each namespace partitions a global resource:

| Namespace | Isolates | Kernel Flag |
|:---|:---|:---|
| **mnt** | Filesystem mount points | `CLONE_NEWNS` |
| **pid** | Process IDs | `CLONE_NEWPID` |
| **net** | Network stack (interfaces, routes, iptables) | `CLONE_NEWNET` |
| **ipc** | System V IPC, POSIX message queues | `CLONE_NEWIPC` |
| **uts** | Hostname and NIS domain name | `CLONE_NEWUTS` |
| **user** | UIDs and GIDs | `CLONE_NEWUSER` |
| **cgroup** | Cgroup root directory | `CLONE_NEWCGROUP` |

### Isolation as Set Partitioning

For a global resource set $R$ and $N$ containers:

$$R = R_{host} \cup R_1 \cup R_2 \cup \cdots \cup R_N$$

$$R_i \cap R_j = \emptyset \quad \forall i \neq j$$

Each container sees only its partition $R_i$. PID 1 inside the container maps to a high PID on the host — the mapping function:

$$\text{pid}_{host} = f(\text{pid}_{container}, \text{ns}_{id})$$

### Network Namespace: veth Pair Model

Each container gets a virtual ethernet pair:

$$\text{veth}_{host} \longleftrightarrow \text{veth}_{container}$$

Traffic flow through docker0 bridge:

$$\text{Container} \xrightarrow{\text{veth}} \text{Bridge (docker0)} \xrightarrow{\text{iptables NAT}} \text{Host NIC} \xrightarrow{} \text{Network}$$

The bridge adds latency: typically 10-50 us per packet vs. host networking (0 us overhead).

---

## 4. Image Layer Graph (Directed Acyclic Graph)

### The Problem

Docker images form a DAG of layers. Understanding the graph structure enables optimization.

### The Layer DAG

$$G = (V, E) \text{ where } V = \text{layers}, E = \text{parent relationships}$$

$$\text{depth}(image) = \text{longest path from scratch to top layer}$$

### Layer Count Impact on Pull Time

$$T_{pull} = \sum_{i=1}^{L} \max\left(\frac{S_i}{BW}, T_{overhead}\right)$$

Where layers are pulled in parallel (up to concurrency limit, default 3):

$$T_{parallel} = \max_{batch} \sum_{i \in batch} \frac{S_i}{BW}$$

### Build Cache Hit Rate

$$\text{Hit Rate} = \frac{\text{Unchanged layers}}{\text{Total layers}} = \frac{L - I}{L}$$

Where $I$ = number of invalidated layers. Cache invalidation cascades — changing layer $k$ invalidates all layers $k+1, k+2, \ldots, L$:

$$I = L - k + 1$$

**Optimization:** Put rarely-changing layers (OS, deps) first, frequently-changing layers (source code) last.

---

## 5. Networking: Port Mapping and NAT

### Port Mapping Formula

$$\text{Mapping}: (0.0.0.0:\text{host\_port}) \rightarrow (\text{container\_ip}:\text{container\_port})$$

Maximum containers per host port:

$$C_{max} = 1 \text{ (port conflict)}$$

Maximum containers with dynamic ports:

$$C_{max} = 65535 - 1024 = 64{,}511 \text{ (ephemeral range)}$$

### Docker Network MTU

$$\text{MTU}_{container} = \text{MTU}_{host} - H_{encap}$$

| Network Mode | Encap Overhead | Effective MTU |
|:---|:---:|:---:|
| bridge | 0 | 1500 |
| overlay (VXLAN) | 50 bytes | 1450 |
| macvlan | 0 | 1500 |

---

## 6. Storage Driver I/O Amplification

### Write Amplification Factor

$$\text{WAF} = \frac{\text{Bytes written to disk}}{\text{Bytes written by application}}$$

| Operation | overlay2 WAF | Reason |
|:---|:---:|:---|
| New file write | 1.0 | Direct to upper layer |
| Modify existing small file (4KB) | 1.0 | Full copy-up, but file is small |
| Modify 1 byte in 100MB file | 26,214 | Copies 100MB for 1 byte change |
| Append to log file (repeated) | 1.0 first, then 1.0 | Already in upper layer after first CoW |

### The Lesson

$$\text{WAF}_{modify} = \frac{S_{file}}{S_{change}}$$

This is why containers should write to volumes, not the container filesystem.

---

## 7. Summary of Functions by Type

| Formula | Math Type | Domain |
|:---|:---|:---|
| $\Sigma S_{\text{layer}_i}$ | Summation | Image sizing |
| $\text{quota}/\text{period}$ | Ratio | CPU accounting |
| $W_i / \Sigma W_j$ | Weighted proportion | CPU shares |
| $R_i \cap R_j = \emptyset$ | Set partition | Namespace isolation |
| $S_{file} / S_{change}$ | Amplification ratio | Storage I/O |
| $(L-k+1)$ cache invalidation | Cascade counting | Build optimization |

---

*Every `docker run` invocation creates 7 namespaces, configures cgroup limits, prepares an overlay mount, and sets up a veth pair — all in under 500ms. The math governs what you pay for that isolation.*

## Prerequisites

- Linux process model (fork, exec, PID)
- Filesystem basics (mount points, inodes)
- Networking fundamentals (IP, bridges, NAT, iptables)
- Basic understanding of cgroups and namespaces

## Complexity

- Beginner: pulling images, running containers, basic Dockerfile
- Intermediate: multi-stage builds, networking, volumes, compose
- Advanced: overlay2 internals, cgroup tuning, custom runtimes, security profiles
