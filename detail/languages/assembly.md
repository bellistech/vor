# Assembly — Deep Dive (x86-64 + ARM64)

> *Assembly is not "low-level C." It is the human-readable form of the instruction set architecture (ISA) — the contract between silicon and software. This deep dive covers the two ISAs that matter today (x86-64 and AArch64/ARM64): their register files, addressing modes, calling conventions, memory models, SIMD extensions, inline asm, syscall ABIs, and the cycle-level cost of every memory access.*

---

## 1. ISA Primer — RISC vs CISC

An **instruction set architecture (ISA)** defines:

1. **Registers** — the small set of named storage locations the CPU operates on directly.
2. **Instructions** — the primitive operations (encoded as bytes) that the silicon decoder understands.
3. **Addressing modes** — how operands name memory.
4. **Memory model** — the rules under which loads and stores from different cores observe each other.
5. **Calling convention (ABI)** — how function arguments and return values are passed (this is software, but in practice frozen per-OS).

Two living families dominate:

| Property | x86-64 (Intel/AMD) | ARM64 (AArch64) |
|:---------|:-------------------|:----------------|
| Heritage | CISC, descended from Intel 8086 (1978) | RISC, ARMv8-A (2011) clean-slate 64-bit |
| Encoding | Variable length, 1 to 15 bytes | Fixed width, 4 bytes (A64) |
| Registers | 16 GP, 16 SIMD (32 with AVX-512) | 31 GP + zero reg, 32 SIMD (NEON/SVE) |
| Memory ops | Most instructions can read/write memory | Load/store only — pure load-store architecture |
| Operands | Two-operand (`add rax, rbx` => `rax += rbx`) | Three-operand (`add x0, x1, x2` => `x0 = x1 + x2`) |
| Memory model | TSO (Total Store Order) — relatively strong | Weak — explicit barriers required |
| Endianness | Little-endian only | Bi-endian, little-endian in practice |
| Decode complexity | Microcoded, μops, register renaming | Direct decode, but still OoO |

### 1.1 The CISC vs RISC Trade-Off

Classical (1980s) framing:

- **CISC** — many complex instructions, variable encoding, instructions can do work-in-one (e.g. `repnz scasb`, string instructions, `enter`/`leave`).
- **RISC** — small, fixed-encoding instruction set; everything done by composing primitives; load-store separation.

Modern reality: x86-64 internally cracks complex instructions into **micro-ops (μops)** that look very RISC-like; ARM64 has gained complex instructions (e.g. `cbz`, `csel`, `madd`). The pipelines are similarly deep (15–20 stages), out-of-order, with register renaming. The visible difference is the *encoding* and the *philosophy of the ISA*, not the silicon underneath.

### 1.2 Instruction Encoding

**x86-64** uses variable-length encoding. A single instruction can range from 1 byte (`ret` = `0xC3`) up to 15 bytes (the architectural maximum). Encoding fields:

```
[Legacy prefixes][REX prefix][Opcode][ModR/M][SIB][Displacement][Immediate]
   0–4 bytes      0–1 byte    1–3      1       1     1/2/4         1/2/4/8
```

- **REX prefix** (`0x40`–`0x4F`) — introduced for AMD64 to access r8–r15, the high-byte registers, and 64-bit operand size. The W bit (`REX.W`) selects 64-bit operands.
- **ModR/M** byte — encodes the addressing mode and which register/memory operand goes where.
- **SIB** (Scale-Index-Base) — present when ModR/M says "complex addressing"; encodes `base + index*scale`.

Variable encoding is dense (good icache footprint) but slow to decode — modern x86 cores have wide decoders (4–6 wide on Intel Golden Cove, 4-wide on AMD Zen 4) and **μop caches** (a.k.a. "decoded stream buffer" or "DSB") to bypass the decoder on hot loops.

**ARM64** uses fixed 4-byte (32-bit) encoding for A64. Every instruction is exactly 32 bits, naturally word-aligned. Decoding is simple and parallel — the front-end can fan out 4–8 instructions per cycle without speculation about boundaries. The cost is lower code density: large immediates need multiple `movk` instructions to load, branches have ±128 MiB range without a literal pool, etc.

### 1.3 Addressing Modes

x86-64 effective address formula:

```
EA = base + (index * scale) + displacement
```

where `base` and `index` are 64-bit registers, `scale` is 1/2/4/8, and `displacement` is a sign-extended 8 or 32-bit immediate. In assembly:

```nasm
mov rax, [rbx]                     ; simple indirect
mov rax, [rbx + 8]                 ; base + disp
mov rax, [rbx + rcx*4]             ; base + index*scale
mov rax, [rbx + rcx*4 + 16]        ; full form
mov rax, [rip + symbol]            ; RIP-relative (PIC)
mov rax, [rel symbol]              ; NASM syntax for RIP-relative
```

ARM64 addressing modes (load/store only):

```gas
ldr x0, [x1]                ; pre-indexed, no offset
ldr x0, [x1, #16]           ; immediate offset
ldr x0, [x1, x2]            ; register offset
ldr x0, [x1, x2, lsl #3]    ; scaled register (lsl=3 means *8)
ldr x0, [x1], #16           ; post-indexed: read [x1], then x1 += 16
ldr x0, [x1, #16]!          ; pre-indexed write-back: x1 += 16, then read [x1]
ldr x0, =literal            ; literal pool load (PC-relative)
adrp x0, sym                ; address of page (4 KiB) containing symbol
add  x0, x0, :lo12:sym      ; add low 12 bits to get exact address
```

The `adrp + add` pair is the canonical way to get a 32-bit-PIC absolute address on AArch64 — `adrp` provides ±4 GiB PC-relative reach.

---

## 2. x86-64 Architecture

### 2.1 Register File

Sixteen 64-bit general-purpose registers. Each has aliased names for accessing 32, 16, and 8-bit subregisters:

| 64-bit | 32-bit | 16-bit | 8-bit (low) | 8-bit (high) | Convention |
|:-------|:-------|:-------|:------------|:-------------|:-----------|
| `rax` | `eax`  | `ax`   | `al`        | `ah`         | Return value, syscall return |
| `rbx` | `ebx`  | `bx`   | `bl`        | `bh`         | Callee-saved |
| `rcx` | `ecx`  | `cx`   | `cl`        | `ch`         | 4th arg (SysV), 1st arg (Win) |
| `rdx` | `edx`  | `dx`   | `dl`        | `dh`         | 3rd arg (SysV), 2nd arg (Win), high half of mul/div |
| `rsi` | `esi`  | `si`   | `sil`       | —            | 2nd arg (SysV), source for string ops |
| `rdi` | `edi`  | `di`   | `dil`       | —            | 1st arg (SysV), destination for string ops |
| `rbp` | `ebp`  | `bp`   | `bpl`       | —            | Frame pointer (callee-saved) |
| `rsp` | `esp`  | `sp`   | `spl`       | —            | Stack pointer |
| `r8`–`r15` | `r8d`–`r15d` | `r8w`–`r15w` | `r8b`–`r15b` | — | r8/r9 SysV args 5–6; r12–r15 callee-saved |

**Critical zeroing rule:** writes to a 32-bit subregister automatically zero the upper 32 bits.

```nasm
mov rax, 0xDEADBEEF12345678
mov eax, 1                   ; rax now == 0x0000000000000001 — top half zeroed
```

This is exploited heavily for codesize: `xor eax, eax` (2 bytes) is the canonical idiom for `rax = 0`, smaller than `mov rax, 0` (10 bytes) and recognized by every modern CPU as a zero-idiom that doesn't need any execution unit.

In contrast, writes to 16- or 8-bit subregisters **preserve** the upper bits and create false dependencies — avoid `mov al, ...` in hot loops.

### 2.2 RIP and Special Registers

- **`rip`** — instruction pointer. Not directly readable except via `lea` (`lea rax, [rip]`) or by calling and popping the return address.
- **`rflags`** — 64-bit flags register. The interesting low bits:

| Flag | Bit | Meaning | Set By |
|:-----|:----|:--------|:-------|
| CF | 0 | Carry — unsigned overflow | `add`, `sub`, `cmp`, etc. |
| PF | 2 | Parity of low byte | Arithmetic |
| AF | 4 | Auxiliary carry (BCD) | Arithmetic |
| ZF | 6 | Zero — result was zero | `cmp`, `test`, arithmetic |
| SF | 7 | Sign — result MSB | Arithmetic |
| TF | 8 | Trap (single-step) | Debugger |
| IF | 9 | Interrupt enable | `cli`/`sti` |
| DF | 10 | Direction (string ops) | `cld`/`std` |
| OF | 11 | Signed overflow | Arithmetic |

`cmp a, b` computes `a - b` and sets flags but discards the result; `test a, b` does `a & b` likewise. Branches consume these flags:

| Mnemonic | Flag condition | After `cmp a, b` (unsigned/signed) |
|:---------|:---------------|:-----------------------------------|
| `jz`/`je` | ZF=1 | a == b |
| `jnz`/`jne` | ZF=0 | a != b |
| `jb`/`jc`/`jnae` | CF=1 | unsigned a < b |
| `ja`/`jnbe` | CF=0 ∧ ZF=0 | unsigned a > b |
| `jl`/`jnge` | SF≠OF | signed a < b |
| `jg`/`jnle` | ZF=0 ∧ SF=OF | signed a > b |
| `js` | SF=1 | result negative |
| `jo` | OF=1 | signed overflow |

### 2.3 SIMD Register File Evolution

| Extension | Year | Width | Register names | Lanes (typical) |
|:----------|:----:|:-----:|:---------------|:----------------|
| MMX | 1996 | 64 | mm0–mm7 (alias FPU) | 8×i8, 4×i16, 2×i32 |
| SSE | 1999 | 128 | xmm0–xmm7 | 4×f32 |
| SSE2 | 2001 | 128 | xmm0–xmm7 | + 2×f64, integer |
| SSE2 (x64) | 2003 | 128 | xmm0–xmm15 | (extra 8 in long mode) |
| AVX | 2011 | 256 | ymm0–ymm15 | 8×f32, 4×f64 |
| AVX2 | 2013 | 256 | ymm0–ymm15 | integer 256 |
| AVX-512 | 2016 | 512 | zmm0–zmm31 | 16×f32, 8×f64 |
| AVX-512 mask | — | 64 | k0–k7 | predicate registers |

The `xmm`/`ymm`/`zmm` registers alias: `xmm0` is the low 128 bits of `ymm0`, which is the low 256 bits of `zmm0`. Writes to `ymm` zero the upper 256 bits of `zmm` (avoiding the SSE→AVX transition penalty). Pre-AVX SSE writes preserve upper bits — the transition penalty is real.

### 2.4 The Stack

The stack grows **downward** (toward lower addresses). `rsp` always points at the top valid byte:

```
high addr  ┌─────────────────┐
           │ argv strings    │
           │ envp strings    │
           ├─────────────────┤
           │ argv[]          │
           │ envp[]          │  ← rsp at _start
           ├─────────────────┤
           │ caller frame    │
           │   ...           │
           │ return address  │  ← pushed by `call`
           │ saved rbp       │  ← pushed by callee prologue
           │ local vars      │  ← rbp - n
low addr   │ rsp →           │
```

Standard prologue/epilogue:

```nasm
my_func:
    push rbp                 ; save caller's frame pointer
    mov  rbp, rsp            ; establish our frame
    sub  rsp, 32             ; allocate locals (must keep 16-byte align)
    ;; ... body ...
    leave                    ; mov rsp, rbp ; pop rbp
    ret                      ; pop rip
```

With `-fomit-frame-pointer` (default at `-O2` for many compilers) prologue is skipped and rbp becomes a normal GP register.

---

## 3. x86-64 Calling Conventions

### 3.1 System V AMD64 ABI (Linux, macOS, BSD)

Integer/pointer arguments in registers, in this order:

```
arg1  rdi
arg2  rsi
arg3  rdx
arg4  rcx
arg5  r8
arg6  r9
arg7+ stack (right-to-left, 8-byte slots)
```

Floating-point arguments go in `xmm0`–`xmm7`. Variadic functions: the caller sets `al` to the number of `xmm` registers used (so `printf` knows how many vector regs to spill).

Return values:
- 64-bit integer/pointer in `rax`.
- 128-bit (e.g. `__int128`, two-word struct): `rdx:rax`.
- Float/double: `xmm0`; second component in `xmm1`.
- Larger structs: caller passes hidden first arg in `rdi` pointing to caller-allocated buffer; callee writes there and returns the same pointer in `rax`.

Register save discipline:

| Class | Registers | Who owns it |
|:------|:----------|:------------|
| Caller-saved (volatile) | `rax`, `rcx`, `rdx`, `rsi`, `rdi`, `r8`–`r11`, all `xmm`/`ymm` | Caller must save if needed |
| Callee-saved (non-volatile) | `rbx`, `rbp`, `r12`–`r15`, `rsp` | Callee must restore on return |

**Stack alignment:** rsp must be 16-byte aligned **immediately before** a `call` instruction. Since `call` pushes an 8-byte return address, on entry to a function rsp is `(16k + 8)`. So a function that calls another function must adjust rsp so it lands on 16-byte alignment before the next call. The standard prologue (`push rbp; sub rsp, 16k`) does this naturally.

The first 128 bytes below rsp are the **red zone** — leaf functions (those that call no other functions) may use them without adjusting rsp. Signal handlers respect the red zone on Linux. Kernel code is compiled with `-mno-red-zone`.

### 3.2 Microsoft x64 (Windows)

Different choice for the same hardware:

```
arg1  rcx (xmm0 if float)
arg2  rdx (xmm1 if float)
arg3  r8  (xmm2 if float)
arg4  r9  (xmm3 if float)
arg5+ stack
```

Register classes:
- Volatile: `rax`, `rcx`, `rdx`, `r8`–`r11`, `xmm0`–`xmm5`.
- Non-volatile: `rbx`, `rbp`, `rdi`, `rsi`, `rsp`, `r12`–`r15`, `xmm6`–`xmm15`.

**Shadow space:** the caller MUST allocate 32 bytes of stack (4 register slots) before `call`, even if the callee uses the register-arg form. This is a slot the callee can spill the four register args into. Stack alignment is 16-byte before `call` (same as SysV).

There is **no red zone** on Windows.

### 3.3 Why Two Conventions?

History: AMD chose 6 integer registers because Linux compilers wanted more; Microsoft chose 4 to match register pressure observed on its own benchmarks circa 2003. The result is incompatible ABIs. Cross-platform tools that JIT (LLVM, Java HotSpot, JS engines) emit different prologues for each. C compilers handle this transparently; inline asm authors must check `_WIN64` / `__linux__` macros.

---

## 4. ARM64 (AArch64) Architecture

### 4.1 Register File

31 general-purpose 64-bit registers plus a zero register:

| Name | Size | Aliases | Convention (AAPCS64) |
|:-----|:----:|:--------|:---------------------|
| `x0`–`x7` | 64 | `w0`–`w7` (low 32) | Args + return values |
| `x8` | 64 | `w8` | Indirect result location register |
| `x9`–`x15` | 64 | `w9`–`w15` | Caller-saved temporaries |
| `x16`/`ip0` | 64 | — | Intra-procedure scratch (linker veneers) |
| `x17`/`ip1` | 64 | — | Intra-procedure scratch |
| `x18` | 64 | — | Platform register (TLS on some OS, reserved) |
| `x19`–`x28` | 64 | — | Callee-saved |
| `x29`/`fp` | 64 | — | Frame pointer (callee-saved) |
| `x30`/`lr` | 64 | — | Link register (return address) |
| `xzr`/`wzr` | 64/32 | — | Zero register — reads as 0, writes are discarded |
| `sp` | 64 | — | Stack pointer (separate from x-regs) |
| `pc` | 64 | — | Program counter (not a GP reg) |

Crucial: there is no `x31`. The encoding slot `0b11111` means either `xzr/wzr` or `sp` depending on instruction context. The PC is never an operand — branches are explicit.

Writes to `w` registers (low 32 bits) zero the upper 32 — same as x86-64. This is a deliberate ABI choice that avoids partial-register stalls.

### 4.2 Flags Register — NZCV

ARM64 has a `PSTATE` register. The condition flags live in `NZCV`:

| Flag | Bit | Meaning |
|:-----|:---:|:--------|
| N | 31 | Negative — result MSB |
| Z | 30 | Zero — result was zero |
| C | 29 | Carry — unsigned overflow / borrow |
| V | 28 | oVerflow — signed overflow |

Most data-processing instructions do **not** set flags by default. To set flags, use the `s` suffix: `adds`, `subs`, `ands` (note: `cmp` is `subs xzr, x, y` and `tst` is `ands xzr, x, y`).

Conditional execution uses condition codes (the canonical 16):

| Code | Condition | Meaning |
|:-----|:----------|:--------|
| EQ | Z=1 | Equal |
| NE | Z=0 | Not equal |
| CS/HS | C=1 | Carry set / unsigned ≥ |
| CC/LO | C=0 | Carry clear / unsigned < |
| MI | N=1 | Minus |
| PL | N=0 | Plus or zero |
| VS | V=1 | Overflow |
| VC | V=0 | No overflow |
| HI | C=1 ∧ Z=0 | Unsigned > |
| LS | C=0 ∨ Z=1 | Unsigned ≤ |
| GE | N=V | Signed ≥ |
| LT | N≠V | Signed < |
| GT | Z=0 ∧ N=V | Signed > |
| LE | Z=1 ∨ N≠V | Signed ≤ |
| AL | always | Unconditional |
| NV | reserved | Same as AL on AArch64 |

### 4.3 SIMD/FP — NEON and Beyond

32 × 128-bit vector registers `v0`–`v31`. Per-instruction width-and-arrangement specifier follows the register name:

| Specifier | Element size | Lanes | Usage |
|:----------|:-------------|:-----:|:------|
| `.8b`  | 8-bit  | 8  | low 64 bits |
| `.16b` | 8-bit  | 16 | full 128 bits |
| `.4h`  | 16-bit | 4  | low 64 bits (h = halfword) |
| `.8h`  | 16-bit | 8  | full |
| `.2s`  | 32-bit | 2  | low 64 bits (s = single) |
| `.4s`  | 32-bit | 4  | full (4 floats) |
| `.1d`  | 64-bit | 1  | low half |
| `.2d`  | 64-bit | 2  | full (2 doubles) |

Aliases for FP scalars: `s0`–`s31` (32-bit), `d0`–`d31` (64-bit), `h0`–`h31` (16-bit FP16), `b0`–`b31` (8-bit), `q0`–`q31` (128-bit).

```gas
fadd v2.4s, v0.4s, v1.4s    ; 4 lane f32 add
add  v2.16b, v0.16b, v1.16b ; 16 lane u8 add
ld1  {v0.4s, v1.4s}, [x0]   ; load 32 bytes into v0,v1 (4-element struct load)
```

**SVE / SVE2** — Scalable Vector Extension. Register width is implementation-defined from 128 bits up to 2048 in 128-bit increments. Code is **vector-length-agnostic** (VLA): you write loops in terms of vector predicates and the same binary runs on 128-bit, 256-bit, 512-bit hardware. Apple silicon (M1–M3) ships NEON only; AWS Graviton 3 implements SVE 256-bit; Fugaku supercomputer 512-bit.

---

## 5. ARM64 Calling Convention (AAPCS64)

### 5.1 Argument and Return Registers

```
arg1  x0   (return value 1 in x0 on return)
arg2  x1   (return value 2 in x1 on return — for 128-bit returns)
arg3  x2
arg4  x3
arg5  x4
arg6  x5
arg7  x6
arg8  x7
arg9+ stack (16-byte aligned slots)
```

Floats/doubles use `v0`–`v7` analogously.

### 5.2 Indirect Result and Linkage

- `x8` — when the return value is too large to fit in `x0`/`x1` (or returns a HFA/HVA larger than 128 bits), the caller passes a pointer to caller-allocated storage in `x8`; the callee writes the return value through that pointer.
- `x9`–`x15` — caller-saved scratch.
- `x16`/`x17` (a.k.a. `ip0`/`ip1`) — intra-procedure scratch. The linker may overwrite these in PLT veneers, so don't expect them to survive a `bl`.
- `x18` — platform register. iOS/macOS reserve it; Linux leaves it free in user space (but may use it in the kernel for percpu).
- `x19`–`x28` — callee-saved.
- `x29` (`fp`) — frame pointer, callee-saved.
- `x30` (`lr`) — link register: `bl` writes the return address here. A leaf function need not save lr; a non-leaf function must push lr.

### 5.3 Stack Discipline

SP must be 16-byte aligned at all public function boundaries. The standard prologue:

```gas
my_func:
    stp x29, x30, [sp, #-32]!   ; push fp and lr, allocate 32 bytes
    mov x29, sp                 ; new frame pointer
    stp x19, x20, [sp, #16]     ; save callee-saved regs we use
    ;; ... body ...
    ldp x19, x20, [sp, #16]
    ldp x29, x30, [sp], #32     ; pop and deallocate
    ret
```

`stp`/`ldp` (store-pair / load-pair) are encoded for adjacent pair access — saves a cycle versus two separate `str`/`ldr`.

---

## 6. Common x86-64 Instructions

### 6.1 Data movement

```nasm
mov     rax, rbx            ; rax = rbx
mov     rax, [rbx]          ; rax = *rbx (load 8 bytes)
mov     [rbx], rax          ; *rbx = rax
mov     rax, 0xDEADBEEF     ; immediate
movzx   rax, byte [rbx]     ; zero-extend 8→64
movsx   rax, byte [rbx]     ; sign-extend 8→64
movsxd  rax, dword [rbx]    ; sign-extend 32→64
lea     rax, [rbx + rcx*4 + 16]  ; load effective address — pure address math, no memory access
xchg    rax, rbx            ; atomic swap (always implicit-locked when memory operand)
cmpxchg [mem], rcx          ; if (rax == [mem]) [mem] = rcx; else rax = [mem]; (with lock prefix)
```

**`lea` is the workhorse** for non-load arithmetic. `lea rax, [rbx + rbx*2]` computes `rax = 3 * rbx` in one fast instruction.

### 6.2 Arithmetic

```nasm
add  rax, rbx          ; rax += rbx (sets ZF/CF/OF/SF)
sub  rax, rbx          ; rax -= rbx
imul rax, rbx          ; signed rax *= rbx (low 64 bits)
imul rax, rbx, 5       ; rax = rbx * 5 (3-operand)
mul  rbx               ; unsigned: rdx:rax = rax * rbx
div  rbx               ; unsigned: rax = rdx:rax / rbx, rdx = rdx:rax % rbx
idiv rbx               ; signed equivalent. Slow — 20–80 cycles.
neg  rax               ; rax = -rax
inc  rax / dec rax     ; rax += 1 / -= 1 (don't update CF)
adc  rax, rbx          ; rax += rbx + CF (multi-precision add)
sbb  rax, rbx          ; rax -= (rbx + CF)
```

`idiv` is one of the slowest common instructions. Compilers replace `x / const` with multiply-by-magic-constant + shift sequences (`__udiv_qrnnd`-style).

### 6.3 Bitwise

```nasm
and  rax, rbx          ; bitwise AND
or   rax, rbx
xor  rax, rbx          ; xor rax, rax = canonical zero
not  rax               ; bitwise NOT (no flags)
shl  rax, 5            ; logical left shift
shr  rax, 5            ; logical right shift (unsigned)
sar  rax, 5            ; arithmetic right shift (sign-extend)
rol  rax, 5            ; rotate left
ror  rax, 5            ; rotate right
bsf  rax, rbx          ; bit-scan forward — index of lowest set bit
bsr  rax, rbx          ; bit-scan reverse — index of highest set bit
popcnt rax, rbx        ; count set bits (BMI / SSE4.2)
tzcnt  rax, rbx        ; trailing zeros (BMI1)
lzcnt  rax, rbx        ; leading zeros (LZCNT)
```

### 6.4 Stack and Calls

```nasm
push rax               ; rsp -= 8; [rsp] = rax
pop  rax               ; rax = [rsp]; rsp += 8
call func              ; push rip; jmp func
ret                    ; pop rip
ret 16                 ; pop rip; rsp += 16 (callee cleans args — Microsoft __stdcall)
enter 32, 0            ; high-cost variable frame setup; almost never used
leave                  ; mov rsp, rbp; pop rbp
```

### 6.5 Control flow

```nasm
jmp  label             ; unconditional
jmp  rax               ; indirect jump
jmp  [rbx + rcx*8]     ; jump table
jz   label             ; jump if ZF=1
jne  label             ; jump if ZF=0
loop label             ; rcx--; if rcx != 0 jump (avoid — slow)
cmovz rax, rbx         ; conditional move if ZF=1 (branchless)
setz al                ; al = (ZF == 1) ? 1 : 0 (clear upper bits separately)
```

### 6.6 String operations (rep prefixes)

```nasm
rep movsb              ; copy [rsi]→[rdi] rcx bytes (depends on DF)
rep stosb              ; fill [rdi] with al, rcx times
rep cmpsb              ; compare [rsi] vs [rdi] until rcx=0 or mismatch
repnz scasb            ; scan [rdi] for al until rcx=0 or match
```

On modern Intel CPUs (Ivy Bridge+), `rep movsb` is often the fastest memcpy for medium sizes due to "Fast Short REP MOV" (FSRM) and "Enhanced REP MOVSB" (ERMSB).

### 6.7 Atomics

```nasm
lock add [mem], rax    ; atomic add, full barrier
lock cmpxchg [mem], rcx ; atomic compare-and-swap — basis of mutexes
xadd rax, [mem]        ; (with lock) atomic fetch-and-add
mfence                 ; memory fence — load+store
lfence / sfence        ; load/store-only fences
```

---

## 7. Common ARM64 Instructions

### 7.1 Move and immediate construction

```gas
mov  x0, x1                 ; x0 = x1
mov  x0, #42                ; small immediate (encoded, ≤ 16 bits with shift)
movz x0, #0xBEEF, lsl #16   ; clear x0, then x0[31:16] = 0xBEEF
movk x0, #0xDEAD, lsl #48   ; keep, then x0[63:48] = 0xDEAD
;; large immediate construction
movz x0, #0x5678
movk x0, #0x1234, lsl #16
movk x0, #0xBEEF, lsl #32
movk x0, #0xDEAD, lsl #48   ; x0 = 0xDEADBEEF12345678 — 4 instructions
```

For constants that fit a 12-bit immediate or a "logical immediate" pattern, single-instruction encoding works. Otherwise, big constants come from a literal pool: `ldr x0, =0xDEADBEEF12345678`.

### 7.2 Loads and stores

```gas
ldr  x0, [x1]              ; x0 = *x1, load 8 bytes
ldr  w0, [x1]              ; w0 = *x1, load 4 bytes (zero-extends to x0)
ldrb w0, [x1]              ; load byte, zero-extend
ldrh w0, [x1]              ; load halfword (16 bits), zero-extend
ldrsb x0, [x1]             ; load signed byte, sign-extend to 64
str  x0, [x1]              ; *x1 = x0, store 8 bytes
strb w0, [x1]              ; store low byte
ldp  x0, x1, [x2]          ; load pair (x0 from x2, x1 from x2+8)
stp  x0, x1, [x2, #-16]!   ; store pair with pre-index, push two regs
```

Pair load/store doubles bandwidth on most cores.

### 7.3 Arithmetic

```gas
add  x0, x1, x2            ; x0 = x1 + x2
add  x0, x1, #16           ; immediate
add  x0, x1, x2, lsl #2    ; x0 = x1 + (x2 << 2)
adds x0, x1, x2            ; set flags
sub  x0, x1, x2
mul  x0, x1, x2            ; low 64 bits of signed multiply
madd x0, x1, x2, x3        ; x0 = x3 + x1*x2 (multiply-add fused)
msub x0, x1, x2, x3        ; x0 = x3 - x1*x2
sdiv x0, x1, x2            ; signed divide
udiv x0, x1, x2            ; unsigned divide
;; remainder: sdiv + msub
sdiv x3, x1, x2
msub x0, x3, x2, x1        ; x0 = x1 - (x3 * x2) = x1 % x2
neg  x0, x1                ; x0 = -x1
```

There is no carry-style `adc`/`sbb` for plain GP — multi-precision uses `adcs`/`sbcs` if you really need it, but it's rare on a 64-bit ISA.

### 7.4 Bitwise

```gas
and  x0, x1, x2            ; x0 = x1 & x2
orr  x0, x1, x2            ; OR
eor  x0, x1, x2            ; XOR
mvn  x0, x1                ; bitwise NOT (move-not)
lsl  x0, x1, #5            ; logical shift left
lsr  x0, x1, #5            ; logical shift right
asr  x0, x1, #5            ; arithmetic shift right
ror  x0, x1, #5            ; rotate right
clz  x0, x1                ; count leading zeros
rbit x0, x1                ; reverse bit order — pair with clz for ctz
rev  x0, x1                ; reverse byte order — htobe64
```

### 7.5 Branches and conditional select

```gas
b    label                 ; unconditional branch (±128 MiB)
b.eq label                 ; conditional branch on EQ
bl   label                 ; branch with link — sets x30=PC+4
blr  x16                   ; branch with link to register — indirect call
br   x16                   ; indirect branch
ret                        ; return — implicit `br x30`
ret  x0                    ; unusual; ret from arbitrary register

cbz  x0, label             ; compare-and-branch-if-zero
cbnz x0, label             ; compare-and-branch-if-nonzero
tbz  x0, #5, label         ; test-bit-and-branch-if-zero (bit 5)
tbnz x0, #5, label         ; test-bit-and-branch-if-nonzero
```

`cbz`/`cbnz` and `tbz`/`tbnz` are RISC-style fused compare+branch — they don't go through NZCV.

`csel`, `csinc`, `csinv`, `csneg` provide branchless conditional selection:

```gas
cmp  x0, x1
csel x2, x3, x4, lt        ; x2 = (x0 < x1) ? x3 : x4
csinc x2, x3, xzr, eq      ; x2 = (eq) ? x3 : 0 + 1 — useful for boolean
```

### 7.6 Atomics

ARMv8.0 used Load-Linked/Store-Conditional (LL/SC):

```gas
1: ldaxr x0, [x1]          ; load-acquire-exclusive
   add   x0, x0, #1
   stlxr w2, x0, [x1]      ; store-release-exclusive, w2=0 on success
   cbnz  w2, 1b            ; retry on failure
```

ARMv8.1 added LSE (Large System Extensions) — single-instruction atomics:

```gas
ldaddal x0, x1, [x2]       ; atomic add, acquire+release
casal   x0, x1, [x2]       ; compare-and-swap, acquire+release
swpal   x0, x1, [x2]       ; atomic swap
```

LSE makes a measurable difference on contended atomics (a 2× to 10× improvement is typical on Graviton 3 vs Graviton 2).

---

## 8. Memory Model

The **memory model** specifies what one core sees of another core's writes. This is *not* "what the silicon does" — it's what the ISA *promises*. Compilers and CPUs may reorder freely as long as observation conforms.

### 8.1 Sequential Consistency (SC)

The strongest model. Every operation appears to execute in some global total order consistent with each thread's program order. No real CPU implements full SC; programmers reach SC via synchronization primitives.

### 8.2 x86-64 — TSO (Total Store Order)

x86-64 is **TSO**: every core has a store buffer; loads can move forward of older stores **to different addresses**. Other cores see all stores in a single total order, but a core can observe its own pending store before the world does.

What this means concretely:

| Reordering | Allowed on x86 TSO? |
|:-----------|:-------------------:|
| Load–Load reordering | No |
| Load–Store reordering | No |
| Store–Store reordering | No |
| Store–Load reordering (different addr) | **Yes** |
| Load forwarding from own store buffer | Yes |

The classic "Dekker" / "store buffering" example exhibits the one allowed reorder:

```
  Thread A          Thread B
  store r1=1        store r2=1
  load  r2          load  r1
```

On x86, both loads can return 0 — each thread's store may sit in its own store buffer when the load completes. To prevent, insert `mfence` between store and load.

x86 atomic ops (`lock`-prefixed) imply a full barrier. Plain `mov` to a naturally-aligned address is atomic. Volatile `mov` does NOT imply a barrier.

### 8.3 ARM64 — Weak Memory

ARM64 allows essentially **all** reorderings unless you use ordering instructions:

| Reordering | Allowed on ARM64? |
|:-----------|:-----------------:|
| Load–Load | Yes |
| Load–Store | Yes |
| Store–Store | Yes |
| Store–Load | Yes |
| Load forwarding | Yes |

Ordering primitives:

```gas
dmb sy        ; data memory barrier — full system, full
dmb ish       ; inner shareable, full
dmb ishld     ; inner shareable, load-only
dmb ishst     ; inner shareable, store-only
dsb ish       ; data sync barrier — stronger; waits for completion
isb           ; instruction sync barrier — flushes pipeline
```

Acquire/release semantics are baked into specific instructions:

```gas
ldar  x0, [x1]    ; load-acquire — no later op can pass it
stlr  x0, [x1]    ; store-release — no earlier op can pass it
ldapr x0, [x1]    ; load-acquire-release-consistent (ARMv8.3 RCpc)
```

### 8.4 Mapping C++/Rust Atomics to Silicon

| C++/Rust order | x86-64 emit | ARM64 emit |
|:---------------|:------------|:-----------|
| `relaxed` | plain `mov` | `ldr` / `str` |
| `acquire` (load) | `mov` (free) | `ldar` |
| `release` (store) | `mov` (free) | `stlr` |
| `acq_rel` (RMW) | `lock cmpxchg` | `casal` |
| `seq_cst` (load) | `mov` | `ldar` |
| `seq_cst` (store) | `mov; mfence` or `xchg` | `stlr` then `dmb ish` (or `casal`) |

So `seq_cst` is "almost free" on x86 but costs a barrier on ARM. This explains the ~2× cost of unguarded `Arc::clone` on Apple Silicon vs Intel.

---

## 9. SIMD and Vectorization

### 9.1 x86 SSE → AVX → AVX-512

Width progression:

```
SSE     128 bit   4×f32
AVX     256 bit   8×f32
AVX-512 512 bit  16×f32 (and 32 zmm regs, plus mask k0–k7)
```

AVX-512 also adds **mask registers** (`k0`–`k7`) — predicate registers that gate per-lane writes:

```nasm
vmovdqu32 zmm1, [rdi]
vptestmd  k1, zmm1, zmm1            ; k1[i] = (zmm1[i] != 0)
vmovdqu32 [rdi]{k1}, zmm2           ; only write lanes where k1 is 1
```

Scaling caveat: pre-Ice Lake, executing AVX-512 lowered the "vector-license" frequency of the core — peak GHz dropped 100–300 MHz. Modern Intel cores (Sapphire Rapids) and all AMD Zen 4+ cores do AVX-512 without throttling.

Floating-point fused multiply-add:

```nasm
vfmadd231ps zmm0, zmm1, zmm2        ; zmm0 += zmm1 * zmm2 (FMA3)
```

FMA halves the latency and roundoff error of a multiply-then-add pair.

### 9.2 ARM NEON → SVE / SVE2

NEON is fixed 128-bit. SVE is **vector-length-agnostic** — write the loop once, run on 128 to 2048-bit hardware.

Canonical SVE loop pattern:

```gas
;; sum N floats from x0
mov  x1, #0                 ; index i
mov  z0.s, #0               ; vector accumulator
whilelt p0.s, x1, x2        ; predicate p0 = lanes where i+lane < N
1:
   ld1w  z1.s, p0/z, [x0, x1, lsl #2]   ; load with predicate (zeroing)
   fadd  z0.s, p0/m, z0.s, z1.s         ; add into accumulator (merging)
   incw  x1                              ; i += vector length (in 32-bit elements)
   whilelt p0.s, x1, x2                  ; refresh predicate; sets flags
   b.first 1b                            ; branch if any lane true
;; horizontal reduce
faddv s0, p0, z0.s
```

The same code runs on a 128-bit chip (4 lanes), a 256-bit chip (8 lanes), or a 512-bit chip (16 lanes). The hardware "tells" you the vector length via `cntw` (count words).

### 9.3 Autovectorization vs Intrinsics

Compilers vectorize when:
- Loop trip count is bounded or has a runtime guard.
- No data dependences between iterations (or recognizable reductions).
- Memory access pattern is unit-stride (otherwise gather/scatter is slow).
- Pointers don't alias (use `restrict` in C, `&mut` exclusivity in Rust).

When autovectorization fails, hand-written intrinsics are the next step:

```c
#include <immintrin.h>
void add_arrays(float *a, float *b, float *c, size_t n) {
    size_t i;
    for (i = 0; i + 8 <= n; i += 8) {
        __m256 va = _mm256_loadu_ps(a + i);
        __m256 vb = _mm256_loadu_ps(b + i);
        _mm256_storeu_ps(c + i, _mm256_add_ps(va, vb));
    }
    for (; i < n; i++) c[i] = a[i] + b[i];     // scalar tail
}
```

For ARM NEON:

```c
#include <arm_neon.h>
void add_arrays(float *a, float *b, float *c, size_t n) {
    size_t i;
    for (i = 0; i + 4 <= n; i += 4) {
        float32x4_t va = vld1q_f32(a + i);
        float32x4_t vb = vld1q_f32(b + i);
        vst1q_f32(c + i, vaddq_f32(va, vb));
    }
    for (; i < n; i++) c[i] = a[i] + b[i];
}
```

---

## 10. Inline Assembly

### 10.1 GCC Extended Asm

Syntax (AT&T mode by default — operand order is `src, dst`):

```c
__asm__ volatile (
    "asm-template"
    : output operands
    : input operands
    : clobbers
);
```

Constraint letters (most common):

| Letter | Meaning |
|:------:|:--------|
| `r` | Any general-purpose register |
| `m` | Memory operand |
| `i` | Immediate (compile-time constant) |
| `=` | Output operand (write-only) |
| `+` | Output that's also read |
| `&` | Early-clobber — written before all inputs read |
| `g` | Any (register, memory, or immediate) |

x86-specific letters: `a` (rax), `b` (rbx), `c` (rcx), `d` (rdx), `S` (rsi), `D` (rdi).

Example — inline `rdtsc`:

```c
static inline uint64_t rdtsc(void) {
    uint32_t lo, hi;
    __asm__ volatile ("rdtsc" : "=a"(lo), "=d"(hi));
    return ((uint64_t)hi << 32) | lo;
}
```

Example — atomic increment via `lock xadd`:

```c
static inline uint32_t atomic_inc(uint32_t *p) {
    uint32_t v = 1;
    __asm__ volatile ("lock xaddl %0, %1"
                      : "+r"(v), "+m"(*p)
                      :
                      : "memory");
    return v;  // value before increment
}
```

The `"memory"` clobber tells the compiler that *any* memory may have been read or written — equivalent to a sequence point for the optimizer.

### 10.2 Intel Syntax in GCC

Use `-masm=intel` or wrap with `.intel_syntax noprefix` / `.att_syntax`. Operand order flips to `dst, src`.

### 10.3 Rust `asm!`

Rust's `core::arch::asm!` (stable since 1.59) uses Intel syntax on x86 and a more readable template syntax:

```rust
use core::arch::asm;

fn rdtsc() -> u64 {
    let lo: u32; let hi: u32;
    unsafe {
        asm!("rdtsc",
             out("eax") lo,
             out("edx") hi,
             options(nomem, nostack, preserves_flags));
    }
    ((hi as u64) << 32) | (lo as u64)
}
```

Operand specifiers: `in("reg") val`, `out("reg") var`, `inout("reg") var`, `lateout("reg") var` (for early-clobber-free outputs), and named operands like `{tmp}` with `tmp = out(reg) _`.

`options()`:
- `nomem` — does not touch memory (else compiler assumes worst case).
- `nostack` — does not push/pop the stack (so red-zone-ok).
- `preserves_flags` — does not modify NZCV / RFLAGS.
- `pure` — same output for same inputs (allows CSE).
- `noreturn` — control flow does not return.
- `att_syntax` — switch to AT&T on x86.

### 10.4 Common Pitfalls

- Forgetting the `"memory"` clobber on a barrier — the compiler reorders memory ops across the asm.
- Forgetting `volatile` on inline asm with side effects — compiler may delete it.
- Outputs marked `=r` instead of `+r` when the value is also read — undefined behavior.
- Using `eax` directly without the constraint, then having the compiler put a value there.
- On ARM, forgetting that `x16`/`x17` are scratch for linker veneers — clobber list must include them across calls.

---

## 11. Calling C from Assembly and Back

### 11.1 Declaring an Extern Function

In assembly, declare the C function as external:

```nasm
;; x86-64 NASM syntax
section .text
extern  printf
global  main
main:
    sub  rsp, 8                 ; align to 16 (return addr already pushed)
    lea  rdi, [rel fmt]         ; arg1: format string, RIP-relative
    mov  esi, 42                ; arg2: integer
    xor  eax, eax               ; 0 vector regs used (variadic)
    call printf wrt ..plt       ; PLT for shared lib
    add  rsp, 8
    xor  eax, eax
    ret
section .rodata
fmt: db "answer = %d", 10, 0
```

ARM64 GAS:

```gas
.text
.extern printf
.global main
main:
    stp  x29, x30, [sp, #-16]!
    mov  x29, sp
    adrp x0, fmt
    add  x0, x0, :lo12:fmt
    mov  w1, #42
    bl   printf
    mov  w0, #0
    ldp  x29, x30, [sp], #16
    ret
.section .rodata
fmt: .asciz "answer = %d\n"
```

### 11.2 GOT and PLT (Position-Independent Code)

When linked into a shared library or PIE binary, calls to libc functions go through:

- **PLT (Procedure Linkage Table)** — small per-function trampoline with an indirect jump.
- **GOT (Global Offset Table)** — table of pointers, lazy-bound by the dynamic linker.

```
   call printf@plt
   ;; first call: trampoline jumps to dynamic linker, which patches GOT entry
   ;; subsequent calls: trampoline reads GOT entry, jumps direct
```

**RIP-relative addressing on x86-64** lets data references be PIC at zero cost (no GOT for data in the same DSO):

```nasm
mov rax, [rel my_data]      ; rip-relative load
```

For data in another DSO, you go through the GOT:

```nasm
mov rax, [rel my_data wrt ..gotpcrel]   ; load address from GOT
mov rbx, [rax]                          ; load data
```

ARM64 uses `adrp + add`:

```gas
adrp x0, my_data            ; page address
add  x0, x0, :lo12:my_data  ; exact address (same DSO)
;; or via GOT for cross-DSO:
adrp x0, :got:my_data
ldr  x0, [x0, :got_lo12:my_data]
```

### 11.3 Linker Switches

Common scenarios:

```bash
# absolute (non-PIC) — only for ET_EXEC, not allowed for PIE
gcc -no-pie main.c

# position-independent executable (default modern Linux)
gcc -pie -fPIE main.c

# shared library (always PIC)
gcc -shared -fPIC libfoo.c -o libfoo.so

# inspect relocations and PIE bit
readelf -d ./a.out | grep -E "(PIE|FLAGS)"
checksec --file=./a.out
```

### 11.4 ABI for Static-Local Calls (Same TU)

If a function is `static` and the compiler keeps it in-module, the ABI degrades to "whatever both ends know" — register allocation can be different, no callee-saves, etc. This is why LTO and inlining matter for performance, and why `__attribute__((noinline))` can ruin micro-benchmarks.

---

## 12. Performance Counters

Modern CPUs expose **hardware performance counters** (PMCs) via Model-Specific Registers (MSRs). On Linux, the `perf_event_open(2)` syscall and the `perf` tool wrap them:

```bash
perf stat -e cycles,instructions,cache-misses,branch-misses ./prog
perf stat -e L1-dcache-loads,L1-dcache-load-misses,LLC-loads,LLC-load-misses ./prog
perf record -e cycles -g ./prog && perf report
```

### 12.1 Derived Metrics

```
IPC      = instructions / cycles
CPI      = cycles / instructions     (= 1/IPC)
MPKI     = misses_per_kilo_instr = (cache_misses / instructions) * 1000
branch_miss_rate = branch_misses / branches
```

Targets on a healthy compute-bound workload:
- IPC ≥ 2.0 (modern wide cores reach 4–6 IPC peak)
- branch miss rate < 1 %
- L1d miss rate < 5 %
- LLC miss rate < 30 % of L1 misses (i.e. most fall through L2)

### 12.2 Top-Down Analysis

Intel's "top-down" methodology classifies each cycle into:

```
      ┌─ Frontend Bound  (decoder/uop cache stall)
      │
Slot ─┼─ Backend Bound   (execution port busy or memory stall)
      │
      ├─ Bad Speculation (branch mispredict, machine clears)
      │
      └─ Retiring        (productive work)
```

On a 4-wide retire core, the goal is `Retiring` near 75–95 %. `perf stat -M TopdownL1 ./prog` produces this breakdown directly.

---

## 13. Cache Hierarchy and Timing

### 13.1 Typical Latencies (Skylake-class, ARM Neoverse-N1 similar)

| Level | Size | Latency (cycles) | Latency (ns @ 3 GHz) |
|:------|:----:|:----------------:|:--------------------:|
| L1d | 32–48 KiB | 4–5 | 1.3–1.7 ns |
| L1i | 32 KiB | 4 | 1.3 ns |
| L2 (private) | 256 KiB – 1.25 MiB | 12–14 | 4–4.7 ns |
| L3 (shared) | 1–4 MiB/core | 35–50 | 12–17 ns |
| DRAM (local) | GiB | 200–300 | 65–100 ns |
| DRAM (remote NUMA) | GiB | 350–500 | 115–165 ns |
| NVMe SSD (4K random) | TiB | — | 30–80 μs (~1e5 cycles) |

### 13.2 Cycle Time

```
cycle_time_ps = 1000 / freq_GHz
3 GHz => 333 ps/cycle
4 GHz => 250 ps/cycle
5 GHz => 200 ps/cycle
```

So a DRAM access at 200 cycles on a 3 GHz core costs about **65 ns**. In that time, the core could have retired ≈ 1000 instructions (at 5 IPC). This is why **memory is the bottleneck** for almost every real workload.

### 13.3 Cache Lines and False Sharing

Cache line = 64 bytes on x86-64 and most ARM64. Two threads writing to different variables on the same cache line force ping-pong through the coherence protocol (MESI / MOESI), nuking IPC. Solution: pad with `alignas(64)` or `[[gnu::aligned(64)]]`, or `#[repr(align(64))]` in Rust.

### 13.4 Prefetching

Hardware prefetchers track stride and stream patterns. For irregular access, manual prefetch:

```nasm
prefetcht0 [rax + 256]      ; into all caches (T0 = closest)
prefetchnta [rax + 256]     ; non-temporal — hint "don't pollute caches"
```

ARM64:

```gas
prfm pldl1keep, [x0, #256]      ; preload, L1, keep
prfm pstl2strm, [x0, #256]      ; preload for store, L2, streaming
```

Effective stride is 1–4 cache lines ahead at typical core throughput; further out and the prefetched line gets evicted before use.

---

## 14. Worked Examples

### 14.1 x86-64 Fibonacci with Stack Frame

```nasm
;; uint64_t fib(uint64_t n)
;; arg in rdi, return in rax
;; n < 2 => n; else fib(n-1) + fib(n-2)
section .text
global fib
fib:
    cmp     rdi, 2
    jl      .base
    push    rbx                 ; callee-saved scratch
    push    rdi                 ; align stack (push of rbx un-aligned us)
    mov     rbx, rdi
    dec     rdi
    call    fib                 ; fib(n-1) -> rax
    mov     rcx, rax             ; save
    mov     rdi, rbx
    sub     rdi, 2
    call    fib                 ; fib(n-2) -> rax
    add     rax, rcx
    pop     rdi                 ; restore alignment
    pop     rbx
    ret
.base:
    mov     rax, rdi
    ret
```

This is an instructional example — the recursion is exponential. Real implementations memoize or use the closed-form O(log n) matrix-power formula:

$$\begin{pmatrix} F_{n+1} \\ F_n \end{pmatrix} = \begin{pmatrix} 1 & 1 \\ 1 & 0 \end{pmatrix}^n \begin{pmatrix} 1 \\ 0 \end{pmatrix}$$

### 14.2 ARM64 strlen (NEON)

The classical scalar strlen has 1 byte/cycle ceiling. NEON-vectorized strlen uses `cmeq` + `umaxv` to find the first zero byte in 16-byte chunks:

```gas
;; size_t my_strlen(const char *s)
;; arg in x0, return in x0
.global my_strlen
my_strlen:
    mov     x1, x0                  ; save start
    bic     x2, x0, #15             ; align down to 16 bytes
    sub     x3, x0, x2              ; offset within first chunk
    movi    v0.16b, #0              ; comparison constant 0
1:
    ld1     {v1.16b}, [x2], #16     ; load 16 bytes, post-inc
    cmeq    v2.16b, v1.16b, v0.16b  ; v2[i] = (v1[i] == 0) ? 0xFF : 0
    umaxv   b3, v2.16b              ; horizontal max byte
    fmov    w4, s3                  ; into GP reg
    cbz     w4, 1b                  ; no zero seen, loop
    ;; found a zero somewhere in v1; locate it
    ;; trick: compress mask into 64-bit lane mask via SHRN
    shrn    v2.8b, v2.8h, #4
    fmov    x4, d2
    rbit    x4, x4                  ; reverse so first match is LSB
    clz     x4, x4                  ; bits to first set
    lsr     x4, x4, #2              ; nibble => byte
    sub     x2, x2, #16             ; rewind to start of this chunk
    add     x0, x2, x4              ; first-zero address
    sub     x0, x0, x1              ; -> length
    ret
```

This is the structure used in glibc/musl and Apple's libSystem. Throughput is ~16 bytes/cycle on Apple M1 vs 1 byte/cycle for the naive byte-by-byte loop.

### 14.3 x86-64 Linux syscall (write to stdout)

The Linux x86-64 syscall ABI uses the `syscall` instruction:

```
syscall number   -> rax
arg1             -> rdi
arg2             -> rsi
arg3             -> rdx
arg4             -> r10  (NOT rcx — rcx is clobbered by syscall)
arg5             -> r8
arg6             -> r9
return           -> rax  (negative errno on failure)
clobbers         -> rcx (return-RIP), r11 (RFLAGS)
```

Hello world without libc:

```nasm
section .text
global  _start
_start:
    mov  rax, 1                  ; sys_write
    mov  rdi, 1                  ; fd = stdout
    lea  rsi, [rel msg]
    mov  rdx, msg_len
    syscall

    mov  rax, 60                 ; sys_exit
    xor  rdi, rdi                ; status = 0
    syscall

section .rodata
msg:     db "hello, world", 10
msg_len: equ $ - msg
```

Build:
```bash
nasm -felf64 hello.s -o hello.o
ld hello.o -o hello
./hello
```

### 14.4 ARM64 Linux syscall (`svc #0`)

```
syscall number   -> x8
arg1             -> x0
...
arg6             -> x5
return           -> x0  (negative errno on failure)
```

```gas
.text
.global _start
_start:
    mov  x8, #64                  ; sys_write
    mov  x0, #1                   ; stdout
    adrp x1, msg
    add  x1, x1, :lo12:msg
    mov  x2, #13
    svc  #0

    mov  x8, #93                  ; sys_exit
    mov  x0, #0
    svc  #0

.section .rodata
msg: .ascii "hello, world\n"
```

Linux syscall numbers are arch-specific — `cat /usr/include/asm-generic/unistd.h` for ARM64; on x86-64 see `arch/x86/entry/syscalls/syscall_64.tbl`.

### 14.5 CPUID Feature Detection (x86)

```c
#include <stdint.h>
#include <stdio.h>

static inline void cpuid(uint32_t leaf, uint32_t subleaf,
                         uint32_t *a, uint32_t *b, uint32_t *c, uint32_t *d) {
    __asm__ volatile (
        "cpuid"
        : "=a"(*a), "=b"(*b), "=c"(*c), "=d"(*d)
        : "a"(leaf), "c"(subleaf)
    );
}

int main(void) {
    uint32_t a, b, c, d;
    cpuid(1, 0, &a, &b, &c, &d);
    int sse42 = (c >> 20) & 1;
    int avx   = (c >> 28) & 1;
    cpuid(7, 0, &a, &b, &c, &d);
    int avx2     = (b >> 5)  & 1;
    int avx512f  = (b >> 16) & 1;
    int bmi1     = (b >> 3)  & 1;
    int bmi2     = (b >> 8)  & 1;
    printf("sse4.2=%d avx=%d avx2=%d avx512f=%d bmi1=%d bmi2=%d\n",
           sse42, avx, avx2, avx512f, bmi1, bmi2);
}
```

ARM64 has no exact CPUID equivalent. Feature detection on Linux uses `getauxval(AT_HWCAP)` / `AT_HWCAP2` or reads `/proc/cpuinfo`.

### 14.6 rdtsc for Cycle-Accurate Timing

```c
#include <stdint.h>

static inline uint64_t rdtsc_serialized(void) {
    uint32_t a, d;
    __asm__ volatile (
        "lfence\n\t"
        "rdtsc\n\t"
        : "=a"(a), "=d"(d)
        :
        : "memory"
    );
    return ((uint64_t)d << 32) | a;
}

static inline uint64_t rdtscp_end(void) {
    uint32_t a, d, aux;
    __asm__ volatile (
        "rdtscp\n\t"
        "lfence\n\t"
        : "=a"(a), "=d"(d), "=c"(aux)
        :
        : "memory"
    );
    return ((uint64_t)d << 32) | a;
}
```

Use `rdtsc_serialized` before the region under test and `rdtscp_end` after — `rdtscp` itself drains the pipeline and `lfence` after blocks reordering.

ARM64 equivalent uses the **virtual counter**:

```gas
mrs x0, cntvct_el0
```

The frequency is in `cntfrq_el0` (typically 24 MHz on Apple silicon, 1 GHz on AWS Graviton). To convert to ns:

$$t_{ns} = \frac{\text{cntvct delta}}{\text{cntfrq}} \cdot 10^9$$

### 14.7 Inline asm Spinlock (x86-64)

```c
typedef struct { volatile int locked; } spinlock_t;

static inline void spin_lock(spinlock_t *lock) {
    __asm__ volatile (
        "1: movl $1, %%eax\n"
        "   xchgl %%eax, %0\n"
        "   testl %%eax, %%eax\n"
        "   jz 3f\n"
        "2: pause\n"
        "   movl %0, %%eax\n"
        "   testl %%eax, %%eax\n"
        "   jnz 2b\n"
        "   jmp 1b\n"
        "3:\n"
        : "+m"(lock->locked)
        :
        : "eax", "memory"
    );
}

static inline void spin_unlock(spinlock_t *lock) {
    __asm__ volatile ("movl $0, %0" : "+m"(lock->locked) : : "memory");
}
```

The `pause` instruction is a hint to the CPU that this is a spin loop — it reduces power and avoids memory-order violations on hyperthreads.

ARM64 equivalent uses `wfe` (wait for event) for power-efficient spinning, paired with `sevl` (send-event-local) on the unlock side.

---

## 15. Reverse Engineering Basics

### 15.1 Disassembling with objdump

```bash
# disassemble all sections, with source interleaving (if -g)
objdump -d -S ./binary

# only the main function, demangle C++
objdump -d --disassemble=main --demangle ./binary

# display in Intel syntax (default is AT&T on Linux)
objdump -M intel -d ./binary

# show relocations and symbols
objdump -r ./binary.o
objdump -t ./binary
objdump -T ./binary       # dynamic symbols
```

ARM64 specifics:
```bash
objdump -d ./binary       # GAS syntax, instruction-level
```

### 15.2 GDB Decoding

```
(gdb) disas /m main             # mixed source+asm
(gdb) disas /r main             # show raw bytes
(gdb) layout asm                # TUI assembly view
(gdb) info registers
(gdb) info registers ymm0
(gdb) x/16wx $rsp               # dump 16 words at rsp in hex
(gdb) x/16i $rip                # 16 instructions at rip
(gdb) si                        ; step one instruction
(gdb) ni                        ; next instruction (over calls)
(gdb) display/i $pc             ; show current insn at every break
(gdb) p/x $rax                  ; print rax in hex
```

### 15.3 Higher-Level RE Tools

| Tool | Cost | Strengths |
|:-----|:----:|:----------|
| Ghidra | Free (NSA) | Decompiler, scripting (Java/Python), x86/ARM/MIPS/PowerPC |
| IDA Pro | $$ | Industry-standard decompiler (Hex-Rays), best UX |
| radare2 / Cutter | Free | CLI-first / Qt GUI, scriptable |
| Binary Ninja | $ | Modern UI, IL-based analysis |
| pwntools | Free | Python lib for exploit dev |

Workflow: load binary → run autoanalysis → look at strings (`strings -tx ./bin | less`) and symbol table → find `main` → follow calls → annotate types → use the decompiler.

### 15.4 Static vs Dynamic Analysis

- **Static** — read the binary without running it. Safer (no execution), but obfuscation, packing, and runtime decryption defeat it.
- **Dynamic** — run under a debugger or sandbox. Reveals actual behavior including self-modifying code, but anti-debug tricks (timing checks, `ptrace` self-attach) can detect you.

Common anti-RE techniques and counters:

| Technique | Counter |
|:----------|:--------|
| `IsDebuggerPresent` / PEB BeingDebugged | Patch the check |
| `rdtsc` timing detection | `gdb` script that fakes increments |
| Self-modifying code | Set hardware execute breakpoints; trace |
| Control-flow obfuscation | Decompile, run through formal verifier |
| Packing (UPX, custom) | Run unpacker; dump from memory |

---

## 16. Toolchain Quick Reference

### 16.1 Assemblers

| Tool | Syntax | Target | Notes |
|:-----|:-------|:-------|:------|
| `gas` (GNU as) | AT&T (default) or Intel | x86, x86-64, ARM, ARM64, RISC-V | Used by gcc; `.s` files |
| `nasm` | Intel-style (custom) | x86, x86-64 | Popular for standalone asm |
| `yasm` | NASM-compatible | x86, x86-64 | Drop-in replacement |
| `armasm` | ARM proprietary | ARM/ARM64 | Used by Arm Compiler 6 |
| `clang -integrated-as` | GAS or Intel | All LLVM targets | Default in modern Clang |

### 16.2 Compile + Assemble + Link

```bash
# C → asm → object → executable, separately
gcc -O2 -S hello.c -o hello.s          # emit asm only
gcc -c hello.s -o hello.o              # assemble only
gcc hello.o -o hello                   # link

# Inspect generated asm with debug + Intel syntax
gcc -O2 -S -masm=intel -fverbose-asm hello.c

# Disassemble a relocatable object
objdump -d -M intel hello.o

# Cross-compile for ARM64 from x86 host
aarch64-linux-gnu-gcc -O2 hello.c -o hello-arm
qemu-aarch64 -L /usr/aarch64-linux-gnu ./hello-arm

# Compile with explicit ISA features
gcc -O2 -mavx2 hello.c                  # require AVX2
gcc -O2 -march=native hello.c           # whatever this host has
gcc -O2 -march=armv8.2-a+crypto+fp16    # ARM64 features
```

### 16.3 Symbol and ABI Inspection

```bash
nm ./binary                    # symbol table (T=text, D=data, U=undefined)
readelf -a ./binary            # everything: header, sections, segments, syms, relocs, dynamic
readelf -d ./binary            # dynamic section (NEEDED libs, RPATH, FLAGS)
file ./binary                  # arch, bitness, endianness, dynamic/static
size ./binary                  # text/data/bss sizes
strings ./binary               # printable ASCII strings
addr2line -e ./binary 0x401530 # map address to source:line (needs -g)
c++filt _ZN3fooEi              # demangle C++
ldd ./binary                   # required shared libraries (Linux)
otool -L ./binary              # required libs (macOS)
otool -tv ./binary             # disassemble (macOS)
```

### 16.4 Compiler Explorer

The single best learning aid is **godbolt.org** (Compiler Explorer): paste C/C++/Rust code, see the asm output of any compiler, any target, any optimization level, side-by-side. Use the `-fverbose-asm` flag for variable-name annotations on GCC output.

---

## 17. Common Errors and Gotchas

### 17.1 Stack Misalignment Crash on a `call printf`

Symptom: program SEGVs inside libc on entry to a function that uses SSE.

Cause: SysV ABI requires `rsp ≡ 8 (mod 16)` *on entry* to a function (i.e. 16-byte aligned right before the `call`). If you `push` an odd number of registers without re-aligning, libc's SSE store of `xmm0` to a `[rsp - 24]` location segfaults.

Fix:

```nasm
push rbp           ; +8 -> aligned
sub  rsp, 8        ; or push another reg / scratch, total slip 16
call printf
add  rsp, 8
pop  rbp
ret
```

### 17.2 ARM64 "unaligned access" on a 16-byte SIMD load

Symptom: `SIGBUS` or alignment fault on `ldr q0, [x0]`.

Cause: in some kernel configurations and on some pre-v8.4 cores, vector loads require natural alignment of the *element*, not 16 bytes — so `ldr q0, [x0]` is fine if x0 is 4-byte aligned (assuming v8.4 unaligned access support is enabled, which it almost always is on modern Linux). But MMIO regions and certain memory types force strict alignment.

Fix: use unaligned variants where intent is unaligned (`ldur`, NEON `ld1`), or pad/align the data with `__attribute__((aligned(16)))`.

### 17.3 `lock` Prefix Without Memory Operand

```nasm
lock add rax, rbx        ; ERROR — lock requires a memory destination
```

The assembler will accept this on some versions but produce a `#UD` (undefined opcode) at runtime. The `lock` prefix is only legal with a memory destination.

### 17.4 Forgetting `cld` Before `rep movsb`

If `DF=1` (set via `std`), `rep movsb` walks **backward** through memory. The SysV ABI requires functions to leave DF=0 on call/return, but signal handlers and old code may violate this. Always `cld` at the top of any function that uses string instructions.

### 17.5 Using `rcx` in a Linux x86-64 Syscall Argument

Symptom: garbage in the syscall.

Cause: the `syscall` instruction itself uses `rcx` to save the return address (and `r11` to save RFLAGS). The kernel ABI uses `r10` instead of `rcx` for the 4th argument.

Fix: copy your 4th-argument value into `r10` before `syscall`.

### 17.6 NEON on Apple Silicon — No SVE

If you're benchmarking SVE code on an M1/M2/M3 — it doesn't exist there. NEON only. Add a runtime check via `getauxval(AT_HWCAP)` (Linux) or `sysctlbyname("hw.optional.arm.FEAT_SVE", ...)` (macOS) before dispatching SVE paths.

### 17.7 RIP-Relative Addressing Confusion

`mov rax, [my_data]` on Linux with a non-PIE binary loads from absolute address `my_data`. With PIE (default on most distros), the same syntax means RIP-relative. NASM forces you to be explicit: `mov rax, [rel my_data]` for RIP-relative, `mov rax, [abs my_data]` for absolute.

### 17.8 AVX-VEX vs Legacy SSE Transition Penalty

If you call code that uses SSE (writes `xmm0` directly via legacy encoding) from code that uses AVX (writes `ymm0` via VEX-encoded instructions), the CPU pays a state transition penalty on every switch — tens of cycles. Always use VEX-encoded instructions in AVX-aware code (the assembler does this automatically when you use `vmovups` instead of `movups`), and call `vzeroupper` before returning to legacy code.

### 17.9 ARM64 Branch Range Limit

`b label` has ±128 MiB reach. For larger jumps use:

```gas
adrp x16, target
add  x16, x16, :lo12:target
br   x16
```

This is what the **linker veneer** does automatically when a function is too far away — it inserts a small trampoline, which is why x16/x17 must be considered scratch across `bl`.

### 17.10 Forgetting Saved Registers

```gas
my_func:
    ;; uses x19 without saving
    mov x19, x0
    bl  other_func        ; x19 corrupted by callee? NO — x19 is callee-saved
    ;; but YOU must save x19 if you set it before calling someone who saves it
```

Wait — that's wrong. Callee-saved means the callee preserves it. So if `other_func` follows the ABI, x19 survives. The trap is the **opposite direction**: if `my_func` is called by code that holds a value in x19, and `my_func` writes x19 without saving, the caller's value is destroyed.

Rule: every register you *write* that's in the callee-saved set must be saved/restored. Every register you *read across a `bl`* that's in the caller-saved set must be saved/restored.

---

## 18. Debugging Recipes

### 18.1 Find Where a Segfault Happened

```bash
ulimit -c unlimited                          # enable core dumps
./prog                                       # crash
gdb ./prog core
(gdb) bt                                     ; backtrace
(gdb) frame 0
(gdb) info registers
(gdb) x/16i $pc-32                           ; 16 instructions around fault
```

### 18.2 Watch a Register Across an Instruction

```
(gdb) display/x $rax
(gdb) ni
;; gdb auto-prints rax after every step
```

### 18.3 Hardware Watchpoint on a Memory Address

```
(gdb) watch *0x404020
(gdb) rwatch *0x404020       ; on read
(gdb) awatch *0x404020       ; on read or write
```

### 18.4 Single-Step Through a Hot Loop

```
(gdb) break my_func
(gdb) r
(gdb) layout asm
(gdb) layout regs
;; now step one insn at a time with `si`
```

### 18.5 Profile a Specific Section with `perf record`

```bash
perf record -F 4000 --call-graph dwarf -e cycles ./prog
perf report --stdio
perf annotate --stdio my_func
```

`perf annotate` shows the asm with sample percentages on the left — instantly tells you which instruction is hot.

---

## 19. Vocabulary

| Term | Meaning |
|:-----|:--------|
| Architectural register | Programmer-visible register (rax, x0). |
| Physical register | Internal renamed register pool (often 200+). |
| μop / micro-op | Internal RISC-like operation x86 instructions decode into. |
| ROB | Reorder buffer — holds in-flight μops until retire. |
| RS | Reservation station — μops waiting for operands. |
| MOB | Memory-ordering buffer — load/store queue. |
| Frontend | Fetch + decode + rename. |
| Backend | Execute + memory + retire. |
| Speculation | Executing past a not-yet-resolved branch or load. |
| Squash | Discarding speculative work after misprediction. |
| TLB | Translation Lookaside Buffer — virtual→physical cache. |
| Page walk | Hardware traversal of page tables on TLB miss. |
| Coherence | Multi-cache view consistency (MESI/MOESI). |
| Consistency (memory model) | Inter-thread visibility ordering. |
| ABI | Application Binary Interface — calling convention + ELF/Mach-O conventions. |
| ISA | Instruction Set Architecture — what the silicon decodes. |
| ELF | Executable and Linkable Format — Linux/BSD binary format. |
| Mach-O | macOS/iOS binary format. |
| PE/COFF | Windows binary format. |
| GOT | Global Offset Table — runtime address resolution. |
| PLT | Procedure Linkage Table — lazy-bound function trampolines. |
| Veneer | Linker-inserted bridge for out-of-range branches. |
| Litpool / Literal pool | Read-only constants placed near code (ARM). |
| Trampoline | Small generated code stub used to redirect control. |
| Prologue / Epilogue | Function entry / exit sequence (frame setup, register save). |
| Frame pointer | Register pointing to the start of the local frame. |
| Red zone | 128-byte area below rsp safe to use without adjustment (SysV). |
| Shadow space | 32-byte reserved area above rsp that callees may spill into (Win64). |
| HFA / HVA | Homogeneous Floating-point/Vector Aggregate (AArch64 ABI return rule). |
| FMA | Fused Multiply-Add — `a += b * c` in one rounding. |
| Barrier | Instruction that constrains memory ordering. |
| Acquire / Release | Half-fence with one-direction ordering. |
| Strong / Weak memory model | How aggressively the ISA reorders memory ops. |

---

## 20. The Cycle-Accurate Mental Model

A back-of-envelope cost table for x86-64 / ARM64 modern OoO cores (Skylake / M1-class):

| Operation | Latency | Throughput |
|:----------|:-------:|:----------:|
| ALU op (add, xor, shift) | 1 cy | 4/cy |
| imul 64×64 | 3 cy | 1/cy |
| idiv 64/64 | 20–80 cy | 0.05/cy |
| L1d load | 4–5 cy | 2/cy |
| L1d store | 1 cy | 1/cy |
| L2 hit | 12 cy | 1/cy |
| L3 hit | 35–50 cy | varies |
| DRAM | 200+ cy | dependent on row buffer |
| Branch (predicted) | 0–1 cy | 1/cy |
| Branch (mispredict) | 12–20 cy | — |
| `lock cmpxchg` (uncontended) | ~20 cy | low |
| `lock cmpxchg` (contended) | ~100+ cy | very low |
| `mfence` | ~30 cy | — |
| FP add/mul (AVX2) | 4 cy | 2/cy |
| FMA (AVX2) | 4 cy | 2/cy |
| Vector permute (AVX-512) | 3 cy | 1/cy |
| TLB hit + L1d | 4 cy | 2/cy |
| TLB miss (4-level walk) | 100+ cy | — |

For ARM64 (Apple M1 P-core, Neoverse N1):

| Operation | Latency | Throughput |
|:----------|:-------:|:----------:|
| ALU | 1 cy | 6/cy (Apple), 4/cy (Neoverse) |
| madd 64-bit | 3 cy | 1/cy |
| sdiv 64-bit | 8–12 cy | low |
| L1d load | 3–4 cy | 3/cy |
| L2 hit | 12–14 cy | — |
| LLC | 30–40 cy | — |
| DRAM | 80–150 cy | — |
| FP madd vector | 3 cy (Apple), 4 cy (Neoverse) | 4/cy / 2/cy |

These numbers are not exact — they vary by core, frequency, and microarchitectural state — but they're accurate to within a factor of 2 for modeling.

---

## 21. Putting It Together — A Tight Memory Walk

Suppose you want to sum a 16 MiB array of `int32_t`. Counting only memory bandwidth (the dominant cost):

```
elements    = 16 MiB / 4 B = 4 Mi = 4,194,304
cache lines = 16 MiB / 64 B = 262,144

DRAM streaming bandwidth (single core, modern DDR4) ≈ 15 GB/s
Time = 16 MiB / 15 GB/s = 16 * 1.04e6 / 15e9 ≈ 1.12 ms

Cycles at 3 GHz = 1.12e-3 * 3e9 = 3.36 million cycles
Cycles per element = 3.36e6 / 4.19e6 ≈ 0.8 cy/elem
```

Compute is essentially free; DRAM is the wall. To sum 1 GiB you need to read 1 GiB of data from DRAM — multiply the number above by 64.

This is the **memory-bound regime**. AVX-512 doesn't help. The only fixes are:
- Make the data smaller (compress, narrower types).
- Reuse data more (cache blocking, fusion).
- Parallelize across cores (each core has its own L1/L2; LLC and DRAM share).
- Skip the read (sparse representation, sketches).

---

## See Also

- `languages/c` — C is the lingua franca of "below the OS"; assembly fluency reads C asm output natively.
- `languages/rust` — `core::arch::asm!` is the modern systems-grade inline asm interface.
- `system/gdb` — disassembly, register inspection, stepping at instruction granularity.
- `performance/perf` — hardware counters, cycles/instructions, top-down analysis, annotated asm.
- `ramp-up/assembly-eli5` — narrative-shaped intro to the same material.
- `ramp-up/binary-numbering-eli5` — how integers, floats, and sign extension actually work in registers.

## References

- Intel® 64 and IA-32 Architectures Software Developer's Manual (SDM), Volumes 1–4. https://www.intel.com/sdm
- AMD64 Architecture Programmer's Manual, Volumes 1–5.
- Arm® Architecture Reference Manual for A-profile architecture (ARM DDI 0487). https://developer.arm.com/architectures/cpu-architecture/a-profile/docs
- Procedure Call Standard for the Arm 64-bit Architecture (AAPCS64). https://github.com/ARM-software/abi-aa
- System V Application Binary Interface AMD64 Architecture Processor Supplement. https://gitlab.com/x86-psABIs/x86-64-ABI
- Microsoft x64 calling convention. https://learn.microsoft.com/cpp/build/x64-calling-convention
- Hennessy, J. & Patterson, D. *Computer Architecture: A Quantitative Approach*, 6th ed. The canonical text.
- Patterson, D. & Hennessy, J. *Computer Organization and Design*, RISC-V edition. Undergraduate text on the same topics.
- Agner Fog. *Optimizing software in C++* and *Optimizing subroutines in assembly language*; *Instruction tables*; *The microarchitecture of Intel, AMD and VIA CPUs*. https://www.agner.org/optimize/
- Intel® Optimization Reference Manual.
- Drepper, U. *What Every Programmer Should Know About Memory*. LWN.
- Travis Downs. *Performance Matters* blog. https://travisdowns.github.io/
- Daniel Lemire. *Daniel Lemire's blog* — vectorization and SIMD techniques. https://lemire.me/blog/
- Linux kernel `Documentation/arch/x86/x86_64/mm.rst` and `Documentation/arch/arm64/`.
- IEEE 754-2019 — floating-point arithmetic.
- `man 2 syscall`, `man 2 perf_event_open`, `man 1 perf`, `man 1 objdump`, `man 1 nm`, `man 1 readelf`, `man 1 gdb`.
- Compiler Explorer (Godbolt). https://godbolt.org/
- The Linux Kernel `tools/perf/Documentation/`.
