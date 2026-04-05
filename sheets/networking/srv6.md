# SRv6 (Segment Routing over IPv6)

SRv6 encodes forwarding instructions as IPv6 addresses in the Segment Routing Header, enabling network programming where each SID specifies both a destination and a behavior, eliminating MPLS label dependency and unifying transport, VPN, and service chaining into native IPv6 forwarding.

## SID Structure

### SID Format

```
SRv6 SID = 128-bit IPv6 address with semantic fields:

|<--- Locator (B bits) --->|<-- Function (F bits) -->|<-- Arguments (A bits) -->|
|<------------------------------- 128 bits ---------------------------------->|

Locator: Routable prefix (advertised in IGP), identifies the node
Function: Behavior to execute at the node (END, END.DT4, etc.)
Arguments: Optional per-flow or per-service parameters

Common sizing:
  Locator = /48 (48 bits)     e.g., fc00:0:1::/48
  Function = 16 bits           e.g., e000 (END), d100 (END.DT4)
  Arguments = 64 bits          e.g., flow label, VRF ID

Example:
  SID:       fc00:0:1:e000::
  Locator:   fc00:0:1::/48        (routed by IGP)
  Function:  0xe000               (END behavior)
  Args:      ::                   (none)
```

### Locator Design

```
Locator allocation strategy:

Provider block:   fc00:AAAA::/32           (per-provider allocation from ULA or GUA)
Site/region:      fc00:AAAA:BBBB::/40      (per-site or per-region)
Node:             fc00:AAAA:BBBB:CC::/48   (per-node, routed in IGP)

Example allocation for a 3-site network:
  Site 1:  fc00:0:1::/48  (Node 1)
           fc00:0:2::/48  (Node 2)
  Site 2:  fc00:0:10::/48 (Node 10)
           fc00:0:11::/48 (Node 11)
  Site 3:  fc00:0:20::/48 (Node 20)

The locator is advertised as a regular IPv6 prefix via IS-IS or OSPFv3.
All nodes in the network can reach any locator via standard IPv6 forwarding.
```

## SRv6 Network Programming Behaviors

### Core Behaviors (RFC 8986)

```
Behavior        Description                                    Equivalent
────────────────────────────────────────────────────────────────────────────────
END             Endpoint: update DA, forward via FIB            Node SID (SR-MPLS)
END.X           Endpoint + L3 cross-connect to adjacency       Adjacency SID
END.T           Endpoint + specific IPv6 table lookup           Table lookup
END.DT4         Decaps + IPv4 table lookup (VRF)               L3VPN PE egress (IPv4)
END.DT6         Decaps + IPv6 table lookup (VRF)               L3VPN PE egress (IPv6)
END.DT46        Decaps + IPv4 or IPv6 lookup (dual-stack VRF)  Dual-stack L3VPN PE
END.DX4         Decaps + IPv4 cross-connect (specific NH)      Per-CE L3VPN
END.DX6         Decaps + IPv6 cross-connect (specific NH)      Per-CE L3VPN
END.DX2         Decaps + L2 cross-connect                      L2VPN / VPWS
END.B6.Encaps   Bound SRv6 policy with encapsulation           BSID (SR-MPLS)
END.B6.Encaps.Red  Reduced encapsulation (reuse outer header)  BSID + reduced SRH
END.BM          Bound to SR-MPLS policy                        SRv6-to-MPLS gateway

Headend behaviors (not local SIDs):
H.Encaps        Push outer IPv6 + SRH                          Label push
H.Encaps.Red    Reduced encapsulation                          Optimized push
H.Insert        Insert SRH into existing IPv6 packet           SRH insertion
H.Insert.Red    Reduced SRH insertion                          Optimized insertion
```

### Behavior Processing Details

```
END processing:
  1. Decrement Segments Left (SL) by 1
  2. Copy Segment List[SL] to IPv6 Destination Address
  3. FIB lookup on new DA, forward

END.X processing:
  1. Decrement SL by 1
  2. Copy Segment List[SL] to IPv6 DA
  3. Forward via specific adjacency (no FIB lookup)
  4. Used for traffic engineering (strict path)

END.DT4 processing:
  1. Verify SL == 0 (last segment)
  2. Remove outer IPv6 header and SRH
  3. Extract inner IPv4 packet
  4. IPv4 FIB lookup in the associated VRF
  5. Forward based on VRF lookup

END.DT6 processing:
  1. Verify SL == 0 (last segment)
  2. Remove outer IPv6 header and SRH
  3. Extract inner IPv6 packet
  4. IPv6 FIB lookup in the associated VRF/table
  5. Forward based on lookup

END.B6.Encaps processing:
  1. Pop active SID (decrement SL, update DA)
  2. Push new outer IPv6 header with new SRH
  3. New SRH contains the bound policy's segment list
  4. Forward based on new outer DA
  5. Enables multi-domain stitching
```

## Segment Routing Header (SRH)

### SRH Format (RFC 8754)

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| Next Header   | Hdr Ext Len   | Routing Type=4| Segments Left |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| Last Entry    |    Flags      |        Tag                    |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                                                               |
|            Segment List[0] (128 bits)                         |
|                 (last segment in path)                        |
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                                                               |
|            Segment List[1] (128 bits)                         |
|                                                               |
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                                                               |
|            Segment List[n] (128 bits)                         |
|                 (first segment in path)                       |
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                                                               |
|            Optional TLVs (variable)                           |
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

Key fields:
  Routing Type:  4 (SRH)
  Segments Left: Index into segment list (decremented at each SID)
  Last Entry:    Index of last element in segment list (0-based)
  Segment List:  Ordered in reverse (List[0] = final destination)

Segment list is stored in REVERSE order:
  List[n] = first segment to visit (copied to DA at headend)
  List[0] = last segment (final destination)
  SL starts at n, decremented at each hop
```

### SRH Size Calculation

```
SRH size = 8 bytes (fixed header) + 16 bytes x N (segment list) + TLVs

Examples:
  1 segment:   8 + 16 =  24 bytes
  3 segments:  8 + 48 =  56 bytes
  5 segments:  8 + 80 =  88 bytes
  10 segments: 8 + 160 = 168 bytes

Compare to SR-MPLS:
  5 segments:  5 x 4 = 20 bytes (SR-MPLS) vs 88 bytes (SRv6)
  Ratio: 4.4x overhead for SRv6

With micro-SID (uSID):
  6 micro-SIDs packed into 1 x 128-bit container = 16 bytes
  Equivalent SR-MPLS: 6 x 4 = 24 bytes
  uSID closes the overhead gap significantly
```

## SRv6 TE Policy

### Policy Structure

```
SRv6 TE Policy components:
  - Color:      Identifies the intent (e.g., low-latency, high-bandwidth)
  - Endpoint:   Destination PE IPv6 address
  - BSID:       Binding SID (IPv6 address representing the policy)
  - Candidate paths:  Ordered list of segment lists (preference-based)

Policy = (Color, Endpoint)
Each policy has one or more candidate paths, each with one or more segment lists.
The highest-preference valid candidate path is active.

Example:
  Policy: color=100, endpoint=fc00:0:5::1
  Candidate path 1 (preference 200):
    Segment list: [fc00:0:2:e001::, fc00:0:4:e001::, fc00:0:5:d100::]
  Candidate path 2 (preference 100):
    Segment list: [fc00:0:3:e001::, fc00:0:5:d100::]
```

### IOS-XR SRv6 TE Policy Configuration

```
segment-routing
 traffic-eng
  policy LOW-LATENCY
   color 100 end-point ipv6 fc00:0:5::1
   candidate-paths
    preference 200
     explicit segment-list SL-VIA-NODE2-NODE4
    preference 100
     explicit segment-list SL-VIA-NODE3
   !
  !
  segment-list SL-VIA-NODE2-NODE4
   index 10 srv6 sid fc00:0:2:e001::
   index 20 srv6 sid fc00:0:4:e001::
   index 30 srv6 sid fc00:0:5:d100::
  !
  segment-list SL-VIA-NODE3
   index 10 srv6 sid fc00:0:3:e001::
   index 20 srv6 sid fc00:0:5:d100::
  !
```

## SRv6 L3VPN

### IOS-XR SRv6 L3VPN Configuration

```
! Enable SRv6
segment-routing
 srv6
  locators
   locator MAIN
    prefix fc00:0:1::/48
   !
  !
 !

! IS-IS with SRv6
router isis CORE
 address-family ipv6 unicast
  segment-routing srv6
   locator MAIN
  !
 !
 interface Loopback0
  address-family ipv6 unicast
 !
 interface GigabitEthernet0/0/0/0
  address-family ipv6 unicast
 !

! BGP with SRv6 for L3VPN
router bgp 65000
 address-family vpnv4 unicast
 address-family vpnv6 unicast
 !
 neighbor fc00:0:2::1
  address-family vpnv4 unicast
   encapsulation-type srv6
  address-family vpnv6 unicast
   encapsulation-type srv6
 !
 vrf CUST-A
  rd 65000:100
  address-family ipv4 unicast
   segment-routing srv6
    locator MAIN
    alloc mode per-vrf
   !
   redistribute connected
  !
  address-family ipv6 unicast
   segment-routing srv6
    locator MAIN
    alloc mode per-vrf
   !
   redistribute connected
  !

! VRF definition
vrf CUST-A
 address-family ipv4 unicast
  import route-target 65000:100
  export route-target 65000:100
 address-family ipv6 unicast
  import route-target 65000:100
  export route-target 65000:100
```

### SRv6 L3VPN Forwarding

```
SRv6 L3VPN packet flow (IPv4-over-SRv6):

CE1 (10.1.1.0/24) ---> PE1 ---> [SRv6 core] ---> PE2 ---> CE2 (10.2.2.0/24)

PE1 encapsulation:
  +------------------------------------------+
  | Outer IPv6 Header                        |
  |   Src: fc00:0:1::1 (PE1 loopback)       |
  |   Dst: fc00:0:2:d100:: (PE2 END.DT4)    |
  +------------------------------------------+
  | SRH (optional if single segment)         |
  |   SL=0, Segment List[0]=fc00:0:2:d100:: |
  +------------------------------------------+
  | Inner IPv4 Header                        |
  |   Src: 10.1.1.10, Dst: 10.2.2.20        |
  +------------------------------------------+
  | Payload                                  |
  +------------------------------------------+

PE2 processing (END.DT4):
  1. Match DA fc00:0:2:d100:: to local SID table
  2. Execute END.DT4: decapsulate, IPv4 lookup in VRF CUST-A
  3. Forward to CE2 via VRF route

SID allocation modes:
  per-vrf:  One SID per VRF (END.DT4/DT6) — fewer SIDs, less granularity
  per-ce:   One SID per CE (END.DX4/DX6) — more SIDs, per-CE steering
```

## SRv6 with EVPN

```
SRv6 as transport for EVPN:

EVPN Type 2 (MAC/IP) with SRv6:
  - BGP carries EVPN routes with SRv6 SID as the tunnel endpoint
  - SID function: END.DT2 (L2 table lookup) or END.DX2 (L2 cross-connect)
  - Replaces VXLAN VNI with SRv6 SID for traffic steering

EVPN Type 5 (IP Prefix) with SRv6:
  - Prefix routes carry SRv6 SID (END.DT4/DT6 for VRF lookup)
  - Same as SRv6 L3VPN but using EVPN route type 5

IOS-XR EVPN + SRv6:
  router bgp 65000
   address-family l2vpn evpn
   !
   neighbor fc00:0:2::1
    address-family l2vpn evpn
     encapsulation-type srv6
   !

  evpn
   segment-routing srv6
    locator MAIN
   !
```

## Micro-SID (uSID)

### uSID Concept

```
Problem: SRv6 SIDs are 128 bits each, creating significant overhead.
Solution: Micro-SID compresses multiple SIDs into a single 128-bit container.

Standard SRv6:
  Each SID = 128 bits (16 bytes)
  3 SIDs = 48 bytes + 8 byte SRH header = 56 bytes

Micro-SID (uSID):
  uSID block = 128-bit container
  Each micro-SID = 16 or 32 bits
  Up to 6 x 16-bit uSIDs per container (with 32-bit block prefix)

uSID container format (16-bit uSID variant):
  |<-- Block (32 bits) -->|<-- uSID1 (16b) -->|<-- uSID2 -->|...|<-- uSID6 -->|
  |<-------------------------------- 128 bits -------------------------------->|

Example:
  Block:    fc00:0000:                    (32-bit provider block)
  uSID1:    0001                          (Node 1 - END)
  uSID2:    0002                          (Node 2 - END)
  uSID3:    0005                          (Node 5 - END.DT4)
  Container: fc00:0000:0001:0002:0005:0000:0000:0000

  3 SIDs in 16 bytes (1 container) instead of 56 bytes (3 full SIDs + SRH)

uSID shift operation:
  At each node, the active uSID (leftmost after block) is consumed
  Remaining uSIDs shift left, zeros fill from the right
  When all uSIDs in a container are consumed, move to next container in SRH
```

### IOS-XR uSID Configuration

```
segment-routing
 srv6
  locators
   locator uSID-MAIN
    micro-segment behavior unode psp-usd
    prefix fc00:0:1::/48
   !
  !
 !

! uSID locator in IS-IS
router isis CORE
 address-family ipv6 unicast
  segment-routing srv6
   locator uSID-MAIN
  !
 !
```

## SRv6 vs SR-MPLS Comparison

```
Dimension              SRv6                          SR-MPLS
──────────────────────────────────────────────────────────────────────────────
Data plane             IPv6 (SRH ext. header)        MPLS (label stack)
Segment size           128 bits (16 bytes)           20 bits (4 bytes)
Overhead/segment       16 bytes                      4 bytes
SRH fixed overhead     8 bytes                       0 bytes
5-segment overhead     88 bytes                      20 bytes
Network programming    Rich (20+ behaviors)          Limited (push/swap/pop)
VPN support            END.DT4/DT6 (native)          MPLS L3VPN (RFC 4364)
Infrastructure         IPv6-capable hardware          MPLS-capable hardware
MTU sensitivity        High (jumbo frames advised)    Low (4B/label)
Extensibility          128-bit function space         Constrained by labels
Hardware maturity      Newer ASICs required           Universal MPLS support
Linux native support   Strong (kernel SRv6)           Limited (MPLS kernel)
uSID compression       Closes overhead gap            N/A
Service chaining       Native (SID = instruction)     Requires NSH or chaining
Interworking           IPv6 end-to-end or gateway     Coexists with LDP/RSVP-TE
```

## Flex-Algo with SRv6

```
Flex-Algo assigns algorithm IDs (128-255) to compute separate topologies
based on custom metrics (delay, TE metric) and constraints (affinity, SRLG).

SRv6 + Flex-Algo: Each node advertises a separate SRv6 SID per flex-algo.

Example:
  Node X SIDs:
    Algo 0 (default):    fc00:0:1:1::     (shortest IGP path)
    Algo 128 (low-delay): fc00:0:1:8001::  (lowest delay path)
    Algo 129 (TE metric): fc00:0:1:8101::  (TE-optimized path)

  To reach Node X via lowest-delay path, use SID fc00:0:1:8001::
  No explicit segment list needed — IGP computes the delay-optimal path.

IOS-XR Flex-Algo with SRv6:

router isis CORE
 flex-algo 128
  metric-type delay
  advertise-definition
 !
 address-family ipv6 unicast
  segment-routing srv6
   locator MAIN
   locator DELAY-OPT algorithm 128
  !
 !

segment-routing
 srv6
  locators
   locator MAIN
    prefix fc00:0:1::/48
   locator DELAY-OPT
    prefix fc00:0:1:8000::/48
    algorithm 128
   !
  !
 !
```

## JunOS SRv6 Configuration

### Basic SRv6 Setup

```
set routing-options segment-routing srv6 locator MAIN fc00:0:1::/48
set routing-options segment-routing srv6 transit-behavior END

set protocols isis interface ge-0/0/0.0 level 2 srv6-adjacency-segment
set protocols isis source-packet-routing srv6 locator MAIN end-sid fc00:0:1:1::

set protocols isis source-packet-routing srv6 locator MAIN
```

### JunOS SRv6 L3VPN

```
set routing-instances CUST-A instance-type vrf
set routing-instances CUST-A route-distinguisher 65000:100
set routing-instances CUST-A vrf-target target:65000:100

set routing-instances CUST-A protocols bgp group CE neighbor 10.1.1.2

set protocols bgp group IBGP neighbor fc00:0:2::1 family inet-vpn unicast srv6
set protocols bgp group IBGP neighbor fc00:0:2::1 family inet6-vpn unicast srv6

set routing-options segment-routing srv6 locator MAIN end-dt4-sid
set routing-options segment-routing srv6 locator MAIN end-dt6-sid
```

### JunOS SRv6 TE Policy

```
set protocols source-packet-routing srv6
set protocols source-packet-routing segment-list SL1 srv6 hop1 sid fc00:0:2:e001::
set protocols source-packet-routing segment-list SL1 srv6 hop2 sid fc00:0:5:d100::

set protocols source-packet-routing sr-policy P1 color 100
set protocols source-packet-routing sr-policy P1 endpoint fc00:0:5::1
set protocols source-packet-routing sr-policy P1 binding-sid fc00:0:1:b001::
set protocols source-packet-routing sr-policy P1 segment-list SL1
```

## Verification Commands

### IOS-XR

```
! SRv6 SID table
show segment-routing srv6 sid                       ! All local SIDs
show segment-routing srv6 sid detail                ! SIDs with behavior and counters
show segment-routing srv6 locator                   ! Configured locators

! SRv6 forwarding
show cef ipv6 fc00:0:2:d100::                      ! CEF entry for remote SID
show cef vrf CUST-A ipv4                            ! VRF CEF with SRv6 encap info
show cef vrf CUST-A ipv4 10.2.2.0/24 detail         ! Specific prefix with SRv6 info

! SRv6 TE policies
show segment-routing traffic-eng policy              ! All SR-TE policies
show segment-routing traffic-eng policy color 100    ! Specific color
show segment-routing traffic-eng forwarding policy   ! Forwarding state

! BGP with SRv6
show bgp vpnv4 unicast                              ! VPNv4 routes with SRv6 SIDs
show bgp vpnv4 unicast rd 65000:100 detail          ! Detailed with SRv6 encap type
show bgp l2vpn evpn                                 ! EVPN routes with SRv6

! IS-IS SRv6
show isis segment-routing srv6 locators              ! IS-IS SRv6 locator advertisements
show isis segment-routing srv6 sid                   ! SRv6 SIDs advertised in IS-IS

! uSID
show segment-routing srv6 micro-segment              ! uSID state
show segment-routing srv6 sid | include uN           ! uSID node behaviors
```

### JunOS

```
# SRv6 SID table
show route table inet6.0 protocol isis match fc00:0:  # SRv6 locators from IS-IS
show segment-routing srv6 locator                      # Local SRv6 locators
show segment-routing srv6 sid                          # Local SID table

# SRv6 forwarding
show route forwarding-table family inet6 matching fc00:0:  # FIB entries for SRv6

# SRv6 TE
show spring-traffic-engineering lsp                    # SR-TE LSPs
show spring-traffic-engineering lsp detail             # Detailed LSP with SID list

# BGP with SRv6
show route receive-protocol bgp fc00:0:2::1 table CUST-A.inet.0
show route table CUST-A.inet.0 detail                  # VRF routes with SRv6 encap
```

### Linux

```bash
# SRv6 SID table (kernel)
ip -6 route show | grep encap                          # Routes with SRv6 encap
ip -6 route show table localsid                        # Local SID table (if using iproute2)

# Add SRv6 encapsulation
ip -6 route add 10.2.2.0/24 encap seg6 mode encap \
  segs fc00:0:2:e001::,fc00:0:5:d100:: dev eth0

# Add local SID behavior
ip -6 route add fc00:0:1:d100::/64 encap seg6local \
  action End.DT4 vrftable 100 dev vrfA

# SRv6 tunnel statistics
ip -6 route show table all | grep seg6
```

## Tips

- Use /48 locators for standard SRv6 and /48 with micro-segment behavior for uSID; this gives 16 bits for function IDs (65,536 SIDs per node).
- Always configure SRv6 locators in IS-IS or OSPFv3 so they are reachable via the IGP; without IGP reachability, remote SIDs cannot be reached.
- For L3VPN, prefer `alloc mode per-vrf` to minimize SID consumption; `per-ce` mode allocates one SID per CE, which can exhaust the function space in large deployments.
- SRv6 adds significant overhead (88 bytes for 5 segments vs 20 bytes in SR-MPLS); use jumbo frames (9000+ MTU) on all provider links to avoid fragmentation.
- Deploy uSID (micro-SID) to compress overhead; 6 micro-SIDs fit in a single 128-bit container (16 bytes vs 96 bytes for 6 full SIDs).
- SRv6 TE policies use the same color/endpoint model as SR-MPLS TE; migration from SR-MPLS to SRv6 can reuse existing policy intent.
- When SRH contains only one segment, many implementations omit the SRH entirely and just set the IPv6 DA to the SID, saving 24 bytes.
- Flex-algo with SRv6 provides constraint-based routing without explicit segment lists; use it for network-wide intent (low-latency, avoiding specific links) rather than per-flow TE.
- SRv6 on Linux is production-ready in kernel 4.14+ for basic encap and 5.x+ for local SID behaviors; use iproute2 for configuration.
- In interworking scenarios (SRv6 domain connected to SR-MPLS domain), use END.BM at the boundary node to bind an SRv6 SID to an SR-MPLS policy.

## See Also

- segment-routing, mpls, ipv6, bgp, vxlan, is-is, ospf, mpls-vpn, mpls-te

## References

- [RFC 8986 — SRv6 Network Programming](https://www.rfc-editor.org/rfc/rfc8986)
- [RFC 8754 — IPv6 Segment Routing Header (SRH)](https://www.rfc-editor.org/rfc/rfc8754)
- [RFC 9252 — BGP Overlay Services Based on Segment Routing over IPv6](https://www.rfc-editor.org/rfc/rfc9252)
- [RFC 9256 — Segment Routing Policy Architecture](https://www.rfc-editor.org/rfc/rfc9256)
- [RFC 8402 — Segment Routing Architecture](https://www.rfc-editor.org/rfc/rfc8402)
- [draft-ietf-spring-srv6-srh-compression — SRv6 SID List Compression (uSID)](https://datatracker.ietf.org/doc/draft-ietf-spring-srv6-srh-compression/)
- [Cisco IOS-XR SRv6 Configuration Guide](https://www.cisco.com/c/en/us/td/docs/routers/asr9000/software/segment-routing/configuration/guide/b-segment-routing-cg-asr9k.html)
- [Juniper SRv6 Documentation](https://www.juniper.net/documentation/us/en/software/junos/segment-routing/topics/concept/srv6-overview.html)
- [Linux SRv6 Documentation](https://segment-routing.org/index.php/Implementation/LinuxKernel)
- [Cisco SRv6 Micro-SID (uSID) Guide](https://www.cisco.com/c/en/us/td/docs/routers/asr9000/software/segment-routing/configuration/guide/b-segment-routing-cg-asr9k/srv6-micro-sid.html)
