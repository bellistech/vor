# JunOS IS-IS Advanced

Advanced IS-IS configuration for SP networks — wide metrics, multi-topology, TE extensions, SR extensions, convergence tuning, route leaking, mesh groups, and operational debugging on JunOS.

## IS-IS Basic Configuration

### Enable IS-IS
```
# Interface configuration
set protocols isis interface ge-0/0/0.0 point-to-point
set protocols isis interface ge-0/0/1.0 point-to-point
set protocols isis interface lo0.0 passive

# System ID (NET — Network Entity Title)
set protocols isis iso 49.0001.0100.0000.0001.00
#                   │      │                  │
#                   area   system-id          SEL (00 = router)

# Level configuration
set protocols isis level 1 disable                    # L2-only (typical SP)
set protocols isis level 2 wide-metrics-only          # wide metrics only
```

### Interface-level configuration
```
set protocols isis interface ge-0/0/0.0 level 2 metric 100
set protocols isis interface ge-0/0/0.0 level 2 hello-interval 5
set protocols isis interface ge-0/0/0.0 level 2 hold-time 15
set protocols isis interface ge-0/0/0.0 point-to-point                # P2P (skip DIS election)
set protocols isis interface ge-0/0/1.0 level 2 passive               # advertise prefix, no hello

# Authentication per interface
set protocols isis interface ge-0/0/0.0 level 2 hello-authentication-key "secret"
set protocols isis interface ge-0/0/0.0 level 2 hello-authentication-type md5
```

### Area-level authentication
```
# LSP authentication (all LSPs in the level must carry valid auth)
set protocols isis level 2 authentication-key "AreaSecret"
set protocols isis level 2 authentication-type md5

# SNP authentication
set protocols isis level 2 no-csnp-authentication false
set protocols isis level 2 no-psnp-authentication false
```

## Wide Metrics

### Enable wide metrics
```
# Narrow metrics: 6-bit (interface max 63, path max 1023)
# Wide metrics: 24-bit (interface max 16777215, path max 4261412864)
# Wide metrics required for TE and SR

set protocols isis level 2 wide-metrics-only

# Or allow both during migration
set protocols isis level 2 wide-metrics-only false
# (advertises both narrow and wide TLVs)
```

### Set wide metric values
```
set protocols isis interface ge-0/0/0.0 level 2 metric 1000
set protocols isis interface ge-0/0/1.0 level 2 metric 500
set protocols isis interface ge-0/0/2.0 level 2 metric 10000

# Reference bandwidth approach (like OSPF auto-cost)
# JunOS does not have auto-cost for IS-IS — set manually
# Convention: metric = reference_bw / interface_bw
#   10G link: 10000/10 = 1000
#   1G link:  10000/1 = 10000
#   100G link: 10000/100 = 100
```

## Multi-Topology IS-IS

### Enable multi-topology
```
# MT allows IPv4 and IPv6 to have different topologies
# Without MT: IPv4 and IPv6 share the same SPF tree
# With MT: separate SPF computation per topology

set protocols isis topologies ipv4-unicast
set protocols isis topologies ipv6-unicast

# Per-interface topology participation
set protocols isis interface ge-0/0/0.0 level 2 ipv4-unicast metric 100
set protocols isis interface ge-0/0/0.0 level 2 ipv6-unicast metric 200

# Interface only in IPv6 topology
set protocols isis interface ge-0/0/2.0 level 2 ipv4-unicast disable
set protocols isis interface ge-0/0/2.0 level 2 ipv6-unicast metric 100
```

### Verify multi-topology
```
show isis overview | match "Topology"            # enabled topologies
show isis database detail | match "Topology|MT"  # MT TLVs in LSDB
show isis spf log | match "topology"             # per-topology SPF runs
```

## IS-IS for IPv6

### Single-topology IPv6
```
# IPv6 in single topology (all interfaces must support both v4 and v6)
set protocols isis interface ge-0/0/0.0 family inet6
set protocols isis interface lo0.0 family inet6

# Shortcut: enable at global level
set protocols isis family inet6
```

### Multi-topology IPv6
```
# Preferred: use multi-topology for independent IPv4/IPv6 topologies
set protocols isis topologies ipv6-unicast
set protocols isis interface ge-0/0/0.0 level 2 ipv6-unicast metric 100
```

## IS-IS TE Extensions

### Traffic engineering TLVs
```
# Enable TE extensions for RSVP-TE and SR-TE
set protocols isis traffic-engineering

# Interface TE attributes
set protocols isis interface ge-0/0/0.0 level 2 te-metric 100
set protocols isis interface ge-0/0/0.0 level 2 admin-group RED
set protocols isis interface ge-0/0/0.0 level 2 admin-group BLUE

# Admin group definitions
set protocols mpls admin-groups RED 0
set protocols mpls admin-groups BLUE 1
set protocols mpls admin-groups GREEN 2
```

### TE metric vs IGP metric
```
# IGP metric: used for SPF shortest path computation
# TE metric: used by CSPF (Constrained SPF) for RSVP-TE / SR-TE path computation
# Both carried in separate TLVs
# TE metric allows different cost model for traffic engineering

set protocols isis interface ge-0/0/0.0 level 2 metric 100        # IGP metric
set protocols isis interface ge-0/0/0.0 level 2 te-metric 500     # TE metric
```

### Verify TE
```
show isis database detail | match "TE|admin-group|metric"
show ted database                                # traffic engineering database
show ted database extensive                      # full TE info
```

## IS-IS SR Extensions

### Enable SR on IS-IS
```
set protocols isis source-packet-routing
set protocols isis source-packet-routing node-segment ipv4-index 1
set protocols isis source-packet-routing srgb start-label 16000 index-range 8000

# Per-interface adjacency-SID
set protocols isis interface ge-0/0/0.0 level 2 ipv4-adjacency-segment protected label 100001

# Verify
show isis database detail | match "SID|SRGB"
show isis adjacency detail | match "SID"
```

## IS-IS BFD

### Enable BFD on IS-IS interfaces
```
set protocols isis interface ge-0/0/0.0 bfd-liveness-detection minimum-interval 300
set protocols isis interface ge-0/0/0.0 bfd-liveness-detection multiplier 3

# Per-level BFD
set protocols isis interface ge-0/0/0.0 level 2 bfd-liveness-detection minimum-interval 300
set protocols isis interface ge-0/0/0.0 level 2 bfd-liveness-detection multiplier 3

# Verify
show bfd session                                 # BFD sessions
show isis adjacency detail | match "BFD"         # BFD state per adjacency
```

## IS-IS Overload

### Set overload bit
```
# Overload bit (OL): tells other routers to not use this router for transit
# Traffic destined TO this router still works
# Traffic THROUGH this router is avoided (if alternate path exists)

# Permanent overload
set protocols isis overload

# Overload on startup (give time for routing tables to converge)
set protocols isis overload timeout 300    # 300 seconds after boot

# Advertise overload for TE only
set protocols isis overload advertise-high-metrics

# Manual operational overload (without config change)
# From operational mode:
set protocols isis overload    # (in configure mode, then commit)
```

### Verify overload
```
show isis overview | match "Overload"
show isis database | match "Overload"
show isis database detail <systemid> | match "Overload"
```

## Level Route Leaking

### L2 to L1 route leaking
```
# By default, L1 routers install a default route toward the nearest L1/L2 router
# Route leaking injects specific L2 prefixes into L1

# Leak policy
set policy-options policy-statement LEAK-L2-TO-L1 term SPECIFIC from route-filter 10.0.0.0/16 orlonger
set policy-options policy-statement LEAK-L2-TO-L1 term SPECIFIC then accept

# Apply leak policy
set protocols isis level 1 import-policy LEAK-L2-TO-L1
```

### L1 to L2 route leaking
```
# By default, L1/L2 routers redistribute L1 routes into L2
# Control what gets leaked with policy

set policy-options policy-statement LEAK-L1-TO-L2 term CUSTOMER-ROUTES from route-filter 172.16.0.0/12 orlonger
set policy-options policy-statement LEAK-L1-TO-L2 term CUSTOMER-ROUTES then accept
set policy-options policy-statement LEAK-L1-TO-L2 term BLOCK-ALL then reject

set protocols isis level 2 import-policy LEAK-L1-TO-L2
```

### Verify route leaking
```
show isis database level 1 detail | match "IP prefix"   # L1 prefixes
show isis database level 2 detail | match "IP prefix"   # L2 prefixes
show route protocol isis                                  # IS-IS routes in RIB
```

## Mesh Groups

### Configure mesh groups
```
# Mesh groups reduce LSP flooding on fully-meshed segments
# Members of a mesh group do NOT flood LSPs received from other members
# of the same mesh group (they assume all members already have it)

set protocols isis interface ge-0/0/0.0 mesh-group 1
set protocols isis interface ge-0/0/1.0 mesh-group 1
set protocols isis interface ge-0/0/2.0 mesh-group 1

# Block flooding entirely on an interface
set protocols isis interface ge-0/0/3.0 mesh-group blocked
```

### When to use mesh groups
```
# Use case: NBMA networks (Frame Relay, ATM) with full mesh
# Without mesh groups: an LSP received on one interface is flooded to ALL others
# With mesh groups: LSP flooded only to interfaces NOT in the same mesh group
# Reduces O(N^2) flooding to O(N)
#
# WARNING: mesh groups can cause LSP propagation failures if misconfigured
# Only use on truly full-mesh segments where all peers have direct connectivity
```

## IS-IS Graceful Restart

### Configure graceful restart
```
set protocols isis graceful-restart

# Helper mode (assist neighbors during their restart)
set protocols isis graceful-restart helper-disable false

# Restart duration
set protocols isis graceful-restart restart-duration 300
```

### Verify graceful restart
```
show isis overview | match "Restart"
show isis adjacency detail | match "Restart"
```

## Purge Originator Identification

### Enable POI
```
# RFC 6232: adds originator identification to LSP purges
# Helps identify which router purged an LSP (debugging aid)
set protocols isis purge-originator-identification
```

## IS-IS Convergence Tuning

### SPF throttling
```
# Control SPF computation frequency
set protocols isis spf-options delay 200              # initial delay (ms)
set protocols isis spf-options holddown 5000          # min time between SPFs (ms)
set protocols isis spf-options rapid-runs 3           # number of rapid SPFs before holddown

# PRC (Partial Route Computation) — recompute only affected routes
set protocols isis overload-bit-advertise false
```

### LSP generation throttling
```
# Control how fast this router generates new LSPs
set protocols isis lsp-lifetime 65535                 # max LSP lifetime (seconds)
set protocols isis lsp-refresh-interval 30000         # refresh before expiry (seconds)
```

### Exponential backoff timers
```
# IS-IS hello interval and hold time
set protocols isis interface ge-0/0/0.0 level 2 hello-interval 1       # 1 second (fast hellos)
set protocols isis interface ge-0/0/0.0 level 2 hold-time 3            # 3 seconds (3x hello)

# Combined with BFD for sub-second failure detection
set protocols isis interface ge-0/0/0.0 bfd-liveness-detection minimum-interval 100
set protocols isis interface ge-0/0/0.0 bfd-liveness-detection multiplier 3
# BFD detection: 300ms, IS-IS hold-time: 3s (BFD triggers faster)
```

## IS-IS Topology Debugging

### Debugging commands
```
show isis adjacency                                   # all adjacencies
show isis adjacency detail                            # detailed adjacency info
show isis adjacency ge-0/0/0.0                        # specific interface adjacency

show isis database                                    # LSDB summary (all nodes)
show isis database detail                             # full LSDB with all TLVs
show isis database <systemid>                         # specific node's LSP
show isis database extensive                          # maximum detail

show isis spf log                                     # SPF computation history
show isis spf results                                 # current SPF tree
show isis spf results detail                          # per-prefix SPF result
show isis statistics                                  # protocol counters

show isis interface                                   # IS-IS enabled interfaces
show isis interface detail                            # interface-level parameters
show isis overview                                    # global IS-IS configuration

show isis route                                       # IS-IS routing table
show isis route detail                                # detailed route info
show isis hostname                                    # system-id to hostname mapping
```

### Troubleshooting adjacency issues
```
# Common adjacency problems:
# 1. MTU mismatch: IS-IS pads hello to interface MTU — check both sides
show isis adjacency detail | match "MTU"
show interfaces ge-0/0/0 | match "MTU"

# 2. Area mismatch: L1 adjacency requires same area ID
show isis adjacency detail | match "Area"

# 3. Authentication mismatch: hello or LSP auth must match
show isis statistics | match "auth"

# 4. Level mismatch: both sides must share at least one level
show isis interface detail | match "Level"

# 5. Metric mismatch (not adjacency blocking, but routing issue):
show isis database detail | match "Metric"
```

### Traceroute and path verification
```
traceroute 10.255.0.3 source 10.255.0.1              # verify IS-IS path
show route 10.255.0.3 detail                          # route attributes
show route 10.255.0.3 table inet.0 active-path        # active forwarding path
show route forwarding-table destination 10.255.0.3     # PFE programmed path
```

## Verification Commands Summary

```
# Adjacencies
show isis adjacency                                   # adjacency table
show isis adjacency detail                            # full adjacency info

# Database
show isis database                                    # LSDB summary
show isis database detail                             # full LSDB
show isis database extensive                          # maximum detail

# SPF
show isis spf log                                     # SPF history
show isis spf results                                 # SPF tree
show isis backup-spf results                          # TI-LFA backup paths

# Routes
show isis route                                       # IS-IS routes
show route protocol isis                              # RIB IS-IS routes

# Configuration
show isis overview                                    # global state
show isis interface                                   # interface state

# TE
show ted database                                     # TE database
show isis database detail | match "TE|admin|metric"   # TE TLVs

# SR
show isis database detail | match "SID|SRGB"          # SR advertisements
show isis adjacency detail | match "SID"              # adjacency SIDs
```

## Tips

- Always use wide-metrics-only in SP networks — narrow metrics are too limited for meaningful cost design
- Point-to-point interface type avoids DIS election overhead on P2P links — always set it
- Multi-topology is preferred over single-topology when IPv4 and IPv6 topologies differ
- BFD timers should be aggressive enough for fast detection but not so aggressive that CPU load causes false positives
- Mesh groups save flooding bandwidth but risk LSDB inconsistency if misconfigured — use only on true full-mesh segments
- Route leaking from L2 to L1 requires explicit policy — the default is only a default route toward the nearest L1/L2 router
- SPF delay of 200ms with holddown of 5000ms is a good starting point — too aggressive causes CPU churn during instability
- Overload-on-startup with a 300s timeout prevents traffic blackholing while routing tables converge after reboot
- IS-IS NET must be unique per router — duplicate system-ids cause LSDB corruption and routing loops
- Authentication should cover hello, CSNP, PSNP, and LSP PDUs for full protection

## See Also

- junos-segment-routing, junos-routing-fundamentals, junos-bgp-advanced, junos-high-availability, isis

## References

- [Juniper TechLibrary — IS-IS Overview](https://www.juniper.net/documentation/us/en/software/junos/is-is/topics/concept/is-is-overview.html)
- [Juniper TechLibrary — IS-IS Wide Metrics](https://www.juniper.net/documentation/us/en/software/junos/is-is/topics/concept/is-is-wide-metrics.html)
- [Juniper TechLibrary — IS-IS Multi-Topology](https://www.juniper.net/documentation/us/en/software/junos/is-is/topics/concept/is-is-multitopology-overview.html)
- [Juniper TechLibrary — IS-IS TE Extensions](https://www.juniper.net/documentation/us/en/software/junos/is-is/topics/concept/is-is-te-overview.html)
- [RFC 1195 — Use of OSI IS-IS for Routing in TCP/IP and Dual Environments](https://www.rfc-editor.org/rfc/rfc1195)
- [RFC 5305 — IS-IS Extensions for Traffic Engineering](https://www.rfc-editor.org/rfc/rfc5305)
- [RFC 5308 — Routing IPv6 with IS-IS](https://www.rfc-editor.org/rfc/rfc5308)
- [RFC 5120 — Multi-Topology IS-IS](https://www.rfc-editor.org/rfc/rfc5120)
- [RFC 8667 — IS-IS Extensions for Segment Routing](https://www.rfc-editor.org/rfc/rfc8667)
- [RFC 6232 — Purge Originator Identification TLV](https://www.rfc-editor.org/rfc/rfc6232)
- [RFC 5306 — IS-IS Restart Signaling](https://www.rfc-editor.org/rfc/rfc5306)
