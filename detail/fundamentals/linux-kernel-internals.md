# Linux Kernel Internals — Architecture, Theory, and Design Analysis

> *Deep dive into Linux kernel architecture: monolithic vs microkernel tradeoffs, the loadable module hybrid, memory management theory, scheduler complexity analysis, and VFS abstraction layer design. From first principles to implementation details.*

---

## Prerequisites

- Familiarity with operating system concepts (processes, memory, filesystems)
- Understanding of C data structures (linked lists, trees, hash tables)
- Basic knowledge of CPU architecture (rings, interrupts, caches, virtual memory)

## Complexity

| Topic | Analysis Type | Key Metric |
|:---|:---|:---|
| Monolithic vs Microkernel | Architecture tradeoff | IPC cost, context switch overhead |
| Loadable Modules | Design pattern | Symbol resolution, reference counting |
| Buddy Allocator | Worst-case O(log n) | External fragmentation ratio |
| Slab Allocator | Amortized O(1) | Internal fragmentation, cache coloring |
| CFS Scheduler | O(log n) pick, O(1) update | Weighted fairness deviation |
| EEVDF Scheduler | O(log n) | Eligibility + virtual deadline |
| VFS Layer | Dispatch O(1) via vtable | Inode cache hit rate |
| Page Cache | Amortized O(1) lookup | Hit ratio, refault distance |

---

## 1. Kernel Architecture — Monolithic vs Microkernel

### The Fundamental Question

An operating system kernel must provide process management, memory management,
filesystems, networking, device drivers, and security. The architectural
question is: how much of this runs in privileged mode (Ring 0)?

### Monolithic Kernel

A monolithic kernel runs **everything** in kernel space: schedulers, file
systems, network stack, device drivers — all in a single address space with
full hardware access.

**Advantages:**

- **Performance.** No inter-component communication overhead. A filesystem
  read calls the block I/O layer directly via function pointer — no message
  passing, no serialization, no context switch.
- **Simplicity of interaction.** Any kernel subsystem can call any other
  subsystem's functions directly. The VFS calls into ext4 via a function
  pointer table. ext4 calls the block layer. The block layer calls the disk
  driver. All are direct calls within the same address space.
- **Shared data structures.** The page cache, inode cache, and dentry cache
  are globally accessible. A single `struct page` is simultaneously visible to
  the memory manager, the filesystem, and the block I/O layer.

**Disadvantages:**

- **Fault isolation.** A bug in any driver or subsystem can corrupt kernel
  memory and crash the entire system. A null pointer dereference in a USB
  driver triggers a kernel panic — the network stack, the filesystem, and
  every process die with it.
- **Attack surface.** Every line of kernel code runs with full privileges. A
  vulnerability in an obscure SCSI driver grants the attacker Ring 0 access
  to the entire system.
- **Monolithic binary.** Historically, adding hardware support required
  recompiling the kernel.

### Microkernel

A microkernel runs only the **minimum** in kernel space: address space
management, thread scheduling, and inter-process communication (IPC). Everything
else — filesystems, device drivers, network protocols — runs as userspace
server processes.

**Examples:** Mach, L4, MINIX 3, QNX, seL4.

**Advantages:**

- **Fault isolation.** A crashing filesystem server can be restarted without
  affecting the rest of the system. Other processes continue running.
- **Reduced TCB.** The Trusted Computing Base (the code that must be correct
  for security) is small. seL4's kernel is ~10,000 lines and has been formally
  verified.
- **Modularity.** Components are independent processes with well-defined
  interfaces. Swapping a filesystem implementation requires no kernel changes.

**Disadvantages:**

- **IPC overhead.** Every cross-component call requires: marshal arguments →
  context switch to kernel → deliver message → context switch to server →
  process → reply → context switch back. This adds measurable latency.
- **Data copying.** Passing data between userspace servers and the kernel
  requires copying (or complex shared-memory schemes with their own overhead).

### The IPC Cost Problem

The critical performance question is the cost of IPC in a microkernel. Consider
a simple `read()` call in each architecture:

**Monolithic (Linux):**

```
Application: read(fd, buf, 4096)
  → syscall entry (trap to Ring 0)               ~100-200 ns
  → VFS lookup (function pointer dispatch)        ~50 ns
  → ext4 read (direct call)                       ~100 ns
  → page cache hit (return cached page)           ~50 ns
  → copy_to_user                                  ~200 ns
  → syscall exit                                  ~100-200 ns
Total: ~600-800 ns for a cached read
```

**Microkernel (conceptual):**

```
Application: read(fd, buf, 4096)
  → IPC to VFS server                            ~1000-2000 ns (context switch + message)
  → VFS server → IPC to FS server                ~1000-2000 ns
  → FS server → IPC to block server              ~1000-2000 ns
  → Block server: cache hit, reply                ~200 ns
  → Reply chain back (3 IPC returns)              ~3000-6000 ns
Total: ~6000-12000 ns for a cached read
```

Modern L4-family microkernels have reduced IPC to ~200-500 ns through careful
engineering (register-only messages, direct process switching, UTCB), but the
fundamental overhead remains: each component boundary adds at minimum one
context switch.

### Linux's Hybrid: The Loadable Module System

Linux chose a third path: **monolithic kernel with loadable modules**. This
preserves the performance of a monolithic architecture while gaining some
modularity benefits.

#### How Loadable Modules Work

A kernel module is a relocatable ELF object (`.ko` file) that is linked into
the running kernel at load time:

1. **Loading** (`insmod` / `modprobe`):
   - The kernel reads the `.ko` file.
   - It allocates memory in kernel space for the module's code and data.
   - It resolves symbol references against the kernel's exported symbol table
     (`EXPORT_SYMBOL`, `EXPORT_SYMBOL_GPL`).
   - It calls the module's `init` function.
   - The module is now part of the kernel — it runs in Ring 0 with full
     privileges.

2. **Symbol Resolution:**
   - Modules can call any function the kernel exports. The kernel maintains a
     symbol table of exported functions and their addresses.
   - `EXPORT_SYMBOL(func)` — available to all modules.
   - `EXPORT_SYMBOL_GPL(func)` — available only to GPL-licensed modules.
   - This is **not** dynamic linking in the userspace sense. There is no PLT/GOT.
     Symbols are resolved once at load time and patched directly into the
     module's code.

3. **Reference Counting:**
   - Each module maintains a use count. The kernel prevents unloading a module
     that is in use (e.g., a filesystem module with mounted filesystems).
   - `try_module_get()` / `module_put()` manage the reference count.

4. **Unloading** (`rmmod` / `modprobe -r`):
   - The kernel calls the module's `exit` function.
   - It verifies the reference count is zero.
   - It frees the module's memory.

#### The Module Tradeoff

Modules solve the "recompile to add hardware" problem without the IPC
overhead of microkernels. A module runs in the same address space as the
kernel — calling a module function is a direct function call, not an IPC
message.

But modules do **not** solve fault isolation. A buggy module still runs in
Ring 0 and can corrupt kernel memory. This is why Linux distributions
carefully vet which modules ship in the default kernel, and why `tainted`
kernel flags exist to track when third-party modules are loaded.

#### Module Dependency Graph

Modules can depend on other modules. `modprobe` (unlike `insmod`) resolves
dependencies automatically using `/lib/modules/$(uname -r)/modules.dep`:

```
drivers/net/ethernet/intel/e1000e/e1000e.ko: kernel/net/core/ptp_classify.ko
```

The dependency graph is a DAG (directed acyclic graph). The kernel enforces
acyclicity — circular dependencies prevent loading.

---

## 2. Memory Management — Theory and Implementation

### Physical Memory Organization

#### Zones

Linux divides physical memory into zones based on hardware constraints:

| Zone | Range (x86_64) | Purpose |
|:---|:---|:---|
| ZONE_DMA | 0 - 16 MB | Legacy ISA DMA devices |
| ZONE_DMA32 | 0 - 4 GB | 32-bit DMA-capable devices |
| ZONE_NORMAL | 4 GB - end | General purpose |
| ZONE_MOVABLE | configurable | Memory hotplug, CMA |

Each zone maintains its own buddy allocator, free page counts, watermarks, and
LRU lists.

#### NUMA Topology

On multi-socket systems, memory access is Non-Uniform:

```
CPU 0 ←→ Local RAM (Node 0) ←→ Interconnect ←→ Remote RAM (Node 1) ←→ CPU 1
         ~100 ns access                          ~200-300 ns access
```

The kernel models this as `pg_data_t` (one per NUMA node), each containing its
own set of zones. The allocator prefers local memory. The ratio of
remote-to-local access cost is the **NUMA ratio** (typically 1.5x - 3x).

### Buddy Allocator — Theory

The buddy allocator manages free physical pages. It organizes free memory into
power-of-two-sized blocks.

#### Algorithm

```
Allocate(order k):
  1. Search free_area[k] for a free block
  2. If found → remove from list, return
  3. If not found → Allocate(k+1), split result into two buddies
     - Return one buddy, place other on free_area[k]

Free(block of order k):
  1. Compute buddy address: buddy = block XOR (1 << (k + PAGE_SHIFT))
  2. If buddy is free and same order → coalesce
     - Remove buddy from free_area[k]
     - Free(merged_block of order k+1)    // recursive coalesce
  3. If buddy not free → place block on free_area[k]
```

#### Complexity Analysis

- **Allocation:** O(MAX_ORDER) worst case. Searching each order level is O(1)
  (free lists are linked lists, take head). Splitting is O(1). Maximum
  MAX_ORDER splits = O(11) = O(1) constant. Effectively O(1).
- **Freeing:** O(MAX_ORDER) worst case for cascading coalesces. Each coalesce
  step is O(1) (buddy address computed via XOR, free status checked via
  page flags). Maximum MAX_ORDER coalesces. Effectively O(1).
- **Fragmentation:** External fragmentation occurs when free memory exists but
  not in contiguous blocks of the requested order. The buddy system minimizes
  this through eager coalescing but cannot eliminate it. The kernel uses
  **compaction** (moving pages to create contiguous free blocks) as a fallback.

#### Fragmentation Analysis

The fragmentation index for a given order k:

$$\text{fragmentation}(k) = \frac{\text{free pages} - \text{free blocks of order} \geq k \times 2^k}{\text{free pages}}$$

A fragmentation index near 1.0 means memory is free but fragmented. Near 0.0
means either memory is not free or it is available in large contiguous blocks.

```bash
# View fragmentation per zone and order
cat /proc/buddyinfo
cat /sys/kernel/debug/extfrag/extfrag_index   # requires debugfs
```

### Slab Allocator (SLUB) — Theory

The buddy allocator deals in whole pages. Kernel objects are much smaller:
a `struct dentry` is ~192 bytes, a `struct inode` is ~600 bytes. Allocating
a full 4 KB page for each 192-byte dentry wastes 95% of the memory.

The slab allocator solves this by:

1. Requesting pages from the buddy allocator.
2. Dividing each page (or group of pages) into fixed-size slots matching a
   particular object type.
3. Maintaining per-CPU free lists for lock-free fast-path allocation.

#### SLUB Fast Path

```
kmalloc(192 bytes):
  1. Determine size class → kmalloc-256 (next power-of-two cache)
  2. Check per-CPU freelist (c->freelist)
  3. If non-empty → return head object, advance freelist pointer
     - No locks, no atomic operations. Single pointer assignment.
  4. If empty → slow path (refill from partial slabs or allocate new slab)
```

The fast path is **amortized O(1)** — literally a pointer load and store. No
locks on the fast path because each CPU has its own freelist (the `cpu_slab`
structure).

#### Cache Coloring

Objects within a slab are offset by varying amounts to reduce cache line
conflicts. If every `struct inode` started at offset 0 within its page, they
would all map to the same L1 cache set. Cache coloring distributes them:

```
Slab 1: offset 0, 256, 512, 768, ...
Slab 2: offset 64, 320, 576, 832, ...
Slab 3: offset 128, 384, 640, 896, ...
```

This reduces **cache thrashing** when iterating over many objects of the same
type.

### Page Cache Theory

The page cache is a write-back cache of disk contents in RAM. It is
conceptually a mapping:

```
(device, inode, offset) → struct page
```

Implemented as a **radix tree** (XArray since 5.x) per inode. Lookup is
O(height) where height = ceil(log_{64}(max_offset)) — effectively O(1) for
realistic file sizes (height <= 4 for files up to 16 TB).

#### Replacement Policy

The page cache uses a two-list LRU approximation (Active/Inactive) with
refault-distance tracking. The key invariant:

$$\frac{|L_{active}|}{|L_{active}| + |L_{inactive}|} \approx \frac{\text{WSS}}{|RAM|}$$

Where WSS is the estimated working set size. The kernel adjusts the lists
to maintain this ratio, ensuring that the active list approximates the working
set.

**MGLRU (Multi-Gen LRU)**, available since Linux 6.1, replaces the two-list
model with multiple generations. Pages age through generations; the oldest
generation is evicted first. This reduces the scan overhead of traditional LRU
by avoiding full list rotations.

---

## 3. Scheduler — Complexity Analysis

### CFS (Completely Fair Scheduler)

CFS models an ideal multi-tasking CPU where each of N runnable tasks receives
exactly 1/N of CPU time. The virtual runtime tracks how far each task deviates
from this ideal:

$$vruntime_i = \int_0^t \frac{1}{w_i / \sum_j w_j} \, d\tau = \int_0^t \frac{W}{w_i} \, d\tau$$

Where $w_i$ is the weight of task i (derived from nice value) and $W = \sum_j w_j$
is the total weight of all runnable tasks.

A task with twice the weight accumulates vruntime at half the rate, so it
gets scheduled twice as often.

#### Data Structure: Red-Black Tree

CFS maintains runnable tasks in a **red-black tree** keyed by vruntime:

- **pick_next_task()**: Return the leftmost node (minimum vruntime). This is
  cached, so the operation is O(1).
- **Enqueue (wake up / new task)**: Insert into the rb-tree. O(log n).
- **Dequeue (sleep / exit)**: Remove from the rb-tree. O(log n).
- **Update vruntime**: Update current task's vruntime on every timer tick and
  on voluntary yields. O(1) — just an arithmetic update; re-insertion into
  the tree only happens on preemption.

#### Timeslice Calculation

CFS does not assign fixed timeslices. Instead, it computes a **target latency**
(the period within which every runnable task should run at least once):

$$\text{timeslice}_i = \text{latency} \times \frac{w_i}{W}$$

With the constraint:

$$\text{timeslice}_i \geq \text{min\_granularity}$$

If there are too many tasks such that every timeslice would be below
`min_granularity`, the actual period stretches beyond the target latency.

**Defaults** (tunable via sysctl):

| Parameter | Default | Meaning |
|:---|:---|:---|
| `sched_latency_ns` | 6 ms (desktop) / 24 ms (server) | Target latency period |
| `sched_min_granularity_ns` | 0.75 ms (desktop) / 3 ms (server) | Minimum timeslice |
| `sched_wakeup_granularity_ns` | 1 ms (desktop) / 4 ms (server) | Preemption threshold on wakeup |

#### Fairness Bound

For N tasks with equal weight, the maximum deviation from perfect fairness:

$$|t_i - \frac{T}{N}| \leq \text{sched\_latency\_ns}$$

Where $t_i$ is actual CPU time for task i over total observation time T. This
bound holds because CFS re-examines the tree at most every `sched_latency_ns`
nanoseconds.

### EEVDF (Earliest Eligible Virtual Deadline First)

Linux 6.6+ replaced CFS with **EEVDF**, which adds a virtual deadline concept:

$$\text{deadline}_i = \text{eligible\_time}_i + \frac{\text{request\_length}}{w_i / W}$$

The scheduler picks the task with the **earliest virtual deadline** among those
that are **eligible** (have accumulated enough lag). This provides better
latency guarantees for interactive tasks while maintaining fairness.

EEVDF uses the same rb-tree structure as CFS (O(log n) insertion/removal) but
the key is the virtual deadline rather than raw vruntime.

### Scheduling Classes

Linux supports multiple scheduling classes, checked in priority order:

```
stop_sched_class          → migration/watchdog (cannot be preempted)
  ↓
dl_sched_class            → SCHED_DEADLINE (EDF, hard real-time)
  ↓
rt_sched_class            → SCHED_FIFO, SCHED_RR (soft real-time)
  ↓
fair_sched_class          → SCHED_NORMAL, SCHED_BATCH (CFS/EEVDF)
  ↓
idle_sched_class          → idle task (runs when nothing else can)
```

`pick_next_task()` iterates through classes in order. If a DEADLINE task is
runnable, it always preempts CFS tasks. This is O(1) in the number of classes
(constant, ~5 classes).

### Load Balancing

On SMP systems, the scheduler must distribute tasks across CPUs. Linux uses a
hierarchical **scheduling domain** system:

```
NUMA node domain (balance every ~1 second)
  └── Socket domain (balance every ~16 ms)
      └── Core domain (balance every ~4 ms)
          └── SMT domain (balance every ~2 ms)
```

At each level, the load balancer computes imbalance and migrates tasks from
the busiest group to the idlest group. Migration has a cost — the task's cache
working set becomes cold on the new CPU. The parameter
`sched_migration_cost_ns` (default 500 us) is the assumed cache-warmup cost;
the balancer will not migrate if the imbalance is small relative to this cost.

---

## 4. VFS Abstraction Layer — Design Analysis

### The Problem VFS Solves

Linux supports dozens of filesystems: ext4, XFS, btrfs, NFS, FUSE, proc, sys,
tmpfs, overlayfs, CIFS, and many more. Applications should not need to know
which filesystem they are talking to. `open()`, `read()`, `write()`, `stat()`
must work the same way regardless of the underlying storage.

### The Solution: Object-Oriented C via Function Pointer Tables

VFS defines four core object types, each with an operations structure (vtable):

#### Superblock

```c
struct super_block {
    struct list_head        s_list;        // list of all superblocks
    dev_t                   s_dev;         // device identifier
    unsigned long           s_blocksize;   // block size in bytes
    struct file_system_type *s_type;       // filesystem type
    struct super_operations *s_op;         // vtable
    struct dentry           *s_root;       // root dentry
    // ...
};

struct super_operations {
    struct inode *(*alloc_inode)(struct super_block *sb);
    void (*destroy_inode)(struct inode *inode);
    void (*dirty_inode)(struct inode *inode, int flags);
    int (*write_inode)(struct inode *inode, struct writeback_control *wbc);
    int (*statfs)(struct dentry *dentry, struct kstatfs *buf);
    int (*sync_fs)(struct super_block *sb, int wait);
    // ...
};
```

One `super_block` exists per mounted filesystem instance. When you mount ext4
on `/data`, the kernel creates a `super_block` with `s_op` pointing to ext4's
implementation of `alloc_inode`, `write_inode`, etc.

#### Inode

```c
struct inode {
    umode_t                 i_mode;        // file type + permissions
    uid_t                   i_uid;         // owner
    gid_t                   i_gid;         // group
    loff_t                  i_size;        // file size
    struct timespec64       i_atime;       // access time
    struct timespec64       i_mtime;       // modification time
    struct inode_operations *i_op;         // vtable for inode ops
    struct file_operations  *i_fop;        // vtable for file ops
    struct super_block      *i_sb;         // parent superblock
    struct address_space    *i_mapping;    // page cache for this inode
    // ...
};

struct inode_operations {
    int (*create)(struct inode *, struct dentry *, umode_t, bool);
    struct dentry *(*lookup)(struct inode *, struct dentry *, unsigned int);
    int (*link)(struct dentry *, struct inode *, struct dentry *);
    int (*unlink)(struct inode *, struct dentry *);
    int (*mkdir)(struct inode *, struct dentry *, umode_t);
    int (*rename)(struct inode *, struct dentry *,
                  struct inode *, struct dentry *, unsigned int);
    // ...
};
```

The `i_op` vtable handles metadata operations (create file, delete, rename).
The `i_fop` vtable handles data operations (read, write, mmap).

#### Dentry (Directory Entry)

```c
struct dentry {
    struct dentry           *d_parent;     // parent dentry
    struct qstr             d_name;        // component name
    struct inode            *d_inode;      // associated inode
    struct super_block      *d_sb;         // superblock
    struct dentry_operations *d_op;        // vtable
    // ...
};
```

Dentries form a tree structure mirroring the directory hierarchy. The **dentry
cache (dcache)** is a hash table that caches recent path lookups:

```
Path lookup for "/home/user/file.txt":
  → hash "/" → dcache hit → dentry for /
  → hash "home" → dcache hit → dentry for /home
  → hash "user" → dcache hit → dentry for /home/user
  → hash "file.txt" → dcache miss → call i_op->lookup() → create dentry
```

Dcache lookup is O(1) average (hash table). On miss, the filesystem's
`lookup()` function is called, which may require disk I/O.

#### File

```c
struct file {
    struct path             f_path;        // dentry + mount
    struct inode            *f_inode;      // associated inode
    const struct file_operations *f_op;    // vtable
    loff_t                  f_pos;         // current offset
    unsigned int            f_flags;       // open flags (O_RDONLY, etc.)
    // ...
};

struct file_operations {
    loff_t (*llseek)(struct file *, loff_t, int);
    ssize_t (*read)(struct file *, char __user *, size_t, loff_t *);
    ssize_t (*write)(struct file *, const char __user *, size_t, loff_t *);
    int (*mmap)(struct file *, struct vm_area_struct *);
    int (*open)(struct inode *, struct file *);
    int (*release)(struct inode *, struct file *);
    int (*fsync)(struct file *, loff_t, loff_t, int datasync);
    // ...
};
```

A `struct file` is created on every `open()` call. Multiple file descriptors
can point to the same inode (via hard links or `dup()`).

### The Dispatch Chain

A complete `read()` call through VFS:

```
sys_read(fd, buf, count)
  → fd_to_file(fd)                         // fd table lookup, O(1)
  → file->f_op->read(file, buf, count)     // vtable dispatch, O(1)
    → [ext4_file_read_iter]                // filesystem implementation
      → generic_file_read_iter()           // common page cache path
        → page_cache_lookup(mapping, index) // XArray lookup, O(1) amortized
          → hit: copy_to_user(buf, page)   // memcpy
          → miss: readpage() → submit_bio() → wait for I/O
```

The VFS dispatch itself is O(1) — a function pointer dereference. The
performance of the overall operation depends on the filesystem implementation
and whether data is cached.

### Design Patterns in VFS

1. **Strategy Pattern.** Each `*_operations` struct is a strategy. The
   filesystem provides its implementation; VFS dispatches polymorphically.

2. **Object Lifetime Management.** Inodes, dentries, and superblocks are
   reference-counted (`i_count`, `d_lockref`, `s_active`). Objects are freed
   when their reference count drops to zero and they are not cached.

3. **Cache Hierarchy.** VFS maintains three caches (dcache, inode cache, page
   cache) that reduce disk I/O. Cache pressure is managed by the shrinker
   subsystem, which reclaims cache entries when memory is low. The tunable
   `vm.vfs_cache_pressure` controls the aggressiveness (default 100; lower
   values favor keeping VFS caches).

4. **Layered Composition.** Stacking filesystems (overlayfs, ecryptfs) work
   by implementing the VFS operations and delegating to a lower filesystem.
   This is the foundation of container image layers (Docker's overlay2 driver).

### VFS Performance Characteristics

| Operation | dcache hit | dcache miss (cached inode) | Cold (disk I/O) |
|:---|:---|:---|:---|
| `stat()` | ~200 ns | ~1-5 us | ~1-10 ms |
| `open()` | ~500 ns | ~2-10 us | ~1-10 ms |
| `read()` 4KB page cache hit | ~500 ns | N/A | ~1-10 ms |
| `readdir()` 100 entries cached | ~10 us | ~50-200 us | ~5-50 ms |

The dcache and inode cache are the most critical performance structures in the
Linux filesystem layer. On a warm system, the vast majority of path lookups
never touch disk.

---

## 5. Connections to Unheaded

The Linux kernel internals directly underpin Unheaded's architecture:

- **eBPF** attaches to kernel hooks (XDP, tc, kprobes, tracepoints) — the
  verifier and JIT are kernel subsystems. Understanding the kernel's
  networking stack and event model is essential for writing correct BPF
  programs.
- **Namespaces and cgroups** are the kernel primitives behind LXD containers
  and Docker. Unheaded's container hardening (seccomp, capabilities,
  read-only FS) maps directly to kernel security features.
- **The VFS layer** explains how Unheaded's services interact with storage —
  each container sees an isolated mount namespace, and overlayfs provides
  image layering.
- **The scheduler** determines how Unheaded's services share CPU time. CFS
  weights and cgroup CPU controllers directly control service resource
  allocation.
- **The page cache** is critical for Unheaded's performance — Wotan's
  message bus and timeguru's SQLite backend both depend heavily on cached I/O.

---

## See Also

- memory-tuning
- cpu-scheduler-tuning
- io-scheduler-tuning
- ebpf
- cgroups
- namespaces
- containers

## References

- Bovet, D. & Cesati, M. "Understanding the Linux Kernel, 3rd Edition" (O'Reilly, 2005)
- Love, R. "Linux Kernel Development, 3rd Edition" (Addison-Wesley, 2010)
- Gorman, M. "Understanding the Linux Virtual Memory Manager" (Prentice Hall, 2004)
- Linux kernel source: https://elixir.bootlin.com/
- LWN.net: "An EEVDF CPU scheduler for Linux" https://lwn.net/Articles/925371/
- LWN.net: "Multi-generational LRU" https://lwn.net/Articles/856931/
- Linux kernel documentation — VFS: https://docs.kernel.org/filesystems/vfs.html
- Linux kernel documentation — Memory Management: https://docs.kernel.org/mm/
- Linux kernel documentation — Scheduler: https://docs.kernel.org/scheduler/
- Brendan Gregg — Linux Performance: https://www.brendangregg.com/linuxperf.html
