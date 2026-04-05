# x86-64 Assembly (From Calculator Slots to Syscalls)

A tiered guide to x86-64 assembly language -- from the simplest analogy to college-level optimization.

## ELI5 (Explain Like I'm 5)

### The Calculator With Labeled Slots

Imagine you have a really fast calculator, but this calculator is special. It
has a bunch of **labeled slots** where it keeps numbers. These slots are called
**registers**.

- **rax** -- the "answer" slot. When the calculator finishes a math problem,
  the answer usually goes here
- **rbx, rcx, rdx** -- extra slots to hold numbers while you work
- **rsp** -- points to the top of a stack of sticky notes (more on that later)
- **rip** -- tells the calculator which step of the recipe to do next

The calculator follows a **recipe** (a program). Each line of the recipe is
one simple instruction:

- "Put the number 5 in slot rax" -- this is a `mov` instruction
- "Add what's in slot rbx to slot rax" -- this is an `add` instruction
- "Jump to step 10" -- this is a `jmp` instruction

The recipe is written in **assembly language** -- it is the closest a human
can get to speaking the computer's own language.

### Why Does This Matter?

Every program you've ever used -- every game, every browser, every app -- is
eventually turned into these tiny recipe steps before the computer can run it.
Assembly is what the computer actually reads.

## Middle School

### Registers: The CPU's Scratch Paper

x86-64 gives you 16 general-purpose registers, each holding a 64-bit number:

```
# The 16 general-purpose registers
  rax   rbx   rcx   rdx
  rsi   rdi   rbp   rsp
  r8    r9    r10   r11
  r12   r13   r14   r15

# Special registers (not directly accessible as general-purpose)
  rip   -- instruction pointer (which line we're on)
  rflags -- flags register (stores results of comparisons)
```

Each register can also be accessed in smaller pieces:

```
# rax (64-bit) can be accessed as:
  rax  = full 64 bits
  eax  = lower 32 bits
  ax   = lower 16 bits
  ah   = bits 8-15 (high byte of ax)
  al   = bits 0-7  (low byte of ax)

# For r8-r15:
  r8   = full 64 bits
  r8d  = lower 32 bits
  r8w  = lower 16 bits
  r8b  = lower 8 bits
```

### Simple Instructions

```
# Moving data around
  mov rax, 42         # put 42 into rax
  mov rbx, rax        # copy rax into rbx

# Basic math
  add rax, rbx        # rax = rax + rbx
  sub rax, 10         # rax = rax - 10
  inc rcx             # rcx = rcx + 1
  dec rcx             # rcx = rcx - 1

# Multiplication and division
  imul rax, rbx       # rax = rax * rbx
  # Division is special: divide rdx:rax by the operand
  # quotient goes in rax, remainder in rdx
```

### Flags: Did Something Happen?

After most math instructions, the CPU sets **flags** that tell you about the
result:

```
# Key flags in rflags register:
  ZF (Zero Flag)    -- set if result was zero
  SF (Sign Flag)    -- set if result was negative
  CF (Carry Flag)   -- set if unsigned overflow occurred
  OF (Overflow Flag) -- set if signed overflow occurred
```

### From Assembly to Machine Code

Assembly is human-readable. The CPU cannot read it directly. An **assembler**
translates assembly into **machine code** -- raw bytes the CPU understands:

```
# Assembly:        Machine code (hex):
  mov rax, 42      48 c7 c0 2a 00 00 00
  add rax, rbx     48 01 d8
  ret              c3

# Each instruction becomes a specific pattern of bytes
# The CPU decodes these bytes back into operations
```

## High School

### Register Conventions (System V AMD64 ABI)

On Linux and macOS, there are rules about which registers do what when
calling functions:

```
# Function arguments (in order):
  rdi = 1st argument
  rsi = 2nd argument
  rdx = 3rd argument
  rcx = 4th argument
  r8  = 5th argument
  r9  = 6th argument
  # Additional arguments go on the stack

# Return value:
  rax = return value (up to 64 bits)
  rdx = second return value (for 128-bit returns)

# Caller-saved (you must save these before calling a function):
  rax, rcx, rdx, rsi, rdi, r8, r9, r10, r11

# Callee-saved (the function you call must preserve these):
  rbx, rbp, r12, r13, r14, r15

# Special:
  rsp = stack pointer (must be 16-byte aligned before call)
  rbp = base pointer (often used as frame pointer)
```

### AT&T vs Intel Syntax

x86-64 assembly has two syntax styles:

```
# Intel syntax (destination first -- used by NASM, Intel docs):
  mov rax, rbx          # rax = rbx
  add rax, [rbx+8]      # rax = rax + *(rbx+8)
  mov dword [rsp], 42   # *(rsp) = 42

# AT&T syntax (source first -- used by GAS, GCC default):
  movq %rbx, %rax       # rax = rbx
  addq 8(%rbx), %rax    # rax = rax + *(rbx+8)
  movl $42, (%rsp)      # *(rsp) = 42

# AT&T differences:
#   - Registers prefixed with %
#   - Immediates prefixed with $
#   - Operand order is reversed (source, destination)
#   - Size suffixes: b=byte, w=word, l=long, q=quad
```

### Stack Operations

The stack grows downward in memory (toward lower addresses):

```
# push: decrements rsp, stores value at new rsp
  push rax       # rsp -= 8; [rsp] = rax
  push rbx       # rsp -= 8; [rsp] = rbx

# pop: loads value from rsp, increments rsp
  pop rbx        # rbx = [rsp]; rsp += 8
  pop rax        # rax = [rsp]; rsp += 8

# Stack layout during a function call:
#   high addresses
#   ┌───────────────────┐
#   │ caller's frame    │
#   │ return address     │ ← pushed by call instruction
#   │ saved rbp         │ ← pushed by prologue
#   │ local variable 1  │ ← rbp-8
#   │ local variable 2  │ ← rbp-16
#   │ ...               │
#   └───────────────────┘ ← rsp (current stack top)
#   low addresses
```

### Function Prologue and Epilogue

Every function follows a pattern:

```asm
my_function:
    # Prologue -- set up stack frame
    push rbp             # save caller's base pointer
    mov rbp, rsp         # set our base pointer to current stack top
    sub rsp, 32          # allocate 32 bytes for local variables

    # ... function body ...
    # local variables at [rbp-8], [rbp-16], etc.
    # arguments in rdi, rsi, rdx, rcx, r8, r9

    # Epilogue -- tear down stack frame
    mov rsp, rbp         # deallocate locals
    pop rbp              # restore caller's base pointer
    ret                  # pop return address, jump to it

# Shorthand equivalents:
#   enter 32, 0   ≈ push rbp; mov rbp,rsp; sub rsp,32  (slow, rarely used)
#   leave         ≈ mov rsp,rbp; pop rbp
```

### Conditional Jumps and Comparisons

```
# cmp sets flags by computing (a - b) without storing result
  cmp rax, rbx          # compute rax - rbx, set flags

# Conditional jumps based on flags:
  je   label            # jump if equal         (ZF=1)
  jne  label            # jump if not equal     (ZF=0)
  jg   label            # jump if greater       (signed: ZF=0 and SF=OF)
  jge  label            # jump if greater/equal (signed: SF=OF)
  jl   label            # jump if less          (signed: SF≠OF)
  jle  label            # jump if less/equal    (signed: ZF=1 or SF≠OF)
  ja   label            # jump if above         (unsigned: CF=0 and ZF=0)
  jb   label            # jump if below         (unsigned: CF=1)

# Example: if (rax > 10) goto bigger
  cmp rax, 10
  jg  bigger

# test performs AND without storing (useful for zero/sign checks)
  test rax, rax         # rax & rax -- sets ZF if rax is zero
  jz   is_zero          # jump if rax was zero
```

### Addressing Modes

```
# Immediate:    mov rax, 42            # rax = 42
# Register:     mov rax, rbx           # rax = rbx
# Direct:       mov rax, [0x601000]    # rax = *(0x601000)
# Indirect:     mov rax, [rbx]         # rax = *rbx
# Base+offset:  mov rax, [rbx+16]      # rax = *(rbx+16)
# Scaled index: mov rax, [rbx+rcx*8]   # rax = *(rbx + rcx*8)
# Full form:    mov rax, [rbx+rcx*4+16]  # rax = *(rbx + rcx*4 + 16)

# General addressing: [base + index*scale + displacement]
#   base:   any register
#   index:  any register except rsp
#   scale:  1, 2, 4, or 8
#   disp:   8-bit or 32-bit signed constant
```

### A Complete Example: Adding Two Numbers

```asm
; int add_two(int a, int b) -- a in edi, b in esi
add_two:
    push rbp
    mov rbp, rsp
    mov eax, edi         ; eax = first argument
    add eax, esi         ; eax += second argument
    pop rbp
    ret                  ; return value in eax

; main calls add_two(3, 7)
main:
    push rbp
    mov rbp, rsp
    mov edi, 3           ; first argument
    mov esi, 7           ; second argument
    call add_two         ; pushes return address, jumps to add_two
    ; eax now holds 10
    pop rbp
    ret
```

## College

### SIMD: Single Instruction, Multiple Data

SIMD lets the CPU operate on multiple values at once using wide registers:

```
# SIMD register sets:
  SSE:      xmm0-xmm15     128-bit (4 floats or 2 doubles)
  AVX:      ymm0-ymm15     256-bit (8 floats or 4 doubles)
  AVX-512:  zmm0-zmm31     512-bit (16 floats or 8 doubles)

# Example: add 4 floats at once (SSE)
  movaps xmm0, [rdi]        # load 4 floats from memory
  movaps xmm1, [rsi]        # load 4 more floats
  addps  xmm0, xmm1         # add all 4 pairs simultaneously
  movaps [rdx], xmm0        # store 4 results

# AVX example: add 8 floats at once
  vmovaps ymm0, [rdi]       # load 8 floats (256 bits)
  vaddps  ymm0, ymm0, [rsi] # add 8 pairs (3-operand form)
  vmovaps [rdx], ymm0       # store 8 results

# Key instruction families:
  addps/addpd     # packed single/double add
  mulps/mulpd     # packed single/double multiply
  shufps          # shuffle elements within register
  blendps         # select elements from two registers
  dpps            # dot product
  pmaddwd         # packed multiply-add (integer)

# AVX-512 adds:
  - 32 zmm registers (vs 16 xmm/ymm)
  - 8 opmask registers (k0-k7) for predicated execution
  - Gather/scatter instructions for non-contiguous memory
```

### Memory Ordering Instructions

x86-64 has TSO (Total Store Order) -- stronger than ARM/RISC-V but still
needs fences for certain operations:

```
# Memory fence instructions:
  mfence        # full barrier -- all loads and stores before this
                # complete before any loads or stores after it
  sfence        # store fence -- all stores before this are visible
                # before any stores after it (used with NT stores)
  lfence        # load fence -- all loads before this complete before
                # any loads after it (also serializes speculative exec)

# Non-temporal stores (bypass cache, write-combining):
  movntps [rdi], xmm0     # store 128 bits, skip cache
  movntdq [rdi], xmm0     # store 128 bits integer, skip cache
  movnti  [rdi], eax      # store 32 bits, skip cache
  # Requires sfence afterward to ensure visibility

# When are fences needed on x86-64?
# - After non-temporal (NT) stores
# - In lock-free data structures (rare, TSO handles most cases)
# - lfence for Spectre mitigation
# - Most x86 code does NOT need explicit fences
```

### Atomic Operations: lock Prefix and cmpxchg

```
# lock prefix makes read-modify-write atomic:
  lock add [rdi], 1          # atomic increment
  lock xadd [rdi], eax       # atomic fetch-and-add (old value in eax)
  lock bts [rdi], 5          # atomic test-and-set bit 5
  lock inc dword [counter]   # atomic increment

# cmpxchg (compare and exchange) -- the CAS primitive:
  # Compares rax with [rdi]. If equal, stores rsi at [rdi].
  # If not equal, loads [rdi] into rax.
  lock cmpxchg [rdi], rsi    # atomic CAS

# 128-bit CAS:
  lock cmpxchg16b [rdi]      # compare rdx:rax with [rdi],
                              # if equal store rcx:rbx at [rdi]
                              # used for double-width CAS (DCAS)

# xchg is always atomic (implicit lock):
  xchg [rdi], rax            # swap [rdi] and rax atomically

# These map to C/C++ atomics:
#   lock add     → atomic_fetch_add
#   lock cmpxchg → atomic_compare_exchange
#   xchg         → atomic_exchange
```

### The syscall Instruction

```
# On x86-64 Linux, system calls use the syscall instruction:
  # Argument passing:
  #   rax = syscall number
  #   rdi = arg1, rsi = arg2, rdx = arg3
  #   r10 = arg4, r8 = arg5, r9 = arg6
  #   (Note: r10 replaces rcx because syscall clobbers rcx and r11)

  # Return:
  #   rax = return value (negative = errno)
  #   rcx and r11 are clobbered (rcx=saved rip, r11=saved rflags)

# Example: write(1, msg, 13) -- write "Hello, world\n" to stdout
  mov rax, 1              # syscall number for write
  mov rdi, 1              # fd = stdout
  lea rsi, [rel msg]      # pointer to message
  mov rdx, 13             # length
  syscall                 # kernel entry

# Example: exit(0)
  mov rax, 60             # syscall number for exit
  xor edi, edi            # status = 0
  syscall
```

### Position-Independent Code (PIC)

```
# PIC uses RIP-relative addressing so code works at any load address
# Required for shared libraries (.so), recommended for executables (PIE)

# RIP-relative addressing:
  lea rax, [rip+offset]       # load address relative to current rip
  mov rax, [rip+my_global]    # load global variable via RIP-relative

# Global Offset Table (GOT):
  mov rax, [rip+var@GOTPCREL] # load address of var from GOT
  mov rax, [rax]              # dereference to get value

# Procedure Linkage Table (PLT) for external function calls:
  call printf@PLT             # indirect call through PLT stub
  # First call: PLT stub jumps to dynamic linker, resolves address
  # Subsequent calls: PLT stub jumps directly to resolved address

# Compiling PIC:
  gcc -fPIC -shared -o libfoo.so foo.c   # shared library
  gcc -fPIE -pie -o program main.c       # position-independent executable
```

### Inline Assembly

```c
// C (GCC extended inline assembly):
int result;
int a = 10, b = 20;
__asm__ __volatile__ (
    "addl %2, %1\n\t"      // add b to a
    "movl %1, %0\n\t"      // move result
    : "=r" (result)         // output: result in any register
    : "r" (a), "r" (b)     // inputs: a and b in registers
    : "cc"                  // clobbers: condition codes
);
// result == 30
```

```rust
// Rust (stable since 1.59):
use std::arch::asm;
let x: u64 = 42;
let y: u64;
unsafe {
    asm!(
        "mov {output}, {input}",
        "add {output}, 10",
        input = in(reg) x,
        output = out(reg) y,
    );
}
// y == 52
```

```go
// Go does not support inline assembly directly.
// Instead, write assembly in .s files:
// file: add_amd64.s
// func Add(a, b int64) int64
// TEXT ·Add(SB), NOSPLIT, $0-24
//     MOVQ a+0(FP), AX
//     ADDQ b+8(FP), AX
//     MOVQ AX, ret+16(FP)
//     RET
//
// Go uses Plan 9 assembly syntax (different from AT&T/Intel):
//   MOVQ = mov, ADDQ = add, FP = frame pointer pseudo-register
//   Arguments accessed via offsets from FP
```

### Performance: IPC, Pipeline Stalls, Branch Misprediction

```
# Instructions Per Cycle (IPC):
#   Modern x86-64 CPUs are 4-6 wide superscalar
#   Theoretical max IPC = decode width (e.g., 6 on Zen 4)
#   Sustained IPC on real code: typically 2-4
#   Measure with: perf stat -e instructions,cycles ./program
#     IPC = instructions / cycles

# Pipeline stalls (things that waste cycles):
#   - Cache miss:        L1 miss costs ~4 cycles, L2 ~12, L3 ~40, RAM ~200+
#   - Branch mispredict: ~15-20 cycles on modern cores (pipeline flush)
#   - Data dependency:   1-3 cycles for simple forwarding
#   - Store-to-load:     ~4-5 cycles if load depends on recent store
#   - Port contention:   instructions compete for execution units

# Branch misprediction cost:
#   Mispredict rate on typical code: ~2-5%
#   Cost per mispredict: ~15-20 cycles (varies by microarchitecture)
#   Total cost: 15-20% branches × 3% mispredict × 17 cycles ≈ 0.08 CPI
#   Mitigate: branchless code (cmov, sbb, bit tricks), profile-guided opt

# Useful perf counters:
  perf stat -e cycles,instructions,cache-misses,branch-misses ./prog
  perf record -e cycles ./prog && perf report
```

## Tips

- Start by reading compiler output: `gcc -S -O2 -masm=intel file.c` or `objdump -d -M intel binary`
- Godbolt Compiler Explorer (godbolt.org) is the fastest way to see what the compiler generates
- On x86-64, writing to a 32-bit register (eax) automatically zero-extends to 64 bits -- this is a common source of subtle bugs when porting from 32-bit code
- `xor eax, eax` is the idiomatic way to zero a register (shorter encoding than `mov eax, 0`, and breaks dependency chains)
- The red zone: on System V AMD64, the 128 bytes below rsp are reserved for leaf functions -- they can use this space without adjusting rsp
- Align hot loops to 32-byte boundaries for best instruction fetch performance
- `rep movsb` / `rep stosb` are highly optimized on modern CPUs (ERMS) and often beat hand-written loops for memcpy/memset

## See Also

- how-computers-work
- linux-kernel-internals
- binary-and-number-systems
- gdb

## References

- Intel 64 and IA-32 Architectures Software Developer Manuals (SDM), Vol. 1-3
- AMD64 Architecture Programmer's Manual, Vol. 1-5
- System V Application Binary Interface, AMD64 Architecture Processor Supplement
- Agner Fog, "Optimizing Assembly" and "Instruction Tables" (agner.org/optimize)
- Matt Godbolt, Compiler Explorer (godbolt.org)
- Ryan A. Chapman, "x86-64 Assembly Language Programming with Ubuntu"
- Felix Cloutier, x86 instruction reference (felixcloutier.com/x86)
