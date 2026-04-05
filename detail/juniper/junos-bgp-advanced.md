# JunOS BGP Advanced — Implementation Internals, Path Selection Analysis, and SP Design

> *JunOS implements BGP with a routing table architecture centered on bgp.l3vpn.0 for VPN routes and inet.3 for labeled transport. Routing policy evaluation occurs at import (pre-RIB) and export (post-RIB), with policy chains evaluated sequentially. Understanding the JunOS-specific path selection algorithm, RIB structure, and policy evaluation chain is critical for designing scalable SP BGP architectures.*

---

## 1. JunOS BGP Implementation Architecture

### Routing Table Structure

JunOS maintains multiple routing tables (RIBs), and BGP interacts with several:

```
┌─────────────────────────────────────────────────────────────┐
│                    JunOS Routing Tables                      │
│                                                             │
│  inet.0          Primary IPv4 unicast RIB                   │
│  inet.3          MPLS next-hop resolution (LDP, RSVP, SR)   │
│  inet6.0         Primary IPv6 unicast RIB                   │
│  bgp.l3vpn.0     L3VPN routes (before RT import)            │
│  <VRF>.inet.0    Per-VRF IPv4 table (after RT import)       │
│  inetflow.0      Flowspec routes                            │
│  mpls.0          MPLS label switching table                  │
│  inet.2          Multicast RPF table                        │
│  lsdist.0        BGP-LS link-state database                 │
└─────────────────────────────────────────────────────────────┘
```

### BGP Route Processing Pipeline

```
Received BGP UPDATE
│
├─ 1. Parse and validate UPDATE message
│     Detect malformed attributes (RFC 7606 handling)
│
├─ 2. Store in Adj-RIB-In (per-peer received routes)
│
├─ 3. Apply IMPORT routing policy
│     Policy can: modify attributes, accept, reject, or next-policy
│     Result: routes accepted into Loc-RIB or rejected
│
├─ 4. Install in appropriate RIB
│     inet-unicast → inet.0
│     inet-vpn → bgp.l3vpn.0 → VRF tables (via RT matching)
│     labeled-unicast → inet.3
│     flowspec → inetflow.0
│
├─ 5. Best path selection (per prefix)
│     Compare all received paths using JunOS selection algorithm
│     Winner installed as active route
│
├─ 6. FIB push (active routes → PFE forwarding table)
│
├─ 7. Apply EXPORT routing policy (for each peer)
│     Policy determines which routes to advertise
│     Can modify attributes per-peer
│
└─ 8. Send BGP UPDATE to peers (Adj-RIB-Out)
```

### Import vs Export Policy Timing

A critical JunOS detail: **import policy runs BEFORE best path selection**. This means attribute modifications in import policy (LP, MED, AS-path prepend) directly influence which path wins:

```
Route A received from Peer 1: LP 100 (default)
Route A received from Peer 2: LP 100 (default)

Import policy on Peer 1: set LP 200
Import policy on Peer 2: (no change, LP stays 100)

After import:
  Route A from Peer 1: LP 200 (modified by import policy)
  Route A from Peer 2: LP 100

Best path selection:
  LP comparison: 200 > 100 → Peer 1 wins
```

---

## 2. Path Selection Algorithm — JunOS vs Cisco

### JunOS Best Path Selection (Detailed)

```
Step  Criterion                    JunOS Behavior
─────────────────────────────────────────────────────────────────
 1    Local preference             Highest wins (default 100)
 2    AS path length               Shortest wins
 3    Origin                       IGP < EGP < INCOMPLETE
 4    MED                          Lowest wins (within same neighbor-AS)
 5    EBGP > IBGP                  External preferred
 6    IGP metric to next-hop       Lowest wins (hot-potato)
 7    Active route preferred       Routes with resolved next-hops
 8    Cluster-list length          Shortest wins (RR loop avoidance)
 9    Router-ID                    Lowest wins (originator-id for reflected)
 10   Peer IP address              Lowest wins (final tiebreaker)
```

### Key Differences from Cisco IOS/XR

```
                              JunOS                    Cisco IOS/XR
─────────────────────────────────────────────────────────────────────
MED comparison:               Same neighbor-AS only    Same neighbor-AS only
                              (deterministic)          (non-deterministic default)
                              Groups by AS, best       Compare in arrival order
                              within group first

MED always-compare:           path-selection           bgp always-compare-med
                              always-compare-med

Missing MED treatment:        MED = 0 (lowest)         MED = 0 (or infinity
                                                       with bestpath missing)

Router-ID tiebreak:           originator-id used       router-id used
                              (for reflected routes)    (originator-id only
                                                       if identical)

Oldest path preference:       Not a default step       Step 7 in Cisco
                              (no "oldest" preference)  (prefer older EBGP path)

Multipath:                    `multipath` knob         `maximum-paths` knob
                              per-group                per-AF under router bgp
```

### Deterministic MED in JunOS

JunOS uses deterministic MED by default. The algorithm:

```
1. Group all paths for a prefix by neighboring AS
2. Within each AS group, select the best path (using MED + other criteria)
3. Compare the winners from each AS group (WITHOUT MED — since different AS)
4. Select the overall best path

Example:
  Path A: AS 65001, MED 50, LP 100
  Path B: AS 65001, MED 100, LP 100
  Path C: AS 65002, MED 10, LP 100
  Path D: AS 65002, MED 200, LP 100

Step 1: Group by AS
  AS 65001: {A(MED 50), B(MED 100)}
  AS 65002: {C(MED 10), D(MED 200)}

Step 2: Within-group best (MED comparison valid)
  AS 65001 winner: A (MED 50 < 100)
  AS 65002 winner: C (MED 10 < 200)

Step 3: Cross-group comparison (NO MED comparison)
  A vs C: LP equal → AS-path length equal → origin equal →
          EBGP/IBGP equal → IGP metric → ... → router-id
```

---

## 3. Routing Policy Evaluation Chain

### Policy Chain Evaluation Logic

```
Multiple policies applied:
  import [ POLICY-A POLICY-B POLICY-C ]

Evaluation:
  Route enters POLICY-A
  ├─ Term matches with "accept" → ACCEPT (stop evaluation)
  ├─ Term matches with "reject" → REJECT (stop evaluation)
  ├─ Term matches with "next term" → continue to next term in POLICY-A
  ├─ Term matches with "next policy" → skip to POLICY-B
  └─ No term matches → fall through to POLICY-B

  Route enters POLICY-B
  ├─ Same logic...
  └─ No term matches → fall through to POLICY-C

  Route enters POLICY-C
  ├─ Same logic...
  └─ No term matches → DEFAULT ACTION

Default action (when no policy matches):
  BGP import: accept
  BGP export: reject (IBGP clients) / accept (EBGP)
  Protocol import (OSPF/IS-IS): accept
  Forwarding-table export: accept
```

### Policy Subroutines

```
# Subroutine: a policy called from within another policy
set policy-options policy-statement MAIN term CHECK from policy SUBROUTINE
set policy-options policy-statement MAIN term CHECK then accept

# Subroutine behavior:
# - "accept" in subroutine → returns TRUE to calling term's "from"
# - "reject" in subroutine → returns FALSE to calling term's "from"
# - Attribute modifications in subroutine ARE applied (side effects persist)

# Example:
set policy-options policy-statement IS-CUSTOMER term 1 from community CUSTOMER
set policy-options policy-statement IS-CUSTOMER term 1 then accept

set policy-options policy-statement MAIN term CUST from policy IS-CUSTOMER
set policy-options policy-statement MAIN term CUST then local-preference 200
# If IS-CUSTOMER returns true (route has CUSTOMER community),
# MAIN term CUST sets LP 200 and accepts
```

### Common Policy Patterns

```
# Pattern 1: Prefer customer over peer over transit
set policy-options policy-statement IMPORT-CUSTOMER term SET-LP then local-preference 200
set policy-options policy-statement IMPORT-CUSTOMER term SET-LP then accept

set policy-options policy-statement IMPORT-PEER term SET-LP then local-preference 100
set policy-options policy-statement IMPORT-PEER term SET-LP then accept

set policy-options policy-statement IMPORT-TRANSIT term SET-LP then local-preference 50
set policy-options policy-statement IMPORT-TRANSIT term SET-LP then accept

set protocols bgp group CUSTOMERS import IMPORT-CUSTOMER
set protocols bgp group PEERS import IMPORT-PEER
set protocols bgp group TRANSIT import IMPORT-TRANSIT

# Pattern 2: Conditional AS-path prepend
set policy-options policy-statement PREPEND-TO-PEER term BACKUP from community BACKUP-PATH
set policy-options policy-statement PREPEND-TO-PEER term BACKUP then as-path-prepend "65000 65000 65000"
set policy-options policy-statement PREPEND-TO-PEER term BACKUP then accept
set policy-options policy-statement PREPEND-TO-PEER term NORMAL then accept
```

---

## 4. Route Reflector Design for Scale

### Hierarchical RR Architecture

```
Tier 1: Core RRs (redundant pair)
  RR1-A, RR1-B (cluster 1.1.1.1)
  Clients: Tier 2 RRs + core routers

Tier 2: Regional RRs (redundant pair per region)
  RR2-EAST-A, RR2-EAST-B (cluster 2.1.1.1)
  RR2-WEST-A, RR2-WEST-B (cluster 2.2.2.2)
  Clients: PE routers in region

          RR1-A ←──→ RR1-B           (Tier 1: IBGP full mesh)
         ╱    ╲      ╱    ╲
        ╱      ╲    ╱      ╲
  RR2-E-A  RR2-E-B  RR2-W-A  RR2-W-B  (Tier 2: clients of Tier 1)
    │╲       │╲       │╲       │╲
   PE1 PE2  PE3 PE4  PE5 PE6  PE7 PE8   (PEs: clients of Tier 2)
```

### RR Scaling Considerations

```
Single RR pair:
  Sessions: N clients * 2 RRs = 2N sessions
  Routes per RR: sum of all client routes * address families
  Suitable: < 100 clients, < 500K routes

Hierarchical RR:
  Tier 1 sessions: R regional-RR-pairs * 2 + 1 (mutual) = small
  Tier 2 sessions: C clients-per-region * 2 = moderate per RR
  Route distribution: only best routes propagate up tiers
  Suitable: 100-10,000 clients, millions of routes

Key constraints:
  - Each RR must process all UPDATE messages from all clients
  - RR CPU = f(update_rate * number_of_clients)
  - RR memory = f(total_routes * path_attributes_size)
  - Place RRs on dedicated hardware (not in data path)
  - Cluster-list prevents loops but adds processing overhead
```

### Out-of-Band RR

```
# RR not in the forwarding path — dedicated control-plane device
# Benefits:
#   - No impact on data plane if RR is overloaded
#   - Can use lower-performance hardware (virtual RR)
#   - Simplifies upgrades (take RR offline without traffic impact)
#
# Requirement:
#   - RR sets next-hop to originator (not itself) — this is BGP default
#   - Clients must have IGP reachability to route originators
#   - RR does NOT need to be in the forwarding path
```

---

## 5. BGP-LU for Seamless MPLS

### Seamless MPLS Architecture

BGP-LU (labeled-unicast) enables end-to-end MPLS label switching across multiple IGP domains:

```
Access Domain      Aggregation Domain      Core Domain
(IGP area 1)       (IGP area 2)            (IGP area 0)

CE ── PE1 ── AG1 ── AG2 ── ABR1 ── P1 ── P2 ── ABR2 ── AG3 ── PE2 ── CE

IGP:   IS-IS L1    IS-IS L2               IS-IS L2    IS-IS L1
MPLS:  LDP/SR      LDP/SR                 LDP/SR      LDP/SR
BGP-LU: ─────────────────────────────────────────────────────────
        PE1 advertises loopback via BGP-LU with label
        Each domain stitches labels: IGP label + BGP-LU label
```

### How BGP-LU Works

```
1. PE1 advertises its loopback (10.255.0.1/32) via BGP-LU with label L1
2. AG1 receives BGP-LU route, installs in inet.3
3. ABR1 receives BGP-LU route, allocates new label L2, re-advertises
4. P1 receives BGP-LU route, resolves next-hop via IGP
5. ABR2 receives, allocates L3, re-advertises to AG3
6. PE2 receives BGP-LU route for PE1's loopback

Label stack at PE2 for traffic to PE1:
  [IGP label to ABR2] [BGP-LU label L3] → ABR2
  ABR2 swaps L3 → L2 → [IGP label to ABR1] [L2] → ABR1
  ABR1 swaps L2 → L1 → [IGP label to PE1] [L1] → PE1
  PE1 pops L1 → IP lookup or VPN lookup
```

### inet.3 Resolution

```
BGP-LU routes install in inet.3 by default
inet.3 is used for MPLS next-hop resolution:
  - BGP VPN next-hops resolve via inet.3
  - LDP routes populate inet.3
  - RSVP LSPs populate inet.3
  - SR routes populate inet.3

Resolution chain:
  bgp.l3vpn.0 route → next-hop = PE loopback
  PE loopback resolved via → inet.3 (BGP-LU, LDP, or SR label)
  inet.3 route → next-hop = IGP neighbor
  IGP neighbor resolved via → inet.0 (direct route)
```

---

## 6. RPKI Implementation

### RPKI Validation Architecture

```
ROA (Route Origin Authorization):
  Prefix: 10.0.0.0/8
  Origin AS: 65000
  Max-length: /24

Validation states:
  VALID:   Prefix + origin AS match a ROA, prefix length <= max-length
  INVALID: Prefix matches a ROA, but origin AS differs OR length > max-length
  UNKNOWN: No ROA covers this prefix (not registered in RPKI)

JunOS validation flow:
  1. BGP session receives route: 10.0.0.0/8 from AS 65000
  2. rpd queries local VRP (Validated ROA Payload) cache
  3. VRP cache maintained via RTR protocol to RPKI validator
  4. Validation state tagged on route (informational attribute)
  5. Routing policy acts on validation-state (accept/reject/modify LP)
```

### RTR Protocol

```
JunOS ←── RTR (RFC 8210) ──→ RPKI Validator ←── rsync/RRDP ──→ RPKI repositories
                                (rpki-client,
                                 Routinator,
                                 Fort, OctoRPKI)

RTR protocol:
  - TCP connection (default port 8282 or 8323 with TLS)
  - Validator pushes VRP updates incrementally (Serial Notify)
  - JunOS sends Serial Query / Reset Query
  - Cache refreshed periodically (refresh-time)
  - Failover to backup validator if primary disconnects
```

---

## 7. LLGR Theory and Use Cases

### Standard GR vs LLGR

```
Standard Graceful Restart (RFC 4724):
  - Restart time: typically 120-300 seconds
  - Stale routes kept during restart window
  - If restart-time expires: all stale routes deleted
  - Designed for: RE switchover, software restart

Long-Lived Graceful Restart (RFC 9494):
  - LLGR time: hours to days (typically 86400 seconds = 24 hours)
  - After GR restart-time expires, routes enter LLGR stale state
  - LLGR stale routes tagged with LLGR_STALE community
  - Routes de-preferenced (lowest priority) but still usable
  - Designed for: persistent failures, WAN outages, maintenance windows
```

### LLGR Timeline

```
T=0:    BGP session drops
T=0-120s: Standard GR in effect
          Stale routes maintained at current priority
          Peer expects session re-establishment

T=120s: GR restart-time expires
        Without LLGR: all stale routes deleted → traffic impact
        With LLGR: routes transition to LLGR stale state
        LLGR_STALE community added
        Routes de-preferenced (last resort)

T=120s-86400s: LLGR in effect
               Routes available as backup if no better path exists
               If session re-establishes: routes refreshed, LLGR_STALE removed

T=86400s: LLGR time expires
          All stale routes finally deleted
```

### LLGR Use Cases

```
1. Remote site with single WAN link:
   WAN outage → BGP session drops → LLGR keeps VPN routes
   If backup path exists: traffic uses backup (LLGR stale = lowest priority)
   If no backup: LLGR stale route is ONLY path → traffic continues

2. Planned maintenance window:
   Take PE offline for 4 hours → LLGR retains routes for 24 hours
   No need to drain traffic before maintenance
   Routes de-preferenced, backup paths used if available

3. Intermittent connectivity:
   Flapping link → LLGR prevents repeated route withdrawal/readvertisement
   Dampening effect without BGP dampening penalties
```

---

## 8. BGP in JunOS — Operational Nuances

### Hidden Routes

```
JunOS concept: a route is "hidden" when it fails validation checks:
  - Next-hop unreachable (no route in inet.3 or inet.0)
  - Import policy rejects it (but it's still in Adj-RIB-In)
  - Protocol preference makes it inactive
  - Route is dampened

show route protocol bgp hidden           # display hidden routes
show route 10.0.0.0/8 all hidden         # all paths including hidden
```

### Route Resolution

```
BGP next-hop resolution order:
  1. Direct routes (connected next-hop)
  2. inet.3 (MPLS transport — LDP, RSVP, SR, BGP-LU)
  3. inet.0 (IGP routes)

For VPN routes (bgp.l3vpn.0):
  Next-hop resolved via inet.3 ONLY (by default)
  If not in inet.3: route is hidden

Override with:
  set routing-options resolution rib bgp.l3vpn.0 resolution-ribs inet.0
  (allows VPN next-hops to resolve via inet.0 — useful for lab/testing)
```

### RIB Groups and Route Leaking

```
# Leak BGP routes from inet.0 to inet.3 (for BGP next-hop resolution)
set routing-options rib-groups LEAK-TO-INET3 import-rib [ inet.0 inet.3 ]
set protocols bgp group IBGP family inet unicast rib-group LEAK-TO-INET3
```

## Prerequisites

- BGP fundamentals (UPDATE messages, path attributes, FSM), MPLS label switching, routing table concepts, regular expressions for AS-path filters

## Complexity

| Operation | Time | Space |
|-----------|------|-------|
| Best path selection (per prefix) | O(paths) | O(1) |
| Policy evaluation (per route) | O(terms * policies) | O(1) |
| RIB installation | O(log N) per route (radix tree) | O(routes * attributes) |
| RPKI validation lookup | O(log V) per prefix (V = VRP entries) | O(VRP_entries) |
| Route reflection (per UPDATE) | O(clients) | O(1) per reflection |

---

*BGP in JunOS is not just a routing protocol — it is a programmable route distribution engine. The policy framework turns BGP into a tool for implementing business logic: customer preference hierarchies, traffic engineering, security filtering, and inter-domain service delivery. Mastering the interaction between import policy, path selection, and export policy is the difference between a network that works and one that scales.*
