# The Theory of Computer Architecture -- From ISA Design to Power Walls

> *Computer architecture is the science of trade-offs: speed vs power, complexity vs correctness, parallelism vs programmability. Every design decision echoes through the entire stack.*

---

## 1. Instruction Set Architecture (ISA) Design Trade-offs

### CISC vs RISC

The oldest debate in architecture. Complex Instruction Set Computing (x86) vs Reduced Instruction Set Computing (ARM, RISC-V, MIPS).

**CISC philosophy:** Provide powerful, multi-step instructions so the compiler has less work. A single instruction might load from memory, add, and store back.

**RISC philosophy:** Keep instructions simple, uniform, and fast. Do one thing per instruction. Let the compiler optimize.

| Property | CISC (x86) | RISC (ARM/RISC-V) |
|:---|:---|:---|
| Instruction length | Variable (1-15 bytes on x86) | Fixed (4 bytes typically) |
| Instructions per task | Fewer (complex instructions) | More (simple instructions) |
| Decode complexity | High (variable length, many formats) | Low (fixed format, easy to decode) |
| Register count | Fewer architectural (16 on x86-64) | More (31 on AArch64, 32 on RISC-V) |
| Memory access | Any instruction can access memory | Load/store architecture only |
| Pipeline friendliness | Harder (variable decode, micro-ops) | Easier (uniform stages) |
| Code density | Higher (variable length helps) | Lower (fixed 4-byte instructions) |
| Power efficiency | Lower (decode hardware is large) | Higher (simpler logic) |

**Modern reality:** The distinction has blurred. x86 CPUs internally translate CISC instructions into RISC-like micro-ops (uops). Apple's M-series (ARM) achieves desktop-class performance. The ISA matters less than the microarchitecture beneath it.

### RISC-V: The Open ISA

RISC-V is notable for being open-source and modular:

- **Base integer ISA (RV32I/RV64I):** 47 instructions, minimal
- **Standard extensions:** M (multiply/divide), A (atomic), F/D (floating point), C (compressed 16-bit instructions), V (vector)
- **Custom extensions:** Anyone can add domain-specific instructions without licensing

This modularity allows chip designers to build exactly the ISA they need -- from tiny microcontrollers (RV32E, 16 registers) to supercomputer cores (RV64GCV).

### Encoding Trade-offs

Fixed-length encoding (RISC) wastes bits on simple instructions but makes fetch/decode trivial -- the CPU always knows where the next instruction starts. Variable-length encoding (CISC) achieves better code density but requires complex decode logic that consumes power and die area.

ARM's Thumb-2 and RISC-V's C extension are compromises: a mix of 16-bit and 32-bit instructions for density without full CISC complexity.

---

## 2. Amdahl's Law

### The Formula

$$S = \frac{1}{(1 - P) + \frac{P}{N}}$$

Where:
- $S$ = overall speedup of the system
- $P$ = fraction of the program that can be parallelized (0 to 1)
- $N$ = number of processors/cores

### Worked Examples

**Example 1:** A program is 90% parallelizable, run on 8 cores:

$$S = \frac{1}{(1 - 0.9) + \frac{0.9}{8}} = \frac{1}{0.1 + 0.1125} = \frac{1}{0.2125} = 4.71\times$$

Even with 8 cores, we get only 4.71x speedup, not 8x.

**Example 2:** Same program, infinite cores ($N \to \infty$):

$$S = \frac{1}{(1 - 0.9) + 0} = \frac{1}{0.1} = 10\times$$

The serial 10% limits us to at most 10x speedup, no matter how many cores we add.

### The Implications Table

| Parallel fraction $P$ | 2 cores | 4 cores | 8 cores | 64 cores | $\infty$ cores |
|:---:|:---:|:---:|:---:|:---:|:---:|
| 50% | 1.33x | 1.60x | 1.78x | 1.97x | 2.00x |
| 75% | 1.60x | 2.29x | 2.91x | 3.67x | 4.00x |
| 90% | 1.82x | 3.08x | 4.71x | 8.77x | 10.00x |
| 95% | 1.90x | 3.48x | 5.93x | 14.29x | 20.00x |
| 99% | 1.98x | 3.88x | 7.48x | 39.26x | 100.00x |

### Gustafson's Law (The Counterargument)

Amdahl assumed fixed problem size. Gustafson observed that in practice, when you get more cores, you solve bigger problems:

$$S_G = N - (1 - P)(N - 1) = 1 - P + P \cdot N$$

This is linear in $N$ for a given $P$. The insight: with 64 cores, you don't run the same simulation -- you run a 64x larger simulation. The serial fraction shrinks as a proportion of the larger problem.

---

## 3. Memory Hierarchy Analysis

### The Fundamental Problem

CPU speed has grown roughly 60% per year (historically), while DRAM latency improves only ~7% per year. This "memory wall" means a modern CPU waiting on main memory wastes hundreds of potential operations.

### Locality of Reference

All cache designs exploit two types of locality:

- **Temporal locality:** If you accessed address X, you'll likely access X again soon (loops, hot variables)
- **Spatial locality:** If you accessed address X, you'll likely access X+1, X+2 soon (arrays, sequential code)

### Cache Organization

A cache with $C$ bytes total, block size $B$, and associativity $A$:

$$\text{Number of sets} = S = \frac{C}{B \times A}$$

Address decomposition: `[Tag | Set Index | Block Offset]`

- Block offset bits: $\log_2(B)$
- Set index bits: $\log_2(S)$
- Tag bits: address width - offset bits - index bits

**Example:** 32 KB L1 cache, 64-byte lines, 8-way associative, 48-bit addresses:

$$S = \frac{32768}{64 \times 8} = 64 \text{ sets}$$

- Offset: $\log_2(64) = 6$ bits
- Index: $\log_2(64) = 6$ bits
- Tag: $48 - 6 - 6 = 36$ bits

### Cache Miss Types (The Three C's)

1. **Compulsory (cold) misses:** First access to a block. Unavoidable. Reduced by prefetching.
2. **Capacity misses:** Cache too small to hold working set. Reduced by increasing cache size.
3. **Conflict misses:** Multiple addresses map to the same set. Reduced by increasing associativity (at the cost of lookup latency and power).

A fourth "C" sometimes added:

4. **Coherence misses:** Caused by invalidation from another core's write (multiprocessor systems).

### Average Memory Access Time (AMAT)

$$\text{AMAT} = T_{hit} + MR \times T_{miss}$$

Where:
- $T_{hit}$ = time for a cache hit
- $MR$ = miss rate (fraction of accesses that miss)
- $T_{miss}$ = miss penalty (time to fetch from next level)

For a multi-level hierarchy:

$$\text{AMAT} = T_{L1} + MR_{L1} \times (T_{L2} + MR_{L2} \times (T_{L3} + MR_{L3} \times T_{mem}))$$

**Example:** L1 hit time 1ns, L1 miss rate 5%, L2 hit time 4ns, L2 miss rate 20%, memory time 100ns:

$$\text{AMAT} = 1 + 0.05 \times (4 + 0.20 \times 100) = 1 + 0.05 \times 24 = 1 + 1.2 = 2.2 \text{ ns}$$

Without any cache, AMAT would be 100 ns. Caches reduced effective latency by ~45x.

---

## 4. Pipelining Theory

### Ideal Pipeline Speedup

For a $k$-stage pipeline executing $n$ instructions:

$$T_{pipelined} = k + (n - 1) \text{ cycles}$$

$$\text{Speedup} = \frac{n \times k}{k + (n - 1)}$$

As $n \to \infty$: speedup approaches $k$ (the number of stages).

### Pipeline Depth Trade-offs

Deeper pipelines allow higher clock frequencies (each stage does less work), but:

- **Branch misprediction penalty grows linearly with depth** (flush entire pipeline)
- **Power consumption increases** (more pipeline registers, more clock distribution)
- **Diminishing returns** on frequency scaling (wire delays, clock skew)

Historical examples:
- Intel Pentium 4 (Prescott): 31-stage pipeline, hit the "power wall" at 3.8 GHz
- ARM Cortex-A76: 13-stage pipeline, much better perf/watt
- Apple M1 Firestorm: ~15 stages, extremely wide (8-wide decode), focuses on IPC over frequency

### Hazard Cost Analysis

The effective CPI (Cycles Per Instruction) for a pipelined processor:

$$\text{CPI} = \text{CPI}_{ideal} + \text{stalls}_{data} + \text{stalls}_{control} + \text{stalls}_{structural}$$

Where $\text{CPI}_{ideal} = 1$ for a scalar pipeline. In practice:

$$\text{CPI} = 1 + f_{branch} \times p_{miss} \times c_{penalty} + f_{load} \times p_{cache\_miss} \times c_{miss}$$

- $f_{branch}$: fraction of instructions that are branches (~15-20%)
- $p_{miss}$: branch misprediction rate (~3-5% for modern predictors)
- $c_{penalty}$: misprediction penalty in cycles (= pipeline depth for full flush)
- $f_{load}$: fraction of instructions that are loads (~25%)
- $p_{cache\_miss}$: cache miss rate
- $c_{miss}$: cache miss penalty in cycles

**Example:** 15-stage pipeline, 20% branches, 3% misprediction, 25% loads, 2% L1 miss, 10-cycle L2 penalty:

$$\text{CPI} = 1 + (0.20 \times 0.03 \times 15) + (0.25 \times 0.02 \times 10)$$
$$\text{CPI} = 1 + 0.09 + 0.05 = 1.14$$

---

## 5. Power and Performance Trade-offs

### The Power Equation

Dynamic power consumption of a CMOS circuit:

$$P_{dynamic} = \alpha \times C \times V^2 \times f$$

Where:
- $\alpha$ = activity factor (fraction of transistors switching per cycle, typically 0.1-0.3)
- $C$ = total capacitance (proportional to transistor count)
- $V$ = supply voltage
- $f$ = clock frequency

**Critical insight:** Power scales with $V^2$. Reducing voltage from 1.0V to 0.7V cuts dynamic power by 51%.

### Static (Leakage) Power

$$P_{static} = V \times I_{leak}$$

Leakage current grows exponentially as transistors shrink. Below ~22nm, static power rivals dynamic power. This is one reason why clock frequencies stopped scaling around 2004-2006.

### The Power Wall

Combining both:

$$P_{total} = \alpha C V^2 f + V I_{leak}$$

To increase frequency $f$, you must increase voltage $V$ (to maintain signal integrity with shorter switching windows). But power grows as $V^2 \times f$ -- roughly cubic in frequency for aggressive scaling.

**Dennard Scaling (ended ~2006):** As transistors shrank, voltage was supposed to scale down proportionally, keeping power density constant. This stopped working below ~65nm because leakage current explodes at low voltages. Consequence: we can't just shrink and speed up anymore, leading to the multi-core era.

### Energy-Delay Product (EDP)

The standard metric for energy efficiency:

$$\text{EDP} = E \times T = P \times T^2$$

Lower EDP means better efficiency. Architects use this to compare designs at different voltage/frequency operating points.

A related metric, $\text{ED}^2\text{P}$ (Energy-Delay-Squared Product), weights performance more heavily and is used for high-performance design comparison.

### DVFS (Dynamic Voltage and Frequency Scaling)

Modern processors adjust voltage and frequency in real-time based on workload:

- **Idle:** Drop to minimum V/f, enter deep sleep states (C-states on x86)
- **Light load:** Run at moderate V/f
- **Heavy load:** Boost to maximum V/f (Intel Turbo Boost, AMD Precision Boost)
- **Thermal throttle:** If temperature exceeds limit, reduce V/f to prevent damage

The OS and hardware work together: the OS sets a power policy (performance, balanced, powersave), and the CPU's power management unit selects the actual operating point.

---

## 6. Superscalar and Out-of-Order Execution Theory

### ILP (Instruction-Level Parallelism)

The theoretical maximum parallelism extractable from a sequential instruction stream. Studies on typical programs show:

- **Integer code:** ILP of 2-4 (lots of dependencies)
- **Floating-point code:** ILP of 4-8 (more independent operations)
- **Multimedia/SIMD code:** ILP of 8-16+ (uniform operations on data arrays)

The hardware ILP a CPU can exploit is limited by:
- Issue width (how many instructions dispatched per cycle)
- Window size (how far ahead the CPU looks for independent instructions)
- Physical register count (for renaming to eliminate false dependencies)
- Functional unit count and types

### Register Renaming

Eliminates false (name) dependencies:

```
# Before renaming (WAW and WAR hazards):
  ADD R1, R2, R3    # R1 = R2 + R3
  MUL R1, R4, R5    # R1 = R4 * R5  (WAW: both write R1)
  SUB R6, R1, R7    # R6 = R1 - R7  (which R1?)

# After renaming (physical registers P1-P99):
  ADD P10, P2, P3   # P10 = P2 + P3
  MUL P11, P4, P5   # P11 = P4 * P5  (no conflict, different physical reg)
  SUB P12, P11, P7  # P12 = P11 - P7  (clear dependency on MUL)
```

The Rename/Allocate stage maintains a Register Alias Table (RAT) mapping architectural registers to physical registers. Modern x86 CPUs have ~180-200 physical integer registers for 16 architectural ones.

### Speculative Execution

The CPU predicts the outcome of branches and executes instructions along the predicted path before the branch resolves. If the prediction is correct, results commit normally. If wrong, all speculative state is discarded.

**Security implications:** Spectre and Meltdown (2018) showed that speculative execution leaves observable traces in the cache hierarchy. Even when results are architecturally discarded, the timing side-channel persists. This led to hardware and software mitigations (IBRS, retpolines, SSBD) that trade performance for security.

---

## 7. Memory Ordering and Consistency Models

### The Problem

When multiple cores access shared memory, what ordering guarantees does the hardware provide?

### Consistency Models (Weakest to Strongest)

**Sequential Consistency (SC):** All cores see memory operations in the same total order. Simplest to reason about, but severely limits hardware optimization. No modern high-performance CPU implements strict SC.

**Total Store Order (TSO):** Used by x86. Stores can be reordered after loads, but stores from a single core are seen in order by all cores. Store buffers are visible to the issuing core before being visible to others. Most x86 code "just works" because TSO is close to SC.

**Relaxed models (ARM, RISC-V, POWER):** Both loads and stores can be reordered. Programmers must insert explicit memory barriers (fences) to enforce ordering. More hardware freedom enables better performance, but programming is harder.

### Memory Barriers

```
# x86 (TSO -- most things are already ordered):
  MFENCE    # full fence, rarely needed
  SFENCE    # store fence
  LFENCE    # load fence (mostly for speculation control)

# ARM (relaxed):
  DMB       # data memory barrier
  DSB       # data synchronization barrier
  ISB       # instruction synchronization barrier

# C11/C++11 atomics abstract over hardware:
  memory_order_relaxed     # no ordering
  memory_order_acquire     # no loads/stores before this can move after
  memory_order_release     # no loads/stores after this can move before
  memory_order_seq_cst     # full sequential consistency
```

---

## 8. Modern Microarchitecture Trends

### Dark Silicon

At advanced process nodes (7nm, 5nm, 3nm), there are more transistors on a die than can be powered simultaneously without overheating. The unutilized transistors are "dark silicon." This drives heterogeneous designs:

- **Big.LITTLE / big-small:** High-performance cores + efficiency cores (ARM, Apple, Intel 12th gen+)
- **Accelerators:** Dedicate transistors to specialized hardware that's dark when unused (GPU, NPU, media encode/decode, cryptography)
- **Chiplets:** Split the die into smaller pieces (AMD Zen), each can be powered independently

### Chiplet Architecture

Instead of one monolithic die, use multiple smaller dies connected via a high-bandwidth interconnect:

- **Yield advantage:** Smaller dies have exponentially higher yield. A defect kills a small chiplet, not an entire large die
- **Mix-and-match process nodes:** I/O chiplet on mature 14nm (where analog is easier), compute chiplets on leading-edge 5nm
- **Scalability:** Add more compute chiplets for more cores
- **AMD EPYC example:** Up to 8 compute chiplets (CCDs) + 1 I/O die, 96+ cores
- **Interconnect challenge:** Cross-chiplet latency and bandwidth must be managed carefully (AMD's Infinity Fabric, Intel's EMIB/Foveros)

### Near-Memory and In-Memory Computing

Moving computation closer to data rather than moving data to computation:

- **HBM (High Bandwidth Memory):** Stacked DRAM dies connected to the processor via a silicon interposer. Used in GPUs and AI accelerators. Bandwidth: 1-3 TB/s vs ~50 GB/s for DDR5
- **Processing-in-Memory (PIM):** Add simple compute units inside DRAM chips. Samsung's HBM-PIM adds MAC (multiply-accumulate) units directly in memory dies
- **CXL (Compute Express Link):** PCIe-based protocol for coherent memory sharing across CPUs, GPUs, and memory expanders. Enables memory pooling and disaggregation

---

## 9. Putting It All Together: Roofline Model

The Roofline model visualizes whether a workload is compute-bound or memory-bound:

$$\text{Attainable Performance} = \min(\text{Peak FLOPS}, \text{Peak Bandwidth} \times \text{Operational Intensity})$$

Where:
- **Operational Intensity** = FLOPs per byte transferred from memory (FLOP/byte)
- **Peak FLOPS** = theoretical max floating-point operations per second
- **Peak Bandwidth** = theoretical max memory bandwidth

On a log-log plot:
- Below the roofline ridge point: **memory-bound** (optimize data movement, cache usage)
- At or near peak: **compute-bound** (optimize arithmetic, use SIMD/vectorization)
- The ridge point where the two lines meet: $\text{OI}_{ridge} = \frac{\text{Peak FLOPS}}{\text{Peak BW}}$

**Example:** CPU with 100 GFLOPS peak, 50 GB/s bandwidth:

$$\text{OI}_{ridge} = \frac{100}{50} = 2 \text{ FLOP/byte}$$

A kernel with OI = 0.5 FLOP/byte is memory-bound. Maximum performance: $0.5 \times 50 = 25$ GFLOPS (only 25% of peak). Adding more ALUs won't help -- you need better memory access patterns or more bandwidth.

---

## References

- Hennessy & Patterson, "Computer Architecture: A Quantitative Approach" (6th Edition) -- the definitive reference
- Patterson & Hennessy, "Computer Organization and Design: RISC-V Edition" -- undergraduate-level companion
- Shen & Lipasti, "Modern Processor Design: Fundamentals of Superscalar Processors"
- Wulf & McKee, "Hitting the Memory Wall: Implications of the Obvious" (1995) -- prescient paper on the memory wall
- Esmaeilzadeh et al., "Dark Silicon and the End of Multicore Scaling" (ISCA 2011)
- Williams, Waterman & Patterson, "Roofline: An Insightful Visual Performance Model" (2009)
- RISC-V Specification: riscv.org/technical/specifications
- Kocher et al., "Spectre Attacks: Exploiting Speculative Execution" (2019)
- Amdahl, "Validity of the Single Processor Approach to Achieving Large Scale Computing Capabilities" (1967)
- Gustafson, "Reevaluating Amdahl's Law" (1988)
