# The Mathematics of L2VPN — Pseudowire Emulation and Scaling

> *L2VPN services emulate Layer 2 connectivity across a packet-switched network. Understanding the scaling properties — from VPLS full-mesh O(N^2) to H-VPLS O(N) — and the signaling trade-offs between LDP and BGP is essential for service provider network design.*

---

## 1. L2VPN Taxonomy

### Service Classification

L2VPN services are classified along two axes:

**By topology:**
- **Point-to-point (P2P):** Single pseudowire between two endpoints (VPWS)
- **Multipoint-to-multipoint (MP2MP):** Full LAN emulation with broadcast domain (VPLS, EVPN)

**By signaling:**
- **LDP-signaled:** Targeted LDP sessions carry PW FEC elements (RFC 4447, 4762)
- **BGP-signaled:** BGP NLRI carries PW/VPLS/EVPN reachability (RFC 4761, 7432)
- **Static:** Manual label assignment, no signaling protocol

### Pseudowire Emulation Architecture (RFC 3985)

The Pseudowire Emulation Edge-to-Edge (PWE3) architecture defines:

```
CE1 --- [AC] --- PE1 === [PSN Tunnel] === PE2 --- [AC] --- CE2
                  |                        |
              Encapsulation            Decapsulation
              (Native → PW)            (PW → Native)
```

- **AC (Attachment Circuit):** Customer-facing interface
- **PE (Provider Edge):** Performs L2 encapsulation/decapsulation
- **PSN Tunnel:** MPLS LSP, GRE, L2TPv3, or IP providing transport
- **Forwarder:** PE component that maps between AC and PW

### Pseudowire Encapsulation

For Ethernet over MPLS (RFC 4448):

```
+--------+--------+--------+--------+
|          Transport Label           |  4 bytes (MPLS)
+--------+--------+--------+--------+
|           PW (VC) Label            |  4 bytes (MPLS)
+--------+--------+--------+--------+
|       Control Word (optional)      |  4 bytes
+--------+--------+--------+--------+
|       Ethernet Frame (payload)     |  Variable
|  (DA + SA + Type/Len + Data + FCS) |
+--------+--------+--------+--------+
```

Overhead per packet:

$$\text{Overhead} = 4 \text{ (transport)} + 4 \text{ (PW)} + 4 \text{ (CW, if used)} = 8 \text{ or } 12 \text{ bytes}$$

With multiple MPLS transport labels (e.g., Unified MPLS):

$$\text{Overhead} = 4 \times L_{transport} + 4 \text{ (PW)} + 4 \text{ (CW)}$$

---

## 2. VPLS Full-Mesh Scaling — O(N^2)

### The Problem

In LDP-signaled VPLS (RFC 4762), every PE in a VPLS instance must establish a pseudowire to every other PE. This is a full-mesh topology.

### The Formula

For $N$ PEs in a VPLS instance, the number of pseudowires:

$$PW = \frac{N(N-1)}{2}$$

This is the binomial coefficient $\binom{N}{2}$ — the same handshake problem as iBGP full-mesh.

### Worked Examples

| PEs ($N$) | Calculation | Pseudowires |
|:---:|:---|:---:|
| 3 | $\frac{3 \times 2}{2}$ | 3 |
| 5 | $\frac{5 \times 4}{2}$ | 10 |
| 10 | $\frac{10 \times 9}{2}$ | 45 |
| 20 | $\frac{20 \times 19}{2}$ | 190 |
| 50 | $\frac{50 \times 49}{2}$ | 1,225 |
| 100 | $\frac{100 \times 99}{2}$ | 4,950 |

### Growth Rate

Expanding:

$$PW = \frac{N^2 - N}{2} = \frac{1}{2}N^2 - \frac{1}{2}N$$

The dominant term is $\frac{1}{2}N^2$, so growth is **quadratic**. Doubling PEs roughly quadruples pseudowires.

### Total Network PW Count

For a service provider with $S$ VPLS instances, each with $N_s$ PEs:

$$PW_{total} = \sum_{s=1}^{S} \frac{N_s(N_s - 1)}{2}$$

If all instances have the same size $N$:

$$PW_{total} = S \times \frac{N(N-1)}{2}$$

Example: 500 VPLS instances, average 10 PEs each:

$$PW_{total} = 500 \times \frac{10 \times 9}{2} = 500 \times 45 = 22,500 \text{ pseudowires}$$

### Per-PE State

Each PE in a VPLS instance maintains:

| State | Per PW | Total per PE |
|:---|:---:|:---:|
| LDP session | 1 | $N - 1$ |
| Label binding (local + remote) | 2 labels | $2(N-1)$ |
| MAC table entries | up to $M_{max}$ | $M_{max}$ (shared across all PWs) |
| Split-horizon group membership | 1 bit | $N - 1$ bits |

---

## 3. H-VPLS Hub-Spoke Optimization

### Architecture

H-VPLS reduces the full-mesh requirement by introducing a hierarchy:

- **Hub PEs:** Form a full-mesh core (small set of PEs)
- **Spoke MTUs:** Connect to hub PE via a single spoke pseudowire
- **Two tiers:** Core tier (full-mesh) + Access tier (hub-spoke)

### Scaling Formula

For $H$ hub PEs and $M$ spoke MTUs (total nodes $N = H + M$):

**Core pseudowires:**
$$PW_{core} = \frac{H(H-1)}{2}$$

**Spoke pseudowires:**
$$PW_{spoke} = M$$

(Each MTU has one spoke PW to its hub PE.)

**Total:**
$$PW_{H-VPLS} = \frac{H(H-1)}{2} + M$$

### Comparison: VPLS vs H-VPLS

For $N = 50$ nodes:

**Flat VPLS:**
$$PW = \frac{50 \times 49}{2} = 1,225$$

**H-VPLS with 5 hubs, 45 spokes:**
$$PW = \frac{5 \times 4}{2} + 45 = 10 + 45 = 55$$

**Reduction:** $\frac{55}{1225} = 4.5\%$ of flat VPLS. A **22x improvement**.

### Optimal Hub Count

To minimize total PWs for $N$ nodes with $H$ hubs:

$$PW(H) = \frac{H(H-1)}{2} + (N - H)$$

Take derivative and set to zero:

$$\frac{dPW}{dH} = H - \frac{1}{2} - 1 = H - \frac{3}{2} = 0$$

$$H_{optimal} = \frac{3}{2} \approx 2$$

Mathematically, 2 hubs minimizes PWs, but this ignores redundancy. In practice, use 2-4 hub PEs per region for resilience.

### Redundant Spoke PWs

With dual-homed MTUs (each MTU connects to 2 hub PEs):

$$PW_{spoke} = 2M$$

$$PW_{total} = \frac{H(H-1)}{2} + 2M$$

For 5 hubs, 45 spokes: $10 + 90 = 100$ PWs (still far less than 1,225).

---

## 4. MAC Learning in VPLS

### Qualified vs Unqualified Learning

**Unqualified learning:**
- MAC address alone determines forwarding
- One MAC table shared across all VLANs in the VPLS
- Simpler but does not support overlapping MAC addresses across VLANs

**Qualified learning:**
- MAC address + VLAN ID determines forwarding
- Separate forwarding entries per VLAN within the VPLS
- Required when VPLS carries multiple customer VLANs (Q-in-Q)

### MAC Table Scaling

For a VPLS with $C$ customer sites, each with $M_c$ MAC addresses:

$$\text{Total MACs per PE} = \sum_{c=1}^{C} M_c$$

With qualified learning and $V$ VLANs:

$$\text{Total entries per PE} = \sum_{c=1}^{C} \sum_{v=1}^{V_c} M_{c,v}$$

### MAC Learning Rate

When a VPLS instance comes up, all MACs must be learned through data-plane flooding:

$$T_{learn} = \frac{M_{total}}{R_{learn}}$$

Where $R_{learn}$ is the PE's MAC learning rate (typically 1,000-10,000 MACs/second for hardware learning).

For 50,000 MACs at 5,000 MACs/sec: $T_{learn} = 10$ seconds of flooding before all MACs are learned.

### MAC Aging and Churn

With aging timer $T_{age}$ (default 300 seconds) and MAC churn rate $R_{churn}$ (MACs changing per second):

$$\text{Steady-state unknown unicast rate} = R_{churn} \times T_{flood}$$

Where $T_{flood}$ is the time to re-learn after aging. High churn combined with aggressive aging leads to excessive flooding.

---

## 5. EVPN as VPLS Successor

### Why EVPN Replaces VPLS

| Limitation of VPLS | EVPN Solution |
|:---|:---|
| Data-plane MAC learning (flood-and-learn) | Control-plane MAC distribution via BGP |
| No active-active multihoming | All-active multihoming with ESI |
| No MAC mobility detection | MAC mobility extended community with sequence numbers |
| BUM flooding to all PEs | ARP/ND suppression reduces flooding |
| No IP route awareness | Type-5 IP prefix routes |
| Full-mesh PW required | BGP auto-discovery + route targets |

### Scaling Comparison

**VPLS with $N$ PEs:**
- Signaling state: $O(N^2)$ PW labels
- MAC learning: Data-plane (flooding)
- Convergence: Depends on MAC relearning after failure

**EVPN with $N$ PEs:**
- Signaling state: $O(N)$ BGP sessions (with RR)
- MAC learning: Control-plane (BGP Type-2 routes)
- Convergence: BGP withdrawal + fast reroute (sub-second with mass withdrawal)

---

## 6. LDP vs BGP Signaling Comparison

### LDP Signaling (RFC 4447 / RFC 4762)

**Mechanism:**
1. Targeted LDP session established between PEs (TCP 646)
2. PW FEC element exchanged: VC type + VC ID + interface parameters
3. Label mapping message carries local label for the PW
4. Both PEs exchange labels; PW is up when both labels are installed

**FEC Element Format:**
```
+--------+--------+--------+--------+
|  FEC Type (128)  |  VC Type       |
+--------+--------+--------+--------+
|  VC Info Length   |  Group ID      |
+--------+--------+--------+--------+
|           VC ID (32 bits)          |
+--------+--------+--------+--------+
|  Interface Parameters (MTU, etc.) |
+--------+--------+--------+--------+
```

**Scaling properties:**
- One LDP session per PE-PE pair per VPLS instance
- Manual neighbor configuration required (no auto-discovery)
- Sessions: $\frac{N(N-1)}{2}$ for $N$ PEs

### BGP Signaling (RFC 4761)

**Mechanism:**
1. BGP session (typically via RR) carries L2VPN NLRI
2. NLRI contains VPLS information: RD + VE ID + label block
3. Route targets control which PEs import/export VPLS routes
4. Auto-discovery: PEs discover each other via BGP, no manual neighbor config

**NLRI Format:**
```
+--------+--------+--------+--------+
|    Route Distinguisher (8 bytes)   |
+--------+--------+--------+--------+
|    VE ID (2 bytes)                 |
+--------+--------+--------+--------+
|    VE Block Offset (2 bytes)       |
+--------+--------+--------+--------+
|    VE Block Size (2 bytes)         |
+--------+--------+--------+--------+
|    Label Base (3 bytes)            |
+--------+--------+--------+--------+
```

**Label block allocation:**
- Each PE advertises a label base and block size
- Remote PEs compute labels: $L = \text{Label Base} + (VE_{remote} - \text{Block Offset})$
- Single BGP UPDATE can signal labels for all PEs in the VPLS

**Scaling properties:**
- $N - 1$ BGP sessions with RR (or $\frac{N(N-1)}{2}$ without)
- Auto-discovery eliminates manual configuration
- One BGP UPDATE per PE per VPLS (vs. $N-1$ LDP messages)

### Head-to-Head Comparison

| Aspect | LDP (RFC 4762) | BGP (RFC 4761) |
|:---|:---|:---|
| Auto-discovery | No (manual PW config) | Yes (route target filtering) |
| Sessions for N PEs | $\frac{N(N-1)}{2}$ targeted LDP | $N$ (with single RR) |
| Label allocation | Individual per PW | Label block (efficient) |
| Configuration per PE | List all neighbors | RT import/export only |
| Adding a new PE | Touch all existing PEs | Touch only the new PE |
| H-VPLS support | Yes (spoke PWs) | Yes (via RR hierarchy) |
| Interop | Widely supported | Requires BGP L2VPN AFI |

---

## 7. Pseudowire Status and OAM

### PW Status Codes (RFC 4447)

PW status is signaled via LDP notification messages:

| Code | Meaning | Bit |
|:---:|:---|:---:|
| 0x00000000 | PW forwarding (OK) | — |
| 0x00000001 | PW not forwarding | 0 |
| 0x00000002 | Local AC receive fault | 1 |
| 0x00000004 | Local AC transmit fault | 2 |
| 0x00000008 | Local PSN-facing PW receive fault | 3 |
| 0x00000010 | Local PSN-facing PW transmit fault | 4 |

Status code is a bitmask; multiple faults can be signaled simultaneously:

$$\text{Status} = \text{bit}_0 \lor \text{bit}_1 \lor \text{bit}_2 \lor \ldots$$

### PW OAM Mechanisms

**VCCV (Virtual Circuit Connectivity Verification):**
- BFD over VCCV for pseudowire liveliness detection
- Uses the PW control channel (CW with channel type)
- Two modes: Type 1 (in-band, uses PW label) and Type 3 (uses Router Alert label)

**G-ACh (Generic Associated Channel):**
- Uses the Associated Channel Header (ACH) in the control word position
- Channel type identifies the OAM protocol (BFD, LSP-Ping, OAM, etc.)
- ACH format: First nibble = 0001 (distinguishes from CW which starts with 0000)

### PW Redundancy State Machine

```
           +---> Active ---+
           |               |
  Init ----+               +---> Switchover
           |               |
           +---> Standby --+
```

States:
- **Active:** Primary PW forwarding traffic
- **Standby:** Backup PW ready, not forwarding (PW status = not forwarding)
- **Switchover trigger:** Primary PW fault (AC down, PSN failure, BFD timeout)

Switchover time depends on:
$$T_{switchover} = T_{detect} + T_{signal} + T_{reprogram}$$

Where:
- $T_{detect}$: BFD (150ms) or holdtimer (seconds)
- $T_{signal}$: LDP notification propagation (~10ms)
- $T_{reprogram}$: FIB update (~10ms hardware, ~100ms software)

---

## 8. Control Word Deep Dive

### Format Analysis

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|0 0 0 0| Flags |FRG|  Length   |     Sequence Number           |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

**Bit layout:**
- Bits 0-3: Always 0000 (identifier nibble)
- Bits 4-7: Flags (reserved, must be 0)
- Bits 8-9: Fragmentation (FRG)
  - 00: Unfragmented
  - 01: First fragment
  - 10: Last fragment
  - 11: Intermediate fragment
- Bits 10-15: Length (0 if payload >= 64 bytes; actual length if < 64 for padding)
- Bits 16-31: Sequence number (0 if sequencing disabled)

### The ECMP Problem Without Control Word

Without CW, the first nibble after the label stack is the customer Ethernet frame. ECMP hash algorithms may inspect bytes at offsets that align with IP header fields in the customer payload:

- IPv4 packets: first nibble = 0x4
- IPv6 packets: first nibble = 0x6
- Some hardware uses the first nibble to identify the payload as IP, then hashes on src/dst IP
- For non-IP L2 traffic (ARP, STP, etc.), the hash may be unpredictable

**With CW:** First nibble = 0x0, which no hardware interprets as IP. ECMP falls back to label-based hashing, which is deterministic for the PW.

### When Control Word Is Mandatory

1. **FAT PW (Flow-Aware Transport PW):** Uses flow label above CW for entropy
2. **Interworking:** Ethernet-VLAN or IP-mode interworking requires CW
3. **Frame ordering:** When strict ordering is required (voice, video)
4. **MPLS-TP:** CW is mandatory (no PHP, no ECMP)

---

## 9. Interworking Modes

### Ethernet Mode (Raw)

- Transports the entire Ethernet frame (DA + SA + EtherType + payload + FCS)
- FCS is typically stripped and regenerated
- VLAN tags included as-is
- Both ends must use the same encapsulation

### VLAN Mode (Tagged)

- Transports Ethernet frame with specific VLAN treatment
- Service-delimiting VLAN tag may be stripped or preserved
- Used with dot1q sub-interfaces

### Ethernet-VLAN Interworking

- One end: Ethernet mode (port-based, untagged)
- Other end: VLAN mode (VLAN-based, tagged)
- PE at the VLAN end strips/adds the service-delimiting tag

### IP Mode Interworking

- Strips all L2 headers; transports only the IP packet
- Reconstructs L2 header at egress based on local interface configuration
- Used when L2 encapsulations are incompatible
- ARP/ND must be proxied or statically configured

**Limitations of IP mode:**
- Only IP traffic is transported (non-IP L2 protocols dropped)
- L2 header transparency lost (source/destination MAC rewritten)
- VLAN information lost
- Bridge protocol frames (STP, LACP) cannot be transported

---

## References

- [RFC 3985 — PWE3 Architecture](https://www.rfc-editor.org/rfc/rfc3985)
- [RFC 4447 — Pseudowire Setup and Maintenance Using LDP](https://www.rfc-editor.org/rfc/rfc4447)
- [RFC 4448 — Encapsulation Methods for Ethernet over MPLS](https://www.rfc-editor.org/rfc/rfc4448)
- [RFC 4762 — VPLS Using LDP Signaling](https://www.rfc-editor.org/rfc/rfc4762)
- [RFC 4761 — VPLS Using BGP](https://www.rfc-editor.org/rfc/rfc4761)
- [RFC 6718 — Pseudowire Redundancy](https://www.rfc-editor.org/rfc/rfc6718)
- [RFC 6310 — Pseudowire OAM](https://www.rfc-editor.org/rfc/rfc6310)
- [RFC 5921 — MPLS-TP Framework](https://www.rfc-editor.org/rfc/rfc5921)
- [RFC 7432 — BGP MPLS-Based Ethernet VPN](https://www.rfc-editor.org/rfc/rfc7432)
- [RFC 4385 — Pseudowire Emulation Edge-to-Edge Control Word](https://www.rfc-editor.org/rfc/rfc4385)
