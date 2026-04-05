# Multicast VPN (mVPN)

Multicast VPN extends multicast delivery into L3VPN environments, solving the fundamental problem of transporting customer multicast across a shared provider backbone using either GRE-based tunnels (Draft Rosen) or BGP-signaled trees (NG-mVPN) with multiple profile options trading simplicity against efficiency.

## Core Concepts

### The Multicast-in-VPN Problem

```
Without mVPN, provider choices are:

1. Replicate multicast per-VRF on every PE      -> O(PEs x VRFs) state, unscalable
2. Ingress replicate unicast copies to each PE   -> O(receivers) bandwidth at source PE
3. Use native provider multicast per-VRF group   -> provider PIM state per customer group

mVPN solution:
  - Aggregate customer multicast into provider-level tunnels
  - One provider tunnel per VPN (default MDT) or per high-rate source (data MDT)
  - Customer PIM runs inside VRF; provider PIM runs in global table
  - Separation of customer and provider multicast state
```

### MDT Architecture (Multicast Distribution Tree)

```
MDT types:

Default MDT:
  - Always-on tunnel connecting all PEs in a given VPN
  - All VRF multicast traffic uses this tunnel by default
  - All PEs join the default MDT group (even without active receivers)
  - Low-bandwidth, all-PE reachability (control plane + low-rate data)

Data MDT:
  - Dynamically created for high-bandwidth sources
  - Only PEs with active receivers join the data MDT
  - Triggered when source exceeds configured threshold (kbps)
  - Reduces bandwidth waste on PEs without receivers
  - Announced via MDT Join TLV (Draft Rosen) or Type 3/4 routes (NG-mVPN)

                    PE1 ──── P ──── P ──── PE2
                     │                      │
  Default MDT:  ═════╪══════════════════════╪═══════  (all PEs, always)
                     │                      │
  Data MDT:    ------╪---------->-----------╪------   (source PE -> receiver PEs only)
                     │                      │
                   VRF-A                  VRF-A
                 (source)              (receivers)
```

## Draft Rosen mVPN (GRE-Based)

### Architecture

```
Draft Rosen (RFC 6037) uses GRE encapsulation with provider PIM:

Customer packet:
  [CE multicast] -> [VRF on PE] -> [GRE encap] -> [Provider PIM tree] -> [Remote PEs]

Encapsulation:
  +-------------------+
  | Outer IP Header   |  src=PE-loopback, dst=MDT-group (e.g., 239.1.1.1)
  +-------------------+
  | GRE Header        |  Protocol type = 0x0800 (IPv4)
  +-------------------+
  | Inner IP Header   |  src=customer-source, dst=customer-group
  +-------------------+
  | Payload           |
  +-------------------+

Key characteristics:
  - Provider PIM-SM or PIM-SSM builds the MDT tree in global table
  - Each VPN gets a unique default MDT group address
  - Customer PIM adjacencies form over the MDT tunnel
  - PIM hellos, joins, prunes all ride the default MDT
  - GRE adds 24 bytes overhead (outer IP + GRE header)
```

### IOS-XE Draft Rosen Configuration

```
! VRF definition with MDT
vrf definition CUST-A
 rd 65000:100
 route-target export 65000:100
 route-target import 65000:100
 !
 address-family ipv4
  mdt default 239.1.1.1                  ! Default MDT group
  mdt data 239.1.2.0 0.0.0.255          ! Data MDT group range
  mdt data threshold 100                 ! Trigger data MDT at 100 kbps
  mdt log-reuse                          ! Log data MDT creation/deletion
 exit-address-family

! Enable PIM in VRF
interface GigabitEthernet0/0/0
 vrf forwarding CUST-A
 ip address 10.1.1.1 255.255.255.0
 ip pim sparse-mode

! RP for VRF (customer RP)
ip pim vrf CUST-A rp-address 10.255.255.1

! Provider PIM (global table)
ip pim ssm default                       ! SSM for MDT groups (recommended)
ip pim rp-address 10.0.0.99              ! RP for default MDT if using PIM-SM
ip multicast-routing                      ! Global multicast routing
ip multicast-routing vrf CUST-A           ! VRF multicast routing
```

### IOS-XR Draft Rosen Configuration

```
vrf CUST-A
 address-family ipv4 unicast
  import route-target 65000:100
  export route-target 65000:100

multicast-routing
 address-family ipv4
  interface Loopback0
   enable
  interface GigabitEthernet0/0/0/0
   enable
 !
 vrf CUST-A
  address-family ipv4
   mdt default ipv4 239.1.1.1
   mdt data 239.1.2.0/24 threshold 100
   interface GigabitEthernet0/0/0/1
    enable
   !
  !
 !

router pim
 vrf CUST-A
  address-family ipv4
   rp-address 10.255.255.1
```

## NG-mVPN (BGP-Based, RFC 6513/6514)

### BGP SAFI 5 Route Types

```
NG-mVPN uses BGP with AFI 1 (IPv4) or AFI 2 (IPv6), SAFI 5 (MCAST-VPN):

Type  Name                            Purpose
──────────────────────────────────────────────────────────────────────────────
 1    Intra-AS I-PMSI A-D             Advertises default tunnel binding per VPN
 2    Inter-AS I-PMSI A-D             Extends Type 1 across AS boundaries
 3    S-PMSI A-D                      Advertises selective (data) tunnel for (S,G)/(*, G)
 4    Leaf A-D                        Receiver PE signals interest in a selective tunnel
 5    Source Active A-D               Replaces MSDP: announces active sources
 6    Shared Tree Join                (*,G) join toward RP (C-multicast signaling)
 7    Source Tree Join                (S,G) join toward source (C-multicast signaling)

A-D = Auto-Discovery
I-PMSI = Inclusive Provider Multicast Service Interface (= default MDT)
S-PMSI = Selective Provider Multicast Service Interface (= data MDT)
C-multicast = Customer multicast
```

### Route Type Details

```
Type 1 — Intra-AS I-PMSI A-D:
  NLRI: [RD, Originator-PE]
  Carried in: PMSI Tunnel Attribute (PTA) with tunnel type + tunnel ID
  Purpose: Each PE advertises its default tunnel per VPN
  PTA tunnel types: PIM-SM (0), PIM-SSM (1), PIM-BIDIR (2),
                    IR (6), mLDP P2MP (7), mLDP MP2MP (8)

Type 2 — Inter-AS I-PMSI A-D:
  NLRI: [RD, Source-AS]
  Purpose: Extends I-PMSI discovery across AS boundaries

Type 3 — S-PMSI A-D:
  NLRI: [RD, C-Source, C-Group, Originator-PE]
  Purpose: Source PE advertises a selective tunnel for a specific (C-S, C-G)
  PTA contains the selective tunnel binding

Type 4 — Leaf A-D:
  NLRI: [Route-Key from Type 3, Originator-PE]
  Purpose: Receiver PE signals it wants to join the selective tunnel
  Required for: IR, mLDP P2MP, RSVP-TE P2MP (leaf-initiated joins)

Type 5 — Source Active A-D:
  NLRI: [RD, C-Source, C-Group]
  Purpose: Replaces MSDP Source-Active messages
  Source PE announces active multicast sources to all PEs in the VPN

Type 6 — Shared Tree Join (C-multicast):
  NLRI: [RD, C-Source, C-Group, RP-Address]
  Purpose: Signals (*,G) join toward the customer RP
  Maps to PIM (*,G) join in the VRF overlay

Type 7 — Source Tree Join (C-multicast):
  NLRI: [RD, C-Source, C-Group]
  Purpose: Signals (S,G) join toward the customer source
  Maps to PIM (S,G) join in the VRF overlay
```

## mVPN Profiles

### Profile Matrix

```
Profile  P-tunnel Technology      Default Tunnel    Data Tunnel       Key Trait
─────────────────────────────────────────────────────────────────────────────────
  0      PIM/GRE (Draft Rosen)    PIM-SM/SSM tree   PIM data MDT      Legacy, simple
  1      mLDP P2MP                mLDP P2MP         mLDP P2MP         Label-switched
  2      mLDP MP2MP               mLDP MP2MP        N/A               Bidirectional
  3      PIM-SSM (with BGP AD)    PIM-SSM tree      PIM-SSM tree      BGP + PIM
  4      PIM-SM (with BGP AD)     PIM-SM tree       PIM-SM tree       BGP + PIM
  5      mLDP MP2MP (default)     mLDP MP2MP        mLDP P2MP         Hybrid
         + mLDP P2MP (data)
  6      PIM-SSM + Ingress Rep    PIM-SSM tree      IR                PIM default + IR data
  7      Ingress Replication      IR                IR                No P-multicast needed
  8      mLDP P2MP (partitioned)  mLDP P2MP         mLDP P2MP         Per-VRF partitioned
  9      PIM/GRE (partitioned)    PIM-SM/SSM        PIM data MDT      Per-VRF partitioned
 10      mLDP P2MP + IR           mLDP P2MP         IR                mLDP default + IR data
 11      RSVP-TE P2MP             RSVP-TE P2MP      RSVP-TE P2MP      TE-based
 12      IR + mLDP P2MP           IR                mLDP P2MP         IR default + mLDP data
 13      mLDP (partitioned + IR)  mLDP MP2MP        IR                Partitioned + IR
 14      IR + IR                  IR                IR (partitioned)  Full IR, EVPN-ready

Common profiles in production:
  Profile 0:  Legacy Draft Rosen, widely deployed, simple
  Profile 6:  PIM-SSM default MDT + IR for selective (efficient data plane)
  Profile 7:  Pure ingress replication (no provider multicast needed)
  Profile 11: RSVP-TE P2MP tunnels (traffic engineering for multicast)
  Profile 12: IR default + mLDP data (balance of simplicity and efficiency)
  Profile 14: Full IR (popular with EVPN/VXLAN fabrics)
```

### Profile 0 — Draft Rosen GRE (IOS-XE)

```
! Profile 0 is the classic Draft Rosen configuration (see above)
! Uses mdt default/data under VRF address-family
! No BGP MVPN address family needed
! Provider PIM builds the MDT tree

ip multicast-routing
ip multicast-routing vrf CUST-A

vrf definition CUST-A
 rd 65000:100
 route-target export 65000:100
 route-target import 65000:100
 address-family ipv4
  mdt default 239.1.1.1
  mdt data 239.1.2.0 0.0.0.255 threshold 100
 exit-address-family
```

### Profile 6 — PIM-SSM Default + IR Data (IOS-XR)

```
! Default tunnel: PIM-SSM in provider core
! Data tunnel: Ingress Replication (unicast copies)
! Requires BGP MVPN address family

router bgp 65000
 address-family ipv4 mvpn
 !
 vrf CUST-A
  address-family ipv4 mvpn
  !

multicast-routing
 address-family ipv4
  interface Loopback0
   enable
  mdt source Loopback0
 !
 vrf CUST-A
  address-family ipv4
   mdt default ipv4 232.1.1.1                  ! SSM group
   mdt source Loopback0
   mdt data ingress-replication 10              ! IR for data MDT, max 10 tunnels
   interface GigabitEthernet0/0/0/1
    enable

router pim
 vrf CUST-A
  address-family ipv4
   rp-address 10.255.255.1
   rpf topology route-policy MVPN-RPF          ! RPF for mVPN
```

### Profile 7 — Pure Ingress Replication (IOS-XR)

```
! No provider multicast needed at all
! All replication done by ingress PE via unicast copies
! Simple to deploy, scales for moderate receiver counts
! Popular in DC/EVPN environments

router bgp 65000
 address-family ipv4 mvpn
 !
 vrf CUST-A
  address-family ipv4 mvpn
  !

multicast-routing
 vrf CUST-A
  address-family ipv4
   mdt default ingress-replication
   mdt data ingress-replication 10
   interface GigabitEthernet0/0/0/1
    enable

router pim
 vrf CUST-A
  address-family ipv4
   rp-address 10.255.255.1
```

### Profile 14 — Full IR Partitioned (IOS-XR)

```
! Partitioned IR: each PE only sends to PEs with active receivers
! No provider multicast infrastructure required
! Integrates well with EVPN/VXLAN

router bgp 65000
 address-family ipv4 mvpn
 !
 vrf CUST-A
  address-family ipv4 mvpn
  !

multicast-routing
 vrf CUST-A
  address-family ipv4
   mdt default ingress-replication
   mdt partitioned ingress-replication
   mdt data ingress-replication 10
   interface GigabitEthernet0/0/0/1
    enable
```

### Profile 0 — Draft Rosen GRE (IOS-XR)

```
multicast-routing
 address-family ipv4
  interface Loopback0
   enable
  !
 vrf CUST-A
  address-family ipv4
   mdt default ipv4 239.1.1.1
   mdt data 239.1.2.0/24 threshold 100
   interface GigabitEthernet0/0/0/1
    enable

router pim
 address-family ipv4
  interface Loopback0
   enable
  interface GigabitEthernet0/0/0/0
   enable
 !
 vrf CUST-A
  address-family ipv4
   rp-address 10.255.255.1
```

## PIM in VRF

```
! PIM operates independently per VRF
! Each VRF has its own:
!   - PIM neighbor table
!   - Multicast routing table (mroute)
!   - RP configuration
!   - RPF lookups (against VRF RIB)
!   - IGMP membership

! IOS-XE: PIM in VRF
ip pim vrf CUST-A rp-address 10.255.255.1
ip pim vrf CUST-A spt-threshold infinity           ! Keep on shared tree
ip pim vrf CUST-A register-source Loopback100

! IOS-XR: PIM in VRF
router pim
 vrf CUST-A
  address-family ipv4
   rp-address 10.255.255.1
   spt-threshold infinity
   log neighbor changes

! Verification
show ip pim vrf CUST-A neighbor                     ! IOS-XE
show ip mroute vrf CUST-A                           ! IOS-XE
show pim vrf CUST-A neighbor                        ! IOS-XR
show mrib vrf CUST-A route                          ! IOS-XR
```

## mVPN with VXLAN/EVPN

```
mVPN integration with EVPN/VXLAN fabric:

Approach 1: Profile 14 (IR) + EVPN
  - Ingress replication for BUM traffic already used by EVPN
  - mVPN Type 1/3/4 routes carry VXLAN tunnel info
  - No separate multicast underlay needed
  - Natural fit for DC overlay multicast

Approach 2: Profile 6/7 + EVPN
  - EVPN handles L2 BUM via IR or multicast underlay
  - mVPN handles L3 multicast across VRFs
  - Separate control planes for L2 (EVPN) and L3 (mVPN)

EVPN Multicast Handling (without mVPN):
  - EVPN Type 6 (Multicast Membership) for IGMP sync
  - EVPN Type 7 (Multicast Join Synch) for PIM sync
  - EVPN Type 8 (Multicast Leave Synch) for prune sync
  - These handle L2 multicast optimization within EVPN fabric

Combined EVPN + mVPN:
  - L2 multicast: EVPN Type 6/7/8 (IGMP proxy, optimized flooding)
  - L3 multicast: mVPN BGP routes (Type 1-7), PMSI tunnel attribute
  - VXLAN carries both L2 and L3 multicast encapsulated traffic
```

## Inter-AS mVPN

```
Inter-AS mVPN Options:

Option A (Back-to-back VRF):
  - ASBR has VRF on inter-AS link
  - Customer multicast crosses as native PIM in VRF
  - Simple but does not scale (per-VRF interface on ASBR)

Option B (VPNv4 + mVPN routes at ASBR):
  - ASBR exchanges BGP VPNv4 and mVPN routes
  - Type 2 Inter-AS I-PMSI A-D route extends tunnel discovery
  - Segmented P-tunnel: each AS builds its own P-tunnel
  - ASBR stitches tunnels at AS boundary

Option C (Hierarchical with RR):
  - Multihop BGP between RRs in different ASes
  - VPNv4 and mVPN routes carried end-to-end
  - Provider tunnel spans ASes (requires inter-AS PIM or mLDP)
  - Most scalable but most complex

Inter-AS challenges:
  - RPF check across AS boundary (source in remote AS)
  - P-tunnel stitching at ASBR
  - BGP next-hop resolution across ASes
  - Data MDT signaling across AS boundaries
```

## Verification Commands

### IOS-XE

```
! mVPN tunnel state
show ip pim vrf CUST-A mdt send                    ! MDT groups this PE is sending
show ip pim vrf CUST-A mdt receive                  ! MDT groups this PE receives
show ip pim vrf CUST-A mdt history                  ! Data MDT creation history

! BGP mVPN routes (NG-mVPN)
show bgp ipv4 mvpn all                              ! All mVPN routes
show bgp ipv4 mvpn all route-type 1                 ! Type 1 (I-PMSI AD)
show bgp ipv4 mvpn all route-type 3                 ! Type 3 (S-PMSI AD)
show bgp ipv4 mvpn all route-type 5                 ! Type 5 (Source Active)
show bgp ipv4 mvpn vrf CUST-A                       ! mVPN routes for specific VRF

! VRF multicast state
show ip mroute vrf CUST-A                           ! Multicast routing table in VRF
show ip mroute vrf CUST-A count                     ! Packet/byte counts
show ip mroute vrf CUST-A active                    ! Active sources
show ip pim vrf CUST-A neighbor                     ! PIM neighbors in VRF
show ip pim vrf CUST-A rp mapping                   ! RP mappings in VRF
show ip rpf vrf CUST-A <source-ip>                  ! RPF check in VRF context
show ip igmp vrf CUST-A groups                      ! IGMP groups in VRF

! Provider tunnel state
show ip mroute 239.1.1.1                            ! Default MDT in global table
show ip pim neighbor                                ! Provider PIM neighbors
show mpls mldp database                             ! mLDP tunnel state (if using mLDP)
```

### IOS-XR

```
! mVPN tunnel state
show mvpn vrf CUST-A                                ! mVPN summary
show mvpn vrf CUST-A database                       ! mVPN tunnel database
show mvpn vrf CUST-A context                        ! mVPN context (tunnel bindings)
show mvpn vrf CUST-A pe                             ! PEs in the mVPN

! BGP mVPN routes
show bgp ipv4 mvpn                                  ! All mVPN routes
show bgp ipv4 mvpn rd 65000:100                     ! mVPN routes for specific RD
show bgp ipv4 mvpn route-type 1                     ! Type 1 routes
show bgp ipv4 mvpn route-type 3                     ! Type 3 routes
show bgp ipv4 mvpn route-type 7                     ! Type 7 (Source Tree Join)

! VRF multicast state
show mrib vrf CUST-A route                          ! Multicast RIB in VRF
show mfib vrf CUST-A route                          ! Multicast FIB in VRF
show pim vrf CUST-A neighbor                        ! PIM neighbors in VRF
show pim vrf CUST-A topology                        ! PIM topology table
show pim vrf CUST-A rp mapping                      ! RP mappings
show igmp vrf CUST-A groups                         ! IGMP groups
show mrib vrf CUST-A route detail                   ! Detailed mroute with interfaces

! Provider tunnel
show pim topology 239.1.1.1                         ! Provider MDT tree (PIM-based)
show mpls mldp database                             ! mLDP database (if mLDP profile)
show mpls mldp neighbors                            ! mLDP neighbor state
```

### Troubleshooting Workflow

```
mVPN troubleshooting sequence:

1. Verify VRF multicast routing is enabled
   show ip multicast-routing vrf CUST-A             ! IOS-XE
   show multicast-routing vrf CUST-A                ! IOS-XR

2. Check PIM neighbors in VRF (should see remote PEs over MDT)
   show ip pim vrf CUST-A neighbor

3. Verify RP configuration in VRF
   show ip pim vrf CUST-A rp mapping

4. Check IGMP membership on receiver-side PE
   show ip igmp vrf CUST-A groups

5. Verify mroute state in VRF
   show ip mroute vrf CUST-A
   ! Look for (S,G) with incoming interface = MDT tunnel
   ! Outgoing interface list should include CE-facing interface

6. Check provider tunnel (default MDT)
   show ip pim vrf CUST-A mdt send                  ! IOS-XE
   show mvpn vrf CUST-A database                    ! IOS-XR

7. Verify BGP mVPN routes (NG-mVPN only)
   show bgp ipv4 mvpn all
   ! Type 1 must exist for each PE in the VPN
   ! Type 7 must exist for each (S,G) join

8. Check RPF in VRF context
   show ip rpf vrf CUST-A <source>
   ! RPF interface should be the MDT tunnel or MPLS interface

9. Verify provider multicast tree (if PIM-based profile)
   show ip mroute <MDT-group>                       ! Must have OIF to remote PEs

10. Check data MDT (if configured)
    show ip pim vrf CUST-A mdt receive              ! Should show data MDT groups
    show mvpn vrf CUST-A database segment-type 2    ! IOS-XR, Type 2 = S-PMSI
```

## Tips

- Start with Profile 0 (Draft Rosen) for initial mVPN deployments; it requires no BGP mVPN address family and is the simplest to troubleshoot.
- Profile 7 (pure IR) is the easiest NG-mVPN profile; it eliminates all provider multicast dependencies but does not scale well beyond 20-30 receiver PEs per group.
- Profile 14 (partitioned IR) improves on Profile 7 by only replicating to PEs with active receivers, making it the preferred choice for EVPN/VXLAN data center fabrics.
- Always verify PIM neighbors across the MDT tunnel; if remote PEs do not appear as PIM neighbors in the VRF, the default MDT is broken.
- Data MDTs save bandwidth but add complexity; only configure them when high-bandwidth sources (video, IPTV) cause unnecessary replication on the default MDT.
- For PIM-SSM based profiles (3, 6), use SSM groups (232.0.0.0/8) for MDT groups to avoid RP dependency in the provider core.
- RPF failures in VRF context are the most common mVPN issue; the RPF check must resolve through the MDT tunnel or the MPLS/GRE path to the source PE.
- When migrating from Draft Rosen to NG-mVPN, run both in parallel during transition; BGP mVPN routes and GRE MDT can coexist temporarily.
- Inter-AS mVPN Option B is the practical choice for most multi-AS deployments; Option C is theoretically cleaner but operationally much harder.
- Monitor data MDT creation with `mdt log-reuse` (IOS-XE) to track tunnel churn and verify threshold tuning.

## See Also

- multicast-routing, bgp, mpls-vpn, vxlan, igmp, vrf, segment-routing, fabric-multicast

## References

- [RFC 6513 — Multicast in MPLS/BGP IP VPNs](https://www.rfc-editor.org/rfc/rfc6513)
- [RFC 6514 — BGP Encodings and Procedures for Multicast in MPLS/BGP IP VPNs](https://www.rfc-editor.org/rfc/rfc6514)
- [RFC 6515 — IPv4 and IPv6 Infrastructure Addresses in BGP Updates for Multicast VPN](https://www.rfc-editor.org/rfc/rfc6515)
- [RFC 6037 — Cisco Systems' Solution for Multicast in BGP/MPLS IP VPNs (Draft Rosen)](https://www.rfc-editor.org/rfc/rfc6037)
- [RFC 6625 — Wildcards in Multicast VPN Auto-Discovery Routes](https://www.rfc-editor.org/rfc/rfc6625)
- [RFC 7716 — Global Table Multicast with BGP Multicast VPN (BGP-MVPN)](https://www.rfc-editor.org/rfc/rfc7716)
- [RFC 7761 — PIM-SM Protocol Specification](https://www.rfc-editor.org/rfc/rfc7761)
- [Cisco IOS-XR mVPN Configuration Guide](https://www.cisco.com/c/en/us/td/docs/routers/asr9000/software/multicast/configuration/guide/b-multicast-cg-asr9k.html)
- [Cisco IOS-XE mVPN Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/ipmulti_pim/configuration/xe-16/imc-pim-xe-16-book/imc-mvpn.html)
- [Juniper mVPN Documentation](https://www.juniper.net/documentation/us/en/software/junos/multicast/topics/concept/multicast-vpn-overview.html)
