# The Mathematics of Privilege Escalation — SUID, Capabilities, and Kernel Exploitation

> *Privilege escalation exploits the gap between a user's current permissions and root. The mathematics involve SUID bit arithmetic, Linux capability bitmasks, kernel address space entropy (KASLR), and the probability calculations behind brute-forcing stack canaries and memory layout.*

---

## 1. SUID Bit Mathematics

### Permission Bitmask

Unix file permissions are a 12-bit integer:

```
Bit:  11  10   9   8 7 6   5 4 3   2 1 0
      SUID SGID Sticky  Owner(rwx)  Group(rwx)  Other(rwx)
```

$$\text{SUID} = \text{mode} \mathbin{\&} \text{0o4000} \neq 0$$

### SUID Execution Model

When a SUID binary executes:

$$\text{effective UID} = \begin{cases} \text{file owner UID} & \text{if SUID bit set} \\ \text{calling user UID} & \text{otherwise} \end{cases}$$

If file owner = root (UID 0), the process runs as root regardless of who invoked it.

### Attack Surface Enumeration

$$\text{SUID attack surface} = |\{f : (f.\text{mode} \mathbin{\&} \text{0o4000}) \land (f.\text{uid} = 0)\}|$$

| Distribution | Default SUID Root Binaries | Known Exploitable |
|:---|:---:|:---:|
| Ubuntu 22.04 | ~45 | 3-5 (historically) |
| CentOS 9 | ~35 | 2-4 |
| Debian 12 | ~40 | 3-5 |
| Alpine (container) | 3-5 | 0-1 |

### SUID Exploitation Pattern

Common SUID escalation via file read/write:

| Binary | Capability Abused | Escalation Method |
|:---|:---|:---|
| `find` | `-exec` flag | `find / -exec /bin/sh \;` |
| `vim` | Shell escape | `:!/bin/sh` |
| `nmap` (old) | Interactive mode | `!sh` |
| `env` | Command execution | `env /bin/sh` |
| Custom app | Buffer overflow | ROP chain to `execve("/bin/sh")` |

---

## 2. Linux Capabilities — Fine-Grained Privileges

### Capability Bitmask

Linux capabilities split root privileges into 41 distinct bits:

$$\text{caps} = \sum_{i=0}^{40} b_i \times 2^i$$

### Dangerous Capabilities for Escalation

| CAP Value | Name | Escalation Path | Severity |
|:---:|:---|:---|:---:|
| 0 | CAP_CHOWN | Change ownership of any file | High |
| 1 | CAP_DAC_OVERRIDE | Read/write any file | Critical |
| 5 | CAP_KILL | Signal any process | Medium |
| 6 | CAP_SETGID | Set GID to any group | High |
| 7 | CAP_SETUID | Set UID to any user | Critical |
| 12 | CAP_NET_RAW | Raw sockets (sniffing) | Medium |
| 21 | CAP_SYS_ADMIN | Mount, ptrace, BPF, namespace | Critical |
| 23 | CAP_SYS_RAWIO | Direct I/O to devices | Critical |
| 25 | CAP_SYS_PTRACE | Trace any process | High |

### Capability Sets

Each process has three capability sets:

$$\text{Effective} = \text{Permitted} \cap \text{Ambient} \cup (\text{Inheritable} \cap \text{File Inheritable})$$

Simplified: Effective is what the process CAN do. Permitted is the ceiling.

### Exploitation via CAP_SYS_ADMIN

CAP_SYS_ADMIN is the "catch-all" capability — it allows:

$$|\text{SYS\_ADMIN operations}| > 30 \text{ distinct privileges}$$

Including: mount filesystems, create namespaces, use BPF, modify kernel parameters. A process with only CAP_SYS_ADMIN can often escalate to full root.

---

## 3. Kernel Exploitation — KASLR and Canaries

### KASLR (Kernel Address Space Layout Randomization)

KASLR randomizes the kernel base address:

$$\text{kernel\_base} = \text{fixed\_base} + \text{random offset}$$

| Architecture | Entropy Bits | Possible Offsets | Brute Force |
|:---|:---:|:---:|:---:|
| x86_64 | 9 bits | 512 positions | 512 attempts |
| ARM64 | 16 bits | 65,536 positions | 65,536 attempts |
| x86 (32-bit) | 8 bits | 256 positions | 256 attempts |

### KASLR Defeat Probability

$$P(\text{guess correct offset in } k \text{ attempts}) = \frac{k}{2^n}$$

For x86_64 (9 bits): $P(\text{first try}) = \frac{1}{512} = 0.195\%$

With a kernel info leak (e.g., `/proc/kallsyms` readable, or side-channel):

$$P(\text{known offset}) = 1.0 \quad \text{(KASLR completely defeated)}$$

### Stack Canaries

A random value placed between the return address and local variables:

$$\text{canary} \in [0, 2^{64} - 1] \quad \text{(64-bit systems)}$$

### Canary Brute Force

Direct brute force: $2^{64}$ attempts — infeasible.

Byte-at-a-time brute force (if partial overwrite possible):

$$\text{Attempts} = 8 \times 256 = 2{,}048 \quad \text{(one byte at a time)}$$

| Method | Attempts | Feasibility |
|:---|:---:|:---|
| Full brute force | $2^{64}$ | Impossible |
| Byte-at-a-time | 2,048 | Feasible (if fork server) |
| Info leak | 1 | Trivial (if memory read) |
| Null canary bypass | 1 | Trivial (if format string) |

A forking server (like Apache prefork) shares the canary across children — byte-at-a-time works because each failed guess crashes only the child, not the parent.

---

## 4. Kernel Exploit Prerequisites

### Exploit Reliability Model

$$P(\text{successful exploit}) = P(\text{KASLR bypass}) \times P(\text{canary bypass}) \times P(\text{trigger}) \times P(\text{payload})$$

### Worked Example: Dirty Pipe (CVE-2022-0847)

| Factor | Probability | Reason |
|:---|:---:|:---|
| KASLR bypass | 1.0 | Not needed (no kernel addresses) |
| Canary bypass | 1.0 | Not needed (no stack overflow) |
| Trigger | 0.99 | Deterministic pipe race |
| Payload | 0.99 | File overwrite (no shellcode) |
| **Total** | **0.98** | Near-certain exploitation |

### Worked Example: Generic Kernel Stack Overflow

| Factor | Probability | Reason |
|:---|:---:|:---|
| KASLR bypass | 0.002 (no leak) or 1.0 (with leak) | 9-bit entropy |
| Canary bypass | 0.0005 (no leak) or 1.0 (with leak) | 64-bit random |
| Trigger | 0.5-0.9 | Race condition dependent |
| Payload | 0.8 | ROP chain reliability |
| **Without leaks** | **$8 \times 10^{-7}$** | Unreliable |
| **With info leaks** | **0.4-0.72** | Reliable |

**Info leaks are the keystone of kernel exploitation.** Without them, exploitation is largely impractical.

---

## 5. Common Escalation Vectors — By Probability

### Linux Local Privilege Escalation Survey

| Vector | Prevalence | Detection Difficulty | Reliability |
|:---|:---:|:---:|:---:|
| Misconfigured sudo | 30-40% | Easy | 100% |
| SUID binaries | 20-30% | Easy | 95% |
| Writable service configs | 15-25% | Medium | 90% |
| Cron job abuse | 10-20% | Medium | 85% |
| Kernel exploit | 5-15% | Hard | 50-98% |
| Container escape | 5-10% | Hard | 60-95% |
| Capability abuse | 5-10% | Medium | 90% |
| LD_PRELOAD hijack | 3-5% | Hard | 95% |
| Path hijacking | 5-10% | Medium | 90% |

### Expected Time to Escalate

$$E[T_{privesc}] = \sum_{v \in \text{vectors}} P(v \text{ exists}) \times T_{exploit}(v) \times P(\text{success}(v))$$

On an average unpatched Linux server:

$$E[T] \approx 15 \text{ minutes (experienced attacker with tools)}$$

On a hardened server (CIS Level 2, SELinux enforcing):

$$E[T] \approx 4-8 \text{ hours (if possible at all)}$$

---

## 6. Sudo Misconfiguration

### Dangerous Sudo Rules

| Rule | Risk | Escalation |
|:---|:---|:---|
| `ALL=(ALL) NOPASSWD: ALL` | Full root | `sudo su` |
| `user ALL=(ALL) /usr/bin/vim` | Shell escape | `sudo vim -c ':!/bin/sh'` |
| `user ALL=(ALL) /usr/bin/find` | Exec | `sudo find / -exec /bin/sh \;` |
| `user ALL=(ALL) /usr/bin/env` | Full exec | `sudo env /bin/sh` |
| `user ALL=(ALL) /usr/bin/python3` | Interpreter | `sudo python3 -c 'import os; os.system("/bin/sh")'` |

### Sudo Rule Combinations

With $n$ allowed commands, each potentially exploitable with probability $p$:

$$P(\text{at least one exploitable}) = 1 - (1-p)^n$$

If 10 commands are allowed with 30% individual exploitability:

$$P = 1 - 0.7^{10} = 97.2\%$$

Even seemingly safe sudo rules can be chained for escalation.

---

## 7. Container Escape Probability

### Escape Vectors

| Vector | Requires | $P(\text{exploitable})$ |
|:---|:---|:---:|
| --privileged flag | Container started privileged | 0.15 |
| Docker socket mount | /var/run/docker.sock | 0.10 |
| Host path mount | Writable host directories | 0.20 |
| Kernel exploit | Vulnerable kernel | 0.10 |
| CAP_SYS_ADMIN | Capability granted | 0.08 |
| Host PID namespace | --pid=host | 0.05 |

### Combined Escape Probability

$$P(\text{any escape}) = 1 - \prod(1 - P_i) = 1 - (0.85)(0.90)(0.80)(0.90)(0.92)(0.95) = 49.5\%$$

On an average Docker deployment, there is roughly a **50% chance** that at least one escape vector exists. Hardened deployments (rootless, no mounts, seccomp) drop this below 5%.

---

## 8. Summary of Functions by Type

| Concept | Math Type | Application |
|:---|:---|:---|
| SUID bit test $\mathbin{\&}$ 0o4000 | Bitwise AND | Privilege identification |
| Capability bitmask $2^i$ | Binary integer | Fine-grained privileges |
| KASLR $2^9$ entropy | Exponential (small) | Kernel randomization |
| Canary $2^{64}$ or $8 \times 256$ | Brute force bounds | Stack protection |
| $\prod P_i$ exploit chain | Probability product | Exploit reliability |
| $(1-p)^n$ sudo | Complement probability | Misconfiguration risk |
| Container escape $1-\prod(1-P_i)$ | Union probability | Escape likelihood |

---

*Privilege escalation is a probability calculation — the attacker surveys all vectors and needs only ONE to succeed, while the defender must block ALL of them. This fundamental asymmetry is why defense in depth (patching + SUID reduction + capabilities + MAC + audit) is the only viable strategy.*
