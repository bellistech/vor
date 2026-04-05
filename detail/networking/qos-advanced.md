# Advanced QoS — Token Buckets, Scheduling Theory, and End-to-End Deployment

> *Quality of Service in packet networks is fundamentally about managing the conflict between statistical multiplexing efficiency and deterministic performance guarantees. Statistical multiplexing — sharing a link among many flows that burst at different times — achieves higher utilization than circuit switching, but at the cost of variable delay, jitter, and potential loss during congestion. QoS mechanisms restore predictability by classifying traffic into behavior aggregates, allocating resources per class, and applying different forwarding treatments (per-hop behaviors). This document covers the mathematical foundations of token bucket algorithms, the GPS model underlying weighted fair queuing, WRED probability curves, DiffServ PHB specifications, IntServ signaling theory, hierarchical scheduling, and the engineering calculations required for end-to-end voice quality.*

---

## 1. Token Bucket Algorithms

### 1.1 Single-Rate Token Bucket (Policer)

The token bucket is the fundamental rate-limiting mechanism in QoS. It consists of a bucket of depth B (burst size) that fills with tokens at rate R (committed information rate). Each token permits transmission of one bit (or one byte, depending on implementation).

**Algorithm (single-rate, two-color):**

```
State: tokens = B (bucket starts full)
Constants: R = CIR (token refill rate), B = Bc (committed burst)

On packet arrival of size L:
  1. Refill tokens: tokens = min(B, tokens + R * (now - last_update))
  2. last_update = now
  3. If tokens >= L:
       tokens = tokens - L
       Action: CONFORM (transmit)
     Else:
       Action: EXCEED (drop or re-mark)
```

The token refill is continuous (conceptually) but implemented as a lazy update on each packet arrival. The time since the last packet determines how many tokens have accumulated.

**Burst behavior:**

When the bucket is full (tokens = B), a burst of B bits can be transmitted instantaneously at line rate before the policer begins dropping. This accommodates the natural burstiness of TCP (which sends window-sized bursts). If B is too small, TCP sawtooth oscillations are clipped, reducing throughput. If B is too large, the policer permits sustained overruns.

**Recommended burst sizing:**

```
Bc = CIR * Tc

Where:
  Bc = committed burst size (bits)
  CIR = committed information rate (bps)
  Tc = measurement interval (seconds)

Common values:
  Tc = 0.25s (250ms) for WAN circuits
  Tc = 0.125s (125ms) for data center

Example:
  CIR = 50 Mbps, Tc = 250ms
  Bc = 50,000,000 * 0.25 = 12,500,000 bits = 1,562,500 bytes ≈ 1.5 MB
```

### 1.2 Single-Rate Three Color Marker (srTCM, RFC 2697)

The srTCM extends the two-color model with an excess burst (Be) bucket that catches traffic exceeding Bc but within a tolerance:

```
State: Tc = Bc (committed tokens), Te = Be (excess tokens)
Constants: CIR, Bc, Be

On packet arrival of size L:
  1. Refill Tc: Tc = min(Bc, Tc + CIR * elapsed)
  2. Refill Te: Te = min(Be, Te + overflow_from_Tc)
     Note: tokens only flow from CIR → Tc → Te (overflow)
  3. If Tc >= L:
       Tc = Tc - L
       Color: GREEN (conform)
     Else if Te >= L:
       Te = Te - L
       Color: YELLOW (exceed)
     Else:
       Color: RED (violate)
```

The key insight is that Te (excess bucket) is only refilled by overflow from Tc. Tokens accumulate in Te only when Tc is full (i.e., the source is sending below CIR). This means Be provides credit for past underutilization: a flow that was idle for a while earns excess burst capacity.

**Typical actions:**
- GREEN (conform): transmit, preserve DSCP
- YELLOW (exceed): transmit, re-mark to higher drop precedence (AF11 → AF12)
- RED (violate): drop

### 1.3 Two Rate Three Color Marker (trTCM, RFC 2698)

The trTCM uses two independent token buckets with separate rates:

```
State: Tc = Bc, Tp = Bp
Constants: CIR, Bc, PIR (peak information rate), Bp (peak burst)

On packet arrival of size L:
  1. Refill Tc at rate CIR: Tc = min(Bc, Tc + CIR * elapsed)
  2. Refill Tp at rate PIR: Tp = min(Bp, Tp + PIR * elapsed)
  3. If Tp < L:
       Color: RED (violate) — exceeds peak rate
     Else if Tc < L:
       Tp = Tp - L
       Color: YELLOW (exceed) — between CIR and PIR
     Else:
       Tc = Tc - L
       Tp = Tp - L
       Color: GREEN (conform) — within CIR
```

The trTCM differs fundamentally from srTCM: the two buckets are independent, refilled at different rates (CIR and PIR). This creates a clear two-tier service: traffic up to CIR is guaranteed (green), traffic between CIR and PIR is allowed but may be dropped during congestion (yellow), and traffic above PIR is always dropped (red).

**srTCM vs trTCM comparison:**

| Aspect | srTCM (RFC 2697) | trTCM (RFC 2698) |
|:---|:---|:---|
| Rates | Single (CIR) | Dual (CIR + PIR) |
| Excess bucket refill | Overflow from committed | Independent at PIR |
| Use case | Reward past underuse | Enforce peak rate ceiling |
| Burst behavior | Long idle → large burst allowed | Peak burst always bounded by Bp |
| Typical deployment | Enterprise | Service provider |

### 1.4 Shaping Token Bucket

Shaping uses the same token bucket but adds a buffer queue. When tokens are insufficient, packets are queued rather than dropped:

```
State: tokens = Bc
Constants: CIR, Bc, Be (optional extended burst), Tc (shaping interval)

On packet arrival of size L:
  1. If tokens >= L:
       tokens = tokens - L
       Send packet immediately
     Else:
       Enqueue packet
       Schedule transmission when tokens accumulate

On timer (every Tc interval):
  1. tokens = min(Bc + Be, tokens + CIR * Tc)
  2. While queue is non-empty AND tokens >= front_packet_size:
       Dequeue and send front packet
       tokens = tokens - front_packet_size
```

Shaping produces smooth output at the configured rate but introduces delay (serialization delay + queuing delay in the shaping buffer). The maximum delay is bounded by the buffer size: max_delay = buffer_size / CIR.

---

## 2. Weighted Fair Queuing and the GPS Model

### 2.1 Generalized Processor Sharing (GPS)

Weighted Fair Queuing (WFQ) is an approximation of the idealized Generalized Processor Sharing (GPS) model. GPS assumes infinitely divisible traffic — each flow receives service simultaneously in proportion to its weight.

**GPS model:**

```
Given:
  N active flows with weights w1, w2, ..., wN
  Link capacity C

Flow i receives service rate:
  ri = C * wi / sum(wj for all active j)

Example:
  C = 100 Mbps, three flows with weights w1=4, w2=2, w3=1
  r1 = 100 * 4/7 = 57.14 Mbps
  r2 = 100 * 2/7 = 28.57 Mbps
  r3 = 100 * 1/7 = 14.29 Mbps
```

GPS provides perfect fairness and isolation: each flow's delay depends only on its own rate, not on other flows' behavior. However, GPS is unrealizable because real packets are indivisible — you cannot serve a fraction of a packet from multiple flows simultaneously.

### 2.2 WFQ as GPS Approximation

WFQ approximates GPS by computing a virtual finish time for each packet and serving the packet with the earliest finish time:

```
For flow i, packet k with length L_k:
  Virtual finish time: F_i(k) = max(F_i(k-1), V(arrival_time)) + L_k / w_i

Where:
  F_i(k-1) = finish time of previous packet in flow i
  V(t) = virtual time function (advances proportionally to link capacity / active flows)
  w_i = weight of flow i
```

**WFQ guarantees:**

1. **Fairness:** Each flow receives bandwidth proportional to its weight within one maximum-packet-size time of GPS
2. **Delay bound:** For a flow with weight w and burst sigma, the maximum delay is bounded by: D_max = sigma / (C * w / sum_w) + L_max / C, where L_max is the maximum packet size
3. **Work-conserving:** If any flow has packets queued and the link is idle, a packet will be sent (no bandwidth is wasted)

### 2.3 CBWFQ Weight Computation

Class-Based WFQ (CBWFQ) extends WFQ from per-flow to per-class. The weight for each class is derived from its configured bandwidth:

```
Given:
  Class bandwidth allocation: BW_i (from 'bandwidth' command)
  Interface speed: C

Weight calculation:
  w_i = BW_i / GCD(all BW_i values)

Effective rate when all classes are active:
  rate_i = C * BW_i / sum(BW_j for all active j)

Example:
  C = 100 Mbps
  Voice:  bandwidth 10000 (kbps)    → weight = 10
  Video:  bandwidth 30000           → weight = 30
  Data:   bandwidth 40000           → weight = 40
  Default: bandwidth 20000          → weight = 20
  sum = 100

  When all active: Voice gets 10 Mbps, Video gets 30 Mbps, etc.
  When Voice is idle: remaining classes share proportionally:
    Video:  100 * 30/90 = 33.33 Mbps
    Data:   100 * 40/90 = 44.44 Mbps
    Default: 100 * 20/90 = 22.22 Mbps
```

CBWFQ guarantees are minimums, not maximums. When a class is idle, its bandwidth is redistributed proportionally among active classes (work-conserving).

### 2.4 Low Latency Queuing (LLQ)

LLQ adds a strict-priority queue to CBWFQ. The priority queue is always served first, but with a policer to prevent starvation:

```
Scheduling algorithm:
  1. If priority queue has packets AND priority policer permits:
       Dequeue and send from priority queue
  2. Else:
       Serve CBWFQ classes in order of virtual finish time

Priority policer:
  Built-in token bucket at the configured priority rate
  Conforming packets: sent immediately (strict priority)
  Exceeding packets: DROPPED (not queued, not re-marked)

This means:
  - Voice gets near-zero queuing delay when within its allocation
  - Voice above its allocation is hard-dropped
  - Non-voice classes are never starved beyond the priority allocation
```

---

## 3. WRED Probability Curves

### 3.1 Random Early Detection Algorithm

WRED extends RED by applying different drop probabilities based on traffic class (DSCP). The core algorithm:

```
State: avg_queue_depth (exponentially weighted moving average)

On packet arrival:
  1. Update average: avg = (1 - 2^(-n)) * avg + 2^(-n) * current_depth
     where n = exponential weight factor (default 9, range 1-16)
     Higher n → slower response to queue changes (smoother)

  2. Look up packet's DSCP in WRED profile to get (min_th, max_th, mark_prob_denom)

  3. If avg < min_threshold:
       ENQUEUE (no drop)
  4. If avg >= max_threshold:
       DROP (tail drop beyond max)
  5. If min_threshold <= avg < max_threshold:
       Calculate drop probability:
       p_temp = mark_prob * (avg - min_th) / (max_th - min_th)
       p_actual = p_temp / (1 - count * p_temp)
       where count = packets since last drop

       Drop with probability p_actual
```

### 3.2 Drop Probability Curve

```
Drop
Prob    1.0 ──────────────────────────────────────────────── tail drop
 (%)    │                                           ╱│
        │                                          ╱ │
        │                                         ╱  │
  1/mpd │─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ╱─ ─│
        │                                      ╱    │
        │                                     ╱     │
        │                                    ╱      │
        │                                   ╱       │
        │                                  ╱        │
    0   │_________________________________╱_________│________
        0                  min_th       max_th     queue_max
                      Average Queue Depth

  Where 1/mpd = 1/mark_prob_denom (max drop probability before tail drop)
  mpd=10 → max 10% drop probability at max_threshold
  mpd=5  → max 20% drop probability at max_threshold
```

### 3.3 AF Drop Precedence WRED Configuration Theory

Within an AF class (e.g., AF2x), the three drop precedences have progressively more aggressive WRED profiles:

```
For AF2x (transactional data) on a queue with max depth 64 packets:

                AF21 (low drop)     AF22 (med drop)    AF23 (high drop)
min_threshold   30 packets          20 packets         10 packets
max_threshold   50 packets          40 packets         30 packets
mark_prob       1/10 (10%)          1/8 (12.5%)        1/5 (20%)

Result: During congestion buildup:
  1. AF23 starts dropping first (at 10 packets avg depth)
  2. AF22 starts dropping next (at 20 packets)
  3. AF21 starts dropping last (at 30 packets)

This implements in-class drop priority: packets marked AF23 (high drop)
are preferentially dropped over AF21 (low drop) within the same
bandwidth class.
```

### 3.4 WRED Exponential Weight Factor

The exponential weight factor (n) controls how quickly the average queue depth responds to changes in actual queue depth:

```
avg_new = (1 - 2^(-n)) * avg_old + 2^(-n) * instantaneous_depth

n=1:  avg = 0.5 * avg + 0.5 * depth    → very responsive, jittery
n=4:  avg = 0.9375 * avg + 0.0625 * depth  → moderate smoothing
n=9:  avg = 0.998 * avg + 0.002 * depth    → very smooth, slow response (default)
n=16: avg = 0.999985 * avg + 0.000015 * depth → extremely slow

Trade-off:
  - Small n: drops react quickly to bursts → may drop during short spikes
  - Large n: drops react slowly → absorbs short bursts but slow to respond to sustained congestion

Recommendation: n=9 (default) for most environments. Decrease to n=4-6 for
latency-sensitive links where you want faster congestion response.
```

---

## 4. DiffServ Per-Hop Behavior Specifications

### 4.1 Expedited Forwarding (EF, RFC 3246)

EF provides a service with low loss, low latency, low jitter, and assured bandwidth. The formal specification defines EF as a forwarding treatment where the departure rate of EF aggregate traffic from any DiffServ node must equal or exceed a configurable rate, independent of the intensity of any other traffic.

**Formal requirement:** The minimum departure rate from a node for EF traffic must be at least the configured EF rate, and the maximum latency experienced by an EF packet at a single hop must be bounded.

**Implementation via LLQ:**

```
EF is implemented as a strict priority queue with a policer:

  Guarantee:   Packets within the EF rate get near-zero queuing delay
  Policing:    Packets exceeding the EF rate are dropped
  Constraint:  EF aggregate must be bounded (policed at ingress)

  If EF traffic exceeds the configured rate:
    - Without policing: other classes starve (EF always served first)
    - With policing: excess EF traffic is dropped, preserving fairness

  Practical bound on EF rate: 10-33% of link capacity
  Below 10%: underutilizing the premium service
  Above 33%: risk of starving non-EF classes during bursts
```

### 4.2 Assured Forwarding (AF, RFC 2597)

AF defines four independent AF classes (AF1 through AF4), each with three drop precedence levels. The formal specification requires:

1. **Independence:** An AF class must be treated independently from other AF classes. Congestion in one AF class must not affect the forwarding of another AF class.

2. **Minimum bandwidth:** Each AF class must receive at least its configured share of bandwidth at a congested node.

3. **Drop precedence ordering:** Within an AF class, packets with higher drop precedence must be dropped with greater probability than packets with lower drop precedence. Specifically: P(drop | AFx3) >= P(drop | AFx2) >= P(drop | AFx1).

4. **No reordering:** Packets within the same microflow (same 5-tuple) and same AF class must not be reordered, regardless of drop precedence differences.

**AF class to CBWFQ mapping:**

```
Each AF class maps to a CBWFQ queue with WRED:
  AF1x → Queue with bandwidth guarantee G1, WRED profiles for AF11/12/13
  AF2x → Queue with bandwidth guarantee G2, WRED profiles for AF21/22/23
  AF3x → Queue with bandwidth guarantee G3, WRED profiles for AF31/32/33
  AF4x → Queue with bandwidth guarantee G4, WRED profiles for AF41/42/43

The WRED profile (min_th, max_th, mpd) differs by drop precedence:
  AFx1: high min_th, high max_th → dropped last
  AFx2: medium min_th, medium max_th → dropped second
  AFx3: low min_th, low max_th → dropped first
```

### 4.3 Class Selector (CS, RFC 2474)

CS PHBs provide backward compatibility with IP Precedence (the original 3-bit ToS field). CS0 through CS7 map to DSCP values 0, 8, 16, 24, 32, 40, 48, 56 — the same values as IP Precedence 0-7 shifted left by 3 bits.

**CS ordering requirement:** A DS node must give traffic marked CSi at least as favorable forwarding treatment as traffic marked CSj, when i > j. This means CS6 (network control) must receive better treatment than CS0 (best effort).

---

## 5. IntServ Signaling Theory

### 5.1 RSVP Message Exchange

RSVP (RFC 2205) uses two message types for path setup:

```
PATH Message Flow (sender → receiver):
  Sender → R1 → R2 → R3 → Receiver

  PATH carries:
    - SENDER_TSPEC: traffic specification (token bucket parameters: r, b, p, m, M)
      r = token rate (bytes/sec), b = bucket depth (bytes)
      p = peak rate (bytes/sec), m = minimum policed unit (bytes)
      M = maximum packet size (bytes)
    - PHOP (previous hop): for routing RESV messages back
    - Each router records path state (soft state, refreshed every 30s)

RESV Message Flow (receiver → sender):
  Receiver → R3 → R2 → R1 → Sender

  RESV carries:
    - FLOWSPEC: requested QoS (Guaranteed or Controlled Load)
    - FILTER_SPEC: which flows this reservation applies to
    - Each router performs admission control:
      If sufficient resources: install reservation, forward RESV
      If insufficient: send RESV_ERR back toward receiver
```

### 5.2 Guaranteed Service Delay Bound

The Guaranteed Service (RFC 2212) provides a mathematical delay bound:

```
For a flow with token bucket (r, b) traversing a path of K hops:

Total delay bound:
  D_total = sum(D_k for k=1..K)

Per-hop delay:
  D_k = (b - M) * (p - R_k) / (R_k * (p - r)) + M/R_k + C_k/R_k + D_k_fixed

Where:
  b = token bucket depth (bytes)
  M = maximum packet size
  p = peak rate
  r = token rate
  R_k = reserved rate at hop k
  C_k = rate-dependent error term at hop k (implementation-specific)
  D_k_fixed = rate-independent error term at hop k

Simplified (when R_k = r, the minimum reservation):
  D_k ≈ b/r + C_k/r + D_k_fixed

End-to-end delay for voice:
  b = 640 bytes (G.711, 4 packets * 160 bytes)
  r = 64,000 bytes/sec (G.711 64 kbps)
  5 hops, C_k = 1500 bytes, D_k_fixed = 1ms each

  D_total = 5 * (640/64000 + 1500/64000 + 0.001)
          = 5 * (0.01 + 0.0234 + 0.001)
          = 5 * 0.0344
          = 0.172 seconds = 172 ms

  This barely meets the 150ms one-way target, illustrating why
  IntServ with Guaranteed Service is difficult for multi-hop paths.
```

### 5.3 Scalability Limitations

IntServ requires per-flow state at every router on the path:

```
State per flow per router:
  - RSVP path state: ~200 bytes
  - RSVP reservation state: ~200 bytes
  - Classifier entry: ~100 bytes
  - Scheduler state: ~100 bytes
  Total: ~600 bytes per flow per router

For a core router handling 100,000 voice flows:
  State = 100,000 * 600 = 60 MB (memory)
  RSVP refresh processing = 100,000 / 30s ≈ 3,333 messages/sec (CPU)

This is why IntServ failed at Internet scale. DiffServ aggregates flows
into ~8-16 classes at the edge, requiring only O(1) state per router
regardless of the number of flows.
```

---

## 6. Hierarchical QoS Scheduling Theory

### 6.1 H-QoS Architecture

H-QoS implements a two-level (or multi-level) scheduling hierarchy:

```
Level 1 (Parent): Shaper — enforces aggregate rate per customer/service
Level 2 (Child): Scheduler — allocates bandwidth among traffic classes

Processing order:
  1. Classify packet into class (child level)
  2. Enqueue into appropriate class queue
  3. Child scheduler selects next packet to send (WFQ/LLQ)
  4. Parent shaper checks token bucket:
     - Tokens available → send packet to wire
     - No tokens → hold packet in shaping buffer

                     Wire (physical link)
                          ▲
              ┌───────────┤───────────┐
              │     Parent Shaper     │  ← Level 1: rate limit
              │   (token bucket at    │     per customer
              │    contracted CIR)    │
              └───────────┤───────────┘
                          ▲
              ┌───────────┤───────────┐
              │    Child Scheduler    │  ← Level 2: allocate
              │  LLQ │ CBWFQ │ WFQ   │     among classes
              └──┬────┬────┬────┬────┘
                 │    │    │    │
               Voice Video Data Default   ← Per-class queues
```

### 6.2 Bandwidth Guarantee Consistency

For H-QoS to function correctly, the child class guarantees must be consistent with the parent shaping rate:

```
Constraint: sum(child_bandwidth_i) <= parent_shape_rate

If child guarantees exceed parent rate:
  - During congestion, the parent shaper throttles output
  - Child scheduler cannot drain queues fast enough
  - LLQ priority no longer provides zero-delay service
  - CBWFQ guarantees are violated

Example (CORRECT):
  Parent: shape average 100 Mbps
  Child:  priority 10 Mbps + bandwidth 30 + 40 + 20 = 100 Mbps ✓

Example (INCORRECT):
  Parent: shape average 50 Mbps
  Child:  priority 10 Mbps + bandwidth 30 + 40 + 20 = 100 Mbps ✗
  Result: classes compete for 50 Mbps, guarantees meaningless
```

### 6.3 Multi-Level H-QoS

Some platforms support three or more levels:

```
Level 1: Port/physical interface shaper (10 Gbps port)
Level 2: Per-customer shaper (1 Gbps per customer VLAN)
Level 3: Per-class scheduler within each customer (LLQ/CBWFQ)

This is common in SP aggregation:
  - 10G uplink carries 100 customers
  - Each customer shaped to their SLA (100M-1G)
  - Within each customer, classes scheduled by priority

  10G Port Shaper
      ├── Customer A (1G shape)
      │       ├── Voice (priority 100M)
      │       ├── Video (bandwidth 400M)
      │       └── Data (bandwidth 500M)
      ├── Customer B (500M shape)
      │       ├── Voice (priority 50M)
      │       ├── Video (bandwidth 200M)
      │       └── Data (bandwidth 250M)
      └── ...
```

---

## 7. End-to-End QoS Deployment Theory

### 7.1 The QoS Toolkit per Network Segment

```
Segment          Classification     Queuing         Congestion Avoidance
──────────────────────────────────────────────────────────────────────────
Access Layer     MF classifier      Not usually     Not usually
                 (mark DSCP)        (high speed)    (high speed)

Distribution     Trust DSCP         LLQ/CBWFQ       WRED on data classes
                                    (if congested)

WAN Edge         Trust DSCP         LLQ/CBWFQ       WRED + shaping to CIR
                 Shape to CIR

SP Core (MPLS)   EXP classifier     LLQ/CBWFQ       WRED on EXP classes
                 (from DSCP→EXP)

SP PE Egress     EXP→DSCP rewrite   LLQ/CBWFQ       WRED + shaping
```

### 7.2 Voice Quality Calculations

**Delay budget for VoIP (one-way):**

```
Component                     Budget
─────────────────────────────────────
Codec delay (G.711)           0.125 ms (algorithmic)
Packetization delay           20 ms (160 bytes / 8000 samples/sec)
Serialization delay           variable per link speed
Propagation delay             ~5 ms per 1000 km fiber
Queuing delay (per hop)       variable (target: < 5 ms with LLQ)
Dejitter buffer               20-60 ms (receiver)
─────────────────────────────────────
Total target:                 < 150 ms one-way
Acceptable:                   < 200 ms one-way
Unacceptable:                 > 250 ms one-way (noticeable echo/overlap)
```

**Serialization delay calculation:**

```
Serialization delay = packet_size / link_speed

For G.711 voice packet (214 bytes with all headers):
  Link Speed    Serialization Delay
  64 kbps       26.75 ms          ← significant!
  256 kbps      6.69 ms
  512 kbps      3.34 ms
  1.544 Mbps    1.11 ms (T1)
  10 Mbps       0.17 ms
  100 Mbps      0.017 ms
  1 Gbps        0.0017 ms         ← negligible

Serialization delay is only significant on links below 1 Mbps.
On T1/E1 WAN links, LLQ is critical to avoid voice packets waiting
behind large data packets (a 1500-byte frame takes 7.77 ms on a T1).
```

**Jitter calculation:**

```
Jitter = variation in one-way delay between consecutive packets

Sources of jitter:
  1. Queuing delay variation (primary source)
  2. Serialization delay variation (only if packet sizes vary)
  3. Route changes (different path lengths)

With LLQ:
  Jitter ≈ max_serialization_delay (one large packet in transit)
  On T1:  jitter ≈ 7.77 ms (1500 bytes / 1.544 Mbps)
  On 10M: jitter ≈ 1.2 ms

Without LLQ:
  Jitter ≈ queue_depth * serialization_delay_per_packet
  On T1 with 75-packet queue: jitter ≈ 75 * 7.77ms = 582 ms (unacceptable)
```

**R-value and MOS relationship (E-model, ITU-T G.107):**

```
R-value (0-100) determines voice quality:

R = R0 - Is - Id - Ie + A

Where:
  R0 = 93.2 (base signal-to-noise)
  Is = simultaneous impairment (quantization noise, etc.)
  Id = delay impairment:
       Id = 0.024*d + 0.11*(d-177.3)*H(d-177.3)
       where d = one-way delay (ms), H = Heaviside function
  Ie = equipment impairment (codec loss):
       G.711: Ie = 0 (best quality)
       G.729: Ie = 11 (compression artifacts)
  A  = advantage factor (0 for wired, 5 for mobile)

R-value    MOS      Quality
───────────────────────────
90-100     4.3-4.5  Excellent (toll quality)
80-90      4.0-4.3  Good
70-80      3.6-4.0  Fair (some dissatisfied users)
60-70      3.1-3.6  Poor
< 60       < 3.1    Unacceptable

For G.711 with 100ms one-way delay and 0.5% packet loss:
  Id = 0.024 * 100 = 2.4
  Ie = 0 (G.711)
  Loss impairment ≈ 0.5 * 25 = 12.5 (rough approximation)
  R = 93.2 - 0 - 2.4 - 12.5 + 0 = 78.3 → MOS ≈ 3.9 (Fair/Good boundary)
```

### 7.3 Bandwidth Engineering

**Oversubscription analysis:**

```
Given:
  WAN link: 100 Mbps
  Customers: 50
  Per-customer SLA: 10 Mbps CIR, 20 Mbps PIR

  Total CIR commitment: 50 * 10 = 500 Mbps
  Oversubscription ratio: 500 / 100 = 5:1

  This works if:
  - Peak utilization rarely exceeds 100 Mbps simultaneously
  - Statistical multiplexing gain: not all customers burst at once
  - QoS ensures high-priority traffic survives during peaks

  H-QoS handles this:
  - Parent shaper: 10 Mbps per customer (enforced)
  - Child LLQ: voice guaranteed even at 100% customer utilization
  - Child CBWFQ: data classes share remaining bandwidth
  - WRED: graceful degradation during congestion (no global sync)
```
