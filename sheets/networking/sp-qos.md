# Service Provider QoS

Quality of Service architecture for service provider networks — DiffServ domains, hierarchical QoS (H-QoS) for multi-tenant services, MPLS QoS models, and traffic contracts.

## Concepts

### SP QoS Architecture Layers

- **Access:** Customer-facing edge — classification, policing, ingress marking. Trust boundary is here
- **Aggregation:** Traffic from many customers aggregated — H-QoS shaping per customer, queuing per service class
- **Core:** High-speed forwarding — simple DSCP/EXP-based queuing, no per-flow state, minimal policy

### DiffServ Domain

- A DiffServ domain is a contiguous set of nodes that use the same PHB (Per-Hop Behavior) definitions
- At domain boundaries, DSCP values are re-marked to match the local domain's policy
- SP network = one DiffServ domain; customer networks = separate domains
- Trust boundary: SP edge classifies and re-marks customer traffic at ingress

### Traffic Contracts

- **CIR (Committed Information Rate):** Guaranteed bandwidth — traffic up to CIR is green (conforming)
- **EIR (Excess Information Rate):** Bandwidth above CIR that may be carried if capacity exists — yellow (exceeding)
- **CBS (Committed Burst Size):** Maximum burst of conforming traffic (token bucket depth for CIR)
- **EBS (Excess Burst Size):** Maximum burst of exceeding traffic (token bucket depth for EIR)
- **PIR (Peak Information Rate):** Hard cap — traffic above PIR is red (violating) and dropped

### Color-Aware Policing (trTCM — RFC 4115)

```
                    ┌─────────────┐
  Ingress Traffic ──► Two-Rate    ├──► Green  (≤ CIR)  → Forward
                    │ Three-Color │
                    │ Marker      ├──► Yellow (CIR < x ≤ PIR) → Forward if capacity
                    │             │
                    └─────────────┘──► Red    (> PIR)  → Drop
```

- **Color-blind mode:** Ignores pre-existing markings; re-classifies everything
- **Color-aware mode:** Respects upstream markings; green packets can be re-colored yellow/red but never promoted

## H-QoS (Hierarchical QoS)

### Architecture

```
  Customer A (10 Gbps contract)
  ├── Voice     — strict priority, CIR 500 Mbps
  ├── Video     — WFQ weight 40%, CIR 4 Gbps
  ├── Business  — WFQ weight 30%, CIR 3 Gbps
  └── Best Effort — WFQ weight 30%, no CIR

  Customer B (5 Gbps contract)
  ├── Voice     — strict priority, CIR 200 Mbps
  ├── Business  — WFQ weight 50%, CIR 2.5 Gbps
  └── Best Effort — WFQ weight 50%, no CIR
```

- **Level 1 (outer):** Per-customer shaper — enforces the customer's aggregate rate (CIR/PIR)
- **Level 2 (inner):** Per-service-class queuing within each customer — priority queue for voice, weighted fair queuing for data classes
- Ensures one customer cannot starve another even when sharing a physical interface

### IOS-XR H-QoS Configuration

```
! Step 1: Define inner policy (per-service-class queuing)
policy-map CUSTOMER-CHILD
 class VOICE
  priority level 1
  police rate 500 mbps
 !
 class VIDEO
  bandwidth remaining percent 40
  random-detect dscp-based
 !
 class BUSINESS
  bandwidth remaining percent 30
  random-detect dscp-based
 !
 class class-default
  bandwidth remaining percent 30
  random-detect
 !
end-policy-map

! Step 2: Define outer policy (per-customer shaper)
policy-map CUSTOMER-A-PARENT
 class class-default
  shape average 10 gbps
  service-policy CUSTOMER-CHILD
 !
end-policy-map

! Step 3: Apply to interface (or sub-interface for per-VLAN)
interface TenGigE0/0/0/1.100
 service-policy output CUSTOMER-A-PARENT
!
```

### Class Maps for Classification

```
! IOS-XR class-map definitions
class-map match-any VOICE
 match dscp ef
 match dscp cs5
end-class-map

class-map match-any VIDEO
 match dscp af41
 match dscp af42
 match dscp af43
end-class-map

class-map match-any BUSINESS
 match dscp af31
 match dscp af32
 match dscp af33
 match dscp cs3
end-class-map

! Network control traffic
class-map match-any NETWORK-CONTROL
 match dscp cs6
 match dscp cs7
end-class-map
```

## MPLS QoS

### EXP Bits (Traffic Class Field)

- MPLS header has a 3-bit TC field (formerly EXP) — carries QoS marking through the MPLS domain
- 3 bits = 8 values (0-7); must map DSCP (64 values) to EXP at ingress, and EXP back to DSCP at egress
- Typical mapping:

| Traffic Class | DSCP | EXP |
|:---|:---:|:---:|
| Voice (EF) | 46 | 5 |
| Video (AF41) | 34 | 4 |
| Business Critical (AF31) | 26 | 3 |
| Transactional (AF21) | 18 | 2 |
| Best Effort (BE) | 0 | 0 |
| Network Control (CS6) | 48 | 6 |
| Scavenger (CS1) | 8 | 1 |

### MPLS QoS Models

**Uniform Mode:**
- DSCP and EXP are always synchronized
- At push: EXP copied from DSCP
- At swap: EXP preserved
- At pop: EXP copied back to DSCP
- Use case: Single SP domain where DSCP and EXP represent the same PHB

**Pipe Mode:**
- DSCP and EXP are independent after push
- At push: EXP set from DSCP (or explicit policy)
- At swap: EXP used for queuing in core
- At pop: Original DSCP restored from inner header (EXP is discarded)
- Use case: SP carries customer traffic — customer DSCP preserved end-to-end, SP uses its own EXP scheme in core

**Short-Pipe Mode:**
- Like pipe, but egress PE uses DSCP (not EXP) for queuing on the egress interface
- At push: EXP set from DSCP
- At swap: EXP used for core queuing
- At penultimate hop (PHP): Label popped, exposing DSCP
- At egress PE: DSCP from IP header used for egress queuing
- Use case: SP wants customer's DSCP to drive egress queuing toward CE

### IOS-XR MPLS QoS Configuration

```
! Pipe mode — set EXP at ingress PE, preserve customer DSCP
policy-map MPLS-INGRESS
 class VOICE
  set mpls experimental imposition 5
 !
 class VIDEO
  set mpls experimental imposition 4
 !
 class BUSINESS
  set mpls experimental imposition 3
 !
 class class-default
  set mpls experimental imposition 0
 !
end-policy-map

! Core queuing based on EXP
policy-map CORE-EGRESS
 class MPLS-EXP-5
  priority level 1
  police rate percent 10
 !
 class MPLS-EXP-4
  bandwidth remaining percent 30
 !
 class MPLS-EXP-3
  bandwidth remaining percent 30
 !
 class class-default
  bandwidth remaining percent 40
 !
end-policy-map

! Core class-maps matching EXP
class-map match-any MPLS-EXP-5
 match mpls experimental topmost 5
end-class-map

class-map match-any MPLS-EXP-4
 match mpls experimental topmost 4
end-class-map

class-map match-any MPLS-EXP-3
 match mpls experimental topmost 3
end-class-map
```

## Ingress Classification and Re-Marking

### Trust Boundary at SP Edge

```
! PE ingress — classify customer traffic and re-mark to SP DSCP scheme
policy-map PE-INGRESS
 class CUSTOMER-VOICE
  ! Customer marks EF — trust it if SLA allows
  set dscp ef
  police rate 500 mbps burst 62500 bytes
   conform-action transmit
   exceed-action drop
 !
 class CUSTOMER-VIDEO
  set dscp af41
  police rate 2 gbps burst 250000 bytes
   conform-action transmit
   exceed-action set dscp af42
   ! Exceeding video gets AF42 (higher drop precedence)
 !
 class CUSTOMER-BUSINESS
  set dscp af31
  police rate 1 gbps burst 125000 bytes
   conform-action transmit
   exceed-action set dscp af32
 !
 class class-default
  set dscp default
  police rate 5 gbps
   conform-action transmit
   exceed-action drop
 !
end-policy-map

interface GigabitEthernet0/0/0/0
 service-policy input PE-INGRESS
!
```

## QoS Policy Propagation with BGP (QPPB)

```
! QPPB — classify traffic based on BGP attributes (community, AS-path, prefix)
! Useful for peering traffic engineering and inter-AS QoS

! Step 1: Tag routes with a QoS group via route-policy
route-policy QPPB-CLASSIFY
 if community matches-any PREMIUM-PEERS then
  set qos-group 1
 elseif community matches-any STANDARD-PEERS then
  set qos-group 2
 else
  set qos-group 0
 endif
end-policy

! Step 2: Apply to BGP
router bgp 65001
 address-family ipv4 unicast
  table-policy QPPB-CLASSIFY
 !

! Step 3: Enable QPPB on the interface
interface GigabitEthernet0/0/0/0
 ipv4 bgp policy propagation input qos-group destination
!

! Step 4: Match qos-group in QoS policy
class-map match-any PREMIUM-TRAFFIC
 match qos-group 1
end-class-map
```

## QoS for Specific SP Services

### Mobile Backhaul QoS

```
! Mobile backhaul — map 3GPP QCI to DSCP
! QCI 1 (Conversational Voice) → EF
! QCI 2 (Conversational Video) → AF41
! QCI 5 (IMS Signaling) → CS5 / EF
! QCI 6 (Video/TCP) → AF31
! QCI 7 (Voice/Video Live) → AF21
! QCI 8 (Video/TCP Premium) → AF11
! QCI 9 (Default Bearer) → BE

policy-map MOBILE-BACKHAUL
 class QCI-1
  priority level 1
  police rate percent 10
 !
 class QCI-5
  priority level 2
  police rate percent 5
 !
 class QCI-2
  bandwidth remaining percent 25
  random-detect dscp-based
 !
 class QCI-6-7
  bandwidth remaining percent 25
  random-detect dscp-based
 !
 class class-default
  bandwidth remaining percent 45
  random-detect
 !
end-policy-map
```

### Business VPN QoS (L3VPN)

```
! Per-VRF QoS — apply H-QoS per customer VPN
policy-map VPN-CUSTOMER-GOLD
 class class-default
  shape average 1 gbps
  service-policy VPN-CHILD-GOLD
 !
end-policy-map

policy-map VPN-CHILD-GOLD
 class VOICE
  priority level 1
  police rate 100 mbps
 !
 class REALTIME-VIDEO
  bandwidth remaining percent 30
 !
 class TRANSACTIONAL
  bandwidth remaining percent 30
 !
 class class-default
  bandwidth remaining percent 40
  random-detect dscp-based
 !
end-policy-map

! Apply per sub-interface (one sub-if per VPN customer)
interface GigabitEthernet0/0/0/1.200
 vrf CUSTOMER-GOLD
 service-policy output VPN-CUSTOMER-GOLD
!
```

## Egress Queuing and Scheduling

### Scheduling Disciplines

- **Strict Priority (SP/LLQ):** Served first — always. Voice and real-time traffic. Must police to prevent starvation
- **Weighted Fair Queuing (WFQ):** Bandwidth shared proportionally by weight among non-priority classes
- **Deficit Weighted Round Robin (DWRR):** Round-robin with deficit counter — fairer for variable packet sizes
- **Shaping:** Delays packets to smooth output rate to a configured maximum (token bucket with buffer)

### WRED Configuration (SP Context)

```
! Weighted Random Early Detection — drop packets probabilistically before queue fills
! SP uses DSCP-based WRED to differentiate drop precedence within AF classes

policy-map EGRESS-QUEUING
 class AF3-CLASS
  bandwidth remaining percent 25
  random-detect dscp-based
  ! AF31 — low drop: begin dropping at 50% queue depth, 100% at 80%
  random-detect dscp af31 500 packets 800 packets
  ! AF32 — medium drop: begin at 40%, 100% at 70%
  random-detect dscp af32 400 packets 700 packets
  ! AF33 — high drop: begin at 30%, 100% at 60%
  random-detect dscp af33 300 packets 600 packets
 !
end-policy-map
```

## Operational Commands

```bash
# IOS-XR — show QoS policy applied to interface
show policy-map interface TenGigE0/0/0/1 output

# Show per-class statistics (packets/bytes classified, queued, dropped)
show policy-map interface TenGigE0/0/0/1.100 output detail

# Verify H-QoS parent/child relationship
show policy-map interface TenGigE0/0/0/1.100 output detail | include "Class|Shape|Priority|Band"

# Show WRED drop statistics
show policy-map interface TenGigE0/0/0/1 output detail | include "random"

# Check MPLS EXP marking
show policy-map interface TenGigE0/0/0/1 input detail | include "EXP|set"

# Platform QoS resource usage (NCS 5500 / ASR 9000)
show controllers npu resources qos all

# Verify QPPB operational state
show bgp ipv4 unicast policy
show cef ipv4 203.0.113.0/24 detail | include qos
```

## Best Practices

- Always police voice/priority traffic at ingress — strict priority without policing leads to starvation of all other classes during overload.
- Use H-QoS on aggregation and PE nodes; flat QoS in the core where per-customer state is unnecessary.
- Keep the number of traffic classes to 4-6 in the core; more classes increase memory and processing overhead with diminishing returns.
- Map DSCP to EXP at the ingress PE and do not re-examine IP headers in the core — MPLS forwarding should be based solely on EXP.
- Use pipe mode for L3VPN services to preserve customer DSCP end-to-end; uniform mode conflates SP and customer markings.
- Color-aware policing at ingress ensures upstream marking is respected; color-blind policing ignores customer intent.
- Tune WRED thresholds based on actual traffic profiles — default thresholds are rarely optimal for SP workloads.
- Monitor per-class drop counters continuously; drops in priority queue indicate policing mismatch or CIR over-subscription.
- For mobile backhaul, align QCI-to-DSCP mapping with the mobile operator's requirements — there is no universal standard mapping.
- Document the DSCP/EXP mapping table and distribute to all customer-facing teams — QoS troubleshooting requires knowing the mapping.
- Test H-QoS under congestion before production deployment — shaping and scheduling behaviors are non-obvious at edge cases.

## See Also

- mpls, bgp, diffserv, traffic-shaping, qos

## References

- [RFC 2474 — Definition of the Differentiated Services Field](https://www.rfc-editor.org/rfc/rfc2474)
- [RFC 2475 — Architecture for Differentiated Services](https://www.rfc-editor.org/rfc/rfc2475)
- [RFC 2597 — Assured Forwarding PHB Group](https://www.rfc-editor.org/rfc/rfc2597)
- [RFC 3246 — Expedited Forwarding PHB](https://www.rfc-editor.org/rfc/rfc3246)
- [RFC 3270 — MPLS Support of Differentiated Services](https://www.rfc-editor.org/rfc/rfc3270)
- [RFC 4115 — A Differentiated Service Two-Rate, Three-Color Marker](https://www.rfc-editor.org/rfc/rfc4115)
- [RFC 2698 — A Two Rate Three Color Marker (trTCM)](https://www.rfc-editor.org/rfc/rfc2698)
- [Cisco IOS-XR QoS Configuration Guide](https://www.cisco.com/c/en/us/td/docs/routers/asr9000/software/qos/configuration/guide/b-qos-cg-asr9000.html)
- [Juniper JUNOS QoS Configuration Guide](https://www.juniper.net/documentation/us/en/software/junos/cos/index.html)
- [MEF Forum — Carrier Ethernet QoS](https://www.mef.net/)
