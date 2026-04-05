# CoS & QoS (Classification, Queuing & Traffic Shaping)

End-to-end traffic prioritization using DiffServ classification (DSCP, 802.1p CoS), queuing disciplines (WFQ, DWRR, strict priority), policing, shaping, and congestion avoidance (WRED) to guarantee latency, jitter, and throughput SLAs for delay-sensitive applications over shared network infrastructure.

## Why QoS Matters

```
Without QoS, all traffic is best-effort:
  - Voice/video compete with bulk downloads for bandwidth
  - Latency-sensitive apps suffer during congestion
  - A single elephant flow can starve mouse flows
  - No fairness guarantees between applications or tenants

QoS solves three problems:
  1. Bandwidth allocation   — who gets how much
  2. Latency/jitter control — who gets served first
  3. Drop priority          — who gets dropped first during congestion
```

## IntServ vs DiffServ

```
Model       Granularity   Signaling    Scalability   State
────────────────────────────────────────────────────────────────
IntServ     Per-flow      RSVP         Poor          Per-flow state in every router
DiffServ    Per-class     None (PHB)   Excellent     No per-flow state, mark at edge

DiffServ won. Mark traffic at the edge, treat it per-hop-behavior (PHB) in the core.
IntServ is only used in small/controlled environments (MPLS TE uses RSVP concepts).
```

## Trust Boundaries

```
Trust boundary = the point where QoS markings are accepted or overwritten

                  Untrusted          Trust Boundary         Trusted
  [End Host] ───> [Access Port] ───> [Distribution] ───> [Core]
                       ^
                       |
              Classify & mark here
              (multi-field classifier)

Rules:
  - Never trust markings from end hosts (they can set DSCP to EF)
  - Classify and mark at the access layer ingress
  - Trust markings from IP phones (CDP/LLDP negotiated, separate voice VLAN)
  - Core routers should trust and preserve markings (no reclassification)
```

## DSCP Values (DiffServ Code Point)

```
DSCP occupies bits 0-5 of the IPv4 ToS byte / IPv6 Traffic Class byte

 0   1   2   3   4   5   6   7
+---+---+---+---+---+---+---+---+
|     DSCP (6 bits)     | ECN   |
+---+---+---+---+---+---+---+---+

PHB Name          DSCP Binary   DSCP Decimal   IP Precedence   Use Case
──────────────────────────────────────────────────────────────────────────────
CS0 / BE          000 000       0              0 (Routine)     Best effort / default
CS1               001 000       8              1 (Priority)    Scavenger / bulk
AF11              001 010       10             1               Low-drop assured fwd
AF12              001 100       12             1               Med-drop assured fwd
AF13              001 110       14             1               High-drop assured fwd
CS2               010 000       16             2 (Immediate)   OAM
AF21              010 010       18             2               Low-drop assured fwd
AF22              010 100       20             2               Med-drop assured fwd
AF23              010 110       22             2               High-drop assured fwd
CS3               011 000       24             3 (Flash)       Signaling
AF31              011 010       26             3               Low-drop assured fwd
AF32              011 100       28             3               Med-drop assured fwd
AF33              011 110       30             3               High-drop assured fwd
CS4               100 000       32             4 (Flash Ovrd)  Real-time interactive
AF41              100 010       34             4               Low-drop assured fwd
AF42              100 100       36             4               Med-drop assured fwd
AF43              100 110       38             4               High-drop assured fwd
CS5               101 000       40             5 (Critical)    Broadcast video
EF                101 110       46             5               Expedited fwd (voice)
CS6               110 000       48             6 (Internetwk)  Network control (OSPF, BGP)
CS7               111 000       56             7 (Network)     Reserved

AF naming: AFxy  x = class (1-4), y = drop precedence (1=low, 2=med, 3=high)
  Higher class = higher priority
  Higher drop precedence = dropped sooner during congestion
```

## 802.1p CoS Values (PCP in 802.1Q Tag)

```
3-bit Priority Code Point (PCP) in the 802.1Q VLAN tag

 0                   1
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| PCP |D|       VLAN ID         |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

PCP   Priority Name              Typical Mapping
─────────────────────────────────────────────────────
0     Best Effort (BE)           DSCP 0 (default)
1     Background (BK)            DSCP 8 (CS1)
2     Excellent Effort (EE)      DSCP 16 (CS2) / AF21
3     Critical Applications      DSCP 24 (CS3) / AF31
4     Video (< 100ms latency)    DSCP 32 (CS4) / AF41
5     Voice (< 10ms latency)     DSCP 46 (EF)
6     Internetwork Control       DSCP 48 (CS6)
7     Network Control            DSCP 56 (CS7)

# CoS is Layer 2 only — stripped when frame is routed
# Must map CoS <-> DSCP at L2/L3 boundaries
```

## Classification Methods

```
Behavior Aggregate (BA) classification:
  - Reads existing DSCP or CoS markings in packet header
  - Used at trust boundaries inside the network
  - Fast: single field lookup

Multi-Field (MF) classification:
  - Matches on multiple fields: src/dst IP, port, protocol, VLAN, etc.
  - Used at the network edge (access layer)
  - Flexible but more CPU/TCAM intensive

IP Precedence (legacy):
  - 3-bit field (bits 0-2 of ToS byte), values 0-7
  - Superseded by DSCP (backward compatible: CS values match)
```

## Marking

```bash
# Marking = setting DSCP/CoS/MPLS EXP values on packets
# Always mark at ingress on the access switch (trust boundary)

# Cisco IOS — class-map + policy-map
class-map match-any VOICE
  match protocol rtp
  match dscp ef
class-map match-any SIGNALING
  match protocol sip
  match dscp cs3

policy-map MARK-INGRESS
  class VOICE
    set dscp ef
  class SIGNALING
    set dscp cs3
  class class-default
    set dscp default

interface GigabitEthernet0/1
  service-policy input MARK-INGRESS

# Junos — firewall filter to set DSCP
set firewall family inet filter CLASSIFY term VOICE from protocol udp
set firewall family inet filter CLASSIFY term VOICE from port 5060-5061
set firewall family inet filter CLASSIFY term VOICE then forwarding-class expedited-forwarding
set firewall family inet filter CLASSIFY term VOICE then loss-priority low
set firewall family inet filter CLASSIFY term DEFAULT then forwarding-class best-effort

set interfaces ge-0/0/0 unit 0 family inet filter input CLASSIFY
```

## Queuing (Queue Assignment)

```
Packets are placed into queues based on forwarding class / DSCP

Typical 4-Queue Model:
  Queue 0: Best Effort        (BE, CS1)          — bulk, scavenger
  Queue 1: Assured Forwarding  (AF1x-AF4x, CS2-CS4) — business apps
  Queue 2: Expedited Forwarding (EF, CS5)         — voice/video
  Queue 3: Network Control     (CS6, CS7)          — routing protocols

8-Queue Model (high-end platforms):
  Q0: Best Effort          Q4: Multimedia Streaming
  Q1: Scavenger            Q5: Voice (EF)
  Q2: Bulk Data            Q6: Internetwork Control (CS6)
  Q3: Transactional Data   Q7: Network Control (CS7)
```

## Scheduling Disciplines

```
Strict Priority (SP):
  - Always serves highest-priority queue first
  - Lower queues starve if high queue is always full
  - Used for voice queue (EF) with bandwidth cap

Weighted Round Robin (WRR):
  - Each queue gets a weight (% of bandwidth)
  - Serves queues in round-robin, proportional to weight
  - Problem: unfair with variable packet sizes

Deficit Weighted Round Robin (DWRR):
  - Fixes WRR's packet-size unfairness
  - Tracks deficit counter: unused bytes carry over to next round
  - Each queue has a quantum (bytes per round)

Weighted Fair Queuing (WFQ):
  - Emulates bit-by-bit round-robin (GPS model)
  - Fair across flows regardless of packet size
  - Computationally expensive (finish-time calculation per packet)

Common combination:
  - Strict Priority for voice (EF) with policer to cap at 10-15%
  - DWRR for remaining queues (AF classes + BE)
```

## Shaping (Output Rate Limiting)

```
Shaping = buffers excess traffic, smooths bursts, delays packets
Token bucket algorithm:
  - Tokens added at committed rate (CIR)
  - Each packet consumes tokens equal to its size
  - If tokens available: transmit immediately
  - If no tokens: buffer packet (queue it), send when tokens refill

            ┌──────────────────┐
  Tokens ──>│  Bucket (Bc)     │──> Conforming traffic
  (CIR)     │  depth = burst   │
            └──────────────────┘
                     │
              Bucket empty?
                     │
              ┌──────┴──────┐
              │ Buffer/Queue │──> Delayed traffic
              └─────────────┘

Parameters:
  CIR  = Committed Information Rate (bits/sec)
  Bc   = Committed Burst Size (bits) — bucket depth
  Tc   = Time interval = Bc / CIR (seconds per refill)
```

```bash
# Cisco — shape to 50 Mbps on egress
policy-map SHAPE-50M
  class class-default
    shape average 50000000

interface GigabitEthernet0/1
  service-policy output SHAPE-50M

# Junos — shaping rate on scheduler
set class-of-service schedulers BE-SCHED shaping-rate 50m
```

## Policing (Input Rate Limiting)

```
Policing = drops or remarks excess traffic immediately (no buffering)
Faster and simpler than shaping, used at ingress

Single-rate two-color policer:
  - Traffic <= CIR: conform (forward)
  - Traffic > CIR: violate (drop or remark to lower DSCP)

Single-rate three-color policer (srTCM, RFC 2697):
  - Conform (green):  <= CIR, within Bc
  - Exceed (yellow):  <= CIR, within Be (excess burst)
  - Violate (red):    > CIR + Be (drop)

Dual-rate three-color policer (trTCM, RFC 2698):
  - Conform (green):  <= CIR
  - Exceed (yellow):  <= PIR (Peak Information Rate)
  - Violate (red):    > PIR (drop)
```

```bash
# Cisco — police to 10 Mbps, remark excess
policy-map POLICE-10M
  class class-default
    police 10000000 1500000 conform-action transmit exceed-action set-dscp-transmit af11

interface GigabitEthernet0/1
  service-policy input POLICE-10M

# Junos — single-rate two-color policer
set firewall policer RATE-LIMIT-10M if-exceeding bandwidth-limit 10m burst-size-limit 1500000
set firewall policer RATE-LIMIT-10M then discard

set interfaces ge-0/0/0 unit 0 family inet policer input RATE-LIMIT-10M
```

## RED / WRED (Congestion Avoidance)

```
Tail Drop (default):
  - Queue fills up, all new packets dropped
  - Causes TCP global synchronization (all flows back off and restart together)

Random Early Detection (RED):
  - Starts randomly dropping packets BEFORE queue is full
  - Drop probability increases as queue depth grows
  - Prevents TCP synchronization by spreading drops across flows

Weighted RED (WRED):
  - Different drop profiles per DSCP/class
  - High-priority traffic (low drop precedence) has higher thresholds
  - Low-priority traffic starts dropping earlier

  Drop
  Prob
  100%|                              ___________
      |                         ____/
      |                    ____/ AF13 (high drop)
      |               ____/
      |          ____/ AF12 (med drop)
      |     ____/
      |____/ AF11 (low drop)
      |__________________________________
   0% |min-th                    max-th    Queue Depth

Parameters per drop profile:
  min-threshold:  queue depth where drops begin
  max-threshold:  queue depth where drop probability = max
  mark-probability: max drop probability (e.g., 1/10 = 10%)
```

```bash
# Junos — drop profile
set class-of-service drop-profiles LOW-DROP fill-level 50 drop-probability 0
set class-of-service drop-profiles LOW-DROP fill-level 80 drop-probability 5
set class-of-service drop-profiles LOW-DROP fill-level 100 drop-probability 100

set class-of-service drop-profiles HIGH-DROP fill-level 25 drop-probability 0
set class-of-service drop-profiles HIGH-DROP fill-level 60 drop-probability 10
set class-of-service drop-profiles HIGH-DROP fill-level 100 drop-probability 100
```

## Junos CoS Architecture

```
Junos CoS pipeline:

  Ingress                                                    Egress
  ┌──────────┐   ┌────────────┐   ┌───────────┐   ┌──────────────┐
  │Classifier│──>│Forwarding  │──>│Scheduler + │──>│Rewrite Rule  │
  │(BA or MF)│   │Class Queue │   │Policer/    │   │(set DSCP/CoS │
  │          │   │Assignment  │   │Shaper      │   │ on egress)   │
  └──────────┘   └────────────┘   └───────────┘   └──────────────┘

Four default forwarding classes:
  best-effort            (queue 0)
  expedited-forwarding   (queue 1)
  assured-forwarding     (queue 2)
  network-control        (queue 3)
```

### Junos Classifier (BA Classification)

```bash
# Classify incoming DSCP to forwarding class + loss priority
set class-of-service classifiers dscp MY-CLASSIFIER import default

set class-of-service classifiers dscp MY-CLASSIFIER forwarding-class best-effort loss-priority low code-points 000000
set class-of-service classifiers dscp MY-CLASSIFIER forwarding-class expedited-forwarding loss-priority low code-points 101110
set class-of-service classifiers dscp MY-CLASSIFIER forwarding-class assured-forwarding loss-priority low code-points 001010
set class-of-service classifiers dscp MY-CLASSIFIER forwarding-class assured-forwarding loss-priority high code-points 001110
set class-of-service classifiers dscp MY-CLASSIFIER forwarding-class network-control loss-priority low code-points 110000

# Apply classifier to interface
set class-of-service interfaces ge-0/0/0 unit 0 classifiers dscp MY-CLASSIFIER
```

### Junos Schedulers and Scheduler Maps

```bash
# Scheduler defines per-queue behavior: bandwidth, priority, buffer, drop profile
set class-of-service schedulers BE-SCHED transmit-rate remainder
set class-of-service schedulers BE-SCHED buffer-size remainder
set class-of-service schedulers BE-SCHED priority low
set class-of-service schedulers BE-SCHED drop-profile-map loss-priority low protocol any drop-profile LOW-DROP
set class-of-service schedulers BE-SCHED drop-profile-map loss-priority high protocol any drop-profile HIGH-DROP

set class-of-service schedulers EF-SCHED transmit-rate percent 15
set class-of-service schedulers EF-SCHED buffer-size percent 10
set class-of-service schedulers EF-SCHED priority strict-high
set class-of-service schedulers EF-SCHED drop-profile-map loss-priority any protocol any drop-profile LOW-DROP

set class-of-service schedulers AF-SCHED transmit-rate percent 30
set class-of-service schedulers AF-SCHED buffer-size percent 25
set class-of-service schedulers AF-SCHED priority medium-high
set class-of-service schedulers AF-SCHED drop-profile-map loss-priority low protocol any drop-profile LOW-DROP
set class-of-service schedulers AF-SCHED drop-profile-map loss-priority high protocol any drop-profile HIGH-DROP

set class-of-service schedulers NC-SCHED transmit-rate percent 5
set class-of-service schedulers NC-SCHED buffer-size percent 5
set class-of-service schedulers NC-SCHED priority medium-low

# Map schedulers to forwarding classes
set class-of-service scheduler-maps MY-SCHED-MAP forwarding-class best-effort scheduler BE-SCHED
set class-of-service scheduler-maps MY-SCHED-MAP forwarding-class expedited-forwarding scheduler EF-SCHED
set class-of-service scheduler-maps MY-SCHED-MAP forwarding-class assured-forwarding scheduler AF-SCHED
set class-of-service scheduler-maps MY-SCHED-MAP forwarding-class network-control scheduler NC-SCHED

# Apply scheduler map to interface
set class-of-service interfaces ge-0/0/0 scheduler-map MY-SCHED-MAP
```

### Junos Rewrite Rules

```bash
# Rewrite DSCP/CoS on egress based on forwarding class + loss priority
set class-of-service rewrite-rules dscp MY-REWRITE forwarding-class best-effort loss-priority low code-point 000000
set class-of-service rewrite-rules dscp MY-REWRITE forwarding-class expedited-forwarding loss-priority low code-point 101110
set class-of-service rewrite-rules dscp MY-REWRITE forwarding-class assured-forwarding loss-priority low code-point 001010
set class-of-service rewrite-rules dscp MY-REWRITE forwarding-class assured-forwarding loss-priority high code-point 001110
set class-of-service rewrite-rules dscp MY-REWRITE forwarding-class network-control loss-priority low code-point 110000

# Apply rewrite rule to interface
set class-of-service interfaces ge-0/0/0 unit 0 rewrite-rules dscp MY-REWRITE
```

### Junos CoS Verification

```bash
# Show CoS configuration
show class-of-service interface ge-0/0/0
show class-of-service classifier name MY-CLASSIFIER
show class-of-service scheduler-map MY-SCHED-MAP
show class-of-service rewrite-rule name MY-REWRITE
show class-of-service drop-profile LOW-DROP

# Show queue statistics
show interfaces queue ge-0/0/0
show interfaces ge-0/0/0 extensive | find "Queue"

# Show forwarding class counters
show class-of-service interface ge-0/0/0 comprehensive
```

## Tips

- Always classify and mark traffic at the access layer ingress (trust boundary). Never trust DSCP markings from untrusted endpoints since any application can set DSCP to EF and steal priority from legitimate voice traffic.
- Cap strict-priority (EF) queue bandwidth to 10-15% of the link. Without a policer, a misbehaving voice VLAN can consume the entire link and starve all other traffic classes indefinitely.
- DSCP is end-to-end (Layer 3) while CoS is per-hop (Layer 2, stripped at routers). You must map CoS to DSCP at every L2/L3 boundary or markings are lost. Junos rewrite rules automate this.
- WRED is essential for TCP-heavy environments. Tail drop causes global TCP synchronization where all flows back off simultaneously, then all burst simultaneously, creating oscillating link utilization between 0% and 100%.
- The AF drop precedence (AFx1, AFx2, AFx3) controls what gets dropped first within a class, not between classes. AF13 is dropped before AF11 during congestion, but AF1x traffic never preempts AF4x traffic.
- Shaping buffers excess traffic (adds delay) while policing drops it immediately (adds loss). Use shaping on egress toward rate-limited links (WAN circuits). Use policing on ingress from untrusted sources.
- On Junos, the default four forwarding classes (best-effort, expedited-forwarding, assured-forwarding, network-control) map to queues 0-3. You can define up to 16 custom forwarding classes but most designs use 4-8.
- In JNCIA-Junos exams, remember the CoS processing order: classify (ingress) then queue then schedule then rewrite (egress). Classifiers and rewrite rules are applied per-interface per-direction.
- Use `show interfaces queue` on Junos to see per-queue packet counts, drops, and tail-drop counters. Non-zero tail drops on the EF queue indicate the strict-priority policer is too low or there is a traffic anomaly.
- MPLS networks use the EXP field (3 bits, like CoS) for QoS. At PE routers, map DSCP to EXP on ingress and EXP back to DSCP on egress to maintain end-to-end QoS across the MPLS core.

## See Also

- vlan, mpls, tcp, ip, iptables, tc, stp

## References

- [RFC 2474 — Definition of the DS Field (DSCP)](https://datatracker.ietf.org/doc/html/rfc2474)
- [RFC 2475 — DiffServ Architecture](https://datatracker.ietf.org/doc/html/rfc2475)
- [RFC 2597 — Assured Forwarding PHB Group](https://datatracker.ietf.org/doc/html/rfc2597)
- [RFC 3246 — Expedited Forwarding PHB](https://datatracker.ietf.org/doc/html/rfc3246)
- [RFC 2697 — Single Rate Three Color Marker (srTCM)](https://datatracker.ietf.org/doc/html/rfc2697)
- [RFC 2698 — Two Rate Three Color Marker (trTCM)](https://datatracker.ietf.org/doc/html/rfc2698)
- [IEEE 802.1p — Traffic Class Expediting](https://standards.ieee.org/standard/802_1p-1998.html)
- [Juniper — CoS Configuration Guide](https://www.juniper.net/documentation/us/en/software/junos/cos/index.html)
- [Juniper — Understanding CoS Classifiers](https://www.juniper.net/documentation/us/en/software/junos/cos/topics/concept/cos-classifiers-overview.html)
