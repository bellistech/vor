# The Mathematics of LDP — Label Space Efficiency & Convergence Analysis

> *LDP's distributed label allocation creates a fascinating tension between memory consumption and convergence speed: liberal retention caches O(N*P) bindings for instant failover, conservative retention saves memory but pays with re-signaling delay, and the FEC-to-label mapping itself follows graph-theoretic properties of the underlying IGP shortest-path tree.*

---

## 1. Label Information Base Scaling (Memory Complexity)

### LIB Size Under Liberal Retention

With liberal label retention, each LSR stores label bindings received from ALL LDP peers, not just the best next-hop. For a network with $N$ LSRs, $P$ prefixes advertised by LDP, and average peer count $k$:

$$|LIB| = P \times k$$

Each LIB entry stores: FEC prefix (5 bytes), label (3 bytes), peer ID (6 bytes), flags (2 bytes):

$$M_{LIB} \approx P \times k \times 16 \text{ bytes}$$

### Conservative vs Liberal

Under conservative retention, only the best next-hop binding is kept:

$$|LIB_{conservative}| = P$$

$$|LIB_{liberal}| = P \times k$$

| LSRs $N$ | Prefixes $P$ | Avg Peers $k$ | Liberal LIB | Conservative LIB |
|:---:|:---:|:---:|:---:|:---:|
| 50 | 1,000 | 3 | 3,000 (48 KB) | 1,000 (16 KB) |
| 200 | 5,000 | 4 | 20,000 (320 KB) | 5,000 (80 KB) |
| 500 | 10,000 | 5 | 50,000 (800 KB) | 10,000 (160 KB) |
| 1,000 | 50,000 | 6 | 300,000 (4.8 MB) | 50,000 (800 KB) |

The liberal-to-conservative ratio is exactly $k$, the average peer degree. In practice, filtering labels to loopbacks only ($P \approx N$) keeps the LIB manageable even with liberal retention.

---

## 2. Convergence Time Analysis (Session Establishment)

### LDP Session Bring-Up Phases

Total LDP session establishment time:

$$T_{session} = T_{hello} + T_{tcp} + T_{init} + T_{binding}$$

Where:
- $T_{hello}$: time to discover neighbor via UDP hellos
- $T_{tcp}$: TCP three-way handshake
- $T_{init}$: Initialization message exchange
- $T_{binding}$: label binding download

### Hello Discovery Time

Hellos are sent at interval $H_I$ (default 5s for link, 15s for targeted). Worst case discovery time:

$$T_{hello} = H_I + \delta$$

Expected (both sides sending, independent uniform phase):

$$E[T_{hello}] = \frac{H_I}{2}$$

### Binding Download Time

For $P$ prefixes, each Label Mapping message containing $b$ bindings, at TCP throughput $R$:

$$T_{binding} = \frac{P}{b} \times \frac{S_{msg}}{R}$$

Where $S_{msg}$ is the average message size. With 100 bindings per PDU (~3 KB) at 10 Mbps TCP throughput:

$$T_{binding} = \frac{10000}{100} \times \frac{3000 \times 8}{10^7} = 100 \times 0.0024 = 0.24 \text{s}$$

### Total Convergence Budget

| Phase | Typical Duration | Dominant Factor |
|:---|:---:|:---|
| Hello discovery | 2.5s | Hello interval |
| TCP handshake | <1ms (LAN) | RTT |
| Initialization | <10ms | 2 messages |
| Binding exchange | 0.1--1s | Number of prefixes |
| **Total** | **~3--4s** | Hello interval dominates |

With targeted LDP (pre-existing session), reconvergence skips hello and TCP phases:

$$T_{reconverge}^{targeted} = T_{binding} \approx 0.1\text{--}1 \text{s}$$

---

## 3. Label Space Utilization (Allocation Efficiency)

### Dynamic Label Range

The usable label space is $[16, L_{max}]$ where $L_{max}$ is platform-dependent (typically $2^{20} - 1 = 1,048,575$ for 20-bit MPLS labels).

Usable labels:

$$|L| = L_{max} - 16 + 1 = 1,048,560$$

### Per-Platform vs Per-Interface Label Space

Per-platform (default): one label space shared across all interfaces. Each FEC gets one local label regardless of how many peers request it:

$$|labels_{allocated}| = P$$

Per-interface: each interface has its own label space. The same FEC can have different labels on different interfaces:

$$|labels_{allocated}| = P \times I$$

Where $I$ is the number of LDP-enabled interfaces.

| Mode | Labels Used | Label Reuse | Use Case |
|:---|:---:|:---|:---|
| Per-platform | $P$ | Same label all interfaces | Standard Ethernet MPLS |
| Per-interface | $P \times I$ | Different per interface | ATM/Frame-Relay MPLS |

### Fragmentation

With label allocation and deallocation over time, the label space can fragment. If labels are allocated sequentially and freed randomly, after $A$ allocations and $D$ deallocations:

$$|free| = |L| - (A - D)$$

But the fragmentation ratio (largest contiguous block / total free) can degrade. In practice, LDP implementations use bitmap allocators that avoid fragmentation entirely.

---

## 4. Penultimate Hop Popping Decision (Forwarding Optimization)

### CPU Cost Analysis

At the egress LSR without PHP, each packet requires two lookups:

$$C_{no\_PHP} = C_{label} + C_{IP}$$

With PHP, the penultimate hop pops the label, so the egress sees a plain IP packet:

$$C_{PHP} = C_{IP}$$

Savings per packet:

$$\Delta C = C_{label} \approx 50\text{--}100 \text{ ns (hardware lookup)}$$

For a 10M pps egress router:

$$\text{Saved cycles} = 10^7 \times C_{label}$$

### When PHP Hurts

PHP removes the MPLS header, losing the EXP (TC) bits used for QoS classification. For QoS-sensitive traffic, Explicit NULL (label 0) preserves the MPLS header through the egress:

$$C_{explicit\_null} = C_{label} + C_{IP} + \text{QoS classification from EXP bits}$$

The tradeoff: PHP saves CPU but loses QoS; Explicit NULL costs CPU but preserves QoS marking.

---

## 5. LDP-IGP Sync Blackhole Window (Availability Analysis)

### The Problem

When an interface comes up, IGP converges before LDP:

$$T_{IGP} < T_{LDP}$$

During the gap $[T_{IGP}, T_{LDP}]$, IGP directs traffic to the interface but no MPLS label is available, causing drops.

### Blackhole Duration

$$T_{blackhole} = T_{LDP} - T_{IGP}$$

$$T_{IGP} \approx T_{SPF} + T_{flood} \approx 0.1\text{--}1 \text{s}$$

$$T_{LDP} \approx T_{hello} + T_{tcp} + T_{init} + T_{binding} \approx 3\text{--}5 \text{s}$$

$$T_{blackhole} \approx 2\text{--}4 \text{s}$$

### With LDP-IGP Sync

LDP-IGP sync advertises maximum metric on the link until LDP is operational:

$$T_{blackhole}^{sync} = 0$$

The cost is delayed link utilization:

$$T_{delay} = T_{LDP} - T_{IGP} + T_{sync\_holddown}$$

| Scenario | Blackhole Duration | Link Utilization Delay |
|:---|:---:|:---:|
| No sync | 2--4s | 0s (immediate) |
| Sync, no holddown | 0s | 3--5s |
| Sync, holddown=10s | 0s | 10s |

The holddown timer adds safety margin for label exchange to complete.

---

## 6. Session State Scaling (LDP vs RSVP-TE vs SR)

### Control Plane State Complexity

LDP state scales with neighbors and prefixes:

$$S_{LDP} = O(N \times P)$$

Where $N$ = LDP neighbors and $P$ = prefixes. State is per-neighbor (one TCP session, multiple bindings).

RSVP-TE state scales with tunnels:

$$S_{RSVP} = O(T \times H)$$

Where $T$ = number of tunnels and $H$ = average path length (hops). Every transit LSR maintains per-tunnel state.

Segment Routing has no signaling state:

$$S_{SR} = O(0) \text{ (control plane)} + O(N) \text{ (IGP extensions)}$$

### Full-Mesh Tunnel Scaling

For $N$ routers requiring full-mesh connectivity:

| Protocol | Control Plane State | Signaling Messages |
|:---|:---:|:---:|
| LDP | $O(N^2)$ bindings (liberal) | $O(N)$ per event |
| RSVP-TE | $O(N^2 \times H)$ per-tunnel | $O(N^2)$ refresh |
| SR-MPLS | $O(N)$ SIDs in IGP | $O(N)$ per event |

This is why RSVP-TE struggles beyond ~1,000 tunnels and why SR is displacing both LDP and RSVP-TE in modern networks.

---

## 7. Graceful Restart Recovery (Stale Binding Analysis)

### Forwarding Continuity

During graceful restart, the restarting LSR preserves its LFIB (Label Forwarding Information Base) while the control plane restarts. The helper peer maintains bindings for a recovery time $T_R$:

$$P(\text{forwarding preserved}) = P(T_{restart} < T_R)$$

If restart time follows a distribution with mean $\mu$ and the recovery timer is fixed at $T_R$:

$$P(\text{success}) = P(T_{restart} < T_R) = F_{restart}(T_R)$$

For exponentially distributed restart times:

$$P(\text{success}) = 1 - e^{-T_R / \mu}$$

| Recovery Timer $T_R$ | Mean Restart $\mu$ | $P(\text{success})$ |
|:---:|:---:|:---:|
| 60s | 10s | 99.75% |
| 120s | 30s | 98.17% |
| 120s | 60s | 86.47% |
| 180s | 60s | 95.02% |

### Stale Binding Cleanup

After recovery, the restarting LSR re-advertises current bindings. Bindings not re-advertised within $T_R$ are considered stale and deleted:

$$|stale| = |LIB_{before}| - |LIB_{recovered}|$$

Stale entries indicate prefixes that disappeared during restart (topology change coinciding with restart — the worst case for GR).

---

*LDP's mathematics reveal a protocol optimized for simplicity over expressiveness: it maps IGP topology to labels with O(N) signaling complexity, achieves instant failover via liberal retention at the cost of O(k) memory multiplier, and converges in seconds dominated by hello timers rather than computation. The shift to Segment Routing eliminates LDP's state entirely, but understanding LDP's scaling properties remains essential for the millions of MPLS routers still running it in production.*

## Prerequisites

- Graph theory (shortest-path trees, neighbor degree, full-mesh scaling)
- Complexity analysis (big-O notation for state and signaling)
- Probability (convergence timing, graceful restart success probability)

## Complexity

- **Beginner:** LIB size calculation, label range utilization, basic convergence timing
- **Intermediate:** Liberal vs conservative retention tradeoffs, LDP-IGP sync blackhole analysis, PHP cost model
- **Advanced:** RSVP-TE vs SR state scaling comparison, graceful restart stale binding probability, per-interface label space fragmentation
