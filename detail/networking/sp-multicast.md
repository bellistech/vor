# The Mathematics of SP Multicast — Trees, Labels, and Bandwidth

> *Multicast is the only technology where sending the same data to a million receivers costs the same as sending it to one. The math proves why, and reveals where the cost hides: in the control plane.*

---

## 1. SP Multicast Scaling Challenges

### The Bandwidth Argument for Multicast

The fundamental value proposition of multicast is bandwidth efficiency. For a source sending at rate $R$ to $N$ receivers:

**Unicast replication:**

$$BW_{unicast} = N \times R$$

**Multicast (shortest-path tree):**

$$BW_{multicast} = E_{tree} \times R$$

Where $E_{tree}$ = number of edges (links) in the multicast distribution tree. In any tree:

$$E_{tree} = V_{tree} - 1$$

Where $V_{tree}$ = number of nodes in the tree (routers that are part of the distribution path).

**Bandwidth savings ratio:**

$$S = \frac{BW_{unicast}}{BW_{multicast}} = \frac{N \times R}{E_{tree} \times R} = \frac{N}{E_{tree}}$$

**Worked example:** 500 receivers across a 200-router SP network. The multicast tree might include 80 routers (not all routers have receivers):

$$S = \frac{500}{79} \approx 6.3\times \text{ bandwidth savings}$$

For a 10 Mbps HD IPTV channel: unicast needs 5 Gbps total, multicast needs ~790 Mbps across all links combined.

### State Scaling: The Hidden Cost

While multicast saves bandwidth, it creates per-group state on every router in the tree:

$$\text{Total state entries} = \sum_{r \in \text{routers}} G_r$$

Where $G_r$ = number of (S,G) or (*,G) entries on router $r$.

For a router at the core of an IPTV network with 500 channels (SSM):

$$\text{State per core router} = 500 \text{ (S,G) entries}$$

For ASM with shared trees and source-specific trees:

$$\text{State per core router} = G_{shared} + G_{source} = 500 + 500 = 1{,}000 \text{ entries}$$

With multicast VPN (multiple VRFs):

$$\text{Total state} = \sum_{v \in \text{VRFs}} G_v = n_{VRF} \times G_{avg}$$

100 VRFs with 50 groups each: 5,000 entries per core router. This is manageable for modern hardware, but the control plane processing (PIM join/prune, mroute updates) scales with state count.

---

## 2. mLDP Label Tree Construction

### How mLDP Builds P2MP Trees

mLDP extends LDP to build label-switched multicast trees. The process is receiver-driven (leaf-initiated), similar to PIM:

**Step 1: Leaf node signals interest**

A leaf PE wants to join a P2MP tree rooted at $R$ with opaque value $O$ (which identifies the multicast stream):

$$\text{FEC} = (R, O) \text{ where } R = \text{root IP}, O = \text{opaque identifier}$$

**Step 2: Label Mapping propagates hop-by-hop toward root**

Each router along the IGP shortest path to $R$:
1. Receives Label Mapping from downstream
2. Allocates a local label for the FEC
3. Installs forwarding entry: `incoming_label -> {outgoing_label_1, outgoing_label_2, ...}`
4. Sends Label Mapping upstream (toward root)

```
Root (R)
  │ ← Label Mapping from LSR-A (label 5000)
  │
LSR-A
  │ ← Label Mapping from LSR-B (label 6000)
  │ ← Label Mapping from LSR-C (label 7000)
  │
  ├── LSR-B (leaf)  → allocates label 6000, sends upstream
  └── LSR-C
       │ ← Label Mapping from LSR-D (label 8000)
       │
       └── LSR-D (leaf) → allocates label 8000, sends upstream
```

**Step 3: Forwarding state installed**

At LSR-A (transit node):

| Incoming Label | Outgoing Labels | Outgoing Interfaces |
|---------------|----------------|-------------------|
| 5000 | 6000, 7000 | eth1 (to LSR-B), eth2 (to LSR-C) |

The root pushes the mLDP label and sends the packet. Each transit LSR performs label swap and replication. Each leaf pops the label and delivers the payload.

### Tree Merge Optimization

When multiple leaves join the same P2MP tree, their Label Mapping messages merge at common ancestors:

```
Time T1: Leaf-B joins tree (R, O)
  Path: Leaf-B -> LSR-C -> LSR-A -> Root
  Labels allocated at each hop

Time T2: Leaf-D joins same tree (R, O)
  Path: Leaf-D -> LSR-C -> LSR-A -> Root
  At LSR-C: tree already exists toward LSR-A
  LSR-C only adds Leaf-D to its outgoing interface list
  No new Label Mapping sent upstream (tree already reaches root)
```

**Merge point efficiency:**

$$\text{New state added} = \text{path from new leaf to nearest existing tree node}$$

Not the full path to root. This is the fundamental efficiency of receiver-driven tree construction.

### Make-Before-Break (MBB)

When IGP topology changes, the mLDP tree must reconverge. Make-before-break ensures traffic continues during reconvergence:

1. New tree is built along updated IGP shortest path
2. Root starts sending on both old and new tree
3. Leaf confirms new tree is receiving
4. Old tree is torn down

$$T_{MBB} = T_{IGP\_converge} + T_{LDP\_signal} + T_{verify} + T_{teardown}$$

Typical: 1-5 seconds for the full process, with zero packet loss during transition.

---

## 3. P2MP RSVP-TE Signaling

### Signaling Flow

P2MP RSVP-TE uses extensions defined in RFC 4875. The signaling is source-driven (head-end initiated):

**Step 1: Head-end computes paths to all leaves using CSPF**

For each leaf $L_i$, compute constrained shortest path considering:
- Bandwidth constraints (each branch must have $BW_{stream}$ available)
- Admin groups (link affinities/colors)
- SRLG (Shared Risk Link Group) diversity

**Step 2: Path messages sent toward leaves**

Unlike P2P RSVP-TE (one Path per tunnel), P2MP uses sub-LSPs:

$$\text{P2MP LSP} = \{S2L_1, S2L_2, ..., S2L_n\}$$

Where each $S2L_i$ (Source-to-Leaf) is a sub-LSP to leaf $L_i$.

Path messages for sub-LSPs sharing common hops are bundled to reduce signaling:

```
Head-end sends Path to LSR-A:
  S2L sub-LSPs: {L1, L2, L3}  (all three leaves reachable via LSR-A)

LSR-A splits and sends:
  Path to LSR-B: S2L sub-LSPs {L1, L2}
  Path to LSR-C: S2L sub-LSPs {L3}
```

**Step 3: Resv messages return (leaf to head-end)**

Each leaf sends Resv confirming label allocation and bandwidth reservation. Resv messages merge at branch points.

### Bandwidth Reservation

Each link in the P2MP tree reserves bandwidth for the multicast stream:

$$BW_{reserved}(link) = R_{stream} \times \mathbb{1}[\text{link is in tree}]$$

Total network bandwidth reserved:

$$BW_{total} = E_{tree} \times R_{stream}$$

Note: a link only reserves $R_{stream}$ once, regardless of how many downstream leaves it serves. This is the multicast bandwidth advantage expressed in RSVP-TE terms.

### FRR (Fast Reroute) for P2MP

Each branch point in the P2MP tree can have FRR backup:

- **Facility backup:** Pre-computed bypass tunnel protects each link
- **One-to-one backup:** Dedicated backup path per sub-LSP

$$T_{FRR} = T_{detect} + T_{switch} \approx 50 \text{ms (BFD)} + 0 \text{ms (pre-installed)} = 50 \text{ms}$$

For IPTV: 50ms protection means zero visible impact to viewers (below the video codec error concealment threshold).

---

## 4. Inter-AS Multicast Options

### Option A: Back-to-Back VRF

```
AS 65001                    AS 65002
PE1 ─── (VRF) ─── ASBR1 ═══ ASBR2 ─── (VRF) ─── PE2
                   VRF on     VRF on
                   both sides both sides
```

- PIM runs per-VRF on the inter-AS link
- Simple but does not scale (per-VRF interface required)
- Each VRF maintains independent multicast state

### Option B: MP-eBGP with Label

```
AS 65001                         AS 65002
PE1 ─── P ─── ASBR1 ═══ ASBR2 ─── P ─── PE2
                   MP-eBGP exchanges
                   VPNv4/mcast routes
                   with labels
```

- ASBRs exchange mVPN routes via MP-BGP
- mLDP or RSVP-TE within each AS
- Inter-AS segment uses labeled unicast or mLDP stitching

### Option C: Multihop MP-eBGP via RR

```
AS 65001                              AS 65002
PE1 ─── P ─── ASBR1 ═══ ASBR2 ─── P ─── PE2
                    │                │
                    RR1 ════════════ RR2
                    (multihop MP-eBGP)
```

- Route Reflectors exchange mVPN routes across AS boundaries
- ASBRs only handle label forwarding (no mVPN awareness)
- Most scalable but requires inter-AS LDP/RSVP-TE stitching

### Inter-AS Multicast State Scaling

For $A$ autonomous systems, each with $V$ VRFs and $G$ groups per VRF:

**Option A:** State per ASBR = $V \times G$ (per-VRF, per-group)

**Option B:** State per ASBR = $V \times G$ (same, but with labels)

**Option C:** State per ASBR = 0 (multicast state only on PE/RR)

---

## 5. Multicast Convergence in Large Networks

### Convergence Components

When a link or node fails, multicast must reconverge:

$$T_{mcast\_converge} = T_{detect} + T_{IGP\_converge} + T_{RPF\_update} + T_{tree\_rebuild}$$

| Component | Mechanism | Typical Time |
|-----------|-----------|-------------|
| Detection | BFD (50ms intervals) | 150ms (3x miss) |
| IGP convergence | OSPF/IS-IS SPF | 200ms-1s |
| RPF update | PIM RPF check recalculation | 50-200ms |
| Tree rebuild | PIM Join/Prune propagation | 100ms-3s (diameter dependent) |
| **Total (no FRR)** | | **500ms - 5s** |
| **Total (with mLDP MBB)** | | **200ms - 2s** |
| **Total (with RSVP-TE FRR)** | | **50ms - 200ms** |

### PIM Join/Prune Propagation Time

PIM Join messages propagate hop-by-hop from receiver to source. Each hop adds processing delay:

$$T_{join} = H \times (T_{process} + T_{propagation})$$

Where:
- $H$ = number of hops from receiver to source (or RP)
- $T_{process}$ = per-hop PIM processing time (~10-50ms)
- $T_{propagation}$ = link propagation delay (~1ms per 200km fiber)

For a 10-hop path with 20ms processing per hop:

$$T_{join} = 10 \times (20 + 1) = 210 \text{ms}$$

### Multicast Reconvergence with Anycast RP

When an RP fails, PIM must:
1. Detect RP failure (IGP convergence to RP loopback)
2. RPF shifts to surviving anycast RP (automatic, no PIM reconfiguration)
3. Sources re-register with new closest RP
4. MSDP SA cache on surviving RP already has source information

$$T_{RP\_failover} = T_{IGP\_converge} + T_{re\_register}$$

With pre-populated MSDP SA cache, re-registration is minimal. Total: ~1-3 seconds for steady-state multicast to recover.

---

## 6. Bandwidth Optimization Analysis

### Multicast vs Unicast Break-Even Point

At what receiver count does multicast become worthwhile? Define the overhead of running multicast:

$$C_{multicast} = C_{state} + C_{signaling}$$

Where:
- $C_{state}$ = router memory and TCAM for mroute entries
- $C_{signaling}$ = control plane CPU for PIM/mLDP

Break-even when bandwidth saved exceeds overhead:

$$N_{break-even} = \frac{C_{multicast}}{R_{stream} \times \Delta_{path}}$$

Where $\Delta_{path}$ = average path length savings per receiver.

In practice, multicast pays for itself at **2-3 receivers per group** for high-bandwidth streams (video) and at **10-20 receivers** for low-bandwidth streams (software updates).

### IPTV Bandwidth Planning

For an IPTV service with $C_{SD}$ SD channels, $C_{HD}$ HD channels, and $C_{4K}$ 4K channels:

$$BW_{lineup} = C_{SD} \times R_{SD} + C_{HD} \times R_{HD} + C_{4K} \times R_{4K}$$

Typical bitrates:
- $R_{SD} = 3.5$ Mbps (MPEG-4 AVC)
- $R_{HD} = 8$ Mbps (MPEG-4 AVC) or 5 Mbps (HEVC)
- $R_{4K} = 25$ Mbps (HEVC) or 15 Mbps (VVC)

**Example:** 200 SD + 100 HD + 20 4K channels:

$$BW_{lineup} = 200 \times 3.5 + 100 \times 8 + 20 \times 25 = 700 + 800 + 500 = 2{,}000 \text{ Mbps}$$

**Key insight:** With multicast, this 2 Gbps is the maximum bandwidth per link, regardless of whether 100 or 100,000 subscribers watch. With unicast, 100,000 subscribers watching 2 channels each at 8 Mbps = 1.6 Tbps.

### Popular Channel Distribution (Zipf's Law)

Channel popularity follows a Zipf-like distribution:

$$P(k) \propto \frac{1}{k^s}$$

Where $k$ = channel rank and $s \approx 0.8$ for TV viewing.

This means the top 10% of channels carry ~60% of viewers. Implications:
- Pre-join the top 20 channels (always flowing through the network)
- Long-tail channels may have no viewers on some tree branches
- Data MDT (dynamic multicast trees) can be used to build trees only when receivers exist

### Data MDT Threshold Analysis

Multicast VPN uses a default MDT (always-on shared tree) and data MDTs (built on demand for high-bandwidth groups):

$$\text{Use data MDT when } R_{group} > T_{threshold}$$

Typical threshold: 10-50 kbps. Below threshold, traffic rides the default MDT (shared among all groups). Above threshold, a dedicated P2MP tree is built.

Trade-off:
- Low threshold: More trees, more state, better bandwidth efficiency
- High threshold: Fewer trees, less state, some bandwidth waste on default MDT

---

## 7. IGMP/MLD Scaling at Aggregation

### State Explosion at the BNG

A BNG serving 50,000 subscribers, each allowed to join up to 10 multicast groups:

$$\text{Max IGMP state} = 50{,}000 \times 10 = 500{,}000 \text{ entries}$$

This is a control plane problem. Each IGMP Report must be processed, and state must be maintained per-subscriber per-group.

### IGMP Proxy Aggregation

The BNG acts as an IGMP proxy, aggregating thousands of subscriber joins into a single PIM join per group upstream:

$$\text{PIM state upstream} = G \text{ (unique groups)}$$

Even if 10,000 subscribers join the same channel, the BNG sends one PIM join. The aggregation ratio:

$$R_{aggregation} = \frac{\text{IGMP entries (subscriber-facing)}}{\text{PIM state (upstream)}} = \frac{N_{subs} \times G_{avg}}{G_{unique}}$$

**Example:** 50,000 subscribers averaging 3 groups each, 300 unique groups:

$$R_{aggregation} = \frac{50{,}000 \times 3}{300} = 500\times$$

### IGMP Query/Report Rate

IGMP general queries are sent at `query-interval` (default 125 seconds). Each subscriber responds:

$$\text{Reports per second} = \frac{N_{subs} \times G_{per\_sub}}{\text{max-response-time}}$$

With 50,000 subscribers, 3 groups each, 10-second response window:

$$\text{Report rate} = \frac{50{,}000 \times 3}{10} = 15{,}000 \text{ reports/sec}$$

This is why tuning max-response-time and using IGMP snooping to limit report scope is critical at scale.

### Fast Leave Analysis

Without fast leave, the BNG must send a group-specific query and wait for response:

$$T_{leave} = T_{query} + T_{response} = \text{last-member-query-interval} \times \text{last-member-query-count}$$

Default: $T_{leave} = 1\text{s} \times 2 = 2\text{s}$

With fast leave (immediate-leave): $T_{leave} = 0$

For IPTV channel change, this 2-second difference is the difference between acceptable and unacceptable zap time.

---

## 8. Multicast Security in SP Networks

### Threat Model

| Threat | Description | Impact |
|--------|-------------|--------|
| Source spoofing | Attacker sends multicast as legitimate source | Content injection, DoS |
| Group join flood | Subscriber joins thousands of groups | State exhaustion on BNG |
| PIM message injection | Forged PIM Join/Register/Assert | Tree hijacking, black-holing |
| MSDP SA injection | Forged SA messages between domains | Rogue source propagation |
| RPF bypass | Traffic injected on non-RPF interface | Loop creation, amplification |

### Mitigations

**Source validation (uRPF for multicast):**

$$\text{Accept multicast from source } S \text{ only if RPF check passes}$$

The RPF check ensures the packet arrived on the interface that the routing table says is the shortest path toward the source. This is inherent in PIM-SM but must be explicitly enforced at network edges.

**IGMP rate limiting per subscriber:**

$$\text{Max groups per subscriber} \leq G_{max}$$

$$\text{IGMP report rate per subscriber} \leq R_{max} \text{ reports/sec}$$

**PIM neighbor filtering:**

Only accept PIM Hello/Join/Prune from known neighbors:

```
! Only form PIM adjacency with known routers
router pim
 interface TenGigE0/0/0/0
  neighbor-filter PIM-NEIGHBORS
  !
 !
!
ipv4 access-list PIM-NEIGHBORS
 10 permit ipv4 host 10.0.0.2 any
 20 permit ipv4 host 10.0.0.3 any
 30 deny ipv4 any any
!
```

**MSDP SA filtering:**

Only accept SA entries for groups and sources within expected ranges. Reject SAs with sources in RFC 1918 space or unexpected group ranges.

**SSM security advantage:**

SSM inherently validates the source because the receiver specifies (S,G):

$$\text{SSM}: \text{join}(S, G) \implies \text{only accept from } S$$

With ASM: $\text{join}(*, G) \implies \text{accept from any source}$ (requires separate source filtering).

---

## See Also

- bgp, mpls, ospf, is-is, bng, ipv4, ipv6

## References

- [RFC 7761 — Protocol Independent Multicast - Sparse Mode (PIM-SM)](https://www.rfc-editor.org/rfc/rfc7761)
- [RFC 4607 — Source-Specific Multicast for IP](https://www.rfc-editor.org/rfc/rfc4607)
- [RFC 6388 — Label Distribution Protocol Extensions for P2MP and MP2MP LSPs](https://www.rfc-editor.org/rfc/rfc6388)
- [RFC 4875 — Extensions to RSVP-TE for P2MP TE LSPs](https://www.rfc-editor.org/rfc/rfc4875)
- [RFC 3618 — Multicast Source Discovery Protocol (MSDP)](https://www.rfc-editor.org/rfc/rfc3618)
- [RFC 6513 — Multicast in MPLS/BGP IP VPNs](https://www.rfc-editor.org/rfc/rfc6513)
- [RFC 6514 — BGP Encodings and Procedures for Multicast in MPLS/BGP IP VPNs](https://www.rfc-editor.org/rfc/rfc6514)
- [RFC 3376 — Internet Group Management Protocol, Version 3 (IGMPv3)](https://www.rfc-editor.org/rfc/rfc3376)
- [RFC 3810 — Multicast Listener Discovery Version 2 (MLDv2)](https://www.rfc-editor.org/rfc/rfc3810)
- [RFC 4601 — Protocol Independent Multicast - Sparse Mode (PIM-SM) — Original](https://www.rfc-editor.org/rfc/rfc4601)
