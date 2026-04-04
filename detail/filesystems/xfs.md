# The Mathematics of XFS — High-Performance Filesystem Internals

> *XFS was designed at SGI for extreme scalability. The math covers Allocation Group parallelism, B+ tree structures, real-time I/O guarantees, delayed allocation, and the log (journal) sizing model.*

---

## 1. Allocation Groups — Parallel Allocation

### The Model

XFS divides the filesystem into **Allocation Groups (AGs)**, each independently managing its own free space, inodes, and B+ trees. This enables parallel allocation by multiple threads.

### AG Sizing

$$\text{AG Count} = \lceil \frac{\text{FS Size}}{\text{AG Size}} \rceil$$

$$\text{Default AG Size} \approx \frac{\text{FS Size}}{4} \quad (\text{but capped at 1 TiB})$$

$$\text{AG Count Range:} \quad 4 \leq \text{AGs} \leq 2^{32}$$

| FS Size | AG Size | AG Count | Parallelism |
|:---:|:---:|:---:|:---:|
| 100 GiB | 25 GiB | 4 | 4-way |
| 1 TiB | 256 GiB | 4 | 4-way |
| 10 TiB | 1 TiB | 10 | 10-way |
| 100 TiB | 1 TiB | 100 | 100-way |
| 1 PiB | 1 TiB | 1,024 | 1,024-way |

### Parallel Throughput

$$\text{Aggregate IOPS} = \min(n_{AGs}, n_{threads}) \times \text{IOPS per AG}$$

$$\text{Aggregate BW} = \min(n_{AGs}, n_{threads}) \times \text{BW per AG}$$

Where each AG operates independently with its own locks:

| Threads | AGs | Effective Parallelism | IOPS Scaling |
|:---:|:---:|:---:|:---:|
| 1 | 4 | 1 | 1x |
| 4 | 4 | 4 | ~4x |
| 16 | 4 | 4 | ~4x (AG lock contention) |
| 16 | 16 | 16 | ~16x |
| 64 | 100 | 64 | ~64x |

---

## 2. B+ Tree Structures — XFS's Core Data Structure

### The Model

XFS uses B+ trees for everything: free space, inodes, extents, directory entries. Each tree node is one filesystem block.

### B+ Tree Fan-out

$$f = \frac{\text{Block Size} - \text{Header}}{\text{Key + Pointer Size}}$$

For free space B+ tree (4 KiB blocks):

$$f = \frac{4096 - 16}{16} = 255 \text{ keys per node}$$

### Tree Depth vs Items

$$\text{Depth} = \lceil \log_f(N) \rceil$$

| Items ($N$) | Fan-out 255 | Depth | Nodes |
|:---:|:---:|:---:|:---:|
| 100 | 255 | 1 | 1 |
| 10,000 | 255 | 2 | ~40 |
| 1,000,000 | 255 | 3 | ~15,600 |
| 100,000,000 | 255 | 4 | ~1,540,000 |

### Lookup Performance

$$T_{lookup} = \text{Depth} \times T_{block\_read}$$

On NVMe ($T_{block} \approx 0.01$ ms): 4-level lookup = 0.04 ms.

---

## 3. Extent Map — The B+ Tree of Data Extents

### Extent Record (16 bytes each)

$$\text{Extent} = (\text{logical\_offset}, \text{physical\_block}, \text{block\_count}, \text{flags})$$

$$\text{Max blocks per extent} = 2^{21} = 2,097,152$$

$$\text{Max extent size (4K blocks)} = 2,097,152 \times 4 \text{ KiB} = 8 \text{ GiB}$$

### Extents per Inode

XFS inodes can hold extents directly (data fork) or use a B+ tree:

$$\text{Inline extents} = \frac{\text{Inode Size} - \text{Core (96 bytes)}}{16}$$

| Inode Size | Inline Extents | Before B+ Tree |
|:---:|:---:|:---|
| 256 bytes | 10 | 80 GiB contiguous |
| 512 bytes | 26 | 208 GiB contiguous |

### B+ Tree Extent Capacity

$$\text{Extents in B+ tree} = f^{d} \times \text{leaf entries}$$

With 4K blocks: $\approx 255^d \times 254$ leaf entries per leaf.

| Depth | Max Extents | Max File Size (8 GiB extents) |
|:---:|:---:|:---:|
| 1 | 254 | 2 TiB |
| 2 | 64,770 | 506 TiB |
| 3 | 16,516,350 | 126 PiB |

---

## 4. Log (Journal) Sizing and Performance

### The Model

XFS's log records metadata changes. Log size directly impacts performance — a too-small log causes stalls.

### Log Size Formula

$$\text{Min Log Size} = \max(512 \text{ blocks}, \frac{\text{FS Size}}{2048})$$

$$\text{Recommended Log} = 32 \text{ MiB} - 2 \text{ GiB}$$

### Log Write Rate

$$\text{Log BW} = \text{Metadata Change Rate} \times \text{Log Overhead Factor}$$

$$\text{Log Overhead Factor} \approx 1.5 - 3.0\times \text{ metadata size (headers, padding)}$$

### Log Wraparound Time

$$T_{wrap} = \frac{\text{Log Size}}{\text{Log Write Rate}}$$

| Log Size | Metadata Rate | Wraparound Time | Performance Impact |
|:---:|:---:|:---:|:---|
| 10 MiB | 1 MB/s | 10 sec | Frequent stalls |
| 32 MiB | 1 MB/s | 32 sec | Some stalls |
| 128 MiB | 1 MB/s | 128 sec | Good |
| 512 MiB | 1 MB/s | 512 sec | Excellent |
| 2 GiB | 1 MB/s | 2048 sec | Overkill for most |

### External Log Optimization

Placing the log on a separate fast device (NVMe):

$$T_{metadata\_op} = T_{log\_write} + T_{data\_write}$$

| Log Device | $T_{log\_write}$ | Metadata IOPS |
|:---|:---:|:---:|
| Same HDD | 5-10 ms | 100-200 |
| Same SSD | 0.1 ms | 10,000 |
| External NVMe | 0.02 ms | 50,000 |

---

## 5. Delayed Allocation and Speculative Preallocation

### The Model

Like ext4, XFS delays block allocation until writeback. Additionally, XFS speculatively preallocates beyond the current file size.

### Speculative Preallocation Formula

$$\text{Prealloc Size} = \min(2^{\lceil \log_2(\text{File Size}) \rceil}, \text{Free Space}/4, 64 \text{ GiB})$$

The preallocation doubles each time the file grows past its allocation:

| File Size | Preallocated | Total Reserved |
|:---:|:---:|:---:|
| 4 KiB | 64 KiB | 68 KiB |
| 64 KiB | 128 KiB | 192 KiB |
| 1 MiB | 2 MiB | 3 MiB |
| 100 MiB | 128 MiB | 228 MiB |
| 1 GiB | 2 GiB | 3 GiB |
| 10 GiB | 16 GiB | 26 GiB |

**This is why `df` shows less free space than expected during writes — XFS has reserved blocks speculatively.**

### Preallocation Trimming

When a file is closed, unused preallocation is released:

$$\text{Released} = \text{Preallocated} - \text{Actual Size}$$

---

## 6. XFS Capacity Limits

### Maximum Sizes

$$\text{Max FS Size} = 2^{63} \text{ blocks} \times \text{Block Size}$$

| Block Size | Max FS Size | Max File Size |
|:---:|:---:|:---:|
| 512 bytes | 4 PiB | 8 EiB |
| 1 KiB | 8 PiB | 8 EiB |
| 4 KiB (default) | 16 EiB | 8 EiB |

### Directory Scaling

| Dir Entries | Storage Mode | Lookup Time |
|:---:|:---|:---:|
| 1-4 | Inline (in inode) | O(n) |
| 5-~100 | Block format (single block) | O(n) |
| ~100+ | Leaf format (B+ tree) | O(log n) |
| ~1M+ | Node format (multi-level B+ tree) | O(log n) |

### Inode Allocation

XFS allocates inodes dynamically (unlike ext4 which fixes them at mkfs time):

$$\text{Max Inodes} = \frac{\text{FS Size}}{256 \text{ bytes}} \quad (\text{theoretical, if entire FS were inodes})$$

$$\text{Practical Limit} = \frac{\text{Free Space}}{\text{Inode Size}} \quad (\text{grows as needed})$$

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| $\frac{\text{FS Size}}{\text{AG Size}}$ | Division | Allocation Group count |
| $\lceil \log_f(N) \rceil$ | Logarithmic | B+ tree depth |
| $2^{21} \times 4\text{K}$ | Exponential | Max extent size |
| $\frac{\text{Log Size}}{\text{Write Rate}}$ | Rate equation | Log wraparound time |
| $2^{\lceil \log_2(S) \rceil}$ | Power of 2 rounding | Speculative prealloc |
| $\min(n_{AG}, n_{threads})$ | Min function | Parallel scaling |

---

*Every `xfs_info`, `xfs_repair`, and `xfs_bmap` command exposes these B+ tree structures — a filesystem designed at SGI in 1993 for IRIX supercomputers, now the default for RHEL and handling petabyte-scale deployments.*

## Prerequisites

- B+ tree data structure concepts
- Allocation group parallelism and concurrency
- Journaling and write-ahead logging
- Block device and partition management

## Complexity

- **Beginner:** Filesystem creation, mounting, basic xfs_info
- **Intermediate:** AG sizing, log tuning, realtime volume configuration
- **Advanced:** B+ tree fan-out modeling, delayed logging internals, reflink COW mechanics
