# WebRTC (Web Real-Time Communication)

Browser-native framework for peer-to-peer audio, video, and data communication without plugins. WebRTC combines ICE (RFC 8445) for connectivity, DTLS-SRTP for encryption, and SDP for media negotiation, enabling real-time communication directly between endpoints.

---

## ICE Framework

### Candidate Gathering and Priority (RFC 8445)

```bash
# ICE finds the best path by gathering and testing candidates:
# 1. Host — local interface addresses
# 2. Server-reflexive (srflx) — public IP via STUN
# 3. Relay — TURN-allocated address (guaranteed fallback)

# SDP candidate examples:
# a=candidate:1 1 udp 2130706431 192.168.1.100 54321 typ host
# a=candidate:2 1 udp 1694498815 203.0.113.50 55555 typ srflx
# a=candidate:3 1 udp 16777215 198.51.100.10 12345 typ relay

# Priority = (2^24 * type_pref) + (2^8 * local_pref) + (256 - component_id)
# Type preferences: host=126, prflx=110, srflx=100, relay=0
# Pair priority = 2^32*MIN(G,D) + 2*MAX(G,D) + (G>D ? 1 : 0)
```

## STUN and TURN

### STUN Server Discovery (RFC 8489)

```bash
# STUN reveals public IP/port mapping (port 3478 UDP/TCP, 5349 TLS)
stunclient stun.l.google.com:19302
stunclient stun.cloudflare.com:3478

# NAT types: Full Cone, Restricted, Port Restricted, Symmetric
# Symmetric NAT = STUN fails, must use TURN
python3 -c "
import stun
nat_type, ext_ip, ext_port = stun.get_ip_info()
print(f'NAT: {nat_type}, External: {ext_ip}:{ext_port}')
"
```

### coturn TURN Server Setup (RFC 8656)

```bash
# /etc/turnserver.conf
listening-port=3478
tls-listening-port=5349
external-ip=203.0.113.10/192.168.1.10
realm=turn.example.com

# REST API credentials (time-limited, HMAC-SHA1)
use-auth-secret
static-auth-secret=my-super-secret-key-change-this

# TLS
cert=/etc/ssl/certs/turn.pem
pkey=/etc/ssl/private/turn.key

# Security
no-multicast-peers
denied-peer-ip=10.0.0.0-10.255.255.255
denied-peer-ip=172.16.0.0-172.31.255.255

# Generate ephemeral credentials
SECRET="my-super-secret-key-change-this"
USERNAME="$(date +%s):webrtcuser"
PASSWORD=$(echo -n "$USERNAME" | openssl dgst -sha1 -hmac "$SECRET" -binary | base64)

# Test TURN allocation
turnutils_uclient -t -u webrtc -w secretpassword turn.example.com
```

## SDP Offer/Answer

### WebRTC SDP Anatomy

```bash
# v=0
# o=- 4625943528175357025 2 IN IP4 127.0.0.1
# s=-
# a=group:BUNDLE 0 1 2               # multiplex on one port
# a=ice-ufrag:EsAw                    # ICE credentials
# a=ice-pwd:P2uYro0UCOQ4zxjKXaWCBui1
# a=fingerprint:sha-256 D2:FA:...     # DTLS cert fingerprint
# a=setup:actpass                      # DTLS role
#
# m=audio 9 UDP/TLS/RTP/SAVPF 111
# a=rtpmap:111 opus/48000/2
# a=fmtp:111 minptime=10;useinbandfec=1
# a=rtcp-fb:111 transport-cc
#
# m=video 9 UDP/TLS/RTP/SAVPF 96 97
# a=rtpmap:96 VP8/90000
# a=rtpmap:97 VP9/90000
# a=rtcp-fb:96 goog-remb             # receiver BW estimation
# a=rtcp-fb:96 nack pli              # picture loss indication
#
# m=application 9 UDP/DTLS/SCTP webrtc-datachannel
# a=sctp-port:5000
```

### Codec Preferences

```bash
# Audio: Opus (6-510kbps, default), G.722 (64kbps), PCMU/PCMA (64kbps)
# Video: VP8 (mandatory), VP9, H.264 (hw accel), AV1 (best compression)
```

## DTLS-SRTP Encryption

### Media Security

```bash
# WebRTC encrypts ALL media — no opt-out
# 1. ICE establishes connectivity
# 2. DTLS handshake over same UDP port (exchanges certificates)
# 3. DTLS exports keying material for SRTP
# 4. SRTP encrypts RTP, SRTCP encrypts RTCP

# Port demux by first byte: 0-1=STUN, 20-63=DTLS, 128-191=RTP/RTCP
# DTLS fingerprint in SDP binds crypto to signaling channel
```

## Bandwidth Estimation

### REMB and Transport-CC

```bash
# REMB — receiver-side estimation (deprecated but widespread)
# a=rtcp-fb:96 goog-remb
# Uses packet loss + jitter to estimate capacity

# TWcc — sender-side estimation (current standard)
# a=rtcp-fb:96 transport-cc
# 1. Sender adds transport-wide sequence numbers to ALL RTP
# 2. Receiver sends periodic feedback (arrival times)
# 3. Sender runs GCC (Google Congestion Control) algorithm
# 4. Bitrate adjusts based on one-way delay gradient
```

## Simulcast and SVC

### Multi-Resolution Streaming

```bash
# Simulcast — sender encodes at multiple resolutions
# SFU forwards appropriate layer per receiver bandwidth
# a=simulcast:send h;m;l
# a=rid:h send    # 1280x720 @ 2.5 Mbps
# a=rid:m send    #  640x360 @ 500 kbps
# a=rid:l send    #  320x180 @ 150 kbps

# SVC — single encoded stream with temporal/spatial layers
# VP9/AV1 SVC: L0=7.5fps, L1=15fps, L2=30fps
# Pro: single encode, less CPU. Con: codec must support SVC
```

## Media Server Architectures

### MCU vs SFU

```bash
# MCU (Multipoint Conferencing Unit)
# - Decodes + composites all streams into one per participant
# - High server CPU, low client bandwidth, higher latency
# - Use: low-bandwidth clients, recording, legacy interop

# SFU (Selective Forwarding Unit)
# - Forwards packets without decoding (routing only)
# - Low server CPU, higher client bandwidth, lower latency
# - Use: modern WebRTC, simulcast, group calls
# - Examples: Janus, mediasoup, LiveKit, Pion
```

### Janus Gateway

```bash
# Install and build Janus
sudo apt install libmicrohttpd-dev libjansson-dev libssl-dev \
    libsrtp2-dev libglib2.0-dev libopus-dev pkg-config
git clone https://github.com/meetecho/janus-gateway.git
cd janus-gateway && ./autogen.sh && ./configure --prefix=/opt/janus
make && sudo make install
/opt/janus/bin/janus --config /opt/janus/etc/janus/janus.jcfg
# HTTP API: http://localhost:8088/janus  WS: ws://localhost:8188
```

### mediasoup (Node.js SFU)

```bash
# Install mediasoup
npm install mediasoup

# Architecture:
# Worker  — C++ media process (one per CPU core)
# Router  — manages transports, producers, consumers
# Transport — WebRTC, Plain, or Pipe transport
# Producer — sends media (audio/video track)
# Consumer — receives media from a producer

# Create worker and router:
# const worker = await mediasoup.createWorker({ rtcMinPort: 10000, rtcMaxPort: 10100 });
# const router = await worker.createRouter({ mediaCodecs });
```

### Pion WebRTC (Go)

```bash
# Go-native WebRTC stack
go get github.com/pion/webrtc/v4
# Components: pion/webrtc, pion/turn, pion/ice, pion/dtls, pion/srtp
# pion/interceptor — RTP/RTCP middleware (NACK, TWcc)
# Build SFU: PeerConnection per participant, OnTrack receives remote,
# AddTrack forwards to others, interceptors handle congestion control
```

## Signaling and Data Channels

### WebSocket Signaling

```bash
# WebRTC does NOT define signaling — you choose the transport
# Typical flow over WebSocket:
# 1. Caller: createOffer() -> setLocalDescription -> send offer
# 2. Callee: setRemoteDescription -> createAnswer -> setLocalDescription -> send answer
# 3. Both: trickle ICE candidates via signaling in parallel
# Message types: { "type": "offer/answer/candidate", ... }
```

### SCTP Data Channels

```bash
# Data channels: SCTP over DTLS (reliable/unreliable, ordered/unordered)
# Config options:
#   ordered: true/false (in-order delivery)
#   maxRetransmits: N (unreliable with retry limit)
#   maxPacketLifeTime: ms (unreliable with time limit)
#   protocol: string (sub-protocol identifier)
#
# Use cases:
#   File transfer — reliable, ordered
#   Game state   — unreliable, unordered (latest state wins)
#   Chat/text    — reliable, ordered
#   Telemetry    — unreliable (loss acceptable)
# Max message: ~256 KB, throughput: 30-100 Mbps typical
```

## Debugging

### Browser and Network Tools

```bash
# Chrome: chrome://webrtc-internals (PeerConnections, SDP, stats, BWE graphs)
# Firefox: about:webrtc

# getStats() key metrics:
# packetsLost/packetsReceived, jitter, roundTripTime,
# framesPerSecond, availableOutgoingBitrate

# tcpdump for WebRTC
tcpdump -i eth0 -n udp port 3478                    # STUN/TURN
tcpdump -i eth0 -n udp portrange 10000-10100         # media

# Wireshark: stun || dtls || rtp || rtcp || sctp
tshark -i eth0 -f "udp port 3478" -Y stun -T fields \
    -e stun.type -e stun.att.mapped-address
```

---

## Tips

- Always provide a TURN server — 10-15% of users are behind symmetric NATs where STUN alone fails.
- Use `iceTransportPolicy: "relay"` during debugging to force TURN and isolate media-path issues.
- Enable Opus DTX (Discontinuous Transmission) to reduce bandwidth during silence to near zero.
- Set `maxBitrate` via `RTCRtpSender.setParameters()` to prevent video from consuming all bandwidth.
- Use simulcast with an SFU for group calls; single high-res to everyone fails past 3-4 users.
- Monitor `packetsLost` and `jitter` from getStats() — loss above 2% degrades voice quality.
- Trickle ICE candidates instead of waiting for gathering to complete; saves seconds to first frame.
- Implement TURN credential rotation (REST API with HMAC) rather than static passwords.
- Test with network link conditioner tools (tc netem, macOS Network Link Conditioner).
- Use Unified Plan SDP (not Plan B) — Plan B is deprecated and removed from modern browsers.

---

## See Also

- sip, websocket, tls, stun

## References

- [RFC 8825 — Overview: Real-Time Protocols for Browser-Based Applications](https://www.rfc-editor.org/rfc/rfc8825)
- [RFC 8445 — Interactive Connectivity Establishment (ICE)](https://www.rfc-editor.org/rfc/rfc8445)
- [RFC 8489 — Session Traversal Utilities for NAT (STUN)](https://www.rfc-editor.org/rfc/rfc8489)
- [RFC 8656 — Traversal Using Relays around NAT (TURN)](https://www.rfc-editor.org/rfc/rfc8656)
- [RFC 8834 — Media Transport and Use of RTP in WebRTC](https://www.rfc-editor.org/rfc/rfc8834)
- [RFC 8831 — WebRTC Data Channels](https://www.rfc-editor.org/rfc/rfc8831)
- [WebRTC API — MDN Web Docs](https://developer.mozilla.org/en-US/docs/Web/API/WebRTC_API)
- [Janus WebRTC Gateway](https://janus.conf.meetecho.com/)
- [mediasoup — WebRTC SFU](https://mediasoup.org/)
- [Pion WebRTC — Pure Go Implementation](https://github.com/pion/webrtc)
- [coturn TURN Server](https://github.com/coturn/coturn)
