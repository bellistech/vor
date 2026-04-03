# The Mathematics of Btrfs — B-tree Filesystem Internals

> *Btrfs is a copy-on-write filesystem built on B-trees. The math covers chunk allocation, RAID profiles, subvolume snapshots, balance operations, and space accounting with metadata overhead.*

---

## 1. B-tree Structure — The Core Data Structure

### The Model

Btrfs stores everything in B-trees (specifically B+ tree variants). Each tree node is a fixed-size block (default 16 KiB, called `nodesize`).

### Tree Depth Formula

$$\text{Depth} = \lceil \log_f(N) \rceil$$

Where:
- $f$ = fan-out (keys per node) $\approx \frac{\text{nodesize}}{\text{key size + pointer size}}$
- $N$ = total items in the tree

For a 16 KiB node with ~60-byte items:

$$f \approx \frac{16384}{60} \approx 273$$

| Items ($N$) | Depth | Nodes at Depth |
|:---:|:---:|:---:|
| 100 | 1 | 1 |
| 10,000 | 2 | 37 |
| 1,000,000 | 3 | ~13,400 |
| 100,000,000 | 4 | ~1,340,000 |
| 10,000,000,000 | 5 | ~134,000,000 |

**Lookup cost:** $O(\log_f N)$ — 5 levels handles 10 billion items.

---

## 2. Chunk Allocation and RAID Profiles

### The Model

Btrfs allocates space in **chunks** (default 1 GiB for data, 256 MiB for metadata). Each chunk has a RAID profile.

### RAID Capacity Formulas

$$\text{Usable (raid0)} = \sum_{i=1}^{n} D_i$$

$$\text{Usable (raid1)} = \frac{\min(D_1, D_2, \ldots, D_n)}{1} \times \lfloor n/2 \rfloor$$

$$\text{Usable (raid1c3)} = \frac{\sum D_i}{3}$$

$$\text{Usable (raid5)} = \sum D_i - \max(D_i) \quad \text{(approx, for equal disks: } (n-1) \times D\text{)}$$

$$\text{Usable (raid6)} = \sum D_i - 2 \times \max(D_i) \quad \text{(approx: } (n-2) \times D\text{)}$$

$$\text{Usable (raid10)} = \frac{\sum D_i}{2}$$

### Worked Examples — Mixed Device Sizes

*"3 disks: 4 TiB, 4 TiB, 8 TiB, raid1 profile."*

Btrfs raid1 mirrors in pairs of chunks. With uneven devices:

$$\text{Usable} \approx \frac{4 + 4 + 8}{2} = 8 \text{ TiB}$$

| Profile | Disks | Each Size | Raw Total | Usable | Overhead |
|:---|:---:|:---:|:---:|:---:|:---:|
| single | 4 | 4 TiB | 16 TiB | 16 TiB | 0% |
| raid0 | 4 | 4 TiB | 16 TiB | 16 TiB | 0% |
| raid1 | 4 | 4 TiB | 16 TiB | 8 TiB | 50% |
| raid1c3 | 4 | 4 TiB | 16 TiB | 5.3 TiB | 67% |
| raid5 | 4 | 4 TiB | 16 TiB | 12 TiB | 25% |
| raid6 | 4 | 4 TiB | 16 TiB | 8 TiB | 50% |
| raid10 | 4 | 4 TiB | 16 TiB | 8 TiB | 50% |

---

## 3. Metadata Space Accounting

### The Model

Btrfs reserves space for metadata separately from data. The metadata ratio depends on file sizes and count.

### Metadata per File

$$\text{Metadata per file} \approx 300 \text{ bytes (inode)} + 60 \text{ bytes (dir entry)} + 80 \text{ bytes (extent ref)}$$

$$\text{Total Metadata} \approx \text{File Count} \times 440 \text{ bytes} + \text{Tree Internal Nodes}$$

### Metadata Reservation

Btrfs reserves metadata chunks. Default global reserve:

$$\text{Metadata Reserve} = \max(256 \text{ MiB}, \text{Estimated Metadata Need})$$

### Worked Example

*"1 million files, average 100 KiB each."*

$$\text{Data} = 10^6 \times 100 \text{ KiB} = 95.4 \text{ GiB}$$

$$\text{Metadata} \approx 10^6 \times 440 = 419 \text{ MiB}$$

$$\text{Metadata Ratio} = \frac{419 \text{ MiB}}{95.4 \text{ GiB}} = 0.43\%$$

| File Count | Avg Size | Data Size | Metadata | Ratio |
|:---:|:---:|:---:|:---:|:---:|
| 10,000 | 1 MiB | 9.8 GiB | 4.2 MiB | 0.04% |
| 1,000,000 | 100 KiB | 95.4 GiB | 419 MiB | 0.43% |
| 10,000,000 | 4 KiB | 38.1 GiB | 4.1 GiB | 10.7% |
| 100,000,000 | 4 KiB | 381 GiB | 41 GiB | 10.7% |

**Key insight:** Many small files dramatically increase metadata overhead.

---

## 4. Snapshot Space — COW Efficiency

### The Model

Btrfs snapshots are instant (COW). Space is consumed only when original blocks are modified.

### Snapshot Space Formula

$$\text{Snapshot Exclusive} = \text{Modified Blocks Since Snapshot} \times \text{Block Size}$$

$$\text{Shared Data} = \text{Total Data} - \text{Snapshot Exclusive}$$

### Clone/Reflink Space

$$\text{Physical Storage} = \text{Unique Data} + \text{Metadata for Refs}$$

$$\text{Space Savings} = 1 - \frac{\text{Physical}}{\text{Logical}}$$

### Worked Example

*"50 GiB subvolume, snapshot taken, then 10% modified over a week."*

$$\text{Snapshot Exclusive} = 50 \times 0.10 = 5 \text{ GiB}$$

$$\text{Total Space Used} = 50 + 5 = 55 \text{ GiB (not 100 GiB)}$$

| Snapshots | Modification Rate | Original Size | Actual Storage | vs Naive Copy |
|:---:|:---:|:---:|:---:|:---:|
| 1 | 5% | 50 GiB | 52.5 GiB | 100 GiB |
| 7 (daily) | 2%/day | 50 GiB | 57.0 GiB | 400 GiB |
| 30 (daily) | 2%/day | 50 GiB | 72.6 GiB | 1550 GiB |
| 365 (daily) | 1%/day | 50 GiB | 206 GiB | 18,300 GiB |

---

## 5. Balance Operation Cost

### The Model

`btrfs balance` redistributes data across devices. It reads and rewrites chunks to rebalance allocations.

### Balance Time

$$T_{balance} = \frac{\text{Chunks to Relocate} \times \text{Chunk Size}}{\text{I/O Bandwidth}}$$

### Worked Example

*"4 TiB filesystem, 80% full, adding a new device. 500 chunks of 1 GiB each need relocating."*

$$\text{Data to Move} = 500 \times 1 \text{ GiB} = 500 \text{ GiB}$$

At 150 MB/s effective I/O (read + rewrite):

$$T = \frac{500 \times 1024}{150} = 3,413 \text{ sec} \approx 57 \text{ min}$$

**Note:** Balance is I/O intensive — it reads and writes every relocated chunk. During balance, filesystem performance degrades by approximately:

$$\text{Degradation} \approx \frac{\text{Balance I/O}}{\text{Balance I/O} + \text{Workload I/O}} \times 100\%$$

---

## 6. Compression and Inline Extents

### Compression Ratio Impact

$$\text{Effective Capacity} = \text{Raw Size} \times \text{Compression Ratio}$$

| Algorithm | Typical Ratio | CPU Overhead | Btrfs Option |
|:---|:---:|:---:|:---|
| zlib (level 3) | 2.0-3.5x | Medium | `compress=zlib` |
| lzo | 1.5-2.5x | Low | `compress=lzo` |
| zstd (level 3) | 2.0-4.0x | Low-medium | `compress=zstd` |
| zstd (level 15) | 2.5-5.0x | High | `compress=zstd:15` |

### Inline Extents

Files smaller than `nodesize - header` (~3.8 KiB with 4K nodesize, ~15.8 KiB with 16K nodesize) are stored **inline** in the metadata B-tree:

$$\text{Inline Threshold} = \text{nodesize} - 200 \text{ bytes (header)}$$

$$\text{Space Saved per Tiny File} = 4 \text{ KiB (one block)} - \text{File Size}$$

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| $\lceil \log_f(N) \rceil$ | Logarithmic | B-tree depth |
| $(n - p) \times D$ | Linear arithmetic | RAID capacity |
| $\text{files} \times 440$ | Linear scaling | Metadata sizing |
| $\text{Size} \times \text{Change Rate}$ | Rate / linear | Snapshot cost |
| $\frac{\text{Data}}{\text{BW}}$ | Rate equation | Balance time |
| $\text{Raw} \times \text{Ratio}$ | Multiplication | Compression capacity |

---

*Every `btrfs subvolume snapshot`, `btrfs balance`, and `btrfs scrub` is walking these B-trees — a COW filesystem where nothing is ever overwritten in place.*
