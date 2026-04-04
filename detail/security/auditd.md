# The Mathematics of auditd — System Call Auditing and Log Analysis

> *The Linux Audit Framework intercepts system calls in the kernel, matching them against rule sets to generate structured audit records. Performance impact is a function of rule count and syscall frequency; storage requirements grow linearly with event volume.*

---

## 1. Audit Rule Matching

### Rule Structure

Each audit rule is a predicate on system call attributes:

$$\text{rule} = (\text{syscall}, \text{filters}, \text{action}, \text{key})$$

Filter types form a conjunction (AND):

$$\text{match}(e) = \bigwedge_{i=1}^{n} f_i(e)$$

| Filter Field | Type | Example |
|:---|:---|:---|
| `-S syscall` | System call number | `open`, `execve`, `connect` |
| `-F uid=` | User ID | `uid=0` (root) |
| `-F path=` | Filesystem path | `path=/etc/shadow` |
| `-F arch=` | Architecture | `b64` (64-bit) |
| `-F perm=` | Permission type | `wa` (write + attribute change) |
| `-F success=` | Outcome | `success=0` (failures only) |

### Rule Evaluation Order

Rules are evaluated sequentially. First matching rule determines the action:

$$T_{match} = O(n_{rules}) \text{ per system call}$$

---

## 2. Performance Impact

### Syscall Overhead Model

Each audited syscall incurs overhead from rule matching and log generation:

$$T_{overhead} = T_{match} + T_{log} \times \mathbb{1}[\text{match}]$$

Where $\mathbb{1}[\text{match}]$ is 1 if the rule matches (event generated), 0 otherwise.

| Component | Cost | Notes |
|:---|:---:|:---|
| Rule matching (no match) | 0.1-1 $\mu$s | Per rule evaluated |
| Event generation | 2-10 $\mu$s | Allocate + format record |
| Userspace dispatch | 5-50 $\mu$s | kauditd → auditd via netlink |
| Disk write | 10-100 $\mu$s | Depends on I/O subsystem |

### Total System Overhead

$$\text{Overhead} = \sum_{s \in \text{syscalls}} R_s \times (n_{rules} \times T_{match} + P_s \times T_{log})$$

Where $R_s$ = rate of syscall $s$ per second, $P_s$ = probability syscall $s$ matches a rule.

### Worked Example

A web server performing 50,000 syscalls/second with 20 audit rules:

| Scenario | Overhead | CPU Impact |
|:---|:---:|:---:|
| No rules | 0 | 0% |
| 20 rules, 1% match rate | $50K \times 20 \times 0.5\mu s + 500 \times 25\mu s = 512.5$ ms/s | ~0.05% |
| 20 rules, 10% match rate | $50K \times 20 \times 0.5\mu s + 5K \times 25\mu s = 625$ ms/s | ~0.06% |
| 100 rules, 10% match rate | $50K \times 100 \times 0.5\mu s + 5K \times 25\mu s = 2,625$ ms/s | ~0.26% |
| All syscalls audited | $50K \times 25\mu s = 1,250$ ms/s | ~0.13% |

**Key insight:** Rule matching cost dominates over event generation when match rate is low.

---

## 3. Log Volume Estimation

### Event Size

$$\text{Event size} = \text{header} + \text{syscall record} + \sum \text{auxiliary records}$$

| Record Type | Typical Size | Content |
|:---|:---:|:---|
| SYSCALL | 200-300 bytes | Syscall args, uid, gid, pid |
| PATH | 100-200 bytes | Filename, inode, dev |
| CWD | 50-100 bytes | Current working directory |
| EXECVE | 100-500 bytes | Command + arguments |
| SOCKADDR | 50-100 bytes | Network address |
| PROCTITLE | 50-200 bytes | Process command line |

Average event: ~400 bytes (varies with auxiliary records).

### Daily Volume

$$V_{daily} = \sum_{i=1}^{n_{rules}} R_i \times S_i \times 86400$$

| Audit Scope | Events/sec | Daily Volume | Monthly (30d) |
|:---|:---:|:---:|:---:|
| Logins only | 0.1 | 3.5 MB | 104 MB |
| File integrity (critical) | 10 | 346 MB | 10.1 GB |
| Process execution | 50 | 1.7 GB | 51 GB |
| All syscalls | 5,000 | 173 GB | 5.2 TB |

### Compression

Audit logs compress well (repetitive structured text):

$$\text{Compressed} \approx V_{raw} \times 0.15 \text{ (85% compression with gzip)}$$

| Raw Volume | Compressed | Storage Savings |
|:---:|:---:|:---:|
| 1 GB/day | 150 MB/day | 85% |
| 10 GB/day | 1.5 GB/day | 85% |
| 100 GB/day | 15 GB/day | 85% |

---

## 4. Backlog and Queue Mathematics

### Kernel Audit Backlog

The kernel maintains a backlog queue for events awaiting userspace dispatch:

$$\text{Queue state} = \frac{\text{event arrival rate}}{\text{dispatch rate}}$$

If arrival rate > dispatch rate, the queue fills:

$$T_{queue\_full} = \frac{\text{backlog\_limit}}{\text{arrival rate} - \text{dispatch rate}}$$

Default `backlog_limit = 8192` events.

| Arrival Rate | Dispatch Rate | Time to Full | Action |
|:---:|:---:|:---:|:---|
| 1,000/s | 5,000/s | Never | Healthy |
| 5,000/s | 5,000/s | Equilibrium | At capacity |
| 10,000/s | 5,000/s | 1.6 seconds | Events lost or system stalls |
| 50,000/s | 5,000/s | 0.18 seconds | Immediate overflow |

### Failure Modes

$$\text{failure\_action} = \begin{cases} \text{SYSLOG} & \text{log warning, continue (drop events)} \\ \text{IGNORE} & \text{silently drop events} \\ \text{PANIC} & \text{halt the system} \end{cases}$$

High-security systems use `failure_action = PANIC` — losing audit events is considered worse than a system outage.

---

## 5. Audit Rule Design Patterns

### File Integrity Monitoring

Watch critical files for changes:

```
-w /etc/passwd -p wa -k identity
-w /etc/shadow -p wa -k identity
-w /etc/sudoers -p wa -k privilege
```

Event rate for file watches:

$$R_{watch} = \sum_{f \in \text{watched}} R_{access}(f) \times P(\text{write or attr change})$$

### Syscall Monitoring

Monitor specific system calls:

```
-a always,exit -F arch=b64 -S execve -k exec
-a always,exit -F arch=b64 -S connect -k network
-a always,exit -F arch=b64 -S ptrace -k tracing
```

### Rule Count Optimization

More targeted rules = less overhead:

$$\text{Overhead} \propto n_{rules} \times R_{syscalls}$$

| Approach | Rules | Events/day | Coverage |
|:---|:---:|:---:|:---|
| Audit everything | 1 (catch-all) | 500M+ | 100% (unusable noise) |
| Broad syscall classes | 20-30 | 5-50M | 80% |
| Targeted (CIS/STIG) | 40-60 | 500K-5M | 95% of attacks |
| Minimal (compliance) | 10-15 | 50K-500K | 60% |

---

## 6. Log Correlation — Event Reconstruction

### Event Grouping

Related audit records share a serial number and timestamp:

$$\text{Event}(s) = \{r : r.\text{serial} = s\}$$

A single `execve` event may generate 3-6 records:

$$|\text{Event}(s)| = 1(\text{SYSCALL}) + 1(\text{EXECVE}) + n(\text{PATH}) + 1(\text{CWD}) + 1(\text{PROCTITLE})$$

### Process Tree Reconstruction

Using `ppid` chains from SYSCALL records:

$$\text{tree}(p) = p \rightarrow \text{ppid}(p) \rightarrow \text{ppid}(\text{ppid}(p)) \rightarrow \cdots \rightarrow \text{init}(1)$$

Depth of process tree: typically 3-10 levels.

### Timeline Correlation

Events are ordered by timestamp (epoch seconds + milliseconds):

$$t_1 < t_2 \iff (t_1.\text{epoch} < t_2.\text{epoch}) \lor (t_1.\text{epoch} = t_2.\text{epoch} \land t_1.\text{serial} < t_2.\text{serial})$$

---

## 7. Compliance Mapping

### CIS Benchmark Audit Rules

| CIS Control | Audit Rule | Events/Day (est.) |
|:---|:---|:---:|
| 4.1.4 — Login/logout | `-w /var/log/lastlog` | ~500 |
| 4.1.5 — DAC changes | `-S chmod,chown,fchmod` | ~10,000 |
| 4.1.6 — File access | `-S open,openat -F exit=-EACCES` | ~5,000 |
| 4.1.7 — User/group mods | `-w /etc/passwd -p wa` | ~50 |
| 4.1.8 — Mount operations | `-S mount,umount2` | ~100 |
| 4.1.11 — Privileged cmds | `-S execve -C uid!=euid` | ~1,000 |

Total CIS audit event volume: ~15,000-50,000 events/day on a typical server.

---

## 8. Summary of Functions by Type

| Concept | Math Type | Application |
|:---|:---|:---|
| Rule conjunction $\bigwedge f_i$ | Boolean logic | Event matching |
| $O(n_{rules})$ evaluation | Linear scan | Per-syscall overhead |
| Queue $\lambda / \mu$ | Queueing theory | Backlog management |
| $V = R \times S \times t$ | Linear growth | Storage planning |
| Compression ratio 0.15 | Data compression | Storage optimization |
| Process tree | Directed tree (DAG) | Attack reconstruction |
| Serial ordering | Total order | Timeline correlation |

## Prerequisites

- system call interface, rule matching, queue theory, log correlation

---

*auditd transforms every system call into a court-admissible record — the mathematics of rule matching, queue management, and storage planning determine whether your audit trail is comprehensive or overwhelmed.*
