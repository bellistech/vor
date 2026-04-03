# The Mathematics of BFD — Sub-Second Failure Detection Timing

> *Bidirectional Forwarding Detection is a pure timing protocol — it detects link or path failures by exchanging packets at precise intervals and declaring failure after a calculated number of missed packets. The math is simple but the engineering impact is profound: replacing 40-second OSPF dead timers with 150 ms detection.*

---

## 1. Detection Time Formula

### The Core Equation

$$T_{detect} = T_{interval} \times M$$

Where:
- $T_{interval}$ = BFD packet transmission interval
- $M$ = detect multiplier (number of consecutive missed packets)

### Common Configurations

| Interval | Multiplier | Detection Time | Use Case |
|:---:|:---:|:---:|:---|
| 50 ms | 3 | 150 ms | Data center links |
| 100 ms | 3 | 300 ms | Enterprise WAN |
| 300 ms | 3 | 900 ms | Low-speed links |
| 1,000 ms | 3 | 3 sec | Constrained devices |
| 50 ms | 5 | 250 ms | Noisy links |

### Comparison with Protocol-Native Detection

| Protocol | Native Detection | With BFD | Improvement |
|:---|:---:|:---:|:---:|
| OSPF | 40 sec (4x hello) | 150 ms | 267x faster |
| IS-IS | 30 sec (3x hello) | 150 ms | 200x faster |
| BGP | 90 sec (3x keepalive) | 150 ms | 600x faster |
| EIGRP | 15 sec (3x hello) | 150 ms | 100x faster |
| Static routes | None (manual) | 150 ms | Infinite |

### Impact on Traffic Loss

$$\text{Traffic lost during failover} = T_{detect} \times R_{traffic}$$

| Detection Time | At 1 Gbps | At 10 Gbps |
|:---:|:---:|:---:|
| 150 ms (BFD) | 18.75 MB | 187.5 MB |
| 3 sec (fast OSPF) | 375 MB | 3.75 GB |
| 40 sec (default OSPF) | 5 GB | 50 GB |

---

## 2. Packet Rate — Bandwidth Overhead

### BFD Packet Size

| Component | Size |
|:---|:---:|
| BFD control packet | 24 bytes |
| UDP header | 8 bytes |
| IP header | 20 bytes |
| Ethernet | 14 bytes |
| **Total on wire** | **66 bytes** |

### Bandwidth per BFD Session

$$BW_{session} = \frac{S_{packet} \times 8}{T_{interval}}$$

| Interval | Packets/sec | Bandwidth (each direction) |
|:---:|:---:|:---:|
| 50 ms | 20 pps | 10.56 kbps |
| 100 ms | 10 pps | 5.28 kbps |
| 300 ms | 3.33 pps | 1.76 kbps |
| 1,000 ms | 1 pps | 0.53 kbps |

### Aggregate BFD Load

$$BW_{total} = N_{sessions} \times BW_{session} \times 2 \quad \text{(bidirectional)}$$

| Sessions | Interval | Total Bandwidth |
|:---:|:---:|:---:|
| 10 | 50 ms | 211 kbps |
| 100 | 50 ms | 2.1 Mbps |
| 1,000 | 100 ms | 10.6 Mbps |
| 10,000 | 300 ms | 35.2 Mbps |

At 10,000 sessions with 50 ms interval: $10,000 \times 10.56 \times 2 = 211$ Mbps — significant for control plane processing.

---

## 3. False Positive Analysis

### The Problem

If BFD packets are delayed (not lost), a false failure declaration occurs.

### False Positive Probability

For a link with packet loss rate $p$ (independent per packet) and multiplier $M$:

$$P_{false} = p^M$$

| Loss Rate | M=3 | M=5 | M=10 |
|:---:|:---:|:---:|:---:|
| 0.1% | $10^{-9}$ | $10^{-15}$ | $10^{-30}$ |
| 1% | $10^{-6}$ | $10^{-10}$ | $10^{-20}$ |
| 5% | $1.25 \times 10^{-4}$ | $3.1 \times 10^{-7}$ | $9.8 \times 10^{-14}$ |
| 10% | $10^{-3}$ | $10^{-5}$ | $10^{-10}$ |

### Design Tradeoff

- **Lower multiplier** → faster detection, higher false positive risk
- **Higher multiplier** → slower detection, fewer false positives

$$\text{Tradeoff:} \quad T_{detect} = T_{interval} \times M \propto -\log(P_{false})$$

### Jitter Consideration

BFD adds random jitter (up to 25%) to avoid synchronization:

$$T_{actual} = T_{interval} \times (1 - J)$$

Where $J$ is random in $[0, 0.25]$. This means detection time can vary:

$$T_{detect} \in [0.75 \times T_{interval} \times M, \;\; T_{interval} \times M]$$

For 50 ms / M=3: detection between 112.5 ms and 150 ms.

---

## 4. Echo Mode — Reduced CPU Load

### The Problem

BFD at 50 ms intervals means the control plane must process 20 packets/sec per session. At scale, this overloads the CPU.

### Echo Mode Solution

BFD Echo packets are forwarded by the data plane (no CPU involvement on the remote end):

$$\text{CPU load}_{remote} = 0 \quad \text{(data plane forwarding only)}$$

### CPU Savings

| Sessions | Standard (both ends process) | Echo (one end processes) |
|:---:|:---:|:---:|
| 100 | 4,000 pps (CPU) | 2,000 pps (CPU) |
| 1,000 | 40,000 pps | 20,000 pps |
| 10,000 | 400,000 pps | 200,000 pps |

### Echo Detection Formula

$$T_{detect\_echo} = T_{echo\_interval} \times M_{echo}$$

Echo interval is independent of control packet interval — can be set more aggressively since echo doesn't load the remote CPU.

---

## 5. Multi-Hop BFD — Path Monitoring

### The Difference

| Mode | Scope | TTL | Use Case |
|:---|:---|:---:|:---|
| Single-hop | Direct link | 255 | Point-to-point links |
| Multi-hop | End-to-end path | Configurable | eBGP multihop, tunnels |

### Multi-Hop Timing

Multi-hop BFD detects path failures between non-adjacent routers:

$$T_{detect} = T_{interval} \times M + T_{jitter}$$

The interval is typically longer (300 ms+) due to path variation:

$$T_{detect} = 300 \times 3 = 900 \text{ ms (multi-hop typical)}$$

---

## 6. BFD with Protocol Integration — Convergence Stack

### Total Convergence Time

$$T_{total} = T_{BFD\_detect} + T_{protocol\_notify} + T_{SPF/recalc} + T_{FIB\_update}$$

| Component | Without BFD | With BFD |
|:---|:---:|:---:|
| Failure detection | 40 sec (OSPF) | 150 ms |
| Protocol notification | ~0 | ~1 ms |
| SPF computation | ~5 ms | ~5 ms |
| FIB programming | ~50 ms | ~50 ms |
| **Total** | **~40 sec** | **~206 ms** |

### Convergence improvement: **194x faster.**

---

## 7. Summary of Formulas

| Formula | Math Type | Application |
|:---|:---|:---|
| $T_{interval} \times M$ | Product | Detection time |
| $S \times 8 / T_{interval}$ | Rate | Per-session bandwidth |
| $p^M$ | Exponential | False positive probability |
| $T \times (1 - J)$ | Random scaling | Jitter adjustment |
| $T_{BFD} + T_{notify} + T_{SPF} + T_{FIB}$ | Summation | Total convergence |

---

*BFD does one thing and does it with mathematical precision: detect failures in milliseconds. It replaced the coarse timers of routing protocols with a dedicated heartbeat, cutting network convergence from tens of seconds to hundreds of milliseconds.*
