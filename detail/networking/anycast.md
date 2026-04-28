# Anycast — Deep Dive

> *Anycast is the trick of giving the same IP address to many machines and letting the routing system decide which one a packet reaches. The math is mostly about graphs (BGP topology), latency budgets (RTT), and convergence timing (MRAI + propagation). Anycast doesn't choose the closest server — the routing protocol does, and the routing protocol's idea of "closest" is rarely yours.*

---

## What anycast actually is

Anycast is a **routing-layer addressing model** in which a single IP address is announced from many physical origins. The network forwards each packet to **one** of those origins — usually the topologically nearest one as judged by the routing protocol — without the sender knowing or caring which.

```
    client                          BGP
     |                               |
     v                               v
  10.0.0.1  -- shortest AS_PATH -->  PoP-NYC  (one of N origins)
   (anycast)                         PoP-LON
                                     PoP-FRA
                                     PoP-TYO
                                     PoP-SYD
```

It is *not* multicast. Multicast delivers one packet to **all** receivers in a group. Anycast delivers one packet to **exactly one** receiver, chosen by routing.

It is *not* a load balancer. A load balancer is a stateful Layer-4/7 device that tracks connections. Anycast is a Layer-3 property of the routing graph: stateless, no session memory, no health checks beyond what BGP gives you.

### Formal definition

Let $A$ be an IP prefix and $\{O_1, O_2, \ldots, O_n\}$ be a set of origin ASNs (or distinct routers) advertising $A$. For any source $s$, the destination chosen is:

$$O^*(s) = \arg\min_{O_i} \; d_{\text{BGP}}(s, O_i)$$

Where $d_{\text{BGP}}$ is the BGP best-path metric — typically:

1. Highest **local-pref** (operator policy)
2. Shortest **AS_PATH** length
3. Lowest **MED** (with caveats)
4. eBGP > iBGP
5. Lowest **IGP cost** to next-hop
6. Lowest **router-ID** tiebreaker

Note that **geographic proximity does not appear**. Two PoPs 500 km apart can be 1 and 6 AS hops away. The 1-AS-hop one wins, even if it crosses an ocean to get there.

### IPv4 vs IPv6 anycast

IPv4 anycast is purely operational — there is **no protocol-level support**. You announce the same prefix from multiple sites; that's it. RFCs:

- **RFC 1546 (1993)** — "Host Anycasting Service" — original concept paper. Mostly historical; describes anycast as a service-discovery primitive that never really materialized in IPv4.
- **RFC 4786 (2006)** — "Operation of Anycast Services" — BCP 126. The operational bible. Failure scenarios, deployment patterns, MED/local-pref guidance.
- **RFC 7094 (2014)** — "Architectural Considerations of IP Anycast" — IAB document. Discusses anycast for stateless vs stateful protocols, treats UDP-only as the safe regime, lists known footguns.

IPv6 anycast is **protocol-recognised** in RFC 4291 — anycast addresses are syntactically indistinguishable from unicast (no special prefix), but the semantics are blessed by the spec. Subnet-Router Anycast Address (the all-zeros host within a subnet) is reserved by RFC 4291 § 2.6.1. Reserved subnet anycast IDs are listed in RFC 2526.

```
IPv6 reserved anycast ID space (within an EUI-64 host part):
  0xFDFFFFFFFFFFFF80 - 0xFDFFFFFFFFFFFFFF
  i.e. last 7 bits = anycast IDs 0x00 .. 0x7F (128 reserved IDs)
```

### Why it works at all

Anycast survives because BGP is a **destination-based** protocol. Each router along the path makes an independent decision based purely on the destination IP. As long as the routing graph stays stable for the duration of one packet (microseconds to milliseconds), the packet gets to *some* origin. For UDP, that's enough. For TCP, "stable for the duration of a session" is a much stronger requirement and is the source of every anycast horror story.

---

## BGP-anycast — the workhorse

### Announcement origination

The simplest deployment: every PoP runs a BGP speaker that originates the anycast prefix from the **same ASN** (single-origin) or from **per-PoP ASNs** (multi-origin, MOAS — Multiple Origin AS).

```
single-origin anycast:
  PoP-NYC: AS 13335 announces 1.1.1.0/24 origin 13335
  PoP-LON: AS 13335 announces 1.1.1.0/24 origin 13335
  PoP-TYO: AS 13335 announces 1.1.1.0/24 origin 13335

multi-origin anycast (MOAS):
  PoP-NYC: AS 65001 announces 1.1.1.0/24 origin 65001
  PoP-LON: AS 65002 announces 1.1.1.0/24 origin 65002
  PoP-TYO: AS 65003 announces 1.1.1.0/24 origin 65003
```

MOAS triggers RPKI **invalid** state unless every origin AS has a valid ROA for the prefix. Cloudflare, Google, Facebook all use **single-origin** to keep RPKI happy with one ROA.

### AS_PATH manipulation (prepending)

To make a PoP **less preferred**, you prepend your own ASN multiple times:

```
PoP-NYC: announce 1.1.1.0/24 AS_PATH [13335]                  (preferred near NYC)
PoP-LON: announce 1.1.1.0/24 AS_PATH [13335]                  (preferred near LON)
PoP-FRA: announce 1.1.1.0/24 AS_PATH [13335 13335]            (1× prepend, fallback)
PoP-SYD: announce 1.1.1.0/24 AS_PATH [13335 13335 13335]      (2× prepend, last resort)
```

**Effective fan-out**: if PoP-NYC fails, traffic that was reaching it via shortest AS_PATH now sees:

$$\text{new winner} = \arg\min(\text{path lengths to remaining PoPs})$$

If all surviving PoPs were prepended once, you've now flattened the topology — the packet might end up on the other side of the planet.

**Diminishing returns**: after about 3 prepends, ISPs along the way often stop honoring it (their local-pref policy beats AS_PATH length). Prepend math:

$$P_{\text{honored}} \approx 1 - (1 - p)^k$$

Where $p$ is the per-hop probability of prepend being honored (~0.7 in the wild) and $k$ is the prepend count. After $k=4$, you've added length but not differentiation.

### Local-preference signaling

Inside your own AS, **local-pref** dominates AS_PATH length. The classic anycast pattern:

```
PoP-NYC ingress: local-pref 200 (primary)
PoP-LON ingress: local-pref 200 (primary)
PoP-DAL ingress: local-pref 100 (cold standby — only used if NYC withdraws)
PoP-CHI ingress: local-pref 50  (canary — never wins unless everyone else is down)
```

A neighbour AS sees this through **BGP communities**:

```
13335:1     → set local-pref 200
13335:2     → set local-pref 100
13335:3     → set local-pref 50
13335:666   → blackhole (RFC 7999)
```

You announce the prefix once but tag it with different communities at different PoPs to nudge neighbour-AS preference.

### Withdrawal failover timing

When a PoP fails (process crash, BGP session drop, BFD loss), the BGP speaker withdraws the prefix. Convergence time at the client is:

$$t_{\text{failover}} = t_{\text{detect}} + t_{\text{withdraw}} + t_{\text{propagate}} + t_{\text{best-path}}$$

Component breakdown:

| Component | Typical | Tuned |
|:---|:---:|:---:|
| $t_{\text{detect}}$ (BFD or BGP keepalive) | 90s (BGP default 3× hold) | 0.3s (BFD 100 ms × 3) |
| $t_{\text{withdraw}}$ (encode+send UPDATE) | <0.1s | <0.1s |
| $t_{\text{propagate}}$ (per-hop MRAI) | 30s × hops | 0–2s × hops |
| $t_{\text{best-path}}$ (each router recomputes) | 0.1–1s | 0.1s |

For an unprepared deployment, real-world failover is **30–90 seconds**. For a tuned anycast deployment (BFD, low MRAI, fast best-path), it's **2–5 seconds**.

### MRAI math

**MRAI** = Minimum Route Advertisement Interval. It throttles how often a BGP speaker sends UPDATE messages for the same prefix to the same peer. RFC 4271 defaults:

- eBGP: **30 seconds** (with jitter ±25%)
- iBGP: **5 seconds**

For anycast, you want MRAI **low** (fast propagation) but not zero (unbounded UPDATE storms during instability):

$$t_{\text{propagate}} \approx \text{AS\_hops} \times \text{MRAI}_{\text{eBGP}}$$

Worked example: an EU client reaches a Tokyo anycast origin via 4 AS hops with default MRAI:

$$t_{\text{propagate}} = 4 \times 30 = 120 \text{ s}$$

Two minutes is forever for a service. With anycast-tuned MRAI = 1 s on your peering edge:

$$t_{\text{propagate}} = 4 \times 1 = 4 \text{ s on your edge}$$

But the rest of the internet still applies its own MRAI. Realistic global convergence: **15–45 s** even with tight tuning.

### IGP convergence inside a PoP

Inside a single PoP, anycast prefixes are usually injected into the IGP (OSPF/IS-IS) and **redistributed** into BGP. IGP convergence is much faster:

$$t_{\text{IGP}} \approx t_{\text{hello-loss}} + t_{\text{SPF}} + t_{\text{install}}$$

With sub-second timers (hello 50 ms, dead 150 ms, SPF delay 50 ms):

$$t_{\text{IGP}} \approx 0.15 + 0.05 + 0.02 = 0.22 \text{ s}$$

Useful for rapid intra-PoP failover (server-A dies, server-B picks up the anycast loopback).

---

## DNS-anycast

DNS over UDP is anycast's native habitat. UDP is stateless, queries are tiny (one packet up, one packet down), and tolerable failure modes are "retry the query" and "ask a different resolver."

### Root server math

The DNS root has **13 letter identities** (A–M), but each letter is implemented as **N anycast instances** scattered globally:

```
A.ROOT-SERVERS.NET     Verisign        ~ 50 instances
B.ROOT-SERVERS.NET     USC-ISI         ~ 6 instances
C.ROOT-SERVERS.NET     Cogent          ~ 12 instances
D.ROOT-SERVERS.NET     U. Maryland     ~ 200 instances
E.ROOT-SERVERS.NET     NASA            ~ 250 instances
F.ROOT-SERVERS.NET     ISC             ~ 240 instances
G.ROOT-SERVERS.NET     DISA            ~ 6 instances
H.ROOT-SERVERS.NET     ARL             ~ 12 instances
I.ROOT-SERVERS.NET     Netnod          ~ 80 instances
J.ROOT-SERVERS.NET     Verisign        ~ 200 instances
K.ROOT-SERVERS.NET     RIPE NCC        ~ 100 instances
L.ROOT-SERVERS.NET     ICANN           ~ 200 instances
M.ROOT-SERVERS.NET     WIDE            ~ 12 instances
                                       -----
TOTAL                                  ~ 1500+ instances
```

(Numbers fluctuate; ICANN root-servers.org publishes the live count.)

The 13-letter limit comes from a historical UDP/DNS constraint: a priming query response must fit in **512 bytes** (the original UDP DNS limit before EDNS0). 13 root NS records + glue plus header just barely fits.

$$\text{Priming response size} = \text{header} + 13 \times (\text{NS RR}) + 13 \times (\text{glue A RR})$$

With name compression, this is approximately:

$$12 + 13 \times 32 + 13 \times 16 = 12 + 416 + 208 = 636 \text{ bytes}$$

That exceeds 512, so EDNS0 (RFC 6891) is mandatory in practice. Inside each letter, anycast is invisible to clients — they see one IP per letter.

### Latency budget

The design goal for root anycast is that **median client RTT** to the nearest instance is bounded:

$$\text{RTT}_{\text{median}} \leq 50 \text{ ms (target)}$$

In practice:

| Region | RTT to nearest root letter | Number of nearby letters |
|:---|:---:|:---:|
| US East | 5–15 ms | 8–10 |
| Western Europe | 5–20 ms | 10+ |
| East Asia | 10–30 ms | 6–8 |
| South America | 15–40 ms | 4–6 |
| Africa | 25–80 ms | 2–4 |
| Pacific Islands | 100–250 ms | 0–1 (mainland letters) |

### Recursive resolver anycast

**1.1.1.1** (Cloudflare), **8.8.8.8** (Google), **9.9.9.9** (Quad9) are recursive resolvers operated as anycast services. Cloudflare alone advertises 1.1.1.1 from **300+ PoPs** as of 2024. Google announces 8.8.8.8 from a similar order of magnitude.

Per-PoP capacity math: if each PoP can sustain $C$ queries/sec and you have $P$ PoPs:

$$Q_{\text{global}} = P \times C$$

But anycast distributes traffic by topology, not load — so the actual per-PoP load is:

$$Q_i = Q_{\text{global}} \times f_i$$

Where $f_i$ is the fraction of internet population (weighted by query rate) that resolves to PoP $i$ as nearest. The largest PoPs (LAX, ORD, AMS, FRA, NRT) can carry **10–20%** of global traffic each.

```bash
dig @1.1.1.1 +short CHAOS TXT id.server
# Returns the PoP airport code, e.g. "LHR" or "DFW"
dig @1.1.1.1 +short CHAOS TXT hostname.bind
# Same idea, slightly different format

dig @8.8.8.8 +short CHAOS TXT id.server
# Google: returns location-server-id like "edns0-client-subnet-..."
```

These chaos-class queries are how you map which PoP your client lands on.

### Authoritative anycast (as a service)

Managed authoritative DNS providers (Cloudflare, AWS Route 53, NS1, Google Cloud DNS, Akamai) all use anycast. Number of PoPs varies:

```
AWS Route 53      ~ 200 anycast PoPs (4 nameserver IPs per zone, each anycast)
Cloudflare DNS    ~ 300 PoPs
NS1               ~ 30 PoPs
Akamai Edge DNS   ~ 100+ PoPs
```

Zone delegation typically uses **4 NS records** per zone, each pointing to a separately-anycast IP — so you get **anycast of anycast**: 4 letters × N instances = effectively 4N possible answer paths.

---

## CDN-anycast vs DNS-direct

There are two dominant ways CDNs steer clients to a PoP:

### Method A: Anycast (BGP topology)

```
client query → anycast IP → BGP-nearest PoP → serve content
```

The CDN announces a single IP from every PoP and lets BGP choose. **Pros**: stateless, fails over without DNS TTL waits, zero infrastructure outside BGP. **Cons**: BGP topology often disagrees with RTT — a 1-AS-hop path through a saturated peer can be slower than a 3-AS-hop path through a healthy one.

### Method B: DNS-based steering (geo/RTT)

```
client query → recursive resolver → CDN nameserver (anycast or unicast)
            → returns IP of best PoP based on EDNS Client Subnet (ECS) or resolver IP
            → client connects to that PoP's unique IP
```

The CDN nameserver inspects the resolver's IP (or ECS subnet, RFC 7871) and returns the **specific** unicast IP of the recommended PoP. **Pros**: per-client decisions, RTT-aware, capacity-aware. **Cons**: TTL-bounded failover (clients cache for $T$ seconds), harder for stateful protocols, ECS leaks subnet metadata.

### How PoP selection actually works

Real CDNs use **hybrid** strategies:

```
              Anycast-only           DNS-only            Hybrid (typical)
              ------------           ---------           ----------------
PoP discovery BGP topology           ECS/RTT lookup      Anycast IP per region
Granularity   AS-level               /24 or finer        AS-level + per-resolver
Failover      <2 s (BGP withdraw)    TTL-bounded         <5 s
RTT awareness no                     yes                 partial
Stateful TCP  fragile                stable              stable
```

Cloudflare uses anycast-everywhere. Akamai uses DNS-steering primarily. Fastly uses anycast plus an internal "GLB" (load balancer) for finer control.

### BGP topology distance vs geographic distance vs RTT

These three "distances" can disagree spectacularly:

```
          NYC --- AMS:    geo  ~5800 km    AS-hops 1   RTT  ~75 ms
          NYC --- LAX:    geo  ~3950 km    AS-hops 2   RTT  ~70 ms
          NYC --- TYO:    geo ~10800 km    AS-hops 1   RTT ~140 ms (transpacific cable)
          NYC --- SFO:    geo  ~4100 km    AS-hops 3   RTT  ~75 ms
```

A client in NYC seeking a service announced from {AMS, LAX, TYO, SFO}:
- **Anycast** picks AMS or TYO (1 AS-hop) — could be the worst RTT
- **DNS-RTT-steering** picks LAX or SFO (~70 ms RTT) — geographically further but lower latency

Anycast operators counter this by **carefully selecting peering** — they pay to be close (low AS-hop count) on the paths that matter most.

### Hot-potato vs cold-potato routing

These describe **outbound** behaviour for a multi-homed AS, but they're crucial for anycast because they determine which PoP receives traffic from a given peer.

**Hot-potato routing**: hand traffic off to the next AS as quickly as possible (lowest IGP cost to the egress). The receiving AS pays the long-haul cost.

```
Sender AS                          Receiver AS (anycast operator)
  client at NYC                    PoPs at NYC, LAX
  egress to receiver at NYC peer   ----> NYC PoP wins (geographically aligned)

  client at NYC                    PoPs at LAX only
  egress to receiver at NYC peer   ----> LAX, but receiver carries cross-country (their cost)
```

**Cold-potato routing**: carry traffic on your own network as far as possible before handing off. The sending AS pays.

```
Sender (cold-potato)               Receiver
  client at NYC                    PoPs at NYC, LAX
  carry traffic to LAX backbone    ----> LAX PoP (sender's choice)
  hand off to receiver at LAX peer
```

For anycast operators, **hot-potato from peers is preferred** — peers drop traffic at your nearest PoP, so the routing-graph "closest" matches their geographic "closest." Long-haul ISPs that prefer cold-potato (their backbone is good) can muddy this.

### RFC 1546 historical context

RFC 1546 (1993, Partridge/Mendez/Milliken) introduced "Host Anycasting Service." Key historical points:

- Proposed anycast as a **service-discovery primitive** ("connect me to *any* mail server").
- Suggested per-service anycast addresses with router-level state.
- Was never deployed as designed — too much router state, no clear billing/peering model.

Modern anycast is **pure routing-layer** (no service awareness in routers) and uses **regular IPs** — no special anycast address class.

---

## Convergence math under failure

### Withdrawal propagation

When PoP $i$ withdraws prefix $A$, the withdrawal walks the BGP graph hop by hop. Per-hop time:

$$t_{\text{hop}} = \text{MRAI} + t_{\text{best-path}} + t_{\text{queue}}$$

Total propagation:

$$t_{\text{propagate}} \approx \text{AS\_hops} \times t_{\text{hop}}$$

With aggressive tuning (MRAI 1s, fast best-path, low queue depth), a 4-hop path converges in **~5 seconds**. With defaults, it can be **2 minutes**.

### Path Hunting

A subtle pathology: when a route is withdrawn, BGP doesn't learn "this prefix is dead globally" — each node only learns "the path I had is gone." Nodes then advertise **other paths** they know, even if those alternative paths transit the failed PoP. This causes bursts of UPDATE traffic and prolonged convergence:

$$t_{\text{convergence}} \leq 2 \times \text{diameter} \times \text{MRAI}$$

Where diameter is the longest path in the AS graph (typically 6–10 in the public internet). Worst case for default MRAI: $2 \times 8 \times 30 = 480$ s ≈ 8 minutes for the global internet to settle. In practice, 30–90 s.

### Prefix de-aggregation

A standard anycast trick: if you announce a /23, but you want a specific /24 to fail over to a different PoP, **split the announcement**:

```
all PoPs: announce 192.0.2.0/23
PoP-A:    announce 192.0.2.0/24 (more specific)
PoP-B:    announce 192.0.2.0/23 only
```

BGP **longest-prefix match** wins — the /24 is preferred everywhere. If PoP-A withdraws the /24, traffic falls back to the /23 (everywhere).

Math: with $k$ /24s announced from $n$ PoPs out of a /16 (256 /24s), the routing table cost is:

$$\text{routes} = 1 + k \times n_{\text{PoPs-per-/24}}$$

The /16 minimum globally-routable (per most DFZ filtering policies) is **/24** — anything more specific than /24 is filtered by major providers. So you can't anycast a /25 or /26 to the public internet.

### Graceful Restart

**BGP-GR** (RFC 4724) lets a router restart its BGP daemon without flapping prefixes. The restarting router's peer keeps forwarding to the old next-hop for a "stale timer" duration (default 120 s) while the daemon reboots.

For anycast: GR means a single PoP can restart its BGP speaker without triggering global withdrawal. Combined with **NSF** (Non-Stop Forwarding), the data plane keeps moving while the control plane reboots.

### Add-Path

**BGP Add-Path** (RFC 7911) lets a router advertise multiple paths for the same prefix to a peer. For anycast scenarios with iBGP route reflectors:

- Without Add-Path: RR sends only one best path → if that one fails, IBGP must reconverge.
- With Add-Path: RR sends N paths → routers can immediately switch to a backup without waiting.

Convergence improvement: $t_{\text{IBGP}}$ drops from "wait for re-advertisement" to "switch to pre-installed backup":

$$t_{\text{Add-Path}} \approx t_{\text{best-path}} \approx 0.1 \text{ s}$$

### BGP-LS

**BGP-LS** (RFC 7752) carries IGP topology (OSPF/IS-IS LSDB) over BGP. Useful for anycast SDN controllers that need a global view:

```
                  +------+
                  | SDN  |
                  | ctrl |
                  +--+---+
                     |
                     | BGP-LS sessions
            +--------+---------+
            |        |         |
         PoP-A    PoP-B     PoP-C
        (IGP A)  (IGP B)   (IGP C)
```

The controller computes optimal anycast prefix placement and pushes config back via NETCONF/gRPC.

---

## Layer-3 anycast vs Layer-4 load-balancing

| Property | Anycast (L3) | Load Balancer (L4/L7) |
|:---|:---|:---|
| State | Stateless (routing-layer) | Stateful (per-connection) |
| Session affinity | None | Cookie / 5-tuple hash / source IP |
| Failover speed | BGP-bound (1–30 s) | Sub-second (LB health check) |
| TCP behaviour | Fragile under route flap | Stable until LB itself fails |
| Capacity awareness | None (BGP can't see load) | Direct (LB sees backend health) |
| Geographic precision | AS-level | Per-client |
| Cost | BGP peering + servers | LB hardware/cloud + servers |
| Scale | Trivial (announce IP) | Bounded by LB capacity |

### Why anycast can drop TCP sessions

A TCP session requires that all packets in both directions reach the same endpoint. If the routing-graph "nearest" anycast origin changes mid-session — because of a peering change, a new BGP UPDATE, a withdrawal anywhere along the path — packets start reaching a *different* anycast origin. That origin has no TCP state for this 5-tuple and will respond with **RST**, killing the connection.

Failure rate during route flap, modeled as Poisson:

$$P_{\text{flap}}(\Delta t) = 1 - e^{-\lambda \cdot \Delta t}$$

Where $\lambda$ is the route-flap rate (events/sec at this client). For a typical client with $\lambda \approx 10^{-5}$ /s (one flap per ~28 hours of session-time):

$$P_{\text{flap}}(\text{1 hour session}) = 1 - e^{-10^{-5} \times 3600} = 1 - e^{-0.036} \approx 3.5\%$$

3.5% session failure rate per hour is unacceptable for most applications — hence anycast's reputation for "weird random TCP drops."

### Mitigation strategies

1. **Short-lived connections** — DNS UDP (one packet), HTTP/3 0-RTT (one packet for handshake).
2. **Connection migration** — QUIC (RFC 9000) connection IDs survive path changes.
3. **Fate-sharing** — every PoP runs the same TCP state replication backbone (Maglev, Glouton).
4. **Stickiness via SO_REUSEPORT** + consistent hashing — see Anycast for stateful protocols below.

---

## Anycast for stateful protocols

Stateless protocols (DNS UDP, NTP, HTTP/3 0-RTT) are anycast-trivial. Stateful protocols (TCP, TLS, QUIC long-lived) require trickery.

### Maglev consistent hashing (Google)

Google's Maglev paper (2016) describes a software load balancer that fronts anycast IPs. The key insight: every node in a Maglev cluster computes the **same hash table** mapping 5-tuples to backends, so any node can receive a packet and forward it to the right backend.

Hash function:

$$h(\text{5-tuple}) = H(\text{src\_ip} \| \text{src\_port} \| \text{dst\_ip} \| \text{dst\_port} \| \text{proto}) \mod M$$

Where $M$ is the size of the lookup table (typically 65,537 — prime). The lookup table is computed offline by deterministically permuting backends:

```
permutation[i][j] = (offset[i] + j × skip[i]) mod M
```

Where $\text{offset}[i] = h_1(\text{backend}_i) \mod M$ and $\text{skip}[i] = h_2(\text{backend}_i) \mod (M-1) + 1$.

When a backend is added or removed, the table re-permutes, but **most entries remain stable**:

$$\text{disruption} \approx \frac{1}{N_{\text{backends}}}$$

Adding a 100th backend disrupts ~1% of flows. Acceptable for short-lived connections; problematic for long sessions, which is why Maglev pairs with **connection tracking** (per-flow state) on top of consistent hashing.

### Beamer / Quincy — packet redirection

When a packet arrives at "the wrong" Maglev node (because the hash table differs by a few entries during reconfiguration), Beamer-style systems **redirect** the packet to the correct backend by encapsulating it (GUE/GRE/IPIP) and forwarding to a sibling node:

```
packet --> Maglev node A --> hash says "should be on node B" --> tunnel to B --> backend
```

Adds one extra hop for ~1% of flows during reconfiguration; eliminates RSTs during anycast PoP changes.

### Flowmap stability across re-balancing

For a Maglev-style table with $M$ slots and $N$ backends, after adding/removing $k$ backends:

$$\text{flow re-mapping rate} \leq \frac{k}{N}$$

In the limit, with $M \gg N$ and well-mixed hashing:

$$\text{re-mapping bound} = \frac{1}{M}$$

per slot per backend change. For $M = 65537$ and one backend change, ~0.0015% of flows re-map per slot.

### DSR (Direct Server Return)

A complementary technique: the LB rewrites packets to the backend, but the backend replies **directly to the client** (skipping the LB on egress). This halves LB load and removes the LB as a state holder for the return path. For anycast: combined with DSR, you can have a per-PoP LB layer that fronts the anycast IP, with consistent hashing across PoP boundaries.

---

## TCP over anycast — why it's hard

The fundamental problem: TCP has a **6-second RTO** floor in some kernels and a **default of 1 s**. A BGP convergence event lasting longer than RTO triggers retransmits. If the retransmits land on a different anycast origin, the destination has no state — it sends RST.

```
client                     anycast                       server
  |---- SYN ---->            (PoP-A wins)            ---->  PoP-A
  |<--- SYN/ACK -            (PoP-A wins)            ----  PoP-A
  |---- ACK --->             (PoP-A wins)            ---->  PoP-A
  |---- DATA -->             (BGP withdrawal)        ----X
                             (re-converges to PoP-B)
  |---- DATA(retransmit) -->                         ---->  PoP-B
  |<---- RST ----            (PoP-B has no state)    ----  PoP-B
```

### Persistent route flap

If a route is **flapping** (alternately announced and withdrawn), TCP sessions through that anycast IP are condemned. The **route flap dampening** technique (RFC 2439) tries to filter unstable prefixes — but it has its own pathology of suppressing prefixes that are merely transiently flapping during normal recovery.

Penalty math (RFC 2439):

$$\text{penalty}(t) = \text{penalty}(0) \times e^{-\lambda t}$$

Where $\lambda = \ln 2 / \text{half-life}$ (default half-life 15 min). Suppress threshold 2000, reuse threshold 750. A prefix that flaps 4 times in quick succession (penalty 4000) gets suppressed for ~30 minutes.

Modern recommendation (RIPE-580): **don't use flap dampening with default values**, or you'll suppress legitimate failover events. Use longer half-lives or skip dampening entirely.

### SO_REUSEPORT pool sharing across PoPs

An advanced mitigation: every PoP runs a TCP socket pool that **shares state** across PoPs via a side-channel (Redis, Memcache, custom protocol). When a packet arrives at the "wrong" PoP, the kernel's `SO_REUSEPORT` group lookup checks the shared state, finds the original PoP, and forwards the packet via tunneling.

Performance cost: one extra round-trip across the operator's backbone for the lookup. Latency hit: **5–50 ms** depending on backbone topology. Overhead of sharing TCP control blocks across PoPs can be large; usually only the SYN cookie + initial sequence number are shared, and the actual TCB lives at the receiving PoP.

### TCP MD5 / TCP-AO complications

If your TCP sessions use **TCP-MD5** (RFC 2385) or **TCP-AO** (RFC 5925) — common for BGP itself — the keys are per-connection. Sharing state across anycast PoPs requires shared keys, which is its own security can-of-worms.

---

## QUIC over anycast

QUIC (RFC 9000) was designed with anycast in mind. The key property: QUIC connections are identified by **connection IDs** (CIDs) chosen by the endpoints, not by the 5-tuple.

### Connection ID rebinding

```
client (CID=0xABCD) ---initial 5-tuple A----> PoP-1
client (CID=0xABCD) ---path migrated to 5-tuple B---> PoP-2
                                                       (PoP-2 sees CID 0xABCD,
                                                        but has no state)
```

To survive anycast reconvergence, PoPs share CID-to-PoP mapping via a backend KV store, or use **CID-based routing**: the LB inspects the CID and forwards to the correct PoP.

CID structure (as chosen by server):

```
CID = | server_id (8 bits) | cluster_id (8 bits) | random (n bits) |
              |                  |                       |
              v                  v                       v
        which PoP        which cluster in PoP      uniqueness
```

### Stateless reset tokens

If a PoP receives a QUIC packet for an unknown CID, RFC 9000 § 10.3 says it should respond with a **stateless reset** — a packet that includes a token derived from a shared secret. Clients verify the token; if it matches the expected token for the CID, the client closes the connection cleanly.

Token math:

$$\text{token} = \text{HKDF}(\text{secret}, \text{CID}, \text{label})$$

Where $\text{secret}$ is shared across all PoPs (per service), $\text{CID}$ is the unknown CID, and $\text{label}$ is "stateless reset". Token length: 16 bytes.

### 0-RTT addresses anycast naturally

QUIC 0-RTT lets clients send application data in the **first packet** alongside the TLS handshake. Cost of an anycast PoP change during 0-RTT: zero — the client just sends another initial packet, which lands wherever BGP sends it. Compare to TCP, where the connection is mid-state and a PoP change is fatal.

### QUIC connection migration

RFC 9000 § 9 ("Connection Migration") explicitly addresses path changes:

```
client --- initial path --->  server  (CID 0xABCD)
client (NAT rebinding, mobile handoff, anycast change)
client --- new path ---> server  (CID 0xABCD, new 5-tuple)
                                  (server validates new path with PATH_CHALLENGE)
```

The server sends `PATH_CHALLENGE` with a random 8-byte token; client echoes `PATH_RESPONSE` with the same token. Round-trip required: 1 RTT.

For anycast: as long as the new PoP can be told about the existing connection (via CID lookup in shared state), migration is seamless.

---

## Failover patterns

### Cold standby

```
PoP-A (primary):   announce prefix, local-pref 200
PoP-B (cold):      DO NOT announce until PoP-A withdraws
```

When PoP-A withdraws (BGP-detected), PoP-B's automation announces the prefix. Failover time:

$$t_{\text{cold}} = t_{\text{detect-A-down}} + t_{\text{B-announce}} + t_{\text{global-converge}}$$

Typically 30–60 s. Simple, cheap, but slow.

### Hot standby (selective preference)

```
PoP-A: announce, AS_PATH [13335]
PoP-B: announce, AS_PATH [13335 13335]  (1× prepend = always less preferred)
```

Both PoPs are active and serving; PoP-B only wins for clients whose BGP path to PoP-A is genuinely worse. When PoP-A withdraws, PoP-B's announcement is already in-graph and immediately becomes best.

Failover time:

$$t_{\text{hot}} = t_{\text{withdraw-converge}} \approx 5-30 \text{ s}$$

Faster than cold standby because no new announcement is needed.

### Canary (low-localpref)

```
PoP-canary: announce, BGP community 13335:50  (set local-pref = 50, very low)
all others: BGP community 13335:200            (local-pref = 200)
```

The canary PoP only attracts traffic when **everything else** is down (or for specific clients with policy that prefers it). Useful for testing new code paths in production with safety.

### Per-region selective preference

```
PoP-NYC (East coast): community 13335:200
PoP-LAX (West coast): community 13335:200
PoP-FRA (EU):         community 13335:200
PoP-TYO (APAC):       community 13335:200

PoP-DAL (cold backup for NYC):  community 13335:100 + AS_PATH [13335 13335]
```

Each region has a primary PoP at high local-pref. A backup PoP per region has lower local-pref and a prepended AS_PATH, attracting traffic only when the regional primary fails.

### Selective community policy

| Community | Effect |
|:---|:---|
| 0:peer-asn | Don't announce to this peer |
| 65535:0 | NO_EXPORT (don't announce outside this AS) |
| 65535:1 | NO_ADVERTISE (don't tell anyone) |
| 65535:65281 | NO_PEER (don't announce to peers, only customers) |
| operator:1xx | local-pref steering tier 1 |
| operator:2xx | local-pref steering tier 2 |
| operator:666 | RTBH blackhole (RFC 7999) |

Anycast operators publish their community policy at PeeringDB or NOC pages.

---

## Anycast monitoring

### RIPE Atlas probes

[RIPE Atlas](https://atlas.ripe.net) is a global network of ~12,000 hardware probes that you can rent (with credits) to issue measurements toward your prefix. Probe types: ping, traceroute, DNS, HTTP, SSL.

```bash
# create a measurement against your anycast prefix from 100 random probes
curl -X POST https://atlas.ripe.net/api/v2/measurements/ \
     -H "Authorization: Key ATLAS_KEY" \
     -d '{
       "definitions": [{"target": "1.1.1.1", "af": 4, "type": "traceroute"}],
       "probes": [{"requested": 100, "type": "area", "value": "WW"}]
     }'
```

Atlas results show **per-probe AS path** to your anycast IP, letting you detect "this region of the internet is reaching the wrong PoP."

### Traceroute mapping

Mass-traceroute campaigns map the anycast catchment by region. Key tool: **Verfploeter** (Dutch for "rinser") — a UDP-pair probing technique that uses the source-IP of an ICMP-unreachable response to identify which PoP responded.

```
client(IP_X) --- ICMP echo to anycast --> PoP-Y replies --- ICMP echo reply

trace headers reveal which PoP (the source IP of the reply or its TTL pattern)
```

### Looking glasses

Public BGP looking glasses let you inspect routes from inside provider networks:

- **bgp.he.net** (Hurricane Electric) — view routes to your prefix
- **lg.ring.nlnog.net** (NLNOG ring) — community LG
- **stat.ripe.net** (RIPE STAT) — historical RIB views
- **routeviews.org** (Oregon) — BGP RIB dumps every 2 hours

```bash
# Use HE looking glass via web or, equivalently, query their public BGP RIB
curl 'https://bgp.he.net/AS13335#_prefixes' -o he-prefixes.html

# RIPE RIS via REST API
curl 'https://stat.ripe.net/data/looking-glass/data.json?resource=1.1.1.0/24'
```

### PeeringDB

Your peering policy and PoP locations are listed in **PeeringDB** (peeringdb.com). Anycast operators publish:

- Per-PoP IXP presence
- Open vs selective peering policy
- Traffic level estimates
- Looking glass URL per PoP

### Anycast-aware health checks

Internal tooling: every PoP exports BGP-speaker status, prefix-announcement health, and per-prefix client volume. A dashboard correlates **announced** vs **observed** RTT-from-Atlas to spot PoPs that are advertised but not actually serving well.

```
# minimal per-PoP exporter
bgp_neighbor_state{neighbor="ix.peer.1"} 6   # ESTABLISHED
bgp_advertised_prefixes_total 327
client_qps_total 1.2e6
client_rtt_p50_ms 4.3
client_rtt_p99_ms 38.1
```

---

## Geographic and topological dispersion math

### PoP placement formula

Goal: place $N$ PoPs to minimize total weighted RTT to a population $P = \{p_1, \ldots, p_K\}$ (e.g., cities) subject to a budget constraint.

$$\min_{\{x_1, \ldots, x_N\}} \sum_{i=1}^{K} w_i \cdot \min_{j \in [1,N]} \text{RTT}(p_i, x_j)$$

Where $w_i$ is the population weight (or query volume) and $\text{RTT}$ is measured RTT (not great-circle distance — use empirical RTT from probes).

This is a **k-medians** problem on a graph; NP-hard in general but well-approximated by:
- k-medoids clustering on probe RTT data
- Greedy facility location with submodular bounds (factor-3 approx)
- Linear programming relaxation + rounding

### Chebyshev center

For a continuous geographic placement on Earth (lat/lon), the **Chebyshev center** of a set of points is the point that minimizes the maximum distance:

$$x^* = \arg\min_x \max_i d(x, p_i)$$

For 2D Euclidean it's the center of the smallest enclosing circle. For Earth (spherical), it's the smallest enclosing spherical cap. Useful when you want to **cover** an audience with one PoP rather than minimize the average.

### Weighted Voronoi tessellation

Once PoPs are placed, the **catchment** (which PoP wins for each location) is a **weighted Voronoi diagram** where weights are PoP capacities or RTT offsets:

$$V_j = \{x : \forall k \neq j, \, \text{RTT}(x, x_j) - \alpha_j \leq \text{RTT}(x, x_k) - \alpha_k\}$$

Where $\alpha_j$ is a per-PoP bias (set high for under-utilized PoPs, low for hot ones). Anycast operators tune $\alpha_j$ via local-pref / prepending to balance load.

### Population coverage

A common metric: percentage of users within $\tau$ ms RTT of nearest PoP.

$$\text{Coverage}(\tau) = \frac{\sum_i w_i \cdot \mathbf{1}[\min_j \text{RTT}(p_i, x_j) \leq \tau]}{\sum_i w_i}$$

Common targets:

| Target | $\tau$ |
|:---|:---:|
| Web content | 50 ms |
| DNS | 30 ms |
| Real-time (gaming, voice) | 20 ms |
| HFT / latency-critical | <5 ms |

### PoP count vs coverage

Empirically (Cloudflare, CDN reports), coverage scales sublinearly with PoP count:

$$\text{Coverage}(N) \approx 1 - \frac{c}{N^{0.5}}$$

Doubling PoPs from 50 to 100 might move coverage from 90% to 93% — diminishing returns. The first 20 PoPs (placed in major peering hubs: AMS, FRA, LON, NYC, LAX, SFO, ORD, ATL, NRT, SIN, SYD, etc.) capture 80% of population.

---

## Worked examples

### Example 1: Cloudflare 1.1.1.1 traceroute analysis

```bash
# from a US East Coast client
$ mtr -rwbzc 50 1.1.1.1
HOST: client.example.com               Loss%   Snt   Last   Avg  Best  Wrst StDev
  1. 192.168.1.1                        0.0%    50    0.5   0.5   0.5   0.6   0.0
  2. AS7922  10.0.0.1                   0.0%    50    8.3   8.5   8.0  10.1   0.4
  3. AS7922  ip-comcast-edge.net        0.0%    50    8.7   9.0   8.5  11.0   0.5
  4. AS174   te0-0-0.cogent-iah.net     0.0%    50   12.2  12.4  12.0  14.2   0.4
  5. AS13335 1.1.1.1                    0.0%    50   12.5  12.6  12.3  14.5   0.4
```

Reading this: 4 IP hops, 3 AS hops (Comcast → Cogent → Cloudflare), RTT 12.5 ms. The CHAOS-class query reveals the PoP:

```bash
$ dig @1.1.1.1 +short CHAOS TXT id.server
"IAH"
```

PoP IAH (Houston, TX). The traceroute tells us the path traversed Cogent's IAH router; Cloudflare's IAH PoP was hot-potato'd from Cogent.

### Example 2: DNS root letter L (Cogent) failover sequence

L-root is operated by ICANN with anycast. Around 2014, a config error briefly de-aggregated L-root's prefix; within 30 seconds, ~12% of global DNS queries to L-root were redirected to a partial set of healthy instances. Sequence:

```
T+0:    Bug pushes /24 announcement at LAX instance
T+1s:   LAX announces 199.7.83.42/32 (more specific) via local IGP
T+5s:   IGP propagates inside ICANN backbone
T+10s:  EBGP advertises 199.7.83.0/24 from LAX peer
T+15s:  Tier-1s receive UPDATE, install /24 (longer prefix wins)
T+30s:  Global queries to 199.7.83.42 funnel toward LAX (now overloaded)
T+45s:  Operators detect, withdraw the /24 announcement
T+90s:  Global convergence, traffic re-distributes to all L-root instances
```

Total user impact: ~75 seconds of skewed catchment. Lessons:
- Always test BGP changes in lab before deploying.
- Monitor per-instance traffic to detect skew.
- Have a "kill switch" — fastest way to undo is withdrawal of the bad announcement.

### Example 3: Anycast IPv6 deployment for a SaaS endpoint

Suppose you operate a SaaS API and want a single IPv6 anycast endpoint (`2001:db8::api`) served from PoPs in NYC, FRA, SIN. Steps:

```bash
# Step 1: get a routable /48 from your ARIN/RIPE/APNIC allocation
# 2001:db8:1::/48 (replace with real allocation)

# Step 2: at each PoP, configure loopback on each server
ip -6 addr add 2001:db8::api/128 dev lo

# Step 3: announce 2001:db8::/48 via BGP from each PoP
# (Cisco IOS-XR example)
router bgp 65001
 address-family ipv6 unicast
  network 2001:db8::/48

# Step 4: tag with regional communities
route-policy ANYCAST-OUT
  if destination in (2001:db8::/48) then
    set community (65001:200) additive
  endif
end-policy

# Step 5: verify with looking glass
curl 'https://lg.he.net/?protocol=ipv6&query=2001:db8::api&type=bgp'
```

Now traffic to `2001:db8::api` lands at the BGP-nearest of {NYC, FRA, SIN}. RPKI ROA must cover the /48 from AS 65001 to keep prefix valid:

```bash
# Cryptech RPKI signing
rpki-client -s 65001 -p 2001:db8::/48 -m 48 > anycast.roa
```

### Example 4: BGP withdrawal timing in a 2-PoP failover

PoP-A (NYC) primary, PoP-B (LAX) hot standby with 1× prepend. PoP-A's BGP speaker crashes.

```
T+0.0s:    PoP-A BGP daemon dies (no graceful shutdown)
T+0.0s:    PoP-A's eBGP peer doesn't see TCP RST yet
T+0.1s:    BFD session to PoP-A peer goes down (50 ms × 3)
T+0.2s:    PoP-A peer marks session DEAD, scans RIB for prefixes via PoP-A
T+0.3s:    PoP-A peer issues UPDATE withdrawal for affected prefixes to its peers
T+0.5s:    Withdrawals propagate through Tier-1
T+1.0s:    Tier-1 routers compute new best path → PoP-B (1× prepended)
T+1.5s:    Tier-1 advertises new path to its customers
T+2-5s:    Eyeball networks receive new path, install in FIB
T+5-15s:   Global convergence — most clients now hit PoP-B
T+30-90s:  Long-tail clients with slow peers finally converge
```

With BFD + low MRAI, the bulk of clients fail over in **under 5 seconds**. Without BFD (BGP keepalive only), the detection alone is **~90 seconds** (default 60 s keepalive, 180 s hold) and total failover is **2–3 minutes**.

### Example 5: Convergence math worked out

Setup: 4 PoPs (NYC, LAX, FRA, TYO), client in Mumbai. Default BGP settings.

```
client--Mumbai  --AS_hops 3-->  TYO  (preferred)
                --AS_hops 4-->  FRA
                --AS_hops 4-->  LAX
                --AS_hops 5-->  NYC
```

TYO crashes. New best path: FRA (4 hops).

$$t_{\text{detect}} = 90 \text{ s (BGP hold timer expiry)}$$
$$t_{\text{withdraw}} = 0.1 \text{ s}$$
$$t_{\text{propagate}} = 4 \text{ AS hops} \times 30 \text{ s MRAI} = 120 \text{ s}$$
$$t_{\text{best-path}} = 0.5 \text{ s per node}$$
$$t_{\text{total}} = 90 + 0.1 + 120 + 0.5 \approx 211 \text{ s}$$

3.5 minutes. With BFD (300 ms detect) and MRAI=2:

$$t_{\text{tuned}} = 0.3 + 0.1 + 4 \times 2 + 0.5 \approx 9 \text{ s}$$

20× faster. This is why anycast operators obsess over MRAI/BFD tuning.

---

## When NOT to use anycast

### Long-lived TCP sessions

If your application maintains TCP sessions for **minutes to hours** (databases, streaming RPC, persistent WebSockets), anycast is dangerous. Even small route flap rates accumulate over time.

$$P_{\text{session-failure}}(\text{1 hour}) = 1 - e^{-\lambda \cdot 3600}$$

For $\lambda = 10^{-4}$ /s: 30% failure rate per session. For $\lambda = 10^{-5}$ /s: 3.5%. Both unacceptable for a SaaS API.

Better alternative: **DNS-based steering** with short TTLs and an L4/L7 LB. Or **anycast for connection bootstrap, unicast for the session** — the anycast IP returns a unicast IP via early protocol message; the long-lived session goes to that unicast IP.

### Sticky-state applications

Shopping carts, multi-step workflows, anything that remembers who you are without explicit cookies — these break under anycast PoP changes. Backends across PoPs would need to share state, which often means a global database with global write latency.

### Per-PoP cost models

Some traffic carriers bill per-PoP. If you announce from 50 PoPs and pay $10/Gbps/month per PoP, you're committing to 50× the egress overhead vs a unicast deployment.

### Low-volume services

Anycast adds operational complexity (BGP peering, RPKI ROAs, monitoring per-PoP, multi-region deployments). For services pushing <1 Gbps globally, a single well-chosen unicast region with DNS failover is cheaper and simpler.

### Compliance-restricted data

Anycast routes packets to **whichever** PoP wins the BGP race — including cross-border. If your data has data-residency requirements (GDPR, HIPAA, PCI), anycast routing may put data through the wrong PoP. Restrict per-region announcements with BGP communities or use DNS-based steering.

### Anti-DDoS scenarios (counter-intuitive)

Anycast is often the right answer for absorbing volumetric DDoS — it spreads attack traffic across many PoPs. But if the attack is **adaptive** (detects which PoP responds and floods that exact one), anycast doesn't help — it's just inviting attackers to flood every PoP simultaneously. Dedicated scrubbing services (Arbor, Cloudflare Magic Transit, Akamai Prolexic) front the anycast endpoint.

---

## Operational checklist

### Before deploying anycast

```
[ ] Allocated /24 (IPv4) or /48 (IPv6) — minimum globally-routable prefix
[ ] RPKI ROA published for the prefix from each origin AS
[ ] BGP peering established at every PoP (transit + IXP peers)
[ ] Loopback IPs configured at every server on every PoP
[ ] BGP communities defined for local-pref steering
[ ] BFD/sub-second timers tuned on all peering sessions
[ ] Looking glass test — verify announcement from each PoP
[ ] RIPE Atlas measurement — verify catchment from intended user regions
[ ] Failover test — simulate single-PoP failure and time convergence
[ ] Monitoring: per-PoP traffic, per-PoP BGP state, per-PoP application health
[ ] Run-book: how to manually withdraw a PoP, how to add prepending mid-incident
```

### During an incident

```
1. Identify affected PoP(s) — looking glass + Atlas
2. Verify whether the issue is:
   a) PoP application-layer failure  → withdraw BGP, fail over
   b) BGP propagation issue           → check upstream peer state
   c) Routing-table corruption        → reload, escalate
3. Communicate to peers via NOC contact (PeeringDB)
4. Monitor convergence — Atlas + internal metrics
5. Post-mortem: capture BGP timing logs, MRAI behavior, FIB churn
```

### Post-deployment review

```
- Catchment map: which AS prefixes hit which PoP?
- Latency map: p50, p95, p99 RTT from each major eyeball AS
- Capacity map: per-PoP utilization vs designed budget
- Cost map: per-PoP transit cost vs traffic delivered
- Failure history: count of BGP sessions flapping, mean time to convergence
```

---

## Common BGP looking-glass commands

```bash
# Hurricane Electric
curl 'https://bgp.he.net/AS13335#_prefixes'        # prefixes announced by AS
curl 'https://bgp.he.net/net/1.1.1.0/24#_routes'   # routes seen by HE for prefix

# RIPE STAT
curl 'https://stat.ripe.net/data/looking-glass/data.json?resource=1.1.1.0/24'
curl 'https://stat.ripe.net/data/bgp-state/data.json?resource=AS13335&starttime=2024-01-01'

# RouteViews (Oregon, public BGP collectors)
telnet route-views.routeviews.org
> show ip bgp 1.1.1.0/24
> show ip bgp regexp _13335$       # prefixes originating from AS13335
> show ip bgp summary

# Use RIPE's online traceroute (Atlas one-off)
curl -X POST https://atlas.ripe.net/api/v2/measurements/ \
  -H "Authorization: Key $ATLAS_KEY" \
  -d '{"definitions":[{"target":"1.1.1.1","type":"traceroute","af":4}],
       "probes":[{"requested":50,"type":"area","value":"WW"}]}'

# dig for anycast PoP identification
dig @1.1.1.1 +short CHAOS TXT id.server         # Cloudflare PoP
dig @8.8.8.8 +short CHAOS TXT id.server         # Google PoP
dig @9.9.9.9 +short CHAOS TXT id.server         # Quad9 PoP

# mtr for anycast trace
mtr -rwbzc 100 1.1.1.1                          # 100 cycles, AS-aware, no DNS

# IPv6 anycast trace
mtr -6 -rwbzc 100 2606:4700:4700::1111
```

---

## Quick math reference

| Quantity | Formula | Typical |
|:---|:---|:---:|
| BGP peering sessions (full mesh) | $\frac{N(N-1)}{2}$ | $O(N^2)$ |
| BGP peering with RR | $N - 1$ | $O(N)$ |
| Convergence time | MRAI × hops | 30–120 s default |
| BFD detection time | hello × multiplier | 0.3 s typical |
| Cache hit ratio (DNS) | $1 - 1/(\lambda T)$ | 95–99.7% |
| Stateless reset token | HKDF(secret, CID) | 16 bytes |
| Maglev table size | prime $M$ | 65,537 |
| Maglev disruption | $\approx 1/N$ | <2% |
| Coverage at $\tau$ ms | $1 - c/N^{0.5}$ | sublinear |
| Session failure rate | $1 - e^{-\lambda t}$ | $\lambda \approx 10^{-5}$ |
| Prefix de-agg longest match | /24 IPv4, /48 IPv6 | DFZ filter |
| MRAI eBGP | 30 s default, 0–2 s tuned | RFC 4271 |
| MRAI iBGP | 5 s default | RFC 4271 |

---

## Anycast vs other distribution patterns

| Pattern | Layer | State | Scale | Failover | Notes |
|:---|:---|:---|:---|:---|:---|
| Anycast | L3 | Stateless | Global | BGP-bound | Simple, fragile for TCP |
| GeoDNS | L7 (DNS) | Stateless | Global | TTL-bound | Requires resolver IP / ECS |
| Round-robin DNS | L7 | Stateless | Limited | TTL + retry | Crude, no health awareness |
| L4 LB (HAProxy, IPVS) | L4 | Per-connection | Per-region | Sub-second | Stateful, capacity-bounded |
| L7 LB (Envoy, Nginx) | L7 | Per-request | Per-region | Sub-second | Stateful, expensive |
| GTM (BIG-IP, Akamai) | L7 + DNS | Stateful | Global | Health-check + TTL | Commercial GSLB |
| Traffic Manager (Azure, AWS) | DNS | Stateless | Global | TTL | Cloud-managed |
| Multipath (MPTCP, QUIC) | L4 | Per-session | Limited | In-protocol | Survives anycast flap |

---

## Glossary

- **Anycast** — same IP advertised from many origins; routing chooses one.
- **AS_PATH prepending** — adding your AS multiple times to make a route less attractive.
- **BFD** — Bidirectional Forwarding Detection. Sub-second link/peer failure detection.
- **BGP-LS** — BGP Link-State; carries IGP topology in BGP for SDN consumption.
- **CID** — Connection ID. QUIC's connection identifier, independent of 5-tuple.
- **DFZ** — Default-Free Zone. The portion of the internet routing table without default routes (Tier-1 backbone).
- **ECS** — EDNS Client Subnet (RFC 7871). Reveals client subnet to authoritative DNS.
- **GR / NSF** — Graceful Restart / Non-Stop Forwarding. Lets BGP daemon restart without flapping prefixes.
- **IXP** — Internet Exchange Point. Shared peering fabric (e.g., AMS-IX, DE-CIX).
- **Local-pref** — BGP path attribute, highest wins (operator policy).
- **Maglev** — Google's anycast-friendly LB algorithm with consistent hashing.
- **MED** — Multi-Exit Discriminator. BGP path attribute for multi-link comparison.
- **MOAS** — Multiple Origin AS. Same prefix announced from different ASes.
- **MRAI** — Minimum Route Advertisement Interval. Throttles BGP UPDATEs (default 30s eBGP).
- **PeeringDB** — public registry of peering policies and PoPs.
- **PoP** — Point of Presence. A physical site with peering and servers.
- **Prepending** — see AS_PATH prepending.
- **RPKI** — Resource Public Key Infrastructure. Cryptographic origin validation for BGP.
- **ROA** — Route Origin Authorization. RPKI signed object: "AS X may originate prefix Y."
- **RR** — Route Reflector. iBGP scaling — replaces full mesh.
- **RTBH** — Remote-Triggered Black Hole. Use BGP community to drop traffic at edge.
- **SO_REUSEPORT** — Linux socket option for multi-process sharing of a TCP listener.

---

## See Also

- `networking/bgp` — BGP fundamentals: best-path algorithm, attributes, MRAI
- `networking/bgp-advanced` — route reflection, BGP communities, RPKI, BFD, Add-Path
- `networking/dns` — recursion, TTL, caching, anycast root math
- `networking/http3-quic` — QUIC connection IDs, migration, 0-RTT
- `networking/ecmp` — equal-cost multipath, related stateless distribution
- `ramp-up/anycast-eli5` — the friendly companion narrative
- `ramp-up/bgp-eli5` — BGP from first principles, ELI5

---

## References

- **RFC 1546** — Host Anycasting Service (1993). Original anycast concept paper. <https://www.rfc-editor.org/rfc/rfc1546>
- **RFC 4271** — A Border Gateway Protocol 4 (BGP-4). MRAI, hold timer, decision process. <https://www.rfc-editor.org/rfc/rfc4271>
- **RFC 4291** — IP Version 6 Addressing Architecture. IPv6 anycast (§ 2.6). <https://www.rfc-editor.org/rfc/rfc4291>
- **RFC 4724** — Graceful Restart Mechanism for BGP. <https://www.rfc-editor.org/rfc/rfc4724>
- **RFC 4786** — Operation of Anycast Services (BCP 126). The operational bible. <https://www.rfc-editor.org/rfc/rfc4786>
- **RFC 5925** — TCP Authentication Option. Replaces TCP-MD5 for BGP. <https://www.rfc-editor.org/rfc/rfc5925>
- **RFC 6891** — Extension Mechanisms for DNS (EDNS0). Required for modern DNS over UDP. <https://www.rfc-editor.org/rfc/rfc6891>
- **RFC 7094** — Architectural Considerations of IP Anycast (IAB). Stateful protocol caveats. <https://www.rfc-editor.org/rfc/rfc7094>
- **RFC 7752** — North-Bound Distribution of Link-State and TE Information using BGP (BGP-LS). <https://www.rfc-editor.org/rfc/rfc7752>
- **RFC 7871** — Client Subnet in DNS Queries (ECS). <https://www.rfc-editor.org/rfc/rfc7871>
- **RFC 7911** — Advertisement of Multiple Paths in BGP (Add-Path). <https://www.rfc-editor.org/rfc/rfc7911>
- **RFC 7999** — BLACKHOLE Community. <https://www.rfc-editor.org/rfc/rfc7999>
- **RFC 8092** — BGP Large Communities Attribute. <https://www.rfc-editor.org/rfc/rfc8092>
- **RFC 9000** — QUIC: A UDP-Based Multiplexed and Secure Transport. Connection IDs, migration. <https://www.rfc-editor.org/rfc/rfc9000>
- **RFC 9001** — Using TLS to Secure QUIC. <https://www.rfc-editor.org/rfc/rfc9001>
- **RFC 9002** — QUIC Loss Detection and Congestion Control. <https://www.rfc-editor.org/rfc/rfc9002>
- **RFC 2439** — BGP Route Flap Damping. <https://www.rfc-editor.org/rfc/rfc2439>
- **RFC 2526** — Reserved IPv6 Subnet Anycast Addresses. <https://www.rfc-editor.org/rfc/rfc2526>
- **Cloudflare Engineering Blog — Peering** — series on anycast PoP design, hot-potato, peering economics. <https://blog.cloudflare.com/tag/peering/>
- **Cloudflare — Anycast TCP** — early notes on serving TCP via anycast at scale. <https://blog.cloudflare.com/cloudflares-relationship-with-the-bgp-tcp-protocol/>
- **Cloudflare — How Cloudflare Works** — anycast catchment, PoP architecture. <https://blog.cloudflare.com/a-brief-anycast-primer/>
- **Google — Maglev paper (NSDI 2016)** — "Maglev: A Fast and Reliable Software Network Load Balancer." <https://research.google/pubs/pub44824/>
- **Google — Espresso paper (SIGCOMM 2017)** — peering edge architecture. <https://research.google/pubs/pub46638/>
- **Microsoft — SWAN paper** — wide-area traffic engineering. <https://www.microsoft.com/en-us/research/publication/achieving-high-utilization-with-software-driven-wan/>
- **Verfploeter** — UDP-based anycast catchment measurement. <https://www.usenix.org/conference/atc17/technical-sessions/presentation/de-vries>
- **RIPE Atlas** — global probe network. <https://atlas.ripe.net>
- **RIPE STAT** — looking-glass + RIB historical viewer. <https://stat.ripe.net>
- **RouteViews** — public BGP RIB collector. <https://www.routeviews.org/>
- **PeeringDB** — peering policy registry. <https://www.peeringdb.com>
- **ICANN Root Server System Advisory Committee (RSSAC)** — root-server reports, instance counts. <https://www.icann.org/groups/rssac>
- **Renesys / Oracle BGP analyses** — historical BGP incident write-ups. <https://blogs.oracle.com/internetintelligence/>
- **Renesys — "Anycast: A Practical Look"** (2007) — early empirical study of root-server anycast. (See archived Renesys blog via Oracle.)
- **NLNOG ring** — community looking-glass network. <https://ring.nlnog.net>
- **Hurricane Electric BGP toolkit** — public LG, prefix lookup, AS info. <https://bgp.he.net>
- **RIPE-580** — recommendations on route flap damping. <https://www.ripe.net/publications/docs/ripe-580>
