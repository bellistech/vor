# JunOS Routing Policy (JNCIA-Junos Exam Topic)

Junos routing policy framework for controlling route import, export, redistribution, and attribute manipulation -- policy-options, terms, match criteria, actions, and default policies.

## Default Routing Policies

```
# What Junos does when NO explicit policy is configured:

Protocol        Import Default          Export Default
─────────────── ─────────────────────── ──────────────────────────────
BGP             Accept all BGP routes   Re-advertise all active BGP routes
OSPF            Accept all OSPF routes  Reject all (OSPF uses flooding, not policy)
IS-IS           Accept all IS-IS routes Reject all (IS-IS uses flooding, not policy)
RIP             Accept all RIP routes   Reject all
Static          N/A (always installed)  N/A
Direct          N/A (always installed)  N/A
Aggregate       N/A                     N/A

# Key exam point: BGP is the ONLY protocol that re-advertises by default
# OSPF/IS-IS use LSA flooding -- export policy controls REDISTRIBUTION into OSPF, not flooding
# Static/Direct routes are never advertised unless explicitly exported
```

## Import vs Export Policies

```
# IMPORT policy: controls what routes enter the routing table FROM a protocol
#   - Applied AFTER routes are received from neighbors
#   - Filters what a protocol contributes to inet.0
#   - Can modify route attributes before installation

# EXPORT policy: controls what routes are ADVERTISED BY a protocol
#   - Applied BEFORE routes are sent to neighbors
#   - Controls what is redistributed into a protocol
#   - Can modify route attributes before advertisement

# Application levels (most specific wins):
# 1. Peer/neighbor level     (most specific)
# 2. Group level
# 3. Protocol/global level   (least specific)

# BGP peer-level policy
set protocols bgp group EBGP neighbor 10.1.1.1 import FILTER-IN
set protocols bgp group EBGP neighbor 10.1.1.1 export FILTER-OUT

# BGP group-level policy
set protocols bgp group EBGP import GROUP-FILTER-IN
set protocols bgp group EBGP export GROUP-FILTER-OUT

# OSPF protocol-level policy (OSPF has no peer-level)
set protocols ospf export REDIST-INTO-OSPF
set protocols ospf import FILTER-OSPF-ROUTES
```

## Policy Evaluation Flow

```
# Evaluation order for MULTIPLE policies:
#
#   Policy 1 ──> Policy 2 ──> ... ──> Policy N ──> Default Policy
#
# Within each policy:
#
#   Term 1 ──> Term 2 ──> ... ──> Term N
#
# Within each term:
#   1. Evaluate ALL "from" conditions (logical AND)
#   2. If ALL match ──> execute "then" actions
#   3. If any "from" fails ──> skip to next term
#
# Flow control:
#   accept         Stop processing. Route is accepted.
#   reject         Stop processing. Route is rejected.
#   next term      Skip to next term in SAME policy.
#   next policy    Skip to first term of NEXT policy.
#   (no action)    Implicit "next term" -- continue evaluating.
#
# If a term has NO "from" clause: it matches EVERYTHING
# If a term has NO "then" clause: implicit "next term"
# If ALL terms/policies evaluated with no accept/reject: DEFAULT POLICY applies
```

## Policy Structure and Terms

```
# Basic structure
policy-options {
    policy-statement POLICY-NAME {
        term TERM-1 {
            from {
                /* match conditions -- all must match (AND logic) */
            }
            then {
                /* actions to take when matched */
            }
        }
        term TERM-2 {
            from { ... }
            then { ... }
        }
        /* terms without a name -- applies after all named terms */
        then reject;
    }
}

# Set-command syntax
set policy-options policy-statement POLICY-NAME term TERM-1 from protocol ospf
set policy-options policy-statement POLICY-NAME term TERM-1 from route-filter 10.0.0.0/8 orlonger
set policy-options policy-statement POLICY-NAME term TERM-1 then accept

# A term with no "from" matches everything (catch-all)
set policy-options policy-statement DENY-ALL term BLOCK then reject
```

## Match Criteria (from)

### Protocol and Route Source

```
from {
    protocol ospf;                    # match OSPF-learned routes
    protocol bgp;                     # match BGP-learned routes
    protocol static;                  # match static routes
    protocol direct;                  # match directly connected
    protocol aggregate;               # match aggregate routes
    protocol [ ospf bgp ];            # match multiple protocols (OR)
}
```

### Route Filters

```
from {
    # route-filter tests a route's prefix against patterns
    route-filter 10.0.0.0/8 exact;              # exactly 10.0.0.0/8
    route-filter 10.0.0.0/8 orlonger;           # 10.0.0.0/8 and any longer (/9,/10,...,/32)
    route-filter 10.0.0.0/8 longer;             # longer than /8 only (not /8 itself)
    route-filter 10.0.0.0/8 upto /24;           # /8 through /24
    route-filter 10.0.0.0/8 prefix-length-range /16-/24;  # /16 through /24 only
    route-filter 10.0.0.0/8 through 10.255.0.0/16;       # range of prefixes

    # Multiple route-filters in one term = OR logic
    route-filter 10.0.0.0/8 orlonger;
    route-filter 172.16.0.0/12 orlonger;
    route-filter 192.168.0.0/16 orlonger;
}
```

### Prefix Lists and Prefix-List Filters

```
# Define reusable prefix lists
set policy-options prefix-list RFC1918 10.0.0.0/8
set policy-options prefix-list RFC1918 172.16.0.0/12
set policy-options prefix-list RFC1918 192.168.0.0/16

set policy-options prefix-list CUSTOMER-NETS 198.51.100.0/24
set policy-options prefix-list CUSTOMER-NETS 203.0.113.0/24

from {
    prefix-list RFC1918;                         # exact match only
}

# prefix-list-filter allows match types (like route-filter)
from {
    prefix-list-filter RFC1918 orlonger;         # match list entries and longer
    prefix-list-filter CUSTOMER-NETS exact;      # exact match only
}
```

### BGP Attributes

```
from {
    community MY-COMMUNITY;                      # match community value
    community [ COMM-A COMM-B ];                 # match any listed community (OR)
    as-path MY-AS-PATH;                          # match AS path regex
    neighbor 10.1.1.1;                           # match specific BGP neighbor
    neighbor [ 10.1.1.1 10.1.1.2 ];              # match any listed neighbor
    local-preference 200;                        # match local-pref value
    metric 100;                                  # match MED value
}

# Community definition
set policy-options community MY-COMMUNITY members 65001:100
set policy-options community CUSTOMER-RT members 65001:*

# AS-path definition
set policy-options as-path MY-AS-PATH ".* 65002 .*"
set policy-options as-path DIRECT-PEER "^65002$"
set policy-options as-path ANY-PATH ".*"
```

### Interface and Other Criteria

```
from {
    interface ge-0/0/0.0;                        # match routes from interface
    interface [ ge-0/0/0.0 ge-0/0/1.0 ];         # match multiple interfaces
    next-hop 10.0.0.1;                           # match next-hop address
    area 0.0.0.0;                                # match OSPF area
    preference 170;                              # match route preference
    tag 100;                                     # match route tag
}
```

## Route-Filter Match Types

```
# Prefix: 10.0.0.0/8    Route being evaluated: 10.1.0.0/16

exact                   # ONLY 10.0.0.0/8 -- prefix and length must match exactly
orlonger                # 10.0.0.0/8 through 10.x.x.x/32 -- /8 and anything more specific
longer                  # 10.x.x.x/9 through 10.x.x.x/32 -- more specific only, NOT /8 itself
upto /24                # 10.0.0.0/8 through 10.x.x.x/24 -- /8 up to /24
prefix-length-range /16-/24   # 10.x.x.x/16 through 10.x.x.x/24 -- specific range only
through 10.255.0.0/16   # 10.0.0.0/8 through 10.255.0.0/16 -- address range

# Exam examples:
# Route 10.1.2.0/24 matches:
#   route-filter 10.0.0.0/8 orlonger        YES (/24 is longer than /8)
#   route-filter 10.0.0.0/8 exact           NO  (prefix length differs)
#   route-filter 10.0.0.0/8 longer          YES (/24 is strictly longer than /8)
#   route-filter 10.0.0.0/8 upto /24        YES (/24 is within /8 to /24)
#   route-filter 10.0.0.0/8 upto /16        NO  (/24 is longer than /16)
#   route-filter 10.1.2.0/24 exact          YES (exact match)
```

## Actions (then)

### Flow Control Actions

```
then {
    accept;                     # accept route, stop processing ALL policies
    reject;                     # reject route, stop processing ALL policies
    next term;                  # skip to next term in current policy
    next policy;                # skip to next policy entirely
}
# If "then" has ONLY modifying actions (no accept/reject/next):
#   implicit "next term" -- modification applied, evaluation continues
```

### Modifying Actions

```
then {
    local-preference 200;                   # set BGP local-pref (higher = preferred)
    local-preference add 50;                # increment local-pref
    local-preference subtract 50;           # decrement local-pref
    metric 100;                             # set MED
    metric add 10;                          # increment MED
    preference 15;                          # set route preference
    next-hop 10.0.0.1;                      # set next-hop address
    next-hop self;                          # set next-hop to self
    next-hop reject;                        # install as reject route
    next-hop discard;                       # install as discard route
    origin igp;                             # set BGP origin to IGP
    origin egp;                             # set BGP origin to EGP
    origin incomplete;                      # set BGP origin to incomplete
    tag 100;                                # set route tag

    community add MY-COMM;                  # add community (keep existing)
    community delete MY-COMM;               # remove specific community
    community set MY-COMM;                  # replace all communities
    as-path-prepend "65001 65001";          # prepend AS numbers
    as-path-prepend "65001" count 3;        # prepend 65001 three times
}

# IMPORTANT: modifying actions without accept/reject = implicit "next term"
# To modify AND accept: include both the modification AND accept
then {
    local-preference 200;
    accept;
}
```

## Policy Application

```
# Apply export policy to OSPF (redistribute into OSPF)
set protocols ospf export STATIC-TO-OSPF

# Apply import policy to OSPF (filter OSPF routes entering routing table)
set protocols ospf import FILTER-OSPF

# Apply export policy to BGP group
set protocols bgp group EBGP export ADVERTISE-ROUTES

# Apply import policy to BGP group
set protocols bgp group EBGP import INBOUND-FILTER

# Apply policy to specific BGP neighbor (overrides group policy)
set protocols bgp group EBGP neighbor 10.1.1.1 import NEIGHBOR-FILTER

# Apply multiple policies (evaluated in order listed)
set protocols bgp group EBGP import [ POLICY-1 POLICY-2 POLICY-3 ]
# POLICY-1 evaluated first, then POLICY-2, then POLICY-3, then default
```

## Practical Examples

### Redistribute Static Routes into OSPF

```
# Define which static routes to redistribute
set policy-options prefix-list STATIC-NETS 192.168.10.0/24
set policy-options prefix-list STATIC-NETS 192.168.20.0/24

set policy-options policy-statement STATIC-TO-OSPF term MATCH-STATIC from protocol static
set policy-options policy-statement STATIC-TO-OSPF term MATCH-STATIC from prefix-list STATIC-NETS
set policy-options policy-statement STATIC-TO-OSPF term MATCH-STATIC then accept
set policy-options policy-statement STATIC-TO-OSPF term DENY-REST then reject

# Apply to OSPF
set protocols ospf export STATIC-TO-OSPF
```

### BGP Inbound Route Filtering

```
# Reject RFC1918, accept customer prefixes, reject everything else
set policy-options prefix-list RFC1918 10.0.0.0/8
set policy-options prefix-list RFC1918 172.16.0.0/12
set policy-options prefix-list RFC1918 192.168.0.0/16

set policy-options prefix-list CUSTOMER-PREFIXES 198.51.100.0/24
set policy-options prefix-list CUSTOMER-PREFIXES 203.0.113.0/24

set policy-options policy-statement BGP-INBOUND term REJECT-RFC1918 from prefix-list-filter RFC1918 orlonger
set policy-options policy-statement BGP-INBOUND term REJECT-RFC1918 then reject

set policy-options policy-statement BGP-INBOUND term ACCEPT-CUSTOMER from prefix-list CUSTOMER-PREFIXES
set policy-options policy-statement BGP-INBOUND term ACCEPT-CUSTOMER then accept

set policy-options policy-statement BGP-INBOUND term DENY-ALL then reject

set protocols bgp group EBGP neighbor 203.0.113.1 import BGP-INBOUND
```

### Community-Based Policy

```
# Tag routes with community on import
set policy-options community CUSTOMER-A members 65001:100
set policy-options community CUSTOMER-B members 65001:200
set policy-options community BLACKHOLE members 65001:666

set policy-options policy-statement TAG-CUSTOMER-A term SET-COMM from neighbor 10.1.1.1
set policy-options policy-statement TAG-CUSTOMER-A term SET-COMM then community add CUSTOMER-A
set policy-options policy-statement TAG-CUSTOMER-A term SET-COMM then accept

# Use community for export decisions
set policy-options policy-statement UPSTREAM-EXPORT term CUST-A from community CUSTOMER-A
set policy-options policy-statement UPSTREAM-EXPORT term CUST-A then local-preference 200
set policy-options policy-statement UPSTREAM-EXPORT term CUST-A then accept

set policy-options policy-statement UPSTREAM-EXPORT term BLACKHOLE from community BLACKHOLE
set policy-options policy-statement UPSTREAM-EXPORT term BLACKHOLE then reject
```

### BGP Local Preference for Primary/Backup

```
set policy-options policy-statement PREFER-PRIMARY term SET-LP from neighbor 10.1.1.1
set policy-options policy-statement PREFER-PRIMARY term SET-LP then local-preference 200
set policy-options policy-statement PREFER-PRIMARY term SET-LP then accept

set policy-options policy-statement PREFER-BACKUP term SET-LP from neighbor 10.2.2.2
set policy-options policy-statement PREFER-BACKUP term SET-LP then local-preference 50
set policy-options policy-statement PREFER-BACKUP term SET-LP then accept

set protocols bgp group EBGP neighbor 10.1.1.1 import PREFER-PRIMARY
set protocols bgp group EBGP neighbor 10.2.2.2 import PREFER-BACKUP
```

### AS-Path Prepending on Export

```
set policy-options as-path-group PREPEND-3 as-path-prepend "65001 65001 65001"

set policy-options policy-statement MAKE-BACKUP term PREPEND then as-path-prepend "65001 65001 65001"
set policy-options policy-statement MAKE-BACKUP term PREPEND then accept

set protocols bgp group EBGP-BACKUP export MAKE-BACKUP
```

## Verification Commands

```
show policy                                      # list all configured policies
show route receive-protocol bgp 10.1.1.1         # routes received from peer (pre-policy)
show route protocol bgp                          # BGP routes in routing table (post-import-policy)
show route advertising-protocol bgp 10.1.1.1     # routes advertised to peer (post-export-policy)

test policy POLICY-NAME 10.0.0.0/8               # test a route against a policy
show route 10.0.0.0/8 detail                     # see which policy accepted a route
show configuration policy-options                # view all policy config
show configuration policy-options | display set  # view as set commands
```

## Tips

- The default BGP import policy is accept; the default BGP export policy re-advertises all active BGP routes -- always apply explicit policies in production.
- OSPF and IS-IS use flooding, not policy, for route distribution; export policy on OSPF controls redistribution INTO OSPF from other protocols.
- Multiple `from` conditions in one term are logical AND; multiple values in one condition (brackets) are logical OR.
- Multiple `route-filter` entries in one term are logical OR -- if any route-filter matches, the from clause for that part is satisfied.
- A modifying action without `accept` or `reject` causes an implicit `next term` -- the modification is applied but evaluation continues.
- Always include a final catch-all term (`then reject` or `then accept`) to make policy behavior explicit rather than relying on default policy.
- Use `test policy` to verify policy behavior before committing -- catches logic errors without impacting live traffic.
- When multiple policies are chained with `[ POLICY-1 POLICY-2 ]`, an `accept` or `reject` in any policy is final and terminates ALL processing.
- `prefix-list` matches exact prefixes only; use `prefix-list-filter` when you need orlonger/upto/etc. match types.
- Community `set` replaces ALL communities; use `add` to append without removing existing communities.

## See Also

- junos
- bgp
- ospf
- is-is

## References

- [Junos OS Routing Policy Configuration Guide](https://www.juniper.net/documentation/us/en/software/junos/routing-policy/topics/topic-map/policy-overview.html)
- [Juniper JNCIA-Junos Study Guide](https://www.juniper.net/us/en/training/certification/tracks/junos/jncia-junos.html)
- [Junos Policy Framework Overview](https://www.juniper.net/documentation/us/en/software/junos/routing-policy/topics/concept/policy-routing-policies-overview.html)
- [Junos Route Filters and Match Conditions](https://www.juniper.net/documentation/us/en/software/junos/routing-policy/topics/concept/policy-configuring-route-lists-for-use-in-routing-policy-match-conditions.html)
- [Juniper TechLibrary — Routing Policy](https://www.juniper.net/documentation/)
- [Juniper Learning Portal](https://learningportal.juniper.net/)
