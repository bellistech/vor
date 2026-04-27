# eBPF — ELI5 (Tiny Robots Inside the Kernel)

> eBPF lets you send tiny safe robots into the brain of your computer to watch things, count things, and stop bad things, without ever taking the brain apart.

## Prerequisites

(none — but `cs ramp-up linux-kernel-eli5` helps; eBPF lives inside the kernel, and it helps to know what the kernel even is before we start putting little robots inside it)

If the word "kernel" feels weird, read that sheet first. We will explain it again here in case you skipped it. We will explain everything here in case you skipped everything. The whole point of an ELI5 sheet is that you do not need to have read anything else to read this one.

If you see a `$` at the start of a line in a code block, that means "type the rest of the line into your terminal." You do not type the `$`. The lines underneath the `$` line, the ones without a `$`, are what your computer prints back at you. We call that "output."

If a word feels weird, look it up in the **Vocabulary** table near the bottom. Every weird word in this sheet is in that table with a one-line plain-English definition.

## What Even Is eBPF?

### The kernel is the brain of your computer

Deep inside every computer there is a special program called the kernel. The kernel is the boss of everything. It decides which program gets to use the CPU. It decides which program gets memory. It decides which program is allowed to talk to the network card, the hard drive, the keyboard, the screen, the speakers. Everything that any program ever does, the kernel knows about, because the kernel is the one who actually does it.

You can think of the kernel as the brain of your computer. It is incredibly powerful. It is also very protected. You cannot just walk up to it and start changing how it works. That would be like performing brain surgery on yourself with a butter knife. One wrong move and the whole computer falls over. The screen freezes. The lights blink. You have to pull the plug and start over and you probably lost everything you were working on.

So the kernel is locked behind a thick wall. Programs talk to the kernel by knocking on a little window in the wall and asking politely. "Please give me some memory." "Please open this file." "Please send this packet to the internet." The kernel answers through the window. The programs never go inside.

This is great for safety. Nobody can break the brain. But it is annoying when you actually need to know what the brain is doing. Because everything important happens inside the brain. The brain is where the network packets are processed. The brain is where the files are written. The brain is where one program steals time from another. If you want to see any of that, you have to look inside the brain.

### The old options were both bad

Before eBPF, if you wanted to see what was happening inside the kernel, you had two choices and both of them were bad.

**Choice number one: ask the kernel for reports.** The kernel has a few built-in ways to tell you what it is doing. It can tell you how many packets came in. It can tell you how much memory is used. It can tell you a few other things. But the kernel only tells you the things its programmers thought to expose, on the day they wrote the kernel. If you want to know something they did not expose, tough luck.

**Choice number two: change the kernel itself.** You could download the kernel's source code, find the spot you wanted to look at, write some new code there, recompile the entire kernel (this takes hours), reboot your computer, and pray. If your code had a bug, the kernel crashed. The whole computer fell over. You had to reboot, fix the bug, recompile, reboot again, and try again. And every time the kernel got a new version, you had to redo all of your changes.

This is why almost nobody changed the kernel for fun. It was too dangerous and too slow.

### eBPF is the third option

eBPF (which stands for "extended Berkeley Packet Filter," a name we will explain later because the name is not really important) is the third choice. With eBPF, you can put little programs INSIDE the running kernel, on the fly, without rebooting, without recompiling, and without any chance of crashing the kernel. You can put one in, watch what it does, take it out, write a new one, put that one in, take it out. All while your computer keeps running normally.

It is like opening a little window in the brain and slipping in a tiny robot. The robot has a job. Maybe its job is "count every time a packet comes in." Maybe its job is "watch every file the computer opens and write down which program opened it." Maybe its job is "drop any packet that looks like an attack." You build the robot, you give it its instructions, you slip it in through the window. The robot does its job. When you are done, you pull the robot out. The brain never noticed. The brain never crashed. Everything just kept running.

That is eBPF.

### Tiny robots inside the kernel

Every time you read about eBPF in this sheet, picture a tiny robot. The robot is very small. The robot has a single job. The robot lives in the brain, in a very specific room (we will get to the rooms). The robot watches what happens in that room and writes notes. Sometimes the robot is allowed to make decisions, like "drop this packet" or "let this packet through." Sometimes the robot just counts.

The robots have rules. They have a lot of rules. They cannot run forever. They cannot use too much memory. They cannot reach into rooms they were not invited into. Before any robot is allowed inside the brain, a very strict safety inspector checks the robot from head to toe. If the robot fails any safety check, the inspector throws the robot in the trash. Only safe robots get to go in.

This is why eBPF is amazing. The robots are powerful. They can see everything in the kernel. They can count things, drop things, move things around, make decisions. But they can never crash the kernel, because the safety inspector would never let them in if they could.

### Why the funny name?

eBPF stands for "extended Berkeley Packet Filter." Way back in 1992, two researchers at the University of California, Berkeley invented a tiny programming language for filtering network packets. The whole language only existed to answer the question "is this network packet interesting?" If the answer was yes, the kernel would copy the packet up to the program (like `tcpdump`). If the answer was no, the kernel would throw the packet away.

That tiny language was called "BPF." Just BPF. Berkeley Packet Filter.

Around 2014, a kernel developer named Alexei Starovoitov took the BPF idea and supercharged it. He made the language much bigger, gave it more registers (we will explain registers later), let it talk to data structures called maps, and let it attach to lots of places in the kernel besides just packet filtering. Because the new version was an extension of the old BPF, he called it "eBPF." The "e" stands for "extended."

Today, almost everybody just says "BPF" or "eBPF" interchangeably. The official kernel docs say "BPF" most of the time. The marketing material says "eBPF" most of the time. They mean the same thing. The old 1992 version is sometimes called "classic BPF" or "cBPF" to tell them apart. We will say "eBPF" in this sheet.

### What the rest of this sheet is

The rest of this sheet is a tour. We will visit:

1. The safety inspector (the verifier).
2. The robots themselves (programs).
3. The notes the robots leave for each other and for you (maps).
4. The rooms in the brain where robots can live (attach points).
5. How you and your robots talk to each other (the bpf() syscall, libbpf, BPF skeletons).
6. The magic that lets one robot work in any version of the brain (CO-RE and BTF).
7. What people actually use this stuff for (observability, networking, security, performance, profiling).
8. The high-level scripting tool that turns this all into one-liners (bpftrace).
9. A long list of paste-and-runnable commands you can try right now.
10. A long list of "wait, what?" questions that almost everyone has when they first learn eBPF.
11. A vocabulary table.
12. A pile of safe experiments.

It is a long ride. Buckle up. There is no part you cannot read. If a section feels too dense, skip ahead. The vocabulary table at the bottom catches every word.

## Why eBPF Is Special

### Kernel modules can crash the kernel

Before eBPF, if you wanted to add code to the running kernel, the way to do it was a "kernel module." A kernel module is a chunk of code that you can load into the running kernel without rebuilding the whole kernel. Drivers (the code that talks to a specific brand of network card or graphics card) are kernel modules. Filesystems are usually kernel modules. Lots of stuff is kernel modules.

Kernel modules are very powerful. They run inside the kernel. They have full access to everything in the kernel. They can read and write any memory. They can call any kernel function. They can do whatever they want.

This is also why kernel modules are dangerous. There is no safety net. If a kernel module has a bug, the kernel crashes. If a kernel module dereferences a null pointer, the kernel crashes. If a kernel module loops forever, the kernel hangs. If a kernel module overwrites the wrong memory, the kernel might behave fine for ten minutes and then explode in some weird way and you have to spend three days figuring out which module did it.

This is why people were nervous about kernel modules. You really had to trust the people who wrote them. You really had to test them. And testing kernel modules is hard, because the only way to test them is to load them into a kernel and see what happens, and if it goes wrong the kernel crashes.

### eBPF programs cannot crash the kernel

eBPF programs are different. They run inside the kernel, just like kernel modules. But they cannot crash the kernel.

This sounds impossible. How can a program run inside the kernel without being able to crash it? The trick is the verifier. Before any eBPF program is allowed to run, a piece of code called the verifier walks through every possible path the program could take, and proves to itself that the program is safe. If the verifier finds even one path where the program might do something bad, the verifier rejects the program. The program does not load. The kernel never runs it. There is no crash because there is no execution.

This is huge. It means anyone can write eBPF programs. You do not have to be a kernel kernel-genius. You do not have to be employed at Red Hat or Google or Cloudflare. You can write a tiny eBPF program on your laptop and load it into your kernel and if there is a problem, the verifier will yell at you and refuse to load it. The verifier is your safety net.

It is like the safety inspector at a factory. Every machine that anyone wants to bring onto the factory floor goes through the safety inspector first. The inspector reads the machine's plans, looks at every gear, every wire, every belt. If anything could possibly hurt anyone, the inspector says "no, take it back, fix it, come back when it is safe." Only machines that the inspector signs off on get to come onto the floor.

Same idea. The verifier is the safety inspector. The eBPF program is the machine. The kernel is the factory floor.

### Speed

The other thing that makes eBPF special is that it is fast. Really fast.

When you write an eBPF program, you usually write it in C, a language that is famous for being fast and famous for being scary. (You can also use Rust now. We will mention Rust later. Aya is a Rust library for eBPF.) Then you compile your C program into something called eBPF bytecode. Bytecode is a kind of pretend machine code for a pretend computer. The pretend computer has 11 registers, a small stack, and about 100 different instructions.

When you load the bytecode into the kernel, the kernel does two things. First, the verifier checks the bytecode for safety. Second, if the bytecode is safe, a thing called the JIT (Just-In-Time compiler) translates the bytecode into actual machine code for your actual CPU. So when your eBPF program runs, it is not running in some slow interpreter. It is running as native machine code. It is as fast as the kernel itself.

This means eBPF programs can run on the hot path. They can run on every single network packet. They can run on every system call. They can run on every disk read. They can run sixty million times per second and not slow anything down measurably. That is what makes them useful in production. You can leave them turned on all the time without paying a tax.

### Removable

The other amazing thing about eBPF is that the robots can come out as easily as they went in. There is no reboot. There is no shutdown. You just say "remove that program" and the kernel unloads it. The next instant, it is gone. The kernel is back to normal. The robot has left.

This means you can experiment. You can try a robot. If you do not like it, you remove it. You write a new one. You try that. You remove it. Production never went down. Nobody noticed.

Compare that with kernel modules. To unload a kernel module you usually need root and you need to convince the kernel that nothing is using the module right now. And if the module is buggy, the unload itself can crash the kernel. Removable, sure, but with a lot of caveats.

eBPF is removable for real.

### Summary table

Robot Property | eBPF Equivalent | Why It Matters
---|---|---
Tiny | Max ~1 million instructions in one program | Small enough not to slow the brain down
Safe | Verified before running | A safety inspector checks every path
Fast | JIT-compiled to native machine code | Runs at kernel speed, nanoseconds per call
Removable | Unloaded any time, no reboot | Pull the robot out when you are done
Cannot crash the brain | No infinite loops, no out-of-bounds reads | The verifier guarantees it

## The Verifier

The verifier is the most important part of eBPF and it is the part that confuses people the most. So we are going to spend a long time on it.

### What the verifier is doing

The verifier is a piece of code inside the kernel. It runs whenever you try to load an eBPF program. Its job is to look at the program and prove that the program is safe.

Specifically, the verifier wants to know:

1. Does the program always finish? It must not have an infinite loop. If a program loops forever inside the kernel, the kernel hangs. The verifier will not allow that.

2. Does the program always read memory it is allowed to read? It must not read past the end of an array. It must not read uninitialized memory. It must not read random pointers. The verifier will not allow that.

3. Does the program always write memory it is allowed to write? Same idea, but for writes. Writing into the wrong place is even worse than reading.

4. Does the program return a sensible value? Every eBPF program ends with a return value. The verifier wants to know that the program will set a return value before it exits, and the value must make sense for the program's type.

5. Does the program call only the helper functions it is allowed to call? eBPF programs can call special kernel functions called "helpers" (we will see them later). Different program types are allowed to call different helpers. The verifier checks that you only call the ones you are allowed to call.

If any of those checks fail, the verifier rejects the program with an error message. The program does not load.

### How the verifier walks the program

The verifier walks the program. Imagine the program as a flowchart with branches. The verifier starts at the top, follows every possible path through the flowchart, and at every step writes down what it knows about every register.

What does "what it knows about every register" mean? eBPF has 11 registers (R0 through R10). Each register holds a 64-bit value. The verifier tracks for each register:

- What kind of thing is in this register? Is it a number? Is it a pointer to a packet? Is it a pointer to the program's stack? Is it nothing yet (uninitialized)?
- If it is a number, what is the smallest it could be? What is the largest it could be?
- If it is a pointer, what does it point into? What is the offset from the base?
- If it is uninitialized, that is a fact too: the verifier will refuse to let you read from an uninitialized register.

At every instruction, the verifier updates this knowledge. If you do `r0 = r1 + 5`, the verifier says "OK, r0 is now whatever r1 was, plus 5. If r1 was a number between 0 and 100, then r0 is between 5 and 105."

When the program reaches a branch (like an `if` statement), the verifier splits and walks both sides. On the "if true" side, the verifier knows the condition was true, so it tightens its knowledge accordingly. On the "if false" side, it tightens the other way.

When the program reaches a place that joins two paths back together, the verifier merges what it knew from each side. Sometimes the merge loses precision. Sometimes the verifier has to give up some certainty.

If the verifier is ever about to do something dangerous (like read past the end of an array, or use an uninitialized register), it says "nope" and rejects the whole program.

### "Safety > performance > convenience"

This is the verifier's motto. The verifier picks safety first, performance second, convenience third.

That means: sometimes you write a program that is actually safe, but the verifier cannot prove it is safe. In that case, the verifier still rejects the program. The verifier would rather reject a safe program than risk accepting an unsafe one. This is annoying when it happens to you. You stare at the program. You can see with your own eyeballs that the program is safe. But the verifier disagrees. The verifier wins.

When this happens, you have to rewrite the program in a way that the verifier can understand. Often this means simplifying it, removing loops, splitting big programs into smaller ones, or being more explicit about your bounds checks. We will give examples of this in **Common Confusions**.

### Common verifier rejections

Here are the verifier errors you will see most often. Each one comes with the actual error message and the canonical fix.

**`R0 !read_ok`** — you tried to read from register R0 before writing anything to it. Fix: assign R0 a value before you use it. Most often this means setting your return value with `return 0;` or similar.

**`invalid mem access`** — you tried to read or write memory in a way the verifier does not understand. Often this is reading past the end of a packet without a bounds check. Fix: add an explicit `if (ptr + size > data_end) return XDP_PASS;` before the access.

**`R3 type=inv expected=fp`** — you passed something to a helper that expected a pointer to the stack ("fp" = frame pointer), but you passed something that is not a stack pointer. Fix: pass a stack-allocated buffer.

**`back-edge from insn X to Y`** — your program has a loop that the verifier could not prove always terminates. Fix: in older kernels (before 5.3), unroll the loop manually with `#pragma unroll`. In newer kernels, use bounded loops, where the loop count is a compile-time-known small number, or use the `bpf_loop()` helper.

**`processed X insns, limit is 1000000`** — your program is too big. The verifier walks every path, and the total instruction count it ends up walking exceeded the limit. Fix: split your program into multiple programs and use tail calls (more on those later) to chain them together.

**`combined stack size exceeds 512`** — your program tried to use more than 512 bytes of stack. eBPF programs have a tiny stack. Fix: move your big buffers into a BPF map.

If your program loaded successfully, the verifier liked it. If it didn't load, the verifier did not. Read the error. Fix the issue. Try again.

### A picture of the verifier

```
Your C source code
        |
        v
   clang -target bpf -O2
        |
        v
  eBPF bytecode (in an ELF file)
        |
        v
  bpf(BPF_PROG_LOAD) syscall
        |
        v
  +-----------------+
  |    VERIFIER     |
  |                 |
  | walks all paths |
  | tracks register |
  | types & ranges  |
  | checks bounds   |
  | checks loops    |
  | checks helpers  |
  +-----------------+
        |
   +----+----+
   |         |
   v         v
 REJECT    ACCEPT
   |         |
   |         v
   |    +---------+
   |    |   JIT   |
   |    |         |
   |    | bytecode|
   |    |   to    |
   |    | native  |
   |    | machine |
   |    |  code   |
   |    +---------+
   |         |
   |         v
   |    +---------+
   |    | ATTACH  |
   |    |         |
   |    | hook to |
   |    | kprobe, |
   |    |  XDP,   |
   |    |  etc.   |
   |    +---------+
   |         |
   |         v
   |    PROGRAM LIVE,
   |    ROBOT IS WORKING
   v
 ERROR MESSAGE,
 FIX THE PROGRAM
```

This is the entire eBPF lifecycle in one picture. Source -> bytecode -> verifier -> JIT -> attach -> live. If the verifier rejects, you go back to the start.

## Programs and Maps

### Programs are the robots

A "BPF program" is one tiny robot. Each robot:

- Is written in C (or Rust, or another language that can compile to BPF bytecode).
- Compiled to BPF bytecode.
- Loaded into the kernel via the `bpf()` syscall.
- Verified by the verifier.
- JIT-compiled to native machine code.
- Attached to a specific spot in the kernel (we will see all the spots in the next section).
- Wakes up whenever something happens at that spot.
- Does its tiny job.
- Goes back to sleep until something happens at that spot again.

Each robot has a "type." The type tells the kernel what spot the robot is going to attach to, what arguments it will get, what return values it can give back, and what helper functions it is allowed to call.

A robot of type `BPF_PROG_TYPE_XDP` attaches to a network interface, gets a pointer to the incoming packet, and returns one of `XDP_DROP`, `XDP_PASS`, `XDP_TX`, `XDP_REDIRECT`. It is the packet-fast-path robot.

A robot of type `BPF_PROG_TYPE_KPROBE` attaches to a kernel function, gets the CPU registers as its argument, and returns 0. It is the function-watcher robot.

A robot of type `BPF_PROG_TYPE_TRACEPOINT` attaches to a stable kernel tracepoint, gets a struct of arguments, and returns 0. It is the tracepoint-watcher robot.

Different types, different jobs. Same idea: tiny robot, attached to a place, wakes up on an event, does a job, goes back to sleep.

### Maps are the shared whiteboards

Robots are tiny. They have a 512-byte stack. They cannot allocate memory. So how do they remember anything? How do they tell each other things? How do they tell the user-space program that loaded them?

The answer is **maps**. A map is a key-value store inside the kernel. Robots can read from a map and write to a map. The user-space program that loaded the robots can also read from the map and write to the map. So a map is a shared whiteboard. The robots write notes on the whiteboard. The user-space program reads them and reacts.

You create a map with the `bpf(BPF_MAP_CREATE)` syscall. You tell the kernel what kind of map you want, what the key size is, what the value size is, and how many entries it should have. The kernel gives you back a file descriptor. You use that file descriptor to read and write the map.

Robots access the map through helper functions: `bpf_map_lookup_elem`, `bpf_map_update_elem`, `bpf_map_delete_elem`. The user-space program accesses the map through the `bpf()` syscall (or through `libbpf`, which wraps the syscall).

### Map types

There are a lot of map types. Each one is a different shape of whiteboard. The most common ones are:

Map Type | Shape | Best For
---|---|---
`BPF_MAP_TYPE_HASH` | Hash table, key -> value | General counters, anything keyed by a name or ID
`BPF_MAP_TYPE_ARRAY` | Fixed-size array, index -> value | Configuration, lookup tables, when keys are 0..N
`BPF_MAP_TYPE_LRU_HASH` | Hash table that evicts oldest entries | When you might have too many keys to fit
`BPF_MAP_TYPE_PERCPU_HASH` | One hash table per CPU core | High-speed counters with no lock contention
`BPF_MAP_TYPE_PERCPU_ARRAY` | One array per CPU core | High-speed array counters
`BPF_MAP_TYPE_LPM_TRIE` | Longest prefix match trie | IP routing tables, ACLs
`BPF_MAP_TYPE_RINGBUF` | Lock-free ring buffer (since 5.8) | Streaming events to user-space, the modern way
`BPF_MAP_TYPE_PERF_EVENT_ARRAY` | Per-CPU perf event buffers | Streaming events to user-space, the older way
`BPF_MAP_TYPE_STACK_TRACE` | Stack trace store | Profilers, used by `bpf_get_stackid`
`BPF_MAP_TYPE_PROG_ARRAY` | Array of BPF programs | Tail calls, jumping from one program to another
`BPF_MAP_TYPE_DEVMAP` | Network devices | XDP redirect to another interface
`BPF_MAP_TYPE_CPUMAP` | CPUs | XDP redirect to another CPU for processing
`BPF_MAP_TYPE_XSKMAP` | AF_XDP sockets | XDP -> userspace fast path (DPDK-style)
`BPF_MAP_TYPE_SOCKMAP` | Sockets | sk_msg, socket-level message routing
`BPF_MAP_TYPE_SOCKHASH` | Sockets, hash-keyed | Like sockmap but with hash keys

### A picture of the map types

```
                 BPF Maps
                    |
    +---------------+---------------+
    |               |               |
 HASHISH        ARRAY-ISH       SPECIAL
    |               |               |
  hash           array          ringbuf  (events to user)
  lru_hash       percpu_array   perf_event_array
  percpu_hash    cgroup_storage stack_trace
  hash_of_maps   array_of_maps  prog_array (tail calls)
                                 devmap   (XDP redirect)
                                 cpumap   (XDP redirect)
                                 xskmap   (AF_XDP)
                                 sockmap  (sk_msg)
                                 sockhash (sk_msg)
                                 lpm_trie (IP routes)
                                 reuseport_sockarray
                                 task_storage
                                 inode_storage
                                 sk_storage
```

A lot of options. Most of the time you will use `hash`, `percpu_hash`, `array`, `percpu_array`, or `ringbuf`.

### Helper functions

Robots cannot call arbitrary kernel functions. They can only call a curated list of "helpers." These are functions the kernel provides that are guaranteed to be safe to call from a BPF program. There are about 200 of them.

Some you will see a lot:

- `bpf_map_lookup_elem(map, key)` — read from a map, returns a pointer.
- `bpf_map_update_elem(map, key, value, flags)` — write to a map.
- `bpf_map_delete_elem(map, key)` — remove an entry.
- `bpf_get_current_pid_tgid()` — returns a 64-bit value, top 32 bits are the TGID (process ID), bottom 32 bits are the PID (thread ID).
- `bpf_get_current_uid_gid()` — same idea but UID and GID.
- `bpf_get_current_comm(buf, size)` — copies the current process's command name into `buf`.
- `bpf_ktime_get_ns()` — returns nanoseconds since boot.
- `bpf_get_smp_processor_id()` — returns the current CPU number.
- `bpf_probe_read_kernel(dst, size, src)` — copy from a kernel pointer into the program's stack, safely.
- `bpf_probe_read_user(dst, size, src)` — copy from a user-space pointer into the program's stack, safely.
- `bpf_trace_printk(fmt, ...)` — print a debug line to `/sys/kernel/debug/tracing/trace_pipe`. Useful for debugging. Do not use in production.
- `bpf_perf_event_output(ctx, map, flags, data, size)` — emit an event to a perf buffer.
- `bpf_ringbuf_output(map, data, size, flags)` — emit an event to a ring buffer (since 5.8).
- `bpf_redirect(ifindex, flags)` — XDP/TC: redirect this packet to the named interface.
- `bpf_clone_redirect(skb, ifindex, flags)` — TC: clone the packet and redirect the clone, keep the original.
- `bpf_skb_load_bytes(skb, offset, to, len)` — load bytes from a TC skb.
- `bpf_get_stackid(ctx, map, flags)` — get a stack trace ID for the current call stack.

Each helper is documented in `man bpf-helpers` or in `/usr/include/linux/bpf.h`.

### Tail calls

Sometimes one robot wants to call another robot. This is called a "tail call." The robot says "I am done, please run this other robot in my place." The other robot picks up where the first left off. The first robot does not return.

Tail calls let you build big programs out of small programs. If your one program is too big to fit in the verifier's instruction limit (1 million), you split it into smaller programs and chain them with tail calls. Each program counts separately for the verifier. The chain can be up to 33 programs deep.

Tail calls use a special map: `BPF_MAP_TYPE_PROG_ARRAY`. The user-space program populates the array with file descriptors of loaded BPF programs. A robot calls `bpf_tail_call(ctx, &prog_array, index)` and execution jumps to the program at that index.

## Where Robots Get Attached

Now we get to the rooms. The brain of your computer has many rooms. eBPF robots can attach to many of them. Each room sees something different. Picking the right room for your job is half the battle.

### kprobe — any kernel function

A **kprobe** is a "kernel probe." You pick a kernel function (any kernel function in the whole kernel, of which there are around 100,000) and a kprobe robot attaches there. Every time the kernel calls that function, your robot wakes up. The robot gets the CPU registers, which means it can read the function's arguments. The robot can record what happened.

Example: attach a kprobe to `vfs_read`. Every time anyone reads any file, your robot wakes up and counts it. You now know how many file reads happened.

Strengths: kprobes can attach to nearly any function. They are extremely flexible.

Weaknesses: kprobes break across kernel versions. The function you attach to might be renamed or inlined or removed in a future kernel. Your robot will fail to attach. You will need to update.

You'll see kprobes written as `kprobe:vfs_read` (in bpftrace) or `SEC("kprobe/vfs_read")` (in C).

There is also `kretprobe`, which fires when the function returns. Together with `kprobe`, this lets you measure how long the function took.

Since kernel 5.5 there are `fentry` and `fexit`, which do the same job but faster (they use a different mechanism, BPF trampolines, that has lower overhead than the kprobe machinery). Use `fentry`/`fexit` when you can.

### uprobe — any user-space function

A **uprobe** is the user-space version of a kprobe. You pick a function in some user-space binary (like `/usr/bin/python3` or `/usr/local/bin/myapp`) and a uprobe robot attaches there. Every time anyone runs that function, your robot wakes up.

Strengths: you can trace anything in user-space, including without modifying the source code.

Weaknesses: uprobes are slower than kprobes (they use software interrupts) and break if the binary is recompiled.

There is also `uretprobe` for return events.

### tracepoint — stable kernel hooks

A **tracepoint** is a hook the kernel maintainers explicitly added to the kernel for tracing. Tracepoints have stable names and stable arguments. Your robot will keep working across kernel versions.

Tracepoints look like `tracepoint:syscalls:sys_enter_openat` (the syscalls subsystem, the `sys_enter_openat` event).

You can list available tracepoints with `bpftrace -l 'tracepoint:*'` or by reading `/sys/kernel/debug/tracing/events/`.

Strengths: stable across kernel versions. Lower overhead than kprobes.

Weaknesses: only the events the kernel maintainers thought to add. If you want to trace a function that has no tracepoint, you need a kprobe.

### USDT — user-space stable hooks

**USDT** stands for "User Statically-Defined Tracing." It is the user-space equivalent of a tracepoint: a stable hook that the application's authors added on purpose. PostgreSQL has USDTs. MySQL has USDTs. Python has USDTs. Node.js has USDTs.

Strengths: stable across application versions. Designed for tracing.

Weaknesses: only available if the application's authors added them. Most applications do not have USDTs.

### XDP — packets at the driver

**XDP** stands for "eXpress Data Path." This is the room closest to the network card. Specifically: when a packet arrives at the network card, before the kernel does almost any processing, the XDP robot gets a look at it.

XDP robots can do four things:
- `XDP_DROP`: throw the packet away. The kernel never sees it.
- `XDP_PASS`: pass the packet up to the kernel for normal processing.
- `XDP_TX`: bounce the packet right back out the same network card.
- `XDP_REDIRECT`: send the packet out a different interface, or to a different CPU, or to user-space via AF_XDP.

XDP is the fastest way to handle packets in Linux. It is the basis of high-performance load balancers (Facebook's Katran), DDoS mitigation (Cloudflare uses XDP), and tools like Cilium.

A modern NIC with an XDP-capable driver running in "native mode" can handle line rate (full network speed) with XDP, which is on the order of 14 million packets per second on a 10-gigabit link with 64-byte packets.

There is also a "generic" XDP mode that runs in software, slower but compatible with all drivers. Use native if you can. Use generic if you must.

### tc — traffic control

**tc** stands for "traffic control." It is a slightly later spot in the network stack than XDP. By the time a packet gets to tc, the kernel has built an `sk_buff` (the kernel's packet metadata structure). This costs a little CPU but gives you more information.

tc programs run in two places: ingress (incoming packets) and egress (outgoing packets). XDP only runs on ingress. So if you want to filter outgoing packets, you need tc.

tc programs return one of `TC_ACT_OK` (let it pass), `TC_ACT_SHOT` (drop), `TC_ACT_REDIRECT` (send elsewhere), and a few others.

### sk_msg, sk_skb — socket-level

**sk_msg** and **sk_skb** robots attach to individual sockets. They see messages flowing through the socket and can redirect them, drop them, or modify them. Cilium uses these for socket-level service mesh acceleration: bypassing the network stack entirely for connections between two pods on the same node.

### LSM — security checks

**LSM** stands for "Linux Security Module." LSM is a system in the kernel that lets security software hook into permission checks: every time the kernel asks "is this allowed?" the LSM hooks fire. SELinux is an LSM. AppArmor is an LSM.

Since kernel 5.7, BPF can attach to LSM hooks. Your BPF program can deny operations: file opens, socket creates, capability checks, ptrace attaches, anything the LSM framework hooks. This is how modern security tools like Tetragon enforce policy without a kernel module.

### fentry / fexit — fast function tracing

**fentry** ("function entry") and **fexit** ("function exit") are like kprobe and kretprobe, but faster. They use a mechanism called BPF trampolines, introduced in kernel 5.5. The trampoline is generated on the fly to glue your BPF program to the function with minimum overhead.

If you are on 5.5 or newer and the function you want to trace is supported by fentry, use fentry. It is a couple of times faster than kprobe.

There is also **fmod_ret** ("function modify return"), which lets you change the return value of a kernel function. This is rare and powerful and a little scary.

### perf_event — sampling

**perf_event** robots attach to a perf event source. The most useful kind is "profile at frequency X," which runs your robot every (1/X) seconds on every CPU. This is how you build CPU profilers.

You can also attach perf_event robots to hardware counters: cache misses, branch mispredicts, instructions retired. If your CPU supports a counter, perf can sample it, and BPF can run on every sample.

### cgroup hooks

**cgroup_skb**, **cgroup_sock**, **cgroup_sock_addr**, and friends are robots that attach to cgroups (control groups, the kernel feature that powers Docker and Kubernetes). They run only for processes inside that cgroup. This is how Cilium and similar tools enforce per-pod network policy without iptables.

### sock_ops

**sock_ops** robots attach to TCP socket events: connection establishment, retransmits, RTT estimates. They are how you implement custom TCP congestion control or socket tuning.

### Other attach points

There are more:
- `raw_tracepoint` — a faster, lower-level version of tracepoint.
- `iter` — robots that iterate over kernel data structures (since 5.8).
- `struct_ops` — robots that implement a kernel interface (since 5.6, used for custom TCP CC algorithms).
- `flow_dissector` — robots that classify flows for the network stack.
- `socket_filter` — the original BPF program type, packet filtering on a socket (this is what `tcpdump` uses).
- `sched_cls`, `sched_act` — older names for tc programs.

You will not need most of these. Stick to kprobe, uprobe, tracepoint, XDP, tc, perf_event, and LSM and you will cover 99% of what people do with eBPF.

### A picture of the attach points

```
              eBPF Attach Points
                     |
    +----------------+-----------------+
    |                |                 |
 NETWORK         FUNCTIONS         SECURITY
    |                |                 |
  XDP            kprobe            LSM (5.7+)
  TC             kretprobe         seccomp
  cgroup_skb     uprobe
  cgroup_sock    uretprobe
  sk_msg         tracepoint
  sk_skb         raw_tracepoint
  sock_ops       USDT
  flow_dissector fentry  (5.5+)
                 fexit   (5.5+)
                 fmod_ret(5.5+)
                 perf_event
                 iter    (5.8+)
                 struct_ops (5.6+)
```

That tree is not exhaustive but it covers what you will use.

## Maps and Communication

Now we will talk about how the robots and the user-space program talk to each other.

### The bpf() syscall

There is a system call (a request to the kernel) called `bpf()`. Almost everything you do with BPF goes through this one syscall, with different "commands" telling it what you want.

```
bpf(cmd, attr, size)
```

`cmd` is one of:

- `BPF_PROG_LOAD` — load a BPF program. The kernel verifies it and JIT-compiles it. Returns a file descriptor.
- `BPF_MAP_CREATE` — create a map. Returns a file descriptor.
- `BPF_MAP_LOOKUP_ELEM` — read an entry from a map.
- `BPF_MAP_UPDATE_ELEM` — write an entry to a map.
- `BPF_MAP_DELETE_ELEM` — remove an entry.
- `BPF_PROG_ATTACH` — attach a loaded program to a hook.
- `BPF_PROG_DETACH` — detach a program from a hook.
- `BPF_OBJ_PIN` — pin a program or map to a path under `/sys/fs/bpf/` so it survives the loader exiting.
- `BPF_OBJ_GET` — get a pinned program or map by path.
- A bunch more for less-common operations.

You almost never call `bpf()` directly. You use a library.

### libbpf

**libbpf** is the standard C library for working with BPF. It hides the syscall ugliness behind a clean API. You point libbpf at a `.o` file (an ELF file containing your compiled BPF programs and map definitions), and libbpf:

1. Parses the ELF.
2. Creates the maps via `bpf(BPF_MAP_CREATE)`.
3. Loads the programs via `bpf(BPF_PROG_LOAD)`.
4. Attaches the programs to their hooks via `bpf(BPF_PROG_ATTACH)` or one of the per-hook syscalls.
5. Returns handles you can use to read maps, send events, etc.

libbpf is the official, kernel-blessed way to load BPF programs. Everyone uses it.

### BPF skeletons

A **BPF skeleton** is an auto-generated header file that wraps libbpf for a specific BPF object. You run `bpftool gen skeleton my_prog.bpf.o > my_prog.skel.h`, include `my_prog.skel.h` in your user-space program, and now you have typed access to your maps and programs:

```c
struct my_prog *skel = my_prog__open_and_load();
my_prog__attach(skel);
// access skel->maps.my_map, skel->progs.my_prog, etc.
my_prog__destroy(skel);
```

This is the modern C/C++ way to write eBPF tools. Most projects in `libbpf-bootstrap` use skeletons.

### Other languages

You don't have to use C in user-space.

- **Go**: `cilium/ebpf` is a popular pure-Go library. Cilium itself is written in Go and uses this library.
- **Rust**: `aya` is a pure-Rust library, used in the Unheaded Kingdom and others. `redbpf` is older.
- **Python**: `bcc` (BPF Compiler Collection) is the original Python framework for eBPF. It compiles BPF C at runtime, which is convenient but slower.
- **Ruby/Lua/Node**: less common but exists.

The kernel does not care what language you wrote your loader in. It only sees the bytecode and the syscalls.

### Uploading the firmware

A useful mental model: loading a BPF program is like uploading firmware to the kernel. You write the firmware in C. You compile it. You upload it. The kernel executes it. When you want to update the firmware, you upload a new version. When you want to remove it, you say "remove."

Just like how a router has firmware, your kernel now has BPF programs running inside it. You can ship your software as a BPF program plus a tiny user-space loader, and that combination acts like a kernel module that cannot crash the kernel.

## CO-RE (Compile Once Run Everywhere)

Now we get to the deep magic.

### The portability problem

Here is the problem CO-RE solves. The Linux kernel is constantly changing. Internal data structures (structs) get fields added, fields removed, fields renamed, fields rearranged. The `task_struct` (the kernel's struct for "a process or thread") looks different on kernel 5.4 than it does on 5.10 than it does on 6.6.

If your BPF program reads `task->tgid` (the process ID), the offset of `tgid` inside `task_struct` is different on every kernel. Your compiled bytecode hardcodes the offset that was true on the kernel you compiled for. Run it on a different kernel and you read garbage.

The old fix was to recompile your BPF program on every machine. Tools like `bcc` did exactly that: they shipped C source code, and at runtime they compiled the source on your machine using your machine's kernel headers. This worked but was slow, required clang and headers on every machine, and was a pain.

### The CO-RE fix: BTF + relocations

CO-RE stands for **Compile Once, Run Everywhere**. The idea: compile your BPF program once, on your laptop, and run that one binary on any kernel from 5.4 onward.

How? Two pieces.

**BTF** (BPF Type Format) is a compact encoding of all the type information about the kernel: every struct, every union, every enum, every function signature. The kernel ships its own BTF in `/sys/kernel/btf/vmlinux`. You can dump it: `bpftool btf dump file /sys/kernel/btf/vmlinux format c > vmlinux.h` produces a giant header file with every kernel type in it.

When you compile your BPF program, instead of reading `task->tgid` directly, you use a special macro `BPF_CORE_READ(task, tgid)`. The compiler emits a CO-RE relocation: "at this point, please read the offset of `tgid` inside `task_struct`."

When you load the program, libbpf reads the running kernel's BTF, finds `task_struct`, finds the `tgid` field, looks up its offset for THIS kernel, and patches your bytecode to use the right offset. The relocations get fixed up at load time.

So your one binary, compiled once, runs on any kernel that has BTF. Since 5.4, every mainline kernel has BTF. Major distributions enable BTF in their kernel builds. Most modern Linux you encounter will have BTF.

### A picture of CO-RE

```
Compile time (on your laptop)
=============================

Your C source:
    pid_t pid = BPF_CORE_READ(task, tgid);

clang compiles this. Instead of hardcoding the offset of `tgid`,
it emits a relocation: "at this load instruction, look up the
offset of `task_struct.tgid` in the target kernel's BTF and
patch this immediate value."

Bytecode (with relocation):
    r1 = *(u64 *)(r1 + RELOCATABLE_OFFSET)
                          ^^^^^^^^^^^^^^^^
                          gets patched at load time



Load time (on the target machine)
=================================

libbpf reads /sys/kernel/btf/vmlinux:
    finds struct task_struct
    finds field tgid
    sees offset is, say, 0x920 on this kernel
    patches the bytecode: RELOCATABLE_OFFSET -> 0x920

Now the bytecode has the right offset for THIS kernel.
The verifier accepts it. The JIT compiles it.
The program attaches. It runs correctly.

Same binary on a different kernel: different offset, libbpf
patches in the new offset. Same binary, different machine,
correct behavior. CO-RE.
```

### Why this is amazing

Before CO-RE, distributing eBPF tools was a nightmare. With CO-RE, you ship one statically-linked binary and it works everywhere. This is why modern eBPF tools (Cilium, Tetragon, Falco, Pixie, Parca, Beyla) work on basically any Linux box without any per-machine setup. CO-RE made eBPF practical for production.

### Vmlinux.h

When you write CO-RE-style BPF C, you do not include the kernel's actual headers. You include `vmlinux.h`, which is a single giant header file generated from the kernel's BTF. It contains type definitions for every kernel type in one place. You generate it with:

```
bpftool btf dump file /sys/kernel/btf/vmlinux format c > vmlinux.h
```

You commit it to your repo. You include it in your BPF C. You are good.

### Pinning

If your loader exits, the kernel removes its programs and maps. To keep them around, you can "pin" them to a path in the special filesystem `/sys/fs/bpf/`. A pinned object survives the loader exiting; another process can pick up the same object by path.

```
bpftool prog pin id 42 /sys/fs/bpf/my_prog
bpftool map pin id 17 /sys/fs/bpf/my_map
```

Or in code with `BPF_OBJ_PIN`. Many libbpf-based tools have an "auto-pin" mode where the loader pins everything on startup.

## What People Actually Use eBPF For

Now we will tour the four big use cases.

### 1. Observability — see what is happening

The first big use case is observability: looking at what the kernel and applications are doing, in real time, with no agent and no code changes.

bpftrace is the star here. We will spend a whole section on it later. With one line you can:

- Count syscalls per process.
- Trace every file open by every process.
- See every TCP retransmit on the box.
- Build a histogram of read sizes.
- Profile what every CPU is doing 99 times a second.

Tools that build on this idea:
- **bcc** (`/usr/share/bcc/tools/`) — a Python framework with hundreds of pre-written tools (`opensnoop`, `execsnoop`, `tcpaccept`, `biolatency`, `runqlat`, ...).
- **bpftrace** — high-level scripting, the awk of the kernel.
- **Pixie** — a CNCF project that auto-instruments Kubernetes clusters with eBPF and gives you a SQL/PXL query language over the result.
- **Parca** — continuous profiling. Always-on flame graphs of every process.
- **Pyroscope** — same idea, different vendor.
- **Grafana Beyla** — auto-instrument any HTTP/gRPC/SQL service for metrics and traces.

### 2. Networking — process packets fast

The second big use case is networking: load balancers, firewalls, service meshes, DDoS protection.

- **Cilium** is the headliner. Cilium is a Kubernetes CNI (container network interface) plug-in that uses eBPF for everything: pod networking, NetworkPolicy enforcement, kube-proxy replacement, service load balancing, transparent encryption, observability (Hubble), and a service mesh. It scales to thousands of pods per node where iptables-based stacks fall over.
- **Katran** is Facebook's L4 load balancer. eBPF/XDP at line rate.
- **Cloudflare** uses XDP for DDoS mitigation: bad packets get dropped at the NIC, never enter the kernel stack, never hit the CPU for parsing.
- **Calico** also uses eBPF for some of its Kubernetes networking modes.

The key insight: eBPF programs are flexible (you can write whatever logic you want) and fast (JIT-compiled, near-native). With XDP you process packets before the kernel does anything. With tc you process them with full sk_buff context. With sockmap and sk_msg you bypass the network stack entirely between local sockets. The combinations are the future of network software.

### 3. Security — enforce policy

The third big use case is security: detecting suspicious behavior and (with BPF LSM) blocking it.

- **Falco** is the original eBPF security tool. It watches for syscalls and other events that match a rule (like "execve of /bin/sh inside a container that should not be running shells") and alerts. Falco does not block; it just alerts.
- **Tetragon** (Cilium's runtime-security sibling) does both: it watches with eBPF and can also enforce, via BPF LSM hooks, by killing the offending process or refusing the syscall.
- **Tracee** (from Aqua Security) is similar: comprehensive runtime tracing focused on security events.
- **Capabilities watching**: simple eBPF programs can detect when a process changes its UID, when a kernel module is loaded, when a binary executes, when sensitive files are read.

### 4. Performance — find the slow stuff

The fourth big use case is performance investigation. Why is my disk slow? Why is my app slow? Where are the locks contending? Where is the kernel spending time?

bcc tools are the bread and butter:
- `biolatency` — histogram of disk I/O latencies.
- `biotop` — top-style view of disk I/O by process.
- `runqlat` — histogram of how long processes wait on the run queue (scheduler latency).
- `cpudist` — histogram of CPU time slices.
- `tcpconnect`, `tcpaccept`, `tcpretrans` — TCP visibility.
- `funclatency` — measure how long any kernel function takes.
- `softirqs`, `hardirqs` — interrupt time.
- `slabratetop` — kernel memory allocations.
- `cachetop`, `cachestat` — file cache hits/misses.
- `pidstat`, `vmstat`, `iostat` — but for things the legacy tools cannot see.

### 5. Profiling — flame graphs without instrumenting

The fifth (related) use case: continuous profiling. Tools like `parca`, `pyroscope`, and `bcc/profile` use perf_event sampling at, say, 99 Hz to grab the kernel and user stack on every CPU, and aggregate that into flame graphs.

The output: you can see exactly where every CPU cycle in your fleet is being spent, with no instrumentation in your application, no SDK, no profiler library. You just turn it on and look.

Brendan Gregg's flame graphs are the picture you want in your head: every box is a function, the wider the box the more time spent there, and you can see the whole call tree from kernel scheduler down through your app's deepest call. eBPF is what makes this practical to leave on in production.

## bpftrace: The Awk of the Kernel

bpftrace is a high-level scripting language for eBPF. It is to eBPF what awk is to text processing: terse, expressive, perfect for one-liners.

You write a bpftrace one-liner like this:

```
bpftrace -e 'tracepoint:syscalls:sys_enter_openat { @[comm] = count(); }'
```

Read that left to right.
- `tracepoint:syscalls:sys_enter_openat` — attach to the `sys_enter_openat` tracepoint (this fires every time anyone makes an `openat` syscall, which is most file opens).
- `{ ... }` — the action block, what to do when the event fires.
- `@[comm] = count();` — there is a map called `@` (the default map) keyed by `comm` (the current process's command name), and we add 1 to that key.

Run that for ten seconds, hit Ctrl-C, and bpftrace prints the map. You see exactly which programs opened files and how many.

bpftrace compiles the script down to BPF bytecode and loads it. You never see the bytecode. You never call `bpf()`. bpftrace handles all of that.

### bpftrace one-liners

**Count syscalls per process:**
```
bpftrace -e 'tracepoint:raw_syscalls:sys_enter { @[comm] = count(); }'
```

**Trace every execve (every program launch):**
```
bpftrace -e 'tracepoint:syscalls:sys_enter_execve { printf("%s %s\n", comm, str(args->filename)); }'
```

**Count file opens by process:**
```
bpftrace -e 'tracepoint:syscalls:sys_enter_openat { @[comm] = count(); }'
```

**Histogram of file read sizes:**
```
bpftrace -e 'tracepoint:syscalls:sys_exit_read /args->ret > 0/ { @bytes = hist(args->ret); }'
```

**Count TCP retransmits per process:**
```
bpftrace -e 'kprobe:tcp_retransmit_skb { @[comm] = count(); }'
```

**TCP connections initiated, with destination IP:**
```
bpftrace -e 'kprobe:tcp_connect { printf("%s -> %s\n", comm, ntop(((struct sock *)arg0)->__sk_common.skc_daddr)); }'
```

**CPU profile at 99 Hz, kernel + user stacks:**
```
bpftrace -e 'profile:hz:99 { @[ustack, kstack] = count(); }'
```

**Histogram of vfs_read latencies:**
```
bpftrace -e '
  kprobe:vfs_read { @start[tid] = nsecs; }
  kretprobe:vfs_read /@start[tid]/ {
    @ns = hist(nsecs - @start[tid]); delete(@start[tid]);
  }'
```

These are not toys. These are the kinds of scripts production engineers run when something is wrong.

### bpftrace built-ins

Inside an action block you have:
- `comm` — the current process's command name, 16 chars.
- `pid` — the current process ID.
- `tid` — the current thread ID.
- `uid` / `gid` — UID and GID.
- `cpu` — the current CPU.
- `nsecs` — nanoseconds since boot.
- `arg0`, `arg1`, ... — function arguments (kprobe).
- `args->whatever` — tracepoint arguments.
- `retval` — return value (kretprobe).
- `ustack`, `kstack` — user / kernel stack trace.
- `str(p)` — turn a pointer into a string (with safe copy).
- `printf(fmt, ...)` — print.
- `count()`, `sum(x)`, `avg(x)`, `min(x)`, `max(x)`, `hist(x)`, `lhist(x, lo, hi, step)` — aggregations.

### bpftrace probe types

Probe types you can attach to:
- `tracepoint:subsys:event` — kernel tracepoints.
- `kprobe:func` / `kretprobe:func` — kernel function entry/exit.
- `uprobe:/path:func` / `uretprobe:/path:func` — user-space function entry/exit.
- `usdt:/path:provider:probe` — USDT probes.
- `profile:hz:N` — sample every CPU at N Hz.
- `interval:s:N` — fire on a wall-clock interval.
- `BEGIN` / `END` — at script start / end.
- `software:event:rate` / `hardware:event:rate` — perf software/hardware events.
- `watchpoint:addr:size:rwx` — hardware watchpoint.

Look at `man bpftrace` for the full list.

## Hands-On

This section is paste-and-runnable. Most of these need root or `CAP_BPF`. Where they don't, that is noted. Expected outputs will vary by machine and load — what we show is the *shape* of the output you should expect.

### List loaded BPF programs

```
$ bpftool prog list 2>&1 | head -10
1: cgroup_skb  tag 6deef7357e7b4530  gpl
        loaded_at 2026-04-27T10:14:23+0000  uid 0
        xlated 64B  jited 54B  memlock 4096B  map_ids 4,5
2: cgroup_skb  tag 6deef7357e7b4530  gpl
        loaded_at 2026-04-27T10:14:23+0000  uid 0
        xlated 64B  jited 54B  memlock 4096B  map_ids 4,5
3: cgroup_device  tag c879cc44dafd1996  gpl
        loaded_at 2026-04-27T10:14:24+0000  uid 0
```

That's a typical list on a Docker host: cgroup-skb programs are systemd's per-service network filtering, cgroup_device is for device cgroup controls.

### List BPF maps

```
$ bpftool map list 2>&1 | head -10
4: array  name <anon>  flags 0x0
        key 4B  value 8B  max_entries 2  memlock 4096B
5: array  name <anon>  flags 0x0
        key 4B  value 8B  max_entries 2  memlock 4096B
```

Each map shows its type, key/value sizes, and how big it is.

### Probe BPF features

```
$ bpftool feature probe | head -30
Scanning system call availability...
bpf() syscall is available

Scanning eBPF program types...
eBPF program_type socket_filter is available
eBPF program_type kprobe is available
eBPF program_type sched_cls is available
eBPF program_type sched_act is available
eBPF program_type tracepoint is available
eBPF program_type xdp is available
eBPF program_type perf_event is available
eBPF program_type cgroup_skb is available
eBPF program_type cgroup_sock is available
eBPF program_type lwt_in is available
```

This tells you what your kernel supports. Newer kernels support more.

### How many kernel functions can you probe?

```
$ cat /sys/kernel/debug/tracing/available_filter_functions | wc -l
46812
```

A lot. You can attach a kprobe to any of them.

### Is unprivileged BPF allowed?

```
$ cat /proc/sys/kernel/unprivileged_bpf_disabled
2
```

`0` means anyone can load BPF programs. `1` means only root or CAP_BPF can. `2` means only CAP_BPF (no longer reading from `unprivileged_bpf_disabled` flag at runtime). On most modern distros this is `2`. Setting it back to `0` is a security risk.

### Count file opens per process for 10 seconds

```
$ sudo bpftrace -e 'tracepoint:syscalls:sys_enter_openat { @[comm] = count(); }'
Attaching 1 probe...
^C

@[gnome-shell]: 47
@[chrome]: 612
@[bash]: 8
@[systemd]: 124
```

Press Ctrl-C after a few seconds and bpftrace prints the map.

### Trace every program launched (execve)

```
$ sudo bpftrace -e 'tracepoint:syscalls:sys_enter_execve { printf("%s %s\n", comm, str(args->filename)); }'
Attaching 1 probe...
bash /usr/bin/ls
ls /usr/bin/ls
make /usr/bin/cc
cc1 /usr/libexec/gcc/x86_64-linux-gnu/13/cc1
^C
```

Open a new shell, run a command — you will see it appear here.

### Count vfs_read calls per process

```
$ sudo bpftrace -e 'kprobe:vfs_read { @[comm] = count(); }'
Attaching 1 probe...
^C

@[node]: 12
@[bash]: 3
@[chrome]: 4521
```

### Histogram of vfs_read return sizes

```
$ sudo bpftrace -e 'kretprobe:vfs_read /retval > 0/ { @bytes = hist(retval); }'
Attaching 1 probe...
^C

@bytes:
[1]                    8 |@@@                                                 |
[2, 4)                12 |@@@@                                                |
[4, 8)                23 |@@@@@@@@                                            |
[8, 16)               44 |@@@@@@@@@@@@@@@                                     |
[16, 32)              67 |@@@@@@@@@@@@@@@@@@@@@@                              |
[32, 64)              92 |@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@                      |
[64, 128)            134 |@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@        |
[128, 256)           150 |@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@|
[256, 512)            87 |@@@@@@@@@@@@@@@@@@@@@@@@@@@@@                       |
[512, 1K)             45 |@@@@@@@@@@@@@@@                                     |
[1K, 2K)              22 |@@@@@@@                                             |
[2K, 4K)              11 |@@@                                                 |
[4K, 8K)               4 |@                                                   |
```

A real read-size distribution.

### CPU profile at 99 Hz

```
$ sudo bpftrace -e 'profile:hz:99 { @[ustack, kstack] = count(); }' -o /tmp/flame.txt
Attaching 1 probe...
^C
```

Lets it run for 30 seconds, you Ctrl-C, the file `/tmp/flame.txt` has stack traces and counts. Pipe through `flamegraph.pl` for a flame graph.

### Open-snoop with bcc

```
$ sudo opensnoop-bpfcc 2>&1 | head -10
PID    COMM              FD ERR PATH
1234   chrome            45   0 /etc/passwd
1234   chrome            46   0 /etc/nsswitch.conf
2345   systemd            8   0 /proc/1/cgroup
3456   bash              -1   2 /tmp/nope
1234   chrome            47   0 /home/user/.config/chrome/Cookies
```

Live tail of every file-open on the box.

### Execve-snoop with bcc

```
$ sudo execsnoop-bpfcc 2>&1 | head -10
PCOMM            PID    PPID   RET ARGS
bash             5678   1234     0 /usr/bin/ls --color=auto
ls               5679   5678     0 /usr/bin/ls --color=auto
make             5680   1234     0 /usr/bin/make
cc1              5681   5680     0 /usr/libexec/gcc/x86_64-linux-gnu/13/cc1
```

Every program launched.

### TCP-accept with bcc

```
$ sudo tcpaccept-bpfcc 2>&1 | head -10
PID    COMM         IP RADDR            RPORT LADDR            LPORT
1234   nginx         4 192.0.2.10        43210 198.51.100.5     443
1234   nginx         4 192.0.2.11        43217 198.51.100.5     443
2345   sshd          4 203.0.113.7       55512 198.51.100.5      22
```

Every accepted TCP connection on the box.

### bio-latency histogram

```
$ sudo biolatency-bpfcc 2>&1 | head -20
Tracing block device I/O... Hit Ctrl-C to end.
^C
     usecs               : count     distribution
         0 -> 1          : 0        |                                        |
         2 -> 3          : 0        |                                        |
         4 -> 7          : 12       |                                        |
         8 -> 15         : 87       |@@@                                     |
        16 -> 31         : 432      |@@@@@@@@@@@@@@@                         |
        32 -> 63         : 1109     |@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@|
        64 -> 127        : 743      |@@@@@@@@@@@@@@@@@@@@@@@@@@@             |
       128 -> 255        : 312      |@@@@@@@@@@@                             |
       256 -> 511        : 89       |@@@                                     |
       512 -> 1023       : 24       |                                        |
      1024 -> 2047       : 7        |                                        |
      2048 -> 4095       : 2        |                                        |
```

Distribution of disk I/O latencies. If you see things in the millisecond bins, your disk is unhappy.

### Run-queue latency histogram

```
$ sudo runqlat-bpfcc 2>&1 | head -20
Tracing run queue latency... Hit Ctrl-C to end.
^C

     usecs               : count     distribution
         0 -> 1          : 12       |                                        |
         2 -> 3          : 89       |@@@                                     |
         4 -> 7          : 432      |@@@@@@@@@@@@@@@                         |
         8 -> 15         : 1023     |@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@    |
        16 -> 31         : 1198     |@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@|
        32 -> 63         : 632      |@@@@@@@@@@@@@@@@@@@@@                   |
        64 -> 127        : 187      |@@@@@@                                  |
       128 -> 255        : 47       |@                                       |
```

How long processes wait on the CPU run queue. If the bins past 100us are full, the box is CPU-saturated.

### Show one program's details

```
$ bpftool prog show id 42
42: kprobe  name kprobe__vfs_read  tag c4d8b8c7a1b59fff  gpl
        loaded_at 2026-04-27T10:30:15+0000  uid 0
        xlated 256B  jited 198B  memlock 4096B  map_ids 17
        btf_id 9
```

`xlated` = size of the BPF bytecode after translation. `jited` = size of the native machine code. `map_ids` = which maps it uses.

### Disassemble a BPF program

```
$ bpftool prog dump xlated id 42
   0: (bf) r6 = r1
   1: (85) call bpf_get_current_pid_tgid#136864
   2: (77) r0 >>= 32
   3: (63) *(u32 *)(r10 -4) = r0
   4: (bf) r2 = r10
   5: (07) r2 += -4
   6: (18) r1 = map[id:17]
   8: (85) call bpf_map_lookup_elem#1
   ...
```

You can see the actual eBPF bytecode. Helpful for debugging.

### Dump a map's contents

```
$ bpftool map dump id 17
key: 4d 65 5f 31 32 33 00 00 00 00 00 00 00 00 00 00  value: 0c 00 00 00 00 00 00 00
key: 4e 65 5f 34 35 36 00 00 00 00 00 00 00 00 00 00  value: 03 00 00 00 00 00 00 00
Found 2 elements
```

Hex dump of every entry. Combine with `--pretty` for JSON.

### Read printk-style debug output

```
$ sudo cat /sys/kernel/debug/tracing/trace_pipe
          chrome-1234  [003] d..1.  9472.123: bpf_trace_printk: hello from BPF
          systemd-1     [001] d..1.  9472.234: bpf_trace_printk: opened /etc/passwd
```

If your BPF program calls `bpf_trace_printk("...")`, the output ends up here. Useful for debugging your program.

### Check bpftool's capabilities

```
$ getcap /usr/bin/bpftool
/usr/bin/bpftool cap_bpf,cap_perfmon,cap_sys_admin=ep
```

If `bpftool` is set up with file capabilities, you don't need sudo for read-only ops.

### See kernel BPF errors in dmesg

```
$ dmesg | grep -i bpf | tail -20
[ 1234.567890] bpf: Failed to load program: Permission denied
[ 1235.123456] bpf: arg#0 expected pointer to ctx, but got pkt
[ 2345.678901] bpf: verification time 124 usec
[ 2345.679000] bpf: stack depth 256
[ 2345.679100] bpf: processed 1234 insns (limit 1000000)
```

Kernel logs verifier complaints and load events here.

### Your shell's capability set

```
$ cat /proc/$$/status | grep -i cap
CapInh: 0000000000000000
CapPrm: 0000000000000000
CapEff: 0000000000000000
CapBnd: 000001ffffffffff
CapAmb: 0000000000000000
```

If `CapEff` does not have `CAP_BPF` (bit 39), you cannot load BPF programs as this shell. (Hex 800000... has bit 39 set; check `capsh --decode=<hex>` for human-readable.)

### List TCP-related kprobes you could attach to

```
$ sudo bpftrace -l '*tcp*' | head -20
kprobe:tcp_ack
kprobe:tcp_check_oom
kprobe:tcp_close
kprobe:tcp_connect
kprobe:tcp_create_openreq_child
kprobe:tcp_disconnect
kprobe:tcp_fin
kprobe:tcp_init_sock
kprobe:tcp_release_cb
kprobe:tcp_retransmit_skb
kprobe:tcp_send_active_reset
kprobe:tcp_sendmsg
kprobe:tcp_v4_connect
kprobe:tcp_v6_connect
tracepoint:tcp:tcp_destroy_sock
tracepoint:tcp:tcp_probe
tracepoint:tcp:tcp_rcv_space_adjust
tracepoint:tcp:tcp_receive_reset
tracepoint:tcp:tcp_retransmit_skb
tracepoint:tcp:tcp_send_reset
```

That's the menu of TCP events you can hook.

### List all BTF info on the kernel

```
$ sudo bpftool btf list | head -10
1: name [vmlinux]  size 5267824B
2: name [bpf_testmod]  size 1234B
9: name [my_program]  size 10288B
```

Each BPF program with BTF gets its own entry. `vmlinux` is the kernel's own BTF.

### Dump kernel BTF as a header

```
$ sudo bpftool btf dump file /sys/kernel/btf/vmlinux format c | head -30
#ifndef __VMLINUX_H__
#define __VMLINUX_H__
typedef unsigned char __u8;
typedef short int __s16;
typedef short unsigned int __u16;
typedef int __s32;
typedef unsigned int __u32;
typedef long long int __s64;
typedef long long int __u64;
...
struct task_struct {
    struct thread_info thread_info;
    unsigned int __state;
    void *stack;
    refcount_t usage;
    ...
};
```

This is the magic header that powers CO-RE.

### Show BPF prog histograms

```
$ bpftool prog profile id 42 duration 5 cycles instructions
  10312987 cycles (100.00%)
   8721344 instructions (100.00%)
```

Profile a BPF program itself: how many cycles and instructions it uses while running. Helpful to confirm your program is not too heavy.

### Pin a program

```
$ sudo bpftool prog pin id 42 /sys/fs/bpf/my_prog
$ ls /sys/fs/bpf/
my_prog
```

Pinned. Now another process can `BPF_OBJ_GET` it without needing the original loader.

## Common Confusions

This is the part where we answer the 12+ things that confuse everyone the first time they meet eBPF.

### 1. "Why is eBPF safe but kernel modules aren't?"

Because of the verifier. Kernel modules can do anything: they share the kernel's memory space and have full access. There is no safety net. eBPF programs go through the verifier first, which proves the program cannot crash the kernel: bounded loops, in-bounds memory access, defined return values, only allowed helper calls. If the verifier says no, the program never runs. Kernel modules have no such gatekeeper.

### 2. "Why does my eBPF program get rejected?"

The verifier found something it could not prove safe. The most common reasons:

- You did a memory access without a bounds check. Add an `if (ptr + size > data_end) return XDP_PASS;` first.
- You used an uninitialized register (often the return value). Add `return 0;` or similar.
- You have a loop the verifier can't prove terminates. Use bounded loops or `bpf_loop()`.
- Your program is too big. Split with tail calls.
- You used too much stack. Move buffers to a map.
- You called a helper that's not allowed for your program type. Check `bpf_helpers.h`.

The kernel's error message tells you which line and which register. Read it carefully.

### 3. "What's the difference between bcc and libbpf?"

**bcc** (BPF Compiler Collection) ships your BPF program as C source, and at runtime it compiles the source on the target machine using the target's kernel headers. It needs clang and headers on every machine. It's slower to start (compile happens at startup). It's been around longer.

**libbpf** (the modern way) ships your BPF program as already-compiled bytecode plus CO-RE relocations. At runtime libbpf reads the kernel's BTF and patches in the right offsets. No clang on the target. No headers needed. Much faster startup. Smaller deploys. Works on any kernel ≥ 5.4 with BTF.

If you are starting fresh, use libbpf with CO-RE.

### 4. "Is eBPF a kernel module?"

No. An eBPF program is bytecode loaded via the `bpf()` syscall. It is not packaged as a `.ko` file. It cannot do unrestricted kernel-mode operations. It runs inside the kernel but inside the BPF VM, not as native kernel code (until the JIT compiles it, but the JIT only compiles bytecode the verifier already accepted).

### 5. "Why does Cilium replace iptables?"

iptables is a chain of rules. To process a packet, the kernel walks the chain in order, evaluating each rule. With thousands of pods (each contributing rules), iptables chains get enormous. Performance falls off a cliff. Adding or removing a rule takes lock time and rebuilds chains.

eBPF programs are different. They are compiled code: the chain of rules is *baked into* the JIT-compiled BPF program. Lookups are O(1) hash-table lookups (BPF maps). Rule changes are map updates, atomic and concurrent. No chain to walk.

So Cilium replaces iptables because eBPF is a fundamentally faster mechanism for the same job. Same goes for kube-proxy, which Cilium also replaces with BPF.

### 6. "What's BTF?"

**BTF** (BPF Type Format) is the kernel's compiled type metadata. It says "struct task_struct has these fields at these offsets." When you load a BPF program, libbpf reads BTF and patches your program to use the right offsets for THIS kernel. That makes your one binary work on many kernels. CO-RE depends on BTF.

Modern distributions enable `CONFIG_DEBUG_INFO_BTF=y`. If your kernel has `/sys/kernel/btf/vmlinux`, you have BTF.

### 7. "What's the difference between kprobe, tracepoint, fentry?"

All three attach to kernel functions or events.

- **kprobe**: attach to (almost) any kernel function. Mechanism uses an interrupt. Higher overhead. Can break across kernel versions when functions change.
- **tracepoint**: attach to a hand-curated stable hook the kernel maintainers added on purpose. Lower overhead than kprobe. Stable across versions, but only the events someone exposed.
- **fentry/fexit**: attach to kernel functions like kprobe, but using BPF trampolines (since 5.5). Faster than kprobe. Most kernel functions are supported.

If you're on 5.5+, prefer fentry over kprobe. If a tracepoint exists for what you want, prefer it over either.

### 8. "Why do I need root?"

Loading BPF programs requires CAP_BPF (or CAP_SYS_ADMIN on older kernels). Reading some maps requires it. Attaching to most hooks requires it. This is by design: BPF programs run in the kernel and could leak information if abused. Most BPF tools want root.

You can grant CAP_BPF to a binary with `setcap cap_bpf,cap_perfmon=ep /usr/bin/foo`. That's how `bpftool` is sometimes installed.

The `unprivileged_bpf_disabled` sysctl controls whether unprivileged users can load some BPF programs. Default is `2` (cannot). Don't change it.

### 9. "Why is XDP faster than tc?"

XDP runs at the driver level, before the kernel allocates an `sk_buff` for the packet. Allocating an `sk_buff` costs CPU. Skipping that allocation is a big win.

XDP also runs before the kernel does any of its layered network processing: no Netfilter hooks, no routing lookup, no socket lookup. It's the closest to "raw packet from the wire" you can be in software.

The tradeoff: XDP gets a `xdp_md` (a thin packet wrapper) instead of an `sk_buff`. You don't have all the convenience structures. For complex policy you might want tc instead, where you have the full sk_buff and can use more helpers.

### 10. "What does `BPF_PROG_TYPE_XDP` mean?"

It's a constant the kernel uses to identify the program type. Each program type has its own context type (what arguments your program gets), its own allowed helpers, its own attachment mechanism. `BPF_PROG_TYPE_XDP` programs get an `xdp_md` argument and return `XDP_DROP/PASS/TX/REDIRECT`. `BPF_PROG_TYPE_KPROBE` programs get a `pt_regs` and return 0. And so on.

In libbpf source you mark them with `SEC("xdp")`, `SEC("kprobe/vfs_read")`, `SEC("tp/syscalls/sys_enter_openat")`. libbpf maps the section name to a program type.

### 11. "Can eBPF programs talk to the network?"

Sort of. They can send packets via XDP_TX or `bpf_redirect`. They can write events to a ring buffer that user-space reads, and user-space can then make network calls. But eBPF programs cannot directly open a socket and call something. They are sandboxed.

### 12. "Why do BPF programs have a 1 million instruction limit?"

Because the verifier walks every possible path, and the time it takes is roughly proportional to the number of instructions times the path complexity. A 1 million instruction limit keeps verification time bounded. If your program is bigger, you split it with tail calls, or you solve the same problem differently.

In older kernels the limit was 4096 instructions. It's been raised over time as the verifier got smarter.

### 13. "What's a tail call again?"

When BPF program A says `bpf_tail_call(ctx, &prog_array, idx)`, the kernel jumps to the program at index `idx` in `prog_array`. A's stack and registers are reset. A does not return; it's replaced by the new program. This lets you build a chain of programs, each one smaller than the verifier's limit. Up to 33 deep.

### 14. "What's a ring buffer vs a perf buffer?"

Both are mechanisms for streaming events from BPF to user-space.

**Perf event array** (`BPF_MAP_TYPE_PERF_EVENT_ARRAY`) is one per-CPU buffer. Older. Has rough edges with ordering across CPUs and is somewhat awkward.

**Ring buffer** (`BPF_MAP_TYPE_RINGBUF`, since 5.8) is a single shared buffer with proper ordering and lower overhead. Use this if your kernel is recent enough.

### 15. "Why does my probe say `arg0` is the wrong thing?"

Because kprobe arguments are CPU registers, and which register holds which argument depends on the architecture's calling convention. On x86_64, `arg0` is RDI, `arg1` is RSI, etc. bpftrace abstracts this. In raw libbpf you cast `(struct pt_regs *)ctx` and use `PT_REGS_PARM1(ctx)` etc.

If you're using fentry instead, the arguments are clean and typed: `int BPF_PROG(my_prog, struct sock *sk, int flag) { ... }`. No registers, no casts. Another reason to use fentry on 5.5+.

## Vocabulary

A long table. If a word in this sheet feels weird, look it up here.

Term | Plain English
---|---
**BPF** | Berkeley Packet Filter. Originally a 1992 packet-filter VM in Linux. Today often used as shorthand for eBPF.
**eBPF** | Extended Berkeley Packet Filter. The 2014+ general-purpose in-kernel VM that this whole sheet is about.
**classic BPF (cBPF)** | The original 1992 BPF. Two registers, ~30 instructions, packet-filter-only. Still works (e.g. for socket filters).
**bytecode** | An intermediate compiled form. eBPF source compiles to bytecode, which the kernel verifies and JIT-compiles.
**opcode** | One instruction's numeric code. Like the 8-bit `opcode` field at the start of every BPF instruction.
**instruction** | One step in a program. eBPF has a fixed 64-bit instruction format.
**map** | A key/value data structure shared between BPF programs and user-space. The "shared whiteboard."
**program** | A compiled BPF object that attaches to a hook and runs on events. A "robot."
**attach point** | The place in the kernel where a BPF program hooks in. kprobe, tracepoint, XDP, tc, etc.
**kprobe** | A probe that fires when a kernel function is called. Can attach to almost any kernel function. Older mechanism.
**kretprobe** | Same as kprobe but fires on function return. Often paired with kprobe to measure latency.
**uprobe** | User-space version of kprobe: fires when a user-space function is called.
**uretprobe** | User-space version of kretprobe.
**tracepoint** | A stable kernel-side trace event. Hand-curated by kernel devs. Stable arguments across kernel versions.
**raw_tracepoint** | A faster, lower-level tracepoint variant that skips some bookkeeping.
**USDT** | User Statically-Defined Tracing. Tracepoints baked into user-space programs.
**fentry** | Fast function entry tracing using BPF trampolines. Since 5.5. Lower overhead than kprobe.
**fexit** | Fast function exit tracing. Pair to fentry.
**fmod_ret** | Fast function return modification: change a kernel function's return value. Powerful, scary.
**perf_event** | A perf-subsystem event. BPF programs can sample on these (CPU profiling, hardware counters, etc.).
**sched_switch** | A tracepoint that fires when the scheduler switches from one process to another. Used in scheduler-latency tools.
**XDP** | eXpress Data Path. The earliest place in the network stack to attach a BPF program. Fastest.
**XDP_DROP** | XDP verdict: drop the packet, kernel never sees it.
**XDP_PASS** | XDP verdict: pass the packet up to the kernel for normal processing.
**XDP_TX** | XDP verdict: bounce the packet back out the same interface.
**XDP_REDIRECT** | XDP verdict: send the packet out a different interface or to a CPU/AF_XDP socket.
**XDP_ABORTED** | XDP verdict: error condition. Drops + emits trace event.
**tc** | Traffic Control. The Linux network-shaping subsystem. BPF programs can attach as classifiers.
**TC_ACT_OK** | TC verdict: let the packet proceed.
**TC_ACT_SHOT** | TC verdict: drop the packet.
**sk_msg** | Program type that attaches to socket message paths. Used for socket-level service mesh.
**sk_skb** | Program type that attaches to socket-receive skbuffs.
**sock_ops** | Program type that attaches to TCP socket events: connect, retransmit, RTT.
**LSM** | Linux Security Module. Permission-check framework in the kernel. BPF can attach since 5.7.
**BPF LSM** | The BPF program type that attaches to LSM hooks. Lets you write security policy in BPF.
**cgroup_skb** | Program type that attaches to cgroup network filters: per-container/per-pod policy.
**cgroup_sock** | Program type that attaches to socket-level cgroup hooks.
**sock_addr** | Program type that hooks address resolution at connect()/bind() time.
**bpf() syscall** | The single Linux system call you use to load programs, manage maps, and attach things.
**BPF_PROG_LOAD** | The bpf() command that loads and verifies a program.
**BPF_MAP_CREATE** | The bpf() command that creates a map.
**BPF_PROG_ATTACH** | The bpf() command that attaches a program to certain hook types (cgroups, etc.).
**libbpf** | The standard C library for working with BPF. The recommended way.
**BPF skeleton** | An auto-generated header file (from `bpftool gen skeleton`) that wraps libbpf for one BPF object.
**libxdp** | A library specifically for XDP programs and AF_XDP sockets.
**bpftool** | The Swiss-army-knife CLI for BPF: load, inspect, attach, dump, generate skeletons.
**bpftrace** | High-level scripting language for BPF. The "awk of the kernel."
**bcc** | BPF Compiler Collection. Older Python framework with hundreds of pre-written tools.
**ply** | An older bpftrace-like tool, less common today.
**helper functions** | Curated kernel functions that BPF programs are allowed to call (`bpf_*`).
**tail call** | One BPF program calling into another via `bpf_tail_call` and a prog_array map. Builds program chains.
**prog_array** | The map type used for tail calls. An array of program file descriptors.
**BPF_PROG_RUN** | Run a BPF program against test input from user-space, mostly for testing.
**JIT** | Just-In-Time compiler. Translates BPF bytecode to native machine code at load time.
**verifier** | The static analyzer that checks BPF programs for safety before they run.
**range analysis** | Verifier technique: track each register's possible value range.
**dead code elimination** | Verifier technique: prove some code is unreachable, mark it for the JIT to skip.
**bounds check** | An explicit check (in code) that a pointer access is within allowed memory.
**BTF** | BPF Type Format. Compact kernel type metadata. Powers CO-RE.
**CO-RE** | Compile Once, Run Everywhere. The technology that lets one BPF binary work on many kernel versions.
**vmlinux.h** | The big auto-generated header with every kernel struct, used by CO-RE programs.
**relocation** | A patch the loader applies to your bytecode. CO-RE relocations fix up field offsets at load time.
**BPF spinlock** | A tiny spinlock embedded in BPF map values. Lets you do atomic compound updates.
**ringbuf** | `BPF_MAP_TYPE_RINGBUF`. Lock-free ring buffer for streaming events to user-space (since 5.8).
**perf buffer** | `BPF_MAP_TYPE_PERF_EVENT_ARRAY`. Older, per-CPU event buffer.
**hash map** | `BPF_MAP_TYPE_HASH`. A general-purpose hash table.
**LRU map** | `BPF_MAP_TYPE_LRU_HASH`. Hash map that evicts least-recently-used entries.
**array map** | `BPF_MAP_TYPE_ARRAY`. Index-keyed fixed-size array.
**percpu hash** | `BPF_MAP_TYPE_PERCPU_HASH`. One hash table per CPU. No locking. Faster.
**percpu array** | `BPF_MAP_TYPE_PERCPU_ARRAY`. One array per CPU. Same idea.
**sock_hash** | `BPF_MAP_TYPE_SOCKHASH`. A hash map of sockets. Used in sk_msg.
**sock_map** | `BPF_MAP_TYPE_SOCKMAP`. An array map of sockets. Used in sk_msg.
**devmap** | `BPF_MAP_TYPE_DEVMAP`. An array of network interfaces, used by XDP redirect.
**cpumap** | `BPF_MAP_TYPE_CPUMAP`. An array of CPUs, used by XDP redirect to fan out work.
**xskmap** | `BPF_MAP_TYPE_XSKMAP`. An array of AF_XDP sockets, used by XDP redirect to user-space.
**lookup** | Read an entry from a map.
**update** | Write an entry to a map.
**delete** | Remove an entry from a map.
**iteration** | Walk every entry in a map. Done from user-space via `bpf()`, or from BPF via the iter program type.
**pinning** | Persisting a program or map to `/sys/fs/bpf/<name>` so it survives the loader exiting.
**auto-pinning** | A libbpf feature that pins everything on load.
**bpf_get_current_pid_tgid** | Helper that returns top 32 bits = TGID (process ID), bottom 32 = PID (thread ID).
**bpf_get_current_uid_gid** | Helper that returns top 32 bits = GID, bottom 32 bits = UID.
**bpf_ktime_get_ns** | Helper that returns nanoseconds since boot. Used for latency measurements.
**bpf_get_smp_processor_id** | Helper that returns the current CPU number.
**bpf_probe_read_kernel** | Helper that safely copies bytes from a kernel pointer to BPF stack.
**bpf_probe_read_user** | Helper that safely copies bytes from a user-space pointer to BPF stack.
**bpf_trace_printk** | Helper that prints to `/sys/kernel/debug/tracing/trace_pipe`. Debug only.
**bpf_perf_event_output** | Helper that emits an event to a perf event array map.
**bpf_ringbuf_output** | Helper that emits an event to a ring buffer (5.8+).
**bpf_redirect** | Helper that redirects a packet to a different interface (XDP/TC).
**bpf_clone_redirect** | Helper that clones a packet and redirects the clone (TC only).
**bpf_skb_load_bytes** | Helper that loads bytes from a skb at an offset (TC).
**verdict** | The return value a BPF program produces that tells the kernel what to do with the event. XDP verdicts, TC verdicts, etc.
**TC_ACT_OK** | TC verdict, "let it pass."
**TC_ACT_SHOT** | TC verdict, "drop it."
**BPF_OK** | Generic OK return value.
**BPF_DROP** | Some program types' "drop" verdict.
**stack** | Each BPF program has 512 bytes of stack memory. No more.
**R0..R10** | The eBPF VM's 11 registers. R0 is return value; R1..R5 are args; R6..R9 callee-saved; R10 is the stack pointer (read-only).
**ALU** | Arithmetic Logic Unit. The instruction class for arithmetic and bitwise ops.
**ALU64** | Same, but for 64-bit operations.
**JMP** | The instruction class for jumps and branches. Both 32 and 64-bit variants.
**LD/LDX/ST/STX** | Memory load/store instruction classes.
**back-edge** | A jump from a later instruction to an earlier one. Used to detect loops in the verifier.
**bounded loop** | A loop with a known small upper bound that the verifier can unroll. Allowed since 5.3.
**bpf_loop** | A helper introduced in 5.13 that lets you write loops the verifier doesn't have to fully unroll.
**JIT spraying** | An attack against JIT compilers. Constant blinding mitigates it.
**constant blinding** | XOR-ing immediate values with a per-program random key in the JIT output.
**retpoline** | A code sequence that prevents Spectre v2 indirect-call branch target injection.
**unprivileged BPF** | Running BPF as a non-root user. Heavily restricted by default and disabled on most distros.
**CAP_BPF** | Linux capability that grants BPF program loading privileges. Introduced in 5.8.
**CAP_PERFMON** | Linux capability for perf events. Often paired with CAP_BPF.
**CAP_SYS_ADMIN** | The big-stick capability. Older kernels required this for BPF.
**iter program** | BPF program that iterates over kernel structures (tasks, files, sockets). Since 5.8.
**struct_ops** | BPF program that implements a kernel interface. Used for custom TCP CC algorithms.
**tnum** | Tristate Number. The verifier's bit-by-bit known/unknown tracking.
**abstract interpretation** | The general technique behind the verifier: track abstract states instead of concrete values.
**range** | A min/max bound on a register's possible values. The verifier tracks both signed and unsigned ranges.
**reference tracking** | The verifier tracks which resources you've acquired (e.g. via `bpf_sk_lookup`) and ensures you release them.

## Try This

Five-to-ten safe experiments. None of these will harm your system.

### 1. List loaded programs

```
bpftool prog list 2>&1 | head
```

What's already running on your box. On a stock Ubuntu/Fedora, you'll see systemd cgroup_skb programs.

### 2. Count file opens for 5 seconds

```
sudo bpftrace -e 'tracepoint:syscalls:sys_enter_openat { @[comm] = count(); }' -c 'sleep 5'
```

The `-c` flag runs `sleep 5` and bpftrace exits when it finishes. You'll see who opened how many files in those 5 seconds.

### 3. See every program launched in a minute

```
sudo bpftrace -e 'tracepoint:syscalls:sys_enter_execve { printf("%-16s %-5d %s\n", comm, pid, str(args->filename)); }' -c 'sleep 60'
```

Open a few terminals, run a few commands. You'll see them appear here.

### 4. Histogram of read sizes during a `dd`

In one terminal:
```
dd if=/dev/zero of=/tmp/test bs=4k count=1000
```

In another:
```
sudo bpftrace -e 'tracepoint:syscalls:sys_exit_read /args->ret > 0/ { @bytes = hist(args->ret); }' -c 'sleep 5'
```

You'll see the read-size distribution, with a fat bin at 4096 (4KB) for the dd.

### 5. Profile your CPU

```
sudo bpftrace -e 'profile:hz:99 { @[kstack] = count(); }' -c 'sleep 10' | head -50
```

Profiles all CPUs at 99 Hz for 10 seconds. Counts the kernel stacks. The most-frequent stack is what your kernel was busiest doing.

### 6. Watch DNS lookups

```
sudo bpftrace -e 'kprobe:udp_sendmsg { printf("%s sending UDP\n", comm); }' -c 'sleep 30'
```

Browse the web; you'll see DNS resolvers (`systemd-resolved`, `dnsmasq`, etc.) sending UDP.

### 7. See your shell's syscalls

```
strace -c -p $$ &
sleep 5
fg
```

Hit a few keys. Strace will show a count of syscalls. Then try the BPF version:

```
sudo bpftrace -e 'tracepoint:raw_syscalls:sys_enter /pid == '$$'/ { @[args->id] = count(); }' -c 'sleep 5'
```

You're filtering only your own shell's syscalls and counting them by syscall number.

### 8. List supported features

```
bpftool feature probe | grep -i "is available" | head -30
```

Tells you what your kernel can do with BPF.

### 9. See your kernel's BPF version

```
uname -r
```

Anything ≥ 5.4 has BTF (with the right config). Anything ≥ 5.8 has CAP_BPF, ringbuf. Anything ≥ 5.13 has bounded loops via bpf_loop.

### 10. Inspect a running BPF map

```
sudo bpftool map list
sudo bpftool map dump id <ID>
```

Pick an ID from the list. Look at what's in there. (Most system-loaded maps will be empty or full of cgroup info.)

These experiments are all read-only or short-lived. Nothing here changes any persistent state.

## Where to Go Next

- `cs fundamentals ebpf-bytecode` — the bytecode reference: every opcode, the instruction format, register conventions.
- `cs detail fundamentals/ebpf-bytecode` — the academic underpinning: verifier proofs, JIT correctness, formal verification.
- `cs performance ebpf` — applied performance tracing with BPF, including the bcc tools tour.
- `cs performance bpftrace` — the scripting frontend reference.
- `cs performance bpftool` — the inspection/loading tool.
- `cs offensive ebpf-security` — eBPF as a security tool (and as an attacker primitive).
- `cs service-mesh cilium` — Cilium, the BPF-native service mesh.
- `cs ramp-up linux-kernel-eli5` — what the kernel IS, in plain English.

## Version Notes

Quick timeline of when each major eBPF feature landed:

- **3.18 (Dec 2014)**: eBPF arrives in mainline. First program type (socket filters, like cBPF) and first map type (hash, array).
- **4.1 (Jun 2015)**: kprobe program type. Now you can probe any kernel function.
- **4.4 (Jan 2016)**: BPF persists across syscall boundaries (object pinning). XDP comes a few releases later.
- **4.8 (Oct 2016)**: XDP arrives.
- **4.10 (Feb 2017)**: cgroup-skb programs.
- **4.14 (Nov 2017)**: BPF tools: kprobe with BPF programs, perf-event programs, sock_ops, tail calls.
- **4.18 (Aug 2018)**: BTF lands in the kernel.
- **5.2 (Jul 2019)**: Pinning maps in subdirs, more verifier improvements.
- **5.3 (Sep 2019)**: Bounded loops (you can write `for (i = 0; i < N; i++)` for compile-time-constant N and the verifier will unroll).
- **5.4 (Nov 2019)**: BTF + CO-RE go mainstream. This is the real "modern eBPF starts here" moment.
- **5.5 (Jan 2020)**: fentry, fexit, fmod_ret using BPF trampolines.
- **5.7 (May 2020)**: BPF LSM. You can write security policy in BPF.
- **5.8 (Aug 2020)**: BPF ring buffer. CAP_BPF capability. iter program type.
- **5.10 (Dec 2020)**: Static linking of multiple BPF objects.
- **5.11 (Feb 2021)**: Atomic operations on BPF maps.
- **5.13 (Jun 2021)**: `bpf_loop()` helper. Spectre verifier hardening.
- **5.15 (Nov 2021)**: BPF user-space queue/stack maps.
- **5.16 (Jan 2022)**: Compile-once-run-everywhere modules: BPF can include from kernel modules' BTF.
- **5.18 (May 2022)**: Generic kfuncs.
- **6.0–6.6 (2022–2023)**: Continuous polish: more program types, better verifier precision, kfunc explosion.

If your kernel is ≥ 5.4 with BTF you have the modern eBPF experience. ≥ 5.8 unlocks ringbuf and is the modern baseline. ≥ 5.13 unlocks `bpf_loop`. Most production Linux today (RHEL 9, Ubuntu 22.04+, Debian 12+) is ≥ 5.10 and has all the goodies.

## See Also

- `fundamentals/ebpf-bytecode`
- `performance/ebpf`
- `performance/bpftrace`
- `performance/bpftool`
- `offensive/ebpf-security`
- `service-mesh/cilium`
- `system/strace`
- `ramp-up/linux-kernel-eli5`
- `ramp-up/http3-quic-eli5`

## References

- ebpf.io — the canonical site, with curated docs, project list, talks, papers.
- "BPF Performance Tools" by Brendan Gregg (Addison-Wesley, 2019). The reference book.
- "Linux Observability with BPF" by David Calavera and Lorenzo Fontana (O'Reilly, 2019). A practitioner's intro.
- `man bpf` — the syscall man page. Concise and authoritative.
- `man bpf-helpers` — every helper function documented.
- `man bpftool` — the tool's reference.
- kernel.org/doc/html/latest/bpf/ — the kernel's official BPF documentation tree.
- iovisor/bcc — the BCC toolkit on GitHub. Hundreds of pre-written tools, plus the Python framework.
- libbpf/libbpf-bootstrap — modern starter projects using libbpf and CO-RE.
- aquasecurity/tracee — security-focused BPF tooling.
- cilium/ebpf — the Go library. Pure Go, no cgo.
- iovisor/bpftrace — the bpftrace project on GitHub.
- aya-rs/aya — the Rust library.
- Brendan Gregg's BPF page (brendangregg.com/ebpf.html) — talks, slides, flame graphs, war stories.
- "The eBPF Verifier" by the Linux kernel docs — the deep dive on how the verifier works.
- Kernel source: `kernel/bpf/`, `tools/lib/bpf/`, `tools/bpf/bpftool/`, `samples/bpf/`. If you're the type who reads kernel source, this is where to start.

"The tiny robots carry the Pattern." — The Whispering Void
