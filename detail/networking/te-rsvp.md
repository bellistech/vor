# The Engineering of RSVP-TE — Signaling, State, and Scalability

> *RSVP-TE transforms a connectionless IP network into a circuit-like infrastructure with explicit paths and guaranteed bandwidth, but the cost is per-hop state that fundamentally limits scalability.*

---

## 1. RSVP-TE Message Format

### Message Structure

Every RSVP message consists of a common header followed by a variable number of objects:

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|  Vers | Flags |  Msg Type     |       RSVP Checksum           |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|  Send_TTL     |  (Reserved)   |       RSVP Length              |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

- **IP Protocol 46** — RSVP runs directly over IP, not TCP or UDP
- **Vers:** 1 (current version)
- **Msg Type:** 1=PATH, 2=RESV, 3=PathErr, 4=ResvErr, 5=PathTear, 6=ResvTear

### Object Format

Each object has a 4-byte header:

```
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|         Length (bytes)         |  Class-Num    |   C-Type      |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                    Object contents ...                         |
```

### Key TE Extensions (RFC 3209)

| Object | Class-Num | Purpose |
|--------|-----------|---------|
| LABEL_REQUEST | 19 | Request downstream label allocation |
| LABEL | 16 | Carry allocated label in RESV |
| EXPLICIT_ROUTE (ERO) | 20 | Specify desired path |
| RECORD_ROUTE (RRO) | 21 | Record actual path and labels |
| SESSION_ATTRIBUTE | 207 | Carry tunnel name, setup/hold priority, flags |
| FAST_REROUTE | 205 | Signal FRR requirements |

### ERO Sub-Object Types

- **Type 1:** IPv4 prefix sub-object (strict or loose bit)
- **Type 2:** IPv6 prefix sub-object
- **Type 32:** AS number sub-object (for inter-AS TE)
- **L bit = 0:** Strict hop (must be directly connected)
- **L bit = 1:** Loose hop (routing protocol fills in intermediate hops)

---

## 2. Signaling State Machine

### PATH Processing (Downstream)

```
INGRESS                    TRANSIT                    EGRESS
  |                          |                          |
  |--- PATH (ERO, TSpec) --->|--- PATH (ERO', TSpec) -->|
  |                          |                          |
  |    [Create PSB]          |    [Create PSB]          | [Create PSB]
  |    [Allocate ERO hop]    |    [Remove self from ERO]| [Process label req]
  |                          |                          |
```

**PSB (Path State Block):** Created at each hop, stores:
- SESSION object (tunnel endpoint, tunnel ID, extended tunnel ID)
- SENDER_TEMPLATE (ingress address, LSP ID)
- SENDER_TSPEC (traffic parameters)
- Previous hop address (for RESV routing)
- ERO (remaining hops)

### RESV Processing (Upstream)

```
INGRESS                    TRANSIT                    EGRESS
  |                          |                          |
  |<-- RESV (Label, RRO) ---|<-- RESV (Label, RRO) ---|
  |                          |                          |
  | [Create RSB]             | [Create RSB]             | [Create RSB]
  | [Install label in LFIB]  | [Install swap entry]     | [Allocate label]
  | [LSP is UP]              | [Program forwarding]     | [Send RESV upstream]
```

**RSB (Reservation State Block):** Created at each hop, stores:
- SESSION and FILTER_SPEC (identify the reservation)
- FLOWSPEC (confirmed bandwidth)
- Label mapping (incoming label -> outgoing label)
- Next hop (for RESV forwarding)

### State Machine Transitions

```
              PATH received
                  |
                  v
        +-------------------+
        |   INITIAL STATE   |
        +-------------------+
                  |
           PATH processed,
           PSB created
                  |
                  v
        +-------------------+
        | PATH STATE EXISTS |<-------- PATH refresh
        +-------------------+
                  |
           RESV received
                  |
                  v
        +-------------------+
        |   LSP ACTIVE      |<-------- RESV refresh
        +-------------------+
                  |
        PathTear / timeout
                  |
                  v
        +-------------------+
        |   STATE REMOVED   |
        +-------------------+
```

### Reservation Styles

| Style | Description | Use Case |
|-------|-------------|----------|
| FF (Fixed Filter) | One reservation per sender | Default RSVP-TE |
| SE (Shared Explicit) | Multiple senders share one reservation | Make-before-break |
| WF (Wildcard Filter) | Shared reservation for all senders | Not used in TE |

---

## 3. Soft-State Refresh Overhead Analysis

### The Refresh Problem

Every RSVP-TE router must send a PATH refresh for every LSP it originates and a RESV refresh for every LSP it reserves resources for. The refresh rate is:

$$R_{messages} = \frac{2 \times N_{LSP}}{T_{refresh}}$$

Where:
- $R_{messages}$ = refresh messages per second per router
- $N_{LSP}$ = number of LSPs traversing the router
- $T_{refresh}$ = refresh interval (default 30 seconds)
- Factor of 2: one PATH refresh + one RESV refresh per LSP

### Worked Examples

| LSPs | Refresh Period | Messages/sec | Messages/min |
|:---:|:---:|:---:|:---:|
| 100 | 30s | 6.7 | 400 |
| 1,000 | 30s | 66.7 | 4,000 |
| 10,000 | 30s | 666.7 | 40,000 |
| 50,000 | 30s | 3,333.3 | 200,000 |

At 10,000 LSPs, a transit router processes **667 RSVP messages per second** just for refreshes.

### Cleanup Timeout

State is removed after missing $K$ consecutive refreshes:

$$T_{cleanup} = (K + 0.5) \times 1.5 \times T_{refresh}$$

Default: $K = 3$, so $T_{cleanup} = 3.5 \times 1.5 \times 30 = 157.5$ seconds.

In practice, implementations often simplify to $T_{cleanup} = K \times T_{refresh} = 3 \times 30 = 90$ seconds.

### Summary Refresh Reduction (RFC 2961)

Summary refresh replaces per-LSP PATH/RESV messages with a compact MESSAGE_ID list:

$$R_{summary} = \frac{N_{LSP} \times S_{ID}}{T_{refresh} \times MTU}$$

Where $S_{ID}$ = 8 bytes per MESSAGE_ID entry. For 10,000 LSPs on a 1500-byte MTU:

$$R_{summary} = \frac{10000 \times 8}{30 \times 1500} = \frac{80000}{45000} \approx 1.78 \text{ packets/second}$$

Compared to 667 messages/second without summary refresh. This is a **375x reduction** in message count.

### Reliable Delivery

Without summary refresh, a lost refresh triggers state timeout after 90 seconds. With MESSAGE_ID ACK:
- Each message gets a unique MESSAGE_ID
- Receiver ACKs with MESSAGE_ID_ACK
- If no ACK received, sender retransmits with exponential backoff
- No state loss from transient packet drops

---

## 4. FRR PLR/MP Computation

### Facility Backup Topology

```
                   Protected Link
     PLR ========================= Protected Node
      |                                  |
      |  Bypass Tunnel                   |
      +--------> B1 --------> B2 ------>MP
                                    (Merge Point)
```

### NHOP vs NNHOP Bypass

**NHOP (Next-Hop) Bypass** — protects against link failure only:
- MP = the next hop of the protected link
- Bypass goes around the link, arriving at the same node
- Does NOT protect against node failure of the next hop

**NNHOP (Next-Next-Hop) Bypass** — protects against node failure:
- MP = the node after the protected node
- Bypass goes around both the link AND the node
- Required for full node protection
- PLR must know the path beyond the next hop (uses RRO from PATH/RESV)

### MP Selection Algorithm

1. PLR examines the RRO of the protected LSP to identify downstream nodes
2. For NHOP: MP = next hop in ERO/RRO
3. For NNHOP: MP = node after next hop in ERO/RRO
4. PLR computes a path to MP that avoids the protected link/node (CSPF with exclusion)
5. PLR signals the bypass tunnel to MP

### Label Stack at PLR During FRR

```
Before failure (normal forwarding):
  [Transport Label for protected LSP]
  [Inner labels (VPN, etc.)]

After failure (FRR active):
  [Bypass Tunnel Label]           <- outer, for bypass LSP
  [Transport Label for protected LSP]  <- preserved, MP uses this to merge
  [Inner labels (VPN, etc.)]
```

The MP pops the bypass tunnel label and finds the original transport label, allowing seamless merge back onto the original path.

### One-to-One Backup (DETOUR)

Each protected LSP gets a dedicated detour path:

- PLR signals DETOUR object in PATH message containing (PLR address, avoid-node address)
- Each transit node independently computes its own detour
- More state than facility backup: $N_{detour} = N_{LSP} \times N_{PLR}$
- Facility backup: $N_{bypass} = N_{links}$ (shared across all LSPs)

### Protection Coverage

$$Coverage = \frac{N_{protected\_LSPs}}{N_{total\_LSPs}} \times 100\%$$

Facility backup can achieve 100% coverage with $N_{links}$ bypass tunnels. One-to-one requires $N_{LSP}$ detour paths per PLR, making it $O(N_{LSP})$ vs $O(N_{links})$ for facility backup.

---

## 5. Make-Before-Break SE Style

### Shared Explicit Reservation

In SE (Shared Explicit) style, the old and new LSPs belong to the same SESSION but have different LSP IDs (in SENDER_TEMPLATE). The key property:

**On links common to both old and new LSPs, bandwidth is shared, not doubled.**

### MBB Sequence

```
Time  Action                              State
----  ------                              -----
T0    Old LSP active (LSP-ID=1)           BW reserved on old path
T1    Ingress signals new PATH (LSP-ID=2) New PSB created at each hop
      with SE style, same SESSION
T2    New RESV received at ingress        Both LSPs active
      New LSP programmed in LFIB          Shared BW on common links
T3    Traffic moved to new LSP            Old LSP still has state
T4    Ingress sends PathTear for old LSP  Old state removed
T5    Only new LSP remains                MBB complete
```

### Bandwidth Accounting with SE

On a link shared by both old and new LSPs:

$$BW_{reserved} = \max(BW_{old}, BW_{new})$$

Not $BW_{old} + BW_{new}$. This prevents admission control failure during re-optimization.

### Why SE Style Matters

Without SE style (using FF style), re-optimization would require:
1. Tear down old LSP (traffic disruption)
2. Signal new LSP (admission may fail if bandwidth was freed and re-allocated)
3. Risk of traffic loss during the gap

SE style eliminates both the traffic gap and the double-booking problem.

---

## 6. RSVP-TE Scaling Challenges

### State Per Router

Each transit router maintains:

$$S_{total} = N_{LSP} \times (S_{PSB} + S_{RSB})$$

Where $S_{PSB} \approx 500$ bytes and $S_{RSB} \approx 300$ bytes (implementation dependent).

| LSPs | Memory (approx) |
|:---:|:---:|
| 1,000 | 800 KB |
| 10,000 | 8 MB |
| 100,000 | 80 MB |
| 1,000,000 | 800 MB |

Memory is manageable, but the real bottleneck is **control plane CPU** for refresh processing.

### Convergence Time

After a failure, every affected LSP must be re-signaled end-to-end:

$$T_{reconvergence} = T_{detection} + T_{notification} + N_{affected} \times T_{signal}$$

Where:
- $T_{detection}$: BFD (ms) or Hello timeout (seconds)
- $T_{notification}$: PathErr propagation time
- $T_{signal}$: per-LSP re-signaling time (PATH + RESV round-trip)

With FRR, local repair is immediate ($T_{FRR} \approx T_{detection} + 10ms$), but global repair still takes full re-signaling time.

### Scaling Limits

| Factor | Impact | Mitigation |
|--------|--------|------------|
| Refresh overhead | CPU-bound at ~50K LSPs | Summary refresh (RFC 2961) |
| State memory | 800 bytes/LSP/hop | Acceptable up to ~100K |
| Failure convergence | Re-signal all affected LSPs | FRR for local, headend re-optimization |
| CSPF computation | Full TE-DB SPF per tunnel | Batch CSPF, timer-based reoptimization |
| IGP flooding | TE LSA/TLV for every BW change | Threshold-based flooding |

### The Fundamental Problem

RSVP-TE maintains **per-flow state at every hop**. This is the opposite of IP's stateless forwarding paradigm. The total state in the network is:

$$S_{network} = \sum_{i=1}^{N_{routers}} N_{LSP_i} \times S_{per\_LSP}$$

For a full-mesh of TE tunnels across $N$ routers:

$$N_{LSP_{total}} = N \times (N - 1)$$

And each LSP traverses on average $H$ hops, so total state entries:

$$S_{entries} = N \times (N-1) \times H$$

For 100 routers with average 5 hops: $100 \times 99 \times 5 = 49,500$ state entries per router in the core.

---

## 7. Comparison with SR-TE (Stateful vs Stateless)

### Architectural Difference

| Property | RSVP-TE | SR-TE |
|----------|---------|-------|
| Forwarding state | Per-flow at every hop | Encoded in packet header (label stack) |
| Signaling protocol | RSVP-TE (per-hop, soft-state) | None (or PCE for computation) |
| Bandwidth reservation | Native (FLOWSPEC in RESV) | External (PCE + bandwidth accounting) |
| FRR mechanism | Facility backup / detour | TI-LFA (topology independent) |
| Path encoding | ERO in signaling, labels in LFIB | Segment list in packet header |
| Scalability | O(N_LSP) state per router | O(N_segments) global, no per-flow state |
| MBB | SE style in RSVP | Instantaneous (change segment list) |
| P2MP | Native (RFC 4875) | Replication via ingress replication or tree-SID |

### State Comparison

RSVP-TE state at a transit router:
$$S_{RSVP} = N_{LSP} \times S_{per\_LSP}$$

SR-TE state at a transit router:
$$S_{SR} = N_{prefix\_SIDs} + N_{adjacency\_SIDs} \approx N_{nodes} + N_{links}$$

SR-TE state is **independent of the number of tunnels/policies**. This is the fundamental scalability advantage.

### When RSVP-TE Still Wins

1. **Hard bandwidth guarantees:** RSVP-TE reserves bandwidth at each hop; SR relies on external controllers
2. **Admission control:** RSVP-TE natively rejects LSPs that exceed available bandwidth
3. **Existing deployments:** Mature, widely deployed, well-understood operational model
4. **Inter-vendor interop:** RSVP-TE has decades of interop testing; SR-TE is newer

### Migration Path

Many networks run RSVP-TE and SR-TE simultaneously:
- SR-TE for new services (scalable, simple)
- RSVP-TE for existing tunnels with hard bandwidth guarantees
- Gradual migration as SR-TE bandwidth management matures (PCE-based)

---

## 8. P2MP Tree Construction

### Signaling Model

P2MP RSVP-TE (RFC 4875) extends P2P signaling:

- **P2MP SESSION:** Identified by (destination=P2MP ID, tunnel ID, extended tunnel ID)
- **S2L sub-LSP:** Source-to-Leaf, each leaf has its own sub-LSP within the P2MP session
- **Sub-Group Originator ID:** Identifies the ingress for each S2L

### Tree Construction Process

```
         Ingress (Root)
           /    \
          /      \
        R1        R2        <- Bifurcation points
        |        / \
        |       /   \
       L1      L2    L3     <- Leaves (egress nodes)
```

1. Ingress sends separate PATH messages for each S2L sub-LSP (to L1, L2, L3)
2. At bifurcation points (R1, R2), the router detects multiple S2Ls for the same P2MP session
3. RESV messages from leaves merge at bifurcation points
4. Bifurcation point allocates a single upstream label for the merged branch
5. Data replication occurs at bifurcation points (not at ingress for shared branches)

### Grafting and Pruning

**Grafting** (adding a leaf):
- Ingress sends a new PATH for the new S2L sub-LSP
- Only new branches are signaled; existing branches are unaffected
- Bifurcation point adds the new branch to its replication state

**Pruning** (removing a leaf):
- Ingress sends PathTear for the specific S2L sub-LSP
- Only the branch to the pruned leaf is torn down
- Bifurcation point removes the branch from replication

### P2MP Scaling

The number of S2L sub-LSPs per P2MP session equals the number of leaves:

$$N_{S2L} = N_{leaves}$$

Total state in the network for a P2MP tree:

$$S_{P2MP} = \sum_{i=1}^{N_{nodes\_in\_tree}} N_{S2L\_through\_i} \times S_{per\_S2L}$$

For a balanced binary tree with $L$ leaves and depth $D = \log_2(L)$:
- Root processes $L$ S2L sub-LSPs
- Each level halves the count
- Total state entries: $L + L/2 + L/4 + \ldots + 1 = 2L - 1$

This is $O(L)$, which is optimal — you cannot build a tree to $L$ leaves with fewer than $O(L)$ forwarding entries.

### P2MP vs Ingress Replication

| Method | Replication Point | Bandwidth | State |
|--------|-------------------|-----------|-------|
| P2MP RSVP-TE | Bifurcation nodes | Optimal (no duplication on shared links) | Per-S2L at each node |
| Ingress Replication | Ingress only | $N_{leaves} \times BW$ on ingress link | P2P LSP per leaf |

P2MP saves bandwidth but adds complexity. For small leaf counts ($<10$), ingress replication is simpler. For large multicast groups, P2MP is essential.

---

## See Also

- mpls, ospf, is-is, bgp, bfd

## References

- [RFC 3209 — RSVP-TE: Extensions to RSVP for LSP Tunnels](https://www.rfc-editor.org/rfc/rfc3209)
- [RFC 4090 — Fast Reroute Extensions to RSVP-TE](https://www.rfc-editor.org/rfc/rfc4090)
- [RFC 2961 — RSVP Refresh Overhead Reduction](https://www.rfc-editor.org/rfc/rfc2961)
- [RFC 4875 — Extensions to RSVP-TE for P2MP TE LSPs](https://www.rfc-editor.org/rfc/rfc4875)
- [RFC 3473 — GMPLS Signaling RSVP-TE Extensions](https://www.rfc-editor.org/rfc/rfc3473)
- [RFC 5063 — Extensions to GMPLS RSVP Graceful Restart](https://www.rfc-editor.org/rfc/rfc5063)
- [RFC 8402 — Segment Routing Architecture](https://www.rfc-editor.org/rfc/rfc8402)
