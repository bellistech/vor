# The Mathematics of eBPF Security — Verifier Theory and Exploit Complexity

> *The eBPF verifier is an abstract interpreter that performs static analysis over a directed acyclic graph of BPF instructions. Its security guarantees rely on sound modeling of register states, memory bounds, and control flow — any divergence between the verifier's abstract model and the JIT compiler's concrete execution creates an exploitable gap.*

---

## 1. Verifier State Space (Abstract Interpretation)

### Register State Tracking

The verifier models each of the 11 BPF registers as a bounded value:

$$R_i = (\text{type}, \text{min\_value}, \text{max\_value}, \text{var\_off})$$

where $\text{var\_off}$ is a tnum (tristate number) encoding known and unknown bits:

$$\text{tnum} = (\text{value}, \text{mask}) \quad \text{where } \text{known bits} = \lnot\text{mask}$$

### Verification Path Complexity

For a program with $B$ branches, worst-case verification paths:

$$\text{Paths} = O(2^B)$$

The verifier uses path pruning to avoid exponential blowup. Two states $S_1, S_2$ at the same instruction are equivalent if:

$$\forall i \in [0,10]: S_1.R_i \supseteq S_2.R_i$$

A pruning error occurs when $S_{\text{pruned}} \not\supseteq S_{\text{actual}}$, skipping validation of unsafe operations.

| Kernel Version | Complexity Limit | Bounded Loops |
|:---|:---:|:---:|
| 4.x | 65,536 insns | No |
| 5.2 | 1,000,000 insns | No |
| 5.3+ | 1,000,000 insns | Yes |

---

## 2. Complexity Bomb Analysis (Verification DoS)

### Branch Explosion

A program with $n$ sequential `if` statements and independent conditions:

$$\text{Verification cost} = O(2^n) \text{ without pruning}$$

The most efficient complexity bomb uses $3n + c$ instructions for $n$ branches — each branch adds 3 instructions (load, compare, jump) but doubles verification work.

| Program Size | Branches | Paths (no pruning) | Paths (with pruning) |
|:---|:---:|:---:|:---:|
| 30 insns | 10 | 1,024 | ~50 |
| 90 insns | 30 | $1.07 \times 10^9$ | ~200 |
| 150 insns | 50 | $1.13 \times 10^{15}$ | ~500 |

Pruning effectiveness depends on register state diversity. Crafted programs that maximize state uniqueness defeat pruning:

$$\text{Pruning failure rate} = 1 - \frac{|\text{pruneable states}|}{|\text{total states}|}$$

---

## 3. Out-of-Bounds Access Calculus (Memory Safety)

### Pointer Arithmetic Bounds

The verifier tracks pointer bounds: $\text{ptr} + \text{offset} \in [\text{base}, \text{base} + \text{size})$

### Signed vs Unsigned Confusion

After conditional `if R1 < 100`:
- Unsigned interpretation: $R_1 \in [0, 99]$
- Signed interpretation: $R_1 \in [-2^{63}, 99]$

If the verifier applies unsigned bounds but the JIT uses signed comparison:

$$\text{Exploitation window} = \text{offset}_{\text{actual,max}} - \text{offset}_{\text{verifier,max}}$$

For CVE-2021-3490, $W$ can reach $2^{32}$ bytes of OOB access.

### ALU32 Bound Propagation

32-bit ALU operations on 64-bit registers create truncation opportunities:

$$\text{alu32}(R_i) = R_i \bmod 2^{32}$$

If the verifier fails to propagate the 32-bit truncation to the 64-bit bound tracking, the upper 32 bits remain unconstrained.

---

## 4. JIT Compiler Security (Code Generation)

### Constant Blinding

Without hardening, immediate value $C$ appears directly in JIT output:

$$\text{mov rax}, C \quad \Rightarrow \quad \text{48 b8 } C_{\text{le64}}$$

With constant blinding:

$$C \rightarrow (C \oplus R), R \quad \text{where } R \text{ is random}$$

### JIT Spray Success Probability

| JIT Hardening | Immediates Controlled | Gadget Predictability |
|:---|:---:|:---:|
| Off | Yes | 100% (deterministic) |
| Unprivileged only | Partial | ~50% |
| Full | No | $\approx n/2^{32}$ |

With controlled immediates (no blinding), an attacker deterministically places gadgets. With full hardening, finding a specific 4-byte gadget in $n$ immediates:

$$P(\text{gadget}) = 1 - (1 - 2^{-32})^n \approx n/2^{32}$$

---

## 5. Map Race Condition Timing (Concurrency)

### TOCTOU Window

For map-of-maps inner map replacement, the race window:

$$T_{\text{race}} = T_{\text{lookup}} - T_{\text{replace}}$$

### Probability of Successful Race

If the BPF program's map access takes $t_a$ ns and replacement takes $t_r$ ns:

$$P(\text{race hit}) = \frac{\min(t_a, t_r)}{T_{\text{period}}}$$

Over $N$ attempts: $P(\text{success in } N) = 1 - (1 - P_{\text{single}})^N$

| Race Window | Single Attempt | 1000 Attempts | 1M Attempts |
|:---|:---:|:---:|:---:|
| 10 ns / 1 ms | $10^{-5}$ | 0.995% | 99.995% |
| 100 ns / 1 ms | $10^{-4}$ | 9.5% | 100% |
| 1 us / 1 ms | $10^{-3}$ | 63.2% | 100% |

---

## 6. Capability Lattice (Privilege Model)

### BPF Capability Hierarchy (Kernel 5.8+)

$$\text{CAP\_BPF} \subset \text{CAP\_BPF} \cup \text{CAP\_PERFMON} \subset \text{CAP\_SYS\_ADMIN}$$

| Operation | CAP_BPF | +CAP_PERFMON | +CAP_NET_ADMIN | CAP_SYS_ADMIN |
|:---|:---:|:---:|:---:|:---:|
| Load socket filter | Yes | Yes | Yes | Yes |
| Create maps | Yes | Yes | Yes | Yes |
| Attach kprobe | No | Yes | No | Yes |
| Attach XDP | No | No | Yes | Yes |
| bpf_probe_read_kernel | No | Yes | No | Yes |
| bpf_override_return | No | Yes | No | Yes |

Attack surface by capability set:

$$A(\text{caps}) = |\{h : h \in \text{Helpers}, \text{req}(h) \subseteq \text{caps}\}| \times |\{t : t \in \text{ProgTypes}, \text{req}(t) \subseteq \text{caps}\}|$$

---

## 7. Historical CVE Complexity Analysis

| CVE | Kernel | Root Cause | OOB Window |
|:---|:---:|:---|:---:|
| CVE-2020-8835 | 5.5 | 32-bit bound truncation | $2^{32}$ bytes |
| CVE-2021-3490 | 5.11 | ALU32 bound tracking | $2^{32}$ bytes |
| CVE-2021-31440 | 5.11 | Bounds after bitwise ops | Variable |
| CVE-2021-4204 | 5.8-5.16 | Ringbuf type confusion | $2^{32}$ bytes |
| CVE-2022-23222 | 5.8-5.14 | PTR_TO_MEM bounds | 4096 bytes |

### Exploit Reliability

$$\text{Reliability} = P(\text{verifier bypass}) \times P(\text{useful OOB}) \times P(\text{KASLR bypass})$$

Typical: $0.99 \times 0.8 \times 0.7 \approx 0.55$ for a well-crafted exploit.

---

## 8. Tail Call Chain Analysis (DoS Complexity)

With tail call depth $D = 33$ and per-program budget $T_p$ at 1M instructions (~10ms):

$$T_{\text{max}} = D \times T_p = 33 \times 10\text{ms} = 330\text{ms}$$

A tail call loop with period $k$ programs: $\lfloor D/k \rfloor$ iterations. Total CPU time is constant ($33 \times T_p$) regardless of loop period, but tight loops ($k=2$) maximize stack frame churn.

---

*The fundamental tension in eBPF security is between expressiveness and verifiability: every new feature expands the abstract state space the verifier must soundly model, and history shows that novel state transitions consistently produce exploitable verification gaps within 1-2 kernel release cycles.*

## Prerequisites

- Understanding of abstract interpretation and lattice theory
- Knowledge of x86-64 assembly and JIT compilation
- Familiarity with Linux kernel memory layout and capability system

## Complexity

- **Beginner:** Understanding BPF program types, maps, and capability requirements
- **Intermediate:** Analyzing verifier behavior, crafting complexity bombs, exploiting known CVEs
- **Advanced:** Discovering novel verifier bugs through differential analysis of abstract vs concrete semantics
