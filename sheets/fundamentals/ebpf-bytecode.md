# eBPF Bytecode (The Kernel's Plugin System)

A tiered guide to eBPF's instruction set, execution model, and safety guarantees.

## ELI5

Imagine your computer's operating system is like a big, important factory. The
factory has strict rules: no visitors allowed inside because they might break
something or get hurt.

But what if you want to add a security camera, or a counter that tracks how
many boxes go through a conveyor belt? You cannot just walk in and start
rewiring things.

**eBPF is like a safe plugin system for the factory.** You write a small
program -- a "plugin" -- and hand it to the factory manager. Before the
manager installs it, a **safety inspector** (the verifier) checks your plugin
very carefully:

- Does it ever touch something it should not?
- Could it get stuck in an infinite loop and jam the conveyor belt?
- Does it try to reach outside its allowed area?

Only after the inspector says "this is safe" does the plugin get installed.
Once running, it can watch the conveyor belt, count boxes, or redirect
packages -- but it **cannot** break the factory.

Think of it like **browser extensions for your operating system**. Chrome
extensions can change how web pages look and add features, but they run in a
sandbox so a bad extension cannot delete your files. eBPF does the same thing
but for the Linux kernel -- it lets you add features (networking rules,
security monitoring, performance tracing) without risking a crash.

## Middle School

### What eBPF Actually Is

eBPF stands for **extended Berkeley Packet Filter**. It started as a way to
filter network packets but evolved into a general-purpose system for running
small programs inside the Linux kernel.

An eBPF program is a sequence of **instructions** -- simple operations like
"add two numbers" or "load a value from memory." These instructions run on a
**virtual machine** inside the kernel, not directly on the CPU (though later
they get translated to real CPU instructions for speed).

### The Register Set

eBPF has **11 registers**, numbered r0 through r10. Think of registers as
small, fast scratch pads the program uses for calculations:

| Register | Purpose |
|:---|:---|
| r0 | Return value -- the program's answer goes here |
| r1 - r5 | Function arguments -- used to pass data to helper functions |
| r6 - r9 | Callee-saved -- your scratch space that survives function calls |
| r10 | Frame pointer -- points to the program's stack (read-only) |

All registers are 64 bits wide (can hold very large numbers).

### Basic Operations

eBPF programs are built from a small set of simple operations:

```
mov  r1, 42          # Put the number 42 into register r1
add  r1, r2          # r1 = r1 + r2
sub  r3, 10          # r3 = r3 - 10
ldxw r1, [r10-4]     # Load a 32-bit value from the stack into r1
stxw [r10-8], r2     # Store r2's value onto the stack
jeq  r1, 0, +5       # If r1 equals 0, skip ahead 5 instructions
call bpf_helper      # Call a kernel helper function
exit                 # End the program, return value is in r0
```

### The Verifier: The Safety Checker

Before any eBPF program runs, the kernel's **verifier** inspects every
possible path through the program:

1. **No infinite loops.** The program must always finish. The verifier rejects
   programs with unbounded loops.
2. **No bad memory access.** Every load and store must be within bounds -- no
   reading random kernel memory.
3. **No uninitialized registers.** You cannot use a register before putting a
   value in it.
4. **Limited size.** Programs have a maximum number of instructions (1 million
   in modern kernels).

If the verifier rejects your program, it tells you exactly which instruction
is the problem and why.

## High School

### Instruction Format

Every eBPF instruction is exactly **8 bytes** (64 bits), laid out as:

```
 8 bits    4 bits   4 bits    16 bits      32 bits
┌────────┬────────┬────────┬───────────┬──────────────┐
│ opcode │ dst_reg│ src_reg│  offset   │   immediate  │
└────────┴────────┴────────┴───────────┴──────────────┘
```

- **opcode** (8 bits): the operation to perform, encoded as
  `class | source | operation`.
- **dst_reg** (4 bits): destination register (r0-r10).
- **src_reg** (4 bits): source register (r0-r10).
- **offset** (16 bits): signed offset for memory and branch instructions.
- **immediate** (32 bits): constant value embedded in the instruction.

For 64-bit immediate loads (`BPF_LD | BPF_DW | BPF_IMM`), the instruction
uses 16 bytes -- two consecutive 8-byte slots -- to fit the full 64-bit value.

### Instruction Classes

| Class | Value | Description |
|:---|:---|:---|
| BPF_LD | 0x00 | Non-standard loads (64-bit imm) |
| BPF_LDX | 0x01 | Load from memory into register |
| BPF_ST | 0x02 | Store immediate to memory |
| BPF_STX | 0x03 | Store register to memory |
| BPF_ALU | 0x04 | 32-bit arithmetic |
| BPF_JMP | 0x05 | 64-bit jumps |
| BPF_JMP32 | 0x06 | 32-bit jumps |
| BPF_ALU64 | 0x07 | 64-bit arithmetic |

### ALU Operations

Arithmetic and logical operations come in 32-bit (`BPF_ALU`) and 64-bit
(`BPF_ALU64`) variants. The source can be a register (`BPF_X`) or an
immediate value (`BPF_K`):

```
BPF_ADD:  dst += src        BPF_SUB:  dst -= src
BPF_MUL:  dst *= src        BPF_DIV:  dst /= src
BPF_OR:   dst |= src        BPF_AND:  dst &= src
BPF_LSH:  dst <<= src       BPF_RSH:  dst >>= src (logical)
BPF_NEG:  dst = -dst        BPF_MOD:  dst %= src
BPF_XOR:  dst ^= src        BPF_ARSH: dst >>= src (arithmetic)
BPF_MOV:  dst = src         BPF_END:  byte swap
```

32-bit operations (`BPF_ALU`) zero-extend the result to 64 bits.

### Memory Access

Memory access uses size specifiers:

```
BPF_B  — 8-bit   (byte)
BPF_H  — 16-bit  (half-word)
BPF_W  — 32-bit  (word)
BPF_DW — 64-bit  (double-word)
```

Load from memory:

```
BPF_LDX | BPF_MEM | BPF_W:   dst = *(u32 *)(src + offset)
BPF_LDX | BPF_MEM | BPF_DW:  dst = *(u64 *)(src + offset)
```

Store to memory:

```
BPF_STX | BPF_MEM | BPF_W:   *(u32 *)(dst + offset) = src
BPF_ST  | BPF_MEM | BPF_W:   *(u32 *)(dst + offset) = imm
```

### Branching

Conditional jumps compare two values and skip ahead by `offset` instructions:

```
BPF_JEQ:  jump if dst == src      BPF_JNE:  jump if dst != src
BPF_JGT:  jump if dst > src       BPF_JGE:  jump if dst >= src
BPF_JLT:  jump if dst < src       BPF_JLE:  jump if dst <= src
BPF_JSET: jump if dst & src != 0
BPF_JSGT: jump if dst > src  (signed)
BPF_JSGE: jump if dst >= src (signed)
BPF_JSLT: jump if dst < src  (signed)
BPF_JSLE: jump if dst <= src (signed)
```

Backward jumps are allowed only in bounded loops (kernel 5.3+).

### BPF Maps

Maps are **key-value data structures** shared between eBPF programs and
userspace. They persist beyond a single program invocation:

| Map Type | Use Case |
|:---|:---|
| `BPF_MAP_TYPE_HASH` | General key-value lookup (hash table) |
| `BPF_MAP_TYPE_ARRAY` | Fast indexed access (fixed-size array) |
| `BPF_MAP_TYPE_PERCPU_HASH` | Per-CPU hash map (no lock contention) |
| `BPF_MAP_TYPE_PERCPU_ARRAY` | Per-CPU array |
| `BPF_MAP_TYPE_RINGBUF` | Efficient event streaming to userspace |
| `BPF_MAP_TYPE_LRU_HASH` | Hash map with LRU eviction |
| `BPF_MAP_TYPE_LPM_TRIE` | Longest prefix match (IP routing) |
| `BPF_MAP_TYPE_PROG_ARRAY` | Array of program fds (for tail calls) |

### Key Helper Functions

eBPF programs cannot call arbitrary kernel functions. Instead, they call
**helper functions** -- a stable, verified API:

```c
// Map operations
void *bpf_map_lookup_elem(map, key)      // returns pointer to value or NULL
int   bpf_map_update_elem(map, key, val, flags)
int   bpf_map_delete_elem(map, key)

// Tracing and debugging
int   bpf_trace_printk(fmt, fmt_size, ...)  // debug print to trace_pipe

// Process context
u64   bpf_get_current_pid_tgid()         // current PID and TGID
u64   bpf_get_current_uid_gid()          // current UID and GID
int   bpf_get_current_comm(buf, size)    // current process name

// Time
u64   bpf_ktime_get_ns()                 // monotonic clock in nanoseconds

// Networking
int   bpf_skb_load_bytes(skb, off, buf, len)  // read packet bytes
int   bpf_redirect(ifindex, flags)             // redirect packet
int   bpf_xdp_adjust_head(xdp_md, delta)      // adjust packet head

// Ring buffer
int   bpf_ringbuf_output(ringbuf, data, size, flags)
```

### Program Types

The program type determines **where** the program attaches and **what context**
it receives:

| Program Type | Attach Point | Context | Use Case |
|:---|:---|:---|:---|
| `BPF_PROG_TYPE_XDP` | Network driver (pre-stack) | `xdp_md` | DDoS filtering, load balancing |
| `BPF_PROG_TYPE_SCHED_CLS` | Traffic control (tc) | `__sk_buff` | Packet mangling, policy |
| `BPF_PROG_TYPE_TRACEPOINT` | Static kernel events | Tracepoint args | Observability |
| `BPF_PROG_TYPE_KPROBE` | Any kernel function | `pt_regs` | Dynamic tracing |
| `BPF_PROG_TYPE_CGROUP_SKB` | Cgroup socket buffer | `__sk_buff` | Per-cgroup networking |
| `BPF_PROG_TYPE_LSM` | LSM hooks | Hook-specific | Security policies |
| `BPF_PROG_TYPE_PERF_EVENT` | Perf events | `bpf_perf_event_data` | Profiling |
| `BPF_PROG_TYPE_SOCKET_FILTER` | Socket | `__sk_buff` | Packet filtering |

```bash
# List loaded BPF programs
bpftool prog list

# Show program details (bytecode, JIT, stats)
bpftool prog show id <id>
bpftool prog dump xlated id <id>    # verifier-processed bytecode
bpftool prog dump jited id <id>     # JIT-compiled native code

# List and inspect maps
bpftool map list
bpftool map dump id <id>

# Attach a program to a tracepoint
bpftool prog attach id <id> tracepoint <category> <event>

# View BPF filesystem (pinned objects)
ls /sys/fs/bpf/
```

## College

### JIT Compilation

The eBPF bytecode is an intermediate representation. For production
performance, the kernel **JIT-compiles** eBPF instructions to native machine
code:

1. **Verification.** The verifier processes the bytecode first.
2. **Translation.** Each BPF instruction maps to one or more native
   instructions. For example, `BPF_ALU64 | BPF_ADD | BPF_X` (add two
   registers) becomes a single `add` on x86_64.
3. **Register mapping.** BPF registers map to hardware registers:
   - x86_64: r0→rax, r1→rdi, r2→rsi, r3→rdx, r4→rcx, r5→r8,
     r6→rbx, r7→r13, r8→r14, r9→r15, r10→rbp.
4. **Optimization.** The JIT applies peephole optimizations: constant folding,
   dead code elimination, instruction combining.
5. **Memory placement.** JIT code is placed in executable kernel memory with
   appropriate permissions (W^X: writable during JIT, executable after).

```bash
# Check if JIT is enabled
cat /proc/sys/net/core/bpf_jit_enable
# 0 = interpreter only, 1 = JIT, 2 = JIT + debug output

# Enable JIT
echo 1 > /proc/sys/net/core/bpf_jit_enable

# View JIT output
bpftool prog dump jited id <id>
```

### Tail Calls and Program Chaining

A single eBPF program has size limits (1 million instructions after
verification). **Tail calls** allow one program to hand off to another without
returning, effectively chaining programs:

```c
// tail call: replaces current program with program at index in prog_array
bpf_tail_call(ctx, prog_array_map, index);
```

Key properties:

- The stack frame is reused (no stack growth).
- Maximum tail call depth: 33.
- The called program inherits the same context.
- Tail calls use a `BPF_MAP_TYPE_PROG_ARRAY` map.

Use cases: protocol parsing pipelines (parse ethernet → tail call to IP parser
→ tail call to TCP parser), modular firewall rules.

### BTF (BPF Type Format)

BTF is a compact type metadata format that describes the data structures used
by BPF programs and the kernel. It is a lightweight alternative to DWARF
debug info:

- Encodes struct layouts, field names, field types, and sizes.
- Attached to BPF programs and maps for introspection.
- The kernel exposes its own BTF at `/sys/kernel/btf/vmlinux`.
- Tools like `bpftool` use BTF to pretty-print map contents.

```bash
# View kernel BTF
bpftool btf dump file /sys/kernel/btf/vmlinux format c | head -100

# List BTF objects
bpftool btf list
```

### CO-RE (Compile Once, Run Everywhere)

CO-RE solves the portability problem. Without CO-RE, a BPF program compiled
against one kernel version's struct layouts breaks on another version if
fields moved or were added.

CO-RE combines three components:

1. **BTF** in the program records which struct fields the program accesses.
2. **BTF** in the target kernel describes the actual struct layout.
3. **libbpf's relocator** patches field offsets at load time to match the
   running kernel.

Result: compile a BPF program once on your development machine and run it on
any kernel (with BTF enabled) without recompilation.

### Verifier Internals

The verifier performs **abstract interpretation** -- it simulates every
possible execution path through the program without actually running it:

1. **Register state tracking.** Each register has a tracked type:
   `NOT_INIT`, `SCALAR_VALUE`, `PTR_TO_CTX`, `PTR_TO_MAP_VALUE`,
   `PTR_TO_STACK`, `PTR_TO_PACKET`, etc. The verifier knows what each
   register points to and its valid bounds.

2. **DAG exploration.** The verifier walks all paths through the program's
   control flow graph (which must be a DAG for the main body -- no back-edges
   except in bounded loops). It follows both branches of every conditional
   jump.

3. **State pruning.** At each instruction, the verifier records the abstract
   state (all register types and ranges). If it reaches the same instruction
   with a state that is a subset of a previously verified state, it prunes
   that path -- no need to re-explore. This is critical for keeping
   verification time manageable.

4. **Bounds tracking.** For scalar values, the verifier tracks minimum and
   maximum values (both signed and unsigned). After a comparison like
   `if r1 < 100`, the verifier knows r1 is in [0, 99] on the true branch.
   This enables safe array indexing: `map_value[r1]` is safe if r1 is
   provably within the map value size.

5. **Termination guarantee.** No unbounded back-edges are allowed. Bounded
   loops (kernel 5.3+) are unrolled or verified with a trip count limit.

### Bounded Loops

Since kernel 5.3, the verifier supports loops with provably bounded iteration:

```c
for (int i = 0; i < 100; i++) {
    // verifier tracks i's bounds: [0, 99]
    // body must not modify i in unbounded ways
}
```

The verifier simulates the loop or proves its bound. The maximum number of
verified instructions per program is 1 million -- a loop that iterates too
many times will exceed this budget and be rejected.

### BPF-to-BPF Function Calls

Modern eBPF supports function calls within a program (not just tail calls):

```c
static int helper(int x) {
    return x + 1;
}

int main_prog(struct xdp_md *ctx) {
    int val = helper(42);
    // ...
}
```

- Uses `BPF_CALL` instruction with a relative offset.
- Each function gets its own stack frame (512 bytes max per frame).
- Maximum call depth: 8.
- The verifier verifies each function independently, tracking the call graph.

### Atomic Operations

eBPF supports atomic memory operations for safe concurrent access:

```
BPF_ATOMIC | BPF_W  | BPF_STX:  atomic 32-bit operation
BPF_ATOMIC | BPF_DW | BPF_STX:  atomic 64-bit operation

Operations (encoded in imm field):
  BPF_ADD:   lock *(u64 *)(dst + off) += src
  BPF_OR:    lock *(u64 *)(dst + off) |= src
  BPF_AND:   lock *(u64 *)(dst + off) &= src
  BPF_XOR:   lock *(u64 *)(dst + off) ^= src
  BPF_XCHG:  src = xchg(dst + off, src)
  BPF_CMPXCHG: r0 = cmpxchg(dst + off, r0, src)
```

These map to hardware atomics (`lock add`, `lock cmpxchg` on x86_64).

### BPF LSM (Linux Security Modules)

BPF LSM programs attach to security hooks, enabling dynamic security policies
without compiling custom kernel modules:

```c
SEC("lsm/file_open")
int restrict_open(struct file *file) {
    // Check file path, caller credentials, etc.
    // Return 0 to allow, -EPERM to deny
}
```

BPF LSM is stackable -- it runs alongside existing LSMs (AppArmor, SELinux).
It can enforce per-container, per-cgroup, or per-process security policies at
runtime.

### BPF Iterators

BPF iterators allow eBPF programs to walk kernel data structures and produce
output, replacing `/proc` files with programmable alternatives:

```c
SEC("iter/task")
int dump_tasks(struct bpf_iter__task *ctx) {
    struct task_struct *task = ctx->task;
    if (task)
        BPF_SEQ_PRINTF(ctx->meta->seq, "pid=%d comm=%s\n",
                       task->pid, task->comm);
    return 0;
}
```

Iterator types: `task`, `bpf_map_elem`, `tcp`, `udp`, `bpf_prog`, and more.
They produce sequential output readable from userspace via `read()` on the
iterator's fd.

## Tips

- Start with `bpftool prog list` and `bpftool map list` to see what BPF
  programs are already running on your system. Systemd, container runtimes,
  and observability tools all use BPF.
- Use `bpftool prog dump xlated id <id>` to see the verifier's view of a
  program, and `dump jited` to see the native machine code.
- The verifier's error messages are extremely detailed. Read them carefully --
  they tell you the exact register state and why verification failed.
- For development, use libbpf + CO-RE + BTF. Avoid BCC (Python wrapper) for
  production -- it compiles at runtime and requires kernel headers on every
  machine.
- Test BPF programs with `BPF_PROG_TEST_RUN` (`bpf()` syscall command) to
  feed synthetic input without attaching to live kernel hooks.
- Enable JIT (`bpf_jit_enable=1`) for production. The interpreter is 10-100x
  slower.
- Per-CPU maps (`BPF_MAP_TYPE_PERCPU_HASH`, `BPF_MAP_TYPE_PERCPU_ARRAY`)
  eliminate lock contention for counters and statistics.

## See Also

- linux-kernel-internals
- x86-assembly
- networking-fundamentals
- binary-and-number-systems

## References

- eBPF documentation: https://ebpf.io/what-is-ebpf/
- Linux kernel BPF documentation: https://docs.kernel.org/bpf/
- BPF instruction set specification: https://docs.kernel.org/bpf/standardization/instruction-set.html
- Brendan Gregg "BPF Performance Tools" (Addison-Wesley, 2019)
- Liz Rice "Learning eBPF" (O'Reilly, 2023)
- Cilium BPF reference guide: https://docs.cilium.io/en/latest/bpf/
- bpftool man page: https://man7.org/linux/man-pages/man8/bpftool.8.html
- libbpf documentation: https://libbpf.readthedocs.io/
- Linux kernel source — BPF verifier: https://elixir.bootlin.com/linux/latest/source/kernel/bpf/verifier.c
