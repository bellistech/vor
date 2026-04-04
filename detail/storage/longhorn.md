# The Mathematics of Longhorn — Replica Consistency and Distributed Block Storage

> *Longhorn's architecture decomposes distributed storage into per-volume microservice engines, each managing independent replica chains. The consistency model relies on chain replication with synchronous writes, making the trade-offs between replica count, write latency, and failure tolerance expressible through queuing theory and combinatorial reliability models.*

---

## 1. Replica Reliability (Combinatorial Probability)

### The Problem

Longhorn replicates each volume across $r$ replicas on different nodes. A volume is available if at least one healthy replica exists. What is the probability of data loss as a function of replica count and individual disk failure rate?

### The Formula

Let $p$ be the probability of a single replica failure in a given time window. Replicas fail independently. Volume data loss occurs when all $r$ replicas fail simultaneously:

$$P(\text{data loss}) = p^r$$

Volume availability (at least one replica survives):

$$A = 1 - p^r$$

Number of nines of durability:

$$\text{nines} = -\log_{10}(p^r) = -r \cdot \log_{10}(p)$$

### Worked Examples

**3 replicas, annual disk failure rate 2%** ($p = 0.02$):

$$P(\text{loss}) = 0.02^3 = 8 \times 10^{-6}$$

$$\text{nines} = -3 \times \log_{10}(0.02) = -3 \times (-1.699) = 5.097$$

That is 5 nines of annual durability (99.9992%).

**Increasing to 4 replicas:**

$$P(\text{loss}) = 0.02^4 = 1.6 \times 10^{-7} \quad \Rightarrow \quad 6.8 \text{ nines}$$

**Time-dependent model with repair:** If mean time to repair (MTTR) is $T_r$ and mean time between failures (MTBF) is $T_f$, the steady-state unavailability of a single replica:

$$U = \frac{T_r}{T_f + T_r} \approx \frac{T_r}{T_f} \quad \text{(when } T_f \gg T_r\text{)}$$

Volume unavailability (all replicas down):

$$U_{\text{vol}} = U^r = \left(\frac{T_r}{T_f}\right)^r$$

With $T_f = 50{,}000$ hours, $T_r = 4$ hours, $r = 3$:

$$U_{\text{vol}} = \left(\frac{4}{50{,}000}\right)^3 = (8 \times 10^{-5})^3 = 5.12 \times 10^{-13}$$

---

## 2. Chain Replication (Write Propagation)

### The Problem

Longhorn uses a chain replication model where writes propagate sequentially through replicas. How does chain length affect write latency and throughput?

### The Formula

In chain replication with $r$ replicas, write latency is the sum of sequential propagation delays:

$$L_{\text{write}} = \sum_{i=1}^{r} (d_{\text{net},i} + d_{\text{disk},i})$$

Where $d_{\text{net},i}$ is network latency to replica $i$ and $d_{\text{disk},i}$ is disk write latency at replica $i$.

For homogeneous replicas with average network latency $\bar{d}_n$ and disk latency $\bar{d}_d$:

$$L_{\text{write}} = r \cdot (\bar{d}_n + \bar{d}_d)$$

Write throughput is bounded by the slowest replica in the chain:

$$T_{\text{write}} = \min_{i \in [1,r]} \frac{B}{d_{\text{disk},i}}$$

Where $B$ is the block size. Read latency (served from any replica, typically head):

$$L_{\text{read}} = d_{\text{net},1} + d_{\text{disk},1}$$

### Worked Examples

**3 replicas, 0.5ms network, 2ms disk write:**

$$L_{\text{write}} = 3 \times (0.5 + 2.0) = 7.5 \text{ ms}$$
$$L_{\text{read}} = 0.5 + 2.0 = 2.5 \text{ ms}$$

Write/read latency ratio: $\frac{7.5}{2.5} = 3\times$ (linear in replica count).

**With data locality (one replica is local, $d_{\text{net},1} = 0$):**

$$L_{\text{write}} = (0 + 2.0) + 2 \times (0.5 + 2.0) = 2.0 + 5.0 = 7.0 \text{ ms}$$
$$L_{\text{read}} = 0 + 2.0 = 2.0 \text{ ms} \quad (20\% \text{ improvement})$$

---

## 3. Snapshot Space Efficiency (Copy-on-Write)

### The Problem

Longhorn snapshots use copy-on-write (CoW). Each snapshot only stores blocks that changed since the previous snapshot. How does space consumption grow with snapshot frequency?

### The Formula

Let $S$ be the total volume size and $\delta$ the fraction of blocks modified between consecutive snapshots (change rate). After $n$ snapshots:

$$\text{Space}_{\text{total}} = S + \sum_{i=1}^{n} \delta_i \cdot S$$

With constant change rate $\delta$:

$$\text{Space}_{\text{total}} = S(1 + n\delta)$$

Space amplification factor:

$$\alpha = \frac{\text{Space}_{\text{total}}}{S} = 1 + n\delta$$

### Worked Examples

**100 GB volume, 5% daily change rate, 7 daily snapshots:**

$$\alpha = 1 + 7 \times 0.05 = 1.35$$
$$\text{Space} = 100 \times 1.35 = 135 \text{ GB}$$

**With retention policy (keep last 7, delete older):** Space is bounded:

$$\text{Space}_{\text{max}} = S(1 + 7 \times \delta) = 135 \text{ GB (always)}$$

**High-churn database, 20% daily change, 30-day retention:**

$$\alpha = 1 + 30 \times 0.20 = 7.0$$
$$\text{Space} = 100 \times 7.0 = 700 \text{ GB}$$

This demonstrates why retention policies are critical for high-churn workloads.

---

## 4. Rebuild Bandwidth (Recovery Time)

### The Problem

When a replica is lost, Longhorn must rebuild from a surviving replica. How long does rebuild take, and what is the impact on foreground I/O?

### The Formula

Rebuild time for a volume of size $S$ at rebuild bandwidth $B_r$:

$$T_{\text{rebuild}} = \frac{S}{B_r}$$

If foreground I/O consumes bandwidth $B_f$ and total link capacity is $B_{\text{total}}$:

$$B_r = B_{\text{total}} - B_f$$

During rebuild, foreground I/O latency increases due to bandwidth contention. Effective foreground throughput:

$$T_f' = \frac{B_f}{B_f + B_r} \cdot T_f = \frac{B_f}{B_{\text{total}}} \cdot T_f$$

### Worked Examples

**500 GB volume, 1 Gbps network (125 MB/s), foreground I/O at 50 MB/s:**

$$B_r = 125 - 50 = 75 \text{ MB/s}$$
$$T_{\text{rebuild}} = \frac{500 \times 1024}{75} = 6{,}827 \text{ s} \approx 1.9 \text{ hours}$$

**10 Gbps network, same workload:**

$$B_r = 1250 - 50 = 1200 \text{ MB/s}$$
$$T_{\text{rebuild}} = \frac{512{,}000}{1200} = 427 \text{ s} \approx 7.1 \text{ minutes}$$

This is why 10 Gbps networking is strongly recommended for production Longhorn clusters.

---

## 5. IOPS Capacity Planning (Queuing Theory)

### The Problem

Given a Longhorn node with $d$ disks, each capable of $I_{\text{max}}$ IOPS, how many volumes can the node serve without saturating I/O capacity?

### The Formula

Using M/M/1 queuing model, the mean I/O latency as utilization $\rho$ approaches 1:

$$E[L] = \frac{1}{\mu(1 - \rho)}$$

Where $\mu$ is the service rate (max IOPS per disk) and $\rho = \lambda / \mu$ is utilization. Total node IOPS capacity across $d$ disks:

$$I_{\text{node}} = d \cdot I_{\text{max}}$$

With $v$ volumes each generating $\lambda_v$ IOPS, and each write replicated $r$ times (only local replica counts):

$$\rho_{\text{node}} = \frac{\sum_{j=1}^{v} \lambda_{v,j}}{I_{\text{node}}}$$

To maintain latency below a threshold (e.g., $\rho < 0.7$):

$$v_{\text{max}} = \left\lfloor \frac{0.7 \cdot I_{\text{node}}}{\bar{\lambda}_v} \right\rfloor$$

### Worked Examples

**4 NVMe disks at 100K IOPS each, volumes averaging 500 IOPS:**

$$I_{\text{node}} = 4 \times 100{,}000 = 400{,}000 \text{ IOPS}$$
$$v_{\text{max}} = \left\lfloor \frac{0.7 \times 400{,}000}{500} \right\rfloor = 560 \text{ volumes}$$

**At 80% utilization, mean latency (service time 10 us):**

$$E[L] = \frac{1}{100{,}000 \times (1 - 0.8)} = \frac{1}{20{,}000} = 50 \text{ }\mu\text{s}$$

At 95% utilization: $E[L] = 200 \text{ }\mu\text{s}$ — a $4\times$ latency increase from 15% more load.

---

## Prerequisites

- probability, queuing-theory, distributed-systems, chain-replication, copy-on-write, ceph
