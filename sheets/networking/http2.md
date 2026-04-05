# HTTP/2 (Hypertext Transfer Protocol Version 2)

Binary-framed, multiplexed application-layer protocol that replaces HTTP/1.1's text-based serial processing with concurrent streams over a single TCP connection, using HPACK header compression, flow control, and optional server push.

## Protocol Identifiers

```
h2    — HTTP/2 over TLS (negotiated via ALPN extension)
h2c   — HTTP/2 cleartext (no TLS, direct or via Upgrade)

# ALPN negotiation during TLS handshake
ClientHello: ALPN=[h2, http/1.1]
ServerHello: ALPN=h2          # Server selects h2

# Connection preface (client must send first)
PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n   # 24-byte magic octets
# Followed immediately by a SETTINGS frame
```

## Upgrade from HTTP/1.1 (h2c only)

```
# Client sends HTTP/1.1 request with Upgrade header
GET / HTTP/1.1
Host: example.com
Connection: Upgrade, HTTP2-Settings
Upgrade: h2c
HTTP2-Settings: <base64url-encoded SETTINGS payload>

# Server responds with 101 if it supports h2c
HTTP/1.1 101 Switching Protocols
Connection: Upgrade
Upgrade: h2c

# Then both sides switch to HTTP/2 framing
# NOTE: h2c upgrade is single-request — concurrent streams begin after
```

## Frame Types

```
Type            ID  Flags              Description
────────────────────────────────────────────────────────────────────
DATA            0   END_STREAM, PAD    Carries request/response body
HEADERS         1   END_STREAM,        Opens stream, carries compressed headers
                    END_HEADERS, PAD,
                    PRIORITY
PRIORITY        2   (none)             Stream weight + dependency (deprecated RFC 9113)
RST_STREAM      3   (none)             Immediately terminate one stream
SETTINGS        4   ACK               Connection-level parameters
PUSH_PROMISE    6   END_HEADERS, PAD   Server push announcement
PING            7   ACK               Keepalive / RTT measurement
GOAWAY          8   (none)             Graceful connection shutdown
WINDOW_UPDATE   9   (none)             Flow control credit (per-stream or connection)
CONTINUATION    A   END_HEADERS        Extends a HEADERS/PUSH_PROMISE block
```

## Frame Header (9 bytes)

```
+-----------------------------------------------+
|                 Length (24)                     |
+---------------+---------------+---------------+
|   Type (8)    |   Flags (8)   |
+-+-------------+---------------+-------------------------------+
|R|                 Stream Identifier (31)                       |
+-+-------------------------------------------------------------+
|                   Frame Payload (0...)                          |
+---------------------------------------------------------------+

Length:    payload size in bytes (max 16,384 default, up to 16,777,215)
Type:     frame type (0x0-0x9, 0xa)
Flags:    type-specific (END_STREAM=0x1, END_HEADERS=0x4, PADDED=0x8, PRIORITY=0x20)
R:        reserved bit, must be 0
Stream:   31-bit stream ID (0 = connection-level, odd = client-initiated, even = server push)
```

## SETTINGS Parameters

```
Parameter                   ID  Default    Description
────────────────────────────────────────────────────────────────
HEADER_TABLE_SIZE           1   4,096      HPACK dynamic table size (bytes)
ENABLE_PUSH                 2   1          Server push allowed (0=disabled, 1=enabled)
MAX_CONCURRENT_STREAMS      3   unlimited  Max simultaneous streams
INITIAL_WINDOW_SIZE         4   65,535     Initial flow-control window (bytes)
MAX_FRAME_SIZE              5   16,384     Max frame payload (16,384 - 16,777,215)
MAX_HEADER_LIST_SIZE        6   unlimited  Max header block size (advisory)
```

## HPACK Header Compression

```
# Static table: 61 pre-defined header name-value pairs (indexed 1-61)
#   Index 2 = :method GET
#   Index 4 = :path /
#   Index 7 = :scheme https

# Dynamic table: connection-specific entries added during communication
# Entries evicted FIFO when table exceeds HEADER_TABLE_SIZE

# Encoding types:
#   Indexed (1 byte)         — full match in static/dynamic table
#   Literal + indexing       — add new entry to dynamic table
#   Literal without indexing — do not add to dynamic table
#   Literal never indexed    — sensitive value, never compressed

# Huffman encoding: optional bit-level encoding for string values
# Typical compression ratio: 85-95% for repeated headers
```

## Stream States

```
                         +--------+
                    +--->|  idle  |<---+
                    |    +--------+    |
                    |     |  send H    | recv H
                    |     |            |
                    |     v            v
                    |  +--------+  +--------+
                    |  |reserved|  |reserved|
                    |  | (local)|  |(remote)|
                    |  +--------+  +--------+
                    |     |  send H    | recv H
                    |     |            |
                    |     v            v
                    |    +--------+
                    |    |  open  |
                    |    +--------+
                    |   /          \
              send ES/            \recv ES
                  /                  \
           +--------+          +--------+
           |  half  |          |  half  |
           | closed |          | closed |
           |(local) |          |(remote)|
           +--------+          +--------+
                  \                /
             recv ES\            /send ES
                     \          /
                    +--------+
                    | closed |
                    +--------+

H  = HEADERS frame
ES = END_STREAM flag
R  = RST_STREAM frame (can transition to closed from any state)
```

## Flow Control

```
# Two levels:
#   Connection-level: total bytes across ALL streams (stream 0)
#   Per-stream: bytes for each individual stream

# Initial window: INITIAL_WINDOW_SIZE (default 65,535 bytes)
# Sender decrements window by bytes sent
# Receiver sends WINDOW_UPDATE to grant more credit
# Window can go negative if SETTINGS reduces it mid-connection

# Only DATA frames are flow-controlled (HEADERS, SETTINGS etc. are not)

# WINDOW_UPDATE payload: 4 bytes
# +---+-------------------------------+
# |R|  Window Size Increment (31)     |
# +---+-------------------------------+
# Increment: 1 to 2,147,483,647 (2^31 - 1)
```

## Multiplexing & Streams

```
# All requests/responses are interleaved frames on one TCP connection
# Each stream has a unique 31-bit ID
#   Client-initiated: odd (1, 3, 5, ...)
#   Server push:      even (2, 4, 6, ...)
#   Connection:       0 (SETTINGS, PING, GOAWAY, WINDOW_UPDATE)

# Typical browser: 100-256 concurrent streams per connection
# vs HTTP/1.1: 6-8 parallel TCP connections per domain

# Stream concurrency controlled by MAX_CONCURRENT_STREAMS setting
```

## Stream Prioritization

```
# RFC 7540 model (deprecated in RFC 9113):
#   Weight: 1-256 (relative priority within siblings)
#   Dependency: parent stream ID (forms a tree)
#   Exclusive flag: reprioritize all siblings under this stream

# RFC 9218 — Extensible Priorities (replacement):
#   Uses Priority header field and PRIORITY_UPDATE frame
#   Parameters: urgency (0-7, default 3), incremental (boolean)
#
#   Priority: u=0          # Highest urgency
#   Priority: u=3, i       # Default urgency, incremental (e.g., images)
#   Priority: u=7          # Lowest urgency (background prefetch)
```

## Server Push

```
# Server sends PUSH_PROMISE on an existing client stream
# Promises a response the client has not yet requested
# Client can reject with RST_STREAM on the promised stream

# Flow:
# 1. Client sends GET / on stream 1
# 2. Server sends PUSH_PROMISE (stream 1) promising stream 2 for /style.css
# 3. Server sends HEADERS + DATA on stream 2 (the pushed response)
# 4. Server sends HEADERS + DATA on stream 1 (the original response)

# Client can disable push: SETTINGS ENABLE_PUSH=0

# NOTE: Server Push deprecated in practice
#   Chrome removed support (2022), Firefox never fully supported
#   Use 103 Early Hints instead
```

## Head-of-Line Blocking

```
# HTTP/1.1: Application-layer HOL blocking
#   Pipelined responses must be returned in order
#   One slow response blocks all queued responses

# HTTP/2: TCP-layer HOL blocking
#   Multiplexed streams share one TCP connection
#   One lost TCP segment stalls ALL streams until retransmitted
#   More streams = more likely to be affected by any single packet loss

# HTTP/3 (QUIC): No HOL blocking
#   Each stream has independent loss recovery
#   Packet loss only stalls the affected stream
```

## Comparison: HTTP/1.1 vs HTTP/2 vs HTTP/3

```
Feature              HTTP/1.1          HTTP/2            HTTP/3
──────────────────────────────────────────────────────────────────
Transport            TCP               TCP               QUIC (UDP)
Framing              Text              Binary            Binary
Multiplexing         No (pipelining)   Yes               Yes
Header compression   None              HPACK             QPACK
Server push          No                Yes (deprecated)  Yes (deprecated)
HOL blocking         Application       TCP               None
Connections/domain   6-8               1                 1
TLS requirement      Optional          Practical (ALPN)  Mandatory (1.3)
0-RTT resumption     No                No                Yes
Connection migration No                No                Yes
```

## Common Error Codes (GOAWAY / RST_STREAM)

```
Code  Name                  Description
──────────────────────────────────────────────────────────
0x0   NO_ERROR              Graceful shutdown / normal close
0x1   PROTOCOL_ERROR        Generic protocol violation
0x2   INTERNAL_ERROR        Implementation fault
0x3   FLOW_CONTROL_ERROR    Flow control limits violated
0x4   SETTINGS_TIMEOUT      SETTINGS not acknowledged in time
0x5   STREAM_CLOSED         Frame received on closed stream
0x6   FRAME_SIZE_ERROR      Invalid frame size
0x7   REFUSED_STREAM        Stream refused before processing
0x8   CANCEL                Stream no longer needed
0x9   COMPRESSION_ERROR     HPACK decompression failure
0xa   CONNECT_ERROR         TCP connection for CONNECT failed
0xb   ENHANCE_YOUR_CALM     Peer generating excessive load
0xc   INADEQUATE_SECURITY   TLS requirements not met
0xd   HTTP_1_1_REQUIRED     Use HTTP/1.1 for this request
```

## Practical Examples

```bash
# Verify HTTP/2 support
curl -sI --http2 https://example.com -o /dev/null -w '%{http_version}\n'
# Output: 2

# Verbose HTTP/2 negotiation (shows ALPN, frames)
curl -v --http2 https://example.com 2>&1 | head -30

# Force HTTP/2 cleartext (h2c) with prior knowledge
curl -v --http2-prior-knowledge http://localhost:8080/

# Inspect ALPN negotiation
openssl s_client -connect example.com:443 -alpn h2 2>&1 | grep 'ALPN'
# ALPN protocol: h2

# nghttp2 — dedicated HTTP/2 client
nghttp -v https://example.com/
nghttp -v -H ':method: POST' -d data.json https://api.example.com/

# List server settings and stream info
nghttp -v --stat https://example.com/ 2>&1 | grep -E '(SETTINGS|stream)'

# Test server push
nghttp -v https://example.com/ 2>&1 | grep PUSH_PROMISE

# h2load — HTTP/2 benchmarking tool (from nghttp2)
h2load -n 10000 -c 10 -m 100 https://example.com/
#   -n: total requests  -c: connections  -m: max concurrent streams

# Check HTTP/2 with nmap
nmap --script http2-support -p 443 example.com

# Wireshark filter for HTTP/2
# Display filter: http2
# Decode as: SSL stream → HTTP2
```

```bash
# Go net/http — automatic HTTP/2 when using TLS
# No code changes needed; http.ListenAndServeTLS enables h2 by default

# Disable HTTP/2 in Go client
transport := &http.Transport{
    TLSNextProto: make(map[string]func(string, *tls.Conn) http.RoundTripper),
}

# Explicitly use h2c in Go
import "golang.org/x/net/http2"
import "golang.org/x/net/http2/h2c"

h2s := &http2.Server{}
handler := h2c.NewHandler(myHandler, h2s)
http.ListenAndServe(":8080", handler)
```

## Tips

- HTTP/2 multiplexing eliminates the need for domain sharding, CSS sprites, and resource inlining. These HTTP/1.1 performance hacks are actively harmful under HTTP/2 because they prevent fine-grained caching and prioritization.
- A single lost TCP packet stalls all multiplexed HTTP/2 streams until retransmitted. On lossy networks (mobile, satellite), HTTP/2 can perform worse than HTTP/1.1's multiple connections. This is the primary motivation for HTTP/3/QUIC.
- HPACK's dynamic table is connection-scoped, not request-scoped. Long-lived connections build up a rich table and achieve 85-95% header compression. Short-lived connections barely benefit.
- Always set `MAX_CONCURRENT_STREAMS` on the server. Without it, a client can open unbounded streams, creating a DoS vector. Typical values: 100-256 for web servers, 1000+ for gRPC.
- Server Push is effectively dead. Chrome removed it in 2022. Use `103 Early Hints` with `Link: </style.css>; rel=preload` to achieve similar preloading without the complexity and cache-invalidation pitfalls.
- The `ENHANCE_YOUR_CALM` error code (0xb) is the HTTP/2 equivalent of "you're sending too much." Servers use it to signal rate-limiting at the protocol level before resorting to connection termination.
- `GOAWAY` with `last-stream-id` tells the client which streams were processed. Streams with higher IDs were not processed and must be retried on a new connection. This enables graceful rolling restarts.
- When debugging HTTP/2, `nghttp -v` is more informative than `curl -v` because it shows individual frame types, stream IDs, and flow control window updates.
- For cleartext HTTP/2 (h2c), use `--http2-prior-knowledge` with curl. The Upgrade mechanism adds a round trip and only works for the first request; prior knowledge skips this entirely.
- RFC 9113 (2022) obsoletes RFC 7540 and deprecates the stream priority scheme. Use RFC 9218 Extensible Priorities (`Priority` header) instead of the PRIORITY frame.

## See Also

- http, quic, tls

## See Also (detail)

- http2

## References

- [RFC 9113 -- HTTP/2](https://www.rfc-editor.org/rfc/rfc9113)
- [RFC 7541 -- HPACK: Header Compression for HTTP/2](https://www.rfc-editor.org/rfc/rfc7541)
- [RFC 9218 -- Extensible Prioritization Scheme for HTTP](https://www.rfc-editor.org/rfc/rfc9218)
- [RFC 8740 -- Using TLS 1.3 with HTTP/2](https://www.rfc-editor.org/rfc/rfc8740)
- [nghttp2 -- HTTP/2 C library and tools](https://nghttp2.org/)
- [curl -- HTTP/2 documentation](https://curl.se/docs/http2.html)
- [MDN -- HTTP/2](https://developer.mozilla.org/en-US/docs/Glossary/HTTP_2)
