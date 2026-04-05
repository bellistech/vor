# ARM64 Deep Dive -- From Encoding to Microarchitecture

> *ARM's triumph is not that it competes with x86 on performance -- it's that it delivers competitive performance at a fraction of the energy. The fixed-width encoding that limits expressiveness is the same property that enables efficient decode, wide issue, and the power savings that conquered mobile and now threaten the data center.*

---

## 1. Fixed-Width Instruction Encoding (32-bit)

### The Encoding Constraint

Every ARM64 (AArch64) instruction is exactly 32 bits (4 bytes). This is a fundamental architectural decision with far-reaching consequences.

A 32-bit encoding must pack the opcode, operands, immediates, and condition codes into just 32 bits. This creates a constant tension: you cannot have both a large immediate field and many register specifier bits. ARM64 resolves this with multiple encoding formats, each optimized for a different class of instruction.

### Major Encoding Groups

The top 4 bits (bits [31:28]) and bits [27:25] form the primary classification:

| Bits [28:25] | Encoding Group | Examples |
|:---:|:---|:---|
| 100x | Data processing (immediate) | ADD, SUB, MOV, AND with immediate |
| x101 | Data processing (register) | ADD, SUB, AND, ORR, shift/extend |
| x1x0 | Loads and stores | LDR, STR, LDP, STP, atomics |
| 101x | Branches | B, BL, CBZ, TBZ, RET |

### Immediate Encoding Tricks

The 32-bit constraint means ARM64 cannot encode arbitrary 64-bit immediates. Several clever schemes address this:

**Shifted 16-bit immediates:** The `MOVZ`, `MOVK`, `MOVN` instructions carry a 16-bit immediate that can be placed at any of four 16-bit positions in the 64-bit result:

```
movz  x0, #0xDEAD, lsl #48    // x0 = 0xDEAD_0000_0000_0000
movk  x0, #0xBEEF, lsl #32    // x0 = 0xDEAD_BEEF_0000_0000
movk  x0, #0xCAFE, lsl #16    // x0 = 0xDEAD_BEEF_CAFE_0000
movk  x0, #0xBABE             // x0 = 0xDEAD_BEEF_CAFE_BABE
```

Loading an arbitrary 64-bit constant takes up to 4 instructions. In practice, most constants fit in fewer steps, and the assembler/linker selects the optimal sequence.

**Logical immediates:** AND, ORR, EOR instructions use a 13-bit encoding (N:immr:imms) that can represent any bitmask consisting of a repeating pattern of consecutive set bits. This covers most masks used in practice (alignment masks, field extraction) with a single instruction. The encoding is bijective -- each valid 13-bit value maps to exactly one bitmask.

The set of representable logical immediates includes:

$$\text{Patterns of } 2^k \text{ bits (for } k = 1..6\text{) with } 1..2^k-1 \text{ consecutive set bits, rotated}$$

This yields 5,334 distinct 64-bit bitmask values (and 2,667 for 32-bit).

**PC-relative addressing:** `ADRP` encodes a 21-bit signed page offset (bits [32:12] of the target address relative to PC), giving +/-4 GB reach. Combined with a 12-bit `ADD` for the page offset, any address within 4 GB of the current PC is reachable in two instructions.

### Comparison With x86-64 Variable-Length Encoding

x86-64 instructions range from 1 byte to 15 bytes. This flexibility enables excellent code density but creates profound architectural challenges:

| Property | ARM64 (Fixed 32-bit) | x86-64 (Variable 1-15 bytes) |
|:---|:---|:---|
| Instruction boundary detection | Trivial (every 4 bytes) | Requires length decoding from byte 1 |
| Parallel decode | Easy: fetch 32 bytes = 8 instructions | Hard: must decode serially to find boundaries |
| Decode width | 8-wide decode is straightforward | Pre-decode needed to mark boundaries |
| Decode power | Low (simple pattern match) | High (complex state machine) |
| Branch target validation | PC must be 4-byte aligned | Any byte could be a branch target |
| Code density | ~25-30% larger binaries | Compact (variable length helps) |
| Self-modifying code | Simpler (instruction-aligned writes) | Complex (partial instruction overwrites possible) |

**x86-64 decode complexity in practice:** Modern x86 CPUs (Zen 4, Golden Cove) dedicate ~15-20% of frontend die area and power to instruction decode. They use a micro-op cache (uop cache or DSU) to bypass the complex decode path for hot code. When running from the uop cache, the x86 frontend resembles a RISC machine internally.

**ARM64's decode advantage compounds:** Because decode is cheap, ARM64 cores can afford wider decode (Apple M-series decodes 8 instructions/cycle) without the power penalty that x86 pays. This is a key factor in ARM's performance-per-watt advantage.

---

## 2. Apple M-Series Microarchitecture

### Firestorm / Avalanche / Everest (Performance Cores)

Apple's performance cores are among the widest and deepest out-of-order engines ever built in a commercial processor:

| Property | Apple M1 (Firestorm) | Apple M3 (Everest) | x86 Golden Cove (Alder Lake) |
|:---|:---|:---|:---|
| Decode width | 8-wide | 9-wide | 6-wide (with uop cache: 6 uops) |
| Reorder buffer | ~630 entries | ~700+ entries | 512 entries |
| Physical registers | ~380 int, ~434 FP | estimated larger | ~280 int, ~332 FP |
| Integer ALUs | 6 | 6 | 5 |
| FP/NEON units | 4 (128-bit) | 4 (128-bit) | 2 (256-bit AVX, fused to 512-bit) |
| Load units | 3 | 3 | 2 (+ 1 store-forwarding) |
| Store units | 2 | 2 | 2 |
| L1I cache | 192 KB | 192 KB | 32 KB (+ 4K uop cache) |
| L1D cache | 128 KB | 128 KB | 48 KB |
| L2 cache | 12 MB (shared, 4 P-cores) | 16 MB (shared) | 1.25 MB per core |
| Branch predictor | TAGE variant, very large | Improved | TAGE variant |

**Key innovations:**

1. **Massive L1 caches:** 192 KB L1I and 128 KB L1D are 3-6x larger than typical x86 L1 caches. This is possible because ARM64's fixed-width encoding makes cache line utilization predictable and reduces conflict misses. The large L1I compensates for ARM64's lower code density.

2. **Width over frequency:** Apple's P-cores run at 3.2-3.8 GHz (M1-M3), well below Intel's 5+ GHz. Instead, Apple invests transistors in width (8-9 wide decode, 6 ALUs, 3 load ports). The result: higher IPC at lower frequency, which is more power-efficient because power scales with $V^2 \times f$ and voltage can be lower at lower frequencies.

3. **Unified memory architecture (UMA):** CPU and GPU share the same physical memory pool via a common interconnect. No PCIe bottleneck for CPU-GPU data sharing. Bandwidth: ~200 GB/s (M1) to ~400 GB/s (M3 Max) via wide LPDDR5/LPDDR5X bus.

4. **Extremely deep out-of-order window:** The ~630-700 entry ROB allows the CPU to look far ahead for independent instructions, hiding memory latency more effectively. This partially compensates for ARM64's relaxed memory model requiring more barriers.

### Efficiency Cores (Icestorm / Sawtooth / Blizzard)

Apple's E-cores are themselves competitive with many competitors' performance cores:

- 4-wide decode (Icestorm) to 5-wide (Blizzard)
- ~100-entry reorder buffer
- Significantly smaller caches (64 KB L1I, 64 KB L1D)
- Run at ~2.0-2.4 GHz
- Share the same ISA extensions as P-cores (full NEON, Pointer Authentication, etc.)

The scheduler migrates threads between P-cores and E-cores based on Quality of Service (QoS) class. Background tasks (indexing, updates) run on E-cores; interactive foreground work runs on P-cores. The asymmetry is transparent to applications.

### Apple's SoC Integration

The M-series is not just a CPU -- it is a complete system-on-chip:

- **Neural Engine (ANE):** 16-core matrix accelerator, ~15 TOPS (M1) to ~18 TOPS (M3)
- **GPU:** Custom architecture, 8-10 cores (M3), unified memory with CPU
- **Media engines:** Hardware H.264/H.265/ProRes encode/decode
- **Secure Enclave:** Isolated security processor for biometrics, keys
- **Fabric interconnect:** Low-latency coherent interconnect between all IP blocks

This level of integration eliminates the latency and power overhead of discrete components communicating over PCIe, contributing to Apple's performance-per-watt leadership.

---

## 3. AWS Graviton Performance Analysis

### Architecture Evolution

| Property | Graviton 1 (2018) | Graviton 2 (2019) | Graviton 3 (2021) | Graviton 4 (2023) |
|:---|:---|:---|:---|:---|
| Core design | Cortex-A72 | Neoverse N1 | Neoverse V1 | Neoverse V2 |
| Core count | 16 | 64 | 64 | 96 |
| Architecture | ARMv8.0 | ARMv8.2 | ARMv8.4+ SVE | ARMv9.0+ SVE2 |
| Process node | 16nm | 7nm | 5nm | 5nm |
| SIMD | NEON 128-bit | NEON 128-bit | SVE 256-bit | SVE2 256-bit |
| Memory | DDR4 | DDR4, 8ch | DDR5, 8ch | DDR5, 12ch |
| Interconnect | --- | CMN-600 mesh | CMN-700 mesh | CMN-700S mesh |

### Performance Characteristics

Graviton's design philosophy differs fundamentally from x86 server chips. Instead of maximizing single-thread performance, Graviton optimizes for throughput-per-watt across many threads:

**Single-thread performance:** Graviton 3 Neoverse V1 cores achieve roughly 85-90% of the single-thread IPC of contemporary x86 server cores (Intel Sapphire Rapids, AMD Genoa). The gap narrows further with Graviton 4 (Neoverse V2).

**Throughput:** With 64-96 cores at moderate frequencies (~2.6-2.8 GHz), Graviton achieves higher aggregate throughput per watt than equivalent x86 instances for parallelizable workloads.

**Energy efficiency:** AWS reports ~60% better energy efficiency for Graviton 3 vs comparable x86 instances. This translates directly to cost: Graviton instances are typically 20-40% cheaper than equivalent x86 instances on AWS.

**Workload affinity:**

| Workload Type | Graviton Advantage | Notes |
|:---|:---|:---|
| Web serving (Nginx, Node.js) | Strong (30-40% better price/perf) | Highly parallel, fits throughput model |
| Databases (MySQL, PostgreSQL) | Good (20-30%) | Benefits from high core count, memory bandwidth |
| Containerized microservices | Strong (30-40%) | Many small processes, efficient scheduling |
| Java / JVM workloads | Good (20-25%) | JIT compiles to ARM64, GC benefits from more cores |
| HPC / scientific (vectorized) | Moderate (10-20%) | SVE helps, but x86 AVX-512 is wider (512 vs 256 bit) |
| Legacy x86-only software | Not applicable | Must recompile or emulate |

### Memory Subsystem

Graviton 3's Neoverse V1 core has a sophisticated memory hierarchy:

- L1I: 64 KB, 4-way, 1-cycle hit
- L1D: 64 KB, 4-way, 4-cycle hit
- L2: 1 MB per core, 8-way, ~10-cycle hit
- L3/SLC (System Level Cache): ~32 MB shared, ~30-40 cycle hit
- CMN-700 mesh interconnect: supports AMBA CHI protocol for cache coherence across 64 cores

The mesh interconnect is critical. Unlike a ring bus (Intel's historical choice), the mesh provides $O(\sqrt{N})$ worst-case latency for N cores, scaling better to high core counts.

---

## 4. ARM Memory Model Formalization

### Observer-Based Model

ARM's formal memory model is defined in terms of **observers** (cores, devices) and the **order** in which they observe memory accesses. Unlike x86's TSO (Total Store Order), ARM's model is explicitly relaxed:

**Key principle:** Two memory accesses from the same observer are ordered only if there is an explicit dependency or barrier between them. The hardware is free to reorder independent accesses for performance.

### Formal Ordering Rules

The ARM architecture defines ordering in terms of the "observed by" relation. For two accesses A and B from observer P:

1. **Address dependency:** If B's address depends on the value loaded by A, then A is observed before B.
2. **Data dependency:** If B's data (for a store) depends on the value loaded by A, then A is observed before B.
3. **Control dependency + ISB:** If B is after a conditional branch that depends on A, and an ISB separates them, A is observed before B.
4. **Acquire/Release:** LDAR ensures all subsequent accesses are observed after it. STLR ensures all prior accesses are observed before it.
5. **DMB:** Explicit fence that orders accesses of the specified type on either side.

**What is NOT ordered (without barriers):**
- Independent loads to different addresses
- Independent stores to different addresses
- A store followed by a load to a different address
- Loads and stores separated only by control dependencies (without ISB)

### The Litmus Test Approach

ARM's model is validated using "litmus tests" -- small multi-threaded programs that probe specific reorderings:

**Message passing (MP) test:**

```
# Initial: x = 0, y = 0

# Thread 0:           Thread 1:
  str  w1, [x]         ldr  w0, [y]     // sees 1?
  str  w1, [y]         ldr  w1, [x]     // sees 0?

# On ARM: outcome (w0=1, w1=0) IS allowed
# The store to x can be reordered after the store to y
# AND the load of x can be reordered before the load of y

# Fix with barriers:
  str  w1, [x]         ldr  w0, [y]
  dmb  ish             dmb  ish
  str  w1, [y]         ldr  w1, [x]

# Or with acquire/release:
  str   w1, [x]        ldar  w0, [y]    // acquire
  stlr  w1, [y]        ldr   w1, [x]
```

**Store buffering (SB) test:**

```
# Initial: x = 0, y = 0

# Thread 0:           Thread 1:
  str  w1, [x]         str  w1, [y]
  ldr  w0, [y]         ldr  w1, [x]

# On ARM: outcome (w0=0, w1=0) IS allowed (both read stale values)
# On x86 TSO: this outcome is ALSO allowed (the one reordering TSO permits)
```

### Multi-Copy Atomicity

ARMv8 guarantees **multi-copy atomicity** for all stores: once a store is visible to any observer other than the one that issued it, the store is visible to ALL observers. This rules out a class of subtle bugs possible on older POWER architectures where stores could be partially propagated.

This property simplifies reasoning about ARM's memory model compared to POWER, while still permitting the reorderings that enable high-performance implementations.

### Implications for Lock-Free Programming

The relaxed model means that correct lock-free data structures on ARM require explicit ordering annotations:

- **C/C++ `memory_order_seq_cst`:** Maps to LDAR/STLR with additional DMB on ARM. Most expensive but easiest to reason about.
- **C/C++ `memory_order_acquire` / `memory_order_release`:** Maps directly to LDAR/STLR. The preferred idiom for ARM -- natural fit.
- **C/C++ `memory_order_relaxed`:** Maps to plain LDR/STR. No ordering guarantee, maximum performance.

The general advice: use `acquire`/`release` semantics rather than `seq_cst` on ARM. The `seq_cst` ordering is unnecessarily expensive for most patterns, and `acquire`/`release` maps directly to hardware primitives without extra barriers.

---

## 5. Energy Efficiency Analysis

### Why ARM Is More Power-Efficient

ARM's energy advantage is not a single trick but a compound effect of several architectural decisions:

**1. Simpler decode:**

$$P_{decode} \propto \text{transistor count} \times \text{switching activity} \times V^2$$

ARM64's fixed-width decode uses roughly 5-10x fewer transistors than x86's variable-length decoder. Since decode runs on every instruction, this saving is amortized over every cycle. On a 6-wide decode engine, the power savings compound.

**2. Lower voltage at equivalent throughput:**

ARM cores achieve competitive IPC at 2.5-3.5 GHz, while x86 cores push 4.5-5.5 GHz for maximum single-thread performance. Since dynamic power scales with $V^2 \times f$, and voltage must increase with frequency (to maintain switching margins):

$$\frac{P_{ARM}}{P_{x86}} \approx \frac{V_{ARM}^2 \times f_{ARM}}{V_{x86}^2 \times f_{x86}}$$

At typical operating points (ARM: 0.7V @ 3 GHz, x86: 1.1V @ 5 GHz):

$$\frac{P_{ARM}}{P_{x86}} \approx \frac{0.49 \times 3}{1.21 \times 5} = \frac{1.47}{6.05} \approx 0.24$$

ARM's dynamic power is roughly 1/4 of x86 at these operating points, per core, for the computational logic. The actual ratio is less extreme because static power, memory controllers, and I/O are similar, but the trend holds.

**3. No legacy tax:**

x86 carries decades of backward compatibility: 16-bit real mode, segmentation, variable-length encoding, x87 FPU, MMX, SSE, SSE2/3/4, AVX, AVX-512. All of this decode and microcode logic consumes die area and power even when unused. ARM64 (AArch64) was a clean-sheet design in 2011 with no obligation to legacy 32-bit ARM decode paths (though many implementations include them optionally).

**4. Load/store architecture reduces complexity:**

Because only dedicated load/store instructions access memory, the execution units are simpler -- ALUs never need to interact with the memory system. This simplifies scheduling, reduces port counts on register files, and allows more aggressive clock gating of unused units.

### Perf-Per-Watt Comparison (Server Workloads)

Based on published benchmarks and AWS pricing data (as of early 2025):

| Metric | Graviton 3 (c7g) | Intel Sapphire Rapids (c7i) | AMD Genoa (c7a) |
|:---|:---|:---|:---|
| SPECint rate (est.) | ~310 | ~350 | ~370 |
| TDP per instance | ~70W | ~120W | ~110W |
| SPECint / Watt | ~4.4 | ~2.9 | ~3.4 |
| AWS $/hour (xlarge) | ~$0.145 | ~$0.178 | ~$0.163 |
| Perf/$ (normalized) | 1.00 | 0.72 | 0.84 |

Graviton's advantage is most pronounced in throughput-per-dollar, which is the metric cloud customers optimize for. The absolute single-thread performance gap continues to narrow with each generation.

### Thermal Design Implications

ARM's lower power density enables:

- **Passive cooling in laptops:** Apple's MacBook Air (M1/M2) is fanless -- impossible with x86 at comparable performance
- **Higher density in data centers:** More compute per rack unit, lower cooling costs
- **Sustained performance:** Less thermal throttling under sustained load, more predictable latency

---

## 6. Advanced Topics

### Pointer Authentication Code (PAC) -- Cryptographic Details

PAC uses the QARMA block cipher (or implementation-defined alternative) with:
- **Input:** pointer value, 64-bit context modifier (e.g., SP), key
- **Output:** a PAC value stored in the upper unused bits of the pointer

For a 48-bit virtual address with 16-bit top-byte-ignore (TBI), there are up to 15 bits available for the PAC. This provides $2^{15} = 32768$ possible codes, meaning an attacker has a ~1/32768 chance of guessing a valid PAC for a corrupted pointer per attempt.

The five keys (APIA, APIB, APDA, APDB, APGA) allow different protection domains -- instruction pointers use different keys than data pointers, and the generic key (APGA) can sign arbitrary data.

### Memory Tagging Extension (MTE) -- Detection Probability

With 4-bit tags (16 possible values), a random mismatch has:

$$P(\text{detection}) = 1 - \frac{1}{16} = 93.75\%$$

For use-after-free, the tag is re-randomized on each allocation. For buffer overflow, adjacent allocations get different tags. The 6.25% miss rate is acceptable because:
1. Attackers cannot choose tags (they are randomized)
2. Repeated exploitation attempts hit different tags each time
3. In practice, real-world UAF/overflow exploits are detected with high probability over multiple accesses

MTE's hardware overhead is modest: tag storage requires 3.125% additional memory (4 bits per 16 bytes), and tag checks are pipelined with the load/store units at zero additional latency for the common (matching) case.

---

## References

- ARM Architecture Reference Manual for A-profile (DDI 0487, ~12,000 pages) -- the definitive ISA specification
- Pulte et al., "Simplifying ARM Concurrency: Multicopy-atomic Axiomatic and Operational Models for ARMv8" (POPL 2018) -- formalization of the ARM memory model
- Flur et al., "Modelling the ARMv8 Architecture, Operationally: Concurrency and ISA" (POPL 2016)
- ARM Neoverse V1 Technical Reference Manual (for Graviton 3 analysis)
- Apple M1 microarchitecture analysis by Dougall Johnson (dougallj.github.io)
- Anandtech: "The 2020 Mac Mini Unleashed" (M1 deep dive)
- AWS Graviton performance benchmarks: github.com/aws/aws-graviton-getting-started
- Qualcomm PAC whitepaper: "Pointer Authentication on ARMv8.3" (2017)
- ARM MTE whitepaper: "Memory Tagging Extension" (2019)
- Hennessy & Patterson, "A New Golden Age for Computer Architecture" (ACM Turing Lecture, 2019)
