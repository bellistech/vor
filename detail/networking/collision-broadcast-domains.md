# Collision & Broadcast Domains — CSMA/CD Analysis, Broadcast Scaling & Layer 2 Evolution

> *Collision domains and broadcast domains are the two fundamental partitioning boundaries in Ethernet networking. The first governed shared-media access through probabilistic contention (CSMA/CD); the second governs how far Layer 2 broadcast frames propagate. Understanding their mathematics — collision probability, exponential backoff convergence, broadcast traffic scaling, and STP loop prevention — explains why Ethernet evolved from shared coax to full-duplex switched fabrics with VLAN segmentation.*

---

## 1. CSMA/CD Algorithm — Collision Probability and Exponential Backoff

### The Problem

On shared-media Ethernet (hubs, coax), multiple stations contend for the same wire. CSMA/CD (Carrier Sense Multiple Access with Collision Detection) is the distributed protocol that manages this contention. How efficient is it, and what are the mathematical bounds?

### The Algorithm

```
1. CARRIER SENSE: Listen to the medium.
   - If idle for IFG (96 bit times), begin transmitting.
   - If busy, wait until idle + IFG, then transmit.

2. COLLISION DETECTION: While transmitting, monitor the wire.
   - If transmitted signal differs from received signal → collision detected.
   - Send JAM signal (48 bits) to ensure all stations detect the collision.
   - Abort transmission.

3. BACKOFF: Wait a random time before retrying.
   - After collision c (1 ≤ c ≤ 16):
     - Choose k uniformly from [0, 2^min(c,10) - 1]
     - Wait k × slot_time
   - Slot time: 51.2 μs (10 Mbps), 5.12 μs (100 Mbps), 4.096 μs (1 Gbps)
   - After 16 collisions: discard frame, report error to upper layer.
```

### Collision Probability Analysis

With $N$ stations each attempting to transmit in a given slot, the probability that exactly one station transmits (successful transmission):

$$P_{success}(N) = N \times p \times (1 - p)^{N-1}$$

Where $p$ is the probability that any given station attempts transmission. Maximizing $P_{success}$ over $p$:

$$\frac{dP_{success}}{dp} = 0 \implies p^* = \frac{1}{N}$$

$$P_{success}^* = N \times \frac{1}{N} \times \left(1 - \frac{1}{N}\right)^{N-1} = \left(1 - \frac{1}{N}\right)^{N-1}$$

As $N \to \infty$:

$$\lim_{N \to \infty} P_{success}^* = \frac{1}{e} \approx 0.368$$

This is the theoretical maximum channel utilization under contention — no more than 36.8% of slot times yield successful transmissions, regardless of the number of stations.

### Throughput Under Load

The maximum throughput efficiency of CSMA/CD:

$$\eta = \frac{1}{1 + \frac{e \times 2 \times \tau}{T_{frame}}}$$

Where:
- $\tau$ = propagation delay (end-to-end)
- $T_{frame}$ = frame transmission time
- $a = \tau / T_{frame}$ = normalized propagation delay

$$\eta = \frac{1}{1 + 5.44a}$$

| Link Speed | Frame Size | $a$ (2.5 km) | Max $\eta$ |
|:---:|:---:|:---:|:---:|
| 10 Mbps | 1,500 B | 0.021 | 89.8% |
| 10 Mbps | 64 B | 0.500 | 26.8% |
| 100 Mbps | 1,500 B | 0.208 | 46.8% |
| 100 Mbps | 64 B | 5.000 | 3.5% |

Small frames at high speeds on long cables suffer catastrophic efficiency loss. This is why Gigabit Ethernet extended the slot time to 4,096 bits (carrier extension) and why modern Ethernet uses full-duplex switching instead.

### Exponential Backoff Convergence

The Binary Exponential Backoff (BEB) algorithm:

After collision $c$: wait $k$ slot times, where $k \in [0, 2^{\min(c, 10)} - 1]$.

$$E[\text{backoff}] = \frac{2^{\min(c, 10)} - 1}{2} \times T_{slot}$$

| Collision # $c$ | Window Size | $E[\text{wait}]$ (slots) | $E[\text{wait}]$ at 10 Mbps |
|:---:|:---:|:---:|:---:|
| 1 | 2 | 0.5 | 25.6 $\mu$s |
| 2 | 4 | 1.5 | 76.8 $\mu$s |
| 3 | 8 | 3.5 | 179.2 $\mu$s |
| 5 | 32 | 15.5 | 793.6 $\mu$s |
| 10 | 1,024 | 511.5 | 26.2 ms |
| 11-16 | 1,024 | 511.5 | 26.2 ms (capped) |

The window cap at $2^{10} = 1024$ prevents unbounded delay. After 16 collisions, the frame is dropped. The probability of reaching 16 collisions under moderate load ($P_{collision} = 0.5$):

$$P_{abort} = 0.5^{16} = 1.53 \times 10^{-5}$$

Under heavy load ($P_{collision} = 0.7$):

$$P_{abort} = 0.7^{16} = 3.32 \times 10^{-3}$$

BEB is provably unstable under persistent heavy load: as backoff windows grow, new arrivals gain an advantage over stations with large backoff counters (the "capture effect"). This fairness problem was a key motivation for moving to switched, full-duplex Ethernet.

---

## 2. Broadcast Domain Scaling — Optimal Size Analysis

### The Problem

Every broadcast frame in a VLAN is delivered to and processed by every host. As the number of hosts grows, broadcast overhead consumes bandwidth and CPU cycles. What is the optimal broadcast domain size?

### Broadcast Traffic Model

Assume each host generates $R$ broadcast frames per second (ARP, DHCP, NetBIOS, gratuitous ARP, etc.). With $N$ hosts in a broadcast domain:

Total broadcast frames per second on the segment:

$$B_{total} = N \times R$$

Each host must process all broadcasts from every other host:

$$\text{CPU interrupts/host/sec} = (N - 1) \times R \approx N \times R$$

Broadcast bandwidth consumed:

$$BW_{bcast} = N \times R \times S_{avg} \times 8 \text{ bps}$$

Where $S_{avg}$ is the average broadcast frame size in bytes.

### Empirical Broadcast Rates

Measured broadcast rates per host by traffic type:

| Traffic Source | Rate (frames/sec) | Frame Size | Notes |
|:---|:---:|:---:|:---|
| ARP | 0.5-2 | 42 B | Cache refresh every 30-300s |
| DHCP | 0.001-0.01 | 300-548 B | Lease renewal interval |
| Gratuitous ARP | 0.01-0.1 | 42 B | IP changes, failover |
| NetBIOS | 0.5-5 | 78-250 B | Windows, if enabled |
| mDNS/LLMNR | 0.1-1 | 60-300 B | macOS/Linux/Windows |
| OSPF/VRRP | 0.5-1 | 64-100 B | If running on segment |

Conservative aggregate per host: ~3-5 broadcasts/second in a typical enterprise network.

### Scaling Table

Using $R = 5$ broadcasts/sec, $S_{avg} = 100$ bytes:

| Hosts $N$ | Bcast frames/sec | BW consumed | CPU interrupts/host/sec | % of 1 Gbps |
|:---:|:---:|:---:|:---:|:---:|
| 25 | 125 | 100 kbps | 120 | 0.01% |
| 50 | 250 | 200 kbps | 245 | 0.02% |
| 100 | 500 | 400 kbps | 495 | 0.04% |
| 250 | 1,250 | 1.0 Mbps | 1,245 | 0.10% |
| 500 | 2,500 | 2.0 Mbps | 2,495 | 0.20% |
| 1,000 | 5,000 | 4.0 Mbps | 4,995 | 0.40% |
| 5,000 | 25,000 | 20 Mbps | 24,995 | 2.00% |
| 10,000 | 50,000 | 40 Mbps | 49,995 | 4.00% |

### The Real Bottleneck

Bandwidth is rarely the limit. At 1 Gbps, even 10,000 hosts consume only 4% of link capacity with broadcasts. The real limits are:

1. **CPU processing**: Each broadcast frame triggers an interrupt. At 25,000 interrupts/sec, low-power devices (IoT, IP phones, printers) experience measurable CPU impact.

2. **ARP table size**: Linux defaults `gc_thresh3` = 1024 ARP entries. Beyond this, the kernel garbage-collects entries, causing re-ARPing and more broadcasts. Enterprise environments set this to 4096-16384.

3. **MAC address table overflow**: When a switch's MAC table fills (typically 8K-128K entries), unknown unicast flooding begins — all unicast to unknown MACs is flooded like broadcast.

4. **DHCP lease storms**: Monday morning boot storms with 500+ simultaneous DHCP transactions create broadcast bursts that can overwhelm DHCP servers.

### Optimal Size

Industry consensus from Cisco, Juniper, and operational experience:

| Domain Size | Assessment | Typical Use |
|:---|:---|:---|
| < 100 hosts | Optimal | Server VLANs, management networks |
| 100-254 (/24) | Good | Standard access VLANs |
| 255-500 | Acceptable | Large floor VLANs with modern hardware |
| 500-1,000 | Problematic | Requires tuning (ARP timers, gc_thresh) |
| > 1,000 | Avoid | Split into multiple VLANs immediately |

---

## 3. Spanning Tree Protocol — Loop Prevention and Broadcast Storm Containment

### The Problem

Redundant Layer 2 links create physical loops. Without loop prevention, broadcast frames circulate endlessly, doubling with each iteration (no TTL in Ethernet). Spanning Tree Protocol (STP) builds a loop-free logical topology.

### Why Ethernet Loops Are Catastrophic

Unlike IP packets (which have a TTL field and are decremented at each hop), Ethernet frames have no hop counter. A broadcast frame entering a loop circulates forever:

$$\text{Frames at time } t = F_0 \times 2^{t / T_{loop}}$$

Where $T_{loop}$ is the loop propagation time. On a two-switch loop with 1 Gbps links:

$$T_{loop} \approx 5 \text{ μs (near-instantaneous)}$$

A single broadcast frame becomes millions of copies within milliseconds. The switches' CPUs saturate, MAC tables thrash, and all legitimate traffic is dropped.

### STP Algorithm (802.1D)

1. **Root bridge election**: Bridge with lowest Bridge ID (priority + MAC) becomes root.

$$\text{Bridge ID} = \text{Priority (16 bits)} \| \text{MAC (48 bits)}$$

2. **Root port selection**: Each non-root bridge selects the port with the lowest cost path to the root.

$$\text{Root Path Cost} = \sum_{i} C_i \quad \text{(sum of link costs along path)}$$

3. **Designated port selection**: Each segment selects the port with the lowest root path cost as the designated forwarder.

4. **Blocking**: All remaining ports are placed in blocking state — they receive BPDUs but do not forward data frames.

### STP Timers and Convergence

| Timer | Default | Purpose |
|:---|:---:|:---|
| Hello Time | 2 sec | Interval between BPDU transmissions |
| Max Age | 20 sec | Time before a BPDU is considered stale |
| Forward Delay | 15 sec | Time spent in Listening and Learning states |

Worst-case convergence:

$$T_{convergence} = T_{max\_age} + 2 \times T_{forward\_delay} = 20 + 30 = 50 \text{ seconds}$$

During these 50 seconds, the affected ports do not forward traffic. This was the primary motivation for RSTP.

### RSTP (802.1w) — Rapid Convergence

RSTP replaces the timer-based approach with active topology negotiation:

- **Proposal/Agreement**: Direct handshake between neighbor switches (sub-second)
- **Alternate ports**: Pre-computed backup root ports (instant failover)
- **Edge ports**: Ports connected to end hosts skip directly to forwarding

$$T_{RSTP} \approx 3 \times T_{hello} = 6 \text{ seconds (typical)}$$

In practice, RSTP often converges in under 1 second for direct link failures.

### MSTP (802.1Q/802.1s) — Scaling STP

PVST+ (Per-VLAN Spanning Tree Plus) runs one STP instance per VLAN. With 500 VLANs:

$$\text{BPDUs/sec per trunk} = \frac{500 \text{ VLANs}}{2 \text{ sec hello}} = 250 \text{ BPDUs/sec}$$

$$\text{CPU cost} \propto V \times P \quad \text{(VLANs × trunk ports)}$$

MSTP maps groups of VLANs to a small number of spanning tree instances (typically 2-16), reducing overhead by 30-250x:

| Configuration | Instances | BPDUs/sec (24 trunks) | Relative CPU |
|:---|:---:|:---:|:---:|
| PVST+ (100 VLANs) | 100 | 1,200 | 100x |
| PVST+ (500 VLANs) | 500 | 6,000 | 500x |
| MSTP (4 instances) | 4 | 48 | 4x |
| MSTP (16 instances) | 16 | 192 | 16x |

---

## 4. VLAN Trunk Tagging — 802.1Q Frame Encapsulation

### The Problem

When multiple VLANs traverse a single physical link between switches, each frame must be tagged with its VLAN membership. 802.1Q inserts a 4-byte tag into the Ethernet frame.

### 802.1Q Tag Structure

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|         TPID (0x8100)         |PCP|D|       VLAN ID          |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

- **TPID** (16 bits): Tag Protocol Identifier. 0x8100 for 802.1Q.
- **PCP** (3 bits): Priority Code Point. 0-7 for QoS / CoS mapping.
- **DEI** (1 bit): Drop Eligible Indicator. Marks frames that can be dropped during congestion.
- **VID** (12 bits): VLAN Identifier. 0-4095 (0 and 4095 reserved, 4094 usable).

### Access Ports vs Trunk Ports

| Port Type | Behavior | Frame Format |
|:---|:---|:---|
| Access | Member of one VLAN, strips/adds tag | Untagged (host unaware) |
| Trunk | Carries multiple VLANs, preserves tags | Tagged (except native VLAN) |

### Native VLAN and Security

The native VLAN is the VLAN whose traffic is sent untagged on a trunk. Default: VLAN 1.

**VLAN hopping attack (double tagging)**:

1. Attacker on access port sends a frame with two 802.1Q tags.
2. Outer tag = native VLAN (matches attacker's access VLAN).
3. First switch strips the outer tag (it's the native VLAN), forwards with inner tag.
4. Second switch reads the inner tag and forwards to the target VLAN.
5. Attack is unidirectional — replies cannot return the same way.

Prevention:
- Set native VLAN to an unused VLAN (e.g., 999).
- Tag the native VLAN explicitly (`vlan dot1q tag native` on Cisco).
- Never assign user traffic to VLAN 1.

### Trunk Pruning

By default, a trunk carries all VLANs. Pruning limits the trunk to only the VLANs needed:

$$BW_{saved} = (V_{total} - V_{needed}) \times R_{bcast} \times N_{avg} \times S_{avg}$$

If a trunk carries 200 VLANs but only 20 are needed on the remote switch, pruning eliminates broadcast/multicast/unknown-unicast flooding for 180 unnecessary VLANs.

---

## 5. Historical Context — From Shared Media to Switched Fabric

### Thick Ethernet (10BASE5) — 1980

- 10 Mbps over 50-ohm RG-8 coaxial cable (yellow cable, "thicknet")
- Maximum segment: 500 m, maximum 100 transceivers per segment
- Vampire taps: physically pierced the cable to attach
- Entire cable was ONE collision domain and ONE broadcast domain
- 5-4-3 rule: max 5 segments, 4 repeaters, 3 populated segments

### Thin Ethernet (10BASE2) — 1985

- 10 Mbps over RG-58 coaxial cable ("thinnet", "cheapernet")
- Maximum segment: 185 m, maximum 30 nodes per segment
- BNC T-connectors, terminators at both ends
- Still one collision domain per segment, bus topology
- A single break in the cable took down the entire segment

### Token Ring (IEEE 802.5) — 1985

- 4/16 Mbps, deterministic access (no collisions by design)
- Token-passing protocol: station must hold the token to transmit
- Star-wired ring topology (MAU — Multistation Access Unit)
- Beaconing protocol for fault isolation
- Died commercially due to cost (proprietary IBM hardware) and Ethernet's speed evolution

### Ethernet Hubs (10BASE-T) — 1990

- 10 Mbps over twisted pair (Cat3/Cat5)
- Star topology with central hub
- Hub is a multiport repeater — electrical signal regeneration only
- Still one collision domain per hub, but easier cabling than coax
- Cascading hubs: 5-4-3 rule still applied

### Ethernet Switches — 1994

- Learning bridges with dedicated bandwidth per port
- Full-duplex support eliminated CSMA/CD
- MAC address table: learn source MAC, forward by destination MAC
- Microsegmentation: each port = one collision domain
- Cut-through switching: forward before entire frame is received (low latency)
- Store-and-forward: receive entire frame, verify FCS, then forward (error filtering)

### Timeline Summary

| Year | Technology | Collision Domain | Broadcast Domain |
|:---:|:---|:---|:---|
| 1980 | 10BASE5 (thick coax) | Entire cable segment | Entire cable segment |
| 1985 | 10BASE2 (thin coax) | Entire cable segment | Entire cable segment |
| 1985 | Token Ring | N/A (deterministic) | Entire ring |
| 1990 | 10BASE-T (hub) | All hub ports | All hub ports |
| 1994 | Ethernet switch | Per port | All ports (flat) |
| 1998 | 802.1Q VLANs | Per port | Per VLAN |
| 2004 | RSTP + VLANs | Per port (full-duplex) | Per VLAN, loop-free |
| 2010+ | VXLAN/EVPN | Per port | Per VNI (overlay) |

The evolution from thick coax to VXLAN is the progressive shrinking of both collision and broadcast domains: from "one cable = one domain" to "one virtual tunnel endpoint = one domain" with complete independence from physical topology.

---

## 6. Summary of Formulas

| Formula | Math Type | Domain |
|:---|:---|:---|
| $(1 - 1/N)^{N-1} \to 1/e$ | Probability limit | CSMA/CD max efficiency |
| $k \in [0, 2^{\min(c,10)} - 1]$ | Uniform random | Exponential backoff window |
| $1 / (1 + 5.44a)$ | Throughput bound | CSMA/CD channel utilization |
| $N \times R \times S \times 8$ | Product | Broadcast bandwidth |
| $F_0 \times 2^{t/T_{loop}}$ | Exponential growth | Broadcast storm amplification |
| $T_{max\_age} + 2 \times T_{fwd\_delay}$ | Summation | STP convergence time |

## Prerequisites

- ethernet frame structure, probability theory, VLAN fundamentals, Layer 2 switching

---

*The history of Ethernet is the history of eliminating collisions and containing broadcasts. CSMA/CD was an elegant solution to shared-medium contention, but its 36.8% theoretical ceiling and exponential backoff instability made it obsolete the moment full-duplex switches arrived. Broadcast domains remain the active battleground: every host added to a VLAN imposes a tax on every other host, and the math shows that tax grows linearly with N. VLANs, STP, and eventually VXLAN each pushed the boundary outward, but the fundamental constraint persists — broadcast traffic is an O(N) problem with no shortcut.*
