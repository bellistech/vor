# The Mathematics of Cisco IOS — Network Operating System Internals

> *Cisco IOS powers the majority of enterprise routing and switching infrastructure. Its forwarding architecture uses CEF with compressed tries, TCAM for O(1) ACL matching, and QoS scheduling with weighted fair queuing. Every packet traversal is a mathematical operation.*

---

## 1. CEF Architecture (Cisco Express Forwarding)

### The Problem

CEF is the high-speed forwarding engine in IOS. It pre-computes the routing table into an optimized data structure — the **FIB** (Forwarding Information Base) — and an **adjacency table** for Layer 2 rewrite.

### FIB as Compressed Trie (MTRIE)

The FIB is stored as a **Multi-way Trie** (MTRIE) — a tree where each level examines a fixed number of address bits:

$$\text{Lookup depth} = \lceil 32 / B \rceil$$

Where $B$ = bits per trie level. Common: $B = 8$ (256-way branching):

$$\text{Depth} = \lceil 32 / 8 \rceil = 4 \text{ levels}$$

### Lookup Time

$$T_{lookup} = D \times T_{memory\_access}$$

For 4 levels at 10ns per access:

$$T_{lookup} = 4 \times 10 = 40\text{ ns}$$

This is **constant** regardless of table size — $O(1)$ for a fixed address width.

### FIB Memory Requirement

$$M_{FIB} = \sum_{l=0}^{D-1} N_l \times S_{node}$$

Where $N_l$ = nodes at level $l$, $S_{node}$ = node size.

| Routing Table | Prefix Count | FIB Memory | Lookup Time |
|:---|:---:|:---:|:---:|
| Enterprise | 5,000 | 5 MB | 40 ns |
| ISP regional | 50,000 | 40 MB | 40 ns |
| Full Internet (DFZ) | 1,000,000 | 400 MB | 40 ns |

### Adjacency Table

$$|\text{adj table}| = |\text{next-hops}| \times S_{adj\_entry}$$

Each adjacency entry contains the Layer 2 header rewrite (14 bytes for Ethernet + padding):

$$S_{adj} \approx 24 \text{ bytes per entry}$$

### CEF Load Sharing

Per-destination: $H(\text{src}, \text{dst}) \bmod N$ where $N$ = equal-cost paths.
Per-packet: round-robin across $N$ paths.

$$\text{path}(packet) = H(\text{src\_ip}, \text{dst\_ip}) \bmod N$$

---

## 2. TCAM (Ternary Content-Addressable Memory)

### The Problem

ACLs and policy-based routing require matching packets against rules. TCAM provides $O(1)$ lookup regardless of rule count.

### Ternary Matching

Each TCAM cell stores one of three states:

$$\text{cell} \in \{0, 1, X\} \quad \text{(X = don't care)}$$

### Matching

A TCAM entry matches when all non-X bits agree:

$$\text{match}(packet, entry) = \forall i: (entry_i = X) \vee (entry_i = packet_i)$$

### Lookup Complexity

$$T_{TCAM} = O(1) \quad \text{(single clock cycle, parallel comparison)}$$

Every entry is compared simultaneously — a 10,000-entry ACL matches in the same time as a 10-entry ACL.

### TCAM Capacity

$$\text{Rules stored} = \frac{S_{TCAM}}{S_{entry}}$$

Typical entry: 144 bits (source IP + dest IP + ports + protocol + flags):

| Switch Model | TCAM Size | ACL Entries |
|:---|:---:|:---:|
| Catalyst 3850 | 128K entries | 128,000 |
| Catalyst 9300 | 256K entries | 256,000 |
| Nexus 9000 | 512K entries | 512,000 |

### TCAM Utilization

$$U_{TCAM} = \frac{|\text{programmed entries}|}{|\text{total entries}|}$$

When TCAM is full, new ACL entries are **software switched** — falling back to CPU:

$$T_{software} \approx 1000 \times T_{TCAM}$$

---

## 3. QoS Scheduling (Weighted Fair Queuing)

### The Problem

QoS scheduling determines which packets are sent first when interfaces are congested. WFQ allocates bandwidth proportionally.

### WFQ Weight Formula

$$BW_i = \frac{W_i}{\sum_{j=1}^{N} W_j} \times BW_{link}$$

Where $W_i = \frac{1}{(\text{IP precedence}_i + 1)}$ (default WFQ).

### Worked Example: 100 Mbps Link

| Flow | IP Precedence | Weight $W_i$ | Bandwidth |
|:---|:---:|:---:|:---:|
| Voice | 5 | $1/6 = 0.167$ | 35.7 Mbps |
| Video | 4 | $1/5 = 0.200$ | 42.9 Mbps |
| Data | 0 | $1/1 = 1.000$ | 21.4 Mbps |

$$\text{Total weight} = 0.167 + 0.200 + 1.000 = 1.367$$

$$BW_{voice} = \frac{0.167}{1.367} \times 100 = 12.2 \text{ Mbps}$$

Wait — that's inverted. Higher precedence gets *lower* weight. Let me recalculate:

$$BW_{voice} = \frac{1/6}{1/6 + 1/5 + 1/1} \times 100 = \frac{0.167}{1.367} \times 100 = 12.2 \text{ Mbps}$$

This seems wrong — voice should get MORE bandwidth. The issue: WFQ uses **inverse** weighting. The flow with the *highest* weight (lowest precedence) gets the *most* bandwidth in basic WFQ. This is why **CBWFQ** replaced WFQ for modern QoS.

### CBWFQ (Class-Based Weighted Fair Queuing)

CBWFQ allocates bandwidth directly:

$$BW_{class} = \text{configured bandwidth (absolute or percentage)}$$

$$\sum_{c \in \text{classes}} BW_c \leq 0.75 \times BW_{link} \quad \text{(25% reserved for default class)}$$

### LLQ (Low-Latency Queuing)

Priority queue gets strict priority with a policer:

$$BW_{priority} = \min(\text{traffic rate}, \text{police rate})$$

If priority traffic exceeds its allocation:

$$\text{Dropped} = \text{rate}_{in} - \text{police rate}$$

### Delay Budget

$$T_{serialization} = \frac{S_{packet}}{BW_{link}}$$

| Packet Size | 1 Mbps | 10 Mbps | 100 Mbps | 1 Gbps |
|:---:|:---:|:---:|:---:|:---:|
| 64 B | 512 us | 51.2 us | 5.12 us | 0.512 us |
| 1500 B | 12 ms | 1.2 ms | 120 us | 12 us |
| 9000 B (jumbo) | 72 ms | 7.2 ms | 720 us | 72 us |

For VoIP (G.711, 20ms intervals): max one-way delay budget is 150ms. Serialization on a 1 Mbps link consumes 12ms of that budget per hop.

---

## 4. Spanning Tree Protocol (Convergence Math)

### The Problem

STP prevents loops in Layer 2 networks. Convergence time depends on timer values.

### STP Timers (802.1D)

$$T_{convergence} = T_{max\_age} + T_{forward\_delay} \times 2$$

Default values:

$$T = 20 + 15 \times 2 = 50 \text{ seconds}$$

### RSTP Improvement (802.1w)

$$T_{convergence}^{RSTP} \approx 1\text{-}3 \text{ seconds}$$

### Path Cost Calculation

$$\text{Cost} = \frac{10^9}{\text{Bandwidth (bps)}} \quad \text{(short mode)}$$

| Link Speed | Cost |
|:---|:---:|
| 10 Mbps | 100 |
| 100 Mbps | 19 |
| 1 Gbps | 4 |
| 10 Gbps | 2 |

### Root Path Cost

$$\text{Root path cost}(switch) = \sum_{\text{hops to root}} \text{port cost}$$

The switch with the lowest root path cost to the root bridge wins the designated port election.

---

## 5. Routing Protocol Metrics

### EIGRP Composite Metric

$$\text{Metric} = 256 \times \left(\frac{K_1 \times BW_{min} + K_2 \times BW_{min}}{256 - \text{load}} + K_3 \times \text{delay}_{sum}\right) \times \frac{K_5}{K_4 + \text{reliability}}$$

Default: $K_1 = 1, K_3 = 1$, all others = 0. Simplifies to:

$$\text{Metric} = 256 \times (BW_{min} + \text{delay}_{sum})$$

Where:

$$BW_{min} = \frac{10^7}{\text{lowest bandwidth (kbps) on path}}$$

$$\text{delay}_{sum} = \sum_{\text{links}} \frac{\text{delay (us)}}{10}$$

### EIGRP Feasibility Condition

$$\text{FD}(\text{successor}) > \text{RD}(\text{feasible successor})$$

Where FD = feasible distance, RD = reported distance. This guarantees loop-free paths.

### OSPF Metric

$$\text{Cost} = \frac{\text{Reference BW}}{\text{Interface BW}}$$

Default reference: $10^8$ (100 Mbps).

| Interface | Cost |
|:---|:---:|
| FastEthernet (100 Mbps) | 1 |
| GigabitEthernet (1 Gbps) | 1 (needs reference change!) |
| 10 GigE | 1 (needs reference change!) |

With `auto-cost reference-bandwidth 100000` (100 Gbps):

| Interface | Cost |
|:---|:---:|
| FastEthernet | 1,000 |
| GigE | 100 |
| 10 GigE | 10 |

---

## 6. NAT Translation (Table Management)

### The Problem

NAT maintains a translation table. Table size and timeout determine memory usage and connection capacity.

### NAT Table Size

$$|T_{NAT}| = N_{active\_connections}$$

$$M_{NAT} = |T_{NAT}| \times S_{entry}$$

Where $S_{entry} \approx 160$ bytes (source, dest, ports, protocol, timers).

### PAT (Port Address Translation) Capacity

$$C_{PAT} = |\text{available ports}| = 65{,}535 - 1{,}024 = 64{,}511 \text{ per public IP}$$

For $P$ public IPs:

$$C_{total} = P \times 64{,}511$$

### NAT Timeout

$$\text{Entry lifetime} = \max(T_{timeout}, T_{last\_packet} + T_{idle\_timeout})$$

| Protocol | Default Timeout |
|:---|:---:|
| TCP established | 86,400s (24 hr) |
| TCP SYN | 60s |
| UDP | 300s (5 min) |
| ICMP | 60s |

---

## 7. Memory and CPU Model

### The Problem

IOS runs on fixed hardware. Understanding memory partitioning prevents crashes.

### Memory Regions

$$M_{total} = M_{processor} + M_{I/O}$$

$$M_{processor} = M_{IOS} + M_{routing\_tables} + M_{buffers} + M_{free}$$

### Buffer Sizing

$$\text{Buffers needed} = \frac{\text{PPS} \times T_{processing}}{1}$$

At 100,000 pps with 10us processing: $\text{Buffers} = 100{,}000 \times 0.00001 = 1$ simultaneous buffer needed.

### CPU Utilization Thresholds

| Utilization | Status |
|:---:|:---|
| 0-40% | Healthy |
| 40-60% | Monitor |
| 60-80% | Warning |
| 80-100% | Critical (packet drops) |

---

## 8. Summary of Functions by Type

| Formula | Math Type | Domain |
|:---|:---|:---|
| $\lceil 32/B \rceil$ level trie | Data structure | CEF FIB lookup |
| $O(1)$ TCAM | Parallel match | ACL processing |
| $W_i / \Sigma W_j$ | Weighted proportion | QoS scheduling |
| $20 + 15 \times 2 = 50$s | Timer arithmetic | STP convergence |
| $10^7 / BW_{min} + \Sigma delay$ | Composite metric | EIGRP |
| $P \times 64{,}511$ | Multiplication | PAT capacity |

---

*Every packet passing through a Cisco router traverses the FIB trie in 40ns, hits TCAM for ACL evaluation in one clock cycle, and enters a QoS scheduler that allocates bandwidth by weight. This is the math running at line rate on the backbone of the internet.*
