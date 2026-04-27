# Linux Kernel — College (Part 4 of 4 — Ramp-Up Curriculum)

> Subsystem-level deep dive: buddy allocator, SLUB, page cache + reclaim, blk-mq + io_uring, NAPI + XDP, RCU, CFS/EEVDF, NUMA balancing, custom kernel builds — the level you need before reading kernel/Documentation/ for real.

## Prerequisites

- `cs ramp-up linux-kernel-high-school` — prior tier (kernel/userspace split, syscalls, /proc, /sys, namespaces, cgroups v1/v2, eBPF basics)
- Working knowledge of C: pointers, structs, function pointers, arrays-vs-pointers, the `container_of` trick, linker visibility (`static`, `extern`)
- Familiarity with /proc, /sys, syscalls, namespaces, cgroups, eBPF (covered in HS tier)
- A handful of common syscalls remembered by name: `read`, `write`, `mmap`, `clone`, `epoll_wait`, `io_uring_enter`
- Comfort reading short kernel patches on lkml or LWN
- Tools available: `numactl`, `perf`, `bpftrace`, `ethtool`, `ss`, `chrt`, `taskset`, `make`, `git`, `ftrace` via `trace-cmd` (optional)

## Plain English

The High School tier explained the contract between kernel and userspace and walked through the user-facing surface area: syscalls, namespaces, cgroups, eBPF as a sandboxed extension language. This College tier opens the engine cover. You will see how the kernel actually allocates physical pages (the buddy system), how it sub-allocates structures inside those pages (SLUB), why the system never reports "all memory free" on a healthy box (page cache), how a packet really gets from the wire to your `recv()` call (NAPI + softirq + skb), how the block I/O path multiplexes across CPUs (blk-mq + io_uring), how readers in shared data structures pay zero overhead (RCU), how the scheduler picks the next thread (CFS — and now EEVDF since 6.6), how memory is laid out across sockets (NUMA), and how to roll your own kernel from a configured tree.

The scope is roughly:

- **Memory management:** buddy → zones → SLUB → page cache → reclaim/kswapd → vmalloc/kmalloc → THP
- **I/O:** bio → request_queue → blk-mq → I/O schedulers → direct/buffered → io_uring (since 5.1)
- **Network:** NIC ring → softirq → NAPI → skb → IP/TCP/UDP → socket → XDP (driver-level BPF since 4.8)
- **Synchronization:** spinlock → rwlock → seqlock → mutex → semaphore → atomic_t → memory barriers → RCU → per-CPU
- **Scheduler:** CFS rb-tree by vruntime → real-time classes → load balancer → sched domains → EAS → EEVDF (since 6.6)
- **NUMA:** nodes, distance matrix, AutoNUMA fault scanning, mempolicy, numactl
- **Build:** Kconfig → defconfig → menuconfig → kbuild → bzImage → modules → grub → bisect

You do not need to memorize the whole thing. You need to know what each subsystem owns, which `/proc` or `/sys` file or tool exposes its state, the exact tunable that fixes the most common production complaint, and which kernel-tree directory the source lives in so you can grep when something is wrong. After this tier, the daily-driver reference is `cs fundamentals linux-kernel-internals` and the applied tuning lives in the `kernel-tuning/*` family.

The North Star: you should never need a browser to tune `vm.dirty_ratio`, set up a NUMA-bound workload, decide between `mq-deadline` and `kyber`, or write a `bpftrace` one-liner that counts `vfs_read` per process. Everything is on this page or one `cs` jump away.

## Concepts in Detail

This is the long part. Each H3 below is its own little article. Skim on first pass; come back when you need the detail.

### Kernel Memory Management

The kernel manages four overlapping kinds of memory:

1. **Physical pages** (`struct page`, one per page frame) — the buddy allocator owns these.
2. **Slab objects** (kernel structs of various sizes) — SLUB carves them out of pages.
3. **Page cache** (pages backing files and anonymous regions) — owned by `struct address_space`.
4. **Virtual address spaces** — `init_mm` (kernel) and one `struct mm_struct` per userspace process.

The boundaries between them are fluid: a userspace `read()` becomes a slab `bio` that uses page-cache pages allocated from the buddy. Understanding which layer a tunable affects is half the battle in production.

#### Physical Page Allocator (Buddy System)

The kernel keeps physical memory in **page frames**, typically 4 KiB each on x86_64. The buddy allocator manages free pages in **orders** 0 through 10 (default `MAX_ORDER` = 11), where order *n* means a contiguous block of 2^n pages. Order 0 is one page (4 KiB), order 10 is 1024 pages (4 MiB).

```
Buddy allocator free lists per zone (per-NUMA-node)

order:  0       1       2       3       4       5       6       7       8       9      10
size:   4K      8K      16K     32K     64K    128K    256K    512K     1M      2M      4M
        |       |       |       |       |       |       |       |       |       |       |
        v       v       v       v       v       v       v       v       v       v       v
      [ ][ ]  [  ]    [    ]  [      ][        ]                         [                    ]

Allocation flow when order-3 (32K) is requested but order-3 is empty:

  1. Walk up to the next non-empty order (say order-5).
  2. Split: a 128K block becomes two 64K buddies. Take one, push other to order-4 list.
  3. Split again: 64K becomes two 32K buddies. Return one, push other to order-3 list.
  4. Caller gets a 32K (8 page) contiguous block.

Free flow:
  1. Look at the buddy of the freed block (XOR address with block size).
  2. If buddy is also free and same order, merge (coalesce) into next-higher order.
  3. Repeat until buddy is allocated or order 10 reached.
```

Each `struct page` (one per page frame) lives in the global `mem_map` array (or in the `vmemmap` for SPARSEMEM_VMEMMAP, the default since the late 2010s). The page struct is roughly 64 bytes — multiplied by the page count it amounts to a few percent of RAM. It carries the page's flags (`PG_locked`, `PG_dirty`, `PG_writeback`, `PG_uptodate`, `PG_lru`, `PG_active`, ...), a refcount, a mapcount (how many PTEs reference it), an `lru` list head, and a union of fields whose meaning depends on what the page is being used for (slab metadata, page cache index, anon_vma chain, ...).

The buddy allocator's hot path, `__alloc_pages_nodemask()` (renamed several times across versions; in modern trees it is `__alloc_pages()`), takes a `gfp_mask`, an `order`, a preferred zone, and a node mask. It walks the zonelist for the preferred node, tries the per-CPU page lists first (the `pcp` lists for order 0), then the free-area arrays for the appropriate order, splitting if necessary. On failure it wakes `kswapd`, may enter direct reclaim, may invoke compaction, and ultimately may invoke the OOM killer.

Inspect with `/proc/buddyinfo`:

```text
Node 0, zone      DMA      0      0      1      1      1      0      1      0      1      1      3
Node 0, zone    DMA32     12     11     14      8      7     10      5      4      3      2    811
Node 0, zone   Normal   3145   1024    512    256    128     64     32     16      8      4    192
```

Read left to right: order 0 → order 10. The numbers are **count of free blocks at that order**. A box that is fragmenting will show many low-order blocks and few high-order blocks; that's how `kcompactd` (the compaction kernel thread) decides when to run.

Key code lives in `mm/page_alloc.c`. The hot path is `__alloc_pages()`. Compaction lives in `mm/compaction.c` and migrates movable pages to free up high-order blocks. The unmovable/reclaimable/movable migration types segregate the freelists so unmovable allocations (slab metadata) don't permanently fragment a region that movable pages could otherwise live in.

Pages are also tracked in **migratetype** lists within each zone:

- `MIGRATE_UNMOVABLE` — slab, kernel-internal stuff, can't be moved.
- `MIGRATE_MOVABLE` — anonymous pages, file cache pages mapped through normal PTEs.
- `MIGRATE_RECLAIMABLE` — slab caches that have a shrinker.
- `MIGRATE_HIGHATOMIC` — reserved for atomic high-order allocations.
- `MIGRATE_CMA` — Contiguous Memory Allocator (mainly ARM).
- `MIGRATE_ISOLATE` — temporary state during compaction or memory hotplug.

When the right migrate-list is empty, the allocator **steals** from another list, but tries to steal a whole pageblock to keep fragmentation low.

#### Zone-Based Allocation

Within a NUMA node, physical memory is split into **zones** for legacy device constraints:

```
+--------------------+--------+----------------------------------------+
| Zone               | Range  | Purpose                                |
+--------------------+--------+----------------------------------------+
| ZONE_DMA           | 0-16M  | ISA DMA, ancient devices               |
| ZONE_DMA32         | 16M-4G | 32-bit DMA-capable PCI devices         |
| ZONE_NORMAL        | 4G+    | The bulk of RAM on a 64-bit box        |
| ZONE_MOVABLE       | (any)  | Pages that can be migrated (hotplug)   |
| ZONE_DEVICE        | (any)  | Driver-managed memory (CXL, persistent)|
+--------------------+--------+----------------------------------------+

Fallback order on x86_64 when ZONE_NORMAL has nothing:
   ZONE_NORMAL  ->  ZONE_DMA32  ->  ZONE_DMA
```

Each zone has its own free lists, watermarks (`min`, `low`, `high`), and reclaim state. See `/proc/zoneinfo`. Watermarks scale automatically with `vm.min_free_kbytes` (default a small fraction of total RAM, but you should bump it on multi-NUMA boxes — Red Hat recommends 1 GiB per terabyte of RAM as a starting point).

The fields you care about in `/proc/zoneinfo`:

```
pages free       # current free pages in this zone
pages min        # the min watermark
pages low        # the low watermark; kswapd wakes when free < low
pages high       # the high watermark; kswapd stops when free > high
nr_free_pages    # same as pages free
nr_zone_active_anon, nr_zone_inactive_anon
nr_zone_active_file, nr_zone_inactive_file
nr_zone_unevictable
nr_zone_write_pending
nr_mlock
nr_bounce
present pages    # actual physical pages in this zone
managed pages    # present minus reserved (e.g. firmware)
protection: (...) # the lowmem_reserve_ratio enforcement; reserves
                  # lower zones from being drained by higher-zone allocations
```

The watermark cascade:

```
free pages
   ^
   |   high  -- kswapd stops reclaiming
   |
   |   low   -- kswapd wakes up (background reclaim)
   |
   |   min   -- direct reclaim; allocators in PF_MEMALLOC may dip below
   |   0
```

#### SLUB Allocator (default since 2.6.23)

The buddy allocator gives you whole pages. Most kernel objects are smaller (a `task_struct` is around 7 KiB, a `dentry` is 192 bytes). SLUB carves a page (or several) into a **slab** of identically sized objects, called a **cache**. SLUB is the default since 2.6.23 (Oct 2007); SLAB existed before, SLOB is for tiny embedded boxes.

```
SLUB cache layout per CPU per node:

  kmem_cache "task_struct" (size=7232, align=64, objects-per-slab=4)
       |
       +--- per-CPU active slab (cpu_slab) -----+
       |    [obj][obj][FREE][obj]                |
       |                                          |
       +--- per-CPU partial list (~30 slabs)     |
       |    [partial1][partial2]...              |
       |                                          |
       +--- per-node partial list                |
       |    [node-partial-slab1][...]            |
       |                                          |
       +--- full list (debug only)               |
            [full1]...
```

Why per-CPU? Allocation/free without taking a global lock is the whole point. Free path on the same CPU pushes onto the per-CPU partial list with one `cmpxchg`.

```
Slab object lifecycle:

   kmalloc(size, GFP_KERNEL)
      |
      v
   kmalloc_caches[type][index]    <- pick the right kmem_cache by size class
      |
      v
   slab_alloc()
      |
      v
   per-CPU active slab has free obj?  yes -> pop -> return
      |
      no
      |
      v
   per-CPU partial list non-empty?    yes -> promote slab to active -> retry
      |
      no
      |
      v
   per-node partial list non-empty?   yes -> grab partial -> retry
      |
      no
      |
      v
   alloc_pages(GFP_KERNEL, order)     <- ask the buddy for a fresh slab
      |
      v
   format the page into a slab; place on per-CPU active; return obj
```

The size-class buckets for `kmalloc` are powers of two and a few in-betweens: 8, 16, 32, 64, 96, 128, 192, 256, 512, 1024, 2048, 4096, 8192, ... up to 8 MiB. Above 8 MiB, `kmalloc()` falls back to `kvmalloc()`/`vmalloc()` style allocation. There are also `kmalloc-rcl-*` (for reclaimable allocations), `kmalloc-dma-*`, and `kmalloc-cg-*` (for memcg accounting).

Named caches are created with `kmem_cache_create()`; ones declared by drivers show up in `/proc/slabinfo` with their driver-given names. Look for ones that grow without bound under load — that's a slab leak.

Inspect with `/proc/slabinfo` (root only):

```text
slabinfo - version: 2.1
# name            <active_objs> <num_objs> <objsize> <objperslab> <pagesperslab> : tunables ...
task_struct           1894       1932       7232           4             8 : tunables ...
dentry               48312      48720        192          21             1 : tunables ...
inode_cache          12345      14000        688          11             2 : tunables ...
kmalloc-1024          2048       2048       1024           4             1 : tunables ...
```

Or use the `slabtop(1)` ncurses viewer which sorts live.

The four most common kernel structs you'll see consume the bulk of slab memory:

- `dentry` — directory entries cached in the dcache. One per name lookup. Negative dentries (cached "name does not exist") are also tracked here. Tunable: `vm.vfs_cache_pressure` (default 100; higher reclaims dcache more aggressively).
- `inode_cache`, `ext4_inode_cache`, `xfs_inode` — VFS and FS-specific inode caches. Reclaimed alongside dcache.
- `task_struct` — one per thread/process. About 7-12 KiB depending on configuration. The `task_struct` itself plus the kernel stack (8 KiB or 16 KiB on x86_64) accounts for a meaningful chunk on a fork-heavy server.
- `nf_conntrack` — netfilter connection tracking entries. Each TCP connection is one entry. Default limit `nf_conntrack_max` of about 65k on modest hardware; production load balancers need millions.

Slab debugging is on by adding `slub_debug=FZP` to the kernel command line: F = sanity checks, Z = redzone, P = poison. Or `CONFIG_KFENCE=y` for sample-based use-after-free detection at low overhead in production.

#### Page Cache and Reclaim

Linux caches every read from disk in the **page cache** (file-backed pages) and tracks anonymous (heap, stack, mmap MAP_ANONYMOUS) pages on a separate set of lists. Both share the same reclaim machinery.

```
Two-list LRU per memory cgroup per node:

   active_anon   <----- promote ------   inactive_anon
        |                                       ^
        |                                       |
        v                                       |
   [evict if anon swap allowed]            [add new page here]


   active_file   <----- promote ------   inactive_file
        |                                       ^
        |                                       |
        v                                       |
   [drop pages, no I/O if clean]           [add new page here]


   shrink_node()
        | walks lists, tests reference bit, shuffles pages
        v
   reclaim writes dirty pages, evicts clean ones
```

`kswapd` (one per NUMA node) wakes when free < low watermark. **Direct reclaim** happens in the allocating task's context when `kswapd` cannot keep up — that is the source of "stalls in alloc_pages." `kswapd` runs `shrink_node()` which alternates anon-reclaim and file-reclaim using a ratio derived from the recently-rotated lists; the goal is to balance pressure rather than evict the wrong kind of page.

Shrinkers (`struct shrinker`) are the contract for releasing memory from non-page caches: a callback that takes a target count of objects and returns how many it freed. Filesystems register shrinkers for inode/dentry; subsystems like the GEM/TTM graphics buffer manager have their own. `/sys/kernel/debug/shrinker` lists them.

Two-list LRU details: when a page is referenced for the **second** time on the inactive list, it is promoted to active. A single reference doesn't promote — that prevents a single sequential scan over a huge file from polluting active. The page table accessed bit is sampled and cleared during scanning. The Multi-Generational LRU (MGLRU, since 6.1) replaces the two-list with multiple generations and improves on this; tunables in `/sys/kernel/mm/lru_gen/`.

Dirty pages and writeback:

```
write(fd, buf) -- userspace
     |
     v
  copy into page cache, mark page Dirty, account in /proc/meminfo Dirty
     |
     v
  background writeback (bdi-flush threads) wakes when:
     - dirty bytes / RAM > vm.dirty_background_ratio (default 10)
     - dirty page is older than vm.dirty_expire_centisecs (default 3000 = 30s)
     |
     v
  When dirty bytes / RAM > vm.dirty_ratio (default 20), a writing task
  is throttled (it waits in balance_dirty_pages() until it gets credit).
     |
     v
  fsync(fd) or sync(2) blocks until all dirty pages in scope are on disk
```

Tunables (sysctl):

```
vm.dirty_ratio                  20    # task throttling threshold (% RAM)
vm.dirty_background_ratio       10    # bdi flush wake threshold (% RAM)
vm.dirty_bytes                  0     # absolute alternative to dirty_ratio
vm.dirty_background_bytes       0     # absolute alternative
vm.dirty_expire_centisecs       3000  # 30s; pages older than this get flushed
vm.dirty_writeback_centisecs    500   # 5s; how often bdi-flush wakes
vm.swappiness                   60    # 0-200; higher = prefer swap anon over file evict
vm.vfs_cache_pressure           100   # higher = reclaim dentry/inode harder
vm.overcommit_memory            0     # 0 heuristic, 1 always, 2 strict
vm.overcommit_ratio             50    # only used if overcommit_memory=2
vm.min_free_kbytes              ...   # the watermark scale base
vm.zone_reclaim_mode            0     # NUMA-local reclaim mode
vm.numa_stat                    1     # numastat counters
vm.compaction_proactiveness     20    # since 5.7; background compaction effort
vm.watermark_boost_factor       15000 # since 5.0; temporary boost to combat fragmentation
vm.watermark_scale_factor       10    # since 4.6; controls min/low/high spread
vm.page-cluster                 3     # readahead pages-power for swap (1<<3 = 8)
```

Setting `dirty_bytes` (absolute) overrides `dirty_ratio` (relative). For boxes with > 64 GiB RAM, prefer absolute bytes — 20% of 256 GiB is 51 GiB of dirty pages, and a sudden flush to a 1 GB/s SSD will stall everything for 51 seconds. Set `dirty_bytes=4G dirty_background_bytes=1G` instead.

#### vmalloc vs kmalloc vs alloc_pages

```
+---------------+--------------+--------------+------------------------+
| API           | Backed by    | Contiguous?  | When to use            |
+---------------+--------------+--------------+------------------------+
| alloc_pages() | buddy        | physical     | DMA, hugepages, raw    |
| __get_free_pages | buddy     | phys + virt  | quick page-grain alloc |
| kmalloc()     | SLUB cache   | phys + virt  | <= ~4MB structs        |
| kmem_cache_*  | named SLUB   | phys + virt  | many same-size objects |
| vmalloc()     | non-contig   | virt only    | large bufs, no DMA     |
| vzalloc()     | non-contig   | virt only    | vmalloc + zero         |
| __vmalloc(GFP)| non-contig   | virt only    | atomic ctx forbidden   |
+---------------+--------------+--------------+------------------------+
```

GFP flags govern allocation context:

- `GFP_KERNEL` — the default; may sleep, may invoke reclaim, may swap.
- `GFP_ATOMIC` — interrupt context; may not sleep; no reclaim allowed.
- `GFP_NOIO` — no recursive I/O (use inside a filesystem to avoid deadlock).
- `GFP_NOFS` — no recursive filesystem (use inside a block driver).
- `GFP_USER` — for userspace-visible pages.
- `GFP_DMA` / `GFP_DMA32` — restrict to DMA-capable zone.
- `__GFP_ZERO` — zero the memory.
- `__GFP_NORETRY` / `__GFP_RETRY_MAYFAIL` — bound the reclaim effort.

#### Transparent Hugepages (THP)

Two MiB hugepages reduce TLB pressure. Three modes via `/sys/kernel/mm/transparent_hugepage/enabled`: `always`, `madvise`, `never`. `madvise` is the safe default for most workloads — apps explicitly call `madvise(MADV_HUGEPAGE)` on regions where they want them. `khugepaged` is the daemon that promotes regions in the background; tunables in `/sys/kernel/mm/transparent_hugepage/khugepaged/`.

THP gotchas:

- **Latency spike**: `always` mode can stall an `mmap()` call while the kernel synchronously compacts memory to find a 2 MiB block. Many database workloads (Redis, MongoDB, Postgres) recommend `madvise` or `never`.
- **Memory amplification**: a process that touches one byte of a 2 MiB region gets the full 2 MiB charged to it. A heap of many small allocations may consume far more RSS than expected.
- **defrag setting**: `/sys/kernel/mm/transparent_hugepage/defrag` controls how aggressively the kernel will reclaim/compact to satisfy a THP allocation. `defer` and `defer+madvise` are good middle grounds — never block the caller, but kick `khugepaged` to fix it later.

Static (explicit) hugepages are reserved at boot via `default_hugepagesz=2M hugepagesz=2M hugepages=512` on the kernel command line, and used by apps via `MAP_HUGETLB | MAP_HUGE_2MB` or via `hugetlbfs` mount. 1 GiB hugepages are also possible (`hugepagesz=1G`) on x86_64 with `pdpe1gb` CPU feature.

### I/O Subsystem

#### Block Layer

Every block I/O starts as a `struct bio` (block I/O), which describes a list of pages plus a target device, sector, and direction. The block layer hands `bio`s to a `request_queue`, which used to be a single per-device queue but is now multi-queue (`blk-mq`, default since 3.13).

```
struct bio {
    struct bio       *bi_next;       /* request queue link */
    struct block_device *bi_bdev;
    blk_opf_t         bi_opf;        /* REQ_OP_READ / WRITE / FLUSH / ... */
    struct bvec_iter  bi_iter;       /* sector + size cursor */
    struct bio_vec   *bi_io_vec;     /* (page, offset, len) tuples */
    bio_end_io_t     *bi_end_io;     /* completion callback */
    ...
};
```

```
blk-mq topology:

  CPU 0    CPU 1    CPU 2    CPU 3
    |        |        |        |
    v        v        v        v
  [sw-q0]  [sw-q1]  [sw-q2]  [sw-q3]   (software queues, per-CPU)
       \      |     /        /
        \     |    /        /
         \    |   /        /
          v   v  v        v
        +-----------+   +-----------+
        | hw-q 0    |   | hw-q 1    |  (hardware queues, per NIC/NVMe queue)
        +-----------+   +-----------+
              |               |
              v               v
              NVMe / SCSI / virtio-blk dispatch
```

#### I/O Schedulers

```
+----------------+----------------------------------------------------------+
| none           | No scheduling, FIFO submission. Best on NVMe with many   |
|                | hardware queues; default for blk-mq on fast devices.     |
+----------------+----------------------------------------------------------+
| mq-deadline    | Read-priority deadline scheduler. Good for SATA SSD,     |
|                | reasonable on rotational disks. Default for many distros.|
+----------------+----------------------------------------------------------+
| kyber          | Latency-target based, two queues (sync/async). Tunable   |
|                | via /sys/block/<dev>/queue/iosched/{read,write}_lat_nsec.|
+----------------+----------------------------------------------------------+
| bfq            | Budget Fair Queueing — fair across cgroups. Best on      |
|                | desktops with mixed interactive/background workloads.    |
+----------------+----------------------------------------------------------+
```

Pick with `echo kyber > /sys/block/nvme0n1/queue/scheduler`.

When to pick which (rules of thumb):

- **NVMe with deep hardware queues** — `none`. The hardware does its own scheduling; software scheduling adds latency and inversion risk.
- **SATA SSD** — `mq-deadline` for general use; `kyber` if you have a clear tail-latency target.
- **Rotational disks (rare today)** — `bfq` or `mq-deadline`. Definitely not `none`.
- **Mixed interactive desktop** — `bfq` (fairness across cgroups, good interactive feel).

Each scheduler has tunables under `/sys/block/<dev>/queue/iosched/`. For `kyber`: `read_lat_nsec` and `write_lat_nsec` (target latencies). For `mq-deadline`: `read_expire`, `write_expire`, `front_merges`, `fifo_batch`. For `bfq`: `low_latency`, `slice_idle`, `strict_guarantees`.

The block layer also exposes per-queue knobs in `/sys/block/<dev>/queue/`:

```
nr_requests          # max in-flight requests per hw-queue
read_ahead_kb        # readahead window, default 128 KiB
rq_affinity          # 0/1/2; whether completion happens on issuing CPU
nomerges             # 0=auto-merge, 1=no merges across requests, 2=no merges at all
add_random           # whether I/O contributes to the entropy pool
rotational           # 0=SSD, 1=HDD; affects scheduler heuristics
write_cache          # "write back" or "write through"
discard_granularity  # block-discard alignment for SSD TRIM
zoned                # for zoned block devices (ZNS NVMe, SMR HDDs)
```

#### Direct I/O vs Buffered I/O vs Zero-Copy

```
+---------------------+--------------------------------------------------+
| Path                | Behavior                                         |
+---------------------+--------------------------------------------------+
| read()/write()      | Buffered. Goes through page cache.               |
| open(O_DIRECT)      | Bypass page cache; user buffer must be aligned.  |
| mmap() + memcpy     | Page cache mapping into user VAS.                |
| sendfile(out, in)   | Kernel-only copy file -> socket; one syscall.    |
| splice()            | Generic kernel pipe between two fds.             |
| copy_file_range()   | Kernel-side copy, may use reflinks (XFS/Btrfs).  |
| io_uring            | SQ/CQ rings; batch submit; optional zero syscall.|
+---------------------+--------------------------------------------------+
```

#### io_uring (since 5.1, Mar 2019)

A pair of shared-memory rings between the kernel and user. The user fills **Submission Queue Entries (SQEs)** and the kernel produces **Completion Queue Entries (CQEs)**. With `IORING_SETUP_SQPOLL` the kernel has a thread polling the SQ and you can do I/O without a single syscall.

```
SQ ring (user produces, kernel consumes)
+-----+-----+-----+-----+-----+
| SQE | SQE | SQE | SQE | SQE |
+--^--+-----+-----+-----+-----+
   |
   user writes head; kernel reads head (or sqthread reads it)


CQ ring (kernel produces, user consumes)
+-----+-----+-----+-----+-----+
| CQE | CQE | CQE | CQE | CQE |
+--^--+-----+-----+-----+-----+
   |
   user reads tail; kernel writes tail
```

Key concepts:

- **Registered fds** (`io_uring_register(IORING_REGISTER_FILES)`) — pre-cache fd tables for `IOSQE_FIXED_FILE`.
- **Registered buffers** (`IORING_REGISTER_BUFFERS`) — pre-pin pages, eliminate per-op pin overhead.
- **Linked SQEs** (`IOSQE_IO_LINK`) — chain ops; B does not start until A completes successfully.
- **Multishot** (`IORING_OP_RECV` with `IORING_RECV_MULTISHOT`) — one SQE produces many CQEs.
- **Op codes** — `IORING_OP_READ`, `_WRITE`, `_RECV`, `_SEND`, `_ACCEPT`, `_OPENAT`, `_CLOSE`, `_STATX`, `_FALLOCATE`, `_TIMEOUT`, `_FSYNC`, `_NOP` (testing).

Setup-time flags worth knowing:

```
IORING_SETUP_SQPOLL        # kernel poll thread; no syscall to submit
IORING_SETUP_IOPOLL        # poll for completions (NVMe);  no IRQ
IORING_SETUP_CQSIZE        # custom CQ size (default 2x SQ)
IORING_SETUP_ATTACH_WQ     # share workers with another ring
IORING_SETUP_SQ_AFF        # pin SQ poll thread to a CPU
IORING_SETUP_COOP_TASKRUN  # since 5.19; cooperative task-run; lower latency
IORING_SETUP_DEFER_TASKRUN # since 6.1; even lower-overhead taskrun
IORING_SETUP_SINGLE_ISSUER # since 6.0; allow optimization for single-thread issuer
```

Library `liburing` is the supported wrapper; raw syscalls (`io_uring_setup`, `io_uring_enter`, `io_uring_register`) work but are tedious. Linking: `-luring`.

Sample minimal C:

```c
#include <liburing.h>
#include <stdio.h>
#include <fcntl.h>

int main(void) {
    struct io_uring ring;
    io_uring_queue_init(8, &ring, 0);
    int fd = open("/etc/hostname", O_RDONLY);
    char buf[256];
    struct io_uring_sqe *sqe = io_uring_get_sqe(&ring);
    io_uring_prep_read(sqe, fd, buf, sizeof buf, 0);
    io_uring_submit(&ring);
    struct io_uring_cqe *cqe;
    io_uring_wait_cqe(&ring, &cqe);
    printf("read %d bytes: %.*s", cqe->res, cqe->res, buf);
    io_uring_cqe_seen(&ring, cqe);
    io_uring_queue_exit(&ring);
}
```

Build: `gcc -O2 -o ur ur.c -luring`.

Security note: io_uring has been the source of a stream of CVEs (kernel vulnerabilities in op handlers, ring submission races, overflow in the worker pool). Some security-conscious distros disable it via `kernel.io_uring_disabled=2`. Container runtimes (Docker, containerd) may also block the syscall via seccomp by default. If your container fails `io_uring_setup` with EPERM, that is the cause.

### Network Internals

#### The Packet Path (RX)

```
NIC PHY --DMA--> RX ring buffer in RAM
                       |
                       v
            HW raises MSI-X interrupt on a CPU
                       |
                       v
            Driver IRQ handler (top half) --> schedule NAPI
                       |
                       v
            ksoftirqd / softirq NET_RX_SOFTIRQ
                       |
                       v
            napi->poll() drains ring up to budget (default 64)
                       |
                       v
            netif_receive_skb() -> RPS/RFS may steer to another CPU
                       |
                       v
            ip_rcv() -> tcp_v4_rcv() / udp_rcv() / ...
                       |
                       v
            sk_data_ready() wakes app blocked in epoll_wait/recv
                       |
                       v
            App reads from socket buffer (sk_receive_queue)
```

#### NAPI (New API)

NAPI mixes interrupts with polling: one IRQ per packet kills perf above 100k pps. Instead, the IRQ handler **schedules** NAPI and returns; the softirq later **polls** the ring for up to `budget` packets, then re-enables the IRQ. Inspect `/proc/net/softnet_stat` (one line per CPU; columns: total, dropped, time-squeezed, ...).

```
NAPI lifecycle for one RX queue:

  HW: packet DMAed into ring slot N
       |
       v
  HW raises MSI-X for this queue
       |
       v
  IRQ top half (driver):
       napi_schedule(&q->napi);          // mark NAPI runnable
       disable_irq_for_this_queue();     // we'll re-enable after polling
       return IRQ_HANDLED;
       |
       v
  softirq NET_RX_SOFTIRQ runs (or wakes ksoftirqd if busy):
       net_rx_action():
         loop budget times or until lists empty:
           napi->poll(napi, budget);     // driver-supplied poll
       |
       v
  Driver poll:
       for each ready RX descriptor:
         skb = build_skb(buf);
         napi_gro_receive(napi, skb);    // GRO may merge several
       |
       v
  netif_receive_skb() -> ip_rcv() -> tcp/udp_rcv() -> sk->sk_data_ready()
       |
       v
  If poll hit budget: softirq returns, will re-poll next pass.
  If poll drained: napi_complete_done() -> re-enable IRQ.
```

Per-CPU softirq budget is governed by:

```
net.core.netdev_budget          # max packets per softirq cycle (default 300)
net.core.netdev_budget_usecs    # max time per softirq cycle (default 2000 µs)
net.core.netdev_max_backlog     # input queue size when RPS routes between CPUs
net.core.dev_weight             # legacy per-NAPI weight, default 64
```

Busy-poll mode (set `SO_BUSY_POLL` on a socket) tells the kernel to spin in `recv()` instead of sleeping, polling NAPI directly — sub-microsecond latency at the cost of a hot CPU.

#### sk_buff (the skb) Lifecycle

```
struct sk_buff {
    struct sk_buff       *next, *prev;  /* queue links */
    struct sock          *sk;
    struct net_device    *dev;
    char                  cb[48];       /* control block, per-protocol */
    unsigned int          len, data_len;
    __u16                 mac_len, hdr_len;
    /* Pointers into the linear data area + paged frags: */
    sk_buff_data_t        tail;
    sk_buff_data_t        end;
    unsigned char        *head, *data;
    struct skb_shared_info *shinfo;     /* page frags, GSO state */
    ...
};

Data layout:

  head            data        tail            end
   |               |           |               |
   v               v           v               v
  [headroom][....headers + payload....][tailroom]

  + paged fragments in shinfo->frags[] (GSO/scatter-gather)
```

`skb_pull`, `skb_push`, `skb_reserve`, `skb_put` move pointers without copying.

#### XDP (eXpress Data Path)

A BPF program runs **at the driver level** before an skb is even allocated, on the raw RX descriptor. Verdicts: `XDP_PASS`, `XDP_DROP`, `XDP_TX`, `XDP_REDIRECT`, `XDP_ABORTED`. Available since 4.8 (Oct 2016). With AF_XDP sockets (since 4.18) you can map the RX ring into userspace and bypass the kernel network stack entirely for the packets you choose.

```
+-----------+      +-----------+      +-----------+      +--------+
| RX desc   |----->| XDP prog  |----->| skb alloc |----->| stack  |
| (DMA buf) |      | verdict   |      | (PASS)    |      | (TCP)  |
+-----------+      +-----------+      +-----------+      +--------+
                       |
                       +--> DROP (free buffer, return to NIC)
                       +--> TX (rewrite headers, requeue same NIC)
                       +--> REDIRECT (to another NIC, AF_XDP, CPU map)
```

#### Socket Steering Knobs

- **SO_REUSEPORT** — many sockets bind the same port; kernel hashes connections across them. Spread accept() across N workers.
- **SO_INCOMING_CPU** — a hint to the kernel to deliver to the CPU the socket prefers.
- **RPS (Receive Packet Steering)** — software RX-queue spreading by hashing flow → CPU. `/sys/class/net/eth0/queues/rx-N/rps_cpus` is a CPU mask.
- **RFS (Receive Flow Steering)** — like RPS but tracks the consumer's last CPU and steers the flow there for cache locality. `/proc/sys/net/core/rps_sock_flow_entries`.
- **RSS (Receive Side Scaling)** — hardware version: NIC has multiple RX queues, hashes incoming flows across them.
- **GRO/GSO/LRO/TSO** — coalesce/segment offloads. GRO/GSO are software (and recommended); LRO/TSO are hardware (LRO often disabled for routers because it merges flows the kernel didn't expect).

#### Qdiscs and Traffic Control (tc)

The egress side has its own scheduling: every netdev has a **qdisc** (queueing discipline) attached. Classful qdiscs (HTB, HFSC) build trees; classless qdiscs (FIFO, fq_codel, fq, cake) do their work in one queue.

```
+---------------+----------------------------------------------------------+
| pfifo_fast    | Default in older kernels. Three priority bands.          |
+---------------+----------------------------------------------------------+
| fq_codel      | Default in modern distros. Per-flow CoDel; great default.|
+---------------+----------------------------------------------------------+
| fq            | Per-flow pacing; required for BBR congestion control.    |
+---------------+----------------------------------------------------------+
| cake          | All-in-one home-router qdisc; rate limit + AQM + flow    |
|               | fairness in one config.                                  |
+---------------+----------------------------------------------------------+
| htb           | Hierarchical Token Bucket; classful rate limiting.       |
+---------------+----------------------------------------------------------+
| netem         | Network emulation; drop, delay, reorder, corrupt.        |
+---------------+----------------------------------------------------------+
| mq            | Multi-queue root that places one qdisc per HW TX queue.  |
+---------------+----------------------------------------------------------+
```

`tc qdisc show dev eth0`. `tc qdisc replace dev eth0 root fq`. eBPF `clsact` qdisc is the modern way to attach BPF programs to ingress/egress without changing topology.

### Synchronization

#### Spinlocks

The simplest mutual-exclusion primitive. Spin in a tight loop until you can acquire. Never sleeps. **Holders may not block**. Use in interrupt context, in code holding any other spinlock, when the critical section is short.

```
+---------------------+-------------------------------------------------+
| Variant             | Use                                             |
+---------------------+-------------------------------------------------+
| raw_spinlock_t      | The base, no PREEMPT_RT promotion.              |
| spinlock_t          | Default; on PREEMPT_RT becomes a sleeping mutex.|
| rwlock_t            | Many readers OR one writer; readers can starve  |
|                     | writers, generally avoid in new code.           |
| seqlock_t           | Writer takes spinlock + bumps seq counter;      |
|                     | readers retry if seq changed mid-read.          |
+---------------------+-------------------------------------------------+

API: spin_lock(), spin_unlock(),
     spin_lock_irqsave(lock, flags) / spin_unlock_irqrestore(),
     spin_lock_bh() / spin_unlock_bh()  (disable softirqs)
```

#### Mutexes vs Semaphores

`struct mutex` — sleeping lock, holder-tracking, optimistic spin, can be used in PREEMPT_RT real-time without surprise. `struct semaphore` — counted (allows N concurrent holders); legacy in much of the kernel; `struct rw_semaphore` is the read-write variant and is widely used (e.g. `mmap_lock`).

#### RCU (Read-Copy-Update)

Readers pay essentially zero cost: enter a read-side critical section by `rcu_read_lock()` (a preempt_disable on classic, almost a no-op), publish via `rcu_assign_pointer()`, and *defer* freeing of the old version until all readers that might still hold it have left their critical sections — that interval is a **grace period**.

```
Writer flow:
   new = kmalloc(...);
   *new = updated;
   rcu_assign_pointer(global_ptr, new);   <-- publishes via wmb
   synchronize_rcu();                     <-- waits one grace period
   kfree(old);                            <-- safe; no reader holds old


Reader flow:
   rcu_read_lock();
   p = rcu_dereference(global_ptr);
   use(p);
   rcu_read_unlock();


Grace period diagram (CPUs in read-side noted as R, quiescent as Q):

  CPU 0:  R---R---Q-------Q-------Q
  CPU 1:  Q---R--------R-Q---Q
  CPU 2:  R---Q-Q-Q-Q-Q-Q-Q
                ^                  ^
                |                  |
       writer calls         all CPUs have
       synchronize_rcu()    been Q at least once
                            => grace period over
                            => callbacks fire
```

`call_rcu(head, free_fn)` queues a callback to run *after* the next grace period without blocking the writer. Use this in atomic contexts where `synchronize_rcu()` is not allowed.

RCU flavors:

- **Classic RCU** (`rcu_read_lock`, `synchronize_rcu`, `call_rcu`) — blocking grace periods are tied to scheduler preemption.
- **SRCU (Sleepable RCU)** — for read-side critical sections that can sleep. Each subsystem creates its own `srcu_struct` so a stuck reader only delays its own subsystem.
- **Tasks RCU** — used for tracing/BPF; grace period waits for tasks to voluntarily schedule.
- **Tasks-Trace RCU** — even more specialized for sleepable BPF programs.

The CONFIG knobs that matter: `CONFIG_TREE_RCU` (default for SMP), `CONFIG_PREEMPT_RCU` (when `CONFIG_PREEMPT=y`), and `CONFIG_TINY_RCU` for UP embedded. RCU statistics are visible at `/sys/kernel/debug/rcu/`. The `rcu_nocbs=2-7` boot param offloads call_rcu callback execution onto dedicated `rcuop` threads instead of softirqs on those CPUs — useful for nohz_full real-time isolation.

#### Atomic Ops

```
+-----------------------+-------------------------------------------+
| API                   | Notes                                     |
+-----------------------+-------------------------------------------+
| atomic_t / atomic64_t | Wrappers around int / long with cmpxchg.  |
| atomic_inc(&v)        | Increment, no return.                     |
| atomic_inc_return(&v) | Increment, return new value.              |
| atomic_dec_and_test() | Decrement, return true if zero.           |
| atomic_cmpxchg(&v,o,n)| Compare-and-swap, returns prev value.     |
| atomic_xchg(&v, n)    | Atomic exchange.                          |
| atomic_long_*         | For pointers / long.                      |
+-----------------------+-------------------------------------------+
```

#### Memory Barriers

```
+-----------+--------------------------------------------------------+
| Macro     | Meaning                                                |
+-----------+--------------------------------------------------------+
| smp_mb()  | Full barrier: all loads/stores before complete before  |
|           | any after.                                             |
| smp_rmb() | Read barrier (loads).                                  |
| smp_wmb() | Write barrier (stores).                                |
| barrier() | Compiler-only barrier (no CPU fence).                  |
| READ_ONCE(x) / WRITE_ONCE(x) — prevent compiler from tearing or   |
|     reordering a single access; the right way to access shared    |
|     plain (non-atomic) variables in lockless code.                |
+-----------+--------------------------------------------------------+
```

#### Per-CPU Variables

`DEFINE_PER_CPU(int, counter);` then `this_cpu_inc(counter)` — no lock, no atomic, just a CPU-relative load/store. Aggregated in user-visible counters by `for_each_possible_cpu(cpu) total += per_cpu(counter, cpu);`. Many global stats (network packets, syscalls per second) are per-CPU summed.

Two flavors of per-CPU access:

- `this_cpu_*` — preempt-disabled internally, single instruction on x86 (`fs:` segment-relative access).
- `__this_cpu_*` — caller is responsible for disabling preempt; slightly faster but easier to misuse.

Per-CPU is the default for high-frequency counters precisely because reading-back is rare and the aggregation cost is paid once. The downside is **memory amplification** — a 4-byte counter takes 4*N bytes for N CPUs, but the cache-line padding to avoid false sharing pushes it to roughly 64*N. For 128-CPU boxes, big arrays of per-CPU counters get expensive in cache footprint.

### Scheduler Architecture

#### CFS (Completely Fair Scheduler) — default 2.6.23 through 6.5

CFS picks the task with the smallest **vruntime** (virtual runtime). Each runnable task is in a per-CPU **red-black tree** keyed by vruntime; the leftmost node is the next-to-run.

```
struct sched_entity {
    struct load_weight   load;       /* nice-to-weight conversion */
    struct rb_node       run_node;   /* node in cfs_rq->tasks_timeline */
    unsigned int         on_rq;
    u64                  exec_start;
    u64                  sum_exec_runtime;
    u64                  vruntime;   /* the key in the rb-tree */
    ...
};

cfs_rq (per CPU):

       rb-tree keyed by vruntime
                   o
                  / \
                 o   o
                / \   \
            (left)    o     <- leftmost = next pick
              ^
              |
        cfs_rq->min_vruntime tracks the smallest vruntime
        seen so newly-woken or migrated tasks can be
        floored here, preventing freeloaders from running
        forever.
```

vruntime advances at a rate inversely proportional to the task's nice value (weight). A nice -19 task accumulates vruntime slowly (so it spends more wall time at the leftmost position). Nice +19 accumulates fast.

The weight table for nice values is logarithmic; each step of nice multiplies weight by ~1.25. Nice 0 weight = 1024. Nice -20 weight ≈ 88761. Nice +19 weight ≈ 15. This means nice 0 vs nice 5 sees about a 3x CPU share difference.

Group scheduling (CONFIG_FAIR_GROUP_SCHED, default y) extends CFS so cgroups become first-class scheduling entities. The cfs_rq on a CPU may contain `sched_entity` objects that themselves represent groups (their own cfs_rq), recursing. This is how the cgroup `cpu.weight` (cgroup v2) and `cpu.shares` (v1) work.

#### Real-Time Classes

Above `SCHED_OTHER`/`SCHED_BATCH`/`SCHED_IDLE` (which are CFS-managed) are the real-time classes:

```
+------------------+----------------------------------------------+
| SCHED_FIFO       | FIFO; runs until it yields or is preempted   |
|                  | by a higher-priority RT task.                |
+------------------+----------------------------------------------+
| SCHED_RR         | Like FIFO but with a time slice (default     |
|                  | 100ms via /proc/sys/kernel/sched_rr_timeslice|
|                  | _ms).                                        |
+------------------+----------------------------------------------+
| SCHED_DEADLINE   | EDF + CBS. Specify (runtime, deadline,       |
|                  | period). Highest priority.                   |
+------------------+----------------------------------------------+
| SCHED_OTHER      | Default CFS class.                           |
+------------------+----------------------------------------------+
| SCHED_BATCH      | CFS but the wakeup boost is suppressed —     |
|                  | for batch jobs.                              |
+------------------+----------------------------------------------+
| SCHED_IDLE       | Lower than nice 19 — only runs when nothing  |
|                  | else wants the CPU.                          |
+------------------+----------------------------------------------+

Stop class > deadline > RT > fair > idle  (decreasing priority)
```

To make sense of priorities at the kernel level: the kernel uses a single `prio` value internally where lower = higher priority. RT priorities map: kernel prio = 99 - rt_priority (so user RT priority 99 → kernel prio 0, the highest possible RT). `SCHED_DEADLINE` tasks are above all RT, so their kernel prio is below 0 (they sit in `dl_rq`). `SCHED_OTHER` tasks have kernel prio = 100 + nice + 20 (so nice 0 → 120; nice -20 → 100; nice +19 → 139). Idle tasks have prio 140.

`SCHED_DEADLINE` is the most powerful and most dangerous. You declare a tuple `(runtime, deadline, period)` meaning "give me `runtime` µs of CPU within every `period` µs, and miss the deadline at most by `deadline - period`." The kernel admission-controls (`/proc/sys/kernel/sched_deadline_period_min_us`, `_max_us`) and won't accept overcommitted sets. Use for media decoders, robotics control loops, or audio.

#### Load Balancer and Sched Domains

The kernel models the CPU topology as a hierarchy of **sched domains**: SMT (hyperthread) → MC (cores in a cache cluster) → DIE (the socket) → NUMA (sockets) → cross-NUMA. Load balancing climbs the hierarchy with progressively cheaper rebalances at the lower levels and only occasional cross-NUMA pulls.

```
   NUMA  domain  (entire machine)
       |
       +-- DIE   domain  (socket 0)
       |       |
       |       +-- MC  domain (LLC group 0)
       |       |        |
       |       |        +-- SMT domain (core 0: cpus 0 and 8)
       |       |        +-- SMT domain (core 1: cpus 1 and 9)
       |       +-- MC  domain (LLC group 1)
       |
       +-- DIE   domain  (socket 1)
                |
                +-- ...
```

#### EAS (Energy Aware Scheduling) — ARM big.LITTLE

The scheduler can read an Energy Model from device tree and pick CPUs to minimize energy * task throughput product. Used heavily in Android. x86 typically does not enable EAS.

#### EEVDF (Earliest Eligible Virtual Deadline First) — replacing CFS since 6.6

Linux 6.6 (Oct 2023) replaced CFS with EEVDF. The mental model is the same — a vruntime-like clock — but each task additionally has a **virtual deadline**, and the scheduler picks the eligible task with the earliest virtual deadline. Concretely:

- Eligibility: task's vruntime <= a system-wide eligibility floor.
- Among eligible tasks, pick smallest virtual deadline.
- nice now affects time-slice length (shorter slices for higher priority interactive).

The user-visible knobs are still `nice` and `chrt`; most tunables in `/proc/sys/kernel/sched_*` carry over but a few CFS-specific ones were retired. Two new sysctls landed with EEVDF: `kernel.sched_base_slice_ns` (the target slice length) replaced the old `sched_min_granularity_ns`. The latency target tunable `sched_latency_ns` was removed; EEVDF infers latency targets from slice length and weight.

Why the change? CFS had ad-hoc heuristics piled up over a decade (wakeup preemption, vruntime fudges for sleeping tasks, the GENTLE_FAIR_SLEEPERS / NEXT_BUDDY / LAST_BUDDY toggles). EEVDF replaces them with a single proven algorithm from a 1995 paper (Stoica et al., "Earliest Eligible Virtual Deadline First"). Latency-sensitive interactive tasks set a smaller slice and get an earlier virtual deadline by construction; CPU-hog batch tasks naturally pick up bigger slices.

#### Wakeup Path

When a task is woken (e.g. an interrupt completes a wait, a sleeping task is `wake_up()`-ed), `try_to_wake_up()` runs:

1. Mark task RUNNING.
2. Pick a target CPU via `select_task_rq_*()`. Hooks per sched class. For CFS this looks at `WAKE_AFFINE` (prefer waker's CPU if cache is hot), avoiding overloaded CPUs, and respecting affinity / cpuset constraints.
3. If the target is not the current CPU, IPI it via `resched_curr()`.
4. The target CPU enters its scheduler at the next opportunity (preempt point, idle exit, timer tick).

Wake latency on a busy box is dominated by IPI delivery (~1 µs) plus how quickly the target reaches a preempt point. With `CONFIG_PREEMPT=y`, preempt points are nearly everywhere; with `CONFIG_PREEMPT_VOLUNTARY` only at explicit `cond_resched()` calls.

### NUMA Topology

```
Two-socket system:

  +------ Node 0 ------+        +------ Node 1 ------+
  | CPUs 0-15 (SMT)    |        | CPUs 16-31 (SMT)   |
  | DRAM   384 GB      | <----> | DRAM   384 GB      |
  | LLC    64 MB       |  UPI   | LLC    64 MB       |
  +--------------------+        +--------------------+

       Distance matrix (numactl -H):
       node 0   node 1
node 0   10       21
node 1   21       10

10 = local; 21 = remote (about 2x latency, ~30 ns -> ~70-90 ns).
```

`numactl --hardware` prints the matrix and the per-node free memory. `numactl --membind=0 --cpunodebind=0 ./app` pins both memory and CPUs to node 0 (the strict version; use `--preferred=0` to allow falling back to node 1).

`set_mempolicy(2)` is the syscall behind `numactl`. Policies: `MPOL_DEFAULT`, `MPOL_PREFERRED`, `MPOL_BIND`, `MPOL_INTERLEAVE`, `MPOL_LOCAL` (since 5.10).

#### AutoNUMA (numa_balancing)

Since 3.8 (Feb 2013) the kernel itself can migrate pages and tasks. Periodically the kernel marks PTEs in a task's mm with `_PROT_NONE`, triggering a fault on next access. The fault handler records which **node** accessed the page; if the page is on the wrong node, the kernel migrates it (`numa_balancing_migrate`). Inspect `/sys/devices/system/node/node*/numastat`:

```
numa_hit       12345678   <- alloc satisfied on the requested node
numa_miss      234567     <- alloc spilled to another node
numa_foreign   234567     <- this node served allocs requested elsewhere
local_node     12345000   <- alloc by a CPU on this node
other_node     678        <- alloc by a CPU on another node
```

Tunables: `kernel.numa_balancing` (0/1), `kernel.numa_balancing_scan_period_min_ms`, `_max_ms`.

Workloads that benefit from disabling autobalance: anything where you've already pinned with `numactl` (the periodic PROT_NONE faults are pure overhead), and very fault-heavy mmap workloads. Workloads that benefit from leaving it on: long-running JVM-style workloads where threads and pages drift naturally and you want the kernel to fix it.

Memory bandwidth is also asymmetric. On AMD EPYC and Intel multi-socket systems, cross-socket bandwidth is via UPI/Infinity Fabric and runs about 1/2 to 1/3 of local DRAM bandwidth. Numa-unfriendly workloads see up to 50% throughput loss on memory-bound benchmarks (STREAM Triad, in-memory hash joins, graph traversals).

Userspace API summary:

```
numactl --hardware                       # show topology + free
numactl --show                           # show current process policy
numactl -N 0 -m 0 cmd                    # equivalent to --cpunodebind=0 --membind=0
numactl --interleave=all cmd
numactl --preferred=0 cmd                # soft preference; allow fallback
numastat -p PID                          # per-process numa accounting
migratepages PID NEW_NODE                # migrate a process's pages to another node
mbind(addr, len, mode, nodemask, ...)    # syscall: per-region policy
set_mempolicy(mode, nodemask, ...)       # syscall: process default policy
get_mempolicy(...)                       # query current policy
move_pages(pid, count, pages, nodes, status, flags)  # explicit page migration
```

`/sys/devices/system/node/node*/` exposes `meminfo`, `numastat`, `cpulist`, `cpumap`, `distance`, and `compact` (write 1 to compact this node).

### Custom Kernel Builds

#### Configuration

```
make defconfig         # generic config (uses arch/x86/configs/x86_64_defconfig)
make allyesconfig      # enable everything (CI test; results in a 1+ GB image)
make allnoconfig       # disable everything (minimal)
make tinyconfig        # smallest bootable
make olddefconfig      # take existing .config, default-answer new symbols
make menuconfig        # ncurses TUI
make nconfig           # newer ncurses TUI
make xconfig           # Qt GUI
make localmodconfig    # config based on what's currently loaded (lsmod)
make localyesconfig    # like above but build-in instead of module
```

#### Kconfig Language (in Kconfig files all over the tree)

```
config FOO_DRIVER
        tristate "Foo widget support"
        depends on PCI && X86
        select REGMAP_MMIO
        imply CRC32
        default m if X86_64
        help
          Say Y/M/N here. Module name: foo_driver.

choice
        prompt "Memory model"
        default SPARSEMEM_VMEMMAP
        config FLATMEM
                bool "Flat"
        config SPARSEMEM_VMEMMAP
                bool "Sparse + vmemmap"
endchoice

menu "Networking options"
config NET
        bool "Networking support"
        ...
endmenu
```

- `bool` → y/n; `tristate` → y/m/n.
- `depends on FOO` — option is invisible unless FOO is set.
- `select FOO` — auto-enable FOO (forced).
- `imply FOO` — soft enable FOO if its own deps are met.

#### Building

```
make -j$(nproc)        # build vmlinux + modules
make -j$(nproc) bzImage modules     # explicit
make headers_install INSTALL_HDR_PATH=/usr/local
sudo make modules_install           # /lib/modules/<ver>/
sudo make install                   # copies bzImage to /boot, runs hooks
sudo update-initramfs -c -k <ver>   # Debian/Ubuntu (or dracut on RPM)
sudo grub-mkconfig -o /boot/grub/grub.cfg
```

Artifacts:
- `vmlinux` — the uncompressed ELF kernel (huge, used by gdb / perf).
- `arch/x86/boot/bzImage` — the bootable compressed image.
- `*.ko` — kernel modules (loaded with `insmod`/`modprobe`).
- `System.map` — symbol-to-address map.

Out-of-tree modules:

```
# Build against running kernel:
cd /path/to/module
make -C /lib/modules/$(uname -r)/build M=$PWD modules

# DKMS for auto-rebuild on kernel upgrades:
sudo dkms add /usr/src/foo-1.0
sudo dkms build foo/1.0
sudo dkms install foo/1.0
```

A minimal kbuild Makefile for an out-of-tree module:

```make
obj-m := hello.o
KDIR  ?= /lib/modules/$(shell uname -r)/build
PWD   := $(shell pwd)

default:
	$(MAKE) -C $(KDIR) M=$(PWD) modules

clean:
	$(MAKE) -C $(KDIR) M=$(PWD) clean
```

A minimal hello-world kernel module (`hello.c`):

```c
#include <linux/init.h>
#include <linux/module.h>
#include <linux/kernel.h>

static int __init hello_init(void) {
    pr_info("hello: loaded\n");
    return 0;
}

static void __exit hello_exit(void) {
    pr_info("hello: unloaded\n");
}

module_init(hello_init);
module_exit(hello_exit);
MODULE_LICENSE("GPL");
MODULE_AUTHOR("you");
MODULE_DESCRIPTION("trivial");
```

`make`, `sudo insmod hello.ko`, `dmesg | tail -1`, `sudo rmmod hello`.

#### Boot Parameters vs sysfs vs sysctl

```
+----------------------+-----------------------------------------------+
| /proc/cmdline        | Set in bootloader; fixed for life of boot.    |
|                      | e.g. quiet splash isolcpus=2-7 nohz_full=2-7  |
+----------------------+-----------------------------------------------+
| /sys                 | Subsystem state, often runtime-tunable. e.g.  |
|                      | /sys/block/nvme0n1/queue/scheduler            |
+----------------------+-----------------------------------------------+
| /proc/sys (sysctl)   | Tunables exposed by `sysctl -a`. e.g.         |
|                      | vm.swappiness, net.ipv4.tcp_congestion_control|
+----------------------+-----------------------------------------------+
| module parameters    | Per-module, /sys/module/<name>/parameters/   |
|                      | e.g. /sys/module/nvme/parameters/io_timeout   |
+----------------------+-----------------------------------------------+
```

#### Bisecting a Regression

```
git clone https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git
cd linux
git bisect start
git bisect bad v6.7        # known-bad
git bisect good v6.6       # known-good
# git checks out a midpoint commit:
make olddefconfig && make -j$(nproc) bzImage modules
sudo make modules_install && sudo make install
sudo reboot
# After reboot, test the regression:
git bisect good            # or "git bisect bad"
# Repeat. After ~log2(N) iterations git prints the first bad commit.
git bisect reset           # restore your working tree
```

### Filesystems Quick Tour

A subsystem hop you'll inevitably hit while debugging memory or I/O:

- **VFS** — the abstraction layer. `struct file`, `struct inode`, `struct dentry`, `struct super_block`. Every FS provides operations vectors (`super_operations`, `inode_operations`, `file_operations`, `address_space_operations`). The VFS dispatches generic syscalls down through these vectors.
- **ext4** — copy-on-write metadata via journaling (jbd2). Default in many distros. Block-allocator: extent trees in inodes (since ext4); inode size 256 bytes default. Mount options of note: `noatime`, `data=writeback|ordered|journal`, `commit=N` (jbd2 commit interval, default 5 s), `journal_async_commit`, `barrier=0|1`.
- **xfs** — high-performance journaling FS, once SGI's IRIX FS. Allocation groups parallelize. Mount options: `noatime`, `inode64` (default since 3.7), `logbsize`, `allocsize`. Reflinks via `cp --reflink` (since 4.9 in xfs).
- **btrfs** — copy-on-write, subvolumes, snapshots, transparent compression (`zstd`, `lzo`, `zlib`), built-in checksums, multi-device RAID. Some RAID5/6 modes still considered unstable as of 6.x. Use `btrfs filesystem df`, `btrfs subvolume list`, `btrfs scrub`, `btrfs send`/`receive`.
- **overlayfs** — union FS used by container runtimes (Docker, containerd, podman). Lower=read-only image layers, upper=writable container layer.
- **tmpfs** — RAM-backed FS using page cache pages. `/dev/shm`, `/tmp` on systemd defaults.
- **fuse** — userspace filesystem driver framework.
- **bcachefs** — newer COW FS merged in 6.7 (Jan 2024). Aims to combine btrfs features with better performance.

VFS read path summary:

```
read(fd, buf, n)
   |
   v
ksys_read -> vfs_read -> fileops->read_iter (per FS)
                         (or generic_file_read_iter for page-cache backed FS)
   |
   v
generic_file_read_iter:
   for each page:
      grab page from address_space
      if not uptodate: trigger readahead via aops->readahead
      copy_to_iter() to user buffer
   |
   v
return bytes_read
```

Key tunables: `/sys/block/<dev>/queue/read_ahead_kb` (per-device readahead), `vm.page-cluster` (swap readahead), per-process `posix_fadvise()` calls.

### Time, Timers, and Tickless Mode

- **Tick** — periodic timer interrupt (HZ, default 250 or 1000). Drives accounting, RCU grace-period detection, scheduler bookkeeping. Configured via CONFIG_HZ.
- **NO_HZ_IDLE** (default) — skip ticks when CPU is idle.
- **NO_HZ_FULL** — skip ticks when CPU is running a single task. Boot param `nohz_full=2-7` to enable on a CPU range. Removes scheduler ticks on those CPUs entirely; great for HPC and real-time. Pairs with `isolcpus` and `rcu_nocbs`.
- **High-resolution timers (hrtimers)** — nanosecond-resolution timers used by `nanosleep`, `epoll_wait` timeouts, scheduler.
- **clocksource** — the time source. `tsc` (CPU timestamp counter) on modern x86; `arch_sys_counter` on ARM. `/sys/devices/system/clocksource/clocksource0/`.
- **clockevents** — the device delivering interrupts (lapic-deadline, hpet, etc).

Inspect: `cat /sys/devices/system/clocksource/clocksource0/current_clocksource`, `cat /proc/timer_list | head -40`.

### Linux Security Modules (LSM)

A stack of access-control hooks layered above DAC (file permissions, capabilities). One major module (or multiple stacked since 5.1):

- **SELinux** — Type Enforcement, role-based access. Default on RHEL/Fedora.
- **AppArmor** — path-based profiles. Default on Ubuntu/SUSE.
- **TOMOYO** — pathname learning mode.
- **Smack** — labels, simpler than SELinux.
- **Yama** — ptrace_scope and friends.
- **Lockdown** — restricts kernel-modifying ops on Secure Boot systems.
- **Landlock** — userspace-driven sandboxing API (since 5.13).
- **BPF LSM** — write LSM hooks as BPF programs (since 5.7).

Capabilities (`man 7 capabilities`) split root into ~40 named privileges (`CAP_NET_ADMIN`, `CAP_SYS_ADMIN`, `CAP_BPF`, `CAP_PERFMON`, ...). `getpcaps PID`, `capsh --print`. `CAP_SYS_ADMIN` is the catch-all that hasn't been split yet — minimize giving it.

### Boot Path (Brief)

```
BIOS / UEFI firmware
   |
   v
Bootloader (GRUB / systemd-boot)
   |  loads bzImage and initramfs into RAM, jumps to kernel entry
   v
Kernel (arch/x86/boot/header.S -> startup_64 -> start_kernel)
   |  initializes:
   |    early printk, page tables, percpu, scheduler, memory init,
   |    softirq, timekeeping, RCU, console, IRQ subsystem, drivers,
   |    rootfs probe.
   v
PID 1 (init / systemd) execs from /sbin/init
   |
   v
systemd targets bring up everything else
```

`dmesg --color=always | head -100` is your friend for boot-time issues. `systemd-analyze blame` lists slow units. `journalctl -k --boot` is the kernel ring buffer for the current boot.

### Crashes, Panics, and Postmortem

- **kernel oops** — non-fatal kernel exception. Process killed, kernel tainted; system continues.
- **kernel panic** — fatal. By default the box hangs; `kernel.panic = 10` reboots after 10 s.
- **soft lockup** — a CPU stuck in kernel mode > 20 s without scheduling. Watchdog warns. Tunable `kernel.softlockup_panic`.
- **hard lockup** — IRQs disabled > ~10 s. Watchdog (NMI-driven) panics if `kernel.hardlockup_panic=1`.
- **kdump / kexec** — load a second kernel into reserved memory (`crashkernel=...`) that boots on panic and dumps the dead kernel's core to disk via `makedumpfile`. Analyze with `crash`.

`/proc/sys/kernel/panic_on_oops`, `panic_on_warn`, `panic_on_unrecovered_nmi` — escalation knobs.

### Container Internals (Kernel View)

Containers are not a kernel feature — they're a userspace pattern combining several kernel features. From the kernel's view a container is:

```
+---------------------------------------------------+
| process tree                                      |
|   in pid namespace P                              |
|   in mount namespace M                            |
|   in net namespace N                              |
|   in user namespace U (optional, sometimes)       |
|   in uts namespace H                              |
|   in ipc namespace I                              |
|   in cgroup v2 group G with controllers enabled   |
|   under seccomp filter S                          |
|   under capability set C                          |
|   under apparmor/selinux profile P (LSM)          |
|   chroot/pivot_root'ed to image rootfs            |
+---------------------------------------------------+
```

Each namespace is a kernel object referenced by `/proc/<pid>/ns/<type>` (a magic symlink). `unshare(2)` creates a new namespace for the calling process; `setns(2)` joins an existing one. `clone(2)` with `CLONE_NEWPID|...` creates the child in new namespaces.

Cgroup v2 (default in modern systemd, since 4.5 stable) is a single hierarchy under `/sys/fs/cgroup`. Each cgroup has `cgroup.procs`, `cgroup.controllers`, `cgroup.subtree_control`, plus controller files (`memory.max`, `cpu.weight`, etc.). PSI metrics expose pressure as percent-time-stalled.

Container security is the intersection of:

- **namespaces** — what you can see.
- **capabilities** — what you can do as "root."
- **seccomp** — which syscalls are allowed.
- **cgroups** — how much you can use.
- **LSM** — finer-grained access control.

All of these the kernel enforces uniformly; the runtime (containerd, crun, runc) just sets them up before exec. There is no "container ID" the kernel knows about — only the union of the above objects.

#### A 30-Day Reading Plan

If you want to go deeper systematically:

- **Week 1: VFS + page cache.** Read `Documentation/filesystems/vfs.rst` end-to-end. Read the comments in `mm/filemap.c`. Understand `address_space_operations`. Do `bpftrace` on `vfs_read` until you see the path.
- **Week 2: Networking stack.** Read `Documentation/networking/scaling.rst`. Read `net/core/dev.c` (skb path in `__netif_receive_skb_core`). Read `net/ipv4/tcp_input.c` top of file. `bpftrace` on tcp_v4_rcv, tcp_ack.
- **Week 3: Scheduler.** Read `kernel/sched/fair.c` (or the EEVDF rewrite if on 6.6+). `bpftrace` on `sched_switch`, plot run-queue latency.
- **Week 4: BPF + io_uring.** Read `Documentation/bpf/index.rst`. Write a BPF program with `libbpf` (CO-RE). Write a simple io_uring server.

Plus, by then you'll know which `kernel-tuning/*` sheets to spend more time in.

### Version Notes

The kernel changes fast and a "current best practice" from 2018 may be wrong in 2026. Headline transitions worth knowing:

- **3.8** (Feb 2013) — Initial **NUMA balancing** (AutoNUMA). PROT_NONE-fault scanning landed.
- **3.13** (Jan 2014) — **blk-mq** (Multi-Queue Block Layer) merged. SCSI converted to it later.
- **3.18** (Dec 2014) — **eBPF** classic-BPF replacement landed.
- **4.5** (Mar 2016) — **cgroup v2** (unified hierarchy) declared stable.
- **4.6** (May 2016) — `vm.watermark_scale_factor` sysctl added.
- **4.8** (Oct 2016) — **XDP** (eXpress Data Path) merged.
- **4.9** (Dec 2016) — **BBR** congestion control merged.
- **4.15** (Jan 2018) — **KPTI** (Kernel Page Table Isolation) for Meltdown. **PREEMPT_RT** dropped some merging.
- **4.18** (Aug 2018) — **AF_XDP** sockets. Block layer single-queue removed (SQ devices forced through blk-mq).
- **5.1** (May 2019) — **io_uring** initial release. SQ/CQ + 7 ops. Subsequent every release adds ops/features.
- **5.6** (Mar 2020) — **WireGuard** in-tree. **Multi-path TCP (MPTCP)** initial.
- **5.7** (May 2020) — **BPF LSM**. `vm.compaction_proactiveness`.
- **5.10** (Dec 2020) — `MPOL_LOCAL` policy formalized. Long-term support kernel.
- **5.13** (Jun 2021) — **Landlock** LSM merged.
- **5.15** (Nov 2021) — **ksmbd** (in-kernel SMB3 server). Long-term support.
- **5.19** (Jul 2022) — Initial RISC-V profile maturity. ARM POE added.
- **6.1** (Dec 2022) — **MGLRU** (Multi-Generational LRU) merged. Long-term support.
- **6.2** (Feb 2023) — Default behavior of **transparent hugepages** for some servers. **maple tree** replacing rb-tree for VMA storage.
- **6.6** (Oct 2023) — **EEVDF** scheduler replaces CFS. `kernel.io_uring_disabled` sysctl.
- **6.7** (Jan 2024) — **bcachefs** merged.
- **6.8** (Mar 2024) — initial **nftables** scalability for SO_REUSEPORT BPF.
- **6.10** (Jul 2024) — `mseal()` syscall (memory sealing).
- **6.12** (Nov 2024) — **PREEMPT_RT** fully merged (no separate patch tree).

When you see a tutorial dated before 2020 mentioning `aio(7)`, prefer io_uring; before 2014 mentioning the legacy block layer, assume blk-mq applies. EEVDF tunables in tutorials before late 2023 are CFS-specific and may not exist anymore.

## Hands-On

#### Memory introspection

```bash
$ cat /proc/buddyinfo
Node 0, zone      DMA      0      1      1      1      1      1      1      0      1      1      3
Node 0, zone    DMA32    412    211    134     74     61     30     14      8      6      4    811
Node 0, zone   Normal   3445   1041    532    266    138     74     32     16      8      4    192

$ sudo cat /proc/slabinfo | head -20
slabinfo - version: 2.1
# name            <active_objs> <num_objs> <objsize> <objperslab> <pagesperslab> : tunables ...
nf_conntrack          12388      13050        320          12             1 : tunables ...
ovl_inode               890        920        688          11             2 : tunables ...
mqueue_inode_cache       17         28       1152          14             4 : tunables ...
hugetlbfs_inode_cache    14         32        608          13             2 : tunables ...
ext4_inode_cache    1232114    1233000       1080          15             4 : tunables ...
proc_inode_cache       6233       6240        680          12             2 : tunables ...
shmem_inode_cache      1342       1344        720          11             2 : tunables ...
dentry              512344     517482        192          21             1 : tunables ...
inode_cache         123451     124000        608          13             2 : tunables ...
kmalloc-8k             1024       1024       8192           4             8 : tunables ...
kmalloc-4k             4096       4096       4096           8             8 : tunables ...
kmalloc-2k             8192       8192       2048          16             8 : tunables ...
kmalloc-1k            16384      16384       1024          32             8 : tunables ...
kmalloc-512           32768      32768        512          32             4 : tunables ...
kmalloc-256           81920      81920        256          32             2 : tunables ...
task_struct            1894       1932       7232           4             8 : tunables ...
mm_struct               890        896       1088          15             4 : tunables ...
files_cache             550        584        704          11             2 : tunables ...

$ cat /proc/zoneinfo | head -40
Node 0, zone      DMA
  pages free     3974
        min      71
        low      88
        high     105
        spanned  4095
        present  3998
        managed  3974
        protection: (0, 2766, 31872, 31872)
  nr_free_pages 3974
  ...
Node 0, zone    DMA32
  pages free     708160
        min      12750
        low      15937
        high     19124
        ...

$ cat /proc/meminfo | grep -E 'Active|Inactive|Dirty|Writeback'
Active:         12345678 kB
Inactive:        4567890 kB
Active(anon):    8123456 kB
Inactive(anon):    23456 kB
Active(file):    4222222 kB
Inactive(file): 4544434 kB
Dirty:              4567 kB
Writeback:             0 kB
WritebackTmp:          0 kB

$ cat /proc/sys/vm/dirty_ratio /proc/sys/vm/dirty_background_ratio
20
10

$ sudo slabtop -o | head -15
 Active / Total Objects (% used)    : 1234567 / 1456789 (84.7%)
 Active / Total Slabs (% used)      : 78901 / 78901 (100.0%)
 Active / Total Caches (% used)     : 132 / 174 (75.9%)
 Active / Total Size (% used)       : 387.23M / 444.56M (87.1%)
 Minimum / Average / Maximum Object : 0.01K / 0.30K / 8.00K

  OBJS  ACTIVE  USE OBJ SIZE  SLABS OBJ/SLAB CACHE SIZE NAME
1232114 1232100 100%    1.05K  82141      15  1314256K ext4_inode_cache
 517482  512344  99%    0.19K  24642      21    98568K dentry

$ cat /sys/kernel/mm/transparent_hugepage/enabled
always [madvise] never

$ cat /proc/sys/vm/swappiness
60

$ free -h
               total        used        free      shared  buff/cache   available
Mem:           125Gi        24Gi       2.1Gi       423Mi        99Gi        99Gi
Swap:          8.0Gi          0B       8.0Gi
# Note: free is low (2.1G) because buff/cache is 99G; available is 99G — that
# is the right number to look at. The page cache is releasable on demand.
```

#### NUMA

```bash
$ numactl --hardware
available: 2 nodes (0-1)
node 0 cpus: 0 1 2 3 4 5 6 7 16 17 18 19 20 21 22 23
node 0 size: 391232 MB
node 0 free: 9821 MB
node 1 cpus: 8 9 10 11 12 13 14 15 24 25 26 27 28 29 30 31
node 1 size: 391232 MB
node 1 free: 12455 MB
node distances:
node   0   1
  0:  10  21
  1:  21  10

$ numactl --membind=0 --cpunodebind=0 ./yourapp
# Launches yourapp; both CPU and memory pinned to node 0 (strict).

$ cat /sys/devices/system/node/node0/numastat
numa_hit 9182734523
numa_miss 12345
numa_foreign 67890
interleave_hit 0
local_node 9182000000
other_node 734523

$ numactl --interleave=all ./batch_job
# Spread allocations round-robin across all nodes (best for big sequential
# scans where you want max aggregate bandwidth).

$ cat /proc/sys/kernel/numa_balancing
1
```

#### Network

```bash
$ cat /proc/interrupts | grep eth0 | head -10
 145:    1234567   1233214      ...    PCI-MSI-X eth0-rx-0
 146:     888777    870233      ...    PCI-MSI-X eth0-rx-1
 147:    9123456   9100000      ...    PCI-MSI-X eth0-rx-2
 148:    8132134   8129990      ...    PCI-MSI-X eth0-rx-3
# IRQ counts per CPU; uneven counts = a CPU is doing all the network work.

$ ethtool -S eth0 | head -20
NIC statistics:
     rx_packets: 12345678901
     tx_packets: 9876543210
     rx_bytes: 8888888888888
     tx_bytes: 7777777777777
     rx_dropped: 0
     tx_dropped: 0
     rx_errors: 0
     tx_errors: 0
     rx_csum_errors: 0
     rx_no_buffer_count: 234
     rx_missed_errors: 0
     rx_long_length_errors: 0
     rx_short_length_errors: 0
     ...

$ ethtool -k eth0 | head -20
Features for eth0:
rx-checksumming: on
tx-checksumming: on
generic-segmentation-offload: on
generic-receive-offload: on
large-receive-offload: off
tcp-segmentation-offload: on
udp-fragmentation-offload: off [fixed]
ntuple-filters: off
receive-hashing: on

$ ss -tnpi state established | head -10
State    Recv-Q  Send-Q   Local Address:Port   Peer Address:Port
ESTAB    0       0        10.0.0.5:443         10.0.0.99:54321
         cubic wscale:7,7 rto:204 rtt:3.123/1.5 ato:40 mss:1448 cwnd:10 ssthresh:7
         bytes_acked:12345 bytes_received:67890 segs_out:88 segs_in:90

$ ip -s link show eth0
2: eth0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc fq_codel state UP
    link/ether 00:11:22:33:44:55 brd ff:ff:ff:ff:ff:ff
    RX: bytes  packets  errors  dropped overrun mcast
    8888888888 12345678  0       0       0       12345
    TX: bytes  packets  errors  dropped overrun carrier
    7777777777  9876543  0       0       0       0

$ cat /proc/net/softnet_stat | awk '{print "cpu="NR-1" total="$1" dropped="$2" squeezed="$3}'
cpu=0 total=12345678 dropped=0 squeezed=23
cpu=1 total=8765432  dropped=0 squeezed=11
# squeezed = NAPI ran out of budget; not catastrophic but persistent
# nonzero suggests increasing /sys/class/net/eth0/gro_flush_timeout.
```

#### Scheduler

```bash
$ cat /sys/kernel/debug/sched_features
GENTLE_FAIR_SLEEPERS START_DEBIT NEXT_BUDDY LAST_BUDDY CACHE_HOT_BUDDY
WAKEUP_PREEMPTION HRTICK_DL NO_HRTICK NO_DOUBLE_TICK NONTASK_CAPACITY
TTWU_QUEUE NO_SIS_PROP SIS_UTIL ...
# Toggle a feature live: echo NO_HRTICK | sudo tee /sys/kernel/debug/sched_features

$ chrt -m
SCHED_OTHER min/max priority    : 0/0
SCHED_FIFO  min/max priority    : 1/99
SCHED_RR    min/max priority    : 1/99
SCHED_BATCH min/max priority    : 0/0
SCHED_IDLE  min/max priority    : 0/0
SCHED_DEADLINE min/max priority : 0/0

$ chrt -p $$
pid 1234's current scheduling policy: SCHED_OTHER
pid 1234's current scheduling priority: 0

$ sudo chrt -f -p 50 12345    # set pid 12345 to SCHED_FIFO prio 50

$ ls /proc/sys/kernel/sched_*
sched_autogroup_enabled            sched_rt_period_us
sched_cfs_bandwidth_slice_us       sched_rt_runtime_us
sched_child_runs_first             sched_rr_timeslice_ms
sched_deadline_period_max_us       sched_schedstats
sched_deadline_period_min_us       sched_tunable_scaling
sched_energy_aware                 sched_util_clamp_max
sched_nr_migrate                   sched_util_clamp_min

$ cat /proc/sys/kernel/sched_rr_timeslice_ms
100

$ taskset -cp 0-3 $$
pid 1234's current affinity list: 0-31
pid 1234's new affinity list: 0-3
```

#### Tracing and profiling

```bash
$ sudo bpftrace -e 'kprobe:vfs_read { @[comm] = count(); }'
Attaching 1 probe...
^C
@[bash]: 12
@[python]: 245
@[postgres]: 18923

$ sudo perf top -e cycles:k
# Top kernel symbols by sampling. Press 'a' for assembly view, 'h' for help.

$ sudo perf record -ag sleep 5 && sudo perf report --stdio | head -30
# Captures all CPUs for 5 s with call graphs (-g), then dumps top consumers.
# Samples: 12345 of event 'cycles'
# Event count (approx.): 9876543210
#
# Children   Self  Command   Shared Object         Symbol
# ........  .....  ......... ..................... ..............
   23.45%   2.10%  postgres  [kernel.kallsyms]     [k] copy_user_enhanced_fast
    8.99%   8.99%  postgres  postgres              [.] heap_getnext
    7.21%   3.10%  postgres  [kernel.kallsyms]     [k] schedule

$ cat /proc/cmdline
BOOT_IMAGE=/boot/vmlinuz-6.6.0 root=UUID=... ro quiet splash isolcpus=2-7 nohz_full=2-7 rcu_nocbs=2-7

$ lsmod | head -10
Module                  Size  Used by
xt_conntrack           16384  4
nft_chain_nat          16384  3
xt_MASQUERADE          16384  2
nf_nat                 57344  2 nft_chain_nat,xt_MASQUERADE
overlay               155648  60
nvme                   65536  4
nvme_core             204800  5 nvme

$ modinfo nvme | head -15
filename:       /lib/modules/6.6.0/kernel/drivers/nvme/host/nvme.ko
version:        1.0
license:        GPL
description:    NVM Express device driver
parm:           use_threaded_interrupts:bool (read-only)
parm:           use_cmb_sqes: use controller's memory buffer for I/O SQes (bool)
parm:           io_timeout: timeout in seconds for I/O (uint)
parm:           max_host_mem_size_mb: Maximum Host Memory Buffer (HMB) size per controller (uint)
parm:           sgl_threshold: Use SGLs when average request segment size is larger or equal to this size. Use 0 to disable SGLs. (uint)

$ cat /sys/module/nvme/parameters/io_timeout
30

$ make defconfig && grep -E 'CONFIG_PREEMPT' .config
CONFIG_PREEMPT_BUILD=y
# CONFIG_PREEMPT_NONE is not set
# CONFIG_PREEMPT_VOLUNTARY is not set
CONFIG_PREEMPT=y
CONFIG_PREEMPT_COUNT=y
CONFIG_PREEMPTION=y
CONFIG_PREEMPT_RCU=y
```

#### Block I/O

```bash
$ cat /sys/block/nvme0n1/queue/scheduler
[none] mq-deadline kyber bfq

$ echo kyber | sudo tee /sys/block/nvme0n1/queue/scheduler
kyber

$ cat /sys/block/nvme0n1/queue/nr_requests
1023

$ sudo iostat -xz 1
Device            r/s     w/s     rkB/s     wkB/s   rrqm/s   wrqm/s  %rrqm  %wrqm r_await w_await aqu-sz rareq-sz wareq-sz   svctm  %util
nvme0n1       1234.00  234.00  98760.00  18720.00     0.00    12.00   0.00   4.88    0.20    0.45   0.34    80.0    80.0    0.10  12.34

$ sudo blktrace -d /dev/nvme0n1 -o trace -w 5 && blkparse -i trace.blktrace.0 | head
# Tracks every block-layer event for 5 seconds; tens of thousands of events
# per second on a busy NVMe.

$ sudo bpftrace -e 'tracepoint:block:block_rq_complete /args->dev/ {
    @ms = hist((nsecs - @start[args->dev, args->sector]) / 1000000);
    delete(@start[args->dev, args->sector]);
}'
# Block I/O latency histogram in milliseconds.

$ ls -l /sys/class/block/nvme0n1/queue/
discard_granularity  hw_sector_size  max_segments     read_ahead_kb     write_cache
discard_max_bytes    iostats         minimum_io_size  rotational        write_zeroes_max_bytes
discard_zeroes_data  io_poll         nomerges         scheduler         zoned
fua                  io_poll_delay   nr_requests      stable_writes
hw_sector_size       max_sectors_kb  optimal_io_size  unpriv_sgio       ...

$ cat /proc/diskstats | head
   8       0 sda 12345 678 1234567 890 9876 543 21098765 432 0 1234 1322
# fields: major minor name reads merged sectors_read read_ms writes write_merged
#         sectors_written write_ms in_flight io_ms weighted_io_ms
#         + 4 more fields for discard/flush since 4.18
```

#### io_uring quick-test

```bash
$ cat > test_uring.c <<'EOF'
#include <liburing.h>
#include <stdio.h>
#include <fcntl.h>
int main() {
    struct io_uring r; io_uring_queue_init(8, &r, 0);
    int fd = open("/etc/os-release", O_RDONLY);
    char buf[256];
    struct io_uring_sqe *s = io_uring_get_sqe(&r);
    io_uring_prep_read(s, fd, buf, 256, 0);
    io_uring_submit(&r);
    struct io_uring_cqe *c;
    io_uring_wait_cqe(&r, &c);
    write(1, buf, c->res);
    io_uring_cqe_seen(&r, c);
}
EOF
$ gcc -O2 -o tu test_uring.c -luring
$ ./tu
NAME="Ubuntu"
VERSION="22.04.3 LTS"
...

$ cat /proc/sys/kernel/io_uring_disabled
0     # 0 = allowed for everyone, 1 = CAP_SYS_ADMIN only, 2 = forbidden
```

#### Tracing extras

```bash
$ sudo perf stat -a -e 'block:*' sleep 5
# Counts every block-layer tracepoint hit across all CPUs for 5 s.

$ sudo trace-cmd record -e sched_switch sleep 1 && sudo trace-cmd report | head
# Captures sched_switch tracepoint to trace.dat, then prints first events.

$ sudo bpftrace -l 'tracepoint:syscalls:sys_enter_*' | head -10
tracepoint:syscalls:sys_enter_accept
tracepoint:syscalls:sys_enter_accept4
tracepoint:syscalls:sys_enter_acct
tracepoint:syscalls:sys_enter_add_key
tracepoint:syscalls:sys_enter_adjtimex
...

$ sudo cat /sys/kernel/debug/tracing/available_events | head
syscalls:sys_enter_accept
syscalls:sys_exit_accept
sched:sched_switch
sched:sched_wakeup
block:block_rq_issue
...

$ sudo cat /sys/kernel/debug/tracing/available_filter_functions | wc -l
50432       # number of kprobe-able functions (very kernel-config dependent)
```

### Debugging and Observability Toolbelt

A tier-by-tier cheat list for answering "what is the kernel doing right now":

#### Always-available (no install)

```
/proc/<pid>/status         # threads, state, signals, capabilities
/proc/<pid>/stack          # current kernel stack of every thread
/proc/<pid>/wchan          # one-line "what is it waiting on"
/proc/<pid>/sched          # CFS/EEVDF stats per task
/proc/<pid>/maps           # virtual address space
/proc/<pid>/smaps          # detailed mappings with RSS, PSS, swap
/proc/<pid>/fd/            # all open fds
/proc/<pid>/fdinfo/        # extra info per fd: position, flags, epoll details
/proc/<pid>/io             # bytes read/written by this task
/proc/<pid>/oom_score      # current OOM score
/proc/<pid>/oom_score_adj  # adjustment knob
/proc/<pid>/limits         # ulimit values
/proc/<pid>/cgroup         # cgroup membership
/proc/<pid>/ns/            # symlinks identifying each namespace
/proc/<pid>/syscall        # currently executing syscall (if any)
```

```
/proc/meminfo              # global memory accounting
/proc/vmstat               # per-event memory counters (pgalloc_*, pgsteal_*, etc)
/proc/cpuinfo              # per-CPU info
/proc/stat                 # cumulative CPU times per CPU + softirq totals
/proc/loadavg              # 1/5/15 min runqueue + total threads
/proc/net/sockstat         # socket counts and memory
/proc/net/protocols        # which network protocols are registered
/proc/net/tcp /proc/net/udp # raw socket lists (use ss instead)
/proc/net/route            # routing table (use ip route)
/proc/diskstats            # cumulative I/O per device (use iostat)
/proc/interrupts           # IRQ counts per CPU per device
/proc/softirqs             # softirq counts per CPU per type
/proc/kallsyms             # all kernel symbols (root for full)
/proc/sysrq-trigger        # echo a letter to invoke a sysrq command
```

#### perf (in kernel-tools or perf-tools-unstable)

```
perf list                              # all events
perf list 'sched:*'                    # filter
perf stat -a -e ... sleep 5            # count events system-wide
perf stat -p PID -- sleep 5            # for one pid
perf record -F 99 -ag sleep 30         # cpu-profile system, call graphs
perf record -e syscalls:sys_enter_* -a sleep 5
perf report --stdio                    # tabular dump
perf top                               # live profile
perf sched record sleep 5; perf sched latency
perf bench sched messaging             # microbenchmarks
perf trace -p PID                      # strace replacement, BPF-based
perf probe --add 'tcp_v4_connect'      # add a kprobe via perf
perf c2c record -ag sleep 30; perf c2c report   # cache contention
perf mem record -ag sleep 30; perf mem report   # memory access sampling
```

#### bpftrace one-liners

```
# Syscall count by process:
bpftrace -e 'tracepoint:raw_syscalls:sys_enter { @[comm] = count(); }'

# Top 20 file opens:
bpftrace -e 'tracepoint:syscalls:sys_enter_openat { @[str(args->filename)] = count(); } interval:s:5 { print(@, 20); clear(@); }'

# Block I/O latency histogram:
bpftrace -e 'kprobe:blk_account_io_start { @start[arg0] = nsecs; }
             kprobe:blk_account_io_done  { @us = hist((nsecs - @start[arg0]) / 1000); delete(@start[arg0]); }'

# Page fault count by process:
bpftrace -e 'software:page-faults:1 { @[comm] = count(); }'

# Slab alloc count by cache:
bpftrace -e 'kprobe:kmem_cache_alloc { @[str(((struct kmem_cache *)arg0)->name)] = count(); }'

# Stack trace on every NETDEV_TX_TIMEOUT:
bpftrace -e 'kprobe:dev_watchdog { @[kstack] = count(); }'

# CPU schedule latency histogram:
bpftrace -e 'tracepoint:sched:sched_wakeup { @qt[args->pid] = nsecs; }
             tracepoint:sched:sched_switch /@qt[args->next_pid]/
                { @us = hist((nsecs - @qt[args->next_pid]) / 1000); delete(@qt[args->next_pid]); }'
```

#### bcc tools (in `bpfcc-tools` / `bcc-tools`)

Pre-built tools you should have memorized:

- `execsnoop` — every exec across the box.
- `opensnoop` — every open(at).
- `tcpconnect`, `tcptracer`, `tcpdrop`, `tcptop`, `tcpretrans` — TCP forensics.
- `biosnoop`, `biolatency`, `bitesize` — block I/O.
- `runqlat`, `runqlen`, `cpudist` — scheduler.
- `memleak` — heap-style accounting tool, attaches to malloc/free or kmalloc.
- `offcputime`, `oncputime`, `wakeuptime` — flame-graph data sources.
- `funccount`, `funcslower`, `funclatency` — adhoc kernel function profilers.
- `argdist`, `trace` — flexible per-event filtering.
- `cachestat`, `cachetop` — page-cache hit/miss.
- `slabratetop` — slab allocations per cache per second.
- `vfscount`, `vfsstat` — VFS layer activity.

#### ftrace (`/sys/kernel/tracing/`)

```
echo function_graph > current_tracer
echo 'tcp_*'        > set_ftrace_filter
echo 1              > tracing_on
cat trace | head -50
echo 0              > tracing_on
echo nop            > current_tracer
```

Available tracers: `function`, `function_graph`, `irqsoff`, `preemptoff`, `wakeup`, `wakeup_rt`, `mmiotrace`, `nop`, `hwlat`, `osnoise`. The `osnoise` tracer is excellent for finding RT-disrupting noise sources on isolated CPUs.

#### crash / kdump (postmortem)

```
crash /usr/lib/debug/lib/modules/$(uname -r)/vmlinux  /var/crash/.../vmcore
crash> bt              # backtrace of panicking task
crash> ps              # all tasks
crash> mount           # mounted filesystems
crash> mod             # loaded modules
crash> log             # the dmesg ring buffer
crash> kmem -s         # slab usage
crash> foreach bt      # backtrace of every task
```

#### Memory leak hunting

- `slabtop` — does any cache grow without bound?
- `cat /proc/meminfo | grep -E '(Slab|SUnreclaim|VmallocUsed|KernelStack)'`
- `kmemleak` (CONFIG_DEBUG_KMEMLEAK=y) — write `scan` to `/sys/kernel/debug/kmemleak`, then `cat` it.
- `bcc/memleak` for userspace and kernel hot paths.
- For DMA leaks: CONFIG_DMA_API_DEBUG=y; `/sys/kernel/debug/dma-api/`.

#### Network forensics

```
ss -tnHo state established | wc -l                   # connection count
ss -tnHo state syn-recv                              # half-open
ss -tunap | head                                     # tcp + udp + processes
tcpdump -i eth0 -nnvv -c 50 'tcp[tcpflags] & tcp-syn != 0'
ip -s -s link show eth0                              # all link counters
ip -s neigh                                          # ARP cache
nstat -az | head                                     # all SNMP counters at once
ethtool -S eth0                                      # NIC counters
ethtool -c eth0                                      # interrupt coalescing
ethtool -g eth0                                      # ring sizes
ethtool -l eth0                                      # channel counts
mtr -i 0.1 -c 100 host                               # path latency
```

#### Userspace stack debugging

- `gdb -p PID` — attach.
- `pstack PID` — print call stacks of every thread (uses gdb internally).
- `strace -f -p PID -e trace=network` — narrow filter.
- `lsof -p PID` / `lsof -i :8080` — fds and listeners.
- `pmap -X PID` — same data as `/proc/<pid>/smaps` in tabular form.
- `valgrind --tool=memcheck` — userspace memory errors.
- `perf record -p PID -g -F 99 sleep 30 && perf report` — flamegraph source.

## Common Confusions
### Common Errors

A small library of the most common error messages a kernel-adjacent engineer sees, with the canonical fix.

#### Build / Module errors

```
ERROR: Module nvidia is not currently loaded
        # modprobe nvidia    -- loads it; if it fails, modinfo nvidia for info

modprobe: FATAL: Module foo not found in directory /lib/modules/6.6.0-1
        # depmod -a          -- regenerate dependency map
        # rebuild against the right kernel headers; check 'uname -r'

insmod: ERROR: could not insert module foo.ko: Invalid module format
        # Built against the wrong kernel. Rebuild with the correct
        #   make -C /lib/modules/$(uname -r)/build M=$PWD modules

insmod: ERROR: could not insert module foo.ko: Required key not available
        # The kernel requires signed modules (CONFIG_MODULE_SIG_FORCE).
        # Sign with the build key, or use mokutil --import to enroll.

insmod: ERROR: could not insert module foo.ko: Operation not permitted
        # Kernel lockdown (Secure Boot) is restricting module loading.
        # cat /sys/kernel/security/lockdown -> [integrity] or [confidentiality]

implicit declaration of function 'kmalloc'
        # Add #include <linux/slab.h>

undefined reference to '__stack_chk_guard'
        # In userspace? Build with -fno-stack-protector or link with -lssp.

WARNING: Kernel-version mismatch detected: built for 6.5, running 6.6
        # Reboot into the matching kernel, or rebuild for the running one.
```

#### Memory errors

```
Out of memory: Killed process 12345 (myapp)
        # OOM killer fired. dmesg | grep -A20 'Out of memory' for details.
        # Look at the process scores; tune oom_score_adj or add memory.

cannot allocate memory (errno 12 / ENOMEM)
        # Per-process or cgroup memory limit; check ulimit -v and
        # /sys/fs/cgroup/<cg>/memory.max

vm.overcommit: 0 / Killed (signal 9 / OOM)
        # vm.overcommit_memory=2 with vm.overcommit_ratio set may have
        # rejected a large mmap. Use overcommit_kbytes or relax to mode 1.

mmap: cannot allocate memory
        # /proc/sys/vm/max_map_count too low; default 65530.
        # Java JVMs often need 262144 or higher.

Killed
        # On its own line, after a syscall? OOM killer or someone sent SIGKILL.
        # dmesg | tail   tells you which.
```

#### I/O errors

```
end_request: I/O error, dev sda, sector 12345
        # SMART/medium error. smartctl -H -a /dev/sda

Buffer I/O error on dev nvme0n1, logical block 0, async page read
        # Lower-level read failure surfacing through the page cache.

XFS (sda1): metadata I/O error in "xfs_buf_ioend" at daddr 0x... len 8 error 117
        # Filesystem corruption. Unmount and run xfs_repair.

EXT4-fs error (device sda1): ext4_lookup:1707: inode #131073: comm ls: deleted inode referenced
        # Filesystem inconsistency. fsck.ext4 -y /dev/sda1 from a rescue env.

blk_update_request: I/O error, dev sda, sector 0 op 0x1:(WRITE) flags 0x800
        # Hardware. Check dmesg for SATA link resets, NVMe controller resets.
```

#### Network errors

```
neighbour table overflow
        # ARP cache full. Tune net.ipv4.neigh.default.gc_thresh1/2/3.

nf_conntrack: table full, dropping packet
        # Bump net.netfilter.nf_conntrack_max; investigate connection leaks.

NETDEV WATCHDOG: eth0: transmit queue 0 timed out
        # Driver TX hang. Reset link: ip link set eth0 down up. Update driver.

eth0: hw csum failure
        # NIC reported bad checksum on RX; if persistent, check NIC firmware.

TCP: out of memory -- consider tuning tcp_mem
        # net.ipv4.tcp_mem too small; defaults are RAM-relative but containers
        # may use the host value with too-small memory.cgroup limit.
```

#### Lock / Sync errors

```
INFO: rcu_sched detected stalls on CPUs/tasks:
       8-...!: (0 ticks this GP) idle=...
        # A CPU stuck not making progress for the RCU grace period.
        # Often a real-time task spinning, or kernel bug. dmesg context.

BUG: scheduling while atomic: foo/12345/0x00000002
        # Code holding a spinlock or in IRQ ctx called something that sleeps.
        # Look at the stacktrace; replace kmalloc(GFP_KERNEL) with GFP_ATOMIC,
        # or move the work to a workqueue.

INFO: task foo:12345 blocked for more than 120 seconds
        # Hung task watchdog. Tunable: kernel.hung_task_timeout_secs.
        # The stacktrace tells you which lock or wait it's stuck on.

BUG: soft lockup - CPU#3 stuck for 23s!
        # CPU not yielding. Real-time loop, broken IRQ handler, livelock.

possible recursive locking detected
        # lockdep: code tried to acquire a lock it already holds. Fix logic
        # or use the *_nested variant if intentional.
```

### Confusion Pairs

- **"Why is my server's free memory so low?"** It's not. Look at `available` in `free -h` and `MemAvailable` in `/proc/meminfo`. The page cache is shown under `buff/cache` and is released on demand. Low `free` + high `available` is the normal healthy state. The mistake is treating `free` as the canonical "how much is left" — it's not, since 2014ish when MemAvailable landed.

- **"My app slowed down right after I freed a file. Why does the page cache stay full?"** Because dropping a file's contents from cache doesn't free pages immediately — it marks them clean and reclaimable. They stay populated until something else needs them. Run `echo 3 | sudo tee /proc/sys/vm/drop_caches` if you want to force-drop for a benchmark (NEVER do this in production under load — you remove every cache hit).

- **"`fsync()` is slow. Disk is fast. What gives?"** `fsync()` forces a writeback queue drain, an FUA (Force Unit Access) to bypass the device cache, and on a journaled FS a journal commit. Three round trips minimum. If you're calling it every write, batch with `O_SYNC` or use `fdatasync()` when you don't care about metadata.

- **"My app runs faster on a cold CPU"** — cold CPU = turbo headroom (it can clock higher when nearby cores are idle), AND the L1/L2 may be empty of unrelated junk. Run on a warm-up loop AND consider NUMA-local allocations; a fresh process inherits no NUMA preference and may straddle nodes.

- **"`perf top` shows `schedule()` at 30% of CPU. Is the scheduler the bottleneck?"** No. With kernel-only sampling on an idle box, you sample the idle thread inside `cpu_idle_loop` -> `schedule()`. The bar represents your idle time. Fix by sampling user+kernel (`-e cycles`) or filtering the idle pid (`--exclude-pid 0`).

- **"What's the difference between RCU and seqlock?"** RCU lets readers see *a* consistent old version while a writer publishes a new one — readers never retry. seqlock makes readers retry if a writer touched the data during their read. Pick RCU for read-mostly structures with infrequent updates and high reader concurrency. Pick seqlock for cheap snapshots of small structs (e.g. timekeeping) when you can tolerate read retry under write pressure.

- **"Why does `ksoftirqd/N` burn CPU?"** Because softirqs have backed up. Most often: a NIC is firing more interrupts than the IRQ-handler CPU can drain in line, so NAPI scheduling work piles into the softirq context and `ksoftirqd` is woken up to process it. Check `/proc/net/softnet_stat` (squeezed > 0), pin IRQs across CPUs with `irqbalance` or manually via `/proc/irq/N/smp_affinity`, and make sure RPS is enabled if you only have one RX queue.

- **"Why is `io_uring` faster than `aio(7)`?"** Three reasons. (a) Submission and completion are shared-memory rings — no syscall per op. (b) With SQ polling thread it's literally zero syscall. (c) Far more op codes covered (`aio` is read/write only; `io_uring` does accept, recv, statx, openat, fallocate, …) so you don't fall back to threadpool emulation.

- **"Why does `numa_balancing` cause page faults in my workload?"** Because that's how it works — it strips PTE present bits to provoke faults so it can record the accessing node. The cost is bounded but real for fault-heavy workloads. Disable with `echo 0 | sudo tee /proc/sys/kernel/numa_balancing` if you're already pinning with `numactl`.

- **"My module loads but no `printk` shows up."** The default loglevel filters them. Check with `dmesg -wH` (`H` = human time) or set `loglevel=8` on the kernel command line, or runtime: `echo 8 > /proc/sys/kernel/printk`.

- **"`spin_lock` deadlocked when an interrupt fired."** The interrupt handler tried to grab the same lock; the holder was a normal task. Use `spin_lock_irqsave(&l, flags)` to disable IRQs while held.

- **"Build error: implicit declaration of function `kmalloc`."** You forgot `#include <linux/slab.h>`. The kernel doesn't have a single `linux.h` — every API has a specific header.

- **"`make modules_install` says module 'foo' has no symvers."** You built without CONFIG_MODVERSIONS. Re-run `make modules_prepare` first or accept that the module won't be ABI-versioned across kernels.

- **"`bpftrace` says: ERROR: failed to attach kprobe."** Either the function got inlined and lost its symbol, the kernel is built without `CONFIG_KPROBES`, or you're missing `BPF_LSM`/`CAP_BPF`. `cat /proc/kallsyms | grep <fn>` to confirm the symbol exists.

- **"`io_uring_setup`: EPERM."** Since 6.6, `kernel.io_uring_disabled` is a sysctl. Set to 0 (everyone), 1 (only with `CAP_SYS_ADMIN`), or 2 (off).

- **"`SCHED_FIFO` task hung the box."** It pinned a CPU at 100% and the watchdog tripped. The kernel has a safety net (`/proc/sys/kernel/sched_rt_runtime_us`, default 950000 of 1000000 µs) reserving 5% for non-RT. If you set `sched_rt_runtime_us` to -1 (no throttling) and your RT task busy-loops, only a hard-reset will save you.

- **"My huge page allocation failed during runtime even though `nr_hugepages` is 64."** Hugepages need physically contiguous order-9 (2 MiB) blocks. Long-running boxes fragment. Pre-reserve at boot via `default_hugepagesz=2M hugepagesz=2M hugepages=N` on the kernel command line, OR run `echo 1 > /proc/sys/vm/compact_memory` to force compaction.

- **"`iostat` shows `%util` of 100% but my SSD is supposedly idle."** `%util` is wallclock-busy at the legacy SCSI layer. On NVMe with deep queues, `%util` saturates at one outstanding request and is meaningless above ~10% queue depth. Look at `aqu-sz` (average queue size) and latency instead.

- **"Why does the kernel use the `container_of` macro?"** Embedded structs. A linked-list element is itself part of a larger struct; given a pointer to the list node you compute the outer struct's address by subtracting the offset of the node. It's the kernel's way of getting OO-like polymorphism without virtual tables. `container_of(ptr, type, member)`.

- **"`vm.swappiness=0` should disable swap, right?"** Not quite. Since 3.5, `swappiness=0` means "swap only if avoiding OOM"; it does *not* mean "never swap." If you genuinely need no swap, `swapoff -a` and don't put a swap entry in fstab. `swappiness=1` is sometimes recommended for boxes that "should not swap unless emergency."

- **"Why does my `mmap()` succeed but the first write segfaults?"** mmap allocated virtual address space, not physical memory. The first access page-faults; if the kernel can't allocate a page (cgroup limit, no swap, overcommit refusal), the fault delivers SIGSEGV. Pre-fault with `MAP_POPULATE` to fail at mmap-time instead.

- **"`perf record` shows huge skid — samples land on the wrong instruction."** Modern x86 has PEBS (Precise Event-Based Sampling); use `-e cycles:p` or `:pp` for precise. Without it, the sampled IP can be tens of instructions past the actual event.

- **"`CONFIG_PREEMPT_RT` and `PREEMPT_DYNAMIC` are different things, right?"** Yes. `CONFIG_PREEMPT_RT` (the realtime patchset, fully merged in 6.12) makes most kernel critical sections preemptible — spinlocks become sleeping. `CONFIG_PREEMPT_DYNAMIC` (since 5.12) lets you switch between `none / voluntary / full` at boot via `preempt=` kernel param without recompiling.

- **"Why is `vsyscall` mentioned but `vDSO` does the work?"** `vsyscall` is the legacy 4 KiB page mapped at a fixed address with a few hot syscalls (`gettimeofday`, etc.). It's mostly disabled today (boot param `vsyscall=none`). The modern equivalent is the **vDSO** — a small ELF mapped into every process exposing `__vdso_clock_gettime` and friends; same idea but ASLR-friendly and extensible.

- **"`ftrace` is too noisy."** Use the function filter: `echo 'tcp_*' > /sys/kernel/tracing/set_ftrace_filter`. Or use the `function_graph` tracer with `set_graph_function`. Or just use `bpftrace` instead — much higher signal-to-noise.

- **"`cgroup` memory.usage_in_bytes drops slowly after I delete files."** Page cache pages charged to the cgroup are released lazily as part of normal reclaim. Force with `echo 1 > memory.force_empty` (cgroup v1) or by triggering reclaim.

- **"`epoll_wait` returns 0 with no events even though I set timeout = -1."** A signal was delivered (and a signal handler was installed). Check `errno == EINTR`. Restart the call.

- **"`bind: address already in use`."** Either another process holds the port (`ss -tnlp | grep :PORT`), or the previous owner is in TIME_WAIT and you didn't set `SO_REUSEADDR`. For TCP servers, set both `SO_REUSEADDR` and `SO_REUSEPORT`.

## Vocabulary

- **buddy allocator** — physical-page allocator using power-of-two block lists.
- **order** — log2 of the page count in a buddy block (order 0 = 1 page = 4 KiB).
- **page frame** — physical 4 KiB chunk; the kernel's unit of physical memory.
- **ZONE_DMA / DMA32 / NORMAL / MOVABLE / DEVICE** — physical address ranges with allocation constraints.
- **fallback order** — NORMAL → DMA32 → DMA on x86_64.
- **watermark (min/low/high)** — free-page thresholds that wake `kswapd` and gate direct reclaim.
- **kswapd** — per-NUMA-node kernel thread that reclaims memory in the background.
- **direct reclaim** — caller-thread reclaim when allocation fails fast path.
- **SLUB** — default slab allocator (since 2.6.23, Oct 2007).
- **SLAB** — older slab allocator, replaced by SLUB.
- **SLOB** — minimalist slab for tiny embedded.
- **slabinfo** — `/proc/slabinfo`, per-cache stats.
- **slabtop** — ncurses live viewer for slab caches.
- **kmem_cache** — a named slab cache (e.g. `task_struct`, `dentry`).
- **page cache** — file-backed page cache shared across processes.
- **buffer head** — legacy 512-byte block descriptor, mostly deprecated.
- **dirty page** — modified-but-not-yet-written page.
- **writeback** — flushing dirty pages to backing store.
- **bdi-flush thread** — per-backing-device flush thread.
- **vm.dirty_ratio / dirty_background_ratio** — sysctl thresholds for throttling/background writeback.
- **vm.swappiness** — anon-vs-file reclaim preference (0-200).
- **vmalloc** — virtually contiguous, physically scattered allocator.
- **kmalloc** — physically contiguous, slab-backed allocator.
- **alloc_pages** — direct buddy-allocator interface.
- **GFP flags** — Get-Free-Pages flags governing allocation context.
- **GFP_KERNEL** — sleeping, default flag.
- **GFP_ATOMIC** — non-sleeping, IRQ-safe.
- **GFP_NOIO / GFP_NOFS** — recursion-safe inside FS/block layer.
- **THP (Transparent Hugepages)** — 2 MiB pages, modes always/madvise/never.
- **khugepaged** — promotes 4 KiB → 2 MiB in background.
- **KPTI (Kernel Page Table Isolation)** — Meltdown mitigation, separate PT for kernel/user.
- **KASLR** — Kernel Address Space Layout Randomization.
- **IOMMU** — IO Memory Management Unit; per-device DMA address translation.
- **DMA** — Direct Memory Access; device reads/writes RAM without CPU.
- **MSI / MSI-X** — Message-Signaled Interrupts (replacement for INTx).
- **NUMA** — Non-Uniform Memory Access (multi-socket).
- **node** — a NUMA domain (typically a socket + its DRAM).
- **distance matrix** — relative latency between NUMA nodes (`numactl -H`).
- **NUMA balancing / AutoNUMA** — automatic page/task migration (since 3.8).
- **numactl** — userspace tool to set NUMA affinity.
- **set_mempolicy** — syscall to set NUMA mempolicy.
- **MPOL_BIND / PREFERRED / INTERLEAVE / LOCAL** — mempolicy modes.
- **NAPI** — interrupt+poll hybrid net-RX API.
- **sk_buff (skb)** — kernel network packet buffer.
- **qdisc** — queueing discipline on a netdev (e.g. `fq_codel`, `pfifo_fast`).
- **tc** — traffic control userspace tool to manipulate qdiscs/filters.
- **XDP** — eXpress Data Path, BPF at driver layer (since 4.8).
- **AF_XDP** — userspace socket bypassing the kernel stack (since 4.18).
- **RPS** — Receive Packet Steering (software RX-queue spread).
- **RFS** — Receive Flow Steering (RPS + flow-aware).
- **RSS** — Receive Side Scaling (hardware RX-queue spread).
- **SO_REUSEPORT** — bind-share across processes for accept-spread.
- **SO_INCOMING_CPU** — socket pin to a CPU.
- **GRO** — Generic Receive Offload (software).
- **GSO** — Generic Segmentation Offload (software).
- **LRO** — Large Receive Offload (hardware; risky for routers).
- **TSO** — TCP Segmentation Offload (hardware).
- **NIC ring** — DMA ring buffer between NIC and host.
- **RX queue / TX queue** — directional ring sets, often per-CPU.
- **MTU** — Maximum Transmission Unit (default 1500, jumbo 9000).
- **jumbo frames** — Ethernet frames > 1500 byte payload.
- **bio** — block I/O descriptor.
- **request_queue** — per-block-device dispatch queue.
- **blk-mq** — Multi-Queue block layer (default since 3.13).
- **mq-deadline** — read-priority deadline scheduler.
- **kyber** — latency-target I/O scheduler.
- **bfq** — Budget Fair Queueing scheduler.
- **none** — null I/O scheduler (raw FIFO, default for fast NVMe).
- **io_uring** — shared-memory I/O ring API (since 5.1, Mar 2019).
- **SQE** — Submission Queue Entry (io_uring).
- **CQE** — Completion Queue Entry (io_uring).
- **IORING_OP_*** — opcode set (READ, WRITE, RECV, ACCEPT, ...).
- **IOSQE_FIXED_FILE** — flag: use registered fd table.
- **IOSQE_IO_LINK** — flag: chain SQEs.
- **registered fds / buffers** — pre-cached resources (`io_uring_register`).
- **splice** — pipe-to-pipe in-kernel transfer syscall.
- **sendfile** — file-to-socket in-kernel copy.
- **copy_file_range** — file-to-file in-kernel copy (may use reflinks).
- **mmap** — map fd into process VAS.
- **MAP_POPULATE** — pre-fault pages on mmap.
- **mlock** — pin pages, prevent swap.
- **fadvise / posix_fadvise** — pattern hints to page cache.
- **RCU** — Read-Copy-Update lockless reader synchronization.
- **grace period** — interval where all CPUs have passed through a quiescent state.
- **synchronize_rcu** — block until grace period ends.
- **call_rcu** — schedule callback after next grace period.
- **rcu_read_lock / unlock** — read-side critical section markers.
- **lockdep** — runtime lock-ordering validator.
- **spinlock** — busy-wait mutex; never sleeps.
- **rwlock** — many readers OR one writer spinlock.
- **seqlock** — writer-bumps-seq, readers retry on mismatch.
- **mutex** — sleeping single-holder lock.
- **semaphore** — sleeping counted lock.
- **completion** — one-shot wait/wake primitive.
- **atomic_t** — atomic int wrapper.
- **cmpxchg** — compare-and-swap atomic op.
- **smp_mb / rmb / wmb** — memory barriers.
- **READ_ONCE / WRITE_ONCE** — compiler-fence single-access macros.
- **percpu** — per-CPU storage (`DEFINE_PER_CPU`).
- **sched class** — scheduler class (stop, deadline, RT, fair, idle).
- **CFS** — Completely Fair Scheduler (default 2.6.23 - 6.5).
- **vruntime** — virtual runtime, the CFS rb-tree key.
- **min_vruntime** — runqueue floor for new tasks.
- **sched_entity** — schedulable unit (task or task group).
- **runqueue (rq, cfs_rq, rt_rq, dl_rq)** — per-CPU per-class queue.
- **sched domain** — CPU-topology balancing scope (SMT/MC/DIE/NUMA).
- **SCHED_FIFO** — RT FIFO class.
- **SCHED_RR** — RT round-robin class.
- **SCHED_DEADLINE** — EDF + CBS class.
- **SCHED_OTHER** — default CFS class.
- **SCHED_BATCH** — CFS variant for batch jobs.
- **SCHED_IDLE** — below nice 19.
- **EEVDF** — Earliest Eligible Virtual Deadline First (replacing CFS since 6.6).
- **EAS** — Energy Aware Scheduling (ARM big.LITTLE).
- **sched_features** — `/sys/kernel/debug/sched_features` toggles.
- **isolcpus** — boot param to keep CPUs out of the general scheduler.
- **nohz_full** — boot param for full dynticks (no scheduler tick).
- **rcu_nocbs** — boot param to offload RCU callbacks.
- **defconfig** — `make` target: arch's default config.
- **menuconfig** — ncurses Kconfig editor.
- **olddefconfig** — preserve existing answers, default new symbols.
- **localmodconfig** — config from currently loaded modules.
- **Kconfig** — the config language (`config`, `menu`, `choice`, `bool`, `tristate`, `depends on`, `select`, `imply`, `default`).
- **vmlinux** — uncompressed ELF kernel.
- **bzImage** — compressed bootable image (x86).
- **zImage** — older small compressed image.
- **initramfs** — initial RAM filesystem (early userspace).
- **makefile** — kbuild Makefile fragment per dir.
- **kbuild** — the kernel's recursive Make build system.
- **dkms** — Dynamic Kernel Module Support (out-of-tree module rebuilds across kernels).
- **modules_install** — `make` target copying `*.ko` to `/lib/modules/$(uname -r)/`.
- **modprobe / insmod / rmmod / lsmod / modinfo** — module userspace tools.
- **System.map** — symbol-to-address dump for the build.
- **kallsyms** — runtime symbol table (`/proc/kallsyms`).
- **lockdep / KASAN / KMSAN / UBSAN / KCSAN** — kernel debug infrastructure.
- **ftrace** — kernel function tracer (`/sys/kernel/tracing/`).
- **trace-cmd / kernelshark** — userspace ftrace frontends.
- **perf** — sampling/counting profiler.
- **bpftrace** — BPF tracing language.
- **bpf()** — syscall to load and attach BPF programs.
- **CO-RE** — Compile Once, Run Everywhere BPF mechanism.
- **BTF** — BPF Type Format, kernel debug-info for BPF.
- **eBPF maps** — BPF-side data structures (hash, array, ringbuf, etc.).
- **kprobe / kretprobe / uprobe / fentry / fexit** — kernel/user dynamic probes.
- **tracepoint** — static trace markers in the kernel source.
- **softirq** — deferred work category (NET_RX, NET_TX, TIMER, BLOCK, RCU, ...).
- **ksoftirqd/N** — per-CPU softirq kernel thread.
- **tasklet** — older softirq-equivalent, mostly being replaced by workqueues.
- **workqueue** — kernel deferred-work in process context.
- **kworker/N** — workqueue worker threads.
- **kthread** — kernel-thread base abstraction.
- **container_of** — macro to recover outer-struct pointer from member pointer.
- **list_head** — kernel doubly-linked list head/node embedded in struct.
- **rb_node** — red-black tree node embedded in struct.
- **xarray / radix tree / maple tree** — kernel sparse-array containers.
- **idr / ida** — id allocator (sparse int → ptr maps).
- **lockdep** — runtime lock-ordering checker.
- **KASAN** — Kernel Address Sanitizer; UAF/OOB detector.
- **KMSAN** — Kernel Memory Sanitizer; uninit-memory detector.
- **UBSAN** — Undefined Behavior Sanitizer.
- **KCSAN** — Kernel Concurrency Sanitizer; data-race detector.
- **kfence** — sample-based UAF detector for production.
- **BUG_ON / WARN_ON / WARN_ONCE** — assertions; BUG kills, WARN logs.
- **panic_on_oops** — sysctl to escalate any oops to panic.
- **kdump** — crash-dump-via-kexec mechanism.
- **kexec** — boot a new kernel from inside a running one.
- **vmcore** — the dump file produced by kdump.
- **crash** — analyzer for vmcore.
- **dmesg ring buffer** — kernel printk output.
- **printk loglevel** — KERN_EMERG .. KERN_DEBUG, severity 0..7.
- **systemd-journald** — captures dmesg via /dev/kmsg as well as syslog.
- **rcuop / rcuog / rcuc** — RCU offload kernel threads.
- **migration/N** — per-CPU thread that runs the stop-class for migrating tasks.
- **kcompactd** — per-NUMA compaction kernel thread.
- **khugepaged** — THP promotion kernel thread.
- **kdevtmpfs** — populates `/dev` from devtmpfs.
- **udev** — userspace device manager (systemd-udevd today).
- **devtmpfs** — kernel-managed `/dev`.
- **devpts** — pseudo-tty filesystem at `/dev/pts`.
- **debugfs** — debug-only fs at `/sys/kernel/debug`.
- **tracefs** — ftrace's fs (was under debugfs, now `/sys/kernel/tracing`).
- **bpffs** — pinned BPF objects fs at `/sys/fs/bpf`.
- **cgroupfs** — cgroup v1/v2 fs at `/sys/fs/cgroup`.
- **selinuxfs** — `/sys/fs/selinux`.
- **securityfs** — `/sys/kernel/security`.
- **fanotify** — content+access notification (anti-virus, hierarchical watching).
- **inotify** — file watch API (per-mount).
- **poll/select/epoll/io_uring** — readiness/completion families.
- **eventfd / timerfd / signalfd** — file-descriptor wrappers for events/timers/signals.
- **memfd_create** — anonymous memory-backed fd; useful for shared mem and seccomp filters.
- **userfaultfd** — userspace handles page faults (CRIU, qemu live-migration).
- **process_vm_readv/writev** — cross-process memory access syscalls.
- **prctl** — per-process behavior knob (NO_NEW_PRIVS, dumpable, child-subreaper, ...).
- **ptrace** — debugger primitive; underlies gdb, strace.
- **seccomp** — system-call filter (BPF-based since 3.5).
- **landlock** — userspace-driven sandboxing LSM (since 5.13).
- **CRIU** — Checkpoint/Restore In Userspace, dump+restore process trees.
- **uffd** — userfaultfd; CRIU and qemu-kvm post-copy.
- **vDSO** — virtual Dynamic Shared Object; small ELF in every process for fast pseudo-syscalls.
- **vsyscall** — legacy fixed-address pseudo-syscall page (mostly disabled).
- **VDSO clock_gettime** — accelerated clock read; ~10ns vs ~50ns syscall.
- **smaps_rollup** — process-wide PSS/RSS aggregation.
- **PSI (Pressure Stall Information)** — cgroup-aware pressure metrics: cpu/memory/io.
- **/proc/pressure/{cpu,memory,io}** — system-wide PSI files.
- **cgroup v2 controllers** — cpu, memory, io, pids, cpuset, hugetlb, rdma, misc, ...
- **memory.high / memory.max** — soft and hard cgroup memory limits.
- **memory.events** — counts of low/high/max/oom hits.
- **io.weight / io.max / io.cost.qos** — I/O cgroup tunables.
- **PSI memory.full / memory.some** — pressure flavors.

## Try This

1. **Bench numactl-bound vs unbound**: `numactl --membind=0 --cpunodebind=0 stream` versus `numactl --interleave=all stream`. Compare. On a 2-socket box with 30% NUMA distance you'll see ~25% throughput delta on STREAM Triad. Then `cat /sys/devices/system/node/node*/numastat` before and after to confirm `numa_miss` did or did not grow.

2. **Watch page-cache eviction with pcstat**: `go install github.com/tobert/pcstat/cmd/pcstat@latest`. Then `pcstat /var/log/syslog` to see per-page residency. `dd if=/dev/zero of=fill bs=1M count=$(($(free -m | awk '/Mem:/{print $7}') / 2))` — fill half of available — and re-run pcstat to see the original log's page-cache residency drop.

3. **Watch a slab grow under load**: `sudo slabtop -o -s c | head` in one terminal. In another, `for i in $(seq 1 100000); do touch /tmp/test_$i; done; rm /tmp/test_*`. Observe `dentry` and `inode_cache` swell and then stabilize.

4. **io_uring fio**: `fio --name=test --ioengine=io_uring --rw=randread --bs=4k --size=1G --filename=/data/file --iodepth=128 --runtime=10`. Compare to `--ioengine=libaio` and `--ioengine=psync`. On NVMe expect 3-10x IOPS for io_uring vs psync.

5. **bpftrace one-liner**: `sudo bpftrace -e 'tracepoint:syscalls:sys_enter_openat { @[comm] = count(); }'`. Watch open() rate per process for ~30s, Ctrl-C, read the histogram. Try the `vfs_read` variant from Hands-On.

6. **Rebuild a module**: `cd /lib/modules/$(uname -r)/build`, copy a module's source under there, edit one `pr_info` line, `make M=path/to/dir`, `sudo insmod path/to/your.ko`, `dmesg -t | tail -5`.

7. **Compaction**: `cat /proc/buddyinfo` (note distribution), then `echo 1 | sudo tee /proc/sys/vm/compact_memory`, then `cat /proc/buddyinfo` again — high-order blocks should grow.

8. **NUMA balancing in action**: pin a malloc-hammering workload to node 1 with `numactl --cpunodebind=1` *but* allocate the buffer from node 0 with `numactl --membind=0` (use a 2-process trick or `mbind(2)`). Watch `numastat` over 60 s with autobalance on; pages migrate. Then `echo 0 > /proc/sys/kernel/numa_balancing` and rerun — pages stay put.

9. **Drop a packet with XDP**: install `xdp-tools`. `sudo xdp-loader load -m skb eth0 xdp_pass.o` then `sudo xdp-filter load eth0 && sudo xdp-filter ip 10.0.0.99 -m drop`. Verify with `tcpdump`.

10. **Bisect a regression**: with two known-good and known-bad kernel versions in your tree, run `git bisect start && git bisect bad <bad> && git bisect good <good>` and walk through 4-5 builds. Even if you don't reach a commit, you'll have built five kernels and learned the modules_install + grub-mkconfig dance cold.

11. **Lockdep walk-through**: enable `CONFIG_PROVE_LOCKING=y CONFIG_DEBUG_LOCK_ALLOC=y CONFIG_LOCKDEP=y` in a kernel build, boot it, run a stress test (`stress-ng --hdd 4 --io 4 --vm 2 --timeout 60s`). Check `/proc/lockdep_stats`. If lockdep flags an issue it dumps the dependency chain in `dmesg` — read one such report end-to-end.

12. **Watch a hung task**: `sudo bash -c 'cat > /tmp/wedge.sh' <<'EOF'
#!/bin/bash
for i in 1 2 3 4 5; do dd if=/dev/zero of=/tmp/big_$i bs=1M count=1024 oflag=direct &; done
wait
EOF`. Run `chmod +x /tmp/wedge.sh && sudo /tmp/wedge.sh`. In another shell `dmesg -w` and watch for any "task ... blocked for more than 120 seconds" if your storage is slow.

13. **Build a custom kernel with one CONFIG flipped**: `make defconfig`, then `scripts/config --enable CONFIG_DEBUG_INFO_DWARF5 --disable CONFIG_RANDOM_TRUST_CPU`, `make olddefconfig`, `make -j$(nproc) bzImage`. Inspect the result with `file vmlinux`, `size vmlinux`, `objdump -h vmlinux | head`.

14. **Trace a syscall path** with `ftrace`: `echo function_graph > /sys/kernel/tracing/current_tracer; echo do_sys_openat2 > /sys/kernel/tracing/set_graph_function; echo 1 > /sys/kernel/tracing/tracing_on; cat /etc/hostname; echo 0 > /sys/kernel/tracing/tracing_on; cat /sys/kernel/tracing/trace | head -100`. Witness the recursive descent through VFS.

15. **Measure the cost of a syscall**: write a tight loop that calls `getpid()` 10^7 times, time it; same with `gettimeofday()` (vDSO so far cheaper); same with `clock_gettime(CLOCK_REALTIME)`. The ratio of `getpid` (real syscall) to `gettimeofday` (vDSO) shows the IRET/SYSRET cost.

### Production Tuning Recipes

A small cookbook. None of these is universal — measure first.

#### Database server (Postgres / MySQL / MongoDB)

```
# Memory / dirty page thresholds — flush small/often instead of large/seldom.
vm.dirty_background_bytes = 268435456    # 256 MiB
vm.dirty_bytes            = 536870912    # 512 MiB
vm.swappiness             = 1
vm.vfs_cache_pressure     = 50

# Hugepages
/sys/kernel/mm/transparent_hugepage/enabled = madvise   # never for some

# NUMA — DBs benefit from --interleave or being pinned per-instance
numactl --interleave=all postgres ...

# Filesystem (XFS recommended for Postgres)
mount -o noatime,inode64,logbsize=256k /dev/sdb1 /var/lib/postgresql

# I/O scheduler
echo none > /sys/block/nvme0n1/queue/scheduler

# kernel.shmmax / kernel.shmall (legacy SysV; mostly auto-tuned now)
```

#### High-throughput web server / proxy (nginx / haproxy / envoy)

```
# Network stack
net.core.somaxconn               = 65535
net.core.netdev_max_backlog      = 10000
net.ipv4.tcp_max_syn_backlog     = 65535
net.ipv4.tcp_tw_reuse            = 1
net.ipv4.tcp_fin_timeout         = 30
net.ipv4.tcp_keepalive_time      = 300
net.ipv4.ip_local_port_range     = "1024 65535"
net.core.default_qdisc           = fq
net.ipv4.tcp_congestion_control  = bbr
net.ipv4.tcp_notsent_lowat       = 131072

# fd limits (per-process)
ulimit -n 1048576

# Cgroup memory limit; do NOT rely on host vm.swappiness inside containers.

# RPS / RFS for many small flows
echo ffff > /sys/class/net/eth0/queues/rx-0/rps_cpus
echo 32768 > /proc/sys/net/core/rps_sock_flow_entries
echo 4096  > /sys/class/net/eth0/queues/rx-0/rps_flow_cnt
```

#### Latency-sensitive HFT / real-time

```
# Boot params:  isolcpus=2-7 nohz_full=2-7 rcu_nocbs=2-7 mitigations=off
# Pin app: taskset -c 2-7 ./app   (and SCHED_FIFO via chrt -f 50)
# Disable CPU frequency scaling: cpupower frequency-set -g performance
# Disable C-states: cpupower idle-set -d 1   (or use cpu_dma_latency hint)
# IRQ affinity: keep interrupts off the hot CPUs.
echo 3 > /proc/irq/default_smp_affinity   # default IRQs to CPU 0,1
# RT throttling off (CAREFUL — can wedge box if RT loops):
echo -1 > /proc/sys/kernel/sched_rt_runtime_us
```

#### Container host (Kubernetes node)

```
# Required for Kubernetes:
net.ipv4.ip_forward            = 1
net.bridge.bridge-nf-call-iptables = 1
net.netfilter.nf_conntrack_max = 524288
fs.inotify.max_user_watches    = 1048576
fs.inotify.max_user_instances  = 8192
vm.max_map_count               = 262144
kernel.pid_max                 = 4194304
fs.file-max                    = 9223372036854775807

# Cgroup v2 unified for kubelet:  systemd.unified_cgroup_hierarchy=1
# Block I/O: leave 'none' for NVMe.
```

#### Memory-constrained edge / IoT

```
# Tinier per-thread stack:
ulimit -s 256

# Aggressive reclaim:
vm.swappiness                 = 100
vm.vfs_cache_pressure         = 200

# Compress swap with zswap or zram:
echo 1 > /sys/module/zswap/parameters/enabled
echo zstd > /sys/module/zswap/parameters/compressor

# Drop the page cache after big batch jobs:
echo 1 > /proc/sys/vm/drop_caches    # only in maintenance windows
```

## Where to Go Next

End of the curriculum tier. Next:

- `cs fundamentals linux-kernel-internals` — dense reference, the daily driver from here on.
- `cs detail fundamentals/linux-kernel-internals` — academic underpinning of the same material.
- `cs kernel-tuning sysctl` — applied sysctl tour with production examples.
- `cs kernel-tuning cgroups` — cgroups v2 unified hierarchy controllers.
- `cs kernel-tuning namespaces` — every namespace, deeply.
- `cs kernel-tuning ebpf` — eBPF for tracing and observability.
- `cs kernel-tuning memory-tuning` — vm.* in production.
- `cs kernel-tuning network-stack-tuning` — net.core / net.ipv4 / qdiscs.
- `cs performance perf` — perf top/record/script.
- `cs performance bpftrace` — bpftrace cookbook.
- `cs performance ebpf` — bcc and libbpf.
- Read kernel/Documentation/ in a kernel tree: start with `admin-guide/`, `core-api/`, `scheduler/`, `vm/`, `networking/`, `block/`.
- Subscribe to LWN.net's weekly kernel page.

### Concretely, a Path Through `cs`

If you want to step from this sheet to mastery, here is the recommended trail of `cs` jumps:

1. `cs fundamentals linux-kernel-internals` — pin this open in another terminal and treat it as your reference manual.
2. `cs kernel-tuning sysctl` for the day-1 sysctl walk-through. Read every section even if you don't apply it; you are calibrating your sense of which knob lives where.
3. `cs kernel-tuning memory-tuning` and `cs kernel-tuning network-stack-tuning` together — they share a vocabulary.
4. `cs kernel-tuning cgroups` and `cs kernel-tuning namespaces` for the container side.
5. `cs kernel-tuning ebpf` to round out kernel-attached programs.
6. `cs performance perf` and `cs performance bpftrace` — these are the "how do you find out" sheets that pair with this one's "what is happening."
7. `cs performance ebpf` for libbpf-style program development.
8. `cs networking tcp` for TCP state machine reference and tuning.
9. `cs filesystems ext4` and `cs filesystems xfs` for FS-specific layout, mount options, and recovery.
10. `cs system perf`, `cs system strace`, `cs system gdb` for the user-side toolchain.
11. `cs fundamentals ebpf-bytecode` if you ever want to read raw BPF bytecode.

After that, the kernel source tree is genuinely browsable. Start with `mm/page_alloc.c`, `kernel/sched/core.c`, `net/core/dev.c`, `block/blk-mq.c`, and `fs/read_write.c` — they are the central files of their subsystems. Use `make cscope` or `ctags` once and the navigation cost drops dramatically.

### What You Should Now Be Able to Do

By the end of this sheet you should be able to, without leaving the terminal:

- Diagnose an OOM by reading the dmesg report and pointing to the offending cgroup, the score, and the policy fix.
- Switch I/O schedulers and explain why one would choose each.
- Tune `vm.dirty_*` for a write-heavy database workload and predict the latency impact.
- Read `/proc/buddyinfo` and `/proc/slabinfo` and reason about memory fragmentation vs. slab leaks.
- Pin a process to a NUMA node, prove it stays pinned, and measure the bandwidth delta.
- Trace a syscall path with `bpftrace` or `ftrace` end-to-end.
- Build a custom kernel with one CONFIG flipped, verify it boots, and bisect a regression.
- Explain RCU grace periods to a coworker without checking notes.
- Write a one-line `bpftrace` for "block I/O latency by device" or "page fault rate by process."
- Pick the right offload knob (`ethtool -K`) for a given workload.
- Identify whether a problem is page-cache, dirty-writeback, scheduler latency, NUMA imbalance, lock contention, or driver IRQ saturation, and which `/proc` / `/sys` file to consult to confirm.

If any of those still feel uncertain, that section above is where to spend more time before moving on.

### A Final Note on Kernel Source Reading

The most common stumbling block when reading kernel source is the indirection through operations vectors. `inode->i_op->lookup(...)` does not have one definition — it has dozens, one per filesystem, glued in by `static const struct inode_operations ext4_dir_inode_operations`. To navigate this:

- Use `cscope` (`make cscope`) and `Ctrl-]` jumps in vim, or your editor's LSP. `git grep -n 'i_op = &'` locates every assignment of an inode_operations.
- When you hit a function pointer, search for its assignment, then for the name of the assigned struct — that gives you "for FS X, this is what runs."
- Many subsystems have a "core" + "drivers/protocols" split: `mm/` is the core, `drivers/` are the wires; `net/core/` is the core, `net/ipv4/` and `net/ipv6/` and `net/sched/` are families; `fs/` has core (super.c, namei.c, dcache.c) and per-FS subdirs.
- Function suffixes are conventions: `_unlocked` means caller already holds a lock; `_locked` means we take it; `__name` (double underscore) means a low-level helper; `name` is the public API; `_rcu` means inside an RCU read-side; `_safe` means iteration-safe (iterators that handle deletion).
- Macros like `for_each_*`, `list_for_each_entry`, `rcu_dereference`, `WARN_ON`, and `BUG_ON` are everywhere. Learn the dozen most common from `include/linux/list.h`, `include/linux/rculist.h`, `include/linux/bug.h`.
- Many headers in `include/linux/` are the canonical API surface. `include/linux/fs.h`, `include/linux/sched.h`, `include/linux/mm.h`, `include/net/sock.h`, `include/linux/blkdev.h` — be willing to skim 3000-line headers; they document themselves.

Reading kernel code becomes natural after the first dozen serious sessions. The investment pays back forever.

## See Also

- `ramp-up/linux-kernel-high-school`
- `fundamentals/linux-kernel-internals`
- `fundamentals/ebpf-bytecode`
- `kernel-tuning/sysctl`
- `kernel-tuning/cgroups`
- `kernel-tuning/namespaces`
- `kernel-tuning/ebpf`
- `kernel-tuning/memory-tuning`
- `kernel-tuning/network-stack-tuning`
- `system/perf`
- `system/strace`
- `system/gdb`
- `performance/perf`
- `performance/bpftrace`
- `performance/ebpf`
- `networking/tcp`
- `filesystems/ext4`
- `filesystems/xfs`

## References

- kernel.org Documentation/ — `admin-guide/`, `core-api/`, `scheduler/`, `vm/`, `networking/`, `block/`.
- "Understanding the Linux Kernel" — Bovet & Cesati (3rd ed.).
- "Linux Kernel Development" — Robert Love (3rd ed.).
- "Linux Device Drivers" — Corbet, Rubini, Kroah-Hartman (LDD3).
- "BPF Performance Tools" — Brendan Gregg.
- "Systems Performance" — Brendan Gregg (2nd ed.).
- "What Every Programmer Should Know About Memory" — Ulrich Drepper (LWN series, 2007).
- LWN.net — kernel article archive (`https://lwn.net/Kernel/Index/`).
- man 2 io_uring_setup, io_uring_enter, io_uring_register.
- man 7 cpuset, mq_overview, sched, rcu.
- man 8 tc, ip-link, ethtool.
- `Documentation/scheduler/sched-design-CFS.rst` (and the EEVDF design notes added in 6.6).
- `Documentation/RCU/whatisRCU.rst`.
- `Documentation/admin-guide/mm/transhuge.rst`.
- `Documentation/networking/scaling.rst`.
- `Documentation/block/blk-mq.rst`.
- `Documentation/admin-guide/sysctl/vm.rst`.
- `Documentation/admin-guide/kernel-parameters.txt`.
- `Documentation/admin-guide/cgroup-v2.rst`.
- `Documentation/filesystems/ext4/`, `xfs/`, `btrfs/`.
- `Documentation/locking/`.
- `Documentation/io_uring/` and `tools/io_uring/` examples.
- `Documentation/admin-guide/pm/` — power management.
- `Documentation/dev-tools/kasan.rst`, `kmemleak.rst`, `kfence.rst`, `kcsan.rst`.
- `Documentation/scheduler/sched-eevdf.rst` (added in 6.6 with the new scheduler).
- `Documentation/admin-guide/mm/multigen_lru.rst` (MGLRU since 6.1).
- `Documentation/networking/af_xdp.rst`.
- `Documentation/bpf/` — every BPF subdocument.
- "Linux Kernel Programming" — Kaiwan Billimoria.
- "Mastering Linux Kernel Development" — Raghu Bharadwaj.
- Brendan Gregg's blog (`https://www.brendangregg.com/`).

