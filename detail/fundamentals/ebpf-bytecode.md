# eBPF Bytecode -- Architecture, Verification, and Implementation Analysis

> *Deep dive into the eBPF instruction set architecture: verifier algorithm and abstract interpretation, JIT compilation pipeline, cBPF vs eBPF ISA comparison, instruction encoding mechanics, safety guarantees and termination proofs, and map implementation internals. From specification to kernel source.*

---

## Prerequisites

- Familiarity with eBPF concepts (registers, maps, program types, helpers)
- Understanding of CPU architecture (registers, instruction encoding, memory models)
- Basic knowledge of compiler concepts (intermediate representations, code generation)
- Linux kernel fundamentals (syscalls, memory management, concurrency)

## Complexity

| Topic | Analysis Type | Key Metric |
|:---|:---|:---|
| Verifier — abstract interpretation | Worst-case exponential, pruned | State space size, instruction limit |
| Verifier — state pruning | Amortized polynomial | Prune hit ratio, state comparison cost |
| JIT compilation | O(n) single pass | Instructions per BPF insn, register pressure |
| Hash map lookup | Amortized O(1) | Hash collisions, per-CPU lock contention |
| Array map lookup | O(1) | Cache line alignment |
| Ring buffer | O(1) enqueue | Consumer/producer cache bouncing |
| LPM trie lookup | O(prefix length) | Trie depth (max 128 for IPv6) |

---

## 1. cBPF vs eBPF -- ISA Comparison

### Classical BPF (cBPF)

The original Berkeley Packet Filter, designed in 1992 by McCanne and Jacobson,
was a simple packet filtering virtual machine:

| Property | cBPF | eBPF |
|:---|:---|:---|
| Registers | 2 (A: accumulator, X: index) | 11 (r0-r10) |
| Register width | 32-bit | 64-bit |
| Instruction size | 8 bytes | 8 bytes (16 for 64-bit imm loads) |
| Instruction encoding | opcode:16, jt:8, jf:8, k:32 | opcode:8, regs:8, offset:16, imm:32 |
| Program size limit | 4096 instructions | 1 million verified instructions |
| Calling convention | None | r1-r5 arguments, r0 return, r6-r9 callee-saved |
| Memory model | Single packet buffer, scratch memory (16 slots) | Stack (512B/frame), maps, packet data, context |
| Control flow | Forward jumps only, jt/jf per instruction | Forward jumps + bounded loops (5.3+) |
| Maps | None | 30+ map types |
| Helper functions | None | 200+ helpers |
| JIT | Simple (few architectures) | Full JIT on all major architectures |

### cBPF Instruction Encoding

```
 16 bits    8 bits   8 bits       32 bits
┌──────────┬────────┬────────┬──────────────┐
│  opcode  │  jt    │  jf    │      k       │
└──────────┴────────┴────────┴──────────────┘
```

- **opcode** (16 bits): operation class + operation + addressing mode.
- **jt** (8 bits): jump offset if condition is true.
- **jf** (8 bits): jump offset if condition is false.
- **k** (32 bits): generic constant (immediate, memory offset, etc.).

The jt/jf encoding is notable: every conditional instruction specifies both
branch targets. This makes cBPF programs inherently branchless-friendly but
limits expressiveness.

### eBPF Instruction Encoding

```
 8 bits    4 bits   4 bits    16 bits      32 bits
┌────────┬────────┬────────┬───────────┬──────────────┐
│ opcode │dst_reg │src_reg │  offset   │   immediate  │
└────────┴────────┴────────┴───────────┴──────────────┘
```

The opcode byte is further decomposed:

```
Bits 7-4: operation code (ADD, SUB, MOV, JEQ, etc.)
Bit  3:   source (0 = BPF_K immediate, 1 = BPF_X register)
Bits 2-0: instruction class (LD, LDX, ST, STX, ALU, JMP, JMP32, ALU64)
```

### Translation: cBPF to eBPF

The kernel transparently converts cBPF programs (e.g., from `setsockopt
SO_ATTACH_FILTER`) to eBPF before verification and execution. The
translation is implemented in `net/core/filter.c:bpf_convert_filter()`:

```
cBPF:  ld   [12]           →  eBPF:  ldxw r0, [r1 + 12]    (r1 = packet base)
cBPF:  jeq  #0x0800, 1, 0  →  eBPF:  jeq  r0, 0x0800, +1
                                      ja   +0
cBPF:  ret  #0xFFFF        →  eBPF:  mov  r0, 0xFFFF
                                      exit
cBPF:  ret  #0             →  eBPF:  mov  r0, 0
                                      exit
```

The cBPF accumulator maps to r0 (sometimes rA), the X register maps to r7.
cBPF's scratch memory (M[0]-M[15]) maps to the eBPF stack.

### Why eBPF Replaced cBPF

1. **Register-rich ISA.** 11 registers vs 2 eliminates most spilling. The
   calling convention enables efficient helper function calls without memory
   traffic.
2. **64-bit native.** cBPF's 32-bit limitations required awkward multi-
   instruction sequences for pointer arithmetic on 64-bit systems.
3. **Maps and state.** cBPF programs are stateless. eBPF maps enable
   communication between programs, userspace, and across invocations.
4. **Extensibility.** The helper function mechanism allows the kernel to
   expose new functionality without changing the ISA.

---

## 2. BPF Instruction Encoding -- Full Specification

### Opcode Structure

The 8-bit opcode encodes three fields:

```
For ALU/JMP classes:
  Bits 7-4: operation (BPF_ADD=0x0, BPF_SUB=0x1, BPF_MUL=0x2, ...)
  Bit  3:   source flag (BPF_K=0x0, BPF_X=0x8)
  Bits 2-0: class (BPF_ALU=0x4, BPF_ALU64=0x7, BPF_JMP=0x5, BPF_JMP32=0x6)

For memory classes:
  Bits 7-5: size (BPF_W=0x00, BPF_H=0x08, BPF_B=0x10, BPF_DW=0x18)
  Bits 4-3: mode (BPF_IMM=0x00, BPF_ABS=0x20, BPF_IND=0x40, BPF_MEM=0x60,
                   BPF_ATOMIC=0xC0)
  Bits 2-0: class (BPF_LD=0x0, BPF_LDX=0x1, BPF_ST=0x2, BPF_STX=0x3)
```

### Complete Opcode Table

**ALU64 operations** (class = 0x07):

| Opcode | Hex | Operation |
|:---|:---|:---|
| BPF_ADD \| BPF_X \| BPF_ALU64 | 0x0f | dst += src |
| BPF_ADD \| BPF_K \| BPF_ALU64 | 0x07 | dst += imm |
| BPF_SUB \| BPF_X \| BPF_ALU64 | 0x1f | dst -= src |
| BPF_SUB \| BPF_K \| BPF_ALU64 | 0x17 | dst -= imm |
| BPF_MUL \| BPF_X \| BPF_ALU64 | 0x2f | dst *= src |
| BPF_MOV \| BPF_X \| BPF_ALU64 | 0xbf | dst = src |
| BPF_MOV \| BPF_K \| BPF_ALU64 | 0xb7 | dst = imm |
| BPF_RSH \| BPF_X \| BPF_ALU64 | 0x7f | dst >>= src (logical) |
| BPF_ARSH \| BPF_X \| BPF_ALU64 | 0xcf | dst >>= src (arithmetic) |

**Jump operations** (class = 0x05):

| Opcode | Hex | Operation |
|:---|:---|:---|
| BPF_JA \| BPF_JMP | 0x05 | goto +offset |
| BPF_JEQ \| BPF_X \| BPF_JMP | 0x1d | if dst == src goto +offset |
| BPF_JEQ \| BPF_K \| BPF_JMP | 0x15 | if dst == imm goto +offset |
| BPF_JGT \| BPF_X \| BPF_JMP | 0x2d | if dst > src goto +offset |
| BPF_JSGT \| BPF_K \| BPF_JMP | 0x65 | if dst >s imm goto +offset |
| BPF_CALL | 0x85 | call helper (imm = helper ID) |
| BPF_EXIT | 0x95 | return r0 |

**Memory operations**:

| Opcode | Hex | Operation |
|:---|:---|:---|
| BPF_LDX \| BPF_MEM \| BPF_W | 0x61 | dst = *(u32 *)(src + off) |
| BPF_LDX \| BPF_MEM \| BPF_DW | 0x79 | dst = *(u64 *)(src + off) |
| BPF_STX \| BPF_MEM \| BPF_W | 0x63 | *(u32 *)(dst + off) = src |
| BPF_STX \| BPF_MEM \| BPF_DW | 0x7b | *(u64 *)(dst + off) = src |
| BPF_ST \| BPF_MEM \| BPF_W | 0x62 | *(u32 *)(dst + off) = imm |
| BPF_LD \| BPF_IMM \| BPF_DW | 0x18 | dst = imm64 (wide instruction) |

### Wide Instruction (64-bit Immediate Load)

The `BPF_LD | BPF_IMM | BPF_DW` instruction occupies 16 bytes -- two
consecutive 8-byte slots:

```
Slot 0: opcode=0x18, dst_reg, src_reg=0, offset=0, imm=lower_32_bits
Slot 1: opcode=0x00, regs=0, offset=0, imm=upper_32_bits
```

The full 64-bit value is `(upper_32 << 32) | lower_32`. When `src_reg` is
nonzero, it encodes special semantics:

- `src_reg = 1` (`BPF_PSEUDO_MAP_FD`): imm is a map file descriptor,
  relocated to a map pointer at load time.
- `src_reg = 2` (`BPF_PSEUDO_MAP_VALUE`): imm is a map fd + offset into
  the map's first value.
- `src_reg = 5` (`BPF_PSEUDO_FUNC`): imm is a BPF-to-BPF function offset.

### Byte Order

eBPF uses the **host byte order** for instruction encoding and register
values. The `BPF_END` operation provides explicit byte swapping:

```
BPF_ALU | BPF_END | BPF_TO_LE:  dst = htole(dst, imm)  // imm = 16, 32, or 64
BPF_ALU | BPF_END | BPF_TO_BE:  dst = htobe(dst, imm)
```

---

## 3. Verifier Algorithm -- Abstract Interpretation

The BPF verifier (`kernel/bpf/verifier.c`) is the critical safety gate. It
performs static analysis via abstract interpretation to prove that a program
is safe before any instruction executes.

### Abstract State

The verifier maintains an abstract state at each program point:

```c
struct bpf_verifier_state {
    struct bpf_reg_state regs[MAX_BPF_REG];  // r0-r10
    struct bpf_stack_state *stack;             // stack slot states
    u32 curframe;                              // current call frame
    // ...
};

struct bpf_reg_state {
    enum bpf_reg_type type;     // NOT_INIT, SCALAR_VALUE, PTR_TO_MAP_VALUE, ...
    s64 smin_value, smax_value; // signed range
    u64 umin_value, umax_value; // unsigned range
    s32 s32_min_value, s32_max_value; // 32-bit signed range
    u32 u32_min_value, u32_max_value; // 32-bit unsigned range
    struct tnum var_off;        // tristate number (known bits)
    u32 id;                     // for tracking related pointers
    s32 off;                    // offset from base pointer
    // ...
};
```

The `tnum` (tristate number) tracks known bit values: each bit is either
known-0, known-1, or unknown. This enables precise reasoning about bitwise
operations and masking.

### Register Types

The type system is the foundation of safety:

| Type | Meaning | Allowed Operations |
|:---|:---|:---|
| `NOT_INIT` | Uninitialized | None (read causes rejection) |
| `SCALAR_VALUE` | Integer, no pointer | Arithmetic, comparison, bounded use as offset |
| `PTR_TO_CTX` | Pointer to program context | Read fields at known offsets |
| `PTR_TO_MAP_VALUE` | Pointer into map value | Read/write within value bounds |
| `PTR_TO_MAP_VALUE_OR_NULL` | Result of `bpf_map_lookup_elem` | Must null-check before dereference |
| `PTR_TO_STACK` | Pointer to BPF stack | Read/write within frame |
| `PTR_TO_PACKET` | Pointer into packet data | Read within `[data, data_end)` after bounds check |
| `PTR_TO_PACKET_END` | End of packet data | Comparison with `PTR_TO_PACKET` only |
| `PTR_TO_BTF_ID` | Typed kernel pointer | Field access via BTF |

The type system enforces memory safety: a `PTR_TO_MAP_VALUE` can only be
dereferenced at offsets within the map value size. The verifier tracks the
offset and rejects out-of-bounds access statically.

### Verification Algorithm

The verifier implements a depth-first exploration of the program's control
flow graph:

```
verify_program(insns[]):
    worklist = [(insn_idx=0, state=initial_state)]

    while worklist not empty:
        (idx, state) = worklist.pop()

        if idx >= len(insns):
            reject("fell off end of program")

        if seen[idx] and state is subset of saved_state[idx]:
            continue  // STATE PRUNING — this path adds nothing new

        saved_state[idx] = merge(saved_state[idx], state)

        insn = insns[idx]
        new_state = simulate(insn, state)

        if insn is exit:
            check r0 is initialized and has valid type
            continue

        if insn is conditional jump:
            state_true  = refine(new_state, condition=true)
            state_false = refine(new_state, condition=false)
            worklist.push((idx + 1 + offset, state_true))
            worklist.push((idx + 1, state_false))

        else if insn is unconditional jump:
            worklist.push((idx + 1 + offset, new_state))

        else:
            worklist.push((idx + 1, new_state))

        total_insns++
        if total_insns > BPF_COMPLEXITY_LIMIT_INSNS:  // 1,000,000
            reject("program too complex")
```

### State Pruning -- Complexity Control

Without pruning, the verifier's complexity is exponential in the number of
branches. Consider a program with N conditional branches: naively, there are
2^N paths to explore.

**Pruning rule:** At instruction I, if we have previously verified state S1
and now arrive with state S2 where S2 is a subset of S1 (every register in
S2 has a type and range that is within the bounds already verified in S1),
then S2 is safe -- no need to re-verify from this point.

**Subset relation on register states:**

```
reg2 is subset of reg1 if:
  - reg2.type == reg1.type (or reg1 is more general)
  - reg2.umin_value >= reg1.umin_value
  - reg2.umax_value <= reg1.umax_value
  - reg2.smin_value >= reg1.smin_value
  - reg2.smax_value <= reg1.smax_value
  - reg2.var_off is more constrained than reg1.var_off
```

In practice, state pruning reduces the verified instruction count from
exponential to roughly linear for well-structured programs. The instruction
budget (1 million) is the hard backstop.

### Precision Tracking and Backtracking

The verifier uses **precision demand** to avoid unnecessarily tracking exact
register values. By default, register ranges are coarse. When a branch
condition depends on a register, the verifier backtracks through the program
to refine that register's range more precisely.

This is implemented as a backwards dataflow pass (`backtrack_insn()` in
`verifier.c`) that marks registers as "precise" when their values influence
control flow or memory access bounds. This optimization significantly reduces
the number of distinct states that need to be tracked.

### Termination Proof

The verifier guarantees termination through two mechanisms:

1. **DAG structure.** The main program body must be a directed acyclic graph
   (no backward jumps). This is checked before abstract interpretation
   begins. With no backward edges, the program must terminate in at most
   N instructions (where N is the program length).

2. **Bounded loops (kernel 5.3+).** Backward jumps are allowed if the
   verifier can prove the loop terminates. The verifier simulates the loop,
   tracking the loop variable's range. If after `BPF_COMPLEXITY_LIMIT_INSNS`
   simulated instructions the loop has not terminated, the program is
   rejected.

   The verifier does NOT compute a symbolic bound. It literally unrolls the
   loop during verification, consuming instruction budget. A loop that
   iterates 1000 times consumes 1000x the loop body from the million-
   instruction budget.

### Pointer Arithmetic Safety

The verifier tracks pointer arithmetic precisely:

```c
// r1 = PTR_TO_MAP_VALUE, map value size = 100
r2 = *(u32 *)(r1 + 0)     // OK: offset 0, size 4, within [0, 100)
r2 = *(u32 *)(r1 + 96)    // OK: offset 96, size 4, within [0, 100)
r2 = *(u32 *)(r1 + 97)    // REJECTED: offset 97, size 4, end=101 > 100
r2 = *(u32 *)(r1 + r3)    // OK only if r3 range is provably within [0, 96]
```

For variable-offset access, the verifier uses the intersection of the
register's signed/unsigned ranges and tnum to compute the possible offset
range, then verifies that the entire range is within bounds.

---

## 4. JIT Compilation Pipeline

### Architecture

The JIT compiler (`arch/x86/net/bpf_jit_comp.c` for x86_64) translates
verified eBPF bytecode to native machine code in a multi-pass pipeline:

```
Verified BPF bytecode
  → Pass 1: Compute native code size (offsets for jumps)
  → Pass 2: Emit native instructions with correct jump offsets
  → (Possibly repeat if sizes changed due to short/long jump encoding)
  → Allocate executable memory (bpf_jit_alloc_exec)
  → Copy native code
  → Set memory permissions (W^X)
  → Flush instruction cache
  → Return function pointer
```

### Register Mapping (x86_64)

```
BPF r0  →  rax    (return value, same as x86_64 ABI)
BPF r1  →  rdi    (arg1, same as x86_64 ABI)
BPF r2  →  rsi    (arg2)
BPF r3  →  rdx    (arg3)
BPF r4  →  rcx    (arg4)
BPF r5  →  r8     (arg5)
BPF r6  →  rbx    (callee-saved, same as x86_64 ABI)
BPF r7  →  r13    (callee-saved)
BPF r8  →  r14    (callee-saved)
BPF r9  →  r15    (callee-saved)
BPF r10 →  rbp    (frame pointer, callee-saved)
          r12    (used internally by JIT as temporary)
```

This mapping was chosen to align with the x86_64 System V ABI: BPF helper
function calls translate directly to native `call` instructions without
register shuffling for the first 5 arguments.

### Translation Examples

```
BPF: add64 r1, r2                    x86_64: add rdi, rsi
     (0x0f 0x12 0x00 0x00 0x00000000)        (48 01 f7)

BPF: mov64 r0, 42                    x86_64: mov eax, 42
     (0xb7 0x00 0x00 0x00 0x0000002a)        (b8 2a 00 00 00)

BPF: ldxdw r0, [r1+8]                x86_64: mov rax, [rdi+8]
     (0x79 0x10 0x08 0x00 0x00000000)        (48 8b 47 08)

BPF: jeq r1, 0, +5                   x86_64: test rdi, rdi
     (0x15 0x01 0x00 0x05 0x00000000)        (48 85 ff)
                                              je <+offset>
                                              (0f 84 xx xx xx xx)

BPF: call helper_id                   x86_64: mov rax, <helper_addr>
     (0x85 0x00 0x00 0x00 helper_id)         (48 b8 xx xx xx xx xx xx xx xx)
                                              call rax
                                              (ff d0)

BPF: exit                            x86_64: leave; ret
     (0x95 0x00 0x00 0x00 0x00000000)        (c9 c3)
```

### JIT Code Size

Typical expansion ratio: each BPF instruction generates 1-8 native
instructions. Most ALU and memory operations are 1:1 (BPF was designed for
efficient JIT). Branches expand more because x86_64 conditional jumps use
different encoding for short (8-bit offset) vs near (32-bit offset) targets.

The JIT performs multiple passes to resolve this: the first pass assumes all
jumps are near (4-byte offsets), then subsequent passes check if any can be
shortened to short jumps (1-byte offsets), which changes code offsets, which
may enable further shortening. This converges in 2-3 passes.

### Retpoline Mitigation

For indirect calls (tail calls via `BPF_MAP_TYPE_PROG_ARRAY`), the JIT emits
**retpoline** sequences on CPUs vulnerable to Spectre v2:

```asm
; Instead of: jmp *rax
call retpoline_rax
; ...
retpoline_rax:
    call .setup
.inner:
    lfence
    jmp .inner
.setup:
    mov [rsp], rax
    ret
```

This prevents the CPU from speculatively executing the indirect jump target,
at a performance cost of ~10-20 ns per tail call.

### Constant Blinding

When `bpf_jit_harden` is enabled (for unprivileged BPF), the JIT **blinds**
all immediate constants to prevent JIT spraying attacks:

```
Original:  mov r0, 0xdeadbeef
Blinded:   mov r0, 0xdeadbeef ^ random
           xor r0, random
```

This ensures that an attacker cannot embed arbitrary native instruction
sequences as immediate values in BPF programs.

---

## 5. Safety Guarantees and Termination Proofs

### Formal Safety Properties

The BPF verifier provides the following guarantees for any accepted program:

**1. Memory Safety.** Every memory access through a pointer is within the
bounds of its associated object (map value, stack frame, packet buffer,
context structure). The verifier tracks pointer provenance and bounds
statically.

**2. Type Safety.** Registers are typed. Pointer types cannot be forged from
scalars. A scalar value cannot be used as a pointer without first being
derived from a legitimate pointer source (map lookup, context access, etc.).

**3. Termination.** The program always reaches an `exit` instruction within
bounded time. For DAG programs, this is trivially O(N) where N is the program
size. For programs with bounded loops, termination is verified through
simulation with an instruction budget.

**4. Resource Confinement.** The program can only access resources (maps,
helpers, context fields) that its program type and attach type permit. An XDP
program cannot call `bpf_get_current_pid_tgid()` (a tracing helper).

**5. Stack Discipline.** Each function frame uses at most 512 bytes of stack.
The total stack depth is bounded by 512 * MAX_CALL_DEPTH = 512 * 8 = 4096
bytes. Stack reads and writes are bounds-checked.

### The Halting Problem Dodge

The halting problem proves that no algorithm can determine termination for
arbitrary programs. BPF sidesteps this by restricting the program model:

1. **Pre-5.3 kernels:** Only forward jumps allowed. The program is a DAG.
   Termination is trivially guaranteed -- any path through the DAG visits at
   most N instructions.

2. **Post-5.3 kernels:** Backward jumps (loops) are allowed but the verifier
   does not prove termination in the general sense. It **simulates** the
   loop, step by step, within the instruction budget. If simulation completes,
   the loop is safe. If the budget is exhausted, the program is rejected.

   This is sound but incomplete: some terminating programs with long loops
   will be rejected. The budget (1 million instructions) is chosen as a
   practical compromise.

### Liveness Analysis

The verifier performs liveness analysis to determine which registers are
"live" (will be read before being overwritten) at each program point. This
serves two purposes:

1. **Stronger pruning.** Dead registers can be ignored during state
   comparison. If r3 is dead at instruction I, two states that differ only
   in r3 are equivalent, enabling more pruning.

2. **Reduced state size.** Dead registers need not be tracked, reducing memory
   consumption during verification.

The liveness analysis is computed via a backward pass and stored as bitmask
annotations on each instruction.

---

## 6. Map Implementation Internals

### Hash Map (`BPF_MAP_TYPE_HASH`)

Implemented in `kernel/bpf/hashtab.c`.

**Structure:**

```
struct bpf_htab {
    struct bucket *buckets;     // hash table buckets
    u32 n_buckets;              // number of buckets (power of 2)
    u32 elem_size;              // size of each element
    u32 count;                  // current number of elements
    struct bpf_map map;         // common map fields (key_size, value_size, max_entries)
    // ...
};

struct bucket {
    struct hlist_nulls_head head;  // linked list head (hlist with null sentinels)
    raw_spinlock_t lock;           // per-bucket lock
};

struct htab_elem {
    struct hlist_nulls_node hash_node;  // list linkage
    u32 hash;                           // precomputed hash
    char key[] __aligned(8);            // key data (variable size)
    // value follows key, aligned
};
```

**Hash function:** The kernel uses `jhash` (Jenkins hash) or `siphash`
(for security-sensitive contexts) to hash keys. The hash is computed over
the raw key bytes.

**Bucket count:** Always a power of 2. The bucket index is
`hash & (n_buckets - 1)`. Default bucket count is `roundup_pow_of_two(max_entries)`
capped at implementation limits.

**Lookup (`bpf_map_lookup_elem`):**

```
1. hash = jhash(key, key_size, htab->hashrnd)
2. bucket = &htab->buckets[hash & (n_buckets - 1)]
3. Walk bucket's linked list:
   a. Compare stored hash (fast path rejection: hash mismatch → skip)
   b. Compare full key (memcmp)
   c. If match → return pointer to value
4. Return NULL if not found
```

This is O(1) amortized with O(n) worst case per bucket (hash collisions).
The per-bucket spinlock allows concurrent access to different buckets.

**Collision handling:** Linear chaining (linked list per bucket). The
`max_entries` limit prevents unbounded chain length -- once all entries are
allocated, updates to new keys fail with `-ENOMEM` (or replace via LRU in
`BPF_MAP_TYPE_LRU_HASH`).

**Preallocation vs on-demand:** By default, hash maps preallocate all
`max_entries` elements at map creation time. This avoids memory allocation
in the BPF program hot path (which runs in NMI or softirq context where
sleeping allocation is forbidden). The `BPF_F_NO_PREALLOC` flag enables
on-demand allocation for maps where most entries are never used.

### Per-CPU Hash Map (`BPF_MAP_TYPE_PERCPU_HASH`)

Same structure as the regular hash map, but each value slot holds
`num_possible_cpus()` copies, one per CPU. The BPF program accesses only
its CPU's copy, eliminating all locking on the read/write path.

```
Element layout in memory:
  [key][value_cpu_0][value_cpu_1]...[value_cpu_N]
```

Userspace reads/writes all CPU copies at once via `bpf_map_lookup_elem`:
the returned buffer contains `num_cpus * value_size` bytes. Aggregation
(e.g., summing counters) is done in userspace.

### Array Map (`BPF_MAP_TYPE_ARRAY`)

Implemented in `kernel/bpf/arraymap.c`.

```
struct bpf_array {
    struct bpf_map map;
    u32 elem_size;
    u32 index_mask;       // max_entries - 1 (power of 2 for masking)
    char value[] __aligned(8);  // contiguous array of values
};
```

**Lookup:** Pure index arithmetic -- `value + (index * elem_size)`. O(1) with
no hashing, no linked list traversal, no locking. The verifier ensures the
index is within bounds.

**Key constraint:** Keys are always `u32` indices from 0 to `max_entries - 1`.

**Cannot delete:** Array map entries cannot be deleted (`bpf_map_delete_elem`
returns `-EINVAL`). You can only update existing entries. This simplifies the
implementation and guarantees that lookup always returns a valid pointer for
in-bounds indices.

### Ring Buffer (`BPF_MAP_TYPE_RINGBUF`)

Implemented in `kernel/bpf/ringbuf.c`. Added in kernel 5.8.

The ring buffer is a single-producer (BPF program), single-consumer
(userspace) lock-free data structure optimized for event streaming:

```
Memory layout (shared between kernel and userspace via mmap):

  Consumer page:  [consumer_pos]                     // userspace updates this
  Producer page:  [producer_pos]                     // kernel updates this
  Data pages:     [record_0][record_1]...[record_N]  // power-of-2 size

Each record:
  [header: len(28 bits) | busy_bit(1) | discard_bit(1) | pg_off(2 bits) | padding]
  [data: variable length, 8-byte aligned]
```

**Producer (BPF side):**

```
bpf_ringbuf_reserve(ringbuf, size, flags):
  1. Read producer_pos (atomic)
  2. Check free space: (producer_pos - consumer_pos) < ringbuf_size
  3. Advance producer_pos by (header_size + aligned_size) (atomic)
  4. Set busy bit in record header
  5. Return pointer to data area

bpf_ringbuf_submit(data, flags):
  1. Clear busy bit in record header (atomic store with release semantics)
  2. If consumer is waiting, signal via eventfd/epoll
```

**Consumer (userspace):**

```
1. mmap the ring buffer pages (read-only for data, read-write for consumer page)
2. Read consumer_pos
3. While consumer_pos != producer_pos:
   a. Read record header at consumer_pos
   b. If busy bit set → stop (record not yet committed)
   c. If discard bit set → skip
   d. Process record data
   e. Advance consumer_pos by record size
   f. Write consumer_pos (atomic store with release semantics)
```

**Key properties:**

- No memory allocation in the BPF program path. The ring buffer is a
  pre-allocated contiguous region.
- No per-event synchronization (lock-free using atomic positions and the
  busy bit).
- Natural batching: userspace can drain many events per `epoll_wait` wakeup.
- Back-pressure: when the ring is full, `bpf_ringbuf_reserve` fails and
  the BPF program must handle the drop.

### LPM Trie (`BPF_MAP_TYPE_LPM_TRIE`)

Implemented in `kernel/bpf/lpm_trie.c`. Used for longest prefix match
operations (IP routing, CIDR lookups).

**Structure:** A compressed trie (radix tree) where each node stores a
prefix and two children (0-bit and 1-bit branches). Internal nodes represent
the longest common prefix of their subtrees.

```
struct lpm_trie_node {
    struct lpm_trie_node *child[2];  // left (0) and right (1)
    u32 prefixlen;                    // number of significant bits
    u8  data[];                       // prefix + value
};
```

**Lookup:** Walk the trie from root, following bits of the lookup key. At
each node, check if the node's prefix matches the key. The last matching
node with a value is the longest prefix match.

**Complexity:** O(W) where W is the maximum prefix length (32 for IPv4,
128 for IPv6). In practice, compressed trie nodes skip common prefix bits,
so average depth is much less.

---

## 7. Connections to Unheaded

The eBPF bytecode architecture directly underpins Unheaded's security and
observability infrastructure:

- **Shield eBPF** (cuirass service) compiles and loads BPF programs for
  network filtering and security enforcement. Understanding the ISA,
  verifier constraints, and map semantics is essential for writing correct
  and performant BPF programs.
- **XDP programs** in Unheaded's network path are JIT-compiled and run at
  line rate. The register mapping and JIT pipeline determine performance
  characteristics.
- **CO-RE and BTF** enable Unheaded's BPF programs to run across different
  kernel versions in heterogeneous LXD environments without recompilation.
- **Ring buffer maps** provide the event pipeline from kernel-space
  monitoring programs to userspace analysis in the sophia service.
- **BPF LSM** enforcement in cuirass uses the verifier's safety guarantees
  to ensure that security policies cannot crash the kernel.

---

## See Also

- linux-kernel-internals
- x86-assembly
- binary-and-number-systems
- networking-fundamentals

## References

- McCanne, S. & Jacobson, V. "The BSD Packet Filter: A New Architecture for User-level Packet Capture" (USENIX, 1993)
- Linux kernel source — BPF verifier: https://elixir.bootlin.com/linux/latest/source/kernel/bpf/verifier.c
- Linux kernel source — BPF hashtab: https://elixir.bootlin.com/linux/latest/source/kernel/bpf/hashtab.c
- Linux kernel source — BPF ringbuf: https://elixir.bootlin.com/linux/latest/source/kernel/bpf/ringbuf.c
- Linux kernel source — x86 JIT: https://elixir.bootlin.com/linux/latest/source/arch/x86/net/bpf_jit_comp.c
- BPF ISA specification: https://docs.kernel.org/bpf/standardization/instruction-set.html
- Brendan Gregg "BPF Performance Tools" (Addison-Wesley, 2019)
- Liz Rice "Learning eBPF" (O'Reilly, 2023)
- Cilium BPF reference: https://docs.cilium.io/en/latest/bpf/
- LWN.net "BPF: the universal in-kernel virtual machine" https://lwn.net/Articles/599755/
- LWN.net "Bounded loops in BPF programs" https://lwn.net/Articles/794934/
- LWN.net "BPF ring buffer" https://lwn.net/Articles/825415/
- Facebook/Meta BPF documentation: https://facebookmicrosites.github.io/bpf/
- eBPF foundation: https://ebpf.io/
