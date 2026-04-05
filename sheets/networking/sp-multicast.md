# SP Multicast (Service Provider Multicast)

Multicast delivery at service provider scale, covering inter-domain multicast, multicast in MPLS networks, label-based multicast trees, and traffic engineering for efficient one-to-many content distribution.

## Concepts

### SP Multicast Architecture Layers

```
┌───────────────────────────────────────────┐
│         Application / Content              │
│    (IPTV, live streaming, software push)   │
├───────────────────────────────────────────┤
│         Overlay Signaling                  │
│    (IGMP/MLD from subscribers)             │
├───────────────────────────────────────────┤
│         Multicast Routing                  │
│    (PIM-SM, PIM-SSM, MSDP, BGP SAFI 2)    │
├───────────────────────────────────────────┤
│         Transport / Forwarding             │
│    (mLDP, P2MP RSVP-TE, ingress rep)       │
├───────────────────────────────────────────┤
│         Underlay Network                   │
│    (MPLS, SR-MPLS, IP)                     │
└───────────────────────────────────────────┘
```

### PIM-SM in SP Networks

```
# PIM Sparse Mode is the foundation for SP multicast
# Key roles:
# - RP (Rendezvous Point): shared tree root for ASM (Any-Source Multicast)
# - DR (Designated Router): sends PIM Join/Prune upstream
# - FHR (First Hop Router): registers source with RP
# - LHR (Last Hop Router): handles IGMP/MLD from subscribers

# SP-specific considerations:
# - Anycast RP for redundancy across core
# - SSM (Source-Specific Multicast) preferred for known sources (IPTV)
# - PIM inter-domain via MSDP or MP-BGP
```

### Source-Specific Multicast (SSM) at Scale

```
# SSM uses (S,G) joins directly to source — no RP needed
# Ideal for SP: source is known (headend encoder)
# IGMPv3 / MLDv2 required on subscriber-facing interfaces

# SSM range: 232.0.0.0/8 (IPv4), ff3x::/32 (IPv6)
# Subscriber joins (S, G) where S = headend IP, G = channel address

# Advantages for SP:
# - No RP infrastructure to manage
# - No shared tree (no (*,G) state, only (S,G))
# - Source validation inherent (subscriber specifies source)
# - Simpler security model
# - Faster channel change (direct shortest-path tree)
```

### Any-Source Multicast (ASM) with Anycast RP

```
# For ASM groups where source is unknown at join time
# Anycast RP: multiple RPs share the same IP address
# Sources register to nearest RP; MSDP syncs SA between RPs

         RP-1 (10.0.0.100)         RP-2 (10.0.0.100)
         Loopback: 10.0.0.100      Loopback: 10.0.0.100
              │                          │
              │     MSDP peering         │
              ├──────────────────────────┤
              │                          │
         Core Region A              Core Region B
```

## Inter-Domain Multicast

### MSDP (Multicast Source Discovery Protocol)

```
# MSDP distributes Source-Active (SA) messages between RPs
# in different PIM domains / autonomous systems

# SA message contains:
# - Source address (S)
# - Group address (G)
# - RP address (originating RP)

# MSDP peering is TCP-based (port 639)
# Typically established between RP routers or route-reflectors
```

### IOS-XR MSDP Configuration

```
router msdp
 ! Peer with RP in another domain
 peer 10.0.0.200
  connect-source Loopback0
  remote-as 65002
  !
 !
 ! Originator-ID (for anycast RP, use unique loopback)
 originator-id Loopback1
 !
 ! SA filter (only accept/advertise specific groups)
 sa-filter in list MSDP-SA-IN
 sa-filter out list MSDP-SA-OUT
 !
!

ipv4 access-list MSDP-SA-IN
 10 permit ipv4 any 239.0.0.0 0.255.255.255
 20 deny ipv4 any any
!
```

### MP-BGP Multicast (SAFI 2)

```
# BGP Address Family: IPv4 Multicast (SAFI 2)
# Distributes RPF (Reverse Path Forwarding) routes for multicast
# Separate from unicast RIB — allows different RPF topology

router bgp 65001
 address-family ipv4 multicast
  neighbor 10.0.0.2 activate
  network 10.10.0.0/16
  ! Redistribute multicast-specific routes
  redistribute static route-map MCAST-STATIC
  !
 !
!

# When to use SAFI 2:
# - Multicast sources reachable via different path than unicast
# - Dedicated multicast peering links
# - Multicast traffic engineering via separate RPF topology
```

## Multicast in MPLS Networks

### The Problem

```
# Native IP multicast (PIM) requires IP-aware transit routers
# In an MPLS core, P-routers may only do label switching
# Three solutions:
# 1. Run PIM on all P-routers (breaks MPLS abstraction)
# 2. mLDP — Multicast LDP (label-based multicast tree)
# 3. P2MP RSVP-TE — Traffic-engineered multicast tunnels
# 4. Ingress Replication — Unicast copies at ingress (small scale)
```

### mLDP (RFC 6388)

```
# mLDP builds label-switched multicast trees using LDP extensions
# Two tree types:
# - P2MP (Point-to-Multipoint): single root, multiple leaves
# - MP2MP (Multipoint-to-Multipoint): any node can be source

# mLDP FEC types:
# - P2MP FEC: identified by (root, opaque value)
# - MP2MP FEC: upstream and downstream, identified by (root, opaque value)

# mLDP advantages:
# - Follows IGP shortest path (like LDP for unicast)
# - No explicit path setup (unlike RSVP-TE)
# - Integrates with existing LDP sessions
# - Supports in-band multicast VPN signaling
```

### IOS-XR mLDP Configuration

```
! Enable mLDP on MPLS LDP
mpls ldp
 mldp
  ! Enable mLDP globally
  logging notifications
  !
  ! Address family
  address-family ipv4
   ! Make-before-break for tree optimization
   make-before-break delay 10
   !
  !
 !
!

! mLDP with multicast VPN (profile 14)
multicast-routing
 address-family ipv4
  interface Loopback0
   enable
   !
  !
  mdt source Loopback0
  !
 !
!

! VRF multicast with mLDP
vrf CUSTOMER-A
 address-family ipv4
  multicast-routing
   address-family ipv4
    interface all enable
    mdt default mldp p2mp
    mdt data mldp p2mp threshold 10
    !
   !
  !
 !
!
```

### P2MP RSVP-TE

```
# P2MP RSVP-TE builds traffic-engineered multicast tunnels
# Uses RSVP-TE signaling with P2MP extensions (RFC 4875)

# Key differences from mLDP:
# - Explicit path (CSPF-computed, constraint-based)
# - Bandwidth reservation per branch
# - FRR (Fast Reroute) protection per branch
# - Higher control plane overhead (per-tunnel state)
```

### IOS-XR P2MP RSVP-TE Configuration

```
! P2MP tunnel (headend)
interface tunnel-mte 1
 ipv4 unnumbered Loopback0
 destination 10.0.0.2
  path-option 1 dynamic
  !
 destination 10.0.0.3
  path-option 1 dynamic
  !
 destination 10.0.0.4
  path-option 1 dynamic
  !
 signalled-bandwidth 500000   ! kbps
 fast-reroute
!

! RSVP configuration
rsvp
 interface TenGigE0/0/0/0
  bandwidth 10000000   ! kbps available for reservation
  !
 !
!

! MPLS TE
mpls traffic-eng
 interface TenGigE0/0/0/0
  !
 interface TenGigE0/0/0/1
  !
 !
!
```

### Ingress Replication

```
# For small-scale multicast or when mLDP/RSVP-TE is unavailable
# Ingress PE replicates packet to each egress PE as unicast
# Uses P2P LSPs (standard LDP or RSVP-TE tunnels)

# Pros: No multicast-specific MPLS signaling needed
# Cons: Bandwidth multiplied by number of receivers
#        O(N) replication at ingress

# Configuration (multicast VPN profile 7)
vrf CUSTOMER-A
 address-family ipv4
  multicast-routing
   address-family ipv4
    mdt default ingress-replication
    !
   !
  !
 !
!
```

## IGMP/MLD at Aggregation

### IGMP Snooping and Proxy

```
! IGMP snooping on aggregation switch (L2 domain)
! Constrains multicast to ports with interested receivers

! IGMP proxy on BNG (reduces PIM state in core)
! BNG aggregates IGMP joins from thousands of subscribers
! Sends single PIM join upstream per channel

router igmp
 ! Interface toward subscribers
 interface Bundle-Ether100.100
  version 3                    ! IGMPv3 for SSM
  query-interval 60
  query-max-response-time 10
  maximum groups-per-interface 200   ! Limit channels per sub
  !
 !
!

! Static IGMP join (always-on channel)
router igmp
 interface Bundle-Ether100.100
  join-group 239.1.1.1 source 10.100.0.1
  !
 !
!
```

### MLD for IPv6 Multicast

```
router mld
 interface Bundle-Ether100.100
  version 2                    ! MLDv2 for SSM
  query-interval 60
  query-max-response-time 10
  maximum groups-per-interface 200
  !
 !
!
```

## Multicast VPN (mVPN) Profiles

```
# Multicast VPN delivers L3VPN multicast across MPLS core
# Multiple profiles defined for different scale/requirement

# Key profiles:
# Profile 0:  Default MDT (GRE, PIM in core) — legacy
# Profile 1:  mLDP P2MP, BGP-AD, BGP C-mcast signaling
# Profile 7:  Ingress Replication (no mcast in core)
# Profile 12: mLDP P2MP, BGP-AD, PIM C-mcast signaling
# Profile 14: mLDP P2MP partitioned, BGP C-mcast signaling
# Profile 17: P2MP RSVP-TE, BGP C-mcast signaling

# Profile 14 (most common for large SP):
# - Each PE builds independent mLDP tree
# - Partitioned MDT: only PEs with receivers join tree
# - Efficient bandwidth usage
```

## In-Band OAM for Multicast

```
# Multicast tree verification
# No native multicast ping/traceroute (one-way traffic)

# Tools:
# 1. mtrace (multicast traceroute) — traces RPF path
# 2. mfib counters — packet/byte counts per (S,G) on each node
# 3. BFD for multicast (limited support)
# 4. IGMP/MLD query/report monitoring

! mtrace from leaf to source
mtrace 239.1.1.1 source 10.100.0.1

! Check multicast forwarding state
show mfib ipv4 239.1.1.1 detail
show mfib ipv4 route 239.1.1.1/32

! Verify PIM neighbor adjacencies
show pim neighbor

! Check RPF for multicast source
show pim rpf 10.100.0.1

! mLDP tree state
show mpls mldp database
show mpls mldp root 10.0.0.1

! P2MP RSVP-TE tunnel state
show mpls traffic-eng tunnels p2mp
```

## Multicast Traffic Engineering

```
# Multicast TE ensures bandwidth is available for multicast streams
# Approaches:

# 1. P2MP RSVP-TE with bandwidth reservation
#    - Explicit bandwidth per tree branch
#    - CSPF ensures path has capacity
#    - FRR protection

# 2. mLDP with IGP TE metrics
#    - mLDP follows IGP shortest path
#    - Adjust IGP metrics to steer multicast traffic
#    - Less granular than RSVP-TE

# 3. SR-MPLS multicast (Tree-SID)
#    - Segment Routing extension for multicast
#    - P2MP trees identified by Tree-SID
#    - Centralized controller (PCE) computes trees
#    - Emerging technology (draft-ietf-pim-sr-p2mp-policy)

# Bandwidth planning for IPTV:
# SD channel: ~3.5 Mbps
# HD channel: ~8 Mbps
# 4K/UHD channel: ~25 Mbps
# Typical lineup: 200 SD + 100 HD + 20 4K
# Total: (200 * 3.5) + (100 * 8) + (20 * 25)
#      = 700 + 800 + 500 = 2,000 Mbps = 2 Gbps per link
# With multicast, this is constant regardless of subscriber count
```

## Show Commands

```bash
# PIM state and neighbors
show pim neighbor
show pim topology
show pim rpf 10.100.0.1
show pim group-map

# Multicast routes and state
show mroute
show mroute 239.1.1.1
show mroute 239.1.1.1 10.100.0.1 detail

# MFIB (multicast forwarding)
show mfib ipv4
show mfib ipv4 239.1.1.1

# IGMP groups and state
show igmp groups
show igmp interface Bundle-Ether100.100

# RP mapping
show pim rp mapping

# MSDP state
show msdp peer
show msdp sa-cache
show msdp statistics

# mLDP state
show mpls mldp database
show mpls mldp neighbors
show mpls mldp root

# P2MP RSVP-TE
show mpls traffic-eng tunnels p2mp
show rsvp session
show rsvp interface

# Multicast VPN
show mvpn vrf CUSTOMER-A database
show mvpn vrf CUSTOMER-A context

# Counters and statistics
show mfib ipv4 route statistics
show pim topology detail | include "data rate"
```

## Troubleshooting

### Multicast Traffic Not Reaching Subscribers

```bash
# 1. Verify IGMP join received on BNG
show igmp groups | include 239.1.1.1

# 2. Check PIM join sent upstream
show pim topology 239.1.1.1 | include "Join"

# 3. Verify RPF to source
show pim rpf 10.100.0.1
# RPF failure = multicast black-holed
# Fix: ensure unicast route (or SAFI 2 route) exists to source

# 4. Check mroute state on each hop
show mroute 239.1.1.1 10.100.0.1
# Look for (S,G) with incoming interface and outgoing interface list

# 5. Verify mLDP tree (if MPLS transport)
show mpls mldp database | include 239.1.1.1

# 6. Check for OIL (Outgoing Interface List) empty
# Empty OIL = no downstream receiver, tree is pruned
```

### Channel Change Slow (IPTV Zap Time)

```bash
# Target: < 500ms channel change
# Components of zap time:
#   1. IGMP Leave old group (immediate)
#   2. IGMP Join new group (< 1 IGMP query interval)
#   3. PIM Join propagation (hop-by-hop, ~50ms per hop)
#   4. First multicast packet arrives
#   5. Decoder waits for I-frame (up to GOP interval, ~2 sec)

# Reduce zap time:
# - Use SSM (direct SPT, no RP detour)
# - Pre-join popular channels (static IGMP on BNG)
# - Reduce IGMPv3 query-max-response-time
# - Use fast-leave (immediate prune on IGMP Leave)

! Fast leave on subscriber interface
router igmp
 interface Bundle-Ether100.100
  version 3
  immediate-leave
  !
 !
!
```

### RPF Failure

```bash
# RPF (Reverse Path Forwarding) check ensures multicast packet
# arrives on the expected interface (prevents loops)

# Diagnose:
show pim rpf 10.100.0.1
# If "RPF not found" — no route to source in unicast/mcast RIB

# Fix options:
# 1. Add static route to source
# 2. Ensure BGP/IGP has route to source network
# 3. Use SAFI 2 (BGP multicast) for separate RPF topology
# 4. Static RPF override:
!  ip multicast rpf-redirect route-policy RPF-FIX
```

## Tips

- Use SSM (232.0.0.0/8) for all known-source multicast (IPTV, live events); it eliminates RP dependency and simplifies the architecture.
- Deploy anycast RP with MSDP for ASM groups to provide RP redundancy without manual failover.
- Prefer mLDP over ingress replication for multicast VPN when more than 3-4 egress PEs exist; the bandwidth savings compound quickly.
- Always rate-limit IGMP/MLD on subscriber-facing interfaces to prevent state exhaustion attacks (set maximum groups-per-interface).
- Monitor mroute state count on core routers; excessive (S,G) state indicates poor SSM adoption or IGMP filter gaps.
- For IPTV, pre-join the top 10-20 most popular channels as static joins on the BNG to reduce zap time for common channel changes.
- P2MP RSVP-TE is preferred over mLDP when bandwidth guarantees and FRR protection are required (premium IPTV, financial multicast).
- Use mtrace and mfib counters for end-to-end multicast path verification; standard ping/traceroute does not work for multicast.
- When running multicast over MPLS, ensure TTL propagation is consistent (mpls ip propagate-ttl) to avoid unexpected multicast RPF failures.
- Keep MSDP SA filters tight; unfiltered MSDP can propagate thousands of SA entries from the global multicast internet.

## See Also

- bgp, mpls, ospf, is-is, bng, ipv4, ipv6, igmp

## References

- [RFC 7761 — Protocol Independent Multicast - Sparse Mode (PIM-SM)](https://www.rfc-editor.org/rfc/rfc7761)
- [RFC 4607 — Source-Specific Multicast for IP](https://www.rfc-editor.org/rfc/rfc4607)
- [RFC 6388 — Label Distribution Protocol Extensions for P2MP and MP2MP LSPs (mLDP)](https://www.rfc-editor.org/rfc/rfc6388)
- [RFC 4875 — Extensions to RSVP-TE for P2MP TE LSPs](https://www.rfc-editor.org/rfc/rfc4875)
- [RFC 3618 — Multicast Source Discovery Protocol (MSDP)](https://www.rfc-editor.org/rfc/rfc3618)
- [RFC 4760 — Multiprotocol Extensions for BGP-4 (SAFI 2)](https://www.rfc-editor.org/rfc/rfc4760)
- [RFC 6513 — Multicast in MPLS/BGP IP VPNs](https://www.rfc-editor.org/rfc/rfc6513)
- [RFC 6514 — BGP Encodings and Procedures for Multicast in MPLS/BGP IP VPNs](https://www.rfc-editor.org/rfc/rfc6514)
- [Cisco IOS-XR Multicast Configuration Guide](https://www.cisco.com/c/en/us/td/docs/routers/asr9000/software/multicast/configuration/guide/b-multicast-cg-asr9000.html)
