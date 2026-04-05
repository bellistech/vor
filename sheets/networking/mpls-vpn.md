# MPLS VPN (L3VPN, L2VPN, and VRF)

Virtual private network services built on MPLS infrastructure, using VRFs for routing isolation, MP-BGP for VPNv4/v6 route distribution, and pseudowires or VPLS for Layer 2 extension across provider networks.

## Concepts

### L3VPN Architecture

- **VRF (Virtual Routing and Forwarding):** Per-customer isolated routing and forwarding table on a PE router
- **PE (Provider Edge):** Router connecting to customer sites; runs VRFs and MP-BGP
- **CE (Customer Edge):** Customer router peering with PE; unaware of MPLS or VPN infrastructure
- **P (Provider Core):** Core router performing label switching only; no VRF or customer awareness
- **RD (Route Distinguisher):** 8-byte prefix prepended to IPv4/IPv6 routes to make them globally unique in BGP (format: Type0 ASN:nn, Type1 IP:nn, Type2 4-byte-ASN:nn)
- **RT (Route Target):** Extended BGP community used to control import/export of routes between VRFs
- **VPNv4/VPNv6:** Address family in MP-BGP that carries RD+prefix combinations between PEs

### RD vs RT

| Property         | Route Distinguisher (RD)              | Route Target (RT)                    |
|------------------|---------------------------------------|--------------------------------------|
| Purpose          | Make overlapping prefixes unique       | Control route distribution           |
| Scope            | Per-VRF, local to PE                  | Policy-based, can span VRFs/PEs      |
| Format           | 8 bytes prepended to prefix           | Extended community (Type:Value)      |
| Uniqueness       | Must be unique per VRF globally       | Same RT can be shared across VRFs    |
| Direction        | N/A (always applied)                  | Import/Export                        |

### Label Stack in L3VPN

```
+----------------+-------------------+---------------------+
| L2 Header      | Transport Label   | VPN Label           | IP Packet
| (Ethernet)     | (LDP/RSVP-TE)    | (BGP-assigned)      |
+----------------+-------------------+---------------------+

Transport Label: Switched hop-by-hop by P routers (learned via LDP or RSVP-TE)
VPN Label:       Identifies the VRF on the egress PE (learned via MP-BGP VPNv4)
```

### Packet Flow (Ingress to Egress)

1. CE sends IP packet to PE (normal IP routing)
2. Ingress PE performs VRF lookup, finds VPN label and BGP next-hop (remote PE)
3. Ingress PE pushes VPN label (inner) + transport label (outer)
4. P routers swap transport label hop-by-hop (standard MPLS forwarding)
5. Penultimate P router pops transport label (PHP)
6. Egress PE receives packet with VPN label only, pops it, looks up VRF, forwards to CE

## VRF Configuration

### Cisco IOS

```
! Define VRF
ip vrf CUSTOMER-A
 rd 65000:100
 route-target export 65000:100
 route-target import 65000:100

! Assign interface to VRF (must be done before configuring IP address)
interface GigabitEthernet0/1
 ip vrf forwarding CUSTOMER-A
 ip address 192.168.1.1 255.255.255.0

! Multiple RTs for hub-and-spoke or shared services
ip vrf CUSTOMER-B
 rd 65000:200
 route-target export 65000:200
 route-target import 65000:200
 route-target import 65000:999
 ! Import shared-services routes (e.g., DNS, NTP)
```

### Cisco IOS-XR

```
vrf CUSTOMER-A
 address-family ipv4 unicast
  import route-target
   65000:100
  !
  export route-target
   65000:100
  !
 !
!

interface GigabitEthernet0/0/0/1
 vrf CUSTOMER-A
 ipv4 address 192.168.1.1 255.255.255.0
!
```

### Juniper JunOS

```
routing-instances {
    CUSTOMER-A {
        instance-type vrf;
        interface ge-0/0/1.0;
        route-distinguisher 65000:100;
        vrf-target target:65000:100;
        vrf-table-label;
    }
}
```

### VRF-Lite (No MPLS)

```
! IOS: VRF without MPLS — used for local routing isolation
! Identical VRF config but no MP-BGP or MPLS labels
! Routes leaked between VRFs via static routes or route-maps

ip vrf MGMT
 rd 65000:999

interface GigabitEthernet0/2
 ip vrf forwarding MGMT
 ip address 10.255.0.1 255.255.255.0

! Route leak from global to VRF
ip route vrf MGMT 0.0.0.0 0.0.0.0 10.0.0.1 global
```

## MP-BGP for VPNv4

### PE-to-PE Configuration (IOS)

```
router bgp 65000
 no bgp default ipv4-unicast
 ! Disable default IPv4 activation for all neighbors
 neighbor 10.0.0.2 remote-as 65000
 neighbor 10.0.0.2 update-source Loopback0
 !
 address-family vpnv4
  neighbor 10.0.0.2 activate
  neighbor 10.0.0.2 send-community extended
  ! Extended communities carry Route Targets
 exit-address-family
```

### PE-to-PE Configuration (IOS-XR)

```
router bgp 65000
 address-family vpnv4 unicast
 !
 neighbor 10.0.0.2
  remote-as 65000
  update-source Loopback0
  address-family vpnv4 unicast
  !
 !
```

### Route Reflector for VPNv4

```
! RR configuration (IOS)
router bgp 65000
 neighbor 10.0.0.1 remote-as 65000
 neighbor 10.0.0.1 update-source Loopback0
 !
 address-family vpnv4
  neighbor 10.0.0.1 activate
  neighbor 10.0.0.1 send-community extended
  neighbor 10.0.0.1 route-reflector-client
 exit-address-family

! RR does not need VRFs — it only reflects VPNv4 routes
! RR should have nexthop-self disabled (default) to preserve original PE next-hop
```

## PE-CE Routing Protocols

### Static

```
! IOS: Static route in VRF context
ip route vrf CUSTOMER-A 10.10.0.0 255.255.0.0 192.168.1.2

! Redistribute into BGP VPNv4
router bgp 65000
 address-family ipv4 vrf CUSTOMER-A
  redistribute static
 exit-address-family
```

### OSPF as PE-CE

```
! IOS: OSPF in VRF
router ospf 100 vrf CUSTOMER-A
 router-id 192.168.1.1
 redistribute bgp 65000 subnets
 network 192.168.1.0 0.0.0.255 area 0

! BGP: Redistribute OSPF into VPNv4
router bgp 65000
 address-family ipv4 vrf CUSTOMER-A
  redistribute ospf 100 match internal external 1 external 2
 exit-address-family

! OSPF domain-id controls DN bit and route type on remote PE
! Same domain-id = routes appear as inter-area (Type 3 LSA)
! Different domain-id = routes appear as external (Type 5 LSA)
```

### BGP as PE-CE

```
! IOS: eBGP between PE and CE
router bgp 65000
 address-family ipv4 vrf CUSTOMER-A
  neighbor 192.168.1.2 remote-as 65100
  neighbor 192.168.1.2 activate
  ! No redistribution needed — BGP routes are native
  neighbor 192.168.1.2 as-override
  ! as-override: Replace customer ASN with provider ASN to prevent loop detection
  ! when same customer ASN exists at multiple sites
 exit-address-family
```

### EIGRP as PE-CE

```
! IOS: EIGRP in VRF
router eigrp 1
 address-family ipv4 vrf CUSTOMER-A autonomous-system 100
  redistribute bgp 65000 metric 10000 100 255 1 1500
  network 192.168.1.0 0.0.0.255
 exit-address-family

router bgp 65000
 address-family ipv4 vrf CUSTOMER-A
  redistribute eigrp 100
 exit-address-family
```

## Inter-AS VPN Options

### Option A (Back-to-Back VRF)

```
! ASBR in AS 65000
ip vrf CUSTOMER-A
 rd 65000:100
 route-target export 65000:100
 route-target import 65000:100

interface GigabitEthernet0/0
 ip vrf forwarding CUSTOMER-A
 ip address 172.16.0.1 255.255.255.252

router bgp 65000
 address-family ipv4 vrf CUSTOMER-A
  neighbor 172.16.0.2 remote-as 65001
  neighbor 172.16.0.2 activate
 exit-address-family

! ASBR in AS 65001 mirrors the config
! Pros: Simple, each AS is independent
! Cons: Per-VRF interface on ASBR, doesn't scale
```

### Option B (VPNv4 eBGP between ASBRs)

```
! ASBR-1 (AS 65000)
router bgp 65000
 neighbor 172.16.0.2 remote-as 65001
 !
 address-family vpnv4
  neighbor 172.16.0.2 activate
  neighbor 172.16.0.2 send-community extended
  ! ASBRs exchange VPNv4 routes directly
  ! ASBR changes next-hop to self
 exit-address-family

! Requires: next-hop-self on ASBR for VPNv4
! ASBR must allocate per-VRF labels even without local VRFs
! Pros: Scales better than Option A (no per-VRF interface)
! Cons: ASBR must hold all VPNv4 routes
```

### Option C (Multihop MP-eBGP between RRs/PEs)

```
! PE in AS 65000 peers with PE or RR in AS 65001 via multihop
router bgp 65000
 neighbor 10.0.1.1 remote-as 65001
 neighbor 10.0.1.1 update-source Loopback0
 neighbor 10.0.1.1 ebgp-multihop 255
 !
 address-family vpnv4
  neighbor 10.0.1.1 activate
  neighbor 10.0.1.1 send-community extended
  ! Next-hop is the remote PE loopback — requires inter-AS LSP
 exit-address-family

! Requires: end-to-end LSP between PEs across AS boundaries
! ASBRs exchange loopback routes with labels (labeled unicast)
! Pros: Most scalable, ASBRs carry minimal state
! Cons: Most complex, requires inter-AS MPLS
```

### Inter-AS Comparison

| Feature          | Option A            | Option B              | Option C              |
|------------------|--------------------|-----------------------|-----------------------|
| ASBR complexity  | Low (VRF per cust) | Medium (VPNv4 table)  | Low (labeled unicast) |
| Scalability      | Poor               | Medium                | High                  |
| AS independence   | Full               | Partial               | Low                   |
| ASBR VPN state   | Per-VRF            | All VPNv4 routes      | None (just loopbacks) |
| Inter-AS LSP     | No                 | No                    | Yes (required)        |

## L2VPN — Pseudowire (VPWS)

### Point-to-Point Pseudowire (IOS)

```
! xconnect-based (legacy)
interface GigabitEthernet0/1
 xconnect 10.0.0.5 100 encapsulation mpls
 ! 10.0.0.5 = remote PE loopback
 ! 100 = VC ID (must match on both ends)
 ! encapsulation: mpls (default), l2tpv3

! Pseudowire class (for advanced options)
pseudowire-class PW-CLASS-1
 encapsulation mpls
 preferred-path interface Tunnel0
 ! Force pseudowire over a specific TE tunnel

interface GigabitEthernet0/1
 xconnect 10.0.0.5 100 pw-class PW-CLASS-1
```

### L2VPN Configuration (IOS Modern Style)

```
l2vpn xconnect context VPWS-CUST-A
 member GigabitEthernet0/1 service-instance 10
 member 10.0.0.5 100 encapsulation mpls
!

interface GigabitEthernet0/1
 service instance 10 ethernet
  encapsulation dot1q 100
  rewrite ingress tag pop 1 symmetric
 !
```

### IOS-XR Pseudowire

```
l2vpn
 xconnect group CUST-A
  p2p SITE-1
   interface GigabitEthernet0/0/0/1.100
   neighbor ipv4 10.0.0.5 pw-id 100
   !
  !
 !
!

interface GigabitEthernet0/0/0/1.100 l2transport
 encapsulation dot1q 100
!
```

### Pseudowire Label Stack

```
+----------------+-------------------+---------------------+
| L2 Header      | Transport Label   | PW Label (VC label) | Original L2 Frame
| (Ethernet)     | (LDP/RSVP-TE)    | (targeted LDP)      | (with or without VLAN tag)
+----------------+-------------------+---------------------+

PW Label: Identifies the pseudowire on the remote PE (signaled via targeted LDP or BGP)
Control Word: Optional 4-byte field after PW label for sequencing and padding
```

## L2VPN — VPLS

### VPLS Concepts

- **VPLS (Virtual Private LAN Service):** Multipoint L2VPN — emulates a LAN across MPLS
- **VSI (Virtual Switch Instance):** Per-VPLS MAC learning and forwarding table on PE
- **Full-Mesh PWs:** Every PE in a VPLS instance has a pseudowire to every other PE
- **Split Horizon:** Frames received on a PW are never forwarded to another PW (prevents loops)
- **MAC Learning:** Standard Ethernet MAC learning on VSI (source MAC from data plane)

### VPLS Configuration (IOS)

```
l2vpn vfi context VPLS-CUST-A
 vpn id 100
 member 10.0.0.5 encapsulation mpls
 member 10.0.0.6 encapsulation mpls

! Bridge domain ties VFI to local interfaces
bridge-domain 100
 member vfi VPLS-CUST-A
 member GigabitEthernet0/1 service-instance 10
```

### VPLS Configuration (IOS-XR)

```
l2vpn
 bridge group CUST-A
  bridge-domain VPLS-100
   interface GigabitEthernet0/0/0/1.100
   !
   vfi VPLS-100
    neighbor 10.0.0.5 pw-id 100
    neighbor 10.0.0.6 pw-id 100
   !
  !
 !
!
```

### Juniper VPLS

```
routing-instances {
    VPLS-CUST-A {
        instance-type vpls;
        interface ge-0/0/1.100;
        protocols {
            vpls {
                vpls-id 100;
                neighbor 10.0.0.5;
                neighbor 10.0.0.6;
            }
        }
    }
}
```

### H-VPLS (Hierarchical VPLS)

```
! Spoke PE (MTU/PE) connects to Hub PE via single pseudowire
! Reduces full-mesh requirement

! Hub PE config (IOS)
l2vpn vfi context VPLS-CUST-A
 vpn id 100
 member 10.0.0.5 encapsulation mpls
 ! Full-mesh PW to other hub PEs only

! Spoke PE config — single PW to hub
interface GigabitEthernet0/1
 xconnect 10.0.0.1 100 encapsulation mpls
 ! 10.0.0.1 = hub PE

! Hub PE: bridge-domain includes both VFI and spoke PWs
! Split horizon between spoke PWs prevents loops
! Benefit: spoke PE needs only 1 PW instead of N-1
```

### VPLS Scaling

```
Full-mesh PWs required:
  P = N(N-1)/2  where N = number of PEs in the VPLS instance

  5 PEs:  10 PWs
  10 PEs: 45 PWs
  20 PEs: 190 PWs
  50 PEs: 1,225 PWs

H-VPLS with K hub PEs and S spoke PEs:
  Hub-to-hub: K(K-1)/2 PWs
  Spoke-to-hub: S PWs (each spoke connects to 1 or 2 hubs)
  Total: K(K-1)/2 + S  (much less than (K+S)(K+S-1)/2)
```

## VPNv6 (6VPE)

```
! IOS: VPNv6 support
ip vrf CUSTOMER-A
 rd 65000:100
 route-target export 65000:100
 route-target import 65000:100
 address-family ipv6
  rd 65000:100
  route-target export 65000:100
  route-target import 65000:100
 exit-address-family

! MP-BGP VPNv6
router bgp 65000
 address-family vpnv6
  neighbor 10.0.0.2 activate
  neighbor 10.0.0.2 send-community extended
 exit-address-family

 address-family ipv6 vrf CUSTOMER-A
  redistribute connected
 exit-address-family
```

## Troubleshooting

### VRF Verification

```bash
# Show VRF configuration and interfaces
show ip vrf
show ip vrf detail CUSTOMER-A
show ip vrf interfaces

# VRF routing table
show ip route vrf CUSTOMER-A
show ipv6 route vrf CUSTOMER-A

# Ping/traceroute within VRF context
ping vrf CUSTOMER-A 192.168.1.2
traceroute vrf CUSTOMER-A 10.10.0.1
```

### MP-BGP VPNv4 Verification

```bash
# VPNv4 BGP table
show bgp vpnv4 unicast all
show bgp vpnv4 unicast all summary

# Specific VRF BGP table
show bgp vpnv4 unicast vrf CUSTOMER-A
show bgp vpnv4 unicast vrf CUSTOMER-A 10.10.0.0/16

# Verify RD and RT
show bgp vpnv4 unicast all 65000:100:10.10.0.0/16

# Check extended communities (RT)
show bgp vpnv4 unicast all community
```

### MPLS Forwarding Verification

```bash
# MPLS forwarding table (LFIB)
show mpls forwarding-table
show mpls forwarding-table vrf CUSTOMER-A

# Verify label binding for VPNv4 route
show bgp vpnv4 unicast all labels

# Check transport label (LDP or RSVP-TE)
show mpls ldp bindings
show mpls forwarding-table labels 100

# End-to-end label path
traceroute mpls ipv4 10.0.0.5/32
```

### L2VPN Troubleshooting

```bash
# Pseudowire status
show mpls l2transport vc
show mpls l2transport vc detail
show mpls l2transport vc vcid 100

# VPLS bridge domain
show bridge-domain 100
show bridge-domain 100 detail

# MAC address table
show bridge-domain 100 mac-address

# Pseudowire statistics
show mpls l2transport vc vcid 100 statistics

# IOS-XR equivalents
show l2vpn xconnect
show l2vpn xconnect detail
show l2vpn bridge-domain
show l2vpn bridge-domain detail
show l2vpn bridge-domain mac-address
```

### Common Issues Checklist

```
1. VRF route not appearing on remote PE
   - Check: RT export on originating PE matches RT import on receiving PE
   - Check: MP-BGP VPNv4 session is established between PEs
   - Check: send-community extended is configured on BGP neighbor
   - Check: PE-CE routing protocol is redistributing into BGP

2. Ping across VPN fails but routes exist
   - Check: Transport label (LDP/RSVP-TE) exists for remote PE loopback
   - Check: VPN label in LFIB points to correct VRF
   - Check: Return path — remote PE has route back to source VRF

3. VPLS MAC not learning
   - Check: Pseudowire is UP (show mpls l2transport vc)
   - Check: Bridge domain has both VFI and local interface as members
   - Check: Split horizon is not blocking expected traffic

4. Pseudowire down
   - Check: Targeted LDP session to remote PE is established
   - Check: VC ID matches on both ends
   - Check: Encapsulation type matches on both ends
   - Check: MTU matches on both ends (common mismatch issue)
```

## Tips

- RD makes routes unique; RT controls distribution. They can be the same value but serve completely different purposes.
- Always enable `send-community extended` on VPNv4 neighbors; without it, Route Targets are stripped and VRF import fails silently.
- When using OSPF as PE-CE, the DN (Down) bit prevents routing loops by marking LSAs that originated from BGP redistribution; do not filter or ignore it.
- VRF-Lite provides routing isolation without MPLS but requires a separate interface (or sub-interface) per VRF at each hop.
- Inter-AS Option A is simplest but does not scale; Option B scales better; Option C is most scalable but requires inter-AS LSP.
- In VPLS, the full-mesh PW requirement grows as $N(N-1)/2$; use H-VPLS for deployments with more than 10-15 PEs.
- Pseudowire MTU mismatches are a common cause of PW failure; ensure both ends agree on MTU (and control-word usage).
- The `as-override` command on PE-CE BGP sessions solves the loop-detection problem when a customer uses the same ASN at multiple sites, but it weakens BGP loop prevention.
- Always verify the full label chain: transport label (LDP/RSVP-TE to remote PE) + VPN label (BGP-assigned, identifies remote VRF).
- Use `show mpls forwarding-table` liberally; it shows exactly what the router does with each label (swap, push, pop, aggregate).

## See Also

- mpls, mpls-te, bgp, ospf, eigrp, is-is, vxlan, segment-routing, ldp

## References

- [RFC 4364 — BGP/MPLS IP Virtual Private Networks (VPNs)](https://www.rfc-editor.org/rfc/rfc4364)
- [RFC 4659 — BGP-MPLS IP Virtual Private Network (VPN) Extension for IPv6 VPN](https://www.rfc-editor.org/rfc/rfc4659)
- [RFC 4761 — Virtual Private LAN Service (VPLS) Using BGP for Auto-Discovery and Signaling](https://www.rfc-editor.org/rfc/rfc4761)
- [RFC 4762 — Virtual Private LAN Service (VPLS) Using LDP Signaling](https://www.rfc-editor.org/rfc/rfc4762)
- [RFC 6074 — Provisioning, Auto-Discovery, and Signaling in L2VPNs](https://www.rfc-editor.org/rfc/rfc6074)
- [RFC 4447 — Pseudowire Setup and Maintenance Using LDP](https://www.rfc-editor.org/rfc/rfc4447)
- [RFC 4684 — Constrained Route Distribution for BGP/MPLS IP VPNs (RT-Constrain)](https://www.rfc-editor.org/rfc/rfc4684)
- [Cisco L3VPN Configuration Guide (IOS XE)](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/mp_l3_vpns/configuration/xe-16/mp-l3-vpns-xe-16-book.html)
- [Cisco L2VPN and Ethernet Services Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/mp_l2_vpns/configuration/xe-16/mp-l2-vpns-xe-16-book.html)
- [Juniper L3VPN Configuration Guide](https://www.juniper.net/documentation/us/en/software/junos/vpn-l3/topics/topic-map/l3-vpns-overview.html)
