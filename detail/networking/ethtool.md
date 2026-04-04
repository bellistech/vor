# ethtool Deep Dive — Theory & Internals

> *ethtool exposes the physical layer of networking — link speed negotiation, ring buffer sizing, NIC offload engines, and error counters. The math covers auto-negotiation priority, interrupt coalescing tradeoffs, and the relationship between ring buffer depth and packet loss.*

---

## 1. Link Speed Negotiation — Auto-Negotiation

### The Protocol

IEEE 802.3 auto-negotiation uses a priority resolution algorithm. Each end advertises capabilities, and the highest common speed/duplex is selected:

$$\text{Link} = \max(\text{Advertised}_A \cap \text{Advertised}_B)$$

### Priority Order (Highest First)

| Priority | Speed/Duplex | Encoding |
|:---:|:---|:---|
| 1 | 100G Full | IEEE 802.3bj/bm |
| 2 | 40G Full | IEEE 802.3ba |
| 3 | 25G Full | IEEE 802.3by |
| 4 | 10G Full | IEEE 802.3ae |
| 5 | 5G Full | IEEE 802.3bz |
| 6 | 2.5G Full | IEEE 802.3bz |
| 7 | 1000 Full | IEEE 802.3ab |
| 8 | 1000 Half | IEEE 802.3ab |
| 9 | 100 Full | IEEE 802.3u |
| 10 | 100 Half | IEEE 802.3u |
| 11 | 10 Full | IEEE 802.3i |
| 12 | 10 Half | IEEE 802.3i |

### Mismatch Detection

When auto-negotiation fails (one side forced, other auto):

$$\text{Duplex mismatch} \rightarrow \text{Late collisions, CRC errors, degraded throughput}$$

Effective throughput with duplex mismatch:

$$R_{effective} \approx \frac{R_{link}}{10} \quad \text{(severe degradation)}$$

A 1 Gbps link with duplex mismatch may deliver only ~100 Mbps.

---

## 2. Ring Buffer Sizing — Packet Loss Prevention

### The Problem

NIC ring buffers hold packets between hardware receipt and kernel processing. If the kernel can't keep up, the buffer fills and packets are dropped.

### Ring Buffer Model

$$N_{ring} = \text{ethtool -g <dev>} \quad \text{(shows current and max)}$$

$$T_{buffer} = \frac{N_{ring} \times S_{avg}}{R_{line\_rate}}$$

Time before overflow at line rate:

| Ring Size | Avg Packet | At 1 Gbps | At 10 Gbps |
|:---:|:---:|:---:|:---:|
| 256 | 1,500 B | 3.1 ms | 0.31 ms |
| 512 | 1,500 B | 6.1 ms | 0.61 ms |
| 1,024 | 1,500 B | 12.3 ms | 1.23 ms |
| 4,096 | 1,500 B | 49.2 ms | 4.92 ms |
| 8,192 | 1,500 B | 98.3 ms | 9.83 ms |

### Memory Cost

$$M_{ring} = N_{ring} \times S_{descriptor} + N_{ring} \times S_{buffer}$$

Where $S_{descriptor} \approx 16$ bytes, $S_{buffer} \approx 2$ KB (DMA-mapped).

| Ring Size | Descriptor Memory | Buffer Memory | Total |
|:---:|:---:|:---:|:---:|
| 256 | 4 KB | 512 KB | 516 KB |
| 1,024 | 16 KB | 2 MB | ~2 MB |
| 4,096 | 64 KB | 8 MB | ~8 MB |
| 8,192 | 128 KB | 16 MB | ~16 MB |

Per queue. With 8 queues: multiply by 8.

### Optimal Ring Size

$$N_{optimal} = R_{pps} \times T_{latency} \times M_{safety}$$

Where $T_{latency}$ = worst-case kernel scheduling latency, $M_{safety}$ = 2-4x margin.

For 1 Mpps and 1 ms max latency with 3x safety: $N = 1,000,000 \times 0.001 \times 3 = 3,000$ entries.

---

## 3. Interrupt Coalescing — Latency vs CPU Tradeoff

### The Problem

Each packet can trigger a hardware interrupt. At high packet rates, interrupts themselves consume all CPU time.

### Interrupt Rate

Without coalescing:

$$I_{rate} = PPS$$

At 1 Mpps: 1 million interrupts/sec = ~30% of a modern CPU core.

### With Coalescing

$$I_{rate} = \min\left(\frac{PPS}{N_{coalesce}}, \frac{1}{T_{timeout}}\right)$$

Where $N_{coalesce}$ = packets per interrupt, $T_{timeout}$ = max delay before interrupt fires.

| PPS | No Coalescing | Coalesce=64 | Coalesce=256 |
|:---:|:---:|:---:|:---:|
| 100,000 | 100K int/s | 1,563 int/s | 391 int/s |
| 1,000,000 | 1M int/s | 15,625 int/s | 3,906 int/s |
| 10,000,000 | 10M int/s | 156,250 int/s | 39,063 int/s |

### Latency Impact

$$T_{added\_latency} = \frac{N_{coalesce}}{PPS} \quad \text{(avg case)}$$

$$T_{max\_latency} = T_{timeout}$$

| Coalesce Packets | At 100K pps | At 1M pps |
|:---:|:---:|:---:|
| 1 (disabled) | 10 us | 1 us |
| 32 | 320 us | 32 us |
| 64 | 640 us | 64 us |
| 256 | 2.56 ms | 256 us |

**Tradeoff:** Higher coalescing = lower CPU, higher latency. Typical: `rx-usecs 50` (50 us timeout).

---

## 4. NIC Offload — Throughput Impact

### Common Offloads

| Offload | What It Does | CPU Savings |
|:---|:---|:---|
| TSO (TCP Segmentation) | NIC segments large TCP buffers | ~30-50% CPU reduction |
| GRO (Generic Receive) | NIC merges small packets into large | ~20-40% CPU reduction |
| Checksum offload | NIC computes/verifies checksums | ~5-10% CPU reduction |
| RSS (Receive Side Scaling) | Distribute packets across CPU cores | Linear scaling |
| LRO (Large Receive) | Merge received TCP segments | ~20-30% CPU reduction |

### TSO Impact

Without TSO: kernel creates $\lceil S_{buffer} / MSS \rceil$ small packets.

With TSO: kernel hands one large buffer to NIC, NIC segments on hardware.

$$\text{Kernel packets} = \begin{cases} \lceil S / MSS \rceil & \text{without TSO} \\ 1 & \text{with TSO (up to 64 KB)} \end{cases}$$

For a 64 KB send: 44 packets (without TSO) vs 1 call (with TSO) = **44x fewer kernel operations**.

### RSS Queue Distribution

$$\text{Queue} = H(\text{flow 5-tuple}) \mod N_{queues}$$

| Queues | Flows | Expected Flows/Queue | CPU Usage |
|:---:|:---:|:---:|:---|
| 1 | 10,000 | 10,000 | 1 core saturated |
| 4 | 10,000 | 2,500 | Distributed across 4 cores |
| 8 | 10,000 | 1,250 | Distributed across 8 cores |
| 16 | 10,000 | 625 | Distributed across 16 cores |

---

## 5. Error Counter Analysis

### `ethtool -S` Statistics

| Counter | Meaning | Threshold |
|:---|:---|:---:|
| rx_crc_errors | CRC failures (bad cable/transceiver) | Any > 0 per hour |
| rx_frame_errors | Alignment errors | Any > 0 |
| rx_missed_errors | Ring buffer overflow | Any > 0 (tune ring/coalesce) |
| rx_fifo_errors | NIC FIFO overflow | Any > 0 (hardware issue) |
| tx_aborted_errors | Transmission aborted | Any > 0 |
| collisions | Ethernet collisions | 0 (full duplex = no collisions) |

### Error Rate Calculation

$$R_{error} = \frac{\Delta E}{\Delta t}$$

$$\text{Error ratio} = \frac{E_{errors}}{P_{total}} \times 10^6 \quad \text{(errors per million packets)}$$

| Errors/Hour | At 100K pps | EPM (errors per million) |
|:---:|:---:|:---:|
| 1 | 360M packets/hour | 0.003 |
| 10 | 360M | 0.028 |
| 100 | 360M | 0.28 |
| 1,000 | 360M | 2.78 |

Anything above 1 EPM warrants investigation. Above 10 EPM indicates a failing link.

---

## 6. Pause Frames — Flow Control

### IEEE 802.3x Pause

When a receiver's buffer is filling, it sends a PAUSE frame:

$$T_{pause} = \text{quanta} \times 512 \text{ bit-times}$$

At 10 Gbps: 1 bit-time = 0.1 ns. Max quanta = 65,535:

$$T_{max\_pause} = 65,535 \times 512 \times 0.1 \text{ ns} = 3.36 \text{ ms}$$

### Impact on Throughput

During pause, the sender is idle:

$$R_{effective} = R_{link} \times (1 - \frac{T_{pause}}{T_{total}})$$

Heavy pause frame activity can reduce throughput by 30-50%.

---

## 7. Summary of Formulas

| Formula | Math Type | Application |
|:---|:---|:---|
| $\max(A \cap B)$ | Set intersection | Auto-negotiation |
| $N_{ring} \times S / R$ | Division | Ring buffer time |
| $PPS / N_{coalesce}$ | Division | Interrupt rate |
| $N_{coalesce} / PPS$ | Division | Added latency |
| $H(\text{5-tuple}) \mod N$ | Hash/modular | RSS queue assignment |
| $\Delta E / \Delta t$ | Rate | Error rate |
| $\text{quanta} \times 512 \times T_{bit}$ | Product | Pause duration |

## Prerequisites

- NIC hardware concepts, ring buffer arithmetic, link negotiation

---

*ethtool is the window into your NIC's soul — it shows you link speed, ring buffers, offload engines, and error counters that no other tool can access. When tcpdump shows packet loss but the application looks fine, ethtool's error counters tell you whether it's a software problem or a dying cable.*
