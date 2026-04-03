# The Mathematics of Vagrant — Development Environment Provisioning

> *Vagrant manages development environments as reproducible, portable VMs. Its architecture involves box versioning, provider abstraction, multi-machine networking, and the resource overhead model of running local VMs vs. containers.*

---

## 1. Box Versioning (Semantic Version Resolution)

### The Problem

Vagrant boxes use semantic versioning. Understanding version resolution ensures environment reproducibility.

### Version Constraint Solving

$$\text{box\_version} = \text{latest}\{v \in \text{available} : v \in \text{constraint}\}$$

### Constraint Syntax

| Constraint | Meaning | Example |
|:---|:---|:---|
| `= 1.2.3` | Exact | Only 1.2.3 |
| `>= 1.2, < 2.0` | Range | 1.2.0 through 1.x.x |
| `~> 1.2` | Pessimistic | >= 1.2.0, < 2.0.0 |
| `~> 1.2.3` | Pessimistic (patch) | >= 1.2.3, < 1.3.0 |

### Box Storage

$$S_{boxes} = \sum_{b \in \text{boxes}} \sum_{v \in \text{versions}(b)} S_{v}$$

Vagrant keeps old versions by default:

| Boxes | Versions Each | Size Each | Total |
|:---:|:---:|:---:|:---:|
| 3 | 1 | 500 MB | 1.5 GB |
| 3 | 3 | 500 MB | 4.5 GB |
| 5 | 5 | 700 MB | 17.5 GB |

### Cleanup Formula

$$S_{freed} = \sum_{b} \sum_{v \neq v_{latest}} S_{v}$$

`vagrant box prune` removes all but the latest version of each box.

---

## 2. Provider Resource Model (VM Overhead)

### The Problem

Each Vagrant VM consumes host resources. Understanding the overhead model prevents overcommitting.

### Resource Allocation

$$R_{available} = R_{host} - R_{host\_os} - \sum_{i=1}^{N} R_{vm_i}$$

### Memory Overhead

$$M_{total} = M_{host\_os} + \sum_{i=1}^{N} (M_{vm_i} + M_{hypervisor_i})$$

Where $M_{hypervisor} \approx 50\text{-}200$ MB per VM (VirtualBox/libvirt overhead).

### Worked Example: 16 GB Host

| Component | Memory |
|:---|:---:|
| Host OS | 2 GB |
| Hypervisor overhead (3 VMs) | 0.5 GB |
| VM 1 (web) | 2 GB |
| VM 2 (db) | 4 GB |
| VM 3 (cache) | 1 GB |
| **Remaining for host** | **6.5 GB** |

### CPU Overcommit

$$\text{Overcommit ratio} = \frac{\sum \text{vCPUs}_{vm}}{\text{pCPUs}_{host}}$$

| Ratio | Performance Impact |
|:---:|:---|
| < 1.0 | No contention |
| 1.0 - 2.0 | Acceptable for dev |
| 2.0 - 4.0 | Noticeable slowdown |
| > 4.0 | Unusable |

### Disk I/O (VDI/VMDK Overhead)

$$\text{IOPS}_{vm} \approx \text{IOPS}_{host} \times \eta_{driver}$$

| Driver | $\eta$ (Efficiency) |
|:---|:---:|
| VirtualBox (VDI, dynamically allocated) | 0.3 - 0.5 |
| VirtualBox (VDI, fixed size) | 0.5 - 0.7 |
| libvirt (qcow2) | 0.6 - 0.8 |
| libvirt (raw) | 0.8 - 0.95 |

---

## 3. Multi-Machine Networking

### The Problem

Vagrant multi-machine environments create private networks between VMs. The network topology affects communication performance.

### Network Modes

| Mode | Latency | Host Access | External Access |
|:---|:---:|:---:|:---:|
| NAT (default) | 0.5-2 ms | Port forward only | Yes (outbound) |
| Private network | 0.1-0.5 ms | Yes | No |
| Public network (bridged) | 0.1 ms | Yes | Yes |

### Private Network Addressing

$$\text{IP}_{vm_i} = \text{subnet\_base} + i$$

Example: `192.168.56.0/24` network with 3 VMs:

| VM | IP | Reachable By |
|:---|:---|:---|
| web | 192.168.56.10 | db, cache, host |
| db | 192.168.56.11 | web, cache, host |
| cache | 192.168.56.12 | web, db, host |

### Port Forward Collision

Each forwarded port must be unique on the host:

$$\forall i \neq j: \text{host\_port}(i) \neq \text{host\_port}(j)$$

Vagrant auto-corrects collisions by incrementing:

$$\text{host\_port}_{corrected} = \text{host\_port} + k \quad \text{where } k = \text{smallest available offset}$$

---

## 4. Provisioning Strategies

### The Problem

Vagrant supports multiple provisioners. Each has different overhead and capabilities.

### Provisioner Execution Time

$$T_{provision} = T_{upload} + T_{execute}$$

| Provisioner | Upload | Execute | Total (typical) |
|:---|:---:|:---:|:---:|
| Shell (inline) | 0 | 1-60s | 1-60s |
| Shell (script file) | 0.1s | 1-60s | 1-60s |
| Ansible (local) | 0 | 10-300s | 10-300s |
| Ansible (remote) | 5s | 10-300s | 15-305s |
| Puppet (masterless) | 2s | 10-120s | 12-122s |
| Docker | 1s | 10-60s | 11-61s |

### Provisioner Ordering

$$T_{total} = \sum_{p \in \text{provisioners}} T_p$$

Provisioners run in the order defined in the Vagrantfile — sequential within a VM, parallel across VMs.

### Multi-Machine Provisioning

$$T_{multi} = \max_{m \in \text{machines}} \left(T_{boot}(m) + \sum_{p \in \text{provisioners}(m)} T_p(m)\right)$$

Vagrant boots and provisions machines in the order defined, but `--parallel` flag (for supported providers) enables:

$$T_{parallel} = \max_{m} T_{total}(m)$$

---

## 5. Synced Folders (Performance Model)

### The Problem

Synced folders share host files with the VM. Performance varies dramatically by sync method.

### Sync Methods

| Method | Latency | Throughput | CPU Overhead |
|:---|:---:|:---:|:---:|
| VirtualBox shared folders | High (10-50x native) | Low | High |
| NFS | Low (1.2-2x native) | High | Low |
| rsync | Batch (manual or auto) | N/A (copy) | Low |
| SMB (Windows) | Medium (2-5x native) | Medium | Medium |

### VirtualBox Shared Folder Penalty

$$T_{vbox\_sf} = T_{native} \times k \quad \text{where } k \in [10, 50]$$

For `npm install` with 50,000 files:

$$T_{native} = 10\text{s}, \quad T_{vbox\_sf} = 10 \times 30 = 300\text{s} = 5 \text{ min}$$

### NFS Setup

$$T_{nfs} = T_{native} \times 1.5 \quad \text{(approximate)}$$

Same npm install: $T_{nfs} = 15\text{s}$ — 20x faster than VBox shared folders.

### rsync Strategy

$$T_{sync} = \frac{S_{changed}}{BW_{local}} + N_{files\_changed} \times T_{overhead}$$

For small changes (< 100 files): $T_{sync} < 1\text{s}$.
For initial sync of full project: $T_{sync} = S_{project} / BW$.

---

## 6. Snapshot and Restore (State Management)

### The Problem

Vagrant snapshots save VM state for instant restore. Understanding the cost model helps manage snapshots.

### Snapshot Size

$$S_{snapshot} = S_{memory} + S_{disk\_delta}$$

Where $S_{disk\_delta}$ = changed disk blocks since last snapshot.

### Snapshot Chain Performance

$$T_{restore}(n) = T_{base} + n \times T_{delta\_apply}$$

Where $n$ = number of delta snapshots in the chain.

| Chain Depth | Restore Time | Disk Usage |
|:---:|:---:|:---:|
| 1 | 5s | $S_{mem} + S_{\delta_1}$ |
| 3 | 8s | $S_{mem} + \sum S_{\delta_i}$ |
| 5 | 12s | $S_{mem} + \sum S_{\delta_i}$ |
| 10 | 25s | $S_{mem} + \sum S_{\delta_i}$ (fragmented) |

### Snapshot vs. Rebuild Decision

$$T_{rebuild} = T_{boot} + T_{provision}$$
$$T_{restore} = T_{snapshot\_restore}$$

Use snapshots when: $T_{rebuild} > T_{restore}$, which is almost always.

---

## 7. Vagrant vs. Docker (Resource Comparison)

### The Problem

When should you use Vagrant VMs vs. Docker containers for development?

### Resource Overhead Comparison

| Resource | Vagrant VM | Docker Container |
|:---|:---:|:---:|
| Memory overhead | 200-500 MB | 10-50 MB |
| Disk overhead | 1-5 GB | 50-500 MB |
| Boot time | 30-120s | 0.5-5s |
| CPU overhead | 5-15% | < 1% |
| Kernel | Full guest kernel | Shared host kernel |

### When Vagrant Wins

$$\text{Vagrant preferred} \iff \text{need:} \begin{cases}
\text{Different kernel version} \\
\text{Systemd / init system} \\
\text{Full OS environment (NixOS, FreeBSD)} \\
\text{Network stack isolation} \\
\text{Kernel module testing}
\end{cases}$$

### Density Comparison

On a 32 GB host with 8 cores:

$$N_{vagrant} = \lfloor \frac{32 - 4}{2} \rfloor = 14 \text{ VMs (2GB each)}$$

$$N_{docker} = \lfloor \frac{32 - 2}{0.1} \rfloor = 300 \text{ containers (100MB each)}$$

Containers achieve ~21x higher density.

---

## 8. Summary of Functions by Type

| Formula | Math Type | Domain |
|:---|:---|:---|
| $\text{latest}\{v : v \in \text{constraint}\}$ | Constraint solving | Version resolution |
| $R_{host} - R_{os} - \sum R_{vm}$ | Subtraction | Resource planning |
| $T_{native} \times k$ | Multiplicative penalty | Sync folder performance |
| $S_{mem} + S_{\delta}$ | Summation | Snapshot sizing |
| $\max_m T_{total}(m)$ | Critical path | Multi-machine |
| $\lfloor (M - M_{os}) / M_{vm} \rfloor$ | Floor division | VM density |

---

*Vagrant turns "it works on my machine" into "it works on every developer's machine" — box versioning ensures reproducibility, multi-machine networking simulates production topology, and synced folder performance determines whether your development experience is pleasant or painful.*
