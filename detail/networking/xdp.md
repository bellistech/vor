# The Mathematics of XDP -- Packet Processing at Wire Speed

> *When packets arrive faster than the kernel can allocate memory, the only solution is to never allocate at all.*

---

## 1. Throughput Bounds (Packet Processing Theory)

### The Problem

Given a network interface receiving packets at line rate, what is the maximum
packet processing rate achievable, and how does XDP's position in the stack
determine the theoretical throughput ceiling?

### The Formula

For a link of bandwidth $B$ with minimum packet size $S_{min}$ (including
inter-frame gap and preamble), the maximum packet rate is:

$$R_{max} = \frac{B}{(S_{min} + S_{overhead}) \times 8}$$

For 10 GbE with 64-byte Ethernet frames:

$$S_{overhead} = 20 \text{ bytes (preamble + IFG)}$$

$$R_{max} = \frac{10 \times 10^9}{(64 + 20) \times 8} = 14.88 \text{ Mpps}$$

For 100 GbE:

$$R_{max} = \frac{100 \times 10^9}{(64 + 20) \times 8} = 148.8 \text{ Mpps}$$

### Worked Examples

**Example 1:** A 25 GbE NIC processes 64-byte frames. What fraction of line rate
does XDP native mode achieve at 24 Mpps?

$$R_{line} = \frac{25 \times 10^9}{84 \times 8} = 37.2 \text{ Mpps}$$

$$\text{Efficiency} = \frac{24}{37.2} = 64.5\%$$

**Example 2:** Compare XDP_DROP (24 Mpps) vs iptables DROP (3 Mpps) for DDoS
mitigation. How many servers does iptables require to match one XDP server?

$$N_{iptables} = \left\lceil \frac{24}{3} \right\rceil = 8 \text{ servers}$$

## 2. Latency Analysis (Processing Pipeline Depth)

### The Problem

Each layer in the Linux networking stack adds processing latency. Where in the
pipeline does XDP execute, and how does pipeline depth affect per-packet latency?

### The Formula

Total per-packet latency through the stack:

$$L_{total} = \sum_{i=1}^{n} L_i = L_{driver} + L_{alloc} + L_{netfilter} + L_{routing} + L_{socket}$$

XDP operates at stage 1 (driver), bypassing stages 2-5:

$$L_{xdp} = L_{driver} \approx 3\text{--}5 \text{ } \mu s$$

$$L_{stack} = L_{driver} + L_{alloc} + L_{netfilter} + L_{routing} + L_{socket} \approx 15\text{--}30 \text{ } \mu s$$

The latency reduction factor:

$$\alpha = \frac{L_{stack}}{L_{xdp}} = \frac{20}{4} = 5\times$$

### Worked Examples

**Example 1:** A packet filter inspects headers (200 ns) and performs a map lookup
(100 ns). What is the total XDP processing latency including driver overhead?

$$L_{xdp} = L_{driver\_base} + L_{inspect} + L_{lookup} = 3000 + 200 + 100 = 3300 \text{ ns} = 3.3 \text{ } \mu s$$

**Example 2:** At 14.88 Mpps line rate, what is the per-packet time budget?

$$T_{budget} = \frac{1}{14.88 \times 10^6} = 67.2 \text{ ns}$$

This means each BPF instruction must complete in nanoseconds to sustain line rate.

## 3. Hash-Based Load Distribution (RSS and AF_XDP)

### The Problem

Multi-queue NICs distribute packets across CPU cores using Receive Side Scaling
(RSS). How does the hash function distribute flows, and what is the probability
of queue imbalance?

### The Formula

The Toeplitz hash maps a flow tuple to a queue index:

$$q = H_{toeplitz}(src\_ip, dst\_ip, src\_port, dst\_port) \mod N_{queues}$$

By the birthday problem, the probability that at least two of $k$ flows
collide on the same queue (out of $N$ queues):

$$P_{collision} \approx 1 - e^{-\frac{k(k-1)}{2N}}$$

For load imbalance, the expected maximum queue occupancy when distributing $k$
flows uniformly across $N$ queues follows:

$$E[\max] \approx \frac{k}{N} + \sqrt{\frac{2k \ln N}{N}}$$

### Worked Examples

**Example 1:** 1000 concurrent flows distributed across 8 RSS queues.
Expected max queue occupancy:

$$E[\max] \approx \frac{1000}{8} + \sqrt{\frac{2 \times 1000 \times \ln 8}{8}} = 125 + \sqrt{\frac{2000 \times 2.08}{8}} = 125 + 22.8 = 147.8 \text{ flows}$$

The most loaded queue handles ~18% more than the average (125).

**Example 2:** With 4 AF_XDP sockets bound to 4 queues, what is the collision
probability for 20 flows?

$$P_{collision} \approx 1 - e^{-\frac{20 \times 19}{2 \times 4}} = 1 - e^{-47.5} \approx 1.0$$

Collision is nearly certain; this is expected. The concern is extreme imbalance,
not collisions per se.

## 4. BPF Map Lookup Complexity (Data Structure Performance)

### The Problem

XDP programs use BPF maps for stateful processing. What are the time
complexities of different map types, and how does map size affect throughput?

### The Formula

| Map Type | Lookup | Insert | Space |
|----------|--------|--------|-------|
| `ARRAY` | $O(1)$ | $O(1)$ | $O(n)$ |
| `HASH` | $O(1)$ avg | $O(1)$ avg | $O(n)$ |
| `LRU_HASH` | $O(1)$ avg | $O(1)$ amort | $O(n)$ |
| `LPM_TRIE` | $O(W)$ | $O(W)$ | $O(nW)$ |
| `PERCPU_ARRAY` | $O(1)$ | $O(1)$ | $O(nC)$ |

Where $W$ is the key width in bits, $n$ is the number of entries, and $C$ is the
CPU count.

For hash maps, the expected lookup cost with load factor $\lambda = n/m$:

$$E[probes] = \frac{1}{1 - \lambda} \quad \text{(open addressing)}$$

### Worked Examples

**Example 1:** A hash map with 10,000 entries and 16,384 buckets ($\lambda = 0.61$).
Expected probes per lookup:

$$E[probes] = \frac{1}{1 - 0.61} = 2.56 \text{ probes}$$

At ~50 ns per probe: $2.56 \times 50 = 128$ ns per lookup.

**Example 2:** An LPM trie for IPv4 CIDR matching (W = 32 bits).
Maximum traversal depth:

$$D_{max} = W = 32 \text{ nodes}$$

At ~30 ns per node: worst case $32 \times 30 = 960$ ns per lookup.

## 5. DDoS Mitigation Capacity (Filtering Economics)

### The Problem

How many attack packets can an XDP-based DDoS filter absorb, and what is the
cost comparison with traditional approaches?

### The Formula

The mitigation capacity of a single server:

$$C_{server} = R_{xdp} \times N_{cores} \times \eta$$

where $R_{xdp}$ is per-core XDP drop rate, $N_{cores}$ is core count, and
$\eta$ is the efficiency factor (typically 0.7-0.9 accounting for cache
misses and map lookups).

Cost per Mpps:

$$\text{Cost}_{Mpps} = \frac{\text{Server cost}}{C_{server}}$$

### Worked Examples

**Example 1:** A 16-core server drops packets at 24 Mpps per core with
$\eta = 0.8$. Total capacity:

$$C_{server} = 24 \times 16 \times 0.8 = 307.2 \text{ Mpps}$$

At a 100 Gbps volumetric attack with 64-byte packets (148.8 Mpps):

$$N_{servers} = \left\lceil \frac{148.8}{307.2} \right\rceil = 1 \text{ server}$$

**Example 2:** Compare yearly cost. Server: $5,000/year. Cloud DDoS service:
$3,000/month at 100 Gbps.

$$\text{XDP cost} = \$5{,}000/\text{year}$$
$$\text{Cloud cost} = \$36{,}000/\text{year}$$
$$\text{Savings} = 1 - \frac{5{,}000}{36{,}000} = 86\%$$

## 6. Instruction Budget (BPF Verifier Constraints)

### The Problem

The BPF verifier enforces an instruction limit per program. How does this
constrain the complexity of XDP programs, and how do tail calls extend it?

### The Formula

Maximum instructions per program (kernel 5.2+):

$$I_{max} = 1{,}000{,}000 \text{ (verified instructions)}$$

With tail calls (max depth $D = 33$):

$$I_{total} = I_{max} \times D = 1{,}000{,}000 \times 33 = 33{,}000{,}000$$

Time budget per instruction at line rate (14.88 Mpps on 10 GbE):

$$T_{inst} = \frac{T_{budget}}{I_{typical}} = \frac{67.2 \text{ ns}}{500} = 0.13 \text{ ns}$$

This is approximately 1 CPU cycle at 8 GHz equivalent, meaning XDP programs at
line rate must use very few instructions per packet.

### Worked Examples

**Example 1:** An XDP program uses 2,000 instructions per packet. Maximum
sustainable packet rate on a single 3.5 GHz core (1 instruction per cycle):

$$R_{max} = \frac{3.5 \times 10^9}{2{,}000} = 1.75 \text{ Mpps}$$

**Example 2:** A firewall rule set requires 50 rules, each using 40
instructions. Total instruction count and verifier utilization:

$$I_{total} = 50 \times 40 = 2{,}000 \text{ instructions}$$
$$\text{Utilization} = \frac{2{,}000}{1{,}000{,}000} = 0.2\%$$

## Prerequisites

- Linux networking stack fundamentals (L2/L3/L4 headers, sk_buff lifecycle)
- BPF/eBPF programming model (maps, helpers, verifier constraints)
- NIC architecture (DMA rings, RSS, multi-queue)
- C programming for BPF programs (restricted C subset)
- Probability and combinatorics (birthday problem, hash collisions)
- Queueing theory basics (arrival rates, service times)
- Big-O complexity analysis for data structure selection
