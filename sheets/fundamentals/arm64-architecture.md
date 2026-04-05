# ARM64 Architecture (AArch64)

A tiered guide to ARM64 processor architecture -- from phones to data centers.

## ELI5 (Explain Like I'm 5)

### A Simpler Kind of Brain

Most laptop and desktop computers use a type of brain called **x86** (made by Intel and AMD). It's powerful but complicated, like a Swiss Army knife with 100 tools.

ARM is a **simpler, more energy-efficient** type of brain. Instead of having 100 complicated tools, it has fewer, simpler tools that it uses really fast. Because it's simpler, it uses less electricity and generates less heat.

### Where You'll Find ARM

- **Every smartphone and tablet** -- your iPhone and Android phone run on ARM chips
- **Apple M-series chips** (M1, M2, M3, M4) -- the chips in modern MacBooks and iPads
- **AWS Graviton** -- Amazon's custom ARM chips that run cloud servers
- **Raspberry Pi** -- the tiny hobby computer
- **Game consoles** -- Nintendo Switch uses ARM
- **Smart watches, TVs, routers** -- ARM is everywhere

### Why ARM Won Mobile

Phones live on battery power. ARM chips do the same work as bigger chips but sip electricity instead of guzzling it. That's why your phone lasts all day on a tiny battery but a laptop needs a much bigger one.

## Middle School

### Registers -- The CPU's Scratch Paper

ARM64 has 31 general-purpose registers, each 64 bits wide:

```
# General-purpose registers
  x0  - x30    31 registers, each 64 bits wide
  w0  - w30    Same registers but only the lower 32 bits

# Special registers
  sp           Stack pointer (where the stack is right now)
  pc           Program counter (address of current instruction)
  xzr / wzr    Zero register (always reads as 0, writes are discarded)

# Example: x0 is 64-bit, w0 is the bottom 32 bits of x0
#   x0 = 0x00000001_DEADBEEF
#   w0 = 0xDEADBEEF (lower half only)
```

### Basic Instructions

ARM64 instructions are simple -- each one does exactly one thing:

```
# Moving data
  mov  x0, #42          // x0 = 42
  mov  x1, x0           // x1 = x0

# Arithmetic
  add  x0, x1, x2       // x0 = x1 + x2
  sub  x0, x1, x2       // x0 = x1 - x2
  mul  x0, x1, x2       // x0 = x1 * x2

# Loading from memory (RAM -> register)
  ldr  x0, [x1]         // x0 = memory at address in x1
  ldr  x0, [x1, #8]     // x0 = memory at (x1 + 8)

# Storing to memory (register -> RAM)
  str  x0, [x1]         // memory at address x1 = x0
  str  x0, [x1, #8]     // memory at (x1 + 8) = x0
```

### RISC = Simpler Instructions

ARM is a **RISC** processor (Reduced Instruction Set Computer):

```
# RISC principles:
# 1. Fixed-size instructions (always 4 bytes on ARM64)
# 2. Load/store architecture -- only ldr/str touch memory
#    (you can't add a register and a memory location directly)
# 3. Lots of registers (31 vs x86's 16)
# 4. Each instruction does one simple thing

# Contrast with CISC (x86):
# x86: add eax, [rbx+rcx*4+8]  -- one instruction loads AND adds
# ARM: ldr x0, [x1, x2, lsl #2] // load first
#      add x0, x0, x3            // then add -- two instructions
```

## High School

### AAPCS64 Calling Convention

When functions call other functions, everyone agrees on the rules:

```
# Argument registers (caller puts arguments here)
  x0 - x7       first 8 integer/pointer arguments
  d0 - d7       first 8 floating-point arguments

# Return values
  x0             integer/pointer return value
  x0, x1         pair return (for 128-bit values or two-word structs)
  d0             floating-point return value

# Callee-saved registers (function must preserve these)
  x19 - x28     if a function uses these, it must save and restore them
  x29 (fp)      frame pointer
  x30 (lr)      link register (return address)
  sp            stack pointer (must be restored)

# Caller-saved / scratch registers (may be clobbered by any call)
  x0 - x18      any function can destroy these
  x8             indirect result location (for large struct returns)
  x16, x17       intra-procedure-call scratch (linker veneers)
  x18            platform register (reserved on some OSes, e.g. macOS)
```

### Condition Flags and Comparisons

```
# The PSTATE condition flags (set by CMP, ADDS, SUBS, etc.)
  N    Negative     (result bit 63 is set)
  Z    Zero         (result is zero)
  C    Carry        (unsigned overflow or borrow)
  V    oVerflow     (signed overflow)

# Compare instruction (sets flags, discards result)
  cmp  x0, x1       // computes x0 - x1, sets N/Z/C/V
  cmp  x0, #10      // computes x0 - 10, sets flags

# Conditional branches
  b.eq  label        // branch if equal        (Z == 1)
  b.ne  label        // branch if not equal    (Z == 0)
  b.lt  label        // branch if less than    (N != V, signed)
  b.gt  label        // branch if greater than (Z==0 && N==V, signed)
  b.le  label        // branch if <= (signed)
  b.ge  label        // branch if >= (signed)
  b.lo  label        // branch if lower        (C == 0, unsigned)
  b.hi  label        // branch if higher       (C==1 && Z==0, unsigned)
```

### Conditional Select (Branchless Conditionals)

```
# csel -- conditional select (avoids branch misprediction)
  cmp   x0, x1
  csel  x2, x3, x4, gt   // x2 = (x0 > x1) ? x3 : x4

# csinc -- conditional select and increment
  cmp   x0, #0
  csinc x1, xzr, xzr, ne // x1 = (x0 != 0) ? 0 : 1  (boolean NOT)

# cset -- conditional set (syntactic sugar)
  cmp   x0, x1
  cset  x2, eq            // x2 = (x0 == x1) ? 1 : 0
```

### PC-Relative Addressing

```
# ARM64 uses PC-relative addressing for position-independent code
  adr   x0, label        // x0 = PC + offset to label (+-1 MB range)
  adrp  x0, label        // x0 = page base of label (+-4 GB range)
  add   x0, x0, #:lo12:label // add the 12-bit page offset

# This two-step pattern (adrp + add) is how ARM64 reaches any
# address within +-4 GB without needing a 64-bit immediate
# (since instructions are only 32 bits wide)
```

### Branch Instructions

```
# Unconditional branches
  b     label            // branch to label (+-128 MB range)
  bl    label            // branch and link -- saves return addr in x30 (lr)
  ret                    // return to address in x30 (lr)

# Register-indirect branches
  br    x0               // branch to address in x0
  blr   x0               // branch-and-link to address in x0
  ret   x0               // return to address in x0 (x30 is default)

# Compare and branch (no flags needed)
  cbz   x0, label        // branch if x0 == 0
  cbnz  x0, label        // branch if x0 != 0

# Test bit and branch
  tbz   x0, #5, label    // branch if bit 5 of x0 is 0
  tbnz  x0, #5, label    // branch if bit 5 of x0 is 1
```

### Stack Frame Layout

```
# Standard function prologue:
  stp   x29, x30, [sp, #-16]!  // push frame pointer and link register
  mov   x29, sp                 // set up frame pointer

# Standard function epilogue:
  ldp   x29, x30, [sp], #16    // pop frame pointer and link register
  ret                           // return to caller

# Stack grows downward (toward lower addresses)
# sp must be 16-byte aligned at all times (ABI requirement)

# Stack frame layout (high to low address):
#   +------------------+
#   | caller's frame   |
#   +------------------+
#   | saved x30 (lr)   |  <- x29 points here
#   | saved x29 (fp)   |
#   +------------------+
#   | saved registers   |
#   | (x19-x28 if used)|
#   +------------------+
#   | local variables   |
#   +------------------+  <- sp points here
```

## College

### NEON SIMD (128-bit Vectors)

```
# NEON provides 32 vector registers: v0-v31, each 128 bits wide
# Can be viewed as different element arrangements:

# Register views:
  v0.16b    // 16 x 8-bit  bytes
  v0.8h     //  8 x 16-bit halfwords
  v0.4s     //  4 x 32-bit singles (int or float)
  v0.2d     //  2 x 64-bit doubles

# NEON arithmetic (operate on all lanes simultaneously)
  add   v0.4s, v1.4s, v2.4s   // 4 parallel 32-bit adds
  fmul  v0.4s, v1.4s, v2.4s   // 4 parallel single-precision multiplies
  fmla  v0.4s, v1.4s, v2.4s   // fused multiply-add: v0 += v1 * v2

# Load/store
  ld1   {v0.4s}, [x0]         // load 4 floats from memory
  st1   {v0.4s, v1.4s}, [x0]  // store 8 floats (two registers)
  ld4   {v0.4s-v3.4s}, [x0]   // load 16 floats, deinterleave into 4 regs

# Useful operations
  dup   v0.4s, w1              // broadcast scalar w1 to all 4 lanes
  tbl   v0.16b, {v1.16b}, v2.16b  // permute bytes using index table
  addv  s0, v1.4s             // horizontal sum: s0 = sum of all 4 lanes
```

### SVE / SVE2 (Scalable Vector Extension)

```
# SVE uses vector-length-agnostic (VLA) programming
# Vector length is implementation-defined: 128 to 2048 bits
# Same binary runs on all implementations -- hardware picks width

# Predicate registers p0-p15 enable per-lane masking
  whilelt p0.s, x0, x1     // p0[i] = (x0+i < x1) for each lane

# SVE load/store with predicates
  ld1w  z0.s, p0/z, [x0]   // load 32-bit elements where p0 is true

# SVE arithmetic
  add   z0.s, z1.s, z2.s   // add vectors (width determined by hardware)
  fmla  z0.s, p0/m, z1.s, z2.s  // predicated fused multiply-add

# SVE2 adds fixed-point, crypto, and complex number operations
# Mandatory in ARMv9 (Neoverse V1, Cortex-X2 and later)
```

### Memory Model (Weakly Ordered)

```
# ARM64 has a RELAXED memory model (unlike x86 TSO)
# Loads and stores can be reordered freely unless constrained

# What can be reordered:
# - Load-Load:   yes (different addresses)
# - Load-Store:  yes
# - Store-Load:  yes
# - Store-Store: yes (different addresses)
# (x86 only reorders Store-Load)

# Acquire/Release semantics (preferred over full barriers)
  ldar  x0, [x1]            // load-acquire: no subsequent loads/stores
                             //   can be reordered before this
  stlr  x0, [x1]            // store-release: no preceding loads/stores
                             //   can be reordered after this

# ARMv8.3 adds LDAPR (load-acquire RCpc) for weaker acquire
# that allows reordering with earlier stores to different addresses
```

### Memory Barriers (DMB / DSB / ISB)

```
# DMB (Data Memory Barrier)
#   Ensures ordering of memory accesses before/after the barrier
#   Does NOT wait for completion -- just enforces order
  dmb  ish      // inner-shareable full barrier (most common)
  dmb  ishld    // inner-shareable load-load/load-store barrier
  dmb  ishst    // inner-shareable store-store barrier
  dmb  sy       // full system barrier (includes outer-shareable)

# DSB (Data Synchronization Barrier)
#   Like DMB but also waits for all prior memory accesses to complete
#   Required before cache/TLB maintenance to ensure visibility
  dsb  ish      // wait for inner-shareable accesses to complete
  dsb  sy       // wait for all accesses to complete

# ISB (Instruction Synchronization Barrier)
#   Flushes the instruction pipeline
#   Required after modifying instructions, system registers, or TLB
  isb           // flush pipeline, refetch from cache/memory
```

### Exclusive Monitors (Atomics)

```
# ARM64 atomics use load-exclusive / store-exclusive pairs
# The "exclusive monitor" tracks whether the address was written by another core

# Load-exclusive / Store-exclusive
  ldxr  x0, [x1]            // load-exclusive: begin tracking [x1]
  stxr  w2, x0, [x1]        // store-exclusive: w2=0 if success, 1 if fail
                             //   fails if another core wrote to [x1]

# Typical CAS (compare-and-swap) loop:
# retry:
#   ldxr   x0, [x1]         // load current value
#   cmp    x0, x2            // compare with expected
#   b.ne   fail              // not equal, CAS fails
#   stxr   w3, x4, [x1]     // try to store new value
#   cbnz   w3, retry         // store failed (contention), retry
# success:

# ARMv8.1 adds LSE (Large System Extensions) -- hardware atomics:
  cas    x0, x1, [x2]       // compare-and-swap (hardware CAS)
  ldadd  x0, x1, [x2]       // atomic add: x0 = old [x2], [x2] += x1
  swp    x0, x1, [x2]       // atomic swap
  stadd  x0, [x1]           // atomic add, no return value

# LSE atomics are much faster under contention than ldxr/stxr loops
# because they avoid the retry loop and reduce cache line bouncing
```

### Pointer Authentication (PAC)

```
# ARMv8.3 feature -- embeds a cryptographic signature in unused
# pointer bits to detect corruption/tampering (ROP/JOP attacks)

# How it works:
# 1. Sign a pointer with a key + context (modifier)
#    paciasp            // sign LR with key A, using SP as context
#    pacia x0, x1       // sign x0 with key A, using x1 as context
#
# 2. Verify before use
#    autiasp            // verify LR with key A + SP context
#    autia x0, x1       // verify x0 with key A + x1 context
#    retaa              // authenticate LR + return (combined)
#
# 3. If authentication fails, pointer is corrupted -> fault on use

# 5 keys: APIAKey, APIBKey, APDAKey, APDBKey, APGAKey (generic)
# Keys are per-process, set by the kernel

# Combined instructions (atomic sign-and-branch):
  blraaz x0             // authenticate x0 with zero context, then branch-link
  braaz  x0             // authenticate x0, then branch
```

### MTE (Memory Tagging Extension)

```
# ARMv8.5 feature -- hardware-assisted memory safety
# Detects use-after-free and buffer overflows at runtime

# How it works:
# 1. Memory is divided into 16-byte "granules"
# 2. Each granule has a 4-bit tag stored in separate tag memory
# 3. Pointers also carry a 4-bit tag in bits [59:56]
# 4. On every load/store, hardware checks: pointer tag == memory tag
# 5. Mismatch -> synchronous fault or asynchronous report

# Tag instructions:
  irg   x0, sp          // insert random tag into x0 (allocator use)
  stg   x0, [x0]        // store tag from x0 to memory at [x0]
  ldg   x0, [x1]        // load tag from memory at [x1] into x0

# Three modes:
# - Synchronous:  precise fault on mismatch (debug, ~3-5% overhead)
# - Asynchronous: flag set in TFSR, checked later (~1-2% overhead)
# - Asymmetric:   sync for reads, async for writes
```

### BTI (Branch Target Identification)

```
# ARMv8.5 feature -- forward-edge CFI (Control Flow Integrity)
# Prevents JOP (Jump-Oriented Programming) attacks

# Mark valid branch targets with BTI instructions:
  bti  c     // valid target for BL/BLR (calls)
  bti  j     // valid target for BR (jumps)
  bti  jc    // valid target for both calls and jumps

# If code branches to an instruction that is NOT a BTI
# (and the memory region is marked as BTI-enforced),
# the processor raises a fault

# Typically combined with PAC for full CFI:
# - PAC protects backward edges (return addresses)
# - BTI protects forward edges (indirect calls/jumps)
```

### big.LITTLE and DynamIQ

```
# Heterogeneous multi-processing: mix fast and efficient cores
# on the same chip, sharing a unified memory system

# big.LITTLE (original, ARMv7/v8):
#   "big" cores:    high-performance, high-power (Cortex-A76/A77/X-series)
#   "LITTLE" cores: energy-efficient, low-power (Cortex-A55/A510)
#   Scheduler migrates threads based on load

# DynamIQ (successor):
#   - Cores in a single cluster can be different types
#   - Shared L3 cache across big and LITTLE cores
#   - Fine-grained per-core DVFS
#   - Up to 8 cores per cluster, multiple clusters per chip

# Apple's approach:
#   M1: 4 Firestorm (performance) + 4 Icestorm (efficiency)
#   M3: 4 P-cores + 4 E-cores, plus 10-core GPU
#   Scheduler pins background tasks to E-cores,
#   foreground interactive work to P-cores

# AWS Graviton:
#   Graviton3: 64 Neoverse V1 cores (all same type, no big.LITTLE)
#   Focus on throughput per watt for cloud workloads
#   ~25% better compute perf, ~60% less energy vs comparable x86
```

## Tips

- ARM64 instructions are always 4 bytes -- if you're debugging and see misaligned instructions, something is very wrong
- Use `ldar`/`stlr` (acquire/release) instead of raw `dmb` barriers when implementing atomics -- they're more precise and often faster
- LSE atomics (`cas`, `ldadd`, `swp`) are vastly faster than `ldxr`/`stxr` loops under contention -- ensure your toolchain targets ARMv8.1+
- On Apple M-series, the efficiency cores have a smaller reorder buffer and narrower pipeline -- be aware that perf characteristics differ per core type
- When porting x86 code to ARM64, the biggest pitfall is the relaxed memory model -- code that "accidentally works" on x86 TSO will break on ARM

## See Also

- how-computers-work
- binary-and-number-systems
- linux-kernel-internals

## References

- ARM Architecture Reference Manual for A-profile architecture (DDI 0487)
- ARM Cortex-A Series Programmer's Guide for ARMv8-A (DEN0024A)
- ARM Procedure Call Standard for AArch64 (AAPCS64, IHI 0055)
- ARM NEON Programmer's Guide (DEN0018A)
- ARM Scalable Vector Extension Programmer's Guide (Version 1.2)
- "Programming with 64-Bit ARM Assembly Language" by Stephen Smith
- Apple Silicon documentation: developer.apple.com/documentation/apple-silicon
