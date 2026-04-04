# The Mathematics of du — Disk Usage Tree Traversal

> *du walks directory trees counting allocated blocks. The math covers block allocation waste, hard link counting, sparse file detection, and traversal complexity.*

---

## 1. Block Allocation — Apparent vs Actual Size

### The Model

Files occupy whole blocks on disk. A 1-byte file in a 4 KiB block wastes 4,095 bytes.

### Core Formulas

$$\text{Blocks Allocated} = \left\lceil \frac{\text{File Size}}{\text{Block Size}} \right\rceil$$

$$\text{Disk Usage} = \text{Blocks Allocated} \times \text{Block Size}$$

$$\text{Internal Fragmentation} = \text{Disk Usage} - \text{Apparent Size}$$

$$\text{Waste Ratio} = \frac{\text{Internal Fragmentation}}{\text{Disk Usage}} \times 100\%$$

### Worked Examples (4 KiB block size)

| File Size | Blocks | Disk Usage | Wasted | Waste % |
|:---:|:---:|:---:|:---:|:---:|
| 1 byte | 1 | 4 KiB | 4,095 bytes | 99.98% |
| 100 bytes | 1 | 4 KiB | 3,996 bytes | 97.56% |
| 4,096 bytes | 1 | 4 KiB | 0 bytes | 0% |
| 4,097 bytes | 2 | 8 KiB | 4,095 bytes | 49.99% |
| 10,000 bytes | 3 | 12 KiB | 2,288 bytes | 18.61% |
| 1 MiB | 256 | 1 MiB | 0 bytes | 0% |

### Average Waste per File

For uniformly distributed file sizes within a block:

$$E[\text{Waste per File}] = \frac{\text{Block Size}}{2} = 2 \text{ KiB (for 4 KiB blocks)}$$

$$\text{Total Waste} \approx \text{File Count} \times \frac{\text{Block Size}}{2}$$

| File Count | Avg Waste Each | Total Wasted Space |
|:---:|:---:|:---:|
| 1,000 | 2 KiB | 2 MiB |
| 100,000 | 2 KiB | 195 MiB |
| 1,000,000 | 2 KiB | 1.9 GiB |
| 10,000,000 | 2 KiB | 19.1 GiB |

---

## 2. du vs ls — The Two Measurements

### Apparent Size (ls -l, du --apparent-size)

$$\text{Apparent} = \sum_{i=1}^{n} \text{st\_size}_i \quad \text{(from stat() syscall)}$$

### Disk Usage (du default)

$$\text{Disk Usage} = \sum_{i=1}^{n} \text{st\_blocks}_i \times 512 \quad \text{(512-byte sectors from stat())}$$

**Key:** `st_blocks` is always in 512-byte units regardless of filesystem block size.

### When They Diverge

| Scenario | Apparent | Disk Usage | Ratio |
|:---|:---:|:---:|:---:|
| Normal file (1 MiB) | 1 MiB | 1 MiB | 1.0 |
| Many tiny files | Small | Larger | > 1.0 |
| Sparse file (1 GiB with holes) | 1 GiB | ~0 | << 1.0 |
| Compressed (btrfs/zfs) | Original | Compressed | < 1.0 |
| Hard links (counted once) | Sum of sizes | Unique blocks only | < 1.0 |

---

## 3. Sparse File Detection

### The Model

Sparse files have "holes" — regions that read as zeros but consume no disk blocks.

### Sparseness Formula

$$\text{Sparseness} = 1 - \frac{\text{Disk Usage}}{\text{Apparent Size}}$$

$$\text{Hole Size} = \text{Apparent Size} - \text{Disk Usage}$$

### Worked Example

*"VM disk image: apparent size 100 GiB, disk usage 15 GiB."*

$$\text{Sparseness} = 1 - \frac{15}{100} = 0.85 = 85\%$$

$$\text{Holes} = 100 - 15 = 85 \text{ GiB of zeros not stored}$$

### Detection with du

```bash
du -h --apparent-size file    # shows 100G
du -h file                    # shows 15G  -- difference = holes
```

---

## 4. Directory Traversal Complexity

### The Model

du recursively walks the directory tree using `opendir()`, `readdir()`, `stat()` on each entry.

### Time Complexity

$$T_{du} = O(F + D)$$

Where:
- $F$ = total files
- $D$ = total directories

Each entry requires one `stat()` syscall and one `readdir()` call.

### I/O Cost

For uncached directory traversal on HDD:

$$T_{uncached} \approx F \times T_{seek} + D \times T_{dir\_read}$$

Where $T_{seek} \approx 5-10$ ms for random I/O on HDD.

| Files | HDD (uncached) | SSD (uncached) | Cached (RAM) |
|:---:|:---:|:---:|:---:|
| 1,000 | 5-10 sec | 0.1 sec | <0.01 sec |
| 100,000 | 8-17 min | 10 sec | 0.5 sec |
| 1,000,000 | 83-167 min | 100 sec | 5 sec |
| 10,000,000 | 14-28 hours | 17 min | 50 sec |

**This is why `du` on millions of files on HDD is painfully slow — each stat() is a random seek.**

---

## 5. Hard Link Counting

### The Model

du counts each inode only once. Hard links create multiple directory entries pointing to the same inode.

### Space Calculation

$$\text{Apparent Total} = \sum_{i=1}^{n} \text{Size}_i \quad (\text{counts each link})$$

$$\text{Actual Disk} = \sum_{\text{unique inodes}} \text{Blocks}_j \quad (\text{counts each inode once})$$

$$\text{Hard Link Savings} = \text{Apparent} - \text{Actual}$$

### Worked Example

*"10,000 files of 1 MiB each, but 5,000 are hard links to the other 5,000."*

$$\text{Apparent} = 10,000 \times 1 \text{ MiB} = 9.77 \text{ GiB}$$

$$\text{Actual} = 5,000 \times 1 \text{ MiB} = 4.88 \text{ GiB}$$

$$\text{Savings} = 4.88 \text{ GiB (50\%)}$$

`du` reports 4.88 GiB. `du --apparent-size` reports 9.77 GiB.

---

## 6. du Aggregation — Sort by Size

### The Summarization Formula

$$\text{du -s DIR} = \sum_{\text{all files in DIR recursively}} \text{Blocks}_i \times 512$$

### Finding Space Hogs — The Pareto Principle

In most filesystems, ~20% of directories consume ~80% of space:

$$\text{Top-}k\text{ coverage} = \frac{\sum_{i=1}^{k} \text{Size}_i}{\text{Total Size}} \times 100\%$$

### Practical Pipeline

```bash
du -h --max-depth=1 /path | sort -hr | head -20
```

This gives $O(D_1)$ complexity where $D_1$ = number of first-level subdirectories, but each subtree is fully traversed.

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| $\lceil \frac{\text{Size}}{\text{Block}} \rceil \times \text{Block}$ | Ceiling function | Block allocation |
| $\text{Apparent} - \text{Disk}$ | Subtraction | Sparse detection |
| $1 - \frac{\text{Disk}}{\text{Apparent}}$ | Ratio | Sparseness metric |
| $F \times T_{seek}$ | Linear scaling | Traversal time |
| $\text{Files} \times \frac{\text{Block}}{2}$ | Expected value | Average waste |

---

*Every `du -sh` you run triggers a recursive stat() walk of the entire directory tree — a deceptively expensive operation that becomes the bottleneck on filesystems with millions of small files.*

## Prerequisites

- Filesystem tree structure and directory entries
- stat() syscall and inode metadata
- Hard links vs symbolic links (impact on reported sizes)
- Block allocation vs logical file size

## Complexity

- **Beginner:** Summarizing directory sizes
- **Intermediate:** Apparent size vs disk usage, cross-filesystem traversal
- **Advanced:** stat() walk performance modeling, inode cache effects, sparse file accounting
