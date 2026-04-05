# EIGRP — DUAL Algorithm and Advanced Distance Vector Routing

> *EIGRP is a hybrid routing protocol that marries distance vector simplicity with link-state convergence speed. At its core, the Diffusing Update Algorithm (DUAL) provides mathematically guaranteed loop-free routing through a decentralized computation that runs across every router in the autonomous system. The math spans graph theory (feasibility condition), queuing theory (convergence timing), and combinatorics (query scoping).*

---

## 1. DUAL Algorithm — The Theoretical Foundation

### The Problem DUAL Solves

Classical distance vector protocols (RIP, IGRP) suffer from counting-to-infinity, routing loops, and slow convergence. Bellman-Ford computes shortest paths but offers no loop-freedom guarantee during convergence. DUAL adds a diffusing computation that ensures no router ever installs a loop-causing route, even mid-convergence.

### Formal Definitions

Let $G = (V, E)$ be a directed graph representing the network, where $V$ is the set of routers and $E$ is the set of links.

For a destination $d$ and router $i$:

- **Distance** $D^i(d)$: The computed cost from $i$ to $d$ through the current successor
- **Reported distance** $RD^j(d)$: The cost from neighbor $j$ to $d$ as advertised by $j$
- **Feasible distance** $FD^i(d)$: The minimum distance ever achieved by $i$ to $d$ (monotonically decreasing unless reset)
- **Successor** $S^i(d)$: The neighbor $j$ that provides the least-cost path: $S^i(d) = \arg\min_j \{ c(i,j) + RD^j(d) \}$

### The Feasibility Condition

A neighbor $j$ qualifies as a feasible successor for destination $d$ at router $i$ if:

$$RD^j(d) < FD^i(d)$$

This is the invariant that guarantees loop freedom. The intuition: if neighbor $j$ reports a distance to $d$ that is strictly less than the best distance $i$ has ever known, then $j$ cannot possibly be routing through $i$ to reach $d$. Therefore, using $j$ as a next-hop cannot create a loop.

### Proof of Loop Freedom

**Theorem:** If every router in the network applies the feasibility condition before installing a route, no routing loop can exist.

**Proof sketch (by contradiction):**

1. Assume a loop exists: $r_1 \to r_2 \to \ldots \to r_k \to r_1$ for destination $d$.
2. Each router $r_i$ in the loop chose $r_{i+1}$ as next-hop, meaning the feasibility condition held:
   $RD^{r_{i+1}}(d) < FD^{r_i}(d)$
3. Following the chain: $RD^{r_2}(d) < FD^{r_1}(d)$, $RD^{r_3}(d) < FD^{r_2}(d)$, ..., $RD^{r_1}(d) < FD^{r_k}(d)$
4. Since $FD^{r_i}(d) \leq D^{r_i}(d) = c(r_i, r_{i+1}) + RD^{r_{i+1}}(d)$, we get $RD^{r_{i+1}}(d) < c(r_i, r_{i+1}) + RD^{r_{i+1}}(d)$, which implies $0 < c(r_i, r_{i+1})$ (true for positive link costs).
5. But traversing the full loop: the sum of reported distances strictly decreases at each hop, yet the loop returns to $r_1$, requiring $RD^{r_1}(d) < RD^{r_1}(d)$ — a contradiction.

Therefore, no loop can form under the feasibility condition. QED.

### Successor and Feasible Successor Selection

Given neighbors $N(i) = \{j_1, j_2, \ldots, j_n\}$ of router $i$:

1. **Compute cost through each neighbor:** $D_j(d) = c(i,j) + RD^j(d)$ for all $j \in N(i)$
2. **Select successor:** $S(d) = \arg\min_j D_j(d)$
3. **Update feasible distance:** $FD^i(d) = \min(FD^i(d), D_{S(d)}(d))$
4. **Identify feasible successors:** $FS(d) = \{ j \in N(i) : RD^j(d) < FD^i(d) \wedge j \neq S(d) \}$

### Worked Example — Successor Selection

Consider router A with three neighbors B, C, D for destination 10.1.0.0/24:

| Neighbor | Link Cost $c(A,j)$ | Reported Distance $RD^j$ | Total $D_j$ |
|:---:|:---:|:---:|:---:|
| B | 100 | 200 | 300 |
| C | 150 | 100 | 250 |
| D | 50 | 400 | 450 |

**Step 1:** Successor = C (lowest total: 250)
**Step 2:** $FD^A = 250$
**Step 3:** Check feasible successors:
- B: $RD^B = 200 < FD^A = 250$? Yes. B is a feasible successor.
- D: $RD^D = 400 < FD^A = 250$? No. D is NOT a feasible successor.

If the link to C fails, router A immediately switches to B without entering active state.
If both C and B fail, router A must go active and query neighbors for 10.1.0.0/24.

---

## 2. Active vs Passive State — The Diffusing Computation

### State Machine

EIGRP routes exist in one of two states:

| State | Meaning | Action |
|:---|:---|:---|
| Passive | Route is stable, successor is known | Normal forwarding, no computation |
| Active | Successor lost, no feasible successor | Queries sent, awaiting replies |

### The Diffusing Computation Process

When router $i$ loses its successor to destination $d$ and no feasible successor exists:

1. Router $i$ sets the route to **active** state
2. Router $i$ sends a **Query** to all neighbors (except the old successor if it went down)
3. Each neighbor $j$ that receives the query either:
   - **Has a feasible successor:** Sends a Reply immediately (stays passive)
   - **Has no feasible successor:** Goes active itself, forwards queries to its own neighbors
4. Router $i$ waits for Replies from **all** queried neighbors
5. Once all Replies received, router $i$ selects new successor and returns to **passive**

### Query Scoping and Propagation

The total number of queries generated in the worst case for a single destination:

$$Q_{total} = \sum_{i=1}^{|V|} |N(i)| = 2|E|$$

In a fully meshed network of $n$ routers:

$$Q_{worst} = n(n-1)$$

This is why query scoping through summarization and stub routing is critical.

### Stuck-in-Active (SIA)

A route remains active until all queried neighbors reply. The SIA timer defaults to 3 minutes (180 seconds). In modern implementations, a half-SIA mechanism sends a SIA-Query at 90 seconds:

| Timer | Action |
|:---|:---|
| $t = 0$ | Query sent, route goes active |
| $t = 90s$ | SIA-Query sent to non-responding neighbors |
| $t = 90s + \epsilon$ | If SIA-Reply received, timer resets to 90s |
| $t = 180s$ | No reply: neighbor adjacency torn down |

**SIA probability** increases with:
- Network diameter (hop count)
- Number of alternative paths (more neighbors = more queries to await)
- CPU load on intermediate routers
- Unidirectional links (query arrives, reply cannot return)

---

## 3. Composite Metric — The Mathematics

### Classic Metric Formula

The EIGRP classic composite metric is:

$$M = 256 \times \left[ K_1 \times BW + \frac{K_2 \times BW}{256 - Load} + K_3 \times Delay \right] \times \frac{K_5}{Reliability + K_4}$$

When $K_5 = 0$ (default), the reliability/load term vanishes:

$$M = 256 \times \left[ K_1 \times BW + K_3 \times Delay \right]$$

Where with default K-values ($K_1=1, K_2=0, K_3=1, K_4=0, K_5=0$):

$$M = 256 \times (BW + Delay)$$

### Component Calculations

**Bandwidth component:**

$$BW = \frac{10^7}{\min(\text{bandwidth in kbps along path})}$$

**Delay component:**

$$Delay = \sum_{i=1}^{hops} \frac{\text{delay}_i}{10} \quad \text{(in tens of microseconds)}$$

### Worked Example — Classic Metric

Path from A to D through two links:

| Link | Bandwidth | Delay |
|:---|:---:|:---:|
| A-B | 1 Gbps (1,000,000 kbps) | 10 usec |
| B-D | 100 Mbps (100,000 kbps) | 100 usec |

**Bandwidth component:** $BW = 10^7 / \min(1000000, 100000) = 10^7 / 100000 = 100$

**Delay component:** $Delay = (10 + 100) / 10 = 11$ (tens of microseconds)

**Metric:** $M = 256 \times (100 + 11) = 256 \times 111 = 28416$

### Comparison of Path Metrics

| Path | Min BW (kbps) | Total Delay (usec) | BW Component | Delay Component | Metric |
|:---|:---:|:---:|:---:|:---:|:---:|
| T1 serial | 1,544 | 20,000 | 6,476 | 2,000 | 2,169,856 |
| FastEthernet | 100,000 | 100 | 100 | 10 | 28,160 |
| GigabitEthernet | 1,000,000 | 10 | 10 | 1 | 2,816 |
| 10GigE | 10,000,000 | 10 | 1 | 1 | 512 |

### The 32-Bit Overflow Problem

Classic metric is stored in a 32-bit unsigned integer. Maximum value:

$$M_{max} = 2^{32} - 1 = 4,294,967,295$$

For a 10G Ethernet link: $BW = 10^7 / 10^7 = 1$, $Delay = 1$, $M = 256 \times 2 = 512$

The problem: 10G, 40G, and 100G interfaces all produce $BW = 1$ (since $10^7 / 10^7 = 1$, and anything faster still floors to 1). The metric cannot distinguish between them.

---

## 4. Wide Metrics — 64-Bit EIGRP Named Mode

### The Solution

Named mode EIGRP introduces 64-bit wide metrics with higher precision:

**Throughput (replaces bandwidth):**

$$Throughput = \frac{10^7 \times 65536}{\text{bandwidth in kbps}}$$

**Latency (replaces delay):**

$$Latency = \frac{\text{delay in picoseconds} \times 65536}{10^7}$$

The scaling factor of 65536 ($2^{16}$) provides granularity for high-speed interfaces.

### Wide Metric Comparison

| Interface | Classic BW | Wide Throughput | Distinguishable? |
|:---|:---:|:---:|:---:|
| 1 GigE | 10 | 655,360 | Yes |
| 10 GigE | 1 | 65,536 | Yes |
| 40 GigE | 1 | 16,384 | Yes (wide only) |
| 100 GigE | 1 | 6,553 | Yes (wide only) |

### RIB Scale

The 64-bit wide metric must be scaled down to fit the 32-bit RIB. The `rib-scale` factor (default 128) divides the wide metric:

$$M_{RIB} = \frac{M_{wide}}{rib\_scale}$$

If $M_{RIB}$ would be 0, it is set to 1 to remain reachable.

---

## 5. Unequal-Cost Load Balancing — Variance

### The Variance Multiplier

EIGRP is unique among IGPs in supporting unequal-cost load balancing. A route through feasible successor $j$ is eligible for load balancing if:

$$D_j(d) \leq V \times FD(d)$$

Where $V$ is the variance multiplier (default 1, meaning only equal-cost paths).

### Traffic Share Calculation

For $k$ paths with metrics $M_1 \leq M_2 \leq \ldots \leq M_k$, the traffic share for path $i$ is inversely proportional to metric:

$$\text{Share}_i = \frac{M_{max} / M_i}{\sum_{j=1}^{k} M_{max} / M_j}$$

Where $M_{max} = \max(M_1, \ldots, M_k)$.

### Worked Example — Variance 2

Router A has three paths to destination 10.1.0.0/24:

| Path | Metric | Feasible Successor? | $\leq V \times FD$? | Eligible? |
|:---|:---:|:---:|:---:|:---:|
| Via B | 1000 (successor) | N/A | N/A | Yes |
| Via C | 1500 | Yes ($RD^C < FD$) | $1500 \leq 2 \times 1000 = 2000$ | Yes |
| Via D | 2500 | Yes ($RD^D < FD$) | $2500 \leq 2 \times 1000 = 2000$ | No |
| Via E | 1800 | No ($RD^E \geq FD$) | N/A | No (not FS) |

Only paths via B and C are used. Traffic distribution:

$$\text{Share}_B = \frac{1500/1000}{1500/1000 + 1500/1500} = \frac{1.5}{1.5 + 1.0} = 60\%$$

$$\text{Share}_C = \frac{1500/1500}{1500/1000 + 1500/1500} = \frac{1.0}{2.5} = 40\%$$

CEF implements this through per-destination or per-packet hashing across the weighted paths.

---

## 6. Convergence Analysis

### Convergence Timeline

When a link fails, EIGRP convergence depends on whether a feasible successor exists:

**Case 1: Feasible successor available (local computation)**

$$T_{converge} = T_{detect} + T_{local}$$

Where:
- $T_{detect}$: Failure detection time (hold timer expiry or interface down notification)
- $T_{local}$: Time to swap to feasible successor ($\approx$ microseconds, effectively instant)

With BFD: $T_{detect} \approx 50\text{ms}$ (3 x 15ms intervals)
Without BFD: $T_{detect} \leq \text{hold timer} = 15\text{s}$ (default high-speed)

**Case 2: No feasible successor (diffusing computation)**

$$T_{converge} = T_{detect} + T_{query} + T_{reply} + T_{install}$$

Where:
- $T_{query}$: Time for queries to propagate to network boundary
- $T_{reply}$: Time for replies to propagate back
- $T_{install}$: Time to update RIB/FIB

In the worst case, query propagation time across a network of diameter $d$:

$$T_{query} \approx d \times (T_{process} + T_{transmit})$$

Where $T_{process} \approx 1\text{-}10\text{ms}$ per hop and $T_{transmit}$ depends on link speed.

### Convergence Comparison

| Scenario | Time | Mechanism |
|:---|:---|:---|
| Feasible successor + BFD | ~50 ms | Local swap, BFD detection |
| Feasible successor, no BFD | ~5-15 s | Local swap, hold timer |
| Active query, small network (d=3) | ~100-500 ms | Diffusing computation |
| Active query, large network (d=10) | ~1-5 s | Diffusing computation |
| SIA (worst case) | 180 s | Neighbor killed |

---

## 7. Stub Routing — Query Domain Reduction

### The Scaling Problem

Without stub routing, a query for any destination propagates to every router in the AS. In a hub-and-spoke topology with $h$ hubs and $s$ spokes per hub:

$$Q_{without\_stub} = h \times s \times (\text{average neighbor count})$$

With stub routing on spokes, queries never propagate beyond the hub:

$$Q_{with\_stub} = h \times (\text{hub neighbor count})$$

### Query Scope Reduction

| Topology | Routers | Without Stub | With Stub | Reduction |
|:---|:---:|:---:|:---:|:---:|
| 5 hubs, 20 spokes each | 105 | ~2,100 queries | ~25 queries | 98.8% |
| 10 hubs, 50 spokes each | 510 | ~25,500 queries | ~100 queries | 99.6% |
| 20 hubs, 100 spokes each | 2,020 | ~202,000 queries | ~400 queries | 99.8% |

### Stub Mode Behavior Matrix

| Mode | Sends Routes | Responds to Queries | Goes Active |
|:---|:---|:---|:---|
| `receive-only` | None | Reply with infinite metric | Never |
| `connected` | Connected | Reply for non-connected | Never |
| `summary` | Summary | Reply for non-summary | Never |
| `static` | Static | Reply for non-static | Never |
| `redistributed` | Redistributed | Reply for non-redistributed | Never |
| `connected summary` (default) | Connected + Summary | Reply for others | Never |

Hub routers never send queries to stub neighbors, which is the primary mechanism that limits query scope.

---

## 8. Packet Types and Reliable Transport

### RTP (Reliable Transport Protocol)

EIGRP uses its own transport protocol (RTP) over IP protocol 88. Not to be confused with Real-time Transport Protocol.

| Packet | Multicast/Unicast | Reliable | Sequence Number |
|:---|:---|:---|:---|
| Hello | Multicast 224.0.0.10 | No | None |
| Update | Both | Yes | Tracked |
| Query | Both | Yes | Tracked |
| Reply | Unicast | Yes | Tracked |
| ACK | Unicast | No | Acknowledges |

### Reliable Delivery Mechanism

For each reliable packet:

1. Sender assigns sequence number $seq_n$
2. Receiver must respond with ACK containing $seq_n$
3. If no ACK within retransmit timeout (RTO), packet is retransmitted unicast
4. After 16 retransmissions with no ACK, neighbor is declared dead

**RTO calculation** uses a smoothed round-trip time (SRTT) similar to TCP:

$$SRTT_{new} = (1 - \alpha) \times SRTT_{old} + \alpha \times RTT_{sample}$$

$$RTO = \mu \times SRTT$$

Where $\alpha = 1/8$ and $\mu = 4$ (conservative multiplier).

### Conditional Receive Mode

On multi-access segments with many neighbors, EIGRP uses conditional receive (CR) to manage multicast flow:

1. Sender sets CR flag in multicast packet
2. Specifies a list of lagging peers (those that haven't ACKed previous packets)
3. Non-lagging peers process the packet; lagging peers ignore it
4. Lagging peers receive unicast retransmissions

This prevents one slow neighbor from blocking updates to the entire segment.

---

## 9. Authentication — MD5 and SHA-256

### Key Chain Rotation

EIGRP authentication supports key rotation via send and accept lifetimes:

| Time | Key 1 | Key 2 | Active Key |
|:---|:---:|:---:|:---|
| $t < t_1$ | Send+Accept | Not yet valid | Key 1 only |
| $t_1 \leq t < t_2$ | Send+Accept | Accept only | Key 1 sends, both accepted |
| $t_2 \leq t < t_3$ | Accept only | Send+Accept | Key 2 sends, both accepted |
| $t \geq t_3$ | Expired | Send+Accept | Key 2 only |

This allows hitless key rotation without neighbor adjacency disruption.

### MD5 vs SHA-256

| Property | MD5 | SHA-256 |
|:---|:---|:---|
| Hash length | 128-bit | 256-bit |
| Collision resistance | Broken (2^18 complexity) | Secure ($2^{128}$) |
| Configuration | Classic + Named mode | Named mode only |
| Key chain support | Yes | No (inline password) |
| HMAC | Yes (HMAC-MD5) | Yes (HMAC-SHA-256) |
| Performance impact | Negligible | Negligible |

SHA-256 is recommended for all new deployments. MD5 remains available for backward compatibility.

---

## 10. Route Summarization — Topology Table Reduction

### Summarization Benefits

Summarizing $n$ specific routes into one summary route:

- **Topology table reduction:** $n$ entries become 1
- **Query boundary:** Queries for specific routes stop at the summarizing router
- **Update reduction:** One update instead of $n$ during reconvergence

### Null0 Route and Loop Prevention

When a summary route is created, a Null0 (discard) route is automatically installed:

```
S     10.0.0.0/8 is directly connected, Null0
```

This prevents routing loops when traffic arrives for a specific prefix within the summary range that doesn't exist. Without the Null0 route, the router might forward the traffic based on a less specific route, creating a loop.

### Automatic Summarization (Legacy)

Classic EIGRP with `auto-summary` enabled summarizes routes at classful network boundaries:

| Network | Classful Boundary | Summary |
|:---|:---|:---|
| 10.1.1.0/24 | 10.0.0.0/8 (Class A) | 10.0.0.0/8 |
| 172.16.5.0/24 | 172.16.0.0/16 (Class B) | 172.16.0.0/16 |
| 192.168.1.0/24 | 192.168.1.0/24 (Class C) | 192.168.1.0/24 |

Auto-summary is disabled by default since IOS 15. It should remain disabled in all modern networks.

---

## 11. EIGRP for IPv6

### Differences from IPv4 EIGRP

| Feature | IPv4 EIGRP | IPv6 EIGRP |
|:---|:---|:---|
| Multicast address | 224.0.0.10 | ff02::a |
| Network statement | Required | Not used (interface activation) |
| Router-ID | From IPv4 address | Must be explicitly set (32-bit IPv4 format) |
| Next-hop | IPv4 address | Link-local IPv6 address |
| Neighbor identification | IPv4 address | Link-local address |
| Named mode | Optional | Preferred |

### Key Design Consideration

IPv6 EIGRP uses link-local addresses for all next-hop information. This means:

1. No network statements needed; EIGRP is activated per-interface
2. A router-ID must be manually configured (it is still a 32-bit value in IPv4 dotted notation even though the protocol runs IPv6)
3. Neighbors are identified by their link-local address, not global unicast address

---

## 12. Redistribution — Seed Metrics and Route Feedback

### The Seed Metric Requirement

When redistributing into EIGRP, a seed metric must be provided. Without it, the route has an infinite metric and is unreachable. The seed metric is specified as five components:

$$\text{seed} = (bandwidth,\ delay,\ reliability,\ load,\ MTU)$$

Example: `metric 10000 100 255 1 1500`

| Component | Value | Meaning |
|:---|:---:|:---|
| Bandwidth | 10000 | 10,000 kbps (10 Mbps equivalent) |
| Delay | 100 | 1000 microseconds (100 tens-of-usec) |
| Reliability | 255 | Maximum reliability (255/255) |
| Load | 1 | Minimum load (1/255) |
| MTU | 1500 | Standard Ethernet MTU |

### Mutual Redistribution and Routing Loops

When redistributing between EIGRP and another protocol (e.g., OSPF) in both directions, routing loops can form. Prevention requires:

1. **Route tags:** Tag routes at redistribution point, filter them at the reverse point
2. **Administrative distance tuning:** Set external EIGRP AD (170) vs OSPF external (110)
3. **Distribute lists / route maps:** Explicit filtering at redistribution boundaries
4. **Prefix lists:** Only permit known prefixes during redistribution

### Administrative Distance

| Route Type | Default AD |
|:---|:---:|
| EIGRP summary route | 5 |
| EIGRP internal | 90 |
| EIGRP external | 170 |
| OSPF | 110 |
| RIP | 120 |
| Static | 1 |

The large gap between internal (90) and external (170) is designed to prefer native EIGRP routes over redistributed ones, reducing loop risk.

---

## 13. Scalability Considerations

### Topology Table Memory

Each EIGRP topology entry consumes approximately 200-500 bytes depending on the number of paths. For $n$ destinations with $p$ paths each:

$$Memory_{topo} \approx n \times p \times 350\ \text{bytes}$$

| Destinations | Paths/Dest | Memory |
|:---:|:---:|:---:|
| 1,000 | 2 | ~700 KB |
| 10,000 | 3 | ~10.5 MB |
| 100,000 | 4 | ~140 MB |

### Bandwidth Consumption

EIGRP hello packets on a link:

$$BW_{hello} = \frac{\text{packet size} \times 8}{\text{hello interval}} \approx \frac{60 \times 8}{5} = 96\ \text{bps}$$

Negligible on modern links, but update storms during convergence can spike:

$$BW_{update\_storm} \approx n_{routes} \times 80\ \text{bytes} \times 8 / T_{convergence}$$

For 10,000 routes converging in 1 second: $\approx 6.4$ Mbps burst.

### Maximum Network Size Guidelines

| Factor | Recommended Limit | Constraint |
|:---|:---:|:---|
| Routes per AS | ~50,000 | Memory, convergence time |
| Neighbors per router | ~50 | CPU for hello processing |
| Network diameter | ~15 hops | Query propagation time |
| AS size (routers) | ~500-1,000 | SIA risk, query scope |

---

## 14. Named Mode vs Classic Mode — Feature Matrix

| Feature | Classic Mode | Named Mode |
|:---|:---:|:---:|
| Metric width | 32-bit | 64-bit (wide) |
| Authentication | MD5 only | MD5 + SHA-256 |
| IPv4 and IPv6 | Separate processes | Unified under address-families |
| Per-AF interface config | Global per-interface | Hierarchical af-interface |
| Add-path support | No | Yes |
| Route tag width | 8-bit | 16-bit |
| Next-hop self | Manual | Configurable per af-interface |
| Remote neighbor support | No | Yes (LISP integration) |

Named mode is the recommended configuration model for all new EIGRP deployments.

---

## 15. Troubleshooting Decision Tree

### Neighbor Adjacency Failure

```
Neighbor not forming
|
+-- Interface up/up?
|   +-- No: Fix layer 1/2
|   +-- Yes: Continue
|
+-- Correct IP subnet?
|   +-- No: Fix addressing
|   +-- Yes: Continue
|
+-- AS number match?
|   +-- No: Fix AS config
|   +-- Yes: Continue
|
+-- K-values match?
|   +-- No: Align K-values (all routers in AS)
|   +-- Yes: Continue
|
+-- Authentication match?
|   +-- No: Fix key chain / password
|   +-- Yes: Continue
|
+-- ACL/firewall blocking protocol 88?
|   +-- Yes: Permit IP protocol 88 to/from 224.0.0.10
|   +-- No: Check for passive-interface, MTU issues
```

### Suboptimal Path Selection

```
Traffic taking wrong path
|
+-- Check metrics: show ip eigrp topology <prefix>
|   +-- Bandwidth set correctly on all interfaces?
|   +-- Delay set correctly?
|   +-- Offset-list applied?
|
+-- Check variance if unequal-cost expected
|   +-- Is the alternate path a feasible successor?
|   +-- Is its metric within variance * FD?
|
+-- Check distribute-list or route-map filtering
|
+-- Check for summarization hiding more specific routes
```

---

## 16. Summary of Formulas

| Formula | Math Type | Application |
|:---|:---|:---|
| $RD^j(d) < FD^i(d)$ | Inequality | Feasibility condition |
| $256 \times (BW + Delay)$ | Weighted sum | Classic composite metric |
| $10^7 / \min(BW)$ | Division / minimum | Bandwidth component |
| $\sum \text{delay}_i / 10$ | Summation | Delay component |
| $D_j \leq V \times FD$ | Inequality | Variance eligibility |
| $M_{max}/M_i / \sum M_{max}/M_j$ | Proportion | Traffic share calculation |
| $10^7 \times 65536 / BW$ | Scaled division | Wide metric throughput |
| $(1-\alpha) \times SRTT + \alpha \times RTT$ | EWMA | RTO estimation |
| $n \times p \times 350$ | Linear | Topology table memory |
| $2^{32} - 1$ | Overflow bound | Classic metric maximum |

## Prerequisites

- graph theory, distance vector algorithms, Bellman-Ford algorithm, subnetting, IP routing fundamentals

## Complexity

| Operation | Time | Space |
|-----------|------|-------|
| DUAL feasibility check | O(n) per destination | O(n) neighbors |
| Metric computation | O(h) per path | O(1) |
| Topology table lookup | O(log n) | O(n * p) |
| Diffusing computation (worst) | O(V + E) | O(V) |
| Query propagation | O(d) where d = diameter | O(V) |

---

*Every EIGRP router maintains a private view of the network through its topology table, selecting loop-free paths via the feasibility condition. DUAL's genius is that it proves loop freedom locally — no global synchronization required, no flooding database, just a simple inequality tested at each router that guarantees the entire distributed system converges without loops.*
