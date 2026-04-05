# Linux Kernel Internals (From User Space to Ring 0)

A tiered guide to how the Linux kernel works.

## ELI5

Imagine a big apartment building. You live on one of the floors. You want to
turn on the lights, get water from the tap, or use the elevator. But you
cannot just walk into the basement and flip switches yourself -- that would be
dangerous.

Instead, there is a **building manager** who lives in the basement. The
building manager is the **kernel**.

The building manager has three big jobs:

1. **Decides who gets what.** There are 100 apartments but only so much hot
   water. The manager decides how to share it fairly so nobody gets burned and
   nobody freezes.

2. **Keeps everyone safe.** You cannot walk into your neighbor's apartment.
   You cannot mess with the wiring. The manager enforces the rules so one bad
   tenant cannot wreck the building for everyone.

3. **Talks to all the hardware.** The plumbing, the electrical panel, the
   elevator motor -- the manager knows how to operate all of it. You just say
   "I need water" and the manager turns the right valve.

When you run a program on a computer, it lives "upstairs" (user space). When
it needs something from the hardware -- reading a file, sending a network
packet, allocating memory -- it asks the building manager (the kernel) through
a little request window called a **system call**.

The kernel never lets you downstairs. It takes your request, does the work,
and hands back the result. That separation is what keeps everything stable.

## Middle School

### Two Worlds: User Space vs Kernel Space

A computer's memory is split into two zones:

- **User space** -- where your programs run (browsers, games, terminals). Programs
  here cannot touch hardware directly.
- **Kernel space** -- where the operating system runs. It has full access to
  every piece of hardware: CPU, RAM, disks, network cards.

### System Calls: Asking Permission

When a program needs something from the hardware, it makes a **system call**
(syscall). Think of it as filling out a form and sliding it through a window:

```
Program: "I need to open the file /etc/hostname"
  → syscall: open("/etc/hostname", O_RDONLY)
  → kernel checks permissions, opens the file, hands back a file descriptor
  → Program: "Thanks, here's my number: fd=3"
```

Common syscalls: `read`, `write`, `open`, `close`, `fork`, `exec`, `mmap`, `ioctl`.

### Processes and Threads

- A **process** is a running program. It gets its own memory, its own file
  descriptors, its own identity (PID).
- A **thread** is a lightweight worker inside a process. Threads share memory
  but each gets its own execution path.
- The kernel keeps a list of every process in a structure called the
  **task list** (`struct task_struct` in the source code).

### File Systems: The Filing Cabinet

Everything in Linux is a file (or pretends to be). The kernel provides a
**Virtual File System (VFS)** layer that makes all storage look the same,
whether it is an SSD (ext4), a network drive (NFS), or a USB stick (FAT32).

### Device Drivers: Translators

Hardware speaks its own language. A network card speaks different commands than
a GPU. **Device drivers** are kernel modules that translate between the
kernel's standard interface and each device's proprietary protocol.

## High School

### Process Scheduler — CFS

The CPU can only run one thing at a time (per core). The **Completely Fair
Scheduler (CFS)** decides which process runs next:

- CFS tracks how much CPU time each process has used via a **virtual runtime**
  (`vruntime`).
- The process with the lowest `vruntime` runs next -- it has been "treated
  most unfairly" so far.
- CFS uses a **red-black tree** (self-balancing binary search tree) to find
  the minimum `vruntime` in O(log n).
- Nice values (-20 to +19) adjust the weight: lower nice = more CPU share.

```bash
# See scheduler info for a process
cat /proc/<pid>/sched

# Change scheduling priority
nice -n -5 ./my_program
renice -n 10 -p <pid>

# See all running processes with priorities
ps -eo pid,ni,pri,comm --sort=-pri
```

### Virtual Memory and Page Tables

Each process thinks it has the entire address space to itself. The kernel
maintains **page tables** that map virtual addresses to physical RAM:

```
Virtual Address → Page Table → Physical Frame
0x7fff00001000 → PTE entry → 0x1a3f000
```

- Pages are typically 4 KB.
- The **MMU** (Memory Management Unit) hardware does the translation.
- The **TLB** (Translation Lookaside Buffer) caches recent translations.
- When a page is not in RAM, a **page fault** occurs and the kernel loads it
  from disk (swap).

### The VFS Layer

VFS is the abstraction that lets you `open()` any file regardless of the
underlying filesystem:

```
Application
    ↓ open("/data/file.txt")
   VFS (generic interface)
    ↓
ext4 / XFS / btrfs / NFS (specific implementation)
    ↓
Block I/O layer
    ↓
Disk driver → Hardware
```

### Interrupt Handling

Hardware devices signal the CPU via **interrupts** (IRQs):

1. Device raises an interrupt (e.g., network card received a packet).
2. CPU stops current work, jumps to the kernel's interrupt handler.
3. Handler does minimal work (top half), schedules deferred work (bottom half /
   softirq / tasklet).
4. CPU returns to the interrupted process.

```bash
# View interrupt counts per CPU
cat /proc/interrupts

# View softirq activity
cat /proc/softirqs
```

### Kernel Modules

The kernel is not a single monolithic blob you have to recompile. You can load
and unload pieces at runtime:

```bash
# List loaded modules
lsmod

# Load a module
sudo modprobe br_netfilter

# Remove a module
sudo modprobe -r br_netfilter

# Get module info
modinfo br_netfilter

# View module parameters
ls /sys/module/br_netfilter/parameters/
```

### /proc and /sys — The Kernel's Dashboard

```bash
# Process info
ls /proc/<pid>/
cat /proc/<pid>/status     # memory, state, threads
cat /proc/<pid>/maps       # virtual memory map
cat /proc/<pid>/fd/        # open file descriptors

# System-wide info
cat /proc/cpuinfo          # CPU details
cat /proc/meminfo          # memory statistics
cat /proc/vmstat           # virtual memory statistics
cat /proc/loadavg          # load averages

# /sys — structured hardware/driver info
ls /sys/class/net/         # network interfaces
ls /sys/block/             # block devices
cat /sys/class/thermal/thermal_zone0/temp  # CPU temperature
```

### strace — Watching System Calls

```bash
# Trace all syscalls of a running process
strace -p <pid>

# Trace a command from start
strace ls -la /tmp

# Summary of syscall counts and timing
strace -c ls -la /tmp

# Filter specific syscalls
strace -e trace=open,read,write ls /tmp

# Follow child processes (fork/clone)
strace -f ./my_server
```

## College

### Memory Management Internals

#### Buddy Allocator

The kernel allocates physical page frames using the **buddy allocator**:

- Free memory is organized into lists of blocks: 1 page, 2 pages, 4 pages, ...
  up to 2^(MAX_ORDER-1) pages (typically 2^10 = 4 MB).
- To allocate N pages, find the smallest block >= N, split recursively.
- To free, merge with the "buddy" block if it is also free (coalesce).
- O(log n) allocation and freeing. Minimizes external fragmentation.

```bash
# View buddy allocator state per zone
cat /proc/buddyinfo

# Example output — columns are order 0,1,2,...,10
# Node 0, zone   Normal   4096  2048  1024   512   256   128    64    32    16     8     4
```

#### Slab Allocator (SLUB)

For small kernel objects (inodes, dentries, socket buffers), allocating full
pages is wasteful. The **slab allocator** (modern Linux uses SLUB) carves
pages into fixed-size object caches:

```bash
# View slab cache statistics
cat /proc/slabinfo
# Or use the friendlier tool
slabtop

# Key caches to watch
# dentry          — directory entry cache
# inode_cache     — inode cache
# task_struct     — process descriptors
# kmalloc-*       — general-purpose allocations
```

#### Page Cache

The kernel caches disk contents in unused RAM. Every `read()` checks the page
cache first:

```bash
# See page cache usage
free -h          # "buff/cache" column
cat /proc/meminfo | grep -E "Cached|Buffers|Active|Inactive"

# Drop page cache (careful in production)
echo 3 > /proc/sys/vm/drop_caches
```

### Process Lifecycle

```
fork()          — create child (copy-on-write clone of parent)
  ↓
exec()          — replace child's memory with new program
  ↓
[process runs]
  ↓
exit()          — process terminates, becomes zombie
  ↓
wait()          — parent collects exit status, zombie is reaped
```

Key details:

- `fork()` uses **copy-on-write (COW)**: pages are shared until either
  process writes, then the kernel copies the page.
- `clone()` is the underlying syscall -- `fork()` and `pthread_create()` both
  call `clone()` with different flags.
- Zombies (`Z` state) exist between `exit()` and `wait()`. A process that
  never reaps its children leaks zombie entries.

### Scheduler Internals

- Each CPU has a **runqueue** (`struct rq`) containing the red-black tree of
  runnable tasks.
- The scheduler runs `pick_next_task()` to select the next task from the tree.
- **Load balancing**: the kernel periodically migrates tasks between CPUs to
  balance load. Domains: SMT (hyperthreads) -> cores -> sockets -> NUMA nodes.
- **Scheduling classes** (priority order): stop > deadline > realtime > CFS > idle.

```bash
# View scheduler domains
find /proc/sys/kernel/ -name "sched_*" | sort

# Key tunables
sysctl kernel.sched_min_granularity_ns     # minimum timeslice
sysctl kernel.sched_latency_ns             # target latency period
sysctl kernel.sched_migration_cost_ns      # migration cost threshold
sysctl kernel.sched_nr_migrate             # max tasks to migrate at once
```

### VFS Internals: Inode, Dentry, Superblock

The VFS layer uses three core data structures:

- **Superblock** (`struct super_block`) -- one per mounted filesystem. Holds
  metadata: block size, max file size, filesystem operations.
- **Inode** (`struct inode`) -- one per file/directory. Holds metadata:
  permissions, size, timestamps, data block pointers. Inodes are cached in the
  inode cache.
- **Dentry** (`struct dentry`) -- one per path component. Maps names to inodes.
  Cached in the dentry cache (dcache). Lookup: hash the name, walk the dcache.

```bash
# See inode and dentry cache pressure
sysctl vm.vfs_cache_pressure    # default 100; lower = keep more cache

# Count cached dentries and inodes
cat /proc/sys/fs/dentry-state
cat /proc/sys/fs/inode-nr
```

### Network Stack and sk_buff

Every network packet in the kernel is represented by `struct sk_buff`:

```
Packet arrives at NIC
  → Driver allocates sk_buff, fills it with packet data
  → Passes up through: L2 (ethernet) → L3 (ip) → L4 (tcp/udp)
  → Each layer reads/strips headers by adjusting pointers (no copies)
  → Socket receive queue → application read()
```

**Netfilter hooks** are checkpoints where the kernel can filter, mangle, or
redirect packets:

```
PREROUTING → INPUT → [local process]
PREROUTING → FORWARD → POSTROUTING
[local process] → OUTPUT → POSTROUTING
```

```bash
# View netfilter rules
iptables -L -n -v
nft list ruleset

# View socket buffer tuning
sysctl net.core.rmem_max
sysctl net.core.wmem_max
sysctl net.ipv4.tcp_rmem
sysctl net.ipv4.tcp_wmem
```

### eBPF Subsystem

eBPF (extended Berkeley Packet Filter) lets you run sandboxed programs inside
the kernel without modifying kernel source or loading kernel modules:

```bash
# List loaded BPF programs
bpftool prog list

# List BPF maps (shared data structures)
bpftool map list

# Attach a tracepoint program
bpftool prog attach <id> tracepoint <category> <event>

# View BPF program stats
bpftool prog show id <id>

# Common attach points
# - kprobes (function entry/exit)
# - tracepoints (static kernel events)
# - XDP (network packets at driver level)
# - tc (traffic control)
# - cgroup (process group events)
# - LSM (security module hooks)
```

The verifier ensures every BPF program terminates, accesses only valid memory,
and cannot crash the kernel.

### cgroups v2

Control groups limit, account, and isolate resource usage for groups of
processes:

```bash
# View cgroup hierarchy (unified v2)
ls /sys/fs/cgroup/

# Key controllers
# cpu        — CPU bandwidth and weight
# memory     — memory limits and accounting
# io         — block I/O bandwidth
# pids       — process count limits

# Create a cgroup and set memory limit
mkdir /sys/fs/cgroup/mygroup
echo 512M > /sys/fs/cgroup/mygroup/memory.max
echo $$ > /sys/fs/cgroup/mygroup/cgroup.procs

# View current usage
cat /sys/fs/cgroup/mygroup/memory.current
cat /sys/fs/cgroup/mygroup/cpu.stat
```

### Namespaces

Namespaces provide process-level isolation (the foundation of containers):

| Namespace | Isolates | Flag |
|:---|:---|:---|
| PID | Process IDs | `CLONE_NEWPID` |
| NET | Network stack (interfaces, routes, iptables) | `CLONE_NEWNET` |
| MNT | Mount points | `CLONE_NEWNS` |
| UTS | Hostname, domain name | `CLONE_NEWUTS` |
| IPC | System V IPC, POSIX message queues | `CLONE_NEWIPC` |
| USER | UIDs/GIDs | `CLONE_NEWUSER` |
| CGROUP | Cgroup root | `CLONE_NEWCGROUP` |
| TIME | System clocks (since 5.6) | `CLONE_NEWTIME` |

```bash
# View namespaces for a process
ls -la /proc/<pid>/ns/

# Enter a container's namespaces
nsenter --target <pid> --mount --uts --ipc --net --pid

# Create a new network + PID namespace
unshare --net --pid --fork --mount-proc bash
```

### Futex — Userspace Synchronization

`futex` (fast userspace mutex) is the kernel primitive behind `pthread_mutex`,
`sem_wait`, and Go's `sync.Mutex` (on Linux):

- **Fast path** (no contention): atomic compare-and-swap in userspace, no
  syscall needed.
- **Slow path** (contention): `futex(FUTEX_WAIT)` puts the thread to sleep in
  the kernel; `futex(FUTEX_WAKE)` wakes it.
- The kernel maintains a hash table of wait queues, keyed by the futex address.

```bash
# Trace futex calls for a process
strace -e trace=futex -p <pid>

# View contention in Go programs
GODEBUG=schedtrace=1000 ./my_program
```

## Tips

- Start exploring with `strace` and `/proc` -- they reveal what the kernel is
  actually doing, not what documentation says it should do.
- Use `perf` to profile kernel functions: `perf top -g` shows live kernel
  hotspots.
- Read kernel source via https://elixir.bootlin.com/ -- full cross-referenced
  Linux source searchable by version.
- When debugging performance, check `/proc/vmstat`, `/proc/interrupts`, and
  `/proc/softirqs` before reaching for complex tools.
- Kernel parameters in `/proc/sys/` are live -- changes take effect immediately
  but do not survive reboot unless written to `/etc/sysctl.d/`.
- Use `dmesg -w` to watch kernel messages in real time.
- Containers are not VMs -- they are just processes with namespaces and cgroups.
  Understanding the kernel primitives demystifies Docker and Kubernetes.

## See Also

- memory-tuning
- cpu-scheduler-tuning
- io-scheduler-tuning
- network-stack-tuning
- ebpf
- cgroups
- namespaces
- strace
- perf

## References

- Linux Kernel Documentation: https://docs.kernel.org/
- "Understanding the Linux Kernel" by Bovet & Cesati (O'Reilly)
- "Linux Kernel Development" by Robert Love (Addison-Wesley)
- Bootlin Elixir Cross-Reference: https://elixir.bootlin.com/
- Brendan Gregg's Linux Performance: https://www.brendangregg.com/linuxperf.html
- kernel.org man pages: https://man7.org/linux/man-pages/
- eBPF documentation: https://ebpf.io/what-is-ebpf/
- LWN.net kernel articles: https://lwn.net/Kernel/
