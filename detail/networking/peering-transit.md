# The Economics and Engineering of Internet Interconnection

> *Peering and transit are not purely technical decisions — they are economic negotiations shaped by traffic flows, market power, and game theory. Understanding the math behind interconnection is essential for any network making peering decisions.*

---

## 1. Internet Interconnection Economics

### The Two Fundamental Models

The internet's physical topology is held together by two types of commercial relationships:

**Transit:** A customer pays a provider for access to the entire internet routing table. The provider agrees to carry the customer's traffic to any destination and to announce the customer's prefixes to all of its own peers and upstreams. Transit is a hierarchical relationship — the customer is below the provider in the routing hierarchy.

**Peering:** Two networks agree to exchange traffic destined to each other's customers (and typically their customers' customers). Neither party pays the other — this is "settlement-free interconnection." Peering is a lateral relationship between networks of roughly comparable size or mutual benefit.

The key economic distinction: transit provides **global reachability** while peering provides **direct reachability** only to the peer's customer cone.

### The Cost Structure

Transit costs are measured in dollars per megabit per second per month ($/Mbps/mo). Historical pricing:

| Year | Typical Transit Price ($/Mbps/mo) | Decline Rate |
|:---:|:---:|:---:|
| 2005 | ~$20 | — |
| 2010 | ~$5 | ~75% |
| 2015 | ~$1.50 | ~70% |
| 2020 | ~$0.50 | ~67% |
| 2025 | ~$0.15–$0.30 | ~50% |

Transit pricing follows a roughly exponential decay, but the rate of decline has slowed. As prices approach marginal cost of infrastructure, further compression becomes harder.

Peering costs are not zero — they include:

- **Port fees** at IXPs (monthly, based on port speed: 1G/10G/100G)
- **Cross-connect fees** at colocation facilities
- **Transport costs** to reach IXP or PNI locations
- **Router port costs** (optics, line cards)
- **Engineering time** to establish and maintain sessions

The decision to peer vs buy transit is a breakeven analysis:

$$\text{Peer if: } C_{\text{peer}} < V_{\text{traffic}} \times P_{\text{transit}}$$

Where:
- $C_{\text{peer}}$ = total monthly cost of peering (port + transport + colo + engineering amortized)
- $V_{\text{traffic}}$ = volume of traffic exchangeable via peering (in Mbps, 95th percentile)
- $P_{\text{transit}}$ = transit price per Mbps per month

### Worked Example

A content network sends 2 Gbps toward AS 65002's customers. Current transit cost is $0.25/Mbps/mo.

Transit cost for that traffic:

$$C_{\text{transit}} = 2000 \text{ Mbps} \times \$0.25 = \$500/\text{mo}$$

Peering cost at an IXP:

| Component | Monthly Cost |
|:---|:---:|
| 10G IXP port | $500 |
| Cross-connect | $300 |
| Transport to IXP | $800 |
| Router optics (amortized) | $100 |
| **Total** | **$1,700** |

At $1,700/mo peering cost vs $500/mo transit savings, peering is **not** justified on pure cost. But if the network also peers with 20 other networks at the same IXP, the shared costs (transport, port) are amortized:

$$C_{\text{marginal}} = \frac{\$1,700}{20 \text{ peers}} = \$85/\text{peer}}$$

Now each peer saves $500/mo in transit for $85/mo in cost — a clear win.

This is why **IXPs create peering density** — the more networks present, the lower the marginal cost of each additional peering session.

---

## 2. Coasian Bargaining and Peering Negotiations

### The Coase Theorem Applied to Peering

Ronald Coase's theorem states that if transaction costs are sufficiently low and property rights are well-defined, parties will negotiate to an efficient outcome regardless of the initial allocation of rights. Applied to peering:

- Two networks each "own" their portion of the traffic exchange
- If the transaction cost of peering (technical setup, negotiation) is low relative to the value of direct interconnection, they will peer
- The "efficient outcome" is direct interconnection when it reduces total network costs

However, Coasian bargaining breaks down when:

1. **Information asymmetry:** Networks may not disclose true traffic volumes or costs
2. **Market power:** A dominant network can refuse to peer, forcing the smaller network to buy transit (which may transit through the dominant network anyway)
3. **Bundling:** Large networks bundle peering across multiple locations, creating take-it-or-leave-it dynamics

### Traffic Ratio Analysis

Most networks with selective peering policies require traffic ratios between 1:1 and 2:1 (or sometimes 3:1). The ratio is:

$$R = \frac{\max(\text{inbound}, \text{outbound})}{\min(\text{inbound}, \text{outbound})}$$

A content-heavy network (CDN, streaming service) sends far more than it receives, creating ratios of 10:1 or higher. An eyeball network (residential ISP) receives far more than it sends. Neither wants to peer with the other for free — the eyeball network argues it provides "eyeballs" (valuable destination), while the content network argues it provides "content" (what eyeballs want).

This fundamental tension between content and eyeball networks is why paid peering exists. When settlement-free peering breaks down, one party (usually the content network) pays the other for direct interconnection.

### The Nash Bargaining Solution

Two networks negotiate peering. Each has a threat point (BATNA — best alternative to negotiated agreement), which is the cost of routing traffic via transit instead. The Nash bargaining solution predicts:

$$\text{Payment} = \frac{(V_A - d_A) + (V_B - d_B)}{2}$$

Where $V_X$ is the value of peering to network $X$ and $d_X$ is the disagreement payoff (transit cost). If both benefit equally and have equal BATNAs, the payment is zero — settlement-free peering. When benefits are asymmetric, the advantaged party receives payment.

---

## 3. IXP Architectures

### Layer 2 IXP Design

The dominant IXP model is a shared Layer 2 switching fabric. Members connect via Ethernet ports and exchange BGP sessions across a shared peering LAN.

**Single-switch model:** Simple, low cost. Single point of failure. Suitable for small/regional IXPs.

**Redundant fabric:** Dual independent switches (or switch clusters) with members connected to both. LACP or active/standby. Most production IXPs use this.

**Distributed fabric:** Multiple PoPs connected via dark fiber or DWDM. Appears as a single L2 domain to members. Used by large IXPs (DE-CIX, AMS-IX) spanning a metro area or even multiple cities.

Key design considerations:

- **MAC address management:** Large IXPs may have hundreds of members. ARP/ND storms can be problematic. Solutions include ARP sponge, proxy ARP, or limiting broadcast domains
- **VLAN design:** Typically a single peering VLAN for IPv4 and IPv6. Some IXPs offer private VLANs for bilateral peering or specific services
- **MTU:** Most IXPs support jumbo frames (9000+ bytes) on the peering LAN to avoid fragmentation for tunneled traffic
- **Security:** Port security (MAC limiting), DHCP snooping, storm control, anti-spoofing filters

### Route Server Operation

A route server (RS) is a BGP speaker operated by the IXP that acts as a transparent route reflector for its members. Without a route server, $N$ members need $\frac{N(N-1)}{2}$ bilateral sessions. With a route server, each member needs only 1 session (or 2 for redundant RS).

**Route server mechanics:**

1. Members send their routes to the RS
2. RS applies per-member import/export policies (based on IRR and RPKI)
3. RS re-advertises routes to other members with the original next-hop preserved
4. RS does **not** appear in the AS-path — it is transparent

**Security considerations:**

- RS must validate routes using RPKI (drop invalid origins)
- RS should filter based on IRR data (only accept prefixes registered to the member's AS-SET)
- RS must not modify next-hop — each member's router must be able to ARP/ND for the original next-hop on the peering LAN
- RS should support RFC 9234 (BGP roles) to detect route leaks

**RS software:** Most IXPs use BIRD or OpenBGPD for route servers. Both support transparent RS mode with per-peer filtering.

**Limitations:**

- RS-learned routes have slightly higher latency (route propagation delay through RS)
- Members cannot apply per-peer traffic engineering (communities on RS sessions affect all RS-learned routes)
- Some networks refuse to peer via RS for policy or security reasons

---

## 4. BGP Community Taxonomy

### RFC 1997 — Standard Communities

Standard communities are 32-bit values, conventionally rendered as two 16-bit values separated by a colon: `ASN:Value`.

**Well-known communities:**

| Community | Meaning |
|:---|:---|
| `NO_EXPORT` (65535:65281) | Do not advertise outside the local AS (or confederation) |
| `NO_ADVERTISE` (65535:65282) | Do not advertise to any peer |
| `NO_EXPORT_SUBCONFED` (65535:65283) | Do not advertise outside the local confederation sub-AS |
| `BLACKHOLE` (65535:666) | RFC 7999 — request upstream to blackhole this prefix |

**Operator-defined community schemes** typically follow a taxonomy:

| Community Pattern | Meaning |
|:---|:---|
| `ASN:1XX` | Origin tagging (where route was learned) |
| `ASN:2XX` | Peer type (transit, peer, customer, IXP) |
| `ASN:3XX` | Geographic tagging (region/city codes) |
| `ASN:4XX` | Action communities (prepend, no-export to specific peer) |
| `ASN:5XX` | Blackhole and traffic engineering |
| `ASN:9XX` | Informational (do not act on) |

### RFC 8092 — Large Communities

Large communities use three 32-bit fields: `ASN:Function:Parameter`. This solves the 16-bit ASN limitation of standard communities and provides a structured namespace:

| Field | Purpose |
|:---|:---|
| ASN | The network defining/acting on the community (always 32-bit safe) |
| Function | Category of action (0=informational, 1=origin, 2=no-export, 3=prepend, etc.) |
| Parameter | Specific target or value (peer ASN, location code, prepend count) |

Example scheme for AS 394500:

| Community | Meaning |
|:---|:---|
| `394500:0:1` | Informational: learned from transit |
| `394500:0:2` | Informational: learned from peer |
| `394500:0:3` | Informational: learned from customer |
| `394500:1:174` | Do not advertise to AS 174 |
| `394500:2:0` | Do not advertise to any transit |
| `394500:3:1` | Prepend 1x to all peers |
| `394500:3:2` | Prepend 2x to all peers |
| `394500:4:174` | Prepend 1x toward AS 174 |

### Community-Based Traffic Engineering

Traffic engineering with communities works by signaling intent to upstream providers. The customer attaches communities to outbound announcements, and the provider's policy engine acts on them.

**Inbound traffic engineering** (controlling how traffic enters your network):

1. Attach prepend communities to make a path less attractive via a specific upstream
2. Attach no-export communities to prevent a prefix from being announced to certain peers
3. Use selective announcement — only announce specific prefixes to specific upstreams

**Outbound traffic engineering** (controlling how traffic leaves your network):

1. Set local-preference based on communities attached by upstream (peer vs transit, location)
2. Use community-based route filtering in policy
3. Tag routes at ingress with origin communities; use those tags in egress policy

The key insight: communities are the **control plane** for inter-domain traffic engineering. Without them, the only tools are AS-path prepending and selective announcement — far less granular.

---

## 5. RPKI Deployment Analysis

### How RPKI Works

RPKI (Resource Public Key Infrastructure) is a cryptographic system that binds IP address prefixes to authorized origin ASNs. The chain of trust:

1. **IANA** delegates address space to **RIRs** (ARIN, RIPE, APNIC, AFRINIC, LACNIC)
2. RIRs issue **certificates** to address holders
3. Address holders create **ROAs** (Route Origin Authorizations) — signed objects saying "AS X is authorized to originate prefix Y with max-length Z"
4. **Relying parties** (validators like Routinator, Fort, OctoRPKI) fetch ROAs from RIR publication points and generate a Validated ROA Payload (VRP) table
5. Routers query validators via the RTR protocol and apply origin validation to BGP routes

### Validation States

For each BGP route (prefix, origin AS):

| State | Condition | Recommended Action |
|:---|:---|:---|
| Valid | A VRP matches the prefix and origin AS, and prefix length is within max-length | Accept, prefer (higher local-pref) |
| Invalid | A VRP exists for the prefix but origin AS or length does not match | **Drop** |
| NotFound | No VRP covers the prefix | Accept with lower preference |

### The Max-Length Problem

ROA max-length is a security-sensitive parameter. Consider a ROA:

```
Prefix: 203.0.113.0/24
Origin AS: 65001
Max-Length: /24
```

This says AS 65001 can originate exactly /24. If an attacker announces 203.0.113.0/25 from AS 65001, it would be Invalid (length exceeds max-length for that AS). Good.

But if the ROA had `Max-Length: /28`:

```
Prefix: 203.0.113.0/24
Origin AS: 65001
Max-Length: /28
```

Now any sub-prefix from /24 to /28 is Valid for AS 65001. If the legitimate origin only announces /24, an attacker who hijacks AS 65001 (or exploits a route leak) could announce more-specific /25s or /28s that would be Valid. The more-specifics would win longest-match and attract traffic.

**Best practice:** Set max-length equal to the prefix length you actually announce. Only increase it if you genuinely announce more-specifics.

### Deployment Coverage

As of 2025, RPKI deployment varies significantly:

- **ROA coverage:** ~55-60% of IPv4 prefixes have ROAs, ~45% of IPv6
- **Validation enforcement:** Major transit providers (NTT, Lumen, Cogent, Telia) drop RPKI-invalid routes
- **Regional variation:** RIPE region has highest ROA coverage (~75%); ARIN region lags (~40%)

The "chicken and egg" problem: networks hesitate to create ROAs (effort, fear of misconfiguration) until enough networks enforce validation; networks hesitate to drop invalid routes until enough prefixes have ROAs.

---

## 6. IRR Data Quality and Automation

### The IRR Ecosystem

The Internet Routing Registry is a distributed set of databases containing routing policy objects. The key object types:

| Object | Purpose | Example |
|:---|:---|:---|
| `route` | Binds a prefix to an origin AS | `route: 203.0.113.0/24` + `origin: AS65001` |
| `route6` | Same for IPv6 | `route6: 2001:db8::/32` + `origin: AS65001` |
| `aut-num` | Documents an AS's routing policy | Import/export rules |
| `as-set` | Named set of ASNs (can be nested) | `AS-EXAMPLE: AS65001, AS65002, AS-DOWNSTREAM` |
| `mntner` | Maintainer object controlling who can update | Authentication for changes |

### Data Quality Issues

IRR databases suffer from well-known problems:

1. **Stale data:** Networks create route objects when announcing prefixes but rarely clean up when they stop. Studies show 30-40% of RADB entries are stale.
2. **No validation:** RADB (the largest open IRR) does not verify that the registrant actually holds the address space. Anyone can register any route object.
3. **Inconsistency:** The same prefix may have conflicting route objects in different IRR databases.
4. **AS-SET bloat:** Some AS-SETs contain thousands of members, including ASNs that no longer exist. Recursive expansion can generate enormous prefix-lists.

### Mitigations

- **Use authoritative IRRs:** RIPE, ARIN, APNIC maintain their own IRR databases and validate against their allocation records. Prefer these over RADB when the prefix holder's RIR is known.
- **Cross-reference with RPKI:** Use RPKI ROAs to validate IRR entries. If a route object exists in RADB but no ROA matches, treat with lower confidence.
- **bgpq4 source ordering:** Configure bgpq4 to prefer authoritative sources: `-S RIPE,ARIN,APNIC,RADB`
- **Automated expiry:** Some operators set TTLs on their prefix-lists and regenerate daily. If the IRR entry disappears, the prefix-list entry ages out.

### Peering Automation

Modern peering automation tools integrate IRR, RPKI, and PeeringDB:

1. **peering-manager** — Web application for managing peering sessions, integrates PeeringDB and generates configurations
2. **Alice-LG** — Looking glass for IXP route servers, shows per-peer route filtering results
3. **arouteserver** — Generates BIRD/OpenBGPD configurations for IXP route servers from PeeringDB and IRR data
4. **bgpq4** — CLI tool for generating router filter configurations from IRR data
5. **IRR Explorer** — Web tool for querying and comparing IRR entries across databases

The automation pipeline:

```
PeeringDB (discover peers)
    → IRR (get AS-SET, resolve prefixes)
    → bgpq4 (generate prefix-lists)
    → RPKI (validate origins)
    → Config generation (push to routers)
    → Monitoring (verify routes match expectations)
```

---

## 7. Traffic Engineering Deep Dive

### The Fundamental Challenge

A multi-homed network has limited control over inbound traffic. The internet's routing system makes forwarding decisions at each hop independently. You can influence — but not dictate — which path traffic takes to reach you.

**Outbound:** Full control via local-preference and other BGP attributes. You choose which exit to use for each destination.

**Inbound:** Indirect control via:
- **Selective announcement:** Only announce prefixes to certain upstreams
- **AS-path prepending:** Make a path appear longer (less attractive) via a specific upstream
- **Communities:** Signal actions to upstream providers
- **MED:** Influence path selection within a single upstream's network (when the upstream compares MEDs)
- **More-specifics:** Announce /25s via one upstream and the /24 cover via another (effective but consumes global routing table slots)

### Prepending Mathematics

AS-path prepending adds copies of your own ASN to the AS-path, making the path appear longer. The effectiveness depends on what alternative paths exist.

Consider a prefix announced via two upstreams with no prepending:

```
Path A: [65001 3356] (length 2)
Path B: [65001 174]  (length 2)
```

Equal-length paths — tie-breaking decides. Adding 2 prepends on path B:

```
Path A: [65001 3356]            (length 2)
Path B: [65001 65001 65001 174] (length 4)
```

Now path A is clearly shorter. But the effectiveness of prepending diminishes rapidly:

- 1 prepend: Most traffic shifts away
- 2 prepends: Nearly all traffic shifts
- 3 prepends: Marginal additional shift
- 4+ prepends: Essentially zero additional effect; just clutters the global routing table

**Why diminishing returns?** Because alternative paths through the prepended upstream also get longer (prepending affects all paths through that upstream, not just the direct one). After 2-3 prepends, the prepended path is already longer than any reasonable alternative.

**Caveat:** Prepending does not work against local-preference. If a remote network sets local-preference to prefer a path, no amount of prepending will override it.

### MEDs and Their Limitations

MED (Multi-Exit Discriminator) tells a neighbor "if you have multiple links to me, prefer this one." It is:

- **Non-transitive:** Only compared between paths from the same neighbor AS
- **Lower is better:** MED 100 beats MED 200
- **Optional:** Many networks ignore MEDs entirely
- **Deterministic comparison only with `bgp always-compare-med`:** Without this, paths from different ASes never compare MEDs, which can lead to non-deterministic path selection depending on arrival order

---

## 8. Settlement-Free Peering — Game Theory

### The Prisoner's Dilemma of Depeering

When one party wants to depeer (terminate a peering relationship), the dynamics resemble a game-theoretic problem:

| | Network B Continues Peering | Network B Depeers |
|:---|:---|:---|
| **Network A Continues** | Both save transit costs (cooperative equilibrium) | A loses direct path; B's customers lose direct access to A |
| **Network A Depeers** | B loses direct path; A's customers lose direct access to B | Both pay transit for traffic that was free (worst outcome) |

The Nash equilibrium depends on who benefits more from peering. If Network A has 10x the eyeballs, Network B needs A more than A needs B — giving A leverage to demand paid peering or better terms.

### Peering Disputes and Resolution

Historical depeering disputes (Cogent/Sprint 2008, Netflix/Comcast 2014, Cogent/Google 2024) follow a pattern:

1. One party announces dissatisfaction with terms (traffic ratio, congestion)
2. Congestion builds on shared links as neither party upgrades
3. Quality degrades for end users
4. Regulatory/public pressure forces resolution (or paid peering agreement)

The resolution mechanism is typically market-driven: the party whose customers complain loudest has the most incentive to concede.

---

## 9. Peering at Scale

### The Decision Framework

A structured approach to peering decisions:

1. **Identify top traffic destinations** by AS (NetFlow/sFlow analysis)
2. **Calculate per-AS transit cost** using the 95th percentile billing model
3. **Check PeeringDB** for each target AS: are they present at IXPs where you have ports? What is their peering policy?
4. **Estimate peering cost** (marginal if IXP port already exists, or full if new IXP/PNI)
5. **Compare:** if $C_{\text{peer}} < C_{\text{transit}}$, initiate peering request
6. **Consider non-monetary benefits:** latency reduction, path diversity, reduced transit dependency

### Capacity Planning

Peering port utilization guidelines:

| Utilization | Action |
|:---:|:---|
| < 30% | Normal operation |
| 30-50% | Begin planning upgrade |
| 50-70% | Order upgrade; schedule migration |
| > 70% | Congestion risk; emergency upgrade |

For PNI (private network interconnect), the thresholds are typically higher (up to 50% normal) because the link is dedicated and there is no oversubscription risk from other members.

**Traffic growth forecasting:** Internet traffic grows at roughly 25-30% annually. A link at 40% today will hit 70% in approximately:

$$t = \frac{\ln(0.70/0.40)}{\ln(1.25)} = \frac{\ln(1.75)}{0.223} = \frac{0.559}{0.223} \approx 2.5 \text{ years}$$

Plan capacity upgrades 6-12 months before projected congestion.

---

## See Also

- bgp, bgp-advanced, mpls, ipv4, ipv6

## References

- [RFC 7999 — BLACKHOLE Community](https://www.rfc-editor.org/rfc/rfc7999)
- [RFC 1997 — BGP Communities Attribute](https://www.rfc-editor.org/rfc/rfc1997)
- [RFC 8092 — BGP Large Communities](https://www.rfc-editor.org/rfc/rfc8092)
- [RFC 6811 — BGP Prefix Origin Validation](https://www.rfc-editor.org/rfc/rfc6811)
- [RFC 7454 — BGP Operations and Security](https://www.rfc-editor.org/rfc/rfc7454)
- [RFC 5765 — Security of the Internet Routing Infrastructure](https://www.rfc-editor.org/rfc/rfc5765)
- [Norton, W.B. — "The Internet Peering Playbook"](https://drpeering.net/core/bookOutline.html)
- [MANRS — Mutually Agreed Norms for Routing Security](https://www.manrs.org/)
- [Euro-IX — IXP Technical Standards](https://www.euro-ix.net/)
- [RIPE NCC — RPKI Documentation](https://www.ripe.net/manage-ips-and-asns/resource-management/rpki/)
