# JunOS Routing Policy — Complete Evaluation Framework

> *Junos routing policy is a deterministic decision engine: routes enter, traverse an ordered chain of policies and terms, and exit with a binary accept/reject verdict plus optional attribute modifications. Mastering the evaluation algorithm is the single most important JNCIA routing topic.*

---

## 1. Policy Evaluation Algorithm

### The Complete Flowchart

The Junos policy engine evaluates routes through a strict, deterministic pipeline. Understanding this flow is critical for predicting policy behavior.

```
  Route enters policy evaluation
          │
          ▼
  ┌─────────────────────┐
  │  Policy 1, Term 1   │
  │  Evaluate "from"    │──── No match ────┐
  │  (all conditions)   │                  │
  └─────────┬───────────┘                  │
        Match │                            │
          ▼                                │
  ┌─────────────────────┐                  │
  │  Execute "then"     │                  │
  │  actions            │                  │
  └─────────┬───────────┘                  │
            │                              │
     ┌──────┼──────────┬──────────┐        │
     ▼      ▼          ▼          ▼        ▼
  accept  reject   next term  next policy  │
   (DONE)  (DONE)     │          │         │
                      │          │         │
                      ▼          │         │
              ┌───────────────┐  │         │
              │ Policy 1,     │  │         │
              │ Term 2        │◄─┘─────────┘
              │ Evaluate...   │
              └───────┬───────┘
                      │
               (repeat for each term)
                      │
                      ▼ (all terms exhausted)
              ┌───────────────┐
              │ Policy 2,     │
              │ Term 1        │
              │ Evaluate...   │
              └───────┬───────┘
                      │
               (repeat for each policy)
                      │
                      ▼ (all policies exhausted)
              ┌───────────────┐
              │ DEFAULT POLICY│
              │ (per protocol)│
              └───────────────┘
```

### Algorithm Rules

1. **Terms within a policy** are evaluated top-to-bottom, in the order they appear in the configuration.
2. **Multiple `from` conditions** within a single term are logically ANDed -- ALL must match for the term to match.
3. **Multiple values** within a single `from` condition (using brackets `[ ]`) are logically ORed -- ANY can match.
4. **Multiple `route-filter`** entries within a term are ORed -- any matching route-filter satisfies that condition.
5. **A term with no `from` clause** matches every route unconditionally.
6. **A term with no `then` clause** (or `then` with only modifying actions and no terminating action) performs an implicit **next term**.
7. **`accept` and `reject`** are terminating actions -- they immediately stop ALL policy processing (across all policies in the chain).
8. **`next term`** skips to the next term within the same policy.
9. **`next policy`** skips to the first term of the next policy in the chain.
10. **Modifying actions** (set local-pref, add community, etc.) are applied immediately and persist even if evaluation continues to subsequent terms/policies.

### Critical Subtlety: Sticky Modifications

When a term applies a modifying action but no terminating action (implicit next term), the modification sticks to the route for the remainder of evaluation. If a later term or policy accepts the route, the modification is preserved. If the route is ultimately rejected, the modification is discarded.

```
# Example: modification sticks through evaluation
policy-statement MULTI-STEP {
    term SET-METRIC {
        from protocol static;
        then {
            metric 50;
            # no accept/reject -- implicit next term
            # metric 50 is applied, evaluation continues
        }
    }
    term SET-LP-AND-ACCEPT {
        from protocol static;
        then {
            local-preference 200;
            accept;
            # route is accepted with BOTH metric 50 AND local-pref 200
        }
    }
}
```

---

## 2. Default Policy Per Protocol (Complete Table)

Every routing protocol has a built-in default policy that applies when no explicit policy matches (or no explicit policy is configured). These defaults are the last resort in the evaluation chain.

```
Protocol     Direction   Default Action   Behavior Details
──────────── ────────── ──────────────── ──────────────────────────────────────
BGP          Import      Accept           Accept all received BGP routes into
                                          inet.0; all path attributes preserved

BGP          Export      Accept           Re-advertise all active BGP routes
                                          to all BGP peers (iBGP and eBGP)

OSPF         Import      Accept           Accept all OSPF routes into inet.0
                                          (routes from SPF calculation)

OSPF         Export      Reject           Reject redistribution of non-OSPF
                                          routes; OSPF native routes flood
                                          via LSAs regardless of policy

IS-IS        Import      Accept           Accept all IS-IS routes into inet.0

IS-IS        Export      Reject           Reject redistribution; IS-IS native
                                          routes flood via LSPs regardless

RIP          Import      Accept           Accept all RIP routes into inet.0

RIP          Export      Reject           Do not advertise non-RIP routes

PIM          Import      Accept           Accept all PIM join/prune state

PIM          Export      Reject           Reject all

Aggregate    N/A         N/A              Aggregate routes are generated locally
                                          when contributing routes exist

Static       N/A         N/A              Always installed in routing table
                                          when configured and next-hop valid

Direct       N/A         N/A              Always installed for active interfaces
```

### Why This Matters

The default policies explain common exam scenarios:

- **"Why do BGP routes appear without any import policy?"** -- Because the default BGP import policy is accept.
- **"Why don't static routes appear in OSPF?"** -- Because the default OSPF export policy is reject. You must create an export policy to redistribute.
- **"I deleted my BGP export policy and routes are still advertised. Why?"** -- Because the default BGP export policy re-advertises all active BGP routes.

---

## 3. Route-Filter Match Type Visual Diagrams

Route-filter match types define which prefix lengths are matched for a given base prefix. All examples use base prefix `10.0.0.0/8`.

### exact

```
Prefix length:  /8  /9 /10 /11 /12 /13 /14 /15 /16 ... /24 ... /32
                 ██
                 ↑
                ONLY /8 matches

route-filter 10.0.0.0/8 exact
  10.0.0.0/8     ✓ MATCH
  10.0.0.0/9     ✗
  10.1.0.0/16    ✗
  10.1.2.0/24    ✗
```

### orlonger

```
Prefix length:  /8  /9 /10 /11 /12 /13 /14 /15 /16 ... /24 ... /32
                 ██████████████████████████████████████████████████
                 ↑                                                ↑
                /8 ─────────── everything through ───────────── /32

route-filter 10.0.0.0/8 orlonger
  10.0.0.0/8     ✓ MATCH
  10.0.0.0/9     ✓ MATCH
  10.1.0.0/16    ✓ MATCH
  10.1.2.0/24    ✓ MATCH
  10.1.2.3/32    ✓ MATCH
  11.0.0.0/8     ✗ (wrong prefix)
```

### longer

```
Prefix length:  /8  /9 /10 /11 /12 /13 /14 /15 /16 ... /24 ... /32
                     ██████████████████████████████████████████████
                     ↑                                            ↑
                    /9 ─────────── everything through ──────── /32
                (excludes /8 itself)

route-filter 10.0.0.0/8 longer
  10.0.0.0/8     ✗ (exact /8 excluded)
  10.0.0.0/9     ✓ MATCH
  10.1.0.0/16    ✓ MATCH
  10.1.2.0/24    ✓ MATCH
```

### upto /24

```
Prefix length:  /8  /9 /10 /11 /12 /13 /14 /15 /16 ... /24  /25 ... /32
                 ██████████████████████████████████████████
                 ↑                                        ↑
                /8 ───────────── through ──────────────── /24

route-filter 10.0.0.0/8 upto /24
  10.0.0.0/8     ✓ MATCH
  10.1.0.0/16    ✓ MATCH
  10.1.2.0/24    ✓ MATCH
  10.1.2.128/25  ✗ (longer than /24)
  10.1.2.3/32    ✗ (longer than /24)
```

### prefix-length-range /16-/24

```
Prefix length:  /8  /9 /10 ... /15 /16 /17 /18 ... /23 /24  /25 ... /32
                                    ████████████████████████
                                    ↑                      ↑
                                   /16 ──── through ──── /24
                (excludes /8 through /15)

route-filter 10.0.0.0/8 prefix-length-range /16-/24
  10.0.0.0/8     ✗ (shorter than /16)
  10.0.0.0/12    ✗ (shorter than /16)
  10.1.0.0/16    ✓ MATCH
  10.1.2.0/24    ✓ MATCH
  10.1.2.128/25  ✗ (longer than /24)
```

### through

```
# "through" matches a range of PREFIXES (not just lengths)
route-filter 10.0.0.0/8 through 10.255.0.0/16

  10.0.0.0/8     ✓ MATCH (start of range)
  10.0.0.0/16    ✓ MATCH (within range)
  10.128.0.0/16  ✓ MATCH (within range)
  10.255.0.0/16  ✓ MATCH (end of range)
  10.0.0.0/24    ✗ (not in the range progression)
```

### Comparison Summary Table

```
Match Type               Includes /8?   /9-/15?   /16-/24?   /25-/32?
───────────────────────── ──────────── ───────── ────────── ──────────
exact                         ✓          ✗          ✗          ✗
orlonger                      ✓          ✓          ✓          ✓
longer                        ✗          ✓          ✓          ✓
upto /24                      ✓          ✓          ✓          ✗
prefix-length-range /16-/24   ✗          ✗          ✓          ✗
```

---

## 4. Community Regular Expressions

Communities are 32-bit values expressed as `AS:value` (e.g., `65001:100`). Junos supports regex-like matching for community values using `community` definitions under `policy-options`.

### Community Definition Syntax

```
# Exact community match
set policy-options community CUST-A members 65001:100

# Wildcard in value portion
set policy-options community ALL-FROM-65001 members 65001:*

# Regex patterns (enclosed in quotes)
set policy-options community ANY-CUSTOMER members "65001:[100-199]"
set policy-options community BLACKHOLE members ".*:666"

# Multiple members (route must have ALL listed -- AND logic)
set policy-options community BOTH-TAGS members [ 65001:100 65001:200 ]
```

### Community Regex Patterns

```
Pattern                Matches                           Example
────────────────────── ─────────────────────────────── ──────────
65001:100              Exactly 65001:100                 65001:100
65001:*                Any value in AS 65001             65001:0 through 65001:65535
*:100                  Value 100 in any AS               1:100, 65001:100, 65534:100
65001:[100-199]        Values 100-199 in AS 65001        65001:100, 65001:150, 65001:199
"65001:1.."            Regex: 1 followed by 2 chars      65001:100 through 65001:199
".*:666"               Value 666 in any AS               Any AS with value 666
```

### Well-Known Communities

```
Community Name         Numeric Value    Action
────────────────────── ──────────────── ────────────────────────────────
no-export              65535:65281      Do not export beyond confederation
no-advertise           65535:65282      Do not advertise to any peer
no-export-subconfed    65535:65283      Do not export beyond local AS
                                       (within confederation)

# Usage in policy
set policy-options community NO-EXPORT members no-export

policy-statement KEEP-INTERNAL {
    term TAG-NO-EXPORT {
        from prefix-list INTERNAL-ONLY;
        then {
            community add NO-EXPORT;
            accept;
        }
    }
}
```

### Extended and Large Communities

```
# Extended communities (used with VPNs, typically)
# Format: type:administrator:assigned-number
set policy-options community RT-CUST-A members target:65001:100
set policy-options community SOO-SITE1 members origin:65001:1

# Large communities (RFC 8092) -- 3 x 32-bit fields
# Format: global-admin:local-data-1:local-data-2
set policy-options community LARGE-COMM members large:65001:1:100
```

---

## 5. Policy Testing with `test policy`

The `test policy` command allows you to simulate policy evaluation against specific routes without affecting live traffic. This is essential for validating policy logic before applying it.

### Basic Syntax

```
# Test a specific prefix against a policy
test policy POLICY-NAME 10.0.0.0/8

# Test with specific protocol origin
test policy POLICY-NAME 10.0.0.0/8 protocol bgp

# Test with specific attributes
test policy POLICY-NAME 10.0.0.0/8 protocol bgp neighbor 10.1.1.1
```

### Reading Test Output

```
user@router> test policy BGP-INBOUND 198.51.100.0/24

Policy BGP-INBOUND, term REJECT-RFC1918: not matched
Policy BGP-INBOUND, term ACCEPT-CUSTOMER: matched
    then: accept
    Action: accept

# The output shows:
# 1. Which terms were evaluated
# 2. Which term matched
# 3. What actions were applied
# 4. Final verdict (accept/reject)
```

### Testing Multi-Policy Chains

```
# When multiple policies are applied: [ POLICY-1 POLICY-2 ]
# test each individually to trace behavior

user@router> test policy POLICY-1 192.168.1.0/24

Policy POLICY-1, term ALLOW: not matched
Policy POLICY-1, term DENY: not matched
    Default action: next policy

# Then test POLICY-2 to see what happens when POLICY-1 doesn't match
user@router> test policy POLICY-2 192.168.1.0/24
```

### Common Testing Scenarios

```
# Verify redistribution policy before applying to OSPF
test policy STATIC-TO-OSPF 192.168.10.0/24 protocol static
# Expected: accept

test policy STATIC-TO-OSPF 10.0.0.0/8 protocol static
# Expected: reject (not in prefix-list)

# Verify BGP import filter
test policy BGP-INBOUND 10.0.0.0/8
# Expected: reject (RFC1918)

test policy BGP-INBOUND 198.51.100.0/24
# Expected: accept (customer prefix)

# Verify community-based policy
test policy UPSTREAM-EXPORT 198.51.100.0/24 protocol bgp
# Check if community match works as expected
```

---

## 6. Complex Multi-Term Multi-Policy Chaining Examples

### Example 1: Complete BGP Import Chain

Three policies chained: sanitize, classify, then accept with defaults.

```
# Policy chain: [ SANITIZE CLASSIFY DEFAULT-LP ]

# POLICY 1: SANITIZE -- remove known-bad routes
policy-options {
    prefix-list BOGONS {
        0.0.0.0/8;
        10.0.0.0/8;
        127.0.0.0/8;
        172.16.0.0/12;
        192.168.0.0/16;
        224.0.0.0/4;
    }
    prefix-list TOO-SPECIFIC {
        0.0.0.0/0;
    }

    policy-statement SANITIZE {
        term REJECT-BOGONS {
            from {
                prefix-list-filter BOGONS orlonger;
            }
            then reject;
            /* TERMINATES: rejected routes stop here */
        }
        term REJECT-TOO-LONG {
            from {
                route-filter 0.0.0.0/0 prefix-length-range /25-/32;
            }
            then reject;
            /* TERMINATES: /25 and longer rejected */
        }
        term PASS-CLEAN {
            then next policy;
            /* Non-matching routes continue to CLASSIFY */
        }
    }

    # POLICY 2: CLASSIFY -- tag routes with communities
    community TRANSIT members 65001:300;
    community CUSTOMER members 65001:100;
    community PEER members 65001:200;

    policy-statement CLASSIFY {
        term TAG-CUSTOMER {
            from {
                neighbor 10.1.1.1;
            }
            then {
                community add CUSTOMER;
                next policy;
                /* Tagged, continue to DEFAULT-LP */
            }
        }
        term TAG-PEER {
            from {
                neighbor 10.2.2.2;
            }
            then {
                community add PEER;
                next policy;
            }
        }
        term TAG-TRANSIT {
            then {
                community add TRANSIT;
                next policy;
                /* Catch-all: everything else is transit */
            }
        }
    }

    # POLICY 3: DEFAULT-LP -- set local-pref based on community
    policy-statement DEFAULT-LP {
        term CUSTOMER-LP {
            from community CUSTOMER;
            then {
                local-preference 300;
                accept;
            }
        }
        term PEER-LP {
            from community PEER;
            then {
                local-preference 200;
                accept;
            }
        }
        term TRANSIT-LP {
            from community TRANSIT;
            then {
                local-preference 100;
                accept;
            }
        }
        term REJECT-UNCLASSIFIED {
            then reject;
        }
    }
}

# Apply the chain
set protocols bgp group ALL-PEERS import [ SANITIZE CLASSIFY DEFAULT-LP ]
```

### Trace Through Example 1

```
Route: 198.51.100.0/24 from neighbor 10.1.1.1

Step 1: SANITIZE
  term REJECT-BOGONS    → from: prefix-list BOGONS → 198.51.100.0/24 NOT in list → NO MATCH
  term REJECT-TOO-LONG  → from: /25-/32 → /24 is not in range → NO MATCH
  term PASS-CLEAN       → no from (matches all) → then: next policy → GO TO CLASSIFY

Step 2: CLASSIFY
  term TAG-CUSTOMER     → from: neighbor 10.1.1.1 → MATCH
                        → then: community add CUSTOMER → APPLIED
                        → then: next policy → GO TO DEFAULT-LP

Step 3: DEFAULT-LP
  term CUSTOMER-LP      → from: community CUSTOMER → MATCH (added in step 2)
                        → then: local-preference 300, accept → DONE

Result: ACCEPT with local-preference 300, community 65001:100
```

### Example 2: OSPF Redistribution with Metric Control

```
policy-options {
    prefix-list CONNECTED-LANS {
        192.168.10.0/24;
        192.168.20.0/24;
        192.168.30.0/24;
    }
    prefix-list STATIC-DEFAULTS {
        0.0.0.0/0;
    }
    prefix-list STATIC-SPECIFICS {
        10.100.0.0/16;
        10.200.0.0/16;
    }

    policy-statement REDIST-TO-OSPF {
        term CONNECTED-LOW-METRIC {
            from {
                protocol direct;
                prefix-list CONNECTED-LANS;
            }
            then {
                metric 10;
                external {
                    type 1;       /* OSPF external type 1 */
                }
                accept;
            }
        }
        term DEFAULT-ROUTE {
            from {
                protocol static;
                prefix-list STATIC-DEFAULTS;
            }
            then {
                metric 100;
                external {
                    type 1;
                }
                accept;
            }
        }
        term STATIC-HIGH-METRIC {
            from {
                protocol static;
                prefix-list STATIC-SPECIFICS;
            }
            then {
                metric 500;
                external {
                    type 2;       /* OSPF external type 2 */
                }
                tag 999;
                accept;
            }
        }
        term DENY-EVERYTHING-ELSE {
            then reject;
        }
    }
}

set protocols ospf export REDIST-TO-OSPF
```

### Example 3: BGP Export with AS-Path Prepending

```
policy-options {
    community ANNOUNCE members 65001:1000;
    community NO-ANNOUNCE members 65001:0;
    community PREPEND-3X members 65001:3333;

    as-path INTERNAL "^$";

    policy-statement BGP-EXPORT {
        term BLOCK-NO-ANNOUNCE {
            from community NO-ANNOUNCE;
            then reject;
        }
        term BLOCK-INTERNAL {
            from as-path INTERNAL;
            /* Don't leak iBGP routes without community tag */
            then {
                next term;
                /* Check if it has ANNOUNCE community */
            }
        }
        term PREPEND-BACKUP {
            from community PREPEND-3X;
            then {
                as-path-prepend "65001 65001 65001";
                accept;
            }
        }
        term ANNOUNCE-TAGGED {
            from community ANNOUNCE;
            then accept;
        }
        term REJECT-DEFAULT {
            then reject;
        }
    }
}

set protocols bgp group UPSTREAM export BGP-EXPORT
```

---

## 7. Import vs Export: Effects on Routing Tables

### How Import Policies Affect the Routing Table

```
                    ┌──────────────────┐
  BGP updates ────> │  IMPORT POLICY   │ ────> inet.0 routing table
  from peer         │  (filter/modify) │       (only accepted routes
                    └──────────────────┘        are installed)

  Import policy controls:
  - WHICH routes from a protocol enter the routing table
  - Route ATTRIBUTES as they are installed (local-pref, metric, etc.)
  - Does NOT affect what the protocol itself stores internally
    (BGP RIB-In stores all received routes regardless)
```

### How Export Policies Affect Route Advertisement

```
                    ┌──────────────────┐
  inet.0 routing ─> │  EXPORT POLICY   │ ────> BGP updates to peer
  table (active      │  (filter/modify) │       (only accepted routes
   routes)          └──────────────────┘        are advertised)

  Export policy controls:
  - WHICH active routes are advertised to peers
  - Route ATTRIBUTES as they are sent (MED, AS-path, communities)
  - For OSPF: controls redistribution FROM other protocols INTO OSPF
```

### Key Distinction: Protocol-Native vs Redistributed

```
# OSPF export policy does NOT control OSPF-to-OSPF flooding
# It controls redistribution of OTHER protocols into OSPF as external LSAs

# These OSPF routes flood normally regardless of export policy:
#   - Intra-area routes (Router LSA, Network LSA)
#   - Inter-area routes (Summary LSA)

# Export policy controls THESE becoming OSPF external routes:
#   - Static routes → OSPF External LSA (Type 5/7)
#   - Connected routes → OSPF External LSA
#   - BGP routes → OSPF External LSA
```

---

## Prerequisites

- Solid understanding of IP addressing, subnetting, and CIDR notation
- Familiarity with routing protocols (OSPF, BGP, IS-IS, RIP) at a conceptual level
- Basic JunOS CLI navigation (operational mode, configuration mode, commit model)
- Understanding of the difference between the routing table (RIB) and forwarding table (FIB)
- Knowledge of BGP path attributes (local-preference, MED, AS-path, communities, origin)
- Familiarity with OSPF LSA types and route categories (intra-area, inter-area, external)
