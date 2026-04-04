# The Mathematics of Hadoop — Distributed Storage Theory

> *Hadoop's HDFS applies information theory and reliability mathematics to distribute data across commodity nodes. Understanding block placement probability, replication reliability, and NameNode memory modeling lets you capacity-plan clusters from first principles.*

---

## 1. Block Placement and Replication (Combinatorics)

### The Problem

Given N DataNodes across R racks, how does HDFS place r replicas to maximize fault tolerance while minimizing cross-rack network traffic?

### The Formula

$$P(\text{data loss}) = P(\text{all } r \text{ replicas fail simultaneously})$$

For independent node failures with probability $p$:

$$P(\text{block loss}) = p^r$$

With rack awareness (replicas on $k$ distinct racks):

$$P(\text{block loss from rack failure}) = P(\text{rack fail})^k$$

### Replica Placement Policy

HDFS default policy for $r = 3$:

$$\text{Replica 1: local rack, local node}$$
$$\text{Replica 2: remote rack, random node}$$
$$\text{Replica 3: remote rack (same as 2), different node}$$

Cross-rack writes needed:

$$W_{cross} = r - \lceil r/2 \rceil = 3 - 2 = 1$$

### Worked Examples

| Nodes | Racks | Replication | Node Fail Rate | Block Loss Prob |
|:---:|:---:|:---:|:---:|:---:|
| 10 | 2 | 3 | 0.01 | $10^{-6}$ |
| 100 | 10 | 3 | 0.01 | $10^{-6}$ |
| 100 | 10 | 2 | 0.01 | $10^{-4}$ |
| 1000 | 50 | 3 | 0.02 | $8 \times 10^{-6}$ |

---

## 2. Storage Capacity Planning (Arithmetic)

### The Problem

How much raw disk is needed for a given amount of logical data, accounting for replication, intermediate storage, and overhead?

### The Formula

$$D_{raw} = D_{logical} \times r \times (1 + O_{overhead})$$

Where:
- $D_{logical}$ = logical data size
- $r$ = replication factor
- $O_{overhead}$ = overhead fraction (typically 0.25-0.35 for logs, temp, spill)

Usable capacity per node:

$$C_{usable} = C_{disk} \times U_{threshold}$$

Where $U_{threshold}$ is typically 0.75-0.85 (leave headroom for rebalancing).

Number of DataNodes:

$$N_{nodes} = \left\lceil \frac{D_{raw}}{C_{usable}} \right\rceil$$

### Worked Examples

| Logical Data | Replication | Overhead | Raw Needed | Disk/Node | Nodes |
|:---:|:---:|:---:|:---:|:---:|:---:|
| 100 TB | 3 | 30% | 390 TB | 12 TB (80%) | 41 |
| 1 PB | 3 | 25% | 3.75 PB | 24 TB (80%) | 196 |
| 10 PB | 2 | 30% | 26 PB | 48 TB (85%) | 638 |

---

## 3. NameNode Memory Model (Linear Scaling)

### The Problem

The NameNode stores all metadata in RAM. How much heap memory is needed for a given filesystem?

### The Formula

$$M_{NN} = N_{files} \times B_{file} + N_{blocks} \times B_{block}$$

Where:
- $N_{files}$ = number of files and directories
- $B_{file}$ = memory per file entry (~150 bytes)
- $N_{blocks}$ = total blocks (including replicas counted once in namespace)
- $B_{block}$ = memory per block entry (~150 bytes)

Blocks per file:

$$N_{blocks/file} = \left\lceil \frac{S_{file}}{S_{block}} \right\rceil$$

### Worked Examples

| Files | Avg Size | Block Size | Blocks | NN Memory |
|:---:|:---:|:---:|:---:|:---:|
| 1M | 256 MB | 128 MB | 2M | ~450 MB |
| 10M | 128 MB | 128 MB | 10M | ~3 GB |
| 100M | 64 MB | 128 MB | 100M | ~30 GB |
| 100M | 1 MB | 128 MB | 100M | ~30 GB |

The small-files problem: 100M files of 1 MB each waste the same NN memory as 100M files of 128 MB each, but store 128x less data.

---

## 4. MapReduce Shuffle Cost (I/O Analysis)

### The Problem

MapReduce shuffle is often the bottleneck. What determines shuffle data volume and sort cost?

### The Formula

Total shuffle volume:

$$S_{shuffle} = \sum_{i=1}^{M} O_{map_i} = M \times \bar{O}_{map}$$

Where:
- $M$ = number of map tasks
- $\bar{O}_{map}$ = average map output size

Sort merge cost per reducer:

$$T_{sort} = O\left(\frac{S_r}{B} \log_B \frac{S_r}{B}\right)$$

Where:
- $S_r$ = data arriving at one reducer = $S_{shuffle} / R$
- $B$ = sort buffer size (`mapreduce.task.io.sort.mb`)
- $R$ = number of reducers

### Optimal Reducer Count

$$R_{optimal} = \left\lceil \frac{S_{shuffle}}{S_{reducer\_target}} \right\rceil$$

Where $S_{reducer\_target}$ is typically 1-2x HDFS block size for balanced work.

### Worked Examples

| Map Output Total | Reducers | Data/Reducer | Sort Buffer | Merge Passes |
|:---:|:---:|:---:|:---:|:---:|
| 100 GB | 50 | 2 GB | 256 MB | $\log_{256M}(2G) \approx 3$ |
| 1 TB | 200 | 5 GB | 512 MB | $\log_{512M}(5G) \approx 3$ |
| 10 TB | 1000 | 10 GB | 1 GB | $\log_{1G}(10G) \approx 3$ |

---

## 5. Data Locality Scheduling (Probability)

### The Problem

How likely is it that a map task can be scheduled on a node that holds its input block?

### The Formula

For $r$ replicas across $N$ nodes, with $S$ scheduling slots:

$$P(\text{node-local}) = 1 - \left(\frac{N - r}{N}\right)^S$$

For rack-local (with $k$ nodes per rack, replicas on $R_r$ racks):

$$P(\text{rack-local}) = 1 - \left(\frac{N - k \cdot R_r}{N}\right)^S$$

### Delay Scheduling

Wait $d$ heartbeats before relaxing locality:

$$P(\text{local after } d \text{ waits}) = 1 - \left(1 - \frac{r}{N}\right)^d$$

### Worked Examples

| Nodes | Replicas | Wait Rounds | P(node-local) |
|:---:|:---:|:---:|:---:|
| 100 | 3 | 1 | 3.0% |
| 100 | 3 | 5 | 14.1% |
| 100 | 3 | 20 | 45.6% |
| 100 | 3 | 50 | 78.2% |

---

## 6. HDFS Read/Write Pipeline (Latency)

### The Problem

What determines HDFS write latency for a single block with replication?

### The Formula

Write pipeline (pipelined replication):

$$T_{write} = T_{setup} + \frac{S_{block}}{BW_{min}} + T_{ack}$$

Where:
- $T_{setup}$ = pipeline setup (NameNode RPC + DataNode handshakes) $\approx 10\text{-}50$ ms
- $BW_{min}$ = min bandwidth in the pipeline (bottleneck link)
- $T_{ack}$ = acknowledgment propagation $\approx r \times RTT_{inter-node}$

Total is not $r \times$ transfer time because replication is pipelined:

$$T_{pipeline} \approx T_{single} + (r - 1) \times RTT$$

### Worked Examples

| Block Size | Bandwidth | Setup | Replicas | Write Time |
|:---:|:---:|:---:|:---:|:---:|
| 128 MB | 1 Gbps | 20 ms | 3 | ~1.07 s |
| 128 MB | 10 Gbps | 20 ms | 3 | ~0.12 s |
| 256 MB | 1 Gbps | 20 ms | 3 | ~2.09 s |

---

## 7. Summary of Formulas

| Formula | Type | Domain |
|---------|------|--------|
| $P(\text{loss}) = p^r$ | Reliability | Replication theory |
| $D_{raw} = D_{logical} \times r \times (1 + O)$ | Capacity | Storage planning |
| $M_{NN} = N_{files} \times 150 + N_{blocks} \times 150$ | Memory | NameNode sizing |
| $S_{shuffle} = M \times \bar{O}_{map}$ | I/O volume | MapReduce shuffle |
| $P(\text{local}) = 1 - ((N-r)/N)^d$ | Probability | Delay scheduling |
| $T_{write} \approx S/BW + (r-1) \times RTT$ | Latency | Write pipeline |

## Prerequisites

- probability, combinatorics, information-theory, linux-filesystems, networking

## Complexity

| Operation | Time | Space |
|-----------|------|-------|
| NameNode file lookup | O(1) hash | O(N) files in memory |
| Block report processing | O(B) blocks per DN | O(B * N) total |
| Balancer convergence | O(N * B) moves | O(1) per move |
| MapReduce sort (per reducer) | O(S log S) | O(S) spill to disk |
| HDFS write (r replicas) | O(S / BW) pipelined | O(r * S) storage |
