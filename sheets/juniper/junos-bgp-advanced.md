# JunOS BGP Advanced

Advanced BGP configuration and features for SP environments — routing policy, path selection, route reflection, ADD-PATH, BGP-LU, flowspec, RPKI, graceful restart, and monitoring on JunOS platforms.

## BGP Configuration Structure

### Groups and neighbors
```
# BGP configuration hierarchy in JunOS:
#   protocols bgp → group → neighbor
# Settings inherit downward: global → group → neighbor (neighbor overrides all)

# IBGP group
set protocols bgp group IBGP type internal
set protocols bgp group IBGP local-address 10.255.0.1
set protocols bgp group IBGP family inet unicast
set protocols bgp group IBGP family inet-vpn unicast
set protocols bgp group IBGP family inet6 unicast
set protocols bgp group IBGP neighbor 10.255.0.2
set protocols bgp group IBGP neighbor 10.255.0.3

# EBGP group
set protocols bgp group EBGP type external
set protocols bgp group EBGP peer-as 65001
set protocols bgp group EBGP family inet unicast
set protocols bgp group EBGP neighbor 192.168.1.2

# Per-neighbor override
set protocols bgp group EBGP neighbor 192.168.2.2 peer-as 65002
```

### Address families
```
set protocols bgp group IBGP family inet unicast              # IPv4 unicast
set protocols bgp group IBGP family inet6 unicast             # IPv6 unicast
set protocols bgp group IBGP family inet-vpn unicast          # L3VPN IPv4
set protocols bgp group IBGP family inet6-vpn unicast         # L3VPN IPv6
set protocols bgp group IBGP family l2vpn signaling           # L2VPN
set protocols bgp group IBGP family inet labeled-unicast      # BGP-LU
set protocols bgp group IBGP family inet flow                 # flowspec
set protocols bgp group IBGP family route-target              # RT constrain
set protocols bgp group IBGP family traffic-engineering unicast  # BGP-LS TE
```

### Authentication
```
set protocols bgp group EBGP authentication-key "SecretKey123"
set protocols bgp group EBGP authentication-algorithm md5
# Or AO (Authentication Option, RFC 5925):
set protocols bgp group EBGP authentication-algorithm ao
set protocols bgp group EBGP authentication-key-chain KC-BGP
```

## Routing Policy

### Policy-statement structure
```
# Routing policies control route import/export
# Structure: policy-statement → term → from/then
# Evaluation: terms evaluated sequentially, first match wins
# If no term matches: default action depends on application context

set policy-options policy-statement EXPORT-POLICY term CONNECTED from protocol direct
set policy-options policy-statement EXPORT-POLICY term CONNECTED from route-filter 10.0.0.0/8 orlonger
set policy-options policy-statement EXPORT-POLICY term CONNECTED then accept

set policy-options policy-statement EXPORT-POLICY term STATIC from protocol static
set policy-options policy-statement EXPORT-POLICY term STATIC then reject

set policy-options policy-statement EXPORT-POLICY term DEFAULT then reject
```

### From conditions
```
from {
    protocol bgp;                        # route source protocol
    route-filter 10.0.0.0/8 exact;       # prefix match
    route-filter 10.0.0.0/8 orlonger;    # prefix or more specific
    route-filter 10.0.0.0/8 upto /24;    # prefix up to /24
    route-filter 10.0.0.0/8 prefix-length-range /16-/24;  # range
    prefix-list CUSTOMERS;                # named prefix list
    as-path AS-PATH-FILTER;              # AS path regex
    community COMMUNITY-NAME;            # community match
    neighbor 10.0.0.2;                   # specific neighbor
    family inet;                         # address family
    local-preference 200;                # LP value
    metric 100;                          # MED value
    origin igp;                          # origin attribute
}
```

### Then actions
```
then {
    accept;                              # accept the route
    reject;                              # reject the route
    next term;                           # continue to next term
    next policy;                         # continue to next policy

    # Attribute modifications
    local-preference 200;                # set LP
    metric 100;                          # set MED
    as-path-prepend "65001 65001";       # prepend AS path
    origin igp;                          # set origin
    next-hop self;                       # set next-hop to self
    next-hop 10.0.0.1;                   # set specific next-hop
    community add COMM-NAME;             # add community
    community delete COMM-NAME;          # delete community
    community set COMM-NAME;             # replace community
}
```

### Apply routing policy
```
# Import policy (received routes)
set protocols bgp group EBGP import IMPORT-POLICY

# Export policy (advertised routes)
set protocols bgp group EBGP export EXPORT-POLICY

# Multiple policies (evaluated in order, first definitive match wins)
set protocols bgp group EBGP import [ POLICY-1 POLICY-2 POLICY-3 ]

# Per-neighbor policy override
set protocols bgp group EBGP neighbor 10.0.0.2 import NEIGHBOR-SPECIFIC-POLICY
```

### AS path filters
```
set policy-options as-path AS-ORIGINATE ".* 65001"         # originated by AS 65001
set policy-options as-path AS-TRANSIT ".* 65001 .*"        # transits AS 65001
set policy-options as-path AS-DIRECT "65001"               # directly from AS 65001
set policy-options as-path AS-SHORT ".{0,3}"               # path length 0-3
set policy-options as-path AS-ANY ".*"                     # any path
```

### Community operations
```
# Define communities
set policy-options community NO-EXPORT members no-export
set policy-options community NO-ADVERTISE members no-advertise
set policy-options community CUST-A members 65000:100
set policy-options community BLACKHOLE members 65000:666

# Extended communities
set policy-options community RT-VPN-A members target:65000:100
set policy-options community SOO-SITE-A members origin:65000:1

# Large communities (RFC 8092)
set policy-options community LARGE-COMM members large:65000:1:100

# Regex community match
set policy-options community ANY-CUST members "65000:.*"
```

### Prefix lists
```
set policy-options prefix-list BOGONS 0.0.0.0/8
set policy-options prefix-list BOGONS 10.0.0.0/8
set policy-options prefix-list BOGONS 100.64.0.0/10
set policy-options prefix-list BOGONS 127.0.0.0/8
set policy-options prefix-list BOGONS 169.254.0.0/16
set policy-options prefix-list BOGONS 172.16.0.0/12
set policy-options prefix-list BOGONS 192.0.0.0/24
set policy-options prefix-list BOGONS 192.0.2.0/24
set policy-options prefix-list BOGONS 192.168.0.0/16
set policy-options prefix-list BOGONS 198.18.0.0/15
set policy-options prefix-list BOGONS 198.51.100.0/24
set policy-options prefix-list BOGONS 203.0.113.0/24
set policy-options prefix-list BOGONS 224.0.0.0/4
set policy-options prefix-list BOGONS 240.0.0.0/4
```

## BGP Path Selection (JunOS Order)

### JunOS best path algorithm
```
# JunOS BGP path selection order (top to bottom, first differentiator wins):

1.  Highest local-preference (default 100)
2.  Shortest AS-path length
3.  Lowest origin type (IGP < EGP < INCOMPLETE)
4.  Lowest MED (compared within same neighboring AS, unless always-compare-med)
5.  EBGP over IBGP
6.  Lowest IGP metric to next-hop (nearest exit / hot-potato)
7.  Active route preferred
8.  Shortest route reflection cluster-list length
9.  Lowest router-id (originator-id for reflected routes)
10. Lowest peer IP address (tie-breaker)

# Key JunOS-specific behaviors:
#   - MED comparison: same neighbor AS only (default) — use "path-selection always-compare-med" to compare across AS
#   - Deterministic MED: JunOS groups by neighboring AS and compares within group, then between groups
#   - Router-ID tie-break: originator-id (from RR) used instead of neighbor router-id for reflected routes
```

### Path selection tuning
```
# Always compare MED across different AS origins
set protocols bgp path-selection always-compare-med

# Disable MED comparison entirely
set protocols bgp path-selection med-plus-igp

# External route preference (default: prefer EBGP)
set protocols bgp path-selection external-router-mac

# Cisco-compatible mode
set protocols bgp path-selection cisco-non-deterministic
```

## Route Reflection

### Route reflector configuration
```
# RR server
set protocols bgp group IBGP-CLIENTS type internal
set protocols bgp group IBGP-CLIENTS cluster 10.255.0.1
set protocols bgp group IBGP-CLIENTS neighbor 10.255.0.2
set protocols bgp group IBGP-CLIENTS neighbor 10.255.0.3
set protocols bgp group IBGP-CLIENTS neighbor 10.255.0.4

# RR client — no special config needed (standard IBGP)
set protocols bgp group IBGP type internal
set protocols bgp group IBGP neighbor 10.255.0.1
```

### Route reflection rules
```
# RR reflects routes according to these rules:
# Route learned from:          Reflected to:
#   EBGP peer                → All clients + non-clients
#   Client                   → All other clients + non-clients + EBGP
#   Non-client (IBGP peer)   → All clients only (NOT to other non-clients)
#
# RR sets ORIGINATOR_ID = originator's router-id
# RR appends own cluster-id to CLUSTER_LIST
# Client loop detection: reject if own router-id in ORIGINATOR_ID
# Cluster loop detection: reject if own cluster-id in CLUSTER_LIST
```

### Verify route reflection
```
show bgp neighbor | match "Cluster"           # cluster-id configuration
show route 10.0.0.0/8 detail | match "Originator|Cluster"  # reflected route attributes
```

## Confederations

### Configure BGP confederation
```
set routing-options autonomous-system 65000
set routing-options confederation 65000 members [ 65001 65002 65003 ]

# Sub-AS peering (treated as EBGP within confederation)
set protocols bgp group CONFED type external
set protocols bgp group CONFED peer-as 65002
set protocols bgp group CONFED neighbor 10.0.0.2
```

## ADD-PATH

### Send and receive multiple paths
```
# Allow multiple paths per prefix (not just best path)
set protocols bgp group IBGP family inet unicast add-path receive
set protocols bgp group IBGP family inet unicast add-path send path-count 6

# Verify
show bgp neighbor | match "Add-path"
show route 10.0.0.0/8 all                    # see multiple paths
```

## Multipath

### BGP multipath configuration
```
# Install multiple equal-cost BGP paths into forwarding table
set protocols bgp group EBGP multipath
set protocols bgp group EBGP multipath multiple-as  # allow multipath across different AS

# Limit number of multipaths
set routing-options forwarding-table export ECMP-POLICY

set policy-options policy-statement ECMP-POLICY term ECMP then load-balance per-packet
# "per-packet" is actually per-flow (hash-based) in JunOS
```

## BGP-LU (Labeled-Unicast)

### BGP labeled-unicast configuration
```
# BGP-LU allows BGP to distribute MPLS labels for prefixes
# Used in seamless MPLS and inter-AS VPN Option C

set protocols bgp group IBGP family inet labeled-unicast
set protocols bgp group IBGP family inet labeled-unicast resolve-vpn
set protocols bgp group IBGP family inet labeled-unicast rib inet.3

# Explicit null for connected prefixes
set protocols bgp group IBGP family inet labeled-unicast explicit-null connected-only

# Advertise labeled routes
set policy-options policy-statement BGP-LU-EXPORT term LOOPBACKS from protocol direct
set policy-options policy-statement BGP-LU-EXPORT term LOOPBACKS from route-filter 10.255.0.0/16 orlonger
set policy-options policy-statement BGP-LU-EXPORT term LOOPBACKS then accept
set protocols bgp group IBGP export BGP-LU-EXPORT
```

### Verify BGP-LU
```
show route table inet.3 protocol bgp         # labeled routes
show route 10.255.0.2/32 detail              # label binding
show bgp neighbor | match "labeled"          # labeled-unicast capability
```

## BGP Flowspec

### Flowspec configuration
```
# BGP flowspec distributes traffic filtering rules via BGP
set protocols bgp group IBGP family inet flow
set protocols bgp group IBGP family inet flow no-validate FLOWSPEC-VALID

# Define flowspec routes (local)
set routing-options flow route BLOCK-ATTACK match destination 10.0.0.1/32
set routing-options flow route BLOCK-ATTACK match protocol udp
set routing-options flow route BLOCK-ATTACK match destination-port 53
set routing-options flow route BLOCK-ATTACK then discard

# Rate-limit instead of discard
set routing-options flow route RATE-LIMIT match source 192.0.2.0/24
set routing-options flow route RATE-LIMIT then rate-limit 1m

# Redirect to routing instance
set routing-options flow route REDIRECT match destination 10.0.0.0/24
set routing-options flow route REDIRECT then routing-instance SCRUBBING
```

### Verify flowspec
```
show route table inetflow.0                  # flowspec routes
show firewall filter __flowspec_default_inet__  # auto-generated filter
```

## BGP-LS (Link-State)

### BGP-LS configuration
```
# BGP-LS exports IGP topology to external controllers (SDN)
set protocols bgp group CONTROLLER family traffic-engineering unicast
set protocols bgp group CONTROLLER neighbor 10.0.0.100

# Export IS-IS topology via BGP-LS
set protocols isis traffic-engineering bgp-ls
```

## RPKI / Origin Validation

### Configure RPKI cache connection
```
# Connect to RPKI validation cache (RTR protocol)
set routing-options validation group RPKI-CACHE session 10.0.0.50 port 8282
set routing-options validation group RPKI-CACHE session 10.0.0.50 refresh-time 300
set routing-options validation group RPKI-CACHE session 10.0.0.51 port 8282

# Apply validation to BGP
set protocols bgp group EBGP family inet unicast prefix-limit maximum 500000
set protocols bgp group EBGP import RPKI-POLICY
```

### RPKI routing policy
```
# Policy based on validation state
set policy-options policy-statement RPKI-POLICY term VALID from validation-database valid
set policy-options policy-statement RPKI-POLICY term VALID then local-preference 200
set policy-options policy-statement RPKI-POLICY term VALID then validation-state valid
set policy-options policy-statement RPKI-POLICY term VALID then accept

set policy-options policy-statement RPKI-POLICY term INVALID from validation-database invalid
set policy-options policy-statement RPKI-POLICY term INVALID then validation-state invalid
set policy-options policy-statement RPKI-POLICY term INVALID then reject

set policy-options policy-statement RPKI-POLICY term UNKNOWN then validation-state unknown
set policy-options policy-statement RPKI-POLICY term UNKNOWN then accept
```

### Verify RPKI
```
show validation session                       # RTR session state
show validation database                      # VRP (Validated ROA Payload) entries
show validation statistics                    # validation counters
show route 10.0.0.0/8 detail | match "validation"  # per-route validation state
```

## Graceful Restart

### Standard graceful restart
```
set routing-options graceful-restart
set protocols bgp graceful-restart

# Tuning
set routing-options graceful-restart restart-duration 300         # max restart time (seconds)
set protocols bgp graceful-restart stale-routes-time 300         # how long to keep stale routes
```

### Long-Lived Graceful Restart (LLGR)
```
# LLGR keeps stale routes for hours/days (not just minutes)
# Used for persistent failures where session should not be torn down

set protocols bgp group IBGP family inet unicast long-lived-graceful-restart restart-time 86400
set protocols bgp group IBGP family inet-vpn unicast long-lived-graceful-restart restart-time 86400

# LLGR communities
# Routes in LLGR stale state are tagged with LLGR_STALE community
# These routes have lowest priority but remain usable as last resort
```

### Verify graceful restart
```
show bgp neighbor | match "Restart|LLGR"      # GR/LLGR capability
show bgp neighbor | match "stale"             # stale route counts
```

## BMP (BGP Monitoring Protocol)

### Configure BMP
```
# Send BGP RIB and update data to monitoring station
set routing-options bmp station MONITOR
set routing-options bmp station MONITOR station-address 10.0.0.100
set routing-options bmp station MONITOR station-port 5000
set routing-options bmp station MONITOR connection-mode active
set routing-options bmp station MONITOR statistics-timeout 300
set routing-options bmp station MONITOR route-monitoring pre-policy
set routing-options bmp station MONITOR route-monitoring post-policy
```

### Verify BMP
```
show bgp bmp                                  # BMP station status
```

## BGP Error Handling

### Configure error handling
```
# Treat malformed attributes as withdraw (RFC 7606) instead of session reset
set protocols bgp group EBGP bgp-error-tolerance malformed-route-handling treat-as-withdraw

# Prefix limits
set protocols bgp group EBGP family inet unicast prefix-limit maximum 500000
set protocols bgp group EBGP family inet unicast prefix-limit teardown 80 idle-timeout 30

# Teardown at 80% with warning, idle for 30 minutes after teardown
```

## Verification Commands

### BGP state
```
show bgp summary                              # all peers, state, prefixes
show bgp neighbor                             # detailed neighbor info
show bgp neighbor 10.0.0.2                    # specific neighbor
show bgp neighbor 10.0.0.2 | match "State|Received|Advertised"
```

### Routes
```
show route protocol bgp                       # all BGP routes
show route protocol bgp table inet.0          # IPv4 BGP routes
show route 10.0.0.0/8 detail                  # detailed route info
show route 10.0.0.0/8 all                     # all paths (including non-best)
show route advertising-protocol bgp 10.0.0.2  # routes advertised to peer
show route receive-protocol bgp 10.0.0.2      # routes received from peer
```

### Policy evaluation
```
show policy POLICY-NAME                        # view policy configuration
test policy POLICY-NAME 10.0.0.0/8            # test route against policy
show route 10.0.0.0/8 detail | match "Communities|AS path|Local"  # route attributes
```

### Tables
```
show route summary                             # route counts per table
show route table bgp.l3vpn.0                  # L3VPN routes
show route table inet.3                        # labeled routes
show route table inetflow.0                   # flowspec routes
```

## Tips

- JunOS evaluates import policies on received routes BEFORE best path selection — attribute changes in import policy affect path selection
- Export policies default to rejecting all routes unless explicitly accepted — always include a final term
- `next policy` skips remaining terms in current policy and moves to the next policy in the chain
- MED is compared only within the same neighboring AS by default — enable `always-compare-med` for cross-AS comparison
- BGP-LU routes install in inet.3 by default — use `rib inet.3` or `resolve-vpn` for VPN resolution
- RPKI validation state is an attribute, not a filter — use routing policy to act on valid/invalid/unknown
- LLGR stale routes have lowest priority — they only win if no other route exists for the prefix
- Use `show route advertising-protocol bgp <peer>` to verify what you are actually sending (post-export-policy)
- Flowspec auto-generates firewall filters — verify with `show firewall` to confirm expected behavior
- BMP pre-policy captures all received routes; post-policy captures only accepted routes after import policy

## See Also

- junos-routing-policy, junos-routing-fundamentals, junos-segment-routing, bgp, rpki

## References

- [Juniper TechLibrary — BGP Overview](https://www.juniper.net/documentation/us/en/software/junos/bgp/topics/concept/bgp-overview.html)
- [Juniper TechLibrary — Routing Policy](https://www.juniper.net/documentation/us/en/software/junos/routing-policy/topics/concept/routing-policy-overview.html)
- [Juniper TechLibrary — BGP Route Reflection](https://www.juniper.net/documentation/us/en/software/junos/bgp/topics/concept/bgp-route-reflection-overview.html)
- [Juniper TechLibrary — BGP Flowspec](https://www.juniper.net/documentation/us/en/software/junos/bgp/topics/concept/bgp-flowspec-overview.html)
- [Juniper TechLibrary — RPKI](https://www.juniper.net/documentation/us/en/software/junos/bgp/topics/concept/bgp-origin-validation-overview.html)
- [RFC 4271 — BGP-4](https://www.rfc-editor.org/rfc/rfc4271)
- [RFC 4456 — BGP Route Reflection](https://www.rfc-editor.org/rfc/rfc4456)
- [RFC 7911 — BGP ADD-PATH](https://www.rfc-editor.org/rfc/rfc7911)
- [RFC 8097 — BGP Prefix Origin Validation State](https://www.rfc-editor.org/rfc/rfc8097)
- [RFC 8955 — BGP Flowspec](https://www.rfc-editor.org/rfc/rfc8955)
- [RFC 7854 — BGP Monitoring Protocol](https://www.rfc-editor.org/rfc/rfc7854)
- [RFC 4724 — Graceful Restart](https://www.rfc-editor.org/rfc/rfc4724)
- [RFC 9494 — Long-Lived Graceful Restart](https://www.rfc-editor.org/rfc/rfc9494)
- [RFC 8092 — BGP Large Communities](https://www.rfc-editor.org/rfc/rfc8092)
