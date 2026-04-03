# The Mathematics of LVM — Logical Volume Management Internals

> *LVM abstracts physical storage into logical pools using extent-based allocation. The math centers on extent arithmetic, thin provisioning ratios, snapshot overhead, and data migration costs.*

---

## 1. Extent Arithmetic — The Fundamental Unit

### The Model

LVM divides Physical Volumes (PVs) into fixed-size **Physical Extents (PEs)**. Logical Volumes (LVs) are built from **Logical Extents (LEs)**, mapped 1:1 to PEs.

### The Formulas

$$\text{LV Size} = \text{LE Count} \times \text{PE Size}$$

$$\text{PE Count per PV} = \left\lfloor \frac{\text{PV Size} - \text{Metadata Size}}{\text{PE Size}} \right\rfloor$$

$$\text{VG Total Capacity} = \sum_{i=1}^{n} \text{PE Count}_i \times \text{PE Size}$$

Where:
- Default PE size = **4 MiB** (configurable: 1 MiB to 16 GiB, power of 2)
- Metadata overhead ~= 1 MiB per PV (LVM2 metadata area)

### Worked Examples

| PV Size | PE Size | Metadata | Usable PEs | Usable Capacity |
|:---:|:---:|:---:|:---:|:---:|
| 100 GiB | 4 MiB | 1 MiB | 25,599 | 99.996 GiB |
| 500 GiB | 4 MiB | 1 MiB | 127,999 | 499.996 GiB |
| 1 TiB | 4 MiB | 1 MiB | 262,143 | 1023.996 GiB |
| 1 TiB | 16 MiB | 1 MiB | 65,535 | 1023.984 GiB |

### PE Size Trade-offs

$$\text{Max LV Size} = \text{PE Size} \times 2^{32}$$

| PE Size | Max LV Size | PEs per 1 TiB | Metadata per PE |
|:---:|:---:|:---:|:---:|
| 4 MiB | 16 TiB | 262,144 | ~128 bytes |
| 8 MiB | 32 TiB | 131,072 | ~128 bytes |
| 16 MiB | 64 TiB | 65,536 | ~128 bytes |
| 64 MiB | 256 TiB | 16,384 | ~128 bytes |

Larger PE = higher max LV size but coarser granularity (wasted space on small LVs).

---

## 2. Thin Provisioning — Overcommit Mathematics

### The Model

Thin pools allocate blocks on demand, allowing **overcommit** — advertising more space than physically exists.

### Overcommit Ratio

$$\text{Overcommit Ratio} = \frac{\text{Total Thin LV Virtual Size}}{\text{Thin Pool Physical Size}}$$

$$\text{Actual Usage Rate} = \frac{\text{Allocated Blocks}}{\text{Total Virtual Blocks}}$$

### Capacity Planning

$$\text{Time to Full} = \frac{\text{Pool Free Space}}{\text{Write Rate}}$$

$$\text{Required Physical} = \frac{\sum \text{Virtual Sizes} \times \text{Expected Fill Rate}}{\text{Overcommit Ratio}}$$

### Worked Example

*"3 thin LVs of 500 GiB each on a 500 GiB pool. Each fills at 2 GiB/day."*

$$\text{Overcommit} = \frac{3 \times 500}{500} = 3:1$$

$$\text{Time to Full} = \frac{500 \text{ GiB}}{3 \times 2 \text{ GiB/day}} = 83.3 \text{ days}$$

| Overcommit | Virtual Total | Physical Pool | Risk Level | Use Case |
|:---:|:---:|:---:|:---:|:---|
| 1:1 | 500 GiB | 500 GiB | None | Production databases |
| 2:1 | 1 TiB | 500 GiB | Low | General servers |
| 5:1 | 2.5 TiB | 500 GiB | Medium | Dev environments |
| 10:1 | 5 TiB | 500 GiB | High | CI/CD ephemeral |
| 20:1 | 10 TiB | 500 GiB | Critical | Test-only |

**Alert threshold formula:** Set monitoring at 80% pool usage:

$$\text{Alert at} = \text{Pool Size} \times 0.80$$

---

## 3. Snapshot COW Overhead

### The Model

LVM snapshots use **Copy-on-Write (COW)**. When the origin LV writes a block, the old data is copied to the snapshot volume first.

### Snapshot Size Formula

$$\text{Snapshot Space Needed} = \text{Change Rate} \times \text{Snapshot Lifetime} \times \text{COW Overhead}$$

Where COW overhead = 1.0 for LVM (no compression). The snapshot must hold every original block that gets overwritten.

$$\text{Min Snapshot Size} = \text{Origin Size} \times \text{Change Fraction}$$

### Worked Example

*"200 GiB origin LV, 5% daily change rate, keep snapshot for 7 days."*

$$\text{Snapshot Size} = 200 \times 0.05 \times 7 = 70 \text{ GiB}$$

But blocks may be overwritten multiple times (only first write copies):

$$\text{Unique Changed Blocks} = \text{Origin Size} \times \left(1 - (1 - p)^d\right)$$

Where $p$ = daily change probability per block, $d$ = days.

For $p = 0.05$, $d = 7$:

$$\text{Unique Changed} = 200 \times (1 - 0.95^7) = 200 \times 0.302 = 60.4 \text{ GiB}$$

| Daily Change Rate | 1 Day | 3 Days | 7 Days | 14 Days |
|:---:|:---:|:---:|:---:|:---:|
| 1% | 2.0 GiB | 5.9 GiB | 13.6 GiB | 26.1 GiB |
| 5% | 10.0 GiB | 28.6 GiB | 60.4 GiB | 103.4 GiB |
| 10% | 20.0 GiB | 53.1 GiB | 104.9 GiB | 157.1 GiB |
| 20% | 40.0 GiB | 92.8 GiB | 158.3 GiB | 200.0 GiB |

**Critical:** If a snapshot fills 100%, it becomes **invalid** and is dropped.

---

## 4. pvmove Data Migration

### The Model

`pvmove` relocates extents from one PV to another. The time depends on extent count, I/O bandwidth, and whether the system is under load.

### Migration Time Formula

$$T_{migrate} = \frac{\text{Data Size}}{\text{Effective Bandwidth}}$$

$$\text{Effective Bandwidth} = \text{Disk Sequential Write} \times (1 - \text{System I/O Fraction})$$

### Worked Example

*"Move 500 GiB from an HDD (150 MB/s) to an SSD, system using 30% I/O."*

$$\text{Effective BW} = 150 \times (1 - 0.30) = 105 \text{ MB/s}$$

$$T = \frac{500 \times 1024}{105} = 4,876 \text{ sec} \approx 81 \text{ min}$$

| Data Size | HDD (150 MB/s) | SSD (500 MB/s) | NVMe (2 GB/s) |
|:---:|:---:|:---:|:---:|
| 100 GiB | 11.4 min | 3.4 min | 0.85 min |
| 500 GiB | 56.9 min | 17.1 min | 4.3 min |
| 1 TiB | 116.5 min | 34.1 min | 8.5 min |
| 4 TiB | 466.0 min | 136.5 min | 34.1 min |

*Times assume 0% competing I/O — multiply by $\frac{1}{1 - \text{load}}$ for real-world estimates.*

---

## 5. Striped LV Performance

### The Model

Striped LVs distribute data across multiple PVs for parallel I/O, similar to RAID-0.

### Throughput Formula

$$\text{Throughput}_{striped} = \min\left(n \times \text{BW}_{single}, \text{BW}_{controller}\right)$$

Where $n$ = stripe count (number of PVs).

### IOPS Formula

$$\text{IOPS}_{striped} = n \times \text{IOPS}_{single}$$

### Optimal Stripe Size

Rule of thumb: stripe size should match the dominant I/O pattern.

| Workload | Recommended Stripe | Reasoning |
|:---|:---:|:---|
| Sequential (video, backup) | 256 KiB - 1 MiB | Large blocks, fewer seeks |
| Database (random 8K) | 64 KiB | Align to DB page size |
| VM images | 128 KiB | Mixed access patterns |
| General purpose | 64 KiB | Default, balanced |

---

## 6. RAID LV Parity Overhead

### LVM RAID Capacity

$$\text{Usable} = (\text{Devices} - \text{Parity Devices}) \times \text{Smallest Device}$$

| RAID Level | Parity Devices | Usable Fraction | Min Devices |
|:---:|:---:|:---:|:---:|
| raid0 | 0 | $\frac{n}{n} = 100\%$ | 2 |
| raid1 | $n - 1$ | $\frac{1}{n}$ | 2 |
| raid5 | 1 | $\frac{n-1}{n}$ | 3 |
| raid6 | 2 | $\frac{n-2}{n}$ | 4 |
| raid10 | $n/2$ | $\frac{n}{2n} = 50\%$ | 4 |

### Write Penalty

$$\text{Effective Write IOPS} = \frac{\text{Raw IOPS}}{\text{Write Penalty}}$$

| RAID Level | Write Penalty | Explanation |
|:---:|:---:|:---|
| raid0 | 1 | No parity |
| raid1 | 2 | Mirror write |
| raid5 | 4 | Read old data + parity, write new data + parity |
| raid6 | 6 | Two parity calculations |
| raid10 | 2 | Mirror write |

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| $\text{LEs} \times \text{PE Size}$ | Linear arithmetic | Volume sizing |
| $\frac{\text{Virtual}}{\text{Physical}}$ | Ratio | Thin overcommit |
| $1 - (1-p)^d$ | Geometric probability | Snapshot sizing |
| $\frac{\text{Size}}{\text{BW}}$ | Rate equation | Migration time |
| $n \times \text{BW}_{single}$ | Linear scaling | Stripe throughput |
| $\frac{n - k}{n}$ | Fraction | RAID usable capacity |

---

*Every `lvcreate`, `lvextend`, and `pvmove` you run is executing these extent calculations in the device-mapper kernel subsystem.*
