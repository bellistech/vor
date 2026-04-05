# Advanced IS-IS — Protocol Internals, Convergence, and Segment Routing Theory

> *IS-IS has survived and thrived for four decades not because of its original design, but because of its TLV extensibility model — a forward-looking architectural decision that allowed the protocol to absorb IPv6, traffic engineering, segment routing, and flex-algo without ever breaking backward compatibility. Understanding IS-IS at depth means understanding how a 1980s OSI protocol became the preferred IGP for the world's largest service provider networks.*

---

## 1. IS-IS PDU Format

IS-IS runs directly on Layer 2 (ISO CLNP, protocol ID 0x83) rather than on top of IP. This is both its greatest strength (independence from the network layer it routes for) and its most commonly misunderstood property.

### Common Header (All PDUs)

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| Intradomain Routing Protocol Discriminator (0x83)  | Len Ind |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| Version/Proto | ID Length    |  PDU Type     |  Version      |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|    Reserved   | Max Area Addr|
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

- **Intradomain Routing Protocol Discriminator:** Always 0x83 (identifies IS-IS)
- **Length Indicator:** Length of the fixed header
- **PDU Type:** Identifies the specific PDU (IIH, LSP, CSNP, PSNP)
- **ID Length:** System ID length (0 = default 6 bytes, used universally)

### IIH (IS-IS Hello) PDU

IIH PDUs perform three functions: neighbor discovery, adjacency formation, and adjacency maintenance.

```
PDU Types:
  15 (0x0F): L1 LAN IIH
  16 (0x10): L2 LAN IIH
  17 (0x11): Point-to-Point IIH

IIH Fixed Fields (after common header):
+--------------------+-------------------------------------------+
| Circuit Type       | 1=L1 only, 2=L2 only, 3=L1/L2            |
| Source ID          | 6-byte System ID of sender                |
| Holding Timer      | Seconds before declaring neighbor dead    |
| PDU Length         | Total length of PDU                       |
| Priority           | DIS election priority (LAN only, 0-127)   |
| LAN ID             | System ID + pseudonode ID (LAN only)      |
| Local Circuit ID   | Local interface identifier (P2P only)     |
+--------------------+-------------------------------------------+

Variable TLVs carried in IIH:
  TLV 1:   Area Addresses
  TLV 6:   IS Neighbors (MAC address, LAN only — 3-way handshake)
  TLV 8:   Padding (fill to interface MTU)
  TLV 10:  Authentication
  TLV 129: Protocols Supported (0xCC=IPv4, 0x8E=IPv6)
  TLV 232: IPv6 Interface Address
  TLV 240: Point-to-Point Three-Way Adjacency (P2P only)
```

### Three-Way Handshake (P2P, RFC 5303)

The three-way handshake on point-to-point links prevents false adjacency formation:

```
State Machine:
  DOWN -> INITIALIZING -> UP

Router A                          Router B
  |                                  |
  |--- IIH (state=DOWN) ----------->|  B sees A, moves to INIT
  |<-- IIH (state=INIT, nbr=A) ----|  A sees B knows A, moves to UP
  |--- IIH (state=UP, nbr=B) ----->|  B sees A is UP, moves to UP
  |                                  |
  Both adjacencies now UP
```

Without three-way handshake, a unidirectional link could result in one router believing the adjacency is up while the other does not, causing traffic blackholing.

### LSP (Link State PDU)

The LSP is the core data structure — it carries all topology and reachability information.

```
PDU Types:
  18 (0x12): L1 LSP
  20 (0x14): L2 LSP

LSP Fixed Fields:
+--------------------+-------------------------------------------+
| PDU Length         | Total length                              |
| Remaining Lifetime | Seconds until LSP expires (max 65535)     |
| LSP ID             | Source ID (6) + Pseudonode (1) + Frag (1) |
| Sequence Number    | 32-bit, monotonically increasing          |
| Checksum           | Fletcher checksum over LSP content         |
| P/ATT/LSPDBOL/IS  | Partition, Attached, Overload, IS Type    |
+--------------------+-------------------------------------------+

LSP ID Format: SSSSSS.PP-FF
  S = System ID (6 bytes)
  P = Pseudonode ID (0 = router itself, >0 = DIS pseudonode)
  F = Fragment number (0-255, for LSPs exceeding MTU)
```

Key fields:

- **Sequence Number:** 32-bit unsigned integer. Each re-origination increments by 1. Higher sequence number wins during synchronization. Range: 1 to $2^{32} - 1$ ($4,294,967,295$).
- **Remaining Lifetime:** Counts down from `max-lsp-lifetime` (default 1200s, recommended 65535s). When it reaches 0, the LSP is purged.
- **Overload (OL) Bit:** When set, transit traffic avoids this router. The router is still reachable as a destination.
- **ATT Bit:** Set by L1/L2 routers in L1 LSPs. L1-only routers use this to install a default route toward the L1/L2 router.

### Fragment Handling

An LSP cannot exceed the link MTU. When a router has more TLVs than fit in one LSP:

$$\text{Max fragments} = 256 \text{ (fragment 0-255)}$$
$$\text{Max LSP data per fragment} \approx \text{MTU} - \text{header overhead}$$
$$\text{Max total LSP data} = 256 \times (\text{MTU} - \text{overhead})$$

For a 1492-byte MTU with ~27 bytes of fixed header: $256 \times 1465 \approx 366 \text{KB}$ per router. In practice, this is more than sufficient for even the largest router configurations.

### CSNP (Complete Sequence Numbers PDU)

CSNPs synchronize databases between neighbors. They list every LSP in the database with its sequence number and remaining lifetime.

```
PDU Types:
  24 (0x18): L1 CSNP
  25 (0x19): L2 CSNP

CSNP Fixed Fields:
+--------------------+-------------------------------------------+
| PDU Length         | Total length                              |
| Source ID          | System ID + circuit ID (7 bytes)          |
| Start LSP ID       | First LSP ID in this CSNP range           |
| End LSP ID         | Last LSP ID in this CSNP range            |
+--------------------+-------------------------------------------+

Variable: TLV 9 (LSP Entries) — list of (Remaining Lifetime, LSP ID, Seq, Checksum)
```

On LAN segments, the DIS (Designated Intermediate System) sends CSNPs every 10 seconds. On point-to-point links, CSNPs are sent once at adjacency formation (for initial synchronization only).

### PSNP (Partial Sequence Numbers PDU)

PSNPs serve two purposes: requesting missing LSPs and acknowledging received LSPs.

```
PDU Types:
  26 (0x1A): L1 PSNP
  27 (0x1B): L2 PSNP

On P2P links: PSNP = explicit ACK for each received LSP
On LAN segments: PSNP = request for LSPs the router is missing
  (The DIS CSNP implicitly ACKs on LANs)
```

### Database Synchronization Flow

```
LAN synchronization:
  1. DIS sends CSNP listing all LSPs in its database
  2. Non-DIS routers compare CSNP entries against their own database
  3. For missing LSPs: router sends PSNP requesting them
  4. DIS (or any router with the LSP) floods the requested LSPs
  5. Repeat every 10 seconds (CSNP interval)

P2P synchronization:
  1. Adjacency forms (three-way handshake)
  2. Both routers exchange CSNPs (complete database summary)
  3. Each router sends PSNPs for LSPs it is missing
  4. Missing LSPs are transmitted
  5. Each received LSP is individually ACKed with a PSNP
```

---

## 2. Dijkstra SPF Computation

IS-IS uses Dijkstra's Shortest Path First algorithm to compute the shortest path tree from the local router to all destinations.

### Algorithm Pseudocode

```
function DIJKSTRA(source, graph):
    // Initialize
    dist[source] = 0
    for each vertex v in graph:
        if v != source:
            dist[v] = INFINITY
        prev[v] = UNDEFINED

    // Priority queue of (distance, vertex)
    PQ = priority_queue()
    PQ.insert(0, source)
    visited = empty_set()

    while PQ is not empty:
        (d, u) = PQ.extract_min()

        if u in visited:
            continue
        visited.add(u)

        for each neighbor v of u:
            if v in visited:
                continue
            new_dist = dist[u] + metric(u, v)
            if new_dist < dist[v]:
                dist[v] = new_dist
                prev[v] = u
                PQ.insert(new_dist, v)

    return dist[], prev[]
```

### Complexity Analysis

| Implementation | Time Complexity | Space Complexity |
|:---------------|:---------------|:----------------|
| Array (naive) | $O(V^2)$ | $O(V)$ |
| Binary heap | $O((V + E) \log V)$ | $O(V)$ |
| Fibonacci heap | $O(V \log V + E)$ | $O(V)$ |

Where $V$ = number of vertices (routers + pseudonodes) and $E$ = number of edges (links).

In practice, IS-IS implementations use binary heaps. For a network with 1000 routers and 5000 links:

$$T = O((1000 + 5000) \times \log_2 1000) = O(6000 \times 10) = O(60000)$$

Modern routers compute full SPF in under 10ms for networks of this scale.

### Two-Phase SPF

IS-IS computes SPF in two phases:

**Phase 1 — Shortest Path Tree:** Run Dijkstra on the IS-IS topology (TLV 22 / Extended IS Reachability) to determine the shortest path and next-hop to every router.

**Phase 2 — Prefix Attachment:** Walk the SPF tree and attach IP prefixes (TLV 135 / Extended IP Reachability) advertised by each router, inheriting the next-hop from Phase 1.

This two-phase approach is significant for optimization: if only a prefix changes (not the topology), only Phase 2 needs to rerun — this is Partial Route Computation (PRC).

### Equal-Cost Multipath (ECMP)

When Dijkstra finds multiple paths with identical cost to a destination:

$$\text{If } \text{dist}[v] \text{ via } u_1 = \text{dist}[v] \text{ via } u_2, \text{ install both next-hops}$$

IS-IS supports ECMP natively. The number of ECMP paths is typically limited by implementation (commonly 8, 16, or 32 parallel paths).

---

## 3. TLV Extensibility Model

The TLV (Type-Length-Value) encoding is the architectural decision that has made IS-IS the most extensible routing protocol in existence.

### Design Principles

1. **Forward compatibility:** A router that does not understand a TLV simply ignores it. No protocol version change is needed.
2. **Nesting:** TLVs can contain sub-TLVs, enabling hierarchical extensions without consuming top-level type space.
3. **Independent evolution:** New features (SR, flex-algo, TE) are added as new TLV types or sub-TLVs, requiring no changes to the base protocol machinery.

### Comparison with OSPF Extensibility

| Aspect | IS-IS TLVs | OSPF LSA Types |
|:-------|:-----------|:---------------|
| Extension mechanism | Add new TLV type (trivial) | New LSA type or Opaque LSA |
| Backward compat | Unknown TLVs ignored | Unknown LSAs flooded but not processed |
| Nesting | Sub-TLVs within TLVs | Limited (Opaque LSA sub-TLVs) |
| Address family | Single protocol, multi-AF TLVs | Separate OSPFv2 and OSPFv3 |
| Protocol versioning | None needed | OSPFv2 vs OSPFv3 split |
| SR integration | TLV 135 sub-TLV 3 (prefix-SID) | Requires OSPFv2 Extended Prefix LSA (type 7/8) |

IS-IS added IPv6 support (RFC 5308) by defining three new TLVs (232, 233, 235). OSPF required an entirely new protocol version (OSPFv3, RFC 5340) with different packet formats, different LSA types, and a separate implementation.

### TLV Type Allocation

```
TLV types are managed by IANA:
  1-127:    Standard TLVs (most commonly used)
  128-199:  IP-specific extensions
  200-255:  Vendor/experimental and newer standards

Sub-TLV types are scoped to their parent TLV:
  Sub-TLV 3 in TLV 135 = Prefix-SID
  Sub-TLV 3 in TLV 22  = Admin Group (completely different meaning)
```

---

## 4. Convergence Optimization

IS-IS convergence is the time from failure detection to forwarding plane update. The total convergence time is:

$$T_{convergence} = T_{detect} + T_{LSP\_gen} + T_{flood} + T_{SPF} + T_{RIB/FIB}$$

### Partial Route Computation (PRC)

When a change affects only reachability (prefix added/removed/metric changed) without modifying the topology:

- **Full SPF:** Recompute Dijkstra + reattach all prefixes. Cost: $O((V + E) \log V)$
- **PRC:** Skip Dijkstra, only reprocess affected prefixes. Cost: $O(P)$ where $P$ = number of changed prefixes

PRC is triggered when:
- A prefix TLV (135/232) changes but the IS reachability TLV (22/222) does not
- A router's metric to a prefix changes but its adjacency metrics remain the same

### Incremental SPF (iSPF)

When the topology changes but only a small portion of the network is affected:

- **Full SPF:** Recompute from scratch. Cost: $O((V + E) \log V)$
- **iSPF:** Only recompute the affected portion of the SPF tree. Cost: $O((V' + E') \log V')$ where $V'$ and $E'$ are the affected vertices and edges

iSPF maintains the previous SPF tree and identifies the "affected area" — the subtree rooted at the point of change. Only that subtree is recomputed.

### SPF Throttle Timers

```
Exponential backoff prevents SPF storms during instability:

        |<-- initial -->|<-- secondary -->|<-------- max -------->|
Event 1: [___50ms___][SPF]
Event 2:              [______200ms______][SPF]
Event 3:                                 [________400ms________][SPF]
Event 4:                                                        [_______800ms_______][SPF]
...until max-wait (5000ms) is reached
...timer resets to initial-wait after quiet period
```

The three-parameter model (`initial-wait`, `secondary-wait`, `max-wait`) provides:
- Fast initial response (50ms) for the first failure
- Increasing backoff (doubling secondary-wait) for rapid subsequent events
- A ceiling (max-wait) to bound worst-case convergence

### LSP Generation Throttle

Similar exponential backoff applies to LSP generation:
- Prevents a flapping interface from generating excessive LSPs
- Each LSP re-origination increments the sequence number
- Rapid re-origination wastes bandwidth and triggers SPF on all routers

### IS-IS Flooding Optimization

Flooding is the most expensive operation in link-state protocols. Each LSP must reach every router in the area.

**Standard flooding on a full mesh of N routers:**

$$\text{LSP copies flooded} = N \times (N - 1)$$

For 50 routers in a full mesh: $50 \times 49 = 2,450$ copies of each LSP.

**Mesh group optimization:**

Routers in the same mesh group do not reflood LSPs received from other mesh group members:

$$\text{LSP copies with mesh group} = N + (N - 1) = 2N - 1$$

For 50 routers: $2 \times 50 - 1 = 99$ copies. A 96% reduction.

**Flooding reduction protocol:** Some implementations support dynamic flooding reduction (draft-ietf-lsr-dynamic-flooding) where a subset of routers forms a "flooding topology" and only those routers participate in flooding. Other routers receive LSPs but do not reflood.

---

## 5. Multi-Area Design Theory

IS-IS uses a two-level hierarchy (L1 and L2) analogous to OSPF's area/backbone model but with important differences.

### Level Architecture

```
                    L2 Backbone
         +------+     +------+     +------+
         |L1/L2 |-----|L1/L2 |-----|L1/L2 |
         |  R1  |     |  R2  |     |  R3  |
         +--+---+     +--+---+     +--+---+
            |             |             |
     Area 49.0001   Area 49.0002   Area 49.0003
         +--+---+     +--+---+     +--+---+
         | L1   |     | L1   |     | L1   |
         |  R4  |     |  R5  |     |  R6  |
         +------+     +------+     +------+
```

### Key Differences from OSPF Areas

| Aspect | IS-IS | OSPF |
|:-------|:------|:-----|
| Backbone | L2, no area ID constraint | Area 0, must be contiguous |
| Border router | L1/L2 router (runs both levels) | ABR (interfaces in multiple areas) |
| Inter-area routing | L1 uses default via ATT bit | Stub areas use default; normal areas get Type 3 LSAs |
| Area boundary | Per-link (each interface in one area) | Per-router (router can be in multiple areas) |
| Virtual links | Not needed (backbone is logical) | Required to connect discontiguous Area 0 |
| Level transition | On the L1/L2 router | On the ABR |

### Why IS-IS L2 Does Not Require Contiguity Hacks

In OSPF, Area 0 must be contiguous because inter-area routes transit through it. A discontiguous Area 0 requires virtual links.

In IS-IS, the L2 backbone is formed by all L1/L2 and L2-only routers. Since IS-IS area boundaries are on links (not routers), an L1/L2 router participates fully in both the L1 area and the L2 backbone. The L2 topology is independent of the L1 area topology, and there is no structural requirement for L2 contiguity beyond basic reachability.

### Route Leaking Theory

Default behavior: L1/L2 routers redistribute all L1 routes into L2 automatically. L2 routes are NOT redistributed into L1 — instead, L1-only routers follow the ATT bit default route.

Route leaking (L2-to-L1) breaks this model intentionally to:
- Enable L1-only routers to make optimal path decisions for specific L2 prefixes
- Avoid suboptimal routing through the nearest L1/L2 router when a better path exists through a different L1/L2 router
- Support traffic engineering within L1 areas that depends on knowledge of L2 topology

The `up/down bit` in TLV 135 prevents routing loops: a prefix leaked from L2 to L1 has the down bit set, preventing it from being re-leaked back from L1 to L2.

---

## 6. IS-IS vs. OSPF Deep Comparison

### Protocol Fundamentals

| Property | IS-IS | OSPF |
|:---------|:------|:-----|
| Standards body | ISO 10589, adopted by IETF | IETF (RFC 2328 / RFC 5340) |
| Runs on | Layer 2 (CLNP, protocol 0x83) | Layer 3 (IP protocol 89) |
| Address format | NET (NSAP-style, e.g., 49.0001.0000.0000.0001.00) | IP-based (router-id, area-id) |
| Authentication | TLV 10 (HMAC-MD5, HMAC-SHA) | AuType field + crypto auth |
| Neighbor discovery | IIH PDUs (multicast to 0180.C200.0014/15) | Hello packets (multicast to 224.0.0.5/6) |
| DR/DIS election | Preemptive (new higher priority wins immediately) | Non-preemptive (existing DR stays) |
| DR/DIS failure | No wait timer, immediate re-election | 40s dead interval before re-election |
| LSA/LSP flooding | Reliable (P2P: PSNP ACK; LAN: CSNP implicit) | Reliable (explicit ACK per LSA) |
| MTU sensitivity | IIH padded to MTU (detects mismatch at adjacency) | MTU mismatch can cause hidden database issues |
| Metric width | 6-bit (narrow) or 32-bit (wide) | 16-bit (OSPFv2) or 24-bit (OSPFv3) |

### Why Service Providers Prefer IS-IS

1. **Transport independence:** IS-IS does not depend on IP. A misconfigured IP address does not break IS-IS adjacency. OSPF running on IP means an IP issue can cascade into routing failure.

2. **Simpler area design:** No virtual links, no NSSA complexity, no requirement for Area 0 contiguity. L1/L2 is cleaner than the OSPF area hierarchy.

3. **Better extensibility:** The TLV model makes it trivial to add new features. OSPF required a new protocol version for IPv6; IS-IS added it with three TLVs.

4. **Faster DIS convergence:** IS-IS DIS election is preemptive and immediate. OSPF DR election is non-preemptive — a higher-priority router does not take over from an existing DR until it fails.

5. **Proven scale:** IS-IS has been the IGP of choice for tier-1 ISPs (AT&T, Verizon, Level3/Lumen) for over two decades.

---

## 7. IS-IS for SR-MPLS

Segment Routing with MPLS data plane (SR-MPLS) uses IS-IS to distribute segment identifiers (SIDs) alongside topology information.

### Prefix-SID Distribution

The Prefix-SID sub-TLV (type 3) within Extended IP Reachability (TLV 135) carries:

```
Prefix-SID Sub-TLV (type 3):
  Flags:        R(re-advertisement), N(node-SID), P(no-PHP), E(explicit-null), V(value), L(local)
  Algorithm:    0 (SPF), 128-255 (flex-algo)
  SID/Index:    If V=0: index into SRGB (label = SRGB_base + index)
                If V=1: absolute label value
```

### SRGB (Segment Routing Global Block)

The SRGB is a contiguous range of MPLS labels reserved for prefix-SIDs:

$$\text{Label} = \text{SRGB\_base} + \text{SID\_index}$$

Default SRGB: 16000-23999 (8000 labels). All routers in the SR domain should use the same SRGB range for operational simplicity, though this is a recommendation, not a requirement.

With different SRGBs, each router computes a different label for the same SID index:

| Router | SRGB Base | SID Index 1 | Label |
|:-------|:----------|:------------|:------|
| R1 | 16000 | 1 | 16001 |
| R2 | 16000 | 1 | 16001 |
| R3 | 20000 | 1 | 20001 |

R3's different SRGB means every router must swap labels at the boundary — additional forwarding complexity.

### Adjacency-SID Distribution

Adj-SIDs are distributed in Extended IS Reachability (TLV 22), sub-TLV 32 (P2P) or 33 (LAN):

```
Adj-SID Sub-TLV (type 32):
  Flags:     F(address-family), B(backup), V(value), L(local), S(set), P(persistent)
  Weight:    For ECMP/UCMP load balancing
  SID/Label: Locally significant MPLS label
```

Adj-SIDs are locally significant (unlike prefix-SIDs which are domain-wide). They enable strict explicit routing: a packet steered through a series of adj-SIDs follows an exact path regardless of IGP metrics.

### Topology-Independent Loop-Free Alternate (TI-LFA)

IS-IS computes backup paths using TI-LFA (RFC 8400):

1. Compute post-convergence SPF (assuming the failed link/node is removed)
2. Find the backup next-hop that provides loop-free forwarding
3. If the backup path requires a detour, encode it as a stack of SIDs (prefix-SID or adj-SID)
4. Pre-install the backup path in the FIB with the SID stack

The SID stack depth determines the protection capability:
- **Link protection:** Usually 0-1 additional SIDs
- **Node protection:** Usually 1-2 additional SIDs
- **SRLG protection:** May require 2-3 additional SIDs

---

## 8. IS-IS for SRv6

SRv6 (Segment Routing over IPv6) uses IS-IS to distribute SRv6 SIDs, which are encoded as IPv6 addresses rather than MPLS labels.

### SRv6 Locator Advertisement

IS-IS advertises SRv6 locators via TLV 135 (or TLV 232 for IPv6) with the SRv6 SID Structure sub-TLV:

```
SRv6 SID format:
  +------------------+----------+-----------+----------+
  | Locator          | Function | Argument  | (unused) |
  | (network prefix) | (action) | (params)  |          |
  +------------------+----------+-----------+----------+
  |<--- Locator Block --->|<-- Function --->|
  |<-------------- 128 bits (IPv6 address) ----------->|
```

### SRv6 End.X SID (Adjacency)

```
IS-IS distributes End.X SIDs (cross-connect function) as adjacency
information, analogous to MPLS adj-SIDs:

  End.X SID: Forward packet to specific neighbor
  End.DT4:   Decapsulate and lookup in IPv4 VRF
  End.DT6:   Decapsulate and lookup in IPv6 VRF
  End.DX4:   Decapsulate and forward to IPv4 CE
```

### Comparison: SR-MPLS vs. SRv6 in IS-IS

| Aspect | SR-MPLS | SRv6 |
|:-------|:--------|:-----|
| SID encoding | 20-bit MPLS label | 128-bit IPv6 address |
| SID distribution | TLV 135 sub-TLV 3 | TLV 135/232 with SRv6 sub-TLVs |
| Max SID stack | Limited by label stack depth (typically 3-5) | Limited by IPv6 extension header (theoretically larger) |
| Encapsulation overhead | 4 bytes per label | 16 bytes per SID |
| Hardware support | Mature (all platforms) | Newer (requires SRv6-capable ASICs) |
| Network programming | Push/swap/pop labels | SRH (Segment Routing Header) |

---

## Prerequisites

- is-is, ospf, segment-routing, mpls, ipv6, dijkstra-algorithm, graph-theory

## References

- [ISO 10589 — Intermediate System to Intermediate System Intra-Domain Routing Protocol](https://www.iso.org/standard/30932.html)
- [RFC 1195 — Use of OSI IS-IS for Routing in TCP/IP and Dual Environments](https://www.rfc-editor.org/rfc/rfc1195)
- [RFC 5120 — Multi Topology Routing in IS-IS](https://www.rfc-editor.org/rfc/rfc5120)
- [RFC 5303 — Three-Way Handshake for IS-IS Point-to-Point Adjacencies](https://www.rfc-editor.org/rfc/rfc5303)
- [RFC 5304 — IS-IS Cryptographic Authentication](https://www.rfc-editor.org/rfc/rfc5304)
- [RFC 5305 — IS-IS Extensions for Traffic Engineering](https://www.rfc-editor.org/rfc/rfc5305)
- [RFC 5308 — Routing IPv6 with IS-IS](https://www.rfc-editor.org/rfc/rfc5308)
- [RFC 5310 — IS-IS Generic Cryptographic Authentication](https://www.rfc-editor.org/rfc/rfc5310)
- [RFC 8400 — Extensions to RSVP-TE for LSP Egress Protection](https://www.rfc-editor.org/rfc/rfc8400)
- [RFC 8667 — IS-IS Extensions for Segment Routing](https://www.rfc-editor.org/rfc/rfc8667)
- [RFC 9350 — IGP Flexible Algorithm](https://www.rfc-editor.org/rfc/rfc9350)
- [RFC 9352 — IS-IS Extensions to Support Segment Routing over IPv6](https://www.rfc-editor.org/rfc/rfc9352)

---

*IS-IS was designed in 1987 for a protocol suite (OSI) that lost the protocol wars. That it became the IGP of choice for the internet's largest networks — routing IPv4, IPv6, MPLS, SR-MPLS, and SRv6 — is a testament to the power of getting the extensibility model right. The TLV is IS-IS's secret weapon, and it has not yet run out of ammunition.*
