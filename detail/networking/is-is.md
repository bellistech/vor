# The Mathematics of IS-IS — SPF on a Two-Level Hierarchy

> *IS-IS (Intermediate System to Intermediate System) runs the same Dijkstra SPF algorithm as OSPF but on a fundamentally different area model — a two-level hierarchy with no backbone dependency. The math covers SPF complexity, TLV encoding efficiency, metric analysis, and flooding domain scaling.*

---

## 1. SPF Algorithm — Same Math, Different Topology

### Dijkstra Complexity (Identical to OSPF)

$$O(V^2) \quad \text{naive}$$

$$O((V + E) \log V) \quad \text{binary heap}$$

$$O(V \log V + E) \quad \text{Fibonacci heap}$$

### The Two-Level Hierarchy

IS-IS uses Level-1 (intra-area) and Level-2 (inter-area) routing. Unlike OSPF's mandatory backbone (Area 0), IS-IS Level-2 is a connected backbone of L2-capable routers.

| Topology | SPF Runs Required |
|:---|:---|
| L1-only router | 1 SPF over L1 LSDB |
| L2-only router | 1 SPF over L2 LSDB |
| L1/L2 router | 2 SPF runs (one per level) |

### SPF Computation per Level

For a network with $V$ total routers split into $A$ L1 areas of $V/A$ routers each, with $B$ L2 routers:

$$T_{L1} \propto \left(\frac{V}{A}\right)^2 \quad \text{(per area)}$$

$$T_{L2} \propto B^2$$

### Comparison: IS-IS vs OSPF Area Models

| Property | IS-IS | OSPF |
|:---|:---|:---|
| Backbone | L2 domain (any topology) | Area 0 (must be contiguous) |
| Area boundary | On the link (between routers) | On the router (ABR) |
| Virtual links | Not needed | Required if Area 0 is partitioned |
| Route leaking | L2→L1 (controlled) | Inter-area via ABR summary LSAs |

---

## 2. Metric System — Narrow vs Wide

### Narrow Metrics (Original)

| Field | Bits | Range |
|:---|:---:|:---:|
| Default metric | 6 | 0-63 |
| Path maximum | 10 bits total | 0-1,023 |

### Wide Metrics (RFC 5305)

| Field | Bits | Range |
|:---|:---:|:---:|
| Default metric | 24 | 0-16,777,215 |
| Path maximum | 32 bits | 0-4,294,967,295 |

### Why Wide Metrics Matter

With narrow metrics, differentiation is limited:

$$\text{Cost}_{narrow} = \lfloor \frac{R}{BW} \rfloor \quad \text{where } R = \text{reference, max cost = 63}$$

| Link Speed | Narrow Cost (ref=6300) | Wide Cost (ref=10^10) |
|:---|:---:|:---:|
| T1 (1.544M) | 63 (max) | 6,476,684 |
| 10 Mbps | 63 (max) | 1,000,000 |
| 100 Mbps | 63 | 100,000 |
| 1 Gbps | 6 | 10,000 |
| 10 Gbps | 1 | 1,000 |
| 100 Gbps | 1 (same!) | 100 |

Narrow metrics cannot distinguish between 10 Mbps and 100 Mbps links. Wide metrics provide 6+ orders of magnitude of granularity.

### Path Cost with Wide Metrics

$$\text{Path Cost} = \sum_{i=1}^{h} w_i \quad \text{where } w_i \leq 16,777,215$$

Maximum path cost with 32-bit accumulator: $2^{32} - 1 = 4,294,967,295$.

Max hops before overflow at max per-link cost: $\lfloor 4,294,967,295 / 16,777,215 \rfloor = 255$ hops.

---

## 3. TLV Structure — Encoding Efficiency

### TLV Format

Every IS-IS PDU carries data in Type-Length-Value triplets:

| Component | Size | Purpose |
|:---|:---:|:---|
| Type | 1 byte | Identifies the TLV |
| Length | 1 byte | Length of value field |
| Value | 0-255 bytes | Data payload |

### Maximum TLV Capacity per LSP

An IS-IS LSP has a maximum size (typically 1,492 bytes for ISO 10589, or up to MTU-based).

$$\text{TLVs per LSP} = \lfloor \frac{L_{LSP} - H_{LSP}}{S_{TLV_{avg}}} \rfloor$$

Where $H_{LSP} = 27$ bytes (LSP header).

| LSP Size | Header | Available | Avg TLV Size | TLVs/LSP |
|:---:|:---:|:---:|:---:|:---:|
| 1,492 B | 27 B | 1,465 B | 50 B | 29 |
| 1,492 B | 27 B | 1,465 B | 100 B | 14 |
| 9,000 B (jumbo) | 27 B | 8,973 B | 50 B | 179 |

### LSP Fragmentation

When a router's TLV data exceeds one LSP, it generates fragments (LSP number 0, 1, 2, ...):

$$F = \lceil \frac{D_{total}}{L_{LSP} - H_{LSP}} \rceil$$

A router with 500 IS-IS adjacencies (each requiring ~20 bytes):

$$F = \lceil \frac{500 \times 20}{1,465} \rceil = \lceil 6.83 \rceil = 7 \text{ LSP fragments}$$

---

## 4. Flooding Domain Scaling

### LSDB Size

$$|LSDB| = \sum_{r=1}^{R} F_r$$

Where $F_r$ = number of LSP fragments originated by router $r$.

### Flooding Traffic

When a link change occurs, the originating router re-floods its LSP(s):

$$\text{Flood messages} = F_r \times (N_{adj} - 1)$$

Where $N_{adj}$ = number of adjacencies on the router.

### Convergence After Link Failure

$$T_{convergence} = T_{detect} + T_{flood} + T_{SPF} + T_{RIB} + T_{FIB}$$

| Phase | Typical Duration |
|:---|:---:|
| Detection (BFD) | 50-150 ms |
| LSP flooding | 10-50 ms |
| SPF computation | 1-10 ms |
| RIB update | 1-5 ms |
| FIB programming | 5-50 ms |
| **Total** | **67-265 ms** |

### SPF Throttling (Same Model as OSPF)

$$T_{wait}(n) = \min(T_{init} \times 2^n, T_{max})$$

Typical: $T_{init} = 50$ ms, $T_{max} = 5$ sec.

---

## 5. Mesh Groups and Flooding Reduction

### The Problem

In a full mesh of $N$ routers, each LSP is flooded $N - 1$ times — but each router only needs one copy. Mesh groups suppress redundant flooding.

### Without Mesh Groups

$$\text{Total LSP copies} = R \times F_{avg} \times (N - 1)$$

### With Mesh Groups

Routers in a mesh group only flood to non-mesh-group neighbors:

$$\text{Total LSP copies} = R \times F_{avg} \times (N_{non-mesh})$$

| Routers | Mesh Size | Without Groups | With Groups | Reduction |
|:---:|:---:|:---:|:---:|:---:|
| 20 | 15 | 19/router | 5/router | 74% |
| 50 | 40 | 49/router | 10/router | 80% |
| 100 | 80 | 99/router | 20/router | 80% |

---

## 6. IS-IS for IP — Multi-Topology Support

### The Problem

IPv4 and IPv6 may have different topologies (some links IPv4-only, some IPv6-only). Multi-topology IS-IS (RFC 5120) runs separate SPF computations.

### Computational Cost

$$T_{SPF\_MT} = \sum_{t=1}^{T} O((V_t + E_t) \log V_t)$$

Where $T$ = number of topologies (typically 2: IPv4 + IPv6).

In the common case where both topologies are identical ($V_1 = V_2$, $E_1 = E_2$):

$$T_{SPF\_MT} = 2 \times T_{SPF} \quad \text{(exactly double the computation)}$$

---

## 7. Summary of Formulas

| Formula | Math Type | Application |
|:---|:---|:---|
| $O((V + E) \log V)$ | Algorithmic complexity | SPF computation |
| $(V/A)^2$ | Quadratic partition | Level-based scaling |
| $\sum w_i \leq 2^{32}$ | Bounded summation | Path cost (wide metrics) |
| $\lceil D / (L - H) \rceil$ | Ceiling division | LSP fragmentation |
| $T_{init} \times 2^n$ | Exponential backoff | SPF throttling |
| $R \times F \times (N-1)$ | Combinatorial | Flooding traffic |

## Prerequisites

- graph theory, shortest path algorithms, binary arithmetic

## Complexity

| Operation | Time | Space |
|-----------|------|-------|
| SPF (Dijkstra) | O(n log n) | O(n) |
| LSP flooding | O(n) | O(n) |
| Route lookup | O(log n) | O(n) |

---

*IS-IS runs underneath some of the largest networks on Earth — most major ISP backbones and hyperscaler data centers chose it over OSPF precisely because its two-level hierarchy scales without the constraints of a mandatory backbone area. Same algorithm, different topology math.*
