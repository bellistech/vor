# SCTP (Stream Control Transmission Protocol)

Message-oriented, multi-homed, multi-streamed transport protocol (RFC 9260) providing reliable delivery with built-in resistance to SYN flood attacks via a four-way handshake with cookie mechanism.

## Packet Format

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|           Source Port         |       Destination Port        |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                    Verification Tag                           |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                         Checksum (CRC-32c)                    |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                         Chunk 1                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                         ...                                   |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                         Chunk N                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

- **Source/Destination Port**: 16-bit port numbers (same range as TCP/UDP)
- **Verification Tag**: 32-bit value set during association setup; must match in every packet (anti-spoofing)
- **Checksum**: CRC-32c over entire packet (stronger than TCP's ones-complement checksum)
- **Chunks**: One or more self-describing TLV chunks bundled in a single packet

### Chunk Header

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|   Chunk Type  |  Chunk Flags  |         Chunk Length          |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                    Chunk Value (variable)                      |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

## Chunk Types

```
Type  Name            Description
──────────────────────────────────────────────────────────────────
0     DATA            Carries user data with TSN, stream ID, sequence number
1     INIT            Initiate an association
2     INIT-ACK        Acknowledge INIT, return cookie
3     SACK            Selective Acknowledgment of received DATA chunks
4     HEARTBEAT       Probe reachability of a peer address
5     HEARTBEAT-ACK   Respond to HEARTBEAT
6     ABORT           Abort the association immediately
7     SHUTDOWN        Initiate graceful shutdown
8     SHUTDOWN-ACK    Acknowledge SHUTDOWN
9     ERROR           Report error conditions
10    COOKIE-ECHO     Return the cookie from INIT-ACK to complete handshake
11    COOKIE-ACK      Acknowledge COOKIE-ECHO, association established
14    SHUTDOWN-COMPLETE Final step of graceful shutdown
192   FORWARD-TSN     Advance cumulative TSN for PR-SCTP (RFC 3758)
```

## Association Lifecycle

### Terminology

```
TCP term            SCTP equivalent
──────────────────────────────────────────────
Connection          Association
Socket pair         Association (multi-homed: multiple IP pairs)
Stream              Stream (multiple independent streams per association)
Segment             Chunk (multiple chunks per packet)
Sequence number     Transmission Sequence Number (TSN)
```

### Four-Way Handshake (Open)

```
Client (Initiator)                          Server (Responder)
  |                                           |
  |  ---- INIT (init-tag=A, OS, MIS) ------>  |   Propose association parameters
  |                                           |
  |  <--- INIT-ACK (init-tag=B, cookie) ----  |   Server creates NO state, returns
  |                                           |   signed cookie with all parameters
  |                                           |
  |  ---- COOKIE-ECHO (cookie) ------------>  |   Client echoes cookie back
  |                                           |
  |  <--- COOKIE-ACK ----------------------  |   Server validates cookie, creates
  |                                           |   state, association ESTABLISHED
  |                                           |
```

- Server allocates NO resources until COOKIE-ECHO is validated
- Cookie contains a MAC (HMAC) binding client/server addresses, ports, and init tags
- Prevents SYN flood: attacker must receive and replay the cookie (requires real IP)

### Graceful Shutdown (Three-Way)

```
Initiator                               Responder
  |                                       |
  |  ---- SHUTDOWN ------------------>   |   Stop accepting new data
  |                                       |
  |  <--- SHUTDOWN-ACK ---------------   |   All data delivered
  |                                       |
  |  ---- SHUTDOWN-COMPLETE --------->   |   Association terminated
  |                                       |
```

### Association States

```
State               Description
──────────────────────────────────────────────────────────────
CLOSED              No association exists
COOKIE-WAIT         INIT sent, awaiting INIT-ACK
COOKIE-ECHOED       COOKIE-ECHO sent, awaiting COOKIE-ACK
ESTABLISHED         Association open, data transfer in progress
SHUTDOWN-PENDING    Application requested shutdown, draining outbound
SHUTDOWN-SENT       SHUTDOWN sent, awaiting SHUTDOWN-ACK
SHUTDOWN-RECEIVED   SHUTDOWN received, draining inbound
SHUTDOWN-ACK-SENT   SHUTDOWN-ACK sent, awaiting SHUTDOWN-COMPLETE
```

## Multi-Homing

```
# Each endpoint can bind to multiple IP addresses
# One primary path, one or more alternate paths

Endpoint A                              Endpoint B
  10.0.1.1 -----(primary path)-------- 10.0.2.1
  10.0.3.1 -----(alternate path)------ 10.0.4.1

# HEARTBEAT chunks probe alternate paths for reachability
# Failover occurs when primary path fails (configurable thresholds)
```

### Path Management Parameters

```bash
# Path.Max.Retrans (PMR) — max retransmissions before declaring path inactive
#   Default: 5
# Association.Max.Retrans (AMR) — max total retransmissions before abort
#   Default: 10
# HB.interval — heartbeat interval for probing alternate paths
#   Default: 30 seconds
# RTO.Initial — initial retransmission timeout
#   Default: 3 seconds
# RTO.Min / RTO.Max — bounds for computed RTO
#   Default: 1 second / 60 seconds
```

### Failover Behavior

```
Primary path (10.0.1.1 -> 10.0.2.1):
  Retransmit 1 ... Retransmit 5 (PMR exceeded)
  -> Mark primary path INACTIVE
  -> Switch to alternate path (10.0.3.1 -> 10.0.4.1)
  -> Continue transmitting on alternate path
  -> HEARTBEAT continues probing primary for recovery
```

## Multi-Streaming

```
# Multiple independent streams within one association
# Each stream has its own sequence numbering (SSN)
# Head-of-line blocking is limited to within a single stream

Association
  Stream 0: MSG-A1 -> MSG-A2 -> MSG-A3   (ordered within stream)
  Stream 1: MSG-B1 -> MSG-B2 -> MSG-B3   (independent ordering)
  Stream 2: MSG-C1 -> MSG-C2 -> MSG-C3   (loss in stream 2 does NOT
                                            block streams 0 or 1)
```

### Ordered vs Unordered Delivery

```
# Ordered (default): DATA chunks delivered in SSN order per stream
# Unordered: U-bit set in DATA chunk flags, delivered immediately
#            upon arrival regardless of SSN

# Unordered delivery is useful for:
# - Time-sensitive data where old messages are irrelevant
# - Independent messages that don't need sequencing
# - Emulating UDP-like behavior with reliability
```

## Partial Reliability (PR-SCTP)

```
# RFC 3758 — allows sender to abandon retransmission of DATA chunks
# FORWARD-TSN chunk advances receiver's cumulative TSN, skipping gaps

Policies:
  Timed Reliability    — abandon chunk after a time limit expires
  Limited Retransmission — abandon after N retransmission attempts
  Priority-based       — abandon lower-priority data under congestion

# Useful for real-time applications where stale data is worthless
# WebRTC data channels use PR-SCTP for unreliable/unordered messaging
```

## Congestion Control

```
# SCTP maintains independent congestion state PER PATH:
#   cwnd      — congestion window (bytes)
#   ssthresh  — slow start threshold
#   RTO       — retransmission timeout
#   srtt      — smoothed round-trip time
#   rttvar    — round-trip time variation

# Algorithms mirror TCP: slow start, congestion avoidance, fast retransmit
# Each path independently adjusts cwnd based on its own ACK clock
# Failover to alternate path starts with slow start (cwnd = MTU)
```

## SCTP over UDP Encapsulation

```
# RFC 6951 — encapsulate SCTP in UDP for NAT traversal
# Required because most NATs don't understand SCTP natively

UDP Header (src, dst port 9899 default)
  +-- SCTP Common Header
        +-- SCTP Chunks

# WebRTC uses this: SCTP-over-DTLS-over-UDP
# Linux: kernel support since 5.11
# Configure encapsulation port:
#   sysctl -w net.sctp.encap_port=9899
#   sysctl -w net.sctp.udp_port=9899
```

## SCTP vs TCP vs UDP

```
Feature              SCTP              TCP              UDP
─────────────────────────────────────────────────────────────────
Delivery             Reliable          Reliable         Unreliable
Message boundary     Preserved         Byte stream      Preserved
Multi-homing         Yes               No (MPTCP ext)   No
Multi-streaming      Yes               No               N/A
Handshake            4-way (cookie)    3-way (SYN)      None
DoS resistance       Cookie mech       SYN cookies      None
Ordered delivery     Per-stream        Global           None
Unordered mode       Yes               No               Inherent
Partial reliability  PR-SCTP ext       No               Inherent
Head-of-line block   Per-stream only   Entire conn      None
NAT traversal        Poor (UDP encap)  Good             Good
```

## Kernel Module and Tools

### Installation

```bash
# Debian/Ubuntu
apt install lksctp-tools libsctp-dev

# RHEL/CentOS/Fedora
dnf install lksctp-tools lksctp-tools-devel

# Load kernel module
modprobe sctp

# Verify module loaded
lsmod | grep sctp
```

### sctp_darn (SCTP diagnostic and relay tool)

```bash
# Start an SCTP server on port 5000
sctp_darn -H 0.0.0.0 -P 5000 -l

# Connect as client
sctp_darn -H 0.0.0.0 -P 5001 -h 10.0.0.1 -p 5000 -s

# Bind to multiple local addresses (multi-homing)
sctp_darn -H 10.0.1.1 -H 10.0.3.1 -P 5000 -l
```

### sctp_test (throughput testing)

```bash
# Server mode
sctp_test -H 0.0.0.0 -P 5000 -l

# Client — send 1000 messages of 1024 bytes
sctp_test -H 0.0.0.0 -P 5001 -h 10.0.0.1 -p 5000 -s -c 1000 -z 1024
```

### sysctl Tuning

```bash
# View all SCTP sysctls
sysctl -a | grep sctp

# Key parameters
sysctl net.sctp.rto_initial       # Initial RTO (ms), default 3000
sysctl net.sctp.rto_min           # Minimum RTO (ms), default 1000
sysctl net.sctp.rto_max           # Maximum RTO (ms), default 60000
sysctl net.sctp.max_init_retransmits  # INIT retransmits, default 8
sysctl net.sctp.path_max_retrans  # Per-path max retransmits, default 5
sysctl net.sctp.association_max_retrans  # Per-association max, default 10
sysctl net.sctp.hb_interval       # Heartbeat interval (ms), default 30000
sysctl net.sctp.max_burst         # Max burst size, default 4
```

### Socket Programming (C)

```c
#include <netinet/sctp.h>

// One-to-one style (like TCP)
int sd = socket(AF_INET, SOCK_STREAM, IPPROTO_SCTP);

// One-to-many style (multiple associations on one socket)
int sd = socket(AF_INET, SOCK_SEQPACKET, IPPROTO_SCTP);

// Bind to multiple addresses
sctp_bindx(sd, addrs, num_addrs, SCTP_BINDX_ADD_ADDR);

// Send with stream/protocol info
struct sctp_sndrcvinfo sinfo = { .sinfo_stream = 3 };
sctp_send(sd, buf, len, &sinfo, 0);

// Receive with stream info
sctp_recvmsg(sd, buf, len, NULL, NULL, &sinfo, &flags);
```

## Use Cases

```
Protocol/Application       Why SCTP
─────────────────────────────────────────────────────────────────
Diameter (RFC 6733)         Multi-homing for AAA server failover
SS7/SIGTRAN (M2UA, M3UA)   Telecom signaling transport over IP
5G/LTE S1-AP               eNodeB-to-MME control plane
5G/LTE X2-AP               Inter-eNodeB handover signaling
WebRTC data channels        PR-SCTP over DTLS for browser-to-browser
SIP (optional transport)    Message boundaries, multi-homing
```

## Tips

- SCTP associations are not visible to most NATs, firewalls, and load balancers. If deploying across NAT, use SCTP-over-UDP encapsulation (RFC 6951) or ensure your network gear has SCTP awareness.
- The Verification Tag in every packet provides a lightweight anti-spoofing mechanism beyond what TCP offers. An attacker must know the tag to inject packets into an existing association.
- Multi-homing failover is not instant. The endpoint must exhaust Path.Max.Retrans attempts on the primary path before switching. Tune `path_max_retrans` and `rto_max` down for faster failover at the cost of more false positives.
- Unlike TCP, SCTP preserves message boundaries. Each `sctp_send()` maps to exactly one DATA chunk delivered as a complete message. No application-layer framing needed.
- Unordered delivery (`SCTP_UNORDERED` flag) bypasses the per-stream reordering queue entirely. Combine with PR-SCTP for a reliable-UDP hybrid.
- WebRTC data channels use SCTP tunneled inside DTLS inside UDP. The browser negotiates reliability and ordering per data channel using PR-SCTP parameters.
- On Linux, `ss -S` shows SCTP association statistics. Use `ss -a --sctp` to list all SCTP sockets.
- The four-way handshake adds one extra round trip compared to TCP. For latency-sensitive setups, keep associations long-lived rather than creating them frequently.
- SCTP uses CRC-32c for checksums rather than TCP's weak ones-complement sum. This catches more corruption but costs slightly more CPU.

## See Also

- tcp, udp, webrtc

## References

- [RFC 9260 -- Stream Control Transmission Protocol (SCTP)](https://www.rfc-editor.org/rfc/rfc9260)
- [RFC 3758 -- SCTP Partial Reliability Extension (PR-SCTP)](https://www.rfc-editor.org/rfc/rfc3758)
- [RFC 6951 -- UDP Encapsulation of SCTP Packets](https://www.rfc-editor.org/rfc/rfc6951)
- [RFC 6458 -- Sockets API Extensions for SCTP](https://www.rfc-editor.org/rfc/rfc6458)
- [RFC 6733 -- Diameter Base Protocol (uses SCTP)](https://www.rfc-editor.org/rfc/rfc6733)
- [RFC 4960 -- Stream Control Transmission Protocol (obsoleted by RFC 9260)](https://www.rfc-editor.org/rfc/rfc4960)
- [Linux Kernel -- SCTP Documentation](https://www.kernel.org/doc/html/latest/networking/sctp.html)
- [lksctp-tools -- Linux SCTP userspace tools](https://github.com/sctp/lksctp-tools)
