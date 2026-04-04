# TCP (Transmission Control Protocol)

Reliable, ordered, connection-oriented byte stream protocol over IP providing flow control, congestion control, and error recovery.

## Header Format

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|          Source Port          |       Destination Port        |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                        Sequence Number                       |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                    Acknowledgment Number                     |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| Offset| Rsvd |C|E|U|A|P|R|S|F|            Window             |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|           Checksum            |        Urgent Pointer         |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                    Options (variable)                         |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

- **Source/Destination Port**: 16-bit port numbers (0-65535)
- **Sequence Number**: Byte position of first data byte in this segment (32-bit, wraps)
- **Acknowledgment Number**: Next sequence number the sender expects to receive
- **Data Offset**: Header length in 32-bit words (min 5 = 20 bytes, max 15 = 60 bytes)
- **Flags**: CWR, ECE, URG, ACK, PSH, RST, SYN, FIN
- **Window Size**: Receive window in bytes (up to 65535, or larger with window scaling)
- **Checksum**: Covers header, data, and pseudo-header (src/dst IP, protocol, length)
- **Urgent Pointer**: Offset from sequence number to last urgent data byte

## Flags

```
SYN  — Synchronize sequence numbers (connection initiation)
ACK  — Acknowledgment number field is valid
FIN  — Sender has finished sending data (graceful close)
RST  — Reset the connection (abort)
PSH  — Push data to application immediately (no buffering)
URG  — Urgent pointer field is valid
ECE  — ECN-Echo: received CE-marked packet (congestion)
CWR  — Congestion Window Reduced: sender responded to ECE
```

## Connection Lifecycle

### Three-Way Handshake (Open)

```
Client                          Server
  |                               |
  |  ---- SYN (seq=x) -------->  |   Client picks initial sequence number
  |                               |
  |  <-- SYN-ACK (seq=y,ack=x+1) |   Server picks its own ISN, acks client
  |                               |
  |  ---- ACK (ack=y+1) ------->  |   Connection ESTABLISHED both sides
  |                               |
```

### Four-Way Teardown (Close)

```
Initiator                       Responder
  |                               |
  |  ---- FIN (seq=u) -------->  |   Initiator done sending
  |                               |
  |  <-- ACK (ack=u+1) --------  |   Responder acknowledges FIN
  |                               |
  |  <-- FIN (seq=v) ----------  |   Responder also done sending
  |                               |
  |  ---- ACK (ack=v+1) ------->  |   Initiator acknowledges, enters TIME_WAIT
  |                               |
```

### Connection States

```
State          Description
───────────────────────────────────────────────────────────
CLOSED         No connection exists
LISTEN         Server waiting for incoming SYN
SYN_SENT       Client sent SYN, awaiting SYN-ACK
SYN_RECEIVED   Server received SYN, sent SYN-ACK, awaiting ACK
ESTABLISHED    Connection open, data transfer in progress
FIN_WAIT_1     Initiator sent FIN, awaiting ACK
FIN_WAIT_2     Initiator received ACK of FIN, awaiting peer's FIN
CLOSE_WAIT     Received FIN, sent ACK, waiting for app to close
CLOSING        Both sides sent FIN simultaneously
LAST_ACK       Responder sent FIN, awaiting final ACK
TIME_WAIT      Initiator received final FIN+ACK, waiting 2*MSL before CLOSED
```

## Flow Control

```
# Window size advertises how many bytes the receiver can buffer
# Sliding window allows sender to have multiple unACKed segments in flight

Sender window:
  [already ACKed] [sent, not ACKed] [can send] [cannot send yet]
                  <--- flight size --><- avail ->
                  <---------- send window -------->

# Window scaling (RFC 7323) — negotiated in SYN
# Extends 16-bit window field by a shift count (0-14)
# Effective window = window_field << scale_factor
# Max window = 65535 << 14 = 1,073,725,440 bytes (~1 GB)
```

## Congestion Control

### Algorithms

```
Slow Start        — Exponential growth: cwnd doubles each RTT until ssthresh
Congestion Avoid  — Linear growth: cwnd += 1 MSS per RTT after ssthresh
Fast Retransmit   — Retransmit after 3 duplicate ACKs (don't wait for timeout)
Fast Recovery     — After fast retransmit, halve cwnd, skip slow start

# Major implementations
Reno     — Classic: slow start + congestion avoidance + fast retransmit/recovery
NewReno  — Improved Reno: better handling of multiple losses in one window
CUBIC    — Linux default since 2.6.19. BIC-like cubic function for cwnd growth.
           Aggressive on high-BDP paths, fair on low-BDP.
BBR      — Google's model-based CC. Estimates bottleneck bandwidth and RTprop.
           Better on lossy links; can cause bufferbloat in some scenarios.
           BBRv2 addresses fairness issues.
```

### Checking / Changing Algorithm

```bash
# View current congestion control algorithm
sysctl net.ipv4.tcp_congestion_control

# View available algorithms
sysctl net.ipv4.tcp_available_congestion_control

# Set algorithm (temporary)
sysctl -w net.ipv4.tcp_congestion_control=bbr

# Load algorithm module
modprobe tcp_bbr
```

## TCP Options

```
Option            Kind  Length  Negotiated in SYN
──────────────────────────────────────────────────
MSS               2     4      Yes — Maximum Segment Size (default 536 if not set)
Window Scale      3     3      Yes — shift count for window size (0-14)
SACK Permitted    4     2      Yes — enables selective acknowledgments
SACK              5     var    No  — carried in ACKs, lists received byte ranges
Timestamps        8     10     Yes — used for RTTM and PAWS
No-Operation      1     1      No  — padding between options
End of Options    0     1      No  — marks end of option list
```

## Linux Tuning

### Connection Handling

```bash
# SYN backlog — max pending connections in SYN_RECEIVED state
sysctl -w net.ipv4.tcp_max_syn_backlog=65535

# Listen backlog — max pending connections in accept queue
sysctl -w net.core.somaxconn=65535

# SYN cookies — protection against SYN flood attacks
sysctl -w net.ipv4.tcp_syncookies=1           # 1=enabled (default)

# TIME_WAIT duration — how long sockets stay in TIME_WAIT
sysctl -w net.ipv4.tcp_fin_timeout=30          # default 60 seconds
                                                # (controls FIN_WAIT_2, not TIME_WAIT directly)

# Reuse TIME_WAIT sockets for new outgoing connections
sysctl -w net.ipv4.tcp_tw_reuse=1              # safe for clients
```

### Keepalive

```bash
# Time before first keepalive probe (default 7200 = 2 hours)
sysctl -w net.ipv4.tcp_keepalive_time=600

# Interval between keepalive probes (default 75 seconds)
sysctl -w net.ipv4.tcp_keepalive_intvl=15

# Number of unACKed probes before declaring dead (default 9)
sysctl -w net.ipv4.tcp_keepalive_probes=5
```

### Performance

```bash
# Window scaling — must be on for windows > 64KB
sysctl -w net.ipv4.tcp_window_scaling=1        # default 1

# Selective ACK — allows receiver to report non-contiguous blocks
sysctl -w net.ipv4.tcp_sack=1                  # default 1

# Timestamps — needed for PAWS (Protection Against Wrapped Sequences)
sysctl -w net.ipv4.tcp_timestamps=1            # default 1

# Buffer sizes (min, default, max in bytes)
sysctl -w net.ipv4.tcp_rmem="4096 131072 16777216"    # receive buffer
sysctl -w net.ipv4.tcp_wmem="4096 65536 16777216"     # send buffer
sysctl -w net.core.rmem_max=16777216                   # global max receive
sysctl -w net.core.wmem_max=16777216                   # global max send

# TCP autotuning (enabled by default, uses rmem/wmem ranges)
sysctl net.ipv4.tcp_moderate_rcvbuf              # 1=enabled
```

## Monitoring

```bash
# Socket statistics (preferred over netstat)
ss -t                              # all TCP connections
ss -tl                             # listening TCP sockets
ss -tn                             # TCP connections, numeric (no DNS)
ss -ti                             # TCP connections with internal info (cwnd, rtt, etc.)
ss -s                              # summary statistics
ss state time-wait                 # filter by state
ss state established '( dport = 443 )'  # filter by destination port

# Legacy netstat
netstat -tn                        # TCP connections, numeric
netstat -tlnp                      # listening TCP with PID

# Kernel TCP stats
cat /proc/net/tcp                  # raw TCP socket table
cat /proc/net/netstat              # TCP extension counters

# Packet capture — TCP flags
tcpdump -i eth0 'tcp[tcpflags] & tcp-syn != 0'     # SYN packets
tcpdump -i eth0 'tcp[tcpflags] & tcp-rst != 0'     # RST packets
tcpdump -i eth0 'tcp[tcpflags] & (tcp-syn|tcp-fin) != 0'  # SYN or FIN
tcpdump -i eth0 port 443 -w capture.pcap           # capture to file

# Count connections by state
ss -tan | awk '{print $1}' | sort | uniq -c | sort -rn

# Count TIME_WAIT sockets
ss -tan state time-wait | wc -l
```

## Common Issues

### TIME_WAIT Accumulation

```bash
# Symptoms: "cannot assign requested address" on outbound connections
# Cause: many short-lived connections to same destination (port exhaustion)

# Check count
ss -tan state time-wait | wc -l

# Mitigations
sysctl -w net.ipv4.tcp_tw_reuse=1              # reuse TIME_WAIT for outbound
sysctl -w net.ipv4.ip_local_port_range="1024 65535"  # expand ephemeral range
# Use connection pooling / keep-alive at application level
```

### SYN Floods

```bash
# Symptoms: SYN_RECEIVED count spikes, legitimate connections time out
# Defense
sysctl -w net.ipv4.tcp_syncookies=1
sysctl -w net.ipv4.tcp_max_syn_backlog=65535
# Also consider iptables rate limiting:
# iptables -A INPUT -p tcp --syn -m limit --limit 50/s --limit-burst 100 -j ACCEPT
```

### Retransmissions

```bash
# Check retransmit stats
ss -ti                             # look for "retrans:" field
nstat -az TcpRetransSegs           # total retransmissions since boot

# High retransmissions indicate: packet loss, congestion, or MTU issues
```

### Window Zero (Receiver Not Consuming Data)

```bash
# Symptoms: sender stalls, ss shows "snd_wnd:0"
# Cause: application not reading from socket fast enough

# Check with tcpdump
tcpdump -i eth0 'tcp[14:2] = 0'   # window size = 0 (approximate)

# Fix: speed up application reads, increase SO_RCVBUF, profile application
```

### Connection Resets (RST)

```bash
# Common causes
# - Connecting to a closed port
# - Firewall injecting RST
# - Application crash / abort
# - Half-open connection (one side crashed, other still sends)
# - SO_LINGER with timeout 0 (abortive close)

# Debug
tcpdump -i eth0 'tcp[tcpflags] & tcp-rst != 0'   # capture all RSTs
```

## Tips

- TIME_WAIT lasts for 2 * MSL (Maximum Segment Lifetime), which is 60 seconds on Linux. This is not configurable. `tcp_fin_timeout` controls FIN_WAIT_2 duration, not TIME_WAIT.
- Never set `tcp_tw_recycle=1` (removed in kernel 4.12). It breaks connections behind NAT because it rejects timestamps from different hosts sharing one IP.
- SYN cookies bypass the SYN queue entirely, so `tcp_max_syn_backlog` is irrelevant when cookies activate. The tradeoff is that TCP options (MSS, window scale, SACK) are encoded in limited space, potentially reducing performance.
- BBR requires `tcp_timestamps=1` and benefits from pacing (`fq` qdisc): `tc qdisc replace dev eth0 root fq`.
- A RST in response to a SYN means the port is closed. A RST after ESTABLISHED usually means the remote application crashed or a firewall is injecting resets.
- `ss -ti` shows per-connection internals: cwnd, ssthresh, rtt, retransmits. Invaluable for diagnosing performance issues.
- When tuning buffer sizes, the kernel auto-tunes between the min and max values in `tcp_rmem`/`tcp_wmem`. Setting `SO_RCVBUF` or `SO_SNDBUF` in the application disables auto-tuning for that socket.
- MSS is typically MTU minus 40 bytes (20 IP + 20 TCP). On Ethernet with 1500 MTU, MSS = 1460. With timestamps (12 bytes), effective payload per segment drops to 1448.
- CLOSE_WAIT sockets indicate a bug in your application: the remote side closed the connection but your code never called `close()`. These will persist until the process exits.

## See Also

- udp, quic, ss, netstat, tcpdump, iptables

## References

- [RFC 9293 — Transmission Control Protocol (TCP)](https://www.rfc-editor.org/rfc/rfc9293)
- [RFC 7323 — TCP Extensions for High Performance (Timestamps, Window Scaling)](https://www.rfc-editor.org/rfc/rfc7323)
- [RFC 5681 — TCP Congestion Control](https://www.rfc-editor.org/rfc/rfc5681)
- [RFC 6298 — Computing TCP's Retransmission Timer](https://www.rfc-editor.org/rfc/rfc6298)
- [RFC 7413 — TCP Fast Open](https://www.rfc-editor.org/rfc/rfc7413)
- [RFC 8684 — TCP Extensions for Multipath Operation with Multiple Addresses (MPTCP)](https://www.rfc-editor.org/rfc/rfc8684)
- [Linux Kernel — TCP Sysctl Documentation](https://www.kernel.org/doc/html/latest/networking/ip-sysctl.html)
- [man tcp](https://man7.org/linux/man-pages/man7/tcp.7.html)
- [Cloudflare Blog — A Brief History of TCP Congestion Control](https://blog.cloudflare.com/an-introduction-to-bbr-and-its-implementation-in-quiche/)
