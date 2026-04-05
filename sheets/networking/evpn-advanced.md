# Advanced EVPN (Ethernet VPN)

BGP-based control-plane technology for Layer 2 and Layer 3 VPN services that provides MAC/IP advertisement, all-active multihoming, MAC mobility, ARP suppression, and integrated routing and bridging across MPLS and VXLAN fabrics.

## EVPN Route Types

### Type 1 — Ethernet Auto-Discovery (EAD)

- **Purpose:** Multihoming — fast convergence, aliasing, split-horizon
- **Two sub-types:**
  - **Per-ES (Ethernet Segment):** Advertised per ESI for mass withdrawal on ES failure
  - **Per-EVI (EVPN Instance):** Advertised per ESI per EVI for aliasing (load balancing)
- **Key fields:** RD, ESI, Ethernet Tag ID, MPLS label
- **Withdrawal:** When ES goes down, single withdrawal of per-ES EAD route triggers fast convergence on all remote PEs

```
Route Type 1 NLRI:
+-----------------------------------+
| Route Distinguisher (8 bytes)     |
| Ethernet Segment Identifier (10)  |
| Ethernet Tag ID (4 bytes)         |
| MPLS Label (3 bytes)              |
+-----------------------------------+
```

### Type 2 — MAC/IP Advertisement

- **Purpose:** Advertise learned MAC addresses and optionally bound IP addresses
- **Key fields:** RD, ESI, Ethernet Tag ID, MAC length, MAC address, IP length, IP address, MPLS label 1 (L2), MPLS label 2 (L3/optional)
- **MAC+IP binding:** Enables ARP suppression (PE answers ARP from control plane)
- **Two labels:** L2 VNI label for bridging, L3 VNI label for routing (symmetric IRB)

```
Route Type 2 NLRI:
+-----------------------------------+
| Route Distinguisher (8 bytes)     |
| ESI (10 bytes)                    |
| Ethernet Tag ID (4 bytes)         |
| MAC Address Length (1 byte)       |
| MAC Address (6 bytes)             |
| IP Address Length (1 byte)        |
| IP Address (0, 4, or 16 bytes)    |
| MPLS Label 1 (3 bytes)           |
| MPLS Label 2 (0 or 3 bytes)      |
+-----------------------------------+
```

### Type 3 — Inclusive Multicast Ethernet Tag

- **Purpose:** Advertise BUM (Broadcast, Unknown unicast, Multicast) handling method
- **Indicates:** Ingress replication list or multicast tunnel for a given EVI
- **Key fields:** RD, Ethernet Tag ID, IP length, Originating Router IP
- **PMSI Tunnel attribute:** Specifies tunnel type (ingress replication, P2MP LSP, mLDP, PIM)

```
Route Type 3 NLRI:
+-----------------------------------+
| Route Distinguisher (8 bytes)     |
| Ethernet Tag ID (4 bytes)         |
| IP Address Length (1 byte)        |
| Originating Router IP (4 or 16)   |
+-----------------------------------+
```

### Type 4 — Ethernet Segment Route

- **Purpose:** DF (Designated Forwarder) election among PEs sharing an ES
- **Key fields:** RD, ESI, IP length, Originating Router IP
- **Used for:** Discovering which PEs are attached to the same Ethernet Segment, then electing a DF per VLAN

```
Route Type 4 NLRI:
+-----------------------------------+
| Route Distinguisher (8 bytes)     |
| ESI (10 bytes)                    |
| IP Address Length (1 byte)        |
| Originating Router IP (4 or 16)   |
+-----------------------------------+
```

### Type 5 — IP Prefix Route

- **Purpose:** Advertise IP prefixes for inter-subnet routing in EVPN fabric
- **Key fields:** RD, ESI, Ethernet Tag ID, IP Prefix Length, IP Prefix, GW IP, MPLS Label
- **Use case:** External route injection, default route advertisement, inter-VRF leaking
- **Gateway IP:** Overlay next-hop for recursive resolution (0 if directly connected)

```
Route Type 5 NLRI:
+-----------------------------------+
| Route Distinguisher (8 bytes)     |
| ESI (10 bytes)                    |
| Ethernet Tag ID (4 bytes)         |
| IP Prefix Length (1 byte)         |
| IP Prefix (4 or 16 bytes)         |
| GW IP Address (4 or 16 bytes)     |
| MPLS Label (3 bytes)              |
+-----------------------------------+
```

### Route Type Summary

| Type | Name | When Sent | Required For |
|:---:|:---|:---|:---|
| 1 | Ethernet Auto-Discovery | ES comes up, per-EVI | Multihoming, aliasing, fast convergence |
| 2 | MAC/IP Advertisement | MAC learned (data or control plane) | Forwarding, ARP suppression, IRB |
| 3 | Inclusive Multicast | EVI created on PE | BUM traffic replication |
| 4 | Ethernet Segment | ES comes up | DF election |
| 5 | IP Prefix | IP route injected | Inter-subnet routing, external routes |

## Multihoming

### Ethernet Segment Identifier (ESI)

- 10-byte identifier for a multi-homed Ethernet Segment
- **ESI 0:** Single-homed (no multihoming)
- **ESI Type 0:** Manually configured (arbitrary value)
- **ESI Type 1:** Auto-derived from LACP system-id + port key
- **ESI Type 3:** Auto-derived from system MAC + local discriminator

```
ESI Format (10 bytes):
+--------+----------------------------------------+
| Type   | Value (9 bytes)                        |
| (1 B)  |                                        |
+--------+----------------------------------------+

Type 0: 00:AA:BB:CC:DD:EE:FF:00:01:00  (manual)
Type 1: 01:<LACP system-id>:<port-key>:00  (auto LACP)
Type 3: 03:<system-MAC>:<local-disc>  (auto MAC)
```

### All-Active Multihoming

- All PEs attached to the same ES forward traffic simultaneously
- Per-flow load balancing across PEs using aliasing (Type 1 per-EVI routes)
- Remote PE sees multiple next-hops for the same MAC and distributes traffic
- BUM traffic: only the DF forwards to the CE (prevents duplicates)
- Split-horizon: ESI label in the MPLS stack prevents loops (PE does not forward back to same ES)

### Single-Active Multihoming

- Only one PE (the DF) forwards traffic at a time per ES or per VLAN
- Backup PE is standby; takes over on primary failure
- Simpler CE config (no LAG required on CE)
- Used when CE does not support multi-chassis LAG

### DF Election

```
! IOS-XR — Ethernet Segment configuration
evpn
 interface Bundle-Ether1
  ethernet-segment
   identifier type 0 00.aa.bb.cc.dd.ee.ff.00.01
   bgp route-target 0001.0001.0001
   load-balancing-mode all-active
  !
 !
```

```
# JunOS — Ethernet Segment configuration
interfaces {
    ae0 {
        esi {
            00:aa:bb:cc:dd:ee:ff:00:01:00;
            all-active;
        }
    }
}
```

## MAC Mobility

### Concept

- When a host moves between PEs, the new PE advertises a Type 2 route with a higher MAC Mobility sequence number
- Remote PEs update their forwarding to point to the new PE
- Prevents MAC duplication and ensures correct forwarding after moves

### Extended Community

```
MAC Mobility Extended Community:
+--------+--------+--------+--------+--------+--------+--------+--------+
| Type   | Sub-   | Flags  | Rsvd   |     Sequence Number (4 bytes)     |
| 0x06   | 0x00   | S=0/1  |        |                                   |
+--------+--------+--------+--------+--------+--------+--------+--------+

S (Sticky) bit: 1 = static MAC, cannot be moved
Sequence Number: Incremented on each move, higher wins
```

```
! IOS-XR — MAC mobility configuration
evpn
 evi 100
  bgp
   route-target 65000:100
  !
  ! Detect duplicate MAC (loop/flap protection)
  mac
   duplicate-detection
    move-count 5
    move-interval 180
    freeze-time 30
   !
  !
 !
```

## ARP Suppression

- PE intercepts ARP requests from CEs and replies from its local ARP/ND cache
- Cache populated from EVPN Type 2 MAC+IP routes received via BGP
- Reduces BUM flooding (ARP broadcasts) significantly
- Also called ARP/ND proxy or ARP suppression

```
! NX-OS — ARP suppression
fabric forwarding anycast-gateway-mac 0000.1111.2222
interface nve1
 member vni 10100
  suppress-arp
  ingress-replication protocol bgp

! IOS-XR — ARP suppression (automatic with IRB)
! Enabled by default when MAC+IP bindings are present in Type 2 routes
```

## EVPN-VXLAN Fabric

### NX-OS Configuration

```
! Enable features
feature nv overlay
feature bgp
feature fabric forwarding
feature interface-vlan
feature vn-segment-vlan-based

! Anycast gateway MAC (same on all VTEPs)
fabric forwarding anycast-gateway-mac 0000.1111.2222

! VLAN to VNI mapping
vlan 100
 vn-segment 10100

! L2VNI SVI (for ARP suppression and IRB)
interface Vlan100
 no shutdown
 vrf member TENANT-A
 ip address 10.100.0.1/24
 fabric forwarding mode anycast-gateway

! L3VNI for symmetric IRB
vlan 900
 vn-segment 50900

interface Vlan900
 no shutdown
 vrf member TENANT-A
 ip forward

! VRF definition with L3VNI
vrf context TENANT-A
 vni 50900
 rd auto
 address-family ipv4 unicast
  route-target both auto
  route-target both auto evpn

! NVE (VXLAN Tunnel Endpoint)
interface nve1
 no shutdown
 host-reachability protocol bgp
 source-interface loopback1
 member vni 10100
  suppress-arp
  ingress-replication protocol bgp
 member vni 50900 associate-vrf

! BGP EVPN
router bgp 65000
 neighbor 10.0.0.1
  remote-as 65000
  update-source loopback0
  address-family l2vpn evpn
   send-community extended
   route-reflector-client
```

### IOS-XR Configuration (EVPN-VXLAN)

```
! Bridge domain with EVPN
l2vpn
 bridge group EVPN-BG
  bridge-domain BD-100
   interface GigabitEthernet0/0/0/1.100
   !
   vni 10100
   !
   evi 100
   !
  !
 !

! EVPN instance
evpn
 evi 100
  bgp
   route-target import 65000:100
   route-target export 65000:100
  !
  advertise-mac
  !
 !

! BGP
router bgp 65000
 address-family l2vpn evpn
 !
 neighbor 10.0.0.1
  remote-as 65000
  update-source Loopback0
  address-family l2vpn evpn
   route-policy PASS in
   route-policy PASS out
  !
 !
```

### JunOS Configuration (EVPN-VXLAN)

```
routing-instances {
    EVPN-FABRIC {
        instance-type mac-vrf;
        protocols {
            evpn {
                encapsulation vxlan;
                default-gateway do-not-advertise;
                extended-vni-list all;
            }
        }
        vtep-source-interface lo0.0;
        service-type vlan-based;
        interface ae0.100;
        route-distinguisher 10.0.0.1:100;
        vrf-target target:65000:100;
        vlans {
            vlan-100 {
                vlan-id 100;
                interface ae0.100;
                vxlan {
                    vni 10100;
                    ingress-node-replication;
                }
            }
        }
    }
}
```

## EVPN-MPLS

```
! IOS-XR — EVPN over MPLS (instead of VXLAN)
evpn
 evi 100
  bgp
   route-target import 65000:100
   route-target export 65000:100
  !
  advertise-mac
 !

l2vpn
 bridge group EVPN-MPLS-BG
  bridge-domain EVPN-MPLS-BD
   interface GigabitEthernet0/0/0/1.100
   !
   ! No VNI — uses MPLS labels for transport
   evi 100
   !
  !
 !
```

## Symmetric vs Asymmetric IRB

### Asymmetric IRB

- **Ingress PE:** Performs both L2 lookup (source VLAN) and L3 routing (destination VLAN)
- **Egress PE:** Performs only L2 lookup (destination VLAN to port)
- **Requirement:** Every PE must have every VLAN/VNI configured (even if no local hosts)
- **Label/VNI:** Uses destination L2 VNI for encapsulation

```
Packet flow (asymmetric):
  Ingress PE: Route from VLAN 100 → VLAN 200
              Encapsulate with VNI 10200 (destination L2 VNI)
  Egress PE:  Decapsulate VNI 10200 → bridge to VLAN 200 port
```

### Symmetric IRB

- **Ingress PE:** Routes packet and encapsulates with L3 VNI (tenant VRF VNI)
- **Egress PE:** Decapsulates L3 VNI, routes in VRF, bridges to destination VLAN
- **Advantage:** PEs only need locally-present VLANs configured
- **Labels/VNIs:** Uses L3 VNI for inter-subnet, L2 VNI for intra-subnet

```
Packet flow (symmetric):
  Ingress PE: Route from VLAN 100, source in VRF TENANT-A
              Encapsulate with L3 VNI 50900 (tenant VRF VNI)
              Inner MAC: router MAC of egress PE
  Egress PE:  Decapsulate L3 VNI 50900 → VRF lookup
              Route to VLAN 200 → bridge to local port
```

### Comparison

| Aspect | Asymmetric | Symmetric |
|:---|:---|:---|
| VLAN config | All VLANs on all PEs | Only local VLANs needed |
| Scalability | Poor (all VLANs everywhere) | Good (VLANs only where needed) |
| Label/VNI usage | Destination L2 VNI | L3 VNI (per VRF) |
| Routing lookups | Ingress only | Ingress and egress |
| Type 2 routes | MAC+IP with L2 label | MAC+IP with L2 and L3 labels |
| Type 5 routes | Not used | Used for external prefixes |
| Inter-VRF routing | N/A | Supported via RT import/export |

## Distributed Anycast Gateway

- All VTEPs/PEs share the same gateway IP and MAC for each subnet
- Host ARPs for gateway → local PE responds (no hair-pinning to central gateway)
- MAC: configured anycast-gateway-mac (same on all PEs)
- IP: same SVI IP on all PEs (e.g., 10.100.0.1/24 on all)
- Enables optimal east-west traffic (routed locally at ingress PE)

```
! NX-OS
fabric forwarding anycast-gateway-mac 0000.1111.2222
interface Vlan100
 ip address 10.100.0.1/24
 fabric forwarding mode anycast-gateway

! JunOS
routing-instances {
    TENANT-A {
        protocols {
            evpn {
                default-gateway do-not-advertise;
            }
        }
    }
}
interfaces {
    irb {
        unit 100 {
            virtual-gateway-accept-data;
            family inet {
                address 10.100.0.1/24 {
                    virtual-gateway-address 10.100.0.1;
                }
            }
            virtual-gateway-v4-mac 00:00:11:11:22:22;
        }
    }
}
```

## Type 5 IP Prefix Routes

- Advertise IP prefixes (not just host routes from Type 2)
- Used for: external routes from border leaf, summary routes, default routes
- Gateway IP field: next-hop for recursive resolution (or 0.0.0.0 for direct)
- Enables inter-VRF route leaking via RT import/export

```
! NX-OS — Advertise IP prefixes via Type 5
router bgp 65000
 vrf TENANT-A
  address-family ipv4 unicast
   advertise l2vpn evpn
   ! Redistributes VRF routes into EVPN Type 5
   redistribute direct route-map CONNECTED
  !

! IOS-XR — Type 5 advertisement
router bgp 65000
 vrf TENANT-A
  rd auto
  address-family ipv4 unicast
   ! Advertise connected/static routes as EVPN Type 5
   redistribute connected
  !
  address-family l2vpn evpn
   advertise ipv4 unicast
  !
 !
```

## Verification Commands

### Common Across Platforms

```bash
# BGP EVPN table (all route types)
# NX-OS
show bgp l2vpn evpn

# IOS-XR
show bgp l2vpn evpn

# JunOS
show route table bgp.evpn.0

# Filter by route type
# NX-OS
show bgp l2vpn evpn route-type 2   # MAC/IP routes
show bgp l2vpn evpn route-type 5   # IP prefix routes

# IOS-XR
show bgp l2vpn evpn route-type 2
show bgp l2vpn evpn route-type 5

# EVPN instance / EVI info
# NX-OS
show nve peers
show nve vni
show l2route evpn mac all
show l2route evpn mac-ip all

# IOS-XR
show evpn evi
show evpn evi detail
show evpn ethernet-segment

# JunOS
show evpn instance
show evpn database

# Multihoming / ES status
# IOS-XR
show evpn ethernet-segment interface Bundle-Ether1 detail
show evpn ethernet-segment esi <esi> detail

# NX-OS
show nve ethernet-segment

# DF election result
# IOS-XR
show evpn ethernet-segment esi <esi> detail | include "DF"

# MAC mobility
# NX-OS
show l2route evpn mac all | include "seq"

# ARP suppression cache
# NX-OS
show ip arp suppression-cache detail
```

## Interop Considerations

- **ESI encoding:** Must match across vendors (Type 0 manual is safest for multi-vendor)
- **DF election algorithm:** Default is mod-based; all PEs on an ES must use the same algorithm
- **Control word:** Some vendors enable by default, others do not; mismatch breaks PW
- **Route type support:** Not all platforms support all route types (Type 5 requires recent firmware)
- **VXLAN vs MPLS:** Cannot mix encapsulations within the same EVI without a gateway
- **Anycast gateway MAC:** Must be identical on all VTEPs in the fabric; mismatch causes asymmetric routing failures
- **RT format:** auto-derived RTs differ across vendors; use explicit RT configuration in multi-vendor environments

## Tips

- Always use symmetric IRB for scalable EVPN-VXLAN fabrics; asymmetric requires all VLANs on all VTEPs, which defeats the purpose of distributed fabrics.
- The anycast gateway MAC must be identical everywhere; even a single mismatch causes traffic black-holes for hosts behind the misconfigured VTEP.
- For multihoming, prefer ESI Type 0 (manual) in multi-vendor environments; auto-derived ESI types vary across implementations.
- Type 1 per-ES route withdrawal is the key to fast convergence; when an ES goes down, a single BGP withdrawal invalidates all MACs on that ES across all remote PEs.
- MAC mobility sequence number prevents flip-flopping, but enable duplicate detection to catch loops; without it, a loop between two PEs will increment the sequence counter indefinitely.
- ARP suppression dramatically reduces BUM traffic but requires MAC+IP bindings (Type 2 with IP); ensure hosts send gratuitous ARPs on link-up.
- Type 5 routes are essential for external connectivity; without them, border leafs cannot inject external routes into the EVPN fabric.
- In EVPN-VXLAN fabrics, ensure the underlay MTU supports the VXLAN overhead (50 bytes for VXLAN + outer UDP/IP + outer Ethernet); set underlay MTU to at least 9216.
- Use ingress replication for BUM in small-to-medium fabrics (< 100 VTEPs); switch to multicast underlay for larger deployments to reduce PE replication load.
- When migrating from VPLS to EVPN, run both in parallel using interworking mode during transition; EVPN can import VPLS MAC state via Type 2 routes.

## See Also

- mpls, vxlan, bgp, l2vpn-services, unified-mpls, ethernet

## References

- [RFC 7432 — BGP MPLS-Based Ethernet VPN](https://www.rfc-editor.org/rfc/rfc7432)
- [RFC 7209 — Requirements for Ethernet VPN](https://www.rfc-editor.org/rfc/rfc7209)
- [RFC 8365 — A Network Virtualization Overlay Solution Using EVPN](https://www.rfc-editor.org/rfc/rfc8365)
- [RFC 9135 — Integrated Routing and Bridging in EVPN](https://www.rfc-editor.org/rfc/rfc9135)
- [RFC 9136 — IP Prefix Advertisement in EVPN](https://www.rfc-editor.org/rfc/rfc9136)
- [RFC 8584 — Framework for EVPN DF Election](https://www.rfc-editor.org/rfc/rfc8584)
- [RFC 7432bis — EVPN (updated)](https://datatracker.ietf.org/doc/draft-ietf-bess-rfc7432bis/)
- [Cisco NX-OS EVPN Configuration Guide](https://www.cisco.com/c/en/us/td/docs/switches/datacenter/nexus9000/sw/93x/vxlan/configuration/guide/b-cisco-nexus-9000-series-nx-os-vxlan-configuration-guide-93x.html)
- [Cisco IOS-XR EVPN Configuration Guide](https://www.cisco.com/c/en/us/td/docs/routers/asr9000/software/l2vpn/configuration/guide/l2vpn-cg.html)
- [Juniper EVPN-VXLAN Configuration Guide](https://www.juniper.net/documentation/us/en/software/junos/evpn-vxlan/topics/concept/evpn-vxlan-overview.html)
