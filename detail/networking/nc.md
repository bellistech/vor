# nc (netcat) Deep Dive — Theory & Internals

> *Netcat is the "Swiss Army knife" of networking — a raw TCP/UDP socket tool. Understanding its internals means understanding socket state machines, port scanning timing, file transfer throughput, and the security implications of arbitrary network I/O.*

---

## 1. Socket Connection — State Machine

### TCP Connection via nc

nc follows the standard TCP state machine:

$$\text{CLOSED} \xrightarrow{SYN} \text{SYN\_SENT} \xrightarrow{SYN-ACK} \text{ESTABLISHED}$$

### Connection Timing

$$T_{connect} = \begin{cases} 1 \times RTT & \text{(SYN → SYN-ACK, successful)} \\ T_{timeout} & \text{(no response, default 5-30 sec)} \end{cases}$$

### Connection States nc Can Test

| Scenario | nc Behavior | Time |
|:---|:---|:---:|
| Port open | Immediate connect | ~1 RTT |
| Port closed | RST received | ~1 RTT |
| Port filtered (firewall DROP) | Timeout | $T_{timeout}$ |
| Port filtered (firewall REJECT) | ICMP unreachable | ~1 RTT |

---

## 2. Port Scanning — Timing Analysis

### Sequential Scan (`-z` flag)

$$T_{scan} = \sum_{i=1}^{N} T_{probe,i}$$

For open/closed ports: $T_{probe} \approx RTT$.
For filtered ports: $T_{probe} = T_{timeout}$.

### Worst Case (All Filtered)

$$T_{worst} = N_{ports} \times T_{timeout}$$

| Ports | Timeout | Scan Time |
|:---:|:---:|:---:|
| 100 | 3 sec | 300 sec (5 min) |
| 1,000 | 3 sec | 3,000 sec (50 min) |
| 65,535 | 3 sec | 196,605 sec (54.6 hours) |

### Best Case (All Open/Closed)

$$T_{best} = N_{ports} \times RTT$$

| Ports | RTT | Scan Time |
|:---:|:---:|:---:|
| 100 | 5 ms | 0.5 sec |
| 1,000 | 5 ms | 5 sec |
| 65,535 | 5 ms | 328 sec (5.5 min) |

### Port Range Math

$$N_{ports} = P_{end} - P_{start} + 1$$

Common ranges:
- Well-known (0-1023): 1,024 ports
- Registered (1024-49151): 48,128 ports
- Dynamic (49152-65535): 16,384 ports
- All: 65,536 ports

---

## 3. Data Transfer — Throughput Analysis

### File Transfer via nc

Sender: `nc -l 1234 < file`
Receiver: `nc host 1234 > file`

### Throughput Model

$$T_{transfer} = \frac{S_{file}}{R_{effective}}$$

$$R_{effective} = \min(BW_{link}, BW_{TCP\_window}, R_{disk\_IO})$$

### Comparison with scp/rsync

| Method | Overhead | Encryption | Throughput (1G LAN) |
|:---|:---|:---:|:---:|
| nc (raw TCP) | None | No | ~940 Mbps |
| scp (SSH) | Encryption + compression | Yes | ~400-700 Mbps |
| rsync (SSH) | Checksumming + encryption | Yes | ~300-600 Mbps |

nc achieves near line-rate because it adds zero overhead — raw bytes from stdin to socket.

### UDP Transfer Considerations

With `nc -u`:
- No flow control — sender can overwhelm receiver
- No retransmission — lost data is gone
- No ordering guarantee

$$\text{Effective throughput} = R_{send} \times (1 - L_{loss})$$

---

## 4. Listener Mode — Server Socket

### Backlog Queue

When nc listens (`-l`), it creates a server socket with a connection backlog:

$$Q_{backlog} = \min(Q_{requested}, \text{somaxconn})$$

Default `somaxconn`: 128 (Linux), 128 (macOS).

### Concurrent Connections

Traditional nc: 1 connection at a time (exits after disconnect).
With `-k` (keep-open): accepts new connections after each closes (sequential, not parallel).

$$\text{Concurrent connections} = 1$$

For parallel: use `ncat --max-conns` or `socat fork`.

---

## 5. Proxy and Relay — Data Pipeline

### Named Pipe Relay

Using nc as a proxy:

```
mkfifo pipe
nc -l 8080 < pipe | nc target 80 > pipe
```

### Throughput Through Relay

$$R_{relay} = \min(R_{client \to proxy}, R_{proxy \to server})$$

### Latency Addition

$$\Delta T_{relay} = T_{read} + T_{copy} + T_{write} \approx 0.1 \text{ ms (userspace copy)}$$

For each relayed packet: ~100 microseconds of additional latency.

---

## 6. UDP Mode — Datagram Math

### Datagram Size

$$S_{max} = 65,535 - 8_{UDP} - 20_{IP} = 65,507 \text{ bytes}$$

In practice, limited to MTU to avoid fragmentation: $1500 - 28 = 1,472$ bytes.

### nc UDP Behavior

| Feature | TCP Mode | UDP Mode |
|:---|:---:|:---:|
| Connection establishment | 3-way handshake | None |
| Delivery guarantee | Yes | No |
| Detection of peer down | RST/FIN | No feedback |
| `nc -z` scan | SYN → response | Send → hope for ICMP |

### UDP Scan Reliability

UDP port scanning is unreliable:

$$P_{detect\_closed} = P_{ICMP\_unreachable\_received}$$

Many firewalls suppress ICMP, making open and filtered ports indistinguishable:

$$\text{Open port} \rightarrow \text{no response}$$
$$\text{Filtered port} \rightarrow \text{no response}$$

---

## 7. Bandwidth Testing with nc

### Quick Speed Test

Sender: `dd if=/dev/zero bs=1M count=1000 | nc host 1234`
Receiver: `nc -l 1234 > /dev/null`

### Measurement

$$BW = \frac{S_{transferred}}{T_{elapsed}}$$

| Data | Time | Bandwidth |
|:---:|:---:|:---:|
| 1 GB | 8.5 sec | 941 Mbps |
| 1 GB | 85 sec | 94 Mbps |
| 1 GB | 850 sec | 9.4 Mbps |

Advantages over iperf: no installation needed, uses any available nc.
Disadvantages: no parallel streams, no UDP bandwidth control, no statistics.

---

## 8. Summary of Formulas

| Formula | Math Type | Application |
|:---|:---|:---|
| $N \times T_{timeout}$ | Product | Worst-case scan time |
| $N \times RTT$ | Product | Best-case scan time |
| $S / R_{effective}$ | Division | File transfer time |
| $R \times (1 - L)$ | Product | Effective UDP throughput |
| $\min(R_{in}, R_{out})$ | Minimum | Relay throughput |
| $P_{end} - P_{start} + 1$ | Range | Port count |

---

*Netcat's power is its simplicity — it's a raw socket with stdin/stdout. Every other network tool adds abstraction layers; nc strips them away, giving you direct access to the TCP/UDP byte stream. When you need to debug at the socket level, nc is the scalpel.*
