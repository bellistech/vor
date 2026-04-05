# RISC-V (The Open Instruction Set Architecture)

A tiered guide to the RISC-V processor architecture.

## ELI5

Imagine you want to build a toy robot. Most robot kits come from big
companies who charge you money just to use their *instruction manual* --
even before you buy any parts. If you change anything, you need their
permission.

**RISC-V** is a free, open-source instruction manual for building
processors (the brain of a computer). Anyone in the world can read it,
use it, and build their own chip -- without paying anyone or asking
permission.

Think of it like **Linux, but for chips**:

- Linux is a free operating system that anyone can use and modify.
- RISC-V is a free processor design that anyone can use and modify.

Big companies like Intel and ARM keep their instruction manuals secret
and charge licensing fees. RISC-V said: "What if the manual was free?"
Now universities, startups, and even big companies can build their own
processors without paying royalties.

The name means "Reduced Instruction Set Computer, version five" -- it is
the fifth generation of a research project from UC Berkeley.

## Middle School

### What Is an ISA?

An **Instruction Set Architecture (ISA)** is the contract between
software and hardware. It defines:

- What instructions the processor understands (add, subtract, load, store).
- How many registers (fast storage slots) the processor has.
- How data is laid out in memory.

RISC-V is an ISA. It does not describe a specific chip -- it describes
the *language* that chips speak. Different companies can build different
chips that all understand the same RISC-V instructions.

### Registers

RISC-V has **32 general-purpose registers**, named `x0` through `x31`:

| Register | ABI Name | Purpose |
|:---|:---|:---|
| `x0` | `zero` | Hardwired to 0 (writes are ignored) |
| `x1` | `ra` | Return address |
| `x2` | `sp` | Stack pointer |
| `x3` | `gp` | Global pointer |
| `x4` | `tp` | Thread pointer |
| `x5-x7` | `t0-t2` | Temporaries |
| `x8` | `s0/fp` | Saved register / frame pointer |
| `x9` | `s1` | Saved register |
| `x10-x11` | `a0-a1` | Function arguments / return values |
| `x12-x17` | `a2-a7` | Function arguments |
| `x18-x27` | `s2-s11` | Saved registers |
| `x28-x31` | `t3-t6` | Temporaries |

The `x0 = zero` register is a clever trick. Need to compare something
to zero? Just use `x0`. Need to discard a result? Write it to `x0`.
This eliminates special instructions that other architectures need.

### Basic RV32I Instructions

RV32I is the base 32-bit integer instruction set -- the minimum every
RISC-V processor must support:

```asm
# Arithmetic
add  x3, x1, x2      # x3 = x1 + x2
sub  x3, x1, x2      # x3 = x1 - x2
addi x3, x1, 10      # x3 = x1 + 10 (immediate)

# Logical
and  x3, x1, x2      # x3 = x1 & x2 (bitwise AND)
or   x3, x1, x2      # x3 = x1 | x2 (bitwise OR)
xor  x3, x1, x2      # x3 = x1 ^ x2 (bitwise XOR)

# Load and Store (memory access)
lw   x3, 0(x1)       # x3 = Memory[x1 + 0]  (load word, 32 bits)
sw   x3, 0(x1)       # Memory[x1 + 0] = x3  (store word, 32 bits)
lb   x3, 0(x1)       # load byte (8 bits, sign-extended)
lh   x3, 0(x1)       # load halfword (16 bits, sign-extended)

# Branching (conditional jumps)
beq  x1, x2, label   # if x1 == x2, jump to label
bne  x1, x2, label   # if x1 != x2, jump to label
blt  x1, x2, label   # if x1 < x2 (signed), jump to label
bge  x1, x2, label   # if x1 >= x2 (signed), jump to label

# Jump
jal  x1, label        # jump to label, save return address in x1
jalr x1, x2, 0        # jump to address in x2, save return in x1
```

### Instruction Formats

Every RV32I instruction is exactly **32 bits** (4 bytes). There are six
formats, each arranging the bits differently:

```
R-type:  [funct7 | rs2 | rs1 | funct3 | rd  | opcode]  — register-register (add, sub, and, or)
I-type:  [imm[11:0]    | rs1 | funct3 | rd  | opcode]  — immediate (addi, lw, jalr)
S-type:  [imm[11:5] | rs2 | rs1 | funct3 | imm[4:0] | opcode]  — store (sw, sh, sb)
B-type:  [imm[12|10:5] | rs2 | rs1 | funct3 | imm[4:1|11] | opcode]  — branch (beq, bne)
U-type:  [imm[31:12]                       | rd  | opcode]  — upper immediate (lui, auipc)
J-type:  [imm[20|10:1|11|19:12]            | rd  | opcode]  — jump (jal)
```

The opcode is always in the same position (bits 6:0). The register
fields (`rs1`, `rs2`, `rd`) are always in the same position when present.
This regularity makes hardware decoding simple and fast.

## High School

### RV64I — 64-bit Extension

RV64I extends the base to 64-bit registers and addresses. It adds
word-suffixed instructions for 32-bit operations on 64-bit registers:

```asm
addw  x3, x1, x2     # 32-bit add, result sign-extended to 64 bits
subw  x3, x1, x2     # 32-bit subtract
sllw  x3, x1, x2     # 32-bit shift left logical
ld    x3, 0(x1)      # load doubleword (64 bits)
sd    x3, 0(x1)      # store doubleword (64 bits)
```

### Standard Extensions

RISC-V is modular. The base ISA is minimal; capabilities are added via
lettered extensions:

| Extension | Name | What It Adds |
|:---|:---|:---|
| **M** | Multiply/Divide | `mul`, `div`, `rem` instructions |
| **A** | Atomics | `lr` (load-reserved), `sc` (store-conditional), atomic swap/add/and/or |
| **F** | Single-Precision Float | 32 float registers (`f0-f31`), `fadd.s`, `fmul.s`, `fcvt` |
| **D** | Double-Precision Float | extends F to 64-bit floats: `fadd.d`, `fmul.d` |
| **C** | Compressed | 16-bit encodings of common instructions (reduces code size ~25-30%) |
| **G** | General | Shorthand for `IMAFD` — the standard general-purpose combination |

A processor described as **RV64GC** supports: 64-bit base (I) + multiply
(M) + atomics (A) + single float (F) + double float (D) + compressed
(C). This is the standard profile for Linux-capable RISC-V processors.

### Calling Convention

The RISC-V calling convention defines how functions pass data:

```
Arguments:       a0-a7  (x10-x17)    — first 8 integer arguments
Return values:   a0-a1  (x10-x11)    — return value(s)
Callee-saved:    s0-s11 (x8-x9, x18-x27) — function must preserve these
Caller-saved:    t0-t6, a0-a7, ra    — function may overwrite these
Stack pointer:   sp (x2)             — must be 16-byte aligned at call
Frame pointer:   s0/fp (x8)          — optional, used for debugging
```

Example function prologue and epilogue:

```asm
my_function:
    addi sp, sp, -32      # allocate stack frame
    sd   ra, 24(sp)       # save return address
    sd   s0, 16(sp)       # save frame pointer
    addi s0, sp, 32       # set frame pointer

    # ... function body uses a0-a7 for args ...

    ld   ra, 24(sp)       # restore return address
    ld   s0, 16(sp)       # restore frame pointer
    addi sp, sp, 32       # deallocate stack frame
    ret                   # jalr x0, ra, 0
```

### System Calls (ECALL)

RISC-V uses the `ecall` instruction to request services from a higher
privilege level:

```asm
# Linux syscall: write(1, "hello", 5)
li   a7, 64              # syscall number for write
li   a0, 1               # fd = stdout
la   a1, hello_str       # buffer address
li   a2, 5               # length
ecall                     # trap to kernel
# return value in a0
```

### CSR Registers

**Control and Status Registers (CSRs)** control processor behavior:

```asm
csrr  x1, mstatus        # read CSR into x1
csrw  mstatus, x1        # write x1 into CSR
csrrs x1, mie, x2        # read CSR, then set bits from x2
csrrc x1, mie, x2        # read CSR, then clear bits from x2
```

Key CSRs:

| CSR | Name | Purpose |
|:---|:---|:---|
| `mstatus` | Machine Status | Global interrupt enable, privilege mode bits |
| `mtvec` | Machine Trap Vector | Address of trap handler |
| `mepc` | Machine Exception PC | PC at time of trap |
| `mcause` | Machine Cause | Reason for trap (interrupt or exception code) |
| `mie` / `mip` | Machine Interrupt Enable/Pending | Per-interrupt enable and pending bits |
| `cycle` | Cycle Counter | Hardware cycle counter (read-only from U-mode) |

### Privilege Levels

RISC-V defines three privilege levels:

```
M-mode (Machine)      — highest privilege, always present, runs firmware/bootloader
  ↓
S-mode (Supervisor)   — runs the OS kernel, manages virtual memory
  ↓
U-mode (User)         — lowest privilege, runs applications
```

- **M-mode** is mandatory. Simple embedded systems may only have M-mode.
- **S-mode** adds virtual memory (page tables) and is required for
  running Linux.
- **U-mode** provides user/kernel isolation.
- Traps (interrupts, exceptions, ecall) cause upward transitions.
  `mret` / `sret` return to lower privilege.

## College

### Vector Extension (V)

The RISC-V Vector extension (RVV) provides scalable SIMD:

```asm
# Set vector length for 32-bit elements
vsetvli t0, a0, e32, m1    # t0 = min(a0, VLMAX), 32-bit elements, LMUL=1

# Vector load, add, store
vle32.v v1, (a1)            # load vector from memory at a1
vle32.v v2, (a2)            # load vector from memory at a2
vadd.vv v3, v1, v2          # v3 = v1 + v2 (element-wise)
vse32.v v3, (a3)            # store result to memory at a3
```

Unlike x86 SSE/AVX (fixed 128/256/512-bit), RVV is **vector-length
agnostic**: the same binary runs on hardware with different vector
register widths (128-bit to 16384-bit). The `vsetvli` instruction
negotiates the actual vector length at runtime.

### Custom Extensions (X)

RISC-V reserves opcode space for custom instructions:

| Opcode Range | Name | Purpose |
|:---|:---|:---|
| `custom-0` (0x0B) | Reserved | Custom instructions |
| `custom-1` (0x2B) | Reserved | Custom instructions |
| `custom-2` (0x5B) | Reserved | 48-bit+ custom instructions |
| `custom-3` (0x7B) | Reserved | 64-bit+ custom instructions |

Companies use these for hardware accelerators (AI inference, crypto,
DSP) without forking the ISA.

### RISC-V Memory Model (RVWMO)

RISC-V defines the **RVWMO** (RISC-V Weak Memory Ordering) model:

- Single-threaded code sees loads and stores in program order.
- Multi-threaded code may observe reordering across harts (hardware
  threads) unless constrained by ordering annotations or fences.

Ordering primitives:

```asm
# Fence instruction
fence rw, rw              # full memory barrier (all loads/stores before
                          # this complete before any after)
fence r, r                # load-load barrier
fence w, w                # store-store barrier
fence.tso                 # TSO fence (acquire + release, x86-like ordering)

# Atomic instructions with ordering
amoswap.w.aq x3, x2, (x1)   # atomic swap with acquire semantics
amoswap.w.rl x3, x2, (x1)   # atomic swap with release semantics
amoswap.w.aqrl x3, x2, (x1) # atomic swap with acquire-release
lr.w.aq  x3, (x1)            # load-reserved with acquire
sc.w.rl  x3, x2, (x1)       # store-conditional with release
```

The `.aq` (acquire) and `.rl` (release) bits on atomic instructions
provide release-acquire semantics without requiring full fences.

### Hypervisor Extension (H)

The H extension adds a new privilege level for virtualization:

```
M-mode (Machine)
  ↓
HS-mode (Hypervisor-extended Supervisor)  — runs the hypervisor
  ↓
VS-mode (Virtual Supervisor)              — runs guest OS kernel
  ↓
VU-mode (Virtual User)                    — runs guest applications
```

Key features: two-stage address translation (guest virtual -> guest
physical -> host physical), virtual interrupt injection, and trap
delegation to the hypervisor.

### PLIC (Platform-Level Interrupt Controller)

The PLIC distributes external interrupts to harts:

```
External Sources (UART, SPI, GPIO, PCIe, ...)
         ↓
   ┌─────────────┐
   │    PLIC      │  — prioritizes, routes, and gates interrupts
   └──┬──┬──┬────┘
      ↓  ↓  ↓
   Hart0 Hart1 Hart2  — each hart claims and completes interrupts
```

- Each source has a configurable priority (0 = disabled).
- Each hart has a priority threshold; only higher-priority interrupts
  are delivered.
- Interrupt flow: pending -> claimed (by a hart) -> completed.

### PMP (Physical Memory Protection)

PMP restricts physical memory access for lower privilege levels:

```
pmpcfg0-pmpcfg15     — 64 PMP configuration entries (8 bits each)
pmpaddr0-pmpaddr63   — address boundaries

Each entry specifies:
  - Address range (TOR, NA4, or NAPOT matching)
  - Permissions: R (read), W (write), X (execute)
  - L (lock): once set, M-mode cannot change the entry
```

PMP is used by M-mode firmware to restrict what S-mode (the OS kernel)
can access -- for example, protecting firmware memory regions from a
compromised kernel.

### RISC-V vs x86-64 vs ARM64

| Feature | RISC-V | x86-64 | ARM64 (AArch64) |
|:---|:---|:---|:---|
| **License** | Open (BSD) | Proprietary (Intel/AMD) | Proprietary (ARM Ltd) |
| **Royalties** | None | N/A (tied to silicon) | Per-chip royalty |
| **ISA Type** | RISC (load-store) | CISC | RISC (load-store) |
| **Instruction Width** | Fixed 32-bit (+ 16-bit C) | Variable 1-15 bytes | Fixed 32-bit |
| **Registers (int)** | 32 (x0-x31) | 16 (rax-r15) | 31 (x0-x30) |
| **Endianness** | Little (default) | Little | Little (default) |
| **SIMD** | V extension (scalable) | SSE/AVX (fixed width) | SVE/SVE2 (scalable) |
| **Memory Model** | RVWMO (weak) | TSO (strong) | Weak |
| **Decode Complexity** | Simple (fixed format) | Complex (variable length) | Moderate |
| **Ecosystem Maturity** | Growing | Dominant (desktop/server) | Dominant (mobile/embedded) |
| **Custom Extensions** | Yes (reserved opcodes) | No | No (limited) |
| **Formal Spec** | Yes (Sail model) | No | Partial (ASL) |

### RISC-V in Production

| Company | Product | Application |
|:---|:---|:---|
| SiFive | HiFive Unmatched, P670/P870 cores | Development boards, licensed IP cores |
| StarFive | JH7110 (VisionFive 2) | Linux SBC (quad-core RV64GC, 1.5 GHz) |
| Milk-V | Mars, Pioneer, Duo | SBCs and server boards |
| Espressif | ESP32-C3, ESP32-C6 | IoT microcontrollers |
| T-Head (Alibaba) | C910, C920 | Server and AI processors |
| Qualcomm | Acquired Nuvia/SiFive tech | Future mobile/server designs |
| Google | Titan M2 security chip | Pixel phone security enclave |

## Tips

- Start learning RISC-V with the online simulators (RARS, Venus) before
  touching real hardware -- they provide step-by-step execution and
  register visualization.
- The RV32I base has only 47 instructions. Learn those before tackling
  extensions. The simplicity is the point.
- When reading RISC-V assembly, remember that `x0` absorbs writes
  silently. Pseudo-instructions like `mv x1, x2` are really
  `addi x1, x2, 0` and `nop` is `addi x0, x0, 0`.
- The C (compressed) extension is almost always worth enabling -- it
  reduces code size by 25-30% with no performance penalty.
- RISC-V's weak memory model (RVWMO) means you need explicit fences or
  acquire/release annotations for correct multi-threaded code. Do not
  assume x86-like TSO ordering.
- For embedded work, many microcontrollers only implement M-mode. You
  do not need S-mode or virtual memory for bare-metal firmware.
- Use `objdump -d` with a RISC-V cross-toolchain to study how C code
  maps to RISC-V assembly.

## See Also

- how-computers-work
- binary-and-number-systems
- linux-kernel-internals

## References

- RISC-V ISA Specifications: https://riscv.org/technical/specifications/
- RISC-V Unprivileged ISA Manual (Volume 1): https://github.com/riscv/riscv-isa-manual
- RISC-V Privileged ISA Manual (Volume 2): https://github.com/riscv/riscv-isa-manual
- "Computer Organization and Design: RISC-V Edition" by Patterson & Hennessy (Morgan Kaufmann)
- RISC-V Assembly Programmer's Manual: https://github.com/riscv-non-isa/riscv-asm-manual
- SiFive: https://www.sifive.com/
- RISC-V International: https://riscv.org/
- RARS Simulator: https://github.com/TheThirdOne/rars
- Venus Online Simulator: https://venus.kvakil.me/
