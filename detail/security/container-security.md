# The Mathematics of Container Security — Isolation Boundaries and Attack Surface

> *Container security is the mathematics of shared kernel isolation: namespace partitioning, cgroup resource limits, capability reduction, and image layer verification. The security model is a defense-in-depth probability calculation where each isolation mechanism is an independent barrier.*

---

## 1. Namespace Isolation — Kernel Partitioning

### Linux Namespaces

Each namespace creates an independent view of a kernel resource:

| Namespace | Isolates | Kernel Struct |
|:---|:---|:---|
| PID | Process IDs | `pid_namespace` |
| Network | Network stack | `net_namespace` |
| Mount | Filesystem mounts | `mnt_namespace` |
| UTS | Hostname | `uts_namespace` |
| IPC | System V IPC | `ipc_namespace` |
| User | UID/GID mappings | `user_namespace` |
| Cgroup | Cgroup root | `cgroup_namespace` |

### Isolation Strength

Total isolation boundaries for a container:

$$I_{total} = \prod_{i=1}^{7} I_i$$

Where $I_i$ is the isolation effectiveness of namespace $i$. Each namespace is an independent boundary — an attacker must escape ALL of them.

### Container vs VM Isolation

| Mechanism | Shared Kernel | Attack Surface | Escape CVEs (2020-2024) |
|:---|:---:|:---:|:---:|
| Container (namespaces) | Yes | Kernel syscalls | ~15 |
| gVisor (user-space kernel) | Partial | gVisor syscall layer | ~3 |
| Kata (microVM) | No | Hypervisor | ~1 |
| Full VM (KVM/Xen) | No | Hypervisor + firmware | ~2 |

### Syscall Attack Surface

The Linux kernel exposes ~450 system calls. Containers with seccomp:

$$\text{Attack surface ratio} = \frac{|\text{allowed syscalls}|}{|\text{total syscalls}|}$$

| Profile | Allowed Syscalls | Blocked | Ratio |
|:---|:---:|:---:|:---:|
| No seccomp | 450 | 0 | 100% |
| Docker default | ~310 | ~140 | 69% |
| Hardened profile | ~200 | ~250 | 44% |
| Minimal (static binary) | ~40 | ~410 | 9% |

---

## 2. Capability Reduction

### Default Docker Capabilities

Docker drops 22 of 41 capabilities by default:

$$\text{Dropped by default} = \frac{22}{41} = 53.7\%$$

### Remaining Dangerous Capabilities

| Capability | Risk | Attack Scenario |
|:---|:---|:---|
| CAP_SYS_ADMIN | Critical | Mount host filesystem, BPF, namespace ops |
| CAP_NET_RAW | High | ARP spoofing, packet sniffing |
| CAP_NET_BIND_SERVICE | Medium | Bind to privileged ports |
| CAP_DAC_OVERRIDE | High | Read any file |
| CAP_SETUID/SETGID | High | Change identity |

### Privileged Container

`--privileged` restores ALL 41 capabilities plus device access:

$$\text{Isolation}_{privileged} \approx 0 \text{ (effectively root on host)}$$

This is equivalent to running directly on the host — every container escape is trivial.

---

## 3. Image Layer Verification

### Content-Addressable Storage

Each image layer is identified by its SHA-256 digest:

$$\text{layer\_id} = \text{SHA-256}(\text{layer\_content})$$

### Image Manifest Verification

$$\text{verified} = \text{Verify}(\text{manifest.signature}, \text{registry.publicKey})$$

And for each layer:

$$\text{SHA-256}(\text{downloaded\_layer}) \stackrel{?}{=} \text{manifest.layers}[i].\text{digest}$$

### Supply Chain Attack Probability

Without image signing:

$$P(\text{tampered image accepted}) = P(\text{registry compromise}) + P(\text{MITM})$$

With content trust (Notary/cosign):

$$P(\text{tampered image accepted}) = P(\text{registry compromise}) \times P(\text{signing key compromise})$$

The product makes dual compromise required — independent probability multiplication.

### Layer Deduplication

For $n$ images sharing base layers:

$$\text{Storage} = |L_{unique}| + \sum_{i=1}^{n} |L_{app,i}|$$

| Images | Without Dedup | With Dedup | Savings |
|:---:|:---:|:---:|:---:|
| 10 (same base) | 2 GB | 400 MB | 80% |
| 100 (same base) | 20 GB | 1.2 GB | 94% |
| 100 (5 bases) | 20 GB | 3 GB | 85% |

---

## 4. Cgroup Resource Limits

### CPU Limits

CPU allocation uses CFS (Completely Fair Scheduler) quotas:

$$\text{CPU fraction} = \frac{\text{quota}}{\text{period}}$$

Default period: 100,000 $\mu$s (100 ms).

| Limit | Quota | Period | CPU Fraction |
|:---|:---:|:---:|:---:|
| 0.5 CPU | 50,000 | 100,000 | 50% |
| 1.0 CPU | 100,000 | 100,000 | 100% |
| 2.0 CPU | 200,000 | 100,000 | 200% |
| 0.25 CPU | 25,000 | 100,000 | 25% |

### Memory Limits and OOM

When a container exceeds its memory limit:

$$\text{OOM score} = \frac{\text{RSS}_{process}}{\text{memory.limit}} \times 1000 + \text{oom\_score\_adj}$$

The process with the highest OOM score is killed first.

### Fork Bomb Protection

Without PID limits, a fork bomb creates $2^n$ processes in $n$ iterations:

| Iterations | Processes | Time (~1ms/fork) |
|:---:|:---:|:---:|
| 10 | 1,024 | 10 ms |
| 20 | 1,048,576 | 20 ms |
| 30 | 1,073,741,824 | 30 ms (system dead) |

PID limit (`--pids-limit 100`) caps the explosion: the container fails but the host survives.

---

## 5. Network Policy — Graph Theory

### Pod Network as a Graph

In Kubernetes, network policies define allowed communication:

$$G = (P, E) \quad \text{where } P = \text{pods}, E = \text{allowed connections}$$

### Default Allow vs Default Deny

| Policy | Edge Count | Security |
|:---|:---:|:---|
| Default allow | $|P|^2$ (complete graph) | No isolation |
| Default deny + specific rules | $|E| \ll |P|^2$ | Least privilege |

### Micro-segmentation

With $n$ pods and $k$ legitimate communication paths:

$$\text{Blocked connections} = n^2 - k$$

$$\text{Reduction ratio} = 1 - \frac{k}{n^2}$$

| Pods | Legitimate Paths | Blocked | Reduction |
|:---:|:---:|:---:|:---:|
| 50 | 200 | 2,300 | 92% |
| 100 | 500 | 9,500 | 95% |
| 500 | 2,000 | 248,000 | 99.2% |

### Lateral Movement Impact

With network policies: an attacker who compromises pod $p$ can only reach neighbors $N(p)$:

$$\text{Blast radius} = |N(p)| \text{ vs } |P| - 1 \text{ (without policies)}$$

---

## 6. Vulnerability Scanning — Image Risk Scoring

### CVSS Score Distribution

For a typical container image:

$$\text{Risk score} = \sum_{i=1}^{n} w(s_i) \times s_i$$

Where $s_i$ is the CVSS score of vulnerability $i$ and $w(s_i)$ is the weight.

| Severity | CVSS Range | Weight | Typical Count per Image |
|:---|:---:|:---:|:---:|
| Critical | 9.0-10.0 | 10 | 0-5 |
| High | 7.0-8.9 | 5 | 5-20 |
| Medium | 4.0-6.9 | 2 | 20-50 |
| Low | 0.1-3.9 | 1 | 10-30 |

### Image Age vs Vulnerability Count

Vulnerabilities accumulate over time:

$$V(t) \approx V_0 + r \times t$$

Where $r$ is the vulnerability discovery rate (~2-5 new CVEs/week for popular base images).

| Image Age | Expected New CVEs | Critical/High |
|:---:|:---:|:---:|
| 1 week | 3-5 | 0-1 |
| 1 month | 12-20 | 2-4 |
| 3 months | 36-60 | 6-12 |
| 6 months | 72-120 | 12-24 |
| 1 year | 144-240 | 24-48 |

This is why base images should be rebuilt weekly or biweekly.

---

## 7. Runtime Security — Behavioral Detection

### Syscall Profile Anomaly Detection

Normal container behavior forms a syscall distribution:

$$D_{normal} = \{(s_i, f_i) : s_i \text{ is syscall, } f_i \text{ is frequency}\}$$

Alert when:

$$\chi^2 = \sum_i \frac{(O_i - E_i)^2}{E_i} > \text{threshold}$$

Or when a never-before-seen syscall appears:

$$\text{Alert if } s \notin D_{normal}.\text{syscalls}$$

### Drift Detection

$$\text{Drift}(t) = |\text{files}_{runtime}(t) \setminus \text{files}_{image}|$$

An immutable container should have drift = 0 (no new executables). Any positive drift indicates potential compromise.

---

## 8. Summary of Functions by Type

| Concept | Math Type | Application |
|:---|:---|:---|
| Namespace product $\prod I_i$ | Independence multiplication | Isolation strength |
| Syscall ratio | Set cardinality | Attack surface |
| SHA-256 digest | Cryptographic hash | Layer verification |
| CPU quota/period | Ratio (fraction) | Resource limits |
| $2^n$ fork bomb | Exponential growth | PID limit justification |
| Network graph $G = (P, E)$ | Graph theory | Network policy |
| $V_0 + rt$ | Linear growth | Vulnerability accumulation |

## Prerequisites

- namespace isolation, capability bitmasks, syscall filtering, cgroup arithmetic

---

*Container security is defense in depth with shared kernel risk — namespaces, capabilities, seccomp, cgroups, and network policies each reduce the attack surface independently, and their compound effect makes container escape a multi-barrier challenge.*
