# Advanced JunOS MPLS — Deep Dive Theory and Analysis

> In-depth exploration of JunOS MPLS implementation internals: LDP label retention modes, RSVP-TE state machines, CSPF algorithm mechanics, FRR PLR/MP computation, LSP hierarchy theory, admin group bit operations, and SR-MPLS architecture. For JNCIE-SP level understanding.

## 1. JunOS MPLS Implementation Architecture

### 1.1 Control Plane vs Forwarding Plane

JunOS implements MPLS across two distinct planes:

- **Routing Engine (RE):** Runs LDP, RSVP, BGP, and the CSPF algorithm. Computes label bindings, maintains LSP state, and populates `mpls.0` and `inet.3` tables.
- **Packet Forwarding Engine (PFE):** Hardware-based MPLS forwarding using ASIC-programmed label lookup tables derived from `mpls.0`.

The separation means label operations (push/swap/pop) happen at line rate on the PFE while protocol signaling and path computation happen on the RE. The RE pushes the active label forwarding entries to the PFE via the internal RE-PFE link.

### 1.2 MPLS Routing Tables in JunOS

| Table       | Purpose                                                        |
|-------------|----------------------------------------------------------------|
| `mpls.0`    | Maps incoming MPLS labels to next-hop and outgoing label       |
| `inet.3`    | MPLS tunnel endpoints for BGP next-hop resolution              |
| `inet.0`    | Can receive LSP routes via `traffic-engineering bgp-igp`       |

Key distinction: BGP resolves next-hops via `inet.3` by default, which contains MPLS tunnel endpoints. This is why an RSVP-TE or LDP LSP to a PE loopback automatically enables BGP to use MPLS transport without additional configuration.

### 1.3 Label Allocation

JunOS allocates labels from a platform-specific dynamic range (typically starting at label 16 or higher). Label allocation policies:

- **Per-platform label space:** LDP uses a single label space shared across all interfaces (label space ID `0:0`). This is standard for frame-mode MPLS.
- **Per-interface label space:** Not used in standard JunOS frame-mode MPLS but relevant for cell-mode ATM.
- **SRGB (Segment Routing Global Block):** Configured explicitly for SR-MPLS, reserving a contiguous label range for globally significant node/adjacency SIDs.

## 2. LDP Label Distribution and Retention

### 2.1 Liberal vs Conservative Label Retention

LDP defines two label retention modes:

- **Liberal retention (JunOS default):** The router retains all label mappings received from all LDP peers, even if the peer is not the next-hop for that FEC. This pre-populates backup label paths for faster convergence.
- **Conservative retention:** Only retains label mappings from the peer that is the current next-hop for the FEC. Saves memory but slows convergence.

JunOS uses **liberal retention** by default. This means `show ldp database` shows label bindings from all peers, but only the binding matching the IGP next-hop is installed in `mpls.0`.

### 2.2 Downstream Unsolicited (DU) Mode

JunOS LDP operates in **Downstream Unsolicited** mode:

1. Each LSR independently assigns a local label to each FEC (prefix) in its routing table
2. It then advertises this label-FEC binding to all LDP peers without being asked
3. Upon receiving a binding, each peer stores it (liberal retention) but only installs the binding from the current IGP next-hop

This results in immediate label availability when IGP convergence changes the next-hop. The new label is already cached via liberal retention.

### 2.3 Ordered Control vs Independent Control

- **Independent control (JunOS default):** Each LSR assigns labels and creates forwarding entries independently, without waiting for downstream label bindings. This means partial LSPs can exist during convergence.
- **Ordered control:** An LSR only creates a forwarding entry for a FEC if it is the egress for that FEC or it has received a label binding from the next-hop. Guarantees end-to-end LSP establishment but slower.

### 2.4 LDP-IGP Synchronization Mechanics

When LDP-IGP sync is enabled on an interface:

1. Interface comes up with IGP adjacency
2. JunOS sets the IGP metric to maximum (65535 for OSPF, 16777214 for IS-IS) on that interface
3. Traffic avoids the interface because of the high metric
4. LDP session establishes and label bindings are exchanged
5. Once LDP is fully converged, JunOS restores the original IGP metric
6. Traffic can now safely use the MPLS-enabled path

The holddown timer (`holddown-interval`) controls how long to wait for LDP convergence before restoring the metric regardless.

## 3. RSVP-TE State Machine

### 3.1 LSP Establishment Sequence

The RSVP-TE LSP setup follows this state machine:

```
IDLE --> PATH_SENT --> PATH_ACK --> UP
  |          |            |         |
  |          v            v         v
  +---- PATH_ERR    RESV_ERR    DOWN
```

Detailed steps:

1. **Ingress (head-end):** CSPF computes ERO. Sends PATH message downstream with:
   - SESSION object (destination, tunnel ID, extended tunnel ID)
   - SENDER_TEMPLATE (source, LSP ID)
   - EXPLICIT_ROUTE (ERO) computed by CSPF
   - LABEL_REQUEST
   - TSPEC (traffic spec for bandwidth)
   - RECORD_ROUTE (RRO) for path recording
   - FAST_REROUTE (if FRR requested)

2. **Transit LSRs:** Process PATH, update RRO, subtract bandwidth from available, forward downstream. Each transit node records itself in the RRO.

3. **Egress:** Receives PATH, allocates label, sends RESV upstream with:
   - LABEL object (assigned label)
   - RRO (recorded route back)
   - If PHP: sends label 3 (implicit null)

4. **Transit LSRs (reverse):** Receive RESV, allocate label, install forwarding state (incoming label -> swap to downstream label, forward to next-hop), send RESV upstream.

5. **Ingress:** Receives RESV, installs push label, LSP is UP.

### 3.2 State Refresh

RSVP is soft-state: PATH and RESV messages must be refreshed periodically (default 30 seconds). If three consecutive refreshes are missed, the state is torn down.

JunOS supports **summary refresh** (RFC 2961) to reduce refresh overhead:
- Instead of resending full PATH/RESV, a summary refresh message contains message IDs
- Dramatically reduces CPU load on transit routers with thousands of LSPs

### 3.3 Graceful Restart for RSVP

During RE failover (GRES/NSR), RSVP state must be preserved:
- The new RE reads preserved RSVP state from kernel
- Sends Hello messages with restart capability
- Neighbors maintain forwarding state during restart period
- Full RSVP state is recovered without traffic loss

## 4. CSPF Algorithm in JunOS

### 4.1 Constrained Shortest Path First

CSPF extends Dijkstra's SPF with additional constraints:

1. **Build TED (Traffic Engineering Database):** Populated from OSPF-TE (opaque LSAs) or IS-IS-TE (TLVs). Contains:
   - Link bandwidth (maximum, reservable, available per priority)
   - TE metric
   - Admin groups (link colors)
   - SRLG (Shared Risk Link Group)

2. **Prune ineligible links:** Remove links that fail any constraint:
   - Insufficient available bandwidth for requested reservation
   - Missing required admin groups (include-any/include-all)
   - Present excluded admin groups
   - Links in excluded SRLGs

3. **Run Dijkstra on pruned topology:** Find shortest path using either IGP metric or TE metric (if `metric-based-computation` is configured).

4. **Result:** An Explicit Route Object (ERO) — an ordered list of strict hops from ingress to egress.

### 4.2 CSPF Tiebreaking

When multiple equal-cost paths exist after CSPF:

1. **Least-fill:** Prefer path with maximum available bandwidth (default in JunOS)
2. **Most-fill:** Prefer path with minimum available bandwidth (pack LSPs)
3. **Random:** Random selection among equal paths

```
protocols {
    mpls {
        label-switched-path to-PE2 {
            to 10.0.0.2;
            least-fill;     /* default */
        }
    }
}
```

### 4.3 Bandwidth Reservation Priorities

RSVP supports 8 priority levels (0-7, where 0 is highest):
- **Setup priority:** Priority to establish the LSP (can preempt lower priority LSPs)
- **Hold priority:** Priority to maintain the LSP (defended against preemption)
- Rule: setup priority value >= hold priority value (lower number = higher priority)

The TED tracks available bandwidth at each of the 8 priority levels per link, enabling preemption-aware path computation.

## 5. FRR PLR/MP Computation

### 5.1 Terminology

- **PLR (Point of Local Repair):** The router that detects a failure and activates the backup path.
- **MP (Merge Point):** The router where the backup path rejoins the primary path.
- **Protected LSP:** The original RSVP-TE LSP.
- **Bypass LSP (Facility Backup):** Pre-established tunnel from PLR to MP, shared by multiple protected LSPs.
- **Detour LSP (One-to-One Backup):** Dedicated backup path per protected LSP.

### 5.2 Facility Backup (Bypass) Computation

When link-protection or node-link-protection is configured:

1. PLR identifies the protected resource (link or node)
2. PLR computes a bypass LSP path that avoids the protected resource
3. Bypass LSP is signaled via RSVP-TE (with ERO excluding the protected link/node)
4. Upon failure detection:
   - PLR pushes the bypass label onto the protected packet's label stack
   - Traffic is label-stacked: `[bypass-label | original-label | payload]`
   - MP pops the bypass label, continues forwarding with original label

**Node-Link Protection** extends this: the bypass must avoid both the directly connected link AND the downstream node. The MP is two hops downstream from the PLR, not one.

### 5.3 Detour Computation

One-to-one backup:
1. Each PLR along the protected LSP computes its own detour
2. Detour LSP is signaled end-to-end from PLR to the egress
3. Detour ERO avoids the protected link/node
4. Upon failure: PLR swaps to detour label (no label stacking needed)

**Trade-offs:**

| Aspect           | Facility Backup         | Detour (One-to-One)     |
|------------------|------------------------|--------------------------|
| Scalability      | High (shared bypass)   | Low (per-LSP detour)     |
| Label overhead   | Extra label (stacking) | No extra label           |
| Bandwidth aware  | Bypass bandwidth fixed | Per-LSP bandwidth        |
| Setup overhead   | One bypass per link    | One detour per LSP       |

### 5.4 FRR Switchover Timing

JunOS achieves sub-50ms FRR switchover:
- BFD detects failure in ~150ms (3x 50ms intervals) or faster
- Hardware notification of link-down: ~10ms
- PLR label table already programmed with backup path
- Switchover is a forwarding table pointer change in the PFE ASIC

## 6. LSP Hierarchy Theory

### 6.1 Concept

LSP hierarchy allows one LSP (inner) to be tunneled through another LSP (outer):
- **Outer LSP:** Provides transport between two intermediate points
- **Inner LSP:** End-to-end LSP from ingress PE to egress PE

Use cases:
- Large service provider networks with multiple IGP areas/levels
- Inter-area TE where a single RSVP-TE LSP cannot span areas
- Scalability: reduces the number of end-to-end LSPs

### 6.2 Label Stack in Hierarchy

At the ingress of the outer LSP, the packet carries a two-label stack:

```
[outer-transport-label | inner-service-label | payload]
```

- Outer label is swapped/popped along the outer LSP
- At the outer LSP egress (which is a transit for the inner LSP), the outer label is popped, inner label is swapped
- This continues until the packet reaches the inner LSP egress

### 6.3 inet.3 and Recursive Resolution

JunOS uses `inet.3` for recursive next-hop resolution:
1. Inner LSP's next-hop is the egress PE loopback
2. This next-hop resolves in `inet.3` to the outer LSP
3. Forwarding table entry: push inner label, then push outer label

This recursive resolution is automatic when both inner and outer LSPs are signaled.

## 7. Admin Group Bit Operations

### 7.1 Bit Mask Representation

Admin groups are represented as a 32-bit bitmask. Each named group maps to a bit position (0-31):

```
gold   = bit 0 = 0x00000001
silver = bit 1 = 0x00000002
bronze = bit 2 = 0x00000004
```

### 7.2 Constraint Matching

- **include-any:** Bitwise AND between link's admin-group bits and the constraint mask must be non-zero (at least one matching bit).
  ```
  link_bits & include_any_mask != 0  -->  link is eligible
  ```

- **include-all:** Bitwise AND must equal the full constraint mask (all required bits must be set).
  ```
  link_bits & include_all_mask == include_all_mask  -->  link is eligible
  ```

- **exclude:** Bitwise AND must be zero (none of the excluded bits may be set).
  ```
  link_bits & exclude_mask == 0  -->  link is eligible
  ```

### 7.3 Example Calculation

Link has admin-groups: gold, silver (bits 0,1 set = `0x00000003`)

| Constraint                       | Mask         | Calculation            | Result    |
|----------------------------------|--------------|------------------------|-----------|
| include-any [ gold bronze ]      | `0x00000005` | `0x03 & 0x05 = 0x01`  | PASS (!=0)|
| include-all [ gold silver ]      | `0x00000003` | `0x03 & 0x03 = 0x03`  | PASS (==mask)|
| include-all [ gold bronze ]      | `0x00000005` | `0x03 & 0x05 = 0x01`  | FAIL (!=mask)|
| exclude [ bronze ]               | `0x00000004` | `0x03 & 0x04 = 0x00`  | PASS (==0)|
| exclude [ gold ]                 | `0x00000001` | `0x03 & 0x01 = 0x01`  | FAIL (!=0)|

## 8. SR-MPLS (SPRING) in JunOS

### 8.1 Segment Routing Architecture

Segment Routing eliminates per-flow state on transit routers by encoding the path as an ordered list of segments (SIDs) in the packet header.

Two SID types:
- **Node SID (Prefix SID):** Globally significant identifier for a node. Computed as `SRGB_base + index`. All routers in the domain agree on the same SRGB range.
- **Adjacency SID:** Locally significant identifier for a specific link/adjacency. Dynamically allocated from the local label space.

### 8.2 SRGB in JunOS

The SRGB must be identically configured across all SR-capable routers in the domain:

```
routing-options {
    source-packet-routing {
        srgb start-label 16000 index-range 8000;
        /* Labels 16000-23999 reserved for SR */
    }
}
```

If router A has `ipv4-index 100`, its node SID label is `16000 + 100 = 16100` on every router in the domain.

### 8.3 Forwarding Behavior

For a given destination prefix with node SID label L:
- **Ingress:** Pushes label L onto the packet
- **Transit:** If L is the node SID of a remote router, swaps L to the appropriate next-hop label for that destination (standard MPLS swap based on the IGP shortest path)
- **Penultimate hop:** Pops label L (PHP, same as LDP behavior)
- **Egress:** Receives unlabeled packet (after PHP) or pops explicit null

### 8.4 TI-LFA (Topology Independent Loop Free Alternate)

TI-LFA provides sub-50ms FRR for SR-MPLS without RSVP-TE:

1. **Pre-failure computation:** For each destination, compute the post-convergence path (the path that would be used after the failure)
2. **Repair segment list:** Compute a segment list that steers traffic along the post-convergence path from the PLR
3. **Upon failure:** PLR pushes the repair segment list, traffic follows the post-convergence path immediately
4. **After convergence:** Normal SR forwarding resumes, repair segments are no longer needed

Advantages over RSVP FRR:
- No bypass/detour LSP signaling required
- Guaranteed loop-free behavior (topology independent)
- Automatic — no manual bypass configuration
- Scales with number of prefixes, not number of LSPs

### 8.5 SR-MPLS vs LDP in JunOS

| Aspect                | SR-MPLS                         | LDP                              |
|-----------------------|---------------------------------|----------------------------------|
| State on transit      | None (stateless)                | Per-FEC label binding            |
| Signaling protocol    | IGP extensions (OSPF/IS-IS)     | LDP (separate protocol)         |
| FRR mechanism         | TI-LFA                          | LFA / RSVP FRR                  |
| Traffic engineering   | SR-TE (segment lists)           | RSVP-TE                         |
| Label allocation      | Globally significant (SRGB)     | Locally significant             |
| Scalability           | Excellent (no per-flow state)   | Good (per-prefix state)         |
| Transition            | Can coexist with LDP            | Legacy, being replaced by SR    |

## See Also

- junos-l3vpn
- junos-l2vpn
- junos-routing-fundamentals

## References

- RFC 5036 — LDP Specification
- RFC 3209 — RSVP-TE Extensions to RSVP for LSP Tunnels
- RFC 4090 — Fast Reroute Extensions to RSVP-TE
- RFC 2961 — RSVP Refresh Overhead Reduction Extensions
- RFC 8402 — Segment Routing Architecture
- RFC 8667 — IS-IS Extensions for Segment Routing
- RFC 8665 — OSPF Extensions for Segment Routing
- RFC 7490 — Remote LFA (rLFA)
- draft-ietf-rtgwg-segment-routing-ti-lfa — TI-LFA
- Juniper TechLibrary: MPLS Feature Guide
