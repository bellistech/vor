# RISC-V — Architecture, Design Philosophy, and Ecosystem Analysis

> *Deep dive into the RISC-V ISA: formal specification via the Sail model, fixed-width encoding and decode implications, modular extension philosophy, RVWMO memory model formalization, custom instruction economics, and open vs proprietary ISA ecosystem comparison.*

---

## Prerequisites

- Familiarity with assembly language concepts (registers, instructions, memory addressing)
- Understanding of processor pipelines (fetch, decode, execute, memory, writeback)
- Basic knowledge of memory consistency models (sequential consistency, TSO)
- Exposure to instruction set architecture concepts (RISC vs CISC)

## Complexity

| Topic | Analysis Type | Key Metric |
|:---|:---|:---|
| Formal ISA Specification (Sail) | Formal verification | State space coverage, conformance testing |
| Fixed-Width Decode | Hardware complexity | Gate count, critical path delay |
| Modular Extensions | Architecture design | Opcode space utilization, composability |
| RVWMO Memory Model | Formal ordering | Preserved program order rules, litmus tests |
| Custom Instruction Economics | Business analysis | NRE cost, time-to-market, royalty structure |
| Open vs Proprietary ISA | Ecosystem comparison | License cost, toolchain maturity, market share |

---

## 1. The RISC-V Formal Specification (Sail Model)

### Why Formal Specification Matters

Most instruction set architectures are specified in natural language --
English prose documents with diagrams. The x86-64 ISA is defined across
thousands of pages of Intel and AMD manuals. ARM's architecture
reference manuals run to similar lengths. Natural language specifications
suffer from three fundamental problems:

1. **Ambiguity.** English is inherently ambiguous. When the spec says
   "the result is undefined," does that mean the hardware can produce
   any value, or that the entire system state becomes unpredictable?
   Different implementors interpret this differently.

2. **Incompleteness.** Corner cases are easy to miss in prose. What
   happens when an atomic instruction targets a misaligned address
   that crosses a cache line boundary? The prose may simply not
   address this scenario.

3. **Untestability.** You cannot execute an English document. Verifying
   that a hardware implementation conforms to a prose specification
   requires human interpretation at every step.

RISC-V addresses this with a **formal specification** written in the
**Sail** language.

### The Sail Language

Sail is a domain-specific language designed for describing instruction
set architectures. It was developed at the University of Cambridge and
provides:

- **Executable semantics.** The Sail model can be compiled and run as
  an emulator. Given a RISC-V binary, the Sail model produces the
  correct execution trace, register state, and memory state.

- **Type-safe bitvector operations.** Sail has first-class support for
  fixed-width bitvectors with compile-time width checking. An operation
  that mixes 32-bit and 64-bit values without explicit conversion is a
  type error, not a subtle bug.

- **Theorem prover export.** Sail can generate proof targets for
  Isabelle/HOL, Coq, and other theorem provers. This enables formal
  verification of properties like "no unprivileged instruction can
  modify a machine-mode CSR."

- **C and OCaml generation.** The Sail model can be compiled to C
  (for performance) or OCaml (for integration with formal tools),
  producing reference emulators directly from the spec.

### What the Sail Model Covers

The RISC-V Sail model (https://github.com/riscv/sail-riscv) defines:

```
Instruction decode:
  For each instruction encoding, a pattern match on the 32-bit word
  extracts fields (opcode, funct3, funct7, rd, rs1, rs2, immediate)
  and dispatches to the execution semantics.

Execution semantics:
  For each instruction, a function that reads register/memory state,
  computes the result, and writes the new state. Example (simplified):

  function execute(ADD(rd, rs1, rs2)) =
    let result = X(rs1) + X(rs2) in
    X(rd) = result

Trap and exception handling:
  Precise definition of when traps occur (illegal instruction,
  misaligned access, page fault), what state is saved (mepc, mcause,
  mtval), and how control transfers to the trap handler.

Privilege transitions:
  Formal rules for mret/sret, CSR access permissions, and the
  interaction between privilege levels.

Memory access:
  Load/store semantics including alignment requirements, endianness,
  and interaction with the memory model.
```

### Conformance Testing with the Sail Model

The Sail model enables automated conformance testing:

1. **Generate test vectors.** A test generator produces instruction
   sequences that exercise corner cases (overflow, misaligned access,
   privilege violations).

2. **Run on Sail model.** The Sail emulator produces the reference
   output -- the correct register and memory state after execution.

3. **Run on hardware/RTL.** The same test runs on the implementation
   under test.

4. **Compare.** Any divergence between the Sail model output and the
   implementation output is a conformance failure -- either the
   implementation has a bug, or the test has found an ambiguity in
   the specification.

The RISC-V Architectural Test Suite (riscv-arch-test) uses this
methodology. Implementors submit their results; passing the test suite
is a requirement for claiming RISC-V compatibility.

### Comparison to Other ISA Specifications

| ISA | Specification Format | Executable? | Formally Verifiable? |
|:---|:---|:---|:---|
| RISC-V | Sail (formal DSL) | Yes | Yes (Isabelle/HOL, Coq) |
| x86-64 | English prose (Intel SDM, AMD APM) | No | No (some academic efforts) |
| ARM (AArch64) | ASL (Architecture Specification Language) | Partially | Limited |
| MIPS | English prose | No | No |
| POWER | English + pseudocode | No | Some formal memory model work |

ARM's ASL is the closest competitor to Sail, but ASL was developed
internally at ARM and is not fully open. The RISC-V Sail model is
open-source (BSD license), enabling any implementor to use it.

---

## 2. Why Fixed-Width Encoding Matters for Decode

### The Decode Problem

The first stage of instruction execution is **decode**: given a stream
of bytes from memory, determine where each instruction begins, what
operation it performs, and what operands it uses. The complexity of
decode directly impacts:

- **Clock frequency.** The decode stage is on the critical path. A
  slower decode stage means a lower maximum clock speed.
- **Superscalar width.** A superscalar processor decodes multiple
  instructions per cycle. Variable-width encoding makes parallel decode
  exponentially harder.
- **Power consumption.** Complex decode logic requires more transistors,
  more switching activity, and more energy.

### Variable-Width Encoding: The x86 Problem

x86-64 instructions range from 1 to 15 bytes. The decoder does not
know where an instruction ends until it has parsed the prefixes, opcode
bytes, ModR/M byte, SIB byte, displacement, and immediate fields. Each
field's presence depends on previous fields.

```
x86-64 instruction format (variable, 1-15 bytes):
[Legacy Prefixes (0-4)] [REX (0-1)] [Opcode (1-3)] [ModR/M (0-1)]
[SIB (0-1)] [Displacement (0-4)] [Immediate (0-4)]
```

Consequences:

1. **Instruction boundaries are unknown.** Given a stream of bytes
   starting at an arbitrary position, you cannot determine instruction
   boundaries without sequential parsing from a known start point.
   This is the **instruction boundary detection problem**.

2. **Superscalar decode requires pre-decode.** Modern x86 processors
   (Intel since P6, AMD since K7) use a **pre-decode** stage that
   scans the byte stream and marks instruction boundaries before the
   actual decode stage. This adds pipeline stages, latency, and
   complexity.

3. **Micro-op caches.** To avoid the decode penalty on hot code, modern
   x86 processors cache decoded micro-ops (Intel's DSB / decoded stream
   buffer, AMD's op cache). This is an entire cache hierarchy added
   solely to work around decode complexity.

4. **Branch target ambiguity.** When jumping to a computed address, the
   processor must verify it is a valid instruction boundary. This
   creates a security concern (ROP gadgets can start mid-instruction)
   and a performance concern (branch target buffer misses require
   full re-decode).

### Fixed-Width Encoding: The RISC-V Advantage

RISC-V base instructions are always 32 bits (4 bytes), aligned to
4-byte boundaries:

```
RISC-V instruction format (fixed, 32 bits):
[31                                                            0]
 Always exactly 32 bits. Always aligned. Opcode always at [6:0].
 rs1 always at [19:15]. rs2 always at [24:20]. rd always at [11:7].
```

Consequences:

1. **Trivial boundary detection.** Instructions start at every 4-byte
   boundary. No pre-decode stage needed. Given N bytes of instruction
   memory, there are exactly N/4 instructions.

2. **Parallel decode.** To decode 4 instructions per cycle (4-wide
   superscalar), fetch 16 bytes and split into four 32-bit words. Each
   word decodes independently in parallel. No dependencies between
   decode lanes.

3. **Fixed field positions.** The opcode, register fields, and function
   codes are always in the same bit positions. The decoder can extract
   all fields simultaneously with simple wire routing -- no
   multiplexing based on instruction length.

4. **Simplified branch targets.** All branch targets must be
   instruction-aligned. Mid-instruction jumps are impossible by
   construction.

### The Compressed Extension Trade-off

The C (compressed) extension adds 16-bit encodings for common
instructions. This breaks pure fixed-width encoding but preserves most
decode simplicity:

- 16-bit instructions are identified by bits [1:0] not being `11`.
- 32-bit instructions always have bits [1:0] = `11`.
- The decoder checks 2 bits to determine width, then routes to the
  appropriate 16-bit or 32-bit decode logic.

This is a one-bit multiplexer, not a multi-stage sequential parse. The
decode complexity increase is minimal compared to x86's variable-width
format.

The code size benefit is significant: the C extension reduces binary
size by 25-30%, which is critical for embedded systems with limited
flash memory and for improving instruction cache hit rates.

### Decode Complexity Comparison

| Metric | RISC-V (no C) | RISC-V (with C) | ARM64 | x86-64 |
|:---|:---|:---|:---|:---|
| Instruction widths | 1 (32-bit) | 2 (16/32-bit) | 1 (32-bit) | 1-15 bytes |
| Boundary detection | Trivial | 2-bit check | Trivial | Sequential parse |
| Pre-decode stage | No | No | No | Yes (essential) |
| Parallel decode (4-wide) | 4 independent lanes | 4 lanes + alignment | 4 independent lanes | Complex alignment + pre-mark |
| Micro-op cache needed | No | No | No | Yes (critical for perf) |
| Gate count estimate | ~5K gates | ~8K gates | ~10K gates | ~50-100K gates |

---

## 3. The Modular Extension Philosophy

### Design Principle

RISC-V's core design principle is **modularity**: define a minimal base
ISA, then add capabilities through composable extensions. This is a
deliberate rejection of the "kitchen sink" approach where every feature
is baked into the base specification.

### The Base ISA: Intentional Minimalism

RV32I contains exactly 47 instructions. By comparison:

| ISA | Base Instruction Count | Notes |
|:---|:---|:---|
| RV32I | 47 | Minimal base, everything else is optional |
| ARMv8-A (AArch64) | ~1000+ | Mandatory NEON SIMD, crypto, etc. |
| x86-64 (baseline) | ~1000+ | SSE2 mandatory since AMD64, complex legacy |
| MIPS32 | ~100 | Small but not modular |

The RV32I minimalism is intentional:

1. **Verification cost scales with instruction count.** Formally
   verifying 47 instructions is tractable. Formally verifying 1000+
   is a research challenge. The seL4 microkernel (formally verified)
   runs on RISC-V partly because the ISA's simplicity reduces the
   verification burden.

2. **Silicon area scales with instruction count.** A processor that
   implements only RV32I can be extremely small -- suitable for IoT
   sensors, smart cards, or embedded controllers where every square
   millimeter of die area matters.

3. **Teaching.** A student can learn the complete RV32I instruction
   set in a semester. This is why RISC-V has largely replaced MIPS
   in computer architecture courses.

### Extension Composability

Extensions are designed to compose without conflict:

```
RV32I   — base integer
RV32IM  — base + multiply/divide
RV32IMA — base + multiply + atomics
RV32IMAC — base + multiply + atomics + compressed
RV32GC  — general purpose (I + M + A + F + D + C)
RV64GC  — 64-bit general purpose
```

Each extension occupies a designated region of the opcode space. The
base ISA reserves substantial opcode space for future standard
extensions and for custom (X) extensions. This means adding an
extension never requires changing the encoding of existing
instructions.

### Ratified vs Draft vs Custom

Extensions go through a standardization lifecycle:

```
Proposal → Draft → Frozen → Ratified
              ↓
         (may be abandoned)
```

- **Ratified** extensions (I, M, A, F, D, C, V, etc.) are stable.
  Hardware and software can rely on them not changing.
- **Draft** extensions are under development. Implementations should
  not ship with draft extensions enabled by default.
- **Custom (X)** extensions are vendor-specific and explicitly outside
  the standard. They use reserved opcode space and are guaranteed not
  to conflict with any current or future standard extension.

### The Fragmentation Question

Critics of RISC-V's modularity argue it leads to fragmentation: if
every vendor picks a different subset of extensions, software portability
suffers. The RISC-V community addresses this through **profiles**:

- **RVA20** (Application Processor Profile, 2020): mandates RV64GC +
  specific CSRs + privilege modes for Linux-capable processors.
- **RVA22**: extends RVA20 with vector, bit manipulation, and
  additional requirements.
- **RVM23** (Microcontroller Profile): defines requirements for
  embedded/real-time processors.

Profiles serve as the interoperability contract: "if your chip
implements RVA22, all RVA22 software will run on it." This is
analogous to ARM's architecture versions (ARMv8.0, ARMv8.2, etc.)
but with the flexibility to add custom extensions on top.

---

## 4. RVWMO Memory Model Formalization

### The Problem of Memory Ordering

When multiple harts (hardware threads) access shared memory, the order
in which loads and stores become visible to other harts matters for
correctness. A **memory model** defines the rules.

Consider this two-hart scenario:

```
Initially: x = 0, y = 0

Hart 0:              Hart 1:
  store x = 1          store y = 1
  load r1 = y          load r2 = x
```

Under **sequential consistency** (the strongest model), at least one
hart must see the other's store: `r1 = 0 AND r2 = 0` is forbidden.

Under a **weak** model, both harts may see stale values: `r1 = 0 AND
r2 = 0` is allowed.

### RVWMO: RISC-V Weak Memory Ordering

RISC-V chose a **weak** memory model (RVWMO) for performance reasons.
A weak model allows hardware to reorder loads and stores freely unless
the program explicitly constrains the ordering. This enables:

- **Store buffers.** A hart can continue executing after a store
  without waiting for the store to reach cache/memory.
- **Load speculation.** A hart can satisfy a load from its store buffer
  or from a speculative cache line before prior stores are globally
  visible.
- **Out-of-order execution.** The memory system can reorder requests
  for higher throughput.

### Preserved Program Order (PPO)

RVWMO defines **preserved program order (PPO)** -- the set of ordering
rules that hardware must respect. The formal model specifies 13 rules:

```
Rule 1:  Overlapping addresses — if instruction A precedes B in
         program order, and both access an overlapping address,
         and at least one is a store, A must be ordered before B.

Rule 2:  Explicit fences — fence instructions create ordering
         between preceding and succeeding memory operations.

Rule 3:  Acquire loads — a load with .aq ensures all subsequent
         memory operations are ordered after it.

Rule 4:  Release stores — a store with .rl ensures all preceding
         memory operations are ordered before it.

Rule 5:  Acquire-release (both .aq and .rl) — full bidirectional
         ordering.

Rule 6:  Address dependency — if load A's result is used to compute
         load/store B's address, A is ordered before B.

Rule 7:  Data dependency — if load A's result is used as store B's
         data, A is ordered before B.

Rule 8:  Control dependency — if load A's result determines (via
         branch) whether store B executes, A is ordered before B.

Rule 9:  Same-address ordering for AMOs — atomic read-modify-write
         operations to the same address are totally ordered.

Rules 10-13: Additional rules for CSR accesses, I/O ordering,
             and mixed-size accesses.
```

### Formal Axioms

The RVWMO model is formalized axiomatically. Given a set of memory
events (loads, stores, fences, AMOs) and relations between them:

```
ppo    — preserved program order (from the 13 rules above)
gmo    — global memory order (a total order consistent with ppo)
rf     — reads-from (which store a load reads its value from)
co     — coherence order (total order of stores to each address)
fr     — from-reads (a load is ordered before a store if the load
         reads from a store that precedes the store in co)
```

The memory model is consistent if and only if there exists a `gmo`
(total order over all memory events) such that:

1. `gmo` is consistent with `ppo` (preserved program order is
   respected).
2. `gmo` is consistent with `co` (coherence order per address).
3. No load reads from a store that is overwritten by a later store
   in `gmo` (no stale reads after the point of global visibility).

### Litmus Tests

The formal model is validated against **litmus tests** -- small
multi-threaded programs with known allowed/forbidden outcomes:

```
Message Passing (MP):
  Hart 0: store x=1; fence w,w; store y=1
  Hart 1: load y; fence r,r; load x
  Forbidden outcome: y=1, x=0
  (The fences enforce ordering: if Hart 1 sees y=1, it must see x=1)

Store Buffering (SB):
  Hart 0: store x=1; load y
  Hart 1: store y=1; load x
  Allowed outcome: both loads return 0
  (RVWMO is weak enough to allow store buffering)

Load Buffering (LB):
  Hart 0: load x; store y=1
  Hart 1: load y; store x=1
  Allowed outcome: both loads return 1
  (Allowed under RVWMO, forbidden under TSO)
```

The RISC-V memory model team maintains a comprehensive litmus test
suite and has validated the model using the herd7 and rmem tools from
the University of Cambridge.

### RVWMO vs TSO vs ARM's Memory Model

| Property | RVWMO (RISC-V) | TSO (x86-64) | ARM Weak |
|:---|:---|:---|:---|
| Store-store reorder | Allowed | Forbidden | Allowed |
| Load-load reorder | Allowed | Forbidden | Allowed |
| Store-load reorder | Allowed | Allowed | Allowed |
| Load-store reorder | Allowed | Forbidden | Allowed |
| Store buffering | Allowed | Allowed | Allowed |
| Load buffering | Allowed | Forbidden | Allowed |
| Dependency ordering | Address + data + control | All (TSO) | Address + data (not control) |
| Fence granularity | Per-type (r/w combinations) | `mfence` (all-or-nothing) | `dmb` with options |
| Acquire/release | On AMO instructions | Not needed (TSO suffices) | On load/store instructions |

RISC-V also defines `fence.tso` as a lighter-weight fence that provides
TSO semantics. Software ported from x86 can use `fence.tso` instead of
the heavier `fence rw, rw`, preserving much of the performance benefit
of weak ordering while maintaining x86-compatible ordering guarantees.

---

## 5. Custom Instruction Economics

### The Cost of Proprietary ISA Extensions

Under the ARM licensing model, adding custom instructions to a
processor requires:

1. **Architecture license** (~$10-15M upfront) granting the right to
   design a custom core that implements the ARM ISA.
2. **Per-chip royalties** (1-2% of chip selling price) on every unit
   shipped.
3. **Compliance testing** fees to ARM to verify ISA conformance.
4. **Legal risk** that custom extensions infringe on ARM's patent
   portfolio.

Under x86, custom extensions are effectively impossible for third
parties. Intel and AMD control the ISA. x86 extensions (AVX-512, AMX)
are defined by Intel, and AMD can choose to implement them (or not).
No third party can add instructions to x86.

### The RISC-V Custom Extension Model

RISC-V eliminates ISA licensing costs entirely:

```
Upfront ISA license:        $0  (open standard, BSD license)
Per-chip royalty:            $0  (no royalty obligations)
Custom extension opcode:    Reserved in the spec, guaranteed conflict-free
Compliance testing:         Self-service (open test suite)
```

The economics shift fundamentally:

- **NRE (Non-Recurring Engineering)** for chip design remains
  significant ($5-50M for a modern SoC), but this cost exists
  regardless of ISA choice.
- **Marginal cost per chip** drops because there are no per-unit
  royalties. For high-volume products (billions of IoT sensors, for
  example), this savings is substantial.
- **Custom instruction ROI** improves because the cost of adding
  domain-specific acceleration is only the engineering cost of the
  hardware implementation, not engineering + licensing + royalties.

### Custom Extension Use Cases

| Domain | Custom Instructions | Benefit |
|:---|:---|:---|
| AI/ML Inference | Matrix multiply-accumulate, quantized ops | 10-100x over general RISC-V for inference |
| Cryptography | AES rounds, SHA3 absorb, post-quantum lattice ops | Constant-time execution, 5-50x speedup |
| Signal Processing | Fixed-point MAC, FFT butterfly, bit-reverse | Real-time audio/video processing |
| Network Processing | Packet parsing, checksum, header manipulation | Line-rate packet processing |
| Storage | CRC32, compression, dedup hash | NVMe controller acceleration |

### The Ecosystem Lock-in Consideration

Custom extensions create a trade-off: performance gains vs software
portability. RISC-V mitigates this through:

1. **Compiler intrinsics.** Custom instructions are exposed via
   `__builtin_riscv_*` intrinsics, with fallback to software
   implementations on hardware without the extension.

2. **Runtime detection.** Software can probe for extensions via the
   `misa` CSR or OS-provided feature flags and dispatch to optimized
   code paths at runtime.

3. **Standardization path.** Popular vendor extensions can be proposed
   for standardization. The bit manipulation extension (Zba/Zbb/Zbc/Zbs)
   started as vendor-specific and was later ratified as a standard
   extension.

---

## 6. Open vs Proprietary ISA Ecosystems

### The ISA as Infrastructure

An instruction set architecture is infrastructure -- comparable to a
road network or electrical grid. Software is built on top of it.
The question is whether this infrastructure should be controlled by
a single company or governed as an open standard.

### The Proprietary Model (ARM, x86)

**ARM Ltd** licenses the ARM ISA to chip companies. The licensing
structure:

```
Cortex License (~$1-5M):
  - Use a pre-designed ARM core (Cortex-A76, Cortex-M4, etc.)
  - No modification to the core
  - Per-chip royalty: 1-2% of chip price

Architecture License (~$10-15M):
  - Design your own core implementing the ARM ISA
  - Full freedom in microarchitecture
  - Per-chip royalty: 1-2% of chip price
  - Licensees: Apple, Qualcomm, Samsung, Nvidia
```

**Intel/AMD** do not license x86 to third parties. The ISA is
inseparable from the silicon. Third-party x86 attempts (Transmeta,
VIA) have largely failed due to patent litigation and the complexity
of x86 compatibility.

### The Open Model (RISC-V)

RISC-V International (a Swiss non-profit) governs the ISA:

```
Membership:
  - Community member: free (read specs, participate in discussions)
  - Technical member: $0-35K/year (propose and vote on extensions)
  - Strategic member: $50-250K/year (board seat, governance influence)

ISA License:
  - BSD-style: irrevocable, royalty-free, perpetual
  - No patent grant from RISC-V International itself, but members
    commit to FRAND (fair, reasonable, and non-discriminatory) licensing
    of essential patents
```

### Ecosystem Comparison

| Factor | RISC-V (Open) | ARM (Proprietary) | x86 (Proprietary) |
|:---|:---|:---|:---|
| **ISA cost** | Free | $1-15M license + royalty | N/A (tied to silicon) |
| **Custom extensions** | Encouraged (reserved opcodes) | Architecture license only | Impossible for 3rd party |
| **Toolchain** | GCC + LLVM/Clang (open) | GCC + LLVM + armcc (mixed) | GCC + LLVM + MSVC (mixed) |
| **Linux support** | Mainline since 5.x | Mainline, mature | Mainline, mature |
| **Software ecosystem** | Growing rapidly | Very mature (mobile, embedded) | Dominant (desktop, server, cloud) |
| **Hardware ecosystem** | Early but accelerating | Dominant (mobile, embedded) | Dominant (desktop, server) |
| **Formal spec** | Yes (Sail, open) | Partial (ASL, semi-open) | No |
| **Geopolitical risk** | None (Swiss non-profit) | UK company, US export controls apply | US companies, export controls |
| **Verification** | Open test suite | ARM compliance suite (licensed) | Intel/AMD internal |
| **IP fragmentation risk** | Mitigated by profiles | None (ARM controls) | None (Intel/AMD control) |

### The Geopolitical Dimension

RISC-V's openness has a geopolitical dimension that accelerates
adoption. Countries and regions seeking semiconductor sovereignty
cannot build on ISAs controlled by foreign companies subject to
export controls. RISC-V, governed by a Swiss non-profit with an
irrevocable open license, provides a foundation that cannot be
withdrawn for political reasons.

This has driven significant RISC-V investment in China (Alibaba T-Head,
Andes Technology), India (IIT Madras Shakti), and the EU (European
Processor Initiative), in addition to adoption by US companies
(SiFive, Qualcomm, Google, Western Digital).

### The Software Gap

The primary barrier to RISC-V adoption in application processors is
software ecosystem maturity:

- **Operating systems:** Linux runs well on RISC-V. Android has
  experimental support. Windows does not support RISC-V.
- **Compilers:** GCC and LLVM have robust RISC-V backends. Performance
  optimization is ongoing -- RISC-V codegen is improving but not yet
  at parity with ARM64 or x86-64 in all workloads.
- **Libraries:** Core libraries (glibc, OpenSSL, zlib) support RISC-V.
  Hand-optimized SIMD/vector routines lag behind ARM NEON and x86 AVX.
- **Applications:** Most server and desktop applications are available
  via Linux distro packages (Debian, Fedora, Ubuntu have RISC-V ports).
  Proprietary software support is limited.

The software gap is closing rapidly as the hardware ecosystem grows,
following the same trajectory ARM took from embedded-only to
phones (2007+) to servers (2018+) to desktops (Apple M1, 2020+).

---

## See Also

- how-computers-work
- binary-and-number-systems
- linux-kernel-internals

## References

- Waterman, A. & Asanovic, K. "The RISC-V Instruction Set Manual" (RISC-V International)
- Patterson, D. & Hennessy, J. "Computer Organization and Design: RISC-V Edition" (Morgan Kaufmann, 2020)
- RISC-V Sail Model: https://github.com/riscv/sail-riscv
- Sail Language: https://github.com/rems-project/sail
- RISC-V Memory Consistency Model: https://riscv.org/wp-content/uploads/2018/05/14.25-15.00-Daniel-Lustig-RISC-V-Memory-Consistency-Model.pdf
- Lustig, D. et al. "RVWMO: The RISC-V Memory Consistency Model" (Chapter 17, RISC-V Unprivileged Spec)
- RISC-V Architectural Tests: https://github.com/riscv-non-isa/riscv-arch-test
- Armstrong, A. et al. "ISA Semantics for ARMv8-A, RISC-V, and CHERI-MIPS" (POPL 2019)
- RISC-V Profiles: https://github.com/riscv/riscv-profiles
- RISC-V International: https://riscv.org/
- SiFive: https://www.sifive.com/
- herd7 Memory Model Tool: https://github.com/herd/herdtools7
