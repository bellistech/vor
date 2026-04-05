# LISP — Locator/ID Separation Protocol Theory

> *LISP fundamentally challenges a core assumption of IP networking: that a single address serves as both identity and location. By splitting these roles, LISP creates an overlay architecture that solves mobility, multihoming, and scalability problems that have plagued the internet since the exhaustion of the IPv4 address space and the explosion of the global routing table.*

---

## 1. The Identity/Location Problem

Traditional IP addressing conflates two distinct functions:

- **Identity:** "Who is this endpoint?" (used by transport protocols, applications, DNS)
- **Location:** "Where is this endpoint in the network topology?" (used by routing protocols)

When a host moves, its IP address must change (new location), which breaks all existing connections (identity changed). This is the fundamental tension that LISP resolves.

### Architectural Separation

```
Traditional IP:
  Single address = Identity + Location
  Moving host = address change = broken sessions

LISP:
  EID (Endpoint Identifier) = Identity (stable, host-facing)
  RLOC (Routing Locator)    = Location (dynamic, infrastructure-facing)
  Moving host = RLOC change only = sessions preserved
```

The EID namespace is provider-independent and potentially portable. The RLOC namespace is provider-assigned and topologically aggregatable. This separation mirrors the distinction between a person's name (identity) and their mailing address (location).

### Mapping System as Indirection Layer

The mapping system provides the binding between EID and RLOC, functioning as a distributed directory service. This indirection is the architectural core of LISP:

$$\text{Mapping}: \text{EID-prefix} \rightarrow \{(\text{RLOC}_1, p_1, w_1), (\text{RLOC}_2, p_2, w_2), \ldots\}$$

Where each RLOC has an associated priority $p$ (lower is preferred) and weight $w$ (relative traffic proportion within the same priority level).

---

## 2. Map-and-Encap Forwarding Model

LISP uses a **map-and-encapsulate** paradigm rather than the traditional longest-prefix-match forwarding. The forwarding decision is split into two stages.

### Stage 1: Map Lookup (Control Plane)

When an ITR receives a packet destined for an unknown EID, it performs a map-cache lookup. On a cache miss:

1. ITR sends a **Map-Request** to the configured Map-Resolver
2. Map-Resolver forwards to the authoritative Map-Server (or directly to the ETR)
3. The authoritative ETR responds with a **Map-Reply** containing the EID-to-RLOC binding
4. ITR installs the mapping in its local map-cache

### Stage 2: Encapsulation (Data Plane)

Once the mapping is resolved:

1. ITR prepends a new outer IP header with RLOC source and RLOC destination
2. Between the outer IP and inner IP, a UDP header (dst port 4341) and LISP header are inserted
3. The encapsulated packet is routed through the underlay based on RLOC addresses
4. The ETR strips the outer headers and delivers the inner packet to the destination EID

### Encapsulation Overhead Analysis

| Encap Type | Outer IP | UDP | LISP Header | Total Overhead |
|:-----------|:---------|:----|:------------|:---------------|
| IPv4-in-IPv4 | 20 bytes | 8 bytes | 8 bytes | 36 bytes |
| IPv4-in-IPv6 | 40 bytes | 8 bytes | 8 bytes | 56 bytes |
| IPv6-in-IPv4 | 20 bytes | 8 bytes | 8 bytes | 36 bytes |
| IPv6-in-IPv6 | 40 bytes | 8 bytes | 8 bytes | 56 bytes |

The LISP header (8 bytes) contains:

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|N|L|E|V|I|R|K|K|        Nonce/Map-Version                     |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|              Instance ID / Locator-Status-Bits               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

N: Nonce present
L: Locator-Status-Bits field contains RLOC status
E: Echo-Nonce request
V: Map-Version present
I: Instance ID present (24-bit field)
```

The Instance ID (24-bit) supports up to $2^{24} = 16,777,216$ separate VPN instances, far exceeding VLAN (4096) or VRF limitations in most deployments.

---

## 3. LISP Control Plane Messages

### Map-Register (ETR to Map-Server)

The Map-Register is the fundamental registration message. An ETR periodically sends Map-Register messages to its configured Map-Server(s) to maintain its EID-to-RLOC bindings.

```
ETR -> MS: Map-Register
  Fields:
    - EID-prefix and prefix length
    - RLOC set with priority/weight for each
    - Map-Register TTL (default 180 seconds, re-register at 60s intervals)
    - Authentication data (HMAC-SHA-1-96 or HMAC-SHA-256)
    - Want-Map-Notify bit (request acknowledgement)
    - Proxy-Map-Reply bit (allow MS to answer on ETR's behalf)
```

If the Proxy-Map-Reply (P) bit is set, the Map-Server can respond to Map-Requests directly without forwarding to the ETR. This reduces latency but means the MS must have current data.

### Map-Notify (Map-Server to ETR)

Acknowledgement of successful registration. Contains an authentication field matching the Map-Register, confirming the MS accepted the binding.

### Map-Request (ITR to Map-Resolver)

```
ITR -> MR: Map-Request
  Fields:
    - Source EID (the requesting host's address)
    - Destination EID prefix (the target being resolved)
    - ITR-RLOC(s) (where to send the Map-Reply)
    - Nonce (for matching request to reply)
    - Subscribe bit (for pub-sub mode, RFC 9437)
```

The Map-Resolver either forwards the request to the appropriate Map-Server (which forwards to the authoritative ETR) or, if the MS has proxy-map-reply authority, the MS answers directly.

### Map-Reply (ETR or MS to ITR)

```
ETR -> ITR: Map-Reply
  Fields:
    - EID-prefix and prefix length
    - Record TTL (how long the ITR should cache this mapping)
    - RLOC records with:
      - Priority (0-255, 255 = do not use)
      - Weight (0-100, relative within same priority)
      - Reachability flags
    - Nonce (matching the request)
    - Security fields (if LISP-SEC is enabled)
```

### Negative Map-Reply

When no mapping exists, the response is a Negative Map-Reply with an action code:

| Action | Meaning |
|:-------|:--------|
| No-Action | Silently drop packets (prefix does not exist) |
| Natively-Forward | EID is reachable without LISP encapsulation |
| Send-Map-Request | Try again later (transient condition) |
| Drop | Explicitly drop with optional ICMP unreachable |

---

## 4. Convergence Analysis

LISP convergence depends on the interplay of multiple timers and mechanisms.

### Convergence Components

$$T_{convergence} = T_{detect} + T_{withdraw} + T_{propagate} + T_{resolve}$$

Where:

- $T_{detect}$: Failure detection time (BFD: ~50ms, IGP: 1-40s, registration timeout: 180s)
- $T_{withdraw}$: Time for ETR to send registration withdrawal or for MS to expire stale registration
- $T_{propagate}$: Time for MS to notify affected ITRs (SMR or pub-sub)
- $T_{resolve}$: Time for ITRs to obtain and install new mapping

### Pull Model (SMR-based)

In the traditional Solicit-Map-Request (SMR) model:

1. MS detects mapping change
2. MS sends SMR to all ITRs that have cached the affected EID prefix
3. Each ITR sends a new Map-Request in response to the SMR
4. ETR (or MS with proxy-reply) responds with Map-Reply
5. ITR updates map-cache

$$T_{pull} = T_{detect} + T_{smr} + T_{request} + T_{reply} \approx 200\text{-}500\text{ms}$$

The pull model has a scalability concern: during mass mobility events (e.g., disaster failover), many SMRs trigger a storm of Map-Requests.

### Push Model (Pub-Sub, RFC 9437)

In the pub-sub model:

1. ITRs subscribe to EID prefixes of interest
2. On mapping change, MS pushes Map-Notify with updated mapping to all subscribers
3. ITRs update map-cache immediately upon receiving notification

$$T_{push} = T_{detect} + T_{notify} \approx 50\text{-}200\text{ms}$$

The push model eliminates the Map-Request/Map-Reply round trip, roughly halving convergence time and eliminating the request storm problem.

### Comparison with Other Convergence Mechanisms

| Protocol | Mechanism | Typical Convergence |
|:---------|:----------|:-------------------|
| LISP (pull/SMR) | SMR + Map-Request/Reply | 200-500ms |
| LISP (push/pub-sub) | Map-Notify push | 50-200ms |
| BGP | Withdraw + propagation | 1-60s (MRAI dependent) |
| OSPF/IS-IS | LSA/LSP flood + SPF | 50-200ms (with tuning) |
| VXLAN/EVPN | BGP update propagation | 1-5s |

---

## 5. Multihoming with LISP

LISP provides native multihoming without the complexity of BGP-based multihoming (no provider-independent address space, no AS number required for the site).

### Priority/Weight Model

Given a set of RLOCs $\{R_1, R_2, \ldots, R_n\}$ with priorities $\{p_1, p_2, \ldots, p_n\}$ and weights $\{w_1, w_2, \ldots, w_n\}$:

1. Select all RLOCs with the lowest (best) priority value: $R_{active} = \{R_i : p_i = \min(p_1, \ldots, p_n)\}$
2. Distribute traffic among $R_{active}$ proportional to weights: traffic share for $R_i = \frac{w_i}{\sum_{j \in active} w_j}$
3. If all RLOCs in the active set fail, promote the next priority tier

### Worked Example

Site with three uplinks:

| RLOC | Priority | Weight | Role |
|:-----|:---------|:-------|:-----|
| 1.1.1.1 | 1 | 60 | Primary (60% traffic) |
| 2.2.2.2 | 1 | 40 | Primary (40% traffic) |
| 3.3.3.3 | 2 | 100 | Backup |

Normal operation: traffic split 60/40 between 1.1.1.1 and 2.2.2.2.

If 1.1.1.1 fails: 100% to 2.2.2.2 (only remaining priority-1 RLOC).

If both 1.1.1.1 and 2.2.2.2 fail: 100% to 3.3.3.3 (priority-2 promoted).

### Inbound Traffic Engineering

Unlike BGP-based multihoming (where inbound TE requires AS-path prepending, MEDs, or communities), LISP inbound TE is explicit and deterministic. The ETR controls exactly how remote ITRs distribute traffic by setting priority/weight in its Map-Register, and these values are propagated in Map-Replies to ITRs.

---

## 6. LISP-SEC (RFC 9303)

LISP-SEC addresses the critical security gap in LISP: without authentication, a rogue ETR could register any EID prefix and hijack traffic.

### Threat Model

- **EID hijacking:** Attacker registers a victim's EID prefix with its own RLOC
- **Map-Reply spoofing:** Attacker sends forged Map-Reply to ITR with malicious RLOC
- **Overclaiming:** ETR registers a broader prefix than authorized

### LISP-SEC Authentication Chain

```
1. ETR authenticates to MS via shared key (Map-Register auth)
2. MS verifies ETR is authorized for the claimed EID prefix
3. On Map-Request, MS generates an OTK (One-Time Key)
4. MS sends OTK to both ITR (in encapsulated Map-Request) and ETR
5. ETR signs Map-Reply with the OTK
6. ITR verifies the Map-Reply signature using the OTK received from the trusted MS

Trust chain: ITR <--trusts--> MS <--trusts--> ETR
```

Without LISP-SEC, the Map-Reply is not authenticated end-to-end. The MS authenticates the ETR registration, but the ITR has no cryptographic assurance that the Map-Reply it receives actually came from the authorized ETR.

### LISP-SEC Key Hierarchy

| Key | Purpose | Scope |
|:----|:--------|:------|
| Authentication Key | ETR-to-MS registration auth | Per-site, pre-shared |
| One-Time Key (OTK) | Map-Reply integrity | Per-request, generated by MS |
| AD (Authentication Data) | HMAC over Map-Reply fields | Per-reply, derived from OTK |

---

## 7. Proxy Operations (PITR and PETR)

Proxy tunnel routers provide interworking between LISP sites and non-LISP networks, solving the deployment chicken-and-egg problem.

### Proxy Ingress Tunnel Router (PITR)

The PITR makes LISP EID prefixes reachable from the non-LISP internet:

1. PITR advertises LISP EID prefixes into BGP (or the site's IGP)
2. Non-LISP sources route toward the PITR based on these advertisements
3. PITR receives native packets, performs map-cache lookup for the destination EID
4. PITR encapsulates toward the destination ETR's RLOC
5. ETR decapsulates and delivers to the destination host

The PITR essentially acts as a default gateway between the LISP and non-LISP worlds for inbound traffic.

### Proxy Egress Tunnel Router (PETR)

The PETR handles outbound traffic from LISP sites toward non-LISP destinations:

1. LISP ITR receives a packet for a non-LISP destination
2. Map-cache lookup returns a Negative Map-Reply with "natively-forward" action
3. If the ITR cannot natively forward (e.g., it only has RLOC connectivity), it encapsulates toward the PETR
4. PETR decapsulates and forwards natively toward the non-LISP destination

### Deployment Considerations

PETRs are primarily needed when the ITR site does not have native (non-LISP) connectivity to the internet. In many deployments, the ITR can simply forward non-LISP traffic natively without a PETR, making PETRs less common than PITRs.

$$\text{PITR need} = \text{always (for non-LISP} \rightarrow \text{LISP reachability)}$$
$$\text{PETR need} = \text{only when ITR lacks native forwarding path}$$

---

## 8. Comparison with Other Overlay Protocols

### LISP vs. VXLAN

| Aspect | LISP | VXLAN |
|:-------|:-----|:------|
| Primary use case | WAN overlay, mobility, multihoming | Data center L2 extension |
| Control plane | LISP MS/MR (purpose-built) | Flood-and-learn or EVPN/BGP |
| Encapsulation | UDP 4341 + LISP header | UDP 4789 + VXLAN header |
| Segmentation | 24-bit Instance ID | 24-bit VNI |
| Host mobility | Native (map-register update) | Requires EVPN MAC mobility |
| Multihoming | Native (priority/weight) | EVPN multihoming (ESI) |
| Standardization | IETF (RFC 9300/9301) | IETF (RFC 7348) |

In Cisco SD-Access, both coexist: LISP provides the control plane (EID-to-RLOC mapping, host tracking) while VXLAN provides the data plane (L2/L3 encapsulation with SGT in the GPO extension).

### LISP vs. GRE/IPsec

| Aspect | LISP | GRE | IPsec |
|:-------|:-----|:----|:------|
| Tunnel model | Dynamic (map-and-encap) | Static point-to-point | Static or dynamic (IKE) |
| Scalability | $O(1)$ config per site | $O(N^2)$ tunnels for full mesh | $O(N^2)$ SAs for full mesh |
| Mobility | Native | Not supported | Not supported |
| Multicast | Via LISP multicast or head-end replication | Native | Not natively |
| Encryption | No (pair with IPsec if needed) | No | Yes |

### LISP vs. OTV (Overlay Transport Virtualization)

| Aspect | LISP | OTV |
|:-------|:-----|:----|
| Layer | L3 overlay | L2 overlay (DCI) |
| Purpose | Routing, mobility, multihoming | L2 extension across DCI |
| Loop prevention | Not applicable (L3) | Built-in (no STP across overlay) |
| Scope | Campus, WAN, internet | Data center interconnect |

---

## Prerequisites

- ip-fundamentals, routing-theory, bgp, udp, encapsulation, dns, vxlan

## Complexity

| Operation | Time | Space |
|-----------|------|-------|
| Map-cache lookup | O(log n) prefix match | O(n) entries |
| Map-Register processing | O(1) per registration | O(n) site database |
| Map-Request resolution | O(1) lookup + RTT | O(1) |
| Pub-sub notification fan-out | O(s) for s subscribers | O(s) subscriber list |
| Full site failover convergence | O(s) notifications | O(n) cache updates |

---

*LISP's architectural insight — that identity and location are fundamentally different concerns — is not merely a protocol optimization. It is a recognition that the internet's original addressing model, designed for a static world of desktop computers, cannot serve a world where endpoints move between networks, multihome across providers, and number in the tens of billions.*
