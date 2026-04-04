# The Mathematics of QEMU — Emulation Theory, Translation Overhead & I/O Modeling

> *QEMU translates guest instructions to host instructions through dynamic binary translation, a process whose overhead can be modeled as a function of basic block size, translation cache hit rates, and the ratio of privileged to unprivileged instructions. Understanding this math explains when emulation is viable and when hardware acceleration becomes essential.*

---

## 1. Dynamic Binary Translation (Compiler Theory)

### The Problem

QEMU's Tiny Code Generator (TCG) translates guest ISA instructions into host ISA instructions. Each guest basic block is translated once and cached. The key performance question: what is the amortized cost per guest instruction?

### The Formula

$$C_{amortized} = \frac{C_{translate}}{N_{exec} \cdot B_{size}} + C_{host} \cdot R_{expansion}$$

Where:
- $C_{translate}$ = cost to translate one basic block (microseconds)
- $N_{exec}$ = number of times the translated block executes
- $B_{size}$ = number of guest instructions per basic block
- $C_{host}$ = cost of one host instruction (nanoseconds)
- $R_{expansion}$ = instruction expansion ratio (guest-to-host)

### Worked Examples

**Hot loop (1000 iterations, 20-instruction block):**

$$C_{amortized} = \frac{50\mu s}{1000 \times 20} + 0.3ns \times 1.4 = 2.5ns + 0.42ns = 2.92ns$$

**Cold path (executed once, 5-instruction block):**

$$C_{amortized} = \frac{50\mu s}{1 \times 5} + 0.3ns \times 1.4 = 10\mu s + 0.42ns \approx 10\mu s$$

The translation cache hit rate determines overall performance:

$$T_{effective} = h \cdot C_{cached} + (1 - h) \cdot C_{translate}$$

For typical workloads, $h > 0.95$ after warmup, making TCG overhead roughly $1.5\text{-}3\times$ native.

---

## 2. KVM Hardware Acceleration (Trap-and-Emulate Model)

### The Problem

With KVM, unprivileged guest instructions run natively. Only privileged operations (I/O, page table updates, MSR access) cause VM exits. Performance depends on the exit frequency.

### The Formula

$$Overhead_{KVM} = f_{exit} \cdot C_{exit}$$

Where:
- $f_{exit}$ = VM exit frequency (exits per second)
- $C_{exit}$ = cost per VM exit (typically 1-5 microseconds)

The effective CPU utilization:

$$\eta_{CPU} = \frac{T_{guest}}{T_{guest} + T_{exit}} = \frac{1}{1 + f_{exit} \cdot C_{exit} / IPS_{guest}}$$

### Worked Examples

**Compute-bound workload (100 exits/sec):**

$$\eta_{CPU} = \frac{1}{1 + 100 \times 2\mu s / 1} = \frac{1}{1 + 0.0002} \approx 99.98\%$$

**I/O-heavy workload (50,000 exits/sec):**

$$\eta_{CPU} = \frac{1}{1 + 50000 \times 2\mu s / 1} = \frac{1}{1.1} \approx 90.9\%$$

This is why VirtIO exists: it batches I/O to reduce exit frequency from 50,000+ to under 5,000 exits/sec.

---

## 3. Copy-on-Write Disk Algebra (Graph Theory)

### The Problem

qcow2 backing chains form a directed acyclic graph. Each read must walk the chain until it finds a cluster that has been written. What is the expected read latency?

### The Formula

For a chain of depth $d$ with write density $w$ at each layer:

$$E[lookups] = \sum_{i=0}^{d-1} (1-w)^i \cdot 1 = \frac{1 - (1-w)^d}{w}$$

$$Latency_{read} = E[lookups] \cdot L_{seek} + L_{read}$$

Where $L_{seek}$ is the cost to check one layer's L2 table (refcount lookup).

### Worked Examples

**2-layer chain, overlay 30% written:**

$$E[lookups] = \frac{1 - (0.7)^2}{0.3} = \frac{1 - 0.49}{0.3} = \frac{0.51}{0.3} = 1.7$$

**5-layer chain, each layer 10% written:**

$$E[lookups] = \frac{1 - (0.9)^5}{0.1} = \frac{1 - 0.59}{0.1} = 4.1$$

This shows why deep backing chains degrade read performance. Committing (flattening) reduces $d$ to 1.

---

## 4. VirtIO Ring Buffer Throughput (Queueing Theory)

### The Problem

VirtIO uses split or packed virtqueues (ring buffers) to batch I/O requests, reducing VM exits. The throughput depends on queue depth and batch processing.

### The Formula

$$Throughput = \frac{Q_{depth}}{L_{round\_trip}} \cdot S_{request}$$

Where the effective batch size under load:

$$B_{eff} = \min(Q_{depth}, \lambda \cdot L_{notify})$$

- $\lambda$ = request arrival rate
- $L_{notify}$ = notification latency (time between kicks)

The exit reduction ratio compared to per-request traps:

$$R_{reduction} = \frac{\lambda}{ceil(\lambda / B_{eff})} = \min(B_{eff}, \lambda \cdot L_{notify})$$

### Worked Examples

**Queue depth 256, 100K IOPS, 10us notification latency:**

$$B_{eff} = \min(256, 100000 \times 10\mu s) = \min(256, 1) = 1 \text{ (low load, no batching)}$$

**Queue depth 256, 500K IOPS, 50us notification latency:**

$$B_{eff} = \min(256, 500000 \times 50\mu s) = \min(256, 25) = 25 \text{ (effective batching)}$$

VM exits drop from 500K/sec to 20K/sec, a 25x reduction.

---

## 5. Memory Overcommit and Ballooning (Resource Allocation)

### The Problem

The balloon driver reclaims guest memory for the host. Given $N$ VMs, how much physical memory is needed with ballooning?

### The Formula

$$M_{physical} = \sum_{i=1}^{N} (M_{allocated,i} - M_{balloon,i}) + M_{host}$$

The overcommit ratio:

$$R_{overcommit} = \frac{\sum M_{configured,i}}{M_{physical} - M_{host}}$$

Safe overcommit depends on actual utilization variance:

$$P(OOM) = P\left(\sum_{i=1}^{N} U_i > M_{physical} - M_{host}\right)$$

For independent workloads with mean $\mu_i$ and variance $\sigma_i^2$:

$$P(OOM) \approx \Phi\left(\frac{M_{physical} - M_{host} - \sum \mu_i}{\sqrt{\sum \sigma_i^2}}\right)$$

### Worked Examples

**8 VMs, each configured 4GB, mean usage 1.5GB, stddev 0.5GB, 16GB host:**

$$R_{overcommit} = \frac{32}{16 - 2} = 2.29\times$$

$$P(OOM) = \Phi\left(\frac{14 - 12}{\sqrt{8 \times 0.25}}\right) = \Phi\left(\frac{2}{1.41}\right) = \Phi(1.41) \approx 0.079$$

Roughly 8% OOM probability -- ballooning with a target of 1GB per VM reduces this significantly.

---

## 6. Live Migration Transfer Time (Network Theory)

### The Problem

Pre-copy migration iteratively transfers dirty pages. Convergence depends on the dirty rate versus transfer rate.

### The Formula

Convergence condition:

$$R_{dirty} < R_{transfer}$$

$$R_{transfer} = \frac{BW_{network}}{P_{size}} \text{ (pages/sec)}$$

Total migration time for $n$ rounds:

$$T_{migration} = \sum_{k=0}^{n} \frac{D_k}{R_{transfer}}$$

Where $D_k$ is dirty pages in round $k$:

$$D_k = D_0 \cdot \left(\frac{R_{dirty}}{R_{transfer}}\right)^k$$

### Worked Examples

**4GB VM, 1Gbps link, 4KB pages, 5000 pages/sec dirty rate:**

$$R_{transfer} = \frac{1 \times 10^9 / 8}{4096} \approx 30,500 \text{ pages/sec}$$

$$D_0 = \frac{4 \times 10^9}{4096} = 976,562 \text{ pages}$$

$$r = \frac{5000}{30500} = 0.164$$

$$T_{total} = \frac{976562}{30500} \cdot \frac{1}{1 - 0.164} = 32.0 \cdot 1.196 = 38.3 \text{ sec}$$

---

## Prerequisites

- binary-translation, instruction-set-architecture, virtual-memory
- queueing-theory, ring-buffers, DMA
- copy-on-write, directed-acyclic-graphs
- probability-distributions, normal-distribution
