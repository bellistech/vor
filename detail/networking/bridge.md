# The Mathematics of Linux Bridges — STP Convergence & MAC Learning Dynamics

> *A Linux bridge is a software implementation of an Ethernet switch whose correctness depends on spanning tree algorithms preventing loops, MAC address learning following LRU cache dynamics, and VLAN filtering creating partition functions over the forwarding domain.*

---

## 1. Spanning Tree Convergence (Graph Theory)

### The Problem

STP prevents loops in a bridged network by computing a spanning tree. Given a network of $B$ bridges and $L$ links, how long does STP take to converge, and how many links are blocked?

### The Formula

A connected graph with $V$ vertices (bridges + segments) and $E$ edges (links) has a spanning tree with exactly $V - 1$ edges. The number of blocked links:

$$L_{\text{blocked}} = E - (V - 1)$$

STP convergence time (classic 802.1D):

$$T_{\text{converge}} = 2 \times \text{forward\_delay} + \text{max\_age}$$

With defaults ($\text{forward\_delay} = 15$s, $\text{max\_age} = 20$s):

$$T_{\text{converge}} = 2 \times 15 + 20 = 50 \text{ seconds}$$

RSTP (802.1w) converges in approximately:

$$T_{\text{RSTP}} \approx 3 \times \text{hello\_time} = 6 \text{ seconds}$$

### Worked Examples

**Example:** Network with 5 bridges forming a ring plus one diagonal link. $V = 5$, $E = 6$:

$$L_{\text{blocked}} = 6 - (5 - 1) = 2$$

Two links will be in blocking state. Classic STP convergence: 50 seconds of downtime during topology change. RSTP: ~6 seconds.

For a full mesh of $B = 4$ bridges ($E = \binom{4}{2} = 6$):

$$L_{\text{blocked}} = 6 - 3 = 3$$

Half the links are blocked — significant redundancy wasted. This is why modern networks prefer routing (L3) over bridging (L2) for large topologies.

---

## 2. MAC Address Table Dynamics (LRU Cache Analysis)

### The Problem

The bridge FDB learns MAC addresses as frames arrive. With aging time $A$ and $N$ hosts sending frames at rate $\lambda$ each, what is the steady-state FDB size and miss rate?

### The Formula

A MAC entry stays in the FDB for duration $A$ after its last frame. The probability a host's entry is present (hit rate) depends on frame rate:

$$P(\text{hit}) = 1 - e^{-\lambda \cdot A}$$

Where $\lambda$ is the frame rate per host (frames/sec).

Expected FDB size at steady state:

$$E[\text{FDB}] = N \cdot P(\text{hit}) = N \cdot (1 - e^{-\lambda A})$$

Miss rate (frames that trigger flooding):

$$R_{\text{miss}} = N \cdot \lambda \cdot e^{-\lambda A}$$

### Worked Examples

**Example 1:** 200 hosts, aging time $A = 300$ sec, each host sends 1 frame/sec:

$$P(\text{hit}) = 1 - e^{-300} \approx 1.0$$

Every host is in the FDB. Miss rate effectively zero.

**Example 2:** 1000 IoT devices, $A = 300$ sec, each sends 1 frame every 10 minutes ($\lambda = 0.00167$/sec):

$$P(\text{hit}) = 1 - e^{-0.00167 \times 300} = 1 - e^{-0.5} = 1 - 0.607 = 0.393$$

Only 39.3% of devices are in the FDB. Expected FDB size: 393 entries.

Miss rate: $1000 \times 0.00167 \times 0.607 = 1.01$ floods/sec. Each miss causes broadcast flooding — significant bandwidth waste with 1000 devices.

Solution: Increase aging time to 3600 sec or add static FDB entries for critical devices.

---

## 3. Broadcast Storm Impact (Exponential Growth)

### The Problem

Without STP, a broadcast frame in a looped topology is replicated infinitely. If a bridge has $P$ ports and the loop has $H$ hops, how fast does the storm grow?

### The Formula

At each bridge with $P$ ports, a broadcast is forwarded to $P - 1$ ports. In a simple loop of $H$ bridges:

After $n$ traversals of the loop, the number of copies:

$$C(n) = (P - 1)^n$$

With each copy generating $(P-1)$ more copies per hop, and frames traversing the loop in time $t_{\text{loop}}$:

$$C(t) = (P-1)^{t / t_{\text{loop}}}$$

Bandwidth consumed at time $t$:

$$B(t) = F \cdot C(t) = F \cdot (P-1)^{t/t_{\text{loop}}}$$

Where $F$ is the frame size in bits.

### Worked Examples

**Example:** Two bridges, each with 4 ports, loop latency $t_{\text{loop}} = 0.1$ ms, frame size $F = 1518$ bytes:

$$C(t) = 3^{t/0.0001}$$

After 1 ms ($t/t_{\text{loop}} = 10$):

$$C = 3^{10} = 59049 \text{ copies}$$

Bandwidth: $59049 \times 1518 \times 8 = 717$ Mbit — saturating a gigabit link in 1 ms.

After 2 ms:

$$C = 3^{20} = 3.49 \times 10^9 \text{ copies}$$

This is why STP is not optional and why loop detection is critical in any bridged network.

---

## 4. VLAN Forwarding Domain Partitioning (Set Theory)

### The Problem

VLAN filtering partitions the bridge's forwarding domain. With $V$ VLANs across $P$ ports, how many distinct forwarding domains exist, and what is the broadcast domain size?

### The Formula

Each VLAN $v$ creates a forwarding domain $D_v$ consisting of ports that are members of that VLAN:

$$D_v = \{p \in P : v \in \text{VLANs}(p)\}$$

The broadcast domain size for VLAN $v$:

$$|D_v| = |\{p : v \in \text{VLANs}(p)\}|$$

Total broadcast traffic reduction compared to no VLANs:

$$\text{Reduction} = 1 - \frac{\sum_{v} |D_v|^2}{\left(\sum_v |D_v|\right)^2}$$

For equal-sized VLANs with $n$ ports each across $V$ VLANs:

$$\text{Reduction} = 1 - \frac{V \cdot n^2}{(V \cdot n)^2} = 1 - \frac{1}{V \cdot n}$$

### Worked Examples

**Example:** 48-port switch, 4 VLANs with 12 ports each:

$$\text{Broadcast per VLAN} = 12 \text{ ports (vs. 48 without VLANs)}$$

$$\text{Reduction} = 1 - \frac{4 \times 144}{(48)^2} = 1 - \frac{576}{2304} = 1 - 0.25 = 75\%$$

75% less broadcast traffic compared to a flat bridge. With 8 VLANs of 6 ports:

$$\text{Reduction} = 1 - \frac{8 \times 36}{2304} = 1 - 0.125 = 87.5\%$$

More VLANs = less broadcast traffic, but more management overhead.

---

## 5. Bridge Port State Machine (Finite Automata)

### The Problem

Each bridge port transitions through STP states: Disabled -> Blocking -> Listening -> Learning -> Forwarding. What is the minimum time from link-up to forwarding?

### The Formula

The STP port state machine transitions:

$$\text{Blocking} \xrightarrow{\text{max\_age}} \text{Listening} \xrightarrow{\text{forward\_delay}} \text{Learning} \xrightarrow{\text{forward\_delay}} \text{Forwarding}$$

Minimum time to forwarding (port already designated):

$$T_{\text{min}} = 2 \times \text{forward\_delay}$$

Time to forwarding after topology change (includes max_age):

$$T_{\text{topo}} = \text{max\_age} + 2 \times \text{forward\_delay}$$

With RSTP proposal/agreement:

$$T_{\text{RSTP}} \approx 1\text{--}2 \text{ round trips} \approx 2 \times \text{RTT}$$

### Worked Examples

**Example:** STP defaults (forward_delay=15s, max_age=20s):

$$T_{\text{min}} = 2 \times 15 = 30 \text{ sec}$$

$$T_{\text{topo}} = 20 + 30 = 50 \text{ sec}$$

With aggressive timers (forward_delay=4s, max_age=6s):

$$T_{\text{min}} = 8 \text{ sec}$$

$$T_{\text{topo}} = 14 \text{ sec}$$

Minimum forward_delay is 2 seconds (IEEE 802.1D constraint). RSTP achieves convergence in under 1 second for point-to-point links — three orders of magnitude faster.

---

## 6. Veth Pair Throughput in Bridged Containers (Queueing)

### The Problem

Docker containers connect to bridges via veth pairs. Each veth pair adds processing overhead. With $N$ containers on a bridge, each sending at rate $\lambda$, what is the aggregate throughput limit?

### The Formula

Each packet through a veth pair costs $C_v$ CPU cycles. The bridge forwarding costs $C_b$ cycles per packet (FDB lookup + forwarding decision). Total CPU cost per packet:

$$C_{\text{total}} = C_v + C_b + C_v = 2C_v + C_b$$

Maximum aggregate throughput with CPU capacity $F$ cycles/sec:

$$\lambda_{\max} = \frac{F}{C_{\text{total}}} = \frac{F}{2C_v + C_b}$$

Per-container fair share:

$$\lambda_{\text{per}} = \frac{\lambda_{\max}}{N}$$

### Worked Examples

**Example:** $C_v = 2000$ cycles, $C_b = 1500$ cycles, CPU at $F = 3 \times 10^9$ cycles/sec (single core):

$$\lambda_{\max} = \frac{3 \times 10^9}{2 \times 2000 + 1500} = \frac{3 \times 10^9}{5500} = 545454 \text{ packets/sec}$$

At 1500-byte packets: $545454 \times 1500 \times 8 = 6.55$ Gbps on a single core.

With 50 containers:

$$\lambda_{\text{per}} = \frac{545454}{50} = 10909 \text{ pps} = 131 \text{ Mbps}$$

Using multiple CPU cores (RSS/RPS) scales linearly. With 4 cores: 2.18M pps aggregate, 524 Mbps per container.

---

## Prerequisites

- Graph theory (spanning trees, connected components, cycles)
- Probability (exponential distribution, Poisson processes)
- Set theory (partitions, domain decomposition)
- Finite automata (state machines, transitions)
- Cache theory (LRU, hit rates, aging)
- Queueing theory (arrival rates, service capacity)
