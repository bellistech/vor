# Unified MPLS (Seamless MPLS)

End-to-end MPLS architecture that extends label switched paths across multiple IGP domains (access, aggregation, core) without requiring a single flat IGP, using BGP Labeled Unicast (BGP-LU) to stitch per-domain LSPs into seamless transport.

## Concepts

### The Problem: Per-Domain MPLS

- Traditional MPLS uses one IGP + LDP per domain
- Inter-domain traffic requires per-hop label swap at every ABR
- ABRs must carry full VPN state, becoming scaling bottlenecks
- No end-to-end LSP means no end-to-end FRR or OAM
- Each domain boundary breaks the label stack continuity

### Seamless MPLS Architecture

- **Access domain:** Last-mile aggregation (IS-IS L1 or OSPF area, LDP)
- **Aggregation domain:** Metro ring/hub (IS-IS L1/L2 or OSPF area, LDP or RSVP-TE)
- **Core domain:** Backbone (IS-IS L2 or OSPF area 0, RSVP-TE or LDP)
- **ABR (Area Border Router):** Stitches domains using BGP-LU
- **End-to-end LSP:** Built by stacking labels from each domain

### BGP Labeled Unicast (BGP-LU / SAFI 4)

- AFI 1, SAFI 4 (IPv4) or AFI 2, SAFI 4 (IPv6)
- Carries prefix + MPLS label binding in BGP UPDATE
- Each ABR allocates a local label, creating a label-stitching chain
- Enables inter-area/inter-AS label continuity without merging IGPs
- RFC 8277 (supersedes RFC 3107)

### Label Stitching

- Each ABR receives a BGP-LU route with a remote label
- ABR allocates a new local label and advertises it upstream
- Forwarding: swap local label to remote label at each ABR
- Result: hierarchical label stack (transport + service labels)

### Hierarchical Label Stack

```
+------------------+------------------+------------------+---------+
| Access LDP label | Aggr RSVP label  | Core RSVP label  | VPN lbl |
| (to aggr PE)     | (to core ABR)    | (to remote ABR)  | (svc)   |
+------------------+------------------+------------------+---------+
                    Outer <------------------------------> Inner
```

- Outermost label: reaches the next domain boundary
- Intermediate labels: swapped at each ABR via BGP-LU stitching
- Innermost label: service label (L3VPN, L2VPN, EVPN)

## IOS-XR Configuration

### Core ABR (BGP-LU)

```
router bgp 65000
 address-family ipv4 unicast
  ! Allocate per-prefix labels for BGP-LU
  allocate-label all
 !
 neighbor-group IBGP-LU
  remote-as 65000
  update-source Loopback0
  address-family ipv4 labeled-unicast
   ! Advertise labeled routes to peers
   route-policy PASS in
   route-policy PASS out
   ! Next-hop-self to stitch labels at ABR
   next-hop-self
  !
 !
 neighbor 10.0.0.1
  use neighbor-group IBGP-LU
  description core-rr
 !
 neighbor 10.0.1.1
  use neighbor-group IBGP-LU
  description aggr-rr
 !
```

### Route Reflector for BGP-LU

```
router bgp 65000
 address-family ipv4 unicast
  allocate-label all
 !
 neighbor-group LU-CLIENTS
  remote-as 65000
  update-source Loopback0
  address-family ipv4 labeled-unicast
   route-reflector-client
   route-policy PASS in
   route-policy PASS out
   next-hop-self
  !
 !
 ! Core ABRs as clients
 neighbor 10.0.0.2
  use neighbor-group LU-CLIENTS
 neighbor 10.0.0.3
  use neighbor-group LU-CLIENTS
```

### Access Node (PE with LDP + BGP-LU)

```
router bgp 65000
 address-family ipv4 unicast
  allocate-label all
 !
 neighbor 10.0.1.1
  remote-as 65000
  update-source Loopback0
  address-family ipv4 labeled-unicast
   route-policy PASS in
   route-policy PASS out
  !
  ! Also carry VPNv4/EVPN for services
  address-family vpnv4 unicast
   route-policy PASS in
   route-policy PASS out
  !
 !
!
! LDP on access-facing interfaces
mpls ldp
 router-id 10.0.2.1
 address-family ipv4
  ! LDP for local domain transport
  interface GigabitEthernet0/0/0/0
  interface GigabitEthernet0/0/0/1
 !
```

### LDP-over-RSVP (Aggregation Domain)

```
! RSVP-TE tunnel provides transport across aggregation
interface tunnel-te100
 ipv4 unnumbered Loopback0
 destination 10.0.1.2
 autoroute announce
 path-option 1 dynamic
!
! LDP runs over RSVP tunnel (targeted LDP session)
mpls ldp
 router-id 10.0.1.1
 address-family ipv4
  label local allocate for host-routes
 !
 ! Targeted LDP session over RSVP-TE tunnel
 interface tunnel-te100
```

### Label Allocation Modes

```
! Per-prefix allocation (default, one label per FEC)
router bgp 65000
 address-family ipv4 unicast
  allocate-label all

! Per-VRF allocation (one label per VRF, requires table lookup)
router bgp 65000
 address-family vpnv4 unicast
  label mode per-vrf

! Per-CE allocation (one label per CE neighbor)
router bgp 65000
 address-family vpnv4 unicast
  label mode per-ce
```

### Integration with L3VPN

```
! PE in access domain provides L3VPN service
! Transport: LDP (local) + BGP-LU (inter-domain) + RSVP-TE (core)
! Service: VPNv4 label from BGP
!
vrf CUSTOMER-A
 address-family ipv4 unicast
  import route-target 65000:100
  export route-target 65000:100
 !
!
router bgp 65000
 vrf CUSTOMER-A
  rd 65000:100
  address-family ipv4 unicast
   redistribute connected
  !
 !
```

### Integration with L2VPN/EVPN

```
! EVPN over Unified MPLS transport
router bgp 65000
 address-family l2vpn evpn
  ! BGP-LU provides the transport label
  ! EVPN provides the service label
 !
!
l2vpn
 bridge group EVPN-BG
  bridge-domain EVPN-BD
   interface GigabitEthernet0/0/0/2.100
   !
   evi 100
   !
  !
 !
!
evpn
 evi 100
  bgp
   route-target import 65000:100
   route-target export 65000:100
  !
  advertise-mac
 !
```

## Verification Commands (IOS-XR)

```bash
# Verify BGP-LU table (labeled unicast routes)
show bgp ipv4 labeled-unicast

# Check label allocated for a specific prefix
show bgp ipv4 labeled-unicast 10.0.2.1/32

# Verify label stitching at ABR
show mpls forwarding

# Check end-to-end LSP (traceroute with labels)
traceroute mpls ipv4 10.0.2.1/32

# Verify LDP sessions (local domain)
show mpls ldp neighbor

# Verify RSVP-TE tunnels (if used)
show mpls traffic-eng tunnels brief

# Check VPNv4 routes with label stack
show bgp vpnv4 unicast rd 65000:100

# Verify CEF entry shows hierarchical label stack
show cef ipv4 10.0.2.1/32 detail

# BGP-LU neighbor state
show bgp ipv4 labeled-unicast summary
```

## Scaling Considerations

### Label Space

- LDP: 20-bit label space = 1,048,576 labels per router
- BGP-LU: one label per loopback prefix advertised inter-domain
- Access nodes only need labels for local domain + BGP-LU loopbacks
- Core ABRs carry labels for all inter-domain loopbacks

### Route Reflector Hierarchy

- Per-domain RRs reduce iBGP full-mesh within each domain
- Inter-domain RR hierarchy for BGP-LU (core RR reflects to aggregation RRs)
- Separate RR for VPNv4/EVPN (service plane) vs BGP-LU (transport plane)

### Convergence

- Local domain convergence: IGP + LDP/RSVP FRR (sub-second)
- Inter-domain convergence: BGP-LU reconvergence (seconds)
- BGP PIC (Prefix Independent Convergence): pre-programs backup paths
- BGP best-external: advertises best external path for faster failover

### Design Rules

- Keep per-domain IGP small (hundreds of nodes, not thousands)
- Use hierarchical RR topology matching the physical topology
- Deploy BGP PIC Edge + BGP PIC Core for fast convergence
- Use RSVP-TE FRR in core, LDP FRR (LFA/rLFA) in access/aggregation
- Plan label allocation: per-prefix for transport, per-VRF or per-CE for services

## Tips

- BGP-LU `next-hop-self` at ABRs is critical; without it, label stitching breaks because the next-hop remains unreachable across domain boundaries.
- The `allocate-label all` command is required on every BGP-LU speaker; without it, no labels are attached to advertised prefixes.
- LDP-over-RSVP provides TE and FRR in the core while keeping LDP simplicity at the edge; the LDP session is targeted over the RSVP tunnel.
- Separate your transport RRs (BGP-LU) from your service RRs (VPNv4/EVPN) for independent scaling and failure isolation.
- Unified MPLS preserves the existing per-domain IGP and LDP; you do not need to flatten your IGP to a single area.
- Use BGP PIC to pre-program backup label stacks; without it, inter-domain convergence depends on full BGP reconvergence.
- When migrating from per-domain MPLS, start by enabling BGP-LU on ABRs, then extend to access PEs incrementally.
- Segment Routing (SR-MPLS or SRv6) is the modern alternative that eliminates LDP and simplifies label stitching with SID-based forwarding.
- Always verify the full label stack with `show cef detail` to confirm each domain contributes the correct label layer.
- MTU planning is critical: count the maximum label stack depth and add 4 bytes per label to the required MTU (e.g., 3 labels = 12 bytes overhead).

## See Also

- mpls, bgp, ospf, is-is, vxlan, evpn-advanced, l2vpn-services

## References

- [RFC 8277 — Using BGP to Bind MPLS Labels to Address Prefixes](https://www.rfc-editor.org/rfc/rfc8277)
- [RFC 3107 — Carrying Label Information in BGP-4 (historic)](https://www.rfc-editor.org/rfc/rfc3107)
- [RFC 4364 — BGP/MPLS IP Virtual Private Networks](https://www.rfc-editor.org/rfc/rfc4364)
- [draft-ietf-mpls-seamless-mpls — Seamless MPLS Architecture](https://datatracker.ietf.org/doc/draft-ietf-mpls-seamless-mpls/)
- [Cisco IOS-XR — BGP Labeled Unicast Configuration Guide](https://www.cisco.com/c/en/us/td/docs/routers/asr9000/software/routing/configuration/guide/rcg-bgp.html)
- [Juniper — Seamless MPLS Configuration](https://www.juniper.net/documentation/us/en/software/junos/mpls/topics/concept/seamless-mpls-overview.html)
- [Nokia SR OS — Seamless MPLS Application Note](https://documentation.nokia.com/sr/)
