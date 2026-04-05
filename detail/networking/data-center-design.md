# Data Center Design — Network Architecture Principles and Topologies

> *Data center network design has evolved from spanning-tree-dependent 3-tier architectures to non-blocking Clos fabrics. The mathematics of oversubscription, bisection bandwidth, and ECMP hashing govern how traffic flows between tens of thousands of servers. Understanding these principles — from the original Clos paper to modern VXLAN EVPN overlays — is the foundation of every DC network engineer's practice.*

---

## 1. The Traditional 3-Tier Architecture

### Historical Context

The 3-tier model (core, aggregation, access) dominated data center design from the late 1990s through the early 2010s. It was inherited from campus network design, where traffic was predominantly **north-south** (clients outside the DC accessing servers inside).

### Layer Responsibilities

**Core layer:** High-speed backbone connecting aggregation blocks. Typically a pair of chassis switches (e.g., Cisco Catalyst 6500, Nexus 7000) running OSPF or IS-IS with fast convergence. The core performs pure L3 forwarding — no policy, no access lists, no services.

**Aggregation layer:** The policy enforcement point. This is where L2 meets L3 — VLANs terminate here, and inter-VLAN routing happens. Aggregation switches run STP root, enforce ACLs, and connect to firewalls, load balancers, and other service appliances. The aggregation layer defines the **pod** (point of delivery) — a self-contained unit of access switches sharing a pair of aggregation switches.

**Access layer:** Server connectivity. Top-of-rack or end-of-row switches providing 1G/10G ports to servers. VLANs are assigned here. Access switches are L2-only (or L2 with minimal L3 for management).

### The Spanning Tree Problem

The fundamental limitation of 3-tier is **Spanning Tree Protocol (STP)**. STP prevents L2 loops by blocking redundant links, which means only one of two parallel paths carries traffic at any time.

For a network with $N$ redundant links, the available bandwidth is:

$$BW_{effective} = \frac{BW_{total}}{2}$$

This 50% waste is compounded at each tier. With dual links at each layer:

$$BW_{usable} = BW_{link} \times 1 \quad \text{(per pair, one active, one standby)}$$

**Convergence time** adds insult to injury. Classic STP takes 30-50 seconds to reconverge after a failure. RSTP (802.1w) improves this to 1-3 seconds, but the fundamental bandwidth waste remains.

### Oversubscription in 3-Tier

Oversubscription compounds across tiers:

$$R_{total} = R_{access} \times R_{aggregation} \times R_{core}$$

If access is 3:1, aggregation is 4:1, and core is 2:1:

$$R_{total} = 3 \times 4 \times 2 = 24:1$$

This means the core provides only 1/24th of the aggregate server bandwidth — acceptable for north-south-dominant traffic in 2005, catastrophic for east-west-dominant traffic in 2025.

---

## 2. The Clos Topology — Mathematical Foundations

### Charles Clos and the Bell Labs Paper (1953)

Charles Clos published "A Study of Non-Blocking Switching Networks" in the Bell System Technical Journal in 1953. His insight was that a multi-stage network of small switches could provide the same non-blocking capacity as a single enormous crossbar, at far lower cost.

### Formal Definition

A 3-stage Clos network is defined by three parameters $(m, n, r)$:

- $n$ = number of input (and output) ports on each ingress/egress switch
- $m$ = number of middle-stage switches
- $r$ = number of ingress (and egress) switches

The total number of input ports is:

$$N = n \times r$$

### Non-Blocking Conditions

**Strictly non-blocking** (any new connection can be routed without rearranging existing connections):

$$m \geq 2n - 1$$

**Rearrangeably non-blocking** (any set of connections can be routed, but some existing connections may need to be rearranged):

$$m \geq n$$

**Proof sketch for strict non-blocking:** In the worst case, an input port has $n-1$ other ports on its ingress switch already connected, each consuming a different middle-stage switch. Similarly, the destination output port has $n-1$ other ports on its egress switch consuming middle-stage switches. In the worst case, these are all different middle-stage switches:

$$\text{Middle switches consumed} = (n-1) + (n-1) = 2n - 2$$

So we need at least $2n - 1$ middle switches to guarantee a free path.

### Crossbar Comparison

A single crossbar switch with $N$ inputs and $N$ outputs requires:

$$\text{Crosspoints}_{crossbar} = N^2$$

A 3-stage Clos with $m = 2n - 1$ requires:

$$\text{Crosspoints}_{Clos} = 2Nm + m \times r^2 = 2Nr + (2n-1) \times r^2$$

For large $N$, the Clos network requires $O(N^{3/2})$ crosspoints versus $O(N^2)$ for a crossbar. This cost reduction is why every large data center uses Clos topology.

### The Folded Clos

In a telephone network, ingress and egress switches are separate (calls go in one side and out the other). In a data center, servers both send and receive, so the ingress and egress stages are **folded** into the same physical switches — the **leaf switches**.

A folded 3-stage Clos with $S$ spines and $L$ leaves, where each leaf has $P_{down}$ server-facing ports and $S$ spine-facing ports:

$$N_{servers} = L \times P_{down}$$

$$BW_{bisection} = L \times S \times BW_{uplink}$$

The bisection bandwidth is the minimum bandwidth available across any cut that divides the servers into two equal halves. In a non-blocking Clos, the bisection bandwidth equals the aggregate server bandwidth.

---

## 3. Spine-Leaf — The Modern DC Fabric

### Design Principles

The spine-leaf architecture is a practical implementation of the folded 3-stage Clos:

1. **Every leaf connects to every spine** (full mesh between tiers)
2. **No leaf-to-leaf links** (all inter-leaf traffic traverses exactly one spine)
3. **No spine-to-spine links** (spines are independent; adding spines adds bandwidth)
4. **Equal-cost paths** (every leaf-to-leaf path is exactly 2 hops: leaf -> spine -> leaf)
5. **L3 to the leaf** (all links are routed, no STP)

### Bandwidth Calculations

For a fabric with $L$ leaves, $S$ spines, leaf uplink bandwidth $BW_u$, and leaf downlink bandwidth $BW_d$:

**Total southbound bandwidth per leaf:**

$$BW_{south} = P_{down} \times BW_d$$

**Total northbound bandwidth per leaf:**

$$BW_{north} = S \times BW_u$$

**Oversubscription ratio:**

$$R = \frac{BW_{south}}{BW_{north}} = \frac{P_{down} \times BW_d}{S \times BW_u}$$

**Worked example:** 48x25G downlinks, 8x100G uplinks:

$$R = \frac{48 \times 25}{8 \times 100} = \frac{1200}{800} = 1.5:1$$

**Bisection bandwidth** (total bandwidth across the spine tier):

$$BW_{bisection} = \frac{L \times S \times BW_u}{2}$$

For 64 leaves, 8 spines, 100G uplinks:

$$BW_{bisection} = \frac{64 \times 8 \times 100}{2} = 25,600 \text{ Gbps} = 25.6 \text{ Tbps}$$

### Maximum Fabric Size

The maximum number of leaves is constrained by spine port density:

$$L_{max} = \min(\text{Spine ports}, \text{desired fabric size})$$

With 64-port spine switches and one port per leaf:

$$L_{max} = 64 \text{ leaves} \times 48 \text{ servers/leaf} = 3,072 \text{ servers}$$

To exceed this, move to a 5-stage Clos (super-spine) or use higher-radix switches.

### 5-Stage Clos (Super-Spine)

When a single spine tier cannot support enough leaves, add a super-spine layer:

$$\text{Pods} = P, \quad \text{Leaves/pod} = L_p, \quad \text{Spines/pod} = S_p, \quad \text{Super-spines} = SS$$

Total servers:

$$N_{servers} = P \times L_p \times P_{down}$$

With 8 pods of 64 leaves each, 48 servers per leaf:

$$N_{servers} = 8 \times 64 \times 48 = 24,576$$

The inter-pod oversubscription depends on how many spine ports face the super-spine tier versus the leaf tier. Typical designs allocate half of spine ports downward (to leaves) and half upward (to super-spines).

---

## 4. East-West vs North-South Traffic Analysis

### Historical Shift

In 2005, a typical enterprise DC had 80% north-south traffic (users accessing web/app/DB servers) and 20% east-west (inter-server). By 2015, this had inverted. By 2025, east-west traffic dominates at 70-90% in most modern DCs.

### Drivers of East-West Growth

**Microservices:** A single user request may trigger 50-200 internal service-to-service calls, all east-west. In a monolithic application, those calls were function calls within a single server — zero network traffic.

**Storage replication:** A 3-way replica write to a distributed storage system generates 2 additional east-west copies. Erasure coding (e.g., Reed-Solomon 6+3) generates even more cross-server traffic.

**ML/AI training:** Distributed training with data parallelism requires all-reduce operations across GPU servers. For a ring all-reduce with $N$ workers and model size $M$:

$$\text{East-west data per iteration} = 2 \times M \times \frac{N-1}{N}$$

With 64 GPUs and a 10B parameter model (40 GB in FP32):

$$\text{Data per iteration} = 2 \times 40 \times \frac{63}{64} \approx 78.75 \text{ GB}$$

At 100 iterations per second, that is 7.87 TB/s of pure east-west traffic.

**Big data / MapReduce:** The shuffle phase of MapReduce sends every mapper's output to every reducer, generating $O(M \times R)$ east-west flows where $M$ is mapper count and $R$ is reducer count.

### Impact on Architecture

The 3-tier model forced east-west traffic to **hairpin** through the aggregation and core layers:

```
Server A (Rack 1) -> Access -> Aggregation -> Core -> Aggregation -> Access -> Server B (Rack 2)
```

This is 6 hops with STP blocking half the links. In spine-leaf:

```
Server A (Leaf 1) -> Spine -> Leaf 2 -> Server B
```

This is 3 hops with ECMP using all links. The path length reduction and bandwidth utilization improvement is the primary reason for the architectural shift.

### Quantifying the Benefit

For $F$ flows of east-west traffic in a 3-tier with oversubscription $R_3$ versus spine-leaf with oversubscription $R_{SL}$:

$$\text{Effective east-west BW}_{3\text{-tier}} = \frac{BW_{link}}{R_3 \times 2} \quad \text{(STP blocks half)}$$

$$\text{Effective east-west BW}_{SL} = \frac{BW_{link}}{R_{SL}} \quad \text{(ECMP uses all paths)}$$

Improvement factor:

$$\frac{BW_{SL}}{BW_{3\text{-tier}}} = \frac{R_3 \times 2}{R_{SL}}$$

With $R_3 = 24:1$ (compounded) and $R_{SL} = 1.5:1$:

$$\text{Improvement} = \frac{24 \times 2}{1.5} = 32\times$$

---

## 5. ECMP in Spine-Leaf Fabrics

### Why ECMP is Fundamental

In a spine-leaf fabric with $S$ spines, every leaf has $S$ equal-cost paths to every other leaf. ECMP (Equal-Cost Multi-Path) distributes flows across all $S$ paths, achieving $S \times BW_{uplink}$ of aggregate bandwidth.

Without ECMP, only one path would be used (like STP), and the fabric would have $1/S$th of its potential bandwidth.

### Hash-Based Path Selection

For a flow $f$ with 5-tuple $(src\_ip, dst\_ip, src\_port, dst\_port, proto)$:

$$\text{Spine index} = H(f) \mod S$$

Where $H$ is a hash function (CRC16, CRC32, or XOR-based on hardware platforms).

### Flow Distribution Quality

With $F$ flows and $S$ spines, the expected flows per spine:

$$E[\text{flows per spine}] = \frac{F}{S}$$

The standard deviation (assuming uniform hashing):

$$\sigma = \sqrt{\frac{F \times (S-1)}{S^2}} \approx \sqrt{\frac{F}{S}}$$

The coefficient of variation (measure of imbalance):

$$CV = \frac{\sigma}{E} = \sqrt{\frac{S-1}{F}} \approx \frac{1}{\sqrt{F/S}}$$

For 1,000 flows across 8 spines:

$$CV = \sqrt{\frac{7}{1000}} = 0.084 = 8.4\%$$

This means spine loads will vary by about 8.4% from the mean. More flows = better balance.

### Polarization

If spine switches and leaf switches use the same hash function with the same seed, traffic that was grouped together at one stage stays grouped at the next — this is **polarization**.

In a 2-tier fabric, polarization doesn't apply (traffic is hashed once at the leaf). In a 3-tier or 5-stage Clos with super-spines, polarization between the spine and super-spine tiers can reduce effective paths:

$$\text{Effective paths without depolarization} = S$$
$$\text{Effective paths with polarization} = \min(S_{spine}, S_{super}) \text{ in worst case}$$

**Solution:** Use different hash seeds at each tier, or use resilient hashing with per-tier entropy injection.

### Resilient Hashing

Standard modular hashing causes a **full rehash** when a spine fails or is added:

$$\text{Disrupted flows (modular)} = F \times \frac{S-1}{S}$$

For 10,000 flows and 8 spines, removing one spine disrupts:

$$10000 \times \frac{7}{8} = 8,750 \text{ flows}$$

Resilient (consistent) hashing limits disruption to approximately:

$$\text{Disrupted flows (resilient)} \approx \frac{F}{S}$$

$$\frac{10000}{8} = 1,250 \text{ flows}$$

This 7x reduction in disruption is critical for latency-sensitive workloads.

---

## 6. VXLAN EVPN Overlay Design

### Why an Overlay

The underlay (physical spine-leaf fabric) is pure L3 — every link is a routed /31 or unnumbered interface. This eliminates STP, enables ECMP, and simplifies the forwarding table. But applications often need L2 adjacency (same broadcast domain) across different racks.

VXLAN provides virtual L2 segments over the L3 underlay. EVPN provides a BGP-based control plane to distribute MAC/IP bindings, eliminating flood-and-learn.

### Encapsulation Overhead

Each VXLAN-encapsulated frame adds:

$$\text{Overhead} = 50 \text{ bytes (outer Ethernet 14 + outer IP 20 + UDP 8 + VXLAN 8)}$$

With a standard 1500-byte MTU on the underlay:

$$\text{Effective MTU} = 1500 - 50 = 1450 \text{ bytes}$$

To avoid fragmentation, set the underlay MTU to 9216 (jumbo frames):

$$\text{Effective MTU}_{jumbo} = 9216 - 50 = 9166 \text{ bytes}$$

### EVPN Route Types and Their Purpose

**Type 2 (MAC/IP Advertisement):** The workhorse. When a host sends an ARP or traffic through a leaf's VTEP, the leaf advertises the host's MAC and IP in a BGP EVPN Type 2 route. All other VTEPs in the same VNI learn this binding without flooding.

Scaling: With $H$ hosts across $L$ leaves, the BGP table holds $H$ Type 2 routes. Each route is approximately 100 bytes in BGP, so:

$$\text{BGP table size} = H \times 100 \text{ bytes}$$

For 100,000 hosts: $\approx$ 10 MB — trivial for modern route reflectors.

**Type 3 (Inclusive Multicast):** One per VTEP per VNI. With $V$ VNIs and $L$ VTEPs:

$$\text{Type 3 routes} = V \times L$$

**Type 5 (IP Prefix):** For inter-subnet routing. Allows a VTEP to advertise entire subnets, not just individual hosts. Essential for scaling when host count is very large.

### Symmetric vs Asymmetric IRB

**Asymmetric IRB:** The ingress leaf performs both L2 lookup (destination MAC) and L3 routing (inter-subnet), then encapsulates toward the destination VTEP. The egress leaf only performs L2 forwarding. This requires every leaf to have every VNI configured (since the ingress leaf must know the destination VNI's subnet).

$$\text{VNIs per leaf}_{asymmetric} = V_{total}$$

**Symmetric IRB:** Both ingress and egress leaves perform L3 routing. Traffic is encapsulated in a **transit L3 VNI** for the inter-subnet hop. Each leaf only needs the VNIs for locally attached hosts.

$$\text{VNIs per leaf}_{symmetric} = V_{local} + 1 \text{ (transit VNI)}$$

For 500 VNIs with 10 locally attached per leaf:

- Asymmetric: 500 VNIs per leaf
- Symmetric: 11 VNIs per leaf

Symmetric IRB is strongly preferred at scale.

### Distributed Anycast Gateway

In EVPN symmetric IRB, every leaf advertises the same gateway IP and MAC (anycast) for each subnet. When a server ARPs for its default gateway, the local leaf responds — no traffic crosses the fabric for gateway resolution.

$$\text{Gateway MAC} = \text{same on all leaves (e.g., 00:00:5e:00:01:01)}$$
$$\text{Gateway IP} = \text{same on all leaves (e.g., 10.1.1.1/24)}$$

This enables seamless VM mobility: a VM moved to a different leaf finds the same gateway MAC and IP.

---

## 7. DC Interconnect (DCI)

### The L2 Stretch Problem

Stretching VLANs across data centers is technically possible (via VXLAN, OTV, or VPLS) but operationally dangerous:

- **Split brain:** If the DCI link fails, both sites have the same IP subnets active, causing routing black holes
- **Broadcast storms:** BUM traffic floods across the WAN link, consuming expensive bandwidth
- **Failure domain expansion:** A loop or misconfiguration in one site propagates to the other

### EVPN Multi-Site

The preferred DCI approach uses EVPN multi-site:

1. Each site has its own autonomous spine-leaf fabric with local EVPN
2. **Border leaves** (or DCI gateways) peer across the WAN via eBGP EVPN
3. VNI-to-VNI mapping allows different VNI numbering per site
4. Type 5 routes (IP prefixes) are preferred over Type 2 (MAC/IP) across DCI to reduce table size
5. Anycast gateways provide active-active multi-site connectivity

### DCI Bandwidth Sizing

The DCI link carries only inter-site traffic. For a typical deployment:

$$BW_{DCI} = BW_{total} \times R_{intersite}$$

Where $R_{intersite}$ is the fraction of traffic crossing sites (typically 5-15%):

$$BW_{DCI} = 25.6 \text{ Tbps} \times 0.10 = 2.56 \text{ Tbps}$$

This is typically provisioned as multiple 100G or 400G DWDM wavelengths.

---

## 8. Physical Design Considerations

### Top-of-Rack vs End-of-Row

**Top-of-Rack (ToR):** One or two leaf switches per rack. Servers connect with short (1-3m) DAC cables. The most common modern design.

Advantages:
- Short, cheap server cables (DAC at $20-50 vs $200+ for optics)
- Self-contained rack: server + switch move as a unit
- Simple cable management

Disadvantages:
- Many switches to manage (one per rack, 200+ in a large DC)
- Lower spine port utilization (each rack uses one spine port per spine)

**End-of-Row (EoR):** One pair of leaf switches serves 4-8 racks in a row. Servers connect with longer (10-30m) structured cabling through patch panels.

Advantages:
- Fewer switches (1/4 to 1/8 of ToR count)
- Better switch utilization
- Easier to maintain (fewer locations)

Disadvantages:
- Longer cables require SFP+ optics instead of DAC ($200+ per port vs $30)
- Patch panel adds a failure point
- Rack additions require re-cabling

### Cable Selection

| Cable Type | Medium | Max Distance (100G) | Cost/m | Power (per end) | Latency |
|:---|:---|:---:|:---:|:---:|:---:|
| DAC (passive) | Copper twinax | 3-5 m | $5-10 | 0 W | Lowest |
| DAC (active) | Copper twinax | 7-10 m | $10-20 | 0.5-1 W | Low |
| AOC | Fiber (integrated) | 30-100 m | $15-30 | 1-2 W | Low |
| SR4 (multimode) | OM3/OM4 fiber | 70-100 m | $40-80 | 3-4 W | Low |
| LR4 (singlemode) | OS2 fiber | 10 km | $200-500 | 3-4 W | Medium |
| ER4 (singlemode) | OS2 fiber | 40 km | $1000+ | 4-6 W | Higher |
| ZR (singlemode) | OS2 fiber | 80+ km | $3000+ | 15-20 W | Higher |

### Structured Cabling Standards

- **TIA-942:** Data center telecommunications infrastructure standard
- **EN 50173-5:** European structured cabling standard for data centers
- **ISO/IEC 24764:** International DC cabling standard
- Fiber polarity: follow Method A, B, or C consistently; document which method is used
- Minimum bend radius: 10x outer diameter for multimode, 15x for singlemode
- Pull tension: never exceed rated tension (typically 25 lbs for indoor fiber)

---

## 9. Power and Cooling Architecture

### Power Distribution

The power chain in a modern DC:

$$\text{Utility} \rightarrow \text{Transformer} \rightarrow \text{UPS} \rightarrow \text{PDU} \rightarrow \text{Rack PDU} \rightarrow \text{Server PSU}$$

Each conversion introduces efficiency losses:

| Component | Typical Efficiency | Loss |
|:---|:---:|:---:|
| Transformer | 98-99% | 1-2% |
| UPS (double conversion) | 94-97% | 3-6% |
| PDU | 98-99% | 1-2% |
| Rack PDU | 99% | 1% |
| Server PSU (80+ Titanium) | 96% | 4% |

**Cumulative efficiency:**

$$\eta_{total} = 0.985 \times 0.955 \times 0.985 \times 0.99 \times 0.96 = 0.88$$

For every 1 kW consumed by the IT load, 1.14 kW must be delivered from the utility (power chain losses alone, before cooling).

### PUE (Power Usage Effectiveness)

$$PUE = \frac{\text{Total Facility Power}}{\text{IT Equipment Power}}$$

$$\text{Total Facility Power} = \text{IT} + \text{Cooling} + \text{Lighting} + \text{Power losses} + \text{Other}$$

A PUE of 1.5 means that for every watt of IT load, 0.5 watts are consumed by overhead (cooling, power losses, lighting). Industry averages:

| Segment | Average PUE | Best-in-Class PUE |
|:---|:---:|:---:|
| Legacy enterprise | 2.0-2.5 | 1.6 |
| Modern colocation | 1.4-1.6 | 1.2 |
| Hyperscaler | 1.1-1.2 | 1.05 |

### Cooling Capacity Planning

$$Q_{cooling} = P_{IT} \times (PUE - 1) \times \text{safety factor}$$

For a 5 MW IT load with PUE 1.3 and 1.2x safety factor:

$$Q_{cooling} = 5000 \times 0.3 \times 1.2 = 1,800 \text{ kW of cooling capacity}$$

### Rack Power Density Trends

| Workload | Power per Rack | Cooling Method |
|:---|:---:|:---|
| General compute | 5-10 kW | Air (hot/cold aisle) |
| High-density compute | 15-25 kW | In-row + containment |
| GPU training (8x A100) | 30-40 kW | Rear-door heat exchanger |
| GPU training (8x H100) | 40-70 kW | Direct liquid cooling |
| Dense GPU (8x B200) | 70-120 kW | Immersion or direct liquid |

Air cooling reaches practical limits around 25-30 kW per rack. Beyond that, liquid cooling is required.

---

## 10. Uptime Institute Tier Classification

### Tier Requirements

**Tier I — Basic Site Infrastructure:**
- Single path for power and cooling
- No redundant components
- Must shut down entirely for maintenance
- 99.671% availability (28.8 hours downtime/year)

**Tier II — Redundant Site Infrastructure Components:**
- Single path for power and cooling
- Redundant components: N+1 UPS modules, N+1 cooling units
- Maintenance of redundant components without IT shutdown
- 99.741% availability (22.0 hours downtime/year)

**Tier III — Concurrently Maintainable:**
- Multiple (dual) paths for power and cooling, one active
- Redundant components
- Any component can be removed for maintenance without affecting IT load
- 99.982% availability (1.6 hours downtime/year)

**Tier IV — Fault Tolerant:**
- Multiple active power and cooling paths
- Redundant components
- Single fault in any system does not cause IT downtime
- 2N or 2(N+1) redundancy on all critical systems
- 99.995% availability (26.3 minutes downtime/year)

### Availability Mathematics

Availability of a single component with MTBF (Mean Time Between Failures) and MTTR (Mean Time To Repair):

$$A = \frac{MTBF}{MTBF + MTTR}$$

For two components in parallel (redundant):

$$A_{parallel} = 1 - (1 - A_1)(1 - A_2)$$

For two components in series (both required):

$$A_{series} = A_1 \times A_2$$

**Example:** A UPS with 99.9% availability. Two in parallel:

$$A_{2N} = 1 - (1 - 0.999)^2 = 1 - 0.000001 = 0.999999 = 99.9999\%$$

This is why Tier IV (2N redundancy) achieves dramatically higher availability than Tier II (N+1).

### Network Tier Mapping

| Tier | Network Requirements |
|:---|:---|
| I | Single switch/router, single ISP, no redundancy |
| II | Redundant switches (stacked or MLAG), single path topology |
| III | Dual network paths, dual ISPs, ECMP or active/standby, concurrently maintainable |
| IV | Fully redundant and fault-tolerant: dual fabrics, dual ISPs, automated failover, no SPOF |

For Tier III and IV, the network design must support maintenance without downtime. This means:

- Removing any single spine switch must not cause traffic loss (ECMP across remaining spines)
- Removing any single leaf must not affect other leaves
- Dual-homed servers (MLAG or bonding) for Tier IV

---

## 11. Micro-Segmentation in the DC Fabric

### The Perimeter is Dead

Traditional DC security placed firewalls at the north-south border. East-west traffic between servers was implicitly trusted. This model fails catastrophically when an attacker gains a foothold on any server — they can move laterally without restriction.

### Micro-Segmentation Approaches

**Network-based (ACLs on leaf switches):**
- Applied per-port, per-VLAN, or per-VNI
- Hardware-accelerated in switch ASICs (line-rate enforcement)
- Limited flexibility: rules are based on IP/port, not application identity
- Scale limit: TCAM size limits ACL entries (typically 2,000-8,000 entries)

**Host-based (eBPF, iptables, nftables):**
- Applied at the hypervisor vSwitch or bare-metal host kernel
- Unlimited rule scale (software enforcement)
- Can enforce L7 policies (HTTP path, gRPC method)
- Higher CPU overhead than hardware ACLs

**Overlay-based (EVPN security groups):**
- Security group tags (SGT) carried in VXLAN Group Policy ID field
- Policy enforced at VTEP based on source SGT + destination SGT matrix
- Scales better than per-flow ACLs: $O(G^2)$ policies for $G$ groups vs $O(H^2)$ for $H$ hosts

**Service mesh (Istio, Linkerd, Cilium):**
- Mutual TLS (mTLS) between all services
- L7-aware policy (allow GET but deny DELETE)
- Identity-based, not IP-based: policies follow services across hosts
- Overhead: sidecar proxy adds 1-5ms latency per hop

### Zero Trust Principles Applied

1. **No implicit trust** based on network location
2. **Least privilege** access between every pair of services
3. **Continuous verification** (mTLS certificate rotation, token validation)
4. **Assume breach** — design as if the attacker is already inside

---

## 12. Operational Considerations

### Underlay Protocol Selection

| Protocol | Type | Convergence | Complexity | Recommended For |
|:---|:---|:---:|:---:|:---|
| eBGP | Path vector | 1-3s | Low | Most spine-leaf fabrics |
| OSPF | Link state | < 1s | Medium | Small/medium fabrics |
| IS-IS | Link state | < 1s | Medium | Large/multi-vendor fabrics |

**eBGP is the dominant choice** for spine-leaf because:
- Each device gets its own ASN (no area design needed)
- AS path naturally prevents loops
- Multipath works without OSPF equal-cost path limitations
- BGP peering = explicit neighbor relationships (no surprises from auto-discovery)

### Monitoring and Telemetry

- **Streaming telemetry** (gNMI, OpenConfig) preferred over SNMP polling for real-time fabric health
- **BFD** (Bidirectional Forwarding Detection) on all underlay links for sub-second failure detection (50-300ms)
- **Interface error counters** (CRC errors, input errors, drops) — even 1 error per hour indicates a bad cable or optic
- **ECMP balance monitoring** — compare per-spine byte counters; variance > 20% indicates polarization or elephant flows
- **BGP session state** — alert on any session flap; a flapping session causes route churn across the fabric

### Capacity Planning

Annual growth factor for DC traffic:

$$BW_{year+1} = BW_{year} \times (1 + g)$$

With 30% annual growth, bandwidth doubles in:

$$t_{double} = \frac{\ln 2}{\ln 1.3} \approx 2.64 \text{ years}$$

Design the fabric for 3 years of growth at initial deployment:

$$BW_{design} = BW_{initial} \times (1.3)^3 = BW_{initial} \times 2.197$$

This means provisioning roughly 2.2x current requirements at day-one deployment.

---

## Prerequisites

- networking fundamentals (L2/L3, VLANs, routing, switching), bgp, ospf, ecmp, vxlan, ethernet, subnetting

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| ECMP hash lookup | O(1) | O(S) per leaf |
| EVPN route lookup | O(log H) | O(H) per VTEP |
| Clos path selection | O(1) | O(S) per stage |
| ACL/TCAM lookup | O(1) | O(R) rules |

---

*Data center network design is applied mathematics: Clos theory determines the topology, oversubscription ratios set the economics, ECMP hashing distributes the load, and availability calculations justify the redundancy budget. The shift from 3-tier to spine-leaf was not a fashion trend — it was the inevitable consequence of east-west traffic dominance in a world of microservices, distributed storage, and GPU clusters.*
