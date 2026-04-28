# HTTP/3 over QUIC — Deep Dive

> *Math-heavy companion to `ramp-up/http3-quic-eli5`. Every formula, every bit, every constant
> from RFC 9000 / 9001 / 9002 / 9114 / 9204 / 9221 / 8446. Optimised for the in-terminal
> reader who needs the number, the encoding, and the worked example without leaving the shell.*

---

## 0. Stack View and Notation

```
+------------------------------+
|     HTTP/3 (RFC 9114)        |  semantics: methods, status, headers, body
|     QPACK (RFC 9204)         |  header compression with out-of-order refs
+------------------------------+
|     QUIC (RFC 9000)          |  streams, flow control, migration
|     QUIC-TLS (RFC 9001)      |  TLS 1.3 inside CRYPTO frames, AEAD framing
|     QUIC Recovery (RFC 9002) |  ACK, loss detection, congestion control
|     DATAGRAM (RFC 9221)      |  optional unreliable user-data extension
+------------------------------+
|     UDP                      |  datagram transport
|     IP                       |  network layer
+------------------------------+
```

Notation used throughout:

| Symbol | Meaning |
|:---|:---|
| `varint(n)` | QUIC variable-length integer encoding of `n` (1, 2, 4, or 8 bytes) |
| `||`        | byte concatenation |
| `XOR`       | bitwise exclusive OR |
| `[a..b]`    | byte slice indexed `a` through `b-1` |
| `len(x)`    | byte length of `x` |
| `iv`        | per-key, per-direction 12-byte IV ("write_iv") |
| `pn`        | full 62-bit packet number (decoded) |
| `pn_enc`    | truncated packet number on the wire (8/16/24/32 bits) |
| `dcid`      | destination connection ID |
| `scid`      | source connection ID |
| `H()`       | HKDF-Extract |
| `E()`       | HKDF-Expand-Label (TLS 1.3 §7.1) |

---

## 1. QUIC Packet Structure

### 1.1 Long Header (Initial / 0-RTT / Handshake / Retry)

Used during the handshake and for any packet whose key context the receiver may not yet
share. The first byte's high bit is always `1`.

```
Long header (RFC 9000 §17.2):

  0                   1                   2                   3
  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
 +-+-+-+-+-+-+-+-+
 |1|1|T T|R R|P P|        T = packet type (00..11)
 +-+-+-+-+-+-+-+-+        R = reserved, MUST be 0 (header protection masks them)
 |     Version (32)     | P = packet number length minus one (0..3)
 +-----------------------+
 | DCID Len (8) | DCID  | DCID Len in 0..20
 +--------------+-------+
 | SCID Len (8) | SCID  | SCID Len in 0..20
 +--------------+-------+
 | Type-specific payload (Token, Length, Packet Number, Frames)
```

Type bits to packet-type table:

| `T T` | Packet type        | Crypto level | Carries CRYPTO? | Carries STREAM? |
|:------|:-------------------|:-------------|:----------------|:----------------|
| 00    | Initial            | Initial      | yes             | no              |
| 01    | 0-RTT              | 0-RTT        | no              | yes             |
| 10    | Handshake          | Handshake    | yes             | no              |
| 11    | Retry              | n/a          | no              | no              |

Version Negotiation is special: it has version `0x00000000`, no length, and contains a list
of supported 32-bit versions. It is not encrypted.

### 1.2 Short Header (1-RTT)

After the handshake completes, application data uses the short header. The first byte's
high bit is `0`.

```
Short header (RFC 9000 §17.3):

  0                   1
  0 1 2 3 4 5 6 7 ...
 +-+-+-+-+-+-+-+-+
 |0|1|S|R|R|K|P P|     S = spin bit (latency observation, optional)
 +-+-+-+-+-+-+-+-+     R = reserved (MUST be 0 under header protection)
 |   DCID (variable) | K = key phase (toggles on key update)
 +-------------------+ P P = packet number length minus one
 |   Packet Number   |
 +-------------------+
 |   Protected payload (frames)
```

The DCID length is implicit — a server picks the length when issuing CIDs, and the
receiver must already know it. There is no SCID on the wire (the connection is
already established).

### 1.3 Packet Number Spaces

QUIC has three independent packet number spaces. Each has its own monotonically
increasing 62-bit counter, its own AEAD keys, and its own ACK ranges.

| Space            | Used by                          | Lifetime                         |
|:-----------------|:---------------------------------|:---------------------------------|
| Initial          | Initial packets                  | Until Handshake keys derived     |
| Handshake        | Handshake packets                | Until 1-RTT keys derived         |
| Application Data | 0-RTT and 1-RTT packets          | Lifetime of the connection       |

Critical consequence: ACKs are space-scoped. An Initial packet can only be ACKed by an
Initial-space ACK; the same `pn=0` may legitimately appear in all three spaces.

### 1.4 AEAD Nonce Derivation

For each protected packet, the AEAD nonce is computed:

```
let pn_padded = zero_extend_left(pn, 12)       # 96-bit big-endian
let nonce     = pn_padded XOR iv               # iv is the 12-byte write_iv
```

This is RFC 9001 §5.3 verbatim. `iv` is per-direction and rotates on key update. Because
`pn` is unique within a packet number space and `iv` is unique per direction per key,
nonce reuse is prevented as long as `pn` is never reused — which is guaranteed by the
62-bit monotonic counter and key-update protocol.

```
Worked example (toy values):
  iv  = 00 00 00 00 00 00 00 00 00 00 00 01
  pn  = 0x12_34_56_78_9A_BC_DE_F0
  pn_padded = 00 00 00 00 12 34 56 78 9A BC DE F0
  nonce     = 00 00 00 00 12 34 56 78 9A BC DE F1
```

### 1.5 62-bit Variable-Length Integer Encoding

QUIC's `varint` encodes any value in `[0, 2^62 - 1]` using 1/2/4/8 bytes. The two most
significant bits of the first byte select the length; the remaining bits are the value.

| 2-bit prefix | Total bytes | Useable bits | Value range                 |
|:-------------|:-----------:|:------------:|:----------------------------|
| `00`         | 1           | 6            | `0` to `2^6 - 1` = 63       |
| `01`         | 2           | 14           | `0` to `2^14 - 1` = 16383   |
| `10`         | 4           | 30           | `0` to `2^30 - 1`           |
| `11`         | 8           | 62           | `0` to `2^62 - 1`           |

Encode pseudo-code:

```python
def varint_encode(n: int) -> bytes:
    if n < (1 << 6):
        return bytes([n])                                       # 00xxxxxx
    if n < (1 << 14):
        return (0x4000 | n).to_bytes(2, "big")                  # 01xxxxxx ...
    if n < (1 << 30):
        return (0x80000000 | n).to_bytes(4, "big")              # 10xxxxxx ...
    if n < (1 << 62):
        return (0xC000000000000000 | n).to_bytes(8, "big")      # 11xxxxxx ...
    raise ValueError("value out of range")

def varint_decode(buf: bytes, off: int) -> tuple[int, int]:
    prefix = buf[off] >> 6
    length = 1 << prefix                                        # 1, 2, 4, 8
    val = buf[off] & 0x3F
    for i in range(1, length):
        val = (val << 8) | buf[off + i]
    return val, off + length
```

Concrete examples (RFC 9000 Appendix A):

| Value         | Encoded bytes                      | Length |
|:-------------:|:-----------------------------------|:------:|
| 37            | `25`                               | 1      |
| 15293         | `7B BD`                            | 2      |
| 494,878,333   | `9D 7F 3E 7D`                      | 4      |
| 151,288,809,941,952,652 | `C2 19 7C 5E FF 14 E8 8C` | 8      |

### 1.6 Header Protection

Even the packet number bits and reserved bits are encrypted on the wire. The procedure:

```
sample_offset = pn_offset + 4                      # always sample 4 bytes past pn_offset
sample        = packet[sample_offset .. sample_offset + 16]
mask          = HP_AES_ECB(hp_key, sample)[0..5]   # AES-based, or ChaCha20 keystream

# unprotect first byte:
if long_header:  packet[0] ^= mask[0] & 0x0F
else:            packet[0] ^= mask[0] & 0x1F

# unprotect packet number bytes (1..4 bytes after recovering length):
pn_len = (packet[0] & 0x03) + 1
for i in range(pn_len):
    packet[pn_offset + i] ^= mask[1 + i]
```

This means an on-path observer cannot trivially correlate retransmissions by packet
number, and reserved bits cannot be set by a meddling middlebox without breaking the
AEAD integrity check.

---

## 2. Connection ID Negotiation

### 2.1 Source vs Destination CID

A QUIC connection is identified at each endpoint by *its* DCID, which is the *peer's*
chosen identifier. They are NOT the same value at both ends.

```
Client                                                      Server
  scid_c = pick(rand)
  dcid_c = pick(rand)        # peer-unaware initial DCID
  --[ Initial: DCID=dcid_c, SCID=scid_c ]-->
                                                  scid_s = pick(rand)
                                                  dcid_s = scid_c       # echo back
  <--[ Initial: DCID=scid_c, SCID=scid_s ]--
  dcid_c := scid_s           # adopt server's chosen value
```

After the first server packet, the client uses `scid_s` as its DCID and the server uses
`scid_c` as its DCID. The Initial DCID is also used as the salt for derivation of the
Initial AEAD keys (§5.2 below) — so it MUST be at least 8 bytes per RFC 9000 §7.2.

### 2.2 NEW_CONNECTION_ID and retire_prior_to

To support migration, each peer can advertise additional CIDs. Frame layout:

```
NEW_CONNECTION_ID Frame {
    Frame Type           = 0x18 (varint)
    Sequence Number      (varint)
    Retire Prior To      (varint)         # MUST be <= Sequence Number
    Length               (8)              # 1..20
    Connection ID        (8 * Length)
    Stateless Reset Token (128)
}
```

Constraint inequality:

```
0 <= retire_prior_to <= sequence_number
issued_unretired_count <= active_connection_id_limit
```

Where `active_connection_id_limit` is a transport parameter (default 2; minimum 2;
practical max 8). If a peer issues more than the limit allows, that's a
`CONNECTION_ID_LIMIT_ERROR` (transport error 0x09).

### 2.3 Retiring CIDs (RETIRE_CONNECTION_ID)

When `retire_prior_to` advances, the receiver MUST send `RETIRE_CONNECTION_ID` for every
CID with sequence number `< retire_prior_to`. It does so on the new path's CID — the
retired CID is no longer valid for routing.

```
  active_set := {cid_seq_2, cid_seq_3, cid_seq_4}
  peer issues NEW_CONNECTION_ID(seq=5, retire_prior_to=4)
  -->  retire seq 2 and seq 3
  -->  active_set := {cid_seq_4, cid_seq_5}
```

### 2.4 Connection Migration & NAT Rebinding

Migration is initiated when the client sends a non-probing packet from a new 4-tuple.
The server triggers path validation:

```
Server:    PATH_CHALLENGE  data=R (8 random bytes)        --> new path
Client:    PATH_RESPONSE   data=R                         --> server
```

Validation succeeds when the response arrives on the new path with matching `data`. The
amplification limit (§9.2) still applies until the new path is validated.

NAT rebinding is the silent case: same client, same DCID, but a different src IP/port
because the NAT mapping refreshed. The server detects this by 4-tuple change with a
known DCID and validates the new path before sending more than 3x the bytes received on
that new path.

```
  PRE-REBIND   src=192.0.2.5:51000  ->  server.dst=:443     (validated)
  POST-REBIND  src=192.0.2.5:62000  ->  server.dst=:443     (NEW)
                                            |
                                            | server: max_send <= 3 * recv_from_new_path
                                            | server -> PATH_CHALLENGE
                                            | client -> PATH_RESPONSE
                                            | path validated; congestion controller reset
```

Per RFC 9000 §9.4, the congestion controller for the new path is reset to its initial
state (cwnd to `kInitialWindow`, srtt undefined). The old path may be retained for a
short period as a fallback.

---

## 3. Stream Multiplexing Math

### 3.1 Stream ID Encoding

Stream IDs are 62-bit varints whose two lowest bits encode role and direction:

| Bits 1..0 | Initiator | Direction       | Stream ID examples              |
|:---------:|:----------|:----------------|:--------------------------------|
| `00`      | Client    | Bidirectional   | 0, 4, 8, 12, ... (≡ 0 mod 4)    |
| `01`      | Server    | Bidirectional   | 1, 5, 9, 13, ... (≡ 1 mod 4)    |
| `10`      | Client    | Unidirectional  | 2, 6, 10, 14, ... (≡ 2 mod 4)   |
| `11`      | Server    | Unidirectional  | 3, 7, 11, 15, ... (≡ 3 mod 4)   |

So `next_client_bidi = client_bidi_count * 4` and similarly for the other three buckets.
The total number of streams of one (direction, initiator) pair is bounded by the
peer's `MAX_STREAMS` limit — but the maximum theoretical stream ID is `2^62 - 1`, giving
roughly `2^60` streams per bucket.

```
total stream IDs reachable from one endpoint per bucket = floor((2^62 - bits) / 4) + 1
                                                       ≈ 2^60 ≈ 1.15e18
```

### 3.2 MAX_STREAMS Frame

```
MAX_STREAMS Frame {
    Frame Type     = 0x12 (bidi) | 0x13 (uni)
    Maximum Streams (varint)        # cumulative cap, MUST be monotonic non-decreasing
}
```

Constraints:

```
peer_bidi_streams_open <= max_bidi_streams
peer_uni_streams_open  <= max_uni_streams
0 <= max_*_streams     <= 2^60                # MAX_STREAMS limit
```

If a peer opens stream `s` such that `s/4 + 1 > max_streams`, that's a flow control
error: `STREAM_LIMIT_ERROR` (transport error 0x04).

### 3.3 Flow Control: MAX_DATA and MAX_STREAM_DATA

Two layers of credit:

| Layer       | Frame              | Receive bound                        |
|:------------|:-------------------|:-------------------------------------|
| Connection  | `MAX_DATA`         | sum of bytes consumed, all streams   |
| Stream      | `MAX_STREAM_DATA`  | bytes consumed on a single stream    |

The send-side rule:

```
for any byte offset b on stream S being sent:
    b < min( max_stream_data[S], max_data - sum(b' on other streams) )
```

A common implementation strategy (the "auto-tune" rule) is to extend credit when the
peer has consumed half of the current window:

```
if consumed >= 0.5 * window:
    new_window = max(window, 2 * bytes_in_flight_on_stream)
    send MAX_STREAM_DATA(stream=S, max=new_window)
```

Numerical example:

```
RTT       = 50 ms
BW target = 100 Mbps (= 12.5 MB/s)
BDP       = RTT * BW = 0.05 * 12_500_000 = 625_000 bytes  ≈ 0.6 MB

initial MAX_STREAM_DATA = 256 KiB
after 128 KiB consumed -> peer extends to max(256 KiB, 2 * 128 KiB) = 256 KiB ... no growth
keep going; once bytes_in_flight grows, window doubles to BDP
final stable window      ≈ 1 MiB (covers BDP with headroom)
```

If the receiver fails to extend `MAX_STREAM_DATA`, the sender must throttle. A repeated
window-zero stall is observable as flat throughput at exactly `window / RTT`.

### 3.4 Head-of-Line Elimination Math

In TCP, any single packet loss stalls *all* multiplexed streams until retransmission.
Let `S` be the number of concurrently active streams and `p` the per-packet loss
probability:

```
P(any stream blocked, TCP)  = 1 - (1 - p)^N            # N = packets in flight, all streams
P(stream i blocked, QUIC)   = 1 - (1 - p)^N_i          # N_i = packets in flight for stream i
```

The expected number of streams blocked by a single random loss:

```
E[blocked streams | TCP]   = S         (always all)
E[blocked streams | QUIC]  = 1
```

Per-stream retransmit cost:

```
C_retx,TCP   = 1 RTT * S      (S streams stalled for one RTT)
C_retx,QUIC  = 1 RTT * 1      (only the affected stream)
```

So QUIC's HOL elimination converts a multiplicative cost into an additive one. With
`S = 10` streams and 1% packet loss, expected cumulative stall per second of activity:

```
TCP:   p * S * RTT = 0.01 * 10 * 50ms = 5 ms/s of multiplexed stall
QUIC:  p * 1 * RTT = 0.01 *  1 * 50ms = 0.5 ms/s
```

A 10x reduction in user-visible jitter on a 1% lossy path.

---

## 4. Crypto Handshake

### 4.1 TLS 1.3 Inside CRYPTO Frames

QUIC-TLS does NOT use TLS records. It carries TLS handshake messages (ClientHello,
ServerHello, Certificate, Finished, etc.) inside QUIC `CRYPTO` frames:

```
CRYPTO Frame {
    Frame Type   = 0x06 (varint)
    Offset       (varint)
    Length       (varint)
    Crypto Data  (Length bytes)
}
```

CRYPTO frames have their own offset stream per packet number space. This means TLS
handshake bytes have stream-like ordering semantics within a space, but cross-space
ordering is enforced by the protocol state machine (Initial → Handshake → 1-RTT).

### 4.2 Initial Secret Derivation

The Initial keys are derived from the client's first DCID using a fixed salt:

```
initial_salt = 0x38762cf7f55934b34d179ae6a4c80cadccbb7f0a    # RFC 9001 §5.2 (v1)
initial_secret = HKDF-Extract(salt = initial_salt, IKM = client_dcid)

client_initial_secret = HKDF-Expand-Label(initial_secret, "client in", "", 32)
server_initial_secret = HKDF-Expand-Label(initial_secret, "server in", "", 32)
```

Where `HKDF-Expand-Label` is TLS 1.3's labelled expansion (RFC 8446 §7.1):

```
HKDF-Expand-Label(secret, label, ctx, len) =
    HKDF-Expand(secret, HkdfLabel, len)

struct HkdfLabel {
    uint16  length      = len
    opaque  label<7..255>  = "tls13 " || label
    opaque  context<0..255> = ctx
}
```

From each direction's secret, key/iv/hp are derived:

```
key  = HKDF-Expand-Label(secret, "quic key", "", 16)
iv   = HKDF-Expand-Label(secret, "quic iv",  "", 12)
hp   = HKDF-Expand-Label(secret, "quic hp",  "", 16)
```

The 16-byte `hp` is used by header protection (AES-128-ECB or ChaCha20 keystream
sample). Note `iv` is exactly 12 bytes — matching AEAD nonce length.

### 4.3 General HKDF Math

For the curious, the underlying HKDF (RFC 5869) is:

```
HKDF-Extract(salt, IKM)         = HMAC-Hash(salt, IKM)              -> PRK (Pseudo-Random Key)
HKDF-Expand(PRK, info, L)       = T(1) || T(2) || ... || T(N)       -> OKM (truncated to L)

where:
    T(0) = empty string
    T(i) = HMAC-Hash(PRK, T(i-1) || info || byte(i))
    N    = ceil(L / HashLen)
```

So `OKM = HKDF-Expand(PRK, info, L)` produces `L` bytes by chaining HMAC outputs. With
SHA-256, `HashLen = 32`, so a `key` of 16 bytes needs a single iteration; an `iv` of 12
bytes also fits in one iteration; an `hp` of 16 bytes likewise.

### 4.4 1-RTT Secret Derivation (After Handshake)

After the TLS 1.3 handshake completes, the master secret yields client/server
application traffic secrets:

```
master_secret = HKDF-Extract(handshake_secret, derived_secret)
client_app_traffic_secret_0 = Derive-Secret(master_secret, "c ap traffic", ClientHello..server Finished)
server_app_traffic_secret_0 = Derive-Secret(master_secret, "s ap traffic", ClientHello..server Finished)
```

These secrets feed the same `key/iv/hp` derivation (§4.2 lines 3-5 substituting the new
secret). They form the keys for 1-RTT packets — the steady-state of the connection.

### 4.5 Key Update — KEY_PHASE Bit

To rotate keys without a renegotiation, QUIC uses HKDF chaining. The current secret is
fed back through `HKDF-Expand-Label` to derive the next:

```
secret_(n+1) = HKDF-Expand-Label(secret_n, "quic ku", "", HashLen)
key_(n+1)    = HKDF-Expand-Label(secret_(n+1), "quic key", "", 16)
iv_(n+1)     = HKDF-Expand-Label(secret_(n+1), "quic iv",  "", 12)
# hp does NOT rotate
```

Wire signaling: the short header contains a single `K` (key phase) bit. Toggling it from
the previous packet number signals "I am now sending under generation N+1 keys." The
peer must:

1. Detect the toggle on a packet whose AEAD verifies under generation `n+1`.
2. Derive `secret_(n+1)`, `key_(n+1)`, `iv_(n+1)`.
3. Continue using its own current generation for sending until it chooses to rotate.
4. After 3 RTTs (or `kKeyUpdateDelay`), discard the old generation's keys.

```
                  KEY_PHASE
Packet stream:  0 0 0 0 1 1 1 1 0 0 0 0 1 1 1 1 ...
Generation:     n n n n n+1...   n+2 ...     n+3 ...
```

A receiver MUST NOT initiate key update unless it has confirmed the handshake.
After more than `2^60` packets in a generation, an endpoint MUST initiate key update
or close the connection (`KEY_UPDATE_ERROR`, 0x0E).

---

## 5. Loss Detection & ACK

### 5.1 Probe Timeout (PTO)

```
PTO = smoothed_rtt + max(4 * rttvar, kGranularity) + max_ack_delay
```

Constants and meanings:

| Symbol           | Value / Source                                  |
|:-----------------|:------------------------------------------------|
| `smoothed_rtt`   | Exponentially smoothed RTT (initial: kInitialRtt = 333 ms) |
| `rttvar`         | RTT variance estimator                          |
| `kGranularity`   | 1 ms (timer minimum)                            |
| `max_ack_delay`  | Peer's transport parameter, default 25 ms       |

The PTO doubles on each consecutive trigger (`pto_count`), bounded by 60 s by most
implementations:

```
effective_PTO = min(PTO * 2^pto_count, 60s)
```

Numerical example, steady-state:

```
smoothed_rtt = 50 ms
rttvar       = 10 ms          ->  4 * rttvar = 40 ms
max_ack_delay = 25 ms

PTO = 50 + max(40, 1) + 25 = 50 + 40 + 25 = 115 ms
```

After 1 PTO trigger (`pto_count = 1`): `effective_PTO = 230 ms`. After 6 triggers:
`effective_PTO = 7360 ms` (capped well below 60 s).

### 5.2 ACK Frame Layout

```
ACK Frame {
    Frame Type           = 0x02 or 0x03 (with ECN counts)
    Largest Acknowledged (varint)
    ACK Delay            (varint)         # microseconds, scaled by ack_delay_exponent
    ACK Range Count      (varint)
    First ACK Range      (varint)         # contiguous from Largest Acknowledged

    ACK Range {
        Gap                (varint)
        ACK Range Length   (varint)
    } [ACK Range Count]

    [ECN Counts {
        ECT0 Count       (varint)
        ECT1 Count       (varint)
        CE Count         (varint)
    }]
}
```

The Gap encoding is offset-based (one less than the actual gap, to allow zero-byte
encoding for tightly-packed ranges):

```
gap_actual    = gap_field + 1
range_length  = ACK Range Length + 1
```

Decoding ranges proceeds from `Largest Acknowledged` downward:

```python
def ack_decode(largest, first_range, ranges):
    acked = []
    end = largest
    start = end - first_range
    acked.append((start, end))
    for gap, length in ranges:
        end = start - gap - 2          # gap_actual = gap + 1, then -1 for the start
        start = end - length
        acked.append((start, end))
    return acked
```

### 5.3 ACK Delay Encoding

`ack_delay` is encoded scaled by `2^ack_delay_exponent` (transport parameter, default 3,
meaning 8x). The actual delay in microseconds:

```
ack_delay_us = ack_delay_field * 2^ack_delay_exponent
```

A peer with `ack_delay_exponent = 3` and `ack_delay_field = 1000` reports a 8000 µs (8
ms) delay between receipt and ACK transmission.

### 5.4 RTT Sampling (Adjusted)

```
rtt_sample = ack_arrival_time - largest_acknowledged_send_time
adjusted   = max(rtt_sample - ack_delay_us, kGranularity)         # subtract peer-side delay
smoothed   = 7/8 * smoothed + 1/8 * adjusted
rttvar     = 3/4 * rttvar  + 1/4 * abs(smoothed - adjusted)
```

The `ack_delay` subtraction is what eliminates the systematic bias TCP suffers from
delayed ACKs without explicit signaling.

### 5.5 Packet Loss: kPacketThreshold and kTimeThreshold

A packet `P` is declared lost if either:

```
(reorder threshold)  any packet sent after P with pn > P.pn + kPacketThreshold has been ACKed
(time threshold)     now() - P.send_time > kTimeThreshold * max(smoothed_rtt, latest_rtt)

kPacketThreshold = 3
kTimeThreshold   = 9/8                # 1.125
```

Pseudo-code:

```python
def detect_loss(in_flight, largest_acked, now, srtt, latest_rtt):
    loss_delay = (9 / 8) * max(srtt, latest_rtt)
    loss_delay = max(loss_delay, K_GRANULARITY)
    lost = []
    for p in in_flight:
        if p.pn + 3 <= largest_acked or (now - p.send_time) > loss_delay:
            lost.append(p)
    return lost
```

### 5.6 Persistent Congestion

If multiple consecutive packets are lost across a duration exceeding the persistent
congestion threshold:

```
duration = (smoothed_rtt + max(4 * rttvar, kGranularity) + max_ack_delay)
           * kPersistentCongestionThreshold

kPersistentCongestionThreshold = 3
```

In persistent congestion, the controller resets cwnd to `kMinimumWindow` (typically
2 * max_datagram_size).

---

## 6. Congestion Control

### 6.1 NewReno (Default per RFC 9002)

Phases:

```
SLOW_START:        cwnd  += bytes_acked                           (every ACK)
CONGESTION_AVOID:  cwnd  += max_datagram_size * bytes_acked / cwnd
RECOVERY:          cwnd unchanged; ssthresh = cwnd / 2 on entry
```

State transitions:

```
       (loss event)         (cwnd > ssthresh AND no loss)
SS  ----------------> RECO -------------------------------> CA
                       |                                    |
                       | (recovery_start_time elapsed)      |
                       +------------------------------------+
                                   re-enter SS on persistent_congestion
```

Constants:

```
kInitialWindow             = min(10 * max_datagram_size, max(14_720, 2 * max_datagram_size))
kMinimumWindow             = 2 * max_datagram_size
kLossReductionFactor       = 0.5
kPersistentCongestionThres = 3
```

### 6.2 CUBIC

CUBIC (RFC 9438) replaces NewReno's linear growth with a cubic curve centred on the
last loss:

```
W_cubic(t) = C * (t - K)^3 + W_max
K          = cbrt( W_max * (1 - beta_cubic) / C )

C            = 0.4
beta_cubic   = 0.7
```

Where `W_max` is cwnd at the last loss event. The result: rapid probing far from `W_max`,
slow probing near `W_max`. `t` is wall time since the loss.

### 6.3 BBRv2 (Sketch)

BBRv2 estimates two state variables:

```
BtlBw     = max-filtered delivered-bytes / RTT over a 10-RTT window
RTprop    = min-filtered RTT over a 10-second window
inflight_target = BtlBw * RTprop * gain
```

It does not react to single-packet loss; instead, it cycles through pacing gains
{1.25, 0.75, 1, 1, 1, 1, 1, 1} to probe for additional bandwidth and cycle the buffer
empty.

### 6.4 ECN in QUIC

QUIC negotiates ECN by the sender setting ECT(0) or ECT(1) in the IP header, and the
receiver echoing CE counts in the `ACK_ECN` frame variant (frame type `0x03`):

```
ECN code points (IP header, 2 bits):
  Not-ECT  = 00
  ECT(0)   = 10
  ECT(1)   = 01
  CE       = 11   (Congestion Experienced)
```

Path validation: the sender confirms the path preserves ECN markings by sending a few
ECT-marked packets and checking the ECT0/ECT1 counts in the receiver's ACK match what
was sent. If they don't, the sender disables ECN ("ECN blackhole detected").

A CE count increment is treated equivalently to a single packet loss for congestion
control purposes (cwnd reduction by `kLossReductionFactor`), but does NOT trigger
retransmission.

---

## 7. 0-RTT and Replay

### 7.1 Session Tickets (NewSessionTicket)

After the 1-RTT handshake, the server may issue session tickets:

```
NewSessionTicket {
    ticket_lifetime    uint32      # seconds, MUST be <= 7 days = 604800
    ticket_age_add     uint32
    ticket_nonce       opaque<0..255>
    ticket             opaque<1..2^16-1>
    extensions         (incl. early_data with max_early_data_size)
}
```

`max_early_data_size` is the byte budget the server will accept under the resumption
PSK in 0-RTT. A common default is 16384 (16 KiB) to bound replay damage and CPU cost.

### 7.2 Replay Risk Model

0-RTT data has zero crypto-level freshness — an attacker who captures the original
0-RTT packet can replay it byte-identical to the server, and AEAD will verify because
the keys are deterministic from the PSK.

Risk decomposition:

```
P(replay accepted) = P(within_validity_window) * P(server_lacks_anti_replay_cache)
```

Mitigations (RFC 8446 §8 + RFC 9001 §4.3):

| Mitigation                | Implementation cost                   | Coverage             |
|:--------------------------|:--------------------------------------|:---------------------|
| Single-use tickets        | Per-ticket bookkeeping at server      | All replays         |
| Time-windowed cache       | `O(rate * window)` memory             | Bounded windows     |
| Idempotent endpoints only | Application-level routing rule        | All non-idempotent  |

### 7.3 Anti-Replay Window Math

If the server uses a sliding window cache of size `W` storing `(client_id, hash, ts)`
tuples for `T` seconds:

```
Acceptance probability for an attacker with N replay attempts:

  P(replay succeeds at all) = P(replay_arrives_within_T) - P(cache_evicted)
                            ≈ 1                            (for typical T = 5 minutes)
                            - 0                            (if cache holds all in T window)

Memory cost per second of capacity:
  M = R * sizeof(entry)                      # R = ops/sec, e.g., 50000 req/s, entry = 64 B
  M_total = R * T * sizeof(entry)            # 50000 * 300 * 64 = 960 MB
```

So for a busy server, a strict anti-replay cache is non-trivial — which is exactly why
RFC 8446 advises 0-RTT only for idempotent operations.

### 7.4 Limit Enforcement

On the server, three bounds apply concurrently to a 0-RTT-resumed connection:

```
bytes_received_0rtt   <= max_early_data_size
streams_opened_0rtt   <= negotiated max_streams (subject to resumption-time limit)
time_since_ticket     <= ticket_lifetime
```

Violation closes the connection with `PROTOCOL_VIOLATION` (0x0a) or rejects the early
data and forces a 1-RTT handshake.

---

## 8. HTTP/3 Over QUIC

### 8.1 HTTP/3 Frame Types

```
HTTP/3 Frame {
    Type    (varint)        # selector
    Length  (varint)        # bytes of payload
    Payload (Length bytes)
}
```

| Type | Name           | Allowed on streams                | Payload                          |
|:----:|:---------------|:-----------------------------------|:---------------------------------|
| 0x00 | DATA           | Request streams                   | Opaque body bytes                |
| 0x01 | HEADERS        | Request streams                   | QPACK encoded field section      |
| 0x03 | CANCEL_PUSH    | Control stream                    | Push ID (varint)                 |
| 0x04 | SETTINGS       | Control stream (first frame)      | Setting ID/Value pairs           |
| 0x05 | PUSH_PROMISE   | Request streams                   | Push ID + encoded headers        |
| 0x07 | GOAWAY         | Control stream                    | Last accepted stream/push ID     |
| 0x0D | MAX_PUSH_ID    | Control stream                    | New maximum Push ID              |

Frame types `0x02`, `0x06`, `0x08`, `0x09`, `0xFF`, `0x0E` are explicitly reserved or
used as "grease" frames (intentional unrecognised values to ensure peers correctly
ignore them).

### 8.2 Stream Type Selectors (Unidirectional)

Each unidirectional stream's first byte selects its role:

| Stream type | Hex  | Role                                    |
|:------------|:----:|:----------------------------------------|
| 0x00        | 0x00 | HTTP/3 control stream                   |
| 0x01        | 0x01 | HTTP/3 push stream                      |
| 0x02        | 0x02 | QPACK encoder stream                    |
| 0x03        | 0x03 | QPACK decoder stream                    |
| 0x21        | 0x21 | WebTransport stream (RFC 9297)          |

### 8.3 SETTINGS Frame

```
SETTINGS Frame Payload {
    Setting {
        Identifier  (varint)
        Value       (varint)
    } [...]
}
```

Standard identifiers (RFC 9114 §7.2.4 + RFC 9204):

| ID     | Name                                  | Default |
|:-------|:--------------------------------------|:--------|
| 0x06   | SETTINGS_MAX_FIELD_SECTION_SIZE       | 0 (unbounded) |
| 0x01   | SETTINGS_QPACK_MAX_TABLE_CAPACITY     | 0       |
| 0x07   | SETTINGS_QPACK_BLOCKED_STREAMS        | 0       |

Constraint: `SETTINGS_QPACK_BLOCKED_STREAMS = 0` disables out-of-order references and
turns QPACK into "static-table only," sacrificing compression ratio for predictability.

### 8.4 GOAWAY Frame

Used to gracefully wind down a connection:

```
GOAWAY {
    Frame Type = 0x07
    Length     = varint(N)
    Stream/Push ID  (varint)   # value semantics depend on direction
}
```

Server-sent: indicates the largest stream ID that will be processed. Client-sent:
indicates the largest Push ID that will be accepted. The endpoint may continue
processing already-open streams below the threshold, but new streams above it MUST be
refused with `H3_REQUEST_REJECTED`.

---

## 9. Performance Models

### 9.1 1-RTT vs 0-RTT vs TCP+TLS

Latency to first application byte:

```
L_TCP+TLS1.2  = 3 RTT     (SYN, ServerHello+Cert, ChangeCipherSpec+Finished, then HTTP)
L_TCP+TLS1.3  = 2 RTT     (SYN, ClientHello+ServerHello+Finished, then HTTP)
L_QUIC_1RTT   = 1 RTT     (Initial+Handshake collapsed; HTTP rides 1-RTT after)
L_QUIC_0RTT   = 0 RTT     (resumption with PSK; HTTP rides early_data)
```

Concrete: at 100 ms RTT, the savings from QUIC 0-RTT vs TCP+TLS 1.3 is `200 ms` per
connection, which translates to ~10% throughput improvement on short-lived HTTP
transactions like analytics beacons or thumbnail GETs.

### 9.2 Amplification Limit

To prevent QUIC from being abused as a UDP amplification vector, RFC 9000 §8 mandates:

```
bytes_sent_to_peer_X <= 3 * bytes_received_from_peer_X
```

Until `peer_X`'s address has been validated (either by completing handshake or by
PATH_CHALLENGE/PATH_RESPONSE on the new path).

Implication: a server cannot send a full Initial + ServerHello + Certificate in a
single flight if the client's Initial was very small. The server may need to pad client
initials (the client does this — `len(Initial) >= 1200` per RFC 9000 §14) so the server
has 3600 bytes of allowance to cover the certificate flight.

```
Worked example:
  client sends Initial of 1200 bytes (padded)
  server allowance      = 3 * 1200 = 3600 bytes
  server flight:
      Initial(ServerHello+EncryptedExtensions+CRYPTO start)  ~ 700 B
      Handshake(Certificate + CertificateVerify + Finished)  ~ 1500 B
      total ~ 2200 B  -> fits in 3600 B allowance ✓
```

If the server's flight exceeds the allowance, it MUST stop sending and wait for the
client's response (or for path validation to lift the bound).

### 9.3 PMTU Discovery — DPLPMTUD

RFC 8899 (DPLPMTUD = Datagram Packetization Layer PMTU Discovery) replaces ICMP-based
PMTUD which is unreliable because middleboxes drop ICMP. The QUIC algorithm:

```
state = SEARCHING
PROBED_SIZE := initial_min = 1200
MAX_PROBE   := 1500            # or higher with extensions

while state == SEARCHING:
    probe_size = next_step(PROBED_SIZE)        # often binary search or doubling
    send probe packet of probe_size with PADDING
    wait for ACK or PTO

    if ACKed:
        PROBED_SIZE = probe_size               # confirm
        if PROBED_SIZE >= MAX_PROBE: state = SEARCH_COMPLETE
    else if PTO_count >= 3:
        state = SEARCH_COMPLETE                # PROBED_SIZE is the last confirmed
```

A `PADDING` frame (`0x00`) inflates the packet to the probe size. PTOs from probes
should NOT be counted against persistent congestion.

```
Example trajectory on a 1500-MTU path:
  Step 1: probe 1280 -> ACK -> PROBED_SIZE = 1280
  Step 2: probe 1392 -> ACK -> PROBED_SIZE = 1392
  Step 3: probe 1448 -> ACK -> PROBED_SIZE = 1448
  Step 4: probe 1500 -> ACK -> PROBED_SIZE = 1500, SEARCH_COMPLETE
```

After completion, periodic re-probing every ~600 s confirms PMTU stability.

---

## 10. ECN, Pacing, and Datagram Extension

### 10.1 Pacing — Why It's Required

Without pacing, a sender can release a full cwnd of bytes back-to-back, instantly
overwhelming the bottleneck queue. Pacing target inter-packet gap:

```
inter_packet_gap = (max_datagram_size * 8) / pacing_rate_bps
pacing_rate      = (cwnd / smoothed_rtt) * pacing_gain
pacing_gain      = 1.25 in slow start, 1.0 in steady state
```

Numerical example:

```
cwnd        = 100 KB = 819200 bits
smoothed_rtt= 50 ms = 0.05 s
pacing_rate = 819200 / 0.05 = 16_384_000 bps = 16.4 Mbps

max_datagram = 1200 B = 9600 bits
gap          = 9600 / 16_384_000 = 586 µs
```

A timer fires every 586 µs to emit one packet — turning a 100 KB burst into a smooth 50
ms pour.

### 10.2 DATAGRAM Frame (RFC 9221)

For applications that prefer unreliable delivery (real-time audio/video, gaming):

```
DATAGRAM Frame (no length variant) {
    Frame Type = 0x30    # spans rest of packet
    Datagram Data
}

DATAGRAM Frame (with length) {
    Frame Type = 0x31
    Length     (varint)
    Datagram Data (Length)
}
```

Properties:

| Property                     | DATAGRAM | STREAM |
|:-----------------------------|:---------|:-------|
| Reliability                  | None     | Yes    |
| Ordering                     | None     | Yes per stream |
| Congestion control           | Yes      | Yes    |
| Flow control                 | None     | Yes    |
| AEAD-protected               | Yes      | Yes    |
| Counted in MAX_DATA          | No       | Yes    |

Negotiated by `max_datagram_frame_size` transport parameter (`0x20`). Value `0`
disables; `>0` is the maximum payload size the sender allows.

### 10.3 ECN Forward and Reverse Path Validation

ECN crosses the Internet from sender to receiver and back. The sender validates:

```
forward (sender -> receiver):
    sender marks packet with ECT(0)
    receiver records ECN bits in ACK frame ECT0/ECT1/CE counts
    sender confirms counts increment as expected

reverse (ECN-Echo via ACK):
    receiver counts CE marks
    receiver echoes counts in next ACK frame
    sender treats CE count delta as congestion signal
```

ECN bleach (where the path zeros the ECN bits) is detected by counts not incrementing.
ECN black-hole (path drops marked packets) is detected by total-loss spikes correlating
with ECT marking. In both cases the sender disables ECN for the connection.

---

## 11. Connection Close

### 11.1 CONNECTION_CLOSE Frame

Two flavours:

```
CONNECTION_CLOSE (transport, type 0x1c) {
    Error Code            (varint)        # transport error space, e.g. 0x0a (PROTOCOL_VIOLATION)
    Frame Type            (varint)        # offending frame type, 0 if not applicable
    Reason Phrase Length  (varint)
    Reason Phrase         (UTF-8 bytes)
}

CONNECTION_CLOSE (application, type 0x1d) {
    Error Code            (varint)        # application error space (HTTP/3 has its own)
    Reason Phrase Length  (varint)
    Reason Phrase         (UTF-8 bytes)
}
```

Transport error codes (selected, RFC 9000 §20):

| Code  | Symbol                        | Meaning                                   |
|:-----:|:------------------------------|:------------------------------------------|
| 0x00  | NO_ERROR                      | Graceful close                            |
| 0x01  | INTERNAL_ERROR                | Implementation bug                        |
| 0x02  | CONNECTION_REFUSED            | Server actively refused                   |
| 0x03  | FLOW_CONTROL_ERROR            | Peer exceeded MAX_DATA / MAX_STREAM_DATA  |
| 0x04  | STREAM_LIMIT_ERROR            | Peer exceeded MAX_STREAMS                 |
| 0x05  | STREAM_STATE_ERROR            | Frame sent in invalid stream state        |
| 0x06  | FINAL_SIZE_ERROR              | STREAM with FIN size mismatch             |
| 0x07  | FRAME_ENCODING_ERROR          | Malformed frame                           |
| 0x08  | TRANSPORT_PARAMETER_ERROR     | Bad transport parameter                   |
| 0x09  | CONNECTION_ID_LIMIT_ERROR     | Peer issued too many CIDs                 |
| 0x0a  | PROTOCOL_VIOLATION            | Generic protocol error                    |
| 0x0b  | INVALID_TOKEN                 | Bad Retry token                           |
| 0x0c  | APPLICATION_ERROR             | Application-level close (legacy use)      |
| 0x0d  | CRYPTO_BUFFER_EXCEEDED        | CRYPTO frame buffer overflow              |
| 0x0e  | KEY_UPDATE_ERROR              | Improper key update                       |
| 0x0f  | AEAD_LIMIT_REACHED            | Per-key packet count exhausted            |
| 0x10  | NO_VIABLE_PATH                | No usable path for migration              |
| 0x0100..0x01ff | CRYPTO_ERROR(TLS alert) | TLS alert mapped: code = 0x0100 + alert  |

HTTP/3 application error codes (RFC 9114 §8.1, partial):

| Code         | Symbol                       | Meaning                              |
|:------------:|:-----------------------------|:-------------------------------------|
| 0x0100       | H3_NO_ERROR                  | Graceful HTTP/3 close                |
| 0x0101       | H3_GENERAL_PROTOCOL_ERROR    | Protocol error not otherwise covered |
| 0x0102       | H3_INTERNAL_ERROR            | Internal problem                     |
| 0x0103       | H3_STREAM_CREATION_ERROR     | Bad stream usage                     |
| 0x0104       | H3_CLOSED_CRITICAL_STREAM    | Critical stream closed unexpectedly  |
| 0x0105       | H3_FRAME_UNEXPECTED          | Frame in wrong place                 |
| 0x0106       | H3_FRAME_ERROR               | Frame malformed                      |
| 0x0107       | H3_EXCESSIVE_LOAD            | Generic overload                     |
| 0x0108       | H3_ID_ERROR                  | Push ID or stream ID misuse          |
| 0x010c       | H3_REQUEST_CANCELLED         | Request abandoned                    |
| 0x010d       | H3_REQUEST_INCOMPLETE        | Request body incomplete              |
| 0x010f       | H3_MESSAGE_ERROR             | Malformed message                    |
| 0x0110       | H3_CONNECT_ERROR             | CONNECT failed                       |
| 0x0111       | H3_VERSION_FALLBACK          | Should retry with HTTP/1.1 or HTTP/2 |

### 11.2 Stateless Reset

If an endpoint loses state (crash, reboot, key rotation) but the peer keeps sending
1-RTT packets to a dead 4-tuple, the responder cannot decrypt — it has no keys. To
gracefully tear down without keys, RFC 9000 §10.3 defines stateless reset.

The reset packet is camouflaged as a 1-RTT packet:

```
Stateless Reset {
    Random      (>= 5 bytes, indistinguishable from short-header bytes)
    Reset Token (16 bytes)               # tail of the packet
}
```

The 16-byte `Reset Token` is bound to the DCID via a server-side static secret:

```
reset_token = HMAC(static_key, dcid)[0..16]
```

On receipt, the peer compares the trailing 16 bytes of an unprocessable packet against
the stored `Reset Token` for the connection. A match means: drop state, do not retry.

```
T_reset_collision = 1 / 2^128 ≈ 0
```

Reset Tokens are therefore considered cryptographic-quality identifiers. A leaked token
is a "kill switch" for that connection and MUST be transported only inside protected
frames (in `NEW_CONNECTION_ID`).

---

## 12. Worked Examples

### 12.1 0-RTT Replay Walkthrough

```
Setup:
  Client and Server completed a 1-RTT handshake yesterday.
  Server issued NewSessionTicket with max_early_data_size = 16384.
  Attacker captured the client's resumption ClientHello + 0-RTT data on the wire.

Today, replay attempt:

  Step 1: Attacker forwards captured packet bytes to server.
          - Initial: contains resumption ClientHello, encrypted under fixed initial keys.
            Server decrypts (initial keys are derived from public DCID + fixed salt).
          - 0-RTT: contains app data, encrypted under 0-RTT keys derived from PSK.
            Server, having the same PSK in cache, decrypts successfully.

  Step 2: Server checks anti-replay cache.
          Case (a): server uses single-use tickets, ticket already consumed yesterday.
                    -> server rejects 0-RTT, falls back to 1-RTT (key_share fresh).
                    -> attacker's 0-RTT data discarded; only a fresh 1-RTT path
                       proceeds, which the attacker can't drive (no private key).
          Case (b): server has time-windowed cache, T = 5 min.
                    Yesterday's transmission is older than 5 min.
                    -> hash not in cache; server accepts 0-RTT data.
                    -> if 0-RTT contained a non-idempotent POST, harm.
                    -> if idempotent GET, no real damage (response goes to attacker
                       but only contains public-bytes the server would serve anyway).

  Step 3: Lessons.
          - Server MUST gate 0-RTT data to idempotent operations.
          - Single-use tickets are the strongest mitigation but most expensive.
          - Time-windowed caches must be sized: T * peak_rate * sizeof(entry).
            Example: 5 min * 50000 ops/s * 64 B = 960 MB.
```

### 12.2 NAT Rebinding Migration

```
Pre-rebind:
  Client (192.0.2.5:51000) <---> Server (203.0.113.10:443)
  Client DCID = AABBCC...; Server DCID = DDEEFF...
  cwnd_old = 100 KB, srtt_old = 50 ms

  Client home router NAT mapping (51000 <-> public:42000) refreshes;
  router invents a new mapping (51000 <-> public:55555).
  Subsequent client packets to server arrive with apparent src=public:55555.

T = 0:
  Server receives short-header packet, DCID matches active CID, but src 4-tuple is new.
  Server enters "address validation pending" for new path.
  amplification_budget_new = 3 * received_bytes_on_new_path

T = ε:
  Server emits PATH_CHALLENGE(data=R) on new path with a *fresh* DCID
  (new_cid_seq = 7, drawn from CIDs the client previously gave the server).
  Server keeps old path congestion controller for a moment in case rebind reverses.

T = ~1 RTT/2:
  Client receives PATH_CHALLENGE, sends PATH_RESPONSE(data=R).

T = ~RTT:
  Server receives PATH_RESPONSE matching R.
  Server "validates" new path; lifts amplification limit.
  Server resets congestion controller for new path:
       cwnd_new = kInitialWindow (10 * max_datagram_size = ~14 KB)
       srtt_new = NEW measurements only

T > RTT:
  Server eventually sends NEW_CONNECTION_ID(seq=8, retire_prior_to=7) so the migrated
  path uses a fresh CID for unlinkability.
```

The cost: a single RTT of validation, plus a temporary cwnd reduction. A heavy
sustained flow may take several RTTs to recover full throughput on the new path.

### 12.3 Packet Number Space Crossover

```
Initial          space pn:  0, 1, 2, 3, 4
Handshake        space pn:  0, 1, 2
Application Data space pn:  0, 1, 2, 3, 4, 5, ..., 1000

Note: each space's packet numbers are independent. An "ACK 4" in Initial space
ACKs only the Initial-4 packet, not Application-4.

After Handshake completes:
  Endpoint discards Initial keys (RFC 9001 §4.9.1) when it receives a Handshake-space
  packet from peer.
  Endpoint discards Handshake keys when it confirms the handshake (server: receives
  Handshake-ACK; client: receives HANDSHAKE_DONE frame).

Once Initial keys are gone, Initial-space packet numbers are "burned" — any
retransmissions must move to whatever space is appropriate for their content.
CRYPTO data destined for Initial level cannot be sent if Initial keys are gone;
the connection must close with PROTOCOL_VIOLATION.
```

Visualization of the timeline:

```
Time --->
Client:   I0 -------- H0,H1 ----- A0,A1,A2 ----- A3,A4 -- ... -- A999
Server:   --- I0 ----- H0 ------ A0 ----- A1,A2,A3,A4 -- ... -- A998

Discards:  ^Init keys gone here for client ^ when first H received
                              ^ when HANDSHAKE_DONE seen, Handshake keys also gone
```

### 12.4 Key Update Sequence

```
KEY_PHASE bit on the wire:

   pn:    100 101 102 103 104 105 106 107 108 109 110 111
   K:     0   0   0   0   1   1   1   1   1   1   0   0

Generations (under the hood):
   pn 100..103: keys_gen_n (KEY_PHASE=0)
   pn 104..109: keys_gen_n+1 (KEY_PHASE=1)
   pn 110..xxx: keys_gen_n+2 (KEY_PHASE=0 again)

Initiator side (let's say sender) at pn=104:
  1. derives secret_(n+1) = HKDF-Expand-Label(secret_n, "quic ku", "", HashLen)
  2. derives key_(n+1), iv_(n+1) from secret_(n+1)
  3. starts emitting with KEY_PHASE = 1 (was 0)

Receiver upon seeing KEY_PHASE=1:
  - tries to AEAD-decrypt with current keys (gen_n) -- fails authentication tag
  - speculatively derives keys_(n+1)
  - re-tries decryption -- succeeds
  - commits to gen_n+1 for receiving
  - retains gen_n keys briefly in case of out-of-order packets pre-update

After 3 * RTT (or kKeyUpdateDelay), receiver discards gen_n keys.

If a packet with KEY_PHASE=0 arrives after gen_n keys are discarded but before
the next update: AEAD verification fails -> packet dropped silently. This is the
correct behaviour: KEY_PHASE alone is not authenticated; only the AEAD success
distinguishes generations.
```

### 12.5 Slow-Start to Congestion-Avoidance Transition

```
Initial state (NewReno):
  cwnd      = 10 * MSS = 14400 B (assume MSS=1440)
  ssthresh  = infinity
  srtt      = 50 ms

Phase 1 (slow start, exponential growth):
  RTT 1: send 10 packets, all ACKed -> cwnd += 10 * MSS = 28800 B
  RTT 2: send 20 packets, all ACKed -> cwnd += 20 * MSS = 57600 B
  RTT 3: send 40 packets, all ACKed -> cwnd += 40 * MSS = 115200 B
  ...
  RTT k: cwnd ≈ 10 * MSS * 2^(k-1)

Loss event at RTT 8 (cwnd = 1280 * MSS = 1.84 MB):
  ssthresh := cwnd / 2 = 920 KB
  cwnd      := ssthresh = 920 KB    (NewReno fast retransmit + recovery)
  state     := RECOVERY

Phase 2 (recovery, cwnd unchanged):
  Retransmit lost packet; further ACKs do NOT grow cwnd until new data segment ACKed.

Phase 3 (congestion avoidance, additive growth):
  Each RTT after recovery exits:
      cwnd += MSS               (i.e., cwnd_new = cwnd + MSS)

  RTT 9:  cwnd = 920 KB + 1.44 KB ≈ 921.44 KB
  RTT 10: cwnd = 922.88 KB
  ...
  RTT 100: cwnd ≈ 920 KB + 91 * MSS ≈ 1051 KB

  Linear growth at 1 MSS/RTT.

Persistent congestion check:
  If a contiguous run of packets is lost across more than
  (smoothed_rtt + 4 * rttvar + max_ack_delay) * 3 = (50 + 40 + 25) * 3 = 345 ms:
      cwnd := kMinimumWindow = 2 * MSS = 2880 B
      state := SLOW_START
```

### 12.6 HOL Blocking Elimination (Side-by-Side)

```
Setup: 4 streams (A, B, C, D), each with 5 packets.
Pacing: round-robin so packets on the wire are A1, B1, C1, D1, A2, B2, C2, D2, ..., A5, B5, C5, D5.
Loss:  packet B3 dropped.

TCP+HTTP/2 (single byte stream):
  byte stream order: A1 B1 C1 D1 A2 B2 C2 D2 A3 [B3 LOST] C3 D3 A4 B4 C4 D4 A5 B5 C5 D5
  receiver buffers everything past B3.
  Until B3 is retransmitted (1 RTT later), HEADERS for streams A, C, D
  cannot be reassembled in order at the HTTP/2 layer because the TCP
  byte stream is held back by the missing B3 byte range.
  -> All 4 streams stall for 1 RTT.

HTTP/3 over QUIC (4 independent streams):
  stream A: A1 A2 A3 A4 A5    (all delivered, in order, immediately)
  stream B: B1 B2 [B3 LOST] B4 B5
            QUIC delivers B1, B2 immediately;
            holds B4, B5 until B3 retransmitted (1 RTT).
  stream C: C1 C2 C3 C4 C5    (all delivered, in order, immediately)
  stream D: D1 D2 D3 D4 D5    (all delivered, in order, immediately)
  -> Only stream B stalls; A, C, D continue uninterrupted.

Net effect with 4 streams and 1% loss:
  P(any stream affected, TCP)  ≈ 1 - (1 - p)^N_total ≈ 1 - 0.99^20 ≈ 18%
  P(any stream affected, QUIC) ≈ 1 - (1 - p)^N_per_stream ≈ 1 - 0.99^5 ≈ 5%

  Average streams stalled per loss event:
    TCP:  4    (always all)
    QUIC: 1    (just the unlucky one)
```

---

## 13. Appendix — Selected Frame Quick Reference

```
| Type   | Name                          | Spaces        | I-bit on Frames |
|--------|-------------------------------|---------------|-----------------|
| 0x00   | PADDING                       | I, H, A       | yes             |
| 0x01   | PING                          | I, H, A       | yes             |
| 0x02   | ACK                           | I, H, A       | no              |
| 0x03   | ACK with ECN counts           | I, H, A       | no              |
| 0x04   | RESET_STREAM                  | A             | yes             |
| 0x05   | STOP_SENDING                  | A             | yes             |
| 0x06   | CRYPTO                        | I, H          | yes             |
| 0x07   | NEW_TOKEN                     | A             | yes             |
| 0x08   | STREAM (variants 0x08..0x0f)  | A             | yes             |
| 0x10   | MAX_DATA                      | A             | yes             |
| 0x11   | MAX_STREAM_DATA               | A             | yes             |
| 0x12   | MAX_STREAMS (bidi)            | A             | yes             |
| 0x13   | MAX_STREAMS (uni)             | A             | yes             |
| 0x14   | DATA_BLOCKED                  | A             | yes             |
| 0x15   | STREAM_DATA_BLOCKED           | A             | yes             |
| 0x16   | STREAMS_BLOCKED (bidi)        | A             | yes             |
| 0x17   | STREAMS_BLOCKED (uni)         | A             | yes             |
| 0x18   | NEW_CONNECTION_ID             | A             | yes             |
| 0x19   | RETIRE_CONNECTION_ID          | A             | yes             |
| 0x1a   | PATH_CHALLENGE                | A             | yes             |
| 0x1b   | PATH_RESPONSE                 | A             | yes             |
| 0x1c   | CONNECTION_CLOSE (transport)  | I, H, A       | no              |
| 0x1d   | CONNECTION_CLOSE (application)| A             | no              |
| 0x1e   | HANDSHAKE_DONE                | A             | yes             |
| 0x30   | DATAGRAM (no-len)             | A             | yes             |
| 0x31   | DATAGRAM (with-len)           | A             | yes             |
```

The "I-bit" column indicates whether the frame counts toward in-flight bytes for
congestion control (`true` = ACK-eliciting + counts as in-flight).

---

## 14. Appendix — Transport Parameter Reference

Selected transport parameters from RFC 9000 §18.2 and RFC 9221 §3:

| ID    | Name                                  | Default        | Notes                               |
|:-----:|:--------------------------------------|:---------------|:------------------------------------|
| 0x00  | original_destination_connection_id    | n/a            | Server only; echo of client DCID    |
| 0x01  | max_idle_timeout                      | 0 (off)        | ms; smallest of two peers wins      |
| 0x02  | stateless_reset_token                  | n/a            | 16 bytes, server-to-client          |
| 0x03  | max_udp_payload_size                  | 65527          | bytes; lower bound 1200             |
| 0x04  | initial_max_data                      | 0              | Connection-level flow control       |
| 0x05  | initial_max_stream_data_bidi_local    | 0              | Local-initiated bidi stream window  |
| 0x06  | initial_max_stream_data_bidi_remote   | 0              | Peer-initiated bidi stream window   |
| 0x07  | initial_max_stream_data_uni           | 0              | Uni-stream window                   |
| 0x08  | initial_max_streams_bidi              | 0              | Concurrent bidi streams cap         |
| 0x09  | initial_max_streams_uni               | 0              | Concurrent uni streams cap          |
| 0x0a  | ack_delay_exponent                    | 3              | Scale factor for ack_delay field    |
| 0x0b  | max_ack_delay                         | 25             | ms; used in PTO formula             |
| 0x0c  | disable_active_migration              | false          | If set, peer must not migrate       |
| 0x0d  | preferred_address                     | n/a            | IPv4/IPv6 + new DCID + reset token  |
| 0x0e  | active_connection_id_limit            | 2 (min)        | Cap on issued unretired CIDs        |
| 0x0f  | initial_source_connection_id          | n/a            | Echo of own SCID                    |
| 0x10  | retry_source_connection_id            | n/a            | Echoed Retry SCID                   |
| 0x20  | max_datagram_frame_size               | 0 (off)        | RFC 9221; >0 enables DATAGRAM       |

Validation rules:

```
- max_udp_payload_size MUST be in [1200, 65527]
- ack_delay_exponent MUST be <= 20 (else AEAD_LIMIT_REACHED on decoding)
- max_ack_delay MUST be < 2^14 ms (16384)
- active_connection_id_limit MUST be >= 2
- Initial flow control values may be 0 only if streams of that kind are not used
```

---

## 15. Appendix — Constants Reference Table

Pulled from RFC 9001/9002 and common implementation defaults. Never carry these in
your head; always look them up from the spec or your implementation:

| Constant                         | Value                                    | Source         |
|:---------------------------------|:-----------------------------------------|:---------------|
| `kInitialRtt`                    | 333 ms                                   | RFC 9002 §6.2  |
| `kPacketThreshold`               | 3                                        | RFC 9002 §6.1  |
| `kTimeThreshold`                 | 9/8 (1.125)                              | RFC 9002 §6.1  |
| `kGranularity`                   | 1 ms                                     | RFC 9002 §6.1  |
| `kPersistentCongestionThreshold` | 3                                        | RFC 9002 §7.6  |
| `kInitialWindow`                 | min(10 * MDS, max(14720, 2 * MDS))       | RFC 9002 §7.2  |
| `kMinimumWindow`                 | 2 * max_datagram_size                    | RFC 9002 §7.2  |
| `kLossReductionFactor`           | 0.5 (NewReno) / 0.7 (CUBIC)              | RFC 9002 §7.3  |
| `kMaxDatagramSize_default`       | 1200 bytes                               | RFC 9000 §14.1 |
| `initial_salt v1`                | 0x38762cf7f55934b34d179ae6a4c80cadccbb7f0a | RFC 9001 §5.2 |
| `max_early_data_size_typical`    | 16384 (16 KiB)                           | implementation |
| `max_idle_timeout_typical`       | 30000 (30 s)                             | implementation |
| `kKeyUpdateDelay`                | 3 RTTs                                   | RFC 9001 §6.5  |
| `kAEAD_AES_128_GCM_max_packets`  | 2^23                                     | RFC 9001 §6.6  |
| `kAEAD_AES_256_GCM_max_packets`  | 2^23                                     | RFC 9001 §6.6  |
| `kAEAD_CHACHA20_POLY1305_max_pkt`| 2^62 (effectively unbounded)             | RFC 9001 §6.6  |

`MDS` = `max_datagram_size`. Implementations enforce key-update or connection-close
when `pkts_per_key` reaches the AEAD's safe limit.

---

## 16. Appendix — QPACK Compression Math

QPACK (RFC 9204) splits header compression into:

1. **Static table** — 99 entries, indexed by varint reference.
2. **Dynamic table** — fixed-size sliding buffer of recent (name, value) pairs.
3. **Encoder stream** — unidirectional stream that the encoder uses to insert into the
   dynamic table.
4. **Decoder stream** — unidirectional stream the decoder uses to acknowledge sections.

### 16.1 Dynamic Table Sizing

```
dynamic_table_size  = sum( name.len + value.len + 32 )       # 32 = overhead per entry
dynamic_table_size <= SETTINGS_QPACK_MAX_TABLE_CAPACITY
```

The 32-byte overhead approximates per-entry pointer/metadata cost (mirrors HPACK §4.1).

### 16.2 Section Acknowledgement and Insert Count

Each header section reference includes a "Required Insert Count" — the highest insert
index the section depends on. The decoder uses this to know when it can begin decoding.

```
RIC = (Required Insert Count - 1) mod (2 * MaxEntries) + 1
MaxEntries = floor(SETTINGS_QPACK_MAX_TABLE_CAPACITY / 32)
```

The wire encoding of RIC is folded modulo `2 * MaxEntries` to fit a smaller varint —
the decoder reconstructs the absolute insert count from its own `Insert Count`.

### 16.3 Out-of-Order Tolerance Cap

`SETTINGS_QPACK_BLOCKED_STREAMS` is the maximum number of streams the decoder will
hold blocked while waiting for missing dynamic table entries:

```
streams_blocked_now <= SETTINGS_QPACK_BLOCKED_STREAMS
```

If exceeded, the encoder MUST avoid producing references that would push another
stream into the blocked set; instead it must duplicate or use literal encoding.

```
Trade-off matrix:

  SETTINGS_QPACK_BLOCKED_STREAMS=0:   no out-of-order; QPACK degenerates toward HPACK; lowest memory.
  SETTINGS_QPACK_BLOCKED_STREAMS=N:   up to N streams may stall in decoder buffer; better compression.
```

---

## 17. Appendix — Initial Packet Hex Dump (Annotated)

A canonical example from RFC 9001 §A.2 of a client's first Initial packet, decoded:

```
Wire bytes (header):
  c3       # first byte: long(1) || fixed(1) || 00(Initial) || 0011(reserved+pn_len-1)
  00 00 00 01      # version = 0x00000001 (QUIC v1)
  08              # DCID Len = 8
  83 94 c8 f0 3e 51 57 08    # DCID
  00              # SCID Len = 0
  00              # Token Length (varint) = 0
  44 9e          # Length = 0x449e = 1182 (varint, prefix=01)
  ...            # Packet Number (encrypted, 1..4 bytes)
  ...            # Payload (encrypted, AEAD-tagged)

Crypto context (Initial keys):
  initial_salt = 38 76 2c f7 f5 59 34 b3 4d 17 9a e6 a4 c8 0c ad cc bb 7f 0a
  PRK = HKDF-Extract(salt=initial_salt, IKM=DCID)
  client_secret = HKDF-Expand-Label(PRK, "client in", "", 32)
  client_key    = HKDF-Expand-Label(client_secret, "quic key", "", 16)
  client_iv     = HKDF-Expand-Label(client_secret, "quic iv",  "", 12)
  client_hp     = HKDF-Expand-Label(client_secret, "quic hp",  "", 16)
```

The receiver derives the same secrets, removes header protection, recovers the
packet number, computes the AEAD nonce as `pn XOR iv`, and decrypts the payload.

---

## 18. Appendix — HTTP Wire Examples

### 18.1 Minimal HTTP/3 GET

```http
# Client sends on a fresh client-initiated bidirectional stream:
HEADERS frame:
    :method = GET
    :scheme = https
    :authority = example.com
    :path = /api/users/42
    user-agent = vor/1.0
    accept = application/json
# Stream FIN follows the HEADERS frame (no body).
```

```http
# Server replies on the same stream:
HEADERS frame:
    :status = 200
    content-type = application/json
    content-length = 42
DATA frame:
    {"id": 42, "name": "alice", "active": true}
# Stream FIN.
```

### 18.2 Minimal HTTP/3 POST with Body

```http
# Client stream (bidi):
HEADERS frame:
    :method = POST
    :scheme = https
    :authority = example.com
    :path = /api/messages
    content-type = application/json
    content-length = 18
DATA frame:
    {"text": "hello"}
# Stream FIN.
```

```http
# Server stream:
HEADERS frame:
    :status = 201
    location = /api/messages/9001
DATA frame:
    {"id": 9001}
# Stream FIN.
```

### 18.3 Server Push (Optional)

```http
# Server pushes a stylesheet alongside an HTML response:
PUSH_PROMISE frame (on request stream 0):
    push_id = 0
    :method = GET
    :scheme = https
    :authority = example.com
    :path = /static/app.css

# Server then opens a server-initiated unidirectional push stream:
Stream type byte: 0x01    # push stream
push_id varint:    0
HEADERS frame:
    :status = 200
    content-type = text/css
    cache-control = max-age=31536000
DATA frame:
    body { ... }
# Stream FIN.
```

If the client has set `MAX_PUSH_ID` to a value `< push_id`, it MUST close the
connection with `H3_ID_ERROR`.

---

## 19. Appendix — State Machine Sketches

### 19.1 Connection-Level State Machine

```python
# Pseudo-code — client side, simplified.

class ConnState:
    INITIAL_KEYS_PENDING  = 0
    HANDSHAKE_PROGRESS    = 1
    HANDSHAKE_CONFIRMED   = 2
    ACTIVE                = 3
    DRAINING              = 4
    CLOSED                = 5

def on_packet(pkt, state):
    if state == ConnState.INITIAL_KEYS_PENDING:
        # received first server Initial; derive Handshake keys via TLS 1.3
        if pkt.is_initial() and contains_serverhello(pkt.payload):
            derive_handshake_keys()
            return ConnState.HANDSHAKE_PROGRESS
    elif state == ConnState.HANDSHAKE_PROGRESS:
        if pkt.is_handshake() and tls_finished_received(pkt.payload):
            send_finished_back()
            derive_app_keys()
            return ConnState.HANDSHAKE_CONFIRMED
    elif state == ConnState.HANDSHAKE_CONFIRMED:
        if pkt.is_app_data() and frame_type(pkt) == HANDSHAKE_DONE:
            discard_handshake_keys()
            return ConnState.ACTIVE
    elif state == ConnState.ACTIVE:
        if frame_type(pkt) in (CONNECTION_CLOSE_TRANSPORT, CONNECTION_CLOSE_APP):
            return ConnState.DRAINING
    elif state == ConnState.DRAINING:
        # No new frames; only respond with CONNECTION_CLOSE if needed.
        if idle_timeout_elapsed() or 3 * PTO_elapsed():
            return ConnState.CLOSED
    return state
```

### 19.2 Stream-Level State Machine (RFC 9000 §3)

```c
// Sending side
typedef enum {
    SEND_READY,        // before STREAM frame sent
    SEND_SEND,         // sending data
    SEND_DATA_SENT,    // FIN sent, waiting for all to be ACKed
    SEND_DATA_RECVD,   // peer ACKed all sent bytes incl. FIN
    SEND_RESET_SENT,   // sent RESET_STREAM
    SEND_RESET_RECVD   // peer ACKed RESET_STREAM
} send_state_t;

// Transitions:
//   READY -> SEND_SEND      on first STREAM frame sent
//   SEND_SEND -> DATA_SENT  on STREAM frame with FIN
//   DATA_SENT -> DATA_RECVD on all bytes ACKed
//   any -> RESET_SENT       on application abort
//   RESET_SENT -> RESET_RECVD on RESET_STREAM ACKed

// Receiving side
typedef enum {
    RECV_RECV,         // accepting STREAM frames
    RECV_SIZE_KNOWN,   // FIN received, final size known
    RECV_DATA_RECVD,   // all bytes received in order
    RECV_DATA_READ,    // application has read all bytes
    RECV_RESET_RECVD,  // received RESET_STREAM
    RECV_RESET_READ    // application notified of reset
} recv_state_t;

// Validity:
//   STREAM frame valid only in RECV_RECV / RECV_SIZE_KNOWN
//   final_size mismatch (different STREAM frames claim different FIN offsets)
//      -> CONNECTION_CLOSE(FINAL_SIZE_ERROR, 0x06)
```

---

## 20. Appendix — Implementation Decision Matrix

For an engineer building or operating a QUIC stack, the choices and their consequences:

| Decision                                | Choice A                  | Choice B                   | Trade-off                                  |
|:----------------------------------------|:--------------------------|:---------------------------|:-------------------------------------------|
| Congestion controller                   | NewReno (default)         | CUBIC / BBRv2              | Reno simple/safe; CUBIC throughput; BBR fairness questions |
| 0-RTT acceptance                        | Disabled                   | Enabled (idempotent only) | latency vs replay attack surface           |
| Anti-replay strategy                    | Single-use tickets        | Time-windowed cache       | ticket bookkeeping vs cache memory         |
| `active_connection_id_limit`            | 2                         | 4–8                       | memory vs migration/unlinkability          |
| `max_datagram_frame_size`               | 0 (off)                   | 1200+                     | feature surface vs simplicity              |
| `SETTINGS_QPACK_MAX_TABLE_CAPACITY`     | 0                         | 4096–16384                | compression vs memory + complexity          |
| `SETTINGS_QPACK_BLOCKED_STREAMS`        | 0                         | 100+                      | compression vs decoder buffering           |
| ECN                                     | Off                       | On (validated)            | path-aware CC vs middlebox compatibility   |
| Pacing                                  | Off                       | On                        | bursty vs smooth; required for high BW     |
| Key update cadence                      | At AEAD limit only        | Every N seconds proactive | safety vs CPU                              |

The default-correct posture for a typical web server: NewReno or CUBIC; 0-RTT
disabled or strictly idempotent-only; QPACK blocked streams = 100; ECN on with
validation; pacing on; key update on AEAD limit + every 24h.

---

## 21. Appendix — Common Errors and Their Wire Causes

| Symptom                                       | Probable wire cause                                       |
|:----------------------------------------------|:----------------------------------------------------------|
| Connection stalls at handshake                | UDP blocked by middlebox; fall back to TCP+TLS            |
| Handshake completes but throughput is low     | PMTU stuck low; DPLPMTUD not converging                   |
| Cliff-edge throughput drop after migration    | Path validation incomplete; cwnd reset; ECN bleach        |
| Frequent CONNECTION_CLOSE(FLOW_CONTROL_ERROR) | MAX_DATA / MAX_STREAM_DATA not extending; receiver bug    |
| 0-RTT data ignored                            | Server's `max_early_data_size = 0` or ticket expired      |
| H3_REQUEST_REJECTED on new requests           | GOAWAY received; reconnect required                       |
| H3_FRAME_UNEXPECTED                           | DATA before HEADERS, or HEADERS on control stream         |
| H3_CLOSED_CRITICAL_STREAM                     | One of the unidirectional control/QPACK streams closed    |
| AEAD_LIMIT_REACHED                            | Same key used past 2^23 packets without key update        |
| KEY_UPDATE_ERROR                              | Key update before handshake confirmed, or wrong direction |

---

## See Also

- `networking/tcp` — comparison transport: ordered-byte abstraction, RTO mechanics, fast retransmit
- `networking/quic` — companion overview for quick reference; this page is the math depth
- `networking/http2` — predecessor transport; the HOL-blocking problem QUIC eliminates
- `security/tls` — TLS 1.3 internals; QUIC-TLS is TLS 1.3 carried in CRYPTO frames
- `ramp-up/tcp-eli5` — narrative kindergarten introduction to the ordering abstraction
- `ramp-up/http3-quic-eli5` — narrative companion to this deep dive

## References

- RFC 9000 — QUIC: A UDP-Based Multiplexed and Secure Transport
- RFC 9001 — Using TLS to Secure QUIC
- RFC 9002 — QUIC Loss Detection and Congestion Control
- RFC 9114 — HTTP/3
- RFC 9204 — QPACK: Field Compression for HTTP/3
- RFC 9221 — An Unreliable Datagram Extension to QUIC
- RFC 9298 — Proxying UDP in HTTP (CONNECT-UDP)
- RFC 9297 — HTTP Datagrams and the Capsule Protocol
- RFC 8446 — The Transport Layer Security (TLS) Protocol Version 1.3
- RFC 5869 — HMAC-based Extract-and-Expand Key Derivation Function (HKDF)
- RFC 8311 — Relaxing Restrictions on Explicit Congestion Notification (ECN) Experimentation
- RFC 8899 — Packetization Layer Path MTU Discovery for Datagram Transports
- RFC 9438 — CUBIC for Fast and Long-Distance Networks
- RFC 9293 — Transmission Control Protocol (TCP) — for HOL-blocking comparison baseline
- RFC 7541 — HPACK: Header Compression for HTTP/2 (for the QPACK contrast)
- RFC 9218 — Extensible Prioritization Scheme for HTTP
