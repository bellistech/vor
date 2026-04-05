# Linux Kernel Hardening — Exploit Mitigation Theory, ASLR Entropy, and Defense in Depth

> *Deep dive into kernel hardening internals: ASLR entropy analysis per memory region, attack surface quantification, ROP/JOP mitigation effectiveness, Control Flow Integrity theory, and the defense-in-depth model that layers these protections. Math and theory behind the tunables.*

---

## Prerequisites

- Understanding of x86-64 virtual memory layout (canonical addresses, page tables, user/kernel split)
- Familiarity with Linux kernel hardening tunables (`sysctl`, boot parameters, LSM)
- Basic information theory (entropy, bits of randomness)
- Assembly-level understanding of control flow (call, ret, indirect jumps)

## Complexity

| Topic | Analysis Type | Key Metric |
|:---|:---|:---|
| ASLR Entropy | Information-theoretic | Bits of randomness per region |
| Attack Surface | Combinatorial | Reachable syscall/ioctl surface area |
| ROP/JOP Mitigation | Probabilistic | Gadget survival rate under KASLR |
| CFI | Graph-theoretic | Forward/backward edge coverage |
| Defense in Depth | Layered model | Independent bypass probability product |

---

## 1. ASLR Entropy Analysis

### Virtual Address Space Layout

On x86-64 Linux, ASLR randomizes the base address of each memory region independently. The entropy (bits of randomness) determines brute-force resistance:

$$\text{Brute-force attempts} = 2^{H}$$

where $H$ is the entropy in bits for a given region.

### Per-Region Entropy (x86-64, 4-level paging)

| Region | Default Entropy | Alignment | Source |
|:---|:---|:---|:---|
| Stack | 22 bits | 16 bytes | `arch/x86/mm/mmap.c` |
| mmap base | 28 bits | page (4 KB) | `mmap_rnd_bits` sysctl |
| Executable (PIE) | 28 bits | page (4 KB) | ET_DYN ELF loader |
| Heap (brk) | 13 bits | page (4 KB) | `arch_randomize_brk()` |
| VDSO | 11 bits | page (4 KB) | `vdso_addr()` |
| Kernel text (KASLR) | ~9 bits | 2 MB (PMD) | boot-time, 512 slots |

The mmap entropy is configurable:

```
# Default x86-64: 28 bits for mmap, 32 bits max
sysctl vm.mmap_rnd_bits       # default: 28
sysctl vm.mmap_rnd_compat_bits  # for 32-bit compat: default 8
```

### Entropy Lower Bound for Security

For a remote exploit with no information leak, the probability of guessing a single region correctly:

$$P(\text{guess}) = 2^{-H}$$

For an attack requiring $k$ independent region addresses (e.g., stack + libc + heap):

$$P(\text{guess all}) = \prod_{i=1}^{k} 2^{-H_i} = 2^{-\sum H_i}$$

Example: attacking a PIE binary with ASLR, needing stack (22 bits) + mmap/libc (28 bits):

$$P = 2^{-(22+28)} = 2^{-50} \approx 8.9 \times 10^{-16}$$

This makes blind brute-force infeasible even at 1000 attempts/second:

$$\text{Expected time} = \frac{2^{50}}{1000 \cdot 3600 \cdot 24 \cdot 365} \approx 35,702 \text{ years}$$

### KASLR Entropy Weakness

Kernel KASLR provides only ~9 bits of entropy (512 possible offsets in the physical mapping area). This is significantly weaker than userspace ASLR:

$$P(\text{KASLR guess}) = 2^{-9} = \frac{1}{512} \approx 0.2\%$$

An attacker with ~512 attempts (or a single info leak via `/proc/kallsyms`, dmesg, or side-channel) defeats KASLR entirely. This is why `kptr_restrict=2` and `dmesg_restrict=1` are essential companions.

### Per-Syscall Kernel Stack Randomization

The `randomize_kstack_offset=on` boot parameter adds a random offset (0-255 bytes) to the kernel stack on each syscall entry:

$$H_{\text{kstack}} = \log_2(256) = 8 \text{ bits}$$

This makes stack-based kernel exploits position-dependent across syscalls, even when KASLR base is known.

---

## 2. Attack Surface Reduction Quantification

### Syscall Surface Area

The Linux kernel exposes ~450 syscalls on x86-64. Each is a potential entry point for exploits. The hardened surface after restriction:

$$S_{\text{effective}} = S_{\text{total}} - S_{\text{blocked}} - S_{\text{filtered}}$$

With seccomp-BPF, a typical container profile (Docker default) blocks ~44 syscalls. A strict profile (like gVisor's) permits only ~70:

$$\text{Reduction factor} = \frac{S_{\text{allowed}}}{S_{\text{total}}} = \frac{70}{450} \approx 15.6\%$$

### Module Attack Surface

Each loaded kernel module adds code to ring 0. The attack surface contribution:

$$A_{\text{module}} = \text{LOC}_{\text{module}} \times d_{\text{vuln}}$$

where $d_{\text{vuln}}$ is the historical vulnerability density (CVEs per KLOC per year). For Linux kernel code, empirical data suggests $d_{\text{vuln}} \approx 0.5\text{--}2.0$ CVEs/KLOC/year.

A minimal server with 50 loaded modules (~2M LOC total) versus a default install with 200 modules (~8M LOC):

$$\frac{A_{200}}{A_{50}} = \frac{8000 \times d}{2000 \times d} = 4\times \text{ attack surface}$$

Module count minimization strategies:
- `kernel.modules_disabled=1` after init (zero new modules)
- `CONFIG_MODULE_SIG_FORCE=y` (only signed modules)
- Blacklisting unused modules in `/etc/modprobe.d/`
- Custom kernel with `make localmodconfig` (build only currently loaded modules)

### Ioctl Surface

Each device driver exposes ioctl commands, historically the richest exploit surface in the kernel. The attack surface per driver:

$$A_{\text{ioctl}} = \sum_{i=1}^{N_{\text{cmds}}} C_i \cdot R_i$$

where $C_i$ is the complexity of handler $i$ and $R_i$ is its reachability from unprivileged context. Device cgroup (used in containers) restricts $R_i$ to zero for blocked device nodes.

---

## 3. ROP/JOP Mitigation Effectiveness

### ROP (Return-Oriented Programming) Basics

ROP chains existing code fragments ("gadgets") ending in `ret` to build arbitrary computation without injecting code. Each gadget is a short instruction sequence:

$$G = \{g_1, g_2, \ldots, g_n\} \quad \text{where } g_i = (\text{addr}_i, \text{instructions}_i, \texttt{ret})$$

The attacker needs to know the address of each gadget, making ASLR the primary defense.

### Gadget Survival Under KASLR

With KASLR providing $H$ bits of entropy, a gadget at known offset $\delta$ from kernel base has:

$$P(\text{gadget at expected addr}) = 2^{-H}$$

For a ROP chain of length $n$ (all from the same module, so same KASLR slide):

$$P(\text{chain works}) = 2^{-H} \quad \text{(single slide applies to all gadgets in same image)}$$

This is why KASLR alone (9 bits) is insufficient. The probability of a successful single-guess ROP attack is $2^{-9} \approx 0.2\%$.

### SMEP/SMAP Impact on ROP

Without SMEP, the attacker can `ret` to user-mapped executable pages (ret2usr). SMEP forces the attacker to use kernel-space gadgets only.

Without SMAP, the attacker can read/write user-space from kernel context to construct fake kernel objects. SMAP forces the attacker to find kernel-space read/write primitives.

The combined effect on available attack techniques:

| Mitigation | Technique Blocked | Remaining Techniques |
|:---|:---|:---|
| None | — | ret2usr, fake structs in userspace, kernel ROP |
| SMEP only | ret2usr | Kernel ROP, fake structs in userspace |
| SMEP + SMAP | ret2usr + user data access | Kernel ROP only |
| SMEP + SMAP + KASLR | + address guessing | Kernel ROP with info leak |

### JOP (Jump-Oriented Programming)

JOP uses indirect jump/call gadgets instead of `ret`. The dispatcher gadget pattern:

$$\texttt{mov reg, [table + idx*8]; jmp reg}$$

JOP is harder to chain but survives return-address-focused defenses like shadow stacks. CFI (Section 4) is the primary defense against JOP.

---

## 4. Control Flow Integrity (CFI) Theory

### The CFI Problem

A program's **Control Flow Graph** (CFG) is a directed graph $G = (V, E)$ where vertices are basic blocks and edges are valid control flow transfers. An exploit corrupts an indirect branch to target a node $v' \notin \text{successors}(v)$.

$$\text{CFI violation: } (v, v') \notin E \text{ but execution follows } v \to v'$$

CFI enforces that every indirect branch target is a valid edge in the CFG.

### Forward-Edge CFI

Forward-edge CFI protects indirect calls and jumps (not returns). The kernel implementation (Clang CFI / `CONFIG_CFI_CLANG`):

For each indirect call site, the compiler inserts a check:

$$\text{target} \in \mathcal{T}(\text{call site}) \quad \text{where } \mathcal{T} = \{f : \text{type}(f) = \text{type}_{\text{expected}}\}$$

The granularity of $\mathcal{T}$ determines security:

- **Coarse-grained**: $\mathcal{T}$ = all functions (type-agnostic). Large set, many valid-looking targets.
- **Type-based** (kCFI): $\mathcal{T}$ = functions matching the call site's function pointer type. Much smaller set.

$$|\mathcal{T}_{\text{coarse}}| \gg |\mathcal{T}_{\text{type}}| \gg 1$$

Linux kernel uses kCFI (since 6.1), which checks function type hashes:

$$h(\text{target\_type}) \stackrel{?}{=} h(\text{expected\_type})$$

The reduction in valid targets:

$$\text{Gadget reduction} = 1 - \frac{|\mathcal{T}_{\text{kCFI}}|}{|\mathcal{T}_{\text{coarse}}|}$$

Empirically, kCFI reduces valid indirect call targets by 90-99% depending on the function signature.

### Backward-Edge CFI — Shadow Stacks

Backward-edge CFI protects return addresses. Intel CET (Control-flow Enforcement Technology) provides hardware shadow stacks:

$$\text{On } \texttt{call}: \text{push } \texttt{ret\_addr} \text{ to shadow stack (SSP)}$$
$$\text{On } \texttt{ret}: \text{compare } \texttt{ret\_addr}_{\text{shadow}} \stackrel{?}{=} \texttt{ret\_addr}_{\text{primary}}$$

A mismatch triggers `#CP` (Control Protection) exception. This makes ROP chains require corrupting both stacks simultaneously.

$$P(\text{ROP with shadow stack}) = P(\text{corrupt primary}) \times P(\text{corrupt shadow})$$

Since the shadow stack is in a separate, supervisor-mode-only memory region:

$$P(\text{corrupt shadow}) \approx 0 \quad \text{(without a separate vulnerability)}$$

### CFI Coverage Model

Full CFI coverage has three components:

$$\text{CFI}_{\text{total}} = \text{CFI}_{\text{forward}} \times \text{CFI}_{\text{backward}} \times \text{CFI}_{\text{exception}}$$

| Component | Mechanism | Linux Status |
|:---|:---|:---|
| Forward edge | kCFI (Clang) | `CONFIG_CFI_CLANG` (6.1+) |
| Backward edge | Shadow Stack (CET) | `CONFIG_X86_USER_SHADOW_STACK` (6.6+) |
| Exception edge | Safe stack unwinding | Partial (ORC unwinder) |

---

## 5. Defense-in-Depth — Layered Mitigation Model

### Independent Layer Assumption

The defense-in-depth model assumes $n$ independent mitigation layers. An attacker must bypass all of them:

$$P(\text{exploit}) = \prod_{i=1}^{n} P(\text{bypass layer } i)$$

If each layer independently blocks with probability $p_i$, the overall bypass probability:

$$P(\text{bypass all}) = \prod_{i=1}^{n} (1 - p_i)$$

### Practical Layer Stack

For a hardened x86-64 Linux kernel with modern mitigations:

| Layer | Mitigation | Bypass Probability | Notes |
|:---|:---|:---|:---|
| 1 | User-space ASLR | $2^{-28} \approx 3.7 \times 10^{-9}$ | Without info leak |
| 2 | KASLR | $2^{-9} \approx 1.95 \times 10^{-3}$ | Weak alone |
| 3 | SMEP | $\approx 0$ (hardware) | Bypassed by kernel ROP |
| 4 | SMAP | $\approx 0$ (hardware) | Bypassed by kernel gadgets |
| 5 | Stack canary | $2^{-64}$ per guess | Random canary value |
| 6 | kCFI | $\sim 0.01\text{--}0.10$ | Depends on type set size |
| 7 | Seccomp | $1 - \frac{S_{\text{blocked}}}{S_{\text{total}}}$ | Profile-dependent |
| 8 | LSM (AppArmor) | Profile-dependent | Limits post-exploit capability |
| 9 | Lockdown | $\approx 0$ | Blocks kernel memory access |

### Correlation and Dependent Bypasses

The independent layer assumption breaks when a single primitive bypasses multiple layers. An arbitrary kernel read defeats layers 1, 2, 5, and 6 simultaneously:

$$P(\text{bypass} \mid \text{arb read}) \gg \prod P_i$$

The correlated bypass model groups mitigations by the primitive needed:

$$P(\text{exploit}) = \min_j P(\text{obtain primitive}_j) \times P(\text{bypass remaining} \mid \text{primitive}_j)$$

**Primitive classes and layers they defeat:**

- **Info leak (arbitrary read)**: Defeats ASLR (all forms), stack canaries, CFI type checks (with enough data)
- **Arbitrary write**: Defeats all software checks, requires hardware enforcement to stop
- **Code execution in kernel**: Defeated by SMEP; if in kernel text, bypasses everything except CFI

This is why hardware mitigations (SMEP, SMAP, CET shadow stacks, MTE) are categorically stronger than software-only defenses: they cannot be bypassed by arbitrary read/write alone.

### Quantifying Hardening Improvement

Define the **Mitigation Strength Score** as the negative log of bypass probability:

$$M = -\log_2 P(\text{bypass})$$

A fully hardened kernel versus a default (unhardened) kernel:

| Configuration | Approximate $M$ |
|:---|:---|
| Unhardened (no ASLR, no stack canary, no CFI) | $\sim 0$ bits |
| Default Ubuntu (ASLR + canary + KASLR + SMEP/SMAP) | $\sim 41$ bits |
| Hardened (+ kCFI + seccomp + lockdown + module signing) | $\sim 55\text{--}65$ bits |
| Maximum (+ CET shadow stack + MTE + gVisor) | $\sim 80+$ bits |

Each additional independent bit of mitigation doubles the attacker's cost. The marginal value of adding a new layer:

$$\Delta M = M_{n+1} - M_n = -\log_2 P(\text{bypass layer}_{n+1})$$

Hardware CFI (shadow stack + kCFI) adds approximately 20+ bits. Seccomp with a strict profile adds 3-5 bits. Kernel lockdown adds the equivalent of closing entire attack classes rather than probabilistic protection.

---

## References

- Shacham, H. "The Geometry of Innocent Flesh on the Bone: Return-into-libc without Function Calls" — CCS 2007
- Abadi, M. et al. "Control-Flow Integrity: Principles, Implementations, and Applications" — CCS 2005
- Intel, "Control-flow Enforcement Technology Specification" — Intel SDM, Vol. 1, Chapter 18
- Burow, N. et al. "Control-Flow Integrity: Precision, Security, and Performance" — ACM Computing Surveys 2017
- Linux kernel source: `arch/x86/mm/kaslr.c`, `security/lockdown/lockdown.c`, `kernel/cfi.c`
- Cook, K. "The Status of Kernel Self-Protection" — Linux Security Summit 2023
- PaX/grsecurity: "ASLR Design and Implementation" — grsecurity.net
- CIS Benchmarks: Ubuntu Linux — cisecurity.org
- Marco-Gisbert, H. et al. "On the Effectiveness of Full-ASLR on 64-bit Linux" — In-depth 2014
- Intel 64 and IA-32 Architectures Software Developer's Manual, Vol. 3A: CR4 (SMEP, SMAP, CET)
