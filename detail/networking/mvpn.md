# Multicast VPN — Theory, BGP Signaling, and Convergence Analysis

> *Multicast in VPN environments presents a fundamental scaling problem: customer multicast state multiplied by VPN count overwhelms provider infrastructure. mVPN solves this by abstracting customer multicast into provider-level tunnels, evolving from GRE-based Draft Rosen to fully BGP-signaled NG-mVPN. The design space spans 15 profiles trading tunnel technology, state overhead, bandwidth efficiency, and operational complexity.*

---

## 1. The Multicast-in-VPN Scaling Problem

### State Explosion Without mVPN

Consider a provider with $V$ VPNs, $P$ PE routers, and each VPN having $S$ sources and $G$ groups. Without mVPN, there are three naive approaches:

**Approach 1: Per-VRF Provider PIM State**

Each customer (S,G) creates provider-level multicast state:

$$\text{Provider PIM state} = V \times S \times G \times \bar{H}$$

where $\bar{H}$ is the average hop count of the provider multicast tree. For $V = 1000$ VPNs, $S = 10$ sources/VPN, $G = 50$ groups/VPN, $\bar{H} = 5$:

$$\text{State} = 1000 \times 10 \times 50 \times 5 = 2,500,000 \text{ entries}$$

This is untenable on P-routers that must carry state for all VPNs.

**Approach 2: Ingress Replication Without Aggregation**

The source PE unicasts a copy to each receiver PE:

$$\text{Bandwidth at source PE} = R_{VPN} \times B_{stream}$$

where $R_{VPN}$ is the number of receiver PEs and $B_{stream}$ is the stream bandwidth. For a 10 Mbps IPTV stream with 50 receiver PEs:

$$\text{Bandwidth} = 50 \times 10 = 500 \text{ Mbps per stream}$$

Replication at the source is $O(P)$ per stream rather than $O(1)$ with a multicast tree.

**Approach 3: Static GRE Tunnels**

Full mesh of GRE tunnels between PEs per VPN:

$$\text{Tunnels} = V \times \binom{P}{2} = V \times \frac{P(P-1)}{2}$$

For 1000 VPNs across 20 PEs:

$$\text{Tunnels} = 1000 \times \frac{20 \times 19}{2} = 190,000$$

Unmanageable configuration and state overhead.

### mVPN Aggregation

mVPN reduces provider state by aggregating all customer multicast within a VPN onto a single provider tunnel (the default MDT). The provider multicast state becomes:

$$\text{Provider PIM state} = V \times \bar{H}$$

One tree per VPN regardless of how many customer sources and groups exist. For 1000 VPNs with average 5-hop trees:

$$\text{State} = 1000 \times 5 = 5,000 \text{ entries}$$

A 500x reduction compared to per-VRF provider state.

Data MDTs add selective tunnels for high-bandwidth streams, but the total is bounded by the number of high-rate sources, not the total customer multicast state.

---

## 2. Draft Rosen MDT Architecture

### Default MDT Mechanics

The default MDT is a provider-level multicast group that all PEs in a given VPN join. It acts as a virtual broadcast domain for VRF multicast traffic.

**Tunnel construction:**

1. Each PE is configured with a default MDT group per VPN (e.g., 239.1.1.1 for VPN-A)
2. The PE joins the group in the global table using provider PIM
3. Provider PIM-SM (or PIM-SSM) builds a shared or source tree for the MDT group
4. All PEs in the VPN become leaves of this tree
5. Customer multicast traffic is GRE-encapsulated and sent to the MDT group address
6. All PEs receive all VRF multicast traffic for that VPN (flood-and-filter)

**GRE encapsulation format:**

```
Byte offset:
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-------+-------+-------+-------+-------+-------+-------+-------+
| Outer IPv4 Header (20 bytes)                                    |
| Src = PE Loopback, Dst = MDT Group (e.g., 239.1.1.1)          |
+-------+-------+-------+-------+-------+-------+-------+-------+
| GRE Header (4 bytes minimum)                                    |
| Flags = 0x0000, Protocol = 0x0800 (IPv4)                       |
+-------+-------+-------+-------+-------+-------+-------+-------+
| Inner IPv4 Header                                               |
| Src = Customer Source, Dst = Customer Group                     |
+-------+-------+-------+-------+-------+-------+-------+-------+
| Customer Multicast Payload                                      |
+-------+-------+-------+-------+-------+-------+-------+-------+

Total overhead: 24 bytes (outer IP 20B + GRE 4B)
With GRE key (optional): 28 bytes (GRE 8B with key field)
```

### Data MDT Mechanics

The default MDT carries all VRF multicast, including high-bandwidth streams, to all PEs. This wastes bandwidth on PEs without receivers for those streams. Data MDTs solve this.

**Data MDT trigger sequence:**

1. Source PE detects a customer (S,G) stream exceeding the configured threshold
2. Source PE selects an unused group from the data MDT range
3. Source PE sends an MDT Join TLV on the default MDT (announcing the data MDT group and the customer (S,G) it carries)
4. Receiver PEs with active receivers for that (S,G) join the data MDT group
5. Source PE switches the high-rate stream from the default MDT to the data MDT
6. When the stream stops or drops below threshold, the data MDT is torn down

**Bandwidth savings analysis:**

With $P$ PEs in a VPN, $R$ receiver PEs for a specific high-rate stream, and $B$ as the stream bandwidth:

$$\text{Default MDT bandwidth waste} = (P - R) \times B \times \bar{L}$$

where $\bar{L}$ is the average number of provider links carrying the unnecessary traffic. Data MDT reduces this to near zero, as only $R$ PEs receive the stream.

For a 20 Mbps IPTV stream, 50 PEs in the VPN, 5 with receivers:

$$\text{Waste (default MDT)} = (50 - 5) \times 20 = 900 \text{ Mbps aggregate}$$
$$\text{Waste (data MDT)} = 0 \text{ Mbps (only 5 PEs receive)}$$

### Draft Rosen Limitations

| Limitation | Impact |
|:---|:---|
| GRE overhead (24-28 bytes) | Reduces effective MTU, fragmentation risk |
| Provider PIM state per VPN | P-routers carry $O(V)$ multicast state |
| MDT group address consumption | Each VPN needs 1 default + N data groups |
| Flood-and-filter on default MDT | All PEs process all VRF multicast traffic |
| No BGP signaling for tunnel binding | Tunnel discovery is PIM-based, not BGP |
| Limited inter-AS support | Provider PIM must span AS boundaries |

---

## 3. NG-mVPN BGP Signaling

### NLRI Format (RFC 6514)

All mVPN routes are carried in BGP with AFI 1 (IPv4) or AFI 2 (IPv6), SAFI 5. The NLRI has a common header:

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-------+-------+-------+-------+-------+-------+-------+-------+
| Route Type    | Length         | Route Type Specific NLRI       |
+-------+-------+-------+-------+-------+-------+-------+-------+
```

### Type 1 — Intra-AS I-PMSI Auto-Discovery

**NLRI structure:**

```
+-------+-------+-------+-------+-------+-------+-------+-------+
| Route Distinguisher (8 bytes)                                   |
+-------+-------+-------+-------+-------+-------+-------+-------+
| Originating Router's IP (4 bytes for IPv4)                     |
+-------+-------+-------+-------+-------+-------+-------+-------+
```

**Purpose:** Each PE advertises its presence in a given VPN and binds it to a provider tunnel via the PMSI Tunnel Attribute (PTA).

**PMSI Tunnel Attribute (PTA):**

```
+-------+-------+-------+-------+
| Flags | Tunnel Type | MPLS Label |
+-------+-------+-------+-------+
| Tunnel Identifier (variable)   |
+-------+-------+-------+-------+

Tunnel Types:
  0  - No tunnel (control plane only)
  1  - RSVP-TE P2MP LSP
  2  - mLDP P2MP LSP
  3  - PIM-SSM tree
  4  - PIM-SM tree
  5  - PIM-BIDIR tree
  6  - Ingress Replication
  7  - mLDP MP2MP LSP
```

The PTA is the key innovation of NG-mVPN: it binds a BGP route to a specific provider tunnel technology, decoupling the multicast signaling (BGP) from the tunnel data plane.

**BGP communities attached to Type 1:**

- Route Target (extended community): identifies the VPN
- PE Distinguisher Labels (optional): for inter-AS scenarios

### Type 2 — Inter-AS I-PMSI Auto-Discovery

**NLRI structure:**

```
+-------+-------+-------+-------+-------+-------+-------+-------+
| Route Distinguisher (8 bytes)                                   |
+-------+-------+-------+-------+-------+-------+-------+-------+
| Source AS (4 bytes)                                             |
+-------+-------+-------+-------+-------+-------+-------+-------+
```

**Purpose:** Extends I-PMSI discovery across AS boundaries. Each ASBR originates a Type 2 route to signal its AS's participation in the mVPN. This enables segmented inter-AS tunnels where each AS builds its own I-PMSI tunnel, and the ASBR stitches them together.

### Type 3 — S-PMSI Auto-Discovery

**NLRI structure:**

```
+-------+-------+-------+-------+-------+-------+-------+-------+
| Route Distinguisher (8 bytes)                                   |
+-------+-------+-------+-------+-------+-------+-------+-------+
| Multicast Source (4 bytes) — C-Source (or 0 for *,G)           |
+-------+-------+-------+-------+-------+-------+-------+-------+
| Multicast Group (4 bytes) — C-Group                            |
+-------+-------+-------+-------+-------+-------+-------+-------+
| Originating Router's IP (4 bytes)                              |
+-------+-------+-------+-------+-------+-------+-------+-------+
```

**Purpose:** The source PE advertises a selective tunnel for a specific (C-S, C-G) or (*, C-G). The PTA contains the selective tunnel binding. This is the NG-mVPN equivalent of Draft Rosen's data MDT.

**Behavior:**
- Source PE originates Type 3 when a high-bandwidth stream is detected
- PTA specifies the selective tunnel (e.g., mLDP P2MP with opaque value, or PIM-SSM with group address, or IR)
- Receiver PEs interested in this (C-S, C-G) respond with Type 4 (Leaf A-D) routes
- Source PE switches the stream from the I-PMSI (default) to the S-PMSI (selective) tunnel

### Type 4 — Leaf Auto-Discovery

**NLRI structure:**

```
+-------+-------+-------+-------+-------+-------+-------+-------+
| Route Key (copied from Type 3 NLRI that triggered this)        |
| [RD + C-Source + C-Group + Originator PE]                      |
+-------+-------+-------+-------+-------+-------+-------+-------+
| Originating Router's IP (4 bytes) — the leaf PE                |
+-------+-------+-------+-------+-------+-------+-------+-------+
```

**Purpose:** A receiver PE signals its interest in joining a selective tunnel advertised by a Type 3 route. Required for tunnel technologies where the leaf must explicitly join (IR, mLDP P2MP, RSVP-TE P2MP). Not needed for PIM-based tunnels where PIM join handles leaf attachment.

### Type 5 — Source Active Auto-Discovery

**NLRI structure:**

```
+-------+-------+-------+-------+-------+-------+-------+-------+
| Route Distinguisher (8 bytes)                                   |
+-------+-------+-------+-------+-------+-------+-------+-------+
| Multicast Source (4 bytes) — C-Source                          |
+-------+-------+-------+-------+-------+-------+-------+-------+
| Multicast Group (4 bytes) — C-Group                            |
+-------+-------+-------+-------+-------+-------+-------+-------+
```

**Purpose:** Replaces MSDP Source-Active messages. When a customer source starts sending multicast, the PE connected to that source originates a Type 5 route. All PEs in the VPN receive it via BGP and know that source S is actively sending to group G.

**Advantages over MSDP:**
- Uses BGP (already deployed for VPNv4) instead of a separate MSDP mesh
- Benefits from BGP route reflection (no separate MSDP mesh topology)
- Subject to BGP policy (route-maps, communities) for filtering
- Consistent operational model with other mVPN signaling

### Type 6 — Shared Tree Join (C-multicast)

**NLRI structure:**

```
+-------+-------+-------+-------+-------+-------+-------+-------+
| Route Distinguisher (8 bytes)                                   |
+-------+-------+-------+-------+-------+-------+-------+-------+
| Multicast Source (4 bytes) — 0.0.0.0 for (*,G)                |
+-------+-------+-------+-------+-------+-------+-------+-------+
| Multicast Group (4 bytes) — C-Group                            |
+-------+-------+-------+-------+-------+-------+-------+-------+
| RP Address (4 bytes)                                           |
+-------+-------+-------+-------+-------+-------+-------+-------+
```

**Purpose:** Signals a (*,G) join toward the customer RP. When a receiver PE has a host join a group via IGMP and needs to join the shared tree, it originates a Type 6 route. The PE connected to the RP (or the RP itself if it is a CE) processes this and builds the (*,G) state.

**BGP-to-PIM mapping:** The Type 6 route is received by the PE closest to the RP (determined by the RT and RD). That PE translates it into a PIM (*,G) join in the VRF toward the customer RP.

### Type 7 — Source Tree Join (C-multicast)

**NLRI structure:**

```
+-------+-------+-------+-------+-------+-------+-------+-------+
| Route Distinguisher (8 bytes)                                   |
+-------+-------+-------+-------+-------+-------+-------+-------+
| Multicast Source (4 bytes) — C-Source                          |
+-------+-------+-------+-------+-------+-------+-------+-------+
| Multicast Group (4 bytes) — C-Group                            |
+-------+-------+-------+-------+-------+-------+-------+-------+
```

**Purpose:** Signals a (S,G) join toward the customer source (SPT join). When a receiver PE performs SPT switchover, it originates a Type 7 route. The PE connected to the source processes this and builds (S,G) state in the VRF.

**C-multicast routing:** Type 6 and Type 7 routes carry RT extended communities that identify the VPN. The BGP next-hop processing must map the route to the correct upstream PE using the RT and the customer source/RP address. This mapping is called "upstream PE selection" and relies on VPNv4 route information.

---

## 4. Profile Comparison Matrix

### Tunnel Technology Properties

| Property | PIM-SM/SSM | mLDP P2MP | mLDP MP2MP | RSVP-TE P2MP | Ingress Replication |
|:---|:---|:---|:---|:---|:---|
| Signaling protocol | PIM | LDP (mLDP ext.) | LDP (mLDP ext.) | RSVP-TE | BGP (Type 1/3/4) |
| Tree type | Source or shared | Source (P2MP) | Bidirectional | Source (P2MP) | Unicast copies |
| Provider state/tree | $O(V \times H)$ | $O(V \times H)$ | $O(V \times H)$ | $O(V \times H)$ | $O(P^2)$ per VPN |
| Bandwidth efficiency | High (tree) | High (tree) | High (bidir) | High (tree) | Low ($P \times B$) |
| Requires P-multicast | Yes | No (label-switched) | No (label-switched) | No (TE LSP) | No |
| Leaf-initiated join | No (PIM join) | Yes (Type 4) | Yes (MP2MP join) | Yes (Type 4) | Yes (Type 4) |
| TE capability | No | No | No | Yes (bandwidth, constraints) | N/A |
| Inter-AS support | Hard (PIM across AS) | Moderate (mLDP across AS) | Moderate | Hard (TE across AS) | Easy (unicast) |
| FRR support | PIM FRR (limited) | mLDP FRR | mLDP FRR | RSVP FRR (mature) | Not applicable |

### Profile Selection Decision Tree

```
Start
  |
  |-- Is provider multicast infrastructure available?
  |     |
  |     |-- Yes: Is MPLS deployed?
  |     |     |
  |     |     |-- Yes: Profile 1 (mLDP P2MP) or Profile 11 (RSVP-TE P2MP)
  |     |     |       Profile 11 if TE/bandwidth reservation needed
  |     |     |       Profile 1 for simpler mLDP-based transport
  |     |     |
  |     |     |-- No: Profile 0 (Draft Rosen, PIM/GRE)
  |     |           Simple, no MPLS or BGP mVPN needed
  |     |
  |     |-- No: Is MPLS deployed?
  |           |
  |           |-- Yes: Profile 12 (IR default + mLDP data)
  |           |       IR for low-rate default, mLDP for high-rate data
  |           |
  |           |-- No: Profile 7 (pure IR) or Profile 14 (partitioned IR)
  |                   Profile 14 if > 15 PEs or EVPN integration needed
  |                   Profile 7 for small deployments
```

---

## 5. PIM-SSM vs Ingress Replication for mVPN

### PIM-SSM as Provider Tunnel

When PIM-SSM is used for the provider tunnel (profiles 3, 6):

- Each PE acts as a PIM-SSM source for its MDT group
- Remote PEs join (PE-loopback, MDT-group) via PIM (S,G) join
- Provider P-routers carry one (S,G) entry per PE per VPN in the worst case

**Provider state with PIM-SSM:**

$$\text{State per P-router} = V \times P_{active} \times f_{affected}$$

where $P_{active}$ is the number of PEs actively sending in that VPN and $f_{affected}$ is the fraction of P-routers on the multicast tree. In a well-designed core:

$$f_{affected} \approx \frac{\bar{H}}{P_{total\_routers}}$$

### Ingress Replication as Provider Tunnel

With IR (profiles 7, 14), the source PE creates $R$ unicast copies, one per receiver PE:

**Bandwidth at source PE:**

$$B_{source} = R \times B_{stream}$$

**Total provider bandwidth (default MDT, non-partitioned):**

For $P$ PEs in a VPN with $S$ active source PEs and average stream bandwidth $B$:

$$B_{total\_IR} = S \times P \times B$$

**Total provider bandwidth (partitioned IR, Profile 14):**

$$B_{total\_partitioned} = S \times R \times B$$

where $R \leq P$ is the number of PEs with active receivers.

### Crossover Analysis

IR is more bandwidth-efficient than PIM when the number of receiver PEs is small relative to the total PEs. The crossover point depends on the topology.

For a simple model where PIM delivers a single copy per link on the shortest path tree:

$$\text{PIM bandwidth} \approx B \times \bar{H} \times f_{branching}$$

$$\text{IR bandwidth} = R \times B$$

IR wins when:

$$R < \bar{H} \times f_{branching}$$

For typical provider backbones ($\bar{H} = 4$, $f_{branching} = 1.5$):

$$R < 6$$

When fewer than 6 PEs have receivers, IR uses less total bandwidth than a PIM tree.

---

## 6. Inter-AS Multicast Challenges

### RPF Across AS Boundaries

The fundamental challenge in inter-AS mVPN is the RPF check. Within a single AS, the RPF check for a customer source resolves through the VPNv4 route, which points to the source PE's loopback. This loopback is reachable via the IGP, so the RPF interface is well-defined.

Across AS boundaries:
- The source PE's loopback is in a remote AS
- The local IGP does not know the route to the remote PE
- BGP provides the route, but the RPF check must resolve through the ASBR

**RPF resolution approaches:**

| Approach | Mechanism | Limitation |
|:---|:---|:---|
| Static mroute | `ip mroute` pointing RPF to ASBR | Manual, does not scale |
| BGP next-hop as RPF | Use BGP next-hop for RPF resolution | Requires MBGP or special RPF config |
| Inter-AS MDT | Extend PIM/mLDP across AS boundary | Requires PIM/mLDP peering at ASBR |
| Segmented tunnel | Each AS builds own tunnel, ASBR stitches | Complex but scalable |

### Option B Tunnel Stitching

In inter-AS Option B, the ASBR terminates the I-PMSI tunnel from one AS and originates a new I-PMSI tunnel in the other AS:

```
AS 65000                    ASBR                     AS 65001
  PE1 ---[I-PMSI-1]---> ASBR-L --eBGP-- ASBR-R ---[I-PMSI-2]---> PE2
         (tunnel in AS1)              (re-originate)   (tunnel in AS2)

ASBR processing:
  1. Receive customer multicast on I-PMSI-1 (decapsulate)
  2. Apply VRF import policy (RT matching)
  3. Re-encapsulate into I-PMSI-2 for the remote AS
  4. Signal via Type 2 Inter-AS I-PMSI AD route
```

The ASBR must carry per-VPN state and perform encap/decap, making it a potential bottleneck. The number of VPNs requiring inter-AS multicast determines the ASBR load.

### Option C End-to-End Tunnels

Option C avoids ASBR stitching by establishing end-to-end tunnels:

- Multihop eBGP between RRs carries mVPN routes with next-hop unchanged
- The provider tunnel (mLDP or PIM) must span both ASes
- Requires inter-AS LDP (for mLDP) or inter-AS PIM (for PIM-based profiles)
- BGP next-hop resolution uses labeled BGP (VPNv4 with labels) for MPLS reachability

**State comparison:**

$$\text{Option B ASBR state} = V_{inter-AS} \times (S_{tunnel} + S_{mroute})$$

$$\text{Option C ASBR state} = S_{transit\_labels} \text{ (minimal, transit only)}$$

Option C pushes state to the PEs (end-to-end tunnel) while Option B concentrates state at the ASBR.

---

## 7. mVPN Convergence Analysis

### Failure Scenarios

mVPN convergence depends on the failure type and the tunnel technology:

**Scenario 1: Provider link failure (PE-P or P-P link)**

$$T_{converge} = T_{detect} + T_{IGP} + T_{tunnel\_repair} + T_{mroute\_update}$$

| Component | PIM tunnel | mLDP tunnel | IR | RSVP-TE tunnel |
|:---|:---|:---|:---|:---|
| $T_{detect}$ | BFD: 50-150ms | BFD: 50-150ms | BFD: 50-150ms | BFD: 50-150ms |
| $T_{IGP}$ | 50-200ms | 50-200ms | 50-200ms | 50-200ms |
| $T_{tunnel\_repair}$ | PIM reconvergence: 1-3s | mLDP MoFRR: <50ms | Reroute via IGP: <100ms | FRR facility: <50ms |
| $T_{mroute\_update}$ | RPF recheck: 100-500ms | Immediate (label swap) | N/A (unicast) | Immediate (label swap) |
| **Total** | **1.2-3.9s** | **150-500ms** | **200-500ms** | **150-500ms** |

mLDP and RSVP-TE benefit from MPLS FRR mechanisms that protect the label-switched path. PIM convergence is slower because PIM must reconverge the multicast tree using RPF recalculation and join/prune signaling.

**Scenario 2: PE failure (source PE goes down)**

$$T_{converge} = T_{detect} + T_{BGP\_withdraw} + T_{PE\_selection} + T_{new\_tunnel\_setup}$$

| Component | Typical Duration |
|:---|:---|
| $T_{detect}$ (BFD or BGP holdtime) | 50ms - 90s |
| $T_{BGP\_withdraw}$ (mVPN routes withdrawn) | 0-3s (depends on BGP timers) |
| $T_{PE\_selection}$ (if redundant source PEs) | Immediate (if pre-computed) |
| $T_{new\_tunnel\_setup}$ | 1-5s (new I-PMSI/S-PMSI binding) |

PE failure is the worst case because all mVPN routes from that PE are withdrawn, requiring receiver PEs to find an alternative source PE (if multipath exists) or lose the stream until the PE recovers.

**Scenario 3: Customer source failover (CE-PE link failure)**

$$T_{converge} = T_{detect} + T_{PIM\_prune} + T_{Type5\_withdraw} + T_{new\_source\_register}$$

This is a VRF-level event. The PE detects the CE link failure, prunes the (S,G) tree in the VRF, and withdraws the Type 5 Source Active route. If a redundant source exists at another PE, that PE's Type 5 route triggers receiver PEs to build new (S,G) state.

### Convergence Optimization Techniques

| Technique | Applicable To | Mechanism |
|:---|:---|:---|
| MoFRR (Multicast only FRR) | mLDP profiles | Dual join on primary and backup path; instant switchover on failure |
| PIM FRR | PIM-based profiles | Pre-computed backup RPF interface; requires IGP convergence first |
| BGP PIC (Prefix Independent Convergence) | All NG-mVPN profiles | Pre-installs backup next-hop; switchover without full BGP convergence |
| BFD | All profiles | Sub-second failure detection on provider links |
| mLDP Make-Before-Break | mLDP profiles | New tree is built before old tree is torn down |

---

## 8. EVPN and mVPN Integration

### L2 vs L3 Multicast in EVPN Fabrics

EVPN (RFC 7432) and mVPN (RFC 6513) address different layers of multicast:

**EVPN handles L2 multicast:**
- IGMP/MLD snooping synchronization across VTEPs (EVPN Type 6/7/8)
- BUM traffic forwarding (ingress replication or multicast underlay)
- ARP suppression (reduces broadcast)
- Optimized flooding based on IGMP membership

**mVPN handles L3 multicast:**
- Inter-subnet multicast routing across VRFs
- Customer PIM signaling between PEs
- Source discovery (Type 5, replacing MSDP)
- Provider tunnel selection and binding

### Combined Architecture

In a VXLAN/EVPN fabric with L3 multicast:

```
Layer Stack:
  Application (multicast stream)
       |
  Customer PIM (within VRF, signaled via mVPN Type 6/7)
       |
  mVPN provider tunnel (IR, mLDP, PIM — carries L3 mcast across fabric)
       |
  VXLAN encapsulation (if VTEP-to-VTEP transport)
       |
  Underlay IP (spine-leaf or provider core)
```

**Profile 14 synergy with EVPN:**

Profile 14 (partitioned IR) aligns naturally with EVPN's ingress replication model:
- Both use unicast tunnels (no provider multicast infrastructure)
- EVPN IR for L2 BUM and mVPN IR for L3 multicast share the same VTEP peer list
- Type 4 Leaf A-D routes enable selective replication (only PEs with receivers get traffic)
- The partitioned model ensures the default I-PMSI only includes PEs with active VRF membership

---

## 9. Summary of Key Relationships

| Formula | Description |
|:---|:---|
| $\text{State}_{naive} = V \times S \times G \times \bar{H}$ | Provider state without mVPN aggregation |
| $\text{State}_{mVPN} = V \times \bar{H}$ | Provider state with mVPN (one tree per VPN) |
| $B_{IR} = R \times B_{stream}$ | IR bandwidth at source PE |
| $B_{PIM} \approx B \times \bar{H} \times f_{branching}$ | PIM tree bandwidth (approximate) |
| $\text{IR wins when } R < \bar{H} \times f_{branching}$ | Crossover: IR vs PIM efficiency |
| $T_{converge} = T_{detect} + T_{IGP} + T_{tunnel} + T_{mroute}$ | mVPN convergence time |
| $\text{Tunnels}_{full-mesh} = V \times P(P-1)/2$ | Static GRE tunnel count (why mVPN exists) |

## Prerequisites

- multicast-routing (PIM modes, IGMP, RPF), bgp (address families, extended communities, route targets), mpls-vpn (L3VPN architecture, RD/RT), vxlan (VXLAN encapsulation and EVPN integration), mpls (label switching, LDP/mLDP fundamentals)

---

*Multicast VPN transforms an O(sources x groups x VPNs) scaling problem into O(VPNs) by aggregating customer multicast into provider-level tunnels. Draft Rosen proved the concept with GRE and PIM; NG-mVPN refined it with BGP signaling, decoupling tunnel technology from multicast discovery. The 15 profiles represent a design spectrum: PIM/GRE for operational simplicity, mLDP for MPLS-native label-switched transport, RSVP-TE for bandwidth-guaranteed paths, and ingress replication for environments where provider multicast is unavailable or undesirable. Profile 14's partitioned IR has emerged as the dominant choice for modern EVPN fabrics, aligning multicast delivery with the unicast tunnel model that EVPN already provides.*
