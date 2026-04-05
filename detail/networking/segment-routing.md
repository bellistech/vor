# Segment Routing Deep Dive -- Label Stacks, Network Programming, and Migration

> *Segment Routing encodes forwarding instructions at the source, replacing distributed signaling state with a compact ordered list of segments. The two data planes -- SR-MPLS (label stacks) and SRv6 (IPv6 extension headers) -- share the same architecture but differ fundamentally in encoding, overhead, and programmability.*

---

## 1. SR-MPLS Label Stack Operations

### Label Computation

Every SR-MPLS node advertises a Segment Routing Global Block (SRGB) via IGP extensions. The label for a given prefix SID index is:

$$Label = SRGB_{base} + SID_{index}$$

Because SRGB ranges can differ per node (heterogeneous SRGB), each transit node performs its own computation:

| Node | SRGB Range | SID Index 10 Label |
|:---|:---:|:---:|
| A | 16000-23999 | 16010 |
| B | 20000-27999 | 20010 |
| C | 16000-23999 | 16010 |

Transit node B swaps the incoming label to its own SRGB-based label for the next segment.

### Label Stack Construction

The headend builds a label stack representing the segment list, with the first segment at the top (outermost) and the last segment at the bottom:

```
Packet from headend (path: A -> B -> D -> E):

+------------------+
| Label: 20002     |  <- Segment to B (B's SRGB + index for B)
| S=0              |
+------------------+
| Label: 16004     |  <- Segment to D (D's SRGB + index for D)
| S=0              |
+------------------+
| Label: 16005     |  <- Segment to E (final destination)
| S=1              |  <- Bottom of stack
+------------------+
| IP Payload       |
+------------------+
```

### Transit Processing (Label Swap)

At each transit node, the top label is processed:

1. Pop the top label (the segment for this node)
2. Look up the next label in the stack
3. If the next label corresponds to a directly connected node, forward via that link (CONTINUE)
4. If the next label is a remote node SID, swap to the local SRGB-based label and forward via IGP shortest path

### Penultimate Hop Popping (PHP)

The penultimate node pops the transport label so the egress node receives a plain IP packet (or the inner VPN label). This mirrors traditional MPLS PHP behavior. In SR-MPLS, PHP applies to the last node SID in the stack.

### Adjacency SID Processing

Adjacency SIDs are locally significant (from the SRLB). When a transit node encounters an adjacency SID at the top of the stack:

1. Pop the adjacency SID label
2. Forward the packet out the specific interface identified by that adjacency SID
3. No IGP lookup is performed -- this provides strict path steering

### Stack Depth and MSD

Maximum SID Depth (MSD) limits how many labels a node can push or process. MSD is advertised in IGP:

$$\text{Stack overhead} = 4 \times D \text{ bytes (SR-MPLS)}$$

| MSD Value | Typical Platform |
|:---:|:---|
| 3-5 | Older merchant silicon (e.g., Memory/MEMORY.md Memory/MEMORY.md) |
| 6-10 | Modern merchant silicon (Memory/MEMORY.md Memory/MEMORY.md Memory/MEMORY.md) |
| 10-16 | High-end NPUs (Juniper Trio, Nokia FP4/FP5) |

When the required stack depth exceeds MSD, Binding SIDs (BSIDs) are used to break the path into sub-policies.

---

## 2. SRv6 Network Programming

### The SRv6 SID Structure

An SRv6 SID is a 128-bit IPv6 address with three semantic fields:

```
|<--- Locator (B bits) --->|<- Function (F bits) ->|<- Arguments (A bits) ->|
|<--------------------------- 128 bits ----------------------------------->|
```

- **Locator:** Routed by the IPv6 IGP; identifies the node. Typically a /48 or /64 prefix.
- **Function:** Identifies the behavior (END, END.X, END.DT4, etc.). Allocated from the node's function space.
- **Arguments:** Optional parameters (e.g., VRF ID, flow label). Often zero.

Example breakdown:

```
SID:        fc00:0:1:e000::
Locator:    fc00:0:1::/48      (routed in IGP)
Function:   e000               (END behavior)
Arguments:  ::                 (none)
```

### SRv6 Behaviors (Detailed)

#### END (Endpoint)

The basic transit function. When a node receives a packet where the IPv6 DA matches a local END SID:

```
1. Decrement Segments Left (SL) by 1
2. Copy Segment List[SL] into IPv6 Destination Address
3. Forward based on new DA using standard IPv6 FIB lookup
```

No decapsulation occurs. The packet continues with the SRH intact.

#### END.X (Endpoint with Layer-3 Cross-Connect)

Like END, but instead of an FIB lookup, the packet is forwarded to a specific adjacency (next-hop + interface):

```
1. Decrement SL by 1
2. Copy Segment List[SL] into IPv6 DA
3. Forward via the adjacency associated with this END.X SID
   (no FIB lookup -- strict forwarding)
```

Used for traffic engineering (equivalent to adjacency SID in SR-MPLS).

#### END.DT4 (Endpoint with Decapsulation and IPv4 Table Lookup)

Terminates the SRv6 tunnel and performs an IPv4 lookup in a specific VRF:

```
1. Remove the outer IPv6 header and SRH
2. Extract the inner IPv4 packet
3. Perform an IPv4 FIB lookup in the VRF table associated with this SID
4. Forward based on the VRF lookup result
```

This is the SRv6 equivalent of an MPLS L3VPN PE egress function.

#### END.DT6 (Endpoint with Decapsulation and IPv6 Table Lookup)

Same as END.DT4 but for inner IPv6 packets:

```
1. Remove the outer IPv6 header and SRH
2. Extract the inner IPv6 packet
3. Perform an IPv6 FIB lookup in the specified table/VRF
4. Forward based on the lookup result
```

#### END.B6.ENCAPS (Endpoint Bound to an SRv6 Policy with Encapsulation)

Receives a packet, applies a new SRv6 encapsulation (new outer IPv6 header + SRH), and steers the packet into a bound SRv6 policy:

```
1. Pop the active SID (decrement SL, update DA)
2. Push a new IPv6 header with a new SRH containing the bound policy's segment list
3. Forward based on the new outer DA
```

Used for multi-domain stitching and hierarchical policies. The BSID of the bound policy acts as the entry point.

#### H.Encaps (Headend with Encapsulation)

Not a local SID behavior but a headend operation:

```
1. Push a new outer IPv6 header
2. Insert SRH with the segment list
3. Set DA to Segment List[last] (first segment to process)
4. Set SL to (number of segments - 1)
5. Forward based on DA
```

#### H.Encaps.Red (Reduced Encapsulation)

Optimization where the last SID in the segment list is not included in the SRH. Instead, it is only placed in the DA field. This saves 16 bytes in the SRH.

---

## 3. Segment Routing Header (SRH) Format and Processing

### SRH Wire Format

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| Next Header   | Hdr Ext Len   | Routing Type  | Segments Left |
|               |               |    = 4        |     (SL)      |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| Last Entry    |    Flags      |            Tag                |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                                                               |
|            Segment List[0] (128 bits)                         |
|          (last segment in the path -- processed last)         |
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                                                               |
|            Segment List[1]                                    |
|                                                               |
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|            ...                                                |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                                                               |
|            Segment List[n] (first segment -- processed first) |
|                                                               |
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|            Optional TLVs (HMAC, Padding, etc.)                |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

### Key Fields

| Field | Size | Description |
|:---|:---:|:---|
| Next Header | 8 bits | Protocol of the next header (e.g., 41 for IPv6, 4 for IPv4) |
| Hdr Ext Len | 8 bits | Length of SRH in 8-octet units, not including first 8 octets |
| Routing Type | 8 bits | Always 4 for SRH |
| Segments Left (SL) | 8 bits | Index into segment list of the active segment |
| Last Entry | 8 bits | Index of the last element in the segment list (0-based) |
| Flags | 8 bits | Currently unused, must be 0 |
| Tag | 16 bits | Packet grouping tag (for policy marking) |

### SRH Size Calculation

$$\text{SRH size} = 8 + (16 \times N) \text{ bytes}$$

Where $N$ = number of segments in the list. For a 5-segment SRH:

$$8 + (16 \times 5) = 88 \text{ bytes}$$

Compare to SR-MPLS with 5 labels: $4 \times 5 = 20$ bytes. SRv6 carries 4.4x more overhead per segment but provides 128-bit addressable functions.

### Segment List Ordering

The segment list is encoded in reverse order: Segment List[0] is the *last* segment processed, and Segment List[n] is the *first*. The Segments Left field starts at $n$ and decrements toward 0.

```
Path: H -> A -> B -> C (destination)

Segment List[0] = SID_C  (last segment, processed when SL=0)
Segment List[1] = SID_B
Segment List[2] = SID_A  (first segment, processed when SL=2)

Initial SL = 2
Initial DA = SID_A (= Segment List[2])
```

### Processing at Each Hop

```
At node matching Segment List[SL]:
  1. Execute the function encoded in the SID
  2. SL = SL - 1
  3. DA = Segment List[SL]
  4. Forward packet (FIB lookup or cross-connect per function)

When SL = 0 and the active SID is the final endpoint:
  - Execute the terminal function (e.g., END.DT4 decaps)
  - SRH may be removed or left in place depending on function
```

---

## 4. TI-LFA Computation Algorithm

### Problem Statement

Given a network graph $G = (V, E)$ and a single failure (link or node), compute a backup path from the Point of Local Repair (PLR) to the destination that is loop-free in the post-convergence topology.

### Algorithm Steps

**Step 1: Post-Convergence SPF**

Run SPF on the graph $G' = G \setminus \{failed\_element\}$ to compute the post-convergence shortest paths. The backup path must follow the post-convergence routing.

**Step 2: P-Space and Q-Space**

- **P-Space** of PLR: the set of nodes reachable from the PLR without traversing the failed element.

$$P(PLR) = \{v \in V : SP_{G'}(PLR, v) = SP_G(PLR, v) \text{ and path avoids failure}\}$$

- **Q-Space** of destination D: the set of nodes from which D is reachable without traversing the failed element.

$$Q(D) = \{v \in V : SP_{G'}(v, D) = SP_G(v, D) \text{ and path avoids failure}\}$$

**Step 3: Find Repair Node**

- If $P \cap Q \neq \emptyset$: Direct backup exists. Choose the node in the intersection closest to PLR. Backup path needs at most 1 extra segment (the node SID of the repair node).

- If $P \cap Q = \emptyset$: No single repair node suffices. Find a PQ-pair $(p, q)$ where $p \in P$, $q \in Q$, and there exists a direct link $p \to q$ that does not traverse the failed element. Backup requires 2 extra segments (node SID of $p$ + adjacency SID from $p$ to $q$).

**Step 4: Build Repair Segment List**

| Case | Repair Segments | Stack Depth Added |
|:---|:---|:---:|
| Direct (P-Q node exists) | [Node SID of PQ-node] | 1 |
| Via PQ-link | [Node SID of P-node, Adj SID P->Q] | 2 |
| Via multiple intermediate nodes | [Node SID 1, Adj SID, Node SID 2] | 3+ |

**Step 5: Pre-install Backup Path**

The repair segment list is pre-computed and stored in the FIB. On failure detection (BFD, loss-of-light), the PLR immediately pushes the repair segments onto packets and forwards them -- achieving sub-50ms switchover.

### TI-LFA Coverage

TI-LFA achieves 100% coverage for any single link or node failure in a connected graph. The segment list depth required depends on the topology:

- Ring topologies: typically 1 extra segment
- Grid/mesh topologies: typically 1-2 extra segments
- Sparse topologies: may require 2-3 extra segments

---

## 5. SR Policy and Binding SID (BSID)

### SR Policy Components

An SR Policy is identified by the tuple (headend, color, endpoint):

```
SR Policy {
  Headend:   Node originating the policy
  Color:     32-bit value identifying policy intent
  Endpoint:  Destination address

  Binding SID (BSID):  Local label/SID that represents this policy

  Candidate Paths (ordered by preference):
    Path 1 (preference 200):
      Segment List A: [SID1, SID2, SID3] weight=1
      Segment List B: [SID4, SID5]       weight=1  <- ECMP/WECMP
    Path 2 (preference 100, fallback):
      Segment List C: [SID6, SID7]       weight=1
}
```

### Binding SID (BSID) Mechanics

A BSID is a local SID (from SRLB or SRv6 function space) that acts as an alias for an entire SR Policy. When a packet arrives with the BSID at the top of the stack:

1. Pop the BSID
2. Push the segment list of the active candidate path
3. Forward based on the new top-of-stack label/SID

BSID use cases:

- **Inter-domain stitching:** Domain A steers to BSID at the border, domain B resolves the BSID to its internal SR Policy.
- **Policy indirection:** Routes point to a BSID. The underlying SR Policy (segment list) can change without updating routes.
- **Hierarchical policies:** A top-level policy uses BSIDs of sub-policies as segments.

### Candidate Path Selection

When multiple candidate paths exist:

1. Highest preference value wins (active path)
2. If the active path becomes invalid (segment unreachable), fall back to next preference
3. Within a candidate path, multiple segment lists with weights enable weighted ECMP

$$\text{Traffic share for list } i = \frac{w_i}{\sum_{j} w_j}$$

### Color-Based Steering

BGP color extended community triggers automatic steering into SR Policies:

```
BGP prefix 10.20.0.0/16 advertised with:
  Next-hop: 10.0.0.5
  Color:    100 (low-latency)

Router lookup: SR Policy (headend=self, color=100, endpoint=10.0.0.5)
  -> If exists, steer traffic through this policy
  -> If not, fall back to default path to 10.0.0.5
```

---

## 6. Flex-Algo Metric Types and Computation

### Overview

Flexible Algorithm (flex-algo) allows defining custom routing computations (algorithms 128-255) with:

- Custom metric type
- Custom constraints (affinities, SRLGs)
- Custom calculation type

Each flex-algo produces its own SPF tree and set of prefix SIDs.

### Metric Types

| Metric Type | Value | Description |
|:---|:---:|:---|
| IGP metric | 0 | Standard link cost (default algo 0 metric) |
| Min unidirectional delay | 1 | Delay in microseconds (from IETF performance measurement) |
| TE metric | 2 | Traffic engineering metric (independent of IGP metric) |

### Flex-Algo Definition (FAD)

The FAD is advertised by one or more nodes (typically route reflectors or designated nodes):

```
Flex-Algo Definition {
  Algorithm:        128
  Metric Type:      1 (delay)
  Calculation Type: 0 (SPF)
  Priority:         128
  Constraints:
    Include-any affinity: blue
    Exclude affinity: red
    Exclude SRLG: [100, 200]
}
```

All participating nodes receive the FAD via IGP flooding and run a separate SPF for algorithm 128 using delay metrics, including only links with the "blue" affinity and excluding links with "red" affinity or SRLGs 100/200.

### Prefix SID per Flex-Algo

Each node advertises a separate prefix SID per flex-algo:

```
Node X:
  Algo 0 (default):  prefix SID index 10  -> shortest IGP path
  Algo 128 (delay):  prefix SID index 110 -> lowest delay path
  Algo 129 (TE):     prefix SID index 210 -> TE-optimized path
```

To reach node X via the lowest-delay path, the headend uses SID index 110. No explicit segment list is needed -- the IGP computes the best delay-optimized path automatically.

### Flex-Algo vs Explicit SR-TE

| Aspect | Flex-Algo | Explicit SR-TE |
|:---|:---|:---|
| Path specification | Computed per algo (distributed) | Explicit segment list (headend or PCE) |
| Metric | Single metric per algo | Any combination |
| Constraints | Affinity, SRLG | Full flexibility |
| State overhead | 1 prefix SID per node per algo | Full segment list per policy |
| Use case | Network-wide metric optimization | Per-flow path engineering |

---

## 7. SR-MPLS vs SRv6 Comparison

| Dimension | SR-MPLS | SRv6 |
|:---|:---|:---|
| Data plane | MPLS (label stack) | IPv6 (SRH extension header) |
| Segment encoding | 20-bit label (4 bytes each) | 128-bit IPv6 address (16 bytes each) |
| Overhead per segment | 4 bytes | 16 bytes |
| SRH base overhead | 0 (pure label stack) | 8 bytes (SRH fixed header) |
| 5-segment overhead | 20 bytes | 88 bytes |
| Network programming | Limited (push/swap/pop) | Rich (END, END.X, END.DT4, END.B6.ENCAPS, etc.) |
| Function extensibility | Constrained by label semantics | Unlimited (128-bit function space) |
| Infrastructure requirement | MPLS-capable hardware | IPv6-capable hardware |
| MTU sensitivity | Low (4B/label) | High (16B/SID) -- jumbo frames recommended |
| Deployment maturity | Production-ready, wide vendor support | Growing, Linux native support strong |
| VPN support | Via MPLS L3VPN (RFC 4364) | Via END.DT4/DT6 (native SRv6) |
| Hardware support | Universal (all MPLS ASICs) | Newer ASICs required for SRH processing |
| Interworking | Coexists with LDP, RSVP-TE | Requires IPv6 end-to-end (or gateway) |

### When to Choose Each

**SR-MPLS** is appropriate when:
- Existing MPLS infrastructure is in place
- Hardware does not support SRv6 (older line cards)
- Overhead budget is tight (4 bytes vs 16 bytes per segment)
- Interworking with LDP or RSVP-TE is required during migration

**SRv6** is appropriate when:
- Greenfield IPv6 deployment (data center fabric, 5G transport)
- Network programming flexibility is needed (service chaining, VNF steering)
- Simplified operations desired (single protocol -- IPv6 -- for transport and services)
- The network can accommodate SRH overhead (jumbo frames available)

---

## 8. Migration from LDP and RSVP-TE

### LDP to SR-MPLS Migration

**Phase 1: Parallel Deployment**

1. Enable SR on all nodes with consistent SRGB
2. Advertise prefix SIDs via IGP (IS-IS or OSPF)
3. LDP and SR coexist; both label types are present in the LFIB
4. Configure `segment-routing sr-prefer` so SR labels are preferred over LDP labels when both exist

```
router isis CORE
 segment-routing on
 segment-routing global-block 16000 23999
 segment-routing prefix 10.0.0.1/32 index 1
 # Prefer SR labels over LDP labels
 segment-routing sr-prefer
```

**Phase 2: Validation**

1. Verify SR labels are installed in the forwarding table
2. Confirm TI-LFA is providing backup paths
3. Monitor traffic to ensure SR paths are active
4. Check that all nodes have SR enabled (a single non-SR node breaks the SR LSP through it)

**Phase 3: LDP Removal**

1. Once all traffic flows over SR labels, disable LDP on each interface
2. Remove LDP configuration
3. Verify no LDP labels remain in the LFIB

### RSVP-TE to SR-MPLS Migration

**Phase 1: Mapping**

Map existing RSVP-TE tunnels to SR-TE policies:

| RSVP-TE Concept | SR Equivalent |
|:---|:---|
| Explicit path (ERO) | Segment list (node SIDs + adjacency SIDs) |
| Tunnel interface | SR Policy (color, endpoint) |
| FRR facility backup | TI-LFA (automatic, topology-independent) |
| CSPF | PCE-computed segment list or flex-algo |
| Bandwidth reservation | PCE bandwidth-aware path computation |
| Tunnel autoroute | BGP color-based steering |

**Phase 2: Parallel Operation**

1. Create SR Policies that mirror each RSVP-TE tunnel path
2. Use BSID to represent the SR Policy
3. Gradually steer traffic from RSVP-TE tunnels to SR Policies
4. Enable TI-LFA to replace RSVP-TE FRR

**Phase 3: RSVP-TE Removal**

1. Shut down RSVP-TE tunnels one at a time
2. Verify traffic is using SR Policies (check BSID counters)
3. Remove RSVP-TE configuration from all nodes
4. Remove RSVP-TE bandwidth reservation from links

### Benefits of Migration

| Aspect | Before (LDP/RSVP-TE) | After (SR-MPLS) |
|:---|:---|:---|
| Control plane protocols | IGP + LDP + RSVP-TE | IGP only |
| Transit node state | O(tunnels) or O(FECs) | O(nodes) |
| Failure recovery | FRR (pre-signaled) | TI-LFA (computed) |
| Operational complexity | High (3 protocols) | Low (1 protocol) |
| New tunnel provisioning | Minutes (signaling) | Instant (segment list) |

---

## 9. Summary of Key Formulas

| Formula | Description |
|:---|:---|
| $Label = SRGB_{base} + SID_{index}$ | SR-MPLS label computation |
| $Overhead_{SR-MPLS} = 4 \times D$ bytes | SR-MPLS label stack overhead |
| $Overhead_{SRv6} = 8 + 16 \times N$ bytes | SRv6 SRH total size |
| $\frac{w_i}{\sum w_j}$ | Weighted ECMP traffic share |
| $P \cap Q$ | TI-LFA repair node existence check |
| $MSD \geq D_{required}$ | Stack depth feasibility constraint |

## Prerequisites

- mpls (label switching fundamentals), ipv6 (addressing and extension headers), is-is or ospf (IGP shortest path computation), graph theory (SPF algorithm, P-space/Q-space)

---

*Segment Routing distills decades of MPLS complexity into a single architectural principle: encode the path at the source. SR-MPLS preserves existing hardware investment while eliminating signaling protocols. SRv6 trades overhead for programmability, turning every IPv6 address into a network instruction. The migration path from LDP and RSVP-TE is incremental and reversible -- deploy SR alongside existing protocols, validate, then remove the old signaling plane.*
