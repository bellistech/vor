# The Mathematics of MPLS — Label Stacking, LSP Computation, and Traffic Engineering

> *MPLS replaces destination-based routing with label-based forwarding, turning a complex IP lookup into a simple table index. The math covers label space combinatorics, stack depth implications, RSVP-TE bandwidth reservation, and FEC equivalence class theory.*

---

## 1. Label Space — Combinatorics

### The Label Field

An MPLS label is 20 bits:

$$L_{total} = 2^{20} = 1,048,576 \text{ possible labels}$$

Reserved labels (0-15): 16 labels for special purposes (Explicit NULL, Router Alert, etc.).

$$L_{usable} = 2^{20} - 16 = 1,048,560$$

### Per-Platform vs Per-Interface Label Space

| Mode | Labels Available per Link | Total Label Capacity |
|:---|:---:|:---|
| Per-platform | $2^{20}$ shared across all interfaces | One label = one FEC globally |
| Per-interface | $2^{20}$ per interface | $I \times 2^{20}$ total (where $I$ = interfaces) |

A router with 48 interfaces in per-interface mode: $48 \times 1,048,576 = 50,331,648$ label bindings possible.

### MPLS Header Structure

Each label entry is exactly 32 bits (4 bytes):

| Field | Bits | Purpose |
|:---|:---:|:---|
| Label | 20 | Forwarding identifier |
| TC (Traffic Class) | 3 | QoS / EXP bits |
| S (Bottom of Stack) | 1 | 1 = last label |
| TTL | 8 | Time to live |

---

## 2. Label Stack Depth — Overhead Analysis

### The Problem

MPLS supports label stacking — multiple labels pushed onto a packet. Each label adds 4 bytes. How does this affect MTU and throughput?

### Stack Overhead

$$O_{stack} = 4 \times D \text{ bytes}$$

Where $D$ = stack depth.

### Common Stack Depths

| Application | Stack Depth | Overhead | Effective MTU (1500) |
|:---|:---:|:---:|:---:|
| Simple LSP | 1 | 4 B | 1,496 |
| L3VPN (VPN + transport) | 2 | 8 B | 1,492 |
| L3VPN over TE tunnel | 3 | 12 B | 1,488 |
| 6PE/6VPE (IPv6 over MPLS) | 2-3 | 8-12 B | 1,488-1,492 |
| EVPN-VXLAN over MPLS | 3-4 | 12-16 B | 1,484-1,488 |
| Segment Routing (deep) | 5-10 | 20-40 B | 1,460-1,480 |

### MTU Planning Rule

$$MTU_{MPLS} = MTU_{link} + 4 \times D_{max}$$

Most providers set core link MTU to 9,216 (jumbo frames) to accommodate deep stacks:

$$9,216 - 4(10) = 9,176 \text{ bytes available for IP payload}$$

---

## 3. FEC Equivalence Classes — Partition Theory

### The Definition

A Forwarding Equivalence Class (FEC) is a set of packets that receive identical forwarding treatment. The total traffic is partitioned into disjoint FECs:

$$\text{Traffic} = \bigsqcup_{i=1}^{F} FEC_i$$

Where $F$ = number of FECs, and $\bigsqcup$ denotes disjoint union.

### FEC Granularity

| FEC Type | Basis | Typical Count | Granularity |
|:---|:---|:---:|:---|
| Per-prefix | Destination IP prefix | $\sim P$ (routing table size) | Coarse |
| Per-VRF | VPN customer | $\sim V$ (VPN count) | Medium |
| Per-flow | 5-tuple hash | $\sim$ millions | Fine |
| Per-TE tunnel | Explicit path | $\sim T$ (tunnel count) | Engineered |

### Label Binding Efficiency

Each FEC requires one label. If a router has $P$ prefixes and $V$ VPNs:

$$L_{needed} = P + V + T + R$$

Where $R$ = reserved/special labels.

For a PE router: $P = 500,000$ (Internet table) + $V = 1,000$ VPNs + $T = 200$ TE tunnels:

$$L_{needed} = 501,200 \quad (48\% \text{ of label space})$$

---

## 4. RSVP-TE Bandwidth Reservation

### The Problem

Traffic Engineering requires reserving bandwidth along a path. This is a constrained shortest path problem.

### Constraint-Based Routing

Find the shortest path $P$ from $s$ to $d$ where every link $e \in P$ has:

$$B_{available}(e) \geq B_{requested}$$

### Available Bandwidth

$$B_{available}(e) = B_{capacity}(e) - \sum_{i} B_{reserved_i}(e)$$

### Worked Example

Path: A → B → C → D. Requesting 500 Mbps.

| Link | Capacity | Reserved | Available | Fits? |
|:---|:---:|:---:|:---:|:---:|
| A→B | 10 Gbps | 6 Gbps | 4 Gbps | Yes |
| B→C | 10 Gbps | 9.8 Gbps | 200 Mbps | **No** |
| A→E→C (alternate) | 10 Gbps | 3 Gbps | 7 Gbps | Yes |
| C→D | 10 Gbps | 5 Gbps | 5 Gbps | Yes |

CSPF selects: A → E → C → D (longer path, but meets constraint).

### Admission Control

$$\text{Accept if: } \forall e \in P: B_{available}(e) \geq B_{requested}$$

**Overbooking ratio:**

$$R_{overbook} = \frac{\sum B_{reserved}}{B_{capacity}}$$

Typical ISP overbooking: $R = 2$-$5\times$ (statistical multiplexing means not all tunnels are fully utilized simultaneously).

---

## 5. LSP Path Computation — CSPF Complexity

### The Algorithm

Constrained Shortest Path First (CSPF) is Dijkstra's algorithm with link pruning:

1. Remove links that don't meet constraints (bandwidth, affinity, SRLG)
2. Run Dijkstra on the pruned graph

### Complexity

$$O((V + E') \log V)$$

Where $E' \leq E$ is the edge set after constraint pruning.

### With SRLG Diversity (Disjoint Paths)

Finding two link-disjoint paths is the **min-cost disjoint path problem**:

$$O(V \times E)$$

Using Suurballe's algorithm. For SRLG-disjoint paths (shared risk link groups), the problem becomes NP-hard in general — heuristics are used.

---

## 6. PHP (Penultimate Hop Popping) — Lookup Optimization

### The Problem

At the egress router, the MPLS label must be removed and an IP lookup performed. PHP moves label removal to the penultimate hop:

**Without PHP:** Egress does: label lookup → label pop → IP lookup (2 lookups).

**With PHP:** Penultimate does: label lookup → pop. Egress does: IP lookup only (1 lookup each).

### Forwarding Rate Impact

If each lookup takes $T_{lookup}$:

$$T_{without\_PHP} = T_{label} + T_{pop} + T_{IP} \quad \text{(at egress)}$$

$$T_{with\_PHP} = \max(T_{label} + T_{pop}, T_{IP}) \quad \text{(distributed across 2 routers)}$$

For hardware-based forwarding ($T_{lookup} \approx$ constant), PHP reduces egress router load by ~50%.

---

## 7. Segment Routing Label Depth vs State

### The Tradeoff

Traditional MPLS: per-hop state (each router maintains label forwarding entries).
Segment Routing: source-routed (label stack encodes the path).

$$\text{State (traditional)}: S = N \times F \quad \text{(N nodes, F FECs)}$$

$$\text{State (SR)}: S = N \quad \text{(only node SIDs)}$$

$$\text{Header overhead (SR)}: O = 4 \times H \quad \text{(H = hops in stack)}$$

| Metric | Traditional MPLS | Segment Routing |
|:---|:---:|:---:|
| Per-node state | $O(F)$ FEC entries | $O(N)$ node SIDs |
| Header overhead | 4-8 bytes (1-2 labels) | $4H$ bytes |
| Path flexibility | TE tunnels (signaled) | Stack (computed) |
| Failure recovery | FRR (pre-computed) | TI-LFA (topology-independent) |

---

## 8. Summary of Formulas

| Formula | Math Type | Application |
|:---|:---|:---|
| $2^{20}$ label space | Exponent | Label capacity |
| $4 \times D$ bytes overhead | Linear | Stack depth cost |
| $B_{cap} - \sum B_{reserved}$ | Subtraction | Available bandwidth |
| $O((V + E')\log V)$ | Algorithmic complexity | CSPF computation |
| $P + V + T$ | Summation | Label requirement |
| $N \times F$ vs $N$ | Linear vs constant | State scaling |

---

*MPLS turned the internet's routing problem inside out — instead of every router making an independent forwarding decision, the ingress router makes one decision and encodes it as a label stack. The math of label space, stack depth, and bandwidth reservation governs how service providers move petabytes per day.*
