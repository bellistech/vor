# The Mathematics of df — Disk Free Space Internals

> *df reports filesystem space usage by reading the superblock. The math covers block accounting, inode exhaustion, reserved blocks, and capacity planning thresholds.*

---

## 1. Block Accounting — How df Calculates Space

### The Model

Every filesystem tracks space in fixed-size **blocks**. df reads the superblock (or statvfs syscall) to report three values.

### Core Formulas

$$\text{Total} = \text{Block Count} \times \text{Block Size}$$

$$\text{Used} = (\text{Total Blocks} - \text{Free Blocks}) \times \text{Block Size}$$

$$\text{Available} = \text{Free Blocks} - \text{Reserved Blocks}$$

$$\text{Use\%} = \frac{\text{Used}}{\text{Used} + \text{Available}} \times 100$$

**Note:** `Use%` is NOT `Used / Total` — it excludes reserved blocks from the denominator.

### The statvfs System Call

```
struct statvfs {
    f_bsize    // block size
    f_blocks   // total blocks
    f_bfree    // free blocks (including reserved)
    f_bavail   // available blocks (excluding reserved)
    f_files    // total inodes
    f_ffree    // free inodes
}
```

$$\text{df Total} = \texttt{f\_blocks} \times \texttt{f\_bsize}$$

$$\text{df Used} = (\texttt{f\_blocks} - \texttt{f\_bfree}) \times \texttt{f\_bsize}$$

$$\text{df Available} = \texttt{f\_bavail} \times \texttt{f\_bsize}$$

---

## 2. Reserved Blocks — The Hidden 5%

### The Model

ext4 reserves 5% of blocks by default for root (tunable via `tune2fs -m`). This prevents non-root users from filling the disk and crashing the system.

$$\text{Reserved} = \text{Total Blocks} \times \frac{\text{Reserved Percentage}}{100}$$

$$\text{Available to Users} = \text{Free} - \text{Reserved}$$

### Worked Examples

| Disk Size | Reserved % | Reserved Space | User-Visible Capacity |
|:---:|:---:|:---:|:---:|
| 100 GiB | 5% | 5 GiB | 95 GiB |
| 500 GiB | 5% | 25 GiB | 475 GiB |
| 1 TiB | 5% | 51.2 GiB | 972.8 GiB |
| 10 TiB | 5% | 512 GiB | 9.5 TiB |
| 10 TiB | 1% | 102.4 GiB | 9.9 TiB |

### When to Reduce Reserved Blocks

$$\text{Recovered Space} = \text{Total} \times \frac{\text{Old\%} - \text{New\%}}{100}$$

*"10 TiB data-only volume (no OS), reducing from 5% to 0%."*

$$\text{Recovered} = 10 \times 0.05 = 512 \text{ GiB}$$

| Use Case | Recommended Reserve | Reasoning |
|:---|:---:|:---|
| Root filesystem (/) | 5% | Prevent system lockup |
| /home | 1-2% | User data, some safety |
| Data volume (/data) | 0-1% | No system files |
| Temp/scratch | 0% | Ephemeral data |

---

## 3. Inode Exhaustion — Disk Full with Space Remaining

### The Model

Each file requires one inode. A filesystem can run out of inodes before running out of space.

$$\text{Inode Ratio} = \frac{\text{Disk Size}}{\text{Max Inodes}}$$

Default: 1 inode per 16 KiB of disk space (ext4).

$$\text{Max Files} = \frac{\text{Disk Size}}{16 \text{ KiB}} = \frac{\text{Disk Size (bytes)}}{16384}$$

### Worked Examples

| Disk Size | Inodes | Avg File Size for Exhaustion |
|:---:|:---:|:---:|
| 100 GiB | 6,553,600 | 16 KiB (all inodes used) |
| 1 TiB | 67,108,864 | 16 KiB |
| 10 TiB | 671,088,640 | 16 KiB |

### The Danger Zone

If average file size < inode ratio, inodes exhaust first:

$$\text{Inode Exhaustion Point} = \text{Max Inodes} \times \text{Avg File Size}$$

$$\text{Disk Used at Inode Exhaustion} = \frac{\text{Max Inodes} \times \text{Avg File Size}}{\text{Disk Size}} \times 100\%$$

| Disk Size | Avg File Size | Files at Full | Disk Used at Inode Exhaustion |
|:---:|:---:|:---:|:---:|
| 100 GiB | 1 KiB | 6,553,600 | 6.25% |
| 100 GiB | 4 KiB | 6,553,600 | 25% |
| 100 GiB | 16 KiB | 6,553,600 | 100% |
| 100 GiB | 100 KiB | 1,048,576 | 100% (space first) |

---

## 4. Filesystem Overhead — Why df Shows Less Than Disk Size

### Sources of Overhead

$$\text{Usable} = \text{Raw Disk} - \text{Partition Table} - \text{Superblocks} - \text{Inode Tables} - \text{Journal} - \text{Reserved}$$

### Overhead Components (ext4, 1 TiB disk)

| Component | Size | Percentage |
|:---|:---:|:---:|
| Partition table (GPT) | 34 sectors (~17 KiB) | ~0% |
| Superblock + backups | ~4 KiB × groups | ~0.01% |
| Inode table | $\frac{1 \text{ TiB}}{16 \text{ KiB}} \times 256 \text{ bytes}$ = 16 GiB | 1.56% |
| Journal | 128 MiB (default) | 0.01% |
| Block group descriptors | ~64 MiB | 0.006% |
| Reserved blocks (5%) | 51.2 GiB | 5% |
| **Total overhead** | **~67.4 GiB** | **~6.6%** |

This is why a "1 TiB" disk shows ~957 GiB in `df`.

---

## 5. Unit Conversion — The df Gotchas

### SI vs Binary Units

$$1 \text{ GB} = 10^9 \text{ bytes} = 1,000,000,000 \text{ bytes (SI)}$$

$$1 \text{ GiB} = 2^{30} \text{ bytes} = 1,073,741,824 \text{ bytes (binary)}$$

$$\text{Conversion: } 1 \text{ GiB} = 1.0737 \text{ GB}$$

### df Output Modes

| Flag | Unit | Block Size | 1 TiB disk shows |
|:---|:---:|:---:|:---:|
| `df` (default) | 1K blocks | 1,024 bytes | 1,073,741,824 |
| `df -h` | Human (binary) | Powers of 1024 | 1.0T |
| `df -H` | Human (SI) | Powers of 1000 | 1.1T |
| `df -B1` | Bytes | 1 byte | 1,099,511,627,776 |
| `df -BM` | Mebibytes | 1 MiB | 1,048,576M |

### The "Missing Space" Calculation

*"Bought a 2 TB drive. df -h shows 1.8T. Where did 200 GB go?"*

$$\text{Drive label} = 2 \text{ TB} = 2 \times 10^{12} \text{ bytes}$$

$$\text{In TiB} = \frac{2 \times 10^{12}}{2^{40}} = 1.818 \text{ TiB}$$

$$\text{After ext4 overhead (6.6\%)} = 1.818 \times 0.934 = 1.698 \text{ TiB} \approx 1.7 \text{T}$$

---

## 6. Monitoring Thresholds — Capacity Planning

### Alert Formula

$$\text{Days Until Full} = \frac{\text{Available Space}}{\text{Daily Growth Rate}}$$

$$\text{Growth Rate} = \frac{\Delta \text{Used}}{\Delta \text{Time}}$$

### Worked Example

*"500 GiB volume, 80% full, growing 2 GiB/day."*

$$\text{Available} = 500 \times 0.20 = 100 \text{ GiB}$$

$$\text{Days} = \frac{100}{2} = 50 \text{ days}$$

| Usage % | Available (500 GiB) | At 2 GiB/day | At 10 GiB/day |
|:---:|:---:|:---:|:---:|
| 70% | 150 GiB | 75 days | 15 days |
| 80% | 100 GiB | 50 days | 10 days |
| 85% | 75 GiB | 37 days | 7.5 days |
| 90% | 50 GiB | 25 days | 5 days |
| 95% | 25 GiB | 12 days | 2.5 days |

### Standard Alert Thresholds

| Threshold | Action | Urgency |
|:---:|:---|:---|
| 70% | Informational — plan expansion | Low |
| 80% | Warning — schedule cleanup | Medium |
| 90% | Critical — immediate action | High |
| 95% | Emergency — risk of outage | Immediate |

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| $\text{Blocks} \times \text{Size}$ | Linear arithmetic | Space calculation |
| $\text{Total} \times \frac{r}{100}$ | Percentage | Reserved blocks |
| $\frac{\text{Disk}}{16\text{K}}$ | Division | Inode count |
| $\frac{\text{Available}}{\text{Rate}}$ | Rate equation | Days until full |
| $\frac{\text{bytes}}{2^{30}}$ | Unit conversion | SI vs binary |

---

*Every `df -h` you run is a statvfs() syscall reading the superblock — a constant-time operation that reports the filesystem's own internal block accounting.*

## Prerequisites

- Filesystem block allocation concepts
- Inode fundamentals
- Mount points and filesystem hierarchy
- Binary vs decimal size units (GiB vs GB)

## Complexity

- **Beginner:** Reading disk usage output, human-readable formatting
- **Intermediate:** Reserved block accounting, inode exhaustion diagnosis
- **Advanced:** statvfs() syscall internals, filesystem-specific metadata overhead calculations
