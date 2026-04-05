# Carrier Ethernet — MEF Architecture, OAM Theory, and Scaling Analysis

> *Carrier Ethernet transforms the ubiquitous Ethernet technology from a best-effort LAN protocol into a carrier-grade wide-area service with guaranteed performance, fault detection, and service management. The Metro Ethernet Forum (MEF) defines the service abstractions (E-Line, E-LAN, E-Tree), the performance attributes (delay, loss, jitter, availability), and the operational frameworks (OAM, LSO) that enable service providers to deliver Ethernet services with the same reliability expectations as legacy TDM circuits. This document covers the MEF reference architecture, the CFM domain hierarchy and its fault isolation model, Y.1731 performance monitoring theory, ring protection switching algorithms, PBB MAC-in-MAC encapsulation mechanics, VPLS signaling comparison, pseudowire emulation theory, and the scaling characteristics that determine which technology is appropriate at each network scale.*

---

## 1. MEF Reference Architecture

### 1.1 Service Model

The MEF service model separates the customer-facing service definition from the underlying transport technology. This abstraction is what makes "Carrier Ethernet" technology-agnostic — the same E-Line service can be delivered over MPLS pseudowires, PBB, VPLS, or even native Ethernet with OAM.

**Core abstractions:**

**EVC (Ethernet Virtual Connection):** The fundamental service primitive. An EVC defines a logical association between two or more UNIs that constrains the delivery of frames. An EVC guarantees that frames entering one UNI can only exit through UNIs that are part of the same EVC — isolation between EVCs is absolute.

**UNI (User Network Interface):** The physical and logical demarcation point where the customer network connects to the provider network. The UNI defines what the customer "sees" — physical medium, speed, bundling/multiplexing, VLAN handling, and bandwidth profile.

**ENNI (External Network Network Interface):** The interconnection point between two operator networks. The ENNI is where one provider's service domain ends and another's begins. ENNI uses operator VLAN tags (S-VLAN) to multiplex multiple customer services across the inter-provider link.

### 1.2 Service Type Formal Definitions

**E-Line (Point-to-Point EVC):**

An EVC that associates exactly two UNI endpoints. Every frame entering UNI-A exits at UNI-B and vice versa. The E-Line is the Carrier Ethernet equivalent of a leased line or point-to-point circuit.

```
Formal properties:
  - Exactly 2 UNIs per EVC
  - Unicast frames are delivered only to the peer UNI
  - Multicast/broadcast frames are delivered only to the peer UNI
  - No MAC learning required (single destination)
  - Bandwidth profile applied per UNI per EVC

Implementation options:
  - MPLS pseudowire (VPWS) — most common
  - QinQ tunneling (S-VLAN transport) — metro networks
  - PBB point-to-point — large-scale with MAC isolation
```

**E-LAN (Multipoint-to-Multipoint EVC):**

An EVC that associates two or more UNI endpoints with multipoint connectivity. Any UNI can communicate with any other UNI in the same EVC. The E-LAN is the Carrier Ethernet equivalent of a transparent LAN service.

```
Formal properties:
  - 2 or more UNIs per EVC
  - Unicast: delivered to the specific UNI based on MAC learning
  - Multicast/broadcast: delivered to all other UNIs in the EVC
  - Requires MAC learning (or flooding) at each PE
  - Provider must prevent loops (split-horizon for VPLS, or STP)

Implementation options:
  - VPLS (RFC 4762/4761) — mature, widely deployed
  - BGP-EVPN (RFC 7432) — modern, better scaling/multi-homing
  - PBB-EVPN — extreme scale with MAC isolation
```

**E-Tree (Rooted Multipoint EVC):**

An EVC with root and leaf UNI roles. Root UNIs can communicate with all other UNIs (root and leaf). Leaf UNIs can communicate only with root UNIs, not with other leaf UNIs.

```
Formal properties:
  - 2 or more UNIs, each designated Root or Leaf
  - Root → Root: allowed
  - Root → Leaf: allowed
  - Leaf → Root: allowed
  - Leaf → Leaf: BLOCKED

Use cases:
  - Internet access (root = PE/gateway, leaf = customer sites)
  - Video distribution (root = head-end, leaf = viewing sites)
  - Multitenancy (leaf isolation without separate EVCs)

Implementation complexity:
  - VPLS: requires extensions for leaf indication
  - EVPN: native E-Tree support via route type 1 leaf flag
  - PBB: I-SID per root-leaf pair or leaf-indication bit
```

### 1.3 MEF Service Attributes

Each EVC carries a set of service attributes that define the SLA:

| Attribute | Description | Specified Per |
|:---|:---|:---|
| EVC Type | P2P, MP2MP, Rooted MP | EVC |
| UNI Count | Number of UNI endpoints | EVC |
| CE-VLAN ID Preservation | Whether customer VLANs are preserved | EVC |
| CE-VLAN CoS Preservation | Whether customer CoS bits are preserved | EVC |
| Unicast Frame Delivery | Unconditional, Conditional, or Discard | EVC + CoS |
| Multicast Frame Delivery | Unconditional, Conditional, or Discard | EVC + CoS |
| Broadcast Frame Delivery | Unconditional, Conditional, or Discard | EVC + CoS |
| Frame Delay Performance | One-way delay objective | CoS |
| Frame Loss Performance | Loss ratio objective | CoS |
| Bandwidth Profile | CIR/CBS/EIR/EBS per CoS | UNI per EVC per CoS |

---

## 2. CFM Domain Hierarchy Theory

### 2.1 Maintenance Domain (MD) Architecture

CFM (IEEE 802.1ag) uses a hierarchical nesting of maintenance domains to provide fault isolation at different administrative levels. Each domain operates at a specific maintenance level (0-7), and higher levels encapsulate lower levels.

```
The nesting property:
  An MD at level K spans OVER all MDs at levels < K.
  An MD at level K cannot see INSIDE MDs at levels > K.

Physical topology:

  CE-A ─── Access-SW1 ─── Agg-SW1 ═══ Core ═══ Agg-SW2 ─── Access-SW2 ─── CE-B
  │         │              │                      │             │            │
  └─ L7 MEP─┘              │                      │             └─ L7 MEP ──┘
            └─── L4 MEP ───┘                      └─── L4 MEP ──┘
                           └──── L1 MEP ──────────┘

Level 7 MD (Customer):
  Spans end-to-end (CE-A to CE-B)
  MEPs at customer edge devices
  Detects: entire service path failure

Level 4 MD (Provider):
  Spans provider network edge-to-edge
  MEPs at provider PE routers
  Detects: provider network internal failures
  Cannot see inside customer domain (level 7)

Level 1 MD (Operator):
  Spans individual operator segment
  MEPs at segment endpoints
  Detects: link or node failure within segment
  Cannot see inside provider domain (level 4)
```

### 2.2 MEP and MIP Behavior

**MEP (Maintenance End Point):**

A MEP is the active entity that generates, receives, and processes OAM frames. MEPs define the boundary of a maintenance domain.

```
MEP responsibilities:
  1. Generate CCM (Continuity Check Messages) at configured interval
  2. Receive CCMs from peer MEPs and detect:
     - Missing MEP (no CCM received within 3.5x interval)
     - Unexpected MEP (CCM from unknown MEP ID)
     - Cross-connect (CCM from wrong MA or wrong level)
     - Error CCM (invalid content)
  3. Respond to LBM (Loopback Messages) — L2 ping
  4. Generate/process LTM/LTR (Linktrace) — L2 traceroute
  5. Generate AIS (Alarm Indication Signal) toward client level
  6. Y.1731: generate/process DM, SLM, LM frames

MEP direction:
  - Down MEP: faces toward the wire (sends OAM out the port)
  - Up MEP: faces toward the switch fabric (sends OAM into the bridge)

  Down MEPs are used at external-facing interfaces (UNI, ENNI)
  Up MEPs are used internally (service instance facing the bridge domain)
```

**MIP (Maintenance Intermediate Point):**

A MIP is a passive entity that responds to OAM queries but does not generate them. MIPs exist at transit nodes within a maintenance domain.

```
MIP responsibilities:
  1. Respond to LBM (L2 ping) targeting this MIP's MAC
  2. Forward LTM and generate LTR (L2 traceroute hop response)
  3. Catalog received CCMs (for debugging, not for fault detection)
  4. Do NOT generate CCMs
  5. Do NOT generate AIS or other alarms

MIP creation:
  MIPs are typically auto-created at every bridge port that lies between
  two MEPs of the same level. Some implementations require explicit
  configuration.
```

### 2.3 CCM-Based Fault Detection

CCM (Continuity Check Messages) provide the heartbeat mechanism for CFM:

```
CCM Intervals (IEEE 802.1ag):
  3.3 ms    (300 pps)   — sub-second detection, high overhead
  10 ms     (100 pps)   — fast detection
  100 ms    (10 pps)    — common for SP
  1 second  (1 pps)     — standard (most common)
  10 seconds            — low overhead
  1 minute              — minimal overhead
  10 minutes            — minimal overhead

Detection time = 3.5 * CCM interval
  1s CCM → 3.5s detection
  100ms CCM → 350ms detection
  3.3ms CCM → 11.55ms detection

CCM frame contents:
  - MD Level (3 bits): which domain this CCM belongs to
  - Version (5 bits): protocol version
  - OpCode: 1 (CCM)
  - Flags: RDI (Remote Defect Indication), interval
  - First TLV Offset
  - Sequence Number: incrementing counter
  - MA ID: identifies the Maintenance Association
  - MEP ID: identifies the sending MEP (1-8191)
  - Sender ID TLV (optional): chassis/port identification

Fault detection scenarios:
  1. LOC (Loss of Continuity): no CCM received from known remote MEP
     within 3.5x interval → link or node failure
  2. Unexpected MEP: CCM received from MEP ID not in crosscheck list
     → misconfiguration or unauthorized device
  3. Mismerge: CCM received with different MA ID at same level
     → two services merged (VLAN misconfiguration)
  4. Unexpected Level: CCM at unexpected level
     → domain hierarchy misconfiguration
  5. RDI: remote MEP declares a defect
     → far-end failure condition
```

---

## 3. Y.1731 Performance Monitoring

### 3.1 Frame Delay Measurement (DM)

Y.1731 defines two delay measurement modes:

**Two-Way Delay Measurement (1DM/DMM/DMR):**

```
MEP-A                                            MEP-B
  │                                                │
  │──── DMM (TxTimestampf = T1) ──────────────────►│
  │                                                │ T2 = RxTimestampf
  │                                                │ T3 = TxTimestampb
  │◄──── DMR (RxTimestampf=T2, TxTimestampb=T3) ──│
  │ T4 = RxTimestampb                              │

Two-way frame delay:
  2WFD = (T4 - T1) - (T3 - T2)
       = round-trip time - remote processing time

Two-way frame delay variation (jitter):
  2WFDV = |2WFD(n) - 2WFD(n-1)|

Note: Two-way DM does not require clock synchronization between
MEP-A and MEP-B because the remote processing time (T3-T2) is
subtracted out.
```

**One-Way Delay Measurement (1DM):**

```
MEP-A                                            MEP-B
  │                                                │
  │──── 1DM (TxTimestampf = T1) ──────────────────►│
  │                                                │ T2 = RxTimestampf
  │                                                │
  │ One-way delay = T2 - T1                        │

Requirement: MEP-A and MEP-B clocks must be synchronized
  - PTP (IEEE 1588v2) provides sub-microsecond accuracy
  - GPS can provide nanosecond accuracy
  - NTP accuracy (~1ms) is insufficient for meaningful 1DM
```

### 3.2 Synthetic Loss Measurement (SLM)

SLM measures frame loss by injecting synthetic test frames and counting arrivals:

```
MEP-A                                            MEP-B
  │                                                │
  │──── SLM (TxFCf = 100) ───────────────────────►│
  │                                                │ RxFCf = 100
  │                                                │ TxFCb = count
  │◄──── SLR (TxFCf=100, RxFCf=98, TxFCb=n) ─────│
  │                                                │

Near-end frame loss (A→B):
  Loss_near = TxFCf(n) - TxFCf(n-1) - (RxFCf(n) - RxFCf(n-1))

Far-end frame loss (B→A):
  Loss_far = TxFCb(n) - TxFCb(n-1) - (RxFCb(n) - RxFCb(n-1))

Frame Loss Ratio (FLR):
  FLR = Lost_frames / Sent_frames * 100%

SLM vs LM:
  SLM: uses synthetic test frames, measures loss of OAM traffic
  LM:  counts actual data frames, measures loss of customer traffic
  SLM is easier to deploy (no data plane counters needed)
  LM is more accurate (measures actual service quality)
```

### 3.3 Loss Measurement (LM)

LM provides exact frame loss by counting data frames at ingress and egress of each MEP:

```
Frame counters maintained per MEP:
  TxFCl: total data frames transmitted toward peer MEP
  RxFCl: total data frames received from peer MEP

MEP-A                                            MEP-B
  │                                                │
  │── LMM (TxFCf = A's TxFCl) ──────────────────►│
  │                                                │ Records:
  │                                                │   RxFCf = B's RxFCl
  │                                                │   TxFCb = B's TxFCl
  │◄── LMR (TxFCf, RxFCf, TxFCb) ────────────────│
  │ Records:                                       │
  │   RxFCb = A's RxFCl                            │

Between two consecutive LM exchanges (at times n-1 and n):

Near-end loss (A → B):
  Sent by A:     TxFCf(n) - TxFCf(n-1)
  Received by B: RxFCf(n) - RxFCf(n-1)
  Lost:          Sent - Received

Far-end loss (B → A):
  Sent by B:     TxFCb(n) - TxFCb(n-1)
  Received by A: RxFCb(n) - RxFCb(n-1)
  Lost:          Sent - Received
```

---

## 4. ERP Ring Protection Switching

### 4.1 G.8032 State Machine

G.8032 Ethernet Ring Protection (ERP) uses a finite state machine at each ring node to coordinate protection switching. The key states are:

```
State Machine per ring node:

  ┌─────────┐   link fail    ┌────────────┐
  │  IDLE   │ ──────────────►│ PROTECTION │
  │(normal) │                │  (active)  │
  └────┬────┘   ◄────────── └─────┬──────┘
       │        link recover      │
       │        + WTR expires     │
       │                          │
       │   forced switch    ┌─────┴──────┐
       └───────────────────►│   FORCED   │
                            │  SWITCH    │
                            └────────────┘

IDLE State:
  - RPL is blocked (at RPL Owner)
  - All other ring ports are forwarding
  - CCMs monitor ring link health
  - R-APS (NR) messages circulate (No Request)

PROTECTION State:
  - Triggered by R-APS(SF) — Signal Fail
  - RPL Owner unblocks the RPL
  - Failed link ports are blocked
  - All nodes flush FDB (MAC table)
  - Traffic reroutes around the ring via the now-unblocked RPL

WTR (Wait-to-Restore):
  - After link recovery, WTR timer starts (default 5 minutes)
  - Prevents flapping if the link is unstable
  - After WTR expires, ring returns to IDLE state
  - RPL is re-blocked, recovered link is unblocked
```

### 4.2 R-APS Protocol

R-APS (Ring Automatic Protection Switching) messages coordinate protection state across the ring:

```
R-APS message format:
  - Request/State: NR (No Request), SF (Signal Fail), FS (Forced Switch),
                   MS (Manual Switch), WTR (Wait to Restore)
  - RPL Blocked:   indicates if this node is blocking the RPL
  - DNF (Do Not Flush): suppress FDB flush during certain events
  - Node ID:       identifier of the sending node
  - BPR:           blocked port reference

R-APS message flow:
  R-APS messages are sent on a dedicated ring APS VLAN (R-APS channel)
  Each node processes R-APS and may change its ring port state

Timing:
  R-APS interval: 5 seconds (default)
  Hold-off timer: 0-10 seconds (delay before reporting SF)
  WTR timer: 1-12 minutes (default 5 minutes)
  Guard timer: 500ms (ignores R-APS during FDB flush)
```

### 4.3 Protection Switching Time Analysis

```
Total switching time = Detection + Signaling + Action

Detection:
  CFM CCM-based: 3.5 * CCM_interval
    1s CCM → 3.5s (too slow for sub-50ms)
    3.3ms CCM → 11.55ms (meets target)
  Hardware LOF/LOS: < 10ms
  Recommendation: use hardware detection + CCM for confirmation

Signaling:
  R-APS propagation: < 1ms per node (wire speed)
  Full ring propagation: N_nodes * 1ms (typically < 10ms)

Action:
  Port state change: < 1ms (hardware)
  FDB flush: 1-5ms (depends on table size)
  MAC relearning: 1-50ms (depends on traffic rate)

Total (hardware detection):
  10ms + 10ms + 5ms = 25ms (well within 50ms target)

Total (CCM 3.3ms detection):
  11.55ms + 10ms + 5ms = 26.55ms (within 50ms)

Total (CCM 1s detection):
  3500ms + 10ms + 5ms = 3515ms (fails 50ms target)

Conclusion: For sub-50ms switching, hardware detection or 3.3ms CCM
is required. The commonly-used 1s CCM is too slow.
```

### 4.4 Multiple Ring Topologies

G.8032v2 supports interconnected rings:

```
Sub-ring (open ring connected to major ring):

  Major Ring:                Sub-Ring:
  ┌─── A ─── B ───┐         ┌─── D ─── E ───┐
  │                │─────────│                │
  └─── C ──────────┘         └────────────────┘
                   ^                           ^
            interconnection              virtual channel
            node (B)                     (no physical ring closure)

Sub-ring without virtual channel:
  - R-APS messages from sub-ring do NOT traverse major ring
  - Sub-ring protects independently
  - Interconnection node participates in both ring instances

Sub-ring with virtual channel:
  - R-APS messages tunnel through major ring
  - Provides end-to-end coordination
  - More complex but better convergence for multi-ring topologies
```

---

## 5. PBB MAC-in-MAC Encapsulation

### 5.1 Frame Format

PBB (IEEE 802.1ah) encapsulates the entire customer Ethernet frame inside a new backbone MAC header:

```
Original customer frame:
┌──────┬──────┬────────┬─────────────┬─────┐
│ C-DA │ C-SA │ C-VLAN │   Payload   │ FCS │
│ 6B   │ 6B   │ 4B     │             │ 4B  │
└──────┴──────┴────────┴─────────────┴─────┘

PBB encapsulated frame:
┌──────┬──────┬────────┬──────┬──────┬──────┬────────┬─────────────┬─────┐
│ B-DA │ B-SA │ B-VLAN │ I-TAG│ C-DA │ C-SA │ C-VLAN │   Payload   │ FCS │
│ 6B   │ 6B   │ 4B     │ 6B  │ 6B   │ 6B   │ 4B     │             │ 4B  │
└──────┴──────┴────────┴──────┴──────┴──────┴────────┴─────────────┴─────┘
│←── backbone header ──→│←── I-SID──→│←── original customer frame ─────→│

I-TAG format:
┌──────────┬───┬───┬────┬────────────────────────┐
│ EtherType│ P │ D │ U  │      I-SID             │
│  0x88E7  │3b │1b │2b  │      24 bits           │
└──────────┴───┴───┴────┴────────────────────────┘
  P = Priority (3 bits, CoS for backbone)
  D = Drop Eligible (1 bit)
  U = Use Customer Address (2 bits)
  I-SID = Instance Service Identifier (24 bits = 16,777,216 instances)
```

### 5.2 MAC Address Handling

```
Backbone network MAC learning:

Without PBB (VPLS/QinQ):
  Provider core switches learn CUSTOMER MAC addresses
  N customers * M MACs_per_customer = N*M entries in provider FDB
  Example: 10,000 customers * 100 MACs = 1,000,000 FDB entries

With PBB:
  Provider core switches learn only BACKBONE MAC addresses (B-MAC)
  One B-MAC per provider edge device
  Example: 100 PBB edge devices = 100 FDB entries in the core
  1,000,000 customer MACs are encapsulated and invisible to core

This is the MAC scalability advantage of PBB:
  Core FDB size = O(PE_devices) instead of O(customers * MACs)

B-DA determination:
  PBB edge node maintains: I-SID → (C-MAC → B-MAC) mapping
  When customer frame arrives:
    1. Look up I-SID from customer VLAN/port mapping
    2. Look up B-DA from C-DA in the I-SID's FDB
    3. If unknown: flood to all PBB edge nodes in the I-SID (backbone multicast)
    4. If known: unicast to specific B-DA
```

### 5.3 PBB-EVPN Integration

```
PBB-EVPN combines PBB encapsulation with EVPN control plane:

  EVPN provides:
    - BGP-based MAC advertisement (no data-plane flooding for known MACs)
    - Multi-homing with all-active forwarding
    - Fast convergence via mass MAC withdrawal

  PBB provides:
    - MAC scalability (only B-MACs in BGP, not C-MACs)
    - 16M service instances via I-SID

  PBB-EVPN operation:
    1. Customer MAC learned locally at PBB PE
    2. PE advertises (B-MAC, I-SID) via EVPN Route Type 2
       Note: C-MAC is NOT advertised in BGP (stays local)
    3. Remote PE receives route, installs B-MAC in backbone FDB
    4. Data forwarding: PBB encapsulation with B-DA from EVPN

  Scaling comparison:
    VPLS:     O(N^2) PWs, O(C*M) MAC routes
    EVPN:     O(1) per PE (route reflection), O(C*M) MAC routes
    PBB-EVPN: O(1) per PE (route reflection), O(PE) MAC routes
              (C-MACs stay local, only B-MACs advertised)
```

---

## 6. VPLS Signaling Comparison

### 6.1 LDP-Based VPLS (RFC 4762, "Lasserre-Kompella")

```
LDP signaling uses Targeted LDP (T-LDP) sessions between PEs:

Setup sequence:
  1. PE discovers remote PEs (manual configuration of neighbors)
  2. T-LDP session established between each PE pair
  3. Each PE sends FEC 128 Label Mapping for the VPN (VC ID)
  4. Both PEs install the pseudowire (VC label + transport LSP)

Label assignment:
  - VC label: allocated by each PE, advertised via T-LDP FEC 128
  - Transport label: from LDP or RSVP-TE (underlay LSP)

  PE-A allocates VC label 100, sends to PE-B via T-LDP
  PE-B allocates VC label 200, sends to PE-A via T-LDP

  PE-A → PE-B: [Transport Label][VC Label 200][Customer Frame]
  PE-B → PE-A: [Transport Label][VC Label 100][Customer Frame]

Advantages:
  - Simple configuration (just VC ID + peer list)
  - Mature, widely supported
  - No BGP dependency

Disadvantages:
  - Manual full-mesh configuration: N*(N-1)/2 peer statements
  - No auto-discovery: adding a PE requires updating all existing PEs
  - No multi-homing support (single-homed only)
  - MAC withdrawal requires per-PE notification (slow convergence)
```

### 6.2 BGP-Based VPLS (RFC 4761, "Kompella")

```
BGP signaling uses BGP auto-discovery and label allocation:

Setup sequence:
  1. PE advertises VPLS membership via BGP NLRI:
     - Route Distinguisher (RD)
     - VE ID (Virtual Edge ID): unique per PE in the VPLS
     - VE Block: range of VE IDs this PE will accept
     - Label Block Base: starting label for this VE block
  2. Remote PEs receive BGP update, compute PW labels from VE block
  3. PWs are automatically established (no manual peer configuration)

Auto-discovery:
  BGP-VPLS uses the L2VPN AFI/SAFI (25/65) for NLRI distribution
  Route targets (RT) control which PEs join which VPLS instances:
    Export RT: advertise membership in this VPLS
    Import RT: accept membership from PEs with this RT

Label computation:
  PE-A announces: VE-ID=1, Label-Base=100, VE-Block-Size=10
  PE-B computes label for PW to PE-A: Label-Base + VE-ID(A) - 1 = 100

  This eliminates per-PW label signaling — labels are derived
  from the BGP advertisement deterministically.

Advantages:
  - Auto-discovery (add PE → all PEs learn automatically via BGP)
  - Route reflection reduces BGP sessions to O(N) instead of O(N^2)
  - Multi-homing support (BGP can advertise backup paths)
  - Scales much better than LDP-VPLS for large deployments

Disadvantages:
  - Requires BGP infrastructure (route reflectors)
  - More complex initial setup
  - VE Block sizing requires planning
```

### 6.3 Comparison Summary

| Aspect | LDP-VPLS (RFC 4762) | BGP-VPLS (RFC 4761) |
|:---|:---|:---|
| Discovery | Manual (configure peers) | Automatic (BGP RT) |
| Signaling sessions | O(N^2) T-LDP sessions | O(N) BGP sessions (with RR) |
| Adding a PE | Update all existing PEs | Configure new PE only |
| Label allocation | Dynamic per-PW | Computed from label block |
| Multi-homing | Not supported | Supported |
| H-VPLS | Supported (spoke/hub PW) | Supported |
| Operational maturity | Very mature | Mature |
| Migration to EVPN | Requires rearchitect | Incremental (same BGP infra) |

---

## 7. Pseudowire Emulation Theory

### 7.1 Pseudowire Architecture (RFC 3985)

A pseudowire emulates a point-to-point connection over a packet-switched network. The architecture defines three layers:

```
┌──────────────────────────────────────────────────────────────┐
│                    Payload (customer frame)                   │
├──────────────────────────────────────────────────────────────┤
│         PW Demux Layer (VC label or PW label)                │
├──────────────────────────────────────────────────────────────┤
│         PSN Tunnel (transport LSP: LDP, RSVP-TE, or SR)     │
├──────────────────────────────────────────────────────────────┤
│         PSN (MPLS, IP/GRE, L2TPv3)                           │
└──────────────────────────────────────────────────────────────┘

Payload types:
  - Ethernet (raw or tagged): PW type 0x0005 / 0x0004
  - Frame Relay: PW type 0x0001
  - ATM (cell/AAL5): PW types 0x0002, 0x0003, 0x000A
  - TDM (CESoPSN, SAToP): PW types 0x0011, 0x0012
  - PPP: PW type 0x0007

The PW preserves the native service semantics:
  - Ethernet PW: preserves MAC learning, flooding, VLAN tags
  - TDM PW: preserves bit timing, clock recovery
  - FR PW: preserves DLCI mapping, BECN/FECN
```

### 7.2 Control Word

The optional control word (4 bytes) is inserted between the PW label and the payload:

```
Control Word format:
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
├─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┤
│0 0 0 0│ Flags │FRG│  Length  │      Sequence Number              │
├─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┤

Purposes:
  1. Padding: for payloads shorter than 64 bytes (Length field indicates real size)
  2. Sequencing: optional packet ordering (Sequence Number)
  3. Fragmentation: for payloads exceeding PSN MTU (FRG bits)
  4. ECMP disambiguation: first nibble = 0000, not confused with IPv4/IPv6

Why control word matters for Ethernet PW:
  Without CW, an Ethernet payload starting with 0x4 or 0x6 in the
  first nibble of the destination MAC could be misidentified as
  IPv4/IPv6 by ECMP hash algorithms in the MPLS core. The CW's
  leading 0000 prevents this misidentification.

  When CW is negotiated: both PE endpoints must agree.
  IOS: pseudowire-class → control-word
  T-LDP FEC 128: C-bit indicates CW capability
```

### 7.3 PW Redundancy and Resiliency

```
PW redundancy provides failover between primary and backup PWs:

Active/Standby PW Redundancy:

  CE ─── PE-A ═══════ PW1 (active) ═══════ PE-B ─── CE
                ╚═════ PW2 (standby) ═════╝

  PE-A signals PW status via LDP Status TLV:
    Active PW: status = 0x00000000 (forwarding)
    Standby PW: status = 0x00000020 (standby)

  On PW1 failure:
    1. PE-A detects failure (PSN tunnel down, BFD, or PW OAM)
    2. PE-A switches to PW2 (changes status to forwarding)
    3. PE-A sends LDP Notification with new status
    4. Switchover time: 50ms-3s depending on detection mechanism

Multi-Segment PW (MS-PW):
  CE ─── PE-A ══ PW1 ══ S-PE ══ PW2 ══ PE-B ─── CE

  S-PE (Switching PE) stitches two PW segments together.
  Used for inter-AS pseudowire or extending PW reach.
  S-PE performs label swapping between PW segments.
```

---

## 8. Carrier Ethernet Scaling Analysis

### 8.1 Technology Scaling Comparison

```
Technology        Service    Control Plane     Data Plane         Max Scale
                  Instances  Overhead          Overhead
──────────────────────────────────────────────────────────────────────────────
QinQ (802.1ad)    4,094      None (L2 only)    4-byte S-VLAN tag  Small metro
                             STP convergence   12-byte overhead    (< 100 sites)

VPLS (LDP)        Unlimited  O(N^2) T-LDP      2 MPLS labels      Medium SP
                             Manual peers       8-byte overhead     (< 50 PEs)

VPLS (BGP)        Unlimited  O(N) with RR      2 MPLS labels      Large SP
                             Auto-discovery     8-byte overhead     (< 200 PEs)

PBB (802.1ah)     16M        None (L2 only)    18-byte I-TAG+BMAC Metro/regional
                             STP or G.8032     MAC isolation       (< 500 nodes)

BGP-EVPN          Unlimited  O(N) with RR      2 MPLS labels      Very large
                             MAC in BGP         8-byte overhead     (< 1000 PEs)

PBB-EVPN          16M        O(N) with RR      MPLS + I-TAG       Largest
                             B-MAC in BGP       26-byte overhead    (< 5000 PEs)
                             (not C-MAC!)       MAC isolation
```

### 8.2 MAC Table Scaling

```
MAC table pressure is the primary scaling concern for L2 VPN:

VPLS MAC table at a PE:
  Each PE learns C-MACs from ALL remote sites in the VPLS instance.
  Total MACs per PE = sum(MACs at each remote site)

  Example: 100-site VPLS, 200 MACs per site
  PE FDB = 100 * 200 = 20,000 entries per VPLS instance
  10 VPLS instances = 200,000 FDB entries

EVPN MAC table optimization:
  EVPN advertises MACs via BGP (no flooding for known unicast)
  PE still installs remote MACs in FDB, but:
  - Unknown unicast is suppressed (ARP proxy in EVPN)
  - MAC moves are signaled via BGP (faster convergence)
  - Multi-homing reduces duplicate MAC learning

PBB-EVPN MAC table at a PE:
  Core: only B-MACs (number of PBB edge nodes)
  Edge: C-MACs per I-SID (local + learned)

  Example: 100-site PBB-EVPN, 200 MACs per site
  Core switch FDB: ~100 B-MACs (one per PE)
  Edge PE FDB: 200 local + remote C-MACs per I-SID

  Core FDB reduction: 200,000 → 100 (2000x improvement)
```

### 8.3 Convergence Time Comparison

```
Technology        Failure Detection    Service Restoration    Total
──────────────────────────────────────────────────────────────────────
STP/RSTP          3-30 seconds         1-6 seconds           4-36 sec
REP               < 10ms (HW)          < 50ms                < 50ms
G.8032 ERP        < 10ms (HW)          < 50ms                < 50ms
VPLS (LDP)        BFD: 50-150ms        MAC flush + relearn   200ms-3s
VPLS (BGP)        BFD: 50-150ms        MAC withdrawal: <1s   200ms-1.5s
EVPN              BFD: 50-150ms        Mass withdrawal: <1s  200ms-1s
PW redundancy     BFD: 50-150ms        PW switchover: <50ms  100ms-200ms

Factors affecting convergence:
  1. Detection: BFD (fast) vs CCM (slower) vs holddown timer
  2. Signaling: R-APS (ms) vs LDP notification (ms) vs BGP withdrawal (s)
  3. FDB flush: immediate vs timer-based
  4. MAC relearning: flooding (slow) vs BGP advertisement (fast)
```

### 8.4 When to Use What

```
Scenario                              Recommended Technology
──────────────────────────────────────────────────────────────────
Metro ring, < 20 nodes, simple        G.8032 + QinQ
Metro, < 100 nodes, multi-service     MPLS + VPLS (LDP)
Regional SP, < 200 PEs                MPLS + VPLS (BGP) or EVPN
Large SP, > 200 PEs                   MPLS + BGP-EVPN
Extreme scale, > 1000 PEs             PBB-EVPN
Data center interconnect (DCI)        VXLAN + BGP-EVPN
Multi-vendor, standards required      G.8032 + 802.1ag + Y.1731

Migration path:
  QinQ → VPLS (LDP) → VPLS (BGP) → EVPN → PBB-EVPN
  Each step adds capability and complexity.
  EVPN is the current industry direction for new deployments.
```
