# The Mathematics of Ceph — Distributed Storage Internals

> *Ceph distributes data across a cluster using CRUSH (Controlled Replication Under Scalable Hashing). The math covers placement group mapping, replication overhead, recovery bandwidth, and capacity planning.*

---

## 1. CRUSH Algorithm — Deterministic Placement

### The Model

CRUSH maps objects to OSDs (Object Storage Daemons) without a central lookup table. It uses a pseudo-random hash function with a hierarchical cluster map.

### Placement Group (PG) Mapping

$$\text{PG ID} = \text{hash}(\text{pool\_id}, \text{object\_name}) \mod \text{PG Count}$$

$$\text{OSD Set} = \text{CRUSH}(\text{PG ID}, \text{CRUSH Map}, \text{Replication Rule})$$

### Recommended PG Count Formula

$$\text{PG Count} = \frac{\text{OSDs} \times 100}{\text{Replication Factor}}$$

Round up to nearest power of 2.

### Worked Examples

| OSDs | Replication | Calculated | Rounded (power of 2) |
|:---:|:---:|:---:|:---:|
| 10 | 3 | 333 | 512 |
| 30 | 3 | 1,000 | 1,024 |
| 100 | 3 | 3,333 | 4,096 |
| 300 | 3 | 10,000 | 16,384 |
| 1,000 | 3 | 33,333 | 32,768 |

### PGs per OSD

$$\text{PGs per OSD} = \frac{\text{Total PGs across all pools} \times \text{Replication Factor}}{\text{Total OSDs}}$$

**Target:** 100-200 PGs per OSD. Too few = uneven distribution. Too many = excessive memory and peering overhead.

$$\text{Memory per PG} \approx 2-4 \text{ MiB (OSD process)}$$

| PGs/OSD | RAM per OSD (at 3 MiB/PG) | Status |
|:---:|:---:|:---|
| 50 | 150 MiB | Under-distributed |
| 100 | 300 MiB | Optimal |
| 200 | 600 MiB | Acceptable |
| 500 | 1.5 GiB | Excessive |
| 1000 | 3.0 GiB | Dangerous |

---

## 2. Replication and Erasure Coding Overhead

### Replication (Default: 3x)

$$\text{Raw Required} = \text{Logical Data} \times \text{Replication Factor}$$

$$\text{Usable Capacity} = \frac{\text{Raw Capacity}}{\text{Replication Factor}}$$

$$\text{Storage Efficiency} = \frac{1}{\text{Replication Factor}} \times 100\%$$

| Replication | Raw for 10 TiB Logical | Efficiency |
|:---:|:---:|:---:|
| 2x | 20 TiB | 50% |
| 3x | 30 TiB | 33% |
| 4x | 40 TiB | 25% |

### Erasure Coding (EC)

EC splits data into $k$ data chunks and $m$ parity chunks:

$$\text{Raw Required} = \text{Logical Data} \times \frac{k + m}{k}$$

$$\text{Storage Efficiency} = \frac{k}{k + m} \times 100\%$$

$$\text{Can Tolerate} = m \text{ simultaneous OSD failures}$$

| Profile ($k+m$) | Efficiency | Overhead | Fault Tolerance |
|:---:|:---:|:---:|:---:|
| 2+1 | 66.7% | 1.5x | 1 OSD |
| 4+2 | 66.7% | 1.5x | 2 OSDs |
| 8+3 | 72.7% | 1.375x | 3 OSDs |
| 8+4 | 66.7% | 1.5x | 4 OSDs |
| 16+4 | 80.0% | 1.25x | 4 OSDs |

### EC vs Replication Comparison

For 100 TiB logical data:

| Method | Raw Storage | Efficiency | Min Recovery Read |
|:---|:---:|:---:|:---|
| 3x replication | 300 TiB | 33.3% | 1 full copy |
| EC 4+2 | 150 TiB | 66.7% | 4 chunks |
| EC 8+3 | 137.5 TiB | 72.7% | 8 chunks |

---

## 3. Recovery and Backfill Bandwidth

### Recovery Time Formula

$$T_{recovery} = \frac{\text{Data per OSD}}{\text{Recovery Bandwidth}}$$

$$\text{Data per OSD} = \frac{\text{Total Data} \times \text{Replication Factor}}{\text{OSDs}}$$

### Recovery Bandwidth

$$\text{Recovery BW} = \min(\text{osd\_recovery\_max\_active} \times \text{per-thread BW}, \text{Network BW}, \text{Disk BW})$$

Default: `osd_recovery_max_active = 3`, `osd_recovery_op_priority = 3` (low).

### Worked Example

*"100 OSDs, each 4 TiB HDD, 3x replication. One OSD dies. Recovery capped at 100 MB/s per OSD."*

$$\text{Data on Failed OSD} = \frac{\text{Total Logical} \times 3}{100} = \frac{133 \text{ TiB} \times 3}{100} = 4 \text{ TiB}$$

This data is distributed across other OSDs. Each source OSD contributes:

$$\text{Per-Source Recovery} \approx \frac{4 \text{ TiB}}{99 \text{ OSDs}} = 41.4 \text{ GiB per source}$$

At 100 MB/s per source, but bottleneck is destination:

$$T = \frac{4 \text{ TiB}}{100 \text{ MB/s}} = 40,960 \text{ sec} \approx 11.4 \text{ hours}$$

| Cluster Size | OSD Size | Data/OSD | At 100 MB/s | At 500 MB/s |
|:---:|:---:|:---:|:---:|:---:|
| 20 OSDs | 4 TiB | 4 TiB | 11.4 hr | 2.3 hr |
| 100 OSDs | 8 TiB | 8 TiB | 22.8 hr | 4.6 hr |
| 100 OSDs | 16 TiB | 16 TiB | 45.5 hr | 9.1 hr |
| 500 OSDs | 16 TiB | 16 TiB | 45.5 hr | 9.1 hr |

**Risk window:** During recovery, another OSD failure may cause data loss (for replication=3, need 2 more failures on same PGs).

---

## 4. Network Bandwidth Requirements

### Write Path Bandwidth

For 3x replication, a client write generates:

$$\text{Network Write} = \text{Client Write} \times (\text{Replication Factor} - 1 + 1)$$

Primary OSD receives from client (1x), then replicates to secondaries (2x):

$$\text{Total Network} = 3 \times \text{Client Throughput}$$

### Minimum Network Sizing

$$\text{Network per OSD} \geq \text{OSD Disk BW} \times \text{Replication Factor}$$

| OSD Type | Disk BW | 3x Replication | Recommended NIC |
|:---|:---:|:---:|:---|
| HDD | 150 MB/s | 450 MB/s | 10 GbE |
| SATA SSD | 500 MB/s | 1.5 GB/s | 25 GbE |
| NVMe SSD | 3 GB/s | 9 GB/s | 100 GbE |

---

## 5. IOPS and Latency Model

### Replication Write Latency

$$T_{write} = T_{network} + \max(T_{journal\_primary}, T_{journal\_secondary})$$

For BlueStore with WAL on NVMe:

$$T_{write} \approx 0.1\text{ms (net)} + 0.1\text{ms (NVMe WAL)} = 0.2\text{ms}$$

### Aggregate IOPS

$$\text{Cluster IOPS}_{read} = \text{OSDs} \times \text{IOPS per OSD}$$

$$\text{Cluster IOPS}_{write} = \frac{\text{OSDs} \times \text{IOPS per OSD}}{\text{Replication Factor}}$$

| OSDs | OSD Type | IOPS/OSD | Read IOPS | Write IOPS (3x) |
|:---:|:---|:---:|:---:|:---:|
| 30 | HDD | 150 | 4,500 | 1,500 |
| 30 | SATA SSD | 20,000 | 600,000 | 200,000 |
| 30 | NVMe | 100,000 | 3,000,000 | 1,000,000 |

---

## 6. Capacity Planning Formula

### Usable Capacity with Overhead

$$\text{Usable} = \text{Raw} \times (1 - \text{Reserved\%}) \times \frac{1}{\text{Replication}} \times (1 - \text{Safety Margin})$$

Where:
- Reserved% = OSD journal/WAL space (~2-5%)
- Safety Margin = headroom to avoid full OSDs (typically 15-25%)

### Worked Example

*"50 OSDs × 8 TiB each, 3x replication, 5% reserved, 20% safety margin."*

$$\text{Raw} = 50 \times 8 = 400 \text{ TiB}$$

$$\text{Usable} = 400 \times 0.95 \times \frac{1}{3} \times 0.80 = 101.3 \text{ TiB}$$

**OSD fullness thresholds:**
- `nearfull_ratio` = 0.85 (warning)
- `full_ratio` = 0.95 (read-only)
- `backfillfull_ratio` = 0.90 (stops backfill)

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| $\text{hash} \mod \text{PGs}$ | Modular arithmetic | PG placement |
| $\frac{\text{OSDs} \times 100}{R}$ | Linear scaling | PG count |
| $\frac{k}{k+m}$ | Fraction | EC efficiency |
| $\frac{\text{Data}}{\text{BW}}$ | Rate equation | Recovery time |
| $\frac{\text{Raw}}{R} \times (1-m)$ | Compound fraction | Usable capacity |
| $\text{OSDs} \times \text{IOPS}$ | Linear scaling | Cluster performance |

---

*Every `ceph osd pool create`, `ceph pg dump`, and `ceph -s` you run reflects these CRUSH calculations — a pseudo-random, deterministic algorithm that replaces centralized metadata lookup with math.*

## Prerequisites

- Distributed systems fundamentals (replication, consistency, failure domains)
- Networking concepts (cluster communication, latency estimation)
- Linux storage basics (block devices, filesystems)
- Hashing algorithms (CRUSH map is hash-based placement)

## Complexity

- **Beginner:** Cluster health monitoring, basic pool creation
- **Intermediate:** CRUSH rule design, PG calculation, replication factor tuning
- **Advanced:** Recovery I/O modeling, CRUSH weight optimization, erasure coding overhead analysis
