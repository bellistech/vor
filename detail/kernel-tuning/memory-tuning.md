# Linux Memory Tuning — Algorithms, Math, and Performance Analysis

> *Deep dive into the Linux memory subsystem internals: page replacement algorithms, buddy allocator fragmentation, TLB miss costs, NUMA topology modeling, and Transparent Huge Pages performance analysis. Theory and math behind the tunables.*

---

## Prerequisites

- Understanding of virtual memory concepts (pages, page tables, TLB)
- Familiarity with Linux memory tunables (`sysctl vm.*`, `/proc/meminfo`)
- Basic probability and asymptotic analysis

## Complexity

| Topic | Analysis Type | Key Metric |
|:---|:---|:---|
| Page Replacement | Amortized O(1) | Scan rate, refault distance |
| Buddy Allocator | O(log n) split/coalesce | Fragmentation index |
| TLB Misses | Cycle-level | Cycles per miss, miss rate |
| NUMA Access | Latency model | Local vs remote ratio |
| THP | Throughput model | TLB coverage, compaction cost |

---

## 1. Page Replacement — LRU and Active/Inactive Lists

### The Two-List LRU Approximation

Linux does not use a true LRU. It maintains two LRU lists per memory zone per cgroup:

- **Active list**: pages recently accessed (second chance given)
- **Inactive list**: pages candidates for eviction

Each list tracks two page types separately: **anonymous** (heap, stack, mmap private) and **file-backed** (page cache).

The promotion/demotion flow:

```
Fault → Inactive(file/anon)
         ↓ (accessed again)
       Active(file/anon)
         ↓ (pressure / rotation)
       Inactive(file/anon)
         ↓ (not accessed)
       Evict / Writeback
```

### Clock / Second-Chance Approximation

Each page has a **Referenced** bit (PTE accessed bit). The kernel's `shrink_active_list()` scans pages:

$$P(\text{demote}) = \begin{cases} 0 & \text{if Referenced = 1 (clear bit, keep in Active)} \\ 1 & \text{if Referenced = 0 (move to Inactive)} \end{cases}$$

The scan rate is controlled by `vm.vfs_cache_pressure` (for file caches) and the kswapd watermark logic (for all pages).

### Refault Distance — MGLRU Foundation

Multi-Gen LRU (available since Linux 6.1) replaces the two-list model with a generation-based tracker. The key insight is **refault distance**:

$$\text{Refault Distance} = \text{NR\_evicted\_between}(eviction, refault)$$

If the refault distance is less than the size of the inactive list, the page was evicted too eagerly — it should have been kept:

$$D_{refault} < |L_{inactive}| \implies \text{page was useful, promote to active}$$

### Working Set Size Estimation

The kernel estimates working set size (WSS) via refault tracking:

$$WSS \approx |L_{active}| + |\{p \in L_{inactive} : D_{refault}(p) < |L_{inactive}|\}|$$

The balance between active and inactive lists is maintained by:

$$\frac{|L_{active}|}{|L_{active}| + |L_{inactive}|} \approx \frac{\text{refault rate}}{\text{refault rate} + \text{scan rate}}$$

### Scan Cost Analysis

kswapd performs background reclaim. The per-page scan cost:

$$C_{scan} = C_{lock} + C_{PTE\_check} + C_{referenced\_clear} \approx 50\text{-}200 \text{ ns/page}$$

At high memory pressure, direct reclaim stalls the allocating process:

$$T_{stall} = \frac{N_{pages\_needed}}{R_{scan}} \times C_{scan}$$

where $R_{scan}$ is the scan rate in pages/second. Monitor via:

```
/proc/vmstat: pgscan_kswapd, pgscan_direct, pgsteal_kswapd, pgsteal_direct
```

The **scan efficiency** ratio should be high:

$$\eta_{scan} = \frac{\text{pgsteal}}{\text{pgscan}} \quad (\text{target} > 0.5)$$

---

## 2. Buddy Allocator — Math and Fragmentation

### The Buddy System

Linux uses a binary buddy allocator for physical page management. Memory is divided into blocks of order $k$, where each block contains $2^k$ contiguous pages.

$$\text{Block size at order } k = 2^k \times 4\text{KB} = 2^{k+12} \text{ bytes}$$

| Order | Pages | Size |
|:---:|:---:|:---:|
| 0 | 1 | 4 KB |
| 1 | 2 | 8 KB |
| 2 | 4 | 16 KB |
| ... | ... | ... |
| 9 | 512 | 2 MB |
| 10 | 1024 | 4 MB |

The maximum order is `MAX_ORDER - 1 = 10` (default), yielding 4 MB maximum contiguous allocation.

### Allocation and Splitting

To allocate $2^k$ pages, find the smallest available block of order $j \geq k$:

$$\text{splits required} = j - k$$

Each split produces one block of order $j-1$ for the request and one free buddy of order $j-1$:

$$\text{Allocation cost} = O(\log_2(\text{MAX\_ORDER})) = O(1) \quad (\text{bounded constant})$$

### Coalescing (Free)

When freeing a block of order $k$, check if its buddy is free. If so, merge into order $k+1$ and recurse:

$$\text{Buddy address} = \text{block\_addr} \oplus (2^k \times \text{PAGE\_SIZE})$$

The XOR property ensures buddy pairs tile the address space without overlap. Maximum coalesce depth:

$$\text{Max merges} = \text{MAX\_ORDER} - 1 - k$$

### Fragmentation Analysis

**External fragmentation** occurs when free pages exist but not in contiguous blocks. The fragmentation index for order $k$:

$$F_k = 1 - \frac{\text{free blocks of order} \geq k}{\text{total free pages} / 2^k}$$

- $F_k = 0$: no fragmentation at this order (all free memory is in large-enough blocks)
- $F_k \to 1$: severe fragmentation (free pages scattered across small blocks)

From `/proc/buddyinfo`, for zone `Z` with free block counts $b_0, b_1, \ldots, b_{10}$:

$$\text{Total free pages} = \sum_{i=0}^{10} b_i \times 2^i$$

$$\text{Free pages available at order } k = \sum_{i=k}^{10} b_i \times 2^i$$

$$\text{Largest available contiguous} = \max\{2^i : b_i > 0\} \times \text{PAGE\_SIZE}$$

### Compaction Cost Model

The kernel's compaction daemon (`kcompactd`) migrates pages to create contiguous free blocks. The cost per compacted huge page:

$$C_{compact} = N_{migrate} \times (C_{copy} + C_{remap})$$

$$C_{copy} = \frac{2^k \times 4096}{BW_{mem}} \quad (\text{memory copy bandwidth})$$

$$C_{remap} = 2^k \times C_{PTE\_update} + C_{TLB\_flush}$$

For a 2 MB huge page ($k = 9$, 512 pages), with 20 GB/s memory bandwidth:

$$C_{copy} = \frac{512 \times 4096}{20 \times 10^9} \approx 100 \text{ }\mu\text{s}$$

---

## 3. TLB Miss Cost Analysis

### Page Table Walk Cost

On x86-64, a TLB miss requires a 4-level page table walk (5-level with LA57):

$$\text{Levels} = \{PGD, PUD, PMD, PTE\}$$

Each level requires one memory access. With caches:

| Access | Latency (L1 hit) | Latency (L2 hit) | Latency (LLC hit) | Latency (DRAM) |
|:---|:---:|:---:|:---:|:---:|
| Per level | ~1 ns | ~5 ns | ~15 ns | ~80 ns |
| 4-level walk | ~4 ns | ~20 ns | ~60 ns | ~320 ns |

The expected TLB miss cost with page walk caches:

$$E[C_{miss}] = \sum_{i=1}^{4} P(\text{level } i \text{ not cached}) \times C_{mem}(i)$$

In practice, upper page table levels (PGD, PUD) are almost always cached:

$$E[C_{miss}] \approx C_{L1} + C_{L1} + C_{L2} + C_{DRAM} \approx 1 + 1 + 5 + 80 = 87 \text{ ns}$$

### TLB Coverage Model

The TLB covers a limited virtual address range:

$$\text{TLB Coverage} = N_{entries} \times \text{Page Size}$$

| Page Size | Typical dTLB Entries | Coverage |
|:---:|:---:|:---:|
| 4 KB | 64 (L1) + 1536 (L2) | 256 KB + 6 MB |
| 2 MB | 32 (L1) + 1024 (L2) | 64 MB + 2 GB |
| 1 GB | 4 (L1) | 4 GB |

For a workload with working set $W$:

$$P(\text{TLB miss}) \approx \max\left(0, 1 - \frac{\text{TLB Coverage}}{W}\right)$$

### Performance Impact of Page Size

The throughput overhead from TLB misses:

$$\text{Overhead} = \text{Memory Accesses per Second} \times P(\text{miss}) \times E[C_{miss}]$$

Example: A workload doing 100M memory accesses/sec with 8 GB working set:

**With 4 KB pages** (6.25 MB effective TLB coverage):

$$P(\text{miss}) \approx 1 - \frac{6.25 \text{ MB}}{8 \text{ GB}} \approx 0.999$$

$$\text{Overhead} = 10^8 \times 0.999 \times 87 \text{ ns} \approx 8.7 \text{ seconds/sec (impossible, CPU-bound)}$$

This shows the model breaks down for random access across a huge working set. In practice, spatial and temporal locality reduce the effective miss rate dramatically. The realistic miss rate is measured, not computed from coverage alone:

$$\text{Measured miss rate} = \frac{\text{dTLB-load-misses}}{\text{dTLB-load-accesses}} \quad (\text{via } \texttt{perf stat})$$

**With 2 MB pages** (2 GB effective coverage):

$$P(\text{miss}) \approx 1 - \frac{2 \text{ GB}}{8 \text{ GB}} = 0.75$$

Reducing page-walk-induced latency by covering 256x more address space per TLB entry.

### Measuring TLB Impact

```
perf stat -e dTLB-load-misses,dTLB-loads,dTLB-store-misses,dTLB-stores,
           iTLB-load-misses,page-faults ./workload
```

The miss rate ratio between 4 KB and 2 MB pages:

$$\frac{\text{Miss Rate}_{4K}}{\text{Miss Rate}_{2M}} \approx \frac{\text{Page Size}_{2M}}{\text{Page Size}_{4K}} = 512$$

This is an upper bound; real improvement is 2-10x due to TLB entry count trade-offs.

---

## 4. NUMA Distance Matrix and Memory Access Latency

### NUMA Topology Model

A NUMA system has $N$ nodes, each with local memory and CPUs. The **distance matrix** $D$ is an $N \times N$ matrix where $D_{ij}$ represents the relative access cost from node $i$ to node $j$'s memory.

$$D_{ii} = 10 \quad (\text{local access, normalized baseline})$$

$$D_{ij} > 10 \quad \text{for } i \neq j \quad (\text{remote access})$$

Typical 2-socket Intel system:

$$D = \begin{pmatrix} 10 & 21 \\ 21 & 10 \end{pmatrix}$$

4-socket AMD EPYC (each socket has 4 NUMA nodes = 16 total):

$$D = \begin{pmatrix} 10 & 12 & 12 & 12 & 32 & 32 & 32 & 32 \\ 12 & 10 & 12 & 12 & 32 & 32 & 32 & 32 \\ \vdots & & & & & & & \vdots \end{pmatrix}$$

Read from the kernel:

```
cat /sys/devices/system/node/node*/distance
numactl --hardware
```

### Memory Access Latency Model

For a process on node $i$ accessing memory distributed across nodes:

$$E[T_{access}] = \sum_{j=0}^{N-1} P(\text{access node } j) \times T_{local} \times \frac{D_{ij}}{10}$$

Where $T_{local} \approx 80\text{-}100$ ns for DRAM access.

For a 2-node system with fraction $\alpha$ of accesses going to local memory:

$$E[T_{access}] = \alpha \times T_{local} + (1 - \alpha) \times T_{local} \times \frac{D_{remote}}{10}$$

$$E[T_{access}] = T_{local} \left(\alpha + (1 - \alpha) \times \frac{D_{remote}}{10}\right)$$

With $D_{remote} = 21$ and $T_{local} = 80$ ns:

| Local Access Fraction ($\alpha$) | Expected Latency | Slowdown |
|:---:|:---:|:---:|
| 1.0 | 80 ns | 1.00x |
| 0.9 | 88.8 ns | 1.11x |
| 0.7 | 106.4 ns | 1.33x |
| 0.5 | 124.0 ns | 1.55x |
| 0.0 | 168.0 ns | 2.10x |

### Bandwidth Model

Each NUMA node has independent memory channels. Total system bandwidth:

$$BW_{total} = \sum_{i=0}^{N-1} BW_i$$

But cross-node traffic shares the interconnect (UPI/xGMI). If a fraction $f$ of traffic is remote:

$$BW_{effective} = \frac{BW_{local}}{1 + f \times \left(\frac{BW_{local}}{BW_{interconnect}} - 1\right)}$$

For Intel UPI with 20.8 GT/s per link (2 bytes/transfer):

$$BW_{UPI} \approx 41.6 \text{ GB/s per link}$$

If local memory bandwidth is 80 GB/s per socket, the interconnect becomes the bottleneck when:

$$f \times N_{accesses/s} \times 64 \text{ bytes} > BW_{UPI}$$

### Optimal Memory Policy Selection

Given workload characteristics:

| Access Pattern | Optimal Policy | Rationale |
|:---|:---|:---|
| Thread-local buffers | `membind` to local node | Minimize latency |
| Shared hash tables | `interleave` across all | Balance bandwidth |
| Sequential streaming | `membind` to local node | Maximize local BW |
| Random across large set | `interleave` across all | Distribute misses |
| Read-mostly shared | `preferred` to one node | One copy, tolerate remote reads |

The interleave policy distributes pages round-robin, giving:

$$E[T_{access}^{interleave}] = T_{local} \times \frac{1}{N} \sum_{j=0}^{N-1} \frac{D_{ij}}{10}$$

For a 2-node system: $E[T] = T_{local} \times \frac{10 + 21}{2 \times 10} = 1.55 \times T_{local}$

---

## 5. Transparent Huge Pages — Performance Impact Analysis

### THP Allocation Flow

When THP is enabled (`always` or `madvise`), the kernel attempts to allocate 2 MB pages for anonymous memory:

```
Page Fault → Check alignment (2MB boundary)
            → Try allocate order-9 compound page
            → If fail: try compaction
            → If compaction fails: fallback to 4KB pages
```

### Benefits Model

THP benefits come from three sources:

**1. Reduced TLB misses** (see Section 3):

$$\Delta_{TLB} = (\text{Miss Rate}_{4K} - \text{Miss Rate}_{2M}) \times E[C_{miss}] \times \text{Access Rate}$$

**2. Reduced page table memory**:

$$\text{PTE overhead}_{4K} = \frac{W}{4 \text{ KB}} \times 8 \text{ bytes} = \frac{W}{512}$$

$$\text{PTE overhead}_{2M} = \frac{W}{2 \text{ MB}} \times 8 \text{ bytes} = \frac{W}{256 \text{ K}}$$

$$\text{Savings} = \frac{W}{512} - \frac{W}{256\text{K}} \approx \frac{W}{512} \quad (\text{for large } W)$$

For $W = 64$ GB: savings $= 128$ MB of page table entries.

**3. Reduced page fault count**:

$$\text{Faults}_{4K} = \frac{W}{4 \text{ KB}}, \quad \text{Faults}_{2M} = \frac{W}{2 \text{ MB}}$$

$$\text{Ratio} = \frac{1}{512} \quad (\text{512x fewer faults})$$

### Cost Model — Compaction and Latency Spikes

THP has three major costs:

**1. Allocation latency (compaction stalls)**:

When no order-9 blocks are available, synchronous compaction runs:

$$T_{compact} = N_{pages\_migrated} \times (C_{copy} + C_{remap} + C_{TLB\_flush})$$

Worst case: migrate 512 pages to form one huge page:

$$T_{compact}^{worst} = 512 \times (200\text{ ns} + 100\text{ ns} + 1\text{ }\mu\text{s}) \approx 660\text{ }\mu\text{s}$$

This is a tail latency event. The p99 allocation latency with THP enabled:

$$T_{p99}^{alloc} = \begin{cases} \sim 1\text{ }\mu\text{s} & \text{with THP disabled} \\ \sim 1\text{ ms} & \text{with THP enabled (compaction)} \\ \sim 10\text{ ms} & \text{with THP enabled (under fragmentation)} \end{cases}$$

**2. Memory waste (internal fragmentation)**:

A 2 MB page allocated for a region using only $k$ KB wastes $2048 - k$ KB:

$$E[\text{waste per THP}] = \frac{2 \text{ MB}}{2} = 1 \text{ MB} \quad (\text{if usage is uniform random within page})$$

For applications with many small allocations, THP can increase RSS by 10-30%.

**3. khugepaged CPU cost**:

The background thread `khugepaged` scans for collapse opportunities:

$$\text{CPU overhead} = R_{scan} \times C_{scan\_per\_page} \approx 1\text{-}5\% \text{ of one core}$$

### When THP Helps vs Hurts

Quantitative decision framework:

$$\text{Net benefit} = \Delta_{TLB\_savings} - C_{compaction\_stalls} - C_{memory\_waste} - C_{khugepaged}$$

| Workload | TLB Benefit | Compaction Cost | Verdict |
|:---|:---:|:---:|:---|
| Large working set, sequential | High | Low | THP ON (`always`) |
| Large working set, random | Very high | Medium | THP ON (`madvise`) |
| Many small allocations | Low | High | THP OFF (`never`) |
| Redis, MongoDB | Low (small keys) | High (fragmentation) | THP OFF (`never`) |
| JVM with large heap | High | Low (preallocated) | THP ON or explicit hugepages |
| DPDK / packet processing | Very high | N/A | Explicit 1GB hugepages |
| HPC / scientific computing | High | Low | Explicit 2MB hugepages |

### The Database THP Problem

Databases like Redis, PostgreSQL, and MongoDB use `fork()` for persistence (BGSAVE, `pg_dump`). With THP:

$$\text{CoW cost per THP} = 2 \text{ MB copy} \quad (\text{vs 4 KB with regular pages})$$

If the database modifies one byte in a 2 MB THP after fork:

$$\text{Amplification factor} = \frac{2 \text{ MB}}{4 \text{ KB}} = 512\times$$

This causes massive memory spikes during `BGSAVE`:

$$\text{Extra memory} = N_{modified\_THPs} \times 2 \text{ MB}$$

This is why every database tuning guide says to disable THP.

---

## References

- Gorman, M. "Understanding the Linux Virtual Memory Manager" (2004)
- Corbet, J. "Multi-Gen LRU" — LWN.net series (2022)
- Intel 64 and IA-32 Architectures Optimization Reference Manual, Chapter 7: TLB
- Lameter, C. "NUMA in Linux" — Linux Plumbers Conference (2013)
- Arcangeli, A. "Transparent Huge Pages" — KVM Forum (2010)
- Linux kernel source: `mm/vmscan.c`, `mm/page_alloc.c`, `mm/huge_memory.c`, `mm/compaction.c`
- Brendan Gregg, "Systems Performance", 2nd Edition, Chapters 7-8
- `/proc/vmstat`, `/proc/buddyinfo`, `/proc/pagetypeinfo` — kernel documentation
