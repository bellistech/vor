# The Mathematics of Linux Hardening — Attack Surface Quantification

> *Hardening is the systematic reduction of attack surface. Each disabled service, removed package, and tightened permission reduces the probability of compromise — measurable through CIS scoring, exposure metrics, and defense-in-depth probability models.*

---

## 1. CIS Benchmark Scoring

### Scoring Formula

The Center for Internet Security (CIS) benchmark assigns a compliance score:

$$S = \frac{\text{Controls Passed}}{\text{Total Applicable Controls}} \times 100\%$$

### CIS Levels

| Level | Description | Typical Controls | Target Audience |
|:---:|:---|:---:|:---|
| Level 1 | Basic hardening, minimal perf impact | ~180 | All systems |
| Level 2 | Defense in depth, may reduce functionality | ~250 | High-security |
| STIG | DoD Security Technical Implementation Guide | ~400 | Military/gov |

### Worked Example (Ubuntu 22.04 CIS)

A fresh Ubuntu install might pass ~60 of 180 Level 1 controls:

$$S_{initial} = \frac{60}{180} = 33\%$$

After hardening:

$$S_{hardened} = \frac{172}{180} = 95.6\%$$

The 8 remaining failures might be justified exceptions (documented risk acceptance).

---

## 2. Attack Surface Quantification

### Attack Surface Formula

$$A = \sum_{i=1}^{n} w_i \cdot e_i$$

Where:
- $A$ = total attack surface score
- $n$ = number of exposed components
- $w_i$ = weight (criticality) of component $i$
- $e_i$ = exposure level of component $i$ (0 = not exposed, 1 = fully exposed)

### Exposure Components

| Component | Weight ($w$) | Before Hardening ($e$) | After Hardening ($e'$) |
|:---|:---:|:---:|:---:|
| Network services | 10 | 1.0 (22 services) | 0.2 (4 services) |
| SUID binaries | 8 | 1.0 (45 binaries) | 0.3 (12 binaries) |
| Writable directories | 5 | 0.8 | 0.2 |
| Kernel modules | 7 | 1.0 (200+ loaded) | 0.4 (80 loaded) |
| User accounts | 6 | 0.7 (interactive) | 0.2 (locked shells) |
| Open ports | 9 | 1.0 | 0.15 |

$$A_{before} = 10(1) + 8(1) + 5(0.8) + 7(1) + 6(0.7) + 9(1) = 42.2$$
$$A_{after} = 10(0.2) + 8(0.3) + 5(0.2) + 7(0.4) + 6(0.2) + 9(0.15) = 10.55$$

**Reduction:** $\frac{42.2 - 10.55}{42.2} = 75\%$ attack surface reduction.

---

## 3. Defense in Depth — Layered Probability Model

### The Principle

Each security layer is an independent barrier. An attacker must bypass ALL layers:

$$P(\text{breach}) = \prod_{i=1}^{n} P(\text{bypass layer } i)$$

### Worked Example: SSH Access

| Layer | Control | $P(\text{bypass})$ |
|:---:|:---|:---:|
| 1 | Firewall (port 22 restricted to VPN) | 0.05 |
| 2 | fail2ban (rate limiting) | 0.3 |
| 3 | Key-only auth (no passwords) | 0.001 |
| 4 | Non-root login + sudo | 0.2 |
| 5 | SELinux/AppArmor confinement | 0.1 |
| 6 | Audit logging + alerting | 0.8 (detection, not prevention) |

$$P(\text{breach}) = 0.05 \times 0.3 \times 0.001 \times 0.2 \times 0.1 = 3 \times 10^{-8}$$

Each additional layer multiplies the difficulty. Even imperfect layers compound.

### Layer Independence Assumption

If layers share a common vulnerability (e.g., same kernel exploit bypasses SELinux + AppArmor):

$$P(\text{breach}) > \prod P_i$$

True defense in depth requires **diverse** mechanisms (network + host + application + human).

---

## 4. SUID Bit Mathematics

### File Permission Encoding

Unix permissions are a 12-bit field:

```
SUID SGID Sticky | Owner(rwx) | Group(rwx) | Other(rwx)
  1    1     1   |  1 1 1     |  1 1 1     |  1 1 1
```

$$\text{Octal} = 4096 \times S_{uid} + 2048 \times S_{gid} + 1024 \times T + 8^2 \times O + 8^1 \times G + 8^0 \times W$$

### SUID Attack Surface

SUID root binaries execute with root privileges regardless of caller:

$$\text{SUID attack surface} = |\{f : f.\text{suid} = 1 \land f.\text{owner} = \text{root}\}|$$

Typical counts:

| Distribution | Default SUID Count | Hardened Count |
|:---|:---:|:---:|
| Ubuntu 22.04 | ~45 | ~12 |
| CentOS 9 | ~35 | ~10 |
| Alpine Linux | ~5 | ~3 |
| Minimal container | 0-2 | 0 |

Each SUID binary is a potential privilege escalation vector. Finding them:

```bash
find / -perm -4000 -type f 2>/dev/null | wc -l
```

---

## 5. Kernel Hardening Parameters

### sysctl Security Parameters

| Parameter | Default | Hardened | Effect |
|:---|:---:|:---:|:---|
| `kernel.randomize_va_space` | 2 | 2 | ASLR (address space layout randomization) |
| `kernel.kptr_restrict` | 0 | 2 | Hide kernel pointers |
| `kernel.dmesg_restrict` | 0 | 1 | Restrict dmesg to root |
| `kernel.yama.ptrace_scope` | 0 | 2 | Restrict ptrace |
| `net.ipv4.conf.all.rp_filter` | 0 | 1 | Reverse path filtering |
| `net.ipv4.tcp_syncookies` | 1 | 1 | SYN flood protection |
| `fs.protected_hardlinks` | 1 | 1 | Hardlink protection |
| `fs.protected_symlinks` | 1 | 1 | Symlink protection |

### ASLR Entropy

Address Space Layout Randomization adds entropy to memory addresses:

$$\text{Entropy}_{ASLR} = \log_2(\text{randomization range})$$

| Component | Bits of Entropy (x86_64) | Brute Force Attempts |
|:---|:---:|:---:|
| Stack | 22 bits | $2^{22} = 4.2M$ |
| mmap/libraries | 28 bits | $2^{28} = 268M$ |
| Heap | 13 bits | $2^{13} = 8K$ |
| PIE executable | 28 bits | $2^{28} = 268M$ |
| KASLR (kernel) | 9 bits | $2^9 = 512$ |

KASLR's 9 bits is notably weak — a kernel info leak can defeat it easily.

---

## 6. Password Policy Mathematics

### Entropy of Password Policies

$$H = L \times \log_2(C)$$

Where $L$ = minimum length, $C$ = character set size.

| Policy | Charset ($C$) | Min Length ($L$) | Entropy ($H$) |
|:---|:---:|:---:|:---:|
| Lowercase only | 26 | 8 | 37.6 bits |
| Mixed case | 52 | 8 | 45.6 bits |
| Mixed + digits | 62 | 8 | 47.6 bits |
| Mixed + digits + symbols | 95 | 8 | 52.6 bits |
| Mixed + digits + symbols | 95 | 12 | 78.8 bits |
| Mixed + digits + symbols | 95 | 16 | 105.1 bits |
| 4-word diceware | 7776 words | 4 | 51.7 bits |
| 6-word diceware | 7776 words | 6 | 77.5 bits |

### PAM Password Complexity (pam_pwquality)

```
minlen=12 minclass=3 dcredit=-1 ucredit=-1 lcredit=-1 ocredit=-1
```

This enforces: 12+ characters, 3+ character classes, at least 1 digit, 1 upper, 1 lower, 1 special.

---

## 7. Firewall Rule Ordering

### Rule Evaluation Cost

Firewall rules are evaluated sequentially (first-match wins):

$$T_{eval} = \sum_{i=1}^{k} T_i \quad \text{where } k = \text{index of matching rule}$$

### Optimal Rule Ordering

Place high-frequency rules first to minimize average evaluation:

$$T_{avg} = \sum_{i=1}^{n} p_i \cdot i \cdot T_{rule}$$

Where $p_i$ is the probability of matching rule $i$.

**Example:** 3 rules with match probabilities 0.7, 0.2, 0.1:

Optimal order (descending probability):
$$T_{avg} = 0.7(1) + 0.2(2) + 0.1(3) = 1.4 \cdot T_{rule}$$

Worst order (ascending probability):
$$T_{avg} = 0.1(1) + 0.2(2) + 0.7(3) = 2.6 \cdot T_{rule}$$

The optimal ordering is **86% faster** for this example.

---

## 8. Audit Log Volume Estimation

### Log Growth Formula

$$V_{daily} = \sum_{i=1}^{n} R_i \times S_i$$

Where $R_i$ = events per day for rule $i$, $S_i$ = average event size in bytes.

| Audit Rule | Events/Day | Avg Size | Daily Volume |
|:---|:---:|:---:|:---:|
| Login/logout | 500 | 200 B | 100 KB |
| File access (sensitive) | 10,000 | 300 B | 3 MB |
| Syscall monitoring | 1,000,000 | 150 B | 150 MB |
| Process execution | 50,000 | 250 B | 12.5 MB |
| Network connections | 100,000 | 200 B | 20 MB |

### Storage Planning

$$\text{Storage}_{30d} = V_{daily} \times 30 \times (1 + \text{compression overhead})$$

With syscall monitoring: $185 \text{ MB/day} \times 30 = 5.6 \text{ GB/month}$

Without syscall monitoring: $35 \text{ MB/day} \times 30 = 1.05 \text{ GB/month}$

**Decision:** Broad syscall monitoring is 5x more storage but catches insider threats that other rules miss.

---

## 9. Summary of Functions by Type

| Concept | Math Type | Application |
|:---|:---|:---|
| CIS scoring | Ratio (pass/total) | Compliance measurement |
| Attack surface | Weighted sum | Exposure quantification |
| Defense in depth | Probability product | Layered security |
| SUID count | Set cardinality | Privilege escalation surface |
| ASLR entropy | $\log_2(\text{range})$ | Memory randomization |
| Password entropy | $L \times \log_2(C)$ | Policy strength |
| Firewall ordering | Weighted expected value | Rule optimization |

---

*Hardening is not a checklist — it's an engineering discipline. Each control reduces a measurable probability, and the compound effect of layered defenses makes breach exponentially harder.*
