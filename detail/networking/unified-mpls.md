# The Mathematics of Unified MPLS — Scaling Multi-Domain Networks

> *Seamless MPLS solves the fundamental scaling problem of per-domain MPLS by replacing O(N) ABR state with hierarchical BGP-LU label stitching, trading a flat IGP for a structured label hierarchy that scales logarithmically with network depth.*

---

## 1. Why Per-Domain MPLS Fails at Scale

### The ABR Bottleneck Problem

In traditional per-domain MPLS, every Area Border Router must maintain VPN state for all services traversing the boundary. For a network with $S$ services and $A$ ABRs:

$$\text{ABR VPN state} = S \times R_{vpn}$$

Where $R_{vpn}$ is the number of VPN routes per service. With thousands of VPN customers and hundreds of thousands of routes, ABRs become memory and CPU bottlenecks.

### The IGP Scaling Wall

A single flat IGP domain has practical limits:

| IGP | Practical Node Limit | LSA/LSP Overhead |
|:---:|:---:|:---|
| OSPF | ~500 routers per area | Full LSDB flood on any topology change |
| IS-IS L2 | ~1,000 routers | CSNP/PSNP overhead, SPF computation |

SPF computation complexity is $O(N \log N + E)$ where $N$ = nodes, $E$ = edges. For a flat domain with 1,000 nodes and average degree 4:

$$\text{SPF time} \propto 1000 \times \log(1000) + 4000 = 1000 \times 9.97 + 4000 \approx 13,970 \text{ operations}$$

Splitting into 5 areas of 200 nodes each:

$$\text{SPF time per area} \propto 200 \times \log(200) + 800 = 200 \times 7.64 + 800 \approx 2,328 \text{ operations}$$

A **6x reduction** in per-node SPF computation cost.

### The Label Space Problem

LDP allocates one label per IGP prefix. In a flat domain of $N$ nodes, each advertising $L$ loopbacks:

$$\text{Labels per router} = N \times L$$

For 1,000 nodes with 1 loopback each: 1,000 labels. For 5,000 nodes: 5,000 labels. The 20-bit MPLS label space (1,048,576) is not the constraint; the FIB programming rate and memory are.

---

## 2. Seamless MPLS Label Stack Operations

### The Label Hierarchy

Unified MPLS constructs a hierarchical label stack where each domain contributes one label layer:

$$\text{Stack depth} = D + 1$$

Where $D$ = number of domain boundaries crossed, and $+1$ is the service label.

For a 3-tier network (access → aggregation → core → aggregation → access):

$$\text{Max stack depth} = 4 \text{ domain boundaries} + 1 \text{ service label} = 5$$

### Label Operations at Each Hop

**Within a domain (LDP or RSVP-TE):**
- Transit LSR: **swap** outermost label
- Penultimate hop: **pop** outermost label (PHP)

**At a domain boundary (ABR with BGP-LU):**
- **Pop** incoming domain's transport label (or receive via PHP)
- **Swap** BGP-LU label to next-domain's BGP-LU label
- **Push** next-domain's transport label (LDP or RSVP-TE)

### Packet Walk: Access PE to Access PE

Consider a packet from PE1 (access domain A) to PE2 (access domain B), crossing aggregation and core:

```
PE1 (ingress):
  Push VPN label (from BGP VPNv4)
  Push BGP-LU label for PE2 loopback
  Push LDP label for Aggr-ABR-A (local domain transport)

Access Domain A transit:
  Swap LDP label at each LSR
  PHP: pop LDP label at penultimate hop

Aggr-ABR-A:
  Receive packet with BGP-LU label on top
  Swap BGP-LU label (local → remote)
  Push RSVP-TE label for Core-ABR-A (aggregation transport)

Aggregation Domain A transit:
  Swap RSVP-TE label at each LSR
  PHP: pop RSVP-TE label

Core-ABR-A:
  Swap BGP-LU label
  Push RSVP-TE label for Core-ABR-B (core transport)

Core transit:
  Swap RSVP-TE label

Core-ABR-B:
  Pop/swap RSVP-TE, swap BGP-LU, push aggregation transport label

... symmetric through Aggregation B and Access B ...

PE2 (egress):
  Pop BGP-LU label
  Pop VPN label → VRF lookup → forward to CE
```

### MTU Impact

Each MPLS label adds 4 bytes. Maximum overhead:

$$\text{MPLS overhead} = 4 \times \text{stack depth}$$

| Stack Depth | Overhead | Required MTU (for 1500B IP) |
|:---:|:---:|:---:|
| 2 | 8 bytes | 1508 |
| 3 | 12 bytes | 1512 |
| 4 | 16 bytes | 1516 |
| 5 | 20 bytes | 1520 |

**Design rule:** Set core and aggregation links to MTU 9216 (jumbo frames) to avoid fragmentation with deep label stacks.

---

## 3. BGP-LU Inter-Area Stitching

### How Label Stitching Works

At each ABR, BGP-LU performs label stitching:

1. ABR receives BGP-LU route from domain B: prefix $P$, label $L_{remote}$, next-hop $NH_{remote}$
2. ABR allocates local label $L_{local}$ for prefix $P$
3. ABR installs LFIB entry: incoming $L_{local}$ → swap to $L_{remote}$, push transport label toward $NH_{remote}$
4. ABR re-advertises to domain A: prefix $P$, label $L_{local}$, next-hop-self

### The Stitching Chain

For $K$ ABRs between source PE and destination PE:

$$\text{LFIB entries per ABR} = \text{number of remote PE loopbacks}$$

The total label-swap operations for inter-domain traffic:

$$\text{Label operations} = K \text{ (BGP-LU swaps)} + \sum_{i=1}^{K+1} H_i \text{ (intra-domain swaps)}$$

Where $H_i$ is the hop count within domain $i$.

### Next-Hop-Self Behavior

When ABR sets next-hop-self on BGP-LU:

- The advertising ABR becomes the BGP next-hop
- Upstream routers resolve the ABR's loopback via local IGP + LDP/RSVP
- This creates the label hierarchy: local transport label (to ABR) + BGP-LU label (for stitching)

**Without next-hop-self:** The original PE's loopback remains the next-hop, which is unreachable via the local IGP. Traffic black-holes.

---

## 4. Label Allocation Modes

### Per-Prefix Allocation

One label per FEC (prefix). Used by BGP-LU for transport.

$$\text{Labels consumed} = N_{prefixes}$$

**Advantage:** Direct label-to-destination mapping; no additional lookup at egress.
**Disadvantage:** Consumes more label space.

### Per-VRF Allocation

One label per VRF on the egress PE. Used by VPNv4/VPNv6 for service labels.

$$\text{Labels consumed} = N_{VRFs}$$

**Advantage:** Minimal label consumption.
**Disadvantage:** Egress PE must perform a full IP lookup in the VRF table after popping the label.

### Per-CE Allocation

One label per CE neighbor per VRF.

$$\text{Labels consumed} = \sum_{v=1}^{N_{VRFs}} CE_v$$

Where $CE_v$ is the number of CE neighbors in VRF $v$.

**Advantage:** Egress PE can forward directly to the CE without an IP lookup.
**Disadvantage:** More labels than per-VRF.

### Comparison

| Mode | Label Count | Egress Lookup | Use Case |
|:---|:---:|:---|:---|
| Per-prefix | $N_{prefixes}$ | None (label maps to FEC) | BGP-LU transport |
| Per-VRF | $N_{VRFs}$ | Full IP lookup in VRF | Scaled L3VPN |
| Per-CE | $\sum CE_v$ | Direct to CE interface | Low-latency services |

---

## 5. Convergence Analysis

### Failure Scenarios and Recovery Times

**Intra-domain link/node failure:**
- Detection: BFD (3x 50ms = 150ms typical)
- Recovery: IGP reconvergence + LDP/RSVP FRR
- RSVP-TE FRR (facility backup): 50ms switchover
- LDP LFA: 50ms switchover
- Total: **< 200ms** with pre-computed backup paths

**Inter-domain ABR failure:**
- Detection: BFD or BGP holdtimer expiry
- Recovery: BGP-LU reconvergence (BGP scanner + bestpath + advertisement)
- Without BGP PIC: 1-5 seconds (full BGP withdrawal, re-advertisement, re-installation)
- With BGP PIC: **< 200ms** (pre-computed backup label stack, swap to backup on failure detection)

### BGP PIC (Prefix Independent Convergence)

BGP PIC pre-programs a backup forwarding path in the FIB:

$$\text{Convergence time}_{PIC} = T_{detect} + T_{FIB\_swap}$$

Where:
- $T_{detect}$ = BFD detection time (~150ms)
- $T_{FIB\_swap}$ = hardware table pointer swap (~10-50ms)

**Without PIC:**

$$\text{Convergence time}_{no\_PIC} = T_{detect} + T_{BGP\_process} + T_{bestpath} \times N_{prefixes} + T_{FIB\_program} \times N_{prefixes}$$

For 100,000 VPN prefixes:
- $T_{BGP\_process}$ = 500ms (withdrawal processing)
- $T_{bestpath}$ = 1us per prefix = 100ms
- $T_{FIB\_program}$ = 10us per prefix = 1000ms
- Total: ~1.75 seconds

**PIC improvement: ~10x faster convergence.**

### Best-External Advertisement

Without best-external, when an ABR's primary path fails, iBGP clients must wait for:
1. ABR withdraws the route
2. Other ABRs advertise alternative
3. Clients run bestpath selection

With best-external, the ABR pre-advertises the second-best path so clients already have the backup:

$$\text{Failover time} = T_{detect} + T_{local\_bestpath}$$

No dependency on remote BGP re-advertisement.

---

## 6. Scaling Mathematics

### Label Space Analysis

20-bit MPLS label: $2^{20} = 1,048,576$ possible labels.

Reserved ranges: 0-15 (special purpose). Usable: 16 to 1,048,575 = 1,048,560 labels.

For a network with:
- $A$ access domains, each with $N_a$ nodes
- $G$ aggregation domains, each with $N_g$ nodes
- 1 core domain with $N_c$ nodes

Labels consumed per access PE:
$$L_{PE} = N_a \text{ (local LDP)} + \sum_{j \neq \text{local}} N_{a_j} \text{ (BGP-LU for remote access PEs)}$$

Labels consumed per ABR:
$$L_{ABR} = N_{local} \text{ (local LDP)} + \sum_{\text{all remote}} N_i \text{ (BGP-LU)}$$

### RIB/FIB Sizing

**Access PE RIB:**

| Component | Entries |
|:---|:---|
| Local IGP routes | $N_a \times L$ (nodes times loopbacks/links) |
| BGP-LU routes | $\sum N_{a_j}$ (all remote PE loopbacks) |
| VPNv4 routes | Customer routes in local VRFs |

**ABR RIB:**

| Component | Entries |
|:---|:---|
| Local IGP routes | $N_{local}$ |
| Adjacent domain IGP | $N_{adjacent}$ |
| BGP-LU routes | Total PE loopbacks across all domains |
| No VPN routes | ABRs do not carry VPNv4 (key scaling benefit) |

**Key scaling property:** ABRs carry only loopback prefixes via BGP-LU, not VPN routes. If there are 10,000 PEs and 1,000,000 VPN routes, each ABR carries 10,000 BGP-LU entries instead of 1,000,000 VPN entries.

### Route Reflector Scaling

RR memory for BGP-LU:

$$M_{RR} = N_{clients} \times R_{avg} \times S_{entry}$$

Where:
- $N_{clients}$ = number of BGP-LU clients
- $R_{avg}$ = average routes per client
- $S_{entry}$ = memory per RIB entry (~200-500 bytes)

For a hierarchical RR topology with $H$ levels:

$$N_{sessions\_per\_RR} = \frac{N_{total}}{H}$$

Example: 5,000 PEs with 2-level RR hierarchy:
- Level 1 (access domain RR): ~100 sessions each
- Level 2 (core RR): ~50 sessions (to L1 RRs and ABRs)
- vs. flat RR: 5,000 sessions on one RR

---

## 7. Comparison with Segment Routing

### Unified MPLS vs SR-MPLS

| Aspect | Unified MPLS | SR-MPLS |
|:---|:---|:---|
| Label distribution | LDP + RSVP-TE + BGP-LU | IGP extensions (no LDP/RSVP) |
| Label allocation | Dynamic, per-protocol | Static (SRGB) or dynamic |
| Inter-domain stitching | BGP-LU at ABRs | SR-MPLS with Binding SIDs |
| Protocol complexity | 3 label protocols | 1 protocol (IGP + SR extensions) |
| FRR mechanism | LDP LFA, RSVP FRR | TI-LFA (topology independent) |
| TE capability | RSVP-TE explicit paths | SR-TE with SID lists |
| State in network | Per-flow state (RSVP) | Stateless (source routing) |
| Migration | Brownfield (add BGP-LU) | Greenfield or incremental |

### Label Stack Comparison

**Unified MPLS (5-domain traversal):**
```
[LDP:access][BGP-LU:aggr][RSVP-TE:core][BGP-LU:aggr][LDP:access][VPN]
= 6 labels maximum
```

**SR-MPLS (equivalent path):**
```
[Node-SID:destination][VPN]
= 2 labels (if all domains share SRGB)
```

Or with explicit path:
```
[Node-SID:ABR1][Node-SID:ABR2][Node-SID:dest][VPN]
= 4 labels
```

SR-MPLS achieves flatter label stacks because SIDs are globally significant (within the SRGB range), eliminating the need for per-domain transport labels.

### When to Choose Unified MPLS

- Brownfield networks with existing LDP/RSVP-TE investment
- Platforms that do not support Segment Routing
- Networks requiring RSVP-TE bandwidth reservation (SR-TE bandwidth is controller-based)
- Incremental migration path (enable BGP-LU without disrupting existing MPLS)

### When to Choose Segment Routing

- Greenfield deployments or platform refresh
- Desire to eliminate LDP and RSVP-TE protocol complexity
- Need for TI-LFA (100% topology coverage for FRR)
- SDN/controller-driven traffic engineering
- Simpler operations (fewer protocols, stateless forwarding)

---

## 8. LDP-over-RSVP Design

### Why LDP-over-RSVP

LDP follows the IGP shortest path and cannot perform traffic engineering. RSVP-TE provides explicit path control and bandwidth reservation but requires per-tunnel state.

LDP-over-RSVP combines both:
- RSVP-TE tunnels provide the underlay transport with TE capabilities
- LDP sessions ride over RSVP tunnels using targeted adjacency
- LDP provides the label bindings for VPN next-hops

### Session Scaling

For $N$ PEs needing full-mesh LDP reachability over $T$ RSVP tunnels:

Without LDP-over-RSVP (direct LDP):
$$\text{LDP sessions} = \frac{N(N-1)}{2}$$

With LDP-over-RSVP (targeted sessions over tunnels):
$$\text{LDP sessions} = T \text{ (one per tunnel endpoint pair)}$$

The RSVP tunnels can be summarized or use auto-mesh:
$$T = \frac{N(N-1)}{2} \text{ (full mesh tunnels)}$$

The benefit is not session count reduction but **TE capabilities** (FRR, bandwidth, explicit paths) on the transport, while LDP provides the /32 label bindings.

---

## References

- [RFC 8277 — Using BGP to Bind MPLS Labels to Address Prefixes](https://www.rfc-editor.org/rfc/rfc8277)
- [RFC 3107 — Carrying Label Information in BGP-4](https://www.rfc-editor.org/rfc/rfc3107)
- [draft-ietf-mpls-seamless-mpls — Seamless MPLS Architecture](https://datatracker.ietf.org/doc/draft-ietf-mpls-seamless-mpls/)
- [RFC 5283 — LDP Extension for Inter-Area LSP](https://www.rfc-editor.org/rfc/rfc5283)
- [RFC 3209 — RSVP-TE](https://www.rfc-editor.org/rfc/rfc3209)
- [RFC 8402 — Segment Routing Architecture](https://www.rfc-editor.org/rfc/rfc8402)
