# The Mathematics of QUIC — 0-RTT Handshakes, Stream Multiplexing, and Loss Detection

> *QUIC merges transport and cryptography into a single protocol, eliminating the round-trip costs of layered TCP+TLS. The math covers handshake latency savings, stream concurrency modeling, congestion control without head-of-line blocking, and the RACK loss detection algorithm.*

---

## 1. Handshake Latency — Round Trip Savings

### The Problem

TCP+TLS 1.3 requires multiple round trips before application data can flow. QUIC eliminates this overhead.

### Comparison

| Protocol Stack | Round Trips to First Data | Formula |
|:---|:---:|:---|
| TCP + TLS 1.2 | 3 RTT | 1 (SYN) + 2 (TLS) |
| TCP + TLS 1.3 | 2 RTT | 1 (SYN) + 1 (TLS) |
| QUIC (initial) | 1 RTT | Crypto + transport merged |
| QUIC (0-RTT) | 0 RTT | Resumed with cached keys |

### Time Savings

$$\Delta T = (RTT_{TCP+TLS} - RTT_{QUIC}) \times RTT$$

| RTT | TCP+TLS 1.3 | QUIC 1-RTT | QUIC 0-RTT | Savings (0-RTT) |
|:---:|:---:|:---:|:---:|:---:|
| 10 ms | 20 ms | 10 ms | 0 ms | 20 ms |
| 50 ms | 100 ms | 50 ms | 0 ms | 100 ms |
| 100 ms | 200 ms | 100 ms | 0 ms | 200 ms |
| 200 ms (satellite) | 400 ms | 200 ms | 0 ms | 400 ms |

### 0-RTT Security Tradeoff

0-RTT data is replayable (no anti-replay protection from the handshake). The risk:

$$P_{replay} = P_{attacker\_captures} \times P_{server\_accepts\_duplicate}$$

Mitigation: servers must ensure 0-RTT data is idempotent, or use single-use session tickets.

---

## 2. Stream Multiplexing — Eliminating Head-of-Line Blocking

### TCP's Problem

TCP delivers bytes in order. If packet $k$ is lost, packets $k+1, k+2, \ldots$ are delayed even if they're for different HTTP resources.

**Head-of-line blocking delay:**

$$D_{HOL} = T_{retransmit}(k) = RTO \text{ or } RTT \text{ (fast retransmit)}$$

This delay affects ALL streams multiplexed over the connection.

### QUIC's Solution

Each QUIC stream has independent ordering. Loss in stream $i$ only blocks stream $i$:

$$D_{HOL,QUIC} = T_{retransmit}(k) \times P_{stream\_affected}$$

$$P_{stream\_affected} = \frac{1}{S} \quad \text{(if loss is uniformly distributed across } S \text{ streams)}$$

### Worked Examples

With 10 concurrent streams and 1% packet loss:

| Metric | TCP (HTTP/2) | QUIC (HTTP/3) |
|:---|:---:|:---:|
| Loss rate | 1% | 1% |
| Streams affected by any loss | All 10 | ~1 stream |
| HOL blocking probability | $1 - (1-0.01)^{10} = 9.6\%$ | 1% per stream |
| Avg completion delay | $+1 \times RTT$ (10% of time) | $+1 \times RTT$ (1% of time, per stream) |

### Stream Concurrency Limits

QUIC negotiates max streams per connection:

$$\text{Total active streams} \leq \min(S_{client}, S_{server})$$

Default: typically 100 bidirectional + 100 unidirectional streams.

**Stream ID space (62-bit):**

$$S_{max} = 2^{62} = 4,611,686,018,427,387,903$$

Stream IDs are never reused within a connection (monotonically increasing).

---

## 3. Connection Migration — ID-Based Continuity

### The Problem

TCP connections are identified by the 4-tuple (src IP, dst IP, src port, dst port). Changing networks (WiFi → cellular) breaks the connection.

### QUIC Connection IDs

QUIC uses a connection ID (variable length, up to 20 bytes) independent of IP/port:

$$\text{ID space} = 2^{160} \quad \text{(at 20 bytes)}$$

**Migration cost:**

$$T_{migration} = T_{path\_validation} = 1 \times RTT$$

Compare with TCP reconnection: $T_{TCP+TLS} = 2 \times RTT + T_{app\_state\_restore}$.

---

## 4. Congestion Control — CUBIC and BBR in QUIC

### Same Algorithms, Better Information

QUIC carries acknowledgment information in a richer format than TCP:

- **ACK ranges** (not just cumulative ACK): precisely identifies received packets
- **ACK delay** field: explicit measurement of receiver-side delay
- **Packet number** never wraps (62-bit), never ambiguous

### RTT Estimation (Improved)

$$SRTT = (1 - \alpha) \times SRTT + \alpha \times (T_{ack} - T_{sent} - \text{ack\_delay})$$

The explicit `ack_delay` eliminates the ambiguity in TCP's RTT measurement.

### Pacing

QUIC implementations typically pace packet sending:

$$\text{Inter-packet gap} = \frac{MSS}{\text{cwnd} / SRTT} = \frac{MSS \times SRTT}{\text{cwnd}}$$

| cwnd | SRTT | MSS | Inter-packet gap |
|:---:|:---:|:---:|:---:|
| 100 KB | 50 ms | 1,200 B | 0.6 ms |
| 1 MB | 50 ms | 1,200 B | 0.06 ms |
| 10 MB | 50 ms | 1,200 B | 0.006 ms |

---

## 5. Loss Detection — RACK Algorithm

### The Problem

Detecting packet loss quickly and accurately is critical for performance. QUIC uses RACK (Recent ACKnowledgment) as primary loss detection.

### RACK Principle

A packet is considered lost if a later-sent packet has been acknowledged and:

$$T_{now} - T_{sent}(P) > \max(RTT + \text{reorder\_window}, \text{min\_RTT} \times \frac{5}{4})$$

### Reorder Window

$$\text{reorder\_window} = \max(\text{min\_rtt} / 4, \text{timer\_granularity})$$

This adapts to network reordering — networks with more reordering get a larger window before declaring loss.

### PTO (Probe Timeout) — Replacing TCP's RTO

$$PTO = SRTT + \max(4 \times RTTVAR, \text{granularity}) + \text{max\_ack\_delay}$$

PTO triggers a probe packet (not a full retransmit), which is less aggressive than TCP's RTO:

| TCP RTO Action | QUIC PTO Action |
|:---|:---|
| Retransmit oldest unacked | Send 1-2 probe packets |
| Reset cwnd to 1 MSS | Do NOT reset cwnd |
| Exponential backoff | Exponential backoff (same) |

---

## 6. Packet Number Space — Avoiding Ambiguity

### TCP's Ambiguity Problem

TCP reuses sequence numbers for retransmissions. When an ACK arrives, it's ambiguous:

$$\text{ACK for seq } N = \begin{cases} \text{original packet?} \\ \text{retransmission?} \end{cases}$$

This corrupts RTT estimation (Karn's algorithm disables sampling on retransmits).

### QUIC's Solution

Every packet gets a unique, monotonically increasing 62-bit packet number:

$$\text{Packet numbers: } 0, 1, 2, \ldots, 2^{62} - 1$$

At 10 million packets/sec:

$$T_{exhaust} = \frac{2^{62}}{10^7} = 4.6 \times 10^{11} \text{ sec} \approx 14,600 \text{ years}$$

No ambiguity ever. RTT estimation is always correct.

---

## 7. QUIC Overhead vs TCP+TLS

### Per-Packet Overhead

| Component | TCP+TLS 1.3 | QUIC |
|:---|:---:|:---:|
| Transport header | 20-60 B (TCP) | 1-17 B (short header) |
| Crypto framing | 5 B (TLS record) + 8-16 B (AEAD tag) | 16 B (AEAD tag) |
| UDP header | N/A | 8 B |
| **Total min** | **33 B** | **25 B** |
| **Total typical** | **41 B** | **30 B** |

QUIC's short header (after handshake) is more compact than TCP + TLS:

$$\text{Savings} = 41 - 30 = 11 \text{ bytes/packet}$$

At 1 million packets/sec: $11 \times 10^6 = 11$ MB/s saved in header overhead.

---

## 8. Summary of Formulas

| Formula | Math Type | Application |
|:---|:---|:---|
| $\Delta T = (R_{TCP} - R_{QUIC}) \times RTT$ | Linear | Handshake time savings |
| $P_{HOL} = 1/S$ | Probability | Per-stream HOL blocking |
| $MSS \times SRTT / \text{cwnd}$ | Rate control | Packet pacing interval |
| $RTT + \text{reorder\_window}$ | Adaptive threshold | RACK loss detection |
| $SRTT + 4 \times RTTVAR + \text{max\_ack\_delay}$ | Summation | PTO timeout |
| $2^{62}$ packet numbers | Exponent | Unique packet identification |
| $2^{160}$ connection IDs | Exponent | Connection ID space |

---

*QUIC is what TCP would look like if redesigned with 40 years of internet experience. Every design decision — merged handshakes, independent streams, monotonic packet numbers — is a mathematical optimization against the latency and correctness problems that TCP accumulated over decades.*
