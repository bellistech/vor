# The Mathematics of LXD — System Container Internals

> *LXD runs full Linux systems as containers — not single processes, but complete OS instances with init, services, and users. Its resource model, storage pool management, and live migration mechanics require precise mathematical reasoning.*

---

## 1. System Container Resource Model

### The Problem

Unlike application containers (one process), LXD containers run full systems. Resource accounting must handle hundreds of processes per container.

### CPU Pinning and CFS Scheduling

LXD supports both CPU pinning and CFS quota:

**Pinned allocation:**

$$\text{CPUs}(c) \subseteq \{0, 1, \ldots, N_{host}-1\}$$

$$|\text{CPUs}(c)| = \text{limits.cpu}$$

**CFS quota (allowance):**

$$\text{Effective CPUs} = \frac{\text{limits.cpu.allowance (quota)}}{\text{period}}$$

**CFS priority (shares):**

$$\text{CPU}_i = \frac{W_i}{\sum_{j} W_j} \times C_{total}$$

### Memory Accounting with Swap

$$\text{memory.max} = L_{mem}$$
$$\text{memory.swap.max} = L_{swap}$$
$$\text{Total addressable} = L_{mem} + L_{swap}$$

### Worked Example: 4-Container Host

| Container | CPU (pinned) | Memory | Disk |
|:---|:---:|:---:|:---:|
| web | cores 0-1 | 2 GB | 20 GB |
| db | cores 2-5 | 8 GB | 100 GB |
| build | cores 6-7 | 4 GB | 50 GB |
| monitor | 50ms/100ms (0.5 CPU) | 1 GB | 10 GB |
| **Host total** | 8 cores | 16 GB | — |

Overcommit ratio:

$$R_{cpu} = \frac{\text{allocated cores}}{\text{physical cores}} = \frac{6.5}{8} = 0.81 \quad \text{(safe)}$$

$$R_{mem} = \frac{15}{16} = 0.94 \quad \text{(tight, no overcommit margin)}$$

---

## 2. Storage Pool Drivers (Copy-on-Write Efficiency)

### The Problem

LXD supports ZFS, btrfs, LVM, and dir storage backends. Each has different snapshot and clone costs.

### ZFS Block Cloning

ZFS clones are instant — metadata only:

$$T_{clone} = O(1) \quad \text{(regardless of dataset size)}$$
$$S_{clone} = 0 \quad \text{(initial — grows with divergence)}$$

Over time, clone space grows with writes:

$$S_{clone}(t) = \sum_{\text{writes}} S_{block}$$

### Snapshot Space Accounting

Snapshots hold only changed blocks (copy-on-write):

$$S_{snapshot} = S_{blocks\_changed\_since\_snapshot}$$

**Not** $S_{dataset}$. A 50 GB container with a snapshot that has seen 2 GB of changes:

$$S_{total} = 50 + 2 = 52 \text{ GB (not 100 GB)}$$

### Storage Driver Comparison

| Driver | Clone Time | Snapshot Space | Resize | Quotas |
|:---|:---:|:---:|:---:|:---:|
| ZFS | $O(1)$ | CoW blocks only | Online | Native |
| btrfs | $O(1)$ | CoW blocks only | Online | Subvolume |
| LVM | $O(S)$ copy | Requires pre-alloc | Offline | LV size |
| dir (rsync) | $O(S)$ copy | Full copy | Instant | Filesystem |

### Pool Fill Rate

$$T_{full} = \frac{S_{pool} - S_{used}}{R_{write}}$$

Where $R_{write}$ = aggregate write rate across all containers. For a 500 GB pool, 400 GB used, 10 MB/s write rate:

$$T_{full} = \frac{100 \text{ GB}}{10 \text{ MB/s}} = 10{,}000\text{ s} \approx 2.8 \text{ hours}$$

---

## 3. Image Distribution (Content Hashing)

### The Problem

LXD images are distributed via simplestreams protocol. Images are identified by fingerprints.

### The Fingerprint

$$\text{fingerprint} = \text{SHA-256}(\text{rootfs.tar.xz} \| \text{metadata.yaml})$$

### Delta Updates

When pulling an updated image, LXD can compute the delta:

$$S_{transfer} = S_{new} - S_{shared}$$

Using binary diff (bsdiff):

$$\text{Compression ratio} = \frac{S_{diff}}{S_{full}}$$

| Image Update Type | Full Size | Delta Size | Savings |
|:---|:---:|:---:|:---:|
| Security patch | 350 MB | 15 MB | 95.7% |
| Minor version bump | 350 MB | 80 MB | 77.1% |
| Major OS upgrade | 350 MB | 320 MB | 8.6% |

---

## 4. Live Migration (Pre-Copy Algorithm)

### The Problem

LXD supports live migration using CRIU (Checkpoint/Restore In Userspace). The pre-copy algorithm iteratively transfers memory pages.

### The Pre-Copy Model

In each iteration $i$, transfer dirty pages while the container runs:

$$D_i = D_0 \times r^i$$

Where:
- $D_0$ = total memory pages
- $r$ = dirty rate (fraction of pages dirtied per iteration), $0 < r < 1$

### Convergence Condition

Migration converges when dirty pages in an iteration can be transferred faster than they are dirtied:

$$\frac{D_i \times P_{size}}{BW} < T_{iteration}$$

$$D_0 \times r^i \times P_{size} < BW \times T_{iter}$$

Solving for iterations to converge:

$$i > \frac{\log(BW \times T_{iter}) - \log(D_0 \times P_{size})}{\log(r)}$$

### Worked Example

- Container memory: 4 GB = $D_0 = 1{,}048{,}576$ pages (4 KB each)
- Network bandwidth: 10 Gbps = 1.25 GB/s
- Dirty rate: $r = 0.1$ (10% of pages dirtied per iteration)
- Iteration time: 1 second

$$i > \frac{\log(1.25 \times 10^9) - \log(1{,}048{,}576 \times 4096)}{\log(0.1)}$$

$$i > \frac{9.097 - 9.633}{-1} = 0.536$$

**After 1 iteration**, the remaining dirty pages fit in one transfer. Total downtime:

$$T_{downtime} = \frac{D_1 \times P_{size}}{BW} = \frac{104{,}858 \times 4096}{1.25 \times 10^9} \approx 0.34\text{ s}$$

### Migration Failure

If the dirty rate is too high ($r \rightarrow 1$), migration never converges:

$$r > 1 - \frac{BW \times T_{iter}}{D_0 \times P_{size}} \implies \text{non-convergent}$$

---

## 5. Network Modes (Bridge, macvlan, SR-IOV)

### Bridge Mode Throughput

$$\text{Throughput}_{bridge} = BW_{host} - O_{bridge}$$

Where $O_{bridge}$ is the bridge processing overhead (typically 2-5% for software bridge).

### macvlan Direct Attach

$$\text{Throughput}_{macvlan} \approx BW_{host} \quad \text{(near line-rate)}$$

No bridge overhead, but containers cannot communicate with the host on the same interface.

### Network Bandwidth Allocation

With LXD network limits:

$$BW_i = \min(\text{limits.ingress}_i, BW_{link})$$

$$\sum_{i=1}^{N} BW_i \leq BW_{link} \quad \text{(when all saturated)}$$

| Container | Ingress Limit | Egress Limit | Share of 10G Link |
|:---|:---:|:---:|:---:|
| web | 2 Gbps | 2 Gbps | 20% |
| db | 5 Gbps | 1 Gbps | 50% / 10% |
| backup | 1 Gbps | 1 Gbps | 10% |
| unrestricted | 10 Gbps | 10 Gbps | Remainder |

---

## 6. Profile Inheritance (Layered Configuration)

### The Model

LXD profiles stack — later profiles override earlier ones:

$$\text{Config}(c) = P_1 \triangleleft P_2 \triangleleft \cdots \triangleleft P_k \triangleleft C_{overrides}$$

Where $\triangleleft$ means "right overrides left for conflicting keys."

### Device Resolution

$$\text{device}(name) = \text{last profile defining } name$$

For $N$ profiles with $D$ devices each:

$$|\text{effective devices}| \leq \sum_{i=1}^{N} D_i \quad \text{(upper bound)}$$
$$|\text{effective devices}| \geq \max_i D_i \quad \text{(lower bound, if all names conflict)}$$

---

## 7. Clustering (Raft Consensus for LXD Database)

### The Problem

LXD clusters use Raft consensus for the distributed database (dqlite). The quorum requirement:

$$\text{Quorum} = \lfloor N/2 \rfloor + 1$$

### Fault Tolerance

$$F = N - \text{Quorum} = \lfloor (N-1)/2 \rfloor$$

| Cluster Size | Quorum | Tolerates Failures |
|:---:|:---:|:---:|
| 3 | 2 | 1 |
| 5 | 3 | 2 |
| 7 | 4 | 3 |

### Container Placement

When creating a container on a cluster, LXD selects the member with the most free resources:

$$\text{target} = \arg\max_{m \in \text{members}} \left(\alpha \times \text{free\_cpu}(m) + \beta \times \text{free\_mem}(m) + \gamma \times \text{free\_disk}(m)\right)$$

Where $\alpha, \beta, \gamma$ are weighting factors.

---

## 8. Summary of Functions by Type

| Formula | Math Type | Domain |
|:---|:---|:---|
| $W_i / \Sigma W_j$ | Weighted proportion | CPU scheduling |
| $D_0 \times r^i$ | Geometric series | Live migration |
| $\lfloor N/2 \rfloor + 1$ | Floor function | Raft quorum |
| $O(1)$ clone | Complexity | ZFS/btrfs storage |
| $P_1 \triangleleft P_2 \triangleleft \cdots$ | Override algebra | Profile inheritance |

---

*LXD bridges the gap between VMs and containers — you get the density of containers with the operational model of full machines. The math of live migration, ZFS cloning, and Raft consensus makes this possible.*

## Prerequisites

- Linux system administration (systemd, networking, storage)
- Cgroups and namespaces concepts
- Storage fundamentals (ZFS, btrfs, or LVM)
- Networking (bridges, VLANs, macvlan)

## Complexity

- Beginner: launching instances, basic config, snapshots
- Intermediate: profiles, storage pools, networking, cloud-init integration
- Advanced: live migration, clustering, Raft consensus, SR-IOV passthrough
