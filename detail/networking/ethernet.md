# The Mathematics of Ethernet — Frame Timing, Collision Domains, and Switching Capacity

> *Ethernet evolved from a shared-medium collision protocol governed by exponential backoff to a full-duplex switched fabric with deterministic timing. The math spans collision probability (CSMA/CD), frame gap calculations, switch fabric capacity, and MAC address space.*

---

## 1. Frame Structure — Overhead Analysis

### Ethernet Frame (IEEE 802.3)

| Component | Size | On Wire? |
|:---|:---:|:---:|
| Preamble | 7 bytes | Yes |
| SFD (Start Frame Delimiter) | 1 byte | Yes |
| Destination MAC | 6 bytes | Yes |
| Source MAC | 6 bytes | Yes |
| EtherType / Length | 2 bytes | Yes |
| Payload | 46-1,500 bytes | Yes |
| FCS (CRC-32) | 4 bytes | Yes |
| Inter-Frame Gap (IFG) | 12 bytes | Yes (idle) |
| **Total wire overhead** | **38 bytes** | |

### Efficiency

$$\eta = \frac{L_{payload}}{L_{payload} + 38}$$

| Payload | Total Wire | Efficiency |
|:---:|:---:|:---:|
| 46 B (minimum) | 84 B | 54.8% |
| 64 B | 102 B | 62.7% |
| 512 B | 550 B | 93.1% |
| 1,500 B (standard max) | 1,538 B | 97.5% |
| 9,000 B (jumbo) | 9,038 B | 99.6% |

### Why 46-Byte Minimum Payload?

The minimum frame (64 bytes on wire, excluding preamble/SFD/IFG) ensures collision detection works. At 10 Mbps on a 2,500 m cable:

$$T_{slot} = \frac{2 \times D_{max}}{v} = \frac{2 \times 2500}{2 \times 10^8} = 25 \mu s$$

$$\text{Min frame bits} = T_{slot} \times \text{Rate} = 25 \times 10^{-6} \times 10^7 = 250 \text{ bits}$$

Rounded up to 512 bits = 64 bytes. Below this, a collision might not be detected before transmission completes.

---

## 2. CSMA/CD — Collision and Backoff Math

### Collision Probability

With $N$ stations, the probability that a transmission attempt succeeds (no collision):

$$P_{success} = \left(1 - \frac{1}{N}\right)^{N-1} \times \frac{1}{N} \times N = \left(1 - \frac{1}{N}\right)^{N-1}$$

$$\lim_{N \to \infty} P_{success} = \frac{1}{e} \approx 36.8\%$$

### Exponential Backoff (BEB)

After $c$ collisions, the station waits a random number of slot times from $[0, 2^c - 1]$:

$$E[\text{wait}] = \frac{2^{\min(c, 10)} - 1}{2} \times T_{slot}$$

| Collision # | Window | Max Wait (slots) | Avg Wait (slots) |
|:---:|:---:|:---:|:---:|
| 1 | [0, 1] | 1 | 0.5 |
| 2 | [0, 3] | 3 | 1.5 |
| 3 | [0, 7] | 7 | 3.5 |
| 5 | [0, 31] | 31 | 15.5 |
| 10 | [0, 1023] | 1,023 | 511.5 |
| 16 | Abort | Frame discarded | N/A |

After 16 collisions, the frame is dropped — total probability of reaching this:

$$P_{16\_collisions} \approx (1 - P_{success})^{16}$$

For heavy load ($P_{success} = 0.3$): $P_{abort} = 0.7^{16} = 0.0033$ (0.33%).

---

## 3. Switching Capacity — Fabric Math

### Non-Blocking Requirement

A switch is non-blocking if it can forward at line rate on all ports simultaneously:

$$C_{fabric} = N_{ports} \times R_{port} \times 2 \quad \text{(full duplex)}$$

But typically expressed as:

$$C_{fabric} = N_{ports} \times R_{port} \quad \text{(unidirectional aggregate)}$$

### Worked Examples

| Switch | Ports | Port Speed | Fabric Required |
|:---|:---:|:---:|:---:|
| Access | 48 | 1G | 48 Gbps |
| Distribution | 48 | 10G | 480 Gbps |
| Core | 32 | 100G | 3.2 Tbps |
| DC spine | 64 | 400G | 25.6 Tbps |

### Forwarding Rate (PPS)

$$PPS_{max} = \frac{C_{fabric}}{(64 + 20) \times 8} = \frac{C_{fabric}}{672}$$

For a 48-port 1G switch: $\frac{48 \times 10^9}{672} = 71.4 \text{ Mpps}$.

### MAC Address Table Sizing

$$M = \frac{N_{hosts}}{P_{active}}$$

Where $P_{active}$ = fraction of hosts active concurrently.

| Hosts | Table Size | Memory (~80 B/entry) |
|:---:|:---:|:---:|
| 1,000 | 1,000 | 80 KB |
| 16,000 | 16,000 | 1.28 MB |
| 128,000 | 128,000 | 10 MB |

Common switch limits: 8K-128K MAC entries. Overflow causes flooding (performance and security concern).

---

## 4. MAC Address Space

### 48-Bit Address

$$N_{MAC} = 2^{48} = 281,474,976,710,656 \approx 281 \text{ trillion}$$

### OUI Structure

First 24 bits = Organizationally Unique Identifier (OUI). Remaining 24 bits = vendor-assigned.

$$\text{OUIs available} = 2^{24} = 16,777,216$$

$$\text{Addresses per OUI} = 2^{24} = 16,777,216$$

### Collision Probability (Random MACs)

With $n$ random MACs on a network (birthday problem):

$$P_{collision} \approx \frac{n^2}{2 \times 2^{48}}$$

For 1 million hosts: $P = \frac{10^{12}}{2 \times 2^{48}} = 1.78 \times 10^{-3}$ (0.18%).

---

## 5. VLAN Scaling

### 802.1Q Tag

4-byte tag inserted into the frame: 2 bytes TPID (0x8100) + 2 bytes TCI (3-bit PCP + 1-bit DEI + 12-bit VID).

$$N_{VLANs} = 2^{12} = 4,096$$

Reserved: VLAN 0 (priority only), VLAN 4095 (reserved). Usable: 4,094.

### 802.1ad (Q-in-Q) Double Tagging

$$N_{segments} = 4,094 \times 4,094 = 16,760,836$$

### Frame Size Impact

| Tagging | Additional Bytes | Max Payload (1500 MTU) |
|:---|:---:|:---:|
| None | 0 | 1,500 |
| 802.1Q | 4 | 1,500 (MTU increased to 1,504 on trunk) |
| Q-in-Q | 8 | 1,500 (MTU increased to 1,508) |

---

## 6. Spanning Tree — Convergence Timing

### STP (802.1D) Convergence

$$T_{STP} = T_{max\_age} + 2 \times T_{forward\_delay}$$

$$= 20 + 2 \times 15 = 50 \text{ seconds}$$

### RSTP (802.1w) Convergence

$$T_{RSTP} \approx 3 \times T_{hello} = 3 \times 2 = 6 \text{ seconds (typical)}$$

In practice, often sub-second with proposal/agreement mechanism.

### MSTP Instance Scaling

$$I_{max} = 65 \text{ MST instances (0-64)}$$

Each instance maps a set of VLANs to a spanning tree topology.

---

## 7. Summary of Formulas

| Formula | Math Type | Application |
|:---|:---|:---|
| $L / (L + 38)$ | Ratio | Frame efficiency |
| $(1 - 1/N)^{N-1}$ | Probability | Collision avoidance |
| $[0, 2^{\min(c,10)}-1]$ | Exponential backoff | CSMA/CD retry |
| $N \times R \times 2$ | Product | Fabric capacity |
| $C / 672$ | Division | Max PPS (64B) |
| $2^{48}$ | Exponent | MAC address space |
| $20 + 2(15) = 50$ sec | Summation | STP convergence |

---

*Ethernet's journey from a 2.94 Mbps shared coax cable to 800 Gbps full-duplex switched fabric is a story told in math — from collision probability to forwarding rate calculations to the scaling limits that drove VXLAN's creation.*
