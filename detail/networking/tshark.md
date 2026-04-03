# tshark Deep Dive — Theory & Internals

> *tshark is Wireshark's command-line packet analyzer with the same protocol dissection engine — over 3,000 protocols decoded in depth. The math covers display filter optimization, conversation statistics, IO graph intervals, and the reassembly engine that reconstructs TCP streams from individual segments.*

---

## 1. Protocol Dissection — Decode Depth

### The Dissection Pipeline

$$\text{Raw frame} \xrightarrow{L2} \text{Ethernet} \xrightarrow{L3} \text{IP} \xrightarrow{L4} \text{TCP} \xrightarrow{L7} \text{HTTP/TLS/...}$$

### Dissection Cost per Packet

$$T_{dissect} = \sum_{l=1}^{D} T_{layer_l}$$

| Layer | Dissection Cost | Protocols |
|:---|:---:|:---|
| L2 (Ethernet) | ~0.5 us | Ethernet, 802.1Q, MPLS |
| L3 (Network) | ~1 us | IPv4, IPv6, ARP |
| L4 (Transport) | ~1 us | TCP, UDP, SCTP |
| L7 (Application) | ~2-50 us | HTTP, TLS, DNS, SMB |
| Reassembly | ~5-100 us | TCP stream, IP fragments |

### Packets per Second

$$PPS_{max} = \frac{1}{T_{dissect\_avg}}$$

| Decode Depth | Avg Time | Max PPS |
|:---|:---:|:---:|
| L2-L3 only | 2 us | 500,000 |
| Full stack (no reassembly) | 5 us | 200,000 |
| Full stack + TCP reassembly | 20 us | 50,000 |
| Full stack + application decode | 50 us | 20,000 |

---

## 2. Display Filters vs Capture Filters

### Two-Stage Filtering

**Capture filter (BPF):** Runs in kernel, decides what to capture.

$$\text{Complexity: } O(1) \text{ per packet (BPF bytecode)}$$

**Display filter:** Runs in userspace on captured data, full protocol awareness.

$$\text{Complexity: } O(D) \text{ per packet (protocol tree traversal)}$$

### When to Use Each

| Need | Capture Filter | Display Filter |
|:---|:---|:---|
| Reduce capture volume | `tcp port 443` | N/A (too late) |
| Protocol-specific | N/A (BPF can't decode L7) | `http.response.code == 200` |
| Field extraction | N/A | `tls.handshake.type == 1` |
| Post-capture analysis | N/A | `tcp.analysis.retransmission` |

### Display Filter Performance

$$T_{filter} = N_{packets} \times T_{evaluate}$$

| Packets | Simple Filter | Complex Filter (5 terms) |
|:---:|:---:|:---:|
| 10,000 | 0.1 sec | 0.5 sec |
| 100,000 | 1 sec | 5 sec |
| 1,000,000 | 10 sec | 50 sec |
| 10,000,000 | 100 sec | 500 sec |

---

## 3. Conversation Statistics — Flow Analysis

### `-z conv,tcp` Output

tshark computes per-conversation statistics:

$$\text{Conversation} = (\text{src IP}, \text{src port}, \text{dst IP}, \text{dst port})$$

### Metrics per Conversation

| Metric | Formula |
|:---|:---|
| Duration | $t_{last} - t_{first}$ |
| Bytes A→B | $\sum S_{packets,A \to B}$ |
| Bytes B→A | $\sum S_{packets,B \to A}$ |
| Throughput A→B | $\text{Bytes}_{A\to B} / \text{Duration}$ |
| Packets | $N_{A\to B} + N_{B\to A}$ |

### Top-N Conversations

tshark can sort by any column. The top-N by bytes:

$$\text{Top-N} = \text{sort}(\text{conversations}, \text{bytes}, \text{desc})[:N]$$

Identifying the top 10 conversations often reveals 80-90% of traffic (Pareto principle).

---

## 4. IO Statistics — Time-Series Analysis

### `-z io,stat,<interval>`

tshark bins packets into time intervals:

$$\text{Bin}_k = \{p : k \times I \leq t_p < (k+1) \times I\}$$

Where $I$ = interval in seconds.

### Metrics per Bin

$$\text{PPS}_k = \frac{|\text{Bin}_k|}{I}$$

$$\text{Throughput}_k = \frac{\sum_{p \in \text{Bin}_k} S_p}{I}$$

### Interval Selection

| Interval | Resolution | Bins per Hour | Use Case |
|:---:|:---|:---:|:---|
| 0.001 sec (1 ms) | Microsecond bursts | 3,600,000 | Microburst detection |
| 0.1 sec | Sub-second | 36,000 | Latency analysis |
| 1 sec | Per-second | 3,600 | Normal monitoring |
| 10 sec | Smoothed | 360 | Trend analysis |
| 60 sec | Per-minute | 60 | Capacity planning |

---

## 5. TCP Stream Reassembly

### The Problem

TCP segments may arrive out of order, be retransmitted, or overlap. tshark reassembles them into the application-layer stream.

### Reassembly Buffer

$$B_{stream} = \sum_{seg \in \text{stream}} S_{payload,seg}$$

### TCP Analysis Flags

tshark annotates TCP behavior:

| Flag | Meaning | Detection Logic |
|:---|:---|:---|
| `tcp.analysis.retransmission` | Resent segment | Same seq, different time |
| `tcp.analysis.fast_retransmission` | Resent after 3 dup-ACKs | Within RTT of dup-ACKs |
| `tcp.analysis.out_of_order` | Seq < expected | Seq precedes next expected |
| `tcp.analysis.duplicate_ack` | Same ACK repeated | ACK number unchanged |
| `tcp.analysis.window_full` | Receiver window exhausted | Bytes in flight >= window |
| `tcp.analysis.zero_window` | Receiver advertises 0 | Window size = 0 |

### Retransmission Rate from Capture

$$R_{retrans} = \frac{N_{retransmissions}}{N_{total\_segments}} \times 100\%$$

$$\text{tshark command: } \text{-z io,stat,0,"tcp.analysis.retransmission"}$$

---

## 6. Field Extraction — Structured Output

### `-T fields -e <field>` Performance

$$T_{extract} = N_{packets} \times (T_{dissect} + T_{field\_lookup})$$

### Common Extractions

| Field | Example Output | Size per Record |
|:---|:---|:---:|
| `frame.time` | `2024-01-15 10:30:45.123` | ~25 B |
| `ip.src` | `10.0.0.1` | ~12 B |
| `tcp.stream` | `42` | ~3 B |
| `http.host` | `www.example.com` | ~20 B |
| `tls.handshake.extensions_server_name` | `www.example.com` | ~20 B |

### Output Size Estimation

$$S_{output} = N_{matched} \times (F_{count} \times S_{field\_avg} + F_{count} \times S_{delimiter})$$

For 100,000 matching packets, 5 fields, ~15 bytes/field:

$$S = 100,000 \times (5 \times 15 + 5 \times 1) = 8 \text{ MB}$$

---

## 7. Expert Analysis — Automatic Problem Detection

### `-z expert`

tshark's expert system flags issues by severity:

| Severity | Examples | Action |
|:---|:---|:---|
| Error | Malformed packet, checksum error | Investigate immediately |
| Warning | Retransmission, out-of-order | Monitor for patterns |
| Note | TCP window update, keepalive | Informational |
| Chat | Connection setup/teardown | Normal operation |

### Expert Event Rate

$$R_{events} = \frac{N_{events}}{T_{capture}}$$

A "healthy" capture should have:
- Errors: < 0.01% of packets
- Warnings (retransmissions): < 1% of TCP segments
- Notes: varies by traffic type

---

## 8. Summary of Formulas

| Formula | Math Type | Application |
|:---|:---|:---|
| $\sum T_{layer}$ | Summation | Per-packet dissection cost |
| $N \times T_{evaluate}$ | Product | Display filter time |
| $\text{Bytes} / \text{Duration}$ | Rate | Per-conversation throughput |
| $|\text{Bin}_k| / I$ | Rate | IO statistics (PPS) |
| $N_{retrans} / N_{segments}$ | Ratio | Retransmission rate |
| $N \times F \times S_{field}$ | Product | Extract output size |

---

*tshark is the command-line equivalent of an X-ray machine for networks — it decodes over 3,000 protocols down to individual field values, computes conversation and IO statistics, and flags problems automatically. The same dissection engine that powers Wireshark, scriptable and pipeable.*
