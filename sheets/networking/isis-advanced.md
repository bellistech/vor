# Advanced IS-IS (Intermediate System to Intermediate System)

Advanced IS-IS features including TLV extensions, segment routing, multi-topology, flex-algo, and convergence optimization for modern service provider and enterprise networks.

## TLV Structure

### Key TLV Reference

| TLV Type | Name | Description |
|:---------|:-----|:------------|
| 1 | Area Addresses | List of area addresses for this IS |
| 2 | IS Neighbors (old) | L1/L2 IS neighbors with default metric |
| 6 | IS Neighbors MAC Address | MAC of neighbor on LAN |
| 8 | Padding | Pads IIH to MTU for MTU validation |
| 10 | Authentication | Password/HMAC for PDU authentication |
| 22 | Extended IS Reachability | Wide metrics IS neighbors (replaces TLV 2) |
| 128 | IP Internal Reach (old) | IPv4 prefixes with narrow metrics |
| 130 | IP External Reach (old) | External IPv4 prefixes (narrow metrics) |
| 135 | Extended IP Reachability | Wide metrics IPv4 prefixes (replaces 128/130) |
| 137 | Dynamic Hostname | Hostname-to-System-ID mapping |
| 222 | MT IS Neighbors | Multi-topology extended IS neighbors |
| 229 | MT Entries | Multi-topology membership |
| 232 | IPv6 Reachability | IPv6 prefix reachability information |
| 233 | IPv6 Interface Address | IPv6 interface addresses |
| 235 | MT IPv6 Reachability | Multi-topology IPv6 reachability |
| 242 | Router Capability | SR, flex-algo, and other router capabilities |

### Sub-TLV Structure (within TLV 22/135/222/232)

| Sub-TLV | Parent TLV | Description |
|:---------|:-----------|:------------|
| 3 | 22 | Admin Group (link coloring for TE) |
| 6 | 22 | IPv4 Interface Address |
| 8 | 22 | IPv4 Neighbor Address |
| 9 | 22 | Maximum Link Bandwidth |
| 10 | 22 | Maximum Reservable Bandwidth |
| 11 | 22 | Unreserved Bandwidth |
| 12 | 22 | IPv6 Interface Address |
| 13 | 22 | IPv6 Neighbor Address |
| 18 | 22 | TE Default Metric |
| 3 | 135 | Prefix-SID (Segment Routing) |
| 4 | 135 | Flex-Algo Prefix Metric |
| 11 | 135 | IPv4 Source Router ID |
| 32 | 22 | Adj-SID (Segment Routing) |
| 33 | 22 | LAN Adj-SID |

### TLV Encoding Format

```
+--------+--------+--------+--------+--------+---
|  Type  | Length | Value (variable length)   ...
| 1 byte | 1 byte | 0-255 bytes               |
+--------+--------+--------+--------+--------+---

Sub-TLV (nested within a TLV's value field):
+--------+--------+--------+---
|  Type  | Length | Value  ...
| 1 byte | 1 byte | ...    |
+--------+--------+--------+---
```

## Multi-Topology IS-IS (RFC 5120)

### Concept

```
Multi-Topology allows separate SPF trees for different address families
or traffic types, each with independent metrics and paths.

Standard MT-IDs:
  MT-ID 0: Default topology (IPv4 unicast)
  MT-ID 2: IPv6 unicast
  MT-ID 3: IPv4 multicast
  MT-ID 4: IPv6 multicast
```

### IOS-XE Configuration

```
router isis CORE
 net 49.0001.0000.0000.0001.00
 metric-style wide
 !
 address-family ipv4 unicast
  ! Default topology (MT-ID 0)
 exit-address-family
 !
 address-family ipv6 unicast
  multi-topology
  ! Creates separate topology (MT-ID 2)
 exit-address-family

interface GigabitEthernet0/0
 ip router isis CORE
 ipv6 router isis CORE
 isis ipv6 metric 100
```

### IOS-XR Configuration

```
router isis CORE
 net 49.0001.0000.0000.0001.00
 address-family ipv4 unicast
  metric-style wide
 !
 address-family ipv6 unicast
  multi-topology
 !
 interface GigabitEthernet0/0/0/0
  address-family ipv4 unicast
   metric 10
  !
  address-family ipv6 unicast
   metric 100
  !
```

### Verification

```
show isis topology                    ! IPv4 default topology
show isis ipv6 topology               ! IPv6 multi-topology
show isis database detail             ! Shows MT entries in LSPs
show isis route ipv6                  ! IPv6 routes via MT
```

## IS-IS for IPv6

### Single Topology (default)

```
! IPv4 and IPv6 share the same SPF tree and metric
! Simpler but constrains IPv6 paths to match IPv4

router isis CORE
 net 49.0001.0000.0000.0001.00
 metric-style wide
 address-family ipv6 unicast
  ! No multi-topology = single topology mode
 exit-address-family

interface GigabitEthernet0/0
 ip router isis CORE
 ipv6 router isis CORE
 ! Single metric applies to both AF
```

### IPv6 TLVs in LSPs

```
TLV 232 — IPv6 Reachability:
  Carries IPv6 prefixes with wide metric (32-bit)
  Supports sub-TLVs for SR, tags, etc.

TLV 233 — IPv6 Interface Address:
  Link-local or global IPv6 address of the interface

TLV 235 — MT IPv6 Reachability:
  IPv6 prefixes associated with a specific MT-ID
```

## Segment Routing Extensions

### Prefix-SID (TLV 135, Sub-TLV 3)

```
! Prefix-SID: globally unique segment ID bound to an IP prefix (typically a loopback)
! SRGB (Segment Routing Global Block): label range allocated for prefix SIDs

! IOS-XE
router isis CORE
 net 49.0001.0000.0000.0001.00
 metric-style wide
 segment-routing mpls
 segment-routing prefix-sid-map advertise-local

interface Loopback0
 ip address 10.0.0.1 255.255.255.255
 ip router isis CORE
 isis prefix-sid index 1
 ! Absolute label = SRGB base + index = 16000 + 1 = 16001
```

```
! IOS-XR
router isis CORE
 address-family ipv4 unicast
  segment-routing mpls
  segment-routing prefix-sid-map advertise-local
 !
 interface Loopback0
  address-family ipv4 unicast
   prefix-sid index 1
```

### Adjacency-SID (TLV 22, Sub-TLV 32/33)

```
! Adj-SID: locally significant label bound to a specific adjacency
! Automatically allocated by IS-IS for each adjacency
! Used for traffic engineering (explicit path via adj-SIDs)

! Verify adjacency SIDs
show isis adjacency detail           ! Shows adj-SID per neighbor
show isis segment-routing adjacency  ! IOS-XR
show mpls forwarding-table           ! Verify label forwarding
```

### SRGB and SRLB Configuration

```
! IOS-XE
segment-routing mpls
 global-block 16000 23999
 local-block 15000 15999

! IOS-XR
segment-routing
 global-block 16000 23999
 local-block 15000 15999
```

### Verification

```
show isis segment-routing prefix-sid-map
show isis segment-routing global-block
show isis database detail | section SID
show mpls forwarding-table labels 16001
show segment-routing mpls state
```

## Flex-Algo (RFC 9350)

### Concept

```
Flex-Algo allows defining multiple algorithm instances within IS-IS,
each computing a separate SPF with its own constraints:

  Algorithm 128-255: User-defined flex-algos
  Algorithm 0: Standard SPF (default)

Each flex-algo can use different:
  - Metric types (IGP, TE, latency, etc.)
  - Constraints (include/exclude affinities)
  - Calculation types (SPF, strict SPF)
```

### IOS-XR Configuration

```
router isis CORE
 flex-algo 128
  metric-type delay           ! Use latency metric instead of IGP
  advertise-definition
 !
 flex-algo 129
  affinity exclude-any RED    ! Avoid RED-colored links
  advertise-definition
 !
 interface Loopback0
  address-family ipv4 unicast
   prefix-sid algorithm 128 index 101   ! Label = SRGB + 101 for algo 128
   prefix-sid algorithm 129 index 201   ! Label = SRGB + 201 for algo 129

! Link coloring (affinity)
 interface GigabitEthernet0/0/0/0
  affinity flex-algo RED
```

### Verification

```
show isis flex-algo                   ! All defined flex-algos
show isis flex-algo 128               ! Specific algo details
show isis route flex-algo 128         ! Routes computed by algo 128
show isis topology flex-algo 128      ! Topology for algo 128
```

## IS-IS Overload Bit

```
! Setting the overload (OL) bit causes other routers to avoid transiting
! through this router (still reachable as a destination)

! IOS-XE — permanent overload
router isis CORE
 set-overload-bit

! IOS-XE — overload on startup (wait for BGP to converge)
router isis CORE
 set-overload-bit on-startup wait-for-bgp

! IOS-XR
router isis CORE
 set-overload-bit on-startup wait-for-bgp
 set-overload-bit on-startup 300      ! 300 seconds timeout

! Verify
show isis database detail | include OL
show isis neighbors        ! Check if neighbors show OL flag
```

| Use Case | Configuration |
|:---------|:-------------|
| Maintenance window | `set-overload-bit` (manually remove after) |
| Post-reload convergence | `set-overload-bit on-startup 300` |
| Wait for BGP | `set-overload-bit on-startup wait-for-bgp` |
| Stub router (destination only) | Permanent `set-overload-bit` |

## Authentication

### Interface Authentication (IIH PDUs)

```
! IOS-XE
interface GigabitEthernet0/0
 isis authentication mode md5
 isis authentication key-chain ISIS-KEY

key chain ISIS-KEY
 key 1
  key-string MY-SECRET-KEY
  accept-lifetime 00:00:00 Jan 1 2025 infinite
  send-lifetime 00:00:00 Jan 1 2025 infinite

! IOS-XR
router isis CORE
 interface GigabitEthernet0/0/0/0
  hello-password hmac-md5 MY-SECRET-KEY
```

### Domain/Area Authentication (LSP/CSNP/PSNP)

```
! IOS-XE
router isis CORE
 authentication mode md5 level-2
 authentication key-chain ISIS-KEY level-2
 authentication mode md5 level-1
 authentication key-chain ISIS-KEY level-1

! IOS-XR
router isis CORE
 lsp-password hmac-md5 MY-LSP-KEY level 2
 snp-password hmac-md5 MY-SNP-KEY level 2
```

### Authentication TLV (Type 10)

```
+--------+--------+--------+--------+---
|  10    | Length |  Auth  | Auth   ...
|        |        |  Type  | Value  ...
+--------+--------+--------+--------+---

Auth Type:
  1 = Cleartext password
  54 = HMAC-MD5 (RFC 5304)
  3 = HMAC-SHA (RFC 5310) — recommended
```

## Mesh Groups

```
! Mesh groups reduce flooding overhead in fully meshed topologies
! (e.g., MPLS/VPLS full-mesh of PE routers)
! Members of a mesh group do not flood LSPs received from the group
! to other members of the same group

! IOS-XE
interface GigabitEthernet0/0
 isis mesh-group 1

interface GigabitEthernet0/1
 isis mesh-group 1

! IOS-XR
router isis CORE
 interface GigabitEthernet0/0/0/0
  mesh-group 1
 interface GigabitEthernet0/0/0/1
  mesh-group 1

! Block flooding entirely on an interface
interface GigabitEthernet0/2
 isis mesh-group blocked
```

## Graceful Restart (RFC 5306)

```
! Allows IS-IS to restart without disrupting forwarding
! Neighbors maintain adjacency during restart period

! IOS-XE
router isis CORE
 nsf ietf                   ! IETF graceful restart (RFC 5306)
 nsf cisco                  ! Cisco proprietary NSF
 nsf interval 120           ! Restart window in seconds

! IOS-XR
router isis CORE
 nsf ietf
 nsf lifetime 120

! Verify
show isis nsf               ! NSF status and timers
show isis neighbors         ! Check restart flags (R bit)
```

## IS-IS BFD Integration

```
! BFD provides sub-second failure detection for IS-IS adjacencies

! IOS-XE
router isis CORE
 bfd all-interfaces          ! Enable BFD for all IS-IS interfaces

interface GigabitEthernet0/0
 bfd interval 100 min_rx 100 multiplier 3
 ! 100ms Tx/Rx, 300ms detect time

! IOS-XR
router isis CORE
 interface GigabitEthernet0/0/0/0
  bfd minimum-interval 100
  bfd multiplier 3
  bfd fast-detect ipv4
  bfd fast-detect ipv6

! Verify
show bfd neighbors detail
show isis adjacency detail    ! Shows BFD status per adjacency
```

## L1/L2 Route Leaking

```
! By default, L2 routes are not leaked into L1
! L1 routers use default route (ATT bit) to reach L2
! Route leaking allows selective L2->L1 or L1->L2 redistribution

! IOS-XE: L2 to L1 leaking
router isis CORE
 redistribute isis ip level-2 into level-1 route-map L2-TO-L1

route-map L2-TO-L1 permit 10
 match ip address prefix-list LEAKED-PREFIXES

ip prefix-list LEAKED-PREFIXES seq 10 permit 10.0.0.0/8 le 24

! IOS-XE: L1 to L2 leaking (suppress default L1->L2 redistribution)
router isis CORE
 redistribute isis ip level-1 into level-2 route-map L1-TO-L2
 no redistribute isis ip level-1 into level-2

! IOS-XR
router isis CORE
 address-family ipv4 unicast
  propagate level 2 into level 1 route-policy L2-TO-L1
```

### ATT (Attached) Bit Behavior

```
! L1/L2 routers set the ATT bit in L1 LSPs
! L1-only routers install a default route toward the nearest L1/L2 router
! Route leaking allows replacing this default with specific routes

show isis database detail | include ATT
```

## IS-IS Database Overload Protection

```
! Limit LSP database size to prevent memory exhaustion

! IOS-XR
router isis CORE
 lsp-mtu 1492                        ! Match interface MTU minus overhead
 max-lsp-lifetime 65535              ! Maximum LSP lifetime (seconds)
 lsp-refresh-interval 65000          ! LSP refresh before expiry
 lsp-gen-interval maximum-wait 5000 initial-wait 50 secondary-wait 200

! IOS-XE
router isis CORE
 lsp-gen-interval 5 50 200           ! max initial secondary (ms)
 spf-interval 5 50 200               ! SPF throttle timers
 max-lsp-lifetime 65535
 lsp-refresh-interval 65000
```

## Quick Reference: Timers

| Timer | Default | Tuned (Fast) | Purpose |
|:------|:--------|:-------------|:--------|
| Hello interval (L1) | 10s | 1-3s | IIH keepalive |
| Hello interval (L2) | 10s | 1-3s | IIH keepalive |
| Hello multiplier | 3 | 3 | Dead interval = hello x multiplier |
| LSP lifetime | 1200s | 65535s | Maximum LSP age |
| LSP refresh | 900s | 65000s | Re-originate before expiry |
| SPF initial wait | 5000ms | 50ms | Delay before first SPF run |
| SPF secondary wait | 5000ms | 200ms | Delay between subsequent SPF runs |
| SPF maximum wait | 5000ms | 5000ms | Max delay cap (exponential backoff) |
| LSP generation initial | 5000ms | 50ms | Delay before first LSP generation |
| CSNP interval | 10s | 10s | CSNP broadcast on LANs |
| PSNP interval | 2s | 2s | PSNP request interval |
| BFD interval | N/A | 50-100ms | Sub-second failure detection |

## Tips

- Always use `metric-style wide` (32-bit metrics); narrow metrics (6-bit, max 63) are legacy and limit network design.
- Enable multi-topology when IPv4 and IPv6 require different forwarding paths; single topology is simpler but constrains both to identical paths.
- Set the overload bit on-startup with wait-for-bgp to prevent traffic blackholing while BGP converges after a reload.
- Use HMAC-SHA-256 (RFC 5310) over HMAC-MD5 for authentication; MD5 is cryptographically weakened.
- Tune SPF and LSP generation timers aggressively (50ms initial, 200ms secondary) for sub-second convergence in modern networks.
- Deploy BFD on all IS-IS interfaces for fast failure detection; relying on hello multiplier alone gives 30s detection time.
- Use mesh groups on fully meshed topologies (e.g., between PE routers) to reduce exponential flooding overhead.
- When leaking L2 routes into L1, always use a route-map/policy to control which prefixes are leaked; leaking everything defeats the purpose of L1/L2 separation.
- Flex-algo prefix-SIDs must use algorithm-specific index values; do not reuse the same index across different algorithms.
- Keep SRGB ranges consistent across all routers in the SR domain; mismatched ranges cause label conflicts.
- Set `max-lsp-lifetime` to 65535 and `lsp-refresh-interval` to 65000 to minimize unnecessary LSP flooding.
- Monitor `show isis statistics` for excessive SPF runs, which indicate instability (flapping links, route churn).

## See Also

- is-is, ospf, segment-routing, mpls, bgp, bfd, ipv6, vxlan

## References

- [RFC 1195 — Use of OSI IS-IS for Routing in TCP/IP and Dual Environments](https://www.rfc-editor.org/rfc/rfc1195)
- [RFC 5120 — M-IS-IS: Multi Topology Routing in IS-IS](https://www.rfc-editor.org/rfc/rfc5120)
- [RFC 5304 — IS-IS Cryptographic Authentication](https://www.rfc-editor.org/rfc/rfc5304)
- [RFC 5305 — IS-IS Extensions for Traffic Engineering](https://www.rfc-editor.org/rfc/rfc5305)
- [RFC 5306 — Restart Signaling for IS-IS](https://www.rfc-editor.org/rfc/rfc5306)
- [RFC 5308 — Routing IPv6 with IS-IS](https://www.rfc-editor.org/rfc/rfc5308)
- [RFC 5310 — IS-IS Generic Cryptographic Authentication](https://www.rfc-editor.org/rfc/rfc5310)
- [RFC 7981 — IS-IS Extensions for Advertising Router Information](https://www.rfc-editor.org/rfc/rfc7981)
- [RFC 8667 — IS-IS Extensions for Segment Routing](https://www.rfc-editor.org/rfc/rfc8667)
- [RFC 9350 — IGP Flexible Algorithm](https://www.rfc-editor.org/rfc/rfc9350)
- [Cisco IS-IS Configuration Guide (IOS-XE)](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/iproute_isis/configuration/xe-17/irs-xe-17-book.html)
- [Cisco IS-IS Configuration Guide (IOS-XR)](https://www.cisco.com/c/en/us/td/docs/routers/asr9000/software/asr9k-r7-9/routing/configuration/guide/b-routing-cg-asr9000-79x/configuring-is-is.html)
