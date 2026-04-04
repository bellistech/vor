# The Mathematics of SNMP — OID Trees, Polling Intervals, and Trap Storm Analysis

> *SNMP organizes all managed information in a global tree (MIB) addressed by Object Identifiers, polls devices at intervals that determine monitoring resolution, and generates traps whose storm behavior follows queuing theory. The math governs monitoring system scalability.*

---

## 1. OID Tree — Hierarchical Addressing

### The Structure

An OID is a path through a global tree:

$$\text{OID} = n_1.n_2.n_3.\ldots.n_k$$

Each node $n_i$ is a non-negative integer. The tree has fixed roots:

$$\text{iso.org.dod.internet.mgmt.mib-2} = 1.3.6.1.2.1$$

### MIB-2 Standard Objects

| OID | Name | Type |
|:---|:---|:---|
| 1.3.6.1.2.1.1.1 | sysDescr | String |
| 1.3.6.1.2.1.1.3 | sysUpTime | TimeTicks |
| 1.3.6.1.2.1.2.1 | ifNumber | Integer |
| 1.3.6.1.2.1.2.2.1.10 | ifInOctets | Counter32 |
| 1.3.6.1.2.1.2.2.1.16 | ifOutOctets | Counter32 |

### OID Space Size

Each node can have $2^{32}$ children (in practice, much fewer). The theoretical tree size:

$$|T| = (2^{32})^D \text{ (for depth } D\text{)}$$

In practice, the active OID space is sparse — a typical device exposes 1,000-100,000 OIDs.

---

## 2. Polling Math — Bandwidth and Timing

### SNMP Poll Bandwidth

Each SNMP GET/RESPONSE exchange:

$$B_{poll} = (S_{request} + S_{response}) \times \frac{N_{objects}}{O_{per\_PDU}}$$

Typical sizes: request ~80 bytes, response ~100 bytes per object.

### Polling Bandwidth per Device

For $O$ monitored objects, polled every $T$ seconds:

$$BW_{device} = \frac{O \times (80 + 100)}{T} = \frac{180 \times O}{T} \text{ bytes/sec}$$

| Objects | Poll Interval | Bandwidth |
|:---:|:---:|:---:|
| 50 | 300 sec (5 min) | 30 B/s |
| 50 | 60 sec (1 min) | 150 B/s |
| 200 | 60 sec | 600 B/s |
| 1,000 | 30 sec | 6,000 B/s |

### Network-Wide Polling Load

$$BW_{total} = N_{devices} \times BW_{device}$$

| Devices | Objects/Device | Interval | Total Bandwidth |
|:---:|:---:|:---:|:---:|
| 100 | 50 | 60 sec | 15 KB/s |
| 1,000 | 100 | 60 sec | 300 KB/s |
| 10,000 | 200 | 60 sec | 6 MB/s |
| 10,000 | 200 | 30 sec | 12 MB/s |

### Poller Capacity

$$\text{Polls/sec} = \frac{N_{devices} \times O_{per\_device}}{T_{interval}}$$

Each poll takes $T_{poll} \approx RTT + T_{agent}$. A single-threaded poller at 5 ms per poll:

$$\text{Max polls/sec} = \frac{1000}{5} = 200 \text{ polls/sec}$$

$$\text{Max devices (60s interval, 50 objects)} = \frac{200 \times 60}{50} = 240 \text{ devices}$$

Multi-threaded poller with 32 threads: $240 \times 32 = 7,680$ devices.

---

## 3. Counter Math — Rate Derivation

### The Problem

SNMP counters (ifInOctets, ifOutOctets) are cumulative. To get a rate:

$$\text{Rate} = \frac{\Delta C}{\Delta T} = \frac{C_{t_2} - C_{t_1}}{t_2 - t_1}$$

### Counter Wrap

Counter32: wraps at $2^{32} = 4,294,967,296$.

$$T_{wrap} = \frac{2^{32}}{R_{bytes/sec}}$$

| Interface Speed | Wrap Time |
|:---:|:---:|
| 10 Mbps | 57.3 minutes |
| 100 Mbps | 5.7 minutes |
| 1 Gbps | 34.4 seconds |
| 10 Gbps | 3.4 seconds |

At 1 Gbps, Counter32 wraps every 34 seconds — if polled every 60 seconds, wraps are missed.

**Solution:** Counter64 (ifHCInOctets, ifHCOutOctets) wraps at $2^{64}$:

$$T_{wrap}(64) = \frac{2^{64}}{R} = \frac{1.8 \times 10^{19}}{1.25 \times 10^9} = 1.47 \times 10^{10} \text{ sec} \approx 467 \text{ years at 10 Gbps}$$

### Counter Wrap Detection

If $C_{t_2} < C_{t_1}$ (wrap occurred):

$$\Delta C = (2^{32} - C_{t_1}) + C_{t_2}$$

For Counter64: $\Delta C = (2^{64} - C_{t_1}) + C_{t_2}$

---

## 4. Trap Storms — Queuing Analysis

### The Problem

Network events can trigger bursts of traps from many devices simultaneously (e.g., power outage affects 500 switches).

### Trap Rate Model

During a failure event affecting $N$ devices, each generating $T$ traps:

$$R_{storm} = N \times T \times \frac{1}{\Delta t}$$

Where $\Delta t$ = time window of the event.

| Devices | Traps/Device | Event Duration | Trap Rate |
|:---:|:---:|:---:|:---:|
| 100 | 5 | 10 sec | 50/sec |
| 500 | 10 | 5 sec | 1,000/sec |
| 1,000 | 20 | 2 sec | 10,000/sec |

### Trap Receiver Capacity

$$C_{receiver} = \frac{1}{T_{process}} \times N_{threads}$$

If processing takes 1 ms per trap with 10 threads: $C = 10,000$ traps/sec.

### Queue Overflow

When $R_{storm} > C_{receiver}$:

$$\text{Queue growth rate} = R_{storm} - C_{receiver}$$

$$T_{overflow} = \frac{Q_{max}}{R_{storm} - C_{receiver}}$$

With a 10,000-trap queue and $R = 20,000$/sec overflow of $C = 10,000$/sec:

$$T_{overflow} = \frac{10,000}{10,000} = 1 \text{ second}$$

After overflow, traps (UDP) are silently dropped — exactly when monitoring is most needed.

---

## 5. SNMPv3 Security — Crypto Overhead

### Authentication and Privacy

| Security Level | Operations | Overhead per PDU |
|:---|:---|:---:|
| noAuthNoPriv | None | 0 |
| authNoPriv | HMAC-SHA-1/256 | ~30 bytes + ~0.01 ms |
| authPriv | HMAC + AES-128-CBC | ~50 bytes + ~0.02 ms |

### Time Window Security

SNMPv3 uses time synchronization to prevent replay attacks:

$$|T_{sender} - T_{receiver}| \leq 150 \text{ seconds}$$

Messages outside this window are rejected. This requires clock synchronization within 2.5 minutes.

### USM Key Derivation

SNMP passwords are localized to each engine using:

$$K_{localized} = H(K_{master} \| \text{EngineID} \| K_{master})$$

Each device gets a unique key derived from the master password. Compromise of one device doesn't reveal the master password.

---

## 6. MIB Walk Efficiency

### GETNEXT vs GETBULK

GETNEXT retrieves one OID per request: $N$ objects = $N$ round trips.

GETBULK (SNMPv2c/v3) retrieves $M$ objects per request:

$$\text{Round trips} = \lceil \frac{N}{M} \rceil$$

| Objects | GETNEXT RTTs | GETBULK (M=50) RTTs | Speedup |
|:---:|:---:|:---:|:---:|
| 50 | 50 | 1 | 50x |
| 200 | 200 | 4 | 50x |
| 1,000 | 1,000 | 20 | 50x |
| 10,000 | 10,000 | 200 | 50x |

At 5 ms per RTT:
- GETNEXT for 10,000 objects: 50 seconds
- GETBULK: 1 second

---

## 7. Summary of Formulas

| Formula | Math Type | Application |
|:---|:---|:---|
| $180 \times O / T$ | Rate | Per-device poll bandwidth |
| $\Delta C / \Delta T$ | Derivative | Counter-to-rate conversion |
| $2^{32} / R$ | Division | Counter32 wrap time |
| $N \times T / \Delta t$ | Rate | Trap storm intensity |
| $Q_{max} / (R - C)$ | Queue theory | Time to overflow |
| $\lceil N/M \rceil$ | Ceiling division | GETBULK round trips |

## Prerequisites

- counter arithmetic, polling intervals, integer overflow

---

*SNMP has monitored billions of network interfaces for three decades. The protocol's math — polling intervals, counter wraps, and trap storm capacity — determines whether your monitoring system catches a network outage or becomes a victim of it.*
