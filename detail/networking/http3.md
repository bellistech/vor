# HTTP/3 Deep Dive -- Stream Architecture, QPACK Internals, and Migration from HTTP/2

> *HTTP/3 maps HTTP semantics onto QUIC's independent stream model, replacing TCP's ordered byte stream with per-request isolation. This eliminates head-of-line blocking at the transport layer, but requires a new header compression scheme (QPACK) and fundamentally different stream management. Understanding these internals is critical for performance tuning and deployment planning.*

---

## 1. Stream Mapping in HTTP/3

### Stream Roles

HTTP/3 assigns specific roles to QUIC streams. Unlike HTTP/2, which multiplexes all frames onto a single TCP connection with a frame-level multiplexer, HTTP/3 delegates multiplexing entirely to QUIC.

| Stream Type | QUIC Stream Kind | Initiator | Count | Purpose |
|:---|:---|:---|:---|:---|
| Request stream | Bidirectional | Client | One per request | Carries HEADERS + DATA for a single request/response |
| Control stream | Unidirectional | Both | Exactly one per direction | Carries SETTINGS, GOAWAY, MAX_PUSH_ID |
| QPACK encoder stream | Unidirectional | Both | One per direction | Sends dynamic table insertions to peer |
| QPACK decoder stream | Unidirectional | Both | One per direction | Sends acknowledgments back to encoder |
| Push stream | Unidirectional | Server | Zero or more | Carries pushed responses |

### Stream Identification

```
# QUIC stream IDs encode type and initiator in the low 2 bits:
#   0x0  Client-initiated bidirectional   (request streams)
#   0x1  Server-initiated bidirectional   (not used by HTTP/3)
#   0x2  Client-initiated unidirectional  (control, QPACK encoder/decoder)
#   0x3  Server-initiated unidirectional  (control, QPACK encoder/decoder, push)

# Unidirectional streams carry a stream type byte as their first payload:
#   0x00  Control stream
#   0x01  Push stream
#   0x02  QPACK encoder stream
#   0x03  QPACK decoder stream

# The control stream, QPACK encoder stream, and QPACK decoder stream
# must each be opened exactly once. Opening a second stream of any of
# these types is a connection error (H3_STREAM_CREATION_ERROR).
```

### Request Stream Lifecycle

```
# A single HTTP request/response occupies one bidirectional QUIC stream:

Client                                        Server
  |                                             |
  |  -- HEADERS frame (request headers) ----->  |
  |  -- DATA frame(s) (request body) -------->  |  optional
  |  -- (stream FIN if no trailers) --------->  |
  |                                             |
  |  <--- HEADERS frame (response headers) --   |
  |  <--- DATA frame(s) (response body) -----   |
  |  <--- HEADERS frame (trailers) ----------   |  optional
  |  <--- (stream FIN) ----------------------   |
  |                                             |

# Key differences from HTTP/2:
# - No stream ID negotiation — QUIC handles stream creation
# - No WINDOW_UPDATE — QUIC flow control is automatic
# - No RST_STREAM — use QUIC RESET_STREAM and STOP_SENDING
# - No PRIORITY frame — replaced by Priority header field (RFC 9218)
# - Closing the stream (FIN) terminates the request, not a special frame
```

---

## 2. QPACK Encoder and Decoder Streams

### Why HPACK Cannot Work Over QUIC

HPACK (RFC 7541) maintains a dynamic table that both encoder and decoder update synchronously. Each header block references or modifies the table, and the decoder must process header blocks in exactly the order they were encoded. This strict ordering requirement is satisfied by TCP's in-order delivery.

Over QUIC, request streams are delivered independently. If request stream 4 references a dynamic table entry added by request stream 0, but stream 0's headers have not yet arrived, the decoder cannot proceed. Using HPACK directly would reintroduce head-of-line blocking -- the very problem HTTP/3 is designed to solve.

### QPACK Architecture

QPACK solves this with two dedicated unidirectional streams that operate independently of request streams:

```
Encoder side (e.g., client):
  Request stream 0:  HEADERS [refs to static table + known dynamic entries]
  Request stream 4:  HEADERS [refs to static table + known dynamic entries]
  Encoder stream:    INSERT name=foo value=bar
                     INSERT name=baz value=qux
                     DUPLICATE index=3

Decoder side (e.g., server):
  Decoder stream:    Section Acknowledgment (stream 0)
                     Section Acknowledgment (stream 4)
                     Insert Count Increment (2)
```

### Static Table

QPACK's static table contains 99 entries (compared to HPACK's 61). It includes commonly used HTTP/3 headers and status codes:

| Index | Name | Value |
|:---:|:---|:---|
| 0 | :authority | (empty) |
| 1 | :path | / |
| 15 | :method | GET |
| 16 | :method | POST |
| 24 | :status | 200 |
| 25 | :status | 304 |
| 27 | :status | 404 |
| 63 | content-type | application/json |
| 71 | origin | (empty) |
| 80 | x-content-type-options | nosniff |

### Dynamic Table Synchronization

```
# The encoder tracks which dynamic table entries the decoder has acknowledged.
# Header blocks may only reference entries the decoder is known to have received.

# Required Insert Count (RIC):
#   Each header block declares the minimum number of dynamic table insertions
#   the decoder needs to have processed before it can decode the block.
#   If the decoder has not yet received that many insertions, it blocks.

# SETTINGS_QPACK_BLOCKED_STREAMS:
#   Maximum number of streams that can be simultaneously blocked waiting
#   for QPACK dynamic table updates.
#   Set to 0: no blocking allowed (only static table + literals)
#   Set to 100: up to 100 streams can wait for table updates

# Tradeoff:
#   Higher blocked streams limit = better compression (more dynamic refs)
#   Lower blocked streams limit = less HOL blocking risk
#   In practice, a value of 100 provides good compression with negligible blocking
```

### Encoder Instructions

```
# Sent on the QPACK encoder stream:

# 1. Insert With Name Reference
#    Reference an existing entry's name, provide new value
#    Format: 1SNNNNNN [name_index] H value
#    S=1: static table reference, S=0: dynamic table reference

# 2. Insert With Literal Name
#    Provide both name and value as literals
#    Format: 01 H name H value

# 3. Duplicate
#    Copy an existing dynamic table entry (resets eviction order)
#    Format: 000 index

# 4. Set Dynamic Table Capacity
#    Change the maximum size of the dynamic table
#    Format: 001 capacity
```

### Decoder Instructions

```
# Sent on the QPACK decoder stream:

# 1. Section Acknowledgment
#    Tells encoder that a header block on a given stream was decoded
#    Format: 1 stream_id
#    Effect: encoder knows all entries referenced by that block are safe

# 2. Stream Cancellation
#    Tells encoder that a stream was abandoned (e.g., request cancelled)
#    Format: 01 stream_id
#    Effect: encoder will not count that block as acknowledged

# 3. Insert Count Increment
#    Tells encoder that decoder has processed N new insertions
#    Format: 00 increment
#    Effect: encoder can reference those entries in new header blocks
```

---

## 3. Head-of-Line Blocking Comparison: HTTP/1.1 vs HTTP/2 vs HTTP/3

### HTTP/1.1: Connection-Level Blocking

In HTTP/1.1, each TCP connection handles one request at a time. Responses must be sent in the same order as requests (no out-of-order delivery). A slow response blocks all subsequent responses on that connection.

```
# Scenario: 4 requests on 1 connection, request 2 takes 500ms

Timeline:
  0ms     100ms    200ms    300ms    400ms    500ms    600ms    700ms
  |--------|--------|--------|--------|--------|--------|--------|
  [  req 1 (50ms)  ]
                    [           req 2 (500ms)                    ]
                                                                  [ req 3 ]
                                                                           [ req 4 ]

# req 3 and req 4 wait for req 2 even though the server has their responses ready
# Mitigation: browsers open 6 connections per origin (but 6x TCP+TLS overhead)
```

### HTTP/2: Transport-Level Blocking

HTTP/2 multiplexes all requests onto a single TCP connection. The application layer can interleave frames from different streams, but TCP delivers bytes strictly in order. A single lost TCP segment blocks all streams.

```
# Scenario: 4 concurrent streams, packet carrying stream 2 data is lost

Application layer sees:
  Stream 1: [HEADERS][DATA][DATA]        -- has data ready
  Stream 2: [HEADERS][DATA]___[DATA]     -- lost segment
  Stream 3: [HEADERS][DATA][DATA]        -- has data ready
  Stream 4: [HEADERS][DATA]              -- has data ready

TCP layer delivers:
  All streams blocked until stream 2's lost segment is retransmitted

# Impact: with 1% loss and 50 concurrent streams
# P(at least one stream blocked) = 1 - (0.99)^50 = 39.5%
# When blocked, ALL 50 streams stall for ~1 RTT (retransmission time)
```

### HTTP/3: No Transport-Level Blocking

Each HTTP/3 request uses an independent QUIC stream. Packet loss on one stream does not affect others because QUIC maintains separate reassembly buffers per stream.

```
# Same scenario: 4 concurrent streams, packet carrying stream 2 data is lost

QUIC delivers:
  Stream 1: [HEADERS][DATA][DATA]        -- delivered immediately
  Stream 2: [HEADERS][DATA]___[DATA]     -- blocked, waiting for retransmit
  Stream 3: [HEADERS][DATA][DATA]        -- delivered immediately
  Stream 4: [HEADERS][DATA]              -- delivered immediately

# Only stream 2 is blocked. Streams 1, 3, 4 proceed without delay.

# Impact: with 1% loss and 50 concurrent streams
# Expected blocked streams = 50 * 0.01 = 0.5 streams
# Other 49.5 streams are unaffected
# Per-stream blocking probability: 1% (independent of stream count)
```

### Quantitative Comparison

| Metric | HTTP/1.1 (6 conns) | HTTP/2 | HTTP/3 |
|:---|:---:|:---:|:---:|
| Loss rate | 1% | 1% | 1% |
| Concurrent requests | 6 (one per conn) | 50 | 50 |
| P(any blocking event) | 5.9% per conn | 39.5% | 1% per stream |
| Streams affected per event | 1 | All 50 | 1 |
| Average delay per event | 1 RTT | 1 RTT (all streams) | 1 RTT (one stream) |
| Connections needed | 6 per origin | 1 | 1 |
| Handshake overhead | 6 x (TCP + TLS) | 1 x (TCP + TLS) | 1 x QUIC |

---

## 4. Connection Coalescing

HTTP/3 (like HTTP/2) supports connection coalescing: reusing a single connection for multiple origins that meet specific criteria.

### Requirements for Coalescing

```
# A client MAY reuse an HTTP/3 connection for a different origin if:
# 1. The new origin resolves to the same IP address as the existing connection
# 2. The TLS certificate on the existing connection is valid for the new origin
#    (via Subject Alternative Name or wildcard)
# 3. The new origin uses the same port

# Example:
#   Connection established to cdn.example.com (IP: 203.0.113.1)
#   Certificate SAN includes: *.example.com
#   Client can reuse this connection for:
#     - www.example.com (if it resolves to 203.0.113.1)
#     - api.example.com (if it resolves to 203.0.113.1)
#   Client cannot reuse for:
#     - other.example.net (different domain, not in SAN)
#     - www.example.com (if it resolves to 203.0.113.2 -- different IP)
```

### Benefits and Risks

```
# Benefits:
# - Fewer connections = fewer handshakes = faster page loads
# - Single congestion controller = better bandwidth utilization
# - Reduced server resource consumption (memory, file descriptors)

# Risks:
# - Single connection failure affects all coalesced origins
# - Certificate management complexity (must cover all coalesced domains)
# - Privacy implications (server sees requests for all origins on one connection)
# - Middleboxes may not expect traffic for multiple SNIs on one connection
```

---

## 5. QUIC Version Negotiation

### Version Negotiation Packet

```
# If a server receives a QUIC packet with a version it does not support,
# it responds with a Version Negotiation packet:

Client -> Server:  Initial packet (Version: 0x00000002)
Server -> Client:  Version Negotiation packet
                   Supported Versions: [0x00000001]

# The Version Negotiation packet is NOT authenticated
# It is vulnerable to downgrade attacks
# QUIC addresses this with the version negotiation extension (RFC 9369)
```

### Compatible Version Negotiation (RFC 9369)

```
# RFC 9369 defines authenticated version negotiation:
# 1. Client sends Initial with chosen version + version_information transport param
# 2. Server selects compatible version and includes its own version_information
# 3. Both sides verify the negotiation was not tampered with
# 4. No extra round trip required (piggybacks on handshake)

# Key concept: "compatible versions"
# Two QUIC versions are compatible if they use the same handshake structure
# A server can switch to a compatible version without restarting the handshake

# Currently deployed versions:
# 0x00000001  QUIC v1 (RFC 9000)
# 0x6b3343cf  QUIC v2 (RFC 9369) — identical wire format, different initial salts
```

---

## 6. Performance Measurements

### Handshake Latency

| Scenario | HTTP/2 (TCP+TLS 1.3) | HTTP/3 (QUIC) | Improvement |
|:---|:---:|:---:|:---:|
| New connection, 20ms RTT | 40 ms | 20 ms | 50% |
| New connection, 100ms RTT | 200 ms | 100 ms | 50% |
| New connection, 200ms RTT | 400 ms | 200 ms | 50% |
| Resumed, 20ms RTT | 20 ms | 0 ms | 100% |
| Resumed, 100ms RTT | 100 ms | 0 ms | 100% |
| Resumed, 200ms RTT | 200 ms | 0 ms | 100% |

### Page Load Time Impact

Real-world measurements vary significantly based on page complexity, network conditions, and server configuration. General findings from published studies:

```
# Low-latency, low-loss networks (datacenter, wired broadband):
#   HTTP/3 improvement over HTTP/2: 0-5%
#   Marginal benefit — TCP performs well in these conditions

# Moderate-latency networks (typical mobile, 50-100ms RTT):
#   HTTP/3 improvement: 5-15% for multi-resource pages
#   Primary benefit: reduced handshake time + reduced HOL blocking

# High-loss networks (congested WiFi, poor cellular, >2% loss):
#   HTTP/3 improvement: 15-40%
#   HOL blocking elimination provides largest benefit here
#   Benefit scales with number of concurrent streams

# Connection migration scenarios (WiFi to cellular):
#   HTTP/2: full reconnect (handshake + TLS) = 200-600ms interruption
#   HTTP/3: seamless migration = 0-50ms interruption
```

### Bandwidth Utilization

```
# QUIC has slightly higher per-packet overhead than TCP:
#   TCP header:  20 bytes (minimum)
#   UDP header:  8 bytes
#   QUIC header: 20-30 bytes (short header, including connection ID + packet number)
#   QUIC crypto: 16 bytes AEAD tag per packet

# Total per-packet overhead:
#   TCP + TLS 1.3: ~20 + 5 (TLS record) + 16 (AEAD) = ~41 bytes
#   QUIC:          8 (UDP) + 25 (QUIC avg) + 16 (AEAD) = ~49 bytes

# Difference: ~8 bytes per packet
# At 1200-byte packets: 0.7% overhead increase (negligible for most workloads)
# At high packet rates: may matter for bulk transfer benchmarks

# Note: QUIC implementations often use GSO/GRO to amortize system call
# overhead, achieving comparable or better throughput than TCP on Linux
```

---

## 7. Migration from HTTP/2 -- Deployment Considerations

### Server-Side Requirements

```
# 1. QUIC-capable server or reverse proxy
#    Options: nginx 1.25+ (quic module), Caddy (built-in), HAProxy 2.6+,
#    Envoy, Cloudflare/AWS/GCP edge (managed)

# 2. UDP port 443 must be open
#    Many firewalls only allow TCP 443 — update rules to include UDP 443
#    Load balancers must support UDP or terminate QUIC

# 3. TLS 1.3 certificate (same certificate works for both HTTP/2 and HTTP/3)
#    QUIC mandates TLS 1.3 — TLS 1.2 is not supported

# 4. Alt-Svc header or DNS HTTPS record
#    Must advertise HTTP/3 availability to clients
#    Without advertisement, clients will never attempt QUIC

# 5. Increased UDP buffer sizes on Linux
#    sysctl net.core.rmem_max=2500000
#    sysctl net.core.wmem_max=2500000
#    Default UDP buffers are too small for sustained QUIC throughput
```

### Load Balancer Considerations

```
# HTTP/2 load balancing: straightforward (TCP 4-tuple based)
# HTTP/3 load balancing: requires QUIC awareness

# Challenge: QUIC connection migration changes source IP/port
# TCP-based LB routes by (src IP, src port, dst IP, dst port)
# After migration, the 4-tuple changes but connection ID stays the same

# Solutions:
# 1. Terminate QUIC at the load balancer
#    LB decrypts QUIC, forwards as HTTP/2 or HTTP/1.1 to backends
#    Simplest but adds latency and loses migration benefit end-to-end

# 2. QUIC-aware load balancer (route by Connection ID)
#    LB reads the CID from the UDP packet header (unencrypted)
#    Routes to the same backend regardless of source IP change
#    Requires: CID encoding scheme that embeds server ID
#    See: draft-ietf-quic-load-balancers

# 3. Anycast + consistent hashing
#    Works for CDN-style deployments where servers are stateless
#    Connection migration may land on a different server (connection loss)
```

### Monitoring and Observability

```
# HTTP/2 (TCP): standard tools work (tcpdump, Wireshark, netstat)
# HTTP/3 (QUIC): encrypted transport makes passive monitoring difficult

# Challenges:
# - Packet payload is encrypted — no deep packet inspection
# - Connection IDs rotate — harder to track connections across time
# - UDP means no TCP state to monitor (no SYN/FIN/RST)
# - Standard flow exporters (NetFlow/IPFIX) treat it as "UDP traffic"

# Solutions:
# - qlog (structured event logging from QUIC stack)
# - SSLKEYLOGFILE for Wireshark decryption in test environments
# - Application-level logging (access logs still work normally)
# - QUIC-aware monitoring (some CDN dashboards break out h3 vs h2)
# - Prometheus/metrics from server's QUIC implementation
```

### Incremental Rollout Strategy

```
# Recommended migration path:

# Phase 1: Add QUIC/HTTP/3 support alongside HTTP/2
#   - Deploy HTTP/3 on edge servers or CDN
#   - Add Alt-Svc header: h3=":443"; ma=3600  (short max-age initially)
#   - Monitor: error rates, latency percentiles, fallback frequency
#   - Keep HTTP/2 as primary — HTTP/3 is opportunistic upgrade

# Phase 2: Increase Alt-Svc max-age, add DNS HTTPS records
#   - Extend ma= to 86400 (24 hours) once stable
#   - Add HTTPS DNS record for faster discovery
#   - Monitor: percentage of traffic on HTTP/3 vs HTTP/2
#   - Typical: 25-40% of traffic migrates to HTTP/3

# Phase 3: Optimize QUIC configuration
#   - Tune 0-RTT (enable for safe endpoints)
#   - Adjust QPACK dynamic table size
#   - Configure connection migration support
#   - Enable ECN if network path supports it

# Phase 4: Steady state
#   - HTTP/3 for clients that support it (majority of browsers)
#   - HTTP/2 fallback for restricted networks
#   - Never remove HTTP/2 — UDP blocking is common enough to matter

# What NOT to do:
# - Do not deploy HTTP/3-only (breaks 10-30% of clients)
# - Do not disable HTTP/2 after adding HTTP/3
# - Do not assume all load balancers handle QUIC transparently
# - Do not skip UDP firewall rules (most common deployment failure)
```

### Configuration Differences from HTTP/2

```
# Settings that do NOT carry over to HTTP/3:
#   SETTINGS_MAX_CONCURRENT_STREAMS  -> QUIC MAX_STREAMS transport param
#   SETTINGS_INITIAL_WINDOW_SIZE     -> QUIC initial_max_stream_data
#   SETTINGS_MAX_FRAME_SIZE          -> No equivalent (QUIC packets, not frames)
#   SETTINGS_ENABLE_PUSH             -> MAX_PUSH_ID frame (0 = disabled)

# Settings that are new in HTTP/3:
#   SETTINGS_QPACK_MAX_TABLE_CAPACITY   Controls QPACK dynamic table size
#   SETTINGS_QPACK_BLOCKED_STREAMS      Max streams blocked on QPACK updates
#   SETTINGS_MAX_FIELD_SECTION_SIZE     Max header block size (same concept as HTTP/2)

# HTTP/2 features removed from HTTP/3:
#   PRIORITY frame     -> Replaced by Priority header field (RFC 9218)
#   WINDOW_UPDATE      -> Handled by QUIC flow control
#   PING frame         -> Handled by QUIC PING
#   Padding            -> Not needed (QUIC encrypts packet length)
```

---

## 8. Summary

| Aspect | HTTP/2 | HTTP/3 |
|:---|:---|:---|
| Transport | TCP + TLS | QUIC (UDP + integrated TLS 1.3) |
| HOL blocking | Transport-level (TCP) | Per-stream only (QUIC) |
| Header compression | HPACK (ordered) | QPACK (unordered, stream-safe) |
| Flow control | HTTP/2 WINDOW_UPDATE | QUIC transport layer |
| Connection identity | TCP 4-tuple | QUIC Connection ID |
| Migration | Not supported | Seamless (CID-based) |
| 0-RTT | TLS 1.3 early data (limited) | Native QUIC 0-RTT |
| Deployment barrier | Low (TCP everywhere) | Moderate (UDP often blocked) |
| Monitoring | Standard TCP tools | Requires QUIC-aware tooling |

## Prerequisites

- TCP/IP fundamentals, TLS 1.3 handshake, HTTP/2 framing, HPACK compression

---

*HTTP/3 is not a revolutionary change in HTTP semantics -- it is the same request/response model mapped onto a transport layer that was designed from scratch to avoid the problems TCP accumulated over four decades. The complexity lives in the transport (QUIC), not the application protocol. Understanding the stream mapping and QPACK synchronization model is the key to effective deployment and debugging.*
