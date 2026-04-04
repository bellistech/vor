# The Mathematics of ext4 — Fourth Extended Filesystem Internals

> *ext4 is Linux's default filesystem. The math covers inode structure, extent trees, block group allocation, journal modes, and the delayed allocation strategy that makes it fast.*

---

## 1. Inode Structure — The File Metadata Unit

### The Model

Every file and directory has an inode. ext4 inodes are fixed at **256 bytes** (vs ext3's 128 bytes). The extra 128 bytes store extended attributes and nanosecond timestamps.

### Inode Capacity

$$\text{Max Inodes} = \frac{\text{Filesystem Size}}{\text{Bytes per Inode}}$$

Default bytes-per-inode ratio = 16,384 (one inode per 16 KiB of space).

| FS Size | Bytes/Inode | Max Inodes | Inode Table Size |
|:---:|:---:|:---:|:---:|
| 100 GiB | 16,384 | 6,553,600 | 1.56 GiB |
| 1 TiB | 16,384 | 67,108,864 | 16 GiB |
| 10 TiB | 16,384 | 671,088,640 | 160 GiB |

### Inode Table Overhead

$$\text{Inode Table Size} = \text{Max Inodes} \times 256 \text{ bytes}$$

$$\text{Inode Overhead \%} = \frac{\text{Inode Table Size}}{\text{FS Size}} \times 100 = \frac{256}{16,384} \times 100 = 1.56\%$$

### Inode Internal Layout (256 bytes)

| Field | Bytes | Purpose |
|:---|:---:|:---|
| Mode, UID | 4 | File type and permissions |
| Size (low 32) | 4 | File size lower bits |
| Timestamps (atime, ctime, mtime, crtime) | 16 | 4 timestamps, second precision |
| Links count | 2 | Hard link count |
| Blocks count | 4 | 512-byte blocks allocated |
| Flags | 4 | Extent flag, immutable, etc. |
| Extent tree / block map | 60 | 4 extents or 12 direct + 3 indirect |
| Extended fields | 128 | Nanosecond timestamps, extra size, checksum |

---

## 2. Extent Tree — Replacing Indirect Blocks

### The Model

ext4 uses **extents** instead of indirect block pointers. An extent maps a contiguous range of logical blocks to physical blocks.

### Extent Structure (12 bytes each)

$$\text{Extent} = (\text{logical\_block}, \text{length}, \text{physical\_block})$$

$$\text{Max blocks per extent} = 2^{15} = 32,768 \text{ blocks}$$

$$\text{Max extent size} = 32,768 \times 4 \text{ KiB} = 128 \text{ MiB}$$

### Extent Tree Depth

The inode holds 4 extents directly. If more are needed, an extent tree is built:

$$\text{Extents per leaf node} = \frac{4096 - 12}{12} = 340$$

$$\text{Pointers per index node} = \frac{4096 - 12}{12} = 340$$

| Tree Depth | Max Extents | Max Fragmented File Size |
|:---:|:---:|:---:|
| 0 (inline) | 4 | 512 MiB |
| 1 | 4 × 340 = 1,360 | 42.5 GiB |
| 2 | 4 × 340 × 340 = 462,400 | 14.1 TiB |
| 3 | 4 × 340^2 × 340 = 157,216,000 | 4.7 PiB |

$$\text{Max Depth} = 5 \quad (\text{limited by ext4 implementation})$$

### Contiguous File — Best Case

A perfectly contiguous file needs only 1 extent per 128 MiB:

$$\text{Extents needed} = \lceil \frac{\text{File Size}}{128 \text{ MiB}} \rceil$$

| File Size | Extents (contiguous) | Extents (fragmented, 4K each) |
|:---:|:---:|:---:|
| 1 GiB | 8 | 262,144 |
| 100 GiB | 800 | 26,214,400 |
| 1 TiB | 8,192 | 268,435,456 |

---

## 3. Block Group Structure — Locality Optimization

### The Model

ext4 divides the filesystem into **block groups** (default 128 MiB each with 4K blocks). Each group has its own bitmaps and inode table for locality.

### Block Group Size

$$\text{Blocks per Group} = 8 \times \text{Block Size} = 8 \times 4096 = 32,768$$

$$\text{Group Size} = 32,768 \times 4 \text{ KiB} = 128 \text{ MiB}$$

$$\text{Total Groups} = \lceil \frac{\text{FS Size}}{128 \text{ MiB}} \rceil$$

| FS Size | Block Groups | Block Bitmap Size | Inode Bitmap Size |
|:---:|:---:|:---:|:---:|
| 100 GiB | 800 | 800 × 4 KiB = 3.1 MiB | 800 × 4 KiB = 3.1 MiB |
| 1 TiB | 8,192 | 32 MiB | 32 MiB |
| 10 TiB | 81,920 | 320 MiB | 320 MiB |

### Flex Block Groups

ext4 groups multiple block groups into **flex groups** (default flex_bg_size = 16):

$$\text{Flex Group Size} = \text{flex\_bg\_size} \times 128 \text{ MiB} = 2 \text{ GiB}$$

This clusters metadata for better sequential scanning.

---

## 4. Journal Modes — Write Path Trade-offs

### The Model

ext4's journal (jbd2) ensures crash consistency. Three modes offer different safety/performance trade-offs.

### Journal Modes

| Mode | What's Journaled | Write Path | Durability |
|:---|:---|:---|:---|
| `journal` | Metadata + Data | Data → Journal → Filesystem | Strongest |
| `ordered` (default) | Metadata only | Data → Filesystem, then Metadata → Journal | Strong |
| `writeback` | Metadata only | Data + Metadata in any order | Weakest |

### Write Amplification by Mode

$$\text{Write Amp}_{journal} = 2 \quad (\text{data written twice: journal + final location})$$

$$\text{Write Amp}_{ordered} = 1 \quad (\text{data written once, metadata journaled})$$

$$\text{Write Amp}_{writeback} = 1 \quad (\text{same as ordered but unordered})$$

### Journal Size

$$\text{Default Journal Size} = \min(128 \text{ MiB}, \frac{\text{FS Size}}{1024})$$

| FS Size | Journal Size | Journal as % of FS |
|:---:|:---:|:---:|
| 10 GiB | 10 MiB | 0.1% |
| 100 GiB | 100 MiB | 0.1% |
| 1 TiB | 128 MiB | 0.01% |
| 10 TiB | 128 MiB | 0.001% |

### Checkpoint Distance

$$\text{Checkpoint Interval} = \frac{\text{Journal Size}}{\text{Write Rate}}$$

At 100 MB/s writes with 128 MiB journal:

$$\text{Checkpoint Every} = \frac{128}{100} = 1.28 \text{ seconds}$$

Frequent checkpoints = more sync overhead. Larger journal = less frequent checkpoints.

---

## 5. Delayed Allocation — The mballoc Strategy

### The Model

ext4 delays block allocation until writeback time. This allows the **multiblock allocator (mballoc)** to see the full write pattern and allocate contiguously.

### Benefits Quantified

$$\text{Fragmentation}_{immediate} \gg \text{Fragmentation}_{delayed}$$

| Allocation Strategy | Extents for 1 GiB file (written in 4K chunks) |
|:---|:---:|
| Immediate (ext3) | Up to 262,144 (worst case) |
| Delayed (ext4 mballoc) | 8-16 (best case, large contiguous) |

### mballoc Allocation Order

1. **Goal allocation:** Try to extend the last extent
2. **Preallocation:** Reserve extra blocks (group prealloc = 512 blocks, inode prealloc = 8)
3. **Buddy allocator:** Find free chunks using buddy bitmaps (powers of 2)

### Preallocation Math

$$\text{Preallocated Blocks} = \min(\text{inode\_prealloc}, \text{Group Free Blocks})$$

$$\text{Preallocated Size} = 8 \times 4 \text{ KiB} = 32 \text{ KiB (per inode default)}$$

$$\text{Group Prealloc} = 512 \times 4 \text{ KiB} = 2 \text{ MiB}$$

---

## 6. Fill Factor and Free Space Fragmentation

### The Model

As a filesystem fills, allocation becomes harder. ext4 reserves space for root (default 5%) and experiences performance degradation.

### Performance vs Fill Level

$$\text{Fragmentation Risk} \propto \frac{1}{1 - \text{Fill Ratio}}$$

| Fill % | Free Space | Allocation Quality | Performance |
|:---:|:---:|:---|:---|
| 50% | 50% | Excellent, large contiguous | Baseline |
| 75% | 25% | Good, some fragmentation | ~95% |
| 85% | 15% | Moderate fragmentation | ~85% |
| 90% | 10% | Poor, many small fragments | ~70% |
| 95% | 5% | Very poor, heavy seeking | ~50% |
| 99% | 1% | Unusable without defrag | ~20% |

### Defragmentation Effectiveness

$$\text{Extents After Defrag} = \lceil \frac{\text{File Size}}{128 \text{ MiB}} \rceil \quad (\text{ideal})$$

$$\text{Defrag Improvement} = \frac{\text{Extents Before} - \text{Extents After}}{\text{Extents Before}} \times 100\%$$

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| $\frac{\text{FS Size}}{16,384}$ | Division | Inode count |
| $340^d \times 4$ | Exponential | Extent tree capacity |
| $8 \times \text{Block Size}$ | Multiplication | Block group sizing |
| Write amp = 1 or 2 | Constant | Journal mode cost |
| $\frac{1}{1 - \text{fill}}$ | Reciprocal | Fragmentation risk |
| $\lceil \frac{\text{Size}}{128\text{M}} \rceil$ | Ceiling | Minimum extents |

---

*Every `mkfs.ext4`, `e2fsck`, and `filefrag` command works with these structures — a 30-year-old design (ext2, 1993) that evolved through extent trees and delayed allocation into the most battle-tested Linux filesystem.*

## Prerequisites

- Inode and block allocation fundamentals
- Journaling concepts (write-ahead logging)
- Binary tree structures (extent trees)
- Disk partitioning basics (fdisk, parted)

## Complexity

- **Beginner:** Filesystem creation, mounting, basic tuning
- **Intermediate:** Block group layout, journal sizing, reserved block management
- **Advanced:** Extent tree depth analysis, HTree directory indexing, delayed allocation write amplification
