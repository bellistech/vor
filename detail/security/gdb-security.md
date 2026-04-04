# The Mathematics of GDB Security Research -- Memory Models and Crash Analysis Theory

> *A debugger is a theorem prover for concrete program states: each breakpoint is an assertion, each watchpoint a loop invariant, and each crash a constructive proof of a violated safety property.*

---

## 1. Heap Allocator Data Structures (Graph Theory of Freelists)

### The Problem

Understanding heap exploitation requires a formal model of how glibc malloc organizes memory. Free chunks are stored in linked lists (bins) organized by size class. Corruption of these data structures -- forward pointers, backward pointers, size fields -- enables arbitrary write primitives that form the foundation of modern heap exploits.

### The Formula

glibc malloc maintains a hierarchy of free lists. For a chunk of size $s$ (including header), the bin assignment:

$$\text{bin}(s) = \begin{cases} \text{tcache}[i] & \text{if } s = 16(i+1) + 16, \; i \in [0, 63], \; \text{count} \leq 7 \\ \text{fastbin}[j] & \text{if } s = 16(j+2), \; j \in [0, 9] \\ \text{unsorted} & \text{if recently freed and not fast/tcache} \\ \text{smallbin}[k] & \text{if } s < 1024, \; k = s/16 \\ \text{largebin}[l] & \text{if } s \geq 1024, \; l = \lfloor\log_2(s)\rfloor - 6 \end{cases}$$

Chunk structure (allocated):

$$[\underbrace{\text{prev\_size}}_{\text{8 bytes}} | \underbrace{\text{size} | \text{AMP}}_{\text{8 bytes}} | \underbrace{\text{user data}}_{\text{requested size}}]$$

Chunk structure (freed, in tcache):

$$[\text{prev\_size} | \text{size} | \underbrace{\text{fd (next free)}}_{\text{8 bytes}} | \underbrace{\text{key (tcache)}}_{\text{8 bytes}} | \text{unused...}]$$

The tcache freelist is singly-linked (LIFO):

$$\text{tcache}[i] \to c_1 \to c_2 \to \ldots \to c_k \to \text{NULL}, \; k \leq 7$$

The unlink operation for doubly-linked bins (smallbin/largebin) performs:

$$\text{fd}(\text{victim})\text{.bk} = \text{bk}(\text{victim})$$
$$\text{bk}(\text{victim})\text{.fd} = \text{fd}(\text{victim})$$

This is exploitable if the attacker controls fd and bk:

$$\text{write}(\text{fd} + \text{bk\_offset}) = \text{bk}$$
$$\text{write}(\text{bk} + \text{fd\_offset}) = \text{fd}$$

Giving a constrained write-what-where primitive.

### Worked Example

Tcache poisoning attack. Initial state: tcache bin for size 0x20 contains chunk A.

$$\text{tcache}[0] \to A \to \text{NULL}$$

Attacker overwrites A's fd pointer to target address T:

$$\text{tcache}[0] \to A \to T \to \text{???}$$

First malloc(0x18) returns A. Second malloc(0x18) returns T. Attacker now has a pointer to any address and can write arbitrary data there.

With safe-linking (glibc 2.32+): $\text{fd}_{\text{stored}} = \text{fd}_{\text{real}} \oplus (\text{chunk\_addr} \gg 12)$. Bypass requires knowing the heap base address.

## 2. Crash Signal Classification (Fault Theory)

### The Problem

When a program crashes, the operating system delivers a signal indicating the fault type. The signal number, faulting address, and faulting instruction together determine the vulnerability class and exploitability. GDB's crash analysis begins with classifying these signals.

### The Formula

The fault classification decision tree:

$$\text{vuln\_class}(\text{signal}, \text{addr}, \text{insn}) = \begin{cases} \text{null deref} & \text{if addr} < \text{page\_size} \\ \text{stack overflow} & \text{if addr} \in [\text{stack\_guard}] \\ \text{UAF/OOB} & \text{if addr} \in [\text{heap region}] \\ \text{wild pointer} & \text{if addr} \notin \text{mapped regions} \\ \text{type confusion} & \text{if insn is call/jmp} \end{cases}$$

Exploitability scoring (simplified MSEC model):

| Criterion | Score | Description |
|:---|:---:|:---|
| Attacker controls RIP | +4 | Code execution likely |
| Attacker controls write addr | +3 | Arbitrary write |
| Attacker controls read addr | +2 | Info leak |
| Crash in heap metadata | +3 | Heap corruption |
| Crash in memcpy/strcpy | +2 | Buffer overflow |
| Null dereference | +0 | Usually not exploitable |
| Stack canary failure | +1 | Overflow present but guarded |

Total score:

$$\text{exploitability} = \begin{cases} \text{EXPLOITABLE} & \text{if score} \geq 4 \\ \text{PROBABLY\_EXPLOITABLE} & \text{if score} \in [2, 3] \\ \text{PROBABLY\_NOT} & \text{if score} \in [1] \\ \text{NOT\_EXPLOITABLE} & \text{if score} = 0 \end{cases}$$

### Worked Example

Crash: SIGSEGV at instruction `mov [rdi], rax` where RDI = 0x4141414141414141.

Analysis:
- Signal: SIGSEGV (memory access violation)
- Faulting instruction: write (`mov [rdi], rax`)
- Address 0x4141414141414141: clearly attacker-controlled (pattern)
- RDI controlled: attacker controls write destination
- RAX may also be controlled: attacker controls write value

Score: +3 (controlled write addr) + potential +4 if RAX leads to RIP control = EXPLOITABLE.

In GDB: `x/i $rip` shows the faulting instruction, `info registers` confirms controlled registers.

## 3. Watchpoint Implementation and Detection Theory (Hardware Debug Registers)

### The Problem

Hardware watchpoints use CPU debug registers (DR0-DR3 on x86) to trigger breakpoints on memory access without modifying code. Understanding their capabilities and limitations is essential for tracking memory corruption and for understanding anti-debug techniques that manipulate debug registers.

### The Formula

x86-64 provides 4 debug address registers (DR0-DR3) and a control register (DR7). Each watchpoint can monitor:

$$\text{watch\_size} \in \{1, 2, 4, 8\} \text{ bytes}$$

$$\text{watch\_type} \in \{\text{execute}, \text{write}, \text{read/write}\}$$

DR7 encodes the configuration for all 4 watchpoints:

$$\text{DR7} = \bigcup_{i=0}^{3} [\text{enable}_i | \text{condition}_i | \text{length}_i]$$

The total watchable memory with all 4 registers:

$$\text{watched\_bytes} = \sum_{i=0}^{3} \text{size}_i \leq 4 \times 8 = 32 \text{ bytes}$$

For monitoring a larger region (e.g., a 64-byte heap chunk header), software watchpoints are needed:

$$T_{\text{software\_watch}} = T_{\text{single\_step}} \times |\text{instructions}|$$

This results in approximately 1000x slowdown since every instruction must be single-stepped and the watched address checked.

The probability of detecting corruption in a $W$-byte watched region within an $N$-byte target:

$$P(\text{detect}) = \frac{\min(W, N)}{N}$$

With 4 hardware watchpoints of 8 bytes each on a 64-byte heap chunk: $P = 32/64 = 50\%$ coverage.

### Worked Example

Debugging a heap corruption where a 4096-byte buffer overflows into the next chunk's metadata (16 bytes: prev_size + size).

Hardware watchpoints: set 2 watchpoints on the metadata (16 bytes total). Coverage: 100% of the metadata.

Remaining 2 watchpoints available for: fd/bk pointers, canary, or other critical fields.

The corruption is detected at the exact instruction that writes past the buffer boundary, giving:
- The faulting instruction address (which function overflows)
- The call stack at the time of corruption
- The exact value being written

## 4. Stack Frame Reconstruction and Call Chain Analysis (Automata Theory)

### The Problem

Crash analysis requires reconstructing the call chain from a corrupted stack. When stack frames are smashed by buffer overflows, the frame pointer chain is broken and GDB's backtrace fails. Manual reconstruction requires understanding the frame layout and scanning for valid return addresses.

### The Formula

A valid stack frame on x86-64 (with frame pointers):

$$\text{frame} = [\text{locals}][\text{saved RBP}][\text{return addr}]$$

The frame pointer chain:

$$\text{RBP} \to \text{saved\_RBP}_1 \to \text{saved\_RBP}_2 \to \ldots \to 0$$

Each saved RBP value $f_i$ must satisfy:

$$f_i \in [\text{stack\_bottom}, \text{stack\_top}] \land f_i > f_{i-1} \quad \text{(grows downward)}$$

Each return address $r_i$ must satisfy:

$$r_i \in \text{executable\_regions} \land (r_i - 5) \text{ is a CALL instruction}$$

The heuristic for finding return addresses in a corrupted stack:

$$\text{candidates} = \{v \in \text{stack} \mid v \in [\text{text\_start}, \text{text\_end}] \land \text{is\_call\_site}(v)\}$$

The false positive rate for random stack scanning:

$$P(\text{false positive}) = \frac{|\text{text}|}{2^{64}} \approx 10^{-13} \text{ per 8-byte slot}$$

For a 4096-byte stack region (512 slots):

$$E[\text{false positives}] = 512 \times 10^{-13} \approx 5 \times 10^{-11}$$

Essentially zero false positives, making heuristic stack scanning very reliable.

### Worked Example

Stack overflow corrupted 200 bytes, destroying 25 stack slots including saved RBP and return address.

GDB `bt` shows only the current frame. Manual recovery:

1. Scan stack for values in .text range: `find /g $rsp, $rsp+4096, 0x400000, 0x500000` (conceptually)
2. Found candidates at $rsp+208, $rsp+296, $rsp+344
3. Verify each: `x/i candidate-5` should show a `call` instruction
4. Reconstruct: `frame 0 -> rsp+208 -> rsp+296 -> rsp+344`

This recovers the call chain despite the corruption.

## 5. Breakpoint Density and Detection Theory (Steganography)

### The Problem

Anti-debugging techniques scan executable code for software breakpoint opcodes (0xCC / INT3). The debugger must either use hardware breakpoints (limited to 4) or hide software breakpoints from integrity checks. Understanding the detection probability helps choose the right debugging strategy.

### The Formula

A software breakpoint replaces the first byte of an instruction with 0xCC. If the program checksums its own code, the detection probability:

$$P(\text{detect}) = \begin{cases} 1 & \text{if checksum covers breakpoint location} \\ 0 & \text{otherwise} \end{cases}$$

For a binary that checksums its entire .text section ($S$ bytes) with $n$ breakpoints:

$$\text{checksum}(\text{original}) \neq \text{checksum}(\text{modified})$$

The probability of the checksum detecting a single-byte change (for CRC-32):

$$P(\text{CRC detects}) = 1 - 2^{-32} \approx 1.0$$

For a naive sum-based checksum, a single 0xCC insertion changes the sum by:

$$\Delta = 0\text{xCC} - \text{original\_byte}$$

Counter-strategy: use hardware breakpoints (invisible to checksums):

$$|\text{hw\_breakpoints}| = 4 \quad \text{(x86-64 limit)}$$

If the analyst needs more than 4 breakpoints, alternatives include:

1. Patch the checksum routine to always pass
2. Use single-stepping past the checksum region
3. Use dynamic binary instrumentation (DBI) via Frida or PIN

The work factor for each approach:

| Technique | Breakpoint Limit | Detectability | Overhead |
|:---|:---:|:---:|:---:|
| Software BP | Unlimited | High (0xCC visible) | None |
| Hardware BP | 4 | None | None |
| Single-step | Unlimited | Low | 1000x slowdown |
| DBI (Frida) | Unlimited | Medium | 5-10x slowdown |

## 6. Memory Corruption Root-Cause Analysis (Causal Inference)

### The Problem

A crash is an effect, not a cause. The root-cause corruption may have occurred thousands of instructions before the crash, and the corrupted data may have propagated through multiple data structures. GDB's challenge is bridging the temporal gap between corruption and crash.

### The Formula

The corruption propagation model: a corruption event $e$ at time $t_0$ affects memory location $m_0$. Subsequent reads from $m_0$ propagate the corruption:

$$\text{taint}(t) = \{m \mid \exists \text{path } m_0 \to m \text{ via read/write chain by time } t\}$$

The taint set grows as:

$$|\text{taint}(t)| \leq |\text{taint}(t_0)| \cdot d^{(t - t_0)}$$

where $d$ is the average data dependency fan-out per instruction. In practice, $d \approx 1.2$ for sequential code.

The temporal distance between corruption and crash:

$$\Delta t = t_{\text{crash}} - t_{\text{corrupt}}$$

| Corruption Type | Typical $\Delta t$ | Taint Spread | Difficulty |
|:---|:---:|:---:|:---:|
| Stack overflow -> RIP | 1-10 insns | Minimal | Easy |
| Heap UAF -> vtable call | 100-10K insns | Moderate | Medium |
| Heap metadata corruption | 10K-1M insns | Extensive | Hard |
| Type confusion -> later use | 1M+ insns | Extensive | Very Hard |

The diagnostic strategy depends on $\Delta t$:

$$\text{strategy} = \begin{cases} \text{backtrace + registers} & \Delta t < 100 \\ \text{watchpoints} & \Delta t < 10{,}000 \\ \text{record/replay + reverse} & \Delta t < 1{,}000{,}000 \\ \text{ASAN + reproduction} & \Delta t > 1{,}000{,}000 \end{cases}$$

### Worked Example

Crash in `free()` with corrupted heap chunk size. The corruption happened during a previous loop iteration that wrote past a buffer boundary.

Temporal distance estimate: the overflowing loop ran for approximately 5,000 iterations at 3 instructions each = 15,000 instructions between corruption and crash.

Strategy: set a hardware watchpoint on the chunk size field and re-run. The watchpoint fires at the exact loop iteration that overflows, revealing:
- The overflowing buffer's address
- The loop index at the time of corruption
- The off-by-one error in the loop bound

Without the watchpoint, the analyst would only see the crash in `free()` with no indication of which buffer or which loop caused it.

---

*GDB bridges the gap between theoretical vulnerability and practical exploit by making the abstract concrete: every register value is observable, every memory byte is inspectable, and every execution path is reproducible. The debugger transforms the exponential search space of "what went wrong" into a directed, systematic investigation guided by hardware watchpoints, conditional breakpoints, and reverse execution.*

## Prerequisites

- Computer architecture (x86-64 registers, calling conventions, debug registers, MMU)
- Data structures (linked lists, hash tables, tree structures in heap allocators)
- Operating systems (signals, virtual memory, process memory layout, ptrace)
- Graph theory (call graphs, data flow graphs, taint propagation)
- Probability theory (crash classification confidence, brute-force analysis)
- Automata theory (stack machine model, frame reconstruction)

## Complexity

- **Beginner:** Setting breakpoints, examining registers and memory, basic backtrace analysis, using pwndbg context display
- **Intermediate:** Hardware watchpoints for corruption tracking, heap bin inspection, Python-scripted analysis, anti-debug bypass techniques
- **Advanced:** Record/replay debugging for temporal root-cause analysis, custom GDB Python commands for automated exploit development, kernel debugging with KGDB, multi-threaded race condition analysis
