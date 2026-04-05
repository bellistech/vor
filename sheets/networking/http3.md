# HTTP/3

The third major version of HTTP, running over QUIC (UDP-based transport) instead of TCP. Eliminates head-of-line blocking at the transport layer, provides built-in encryption, connection migration, and faster handshakes via 0-RTT resumption.

## Overview

```
# HTTP/3 at a glance
Transport:          QUIC over UDP (typically port 443)
Encryption:         TLS 1.3 (mandatory, integrated into QUIC)
Multiplexing:       Independent QUIC streams per request (no HOL blocking)
Header compression: QPACK (RFC 9204), replaces HPACK
Handshake:          1-RTT (new), 0-RTT (resumption)
Server push:        Supported but rarely used (same as HTTP/2)
Standardized:       RFC 9114 (HTTP/3), RFC 9204 (QPACK)
Fallback:           HTTP/2 over TCP via Alt-Svc + Happy Eyeballs
```

## QUIC Transport vs HTTP/3 Application Layer

```
# HTTP/3 is NOT the same as QUIC
# QUIC = transport protocol (RFC 9000) — handles connections, streams, encryption
# HTTP/3 = application protocol (RFC 9114) — maps HTTP semantics onto QUIC streams

# Layer comparison:
#
#   HTTP/2 stack              HTTP/3 stack
#   ──────────────            ──────────────
#   HTTP/2 framing            HTTP/3 framing
#   TLS 1.2/1.3               (integrated)
#   TCP                       QUIC
#   IP                        UDP / IP

# QUIC provides: streams, flow control, loss recovery, encryption
# HTTP/3 provides: request/response mapping, header compression, server push
```

## HTTP Version Comparison

```
Feature                     HTTP/1.1         HTTP/2           HTTP/3
──────────────────────────────────────────────────────────────────────────
Transport                   TCP              TCP              QUIC (UDP)
Encryption                  Optional (TLS)   Optional (TLS)   Always (TLS 1.3)
Multiplexing                No (pipelining)  Yes (streams)    Yes (QUIC streams)
HOL blocking                Per-connection   Transport-level  None (per-stream)
Header compression          None             HPACK            QPACK
Handshake (new)             2-3 RTT          2 RTT            1 RTT
Handshake (resume)          1-2 RTT          1 RTT            0 RTT
Connection migration        No               No               Yes
Server push                 No               Yes              Yes (rarely used)
```

## Head-of-Line Blocking

```
# HTTP/1.1: Each connection carries one request at a time
#   Lost packet blocks everything on that connection
#   Workaround: open 6+ parallel TCP connections

# HTTP/2: Multiple streams share one TCP connection
#   Lost TCP segment blocks ALL streams (TCP guarantees ordered delivery)
#   A single lost packet stalls every in-flight request

# HTTP/3: Each request uses an independent QUIC stream
#   Lost packet on stream N only blocks stream N
#   Other streams continue receiving data immediately
#   This is the primary performance benefit of HTTP/3
```

## Stream Types

```
# HTTP/3 uses QUIC streams in specific roles:

# Request streams (client-initiated bidirectional)
#   One per HTTP request/response exchange
#   Client sends HEADERS + DATA, server responds with HEADERS + DATA

# Control stream (unidirectional, one per direction)
#   Carries SETTINGS, GOAWAY, MAX_PUSH_ID frames
#   Must be the first unidirectional stream opened by each peer
#   Lasts the lifetime of the connection

# QPACK encoder stream (unidirectional)
#   Sends dynamic table updates from encoder to decoder

# QPACK decoder stream (unidirectional)
#   Sends acknowledgments from decoder to encoder

# Push streams (server-initiated unidirectional)
#   Carry server push responses (PUSH_PROMISE on request stream,
#   then pushed response on push stream)
```

## QPACK Header Compression

```
# QPACK (RFC 9204) replaces HTTP/2's HPACK

# Why not HPACK?
#   HPACK requires strictly ordered delivery of header blocks
#   TCP guarantees this; QUIC streams do not
#   If HPACK were used over QUIC, streams would need to wait
#   for earlier header blocks — reintroducing HOL blocking

# QPACK design:
#   Static table:   99 pre-defined header field entries
#   Dynamic table:  Built incrementally via encoder instructions
#   Encoder stream: Sends table updates (insertions, duplications)
#   Decoder stream: Sends acknowledgments (section acks, stream cancellations)

# Key difference from HPACK:
#   QPACK allows referencing only entries acknowledged by the decoder
#   This means header blocks can be decoded independently
#   No cross-stream ordering dependency
```

## Frame Types

```
# HTTP/3 frames (carried on QUIC streams)

Frame Type        ID    Stream Type       Purpose
──────────────────────────────────────────────────────────────────
DATA              0x00  Request           Carries request/response body
HEADERS           0x01  Request           Carries header block (QPACK-encoded)
CANCEL_PUSH       0x03  Control           Cancels a server push before delivery
SETTINGS          0x04  Control           Connection-level configuration
PUSH_PROMISE      0x05  Request           Announces a server push
GOAWAY            0x07  Control           Graceful connection shutdown
MAX_PUSH_ID       0x0D  Control           Limits push stream IDs server can use

# Notable differences from HTTP/2:
# - No WINDOW_UPDATE (QUIC handles flow control natively)
# - No PING (QUIC has its own PING frame)
# - No RST_STREAM (QUIC RESET_STREAM replaces it)
# - No PRIORITY (removed from HTTP/3; see RFC 9218 for extensible priorities)
# - Frame types are on different stream types than HTTP/2
```

## SETTINGS Frame Parameters

```
# HTTP/3 SETTINGS (sent on control stream, once per connection)

SETTINGS_MAX_FIELD_SECTION_SIZE   0x06   Max header block size (bytes)
SETTINGS_QPACK_MAX_TABLE_CAPACITY 0x01  Max QPACK dynamic table size
SETTINGS_QPACK_BLOCKED_STREAMS   0x07   Max streams that can be blocked
                                         waiting for QPACK table updates

# No SETTINGS_ENABLE_PUSH — push is always available unless MAX_PUSH_ID = 0
# No flow control settings — handled at QUIC layer
# No SETTINGS_MAX_CONCURRENT_STREAMS — use QUIC MAX_STREAMS
```

## Alt-Svc Advertisement

```
# Servers advertise HTTP/3 support via the Alt-Svc header or frame
# Clients discover HTTP/3 only after an initial HTTP/1.1 or HTTP/2 response

# HTTP response header method:
Alt-Svc: h3=":443"; ma=86400

# h3        = HTTP/3 protocol identifier (ALPN token)
# :443      = port (same port or different)
# ma=86400  = max-age in seconds (client caches this for 24 hours)

# DNS HTTPS record method (avoids first-request penalty):
# _https.example.com  IN  HTTPS  1 . alpn="h3,h2" port="443"

# ALPN negotiation:
# During QUIC handshake, client offers alpn=h3
# Server selects h3 if supported

# Browser behavior:
# 1. First visit: connects via HTTP/2 over TCP
# 2. Receives Alt-Svc: h3=":443"
# 3. Next request: races QUIC vs TCP (Happy Eyeballs v2)
# 4. If QUIC succeeds, uses HTTP/3 going forward
# 5. If QUIC fails (blocked UDP), falls back to HTTP/2
```

## Connection Migration

```
# QUIC connections are identified by Connection IDs, not IP:port tuples
# When a client's network changes (WiFi -> cellular), the connection survives

# How it works:
# 1. Client moves to new network, gets new IP address
# 2. Client sends QUIC packet with same Connection ID from new IP
# 3. Server validates via PATH_CHALLENGE / PATH_RESPONSE
# 4. Connection continues — no new handshake, no lost requests

# HTTP/3 benefits:
# - In-flight requests are not interrupted
# - No reconnect + re-authentication overhead
# - Background downloads survive network transitions
# - Especially valuable for mobile users
```

## 0-RTT Resumption

```
# QUIC supports sending application data in the first flight (0-RTT)
# Client uses a previously cached session ticket and transport parameters

# Flow:
# 1. Client caches server's session ticket (from previous connection)
# 2. On reconnect, client sends ClientHello + 0-RTT data in first packet
# 3. Server processes 0-RTT data immediately (no round-trip wait)

# Security constraints:
# - 0-RTT data is NOT forward secret (uses previous session key)
# - 0-RTT data is replayable by an attacker
# - Only safe for idempotent requests (GET, HEAD)
# - Server must reject non-idempotent 0-RTT requests
# - Server should implement anti-replay mechanisms (RFC 8446, Section 8)

# HTTP/3 0-RTT example flow:
#   Client -> Server: QUIC Initial + 0-RTT [GET /index.html]
#   Server -> Client: Handshake + 1-RTT [200 OK, body...]
#   Total: application data arrives in 0 round trips
```

## Server Push in HTTP/3

```
# Server push allows sending responses before the client requests them
# Mechanism differs from HTTP/2 due to stream model

# HTTP/3 push flow:
# 1. Server sends PUSH_PROMISE frame on a request stream
#    (contains promised request headers)
# 2. Server opens a new push stream (unidirectional)
#    and sends the pushed response
# 3. Client can cancel with CANCEL_PUSH frame (by Push ID)

# Client controls:
# - Send MAX_PUSH_ID=0 on control stream to disable push entirely
# - CANCEL_PUSH to reject specific pushes
# - Most browsers disable server push by default

# In practice:
# Server push is rarely used in HTTP/2 or HTTP/3
# 103 Early Hints is generally preferred for preloading
```

## Fallback to HTTP/2

```
# HTTP/3 requires UDP; many networks block or deprioritize UDP traffic
# Browsers implement automatic fallback:

# 1. Client tries QUIC (UDP 443) and TCP (TCP 443) in parallel
# 2. If QUIC connects first, uses HTTP/3
# 3. If TCP connects first (or QUIC fails), uses HTTP/2
# 4. Cached Alt-Svc entries are cleared after persistent QUIC failures

# Network conditions that trigger fallback:
# - Corporate firewalls blocking UDP 443
# - ISPs throttling or dropping UDP
# - NAT devices with short UDP timeouts
# - VPNs that only tunnel TCP

# Server-side: always serve both HTTP/2 (TCP) and HTTP/3 (QUIC)
# Never deploy HTTP/3-only — significant fraction of clients cannot reach UDP
```

## curl Examples

```bash
# Basic HTTP/3 request (prefer HTTP/3, fall back to HTTP/2)
curl --http3 https://example.com

# Strict HTTP/3 only (fail if HTTP/3 not available)
curl --http3-only https://example.com

# Verbose output showing QUIC handshake details
curl -v --http3 https://example.com

# Show which HTTP version was actually used
curl --http3 -w '%{http_version}\n' -o /dev/null -s https://example.com
# Output: 3 (for HTTP/3), 2 (for HTTP/2)

# Check if your curl build supports HTTP/3
curl --version | grep -i http3
# Look for: HTTP3 in Features line
# curl must be built with a QUIC backend (quiche, ngtcp2, or msh3)

# Download with HTTP/3 and show timing
curl --http3 -o /dev/null -s -w \
  'connect: %{time_connect}s\nttfb: %{time_starttransfer}s\ntotal: %{time_total}s\n' \
  https://example.com

# Check if server advertises HTTP/3 (look for Alt-Svc header)
curl -sI https://example.com | grep -i alt-svc
```

## Browser Support

```
# As of 2025, HTTP/3 is supported by all major browsers:

Browser           Support Status
──────────────────────────────────────────────
Chrome/Chromium   Enabled by default since Chrome 87 (Nov 2020)
Firefox           Enabled by default since Firefox 88 (Apr 2021)
Safari            Enabled by default since Safari 14 (macOS Big Sur)
Edge              Enabled by default (Chromium-based)
Opera             Enabled by default (Chromium-based)

# Check current protocol in browser:
# Chrome:  DevTools -> Network tab -> Protocol column (shows "h3")
# Firefox: DevTools -> Network tab -> hover request -> "HTTP Version"

# Disable HTTP/3 in browser (for debugging):
# Chrome:  chrome://flags/#enable-quic -> Disabled
# Firefox: about:config -> network.http.http3.enable -> false
```

## Performance Benefits over HTTP/2

```
# Primary performance improvements:

# 1. Faster connection establishment
#    HTTP/2: TCP handshake (1 RTT) + TLS handshake (1 RTT) = 2 RTT
#    HTTP/3: QUIC handshake (1 RTT, crypto included) = 1 RTT
#    Resumption: HTTP/3 achieves 0-RTT, HTTP/2 still needs 1 RTT minimum

# 2. No transport-level HOL blocking
#    HTTP/2: single lost TCP segment stalls all streams
#    HTTP/3: loss on one stream does not affect others
#    Impact scales with loss rate and concurrent stream count

# 3. Connection migration
#    HTTP/2: network change = new TCP connection = full re-handshake
#    HTTP/3: seamless transition, in-flight requests survive

# 4. Better loss recovery
#    QUIC packet numbers are monotonically increasing (no ambiguity)
#    More accurate RTT measurements for faster retransmission

# Where HTTP/3 helps most:
# - High-latency links (satellite, mobile)
# - Lossy networks (WiFi, cellular)
# - Mobile users changing networks frequently
# - Pages loading many small resources concurrently

# Where HTTP/3 advantage is minimal:
# - Low-latency, low-loss datacenter connections
# - Single large file downloads (one stream, limited benefit)
# - Networks that throttle or block UDP
```

## Tips

- Always deploy HTTP/2 alongside HTTP/3. A significant fraction of networks still block UDP 443, and browsers will silently fall back. Use the Alt-Svc header to advertise HTTP/3 availability.
- QPACK's SETTINGS_QPACK_BLOCKED_STREAMS controls how many request streams can block waiting for dynamic table updates. Setting it to 0 avoids blocking entirely but reduces compression efficiency. A value of 100 is typical.
- The QUIC transport layer handles flow control, not HTTP/3. Do not look for WINDOW_UPDATE or SETTINGS_MAX_CONCURRENT_STREAMS in HTTP/3 -- those are QUIC-level concerns (MAX_STREAM_DATA, MAX_STREAMS).
- 0-RTT is powerful but dangerous for non-idempotent requests. POST requests should never be sent as 0-RTT data because attackers can replay the initial packet. Servers should reject 0-RTT POST at the application layer.
- HTTP/3 server push exists but is disabled by default in most browsers. Prefer 103 Early Hints for resource preloading instead.
- When debugging HTTP/3 issues, use SSLKEYLOGFILE to export TLS keys and Wireshark (3.x+) with the QUIC dissector. The qvis tool can visualize qlog traces for detailed stream and congestion analysis.
- DNS HTTPS records (SVCB/HTTPS RR) allow clients to discover HTTP/3 support without the Alt-Svc round trip. Deploy these alongside Alt-Svc headers for the fastest upgrade path.
- Connection coalescing allows a browser to reuse a single HTTP/3 connection for multiple origins that resolve to the same IP and share a TLS certificate. This reduces total connection count but requires careful certificate management.

## See Also

- http, http2, quic

## References

- [RFC 9114 -- HTTP/3](https://www.rfc-editor.org/rfc/rfc9114)
- [RFC 9204 -- QPACK: Field Compression for HTTP/3](https://www.rfc-editor.org/rfc/rfc9204)
- [RFC 9000 -- QUIC: A UDP-Based Multiplexed and Secure Transport](https://www.rfc-editor.org/rfc/rfc9000)
- [RFC 9218 -- Extensible Prioritization Scheme for HTTP](https://www.rfc-editor.org/rfc/rfc9218)
- [RFC 8446 -- TLS 1.3 (0-RTT and Anti-Replay)](https://www.rfc-editor.org/rfc/rfc8446)
- [curl -- HTTP/3 Support](https://curl.se/docs/http3.html)
- [Cloudflare -- HTTP/3: From Root to Tip](https://blog.cloudflare.com/http3-the-past-present-and-future/)
- [Can I Use -- HTTP/3](https://caniuse.com/http3)
