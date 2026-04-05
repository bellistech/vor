# MPLS VPN — Virtual Private Network Architecture over MPLS

> *MPLS VPN uses a two-label stack and MP-BGP route distribution to create isolated virtual networks over shared infrastructure. The math covers VRF scaling combinatorics, Route Target algebra, label space allocation, inter-AS topology models, VPLS full-mesh growth, and the forwarding pipeline from CE ingress to CE egress.*

---

## 1. Route Distinguisher — Address Space Extension

### The Problem

Multiple customers can use the same private IP address space (e.g., 10.0.0.0/8). BGP requires globally unique prefixes. The Route Distinguisher (RD) extends the address space to prevent collisions.

### VPNv4 Address Construction

A VPNv4 address is the concatenation of an 8-byte RD and a 4-byte IPv4 prefix:

$$\text{VPNv4 address} = \text{RD}_{8B} \| \text{IPv4 prefix}_{4B}$$

Total VPNv4 address size: 12 bytes (96 bits) vs IPv4's 4 bytes (32 bits).

### RD Formats

| Type | Format | RD Composition | Example |
|:---:|:---|:---|:---|
| 0 | ASN(2B):Value(4B) | 2-byte ASN + 4-byte number | 65000:100 |
| 1 | IP(4B):Value(2B) | 4-byte IP + 2-byte number | 10.0.0.1:100 |
| 2 | ASN(4B):Value(2B) | 4-byte ASN + 2-byte number | 65536:100 |

### Uniqueness Guarantee

With Type 0 RD and a 2-byte ASN (65,535 possible ASNs) and 4-byte value:

$$\text{Unique RDs per ASN} = 2^{32} = 4,294,967,296$$

$$\text{Total unique RDs (Type 0)} = 2^{16} \times 2^{32} = 2^{48} \approx 2.81 \times 10^{14}$$

Even with one unique RD per VRF per PE, this is effectively unlimited.

### RD Does Not Affect Routing

Critical distinction: the RD makes prefixes unique in BGP, but it is stripped before the prefix enters the VRF routing table. Two VPNv4 routes with different RDs but the same IPv4 prefix and matching Route Targets can both be imported into the same VRF — this is how multi-homed VPNs work.

---

## 2. Route Target — Set-Theoretic Import/Export

### RT as Set Operations

Each VRF has two sets of Route Targets:

$$\text{Export RTs:} \quad E(v) = \{rt_1, rt_2, \ldots\}$$
$$\text{Import RTs:} \quad I(v) = \{rt_1, rt_2, \ldots\}$$

A VPNv4 route $r$ originated by VRF $v_1$ with export RTs $E(v_1)$ is imported into VRF $v_2$ iff:

$$E(v_1) \cap I(v_2) \neq \emptyset$$

At least one export RT must match at least one import RT.

### Full-Mesh VPN (Any-to-Any)

For a simple full-mesh VPN with $n$ sites, all VRFs use the same RT for both import and export:

$$E(v_i) = I(v_i) = \{rt_{vpn}\} \quad \forall i \in \{1, \ldots, n\}$$

Every VRF exports with $rt_{vpn}$ and imports $rt_{vpn}$, so all sites see all routes.

### Hub-and-Spoke VPN

Hub site VRF:
$$E(hub) = \{rt_{hub}\}$$
$$I(hub) = \{rt_{spoke}\}$$

Spoke site VRF:
$$E(spoke_i) = \{rt_{spoke}\}$$
$$I(spoke_i) = \{rt_{hub}\}$$

Traffic flow: spoke exports with $rt_{spoke}$, hub imports $rt_{spoke}$. Hub exports with $rt_{hub}$, spoke imports $rt_{hub}$. Spokes cannot reach each other directly (no matching RT).

### Shared Services (Extranet)

Shared services VRF (e.g., DNS, NTP):
$$E(shared) = \{rt_{shared}\}$$
$$I(shared) = \{rt_A, rt_B, rt_C\}$$

Customer VRFs:
$$E(v_A) = \{rt_A\}, \quad I(v_A) = \{rt_A, rt_{shared}\}$$
$$E(v_B) = \{rt_B\}, \quad I(v_B) = \{rt_B, rt_{shared}\}$$

Result: Customers A and B can reach shared services but not each other. The shared services VRF receives routes from all customers (to provide return reachability).

### RT Combinatorics

With $n$ VRFs and $m$ available RTs, the number of possible VPN topologies:

Each VRF independently chooses a subset of RTs for import and a subset for export:

$$\text{Configurations per VRF} = 2^m \times 2^m = 2^{2m}$$

$$\text{Total topologies} = (2^{2m})^n = 2^{2mn}$$

For 10 VRFs and 5 available RTs: $2^{100} \approx 1.27 \times 10^{30}$ possible topologies. This extreme flexibility is why RT-based VPN design can express virtually any connectivity policy.

---

## 3. Label Allocation and Forwarding

### Two-Label Stack

In an L3VPN, the packet carries two labels:

$$\text{Label Stack} = [L_{transport}, L_{VPN}]$$

- $L_{transport}$: Outer label, allocated by LDP or RSVP-TE, identifies the LSP to the egress PE
- $L_{VPN}$: Inner label, allocated by BGP on the egress PE, identifies the destination VRF

### Label Allocation Models

| Model | Labels per PE | Total Labels in Network | Description |
|:---|:---:|:---|:---|
| Per-VRF | $V$ | $N_{PE} \times V$ | One label per VRF per PE |
| Per-prefix | $\sum_{v} |P_v|$ | $N_{PE} \times \sum_v |P_v|$ | One label per VPNv4 prefix per PE |
| Per-CE | $\sum_{v} C_v$ | $N_{PE} \times \sum_v C_v$ | One label per CE neighbor per PE |

Where $V$ = VRFs, $|P_v|$ = prefixes in VRF $v$, $C_v$ = CE neighbors in VRF $v$, $N_{PE}$ = number of PEs.

### Per-VRF vs Per-Prefix Trade-off

**Per-VRF labeling:**
- Fewer labels consumed: $V$ labels per PE (e.g., 100 VRFs = 100 labels)
- Egress PE must do an IP lookup after popping VPN label (to determine CE next-hop)
- Cisco IOS default for VPNv4

**Per-prefix labeling:**
- More labels: could be thousands per PE
- Egress PE can forward directly based on label (no IP lookup)
- Required for certain features (e.g., per-prefix traffic statistics)

### Forwarding Pipeline — Step by Step

Given: CE1 in VRF-A on PE1 sends packet to CE2 in VRF-A on PE2.

Step 1 — CE1 to PE1 (IP):
$$\text{CE1} \xrightarrow{dst=10.10.0.1} \text{PE1}$$

Step 2 — PE1 VRF lookup:
$$\text{VRF-A table: } 10.10.0.0/16 \rightarrow \text{NH=PE2 (10.0.0.2), VPN label=500}$$
$$\text{LDP/LFIB: } 10.0.0.2/32 \rightarrow \text{transport label=300, out-if=Gi0/0}$$

Step 3 — PE1 pushes labels:
$$\text{PE1} \xrightarrow{[300, 500] + IP} \text{P1}$$

Step 4 — P1 swaps transport label:
$$\text{P1: } 300 \rightarrow 400 \quad \text{P1} \xrightarrow{[400, 500] + IP} \text{P2}$$

Step 5 — P2 pops transport label (PHP):
$$\text{P2: pop 400} \quad \text{P2} \xrightarrow{[500] + IP} \text{PE2}$$

Step 6 — PE2 processes VPN label:
$$\text{PE2: label 500} \rightarrow \text{VRF-A, NH=CE2 (192.168.2.2)}$$
$$\text{PE2} \xrightarrow{dst=10.10.0.1} \text{CE2}$$

---

## 4. VRF Scaling — Memory and RIB Analysis

### Per-VRF Memory Cost

Each VRF requires:
- Routing table (RIB): proportional to number of prefixes $|P_v|$
- Forwarding table (FIB/CEF): proportional to $|P_v|$
- Interface state
- Protocol adjacencies (OSPF, BGP, EIGRP)

Memory per VRF:

$$M_{VRF}(v) = M_{base} + |P_v| \times m_{prefix} + A_v \times m_{adj}$$

Where:
- $M_{base}$: base VRF overhead (~10-50 KB)
- $m_{prefix}$: memory per route entry (~200-500 bytes)
- $A_v$: number of protocol adjacencies in VRF $v$
- $m_{adj}$: memory per adjacency (~1-5 KB)

### Worked Example

PE with 500 VRFs, average 100 prefixes per VRF, 2 adjacencies per VRF:

$$M_{total} = 500 \times (50\text{KB} + 100 \times 0.5\text{KB} + 2 \times 5\text{KB})$$
$$M_{total} = 500 \times (50 + 50 + 10)\text{KB} = 500 \times 110\text{KB} = 55\text{MB}$$

VPNv4 BGP table (on PE or RR): all VRFs contribute routes to a single table:

$$|BGP_{vpnv4}| = \sum_{v=1}^{V} |P_v| = 500 \times 100 = 50,000 \text{ VPNv4 routes}$$

At ~500 bytes per BGP entry: $50,000 \times 500 = 25\text{MB}$ for BGP table alone.

### Route Reflector Scaling

An RR holds VPNv4 routes from all PEs. For a network with $N$ PEs, each with $V$ VRFs and $P$ prefixes per VRF:

$$|RR_{table}| = N \times V \times P$$

| PEs | VRFs/PE | Prefixes/VRF | Total VPNv4 Routes | BGP Memory (~500B/route) |
|:---:|:---:|:---:|:---:|:---:|
| 10 | 100 | 50 | 50,000 | 25 MB |
| 50 | 200 | 100 | 1,000,000 | 500 MB |
| 100 | 500 | 100 | 5,000,000 | 2.5 GB |
| 200 | 1000 | 200 | 40,000,000 | 20 GB |

At scale, RT-Constrain (RFC 4684) is essential: PEs advertise which RTs they import, and RRs only send relevant routes to each PE, dramatically reducing per-PE memory.

### RT-Constrain Savings

Without RT-Constrain, every PE receives all $N \times V \times P$ VPNv4 routes and filters locally.

With RT-Constrain, a PE with $V_{local}$ VRFs receives only:

$$|PE_{filtered}| = V_{local} \times P \times F$$

Where $F$ is the fanout factor (how many PEs share the same RT). For a PE with 10 VRFs where each VPN spans 5 PEs:

$$|PE_{filtered}| = 10 \times 100 \times 5 = 5,000 \text{ routes}$$

vs 1,000,000 routes without filtering. Reduction: 99.5%.

---

## 5. VPLS — Full-Mesh Pseudowire Scaling

### Full-Mesh Growth

For $N$ PEs in a VPLS instance, the number of pseudowires required:

$$PW = \frac{N(N-1)}{2}$$

This is the handshake problem (same as iBGP full-mesh).

### Scaling Table

| PEs ($N$) | Pseudowires | Growth Factor |
|:---:|:---:|:---:|
| 3 | 3 | - |
| 5 | 10 | 3.3x |
| 10 | 45 | 4.5x |
| 20 | 190 | 4.2x |
| 50 | 1,225 | 6.4x |
| 100 | 4,950 | 4.0x |

Growth is $O(N^2)$. Doubling PEs roughly quadruples pseudowires.

### H-VPLS Reduction

Hierarchical VPLS with $H$ hub PEs and $S$ spoke PEs (where $N = H + S$):

$$PW_{H-VPLS} = \frac{H(H-1)}{2} + S \times R$$

Where $R$ is the hub redundancy factor (typically 1 or 2: each spoke connects to 1 or 2 hubs).

### Worked Example

50-PE VPLS deployment:
- Full mesh: $\frac{50 \times 49}{2} = 1,225$ pseudowires
- H-VPLS with 5 hubs, 45 spokes, dual-homed: $\frac{5 \times 4}{2} + 45 \times 2 = 10 + 90 = 100$ pseudowires
- Reduction: 91.8%

### MAC Table Scaling in VPLS

Each PE learns MAC addresses from all sites in the VPLS instance. For $N$ sites with $M$ MAC addresses each:

$$|MAC_{table}| = N \times M$$

| Sites | MACs/Site | Total MAC Entries | Memory (64B/entry) |
|:---:|:---:|:---:|:---:|
| 10 | 100 | 1,000 | 64 KB |
| 50 | 500 | 25,000 | 1.6 MB |
| 100 | 1,000 | 100,000 | 6.4 MB |
| 200 | 5,000 | 1,000,000 | 64 MB |

MAC aging (default 300 seconds) limits table size. BUM (Broadcast, Unknown unicast, Multicast) traffic is flooded to all PEs — this is the fundamental scalability limit of VPLS.

---

## 6. Inter-AS VPN — Topology Analysis

### Option A — Back-to-Back VRF

The ASBR maintains one VRF per customer with a dedicated sub-interface to the peer ASBR.

Resource cost on ASBR:

$$R_A = V \times (r_{vrf} + r_{interface} + r_{protocol})$$

Where $V$ = number of VRFs, $r_{vrf}$ = per-VRF overhead, $r_{interface}$ = per-interface overhead, $r_{protocol}$ = PE-CE routing protocol state.

For 500 VRFs on the ASBR, this requires 500 sub-interfaces and 500 PE-CE sessions. This is the scaling bottleneck.

### Option B — VPNv4 eBGP

ASBR holds the full VPNv4 table from both ASes:

$$|ASBR_{table}| = |VPNv4_{AS1}| + |VPNv4_{AS2}|$$

ASBR must perform next-hop-self and re-allocate labels. Label consumption on ASBR:

$$L_{ASBR} = |VPNv4_{ASBR}| \times l_{per\_prefix}$$

Where $l_{per\_prefix}$ is typically 1 label per VPNv4 prefix (per-prefix model) or 1 per VRF (per-VRF model).

### Option C — Multihop MP-eBGP

ASBRs only carry labeled unicast routes for PE loopbacks:

$$|ASBR_{table}| = N_{PE,AS1} + N_{PE,AS2}$$

Dramatically smaller than Option B. A network with 100 PEs per AS: ASBR holds 200 labeled unicast routes (vs potentially millions of VPNv4 routes in Option B).

### Comparison Matrix

| Metric | Option A | Option B | Option C |
|:---|:---:|:---:|:---:|
| ASBR State | $O(V)$ VRFs | $O(\sum |P_v|)$ VPNv4 | $O(N_{PE})$ loopbacks |
| Labels on ASBR | 0 (IP routing) | $O(\sum |P_v|)$ | $O(N_{PE})$ |
| Inter-AS Interfaces | $V$ sub-ifs | 1 | 1 |
| E2E LSP Required | No | No | Yes |
| VPN Route Visibility | Per-VRF | Full at ASBR | E2E via RR |

---

## 7. Label Stack MTU Impact

### The Calculation

Each MPLS label adds 4 bytes. For an L3VPN with standard dual-label stack:

$$MTU_{effective} = MTU_{link} - (4 \times D)$$

Where $D$ = label stack depth.

### Common Scenarios

| Scenario | Stack Depth ($D$) | Overhead | Effective MTU (1500) | Recommended Link MTU |
|:---|:---:|:---:|:---:|:---:|
| L3VPN (LDP transport) | 2 | 8 B | 1,492 | 1508 |
| L3VPN over TE tunnel | 3 | 12 B | 1,488 | 1512 |
| L3VPN + FRR active | 3-4 | 12-16 B | 1,484-1,488 | 1516 |
| L2VPN pseudowire | 2 | 8 B | 1,492 | 1508 |
| L2VPN PW + control word | 2 + CW | 12 B | 1,488 | 1512 |
| VPLS over TE + FRR | 4 | 16 B | 1,484 | 1516 |

### The Fragmentation Problem

If link MTU is not increased to accommodate labels, packets at 1,500 bytes will be fragmented. For labeled packets, fragmentation occurs at the ingress PE (label push point), not in the core.

For TCP traffic, PMTUD adjusts MSS:

$$MSS = MTU_{effective} - 20_{IP} - 20_{TCP} = 1492 - 40 = 1,452 \text{ bytes (L3VPN)}$$

For L2VPN, the original Ethernet frame (up to 1,514 bytes with header) plus labels must fit within the core MTU:

$$MTU_{core} \geq MTU_{CE} + 14_{Ethernet} + 4 \times D$$

If CE MTU = 1,500: core needs $\geq 1,500 + 14 + 8 = 1,522$ bytes minimum for L2VPN.

---

## 8. PE-CE Routing — OSPF Domain ID and DN Bit

### The Loop Prevention Problem

When OSPF is used as PE-CE protocol, routes are redistributed: OSPF $\rightarrow$ BGP (at originating PE) $\rightarrow$ BGP $\rightarrow$ OSPF (at remote PE). Without loop prevention, these routes could be re-redistributed back into BGP, creating a routing loop.

### DN Bit (Down Bit)

The DN bit is set in OSPF LSAs generated from BGP redistribution. When a PE receives an OSPF LSA with DN=1 on a VRF interface, it does not redistribute that route back into BGP.

Decision matrix:

| LSA Source | DN Bit | PE Action |
|:---|:---:|:---|
| Local CE (genuine OSPF) | 0 | Redistribute into BGP VPNv4 |
| Remote PE (from BGP) | 1 | Install in OSPF, do NOT re-redistribute into BGP |
| CE behind another PE (reflected) | 1 | Same as above — prevents loop |

### Domain ID

OSPF domain ID (derived from OSPF process ID by default) determines the LSA type on the remote PE:

| Domain ID Match | Remote LSA Type | OSPF Route Type |
|:---|:---:|:---|
| Same domain ID | Type 3 (Summary) | O IA (inter-area) |
| Different domain ID | Type 5 (External) | O E2 (external type 2) |
| No domain ID | Type 5 (External) | O E2 |

Same domain ID across sites preserves the OSPF metric and route type hierarchy. Different domain IDs treat remote routes as external, which affects path preference.

---

## 9. Targeted LDP for Pseudowire Signaling

### The Session Model

Unlike regular LDP (which discovers neighbors via link-local hello messages), pseudowire signaling uses targeted LDP:

- Hello messages are unicast UDP (not multicast)
- TCP session established between PE loopbacks (not link-local addresses)
- FEC is a pseudowire ID (VC ID), not an IP prefix

### Label Mapping

The targeted LDP session exchanges FEC-label bindings for pseudowires:

$$\text{FEC: PW ID (Type 128)} \rightarrow \text{Label: VC label}$$

Each PE allocates a local label for each pseudowire it serves:

$$L_{PE_1}(vc\_id=100) = 600 \quad \text{(PE1 allocates label 600 for VC 100)}$$
$$L_{PE_2}(vc\_id=100) = 750 \quad \text{(PE2 allocates label 750 for VC 100)}$$

PE1 pushes label 750 (remote label) when sending to PE2. PE2 pushes label 600 when sending to PE1.

### PW Status Signaling

Pseudowire status (up/down/standby) is signaled via LDP notification messages. Status codes:

| Status Code | Meaning |
|:---:|:---|
| 0x00000000 | PW forwarding (up) |
| 0x00000001 | PW not forwarding |
| 0x00000002 | Local AC (attachment circuit) receive fault |
| 0x00000004 | Local AC transmit fault |
| 0x00000008 | Local PSN-facing PW receive fault |
| 0x00000010 | Local PSN-facing PW transmit fault |
| 0x00000020 | PW preferential forwarding standby |

---

## 10. Control Word — Sequencing and Load Balancing

### The Problem

Without a control word, ECMP routers in the P network may inspect the first nibble after the label stack to determine if the payload is IPv4 (nibble=4) or IPv6 (nibble=6) for hash-based load balancing. For L2VPN, the payload is an Ethernet frame, and the first nibble is part of the destination MAC address — it could be anything.

If the first nibble happens to be 4 or 6, the P router may incorrectly interpret the Ethernet frame as IP and hash on random bytes, causing flow reordering.

### Control Word Format

The 4-byte control word is inserted between the PW label and the payload:

| Field | Bits | Value |
|:---|:---:|:---|
| First nibble | 4 | 0 (distinguishes from IPv4/IPv6) |
| Flags | 4 | Reserved |
| Fragment | 2 | Fragmentation bits |
| Length | 6 | Payload length (if < 64 bytes, for padding) |
| Sequence Number | 16 | Ordering (optional) |

### With vs Without Control Word

| Scenario | First Nibble After Labels | P Router Behavior |
|:---|:---:|:---|
| L3VPN (IPv4 payload) | 4 | Correctly identifies as IPv4, hashes on IP 5-tuple |
| L2VPN without CW | Variable (MAC byte) | May misidentify payload; unpredictable hashing |
| L2VPN with CW | 0 | Knows payload is not IP; uses label-based hash |

Both PEs must agree on control word usage. Mismatch (one with CW, one without) causes the pseudowire to fail.

---

## 11. BGP Best Path Selection for VPNv4

### Extended Decision Process

VPNv4 routes follow the standard BGP best path algorithm with additional considerations:

1. Weight (Cisco-specific, local)
2. Local Preference
3. Locally originated (network/aggregate)
4. AS Path length
5. Origin (IGP < EGP < Incomplete)
6. MED (Multi-Exit Discriminator)
7. eBGP over iBGP
8. Lowest IGP metric to BGP next-hop
9. Oldest route (stability)
10. Lowest Router ID
11. Shortest cluster list (RR paths)
12. Lowest neighbor address

### VPNv4 Specific Behavior

- Routes with different RDs are treated as different prefixes even if the IPv4 portion matches
- This enables multi-path: a PE can install multiple VPNv4 routes for the same IPv4 prefix from different PEs (with different RDs)
- `maximum-paths ibgp N` enables ECMP over VPNv4 routes with different RDs

### Multi-Homing Example

CE dual-homed to PE1 (RD 65000:100) and PE2 (RD 65000:200):

$$\text{PE1 advertises: } 65000:100:10.10.0.0/16 \quad L_{VPN}=500$$
$$\text{PE2 advertises: } 65000:200:10.10.0.0/16 \quad L_{VPN}=600$$

Remote PE3 receives both. With `maximum-paths ibgp 2`:
- Both routes installed in VRF
- Traffic load-balanced across PE1 and PE2
- If PE1 fails, PE3 immediately uses the PE2 path (no convergence delay for BGP withdrawal)

---

## 12. Convergence Analysis — VPN Route Withdrawal

### Failure Scenario

When a PE-CE link fails, the convergence sequence is:

1. PE detects link failure: $T_{detect}$ (BFD: ~150ms, OSPF dead: ~40s)
2. PE withdraws VPNv4 route via BGP update: $T_{bgp\_update}$ (~1-5s)
3. BGP update propagates to RR: $T_{propagation}$ (~RTT to RR)
4. RR reflects to all PEs: $T_{reflection}$ (RR processing + propagation)
5. Remote PE updates VRF and FIB: $T_{install}$ (~10-100ms)

Total convergence:

$$T_{convergence} = T_{detect} + T_{bgp\_update} + T_{propagation} + T_{reflection} + T_{install}$$

### Worked Example

BFD detection (150ms) + BGP scan (1s) + PE-to-RR RTT (20ms) + RR processing (50ms) + RR-to-PE RTT (20ms) + FIB install (50ms):

$$T_{convergence} = 150 + 1000 + 20 + 50 + 20 + 50 = 1,290 \text{ ms} \approx 1.3\text{s}$$

With BGP PIC (Prefix Independent Convergence):
- Backup path pre-computed and installed in FIB
- On failure, only $T_{detect} + T_{switch}$ needed

$$T_{PIC} = 150 + 10 = 160 \text{ ms}$$

### PIC Edge vs PIC Core

| Scenario | Without PIC | With PIC |
|:---|:---:|:---:|
| PE-CE failure | ~1-5 s | ~150-200 ms |
| PE node failure | ~5-30 s | ~200-500 ms |
| P-P link failure | FRR: ~50 ms | FRR: ~50 ms (same) |

PIC requires a pre-installed backup path. For L3VPN, this means the CE must be dual-homed to two PEs, providing two VPNv4 routes with different RDs.

---

## Prerequisites

- Solid understanding of MPLS fundamentals (labels, LSPs, LDP, LFIB) — see the mpls detail page
- BGP operation (path attributes, best path selection, communities) — see the bgp detail page
- IP routing fundamentals and VRF concept
- Understanding of Ethernet bridging for L2VPN/VPLS sections

## References

- [RFC 4364 — BGP/MPLS IP Virtual Private Networks (VPNs)](https://www.rfc-editor.org/rfc/rfc4364)
- [RFC 4659 — BGP-MPLS IP VPN Extension for IPv6 VPN](https://www.rfc-editor.org/rfc/rfc4659)
- [RFC 4761 — Virtual Private LAN Service (VPLS) Using BGP](https://www.rfc-editor.org/rfc/rfc4761)
- [RFC 4762 — Virtual Private LAN Service (VPLS) Using LDP Signaling](https://www.rfc-editor.org/rfc/rfc4762)
- [RFC 4447 — Pseudowire Setup and Maintenance Using LDP](https://www.rfc-editor.org/rfc/rfc4447)
- [RFC 4684 — Constrained Route Distribution for BGP/MPLS IP VPNs](https://www.rfc-editor.org/rfc/rfc4684)
- [RFC 4385 — Pseudowire Emulation Edge-to-Edge (PWE3) Control Word](https://www.rfc-editor.org/rfc/rfc4385)
- [RFC 3107 — Carrying Label Information in BGP-4](https://www.rfc-editor.org/rfc/rfc3107)
- [RFC 4271 — A Border Gateway Protocol 4 (BGP-4)](https://www.rfc-editor.org/rfc/rfc4271)
- [RFC 6074 — Provisioning, Auto-Discovery, and Signaling in L2VPNs](https://www.rfc-editor.org/rfc/rfc6074)
