# The Mathematics of Proxmox — Cluster Quorum, Ceph Placement & Backup Deduplication

> *Proxmox VE clusters depend on quorum for split-brain prevention, Ceph CRUSH for deterministic data placement, and deduplication ratios for backup storage planning. The mathematics behind these systems determines cluster sizing, storage efficiency, and recovery time objectives.*

---

## 1. Cluster Quorum (Voting Theory)

### The Problem

A Proxmox HA cluster uses Corosync for consensus. A partition must have a majority of votes to operate. What is the minimum cluster size for $f$ node failures?

### The Formula

For a cluster of $N$ nodes with equal votes, quorum requires:

$$Q = \lfloor N/2 \rfloor + 1$$

Maximum tolerated failures:

$$f_{max} = N - Q = \lceil N/2 \rceil - 1$$

Availability as a function of node reliability $p$:

$$A_{cluster} = \sum_{k=Q}^{N} \binom{N}{k} p^k (1-p)^{N-k}$$

### Worked Examples

**3-node cluster, node reliability p = 0.99:**

$$Q = 2, \quad f_{max} = 1$$

$$A = \binom{3}{2}(0.99)^2(0.01) + \binom{3}{3}(0.99)^3 = 3(0.009801)(0.01) + 0.970299$$

$$A = 0.000294 + 0.970299 = 0.999703 \quad (99.97\%)$$

**5-node cluster:**

$$Q = 3, \quad f_{max} = 2$$

$$A = \sum_{k=3}^{5} \binom{5}{k}(0.99)^k(0.01)^{5-k}$$

$$A = 0.999990 \quad (99.999\%)$$

Going from 3 to 5 nodes improves availability from 3 nines to 5 nines.

---

## 2. Ceph CRUSH Placement (Consistent Hashing)

### The Problem

Ceph distributes objects across OSDs using CRUSH (Controlled Replication Under Scalable Hashing). Given a placement group (PG) count and OSD count, how evenly are objects distributed?

### The Formula

Each PG maps to $r$ OSDs (replicas). With $P$ PGs and $O$ OSDs:

$$PGs\_per\_OSD = \frac{P \times r}{O}$$

Target: 100-200 PGs per OSD. The standard deviation of PG distribution:

$$\sigma_{PG} = \sqrt{P \times \frac{1}{O} \times \left(1 - \frac{1}{O}\right)} \approx \sqrt{\frac{P}{O}}$$

$$CV = \frac{\sigma_{PG}}{\mu_{PG}} = \frac{1}{\sqrt{P/O}} = \sqrt{\frac{O}{P}}$$

### Worked Examples

**128 PGs, 12 OSDs, 3 replicas:**

$$\mu = \frac{128 \times 3}{12} = 32 \text{ PGs/OSD}$$

$$CV = \sqrt{12/128} = 0.306 \quad (30.6\% \text{ variation -- too low PG count})$$

**256 PGs, 12 OSDs:**

$$\mu = \frac{256 \times 3}{12} = 64 \text{ PGs/OSD}$$

$$CV = \sqrt{12/256} = 0.217 \quad (21.7\% \text{ -- better})$$

**1024 PGs, 12 OSDs:**

$$\mu = \frac{1024 \times 3}{12} = 256 \text{ PGs/OSD}$$

$$CV = \sqrt{12/1024} = 0.108 \quad (10.8\% \text{ -- good uniformity})$$

Recommended PG count formula:

$$P = 2^{\lceil \log_2(O \times 100 / r) \rceil}$$

---

## 3. Ceph Recovery Time (Reliability Engineering)

### The Problem

When an OSD fails, Ceph re-replicates data. If a second OSD fails before recovery completes, data loss occurs (for $r=3$, need 3 concurrent failures in the same PG). What is the probability?

### The Formula

Mean time to data loss (MTTDL) for a replicated pool:

$$MTTDL = \frac{MTTF^r}{N^{r-1} \cdot r! \cdot T_{recovery}^{r-1}}$$

Where:
- $MTTF$ = mean time to failure of one OSD (hours)
- $N$ = number of OSDs
- $r$ = replica count
- $T_{recovery}$ = time to re-replicate one OSD (hours)

### Worked Examples

**12 OSDs, MTTF = 100,000 hours (11.4 years), r = 3, recovery = 2 hours:**

$$MTTDL = \frac{(10^5)^3}{12^2 \times 6 \times 2^2} = \frac{10^{15}}{144 \times 6 \times 4} = \frac{10^{15}}{3456}$$

$$MTTDL = 2.89 \times 10^{11} \text{ hours} = 33 \text{ million years}$$

**Same but with r = 2:**

$$MTTDL = \frac{(10^5)^2}{12 \times 2 \times 2} = \frac{10^{10}}{48} = 2.08 \times 10^8 \text{ hours} = 23,800 \text{ years}$$

Triple replication provides 1400x better durability than double.

---

## 4. Backup Deduplication Ratio (Information Theory)

### The Problem

Proxmox Backup Server deduplicates backups at the chunk level. Given $B$ daily backups of VMs with change rate $c$ per day, what is the expected deduplication ratio?

### The Formula

For $B$ full backups of size $S$, with daily change rate $c$ (fraction of unique chunks):

$$S_{raw} = B \times S$$

$$S_{dedup} = S + (B-1) \times c \times S = S(1 + (B-1)c)$$

$$R_{dedup} = \frac{S_{raw}}{S_{dedup}} = \frac{B}{1 + (B-1)c}$$

### Worked Examples

**30 daily backups, 5% daily change rate, 100GB VM:**

$$S_{raw} = 30 \times 100 = 3000 \text{ GB}$$

$$S_{dedup} = 100(1 + 29 \times 0.05) = 100 \times 2.45 = 245 \text{ GB}$$

$$R_{dedup} = 3000/245 = 12.2:1$$

**30 backups, 20% daily change (database server):**

$$S_{dedup} = 100(1 + 29 \times 0.20) = 680 \text{ GB}$$

$$R_{dedup} = 3000/680 = 4.4:1$$

Low change-rate workloads (web servers, DNS) benefit enormously from deduplication.

---

## 5. VM Density and Overcommit (Statistical Multiplexing)

### The Problem

Proxmox allows overcommitting CPU and memory. How many VMs can safely run on a host given usage distributions?

### The Formula

If each VM's CPU usage $U_i \sim N(\mu_i, \sigma_i^2)$ independently, total usage:

$$U_{total} \sim N\left(\sum \mu_i, \sum \sigma_i^2\right)$$

For $N$ identical VMs with overcommit ratio $R$:

$$P(\text{contention}) = P\left(U_{total} > C\right) = \Phi\left(\frac{N\mu - C}{\sigma\sqrt{N}}\right)$$

Where $C$ = host capacity, $R = N \cdot \mu_{config} / C$.

### Worked Examples

**Host: 32 cores, 20 VMs each configured 4 vCPUs (R = 2.5x), mean usage 15%, stddev 10%:**

$$\mu_{total} = 20 \times 4 \times 0.15 = 12 \text{ cores}$$

$$\sigma_{total} = \sqrt{20} \times 4 \times 0.10 = 1.789 \text{ cores}$$

$$P(\text{contention}) = \Phi\left(\frac{12 - 32}{1.789}\right) = \Phi(-11.2) \approx 0$$

**40 VMs (R = 5x):**

$$\mu_{total} = 40 \times 0.6 = 24, \quad \sigma_{total} = \sqrt{40} \times 0.4 = 2.53$$

$$P = \Phi\left(\frac{24 - 32}{2.53}\right) = \Phi(-3.16) = 0.0008 \quad (0.08\%)$$

Statistical multiplexing allows significant overcommit for low-utilization VMs.

---

## 6. HA Failover Time Budget (SLA Mathematics)

### The Problem

HA failover involves failure detection, fencing, and VM restart. What is the expected downtime per incident and annual downtime?

### The Formula

$$T_{failover} = T_{detect} + T_{fence} + T_{boot}$$

Annual downtime:

$$D_{annual} = f_{failure} \times T_{failover}$$

Availability:

$$A = 1 - \frac{D_{annual}}{T_{year}} = 1 - \frac{f_{failure} \times T_{failover}}{8760 \times 3600}$$

### Worked Examples

**Detection: 30s, fencing: 10s, boot: 45s, 2 failures/year:**

$$T_{failover} = 30 + 10 + 45 = 85 \text{ seconds}$$

$$D_{annual} = 2 \times 85 = 170 \text{ seconds} = 2.83 \text{ minutes}$$

$$A = 1 - \frac{170}{31536000} = 99.99946\%$$

**With watchdog fencing (2s) and fast boot (15s):**

$$T_{failover} = 30 + 2 + 15 = 47s, \quad D_{annual} = 94s$$

$$A = 99.99970\%$$

---

## Prerequisites

- voting-theory, majority-quorum
- consistent-hashing, uniform-distribution
- reliability-engineering, MTTF, MTTDL
- information-theory, compression-ratio
- normal-distribution, central-limit-theorem
