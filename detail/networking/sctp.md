# SCTP Deep Dive -- Protocol Mechanics, Security, and Reliability

> *SCTP was designed from the ground up to solve TCP's shortcomings for signaling transport: no head-of-line blocking across streams, no SYN flood vulnerability, and native multi-homing for failover without application-layer complexity.*

---

## 1. Four-Way Handshake Security Analysis

### The TCP SYN Flood Problem

TCP's three-way handshake requires the server to allocate state (a Transmission Control Block) upon receiving a SYN, before the client has proven it can receive responses. An attacker spoofing source addresses can exhaust the server's SYN queue:

```
Attacker (spoofed src)        Server
  |                             |
  |  ---- SYN (src=random) -->  |  Server allocates TCB in SYN queue
  |                             |  SYN-ACK sent to spoofed address
  |                             |  TCB sits in queue until timeout
  |  ---- SYN (src=random) -->  |  Another TCB allocated
  |  ---- SYN (src=random) -->  |  Queue fills up
  |                             |  Legitimate SYNs dropped
```

TCP mitigates this with SYN cookies (encoding state in the ISN), but SYN cookies sacrifice TCP options negotiation and are a retrofit, not a design feature.

### SCTP's Cookie Mechanism

SCTP solves this architecturally. The server allocates zero state until the third message (COOKIE-ECHO):

```
Step 1: INIT
  Client -> Server
  Contains: Initiate Tag (I-Tag-A), number of outbound/inbound streams,
            initial TSN, receiver window (a-rwnd), address list

Step 2: INIT-ACK
  Server -> Client
  Server computes a State Cookie containing:
    - Both endpoints' addresses and ports
    - Both Initiate Tags
    - Both TSN values
    - Stream counts
    - Timestamp (for expiration)
    - HMAC signature over all fields using server's secret key

  Server stores NO state. The cookie is the state, carried by the client.

Step 3: COOKIE-ECHO
  Client -> Server
  Client echoes the cookie verbatim. Server:
    1. Verifies HMAC (proves cookie is authentic, not forged)
    2. Checks timestamp (proves cookie is not stale)
    3. Extracts association parameters from cookie
    4. NOW allocates association state (TCB equivalent)
    5. May bundle DATA chunks with COOKIE-ECHO

Step 4: COOKIE-ACK
  Server -> Client
  Association ESTABLISHED. May bundle DATA chunks.
```

### Why This Defeats SYN Floods

An attacker spoofing source addresses sends INIT packets. The server responds with INIT-ACK containing a cookie, but the attacker never receives it (it goes to the spoofed address). Without the cookie, the attacker cannot send COOKIE-ECHO, and the server has allocated nothing.

Key properties:

- **Statelessness until validation**: Server holds zero per-association state during the handshake. No queue to exhaust.
- **Cryptographic binding**: The HMAC prevents an attacker from forging cookies. The secret key is known only to the server and is rotated periodically.
- **Address validation**: The client must receive the INIT-ACK at its real address to extract the cookie. Spoofed-source attacks fail because the INIT-ACK goes to the wrong host.
- **Cookie lifetime**: A timestamp in the cookie prevents replay attacks. Expired cookies are rejected, limiting the window for any captured cookie.

### Cost of the Four-Way Handshake

The four-way handshake requires two round trips before data can flow (INIT -> INIT-ACK -> COOKIE-ECHO -> COOKIE-ACK), compared to TCP's 1.5 round trips (SYN -> SYN-ACK -> ACK). However, DATA chunks can be bundled with COOKIE-ECHO and COOKIE-ACK, partially offsetting the latency penalty.

---

## 2. TSN and SSN Numbering

SCTP uses two independent sequence number spaces:

### Transmission Sequence Number (TSN)

The TSN is a 32-bit number assigned to each DATA chunk at the association level (not per-stream). It serves the same role as TCP's sequence number but counts chunks rather than bytes.

```
Association-level TSN space:

  DATA chunk 1: TSN = 1000, Stream 0, SSN = 0
  DATA chunk 2: TSN = 1001, Stream 1, SSN = 0
  DATA chunk 3: TSN = 1002, Stream 0, SSN = 1
  DATA chunk 4: TSN = 1003, Stream 2, SSN = 0
  DATA chunk 5: TSN = 1004, Stream 1, SSN = 1

All five chunks share one TSN sequence, regardless of stream.
```

TSN properties:
- 32-bit unsigned, wraps at 2^32 with modular arithmetic (same rules as TCP)
- Cumulative TSN ACK: the receiver reports the highest TSN such that all TSNs up to and including it have been received
- Gap ACK blocks: the receiver reports additional TSN ranges received beyond the cumulative point (equivalent to TCP SACK blocks)
- Initial TSN is chosen randomly during INIT/INIT-ACK

### Stream Sequence Number (SSN)

The SSN is a 16-bit number assigned per-stream, per-direction. It governs in-order delivery within a stream.

```
Stream 0: SSN 0 -> SSN 1 -> SSN 2  (delivered in order)
Stream 1: SSN 0 -> SSN 1 -> SSN 2  (delivered in order, independent of stream 0)
```

SSN properties:
- 16-bit unsigned, wraps at 65536
- Only meaningful for ordered DATA chunks (unordered chunks have SSN ignored)
- Receiver holds back out-of-order SSN chunks within a stream until gaps are filled
- Different streams' SSN spaces are completely independent

### The Separation of Concerns

This two-level numbering is the key to avoiding head-of-line blocking:

- TSN handles reliability (retransmission, acknowledgment) at the association level
- SSN handles ordering at the stream level
- A lost DATA chunk on stream 2 blocks delivery of later chunks on stream 2 only
- Streams 0 and 1 continue delivering data even while stream 2 waits for retransmission

---

## 3. SACK Mechanism

SCTP's Selective Acknowledgment is built into the base protocol (unlike TCP where SACK is an optional extension). Every SACK chunk contains:

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|   Type = 3    |  Chunk Flags  |         Chunk Length          |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                 Cumulative TSN ACK                            |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                 Advertised Receiver Window Credit (a-rwnd)    |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|      Number of Gap ACK Blocks = N                             |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|      Number of Duplicate TSNs = D                             |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|   Gap ACK Block #1 Start      |   Gap ACK Block #1 End       |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                          ...                                  |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|   Gap ACK Block #N Start      |   Gap ACK Block #N End       |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                    Duplicate TSN #1                            |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                          ...                                  |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                    Duplicate TSN #D                            |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

### Fields

- **Cumulative TSN ACK**: Highest TSN such that all TSNs up to and including it have been received. This advances the sender's retransmission window.
- **a-rwnd**: Advertised receiver window credit in bytes. Sender must not have more than a-rwnd bytes of unacknowledged data outstanding (flow control).
- **Gap ACK Blocks**: Offsets relative to Cumulative TSN ACK. Block Start/End are 16-bit values, representing TSN ranges [Cumulative + Start, Cumulative + End] that have been received out of order.
- **Duplicate TSNs**: TSNs received more than once. Informs the sender that retransmissions were unnecessary (useful for adjusting retransmission strategy).

### SACK Processing Example

```
Sender transmits TSNs: 100, 101, 102, 103, 104, 105
Receiver gets: 100, 101, [102 lost], 103, 104, [105 lost]

SACK from receiver:
  Cumulative TSN ACK = 101
  Gap ACK Block #1: Start=2, End=3  (TSNs 103-104 received)
  a-rwnd = 48000

Sender knows:
  TSNs 100-101: delivered, can free buffers
  TSN 102: missing, needs retransmission
  TSNs 103-104: received out of order, do NOT retransmit
  TSN 105: missing, needs retransmission

After retransmitting 102 and 105:
  New SACK: Cumulative TSN ACK = 105, no gap blocks
```

### Differences from TCP SACK

- SCTP SACK is mandatory; TCP SACK is optional (negotiated via TCP option)
- SCTP gap blocks are relative offsets; TCP SACK blocks are absolute sequence numbers
- SCTP includes duplicate TSN reporting natively; TCP requires the D-SACK extension (RFC 2883)
- SCTP SACK is a standalone chunk type; TCP SACK is carried in options (limited to 40 bytes, usually 3-4 blocks max)

---

## 4. Head-of-Line Blocking Avoidance

### The TCP Problem

TCP provides a single byte stream. If byte 5000 is lost but bytes 5001-10000 arrive, the receiver must buffer bytes 5001-10000 and wait for the retransmission of byte 5000 before delivering anything to the application. This is head-of-line (HOL) blocking.

For multiplexed protocols (HTTP/2 over TCP, for example), a loss affecting one HTTP stream blocks all streams sharing the TCP connection.

### SCTP's Solution

SCTP separates reliability from ordering using the TSN/SSN split:

```
Time ->

Stream 0:  [SSN=0, TSN=100]  [SSN=1, TSN=103]  [SSN=2, TSN=106]
Stream 1:  [SSN=0, TSN=101]  [SSN=1, TSN=104]  [SSN=2, TSN=107]
Stream 2:  [SSN=0, TSN=102]  [SSN=1, TSN=105]  [SSN=2, TSN=108]

If TSN 104 (Stream 1, SSN=1) is lost:

  Stream 0: delivers SSN=0, SSN=1, SSN=2 normally    (no blocking)
  Stream 1: delivers SSN=0, BLOCKS on SSN=1           (waiting for retransmit)
  Stream 2: delivers SSN=0, SSN=1, SSN=2 normally    (no blocking)
```

SACK at the association level triggers retransmission of TSN 104. Once retransmitted and received, stream 1 resumes delivery. Streams 0 and 2 were never affected.

### Unordered Delivery Eliminates All HOL Blocking

Setting the U-bit in a DATA chunk bypasses the SSN reordering queue entirely. The receiver delivers the chunk to the application as soon as it arrives, regardless of order. Combined with PR-SCTP (which may abandon chunks), this provides UDP-like semantics with optional reliability.

---

## 5. Multi-Homing Failover Algorithm

### Path State Machine

Each remote transport address (IP) has an independent state:

```
        +-----------+
        |  ACTIVE   |<------ HEARTBEAT-ACK received
        +-----+-----+        or DATA-ACK on this path
              |
              | PMR consecutive
              | failures on this path
              v
        +-----------+
        | INACTIVE  |-------> HEARTBEAT probing continues
        +-----------+         at HB.interval
              |
              | HEARTBEAT-ACK received
              v
        +-----------+
        |  ACTIVE   |
        +-----------+
```

### Failure Detection

For the primary path:
1. Each DATA chunk retransmission increments a per-path error counter
2. When the counter exceeds `Path.Max.Retrans` (default 5), the path is marked INACTIVE
3. The sender selects the next ACTIVE alternate path as the new primary
4. Retransmissions of failed DATA are sent on the alternate path

For the association:
1. A separate counter tracks total errors across all paths
2. When it exceeds `Association.Max.Retrans` (default 10), the entire association is aborted

### HEARTBEAT Probing

HEARTBEAT chunks are sent to idle alternate paths:
- Interval: `HB.interval + jitter` (jitter prevents synchronization)
- The HEARTBEAT carries a timestamp for RTT measurement
- HEARTBEAT-ACK echoes the timestamp back
- A successful HEARTBEAT-ACK resets the path error counter and marks the path ACTIVE
- Alternate paths are always probed, even when the primary is healthy (pre-emptive detection)

### Failover Characteristics

When failover occurs:
- The congestion window (cwnd) on the new path starts from scratch (1 MTU for new paths, or the last known cwnd if the path was previously used)
- RTO for the new path uses that path's independently measured RTT
- There is no mechanism for seamless failover; some delay is expected during the transition
- Once the original primary recovers (HEARTBEAT-ACK received), the implementation may or may not switch back (this is implementation-specific)

---

## 6. PR-SCTP Policies

RFC 3758 defines the Partial Reliability extension, which allows a sender to stop retransmitting DATA chunks under certain conditions. The receiver is informed via FORWARD-TSN chunks.

### FORWARD-TSN Mechanism

When the sender abandons a DATA chunk:
1. It sends a FORWARD-TSN chunk with a new cumulative TSN, skipping the abandoned chunk
2. The receiver advances its cumulative TSN ACK to the new value
3. Any buffered out-of-order chunks that were waiting for the skipped TSN are either delivered or discarded depending on stream ordering

```
Sender transmits: TSN 100, 101, 102, 103
TSN 101 is lost and abandoned (timed out per policy)

Sender sends FORWARD-TSN: new cumulative = 101
  Stream info: stream 0 SSN=1 abandoned

Receiver:
  - Advances cumulative TSN to 101
  - Delivers TSN 102, 103 if they complete their stream ordering
  - SSN gap in stream 0: SSN=1 is skipped, SSN=2 delivered next
```

### Policy Definitions

**Timed Reliability**

The sender will retransmit a DATA chunk only for a specified duration after the first transmission. Once the timer expires, the chunk is abandoned.

```
Lifetime = 500ms

  t=0:    First transmission of TSN 200
  t=100:  Retransmit #1 (loss detected by SACK)
  t=300:  Retransmit #2
  t=500:  Lifetime expired -> abandon TSN 200, send FORWARD-TSN
```

Use case: real-time voice/video signaling where data older than a few hundred milliseconds is useless.

**Limited Retransmission**

The sender will retransmit a DATA chunk at most N times. After N retransmissions, the chunk is abandoned.

```
Max retransmissions = 2

  Attempt 0: First transmission of TSN 300
  Attempt 1: Retransmit #1
  Attempt 2: Retransmit #2
  -> Abandon TSN 300, send FORWARD-TSN
```

Use case: application-level retry logic exists, or the data has limited value after initial attempts.

**Priority-Based**

Under congestion, the sender preferentially abandons lower-priority DATA chunks to make room for higher-priority ones. This is less formally standardized but supported by some implementations.

### PR-SCTP and WebRTC

WebRTC data channels map directly to SCTP streams with configurable reliability:

```
RTCDataChannel options:
  ordered: true/false        -> maps to SCTP ordered/unordered delivery
  maxRetransmits: N          -> maps to limited retransmission PR-SCTP policy
  maxPacketLifeTime: ms      -> maps to timed reliability PR-SCTP policy

If neither maxRetransmits nor maxPacketLifeTime is set, the channel is fully reliable.
```

---

## 7. SCTP Packet Format (Complete)

An SCTP packet consists of a common header followed by one or more chunks:

```
+--------------------------------------------------+
|                 IP Header (v4/v6)                 |
+--------------------------------------------------+
|              SCTP Common Header (12 bytes)        |
|  +----------------------------------------------+|
|  | Source Port (16)  | Destination Port (16)     ||
|  +----------------------------------------------+|
|  | Verification Tag (32)                         ||
|  +----------------------------------------------+|
|  | Checksum -- CRC-32c (32)                      ||
|  +----------------------------------------------+|
+--------------------------------------------------+
|                Chunk 1                            |
|  +----------------------------------------------+|
|  | Type (8) | Flags (8) | Length (16)            ||
|  +----------------------------------------------+|
|  | Value (variable, padded to 4-byte boundary)   ||
|  +----------------------------------------------+|
+--------------------------------------------------+
|                Chunk 2                            |
|  +----------------------------------------------+|
|  | Type (8) | Flags (8) | Length (16)            ||
|  +----------------------------------------------+|
|  | Value (variable, padded to 4-byte boundary)   ||
|  +----------------------------------------------+|
+--------------------------------------------------+
|                ...                                |
+--------------------------------------------------+
```

### Common Header Details

- **Source Port / Destination Port**: Same semantics as TCP/UDP. IANA-assigned port 9899 is the default for SCTP-over-UDP encapsulation.
- **Verification Tag**: Set to the peer's Initiate Tag from the INIT/INIT-ACK exchange. INIT chunks use V-Tag = 0. The tag must match on every subsequent packet or the packet is silently discarded (anti-spoofing).
- **Checksum**: CRC-32c (Castagnoli) computed over the entire SCTP packet with the checksum field set to 0 during computation. CRC-32c was chosen over the TCP/UDP internet checksum for stronger error detection, critical for signaling applications where silent corruption is unacceptable.

### DATA Chunk Detail

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|   Type = 0    |  Res  |U|B|E  |         Chunk Length          |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                          TSN (32)                             |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|        Stream Identifier      |     Stream Sequence Number    |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                   Payload Protocol Identifier                 |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                         User Data                             |
|                          ...                                  |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

Flags:
- **U** (Unordered): When set, the SSN is ignored and the chunk is delivered immediately
- **B** (Beginning): First fragment of a user message
- **E** (Ending): Last fragment of a user message
- B=1, E=1 means a complete unfragmented message
- B=1, E=0 starts a fragmented message; B=0, E=0 continues; B=0, E=1 ends it

The Payload Protocol Identifier (PPID) is an application-defined value passed transparently. IANA maintains a registry; well-known values include 5 (S1-AP), 18 (X2-AP), 46 (Diameter), and 50/51 (WebRTC DCEP/String).

---

## References

- [RFC 9260 -- Stream Control Transmission Protocol](https://www.rfc-editor.org/rfc/rfc9260)
- [RFC 3758 -- SCTP Partial Reliability Extension](https://www.rfc-editor.org/rfc/rfc3758)
- [RFC 6951 -- UDP Encapsulation of SCTP Packets](https://www.rfc-editor.org/rfc/rfc6951)
- [RFC 6458 -- Sockets API Extensions for SCTP](https://www.rfc-editor.org/rfc/rfc6458)
- [RFC 4895 -- Authenticated Chunks for SCTP](https://www.rfc-editor.org/rfc/rfc4895)
- [RFC 5061 -- SCTP Dynamic Address Reconfiguration](https://www.rfc-editor.org/rfc/rfc5061)
- [Stewart, R. "Stream Control Transmission Protocol: A Reference Guide" (Addison-Wesley, 2013)](https://www.informit.com/store/stream-control-transmission-protocol-sctp-a-reference-9780321304735)
