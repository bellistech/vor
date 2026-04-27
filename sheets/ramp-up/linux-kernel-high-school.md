# Linux Kernel — High School (Part 3 of 4 — Ramp-Up Curriculum)

> Hardware-enforced privilege rings, the syscall instruction, signals, namespaces, cgroups, and eBPF — the *mechanisms* that let one Linux kernel host thousands of mutually-suspicious workloads.

## Prerequisites

- `cs ramp-up linux-kernel-middle-school` — the prior tier (boot, fork/exec, page tables, CFS, VFS, network stack)
- Comfort reading `/proc/*` and basic `bash` plumbing
- A working notion of "user space" vs "kernel space" — this sheet explains how the CPU enforces it
- Optional: `cs ramp-up linux-kernel-elementary` if you want to refresh the picture-book level
- A Linux box with root (or `sudo`) for the `unshare`, `bpftrace`, and cgroup experiments
- The Middle School tier's vocabulary (process, thread, syscall, fd, inode, mount, packet) — this sheet builds on it

---

## Plain English

In Middle School we learned the kernel is the program that owns the hardware. Every process politely asks the kernel for everything: "open this file", "send these bytes", "give me more memory". The unspoken question was: **what stops a process from just *not* asking?** What stops a malicious program from poking memory it doesn't own, talking to the disk controller directly, or rewriting the page tables to hand itself the kernel's keys?

The answer is not software. The answer is the CPU itself.

Every modern processor — x86, ARM, RISC-V, POWER — has at least two **privilege levels** burned into the silicon. On x86 these are called **rings**, numbered 0 through 3. Ring 0 is "kernel mode": every instruction is legal, including `lgdt` (load the global descriptor table), `mov cr3, rax` (swap address spaces), `wbinvd` (write back and invalidate the cache), and `hlt` (halt the CPU). Ring 3 is "user mode": the same instructions exist as opcodes, but the moment user code tries to execute one, the CPU itself raises a **General Protection Fault** before the instruction completes. The kernel catches that fault, looks at the offender, and usually sends `SIGSEGV` — segmentation fault, the most famous error in computing. The misbehaving process dies. The kernel survives. The hardware — not the kernel, not the compiler, not the linker — drew the line.

ARM uses the same idea with different names. ARM exception levels go from EL0 (user) up through EL1 (kernel), EL2 (hypervisor), and EL3 (secure monitor). x86 has equivalents: Ring 0 for the kernel, Ring -1 (also called "VMX root mode" on Intel or "SVM" on AMD) for the hypervisor, and SMM (System Management Mode) for firmware. Rings 1 and 2 exist on x86 but Linux ignores them — the cost of switching between four rings turned out to exceed the benefit, and OS designers consolidated to two. The takeaway: the CPU has a register that says "current privilege level", and every memory access and every instruction is checked against it. There is no software trick that gets around this. You change your privilege level the same way everyone else does — by *asking the CPU to switch you*, through a controlled doorway.

The doorway is called an **interrupt** or, when the user program initiates it on purpose, a **trap** or **system call**. When you press a key, the keyboard controller raises a wire on the bus that says "I have data". A chip called the **APIC** (Advanced Programmable Interrupt Controller) routes that wire to a CPU core and forces it to stop whatever it was doing — even if it was halfway through a multiplication. The CPU saves its current registers, looks up entry number 1 (the keyboard's IRQ vector) in a table called the **IDT** (Interrupt Descriptor Table), and jumps to the address it finds there. That address is a kernel function. The kernel handles the keystroke, restores the saved registers, and the user program resumes — *with no idea anything happened*. This pattern — hardware event, IDT lookup, kernel handler, return — is how literally every external thing reaches the kernel: keystrokes, mouse clicks, network packets, disk completions, timer ticks, doorbell registers from GPUs.

System calls work the same way, except the user program raises the interrupt on purpose. When you write `read(fd, buf, n)` in C, the libc implementation puts the syscall number in a register, puts your arguments in more registers, and executes a single instruction — historically `int 0x80` on i386, today `syscall` on x86_64 or `svc #0` on arm64. That instruction switches the CPU from Ring 3 to Ring 0, looks up the syscall handler in a table, and runs the kernel code. When the kernel is done it executes `sysret` (or `eret` on ARM) and you're back in Ring 3 with the result. Cost: usually a few hundred CPU cycles per crossing. Linux has roughly 350 syscalls; the *list* — the **syscall table** — is one of the kernel's most important interfaces, because it never breaks userspace.

Now here's where it gets interesting for modern Linux. You have a single kernel running on one machine. You want to run two web servers, each thinking it owns ports 80 and 443. Each thinks it has PID 1. Each has its own filesystem layout. Each is limited to 2 GB of RAM and 1 CPU. Old answer: virtual machines — slow, fat, each VM has a whole guest kernel inside. New answer: **containers** — and a container is not a magic technology, it is a **bundle of kernel features** glued together. The two big ingredients are **namespaces** (which give each container its own view of system-wide resources like PIDs, network interfaces, and mount points) and **cgroups** (which limit how much CPU, memory, and I/O each container can use). Add **capabilities** (fine-grained slices of root), **seccomp** (a filter that says "this container is only allowed these 50 syscalls out of 350"), and **LSMs** like SELinux or AppArmor (mandatory access control), and you have what Docker, Kubernetes, systemd-nspawn, LXC, and Podman all build on. Same kernel. Many isolated boxes.

Last big piece: **eBPF**. Imagine you want to ask a deep question of a running production kernel, like "for every TCP connection that closes, how long did it last?" Old answer: write a kernel module — a shared library loaded into Ring 0 with a footgun pointed at the entire system; one bug crashes the box. New answer: write a tiny program in a restricted instruction set, ship it through the `bpf()` syscall, and the kernel's **verifier** runs a static analysis to *prove* your program halts, doesn't crash, doesn't read invalid memory, and respects the right capabilities. If the verifier accepts, a **JIT** compiles your bytecode to native machine code and attaches it to a hook — a kprobe at any kernel function, a tracepoint at a defined event, an XDP hook at the network driver, an LSM hook at a security check. Your code runs in kernel context with kernel speed but cannot crash the kernel. This is the most important kernel feature of the last decade. It powers `bpftrace`, Cilium's networking, every modern profiler, runtime security tools like Falco and Tetragon, and the load balancers at hyperscalers. eBPF turns the kernel into something you can program safely from outside.

The tour for this sheet: rings → interrupts → IDT → IRQ path → top-half/bottom-half → syscalls (`int 0x80`, `syscall`, `svc`) → vDSO → signals → cgroups (v1 and v2) → namespaces (all eight of them) → capabilities → seccomp → LSMs → eBPF. By the end you should be able to read `/proc/interrupts`, `/proc/$$/ns/*`, `/sys/fs/cgroup/`, and `bpftool prog list` and understand what every column means. You should never need to leave the terminal to ask "what is a namespace" again.

---

## Concepts in Detail

### CPU Privilege Rings — Hardware Enforcement

```
                  x86 Privilege Rings
              ┌─────────────────────────┐
              │      Ring -1 (VMX)      │  Hypervisor (KVM, VMware, Hyper-V)
              │  ┌───────────────────┐  │  also called "VMX root mode"
              │  │     Ring 0        │  │  Linux kernel — full hardware access
              │  │ ┌───────────────┐ │  │
              │  │ │   Ring 1      │ │  │  unused on Linux (originally drivers)
              │  │ │ ┌───────────┐ │ │  │
              │  │ │ │  Ring 2   │ │ │  │  unused on Linux (originally drivers)
              │  │ │ │ ┌───────┐ │ │ │  │
              │  │ │ │ │Ring 3 │ │ │ │  │  user space — bash, browsers, libc
              │  │ │ │ └───────┘ │ │ │  │
              │  │ │ └───────────┘ │ │  │
              │  │ └───────────────┘ │  │
              │  └───────────────────┘  │
              └─────────────────────────┘

                  ARM Exception Levels
              ┌─────────────────────────┐
              │   EL3  Secure monitor   │  ATF — secure boot, TrustZone gateway
              │   EL2  Hypervisor       │  KVM-arm, Xen
              │   EL1  Kernel           │  Linux
              │   EL0  User             │  apps
              └─────────────────────────┘
```

The CPU stores the **Current Privilege Level (CPL)** in the bottom two bits of the CS (code segment) register on x86. Every instruction fetch, every memory access, every IO port read is gated by this number. Privileged instructions — `lgdt`, `lidt`, `mov cr3`, `wbinvd`, `hlt`, `in`, `out`, `rdmsr`, `wrmsr` — fault when CPL > 0. Memory pages are tagged with a "user/supervisor" bit in the page table entry; user-mode code that touches a supervisor page page-faults instantly.

Why two rings, not four? In the early 1990s, OS designers measured: switching between Ring 1 and Ring 0 had nearly the same TLB-flush cost as Ring 3 → Ring 0. The "extra security" of putting drivers in Ring 1 wasn't worth it; you'd still need a switch on every device interaction. Linus picked the simple model: privileged or not. Microkernels (Mach, L4) used more rings; Linux's monolithic-with-modules model just uses two.

**Hypervisor ring (Ring -1):** Intel's VT-x (and AMD's SVM) added a *new* privilege level *below* Ring 0. A hypervisor like KVM runs in VMX root; the guest kernel still thinks it's in Ring 0 but is actually in VMX non-root. Privileged instructions inside the guest trap to the hypervisor instead of the host kernel. This is what makes hardware virtualization fast — no instruction-by-instruction emulation.

```
$ cat /proc/cpuinfo | grep -o '\(vmx\|svm\)' | head -1
vmx                       # Intel — supports VT-x (Ring -1)
# or 'svm' on AMD
```

**Version note:** Intel SGX added "enclaves" (Ring 3 with extra encryption) in 2015. AMD SEV/SEV-SNP added encrypted guest memory in 2017 / 2020. Both are orthogonal to the basic ring model.

---

### Interrupts — Hardware vs Software vs Exceptions

Every reason the CPU stops what it's doing falls into one of three categories:

| Category   | Triggered by              | Examples                              | Synchronous?       |
|------------|---------------------------|---------------------------------------|--------------------|
| Interrupt  | External hardware         | NIC packet, disk I/O done, timer tick | Asynchronous       |
| Trap       | A specific instruction    | `syscall`, `int 0x80`, `int3` (debug) | Synchronous        |
| Exception  | A failing instruction     | `#PF` page fault, `#GP`, `#DE` divide | Synchronous        |

All three go through the IDT. The CPU doesn't really care which category — it just looks up the vector number in the IDT and jumps. The kernel sorts them out from there.

```
                Interrupt Vector Numbers (x86_64)
            ─────────────────────────────────────
            0     #DE   Divide error
            1     #DB   Debug
            2     NMI   Non-maskable interrupt (parity, watchdog)
            3     #BP   Breakpoint (int3)
            6     #UD   Undefined opcode
            8     #DF   Double fault
            13    #GP   General protection
            14    #PF   Page fault
            ─────────────────────────────────────  ← CPU exceptions
            32-47       Legacy IRQs (PIC era)
            48-127      Device IRQs (APIC era)
            128   0x80  i386 syscall trap (legacy)
            ─────────────────────────────────────
```

**Maskable vs non-maskable:** the CPU has an "interrupts enabled" flag (IF on x86, the I bit in CPSR/PSTATE on ARM). The kernel can clear it with `cli` to enter a section that can't be interrupted. **NMI** (non-maskable interrupt, vector 2) ignores this flag — used for hardware errors like ECC failures and the watchdog timer. You cannot block an NMI; if your code is running with interrupts disabled and an NMI arrives, you handle it now.

**IPI (Inter-Processor Interrupt):** one CPU pokes another. Linux uses IPIs for TLB shootdowns ("hey CPU 3, your TLB has stale entries, flush it"), reschedule signals, and `smp_call_function()`.

---

### IDT and APIC — The Plumbing

```
                    Interrupt Descriptor Table (IDT)
   Vector           Pointer to handler          Type / DPL
   ─────  ─────────────────────────────  ──────────────────
   0      &divide_error                  Trap, DPL=0
   1      &debug                         Interrupt, DPL=0
   ...
   14     &page_fault                    Interrupt, DPL=0
   ...
   33     &handle_keyboard_irq           Interrupt, DPL=0
   ...
   128    &entry_INT80_compat            Trap, DPL=3   ← user can call!
   ...
   256 entries total, each 16 bytes wide on x86_64
```

Every CPU has a register called **IDTR** (Interrupt Descriptor Table Register) that holds the *base address* and *limit* (size) of the IDT. On boot the kernel builds the IDT in RAM and tells the CPU "here's where it lives" with `lidt`. After that, every interrupt goes through this table.

The **DPL** (Descriptor Privilege Level) field is what makes `int 0x80` legal from user space but `int 14` (page fault) illegal. Vectors with DPL=3 can be invoked by Ring 3; DPL=0 vectors can only fire as a result of hardware events or kernel-issued traps.

**APIC (Advanced PIC):** the chip that routes IRQs. There are two pieces:

- **Local APIC (LAPIC):** one per CPU core, on-die. Handles timer ticks, IPIs, performance-counter overflow.
- **I/O APIC (IOAPIC):** one per system, on the chipset. Routes external device IRQs to the right LAPIC.

```
   ┌─────────┐                       ┌─────────────────┐
   │ Disk    ├──irq line──┐          │ CPU0 ┌───────┐  │
   ├─────────┤            │          │      │ LAPIC │  │
   │ NIC     ├──irq line──┤          │      └───┬───┘  │
   ├─────────┤            │          └──────────┼──────┘
   │ USB     ├──irq line──┤                     │
   ├─────────┤            ▼                     │
   │ ...     │       ┌─────────┐                │
   └─────────┘       │ I/O APIC├────msg──┬──────┘
                     └─────────┘         │
                                         ▼
                                  ┌─────────────────┐
                                  │ CPU1 ┌───────┐  │
                                  │      │ LAPIC │  │
                                  │      └───────┘  │
                                  └─────────────────┘
```

**MSI (Message Signalled Interrupts) / MSI-X:** rather than dedicating a wire, modern PCIe devices send a memory write to a magic address that the LAPIC interprets as an interrupt. MSI-X allows up to 2048 vectors per device — vital for multiqueue NICs where each receive queue gets its own IRQ pinned to its own CPU.

**x2APIC:** an extension that allows >256 CPUs by widening LAPIC IDs to 32 bits and using MSRs instead of memory-mapped I/O. Required for cloud workloads with 100+ vCPUs.

---

### The Full Interrupt Path

```
   1. Device asserts IRQ
        │
        ▼
   2. APIC routes to a CPU based on affinity (/proc/irq/N/smp_affinity)
        │
        ▼
   3. CPU finishes current instruction
        │
        ▼
   4. CPU reads vector number from APIC, indexes IDT[vector]
        │
        ▼
   5. CPU pushes RIP, CS, RFLAGS, RSP, SS onto kernel stack
      (switches stack to TSS.RSP0 if coming from Ring 3)
        │
        ▼
   6. CPU clears IF (further interrupts of same type masked)
        │
        ▼
   7. Jumps to handler address — the "top half"
        │
        ▼
   8. Top half: read just enough from device to clear the line,
      copy bytes from NIC ring to skb, ack the IRQ, queue work
        │
        ▼
   9. Top half schedules a softirq / tasklet / workqueue / kthread
        │
        ▼
  10. Top half does iret (or sysret) — interrupts re-enabled
        │
        ▼
  11. Bottom half runs later (with IRQs enabled): protocol stack,
      filesystem completion, scheduler tick consequences, etc.
```

The "5 cycles of registers pushed" is a real thing — see Intel SDM Vol 3A "Interrupt 6.12.1". The TSS (Task State Segment) stack switch is why a misbehaving user stack can't take down the kernel: even if RSP is garbage on entry, the CPU loads a known-good RSP from TSS before pushing anything.

---

### Top Half / Bottom Half — Deferred Work

The top half runs with that IRQ disabled (and on most paths with all maskables disabled). It must be **fast** — every microsecond is a microsecond your network card stops draining its ring buffer and starts dropping packets. So the top half does as little as possible and hands off the rest.

| Mechanism      | Context                | Sleep allowed? | Use case                           |
|----------------|------------------------|----------------|------------------------------------|
| Softirq        | Top of stack, IRQ-on   | No             | Networking RX/TX, block I/O end    |
| Tasklet        | Built on softirq       | No             | Per-device deferred work           |
| Workqueue      | Kernel thread          | Yes            | Anything that may block            |
| Threaded IRQ   | Per-IRQ kthread        | Yes            | Real-time kernels, complex devs    |

```
$ cat /proc/softirqs | head
                    CPU0       CPU1       CPU2       CPU3
          HI:          0          0          0          0
       TIMER:    1456789    1455321    1452108    1451229
      NET_TX:        123        119        102         98
      NET_RX:    9876543     325109     325107     325002
       BLOCK:      53210      52109      51203      51098
    IRQ_POLL:          0          0          0          0
     TASKLET:     12345      12211      12103      12001
       SCHED:    1456000    1455100    1452000    1451000
     HRTIMER:          0          0          0          0
         RCU:    9999999    9999100    9998200    9997300
```

NET_RX softirq is where the network stack actually parses incoming packets — IP, TCP, UDP all run there, after the NIC top-half has just copied bytes off the wire.

---

### System Calls — User Space Asks the Kernel for Help

#### Old style: `int 0x80` (i386 / Linux 0.0.1 → forever)

```
   mov $4, %eax       ; syscall number 4 = sys_write (i386 numbers!)
   mov $1, %ebx       ; arg1 = fd 1 (stdout)
   mov $msg, %ecx     ; arg2 = pointer
   mov $13, %edx      ; arg3 = length
   int $0x80          ; trap into kernel (vector 128)
   ; eax now holds the return value
```

Cost: ~600 cycles on a Pentium 4 (sad). `int 0x80` on a 64-bit process still works for backward compat — it dispatches through `entry_INT80_compat` and uses the *32-bit* syscall table. Different numbers! `sys_write` is `4` on i386 but `1` on x86_64. This is a famous footgun.

#### Modern style: `syscall` (x86_64, since Pentium 4 / Athlon 64)

```
   ; What `write(1, "hi\n", 3)` actually emits:
   mov $1, %rax        ; syscall number 1 = sys_write (x86_64 numbers)
   mov $1, %rdi        ; arg1 = fd
   lea msg(%rip), %rsi ; arg2 = pointer
   mov $3, %rdx        ; arg3 = length
   syscall              ; trap; cost ~150 cycles
   ; rax = return value, or -errno on failure
```

What the CPU does on `syscall`:

1. Save RIP → RCX, RFLAGS → R11 (so the kernel knows where to return).
2. Load CS:RIP from the **LSTAR MSR** (Long-mode System Target Address Register) and STAR MSR.
3. Mask interrupts using the SF_MASK MSR.
4. Switch to Ring 0.

Argument registers on x86_64: `rdi, rsi, rdx, r10, r8, r9` (note `r10` not `rcx` — `rcx` got clobbered by the saved RIP). Return is in `rax`. Errors are returned as negative values; libc converts `-EBADF` etc. into `-1` and sets `errno`.

#### ARM64: `svc #0`

```
   mov  x8, #64        ; syscall number 64 = sys_write (arm64 numbers!)
   mov  x0, #1         ; arg1 = fd
   adr  x1, msg        ; arg2 = pointer
   mov  x2, #3         ; arg3 = length
   svc  #0             ; trap
   ; x0 = return value
```

Argument registers: `x0..x5`. Syscall number in `x8`. ARM64 has its own syscall numbering (yet again different — `read` is 63, `write` is 64).

#### Per-architecture syscall numbers

```
$ grep -E '^#define __NR_(read|write|openat) ' \
    /usr/include/asm-generic/unistd.h
#define __NR_openat 56
#define __NR_read 63
#define __NR_write 64
# (these are arm64; x86_64 are 0/1/257)
```

There is no portable syscall number. `open` is `2` on x86_64, doesn't exist on arm64 (which only has `openat`). Always use libc unless you're writing portable assembly.

#### The vDSO — syscalls without context switches

`gettimeofday()` and `clock_gettime()` are called *millions* of times per second by some workloads (database commit timestamps, request latency histograms). Even at 150 cycles per syscall, that's a non-trivial budget.

Solution: the kernel maps a small **shared object** into every process called the **vDSO** (Virtual Dynamic Shared Object). The kernel updates the time once per tick into a memory page; user space reads that page directly. No context switch, no Ring transition. Reads in ~20 cycles.

```
$ ldd /bin/ls | grep vdso
        linux-vdso.so.1 (0x00007ffe5c3fa000)

$ cat /proc/self/maps | grep vdso
7ffe5c3fa000-7ffe5c3fc000 r-xp 00000000 00:00 0   [vdso]
```

The vDSO is also where signal trampolines live (the code that runs `rt_sigreturn` after your signal handler returns).

#### The syscall table

Located in `arch/x86/entry/syscalls/syscall_64.tbl` in the kernel source. Every entry has a number, an ABI tag (common, 64, x32), a name, and the C handler.

```
$ ausyscall --dump | head -10
Using x86_64 syscall table:
0       read
1       write
2       open
3       close
4       stat
5       fstat
6       lstat
7       poll
8       lseek
9       mmap
```

**openat vs open:** `open(path)` resolves `path` from the process's CWD. `openat(dirfd, path)` resolves it from `dirfd` if `path` is relative. The "at" family (`openat`, `unlinkat`, `mkdirat`, `linkat`) is essential for race-free directory traversal — you don't want a symlink to swap underneath you. Modern glibc routes `open()` through `openat(AT_FDCWD, ...)`.

---

### Signals — The Kernel's Notification System

Signals are software interrupts to processes. They predate threads, predate sockets, predate everything POSIX-like. They are *the* original IPC.

```
                 Signal delivery path
   sender                              receiver
   ──────                              ────────
   kill(2) ────► kernel checks perms ─► sets bit in
   tgkill(2)     of sender vs target    target's
   raise(3)                             pending mask
                                            │
                                            ▼
                                    on next return-
                                    to-user-mode the
                                    kernel checks the
                                    pending mask, picks
                                    a deliverable signal,
                                    sets up handler
                                    frame on user stack,
                                    redirects RIP →
                                    handler. When
                                    handler returns,
                                    sigreturn(2) syscall
                                    restores prior state.
```

| Signal     | Default       | Catchable?        | Notes                                |
|------------|---------------|-------------------|--------------------------------------|
| SIGTERM    | terminate     | yes               | Polite "please exit"; default `kill` |
| SIGINT     | terminate     | yes               | Ctrl+C                               |
| SIGQUIT    | core dump     | yes               | Ctrl+\\                              |
| SIGKILL    | terminate     | **NO**            | Cannot be caught/ignored/blocked     |
| SIGSTOP    | pause         | **NO**            | Cannot be caught/ignored/blocked     |
| SIGCONT    | continue      | yes               | Wake from STOP                       |
| SIGCHLD    | ignore        | yes               | Child changed state                  |
| SIGSEGV    | core dump     | yes               | Bad memory access                    |
| SIGBUS     | core dump     | yes               | Misaligned access; mmap past EOF     |
| SIGFPE     | core dump     | yes               | Divide by zero / FP exception        |
| SIGPIPE    | terminate     | yes               | Wrote to closed pipe / socket        |
| SIGUSR1/2  | terminate     | yes               | App-defined                          |
| SIGHUP     | terminate     | yes               | Terminal closed; reload-config idiom |
| SIGWINCH   | ignore        | yes               | Terminal resized                     |
| SIGALRM    | terminate     | yes               | `alarm(2)` timer                     |

**SIGKILL and SIGSTOP cannot be caught.** Period. You can't write a handler for them. You can't block them. The kernel delivers them by setting the task state and skipping userland entirely. This is why `kill -9` is "the big hammer" — the target has no recourse.

**Real-time signals (32–64):** queued, not coalesced. Plain signals (1–31) collapse: if you `kill -USR1 $$` ten times before the handler runs once, the second through tenth are lost.

**Signal handler flags:**
- `SA_RESTART`: when a syscall is interrupted by a signal, retry it automatically. Without this, `read()` returns `-1` with `errno=EINTR` and you have to handle it everywhere.
- `SA_SIGINFO`: handler gets `siginfo_t` with extra info (sender PID, fault address for SIGSEGV).
- `SA_NOCLDWAIT`: don't create zombies for children — kernel auto-reaps them.
- `SA_RESETHAND`: handler self-disables after first invocation.

```c
struct sigaction sa = {
    .sa_handler = my_handler,
    .sa_flags   = SA_RESTART,
};
sigemptyset(&sa.sa_mask);
sigaction(SIGUSR1, &sa, NULL);
```

**Async-signal-safe functions:** the world inside a handler is hostile. Most of libc is *not* signal-safe — `malloc`, `printf`, `fprintf` may deadlock if interrupted mid-mutex. The official safe list is in `man 7 signal-safety`: `write`, `read`, `_exit`, `signalfd`, `kill`, a few dozen others. Modern code: use `signalfd(2)` to convert signals into readable file descriptors, sidestepping handlers entirely.

**SIGCHLD and zombies:** when a child dies, the kernel keeps a stub task_struct (the "zombie" — `Z` state in `ps`) until the parent calls `wait()` or `waitpid()`. SIGCHLD is delivered. If the parent doesn't reap, you accumulate zombies. If the parent dies first, init (PID 1) inherits the children and reaps them.

**ERESTARTSYS, ERESTARTNOINTR, ERESTART_RESTARTBLOCK:** internal kernel return values that say "this syscall was interrupted; if SA_RESTART is set, restart it automatically; otherwise return EINTR to userspace". You'll see them in kernel traces, not in userspace.

---

### cgroups — Control Groups for Resource Limits

cgroups answer the question: *how do I bound how much CPU, memory, disk I/O, or PIDs a group of processes can use?*

A cgroup is a directory in a special filesystem mounted at `/sys/fs/cgroup`. Move a PID into the directory and the cgroup's limits apply.

#### v1 vs v2

**v1 (legacy, kernel 2.6.24, 2008):** every controller (cpu, memory, blkio, ...) had its own hierarchy, each mounted separately. A process could be in different cgroups for different controllers. Configuring it was a maze.

**v2 (kernel 4.5, 2016, default in modern systemd):** **unified hierarchy** — one tree, all controllers attached. A process is in exactly one cgroup. Subtree control is explicit: a parent enables which controllers its children may use via `cgroup.subtree_control`.

```
              cgroup v2 unified hierarchy
              ─────────────────────────────
              /sys/fs/cgroup/                            (root)
                ├── system.slice/                        (systemd services)
                │     ├── nginx.service/
                │     ├── docker.service/
                │     │     └── docker/<container-id>/
                │     └── postgresql.service/
                ├── user.slice/                          (logged-in users)
                │     └── user-1000.slice/
                │           └── user@1000.service/
                │                 └── app.slice/
                └── machine.slice/                       (VMs, containers)
                      └── libpod-<id>.scope/
```

#### v2 controllers (what you can limit)

| Controller | What it limits / accounts                                     |
|------------|---------------------------------------------------------------|
| cpu        | weights (cpu.weight), bandwidth quotas (cpu.max), idle        |
| memory     | memory.max (hard cap), memory.high (soft), swap, OOM behavior |
| io         | io.max (per-device IOPS / bytes), io.weight                   |
| pids       | pids.max — kills `fork()` once cap hit                        |
| cpuset     | cpus / mems — bind to specific CPUs and NUMA nodes            |
| hugetlb    | hugepage allocations per page size                            |
| rdma       | RDMA hca handles / objects                                    |
| misc       | per-class miscellaneous resources (e.g. SEV ASIDs)            |

Read what's available:

```
$ cat /sys/fs/cgroup/cgroup.controllers
cpuset cpu io memory hugetlb pids rdma misc
```

To enable cpu and memory in a subtree:

```
# echo "+cpu +memory" > /sys/fs/cgroup/system.slice/cgroup.subtree_control
```

Add a process and set a cap:

```
# mkdir /sys/fs/cgroup/work
# echo "+memory +cpu" > /sys/fs/cgroup/cgroup.subtree_control
# echo $$ > /sys/fs/cgroup/work/cgroup.procs
# echo "100M" > /sys/fs/cgroup/work/memory.max
# echo "5000 100000" > /sys/fs/cgroup/work/cpu.max     # 5% CPU
```

`cpu.max` is `quota period` in microseconds — `5000 100000` means "5000us out of every 100000us = 5%". `memory.max` accepts suffixes `K M G`; `max` means unlimited.

#### OOM behavior

When a process inside a memory-limited cgroup tries to allocate beyond `memory.max`, one of two things happens:
- If `memory.oom.group=1` and an OOM occurs: the kernel kills *every task in the cgroup* atomically (containers).
- Otherwise: the kernel picks the heaviest task in the cgroup and kills it (oom_score).

`memory.events` is a counter file showing OOM activity:

```
$ cat /sys/fs/cgroup/system.slice/something.service/memory.events
low 0
high 23
max 5
oom 1
oom_kill 1
```

#### Delegation

A privileged process can hand a cgroup subtree to an unprivileged user by `chown`-ing it. systemd does this — your `user@1000.service` gets a subtree you can carve up further without root.

---

### Namespaces — Each Container's Own View of the World

A namespace virtualizes a global resource. Inside the namespace, that resource looks like the whole thing. Outside, it's just one slice.

There are eight namespaces in modern Linux:

| Namespace | Virtualizes                  | Since   | Kernel docs                |
|-----------|------------------------------|---------|----------------------------|
| MNT       | Mount points                 | 2.4.19  | First namespace, 2002      |
| UTS       | Hostname, domainname         | 2.6.19  | `uname()` lies inside      |
| IPC       | SysV IPC, POSIX msg queues   | 2.6.19  |                            |
| PID       | Process IDs                  | 2.6.24  | Inside, your init is PID 1 |
| NET       | Net devices, IPs, ports, fws | 2.6.29  | Each NS has own loopback   |
| USER      | UIDs/GIDs, capabilities      | 3.8     | Unprivileged containers    |
| CGROUP    | cgroup root view             | 4.6     | Hides parent cgroup path   |
| TIME      | CLOCK_MONOTONIC, BOOTTIME    | 5.6     | Pause/resume containers    |

#### How they actually work

Every process has a `task_struct->nsproxy` that points to a `struct nsproxy` containing pointers to one of each namespace type (USER lives separately on the credentials struct). Two processes can share some namespaces and not others — this is what makes Kubernetes pods (multiple containers, shared NET) possible.

The namespace itself is exposed as a file under `/proc/<pid>/ns/`:

```
$ ls -l /proc/$$/ns
lrwxrwxrwx 1 user user 0 ... cgroup -> 'cgroup:[4026531835]'
lrwxrwxrwx 1 user user 0 ... ipc    -> 'ipc:[4026531839]'
lrwxrwxrwx 1 user user 0 ... mnt    -> 'mnt:[4026531840]'
lrwxrwxrwx 1 user user 0 ... net    -> 'net:[4026531840]'
lrwxrwxrwx 1 user user 0 ... pid    -> 'pid:[4026531836]'
lrwxrwxrwx 1 user user 0 ... time   -> 'time:[4026531834]'
lrwxrwxrwx 1 user user 0 ... user   -> 'user:[4026531837]'
lrwxrwxrwx 1 user user 0 ... uts    -> 'uts:[4026531838]'
```

The number in brackets is the inode — a globally unique namespace ID. Two processes in the same NET namespace will have the same `net` inode.

#### Three syscalls

- **`clone(CLONE_NEWPID|CLONE_NEWNET|...)`** — create a new process in fresh namespaces (one per flag).
- **`unshare(CLONE_NEWPID|...)`** — detach the current process into new namespaces.
- **`setns(fd, nstype)`** — join an existing namespace by opening one of those `/proc/N/ns/*` files.

The CLI `unshare(1)` and `nsenter(1)` wrap these.

#### The eight, in detail

**MNT:** each namespace has its own mount table. `mount` and `umount` inside don't affect outside. Used by every container, by `firejail`, and by `systemd`'s per-service `PrivateTmp=`. Mount propagation flags (private, slave, shared) control whether mount events bleed across.

**UTS:** the *only* thing in here is the hostname and domain name. Inside an UTS NS, `hostname foo` doesn't change the host. UTS = "UNIX Time-sharing System", historic Bell Labs name.

**IPC:** SysV semaphores, message queues, shared memory; POSIX message queues (`/dev/mqueue`). Two processes in different IPC namespaces can't see each other's `shmget()` segments. Largely irrelevant in 2026 but Postgres uses SysV shm so containers get one.

**PID:** *the* container namespace. The first process in a fresh PID NS is PID 1. It inherits init's responsibilities — reap zombies. If PID 1 dies, the *whole namespace* dies (all other processes get SIGKILL). This is why `tini` and `dumb-init` exist as Docker init wrappers — most apps aren't designed to handle being PID 1.

**NET:** each NS has its own loopback, routing tables, ARP table, iptables/nftables rules, sockets, listening ports. Two NETNs can both bind 0.0.0.0:80. Connect them with `veth` pairs (a virtual ethernet cable, two ends), bridges, or `ipvlan` / `macvlan`. Kubernetes CNI plugins all live here.

**USER:** the wildest one. Inside a user NS, you can be UID 0 (root!) without actually being root on the host. Mappings are written to `/proc/<pid>/uid_map` and `gid_map`. Capabilities outside the NS may not exist — your "root" can chown files only inside this NS's idmap. Foundational for rootless Docker, rootless Podman, and unprivileged container runtimes. Since 3.8, but a CVE magnet for years.

**CGROUP:** virtualizes the *view* of the cgroup tree. A container in `/sys/fs/cgroup/foo/bar/baz/` sees `/` as its root and doesn't even know `/foo/bar/` exists above. Pretty: prevents leaking host structure to guests.

**TIME:** virtualizes `CLOCK_MONOTONIC` and `CLOCK_BOOTTIME` (NOT wallclock — `CLOCK_REALTIME` is global, by design). Lets you do `clock_settime(CLOCK_MONOTONIC, ...)` on container startup so `uptime` looks fresh. Critical for "pause and snapshot" container migration use cases.

#### Containers = namespaces + cgroups + capabilities + seccomp + LSM

This is the punchline. A "container" is not a thing. It is a **pile of kernel features** that, together, give the illusion of an isolated machine:

```
   ┌────────────────────── A "container" ──────────────────────┐
   │                                                            │
   │   namespaces        cgroups            capabilities        │
   │   ───────────       ──────────         ──────────────      │
   │   PID, NET, MNT,    cpu, memory,       drop CAP_SYS_ADMIN, │
   │   UTS, IPC, USER,   io, pids — caps    keep CAP_NET_BIND;  │
   │   CGROUP, TIME      on resources       fine-grained root   │
   │                                                            │
   │   seccomp                              LSM                 │
   │   ─────────                            ─────────────       │
   │   filter to allowed syscalls; deny     SELinux / AppArmor: │
   │   the other 300; eBPF-classic filter   mandatory access    │
   │                                        control on files,   │
   │                                        sockets, IPC        │
   │                                                            │
   └────────────────────────────────────────────────────────────┘
```

Every container runtime — Docker (containerd/runc), Podman, LXC, Kata, Firecracker (microVM, different model), systemd-nspawn — orchestrates these features. You can do it by hand:

```
# unshare --pid --net --mount --uts --ipc --user --cgroup --fork --map-root-user /bin/sh
```

There you are: in your own world. `ps` shows just you (after mounting `/proc`). `ip addr` shows just `lo`.

---

### Capabilities — Slicing Up Root

Traditional UNIX: UID 0 = root = bypass every check. Linux capabilities (since 2.2 in 1999) split that into ~40 distinct privileges. A binary or process can hold a subset.

Selected capabilities:

| Capability             | What it lets you do                                  |
|------------------------|------------------------------------------------------|
| CAP_SYS_ADMIN          | Almost-root: mount, swap, sethostname, ptrace any   |
| CAP_NET_ADMIN          | Configure interfaces, routing, iptables             |
| CAP_NET_BIND_SERVICE   | Bind to ports < 1024                                |
| CAP_NET_RAW            | Open raw / packet sockets (ping, tcpdump)           |
| CAP_SYS_PTRACE         | ptrace any process                                  |
| CAP_SYS_TIME           | Set system clock                                    |
| CAP_SYS_NICE           | Set scheduling policy / priority                    |
| CAP_DAC_OVERRIDE       | Bypass file permission checks                       |
| CAP_DAC_READ_SEARCH    | Bypass read/exec dir-traversal checks               |
| CAP_CHOWN              | Change UID/GID of files                             |
| CAP_FOWNER             | Treat self as file owner regardless                 |
| CAP_KILL               | Send signals to any process                         |
| CAP_BPF                | Load BPF programs (was CAP_SYS_ADMIN < 5.8)         |
| CAP_PERFMON            | perf_event_open (was CAP_SYS_ADMIN < 5.8)           |
| CAP_SYSLOG             | Read kernel ring buffer (dmesg)                     |
| CAP_SETUID/SETGID      | setuid/setgid                                       |

A process has five capability sets: **Permitted, Effective, Inheritable, Bounding, Ambient**. For our purposes: Effective is "what I can actually use right now"; Bounding is "the ceiling I'll never exceed"; Ambient is "what survives an exec".

```
$ cat /proc/$$/status | grep ^Cap
CapInh: 0000000000000000
CapPrm: 0000000000000000
CapEff: 0000000000000000
CapBnd: 000001ffffffffff
CapAmb: 0000000000000000
```

Decode:

```
$ capsh --decode=000001ffffffffff
0x000001ffffffffff=cap_chown,...,cap_checkpoint_restore
```

File capabilities are stored as xattrs:

```
$ getcap -r /usr/bin 2>/dev/null | head
/usr/bin/ping = cap_net_raw+ep
/usr/bin/mtr-packet = cap_net_raw+ep
/usr/bin/newgidmap = cap_setgid+ep
/usr/bin/newuidmap = cap_setuid+ep
/usr/bin/arping = cap_net_raw+ep
```

This is how `ping` works without setuid root: it's a regular binary with `CAP_NET_RAW` baked into its xattrs. Modern best practice — capabilities, not setuid.

---

### Seccomp — Syscall Filtering

`seccomp` (2005) and `seccomp-bpf` (2012) let a process say "from now on I'll only ever use these syscalls; if I touch any other, kill me / signal me / return ERRNO". The filter is a classic-BPF program (later eBPF in some incarnations).

```c
prctl(PR_SET_NO_NEW_PRIVS, 1);   // no setuid escapes
prctl(PR_SET_SECCOMP, SECCOMP_MODE_FILTER, &filter);
```

The Docker default seccomp profile blocks ~50 syscalls (e.g. `kexec_load`, `reboot`, `bpf` unless explicitly granted). Chrome sandboxes its renderer with seccomp. Systemd has `SystemCallFilter=` which compiles to seccomp-bpf for you.

```
$ systemctl cat sshd | grep -i SystemCall
SystemCallFilter=@system-service
SystemCallErrorNumber=EPERM
```

---

### LSM — Linux Security Modules

A framework, not a thing. The kernel has hundreds of `security_*` hooks; an LSM registers callbacks for them. SELinux, AppArmor, Smack, TOMOYO, and Yama are all LSMs. Newer: BPF LSM (kernel 5.7) lets you write LSM policies in eBPF.

| LSM       | Style                       | Distros                  |
|-----------|-----------------------------|---------------------------|
| SELinux   | Type-enforcement labels     | RHEL, Fedora, Rocky       |
| AppArmor  | Path-based profiles         | Ubuntu, SUSE, Debian      |
| Smack     | Simple labels               | Tizen, embedded           |
| BPF-LSM   | Programmable, eBPF          | Modern observability      |

Check what's loaded:

```
$ cat /sys/kernel/security/lsm
lockdown,capability,landlock,yama,apparmor,bpf
```

---

### eBPF — The Programmable Kernel

The biggest kernel innovation of the 2010s. **eBPF** = "extended Berkeley Packet Filter" — a misnomer; it long ago outgrew packet filtering and is now a general-purpose in-kernel virtual machine.

#### Why it exists

You want to ask the kernel a question: "trace every `openat()` and tell me which process opens which file". Options before eBPF:

- Write a kernel module — root, scary, can crash the kernel, GPL paperwork.
- Use ftrace / kprobes from text-shell scripts — limited; can't filter or aggregate efficiently.
- Use `strace` — `ptrace`-based, slows the target by 10-100x.

After eBPF: load a tiny program into the kernel, attach it to the `sys_enter_openat` tracepoint, aggregate counters in a kernel-side hash map, read the map from user space. Production-safe, sub-microsecond overhead.

#### The eBPF flow

```
       ┌──────────────────────── eBPF program lifecycle ───────────────────────┐
       │                                                                       │
       │ 1) You write C (or Python via bcc, or scripts via bpftrace)           │
       │                                                                       │
       │ 2) Compiler emits eBPF bytecode (clang -target bpf)                   │
       │                                                                       │
       │ 3) bpf(BPF_PROG_LOAD, ...) syscall — userspace ships bytecode in      │
       │                                                                       │
       │ 4) Kernel VERIFIER runs:                                              │
       │      - all paths terminate (bounded loops only)                        │
       │      - no out-of-bounds memory access (every pointer proven)          │
       │      - no uninitialized reads                                         │
       │      - register types tracked across branches                         │
       │      - helper-call permissions checked                                │
       │      - max 1M instructions; max 8K stack                              │
       │      ON FAILURE → -EACCES, program rejected                           │
       │                                                                       │
       │ 5) JIT compiles bytecode → native x86_64 / arm64 / riscv64            │
       │                                                                       │
       │ 6) Attach to a HOOK (kprobe, tracepoint, XDP, tc, LSM, sk_msg, ...)   │
       │                                                                       │
       │ 7) Kernel runs your code at hook fire — at near-native speed          │
       │                                                                       │
       │ 8) Userspace reads results from BPF MAPS via bpf() syscall            │
       │                                                                       │
       └───────────────────────────────────────────────────────────────────────┘
```

#### Attach points

- **kprobe / kretprobe**: any non-inlined kernel function entry / return.
- **uprobe / uretprobe**: any user-space function. (Trace your Postgres `ExecInsert`!)
- **tracepoint**: stable kernel-defined events (`sys_enter_openat`, `sched_switch`, ...). Listed in `/sys/kernel/debug/tracing/events/`.
- **USDT** (Userland Statically Defined Tracing): like tracepoints but in user binaries. PostgreSQL, MySQL, Node, Python all have USDT probes.
- **perf_event**: PMU counters, sampling profilers.
- **XDP** (eXpress Data Path): runs in the NIC driver before sk_buff is allocated. Drop / pass / redirect at line rate. DDoS scrubbing.
- **tc** (traffic control): runs at the qdisc layer; can rewrite or steer packets.
- **sk_msg / sockmap**: rewire socket data paths in userspace's name.
- **LSM**: implement security policy in eBPF (5.7+).
- **iter**: walk kernel structures (every task, every sock) and stream output to userspace.
- **fentry / fexit**: like kprobes but with verified context, lower overhead (5.5+).

#### BPF maps

eBPF programs can't allocate memory. They store state in **maps** allocated up-front. The kernel offers many map types:

| Map type          | Description                                  |
|-------------------|----------------------------------------------|
| HASH              | General hashmap                              |
| ARRAY             | Index → value, fixed size                    |
| LRU_HASH          | Hash with LRU eviction                       |
| PERCPU_HASH       | Per-CPU sharded hashmap (no contention)      |
| RINGBUF           | Lock-free ring buffer to userspace (5.8+)    |
| PERF_EVENT_ARRAY  | Older — perf-style ring buffers             |
| LPM_TRIE          | Longest-prefix-match (routing tables)        |
| SOCKMAP           | Socket lookups for sk_msg                    |
| STACK_TRACE       | Kernel/user stack capture                    |
| CGROUP_STORAGE    | Per-cgroup state                             |

#### CO-RE (Compile Once - Run Everywhere)

Old eBPF was version-fragile: kernel struct layouts changed between releases, so a tracer compiled against 5.10 wouldn't work on 5.15. **CO-RE** + **BTF** (BPF Type Format) record struct layouts in the kernel binary itself; libbpf relocates references at load time. One bytecode runs across kernels. Modern tracers (`bcc-tools`, `bpftrace`, Cilium agent) all use CO-RE.

#### Tools you'll actually use

- `bpftrace` — DTrace-like one-liners
- `bpftool` — load/inspect programs and maps; the canonical CLI
- `bcc` — Python frontend; `opensnoop`, `tcpconnect`, `execsnoop`, `runqlat`, `biolatency`...
- `libbpf` — the C library; CO-RE-aware
- `bpfman` — daemon for managing eBPF programs (post-2024)
- `Cilium`, `Falco`, `Tetragon` — production stacks

#### Version notes

- 3.18 (2014): first eBPF (`bpf()` syscall, sockets only)
- 4.1 (2015): kprobes
- 4.4: socket filter, persistent maps via `bpffs`
- 4.8: XDP
- 4.10: tracepoint attach
- 4.18: BTF basics
- 5.5: fentry/fexit
- 5.7: LSM hooks
- 5.8: ringbuf, CAP_BPF (no longer needs CAP_SYS_ADMIN)
- 5.13: bounded loops (real loops, not just unrolled)
- 6.x: multi-kprobe, kfuncs, eBPF in struct_ops

---

## Hands-On

### `cat /proc/interrupts | head -20` — per-CPU IRQ counters

```
$ cat /proc/interrupts | head -20
            CPU0       CPU1       CPU2       CPU3
   0:         38          0          0          0   IO-APIC    2-edge      timer
   1:          0          0          0       3421   IO-APIC    1-edge      i8042
   8:          0          0          1          0   IO-APIC    8-edge      rtc0
   9:          0          0       4523          0   IO-APIC    9-fasteoi   acpi
  16:        211         12         11         15   IO-APIC   16-fasteoi   ehci_hcd
  19:        103          5          5          5   IO-APIC   19-fasteoi   ata_piix
  24:    9213422       1130       1133       1129   IO-APIC   24-edge      eth0-tx-0
  25:       1129    9213887       1131       1126   IO-APIC   25-edge      eth0-rx-0
  26:       1131       1126    9213744       1129   IO-APIC   26-edge      eth0-rx-1
 NMI:          2          2          2          2   Non-maskable interrupts
 LOC:    8401234    8400112    8401334    8401223   Local timer interrupts
 SPU:          0          0          0          0   Spurious interrupts
 IPI:      40123      40456      40012      40009   Function call interrupts
```

Reading: NIC RX queues are pinned per-CPU (good). LOC is the LAPIC timer on each core. NMI = 2 each = healthy (the watchdog).

### `cat /proc/softirqs` — deferred-work counters

```
$ cat /proc/softirqs
                    CPU0       CPU1       CPU2       CPU3
          HI:          0          0          0          0
       TIMER:    8401234    8400112    8401334    8401223
      NET_TX:        123        119        102         98
      NET_RX:    9876543     325109     325107     325002
       BLOCK:      53210      52109      51203      51098
    IRQ_POLL:          0          0          0          0
     TASKLET:      12345      12211      12103      12001
       SCHED:    8401000    8400110    8401300    8401200
     HRTIMER:          0          0          0          0
         RCU:    9999999    9999100    9998200    9997300
```

NET_RX skewed to CPU0 — likely a single-queue NIC. Multi-queue NICs balance evenly.

### `ls /proc/$$/ns/` — your shell's namespaces

```
$ ls -l /proc/$$/ns/
total 0
lrwxrwxrwx 1 user user 0 Apr 27 10:03 cgroup -> 'cgroup:[4026531835]'
lrwxrwxrwx 1 user user 0 Apr 27 10:03 ipc    -> 'ipc:[4026531839]'
lrwxrwxrwx 1 user user 0 Apr 27 10:03 mnt    -> 'mnt:[4026531840]'
lrwxrwxrwx 1 user user 0 Apr 27 10:03 net    -> 'net:[4026531840]'
lrwxrwxrwx 1 user user 0 Apr 27 10:03 pid    -> 'pid:[4026531836]'
lrwxrwxrwx 1 user user 0 Apr 27 10:03 pid_for_children -> 'pid:[4026531836]'
lrwxrwxrwx 1 user user 0 Apr 27 10:03 time   -> 'time:[4026531834]'
lrwxrwxrwx 1 user user 0 Apr 27 10:03 time_for_children -> 'time:[4026531834]'
lrwxrwxrwx 1 user user 0 Apr 27 10:03 user   -> 'user:[4026531837]'
lrwxrwxrwx 1 user user 0 Apr 27 10:03 uts    -> 'uts:[4026531838]'
```

The inode numbers (`4026531835`...) are the *globally unique namespace IDs*. Two processes share a namespace if their inode numbers match.

### `unshare --user --pid --net --mount --uts /bin/sh` — live experiment

```
$ unshare --user --pid --net --mount --uts --fork --map-root-user /bin/sh
# whoami
root                                          # but only inside this NS!
# id
uid=0(root) gid=0(root) groups=0(root)
# hostname container1
# hostname
container1                                    # outside still your laptop
# ip link
1: lo: <LOOPBACK> mtu 65536 ...               # only loopback!
# mount -t proc proc /proc
# ps
  PID TTY          TIME CMD
    1 pts/0    00:00:00 sh                    # you are init now!
    2 pts/0    00:00:00 ps
# exit
```

You just hand-built the skeleton of a container.

### `find /sys/fs/cgroup -maxdepth 2 -type d | head -20` — cgroup tree

```
$ find /sys/fs/cgroup -maxdepth 2 -type d | head -20
/sys/fs/cgroup
/sys/fs/cgroup/system.slice
/sys/fs/cgroup/system.slice/cron.service
/sys/fs/cgroup/system.slice/dbus.service
/sys/fs/cgroup/system.slice/docker.service
/sys/fs/cgroup/system.slice/NetworkManager.service
/sys/fs/cgroup/system.slice/sshd.service
/sys/fs/cgroup/system.slice/systemd-journald.service
/sys/fs/cgroup/system.slice/systemd-logind.service
/sys/fs/cgroup/user.slice
/sys/fs/cgroup/user.slice/user-1000.slice
/sys/fs/cgroup/init.scope
```

Pure cgroup v2 tree. systemd organizes by `*.slice` (logical groups), `*.service` (daemons), `*.scope` (scopes).

### `cat /sys/fs/cgroup/cgroup.controllers`

```
$ cat /sys/fs/cgroup/cgroup.controllers
cpuset cpu io memory hugetlb pids rdma misc
```

Available controllers in this kernel.

### Read your shell's cgroup

```
$ cat /proc/$$/cgroup
0::/user.slice/user-1000.slice/user@1000.service/app.slice/app-glib-bash-2389.scope
```

The single `0::...` line is unique to v2 (v1 had a line per controller).

### `cat /proc/$$/status | grep -i cap` — capability sets

```
$ cat /proc/$$/status | grep -i cap
CapInh: 0000000000000000
CapPrm: 0000000000000000
CapEff: 0000000000000000
CapBnd: 000001ffffffffff
CapAmb: 0000000000000000
```

Inheritable, Permitted, Effective, Bounding, Ambient. As a normal user you have none effective; the bounding set is "everything except a few new ones".

```
$ capsh --decode=000001ffffffffff | tr ',' '\n' | head
0x000001ffffffffff=cap_chown
cap_dac_override
cap_dac_read_search
cap_fowner
cap_fsetid
cap_kill
cap_setgid
cap_setuid
cap_setpcap
cap_linux_immutable
```

### `getcap -r /usr/bin 2>/dev/null | head -10`

```
$ getcap -r /usr/bin 2>/dev/null | head -10
/usr/bin/ping cap_net_raw=ep
/usr/bin/mtr-packet cap_net_raw=ep
/usr/bin/arping cap_net_raw=ep
/usr/bin/newgidmap cap_setgid=ep
/usr/bin/newuidmap cap_setuid=ep
```

`+ep` = effective + permitted. `cap_net_raw` is the smallest capability that lets `ping` open a raw ICMP socket without being setuid root.

### `bpftool prog list | head -10` (root)

```
# bpftool prog list | head -10
1: cgroup_skb  tag 7be49e3934a125ba  gpl
        loaded_at 2026-04-27T10:00:01+0000  uid 0
        xlated 64B  jited 67B  memlock 4096B
2: cgroup_skb  tag 7be49e3934a125ba  gpl
        loaded_at 2026-04-27T10:00:01+0000  uid 0
        xlated 64B  jited 67B  memlock 4096B
17: kprobe  name handle_mm_fault  tag 5f60a9e68dd4f019  gpl
        loaded_at 2026-04-27T10:01:23+0000  uid 0
        xlated 1024B  jited 1100B  memlock 4096B
```

Every loaded eBPF program. systemd preloads cgroup_skb filters; tracing tools add kprobes.

```
# bpftool map list | head
1: array  name bss  flags 0x0
        key 4B  value 256B  max_entries 1  memlock 4096B
2: hash  name pid_to_latency  flags 0x0
        key 4B  value 8B  max_entries 10240  memlock 1175552B
```

### `bpftrace -e ...` — count `openat` calls per process

```
# bpftrace -e 'tracepoint:syscalls:sys_enter_openat { @[comm] = count(); }'
Attaching 1 probe...
^C

@[bash]: 12
@[ls]: 47
@[bpftrace]: 209
@[systemd]: 1503
@[Xorg]: 8821
```

Ctrl+C to stop and print. Useful for: who is hammering the filesystem? Which container is opening 9000 files/sec?

### `bpftrace -l 'tracepoint:*'` — list tracepoints

```
# bpftrace -l 'tracepoint:syscalls:sys_enter_*' | head -10
tracepoint:syscalls:sys_enter_accept
tracepoint:syscalls:sys_enter_accept4
tracepoint:syscalls:sys_enter_access
tracepoint:syscalls:sys_enter_acct
tracepoint:syscalls:sys_enter_add_key
tracepoint:syscalls:sys_enter_adduser
tracepoint:syscalls:sys_enter_bind
tracepoint:syscalls:sys_enter_bpf
tracepoint:syscalls:sys_enter_brk
tracepoint:syscalls:sys_enter_chdir
```

A few thousand tracepoints exist. All are stable kernel-version-to-version (in theory).

### Watch processes spawn live (root)

```
# bpftrace -e 'tracepoint:sched:sched_process_exec { printf("%-6d %s\n", pid, str(args->filename)); }'
Attaching 1 probe...
14523  /usr/bin/ls
14524  /usr/bin/grep
14525  /usr/sbin/sshd
14526  /usr/bin/cron
```

This is what `execsnoop` does internally — and it's the single best demo of why eBPF is magical.

### Send a signal to yourself

```
$ ( sleep 1; kill -USR1 $$ ) &
$ trap 'echo "got USR1"' USR1
$ wait
got USR1
```

### Block a syscall with seccomp (Python demo)

```python
import seccomp
f = seccomp.SyscallFilter(defaction=seccomp.ALLOW)
f.add_rule(seccomp.ERRNO(1), "openat")     # EPERM
f.load()
open("/etc/hostname")    # PermissionError: [Errno 1] Operation not permitted
```

(Requires `python3-seccomp`.)

### Inspect a running container's namespaces

```
$ docker run -d --rm --name demo nginx
abcd1234ef
$ pid=$(docker inspect --format '{{.State.Pid}}' demo)
$ sudo ls -l /proc/$pid/ns/
lrwxrwxrwx 1 root root 0 ... cgroup -> 'cgroup:[4026534567]'
lrwxrwxrwx 1 root root 0 ... net    -> 'net:[4026534568]'
lrwxrwxrwx 1 root root 0 ... pid    -> 'pid:[4026534569]'
...

$ sudo nsenter -t $pid -n ip addr     # enter the container's NET namespace
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 ...
21: eth0@if22: <BROADCAST,MULTICAST,UP> mtu 1500 ...
    inet 172.17.0.2/16 ...
```

`nsenter -n -t PID` — see the world from inside the namespace.

### Capability drop/keep with `capsh`

```
$ sudo capsh --user=stevie --inh=cap_net_raw -- -c 'ping -c1 1.1.1.1'
PING 1.1.1.1: 56 data bytes
64 bytes from 1.1.1.1: icmp_seq=0 ttl=58 time=2.1 ms
```

Ran ping as a normal user with just `cap_net_raw` set — no setuid, no root.

### Per-cpu interrupts pinned

```
$ for f in /proc/irq/*/smp_affinity_list; do
    n=$(basename $(dirname $f))
    a=$(cat $f)
    echo "IRQ $n -> CPUs $a"
  done | head -10
IRQ 0 -> CPUs 0
IRQ 1 -> CPUs 0
IRQ 8 -> CPUs 0-3
IRQ 9 -> CPUs 0-3
IRQ 24 -> CPUs 0
IRQ 25 -> CPUs 1
IRQ 26 -> CPUs 2
IRQ 27 -> CPUs 3
```

Multiqueue NIC — IRQs 24..27 each pinned to one CPU. This is the "RX flow steering" pattern, set up by `irqbalance` or by hand (`echo 1 > /proc/irq/24/smp_affinity_list`).

### Display the syscall table from the running kernel

```
$ ausyscall --dump | head
Using x86_64 syscall table:
0       read
1       write
2       open
3       close
4       stat
5       fstat
6       lstat
7       poll
8       lseek
9       mmap
```

### Trace which syscalls a process makes (strace)

```
$ strace -c -p $(pidof nginx | awk '{print $1}') 2>&1 | head -20
strace: Process 1234 attached
^Cstrace: Process 1234 detached
% time     seconds  usecs/call     calls    errors syscall
------ ----------- ----------- --------- --------- ----------------
 38.21    0.001234         123        10           epoll_wait
 19.10    0.000617          77         8           accept4
 12.30    0.000397          49         8           recvfrom
 ...
```

`strace` uses `ptrace`, not eBPF — it is *very* slow. For production, prefer `perf trace` or eBPF tools.

---

## Common Confusions

**"Is a container just a VM?"**
Broken: "Containers are little VMs."
Fixed: A VM has its own kernel; a hypervisor in Ring -1 fakes hardware; the guest does its own boot. A container shares the host's kernel; isolation comes from namespaces + cgroups. No virtualized hardware. One kernel. Hundreds of containers.

**"Why does my Docker container show PID 1 = my app?"**
Broken: "Docker remapped the PIDs!"
Fixed: PID namespace. Inside the namespace, the first process is PID 1; the kernel renumbers fork/clone returns. From the host (different namespace), that same task has a different (larger) PID. Both are real; both are this kernel's job to maintain.

**"Why can't my container see the host's network interfaces?"**
Broken: "Some firewall rule."
Fixed: NET namespace. The container has its own list of interfaces — usually just `lo` and a `eth0` veth peer. The host's `eno1`, `wlan0`, `docker0` simply aren't visible from inside that namespace. Add `--network=host` to disable the NET namespace and the container sees the host's stack directly.

**"What's the difference between fork() and clone()?"**
Broken: "fork is for processes, clone is for threads."
Fixed: `fork()` is `clone()` with a default flag set (no shared memory, no shared fds, no shared signal handlers, new PID). `clone(CLONE_VM|CLONE_FS|CLONE_FILES|CLONE_SIGHAND|CLONE_THREAD)` is what `pthread_create` calls — sharing everything. clone with CLONE_NEWPID etc. creates new namespaces. *All* user-space process/thread creation goes through clone (or its modern sibling clone3).

**"Why is eBPF safe but kernel modules aren't?"**
Broken: "eBPF runs in user space."
Fixed: eBPF runs in *kernel* space. The difference is the **verifier**: a static analyzer that proves termination, bounded memory accesses, no uninitialized reads, no stale pointers. Kernel modules are ordinary C ELF — load them and any bug owns the kernel. eBPF programs that don't meet the verifier's proof obligations are rejected at load time.

**"Why does `kill -9` not always work?"**
Broken: "SIGKILL kills everything."
Fixed: SIGKILL is delivered when the target next returns to user mode. A task in **D-state** (uninterruptible sleep) is blocked inside a kernel routine — usually waiting on hardware (NFS, a stuck disk). Until that routine returns, SIGKILL queues but doesn't fire. Solution: fix the underlying I/O issue or reboot.

**"My program crashed with SIGSEGV but I'm not dereferencing null."**
Broken: "Must be the kernel."
Fixed: SIGSEGV fires for any access to an unmapped or wrong-permission page. Stack overflow (recursion ate the page guard), writing to a `mmap(...,PROT_READ)` region, jumping into NX-protected memory, hitting a `mremap`-shifted pointer — all SIGSEGV. The fault address is in `siginfo_t.si_addr` (set `SA_SIGINFO`); compare to `/proc/self/maps`.

**"The container is using more memory than I set with cgroups."**
Broken: "cgroups don't work."
Fixed: Page cache. `memory.max` includes anon + page cache; `memory.current` is what counts. If a workload reads 100GB through a 4GB cap, the kernel evicts pages on the way (RSS stays bounded). What you may be seeing is *the cgroup's whole subtree* — sum of all descendants. Check with `cat /sys/fs/cgroup/.../memory.stat`.

**"I unshared a NET namespace and now there's no internet."**
Broken: "unshare is broken."
Fixed: A fresh NET namespace has *only* loopback. To talk to the outside world you need a `veth` pair, an IP, a route, and (probably) iptables NAT on the host. Manually:
```
# ip link add veth0 type veth peer name veth1
# ip link set veth1 netns <pid>
# ip addr add 10.0.0.1/24 dev veth0
# ip link set veth0 up
# nsenter -t <pid> -n ip addr add 10.0.0.2/24 dev veth1
# nsenter -t <pid> -n ip link set veth1 up
# nsenter -t <pid> -n ip link set lo up
# nsenter -t <pid> -n ip route add default via 10.0.0.1
```
Now you have networking. Docker does this for you.

**"Why does `cgroup.controllers` not list `cpu` even though I want to limit CPU?"**
Broken: "My kernel is broken."
Fixed: cgroup v2 requires you to enable controllers in *parent* `cgroup.subtree_control` before children can use them. The root might list `cpu`, but a deep child only sees what's been delegated downward. Walk the chain: `for d in /sys/fs/cgroup /sys/fs/cgroup/foo /sys/fs/cgroup/foo/bar; do echo $d:; cat $d/cgroup.subtree_control; done`.

**"I added CAP_NET_BIND_SERVICE but my unprivileged process still can't bind port 80."**
Broken: "Capabilities aren't real."
Fixed: Likely the *Effective* set is empty after exec. By default, capabilities don't propagate across exec unless they're in the **Ambient** set. `setcap cap_net_bind_service+eip /path/to/binary` is the fix — file capabilities, persisted in xattrs.

**"Why is my eBPF program rejected for `unbounded loop`?"**
Broken: "The verifier hates me."
Fixed: The verifier requires every loop to be bounded so it can prove termination. Use `#pragma unroll` for small fixed iterations, or — on 5.13+ — use the supported "bounded loop" pattern: `for (int i = 0; i < N; i++)` where `N` is a compile-time constant ≤ 1M. Don't read the bound from a map at runtime.

**"My container's processes show up on the host with wild UIDs like 100000."**
Broken: "Something's leaking."
Fixed: USER namespace ID mapping. `uid_map` says "container UID 0..65535 maps to host UID 100000..165535". Inside, processes are root (UID 0); on the host they're an unprivileged subuid range. This is rootless containers working as designed — a container "root" who actually has no privileges on the host.

**"`int 0x80` worked yesterday, today my 64-bit binary returns the wrong value."**
Broken: "The kernel changed."
Fixed: `int 0x80` on x86_64 dispatches through the *32-bit* compat syscall table, where numbers and arg conventions differ. `sys_write` is `4` on i386 but `1` on x86_64. Use `syscall` (the instruction) on x86_64.

**"I attached a kprobe to `do_sys_open` and got nothing."**
Broken: "kprobes are unreliable."
Fixed: The function may have been inlined, renamed, or replaced. Modern kernels prefer `do_sys_openat2`. Use a tracepoint when one exists (`tracepoint:syscalls:sys_enter_openat`) — tracepoints are stable; kprobe targets are not.

**"`ps -ef` inside a container shows host processes."**
Broken: "PID namespace doesn't work."
Fixed: `ps` reads `/proc`. If you didn't mount a fresh `/proc` inside the namespace, it sees the host's. Inside an unshared PID NS: `mount -t proc proc /proc` to get a `/proc` that reflects the *current* PID namespace.

**"Why can't I `ptrace` my container's process from outside?"**
Broken: "ptrace is broken."
Fixed: PID namespace + Yama LSM + capabilities. From outside, you need `CAP_SYS_PTRACE` *in the target's user namespace*. Yama (`/proc/sys/kernel/yama/ptrace_scope`) may further restrict to ptrace-of-children. Setting it to 0 (off) is a common dev workaround; leave it at 1 (default, restricted) in prod.

---

## Vocabulary

- **ring 0** — top-privilege mode on x86; kernel runs here.
- **ring 1** — historical; unused on Linux.
- **ring 2** — historical; unused on Linux.
- **ring 3** — least-privilege mode on x86; user space.
- **ring -1** — Intel VMX root (or AMD SVM); hypervisor mode.
- **EL0** — ARM exception level 0; user space.
- **EL1** — ARM exception level 1; kernel.
- **EL2** — ARM exception level 2; hypervisor.
- **EL3** — ARM exception level 3; secure monitor (ATF).
- **VMX** — Intel virtualization extensions; vmx-root and vmx-non-root.
- **SVM** — AMD Secure Virtual Machine extensions.
- **SMM** — System Management Mode; firmware ring "below" everything.
- **CPL** — Current Privilege Level (low bits of CS register).
- **DPL** — Descriptor Privilege Level (per IDT entry).
- **GP fault** — General Protection fault, vector 13.
- **#PF** — Page Fault, vector 14.
- **#UD** — Undefined Opcode, vector 6.
- **#DE** — Divide Error, vector 0.
- **#DF** — Double Fault, vector 8.
- **TSS** — Task State Segment; holds the safe kernel stack pointer.
- **GDT** — Global Descriptor Table; segment descriptors.
- **LDT** — Local Descriptor Table; per-process segments (rare on Linux).
- **IDT** — Interrupt Descriptor Table; 256 gates.
- **IDTR** — IDT base + limit register.
- **APIC** — Advanced Programmable Interrupt Controller.
- **LAPIC** — Local APIC, one per core.
- **IOAPIC** — I/O APIC, routes external IRQs.
- **x2APIC** — extended APIC for >256 CPUs.
- **IRQ** — Interrupt Request, hardware interrupt.
- **NMI** — Non-Maskable Interrupt.
- **IPI** — Inter-Processor Interrupt.
- **MSI** — Message Signalled Interrupt.
- **MSI-X** — extended MSI; up to 2048 vectors per device.
- **trap** — synchronous, intentional interrupt (e.g. syscall).
- **exception** — synchronous, fault-induced interrupt (e.g. PF).
- **top half** — fast IRQ handler, runs with IRQ disabled.
- **bottom half** — deferred IRQ work, runs with IRQ enabled.
- **softirq** — softirq, top of stack, no sleep.
- **tasklet** — softirq-based per-device deferred work.
- **workqueue** — kernel thread; can sleep.
- **threaded IRQ** — per-IRQ kthread (RT kernels).
- **syscall** — system call; user→kernel transition.
- **vsyscall** — legacy fast syscall area at fixed VA (deprecated).
- **vDSO** — Virtual Dynamic Shared Object; per-process kernel-mapped page for fast syscalls.
- **sysenter** — older fast-syscall instruction (i386).
- **syscall (instruction)** — x86_64 fast-syscall instruction.
- **SVC** — ARM `svc` instruction; supervisor call.
- **SYSRET** — return from `syscall`.
- **LSTAR** — MSR holding the syscall entry RIP.
- **STAR** — MSR holding syscall CS/SS selectors.
- **EINTR** — errno: syscall interrupted by signal.
- **ERESTARTSYS** — kernel-internal: restart with SA_RESTART.
- **ERESTARTNOINTR** — kernel-internal: always restart.
- **ERESTART_RESTARTBLOCK** — kernel-internal: restart via restart_syscall.
- **signal** — software interrupt to a process.
- **SIGTERM** — polite terminate (15).
- **SIGKILL** — forced terminate (9), uncatchable.
- **SIGSTOP** — pause, uncatchable.
- **SIGCONT** — resume.
- **SIGCHLD** — child status changed (default ignored).
- **SIGSEGV** — segfault.
- **SIGBUS** — bus error (alignment, mmap past EOF).
- **SIGPIPE** — wrote to closed pipe.
- **SIGHUP** — terminal hung up; conventional "reload".
- **SA_RESTART** — sigaction flag: auto-restart syscalls.
- **SA_SIGINFO** — sigaction flag: extended siginfo_t in handler.
- **sigreturn** — syscall used by handler trampoline to restore.
- **rt_sigreturn** — modern realtime sigreturn variant.
- **realtime signal** — queued, not coalesced (32–64).
- **signalfd** — convert signals to a readable fd.
- **namespace** — virtualized view of a global resource.
- **PID NS** — virtual PID space.
- **NET NS** — virtual network stack.
- **MNT NS** — virtual mount table.
- **UTS NS** — virtual hostname/domain.
- **IPC NS** — virtual SysV/POSIX IPC.
- **USER NS** — virtual UID/GID/cap mapping.
- **CGROUP NS** — virtual cgroup root view.
- **TIME NS** — virtual CLOCK_MONOTONIC/BOOTTIME.
- **clone(2)** — create process/thread, optionally with new namespaces.
- **unshare(2)** — detach current task into new namespaces.
- **setns(2)** — join an existing namespace by fd.
- **veth** — virtual ethernet pair; connects netns.
- **macvlan / ipvlan** — additional NIC virtualization for netns.
- **cgroup v1** — legacy multi-hierarchy cgroups.
- **cgroup v2** — unified hierarchy (default in modern systemd).
- **controller** — cgroup subsystem (cpu, memory, io, ...).
- **delegation** — handing a cgroup subtree to an unprivileged user.
- **cgroup.subtree_control** — file enabling controllers downward.
- **cgroup.procs** — file listing PIDs in this cgroup.
- **memory.max** — v2 hard memory cap.
- **cpu.max** — v2 CPU bandwidth cap.
- **OOM killer** — Out-Of-Memory killer; picks tasks to terminate.
- **oom_score** — kernel's per-task OOM weighting.
- **capability** — fine-grained slice of root.
- **CAP_SYS_ADMIN** — almost-root catchall.
- **CAP_NET_ADMIN** — networking config.
- **CAP_NET_BIND_SERVICE** — bind low ports.
- **CAP_NET_RAW** — raw sockets.
- **CAP_SYS_PTRACE** — ptrace any.
- **CAP_BPF** — load eBPF (5.8+).
- **CAP_PERFMON** — perf_event (5.8+).
- **Permitted set** — capability ceiling currently held.
- **Effective set** — capabilities currently in effect.
- **Inheritable set** — pre-exec carryover.
- **Bounding set** — never exceed this.
- **Ambient set** — survives a non-privileged exec.
- **file capability** — capability stored as xattr.
- **seccomp** — syscall filter.
- **seccomp-bpf** — classic-BPF–based seccomp.
- **classic BPF** — original BPF (1992); 32-bit, restricted.
- **eBPF** — extended BPF; 64-bit registers, Turing-incomplete by design.
- **BPF** — generic; almost always means eBPF in modern usage.
- **JIT** — Just-In-Time compiler; bytecode → native.
- **verifier** — kernel static analyzer; proves safety.
- **map (bpf)** — typed kernel data structure shared with userspace.
- **kprobe** — eBPF attach point at any kernel function entry.
- **kretprobe** — kprobe at function return.
- **uprobe** — eBPF attach in user binary.
- **uretprobe** — uprobe at function return.
- **tracepoint** — stable kernel event hook.
- **USDT** — user-space statically-defined tracepoint.
- **perf event** — hardware/software perf counter event.
- **XDP** — eXpress Data Path; eBPF in NIC driver.
- **tc** — traffic control; classic Linux QoS, eBPF-attachable.
- **sk_msg** — socket-message redirect eBPF.
- **fentry / fexit** — verified-context kprobe replacement (5.5+).
- **iter** — eBPF iterator over kernel structures.
- **CO-RE** — Compile Once Run Everywhere.
- **BTF** — BPF Type Format; kernel struct layouts.
- **libbpf** — canonical C library for eBPF.
- **bpftool** — CLI for managing eBPF programs/maps.
- **bpftrace** — DTrace-like one-liner frontend.
- **bcc** — Python frontend for eBPF; tools collection.
- **ftrace** — older, non-eBPF in-kernel tracer.
- **LSM** — Linux Security Module framework.
- **SELinux** — type-enforcement LSM.
- **AppArmor** — path-based LSM.
- **Yama** — ptrace-restriction LSM.
- **Landlock** — unprivileged sandbox LSM (5.13+).
- **BPF LSM** — programmable LSM via eBPF (5.7+).
- **MAC** — Mandatory Access Control.
- **DAC** — Discretionary Access Control.
- **MSR** — Model-Specific Register.
- **TSC** — Time Stamp Counter.
- **CR3** — control register 3; page-table base.
- **PML4** — top-level x86_64 paging table.
- **TLB** — Translation Lookaside Buffer.
- **TLB shootdown** — IPI invalidating remote TLBs.
- **D-state** — uninterruptible sleep; ignores SIGKILL.
- **R-state** — running.
- **S-state** — interruptible sleep.
- **T-state** — stopped.
- **Z-state** — zombie.
- **container** — bundle of namespaces + cgroups + capabilities + seccomp + LSM.
- **runc** — low-level OCI container runtime.
- **containerd** — high-level container manager.
- **CRI** — Container Runtime Interface (Kubernetes).
- **OCI** — Open Container Initiative spec.

---

## Try This

1. **Find your shell's namespaces and a child shell's:**
   ```
   $ ls /proc/$$/ns/ -l > /tmp/parent_ns
   $ bash -c 'ls /proc/$$/ns/ -l > /tmp/child_ns'
   $ diff /tmp/parent_ns /tmp/child_ns
   ```
   Identical inodes → they share every namespace. Now compare with a Docker container; everything differs.

2. **Build a tiny container by hand:**
   ```
   $ unshare --user --pid --net --mount --uts --ipc --cgroup --fork --map-root-user /bin/sh
   # mount -t proc proc /proc
   # ps                # only `sh` and `ps`!
   # hostname mybox; hostname
   # exit
   ```

3. **Limit memory of a shell to 50 MB and try to allocate more:**
   ```
   # mkdir /sys/fs/cgroup/exp
   # echo "+memory" > /sys/fs/cgroup/cgroup.subtree_control
   # echo $$ > /sys/fs/cgroup/exp/cgroup.procs
   # echo 50M > /sys/fs/cgroup/exp/memory.max
   # python3 -c 'a = bytearray(200*1024*1024)'
   Killed                # OOM killer fires inside cgroup
   ```

4. **Trace every `execve` for 10 seconds:**
   ```
   # bpftrace -e 'tracepoint:syscalls:sys_enter_execve { printf("%d %s\n", pid, str(args->filename)); }'
   ```
   Walk around your laptop, open apps, watch the firehose.

5. **Compare strace overhead vs eBPF:**
   ```
   $ time strace -c -- /bin/true
   ...some 30 syscalls counted...

   real    0m0.012s    # took 12ms, vs ~0.5ms unstraced

   # vs bpftrace, which doesn't slow the target
   # bpftrace -e 'tracepoint:syscalls:sys_enter_* /pid == $1/ { @[probe] = count(); }'
   ```

6. **Read a binary's capabilities:**
   ```
   $ getcap /usr/bin/ping
   /usr/bin/ping cap_net_raw=ep
   ```

7. **Drop a capability and see what breaks:**
   ```
   # capsh --drop=cap_net_raw -- -c 'ping -c1 8.8.8.8'
   ping: socket: Operation not permitted
   ```

8. **Inspect what your kernel uses for LSMs:**
   ```
   $ cat /sys/kernel/security/lsm
   lockdown,capability,landlock,yama,apparmor,bpf
   ```

9. **Pin an IRQ to a single CPU:**
   ```
   # cat /proc/irq/24/smp_affinity_list
   0-3
   # echo 2 > /proc/irq/24/smp_affinity_list
   # cat /proc/irq/24/smp_affinity_list
   2
   ```

10. **Use `bpftool` to dump a running program's bytecode:**
    ```
    # bpftool prog list
    # bpftool prog dump xlated id 17
    ```

---

## Deeper Mechanics

This section adds depth that the surface-level walkthroughs above intentionally skipped — appendix-style material, but in the body of the sheet, because the goal is "never leave the terminal".

### The boot path through the IDT

When the kernel boots, one of the very first things it does (after switching into long mode and setting up identity-mapped pages) is build the IDT. The early IDT is hardcoded — vectors for double-fault, machine-check, and a handful of CPU exceptions. After scheduler bring-up, `init_IRQ()` and `apic_intr_mode_init()` populate the rest. By the time userspace exists, all 256 vectors are wired up.

```
   asm-x86 source pointers:
       arch/x86/kernel/idt.c               build IDT
       arch/x86/kernel/traps.c             vector handlers
       arch/x86/entry/entry_64.S           the assembly door (SYSCALL_64, error_entry, ...)
       arch/x86/kernel/apic/apic.c         LAPIC setup
       arch/x86/kernel/irq.c               do_IRQ dispatcher
```

Reading the kernel source is a rite of passage. `entry_64.S` is the most important file in the kernel for understanding the user/kernel boundary; it is hand-written assembly because every cycle counts, and every register save must be exactly right or the kernel crashes.

### Per-CPU areas and `gs` register magic

The kernel needs per-CPU state — the current task, the scheduler runqueue, the irq stack, the page tables for kernel mapping. On x86_64, the **gs** segment register holds the base of the per-CPU area for the current CPU. When you write `current` in kernel code, the macro emits `mov %gs:current_task, %rax`. The CPU dereferences `gs` automatically.

Switching CPUs (e.g. via `swapgs`) is part of every user→kernel transition. A misordered `swapgs` is a famous class of CVEs (Meltdown-related variants).

### The full picture of a `read()` call

Take the simplest user code:

```c
char buf[16];
ssize_t n = read(fd, buf, 16);
```

What happens, top to bottom?

```
   USER SPACE
     │  call read()        — libc (glibc/musl)
     │     │
     │     ▼
     │  mov $0, %rax        ; SYS_read
     │  mov fd, %rdi
     │  lea buf, %rsi
     │  mov $16, %rdx
     │  syscall              ; ← THE TRAP
     │
   ─ ─ ─ ─ ─ Ring 3 → Ring 0 ─ ─ ─ ─ ─
     │
   KERNEL
     │  CPU loads RIP from LSTAR MSR
     │  → entry_SYSCALL_64
     │     swapgs                       ; load kernel %gs
     │     load kernel stack from TSS
     │     push user regs onto kstack
     │     ...
     │     call do_syscall_64
     │       → syscall_table[%rax] = sys_read
     │           → ksys_read
     │              → vfs_read
     │                 → fdget(fd) — find struct file
     │                    → file->f_op->read_iter
     │                       (filesystem-specific: ext4, xfs, tmpfs, ...)
     │                       → may sleep waiting for disk I/O
     │                       → may use page cache (instant if cached)
     │                       → copies bytes into user buf via copy_to_user
     │  syscall_exit:
     │     check signals_pending — if so set up handler frame
     │     restore user regs
     │     swapgs
     │     sysretq                  ; ← RETURN TO USER
     │
   ─ ─ ─ ─ ─ Ring 0 → Ring 3 ─ ─ ─ ─ ─
     │
   USER
        rax = bytes read (or -errno)
```

Every `read`, every `write`, every `epoll_wait` takes that road. Memorize it.

### `copy_to_user` and `copy_from_user`

Inside the kernel, you cannot just dereference a user pointer. The user might pass `0xdeadbeef`. The page might not exist. The address might be in a hostile mapping designed to attack speculative execution. Two helpers exist:

- `copy_from_user(kdst, usrc, n)` — copies n bytes from user `usrc` to kernel `kdst`.
- `copy_to_user(udst, ksrc, n)` — opposite direction.

They're implemented with **exception fixups**: a page-fault handler entry says "if a fault happens at this address (inside the copy loop), don't crash — return -EFAULT". This is how the kernel safely accepts arbitrary user pointers.

### How signals are *actually* delivered

You set a handler with `sigaction()`. A signal arrives. The kernel sets a flag in the task's pending mask. The next time the task returns to user mode (often after the *current* syscall completes), the kernel notices `pending` is non-empty, and:

1. Picks the highest-priority deliverable signal.
2. Reserves space on the user stack (or a separate **signal stack** if `sigaltstack` was set).
3. Writes a `siginfo_t`, the saved register state, and a return trampoline pointer there.
4. Sets the user RIP to the handler entry.
5. Returns to user mode.

The handler runs. When it returns, it doesn't `ret` — it transfers control to the trampoline, which calls `rt_sigreturn(2)` — a syscall that says "restore my pre-signal state". The kernel pops everything back. The original syscall, if interrupted with `SA_RESTART`, restarts.

Signal delivery is one of the most subtle pieces of the kernel. Studied carefully because it's a CVE goldmine: anything that touches the user stack from kernel context must be paranoid.

### Why `unshare --pid` requires `--fork`

```
$ unshare --pid /bin/sh
unshare: cannot fork: Operation not permitted
```

The first process to enter a new PID NS becomes PID 1 *of that NS*. But your existing shell can't simply "become PID 1" — it already has a PID elsewhere. The kernel insists you create a new process with `--fork`, and *that* new process is PID 1. Without `--fork`, anything you exec next will be the first child of a non-existent PID 1, which the kernel refuses.

### Bounded loops and the verifier — a worked example

The verifier rejects this (pre-5.13):

```c
int n = 0;
while (n < unknown_value)  // unbounded
    n += 1;
```

But accepts this:

```c
#pragma unroll
for (int i = 0; i < 16; i++)   // unrolled at compile time
    do_something(i);
```

And on 5.13+ also this, with annotations:

```c
for (int i = 0; i < N && i < 1024; i++)   // explicit bound
    do_something(i);
```

The verifier walks every basic block, tracking register types and value ranges. Every instruction that could go wrong is statically proven OK. If a path can possibly read from `pkt + offset` where `offset` isn't bounded, rejection. The compiler must emit code the verifier can prove. Good news: clang's BPF backend knows the rules.

### The bpf() syscall in detail

```
   bpf(int cmd, union bpf_attr *attr, unsigned int size);
```

`cmd` is one of:

- `BPF_PROG_LOAD` — load a program (verifier runs here)
- `BPF_MAP_CREATE` — allocate a map
- `BPF_MAP_LOOKUP_ELEM` / `UPDATE_ELEM` / `DELETE_ELEM` — userland talking to a map
- `BPF_PROG_ATTACH` / `DETACH` — wire to a hook (cgroup, sockmap)
- `BPF_LINK_CREATE` — modern stable attach (perf, kprobe, tracepoint)
- `BPF_OBJ_GET_INFO_BY_FD` — introspection
- `BPF_BTF_LOAD` — load type info for CO-RE

There are roughly 40 commands today. `bpftool` calls them all under the hood.

### Network namespace data model

```
   struct net {
       struct list_head list;          // global list of all netns
       struct user_namespace *user_ns; // the user NS that owns this netns
       atomic_t          count;
       struct nlmsghdr   netlink_socket;
       struct net_device *loopback_dev;
       struct list_head  dev_base_head; // all interfaces
       struct list_head  rules_ops;     // routing rules
       ...
   };
```

Every network operation in the kernel takes a `struct net *` parameter (often implicit via the current task). All kernel routing lookups, ARP entries, sockets, and netfilter tables are per-`struct net`. There is no way to forget which namespace you're in — the type system enforces it.

### `ip netns` — netns from the command line

```
# ip netns add demo
# ip netns list
demo (id: 0)
# ip -n demo link
1: lo: <LOOPBACK> mtu 65536 ...
# ip link add veth-h type veth peer name veth-d
# ip link set veth-d netns demo
# ip addr add 10.10.0.1/24 dev veth-h
# ip link set veth-h up
# ip -n demo addr add 10.10.0.2/24 dev veth-d
# ip -n demo link set veth-d up
# ip -n demo link set lo up
# ping -c1 10.10.0.2
PING 10.10.0.2 56 data bytes
64 bytes from 10.10.0.2: icmp_seq=1 ttl=64 time=0.046 ms
```

`ip netns` is the friendly wrapper. Everything it does could be replicated with `unshare`, `nsenter`, and `mount`, but `ip netns` keeps a directory of named netns at `/var/run/netns/` for you.

### A walking tour of `/sys/fs/cgroup/`

For a cgroup `foo`:

```
/sys/fs/cgroup/foo/
├── cgroup.controllers       — what controllers are available
├── cgroup.subtree_control   — which controllers children may use
├── cgroup.procs             — PIDs in this cgroup (one per line)
├── cgroup.threads           — TIDs (cgroup-aware threading)
├── cgroup.events            — populated/notify_on_release flag
├── cgroup.freeze            — write 1 to freeze, 0 to thaw (5.2+)
├── cpu.weight               — relative CPU share (1..10000, default 100)
├── cpu.weight.nice          — same as weight, in `nice` values
├── cpu.max                  — quota period (us)
├── cpu.stat                 — usage_usec, nr_periods, throttled_usec
├── cpu.pressure             — PSI (5.2+) — % of time delayed by CPU contention
├── memory.max               — hard cap
├── memory.high              — soft cap; reclaim aggressively above
├── memory.low               — protection floor (don't reclaim below)
├── memory.min               — even harder floor (don't reclaim ever)
├── memory.swap.max          — swap cap
├── memory.current           — actual current usage
├── memory.stat              — file/anon/slab/sock breakdown
├── memory.events            — counters: low, high, max, oom, oom_kill
├── memory.pressure          — PSI for memory
├── io.max                   — per-device IOPS / bytes
├── io.weight                — relative weight
├── io.stat                  — per-device counters
├── pids.max                 — fork ceiling
├── pids.current             — current count
└── ...
```

The cgroup v2 file naming convention is *consistent*: `<resource>.<aspect>`. Take 5 minutes and `cat` every file in your cgroup; it will demystify everything in production.

### PSI — Pressure Stall Information

Kernel 4.20+ exposes `cpu.pressure`, `memory.pressure`, `io.pressure`. Format:

```
$ cat /proc/pressure/cpu
some avg10=0.12 avg60=0.34 avg300=0.21 total=12345678
full avg10=0.05 avg60=0.18 avg300=0.10 total=234567
```

`some` = at least one task delayed; `full` = all runnable tasks delayed. PSI is what modern container orchestrators use to decide when to evict — far more useful than load average.

### eBPF map: a real example

```c
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __type(key, u32);
    __type(value, u64);
    __uint(max_entries, 1024);
} pid_to_count SEC(".maps");

SEC("tracepoint/syscalls/sys_enter_openat")
int trace_open(struct trace_event_raw_sys_enter *ctx)
{
    u32 pid = bpf_get_current_pid_tgid() >> 32;
    u64 *cnt, init = 1;
    cnt = bpf_map_lookup_elem(&pid_to_count, &pid);
    if (cnt) (*cnt)++;
    else bpf_map_update_elem(&pid_to_count, &pid, &init, BPF_ANY);
    return 0;
}
char LICENSE[] SEC("license") = "GPL";
```

That's a complete eBPF program. Compile with `clang -O2 -target bpf -c open.c -o open.o`. Load with `bpftool prog load open.o /sys/fs/bpf/open`. Read the map with `bpftool map dump pinned /sys/fs/bpf/open_map`.

### Some XDP basics

XDP runs in the NIC driver before the kernel allocates an `sk_buff`. You see raw packet bytes; you decide:

- `XDP_PASS` — let the kernel handle it normally.
- `XDP_DROP` — black-hole it (DDoS scrubbing).
- `XDP_TX` — bounce back out the same NIC (load balancers).
- `XDP_REDIRECT` — out a different NIC, or to userspace via AF_XDP.
- `XDP_ABORTED` — error path; counters bump.

Cilium uses XDP for service load balancing at line rate. Cloudflare's Magic Transit uses XDP. The Facebook L4 LB Katran uses XDP. This is what "the kernel can be programmed" means in practice.

### When NOT to use eBPF

- You need to *modify* arbitrary kernel data structures (eBPF is mostly read-only by design).
- You need unbounded loops and arbitrary memory (write a kernel module).
- Your hook isn't supported in your kernel (a 4.18 kernel won't have BTF-based CO-RE).
- You can do it from userspace with `epoll`/`io_uring` at acceptable cost.

eBPF is a scalpel, not a hammer.

### io_uring — the new fast syscall path

Not strictly part of the syscall mechanism but worth a mention. `io_uring` (5.1+) lets userspace queue dozens or hundreds of I/O ops via two ring buffers (submission and completion) shared with the kernel. The kernel processes them in batch. Result: tens of thousands of file ops with one syscall. The hottest async I/O API in modern Linux.

```c
struct io_uring ring;
io_uring_queue_init(64, &ring, 0);
struct io_uring_sqe *sqe = io_uring_get_sqe(&ring);
io_uring_prep_read(sqe, fd, buf, len, off);
io_uring_submit(&ring);
struct io_uring_cqe *cqe;
io_uring_wait_cqe(&ring, &cqe);
```

Worth a sheet on its own (`cs system io-uring`).

### Why `clone3()` exists

`clone()` had eight or so flag arguments and was running out of bits. `clone3(2)` (5.3+) takes a struct, can pass tids, set namespace fds individually, request a particular cgroup, and is open-ended for future flags. New code should prefer it. glibc's `pthread_create` switched to `clone3` in 2.34.

### `prctl` — the catch-all

`prctl(2)` is the "miscellaneous process attribute" syscall. Selected operations:

| Op                      | Effect                                                    |
|-------------------------|-----------------------------------------------------------|
| PR_SET_NAME             | rename your task (visible in `ps -L`)                    |
| PR_SET_PDEATHSIG        | get this signal when parent dies                          |
| PR_SET_NO_NEW_PRIVS     | no setuid/file-cap escalation across exec                |
| PR_SET_DUMPABLE         | mark process as core-dumpable or not                      |
| PR_SET_SECCOMP          | install seccomp filter                                    |
| PR_CAP_AMBIENT          | manipulate ambient capability set                         |
| PR_SET_THP_DISABLE      | disable transparent hugepages for this task              |
| PR_SET_TIMERSLACK       | timer slack value (energy / latency tradeoff)            |
| PR_SET_VMA_ANON_NAME    | name an anon mmap (visible in `/proc/PID/maps`)          |

When you don't know where to put a process attribute, it's probably `prctl`.

### `auditd` — the kernel audit subsystem

The audit subsystem (CAP_AUDIT_WRITE, CAP_AUDIT_CONTROL) logs syscalls, file accesses, and capability uses to userspace via netlink. It's how SELinux and many compliance tools (PCI, FedRAMP) gather evidence.

```
$ sudo auditctl -a always,exit -F arch=b64 -S openat -F path=/etc/passwd -k passwd_watch
$ sudo ausearch -k passwd_watch
type=SYSCALL msg=audit(1714201234.567:891): arch=c000003e syscall=257 ...
```

### Where to look in `/proc` for everything

- `/proc/cpuinfo` — CPU model, features, flags (vmx, svm, sse4_2, ...)
- `/proc/meminfo` — memory totals & cache
- `/proc/interrupts` — IRQ counters per CPU
- `/proc/softirqs` — softirq counters
- `/proc/kallsyms` — kernel symbol table (read-only addresses with kptr_restrict)
- `/proc/cmdline` — kernel command-line at boot
- `/proc/PID/stat`, `status`, `cmdline`, `cwd`, `exe`, `fd/`, `maps`, `mountinfo`, `ns/`, `cgroup`, `status` — per-task introspection
- `/proc/sys/...` — runtime kernel tunables (`sysctl -a`)
- `/proc/net/...` — networking introspection (sockets, route, fib_trie)
- `/proc/pressure/{cpu,memory,io}` — PSI

You will spend an entire career in `/proc`. Get comfortable.

### Where to look in `/sys` for everything

- `/sys/fs/cgroup/...` — cgroup v2 tree
- `/sys/class/net/<iface>/` — per-NIC stats
- `/sys/class/block/<dev>/queue/...` — per-block-device queue tunables
- `/sys/kernel/debug/...` — debugfs (mount: `mount -t debugfs none /sys/kernel/debug`)
- `/sys/kernel/debug/tracing/` — ftrace
- `/sys/kernel/security/` — LSM and integrity attestation
- `/sys/firmware/dmi/` — SMBIOS data (motherboard, vendor)
- `/sys/devices/system/cpu/` — per-CPU online/offline, topology, vulnerabilities

`/sys` is structured per-device; `/proc` is structured per-process and per-subsystem. Together they expose nearly the whole kernel.

### A quick word on real-time

For sub-millisecond determinism (industrial robots, audio, finance), Linux needs the **PREEMPT_RT** patchset — now in mainline since 6.12. RT kernels:

- Convert most spinlocks into rt_mutex (which can sleep).
- Make all interrupt handlers threaded by default.
- Provide `SCHED_DEADLINE` / `SCHED_FIFO` / `SCHED_RR` policies.
- Lower max latency to single-digit microseconds (when tuned).

Most container workloads do *not* want PREEMPT_RT — it has throughput costs. But knowing it exists is part of growing up as a kernel observer.

### Memory protections you may not have heard of

- **NX (No-eXecute) / DEP**: pages marked non-executable. Modern CPUs enforce. Stack is NX since the early 2000s.
- **SMAP (Supervisor Mode Access Prevention)**: kernel cannot dereference user pointers without `stac`/`clac` clearance. Caught Spectre-V1 attempts.
- **SMEP (Supervisor Mode Execution Prevention)**: kernel cannot *jump to* user code. Mitigates ret2usr exploits.
- **KASLR**: kernel address space layout randomization. Makes ROP harder.
- **KPTI (Kernel Page-Table Isolation)**: separate page tables for user/kernel; mitigates Meltdown. Cost: ~5–30% syscall overhead pre-PCID.
- **PCID**: Process Context Identifier — TLB tagging that makes KPTI cheap.
- **L1TF, MDS, RIDL, ZombieLoad** mitigations: speculative-execution side-channel patches; check `/sys/devices/system/cpu/vulnerabilities/`.

```
$ ls /sys/devices/system/cpu/vulnerabilities/
itlb_multihit  l1tf  mds  meltdown  mmio_stale_data  retbleed  spec_store_bypass  spectre_v1  spectre_v2  srbds  tsx_async_abort
$ for f in /sys/devices/system/cpu/vulnerabilities/*; do echo "$(basename $f): $(cat $f)"; done
spectre_v1: Mitigation: usercopy/swapgs barriers
spectre_v2: Mitigation: Retpolines, IBPB conditional, RSB filling
meltdown:   Mitigation: PTI
...
```

### A short tour of `dmesg`

`dmesg` prints the kernel ring buffer. Selected entries you'll see:

- `Linux version ...` — kernel version, gcc, build options.
- `Memory: ...` — boot-time memory map.
- `ACPI:` — firmware tables (interrupt routing, NUMA topology).
- `Out of memory: Killed process X (comm)` — OOM killer fired.
- `traps: foo[123] general protection fault ip:... sp:...` — userspace SIGSEGV.
- `tcp_collapse: ...` — TCP buffer pressure.
- `nf_conntrack: table full, dropping packet` — conntrack overflow.
- `Bridge firewalling registered` — netfilter on bridges.
- `cfg80211: World regulatory domain updated` — wifi regulatory.

`dmesg --follow --color=always` is one of the most useful commands in a sysadmin's toolbox.

### The `keyctl` and kernel keyrings

Often forgotten: the kernel maintains a **key management** subsystem (`man 7 keyrings`). Stores secrets, kerberos tickets, signed blobs. Used by NFS, ecryptfs, dm-crypt for unattended unlock. `keyctl` is the CLI. Capabilities `CAP_SYS_KEYRING` / `CAP_AUDIT_*` apply.

### The `bpftool` cheat reference

```
bpftool prog list                          # show loaded programs
bpftool prog show id 17                    # detail one
bpftool prog dump xlated id 17             # eBPF bytecode disassembly
bpftool prog dump jited id 17              # JIT'd machine code disassembly
bpftool map list                           # all maps
bpftool map dump id 5                      # dump a map's contents
bpftool map update id 5 key hex 00 00 00 01 value hex 00 00 00 00 00 00 00 2a
bpftool prog load file.o /sys/fs/bpf/myprog
bpftool net list                           # XDP / tc programs by interface
bpftool cgroup tree                        # cgroup-attached programs
bpftool feature                            # what this kernel supports
bpftool btf list                           # loaded BTF blobs
bpftool gen skeleton file.o > file.skel.h  # generate libbpf skeleton
```

Skeletons are the modern way to write eBPF user-side glue: instead of opening files and looking up programs by name, you `#include "file.skel.h"` and call generated functions.

### Tracepoints worth knowing

```
syscalls:sys_enter_*           # any syscall entry
syscalls:sys_exit_*            # any syscall return
sched:sched_switch             # context switch (off-CPU profiling)
sched:sched_process_exec       # exec
sched:sched_process_exit       # exit
sched:sched_wakeup             # wakeup
block:block_rq_issue           # disk I/O issued
block:block_rq_complete        # I/O completed
net:netif_receive_skb          # packet received
net:net_dev_queue              # packet queued for tx
syscalls:sys_enter_openat      # file opens
power:cpu_frequency            # frequency changes
```

`bpftrace -l 'tracepoint:*'` lists them all (~3000 in modern kernels).

### Common eBPF helpers

A program calls **helpers** to interact with the kernel safely. Helpers are listed in `man 7 bpf-helpers`. Selected favorites:

- `bpf_get_current_pid_tgid()` — combined TGID:TID
- `bpf_get_current_uid_gid()` — combined UID:GID
- `bpf_get_current_comm(&buf, sizeof(buf))` — process name
- `bpf_ktime_get_ns()` — monotonic clock in ns
- `bpf_map_lookup_elem` / `update_elem` / `delete_elem`
- `bpf_perf_event_output` / `bpf_ringbuf_output` — to userspace
- `bpf_trace_printk` — debug printk to `/sys/kernel/debug/tracing/trace_pipe`
- `bpf_probe_read_user` / `bpf_probe_read_kernel` — safe read
- `bpf_get_stackid` — capture a stack trace
- `bpf_redirect_map` — XDP redirection

Each helper has a verifier-defined signature and capability requirement.

### Watching context switches

```
# bpftrace -e 'tracepoint:sched:sched_switch { @[comm, args->next_comm] = count(); }'
Attaching 1 probe...
^C

@[swapper/0, kworker/0:1]: 12345
@[bash, sshd]: 891
@[Xorg, Compositor]: 7234
```

This is how you find out what's actually keeping your CPU busy — at sub-microsecond resolution, without slowing the system.

### Off-CPU profiling

Most profilers tell you where the CPU is *running*. Off-CPU profiling tells you where threads are *blocked* (sleeping, waiting on I/O, contending on a lock). With eBPF and `sched_switch`:

```
# bpftrace -e '
  tracepoint:sched:sched_switch { @start[args->prev_pid] = nsecs; }
  tracepoint:sched:sched_wakeup /@start[args->pid]/ {
      @offcpu[comm] = sum(nsecs - @start[args->pid]); delete(@start[args->pid]);
  }
'
```

Output is microseconds spent off-CPU per process. Invaluable for latency debugging.

### How `getpid()` got faster

For a long time `getpid()` had a per-process cache in glibc — the assumption being you only get a new PID when you fork. Then NPTL threads broke that assumption. Then `clone3` and PID NS broke it again. As of glibc 2.25, `getpid()` always does a real syscall — but that syscall is one of the cheapest, and on some kernels lives in the vDSO.

Knowing what's a real syscall vs a vDSO call vs a glibc cache is part of mastery.

### Reading `kallsyms`

`/proc/kallsyms` is the kernel's public symbol table:

```
$ sudo cat /proc/kallsyms | grep ' T do_sys_openat2'
ffffffff812ac3a0 T do_sys_openat2
```

`T` = text (function), uppercase = global. Useful for: writing kprobes against the right symbol, decoding crash addresses, sanity-checking what's actually compiled in.

`kptr_restrict=2` (default) zeros the addresses to non-root. Set 0 (insecure) if you're debugging.

### `ftrace` — the older sibling

eBPF is sexy; `ftrace` is silently older and often simpler. To trace one function:

```
# echo do_sys_openat2 > /sys/kernel/debug/tracing/set_ftrace_filter
# echo function > /sys/kernel/debug/tracing/current_tracer
# cat /sys/kernel/debug/tracing/trace_pipe
            <...>-1234   [001] ...   12345.678: do_sys_openat2 <-__x64_sys_openat
```

`trace-cmd` is the friendlier frontend. `perf trace` is the eBPF-aware equivalent.

### `perf` — the profiler that ships with the kernel

```
# perf top                       # live profile, like htop but for code
# perf record -ag -- sleep 10    # 10 second profile, all CPUs
# perf report                    # interactive viewer
# perf trace -- ls               # syscall trace via eBPF (newer perf)
# perf stat -- ./benchmark       # PMU counters: cycles, instructions, ...
```

`perf` and eBPF are siblings — `perf` uses many of the same kernel facilities (perf_event_open, kprobes, tracepoints). For "what's the program doing", reach for `perf`. For "what's the kernel doing", reach for `bpftrace`.

### The `audit` syscall numbers

You'll see kernel logs use `arch=c000003e` or `arch=40000028` — those are `AUDIT_ARCH_X86_64` and `AUDIT_ARCH_AARCH64`. Decode with `ausyscall --dump`.

### KASAN, KCSAN, UBSAN, KFENCE

Kernel sanitizers — compile-time/runtime memory bug catchers used in CI and fuzz farms:

- **KASAN**: KernelAddressSanitizer — catches use-after-free, OOB reads/writes.
- **KCSAN**: KernelConcurrencySanitizer — catches data races.
- **UBSAN**: UndefinedBehaviorSanitizer — catches int overflow, shift-by-too-much, etc.
- **KFENCE**: lightweight, production-safe page-guard sampler.

Distros sometimes ship debug kernels with these enabled. If you're getting weird KASAN reports in `dmesg`, you're running such a kernel.

### `lsns` — list namespaces directly

```
$ lsns
        NS TYPE   NPROCS   PID USER             COMMAND
4026531834 time      203     1 root             /sbin/init
4026531835 cgroup    203     1 root             /sbin/init
4026531836 pid       203     1 root             /sbin/init
4026531837 user      203     1 root             /sbin/init
4026531838 uts       203     1 root             /sbin/init
4026531839 ipc       203     1 root             /sbin/init
4026531840 mnt       195     1 root             /sbin/init
4026531992 net       203     1 root             /sbin/init
4026532563 mnt         1   234 systemd-resolve  /lib/systemd/systemd-resolved
4026532618 mnt         1   245 systemd-timesyn  /lib/systemd/systemd-timesyncd
```

systemd's per-service `PrivateTmp=` and friends create per-service mount namespaces — visible here.

### Filesystems you never think about that matter

- **proc** — `/proc`, kernel state.
- **sysfs** — `/sys`, device hierarchy.
- **debugfs** — `/sys/kernel/debug`, kernel-internal debug knobs.
- **tracefs** — `/sys/kernel/debug/tracing`, ftrace + tracepoints.
- **bpffs** — `/sys/fs/bpf`, persistent eBPF objects.
- **cgroup2** — `/sys/fs/cgroup`, the cgroup tree.
- **tmpfs** — RAM-backed, plus shmfs, devshm, pseudoroot for chroot.
- **mqueue** — `/dev/mqueue`, POSIX message queues.

You can `cat` and `echo` your way through nearly every kernel feature via these.

### How systemd uses what we just learned

Modern systemd is essentially a userspace control plane that orchestrates everything in this sheet:

- Each unit has a cgroup (`Slice=`, `MemoryMax=`, `CPUWeight=`).
- Each unit can have private namespaces (`PrivateTmp=`, `PrivateNetwork=`, `PrivateUsers=`).
- Capabilities are dropped via `CapabilityBoundingSet=` and `AmbientCapabilities=`.
- Seccomp via `SystemCallFilter=`.
- LSMs via `ReadOnlyPaths=`, `InaccessiblePaths=`.

Reading `systemd-analyze security <unit>` is the fastest way to see which knobs a service is using:

```
$ systemd-analyze security sshd
... overall exposure level: 5.7 (medium)
ProtectKernelTunables=yes               OK
ProtectControlGroups=no                 ! exposed
PrivateNetwork=no                       ! exposed
SystemCallFilter=                       ! exposed
...
```

### `runc` config in 30 seconds

A "container" in OCI terms is a tar (rootfs) + a JSON spec (`config.json`). The spec lists:

- `process.args` — what to run
- `process.env` — env vars
- `process.user` — UID/GID inside
- `process.capabilities` — bounding/effective/inheritable/ambient
- `process.rlimits` — RLIMIT_NOFILE etc.
- `linux.namespaces` — which to create
- `linux.uidMappings` / `gidMappings` — for USER NS
- `linux.resources.cgroup` — cpu/memory/io quotas
- `linux.seccomp` — full BPF seccomp profile
- `linux.maskedPaths` / `readonlyPaths` — paths hidden / read-only
- `mounts` — the namespaced mount table

`runc spec` generates a default; you can `runc run` it. The whole stack — Docker, containerd, podman, K8s — sits on top of this format.

### A vocabulary of "I see this in `ps` but what does it mean?"

```
$ ps -eLo pid,tid,stat,comm | head
  PID   TID STAT COMMAND
    1     1 Ss   systemd
    2     2 S    kthreadd
    3     3 I<   rcu_gp
    4     4 I<   rcu_par_gp
   12    12 I    kworker/0:0H-events_highpri
```

State letters:

- `R` running or runnable
- `S` interruptible sleep
- `D` uninterruptible sleep (the "D-state stuck" famous case)
- `T` stopped (SIGSTOP, ptrace)
- `t` traced
- `Z` zombie (waiting reap)
- `X` dead (briefly)
- `I` idle kernel thread

Modifiers:

- `<` high-priority
- `N` low-priority (nice)
- `s` session leader
- `+` foreground process group
- `l` multi-threaded
- `L` has locked pages

A "stuck" `D` for more than a few seconds means the kernel is waiting on hardware. Almost always disk or NFS. Often the disk is dead.

### Reading kernel CVE patches

Browse `kernel.org` `linux-stable` for `Patch_RT*` or look at distro security trackers (Red Hat CVE pages, Ubuntu USN). Most CVEs are:

- Use-after-free in some subsystem (eBPF verifier, netfilter, io_uring, vsock).
- Speculative execution side channels.
- USER NS abuse (mounting things you shouldn't).
- Refcount/race issues in fork/exec/exit paths.

The discipline of reading CVE patches monthly is what separates a hobbyist from a professional kernel observer.

### When to write a kernel module vs eBPF

| Task                                        | Kernel module | eBPF         |
|---------------------------------------------|---------------|--------------|
| Add a new syscall                           | yes           | no           |
| Add a new char/block device driver          | yes           | no           |
| Tracing existing functions                  | sure but bad  | YES          |
| Custom packet rewriting                     | possible      | YES (XDP/tc) |
| Monitor cgroup membership                   | possible      | YES          |
| Implement a security policy                 | possible      | YES (LSM)    |
| Modify arbitrary kernel data structures     | YES           | rarely       |

Default to eBPF. Resort to a module only when you must.

### A note on ABI stability

Linus's promise: **don't break userspace**. Syscall ABI is locked. Behavior is locked. But:

- Internal kernel APIs are *not* locked. Module authors live with churn.
- kprobes may move (function inlined, renamed, replaced).
- Tracepoint argument layouts have minor compat changes (BTF-aware tools handle it).
- Sysfs and procfs files are *mostly* stable; some churn at the deep end of debug stuff.

eBPF's CO-RE was specifically built so tracing tools could survive kernel upgrades.

### Glossary of kernel acronyms you'll keep meeting

- **DMA** — Direct Memory Access (devices to RAM without CPU help)
- **IOMMU** — IO Memory Management Unit (DMA virtualization)
- **MMIO** — Memory-Mapped I/O (treat device registers as RAM)
- **NUMA** — Non-Uniform Memory Access (per-socket memory)
- **PMU** — Performance Monitoring Unit (CPU counters)
- **TSC** — Time Stamp Counter
- **HPET** — High Precision Event Timer
- **RCU** — Read-Copy-Update (lock-free reads)
- **VFS** — Virtual File System (top of FS stack)
- **VMA** — Virtual Memory Area (a contiguous user mapping)
- **SLAB / SLUB / SLOB** — kernel object allocators
- **OOM** — Out Of Memory
- **THP** — Transparent Huge Pages
- **KSM** — Kernel Samepage Merging (dedup of identical anon pages)

### Final visual: process anatomy

```
                  struct task_struct (one per task)
   ┌──────────────────────────────────────────────────────────────────┐
   │ pid_t pid; pid_t tgid;          (TID and PID — the leader's TID) │
   │ task->comm[16]                  (process name, set by exec)       │
   │ task->state                      (R/S/D/T/Z)                       │
   │ task->cred → cred                (UID/GID/cap sets)                │
   │ task->mm → mm_struct → page tables, VMAs                          │
   │ task->files → files_struct → fd table                              │
   │ task->fs → fs_struct → cwd, root, umask                            │
   │ task->signal → signal_struct (per-process signal state)            │
   │ task->sighand → sighand_struct (per-thread-group handlers)         │
   │ task->nsproxy → nsproxy → MNT/PID/NET/UTS/IPC/CGROUP/TIME ns       │
   │   (USER ns lives on cred for UID-mapping reasons)                  │
   │ task->cgroups → css_set                                            │
   │ task->parent / real_parent / children / sibling                    │
   │ task->stack → kernel stack (typically 16K)                         │
   └──────────────────────────────────────────────────────────────────┘
```

A *process* is a task_struct with `pid == tgid` and at least one thread. A *thread* is another task_struct with the same `tgid` sharing `mm`, `files`, `fs`, `sighand`. From the kernel's view, threads are just tasks that happen to share a few resources.

That's the anatomy of "one running thing on Linux". Everything in this sheet is a means of inspecting, isolating, or limiting parts of that struct.

---

## Worked Scenarios

The point of this section: pick a real situation, walk it end-to-end, see how every piece we covered actually combines.

### Scenario 1 — Why is my container OOM-killed when `ps` says it has free memory?

A container with `memory.max=512M` is killed. Inside the container, `free -m` shows 200M used, 312M free. Inside, this looks insane. Outside?

```
# pid=$(docker inspect --format '{{.State.Pid}}' demo)
# cat /proc/$pid/cgroup
0::/system.slice/docker.service/docker/<id>
# cat /sys/fs/cgroup/system.slice/docker.service/docker/<id>/memory.current
536870912         # 512M, hit the cap
# cat /sys/fs/cgroup/system.slice/docker.service/docker/<id>/memory.stat
anon                40000000
file               300000000      # 300M of page cache
slab                15000000
# cat /sys/fs/cgroup/system.slice/docker.service/docker/<id>/memory.events
oom 1
oom_kill 1
```

The page cache counts. Inside the container, `free` shows it as "free-ish" (recoverable). The cgroup says: nope, this is your 512M, you're at it, OOM. **Lesson:** don't trust `free` inside containers. Trust `memory.current` and `memory.stat`. Container memory accounting includes file cache.

### Scenario 2 — A process is stuck in D and I can't kill it

```
$ ps aux | grep my_app
user 12345 ... D ... 1:23 my_app
$ sudo kill -9 12345          # has no effect
$ ps aux | grep my_app
user 12345 ... D ...          # still there
$ cat /proc/12345/wchan
io_schedule_timeout
$ cat /proc/12345/stack
[<0>] io_schedule_timeout+0xa0/0xb0
[<0>] folio_wait_bit_common+0x14b/0x350
[<0>] folio_wait_writeback+0x35/0x70
[<0>] write_cache_pages+0x195/0x3f0
[<0>] ext4_writepages+0x4ad/0x6d0
[<0>] do_writepages+0xd9/0x180
[<0>] filemap_fdatawrite_wbc+0x80/0xc0
[<0>] file_write_and_wait_range+0x71/0xd0
[<0>] ext4_sync_file+0xa6/0x500
[<0>] vfs_fsync_range+0x47/0xa0
[<0>] do_fsync+0x42/0x90
[<0>] __x64_sys_fdatasync+0x1a/0x20
```

D-state. Kernel is waiting on `ext4_writepages` → block layer → likely a stuck disk. SIGKILL is queued; will be delivered when the syscall returns. Until then, you can't kill it.

Diagnosis path: `dmesg | tail` for I/O errors, `iostat -x 1` for disk health, `smartctl -a /dev/sdX` for SMART status. If the disk is genuinely broken, your only options are reboot or hot-removing the drive (often crashes the system). This is the "uninterruptible" in "uninterruptible sleep".

### Scenario 3 — A binary inside a container is denied a syscall it needs

```
# strace -f /usr/bin/ping 1.1.1.1
...
prctl(PR_CAPBSET_READ, CAP_NET_RAW)     = 1
socket(AF_INET, SOCK_RAW, IPPROTO_ICMP) = -1 EPERM (Operation not permitted)
```

EPERM. Three possible culprits:

1. Capabilities — drop list includes CAP_NET_RAW.
2. seccomp — filter blocks `socket(AF_INET, SOCK_RAW, ...)`.
3. AppArmor / SELinux — denies the operation by label/path.

```
$ docker inspect --format '{{.HostConfig.CapDrop}}' demo
# []         <- not capability
$ cat /proc/$pid/status | grep ^Seccomp
Seccomp: 2                                # filter installed
```

Seccomp = 2 means a filter is active. To debug:

```
# bpftrace -e 'tracepoint:syscalls:sys_enter_socket { printf("%d %d %d\n", args->family, args->type, args->protocol); }'
```

Watch what your container actually calls; cross-reference with the seccomp profile. The fix is one of: `--cap-add=NET_RAW`, `--security-opt seccomp=unconfined`, or write a more permissive profile.

### Scenario 4 — Network namespace says "no route to host" but the host has one

You unshared a netns, brought up loopback, but cannot reach 8.8.8.8. Why?

The new netns has only `lo`. There's no veth, no IP, no default route. Networking is *namespaced* — your host's `eth0` is a stranger.

Recipe for "give my netns internet":

```
# pid=$(pidof unshare-target)
# ip link add v0 type veth peer name v1
# ip link set v1 netns $pid
# ip addr add 192.168.99.1/24 dev v0; ip link set v0 up
# nsenter -t $pid -n ip addr add 192.168.99.2/24 dev v1
# nsenter -t $pid -n ip link set v1 up
# nsenter -t $pid -n ip link set lo up
# nsenter -t $pid -n ip route add default via 192.168.99.1
# echo 1 > /proc/sys/net/ipv4/ip_forward
# iptables -t nat -A POSTROUTING -s 192.168.99.0/24 -j MASQUERADE
```

Now the netns can reach the world. Docker's `--network=bridge` is a productionized version of exactly this.

### Scenario 5 — eBPF program rejected with `R3 invalid mem access 'inv'`

You wrote:

```c
SEC("kprobe/something")
int kp(struct pt_regs *ctx) {
    char *p = (char *)PT_REGS_PARM1(ctx);
    char buf[16];
    bpf_probe_read_kernel(buf, 16, p);   // OK
    char c = p[10];                       // BAD — direct deref of kernel ptr
    return 0;
}
```

The verifier rejects the second line: you tried to dereference a "scalar" (verifier's term for arbitrary integer) as a pointer. Inside eBPF, you cannot dereference user or kernel addresses directly; you must use `bpf_probe_read_kernel` / `bpf_probe_read_user`. The verifier knows the difference between a "checked pointer" and a "raw integer" and demands proof.

The fix: only access memory through `bpf_probe_read_*` helpers (or modern type-safe equivalents like CO-RE field reads via `BPF_CORE_READ`).

### Scenario 6 — Container processes can read host /proc and that's bad

```
# docker run --rm -v /proc:/host/proc alpine cat /host/proc/sys/kernel/hostname
```

This works. The host bind-mount overrides namespace isolation. Lesson: don't `--volume /proc`. The whole point of MNT NS + PID NS is to give the container its own `/proc` (mounted via `mount -t proc proc /proc` *inside* its mount NS). Bypassing that is opt-in security suicide.

Modern hardening: AppArmor profile `docker-default` denies many proc paths even via mounts.

### Scenario 7 — Why do `kprobe:sys_open` traces no longer fire?

You wrote `kprobe:sys_open`, expecting it to catch all opens. It returns nothing. Why?

`sys_open` was inlined into the syscall entry path on x86_64 a few kernel releases ago. The function name still appears in `/proc/kallsyms` as `__x64_sys_open` (with arch prefix), but it is now a thin wrapper around `do_sys_openat2`. Modern code uses `openat`/`openat2` exclusively for new opens.

Fixes:

1. Use a tracepoint: `tracepoint:syscalls:sys_enter_openat`. Stable across kernels.
2. Use `kprobe:do_sys_openat2`. Real implementation.
3. Use `fentry:do_sys_openat2`. Faster, verified context.

Tracepoints > kprobes when one exists.

### Scenario 8 — Capabilities seem to do nothing

You tried:

```
$ sudo setcap cap_net_bind_service+p /usr/local/bin/myserver
$ /usr/local/bin/myserver       # tries to bind 80
bind: Permission denied
```

Capabilities have *five* sets. `+p` only adds Permitted. To use a capability across exec, it must end up in **Effective** or **Ambient** post-exec. Try:

```
$ sudo setcap cap_net_bind_service+ep /usr/local/bin/myserver
```

`+e` = effective. The binary can now use the cap immediately on launch. If you also need the cap to survive a child exec, add ambient via `prctl(PR_CAP_AMBIENT_RAISE, ...)`. Most server use cases just need `+ep` on the binary.

### Scenario 9 — A user namespace gives me "root" but `chown` fails

```
$ unshare --user --map-root-user
# touch /tmp/foo
# chown 1000 /tmp/foo
chown: changing ownership of '/tmp/foo': Invalid argument
```

You are "root" inside this user NS, but the kernel's UID 1000 is *outside* your idmap. `chown` to an unmapped UID = EINVAL. Inside, you can chown to mapped UIDs. With `newuidmap` you can map subuid ranges; without it, only `0 -> your-real-UID` is mapped.

```
# cat /proc/$$/uid_map
         0       1000          1
```

One mapping: NS UID 0 → host UID 1000, range 1 (just one UID). chown to NS UID 0 works (mapped). chown to NS UID 1 fails (out of range).

### Scenario 10 — Profiling a Postgres workload with `bpftrace`

Goal: histogram of query latency, broken down by client IP, without modifying Postgres source.

Postgres ships USDT (statically defined tracepoints). They live in libpq and the server binary as `usdt:postgres:query__start` / `query__done`.

```
# bpftrace -e '
    usdt:/usr/lib/postgresql/15/bin/postgres:postgres:query__start { @s[tid] = nsecs; }
    usdt:/usr/lib/postgresql/15/bin/postgres:postgres:query__done /@s[tid]/ {
        @lat = hist(nsecs - @s[tid]); delete(@s[tid]);
    }
' -c 'sleep 60'
```

Output: a power-of-two histogram of query latencies in nanoseconds. Combine with `bpf_get_socket_cookie()` for source-IP breakdown.

This kind of question — "tell me query latencies under load" — used to require Postgres recompiles, log parsing, or expensive APM agents. eBPF + USDT does it in 5 lines, in production, with zero overhead when not running.

---

## Where to Go Next

- `cs ramp-up linux-kernel-college` — buddy allocator, SLUB, page cache, NUMA, RCU, advanced eBPF
- `cs fundamentals linux-kernel-internals` — dense reference for the working sysadmin
- `cs fundamentals ebpf-bytecode` — the actual instruction set, verifier rules, helper functions
- `cs kernel-tuning cgroups` — every cgroup v2 file, weight/quota tuning, memory.high economics
- `cs kernel-tuning namespaces` — by-namespace recipes (rootless containers, nsenter cookbook)
- `cs kernel-tuning ebpf` — production eBPF: CO-RE, libbpf-bootstrap, deployment
- `cs containers docker` — when you're ready to use the high-level tools
- `cs performance perf` — profiling with perf_event_open
- `cs performance bpftrace` — DTrace-style observability
- `cs system strace` — when you need ptrace-based syscall tracing anyway
- `cs system gdb` — when you need debugging at the assembly level

---

## See Also

- `ramp-up/linux-kernel-middle-school` — the previous tier
- `ramp-up/linux-kernel-college` — the next tier
- `fundamentals/linux-kernel-internals` — dense reference
- `fundamentals/ebpf-bytecode` — eBPF deep dive
- `kernel-tuning/cgroups` — cgroup tuning
- `kernel-tuning/namespaces` — namespace tuning
- `kernel-tuning/ebpf` — production eBPF
- `containers/docker` — container runtime
- `system/strace` — ptrace tracing
- `system/gdb` — assembly-level debug
- `performance/perf` — perf_event_open
- `performance/bpftrace` — eBPF one-liners

---

## References

- `man 7 capabilities` — every capability and what it does
- `man 7 namespaces` — overview of all eight
- `man 7 cgroups` — v1/v2 architecture
- `man 2 unshare` — detach into new namespaces
- `man 2 setns` — join an existing namespace
- `man 2 prctl` — many process attributes including no_new_privs
- `man 2 bpf` — the BPF syscall and all its commands
- `man 2 clone` / `clone3` — process/thread/namespace creation
- `man 2 sigaction` — signal handlers and SA_* flags
- `man 7 signal` and `man 7 signal-safety` — signal semantics, async-safe list
- kernel.org `Documentation/userspace-api/` — the official user-facing kernel docs
- kernel.org `Documentation/admin-guide/cgroup-v2.rst` — the cgroup v2 spec
- kernel.org `Documentation/bpf/` — eBPF subsystem docs
- "BPF Performance Tools" — Brendan Gregg, 2019; the canonical eBPF practitioner book
- "Linux Kernel Development", 3rd ed. — Robert Love; the friendliest kernel intro book
- "Understanding the Linux Kernel", 3rd ed. — Bovet & Cesati; deep reference (older but still useful for IDT/APIC/syscall mechanics)
- LWN.net cgroup v2 article series — Tejun Heo's design notes
- LWN.net eBPF article series — long-running, definitive coverage
- ebpf.io — landing page with tutorials, books, talks
- Intel SDM Vol 3A — the actual interrupt and trap specs
- ARM ARM (Architecture Reference Manual) — exception level model, SVC instruction
- Documentation/x86/entry_64.S walk-through (Lameter, Wieland) — annotated syscall path
- "What every systems programmer should know about concurrency" — Sutter (talk; relevant background)
- iovisor/bcc tools collection — `opensnoop`, `execsnoop`, `runqlat`, `tcpconnect`, dozens more
- Cilium docs — the gold standard production eBPF networking story
- man pages: `bpftrace(8)`, `bpftool(8)`, `perf(1)`, `nsenter(1)`, `unshare(1)`, `capsh(1)`, `getcap(1)`, `setcap(1)`, `lsns(1)`
