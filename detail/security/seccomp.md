# The Mathematics of Seccomp — BPF Filter Automata and Syscall Lattices

> *Seccomp BPF filters are finite automata operating over a bounded instruction set, evaluating syscall decisions through deterministic state transitions on a lattice of security actions where the meet operation selects the most restrictive outcome.*

---

## 1. BPF Program as Finite Automaton (Automata Theory)

### Filter State Machine

A seccomp BPF program is a directed acyclic graph (DAG) with at most 4096 instructions. Each instruction is a state transition:

$$\delta: S \times \Sigma \rightarrow S$$

Where $S$ is the set of program counter states $\{0, 1, \ldots, n-1\}$ and $\Sigma$ is the instruction set:

| Instruction | Encoding | Operation |
|:---|:---:|:---|
| `BPF_LD` | 0x00 | Load word into accumulator |
| `BPF_LDX` | 0x01 | Load word into index register |
| `BPF_JMP` | 0x05 | Conditional/unconditional jump |
| `BPF_RET` | 0x06 | Return action value |
| `BPF_ALU` | 0x04 | Arithmetic/logic on accumulator |

### Instruction Format

Each BPF instruction is a 64-bit structure:

$$I = (\text{code}_{16}, \text{jt}_{8}, \text{jf}_{8}, \text{k}_{32})$$

For conditional jumps (`BPF_JMP`):
- If condition is true: $\text{PC} \leftarrow \text{PC} + 1 + \text{jt}$
- If condition is false: $\text{PC} \leftarrow \text{PC} + 1 + \text{jf}$

### Program Complexity

Maximum program length: $|P| \leq 4096$ instructions.

The number of possible distinct seccomp programs of length $n$:

$$|P_n| \leq (2^{16} \times 2^{8} \times 2^{8} \times 2^{32})^n = 2^{64n}$$

But DAG constraints (no backward jumps) and validity checks reduce this dramatically.

---

## 2. Action Lattice (Order Theory)

### Security Action Ordering

Seccomp return values form a totally ordered set under the "restrictiveness" relation:

$$\text{KILL\_PROCESS} \prec \text{KILL\_THREAD} \prec \text{TRAP} \prec \text{ERRNO} \prec \text{TRACE} \prec \text{LOG} \prec \text{ALLOW}$$

The numeric encoding reflects this (lower value = more restrictive):

| Action | Value | Restrictiveness |
|:---|:---:|:---:|
| KILL_PROCESS | 0x80000000 | Most restrictive |
| KILL_THREAD | 0x00000000 | |
| TRAP | 0x00030000 | |
| ERRNO(n) | 0x00050000+n | |
| TRACE(n) | 0x7FF00000+n | |
| LOG | 0x7FFC0000 | |
| ALLOW | 0x7FFF0000 | Least restrictive |

### Multi-Filter Meet Operation

When multiple filters are stacked, the kernel applies the meet (greatest lower bound) operation:

$$\text{result} = \bigwedge_{i=1}^{k} f_i(\text{syscall})$$

For $k$ chained filters, each returning action $a_i$:

$$\text{final\_action} = \min(a_1, a_2, \ldots, a_k)$$

This guarantees monotonic security: adding a filter can only maintain or increase restriction, never weaken.

---

## 3. Syscall Decision Function (Set Theory)

### Syscall Space

The syscall space $\mathcal{S}$ for x86_64 Linux contains approximately 450 syscalls:

$$\mathcal{S} = \{0, 1, 2, \ldots, N_{\max}\}$$

A seccomp profile partitions this into disjoint action sets:

$$\mathcal{S} = A_{\text{allow}} \cup A_{\text{deny}} \cup A_{\text{log}} \cup A_{\text{trap}}$$

$$A_i \cap A_j = \emptyset \quad \forall i \neq j$$

### Default-Deny vs Default-Allow

Default-deny (allowlist) profile:

$$f(\text{sys}) = \begin{cases} \text{ALLOW} & \text{if } \text{sys} \in W \\ \text{ERRNO}(1) & \text{otherwise} \end{cases}$$

Where $W \subset \mathcal{S}$ is the allowlist. Security coverage:

$$\text{coverage} = 1 - \frac{|W|}{|\mathcal{S}|} = 1 - \frac{|W|}{N_{\max}}$$

Docker's default profile: $|W| \approx 406$, $|\mathcal{S}| \approx 450$, coverage $\approx 9.8\%$ blocked.

### Argument Filtering

Syscall arguments extend the decision space to $\mathcal{S} \times \mathbb{Z}^6$ (6 argument registers):

$$f(\text{sys}, a_0, a_1, \ldots, a_5) = \begin{cases} \text{ALLOW} & \text{if } (\text{sys}, \vec{a}) \in R \\ \text{action} & \text{otherwise} \end{cases}$$

Argument comparison operators:

| Operator | Meaning | Expression |
|:---|:---|:---|
| EQ | Equal | $a_i = k$ |
| NE | Not equal | $a_i \neq k$ |
| LT | Less than | $a_i < k$ |
| LE | Less or equal | $a_i \leq k$ |
| GT | Greater than | $a_i > k$ |
| GE | Greater or equal | $a_i \geq k$ |
| MASKED_EQ | Masked equal | $a_i \wedge m = k$ |

---

## 4. Filter Evaluation Complexity (Complexity Theory)

### Single Filter Evaluation

BPF program execution is bounded:

$$T_{\text{eval}} = O(|P|)$$

Since programs are DAGs with no loops and $|P| \leq 4096$:

$$T_{\text{worst}} = O(4096) = O(1)$$

This constant-time bound is critical for kernel security: no filter can cause denial of service.

### Multi-Filter Stack

With $k$ stacked filters:

$$T_{\text{total}} = O\left(\sum_{i=1}^{k} |P_i|\right) \leq O(4096k)$$

The kernel evaluates every filter for every syscall (no short-circuit):

$$\text{overhead per syscall} = \sum_{i=1}^{k} c \cdot |P_i|$$

Where $c \approx 4\text{ns}$ per BPF instruction on modern hardware.

### Verification Time

The kernel verifier checks each filter at load time:

$$T_{\text{verify}} = O(|P|^2)$$

Verification ensures:
- All jumps target valid instructions (DAG property)
- No division by zero
- Program terminates (guaranteed by DAG + length limit)
- Only valid seccomp return values used

---

## 5. Information-Theoretic Attack Surface (Information Theory)

### Syscall Entropy

The entropy of the syscall space available to a process:

$$H(\mathcal{S}_{\text{allowed}}) = \log_2 |W|$$

For Docker's default profile ($|W| \approx 406$):

$$H = \log_2 406 \approx 8.66 \text{ bits}$$

Versus unrestricted ($|\mathcal{S}| \approx 450$):

$$H = \log_2 450 \approx 8.81 \text{ bits}$$

Attack surface reduction:

$$\Delta H = \log_2 450 - \log_2 406 = \log_2 \frac{450}{406} \approx 0.15 \text{ bits}$$

A minimal profile (e.g., 50 syscalls):

$$\Delta H = \log_2 450 - \log_2 50 = \log_2 9 \approx 3.17 \text{ bits}$$

This represents a $9\times$ reduction in the combinatorial attack space.

### Covert Channel Capacity

A seccomp filter with `SCMP_ACT_ERRNO(n)` can leak information through errno values:

$$C_{\text{errno}} = \log_2 |\{n : 1 \leq n \leq 4095\}| \approx 12 \text{ bits per syscall}$$

Using `SCMP_ACT_KILL` eliminates this channel entirely: $C = 0$.

---

## 6. Profile Optimization (Optimization Theory)

### Minimum Filter Problem

Given a set of required syscalls $R$ and the full syscall set $\mathcal{S}$:

$$\text{minimize } |P| \text{ subject to:}$$
$$f(s) = \text{ALLOW} \quad \forall s \in R$$
$$f(s) = \text{DENY} \quad \forall s \in \mathcal{S} \setminus R$$

### Binary Decision Diagram (BDD) Representation

Optimal BPF programs can be derived from BDDs on syscall numbers:

For $b = \lceil \log_2 N_{\max} \rceil$ bits representing syscall numbers:

$$|P_{\text{optimal}}| \leq 2b + |R| = O(\log N_{\max} + |R|)$$

Linear scan (naive approach):

$$|P_{\text{naive}}| = 2|R| + 1$$

BDD approach (binary search on syscall number):

$$|P_{\text{BDD}}| = O(|R| \cdot \log N_{\max})$$

For typical profiles ($|R| = 50$, $N_{\max} = 450$):

| Strategy | Instructions | Avg Jumps |
|:---|:---:|:---:|
| Linear scan | 101 | 50 |
| Binary search | ~45 | ~9 |
| BDD-optimized | ~35 | ~7 |

---

## 7. Container Isolation Guarantees (Security Theory)

### Defense-in-Depth Composition

Container security layers compose as independent filters:

$$P(\text{escape}) = P(\text{bypass seccomp}) \times P(\text{bypass namespaces}) \times P(\text{bypass caps}) \times P(\text{bypass MAC})$$

If each layer has independent bypass probability $p_i$:

$$P(\text{escape}) = \prod_{i=1}^{n} p_i$$

For four layers each with $p = 0.01$:

$$P(\text{escape}) = 0.01^4 = 10^{-8}$$

### Syscall Attack Surface Metric

$$\text{ASM} = \sum_{s \in W} \text{impact}(s) \times \text{complexity}(s)^{-1}$$

Where $\text{impact}(s)$ is the CVSS-like score of exploiting syscall $s$ and $\text{complexity}(s)$ measures exploitation difficulty.

High-impact syscalls:

| Syscall | Impact | Historical CVEs |
|:---|:---:|:---:|
| `ptrace` | 9.8 | 12+ |
| `mount` | 9.0 | 8+ |
| `clone` (NEWUSER) | 8.5 | 15+ |
| `keyctl` | 7.5 | 5+ |
| `bpf` | 8.0 | 10+ |

---

## Prerequisites

automata-theory, lattice-theory, set-theory, information-theory, complexity-theory

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| Filter evaluation (single) | $O(1)$ bounded by 4096 | $O(1)$ — 2 registers |
| Filter evaluation (k stack) | $O(k)$ | $O(1)$ |
| Filter load + verify | $O(n^2)$ | $O(n)$ — program copy |
| Profile optimization (BDD) | $O(R \cdot \log N)$ | $O(R)$ |
| Multi-filter meet | $O(k)$ per syscall | $O(k)$ — result per filter |
