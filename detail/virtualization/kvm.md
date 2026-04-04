# The Mathematics of KVM — VM Exit Costs, Memory Virtualization & NUMA Topology

> *KVM performance hinges on minimizing the frequency and cost of VM exits -- transitions from guest to host execution. The mathematics of hardware-assisted virtualization reveals how EPT/NPT eliminates page walk overhead, why NUMA-aware placement matters exponentially with core count, and how halt-polling trades CPU cycles for latency reduction.*

---

## 1. VM Exit Cost Model (Systems Theory)

### The Problem

Every privileged guest operation causes a VM exit: the CPU saves guest state, loads host state, processes the exit, then re-enters the guest. What is the overhead per exit and how does exit frequency impact throughput?

### The Formula

$$T_{exit} = T_{save} + T_{handle} + T_{restore}$$

Total overhead fraction:

$$\Omega = \frac{f_{exit} \cdot T_{exit}}{f_{exit} \cdot T_{exit} + IPC_{guest} / f_{clock}}$$

Where:
- $T_{save}$ = guest state save time (~0.5 us)
- $T_{handle}$ = host handler time (varies by exit reason)
- $T_{restore}$ = guest state restore time (~0.5 us)
- $f_{exit}$ = exit frequency (exits/sec)
- $IPC_{guest}$ = guest instructions per clock

### Worked Examples

**I/O-bound VM (80K exits/sec, 2us per exit):**

$$\Omega = \frac{80000 \times 2 \times 10^{-6}}{80000 \times 2 \times 10^{-6} + 1} = \frac{0.16}{1.16} = 13.8\%$$

**Compute-bound VM (200 exits/sec, 2us per exit):**

$$\Omega = \frac{200 \times 2 \times 10^{-6}}{200 \times 2 \times 10^{-6} + 1} = \frac{0.0004}{1.0004} = 0.04\%$$

VirtIO reduces I/O exit frequency by 10-50x through batching.

---

## 2. Extended Page Tables (Memory Architecture)

### The Problem

Without EPT/NPT, every guest page table walk requires a VM exit for the hypervisor to translate guest-physical to host-physical addresses. EPT adds a hardware second-level translation. What is the cost?

### The Formula

A guest page walk visits $L_g$ levels. With EPT, each guest level requires a full host walk of $L_h$ levels:

$$W_{ept} = L_g \times L_h + L_h$$

For 4-level paging ($L_g = L_h = 4$):

$$W_{ept} = 4 \times 4 + 4 = 20 \text{ memory accesses}$$

Without EPT (shadow page tables):

$$W_{shadow} = L_g + C_{exit} \cdot f_{fault}$$

### Worked Examples

**TLB miss with EPT (4-level paging):**

$$W_{ept} = 20 \text{ memory accesses} \times 100ns = 2\mu s$$

**TLB miss with shadow paging:**

$$W_{shadow} = 4 \times 100ns + 2\mu s \times 1 = 2.4\mu s \text{ (if fault triggers exit)}$$

EPT is faster for high page-fault rates because it avoids VM exits entirely:

$$Crossover: f_{fault} > \frac{(W_{ept} - L_g) \times T_{mem}}{C_{exit}}$$

$$f_{fault} > \frac{16 \times 100ns}{2\mu s} = 0.8 \text{ faults per walk}$$

EPT wins for virtually all workloads. 5-level paging increases $W_{ept}$ to $5 \times 5 + 5 = 30$.

---

## 3. NUMA Topology Optimization (Graph Theory)

### The Problem

On multi-socket systems, memory access latency depends on whether the access is local or remote. VMs spanning NUMA nodes pay a latency penalty on every remote access.

### The Formula

$$T_{avg} = f_{local} \cdot T_{local} + (1 - f_{local}) \cdot T_{remote}$$

$$Slowdown = \frac{T_{avg}}{T_{local}} = f_{local} + (1 - f_{local}) \cdot R_{NUMA}$$

Where $R_{NUMA} = T_{remote} / T_{local}$ is typically 1.3-2.0x.

For a VM with $V$ vCPUs split across $N$ nodes:

$$f_{local} \approx \frac{\lceil V/N \rceil}{V} \text{ (worst case, uniform access)}$$

### Worked Examples

**8 vCPU VM on 2 NUMA nodes, R_NUMA = 1.5:**

$$f_{local} = 0.5, \quad Slowdown = 0.5 + 0.5 \times 1.5 = 1.25 \text{ (25% slower)}$$

**8 vCPU VM pinned to 1 NUMA node:**

$$f_{local} = 1.0, \quad Slowdown = 1.0 \text{ (optimal)}$$

**16 vCPU VM on 4 nodes, R_NUMA = 1.7:**

$$f_{local} = 0.25, \quad Slowdown = 0.25 + 0.75 \times 1.7 = 1.525 \text{ (52.5% slower)}$$

This is why `numactl --membind` and vCPU pinning to a single node are critical.

---

## 4. Hugepage TLB Coverage (Memory Hierarchy)

### The Problem

The TLB has a fixed number of entries. With 4KB pages, TLB coverage is limited. Hugepages (2MB or 1GB) dramatically increase coverage.

### The Formula

$$Coverage = N_{TLB} \times P_{size}$$

TLB miss rate approximation (working set model):

$$R_{miss} = \max\left(0, 1 - \frac{Coverage}{WSS}\right) \cdot A_{random}$$

Where $WSS$ = working set size, $A_{random}$ = fraction of non-sequential accesses.

### Worked Examples

**64-entry dTLB, 4KB pages, 512MB working set, 30% random access:**

$$Coverage = 64 \times 4KB = 256KB$$

$$R_{miss} = (1 - 256KB/512MB) \times 0.3 \approx 0.3 \text{ (30% miss rate)}$$

**64-entry dTLB, 2MB hugepages:**

$$Coverage = 64 \times 2MB = 128MB$$

$$R_{miss} = (1 - 128/512) \times 0.3 = 0.75 \times 0.3 = 0.225 \text{ (22.5%)}$$

**32-entry 1GB page TLB:**

$$Coverage = 32 \times 1GB = 32GB$$

$$R_{miss} \approx 0 \text{ (working set fully covered)}$$

Combined with EPT, hugepages reduce the 20-access page walk to near-zero frequency.

---

## 5. Halt-Polling Tradeoff (Optimization Theory)

### The Problem

When a vCPU executes HLT, KVM can either immediately schedule the vCPU out (saving CPU) or spin-poll for new work (reducing wakeup latency). The optimal poll duration depends on the inter-arrival time distribution.

### The Formula

$$Cost_{poll} = P_{idle} \cdot T_{poll} \cdot C_{cpu}$$

$$Benefit_{poll} = P_{wake} \cdot (T_{context\_switch} - T_{poll}/2)$$

Optimal polling time:

$$T_{poll}^* = \min\left(T_{max}, \frac{2 \cdot P_{wake} \cdot T_{cs}}{P_{idle} \cdot C_{cpu} + P_{wake}}\right)$$

Where $P_{wake}$ is probability of wake within poll window.

### Worked Examples

**Database VM (median inter-arrival = 50us, T_cs = 10us):**

$$P_{wake} \approx 1 - e^{-T_{poll}/50\mu s}$$

At $T_{poll} = 200\mu s$: $P_{wake} = 1 - e^{-4} = 0.982$

Latency saving: $0.982 \times 10\mu s = 9.82\mu s$ saved per HLT, costing $200\mu s \times 0.018 = 3.6\mu s$ wasted.

The default `halt_poll_ns = 200000` (200us) is optimal for moderate I/O workloads.

---

## 6. Live Migration Convergence (Iterative Methods)

### The Problem

Pre-copy migration converges when the dirty rate drops below the transfer rate. Each round transfers fewer pages if the ratio $r < 1$.

### The Formula

$$D_n = D_0 \cdot r^n, \quad r = \frac{R_{dirty}}{R_{transfer}}$$

Total time:

$$T = \frac{D_0}{R_{transfer}} \cdot \frac{1 - r^{n+1}}{1 - r}$$

Convergence criterion: $r < 1$. Downtime in final round:

$$T_{downtime} = \frac{D_n}{R_{transfer}} = \frac{D_0 \cdot r^n}{R_{transfer}}$$

### Worked Examples

**8GB VM, 10Gbps link, 100MB/s dirty rate:**

$$R_{transfer} = 10Gbps / 8 = 1.25GB/s$$

$$r = 0.1/1.25 = 0.08$$

$$T = \frac{8}{1.25} \cdot \frac{1 - 0.08^3}{1 - 0.08} = 6.4 \times 1.087 = 6.96 \text{ sec}$$

$$T_{downtime} = \frac{8 \times 0.08^2}{1.25} = \frac{0.051}{1.25} = 41ms$$

---

## Prerequisites

- virtual-memory, page-tables, TLB
- NUMA-architecture, memory-hierarchy
- queueing-theory, exponential-distribution
- iterative-convergence, geometric-series
