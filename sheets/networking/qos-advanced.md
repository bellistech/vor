# Advanced QoS (DiffServ, MQC, H-QoS & Traffic Engineering)

Comprehensive QoS deployment using DiffServ classification, Modular QoS CLI (MQC), hierarchical shaping and policing, CBWFQ/LLQ queuing, WRED congestion avoidance, MPLS QoS models, and application-specific tuning for voice, video, and data across enterprise and service provider networks.

## DiffServ Architecture

### DSCP / PHB Complete Mapping Table

```
PHB Class    DSCP Name   Binary      Decimal   IP Prec   Per-Hop Behavior
───────────────────────────────────────────────────────────────────────────────
Default      CS0 / BE    000 000     0         0         Best effort
─────────────────────────────────────────────────────────────────────────────
Class 1      CS1         001 000     8         1         Scavenger / bulk
(Low Pri)    AF11        001 010     10        1         Low drop
             AF12        001 100     12        1         Medium drop
             AF13        001 110     14        1         High drop
─────────────────────────────────────────────────────────────────────────────
Class 2      CS2         010 000     16        2         OAM
(Medium)     AF21        010 010     18        2         Low drop
             AF22        010 100     20        2         Medium drop
             AF23        010 110     22        2         High drop
─────────────────────────────────────────────────────────────────────────────
Class 3      CS3         011 000     24        3         Signaling (SIP/H.323)
(High)       AF31        011 010     26        3         Low drop
             AF32        011 100     28        3         Medium drop
             AF33        011 110     30        3         High drop
─────────────────────────────────────────────────────────────────────────────
Class 4      CS4         100 000     32        4         Real-time interactive
(RT Inter)   AF41        100 010     34        4         Low drop
             AF42        100 100     36        4         Medium drop
             AF43        100 110     38        4         High drop
─────────────────────────────────────────────────────────────────────────────
Class 5      CS5         101 000     40        5         Broadcast video
(EF)         EF          101 110     46        5         Expedited Forwarding
─────────────────────────────────────────────────────────────────────────────
Class 6      CS6         110 000     48        6         Network control
(Network)                                                (routing protocols)
─────────────────────────────────────────────────────────────────────────────
Class 7      CS7         111 000     56        7         Reserved
```

### AF Drop Precedence Matrix

```
             Low Drop (x1)    Med Drop (x2)    High Drop (x3)
Class 1      AF11 (10)        AF12 (12)        AF13 (14)
Class 2      AF21 (18)        AF22 (20)        AF23 (22)
Class 3      AF31 (26)        AF32 (28)        AF33 (30)
Class 4      AF41 (34)        AF42 (36)        AF43 (38)

Formula: DSCP = 8 * class + 2 * drop_precedence
  AF21 = 8*2 + 2*1 = 18
  AF43 = 8*4 + 2*3 = 38
```

## IntServ / RSVP

```
IntServ provides per-flow resource reservation via RSVP signaling:

  Sender ──PATH──► Router1 ──PATH──► Router2 ──PATH──► Receiver
  Sender ◄──RESV── Router1 ◄──RESV── Router2 ◄──RESV── Receiver

PATH message: sender → receiver, installs path state
RESV message: receiver → sender, requests bandwidth reservation

Service Types:
  - Guaranteed Service (RFC 2212): hard delay/bandwidth bounds
  - Controlled Load (RFC 2211): low-loss, low-delay under normal load

! IOS-XE RSVP configuration
ip rsvp bandwidth 100000 50000
! 100 Mbps reservable, 50 Mbps max per flow

interface GigabitEthernet0/0/0
 ip rsvp bandwidth 100000 50000

IntServ limitation: every router on the path must maintain per-flow state.
At N flows through M routers, state = N * M entries.
DiffServ avoids this by aggregating flows into classes at the edge.
```

## Modular QoS CLI (MQC)

### Classification — class-map

```
! Match by DSCP
class-map match-all VOICE
 match dscp ef

! Match by ACL (multi-field classifier)
class-map match-all VIDEO-CONFERENCING
 match access-group name VIDEO-ACL
 match dscp af41

! Match by protocol (NBAR2)
class-map match-any BULK-DATA
 match protocol ftp
 match protocol bittorrent
 match dscp cs1

! Match by CoS (Layer 2)
class-map match-all COS-VOICE
 match cos 5

! Match by MPLS EXP
class-map match-all MPLS-EF
 match mpls experimental topmost 5

! match-all = logical AND (all criteria must match)
! match-any = logical OR (any criterion matches)
```

### Marking — policy-map with set

```
! Classify and mark at ingress (trust boundary)
policy-map INGRESS-MARK
 class VOICE
  set dscp ef
 class VIDEO-CONFERENCING
  set dscp af41
 class SIGNALING
  set dscp cs3
 class BULK-DATA
  set dscp af11
 class class-default
  set dscp default

interface GigabitEthernet0/0/0
 service-policy input INGRESS-MARK
```

### Queuing — CBWFQ + LLQ

```
! Class-Based Weighted Fair Queuing with Low Latency Queue
policy-map WAN-EGRESS
 class VOICE
  priority percent 10          ! LLQ — strict priority with implicit policer
 class VIDEO
  bandwidth percent 30         ! CBWFQ — guaranteed minimum bandwidth
  random-detect dscp-based     ! WRED per DSCP
 class SIGNALING
  bandwidth percent 5
 class TRANSACTIONAL
  bandwidth percent 25
  random-detect
 class BULK
  bandwidth percent 10
  random-detect
 class class-default
  bandwidth percent 20
  random-detect
  fair-queue                   ! WFQ within default class

interface Serial0/0/0
 service-policy output WAN-EGRESS

! Priority keyword = LLQ = strict priority + implicit policer
! Bandwidth keyword = CBWFQ = guaranteed minimum, can burst higher
! Total bandwidth allocation should not exceed 75% (25% reserved for routing/overhead)
```

### Shaping — traffic shaping

```
! Shape output to contracted rate (CIR)
policy-map SHAPE-TO-CIR
 class class-default
  shape average 50000000       ! 50 Mbps average
  shape peak 75000000          ! burst to 75 Mbps (optional)

! Shape with nested queuing policy (H-QoS)
policy-map CUSTOMER-SHAPE
 class class-default
  shape average 100000000      ! 100 Mbps contract
  service-policy WAN-EGRESS    ! nested queuing within shaped rate

interface GigabitEthernet0/0/0.100
 service-policy output CUSTOMER-SHAPE
```

### Policing — single-rate and dual-rate

```
! Single-rate two-color policer
policy-map POLICE-INGRESS
 class CUSTOMER-TRAFFIC
  police cir 50000000 bc 1562500
   conform-action transmit
   exceed-action drop

! Single-rate three-color policer (srTCM — RFC 2697)
policy-map POLICE-3COLOR
 class CUSTOMER-TRAFFIC
  police cir 50000000 bc 1562500 be 3125000
   conform-action transmit
   exceed-action set-dscp-transmit af11
   violate-action drop

! Dual-rate three-color policer (trTCM — RFC 2698)
policy-map POLICE-DUAL-RATE
 class CUSTOMER-TRAFFIC
  police cir 50000000 bc 1562500 pir 75000000 be 2343750
   conform-action transmit
   exceed-action set-dscp-transmit af12
   violate-action drop

! bc (committed burst) = CIR / 8000 * 250ms = CIR * 0.03125
! Typical bc = CIR * 1/32 (supports 1 burst of 31.25ms)
```

## Shaping vs Policing Comparison

```
Feature          Shaping                      Policing
───────────────────────────────────────────────────────────────────
Direction        Egress (output only)         Ingress or Egress
Action           Buffer excess, delay         Drop or re-mark excess
Mechanism        Token bucket + queue         Token bucket, no queue
Effect           Smooth traffic, add delay    Bursty drops, no delay
Use case         WAN egress to match CIR      Ingress from untrusted
Token refill     Per interval (Tc)            Per packet arrival
Burst handling   Bc + Be tokens available     Bc (+ Be for trTCM)

       Shaping:                 Policing:
  ┌──────────────┐         ┌──────────────┐
  │  ═══►Buffer  │         │              │
  │  ══► tokens  ├──► out  │  ═══► check  ├──► conform → transmit
  │  ══► refill  │         │         │     │
  └──────────────┘         │       exceed → drop/remark
                           └──────────────┘
```

## Congestion Avoidance — WRED

```
! WRED drops packets BEFORE the queue is full to avoid tail drop
! TCP Global Synchronization = all flows back off simultaneously after tail drop

! DSCP-based WRED (recommended)
policy-map WRED-POLICY
 class DATA
  bandwidth percent 30
  random-detect dscp-based
  random-detect dscp af21 20 40 10    ! min-thresh max-thresh mark-prob-denom
  random-detect dscp af22 15 35 10    ! higher drop precedence → lower thresholds
  random-detect dscp af23 10 30 10
  random-detect dscp cs2  25 45 10

! min-threshold: queue depth at which random drops BEGIN
! max-threshold: queue depth at which drops reach 100% (tail drop)
! mark-prob-denom: 1/N probability at max-threshold (10 = 10% at max)

! ECN — Explicit Congestion Notification (RFC 3168)
policy-map ECN-POLICY
 class DATA
  bandwidth percent 30
  random-detect dscp-based
  random-detect ecn                   ! mark ECN bits instead of dropping

! ECN uses 2 bits in the IP header (bits 6-7 of ToS byte):
! 00 = Not-ECT (not ECN capable)
! 01 = ECT(1)  (ECN capable)
! 10 = ECT(0)  (ECN capable)
! 11 = CE      (Congestion Experienced — set by router, cleared by receiver ACK)
```

## QoS for Voice

```
Voice Requirements:
  Latency:  < 150ms one-way (< 200ms acceptable)
  Jitter:   < 30ms
  Loss:     < 1%
  Bandwidth: 21-106 kbps per call (codec dependent)

Codec      Bit Rate    Payload   Packets/sec   Bandwidth (L2)
──────────────────────────────────────────────────────────────
G.711      64 kbps     160B      50 pps        87.2 kbps
G.729      8 kbps      20B       50 pps        31.2 kbps
G.722      64 kbps     160B      50 pps        87.2 kbps
Opus       6-510 kbps  variable  50 pps        varies

L2 overhead per packet (Ethernet):
  IP(20) + UDP(8) + RTP(12) + Ethernet(18) + CRC(4) = 62 bytes overhead
  G.711: 160 + 62 = 222 bytes * 50 pps * 8 = 88,800 bps ≈ 87.2 kbps

Voice QoS Policy:
  Mark:     EF (DSCP 46)
  Queue:    LLQ (strict priority)
  Budget:   10-15% of link bandwidth
  Policing: Implicit with priority command

! Voice VLAN QoS with IP phone trust
interface GigabitEthernet0/0/1
 switchport mode access
 switchport access vlan 100
 switchport voice vlan 200
 mls qos trust dscp          ! trust markings from phone
 auto qos voip cisco-phone   ! or use AutoQoS
```

## QoS for Video

```
Video Requirements:
  Latency:  < 200ms one-way (interactive), < 5s (streaming)
  Jitter:   < 30ms (interactive), < 200ms (streaming, with buffer)
  Loss:     < 0.1% (I-frame loss is catastrophic)
  Bandwidth: 384 kbps - 20 Mbps per stream

Type               DSCP        Bandwidth     Loss Sensitivity
──────────────────────────────────────────────────────────────
Interactive video   AF41/CS4   1-6 Mbps      Very high (no retransmit)
Streaming video     AF31       2-20 Mbps     Medium (buffered)
Broadcast video     CS5        1-20 Mbps     Very high

! Video conferencing QoS
class-map match-any VIDEO-CONF
 match dscp af41
 match dscp cs4
 match access-group name VIDEO-ACL

policy-map EGRESS-QOS
 class VIDEO-CONF
  bandwidth percent 30
  random-detect dscp-based
  random-detect dscp af41 30 50 10
  random-detect dscp af42 20 40 10
  random-detect dscp af43 10 30 10
```

## Service Provider QoS — H-QoS

```
Hierarchical QoS: shape per-customer, then queue within that shape

              ┌─────────────────── Parent Policy ──────────────┐
              │  shape average 100 Mbps (customer contract)    │
              │                                                │
              │  ┌── Child Policy ──────────────────────────┐  │
              │  │  class VOICE:  priority 10 Mbps          │  │
              │  │  class VIDEO:  bandwidth 30 Mbps         │  │
              │  │  class DATA:   bandwidth 40 Mbps (WRED)  │  │
              │  │  class DEFAULT: bandwidth 20 Mbps         │  │
              │  └──────────────────────────────────────────┘  │
              └────────────────────────────────────────────────┘

! IOS-XE H-QoS configuration
policy-map CHILD-QOS
 class VOICE
  priority 10000              ! 10 Mbps strict priority
 class VIDEO
  bandwidth 30000             ! 30 Mbps guaranteed
  random-detect dscp-based
 class DATA
  bandwidth 40000
  random-detect
 class class-default
  bandwidth 20000
  fair-queue

policy-map CUSTOMER-A
 class class-default
  shape average 100000000     ! 100 Mbps
  service-policy CHILD-QOS    ! nest child inside parent

interface GigabitEthernet0/0/0.100
 encapsulation dot1Q 100
 service-policy output CUSTOMER-A

! Per-VLAN shaping (common in SP aggregation)
policy-map CUSTOMER-B
 class class-default
  shape average 50000000      ! 50 Mbps
  service-policy CHILD-QOS
```

## QoS Trust Boundaries

```
                    Untrusted Zone            Trust Boundary           Trusted Zone
                                                   │
  [PC/Server] ──► [Access Switch] ──────────────── │ ──► [Distribution] ──► [Core]
                        │                          │
                  Classify & mark               Accept DSCP
                  (MF classifier)               (no reclassification)
                        │
               Trust DSCP from IP Phone ──────────│
               Re-mark all other traffic          │

! Access switch trust configuration
interface GigabitEthernet0/1
 mls qos trust dscp               ! trust phone markings
 mls qos trust cos                ! trust CoS (if applicable)

! Or override — do NOT trust endpoint markings
interface GigabitEthernet0/2
 no mls qos trust
 service-policy input CLASSIFY-AND-MARK

! Distribution/core: trust incoming DSCP
interface GigabitEthernet0/0/0
 mls qos trust dscp
```

## AutoQoS

```
! Cisco AutoQoS generates MQC policy automatically
! Enterprise mode — on WAN interfaces
interface Serial0/0/0
 auto qos

! VoIP mode — on access switchports
interface GigabitEthernet0/1
 auto qos voip cisco-phone
 auto qos voip cisco-softphone
 auto qos voip trust

! View generated policy
show auto qos interface Serial0/0/0
show policy-map interface Serial0/0/0

! AutoQoS generates:
!   - Class-maps for voice, video, signaling, transactional, bulk, scavenger
!   - Policy-map with LLQ for voice, CBWFQ for others, WRED on data
!   - Trust/marking policy on ingress
! Always review and customize generated policy for production
```

## QoS Pre-Classify for Tunnels

```
! Problem: after tunnel encapsulation, original DSCP is hidden inside
! the tunnel header. QoS classification sees only the outer header.
!
! Solution: qos pre-classify examines the inner (original) packet headers
! BEFORE tunnel encapsulation for QoS classification.

interface Tunnel0
 qos pre-classify

! Required for: GRE, IPsec, L2TP, DMVPN, FlexVPN
! Without pre-classify, all tunneled traffic gets same QoS treatment

! IPsec with QoS pre-classify
crypto map IPSEC-MAP 10 ipsec-isakmp
 set peer 203.0.113.1
 set transform-set AES256-SHA
 match address CRYPTO-ACL
 qos pre-classify
```

## MPLS QoS — Uniform, Pipe, Short-Pipe Models

```
Model         Edge Marking        Core Behavior        Egress Behavior
───────────────────────────────────────────────────────────────────────
Uniform       DSCP → EXP          EXP can change       EXP → DSCP (copy back)
              (bidirectional)      (core PHB applied)   (changes propagate)

Pipe          DSCP → EXP          EXP can change       Ignore EXP; use original
              (copy at ingress)    (core PHB applied)   DSCP for egress PHB

Short-Pipe    DSCP → EXP          EXP can change       Use original DSCP for
              (copy at ingress)    (core PHB applied)   egress classification
                                                        (penultimate hop pops label)

! Uniform model: customer and SP share same QoS namespace
! DSCP changes in core are reflected back to customer packet

! Pipe model: SP has independent QoS in core
! Customer original DSCP preserved; SP uses EXP independently

! Short-pipe model: like pipe, but egress PE uses customer DSCP
! Most common in SP networks — provides SP independence + correct egress

! IOS-XE MPLS QoS — Uniform mode (default)
mpls ip
interface GigabitEthernet0/0/0
 mpls ip
 ! DSCP automatically copied to EXP at imposition
 ! EXP automatically copied to DSCP at disposition

! IOS-XE MPLS QoS — Short-pipe mode
policy-map MPLS-INGRESS
 class VOICE
  set mpls experimental imposition 5   ! set EXP without changing DSCP

policy-map MPLS-EGRESS
 class VOICE
  set dscp ef                          ! classify on inner DSCP at egress

! JunOS MPLS QoS
set class-of-service interfaces ge-0/0/0 unit 0 classifiers exp default
set class-of-service interfaces ge-0/0/0 unit 0 rewrite-rules exp default
```

## NX-OS QoS

```
! NX-OS uses system-level queuing (not per-interface MQC)
! Define system QoS policies

! Classification
class-map type qos match-all VOICE
 match dscp ef

! Marking policy (applied at system level)
policy-map type qos SYSTEM-MARKING
 class VOICE
  set qos-group 1

! Queuing policy
class-map type queuing match-any Q-VOICE
 match qos-group 1

policy-map type queuing SYSTEM-QUEUING
 class type queuing Q-VOICE
  priority level 1
  shape min 10 gbps max 10 gbps
 class type queuing class-default
  bandwidth remaining percent 100

! Apply at system level
system qos
 service-policy type qos input SYSTEM-MARKING
 service-policy type queuing output SYSTEM-QUEUING
```

## JunOS QoS (CoS)

```
! JunOS CoS components: classifier, rewrite, scheduler, drop-profile

! Classifier — map incoming DSCP to forwarding class + loss priority
set class-of-service classifiers dscp DSCP-CLASSIFY forwarding-class expedited-forwarding loss-priority low code-points ef
set class-of-service classifiers dscp DSCP-CLASSIFY forwarding-class assured-forwarding loss-priority low code-points af21
set class-of-service classifiers dscp DSCP-CLASSIFY forwarding-class assured-forwarding loss-priority high code-points af23
set class-of-service classifiers dscp DSCP-CLASSIFY forwarding-class best-effort loss-priority low code-points be

! Scheduler — bandwidth allocation and priority
set class-of-service schedulers SCHED-EF transmit-rate percent 10 priority strict-high
set class-of-service schedulers SCHED-AF transmit-rate percent 40 buffer-size percent 40
set class-of-service schedulers SCHED-BE transmit-rate remainder

! Scheduler map
set class-of-service scheduler-maps QOS-MAP forwarding-class expedited-forwarding scheduler SCHED-EF
set class-of-service scheduler-maps QOS-MAP forwarding-class assured-forwarding scheduler SCHED-AF
set class-of-service scheduler-maps QOS-MAP forwarding-class best-effort scheduler SCHED-BE

! Apply to interface
set class-of-service interfaces ge-0/0/0 scheduler-map QOS-MAP
set class-of-service interfaces ge-0/0/0 unit 0 classifiers dscp DSCP-CLASSIFY

! Drop profile (WRED equivalent)
set class-of-service drop-profiles WRED-AF interpolate fill-level 30 drop-probability 0
set class-of-service drop-profiles WRED-AF interpolate fill-level 80 drop-probability 50
set class-of-service drop-profiles WRED-AF interpolate fill-level 100 drop-probability 100

! Rewrite rule — set outgoing DSCP/CoS
set class-of-service rewrite-rules dscp REWRITE-DSCP forwarding-class expedited-forwarding loss-priority low code-point ef
```

## Verification Commands

```
! --- IOS-XE ---
show policy-map interface GigabitEthernet0/0/0    ! per-class stats
show policy-map interface GigabitEthernet0/0/0 input
show class-map                                     ! all class-maps
show policy-map                                    ! all policy-maps
show mls qos interface GigabitEthernet0/0/0        ! trust state
show auto qos                                      ! AutoQoS generated config
show queueing interface GigabitEthernet0/0/0       ! queue depths

! --- NX-OS ---
show queuing interface ethernet 1/1                ! queue stats
show policy-map system type qos                    ! system QoS policy
show policy-map system type queuing                ! queuing policy
show class-map type qos                            ! class-maps

! --- JunOS ---
show class-of-service interface ge-0/0/0           ! CoS stats
show class-of-service interface ge-0/0/0 comprehensive
show interfaces queue ge-0/0/0                     ! per-queue counters
show class-of-service classifier name DSCP-CLASSIFY
show class-of-service scheduler-map QOS-MAP

! --- MPLS QoS ---
show mpls forwarding-table detail                  ! EXP bits
show policy-map interface tunnel0                  ! tunnel QoS
```

## Tips

- Always classify and mark at the access layer ingress, never in the core. The core should trust and preserve markings. Reclassifying in the core wastes CPU and risks inconsistency across paths.
- Cap LLQ (strict priority) bandwidth to 10-15% of the link. Without a cap, a flood of EF-marked traffic will starve all other classes. The priority command includes an implicit policer that enforces this.
- Use WRED on all TCP-carrying queues. Tail drop causes TCP global synchronization where all flows reduce their window simultaneously, recover simultaneously, and create oscillating throughput. WRED drops packets from random flows, keeping most flows in congestion avoidance.
- Never put routing protocol traffic (CS6) in the default queue. Create a dedicated network-control class with guaranteed bandwidth. Routing protocol starvation causes convergence failures far more damaging than any data plane issue.
- For MPLS VPN deployments, use short-pipe model. It gives the SP independent QoS control in the core while preserving the customer's original DSCP for egress classification at the PE. Uniform model leaks SP QoS decisions into customer traffic.
- The committed burst (Bc) determines policing granularity. Too small a Bc drops legitimate bursts; too large allows sustained oversubscription. Start with Bc = CIR / 8000 * 250ms (250ms of traffic at CIR).
- ECN is strictly better than WRED for TCP traffic in data center environments. Instead of dropping packets (which triggers retransmission), ECN marks them, and the receiver signals congestion via TCP ACK. This reduces retransmission overhead while still controlling congestion.
- When troubleshooting QoS, always check `show policy-map interface` counters. Non-zero "exceed" and "violate" counters indicate policing is active. Non-zero "drops" in a queue indicate congestion. Zero "offered rate" in a priority class may indicate misclassification.
- For tunnel interfaces (GRE, IPsec, DMVPN), always enable `qos pre-classify`. Without it, the QoS policy sees only the outer tunnel header and treats all encapsulated traffic identically, defeating the purpose of classification.
- H-QoS parent shaping rate must equal or exceed the sum of all child class guaranteed bandwidths. If the parent shapes at 100 Mbps but child classes guarantee 120 Mbps total, the scheduler cannot honor all guarantees simultaneously.

## See Also

- cos-qos, mpls, mpls-vpn, mpls-te, tc, vlan, bgp, rsvp, ip

## References

- [RFC 2474 — Definition of the DS Field (DSCP)](https://datatracker.ietf.org/doc/html/rfc2474)
- [RFC 2475 — DiffServ Architecture](https://datatracker.ietf.org/doc/html/rfc2475)
- [RFC 2597 — Assured Forwarding PHB Group](https://datatracker.ietf.org/doc/html/rfc2597)
- [RFC 3246 — Expedited Forwarding PHB](https://datatracker.ietf.org/doc/html/rfc3246)
- [RFC 2697 — Single Rate Three Color Marker (srTCM)](https://datatracker.ietf.org/doc/html/rfc2697)
- [RFC 2698 — Two Rate Three Color Marker (trTCM)](https://datatracker.ietf.org/doc/html/rfc2698)
- [RFC 3168 — Explicit Congestion Notification (ECN)](https://datatracker.ietf.org/doc/html/rfc3168)
- [RFC 2205 — RSVP](https://datatracker.ietf.org/doc/html/rfc2205)
- [RFC 3270 — MPLS Support of DiffServ](https://datatracker.ietf.org/doc/html/rfc3270)
- [Cisco MQC Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/qos_mqc/configuration/xe-16/qos-mqc-xe-16-book.html)
- [Cisco Enterprise QoS Design Guide](https://www.cisco.com/c/en/us/td/docs/solutions/Enterprise/QoS/QoSSrnd.html)
- [Juniper CoS Configuration Guide](https://www.juniper.net/documentation/us/en/software/junos/cos/index.html)
