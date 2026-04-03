# The Mathematics of VXLAN — Overlay Capacity, VTEP Scaling, and Encapsulation Overhead

> *VXLAN extends Layer 2 across Layer 3 boundaries by wrapping Ethernet frames in UDP packets. The math centers on the 24-bit VNI address space, the N-squared VTEP scaling problem, and the encapsulation overhead that shrinks your effective MTU.*

---

## 1. VNI Address Space — 24-Bit Capacity

### The Formula

$$N_{VNI} = 2^{24} = 16,777,216 \text{ network segments}$$

Compare with VLAN's 12-bit ID:

$$N_{VLAN} = 2^{12} = 4,096$$

**Expansion factor:**

$$\frac{2^{24}}{2^{12}} = 2^{12} = 4,096\times$$

### Why 4,096 VLANs Wasn't Enough

| Environment | Segments Needed | VLANs Sufficient? | VNIs Sufficient? |
|:---|:---:|:---:|:---:|
| Small campus | 50 | Yes | Yes |
| Enterprise DC | 500 | Yes | Yes |
| Multi-tenant cloud | 10,000 | **No** | Yes |
| Hyperscaler | 100,000+ | **No** | Yes |
| Theoretical max | 16.7M | **No** | Yes |

### Reserved VNIs

VNI 0 is reserved. Usable: $2^{24} - 1 = 16,777,215$.

---

## 2. VTEP Scaling — The N-Squared Problem

### The Problem

Each VTEP (VXLAN Tunnel Endpoint) must know about every other VTEP in the same VNI segment. With $N$ VTEPs per segment:

### Flood-and-Learn (Multicast/Ingress Replication)

$$\text{BUM copies per frame} = N - 1$$

Where BUM = Broadcast, Unknown unicast, Multicast.

### Total BUM Traffic

For $F_{BUM}$ BUM frames/sec from each VTEP:

$$\text{Total BUM traffic} = N \times F_{BUM} \times (N - 1) \times S_{frame}$$

This is $O(N^2)$ — the fundamental scaling problem.

### Worked Examples

| VTEPs per VNI | BUM frames/sec/VTEP | Frame Size | Total BUM Bandwidth |
|:---:|:---:|:---:|:---:|
| 10 | 100 | 128 B | 115 KB/s |
| 50 | 100 | 128 B | 31 MB/s |
| 100 | 100 | 128 B | 127 MB/s |
| 500 | 100 | 128 B | 3.2 GB/s |
| 1,000 | 100 | 128 B | 12.7 GB/s |

At 1,000 VTEPs, BUM replication alone consumes over 100 Gbps of aggregate bandwidth.

### The Solution: EVPN Control Plane

EVPN (RFC 7432) replaces flood-and-learn with BGP-based MAC learning:

$$\text{BUM with EVPN} = O(N) \quad \text{(multicast groups or selective replication)}$$

| Method | BUM Scaling | Control Plane | Complexity |
|:---|:---:|:---|:---:|
| Ingress replication | $O(N^2)$ | None (data plane) | Simple |
| Multicast groups | $O(N)$ | PIM/IGMP | Medium |
| EVPN + selective | $O(1)$ per known MAC | BGP | Complex |

---

## 3. Encapsulation Overhead

### VXLAN Header Stack

| Layer | Size | Fields |
|:---|:---:|:---|
| Outer Ethernet | 14 B | Dst MAC, Src MAC, EtherType |
| Outer IP (IPv4) | 20 B | Src/Dst VTEP IPs |
| Outer UDP | 8 B | Src port (hash), Dst port (4789) |
| VXLAN header | 8 B | Flags, VNI (24-bit) |
| **Total overhead** | **50 B** | |

With outer IPv6: $14 + 40 + 8 + 8 = 70$ bytes.

### Effective MTU

$$MTU_{inner} = MTU_{transport} - O_{VXLAN}$$

| Transport MTU | IPv4 Overlay | IPv6 Overlay |
|:---:|:---:|:---:|
| 1,500 | 1,450 | 1,430 |
| 9,000 (jumbo) | 8,950 | 8,930 |
| 9,216 | 9,166 | 9,146 |

### Throughput Overhead Percentage

$$O\% = \frac{O_{VXLAN}}{MTU_{transport}} \times 100$$

| Transport MTU | Overhead % (IPv4) | Overhead % (IPv6) |
|:---:|:---:|:---:|
| 1,500 | 3.3% | 4.7% |
| 9,000 | 0.6% | 0.8% |

Jumbo frames reduce VXLAN overhead from ~3.3% to ~0.6%.

---

## 4. Entropy and ECMP — Source Port Hashing

### The Problem

All VXLAN traffic goes to UDP port 4789. Without additional entropy, ECMP hashing treats all VXLAN traffic as a single flow.

### The Solution

The outer UDP source port is derived from a hash of the inner frame:

$$\text{Src Port} = H(\text{inner headers}) \mod (65535 - 49152) + 49152$$

This provides $65535 - 49152 = 16,383$ possible source ports for ECMP distribution.

### ECMP Distribution Quality

With $K$ ECMP paths and $F$ flows:

$$\text{Expected flows per path} = \frac{F}{K}$$

$$\text{Standard deviation} = \sqrt{\frac{F(K-1)}{K^2}} \approx \frac{\sqrt{F}}{\sqrt{K}}$$

$$\text{Imbalance} = \frac{\sigma}{\mu} = \frac{\sqrt{K-1}}{\sqrt{F}}$$

| Flows | 2 ECMP paths | 4 ECMP paths | 8 ECMP paths |
|:---:|:---:|:---:|:---:|
| 100 | 10% imbalance | 17% imbalance | 26% imbalance |
| 1,000 | 3.2% | 5.5% | 8.4% |
| 10,000 | 1.0% | 1.7% | 2.6% |

More flows = better ECMP distribution. This is why data center fabrics with thousands of flows achieve near-perfect load balancing.

---

## 5. ARP/ND Suppression — Broadcast Reduction

### The Problem

ARP in a VXLAN segment is broadcast, triggering BUM replication to all VTEPs. With $H$ hosts per segment:

$$\text{ARP rate} \approx H \times \frac{N_{destinations}}{T_{ARP\_cache}}$$

Where $T_{ARP\_cache}$ = ARP cache timeout (typically 300 seconds).

### ARP Suppression Savings

VTEP caches ARP responses and answers locally:

$$\text{Suppressed ARPs} = ARP_{total} \times P_{cache\_hit}$$

| Hosts | ARP/sec (no suppression) | ARP/sec (with suppression) | BUM Reduction |
|:---:|:---:|:---:|:---:|
| 100 | 33 | 3 | 90% |
| 1,000 | 333 | 33 | 90% |
| 10,000 | 3,333 | 333 | 90% |

---

## 6. Spine-Leaf Scaling Model

### Clos Network Capacity

A 2-tier Clos (spine-leaf) fabric with $S$ spines and $L$ leaf switches, each with $P$ ports:

$$\text{Bisection BW} = S \times L \times BW_{link}$$

$$\text{Oversubscription} = \frac{L \times P_{south} \times BW_{south}}{L \times S \times BW_{north}}$$

### VTEP Count in Spine-Leaf

$$N_{VTEPs} = L \quad \text{(one VTEP per leaf)}$$

| Fabric Size | Spines | Leaves | Servers (48 ports/leaf) | VTEPs |
|:---|:---:|:---:|:---:|:---:|
| Small | 2 | 8 | 384 | 8 |
| Medium | 4 | 32 | 1,536 | 32 |
| Large | 8 | 64 | 3,072 | 64 |
| Hyperscale | 32 | 512 | 24,576 | 512 |

---

## 7. Summary of Formulas

| Formula | Math Type | Application |
|:---|:---|:---|
| $2^{24} = 16,777,216$ | Exponent | VNI address space |
| $N(N-1)$ BUM copies | Quadratic | Flood-and-learn scaling |
| $MTU - 50$ (IPv4) | Subtraction | Effective inner MTU |
| $H(\text{inner}) \mod R + B$ | Hash/modular | ECMP source port |
| $\sqrt{K-1}/\sqrt{F}$ | Statistical | ECMP imbalance |
| $S \times L \times BW$ | Product | Bisection bandwidth |

---

*VXLAN solved the 4,096 VLAN limit by moving network segmentation into an overlay, but it traded one scaling problem for another — the N-squared BUM replication that forced the industry to build EVPN. The math of encapsulation overhead and VTEP scaling governs every modern data center fabric.*
