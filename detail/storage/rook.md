# The Mathematics of Rook/Ceph — Erasure Coding, CRUSH, and Placement Group Theory

> *Ceph's CRUSH algorithm deterministically maps objects to OSDs using a pseudo-random hash function and weighted bucket hierarchy, eliminating centralized lookup tables. Combined with erasure coding theory from Reed-Solomon codes, Rook/Ceph achieves configurable redundancy where storage efficiency and fault tolerance are governed by precise information-theoretic bounds.*

---

## 1. CRUSH Algorithm (Controlled Replication Under Scalable Hashing)

### The Problem

Distributing petabytes of data across hundreds of OSDs requires a placement algorithm that is deterministic (any node can compute placement independently), balanced (data spreads evenly by weight), and stable (adding/removing OSDs moves minimal data).

### The Formula

CRUSH maps an object identifier to a set of OSDs through a multi-step process. For object $x$ and placement group $\text{pg}$:

$$\text{pg}(x) = \text{hash}(x) \mod |\text{PG}|$$

Then CRUSH selects $r$ OSDs for placement group $\text{pg}$ using a weighted hash:

$$\text{OSD}_i = \text{CRUSH}(\text{pg}, i, \text{map})$$

The probability that OSD $j$ is selected is proportional to its weight $w_j$:

$$P(\text{select } j) = \frac{w_j}{\sum_{k=1}^{n} w_k}$$

When an OSD with weight $w_j$ is added to a cluster of total weight $W$, the fraction of data that moves:

$$\Delta = \frac{w_j}{W + w_j}$$

### Worked Examples

**Adding a 4TB OSD to a 100TB cluster:**

$$\Delta = \frac{4}{100 + 4} = \frac{4}{104} = 3.85\%$$

Only 3.85% of existing data migrates — compared to a naive hash modulo which would move $\frac{n-1}{n} \approx 99\%$ of data.

**Weighted placement across heterogeneous OSDs:**

Given 3 OSDs with weights $w = [4, 2, 2]$ (total $W = 8$):

$$P(\text{OSD}_1) = \frac{4}{8} = 50\%, \quad P(\text{OSD}_2) = P(\text{OSD}_3) = \frac{2}{8} = 25\%$$

A 1TB pool distributes approximately 500GB, 250GB, 250GB.

---

## 2. Erasure Coding (Reed-Solomon Theory)

### The Problem

Triple replication (3x) provides excellent durability but uses 3 bytes of raw storage per byte of user data. Erasure coding can achieve similar durability with far less overhead. What are the mathematical trade-offs?

### The Formula

An erasure code $\text{EC}(k, m)$ encodes $k$ data chunks into $k + m$ total chunks, tolerating any $m$ simultaneous chunk failures. Storage overhead:

$$\text{overhead} = \frac{k + m}{k}$$

Raw-to-usable ratio (inverse of storage efficiency):

$$\text{efficiency} = \frac{k}{k + m}$$

The code is MDS (Maximum Distance Separable) if it achieves the Singleton bound:

$$d_{\min} = n - k + 1 = m + 1$$

Where $d_{\min}$ is the minimum Hamming distance. This means any $k$ of the $n = k + m$ chunks suffice to reconstruct the original data.

Durability comparison — probability of data loss with disk failure rate $p$:

$$P_{\text{loss}}^{\text{repl}}(r) = p^r$$
$$P_{\text{loss}}^{\text{EC}}(k, m) = \sum_{i=m+1}^{k+m} \binom{k+m}{i} p^i (1-p)^{k+m-i}$$

### Worked Examples

**EC(4,2) vs 3x replication:**

Storage overhead: EC = $\frac{6}{4} = 1.5\times$, Replication = $3.0\times$

Storage savings: $\frac{3.0 - 1.5}{3.0} = 50\%$ less raw storage.

Durability with $p = 0.02$:

$$P_{\text{loss}}^{\text{repl}}(3) = 0.02^3 = 8 \times 10^{-6}$$

$$P_{\text{loss}}^{\text{EC}}(4,2) = \binom{6}{3}(0.02)^3(0.98)^3 + \binom{6}{4}(0.02)^4(0.98)^2 + \cdots$$
$$\approx 20 \times 7.53 \times 10^{-6} + 15 \times 1.54 \times 10^{-7} + \cdots \approx 1.53 \times 10^{-4}$$

EC(4,2) is less durable than 3x replication but uses half the storage. EC(4,3) closes the gap:

$$P_{\text{loss}}^{\text{EC}}(4,3) \approx \binom{7}{4}(0.02)^4(0.98)^3 \approx 35 \times 1.51 \times 10^{-7} \approx 5.28 \times 10^{-6}$$

At $1.75\times$ overhead — better durability than 3x replication at 42% less storage.

---

## 3. Placement Group Optimization (Hashing and Load Balance)

### The Problem

PGs are the intermediate grouping between objects and OSDs. Too few PGs cause uneven distribution; too many waste memory. What is the optimal PG count?

### The Formula

Ceph's recommended formula for PG count per pool:

$$\text{PG}_{\text{target}} = \left\lceil \frac{\text{OSDs} \times 100}{\text{pool\_replica\_size}} \right\rceil$$

Rounded up to the nearest power of 2 for uniform hash distribution:

$$\text{PG}_{\text{actual}} = 2^{\lceil \log_2(\text{PG}_{\text{target}}) \rceil}$$

The expected variance in data per OSD (the "imbalance" problem) follows from balls-into-bins theory. With $m$ PGs distributed across $n$ OSDs:

$$E[\text{max load}] \approx \frac{m}{n} + \sqrt{\frac{2m \ln n}{n}}$$

Relative imbalance:

$$\text{imbalance} = \frac{E[\text{max load}] - m/n}{m/n} = \sqrt{\frac{2n \ln n}{m}}$$

### Worked Examples

**30 OSDs, 3x replicated pool:**

$$\text{PG}_{\text{target}} = \frac{30 \times 100}{3} = 1000$$
$$\text{PG}_{\text{actual}} = 2^{\lceil \log_2(1000) \rceil} = 2^{10} = 1024$$

**Imbalance with 1024 PGs across 30 OSDs:**

$$\text{imbalance} = \sqrt{\frac{2 \times 30 \times \ln(30)}{1024}} = \sqrt{\frac{204}{1024}} = \sqrt{0.199} \approx 0.447$$

A 45% worst-case imbalance. With the autoscaler increasing to 4096 PGs:

$$\text{imbalance} = \sqrt{\frac{204}{4096}} = \sqrt{0.050} \approx 0.223$$

Reduced to 22% — each doubling of PG count reduces imbalance by $\sqrt{2} \approx 1.41\times$.

---

## 4. Recovery and Backfill (Markov Chain Model)

### The Problem

When an OSD fails, Ceph must rebuild data from surviving replicas or EC chunks. During recovery, the cluster operates in a degraded state. How does recovery time affect overall reliability?

### The Formula

Model the cluster as a Markov chain with states representing the number of available replicas per PG. For a 3x replicated system, states are $\{3, 2, 1, 0\}$ where state 0 is data loss.

Transition rates from state $i$ to state $i-1$ (failure rate):

$$\lambda_i = i \cdot \mu_{\text{fail}}$$

Transition rates from state $i$ to state $i+1$ (repair rate):

$$\mu_{\text{repair}} = \frac{B_{\text{rebuild}} \cdot (n_{\text{OSD}} - 1)}{S_{\text{OSD}}}$$

Mean time to data loss (MTTDL) for a 3-way replicated system:

$$\text{MTTDL} = \frac{(\mu_{\text{repair}})^2}{6 \cdot \lambda^3}$$

Where $\lambda$ is single-OSD failure rate.

### Worked Examples

**100 OSDs, each 4TB, 100 MB/s rebuild bandwidth, MTBF = 50,000 hours:**

$$\lambda = \frac{1}{50{,}000} \text{ failures/hour per OSD}$$

Mean repair time per OSD: $\frac{4 \times 10^6 \text{ MB}}{100 \text{ MB/s} \times 99} \approx 404 \text{ s} \approx 0.112 \text{ hours}$

$$\mu_{\text{repair}} = \frac{1}{0.112} = 8.93 \text{ repairs/hour}$$

$$\text{MTTDL} = \frac{8.93^2}{6 \times (100 / 50{,}000)^3} = \frac{79.7}{6 \times 8 \times 10^{-12}} = \frac{79.7}{4.8 \times 10^{-11}} \approx 1.66 \times 10^{12} \text{ hours}$$

That is approximately 190 million years of expected time before data loss.

---

## 5. Pool Capacity and Overcommit (Thin Provisioning)

### The Problem

Rook allows thin-provisioned pools where the sum of volume claims exceeds physical capacity. What is the safe overcommit ratio?

### The Formula

Overcommit ratio:

$$O = \frac{\sum_{i=1}^{v} C_i}{R_{\text{physical}}}$$

Where $C_i$ is the claimed capacity of volume $i$ and $R_{\text{physical}}$ is raw physical capacity. Effective capacity accounting for redundancy:

$$R_{\text{usable}} = \frac{R_{\text{physical}}}{\text{redundancy\_factor}}$$

Probability of capacity exhaustion depends on the distribution of actual usage. If each volume uses fraction $u_i$ of its claim, with $u_i \sim \text{Beta}(\alpha, \beta)$:

$$E[\text{total usage}] = O \cdot R_{\text{usable}} \cdot \frac{\alpha}{\alpha + \beta}$$

Safe overcommit (95th percentile usage stays below physical capacity):

$$O_{\text{safe}} \leq \frac{1}{P_{95}(u)} = \frac{\alpha + \beta}{\alpha + z_{0.95}\sqrt{\frac{\alpha\beta}{(\alpha+\beta)^2(\alpha+\beta+1)}/v}}$$

### Worked Examples

**100TB raw, 3x replication, volumes average 40% utilization ($\alpha=2, \beta=3$):**

$$R_{\text{usable}} = \frac{100}{3} = 33.3 \text{ TB}$$
$$E[u] = \frac{2}{5} = 0.4$$
$$O_{\text{safe}} \approx \frac{1}{0.4} = 2.5\times$$

Can safely provision $2.5 \times 33.3 = 83.3$ TB of claims against 33.3 TB usable. Monitor actual usage and alert at 70% physical utilization.

---

## Prerequisites

- information-theory, reed-solomon-codes, probability, markov-chains, consistent-hashing, ceph, longhorn
