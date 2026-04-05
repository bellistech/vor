# How Computers Work (From Electrons to Applications)

A tiered guide to understanding how computers work -- from the simplest analogy to college-level architecture.

## ELI5 (Explain Like I'm 5)

### The Kitchen Analogy

A computer is like a kitchen where a chef cooks meals:

- **CPU (Chef):** The person who does all the thinking and cooking. Can only do one step at a time, but does it really fast
- **RAM (Counter space):** Where the chef puts ingredients they're using RIGHT NOW. Bigger counter = more things at once. When you turn off the stove, the counter gets cleared
- **Hard drive / SSD (Fridge/pantry):** Stores all your ingredients long-term. Slower to get things from the fridge than the counter, but holds way more
- **Motherboard (Kitchen floor/walls):** Connects everything together so the chef can reach the fridge, counter, and oven
- **Power supply (Electricity):** Without it, nothing works
- **Monitor (Serving window):** Where you see the finished food
- **Keyboard/Mouse (Order tickets):** How you tell the chef what to cook
- **Operating system (Recipe book + kitchen manager):** Tells the chef which recipes to follow and in what order

### How It Actually Thinks

Computers only understand two things: ON and OFF (like a light switch). Everything -- every picture, song, game, and video -- is made of millions of ON/OFF switches called **bits**. ON = 1, OFF = 0.

### What Happens When You Click an Icon

1. Mouse sends a signal: "click happened at position X, Y"
2. Operating system figures out what icon is at that spot
3. OS tells CPU: "load and run this program"
4. CPU reads the program instructions from the hard drive into RAM
5. CPU follows instructions one by one, really fast
6. Results get sent to the screen so you can see them

## Middle School

### Binary -- The Language of Computers

Computers use base-2 (binary) instead of base-10 (decimal):

```
# Decimal (base 10): each digit is a power of 10
  425 = 4x100 + 2x10 + 5x1

# Binary (base 2): each digit is a power of 2
  1101 = 1x8 + 1x4 + 0x2 + 1x1 = 13

# Powers of 2
  2^0 = 1      2^4 = 16     2^8 = 256
  2^1 = 2      2^5 = 32     2^9 = 512
  2^2 = 4      2^6 = 64     2^10 = 1024
  2^3 = 8      2^7 = 128    2^16 = 65536

# Converting decimal 42 to binary:
  42 / 2 = 21 remainder 0
  21 / 2 = 10 remainder 1
  10 / 2 = 5  remainder 0
   5 / 2 = 2  remainder 1
   2 / 2 = 1  remainder 0
   1 / 2 = 0  remainder 1
  Read remainders bottom-up: 101010
```

### Logic Gates -- How Chips Think

Transistors are tiny electronic switches. Combine them to make logic gates:

```
# AND gate: output is 1 only if BOTH inputs are 1
  0 AND 0 = 0
  0 AND 1 = 0
  1 AND 0 = 0
  1 AND 1 = 1

# OR gate: output is 1 if EITHER input is 1
  0 OR 0 = 0
  0 OR 1 = 1
  1 OR 0 = 1
  1 OR 1 = 1

# NOT gate: flips the input
  NOT 0 = 1
  NOT 1 = 0

# XOR gate: output is 1 if inputs are DIFFERENT
  0 XOR 0 = 0
  0 XOR 1 = 1
  1 XOR 0 = 1
  1 XOR 1 = 0
```

### How a CPU Adds Two Numbers

A **half adder** adds two single-bit numbers using XOR (for the sum) and AND (for the carry):

```
# Half adder truth table
  A  B  | Sum (A XOR B) | Carry (A AND B)
  0  0  |      0        |       0
  0  1  |      1        |       0
  1  0  |      1        |       0
  1  1  |      0        |       1

# Chain half adders into a full adder to add multi-bit numbers
# 8 full adders = can add two 8-bit numbers (0-255)
```

### What RAM Physically Is

- RAM chips contain billions of tiny capacitors (like micro batteries)
- Each capacitor stores one bit: charged = 1, discharged = 0
- Capacitors leak charge, so RAM must be refreshed thousands of times per second (that's the "Dynamic" in DRAM)
- When you lose power, all capacitors discharge -- that's why RAM is volatile

### Data Sizes

```
# 1 bit          = single 0 or 1
# 1 byte         = 8 bits (can represent 0-255 or one ASCII character)
# 1 kilobyte     = 1,024 bytes        (a short email)
# 1 megabyte     = 1,024 KB           (a photo or short song)
# 1 gigabyte     = 1,024 MB           (a movie)
# 1 terabyte     = 1,024 GB           (a large hard drive)
```

## High School

### Von Neumann Architecture

Most computers follow the Von Neumann model (1945):

```
# Key idea: programs and data share the same memory

  +------------------+
  |      CPU         |
  |  +-----+ +-----+ |
  |  | ALU | | CU  | |     ALU = Arithmetic Logic Unit (does math)
  |  +-----+ +-----+ |     CU  = Control Unit (fetches/decodes instructions)
  |  +-----------+   |
  |  | Registers |   |     Registers = tiny, ultra-fast storage inside CPU
  |  +-----------+   |
  +--------+---------+
           |
      +----+----+  <-- System Bus (address + data + control lines)
      |         |
  +---+---+ +---+---+
  |  RAM  | |  I/O  |     I/O = keyboard, display, disk, network
  +-------+ +-------+

# Von Neumann bottleneck: CPU is faster than memory,
# so it often waits for data to arrive from RAM
```

### The Fetch-Decode-Execute Cycle

Every instruction goes through this cycle:

```
# 1. FETCH    -- CU reads the next instruction from RAM
#                using the Program Counter (PC) register
# 2. DECODE   -- CU figures out what the instruction means
#                (what operation? which registers? what data?)
# 3. EXECUTE  -- ALU performs the operation
#                (add, subtract, compare, move data, jump)
# 4. STORE    -- Result is written back to a register or RAM
# 5. PC += 1  -- Program counter advances to the next instruction
#                (unless a jump/branch changed it)

# This cycle repeats billions of times per second
# A 3 GHz CPU does ~3 billion cycles per second
```

### Clock Speed and Performance

```
# Clock speed = how many cycles per second (measured in GHz)
# 1 GHz = 1 billion cycles per second

# But clock speed != instructions per second because:
# - Some instructions take multiple cycles (multiply vs add)
# - Modern CPUs execute multiple instructions per cycle (IPC)
# - Pipeline stalls, cache misses, and branch mispredictions waste cycles

# Actual performance = Clock speed x IPC x Core count (simplified)
# A 3 GHz chip with IPC of 4 = up to 12 billion operations/sec
```

### Cache Hierarchy

```
# Problem: CPU is ~100x faster than RAM
# Solution: small, fast memory (cache) between CPU and RAM

# L1 Cache    ~64 KB     ~1 ns     per core, split I-cache + D-cache
# L2 Cache    ~256 KB    ~4 ns     per core
# L3 Cache    ~8-64 MB   ~10 ns    shared across cores
# RAM         ~16-64 GB  ~100 ns   shared, off-chip
# SSD                    ~100 us   persistent storage
# HDD                    ~10 ms    persistent storage (mechanical)

# Each level is slower but larger
# Cache hit  = data found in cache (fast)
# Cache miss = data not in cache, must fetch from slower level
# Typical L1 hit rate: 95-99%
```

### How Programs Become Machine Code

```
# Source code (human-readable)
#   |
#   v  Preprocessor (C/C++: expands #include, #define)
#   |
#   v  Compiler (translates to assembly language)
#   |
#   v  Assembler (translates assembly to machine code / object files)
#   |
#   v  Linker (combines object files + libraries into executable)
#   |
#   v  Executable binary (machine code the CPU runs directly)

# Interpreted languages (Python, JS):
#   Source -> Interpreter reads and executes line by line
#   Or: Source -> Bytecode -> Virtual Machine executes bytecode

# JIT (Just-In-Time) compilation (Java, C#, JS V8):
#   Source -> Bytecode -> VM profiles hot paths -> compiles to native
```

### Operating System Basics

```
# The OS is software that manages hardware and runs programs

# Kernel: core of the OS, runs in privileged/supervisor mode
# - Process management: creates, schedules, kills processes
# - Memory management: gives each process its own virtual address space
# - File system: organizes data on disk into files and directories
# - Device drivers: translates OS requests into hardware commands
# - System calls: API for programs to request kernel services
#   (open a file, allocate memory, send network packet)

# User space vs kernel space:
# - Programs run in user space (restricted, can't touch hardware)
# - When a program needs hardware, it makes a system call
# - CPU switches to kernel mode, kernel does the work, returns result
# - This separation prevents buggy programs from crashing the system
```

## College

### Pipelining

```
# Break instruction execution into stages that overlap:
#
#  Time -->  1  2  3  4  5  6  7  8
#  Instr 1: IF ID EX ME WB
#  Instr 2:    IF ID EX ME WB
#  Instr 3:       IF ID EX ME WB
#  Instr 4:          IF ID EX ME WB
#
# IF=Instruction Fetch, ID=Instruction Decode,
# EX=Execute, ME=Memory Access, WB=Write Back
#
# Without pipelining: 4 instructions x 5 cycles = 20 cycles
# With pipelining: 5 + (4-1) = 8 cycles
# Ideal speedup = number of pipeline stages (5x here)

# Pipeline hazards (things that break the flow):
# - Data hazard:   instruction needs result from previous (not ready yet)
#   Fix: forwarding/bypassing, stalling
# - Control hazard: branch changes PC, pipeline has wrong instructions
#   Fix: branch prediction, speculative execution
# - Structural hazard: two instructions need same hardware unit
#   Fix: duplicate hardware, stall
```

### Branch Prediction

```
# Problem: conditional branches aren't resolved until EX stage
#          pipeline must guess which way to go or stall

# Static prediction:
# - Always predict not-taken (simple, ~60% accuracy)
# - Always predict backward branches taken (loops, ~65%)

# Dynamic prediction:
# - 1-bit predictor: remember last outcome, predict same
# - 2-bit saturating counter: need 2 mispredictions to flip
#     States: Strongly Taken -> Weakly Taken -> Weakly Not -> Strongly Not
#     Accuracy: ~85-90%
# - Two-level adaptive: use pattern history of last N branches
#     Accuracy: ~95%
# - TAGE (modern): multiple tables indexed by different history lengths
#     Accuracy: ~97%+

# Cost of misprediction: flush pipeline, restart from correct path
# On a 15-stage pipeline, misprediction costs ~15 cycles
```

### Out-of-Order Execution

```
# CPU executes instructions in whatever order is efficient,
# not the order written in the program

# Tomasulo's algorithm (IBM, 1967):
# 1. Issue:    instruction enters reservation station
# 2. Execute:  waits for operands, then executes when ready
# 3. Write:    broadcasts result on Common Data Bus (CDB)

# Reorder Buffer (ROB): ensures results commit in program order
# even though execution is out of order
# This maintains "precise exceptions" -- if instruction 5 faults,
# instructions 1-4 are committed, 5+ are discarded

# Register renaming: eliminates false dependencies (WAR, WAW)
# Physical registers >> architectural registers
# Example: x86 has 16 architectural regs, but ~180 physical regs
```

### Superscalar Architectures

```
# Issue and execute multiple instructions per cycle

# 2-wide superscalar: can fetch, decode, issue 2 instructions/cycle
# 4-wide superscalar: 4 instructions/cycle (modern x86)
# 6-8 wide: Apple M-series, some server chips

# Limits to ILP (instruction-level parallelism):
# - True data dependencies (RAW hazards)
# - Branch mispredictions
# - Cache misses
# - Finite hardware resources (functional units, rename registers)
# - In practice, sustained IPC rarely exceeds 3-4 on general code
```

### Cache Coherence Protocols

```
# Problem: multiple cores have private L1/L2 caches
#          what if core 0 writes to address X while core 1 has it cached?

# MESI protocol (4 states per cache line):
# M (Modified):  this cache has only copy, it's dirty (changed)
# E (Exclusive): this cache has only copy, it's clean
# S (Shared):    multiple caches have clean copies
# I (Invalid):   cache line is not valid

# Write to Shared line:
#   1. Core sends invalidate message to all other caches
#   2. Other cores mark their copy Invalid
#   3. Writing core transitions to Modified
#   4. This is called "write-invalidate" protocol

# MOESI (AMD) adds O (Owned): dirty but shared, avoids writeback
# MESIF (Intel) adds F (Forward): one shared copy responds to requests

# False sharing: two cores write to different variables
#   on the same cache line -- causes constant invalidation
#   Fix: pad structures so hot variables are on separate cache lines
```

### Virtual Memory

```
# Every process gets its own virtual address space
# Addresses in the program != physical addresses in RAM

# Page table: maps virtual pages to physical frames
#   Virtual addr: [page number | offset]
#   Physical addr: [frame number | offset]
#   Typical page size: 4 KB (12-bit offset)

# Multi-level page tables (x86-64 uses 4 levels):
#   PML4 -> PDPT -> PD -> PT -> Physical frame
#   Each level is a table of 512 entries (9 bits each)
#   48-bit virtual address: 9+9+9+9+12

# Page fault: accessed page not in RAM
#   1. CPU traps to OS
#   2. OS finds the page on disk (swap)
#   3. OS loads it into a free frame (may evict another page)
#   4. OS updates page table, restarts instruction
```

### TLB (Translation Lookaside Buffer)

```
# Problem: every memory access needs page table lookup (4 levels!)
# Solution: TLB caches recent virtual-to-physical translations

# L1 TLB:  ~64 entries,  ~1 cycle lookup
# L2 TLB:  ~1024 entries, ~7 cycle lookup
# TLB miss: walk the page table (~100+ cycles)

# TLB reach = entries x page size
#   64 entries x 4KB = 256 KB covered by L1 TLB
#   Huge pages (2MB, 1GB) increase TLB reach dramatically
#   64 entries x 2MB = 128 MB covered

# Context switch flushes TLB (new process, new page tables)
# ASID (Address Space ID) tags avoid full flushes
```

### Memory-Mapped I/O and DMA

```
# Memory-Mapped I/O (MMIO):
#   Device registers appear as memory addresses
#   CPU reads/writes to special addresses to control devices
#   Example: writing to address 0xFE00_0000 sends data to GPU
#   x86 also has Port I/O (IN/OUT instructions) -- legacy

# DMA (Direct Memory Access):
#   Problem: CPU copying data byte-by-byte from disk is wasteful
#   Solution: DMA controller copies data directly to/from RAM
#
#   1. CPU tells DMA: source address, dest address, byte count
#   2. DMA takes over the bus and transfers data
#   3. DMA interrupts CPU when done
#   4. CPU is free to do other work during transfer
#
#   Used for: disk I/O, network packets, GPU buffers, audio
```

### Interrupt Handling

```
# Interrupts: hardware signals that demand CPU attention

# Types:
# - Hardware interrupt (IRQ): device signals CPU (keyboard, disk, NIC)
# - Software interrupt (trap): program executes INT instruction (syscall)
# - Exception: CPU detects error (divide by zero, page fault)

# Handling flow:
#   1. Device asserts interrupt line
#   2. CPU finishes current instruction
#   3. CPU saves state (PC, flags, registers) to stack
#   4. CPU looks up handler address in Interrupt Vector Table (IVT)
#      or Interrupt Descriptor Table (IDT on x86)
#   5. CPU jumps to handler (kernel code)
#   6. Handler services the interrupt
#   7. Handler executes IRET -- restores state, resumes program

# Interrupt priority: higher-priority interrupts can preempt lower ones
# NMI (Non-Maskable Interrupt): cannot be disabled (hardware failure, watchdog)

# APIC (Advanced Programmable Interrupt Controller):
#   Modern x86 interrupt routing -- each core has a Local APIC
#   I/O APIC routes external interrupts to cores
```

## Tips

- Understanding cache behavior matters more than clock speed for real-world performance
- Memory access patterns dominate performance in modern systems -- think about data locality
- The "memory wall" (gap between CPU speed and memory speed) has driven most architecture innovation since the 1990s
- When debugging performance: measure first, then check cache misses, branch mispredictions, and TLB misses with `perf stat`
- Virtual memory gives each process isolation for free -- exploits in one program can't read another's memory (barring side-channel attacks like Spectre/Meltdown)

## See Also

- kernel
- strace
- gdb
- valgrind

## References

- Patterson & Hennessy, "Computer Organization and Design" (the standard textbook)
- Hennessy & Patterson, "Computer Architecture: A Quantitative Approach"
- Ben Eater's YouTube series on building an 8-bit computer from logic gates
- Nand2Tetris course (nand2tetris.org) -- build a computer from NAND gates up
- Intel Software Developer Manuals (SDM) Vol. 1-3
- CPU microarchitecture diagrams: wikichip.org
