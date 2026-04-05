# Multicast Routing — Theory and Protocol Internals

> *Multicast routing is a distributed tree-construction problem where routers must build loop-free delivery paths from sources to group members without destination-based forwarding. The Reverse Path Forwarding check — a single-source shortest-path verification — replaces the destination lookup of unicast, but introduces state proportional to the product of sources and groups. Every design decision in multicast protocol engineering trades state capacity against path optimality, convergence speed against control-plane load, and deployment simplicity against operational flexibility.*

---

## 1. Multicast Addressing and Group Management

### Class D Addressing

IPv4 multicast occupies the Class D range 224.0.0.0/4 (224.0.0.0 - 239.255.255.255), providing $2^{28} = 268,435,456$ group addresses. The high-order nibble `1110` identifies the address as multicast. There is no subnet mask in the traditional sense; a multicast address identifies a group, not a topological location.

Key reserved ranges:

| Range | Scope | Purpose |
|:---|:---|:---|
| 224.0.0.0/24 | Link-local | OSPF (224.0.0.5/6), VRRP (224.0.0.18), IGMP (224.0.0.1/22), PIM (224.0.0.13) |
| 224.0.1.0/24 | Internetwork control | NTP (224.0.1.1), Auto-RP (224.0.1.39/40) |
| 232.0.0.0/8 | Source-specific | SSM range (RFC 4607) |
| 233.0.0.0/8 | GLOP | AS-embedded addressing (RFC 3180) |
| 239.0.0.0/8 | Admin-scoped | Organization-local (RFC 2365) |

### Layer 2 to Layer 3 Mapping

The IEEE allocated the OUI 01:00:5E for IPv4 multicast MAC addresses. Only 23 of the 28 group address bits map into the MAC:

$$\text{MAC} = 01\text{:}00\text{:}5E\text{:}\underbrace{0\|b_{22} b_{21} \ldots b_{16}}_{1 \text{ byte}}\text{:}\underbrace{b_{15} \ldots b_8}_{1 \text{ byte}}\text{:}\underbrace{b_7 \ldots b_0}_{1 \text{ byte}}$$

The top bit of byte 4 is always 0, so only 23 bits are available. This creates a 32:1 address ambiguity:

$$\text{Overlap ratio} = \frac{2^{28}}{2^{23}} = 2^5 = 32$$

Thirty-two distinct multicast groups map to the same L2 address. For example, 224.1.1.1, 225.1.1.1, 226.1.1.1, ..., 239.1.1.1 all resolve to 01:00:5E:01:01:01. IGMP snooping at the switch level mitigates this by filtering on L3 group membership rather than relying solely on L2 addresses.

### IGMP Versions

**IGMPv1 (RFC 1112)** defines the basic query-report cycle. The querier sends General Queries to 224.0.0.1 (all-hosts); hosts respond with Membership Reports for each group they belong to. There is no explicit leave mechanism — group state expires via timeout (typically 3 query intervals, default 3 x 60s = 180s).

**IGMPv2 (RFC 2236)** adds three critical features:

1. **Explicit Leave** — hosts send Leave Group messages to 224.0.0.2, enabling fast group departure
2. **Group-Specific Query** — the querier can query a single group after receiving a Leave, rather than waiting for the next General Query
3. **Querier Election** — the router with the lowest IP address on the subnet becomes querier (distributed minimum algorithm)

Leave latency drops from up to 180s (IGMPv1 timeout) to:

$$T_{leave} = \text{LMQI} \times \text{LMQC} = 1\text{s} \times 2 = 2\text{s (default)}$$

Where LMQI is Last Member Query Interval and LMQC is Last Member Query Count.

**IGMPv3 (RFC 3376)** introduces source filtering, enabling two modes per group membership:

- **INCLUDE mode**: receive traffic only from a specified source list $\{S_1, S_2, \ldots, S_n\}$
- **EXCLUDE mode**: receive traffic from all sources except a specified exclusion list

Reports are sent to 224.0.0.22 (IGMPv3-capable routers), not to the group address. This eliminates report suppression — every host reports independently — increasing query-response traffic from $O(1)$ to $O(n)$ per group per query, but enabling per-host source filtering state essential for SSM.

### MLD (Multicast Listener Discovery)

MLD is the IPv6 equivalent of IGMP, carried inside ICMPv6 rather than as a separate IP protocol:

| IGMP Version | MLD Version | RFC | ICMPv6 Type |
|:---:|:---:|:---:|:---:|
| IGMPv1 | MLDv1 | RFC 2710 | 130-132 |
| IGMPv2 | MLDv1 | RFC 2710 | 130-132 |
| IGMPv3 | MLDv2 | RFC 3810 | 143 |

MLD uses link-local scope addresses (ff02::1 for all-nodes, ff02::16 for MLDv2 reports) and inherits IPv6's mandatory link-local addressing, simplifying querier election to the lowest link-local address.

---

## 2. Reverse Path Forwarding

### The RPF Check Algorithm

Reverse Path Forwarding is the fundamental loop-prevention mechanism in multicast routing. For a packet arriving from source $S$ on interface $I_{in}$:

1. Look up $S$ in the unicast routing table (or MRIB if present)
2. Determine the next-hop interface toward $S$ — this is the **RPF interface** $I_{RPF}$
3. If $I_{in} = I_{RPF}$, the RPF check **passes** — the packet arrived on the expected shortest path from the source
4. If $I_{in} \neq I_{RPF}$, the RPF check **fails** — the packet is dropped

The key insight: multicast forwarding verifies the *source* path, not the *destination* path. The router asks "did this packet arrive on the interface I would use to reach the source?" This ensures only one copy of each packet traverses each link in steady state, preventing loops without TTL-based expiration.

### RPF Interface Selection

The RPF interface is selected through a precedence chain:

1. **Static mroute** — administratively configured, highest priority
2. **MRIB (Multicast RIB)** — populated by MBGP, PIM, or MSDP
3. **URIB (Unicast RIB)** — the standard unicast routing table

When multiple equal-cost paths exist to the source:

$$|\text{ECMP paths}| > 1 \implies \text{RPF interface} = f(\text{hash}(S))$$

Most implementations select the RPF interface deterministically using the source address as a hash key, ensuring consistent RPF decisions across the network for a given source. This differs from unicast ECMP where per-flow load balancing distributes traffic across paths.

### RPF Failure Scenarios

RPF failures are the single most common cause of multicast forwarding black holes:

**Asymmetric routing**: When the path from $A$ to $B$ differs from the path from $B$ to $A$, packets from $B$ may arrive at $A$ on a non-RPF interface. Common in networks with policy routing or unequal IGP metrics.

**Route summarization**: If the unicast route to the source is summarized, the RPF interface may point to a summary next-hop that differs from the actual packet arrival interface.

**Routing convergence**: During IGP reconvergence, the unicast route to the source may temporarily point to a different interface than where multicast packets are arriving:

$$T_{RPF\_blackhole} = T_{IGP\_convergence} + T_{MRIB\_update}$$

This window can last seconds in OSPF/IS-IS networks, causing multicast packet loss during topology changes.

**VRF leaking**: In multi-VRF environments, the RPF lookup must occur in the correct VRF context. Misconfigured VRF-to-multicast mappings cause persistent RPF failures.

### RPF Check Complexity

For a router maintaining $|S|$ source entries with a routing table of $|R|$ prefixes:

- Per-packet RPF check: $O(\log |R|)$ (longest-prefix match via trie/TCAM)
- Total per-second RPF operations: $O(P \times \log |R|)$ where $P$ is packets/second
- With hardware TCAM: $O(1)$ per lookup, bounded by TCAM entry count

In software-forwarded environments, RPF checks become the bottleneck at high packet rates. Hardware platforms offload RPF to TCAM, making the check effectively constant-time but consuming TCAM entries proportional to the number of source routes.

---

## 3. PIM Sparse Mode Internals

### Protocol Overview

PIM-SM (RFC 4601) builds shared trees rooted at a Rendezvous Point (RP) and optionally switches to source-rooted shortest-path trees (SPTs) for active flows. It is "protocol independent" because it uses whatever unicast routing table is present for RPF decisions — no multicast-specific routing protocol is needed.

### RP Discovery: BSR and Auto-RP

**Bootstrap Router (BSR)** — defined in RFC 5059:

1. Candidate BSRs (C-BSR) flood Bootstrap Messages (BSMs) hop-by-hop using PIM's ALL-PIM-ROUTERS group (224.0.0.13)
2. The C-BSR with the highest priority (then highest IP) wins the election
3. Candidate RPs (C-RPs) unicast their candidacy to the elected BSR
4. The BSR compiles the RP-set and floods it in subsequent BSMs
5. All PIM routers receive the RP-set and compute a deterministic RP-to-group mapping using a hash:

$$\text{RP}(G) = \text{C-RP}[i] \text{ where } i = \text{hash}(G) \mod |C\text{-RP set}|$$

The hash function ensures all routers agree on the same RP for a given group without explicit coordination.

**Auto-RP** (Cisco proprietary):

1. Candidate RPs announce to 224.0.1.39 (cisco-rp-announce)
2. The RP Mapping Agent listens on 224.0.1.39, selects RPs, and announces mappings to 224.0.1.40 (cisco-rp-discovery)
3. Requires dense-mode flooding for the two Auto-RP groups themselves — creating a chicken-and-egg problem solved by `ip pim autorp listener` or sparse-dense mode

### Shared Trees vs Shortest-Path Trees

**Shared Tree (RPT / (*,G) state)**: Traffic flows Source -> RP -> Receivers along the RP-rooted tree. All sources for group $G$ share the same distribution tree. Path cost from source $S$ to receiver $R$:

$$C_{RPT}(S, R) = C(S, RP) + C(RP, R)$$

**Shortest-Path Tree (SPT / (S,G) state)**: Traffic flows directly Source -> Receivers along the source-rooted tree. Each (source, group) pair has its own tree:

$$C_{SPT}(S, R) = C(S, R)$$

The triangle inequality guarantees:

$$C_{SPT}(S, R) \leq C_{RPT}(S, R) = C(S, RP) + C(RP, R)$$

with equality only when the RP lies on the shortest path from $S$ to $R$.

### SPT Switchover

When the last-hop router (DR closest to the receiver) receives the first multicast packet for $(S, G)$ down the RPT, it can initiate SPT switchover:

1. The DR sends a PIM $(S,G)$ Join toward $S$ (via RPF interface for $S$)
2. The $(S,G)$ SPT is built hop-by-hop toward the source
3. Once traffic arrives on the SPT, the DR sends an $(S,G,\text{rpt})$ Prune toward the RP to remove itself from the shared tree for that source
4. The RP stops forwarding $(S,G)$ traffic down the RPT toward that receiver

The switchover threshold is configurable:

- **Threshold = 0 (default on Cisco IOS)**: Switch immediately upon first packet — every active flow gets an SPT
- **Threshold = infinity**: Never switch — all traffic stays on the RPT, minimizing state but sub-optimal paths
- **Threshold = $K$ kbps**: Switch when source rate exceeds $K$ — a bandwidth-based heuristic

### The Assert Mechanism

When two PIM routers on the same multi-access segment both forward the same multicast stream, receivers get duplicate packets. The Assert mechanism resolves this:

1. Both routers detect the duplicate (they receive multicast on an outgoing interface)
2. Both send PIM Assert messages containing their metric to the source
3. The winner is determined by:
   - Lowest AD (Administrative Distance) to the source
   - If AD ties, lowest metric to the source
   - If metric ties, highest IP address on the segment
4. The loser prunes its outgoing interface for that $(S,G)$ or $(*,G)$

Assert state is soft — it times out after 180s (3x the 60s Assert Timer) and must be refreshed.

### Register and Register-Stop

When a source begins sending to a group and the first-hop router (DR) has no downstream state, it must notify the RP:

1. The DR encapsulates the multicast data packet inside a PIM **Register** message (unicast to RP)
2. The RP receives the Register, de-encapsulates the data, and forwards it down the shared tree
3. The RP simultaneously sends an $(S,G)$ Join toward the source to build the native SPT from source to RP
4. Once native $(S,G)$ traffic arrives at the RP, it sends a **Register-Stop** to the DR
5. The DR stops encapsulating — native multicast flows from source to RP to receivers

The Register tunnel is a critical performance concern. Each Register packet carries a full data packet as payload plus a PIM header, processed in software by the RP. For a high-rate source:

$$\text{Register PPS} = \text{Source PPS}$$

This can overwhelm the RP control plane. The Register-Stop mechanism limits the tunnel duration to the SPT build time from RP to source, typically under 3 seconds.

---

## 4. PIM Dense Mode

### Flood-and-Prune

PIM-DM (RFC 3973) assumes all routers want multicast traffic and operates on an "opt-out" model:

1. **Flood**: When a source begins sending, traffic is forwarded out all interfaces except the RPF interface (flood phase)
2. **Prune**: Routers with no downstream receivers or downstream neighbors send Prune messages upstream
3. **Prune timeout**: Pruned state expires after the Prune Hold Timer (default 210s), causing the flood-prune cycle to repeat

The periodic flood-prune cycle generates bandwidth waste proportional to:

$$B_{waste} = \frac{R \times T_{flood}}{T_{prune}} \times L_{pruned}$$

Where $R$ is the source rate, $T_{flood}$ is the duration of each flood before prunes propagate, $T_{prune}$ is the prune timer, and $L_{pruned}$ is the number of pruned links. This makes PIM-DM unsuitable for networks where most subnets do not have receivers.

### Graft Mechanism

Grafting is the mechanism for a pruned router to rejoin the SPT without waiting for the prune timer to expire:

1. A new receiver joins group $G$ on a pruned interface
2. The router sends a PIM Graft message upstream (unicast, acknowledged)
3. The upstream router adds the interface back to the outgoing interface list (OIL)
4. A Graft-Ack is sent back to confirm
5. Traffic resumes immediately

Graft latency is bounded by:

$$T_{graft} = RTT_{upstream} + T_{forwarding\_restart}$$

This is typically under 100ms on modern hardware, making grafting significantly faster than waiting for prune expiry.

### State-Assert in Dense Mode

The dense-mode Assert operates identically to sparse-mode Assert but fires more frequently because the flood phase guarantees duplicate forwarding on multi-access segments. Every flood cycle triggers Assert elections on shared segments. In a network with $M$ multi-access segments and $A$ active groups:

$$\text{Assert messages per prune cycle} = O(M \times A)$$

---

## 5. PIM Bidirectional Mode

### Designated Forwarder Election

PIM BiDir (RFC 5015) eliminates source-specific state entirely by building a single bidirectional shared tree per group. The key mechanism is the **Designated Forwarder (DF)** election on each link:

1. All PIM BiDir routers on a segment participate in the DF election for each RP (or RP address — the phantom RP)
2. Each router advertises its unicast metric to the RP via DF Offer messages
3. The router with the best metric to the RP becomes the DF
4. The DF is the only router that forwards multicast traffic upstream (toward the RP) and downstream (away from the RP) on that link

The DF election uses a four-message protocol: Offer, Winner, Backoff, Pass. Convergence is bounded:

$$T_{DF} = T_{offer\_interval} \times (N_{candidates} - 1) + T_{processing}$$

### Phantom RP

BiDir PIM does not require the RP to actually exist as a running router. The RP address is a **phantom** — it serves only as a topological anchor point for RPF calculations and DF elections. Traffic addressed to a BiDir group is forwarded toward the RP address along the RPF path; the DF on each segment determines forwarding direction.

This means:

- The RP never processes data-plane traffic (no Register tunnel overhead)
- No SPT switchover exists — there is only one tree
- Source traffic is forwarded upstream toward the RP address by each DF, then downstream on the shared tree
- State is purely $(*,G)$ — no $(S,G)$ entries are ever created

State scaling for BiDir:

$$\text{State}_{BiDir} = O(G) \quad \text{vs} \quad O(S \times G) \text{ for ASM}$$

This makes BiDir ideal for many-to-many applications (e.g., video conferencing with hundreds of sources) where the $S \times G$ state explosion of ASM becomes untenable.

---

## 6. Source-Specific Multicast

### Architecture

SSM (RFC 4607) eliminates the Rendezvous Point entirely. Receivers subscribe to a specific (source, group) channel using IGMPv3 INCLUDE mode:

$$\text{Join}(S, G) \quad \text{where } G \in 232.0.0.0/8$$

The last-hop router builds an $(S,G)$ SPT directly toward the source — no shared tree, no RP, no Register tunnel, no SPT switchover.

### Protocol Mechanics

1. Host sends IGMPv3 Report: INCLUDE $(G, \{S_1\})$
2. Last-hop router creates $(S_1, G)$ state
3. PIM $(S_1, G)$ Join is sent toward $S_1$ via RPF interface
4. SPT is built hop-by-hop to the source
5. Traffic flows natively from source to receiver along SPT

### State Analysis: SSM vs ASM

For $|S|$ sources and $|G|$ groups in an ASM network:

$$\text{State}_{ASM} = O(|S| \times |G|) + O(|G|) = O(|S| \times |G|)$$

The $O(|S| \times |G|)$ term is $(S,G)$ state after SPT switchover; the $O(|G|)$ term is $(*,G)$ shared tree state.

For SSM, although state is formally also $(S,G)$, the crucial difference is operational: each receiver explicitly chooses which $(S,G)$ channels to join. In practice:

$$\text{State}_{SSM} \leq |S| \times |G|$$

But typically $\text{State}_{SSM} \ll |S| \times |G|$ because receivers subscribe to only a small subset of all possible $(S,G)$ combinations. The *maximum* state is bounded identically, but the *expected* state is proportional to actual demand:

$$E[\text{State}_{SSM}] = \sum_{r \in R} |\text{channels}(r)| \times \text{fan-out}(r)$$

### Security Properties

SSM inherently prevents:

- **Source spoofing**: Receivers specify the expected source; packets from other sources are not forwarded
- **Unauthorized sources**: No RP means no Register tunnel — a rogue source cannot inject traffic into a shared tree
- **Group address hijacking**: The $(S,G)$ channel is unambiguous; two sources using the same group address create independent channels

---

## 7. MSDP — Multicast Source Discovery Protocol

### SA Messages

MSDP (RFC 3618) enables inter-domain multicast by allowing RPs in different PIM-SM domains to discover active sources in remote domains:

1. When an RP receives a Register from a local source for $(S, G)$, it creates a **Source Active (SA)** message
2. The SA is flooded to all MSDP peers via TCP (port 639)
3. Remote RPs that have local receivers for group $G$ join the SPT toward $S$ across domain boundaries

SA message format contains: source address, group address, RP address of the originating domain.

### Peer-RPF Check

MSDP uses its own RPF check to prevent SA message loops in full-mesh or complex peering topologies:

1. Determine the MSDP peer from which the SA was received
2. Look up the originating RP's address in the BGP routing table
3. The SA is accepted only if it was received from the MSDP peer that is the next-hop toward the originating RP's AS in the BGP table

This is **peer-RPF**, not data-plane RPF. It operates on the MSDP peering graph using BGP AS-path information:

$$\text{Accept SA from peer } P \iff \text{BGP next-hop to originating RP's AS} = P$$

### Anycast RP with MSDP

Anycast RP deploys the same RP address on multiple routers (each advertising the RP's /32 via IGP). Sources Register to the topologically nearest RP. MSDP peering between the anycast RP instances synchronizes source state:

1. Source $S$ Registers to nearest anycast RP instance $RP_1$
2. $RP_1$ generates an MSDP SA for $(S, G)$ to peer $RP_2$
3. $RP_2$ now knows about source $S$ and can join the SPT if it has local receivers
4. Receivers behind $RP_2$ receive traffic via SPT from $S$ (not via $RP_1$'s shared tree)

This provides:

- **RP redundancy**: If one RP instance fails, sources and receivers failover to the next-nearest instance
- **RP load distribution**: Sources Register to the nearest RP, distributing Register processing
- **Sub-optimal path avoidance**: SPT switchover from the receiver side eliminates the need to traverse the RP

---

## 8. Multicast VPN

### MDT Architecture

Multicast VPN (MVPN) extends multicast into MPLS/VPN networks. The foundational concept is the **Multicast Distribution Tree (MDT)**:

**Default MDT**: A permanent multicast group in the provider (P) network that connects all PEs participating in a given MVPN. Each VRF on each PE joins the Default MDT group. All customer multicast traffic is initially encapsulated and sent on the Default MDT:

$$G_{default} = \text{provider multicast group (e.g., 239.1.1.1)}$$

All PE routers for a given VPN join $G_{default}$, creating a full-mesh multicast overlay. Bandwidth cost:

$$B_{default} = R_{customer} \times L_{MDT}$$

Where $L_{MDT}$ is the number of provider links in the Default MDT tree.

**Data MDT**: Created dynamically when a customer multicast stream exceeds a configured threshold. A dedicated provider multicast group is allocated for the high-bandwidth stream:

1. PE detects customer stream $(S_c, G_c)$ exceeding threshold
2. PE selects a Data MDT group $G_{data}$ from a configured pool
3. PE sends a Data MDT Join TLV on the Default MDT to notify other PEs
4. Interested PEs join $G_{data}$ in the provider network
5. Customer traffic for $(S_c, G_c)$ switches from Default MDT to Data MDT

This prevents high-bandwidth streams from consuming bandwidth to PEs that have no receivers.

### Encapsulation: mGRE

Draft Rosen MVPN uses multicast GRE (mGRE) encapsulation. The customer multicast packet is encapsulated inside a GRE header with the provider multicast group as the outer destination:

```
[Outer IP: src=PE_addr, dst=G_default] [GRE] [Customer IP: src=S_c, dst=G_c] [Payload]
```

The GRE key field carries the VPN identifier, enabling the receiving PE to associate the decapsulated packet with the correct VRF.

### MVPN Profile Types

NG-MVPN (RFC 6514) defines standardized profiles combining provider tunnel types and signaling:

| Profile | P-Tunnel Type | C-Multicast Signaling | Use Case |
|:---:|:---|:---|:---|
| 0 | Default MDT (PIM/mGRE) | PIM (Draft Rosen) | Legacy interop |
| 1 | Default MDT (mLDP) | PIM | mLDP provider, PIM customer |
| 2 | Default MDT (mLDP) | BGP C-multicast | Full BGP signaling |
| 3 | Ingress Replication (P2P) | BGP | No provider multicast |
| 4 | Selective (mLDP) | BGP | Selective P-tunnels |
| 5 | Selective (P2MP TE) | BGP | RSVP-TE P-tunnels |
| 6 | Ingress Replication (selective) | BGP | IR with selective tunnels |

### NG-MVPN BGP SAFI

RFC 6514 introduces BGP MVPN SAFI (SAFI 5) with seven route types:

| Type | Name | Purpose |
|:---:|:---|:---|
| 1 | Intra-AS I-PMSI A-D | Default MDT advertisement |
| 2 | Inter-AS I-PMSI A-D | Cross-AS Default MDT |
| 3 | S-PMSI A-D | Data MDT advertisement |
| 4 | Leaf A-D | Receiver PE interest |
| 5 | Source Active A-D | MSDP-equivalent in BGP |
| 6 | Shared Tree Join | $(*,G)$ customer join |
| 7 | Source Tree Join | $(S,G)$ customer join |

Route Distinguishers and Route Targets from the unicast L3VPN are reused, enabling MVPN to inherit the VPN membership and policy infrastructure of BGP/MPLS VPNs.

---

## 9. Multicast in VXLAN/EVPN Fabrics

### The BUM Problem

VXLAN (RFC 7348) extends Layer 2 domains over an IP underlay. Broadcast, Unknown unicast, and Multicast (BUM) traffic must be delivered to all VTEPs participating in a given VNI. Two mechanisms exist:

### Underlay Multicast

Each VNI is mapped to a multicast group in the underlay:

$$\text{VNI } N \rightarrow G_{underlay}(N)$$

VTEPs join the underlay multicast group for each locally-configured VNI via IGMP/PIM. BUM frames are encapsulated in VXLAN and sent to $G_{underlay}(N)$; the underlay multicast tree delivers them to all participating VTEPs.

Advantages:
- Optimal replication — packets are replicated at branch points, not at the source
- Underlay bandwidth scales with the tree, not the VTEP count

Disadvantages:
- Requires PIM in the underlay — operational complexity
- One multicast group per VNI — group address consumption: $|G_{underlay}| = |VNI_{active}|$
- Shared multicast groups across VNIs reduce group count but increase unnecessary replication

State scaling:

$$\text{Underlay state} = O(|VNI| \times |VTEP|)$$

Each VTEP maintains IGMP/PIM state for each VNI's multicast group.

### Ingress Replication

The ingress VTEP unicast-replicates BUM frames to every remote VTEP in the VNI's flood list:

$$\text{Copies}_{IR} = |VTEP_{VNI}| - 1$$

For a BUM frame on a VNI with $V$ VTEPs:

$$B_{IR} = R_{BUM} \times (V - 1)$$

Bandwidth consumption at the ingress VTEP uplink:

$$B_{uplink} = R_{BUM} \times (V - 1)$$

versus underlay multicast:

$$B_{uplink} = R_{BUM} \times 1$$

The ratio quantifies the ingress replication tax:

$$\text{IR overhead factor} = V - 1$$

For a fabric with 100 VTEPs per VNI, the ingress VTEP sends 99 copies of each BUM frame.

### EVPN Optimizations

BGP EVPN (RFC 7432) significantly reduces BUM traffic through control-plane learning:

1. **Type 2 routes (MAC/IP Advertisement)**: Distribute MAC-to-VTEP bindings, eliminating unknown unicast flooding
2. **Type 3 routes (Inclusive Multicast Ethernet Tag)**: Advertise per-VNI VTEP membership, building optimized flood lists
3. **ARP suppression**: The VTEP answers ARP requests locally from its EVPN-learned MAC/IP table, eliminating ARP broadcast floods

Residual BUM traffic after EVPN optimization is limited to:

- True broadcast (DHCP discover, gratuitous ARP on platforms without suppression)
- Active multicast group traffic (still requires flood or underlay multicast)
- Unknown unicast during the brief window before EVPN Type 2 routes propagate

---

## State Scaling and Convergence Analysis

### Multicast State Complexity

The fundamental state scaling relationships across PIM modes:

| Mode | State per Router | State Variables |
|:---|:---|:---|
| PIM-DM | $O(\|S\| \times \|G\|)$ | $(S,G)$ entries, pruned/forwarding per interface |
| PIM-SM (RPT only) | $O(\|G\|)$ | $(*,G)$ entries |
| PIM-SM (with SPT) | $O(\|S\| \times \|G\|) + O(\|G\|)$ | $(S,G)$ + $(*,G)$ entries |
| PIM BiDir | $O(\|G\|)$ | $(*,G)$ entries + DF state per link |
| PIM-SSM | $O(\|S\| \times \|G\|)$ worst case | $(S,G)$ entries only, but bounded by receiver demand |

### RPF Convergence Time

When a unicast topology change occurs, multicast convergence depends on:

$$T_{mcast\_converge} = T_{detect} + T_{IGP\_SPF} + T_{RIB\_update} + T_{MRIB\_update} + T_{RPF\_recheck} + T_{PIM\_join}$$

Typical values in an OSPF network:

| Component | Typical Duration |
|:---|:---|
| $T_{detect}$ (BFD) | 50-150ms |
| $T_{IGP\_SPF}$ | 50-200ms |
| $T_{RIB\_update}$ | 10-50ms |
| $T_{MRIB\_update}$ | 10-50ms |
| $T_{RPF\_recheck}$ | 0-100ms (implementation-dependent) |
| $T_{PIM\_join}$ | 0-3s (periodic Join interval jitter) |

Total worst-case: approximately 3.5 seconds, dominated by the PIM Join interval. Triggered PIM Joins on RPF change can reduce this to under 500ms.

### MRIB vs URIB Size Impact

The RPF lookup cost scales with the table used:

$$T_{RPF\_lookup} = O(\log |\text{MRIB}|) \leq O(\log |\text{URIB}|)$$

The MRIB is typically a subset of the URIB (containing only multicast-relevant routes), making RPF lookups faster when an MRIB is maintained separately. In hardware, both resolve to TCAM lookups at $O(1)$ per query.

---

## Prerequisites

- IP routing fundamentals (OSPF, BGP, longest-prefix match)
- IGMP theory (see `igmp.md`)
- VXLAN and EVPN fundamentals (see `vxlan.md`, `fabric-multicast.md`)
- MPLS and L3VPN concepts (see `mpls.md`, `mpls-vpn.md`)
- PIM basics and tree-building concepts

---

## References

- RFC 4601 — Protocol Independent Multicast - Sparse Mode (PIM-SM): Protocol Specification (Revised)
- RFC 3376 — Internet Group Management Protocol, Version 3
- RFC 4607 — Source-Specific Multicast for IP
- RFC 6514 — BGP Encodings and Procedures for Multicast in MPLS/BGP IP VPNs
- RFC 3973 — Protocol Independent Multicast - Dense Mode (PIM-DM)
- RFC 5015 — Bidirectional Protocol Independent Multicast (BIDIR-PIM)
- RFC 3618 — Multicast Source Discovery Protocol (MSDP)
- RFC 7348 — Virtual eXtensible Local Area Network (VXLAN)
- RFC 7432 — BGP MPLS-Based Ethernet VPN
- RFC 5059 — Bootstrap Router (BSR) Mechanism for Protocol Independent Multicast (PIM)
- RFC 2365 — Administratively Scoped IP Multicast
- Williamson, Beau. *Developing IP Multicast Networks, Volume I*. Cisco Press, 2000.
