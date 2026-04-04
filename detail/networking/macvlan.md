# The Mathematics of macvlan/ipvlan — Address Space Partitioning & Performance Models

> *macvlan and ipvlan solve the same isolation problem with different mathematical constraints: macvlan partitions the MAC address space (bounded by switch CAM table size) while ipvlan partitions only the IP space (bounded by routing table capacity), creating a fundamental trade-off between layer-2 isolation and scalability.*

---

## 1. MAC Address Space Exhaustion (Counting and Capacity)

### The Problem

Each macvlan interface requires a unique MAC address. Upstream switches store MACs in a CAM (Content Addressable Memory) table of fixed size $C$. With $N$ macvlan interfaces across $H$ hosts, when does the switch CAM overflow?

### The Formula

Total MAC entries consumed by macvlan:

$$M_{\text{total}} = H + \sum_{h=1}^{H} N_h$$

Where $H$ is the number of host physical MACs and $N_h$ is macvlan count on host $h$.

CAM overflow condition:

$$M_{\text{total}} + M_{\text{other}} > C$$

Where $M_{\text{other}}$ is MACs from non-macvlan sources. When the CAM overflows, the switch floods frames to all ports — a security and performance disaster.

Time to overflow with MAC learning rate $\lambda$ and aging time $A$:

$$M_{\text{steady}} = \lambda \cdot A$$

### Worked Examples

**Example 1:** 10 servers, each running 50 Docker macvlan containers. Switch CAM size $C = 8192$:

$$M_{\text{total}} = 10 + 10 \times 50 = 510 \text{ MACs}$$

Well within limits (6.2% of CAM). But at 500 containers per host:

$$M_{\text{total}} = 10 + 10 \times 500 = 5010$$

61% of CAM. Adding other network traffic (printers, APs, other servers): potentially overflows.

**Example 2:** ipvlan alternative — same 5010 endpoints but only 10 MACs (one per host):

$$M_{\text{ipvlan}} = 10$$

0.12% of CAM. ipvlan scales to millions of endpoints without CAM pressure.

---

## 2. macvlan Bridge Mode Forwarding (Internal Switching)

### The Problem

In bridge mode, macvlan interfaces on the same parent can communicate without going through the external switch. The kernel maintains an internal hash table mapping MACs to sub-interfaces. What is the lookup complexity?

### The Formula

The kernel uses a hash table with $B$ buckets. With $N$ macvlan interfaces hashed using a uniform hash:

Expected chain length per bucket:

$$E[L] = \frac{N}{B}$$

Lookup time (linear probing within chain):

$$E[T_{\text{lookup}}] = O\left(1 + \frac{N}{B}\right)$$

With the kernel's typical $B = 256$ buckets:

$$E[L] = \frac{N}{256}$$

Probability of a specific bucket having $k$ entries (Poisson approximation):

$$P(k) = \frac{(N/B)^k \cdot e^{-N/B}}{k!}$$

### Worked Examples

**Example:** 100 macvlan interfaces, 256 buckets:

$$E[L] = \frac{100}{256} = 0.39$$

Average lookup: constant time. Worst-case bucket length (99th percentile for Poisson $\lambda = 0.39$):

$$P(k \geq 3) = 1 - \sum_{i=0}^{2} \frac{0.39^i e^{-0.39}}{i!} = 1 - e^{-0.39}(1 + 0.39 + 0.076) = 1 - 0.677 \times 1.466 = 0.007$$

Less than 1% chance any bucket has 3+ entries. Lookup is effectively $O(1)$.

With 10,000 macvlans:

$$E[L] = \frac{10000}{256} = 39$$

Chains averaging 39 entries — significant performance degradation. This is where ipvlan L3's routing table lookup ($O(\log n)$ with longest-prefix match) becomes faster.

---

## 3. VEPA Mode and Hairpin Bandwidth (Network Topology)

### The Problem

VEPA mode sends all traffic to the external switch, even for local macvlan-to-macvlan communication. This "hairpins" traffic, doubling bandwidth usage on the uplink. What is the bandwidth overhead?

### The Formula

For $F$ macvlan-to-macvlan flows on the same host, each at rate $r$:

Bridge mode uplink bandwidth: $0$ (local switching)

VEPA mode uplink bandwidth:

$$B_{\text{VEPA}} = 2 \cdot F \cdot r$$

Factor of 2: each packet traverses the uplink twice (egress + hairpin return).

Total uplink utilization with $E$ external flows at rate $r_e$:

$$U = \frac{2Fr + \sum_{i=1}^{E} r_{e,i}}{B_{\text{link}}}$$

### Worked Examples

**Example:** 20 macvlan containers, 10 local flows at 100 Mbps each, 15 external flows at 50 Mbps, 10 Gbps uplink:

Bridge mode:

$$U_{\text{bridge}} = \frac{0 + 15 \times 50}{10000} = \frac{750}{10000} = 7.5\%$$

VEPA mode:

$$U_{\text{VEPA}} = \frac{2 \times 10 \times 100 + 750}{10000} = \frac{2750}{10000} = 27.5\%$$

VEPA uses 3.7x more uplink bandwidth. But VEPA enables switch-level policy enforcement (ACLs, monitoring, QoS) on all traffic, which is why hypervisors use it.

---

## 4. ipvlan L3 Routing Table Scaling (Algorithmic Complexity)

### The Problem

ipvlan L3 mode routes packets using the kernel routing table. With $N$ ipvlan endpoints, how does routing lookup performance scale?

### The Formula

Linux uses an LC-trie (Level-Compressed trie) for IPv4 route lookup. For $N$ routes with prefix length $p$:

$$T_{\text{lookup}} = O\left(\frac{p}{\log_2 B}\right)$$

Where $B$ is the trie branching factor (typically 16 for 4-bit stride). For IPv4 ($p = 32$):

$$T_{\text{lookup}} = O\left(\frac{32}{4}\right) = O(8)$$

This is constant regardless of $N$ — routing scales better than macvlan bridge's hash table for large $N$.

Memory per route entry:

$$M_{\text{route}} \approx 128 \text{ bytes (fib\_info + fib\_alias)}$$

Total routing table memory:

$$M_{\text{total}} = N \times M_{\text{route}}$$

### Worked Examples

**Example:** 10,000 ipvlan L3 endpoints:

$$M_{\text{total}} = 10000 \times 128 = 1.28 \text{ MB}$$

Negligible memory. Lookup: 8 trie levels regardless of table size.

Compare to macvlan with 10,000 entries in a 256-bucket hash:
- macvlan: $O(39)$ per lookup (chain traversal)
- ipvlan L3: $O(8)$ per lookup (trie traversal)

ipvlan L3 is approximately 5x faster for lookups at this scale.

---

## 5. Container Density and Network Namespace Overhead (Resource Modeling)

### The Problem

Each macvlan/ipvlan endpoint in a network namespace consumes kernel resources. What is the per-namespace overhead, and how many can a host support?

### The Formula

Per network namespace memory:

$$M_{\text{ns}} = M_{\text{base}} + M_{\text{iface}} + M_{\text{routes}} + M_{\text{conntrack}}$$

Where:
- $M_{\text{base}} \approx 4$ KB (namespace structure)
- $M_{\text{iface}} \approx 8$ KB (net_device + driver state)
- $M_{\text{routes}} \approx 1$ KB (default route + local)
- $M_{\text{conntrack}} \approx 0$ to 320 bytes/connection (if netfilter enabled)

For $N$ namespaces with $C$ connections each:

$$M_{\text{total}} = N \times (M_{\text{base}} + M_{\text{iface}} + M_{\text{routes}}) + N \times C \times 320$$

Maximum namespaces (memory-limited, ignoring conntrack):

$$N_{\max} = \frac{M_{\text{available}}}{M_{\text{base}} + M_{\text{iface}} + M_{\text{routes}}}$$

### Worked Examples

**Example 1:** Server with 16 GB RAM available for networking, no conntrack:

$$M_{\text{per-ns}} = 4 + 8 + 1 = 13 \text{ KB}$$

$$N_{\max} = \frac{16 \times 10^9}{13 \times 10^3} = 1,230,769$$

Over 1M namespaces possible from a memory perspective. In practice, file descriptor limits ($\sim 1M$) and pid limits ($\sim 32768$ default) are the bottleneck.

**Example 2:** With conntrack enabled, 100 connections per namespace:

$$M_{\text{per-ns}} = 13 + 100 \times 0.32 = 45 \text{ KB}$$

$$N_{\max} = \frac{16 \times 10^9}{45 \times 10^3} = 355,555$$

Conntrack reduces density by 3.5x.

---

## 6. Performance Comparison: Packets Per Second (Benchmarking Model)

### The Problem

Different virtual networking modes have different per-packet CPU costs. How do macvlan, ipvlan, and Linux bridge compare in maximum PPS?

### The Formula

For each forwarding path, the per-packet cost in CPU cycles:

$$\text{PPS}_{\max} = \frac{f_{\text{CPU}}}{C_{\text{per-packet}}}$$

Where $f_{\text{CPU}}$ is the CPU frequency (cycles/sec) and $C_{\text{per-packet}}$ is the total cycle cost.

Typical per-packet costs (measured on modern x86):

| Mode | $C_{\text{per-packet}}$ | Path |
|------|------------------------|------|
| macvlan bridge | ~2800 cycles | hash lookup + forward |
| ipvlan L2 | ~2600 cycles | MAC check + forward |
| ipvlan L3 | ~2400 cycles | route lookup + forward |
| Linux bridge | ~4200 cycles | FDB + STP + netfilter |
| veth + bridge | ~5500 cycles | veth + bridge + veth |

### Worked Examples

**Example:** Single core at 3 GHz:

| Mode | PPS | 64B line rate | 1500B throughput |
|------|-----|--------------|-----------------|
| macvlan bridge | 1.07M | 548 Mbps | 12.9 Gbps |
| ipvlan L3 | 1.25M | 640 Mbps | 15.0 Gbps |
| Linux bridge | 714K | 366 Mbps | 8.6 Gbps |
| veth + bridge | 545K | 279 Mbps | 6.5 Gbps |

macvlan is 1.5x faster than veth+bridge for small packets and 2x faster for container networking. ipvlan L3 is the fastest option when L2 features are not needed.

With 4 cores (RSS):

$$\text{PPS}_{\text{macvlan}} = 4 \times 1.07\text{M} = 4.28\text{M pps}$$

Enough to saturate a 25 Gbps NIC at 1500-byte frames.

---

## Prerequisites

- Combinatorics (counting, CAM table capacity, birthday problem)
- Hash table analysis (chain length, Poisson distribution)
- Network topology (hairpin routing, bandwidth doubling)
- Algorithmic complexity (trie lookup, hash vs tree performance)
- Systems resource modeling (memory per connection, CPU cycles per packet)
- Ethernet fundamentals (MAC addresses, ARP, broadcast domains)
