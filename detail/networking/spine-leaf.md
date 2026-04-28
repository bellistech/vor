# Spine-Leaf (Clos) Topology — Deep Dive

> *Spine-leaf is Charles Clos's 1953 telephone-switching paper reincarnated as Ethernet. The math is exactly the math of a 3-stage Clos network — `m` ingress switches, `n` middle switches, `m` egress switches — with the property that any input port can reach any output port through any middle switch, and the non-blocking conditions reduce to two arithmetic inequalities. In the data center the "ingress" and "egress" stages collapse into a single tier of leaves and the middle stage becomes the spine; the math is unchanged. Every modern hyperscale fabric — Google Jupiter, Facebook F16, Microsoft 400G, Amazon Brick — is a Clos network with VXLAN as the encapsulation, eBGP as the underlay, and EVPN as the control plane for layer-2 reachability. The questions an operator must answer ("how many spines?", "what oversubscription?", "what MTU?", "how do I lose a spine without losing service?") all reduce to either a Clos inequality, an ECMP hash-distribution argument, a VXLAN overhead calculation, or a buffer/incast analysis. This deep dive walks through every one.*

---

## 0. Notation and Preliminaries

Throughout this deep dive we use the following notation:

- $L$ — the number of leaf (top-of-rack) switches in a single fabric (or pod).
- $S$ — the number of spine switches in the same fabric.
- $p_h$ — the number of host-facing (downlink) ports per leaf, with bandwidth $b_h$ each.
- $p_u$ — the number of uplink (spine-facing) ports per leaf, with bandwidth $b_u$ each.
- $p_s$ — the number of leaf-facing ports per spine, with bandwidth $b_u$ each.
- $H = L \cdot p_h$ — total host capacity of the fabric.
- $\text{OSR}$ — oversubscription ratio at a leaf, $\text{OSR} = (p_h \cdot b_h) / (p_u \cdot b_u)$.
- $\text{ECMP}_k$ — number of equal-cost next hops per destination prefix, normally $k = S$.
- $V$ — VXLAN encapsulation overhead (50 bytes for IPv4 outer, 70 bytes for IPv6 outer).
- $M_t$ — tenant MTU (default 1500 bytes); $M_u$ — required underlay MTU.
- $\text{ASN}_i$ — BGP ASN of leaf $i$; $\text{ASN}_S$ — BGP ASN of the spine tier (or per-spine ASNs).

Standards-track references that anchor the math:

- **RFC 7348** (VXLAN, August 2014) — the encapsulation format, UDP port 4789, 24-bit VNI.
- **RFC 7432** (BGP MPLS-Based Ethernet VPN, February 2015) — the EVPN control plane and route types 1–5.
- **RFC 8365** (NVO over EVPN, March 2018) — VXLAN/NVGRE/GENEVE applicability of RFC 7432.
- **RFC 7637** (NVGRE) and **RFC 8926** (GENEVE) — alternative encapsulations with different overhead.
- Charles Clos, *"A Study of Non-Blocking Switching Networks,"* Bell System Technical Journal 32:2 (March 1953), pp. 406–424.
- Al-Fares, Loukissas, Vahdat, *"A Scalable, Commodity Data Center Network Architecture,"* SIGCOMM 2008 (the fat-tree paper).
- Singh et al., *"Jupiter Rising: A Decade of Clos Topologies and Centralized Control in Google's Datacenter Network,"* SIGCOMM 2015.

---

## 1. Clos Network Theory — The 1953 Paper

### 1.1 The 3-stage Clos network

Charles Clos studied the problem of building large telephone switches without using a single fully crossbar `N × N` matrix (which scales as $O(N^2)$ crosspoints). His decomposition: build a 3-stage network where the input stage has `r` switches each of size `n × m`, the middle stage has `m` switches each of size `r × r`, and the output stage has `r` switches each of size `m × n`.

```
                  Stage 1            Stage 2 (middle)         Stage 3
              ┌───────────┐         ┌─────────────┐         ┌───────────┐
   n inputs ──┤  n × m    ├─── m ───┤   r × r     ├─── m ───┤  m × n    ├── n outputs
              │  switch 1 │   ╲    │   switch 1   │   ╱    │  switch 1 │
              └───────────┘    ╲   └─────────────┘   ╱     └───────────┘
              ┌───────────┐     ╲  ┌─────────────┐  ╱      ┌───────────┐
              │  switch 2 ├──────╳─┤   switch 2  ├─╳───────┤  switch 2 │
              └───────────┘     ╱  └─────────────┘  ╲      └───────────┘
                ⋮               ╱       ⋮           ╲          ⋮
              ┌───────────┐    ╱   ┌─────────────┐   ╲     ┌───────────┐
              │  switch r ├───     │   switch m  │    ╲────┤  switch r │
              └───────────┘        └─────────────┘         └───────────┘
```

Total crosspoints: $C(n,m,r) = 2 r n m + m r^2$. Clos showed that for a fixed input count $N = r n$, this is minimised by a particular choice of $n, m, r$ — but the more useful result is the *non-blocking condition*.

### 1.2 Rearrangeably non-blocking

A network is **rearrangeably non-blocking** if for any permutation of inputs to outputs, there exists an assignment of paths through the middle stage such that no two paths collide. Clos proved:

$$
m \;\geq\; n
$$

For the data-center translation: the spine count must be at least the per-leaf uplink count for rearrangeable non-blocking. Rearrangement may require breaking and re-establishing existing flows — acceptable in circuit-switched telephony at scale, problematic in packet networks, but in practice ECMP achieves the same outcome statistically without the rearrangement step.

### 1.3 Strictly non-blocking

A network is **strictly non-blocking** if any new connection request can be satisfied without disturbing existing connections. Clos's stricter inequality:

$$
m \;\geq\; 2n - 1
$$

Proof sketch (Clos 1953, §III): a new input on a stage-1 switch can be blocked from reaching its target by at most $(n-1)$ other connections from the same stage-1 switch (occupying $n-1$ middle switches) plus $(n-1)$ other connections from the target's stage-3 switch (occupying another $n-1$ middle switches). So we need at least $(n-1) + (n-1) + 1 = 2n - 1$ middle switches to guarantee a free path.

In the DC fabric: strictly non-blocking is rarely required because flow setup is on the order of microseconds (TCP handshake, RDMA verbs) and ECMP rehashes on flow start, not packet-by-packet rearrangement.

### 1.4 Crosspoint scaling

For $N = rn$ inputs/outputs with strictly non-blocking middle:

$$
C_{\text{Clos}} = 2rn(2n-1) + (2n-1)r^2 = (2n-1)(2rn + r^2)
$$

Compare with the monolithic crossbar: $C_{\text{xbar}} = N^2 = r^2 n^2$.

For $N = 1024$, choosing $n = 32, r = 32$ gives:

$$
C_{\text{Clos}} = 63 \times (2 \cdot 32 \cdot 32 + 32^2) = 63 \times (2048 + 1024) = 63 \times 3072 = 193{,}536
$$

versus $C_{\text{xbar}} = 1{,}048{,}576$. The Clos saves a factor of 5.4× in crosspoints; for $N = 65536$ the savings exceed two orders of magnitude.

This is the original economic argument for spine-leaf: **building a single `N × N` switch with millions of internal crosspoints is harder than building many small switches connected as a Clos.**

---

## 2. DC Translation — Spine-Leaf Math

### 2.1 The folded Clos

The data-center spine-leaf is a *folded* 3-stage Clos: stages 1 and 3 are identified (each leaf is both ingress and egress), and stage 2 is the spine.

```
        ┌─────┐  ┌─────┐  ┌─────┐  ┌─────┐
        │  S1 │  │  S2 │  │  S3 │  │  S4 │       ← spine tier (stage 2)
        └──┬──┘  └──┬──┘  └──┬──┘  └──┬──┘
           │        │        │        │
   ┌───────┼────────┼────────┼────────┼───────┐  full mesh: every leaf
   │       │        │        │        │       │  has one uplink to each
   └───┬───┴───┬────┴───┬────┴───┬────┴───┬───┘  spine
       │       │        │        │        │
     ┌─┴─┐   ┌─┴─┐    ┌─┴─┐    ┌─┴─┐    ┌─┴─┐
     │L1 │   │L2 │    │L3 │    │L4 │    │L5 │    ← leaf tier (stages 1 & 3 folded)
     └───┘   └───┘    └───┘    └───┘    └───┘
       │       │        │        │        │
     hosts   hosts    hosts    hosts    hosts
```

Each leaf has $p_u$ uplinks (one to each of $S$ spines, so $p_u = S$ in the canonical case) and $p_h$ host-facing downlinks. Each spine has $p_s = L$ leaf-facing ports.

### 2.2 Oversubscription

Oversubscription is the ratio of host-facing bandwidth to fabric-facing bandwidth at the leaf:

$$
\text{OSR} \;=\; \frac{p_h \cdot b_h}{p_u \cdot b_u}
$$

| Profile                | OSR     | Use case                                          |
|------------------------|---------|---------------------------------------------------|
| Line-rate (1:1)        | 1:1     | HFT, AI training, RDMA, anything intra-fabric     |
| Standard DC (3:1)      | 3:1     | General compute, mixed traffic                    |
| Storage / hot-tier     | 2:1     | Object stores, distributed databases              |
| Edge-aggregation (5:1) | 5:1     | Web tier with mostly N/S traffic                  |

**Worked example.** A 48-port 25G leaf with 6× 100G uplinks:
$$
\text{OSR} = \frac{48 \times 25}{6 \times 100} = \frac{1200}{600} = 2{:}1
$$

To reach line-rate from the same chassis, double the uplinks to 12× 100G — but most 1U leafs only have 6 or 8 uplink ports, so vendors instead provide 32-port 100G "upgrade" leaves.

### 2.3 Port-count math: $S \times L$ wiring count

Total fabric uplink ports (cables): $S \cdot L$ (one per spine-leaf pair). Each cable terminates two ports (one on a spine, one on a leaf) so total port consumption is $2SL$.

Per-spine port count: $p_s = L$. So a 32-port 100G spine supports up to $L = 32$ leaves. Bigger spines (Tomahawk-3 64×400G) support up to 64 leaves.

Host capacity:
$$
H = L \cdot p_h
$$

For a 32-leaf, 48-port-25G fabric: $H = 32 \cdot 48 = 1536$ host ports of 25G, or 38.4 Tbps of host capacity.

### 2.4 Cabling and optics cost

Total fibre runs $= S \cdot L$. For typical 5–20 m intra-DC distances this is multimode OM4 with 100G-SR4 (parallel ribbon) or singlemode OS2 with 100G-FR1 (single-pair). Cable management at $L = 64$ leaves and $S = 8$ spines means 512 fibre pairs converging on the spine row — non-trivial.

---

## 3. ECMP Load Balancing

### 3.1 The 5-tuple hash

ECMP picks one of $k$ equal-cost next hops per packet by hashing five header fields:

$$
h = H(\text{src\_ip}, \text{dst\_ip}, \text{src\_port}, \text{dst\_port}, \text{proto}) \mod k
$$

The hash is computed by the switch ASIC at line rate. Common hash families:

- **CRC32** — Broadcom Tomahawk default; cheap, but linear under XOR which leads to polarization.
- **xxHash / Murmur** — used in some merchant silicon for better distribution.
- **Toeplitz** — used in NIC RSS; rarely on switches.

Same flow → same hash → same path → in-order delivery (TCP-friendly).

### 3.2 Hash polarization

Polarization occurs when consecutive hops use the same hash function with the same seed. Suppose two leaves L1 and L2 both feed traffic through spines S1–S4. If the hash $H$ takes only `dst_ip` and `proto` as input (a pathological reduction), and L1 hashes flow $f$ to spine S1, L2 will *also* hash $f$ to S1 — but only if the input space at L1 and L2 is identical. In practice the 5-tuple varies, so pure polarization is rare with full 5-tuple hashing, but degenerates with non-IP traffic, MPLS, GRE.

**Remediation:**

1. **Per-switch hash seeds.** Each switch's hash uses a configured seed. Cisco NX-OS: `hardware ecmp hash-offset <0–32>`. Arista EOS: `load-balance ecmp seed <integer>`. Juniper: `forwarding-options enhanced-hash-key`.
2. **Tuple expansion.** Include inner-packet headers when ECMP is used over an encapsulation (VXLAN: hash on inner 5-tuple).
3. **Resilient hashing.** Modern ASICs (Tomahawk-3+, Trident-4) use a flow-table to keep flows pinned across next-hop set changes. NX-OS: `ip load-sharing address source-destination universal-id <n>`.

### 3.3 Birthday paradox and hash collisions

With $k$ buckets and $f$ flows, the expected number of buckets with $\geq 2$ flows (collisions) approaches $f^2 / (2k)$ for $f \ll k$.

Worked: 1000 flows over 4 spines: $E[\text{coll}] \approx 1000^2 / 8 = 125{,}000$. That's a lot of collisions — but in this regime each spine carries 250 flows and the distribution is nearly uniform; the issue is *load* on each spine, not whether two flows ever share a path. The interesting regime is **elephant flow collision**: two 100Gbps flows hashed to the same spine can saturate that link while other spines are 50% idle.

The probability that all $f$ flows hash to distinct spines (no collisions at all):
$$
P(\text{no collision}) = \prod_{i=0}^{f-1} \frac{k - i}{k}
$$

With $f = 4, k = 4$: $P = (4 \cdot 3 \cdot 2 \cdot 1)/256 = 0.094$. Even for just 4 flows over 4 spines, probability of perfect spread is under 10%.

### 3.4 Elephant flows and DLB / CONGA / Ramcloud

Static hashing performs badly when flow size variance is high (a few "elephants" carrying most bytes). Mitigations:

- **DLB (Dynamic Load Balancing)** — per-flow ASIC tracks egress queue depth and reshashes elephants when a less-loaded path exists. Broadcom feature.
- **CONGA** — Cisco research paper (SIGCOMM 2014). Splits flows into "flowlets" (gaps > 100 µs) and re-routes per flowlet using leaf-to-leaf congestion telemetry.
- **HPCC, Swift** — RDMA-era congestion algorithms that share state via INT (in-band network telemetry).
- **Packet-spraying** — used in Arista 7800-class with 7280R-PSE; tolerates reordering under careful TCP pacing or RDMA.

---

## 4. Path Diversity and ECMP Width

With $S$ spines, every leaf-to-leaf path has exactly $S$ equal-cost ASN-disjoint options (one through each spine). This gives:

$$
\text{ECMP fan-out} = S \quad \text{(per leaf-leaf prefix)}
$$

Configured ECMP in BGP underlay:

```bash
# Cisco NX-OS — accept up to 64 paths
router bgp 65001
 maximum-paths 64
 maximum-paths ibgp 64
```

```bash
# Arista EOS
router bgp 65001
 maximum-paths 64 ecmp 64
```

```bash
# Juniper Junos
set policy-options policy-statement ECMP-LOAD-BALANCE then load-balance per-packet
set routing-options forwarding-table export ECMP-LOAD-BALANCE
```

Default `maximum-paths` is usually 1–4 (vendor-dependent). Always increase to at least $S$, typically a power of 2 for hash efficiency.

### 4.1 Path-set deltas under failure

When spine $S_i$ fails, every leaf removes the route via $S_i$ from the FIB and ECMP rebalances over the remaining $S - 1$ spines. The delta is uniform: each remaining spine takes on $1/(S-1)$ extra share, and total fabric capacity drops by $1/S$.

For $S = 4$, single spine fail $\to$ 75% capacity, per-remaining-spine load goes from 25% to 33%.

---

## 5. BGP-EVPN Underlay+Overlay

### 5.1 Underlay: eBGP between spine and leaf

Each leaf is its own private ASN (RFC 6996 reserves 64512–65534 for 16-bit private; RFC 6996bis / RFC 4893 for 4-byte ASNs from 4200000000–4294967294). Each spine is its own private ASN, or all spines share one ASN (less common).

```
       ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
       │ ASN 65500│    │ ASN 65501│    │ ASN 65502│    │ ASN 65503│
       │  Spine-1 │    │  Spine-2 │    │  Spine-3 │    │  Spine-4 │
       └──────────┘    └──────────┘    └──────────┘    └──────────┘
              \              \              /              /
               \   eBGP       \  eBGP      / eBGP         / eBGP
                \              \          /              /
              ┌──┴──────────────┴────────┴──────────────┴──┐
              │   Leaf-1 (ASN 65001)                       │
              └────────────────────────────────────────────┘
              ┌────────────────────────────────────────────┐
              │   Leaf-2 (ASN 65002)                       │
              └────────────────────────────────────────────┘
                            ⋮
              ┌────────────────────────────────────────────┐
              │   Leaf-N (ASN 65000+N)                     │
              └────────────────────────────────────────────┘
```

Loop avoidance is automatic via AS_PATH: leaves never see their own ASN echoed back from another spine.

**Sample BGP config (Arista EOS, leaf 65001):**

```bash
router bgp 65001
   router-id 10.0.0.1
   neighbor SPINES peer group
   neighbor SPINES remote-as external
   neighbor SPINES allowas-in 1
   neighbor SPINES update-source Loopback0
   neighbor SPINES bfd
   neighbor 10.255.0.1 peer group SPINES
   neighbor 10.255.0.2 peer group SPINES
   neighbor 10.255.0.3 peer group SPINES
   neighbor 10.255.0.4 peer group SPINES
   address-family ipv4
      neighbor SPINES activate
      maximum-paths 4 ecmp 4
   address-family evpn
      neighbor SPINES activate
```

`allowas-in 1` is needed because spines re-advertise and the leaf's own ASN may appear once in AS_PATH for ESI multi-homing scenarios.

### 5.2 Overlay: VXLAN + EVPN

Each leaf is a VTEP (VXLAN Tunnel Endpoint). EVPN routes carry MAC/IP reachability between VTEPs. Tenant traffic is encapsulated in VXLAN with a 24-bit VNI selecting the bridge domain (or VRF for L3VNI).

### 5.3 EVPN Route Types (RFC 7432)

| Type | Name                              | Purpose                                              |
|------|-----------------------------------|------------------------------------------------------|
| 1    | Ethernet Auto-Discovery (A-D)     | Per-EVI and per-ESI; MAC mass-withdraw on link fail  |
| 2    | MAC/IP Advertisement              | The bread-and-butter: "MAC X is behind VTEP Y"       |
| 3    | Inclusive Multicast Ethernet Tag  | BUM tree — "I am a receiver for VNI 10010 BUM"       |
| 4    | Ethernet Segment                  | DF election among VTEPs sharing an Ethernet Segment  |
| 5    | IP Prefix                         | Inter-subnet routing — "subnet 10.20.0.0/16 via VTEP"|

Route Type 2 NLRI (RFC 7432 §7.2):

```
 +----------------------+
 | RD (8 bytes)         |
 +----------------------+
 | ESI (10 bytes)       |
 +----------------------+
 | Ethernet Tag (4 B)   |
 +----------------------+
 | MAC Length (1 byte)  |
 +----------------------+
 | MAC (6 bytes)        |
 +----------------------+
 | IP Length (1 byte)   |
 +----------------------+
 | IP (0/4/16 bytes)    |
 +----------------------+
 | MPLS Label1 (3 B)    |  ← VNI in low-order bits for VXLAN
 +----------------------+
 | MPLS Label2 (3 B)    |  ← L3VNI for symmetric IRB
 +----------------------+
```

Route Targets (RTs): import RT and export RT define the VRF/EVI membership. Asymmetric IRB uses one VNI per BD; symmetric IRB uses both an L2VNI (per-BD) and an L3VNI (per-tenant-VRF).

### 5.4 Symmetric vs asymmetric IRB

- **Asymmetric IRB:** the source VTEP routes into the destination subnet's VNI before encap. Fast path, but every VTEP must have every VNI (and every SVI) — scaling problem.
- **Symmetric IRB:** the source VTEP routes into the L3VNI, encaps, and the destination VTEP routes out of the L3VNI into the destination subnet's L2VNI. Each VTEP only needs the L2VNIs of subnets it actually serves. Default for modern fabrics.

---

## 6. VXLAN Math

### 6.1 The 50-byte tax

Per RFC 7348 §5, the VXLAN encap on top of an inner Ethernet frame is:

| Layer            | Bytes | Field                                      |
|------------------|-------|--------------------------------------------|
| Outer Ethernet   | 14    | Outer DMAC (6) + SMAC (6) + EType (2)      |
| Outer IPv4       | 20    | Standard IPv4 header                       |
| Outer UDP        | 8     | Src/dst port + length + checksum           |
| VXLAN Header     | 8     | Flags (1) + Reserved (3) + VNI (3) + Reserved (1) |
| **Total IPv4**   | **50**| **Header bytes added per packet**          |
| Outer IPv6       | 40    | (replaces IPv4) + same UDP + VXLAN         |
| **Total IPv6**   | **70**| **Header bytes added per packet**          |

Double-tagged outer (802.1Q dot1q on the underlay)? Add 4 more bytes per tag.

VXLAN flags byte: bit `I = 1` indicates VNI is valid. The 24-bit VNI gives $2^{24} = 16{,}777{,}216$ tenants — vs the 4096-tenant ceiling of 802.1Q VLANs (12-bit VID).

### 6.2 MTU planning

Tenant MTU $M_t$ requires underlay MTU:

$$
M_u \;\geq\; M_t + V
$$

For default Ethernet ($M_t = 1500$):
$$
M_u \geq 1550 \;\text{(IPv4)} \quad \text{or} \quad M_u \geq 1570 \;\text{(IPv6)}
$$

Practically, fabrics run jumbo: $M_u = 9216$ on the underlay so tenants can also use jumbo (9000 inside, 9050 outside). Some operators set $M_u = 9050$ to keep the inner MTU at exactly 9000 and avoid edge negotiation surprises.

**Path MTU discovery is broken** when a VXLAN endpoint (VTEP) sets DF on the outer header and an underlay link has lower MTU. The ICMP "frag needed" returns to the *VTEP*, not the original sender — the VTEP must translate to inner-PMTU updates, which most ASICs do not do. Operational rule: **set underlay MTU at least 50 bytes above any tenant MTU and forget about PMTUD**.

### 6.3 BUM traffic — multicast vs ingress replication

BUM = Broadcast, Unknown unicast, Multicast. On a VXLAN fabric a tenant frame addressed to the broadcast MAC must be flooded to every VTEP that has the source VNI configured.

**Multicast underlay:** each VNI maps to a multicast group (typically in the 239.x range). VTEPs join the group per-VNI; a single encapsulated copy traverses the spine and is replicated by spine PIM-SM. Pro: efficient bandwidth. Con: requires PIM in the underlay.

**Ingress replication (head-end):** the source VTEP sends one VXLAN-encapped copy per remote VTEP. EVPN Route Type 3 (Inclusive Multicast) populates the head-end's replication list. Pro: no PIM. Con: linear bandwidth blowup with VTEP count.

For a tenant generating $r$ Mbps of BUM with $V$ remote VTEPs:

$$
B_{\text{IR}} = r \cdot V, \qquad B_{\text{Mcast}} = r
$$

For $r = 10$ Mbps, $V = 64$: ingress replication consumes 640 Mbps; multicast consumes 10 Mbps. In low-BUM fabrics (modern, ARP-suppressed) this is academic — typical BUM rate is < 1 Mbps.

ARP suppression: when a VTEP receives an ARP request for a known MAC, it answers locally instead of flooding. EVPN Route Type 2 carries the IP→MAC binding so every VTEP has a complete ARP table without having to flood.

---

## 7. Anycast Gateway

### 7.1 Distributed gateway IP and MAC

In a traditional L2 network, the default gateway is a single SVI on a router (or pair via HSRP/VRRP). All inter-subnet traffic must hairpin through that router.

In an anycast-gateway fabric, **every leaf with the relevant subnet configures the same SVI IP and the same SVI MAC**:

```bash
# NX-OS — anycast gateway MAC (fabric-wide)
fabric forwarding anycast-gateway-mac 0000.0a0a.0a0a

interface Vlan10
  no shutdown
  vrf member tenant-A
  ip address 10.10.0.1/24
  fabric forwarding mode anycast-gateway
```

A host's ARP for 10.10.0.1 is answered by the *local* leaf, regardless of which leaf it is connected to. Inter-subnet traffic is routed locally and encapsulated into the destination's L3VNI — never tromboned to a central router.

### 7.2 Failure modes

- **Leaf dies:** hosts on that leaf go down; the rest of the fabric is unaffected. Traffic from elsewhere destined to those hosts withdraws via Type 2 (MAC withdraw) and Type 1 (A-D per-EVI mass-withdraw).
- **VLAN misconfiguration:** if leaf A has VLAN 10 mapped to VNI 10010 and leaf B has VLAN 10 mapped to VNI 10020, the anycast gateway ARP works on each leaf locally but inter-leaf bridging is silently broken. Standard EVPN diagnostic: `show bgp l2vpn evpn` and look for inconsistent VNI on Type 2 routes for the same subnet.

### 7.3 Comparison to FHRP/HSRP

| Aspect           | HSRP/VRRP                          | Anycast gateway                       |
|------------------|------------------------------------|---------------------------------------|
| Active gateways  | 1 (active), 1 (standby)            | All leaves with the SVI               |
| Failover         | 3–5s (default) / sub-second tuned  | Zero — every leaf is active           |
| North-south path | Hairpins to active                 | Local                                 |
| Tromboning       | Yes                                | No                                    |
| Scale            | Pair per VLAN                      | Per leaf, fabric-wide                 |

---

## 8. Failure Scenarios and Convergence

### 8.1 Single spine failure

**Detection.** BFD runs on each leaf-spine session. With BFD `interval 50ms multiplier 3`, detection latency is $50 \times 3 = 150$ ms.

**Convergence.** On detection:

1. Leaf removes routes via the failed spine from BGP RIB.
2. RIB→FIB programming: typically 50–200 ms on modern silicon.
3. ECMP rebalances over remaining $S-1$ spines.

**Capacity loss.** Exactly $1/S$ of fabric capacity. For $S = 4$: 25% drop. No flows are lost (TCP retransmits any in-flight at the moment of failure). RDMA-NIC failover times depend on NIC firmware; NVIDIA ConnectX-6 default: under 200 ms.

### 8.2 Multi-spine failure

For two simultaneous spine failures: $2/S$ capacity loss. ECMP recomputes on second BFD timeout.

For $k$ simultaneous failures, capacity drops to $(S-k)/S$.

The *probability* of $k$ simultaneous spine failures, given each spine has annual fail rate $\lambda$ and MTTR $\tau$:

$$
P(k \text{ down}) = \binom{S}{k} (\lambda \tau)^k (1 - \lambda \tau)^{S-k}
$$

For $\lambda = 0.5$/year (one fail every 2 years per spine), $\tau = 4$ hours = $4.6 \times 10^{-4}$ years:

$$
P(2 \text{ down}) = \binom{4}{2} (2.3 \times 10^{-4})^2 (1 - 2.3 \times 10^{-4})^2 \approx 6 \times 5.3 \times 10^{-8} \approx 3.2 \times 10^{-7}
$$

Less than once in a million hours. The dominant failure mode is *correlated* spine failure (shared power, shared software bug), which the math above ignores.

### 8.3 Leaf failure

A leaf failure isolates only the hosts attached to that leaf. From the rest of the fabric:

- BGP sessions to spines time out (BFD: 150 ms).
- Spines withdraw the leaf's loopback prefix via BGP UPDATE.
- EVPN Route Type 2 (MAC/IP) and Type 1 (A-D) are withdrawn for all hosts on that leaf.
- Remote leaves remove the failed VTEP from their NVE peer list.

Typical convergence: BFD detection + propagation + local FIB reprogram = 250–400 ms.

For dual-attached hosts (LAG to two leaves with MLAG or ESI multi-homing), the failure is invisible to the host: the LAG drops one member, the remaining leaf advertises the host's MAC.

### 8.4 Link failure (leaf-spine cable cut)

The leaf has lost one of $S$ uplinks. ECMP rebalances over remaining $S-1$. Convergence: BFD or LoS (loss-of-signal, microseconds).

If the leaf has only 2 uplinks and one fails, OSR doubles for that leaf — half the host capacity is now starved at peak. Always have $S \geq 3$ uplinks per leaf.

### 8.5 Convergence-time table (typical, well-tuned fabric)

| Event                          | Detection  | RIB→FIB    | Total visible to TCP |
|--------------------------------|-----------:|-----------:|---------------------:|
| Spine link LoS                 |  < 1 ms    |  10–50 ms  |       10–50 ms       |
| Spine BFD timeout              |   150 ms   |  50–200 ms |       200–350 ms     |
| Spine soft fail (BGP HOLD)     |    9–180 s | 50–200 ms  |       9 s – 3 min    |
| Leaf reload                    |   1 sec    | 200–500 ms |       1.2–1.5 s      |
| Leaf cable cut                 |  < 1 ms    | 50–200 ms  |       50–200 ms      |

The "soft fail" row is the danger zone: a spine that is alive enough to keep BGP sessions but blackholing data plane. BFD is the cure; without BFD, BGP HOLD timer (default 180 s) is the only signal.

---

## 9. MLAG / vPC vs Pure L3

### 9.1 MLAG/vPC mechanics

MLAG (Arista), vPC (Cisco), MC-LAG (Juniper) are vendor names for the same thing: a pair of switches present a single logical L2 device to a downstream LAG-capable host. Implementation:

- A peer link between the two switches carries CFM/LACP state.
- A keepalive (out-of-band or routed loopback) detects peer death.
- MAC tables are synchronised via a peer protocol (Cisco CFS, Arista MLAG sync).
- LACP system-id is the same on both peers.

### 9.2 Pure L3 with anycast gateway

In a pure-L3 fabric, every link is routed (no L2 between leaves). Hosts that need redundancy use:

- **Active/standby NIC bonding** with a virtual MAC (host responsibility, not network).
- **EVPN ESI multi-homing (RFC 7432 §8)** — two leaves share an Ethernet Segment Identifier; DF election; per-flow ECMP from the remote VTEPs.

ESI multi-homing replaces MLAG without the peer-link, peer-keepalive, or CFM machinery.

### 9.3 Tradeoff matrix

| Aspect              | MLAG/vPC                          | Pure L3 + ESI                       |
|---------------------|-----------------------------------|-------------------------------------|
| Peer link           | Required (40–100G typically)      | None                                |
| Peer keepalive      | Required (out-of-band best)       | None                                |
| MAC sync protocol   | Vendor proprietary                | EVPN Route Type 2 (standard)        |
| Failover            | 100–500 ms (LACP-driven)          | Sub-200 ms via ESI Type 1 A-D       |
| L3 features on link | Limited (vPC peer-gateway hacks)  | Full (it's just an L3 link)         |
| Vendor lock         | High                              | Low (RFC-based)                     |
| Operational mind    | "Two switches as one"             | "Just routers all the way down"     |

Modern best practice: pure L3 with anycast gateway and ESI multi-homing for any host that requires LAG-style redundancy. Use MLAG only when forced by legacy hosts (SAN heads, hypervisor clusters with strict HA assumptions).

---

## 10. Border Leaf / Service Leaf

### 10.1 North-south gateway

A **border leaf** terminates connectivity to the outside world: WAN, internet, MPLS PEs, customer carrier handoffs. It is a normal leaf with one extra role: importing external prefixes into the fabric and exporting fabric prefixes (or a default route) outward.

```
                    ┌───── Internet / WAN ──────┐
                    │                            │
               ┌────┴────┐                  ┌────┴────┐
               │ Border  │                  │ Border  │
               │  Leaf 1 │                  │  Leaf 2 │     ← border tier
               └────┬────┘                  └────┬────┘
                    │                            │
                    └─────┬──────────────┬───────┘
                          │              │
                       ┌──┴──┐        ┌──┴──┐
                       │ Sp1 │  ...   │ Sp4 │             ← spine tier
                       └──┬──┘        └──┬──┘
                          │              │
                  ┌───────┴───┬───┬──────┴────┐
                  │           │   │           │
                ┌─┴─┐       ┌─┴─┐ │ ┌─┴─┐
                │L1 │       │L2 │ │ │L3 │                  ← server leaf tier
                └───┘       └───┘   └───┘
```

Border leaves typically run BGP to upstream PEs in addition to the fabric eBGP-EVPN session.

### 10.2 Service leaf — firewall, load balancer, DPI

A **service leaf** integrates appliances (firewalls, load balancers, IPS, DDoS) into the fabric. Approaches:

- **Routed appliance + EBGP:** appliance peers BGP into the fabric, advertises VIPs. Most flexible.
- **Transparent (L2) appliance:** inserted via VRF stitching or Service Chain; complex.
- **One-arm load balancer:** VIP behind a leaf SVI; SNAT to ensure return traffic.

EVPN supports service insertion via Route Type 5 (IP Prefix) and per-VRF leaking.

### 10.3 Multi-tenant border

Each tenant VRF needs its own external connectivity. Patterns:

- **Per-VRF VPNv4 to PE:** scalable, but burns ASN slots.
- **VRF-lite with sub-interfaces:** simple, but per-tenant config sprawl.
- **EVPN Type 5 to internet PE:** clean, requires PE support for EVPN.

---

## 11. Multi-Pod / Super-Spine — 5-stage Clos

### 11.1 When 3-stage runs out

A single 3-stage fabric scales until either (a) the spine port count is exhausted ($L > p_s$) or (b) physical-layer constraints (cable distance, power per row, fault domain size) push you to multiple pods.

A 32-port 100G spine: $L = 32$ leaves. A 64-port 400G spine: $L = 64$ leaves. Beyond that, we go to 5-stage.

### 11.2 The 5-stage Clos

```
                        ┌──────────────────────────┐
                        │      Super-Spine tier    │  ← stage 3 (middle)
                        │  ┌────┐  ┌────┐  ┌────┐  │
                        │  │ SS1│  │ SS2│  │ SS3│  │
                        │  └─┬──┘  └─┬──┘  └─┬──┘  │
                        └────┼───────┼───────┼─────┘
                             │       │       │
            ┌────────────────┼───────┼───────┼─────────────────┐
            │                │       │       │                 │
       ┌────┴───┐       ┌────┴───┐  ...  ┌────┴───┐       ┌────┴───┐
       │ Pod A  │       │ Pod B  │       │ Pod C  │       │ Pod D  │   ← pods
       │ Spines │       │ Spines │       │ Spines │       │ Spines │
       └────┬───┘       └────┬───┘       └────┬───┘       └────┬───┘
            │                │                │                │
        Leaves           Leaves           Leaves           Leaves
            │                │                │                │
         hosts            hosts            hosts            hosts
```

Each pod is itself a 3-stage Clos. Pod spines connect upward to super-spines instead of (only) to pod leaves.

Stage count: leaf → pod-spine → super-spine → pod-spine → leaf = 5 stages.

### 11.3 Inter-pod math

Per-pod uplink bandwidth to super-spine:
$$
B_{\text{pod-up}} = (p_u^{\text{spine}} \cdot b_u) \cdot S_{\text{pod}}
$$

where $S_{\text{pod}}$ is the number of pod-spine switches.

Cross-pod oversubscription is set independently from intra-pod OSR. A common pattern: 1:1 within a pod, 3:1 inter-pod, on the assumption that most traffic is intra-pod (locality).

### 11.4 Cisco ACI multi-pod, Arista UCN, Google Jupiter

- **Cisco ACI Multi-Pod:** the IPN (Inter-Pod Network) is a separate L3 fabric carrying VXLAN between pods; APIC is fabric-wide; OSPF/BGP runs in IPN.
- **Arista Universal Cloud Network:** EOS uses CVP for fabric orchestration; VXLAN-EVPN reaches across pods via the super-spine.
- **Google Jupiter (SIGCOMM 2015):** custom merchant silicon, 5-stage Clos at scale, centralised SDN control plane, per-DC bisection bandwidth in the petabit/s range.

### 11.5 Multi-site EVPN

Beyond a single DC, multi-site EVPN connects pods across WANs:

- **EVPN VXLAN over DCI** with route-target imports between sites.
- **EVPN over MPLS** for carrier-grade interconnect (Type 5 IP-VPN compatibility).
- **Cisco Multi-Site:** introduces a Border Gateway (BGW) per site that translates VTEP IPs (anycast on outside, real on inside).

---

## 12. Buffer Math — Incast Collapse

### 12.1 The incast pathology

**TCP incast:** $N$ servers respond simultaneously to a single client (e.g., a Hadoop reduce, a memcached scatter-gather). All $N$ flows converge at the *last* leaf-host link with synchronised burst arrival.

If each server sends $b$ bytes at 25 Gbps, and the destination link is 25 Gbps, then $N$ flows demand $N \cdot 25$ Gbps for $b/(25\text{Gbps})$ seconds — but only 25 Gbps is available. Excess is buffered or dropped.

### 12.2 Buffer per port

A leaf with shared buffer $B$ and $P$ ports nominally allocates $B/P$ per port — but commercial silicon allows dynamic threshold:

$$
\text{buf}_{\text{per-port}} = B \cdot \alpha \cdot \frac{1}{1 + \alpha \cdot N_{\text{congested}}}
$$

This is Choudhury & Hahne's Dynamic Threshold algorithm. With $\alpha = 1/8$ (typical), one congested port can use up to $B/8$. As more ports congest, each gets less.

For Tomahawk-3: 64 MB shared buffer, 32× 400G ports → 2 MB nominal per port. Per-port at full burst: ~8 MB (DT $\alpha = 1/8 \times 64 = 8$ MB cap).

### 12.3 Incast burst formula

If $N$ flows burst $b$ bytes each into a 25 Gbps port, the buffer occupancy in steady state is:

$$
Q \;=\; \max(0, \; N \cdot b \;-\; (\text{linkrate}) \cdot t_{\text{burst}})
$$

For $N = 100$ servers, $b = 64$ KB each, port = 25 Gbps, burst time $t_{\text{burst}} = 100 \mu s$:

$$
\text{arrived} = 100 \cdot 64 \text{KB} = 6.4 \text{ MB}
$$
$$
\text{drained} = 25 \text{ Gbps} \cdot 100 \mu s = 25 \cdot 10^9 / 8 \cdot 10^{-4} = 312{,}500 \text{ B} \approx 312 \text{ KB}
$$
$$
Q = 6.4 \text{ MB} - 312 \text{ KB} \approx 6.1 \text{ MB}
$$

If $Q$ exceeds per-port buffer, packets drop. With Tomahawk-3 8 MB cap, fine. With older silicon (12 MB total, $\alpha$ stricter): exceeded → drops → TCP timeouts → retransmit storm → throughput collapses to a fraction of link rate. This is incast collapse.

### 12.4 Mitigations — DCQCN, CONGA, ECN

- **ECN (RFC 3168):** mark instead of drop. Sender's TCP/DCTCP reduces window.
- **DCTCP (Alizadeh et al., SIGCOMM 2010):** ECN-aware TCP that reduces window proportional to fraction of marked packets — instead of halving on a single mark.
- **DCQCN (RoCEv2, RFC 9028):** Data Center Quantized Congestion Notification — RDMA's incarnation of DCTCP. PFC + ECN + rate-based reaction.
- **CONGA:** flowlet-level rerouting through under-loaded paths.
- **Per-port shared buffer carving:** reserve guaranteed minimums per port to prevent buffer hogging.
- **PFC (802.1Qbb):** lossless priority. Pause-frames between switch and NIC. Required for RDMA. Risk: head-of-line blocking, deadlock if loops.

ECN config (Arista EOS):

```bash
!
qos profile DEFAULT
  qos trust dscp
  tx-queue 3 random-detect ecn min 16384 max 49152 max-probability 100
!
interface Ethernet1
  qos service-policy DEFAULT
```

That marks ECN when queue 3 occupancy exceeds 16 KB ramping to certain at 48 KB.

### 12.5 Lossless RDMA (RoCEv2) requirements

For RDMA over Converged Ethernet:

- **PFC enabled** on the priority-queue carrying RoCE.
- **ECN enabled** on the same queue.
- **MTU 9216** to amortize per-packet overhead.
- **No loops** anywhere in the path (PFC deadlock).
- **DCQCN tuning** on the NIC (NVIDIA ConnectX, default Pmin/Pmax).

A RoCE-broken fabric is invisible until your AI training job reports a latency tail at p99.99 — then the network team is pulling buffer telemetry for a week.

---

## 13. Worked Examples

### 13.1 Sizing a 2-tier fabric for 200 servers, 25G, with 100G uplinks

- Hosts: 200 × 25G = 5 Tbps.
- Choose 48-port 25G leaves with 6× 100G uplinks. Per-leaf host capacity: 48 × 25 = 1.2 Tbps. Per-leaf uplink: 6 × 100 = 600 Gbps. Leaf OSR: 2:1.
- Leaves needed: $\lceil 200 / 48 \rceil = 5$ leaves (200 / 5 = 40 hosts/leaf, 8 ports spare).
- Spines needed: each leaf has 6 uplinks → minimum 6 spines (1 uplink per spine). Cheaper: 2 spines, 3 uplinks each, but loses path diversity. 4 spines is the sweet spot — per-leaf 1.5× redundancy.
- Spine port count: 5 leaves × (6/4) = 7.5 uplinks/spine → round to 8 ports/spine. 32-port 100G spine has 24 ports unused (room for growth to ~21 leaves).

Final design: 5 leaves × 4 spines, OSR 2:1, fabric capacity 5 × 600 Gbps = 3 Tbps.

### 13.2 3-tier vs 2-tier breakeven

A 2-tier fabric is bounded by spine port count. With 32-port 100G spines: max 32 leaves, max $32 \times 48 = 1536$ host ports (at 25G each).

A 3-tier (5-stage) fabric scales by pod count. Each pod = 32 leaves; super-spine connects $P$ pods. With 32-port super-spines: max 32 pods × 1536 hosts/pod = 49,152 hosts.

Breakeven: when $L > 32$ in a single pod, you must go 3-tier. For 64-port 400G spines: breakeven shifts to $L > 64$.

Cost rule of thumb (2024):

| Tier | Ports/host | Rough $/host | Notes                         |
|------|-----------:|-------------:|-------------------------------|
| 1-tier (TOR only) | 1 | $200 | No redundancy                |
| 2-tier (spine-leaf) | 1 | $400–600 | Standard               |
| 3-tier (with super-spine) | 1 | $800–1200 | Hyperscale         |

### 13.3 VXLAN encap byte-level walkthrough

Inner Ethernet frame: 1500-byte Ethernet payload + 14 byte L2 header + 4 byte FCS = 1518 bytes.

After VXLAN encap (IPv4 outer, no VLAN tag on outer):

```
+---------------------+----------------------+----------------+----------------+----------------------+
|  Outer Eth (14B)    |    Outer IPv4 (20B)  |  Outer UDP(8B) | VXLAN Hdr(8B)  |  Inner Frame (1518B) |
+---------------------+----------------------+----------------+----------------+----------------------+
| dst MAC | src MAC|  | ver|hl|tos|len|...   | sport|dport|.. | flags|VNI|...  | dst|src|EType|payload|
| 6   B   | 6  B   |2 |   ...                |   ...          |   ...          |  ...                 |
+---------------------+----------------------+----------------+----------------+----------------------+
```

Total wire: 14 + 20 + 8 + 8 + 1518 = 1568 bytes (plus 4 byte outer FCS = 1572).

For the underlay link with MTU 9216, this is well under. For the underlay link with MTU 1500, 1568 > 1500 and the outer would need fragmentation — which fails because VTEPs typically set DF.

Outer UDP source port: VTEP hashes the **inner** 5-tuple to populate the outer source port. This randomises the outer 5-tuple so the underlay's ECMP hashing distributes encapsulated flows across spines.

Outer UDP destination port: 4789 (IANA-assigned, RFC 7348). Older Linux kernels used 8472 (the original Cisco-proposed port).

VXLAN header bit-level (8 bytes):

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|R|R|R|R|I|R|R|R|            Reserved                           |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                VNI (24 bits)                  |   Reserved    |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

`I` flag must be 1 (VNI valid). Other reserved bits MUST be 0.

### 13.4 BGP-EVPN Type 2 (MAC/IP) decode

A Type 2 update for MAC `0050.5683.0a01` and IP `10.10.0.42`, behind VTEP at loopback `192.0.2.11`, in VNI `10010`:

```
BGP UPDATE:
  AFI/SAFI: 25/70 (L2VPN/EVPN)
  Path attributes:
    NEXT_HOP: 192.0.2.11
    AS_PATH: 65001
    LOCAL_PREF: 100
    Extended Communities:
      RT: 65500:10010   (route target)
      MAC Mobility: seq=0, sticky=0
      Encap: VXLAN
  NLRI:
    Route Type: 2 (MAC/IP Advertisement)
    RD: 192.0.2.11:1
    ESI: 0 (single-homed)
    Ethernet Tag: 0
    MAC Length: 48
    MAC: 00:50:56:83:0a:01
    IP Length: 32
    IP: 10.10.0.42
    MPLS Label1: 10010 (L2VNI, 24-bit value left-shifted)
    MPLS Label2: 50000 (L3VNI for symmetric IRB)
```

Decoding the MPLS-Label-in-EVPN trick: RFC 7432 §5.1 says the EVPN NLRI carries a 3-byte "MPLS Label" field but for VXLAN it carries the VNI in the high-order 24 bits (the protocol predates a VXLAN-specific NLRI). VNI 10010 = 0x271A in 24 bits = `00 27 1A`.

### 13.5 Failure-recovery time calc for spine fail

Setup: $S = 4$ spines, BFD `interval 100ms multiplier 3`, $L = 32$ leaves, 1000 routes per leaf.

- BFD detection: $100 \times 3 = 300$ ms.
- BGP route invalidation per leaf: ~10 ms for 1000 routes (NSF in many platforms).
- FIB programming: ~50 ms (1000 routes × 50 µs/route).
- Total visible: 360 ms.

For the same fabric with BFD `50ms × 3` and faster ASIC:

- BFD: 150 ms.
- FIB: 30 ms.
- Total: 180 ms.

Without BFD (pure BGP HOLD):

- HOLD timer (default 180 s) until session times out.
- Total: 180 s. Unacceptable.

This is why **BFD is non-negotiable** in any modern spine-leaf fabric.

---

## 14. CLOS Scaling Limits — Fat-Tree, Clos, Torus

### 14.1 Fat-tree (Al-Fares et al. 2008)

A *k-ary fat-tree* is a special Clos with constraints:

- $k$ pods, each with $k/2$ aggregation switches and $k/2$ edge switches.
- Each edge switch has $k/2$ ports facing servers.
- Each pod connects upward to $(k/2)^2$ core switches.
- Total servers: $k^3 / 4$.

For $k = 48$: $48^3 / 4 = 27{,}648$ servers using only 48-port switches. The fat-tree is *exactly* a 3-tier Clos with $n = m = k/2$.

The fat-tree paper's contribution was showing that commodity switches in this topology give the bisection bandwidth of much more expensive monolithic boxes — at a fraction of the cost.

### 14.2 Pure Clos vs hypercube vs torus

| Topology    | Diameter             | Bisection BW            | Wiring                       | Used by           |
|-------------|---------------------|------------------------|------------------------------|-------------------|
| Clos (3-stage)| 2 (any two leaves) | Full ($N \cdot b/2$)   | Full mesh leaf-spine         | Most DCs          |
| Fat-tree    | 4 (any two hosts)   | Full                    | Full-mesh per pod, full at core| Some HPC          |
| Hypercube   | $\log_2 N$          | $N/2 \cdot b$           | $N \log_2 N / 2$ links       | Older HPC (Cray)  |
| Torus 2D    | $\sqrt{N}$          | $\sqrt{N} \cdot b$      | $2N$ links                   | Cray T3D, IBM Blue Gene |
| Torus 3D    | $N^{1/3}$           | $N^{2/3} \cdot b$       | $3N$ links                   | Cray T3E, BG/L    |
| Dragonfly   | 2-4 (hierarchical)  | Variable                | Multi-tier groups            | Cray Slingshot, Aries |

For Ethernet DC (uniform, switched): Clos wins on operational simplicity and is the de facto standard.

For HPC (regular, sync-heavy, RDMA): torus and dragonfly are competitive because the diameter is lower in absolute hops and the wiring cost grows slower.

Modern AI training clusters (NVIDIA SuperPOD, Google TPU v4/v5): often a hybrid — local NVLink/ICI for tight tensor-parallel groups, Clos Ethernet for cluster-wide. The spine-leaf math still applies to the Ethernet layer.

### 14.3 Bisection bandwidth

Bisection bandwidth = the bandwidth across the worst cut that splits the fabric in half. For a Clos with $L$ leaves and $S$ spines each at $b_u$:

$$
B_{\text{bisect}} = \frac{S \cdot L \cdot b_u}{2}
$$

The "/2" because the cut counts each link once. For full-bisection, half the host bandwidth must equal the cut bandwidth:

$$
\frac{H \cdot b_h}{2} \;\leq\; B_{\text{bisect}} \quad \Leftrightarrow \quad H \cdot b_h \leq S \cdot L \cdot b_u
$$

which simplifies to OSR $\leq 1$. A line-rate fabric is exactly a full-bisection fabric.

### 14.4 When to leave Clos behind

Clos breaks down at extreme scale (≥ 100k hosts) where:

- Cabling cost per port crosses $1000 (super-spine cross-connects).
- Per-DC physical layout (multiple buildings, different floors) requires hierarchical fault domains beyond 5-stage.
- Workload locality becomes overwhelming (e.g., 99% of traffic is rack-local for AI training).

Then operators move to:

- **Multi-site EVPN with WAN backbones** (effectively a 7-stage Clos plus WAN).
- **Custom topologies** like Facebook F16 (specialised "FBOSS" SDN), Google's Jupiter Apollo, AWS's "Brick" — each variations on Clos with custom silicon.
- **Optical circuit switching** at the super-spine layer (Google Mission Apollo) — bypass packet switching for bulk flows.

---

## 15. Operational Practices

### 15.1 Show commands across vendors

```bash
# Cisco NX-OS — fabric overview
show ip route summary
show bgp l2vpn evpn summary
show nve peers
show nve vni
show fabric forwarding ip local-host-db all-vrfs
show interface ethernet 1/1 transceiver details

# Arista EOS
show ip route summary
show bgp evpn summary
show vxlan vtep
show vxlan vni
show interfaces counters errors
show interfaces ethernet 1 transceiver

# Juniper Junos
show route summary
show bgp summary | match EVPN
show evpn database
show ethernet-switching mac-table
show interfaces et-0/0/0 extensive
```

### 15.2 Common diagnostics

```bash
# Check ECMP installed paths to a destination
# NX-OS:
show ip route 10.20.0.0/16
# expect "ubest/mbest: N/N" with N >= 2

# Verify VXLAN encap counters
# Arista:
show vxlan counters
# look for non-zero "VxlanEgress" packets

# BFD session state
# all vendors:
show bfd neighbor
# state should be "Up" and timers match config

# EVPN MAC mobility events
show bgp evpn mac-mobility
# count > 0 indicates host moved (or duplicate MAC = bug)
```

### 15.3 Tuning checklist (standard production fabric)

1. **BFD:** every leaf-spine session, `interval 100ms multiplier 3`.
2. **BGP timers:** `timers 10 30` (vs default 60/180).
3. **maximum-paths:** $\geq S$.
4. **MTU:** 9216 underlay everywhere.
5. **ECN:** enabled on all queue 3 (or whichever priority handles bulk).
6. **PFC:** enabled only if RDMA is present, otherwise disabled (avoid PFC storms).
7. **ARP suppression:** enabled.
8. **Anycast gateway:** all SVIs on all leaves use shared anycast MAC.
9. **Route filter:** BGP advertises only loopbacks (underlay) + EVPN routes (overlay). No host /32 leaks.
10. **VTEP loopback:** dedicated `Loopback1` for VXLAN source.
11. **AS_PATH filtering:** prevent transit (eBGP-only fabric should never carry external prefixes through the fabric).
12. **Logging:** BFD up/down, BGP up/down, NVE peer up/down → SIEM.

### 15.4 Fabric-wide audit script

```bash
#!/bin/bash
# Snapshot per-leaf BGP state and route count for drift detection
for leaf in leaf-{01..32}; do
  ssh $leaf "show bgp summary | grep -E 'Total|Established'" \
    > /tmp/bgp-$leaf.txt
  ssh $leaf "show ip route summary | grep ebgp" \
    >> /tmp/bgp-$leaf.txt
done
diff /tmp/bgp-leaf-01.txt /tmp/bgp-leaf-02.txt
# expect identical except for self-routes
```

---

## 16. Capacity Planning Worksheets

### 16.1 Inputs

- Target host count $H$
- Per-host bandwidth $b_h$
- Acceptable OSR (1:1, 2:1, 3:1)
- Per-leaf host port count $p_h$ (typically 48)
- Per-leaf uplink port count $p_u$ (typically 6 or 8)
- Per-spine port count $p_s$ (typically 32 or 64)

### 16.2 Derivations

- $L = \lceil H / p_h \rceil$
- $\text{OSR}_{\text{actual}} = (p_h \cdot b_h) / (p_u \cdot b_u)$ — must $\leq$ target.
- $S = p_u$ (one uplink per spine for full path diversity).
- Check: $S \leq p_s/L$ — i.e., spines have enough leaf-facing ports.
- Total fabric bisection: $S \cdot L \cdot b_u / 2$.

### 16.3 Worked size points

**Small (1 rack):** 1 leaf, 0 spines. ToR-only, no fabric. 48 hosts.

**Medium (1 row):** 4 leaves × 2 spines, 2:1 OSR, 192 hosts.

**Large (1 building):** 32 leaves × 4 spines, 2:1 OSR, 1536 hosts.

**Hyper (1 DC):** 64 leaves × 8 spines + super-spine, 1:1 OSR, 3072 hosts per pod, multi-pod for 50k+.

### 16.4 Cost components (rough 2024)

| Component                    | Unit cost | Notes                        |
|------------------------------|-----------|------------------------------|
| 32-port 100G leaf            | $15–25k    | TOR-class (Tomahawk-2)       |
| 32-port 400G spine           | $50–80k    | Tomahawk-3                   |
| 100G QSFP28 SR4 transceiver  | $100–200    | Multimode, short-reach       |
| 100G QSFP28 LR4 transceiver  | $300–800    | Singlemode, long-reach       |
| 400G QSFP-DD                 | $1k–3k      | Drops fast year-on-year      |
| 30m OM4 fibre patch          | $50         |                              |
| BGP-EVPN license             | varies     | Some vendors lock features   |

For 32-leaf × 4-spine fabric: leaves $\approx \$640k$, spines $\approx \$240k$, optics $\approx \$50k$, fibre $\approx \$10k$. Total $\approx \$1M$ for 1500-host fabric.

---

## 17. Edge Cases and Gotchas

### 17.1 The ARP-suppression mismatch

If one leaf has ARP suppression enabled and another doesn't, BUM behavior diverges. Specifically: a host on the suppressed leaf may see no ARP response when the answering host is on the un-suppressed leaf with a stale MAC entry. Fix: enable ARP suppression fabric-wide.

### 17.2 The MTU mismatch domino

If two leaves have differing MTU configured for the underlay, large VXLAN-encapsulated packets get fragmented or dropped. Symptom: TCP works but jumbo or NFS or ping with large size fails. Fix: standardise MTU; verify with `ping <vtep> df-bit size 9000`.

### 17.3 The MAC mobility ping-pong

If two VLANs accidentally share a MAC (e.g., misconfigured VM cloned), MAC mobility increments forever. EVPN's MAC mobility extended community includes a "sticky" bit; vendors set it on duplicate detection (e.g., 5 moves in 180s → freeze).

### 17.4 The spine-as-route-reflector overload

Spines often act as RR for EVPN. With many leaves and many MACs, RR memory and CPU can saturate. Mitigations:

- Use a dedicated RR (control-plane only) instead of spine-as-RR.
- Filter EVPN routes by VNI (route-target constraint, RFC 4684).
- Tune `maximum-prefix` per neighbor.

### 17.5 The single-transceiver SPOF

A bad transceiver (CRC errors, link flap) can cause one leaf-spine link to flap and re-converge BFD repeatedly. Symptoms: ECMP rebalancing every few minutes, microbursts on remaining spines. Fix: monitor `show interface ethernet 1/N counters errors` and replace optic.

### 17.6 The leaf-without-default-route

In an underlay with strict route filters (only loopbacks), a leaf must learn `0.0.0.0/0` from the border to reach external destinations. If the route filter drops it, internet egress fails for any anycast-gateway tenant. Fix: explicit allow for `0.0.0.0/0` from border leaves only.

### 17.7 The orphaned ESI

If a host is dual-attached via ESI to leaves A and B, but the LAG is misconfigured (different ESIs on A and B), the fabric thinks two separate hosts exist with the same MAC. Result: rapid MAC flap between two VTEPs. Fix: identical ESI on all members of the same multi-homed group.

---

## 18. Hardening the Fabric

### 18.1 BGP authentication

```bash
# Cisco NX-OS — TCP-AO (RFC 5925)
key chain BGP-AUTH
  key 1
    key-string 7 <encrypted>
    cryptographic-algorithm hmac-sha-256
router bgp 65001
  neighbor 10.255.0.1
    authentication keychain BGP-AUTH
```

### 18.2 GTSM (Generalized TTL Security Mechanism, RFC 5082)

eBGP packets must arrive with TTL=255 (set on send, checked on receive). Prevents off-link attackers from injecting BGP. Set `neighbor X ebgp-multihop 1` and `neighbor X ttl-security hops 1`.

### 18.3 RPKI

Route origin validation via RPKI applies to internet-facing border leaves. Internal eBGP-EVPN doesn't need RPKI (no public ASNs).

### 18.4 Underlay separation

Keep management plane (SSH, SNMP, Syslog) on a dedicated mgmt VRF, never on the underlay VRF. Compromise of one shouldn't compromise the other.

### 18.5 Storm control

```bash
# Arista EOS — broadcast storm control
interface Ethernet1
  storm-control broadcast level 1
  storm-control multicast level 1
```

Limits BUM rate per port, prevents a misbehaving host from saturating BUM forwarding.

---

## 19. Observability

### 19.1 Streaming telemetry

Modern fabrics use gNMI/gRPC streaming telemetry instead of SNMP polling. Subscribe to:

- `interfaces/interface/state/counters/in-octets`
- `interfaces/interface/state/counters/out-octets`
- `bgp/neighbors/neighbor/state/session-state`
- `qos/interfaces/interface/queues/queue/state/dropped-pkts`
- `nve/peers/peer/state/peer-state`

Push to a TSDB (Prometheus, InfluxDB) and graph.

### 19.2 INT (In-band Network Telemetry)

Modern silicon (Tofino, Tomahawk-3) supports INT — telemetry inserted into packets in transit. Useful for per-hop latency and queue depth on the actual path the packet took. Requires programmable data plane.

### 19.3 PCAP at the leaf

Mirror traffic from a host port to a packet broker for full PCAP. Modern leaves support sFlow/NetFlow/IPFIX as a sample-based alternative.

```bash
# Cisco NX-OS — sFlow
sflow agent-ip 10.0.0.1
sflow collector-ip 10.99.0.10 vrf default
sflow data-source interface Ethernet1/1-48
```

---

## 20. Migration Patterns

### 20.1 Brownfield to spine-leaf

Existing legacy core-aggregation-access fabric. Migration:

1. Build new spine-leaf alongside (no impact).
2. Connect new leaves to legacy via L3 link (BGP) or L2 trunk (VLAN extension).
3. Migrate workloads one rack at a time.
4. Decom legacy when last workload is moved.

Risks: workload mobility during migration if legacy uses HSRP and new uses anycast — gateway IP must remain reachable from both.

### 20.2 Adding a spine

Start with $S$ spines; add $S+1$th. Procedure:

1. Cable new spine to all leaves.
2. Configure BGP on new spine peering all leaves.
3. Enable BFD.
4. Verify routes propagating.
5. Increase `maximum-paths` on all leaves to $S+1$.

ECMP automatically rebalances. Capacity goes from $S \cdot b_u \cdot L$ to $(S+1) \cdot b_u \cdot L$.

### 20.3 Adding a leaf

Plug in, configure, peer BGP to all spines, advertise new prefixes. Hosts come online via DHCP on the local leaf. Anycast gateway makes the new leaf indistinguishable from existing ones to the tenant.

### 20.4 Adding a pod (5-stage)

Each new pod is its own 3-stage Clos. Pod spines peer to super-spine. This is a major project — weeks of cabling, days of staging.

---

## 21. Summary of Math Identities

For quick recall:

```
OSR              = (p_h · b_h) / (p_u · b_u)
ECMP fan-out     = S
Bisection BW     = S · L · b_u / 2
VXLAN overhead   = 50 B (IPv4) or 70 B (IPv6)
Underlay MTU     ≥ tenant MTU + 50
Capacity loss    = k/S  for k spines down out of S
BFD detect       = interval × multiplier
Per-port buffer  = B · α / (1 + α · N_congested)   [Choudhury-Hahne DT]
Strictly NB Clos = m ≥ 2n − 1
Rearrangeably NB = m ≥ n
Leaves max       = p_s (spine port count)
Fabric ports     = 2 · S · L (cables × 2 endpoints)
```

These ten identities cover 90% of operational sizing decisions in a spine-leaf fabric.

---

## See Also

- `networking/bgp` — full BGP deep dive (path attributes, best-path, RR scaling)
- `networking/vxlan` — VXLAN encapsulation, VTEP semantics, multicast vs ingress replication
- `networking/ecmp` — ECMP hashing, polarization, resilient hashing
- `networking/evpn-advanced` — advanced EVPN (Type 5, multi-homing, multi-site)
- `networking/spanning-tree` — STP/RSTP/MSTP for the L2 contrast
- `ramp-up/spine-leaf-eli5` — narrative-shaped intro to spine-leaf
- `ramp-up/spanning-tree-eli5` — narrative-shaped intro to STP

---

## References

- **RFC 7348** — *Virtual eXtensible Local Area Network (VXLAN): A Framework for Overlaying Virtualized Layer 2 Networks over Layer 3 Networks*, Mahalingam et al., August 2014.
- **RFC 7432** — *BGP MPLS-Based Ethernet VPN*, Sajassi et al., February 2015.
- **RFC 8365** — *A Network Virtualization Overlay Solution Using Ethernet VPN (EVPN)*, Sajassi et al., March 2018.
- **RFC 8584** — *Framework for Ethernet VPN Designated Forwarder Election Extensibility*, Rabadan et al., April 2019.
- **RFC 9135** — *Integrated Routing and Bridging in Ethernet VPN (EVPN)*, Sajassi et al., October 2021.
- **RFC 9251** — *Internet Group Management Protocol (IGMP) and Multicast Listener Discovery (MLD) Proxies for Ethernet VPN (EVPN)*, Sajassi et al., June 2022.
- **RFC 5880** — *Bidirectional Forwarding Detection (BFD)*, Katz & Ward, June 2010.
- **RFC 5881** — *BFD for IPv4 and IPv6 (Single Hop)*, Katz & Ward, June 2010.
- **RFC 6996** — *Autonomous System (AS) Reservation for Private Use*, Mitchell, July 2013.
- **RFC 4271** — *A Border Gateway Protocol 4 (BGP-4)*, Rekhter, Li & Hares, January 2006.
- **RFC 7911** — *Advertisement of Multiple Paths in BGP*, Walton et al., July 2016.
- **RFC 3168** — *The Addition of Explicit Congestion Notification (ECN) to IP*, Ramakrishnan, Floyd & Black, September 2001.
- **RFC 9028** — *Native NAT Traversal Mode for the Host Identity Protocol*, Keränen, Melén & Komu, June 2021. (DCQCN background.)
- Charles Clos — *"A Study of Non-Blocking Switching Networks,"* Bell System Technical Journal, Vol. 32, No. 2, March 1953, pp. 406–424.
- Mohammad Al-Fares, Alexander Loukissas, Amin Vahdat — *"A Scalable, Commodity Data Center Network Architecture,"* SIGCOMM 2008.
- Arjun Singh et al. — *"Jupiter Rising: A Decade of Clos Topologies and Centralized Control in Google's Datacenter Network,"* SIGCOMM 2015.
- Mohammad Alizadeh et al. — *"Data Center TCP (DCTCP),"* SIGCOMM 2010.
- Mohammad Alizadeh et al. — *"CONGA: Distributed Congestion-Aware Load Balancing for Datacenters,"* SIGCOMM 2014.
- Yibo Zhu et al. — *"Congestion Control for Large-Scale RDMA Deployments (DCQCN),"* SIGCOMM 2015.
- Choudhury & Hahne — *"Dynamic Queue Length Thresholds for Shared-Memory Packet Switches,"* IEEE/ACM ToN, 1998.
- Cisco — *VXLAN Network with EVPN Control Plane Design Guide*, Cisco Validated Design.
- Arista — *EVPN VXLAN Routing Configuration Guide*, EOS documentation.
- Juniper — *EVPN Feature Guide*, Junos documentation.
- IEEE 802.1Q-2018 — *Bridges and Bridged Networks*. (For VLAN, MSTP context.)
- IEEE 802.1Qbb — *Priority-based Flow Control*. (PFC specification.)
- IEEE 802.1Qau — *Congestion Notification*. (QCN, DCQCN basis.)
