# The Mathematics of ZFS — Storage Theory and Internals

> *ZFS is a combined filesystem and volume manager built on copy-on-write semantics. The math spans vdev geometry, ARC cache sizing, deduplication memory costs, RAIDZ parity overhead, and scrub time estimation.*

---

## 1. Vdev Geometry — Stripe Width and Parity

### The Model

ZFS pools are composed of **vdevs** (virtual devices). Each vdev is an independent failure domain. Data is striped across vdevs at the pool level, and redundancy is applied within each vdev.

### RAIDZ Capacity Formula

$$\text{Usable per vdev} = (n - p) \times \text{Smallest Disk}$$

Where:
- $n$ = number of disks in the vdev
- $p$ = parity disks (1 for RAIDZ1, 2 for RAIDZ2, 3 for RAIDZ3)

$$\text{Pool Usable} = \sum_{i=1}^{v} (n_i - p_i) \times \text{Disk Size}_i$$

### Parity Overhead Fraction

$$\text{Overhead} = \frac{p}{n}$$

| Layout | Disks ($n$) | Parity ($p$) | Usable Fraction | Overhead |
|:---|:---:|:---:|:---:|:---:|
| RAIDZ1 | 4 | 1 | 75.0% | 25.0% |
| RAIDZ1 | 6 | 1 | 83.3% | 16.7% |
| RAIDZ1 | 8 | 1 | 87.5% | 12.5% |
| RAIDZ2 | 6 | 2 | 66.7% | 33.3% |
| RAIDZ2 | 8 | 2 | 75.0% | 25.0% |
| RAIDZ2 | 12 | 2 | 83.3% | 16.7% |
| RAIDZ3 | 8 | 3 | 62.5% | 37.5% |
| RAIDZ3 | 12 | 3 | 75.0% | 25.0% |
| Mirror | 2 | 1 | 50.0% | 50.0% |
| Mirror | 3 | 2 | 33.3% | 66.7% |

### Stripe Width and Record Size

ZFS writes in variable-size **records** (default `recordsize=128K`). A RAIDZ stripe spans all disks:

$$\text{Stripe Width} = n - p \text{ data blocks per stripe}$$

$$\text{Optimal Record Size} = \text{Stripe Width} \times \text{Sector Size}$$

**Padding penalty:** If a record doesn't fill a full stripe, ZFS pads. The wasted space:

$$\text{Padding Waste} = 1 - \frac{\text{Record Size}}{\lceil \text{Record Size} / \text{Stripe Width} \rceil \times \text{Stripe Width}}$$

---

## 2. ARC Cache Sizing — Adaptive Replacement Cache

### The Model

ARC is ZFS's main memory read cache. It adaptively balances between **recently used (MRU)** and **frequently used (MFU)** lists.

### Default Sizing

$$\text{ARC Max} = \text{Total RAM} - 1 \text{ GiB} \quad (\text{up to } 50\% \text{ of RAM on Linux})$$

$$\text{ARC Target} \approx \frac{\text{RAM}}{2} \quad (\text{default on Linux})$$

### ARC Memory Breakdown

Each cached block consumes:

$$\text{ARC Memory per Block} = \text{Block Size} + \text{ARC Header} (176 \text{ bytes})$$

$$\text{ARC Blocks at Max} = \frac{\text{ARC Max}}{\text{Avg Block Size} + 176}$$

### Worked Example

*"128 GiB RAM server, recordsize=128K."*

$$\text{ARC Max} = 64 \text{ GiB (50\% default)}$$

$$\text{Cached Blocks} = \frac{64 \times 1024 \times 1024 \text{ KiB}}{128 + 0.172 \text{ KiB}} \approx 524,000 \text{ blocks}$$

$$\text{Cached Data} \approx 524,000 \times 128 \text{ KiB} = 63.97 \text{ GiB}$$

### L2ARC (SSD-based Second Level)

L2ARC stores evicted ARC entries on SSD. Memory cost for L2ARC index:

$$\text{L2ARC Header Memory} = \frac{\text{L2ARC Size}}{\text{Avg Block Size}} \times 176 \text{ bytes}$$

| L2ARC SSD Size | Block Size | Index Headers | RAM for Index |
|:---:|:---:|:---:|:---:|
| 100 GiB | 128 KiB | 819,200 | 137 MiB |
| 500 GiB | 128 KiB | 4,096,000 | 687 MiB |
| 1 TiB | 128 KiB | 8,388,608 | 1.31 GiB |
| 1 TiB | 8 KiB | 134,217,728 | 20.9 GiB |

**Key insight:** Small block sizes make L2ARC index consume significant RAM.

---

## 3. Deduplication — DDT Memory Cost

### The Model

ZFS dedup maintains a **Dedup Table (DDT)** in memory. Every unique block gets an entry.

### DDT Memory Formula

$$\text{DDT Memory} = \text{Unique Blocks} \times 320 \text{ bytes}$$

$$\text{Unique Blocks} = \frac{\text{Total Logical Data}}{\text{Record Size} \times \text{Dedup Ratio}}$$

### Dedup Ratio

$$\text{Dedup Ratio} = \frac{\text{Logical Data Referenced}}{\text{Physical Data Stored}}$$

### Worked Example

*"10 TiB logical data, recordsize=128K, dedup ratio 2:1."*

$$\text{Unique Blocks} = \frac{10 \times 1024 \times 1024 \text{ MiB}}{0.125 \text{ MiB} \times 2} = 41,943,040$$

$$\text{DDT Memory} = 41,943,040 \times 320 = 12.5 \text{ GiB}$$

| Logical Data | Record Size | Dedup Ratio | Unique Blocks | DDT RAM |
|:---:|:---:|:---:|:---:|:---:|
| 1 TiB | 128 KiB | 1.5:1 | 5,592,405 | 1.66 GiB |
| 10 TiB | 128 KiB | 2:1 | 41,943,040 | 12.5 GiB |
| 50 TiB | 128 KiB | 3:1 | 139,810,133 | 41.6 GiB |
| 100 TiB | 128 KiB | 2:1 | 419,430,400 | 125.0 GiB |

**Rule of thumb:** Need ~5 GiB RAM per TiB of unique data at 128K records. If DDT doesn't fit in ARC, dedup performance collapses.

---

## 4. Scrub and Resilver Time Estimation

### Scrub Time Formula

$$T_{scrub} = \frac{\text{Total Allocated Data}}{\text{Scrub I/O Rate}}$$

Scrub reads every allocated block and verifies checksums:

$$\text{Scrub I/O Rate} = \min(\text{Disk Sequential Read}, \text{scrub throughput limit})$$

Default `zfs_scrub_delay` throttles to ~200 MB/s per vdev to avoid impacting production.

### Worked Example

*"60 TiB pool, 40 TiB allocated, 6-disk RAIDZ2, each disk 200 MB/s sequential."*

$$\text{Effective Read} = 4 \times 200 = 800 \text{ MB/s (4 data disks)}$$

$$\text{Throttled} = \min(800, 200) = 200 \text{ MB/s}$$

$$T = \frac{40 \times 1024 \times 1024}{200} = 209,715 \text{ sec} \approx 58.3 \text{ hours}$$

### Resilver Time

Resilvering replaces a failed disk. ZFS resilvers only allocated blocks (not entire disk):

$$T_{resilver} = \frac{\text{Allocated Data per Disk}}{\text{Rebuild I/O Rate}}$$

$$\text{Data per Disk} \approx \frac{\text{Total Allocated}}{n - p}$$

| Pool Allocated | Disks | Data/Disk | Disk Speed | Resilver Time |
|:---:|:---:|:---:|:---:|:---:|
| 10 TiB | 6 (RZ2) | 2.5 TiB | 200 MB/s | 3.6 hours |
| 40 TiB | 8 (RZ2) | 6.7 TiB | 200 MB/s | 9.5 hours |
| 80 TiB | 12 (RZ2) | 8.0 TiB | 200 MB/s | 11.4 hours |
| 80 TiB | 12 (RZ2) | 8.0 TiB | 500 MB/s | 4.6 hours |

---

## 5. Copy-on-Write Space Amplification

### The Model

Every write in ZFS creates new blocks (COW). Metadata updates cascade up the block tree (Merkle tree / indirect blocks).

### Write Amplification

$$\text{Write Amp} = 1 + \lceil \log_{b}(N) \rceil$$

Where:
- $b$ = block pointer fan-out (~128 for 128K records with 1K pointers)
- $N$ = total blocks in dataset
- The "+1" is the uberblock update

For a 10 TiB dataset with 128K records:

$$N = \frac{10 \times 1024^4}{128 \times 1024} = 83,886,080 \text{ blocks}$$

$$\text{Tree Depth} = \lceil \log_{128}(83,886,080) \rceil = \lceil 3.76 \rceil = 4$$

$$\text{Write Amp} = 1 + 4 = 5 \text{ (5 blocks written per data block)}$$

### Snapshot Space

Snapshots are free at creation (COW). Space consumed = blocks changed since snapshot:

$$\text{Snapshot Space} = \text{Changed Blocks} \times \text{Avg Block Size}$$

$$\text{Daily Snapshot Cost} = \text{Dataset Size} \times \text{Daily Change Rate}$$

---

## 6. Compression Ratios and Throughput

### Compression Algorithms

$$\text{Effective Capacity} = \text{Raw Capacity} \times \text{Compression Ratio}$$

$$\text{Effective Write Speed} = \frac{\text{Disk Write Speed}}{\text{Compression Ratio}} \times \text{CPU Factor}$$

| Algorithm | Typical Ratio | CPU Cost | Best For |
|:---|:---:|:---:|:---|
| lz4 | 1.5-2.5x | Very low | Default, general purpose |
| zstd (level 3) | 2.0-4.0x | Low-medium | Balanced performance |
| zstd (level 19) | 3.0-6.0x | High | Archival, cold storage |
| gzip-9 | 2.5-5.0x | Very high | Legacy, maximum compat |

### Worked Example

*"20 TiB raw pool with lz4 compression, ratio 2.2x."*

$$\text{Effective Capacity} = 20 \times 2.2 = 44 \text{ TiB logical}$$

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| $(n - p) \times \text{disk}$ | Linear arithmetic | RAIDZ capacity |
| $\frac{p}{n}$ | Fraction | Parity overhead |
| $\frac{\text{Data}}{\text{Rate}}$ | Rate equation | Scrub/resilver time |
| $\text{blocks} \times 320$ | Linear scaling | DDT memory |
| $\lceil \log_b(N) \rceil$ | Logarithmic | COW tree depth |
| $\text{Raw} \times \text{Ratio}$ | Multiplication | Effective capacity |

---

*Every `zpool scrub`, `zfs send`, and `zpool status` you run is traversing these data structures — a Merkle tree of checksummed, copy-on-write blocks that self-heals on read.*
