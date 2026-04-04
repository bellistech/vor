# The Mathematics of containerd — Container Runtime Internals

> *containerd is the industry-standard container runtime — a daemon that manages the complete container lifecycle. Its architecture is built on shim processes, snapshot drivers, and content-addressable storage with precise resource accounting.*

---

## 1. Shim Process Model (Process Tree Arithmetic)

### The Problem

containerd uses a **shim** process per container to decouple the container lifecycle from the daemon. How many processes does a host running $N$ containers require?

### The Formula

$$P_{total} = 1 + N + \sum_{i=1}^{N} C_i$$

Where:
- $1$ = the containerd daemon itself
- $N$ = number of shim processes (one per container)
- $C_i$ = number of user processes in container $i$

### Worked Examples

| Containers ($N$) | Processes/Container | Shims | Total Processes |
|:---:|:---:|:---:|:---:|
| 1 | 1 | 1 | 3 |
| 10 | 1 | 10 | 21 |
| 50 | 3 | 50 | 201 |
| 100 | 2 | 100 | 301 |

### Shim Memory Overhead

Each shim v2 process consumes approximately 2-4 MB RSS. Total shim overhead:

$$M_{shim} = N \times m_{shim}$$

For 100 containers at 3 MB per shim: $M_{shim} = 100 \times 3 = 300$ MB overhead before any container workload memory.

### Why Shims Matter

If containerd crashes, shims keep containers running. The **blast radius** of a daemon restart is zero container downtime — this is the architectural reason Kubernetes chose containerd over dockerd.

---

## 2. Content-Addressable Storage (Hashing)

### The Problem

containerd stores all image layers and configs in a content-addressable store. Every object is identified by its cryptographic digest.

### The Digest Function

$$\text{digest} = \text{sha256}(\text{content})$$

$$\text{key} = \text{sha256:} \| \text{hex}(\text{digest})$$

The probability of a collision (two different objects producing the same hash):

$$P_{collision} \approx \frac{n^2}{2^{257}}$$

Where $n$ = number of objects. For $n = 10^9$ (one billion objects):

$$P_{collision} \approx \frac{10^{18}}{2^{257}} \approx \frac{10^{18}}{2.31 \times 10^{77}} \approx 4.3 \times 10^{-60}$$

### Deduplication Savings

If $L$ layers are pulled across $I$ images, but only $U$ are unique:

$$\text{Savings} = 1 - \frac{U}{\sum_{i=1}^{I} L_i}$$

| Images | Total Layers | Unique Layers | Storage Savings |
|:---:|:---:|:---:|:---:|
| 5 | 25 | 10 | 60% |
| 20 | 80 | 18 | 77.5% |
| 50 | 200 | 30 | 85% |

---

## 3. Snapshot Driver Performance (Copy-on-Write)

### The Problem

containerd supports multiple snapshot drivers (overlayfs, btrfs, zfs, devmapper). Each has different performance characteristics for copy-on-write operations.

### Overlay2 Layer Resolution

When a file is read, the overlay driver searches top-down through $L$ layers:

$$T_{lookup} = O(L)$$

When a file is modified (copy-up):

$$T_{write} = T_{copy} + T_{modify}$$

$$T_{copy} = \frac{S_{file}}{BW_{disk}}$$

Where $S_{file}$ is the full file size — even modifying one byte copies the entire file.

### Snapshot Space Formula

$$S_{container} = S_{writable} + \sum_{j \in \text{shared}} \frac{S_j}{R_j}$$

Where:
- $S_{writable}$ = data written by this container
- $S_j$ = size of shared layer $j$
- $R_j$ = number of containers sharing layer $j$ (amortized cost)

### Driver Comparison

| Driver | Copy-on-Write Unit | Space Overhead | Random Write |
|:---|:---:|:---:|:---:|
| overlayfs | Whole file | Low | $O(S_{file})$ |
| btrfs | 4 KB block | Medium | $O(4\text{KB})$ |
| zfs | 128 KB block | Medium | $O(128\text{KB})$ |
| devmapper | 64 KB block | High (thin pool) | $O(64\text{KB})$ |

---

## 4. Garbage Collection (Reference Counting)

### The Problem

containerd must reclaim storage from unused content. It uses a mark-and-sweep collector over the content store.

### The Algorithm

1. **Mark phase** — walk all references from containers, images, snapshots:

$$\text{Reachable} = \bigcup_{r \in \text{roots}} \text{closure}(r)$$

2. **Sweep phase** — delete unreachable content:

$$\text{Garbage} = \text{ContentStore} \setminus \text{Reachable}$$

### GC Time Complexity

$$T_{GC} = O(|\text{ContentStore}|) + O(|\text{References}|)$$

For a system with 500 content objects and 2000 references, GC typically completes in under 100ms.

### Lease Mechanism

Active operations acquire **leases** to prevent GC races:

$$\text{Protected} = \text{Reachable} \cup \text{Leased}$$

A content object is only collected if it appears in neither set.

---

## 5. CRI Plugin Request Routing (Kubernetes Integration)

### The Problem

When Kubernetes calls containerd via CRI (Container Runtime Interface), how does request latency scale?

### The Pipeline

$$T_{total} = T_{CRI} + T_{image} + T_{snapshot} + T_{runtime} + T_{shim}$$

| Stage | Typical Latency | Depends On |
|:---|:---:|:---|
| CRI gRPC decode | 0.1 ms | Message size |
| Image pull (cached) | 0 ms | Content store lookup |
| Image pull (remote) | 2-30 s | Network, layer count |
| Snapshot prepare | 1-5 ms | Driver, layer count |
| OCI runtime create | 10-50 ms | Namespace setup |
| Shim start | 5-15 ms | Binary load |

### Container Start Latency (Cached Image)

$$T_{start} \approx 20\text{-}70 \text{ ms}$$

This is significantly faster than Docker's path (which adds dockerd routing overhead of 20-50 ms).

---

## 6. Namespace Isolation (Multi-Tenancy)

### The Model

containerd supports **namespaces** (not Linux namespaces — containerd-level logical partitions). Resources in namespace $A$ are invisible to namespace $B$:

$$\text{Visible}(ns) = \{r \in R : r.\text{namespace} = ns\}$$

$$\text{Visible}(A) \cap \text{Visible}(B) = \emptyset \quad \text{for } A \neq B$$

Kubernetes uses the namespace `k8s.io`; Docker uses `moby`. They share the same containerd daemon but cannot see each other's containers.

---

## 7. Summary of Functions by Type

| Formula | Math Type | Domain |
|:---|:---|:---|
| $1 + N + \Sigma C_i$ | Linear / Summation | Process accounting |
| $n^2 / 2^{257}$ | Probability | Content addressing |
| $1 - U / \Sigma L_i$ | Ratio | Deduplication |
| $\text{ContentStore} \setminus \text{Reachable}$ | Set difference | Garbage collection |
| $O(L)$ layer lookup | Complexity | Snapshot drivers |

---

*containerd sits beneath Kubernetes on every major cloud provider — its shim architecture and content-addressable storage are the foundation that makes container orchestration possible.*

## Prerequisites

- Linux process model (PID, process trees, signals)
- OCI image and runtime specifications
- Content-addressable storage concepts (SHA-256 hashing)
- Understanding of Kubernetes CRI (for CRI plugin usage)

## Complexity

- Beginner: pulling images and running containers via nerdctl
- Intermediate: namespace isolation, snapshot drivers, garbage collection
- Advanced: shim architecture, CRI plugin internals, custom snapshot drivers, content store optimization
