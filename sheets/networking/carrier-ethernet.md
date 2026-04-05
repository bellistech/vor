# Carrier Ethernet (MEF Services, OAM, EVC & Provider Backbone)

Deploying MEF-defined Ethernet services (E-Line, E-LAN, E-Tree) over provider networks using Ethernet OAM (CFM/Y.1731), EVCs, bandwidth profiles, REP/G.8032 ring protection, Provider Backbone Bridging (PBB), VPLS, pseudowires, and MPLS-based transport for scalable carrier-grade Ethernet delivery.

## MEF Service Types

```
MEF defines four Ethernet service types based on connectivity:

Service    Connectivity    Topology      Equivalent     Use Case
──────────────────────────────────────────────────────────────────────────
E-Line     Point-to-Point  2 UNIs       Leased line    WAN link, PW
E-LAN      Multipoint      N UNIs       LAN extension  Multi-site L2 VPN
E-Tree     Rooted MP       Root + Leaf  Hub-spoke      Video distribution
E-Access   Point-to-Point  UNI-ENNI     Access link    Wholesale access

                E-Line                    E-LAN
           UNI ══════ UNI          UNI ══╦══ UNI
                                         ║
                                   UNI ══╩══ UNI

                E-Tree                    E-Access
           Root ══╦══ Leaf         UNI ══════ ENNI
                  ║                          (to partner SP)
           Root ══╩══ Leaf
           (Leaf-to-Leaf blocked)
```

## EVC (Ethernet Virtual Connection)

```
An EVC is a logical association between UNIs that defines the service:
  - Which UNIs participate
  - Bandwidth profile per UNI
  - CoS attributes
  - Performance objectives (delay, loss, jitter)

EVC Types (by UNI count):
  Point-to-Point EVC:  exactly 2 UNIs (E-Line)
  Multipoint EVC:      2 or more UNIs (E-LAN, E-Tree)

Bundling (multiple EVCs per UNI):
  All-to-One:    all CE VLANs map to a single EVC
  Bundling:      specific CE VLANs map to specific EVCs
  Multiplexing:  multiple EVCs on one UNI, each with VLAN mapping

! IOS-XE EVC configuration
ethernet evc CUST-A-ELINE
 ! EVC name is administrative label

interface GigabitEthernet0/0/0
 service instance 100 ethernet CUST-A-ELINE
  encapsulation dot1q 100
  rewrite ingress tag pop 1 symmetric
  bridge-domain 100
```

## UNI and ENNI

```
UNI (User-Network Interface):
  Physical demarcation between customer and provider network.
  Defined by MEF 13 (UNI Type 1, 2, 3) and MEF 20.

  UNI Attributes:
    - Physical medium (1G, 10G, 100G)
    - UNI-ID (unique identifier)
    - Bundling/multiplexing mode
    - CE-VLAN ID preservation (yes/no)
    - Maximum number of EVCs
    - Ingress/egress bandwidth profile

ENNI (External Network-Network Interface):
  Interconnection point between two provider networks.
  Defined by MEF 26.

  ENNI uses operator VLANs (S-VLAN/B-VLAN) to separate
  traffic from different customers and services.

        Customer A                              Customer A
  ┌──────┐    ┌──────────────┐    ┌──────────────┐    ┌──────┐
  │ CE-A ├─UNI┤   SP Alpha   ├ENNI┤   SP Beta    ├UNI─┤ CE-A │
  └──────┘    └──────────────┘    └──────────────┘    └──────┘

  UNI: C-VLAN tagged (customer VLAN)
  ENNI: S-VLAN tagged (operator VLAN, QinQ)
```

## Bandwidth Profiles

```
MEF bandwidth profiles use token bucket policing (RFC 2698 trTCM):

Parameter          Description
────────────────────────────────────────────────────────────────
CIR               Committed Information Rate (guaranteed)
CBS               Committed Burst Size
EIR               Excess Information Rate (best-effort above CIR)
EBS               Excess Burst Size
CM                Color Mode (color-blind or color-aware)
CF                Coupling Flag (allow unused CIR tokens to fill EIR bucket)

Three colors:
  Green:  within CIR/CBS — guaranteed delivery
  Yellow: within EIR/EBS — delivered if capacity available
  Red:    exceeds both — dropped

! IOS-XE bandwidth profile (policer)
policy-map CUST-A-BW-PROFILE
 class class-default
  police cir 50m cbs 625000 eir 25m ebs 312500
   conform-action transmit
   exceed-action set-cos-transmit 0
   violate-action drop

! Applied to service instance
interface GigabitEthernet0/0/0
 service instance 100 ethernet CUST-A-ELINE
  encapsulation dot1q 100
  service-policy input CUST-A-BW-PROFILE
```

## Carrier Ethernet Performance Attributes

```
MEF defines SLA performance metrics per CoS:

Metric                        Voice/RT     Data Standard    Data Best-Effort
──────────────────────────────────────────────────────────────────────────────
Frame Delay (one-way)         < 10 ms      < 30 ms          unspecified
Mean Frame Delay              < 5 ms       < 20 ms          unspecified
Frame Delay Range (jitter)    < 3 ms       < 10 ms          unspecified
Frame Loss Ratio              < 0.01%      < 0.1%           < 1%
Availability                  99.999%      99.99%           99.9%

Performance tiers (MEF 23.2):
  - Performance Tier 1 (Metro): < 10 ms delay
  - Performance Tier 2 (Regional): < 20 ms delay
  - Performance Tier 3 (Continental): < 40 ms delay
```

## Ethernet OAM — 802.1ag CFM

### CFM Domain Hierarchy

```
CFM operates at three nested maintenance levels:

Level 7 ┌─────────────────────────────────────────────────┐  Customer MD
        │  MEP ─────────────── MEP                        │
Level 4 │  ┌──────────────────────────────────────────┐   │  Provider MD
        │  │  MEP ──── MIP ──── MIP ──── MEP          │   │
Level 1 │  │  ┌───────────────────────────────┐       │   │  Operator MD
        │  │  │  MEP ──── MEP                 │       │   │
        │  │  └───────────────────────────────┘       │   │
        └──┴──────────────────────────────────────────┴───┘

MD  = Maintenance Domain (defines administrative boundary)
MA  = Maintenance Association (specific service within an MD)
MEP = Maintenance End Point (generates/terminates OAM frames)
MIP = Maintenance Intermediate Point (responds to OAM, does not initiate)

Level assignment (convention):
  Level 0-2: Operator (link-level OAM)
  Level 3-4: Provider (end-to-end across provider network)
  Level 5-7: Customer (end-to-end across customer service)

Higher levels can see through lower levels but not vice versa.
```

### CFM Configuration — IOS-XE

```
! Define maintenance domain
ethernet cfm domain PROVIDER level 4
 service CUST-A-SERVICE evc CUST-A-ELINE
  continuity-check
  continuity-check interval 1s

! MEP on service instance
interface GigabitEthernet0/0/0
 service instance 100 ethernet CUST-A-ELINE
  cfm mep domain PROVIDER mpid 1

! Configure remote MEP for CC (Continuity Check)
ethernet cfm domain PROVIDER level 4
 service CUST-A-SERVICE evc CUST-A-ELINE
  continuity-check
  mep crosscheck mpid 2

! Verification
show ethernet cfm maintenance-points remote
show ethernet cfm errors
show ethernet cfm statistics
```

### CFM Messages

```
Message      Direction        Purpose                    Interval
──────────────────────────────────────────────────────────────────────
CCM          MEP → all MEPs   Continuity check           3.3ms to 10min
             (multicast)      (heartbeat, fault detect)  (typically 1s)

LBM/LBR      MEP → MEP/MIP   Loopback (L2 ping)         On demand
             (unicast)

LTM/LTR      MEP → MEP/MIP   Linktrace (L2 traceroute)  On demand
             (multicast/uni)

AIS          MEP → MEPs       Alarm Indication Signal    1s or 1min
             (toward client)  (suppress alarms upstream)

! CFM loopback (L2 ping)
ping ethernet mpid 2 domain PROVIDER vlan 100

! CFM linktrace (L2 traceroute)
traceroute ethernet mpid 2 domain PROVIDER vlan 100
```

## Ethernet OAM — Y.1731

```
Y.1731 extends CFM with performance monitoring:

Function                     Measurement              Protocol
──────────────────────────────────────────────────────────────────────
Delay Measurement (DM)       One-way/two-way delay    DMM/DMR
Synthetic Loss Measurement   Frame loss ratio         SLM/SLR
Loss Measurement (LM)        Near-end/far-end loss    LMM/LMR
Throughput Measurement       Available bandwidth      TST frames
Availability                 Service availability %   Derived from CCM

! IOS-XE Y.1731 delay measurement
ethernet cfm domain PROVIDER level 4
 service CUST-A-SERVICE evc CUST-A-ELINE
  continuity-check

! Start delay measurement to remote MEP
ethernet cfm delay-measurement domain PROVIDER mpid 2

! Start synthetic loss measurement
ethernet cfm slm domain PROVIDER mpid 2

! Verification
show ethernet cfm delay-measurement
show ethernet cfm slm
show ethernet cfm loss-measurement
```

## REP (Resilient Ethernet Protocol)

```
Cisco proprietary ring protection for Ethernet.
Alternative to STP for ring topologies. Sub-50ms convergence.

           ┌──── SW-A ────┐
           │               │
  [primary edge]    [alternate port ─ blocked]
           │               │
      SW-D │               │ SW-B
           │               │
           └──── SW-C ────┘
  [secondary edge]

REP blocks one port (alternate) to prevent loops.
On failure, unblocks alternate port in < 50ms.

! REP configuration
interface GigabitEthernet0/1
 rep segment 1 edge primary

interface GigabitEthernet0/2
 rep segment 1

! On another switch
interface GigabitEthernet0/1
 rep segment 1

interface GigabitEthernet0/2
 rep segment 1 edge

! Verification
show rep topology
show rep topology segment 1 detail
```

## G.8032 ERP (Ethernet Ring Protection)

```
ITU-T standard ring protection. Sub-50ms switchover.
Vendor-neutral alternative to REP. Uses R-APS (Ring APS) protocol.

States:
  Idle:        normal operation, RPL blocked
  Protection:  failure detected, RPL unblocked, failed link blocked
  MS/FS:       manual switch / forced switch (admin override)

RPL = Ring Protection Link (the port that is blocked in normal state)
RPL Owner = the node responsible for blocking/unblocking the RPL

            ┌──── Node A ────┐
            │    (RPL Owner)  │
   RPL ─────┤                │
   [blocked] │               │
        Node D               Node B
            │                │
            └──── Node C ────┘

On link failure between B and C:
  1. B and C detect failure (< 10ms)
  2. B sends R-APS(SF) — Signal Fail
  3. RPL Owner (A) receives R-APS(SF)
  4. A unblocks RPL (< 50ms total)
  5. All nodes flush FDB
  6. Traffic reroutes around the ring

! IOS-XE G.8032 configuration
ethernet ring g8032 RING1
 port0 interface GigabitEthernet0/1
 port1 interface GigabitEthernet0/2
 instance 1
  description "VLAN 100 ring protection"
  inclusion-list vlan-ids 100
  rpl-owner port0
  aps-channel

! Verification
show ethernet ring g8032 status
show ethernet ring g8032 brief
show ethernet ring g8032 statistics
```

## Provider Backbone Bridging (PBB / 802.1ah)

```
PBB (MAC-in-MAC) encapsulates customer Ethernet frames inside
provider backbone MAC headers, solving the 4K VLAN limit and
hiding customer MACs from the provider core.

Customer Frame:
  [C-DA][C-SA][C-VLAN][Payload]

PBB Encapsulated Frame (MAC-in-MAC):
  [B-DA][B-SA][B-VLAN][I-SID][C-DA][C-SA][C-VLAN][Payload]
  └──── backbone ────┘ └───┘ └──── original customer frame ────┘
                        │
                   24-bit I-SID (Instance Service ID)
                   16 million service instances (vs 4K VLANs)

Benefits:
  - 16M service instances via I-SID (vs 4K with Q-in-Q)
  - Customer MAC isolation (provider only learns B-MACs)
  - Reduced MAC table size in provider core
  - Clear separation of customer and provider address spaces

! NX-OS PBB configuration (conceptual)
! Note: PBB is primarily used in SP core / PBB-EVPN deployments

l2vpn evpn
 service instance 1
  encapsulation pbb
   backbone-vlan 100
   isid 10001
```

## VPLS (Virtual Private LAN Service)

```
VPLS provides multipoint L2 VPN over MPLS core.
Each customer gets a virtual Ethernet switch spanning all PE routers.

         CE-A                                    CE-B
          │                                       │
     ┌────┤ PE-A ─── pseudowire ─── PE-B ────┐   │
     │    │                                   │───┘
     │    └──── pseudowire ─────┐             │
     │                          │             │
     │                     PE-C ─── pseudowire┘
     │                      │
     └───── pseudowire ─────┘
                            │
                          CE-C

VPLS creates full mesh of pseudowires between all PEs in a VPN.
Each PE performs MAC learning, flooding, and forwarding locally.

Split-horizon: frames received on a PW are never forwarded to another PW
               (prevents loops in full-mesh PW topology)
```

### VPLS — LDP Signaling (Kompella / RFC 4762)

```
! IOS-XE VPLS with LDP signaling (Martini-style)
l2vpn vfi context VPLS-CUST-A
 vpn id 100
 member 10.0.0.2 encapsulation mpls
 member 10.0.0.3 encapsulation mpls

bridge-domain 100
 member GigabitEthernet0/0/0 service-instance 100
 member vfi VPLS-CUST-A

interface GigabitEthernet0/0/0
 service instance 100 ethernet
  encapsulation dot1q 100
```

### VPLS — BGP Signaling (Kompella / RFC 4761)

```
! IOS-XE VPLS with BGP signaling
router bgp 65001
 address-family l2vpn vpls
  neighbor 10.0.0.2 activate
  neighbor 10.0.0.2 send-community extended

l2vpn vfi context VPLS-CUST-A
 vpn id 100
 autodiscovery bgp signaling ldp
  ve-id 1
  ve-range 10
  route-target export 65001:100
  route-target import 65001:100
```

### H-VPLS (Hierarchical VPLS)

```
H-VPLS reduces the full-mesh PW requirement by introducing hub-spoke:

Full mesh VPLS (N PEs): N*(N-1)/2 pseudowires
  10 PEs = 45 PWs, 50 PEs = 1225 PWs (scaling problem)

H-VPLS solution: hub PE (aggregation) + spoke PEs (access)
  Spoke PEs connect to hub PE only (spoke PW)
  Hub PEs maintain full mesh between themselves

           ┌── Hub-PE1 ═══ Hub-PE2 ──┐    (full mesh between hubs)
           │                          │
     Spoke-PE1                  Spoke-PE3  (spoke PW to hub only)
     Spoke-PE2                  Spoke-PE4

  10 hub PEs + 40 spoke PEs:
    Hub-Hub PWs:   10*9/2 = 45
    Spoke-Hub PWs: 40
    Total:         85  (vs 1225 for full mesh)

! IOS-XE H-VPLS spoke PE
l2vpn vfi context HVPLS-SPOKE
 vpn id 100
 member 10.0.0.1 encapsulation mpls  ! to hub PE only
```

## Xconnect / Pseudowire

```
Pseudowire (PW) emulates a point-to-point circuit over MPLS.
Used for E-Line services and as building blocks for VPLS.

! IOS-XE xconnect (legacy syntax)
interface GigabitEthernet0/0/0
 xconnect 10.0.0.2 100 encapsulation mpls
 ! 10.0.0.2 = remote PE loopback
 ! 100 = VC ID (must match on both ends)

! IOS-XE pseudowire (modern syntax)
interface pseudowire100
 encapsulation mpls
 neighbor 10.0.0.2 100
 signaling protocol ldp

l2vpn xconnect context ELINE-CUST-A
 member GigabitEthernet0/0/0 service-instance 100
 member pseudowire100

! Pseudowire types (RFC 4446):
Type    Encapsulation            IANA PW Type
────────────────────────────────────────────────
0x0005  Ethernet (port mode)     Raw Ethernet
0x0004  Ethernet VLAN            Tagged Ethernet
0x0001  Frame Relay DLCI         FR PW
0x0006  HDLC                     HDLC PW
0x0011  Structure-agnostic TDM   CESoPSN

! Verification
show l2vpn atom vc
show l2vpn atom vc detail
show mpls l2transport vc 100
show xconnect all
```

## MPLS-Based Carrier Ethernet

```
MPLS provides the transport for carrier ethernet services:

Service          MPLS Implementation        Signaling
──────────────────────────────────────────────────────────────
E-Line           Pseudowire (VPWS)          LDP (T-LDP)
E-LAN            VPLS or EVPN              LDP/BGP or BGP
E-Tree           VPLS with leaf indication  BGP-EVPN
E-Access         Pseudowire (inter-AS)      LDP or BGP

Label Stack:
  [Transport Label][VC Label][Customer Frame]
  └─── to PE ─────┘└─ to PW ─┘

  Transport label: pushed by ingress PE, swapped by P routers
  VC label: identifies the pseudowire/VPN instance at egress PE
```

## MEF 3.0 LSO (Lifecycle Service Orchestration)

```
MEF 3.0 defines APIs for service lifecycle management:

LSO Reference Architecture:
  ┌──────────────┐
  │   Customer   │
  │   (BSS/OSS)  │
  └──────┬───────┘
    Sonata API │  (inter-carrier service ordering)
  ┌──────┴───────┐
  │   Partner    │
  │   (SP BSS)   │
  └──────┬───────┘
    Cantata API │ (service orchestration)
  ┌──────┴───────┐
  │  Orchestrator │
  └──────┬───────┘
    Presto API │  (infrastructure control)
  ┌──────┴───────┐
  │  Controller   │
  │  (SDN/NMS)    │
  └──────────────┘

LSO APIs (RESTful, OpenAPI):
  Sonata:   Inter-carrier service ordering (B2B)
  Cantata:  Business application to orchestrator
  Legato:   Service orchestration to OSS/BSS
  Interlude: Between orchestration layers
  Presto:   Orchestrator to infrastructure/SDN controller
  Adagio:   Monitoring and analytics
```

## Verification Commands

```
! --- EVC / Service Instance ---
show ethernet service instance summary
show ethernet service instance id 100 interface GigabitEthernet0/0/0 detail
show bridge-domain 100

! --- CFM ---
show ethernet cfm maintenance-points local
show ethernet cfm maintenance-points remote
show ethernet cfm maintenance-points remote detail
show ethernet cfm errors
show ethernet cfm statistics

! --- Y.1731 Performance ---
show ethernet cfm delay-measurement
show ethernet cfm slm
show ethernet cfm loss-measurement

! --- REP ---
show rep topology
show rep topology segment 1 detail

! --- G.8032 ---
show ethernet ring g8032 status
show ethernet ring g8032 brief
show ethernet ring g8032 statistics

! --- VPLS ---
show l2vpn vfi
show l2vpn vfi name VPLS-CUST-A
show bridge-domain 100
show l2vpn atom vc

! --- Pseudowire ---
show l2vpn atom vc
show l2vpn atom vc detail
show mpls l2transport vc
show xconnect all

! --- MPLS transport ---
show mpls forwarding-table
show mpls ldp neighbor
show mpls ldp bindings
```

## Tips

- Use 802.1ag CFM with continuity checks (CCM) on every EVC. Without CFM, you have no visibility into end-to-end service health. A 1-second CCM interval detects failures within 3.5 seconds (3.5x multiplier). Use 100ms CCM for sub-second detection where needed.
- Deploy Y.1731 performance monitoring (DM/SLM) proactively, not just during troubleshooting. Continuous delay and loss measurement establishes baselines that make SLA violations immediately visible before customers notice.
- G.8032 provides sub-50ms ring protection and is vendor-neutral (unlike REP). For multi-vendor environments or standards compliance requirements, always prefer G.8032 over proprietary alternatives.
- VPLS full-mesh pseudowire scaling follows O(N^2). For deployments beyond 10-15 PEs, use H-VPLS to reduce the PW count, or migrate to BGP-EVPN which provides better scaling through route reflection and all-active multi-homing.
- PBB (802.1ah) solves two problems simultaneously: the 4K VLAN limit (I-SID provides 16M instances) and MAC table explosion in the provider core (only backbone MACs are learned). Consider PBB-EVPN for large-scale L2 VPN deployments.
- Always match VC ID and encapsulation type on both ends of a pseudowire. A VC ID mismatch silently fails — the pseudowire shows "down" with no obvious error. Check `show l2vpn atom vc detail` for status codes.
- For E-Tree services, ensure leaf-to-leaf traffic is properly blocked. In VPLS-based implementations, this requires split-horizon extensions or dedicated root/leaf PW designations. BGP-EVPN E-Tree uses EVPN route type 1 with leaf indication.
- MEF bandwidth profiles use dual-rate token bucket (trTCM). Set CIR to the guaranteed rate, EIR to the maximum burst rate, and enable the coupling flag (CF=1) so unused CIR tokens can supplement the EIR bucket, maximizing customer throughput during low utilization.
- CFM levels must be planned carefully across the operator/provider/customer hierarchy. A common mistake is using the same level for operator and provider OAM, which causes one to be invisible to the other. Follow the convention: 0-2 operator, 3-4 provider, 5-7 customer.
- When deploying xconnect/pseudowire over MPLS, verify that the MPLS transport LSP exists end-to-end between PE loopbacks. A working LDP adjacency does not guarantee an LSP — check `show mpls forwarding-table` for the remote PE prefix.

## See Also

- mpls, mpls-vpn, vlan, stp, vxlan, ethernet, lacp, cos-qos, qos-advanced, segment-routing

## References

- [MEF 6.3 — Subscriber Ethernet Service Definitions](https://www.mef.net/resources/mef-6-3-subscriber-ethernet-service-definitions/)
- [MEF 10.4 — Subscriber Ethernet Service Attributes](https://www.mef.net/resources/mef-10-4-subscriber-ethernet-service-attributes/)
- [MEF 26.2 — ENNI and Operator Ethernet Service Attributes](https://www.mef.net/resources/mef-26-2/)
- [IEEE 802.1ag — Connectivity Fault Management](https://standards.ieee.org/standard/802_1ag-2007.html)
- [ITU-T Y.1731 — OAM Functions and Mechanisms for Ethernet-Based Networks](https://www.itu.int/rec/T-REC-Y.1731)
- [ITU-T G.8032 — Ethernet Ring Protection Switching](https://www.itu.int/rec/T-REC-G.8032)
- [IEEE 802.1ah — Provider Backbone Bridges](https://standards.ieee.org/standard/802_1ah-2008.html)
- [RFC 4762 — VPLS Using LDP Signaling](https://datatracker.ietf.org/doc/html/rfc4762)
- [RFC 4761 — VPLS Using BGP for Auto-Discovery and Signaling](https://datatracker.ietf.org/doc/html/rfc4761)
- [RFC 4446 — IANA Allocations for Pseudowire Edge to Edge Emulation](https://datatracker.ietf.org/doc/html/rfc4446)
- [RFC 4448 — Encapsulation Methods for Transport of Ethernet over MPLS Networks](https://datatracker.ietf.org/doc/html/rfc4448)
- [Cisco Carrier Ethernet Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/cether/configuration/xe-16/ce-xe-16-book.html)
