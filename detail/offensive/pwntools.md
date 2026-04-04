# The Mathematics of Pwntools -- Exploit Primitives and Memory Corruption Theory

> *Every exploit is a constructive proof that a program's actual state space exceeds its intended state space, and pwntools provides the algebra for navigating between the two.*

---

## 1. Buffer Overflow Geometry (Linear Algebra of the Stack)

### The Problem

A stack buffer overflow allows writing beyond an allocated buffer to overwrite adjacent stack data, including saved frame pointers and return addresses. The precise offset from the buffer start to the return address depends on the stack frame layout, which varies with compiler, optimization level, and alignment requirements.

### The Formula

For a function with local buffer $B$ of declared size $n$, the stack frame on x86-64:

$$\text{layout} = [\underbrace{B[0] \ldots B[n-1]}_{\text{buffer } n \text{ bytes}} \mid \underbrace{\text{padding}}_{\text{alignment}} \mid \underbrace{\text{canary}}_{\text{8 bytes}} \mid \underbrace{\text{saved RBP}}_{\text{8 bytes}} \mid \underbrace{\text{saved RIP}}_{\text{8 bytes}}]$$

The offset to the return address:

$$\delta = n + \text{pad}(n) + 8_{\text{canary}} + 8_{\text{RBP}}$$

where alignment padding is:

$$\text{pad}(n) = (16 - (n \bmod 16)) \bmod 16$$

Without a canary: $\delta = n + \text{pad}(n) + 8_{\text{RBP}}$

The cyclic pattern method determines $\delta$ empirically. A De Bruijn sequence $D(k, n)$ over alphabet $k$ with subsequence length $n$ has the property that every $n$-length subsequence appears exactly once. Length:

$$|D(k,n)| = k^n + n - 1$$

For pwntools' `cyclic()` with $k = 26$ (lowercase letters) and $n = 4$ (32-bit):

$$|D(26, 4)| = 26^4 + 3 = 456{,}979 \text{ bytes max}$$

Any 4-byte value in the crash register uniquely identifies the offset within the pattern.

### Worked Example

Binary with `char buf[64]`, no canary, compiled with `-fno-stack-protector`:

- Buffer size: 64 bytes
- Padding: $\text{pad}(64) = 0$ (already 16-aligned)
- No canary: skip 8 bytes
- Saved RBP: 8 bytes
- Offset: $\delta = 64 + 0 + 8 = 72$ bytes

Payload: `b'A' * 72 + p64(target_address)`

## 2. Return-Oriented Programming (Computational Completeness)

### The Problem

When the stack is non-executable (NX/DEP), attackers chain existing code fragments ("gadgets") ending in RET instructions to perform arbitrary computation. ROP is Turing-complete given sufficient gadgets, and pwntools automates the search and chaining process.

### The Formula

A gadget is a sequence of instructions ending in a control-flow transfer (typically RET). The gadget set $\mathcal{G}$ of a binary is:

$$\mathcal{G} = \{g_i = (I_{i,1}; I_{i,2}; \ldots; I_{i,k_i}; \text{ret}) \mid g_i \in \text{.text}\}$$

A ROP chain is an ordered sequence of gadget addresses on the stack:

$$\text{chain} = [a_{g_1}, d_1, a_{g_2}, d_2, \ldots, a_{g_m}, d_m]$$

where $a_{g_i}$ is the address of gadget $i$ and $d_i$ are data values consumed by `pop` instructions.

The computational power depends on the available gadget classes:

| Gadget Class | Example | Provides |
|:---|:---|:---:|
| Load register | `pop rdi; ret` | Data loading |
| Store memory | `mov [rdi], rax; ret` | Memory writes |
| Arithmetic | `add rax, rbx; ret` | Computation |
| Syscall | `syscall; ret` | Kernel interface |
| Conditional | `cmp; cmov; ret` | Branching |

Turing completeness requires: load, store, arithmetic, and conditional execution. The minimum gadget set for a Linux shell:

$$|\mathcal{G}_{\min}| = \begin{cases} 3 & \text{(pop rdi; ret, pop rsi; ret, syscall)} \\ & \text{for execve("/bin/sh", NULL, NULL)} \end{cases}$$

The number of available gadgets in a typical binary scales with code size:

$$|\mathcal{G}| \approx 0.02 \cdot |\text{.text}|$$

A 1 MB text section yields approximately 20,000 gadgets, virtually guaranteeing Turing completeness.

## 3. Format String Exploitation (Positional Arithmetic)

### The Problem

Format string vulnerabilities allow reading from and writing to arbitrary memory addresses. The `%n` specifier writes the count of bytes printed so far to a pointer argument. By controlling the format string and the values on the stack, an attacker can write arbitrary values to arbitrary addresses.

### The Formula

The `%n` family writes the current output byte count to the address at the corresponding argument position:

$$\text{written}(n) = \sum_{i=1}^{n} \text{field\_width}_i + \text{literal\_chars}$$

For writing a value $V$ to address $A$ using `%hhn` (write one byte):

$$V_{\text{byte}} = V \bmod 256$$

$$\text{padding} = (V_{\text{byte}} - \text{already\_written}) \bmod 256$$

For a 4-byte write using four `%hhn` writes at addresses $A, A+1, A+2, A+3$:

$$\text{total writes} = 4 \quad \text{(one per byte)}$$

The format string payload size for writing value $V$ to address $A$ at stack offset $k$:

$$|\text{payload}| \leq 4 \times 8_{\text{addr}} + 4 \times (\lceil\log_{10}(256)\rceil + 10)_{\text{format spec}}$$

Pwntools' `fmtstr_payload()` optimizes the write order to minimize total padding:

$$\text{optimal order} = \text{sort bytes by } (V_i - V_{i-1}) \bmod 256$$

### Worked Example

Writing `0x0804beef` to address `0x08049010` at stack offset 7:

Bytes to write: `0xef` at `0x08049010`, `0xbe` at `0x08049011`, `0x04` at `0x08049012`, `0x08` at `0x08049013`.

Sorted by value: `0x04, 0x08, 0xbe, 0xef`.

Padding sequence: $4, (8-4)=4, (190-8)=182, (239-190)=49$ bytes.

Total output: $4 + 4 + 182 + 49 = 239$ bytes printed.

## 4. ASLR Entropy and Information Leaks (Information Theory)

### The Problem

Address Space Layout Randomization randomizes the base addresses of the stack, heap, libraries, and (with PIE) the executable itself. Exploits must leak addresses to defeat ASLR. The entropy of each region determines the number of bits an attacker must learn or guess.

### The Formula

On x86-64 Linux, the randomization entropy for each region:

| Region | Random Bits | Possible Positions | Brute-Force (1000 attempts/s) |
|:---|:---:|:---:|:---:|
| Stack | 30 bits | $2^{30} \approx 10^9$ | 12 days |
| mmap/libc | 28 bits | $2^{28} \approx 2.7 \times 10^8$ | 3 days |
| Heap | 13 bits | $2^{13} = 8{,}192$ | 8 seconds |
| PIE executable | 28 bits | $2^{28}$ | 3 days |

A single leaked pointer from any region reduces the entropy of that region to zero:

$$H_{\text{after leak}} = 0 \text{ bits}$$

The remaining entropy after a partial leak (e.g., low 12 bits known from page alignment):

$$H_{\text{remaining}} = H_{\text{total}} - H_{\text{known}}$$

For a 48-bit virtual address with 12-bit page offset fixed:

$$H_{\text{libc}} = 28 \text{ bits (only bits 12-39 randomized)}$$

Multiple independent leaks from correlated regions do not add information if the offset is fixed:

$$H(\text{libc base} \mid \text{any libc pointer}) = 0$$

### Worked Example

An info leak reveals `puts@libc = 0x7f3a4c069420`.

Known: libc offset of `puts` = `0x69420` (from libc binary).
Computed: `libc_base = 0x7f3a4c069420 - 0x69420 = 0x7f3a4c000000`.
Now every libc symbol is known: `system = libc_base + 0x4c330`, `/bin/sh = libc_base + 0x196031`.

ASLR entropy for libc: reduced from 28 bits to 0 bits with a single leak.

## 5. Heap Exploitation Primitives (Graph Theory of Freelists)

### The Problem

Heap allocators maintain free chunks in linked lists (bins). Use-after-free, double-free, and heap overflow vulnerabilities corrupt these data structures, allowing an attacker to achieve arbitrary write or control flow hijack. The exploitation technique depends on the allocator's bin structure.

### The Formula

glibc malloc organizes free chunks into bins by size:

$$\text{bin type} = \begin{cases} \text{tcache} & \text{if size} \leq 1032 \text{ and count} \leq 7 \text{ (per-thread)} \\ \text{fastbin} & \text{if size} \leq 176 \text{ (LIFO, singly-linked)} \\ \text{unsorted} & \text{recently freed (doubly-linked)} \\ \text{small} & \text{if size} < 1024 \text{ (doubly-linked, exact fit)} \\ \text{large} & \text{if size} \geq 1024 \text{ (doubly-linked, sorted)} \end{cases}$$

A tcache poisoning attack corrupts the forward pointer of a freed tcache chunk:

$$\text{free}(A) \to \text{tcache}: A \to \text{NULL}$$

$$\text{overwrite } A\text{.fd} = T \quad (\text{target address})$$

$$\text{malloc}() \to A \quad \text{(consumes A)}$$

$$\text{malloc}() \to T \quad \text{(returns target address!)}$$

The safe-linking mitigation (glibc 2.32+) XORs the forward pointer:

$$\text{fd}_{\text{protected}} = \text{fd} \oplus ((\text{chunk\_addr}) \gg 12)$$

To bypass, the attacker needs the heap base address (partial leak of chunk address).

## 6. Sigreturn-Oriented Programming (SROP Register Control)

### The Problem

When available gadgets are too sparse for traditional ROP, SROP uses a single `sigreturn` syscall to set all registers simultaneously. The kernel restores register state from a `sigcontext` structure on the stack, giving the attacker complete control over all registers in a single frame.

### The Formula

The sigreturn syscall restores all registers from a `sigcontext` frame (296 bytes on x86-64):

$$\text{sigreturn}: \text{stack} \to \text{registers}$$

The frame controls:

$$\{RAX, RBX, RCX, RDX, RSI, RDI, RBP, RSP, R8\text{-}R15, RIP, \text{EFLAGS}, CS, SS, \ldots\}$$

Minimum gadgets needed for SROP:

$$|\mathcal{G}_{\text{SROP}}| = 2 \quad \text{(set RAX to 15, then syscall)}$$

compared to traditional execve ROP:

$$|\mathcal{G}_{\text{ROP}}| \geq 4 \quad \text{(pop rdi, pop rsi, pop rdx, syscall)}$$

The SROP payload size is fixed regardless of complexity:

$$|\text{SROP payload}| = 8_{\text{sigreturn gadget}} + 296_{\text{sigcontext}} = 304 \text{ bytes}$$

This is a constant, whereas traditional ROP chains grow linearly with the number of operations.

---

*The power of pwntools lies in its abstraction: it translates the mathematician's proof of exploitability into executable code, automating the tedious byte-level arithmetic of packing, offset calculation, and gadget chaining so the analyst can focus on the structural vulnerability rather than the mechanical details of the exploit.*

## Prerequisites

- Computer architecture (x86-64 calling convention, stack frame layout, register roles)
- Number theory (modular arithmetic for format string calculations)
- Information theory (entropy of ASLR, information content of pointer leaks)
- Graph theory (linked list structures in heap allocators, freelist traversal)
- Computability theory (Turing completeness of ROP gadget sets)
- Operating systems (virtual memory, signal handling, syscall interface)

## Complexity

- **Beginner:** Buffer overflow offset finding with cyclic patterns, basic ret2win exploits, using checksec to identify protections
- **Intermediate:** ret2libc with ASLR bypass via info leak, ROP chain construction, format string GOT overwrites, DynELF for unknown libc
- **Advanced:** Heap exploitation (tcache poisoning, house-of techniques), SROP, one-gadget constraints, kernel exploitation via ROP, bypassing CFI and shadow stacks
