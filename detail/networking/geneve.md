# The Mathematics of Geneve — Encapsulation Overhead & Tunnel Capacity Analysis

> *Geneve's extensible header trades fixed simplicity for variable overhead, introducing a packing problem for TLV options, an entropy problem for ECMP distribution, and a capacity problem where every byte of tunnel header directly reduces the payload budget on MTU-constrained fabrics.*

---

## 1. Encapsulation Overhead (Byte Budget Accounting)

### The Formula

Total Geneve overhead for IPv4 underlay:

$$O_{geneve} = H_{eth} + H_{ip} + H_{udp} + H_{base} + H_{opts}$$

$$O_{geneve} = 14 + 20 + 8 + 8 + 4 \times L_{opt} = 50 + 4L_{opt} \text{ bytes}$$

Where $L_{opt}$ is the Opt Len field (0--63), representing TLV option space in 4-byte words.

For IPv6 underlay:

$$O_{geneve}^{v6} = 14 + 40 + 8 + 8 + 4L_{opt} = 70 + 4L_{opt} \text{ bytes}$$

### Inner MTU

Given physical MTU $M_{phys}$:

$$M_{inner} = M_{phys} - O_{geneve}$$

| Underlay | $M_{phys}$ | Options (bytes) | $O_{geneve}$ | $M_{inner}$ |
|:---|:---:|:---:|:---:|:---:|
| IPv4 | 1500 | 0 | 50 | 1450 |
| IPv4 | 1500 | 16 | 66 | 1434 |
| IPv6 | 1500 | 0 | 70 | 1430 |
| IPv4 | 9000 | 0 | 50 | 8950 |
| IPv4 | 9000 | 64 | 114 | 8886 |
| IPv6 | 9000 | 64 | 134 | 8866 |

### Overhead Ratio

The encapsulation tax as a fraction of each packet:

$$\eta = \frac{O_{geneve}}{M_{phys}}$$

For a 1500-byte MTU with no options: $\eta = 50/1500 = 3.33\%$

For small packets (e.g., 64-byte inner frame):

$$\eta_{small} = \frac{50}{64 + 50} = 43.9\%$$

This overhead ratio matters most for workloads dominated by small packets (DNS, VoIP, ACKs).

---

## 2. TLV Option Packing (Bin Packing on 4-Byte Boundaries)

### Alignment Constraint

Each TLV option occupies:

$$S_{tlv} = 4 + \lceil D / 4 \rceil \times 4 \text{ bytes}$$

Where $D$ is the data length and the 4-byte option header is always present. The total option space is capped at:

$$H_{opts} \leq 4 \times 63 = 252 \text{ bytes}$$

### Packing Efficiency

For $k$ options with data sizes $d_1, d_2, \ldots, d_k$:

$$H_{opts} = \sum_{i=1}^{k} \left(4 + 4\lceil d_i/4 \rceil\right)$$

Wasted bytes due to alignment padding:

$$W = \sum_{i=1}^{k} \left(4\lceil d_i/4 \rceil - d_i\right)$$

| Option Data Size | Aligned Size | Header | Total | Waste |
|:---:|:---:|:---:|:---:|:---:|
| 1 byte | 4 | 4 | 8 | 3 |
| 4 bytes | 4 | 4 | 8 | 0 |
| 5 bytes | 8 | 4 | 12 | 3 |
| 8 bytes | 8 | 4 | 12 | 0 |
| 12 bytes | 12 | 4 | 16 | 0 |
| 16 bytes | 16 | 4 | 20 | 0 |

Design TLV data fields as multiples of 4 bytes to minimize waste. A 1-byte option wastes 37.5% of its allocated space.

### Maximum Options Count

With minimum-size options (0 bytes of data, 4-byte header only):

$$k_{max} = \lfloor 252 / 4 \rfloor = 63$$

With typical 8-byte options (4 header + 4 data):

$$k_{typical} = \lfloor 252 / 8 \rfloor = 31$$

---

## 3. ECMP Entropy Distribution (Hash Uniformity)

### The Problem

Geneve uses the outer UDP source port as an entropy field for ECMP. The source port is computed as a hash of inner packet headers. How well does this distribute traffic across $N$ ECMP paths?

### Hash Model

Let $H$ be the hash function mapping inner flow tuples to 16-bit source port values (range $[49152, 65535]$ per Linux defaults = 16,384 values). For $F$ flows across $N$ paths:

Expected flows per path:

$$E[f_i] = \frac{F}{N}$$

Standard deviation (assuming uniform hash):

$$\sigma = \sqrt{\frac{F(N-1)}{N^2}} \approx \sqrt{\frac{F}{N}} \text{ for large } N$$

The coefficient of variation (imbalance metric):

$$CV = \frac{\sigma}{E[f_i]} = \frac{1}{\sqrt{F/N}} = \sqrt{\frac{N}{F}}$$

| Flows $F$ | Paths $N$ | Expected/Path | $CV$ (imbalance) |
|:---:|:---:|:---:|:---:|
| 100 | 4 | 25 | 20.0% |
| 1,000 | 4 | 250 | 6.3% |
| 10,000 | 4 | 2,500 | 2.0% |
| 100 | 16 | 6.25 | 40.0% |
| 1,000 | 16 | 62.5 | 12.6% |
| 10,000 | 16 | 625 | 4.0% |

With few flows and many paths, ECMP imbalance is significant. Elephant flows (large, long-lived) exacerbate this because their hash slot carries disproportionate bandwidth.

---

## 4. VNI Address Space and Tenant Density (Combinatorial Capacity)

### VNI Space

The 24-bit VNI field provides:

$$|VNI| = 2^{24} = 16,777,216 \text{ segments}$$

Compared to VLAN's 12-bit ID:

$$\frac{2^{24}}{2^{12}} = 2^{12} = 4,096\times \text{ more segments}$$

### Tenant Isolation Capacity

If each tenant requires $s$ segments (microsegmentation), maximum tenants:

$$T_{max} = \left\lfloor \frac{2^{24}}{s} \right\rfloor$$

| Segments per Tenant $s$ | Max Tenants |
|:---:|:---:|
| 1 | 16,777,216 |
| 10 | 1,677,721 |
| 100 | 167,772 |
| 1,000 | 16,777 |

### FDB Table Scaling

Each tunnel endpoint must maintain an FDB (forwarding database) mapping inner MAC addresses to remote tunnel endpoints. For $T$ tenants, $H$ hosts per tenant, and $E$ endpoints:

$$|FDB| = T \times H$$

Memory per FDB entry (MAC + VNI + remote IP + timer):

$$M_{fdb} \approx 6 + 3 + 4 + 8 = 21 \text{ bytes} \approx 32 \text{ bytes (aligned)}$$

| Tenants | Hosts/Tenant | FDB Entries | Memory |
|:---:|:---:|:---:|:---:|
| 100 | 50 | 5,000 | 160 KB |
| 1,000 | 100 | 100,000 | 3.2 MB |
| 10,000 | 100 | 1,000,000 | 32 MB |

---

## 5. Throughput Loss from Encapsulation (Goodput Analysis)

### Effective Goodput

For a link of capacity $C$ (bps), sending inner frames of size $P$ bytes:

$$G = C \times \frac{P}{P + O_{geneve}}$$

Throughput efficiency:

$$\epsilon = \frac{P}{P + O_{geneve}}$$

| Inner Frame $P$ | Overhead $O$ | Efficiency $\epsilon$ | Goodput at 100 Gbps |
|:---:|:---:|:---:|:---:|
| 64 | 50 | 56.1% | 56.1 Gbps |
| 128 | 50 | 71.9% | 71.9 Gbps |
| 512 | 50 | 91.1% | 91.1 Gbps |
| 1450 | 50 | 96.7% | 96.7 Gbps |
| 8950 | 50 | 99.4% | 99.4 Gbps |

Jumbo frames on the underlay provide near-wire-rate goodput. Small-packet workloads suffer significant encapsulation tax.

### Packets Per Second Impact

Encapsulation adds per-packet CPU cost $c_{encap}$ (without hardware offload). If the CPU can process $R$ packets/second raw:

$$R_{geneve} = \frac{R}{1 + c_{encap}/c_{base}}$$

Typical values without offload: $c_{encap} \approx 0.3 \times c_{base}$ (30% additional cost):

$$R_{geneve} \approx 0.77 \times R$$

With hardware offload, $c_{encap} \approx 0$ and $R_{geneve} \approx R$.

---

## 6. Tunnel Endpoint Discovery (Flooding Cost)

### BUM Traffic in Overlay Networks

Broadcast, Unknown unicast, and Multicast (BUM) traffic must reach all endpoints in a VNI. With $E$ endpoints per VNI:

Head-end replication cost per BUM frame:

$$C_{BUM} = (E - 1) \times (P + O_{geneve})$$

For $E = 100$ endpoints and a 1500-byte BUM frame:

$$C_{BUM} = 99 \times 1550 = 153,450 \text{ bytes} = 1.23 \text{ Mbit}$$

BUM frames per second with $B$ broadcast rate:

$$BW_{BUM} = B \times (E - 1) \times (P + O_{geneve}) \times 8$$

| Endpoints $E$ | BUM Rate $B$ (pps) | Bandwidth per Source |
|:---:|:---:|:---:|
| 10 | 100 | 11.2 Mbps |
| 50 | 100 | 60.8 Mbps |
| 100 | 100 | 122.8 Mbps |
| 100 | 1,000 | 1.23 Gbps |

This is why overlay networks aggressively suppress BUM traffic using EVPN for MAC advertisement or ARP proxy/suppression.

---

## 7. Option Processing Latency (Critical Path Analysis)

### TLV Parsing Cost

A Geneve receiver must parse $k$ TLV options sequentially (linked-list traversal):

$$T_{parse} = k \times (t_{read} + t_{match} + t_{advance})$$

Where $t_{read}$ is the memory read time for the 4-byte header, $t_{match}$ is the option class/type lookup, and $t_{advance}$ is the pointer increment.

In software (OVS):

$$T_{parse} \approx k \times 10 \text{ ns} \text{ (L1 cache hit)}$$

The Critical bit introduces a branch:

$$T_{critical} = T_{parse} + k_{unknown} \times t_{drop\_decision}$$

If C=1 and any option is not recognized, the entire packet is dropped. This makes deployment order critical: upgrade all receivers before senders enable new critical options.

### Hardware Parsing

Hardware offload engines use fixed-depth TLV parsers. Typical limits:

| NIC Generation | Max Options Parsed | Max Option Space |
|:---|:---:|:---:|
| ConnectX-5 | 8 | 64 bytes |
| ConnectX-6 Dx | 16 | 128 bytes |
| Intel E810 | 4 | 32 bytes |
| P4-programmable | Configurable | 252 bytes |

Exceeding the hardware parser depth falls back to software, negating offload benefits.

---

*Geneve's mathematics expose a fundamental engineering tradeoff: extensibility via TLV options costs bytes per packet that reduce goodput, alignment waste that reduces option density, and parsing latency that limits hardware offload depth. The protocol is optimal when options are few, 4-byte-aligned, and pre-negotiated across all endpoints — making the variable header act like a well-planned fixed one.*

## Prerequisites

- Basic information theory (entropy, hashing, collision probability)
- Combinatorics (bin packing, birthday problem for ECMP)
- Queueing theory (per-packet processing cost, goodput vs throughput)

## Complexity

- **Beginner:** Overhead calculation, inner MTU derivation, VNI capacity
- **Intermediate:** ECMP entropy distribution, TLV packing efficiency, BUM flooding cost
- **Advanced:** Hardware parser depth limits, critical-bit deployment ordering, small-packet goodput degradation
