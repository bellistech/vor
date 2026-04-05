# Fabric Multicast — Multicast Architecture in Data Center Fabrics

> *Multicast in a data center fabric is fundamentally a tree-building problem constrained by RPF, bounded by state capacity, and shaped by the tension between optimal paths (SPT) and minimal state (RPT). Understanding PIM variants, IGMP mechanics, RP placement, and VXLAN BUM replication requires grasping both the protocol machinery and the scaling math that drives architectural decisions.*

---

## 1. Multicast Fundamentals — Why Unicast Fails at One-to-Many

### The Bandwidth Problem

Consider a source sending a 10 Mbps video stream to $N$ receivers. With unicast replication:

$$B_{unicast} = N \times R$$

Where $R$ is the stream rate. For 500 receivers at 10 Mbps:

$$B_{unicast} = 500 \times 10 = 5,000 \text{ Mbps} = 5 \text{ Gbps}$$

With multicast, the source sends exactly one copy. Replication happens at branch points in the distribution tree:

$$B_{source} = R = 10 \text{ Mbps}$$

The total network bandwidth depends on the tree topology, but the source always sends exactly $R$ regardless of receiver count. The savings ratio:

$$\text{Savings} = \frac{N \times R - B_{tree}}{N \times R}$$

For a well-designed tree with $L$ links carrying the stream:

$$B_{tree} = L \times R$$

In a spine-leaf fabric with 2 spines and 20 leaves (each with receivers), $L = 2 + 20 = 22$ links carry the stream:

$$\text{Savings} = \frac{500 \times 10 - 22 \times 10}{500 \times 10} = \frac{5000 - 220}{5000} = 95.6\%$$

### The Address Space

IPv4 multicast uses the Class D range: 224.0.0.0 to 239.255.255.255. This is $2^{28}$ addresses:

$$N_{groups} = 2^{28} = 268,435,456$$

But usable groups are far fewer due to reserved ranges:

| Range | Purpose | Size |
|:---|:---|:---:|
| 224.0.0.0/24 | Link-local (OSPF, VRRP, IGMP) | 256 |
| 224.0.1.0 - 224.0.1.255 | Internetwork control | 256 |
| 232.0.0.0/8 | SSM (source-specific) | 16,777,216 |
| 233.0.0.0/8 | GLOP (AS-based) | 16,777,216 |
| 239.0.0.0/8 | Admin-scoped (private) | 16,777,216 |
| Remainder | Globally scoped | ~201,326,336 |

### Layer 2 Multicast MAC Mapping

The IEEE mapping from IPv4 multicast to Ethernet MAC:

$$\text{MAC} = 0100.5e \| (0 \| \text{IP}[8:0] \| \text{IP}[15:8] \| \text{IP}[22:16])$$

Only the lower 23 bits of the IP address are used, but the IP multicast range has 28 variable bits. This means $2^{28-23} = 2^5 = 32$ IP multicast addresses map to the same MAC:

$$\text{Overlap ratio} = 32:1$$

For example, all of these share MAC `0100.5e01.0101`:

- 224.1.1.1, 225.1.1.1, 226.1.1.1, ... 239.1.1.1
- 224.129.1.1, 225.129.1.1, 226.129.1.1, ... 239.129.1.1

This overlap means IGMP snooping operates on (VLAN, Group IP), not (VLAN, MAC), to avoid delivering unwanted streams.

---

## 2. PIM Sparse Mode — The Dominant DC Multicast Protocol

### Protocol Operation

PIM-SM (RFC 7761) is an explicit-join, RP-based protocol. Its operation follows a state machine with four core phases:

**Phase 1 — Receiver Join (RPT Construction):**

1. Host sends IGMP Membership Report for group $G$ to its local router (Designated Router)
2. DR creates $(*, G)$ state and sends PIM Join toward the RP, hop by hop
3. Each router along the path installs $(*, G)$ state with:
   - Incoming interface = RPF interface toward RP
   - Outgoing Interface List (OIL) = interface where Join was received
4. The shared tree (RPT) from RP to receiver is established

**Phase 2 — Source Registration:**

1. Source sends multicast packet to group $G$
2. Source's DR receives the packet and encapsulates it in a PIM Register message (unicast to RP)
3. RP decapsulates the Register, forwards the multicast packet down the RPT
4. RP sends a PIM Register-Stop back to source DR once native multicast path is established

**Phase 3 — SPT Switchover (Optional):**

1. Last-hop router (receiver DR) receives traffic via RPT
2. If traffic rate exceeds SPT threshold (default 0 = immediately), last-hop DR sends $(S, G)$ Join toward the source
3. SPT is built directly from source to receiver
4. Once $(S, G)$ traffic arrives via SPT, router prunes $(S, G)$ from the RPT by sending $(S, G, rpt)$ Prune toward RP
5. Traffic now flows on the optimal shortest path

**Phase 4 — Steady State:**

- PIM Joins are refreshed every 60 seconds (default)
- PIM Hello sent every 30 seconds (holdtime 3.5x = 105 seconds)
- If no Join refresh is received, state is pruned after the join timer expires

### State Complexity Analysis

For a network with $S$ sources, $G$ groups, and $R$ routers:

**RPT-only state (no SPT switchover):**

$$\text{State per router} = O(G)$$

Each router maintains at most one $(*, G)$ entry per active group, regardless of sources.

**With SPT switchover:**

$$\text{State per router} = O(S \times G)$$

In the worst case, each router on the path has an $(S, G)$ entry per source-group pair.

**State comparison for 100 groups, 50 sources, 40 routers:**

| Mode | Entries per router (worst case) | Total state |
|:---:|:---:|:---:|
| RPT only | 100 | 4,000 |
| SPT (all switched) | 5,000 | 200,000 |
| SSM (no RP) | 5,000 | 200,000 |
| BiDir (shared only) | 100 | 4,000 |

### Designated Router (DR) Election

On multi-access networks (Ethernet), the PIM DR is responsible for:

- Sending PIM Register messages to RP (for sources)
- Sending PIM Joins toward RP (for receivers)

DR election:

1. Highest DR priority wins (default 1, configurable 0-4294967295)
2. If tied, highest IP address wins

The DR election is per-interface and independent of IGMP querier election (which uses the lowest IP).

### Assert Mechanism

When two routers on the same multi-access segment both forward the same multicast stream (causing duplicates), the PIM Assert mechanism resolves the conflict:

1. Both routers detect duplicate multicast on the segment
2. Each sends a PIM Assert message containing:
   - Metric preference (administrative distance to source/RP)
   - Route metric (to source/RP)
   - IP address of the router
3. Winner: lowest metric preference, then lowest metric, then highest IP
4. Loser stops forwarding on that interface for that (S,G) or (*,G)

---

## 3. PIM Variants — SSM, Dense Mode, and Bidirectional

### PIM Source-Specific Multicast (SSM) — RFC 4607

SSM eliminates the RP entirely. Receivers specify both source and group in their IGMP join:

**IGMPv3 (S, G) Join:**
- Host sends IGMPv3 Membership Report with INCLUDE mode for source $S$, group $G$
- DR creates $(S, G)$ state directly and sends PIM $(S, G)$ Join toward $S$
- SPT is built directly — no RPT, no RP, no Register process

**Advantages:**

| Property | PIM-SM | PIM-SSM |
|:---|:---|:---|
| RP required | Yes | No |
| IGMP version | v2 or v3 | v3 only |
| Tree type | RPT then SPT | SPT only |
| Source discovery | RP-based | Application-layer |
| Address overlap | Possible (same G, different S) | Impossible ((S,G) is unique) |
| Security | Source spoofing possible | Source validated by (S,G) |
| Group range | 224.0.0.0/4 (minus SSM) | 232.0.0.0/8 (default) |

**When to use SSM:**

- Sources are well-known and can be communicated to receivers out-of-band
- IPTV, financial market data, live video (source address is published)
- Security is important — only legitimate sources are accepted
- RP availability is a concern

### PIM Dense Mode (DM) — RFC 3973

PIM-DM uses a flood-and-prune model. All routers initially receive all multicast traffic for a group, then routers with no downstream receivers send Prune messages.

**Why DM fails in data centers:**

1. **Initial flooding:** Every new source floods to all routers — unacceptable in large fabrics
2. **State refresh:** DM re-floods every 60 seconds (state-refresh interval), causing periodic bandwidth spikes
3. **State:** $O(S \times G)$ on every router, even those with no receivers
4. **No explicit join:** Cannot control which routers receive traffic until after the flood

DM exists mainly for backward compatibility and tiny networks. It should never be used in a DC fabric.

### PIM Bidirectional (BiDir) — RFC 5015

BiDir builds a single shared tree rooted at the RP. Unlike PIM-SM:

- No source registration (no PIM Register messages)
- No SPT switchover — traffic always traverses the RP tree
- Traffic flows in both directions on the shared tree

**Designated Forwarder (DF) Election:**

On each link, one router is elected as the Designated Forwarder toward the RP. Only the DF forwards multicast upstream toward the RP. Election criteria:

1. Best unicast metric to the RP address wins
2. Tie-breaker: highest IP address

The DF election ensures loop-free forwarding without RPF check failures, because the DF is the only router that forwards multicast from a downstream segment toward the RP.

**Phantom RP:**

In BiDir, the RP address does not need to exist on any actual router. It serves only as a routing vector for DF election. This is called a Phantom RP. As long as all routers have a unicast route to the phantom address, DF election works correctly and traffic flows through the tree.

**State scaling:**

$$\text{BiDir state per router} = O(G)$$

No $(S, G)$ state exists at all — only $(*, G)$. This makes BiDir ideal for many-to-many applications where hundreds of sources send to the same group (video conferencing, collaboration tools).

**Trade-off analysis:**

| Factor | PIM-SM (SPT) | PIM-SM (RPT only) | PIM-BiDir |
|:---|:---|:---|:---|
| State | $O(S \times G)$ | $O(G)$ | $O(G)$ |
| Path optimality | Optimal (SPT) | Suboptimal (via RP) | Suboptimal (via RP) |
| Source registration | Yes | Yes | No |
| RP load | Register processing | Register processing | No registration |
| Traffic symmetry | Asymmetric | Asymmetric | Symmetric (both dirs) |

---

## 4. IGMP — Host-to-Router Signaling

### Version Comparison

| Feature | IGMPv1 | IGMPv2 | IGMPv3 |
|:---|:---|:---|:---|
| RFC | 1112 | 2236 | 3376 |
| Join mechanism | Membership Report | Membership Report | Membership Report |
| Leave mechanism | Timeout only (no leave) | Leave Group message | State-change report |
| Source filtering | No | No | Yes (INCLUDE/EXCLUDE) |
| Querier election | No (relies on routing) | Lowest IP wins | Lowest IP wins |
| Leave latency | Up to 260s | ~3s (query interval) | ~3s |
| SSM support | No | No | Yes |
| Max Response Time | Fixed 10s | Variable (tunable) | Variable (tunable) |

### IGMP Timers and Scaling Math

**Group Membership Interval (GMI):**

$$GMI = (QRV \times QI) + QRI$$

Where:
- $QRV$ = Querier Robustness Variable (default 2)
- $QI$ = Query Interval (default 125 seconds)
- $QRI$ = Query Response Interval (default 10 seconds)

$$GMI = (2 \times 125) + 10 = 260 \text{ seconds}$$

A host that does not respond to $QRV$ consecutive queries is removed after $GMI$.

**Last Member Query Interval (LMQI):**

When a leave is received, the querier sends group-specific queries:

$$\text{Leave latency} = LMQC \times LMQI$$

Where:
- $LMQC$ = Last Member Query Count (default = $QRV$ = 2)
- $LMQI$ = Last Member Query Interval (default 1 second)

$$\text{Leave latency} = 2 \times 1 = 2 \text{ seconds}$$

**Tuning for fast convergence (aggressive timers):**

| Parameter | Default | Aggressive | Effect |
|:---|:---:|:---:|:---|
| Query Interval | 125s | 30s | Faster detection of departed receivers |
| Robustness Variable | 2 | 2 | Keep at 2 (lowering risks false removals) |
| Max Response Time | 10s | 5s | Reports spread over shorter window |
| LMQI | 1s | 0.5s | Faster leave latency (1s total) |
| GMI | 260s | 65s | Members timeout in 65s vs 260s |

**Warning:** Aggressive timers increase IGMP query/report traffic proportionally. For a VLAN with $H$ hosts and $G$ active groups:

$$\text{IGMP reports/sec} \approx \frac{H \times G}{QI}$$

At defaults: $\frac{500 \times 20}{125} = 80$ reports/sec. With aggressive QI=30: $\frac{500 \times 20}{30} = 333$ reports/sec.

### Querier Election

On each VLAN/subnet, exactly one IGMP querier exists:

1. All PIM-enabled L3 interfaces on the subnet participate
2. **Lowest IP address wins** (opposite of PIM DR election which uses highest IP by default)
3. Querier sends periodic General Queries
4. Non-querier routers still process reports (for multicast forwarding) but do not query

In an IGMP snooping environment without a PIM router, the switch must run an IGMP snooping querier to maintain group state.

---

## 5. IGMP Snooping — Layer 2 Multicast Optimization

### How Snooping Works

Without IGMP snooping, a Layer 2 switch treats all multicast frames as broadcast — flooding them to every port in the VLAN. IGMP snooping inspects IGMP messages to build a table mapping (VLAN, Group) to a set of ports.

**Snooping Data Structures:**

```
Port Table:
  VLAN 100, Group 239.1.1.1 -> Ports: Eth1/1, Eth1/5, Eth1/12
  VLAN 100, Group 239.1.1.2 -> Ports: Eth1/1, Eth1/3
  VLAN 100, Mrouter ports  -> Ports: Eth1/49, Eth1/50

Forwarding logic:
  If group is in snooping table:
    Forward to group's port set UNION mrouter ports
  Else:
    Flood to all ports in VLAN (unknown multicast)
```

**Mrouter Port Detection:**

Mrouter ports are identified by:
- Receiving PIM Hello messages
- Receiving IGMP General Queries
- Static configuration

All multicast traffic (known and unknown groups) is always forwarded to mrouter ports, ensuring the multicast router sees all groups.

### Report Suppression

In IGMPv1/v2, when a host hears another host's Membership Report for the same group, it suppresses its own report (to reduce traffic). IGMP snooping can also suppress duplicate reports toward the querier.

**Problem with IGMPv3:** Report suppression prevents the switch from seeing individual host source-filter state. For IGMPv3 (SSM), report suppression should be disabled:

```
! IOS
no ip igmp snooping vlan 100 report-suppression

! NX-OS
no ip igmp snooping report-suppression
```

### Fast Leave (Immediate Leave)

Normally, when a Leave is received, the querier sends group-specific queries and waits for the LMQI timeout before removing the port. Fast leave skips this and removes the port immediately.

**Safe only when:** Exactly one host exists per port (access ports, not trunks or port-channels to other switches).

**Unsafe when:** Multiple hosts share a port (downstream switch, wireless AP). Removing the port immediately kills multicast for all hosts behind it.

### Topology Change Notification (TCN) Behavior

When STP detects a topology change:
1. IGMP snooping tables may be invalid (ports changed)
2. Switch floods all multicast for a configurable period (TCN flood time)
3. After the flood period, snooping re-learns from IGMP reports

$$\text{TCN flood duration} = QRV \times QI + QRI = GMI$$

At default timers, this is 260 seconds of flooding after every STP topology change — a strong argument for avoiding STP in DC fabrics (use ECMP/VXLAN instead).

---

## 6. Rendezvous Point Architecture

### Static RP

All routers manually configured with the same RP address:

$$\text{Configuration complexity} = O(R)$$

Where $R$ is the number of routers. Every router needs the same `ip pim rp-address` command. No dynamic discovery, no failover.

**Failure mode:** If the RP router dies, all new joins fail. Existing (S,G) SPT entries continue to work (they do not use the RP), but new receivers cannot join.

### Auto-RP (Cisco Proprietary)

Two roles:
- **RP-Candidate:** Announces itself on 224.0.1.39 (Cisco-RP-Announce)
- **Mapping Agent:** Listens on 224.0.1.39, selects best RP per group, advertises on 224.0.1.40 (Cisco-RP-Discovery)

**Chicken-and-egg problem:** Auto-RP uses multicast (224.0.1.39/40), but PIM-SM requires an RP to deliver multicast. Solution: `ip pim autorp listener` treats Auto-RP groups as dense-mode, ensuring they are flooded without an RP.

### Bootstrap Router (BSR) — RFC 5059

BSR election:
1. Candidate BSRs announce with priority (0-255, highest wins)
2. Tie-breaker: highest IP
3. Elected BSR collects Candidate RP advertisements
4. BSR floods RP-set via hop-by-hop BSR messages (TTL-scoped)
5. All PIM routers install the RP-set from BSR

**BSR Hash Function:**

When multiple RPs are candidates for overlapping group ranges, each router uses a hash function to deterministically select an RP per group:

$$\text{Hash}(G, M, C) = (1103515245 \times ((1103515245 \times (G \mathbin{\&} M) + 12345) \oplus C) + 12345) \mod 2^{31}$$

Where:
- $G$ = group address
- $M$ = hash mask (configured on BSR)
- $C$ = candidate RP address

The RP with the highest hash value for a given group wins. The hash mask length determines how many groups map to the same RP — a longer mask means more groups share an RP (less load distribution).

### Anycast RP with MSDP — RFC 3446

Multiple physical routers share the same loopback IP address (the Anycast RP address). IGP routing delivers PIM Joins/Registers to the topologically closest RP. MSDP synchronizes source state between RPs.

**How Anycast RP with MSDP works:**

1. Source S registers with its nearest RP (say RP1)
2. RP1 creates (S, G) state and generates an MSDP Source-Active (SA) message
3. SA message is sent via TCP to all MSDP peers (RP2, RP3, ...)
4. RP2 receives SA, creates (S, G) state, and can now serve joins for group G with knowledge of source S
5. Receiver behind RP2 joins G, RP2 knows about S from MSDP, and can either forward from cache or trigger SPT to S

**SA Message Contents:**

```
Source-Active (SA):
  Source: S
  Group: G
  RP: originating RP address (unique per router, NOT the Anycast address)
  Encapsulated data: optionally includes the first multicast packet
```

**MSDP Mesh Groups:**

In a full-mesh MSDP topology (all RPs peer with all other RPs), SA messages would be forwarded redundantly. A mesh-group tells MSDP: "All members of this group already have full-mesh peering, so do not re-forward SA messages received from a mesh-group member to other mesh-group members."

Without mesh-groups for $N$ MSDP peers:

$$\text{SA copies} = N \times (N - 1)$$

With a single mesh-group:

$$\text{SA copies} = N - 1$$

For 4 Anycast RPs: $4 \times 3 = 12$ copies without mesh-groups, versus $3$ copies with a mesh-group.

### NX-OS PIM Anycast RP (No MSDP Required)

NX-OS supports Anycast RP natively via `ip pim anycast-rp`. Instead of MSDP SA messages, PIM register messages are forwarded between RP peers using PIM's own mechanism. This is simpler and avoids the MSDP TCP session overhead.

**How it works:**

1. All Anycast RP peers are configured with each other's unique loopback addresses via `ip pim anycast-rp <anycast-addr> <peer-loopback>`
2. When a source registers with one RP, that RP forwards the register to all other RP peers
3. Each RP peer can serve receiver joins independently

This is the preferred approach on NX-OS platforms — MSDP is needed only for inter-domain multicast or mixed-vendor environments.

---

## 7. Reverse Path Forwarding (RPF)

### The RPF Check

RPF is the fundamental loop-prevention mechanism in multicast routing. For every multicast packet received:

1. Look up the source IP in the unicast routing table (FIB)
2. Determine the outgoing interface for reaching that source — this is the RPF interface
3. If the packet arrived on the RPF interface, it passes the RPF check and is forwarded
4. If the packet arrived on any other interface, it fails the RPF check and is dropped

**For shared trees (*, G):**

The RPF check is performed against the RP address, not the source. The RPF interface is the interface toward the RP.

**For source trees (S, G):**

The RPF check is performed against the source address. The RPF interface is the interface toward the source.

### RPF in Asymmetric Routing

In networks with asymmetric routing (traffic from A to B takes a different path than B to A), RPF failures occur because the multicast packet arrives on a non-RPF interface.

**Solutions:**

1. **Make routing symmetric:** Ensure IGP metrics create symmetric paths
2. **Static mroute:** Override RPF with a manual route:
   ```
   ip mroute 10.1.1.0 255.255.255.0 10.2.2.1
   ```
   This tells the router to use 10.2.2.1 as the RPF neighbor for sources in 10.1.1.0/24
3. **Multicast-specific routing:** Run a separate routing instance for multicast RPF (complex, rarely used)

### RPF in ECMP

When multiple equal-cost paths exist to a source (ECMP), the router must deterministically select one RPF interface. The selection method varies by platform:

- **Cisco IOS:** Uses the path with the highest next-hop IP address
- **NX-OS:** Uses a hash of (S, G) to distribute RPF across ECMP paths
- **Some platforms:** Use the first path installed in the RIB

For PIM, the RPF neighbor (not just interface) matters because PIM Joins are sent to the RPF neighbor. If ECMP causes different routers to select different RPF neighbors, the resulting tree may have asymmetric branches. This is generally acceptable but can be surprising during troubleshooting.

---

## 8. Multicast in VXLAN EVPN Fabrics

### The BUM Problem

VXLAN encapsulates Layer 2 frames in UDP/IP. When a frame must be flooded (Broadcast, Unknown unicast, Multicast — BUM), the ingress VTEP must replicate it to all remote VTEPs hosting the same VNI.

Two replication strategies exist:

### Strategy 1: Ingress Replication (Head-End Replication)

The ingress VTEP sends a separate unicast copy to each remote VTEP.

**Bandwidth cost at the source:**

$$B_{source} = (N_{VTEP} - 1) \times S_{frame}$$

For a BUM-heavy workload with $F$ BUM frames/sec:

$$B_{total} = F \times (N_{VTEP} - 1) \times S_{frame}$$

**Worked example:** 100 VTEPs, 1000 BUM frames/sec, 128 bytes average:

$$B_{total} = 1000 \times 99 \times 128 \times 8 = 101.4 \text{ Mbps per VTEP}$$

Total fabric BUM overhead: $100 \times 101.4 = 10.14$ Gbps.

**Advantages:**
- No multicast in the underlay — pure unicast infrastructure
- Simple to deploy — BGP EVPN Type-3 Inclusive Multicast routes distribute VTEP lists
- Works with any underlay (including public cloud)

**Disadvantages:**
- $O(N)$ replication at the source VTEP — CPU and bandwidth intensive
- Does not scale well beyond ~50-64 VTEPs per VNI
- Each BUM frame consumes $N-1$ copies on the source uplink

### Strategy 2: Multicast Underlay

Each VNI is mapped to a multicast group. VTEPs join the multicast group for their local VNIs. BUM traffic is sent to the multicast group and replicated by the underlay PIM infrastructure.

**Bandwidth cost at the source:**

$$B_{source} = 1 \times S_{frame}$$

The source VTEP sends exactly one copy. PIM replicates at spine/intermediate points.

**Fabric bandwidth:**

$$B_{fabric} = L_{tree} \times F \times S_{frame}$$

Where $L_{tree}$ is the number of links in the multicast distribution tree.

In a 2-spine, 20-leaf fabric: $L_{tree} = 20 + 2 = 22$ links, versus $20 \times 19 = 380$ unicast copies with ingress replication.

**VNI-to-Group Mapping Strategies:**

| Strategy | Description | Groups Needed | Pro | Con |
|:---|:---|:---:|:---|:---|
| 1:1 | One group per VNI | $N_{VNI}$ | Finest control, prune unused VNIs | Group exhaustion with many VNIs |
| N:1 | Many VNIs share one group | 1 - few | Simple | Unnecessary replication to VTEPs without all VNIs |
| Range-based | VNI ranges map to group ranges | Moderate | Balance | Moderate complexity |

**Design recommendation:** Use a small number of multicast groups (e.g., 4-8) to keep underlay PIM state minimal, accepting some over-replication. The underlay multicast state is $O(G_{underlay})$ which is far smaller than the overlay VNI count.

### Overlay Multicast (Tenant Multicast Through VXLAN)

Tenant multicast (applications sending multicast within the overlay) is distinct from BUM replication. Options:

1. **Multicast as BUM:** Treat all tenant multicast as BUM — simple but floods everywhere
2. **IGMP snooping in the overlay:** VTEPs snoop IGMP in the overlay VNI, build per-group forwarding tables, and replicate only to VTEPs with interested receivers
3. **Multicast EVPN (Type-6/7/8 routes):** BGP EVPN multicast extensions (draft-ietf-bess-evpn-igmp-mld-proxy) distribute IGMP state via BGP, enabling optimized multicast forwarding in the overlay without flooding

**Current state (2025):** Most DC fabrics treat tenant multicast as BUM. EVPN multicast optimization is available on NX-OS 9.3+ and some other platforms but is not universally deployed.

---

## 9. PIM in Spine-Leaf Fabrics

### Design Principles

1. **RP placement:** Always on spines — never on leaves. Spines are the network core and provide symmetric paths to all leaves
2. **Anycast RP:** All spines share the Anycast RP address. Provides RP redundancy without MSDP (on NX-OS) or with MSDP (mixed vendors)
3. **PIM on all fabric links:** Every spine-leaf interconnect runs PIM sparse-mode
4. **ECMP and multicast:** With ECMP between leaf and spines, RPF selects one spine. Multicast traffic uses one of the N uplinks, not all of them (no ECMP load sharing for multicast)
5. **IGMP on leaf SVIs:** Leaves run IGMP on tenant-facing SVIs; spines do not need IGMP

### State Analysis

In a spine-leaf fabric with $S$ spines, $L$ leaves, $G$ active multicast groups, and $N$ sources:

**Spine state:**

$$\text{State}_{spine} = G \times (1 + N) = G + G \times N$$

Each spine holds $(*, G)$ for every group plus potentially $(S_i, G)$ for each source (if SPT switchover occurs).

**Leaf state:**

$$\text{State}_{leaf} = G_{local} + (S \times G_{local})$$

Where $G_{local}$ is groups with local receivers/sources.

**Optimization:** Use `ip pim spt-threshold infinity` on leaves to prevent SPT switchover. This keeps all traffic on the shared tree (RPT) and reduces state to $O(G)$ per router. The trade-off is suboptimal paths through the RP (spine), but in a spine-leaf topology the RP is at most one extra hop — the penalty is minimal.

### ECMP Considerations for Multicast

Unlike unicast, multicast cannot load-balance across ECMP paths because:

1. RPF requires a single deterministic path back to the source
2. Forwarding multicast on multiple paths would create duplicate packets
3. Each (S,G) or (*, G) entry uses exactly one incoming interface

In a leaf with 4 uplinks to 4 spines, multicast traffic for a given group uses only 1 of the 4 uplinks. Different groups may hash to different uplinks (platform-dependent), providing some distribution across the fabric.

**Implication:** Multicast-heavy workloads may create asymmetric spine utilization. Monitor per-spine multicast traffic and adjust if needed.

---

## 10. Multicast with vPC

### The Dual-Path Challenge

vPC (virtual Port Channel) presents two switches as one logical switch to downstream devices. For multicast, this creates several challenges:

1. **Single forwarder:** Only one vPC peer should forward multicast to downstream devices (otherwise duplicates occur)
2. **IGMP snooping consistency:** Both peers must have identical snooping state
3. **PIM DR/Querier:** Must be consistent — typically the vPC primary acts as DR and querier
4. **Orphan ports:** Ports connected to only one vPC peer (not via port-channel) require special handling

### Forwarding Rules

**Multicast on vPC port-channels:**
- The vPC primary (operationally) is the designated forwarder for multicast on vPC port-channels
- The secondary has the same (*, G) and (S, G) state but does not forward on the vPC leg
- If the primary fails, the secondary takes over forwarding immediately

**IGMP snooping with vPC:**
- Both peers independently run IGMP snooping
- CFS (Cisco Fabric Services) synchronizes snooping state over the vPC peer-link
- IGMP reports received on one peer are relayed to the other over the peer-link
- Both peers must have identical IGMP snooping configuration — verify with `show vpc consistency-parameters global`

**PIM on vPC SVIs:**
- Both vPC peers run PIM independently on SVIs
- PIM DR election occurs normally — ensure the primary has the lower IP (for IGMP querier) or explicitly set DR priority
- PIM Joins are sent by the DR only — the non-DR peer still maintains state for failover

### vPC Consistency Checks

vPC performs consistency checks to ensure both peers have matching configuration. For multicast, the following must match:

- IGMP snooping enabled/disabled per VLAN
- IGMP snooping querier configuration
- IGMP version per interface
- PIM sparse-mode enabled/disabled per SVI
- PIM DR priority per SVI
- IGMP snooping fast-leave configuration

A consistency check failure causes the secondary peer to suspend the affected VLANs on the vPC port-channel — this is a hard failure that disrupts traffic.

---

## 11. MSDP for Inter-Domain Multicast

### Protocol Mechanics

MSDP (RFC 3618) connects separate PIM-SM domains. Each domain has its own RP. MSDP peers (typically the RPs) exchange Source-Active (SA) messages over TCP (port 639).

**SA message flow:**

1. Source $S$ in Domain A sends multicast to group $G$
2. RP-A receives the Register, creates $(S, G)$ state
3. RP-A generates an SA message: $\text{SA}(S, G, \text{RP-A})$
4. SA is sent to all MSDP peers (RP-B, RP-C, ...)
5. RP-B receives SA, performs RPF check on the SA (using the originating RP address)
6. If RPF passes, RP-B accepts the SA and caches it
7. If a receiver in Domain B has joined group $G$, RP-B can now create $(S, G)$ state and build an SPT to the source across domain boundaries

**SA RPF Check:**

MSDP uses its own RPF check on SA messages to prevent loops. The RP address in the SA must be reachable via the MSDP peer that sent the SA. This is checked against the BGP (or unicast) routing table.

**SA caching:**

SA entries are cached with a lifetime of 6 minutes (default). They are refreshed by periodic SA messages (every 60 seconds from the originating RP).

### MSDP Scaling

$$\text{SA cache size} = S_{total} \times G_{total}$$

Where $S_{total}$ is total sources across all domains and $G_{total}$ is total groups. For internet-scale MSDP (global multicast), SA caches can grow to hundreds of thousands of entries.

In a DC context with Anycast RP, MSDP SA caches are small (only local sources) and MSDP is purely for RP synchronization.

---

## Prerequisites

- Solid understanding of IP routing (unicast FIB, administrative distance, IGP metrics)
- Familiarity with VXLAN and BGP EVPN concepts (VNI, VTEP, Type-3 routes)
- Knowledge of STP and L2 switching fundamentals (for IGMP snooping context)
- Understanding of spine-leaf fabric topology and ECMP
- Familiarity with IOS/NX-OS CLI for configuration examples

---

## References

- RFC 7761 — Protocol Independent Multicast - Sparse Mode (PIM-SM): Protocol Specification (Revised)
- RFC 4607 — Source-Specific Multicast for IP
- RFC 5015 — Bidirectional Protocol Independent Multicast (BIDIR-PIM)
- RFC 3973 — Protocol Independent Multicast - Dense Mode (PIM-DM)
- RFC 3376 — Internet Group Management Protocol, Version 3
- RFC 2236 — Internet Group Management Protocol, Version 2
- RFC 1112 — Host Extensions for IP Multicasting (IGMPv1)
- RFC 4541 — Considerations for IGMP and MLD Snooping Switches
- RFC 3618 — Multicast Source Discovery Protocol (MSDP)
- RFC 5059 — Bootstrap Router (BSR) Mechanism for Protocol Independent Multicast (PIM)
- RFC 3446 — Anycast Rendezvous Point (RP) Mechanism using Protocol Independent Multicast (PIM) and Multicast Source Discovery Protocol (MSDP)
- RFC 7432 — BGP MPLS-Based Ethernet VPN
- RFC 8365 — A Network Virtualization Overlay Solution Using Ethernet VPN (EVPN)
- Cisco NX-OS Multicast Routing Configuration Guide
- Cisco IOS IP Multicast Command Reference
- draft-ietf-bess-evpn-igmp-mld-proxy — IGMP and MLD Proxy for EVPN
