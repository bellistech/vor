# RSVP (Resource Reservation Protocol)

Network signaling protocol (RFC 2205) for reserving bandwidth and QoS resources along a data path. RSVP operates at the transport layer, using PATH and RESV messages to establish per-flow reservations. RSVP-TE (RFC 3209) extends the protocol for MPLS traffic engineering, enabling explicit-route label switched paths with guaranteed bandwidth.

---

## IntServ Model

### Integrated Services and Flowspec

```bash
# RSVP is the signaling protocol for IntServ (RFC 1633):
# - Guaranteed Service (RFC 2212): hard bandwidth/delay bounds
# - Controlled Load (RFC 2211): like an unloaded network
# - Best Effort: no reservation (default)

# Token Bucket model describes traffic shape (TSpec):
#   r = token rate (bytes/sec)        — average rate
#   b = bucket depth (bytes)          — max burst
#   p = peak rate (bytes/sec)         — max instantaneous rate
#   m = min policed unit (bytes)      — smallest counted packet
#   M = max datagram size (bytes)     — largest allowed packet

# RSpec (in RESV message) — QoS requested by receiver:
#   R = reserved bandwidth (bytes/sec)
#   S = slack term (microseconds)     — delay budget flexibility
```

## RSVP Messages

### PATH and RESV

```bash
# PATH — sent downstream (sender -> receiver)
#   Travels the routed data path, installs path state at each hop
#   Carries: Sender TSpec, PHOP (previous hop), Session
#
# RESV — sent upstream (receiver -> sender)
#   Follows reverse PHOP chain, installs reservation state
#   Carries: Flowspec, Filter Spec, Reservation Style
#   Each router runs admission control before installing

# Sender        Router A    Router B    Receiver
#   |--- PATH ---->|---->----->|---->----->|
#   |              |           |           |
#   |<--- RESV ----|<----------|<----------|
#   |====== Reserved Data Path ==========>|
```

### Other Message Types

```bash
# PathTear  — remove path state (downstream, sender-initiated)
# ResvTear  — remove reservation (upstream, receiver-initiated)
# PathErr   — error notification upstream (does NOT modify state)
# ResvErr   — error notification downstream
# ResvConf  — confirm reservation installed (optional)
# Hello     — neighbor discovery / fast failure detection (RFC 3209)

# RSVP runs over raw IP (protocol 46, not TCP/UDP)
# Router Alert IP option forces every hop to process
```

## Reservation Styles

### FF, WF, and SE

```bash
# Fixed Filter (FF) — one reservation per sender
#   Each sender gets dedicated bandwidth
#   3 senders @ 64kbps = 192kbps total reserved
#   Use: point-to-point VoIP, individual video streams

# Wildcard Filter (WF) — shared reservation for ALL senders
#   Single reservation, any sender can use it
#   3 senders share 64kbps = 64kbps total reserved
#   Use: audio conferencing (one speaker at a time)

# Shared Explicit (SE) — shared for LISTED senders only
#   Reservation shared among explicit sender subset
#   Senders A,B share 128kbps; Sender C gets nothing
#   Use: multicast with known sender subset
```

## Soft State and Refresh

### Self-Healing Reservations

```bash
# RSVP uses soft state — reservations expire unless refreshed
# Refresh period: R = 30 seconds (default)
# Cleanup timeout: ~3.5 * R = 105 seconds
# No refresh received -> state deleted automatically

# Advantages: auto-recovery from lost messages, auto-cleanup after failures
# Disadvantage: refresh overhead scales with number of reservations

# Refresh reduction (RFC 2961):
# - Message bundling: combine multiple refreshes
# - Summary refresh: send state IDs, not full objects
# - Reduces overhead by 90%+ in large networks
```

## RSVP-TE for MPLS

### Traffic Engineering Extensions (RFC 3209)

```bash
# RSVP-TE signals MPLS LSPs (Label Switched Paths)
# New objects added to PATH/RESV:
# LABEL_REQUEST (PATH) — request label from downstream
# LABEL (RESV) — assigned label from downstream
# EXPLICIT_ROUTE (ERO, PATH) — specify exact path
# RECORD_ROUTE (RRO) — record path taken (loop detection)
# SESSION_ATTRIBUTE — LSP priority, preemption

# Flow:
# 1. Head-end sends PATH with ERO + LABEL_REQUEST
# 2. Each hop processes ERO, forwards PATH
# 3. Tail-end sends RESV with LABEL upstream
# 4. Each hop installs label binding, forwards RESV
# 5. Head-end receives RESV — LSP established
# 6. Data forwarded using MPLS labels

# ERO subobjects: strict (must be next hop) or loose (route via IGP)
```

## Cisco RSVP-TE Configuration

### TE Tunnel Setup

```bash
# Enable RSVP and MPLS on interfaces
interface GigabitEthernet0/0
 ip address 10.0.12.1 255.255.255.0
 mpls ip
 ip rsvp bandwidth 1000000 1000000    # total / per-flow max (kbps)
 ip rsvp signalling hello

# Create TE tunnel
interface Tunnel0
 ip unnumbered Loopback0
 tunnel mode mpls traffic-eng
 tunnel destination 10.0.0.5
 tunnel mpls traffic-eng bandwidth 500000
 tunnel mpls traffic-eng path-option 1 explicit name PATH-A
 tunnel mpls traffic-eng path-option 2 dynamic            # fallback
 tunnel mpls traffic-eng priority 3 3                      # setup/hold
 tunnel mpls traffic-eng record-route
 tunnel mpls traffic-eng fast-reroute

# Explicit path
ip explicit-path name PATH-A enable
 next-address strict 10.0.12.2
 next-address strict 10.0.23.3
 next-address strict 10.0.35.5

# OSPF TE extensions
router ospf 1
 mpls traffic-eng router-id Loopback0
 mpls traffic-eng area 0

# Verify
show mpls traffic-eng tunnels
show ip rsvp reservation
show ip rsvp interface detail
```

### Fast Reroute (FRR)

```bash
# FRR provides 50ms failover via pre-computed bypass tunnels
# Facility backup — bypass tunnel protects link/node
# One-to-one — dedicated detour per protected LSP

interface Tunnel0
 tunnel mpls traffic-eng fast-reroute

# Bypass tunnel on PLR (Point of Local Repair)
interface Tunnel100
 ip unnumbered Loopback0
 tunnel mode mpls traffic-eng
 tunnel destination 10.0.0.3
 tunnel mpls traffic-eng path-option 1 explicit name BYPASS

interface GigabitEthernet0/0
 mpls traffic-eng backup-path Tunnel100

show mpls traffic-eng fast-reroute database
```

## Juniper RSVP-TE Configuration

### JunOS LSP Setup

```bash
set protocols rsvp interface ge-0/0/0.0 bandwidth 1g
set protocols mpls interface ge-0/0/0.0
set protocols mpls label-switched-path to-PE2 to 10.0.0.2
set protocols mpls label-switched-path to-PE2 bandwidth 500m
set protocols mpls label-switched-path to-PE2 priority 3 3
set protocols mpls label-switched-path to-PE2 fast-reroute

# Explicit path
set protocols mpls path PATH-A 10.0.12.2 strict
set protocols mpls path PATH-A 10.0.23.3 strict
set protocols mpls label-switched-path to-PE2 primary PATH-A

# OSPF TE
set protocols ospf traffic-engineering

# show rsvp session / show mpls lsp extensive
```

## PCEP and DiffServ Comparison

### Path Computation Element (RFC 5440)

```bash
# PCEP delegates path computation to centralized PCE
# PCE has full topology (TED), computes optimal EROs
# Modes: passive (PCC requests), active (PCE updates LSPs), PCE-initiated

# Cisco: mpls traffic-eng pce-address ipv4 10.0.0.99
# Juniper: set protocols pcep pce PCE1 address 10.0.0.99
```

### IntServ (RSVP) vs DiffServ

```bash
# IntServ (RSVP):
# - Per-flow signaling and state at every router
# - Guaranteed bandwidth and delay bounds
# - Scalability: O(N) state where N=flows
# - Best for: critical paths, MPLS TE, small number of premium flows
#
# DiffServ:
# - Per-hop behavior (PHB) based on DSCP marking
# - No per-flow signaling or state
# - Scalability: O(1) — fixed number of traffic classes

# Combined approach (common in production):
# RSVP-TE for MPLS core (tunnel-level, not per-flow)
# DiffServ for edge classification (DSCP marking)
# TE tunnels carry DiffServ traffic classes

# Consider Segment Routing (SR-TE) as modern alternative:
# Eliminates per-flow state via source routing
```

## Troubleshooting

### Verification and Common Issues

```bash
# Cisco IOS
show ip rsvp reservation          show ip rsvp sender
show ip rsvp interface detail     show ip rsvp neighbor
show mpls traffic-eng tunnels     show mpls traffic-eng topology
debug ip rsvp                     debug mpls traffic-eng path

# Juniper: show rsvp session / show rsvp interface / show mpls lsp

# Common issues:
# 1. Insufficient BW — PathErr "admission control failure"; increase ip rsvp bandwidth
# 2. ERO failure — "bad strict node"; verify strict hops are directly connected
# 3. Refresh timeout — LSP flaps every ~105s; check packet loss, enable hello
# 4. Preemption — lower-priority LSP preempted; review setup/hold priorities
# 5. FRR inactive — bypass tunnel down; verify it's up and protecting correct interface
```

---

## Tips

- Enable RSVP Hello for fast neighbor failure detection; without it, cleanup waits 105 seconds.
- Set `ip rsvp bandwidth` on every interface in the TE domain; unconfigured interfaces reject PATH silently.
- Use setup/hold priority carefully: 0 is highest and preempts everything; reserve it for critical paths only.
- Enable record-route on all TE tunnels for troubleshooting; RRO shows the exact LSP path.
- Configure both explicit and dynamic path options: explicit primary for determinism, dynamic fallback for resilience.
- FRR provides 50ms failover but requires pre-computed bypass tunnels to be up before protection is active.
- Use RSVP bandwidth sub-pools to partition between traffic classes (voice vs. data TE tunnels).
- Monitor RSVP counters regularly; high PathErr rate indicates topology or configuration mismatches.
- Combine RSVP-TE with DiffServ: TE tunnels for aggregate flows, DiffServ for per-hop classification.
- Consider Segment Routing TE as a modern alternative that eliminates per-flow network state.

---

## See Also

- mpls, tc, bgp, ospf, qos

## References

- [RFC 2205 — Resource ReSerVation Protocol (RSVP)](https://www.rfc-editor.org/rfc/rfc2205)
- [RFC 2211 — Controlled-Load Network Element Service](https://www.rfc-editor.org/rfc/rfc2211)
- [RFC 2212 — Guaranteed Quality of Service](https://www.rfc-editor.org/rfc/rfc2212)
- [RFC 3209 — RSVP-TE: Extensions to RSVP for LSP Tunnels](https://www.rfc-editor.org/rfc/rfc3209)
- [RFC 4090 — Fast Reroute Extensions to RSVP-TE](https://www.rfc-editor.org/rfc/rfc4090)
- [RFC 5440 — Path Computation Element Communication Protocol (PCEP)](https://www.rfc-editor.org/rfc/rfc5440)
- [RFC 2961 — RSVP Refresh Overhead Reduction](https://www.rfc-editor.org/rfc/rfc2961)
- [Cisco MPLS TE Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/mp_te_path/configuration/xe-16/mp-te-path-xe-16-book.html)
- [Juniper RSVP Configuration Guide](https://www.juniper.net/documentation/us/en/software/junos/mpls/topics/topic-map/rsvp-configuration.html)
