# QUIC (Quick UDP Internet Connections)

Encrypted, multiplexed, UDP-based transport protocol with built-in TLS 1.3, stream-level flow control, and connection migration — the foundation of HTTP/3.

## Overview

```
# QUIC at a glance
Transport:          UDP (typically port 443)
Encryption:         TLS 1.3 integrated (always on, no plaintext mode)
Multiplexing:       Independent streams within one connection
Handshake:          1-RTT (first connection), 0-RTT (resumption)
Connection ID:      Survives IP/port changes (connection migration)
Head-of-line:       No HOL blocking between streams (unlike HTTP/2 over TCP)
Standardized:       RFC 9000 (QUIC), RFC 9001 (QUIC-TLS), RFC 9114 (HTTP/3)
```

## Comparison with TCP+TLS

```
Feature                     TCP + TLS 1.3        QUIC
──────────────────────────────────────────────────────────────
Handshake RTTs (new)        2 RTT (1 TCP + 1 TLS) 1 RTT
Handshake RTTs (resume)     1-2 RTT               0 RTT
Encryption                  Optional (TLS layer)   Mandatory (built-in)
Multiplexing                App-layer (HTTP/2)     Native streams
HOL blocking                Yes (TCP byte stream)  No (per-stream)
Connection migration        No (4-tuple bound)     Yes (connection IDs)
Middlebox ossification      Extensive              Resistant (encrypted)
Loss recovery               Per-connection         Per-stream
Congestion control          Per-connection         Per-connection (pluggable)
```

## Connection Establishment

### 1-RTT Handshake (First Connection)

```
Client                                    Server
  |                                         |
  |  ---- Initial (CRYPTO + ClientHello) -> |  Contains TLS ClientHello
  |                                         |  in CRYPTO frame
  |  <- Handshake (CRYPTO + ServerHello) -- |  Server sends certs + finish
  |     + Initial ACK                       |
  |                                         |
  |  ---- Handshake Complete (FINISHED) --> |  Client confirms
  |       + 1-RTT data can start            |  Connection ESTABLISHED
  |                                         |
```

### 0-RTT Resumption

```
Client                                    Server
  |                                         |
  |  ---- Initial + 0-RTT data ----------> |  Client sends early data using
  |       (ClientHello + cached PSK)        |  previously cached session key
  |                                         |
  |  <- Handshake + 1-RTT data ----------- |  Server accepts 0-RTT
  |                                         |
  # 0-RTT data is NOT forward-secret and is replayable
  # Only safe for idempotent operations (GET requests, etc.)
```

## Streams

```
# Stream types (2-bit type field in stream ID)
0x0  Client-initiated, bidirectional
0x1  Server-initiated, bidirectional
0x2  Client-initiated, unidirectional
0x3  Server-initiated, unidirectional

# Stream IDs are 62-bit, sequentially assigned
# Even-numbered: client-initiated, Odd-numbered: server-initiated

# Key properties
# - Each stream has independent flow control (no cross-stream HOL blocking)
# - Streams can be created without handshake (just send on new stream ID)
# - Streams can be reset independently (RESET_STREAM frame)
# - Receiver can stop reading a stream (STOP_SENDING frame)
# - Both sides can limit max concurrent streams (MAX_STREAMS frame)
```

### Stream vs Connection Flow Control

```
# Two-level flow control
# 1. Per-stream: MAX_STREAM_DATA limits bytes on each stream
# 2. Per-connection: MAX_DATA limits total bytes across all streams

# This prevents one stream from starving others
# Receiver sends flow control updates as it consumes data
```

## Connection IDs

```
# Each connection has one or more Connection IDs (CIDs)
# CIDs are chosen by each endpoint independently
# Packets carry the destination CID (not source)

# Why this matters:
# - TCP connection = (src IP, src port, dst IP, dst port)
# - QUIC connection = Connection ID
# - When client moves from WiFi to cellular, IP changes but CID stays the same
# - Server can continue the connection seamlessly = connection migration

# NAT rebinding:
# Even without IP change, NAT may reassign the source port
# QUIC handles this transparently via CID matching

# CID rotation:
# Endpoints provide multiple CIDs via NEW_CONNECTION_ID frames
# Migrate to new CID to prevent linkability across network changes
```

## Packet Format

### Long Header (Initial, Handshake, 0-RTT)

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|1|1| Type (2) | Version (32)                                  |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| DCID Len (8) | Destination Connection ID (0-160)             |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| SCID Len (8) | Source Connection ID (0-160)                  |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                   Type-Specific Payload ...                   |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

### Short Header (1-RTT, after handshake)

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|0|1|S|R|R|K|PP | Destination Connection ID (variable)         |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                   Packet Number (8-32) ...                    |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                   Protected Payload ...                       |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
# Short header used for all post-handshake data
# Bit 0 = 0 distinguishes from long header
# K = Key Phase bit (for key rotation)
# PP = Packet Number length encoding
```

## Loss Detection and Congestion Control

```
# QUIC separates loss detection from congestion control

# Loss detection:
# - Packet numbers never repeat (unlike TCP sequence numbers)
# - ACK frames carry precise timestamps
# - No ambiguity on retransmissions (new packet number each time)
# - Probe Timeout (PTO) replaces TCP's RTO with faster tail recovery

# Congestion control:
# - Pluggable algorithm (same connection can switch)
# - Default: NewReno-like (RFC 9002)
# - Common implementations also support CUBIC and BBR
# - Operates per-connection (not per-stream)
# - ECN support available

# ACK mechanism:
# - ACK frames are not ack-eliciting (no ACK-of-ACK loops)
# - ACK frames include ECN counts
# - Largest Acknowledged + ACK Ranges (similar to TCP SACK)
```

## HTTP/3

```
# HTTP/3 = HTTP semantics over QUIC (RFC 9114)
# Replaces HTTP/2's TCP+TLS transport with QUIC

# Key differences from HTTP/2:
# - No TCP HOL blocking — lost packet only affects its stream
# - Fewer handshake RTTs (1-RTT vs 2+ for TCP+TLS)
# - Connection migration keeps requests alive across network changes

# Header compression:
# - QPACK (RFC 9204) replaces HPACK
# - HPACK requires ordered delivery (TCP), QPACK works with QUIC's unordered streams
# - Uses dedicated unidirectional streams for encoder/decoder

# HTTP/3 uses three stream types:
# - Control stream (unidirectional, one per direction)
# - QPACK encoder/decoder streams (unidirectional)
# - Request streams (bidirectional, one per HTTP request)
```

## Tools and Libraries

### curl

```bash
# Test HTTP/3 with curl (must be built with HTTP/3 support)
curl --http3 https://example.com                      # prefer HTTP/3
curl --http3-only https://example.com                  # HTTP/3 or fail
curl -v --http3 https://example.com                    # verbose output shows QUIC handshake
curl --http3 -w '%{http_version}\n' -o /dev/null -s https://example.com  # show protocol version

# Check if curl supports HTTP/3
curl --version | grep HTTP3
```

### Server Software

```bash
# nginx (quic module, mainline 1.25+)
# In nginx.conf:
# listen 443 quic reuseport;
# listen 443 ssl;
# http3 on;
# ssl_early_data on;         # 0-RTT
# add_header Alt-Svc 'h3=":443"; ma=86400';

# Caddy (built-in HTTP/3 support, enabled by default)
# No special config needed — Caddy serves HTTP/3 automatically

# HAProxy (experimental QUIC support in 2.6+)
# bind quic4@:443 ssl crt /path/to/cert.pem alpn h3
```

### Libraries

```
quiche      — Cloudflare's QUIC/HTTP3 implementation (Rust, C API)
msquic      — Microsoft's cross-platform QUIC (C, used in Windows/.NET)
ngtcp2      — C library implementing QUIC (used by curl)
quinn       — Rust async QUIC implementation
quic-go     — Go implementation of QUIC
aioquic     — Python asyncio QUIC implementation
```

## Debugging

### Wireshark

```bash
# Wireshark decodes QUIC natively (3.x+)
# For decrypting QUIC traffic, export TLS keys:
export SSLKEYLOGFILE=/tmp/quic-keys.log
curl --http3 https://example.com

# In Wireshark: Edit > Preferences > Protocols > TLS > (Pre)-Master-Secret log filename
# Point to /tmp/quic-keys.log
# Filter: quic
```

### qlog

```
# qlog is the standard QUIC event logging format (draft-ietf-quic-qlog)
# Most implementations support qlog output:
# - quiche: QLOGDIR=/tmp/qlogs
# - ngtcp2: --qlog-dir /tmp/qlogs
# - msquic: set QUIC_PARAM_GLOBAL_SETTINGS

# Visualize with qvis: https://qvis.quictools.info/
# Upload qlog files for sequence diagrams, congestion graphs, etc.
```

### Common Debug Commands

```bash
# Check if a server supports HTTP/3 (look for Alt-Svc header)
curl -sI https://example.com | grep -i alt-svc

# Test QUIC connectivity with ngtcp2
ngtcp2-client example.com 443

# Check UDP 443 reachability
nc -zuv example.com 443            # basic UDP port check

# Monitor QUIC connections on a server
ss -unp | grep 443                 # UDP sockets on port 443
```

## Firewall Considerations

```bash
# QUIC uses UDP port 443 (typically)
# Many firewalls/networks block or deprioritize UDP

# Allow QUIC traffic
iptables -A INPUT -p udp --dport 443 -j ACCEPT
iptables -A OUTPUT -p udp --sport 443 -j ACCEPT

# Challenges for middleboxes:
# - QUIC payload is encrypted — no deep packet inspection possible
# - Connection IDs make simple 4-tuple tracking insufficient
# - 0-RTT data arrives before handshake completes
# - Connection migration changes source IP/port mid-connection

# Conntrack considerations:
# - UDP conntrack timeout must be long enough for QUIC idle periods
# - QUIC keep-alive (PING frames) prevent conntrack expiry
sysctl -w net.netfilter.nf_conntrack_udp_timeout_stream=180

# Some networks block UDP 443 entirely
# Browsers fall back to HTTP/2 over TCP when QUIC fails (Happy Eyeballs v2)
```

## Deployment

### CDN Support

```
# Major CDNs with HTTP/3 support:
# Cloudflare  — Enabled by default for all plans
# Akamai      — Available on HTTP/2+QUIC product
# AWS CloudFront — Supported (enable in distribution settings)
# Google Cloud CDN — Supported (uses Google's QUIC implementation)
# Fastly      — Supported via H3

# Alt-Svc header advertises HTTP/3 availability:
# Alt-Svc: h3=":443"; ma=86400
# Browser tries HTTP/3 on next request after seeing this header
```

### Server Configuration Example (nginx)

```
server {
    # HTTP/3 (QUIC)
    listen 443 quic reuseport;

    # HTTP/2 + HTTP/1.1 fallback
    listen 443 ssl;
    http2 on;

    ssl_certificate     /etc/ssl/certs/example.com.pem;
    ssl_certificate_key /etc/ssl/private/example.com.key;

    # Enable 0-RTT (early data)
    ssl_early_data on;

    # Advertise HTTP/3 to clients
    add_header Alt-Svc 'h3=":443"; ma=86400' always;

    # Recommended: increase UDP buffer sizes
    # sysctl net.core.rmem_max=2500000
    # sysctl net.core.wmem_max=2500000
}
```

## Tips

- QUIC 0-RTT data is vulnerable to replay attacks. Only use it for idempotent requests (GET, HEAD). Servers should implement replay protection or only accept idempotent 0-RTT requests.
- If your site sees high mobile traffic, HTTP/3 provides the biggest wins: connection migration handles WiFi-to-cellular transitions, and 0-RTT reduces perceived latency for returning users.
- Many corporate networks and some ISPs block UDP 443. Always serve HTTP/2 over TCP as a fallback. Browsers handle this automatically via the Alt-Svc mechanism and Happy Eyeballs.
- QUIC packet numbers are never reused, unlike TCP sequence numbers. This eliminates retransmission ambiguity but means each retransmitted frame gets a new packet number, making packet traces harder to follow without qlog.
- When deploying QUIC servers, increase UDP buffer sizes (`rmem_max`/`wmem_max`) to at least 2.5 MB. Default Linux UDP buffers are tuned for small DNS-style exchanges, not sustained throughput.
- GSO (Generic Segmentation Offload) for UDP significantly improves QUIC server throughput. Most QUIC libraries support it. Enable with `sysctl net.core.default_qdisc=fq` and verify NIC support via `ethtool -k eth0 | grep gso`.
- QUIC encrypts almost everything, including packet numbers and most of the header. This makes it resistant to middlebox ossification (the problem where middleboxes break when TCP options change) but also makes it opaque to network monitoring tools. Use qlog for visibility.
- Connection migration only works when the server uses connection IDs for routing. If you are behind a load balancer, it must either be QUIC-aware (route by CID) or terminate QUIC at the LB.

## References

- [RFC 9000 — QUIC: A UDP-Based Multiplexed and Secure Transport](https://www.rfc-editor.org/rfc/rfc9000)
- [RFC 9001 — Using TLS to Secure QUIC](https://www.rfc-editor.org/rfc/rfc9001)
- [RFC 9002 — QUIC Loss Detection and Congestion Control](https://www.rfc-editor.org/rfc/rfc9002)
- [RFC 9114 — HTTP/3](https://www.rfc-editor.org/rfc/rfc9114)
- [RFC 9221 — An Unreliable Datagram Extension to QUIC](https://www.rfc-editor.org/rfc/rfc9221)
- [QUIC Working Group — IETF Datatracker](https://datatracker.ietf.org/wg/quic/documents/)
- [Cloudflare — HTTP/3: The Past, the Present, and the Future](https://blog.cloudflare.com/http3-the-past-present-and-future/)
- [Cloudflare quiche — QUIC and HTTP/3 Library](https://github.com/cloudflare/quiche)
- [curl — HTTP/3 Support](https://curl.se/docs/http3.html)
- [nginx — QUIC and HTTP/3 Support](https://nginx.org/en/docs/quic.html)
