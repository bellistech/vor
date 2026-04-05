# JunOS Class of Service (CoS)

End-to-end QoS framework for traffic prioritization, rate limiting, and congestion management on Juniper devices. CoS classifies, marks, queues, schedules, and polices traffic across the forwarding plane.

## CoS Architecture Overview

```
Ingress                                                    Egress
┌──────────────────────────────────────────────────────────────────────┐
│                                                                      │
│  Classification → Forwarding Class → Queue → Scheduler → Rewrite    │
│  (BA or MF)       Assignment         Assignment  (priority,    (mark │
│                                                   rate,         outgoing
│                                                   buffer,       headers)
│                                                   WRED)
│                                                                      │
│  Policing ─────────────────────────────────────────── Shaping        │
│  (ingress rate limit)                                 (egress rate)  │
└──────────────────────────────────────────────────────────────────────┘
```

## Forwarding Classes and Queues

### Default forwarding classes
```
Forwarding Class              Queue    Typical Use
──────────────────────────────────────────────────
best-effort                   0        Default, bulk data
expedited-forwarding          1        Voice, real-time
assured-forwarding            2        Business-critical
network-control               3        Routing protocols, network mgmt
```

### Define custom forwarding classes
```
set class-of-service forwarding-classes class VOICE queue-num 0 priority high
set class-of-service forwarding-classes class VIDEO queue-num 1 priority medium-high
set class-of-service forwarding-classes class DATA queue-num 2 priority medium-low
set class-of-service forwarding-classes class BEST-EFFORT queue-num 3 priority low
```

### View forwarding classes
```
show class-of-service forwarding-class
```

## BA (Behavior Aggregate) Classification

### DSCP-based classification
```
# Classify incoming packets based on DSCP values in IP header
set class-of-service classifiers dscp DSCP-MAP import default

set class-of-service classifiers dscp DSCP-MAP forwarding-class VOICE loss-priority low code-points ef
set class-of-service classifiers dscp DSCP-MAP forwarding-class VIDEO loss-priority low code-points af41
set class-of-service classifiers dscp DSCP-MAP forwarding-class VIDEO loss-priority high code-points af42
set class-of-service classifiers dscp DSCP-MAP forwarding-class DATA loss-priority low code-points af21
set class-of-service classifiers dscp DSCP-MAP forwarding-class DATA loss-priority high code-points af22
set class-of-service classifiers dscp DSCP-MAP forwarding-class BEST-EFFORT loss-priority low code-points be

# Apply to interface
set class-of-service interfaces ge-0/0/0 unit 0 classifiers dscp DSCP-MAP
```

### IEEE 802.1p classification
```
# Classify based on VLAN PCP (Priority Code Point) bits
set class-of-service classifiers ieee-802.1 DOT1P-MAP forwarding-class VOICE loss-priority low code-points 101
set class-of-service classifiers ieee-802.1 DOT1P-MAP forwarding-class DATA loss-priority low code-points 010
set class-of-service classifiers ieee-802.1 DOT1P-MAP forwarding-class BEST-EFFORT loss-priority low code-points 000

# Apply to interface
set class-of-service interfaces ge-0/0/2 unit 0 classifiers ieee-802.1 DOT1P-MAP
```

### MPLS EXP classification
```
# Classify based on MPLS EXP bits (3 bits = 8 values)
set class-of-service classifiers exp EXP-MAP forwarding-class VOICE loss-priority low code-points 101
set class-of-service classifiers exp EXP-MAP forwarding-class VIDEO loss-priority low code-points 100
set class-of-service classifiers exp EXP-MAP forwarding-class DATA loss-priority low code-points 010
set class-of-service classifiers exp EXP-MAP forwarding-class BEST-EFFORT loss-priority low code-points 000

# Apply to interface
set class-of-service interfaces ge-0/0/1 unit 0 classifiers exp EXP-MAP
```

## MF (Multi-Field) Classification

### Using firewall filters for classification
```
# Classify traffic based on multiple header fields
set firewall family inet filter MF-CLASSIFY term VOICE from protocol udp
set firewall family inet filter MF-CLASSIFY term VOICE from destination-port 5060-5061
set firewall family inet filter MF-CLASSIFY term VOICE then forwarding-class expedited-forwarding
set firewall family inet filter MF-CLASSIFY term VOICE then loss-priority low
set firewall family inet filter MF-CLASSIFY term VOICE then accept

set firewall family inet filter MF-CLASSIFY term VIDEO from protocol udp
set firewall family inet filter MF-CLASSIFY term VIDEO from destination-port 6970-6999
set firewall family inet filter MF-CLASSIFY term VIDEO then forwarding-class assured-forwarding
set firewall family inet filter MF-CLASSIFY term VIDEO then loss-priority low
set firewall family inet filter MF-CLASSIFY term VIDEO then accept

set firewall family inet filter MF-CLASSIFY term SSH from protocol tcp
set firewall family inet filter MF-CLASSIFY term SSH from destination-port 22
set firewall family inet filter MF-CLASSIFY term SSH then forwarding-class network-control
set firewall family inet filter MF-CLASSIFY term SSH then accept

set firewall family inet filter MF-CLASSIFY term DEFAULT then forwarding-class best-effort
set firewall family inet filter MF-CLASSIFY term DEFAULT then accept

# Apply filter to interface
set interfaces ge-0/0/0 unit 0 family inet filter input MF-CLASSIFY
```

## Schedulers

### Scheduler configuration
```
# Define schedulers — control how queues are serviced
set class-of-service schedulers SCHED-VOICE transmit-rate percent 30
set class-of-service schedulers SCHED-VOICE priority strict-high
set class-of-service schedulers SCHED-VOICE buffer-size percent 10
set class-of-service schedulers SCHED-VOICE drop-profile-map loss-priority low protocol any drop-profile DP-LOW

set class-of-service schedulers SCHED-VIDEO transmit-rate percent 30
set class-of-service schedulers SCHED-VIDEO priority high
set class-of-service schedulers SCHED-VIDEO buffer-size percent 25
set class-of-service schedulers SCHED-VIDEO drop-profile-map loss-priority any protocol any drop-profile DP-MEDIUM

set class-of-service schedulers SCHED-DATA transmit-rate percent 25
set class-of-service schedulers SCHED-DATA priority medium-high
set class-of-service schedulers SCHED-DATA buffer-size percent 35

set class-of-service schedulers SCHED-BE transmit-rate remainder
set class-of-service schedulers SCHED-BE priority low
set class-of-service schedulers SCHED-BE buffer-size remainder
set class-of-service schedulers SCHED-BE drop-profile-map loss-priority any protocol any drop-profile DP-AGGRESSIVE
```

### Transmit rate options
```
set class-of-service schedulers SCHED transmit-rate percent 50        # percentage of port speed
set class-of-service schedulers SCHED transmit-rate 100m              # absolute rate
set class-of-service schedulers SCHED transmit-rate remainder         # whatever is left
set class-of-service schedulers SCHED transmit-rate exact percent 30  # strict guarantee, no borrowing
```

### Priority levels
```
# Scheduling priorities (strict-high always serviced first):
#   strict-high  — always serviced first (use sparingly — can starve others)
#   high         — serviced after strict-high
#   medium-high  — middle tier
#   medium-low   — below medium
#   low          — serviced last (best-effort)
```

## WRED (Weighted Random Early Detection)

### Drop profiles
```
# Drop profile defines drop probability curve
set class-of-service drop-profiles DP-LOW fill-level 70 drop-probability 0
set class-of-service drop-profiles DP-LOW fill-level 85 drop-probability 30
set class-of-service drop-profiles DP-LOW fill-level 95 drop-probability 80
set class-of-service drop-profiles DP-LOW fill-level 100 drop-probability 100

set class-of-service drop-profiles DP-MEDIUM fill-level 50 drop-probability 0
set class-of-service drop-profiles DP-MEDIUM fill-level 75 drop-probability 50
set class-of-service drop-profiles DP-MEDIUM fill-level 100 drop-probability 100

set class-of-service drop-profiles DP-AGGRESSIVE fill-level 25 drop-probability 0
set class-of-service drop-profiles DP-AGGRESSIVE fill-level 50 drop-probability 25
set class-of-service drop-profiles DP-AGGRESSIVE fill-level 75 drop-probability 75
set class-of-service drop-profiles DP-AGGRESSIVE fill-level 100 drop-probability 100
```

### Apply drop profile to scheduler
```
set class-of-service schedulers SCHED-DATA drop-profile-map loss-priority low protocol any drop-profile DP-LOW
set class-of-service schedulers SCHED-DATA drop-profile-map loss-priority high protocol any drop-profile DP-AGGRESSIVE
```

## Scheduler Maps

### Map forwarding classes to schedulers
```
set class-of-service scheduler-maps SCHED-MAP forwarding-class VOICE scheduler SCHED-VOICE
set class-of-service scheduler-maps SCHED-MAP forwarding-class VIDEO scheduler SCHED-VIDEO
set class-of-service scheduler-maps SCHED-MAP forwarding-class DATA scheduler SCHED-DATA
set class-of-service scheduler-maps SCHED-MAP forwarding-class BEST-EFFORT scheduler SCHED-BE
```

### Apply scheduler map to interface
```
set class-of-service interfaces ge-0/0/0 scheduler-map SCHED-MAP
set class-of-service interfaces ge-0/0/1 scheduler-map SCHED-MAP

# Apply to all interfaces in a range
set class-of-service interfaces ge-0/0/* scheduler-map SCHED-MAP
```

## Rewrite Rules

### DSCP rewrite
```
# Mark outgoing packets with DSCP values
set class-of-service rewrite-rules dscp DSCP-REWRITE forwarding-class VOICE loss-priority low code-point ef
set class-of-service rewrite-rules dscp DSCP-REWRITE forwarding-class VIDEO loss-priority low code-point af41
set class-of-service rewrite-rules dscp DSCP-REWRITE forwarding-class VIDEO loss-priority high code-point af42
set class-of-service rewrite-rules dscp DSCP-REWRITE forwarding-class DATA loss-priority low code-point af21
set class-of-service rewrite-rules dscp DSCP-REWRITE forwarding-class BEST-EFFORT loss-priority low code-point be

# Apply to interface
set class-of-service interfaces ge-0/0/0 unit 0 rewrite-rules dscp DSCP-REWRITE
```

### IEEE 802.1p rewrite
```
set class-of-service rewrite-rules ieee-802.1 DOT1P-REWRITE forwarding-class VOICE loss-priority low code-point 101
set class-of-service rewrite-rules ieee-802.1 DOT1P-REWRITE forwarding-class DATA loss-priority low code-point 010
set class-of-service rewrite-rules ieee-802.1 DOT1P-REWRITE forwarding-class BEST-EFFORT loss-priority low code-point 000

set class-of-service interfaces ge-0/0/2 unit 0 rewrite-rules ieee-802.1 DOT1P-REWRITE
```

### MPLS EXP rewrite
```
set class-of-service rewrite-rules exp EXP-REWRITE forwarding-class VOICE loss-priority low code-point 101
set class-of-service rewrite-rules exp EXP-REWRITE forwarding-class VIDEO loss-priority low code-point 100
set class-of-service rewrite-rules exp EXP-REWRITE forwarding-class DATA loss-priority low code-point 010
set class-of-service rewrite-rules exp EXP-REWRITE forwarding-class BEST-EFFORT loss-priority low code-point 000

set class-of-service interfaces ge-0/0/1 unit 0 rewrite-rules exp EXP-REWRITE
```

## Traffic Shaping

### Interface shaping rate
```
# Limit total egress throughput on an interface
set class-of-service interfaces ge-0/0/0 shaping-rate 100m
set class-of-service interfaces ge-0/0/0 shaping-rate percent 50

# Per-unit shaping
set class-of-service interfaces ge-0/0/0 unit 0 shaping-rate 50m
```

### Shaping with scheduling
```
# Combine shaping with scheduler map for per-class treatment
set class-of-service interfaces ge-0/0/0 shaping-rate 100m
set class-of-service interfaces ge-0/0/0 scheduler-map SCHED-MAP
# Schedulers now operate within the 100m shaping envelope
```

## Policers

### Single-rate two-color policer
```
set firewall policer POLICER-10M if-exceeding bandwidth-limit 10m
set firewall policer POLICER-10M if-exceeding burst-size-limit 625k
set firewall policer POLICER-10M then discard

# Apply in firewall filter
set firewall family inet filter EDGE term POLICE from source-address 10.0.0.0/8
set firewall family inet filter EDGE term POLICE then policer POLICER-10M
set firewall family inet filter EDGE term POLICE then accept
```

### Two-rate three-color policer
```
set firewall three-color-policer TC-POLICER two-rate-three-color
set firewall three-color-policer TC-POLICER two-rate-three-color committed-information-rate 10m
set firewall three-color-policer TC-POLICER two-rate-three-color committed-burst-size 100k
set firewall three-color-policer TC-POLICER two-rate-three-color peak-information-rate 20m
set firewall three-color-policer TC-POLICER two-rate-three-color peak-burst-size 200k

# Apply in filter
set firewall family inet filter EDGE term SHAPED then three-color-policer two-rate TC-POLICER
```

### Single-rate three-color policer
```
set firewall three-color-policer SR-POLICER single-rate
set firewall three-color-policer SR-POLICER single-rate committed-information-rate 5m
set firewall three-color-policer SR-POLICER single-rate committed-burst-size 50k
set firewall three-color-policer SR-POLICER single-rate excess-burst-size 100k

set firewall family inet filter EDGE term SR-SHAPED then three-color-policer single-rate SR-POLICER
```

### Hierarchical policer
```
# Parent policer limits aggregate; child policers subdivide
set firewall hierarchical-policer HP-AGGREGATE logical-interface-policer if-exceeding bandwidth-limit 100m
set firewall hierarchical-policer HP-AGGREGATE logical-interface-policer if-exceeding burst-size-limit 1m
set firewall hierarchical-policer HP-AGGREGATE logical-interface-policer then discard

set firewall hierarchical-policer HP-AGGREGATE premium if-exceeding bandwidth-limit 30m
set firewall hierarchical-policer HP-AGGREGATE premium if-exceeding burst-size-limit 300k
set firewall hierarchical-policer HP-AGGREGATE premium then discard

# Apply to logical interface
set interfaces ge-0/0/0 unit 0 layer2-policer input-hierarchical-policer HP-AGGREGATE
```

### Tri-color marking actions
```
# Map policer color to loss-priority and forwarding-class
# GREEN  → loss-priority low   (in-profile)
# YELLOW → loss-priority medium-high (exceed)
# RED    → loss-priority high  (violate / discard)

set firewall three-color-policer TC-POLICER action loss-priority high then discard
```

## CoS for MPLS (EXP Bits)

### MPLS EXP classification and rewrite
```
# Ingress PE: classify IP DSCP → set MPLS EXP
set class-of-service classifiers dscp INGRESS-DSCP forwarding-class VOICE loss-priority low code-points ef
set class-of-service interfaces ge-0/0/0 unit 0 classifiers dscp INGRESS-DSCP

# Core P: classify EXP → forwarding class
set class-of-service classifiers exp CORE-EXP forwarding-class VOICE loss-priority low code-points 101
set class-of-service interfaces ge-0/0/1 unit 0 classifiers exp CORE-EXP

# Egress PE: rewrite EXP → DSCP on decapsulation
set class-of-service rewrite-rules dscp EGRESS-DSCP forwarding-class VOICE loss-priority low code-point ef
set class-of-service interfaces ge-0/0/2 unit 0 rewrite-rules dscp EGRESS-DSCP

# P router: rewrite EXP on label swap
set class-of-service rewrite-rules exp CORE-EXP-REWRITE forwarding-class VOICE loss-priority low code-point 101
set class-of-service interfaces ge-0/0/1 unit 0 rewrite-rules exp CORE-EXP-REWRITE
```

### EXP-to-DSCP mapping for L3VPN
```
# On ingress PE, map customer DSCP to internal forwarding class
# On egress PE, map forwarding class back to customer DSCP
# EXP bits carry CoS across MPLS core
#
# Customer → [DSCP classify] → PE ingress → [EXP rewrite] → Core → [EXP classify] → PE egress → [DSCP rewrite] → Customer
```

## Interface CoS Application Summary

```
set class-of-service interfaces ge-0/0/0 scheduler-map SCHED-MAP          # scheduler map
set class-of-service interfaces ge-0/0/0 shaping-rate 100m                 # shaping rate
set class-of-service interfaces ge-0/0/0 unit 0 classifiers dscp DSCP-MAP  # BA classifier
set class-of-service interfaces ge-0/0/0 unit 0 rewrite-rules dscp REWRITE # rewrite rules
set class-of-service interfaces ge-0/0/0 unit 0 classifiers exp EXP-MAP    # MPLS classifier
set class-of-service interfaces ge-0/0/0 unit 0 rewrite-rules exp EXP-RW   # MPLS rewrite
```

## Verification Commands

### Show CoS configuration
```
show class-of-service forwarding-class              # forwarding class assignments
show class-of-service classifier type dscp           # DSCP classifiers
show class-of-service classifier type exp            # EXP classifiers
show class-of-service classifier type ieee-802.1     # 802.1p classifiers
show class-of-service scheduler-map                  # scheduler map bindings
show class-of-service rewrite-rule type dscp         # DSCP rewrite rules
show class-of-service rewrite-rule type exp          # EXP rewrite rules
show class-of-service drop-profile                   # WRED drop profiles
```

### Show interface CoS state
```
show class-of-service interface ge-0/0/0             # full CoS state on interface
show class-of-service interface ge-0/0/0 comprehensive  # detailed queue stats
show interfaces queue ge-0/0/0                       # per-queue counters
show interfaces ge-0/0/0 detail | match "CoS|queue"  # quick CoS check
```

### Show policer stats
```
show policer                                         # all policer hit counts
show firewall filter EDGE                            # filter counters (policer refs)
show firewall                                        # all filter statistics
```

### Show queue statistics
```
show interfaces queue ge-0/0/0                       # queued/transmitted/dropped per queue
show class-of-service interface ge-0/0/0 comprehensive  # scheduler/WRED stats
```

## Tips

- Strict-high priority can starve lower queues — use only for voice/control with low volume
- Always assign WRED drop profiles to data queues to enable TCP-friendly congestion avoidance
- Transmit-rate percent values across all schedulers do not need to sum to 100 — but should not exceed 100
- Use `remainder` for best-effort to absorb unused bandwidth
- BA classification is simpler and faster; MF classification is more granular but uses TCAM
- In MPLS networks, EXP bits carry CoS across the core — ensure consistent EXP maps on all P/PE routers
- Test CoS with `monitor traffic` or `show interfaces queue` counters before production deployment
- Shaping rate creates a soft ceiling; policer creates a hard ceiling with drop/mark

## See Also

- junos-firewall-filters, junos-routing-policy, junos-interfaces, mpls, dscp

## References

- [Juniper TechLibrary — CoS Overview](https://www.juniper.net/documentation/us/en/software/junos/cos/topics/concept/cos-overview.html)
- [Juniper TechLibrary — Schedulers](https://www.juniper.net/documentation/us/en/software/junos/cos/topics/concept/cos-schedulers-overview.html)
- [Juniper TechLibrary — Policers](https://www.juniper.net/documentation/us/en/software/junos/cos/topics/concept/policer-overview.html)
- [Juniper TechLibrary — WRED Drop Profiles](https://www.juniper.net/documentation/us/en/software/junos/cos/topics/concept/cos-red-drop-profiles-overview.html)
- [Juniper TechLibrary — CoS for MPLS](https://www.juniper.net/documentation/us/en/software/junos/cos/topics/concept/cos-mpls-overview.html)
- [RFC 2474 — Definition of the Differentiated Services Field](https://www.rfc-editor.org/rfc/rfc2474)
- [RFC 2697 — A Single Rate Three Color Marker](https://www.rfc-editor.org/rfc/rfc2697)
- [RFC 2698 — A Two Rate Three Color Marker](https://www.rfc-editor.org/rfc/rfc2698)
- [RFC 3270 — MPLS Support of Differentiated Services](https://www.rfc-editor.org/rfc/rfc3270)
