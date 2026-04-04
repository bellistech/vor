# The Mathematics of Libvirt — Resource Scheduling, Network Topology & Storage Allocation

> *Libvirt orchestrates virtual resources across hypervisors, and the underlying allocation problems map directly to bin-packing for VM placement, graph coloring for network isolation, and queueing models for storage pool throughput. These models explain why certain configurations perform poorly and how to optimize resource utilization.*

---

## 1. VM Placement as Bin Packing (Combinatorial Optimization)

### The Problem

Given $N$ VMs with resource vectors (CPU, memory, I/O) and $M$ hosts with capacity vectors, find a placement that minimizes hosts used while respecting capacity constraints. This is the vector bin-packing problem.

### The Formula

Minimize the number of active hosts:

$$\min \sum_{j=1}^{M} y_j$$

Subject to:

$$\sum_{i=1}^{N} r_{i,d} \cdot x_{ij} \leq C_{j,d} \cdot y_j \quad \forall j, d$$

$$\sum_{j=1}^{M} x_{ij} = 1 \quad \forall i$$

Where:
- $x_{ij} \in \{0,1\}$ = VM $i$ assigned to host $j$
- $y_j \in \{0,1\}$ = host $j$ is active
- $r_{i,d}$ = VM $i$'s requirement in dimension $d$
- $C_{j,d}$ = host $j$'s capacity in dimension $d$

### Worked Examples

**3 VMs (2CPU/4GB, 4CPU/8GB, 1CPU/2GB) on 2 hosts (8CPU/16GB each):**

Host 1: $2+4 = 6 \leq 8$ CPU, $4+8 = 12 \leq 16$ GB. Fits.
Host 2: $1 \leq 8$ CPU, $2 \leq 16$ GB. Fits.

But First-Fit-Decreasing heuristic packs all three on Host 1:
$2+4+1 = 7 \leq 8$ CPU, $4+8+2 = 14 \leq 16$ GB. All fit on one host.

Optimal: 1 host. FFD achieves $\leq \frac{11}{9} \cdot OPT + 6/9$ hosts.

---

## 2. Virtual Network Isolation (Graph Coloring)

### The Problem

Multiple virtual networks must share physical infrastructure without interference. Each network is a subgraph; overlapping networks on the same bridge need VLAN tags. The minimum number of VLANs needed is the chromatic number of the conflict graph.

### The Formula

The conflict graph $G = (V, E)$ where vertices are virtual networks and edges connect networks sharing physical bridges:

$$\chi(G) \leq \Delta(G) + 1$$

Where $\chi(G)$ is the chromatic number and $\Delta(G)$ is the maximum degree (Brooks' theorem, for connected graphs that aren't complete or odd cycles).

VLAN assignment is a proper coloring $c: V \rightarrow \{1, ..., k\}$ such that:

$$c(u) \neq c(v) \quad \forall (u,v) \in E$$

### Worked Examples

**4 virtual networks, each pair shares at least one bridge (complete graph $K_4$):**

$$\chi(K_4) = 4 \text{ VLANs needed}$$

**4 networks in a ring (each shares bridge with 2 neighbors):**

$$\chi(C_4) = 2 \text{ VLANs sufficient (even cycle)}$$

**5 networks in a ring:**

$$\chi(C_5) = 3 \text{ VLANs needed (odd cycle)}$$

---

## 3. NAT Port Allocation (Counting & Probability)

### The Problem

Libvirt's default NAT network maps guest connections to host ports. Given a port range $[a, b]$ shared among $N$ VMs, what is the collision probability?

### The Formula

This is a birthday problem variant. For $k$ concurrent connections in a range of $n = b - a + 1$ ports:

$$P(\text{collision}) = 1 - \prod_{i=0}^{k-1} \frac{n-i}{n} \approx 1 - e^{-k(k-1)/(2n)}$$

### Worked Examples

**Default range 1024-65535 ($n = 64512$), 100 concurrent connections:**

$$P(\text{collision}) \approx 1 - e^{-100 \times 99 / (2 \times 64512)} = 1 - e^{-0.0767} = 0.0738$$

**500 concurrent connections:**

$$P(\text{collision}) \approx 1 - e^{-500 \times 499 / 129024} = 1 - e^{-1.934} = 0.855$$

This is why port forwarding rules (static mapping) are preferred for servers.

---

## 4. Storage Pool IOPS Distribution (Queueing Theory)

### The Problem

A storage pool serves $N$ VMs. Each VM generates I/O requests at rate $\lambda_i$. The pool has a maximum throughput of $\mu$ IOPS. What is the expected queue delay?

### The Formula

Using M/M/1 queueing model with aggregate arrival rate:

$$\Lambda = \sum_{i=1}^{N} \lambda_i$$

$$\rho = \frac{\Lambda}{\mu} \quad \text{(utilization)}$$

$$W_q = \frac{\rho}{\mu(1 - \rho)} \quad \text{(mean queue wait time)}$$

$$L_q = \frac{\rho^2}{1 - \rho} \quad \text{(mean queue length)}$$

### Worked Examples

**5 VMs at 2000 IOPS each, pool capacity 15000 IOPS:**

$$\Lambda = 10000, \quad \rho = 10000/15000 = 0.667$$

$$W_q = \frac{0.667}{15000 \times 0.333} = 0.133ms$$

**Add 3 more VMs (8 total at 2000 IOPS):**

$$\Lambda = 16000 > \mu = 15000 \quad \Rightarrow \rho > 1 \text{ (queue grows unbounded!)}$$

This shows why overprovisioning storage IOPS is critical. At $\rho > 0.8$, latency increases rapidly:

$$W_q(\rho=0.8) = \frac{0.8}{15000 \times 0.2} = 0.267ms$$
$$W_q(\rho=0.95) = \frac{0.95}{15000 \times 0.05} = 1.267ms$$

---

## 5. Live Migration Bandwidth Allocation (Optimization)

### The Problem

Multiple VMs migrating simultaneously share network bandwidth. How should bandwidth be allocated to minimize total migration time?

### The Formula

For $N$ VMs with memory sizes $M_i$ and dirty rates $d_i$, sharing bandwidth $B$:

$$\min \sum_{i=1}^{N} T_i \quad \text{subject to} \sum_{i=1}^{N} b_i \leq B$$

Each VM's migration time:

$$T_i = \frac{M_i}{b_i - d_i} \quad \text{(requires } b_i > d_i \text{)}$$

Using Lagrange multipliers, the optimal allocation:

$$b_i^* = d_i + \sqrt{\frac{M_i}{\lambda}}$$

Where $\lambda$ satisfies:

$$\sum_{i=1}^{N} \left(d_i + \sqrt{\frac{M_i}{\lambda}}\right) = B$$

### Worked Examples

**2 VMs: (8GB, 50MB/s dirty) and (4GB, 100MB/s dirty), B = 1GB/s:**

$$d_1 + \sqrt{8/\lambda} + d_2 + \sqrt{4/\lambda} = 1000$$

$$150 + \sqrt{8/\lambda} + \sqrt{4/\lambda} = 1000$$

$$\sqrt{8/\lambda} + \sqrt{4/\lambda} = 850$$

Solving: $\sqrt{1/\lambda} \approx 260.5$

$$b_1^* = 50 + 260.5\sqrt{8} = 50 + 737 = 787 \text{ MB/s}$$

$$b_2^* = 100 + 260.5\sqrt{4} = 100 + 521 = 621 \text{ MB/s}$$

Total exceeds $B$, so in practice sequential migration is needed.

---

## 6. Snapshot Chain Read Amplification (DAG Analysis)

### The Problem

Each snapshot adds a layer. A read to an unmodified sector must traverse the chain. For a chain of depth $d$, what is the expected read amplification?

### The Formula

$$A_{read} = \sum_{k=1}^{d} k \cdot P(\text{first write at layer } k)$$

If each layer modifies fraction $w$ of sectors independently:

$$P(\text{first write at layer } k) = w \cdot (1-w)^{k-1}$$

$$A_{read} = \sum_{k=1}^{d} k \cdot w(1-w)^{k-1} = \frac{1 - (1-w)^d(1 + dw)}{w}$$

### Worked Examples

**5-layer chain, 20% write density per layer:**

$$A_{read} = \frac{1 - (0.8)^5(1 + 5 \times 0.2)}{0.2} = \frac{1 - 0.3277 \times 2}{0.2} = \frac{0.3446}{0.2} = 1.72$$

On average each read checks 1.72 layers. At depth 10:

$$A_{read} = \frac{1 - (0.8)^{10}(1 + 2)}{0.2} = \frac{1 - 0.1074 \times 3}{0.2} = \frac{0.678}{0.2} = 3.39$$

This justifies periodic snapshot flattening to maintain read performance.

---

## Prerequisites

- bin-packing, combinatorial-optimization
- graph-coloring, chromatic-number
- queueing-theory, M/M/1-model
- birthday-problem, probability
- lagrange-multipliers, constrained-optimization
