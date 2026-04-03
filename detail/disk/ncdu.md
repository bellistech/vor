# The Mathematics of ncdu — Interactive Disk Usage Analysis

> *ncdu (NCurses Disk Usage) is a fast, interactive disk usage analyzer. The math covers tree traversal algorithms, memory footprint scaling, sorting complexity, and comparison with du performance.*

---

## 1. Scanning Algorithm — Breadth-First Traversal

### The Model

ncdu scans the filesystem using a recursive directory traversal, building an in-memory tree of all files and directories.

### Scan Complexity

$$T_{scan} = O(F + D)$$

Where:
- $F$ = total files
- $D$ = total directories
- Each entry requires one `lstat()` syscall

### I/O Cost Model

$$T_{uncached} = (F + D) \times T_{stat}$$

| Storage | $T_{stat}$ | 1M files | 10M files |
|:---|:---:|:---:|:---:|
| HDD (random) | 5-10 ms | 83-167 min | 14-28 hours |
| SSD (random) | 0.05-0.1 ms | 50-100 sec | 8-17 min |
| NVMe (random) | 0.01-0.02 ms | 10-20 sec | 100-200 sec |
| Cached (RAM) | 0.001 ms | 1 sec | 10 sec |

### ncdu vs du Comparison

ncdu stores the full tree in memory; du can stream and discard. But ncdu only scans once for interactive browsing:

$$\text{ncdu total time} = T_{scan} + T_{browse}$$

$$\text{du repeated} = n \times T_{scan} \quad (\text{for } n \text{ different directory queries})$$

**Break-even:** ncdu wins when you explore more than 1 directory subtree.

---

## 2. Memory Footprint — Tree Structure

### Per-Node Memory

ncdu stores a tree node for every file and directory:

$$\text{Memory per node} \approx 64 + \text{name length} \text{ bytes (ncdu v1)}$$

$$\text{Memory per node} \approx 80 + \text{name length} \text{ bytes (ncdu v2, Zig)}$$

### Total Memory Formula

$$\text{Total RAM} = (F + D) \times (\text{Node Overhead} + \overline{\text{Name Length}})$$

Where $\overline{\text{Name Length}}$ = average filename length (typically 15-25 bytes).

### Worked Examples

| Files + Dirs | Avg Name | Per Node | Total RAM |
|:---:|:---:|:---:|:---:|
| 10,000 | 20 bytes | 84 bytes | 820 KiB |
| 100,000 | 20 bytes | 84 bytes | 8 MiB |
| 1,000,000 | 20 bytes | 84 bytes | 80 MiB |
| 10,000,000 | 20 bytes | 84 bytes | 800 MiB |
| 100,000,000 | 25 bytes | 89 bytes | 8.3 GiB |

### Comparison with du

| Tool | Memory for 10M files | Interactive | Rescan Needed |
|:---|:---:|:---:|:---|
| `du -sh` | ~0 (streaming) | No | Yes, every query |
| `du \| sort` | ~500 MiB (pipe buffer) | No | Yes |
| ncdu | ~800 MiB | Yes | No |

---

## 3. Sorting and Display — Algorithmic Complexity

### Sort Complexity

ncdu sorts directory entries for display:

$$T_{sort} = O(n \log n) \quad \text{per directory level}$$

Where $n$ = entries in the current directory.

### Sort Modes

| Mode | Key | Comparison | Use Case |
|:---|:---:|:---|:---|
| Size (default) | `s` | Descending size | Find space hogs |
| Name | `n` | Alphabetical | Browse by name |
| Items | `C` | Item count | Find dir sprawl |
| Modified time | `M` | Most recent first | Find active dirs |

### Display Percentage Formula

$$\text{Bar Width} = \frac{\text{Entry Size}}{\text{Parent Size}} \times \text{Terminal Width}$$

$$\text{Percentage} = \frac{\text{Entry Size}}{\text{Parent Size}} \times 100$$

---

## 4. Exclusion Patterns — Reducing Scan Scope

### The Model

Excluding directories prunes entire subtrees from the scan, saving both time and memory.

### Savings Formula

$$T_{saved} = \frac{\text{Excluded Files}}{\text{Total Files}} \times T_{scan}$$

$$\text{RAM Saved} = \text{Excluded Entries} \times \text{Per Node Size}$$

### Common Exclusions

| Pattern | Typical Files Skipped | Example |
|:---|:---:|:---|
| `.git` | Thousands-millions | Repo history objects |
| `node_modules` | 50,000-200,000 per project | JS dependencies |
| `__pycache__` | Hundreds per project | Python bytecode |
| `.cache` | Varies widely | User cache data |
| `/proc, /sys` | Virtual files | Pseudo-filesystems |

### Worked Example

*"Scanning /home with 5M files. node_modules contains 2M files across 10 JS projects."*

$$T_{without\_exclusion} = 5,000,000 \times 0.05 \text{ ms} = 250 \text{ sec}$$

$$T_{with\_exclusion} = 3,000,000 \times 0.05 \text{ ms} = 150 \text{ sec}$$

$$\text{Speedup} = \frac{250}{150} = 1.67\times$$

$$\text{RAM Saved} = 2,000,000 \times 84 = 160 \text{ MiB}$$

---

## 5. Disk Usage Accounting Modes

### Apparent Size vs Disk Usage

ncdu supports both modes (toggle with `a` key):

$$\text{Disk Usage} = \sum \text{st\_blocks} \times 512$$

$$\text{Apparent Size} = \sum \text{st\_size}$$

### When They Differ Significantly

| Scenario | Disk/Apparent Ratio | Explanation |
|:---|:---:|:---|
| Normal files (>4K) | ~1.0 | Aligned to blocks |
| Many tiny files | >1.0 | Block allocation waste |
| Sparse files | <<1.0 | Holes not counted |
| Compressed (btrfs) | <1.0 | Transparent compression |
| Hard links | <1.0 | Counted once on disk |

### Shared Size Calculation

ncdu v2 tracks shared extents (hard links, reflinks):

$$\text{Unique Size} = \text{Disk Usage} - \text{Shared Size}$$

$$\text{Reclaimable on Delete} = \begin{cases} \text{Unique Size} & \text{if last link} \\ 0 & \text{if other links remain} \end{cases}$$

---

## 6. Export and Differential Analysis

### Export Format Size

ncdu can export to JSON for offline analysis:

$$\text{Export Size} \approx (F + D) \times 100 \text{ bytes (JSON per entry)}$$

| Entries | Export File Size | Gzip Compressed |
|:---:|:---:|:---:|
| 100,000 | 9.5 MiB | ~2 MiB |
| 1,000,000 | 95 MiB | ~20 MiB |
| 10,000,000 | 953 MiB | ~200 MiB |

### Differential Disk Analysis

Compare two exports to find growth:

$$\Delta\text{Size} = \text{Size}_{t2} - \text{Size}_{t1}$$

$$\text{Growth Rate} = \frac{\Delta\text{Size}}{\Delta t}$$

$$\text{Days to Full} = \frac{\text{Available}}{\text{Growth Rate}}$$

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| $O(F + D)$ | Linear | Scan time |
| $(F+D) \times 84$ bytes | Linear scaling | Memory footprint |
| $O(n \log n)$ | Linearithmic | Sort per directory |
| $\frac{\text{Entry}}{\text{Parent}} \times 100$ | Ratio | Display percentage |
| $\frac{\text{Available}}{\text{Growth}}$ | Rate equation | Capacity forecasting |

---

*ncdu trades memory for interactivity — one O(n) scan builds an in-memory tree that answers unlimited disk usage queries without rescanning, making it the preferred tool for hunting down space hogs.*
