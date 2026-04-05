# Advanced BGP Theory — Decision Process, Convergence, and Security

> *BGP is the glue of the internet, yet its convergence properties are poorly understood, its security was an afterthought, and its decision process contains subtle non-determinism that can cause persistent oscillation in production networks. This deep-dive covers the formal specification of the decision process, convergence analysis, route-reflector correctness, and the security mechanisms being deployed to protect the global routing table.*

---

## 1. BGP Decision Process — Formal Specification

The BGP decision process is a total order over the set of candidate routes for a given prefix. Given routes $R = \{r_1, r_2, \ldots, r_n\}$ for prefix $p$, the best path function $B(R)$ applies a lexicographic ordering over a tuple of attribute values.

### Formal Definition

Define the comparison tuple for route $r$:

$$T(r) = (-W(r),\ -LP(r),\ -L(r),\ |AS(r)|,\ O(r),\ M(r),\ E(r),\ IGP(r),\ A(r),\ RID(r),\ |CL(r)|,\ NA(r))$$

Where:
- $W(r)$ = Weight (Cisco-proprietary, higher preferred, negated for min-ordering)
- $LP(r)$ = LOCAL_PREF (higher preferred, negated)
- $L(r)$ = locally originated flag (1 if local, 0 otherwise, negated)
- $|AS(r)|$ = AS_PATH length (shorter preferred)
- $O(r)$ = ORIGIN code (0=IGP, 1=EGP, 2=Incomplete)
- $M(r)$ = MED value (lower preferred)
- $E(r)$ = eBGP/iBGP flag (0=eBGP, 1=iBGP)
- $IGP(r)$ = IGP metric to NEXT_HOP
- $A(r)$ = route age (negated — older preferred for eBGP)
- $RID(r)$ = neighbor router-id
- $|CL(r)|$ = CLUSTER_LIST length
- $NA(r)$ = neighbor address

The best path is:

$$B(R) = \arg\min_{r \in R} T(r) \quad \text{under lexicographic ordering}$$

### Pre-conditions

Before entering the decision process, routes must satisfy:

1. **NEXT_HOP reachability**: $\exists$ route to NEXT_HOP in RIB
2. **Synchronization**: If `synchronization` enabled, prefix must be in IGP (deprecated, rarely used)
3. **AS loop detection**: Own AS must not appear in AS_PATH (for eBGP)

### MED Comparison Scope

The MED comparison in step 7 operates conditionally:

$$M_{compare}(r_i, r_j) = \begin{cases}
\min(MED(r_i), MED(r_j)) & \text{if } AS_{neighbor}(r_i) = AS_{neighbor}(r_j) \\
\text{skip} & \text{if } AS_{neighbor}(r_i) \neq AS_{neighbor}(r_j) \text{ and not always-compare-med}
\end{cases}$$

This conditional scope is the root cause of BGP MED oscillation (see Section 3).

---

## 2. BGP Convergence Analysis — Path Exploration

### The Path Exploration Problem

When a route is withdrawn, BGP does not instantly converge. Instead, routers explore alternative paths sequentially, with each exploration generating additional UPDATE messages. This is known as **path exploration** or **path hunting**.

### Convergence Time Model

For a linear chain of $n$ ASes, the withdrawal convergence time is bounded by:

$$T_{withdraw} \leq \Delta \times \text{MRAI} \times \binom{n}{2}$$

Where:
- $\Delta$ = number of distinct path exploration phases
- $\text{MRAI}$ = Minimum Route Advertisement Interval (default 30 seconds for eBGP)
- $\binom{n}{2}$ = the number of possible path lengths to explore

In the worst case, the number of UPDATE messages generated during withdrawal convergence for $n$ ASes is:

$$U_{worst} = O(n!)$$

This factorial bound arises because each AS may explore all permutations of remaining paths before concluding the prefix is unreachable.

### Worked Example: 5-AS Linear Chain

```
AS1 ─── AS2 ─── AS3 ─── AS4 ─── AS5 (origin)

When AS5 withdraws the route:
  Phase 1: AS4 sends withdrawal to AS3
  Phase 2: AS3 still has path via AS2→AS4→AS5 (invalid, ghost path)
  Phase 3: AS3 tries AS2→AS4→AS5, AS2 tries AS3→AS4→AS5
  ...
  Multiple MRAI cycles pass before all ASes converge
```

The number of transient UPDATE messages:

| ASes ($n$) | Best case (immediate) | Worst case ($O(n!)$) | With MRAI damping |
|:---:|:---:|:---:|:---:|
| 3 | 2 | 6 | ~60s |
| 4 | 3 | 24 | ~120s |
| 5 | 4 | 120 | ~300s |
| 6 | 5 | 720 | ~600s |
| 10 | 9 | 3,628,800 | hours (theoretical) |

In practice, path exploration is bounded by topology constraints and MRAI timers, so real-world convergence for withdrawals is typically 3-15 minutes.

### Announcement vs Withdrawal Asymmetry

Route announcements converge much faster than withdrawals because each new announcement immediately replaces any inferior path. The announcement convergence time for a fully-connected topology:

$$T_{announce} = O(d) \times \text{MRAI}$$

Where $d$ is the diameter of the AS graph. This is typically 1-3 MRAI intervals (30-90 seconds).

### MRAI and Convergence Trade-off

The MRAI timer introduces a fundamental trade-off:

$$\text{Larger MRAI} \implies \begin{cases} \text{Fewer transient updates (less churn)} \\ \text{Slower convergence} \end{cases}$$

$$\text{Smaller MRAI} \implies \begin{cases} \text{More transient updates} \\ \text{Faster convergence} \end{cases}$$

The optimal MRAI value depends on network topology. Research suggests per-peer MRAI with exponential backoff improves convergence:

$$\text{MRAI}_{adaptive}(n) = \text{MRAI}_{base} \times 2^{\lfloor n/k \rfloor}$$

Where $n$ is the number of consecutive updates for the same prefix and $k$ is a tuning constant.

---

## 3. BGP Oscillation and MED Non-Determinism

### The MED Oscillation Problem

BGP's decision process can produce oscillation (no stable state) when MED is compared across routes from different neighbor ASes. This occurs because MED comparison violates the independence of irrelevant alternatives (IIA) axiom.

### Formal Statement

Given three routes $r_1, r_2, r_3$ from two neighbor ASes:

```
r1: via AS100, MED=10, LOCAL_PREF=100
r2: via AS100, MED=30, LOCAL_PREF=100
r3: via AS200, MED=20, LOCAL_PREF=100
```

With `always-compare-med`:
- $r_1$ vs $r_3$: $r_1$ wins (MED 10 < 20)
- $r_2$ vs $r_3$: $r_3$ wins (MED 20 < 30)

Without `always-compare-med` (default, MED compared only within same AS):
- $r_1$ vs $r_2$: $r_1$ wins (same AS, MED 10 < 30)
- $r_1$ vs $r_3$: Tie on MED (different AS, skip), decided by later tiebreaker
- But if $r_1$ is withdrawn, preference between $r_2$ and $r_3$ may flip

### The Route Reflector MED Oscillation

In a topology with multiple RRs and multiple exit points to the same neighbor AS:

```
        RR-1          RR-2
       /    \        /    \
     PE-A  PE-B   PE-C   PE-D
      |      |      |      |
    AS200  AS200  AS200  AS200
    MED=10 MED=30 MED=20 MED=40
```

Each RR sees different subsets of paths (RR only reflects the best from its clients). The selected best path depends on which paths each RR has visibility of, creating a circular dependency that can oscillate indefinitely.

### Deterministic MED

The `bgp deterministic-med` command groups paths by neighbor AS before comparison:

$$B(R) = \min_{AS_i} \left( \min_{r \in R_{AS_i}} M(r) \right) \quad \text{then apply remaining tiebreakers}$$

This eliminates arrival-order dependence but does not fully prevent oscillation in RR topologies.

### Griffin-Wilfong Proof

Griffin, Wilfong, and others proved that the BGP decision process with arbitrary policies is **not guaranteed to converge**. Specifically:

- The Stable Paths Problem (SPP) is the formal model of BGP convergence
- An instance of SPP has a stable solution if and only if it contains no "dispute wheel"
- Finding whether a dispute wheel exists is NP-hard in general
- MED creates implicit dispute wheels in certain topologies

---

## 4. Route Reflector Correctness

### The Correctness Problem

Route reflectors alter the iBGP topology by replacing full mesh with a hub-and-spoke model. This introduces the risk that the path chosen by the RR (from its perspective) is not the path the client would have chosen with full visibility.

### Reflection Rules

An RR operates under three rules:

1. **Client to client**: Route learned from client $C_i$ is reflected to all other clients $C_j$ and non-clients
2. **Client to non-client**: Route learned from client is advertised to non-clients
3. **Non-client to client**: Route learned from non-client is reflected to all clients
4. **Non-client to non-client**: Route is NOT reflected (requires full mesh among non-clients)

### Loop Prevention

RRs use two attributes for loop prevention:

- **ORIGINATOR_ID**: Set to the router-id of the route originator. If a router receives a route with its own ORIGINATOR_ID, it discards the route.
- **CLUSTER_LIST**: Ordered list of cluster-ids traversed. If a router receives a route with its own cluster-id in the CLUSTER_LIST, it discards the route.

### Correctness Conditions

An RR deployment produces correct routing (equivalent to full iBGP mesh) if and only if:

1. **Path visibility**: The RR has visibility of all candidate paths for every prefix. This is satisfied when:
   - All exit points for a prefix are clients of the same RR, OR
   - Multiple RRs together provide full path visibility

2. **Consistent path selection**: The RR's best path selection must match what clients would select. This requires:
   - RR is on the forwarding path (IGP metric from RR to NEXT_HOP equals the sum of IGP metrics through the RR), OR
   - ADD-PATH is used (RR advertises multiple paths, client makes its own decision)

3. **No MED-induced oscillation**: The combination of RR topology and MED comparison must not create circular dependencies

### Optimal Route Reflection (ORR)

ORR addresses condition 2 by computing best path from the client's perspective:

$$B_{ORR}(R, c) = \arg\min_{r \in R} T_c(r)$$

Where $T_c(r)$ uses the client $c$'s IGP metric to each NEXT_HOP rather than the RR's own IGP metric. This is implemented using the client's position in the IGP shortest-path tree.

---

## 5. Confederation AS_PATH Manipulation

### AS_PATH Encoding in Confederations

Confederations introduce two additional AS_PATH segment types:

| Segment Type | Value | Description |
|:---|:---:|:---|
| AS_SET | 1 | Unordered set (aggregation) |
| AS_SEQUENCE | 2 | Ordered sequence (normal) |
| AS_CONFED_SEQUENCE | 3 | Ordered sequence within confederation |
| AS_CONFED_SET | 4 | Unordered set within confederation |

### Path Length Calculation

When computing AS_PATH length for best path selection:

$$|AS(r)| = |AS\_SEQUENCE| + |AS\_SET| \quad \text{(AS\_CONFED\_* segments excluded)}$$

Confederation path segments are **not counted** in the path length calculation for the best path algorithm. This means inter-sub-AS hops within a confederation do not influence path selection relative to external paths.

### External View

When advertising routes externally, the confederation:

1. Strips all AS_CONFED_SEQUENCE and AS_CONFED_SET segments
2. Prepends the confederation identifier (public AS) as a single AS_SEQUENCE entry
3. External peers see a single AS in the path, regardless of how many sub-ASes the route traversed internally

### Forwarding Path Considerations

Within a confederation, inter-sub-AS sessions behave like eBGP:
- TTL=1 by default (must use `ebgp-multihop` or `disable-connected-check` if not directly connected)
- NEXT_HOP is modified at sub-AS boundary
- LOCAL_PREF is preserved across sub-AS boundaries (unlike true eBGP)
- MED is preserved across sub-AS boundaries

---

## 6. BGP Security — RPKI, BGPsec, and ASPA

### RPKI (Resource Public Key Infrastructure)

RPKI provides cryptographic validation of route origins through a hierarchy of certificates:

```
  IANA (Trust Anchor)
    └── RIR (ARIN, RIPE, APNIC, LACNIC, AFRINIC)
          └── ISP/LIR
                └── ROA (Route Origin Authorization)
                      Binds: (Prefix, MaxLength, Origin AS)
```

### ROA Validation Algorithm

Given a BGP route with prefix $p$ and origin AS $a$, the validation state is computed as:

$$V(p, a) = \begin{cases}
\text{Valid} & \exists\ ROA: p \subseteq ROA.prefix \land |p| \leq ROA.maxLen \land a = ROA.AS \\
\text{Invalid} & \exists\ ROA: p \subseteq ROA.prefix \land |p| \leq ROA.maxLen \land a \neq ROA.AS \\
\text{NotFound} & \nexists\ ROA: p \subseteq ROA.prefix \land |p| \leq ROA.maxLen
\end{cases}$$

Where $|p|$ is the prefix length of the announced route.

### ROA MaxLength Pitfall

Setting MaxLength too broadly weakens protection:

```
ROA: 10.0.0.0/16, MaxLength=/24, AS 65000

This authorizes AS 65000 to announce:
  10.0.0.0/16, 10.0.0.0/17, 10.0.0.0/18 ... 10.0.0.0/24
  10.0.1.0/24, 10.0.2.0/24 ... 10.0.255.0/24

If attacker announces 10.0.5.0/24 from AS 99999 → Invalid (detected)
If attacker announces 10.0.5.0/25 → NotFound (NOT detected — /25 > maxLen)
```

Best practice: set MaxLength equal to the most specific prefix you actually announce.

### BGPsec (RFC 8205)

BGPsec provides cryptographic path validation — each AS in the path signs the announcement, proving the path is authentic. However, BGPsec has significant deployment challenges:

1. **Performance**: Each UPDATE requires signature verification per AS hop
2. **Partial deployment**: BGPsec provides no security benefit until widely deployed (unlike RPKI)
3. **AS_PATH modification**: Any AS that modifies the AS_PATH (prepending, aggregation) must re-sign
4. **No protection for withdrawn routes**: BGPsec only validates announcements, not withdrawals

### ASPA (Autonomous System Provider Authorization)

ASPA is a newer mechanism (draft-ietf-sidrops-aspa-verification) that validates the AS path by declaring authorized upstream providers:

```
ASPA Object: AS 65001 has authorized providers: {AS 65000, AS 65002}

Validation:
  Path [65000, 65001, 65003] — 65000 is authorized provider of 65001 → Valid hop
  Path [65099, 65001, 65003] — 65099 is NOT authorized provider of 65001 → Invalid
```

ASPA provides route-leak detection with much simpler deployment than BGPsec.

---

## 7. BGP in Large-Scale Service Provider Networks

### Scale Considerations

The global BGP table contains approximately 1 million IPv4 prefixes and 200,000 IPv6 prefixes (as of 2025). A large SP router must:

| Resource | Requirement |
|:---|:---|
| RIB memory | ~2-4 GB for full table (IPv4+IPv6) |
| FIB memory | ~1-2 GB (compressed in hardware) |
| BGP sessions | 100-1000+ peers |
| UPDATE processing | 10,000-50,000 updates/second during convergence events |
| Convergence target | < 1 second for PIC-enabled prefixes |

### Scaling Techniques

1. **Route reflectors**: Reduce iBGP mesh from $O(n^2)$ to $O(n)$ sessions
2. **Confederations**: Partition large AS into manageable sub-ASes
3. **ORF (Outbound Route Filtering)**: Push filters to upstream to reduce inbound table size
4. **ADD-PATH**: Advertise multiple paths for multipath and fast convergence
5. **PIC**: Decouple convergence time from prefix count
6. **BGP-LS + SR-TE**: Offload traffic engineering to centralized controller
7. **Aggregation**: Summarize customer prefixes where possible

### Internet Exchange Point (IXP) Design

At IXPs, BGP route servers simplify multilateral peering:

```
  Without route server: n peers → n(n-1)/2 bilateral sessions
  With route server:    n peers → n sessions to route server

  Route server rules:
    - Transparent AS_PATH (does not insert own AS)
    - Per-peer import/export filtering
    - Often uses ADD-PATH to pass multiple options to peers
```

---

## 8. BGP Graceful Restart Theory (RFC 4724)

### Mechanism

Graceful Restart (GR) allows a router to restart its BGP process while the forwarding plane continues using stale routes. The protocol operates in three phases:

**Phase 1 — Negotiation** (during OPEN):
- GR capability advertised with restart time and per-AFI forwarding state bit
- Both peers must support GR for it to function

**Phase 2 — Restart event**:
- Restarting router's BGP process goes down
- Helper peer detects session drop but does NOT remove routes
- Helper marks all routes from restarting peer as "stale"
- Helper starts the restart timer (default 120 seconds)

**Phase 3 — Re-establishment**:
- Restarting router comes back, opens new BGP session
- Sends End-of-RIB (EoR) marker after sending all routes
- Helper removes stale routes not refreshed after EoR
- If restart timer expires before session re-establishes, all stale routes are purged

### Timer Interactions

$$T_{stale} = \min(T_{stalepath},\ T_{restart} + T_{defer})$$

Where:
- $T_{stalepath}$ = stalepath timer (how long to keep stale routes after EoR, default 360s)
- $T_{restart}$ = restart timer (max time to wait for session re-establishment, default 120s)
- $T_{defer}$ = defer timer (time to wait for EoR after session is up)

### Long-Lived Graceful Restart (LLGR — RFC 9494)

LLGR extends GR with a second phase where stale routes are kept for much longer (hours/days) but with reduced priority:

$$LP_{LLGR}(r) = 0 \quad \text{(stale routes get LOCAL_PREF 0 during LLGR phase)}$$

This ensures LLGR routes are only used as last resort, preventing traffic black-holes while maintaining connectivity during extended outages.

---

## 9. BGP-LS NLRI Format (RFC 7752)

### NLRI Structure

BGP-LS distributes link-state topology information using a new address family (AFI=16388, SAFI=71). The NLRI encodes three types of objects:

### Node NLRI

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|            Type (1)           |            Length              |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|  Protocol-ID  |   Identifier (64-bit)                         |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                  Local Node Descriptors (TLV)                 |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

Protocol-ID values:
  1 = IS-IS Level 1    2 = IS-IS Level 2    3 = OSPF
  4 = Direct           5 = Static           6 = OSPFv3
  7 = BGP
```

### Link NLRI

```
Fields:
  - Local Node Descriptors (router advertising the link)
  - Remote Node Descriptors (router at the other end)
  - Link Descriptors: Interface IP, Neighbor IP, Multi-Topology ID

Link Attributes (carried in BGP Path Attributes):
  - TE Default Metric, IGP Metric
  - Max Link Bandwidth, Max Reservable Bandwidth
  - Unreserved Bandwidth (8 priority levels)
  - Admin Group (link colors for TE)
  - SRLG (Shared Risk Link Group)
  - Adjacency SID (Segment Routing)
  - Unidirectional Link Delay, Delay Variation, Loss, Residual BW
```

### Prefix NLRI

```
Fields:
  - Local Node Descriptors
  - Prefix Descriptors: IP Reachability (prefix + length), Multi-Topology ID

Prefix Attributes:
  - Prefix Metric
  - Prefix SID (Segment Routing)
  - Flags (re-advertisement, node SID, no-PHP, explicit-null)
  - Algorithm (shortest path, strict shortest path)
```

### Use Cases

BGP-LS enables centralized applications to consume the complete network topology:

1. **SR-TE controllers**: Compute explicit paths using real-time topology and TE metrics
2. **Network visualization**: Build live topology maps with link utilization
3. **Traffic engineering**: Optimize traffic placement across the network
4. **Fast reroute planning**: Pre-compute backup paths at the controller level

---

## Prerequisites

- bgp, ospf, is-is, mpls, mpls-vpn, segment-routing, rpki, graph theory, combinatorics

## Complexity

| Operation | Time | Space |
|-----------|------|-------|
| Best path selection (per prefix) | O(k) where k=attributes | O(1) |
| Path exploration (worst case) | O(n!) where n=ASes | O(n) |
| RPKI validation (per prefix) | O(log r) where r=ROAs | O(r) |
| RR reflection (per update) | O(c) where c=clients | O(1) |
| BGP-LS topology build | O(V + E) | O(V + E) |
| Convergence with PIC | O(1) per prefix | O(n) backup paths |

## References

- [RFC 4271 — BGP-4 Specification](https://www.rfc-editor.org/rfc/rfc4271)
- [RFC 4456 — BGP Route Reflection](https://www.rfc-editor.org/rfc/rfc4456)
- [RFC 5065 — Autonomous System Confederations for BGP](https://www.rfc-editor.org/rfc/rfc5065)
- [RFC 4724 — Graceful Restart Mechanism for BGP](https://www.rfc-editor.org/rfc/rfc4724)
- [RFC 9494 — Long-Lived Graceful Restart for BGP](https://www.rfc-editor.org/rfc/rfc9494)
- [RFC 7911 — Advertisement of Multiple Paths in BGP](https://www.rfc-editor.org/rfc/rfc7911)
- [RFC 5575 — Dissemination of Flow Specification Rules](https://www.rfc-editor.org/rfc/rfc5575)
- [RFC 7752 — North-Bound Distribution of Link-State and TE Information Using BGP](https://www.rfc-editor.org/rfc/rfc7752)
- [RFC 6811 — BGP Prefix Origin Validation](https://www.rfc-editor.org/rfc/rfc6811)
- [RFC 8205 — BGPsec Protocol Specification](https://www.rfc-editor.org/rfc/rfc8205)
- [RFC 8326 — Graceful BGP Session Shutdown](https://www.rfc-editor.org/rfc/rfc8326)
- [RFC 2439 — BGP Route Flap Damping](https://www.rfc-editor.org/rfc/rfc2439)
- [RFC 8092 — BGP Large Communities](https://www.rfc-editor.org/rfc/rfc8092)
- [Griffin & Wilfong — "An Analysis of BGP Convergence Properties" (SIGCOMM 1999)](https://dl.acm.org/doi/10.1145/316188.316231)
- [Labovitz et al. — "Delayed Internet Routing Convergence" (SIGCOMM 2000)](https://dl.acm.org/doi/10.1145/347059.347428)

---

*The internet runs on a protocol that is not guaranteed to converge, whose security was bolted on decades after deployment, and whose decision process can oscillate indefinitely under specific topologies. Understanding these failure modes is not academic — it is the difference between a 5-minute outage and a 5-hour one.*
