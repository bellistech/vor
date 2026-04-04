# The Mathematics of VLANs — ID Space, Broadcast Domains & TCAM Utilization

> *VLANs are a partitioning problem: how to divide a physical network into isolated logical segments using 12 bits of address space. The mathematics of broadcast domain sizing, TCAM memory, and QinQ stacking determine the practical limits of Layer 2 segmentation.*

---

## 1. VLAN ID Space (12-Bit Address Space)

### The Problem

The 802.1Q VLAN ID field is 12 bits. How many usable VLANs does this provide, and why is this a limiting factor?

### The Formula

$$N_{total} = 2^{12} = 4096$$

$$N_{usable} = 4096 - 2 = 4094$$

Reserved IDs:
- VID 0: Priority tagging only (no VLAN assignment)
- VID 4095: Reserved by IEEE

Practical limits on Cisco platforms:

$$N_{normal} = 1005 \quad (\text{VIDs 1-1005, stored in vlan.dat})$$
$$N_{extended} = 3089 \quad (\text{VIDs 1006-4094, require VTP transparent})$$

### QinQ (802.1ad) — Double Tagging

QinQ stacks two VLAN tags, multiplying the address space:

$$N_{QinQ} = 4094 \times 4094 = 16,760,836$$

Outer tag (S-VLAN, Service): identifies the customer/service provider
Inner tag (C-VLAN, Customer): identifies the customer's internal VLAN

This gives ~16.7 million unique identifiers, sufficient for carrier-scale networks.

### Comparison with VXLAN

$$N_{VXLAN} = 2^{24} = 16,777,216$$

VXLAN's 24-bit VNI provides virtually the same order of magnitude as QinQ but runs over Layer 3 (UDP encapsulation), eliminating the need for end-to-end Layer 2 connectivity.

---

## 2. Broadcast Domain Sizing (Scaling Analysis)

### The Problem

Each VLAN creates a separate broadcast domain. How does VLAN size affect broadcast traffic, and what is the optimal size?

### The Formula

Broadcast traffic within a VLAN of $N$ hosts:

$$B_{broadcast} = N \times R_{bcast} \times S_{frame}$$

Where:
- $R_{bcast}$ = broadcast rate per host (ARP, DHCP, NetBIOS, etc.)
- $S_{frame}$ = average broadcast frame size

Each host must process all broadcast frames in its VLAN:

$$\text{CPU interrupts per host} = (N - 1) \times R_{bcast}$$

### Worked Examples

Typical broadcast rate per host: ~5 broadcasts/second (ARP + DHCP + misc)
Average broadcast frame: 64 bytes

| VLAN Size $N$ | Broadcasts/sec | Bandwidth | CPU interrupts/host/sec |
|:---:|:---:|:---:|:---:|
| 10 | 50 | 25.6 kbps | 45 |
| 50 | 250 | 128 kbps | 245 |
| 100 | 500 | 256 kbps | 495 |
| 254 | 1,270 | 650 kbps | 1,265 |
| 500 | 2,500 | 1.28 Mbps | 2,495 |
| 1,000 | 5,000 | 2.56 Mbps | 4,995 |
| 5,000 | 25,000 | 12.8 Mbps | 24,995 |

### Optimal VLAN Size

Rule of thumb: keep VLANs under 250-500 hosts. Beyond this:

$$\text{Broadcast } \% = \frac{N \times R_{bcast} \times S_{frame} \times 8}{\text{Link Speed}} \times 100$$

At 1 Gbps:

| VLAN Size | Broadcast % of Link |
|:---:|:---:|
| 100 | 0.026% |
| 500 | 0.128% |
| 1,000 | 0.256% |
| 5,000 | 1.28% |
| 10,000 | 2.56% |

Bandwidth is rarely the bottleneck. The real limit is:
1. CPU processing of broadcasts on each host
2. ARP table sizes (kernel `gc_thresh3`)
3. MAC address table capacity on switches

---

## 3. TCAM Utilization Mathematics (Hardware Resources)

### The Problem

Switches store VLAN and MAC information in TCAM (Ternary Content-Addressable Memory). How much TCAM does each VLAN consume?

### The Formula

Each MAC address entry in TCAM:

$$S_{entry} = \text{MAC (48 bits)} + \text{VLAN (12 bits)} + \text{Port (log}_2 P\text{ bits)} + \text{metadata}$$

Total TCAM entries:

$$T_{total} = \sum_{v=1}^{V} MAC_v$$

Where $MAC_v$ is the number of unique MAC addresses learned in VLAN $v$.

### TCAM Sizing

| Switch Tier | Typical TCAM (MAC entries) | VLANs | MACs/VLAN | Utilization |
|:---|:---:|:---:|:---:|:---:|
| Access (low-end) | 8,192 | 10 | 50 | 6.1% |
| Access (mid-range) | 16,384 | 20 | 100 | 12.2% |
| Distribution | 32,768 | 100 | 200 | 61.0% |
| Core | 131,072 | 500 | 200 | 76.3% |
| DC leaf | 65,536 | 50 | 500 | 38.1% |

When TCAM is full, new MAC addresses cannot be learned, causing unknown unicast flooding for all unlearned destinations. This is a common problem on underpowered access switches in large flat networks.

### ACL TCAM per VLAN

VLAN ACLs (VACLs) and routed ACLs consume additional TCAM:

$$T_{ACL} = V \times R \times E$$

Where:
- $V$ = number of VLANs with ACLs
- $R$ = rules per ACL
- $E$ = expansion factor (1 rule may expand to multiple TCAM entries)

| VLANs with ACLs | Rules/ACL | Expansion (2x) | TCAM Entries |
|:---:|:---:|:---:|:---:|
| 10 | 50 | 2 | 1,000 |
| 50 | 100 | 2 | 10,000 |
| 100 | 200 | 2 | 40,000 |
| 200 | 500 | 2 | 200,000 |

Complex ACLs with ranges or wildcards have higher expansion factors (4-8x), rapidly exhausting TCAM.

---

## 4. Spanning Tree Scaling per VLAN (PVST+ vs MSTP)

### The Problem

PVST+ runs one STP instance per VLAN. How does this scale, and when should you switch to MSTP?

### The Formula

PVST+ resource consumption:

$$\text{CPU}_{\text{PVST+}} = V \times C_{STP} \quad \text{(V = VLAN count, C = per-instance CPU cost)}$$

$$\text{BPDUs}_{\text{PVST+}} = V \times \frac{P}{T_{hello}}$$

Where $P$ = trunk ports carrying this VLAN.

MSTP resource consumption:

$$\text{CPU}_{\text{MSTP}} = I \times C_{STP} \quad \text{(I = instance count, typically 2-16)}$$

### Worked Examples

Switch with 48 ports, 24 trunks:

| Protocol | Instances | BPDUs/sec (per trunk) | Total BPDUs/sec | CPU Relative |
|:---|:---:|:---:|:---:|:---:|
| STP (single) | 1 | 0.5 | 12 | 1x |
| PVST+ (100 VLANs) | 100 | 50 | 1,200 | 100x |
| PVST+ (500 VLANs) | 500 | 250 | 6,000 | 500x |
| PVST+ (1000 VLANs) | 1000 | 500 | 12,000 | 1000x |
| MSTP (4 instances) | 4 | 2 | 48 | 4x |
| MSTP (16 instances) | 16 | 8 | 192 | 16x |

MSTP reduces STP overhead by 50-250x compared to PVST+ at scale.

---

## 5. VLAN Trunk Bandwidth Overhead (Frame Expansion)

### The Problem

802.1Q tagging adds 4 bytes to every frame. What is the bandwidth overhead?

### The Formula

Overhead per frame:

$$O_{frame} = \frac{4}{S_{frame}} \times 100\%$$

Aggregate bandwidth overhead for $R$ frames/sec:

$$B_{overhead} = R \times 4 \times 8 = 32R \text{ bps}$$

### Worked Examples

| Frame Size (untagged) | Tagged Size | Overhead |
|:---:|:---:|:---:|
| 64 bytes (min) | 68 bytes | 6.25% |
| 128 bytes | 132 bytes | 3.13% |
| 512 bytes | 516 bytes | 0.78% |
| 1500 bytes (max) | 1504 bytes | 0.27% |

For QinQ (double tag, 8 bytes additional):

| Frame Size (untagged) | QinQ Size | Overhead |
|:---:|:---:|:---:|
| 64 bytes | 72 bytes | 12.5% |
| 1500 bytes | 1508 bytes | 0.53% |

### Impact on Throughput

At 1 Gbps line rate with minimum-size frames (worst case):

$$\text{Untagged PPS} = \frac{10^9}{(64 + 20) \times 8} = 1,488,095 \text{ pps}$$

$$\text{Tagged PPS} = \frac{10^9}{(68 + 20) \times 8} = 1,420,454 \text{ pps}$$

$$\text{Throughput loss} = \frac{1,488,095 - 1,420,454}{1,488,095} \times 100 = 4.55\%$$

The 20-byte inter-frame gap and preamble are included. For average-sized frames (~500 bytes), overhead is under 0.3%.

---

## 6. Private VLANs (Port Isolation Mathematics)

### The Problem

Private VLANs (PVLANs) subdivide a VLAN into isolated ports. How many isolation groups can be created?

### The Formula

A PVLAN uses three VLAN types:
- **Primary VLAN**: The parent VLAN
- **Isolated VLAN**: All ports in one group, cannot communicate with each other (max 1 per primary)
- **Community VLANs**: Ports within a community can communicate; different communities are isolated

$$N_{communities} = N_{available\_VLANs} - 1_{\text{primary}} - 1_{\text{isolated}}$$

In practice, community VLANs are limited by VLAN ID availability. For a /24 subnet with 254 hosts:

| Configuration | VLANs Used | Host Communication Pattern |
|:---|:---:|:---|
| No PVLAN | 1 | All-to-all |
| Isolated only | 2 | All isolated, only via promiscuous port |
| 5 communities | 7 | 5 groups + isolated + promiscuous |
| Per-host isolation | 2 | Each host isolated (shared isolated VLAN) |

---

## 7. Summary of Formulas

| Formula | Math Type | Domain |
|:---|:---|:---|
| $2^{12} - 2 = 4094$ | Power of 2 | VLAN ID space |
| $4094 \times 4094 = 16.7M$ | Multiplication | QinQ space |
| $N \times R_{bcast} \times S$ | Product | Broadcast bandwidth |
| $\sum MAC_v$ | Summation | TCAM utilization |
| $V \times R \times E$ | Product | ACL TCAM |
| $V \times P / T_{hello}$ | Rate | PVST+ BPDUs |
| $4 / S_{frame}$ | Ratio | Tagging overhead |

## Prerequisites

- binary encoding, broadcast domains, TCAM architecture, spanning tree

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| VLAN tag insertion/removal | $O(1)$ hardware | $O(1)$ |
| MAC lookup (VLAN+MAC) | $O(1)$ TCAM | $O(N)$ entries |
| VLAN membership check | $O(1)$ bitmap | $O(4094/8)$ = 512 bytes |
| Broadcast forwarding | $O(P)$ ports in VLAN | $O(1)$ per frame |
| STP per VLAN (PVST+) | $O(V \times D \times T)$ | $O(V \times P)$ states |
| MSTP (all instances) | $O(I \times D \times T)$ | $O(I \times P)$ states |

---

*VLANs solve the broadcast scaling problem with 12 bits and a 4-byte tag. But 4094 is a hard ceiling baked into Ethernet headers, and the real limits are often reached much sooner: TCAM exhaustion at a few thousand MACs, PVST+ CPU saturation at hundreds of VLANs, and broadcast storms at thousands of hosts per VLAN. The math tells you exactly where each wall is.*
