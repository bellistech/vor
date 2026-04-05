# The Theory of Service Provider Quality of Service

> *SP QoS is fundamentally about resource allocation under scarcity. When a link is congested, someone's packets get dropped or delayed. QoS is the policy that decides whose packets suffer and whose are protected. The math is queuing theory; the engineering is making queuing theory work at line rate.*

---

## 1. DiffServ Domain Boundary Theory

### The Trust Model

A DiffServ domain is defined by a consistent set of PHB (Per-Hop Behavior) definitions and a common DSCP-to-behavior mapping. At the boundary between domains, traffic must be conditioned — classified, metered, marked, shaped, or dropped — to conform to the receiving domain's policies.

**The fundamental trust problem:** Customer-facing interfaces cannot trust incoming DSCP markings. A customer could mark all traffic as EF (Expedited Forwarding) to receive priority treatment. The SP edge must therefore:

1. **Classify** traffic based on verifiable criteria (port, protocol, VLAN, access list)
2. **Meter** against the customer's traffic contract (CIR/PIR)
3. **Re-mark** to the SP's internal DSCP scheme
4. **Police** to enforce rate limits

At the SP-to-SP boundary (inter-AS), the same logic applies but with different trust levels. A transit provider may trust DSCP from a paid customer but re-mark traffic from a settlement-free peer.

### PHB Groups and Their Semantics

DiffServ defines four PHB groups:

**Default (BE):** Best-effort forwarding. DSCP 0. No guarantees. Gets whatever bandwidth remains after higher-priority classes are served.

**Expedited Forwarding (EF, RFC 3246):** Low loss, low latency, low jitter. The spec defines EF as: the departure rate of EF traffic from any node must equal or exceed a configurable rate, and EF traffic must not be delayed by non-EF traffic. In practice, this means a strict priority queue with admission control (policing).

The mathematical requirement for EF at a node:

$$R_{\text{EF}} \geq R_{\text{configured}}$$

Where $R_{\text{EF}}$ is the actual departure rate and $R_{\text{configured}}$ is the configured service rate. This must hold even when the output link is fully utilized by non-EF traffic — hence the need for preemptive priority scheduling.

**Assured Forwarding (AF, RFC 2597):** Four classes (AF1-AF4), each with three drop precedences (low/medium/high). Within each class, packets with higher drop precedence are dropped first during congestion. The spec requires:

$$P(\text{drop} | \text{AFx3}) \geq P(\text{drop} | \text{AFx2}) \geq P(\text{drop} | \text{AFx1})$$

This is implemented via WRED with different thresholds per drop precedence.

**Class Selector (CS0-CS7):** Backward compatibility with IP Precedence. CS codepoints map to the three high-order bits of DSCP.

### Domain Boundary Operations

At ingress to a DiffServ domain, traffic conditioning consists of four operations (RFC 2475):

1. **Classifier:** Maps packets to behavior aggregates. Multi-field (MF) classification at edge; BA (behavior aggregate) classification in core.
2. **Meter:** Measures traffic rate against a traffic profile (token bucket). Outputs a conformance level (color).
3. **Marker:** Sets/overwrites the DSCP field based on classification and metering results.
4. **Shaper/Dropper:** Delays (shapes) or discards (drops/polices) packets that exceed the traffic profile.

---

## 2. H-QoS Scheduling Mathematics

### The Shaper-Then-Scheduler Hierarchy

H-QoS implements a two-level scheduling hierarchy:

**Level 1 (Outer / Parent):** A token-bucket shaper that limits the aggregate rate for a customer or service group. The shaper smooths traffic to the contracted rate, buffering bursts up to the configured burst size.

**Level 2 (Inner / Child):** A scheduler (priority + weighted fair queuing) that distributes bandwidth among service classes within the shaped aggregate.

The scheduling decision at each packet departure:

1. The shaper checks if tokens are available (aggregate rate not exceeded)
2. If yes, the scheduler selects which child queue to dequeue from:
   a. Priority queue served first (if non-empty and within police rate)
   b. Remaining bandwidth distributed by weight among WFQ/DWRR queues
3. If no tokens available, all child queues are paused until tokens replenish

### Token Bucket Mathematics

The single-rate token bucket (used for shaping):

- **Token generation rate:** $r$ tokens per second (= CIR in bits/sec, with 1 token = 1 bit)
- **Bucket depth:** $b$ tokens (= CBS in bits)
- **State variable:** $T_c$ = current token count, $0 \leq T_c \leq b$

For a packet of size $L$ bits arriving at time $t$:

$$T_c(t) = \min(T_c(t_{\text{prev}}) + r \cdot (t - t_{\text{prev}}), b)$$

If $T_c(t) \geq L$: packet conforms, $T_c(t) \leftarrow T_c(t) - L$

If $T_c(t) < L$: packet is buffered (shaping) or dropped (policing)

The **dual-rate token bucket** (trTCM, RFC 2698) uses two buckets:

- Bucket $C$: rate $r_C$ (CIR), depth $b_C$ (CBS)
- Bucket $P$: rate $r_P$ (PIR), depth $b_P$ (EBS)

Coloring algorithm:

1. If $T_P < L$: packet is **red** (violating PIR)
2. Else if $T_C < L$: packet is **yellow** (exceeding CIR but within PIR), decrement $T_P$
3. Else: packet is **green** (conforming to CIR), decrement both $T_C$ and $T_P$

### Burst Tolerance

The burst size determines how much traffic can be sent instantaneously above the configured rate. For a shaper with CIR $r$ and CBS $b$:

**Maximum burst duration at line rate $R$:**

$$t_{\text{burst}} = \frac{b}{R - r}$$

**Example:** CIR = 1 Gbps, CBS = 125,000 bytes (1 Mbit), line rate = 10 Gbps:

$$t_{\text{burst}} = \frac{1 \times 10^6}{10 \times 10^9 - 1 \times 10^9} = \frac{10^6}{9 \times 10^9} \approx 111 \text{ microseconds}$$

For TCP performance, CBS should be at least the bandwidth-delay product:

$$\text{CBS} \geq \text{CIR} \times \text{RTT}$$

For CIR = 1 Gbps and RTT = 10 ms:

$$\text{CBS} \geq 1 \times 10^9 \times 0.01 = 10^7 \text{ bits} = 1.25 \text{ MB}$$

### Weighted Fair Queuing Mathematics

In WFQ (or its practical implementation, DWRR), bandwidth is distributed proportionally to configured weights.

For $n$ non-priority queues with weights $w_1, w_2, \ldots, w_n$ and available bandwidth $B_{\text{avail}}$ (total bandwidth minus priority queue consumption):

$$B_i = B_{\text{avail}} \times \frac{w_i}{\sum_{j=1}^{n} w_j}$$

When using "bandwidth remaining percent" (IOS-XR):

$$B_i = (B_{\text{total}} - B_{\text{priority}}) \times \frac{p_i}{100}$$

Where $p_i$ is the configured percent and $\sum p_i = 100$.

**Example:** Interface = 10 Gbps, priority policed at 1 Gbps, three WFQ classes at 40/30/30:

$$B_{\text{avail}} = 10 - 1 = 9 \text{ Gbps}$$

$$B_{\text{video}} = 9 \times 0.40 = 3.6 \text{ Gbps}$$

$$B_{\text{business}} = 9 \times 0.30 = 2.7 \text{ Gbps}$$

$$B_{\text{default}} = 9 \times 0.30 = 2.7 \text{ Gbps}$$

These are **minimum guarantees** during congestion. When a class is idle, its bandwidth is redistributed proportionally to the other active classes.

### DWRR (Deficit Weighted Round Robin)

Standard round robin is unfair for variable-length packets. DWRR fixes this with a deficit counter:

1. Each queue $i$ gets a **quantum** $Q_i$ proportional to its weight
2. A **deficit counter** $D_i$ accumulates unused quantum across rounds
3. In each round, $D_i \leftarrow D_i + Q_i$
4. Dequeue packets from queue $i$ as long as $D_i \geq$ packet size; subtract packet size from $D_i$
5. When $D_i <$ next packet size, move to queue $i+1$

This ensures long-term fairness proportional to quantum ratios, regardless of packet size distribution.

---

## 3. MPLS QoS Model Comparison

### Label Operations and QoS Interaction

MPLS label operations (push, swap, pop) interact with QoS marking in different ways depending on the mode:

**Push (Label Imposition at Ingress PE):**

| Mode | Action |
|:---|:---|
| Uniform | EXP = f(DSCP) — copy DSCP to EXP |
| Pipe | EXP = f(DSCP) — copy DSCP to EXP; DSCP preserved for later restoration |
| Short-Pipe | EXP = f(DSCP) — same as pipe at push |

**Swap (Label Switching in Core):**

| Mode | Action |
|:---|:---|
| Uniform | New EXP = Old EXP (or modified by core QoS policy); if changed, DSCP also updated |
| Pipe | New EXP = Old EXP (or modified); DSCP untouched |
| Short-Pipe | New EXP = Old EXP (or modified); DSCP untouched |

**Pop (Label Disposition at Egress PE):**

| Mode | Action |
|:---|:---|
| Uniform | DSCP = f(EXP) — copy EXP back to DSCP; egress queuing uses this new DSCP |
| Pipe | DSCP = original value (not modified by EXP); egress queuing uses **EXP from the popped label** (requires implementation support) |
| Short-Pipe | DSCP = original value; egress queuing uses **DSCP from IP header** (the key difference from pipe) |

### When to Use Each Model

**Uniform:** The SP network and customer share the same QoS semantics. Changes to EXP in the SP core propagate back to the customer's DSCP. This is appropriate when:
- SP controls both ends (e.g., SP's own infrastructure traffic)
- Customer has delegated QoS decisions to the SP
- Single administrative domain

**Pipe:** The SP core operates its own QoS scheme independently of the customer. Customer DSCP is preserved end-to-end. Egress PE queues based on the MPLS EXP value (the "pipe" through the SP network carries its own QoS state). This is appropriate when:
- L3VPN service where customer DSCP must not be altered
- SP wants to apply its own core QoS independently
- Egress queuing must reflect the SP's classification, not the customer's

**Short-Pipe:** Like pipe, but the egress PE uses the customer's original DSCP for egress queuing (toward the CE). This is appropriate when:
- SP wants to preserve customer DSCP for egress queuing toward the customer
- The SP's EXP scheme should only apply within the MPLS core
- The final hop toward the customer should respect the customer's own markings

### The PHP Complication

Penultimate Hop Popping (PHP) removes the outermost label at the router before the egress PE. This affects QoS because:

- In **pipe mode**, the egress PE needs the EXP from the popped label — but with PHP, the label is already gone. Solutions: use explicit null (label 0/2) instead of implicit null, or use ultimate-hop popping (UHP).
- In **short-pipe mode**, PHP is not a problem because the egress PE uses DSCP from the IP header, which is always available after label pop.
- In **uniform mode**, PHP is not a problem because the DSCP was synchronized with EXP at every swap; after PHP, the DSCP is already correct.

### Label Stack QoS

With stacked labels (e.g., VPN label + transport label), only the **topmost** label's EXP bits are visible for queuing at core routers. The inner label's EXP is irrelevant until the outer label is popped.

When pushing multiple labels, the EXP must be set on all labels. In pipe mode:

1. Inner label (VPN): EXP set from DSCP
2. Outer label (transport): EXP copied from inner label's EXP

This ensures consistent queuing regardless of which label is topmost at any point in the LSP.

---

## 4. Traffic Conditioning at Domain Boundaries

### The srTCM (Single-Rate Three-Color Marker, RFC 2697)

The srTCM uses a single rate (CIR) with two bucket depths (CBS and EBS):

- **Bucket $C$:** depth CBS, refill rate CIR
- **Bucket $E$:** depth EBS, refill rate = overflow from $C$ (tokens that would exceed CBS fill $E$ instead)

Coloring:

1. If $T_C \geq L$: **green**, $T_C \leftarrow T_C - L$
2. Else if $T_E \geq L$: **yellow**, $T_E \leftarrow T_E - L$
3. Else: **red**

The srTCM is useful when the customer has a single committed rate but wants to allow small bursts beyond it (excess traffic carried as yellow, dropped first during congestion).

### The trTCM (Two-Rate Three-Color Marker, RFC 2698)

The trTCM uses two rates (CIR and PIR) with two bucket depths (CBS and PBS):

- **Bucket $P$:** depth PBS, refill rate PIR
- **Bucket $C$:** depth CBS, refill rate CIR

Coloring:

1. If $T_P < L$: **red**
2. Else if $T_C < L$: **yellow**, $T_P \leftarrow T_P - L$
3. Else: **green**, $T_C \leftarrow T_C - L$, $T_P \leftarrow T_P - L$

The trTCM is the standard model for SP traffic contracts because it separates the committed rate (always carried) from the peak rate (hard ceiling), with the gap between them carried as best-effort (yellow).

### Mapping Colors to PHB

The standard approach maps meter colors to AF drop precedences:

| Color | Drop Precedence | Example DSCP | Queuing |
|:---|:---|:---|:---|
| Green | Low (AFx1) | AF31 (DSCP 26) | Forwarded unless extreme congestion |
| Yellow | Medium (AFx2) | AF32 (DSCP 28) | Dropped earlier during congestion (WRED) |
| Red | High (AFx3) | AF33 (DSCP 30) | Dropped first; or policed (dropped immediately) |

In practice, many SP deployments police red traffic (drop immediately) rather than marking AF33, because carrying traffic that will likely be dropped wastes link capacity.

---

## 5. WRED Tuning for Service Providers

### The Problem WRED Solves

Tail drop (the default queue behavior) causes two problems:

1. **TCP global synchronization:** When the queue fills, all TCP flows are dropped simultaneously. They all back off, then ramp up together, causing oscillating congestion cycles.
2. **TCP starvation of small flows:** Large flows with many packets in the queue disproportionately contribute to tail drops, but small flows are equally affected.

WRED drops packets probabilistically before the queue fills. Each TCP flow independently detects loss and backs off, avoiding synchronization.

### WRED Parameters

For each drop precedence (or DSCP), WRED is configured with:

- **Minimum threshold ($\text{min}_{\text{th}}$):** Average queue depth below which no packets are dropped
- **Maximum threshold ($\text{max}_{\text{th}}$):** Average queue depth above which all packets are dropped
- **Mark probability denominator ($\text{max}_p$):** The maximum drop probability at $\text{max}_{\text{th}}$

The drop probability as a function of average queue depth $\bar{q}$:

$$p(\bar{q}) = \begin{cases} 0 & \text{if } \bar{q} < \text{min}_{\text{th}} \\ \frac{\bar{q} - \text{min}_{\text{th}}}{\text{max}_{\text{th}} - \text{min}_{\text{th}}} \times \text{max}_p & \text{if } \text{min}_{\text{th}} \leq \bar{q} \leq \text{max}_{\text{th}} \\ 1 & \text{if } \bar{q} > \text{max}_{\text{th}} \end{cases}$$

### Tuning Guidelines for SP

**Threshold spacing by drop precedence:**

| Precedence | $\text{min}_{\text{th}}$ | $\text{max}_{\text{th}}$ | $\text{max}_p$ |
|:---|:---:|:---:|:---:|
| Low (AFx1 / Green) | 60% of queue | 90% of queue | 1/10 (10%) |
| Medium (AFx2 / Yellow) | 40% of queue | 70% of queue | 1/5 (20%) |
| High (AFx3 / Red) | 20% of queue | 50% of queue | 1/3 (33%) |

This ensures:
- Green traffic is only dropped under severe congestion
- Yellow traffic begins dropping earlier and at a higher rate
- Red traffic is dropped aggressively, providing a clear differentiation

**Queue depth and the EWMA:**

The "average queue depth" used by WRED is an Exponentially Weighted Moving Average:

$$\bar{q}_n = (1 - w) \cdot \bar{q}_{n-1} + w \cdot q_n$$

Where $w$ is the weight factor (typically $2^{-n}$ for hardware efficiency). Smaller $w$ = smoother average, less responsive to short bursts. Larger $w$ = more responsive, but may trigger drops on transient spikes.

For SP environments with bursty traffic (mobile backhaul, video), a moderate weight ($w = 1/16$ to $1/64$) prevents excessive drops on short bursts while still reacting to sustained congestion.

---

## 6. QoS in MPLS VPN

### Per-VRF QoS Architecture

In an MPLS L3VPN, each customer has a VRF (Virtual Routing and Forwarding instance) on the PE router. QoS must provide per-customer isolation:

1. **Ingress PE:** Classify customer traffic from CE, police per contract, mark DSCP, set MPLS EXP on imposed labels
2. **Core (P routers):** Queue based on topmost MPLS EXP — no per-customer awareness, just per-class
3. **Egress PE:** Pop labels, queue toward CE based on DSCP (short-pipe) or EXP (pipe), apply per-customer shaping (H-QoS)

The critical design question: **Where does per-customer scheduling happen?**

- **Ingress PE:** Policing (rate limiting) — enforces the customer's committed rate
- **Egress PE:** Shaping (H-QoS) — smooths output rate toward the CE, prevents customer A from starving customer B on a shared access link
- **Core:** Per-class only, not per-customer — core routers should not maintain per-customer state for scalability

### The Oversubscription Problem

When multiple VPN customers share an aggregation link, the sum of their CIRs may exceed the link capacity. This is by design (statistical multiplexing) but creates QoS challenges:

**Scenario:** 100 customers with 1 Gbps CIR each on a 10 Gbps uplink = 10:1 oversubscription ratio.

If all customers simultaneously send at CIR, total demand = 100 Gbps on a 10 Gbps link. H-QoS at the aggregation node must:

1. Shape each customer to their CIR (first level)
2. Within each customer's shaped output, prioritize voice over data (second level)
3. When the aggregate exceeds link capacity, WRED and scheduling distribute drops fairly

The maximum sustainable oversubscription ratio depends on traffic patterns. For business VPN with typical utilization of 20-30%, a 3:1 to 5:1 ratio is common. For residential services with higher peak-to-mean ratios, 10:1 or higher is used.

---

## 7. End-to-End QoS Across SP Domains

### The Inter-AS QoS Problem

When traffic crosses multiple SP domains, end-to-end QoS requires:

1. **DSCP preservation:** Each SP must not re-mark traffic arbitrarily; inter-AS agreements must specify which DSCPs are honored
2. **Consistent PHB mapping:** EF in SP-A must receive EF treatment in SP-B; this requires bilateral QoS agreements
3. **Traffic conditioning at each boundary:** Each SP polices ingress traffic from the other SP

### Inter-AS QoS Models

**Best-effort interconnection (most common):** No QoS guarantees across domain boundaries. Each SP provides QoS within its own domain. Customer traffic is re-marked to BE at peering points. This is the reality for most internet traffic today.

**Bilateral QoS agreements:** Two SPs agree to honor specific DSCP markings for specific traffic classes. Common for enterprise VPN services where the customer's L3VPN spans multiple SP networks. The SPs agree on:
- Which DSCPs to preserve
- Rate limits per class at the interconnection point
- SLA parameters (latency, jitter, loss per class)

**MEF-defined service models (Carrier Ethernet):** MEF specifications define standardized QoS parameters for Ethernet services across provider boundaries. MEF 23.2 defines performance tiers with specific one-way delay, delay variation, and loss objectives.

### Latency Budget Analysis

For a voice call traversing two SP domains:

| Component | One-Way Delay Budget |
|:---|:---:|
| Codec + packetization | 20 ms |
| SP-A access | 5 ms |
| SP-A core | 10 ms |
| Inter-AS link | 5 ms |
| SP-B core | 10 ms |
| SP-B access | 5 ms |
| De-jitter buffer | 20 ms |
| **Total** | **75 ms** |

ITU-T G.114 recommends one-way delay < 150 ms for voice. The 75 ms budget above is within spec but leaves little margin. Each SP must guarantee its portion of the delay budget through priority queuing and policing.

---

## 8. SLA Measurement and Compliance

### Key SLA Metrics

| Metric | Definition | Measurement Method |
|:---|:---|:---|
| **Availability** | Percentage of time service is operational | $A = \frac{T_{\text{up}}}{T_{\text{total}}} \times 100$ |
| **Latency** | One-way or round-trip delay | TWAMP (RFC 5357), IP SLA, Y.1731 |
| **Jitter** | Variation in inter-packet delay | IPDV per RFC 3393: $J_i = \| (R_i - R_{i-1}) - (S_i - S_{i-1}) \|$ |
| **Packet Loss** | Ratio of dropped to sent packets | $L = \frac{P_{\text{sent}} - P_{\text{received}}}{P_{\text{sent}}} \times 100$ |
| **Throughput** | Sustained achievable bandwidth | RFC 2544 (lab) or production monitoring |

### Availability Calculations

| SLA Level | Downtime per Year | Downtime per Month |
|:---:|:---:|:---:|
| 99.9% ("three nines") | 8h 45m 36s | 43m 12s |
| 99.95% | 4h 22m 48s | 21m 36s |
| 99.99% ("four nines") | 52m 33s | 4m 19s |
| 99.999% ("five nines") | 5m 15s | 25.9s |

The availability of a serial path (where any component failure causes service failure):

$$A_{\text{total}} = \prod_{i=1}^{n} A_i$$

For a path through 4 components each at 99.99%:

$$A_{\text{total}} = 0.9999^4 = 0.9996 = 99.96\%$$

With parallel redundancy (two paths, either suffices):

$$A_{\text{redundant}} = 1 - (1 - A)^2 = 1 - (0.0001)^2 = 1 - 10^{-8} = 99.999999\%$$

### Measurement Probes

**TWAMP (Two-Way Active Measurement Protocol, RFC 5357):**
- Sends test packets between endpoints
- Measures RTT, one-way delay (with synchronized clocks), jitter, and loss
- Can mark test packets with specific DSCP to measure per-class performance
- Supported natively on most SP routers (IOS-XR, JunOS, Nokia SR OS)

**IP SLA (Cisco):**
- Generates synthetic traffic (ICMP, UDP, TCP, HTTP, jitter probes)
- Measures delay, jitter, loss, HTTP response time
- Results stored in MIBs; polled via SNMP or streamed via telemetry
- Can trigger actions (failover) on threshold violations

**Y.1731 (ITU-T, for Ethernet services):**
- Frame-level performance measurement for Carrier Ethernet
- Loss Measurement (LM), Delay Measurement (DM), Synthetic Loss Measurement (SLM)
- Operates at Layer 2 — no IP dependency
- Essential for MEF-compliant Ethernet service SLAs

### SLA Compliance Reporting

SP SLA compliance is typically measured over a calendar month. Common contract terms:

- **Measurement window:** 5-minute intervals averaged over the month
- **Exclusions:** Scheduled maintenance, customer-caused outages, force majeure
- **Credits:** Prorated monthly charge for SLA violations (e.g., 5% credit per 0.01% below availability target)
- **Measurement point:** Between PE routers (SP responsibility); CE-to-CE adds customer last-mile

The distinction between **measured** and **contractual** SLA is important:

- Measured SLA: Actual performance from probes and monitoring
- Contractual SLA: What the SP guarantees and pays credits for
- The contractual SLA is always looser than the measured capability (margin for safety)

---

## See Also

- mpls, bgp, diffserv, traffic-shaping, qos

## References

- [RFC 2474 — Definition of the Differentiated Services Field](https://www.rfc-editor.org/rfc/rfc2474)
- [RFC 2475 — Architecture for Differentiated Services](https://www.rfc-editor.org/rfc/rfc2475)
- [RFC 2597 — Assured Forwarding PHB Group](https://www.rfc-editor.org/rfc/rfc2597)
- [RFC 3246 — Expedited Forwarding PHB](https://www.rfc-editor.org/rfc/rfc3246)
- [RFC 3270 — MPLS Support of Differentiated Services](https://www.rfc-editor.org/rfc/rfc3270)
- [RFC 2697 — A Single Rate Three Color Marker (srTCM)](https://www.rfc-editor.org/rfc/rfc2697)
- [RFC 2698 — A Two Rate Three Color Marker (trTCM)](https://www.rfc-editor.org/rfc/rfc2698)
- [RFC 5357 — Two-Way Active Measurement Protocol (TWAMP)](https://www.rfc-editor.org/rfc/rfc5357)
- [RFC 3393 — IP Packet Delay Variation Metric](https://www.rfc-editor.org/rfc/rfc3393)
- [ITU-T G.114 — One-Way Transmission Time](https://www.itu.int/rec/T-REC-G.114)
- [ITU-T Y.1731 — OAM Functions and Mechanisms for Ethernet](https://www.itu.int/rec/T-REC-Y.1731)
- [MEF 23.2 — Carrier Ethernet Class of Service](https://www.mef.net/)
