# The Mathematics of GDB — Debugging Theory, Memory Layout & Breakpoint Mechanics

> *GDB is applied computer architecture. Every breakpoint is a patched instruction, every watchpoint a debug register, every backtrace a stack frame traversal. Understanding the hardware and ABI beneath the debugger transforms guessing into science.*

---

## 1. Breakpoint Mechanics — INT 3 and Software Traps

### Software Breakpoints

GDB implements breakpoints by replacing the target instruction with `INT 3` (opcode `0xCC`, 1 byte):

$$memory[addr] = \begin{cases} original\_byte & \text{normal execution} \\ 0xCC & \text{breakpoint set} \end{cases}$$

### Breakpoint Cost

| Operation | Mechanism | Cost |
|:---|:---|:---:|
| Set breakpoint | `ptrace(POKE_TEXT)` | 1-5 us |
| Hit breakpoint | CPU exception → signal → ptrace notify | 10-50 us |
| Continue | Restore byte, single-step, re-patch | 20-100 us |

### Maximum Software Breakpoints

$$max\_breakpoints = \frac{text\_segment\_size}{avg\_instruction\_size}$$

Practically unlimited (limited only by GDB memory). Each breakpoint stores:

$$storage = sizeof(addr) + sizeof(original\_byte) + metadata \approx 50 \text{ bytes}$$

1000 breakpoints: $\approx 50 KB$ in GDB memory.

---

## 2. Hardware Watchpoints — Debug Registers

### x86-64 Debug Registers

x86-64 provides exactly **4 debug registers** (DR0-DR3) for hardware watchpoints:

$$max\_hw\_watchpoints = 4$$

Each register watches an address range:

$$watched\_bytes \in \{1, 2, 4, 8\}$$

| Register | Purpose | Width |
|:---|:---|:---:|
| DR0-DR3 | Watchpoint addresses | 64-bit |
| DR6 | Debug status (which triggered) | 64-bit |
| DR7 | Control (enable, type, length) | 64-bit |

### Hardware vs Software Watchpoints

| Type | Speed | Coverage | Mechanism |
|:---|:---:|:---:|:---|
| Hardware | Full speed | 4 addresses, 1-8 bytes each | CPU debug registers |
| Software | ~1000x slower | Unlimited | Single-step + compare |

Software watchpoint cost:

$$T_{sw\_watch} = N_{instructions} \times T_{single\_step} \approx N \times 50\mu s$$

A loop of 1 million iterations: $50\mu s \times 10^6 = 50 \text{ seconds}$ (vs microseconds with hardware).

### Watchpoint Type Encoding (DR7)

$$type = \begin{cases} 00 & \text{Execute (breakpoint)} \\ 01 & \text{Write only} \\ 10 & \text{I/O (privileged)} \\ 11 & \text{Read/Write} \end{cases}$$

---

## 3. Stack Frame Traversal — Backtrace Algorithm

### Frame Pointer Chain

With frame pointers enabled (`-fno-omit-frame-pointer`), the stack is a linked list:

```
RBP → [saved_RBP] → [saved_RBP] → ... → NULL
       [return_addr]  [return_addr]
```

$$backtrace = \{(frame_i, return\_addr_i) : i = 0, 1, ..., depth\}$$

$$depth = \text{number of frames from current to } main()$$

### Traversal Cost

$$T_{backtrace} = depth \times T_{frame\_read}$$

Where $T_{frame\_read} \approx 1-5 \mu s$ (read 16 bytes via ptrace).

Typical depth 10-50: $T_{backtrace} = 10-250\mu s$.

### DWARF Unwinding (No Frame Pointer)

Without frame pointers (modern default with `-fomit-frame-pointer`), GDB uses **DWARF CFI** (Call Frame Information):

$$T_{DWARF} = depth \times (T_{CFI\_lookup} + T_{register\_restore})$$

$T_{CFI\_lookup} \approx 10-100\mu s$ (parse `.eh_frame` section, lookup by PC).

DWARF is slower but works with optimized code.

### Stack Size Bounds

$$stack\_usage = depth \times avg\_frame\_size$$

Default stack limit: 8 MB. Average frame: 100-500 bytes.

$$max\_depth \approx \frac{8 \times 10^6}{avg\_frame} = \frac{8MB}{200B} = 40,000 \text{ frames}$$

Stack overflow at depth $\approx 40,000$ with typical frames.

---

## 4. Memory Layout — Process Address Space

### Virtual Address Space Map

On x86-64 with 48-bit virtual addressing:

$$address\_space = 2^{48} = 256 \text{ TB}$$

Split: lower half (user: 0 to $2^{47}-1$), upper half (kernel: $2^{47}$ to $2^{48}-1$).

### Segment Layout

| Segment | Typical Start | Growth | Contains |
|:---|:---|:---:|:---|
| Text | 0x400000 | Fixed | Code (read-only, executable) |
| Data/BSS | After text | Fixed | Global/static variables |
| Heap | After BSS | ↑ (grows up) | malloc/new allocations |
| Memory maps | Middle | ↕ | mmap, shared libs |
| Stack | 0x7fff...fff | ↓ (grows down) | Local variables, frames |

### Alignment and Padding

GDB shows structure layout with padding:

$$sizeof(struct) = \sum field\_sizes + padding$$

$$padding_i = (align_i - (offset \mod align_i)) \mod align_i$$

**Example:**

```c
struct { char a; int b; char c; };
// a: offset 0 (1 byte) + 3 padding
// b: offset 4 (4 bytes)
// c: offset 8 (1 byte) + 3 padding
// Total: 12 bytes (not 6!)
```

$$wasted = sizeof(struct) - \sum field\_sizes = 12 - 6 = 6 \text{ bytes (50\% waste)}$$

Reordering fields (largest first) minimizes padding.

---

## 5. Expression Evaluation — GDB's Calculator

### Type Arithmetic

GDB follows C type promotion rules:

$$result\_type = \begin{cases} double & \text{if either operand is double} \\ long & \text{if either operand is long} \\ int & \text{otherwise (integer promotion)} \end{cases}$$

### Pointer Arithmetic

$$ptr + n = ptr + n \times sizeof(*ptr)$$

$$ptr_2 - ptr_1 = \frac{addr_2 - addr_1}{sizeof(*ptr)}$$

**Example:** `int *p` at address 0x1000:

$$p + 5 = 0x1000 + 5 \times 4 = 0x1014$$

### Memory Examination (x command)

`x/NFU addr` — examine $N$ units of format $F$ and size $U$:

$$bytes\_examined = N \times size(U)$$

| Unit | Size | Example |
|:---|:---:|:---|
| b (byte) | 1 | `x/16xb $rsp` — 16 hex bytes |
| h (halfword) | 2 | `x/8xh $rsp` — 8 hex shorts |
| w (word) | 4 | `x/4xw $rsp` — 4 hex ints |
| g (giant) | 8 | `x/2xg $rsp` — 2 hex longs |

---

## 6. Conditional Breakpoints — Evaluation Cost

### Condition Check Overhead

A conditional breakpoint `break foo if x > 10`:

1. Hit breakpoint (INT 3 trap): $T_{trap} \approx 20\mu s$
2. Evaluate condition: $T_{eval} \approx 5-50\mu s$
3. If false, continue: $T_{continue} \approx 20\mu s$

$$T_{conditional\_hit} = T_{trap} + T_{eval} + P(false) \times T_{continue}$$

### Impact on Hot Code

If breakpoint is in a loop executing $N$ times:

$$T_{overhead} = N \times (T_{trap} + T_{eval} + (1-P_{match}) \times T_{continue})$$

**Example:** Loop runs 1M times, condition matches 10 times:

$$T_{overhead} = 10^6 \times (20 + 10 + 0.99999 \times 20)\mu s \approx 50 \text{ seconds}$$

**Alternative:** Use hardware-assisted tracing or script the breakpoint to skip quickly.

---

## 7. Remote Debugging — GDB Protocol Cost

### RSP (Remote Serial Protocol) Overhead

Remote debugging adds network round-trips:

$$T_{operation} = T_{local} + 2 \times RTT + T_{encode/decode}$$

| Operation | Packets | Overhead at 1ms RTT |
|:---|:---:|:---:|
| Read register | 1 | 2 ms |
| Read 256 bytes | 1 | 2 ms |
| Single step | 2 | 4 ms |
| Backtrace (10 deep) | ~20 | 40 ms |
| Continue to breakpoint | 1 | 2 ms + execution time |

### Bandwidth for Core Dump

$$T_{core\_transfer} = \frac{core\_size}{bandwidth}$$

A 1 GB core dump over 100 Mbps link:

$$T = \frac{10^9}{12.5 \times 10^6} = 80 \text{ seconds}$$

---

## 8. Summary of GDB Mathematics

| Concept | Formula | Type |
|:---|:---|:---|
| Breakpoint patch | 1-byte INT 3 replacement | Instruction modification |
| HW watchpoints | Max 4 on x86-64 | Hardware limit |
| SW watchpoint cost | $N_{insn} \times 50\mu s$ | Single-step overhead |
| Backtrace depth | $stack\_size / avg\_frame$ | Division |
| Struct padding | $align - (offset \mod align)$ | Alignment math |
| Pointer arithmetic | $ptr + n \times sizeof(*ptr)$ | Type-aware |
| Conditional BP cost | $N \times (T_{trap} + T_{eval})$ | Linear in iterations |
| Remote overhead | $T_{local} + 2 \times RTT$ | Network latency |

---

*GDB is a conversation with the CPU through the lens of ptrace. Every command — break, watch, step, backtrace — maps to hardware features and OS primitives, and understanding that mapping is the difference between debugging and guessing.*
