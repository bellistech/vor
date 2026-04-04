# The Mathematics of mdadm — Linux Software RAID Internals

> *mdadm manages Linux software RAID arrays. The math covers RAID level capacity, rebuild times, stripe calculations, write penalties, and failure probability.*

---

## 1. RAID Capacity Formulas

### Core Equations

Each RAID level has a deterministic capacity formula:

$$\text{RAID 0: } C = n \times D$$

$$\text{RAID 1: } C = D$$

$$\text{RAID 5: } C = (n - 1) \times D$$

$$\text{RAID 6: } C = (n - 2) \times D$$

$$\text{RAID 10: } C = \frac{n}{2} \times D$$

Where:
- $C$ = usable capacity
- $n$ = number of disks
- $D$ = size of smallest disk

### Storage Efficiency

$$\text{Efficiency} = \frac{C}{n \times D} \times 100\%$$

| RAID | Disks | Each Size | Raw | Usable | Efficiency | Fault Tolerance |
|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| 0 | 4 | 4 TiB | 16 TiB | 16 TiB | 100% | 0 disks |
| 1 | 2 | 4 TiB | 8 TiB | 4 TiB | 50% | 1 disk |
| 5 | 4 | 4 TiB | 16 TiB | 12 TiB | 75% | 1 disk |
| 5 | 8 | 4 TiB | 32 TiB | 28 TiB | 87.5% | 1 disk |
| 6 | 4 | 4 TiB | 16 TiB | 8 TiB | 50% | 2 disks |
| 6 | 8 | 4 TiB | 32 TiB | 24 TiB | 75% | 2 disks |
| 10 | 4 | 4 TiB | 16 TiB | 8 TiB | 50% | 1 per mirror |
| 10 | 8 | 4 TiB | 32 TiB | 16 TiB | 50% | 1 per mirror |

---

## 2. Rebuild Time — The Critical Window

### Rebuild Time Formula

$$T_{rebuild} = \frac{D}{\text{Rebuild Speed}}$$

Where:
- $D$ = disk capacity (entire disk, not just used data — mdadm rebuilds whole disk)
- Rebuild speed = sequential write rate, typically throttled

### mdadm Speed Controls

```
/proc/sys/dev/raid/speed_limit_min = 1000 KB/s (default)
/proc/sys/dev/raid/speed_limit_max = 200000 KB/s (default)
```

### Worked Examples

| Disk Size | Rebuild Speed | Rebuild Time |
|:---:|:---:|:---:|
| 1 TiB | 100 MB/s | 2.9 hours |
| 4 TiB | 100 MB/s | 11.4 hours |
| 4 TiB | 200 MB/s | 5.7 hours |
| 8 TiB | 100 MB/s | 22.8 hours |
| 16 TiB | 100 MB/s | 45.5 hours |
| 16 TiB | 200 MB/s | 22.8 hours |

### Effective Rebuild Speed Under Load

$$\text{Effective Speed} = \text{Max Speed} \times (1 - \text{I/O Load Fraction})$$

*"4 TiB disk, max rebuild 200 MB/s, system under 40% I/O load."*

$$\text{Effective} = 200 \times 0.60 = 120 \text{ MB/s}$$

$$T = \frac{4 \times 1024 \times 1024}{120} = 34,952 \text{ sec} \approx 9.7 \text{ hours}$$

---

## 3. Write Penalty — RAID Tax on Writes

### The Model

RAID parity calculation requires read-modify-write cycles, creating a write penalty.

### Write Penalty by RAID Level

| RAID | Write Penalty | I/O Operations per Write | Explanation |
|:---:|:---:|:---:|:---|
| 0 | 1 | 1 write | No redundancy |
| 1 | 2 | 2 writes | Mirror copy |
| 5 | 4 | 2 reads + 2 writes | Read old data+parity, write new data+parity |
| 6 | 6 | 3 reads + 3 writes | Two parity blocks |
| 10 | 2 | 2 writes | Mirror of stripes |

### Effective Write IOPS

$$\text{Effective IOPS}_{write} = \frac{\text{Raw IOPS per disk} \times n}{\text{Write Penalty}}$$

$$\text{Effective IOPS}_{read} = \text{Raw IOPS per disk} \times n \quad (\text{RAID 0/5/6})$$

| RAID | Disks | Disk IOPS | Read IOPS | Write IOPS |
|:---:|:---:|:---:|:---:|:---:|
| 0 | 4 | 200 | 800 | 800 |
| 1 | 2 | 200 | 400 | 200 |
| 5 | 4 | 200 | 800 | 200 |
| 6 | 4 | 200 | 800 | 133 |
| 10 | 4 | 200 | 800 | 400 |

---

## 4. Stripe Size and Chunk Math

### Stripe Geometry

$$\text{Stripe Width} = \text{Data Disks} \times \text{Chunk Size}$$

$$\text{Full Stripe Width} = n \times \text{Chunk Size}$$

Where chunk size (mdadm `--chunk`) defaults to 512 KiB.

### Full Stripe Write Optimization

A write that spans a full stripe avoids the read-modify-write penalty:

$$\text{I/O Size for Full Stripe (RAID5)} = (n - 1) \times \text{Chunk Size}$$

| Disks | Chunk Size | Full Stripe (RAID5) | Full Stripe (RAID6) |
|:---:|:---:|:---:|:---:|
| 4 | 64 KiB | 192 KiB | 128 KiB |
| 4 | 512 KiB | 1.5 MiB | 1 MiB |
| 8 | 64 KiB | 448 KiB | 384 KiB |
| 8 | 512 KiB | 3.5 MiB | 3 MiB |

### Optimal Chunk Size Selection

$$\text{Chunk Size} \geq \text{Typical I/O Size}$$

| Workload | Typical I/O | Recommended Chunk |
|:---|:---:|:---:|
| Database (random 8K) | 8 KiB | 64 KiB |
| File server | 64-256 KiB | 256 KiB |
| Sequential (video) | 1+ MiB | 512 KiB - 1 MiB |
| VM images | 64-128 KiB | 128 KiB |

---

## 5. Failure Probability — URE During Rebuild

### The Model

Unrecoverable Read Errors (UREs) during rebuild can cause array failure. Enterprise drives specify URE rate as errors per bits read.

### URE Probability During Rebuild

$$P_{URE} = 1 - \left(1 - \frac{1}{\text{URE Rate}}\right)^{\text{Bits Read}}$$

For large reads, approximate:

$$P_{URE} \approx \frac{\text{Bits Read}}{\text{URE Rate}}$$

### Worked Example

*"RAID 5, 4 × 4 TiB consumer drives (URE = 10^14 bits/error). Rebuild reads 3 remaining disks."*

$$\text{Bits Read} = 3 \times 4 \times 10^{12} \times 8 = 9.6 \times 10^{13} \text{ bits}$$

$$P_{URE} = \frac{9.6 \times 10^{13}}{10^{14}} = 0.96 = 96\%$$

**96% chance of data loss during RAID 5 rebuild with consumer drives.**

| RAID | Disks × Size | URE Rate | Bits Read | $P_{URE}$ |
|:---:|:---:|:---:|:---:|:---:|
| RAID 5 | 4 × 2 TiB | $10^{14}$ | $4.8 \times 10^{13}$ | 48% |
| RAID 5 | 4 × 4 TiB | $10^{14}$ | $9.6 \times 10^{13}$ | 96% |
| RAID 5 | 4 × 4 TiB | $10^{15}$ | $9.6 \times 10^{13}$ | 9.6% |
| RAID 6 | 4 × 4 TiB | $10^{14}$ | $9.6 \times 10^{13}$ | 96%* |
| RAID 6 | 8 × 4 TiB | $10^{15}$ | $2.24 \times 10^{14}$ | 22.4% |

*RAID 6 survives one URE; data loss requires URE on same stripe across two disks — much lower probability.*

**This is why RAID 5 is considered dangerous for large drives.** Enterprise drives ($10^{15}$) and RAID 6 mitigate this risk.

---

## 6. Sequential Throughput

### Read Throughput

$$\text{Read BW} = \text{Data Disks} \times \text{Per-Disk BW}$$

### Write Throughput

$$\text{Write BW (RAID5)} = (n - 1) \times \text{Per-Disk BW} \quad (\text{full stripe writes})$$

$$\text{Write BW (RAID5, partial)} = \frac{\text{Per-Disk BW}}{2} \quad (\text{read-modify-write bottleneck})$$

| RAID | Disks | Disk BW | Read BW | Write BW (full stripe) |
|:---:|:---:|:---:|:---:|:---:|
| 0 | 4 | 200 MB/s | 800 MB/s | 800 MB/s |
| 5 | 4 | 200 MB/s | 600 MB/s | 600 MB/s |
| 6 | 6 | 200 MB/s | 800 MB/s | 800 MB/s |
| 10 | 8 | 200 MB/s | 800 MB/s | 800 MB/s |

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| $(n - p) \times D$ | Linear arithmetic | RAID capacity |
| $\frac{\text{Raw IOPS} \times n}{\text{Penalty}}$ | Ratio | Write IOPS |
| $\frac{D}{\text{BW}}$ | Rate equation | Rebuild time |
| $\frac{\text{Bits}}{\text{URE Rate}}$ | Probability | Rebuild failure risk |
| $(n-1) \times \text{Chunk}$ | Linear | Full stripe width |
| $\text{Data Disks} \times \text{BW}$ | Linear scaling | Sequential throughput |

---

*Every `mdadm --create`, `mdadm --detail`, and `/proc/mdstat` progress bar is governed by these formulas — simple arithmetic with life-or-death implications for your data.*

## Prerequisites

- RAID level concepts (mirroring, striping, parity)
- Linux block device fundamentals (partitions, device files)
- Basic probability for failure rate calculations (MTTF/MTBF)
- Disk I/O concepts (sequential vs random, read vs write penalties)

## Complexity

- **Beginner:** RAID 1 mirror creation, basic monitoring
- **Intermediate:** RAID 5/6 sizing, rebuild time estimation, spare management
- **Advanced:** Multi-disk failure probability modeling, URE risk during rebuild, nested RAID performance analysis
