# JunOS Class of Service — Processing Pipeline, Scheduling Algorithms, and End-to-End QoS Design

> *JunOS Class of Service implements DiffServ-compliant QoS entirely in the forwarding plane. Packets are classified into forwarding classes, assigned to hardware queues, serviced by schedulers using configurable algorithms (strict priority, WRR, WFQ), and subject to WRED congestion avoidance and token bucket policing. Understanding the full pipeline — from ingress classification through egress rewrite — is essential for designing predictable, end-to-end service quality across an MPLS SP network.*

---

## 1. CoS Processing Pipeline

### Full Packet Path Through CoS

When a packet enters a JunOS device with CoS configured, it traverses the following stages:

```
Ingress Interface
│
├─ 1. Classification (BA or MF)
│     BA: Read existing header markings (DSCP, 802.1p, MPLS EXP)
│     MF: Firewall filter matches on multiple fields (src/dst, port, protocol)
│     Result: Forwarding class + loss-priority assigned
│
├─ 2. Policing (optional, ingress)
│     Token bucket rate limiter applied
│     Packets exceeding rate: discard or re-mark (loss-priority change)
│
├─ 3. Forwarding (FIB lookup, route decision)
│
├─ 4. Queue assignment
│     Packet placed in hardware queue based on forwarding class
│     Each queue has dedicated buffer space
│
├─ 5. Scheduling
│     Scheduler selects which queue to service next
│     Algorithm: strict priority for high-priority, WRR/WFQ for others
│     Transmit-rate guarantees minimum bandwidth per queue
│
├─ 6. WRED (congestion avoidance)
│     When queue fill-level rises, probabilistic drops begin
│     Drop probability based on fill-level and loss-priority
│     Prevents TCP global synchronization
│
├─ 7. Shaping (optional, egress)
│     Limits total egress rate to configured ceiling
│     Uses token bucket to meter aggregate output
│
└─ 8. Rewrite (egress marking)
      Outgoing header fields rewritten based on forwarding class + loss-priority
      DSCP, 802.1p, or MPLS EXP bits set for downstream QoS treatment
```

### Classification Hierarchy

When both BA and MF classification are configured on the same interface, MF classification (firewall filter) takes precedence:

```
Packet arrives with DSCP=EF
│
├─ BA classifier maps EF → forwarding-class VOICE, loss-priority low
│
└─ Firewall filter matches packet → overrides to forwarding-class DATA
   (filter action forwarding-class/loss-priority wins over BA)
```

The MF override is intentional — it allows edge devices to reclassify traffic that arrives with incorrect or untrusted markings.

---

## 2. Scheduling Algorithms

### Strict Priority

The highest-priority queue is always serviced first. As long as packets exist in the strict-high queue, no other queue is serviced.

```
Queue 0: strict-high (VOICE)
Queue 1: high (VIDEO)
Queue 2: medium-high (DATA)
Queue 3: low (BEST-EFFORT)

Scheduling decision:
  if Queue 0 not empty → service Queue 0
  elif Queue 1 not empty → service Queue 1
  elif Queue 2 not empty → service Queue 2
  else → service Queue 3
```

**Starvation risk**: If Queue 0 sustained traffic exceeds the link rate, Queues 1-3 receive zero service. This is why strict-high should only be used for low-volume, latency-sensitive traffic (voice, keepalives).

**JunOS mitigation**: Transmit-rate with `exact` keyword creates a hard cap. Even a strict-high queue cannot exceed its configured transmit-rate when `exact` is specified:

```
scheduler VOICE {
    transmit-rate exact percent 20;   # hard cap at 20%
    priority strict-high;
}
```

Without `exact`, the transmit-rate is a minimum guarantee — the queue can consume additional bandwidth if available.

### Weighted Round-Robin (WRR)

Non-strict-priority queues are serviced in round-robin order, weighted by their transmit-rate:

```
Queue 1: transmit-rate 40% (weight 4)
Queue 2: transmit-rate 30% (weight 3)
Queue 3: transmit-rate 30% (weight 3)

Round-robin cycle:
  Service Queue 1: 4 units
  Service Queue 2: 3 units
  Service Queue 3: 3 units
  (repeat)
```

WRR is byte-based in JunOS — weights determine bytes served per round, not packets. This prevents large-packet queues from getting unfair bandwidth.

### Weighted Fair Queuing (WFQ)

JunOS implements WFQ as an enhancement to WRR with per-flow fairness within each queue. The scheduler computes virtual finish times:

$$VFT_i = \max(V(t), \; VFT_{i-1}) + \frac{P_i}{w_i}$$

Where:
- $V(t)$ is the system virtual time
- $P_i$ is the packet size
- $w_i$ is the queue weight (derived from transmit-rate)

Packets with the smallest virtual finish time are served first. This ensures proportional fairness even when packet sizes vary across queues.

### Combined Scheduling

JunOS typically runs strict priority for one or two queues and WRR for the remaining:

```
Scheduler decision at each service opportunity:
  1. Service all strict-high queues (in priority order)
  2. If strict-high queues empty or at exact rate cap:
     Service remaining queues via WRR weights
```

This hybrid ensures voice gets absolute priority while remaining traffic receives proportional fair treatment.

---

## 3. WRED — Theory and Drop Probability

### Why WRED Exists

Without WRED, a full queue drops all arriving packets (tail drop). This causes TCP global synchronization — all TCP senders detect loss simultaneously, back off, then ramp up together, creating oscillating throughput.

WRED introduces **probabilistic early drops** before the queue is full, causing different TCP senders to detect loss at different times and back off independently.

### Drop Probability Function

JunOS defines drop profiles as piecewise-linear functions mapping queue fill-level to drop probability:

```
Drop Profile: DP-MEDIUM
Fill Level (%)  │  Drop Probability (%)
────────────────┼──────────────────────
     0          │       0
    50          │       0
    75          │      50
   100          │     100

Graph:
100% ┤                          ╱
     │                        ╱
 50% ┤                  ╱───╱
     │                ╱
  0% ┤──────────────╱
     └──────────────┼──────┼──────┤
                   50%    75%   100%   fill-level
```

### Loss-Priority Differentiation

WRED applies different drop profiles based on loss-priority. High loss-priority packets are dropped more aggressively:

```
Same queue, two profiles:
  loss-priority low  → DP-GENTLE  (drops start at 80% fill)
  loss-priority high → DP-AGGRESSIVE (drops start at 40% fill)

Result: During congestion, high-loss-priority packets are preferentially dropped,
preserving capacity for low-loss-priority (in-profile) packets.
```

This creates a two-tier service within a single forwarding class — compliant traffic (green/low) is protected while excess traffic (yellow-red/high) is dropped first.

### WRED Configuration Design

```
Interpolation points define the curve shape:
  - 2 points: linear ramp
  - 3+ points: piecewise-linear, allows gentle start + steep finish

Conservative profile (real-time):     Aggressive profile (best-effort):
  fill 80% → drop 0%                    fill 25% → drop 0%
  fill 95% → drop 30%                   fill 50% → drop 25%
  fill 100% → drop 100%                 fill 75% → drop 75%
                                         fill 100% → drop 100%
```

---

## 4. Token Bucket Policing — In Depth

### Single Token Bucket (Two-Color)

```
Parameters:
  B = bandwidth-limit (bits/sec, the token fill rate)
  S = burst-size-limit (bytes, the bucket capacity)

State:
  T(t) = current tokens (bytes)

Refill: T increases at B/8 bytes/sec, capped at S
Arrival: packet of size P bytes
  if T >= P: accept (green), T = T - P
  else:      discard (red), T unchanged
```

### Dual Token Bucket (Three-Color, Two-Rate)

RFC 2698 trTCM uses two independent buckets with two rates:

```
Bucket C: capacity = CBS, fills at CIR
Bucket P: capacity = PBS, fills at PIR (PIR >= CIR)

Packet of size P arrives:
  1. If P(t) < P → RED (violate)
  2. Else if C(t) < P → YELLOW (exceed), deduct from P only
  3. Else → GREEN (conform), deduct from both C and P

Color mapping in JunOS:
  GREEN  → loss-priority low
  YELLOW → loss-priority medium-high
  RED    → discard (or loss-priority high if color-aware)
```

### Single-Rate Three-Color (srTCM, RFC 2697)

```
Bucket C: capacity = CBS, fills at CIR
Bucket E: capacity = EBS, fills from C overflow

Packet of size P arrives:
  1. If C(t) >= P → GREEN, deduct from C
  2. Else if E(t) >= P → YELLOW, deduct from E
  3. Else → RED

Key difference from trTCM: single rate, excess bucket absorbs bursts above CIR
```

### Burst Size Calculation

The burst-size determines how much traffic can pass instantaneously before policing kicks in:

$$\text{burst-size} = \text{bandwidth-limit} \times \text{max-burst-duration}$$

For a 10 Mbps policer allowing 100ms bursts:

$$S = \frac{10{,}000{,}000}{8} \times 0.1 = 125{,}000 \text{ bytes}$$

Minimum burst-size for reliable operation:

$$S_{min} = 10 \times \text{MTU} = 10 \times 1500 = 15{,}000 \text{ bytes}$$

### Hierarchical Policers

Hierarchical policers enforce aggregate and per-class limits simultaneously:

```
Aggregate policer: 100 Mbps total
├── Premium child: 30 Mbps guaranteed
└── Standard child: gets remainder (up to 70 Mbps)

Processing:
  1. Check aggregate bucket — if exceeded, drop regardless of class
  2. If aggregate permits, check child bucket for packet's class
  3. Premium traffic: passes if within 30 Mbps child limit
  4. Standard traffic: passes if aggregate has room after premium
```

This is commonly used in wholesale/retail SP models where a customer buys 100 Mbps total with 30 Mbps premium SLA.

---

## 5. CoS in MPLS Networks

### The EXP Bit Problem

MPLS headers have only 3 EXP bits (now called Traffic Class per RFC 5462), providing 8 possible values. DSCP has 6 bits (64 values). A mapping strategy is required at MPLS boundaries.

### End-to-End MPLS QoS Design

```
Customer      Ingress PE        P Router         Egress PE       Customer
Network       (Edge)            (Core)           (Edge)          Network
─────────────────────────────────────────────────────────────────────────
IP+DSCP  →  [Classify DSCP]  →  [Classify EXP] → [Classify EXP] → IP+DSCP
             [Push label]        [Swap label]      [Pop label]
             [Set EXP]          [Preserve EXP]    [Set DSCP]
             [Schedule]         [Schedule]         [Schedule]
```

**Ingress PE responsibilities:**
1. Classify customer DSCP into forwarding classes
2. Push MPLS label with EXP bits matching the forwarding class
3. Apply ingress policing per customer contract

**P router responsibilities:**
1. Classify based on EXP bits (not IP DSCP — IP header may be deep in label stack)
2. Schedule queues based on forwarding class
3. Preserve EXP bits on label swap (or rewrite if topology changes class treatment)

**Egress PE responsibilities:**
1. Pop MPLS label
2. Map forwarding class back to customer-appropriate DSCP
3. Rewrite IP DSCP for customer-facing interface

### Uniform vs Pipe vs Short-Pipe Models

```
Uniform Model:
  - EXP copied from IP DSCP at ingress
  - EXP changes in core propagate back to IP DSCP at egress
  - SP and customer share a single QoS domain

Pipe Model:
  - SP sets EXP independently of customer DSCP
  - Customer DSCP preserved unchanged through the core
  - At egress, original customer DSCP restored (from inner label or saved state)
  - SP and customer have independent QoS domains

Short-Pipe Model:
  - Like Pipe, but egress PE uses the customer DSCP (not SP EXP) for egress scheduling
  - SP QoS applies within core, customer QoS applies on egress interface
```

JunOS defaults to uniform model. Pipe model requires explicit `no-decrement-ttl` and EXP configuration on tunnel interfaces.

---

## 6. End-to-End QoS Design with JunOS

### Service Provider QoS Architecture

```
                    Classification          Core               Egress
Customer ──── CE ──── PE (ingress) ──── P ──── P ──── PE (egress) ──── CE ──── Customer

Functions:     MF classify          BA classify (EXP)        BA classify
               Police per SLA       Schedule only            Rewrite DSCP
               Set EXP bits         WRED per queue           Shape per customer
               Shape per customer                            Police per SLA
```

### Design Steps

1. **Define service classes**: Map business requirements to forwarding classes (typically 4-6 classes)
2. **Define SLA parameters**: Bandwidth, latency, jitter, loss targets per class
3. **Design classification**: DSCP trust model (trust customer markings or reclassify at edge)
4. **Design policing**: Per-customer, per-class rate limits at ingress PE
5. **Design scheduling**: Queue weights, priorities, buffer allocation per class
6. **Design WRED profiles**: Drop curves per class, per loss-priority
7. **Design rewrite**: Consistent EXP/DSCP mapping across all PE/P devices
8. **Design shaping**: Per-customer egress shaping on PE

### Typical SP CoS Class Design

| Class | DSCP | EXP | Priority | Transmit Rate | Buffer | WRED |
|:---|:---|:---|:---|:---|:---|:---|
| Voice | EF | 5 | strict-high | 10% exact | 5% | none |
| Video | AF41 | 4 | high | 20% | 15% | gentle |
| Business | AF21-AF23 | 2-3 | medium-high | 30% | 30% | moderate |
| Best-Effort | BE (0) | 0 | low | remainder | remainder | aggressive |

### Scaling Considerations

- TCAM for MF classification at edge: use prefix-lists, minimize inline terms
- BA classification in core: lightweight, no TCAM consumption for classification
- Policer instances: one per customer-class pair at PE (can number in thousands)
- Scheduler maps: typically 1-3 across the network (edge, core, access)
- Testing: use `show interfaces queue` and traffic generators to validate under load

---

## 7. CoS Interaction with Other Features

### CoS and Firewall Filters

Firewall filter actions (`forwarding-class`, `loss-priority`) directly set CoS classification. A filter applied as `input` on an interface overrides any BA classifier on the same interface.

### CoS and Routing Policy

Routing policy operates in the control plane and does not directly affect CoS. However, routing policy can set communities that influence downstream CoS decisions (e.g., a community triggers MF classification at the next hop).

### CoS and MPLS

CoS classifiers and rewrite rules support MPLS EXP natively. On label push, the EXP bits are set based on the forwarding class. On label pop, the forwarding class can be mapped back to IP DSCP via rewrite rules.

### CoS and Layer 2

802.1p classifiers and rewrite rules apply to VLAN-tagged interfaces. On bridged interfaces, CoS uses 802.1p bits for classification and rewrite rather than IP DSCP.

## Prerequisites

- IP networking fundamentals, TCP congestion control basics, MPLS label operations, firewall filter configuration, interface configuration

## Complexity

| Operation | Time | Space |
|-----------|------|-------|
| BA classification (DSCP/EXP lookup) | O(1) | O(1) |
| MF classification (TCAM lookup) | O(1) | O(terms) |
| Token bucket policer decision | O(1) | O(1) per policer |
| Scheduler queue selection (strict + WRR) | O(queues) | O(queues) |
| WRED drop decision | O(1) | O(1) per profile |

---

*CoS is the only mechanism that provides deterministic service guarantees on a shared network. Without it, all traffic competes equally during congestion, and real-time applications fail. The key insight is that CoS does not create bandwidth — it allocates existing bandwidth according to business priorities, accepting that protecting one class necessarily degrades another.*
