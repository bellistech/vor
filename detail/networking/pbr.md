# PBR — Policy-Based Routing Architecture and Design

> *Policy-based routing inverts the fundamental assumption of IP forwarding: that the destination address alone determines the path. By injecting administrator intent into the forwarding decision, PBR transforms routers from passive shortest-path followers into active traffic-engineering tools — at the cost of complexity, opacity, and debugging difficulty.*

---

## 1. The Forwarding Decision Model

### Destination-Based Forwarding

In standard IP forwarding, the router performs a single lookup:

```
packet arrives → extract destination IP → longest prefix match in FIB → forward out interface
```

This model has exactly one degree of freedom: the destination address. Every packet to the same destination takes the same path regardless of source, protocol, application, or packet size.

The Longest Prefix Match (LPM) algorithm:

$$\text{match}(D) = \arg\max_{p \in \text{FIB}} \{ \text{prefix\_length}(p) \mid D \in p \}$$

Where $D$ is the destination address and $p$ is a prefix in the FIB. The prefix with the longest match wins.

### Policy-Based Forwarding

PBR adds additional dimensions to the forwarding decision:

```
packet arrives → evaluate policy (route-map) → if match:
    apply set actions (override forwarding)
  else:
    fall through to normal FIB lookup
```

The PBR decision function can be modeled as:

$$\text{forward}(pkt) = \begin{cases} \text{action}(R_i) & \text{if } \exists R_i \in \text{route-map} : \text{match}(R_i, pkt) = \text{true} \\ \text{FIB\_lookup}(pkt.dst) & \text{otherwise} \end{cases}$$

Where $R_i$ is the $i$-th sequence in the route-map, evaluated in order. The first matching sequence determines the action.

### Degrees of Freedom

| Forwarding Model | Input Dimensions | Decision Basis |
|:---|:---:|:---|
| Destination-based | 1 | Destination IP only |
| PBR with source match | 2 | Source + Destination |
| PBR with protocol match | 3+ | Source + Destination + Protocol + Ports |
| PBR with length match | 4+ | Source + Destination + Protocol + Packet size |
| PBR with DSCP match | 5+ | All above + QoS marking |

Each additional match criterion adds a dimension to the classification space, increasing both granularity and complexity.

---

## 2. Route-Map Evaluation Theory

### Sequential Evaluation Model

A route-map is an ordered list of clauses, each with a sequence number, action (permit/deny), match conditions, and set actions:

$$\text{RouteMap} = [(seq_1, action_1, M_1, S_1), (seq_2, action_2, M_2, S_2), \ldots, (seq_n, action_n, M_n, S_n)]$$

Where:
- $seq_i$ is the sequence number (determines evaluation order)
- $action_i \in \{\text{permit}, \text{deny}\}$
- $M_i$ is the set of match conditions (conjunctive — all must match)
- $S_i$ is the set of set actions (applied if all matches succeed)

Evaluation proceeds:

```
for each clause (seq_i, action_i, M_i, S_i) in ascending seq order:
    if all conditions in M_i match the packet:
        if action_i == permit:
            apply all actions in S_i
            return POLICY_ROUTED
        else (deny):
            return NORMAL_ROUTING
return NORMAL_ROUTING  (implicit deny = normal routing, not drop)
```

### Match Condition Logic

Within a single route-map clause, multiple match statements interact as follows:

- **Multiple match statements of different types** = logical AND
  - `match ip address ACL1` AND `match length 0 500` means both must be true
- **Multiple values in one match statement** = logical OR
  - `match ip address ACL1 ACL2` means ACL1 OR ACL2

This creates a Conjunctive Normal Form (CNF) within each clause:

$$M_i = \bigwedge_{t \in \text{types}} \left( \bigvee_{v \in \text{values}(t)} \text{match}(t, v, pkt) \right)$$

### Clause Count and Complexity

For $n$ distinct traffic classes requiring different forwarding, you need at minimum $n$ route-map clauses. The total classification time is:

$$T_{\text{classify}} = O(n \times c)$$

Where $n$ is the number of clauses and $c$ is the cost of evaluating one clause (ACL lookup, length comparison, etc.). In software-based PBR, this is linear. In hardware (TCAM), it is $O(1)$ but consumes TCAM entries.

---

## 3. Next-Hop Selection and Reachability

### Next-Hop Resolution

When PBR specifies `set ip next-hop`, the router must verify the next-hop is reachable:

1. **Directly connected check:** Is the next-hop on a directly connected subnet?
2. **ARP resolution:** Does an ARP entry exist (or can one be created)?
3. **Interface state:** Is the output interface up/up?

If any check fails, the behavior depends on the set variant:

| Set Action | Next-Hop Unreachable Behavior |
|:---|:---|
| `set ip next-hop` | Falls back to normal routing |
| `set ip next-hop verify-availability ... track` | Uses next entry in list, then normal routing |
| `set ip default next-hop` | Normal routing (only applies when no route exists) |
| `set interface` | Depends on interface type; may blackhole |

### Verify-Availability with Track Objects

The `verify-availability` mechanism creates a conditional next-hop chain:

```
set ip next-hop verify-availability NH1 seq1 track T1
set ip next-hop verify-availability NH2 seq2 track T2
```

The selection algorithm:

$$\text{next\_hop} = \begin{cases} NH_1 & \text{if track}(T_1) = \text{UP} \\ NH_2 & \text{if track}(T_1) = \text{DOWN} \wedge \text{track}(T_2) = \text{UP} \\ \text{FIB}(pkt.dst) & \text{if all tracks DOWN} \end{cases}$$

### IP SLA Probe Theory

IP SLA probes measure reachability and performance:

$$\text{reachable}(target) = \begin{cases} \text{UP} & \text{if } \text{RTT}(target) < \text{timeout} \text{ for } k \text{ of last } n \text{ probes} \\ \text{DOWN} & \text{otherwise} \end{cases}$$

The track object uses a state machine with configurable thresholds:

- **delay-up:** Time after probe success before declaring UP (dampens flapping)
- **delay-down:** Time after probe failure before declaring DOWN

State transition timing:

$$T_{\text{detect}} = T_{\text{failure}} + T_{\text{delay-down}}$$

$$T_{\text{recover}} = T_{\text{success}} + T_{\text{delay-up}}$$

Where $T_{\text{failure}}$ is the time for the SLA probe to detect failure (depends on frequency and timeout settings).

---

## 4. Hardware vs Software PBR

### Software-Based PBR (Process Switching)

On older platforms, PBR is evaluated in the router's CPU:

```
Packet → Input Interface → PBR Route-Map Evaluation (CPU) → Forwarding Decision → Output Interface
```

Performance impact:

$$\text{PPS}_{\text{PBR}} = \frac{\text{CPU\_cycles\_available}}{\text{cycles\_per\_packet\_with\_PBR}}$$

Software PBR can reduce forwarding performance by 50-90% compared to hardware-switched traffic. The impact scales linearly with the number of route-map clauses and ACL entries.

### Hardware-Based PBR (TCAM)

Modern platforms (Catalyst 9000, Nexus 9000, ISR 4000) program PBR into TCAM:

```
Packet → Input Interface → TCAM Lookup (wire speed) → Forwarding Decision → Output Interface
```

TCAM PBR characteristics:

- Wire-speed forwarding regardless of policy complexity
- Consumes TCAM entries (finite resource)
- TCAM usage per PBR entry: typically 1-2 TCAM entries per ACE in the match ACL

TCAM capacity estimation:

$$\text{TCAM}_{\text{PBR}} = \sum_{i=1}^{n} \text{ACE\_count}(M_i) \times \text{entries\_per\_ACE}$$

Where $n$ is the number of route-map clauses and $M_i$ is the match ACL for clause $i$.

### Platform TCAM Budgets

| Platform | Total TCAM | Typical PBR Allocation | PBR Entries |
|:---|:---:|:---:|:---:|
| Catalyst 9300 | 3,072 | 512 | ~500 ACEs |
| Catalyst 9500 | 5,120 | 1,024 | ~1,000 ACEs |
| Nexus 9300 | 8,192 | 2,048 | ~2,000 ACEs |
| ISR 4431 | Software | N/A | CPU-limited |

---

## 5. Linux PBR Architecture — RPDB

### Routing Policy Database

Linux implements PBR through the Routing Policy Database (RPDB), a fundamentally different model from IOS route-maps:

```
Packet → ip rule evaluation (priority-ordered) → selected routing table → FIB lookup → forward
```

The RPDB is a list of rules, each with:
- **Priority:** 0-32767 (lower = evaluated first)
- **Selector:** match condition (from, to, fwmark, iif, oif, tos, ipproto)
- **Action:** lookup table, unreachable, blackhole, prohibit

### Rule Evaluation

$$\text{table}(pkt) = \text{action}(R_j) \text{ where } j = \min\{i \mid \text{selector}(R_i) \text{ matches } pkt\}$$

Rules are evaluated in priority order. The first matching rule determines which routing table is used for the FIB lookup.

### Default Rules

Linux ships with three pre-installed rules:

| Priority | Selector | Action | Purpose |
|:---:|:---|:---|:---|
| 0 | from all | lookup local | Loopback and local addresses |
| 32766 | from all | lookup main | Normal routing table |
| 32767 | from all | lookup default | Fallback (usually empty) |

### Routing Table Isolation

Each routing table is an independent FIB:

$$\text{FIB}_t = \{(prefix_1, nexthop_1), (prefix_2, nexthop_2), \ldots\}$$

Tables are identified by number (0-255) or name (mapped in `/etc/iproute2/rt_tables`). Key tables:

- **local (255):** Auto-populated with local and broadcast addresses
- **main (254):** Default table used by `ip route add` without `table`
- **default (253):** Fallback (empty by default)
- **0 (unspec):** Special — operations apply to all tables

### fwmark-Based Classification

The fwmark mechanism connects packet classification (netfilter/iptables/nftables) to routing policy:

```
Netfilter PREROUTING → set fwmark → ip rule match fwmark → routing table → FIB lookup
```

fwmark is a 32-bit integer stored in the kernel socket buffer (skb->mark). It exists only in kernel memory and is never transmitted on the wire.

The fwmark can be split into fields using masking:

$$\text{match} = (\text{skb.mark} \mathbin{\&} \text{mask}) == \text{value}$$

Example: Use bits 0-7 for PBR table selection, bits 8-15 for QoS class:

```bash
# Match only lower 8 bits for routing
ip rule add fwmark 0x01/0xff table ISP1
ip rule add fwmark 0x02/0xff table ISP2

# Match bits 8-15 for QoS (separate rules or tc)
```

---

## 6. PBR Failure Modes and Risks

### Asymmetric Routing

PBR often creates asymmetric paths where forward and return traffic take different routes:

```
Client → [ISP1 via PBR] → Server
Server → [ISP2 via normal routing] → Client
```

Consequences:
- Stateful firewalls drop return traffic (no matching session)
- TCP performance degrades (different RTT per direction)
- ECMP hash may cause packet reordering within flows
- uRPF (unicast Reverse Path Forwarding) may drop legitimate traffic

Mitigation:

$$\text{symmetric} \iff \text{PBR}(\text{forward}) \wedge \text{PBR}(\text{reverse}) \wedge \text{consistent\_path}$$

Both directions must be policy-routed, or stateful devices must be placed at the convergence point.

### PBR Blackholes

If `set interface` points to a point-to-multipoint interface without a valid next-hop, traffic is silently dropped:

```
set interface GigabitEthernet0/1    ← Ethernet segment, no next-hop specified
```

The router does not know which host on the segment to ARP for. On point-to-point interfaces (serial, tunnel), this works because there is only one possible next-hop.

### State Consistency

PBR state is not visible in `show ip route`. This creates operational blind spots:

| Visible To | Standard Routing | PBR |
|:---|:---:|:---:|
| `show ip route` | Yes | No |
| `show ip cef` | Yes | Partial |
| `traceroute` from router | Yes | No (local PBR only) |
| `show route-map` | N/A | Yes (match counts) |
| `show ip policy` | N/A | Yes (interface mapping) |

This opacity is the single largest operational risk of PBR. Engineers troubleshooting routing issues may never examine PBR because standard tools do not surface it.

---

## 7. PBR Scaling Considerations

### Linear Scaling Model

The number of PBR policies in a network grows with the number of traffic classes and exit points:

$$\text{policies} = \text{classes} \times \text{exit\_points}$$

For an enterprise with 5 traffic classes (voice, video, critical data, bulk, guest) and 2 ISPs:

$$\text{policies} = 5 \times 2 = 10 \text{ route-map clauses}$$

### TCAM Consumption

Each route-map clause translates to TCAM entries proportional to the ACL complexity:

$$\text{TCAM}_{\text{total}} = \sum_{i=1}^{n} (ACE_i + 1)$$

Where $ACE_i$ is the number of ACEs in the match ACL for clause $i$, plus one entry for the set action.

For complex ACLs with many entries, TCAM can be exhausted quickly. Object-group ACLs and summarized prefixes reduce consumption.

### State Tracking Overhead

Each IP SLA probe and track object consumes:
- Memory: ~2 KB per SLA probe history
- CPU: One ICMP echo per probe interval per target
- Bandwidth: Negligible for ICMP, measurable for jitter/HTTP probes

For $n$ tracked next-hops with probe frequency $f$:

$$\text{probes\_per\_second} = n \times f$$

At $f = 0.2$ Hz (one probe every 5 seconds) with 10 tracked next-hops: 2 probes/second, which is negligible.

---

## 8. Design Patterns

### Dual-ISP with Failover

The canonical PBR use case. Traffic is classified by source subnet or application, steered to the appropriate ISP, with failover via IP SLA tracking.

Design considerations:
- Primary/backup vs load-sharing topology
- DNS reply routing (ensure DNS responses return via the correct ISP)
- NAT interaction (each ISP requires different source NAT pool)
- BGP interaction (PBR overrides BGP-learned paths for matched traffic)

### VRF Route Leaking Alternative

PBR with `set vrf` can replace complex VRF route leaking configurations:

Traditional route leaking:
```
ip vrf CUST
 rd 100:1
 route-target import 200:1
 route-target export 100:1
```

PBR alternative:
```
route-map VRF_LEAK permit 10
 match ip address CUST_TO_INET
 set vrf INTERNET
```

PBR is simpler but lacks the control-plane visibility of proper route leaking. BGP-based route leaking scales better and is visible in the routing table.

### Application-Aware Steering

PBR can implement basic application-aware routing by matching on Layer 4 ports:

| Application | Match | Set Action |
|:---|:---|:---|
| VoIP (RTP) | UDP 16384-32767 | Low-latency ISP + DSCP EF |
| Video conferencing | UDP 3478-3481 | Low-latency ISP + DSCP AF41 |
| Web browsing | TCP 80, 443 | High-bandwidth ISP |
| Bulk transfer | TCP 20, 21, 22 | High-bandwidth ISP |

This is a poor substitute for SD-WAN or proper application identification (NBAR/AVC), which can identify applications by DPI rather than port numbers alone.

---

## 9. PBR in Modern Networks

### SD-WAN as PBR Evolution

SD-WAN platforms (Cisco SD-WAN/Viptela, VeloCloud, Prisma SD-WAN) are essentially PBR taken to its logical conclusion:

| Capability | Traditional PBR | SD-WAN |
|:---|:---:|:---:|
| Match criteria | ACL, length, DSCP | DPI, application ID |
| Path selection | Static next-hop | Dynamic (SLA-aware) |
| Failover | IP SLA (seconds) | Sub-second (BFD) |
| Centralized policy | No | Yes (controller) |
| Per-application SLA | No | Yes |
| Encryption | No | Yes (IPsec) |

SD-WAN subsumes PBR functionality while adding centralized management, per-application SLA enforcement, and encrypted overlays.

### Segment Routing as PBR Successor

In service provider networks, Segment Routing with Traffic Engineering (SR-TE) replaces PBR for traffic steering:

- SR-TE uses MPLS labels or IPv6 SRH (Segment Routing Header) to specify the path
- Policy is defined centrally (PCE — Path Computation Element)
- No per-hop PBR configuration required
- Scales to thousands of policies without TCAM impact at intermediate nodes

### When PBR Remains Appropriate

Despite modern alternatives, PBR is still the right tool when:
- Simple dual-ISP failover on a single router
- No SD-WAN or SR-TE infrastructure available
- Quick tactical fix for asymmetric routing problems
- Linux servers needing source-based routing (RPDB is elegant and efficient)
- Lab and testing environments

---

## 10. Comparison: IOS PBR vs Linux RPDB vs NX-OS PBR

| Feature | IOS/IOS-XE | Linux RPDB | NX-OS |
|:---|:---|:---|:---|
| Configuration model | Route-map on interface | ip rule + ip route table | Route-map on interface |
| Match: source IP | Via ACL | `from` selector | Via ACL |
| Match: destination IP | Via ACL | `to` selector | Via ACL |
| Match: fwmark | No | Yes | No |
| Match: input interface | No (applied per-intf) | `iif` selector | No (applied per-intf) |
| Match: packet length | Yes (`match length`) | No (use fwmark via iptables) | Yes (`match length`) |
| Match: DSCP/TOS | Via ACL | `tos` selector | Via ACL |
| Set: next-hop | Yes | Via table default route | Yes |
| Set: VRF | Yes | Via table isolation | Yes (VRF-aware tables) |
| Set: DSCP | Yes | No (use tc/iptables) | Yes |
| Failover tracking | IP SLA + track | External (keepalived, scripts) | IP SLA + track |
| Hardware offload | Yes (TCAM) | No (kernel, but fast) | Yes (TCAM) |
| Visibility | `show route-map`, `show ip policy` | `ip rule show`, `ip route show table` | `show route-map pbr-statistics` |
| Local traffic policy | `ip local policy route-map` | Rules apply to all traffic | `ip local policy route-map` |
| ECMP within PBR | No (single next-hop per clause) | Yes (multiple nexthops in table) | Yes (`load-share`) |

---

## Prerequisites

- acl, route-maps, ip-sla, vrf, qos-marking, linux-networking, tcam

## Complexity

| Operation | Time | Space |
|-----------|------|-------|
| Route-map evaluation (software) | O(n * m) | O(1) |
| Route-map evaluation (TCAM) | O(1) | O(n * m) TCAM entries |
| IP SLA probe processing | O(1) per probe | O(k) history |
| Linux RPDB rule lookup | O(r) | O(1) |
| fwmark classification (netfilter) | O(f) | O(1) |

Where $n$ = route-map clauses, $m$ = ACEs per clause, $k$ = SLA history depth, $r$ = ip rule count, $f$ = firewall rule count.

---

*PBR is a scalpel, not a saw. Used precisely for specific traffic-engineering problems, it is invaluable. Applied broadly as a substitute for proper routing design, it creates an invisible, undocumented forwarding plane that no engineer can troubleshoot at 3 AM.*
