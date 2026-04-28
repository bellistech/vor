# Linux Kernel — Deep Dive

> *The kernel is a real-time decision engine wrapped around hardware abstraction. Every context switch, every page fault, every TLB miss, every interrupt is governed by precise formulas balancing fairness, throughput, latency, and survival. This is the math.*

---

## 1. CFS Scheduler Mathematics

### 1a. The Core Idea — vruntime

The Completely Fair Scheduler (CFS, kernel 2.6.23+) is a **proportional-share** scheduler. Each runnable task accumulates "virtual runtime" weighted by its priority. The scheduler always picks the task with the **smallest vruntime**.

### The vruntime Update Formula

When a task runs for actual delta `delta_exec` nanoseconds:

```
vruntime += delta_exec * NICE_0_LOAD / load_weight(prio)
```

Where:
- `delta_exec` = actual CPU time consumed (nanoseconds)
- `NICE_0_LOAD` = 1024 (the "unit" weight; nice 0 task weight)
- `load_weight(prio)` = sched_prio_to_weight[nice + 20]

For nice 0: weight = 1024, so `vruntime += delta_exec` (1:1 mapping).
For nice -20 (highest prio): weight = 88761, so vruntime grows ~87× more slowly per nanosecond of CPU.
For nice +19 (lowest prio): weight = 15, so vruntime grows ~68× faster.

### The load_weight Table

From `kernel/sched/core.c`:

```c
const int sched_prio_to_weight[40] = {
 /* -20 */     88761,     71755,     56483,     46273,     36291,
 /* -15 */     29154,     23254,     18705,     14949,     11916,
 /* -10 */      9548,      7620,      6100,      4904,      3906,
 /*  -5 */      3121,      2501,      1991,      1586,      1277,
 /*   0 */      1024,       820,       655,       526,       423,
 /*   5 */       335,       272,       215,       172,       137,
 /*  10 */       110,        87,        70,        56,        45,
 /*  15 */        36,        29,        23,        18,        15,
};
```

### Geometric Ratio

Each nice level differs by factor ~1.25:

```
weight(n) ≈ 1024 / 1.25^n
```

Check: `1024 / 1.25^1 = 819.2 ≈ 820` ✓ (nice +1)
       `1024 * 1.25^5 = 3125.9 ≈ 3121` ✓ (nice -5)

### Fairness Property

Two tasks with weights w1 and w2 sharing one CPU receive CPU time in ratio:

```
share_1 / share_2 = w_1 / w_2
```

A nice 0 task (1024) and a nice +5 task (335) on the same CPU:
- Nice 0 share = 1024 / (1024 + 335) = 75.3%
- Nice +5 share = 335 / 1024 + 335) = 24.7%

### sched_latency, sched_min_granularity, sched_wakeup_granularity

Defaults (configurable via /proc/sys/kernel/, vary by kernel and CPU count):

| Tunable | Default (server) | Default (desktop) | Meaning |
|:---|:---:|:---:|:---|
| sched_latency_ns | 24,000,000 (24 ms) | 6,000,000 (6 ms) | Target sched period for ≤8 tasks |
| sched_min_granularity_ns | 3,000,000 (3 ms) | 750,000 (0.75 ms) | Minimum slice per task |
| sched_wakeup_granularity_ns | 4,000,000 (4 ms) | 1,000,000 (1 ms) | Min vruntime gap to preempt on wake |
| sched_migration_cost_ns | 500,000 (0.5 ms) | 500,000 | Penalty for cache-cold migration |

Server values are autoscaled: `sched_latency = sched_latency_base * (1 + log2(num_cpus))`.

### Period vs Slice

If `nr_running > sched_latency / sched_min_granularity` (typically > 8):

```
period = nr_running * sched_min_granularity
```

Else:

```
period = sched_latency
```

Each task's CPU slice within a period:

```
slice_i = period * (weight_i / total_weight)
```

Worked example — 4 nice-0 tasks on a desktop kernel:
- period = sched_latency = 6 ms
- slice each = 6 ms × (1024 / 4096) = 1.5 ms

### Red-Black Tree Selection

Runnable tasks are stored in a per-runqueue **red-black tree** keyed by vruntime. CFS always picks `cfs_rq->rb_leftmost` — the task with smallest vruntime — in O(log N) insert/remove and O(1) "next task" amortised (the leftmost is cached).

```c
/* kernel/sched/fair.c */
struct sched_entity *__pick_next_entity(struct cfs_rq *cfs_rq)
{
    return rb_entry(cfs_rq->rb_leftmost, struct sched_entity, run_node);
}
```

### Wake-Up Preemption

When a task wakes, CFS preempts the current task if:

```
curr->vruntime - se->vruntime > sched_wakeup_granularity
```

…i.e., the woken task has been "idle" long enough to deserve immediate CPU time.

### Min-vruntime and Newly-Woken Tasks

To prevent newly created or long-sleeping tasks from "starving" everyone with their tiny vruntime:

```
new_vruntime = max(se->vruntime, cfs_rq->min_vruntime - sched_latency/2)
```

This keeps newcomers near the scheduling frontier, never freezing everyone else.

### Autogroup (Per-Session) Fairness

If `kernel.sched_autogroup_enabled = 1`, tasks share a session's CPU quota proportionally regardless of how many threads they spawn.

```
session_share = total_cpu * autogroup_weight / sum_of_autogroup_weights
intra_session_share = session_share * weight_i / Σ weight_j (in session)
```

This is why a `make -j64` in one shell doesn't drown your interactive shell in another tmux pane.

---

## 2. Real-Time Scheduling

The kernel ships three real-time scheduling classes ahead of CFS:

| Class | Policy | Selection | Use case |
|:---|:---|:---|:---|
| SCHED_DEADLINE | Earliest Deadline First (EDF) + CBS | Smallest deadline | Hard real-time |
| SCHED_FIFO | Strict priority, no time-slicing | Highest prio (1-99) | Soft real-time |
| SCHED_RR | Strict priority + round-robin within prio | Highest prio, rotate | Soft real-time |

SCHED_DEADLINE preempts FIFO/RR; FIFO/RR preempt CFS.

### 2a. SCHED_FIFO

A SCHED_FIFO task runs until it blocks, exits, yields, or is preempted by a higher-priority RT task. Same-priority FIFO tasks form a queue: the head runs forever (no time-slicing).

```bash
chrt -f -p 80 $$       # set current shell to FIFO prio 80
chrt -f 99 cmd ...     # launch cmd at FIFO 99 (top RT prio)
```

### 2b. SCHED_RR

Same as FIFO except same-priority tasks round-robin every `sched_rr_timeslice_ms` (default 100 ms, settable via `/proc/sys/kernel/sched_rr_timeslice_ms`).

### 2c. SCHED_DEADLINE — EDF + Constant Bandwidth Server

A DEADLINE task is described by a tuple `(runtime, deadline, period)`:

- `runtime` — max ns of CPU per period the task may consume
- `deadline` — relative ns from job release at which work must finish
- `period` — ns between job releases

Constraints (rejected at admit time otherwise):

```
0 < runtime ≤ deadline ≤ period
```

### CBS Bandwidth

Each DEADLINE task has bandwidth:

```
U_i = runtime_i / period_i
```

### Admission Test (Schedulability)

Linux uses a **partitioned EDF** with a global cap. To admit a new DEADLINE task, the kernel checks per-CPU:

```
Σ U_i  +  U_new  ≤  sched_dl_runtime / sched_dl_period
```

Defaults from `/proc/sys/kernel/`:

```
sched_rt_runtime_us  = 950000   (950,000 µs)
sched_rt_period_us   = 1000000  (1,000,000 µs)
```

So total RT+DEADLINE bandwidth caps at **95%** per CPU. The remaining 5% is reserved so non-RT work (kthreads, init) cannot starve.

### Dline Throttling (the 95% rule)

If RT/DEADLINE exceed 95%, the kernel throttles for the rest of the period:

```
throttle_when:  used_runtime > sched_rt_runtime_us
release_at:     period boundary
```

Set `sched_rt_runtime_us = -1` to disable (dangerous: a runaway FIFO task hard-locks the system).

### Worked DEADLINE Example

A robot control loop needing 2 ms of CPU every 10 ms with a 5 ms deadline:

```
runtime  = 2,000,000 ns
deadline = 5,000,000 ns
period   = 10,000,000 ns
U        = 0.2 (20% of one CPU)
```

Admit if Σ U on target CPU + 0.2 ≤ 0.95.

```c
struct sched_attr attr = {
    .sched_policy   = SCHED_DEADLINE,
    .sched_runtime  = 2000000,
    .sched_deadline = 5000000,
    .sched_period   = 10000000,
    .size           = sizeof(attr),
};
sched_setattr(0, &attr, 0);
```

### EDF Selection

Among runnable DEADLINE tasks, the kernel picks the smallest **absolute deadline** (`dl_se->deadline = release_time + dl_se->dl_deadline`). On a tie, FIFO order.

---

## 3. Page Cache & Memory Reclaim

### 3a. The LRU Lists

Each NUMA node maintains 5 LRU lists per memory cgroup:

```
LRU_INACTIVE_ANON
LRU_ACTIVE_ANON
LRU_INACTIVE_FILE
LRU_ACTIVE_FILE
LRU_UNEVICTABLE
```

Promotion path: `inactive → active` on second access. Demotion: `active → inactive` when scanned by `shrink_active_list()`.

### LRU Movement Math

When kswapd or direct reclaim runs `shrink_lruvec()`, it scans both anon and file lists:

```
scan_anon = anon_lru_size * pressure / SWAP_CLUSTER_MAX
scan_file = file_lru_size * pressure / SWAP_CLUSTER_MAX
```

Where SWAP_CLUSTER_MAX = 32 (batch size).

### 3b. swappiness — Anon vs File Bias

`vm.swappiness` (0–200, default 60) biases reclaim:

```
ap = anon_pages * (swappiness / 200)
fp = file_pages * (200 - swappiness) / 200

scan_ratio = ap / (ap + fp)
```

Worked: 1 GB anon + 1 GB file with swappiness = 60:
- ap = 1024 × 0.30 = 307
- fp = 1024 × 0.70 = 717
- scan_ratio = 307 / 1024 = 30% from anon, 70% from file

| Swappiness | Behaviour |
|:---:|:---|
| 0 | Never swap unless OOM is imminent (file pressure must reach min) |
| 1 | Minimal swap (Postgres / databases tuning) |
| 60 | Default — balanced |
| 100 | Equal anon and file pressure |
| 180+ | Aggressive swap (zram-backed systems) |

### 3c. Direct Reclaim vs kswapd

```
free_pages > high_watermark    → no action
free_pages ≤ low_watermark     → wake kswapd (async, kthread per node)
free_pages ≤ min_watermark     → direct reclaim (allocator blocks until pages freed)
free_pages == 0 && reclaim ineffective → OOM killer
```

`kswapd` runs until `free_pages > high_watermark`. Direct reclaim runs in the allocator's caller context — your page fault stalls.

### 3d. Watermarks

For each zone (DMA, DMA32, Normal, Movable):

```
min_watermark  = pages_min  (from min_free_kbytes)
low_watermark  = min + min/4
high_watermark = min + min/2
```

`min_free_kbytes` derivation (in `mm/page_alloc.c`):

```
min_free_kbytes = 4 * sqrt(lowmem_kbytes)
```

…clamped to `[128, 262144]` KB.

| Memory | min_free_kbytes |
|:---:|:---:|
| 1 GB | ~4 MB |
| 16 GB | ~16 MB |
| 128 GB | ~45 MB |
| 1 TB | ~131 MB |

Tune up for large bursty workloads (databases, Java GC) to give kswapd more headroom and avoid direct reclaim stalls.

### 3e. PSI — Pressure Stall Information

PSI (kernel 4.20+) reports the fraction of time tasks were stalled:

- **some** — at least one task is stalled
- **full** — ALL non-idle tasks are stalled

Per resource: `/proc/pressure/{cpu,memory,io}`:

```
some avg10=0.51 avg60=0.32 avg300=0.18 total=12345678
full avg10=0.10 avg60=0.05 avg300=0.02 total=987654
```

Numbers are percentages of the last 10/60/300 seconds. `full` ≥ 5% on memory means real reclaim pain. Cgroup v2 surfaces the same fields in `memory.pressure`, `cpu.pressure`, `io.pressure`.

### Reclaim Efficiency

```
reclaim_efficiency = pages_reclaimed / pages_scanned
```

If efficiency drops below ~10% sustained, the kernel triggers OOM (no point scanning if nothing is freeable).

---

## 4. Memory Allocators

### 4a. Buddy Allocator — Power-of-Two Page Chunks

The buddy allocator manages pages in **orders**:

```
order 0  → 1 page    = 4 KiB
order 1  → 2 pages   = 8 KiB
order 2  → 4 pages   = 16 KiB
...
order 10 → 1024 pages = 4 MiB
order 11 → 2048 pages = 8 MiB  (MAX_ORDER on most arches; 11 by default)
```

Allocate request size $S$:

```
order = ceil(log2(S / PAGE_SIZE))
```

### Splitting and Coalescing

To allocate order N, the allocator finds the smallest free block of order ≥ N. If found at order M > N, it splits down to N, freeing the buddy halves into orders M-1, M-2, ..., N.

On free, the allocator checks the buddy:

```
buddy_pfn = pfn ^ (1 << order)
```

If the buddy is free and same order, coalesce into order+1 and recurse.

### Fragmentation Metric

```
free_external_fragmentation = 1 - (largest_free_block / total_free)
```

`/proc/buddyinfo` shows free counts per order:

```
Node 0, zone   Normal  823  411   55   12    3    1    0    0    0    0    0
                       ^0   ^1   ^2   ^3   ^4   ^5   ^6   ^7   ^8   ^9   ^10
```

If high orders are zero, large allocations fail with `-ENOMEM` even though plenty of order-0 pages exist. That's external fragmentation.

### Compaction

When fragmentation is high, the kernel runs **memory compaction** (`kcompactd`) to migrate movable pages and merge free regions:

```
compaction_score = free_blocks_at_high_order / total_zone_pages
```

Trigger via `vm.compaction_proactiveness` (default 20).

### 4b. SLAB / SLUB / SLOB — Object Caches

Above the buddy allocator sit object caches for small allocations. The default since kernel 6.5 is **SLUB** (SLAB removed in 6.8).

A SLUB cache is a list of slabs:

```
slab = order_n_pages from buddy
slab_size_bytes = (1 << order) * 4096
objs_per_slab = slab_size / object_size
```

For `kmalloc-128` cache on x86-64 with order=0 slabs:

```
slab = 4096 B
objs_per_slab = 4096 / 128 = 32 objects
```

### Allocation Path

```
kmalloc(size)
  → find kmalloc-XX cache (XX = next pow2 ≥ size)
  → kmem_cache_alloc(cache, gfp)
    → per-CPU partial slab freelist (fast path, lockless)
    → per-node partial slabs   (slow path)
    → request new slab from buddy (slowest)
```

Fast path is ~5-15 ns; slow path with new slab can hit µs.

### kmalloc Size Classes

```
kmalloc-8, kmalloc-16, kmalloc-32, kmalloc-64,
kmalloc-128, kmalloc-192, kmalloc-256, kmalloc-512,
kmalloc-1k, kmalloc-2k, kmalloc-4k, kmalloc-8k,
kmalloc-16k, ..., kmalloc-8M
```

Up to ~8 MB; above that, `vmalloc` or direct `__get_free_pages`.

Internal fragmentation:

```
overhead = kmalloc_class_size - requested_size
avg_overhead ≈ requested_size * 0.1   (10-15% typical)
```

### 4c. Per-CPU Page Allocator Caches

To avoid lock contention on the buddy allocator zone lock, each CPU has a hot/cold list of pages:

```
pcp_high = batch * 6
batch    = max(1, zone_managed_pages / 1024 / num_possible_cpus())  /* clamped */
```

A page free first hits the per-CPU list; only when `count > pcp_high` does it drain back to the zone. This makes order-0 alloc/free almost lock-free.

---

## 5. Page Tables & TLB

### 5a. Page Table Hierarchy

x86-64 with 48-bit virtual addresses (256 TB user + 256 TB kernel):

```
VA[47:39] = PML4 index (9 bits → 512 entries)
VA[38:30] = PDPT index
VA[29:21] = PD   index
VA[20:12] = PT   index
VA[11:0]  = page offset (4 KiB)
```

Each level: 4 KiB table × 512 × 8-byte entries.

5-level paging (Ice Lake+, 57-bit VA → 128 PB):

```
VA[56:48] = PML5 index
... (rest identical)
```

### Page Table Walk Cost

Cold walk on miss:

```
walk_cost = 4 * (cache_miss_latency)  ≈ 4 * 100 ns = 400 ns
```

(Five memory loads on 5-level paging; modern CPUs cache intermediate translations in MMU caches.)

### Address Calculation

```
PTE_addr = CR3 + PML4_idx*8
PDPT_addr = (PTE_addr->next) + PDPT_idx*8
...
phys_addr = (PT_entry & PAGE_MASK) | offset
```

### 5b. TLB Sizes (Modern Intel/AMD)

| Cache | Typical Intel (Skylake/Ice Lake/Sapphire Rapids) | AMD Zen 3/4 |
|:---|:---:|:---:|
| L1 dTLB (4K) | 64 entries | 64 entries |
| L1 dTLB (2M) | 32 entries | 32 entries |
| L1 dTLB (1G) | 4 entries | 8 entries |
| L1 iTLB (4K) | 128 entries | 64 entries |
| L2 STLB (unified) | 1024–2048 | 1024–3072 |

Reach of L1 dTLB at 4K: `64 × 4 KB = 256 KB`. STLB at 4K: `1024 × 4 KB = 4 MB`. Working sets larger than the TLB reach incur constant page-walks.

### 5c. Huge Pages — TLB Reach Multiplied

```
2 MiB huge:  reach_per_entry = 2,097,152 B  = 512× reach of 4K
1 GiB huge:  reach_per_entry = 1,073,741,824 B = 262,144× reach
```

Configure persistent hugetlb:

```bash
sysctl vm.nr_hugepages=1024   # 1024 × 2 MiB = 2 GiB
sysctl vm.nr_hugepages_mempolicy
cat /proc/meminfo | grep -i huge
```

### 5d. Transparent Huge Pages (THP)

THP allows the kernel to opportunistically promote 512 contiguous 4K pages to a 2 MiB page:

```
/sys/kernel/mm/transparent_hugepage/enabled = always | madvise | never
/sys/kernel/mm/transparent_hugepage/defrag  = always | defer | madvise | never
```

### khugepaged Scan

Background daemon `khugepaged` scans for promotable regions:

```
scan_interval_ms     = 10000   (10 s default)
pages_to_scan        = 4096    (per cycle)
max_ptes_none        = 511     (allow up to 511 zero PTEs in 512 to still promote)
```

Math: a complete scan of N pages takes `(N / 4096) * 10 s`. On a 64 GB box, ~16M pages → full scan in ~40,000 s (≈ 11 hours). THP is best-effort.

### TLB Shootdown

When a CPU updates a PTE, all CPUs caching that translation must invalidate:

```
shootdown_cost ≈ IPI_latency + flush_overhead
              ≈ 1-3 µs per CPU on x86 (NMI-class IPI)
```

For 64-CPU systems doing many small mmap/munmap calls, this is a real bottleneck. Mitigations: lazy TLB, INVLPG with PCID, batched invalidation.

---

## 6. I/O Subsystem

### 6a. The Block Layer Path

```
write()/read()/mmap'd dirty page
        ↓
filesystem (vfs) → bio (block I/O descriptor)
        ↓
block layer → request (merged bios, sorted)
        ↓
multi-queue (blk-mq) per-CPU software queue
        ↓
hardware queue (one per HW context)
        ↓
device driver → DMA
```

### bio → request Merging

Adjacent bios (contiguous LBAs) are merged into one request to amortise driver overhead:

```
merge_window = elv_merge_window_ms (default 32 ms for some schedulers)
max_merge_size = max_sectors_kb (default 1280 KB on NVMe, 512 KB on SATA)
```

### 6b. I/O Schedulers

Configure via `/sys/block/<dev>/queue/scheduler`.

| Scheduler | Best for | Algorithm |
|:---|:---|:---|
| none (noop) | NVMe / virtio-blk | FIFO, no reordering |
| mq-deadline | Spinning disks, mixed | Read deadline 500 ms, write 5 s |
| kyber | Low-latency, fixed targets | P99 read = 2 ms, write = 10 ms |
| bfq | Desktop, fairness | Weighted budget per cgroup |

### mq-deadline Math

Two deadline lists (read and write) plus two sorted (LBA-ordered) lists:

```
read_deadline_ms  = 500   (default)
write_deadline_ms = 5000  (default)
writes_starved    = 2     (max consecutive read batches before write)
```

Pick request:

```
if expired_request_in_read_list:    pick it
elif expired_request_in_write_list: pick it
elif starvation_count_against_writes > writes_starved: pick from write_sorted
else: pick from read_sorted   (sequential bias)
```

### kyber

Maintains a P99 latency target:

```
target_read_p99_us  = 2000
target_write_p99_us = 10000
```

If observed p99 exceeds target, kyber throttles the offending queue depth:

```
new_depth = max(1, old_depth * (target / observed))
```

### bfq Weighting

Each cgroup or process has weight `[1, 1000]` (default 100). Each round, a process gets a budget proportional to its weight:

```
budget_i = base_budget * (weight_i / Σ weight_j)
```

When budget is exhausted, the next process is selected. bfq is throughput-fair across cgroups, latency-fair within them — great for desktops, often slow for parallel db workloads.

### nr_requests

Per-queue cap:

```
/sys/block/<dev>/queue/nr_requests = 128 (default)
```

Throughput vs latency: larger `nr_requests` = more parallelism but higher tail latency under load.

### 6c. Page Cache Writeback

Dirty pages from page cache flush back to disk via `writeback` kthreads. Tunables:

```
vm.dirty_ratio              = 20    (% of total RAM dirty before sync write)
vm.dirty_background_ratio   = 10    (% before async kflushd starts)
vm.dirty_expire_centisecs   = 3000  (30 s — pages dirty this long flushed unconditionally)
vm.dirty_writeback_centisecs = 500  (5 s — wakeup interval)
```

If dirty memory exceeds `dirty_ratio`, **writes block in the writer's context** until under threshold. This is the source of mysterious multi-second `write()` stalls.

### Throttling Math

```
nr_dirty_pages > total_pages * dirty_background_ratio / 100  → wakeup writeback
nr_dirty_pages > total_pages * dirty_ratio / 100             → blocking write throttle
```

For 16 GB RAM at default 20%, the threshold is 3.2 GB. A dd into a slow disk hits this in seconds.

---

## 7. Network Stack Math

### 7a. sk_buff Lifecycle

```
NIC RX → DMA into ring → IRQ → NAPI poll
       → alloc sk_buff → fill from ring buffer
       → __netif_receive_skb → IP → TCP/UDP
       → socket queue → recvmsg copy → user
```

`sk_buff` is the universal packet descriptor (~256 B header + linear data + frags).

### 7b. NAPI Polling

To avoid an IRQ per packet, NAPI switches to polling under load:

```
napi_poll_budget = 64   (default; net.core.netdev_budget)
napi_poll_time_us = 2000 (net.core.netdev_budget_usecs)
```

A NAPI driver IRQ disables further interrupts and schedules a poll. The poll consumes up to `budget` packets, then re-enables IRQs if queue empty, else reschedules.

```
effective_pps_per_cpu = budget / poll_cost
                     ≈ 64 / 1µs = 64 Mpps   (theoretical, before stack cost)
```

With the full stack: ~1-3 Mpps per core, depending on protocol.

### 7c. RPS / RFS — Steering

**RPS (Receive Packet Steering)** spreads soft-IRQ work across CPUs:

```
target_cpu = cpu_map[skb->hash mod cpu_map_size]
```

`skb->hash` is computed from the 4-tuple (src/dst IP + ports), giving consistent flow-to-CPU mapping.

**RFS (Receive Flow Steering)** extends RPS to track which CPU last touched the flow's userspace consumer (via `rps_sock_flow_table`), so packets land on the cache-warm CPU:

```
cpu = rps_sock_flow_table[hash & mask].cpu  if recently set
    else rps_cpus_for_queue[hash mod count]
```

Configure:

```bash
echo ffff > /sys/class/net/eth0/queues/rx-0/rps_cpus
echo 32768 > /proc/sys/net/core/rps_sock_flow_entries
echo 4096  > /sys/class/net/eth0/queues/rx-0/rps_flow_cnt
```

### 7d. GRO / LRO

**Generic Receive Offload** merges adjacent packets into one large skb before stack entry:

```
gro_max_packets   = 64
gro_max_age_us    = ~50 µs
merged_skb_size   = up to 64 KB
```

Effect: `n` 1500 B packets become one 64 KB skb → `n×` cheaper stack pass.

### 7e. TCP Buffer Auto-Tuning

```
net.ipv4.tcp_rmem = 4096  131072  6291456    (min, default, max in bytes)
net.ipv4.tcp_wmem = 4096   16384  4194304
```

Auto-tune algorithm:

```
target_window = 2 * BDP   (BDP = bandwidth × RTT)
new_rcvbuf = min(tcp_rmem.max, max(tcp_rmem.min, target_window))
```

For a 10 Gbps × 100 ms link: BDP = 125 MB, target = 250 MB — exceeds default max (6.3 MB). Bump `tcp_rmem.max` to ~256 MB for long fat networks.

### 7f. SO_REUSEPORT Load Distribution

When N sockets bind to the same port with SO_REUSEPORT, the kernel hashes incoming connections:

```
target_socket = hash(4-tuple) mod N
```

Even distribution (with random source-port traffic). Per-CPU `SO_INCOMING_CPU` plus `SO_REUSEPORT` lets you pin one accept-socket per CPU, eliminating cross-CPU wakeups.

### Connection Establishment Cost

```
tcp_3whs_rtts = 1.5 RTT (SYN → SYN-ACK → ACK)
tls13_rtts    = 1   RTT (additional)
http2_setup   = 0.5 RTT (SETTINGS / ACK pipelined)
```

For a 50 ms RTT: TLS 1.3 over TCP = 125 ms before first byte. TCP Fast Open + TLS 1.3 0-RTT = 50 ms.

---

## 8. Locking Primitives

### Lock Hierarchy (cheapest → most permissive)

| Primitive | Sleep? | IRQ-safe? | Fair? | Cost (uncontended) |
|:---|:---:|:---:|:---:|:---:|
| atomic_t / READ_ONCE | n/a | yes | n/a | 1 cycle |
| spinlock_t | no | yes (variant) | yes (queued) | 5-15 ns |
| rwlock_t | no | yes (variant) | reader-bias | 10-25 ns |
| mutex | yes | no | yes | 20-50 ns |
| rw_semaphore | yes | no | configurable | 30-80 ns |
| RCU read | no | yes | n/a | ~0 ns (NOP on PREEMPT_NONE) |
| seqlock | no | yes | writer-bias | 5 ns reader, 10 ns writer |

### 8a. spinlock_t — Queued Spinlock

Linux uses a **MCS-like queued spinlock** since 4.2. Each waiter spins on its own cacheline, eliminating the cacheline ping-pong of test-and-set:

```
contended_cost ≈ #waiters_ahead * cache_line_transfer
              ≈ N * ~50 ns
```

Variants:

- `spin_lock_irqsave(lock, flags)` — disable local IRQs (safe from IRQ handlers)
- `spin_lock_bh(lock)` — disable softIRQs only
- `spin_lock(lock)` — disable preemption only (assumes no IRQ handler will grab this lock)

### 8b. mutex — Sleep-On-Contention with Adaptive Spin

```
lock_attempt:
  if (lock_owner_running):
      spin_loop()       # adaptive spin: up to ~osq_lock period
  else:
      schedule()
```

If the holder is currently running (likely to release soon), spin. Otherwise sleep. Avoids context switch when the lock is held briefly.

### 8c. RCU — Read-Copy-Update

The fast-path read is:

```c
rcu_read_lock();
p = rcu_dereference(ptr);
use(p);
rcu_read_unlock();
```

On `PREEMPT_NONE`, `rcu_read_lock()` is just `preempt_disable()` — a single store. Reads scale linearly with cores; writers pay the cost.

### Grace Period

A "grace period" is a window long enough for every CPU to have left every pre-existing read-side critical section. Writers do:

```
new_obj = kmalloc()
populate(new_obj)
rcu_assign_pointer(global, new_obj)
synchronize_rcu()           // blocks current task until grace period elapses
kfree(old_obj)
```

`synchronize_rcu` typically takes 1-30 ms (one tick on each CPU + grace handshake). The asynchronous variant `call_rcu(head, callback)` enqueues the free without blocking.

Approximate cost:

```
grace_period_ms ≈ jiffies_to_ms(2 * HZ)   (typically 2 ticks → 2-20 ms)
```

### 8d. Read-Side Critical Section Rules

Inside `rcu_read_lock()`:
- May not block (sleep, mutex, kmalloc(GFP_KERNEL))
- May call `kfree_rcu()` (deferred free)
- May not call `synchronize_rcu()` (deadlock)

### 8e. lockdep — Static Lock Ordering Validator

`CONFIG_PROVE_LOCKING=y` enables runtime tracking of every lock acquisition order. Lockdep builds a directed graph of "lock A acquired while holding lock B" edges. A cycle = potential deadlock. Detected lazily, but once observed always reported.

### 8f. Per-CPU Variables

```c
DEFINE_PER_CPU(int, count);
this_cpu_inc(count);            /* preempt-safe, single insn */
__this_cpu_inc(count);          /* faster, requires preempt disabled */
```

Aggregate via `for_each_possible_cpu(cpu) per_cpu(count, cpu)`.

### preempt_disable / local_bh_disable

```
preempt_disable()  → no scheduling preemption (still IRQs/softIRQs)
local_bh_disable() → no softIRQ on this CPU (still hard IRQs)
local_irq_disable()→ no interrupts on this CPU (still NMIs)
```

Holding for >100 µs triggers `softlockup`/`hardlockup` watchdogs.

---

## 9. eBPF — Verifier and Runtime

### 9a. Verifier Limits

```
BPF_COMPLEXITY_LIMIT_INSNS = 1,000,000   (kernel 5.3+, was 4096 pre-5.3)
BPF_MAXINSNS               = 4,096       (per program; tail-calls extend total)
BPF_MAX_TAIL_CALL_CNT      = 33          (chain depth)
MAX_BPF_FUNC_ARGS          = 12
```

The verifier explores all reachable instruction sequences with an abstract register state. Worst-case time: `O(insns × states)`; practical: 1-100 ms.

### 9b. Bounded Loops

Since 5.3, loops are allowed if the verifier can prove a finite bound:

```c
for (int i = 0; i < N; i++) { ... }   // N must be a verifiable constant
```

Implementation: the verifier tracks the loop counter as a bounded scalar. If at any iteration the counter range stays bounded, success.

### 9c. JIT

```
arch/x86/net/bpf_jit_comp.c     (x86-64)
arch/arm64/net/bpf_jit_comp.c   (arm64)
arch/s390/net/bpf_jit_comp.c    (s390)
```

JITed bpf is typically 3-5× faster than the interpreter. Toggle via:

```bash
sysctl -w net.core.bpf_jit_enable=1
sysctl -w net.core.bpf_jit_harden=2   # constant blinding (anti-spectre)
```

### 9d. Helper Function Surface

A small selection from `include/uapi/linux/bpf.h`:

```
bpf_map_lookup_elem(map, key)
bpf_map_update_elem(map, key, value, flags)
bpf_map_delete_elem(map, key)
bpf_ringbuf_reserve / bpf_ringbuf_submit
bpf_perf_event_output(ctx, map, flags, data, size)
bpf_get_current_pid_tgid()
bpf_ktime_get_ns()
bpf_get_smp_processor_id()
bpf_probe_read_kernel(dst, size, src)
bpf_probe_read_user(dst, size, src)
bpf_redirect / bpf_clone_redirect / bpf_redirect_map
bpf_trace_printk(fmt, ...)
```

### 9e. Tail Calls

```c
bpf_tail_call(ctx, &prog_array, idx);   // jump to program at prog_array[idx]
```

Cost: ~10-30 ns. Limit: 33 chained calls (`MAX_TAIL_CALL_CNT`). Effective program size up to `33 × 1M = 33M` instructions.

### 9f. CO-RE & BTF

Compile Once, Run Everywhere relies on BTF (BPF Type Format). At program load, the loader rewrites field offsets using the running kernel's BTF:

```
relocations: [field name → offset] resolved at load
typical kernel BTF: 2-5 MB, ~50,000 types
```

---

## 10. Cgroup v2 Resource Math

Single unified hierarchy at `/sys/fs/cgroup/`. No per-controller hierarchies (v1 had this).

### 10a. CPU Controller

```
cpu.weight       1-10000 (default 100)
cpu.weight.nice  -19..20 (alternative, mapped to weight)
cpu.max          "$max $period"   (hard cap)
cpu.max.burst    burst-budget bytes (4.18+ accumulator)
```

### Weight Math

A child cgroup's CPU share among siblings:

```
share_i = weight_i / Σ weight_j (siblings)
absolute_share = parent_absolute_share * share_i
```

Recursive: a cgroup's effective CPU is its parent's share × its weight ratio.

### cpu.max Throttling

```
cpu.max = "200000 100000"   # 200 ms runtime per 100 ms period = 2 CPUs cap
```

If usage exceeds `runtime` in any `period`, the cgroup is throttled until period boundary. Stats in `cpu.stat`:

```
nr_periods         total periods elapsed
nr_throttled       periods where throttle fired
throttled_usec     accumulated throttle time
```

P99 latency in throttled cgroups can be brutal — every period boundary is a stall risk.

### 10b. Memory Controller

Two soft/hard caps:

```
memory.high  = soft cap  (throttle allocations above this; trigger reclaim)
memory.max   = hard cap  (kill on overflow; OOM-in-cgroup)
memory.low   = protected reserve  (last to be reclaimed under global pressure)
memory.min   = guaranteed  (never reclaimed under global pressure)
```

### Throttle Math

Above `memory.high`, allocations sleep proportionally to overshoot:

```
sleep_ns = penalty(usage - high, memory.max)
```

where penalty grows superlinearly. Process never dies; just gets slower.

Above `memory.max`: OOM killer runs **within the cgroup** (the global system stays healthy).

### 10c. I/O Controller

```
io.weight default 100, range 1-10000
io.max    "$MAJOR:$MINOR rbps=N wbps=N riops=N wiops=N"
io.latency target latency in microseconds (kernel chooses depth)
```

### Weight-Based Dispatch

bfq or io.weight in cgroup v2 distributes bandwidth proportionally to weight:

```
dispatch_share_i = weight_i / Σ weight_j
```

### io.latency Target

The kernel adjusts the cgroup's effective queue depth to keep observed read latency ≤ target:

```
if observed_p99_us > target:  shrink depth
if observed_p99_us << target: grow depth
```

### 10d. PSI in Cgroups

Each cgroup has `cpu.pressure`, `memory.pressure`, `io.pressure`:

```
some avg10=0.42 avg60=0.21 avg300=0.10 total=...
full avg10=0.05 avg60=0.02 avg300=0.01 total=...
```

`memory.pressure full > 5%` over `avg60` is a strong signal to either bump `memory.max`, kill the cgroup, or migrate work elsewhere. Userspace OOM daemons (oomd, systemd-oomd) act on these numbers.

---

## 11. Boot, Init, & Modules

### 11a. initcall Ordering

```c
early_initcall(...)   // before SMP, before scheduler
pure_initcall(...)    // level 0 — only safe for pure-info routines
core_initcall(...)    // level 1 — core kernel subsystems
postcore_initcall(...)
arch_initcall(...)    // level 3 — architecture-specific
subsys_initcall(...)
fs_initcall(...)
rootfs_initcall(...)
device_initcall(...)  // level 6 — device drivers
late_initcall(...)    // level 7 — last chance
```

Within a level, link order determines call order. Use `late_initcall` for anything that depends on devices being live.

### 11b. Module Loading

`modprobe foo`:

1. Read `/lib/modules/$(uname -r)/modules.dep` for dependency closure
2. Load each dependency in topological order (depth-first)
3. For each module:
   a. Read .ko file from disk
   b. Resolve relocations against running kernel symbols (`/proc/kallsyms`)
   c. Run `module_init()`, which dispatches via `module_init_func`
4. Refcount tracking (`/sys/module/<name>/refcnt`) gates removal

```bash
modinfo nvme_core    # show dependencies, params, signatures
lsmod | grep nvme    # show refcount and used-by chain
```

### Module Memory

Kernel modules live in vmalloc area. Total module footprint:

```
total_module_mem = Σ (text + ro_data + rw_data + bss + parm) for each loaded module
```

Capped by `MODULES_END - MODULES_VADDR` (typically 1-2 GB on x86-64).

### 11c. Kernel Command-Line Parsing

Two parser stages:

```c
early_param("foo", parse_foo)   // parsed BEFORE setup_arch()
__setup("bar=", parse_bar)      // parsed during start_kernel()
module_param(baz, int, 0644)    // for dynamically loaded modules
```

`early_param` is the hammer for things like `nokaslr`, `hugepages=N`, `nohz_full=...` that must take effect before mm/sched come up.

---

## 12. Tracing & Observability Math

### 12a. perf Events — Sample Rate vs Overhead

```
sample_period = ceil(cpu_freq_hz / target_sample_hz)
event_freq    = 1 / sample_period
overhead      = event_freq * (sample_handler_cost + stack_walk_cost)
```

Defaults:

```
perf_event_max_sample_rate = 100000   (samples/sec, can be raised to ~10k typical)
sample_handler_cost ≈ 1-5 µs (with backtrace)
```

At 4000 Hz sampling on a 4 GHz CPU: every 1M cycles → ~0.5% overhead.

### 12b. ftrace — function_graph Depth and Buffer

```
buffer_size_kb_per_cpu = 1408   (default; configurable per-CPU)
function_graph max_depth = unlimited (clamp via set_graph_function)
```

Storage:

```
total_ring_buffer = nr_cpu * buffer_size_kb_per_cpu
```

64-CPU box at 1408 KB/CPU = 88 MB ring buffer. Events evicted FIFO.

Per-event cost (function tracer): 50-300 ns when enabled, 0 ns when disabled (NOP-patched).

### 12c. Tracepoints (Static)

```c
trace_sched_switch(prev, next);
```

Compiles to `if (unlikely(__tracepoint_sched_switch.enabled)) call(...)`. The branch is a NOP-patched conditional that costs essentially nothing when the tracepoint is disabled — the entire callsite is rewritten to skip the call via the static-branches mechanism. When enabled, cost is 30-100 ns (hash lookup of registered probes + invocation).

### 12d. kprobes / uprobes (Dynamic)

```
kprobe_overhead   ≈ 1-3 µs   (INT3 trap + handler)
optimized_kprobe ≈ 100-300 ns (jump-patched, no trap)
uprobe_overhead   ≈ 5-15 µs (page table dirty + signal)
```

uprobes are much more expensive because they touch userspace pages; place them sparingly and prefer USDT.

### 12e. eBPF Tracing Cost

```
bpf_program_cost = (insn_count × cycle_per_insn) + Σ helper_cost
                ≈ 50-500 ns for a typical tracer
```

For a 100k-events/sec workload at 200 ns/event: 0.02 ms/sec = 2% overhead.

### 12f. PSI as Observability

PSI is the cheapest "is the system in pain?" check:

```
read /proc/pressure/{cpu,memory,io}   ≈ 1-5 µs
```

Cheap enough to scrape every second. Far cheaper than running perf.

---

## 13. Worked End-to-End Examples

### 13a. Why Your Database Server Stalls After 23 Hours of Uptime

Symptom: P99 latency spikes 100× at exactly 1 AM nightly.

Path through the math:

1. cron triggers a `find / -name '*.log' -mtime +30` at 1 AM.
2. `find` reads metadata for 30M files, flooding the page cache with inode + dentry pressure.
3. At ~80% of `dirty_ratio` (default 20%), the database's `pg_writer` thread starts blocking in `balance_dirty_pages()`.
4. Reclaim falls back to direct reclaim because kswapd was already at the high watermark.
5. PSI `memory.pressure full avg10` jumps to 30%.

Fixes (each from the math above):
- Lower `vm.dirty_background_ratio` from 10 → 3 (start flushing earlier).
- Cap `find`'s page cache via cgroup v2 `memory.high`.
- Set `vm.swappiness=10` for database hosts.
- Use `nice -n 19 ionice -c 3 find ...` so CFS de-prioritises it.

### 13b. Why Your "Empty" Queue Burns a Whole Core

Symptom: `top` shows kworker/0:* at 100% CPU on a system that's "doing nothing".

Path:

1. A driver registers a NAPI poll function that runs even when ring is empty.
2. With `net.core.netdev_budget = 64` and IRQ moderation off, NAPI loops constantly.
3. RPS not configured → all softIRQ work pinned to CPU 0.

Fix:

```bash
ethtool -C eth0 rx-usecs 50 rx-frames 32        # IRQ coalescing
echo ffff > /sys/class/net/eth0/queues/rx-0/rps_cpus  # spread across 16 CPUs
sysctl -w net.core.netdev_budget=300            # allow longer poll, fewer wakeups
```

### 13c. Why a `mmap` Performance Regression Appeared After Kernel Upgrade

Symptom: a worker that maps a 200 GB file is now 30% slower.

Diagnosis:

```bash
perf stat -e dTLB-loads,dTLB-load-misses,iTLB-load-misses ./worker
```

Old kernel: `dTLB-load-misses` at 0.2%. New kernel: 4.5%.

Root cause: `transparent_hugepage/enabled` switched from `always` → `madvise` after distro upgrade, and the worker doesn't call `madvise(MADV_HUGEPAGE)`.

Fix: either restore `always`, or patch the worker to madvise its mapping.

### 13d. Why Your Real-Time Audio Pipeline Crackles

Symptom: SCHED_FIFO 80 audio thread skips every few seconds.

Diagnosis:

```bash
cat /proc/sys/kernel/sched_rt_runtime_us   # 950000
cat /proc/sys/kernel/sched_rt_period_us    # 1000000
```

A second (lower-priority) FIFO thread has been added by the new audio plugin and is consuming 940,000 µs/period. Total RT use 99.5% > 95% → throttling fires for the last 50 ms of every period.

Fix: lower the plugin to SCHED_OTHER, or set `sched_rt_runtime_us = 990000` (97% — risky), or pin the audio thread to a dedicated isolated CPU (`isolcpus=3 nohz_full=3 rcu_nocbs=3`).

---

## 14. Summary Tables

### Scheduler Constants

| Constant | Value | Source |
|:---|:---:|:---|
| NICE_0_LOAD | 1024 | kernel/sched/sched.h |
| MAX_NICE | 19 | include/linux/sched/prio.h |
| MIN_NICE | -20 | include/linux/sched/prio.h |
| MAX_RT_PRIO | 100 | include/linux/sched/rt.h |
| BPF_MAX_TAIL_CALL_CNT | 33 | include/linux/bpf.h |
| BPF_COMPLEXITY_LIMIT_INSNS | 1,000,000 | include/linux/bpf_verifier.h |

### Default Sysctls Worth Knowing

| Key | Default | Purpose |
|:---|:---:|:---|
| vm.swappiness | 60 | Anon vs file reclaim ratio |
| vm.dirty_ratio | 20 | Dirty page sync block threshold (%) |
| vm.dirty_background_ratio | 10 | kflushd start threshold (%) |
| vm.dirty_expire_centisecs | 3000 | Dirty page max age (cs) |
| vm.min_free_kbytes | 4·sqrt(RAM) | Reclaim watermark base |
| kernel.sched_latency_ns | 24000000 | CFS target period |
| kernel.sched_min_granularity_ns | 3000000 | Min slice |
| kernel.sched_rt_runtime_us | 950000 | RT bandwidth cap |
| kernel.sched_rt_period_us | 1000000 | RT period |
| net.core.netdev_budget | 64 | NAPI poll budget |
| net.core.somaxconn | 4096 | Listen backlog cap |
| net.ipv4.tcp_rmem | 4k 128k 6M | TCP recv auto-tune range |
| net.ipv4.tcp_wmem | 4k 16k 4M | TCP send auto-tune range |
| fs.file-max | nr_pages/10 | Global FD limit |

### Big-O Reference

| Operation | Complexity |
|:---|:---|
| CFS pick_next_task | O(1) amortised (cached leftmost) |
| CFS enqueue/dequeue | O(log N) RB-tree |
| Buddy alloc / free | O(MAX_ORDER) ≤ 11 |
| SLUB fast-path alloc | O(1) lockless |
| Hash map lookup (eBPF) | O(1) avg |
| LPM trie lookup | O(K) in key bits |
| Page-walk (4-level) | 4 cache loads, ~400 ns cold |
| TLB shootdown | O(N_cpu) IPI fan-out |
| RCU read-side | O(1), often free |
| synchronize_rcu | O(1) but blocks 1-30 ms |

---

## 15. Engineering Tradeoffs

| Choice | Cost | Benefit |
|:---|:---|:---|
| Higher `vm.dirty_ratio` | Larger write stalls | More page-cache absorption |
| `transparent_hugepage=always` | khugepaged CPU, occasional latency spike | Lower TLB miss rate |
| `nohz_full=N` | Slightly higher per-tick overhead on housekeeping CPU | Tickless on isolated CPUs |
| `SCHED_FIFO` workers | Risk of starvation if buggy | Deterministic latency |
| `cpu.max` (cgroup throttle) | P99 cliff at period boundary | Tenant isolation |
| `BPF_MAP_TYPE_PERCPU_HASH` | N_cpu × memory cost | Lockless writes |
| `BPF_MAP_TYPE_RINGBUF` | Single producer per write | Shared, ordering preserved |
| `none` I/O scheduler | No fairness across cgroups | Lowest dispatch latency on NVMe |
| `bfq` I/O scheduler | More CPU per I/O | Fairness on rotating media / desktop |
| Larger `nr_requests` | Higher latency tail | More merging, more throughput |
| Higher `net.core.netdev_budget` | Longer softIRQ runs | Fewer context switches at high pps |

---

## 16. Diagnostic Recipes

```bash
# Scheduler latency (kernel ≥ 4.4)
mpstat -P ALL 1
pidstat -t -d 1
cat /proc/schedstat | head
perf sched record -- sleep 5 && perf sched latency

# Page cache and reclaim
cat /proc/meminfo | egrep 'Active|Inactive|Dirty|Writeback|Slab'
vmstat 1
sar -B 1
cat /proc/pressure/memory

# Buddy fragmentation
cat /proc/buddyinfo
cat /proc/pagetypeinfo

# TLB pressure
perf stat -e dTLB-load-misses,iTLB-load-misses,page-faults,minor-faults ./prog

# I/O scheduler tuning
cat /sys/block/nvme0n1/queue/scheduler
cat /sys/block/nvme0n1/queue/nr_requests
iostat -xz 1

# Network softIRQ load
cat /proc/softirqs | head -3 ; cat /proc/softirqs | grep NET
sar -n DEV 1
ethtool -S eth0 | egrep 'rx_dropped|tx_dropped|fifo'

# Cgroup v2 pressure
find /sys/fs/cgroup -name memory.pressure -exec head -2 {} +
systemd-cgtop

# eBPF program cost
bpftool prog show
bpftool prog profile id <ID> duration 5 cycles instructions

# RCU
cat /sys/kernel/debug/rcu/rcu_sched/rcugp
```

---

## See Also

- `system/systemd`
- `system/sysctl`
- `system/cgroups`
- `performance/ebpf`
- `performance/perf`
- `ramp-up/linux-kernel-eli5`

---

## References

- kernel.org Documentation/ — `Documentation/scheduler/sched-design-CFS.rst`, `Documentation/admin-guide/cgroup-v2.rst`, `Documentation/admin-guide/mm/transhuge.rst`, `Documentation/admin-guide/sysctl/vm.rst`
- `kernel/sched/fair.c`, `kernel/sched/core.c`, `kernel/sched/deadline.c`, `kernel/sched/rt.c` — CFS, DEADLINE, RT scheduling
- `mm/page_alloc.c`, `mm/slub.c`, `mm/vmscan.c`, `mm/compaction.c`, `mm/page-writeback.c` — allocators, reclaim, writeback
- `mm/khugepaged.c`, `mm/huge_memory.c` — THP and khugepaged
- `block/blk-mq.c`, `block/mq-deadline.c`, `block/kyber-iosched.c`, `block/bfq-iosched.c` — block layer
- `net/core/dev.c`, `net/core/dev_addr_lists.c`, `net/ipv4/tcp_input.c`, `net/ipv4/tcp_output.c` — net stack
- `kernel/rcu/tree.c`, `kernel/locking/qspinlock.c`, `kernel/locking/mutex.c` — locking
- `kernel/bpf/verifier.c`, `kernel/bpf/core.c`, `arch/x86/net/bpf_jit_comp.c` — eBPF
- `Documentation/accounting/psi.rst` — Pressure Stall Information
- LKML — search lore.kernel.org for canonical discussions; Ingo Molnar's sched threads, Linus's mm responses
- Brendan Gregg, *Systems Performance* (2nd ed.) — observability methodology and diagnostic flow
- Brendan Gregg, *BPF Performance Tools* — applied eBPF tracing
- Robert Love, *Linux Kernel Development* (3rd ed.) — kernel internals reference
- Mauerer, *Professional Linux Kernel Architecture* — exhaustive subsystem walk
- Bovet & Cesati, *Understanding the Linux Kernel* (3rd ed.) — classic
- man-pages: `sched(7)`, `sched_setattr(2)`, `cgroups(7)`, `proc(5)`, `tcp(7)`, `bpf(2)`, `bpf-helpers(7)`

---

*Every concept in this page is a knob you can turn. The math tells you the cost. The code tells you the limit. The kernel does not ask permission — it just runs the equations, every nanosecond, on every CPU, forever.*
