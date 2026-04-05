# x86-64 Assembly Internals -- From Instruction Encoding to Microarchitectural Optimization

> *The x86-64 instruction set is a living fossil: forty years of backward compatibility layered onto a modern superscalar core. Understanding the encoding and execution machinery beneath the ISA is what separates writing assembly from writing fast assembly.*

---

## 1. Instruction Encoding

x86-64 instructions are variable-length (1-15 bytes). Every instruction is built from up to six fields, some optional:

```
[Legacy Prefixes] [REX Prefix] [Opcode] [ModR/M] [SIB] [Displacement] [Immediate]

# Not all fields are present in every instruction.
# Example: "add rax, rbx" encodes as just 3 bytes: 48 01 d8
```

### Legacy Prefixes (0-4 bytes)

Up to four groups, at most one prefix from each group:

```
# Group 1 (lock/rep):
  F0 = LOCK       (atomic read-modify-write)
  F2 = REPNE/REPNZ (string repeat)
  F3 = REP/REPE   (string repeat / SSE prefix)

# Group 2 (segment override):
  2E = CS, 36 = SS, 3E = DS, 26 = ES, 64 = FS, 65 = GS
  # In 64-bit mode, CS/DS/ES/SS overrides are ignored
  # FS/GS are used for thread-local storage (TLS)

# Group 3 (operand-size override):
  66 = toggle between 16-bit and 32-bit operand size
  # Also used as a mandatory prefix for some SSE/AVX instructions

# Group 4 (address-size override):
  67 = toggle between 32-bit and 64-bit address size
```

### REX Prefix (0-1 byte)

Required for accessing r8-r15, sil, dil, bpl, spl, or 64-bit operand size:

```
# REX byte format: 0100 WRXB
  0100 = fixed pattern (distinguishes REX from other opcodes)
  W    = 1: 64-bit operand size; 0: default operand size
  R    = extends ModR/M reg field (3 bits → 4 bits)
  X    = extends SIB index field
  B    = extends ModR/M r/m field or SIB base field

# Example: mov rax, rbx
  REX.W=1, REX.R=0, REX.B=0 → REX byte = 0x48
  Opcode 89 (mov r/m64, r64)
  ModR/M = 0xD8 (mod=11 reg=011=rbx rm=000=rax)
  Full encoding: 48 89 D8 (but 48 01 D8 is add — different opcode)

# VEX/EVEX prefixes (AVX/AVX-512) replace REX + legacy prefixes:
  VEX:  2 or 3 bytes, encodes REX bits + implied 0F prefix + vector length
  EVEX: 4 bytes, adds opmask, broadcast, rounding, and zmm access

# REX2 prefix (APX extension, proposed):
  Extends GPR set to 32 registers (r16-r31)
  2-byte prefix with additional register extension bits
```

### ModR/M Byte (0-1 byte)

Specifies operand addressing:

```
# ModR/M format: [mod:2][reg:3][r/m:3]

# mod field:
  00 = [r/m]           memory, no displacement (except special cases)
  01 = [r/m + disp8]   memory, 8-bit displacement
  10 = [r/m + disp32]  memory, 32-bit displacement
  11 = r/m is register (register-to-register operation)

# reg field: register operand or opcode extension
# r/m field: register or memory operand

# Special cases when mod=00:
  r/m=100 (rsp/r12): SIB byte follows
  r/m=101 (rbp/r13): RIP-relative addressing (disp32 follows)

# Example: mov rax, [rbx+8]
  mod=01 (disp8), reg=000 (rax), r/m=011 (rbx)
  ModR/M = 0x43, displacement = 0x08
  Full: 48 8B 43 08
```

### SIB Byte (0-1 byte)

Used when the ModR/M r/m field is 100 (indicating SIB follows):

```
# SIB format: [scale:2][index:3][base:3]

# Encodes: [base + index * scale]
  scale: 00=1, 01=2, 10=4, 11=8
  index: register number (100=none, meaning no index)
  base:  register number (101 with mod=00 = disp32 only)

# Example: mov rax, [rbx + rcx*8 + 16]
  mod=01 (disp8), r/m=100 (SIB follows)
  SIB: scale=11(8), index=001(rcx), base=011(rbx)
  ModR/M = 0x44, SIB = 0xCB, disp8 = 0x10
  Full: 48 8B 44 CB 10

# Why does rsp (r/m=100) trigger SIB?
#   Historical: rsp encoding in r/m was repurposed as "SIB follows" escape.
#   To actually use rsp as base: SIB with base=100 (rsp), index=100 (none).
#   This is why [rsp] requires 1 extra byte vs [rbx].
```

### Displacement and Immediate

```
# Displacement: 0, 1, or 4 bytes (signed offset for memory addressing)
  disp8:  -128 to +127    (1 byte, saves encoding space for small offsets)
  disp32: -2^31 to 2^31-1 (4 bytes)

# Immediate: 0, 1, 2, 4, or 8 bytes (constant operand)
  imm8:  1 byte
  imm16: 2 bytes
  imm32: 4 bytes (sign-extended to 64 bits for most 64-bit operations)
  imm64: 8 bytes (only for mov r64, imm64 — "movabs")

# Size optimization: the assembler picks the shortest encoding
  add rax, 1    → 48 83 C0 01  (uses imm8 sign-extended to 64 bits)
  add rax, 256  → 48 05 00 01 00 00  (needs imm32)
```

### Encoding Worked Example

Encoding `add qword [rbx + rcx*4 + 0x20], 7` step by step:

```
# 1. Prefixes: REX.W=1 for 64-bit → 0x48
# 2. Opcode: add r/m64, imm8 → 0x83 (opcode extension /0 in reg field)
# 3. ModR/M: mod=01 (disp8), reg=000 (/0), r/m=100 (SIB) → 0x04
# 4. SIB: scale=10 (4), index=001 (rcx), base=011 (rbx) → 0x8B
# 5. Displacement: 0x20 (8-bit)
# 6. Immediate: 0x07 (8-bit)
#
# Final encoding: 48 83 04 8B 20 07  (6 bytes)
```

---

## 2. Micro-Op Fusion

Modern x86-64 CPUs decompose complex CISC instructions into simpler RISC-like micro-operations (uops). But some instruction pairs can be fused back together.

### Macro-Op Fusion

The decoder fuses two consecutive instructions into a single uop:

```
# Fused pairs (Intel since Core 2, AMD since Zen):
  cmp/test + conditional jump  → single compare-and-branch uop

# Example:
  cmp rax, rbx    # these two instructions
  je  target       # become ONE uop in the decoder

# Conditions for fusion (Intel):
  - First instruction: CMP, TEST, ADD, SUB, AND, INC, DEC
  - Second instruction: any Jcc (conditional jump)
  - Both must be in the same 16-byte decode window
  - ADD/SUB/AND fusion only on some microarchitectures

# AMD Zen conditions:
  - TEST/CMP + Jcc always fuse
  - ADD/SUB + Jcc fuse on Zen 2+

# Impact: effectively doubles decode bandwidth for branch-heavy code
# A 4-wide decoder can process 5 instructions per cycle when fusion occurs
```

### Micro-Op Fusion (Intel)

A single complex instruction that would produce 2 uops is kept as 1 fused uop through the pipeline:

```
# Fused-domain uops:
  add rax, [rbx]    # load + add = 1 fused uop (unfuses at execution)
  cmp [rdi], rax     # load + compare = 1 fused uop

# Non-fusable:
  add [rbx], rax     # load + add + store = 2+ uops (RMW requires store)
  add rax, [rbx+rcx*4+disp32]  # complex addressing may prevent fusion
                                # (RIP-relative and indexed often don't fuse)

# Why it matters:
#   Rename/allocate width is limited (e.g., 6 uops/cycle on Zen 4)
#   Fused uops count as 1 through rename, unfuse at dispatch to ports
#   This effectively increases front-end throughput
```

### Uop Cache (Decoded Stream Buffer / DSB)

```
# The uop cache stores already-decoded micro-ops:
#   Intel: ~1.5K-4K uop entries (since Sandy Bridge)
#   AMD:   Op Cache, ~4K entries (since Zen)

# Bypasses the decode stage entirely for hot loops
# Bandwidth: typically 6-8 uops/cycle from uop cache vs 4-5 from decoders

# Pathological cases that break uop cache efficiency:
#   - Instructions crossing 64-byte cache line boundaries
#   - Very long instructions (>8 bytes eat cache capacity)
#   - Self-modifying code invalidates uop cache entries
#   - LOCK-prefixed instructions in some microarchitectures
```

---

## 3. Out-of-Order Execution Internals

### The Full Pipeline (Simplified Modern x86-64)

```
# Front end:
  1. Instruction Fetch (from I-cache or uop cache)
  2. Instruction Decode (up to 4-6 instructions/cycle)
     - Simple decoder: 1 instruction → 1 uop
     - Complex decoder: 1 instruction → 1-4 uops
     - MSROM (Microcode Sequencer): >4 uops (div, cpuid, etc.)
  3. Macro-op Fusion (fuse cmp+jcc pairs)
  4. Micro-op Fusion (fuse load+op pairs)

# Allocation/Rename:
  5. Register Rename (map architectural → physical registers via RAT)
  6. Allocate ROB entry, reservation station slot, load/store buffer entry

# Back end (out-of-order):
  7. Dispatch to reservation stations (scheduler)
  8. Issue to execution ports when operands ready (out-of-order)
  9. Execute on functional units (ALU, FPU, AGU, load/store)
  10. Write back results, wake dependent uops

# Retirement (in-order):
  11. Reorder Buffer (ROB) retires uops in program order
  12. Architectural state updated, physical registers freed
```

### Key Microarchitectural Buffers

```
# Resource               Intel (Golden Cove)    AMD (Zen 4)
# ROB entries            512                    320
# Physical int regs      280                    224
# Physical FP/vec regs   332                    192
# Reservation stations   ~160 (distributed)     ~128 (distributed)
# Load buffer entries    192                    88
# Store buffer entries   114                    64
# Decode width           6 uops/cycle           4 inst → 6+ uops/cycle
# Rename width           6 uops/cycle           6 uops/cycle
# Retire width           6 uops/cycle           6 uops/cycle
# Scheduler ports        12 execution ports     6 ALU + 3 AGU + ...

# When any of these buffers fills up, the pipeline stalls.
# Long-latency operations (cache misses, divides) occupy entries
# for many cycles, reducing effective out-of-order window.
```

### Execution Ports and Throughput

```
# Each uop dispatches to a specific execution port.
# Throughput is limited by port contention.

# Intel Golden Cove example ports:
  Port 0: ALU, Vec ALU, FMA, DIV
  Port 1: ALU, Vec ALU, FMA
  Port 2: Load, AGU
  Port 3: Load, AGU
  Port 4: Store data
  Port 5: ALU, Vec ALU, Vec shuffle
  Port 6: ALU, Branch
  Port 7: Store AGU
  Port 8: Store AGU
  Port 9: Store data
  Port 10: ALU, Vec ALU
  Port 11: Load, AGU

# Throughput example:
#   add r64, r64  → any of ports 0,1,5,6,10 → throughput: 5/cycle
#   imul r64, r64 → port 1 only → throughput: 1/cycle
#   div r64       → port 0, ~35-90 cycles → throughput: 1/35-90 cycles
#   vfmadd*ps     → ports 0,1 → throughput: 2/cycle
```

---

## 4. x86-64 vs ARM64 (AArch64) Architectural Comparison

### ISA-Level Differences

| Property | x86-64 | ARM64 (AArch64) |
|:---|:---|:---|
| Instruction length | Variable, 1-15 bytes | Fixed, 4 bytes |
| Endianness | Little-endian | Bi-endian (little default) |
| GP registers | 16 (rax-r15) | 31 (x0-x30) + xzr/sp |
| SIMD registers | 16 xmm/ymm (32 zmm with AVX-512) | 32 v0-v31 (128-bit NEON/SVE) |
| Condition codes | Flags set by most ALU ops | Flags set only by explicit S-suffix |
| Memory model | TSO (strong) | Weakly ordered |
| Addressing modes | Complex (base+index*scale+disp) | Simpler (base+offset, base+reg) |
| Load/store | Register-memory for most ops | Load/store architecture |
| Predication | Conditional moves (cmov) | Full conditional select + compare |
| Unaligned access | Always supported (may be slower) | Supported but configurable |

### Decode Complexity

```
# x86-64 decode challenges:
  - Variable-length instructions require length pre-decode
  - Must find instruction boundaries before decoding
  - Prefix bytes change instruction meaning
  - Legacy compatibility: thousands of encoding special cases
  - Solution: uop cache bypasses decode for hot code

# ARM64 decode advantages:
  - Fixed 4-byte instructions: trivial to find boundaries
  - Parallel decode of 6-8 instructions is straightforward
  - No prefix complexity
  - Result: simpler decode hardware, lower power

# ARM64 challenges:
  - Fixed length wastes bits on simple instructions
  - Thumb-2 (AArch32) added variable length, but AArch64 dropped it
  - SVE/SVE2 instructions are still fixed 4-byte but encode complex ops
```

### Register Architecture

```
# x86-64 has fewer registers but compensates with:
  - Aggressive register renaming (~180-280 physical regs)
  - Memory operands in ALU instructions (fewer explicit loads)
  - Higher code density (fewer register spills visible in instruction stream)

# ARM64 has more registers, which means:
  - Fewer register spills and loads
  - More function arguments in registers (x0-x7 = 8 args vs 6 on x86-64)
  - Simpler calling convention
  - Less pressure on rename hardware

# Impact: on register-heavy code (e.g., interpreters), ARM64's 31 regs
# provide a measurable advantage. On memory-heavy code, x86-64's ability
# to fold loads into ALU ops narrows the gap.
```

### Memory Ordering Comparison

```
# x86-64 (TSO -- Total Store Order):
  - Loads are never reordered with other loads
  - Stores are never reordered with other stores
  - Loads can be reordered with older stores (to different addresses)
  - Stores are not reordered with older loads
  - Result: most lock-free code "just works"
  - Cost: store buffer must check all pending stores on every load

# ARM64 (Weakly ordered):
  - Both loads and stores can be reordered freely
  - Must use explicit barriers: DMB, DSB, ISB
  - Or use acquire/release semantics: LDAR/STLR instructions
  - Result: hardware has more freedom, better power efficiency
  - Cost: programmer must think harder about memory ordering

# Practical impact:
#   Porting lock-free code from x86 to ARM is a common source of bugs.
#   Code that "works" on x86 due to TSO may fail on ARM without barriers.
#   C11/C++11 atomics abstract over the difference.
```

### Performance Characteristics

```
# Perf/watt:
#   ARM64 (Apple M-series, Graviton) leads significantly
#   x86-64 (Intel/AMD) catches up with hybrid architectures (E/P cores)
#   ARM advantage comes from simpler decode + more registers + less legacy

# Peak single-thread:
#   Apple M4 Firestorm cores: ~8-wide decode, very high IPC
#   Intel Golden Cove: 6-wide decode, high IPC
#   AMD Zen 4: 4-wide decode → 6+ uops, competitive IPC
#   Competitive, with Apple often leading on integer IPC

# Vectorization:
#   x86-64: SSE (128b), AVX (256b), AVX-512 (512b) -- mature ecosystem
#   ARM64: NEON (128b fixed), SVE/SVE2 (128-2048b scalable)
#   SVE's vector-length-agnostic model is more elegant but newer
#   AVX-512 has broader software support as of 2025
```

---

## 5. Agner Fog's Optimization Guides -- Summary

Agner Fog (Technical University of Denmark) maintains the most comprehensive public microarchitecture documentation. The guides cover Intel, AMD, and VIA CPUs from Pentium through the latest generations.

### Guide 1: Optimizing Software in C++

Key principles:

```
# 1. Profile before optimizing
#    Use perf, VTune, or Instruments to find actual bottlenecks.
#    Most code spends 90% of time in 10% of code.

# 2. Data structures and algorithms dominate
#    No amount of micro-optimization saves a bad algorithm.
#    O(n log n) beats O(n^2) regardless of SIMD.

# 3. Cache-friendly data layout
#    Array of Structs (AoS) vs Struct of Arrays (SoA):
#      AoS: [{x,y,z}, {x,y,z}, ...]  -- bad if you only need x values
#      SoA: {[x,x,...], [y,y,...], [z,z,...]}  -- sequential access to x
#    Hot/cold splitting: put rarely-used fields in a separate struct.

# 4. Branch prediction awareness
#    Sort data before conditional processing when practical.
#    Use branchless code (cmov, ternary → arithmetic) for unpredictable branches.
#    Profile-guided optimization (PGO) lets compiler reorder for prediction.

# 5. Minimize memory allocations
#    Allocate pools/arenas instead of many small malloc calls.
#    Avoid pointer-chasing data structures (linked lists, trees with node alloc).
```

### Guide 2: Optimizing Assembly

```
# Register usage:
#   - Prefer eax over rax when value fits in 32 bits (shorter encoding)
#   - Use r8-r15 last (require REX prefix = 1 extra byte)
#   - xor eax, eax is best zero idiom (3 bytes, breaks deps)
#   - lea for address arithmetic: lea rax, [rbx+rcx*2+4] avoids mul

# Instruction selection:
#   - Avoid partial register stalls: don't write ah/al then read eax
#   - Prefer test over cmp for zero checks: test eax,eax vs cmp eax,0
#   - Use movzx/movsx explicitly (avoid relying on implicit extension)
#   - imul r, r/m, imm is 1 uop; mul r/m is 2+ uops with implicit rdx:rax

# Loop optimization:
#   - Align loop entry to 32-byte boundary (or 64 for uop cache)
#   - Unroll to fill execution ports and hide latency
#   - Use dec+jnz instead of sub+cmp+jne (macro-fuses on all modern CPUs)
#   - Strength reduction: replace mul by shift+add when constant is known
```

### Guide 3: Microarchitecture Reference

```
# What the guide covers for each CPU generation:
#   - Pipeline depth and width
#   - Decode, rename, dispatch, retire bandwidth
#   - Execution port assignments for every instruction class
#   - Branch predictor type and history length
#   - Cache sizes, associativity, latency at each level
#   - TLB structure and miss penalty
#   - Store buffer size and forwarding rules

# Key patterns across generations:
#   - Decode width has grown: 3 → 4 → 5 → 6 instructions/cycle
#   - ROB has grown: 128 → 224 → 352 → 512 entries
#   - Branch predictor: bimodal → gshare → TAGE → perceptron-hybrid
#   - L1 latency has stayed at 4-5 cycles for a decade
#   - L3 latency varies widely: 10-40+ cycles depending on core count/topology
```

### Guide 4: Instruction Tables

```
# For EVERY x86/x86-64 instruction on EVERY modern CPU:
#   Latency:     cycles from input ready to output ready
#   Throughput:  maximum executions per cycle (reciprocal throughput)
#   Ports:       which execution port(s) the instruction uses
#   Uops:        number of micro-ops

# Critical examples (approximate, varies by generation):
# Instruction        Latency  Throughput  Uops  Ports
  add r, r/i         1        0.17-0.25   1     0/1/5/6/10
  imul r, r          3        1           1     1
  div r64            35-90    35-90       ~36   0
  vfmadd*ps ymm     4        0.5         1     0/1
  movaps xmm,[m]     4-5      0.5         1     2/3
  lock cmpxchg [m],r 15-25    15-25       ~9    varies
  cpuid              ~100     ~100        ~200  microcode

# Reading these tables:
#   If throughput < 1, the instruction can sustain >1/cycle
#   If latency > throughput, you can interleave independent operations
#   to hide latency. E.g., imul: lat=3, tput=1 → 3 independent imuls
#   can be in-flight simultaneously, achieving 1 imul/cycle.
```

---

## 6. Advanced Encoding: VEX and EVEX

### VEX Prefix (AVX/AVX2)

```
# 2-byte VEX: C5 [R vvvv L pp]
# 3-byte VEX: C4 [R X B mmmmm] [W vvvv L pp]

# Fields:
  R, X, B:   inverted REX bits (1 = no extension, 0 = extend)
  vvvv:      additional register operand (inverted, 1s complement)
  L:         0 = 128-bit (xmm), 1 = 256-bit (ymm)
  pp:        00=none, 01=66, 10=F3, 11=F2 (replaces legacy prefixes)
  mmmmm:     01=0F, 10=0F38, 11=0F3A (opcode map)

# VEX enables:
  - 3-operand non-destructive form: vaddps ymm0, ymm1, ymm2
    (result in ymm0, sources ymm1 and ymm2 unchanged)
  - Implicit zeroing of upper bits (no SSE/AVX transition penalty)
  - Shorter encoding than REX + legacy prefix + escape bytes
```

### EVEX Prefix (AVX-512)

```
# 4-byte prefix: 62 [R X B R' 0 0 m m] [W v v v v 1 p p] [z L' L b V' a a a]

# New capabilities beyond VEX:
  R', V':    extend to 32 SIMD registers (zmm0-zmm31)
  z:         merge-masking (0) vs zero-masking (1)
  aaa:       opmask register k0-k7
  L'L:       00=128, 01=256, 10=512 bit vector length
  b:         broadcast single element to all lanes,
             or embedded rounding/SAE

# Opmask example:
  vaddps zmm0 {k1}, zmm1, zmm2   # only add lanes where k1 bit is set
  vaddps zmm0 {k1}{z}, zmm1, zmm2  # zero masked-off lanes

# Embedded broadcast:
  vaddps zmm0, zmm1, [rdi]{1to16}  # load 1 float, broadcast to 16 lanes
  # Eliminates separate broadcast instruction
```

---

## 7. Performance Measurement Methodology

### Hardware Performance Counters

```
# x86-64 CPUs expose hundreds of counters via MSRs (Model-Specific Registers)
# Access via: perf stat, perf record, Intel VTune, AMD uProf

# Essential counters for assembly optimization:
  instructions           # retired instructions
  cycles                 # CPU cycles (includes stalls)
  uops_issued.any        # uops issued to back end
  uops_retired.all       # uops that completed and retired
  idq_uops_not_delivered # front-end bottleneck (decode starvation)
  branch-misses          # branch mispredictions
  L1-dcache-load-misses  # L1 data cache misses
  resource_stalls.any    # back-end full (ROB, RS, LB, SB)

# Derived metrics:
  IPC = instructions / cycles
  Front-end bound = idq_uops_not_delivered / (4 * cycles)  # for 4-wide
  Bad speculation = (uops_issued - uops_retired) / (4 * cycles)
  Back-end bound = 1 - front_end_bound - bad_speculation - retiring
```

### Microbenchmarking Pitfalls

```
# Common mistakes when benchmarking assembly sequences:

# 1. Dead code elimination
#   The compiler/CPU may skip work whose result is never used.
#   Fix: consume the result (write to volatile, return it, use asm volatile).

# 2. Measurement overhead
#   rdtsc/rdtscp have ~20-30 cycle overhead.
#   lfence + rdtsc brackets the measurement to prevent reordering.
#   For very short sequences, measure many iterations and divide.

# 3. Cold cache effects
#   First run touches cold cache lines. Warm up with a throwaway run.
#   Or measure cold-cache intentionally if that matches your workload.

# 4. Frequency scaling
#   Turbo boost and thermal throttling change frequency mid-benchmark.
#   Pin frequency: cpupower frequency-set -g performance
#   Or use cycles instead of wall time.

# 5. Branch predictor training
#   The predictor trains on your benchmark loop.
#   Real-world misprediction rate may be higher than microbenchmark.
```

---

## References

- Intel 64 and IA-32 Architectures SDM, Vol. 2 (Instruction Set Reference)
- AMD64 Architecture Programmer's Manual, Vol. 3 (General-Purpose and System Instructions)
- Agner Fog, "The Microarchitecture of Intel, AMD, and VIA CPUs" (agner.org/optimize)
- Agner Fog, "Instruction Tables" (agner.org/optimize)
- Agner Fog, "Optimizing Subroutines in Assembly Language" (agner.org/optimize)
- Intel Architectures Optimization Reference Manual
- AMD Software Optimization Guide for AMD Family 19h/1Ah Processors
- Wikichip microarchitecture pages (en.wikichip.org)
- ARM Architecture Reference Manual for A-profile (DDI 0487)
- Chips and Cheese blog (chipsandcheese.com) -- independent microarchitecture analysis
- Travis Downs, "Performance Analysis Guide" (travisdowns.github.io)
- Hadi Brais, "x86 Instruction Encoding" reference sheets
