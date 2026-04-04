# The Mathematics of cloud-init — Instance Provisioning Theory

> *cloud-init is the industry standard for VM initialization in every major cloud. It runs at boot time through a multi-stage pipeline, processing metadata and userdata to configure instances. Its execution model involves boot stage ordering, module dependency resolution, and datasource polling.*

---

## 1. Boot Stage Pipeline (Sequential Execution Model)

### The Problem

cloud-init runs in 5 distinct boot stages, each at a specific systemd ordering point. Understanding the pipeline is essential for debugging boot timing.

### The Five Stages

| Stage | Systemd Target | Purpose | Typical Duration |
|:---|:---|:---|:---:|
| Generator | Very early boot | Detect cloud environment | < 1s |
| Local | `cloud-init-local.service` | Network-independent config | 1-3s |
| Network | `cloud-init.service` | Metadata fetch, SSH keys | 2-10s |
| Config | `cloud-config.service` | Package install, runcmd | 10-300s |
| Final | `cloud-final.service` | User scripts, phone home | 1-60s |

### Total Boot Time

$$T_{boot} = T_{kernel} + T_{systemd} + \sum_{s=1}^{5} T_{stage_s}$$

### Worked Example

| Component | Time |
|:---|:---:|
| Kernel boot | 3s |
| Systemd to cloud-init-local | 2s |
| Local stage | 1s |
| Network stage | 5s |
| Config stage (apt install) | 45s |
| Final stage (user script) | 10s |
| **Total** | **66s** |

### Boot Ordering Constraints

$$T_{local} < T_{network} < T_{config} < T_{final}$$

Each stage is a hard barrier — stage $k+1$ cannot start until stage $k$ completes.

---

## 2. Datasource Polling (Metadata Discovery)

### The Problem

cloud-init must discover and poll the correct metadata source. Different clouds have different endpoints and polling strategies.

### Metadata Fetch

$$T_{metadata} = T_{discovery} + T_{fetch} + T_{parse}$$

### Datasource Priority

cloud-init checks datasources in priority order:

$$\text{source} = \text{first}(\{ds : ds.\text{available()} = \text{true}\})$$

| Cloud | Datasource | Endpoint | Fetch Time |
|:---|:---|:---|:---:|
| AWS | IMDS v2 | `169.254.169.254` | 1-5 ms |
| GCP | GCE | `metadata.google.internal` | 1-5 ms |
| Azure | IMDS | `169.254.169.254` | 5-50 ms |
| OpenStack | ConfigDrive or HTTP | Link-local or disk | 1-1000 ms |
| NoCloud | Local disk | `/dev/sr0` or seed dir | 10-100 ms |

### IMDS Token Lifecycle (AWS IMDSv2)

$$T_{token} = T_{PUT\_request} + T_{response}$$

Token TTL: configurable, max 21,600s (6 hours).

$$\text{Requests before refresh} = \frac{\text{TTL}}{T_{avg\_between\_requests}}$$

---

## 3. Module Execution (Frequency Control)

### The Problem

cloud-init modules run at specified frequencies — once per instance, once per boot, or always.

### Frequency Model

$$\text{runs}(m, t) = \begin{cases}
1 & \text{if freq} = \text{once} \\
\text{boots}(t) & \text{if freq} = \text{always} \\
\text{boots\_per\_instance}(t) & \text{if freq} = \text{once-per-instance}
\end{cases}$$

### Module Idempotency

Modules with `once` frequency use a semaphore file:

$$\text{run}(m) = \begin{cases}
\text{execute + create sem} & \text{if semaphore absent} \\
\text{skip} & \text{if semaphore present}
\end{cases}$$

$$f(f(x)) = f(x) \quad \text{(enforced by semaphore, even for non-idempotent modules)}$$

### Common Module Timing

| Module | Frequency | Typical Time | Stage |
|:---|:---:|:---:|:---|
| ssh | once-per-instance | 0.5s | Network |
| apt_configure | once-per-instance | 1s | Config |
| package_update_upgrade | once-per-instance | 30-120s | Config |
| runcmd | once-per-instance | Variable | Config |
| write_files | once-per-instance | < 0.1s | Config |
| phone_home | once-per-instance | 1-5s | Final |

---

## 4. Userdata Processing (MIME Multi-Part)

### The Problem

cloud-init supports multiple userdata formats. MIME multi-part allows combining them.

### Format Detection

$$\text{format}(\text{userdata}) = \begin{cases}
\text{cloud-config} & \text{if starts with \#cloud-config} \\
\text{shell script} & \text{if starts with \#!/} \\
\text{MIME multipart} & \text{if Content-Type: multipart/mixed} \\
\text{include} & \text{if starts with \#include} \\
\text{gzip} & \text{if gzip magic bytes}
\end{cases}$$

### Userdata Size Limits

$$S_{userdata} \leq S_{max}$$

| Cloud | Max Userdata Size |
|:---|:---:|
| AWS | 16 KB (raw) / 64 KB (base64) |
| GCP | 256 KB |
| Azure | 64 KB (custom data) |
| OpenStack | 65,535 bytes |

### Compression Savings

For large userdata, gzip is essential:

$$S_{compressed} = S_{raw} \times R_{compression}$$

| Content Type | Raw Size | Compressed | Ratio |
|:---|:---:|:---:|:---:|
| Shell scripts | 10 KB | 3 KB | 0.30 |
| Cloud-config (YAML) | 8 KB | 2.5 KB | 0.31 |
| Binary data | 15 KB | 12 KB | 0.80 |

---

## 5. Network Configuration (Renderers)

### The Problem

cloud-init abstracts network configuration across multiple renderers (netplan, ENI, sysconfig, NetworkManager).

### Configuration Pipeline

$$\text{Cloud metadata} \xrightarrow{\text{parse}} \text{cloud-init v2 config} \xrightarrow{\text{render}} \text{OS-specific config}$$

### Renderer Selection

$$\text{renderer} = \begin{cases}
\text{netplan} & \text{Ubuntu 18.04+} \\
\text{ENI} & \text{Debian, older Ubuntu} \\
\text{sysconfig} & \text{RHEL/CentOS} \\
\text{NetworkManager} & \text{Fedora, RHEL 8+}
\end{cases}$$

### Network Apply Time

$$T_{network} = T_{render} + T_{apply} + T_{dhcp\_or\_static}$$

| Method | Apply Time |
|:---|:---:|
| DHCP | 2-10s (waiting for lease) |
| Static | < 1s |
| DHCP + static routes | 3-12s |

---

## 6. Instance Identity (Instance-ID Lifecycle)

### The Problem

cloud-init uses `instance-id` to determine if it's a fresh instance or a reboot. This controls module re-execution.

### Instance Lifecycle

$$\text{instance\_change} = (\text{current\_id} \neq \text{stored\_id})$$

$$\text{if instance\_change:}$$
$$\quad \text{reset all once-per-instance semaphores}$$
$$\quad \text{re-run full initialization}$$

### Scenarios

| Event | Instance ID Changes? | Full Re-Init? |
|:---|:---:|:---:|
| Reboot | No | No |
| Stop/Start (same disk) | No (usually) | No |
| Rebuild (new disk) | Yes | Yes |
| Clone/Image | Yes | Yes |
| Cloud restore | Depends | Depends |

---

## 7. Performance Optimization (Boot Time Reduction)

### The Problem

cloud-init can add 30-300 seconds to boot. Optimizing it requires understanding where time is spent.

### Profiling Formula

$$T_{cloud-init} = \sum_{m \in \text{modules}} T_m$$

### Optimization Techniques

| Technique | Savings | Mechanism |
|:---|:---:|:---|
| Pre-bake packages in AMI | 30-120s | Skip `package_update_upgrade` |
| Disable unused modules | 1-5s | Skip `phone_home`, `landscape`, etc. |
| Use `write_files` over `runcmd` | 0.5-2s | No shell fork overhead |
| Minimize apt sources | 5-15s | Fewer DNS lookups and HTTP fetches |
| Use compressed userdata | 0.1-0.5s | Less metadata transfer |

### Total Optimization

$$T_{optimized} = T_{baseline} - \sum T_{savings}$$

Typical: from 120s to 15s by pre-baking an AMI with all packages.

---

## 8. Summary of Functions by Type

| Formula | Math Type | Domain |
|:---|:---|:---|
| $\sum T_{stage}$ | Sequential summation | Boot pipeline |
| First available datasource | Priority search | Metadata discovery |
| Semaphore-gated execution | Idempotency | Module frequency |
| $S_{raw} \times R_{compression}$ | Ratio | Userdata sizing |
| $\text{current\_id} \neq \text{stored\_id}$ | Equality test | Instance lifecycle |
| $T_{baseline} - \sum T_{savings}$ | Subtraction | Optimization |

---

*cloud-init runs on virtually every cloud VM in existence — AWS, GCP, Azure, OpenStack, and more. Its 5-stage boot pipeline, metadata polling, and module frequency system transform a blank VM into a configured server in under a minute.*

## Prerequisites

- Linux boot process (systemd, init stages)
- YAML syntax (for cloud-config format)
- Cloud instance metadata concepts (datasources)
- Basic networking (for network-config)

## Complexity

- Beginner: user creation, package installation, runcmd
- Intermediate: write_files, disk setup, network configuration, multi-part MIME
- Advanced: boot stage pipeline optimization, datasource internals, module frequency control, NoCloud seed ISO creation
