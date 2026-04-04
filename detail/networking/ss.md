# ss Deep Dive — Theory & Internals

> *ss (socket statistics) reads socket information directly from kernel netlink, bypassing /proc/net parsing. Understanding its output means understanding TCP state machines, socket buffer tuning, congestion window visibility, and the kernel data structures that govern every network connection.*

---

## 1. Socket States — TCP State Machine

### The States ss Reports

$$\text{State} \in \{LISTEN, SYN\text{-}SENT, SYN\text{-}RECV, ESTABLISHED, FIN\text{-}WAIT\text{-}1, FIN\text{-}WAIT\text{-}2, CLOSE\text{-}WAIT, CLOSING, LAST\text{-}ACK, TIME\text{-}WAIT, CLOSED\}$$

### State Distribution on a Typical Server

| State | Meaning | Typical Count (web server) |
|:---|:---|:---:|
| LISTEN | Waiting for connections | 5-50 |
| ESTABLISHED | Active connections | 100-10,000 |
| TIME-WAIT | Connection recently closed | 100-50,000 |
| CLOSE-WAIT | Peer closed, app hasn't | 0 (>0 = app bug!) |
| SYN-RECV | Handshake in progress | 0-100 |

### State Count Formulas

$$N_{total} = N_{LISTEN} + N_{ESTAB} + N_{TIME\_WAIT} + N_{other}$$

### TIME-WAIT Accumulation

$$N_{TW} = R_{close} \times T_{TW}$$

Where $R_{close}$ = connection close rate/sec, $T_{TW} = 60$ sec (Linux, was 120 sec in older kernels).

| Close Rate | TIME-WAIT Duration | TIME-WAIT Count |
|:---:|:---:|:---:|
| 100/sec | 60 sec | 6,000 |
| 1,000/sec | 60 sec | 60,000 |
| 10,000/sec | 60 sec | 600,000 |

At 10,000 connections/sec, 600K TIME-WAIT sockets consume ~120 MB of kernel memory.

---

## 2. Buffer Sizing — Send/Receive Queue

### ss Output Fields

```
Recv-Q  Send-Q  Local Address:Port  Peer Address:Port
0       0       10.0.0.1:443        10.0.0.2:52431
```

### Interpreting Queue Depths

**For ESTABLISHED sockets:**

$$\text{Recv-Q} = \text{bytes received by kernel, not yet read by app}$$

$$\text{Send-Q} = \text{bytes sent by app, not yet ACKed by peer}$$

| Recv-Q | Send-Q | Interpretation |
|:---:|:---:|:---|
| 0 | 0 | Normal (application keeping up) |
| > 0 | 0 | Application slow to read (CPU-bound?) |
| 0 | > 0 | Network congestion or slow peer |
| > 0 | > 0 | Both application and network issues |

**For LISTEN sockets:**

$$\text{Recv-Q} = \text{current backlog (pending connections)}$$

$$\text{Send-Q} = \text{maximum backlog (listen queue size)}$$

### Backlog Overflow Detection

$$\text{Overflow} = \text{Recv-Q} \geq \text{Send-Q}$$

When the listen backlog is full, new SYN packets are dropped (or SYN cookies are used).

---

## 3. TCP Internal State (`-i` flag)

### The Extended Information

`ss -ti` reveals per-connection TCP internals:

| Field | Meaning | Formula |
|:---|:---|:---|
| `cwnd` | Congestion window (segments) | Controls send rate |
| `ssthresh` | Slow start threshold | AIMD reset point |
| `rtt` | Smoothed RTT | Jacobson/Karels EWMA |
| `rttvar` | RTT variance | Variance estimate |
| `retrans` | Retransmit count | Loss indicator |
| `bytes_sent` | Total bytes transmitted | Throughput calculation |
| `bytes_received` | Total bytes received | Throughput calculation |

### Current Send Rate Estimation

$$R_{send} = \frac{cwnd \times MSS}{RTT}$$

| cwnd (segments) | MSS | RTT | Estimated Rate |
|:---:|:---:|:---:|:---:|
| 10 | 1,460 B | 10 ms | 11.7 Mbps |
| 100 | 1,460 B | 10 ms | 117 Mbps |
| 100 | 1,460 B | 50 ms | 23.4 Mbps |
| 1,000 | 1,460 B | 50 ms | 234 Mbps |

### Loss Rate Estimation

$$p_{loss} \approx \frac{\text{retrans}}{\text{data\_segs\_out}}$$

---

## 4. Socket Memory (`-m` flag)

### Memory Fields

`ss -m` shows kernel memory allocation per socket:

| Field | Meaning |
|:---|:---|
| `skmem:(r<rmem>,rb<rcvbuf>,t<wmem>,tb<sndbuf>,f<fwd>,w<cwnd_mem>,o<opt>,bl<backlog>)` | |
| `r` | Receive queue bytes |
| `rb` | Receive buffer max |
| `t` | Send queue bytes |
| `tb` | Send buffer max |

### Buffer Auto-Tuning

Linux auto-tunes buffer sizes within:

$$\text{net.ipv4.tcp\_rmem} = [min, default, max]$$

Default: $[4096, 131072, 6291456]$ (4 KB, 128 KB, 6 MB).

### Memory per Connection

$$M_{connection} \approx rcvbuf + sndbuf + overhead$$

| Configuration | Per-Connection | 10K Connections | 100K Connections |
|:---|:---:|:---:|:---:|
| Default (128+128 KB) | ~260 KB | 2.6 GB | 26 GB |
| Tuned (32+32 KB) | ~68 KB | 680 MB | 6.8 GB |
| Minimal (4+4 KB) | ~12 KB | 120 MB | 1.2 GB |

---

## 5. Filtering — Kernel-Side vs User-Side

### ss Filter Syntax Performance

ss supports kernel-side filtering via netlink:

$$T_{kernel\_filter} = O(1) \text{ per socket (processed in kernel)}$$

$$T_{user\_filter} = O(N) \text{ (all sockets dumped, filtered in userspace)}$$

### Comparison with netstat

| Tool | Data Source | 10K sockets | 100K sockets |
|:---|:---|:---:|:---:|
| netstat | /proc/net/tcp (parse text) | ~1 sec | ~10 sec |
| ss (no filter) | Netlink (binary) | ~0.1 sec | ~1 sec |
| ss (kernel filter) | Netlink + filter | ~0.05 sec | ~0.2 sec |

### Filter Examples

```
ss state established
ss -tn dport = :443
ss -tn 'sport > :1024 and dport = :80'
```

Each reduces the result set in the kernel before transmission to userspace.

---

## 6. Connection Rate Analysis

### Counting New Connections

$$R_{new} = \frac{\Delta N_{ESTABLISHED}}{\Delta t}$$

### SYN Flood Detection

$$\text{SYN flood indicator} = N_{SYN\text{-}RECV} > \text{threshold}$$

Normal: $N_{SYN\text{-}RECV} < 10$. During SYN flood: $N_{SYN\text{-}RECV} = $ thousands.

### Connection Table Scaling

$$N_{max} = \frac{M_{available}}{M_{per\_connection}}$$

At 260 KB/connection and 16 GB available for sockets:

$$N_{max} = \frac{16 \times 10^9}{260 \times 10^3} \approx 61,500 \text{ connections}$$

With tuned buffers (68 KB/conn): ~235,000 connections.

---

## 7. Summary of Formulas

| Formula | Math Type | Application |
|:---|:---|:---|
| $R_{close} \times T_{TW}$ | Product | TIME-WAIT count |
| $cwnd \times MSS / RTT$ | Rate | Send rate estimation |
| $\text{retrans} / \text{segs\_out}$ | Ratio | Loss rate |
| $rcvbuf + sndbuf + overhead$ | Summation | Per-connection memory |
| $M_{avail} / M_{per\_conn}$ | Division | Max connections |
| Recv-Q >= Send-Q (LISTEN) | Comparison | Backlog overflow |

## Prerequisites

- socket state machines, queue arithmetic, TCP internals

---

*ss is the fastest way to understand what your kernel's network stack is doing right now. Every field maps to a kernel data structure, and the TCP extended info gives you live visibility into congestion windows, RTT estimates, and retransmission counts that no other tool exposes as efficiently.*
