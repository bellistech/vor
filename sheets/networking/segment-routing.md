# Segment Routing (SR)

Source routing paradigm where the ingress node encodes the forwarding path as an ordered list of segments (instructions), eliminating per-flow state from transit nodes and unifying traffic engineering, fast reroute, and service chaining into a single architecture.

## Concepts

### Core Architecture

- **Source Routing:** The headend (ingress) selects the path and encodes it in the packet header as a segment list
- **Segment:** An instruction executed by an SR-capable node (forward to node, forward via link, apply service)
- **SID (Segment Identifier):** The numeric or address-based identifier for a segment
- **SRGB (Segment Routing Global Block):** The label range reserved for SR on each node (e.g., 16000-23999)
- **SRLB (Segment Routing Local Block):** Label range for locally significant segments (adjacency SIDs)
- **Controller-free by default:** Paths are computed by the headend using IGP (IS-IS/OSPF) extensions; PCE is optional

### Segment Types

| Type | Scope | Description |
|------|-------|-------------|
| Prefix SID | Global | Identifies a destination prefix; same index across all nodes in the domain |
| Node SID | Global | A prefix SID associated with a node's loopback (most common use case) |
| Adjacency SID | Local | Identifies a specific link (interface) on a node; used for strict path steering |
| Anycast SID | Global | Same SID assigned to multiple nodes providing the same service (load balancing, redundancy) |
| Binding SID (BSID) | Local/Global | Represents an SR Policy; allows hierarchical or inter-domain stitching |
| Service SID | Local | Identifies a VNF or service function (used in service chaining) |

### SID Computation (SR-MPLS)

```
Label = SRGB_base + SID_index

Example:
  Node A SRGB = 16000-23999
  Node X prefix SID index = 10
  Label for X on A = 16000 + 10 = 16010
```

## SR-MPLS (Segment Routing over MPLS Data Plane)

### Overview

- Uses the existing MPLS data plane (20-bit labels, label stack)
- Segments are encoded as MPLS labels pushed onto the label stack
- No LDP or RSVP-TE signaling required; SIDs advertised via IGP (IS-IS or OSPF)
- Incremental deployment: SR-MPLS and LDP can coexist

### IS-IS SR Configuration (FRRouting)

```
router isis CORE
 net 49.0001.0100.0000.0001.00
 is-type level-2-only
 segment-routing on
 segment-routing global-block 16000 23999
 segment-routing node-msd 12
 segment-routing prefix 10.0.0.1/32 index 1

interface eth0
 ip router isis CORE
 isis network point-to-point
```

### OSPF SR Configuration (FRRouting)

```
router ospf
 ospf router-id 10.0.0.1
 segment-routing on
 segment-routing global-block 16000 23999
 segment-routing node-msd 12
 segment-routing prefix 10.0.0.1/32 index 1

interface eth0
 ip ospf area 0
 ip ospf network point-to-point
```

### Show Commands (FRRouting)

```bash
# Display SRGB and node SID
vtysh -c "show isis segment-routing node"

# Show prefix SID mapping
vtysh -c "show isis segment-routing prefix-sids"

# OSPF variant
vtysh -c "show ip ospf segment-routing node"

# MPLS forwarding table with SR labels
vtysh -c "show mpls forwarding-table"
```

## SRv6 (Segment Routing over IPv6)

### Overview

- Uses IPv6 as the data plane; segments are 128-bit IPv6 addresses
- Instructions encoded in the Segment Routing Header (SRH), an IPv6 routing extension header
- Network programming model: each SID encodes a locator, a function, and optional arguments
- No MPLS infrastructure required; runs natively over IPv6

### SRv6 SID Format

```
|<---- Locator ---->|<- Function ->|<-- Arguments -->|
|    B bits          |   F bits     |    A bits       |
|<------------------ 128 bits ---------------------------->|

Example:
  fc00:0:1::     (Locator — identifies the node, routed via IGP)
           :e000 (Function — END, END.X, END.DT4, etc.)
                 (Arguments — optional, e.g., VRF ID)

Full SID: fc00:0:1:e000::
```

### SRv6 Functions (Network Programming)

| Function | Description |
|----------|-------------|
| END | Endpoint: decrement SL, update DA to next SID |
| END.X | Endpoint with L3 cross-connect: forward to specific adjacency |
| END.DT4 | Endpoint with decaps and IPv4 table lookup (L3VPN) |
| END.DT6 | Endpoint with decaps and IPv6 table lookup (L3VPN) |
| END.DX4 | Endpoint with decaps and IPv4 cross-connect |
| END.DX6 | Endpoint with decaps and IPv6 cross-connect |
| END.B6.ENCAPS | Endpoint bound to an SRv6 policy (insert SRH) |
| END.BM | Endpoint bound to an SR-MPLS policy |
| END.S | Endpoint in shared memory (service chaining) |
| T.ENCAPS | Transit with encapsulation (H.Encaps) |

### SRv6 on Linux (iproute2)

```bash
# Add an SRv6 route with segment list
ip -6 route add 2001:db8::/32 encap seg6 mode encap \
  segs fc00:0:2:e000::,fc00:0:3:e000:: dev eth0

# SRv6 local SID table — define a local END function
ip -6 route add fc00:0:1:e000::/64 encap seg6local action End dev eth0

# END.DT4 — decaps and lookup in IPv4 table
ip -6 route add fc00:0:1:d004::/64 encap seg6local action End.DT4 vrftable 100 dev eth0

# END.DT6 — decaps and lookup in IPv6 table
ip -6 route add fc00:0:1:d006::/64 encap seg6local action End.DT6 table 100 dev eth0

# END.X — cross-connect to next-hop via specific interface
ip -6 route add fc00:0:1:c001::/64 encap seg6local action End.X nh6 fe80::1 dev eth1

# Show SRv6 SID table
ip -6 route show type seg6local

# Show SRv6 encap routes
ip -6 route show encap seg6

# Enable SRv6 in sysctl
sysctl -w net.ipv6.conf.all.seg6_enabled=1
sysctl -w net.ipv6.conf.eth0.seg6_enabled=1
```

### Segment Routing Header (SRH)

```
IPv6 Header (Next Header = 43, Routing)
+------------------------------------------+
| Next Header | Hdr Ext Len | Routing Type |
|             |             |    = 4 (SRH) |
+------------------------------------------+
| Segments Left (SL)  | Last Entry        |
+------------------------------------------+
| Flags        | Tag                       |
+------------------------------------------+
| Segment List[0] (128-bit IPv6 address)   |
+------------------------------------------+
| Segment List[1]                          |
+------------------------------------------+
| ...                                      |
+------------------------------------------+
| Segment List[n] (active = SL index)      |
+------------------------------------------+
| Optional TLVs (HMAC, padding, NSH)       |
+------------------------------------------+

Processing:
  1. Node receives packet, DA matches local SID
  2. Execute function for that SID
  3. Decrement Segments Left (SL)
  4. Copy Segment List[SL] into IPv6 DA
  5. Forward based on new DA
```

## SR Policy

### Structure

- **Headend:** The node that imposes the SR Policy
- **Color:** A numeric value (community) that identifies the policy intent (e.g., low-latency, high-bandwidth)
- **Endpoint:** The destination of the policy
- **Binding SID (BSID):** A local label/SID that maps to the policy; traffic steered to BSID follows the policy
- **Candidate Path:** One or more segment lists with preference and weight
- **Segment List:** Ordered list of SIDs defining the explicit path

### SR Policy Example (Conceptual)

```
SR Policy:
  Headend:  10.0.0.1
  Color:    100 (low-latency)
  Endpoint: 10.0.0.5
  BSID:     24100

  Candidate Path (preference 200):
    Segment List: [16002, 16004, 16005]
    Weight: 1

  Candidate Path (preference 100, backup):
    Segment List: [16003, 16005]
    Weight: 1
```

### Steering Traffic into SR Policy

```bash
# Linux: steer traffic matching a route via BSID
ip route add 10.10.0.0/16 encap mpls 24100 via 10.1.1.2 dev eth0

# Or via SRv6 policy with segment list
ip -6 route add 2001:db8:10::/48 encap seg6 mode encap \
  segs fc00:0:2::,fc00:0:4::,fc00:0:5:: dev eth0
```

## TI-LFA (Topology Independent Loop-Free Alternate)

### Overview

- Provides sub-50ms convergence for any single link, node, or SRGB failure
- Computes backup paths using the post-convergence SPF result
- Backup path encoded as a repair segment list (pushed on failure)
- "Topology Independent" means it protects against any failure, not just specific topologies

### Configuration (FRRouting IS-IS)

```
router isis CORE
 fast-reroute ti-lfa level-2
 fast-reroute ti-lfa node-protection level-2

interface eth0
 ip router isis CORE
 isis ti-lfa
```

### Show TI-LFA State

```bash
vtysh -c "show isis fast-reroute summary"
vtysh -c "show isis route detail"
```

## SR-TE (Segment Routing Traffic Engineering)

### Concepts

- Explicit paths without signaling (no RSVP-TE state on transit nodes)
- Paths defined as segment lists at the headend
- Supports affinity/color-based steering via BGP color communities
- Bandwidth-aware with controller (PCE) assistance

### Flex-Algo (Flexible Algorithm)

```
# Define algorithm 128 with minimum delay metric
router isis CORE
 flex-algo 128
  advertise-definition
  metric-type delay

# Assign a prefix SID for flex-algo 128
 segment-routing prefix 10.0.0.1/32 index 101 algorithm 128
```

- Algorithms 128-255 are user-defined
- Each flex-algo can use a different metric (IGP, delay, TE) and constraints (affinity, SRLG)
- Nodes compute separate SPFs per flex-algo
- Prefix SIDs per flex-algo provide metric-optimized paths without explicit segment lists

## PCE (Path Computation Element) Integration

### Overview

- PCE computes paths centrally and programs SR Policies on headends via PCEP (RFC 5440)
- PCC (Path Computation Client) is the headend router
- Stateful PCE maintains LSP state; can delegate, update, or initiate SR Policies
- PCEP extensions for SR: RFC 8664

### PCEP Configuration (FRRouting)

```
pce
 address 10.0.0.100
 peer 10.0.0.1
  type external
  sr-capable

router isis CORE
 pce address 10.0.0.1
```

## SR-MPLS vs RSVP-TE vs LDP

| Feature | SR-MPLS | RSVP-TE | LDP |
|---------|---------|---------|-----|
| Per-flow transit state | None | Yes (per tunnel) | Yes (per FEC) |
| Traffic engineering | Yes (segment list) | Yes (explicit path, CSPF) | No |
| Fast reroute | TI-LFA (topology-independent) | FRR (facility/one-to-one) | LFA/rLFA (topology-dependent) |
| Signaling protocol | None (IGP extensions) | RSVP | LDP |
| Scalability | Excellent (O(N) state) | Poor (O(N^2) tunnels) | Good (O(prefixes)) |
| Bandwidth reservation | Via PCE/controller | Native (RSVP) | None |
| Incremental deployment | Yes (coexists with LDP) | Requires full deployment | Requires full deployment |
| Data plane | MPLS | MPLS | MPLS |

## Use Cases

- **5G Transport (X-haul):** Low-latency backhaul/midhaul/fronthaul with flex-algo delay optimization
- **DC Fabric:** Simplified underlay with SRv6, no MPLS in the data center
- **WAN Optimization:** SR-TE policies for application-aware routing across WAN
- **Service Chaining:** SRv6 network programming steers traffic through VNFs without overlay tunnels
- **Multi-domain TE:** Binding SIDs stitch SR Policies across domain boundaries
- **Network Slicing:** Flex-algo partitions the network into virtual topologies per slice

## Tips

- Start with SR-MPLS if you have existing MPLS infrastructure; migrate to SRv6 when IPv6 is pervasive.
- Node SIDs should use consistent indexing across the domain (e.g., index = last octet of loopback).
- SRGB should be identical on all nodes for simplicity, though heterogeneous SRGBs are supported.
- Adjacency SIDs are only needed for strict path steering; most deployments rely on node SIDs and TI-LFA.
- Set MSD (Maximum SID Depth) appropriately; hardware forwarding pipelines have label stack limits (typically 5-12).
- Flex-algo 128 with delay metric is the simplest way to get low-latency paths without explicit TE tunnels.
- SRv6 MTU overhead can be significant: each SID in the SRH is 16 bytes. Plan for jumbo frames on core links.
- BSID simplifies SR Policy management: steer traffic to the BSID and change the policy without updating routes.
- TI-LFA provides 100% topology coverage for single failures; always enable it.
- When migrating from LDP, deploy SR alongside LDP using SR-prefer to gradually shift traffic.

## See Also

- mpls, bgp, is-is, ospf

## References

- [RFC 8402 -- Segment Routing Architecture](https://www.rfc-editor.org/rfc/rfc8402)
- [RFC 8660 -- Segment Routing with the MPLS Data Plane](https://www.rfc-editor.org/rfc/rfc8660)
- [RFC 8986 -- Segment Routing over IPv6 (SRv6) Network Programming](https://www.rfc-editor.org/rfc/rfc8986)
- [RFC 8754 -- IPv6 Segment Routing Header (SRH)](https://www.rfc-editor.org/rfc/rfc8754)
- [RFC 9256 -- Segment Routing Policy Architecture](https://www.rfc-editor.org/rfc/rfc9256)
- [RFC 8667 -- IS-IS Extensions for Segment Routing](https://www.rfc-editor.org/rfc/rfc8667)
- [RFC 8665 -- OSPF Extensions for Segment Routing](https://www.rfc-editor.org/rfc/rfc8665)
- [RFC 8664 -- PCEP Extensions for Segment Routing](https://www.rfc-editor.org/rfc/rfc8664)
- [RFC 8355 -- Resiliency Use Cases in SR-TE](https://www.rfc-editor.org/rfc/rfc8355)
- [FRRouting Segment Routing Documentation](https://docs.frrouting.org/en/latest/isisd.html#segment-routing)
- [Linux SRv6 Implementation](https://segment-routing.org/index.php/Implementation/Linux)
- [IETF SPRING Working Group](https://datatracker.ietf.org/wg/spring/about/)
