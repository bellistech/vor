# The Mathematics of tmpfs -- Memory-Backed Performance and Page Allocation

> *tmpfs stores files in the kernel's page cache backed by RAM and swap. The mathematics cover the performance gap between RAM and disk I/O, page allocation and accounting, memory pressure interaction with the swap subsystem, and the size-accounting model that makes tmpfs both elastic and bounded.*

---

## 1. RAM vs Storage Performance (IOPS and Throughput)

### Latency Hierarchy

The fundamental advantage of tmpfs is memory access speed vs storage access speed:

$$\text{Speedup}_{\text{tmpfs}} = \frac{T_{\text{storage}}}{T_{\text{RAM}}}$$

| Medium | Random Read Latency | Sequential Read (1 MB) | Random 4K IOPS |
|:---|:---:|:---:|:---:|
| DDR4 RAM | 80 ns | 0.04 ms | 10,000,000+ |
| NVMe SSD | 20 us | 0.15 ms | 500,000 |
| SATA SSD | 100 us | 0.30 ms | 100,000 |
| HDD (7200 RPM) | 8 ms | 1.00 ms | 150 |
| **RAM/NVMe ratio** | **250x** | **3.75x** | **20x** |
| **RAM/HDD ratio** | **100,000x** | **25x** | **66,667x** |

### Throughput Model

tmpfs throughput is bounded by memory bandwidth, not I/O subsystem:

$$B_{\text{tmpfs}} = B_{\text{memcpy}} = \frac{\text{data size}}{\text{memory latency per cacheline}} \times \text{channels}$$

Typical DDR4 dual-channel:

$$B_{\text{tmpfs}} \approx 25\text{-}50 \text{ GB/s}$$

Compared to storage:

| Operation | tmpfs | NVMe | SATA SSD | HDD |
|:---|:---:|:---:|:---:|:---:|
| Sequential read | 25 GB/s | 3.5 GB/s | 550 MB/s | 150 MB/s |
| Sequential write | 15 GB/s | 3.0 GB/s | 500 MB/s | 120 MB/s |
| Random 4K read | 40 GB/s | 2.0 GB/s | 400 MB/s | 0.6 MB/s |

---

## 2. Page Allocation Model (Demand Paging)

### Lazy Allocation

tmpfs uses demand paging -- pages are allocated only when data is written:

$$\text{Allocated Pages}(t) = |\{p : p \text{ was written to before time } t\}|$$

$$\text{Memory Used} = \text{Allocated Pages} \times \text{PAGE\_SIZE}$$

where PAGE_SIZE = 4096 bytes on most architectures.

### Allocation vs Declared Size

The declared size (`mount -o size=2G`) sets an upper bound, not an allocation:

$$\text{Actual Memory} = \sum_{f \in \text{files}} \lceil \frac{|f|}{\text{PAGE\_SIZE}} \rceil \times \text{PAGE\_SIZE}$$

$$0 \leq \text{Actual Memory} \leq \text{Declared Size} \leq \text{RAM} + \text{Swap}$$

### Worked Example

A 2 GB tmpfs with various file populations:

| Files | Total Data | Pages Allocated | Actual RAM | Utilization |
|:---:|:---:|:---:|:---:|:---:|
| Empty | 0 | 0 | 0 | 0% |
| 1 x 100 MB | 100 MB | 25,600 | 100 MB | 5% |
| 1000 x 1 KB | 1 MB | 1,000 | 3.9 MB | 0.2% |
| 500 x 1 MB | 500 MB | 128,000 | 500 MB | 25% |
| Full | 2 GB | 524,288 | 2 GB | 100% |

Note: 1000 files of 1 KB each consume 3.9 MB (not 1 MB) due to page granularity and inode/dentry overhead.

### Internal Fragmentation

Each file wastes on average half a page at the end:

$$\text{Waste per file} = \frac{\text{PAGE\_SIZE}}{2} = 2048 \text{ bytes}$$

$$\text{Total Waste} = N_{\text{files}} \times 2048$$

For 100,000 small files:

$$\text{Waste} = 100{,}000 \times 2048 = 195 \text{ MB}$$

---

## 3. Memory Accounting (Shmem Tracking)

### Where tmpfs Appears in meminfo

tmpfs memory is tracked as `Shmem` in `/proc/meminfo`:

$$\text{Shmem} = \text{tmpfs pages} + \text{POSIX shared memory} + \text{shared anonymous pages}$$

### Memory Categories

| Category | Counts Against | Visible In | Swappable |
|:---|:---|:---|:---:|
| tmpfs pages | `Shmem` | `df`, meminfo | Yes |
| Page cache (ext4) | `Cached` | meminfo | Yes (clean) |
| Anonymous pages | `AnonPages` | meminfo, proc | Yes |
| Kernel slab | `Slab` | meminfo | No |

### Available Memory Calculation

$$\text{MemAvailable} = \text{MemFree} + \text{Reclaimable Cache} + \text{Reclaimable Slab}$$

tmpfs pages are **not** included in reclaimable memory (they contain user data that cannot be regenerated from disk).

### OOM Interaction

tmpfs memory contributes to memory pressure but is not attributed to any process:

$$\text{Process RSS} + \text{tmpfs Shmem} + \text{Kernel} \leq \text{RAM}$$

When the OOM killer activates, it kills processes by RSS -- but tmpfs data survives OOM kills. A full tmpfs can cause OOM events that kill unrelated processes.

---

## 4. Swap Interaction (Page Reclaim)

### Swap-Out Model

Under memory pressure, the kernel can swap tmpfs pages to disk:

$$\text{tmpfs page lifecycle}: \text{RAM} \xrightarrow{\text{swap out}} \text{Swap} \xrightarrow{\text{swap in}} \text{RAM}$$

### Swap Probability

The probability of a tmpfs page being swapped out depends on its recency of access:

$$P(\text{swap out}) \propto \frac{1}{\text{recency}} \times \text{vm.swappiness}$$

With `vm.swappiness = 60` (default):

| Memory Pressure | $P(\text{tmpfs page swapped})$ | Performance Impact |
|:---:|:---:|:---|
| Low (< 70% used) | ~0% | None |
| Medium (70-85%) | ~5% | Occasional disk I/O |
| High (85-95%) | ~30% | Significant latency spikes |
| Critical (> 95%) | ~80% | tmpfs effectively becomes disk-backed |

### Performance Degradation on Swap

When tmpfs pages are swapped, access time degrades:

$$T_{\text{access}} = (1 - p_s) \times T_{\text{RAM}} + p_s \times T_{\text{swap}}$$

where $p_s$ is the swap probability.

| Swap Rate ($p_s$) | $T_{\text{access}}$ (NVMe swap) | Slowdown vs Pure RAM |
|:---:|:---:|:---:|
| 0% | 80 ns | 1x |
| 1% | 280 ns | 3.5x |
| 5% | 1.08 us | 13.5x |
| 20% | 4.08 us | 51x |
| 50% | 10.08 us | 126x |

### tmpfs vs ramfs

| Property | tmpfs | ramfs |
|:---|:---:|:---:|
| Size limit | Configurable | None (dangerous) |
| Swappable | Yes | No |
| Shows in `df` | Yes | No |
| Risk | Bounded | Can consume all RAM |
| Memory pressure | Survives via swap | Causes OOM |

---

## 5. Inode Accounting (Metadata Overhead)

### Inode Limit

tmpfs limits the number of files via `nr_inodes`:

$$\text{Default nr\_inodes} = \frac{\text{total pages}}{2} = \frac{\text{RAM}}{2 \times \text{PAGE\_SIZE}}$$

For 16 GB RAM:

$$\text{Default nr\_inodes} = \frac{16 \times 2^{30}}{2 \times 4096} = 2{,}097{,}152 \text{ inodes}$$

### Per-Inode Memory Cost

Each tmpfs inode consumes kernel memory for metadata:

$$\text{Inode size}_{\text{kernel}} \approx 600 \text{ bytes (struct inode + dentry)}$$

$$\text{Metadata overhead} = N_{\text{files}} \times 600 \text{ bytes}$$

For 1 million files:

$$\text{Metadata} = 1{,}000{,}000 \times 600 = 572 \text{ MB}$$

This metadata is kernel slab memory and is **not** swappable.

---

## 6. Build Performance Analysis (Real Workload)

### Compilation I/O Profile

A typical compilation consists of:

$$T_{\text{build}} = T_{\text{read\_source}} + T_{\text{compile}} + T_{\text{write\_objects}} + T_{\text{link}}$$

The I/O component:

$$T_{\text{I/O}} = T_{\text{read\_source}} + T_{\text{write\_objects}}$$

### tmpfs Build Speedup

| Build System | Files | I/O Data | Disk Time | tmpfs Time | Speedup |
|:---|:---:|:---:|:---:|:---:|:---:|
| Go (medium project) | 500 | 50 MB | 2.5s | 0.1s | 25x I/O |
| Rust (cargo build) | 2000 | 500 MB | 15s | 0.5s | 30x I/O |
| C++ (cmake) | 5000 | 200 MB | 8s | 0.3s | 27x I/O |
| Node.js (npm install) | 50000 | 300 MB | 25s | 1.0s | 25x I/O |

Overall build speedup (including CPU-bound compilation):

$$\text{Speedup}_{\text{total}} = \frac{T_{\text{CPU}} + T_{\text{I/O\_disk}}}{T_{\text{CPU}} + T_{\text{I/O\_tmpfs}}}$$

For a Go build with 60% CPU, 40% I/O:

$$\text{Speedup} = \frac{0.60 + 0.40}{0.60 + 0.40/25} = \frac{1.0}{0.616} = 1.62\times$$

### Amdahl's Law Application

$$\text{Speedup} = \frac{1}{(1 - f) + f/s}$$

where $f$ is the I/O fraction and $s$ is the tmpfs speedup for I/O.

| I/O Fraction ($f$) | I/O Speedup ($s$) | Overall Speedup |
|:---:|:---:|:---:|
| 10% | 25x | 1.10x |
| 20% | 25x | 1.24x |
| 40% | 25x | 1.62x |
| 60% | 25x | 2.17x |
| 80% | 25x | 3.57x |

---

## 7. Size Planning (Capacity Model)

### Right-Sizing Formula

$$\text{Size}_{\text{tmpfs}} = \text{Peak Usage} \times (1 + \text{Safety Margin})$$

$$\text{Size}_{\text{tmpfs}} \leq \text{RAM} \times \text{Max Fraction}$$

### System-Wide tmpfs Budget

$$\text{RAM}_{\text{available}} = \text{RAM}_{\text{total}} - \text{RAM}_{\text{kernel}} - \text{RAM}_{\text{apps}} - \text{RAM}_{\text{page cache}}$$

| System RAM | Kernel | Applications | Page Cache | Available for tmpfs |
|:---:|:---:|:---:|:---:|:---:|
| 8 GB | 0.5 GB | 4 GB | 1.5 GB | 2 GB |
| 16 GB | 0.5 GB | 8 GB | 3 GB | 4.5 GB |
| 32 GB | 1 GB | 16 GB | 5 GB | 10 GB |
| 64 GB | 1 GB | 24 GB | 8 GB | 31 GB |

### Multiple tmpfs Mounts

Total declared size across all tmpfs mounts can exceed RAM (overcommit is allowed):

$$\sum_{i} \text{size}(T_i) > \text{RAM} + \text{Swap} \quad \text{(allowed but risky)}$$

But actual usage must fit:

$$\sum_{i} \text{used}(T_i) \leq \text{RAM} + \text{Swap}$$

---

## Prerequisites

- Memory hierarchy (RAM, cache, swap, storage)
- Virtual memory and demand paging
- Amdahl's law (speedup with partial optimization)
- Linux memory management (page cache, slab allocator, OOM)

## Complexity

| Operation | Time | Space |
|:---|:---:|:---:|
| File create | $O(1)$ | $O(1)$ inode |
| File write ($n$ bytes) | $O(n / \text{PAGE\_SIZE})$ | $O(n)$ pages |
| File read ($n$ bytes) | $O(n / \text{PAGE\_SIZE})$ | $O(1)$ |
| File delete | $O(\text{pages})$ | Frees $O(n)$ |
| Mount (empty) | $O(1)$ | $O(1)$ superblock |
| Remount (resize) | $O(1)$ | $O(1)$ |
| Swap out (per page) | $O(1)$ + disk I/O | $O(1)$ swap slot |

---

*The mathematics of tmpfs reduce to the memory hierarchy gap. RAM is 250x faster than NVMe for random access and 100,000x faster than HDD. By placing files in the page cache with no disk backing, tmpfs eliminates the storage bottleneck entirely. The tradeoff is capacity (RAM is 10-100x more expensive per GB than SSD) and durability (contents vanish on reboot). The key insight is that tmpfs memory is elastic -- it costs zero when empty and grows only on demand -- making it ideal for transient, performance-critical workloads like builds, tests, and caches.*
