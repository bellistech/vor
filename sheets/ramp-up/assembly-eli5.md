# Assembly — ELI5 (The Alphabet Your CPU Reads)

> Assembly is the alphabet your CPU reads directly; every program you have ever run is eventually translated into it before the brain of the computer can do any thinking.

## Prerequisites

(none — but `cs ramp-up linux-kernel-eli5` and `cs ramp-up binary-numbering-eli5` help if you want to know what hex and binary numbers are first)

This sheet starts from absolute zero. You do not need to know any programming language. You do not need to know what a "compiler" is. You do not need to know any math beyond "1 plus 1 is 2." Every weird word in this sheet is in the **Vocabulary** section near the bottom, with a one-line plain-English definition.

If you see a `$` at the start of a line in a code block, that means "type the rest of this line into your terminal." You do not type the `$`. The lines underneath that don't have a `$` are what your computer prints back at you. We call that "output."

Some of the code blocks have lines that start with a `;` or `//` or `#`. Those are comments. They are little notes for humans, not instructions for the computer. The computer ignores them. Sometimes I'll use them to label what each line does so you can follow along.

## What Even Is Assembly?

### Imagine the computer's brain only knows numbers

Picture the brain inside your computer. It is called the CPU. CPU stands for "Central Processing Unit," which is a fancy way of saying "the part that thinks." The CPU is a tiny little chip in the middle of your computer, and it is the only part that actually does any math, any moving of stuff, any decisions.

But here is the funny thing about the CPU: it does not understand English. It does not understand Python. It does not understand JavaScript or Go or Rust or Java or Swift. It does not even understand "add 1 plus 1" written in plain words. The CPU only understands one thing: **numbers.** Specifically, very specific patterns of numbers, which we call **machine code.**

Imagine you wrote a recipe. A normal recipe says "stir the flour." But the CPU's recipe book is more like:

```
01001000 10001001 11011000
01001000 10000011 11000000 00000001
11000011
```

That is what the CPU actually reads. Long strings of ones and zeros (called **bits**, which is short for "binary digit"). Eight bits in a row is called a **byte.** A whole program is just thousands or millions or billions of bytes, all in a long row, that the CPU goes through one at a time.

If you tried to write a program by typing ones and zeros all day, you would lose your mind in about ten minutes. Nobody does that. Nobody has done that since the 1950s.

### Enter assembly: a slightly friendlier nickname for each pattern

Some clever people decided: "Hey, what if we gave each pattern of bits a short word? Like, instead of writing `01001000 10000011 11000000 00000001`, what if we just wrote `add rax, 1`? It means the same thing to the CPU; we just have a translator that turns the words back into bits."

That is **assembly.** Assembly is a one-to-one nickname system. Every line of assembly turns into one (or sometimes two) instruction that the CPU can run. There is almost no magic happening between assembly and machine code. The only thing that happens is the words get turned into the bit patterns. The translator that does that is called an **assembler.** That is where the name comes from. You assemble the bit patterns out of the words.

So when you write:

```
add rax, 1
```

That is one line of assembly. It tells the CPU: "Take whatever number is in the box called `rax`, add 1 to it, and put the answer back in `rax`." When you run it through an assembler (a program called `as` on Linux, for example), the assembler turns it into the actual bytes that the CPU reads.

### Every program you have ever run becomes assembly first

Here is the thing that surprises a lot of people. Your web browser? Translated to assembly before it ran. Your music player? Translated to assembly. The video game on your phone? Translated to assembly. The fancy AI chatbot? Translated to assembly. The little blinking cursor in your terminal? Translated to assembly.

Every single piece of software ever, on every computer ever, eventually gets translated into machine code, which is just the bit-pattern version of assembly. The CPU does not run anything else. There is nothing below assembly. Assembly is the floor.

When you write code in Python, the Python interpreter is itself a program that runs as assembly, and it reads your Python code and figures out what assembly instructions to issue on your behalf. When you write code in C, a tool called a **compiler** turns your C code into assembly directly, and then the assembler turns the assembly into machine code. When you write JavaScript, the browser has a thing called a JIT (Just-In-Time compiler) that watches your JavaScript run and writes assembly on the fly. There is no escape. Eventually, for the CPU to actually do anything, it must be reading assembly-shaped bytes.

### So why don't we all just write assembly?

If everything ends up as assembly anyway, why do we have all these other languages?

Two reasons.

**Reason 1: Assembly is a pain.** When you write assembly, you have to keep track of every little thing yourself. You move stuff into boxes. You move stuff out of boxes. You count bytes. You can write down the wrong number for the size of something, and the CPU happily does the wrong thing without warning. A simple line of Python like `result = some_list[5] + 7` becomes maybe a dozen or twenty lines of assembly with all kinds of bookkeeping. Imagine writing your whole program at that level. People used to. It was slow.

**Reason 2: Assembly is not portable.** A program written in assembly for one kind of CPU will not run on a different kind of CPU. Each CPU family has its own dialect of assembly. We will see more about that in a minute. If you write your program in C, you can compile it once for x86_64 and once for ARM64 and once for RISC-V, and you get three working programs. If you write it in assembly, you have to write it three separate times, almost from scratch.

So instead, almost everyone writes in higher-level languages, and the compiler does the translation work for them. The assembly is still down there, doing the actual work, but most of us never see it.

### When do you actually need to know assembly?

You do not need to know assembly to be a programmer. Most working programmers never write a line of assembly in their whole career. But you sometimes need to **read** assembly, even if you never write it.

- Your program crashed and left a "core dump." The core dump shows you the assembly instructions that were running when it died.
- Your program is slow. You run a tool like `perf` to find the slow spot. The slow spot is reported as assembly.
- You are reverse-engineering somebody else's program. The only thing you have is the compiled binary. You disassemble it back into assembly and read it.
- You are writing or debugging an **eBPF** program (a tiny program that runs inside the kernel). The kernel checks each eBPF instruction, and the error messages are at the assembly level.
- You are writing a kernel, an operating system, a JIT compiler, or a piece of code that has to talk to the CPU directly.
- You are curious how computers actually work, all the way down.

This sheet is for that last one. By the end, you will be able to read a chunk of assembly and have a feel for what it is doing.

### One line of assembly = one (or two) CPU instructions

This is the most important sentence in this whole sheet, so read it twice.

> **One line of assembly = one (or two) CPU instructions.**

Compare that to higher-level languages. One line of Python might do hundreds of things underneath. One line like `print("hello")` becomes hundreds of assembly instructions. But one line of assembly is just one tiny thing: add two numbers, copy a value, jump to a different spot. That is why assembly programs are so long. To do anything interesting, you need a lot of one-tiny-thing instructions all in a row.

It is like the difference between a recipe that says "make a pizza" and a recipe that says:

```
pick up flour bag
walk to bowl
pour flour into bowl
walk to sink
turn on water
fill cup
walk back to bowl
pour cup into bowl
pick up spoon
stir
...
```

Each of those tiny steps is like an assembly instruction. The whole thing together makes a pizza.

## CPU Architectures Are Like Different Languages

Earlier I said "each CPU family has its own dialect of assembly." Let's unpack that.

A CPU is a physical chip. There are different brands and models. Just like there are different brands of cars: Toyota, Honda, Ford, BMW. Two cars can both go forward, both turn left, both stop at red lights. But the steering wheel is in a different spot, the buttons are different, the gas pedal is shaped differently. They do similar things, but each one has its own way of doing it.

CPUs are the same. Every CPU can do basic stuff: add numbers, move data, compare two values, jump to a different part of the program. But the exact words you say to make it do those things are different, depending on which family of CPU you are talking to.

The four families we will care about are:

### x86_64 (also called AMD64, Intel 64, or x64)

This is the dominant family on desktops, laptops, and servers. Almost every Intel CPU and every AMD CPU in a regular computer is x86_64. If you have a "PC" or a Linux server, almost certainly it is x86_64. Apple Macs from before 2020 were x86_64 too.

x86_64 is **CISC**, which stands for "Complex Instruction Set Computer." That means it has a LOT of instructions, and many of them do complicated things in one shot. There is an instruction that searches a string for a specific byte. There is an instruction that does floating-point math with a vector of 16 numbers at once. There is an instruction that handles cryptography. The instruction list is huge. Reading the official Intel manual is like reading a phone book.

Another quirk of x86_64: instructions are **variable length.** Some instructions are 1 byte. Some are 15 bytes. The CPU has to figure out where each instruction starts and ends as it reads. This is harder for the CPU to do (and for tools like disassemblers to do), but it lets x86_64 pack a lot of meaning into a few bytes.

### ARM64 (also called AArch64)

ARM64 is the dominant family on phones and tablets, and increasingly on laptops and servers. Apple Silicon (M1, M2, M3, M4 chips) is ARM64. The Raspberry Pi 4 and 5 are ARM64. AWS Graviton servers are ARM64. Most Android phones are ARM64 (or its 32-bit cousin ARM32).

ARM64 is **RISC**, which stands for "Reduced Instruction Set Computer." That means it has a small, simple list of instructions. Each instruction does one simple thing. To do something complicated, you string together several simple instructions. The instruction list is short and tidy.

ARM64 instructions are also **fixed length:** every instruction is exactly 4 bytes. This makes the CPU easier to design (it always knows where the next instruction starts, exactly 4 bytes ahead) and makes disassembly tools simpler.

### RISC-V

RISC-V is the new kid. It is **open source.** That means the design of the instruction set is freely available for anyone to look at, anyone to use, anyone to build a CPU around without paying any company a license fee. This is a big deal. Both x86_64 (owned by Intel and AMD) and ARM (owned by Arm Holdings) require licenses. RISC-V does not.

RISC-V is mostly used in embedded systems (tiny computers in your toaster, your car, your smart light bulb), in research, and in some servers. It is growing fast. Some people predict it will be a major player in five or ten years.

It is also RISC, with fixed 4-byte instructions (or 2 bytes if you use the optional "compressed" extension). It is **modular**, which means the basic CPU is tiny and you can pick which extensions to add: integer math, floating point math, vectors, atomic operations, and so on.

### eBPF

eBPF is the weird one. It is **not a real CPU.** There is no eBPF chip you can buy. There is no eBPF computer. Instead, eBPF is a **virtual** CPU that lives inside the Linux kernel.

When you write an eBPF program, you compile it to eBPF bytecode. Then you ask the kernel to run it. The kernel first checks (verifies) that your program is safe — it can't crash the kernel, it can't loop forever, it can't read memory it shouldn't. If the verifier is happy, the kernel translates the eBPF bytecode into real assembly for the actual CPU underneath, and runs it. So eBPF is sort of a layer on top of regular assembly.

eBPF is its own family of instructions, with its own register names and rules. You will see it later in this sheet because it shows up everywhere in modern Linux: in `bpftrace`, in `tcpdump`, in network filtering, in security tools, in performance monitoring.

### Side-by-side: "add two numbers" in each architecture

Here is the same simple program in all four architectures: take 5, add 7, store the result. The result should be 12.

x86_64:

```
mov rax, 5      ; put the number 5 into the box called rax
add rax, 7      ; add 7 to whatever is in rax (now rax holds 12)
```

ARM64:

```
mov x0, #5      // put the number 5 into the box called x0
add x0, x0, #7  // x0 = x0 + 7 (now x0 holds 12)
```

RISC-V:

```
li a0, 5        // load immediate: put the number 5 into a0
addi a0, a0, 7  // a0 = a0 + 7 (now a0 holds 12)
```

eBPF (in C-like pseudo-assembly):

```
r0 = 5          // put 5 into register 0
r0 += 7         // add 7 to r0
```

See the pattern? Same idea in all four. Slightly different words. Different boxes (called **registers**, which we'll get to next). But the core thing — "make this box hold a 5, then add 7 to it" — is the same shape in every dialect.

This is why assembly is sometimes called a family of languages: each architecture has its own vocabulary, but they all describe the same kind of work.

## Registers — The CPU's Tiny Workbench

We keep saying "boxes." Time to get the real name. The boxes are called **registers.**

### What is a register?

A register is a tiny storage spot **inside the CPU itself.** Not in RAM. Not on the disk. Not on the network. Right there on the chip, as close to the math units as you can possibly get.

Picture the CPU as a worker at a workbench. The workbench has a few little compartments built right into it. Those compartments are the registers. The worker can grab something from a compartment and use it instantly — no walking, no reaching across the room. The compartments are the fastest possible storage.

Compare that to RAM, which is across the room. To get something from RAM, the worker has to walk over, fetch it, walk back. That takes time. (Not a lot of time in human terms — maybe a few hundredths of a microsecond — but in CPU terms, that is forever.) So whenever the CPU is going to use something, it first **loads** it from RAM into a register, does the work, and then **stores** the result back to RAM if it needs to be saved.

Picture:

```
+---------------------------+
|         CPU CHIP          |
|                           |
|  +---+ +---+ +---+ +---+  |  <- registers (tiny, on-chip, super fast)
|  |rax| |rbx| |rcx| |rdx|  |
|  +---+ +---+ +---+ +---+  |
|         ^                 |
|         | load / store    |
+---------|-----------------+
          |
          v
+---------------------------+
|         R A M             |   <- main memory (across the room, slower)
|   billions of bytes       |
+---------------------------+
```

### Why so few registers?

You might think "make more registers, then." But there is a tradeoff. Each register has to be wired into the CPU's circuitry, which takes physical space on the chip and adds complexity. Also, each register has to be addressable by name in every instruction, which costs **bits** in the instruction encoding. If you had 1024 registers, each register name would take 10 bits to encode, and instructions would get fatter. So architectures pick a small fixed number, usually somewhere between 8 and 32.

x86_64 has 16 general-purpose registers.

ARM64 has 31 general-purpose registers (plus a 32nd that is hardwired to always read as zero).

RISC-V has 32 general-purpose registers (also with a zero register at slot 0).

eBPF has only 11 registers.

That is fewer registers than fingers and toes (well, except for ARM64 and RISC-V, which are close to fingers-and-toes).

### x86_64 register names

x86_64 has the funniest naming history. The same register has different names depending on how much of it you want to use.

```
+---------------------------------------------------------+
|                       RAX (64-bit)                      |
|                                                         |
|                          +---------------------+        |
|                          |     EAX (32-bit)    |        |
|                          |        +-----------+|        |
|                          |        | AX (16-bit)|        |
|                          |        +--+--+      |        |
|                          |        |AH|AL|      |        |
|                          |        +--+--+      |        |
+---------------------------------------------------------+
                                       8-bit halves
```

The 16 general-purpose registers in x86_64:

| 64-bit | 32-bit | 16-bit | Low 8-bit | Notes                   |
|--------|--------|--------|-----------|-------------------------|
| rax    | eax    | ax     | al        | accumulator             |
| rbx    | ebx    | bx     | bl        | base                    |
| rcx    | ecx    | cx     | cl        | counter                 |
| rdx    | edx    | dx     | dl        | data                    |
| rsi    | esi    | si     | sil       | source index            |
| rdi    | edi    | di     | dil       | destination index       |
| rbp    | ebp    | bp     | bpl       | base pointer (frame)    |
| rsp    | esp    | sp     | spl       | stack pointer           |
| r8     | r8d    | r8w    | r8b       | extended (added in x64) |
| r9     | r9d    | r9w    | r9b       | extended                |
| r10    | r10d   | r10w   | r10b      | extended                |
| r11    | r11d   | r11w   | r11b      | extended                |
| r12    | r12d   | r12w   | r12b      | extended                |
| r13    | r13d   | r13w   | r13b      | extended                |
| r14    | r14d   | r14w   | r14b      | extended                |
| r15    | r15d   | r15w   | r15b      | extended                |

Plus the **special** ones:
- `rip` — the instruction pointer. This holds the address of the next instruction the CPU will run. You almost never name it directly; the CPU manages it for you.
- `rflags` — the flags register. Holds little 1-bit signals about the last operation: was the result zero, was there a carry, was there an overflow. Branch instructions look at these flags.

### ARM64 register names

ARM64 has 31 general-purpose 64-bit registers. They are called `x0` through `x30`. If you only want the low 32 bits, the same registers are called `w0` through `w30`. Writing to `w0` zeros out the upper 32 bits of `x0` automatically (a nice safety feature).

Diagram:

```
+----------------------------------------------+
|                X0 (64-bit)                   |
|                       +-----------------+    |
|                       |    W0 (32-bit)  |    |
|                       +-----------------+    |
+----------------------------------------------+
```

Special registers:
- `sp` — stack pointer (sometimes called x31 in certain contexts, but treat it as its own thing).
- `xzr` / `wzr` — the zero register. Reads as 0, writes are silently discarded. Super handy for "set this thing to zero" or "compare to zero."
- `pc` — program counter (like rip). Not directly writable.
- `pstate` — the flags register. Holds N (negative), Z (zero), C (carry), V (overflow) flags.
- `lr` — link register, which is just another name for `x30`. When you call a function, the return address goes here.
- `fp` — frame pointer, which is just another name for `x29`. Holds the bottom of the current stack frame.

### RISC-V register names

RISC-V has 32 general-purpose registers, called `x0` through `x31`. But almost no human ever uses those names. RISC-V also gives every register an "ABI name" that says what it's typically used for. You will see the ABI names way more often.

```
+--------+----------+--------------------------------+
| Reg    | ABI Name | Typical Use                    |
+--------+----------+--------------------------------+
| x0     | zero     | Hardwired to 0 (always)        |
| x1     | ra       | Return address                 |
| x2     | sp       | Stack pointer                  |
| x3     | gp       | Global pointer                 |
| x4     | tp       | Thread pointer                 |
| x5-x7  | t0-t2    | Temporaries (caller-saved)     |
| x8     | s0/fp    | Saved / Frame pointer          |
| x9     | s1       | Saved (callee-saved)           |
| x10-11 | a0-a1    | Args / Return values           |
| x12-17 | a2-a7    | Args                           |
| x18-27 | s2-s11   | Saved (callee-saved)           |
| x28-31 | t3-t6    | Temporaries (caller-saved)     |
+--------+----------+--------------------------------+
```

Special registers:
- `pc` — the program counter.
- No flags register! RISC-V is unusual: comparison instructions look at register values directly, not at a separate flags register. We will get to that.

### eBPF register names

eBPF is the simplest of all. Just 11 registers, named `r0` through `r10`. No 32-bit halves, no 16-bit names, no aliasing. Every register is 64 bits.

```
+--------+----------------------------------------+
| Reg    | Purpose                                |
+--------+----------------------------------------+
| r0     | Return value (from helpers, from prog) |
| r1     | Arg 1 / context pointer on entry       |
| r2-r5  | Args 2-5 to helpers                    |
| r6-r9  | Callee-saved (preserved across calls)  |
| r10    | Frame pointer (read-only!)             |
+--------+----------------------------------------+
```

Note that `r10` is read-only. You cannot assign to it. The kernel decides where the eBPF program's stack lives, and `r10` is just a fixed pointer to it. Unusual, but it makes eBPF programs easier to verify (the kernel knows the stack address can never be tampered with).

### Why this matters

Whenever you read a chunk of assembly, the first thing you should do is figure out which architecture you're looking at, and then keep the register table handy. The instructions are easier to read when you know what each register name is for.

x86_64 mnemonic to remember: rax, rbx, rcx, rdx, then rsi/rdi (index registers), then rsp/rbp (stack), then r8-r15 (the eight extras added when x86 went 64-bit).

ARM64 mnemonic to remember: x0-x30, plus sp, xzr, lr (=x30), fp (=x29).

RISC-V mnemonic to remember: zero, ra, sp, then a0-a7 for args, t0-t6 for scratch, s0-s11 for saved.

eBPF: r0-r10, where r0 returns and r10 is read-only.

## Instructions

Now let's actually look at what instructions can do. The CPU has a small list of categories. Once you know the categories, you can guess what most instructions are doing even if you have never seen them before.

### Category 1: Arithmetic

Add, subtract, multiply, divide. The basics.

x86_64:
```
add rax, rbx        ; rax = rax + rbx
sub rax, rbx        ; rax = rax - rbx
imul rax, rbx       ; rax = rax * rbx (signed)
div rbx             ; rdx:rax / rbx, quotient in rax, remainder in rdx
inc rax             ; rax = rax + 1
dec rax             ; rax = rax - 1
neg rax             ; rax = -rax (negate)
```

ARM64:
```
add x0, x1, x2      // x0 = x1 + x2
sub x0, x1, x2      // x0 = x1 - x2
mul x0, x1, x2      // x0 = x1 * x2
sdiv x0, x1, x2     // x0 = x1 / x2 (signed)
udiv x0, x1, x2     // x0 = x1 / x2 (unsigned)
neg x0, x1          // x0 = -x1
madd x0, x1, x2, x3 // x0 = (x1 * x2) + x3 (multiply-add in one shot)
```

RISC-V:
```
add a0, a1, a2      // a0 = a1 + a2
sub a0, a1, a2      // a0 = a1 - a2
mul a0, a1, a2      // a0 = a1 * a2
div a0, a1, a2      // a0 = a1 / a2 (signed)
divu a0, a1, a2     // unsigned divide
addi a0, a1, 7      // a0 = a1 + 7 (immediate version)
```

eBPF:
```
r0 += r1            // r0 = r0 + r1
r0 -= r1            // r0 = r0 - r1
r0 *= r1            // r0 = r0 * r1
r0 /= r1            // r0 = r0 / r1
r0 += 5             // r0 = r0 + 5 (immediate)
```

Notice ARM64 and RISC-V usually have **three-operand** form: destination, source1, source2. x86_64 mostly has **two-operand**: the destination and one of the sources are the same register, so `add rax, rbx` actually means `rax = rax + rbx`. This is one of the big shape differences between CISC and RISC.

### Category 2: Logical (bitwise)

These are the ones that work on individual bits: AND, OR, XOR, NOT, shift left, shift right. They are how you do bit-twiddling, masking, packing flags into integers, and a lot of low-level tricks.

x86_64:
```
and rax, rbx        ; rax = rax AND rbx (bit-by-bit)
or rax, rbx         ; rax = rax OR rbx
xor rax, rbx        ; rax = rax XOR rbx
not rax             ; rax = NOT rax (flip every bit)
shl rax, 4          ; rax = rax << 4 (shift left, fill with zeros)
shr rax, 4          ; rax = rax >> 4 (logical, fill with zeros)
sar rax, 4          ; rax = rax >> 4 (arithmetic, fill with sign bit)
```

ARM64:
```
and x0, x1, x2      // bitwise AND
orr x0, x1, x2      // bitwise OR (note: spelled "orr" in ARM)
eor x0, x1, x2      // XOR (note: spelled "eor" in ARM)
mvn x0, x1          // x0 = NOT x1 ("move not")
lsl x0, x1, #4      // logical shift left
lsr x0, x1, #4      // logical shift right
asr x0, x1, #4      // arithmetic shift right
```

RISC-V:
```
and a0, a1, a2      // bitwise AND
or a0, a1, a2       // bitwise OR
xor a0, a1, a2      // XOR
sll a0, a1, a2      // shift left logical
srl a0, a1, a2      // shift right logical
sra a0, a1, a2      // shift right arithmetic
```

eBPF:
```
r0 &= r1            // bitwise AND
r0 |= r1            // bitwise OR
r0 ^= r1            // XOR
r0 <<= 4            // shift left
r0 >>= 4            // shift right (logical)
```

The classic trick: `xor rax, rax` on x86_64 sets `rax` to zero and is shorter (and faster) than `mov rax, 0`. ARM64 and RISC-V have a real zero register (`xzr` and `x0`), so they don't need this trick.

### Category 3: Memory (load and store)

Move data between RAM and registers.

In RISC architectures (ARM64, RISC-V, eBPF), there are **separate** instructions for load and store. You cannot do math directly on memory; you have to load first, do the math, then store. This is called the "load/store architecture."

In CISC (x86_64), the move instruction `mov` works for both loading and storing, and many other instructions can read or write memory directly.

x86_64:
```
mov rax, [rbx]      ; load from memory: rax = *rbx (whatever rbx points to)
mov [rbx], rax      ; store to memory: *rbx = rax
mov rax, [rbx + 8]  ; load with offset (struct field access)
mov rax, [rbx + rcx*8]  ; load with index (array element)
```

ARM64:
```
ldr x0, [x1]        // load: x0 = *x1
str x0, [x1]        // store: *x1 = x0
ldr x0, [x1, #8]    // load with offset
ldp x0, x1, [x2]    // load pair: x0 = *x2, x1 = *(x2+8)
stp x0, x1, [x2]    // store pair
```

RISC-V:
```
ld a0, 0(a1)        // load doubleword (8 bytes): a0 = *a1
sd a0, 0(a1)        // store doubleword: *a1 = a0
lw a0, 4(a1)        // load word (4 bytes) with offset
lb a0, 0(a1)        // load byte (1 byte)
lh a0, 0(a1)        // load halfword (2 bytes)
```

eBPF:
```
r0 = *(u32 *)(r1 + 14)   // load 32-bit value from memory
*(u32 *)(r1 + 14) = r0   // store 32-bit value to memory
```

The square brackets in x86_64, the square brackets in ARM64, and the `(reg)` form in RISC-V all mean the same thing: "treat this register as a pointer to memory and access whatever it points to."

### Category 4: Branches (jumps)

Branches are how you make decisions and loops. Without branches, a program just runs straight through, top to bottom, once. With branches, you can say "if this is true, go over there; otherwise keep going."

x86_64 uses a two-step pattern: first **compare** two values (which sets the flags register), then **conditionally jump** based on the flags.

```
cmp rax, rbx        ; compare rax and rbx (sets ZF, SF, CF, OF flags)
je label            ; jump to "label" if equal (ZF=1)
jne label           ; jump if not equal
jl label            ; jump if less than (signed)
jg label            ; jump if greater than (signed)
jb label            ; jump if below (unsigned)
ja label            ; jump if above (unsigned)
jmp label           ; unconditional jump
```

ARM64 also uses compare-and-branch, with explicit flag-setting:

```
cmp x0, x1          // compare x0 and x1, set flags
b.eq label          // branch if equal
b.ne label          // branch if not equal
b.lt label          // branch if less (signed)
b.gt label          // branch if greater (signed)
b.lo label          // branch if lower (unsigned)
b.hi label          // branch if higher (unsigned)
b label             // unconditional branch
cbz x0, label       // compare-and-branch if x0 is zero (single instruction!)
cbnz x0, label      // compare-and-branch if x0 is not zero
```

RISC-V is different. There is **no flags register.** Instead, the branch instructions compare two registers directly:

```
beq a0, a1, label   // branch if a0 == a1
bne a0, a1, label   // branch if a0 != a1
blt a0, a1, label   // branch if a0 < a1 (signed)
bge a0, a1, label   // branch if a0 >= a1 (signed)
bltu a0, a1, label  // branch if a0 < a1 (unsigned)
bgeu a0, a1, label  // branch if a0 >= a1 (unsigned)
j label             // jump (unconditional)
```

eBPF is similar to RISC-V style: branches compare two values directly.

```
if r0 == r1 goto +5     // branch +5 instructions if equal
if r0 != r1 goto +5
if r0 > r1 goto +5      // unsigned >
if r0 s> r1 goto +5     // signed >
goto +5                  // unconditional
```

Here is a tiny "if/else" example in x86_64:

```
    cmp rax, 0
    je is_zero
    ; not zero — do this
    mov rbx, 1
    jmp done
is_zero:
    ; zero — do this
    mov rbx, 2
done:
    ; both paths land here
```

Imagine a fork in the road. The `cmp` looks at the sign and decides which way to fork. The `je` is the road sign saying "if equal, take the left fork." Otherwise you keep going straight, do some stuff, and then `jmp done` skips over the right fork's code.

### Category 5: Function call and return

When a program calls a function, two things have to happen:
1. The CPU needs to remember where to come back to when the function is done.
2. The CPU needs to jump to the function's code.

When the function is done, the CPU has to look up that "where to come back to" address and jump back there.

x86_64 does this with `call` (which pushes the return address onto the stack and then jumps) and `ret` (which pops the return address off the stack and jumps to it):

```
call my_function    ; push return address on stack, jump to my_function
; ... my_function runs ...
ret                  ; pop return address, jump to it
```

ARM64 has a slightly different style. Instead of pushing the return address on the stack, it puts it in a **register** called `lr` (link register, which is `x30`):

```
bl my_function      // branch with link: lr = address-of-next-instruction; branch to my_function
// ... my_function runs ...
ret                  // branch to whatever's in lr
```

This is faster for **leaf** functions (functions that don't call other functions), because they don't need to touch the stack at all. But if a function does call another function, it has to save its `lr` to the stack first, otherwise the inner `bl` would overwrite it.

RISC-V is similar to ARM64: it uses `ra` (x1) as the link register.

```
jal ra, my_function  // jump and link: ra = next_pc; jump to my_function
ret                   // pseudo-instruction for "jalr zero, ra, 0" (jump to ra)
```

eBPF has its own "call" instruction that calls helper functions provided by the kernel. There's no return address juggling — the kernel handles it all.

## The Stack

We keep mentioning "the stack." Time to explain.

### What the stack is

The stack is a region of memory that grows and shrinks as functions are called and return. It is called a stack because it works like a stack of plates: you push a plate on top, and when you take one off, you take from the top.

Picture a tall narrow column of memory:

```
high addresses
+-------------------+
| caller's frame    |
+-------------------+
| return address    |  <- pushed by `call`
+-------------------+
| saved registers   |
+-------------------+
| local variables   |  <- this function's stuff
+-------------------+  <- rsp / sp points here (top of stack)
low addresses
```

On most modern systems, the stack grows **downward** (from high addresses to low addresses). When you push something, the stack pointer **decreases**. When you pop, the stack pointer **increases**.

### Why do functions need a stack?

Three reasons.

**Reason 1: Return addresses.** When function A calls function B, the CPU needs to remember "after B finishes, come back to this exact spot in A." That address gets saved on the stack (or in a link register that gets later saved on the stack if needed).

**Reason 2: Local variables.** A function might need scratch space — variables that exist only while the function is running. These go on the stack. When the function returns, the space is reclaimed automatically (just by moving the stack pointer back up). No manual cleanup.

**Reason 3: Saved registers.** If the function wants to use a register that the caller is also using, the function must save the old value first and restore it before returning. Saved values go on the stack.

### push and pop

The simplest way to put something on the stack is `push`:

```
push rax            ; rsp -= 8; *rsp = rax  (x86_64)
```

That decrements the stack pointer by 8 (because rax is 8 bytes) and writes rax to the new top of stack.

To take it off:

```
pop rax             ; rax = *rsp; rsp += 8
```

Read the value at the top, then advance the stack pointer back up.

ARM64 doesn't have a push/pop pair as such. Instead it has pre-indexed loads and stores:

```
str x0, [sp, #-16]!   // pre-decrement: sp -= 16, then store x0 to [sp]
ldr x0, [sp], #16     // post-increment: load x0 from [sp], then sp += 16
```

The exclamation mark means "actually update sp." The pre-indexed form is push. The post-indexed form is pop.

RISC-V is more manual:

```
addi sp, sp, -16     // make space on stack (push: subtract 16)
sd ra, 8(sp)         // store ra at sp+8

ld ra, 8(sp)         // load ra back
addi sp, sp, 16      // pop: free the space
```

### Stack frames and the frame pointer

A **stack frame** is the chunk of stack that belongs to one function call. When function A calls function B, A's frame is below (at lower addresses than) B's frame. When B returns, B's frame is gone and A's frame is back on top.

There's usually a register called the **frame pointer** that points to the start of the current frame. The frame pointer makes it easy to find your local variables (always at fixed offsets from the frame pointer) and to walk up the stack if you need to find your caller, your caller's caller, and so on.

- x86_64: frame pointer is `rbp`.
- ARM64: frame pointer is `x29` (also called `fp`).
- RISC-V: frame pointer is `s0` (also called `fp`).

A typical x86_64 function prologue:

```
push rbp            ; save the old frame pointer
mov rbp, rsp        ; set new frame pointer to current stack top
sub rsp, 32         ; allocate 32 bytes for locals
```

And the matching epilogue:

```
mov rsp, rbp        ; restore stack pointer (frees locals)
pop rbp             ; restore old frame pointer
ret                  ; return to caller
```

ARM64 prologue:

```
stp x29, x30, [sp, #-16]!   // save fp and lr together
mov x29, sp                  // set new frame pointer
```

ARM64 epilogue:

```
ldp x29, x30, [sp], #16     // restore fp and lr
ret
```

The pattern is the same in every architecture: save the old frame state, set up the new frame, do the work, restore the old frame state, return.

## System Calls in Assembly

Programs cannot do anything dangerous on their own. They can do math. They can move data around in their own memory. But they cannot read a file. They cannot write to the screen. They cannot send a network packet. For any of that, they have to ask the kernel. (See `cs ramp-up linux-kernel-eli5` for a deep dive on the kernel.)

The polite way to ask the kernel is a **system call**, or **syscall.**

At the assembly level, a syscall is a special instruction that says "hey kernel, I need a thing." The CPU briefly hands control to the kernel, the kernel does the thing, and the CPU comes back to the program with an answer.

Each syscall has a **number.** "Read a file" is a different number than "write to the screen." There are a few hundred syscalls in Linux. The numbers are different on each architecture (yes, even though Linux runs on all of them, the syscall numbers are picked separately for each architecture, for historical reasons).

### x86_64 syscall convention

```
+-----------------------+
|  Syscall number → rax |
+-----------------------+
|  Arg 1 → rdi          |
|  Arg 2 → rsi          |
|  Arg 3 → rdx          |
|  Arg 4 → r10          |  <- note: NOT rcx!
|  Arg 5 → r8           |
|  Arg 6 → r9           |
+-----------------------+
|  syscall instruction  |
+-----------------------+
|  Result ← rax         |
+-----------------------+
```

Why r10 instead of rcx for arg 4? Because the `syscall` instruction itself uses rcx to save the return address. So the kernel ABI swapped in r10 to keep arg 4 from being trashed.

Example: write "hi\n" to stdout, then exit cleanly.

```
section .data
msg: db "hi", 10           ; "hi" plus newline byte
len: equ $ - msg            ; length is 3

section .text
global _start
_start:
    mov rax, 1              ; syscall number for write
    mov rdi, 1              ; fd 1 = stdout
    mov rsi, msg            ; address of message
    mov rdx, len            ; length
    syscall

    mov rax, 60             ; syscall number for exit
    xor rdi, rdi            ; exit code 0
    syscall
```

### ARM64 syscall convention

```
+-----------------------+
|  Syscall number → x8  |
+-----------------------+
|  Arg 1 → x0           |
|  Arg 2 → x1           |
|  Arg 3 → x2           |
|  Arg 4 → x3           |
|  Arg 5 → x4           |
|  Arg 6 → x5           |
+-----------------------+
|  svc #0 instruction   |  <- "supervisor call"
+-----------------------+
|  Result ← x0          |
+-----------------------+
```

Same hello world on ARM64:

```
.data
msg: .ascii "hi\n"

.text
.global _start
_start:
    mov x8, #64            // syscall number for write on ARM64
    mov x0, #1             // fd = stdout
    ldr x1, =msg           // address of message
    mov x2, #3             // length
    svc #0

    mov x8, #93            // syscall number for exit on ARM64
    mov x0, #0             // exit code 0
    svc #0
```

### RISC-V syscall convention

```
+-----------------------+
|  Syscall number → a7  |
+-----------------------+
|  Arg 1 → a0           |
|  Arg 2 → a1           |
|  Arg 3 → a2           |
|  Arg 4 → a3           |
|  Arg 5 → a4           |
|  Arg 6 → a5           |
+-----------------------+
|  ecall instruction    |  <- "environment call"
+-----------------------+
|  Result ← a0          |
+-----------------------+
```

```
.data
msg: .ascii "hi\n"

.text
.global _start
_start:
    li a7, 64              # write
    li a0, 1               # stdout
    la a1, msg             # buffer
    li a2, 3               # length
    ecall

    li a7, 93              # exit
    li a0, 0
    ecall
```

Notice the symmetry across the architectures:
- x86_64: number in `rax`, args in `rdi/rsi/rdx/r10/r8/r9`, instruction `syscall`.
- ARM64: number in `x8`, args in `x0-x5`, instruction `svc #0`.
- RISC-V: number in `a7`, args in `a0-a5`, instruction `ecall`.

Each architecture picked a different way, but the pattern is the same: stuff the number and arguments in known registers, then run a special "ask the kernel" instruction.

## Calling Conventions

A **calling convention** is a set of rules that says how function calls work at the assembly level: which registers carry arguments, which register holds the return value, who is responsible for saving what. Without a calling convention, every function would have to be paired with the exact code that calls it. With a calling convention, any code from any source can call any function, as long as both sides follow the same rules.

### System V AMD64 ABI (x86_64 Linux)

This is the convention on Linux for x86_64. It's also used on macOS and BSD.

Argument passing:
- Args 1-6 (integers/pointers) → `rdi`, `rsi`, `rdx`, `rcx`, `r8`, `r9`
- Args beyond 6 → on the stack
- Floating-point args → `xmm0` through `xmm7`

Return value:
- Integer/pointer → `rax`
- 128-bit return → `rdx:rax`
- Floating-point → `xmm0`

Register save responsibilities:
- **Caller-saved** (volatile): `rax`, `rcx`, `rdx`, `rsi`, `rdi`, `r8`-`r11`, `xmm0`-`xmm15`. The callee can clobber these freely. If the caller wants to keep a value, it must save before the call.
- **Callee-saved** (preserved): `rbx`, `rbp`, `r12`-`r15`. The callee must preserve these. If it wants to use one, it must save and restore.

Stack must be 16-byte aligned at the moment a `call` instruction runs. Don't ask why, just always make sure your stack is aligned, or things break in weird ways.

### AAPCS64 (ARM64)

Argument passing:
- Args 1-8 → `x0` through `x7`
- Args beyond 8 → on the stack
- Floating-point → `v0` through `v7`

Return value:
- Integer/pointer → `x0`
- Larger → `x0:x1`

Register save responsibilities:
- Caller-saved: `x0`-`x18`
- Callee-saved: `x19`-`x28`, `x29` (fp), `x30` (lr)

Stack alignment: 16 bytes.

### RISC-V Calling Convention

Argument passing:
- Args 1-8 → `a0` through `a7` (which are `x10` through `x17`)
- Args beyond 8 → on the stack

Return value:
- → `a0` (and `a1` for 128-bit)

Register save responsibilities:
- Caller-saved (temporary): `t0`-`t6`, `a0`-`a7`
- Callee-saved: `s0`-`s11`, `ra`, `sp`

Stack alignment: 16 bytes (RV64).

### eBPF Calling Convention

Different! eBPF programs aren't called by user code; they're called by the kernel. But they do call **helper functions** provided by the kernel.

- Args 1-5 → `r1` through `r5`
- Return → `r0`
- `r6`-`r9` are callee-saved across helper calls
- `r10` is read-only (frame pointer)

There can be no more than 5 arguments to a helper. There is no stack-passed argument concept in eBPF.

## Linking and Relocation

When you write a program in pieces (multiple files, libraries, etc.), each piece gets compiled to assembly and then assembled to machine code separately. Each piece is an **object file** (extension `.o` on Linux).

Object files are not yet runnable. They have **placeholders** in them. For example, if you call a function that's defined in a different object file, your assembly says `call someother_function`, but the assembler doesn't know the actual address yet. So it leaves a placeholder and writes a note: "fill this in later when you know where someother_function lives."

The **linker** is the program that takes all the object files, decides where everything goes in memory, and fills in all the placeholders. The output is a **runnable executable**, which on Linux is an **ELF file** (Executable and Linkable Format).

```
+------------------+        +------------------+
|   hello.c        |        |   greet.c        |
+------------------+        +------------------+
        |                            |
        | gcc -c hello.c              | gcc -c greet.c
        v                            v
+------------------+        +------------------+
|   hello.o        |        |   greet.o        |
| (placeholders!)  |        | (placeholders!)  |
+------------------+        +------------------+
        \                          /
         \                        /
          \                      /
           v                    v
              +----------+
              |  linker  |
              +----------+
                   |
                   v
            +------------+
            | hello (ELF)|   <- runnable
            +------------+
```

### ELF layout

An ELF file has a few sections:

```
+----------------------+
| ELF header           |   header info: arch, entry point, etc.
+----------------------+
| program headers      |   how to load segments into memory
+----------------------+
| .text                |   the actual code (assembly, assembled)
+----------------------+
| .rodata              |   read-only data (strings, constants)
+----------------------+
| .data                |   read/write initialized data (globals)
+----------------------+
| .bss                 |   read/write zero-initialized data
+----------------------+
| .symtab              |   symbol table (function/variable names)
+----------------------+
| .strtab              |   strings used in symtab
+----------------------+
| section headers      |   metadata about sections
+----------------------+
```

You can look at any of this with `readelf` (we'll see commands for this in the Hands-On section).

### Static vs dynamic linking

**Static** linking: the linker copies all the library code into your executable. Your executable is fat but self-contained.

**Dynamic** linking: the linker leaves library calls as placeholders. At runtime, the **dynamic linker** (yes, another linker) finds the libraries and patches up the placeholders. Your executable is small but depends on the right libraries being present.

Most Linux programs are dynamically linked against `libc.so` (the C standard library). The placeholders for dynamic functions go through two tables:

- **PLT** (Procedure Linkage Table): a stub the program calls. The first time it's called, the stub goes to the dynamic linker. The dynamic linker resolves the address and patches the GOT.
- **GOT** (Global Offset Table): a table of resolved addresses. After resolution, calls go through the GOT directly without bothering the dynamic linker.

This dance is called **lazy binding** because functions are only resolved when first called.

### Position-independent code (PIC)

PIC means: the code can be loaded at any address in memory and still work. This is required for shared libraries (so they can be loaded into different programs at different addresses) and for **ASLR** (Address Space Layout Randomization), a security feature that randomizes addresses to make exploits harder.

PIC works by using **relative** addressing: instead of saying "the variable is at address 0x1234", you say "the variable is at offset 200 from where I am right now." x86_64 has `[rip + offset]` for this. ARM64 has `adr` and `adrp`. RISC-V has `auipc`.

## eBPF Bytecode

eBPF deserves its own deep section because it's the architecture you'll see most often when modern Linux is doing weird modern Linux things.

### What eBPF is

eBPF stands for "extended Berkeley Packet Filter." Originally BPF was a tiny instruction set used to filter network packets in the kernel (you'd write a tiny program that said "keep packets to port 80, drop others"). Then somebody noticed that this could be generalized: what if you could safely run user-supplied programs anywhere in the kernel? Tracing. Networking. Security. Performance. eBPF was the result.

For the long ELI5 take, see `cs ramp-up ebpf-eli5`.

### eBPF instruction encoding

Every eBPF instruction is exactly **8 bytes** (64 bits). They look like this:

```
   bit:  63        47    43    39                            0
         +----------+-----+-----+----------------+--------------------------+
         |  opcode  | dst | src |     offset     |        immediate         |
         |  8 bits  | 4 b | 4 b |    16 bits     |        32 bits           |
         +----------+-----+-----+----------------+--------------------------+
```

- **opcode** (1 byte): which instruction this is.
- **dst** (4 bits): destination register, 0-15 (only 0-10 valid in eBPF).
- **src** (4 bits): source register or 0 if not needed.
- **offset** (2 bytes): for memory operations or jump targets.
- **immediate** (4 bytes): a constant value baked into the instruction.

For 64-bit immediate loads, two consecutive 8-byte slots are used (16 bytes total). That's the only "wide" instruction.

### eBPF instruction classes

eBPF groups instructions into a small set of classes:

| Class      | What it does                              |
|------------|-------------------------------------------|
| BPF_LD     | Load (mostly historical/legacy)            |
| BPF_LDX    | Load from memory into register             |
| BPF_ST     | Store immediate to memory                  |
| BPF_STX    | Store register to memory                   |
| BPF_ALU    | 32-bit arithmetic/logic                    |
| BPF_ALU64  | 64-bit arithmetic/logic                    |
| BPF_JMP    | 64-bit branches and call/exit              |
| BPF_JMP32  | 32-bit branches                            |

### eBPF ALU operations

```
add  : dst = dst + src
sub  : dst = dst - src
mul  : dst = dst * src
div  : dst = dst / src
or   : dst = dst | src
and  : dst = dst & src
lsh  : dst = dst << src   (left shift)
rsh  : dst = dst >> src   (right shift, logical)
neg  : dst = -dst
mod  : dst = dst % src
xor  : dst = dst ^ src
mov  : dst = src
arsh : dst = dst >> src   (arithmetic right shift)
```

### eBPF memory operations

Memory accesses come in 4 sizes:

```
BPF_B  : 8 bits   (byte)
BPF_H  : 16 bits  (halfword)
BPF_W  : 32 bits  (word)
BPF_DW : 64 bits  (doubleword)
```

Example:

```
r0 = *(u32 *)(r1 + 14)   // load 32-bit value from r1+14 into r0
*(u8 *)(r10 - 4) = r2    // store low byte of r2 at stack position r10-4
```

### eBPF branches

```
if r0 == r1 goto +N    // BPF_JEQ
if r0 != r1 goto +N    // BPF_JNE
if r0 > r1 goto +N     // BPF_JGT (unsigned)
if r0 >= r1 goto +N    // BPF_JGE (unsigned)
if r0 s> r1 goto +N    // BPF_JSGT (signed)
if r0 s>= r1 goto +N   // BPF_JSGE (signed)
if r0 < r1 goto +N     // BPF_JLT
if r0 <= r1 goto +N    // BPF_JLE
goto +N                 // BPF_JA (always)
```

The "+N" is an offset relative to the next instruction.

### eBPF call and exit

```
call helper_id          // call kernel helper function (id is in immediate)
exit                     // return; r0 holds the return value
```

There are hundreds of helper functions: get the current time, look up a value in a map, send a network packet, print something to the trace pipe, and so on.

### Why eBPF has only 11 registers

The verifier (the kernel piece that checks every eBPF program before running it) tracks the type and value range of every register at every instruction. With 11 registers, that tracking is tractable. With 32 or 64, the verifier would be way slower or have to be way dumber. Eleven is the sweet spot.

### Why r10 is read-only

The eBPF stack is fixed at 512 bytes per program. The kernel decides where in memory the stack lives. `r10` always points to the start of that stack region. If `r10` could be changed, the verifier could no longer prove that stack accesses stay within the 512-byte region. Making `r10` read-only solves that elegantly.

## Debugging With GDB and Friends

Knowing assembly lets you actually use a debugger usefully. Here's a quick tour of the biggest debugger on Linux: `gdb`.

### Starting gdb

```
gdb ./myprogram
```

Inside gdb, you get a `(gdb)` prompt where you type commands.

### First thing to do: switch to Intel syntax

```
(gdb) set disassembly-flavor intel
```

By default, gdb uses AT&T syntax (the older Unix syntax). Intel syntax is what most documentation uses. We'll talk more about syntax differences later.

### Setting a breakpoint

```
(gdb) break main           # break at function "main"
(gdb) break *0x401050      # break at exact address
(gdb) break file.c:42      # break at line 42 of file.c
(gdb) run                  # start the program; it'll stop at the breakpoint
```

### Stepping

```
(gdb) stepi    (si)        # execute one assembly instruction; step into calls
(gdb) nexti    (ni)        # execute one instruction; step over calls
(gdb) continue (c)         # run until next breakpoint
(gdb) finish               # run until current function returns
```

### Looking at registers

```
(gdb) info registers       # show all GP registers and their current values
(gdb) p/x $rax             # print rax in hex
(gdb) p/d $rax             # print rax in decimal
```

### Looking at memory

```
(gdb) x/10i $rip           # examine 10 instructions starting at rip
(gdb) x/8gx $rsp           # examine 8 64-bit hex values at rsp (the stack)
(gdb) x/16bx $rdi          # examine 16 bytes hex at rdi
(gdb) x/s $rdi             # examine string at rdi
```

The `x` command format is `x/<count><format><size>`. Examples: `i` for instructions, `x` for hex, `d` for decimal, `s` for string. Sizes: `b` byte, `h` halfword (2 bytes), `w` word (4 bytes), `g` giant (8 bytes).

### TUI mode

For a much friendlier visual layout:

```
(gdb) layout split         # source + assembly in side-by-side panes
(gdb) layout asm           # just assembly
(gdb) layout regs          # show register pane
(gdb) tui reg general      # focus the GP register pane
```

You can step around with `si` and `ni` and watch which instruction is current and which registers change.

### lldb

`lldb` is the Apple/LLVM equivalent of gdb. The commands are slightly different. The same ideas apply. If you're on macOS, you'll mostly use lldb.

```
lldb ./myprogram
(lldb) breakpoint set --name main
(lldb) run
(lldb) reg read
(lldb) di -s 0x400000 -c 10    # disassemble 10 instructions
```

## Hands-On

These commands are safe. They will all work on a normal Linux machine. They show you real assembly and real machine state. Try them.

### See what kind of CPU you have

```
$ uname -m
x86_64
```

That single word tells you what architecture your system is running. Possible values: `x86_64`, `aarch64` (ARM64), `riscv64`, `i386`, `i686`, `armv7l`, etc.

### See more details

```
$ cat /proc/cpuinfo | grep -E "model name|flags" | head -5
model name      : Intel(R) Core(TM) i7-8550U CPU @ 1.80GHz
flags           : fpu vme de pse tsc msr pae mce cx8 apic sep mtrr ...
```

The "flags" line lists every CPU feature your processor supports. If you see `sse2`, your CPU can do 128-bit SIMD math. If you see `avx2`, it can do 256-bit. If you see `avx512`, it can do 512-bit.

### Compact CPU info

```
$ lscpu | head -20
Architecture:                       x86_64
CPU op-mode(s):                     32-bit, 64-bit
Byte Order:                         Little Endian
Address sizes:                      39 bits physical, 48 bits virtual
CPU(s):                             8
On-line CPU(s) list:                0-7
Vendor ID:                          GenuineIntel
Model name:                         Intel(R) Core(TM) i7-8550U CPU @ 1.80GHz
CPU family:                         6
Model:                              142
Thread(s) per core:                 2
Core(s) per socket:                 4
Socket(s):                          1
Stepping:                           10
CPU max MHz:                        4000.0000
CPU min MHz:                        400.0000
BogoMIPS:                           3984.00
```

`lscpu` is the friendly version of `cat /proc/cpuinfo`.

### Disassemble a real binary

```
$ objdump -d /usr/bin/ls | head -50
/usr/bin/ls:     file format elf64-x86-64

Disassembly of section .init:

0000000000004000 <_init>:
    4000:       f3 0f 1e fa             endbr64
    4004:       48 83 ec 08             sub    $0x8,%rsp
    4008:       48 8b 05 d9 4f 02 00    mov    0x24fd9(%rip),%rax        # 28fe8 <__gmon_start__>
    400f:       48 85 c0                test   %rax,%rax
    4012:       74 02                   je     4016 <_init+0x16>
    4014:       ff d0                   call   *%rax
    4016:       48 83 c4 08             add    $0x8,%rsp
    401a:       c3                      ret
```

Each line is one instruction. The first column is the address. The middle column is the actual machine code bytes. The last column is the assembly. Note this is AT&T syntax (the `%rsp` and `$0x8` give it away).

### Disassemble in Intel syntax

```
$ objdump -d -M intel /usr/bin/ls | head -50
0000000000004000 <_init>:
    4000:       f3 0f 1e fa             endbr64
    4004:       48 83 ec 08             sub    rsp,0x8
    4008:       48 8b 05 d9 4f 02 00    mov    rax,QWORD PTR [rip+0x24fd9]
    400f:       48 85 c0                test   rax,rax
    4012:       74 02                   je     4016 <_init+0x16>
    4014:       ff d0                   call   rax
    4016:       48 83 c4 08             add    rsp,0x8
    401a:       c3                      ret
```

Same code, different syntax. Intel uses no `%` or `$` prefixes and has destination first. The `mov` lines show the order swap: `mov rsp, 0x8` versus `sub %rsp, $0x8`.

### Disassemble in AT&T syntax (explicit)

```
$ objdump -d -M att /usr/bin/ls | head -50
```

Same as the default `objdump -d`, just being explicit about the syntax.

### List symbols in a binary

```
$ nm /usr/bin/ls | head -20
0000000000028e30 b __bss_start
0000000000028e30 b completed.0
                 U cxa_finalize@GLIBC_2.2.5
0000000000020000 d __data_start
0000000000028e30 d __data_start
                 w __gmon_start__
                 U abort@GLIBC_2.2.5
                 U __asprintf_chk@GLIBC_2.8
                 U bindtextdomain@GLIBC_2.2.5
                 U calloc@GLIBC_2.2.5
                 U __ctype_b_loc@GLIBC_2.3
                 U __ctype_get_mb_cur_max@GLIBC_2.2.5
                 U __ctype_toupper_loc@GLIBC_2.3
                 U __ctype_tolower_loc@GLIBC_2.3
                 U __cxa_atexit@GLIBC_2.2.5
                 U dlsym@GLIBC_2.2.5
                 U error@GLIBC_2.2.5
                 U exit@GLIBC_2.2.5
                 U __exit_group@GLIBC_2.2.5
                 U strchr@GLIBC_2.2.5
```

Each line is one symbol. The letter after the address tells you what kind: `T` is a text (function) symbol, `D` is data, `B` is bss (uninitialized data), `U` is undefined (lives in some other library), `W` is weak.

### See the ELF file header

```
$ readelf -h /usr/bin/ls
ELF Header:
  Magic:   7f 45 4c 46 02 01 01 00 00 00 00 00 00 00 00 00
  Class:                             ELF64
  Data:                              2's complement, little endian
  Version:                           1 (current)
  OS/ABI:                            UNIX - System V
  ABI Version:                       0
  Type:                              DYN (Position-Independent Executable file)
  Machine:                           Advanced Micro Devices X86-64
  Version:                           0x1
  Entry point address:               0x6ab0
  Start of program headers:          64 (bytes into file)
  Start of section headers:          135808 (bytes into file)
  Flags:                             0x0
  Size of this header:               64 (bytes)
  Size of program headers:           56 (bytes)
  Number of program headers:         13
  Size of section headers:           64 (bytes)
  Number of section headers:         30
  Section headers string table index: 29
```

The first 16 bytes of every ELF file are the magic header. `0x7f` followed by `'E'`, `'L'`, `'F'` (the ASCII codes 0x45, 0x4c, 0x46) is the file-type signature.

### See the dynamic section

```
$ readelf -d /usr/bin/ls
Dynamic section at offset 0x21db8 contains 27 entries:
  Tag        Type                         Name/Value
 0x0000000000000001 (NEEDED)             Shared library: [libselinux.so.1]
 0x0000000000000001 (NEEDED)             Shared library: [libc.so.6]
 0x000000000000000c (INIT)               0x4000
 0x000000000000000d (FINI)               0x14d0c
 0x000000000000001d (RUNPATH)            Library runpath: [/usr/lib/x86_64-linux-gnu]
 ...
```

This is what tells the dynamic linker which `.so` files to load when you run the program.

### See linked libraries

```
$ ldd /usr/bin/ls
        linux-vdso.so.1 (0x00007ffd5cdc8000)
        libselinux.so.1 => /lib/x86_64-linux-gnu/libselinux.so.1 (0x00007f3e58200000)
        libc.so.6 => /lib/x86_64-linux-gnu/libc.so.6 (0x00007f3e58000000)
        /lib64/ld-linux-x86-64.so.2 (0x00007f3e58440000)
        libpcre2-8.so.0 => /lib/x86_64-linux-gnu/libpcre2-8.so.0 (0x00007f3e57f00000)
```

This is what `ldd` actually does: it asks the dynamic linker which libraries the binary will load, without actually running the program.

### Compile a C file to assembly

Make a tiny C file:

```
$ cat > hello.c <<'EOF'
#include <stdio.h>
int main(void) {
    int x = 5;
    int y = 7;
    printf("%d\n", x + y);
    return 0;
}
EOF
```

Now compile to assembly (don't assemble, just stop at the `.s` file):

```
$ gcc -S -O2 hello.c -o hello.s && cat hello.s | head -30
        .file   "hello.c"
        .text
        .section        .rodata.str1.1,"aMS",@progbits,1
.LC0:
        .string "%d\n"
        .section        .text.startup,"ax",@progbits
        .p2align 4
        .globl  main
        .type   main, @function
main:
.LFB23:
        .cfi_startproc
        endbr64
        movl    $12, %esi
        leaq    .LC0(%rip), %rdi
        xorl    %eax, %eax
        xorl    %edi, %edi
        ...
```

Notice the compiler already computed `5 + 7 = 12` at compile time and just baked the answer in. That's optimization. With `-O0` (no optimization) it would actually do the addition at runtime.

### Compile to Intel-syntax assembly

```
$ gcc -O2 -masm=intel -S hello.c -o hello-intel.s
$ head -30 hello-intel.s
        .file   "hello.c"
        .intel_syntax noprefix
        .text
        .section        .rodata.str1.1,"aMS",@progbits,1
.LC0:
        .string "%d\n"
        .section        .text.startup,"ax",@progbits
        .p2align 4
        .globl  main
        .type   main, @function
main:
.LFB23:
        .cfi_startproc
        endbr64
        mov     esi, 12
        lea     rdi, .LC0[rip]
        xor     eax, eax
        ...
```

Easier to read for most people. Same instructions.

### Compile to LLVM IR (a different intermediate language)

```
$ clang -S -emit-llvm hello.c -o hello.ll
$ head -20 hello.ll
; ModuleID = 'hello.c'
source_filename = "hello.c"
target datalayout = "..."
target triple = "x86_64-pc-linux-gnu"

@.str = private unnamed_addr constant [4 x i8] c"%d\0A\00"

define dso_local i32 @main() #0 {
  ...
}
```

LLVM IR is what `clang` uses internally. It's higher-level than assembly but lower-level than C.

### Assemble and link by hand

If you have a `hello.s` file:

```
$ as hello.s -o hello.o
$ ld hello.o -o hello
```

(For real programs you also need to link in the C library, which is more involved. This works for tiny programs that only use direct syscalls.)

### Disassemble main()

```
$ gdb /usr/bin/ls -ex "disassemble main" -ex "quit" 2>&1 | head -30
Reading symbols from /usr/bin/ls...
Dump of assembler code for function main:
   0x0000000000006ab0 <+0>:     endbr64
   0x0000000000006ab4 <+4>:     push   %r15
   0x0000000000006ab6 <+6>:     push   %r14
   0x0000000000006ab8 <+8>:     push   %r13
   0x0000000000006aba <+10>:    push   %r12
   0x0000000000006abc <+12>:    push   %rbp
   0x0000000000006abd <+13>:    push   %rbx
   ...
```

The `-ex` flag tells gdb to run a command without entering interactive mode. We disassemble main and quit.

### See registers in gdb

```
$ gdb /usr/bin/ls -ex "break main" -ex "run /tmp" -ex "info registers" -ex "quit" 2>&1 | head -20
```

This sets a breakpoint at `main`, runs `ls /tmp`, hits the breakpoint, prints registers, then quits.

### Find main with grep

```
$ objdump -d /usr/bin/ls | grep -A 10 "<main>:"
0000000000006ab0 <main>:
    6ab0:       f3 0f 1e fa             endbr64
    6ab4:       41 57                   push   %r15
    6ab6:       41 56                   push   %r14
    6ab8:       41 55                   push   %r13
    6aba:       41 54                   push   %r12
    6abc:       55                      push   %rbp
    6abd:       53                      push   %rbx
    6abe:       48 83 ec 38             sub    $0x38,%rsp
    6ac2:       89 fb                   mov    %edi,%ebx
```

The classic function prologue! `push` a bunch of callee-saved registers, then `sub` to allocate locals.

### Count syscalls of a real program

```
$ strace -c /usr/bin/ls /tmp 2>&1 | tail -20
% time     seconds  usecs/call     calls    errors syscall
------ ----------- ----------- --------- --------- -------------------
 25.83    0.000031          15         2           getdents64
 16.67    0.000020           4         5         3 openat
  8.33    0.000010           2         5           mmap
  7.50    0.000009           2         4           mprotect
  ...
------ ----------- ----------- --------- --------- -------------------
100.00    0.000120                    47         5 total
```

47 syscalls just to list /tmp. Each one is a `syscall` instruction in the assembly.

### Count library calls

```
$ ltrace -c /usr/bin/ls /tmp 2>&1 | tail -20
% time     seconds  usecs/call     calls      function
------ ----------- ----------- --------- --------------------
 22.75    0.000045         3         13 strlen
 15.66    0.000031         5          6 free
 13.13    0.000026         5          5 malloc
  ...
```

Each library call is a `call` instruction in the assembly that goes through the PLT.

### Count CPU events with perf

```
$ perf stat /usr/bin/ls /tmp 2>&1 | tail -20
 Performance counter stats for '/usr/bin/ls /tmp':

              0.92 msec task-clock                #    0.875 CPUs utilized
                 0      context-switches          #    0.000 /sec
                 0      cpu-migrations            #    0.000 /sec
               103      page-faults               #  111.957 K/sec
         3,287,141      cycles                    #    3.572 GHz
         3,492,186      instructions              #    1.06  insn per cycle
           682,318      branches                  #  741.541 M/sec
            14,937      branch-misses             #    2.19% of all branches
```

3.5 million instructions to list /tmp. That's a LOT of assembly.

### Disassemble an eBPF program

If there are loaded eBPF programs:

```
$ sudo bpftool prog list | head -10
$ sudo bpftool prog dump xlated id 1 2>&1 | head -30
   0: (b7) r0 = 0
   1: (95) exit
```

That's a real eBPF program disassembled. This one is trivial: set r0 (return value) to 0, exit. The simplest possible program.

### Look at processor info

```
$ cat /proc/cpuinfo | grep -E "^processor|^model name|^flags" | head -10
processor       : 0
model name      : Intel(R) Core(TM) i7-8550U CPU @ 1.80GHz
flags           : fpu vme de pse tsc msr pae ...
processor       : 1
model name      : Intel(R) Core(TM) i7-8550U CPU @ 1.80GHz
flags           : fpu vme de pse tsc msr pae ...
```

One block per logical CPU (counting hyperthreads).

### Check 32 vs 64 bit

```
$ getconf LONG_BIT
64
```

On a 64-bit system, this prints 64. On a 32-bit one, 32.

### Identify a binary

```
$ file /usr/bin/ls
/usr/bin/ls: ELF 64-bit LSB pie executable, x86-64, version 1 (SYSV), dynamically linked, interpreter /lib64/ld-linux-x86-64.so.2, BuildID[sha1]=..., for GNU/Linux 3.2.0, stripped
```

Tells you the architecture, whether it's 32 or 64-bit, whether it's stripped of symbols, the interpreter (dynamic linker), and the BuildID.

### See available architectures in gdb

```
$ gdb -ex "set arch i386:x86-64" -ex "show arch" -ex "quit" 2>&1
The target architecture is set to "i386:x86-64".
```

You can also try `i386`, `i386:x86-64:intel`, `aarch64`, `riscv:rv64`, etc. gdb supports cross-debugging across architectures.

## Common Confusions

### "Why are there two assembly syntaxes (Intel vs AT&T)?"

**The confusion:** You see x86_64 written two completely different ways and they look opposite.

**The fix:** Two syntaxes exist for historical reasons. Intel syntax is what the official Intel manuals use. AT&T syntax was used by Unix tools (gas, objdump default, gdb default) since the 1970s. They differ in three big ways:

| Feature              | Intel                  | AT&T                       |
|----------------------|------------------------|----------------------------|
| Operand order        | `mov dst, src`         | `mov src, dst` (opposite!) |
| Register prefix      | none                   | `%`                         |
| Immediate prefix     | none                   | `$`                         |
| Memory               | `[base+idx*4+8]`       | `8(%base, %idx, 4)`        |

So `mov rax, 7` in Intel is `mov $7, %rax` in AT&T. Same instruction, just typed backwards. Most people learn Intel because it matches the docs. But you need to recognize AT&T when you see it (because gdb and objdump default to it). Always run `set disassembly-flavor intel` in gdb on day one.

### "Why does ARM64 sometimes use x0 and sometimes w0 for the same register?"

**The confusion:** Same register, two names, you can't tell when to use which.

**The fix:** `x0` is the full 64-bit register. `w0` is the lower 32 bits. They literally share the same physical hardware. If you write to `w0`, the upper 32 bits of `x0` are automatically zeroed (a nice safety feature). If you read `w0`, you get the bottom half of `x0`. Use `w0` when you're working with 32-bit values. Use `x0` when you're working with 64-bit values or pointers.

### "Why is RISC-V called 'open'?"

**The confusion:** "Open source" usually means software, but RISC-V is an instruction set, not a piece of software.

**The fix:** "Open" here means the **specification** is freely available with no licensing fees. Anyone can build a RISC-V CPU without paying royalties to anyone. Compare this to x86_64 (only Intel and AMD can legally make them) or ARM (you have to pay Arm Holdings for a license). The CPUs you actually buy are still made by various companies and might not be open source themselves; but the design they're based on is. This means tons of universities, hobbyists, and startups are designing RISC-V CPUs.

### "Why doesn't eBPF have a stack pointer?"

**The confusion:** Every other CPU has a stack pointer. eBPF has r10, which is described as a "frame pointer," but it's read-only. So how does eBPF do recursion? How does it use the stack at all?

**The fix:** eBPF does use a stack — it's a fixed 512-byte region that the kernel sets up before the program starts. But because the verifier needs to prove all stack accesses are within bounds, the stack pointer can't be modified by the program. Instead, `r10` is a constant frame pointer, and stack accesses are always relative to it. This way the verifier always knows exactly where the stack lives. eBPF programs cannot recurse for similar reasons (the verifier can't prove a recursive program will terminate). For deeper "function-like" structure, eBPF uses **tail calls** (jump to another program rather than calling) or **bounded loops**.

### "Why is x86_64 instruction length variable?"

**The confusion:** ARM and RISC-V have nice fixed 4-byte instructions, but x86 instructions can be anywhere from 1 to 15 bytes. Why?

**The fix:** History. x86 started as a 16-bit architecture in 1978 with simple, often 1-byte instructions. Each generation added new features by **prefixing** existing instructions or **extending** their encoding. Instead of throwing the old instructions away, every new variant got a new prefix or escape byte. By the time we got to 64-bit, instructions had accumulated tons of optional pieces: legacy prefixes, REX prefix (for 64-bit), opcode bytes (sometimes 1, 2, or 3 of them), ModRM byte, SIB byte, displacement, immediate. All optional. Result: variable length. RISC architectures, designed from scratch, picked one fixed size.

### "Why do I see different syscall numbers on different architectures?"

**The confusion:** "write" is syscall 1 on x86_64 but syscall 64 on ARM64. Why is the same Linux using different numbers?

**The fix:** When Linux added a new architecture, it sometimes inherited a syscall numbering scheme from an older OS or a similar architecture. x86_64's syscalls inherited mostly from x86 (and from System V Unix conventions). ARM64 used a newer "generic" numbering that the kernel maintainers prefer for new architectures. RISC-V uses the same generic numbering as ARM64. There is no good reason for the divergence other than history. The C library hides this — when you call `write()` from C, libc figures out which number to use on the current architecture.

### "What's the difference between a register and a variable?"

**The confusion:** In C, you have variables. In assembly, you have registers. They sound like the same thing.

**The fix:** A variable in C is a name you choose for a value, and the compiler decides where to store it (sometimes a register, sometimes the stack, sometimes RAM). A register in assembly is an actual physical storage location on the CPU chip itself, and there's a fixed small number of them. Variables are abstract; registers are concrete. When the compiler does its job, every variable in your program ends up either in a register (for hot, frequently-used values) or in memory (for things that don't fit, or for global state).

### "Why does writing to eax zero the top half of rax, but writing to al doesn't zero the rest?"

**The confusion:** It's the same register family. Why don't all sub-register writes zero-extend?

**The fix:** This is a famous x86_64 oddity. When AMD designed x86_64, they wanted writes to 32-bit registers (`mov eax, 1`) to zero-extend to 64 bits, partly to break dependency chains in the out-of-order pipeline, and partly because most code writes 32-bit values most of the time. But for backward compatibility with 32-bit code, they kept the old behavior for 16-bit (`mov ax, 1`) and 8-bit (`mov al, 1`) writes — those preserve the upper bits. Yes, it's inconsistent. Yes, it bites people who don't expect it. Welcome to x86_64.

### "Why does the stack grow downward?"

**The confusion:** "Downward" sounds backwards. Why not grow up like a normal stack of plates?

**The fix:** Convention from very early CPUs. The thinking was: programs grow upward (text and data near the bottom of memory), and we don't want them to collide. So put the stack at the top of memory and have it grow toward the program from the other direction. Modern systems with virtual memory don't really care which way it grows, but the convention stuck. Some systems (some embedded CPUs, some old machines) do grow upward. Linux on x86_64, ARM64, and RISC-V grows downward.

### "Why are there both signed and unsigned branches?"

**The confusion:** "less than" is "less than," right?

**The fix:** Not in two's complement! In an unsigned interpretation, the byte 0xFF is 255 (a big number). In a signed interpretation, 0xFF is -1 (a small number). So "is 0xFF less than 1?" depends on interpretation: signed yes (-1 < 1), unsigned no (255 > 1). The CPU has different branch instructions for each interpretation: `jl`/`jg` for signed, `jb`/`ja` for unsigned (x86_64). ARM64 uses `lt`/`gt` for signed, `lo`/`hi` for unsigned. Picking the wrong one is a classic security bug — and a classic source of bugs that "work" until you hit a case with negative numbers.

### "Is 'mov' a copy or a move?"

**The confusion:** The instruction is called "mov" (move), but the source register isn't cleared.

**The fix:** It's actually a copy. After `mov rax, rbx`, both `rax` and `rbx` hold the same value; `rbx` was not erased. The name is misleading. Some architectures call it `cpy` or `copy`, but `mov` is the historical name on x86 and most others.

### "Why does my function not show up in 'nm'?"

**The confusion:** You compile with optimizations, look for your function, and it's missing.

**The fix:** Two things might have happened. First, if the compiler **inlined** the function (replaced calls to it with the function's body directly), there's no separate function to have a symbol for. Second, if the binary was **stripped** (had its symbol table removed to save space), `nm` finds nothing. Try `objdump --syms` or compile with `-g` (debug info) to keep symbols.

### "What does 'callee-saved' versus 'caller-saved' actually mean?"

**The confusion:** Both involve someone saving something. What's the difference?

**The fix:** It's about responsibility, not about who literally pushes the bytes.
- **Caller-saved** (also called "volatile" or "scratch"): if the **caller** wants to keep a value across a function call, the caller has to save it before the call. The callee can scribble all over these registers and won't restore them.
- **Callee-saved** (also called "non-volatile" or "preserved"): if the **callee** wants to use these registers, the callee has to save them at the start of the function and restore them at the end. From the caller's point of view, these registers will look unchanged after the call.

The convention is fixed for each architecture so any code can call any other code safely.

### "Why do I sometimes see code that looks like it could be in two architectures at once?"

**The confusion:** Some Linux binaries seem to have x86_64 AND ARM64 code in them.

**The fix:** Those are "fat" or "universal" binaries — actual concatenations of two architecture-specific binaries with a header that says which one to use. Apple uses this for "Universal" macOS binaries that run on both Intel and Apple Silicon. Linux has multi-arch packaging but doesn't usually fat-pack at the file level. Most Linux binaries are single-architecture.

### "What's the difference between RIP and an address?"

**The confusion:** "RIP-relative" sounds like an address, but RIP is the program counter.

**The fix:** RIP holds the address of the **next** instruction to execute. So when the CPU is mid-instruction, RIP already points past the current one. "RIP-relative addressing" means "compute an address as RIP + some offset." This is how position-independent code works — instead of a hard-coded address, you say "the data I want is 200 bytes after the current instruction." Wherever the code gets loaded in memory, the offset stays correct.

### "If RISC has fewer instructions, why are RISC binaries sometimes bigger than CISC?"

**The confusion:** RISC = "reduced instruction set," so shouldn't RISC programs be smaller?

**The fix:** RISC has fewer **kinds** of instructions, but each instruction is fixed size (usually 4 bytes on ARM64 and RISC-V). CISC has more kinds, with variable size (1-15 bytes on x86_64). For simple operations, x86_64 might use 1-3 byte instructions, while ARM64 always uses 4 bytes. So a RISC binary often has more instructions, each at fixed size, totalling more bytes. But the instructions are simpler and more uniform, which makes the CPU easier to design and the pipeline easier to keep full.

## Vocabulary

| Term | Plain English |
|------|---------------|
| **AArch64** | The official name for ARM 64-bit. Same as ARM64. |
| **ABI** | Application Binary Interface. The rules for how compiled code interacts (calling conventions, struct layouts, etc.). |
| **Address** | A number that names a specific byte in memory. |
| **AddressSanitizer** | A debugging tool that catches memory errors at runtime by instrumenting addresses. |
| **ALU** | Arithmetic Logic Unit. The CPU part that does math and logic. |
| **AMD64** | Another name for x86_64. AMD invented the 64-bit extension to x86. |
| **Architecture** | A family of CPUs that all understand the same instruction set. |
| **ARM64** | The 64-bit ARM architecture. Used in phones, Macs, and many cloud servers. |
| **as** | The GNU assembler. Turns assembly text into machine code. |
| **Assembler** | A program that translates assembly into machine code. |
| **Assembly** | Human-readable text form of machine code. One assembly line = one (or two) CPU instructions. |
| **AT&T syntax** | One of the two x86 assembly syntaxes. Source first, `%` for registers, `$` for immediates. |
| **AVX** | Advanced Vector Extensions. SIMD instructions on Intel/AMD CPUs (256 or 512 bits). |
| **Binary** | The compiled form of a program (machine code). Also: a number system with only 0 and 1. |
| **Bit** | A single 0 or 1. The smallest piece of information. |
| **BPF** | Berkeley Packet Filter. The original packet-filter VM. |
| **Branch** | An instruction that changes which instruction the CPU will run next. |
| **Branch prediction** | The CPU guessing which way a branch will go before it knows for sure, to keep the pipeline full. |
| **Byte** | 8 bits. The smallest addressable unit on most CPUs. |
| **C** | A programming language. The classic high-level language that maps closely to assembly. |
| **Calling convention** | The rules for how function calls work at the assembly level. |
| **Caller-saved** | A register the called function can clobber freely. The caller saves it if needed. |
| **Callee-saved** | A register the called function must preserve. |
| **Cache** | Fast memory inside or near the CPU that stores recently-used data. |
| **CISC** | Complex Instruction Set Computer. Lots of instructions, often complicated ones. x86 is CISC. |
| **clang** | The LLVM C/C++ compiler. |
| **Compile** | Translate from high-level code to assembly or machine code. |
| **Compiler** | A program that compiles. |
| **Condition code** | A bit in the flags register set by an operation. |
| **Core** | A complete CPU unit. Modern chips have multiple cores. |
| **Core dump** | A snapshot of a crashed process's memory, used for debugging. |
| **CPU** | Central Processing Unit. The chip that runs all the instructions. |
| **Cross-compile** | Compile on one architecture for a different target architecture. |
| **CSR** | Control and Status Register. RISC-V's special registers for system configuration. |
| **Disassemble** | Translate machine code back into readable assembly. |
| **Disassembler** | A program that disassembles. `objdump`, `gdb`, `radare2`. |
| **Driver** | Kernel code that talks to a piece of hardware. |
| **Dynamic linking** | Linking some library code at program-load time, not at compile time. |
| **eBPF** | Extended Berkeley Packet Filter. A virtual instruction set for the Linux kernel. |
| **ELF** | Executable and Linkable Format. The standard binary format on Linux. |
| **Endianness** | The order of bytes within a multi-byte value. Little-endian or big-endian. |
| **Entry point** | The address where program execution starts. |
| **Executable** | A runnable binary file. |
| **Flag** | A 1-bit signal in the flags register, set by previous operations. |
| **FLAGS / RFLAGS** | The x86 flags register. |
| **fp** | Frame pointer. ARM64's `x29` and RISC-V's `s0`. |
| **Frame pointer** | A register that points to the current function's stack frame base. |
| **Function** | A reusable chunk of code, callable from elsewhere. |
| **gcc** | The GNU Compiler Collection. The classic Linux C/C++ compiler. |
| **gdb** | GNU Debugger. The standard Linux debugger. |
| **GP register** | General-purpose register. The "boxes" that hold values for normal computation. |
| **Heap** | A region of memory for dynamic allocation. Managed by `malloc()`/`free()` in C. |
| **Helper** | A kernel-provided function eBPF programs can call. |
| **Hex** | Base-16 number system. Compact way to write binary. `0xFF` = 255. |
| **Immediate** | A constant value baked directly into an instruction (vs. coming from a register or memory). |
| **Instruction** | One step the CPU executes. |
| **Instruction pointer** | The register that holds the address of the next instruction. RIP/PC. |
| **Intel syntax** | One of the two x86 assembly syntaxes. Destination first, no `%` or `$` prefixes. |
| **JIT** | Just-In-Time compilation. Compiling code at runtime. |
| **Kernel** | The core of an operating system. See `cs ramp-up linux-kernel-eli5`. |
| **Label** | A name for an address in assembly. Used as a target for branches and calls. |
| **LDR / STR** | ARM64 load and store instructions. |
| **lea** | x86's "load effective address." Computes an address but doesn't access memory. |
| **Library** | A bundle of reusable compiled code. `.so` (dynamic) or `.a` (static) on Linux. |
| **Linker** | A program that combines object files into an executable. `ld` on Linux. |
| **Little-endian** | LSB at lowest address. x86 and ARM64's default. |
| **lldb** | The LLVM debugger. Apple's default debugger. |
| **Load** | Read a value from memory into a register. |
| **LLVM** | A compiler infrastructure. Backend for clang, swiftc, rustc. |
| **LLVM IR** | LLVM's intermediate representation, between source and machine code. |
| **lr** | ARM64's link register. Holds the return address after `bl`. |
| **Machine code** | The actual byte sequence the CPU reads. |
| **Memory** | RAM. The computer's working storage. |
| **MMU** | Memory Management Unit. CPU hardware that translates virtual addresses to physical. |
| **mov** | The "move" instruction. Actually copies, doesn't move. |
| **NEON** | ARM's SIMD extension. 128-bit vector operations. |
| **nm** | A tool that lists symbols in object files and binaries. |
| **NOP** | "No operation." An instruction that does nothing. Used for padding. |
| **objdump** | A tool that dumps info from object files. Includes a disassembler. |
| **Object file** | The compiled output of one source file, before linking. `.o` extension. |
| **Opcode** | The numeric code that tells the CPU which instruction to run. |
| **Operand** | An input to an instruction. A register, immediate, or memory address. |
| **Operating system** | The base software that manages a computer (kernel + tools). |
| **PC** | Program counter. ARM64 and RISC-V's name for the instruction pointer. |
| **perf** | A Linux performance analysis tool. Reads CPU performance counters. |
| **PIC** | Position-Independent Code. Code that works at any load address. |
| **PIE** | Position-Independent Executable. An executable made of PIC. |
| **Pipeline** | The CPU stage where instructions flow through fetch/decode/execute. |
| **PLT** | Procedure Linkage Table. Trampolines for dynamically-linked function calls. |
| **Pop** | Remove the top item from the stack. |
| **Program counter** | See PC. |
| **Push** | Add an item to the top of the stack. |
| **r0-r10** | eBPF's 11 registers. |
| **RAM** | Random Access Memory. The computer's main memory. |
| **rax** | The "accumulator" 64-bit register on x86_64. Used for syscall numbers and return values. |
| **readelf** | A tool that displays ELF file information. |
| **Register** | A tiny, super-fast storage location inside the CPU itself. |
| **Relocation** | A note from the assembler saying "fill in this address later." |
| **RIP** | x86_64's instruction pointer register. |
| **RISC** | Reduced Instruction Set Computer. Few, simple instructions. ARM and RISC-V are RISC. |
| **RISC-V** | The open-source RISC instruction set. |
| **rsp** | x86_64's stack pointer. |
| **rsi / rdi** | x86_64's source/destination index registers, used for the first two function args. |
| **Section** | A named region in an ELF file (.text, .data, .bss, etc.). |
| **SIMD** | Single Instruction Multiple Data. Doing the same op on a vector of values at once. |
| **sp** | Stack pointer. ARM64 and RISC-V's name for it. |
| **SSE** | Streaming SIMD Extensions. The original SIMD on x86. |
| **Stack** | A region of memory used for function calls, locals, and saved registers. |
| **Stack frame** | One function's chunk of the stack. |
| **Stack pointer** | The register that points to the top of the stack. |
| **Static linking** | Including all library code directly in the executable. |
| **Stepi** | A gdb command: execute one instruction. |
| **Store** | Write a value from a register to memory. |
| **strace** | A tool that traces all syscalls a program makes. |
| **Symbol** | A name for a function or variable. Stored in the symbol table. |
| **Symbol table** | A list of names and their addresses in a binary. |
| **syscall** | A system call. The way user programs ask the kernel to do things. Also: the x86_64 instruction. |
| **Time slice** | The amount of time the scheduler gives one process before switching to another. |
| **TLS** | Thread-Local Storage. Per-thread variables. (Also: Transport Layer Security, but that's different.) |
| **Two's complement** | The standard way to represent signed integers in binary. |
| **User space** | Where regular programs run. Limited privileges. |
| **Variable-length instruction** | An instruction whose size varies from 1 to N bytes (vs. fixed-length). |
| **Verifier** | The eBPF subsystem that proves programs are safe before running them. |
| **Virtual memory** | The illusion that each process has its own address space. |
| **VLIW** | Very Long Instruction Word. A CPU style where one "instruction" packs multiple operations. |
| **Word** | An architecture-dependent natural integer size. On most modern systems, 32 or 64 bits. |
| **x0-x30** | ARM64's 31 general-purpose registers. |
| **x86** | A family of architectures dating to 1978. 16-bit, then 32-bit (i386), now 64-bit (x86_64). |
| **x86_64** | The 64-bit extension to x86. AMD invented it. |
| **xchg** | Exchange. Swap two registers. |
| **xor reg, reg** | Idiomatic way to zero a register on x86_64. |
| **xzr / wzr** | ARM64's zero register. Reads as 0, writes are discarded. |
| **YMM / ZMM** | Wider versions of XMM. 256-bit (YMM, AVX) and 512-bit (ZMM, AVX-512). |
| **zero register** | A register hardwired to always read as 0. RISC-V's `x0`, ARM64's `xzr`. |
| **Zero-extend** | Fill the upper bits of a wider register with zeros. |
| **Sign-extend** | Fill the upper bits with copies of the high bit (preserves negative values). |

## Try This

These experiments take 5-30 minutes each.

### Experiment 1: Watch a single C statement become assembly

```
$ cat > test.c <<'EOF'
int main(void) {
    int a = 3;
    int b = 4;
    return a + b;
}
EOF

$ gcc -O0 -S test.c -o test.s
$ cat test.s
```

Compare with the optimized version:

```
$ gcc -O2 -S test.c -o test-opt.s
$ cat test-opt.s
```

The optimized version probably has just `mov eax, 7; ret` because the compiler did the math at compile time.

### Experiment 2: Single-step a real program in gdb

Compile a tiny program with debug info:

```
$ gcc -g test.c -o test
$ gdb ./test
(gdb) set disassembly-flavor intel
(gdb) layout split
(gdb) break main
(gdb) run
(gdb) si
(gdb) si
(gdb) info registers
```

Each `si` runs one assembly instruction. Watch the registers change. Notice how `rip` increments by the size of each instruction.

### Experiment 3: Watch syscalls during a real command

```
$ strace -e trace=openat,read,write,close cat /etc/hostname
```

You'll see every file the program opens and reads, and every output it writes.

### Experiment 4: Find a function in a stripped binary

```
$ strip /tmp/test
$ nm /tmp/test
nm: /tmp/test: no symbols
$ objdump -d /tmp/test | head -20
```

The disassembly is still there; just no names. This is what reverse engineers face.

### Experiment 5: Compile for ARM64 (if you have a cross-compiler)

```
$ apt install gcc-aarch64-linux-gnu     # or your distro's equivalent
$ aarch64-linux-gnu-gcc -O2 -S test.c -o test-arm64.s
$ cat test-arm64.s
```

Compare the ARM64 output with x86_64. Notice the `bl`/`ret` pattern with `lr`, the fixed instruction sizes, the different register names.

### Experiment 6: Look at a minimal eBPF program

```
$ sudo bpftool prog list 2>&1 | head
$ sudo bpftool prog dump xlated id <some_id> 2>&1
```

If you don't have any loaded eBPF programs, run something that uses them:

```
$ sudo bpftrace -e 'BEGIN { printf("hello eBPF\n"); exit(); }'
```

### Experiment 7: Count instructions in a program

```
$ perf stat -e instructions /usr/bin/ls /tmp 2>&1 | tail -10
```

Now do it with a more complex program:

```
$ perf stat -e instructions /bin/ls -la /usr/bin 2>&1 | tail -10
```

Compare. Notice how the instruction count scales with the work done.

### Experiment 8: Recognize x86 syntax flavors

Take a friend's piece of disassembly. Look for `%` signs and `$` signs. If you see them, it's AT&T. If not, it's Intel. Mentally translate each instruction by flipping the operand order.

## Where to Go Next

- `cs fundamentals x86-64-assembly` — dense x86_64 reference
- `cs fundamentals arm64-architecture` — ARM64 reference
- `cs fundamentals risc-v` — RISC-V reference
- `cs fundamentals ebpf-bytecode` — eBPF dense
- `cs system gdb`, `cs system strace`, `cs system perf`
- `cs ramp-up ebpf-eli5` — eBPF in the same plain-English voice as this sheet
- `cs ramp-up linux-kernel-eli5` — the kernel, in plain English
- `cs ramp-up binary-numbering-eli5` — the counting math underneath everything

## See Also

- `fundamentals/x86-64-assembly`
- `fundamentals/arm64-architecture`
- `fundamentals/risc-v`
- `fundamentals/ebpf-bytecode`
- `fundamentals/binary-and-number-systems`
- `system/gdb`
- `system/strace`
- `ramp-up/ebpf-eli5`
- `ramp-up/linux-kernel-eli5`
- `ramp-up/binary-numbering-eli5`

## References

- "Computer Systems: A Programmer's Perspective" by Bryant & O'Hallaron
- "Programming from the Ground Up" by Jonathan Bartlett (free)
- "ARM Cortex-A Series Programmer's Guide for ARMv8-A"
- "RISC-V Instruction Set Manual" (Volume I: Unprivileged ISA)
- "BPF Performance Tools" by Brendan Gregg
- man as, man ld, man nm, man objdump, man readelf, man gdb
- "Intel(R) 64 and IA-32 Architectures Software Developer's Manual"
- "AArch64 Instruction Set Architecture for Armv8-A"
- godbolt.org Compiler Explorer (in-browser; mention but stay terminal)
