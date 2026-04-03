# The Mathematics of Junos — Network Operating System Internals

> *Junos OS is built on a transactional configuration model, a separation of control and forwarding planes, and deterministic route selection through preference and metric hierarchies. Its commit model, PFE architecture, and firewall filter evaluation are all mathematically precise.*

---

## 1. Commit Model (Transactional Configuration)

### The Problem

Junos uses a **candidate configuration** model with atomic commits. Configuration changes are staged, validated, and applied as a transaction — all or nothing.

### The Transaction Model

$$\text{commit}: C_{candidate} \xrightarrow{\text{validate}} \begin{cases}
C_{active} = C_{candidate} & \text{if valid} \\
C_{active} = C_{active} & \text{if invalid (rollback)}
\end{cases}$$

### ACID Properties

| Property | Junos Implementation |
|:---|:---|
| **Atomicity** | All changes apply or none apply |
| **Consistency** | Syntax + semantic validation before apply |
| **Isolation** | `configure exclusive` locks config |
| **Durability** | Committed config persists across reboots |

### Rollback History

$$|\text{rollbacks}| = 50 \quad \text{(rollback 0 through rollback 49)}$$

$$\text{rollback } n \implies C_{active} \leftarrow C_{n}$$

### Commit Timing

$$T_{commit} = T_{validate} + T_{diff} + T_{activate}$$

| Config Size | Validate | Diff | Activate | Total |
|:---:|:---:|:---:|:---:|:---:|
| Small (100 lines) | 1s | 0.5s | 1s | 2.5s |
| Medium (1000 lines) | 3s | 2s | 3s | 8s |
| Large (10,000 lines) | 10s | 5s | 10s | 25s |

### Commit Confirmed (Auto-Rollback)

$$\text{commit confirmed } T \implies \begin{cases}
C_{active} = C_{candidate} & \text{immediately} \\
C_{active} = C_{previous} & \text{if no second commit within } T \text{ minutes}
\end{cases}$$

This is a **deadman switch** — prevents lockout from bad configs.

---

## 2. Routing Table Selection (Preference/Metric Hierarchy)

### The Problem

When multiple routing protocols offer routes to the same destination, Junos selects based on **preference** (administrative distance), then **metric**.

### Selection Algorithm

$$\text{best route} = \arg\min_{r \in \text{candidates}} (\text{preference}(r), \text{metric}(r))$$

Primary key: preference (lower wins). Tiebreaker: metric (lower wins).

### Default Preferences

| Source | Preference | When Used |
|:---|:---:|:---|
| Direct (connected) | 0 | Interface routes |
| Local | 0 | Interface addresses |
| Static | 5 | Manually configured |
| OSPF internal | 10 | Intra-area routes |
| IS-IS Level 1 | 15 | L1 internal |
| IS-IS Level 2 | 18 | L2 internal |
| OSPF external | 150 | Type 1 and Type 2 |
| BGP (eBGP) | 170 | External peers |
| BGP (iBGP) | 170 | Internal peers |

### Route Selection Worked Example

Destination: 10.1.0.0/24

| Protocol | Preference | Metric | Selected? |
|:---|:---:|:---:|:---:|
| Static | 5 | N/A | **Yes** (lowest preference) |
| OSPF internal | 10 | 100 | No |
| BGP | 170 | MED=50 | No |

If the static route is removed:

| Protocol | Preference | Metric | Selected? |
|:---|:---:|:---:|:---:|
| OSPF internal | 10 | 100 | **Yes** |
| BGP | 170 | MED=50 | No |

---

## 3. PFE Architecture (Packet Forwarding Engine)

### The Problem

Junos separates the control plane (RE — Routing Engine) from the forwarding plane (PFE — Packet Forwarding Engine). The PFE uses custom ASICs for line-rate forwarding.

### RE/PFE Split

$$\text{RE}: \text{routing protocols, CLI, SNMP, management} \quad (FreeBSD)$$
$$\text{PFE}: \text{packet lookup, QoS, filtering, forwarding} \quad (microcode on ASIC)$$

### PFE Lookup Pipeline

$$\text{Packet} \xrightarrow{\text{ingress}} \text{Filter} \xrightarrow{\text{route lookup}} \text{NH resolution} \xrightarrow{\text{QoS}} \text{egress}$$

### Forwarding Table Download

$$T_{sync} = \frac{|FIB_{entries}| \times S_{entry}}{BW_{internal}}$$

RE pushes the forwarding table to PFE via an internal link:

| Entries | Entry Size | Internal BW | Sync Time |
|:---:|:---:|:---:|:---:|
| 10,000 | 64 B | 1 Gbps | 5 ms |
| 100,000 | 64 B | 1 Gbps | 50 ms |
| 1,000,000 | 64 B | 1 Gbps | 500 ms |

### Line-Rate Forwarding Capacity

$$\text{PPS}_{max} = \frac{BW_{interface}}{S_{min\_packet} \times 8}$$

| Interface | Min Packet (64B + 20B overhead) | Max PPS |
|:---|:---:|:---:|
| 1 GigE | 84 bytes | 1,488,095 |
| 10 GigE | 84 bytes | 14,880,952 |
| 100 GigE | 84 bytes | 148,809,524 |

### MEMORY to ASIC Path

$$\text{Packet buffer (MEMORY)} \xrightarrow{\text{DMA}} \text{Lookup ASIC} \xrightarrow{\text{result}} \text{Forwarding ASIC} \xrightarrow{\text{rewrite}} \text{Egress buffer}$$

Total forwarding latency: 1-5 microseconds for ASIC-based platforms.

---

## 4. Firewall Filter Evaluation (Sequential Match)

### The Problem

Junos firewall filters (ACLs) are evaluated sequentially — first match wins. Understanding the evaluation model is critical for security.

### Evaluation Algorithm

$$\text{action}(pkt) = \text{action}(\text{first term matching } pkt)$$

$$\text{If no term matches}: \text{implicit discard}$$

Note: Junos has an **implicit deny** at the end of every filter (unlike IOS which has implicit deny only on named ACLs).

### Term Matching

$$\text{match}(pkt, term) = \bigwedge_{c \in \text{conditions}(term)} c(pkt)$$

All conditions within a term are AND'd. Multiple values within a condition are OR'd:

$$\text{source-address } [A, B] = (\text{src} = A) \vee (\text{src} = B)$$

### Filter Performance

ASIC-based evaluation:

$$T_{filter} = O(1) \quad \text{(hardware TCAM on MX/QFX)}$$

Software-based (RE-processed traffic):

$$T_{filter} = O(N) \quad \text{where } N = \text{number of terms}$$

### Policer Math

$$\text{Bandwidth limit}: R_{police}$$
$$\text{Burst size}: B = R_{police} \times T_{burst}$$

Token bucket:

$$B(t) = \min(B_{max}, B(t-\Delta t) + R_{police} \times \Delta t)$$

$$\text{conform} \iff S_{packet} \leq B(t)$$

---

## 5. OSPF SPF Calculation (Dijkstra's Algorithm)

### The Problem

Junos runs Dijkstra's shortest path first algorithm on the link-state database to compute the routing table.

### Algorithm Complexity

$$T_{SPF} = O((V + E) \log V)$$

Where $V$ = routers, $E$ = links.

### SPF Throttling

Junos uses exponential backoff for SPF runs:

$$T_{delay}(n) = \min(T_{initial} \times 2^n, T_{max})$$

Default: $T_{initial} = 200\text{ms}$, $T_{max} = 5000\text{ms}$.

| Event # | Delay |
|:---:|:---:|
| 1 | 200 ms |
| 2 | 400 ms |
| 3 | 800 ms |
| 4 | 1,600 ms |
| 5 | 3,200 ms |
| 6+ | 5,000 ms (capped) |

### SPF Run Time

| Network Size (V) | Links (E) | SPF Time |
|:---:|:---:|:---:|
| 50 | 200 | < 1 ms |
| 500 | 2,000 | 5-10 ms |
| 5,000 | 20,000 | 50-200 ms |

### Area Sizing

$$|V_{area}| \leq 200 \quad \text{(recommended maximum)}$$

$$\text{Areas} = \lceil V_{total} / 200 \rceil$$

---

## 6. Class of Service (QoS Model)

### The Problem

Junos CoS uses schedulers, shapers, and rewrite rules to manage traffic. The scheduling math determines bandwidth allocation.

### Scheduler Configuration

$$\text{Transmit rate} = \min(\text{configured rate}, \text{demand})$$

### Weighted Round Robin

$$BW_i = \frac{W_i}{\sum W_j} \times BW_{available}$$

After priority queue is served:

$$BW_{available} = BW_{link} - BW_{strict\_high}$$

### Worked Example: 1 Gbps Interface

| Queue | Scheduler | Weight/Rate | Bandwidth |
|:---|:---|:---:|:---:|
| Strict-high | Priority | 100 Mbps max | 100 Mbps |
| Queue 0 (voice) | WRR | 30% | 270 Mbps |
| Queue 1 (video) | WRR | 40% | 360 Mbps |
| Queue 2 (data) | WRR | 20% | 180 Mbps |
| Queue 3 (best-effort) | WRR | 10% | 90 Mbps |

$$BW_{available} = 1000 - 100 = 900 \text{ Mbps (after strict-high)}$$
$$BW_{queue0} = 900 \times 0.30 = 270 \text{ Mbps}$$

### RED (Random Early Detection)

Drop probability increases linearly between min and max threshold:

$$P_{drop}(q) = \begin{cases}
0 & q < q_{min} \\
\frac{q - q_{min}}{q_{max} - q_{min}} \times P_{max} & q_{min} \leq q \leq q_{max} \\
1 & q > q_{max}
\end{cases}$$

---

## 7. Dual RE Redundancy (GRES/NSR)

### The Problem

High-end Junos platforms support dual Routing Engines for redundancy. The failover math determines downtime.

### GRES (Graceful RE Switchover)

$$T_{failover}^{GRES} = T_{detect} + T_{switchover}$$

$$T_{detect} \approx 1\text{s}, \quad T_{switchover} \approx 1\text{-}5\text{s}$$

### NSR (Non-Stop Routing)

NSR synchronizes protocol state to the backup RE:

$$S_{synced} = S_{routes} + S_{neighbors} + S_{timers}$$

$$T_{sync} = \frac{S_{synced}}{BW_{internal}}$$

With NSR, routing protocol adjacencies survive RE failover:

$$T_{traffic\_loss}^{NSR} = T_{detect} + T_{PFE\_reprogram} \approx 2\text{-}5\text{s}$$

Without NSR:

$$T_{traffic\_loss}^{no\_NSR} = T_{detect} + T_{re\_establish\_adj} + T_{SPF} \approx 30\text{-}120\text{s}$$

### Availability

$$A_{single\_RE} = 0.9999 \quad \text{(99.99\% — 52 min/year downtime)}$$

$$A_{dual\_RE} = 1 - (1 - A)^2 = 1 - (0.0001)^2 = 0.99999999$$

---

## 8. Summary of Functions by Type

| Formula | Math Type | Domain |
|:---|:---|:---|
| $C_{candidate} \rightarrow C_{active}$ (atomic) | Transaction | Commit model |
| $\arg\min(\text{pref}, \text{metric})$ | Lexicographic ordering | Route selection |
| $BW / (S_{min} \times 8)$ | Division | PPS capacity |
| $\bigwedge c_i$ (first match) | Boolean logic | Filter evaluation |
| $O((V+E) \log V)$ | Dijkstra complexity | SPF calculation |
| $W_i / \Sigma W_j$ | Weighted proportion | CoS scheduling |
| $1 - (1-A)^2$ | Probability | Dual RE availability |

---

*Junos enforces discipline through its commit model — you cannot apply an invalid configuration. The RE/PFE separation means a routing protocol crash never takes down forwarding, and the ASIC-based PFE processes packets at line rate with deterministic latency. This is the math running on every Juniper router in the world.*
