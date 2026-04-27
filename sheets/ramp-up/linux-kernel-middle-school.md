# Linux Kernel — Middle School (Part 2 of 4 — Ramp-Up Curriculum)

> The kernel as a real operating system: how your machine boots, how processes are born, how memory pretends to be infinite, how the CPU is shared fairly, how every file looks the same, and how packets travel through the stack.

## Prerequisites

- `cs ramp-up linux-kernel-eli5` — the prior tier (the "kernel = manager" mental model)
- A terminal (any modern Linux distro: Ubuntu, Fedora, Arch, Debian, Alpine, Void — they all share the same kernel)
- About 60 minutes. Bring snacks.

If you have not yet read the ELI5 sheet, please do that first. This page assumes you already accept the basic idea that the kernel is the program that manages everything. Now we want to know **how**.

## Plain English

Imagine a school. In Part 1 we said the kernel is the principal — the person who shares the building, protects the lockers, talks to the buses, organizes the lockers themselves, and watches the doors. Good. That metaphor still works. But a real school is more interesting than that, because it has a morning routine, a class schedule, a way to handle a flood of new students, a system for what to do when somebody is in the bathroom, and a procedure for replacing a broken vending machine without sending everyone home.

This sheet is about all of those mechanisms. We are going to give the metaphors real names.

The kernel is not a single program that runs once and then sits there. It is a set of subsystems that all run in a special protected mode of the CPU, called **kernel mode** (also called **ring 0** on x86-64). When you double-click a program, you are not directly making the CPU do anything; you are asking the kernel, very politely, to do something on your behalf. The way you ask is by calling a **system call** (or **syscall** for short). A syscall is the doorway between your program — running in **user mode** (ring 3) — and the kernel. Examples of syscalls: `read`, `write`, `open`, `close`, `fork`, `execve`, `mmap`, `socket`. There are about 350 of them on Linux. They are documented in section 2 of the manual, so `man 2 fork` actually opens the kernel-side documentation for the `fork` syscall.

Now to the parts.

When you press the power button, your computer runs **firmware**: code that lives on a small chip on the motherboard. On older machines this was called the **BIOS** (Basic Input/Output System); on modern machines it is **UEFI** (Unified Extensible Firmware Interface). The firmware does a quick **POST** (Power-On Self-Test) — it counts the RAM, checks the keyboard, asks the disks to introduce themselves — and then it loads a **bootloader**. On most Linux systems the bootloader is **GRUB** (GRand Unified Bootloader). GRUB knows where the kernel image is on disk (typically `/boot/vmlinuz-...`) and it loads that into RAM, plus a small companion archive called the **initramfs** (initial RAM filesystem) which contains just enough drivers to find your real disks. Then GRUB jumps to the kernel's entry point and the kernel takes over. The kernel detects all of your hardware, loads the **device drivers** it needs, mounts your **root filesystem** (the one mounted at `/`), and finally starts the very first user-space program, with **PID 1**. On modern Linux this is almost always **systemd**. Every other process on the system is a descendant of systemd.

Once the system is up, the most important thing the kernel does, all day, every day, is run **processes**. A process is a running program. Every process has a numeric **PID** (process ID) and a **PPID** (parent process ID), because every process was created by some other process. (Except PID 1, which the kernel creates directly during boot.) New processes are born when an existing process calls **`fork()`**. Fork makes a near-identical copy of the calling process — same code, same memory, same open files — and the only thing different is the return value: the parent gets the child's PID back, and the child gets `0` back. That is how each one knows which one it is. After the fork, the child usually immediately calls **`execve()`**, which throws away the copied program and replaces it with a different one. That is how `bash` starts `vim`: bash forks (now there are two bashes), the child execves into vim. The end.

Why do it in two steps? Because between the fork and the exec, the child can change things — close some files, open others, set environment variables, change the user ID — without disturbing the parent. That is exactly how shell redirection (`cmd > file.txt`) is implemented. Beautiful.

Once a process exists, it is in one of four main **states**: running (actually on a CPU), runnable (ready to run, waiting for a CPU), sleeping (waiting for something — disk, network, a timer), or stopped (paused by a signal). When a process finishes (calls `exit()`), it does not immediately vanish; it sticks around as a **zombie** until its parent calls `wait()` to read the exit code. Yes, the kernel really uses the word "zombie."

Each process gets its own **virtual address space**. From the program's perspective it looks like it has the entire machine to itself, with memory addresses ranging from 0 up to a huge number (on 64-bit, 2^48 bytes ≈ 256 TB of address space). This is a fiction. Behind the scenes, the kernel and the **MMU** (Memory Management Unit, a piece of the CPU) translate every virtual address the program uses into a real **physical address** in actual RAM. The translation goes through a data structure called a **page table**. Memory is split into **pages** of 4 KB each (sometimes 2 MB or 1 GB for "huge pages"). The page table is, in effect, a giant lookup table: "virtual page 17 → physical page 9314." When a process touches a virtual page that is not currently mapped, the CPU traps, the kernel runs a **page fault handler**, the kernel either loads the page from disk (a swap file or memory-mapped file) or allocates a fresh zero-filled page, fills in the page table, and resumes the process — which has no idea anything happened. This is how programs can use more memory than is physically installed: cold pages are pushed out to **swap** (a disk area), and brought back when touched. The MMU also has a tiny cache called the **TLB** (Translation Lookaside Buffer) that remembers recent translations so the lookup is fast.

The CPU is shared across all the runnable processes by the **scheduler**. Linux's main scheduler is called **CFS** — the **Completely Fair Scheduler**. The idea is gorgeous in its simplicity: each process accumulates a number called **vruntime** (virtual runtime) every time it runs. The scheduler keeps all runnable processes in a **red-black tree** sorted by vruntime, and whenever a CPU is free it picks the process with the smallest vruntime — that is, the one that has used the least CPU so far. That is what "fair" means. Then it lets that process run for a little while, increases its vruntime by the time it ran, and re-inserts it into the tree. Heavy CPU users naturally drift to the right of the tree (large vruntime); polite processes that mostly sleep stay on the left (small vruntime), so when they do wake up they get the CPU immediately. Niceness — the **`nice`** value, ranging from `-20` (greedy) to `+19` (polite) — is implemented as a multiplier on how fast vruntime grows. A nice process accumulates vruntime quickly so it gets scheduled less; a not-nice process accumulates slowly so it gets scheduled more. (Real-time policies — `SCHED_FIFO`, `SCHED_RR` — bypass CFS entirely; we cover those in the high-school sheet.)

Files are managed through a layer called the **VFS** (Virtual File System). The VFS is one of the great Linux ideas: every filesystem — `ext4`, `xfs`, `btrfs`, `f2fs`, `vfat`, `ntfs3`, even network ones like `nfs` and `cifs` — speaks to the kernel through the same interface. So your program calls `open("/some/file")` and never has to know whether the file is on a local SSD, a USB stick, a CD, a tmpfs in RAM, or a server in another country. The VFS keeps track of files using **inodes** (the actual on-disk metadata: size, permissions, owner, block pointers — but **not the name**), **dentries** (directory entries — they map names to inodes and form the directory tree), and **file structs** (in-memory state of an open file: position, mode, flags). Multiple names can point to the same inode (that is what a **hard link** is); one name can be a pointer to another name (a **symlink**).

Linux extends the file abstraction beyond just disk files: devices appear as files in `/dev` (so reading a microphone is just `read()` on `/dev/snd/...`), kernel state appears as files in `/proc` (one directory per running process, plus system-wide info) and `/sys` (the modern interface for kernel objects, drivers, hardware). This is the famous **"everything is a file"** principle. It is the reason `cat`, `grep`, `tee` and friends compose so beautifully.

Networking is layered. Packets going **out** from your browser walk down the stack: the **application layer** (HTTP, your program calls `send()`); the **transport layer** (TCP or UDP, where your data is broken into segments, sequence numbers are added, and retransmission is handled); the **network layer** (IP, where source and destination IP addresses are added and the routing table is consulted); the **link layer** (Ethernet/Wi-Fi, where a MAC address frame is built and the device driver pushes it to the network card); and finally the wire. Incoming packets walk back up the same ladder. The kernel handles the transport, network, and most of the link layer; the device driver handles the bridge to the hardware.

Finally — and this is the magic that makes Linux feel infinitely flexible — most of the kernel's drivers and features are not built into the running kernel as one giant blob. They live as **kernel modules**, files with the extension `.ko` (kernel object). You can load and unload them at runtime using `modprobe`, `insmod`, and `rmmod`. When you plug in a USB device, **udev** detects the new hardware, asks the kernel to load the right module, and the device just works.

That is the middle-school view. Now let's open the hood.

## Concepts in Detail

### The Boot Process (UEFI/BIOS → POST → GRUB → kernel → initramfs → systemd)

The boot process is a relay race. Each runner only knows enough to start the next one and then pass the baton. Here is the detailed sequence:

```
Power button pressed
        |
        v
+--------------------+
| 1. CPU reset       |  CPU starts in 16-bit real mode (legacy) or
|    vector          |  in 64-bit long mode (modern UEFI). Program
|                    |  counter jumps to a fixed firmware address.
+--------------------+
        |
        v
+--------------------+
| 2. Firmware        |  UEFI (or BIOS on old machines) runs.
|    (UEFI / BIOS)   |  Stored on a flash chip on the motherboard.
+--------------------+
        |
        v
+--------------------+
| 3. POST            |  Power-On Self-Test: count RAM, identify CPU,
|    (Power-On       |  enumerate PCI/PCIe devices, check disks.
|     Self-Test)     |  If a critical thing is broken, beeps occur.
+--------------------+
        |
        v
+--------------------+
| 4. Find boot       |  UEFI reads the EFI System Partition (ESP),
|    target          |  finds an .efi binary listed in NVRAM boot
|                    |  entries. BIOS reads MBR sector 0.
+--------------------+
        |
        v
+--------------------+
| 5. Bootloader      |  GRUB (or systemd-boot, rEFInd, LILO).
|    (GRUB)          |  Shows the boot menu. Loads kernel +
|                    |  initramfs into RAM. Builds the cmdline.
+--------------------+
        |
        v
+--------------------+
| 6. Kernel          |  Kernel decompresses itself, sets up its own
|    init            |  page tables, enumerates CPUs (SMP boot),
|                    |  initializes drivers, mounts the initramfs
|                    |  as a temporary root filesystem (/).
+--------------------+
        |
        v
+--------------------+
| 7. initramfs       |  A tiny in-RAM root with just enough drivers
|    runs            |  to find the *real* root: LVM, LUKS unlock,
|                    |  multipath, network mount, etc. Then pivots.
+--------------------+
        |
        v
+--------------------+
| 8. switch_root     |  Real root is mounted on top, initramfs is
|                    |  freed. The very first process — PID 1 —
|                    |  is exec'd.
+--------------------+
        |
        v
+--------------------+
| 9. systemd (PID 1) |  Reads its target tree (default.target →
|                    |  multi-user.target → ...). Starts services
|                    |  in dependency order (in parallel where
|                    |  possible). Eventually you see a login
|                    |  prompt or a graphical greeter.
+--------------------+
```

The single most useful command for understanding boot is `dmesg`, which prints the kernel ring buffer — every message the kernel emitted from boot onward. `journalctl -b 0` does the same and more, including all of systemd's startup messages.

Version note: UEFI Secure Boot was added to Linux around 2012 (kernel 3.x). Modern distros sign their kernel and shim binaries with keys whose certificates are loaded into UEFI firmware, so the chain of trust extends from firmware to kernel. The `mokutil` command manages your machine's trusted keys.

### Processes: fork() and exec()

The combination of `fork()` and `exec()` is, in a real sense, the single most important idea in UNIX. Almost every program you run was started this way.

```
                       BEFORE fork()
                       =============
                       Process A (PID 100)
                       +-------------------+
                       | code: bash        |
                       | heap, stack       |
                       | open files: 0,1,2 |
                       +-------------------+


                       AFTER fork()
                       ============
       Process A (PID 100)            Process B (PID 101)
       +-------------------+          +-------------------+
       | code: bash        |          | code: bash        |
       | heap, stack (COW) |          | heap, stack (COW) |
       | open files: 0,1,2 |          | open files: 0,1,2 |
       +-------------------+          +-------------------+
       fork() returned 101            fork() returned 0
       (the child's PID)              (we are the child)


                       AFTER exec("/usr/bin/vim")
                       ==========================
       Process A (PID 100)            Process B (PID 101)
       +-------------------+          +-------------------+
       | code: bash        |          | code: vim         |
       | heap, stack       |          | heap, stack (new) |
       | open files: 0,1,2 |          | open files: 0,1,2 |
       +-------------------+          +-------------------+
       waiting for child              running vim
```

A few things to notice. The child inherits open file descriptors from the parent. That is why your shell can do `cmd > file.txt` — it forks, the child opens `file.txt` and dup2's it onto fd 1 (stdout), then execs the command. The new program writes to fd 1 like normal, never knowing it's a file rather than a terminal.

Also notice the **(COW)** marker — copy-on-write. Linux does not actually copy the parent's pages at fork time. It marks them read-only in both processes' page tables, and if either process writes to a page, the kernel allocates a fresh copy on the spot and updates that one process's page table. This makes fork extremely cheap, even for processes with gigabytes of memory. (`vfork()` was an older, less-safe optimization; modern kernels rarely need it because COW is so good.)

Modern Linux actually implements both `fork()` and `pthread_create()` on top of a single primitive syscall called `clone()`, with flags that decide which resources are shared (memory, file descriptors, signal handlers, namespaces). `fork()` ≈ `clone()` with no sharing; `pthread_create()` ≈ `clone()` sharing the address space.

Version note: `clone3()` (since Linux 5.3) replaced the old `clone()` with a struct-based interface that is easier to extend.

### Process State Machine

```
                          fork()
                +---------------------+
                |                     |
                v                     |
            +--------+ runnable   +--------+
   start -> | NEW    |----------->| READY  |<---+
            +--------+            +--------+    |
                                      |         | preempted /
                                      |         | timeslice over
                                      | sched   |
                                      v         |
                                  +--------+    |
                       I/O done   | RUNNING|----+
                  +-------------->|        |
                  |               +--------+
                  |                   | wait for I/O,
                  |                   | sleep, lock, etc.
                  |                   v
                  |               +--------+
                  +---------------|SLEEPING|
                                  +--------+
                                      |
                                      | (parallel branch)
                                      | signal: SIGSTOP
                                      v
                                  +--------+
                                  | STOPPED|---> SIGCONT --> READY
                                  +--------+

   exit() ----+
              v
          +--------+ parent calls wait()
          | ZOMBIE |---------------------> reaped (gone)
          +--------+
```

`ps` shows process state in the `STAT` (or `S`) column with single-letter codes: `R` running/runnable, `S` interruptible sleep, `D` uninterruptible sleep (almost always disk I/O — the dreaded "D state" that survives `kill -9`), `T` stopped, `Z` zombie, `I` idle kernel thread, plus modifiers like `+` (foreground), `s` (session leader), `<` (high priority), `N` (low priority).

A **zombie** is a finished process whose exit status has not been read by its parent. Zombies use no CPU or memory (they're just an entry in the process table) but if you have thousands you can run out of PIDs. Fix: the parent should call `wait()`/`waitpid()`. If the parent dies first, the zombie's parent is reparented to PID 1 (init/systemd), which routinely calls `wait()` on its children, so accidental orphan zombies disappear quickly.

An **orphan** is the opposite: a still-running process whose parent died. Orphans are also reparented to PID 1, which does not kill them; they keep running normally.

### Virtual Memory and Page Tables

Every process gets its own **virtual address space**. The kernel and the **MMU** (Memory Management Unit, hardware on the CPU) together translate virtual addresses to physical addresses on every memory access.

```
       Process's view:                          Reality:
       Virtual Address                          Physical RAM
       Space (per-process)                      (shared between all)

       0xFFFF...                                +-------------+
       +----------------+                       | Page #N     |
       | kernel space   |  (mapped same in     +-------------+
       | (only kernel   |   every process,      | Page #N-1   |
       |  can read)     |   for syscalls)       +-------------+
       +----------------+                       | ...         |
       |  ...           |                       +-------------+
       +----------------+                       | Page #2     |
       | stack          |---+                   +-------------+
       | (grows down)   |   |                   | Page #1     |
       +----------------+   |     page table    +-------------+
       |       ^        |   +-------+           | Page #0     |
       |       | (free) |           |           +-------------+
       |       v        |           |                  ^
       +----------------+           v                  |
       | heap           |    +------------+            |
       | (grows up)     |--->| 0xC0FFEE-> |------------+
       +----------------+    | phys 0x42  |
       | bss            |    | 0xDEADBE-> |
       +----------------+    | phys 0x07  |
       | data           |    | ...        |
       +----------------+    +------------+
       | text (code)    |
       +----------------+
       0x00000000
```

A virtual address is split into multiple parts that index a multi-level page table. On x86-64 with 4 KB pages, a 48-bit virtual address is split into 9+9+9+9+12 bits: PML4 index, PDPT index, PD index, PT index, page offset. Each of the four levels is itself a page of 512 entries. So a single address translation could cost four memory accesses. The **TLB** (Translation Lookaside Buffer) caches recent translations to make this fast.

```
   Virtual address (48 bits)
   +--------+--------+--------+--------+--------------+
   | PML4   | PDPT   | PD     | PT     | offset       |
   | (9b)   | (9b)   | (9b)   | (9b)   | (12b = 4KB)  |
   +--------+--------+--------+--------+--------------+

       |        |        |        |          |
       |        |        |        |          +---> byte within page
       |        |        |        v
       |        |        |        +-------+
       |        |        |   PT   | entry |---> physical page #
       |        |        v        +-------+
       |        |    +-------+
       |        |    |  PD   | entry |---> PT base
       |        v    +-------+
       |    +-------+
       |    | PDPT  | entry |---> PD base
       v    +-------+
   +-------+
   | PML4  | entry |---> PDPT base
   +-------+
       ^
       |
   CR3 register: physical address of this process's PML4
```

When the kernel context-switches between processes, it writes a new value to the **CR3** register (on x86-64) to point at the new process's top-level page table. That single write swaps the entire address space.

A **page fault** happens when the CPU tries to translate an address but the page table entry is absent or has wrong permissions. The CPU traps to the kernel. The kernel inspects what happened: maybe the page is on disk (load it from swap, fix the page table, return); maybe the program is about to grow its stack (allocate a new page); maybe the program just dereferenced a wild pointer (send `SIGSEGV`, the famous segmentation fault).

Useful per-process file: `/proc/<pid>/maps` shows the entire virtual layout of a process — every code section, library, heap, stack, thread stack, and anonymous mapping. We will read it in Hands-On.

Version note: 5-level paging (since Linux 4.12, hardware: Intel Ice Lake-SP) extends the address space from 48 to 57 bits (256 TB → 128 PB).

### The CFS Scheduler (Completely Fair Scheduler)

CFS picks the next task to run by always choosing the runnable task with the smallest **vruntime** (virtual runtime). It stores all runnable tasks in a **red-black tree** keyed by vruntime, so the leftmost node is always the next task.

```
                runnable tasks indexed by vruntime
                          [red-black tree]

                              ( 50 )
                             /      \
                          ( 30 )   ( 80 )
                          /    \   /    \
                       ( 20 ) (40)(70) (90)
                       /
                    ( 10 )  <-- leftmost = next to run
```

When task X with vruntime 10 gets to run for, say, 6 ms, the kernel adds 6 ms (scaled by its weight from `nice`) to its vruntime, removes it from the tree, and re-inserts it at the new position. Now some other task is leftmost.

Pseudocode of the core loop, simplified:

```c
struct task *pick_next_task(struct cfs_rq *rq) {
    struct task *t = leftmost_node(rq->tasks);   // O(log N) at worst, O(1) cached
    return t;
}

void put_prev_task(struct cfs_rq *rq, struct task *prev, u64 ran_ns) {
    prev->vruntime += ran_ns * NICE_0_LOAD / prev->load_weight;
    rb_insert(&rq->tasks, prev);
}
```

`nice` values change `load_weight` according to a fixed table. A nice-0 process has load_weight 1024. A nice-19 process has weight ~15 (so it accumulates vruntime ~70x faster — it runs much less). A nice -20 process has weight ~88761 (it accumulates vruntime ~86x slower — it runs much more). The math is a geometric series with ratio ~1.25 per nice level.

CFS coexists with **real-time** scheduling classes (`SCHED_FIFO`, `SCHED_RR`) which always preempt CFS, and with **`SCHED_DEADLINE`** (since Linux 3.14) for tasks with hard timing requirements (EDF — Earliest Deadline First — with a CBS bandwidth server).

Version note: CFS was introduced in Linux 2.6.23 (October 2007), replacing the old O(1) scheduler. A new scheduler called **EEVDF** (Earliest Eligible Virtual Deadline First) replaced CFS in **Linux 6.6** (October 2023); userspace tools and concepts (vruntime, nice, the rb-tree) are very similar, and most documentation still says "CFS."

### The VFS Layer (one interface, many filesystems)

The Virtual File System is the abstraction that lets you treat ext4, xfs, btrfs, NFS, and tmpfs as if they were the same thing.

```
                              user program
                                    |
                                    | open(), read(), write(), close()
                                    v
                              +-------------+
                              |  syscall    |
                              |  layer      |
                              +-------------+
                                    |
                                    v
                              +-------------+
                              |   VFS       |   <-- generic ops
                              | (vfs.c,     |       struct file_operations
                              |  inode.c,   |       struct dentry
                              |  dcache.c)  |       struct inode
                              +-------------+
                                    |
              +---------+-----------+-----------+----------+----------+
              v         v           v           v          v          v
         +-------+  +-------+   +-------+   +-------+  +-------+  +-------+
         | ext4  |  | xfs   |   | btrfs |   | nfs   |  | tmpfs |  | proc  |
         | .ko   |  | .ko   |   | .ko   |   | .ko   |  |       |  |       |
         +-------+  +-------+   +-------+   +-------+  +-------+  +-------+
              |         |           |           |          |          |
              v         v           v           v          v          v
         +-------+  +-------+   +-------+   +-------+  +------+   +-------+
         | block | | block |   | block |   | net   |  | RAM  |   | kernel|
         | device|  | device|   | device|   | sock  |  |      |  |  data |
         +-------+  +-------+   +-------+   +-------+  +------+   +-------+
```

Key VFS objects:
- **inode** — the actual file (size, owner, perms, block pointers). One inode can have many names.
- **dentry** — directory entry. Maps a name to an inode. Cached aggressively (`/proc/sys/fs/dentry-state`).
- **file** — an open file descriptor, with a position, mode, flags, and a pointer to the inode.
- **superblock** — represents a mounted filesystem. One superblock per mount.
- **vfsmount** — represents a mount point in the namespace tree.

The "everything is a file" principle works because VFS lets each filesystem implement its own `read()`, `write()`, `open()`, `mmap()` etc. so reading `/proc/cpuinfo` runs procfs's read function, which generates the bytes on the fly. There is no "cpuinfo" file on your disk.

Version note: tmpfs was added in 2.4. ext4 in 2.6.28 (2008). btrfs went stable around 3.10 (2013). f2fs (for flash) in 3.8. The kernel's NFS client is much older than the kernel itself in spirit (NFS dates to 1984).

### The Network Stack (Application → Transport → Network → Link → Physical)

The kernel implements the whole transport, network, and link layer for you. Your application speaks to a **socket**.

```
       OUTGOING                                INCOMING
       ========                                ========

   +---------------+                       +---------------+
   | Application   |   send("GET /...")    | Application   |   recv()
   | (user space)  |---+                   | (user space)  |^
   +---------------+   |                   +---------------+|
                       |                                     |
   +---------------+   v   +---- syscall -----+             |
   | socket layer  |<------|                  |             |
   | (BSD sockets) |       +------------------+             |
   +---------------+                                         |
       |                                                     |
       v                                                     |
   +---------------+                                  +---------------+
   | Transport:    |                                  | Transport:    |
   |  TCP / UDP    |  add seq#, checksums, ports     |  TCP / UDP    |
   +---------------+                                  +---------------+
       |                                                     ^
       v                                                     |
   +---------------+                                  +---------------+
   | Network: IP   |  add src/dst IP, route lookup,  | Network: IP   |
   | (netfilter,   |  consult routing table          | (defrag,      |
   |  routing)     |                                  |  netfilter)   |
   +---------------+                                  +---------------+
       |                                                     ^
       v                                                     |
   +---------------+                                  +---------------+
   | Link: queue   |  build Ethernet/Wi-Fi frame,    | Link: receive |
   | discipline    |  ARP for next-hop MAC           | (NAPI poll)   |
   |  (qdisc)      |                                  |               |
   +---------------+                                  +---------------+
       |                                                     ^
       v                                                     |
   +---------------+                                  +---------------+
   | Driver        |  call ndo_start_xmit()          | Driver        |
   |  (e1000, igb, |  to push to NIC ring buffer     | (IRQ → NAPI)  |
   |   r8169, ...) |                                  |               |
   +---------------+                                  +---------------+
       |                                                     ^
       v                                                     |
   +---------------+                                  +---------------+
   | NIC hardware  |   electrical signals on wire,   | NIC hardware  |
   | (PHY, MAC)    |   or RF on Wi-Fi                | (PHY, MAC)    |
   +---------------+                                  +---------------+
                       (the wire / radio / fibre)
```

Highlights:
- **netfilter** (and the modern **nftables** / older **iptables** front-ends) is the firewall hook framework. Every packet passes through hook points (PREROUTING, INPUT, FORWARD, OUTPUT, POSTROUTING) where rules can drop, accept, mangle, or NAT it.
- **NAPI** (since 2.4.20) is the kernel's poll-instead-of-interrupt mechanism for high-rate NICs. The NIC fires one IRQ, and then the kernel polls the ring until empty before re-enabling IRQ.
- **GRO/GSO** (Generic Receive/Segmentation Offload) batch contiguous TCP segments to reduce per-packet cost.
- **XDP** (eXpress Data Path, since 4.8) and **eBPF** let you run user-supplied programs at the very lowest level, before sk_buff allocation. We cover this in the high-school sheet.

Version note: TCP BBR (a congestion control algorithm) since 4.9; MPTCP since 5.6; io_uring (a new async I/O interface that also supports network ops) since 5.1.

### Kernel Modules (loadable .ko files)

The kernel's footprint stays small because most drivers are not built into the binary; they are **loadable modules**, files ending in `.ko` (kernel object), living under `/lib/modules/$(uname -r)/`.

```
              +-------------------------------+
              |  Running kernel (vmlinuz)     |
              |  always-resident core         |
              +-------------------------------+
                            ^
                            | insmod / rmmod
                            |
              +---------------+   +-----------+   +-------------+
              | snd_hda_intel |   | nvidia.ko |   | ext4.ko     |
              | .ko           |   |           |   |             |
              +---------------+   +-----------+   +-------------+
              (audio driver)      (GPU driver)    (filesystem)

              +----------- /lib/modules/$(uname -r) -----------+
              | .../kernel/drivers/sound/...                    |
              | .../kernel/drivers/net/ethernet/...             |
              | .../kernel/fs/ext4/ext4.ko                      |
              | modules.dep   (dependency map, built by depmod) |
              | modules.alias (alias → module map)              |
              +-------------------------------------------------+
```

Tools:
- **`lsmod`** — list currently loaded modules
- **`modinfo <name>`** — print metadata about a module (description, author, license, parameters)
- **`modprobe <name>`** — high-level: resolves dependencies via `modules.dep`, then loads
- **`modprobe -r <name>`** — remove a module and its now-unused deps
- **`insmod /path/to.ko`** — low-level: load a specific .ko file, NO dependency resolution
- **`rmmod <name>`** — low-level: unload by name
- **`depmod`** — rebuild `modules.dep` (run after installing a new kernel)

When you plug in new hardware, the **udev** daemon (part of systemd) reads kernel uevents from `/sys`, finds the matching driver name in `modules.alias`, and calls `modprobe`. This is why USB devices "just work."

Module signing has been mandatory on Secure Boot machines since around Linux 3.7 / 4.4. The kernel will refuse to load an unsigned module on a machine where Secure Boot is enabled, unless you explicitly allow it via MOK.

## Hands-On

Open a terminal. Every command below runs on any modern Linux system. Expected output is shown for an Ubuntu/Debian-style box; your numbers will differ but the shape will not.

### 1. Boot log: dmesg

```bash
$ dmesg | head -50
```

Expected (truncated; first lines):

```
[    0.000000] Linux version 6.5.0-21-generic (buildd@lcy02-amd64-027) ...
[    0.000000] Command line: BOOT_IMAGE=/boot/vmlinuz-6.5.0-21-generic root=UUID=...
[    0.000000] KERNEL supported cpus: Intel, AMD, Hygon, Centaur, ...
[    0.000000] BIOS-provided physical RAM map:
[    0.000000] BIOS-e820: [mem 0x0000000000000000-0x000000000009ffff] usable
[    0.000000] BIOS-e820: [mem 0x00000000000a0000-0x00000000000fffff] reserved
...
[    0.123456] ACPI: PM-Timer IO Port: 0x408
[    0.234567] smpboot: CPU0: Intel(R) Core(TM) i7-...
[    0.345678] Brought up 8 CPUs
[    1.234567] systemd[1]: Hostname set to <my-laptop>.
```

The bracketed numbers are seconds since boot. If you can't run dmesg as a non-root user (some distros restrict it via `kernel.dmesg_restrict`), prefix `sudo`.

### 2. Process tree: pstree

```bash
$ pstree | head -20
```

Expected:

```
systemd---ModemManager---2*[{ModemManager}]
        |-NetworkManager---2*[{NetworkManager}]
        |-accounts-daemon---2*[{accounts-daemon}]
        |-bluetoothd
        |-cron
        |-dbus-daemon
        |-gdm3---gdm-session-wor---gdm-x-session-+-Xorg---{Xorg}
        |                                        |-gnome-session-b---2*[{gnome-session-b}]
        |                                        `-2*[{gdm-x-session}]
        |-gnome-keyring-d---4*[{gnome-keyring-d}]
        |-rsyslogd---3*[{rsyslogd}]
        |-systemd---(sd-pam)
        |-systemd-journal
        |-systemd-logind
        |-systemd-resolve
        |-systemd-timesyn---{systemd-timesyn}
        |-systemd-udevd
        |-2*[bash---pstree]
        ...
```

`pstree` shows the family tree. PID 1 (systemd) is the trunk; everything else descends from it.

### 3. Process forest with PID and PPID

```bash
$ ps -eo pid,ppid,cmd --forest | head -20
```

Expected:

```
    PID    PPID CMD
      1       0 /sbin/init splash
    345       1 /lib/systemd/systemd-journald
    367       1 /lib/systemd/systemd-udevd
    456       1 /sbin/dhclient -1 -v -pf /run/dhclient.eth0.pid -lf ...
    567       1 /usr/sbin/cron -f
    678       1 /usr/sbin/sshd -D
   1234     678  \_ sshd: stevie [priv]
   1245    1234  |   \_ sshd: stevie@pts/0
   1246    1245  |       \_ -bash
   1300    1246  |           \_ ps -eo pid,ppid,cmd --forest
   ...
```

PPID 0 belongs only to PID 1 (the kernel reaped its actual ancestor). The `\_` lines show child processes branching off their parents — the same fork tree you saw in `pstree`.

### 4. Per-process status

```bash
$ cat /proc/$$/status | head -20
```

Expected (`$$` is your shell's PID):

```
Name:	bash
Umask:	0022
State:	S (sleeping)
Tgid:	1246
Ngid:	0
Pid:	1246
PPid:	1245
TracerPid:	0
Uid:	1000	1000	1000	1000
Gid:	1000	1000	1000	1000
FDSize:	256
Groups:	4 24 27 30 46 110 1000
NStgid:	1246
NSpid:	1246
NSpgid:	1246
NSsid:	1245
VmPeak:	   16584 kB
VmSize:	   16584 kB
VmLck:	       0 kB
VmPin:	       0 kB
```

State `S` is interruptible sleep — the shell is waiting for you to type. Try the same on a CPU-bound process and you'll see `R`.

### 5. Virtual memory layout of a process

```bash
$ cat /proc/$$/maps | head -20
```

Expected:

```
55a4b3c00000-55a4b3c2b000 r--p 00000000 fd:00 1311265   /usr/bin/bash
55a4b3c2b000-55a4b3d05000 r-xp 0002b000 fd:00 1311265   /usr/bin/bash
55a4b3d05000-55a4b3d40000 r--p 00105000 fd:00 1311265   /usr/bin/bash
55a4b3d40000-55a4b3d44000 r--p 0013f000 fd:00 1311265   /usr/bin/bash
55a4b3d44000-55a4b3d4d000 rw-p 00143000 fd:00 1311265   /usr/bin/bash
55a4b3d4d000-55a4b3d58000 rw-p 00000000 00:00 0
55a4b50ef000-55a4b525c000 rw-p 00000000 00:00 0         [heap]
7fe1c0000000-7fe1c0021000 rw-p 00000000 00:00 0
7fe1c0021000-7fe1c4000000 ---p 00000000 00:00 0
7fe1c8000000-7fe1c8021000 rw-p 00000000 00:00 0
...
7ffd5a4b1000-7ffd5a4d2000 rw-p 00000000 00:00 0         [stack]
7ffd5a4f8000-7ffd5a4fc000 r--p 00000000 00:00 0         [vvar]
7ffd5a4fc000-7ffd5a4fe000 r-xp 00000000 00:00 0         [vdso]
ffffffffff600000-ffffffffff601000 --xp 00000000 00:00 0 [vsyscall]
```

Each line: virtual address range, permissions (`rwxp` = read/write/execute/private), file offset, dev:inode of the backing file (or 0 0 for anonymous), and the file path or `[heap]` / `[stack]` / `[vdso]` labels. This *is* the process's memory map. The kernel built it.

### 6. System memory and meminfo

```bash
$ free -h && echo --- && cat /proc/meminfo | head -10
```

Expected:

```
               total        used        free      shared  buff/cache   available
Mem:            15Gi       4.2Gi       2.5Gi       512Mi       8.3Gi        10Gi
Swap:          8.0Gi          0B       8.0Gi
---
MemTotal:       16284124 kB
MemFree:         2632108 kB
MemAvailable:   10487640 kB
Buffers:          425900 kB
Cached:          7903904 kB
SwapCached:            0 kB
Active:          5012348 kB
Inactive:        6789012 kB
Active(anon):    1234567 kB
Inactive(anon):   234567 kB
```

**buff/cache** is the page cache — disk pages the kernel keeps in RAM in case anyone reads them again. It's "free in spirit" — the kernel will evict it instantly if a real allocation needs the memory. `available` is the more honest "how much could you actually allocate?" number.

### 7. Swap

```bash
$ cat /proc/swaps
```

Expected:

```
Filename                                Type            Size            Used      Priority
/swapfile                               file            8388604         0         -2
```

If empty, no swap is configured. Modern systems on SSDs often run with little or no swap, relying on **zswap** or **zram** instead.

### 8. Kernel version

```bash
$ uname -a && echo --- && cat /proc/version
```

Expected:

```
Linux mybox 6.5.0-21-generic #21-Ubuntu SMP PREEMPT_DYNAMIC Wed Feb 28 ... x86_64 GNU/Linux
---
Linux version 6.5.0-21-generic (buildd@lcy02-amd64-027) (gcc-12 (Ubuntu 12.3.0-1ubuntu1~23.04) 12.3.0, GNU ld (GNU Binutils for Ubuntu) 2.40) #21-Ubuntu SMP PREEMPT_DYNAMIC ...
```

`uname -r` alone gives just the kernel release. `uname -m` gives the architecture (`x86_64`, `aarch64`, `armv7l`, ...).

### 9. Mounts

```bash
$ mount | head -10
```

Expected:

```
sysfs on /sys type sysfs (rw,nosuid,nodev,noexec,relatime)
proc on /proc type proc (rw,nosuid,nodev,noexec,relatime)
udev on /dev type devtmpfs (rw,nosuid,relatime,size=8051840k,nr_inodes=2012960,mode=755)
devpts on /dev/pts type devpts (rw,nosuid,noexec,relatime,gid=5,mode=620,ptmxmode=000)
tmpfs on /run type tmpfs (rw,nosuid,nodev,noexec,relatime,size=1622984k,mode=755)
/dev/sda2 on / type ext4 (rw,relatime,errors=remount-ro)
tmpfs on /dev/shm type tmpfs (rw,nosuid,nodev)
tmpfs on /run/lock type tmpfs (rw,nosuid,nodev,noexec,relatime,size=5120k)
cgroup2 on /sys/fs/cgroup type cgroup2 (rw,nosuid,nodev,noexec,relatime,nsdelegate,memory_recursive_prot)
tracefs on /sys/fs/cgroup/unified type tracefs ...
```

Notice how many filesystems are mounted that you never thought about: sysfs, procfs, devtmpfs, tmpfs, cgroup2, tracefs. All VFS, all uniform.

### 10. Networking: routes and addresses

```bash
$ ip route show && echo --- && ip addr show
```

Expected:

```
default via 192.168.1.1 dev wlp3s0 proto dhcp metric 600
192.168.1.0/24 dev wlp3s0 proto kernel scope link src 192.168.1.42 metric 600
---
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN group default qlen 1000
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
    inet 127.0.0.1/8 scope host lo
       valid_lft forever preferred_lft forever
    inet6 ::1/128 scope host
       valid_lft forever preferred_lft forever
2: enp2s0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc fq_codel state UP group default qlen 1000
    link/ether ab:cd:ef:01:23:45 brd ff:ff:ff:ff:ff:ff
    inet 192.168.1.42/24 brd 192.168.1.255 scope global dynamic noprefixroute enp2s0
       valid_lft 86384sec preferred_lft 86384sec
3: wlp3s0: ...
```

`ip route` is your routing table — what the kernel consults at the network layer. `ip addr` lists interfaces (link layer up) and their L3 addresses.

### 11. Kernel modules

```bash
$ lsmod | head -10 && echo --- && modinfo $(lsmod | awk 'NR==2{print $1}')
```

Expected:

```
Module                  Size  Used by
nls_iso8859_1          16384  1
snd_hda_codec_realtek 159744  1
snd_hda_codec_generic  98304  1 snd_hda_codec_realtek
snd_hda_codec_hdmi     77824  1
btusb                  65536  0
btintel                49152  1 btusb
btmtk                  16384  1 btusb
nvidia_uvm           1359872  0
nvidia_drm             77824  4
---
filename:       /lib/modules/6.5.0-21-generic/kernel/.../nls_iso8859_1.ko
license:        GPL
description:    NLS ISO 8859-1 (Latin 1; Western European Languages)
author:         <linux-kernel@vger.kernel.org>
srcversion:     ABCDEF1234567890
depends:
retpoline:      Y
intree:         Y
name:           nls_iso8859_1
vermagic:       6.5.0-21-generic SMP preempt mod_unload modversions
sig_id:         PKCS#7
signer:         Build time autogenerated kernel key
...
```

`lsmod` columns: name, size in bytes, refcount + comma-separated list of users. A module with refcount > 0 cannot be removed (`rmmod` will refuse).

## Common Confusions

### 1. "What's the difference between `fork()` and `exec()`?"

Broken thinking: "fork starts a new program, right?"
Fixed: **Fork copies; exec replaces.** `fork()` makes a duplicate of the current process — same code, same data, almost everything. The *child* then usually calls `exec()` (specifically `execve()`), which throws away the duplicated code and image and replaces it with a different program — but keeps the same PID, same parent, same open file descriptors. Two steps because between them the child can adjust things (file descriptors, env, UID) without affecting the parent. This is why shell redirection works.

### 2. "Why is `ps -ef` showing what looks like duplicate processes?"

Broken: "Two `bash` lines? My system is broken!"
Fixed: It's almost always the **parent and its child after a fork**. When you ran `ps`, your shell forked itself; the child execve'd into `ps`. While the child is running, you may briefly see two bash entries if you look at the right moment, or you may see `bash` (parent) and `ps` (child). For `ssh` connections, you commonly see `sshd: user [priv]` AND `sshd: user@pts/0` — that's not a bug, it's the privilege-separated parent and the per-session child.

### 3. "Why does the OOM killer pick MY process?"

Broken: "It's random / it's malicious / it hates me."
Fixed: The **OOM killer** computes a **badness score** for each process when memory runs out, written to `/proc/<pid>/oom_score`. The score grows roughly with the process's memory footprint (RSS), and you can bias it via `/proc/<pid>/oom_score_adj` (`-1000` = never, `+1000` = always pick me). The biggest memory user is the most likely victim. `dmesg | grep -i 'killed process'` shows the kill receipt with reason. To protect a critical process: `echo -1000 > /proc/<pid>/oom_score_adj` (as root) or use systemd's `OOMScoreAdjust=` directive.

### 4. "Is the scheduler the same as the dispatcher?"

Broken: "Aren't they the same thing?"
Fixed: They're related but distinct concepts. The **scheduler** is the *policy* — it decides *which* runnable task should run next (CFS picks smallest vruntime). The **dispatcher** is the *mechanism* — the low-level code that actually saves the previous task's registers, switches the page table (CR3), and restores the next task's registers. In Linux source they're tangled together in `kernel/sched/core.c` and the architecture-specific `__switch_to`. Conceptually: scheduler is brain, dispatcher is hands.

### 5. "Why is `kill -9 <pid>` not working?"

Broken: "Even SIGKILL is failing!"
Fixed: SIGKILL cannot be ignored, but it can be **delayed**. If the process is in **uninterruptible sleep state D** (almost always blocked on disk I/O — bad NFS server, broken USB drive, slow SAN) it cannot be killed until the I/O completes or fails. `ps` shows state `D`. Wait it out, fix the underlying I/O problem, or `echo b > /proc/sysrq-trigger` to reboot.

### 6. "Why is my `free` showing almost no free memory?"

Broken: "Linux ate my RAM!"
Fixed: Linux **uses free RAM as the page cache**. That's a feature, not a bug — empty RAM is wasted RAM. Look at the `available` column (newer `free`) or `MemAvailable` in `/proc/meminfo`. That's how much you could actually allocate. The kernel will instantly evict cache to satisfy a real request. Slogan: "free RAM is wasted RAM."

### 7. "What is the difference between `/proc` and `/sys`?"

Broken: "Aren't they the same?"
Fixed: **`/proc`** (procfs) is older and a grab bag — process info per-PID, plus various global stats and tunables (`/proc/cpuinfo`, `/proc/meminfo`, `/proc/sys/...`). **`/sys`** (sysfs, since 2.6) is newer and disciplined — one file per kernel attribute, organized by the device tree. New tunables tend to go in `/sys`. `/proc/sys` (the sysctl tree) is special — those are kernel parameters, controlled by `sysctl`. Rule of thumb: **process info → /proc; device + driver info → /sys; tunables → either, but `sysctl` configures /proc/sys.**

### 8. "What's the difference between `insmod` and `modprobe`?"

Broken: "Both load modules, just pick one."
Fixed: **`insmod`** is dumb — it loads exactly the .ko file you point at, with no dependency resolution. If the module needs another module (`depends:` field in `modinfo`), the load fails with "Unknown symbol in module." **`modprobe`** is smart — it consults `/lib/modules/<kver>/modules.dep`, loads dependencies first, then loads the requested module. Always use `modprobe` unless you have a very specific reason. To unload: `rmmod` is the dumb counterpart to `insmod`; `modprobe -r` is the smart counterpart.

### 9. "Why does `df -h` show different free space than `du -sh`?"

Broken: "Filesystem is broken / lying."
Fixed: **`du`** sums the sizes of files it can see; **`df`** asks the filesystem how much is allocated. They differ because: (a) deleted-but-still-open files keep their disk blocks until the last fd closes — `df` sees them, `du` doesn't (find with `lsof +L1`); (b) reserved blocks for root (5% on ext4 by default — `tune2fs -m`); (c) filesystems hidden under mountpoints aren't visible to `du` once you mount over them; (d) journal, snapshots, and reflink overhead. To free space from a deleted-but-open file: kill the process holding it.

### 10. "Why is my system swapping when I have free RAM?"

Broken: "There's RAM available, why touch the slow disk?"
Fixed: The kernel parameter **`vm.swappiness`** (0-100, default 60) controls the kernel's eagerness to swap anonymous pages out to make room for page cache. With high swappiness, the kernel will swap idle anonymous memory before evicting page cache, on the theory that recently-cached file pages are more valuable. To reduce swapping aggressiveness: `sudo sysctl vm.swappiness=10`. Also on systems with NUMA, memory might be exhausted on the local node even when other nodes have free RAM (`numactl --hardware`).

### 11. "Why is `top` showing 800% CPU?"

Broken: "That's impossible, I have one CPU."
Fixed: `top` (and `ps`) accumulate CPU usage **per thread per CPU**, so a process with 8 threads pegging 8 cores shows 800%. You have 8 cores. Press `1` in `top` to see per-CPU breakdown; switch to `htop` for a friendlier display. The kernel sees this correctly — it's a presentation choice in userspace tools.

### 12. "Why does my container see all the host's CPUs / memory?"

Broken: "Containers must be virtual machines."
Fixed: Containers share the host kernel. A container is a process with **namespaces** (PID, mount, network, user, IPC, UTS, cgroup, time) and **cgroups** (resource limits). The kernel only enforces what the container's cgroup says. By default a container has no CPU or memory cap and sees all the host's hardware in `/proc/cpuinfo`. You set limits with `--cpus`, `--memory` in Docker, or `CPUQuota=`, `MemoryMax=` in systemd. Tools like `lscpu` and `free` may show host numbers because they read from `/proc` directly.

## Vocabulary

| Term | Meaning |
|---|---|
| **kernel** | The core of the OS, runs in privileged CPU mode (ring 0). |
| **kernel mode (ring 0)** | CPU mode with full hardware access. Only kernel code runs here. |
| **user mode (ring 3)** | CPU mode with restricted access. All user programs run here. |
| **syscall** | The doorway from user mode to kernel mode; how programs ask for service. |
| **fork** | Syscall that clones the calling process. Two of you, returns 0 to child, child PID to parent. |
| **exec / execve** | Syscall that replaces the current process image with a different program (same PID). |
| **vfork** | Older optimization of fork that suspends the parent until child execs. Rarely used today; clone+VM_SHARED-ish. |
| **clone** | The general primitive underlying fork and pthread_create. Flags decide what's shared. |
| **wait / waitpid** | Syscall a parent uses to read the exit status of a finished child (and reap zombies). |
| **process** | A running program with its own address space, fds, etc. |
| **thread** | A schedulable unit sharing address space with siblings in the same process. |
| **PID** | Process ID. Numeric identifier for a process. |
| **PPID** | Parent process ID. The PID of the process that forked this one. |
| **PID 1 / init** | The first user-space process, ancestor of all others. systemd on most modern Linux. |
| **systemd** | The init system used by most modern Linux distros. Manages services. |
| **systemd unit** | A unit of work systemd manages (.service, .socket, .timer, .target, etc.). |
| **target** | A systemd unit grouping other units (multi-user.target, graphical.target). |
| **journal** | systemd's logging facility (`journalctl`). |
| **signal** | A small message sent to a process, e.g. SIGINT, SIGTERM. |
| **SIGTERM** | Signal 15. Polite "please exit." Default of `kill`. Catchable. |
| **SIGKILL** | Signal 9. Uncatchable, unblockable kill. Cannot be ignored. |
| **SIGSTOP** | Signal 19. Uncatchable pause. Resume with SIGCONT. |
| **SIGCONT** | Signal 18. Resume a stopped process. |
| **SIGSEGV** | Signal 11. Segmentation fault; you accessed memory you weren't allowed to. |
| **zombie** | A finished process whose exit status hasn't been read by its parent. |
| **orphan** | A still-running process whose parent died. Reparented to PID 1. |
| **scheduler** | The kernel subsystem that picks which runnable task runs next. |
| **CFS** | Completely Fair Scheduler. Linux's default scheduler since 2.6.23. |
| **vruntime** | Per-task virtual runtime used by CFS to pick the leftmost task. |
| **EEVDF** | Earliest Eligible Virtual Deadline First. Replaced CFS in 6.6. |
| **dispatcher** | The mechanism that swaps register state between tasks during context switch. |
| **context switch** | Saving one task's CPU state and loading another's. |
| **nice** | A per-process priority hint, -20 (greedy) to +19 (polite). Default 0. |
| **renice** | Change the nice value of a running process. |
| **ionice** | Set the I/O scheduling class/priority of a process. |
| **RT priority** | Real-time priority (1-99) for SCHED_FIFO/SCHED_RR tasks. Always preempt CFS. |
| **SCHED_FIFO** | Real-time, no time slicing. Runs until it blocks or yields. |
| **SCHED_RR** | Real-time, round-robin time slicing among same-priority tasks. |
| **SCHED_DEADLINE** | EDF-based, missed-deadline-aware class (since 3.14). |
| **virtual address** | The address a program uses; translated by MMU. |
| **physical address** | The actual RAM location. |
| **page** | Fixed-size chunk of memory, typically 4 KB. Unit of mapping. |
| **page table** | Multi-level data structure mapping virtual to physical pages. |
| **MMU** | Memory Management Unit, hardware that does the translation. |
| **TLB** | Translation Lookaside Buffer; CPU cache of recent address translations. |
| **page fault** | CPU trap when a virtual page isn't currently mapped or is misprotected. |
| **swap** | Disk-backed extension of RAM. Pages can be paged out and back in. |
| **OOM killer** | Out-of-memory killer; picks a victim when RAM is exhausted. |
| **oom_score** | Per-process score the OOM killer uses to choose a victim. |
| **file descriptor (fd)** | Per-process integer handle to an open file/socket/pipe (0=stdin, 1=stdout, 2=stderr). |
| **inode** | On-disk metadata for a file: size, perms, owner, block pointers. No name. |
| **dentry** | Directory entry: maps a name to an inode. Lives in dcache. |
| **superblock** | In-memory representation of a mounted filesystem. |
| **mount point** | A directory where a filesystem is attached to the namespace. |
| **bind mount** | Re-mount of an existing path at a new path (`mount --bind`). |
| **VFS** | Virtual File System; kernel layer abstracting all filesystems. |
| **ext4** | Default Linux filesystem since 2008 (kernel 2.6.28). Journaling. |
| **xfs** | High-performance journaling filesystem; default on RHEL since 7. |
| **btrfs** | Copy-on-write filesystem with snapshots, subvolumes, raid built-in. |
| **/proc** | procfs; per-process and global kernel state as a virtual filesystem. |
| **/sys** | sysfs; device, driver, and kernel object hierarchy as a virtual filesystem. |
| **/dev** | devtmpfs; device nodes — character and block devices as files. |
| **kernel module** | Loadable .ko file extending the running kernel. |
| **modprobe** | Load a module by name with dependency resolution. |
| **depmod** | Build the modules.dep dependency map. Run after kernel install. |
| **lsmod** | List currently loaded kernel modules. |
| **insmod** | Insert a specific .ko file (no dependency resolution). |
| **rmmod** | Remove a loaded module by name. |
| **modinfo** | Print metadata about a module file. |
| **dmesg** | Print the kernel ring buffer (boot and runtime messages). |
| **ring buffer** | Fixed-size circular log; oldest entries dropped when full. |
| **GRUB** | GRand Unified Bootloader; loads the kernel from disk. |
| **UEFI** | Unified Extensible Firmware Interface; modern motherboard firmware. |
| **BIOS** | Basic Input/Output System; legacy motherboard firmware. |
| **POST** | Power-On Self-Test; firmware's hardware self-check. |
| **EFI partition (ESP)** | FAT32 partition containing UEFI boot binaries. |
| **initramfs** | Small in-RAM root filesystem with enough drivers to find the real root. |
| **socket** | Endpoint for network I/O; same syscalls as files. |
| **netfilter** | Kernel framework for firewall hooks (iptables/nftables sit on top). |
| **NAPI** | Polled receive path for high-rate NICs. |
| **qdisc** | Queueing discipline — egress packet queue (`tc`). |
| **routing table** | The L3 forwarding table consulted by IP. |
| **udev** | Userspace device manager (part of systemd) — loads modules on hotplug. |
| **cgroup** | Control group; resource limit/account container for processes. |
| **namespace** | Per-process view isolation (PID/mount/net/user/IPC/UTS/cgroup/time). |

## Try This

Five-to-ten short experiments. Try at least three.

### Experiment 1: Trace a fork

```bash
$ strace -f -e trace=fork,clone,execve bash -c 'echo hi'
```

Watch the parent fork, the child clone or execve into `echo`, then `hi` print. Add `-f` to follow children. You'll see the syscall returns: parent gets a positive PID, child gets `0`.

### Experiment 2: See your shell's scheduling policy

```bash
$ chrt -p $$
```

Expected:

```
pid 1246's current scheduling policy: SCHED_OTHER
pid 1246's current scheduling priority: 0
```

`SCHED_OTHER` (alias `SCHED_NORMAL`) is the default CFS policy. `priority 0` is always shown for non-RT tasks; the actual influence comes from `nice`. Show nice with `ps -o pid,ni,cmd $$`.

### Experiment 3: Find the biggest memory hogs

```bash
$ ps aux --sort=-%mem | head -10
```

Top of the list = your fattest processes by RSS. If `firefox` and `chrome` aren't there, you're not human.

### Experiment 4: Inspect a process's open files

```bash
$ ls -la /proc/$$/fd/
```

Expected:

```
total 0
dr-x------ 2 stevie stevie  0 Apr 27 12:34 .
dr-xr-xr-x 9 stevie stevie  0 Apr 27 12:34 ..
lrwx------ 1 stevie stevie 64 Apr 27 12:34 0 -> /dev/pts/0
lrwx------ 1 stevie stevie 64 Apr 27 12:34 1 -> /dev/pts/0
lrwx------ 1 stevie stevie 64 Apr 27 12:34 2 -> /dev/pts/0
lrwx------ 1 stevie stevie 64 Apr 27 12:34 255 -> /dev/pts/0
```

Symlinks: 0 stdin, 1 stdout, 2 stderr — all pointing at your terminal.

### Experiment 5: Watch context switches in real time

```bash
$ vmstat 1
```

Expected (lines repeat every second):

```
procs -----------memory----------  ---swap-- -----io---- -system-- ------cpu-----
 r  b   swpd   free   buff  cache    si   so    bi    bo   in   cs us sy id wa st
 1  0      0 2632108 425900 7903904  0    0     5    20  120  340  3  1 95  1  0
```

`cs` = context switches per second, `in` = interrupts per second. Type something in another terminal and watch them rise.

### Experiment 6: Generate a synthetic page fault storm

```bash
$ /usr/bin/time -v dd if=/dev/zero of=/dev/null bs=1M count=1024
```

Expected (excerpt):

```
        Minor (reclaiming a frame) page faults: 248
        Major (requiring I/O) page faults: 0
        Voluntary context switches: 5
        Involuntary context switches: 2
```

Major faults = had to read from disk. Minor faults = page was satisfied without I/O. Real workloads should have very few major faults.

### Experiment 7: Walk the /proc tree by hand

```bash
$ cat /proc/cpuinfo | grep 'model name' | head -1
$ cat /proc/loadavg
$ cat /proc/uptime
$ cat /proc/version
$ cat /proc/sys/kernel/hostname
```

Each of these "files" is generated by the kernel on the fly. They have size 0 on `ls` but have content on `cat`.

### Experiment 8: List your network connections

```bash
$ ss -tunap | head
```

`-t` TCP, `-u` UDP, `-n` numeric, `-a` all (incl. listening), `-p` show pids. The replacement for `netstat`.

### Experiment 9: Drop the page cache (read-only test)

```bash
$ free -h
$ sync && sudo sh -c 'echo 1 > /proc/sys/vm/drop_caches'
$ free -h
```

Watch `buff/cache` shrink. This proves the page cache is using "free" memory and is voluntary. Don't do this on a busy server unless you understand the cost.

### Experiment 10: Measure fork cost

```bash
$ /usr/bin/time -v bash -c 'for i in $(seq 1 1000); do /bin/true; done'
```

The `User time` and `System time` divided by 1000 give a rough lower bound on the fork+exec+exit pipeline time. Modern Linux: ~200 microseconds.

## Deeper Dives

This section expands on each Concept above. Read the section that matches the question you're holding.

### Deeper Dive: Boot Process Internals

Modern Linux boot is more nuanced than the relay race diagram suggests. Below is an expanded version of each step with the file paths, processes, and commands you'd touch as a sysadmin.

#### Firmware variant: BIOS vs UEFI

```
  Legacy BIOS                    Modern UEFI
  ===========                    ===========

  Real mode (16-bit)             Long mode (64-bit) right out of reset
  CPU reset vector: 0xFFFFFFF0   CPU starts where firmware says
  Reads MBR (sector 0) of disk   Reads NVRAM boot entries
  MBR is 512 bytes, contains:    Each entry points to an .efi binary
    - 446 bytes bootstrap code    on the EFI System Partition (ESP)
    - 64 bytes partition table    typically /boot/efi or /efi
    - 2 bytes 0x55AA signature   ESP filesystem: FAT32
  Stage1 loads stage2 from disk  ESP path:
  Stage2 (GRUB core) loads        /EFI/BOOT/BOOTX64.EFI (default)
  configuration and shows menu    /EFI/<distro>/grubx64.efi
  Limits: 2 TB MBR; 4 partitions Boots directly into the .efi
  No native graphics              binary in long mode
                                 Has its own filesystem driver,
                                 network stack, protocols
                                 Secure Boot via signed binaries
```

The single most important command for inspecting UEFI variables: `efibootmgr -v`. This reads the boot order and entries directly from NVRAM:

```bash
$ sudo efibootmgr -v
BootCurrent: 0001
Timeout: 1 seconds
BootOrder: 0001,0002,0000
Boot0000* Windows Boot Manager  HD(1,GPT,...)/File(\EFI\Microsoft\Boot\bootmgfw.efi)
Boot0001* ubuntu                HD(1,GPT,...)/File(\EFI\ubuntu\shimx64.efi)
Boot0002* UEFI: USB Hard Drive  ...
```

`shimx64.efi` is the **shim** — a small Microsoft-signed binary that verifies and chain-loads `grubx64.efi`. This is how Linux distros boot under Secure Boot without each vendor needing a Microsoft signature on every kernel.

#### GRUB internals

GRUB has multiple stages:
- **boot.img** — fits in the MBR (legacy BIOS) or boot sector
- **diskboot.img / cdboot.img** — knows about disk partitions
- **core.img** — loaded into memory; contains filesystem drivers
- **grub.cfg** — text configuration, lives at `/boot/grub/grub.cfg`

Useful files:

```bash
$ cat /etc/default/grub        # source of truth (Debian/Ubuntu)
$ ls /etc/grub.d/              # snippets that update-grub assembles
$ sudo update-grub             # regenerate /boot/grub/grub.cfg
$ sudo grub-install /dev/sda   # rewrite the boot sector / EFI entry
```

`/proc/cmdline` shows the kernel command line GRUB passed:

```bash
$ cat /proc/cmdline
BOOT_IMAGE=/boot/vmlinuz-6.5.0-21-generic root=UUID=abcd-1234 ro quiet splash
```

Common cmdline parameters:
- `root=UUID=...` — which partition to mount as `/`
- `ro` / `rw` — initial mount mode
- `quiet` — suppress most kernel boot messages
- `splash` — show graphical splash (Plymouth)
- `nomodeset` — disable kernel mode-setting (legacy text console; useful for graphics troubleshooting)
- `init=/bin/bash` — boot to a bash shell instead of systemd (rescue)
- `single` — boot to single-user mode
- `mem=4G` — pretend system has only 4 GB
- `nosmt` — disable SMT (mitigations for Spectre-class attacks)
- `mitigations=off` — disable all CPU vulnerability mitigations (faster, less safe)

#### Kernel decompression and self-extraction

The file `/boot/vmlinuz-...` is actually a small **bzImage** wrapper around a compressed kernel image. The first thing the kernel does is decompress itself in place. Inspect with:

```bash
$ file /boot/vmlinuz-$(uname -r)
/boot/vmlinuz-6.5.0-21-generic: Linux kernel x86 boot executable bzImage, version 6.5.0-21-generic ..., RO-rootFS, swap_dev 0xC, Normal VGA
```

The compressed payload can be `gzip`, `bzip2`, `lzma`, `xz`, `lzo`, or `zstd` (since 5.9). Most modern distros use zstd.

#### initramfs internals

```bash
$ ls -la /boot/initrd.img-$(uname -r)
$ lsinitramfs /boot/initrd.img-$(uname -r) | head -20    # Debian/Ubuntu
$ lsinitrd /boot/initramfs-$(uname -r).img | head -20    # Fedora/RHEL
```

Inside an initramfs (it's a cpio archive optionally compressed):

```
.
bin -> usr/bin
etc/
init                 <- the entry point (PID 1 inside initramfs)
sbin -> usr/sbin
scripts/
usr/
usr/lib/modules/.../kernel/drivers/...
```

The initramfs's job: bring up enough infrastructure to find the *real* root. That includes loading storage drivers, assembling LVM volumes (`vgchange -ay`), unlocking LUKS volumes (asking for a password), setting up multipath, mounting NFS roots, etc. Once the real root is mounted at `/sysroot` (or `/root`), the script does:

```sh
exec switch_root /sysroot /sbin/init
```

`switch_root` (or `pivot_root` underneath) makes `/sysroot` the new `/`, frees the old initramfs RAM, and execs the real init.

#### systemd boot in detail

systemd boots by activating a **target**. The default target is a symlink:

```bash
$ readlink /etc/systemd/system/default.target
/lib/systemd/system/graphical.target
```

Targets pull in other targets and services via `Wants=` and `Requires=`:

```
default.target -> graphical.target
                       Requires: multi-user.target
                                       Requires: basic.target
                                                       Requires: sysinit.target
                                                                       Wants: systemd-udevd.service
                                                                       Wants: systemd-tmpfiles-setup.service
                                                                       ...
                                       Wants: getty.target
                                       Wants: <user services>
                       Wants: gdm.service / sddm.service / lightdm.service
```

Inspect actual boot performance:

```bash
$ systemd-analyze
Startup finished in 4.123s (firmware) + 2.456s (loader) + 1.234s (kernel) + 12.789s (userspace) = 20.602s
graphical.target reached after 12.789s in userspace.

$ systemd-analyze blame | head
12.456s NetworkManager-wait-online.service
4.567s plymouth-quit-wait.service
2.345s snapd.service
1.123s systemd-journal-flush.service
...

$ systemd-analyze critical-chain
graphical.target @12.789s
└─multi-user.target @12.789s
  └─plymouth-quit-wait.service @8.222s +4.567s
    └─systemd-user-sessions.service @8.111s +109ms
      └─network.target @8.000s
        └─NetworkManager.service @1.234s +6.766s
          └─dbus.service @1.123s
            └─basic.target @1.111s
              └─sockets.target @1.111s
                ...
```

`systemd-analyze plot > boot.svg` produces a beautiful visual timeline.

### Deeper Dive: Process Lifecycle

Beyond fork/exec, the process lifecycle has a few more nuances worth understanding.

#### The exit cascade

When a process calls `exit()` (or returns from `main`), the kernel:
1. Closes all file descriptors (decrementing reference counts; flushing if last reference)
2. Releases its address space (the page tables and physical pages)
3. Releases its CPU time accounting
4. Sends SIGCHLD to its parent
5. Sets state to ZOMBIE (in `/proc` it shows as `(Z)` and has `Tgid=Pid` but no memory mappings)
6. Waits for the parent to call `wait()` / `waitpid()` / `waitid()`
7. On reap: removes from process table, frees the `task_struct`

If the parent dies first, the kernel reparents the orphan to PID 1 (or to a closer "subreaper" registered with `prctl(PR_SET_CHILD_SUBREAPER)`). PID 1 is required to reap zombies, which is why systemd is so careful about its child management.

#### Zombie demonstration (controlled)

```bash
$ bash -c 'sleep 5 & exec sleep 30' &
[1] 4567
$ ps -o pid,ppid,state,cmd --ppid 4567
    PID    PPID S CMD
   4568    4567 S sleep 5
$ # wait 6 seconds
$ ps -o pid,ppid,state,cmd --ppid 4567
    PID    PPID S CMD
   4568    4567 Z [sleep] <defunct>
```

The child `sleep 5` finished, but the parent (`sleep 30`) doesn't call wait, so we see `Z` and `<defunct>`. After 24 more seconds the parent exits and the kernel reaps everything.

#### Daemonization

A "daemon" is a long-running background process. The traditional **double-fork** dance:

```c
pid_t pid = fork();
if (pid > 0) exit(0);          // parent exits, child becomes orphan -> PID 1 adopts it
setsid();                      // new session, no controlling terminal
pid = fork();
if (pid > 0) exit(0);          // grandparent exits; double-orphan can never acquire a TTY
chdir("/");                    // detach from any cwd that might be unmounted
umask(0);
close(0); close(1); close(2);  // detach from terminal
// reopen 0,1,2 to /dev/null or a log file
```

Modern systemd-managed daemons skip almost all of this. systemd handles environment and pipes for you — see `Type=simple` and `Type=notify` in `man systemd.service`.

#### Process credentials

Each process carries identity:
- **UID / EUID** — real and effective user ID
- **GID / EGID** — real and effective group ID
- **Supplementary groups** — additional groups
- **SUID / SGID bits** on executables let a program run as a different UID/GID; `passwd` is SUID-root.
- **Capabilities** (since 2.2) — fine-grained subset of "root" privileges (we cover this in the high school sheet)

```bash
$ id
uid=1000(stevie) gid=1000(stevie) groups=1000(stevie),4(adm),24(cdrom),...
$ cat /proc/$$/status | grep -E '^(Uid|Gid|Groups|CapEff)'
Uid:    1000    1000    1000    1000
Gid:    1000    1000    1000    1000
Groups: 4 24 27 30 46 110 1000
CapEff: 0000000000000000
```

### Deeper Dive: Virtual Memory

#### Demand paging in detail

A process that calls `malloc(1 GB)` does **not** allocate 1 GB of RAM. The C library calls `mmap()` (or extends the heap with `brk()`) and the kernel adds an entry to the process's VMA list. No physical page is touched until the process reads or writes that virtual range. On the first access:

1. CPU translates the virtual address → page table entry says "not present"
2. CPU traps to the kernel's page fault handler
3. Handler checks the VMA list: yes, this address is in a valid mapping
4. Handler allocates a physical page (or finds a zero page to share, COW-style)
5. Handler updates the page table entry
6. Handler returns; CPU retries the instruction; it succeeds this time

For file-backed mappings (`mmap` of a file), the kernel reads the file's pages into the page cache and maps them. This is why memory-mapping a large file does not use much RAM up front.

#### `/proc/<pid>/smaps` — per-mapping memory accounting

```bash
$ cat /proc/$$/smaps | head -30
55a4b3c00000-55a4b3c2b000 r--p 00000000 fd:00 1311265   /usr/bin/bash
Size:                172 kB
KernelPageSize:        4 kB
MMUPageSize:           4 kB
Rss:                 152 kB
Pss:                  19 kB
Shared_Clean:        152 kB
Shared_Dirty:          0 kB
Private_Clean:         0 kB
Private_Dirty:         0 kB
Referenced:          152 kB
Anonymous:             0 kB
LazyFree:              0 kB
AnonHugePages:         0 kB
ShmemPmdMapped:        0 kB
Shared_Hugetlb:        0 kB
Private_Hugetlb:       0 kB
Swap:                  0 kB
SwapPss:               0 kB
Locked:                0 kB
THPeligible:           0
VmFlags: rd mr mw me dw
```

Key fields:
- **Size** — virtual size of the mapping
- **Rss** — resident set size; how much is actually in RAM
- **Pss** — proportional set size; shared pages divided by sharer count (best metric for "this process's footprint")
- **Shared_Clean / Dirty** — pages used by other processes too
- **Private_Clean / Dirty** — pages this process alone uses
- **Swap** — pages currently swapped out
- **VmFlags** — abbreviated flags; `rd`=read, `wr`=write, `ex`=exec, `sh`=shared, `mr`=may read, `mw`=may write, `me`=may exec, `dw`=disabled write

#### Hugepages

A normal page is 4 KB. A hugepage is 2 MB or 1 GB. Fewer page-table entries, fewer TLB misses, much faster on memory-heavy workloads. Two mechanisms:

- **HugeTLB** — explicitly allocated via `mmap(MAP_HUGETLB)` or `/dev/hugepages`. Old, manual.
- **THP** (Transparent Huge Pages, since 2.6.38) — kernel automatically promotes 4 KB pages to 2 MB when possible. Configured via `/sys/kernel/mm/transparent_hugepage/enabled` (`always` / `madvise` / `never`).

```bash
$ cat /sys/kernel/mm/transparent_hugepage/enabled
[always] madvise never
$ grep -i huge /proc/meminfo
AnonHugePages:    819200 kB
ShmemHugePages:        0 kB
FileHugePages:         0 kB
HugePages_Total:       0
HugePages_Free:        0
HugePages_Rsvd:        0
HugePages_Surp:        0
Hugepagesize:       2048 kB
```

THP can hurt latency-sensitive workloads (compaction stalls). Some databases recommend `never`.

### Deeper Dive: CFS Internals

#### The red-black tree at the heart of CFS

A red-black tree is a self-balancing binary search tree with these invariants:
1. Every node is red or black.
2. Root is black.
3. Red nodes have black children.
4. Every path from root to a NULL has the same count of black nodes.

These rules force the tree's height to be O(log N). Insertions and deletions take O(log N) and the leftmost node (smallest key) is found in O(log N) — but the kernel caches it, so picking next is O(1).

```
Find next: leftmost-cached
Pick: O(1)

Insert (current task back into tree after running):
  walk from root, comparing vruntime
  insert as a leaf
  fix red-black properties: recolor and rotate as needed
  if new leftmost: update leftmost-cached
```

The kernel's red-black tree implementation lives in `lib/rbtree.c`. It is used in many other places in the kernel — VMA tracking per process, ext3 directory entries, network sockets — because of its predictable performance.

#### nice → weight table

```
nice  weight    1/weight
-20   88761     ~ 0.0000113
-15   29154     ~ 0.0000343
-10    9548     ~ 0.0001047
 -5    3121     ~ 0.0003204
  0    1024     ~ 0.0009766
  5     335     ~ 0.0029851
 10     110     ~ 0.0090909
 15      36     ~ 0.0277778
 19      15     ~ 0.0666667
```

Each nice level changes load roughly 1.25x. A nice-19 process gets about (15/1024) ≈ 1.5% as much CPU as a nice-0 process, all else equal.

#### Scheduling latency vs throughput

CFS has a target latency: `kernel.sched_latency_ns` (defaults around 6 ms). It tries to run every runnable task within that window. With many tasks, the per-task slice shrinks; below a floor (`sched_min_granularity_ns`, default 750 µs), the latency target is allowed to grow. Tune with `sysctl`:

```bash
$ sysctl kernel.sched_latency_ns kernel.sched_min_granularity_ns kernel.sched_wakeup_granularity_ns
kernel.sched_latency_ns = 24000000
kernel.sched_min_granularity_ns = 3000000
kernel.sched_wakeup_granularity_ns = 4000000
```

Note: these knobs were removed when EEVDF replaced CFS in 6.6 — the new scheduler computes its slices differently. Check `kernel.sched_*` on your kernel.

#### Completely Fair vs real-time

```
Priority high  +-----------------------+
              | SCHED_DEADLINE        |  EDF; uses CBS bandwidth server
              +-----------------------+
              | SCHED_FIFO            |  RT priority 1..99; runs to completion
              | SCHED_RR              |  RT priority 1..99; round-robin slice
              +-----------------------+
              | SCHED_NORMAL/OTHER    |  CFS / EEVDF; nice -20..+19
              | SCHED_BATCH           |  same as normal but yields more
              | SCHED_IDLE            |  lowest; only when nothing else wants
Priority low   +-----------------------+
```

A SCHED_FIFO task at priority 99 will preempt every other task. Mis-using it (infinite loop without yielding) can lock up your machine. Use `chrt` to inspect or change:

```bash
$ chrt -p 1234              # show task 1234's policy
$ chrt -f -p 50 1234        # set FIFO priority 50 on 1234 (root)
$ chrt -r -p 50 1234        # set RR priority 50
$ chrt -i -p 0  1234        # set IDLE
```

### Deeper Dive: VFS

#### The four pillars: superblock, inode, dentry, file

```
              Filesystem mount lifetime
              ==========================

  superblock_operations           inode_operations
  +----------------+              +----------------+
  | sb_alloc_inode |              | ino_create     |
  | sb_destroy_in. |              | ino_lookup     |
  | sb_write_inode |              | ino_link       |
  | sb_evict_inode |              | ino_unlink     |
  | sb_put_super   |              | ino_mkdir      |
  | sb_sync_fs     |              | ino_rename     |
  | sb_freeze_fs   |              | ino_setattr    |
  | sb_remount_fs  |              | ino_getattr    |
  +----------------+              +----------------+

  dentry_operations               file_operations
  +----------------+              +----------------+
  | d_revalidate   |              | f_open         |
  | d_hash         |              | f_release      |
  | d_compare      |              | f_read         |
  | d_delete       |              | f_write        |
  | d_iput         |              | f_llseek       |
  | d_dname        |              | f_mmap         |
  +----------------+              | f_fsync        |
                                  | f_ioctl        |
                                  | f_poll         |
                                  +----------------+
```

Each filesystem (ext4, xfs, btrfs, ...) provides its own implementation of these operation vectors. VFS calls the right one through indirect function pointers. Adding a new filesystem is, in principle, a matter of writing these.

#### Path traversal

When you call `open("/home/stevie/foo.txt", ...)`:

1. VFS starts at the root dentry (cached in `current->fs->root`).
2. `lookup` for `home` in the root inode → returns the home directory's inode and creates a dentry for it.
3. `lookup` for `stevie` in `/home` → returns the user's home inode.
4. `lookup` for `foo.txt` in `/home/stevie` → returns the file's inode.
5. Allocate a new `file` struct, point it at the inode, return an fd.

Each `lookup` is a filesystem-specific call. ext4 reads the directory's inode blocks, parses HTREE-indexed entries. btrfs walks its B-tree.

#### dcache — the dentry cache

Repeating those filesystem lookups every time would be slow. The kernel keeps a giant hash table — the **dcache** — keyed by (parent dentry, name). On a hit, the corresponding inode is found instantly, no disk I/O. The dcache is one of the largest consumers of slab memory on a busy system:

```bash
$ cat /proc/slabinfo | head -1; cat /proc/slabinfo | awk '$1 ~ /dentry|inode_cache/ {print}'
slabinfo - version: 2.1
dentry            612345 612345    192   42    2 : ...
inode_cache       412345 412345    600   54    8 : ...
ext4_inode_cache  234567 234567   1064   30    8 : ...
```

When memory is tight, the kernel shrinks these caches. You can force it: `echo 2 > /proc/sys/vm/drop_caches` (drop dcache + inodes; `1` drops page cache only; `3` drops both).

#### Filesystem feature comparison

```
                ext4     xfs      btrfs    f2fs     zfs (out-of-tree)
                ----     ---      -----    ----     ------------------
journaling       Y        Y       N (CoW)  Y        N (CoW + ZIL)
COW              N        N       Y        N        Y
snapshots        N        N       Y        N        Y
checksums       (md)      Y       Y        Y        Y (always)
encryption       Y(*)     N       Y(*)     Y(*)     Y
quotas           Y        Y       Y        Y        Y
max file size    16TB    8EB      16EB     16TB     16EB
max fs size      1EB     8EB      16EB     16TB     256ZB
fragmentation   low      low      can grow low      low
flash-aware      N        N       N        Y        N
flagship distro  Ubuntu   RHEL    SUSE     Android  illumos/FreeBSD
```

(*) via fscrypt (kernel-native encryption).

### Deeper Dive: Networking

#### sk_buff — the kernel's universal packet container

Every packet inside the kernel travels in a `struct sk_buff` (socket buffer). It carries:
- pointer to the packet data
- pointers to the L2/L3/L4 headers within the data
- routing/destination metadata
- flags for offload, checksum, marks
- a reference count

When a packet enters the network stack, an sk_buff wraps it. Each layer pushes/pops headers (just adjusting pointers, no copy). When the packet leaves the kernel, the sk_buff is freed. This pointer-only design is the reason Linux networking is fast.

#### netfilter hook points

```
                    +---------+
                    |INCOMING |
                    +---------+
                         |
                         v
                  +-------------+
                  | PREROUTING  |  <-- DNAT happens here (mangle/nat tables)
                  +-------------+
                         |
                  routing decision
                  /              \
                 /                \
                v                  v
        +---------------+      +---------------+
        |   for me?     |      |   forward     |
        +---------------+      +---------------+
                |                       |
                v                       v
        +---------------+      +---------------+
        |    INPUT      |      |    FORWARD    |
        +---------------+      +---------------+
                |                       |
                v                       v
        +---------------+               |
        | LOCAL PROCESS |               |
        +---------------+               |
                |                       |
                v                       |
        +---------------+               |
        |    OUTPUT     |               |
        +---------------+               |
                \                      /
                 \                    /
                  v                  v
                  +-------------+
                  | POSTROUTING |  <-- SNAT/MASQUERADE happens here
                  +-------------+
                         |
                         v
                    +---------+
                    |OUTGOING |
                    +---------+
```

Each hook can have rules from the `filter`, `nat`, `mangle`, and `raw` tables. nftables (modern) and iptables (legacy) just register rules at these hook points.

```bash
$ sudo nft list ruleset                  # modern (nftables)
$ sudo iptables -L -v -n                 # legacy (iptables)
$ sudo iptables -t nat -L -v -n          # the NAT table
```

#### Socket family / type matrix

```
                 Stream (SOCK_STREAM)   Datagram (SOCK_DGRAM)   Raw (SOCK_RAW)
                 ----------------------  ----------------------  ----------------
  AF_INET (IPv4)         TCP                    UDP                  ICMP, etc.
  AF_INET6 (IPv6)        TCP                    UDP                  ICMPv6
  AF_UNIX (local IPC)    Unix stream            Unix datagram        not applicable
  AF_PACKET              n/a                    raw L2               raw L2 incl. headers
  AF_NETLINK             n/a                    kernel<->user        kernel events
  AF_XDP (XDP sockets)   n/a                    n/a                  zero-copy from XDP
  AF_VSOCK               yes                    yes                  host<->guest VM
  AF_BLUETOOTH           yes                    yes                  RFCOMM/L2CAP
```

`AF_UNIX` is the most underrated. It's how `systemd`, `dbus`, the X server, journald, and Docker talk to the world.

#### TCP state machine (compressed)

```
   CLOSED ---listen()--> LISTEN
                            |
                            v  SYN received -> send SYN+ACK
                         SYN-RCVD ---ACK---> ESTABLISHED
                            |
   CLOSED ---connect()-->SYN-SENT
                            |
                            v  SYN+ACK received -> send ACK
                         ESTABLISHED                       <- normal data flow
                            |
                            v  close()
                         FIN-WAIT-1
                            |
                            v  ACK received
                         FIN-WAIT-2
                            |
                            v  FIN received -> send ACK
                         TIME-WAIT (2*MSL)  <- sit here; ensure last ACK reached peer
                            |
                            v
                         CLOSED
```

Watching states live:

```bash
$ ss -tan state established | head
$ ss -tan state time-wait | wc -l
$ ss -s
Total: 567
TCP:   234 (estab 12, closed 201, orphaned 0, timewait 198)
```

#### Congestion control

The default congestion control algorithm has changed over the years:
- **Reno** — the original, packet-loss-based
- **CUBIC** — default since 2.6.19; cubic curve, scales to high BDP
- **BBR** — Google, since 4.9; bandwidth+RTT model, ignores loss as a signal
- **BBR v2/v3** — successor to BBR

```bash
$ sysctl net.ipv4.tcp_available_congestion_control
net.ipv4.tcp_available_congestion_control = reno cubic bbr
$ sysctl net.ipv4.tcp_congestion_control
net.ipv4.tcp_congestion_control = cubic
$ sudo sysctl -w net.ipv4.tcp_congestion_control=bbr
```

### Deeper Dive: Kernel Modules

#### Anatomy of a module

A minimal kernel module:

```c
#include <linux/module.h>
#include <linux/kernel.h>
#include <linux/init.h>

static int __init hello_init(void) {
    printk(KERN_INFO "hello: loaded\n");
    return 0;
}

static void __exit hello_exit(void) {
    printk(KERN_INFO "hello: unloaded\n");
}

module_init(hello_init);
module_exit(hello_exit);
MODULE_LICENSE("GPL");
MODULE_AUTHOR("you");
MODULE_DESCRIPTION("A trivial module");
```

`module_init` registers `hello_init` as the function called when the module is loaded; `module_exit` registers `hello_exit` for unload. `MODULE_LICENSE("GPL")` is mandatory — non-GPL modules cannot use most exported kernel symbols (those marked `EXPORT_SYMBOL_GPL`).

Build with a Makefile:

```make
obj-m += hello.o
all:
	make -C /lib/modules/$(shell uname -r)/build M=$(PWD) modules
clean:
	make -C /lib/modules/$(shell uname -r)/build M=$(PWD) clean
```

```bash
$ make
$ sudo insmod hello.ko
$ dmesg | tail -1
[12345.678] hello: loaded
$ sudo rmmod hello
$ dmesg | tail -1
[12346.789] hello: unloaded
```

#### Module parameters

```c
#include <linux/moduleparam.h>
static int howmany = 1;
module_param(howmany, int, 0644);
MODULE_PARM_DESC(howmany, "how many greetings");
```

Now you can:

```bash
$ sudo insmod hello.ko howmany=3
$ ls /sys/module/hello/parameters/
howmany
$ cat /sys/module/hello/parameters/howmany
3
```

#### modprobe.conf (modules.d) tricks

`/etc/modprobe.d/*.conf` lets you:
- Blacklist a module: `blacklist nouveau`
- Set load-time parameters: `options snd_hda_intel power_save=10`
- Define aliases: `alias eth0 e1000e`
- Run a command on load: `install foo /sbin/modprobe --ignore-install foo && /usr/local/bin/post-foo`

```bash
$ ls /etc/modprobe.d/
alsa-base.conf  blacklist.conf  fbdev-blacklist.conf  ...
```

#### Where do modules live?

```
/lib/modules/$(uname -r)/
├── kernel/
│   ├── arch/x86/
│   ├── crypto/
│   ├── drivers/
│   │   ├── acpi/
│   │   ├── block/
│   │   ├── bluetooth/
│   │   ├── gpu/drm/
│   │   ├── net/ethernet/
│   │   ├── usb/
│   │   └── ...
│   ├── fs/
│   │   ├── ext4/
│   │   ├── xfs/
│   │   ├── btrfs/
│   │   └── ...
│   ├── net/
│   ├── sound/
│   └── ...
├── modules.dep         <- dependency map (built by depmod)
├── modules.alias       <- alias -> module map
├── modules.symbols     <- symbol -> module map
└── modules.builtin     <- modules built into the kernel (no .ko needed)
```

After installing a new kernel or copying in custom .ko files, run `sudo depmod -a` to rebuild the maps.

## Common Errors and Fixes

These are the verbatim messages you'll see and how to resolve each.

### "Operation not permitted" on mount/sysctl/ptrace

```
mount: /mnt/foo: only root can do that.
```

Fix: `sudo`. Or check if you're inside a user namespace where some operations require capabilities.

### "Permission denied" reading /proc/<pid>/status of another user's process

```
$ cat /proc/12345/status
cat: /proc/12345/status: Permission denied
```

This is `kernel.yama.ptrace_scope` on most distros — protects against process introspection. `sudo` works, or relax: `sudo sysctl -w kernel.yama.ptrace_scope=0`.

### "Killed" with no further explanation

```
$ ./big_program
Killed
```

OOM killer or `ulimit`. Check:

```bash
$ dmesg | grep -iE 'killed|oom|out of memory' | tail -5
$ ulimit -a
```

### "execve: Exec format error"

```
$ ./mystery
bash: ./mystery: cannot execute binary file: Exec format error
```

The binary's architecture doesn't match the kernel's, or the file is corrupted, or it's missing a `#!` shebang. Try `file ./mystery`.

### "execve: No such file or directory" — but the file is right there!

```
$ ls -la ./script
-rwxr-xr-x 1 stevie stevie 123 Apr 27 12:34 ./script
$ ./script
bash: ./script: No such file or directory
```

The shebang line points to an interpreter that doesn't exist. Common: `#!/bin/python` instead of `#!/usr/bin/env python3`. The kernel's exec is reporting the missing *interpreter*, not the script itself.

### "modprobe: ERROR: could not insert 'foo': Unknown symbol in module"

The module references a symbol the kernel doesn't export, or the kernel was built without a matching feature. `dmesg` shows the missing symbol. If you compiled the module against a different kernel version, rebuild against the running kernel: ensure `/lib/modules/$(uname -r)/build/` symlinks to the matching kernel headers package.

### "modprobe: FATAL: Module foo not found in directory /lib/modules/..."

Either the module isn't installed, or `depmod` hasn't been run after a copy. `sudo depmod -a` and try again. Or the module was renamed in a new kernel version.

### "Failed to mount /sysroot: No such device" during boot

Initramfs couldn't find your root filesystem. Common causes: storage driver missing from initramfs (rebuild with `update-initramfs -u` or `dracut -f`); UUID changed and `/etc/fstab` or kernel cmdline still references the old one; LVM/LUKS hooks missing.

### "FATAL: kernel too old"

Your binary was linked against a glibc that requires a newer kernel ABI than you're running. `ldd --version` shows the glibc version. Either run on a newer kernel or recompile against an older glibc.

### "fork: Cannot allocate memory"

Confusing — the kernel might have plenty of memory but you've hit `RLIMIT_NPROC` or `vm.overcommit_memory` denied. Check:

```bash
$ ulimit -u                            # max user processes
$ cat /proc/sys/vm/overcommit_memory   # 0/1/2 (heuristic / always / never)
$ cat /proc/sys/vm/overcommit_ratio
$ cat /proc/sys/kernel/pid_max         # PIDs are exhausted?
```

### "device or resource busy" on umount

```
$ sudo umount /mnt/usb
umount: /mnt/usb: target is busy
```

A process has an open file or cwd inside that mount. Find with:

```bash
$ sudo lsof +D /mnt/usb
$ sudo fuser -vm /mnt/usb
$ sudo umount -l /mnt/usb       # lazy unmount: detach now, free when refs drop
```

### "Address already in use" on bind()

A previous instance of the server is still in TIME_WAIT, or another process owns the port. Either wait ~60s, set `SO_REUSEADDR` in your code, or use a different port:

```bash
$ sudo ss -tlnp | grep ':8080'
```

### "Too many open files"

```
$ ulimit -n
1024
$ ulimit -n 65536
```

Process-level limit. Also check the system-wide limit `/proc/sys/fs/file-max` and per-user `/etc/security/limits.conf`. systemd services use `LimitNOFILE=` in their unit file.

### "kernel: BUG:" in dmesg

Hardware fault, kernel bug, or driver bug. Capture full dmesg, the call trace, and report. Common follow-on: machine becomes unstable. Reboot into a known-good kernel and consider downgrading the suspect module.

### "Out of memory: Killed process"

Detailed OOM kill receipt:

```
[1234.567] Out of memory: Killed process 4567 (firefox) total-vm:8123456kB, anon-rss:5234567kB, file-rss:0kB, shmem-rss:0kB, UID:1000 pgtables:11264kB oom_score_adj:0
```

Reduce memory pressure (close stuff), add swap, raise the OOM score adjustment for processes you want to protect (`echo -100 > /proc/<pid>/oom_score_adj`), or tune `vm.overcommit_memory` and `vm.overcommit_ratio` for stricter accounting.

## More Hands-On Exploration

These are paste-and-runnable tours of specific subsystems. They build on what you've already done.

### Tour 1: Watch fork happen

In one terminal:

```bash
$ sudo bpftrace -e 'tracepoint:sched:sched_process_fork { printf("%-16s %-6d -> %-6d\n", comm, args->parent_pid, args->child_pid); }'
```

In another terminal: type `ls`, `cat`, `firefox` etc. Watch the first terminal print parent/child PIDs in real time. Each one is a fork in your shell.

(If `bpftrace` isn't installed: `sudo apt install bpftrace` on Debian/Ubuntu.)

### Tour 2: Map a process's address space the long way

```bash
$ pmap -X $$ | head -20
```

`pmap -X` is a friendlier `cat /proc/<pid>/maps` with sizes and per-region RSS.

### Tour 3: Find the process eating disk I/O

```bash
$ sudo iotop -ao
```

`-a` accumulated, `-o` only those doing I/O. Press `q` to quit.

### Tour 4: System call statistics, per process

```bash
$ strace -c -p $(pgrep -n bash)
^C
% time     seconds  usecs/call     calls    errors syscall
------ ----------- ----------- --------- --------- ----------------
 50.00    0.000123          12        10           read
 30.00    0.000074           7        10           write
 20.00    0.000049           5         9           rt_sigprocmask
------ ----------- ----------- --------- --------- ----------------
100.00    0.000246                    29           total
```

Detach with Ctrl-C; the table summarises which syscalls dominated.

### Tour 5: Find files opened by every process for a path

```bash
$ sudo lsof /etc/passwd
COMMAND     PID  USER   FD   TYPE DEVICE SIZE/OFF    NODE NAME
systemd       1  root  mem    REG    8,2     2841 4194309 /etc/passwd
NetworkMa   456  root  mem    REG    8,2     2841 4194309 /etc/passwd
...
```

`mem` means mmap'd, not necessarily an open fd.

### Tour 6: Which physical CPU core is each task scheduled on?

```bash
$ ps -eo pid,psr,cmd | head -10
    PID PSR CMD
      1   3 /sbin/init splash
      2   0 [kthreadd]
      3   0 [rcu_gp]
   1246   2 -bash
   ...
```

`PSR` is the most-recent processor.

### Tour 7: Watch context switches per CPU

```bash
$ pidstat -w 1 5
Linux 6.5.0-21-generic ...
12:34:56  UID    PID   cswch/s nvcswch/s  Command
12:34:57    0      1      0.99      0.00  systemd
12:34:57 1000   1246      4.95      0.00  bash
...
```

`cswch/s` voluntary, `nvcswch/s` involuntary (preempted).

### Tour 8: Find which kernel module owns a device

```bash
$ lspci -k | head -30
00:00.0 Host bridge: Intel Corporation Device 9b73 (rev 0c)
        Subsystem: Lenovo ...
        Kernel driver in use: skl_uncore
        Kernel modules: skl_uncore
00:02.0 VGA compatible controller: Intel Corporation UHD Graphics 630
        Subsystem: Lenovo ...
        Kernel driver in use: i915
        Kernel modules: i915
...
```

`-k` shows the kernel driver bound to each device.

### Tour 9: Trace one specific syscall across the system

```bash
$ sudo bpftrace -e 'tracepoint:syscalls:sys_enter_openat { printf("%-16s open %s\n", comm, str(args->filename)); }' | head -20
```

Watch every file every process opens, system-wide. Useful when you don't know which process is touching a path.

### Tour 10: Check the size of all kernel structures of interest

```bash
$ cat /proc/slabinfo | sort -k2 -n -r | head -10
```

Largest slab caches by object count. Often dentry, inode, kmalloc-* lead.

### Tour 11: List signal handlers a process has installed

```bash
$ cat /proc/$$/status | grep -E '^Sig'
SigQ:   1/63234
SigPnd: 0000000000000000      <- pending
SigBlk: 0000000000000000      <- blocked
SigIgn: 0000000000384004      <- ignored (bitmask)
SigCgt: 000000004b813eff      <- caught (custom handler)
```

Each bit is a signal number. Bash handles SIGINT, SIGTERM, SIGCHLD etc. — that's why Ctrl-C stops the foreground job rather than killing your shell.

### Tour 12: Boot timeline

```bash
$ systemd-analyze
$ systemd-analyze blame | head
$ systemd-analyze critical-chain
$ systemd-analyze plot > /tmp/boot.svg && xdg-open /tmp/boot.svg
```

(Last command requires a desktop.)

### Tour 13: Inspect dynamic kernel parameters

```bash
$ sysctl -a 2>/dev/null | wc -l
1234
$ sysctl -a 2>/dev/null | grep -E 'swappiness|overcommit|panic|tcp_congest'
kernel.panic = 0
net.ipv4.tcp_congestion_control = cubic
vm.overcommit_memory = 0
vm.overcommit_ratio = 50
vm.swappiness = 60
```

Anything in `/proc/sys/...` is a sysctl. Persist via `/etc/sysctl.d/*.conf`.

### Tour 14: See which kernel features were compiled in

```bash
$ zcat /proc/config.gz | grep CONFIG_PREEMPT
CONFIG_PREEMPT_BUILD=y
CONFIG_PREEMPT_NONE=n
CONFIG_PREEMPT_VOLUNTARY=n
CONFIG_PREEMPT=y
CONFIG_PREEMPT_COUNT=y
CONFIG_PREEMPTION=y
CONFIG_PREEMPT_DYNAMIC=y
CONFIG_PREEMPT_RCU=y
```

Not every distro enables `/proc/config.gz`. Alternative: `cat /boot/config-$(uname -r)`.

### Tour 15: Live process-tree explorer

```bash
$ htop
```

`F4` filter, `F5` tree view, `F6` sort, `F9` send signal. Highly recommended over `top`.

## Where to Go Next

- `cs ramp-up linux-kernel-high-school` — privilege rings, IRQs at the CPU level, syscall ABI, cgroups v2, namespaces, eBPF, KASLR, capabilities, audit
- `cs fundamentals linux-kernel-internals` — dense reference of subsystem internals
- `cs system strace` — trace syscalls of any process
- `cs system gdb` — debug user programs and crash cores
- `cs system perf` — sample CPU events to find hot code
- `cs system systemd` — units, targets, journal, scopes, slices
- `cs kernel-tuning sysctl` — `/proc/sys` tuning catalog
- `cs kernel-tuning cgroups` — CPU/memory/IO accounting and limits

## See Also

- `ramp-up/linux-kernel-eli5`
- `ramp-up/linux-kernel-high-school`
- `fundamentals/linux-kernel-internals`
- `fundamentals/how-computers-work`
- `system/strace`
- `system/gdb`
- `system/systemd`
- `kernel-tuning/sysctl`
- `kernel-tuning/cgroups`

## References

- kernel.org Documentation/admin-guide/ — official kernel admin docs
- "Linux Kernel Development" by Robert Love (3rd ed., 2010) — book
- "Understanding the Linux Kernel" by Bovet & Cesati (3rd ed.) — denser book
- man 2 fork — fork(2) syscall
- man 2 execve — execve(2) syscall
- man 2 clone — clone(2) syscall, the primitive under fork and pthread_create
- man 7 sched — sched(7) overview of scheduling policies
- man 7 capabilities — capabilities(7) overview
- man 7 cgroups — cgroups(7) overview
- man 5 proc — proc(5) reference for /proc files
- man 5 sysfs — sysfs(5) reference for /sys
- man 8 systemd — systemd(8) init system
- LWN.net articles on CFS: "An overview of the Linux scheduler" series
- LWN.net articles on EEVDF: "An EEVDF CPU scheduler for Linux" (2023)
- kernel.org Documentation/scheduler/ — `sched-design-CFS.rst`, `sched-deadline.rst`
- kernel.org Documentation/networking/ — networking subsystem docs
- kernel.org Documentation/filesystems/vfs.rst — VFS overview
- kernel.org Documentation/admin-guide/sysctl/ — sysctl reference
- kernel.org Documentation/admin-guide/kernel-parameters.txt — boot cmdline parameters

— End of Part 2: Middle School —
