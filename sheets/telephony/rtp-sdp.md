# RTP & SDP — Real-Time Media Protocols and Session Description

Wire formats, codec parameters, NAT traversal, and the goldmine of SDP attributes.

## Setup

RTP (Real-time Transport Protocol, RFC 3550) is the wire format for audio and video over UDP.
SDP (Session Description Protocol, RFC 4566 / RFC 8866) is the text-based negotiation format
exchanged in SIP/SAP/WebRTC signaling that describes what RTP streams will look like.
RTCP (RTP Control Protocol, also RFC 3550) is RTP's mandatory companion that carries
statistics, sender/receiver reports, source descriptions, and feedback messages.

The relationship is:

- **SIP** (or SAP, or WebRTC's signaling channel) = signaling that carries SDP.
- **SDP** = the menu of codecs, transports, IPs, ports, ICE candidates, and crypto keys.
- **RTP** = the actual media packets flowing once SDP negotiation completes.
- **RTCP** = sidecar control packets that flow alongside RTP.
- **DTLS-SRTP** = modern security: DTLS handshake on the media port keys SRTP for media.

```
SIP INVITE
   Content-Type: application/sdp
   Body: SDP offer  (codecs, ports, ICE, fingerprint)
SIP 200 OK
   Content-Type: application/sdp
   Body: SDP answer (intersection)
SIP ACK
   ↓
DTLS handshake on media port  →  derive SRTP keys
   ↓
SRTP / SRTCP packets flow      ←  this is the "media plane"
```

```bash
# Typical port allocation for one A/V call (no BUNDLE):
#   audio RTP   10000   audio RTCP   10001
#   video RTP   10002   video RTCP   10003
#
# With BUNDLE+rtcp-mux (modern WebRTC):
#   single port 10000 carries both audio+video, RTP+RTCP
```

```bash
# SIP and RTP are separate flows. Signaling can flow through proxies,
# but media usually goes peer-to-peer (or through a media relay
# like rtpengine, FreeSWITCH, Asterisk-RTP-proxy).
sngrep                           # see SIP signaling live
tcpdump -ni any 'udp portrange 10000-20000'  # see RTP/RTCP
```

```bash
# Mandatory triad to understand for any RTP debugging:
#   1. SDP body            -- the contract
#   2. RTP packet header   -- the envelope
#   3. RTCP SR/RR          -- the feedback
```

## RTP Packet Format

RTP has a fixed 12-byte header followed by an optional CSRC list, optional
extension header, then the payload (which is codec-specific).

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|V=2|P|X|  CC   |M|     PT      |       sequence number         |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                           timestamp                           |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|           synchronization source (SSRC) identifier            |
+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+
|            contributing source (CSRC) identifiers             |
|                             ....                              |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|             optional extension header (if X=1)                |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                           payload                             |
|                             ....                              |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

### Field-by-field

- **V (2 bits)** — version. ALWAYS 2. (Version 1 is RTPv1, never seen in the wild.)
- **P (1 bit)** — padding flag. If 1, payload ends with N padding bytes; last byte
  contains N. Used for block-cipher alignment in SRTP and for FEC.
- **X (1 bit)** — extension flag. If 1, an RTP header extension follows the CSRC list.
- **CC (4 bits)** — CSRC count, 0–15. Number of 32-bit CSRC identifiers.
- **M (1 bit)** — marker. Codec-specific:
  - Audio: set on the first packet of a talk-spurt (after silence).
  - Video: set on the last packet of a frame (or last fragment of a NAL unit for H.264).
- **PT (7 bits)** — payload type, 0–127. See payload types section.
- **Sequence Number (16 bits)** — increments by 1 each packet, used to detect loss
  and reorder. Initial value should be random per RFC 3550 §5.1.
- **Timestamp (32 bits)** — codec-clock-rate-based; for 8kHz audio, increments by
  160 per 20ms packet (8000 Hz × 0.020 s = 160 samples). Initial value random.
- **SSRC (32 bits)** — synchronization source. Identifies the *original* source
  of this stream. Random; collision-detection required.
- **CSRC list (32 × CC bits)** — when a mixer combines streams, original SSRCs
  go here so the receiver can identify contributors.

### Marker bit semantics by codec

```
Codec        M=1 means
-----        ---------
PCMU/PCMA    First packet of talk-spurt (after CN/silence)
G.722        First packet of talk-spurt
G.729        First packet of talk-spurt
Opus         First packet of talk-spurt (when DTX active)
H.264        Last packet of an access unit (frame)
VP8          End of frame
VP9          End of frame
AV1          End of OBU sequence
telephone-event (RFC 4733)  Set on FIRST packet of a DTMF event
```

### Sequence number wrap-around

```c
// 16-bit sequence numbers wrap. Compare with care.
static int seq_lt(uint16_t a, uint16_t b) {
    return (int16_t)(a - b) < 0;
}
// Newer is the one with smaller (a - b) modulo 2^16.
```

### Timestamp arithmetic

```bash
# G.711 8000 Hz, 20ms packets:
#   timestamp_increment = 8000 * 0.020 = 160
# Opus 48000 Hz, 20ms packets:
#   timestamp_increment = 48000 * 0.020 = 960
# H.264 90 kHz clock, 30 fps:
#   timestamp_increment per frame = 90000 / 30 = 3000
# Video timestamp clock is ALWAYS 90 kHz (RFC 3551).
```

### Padding bit example

```
 ...payload bytes...  PAD PAD PAD 03
 ^                    ^^^ ^^^ ^^^ ^^
 actual content       3 padding bytes; last byte = N=3
```

### Extension header (when X=1)

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|       defined by profile      |             length            |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                        header extension                       |
|                             ....                              |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

The "defined by profile" field is `0xBEDE` for one-byte form (RFC 8285) or
`0x100x` for two-byte form. See RTP Header Extensions section below.

## RTP Payload Types (RFC 3551)

Static payload types are pre-assigned. Dynamic range (96–127) is negotiated
in SDP via `a=rtpmap`.

```
PT   Codec       Rate(Hz)  Channels  Notes
--   -----       --------  --------  -----
 0   PCMU        8000      1         G.711 μ-law (North America)
 1   reserved
 2   reserved
 3   GSM         8000      1         13.2 kbps
 4   G.723       8000      1         5.3 / 6.3 kbps; royalty-bearing
 5   DVI4        8000      1         IMA ADPCM
 6   DVI4        16000     1
 7   LPC         8000      1         Linear Predictive Coding
 8   PCMA        8000      1         G.711 A-law (Europe / non-NA)
 9   G.722       8000      1         WIDEBAND but historic clock-rate
                                     misencoding: real clock is 16000
10   L16         44100     2         Linear PCM 16-bit stereo
11   L16         44100     1         Linear PCM 16-bit mono
12   QCELP       8000      1
13   CN          8000      1         Comfort Noise (RFC 3389)
14   MPA         90000     -         MPEG-1/2 Audio
15   G.728       8000      1         Low-Delay CELP, 16 kbps
16   DVI4        11025     1
17   DVI4        22050     1
18   G.729       8000      1         8 kbps; royalty-bearing (now expired)
19   reserved
20-23 unassigned
24-25 unassigned (formerly video)
26   JPEG        90000     -         Motion JPEG
27   unassigned
28   nv          90000     -         Xerox NV video
29-30 unassigned
31   H.261       90000     -         legacy video
32   MPV         90000     -         MPEG-1/2 video
33   MP2T        90000     -         MPEG-2 TS
34   H.263       90000     -         legacy video
35-71 unassigned
72-76 reserved (RTCP collision avoidance)
77-95 unassigned
96-127  DYNAMIC — bound by SDP a=rtpmap
```

### Common dynamic payload type assignments (de facto)

```
PT     Codec               Common SDP rtpmap
---    -----               -----------------
96     VP8                 a=rtpmap:96 VP8/90000
97     RTX for 96          a=rtpmap:97 rtx/90000  + a=fmtp:97 apt=96
98     VP9                 a=rtpmap:98 VP9/90000
99     RTX for 98
100    H.264 (one profile)
101    telephone-event     a=rtpmap:101 telephone-event/8000
102    H.264 baseline
103    RTX for 102
104    H.264 high profile
111    Opus                a=rtpmap:111 opus/48000/2
112    red (FEC)
113    ulpfec / flexfec
122    AV1
123    RTX for 122
125    H.264 constrained baseline
```

These are conventions, not standards — always check the actual `a=rtpmap`.

## RTCP — RTP Control Protocol

RTCP runs on the next-higher (odd) port unless `rtcp-mux` is negotiated. Carries:

- Sender Reports (SR) — sender's view of the stream (NTP timestamp, packet/byte counts).
- Receiver Reports (RR) — receiver's view (loss, jitter, last SR received).
- Source Description (SDES) — CNAME (canonical name), real name, email, etc.
- BYE — explicit teardown.
- APP — application-specific.
- Transport / payload feedback (RTPFB / PSFB) — NACK, PLI, FIR, REMB, transport-cc.

Standard rules:

- RTCP traffic SHOULD be ≤ 5% of session bandwidth.
- Compound packets — every RTCP UDP packet MUST start with SR or RR, then SDES.
- Reduced-size RTCP (RFC 5506) relaxes this for tight feedback loops.

```bash
# RTCP scheduled with randomization to avoid synchronization across endpoints
#   T = (avg_rtcp_size * num_members) / (0.05 * session_bw)
#   randomized in [0.5T, 1.5T]
```

## RTCP Packet Types (RFC 3550 + 4585)

```
PT    Name        Semantics
--    ----        ---------
200   SR          Sender Report (sender stats + receiver report blocks)
201   RR          Receiver Report (only RR blocks; for receivers that don't send)
202   SDES        Source Description (CNAME, NAME, EMAIL, PHONE, LOC, TOOL,
                                      NOTE, PRIV)
203   BYE         Goodbye; participant leaving
204   APP         Application-defined payload
205   RTPFB       Generic Transport Feedback (RFC 4585)
                    FMT 1 = generic NACK
                    FMT 3 = TMMBR (Temporary Maximum Media Bit Rate Request)
                    FMT 4 = TMMBN (notification)
                    FMT 15 = transport-cc (RFC draft, ubiquitous in WebRTC)
206   PSFB        Payload-Specific Feedback (RFC 4585)
                    FMT 1 = PLI (Picture Loss Indication)
                    FMT 2 = SLI (Slice Loss Indication)
                    FMT 3 = RPSI (Reference Picture Selection Indication)
                    FMT 4 = FIR (Full Intra Request, RFC 5104)
                    FMT 15 = AFB (Application Feedback) — REMB lives here
207   XR          Extended Reports (RFC 3611)
208   AVB         AVB RTCP (rare)
209-255  reserved
```

### SR (Sender Report) layout

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|V=2|P|    RC   |   PT=200      |             length            |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                         SSRC of sender                        |
+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+
|              NTP timestamp, most significant word             |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|             NTP timestamp, least significant word             |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                         RTP timestamp                         |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                     sender's packet count                     |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                      sender's octet count                     |
+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+
|         report block 1 (SSRC_1, fraction lost, jitter, ...)   |
|                             ....                              |
```

The NTP / RTP timestamp pair lets receivers correlate wall-clock time with
RTP timeline. Used for A/V sync.

### Receiver Report block (inside SR or RR)

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                 SSRC of source being reported                 |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| fraction lost |       cumulative number of packets lost       |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|           extended highest sequence number received           |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                      interarrival jitter                      |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                         last SR (LSR)                         |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                  delay since last SR (DLSR)                   |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

LSR + DLSR + receiver's clock at SR arrival = round-trip time.

### SDES items

```
ITEM    Name        Description
----    ----        -----------
1       CNAME       Canonical end-point identifier (mandatory in EVERY SDES)
2       NAME        User name (real name)
3       EMAIL       Email address (RFC 822)
4       PHONE       Phone number (E.164)
5       LOC         Geographic location
6       TOOL        Application name and version
7       NOTE        Transient note
8       PRIV        Private extensions
```

CNAME format: `user@host` or random-string-tied-to-the-endpoint. Used to bind
multiple SSRCs (audio + video) of the same participant.

## RTCP-Mux (RFC 5761)

Multiplex RTP and RTCP on the **same** UDP port. Negotiated via SDP attribute:

```
a=rtcp-mux
```

When agreed by both ends, RTCP packets are demultiplexed by the second byte
(payload type): RTCP PTs are 200–207 (high values), normal RTP PTs are 0–127
(low values, with the marker bit folded in). Specifically:

- RTP packets have PT 0–127 → second byte 0–127 (M=0) or 128–255 (M=1) but
  PT bits 0–127.
- RTCP packets have PT 200–207 → second byte 200–207.

Edge case: if a chosen RTP PT is 72–76, it can collide with RTCP PT 200–204
when the marker bit flips, so those PTs are reserved.

Why use it: cuts ICE candidate gathering in half, simplifies firewalls.

## Reduced-Size RTCP (RFC 5506)

Normally every UDP datagram with RTCP must start with SR or RR (a "compound"
packet). RFC 5506 lets you send a single feedback message (e.g., NACK or PLI)
in its own UDP datagram without the SR/RR wrapping.

```
a=rtcp-rsize
```

Used heavily in WebRTC video for tight loop NACK/PLI feedback.

## SRTP — Secure RTP (RFC 3711)

SRTP encrypts the RTP **payload** (not the header) and adds an authentication
tag. Default cipher is AES-CTR; default authentication is HMAC-SHA1 truncated
to 80 or 32 bits.

```
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                    RTP header (cleartext)                     |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                  encrypted RTP payload                        |
|                             ....                              |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|        ~ optional MKI ~                                       |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                  authentication tag (80 bits default)         |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

### Crypto suites

```
Suite name                        Cipher       Auth        Notes
----------                        ------       ----        -----
AES_CM_128_HMAC_SHA1_80           AES-CTR-128  HMAC-SHA1   Default; 80-bit tag
AES_CM_128_HMAC_SHA1_32           AES-CTR-128  HMAC-SHA1   32-bit tag (small)
F8_128_HMAC_SHA1_80               AES-F8-128   HMAC-SHA1   Used in 3G IMS
AES_CM_192_HMAC_SHA1_80           AES-CTR-192  HMAC-SHA1   RFC 6188
AES_CM_256_HMAC_SHA1_80           AES-CTR-256  HMAC-SHA1   RFC 6188
AEAD_AES_128_GCM                  AES-GCM-128  built-in    RFC 7714
AEAD_AES_256_GCM                  AES-GCM-256  built-in    RFC 7714
NULL_HMAC_SHA1_80                 none         HMAC-SHA1   integrity only
```

### MKI (Master Key Indicator)

Optional 1–4 byte field appended to ciphertext, indicating which master key
encrypted the packet. Useful when keys rotate but old packets may still arrive.
Most deployments leave MKI = 0 length.

### Replay protection

SRTP maintains a sliding-window replay list per SSRC (default 64 packets).
Late or duplicate packets get dropped silently.

### Key derivation

From master key + master salt, SRTP derives:

- session encryption key
- session authentication key
- session salt

via AES-CM as PRF, with labels 0/1/2 (RTP) and 3/4/5 (RTCP).

## SRTCP — Secure RTCP

SRTCP encrypts everything after the first 8 bytes (so the SSRC of sender stays
visible). Adds a 32-bit SRTCP index (with E-bit indicating encryption on/off)
and a separate auth tag.

```
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|     RTCP header (cleartext, 8 bytes)                          |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|     encrypted RTCP payload                                    |
|     ....                                                      |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|E|              SRTCP index (31 bits)                          |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|     authentication tag (80 bits default)                      |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

The SRTCP index is incremented per-packet (not per-RTCP-report), so each
packet has a unique counter for AES-CTR.

## Keying — How SRTP Gets Its Keys

Three approaches, in order of modern preference:

### 1. DTLS-SRTP (RFC 5764) — modern default

DTLS handshake on the media port itself. The SDP carries a fingerprint of the
DTLS certificate; the actual key material is derived from the DTLS master
secret using `EXTRACTOR-dtls_srtp` exporter.

```
a=fingerprint:sha-256 4A:AD:B9:B1:3F:82:18:3B:54:02:12:DF:3E:5D:49:6B:19:E5:7C:AB
a=setup:actpass
```

WebRTC requires this. SIP-based VoIP often supports it via "SIPS+DTLS-SRTP".

### 2. SDES (RFC 4568) — legacy, key-in-SDP

Master key carried plaintext (base64) inside the SDP body. Only safe if the
SDP transport is encrypted (SIPS over TLS). Easy to misconfigure.

```
a=crypto:1 AES_CM_128_HMAC_SHA1_80 inline:WVNfX19zSGsRX5oqHyVBmhkOHDVKzRWPTXLJtA==
```

### 3. ZRTP (RFC 6189) — opportunistic

Diffie-Hellman key exchange on the media port itself. Authentication via Short
Authentication String (SAS) read aloud by participants. No PKI, no SDP key
material.

## DTLS-SRTP

DTLS handshake happens on the media UDP port BEFORE SRTP starts:

1. Both sides exchange DTLS certificates (self-signed; identity is the fingerprint).
2. Each side verifies the peer's cert hash matches `a=fingerprint` in the SDP.
3. The DTLS exporter derives 128-bit master keys + 112-bit master salts (for
   AES-CM-128, 224 bits total per direction).
4. SRTP starts using those keys.

```
a=fingerprint:sha-256 4A:AD:B9:B1:3F:82:18:3B:54:02:12:DF:3E:5D:49:6B:19:E5:7C:AB:34:97:36:34:97:97:36:34:97:36:34
a=setup:actpass         # offerer says "I'll be either"
a=setup:active          # answerer chose "I'll start the handshake (client)"
a=setup:passive         # answerer chose "I'll wait (server)"
```

Setup negotiation rules (RFC 5763):

- Offerer SHOULD use `actpass`.
- Answerer MUST choose `active` or `passive`.
- `active` = sends ClientHello first.
- `passive` = waits for ClientHello.
- Two `passive` sides will hang forever waiting.

```bash
# Verify a remote DTLS cert fingerprint (after capturing in Wireshark):
openssl x509 -in cert.pem -fingerprint -sha256 -noout
# SHA256 Fingerprint=4A:AD:B9:B1:...
```

## SDES Keying

```
a=crypto:1 AES_CM_128_HMAC_SHA1_80 \
    inline:PS1uQCVeeCFCanVmcjkpaywdRWlhVcQa0NbVwh6dpQwY|2^31|1:1
```

Format: `a=crypto:tag suite key-params [session-params]`

Where `key-params` is `inline:base64-key|lifetime|MKI:length`:
- `base64-key` — 30 bytes for AES-128 (16 master key + 14 master salt).
- `lifetime` — `2^N` packet count, optional.
- `MKI:length` — MKI value and length in bytes, optional.

WARNING: only use over a secure transport (SIPS+TLS). Plaintext SDES = plaintext key.

## ZRTP

ZRTP works *opportunistically* — runs on top of any RTP exchange, no signaling
support needed. Messages are framed with PT=0 (PCMU lookalike) but contain
ZRTP magic numbers.

Flow:

1. Hello / HelloACK exchange.
2. Commit (one peer initiates).
3. DH Part1 / Part2 — actual Diffie-Hellman.
4. Confirm1 / Confirm2 — verify shared secret.
5. Both sides compute SRTP keys + SAS.
6. SAS (e.g., "blast-bingo") is read aloud by participants for verification.

Implementations: Linphone, Jitsi (older), Silent Circle, Zfone (Phil Zimmermann's reference).

## SDP Format (RFC 8866, supersedes RFC 4566)

SDP is line-oriented, `<type>=<value>` form. Order matters at the session
level. Each line MUST end with CRLF.

```
v=  (protocol version) — REQUIRED
o=  (originator)        — REQUIRED
s=  (session name)      — REQUIRED (use "-" if none)
i=* (session info)
u=* (URI)
e=* (email)
p=* (phone)
c=* (connection info)   — REQUIRED at session OR each media level
b=* (bandwidth)
One or more time descriptions:
  t=  (time of session) — REQUIRED
  r=* (zero or more repeat times)
z=* (time zone adjustments)
k=* (encryption key)    — DEPRECATED, never use
a=* (zero or more session-level attributes)

Zero or more media descriptions:
  m=  (media name and transport address) — REQUIRED to start a media block
  i=* (media title)
  c=* (connection info, overrides session)
  b=* (bandwidth, overrides session)
  k=* (encryption key) — DEPRECATED
  a=* (zero or more media-level attributes)
```

`*` = optional. Within a media description, attributes apply to that media.

### v= (protocol version)

```
v=0
```

Always 0. RFC 8866 has no v=1.

### o= (origin)

```
o=<username> <session-id> <session-version> <network-type> <address-type> <address>
o=- 4858277 4858277 IN IP4 192.0.2.1
```

- `username` = "-" if no user.
- `session-id` = numeric, unique per session originator. Often a Unix timestamp.
- `session-version` = increment on SDP changes (re-INVITE).
- `network-type` = `IN` (Internet).
- `address-type` = `IP4` or `IP6`.
- `address` = IP or FQDN of the offerer.

### s= (session name)

```
s=My Session
s=-                       # use dash if no real name
```

### c= (connection)

```
c=<network-type> <address-type> <connection-address>
c=IN IP4 192.0.2.1
c=IN IP6 2001:db8::1
c=IN IP4 224.2.1.1/127/3   # multicast: address/TTL/number-of-groups
```

If at session level, applies to all media unless overridden in `m=` block.

### b= (bandwidth)

```
b=<bwtype>:<bandwidth>
b=CT:1500       # Conference Total
b=AS:64         # Application-Specific (per media; usual)
b=RR:0          # RTCP RR bandwidth
b=RS:0          # RTCP SR bandwidth
b=TIAS:48000    # Transport-Independent Application-Specific (RFC 3890), bps
```

`AS` is in kbps. `TIAS` is in bps.

### t= (time)

```
t=<start-time> <stop-time>
t=0 0           # active forever (typical for SIP)
t=2873397496 2873404696   # NTP timestamps (unusual outside SAP)
```

### r= (repeat)

```
r=7d 1h 0 25h     # repeat every 7 days, 1 hour duration, offsets 0 and 25h
```

### z= (time zone adjustments)

```
z=2882844526 -1h 2898848070 0h
```

### k= (encryption key) — DEPRECATED

Never use. Use `a=crypto` or DTLS-SRTP.

### m= (media description)

```
m=<media> <port>[/<port-count>] <proto> <fmt> ...
m=audio 49170 RTP/AVP 0 8 97
m=video 51372 RTP/AVPF 96 97
m=audio 9 UDP/TLS/RTP/SAVPF 111 0 8     # 9 = "ICE will fix it"
m=audio 49170/2 RTP/AVP 0                # 49170 and 49172 (RTP+RTCP pairs)
m=application 9 UDP/DTLS/SCTP webrtc-datachannel
```

Media types: `audio`, `video`, `text`, `application`, `message`, `image`.

Transport protocol stack:

```
RTP/AVP             plain RTP, no security      (RFC 3550)
RTP/AVPF            RTP + AVPF feedback         (RFC 4585)
RTP/SAVP            SRTP                        (RFC 3711)
RTP/SAVPF           SRTP + AVPF feedback        (RFC 5124)
UDP/TLS/RTP/SAVP    DTLS-SRTP                   (RFC 5764)
UDP/TLS/RTP/SAVPF   DTLS-SRTP + AVPF            (WebRTC standard)
TCP/RTP/AVP         RTP over TCP framing        (rare)
UDP/DTLS/SCTP       WebRTC data channel         (RFC 8841)
```

The format list is payload-type numbers (for RTP transports) or strings
(`webrtc-datachannel`).

### Repeating c= and a= within m=

```
m=audio 49170 RTP/AVP 0
c=IN IP4 192.0.2.5      # overrides session-level c=
a=rtpmap:0 PCMU/8000
a=ptime:20

m=video 51372 RTP/AVPF 96
c=IN IP4 192.0.2.5
a=rtpmap:96 VP8/90000
a=rtcp-fb:96 nack pli
```

## SDP a-line Attributes (the goldmine)

### Codec attributes

```
a=rtpmap:<PT> <encoding-name>/<clock-rate>[/<channels>]
a=rtpmap:0 PCMU/8000
a=rtpmap:111 opus/48000/2
a=rtpmap:96 VP8/90000
a=rtpmap:101 telephone-event/8000

a=fmtp:<PT> <codec-specific-parameters>
a=fmtp:111 minptime=10;useinbandfec=1
a=fmtp:101 0-16
a=fmtp:96 max-fr=30;max-fs=8160
a=fmtp:97 apt=96
a=fmtp:108 profile-level-id=42e01f;level-asymmetry-allowed=1;packetization-mode=1
```

### Packet duration

```
a=ptime:<milliseconds>
a=ptime:20         # 20ms per packet (default for most VoIP)
a=maxptime:120     # max acceptable packet duration
```

### Direction (per-leg flow)

```
a=sendrecv      # both directions (default)
a=sendonly      # I send to you, do not send to me
a=recvonly      # send to me, I won't send to you
a=inactive      # neither direction (hold)
```

These can appear at session OR media level; media-level overrides.

### RTCP

```
a=rtcp:<port>           # RTCP port if NOT default (RTP+1)
a=rtcp:9 IN IP4 0.0.0.0  # placeholder for ICE
a=rtcp-mux              # multiplex RTP and RTCP
a=rtcp-rsize            # reduced-size RTCP (RFC 5506)
a=rtcp-fb:<PT> <feedback-type> [<param>]
a=rtcp-fb:* nack            # NACK feedback for all PTs
a=rtcp-fb:96 nack
a=rtcp-fb:96 nack pli       # Picture Loss Indication
a=rtcp-fb:96 ccm fir        # Full Intra Request (RFC 5104)
a=rtcp-fb:96 goog-remb      # Google REMB (deprecated; use transport-cc)
a=rtcp-fb:96 transport-cc   # transport-wide congestion control
a=rtcp-fb:96 ack rpsi
```

### ICE (RFC 8839)

```
a=ice-ufrag:<random-string>
a=ice-pwd:<random-string-22-chars-min>
a=ice-options:trickle ice2
a=ice-options:renomination
a=candidate:<foundation> <component-id> <transport> <priority>
            <connection-address> <port> typ <type>
            [raddr <related-addr> rport <related-port>]
            [tcptype <active|passive|so>]

a=candidate:842163049 1 udp 1677729535 192.0.2.1 33457 typ srflx \
            raddr 10.0.0.1 rport 33457 generation 0
a=candidate:1 1 UDP 2130706431 10.0.0.1 33457 typ host
a=candidate:2 1 UDP 1694498815 192.0.2.1 33457 typ srflx raddr 10.0.0.1 rport 33457
a=candidate:3 1 UDP 16777215 198.51.100.1 33457 typ relay raddr 10.0.0.1 rport 33457
a=candidate:4 1 TCP 1518149375 10.0.0.1 9 typ host tcptype active
a=end-of-candidates       # signals trickle complete
a=remote-candidates:1 192.0.2.1 33457   # ICE-restart hint
```

Component-id: 1 = RTP, 2 = RTCP. With `rtcp-mux` only component 1 is used.

### DTLS

```
a=fingerprint:<hash-function> <hash-value>
a=fingerprint:sha-256 4A:AD:B9:B1:3F:82:18:3B:54:02:12:DF:3E:5D:49:6B:19:E5:7C:AB:...
a=setup:actpass         # offer
a=setup:active          # answer (sends ClientHello)
a=setup:passive         # answer (waits)
a=connection:new        # always create new DTLS context (vs "existing")
a=tls-id:<id>
```

### MID and BUNDLE

```
a=mid:0                      # arbitrary identifier per m-line
a=mid:audio
a=mid:video1
a=group:BUNDLE 0 1 2         # session-level: bundle these mids on one transport
a=group:BUNDLE audio video1
a=group:LS audio video1      # lip-sync grouping
a=group:FID 0 1              # flow-identification (RTX original + retransmit)
```

### MSID (WebRTC stream/track identification)

```
a=msid:<stream-id> <track-id>
a=msid:5d7c0f1a-3c99-4f4e-93c5-f2d1c3e02cd0 audio_track_0
a=msid:- audio_track_0       # "no stream", just a track
```

### SSRC

```
a=ssrc:<SSRC> <attribute>[:<value>]
a=ssrc:1234 cname:b9bca7c66bdfa1e5
a=ssrc:1234 msid:streamid trackid
a=ssrc:1234 mslabel:streamid
a=ssrc:1234 label:trackid
```

### SSRC group

```
a=ssrc-group:<semantics> <SSRC1> <SSRC2> ...
a=ssrc-group:FID 1234 5678        # 1234=video, 5678=RTX of 1234
a=ssrc-group:FEC-FR 1234 9999     # FlexFEC for SSRC 1234
a=ssrc-group:SIM 1 2 3            # simulcast (legacy)
```

### Header extensions (RFC 8285)

```
a=extmap:<id>[/<direction>] <URI> [<extensionattributes>]
a=extmap:1 urn:ietf:params:rtp-hdrext:ssrc-audio-level
a=extmap:2 http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time
a=extmap:3 http://www.ietf.org/id/draft-holmer-rmcat-transport-wide-cc-extensions-01
a=extmap:4 urn:3gpp:video-orientation
a=extmap:5 urn:ietf:params:rtp-hdrext:toffset
a=extmap:6 urn:ietf:params:rtp-hdrext:sdes:mid
a=extmap:7 urn:ietf:params:rtp-hdrext:sdes:rtp-stream-id
a=extmap:8 urn:ietf:params:rtp-hdrext:sdes:repaired-rtp-stream-id
a=extmap-allow-mixed                                                   # one + two byte mix
```

### Crypto (SDES)

```
a=crypto:<tag> <suite> <key-params> [<session-params>]
a=crypto:1 AES_CM_128_HMAC_SHA1_80 inline:WVNfX19...
a=crypto:2 AES_CM_128_HMAC_SHA1_32 inline:WVNfX19...|2^20|1:4
a=crypto:3 AEAD_AES_256_GCM inline:Vy0vEz...
```

### Identity / origin assertion

```
a=identity:<assertion>
a=ice-lite                     # I'm an ICE lite agent
```

### Simulcast and RID (RFC 8851/8853)

```
a=rid:<id> <direction>[ <pt-list>][;<restrictions>]
a=rid:0 send pt=96;max-width=320;max-height=180;max-fps=15
a=rid:1 send pt=96;max-width=640;max-height=360;max-fps=30
a=rid:2 send pt=96;max-width=1280;max-height=720;max-fps=30
a=simulcast:send 0;1;2 recv 0
```

### Datachannel

```
m=application 9 UDP/DTLS/SCTP webrtc-datachannel
a=sctp-port:5000
a=max-message-size:262144
```

### Recording / archiving

```
a=label:<id>             # label for media stream
a=content:slides         # sharedscreen / slides / main / alt
```

### Type-of-service / QoS hints

```
a=tcap:1 RTP/SAVP RTP/AVP        # transport capabilities (RFC 5939)
a=acap:1 crypto:1 AES_CM_128_HMAC_SHA1_80 inline:...
a=pcfg:1 t=1 a=1                 # profile config
```

### ICE-related session attributes

```
a=ice-options:trickle
a=ice-options:ice2 trickle renomination
a=remote-candidates:1 192.0.2.1 33457 2 192.0.2.1 33458
```

### Time of day

```
a=tool:libsrtp 2.4.0
a=charset:UTF-8
a=lang:en
a=sdplang:en
```

## Codec Catalog — Audio

```
Codec           Bitrate          Sample Rate     Notes
-----           -------          -----------     -----
PCMU (G.711μ)   64 kbps          8000 Hz         μ-law, North America
PCMA (G.711A)   64 kbps          8000 Hz         A-law, Europe / non-NA
G.722           64 kbps          16000 Hz        Wideband; rtpmap clock=8000 historic bug
G.723           5.3 / 6.3 kbps   8000 Hz         Royalty-bearing
G.726           16/24/32/40 kbps 8000 Hz         ADPCM
G.728           16 kbps          8000 Hz         Low-delay CELP
G.729 / 729a    8 kbps           8000 Hz         Patents expired in 2017
GSM             13 kbps          8000 Hz         Legacy mobile
GSM-EFR         12.2 kbps        8000 Hz         Enhanced Full Rate
iLBC            13.3 / 15.2 kbps 8000 Hz         Loss-resilient (RFC 3951)
iSAC            10-32 kbps       16000/32000 Hz  Adaptive (Skype/WebRTC legacy)
SILK            6-40 kbps        8/12/16/24 kHz  Skype's pre-Opus codec
Opus            6-510 kbps       8/12/16/24/48k  WebRTC default; mandatory; SILK+CELT
                                                 hybrid; supports DTX, FEC
AMR (NB)        4.75-12.2 kbps   8000 Hz         Mobile narrowband; 8 modes
AMR-WB          6.6-23.85 kbps   16000 Hz        Wideband; 9 modes
EVS             5.9-128 kbps     8/16/32/48 kHz  3GPP next-gen
Speex (NB)      2.15-24.6 kbps   8000 Hz         Open-source; superseded by Opus
Speex (WB)      4-44.2 kbps      16000 Hz
Speex (UWB)     8-79.2 kbps      32000 Hz
L16             variable         44100/48000 Hz  Linear PCM 16-bit
L8              variable         8000 Hz         Linear PCM 8-bit
DAT12           variable         32000 Hz        12-bit linear PCM
LPC             2.4 kbps         8000 Hz         Very narrow
QCELP           8/4/2/1 kbps     8000 Hz         CDMA mobile
DTMF (RFC 4733) (out-of-band)    8000 Hz         telephone-event PT
CN              variable         8000 Hz         Comfort Noise (RFC 3389)
```

### Opus details

```
a=rtpmap:111 opus/48000/2
a=fmtp:111 minptime=10;useinbandfec=1;usedtx=1;stereo=1;maxplaybackrate=48000;\
           sprop-maxcapturerate=48000;cbr=0;maxaveragebitrate=64000
```

Opus parameters:

```
maxplaybackrate            8000-48000     receive-side max sample rate
sprop-maxcapturerate       8000-48000     send-side max sample rate
maxaveragebitrate          6000-510000    max bitrate (bps)
stereo                     0 | 1          decoder stereo capability
sprop-stereo               0 | 1          encoder stereo
cbr                        0 | 1          constant bitrate
useinbandfec               0 | 1          accept in-band FEC
usedtx                     0 | 1          use DTX (Discontinuous Transmission)
minptime                   3 | 5 | 10 | 20 | 40 | 60 | 120
```

### G.722 — the "8000 Hz lie"

The original RTP profile (RFC 3551) lists G.722 as 8000 Hz clock-rate. This is
WRONG. The actual sampling rate is 16000 Hz. Implementations have to do:

```
a=rtpmap:9 G722/8000      # SDP says 8000 (per the broken spec)
                          # but timestamps actually advance at 16000/sec
```

If your SDP says G.722 at 16000 Hz, some interoperability is broken. Always
advertise as `G722/8000` per RFC 3551 errata.

### AMR / AMR-WB

```
a=rtpmap:97 AMR/8000
a=fmtp:97 octet-align=1;mode-set=0,2,5,7;mode-change-period=2;\
          mode-change-neighbor=1;crc=0;robust-sorting=0;interleaving=0
a=rtpmap:98 AMR-WB/16000
a=fmtp:98 octet-align=1;mode-set=0,1,2,3,4,5,6,7,8
```

### G.729 fmtp

```
a=rtpmap:18 G729/8000
a=fmtp:18 annexb=no       # disable VAD/CNG annex B (interop with old endpoints)
```

### iLBC

```
a=rtpmap:97 iLBC/8000
a=fmtp:97 mode=20         # 20ms frames (alternative is 30)
```

### telephone-event (DTMF)

```
a=rtpmap:101 telephone-event/8000
a=fmtp:101 0-16           # accept events 0–9, *, #, A-D, plus flash hook
```

### Comfort Noise

```
a=rtpmap:13 CN/8000
a=rtpmap:118 CN/16000     # for G.722 / wideband
a=rtpmap:119 CN/32000
```

## Codec Catalog — Video

```
Codec       Notes
-----       -----
H.261       Legacy, ISDN videoconferencing
H.263       Mobile / 3G; many profile-IDs
H.263+      H.263 with extensions
H.264 (AVC) Most universal codec; profile-level-id config (Baseline/Main/High)
H.265 (HEVC) Better compression; license issues hampered adoption
VP8         Google, royalty-free; WebRTC mandatory; libvpx
VP9         Google, royalty-free; SVC support; libvpx
AV1         Royalty-free, ~30% better than HEVC; rapidly adopted
JPEG        Motion JPEG (PT=26)
MPV         MPEG video
MP2T        MPEG-2 Transport Stream (broadcast)
Theora      Xiph; rare
RV          RealVideo
DivX        legacy
```

### H.264 fmtp (RFC 6184)

```
a=rtpmap:108 H264/90000
a=fmtp:108 profile-level-id=42e01f;\
           level-asymmetry-allowed=1;\
           packetization-mode=1;\
           max-mbps=108000;\
           max-fs=3600;\
           max-cpb=2000;\
           max-dpb=15;\
           max-br=2500
```

profile-level-id breakdown (3 bytes hex):
- byte 1 = profile_idc (0x42 baseline, 0x4D main, 0x64 high, 0x53 scalable, ...)
- byte 2 = profile-iop (constraint flags)
- byte 3 = level_idc (0x1f = 3.1, 0x20 = 3.2, 0x29 = 4.1, 0x32 = 5.0)

Common pairings:
- `42e01f` = constrained baseline 3.1 (most-compatible)
- `42001f` = baseline 3.1
- `4d001f` = main 3.1
- `64001f` = high 3.1
- `640c1f` = high 3.1 with constraint flags

packetization-mode:
- 0 = Single NAL Unit (small packets)
- 1 = Non-Interleaved (Single NAL, STAP-A, FU-A)
- 2 = Interleaved (rare)

### H.265 fmtp

```
a=rtpmap:109 H265/90000
a=fmtp:109 profile-id=1;tier-flag=0;level-id=120;\
           interop-constraints=000000000000;\
           sprop-vps=...;sprop-sps=...;sprop-pps=...
```

### VP8 fmtp

```
a=rtpmap:96 VP8/90000
a=fmtp:96 max-fr=30;max-fs=8160      # max-fr = frame rate, max-fs = frame size in macroblocks
```

### VP9 fmtp

```
a=rtpmap:98 VP9/90000
a=fmtp:98 profile-id=0
a=fmtp:98 profile-id=2               # 10-bit 4:2:0
```

### AV1 fmtp (RFC 8851 + AV1 RTP draft)

```
a=rtpmap:122 AV1/90000
a=fmtp:122 level-idx=5;profile=0;tier=0
```

### RTX (RFC 4588)

```
a=rtpmap:97 rtx/90000
a=fmtp:97 apt=96;rtx-time=200       # apt = associated payload type
```

### red / ulpfec / flexfec

```
a=rtpmap:112 red/90000
a=fmtp:112 96/96                     # primary/redundant PT=96
a=rtpmap:113 ulpfec/90000
a=rtpmap:114 flexfec-03/90000
a=fmtp:114 repair-window=10000000    # microseconds
```

## SDP Negotiation (Offer/Answer, RFC 3264)

The offer/answer model:

1. **Offerer** sends an SDP describing what it can send and receive (all PTs
   it supports, all codecs, all candidates known so far).
2. **Answerer** responds with a SUBSET — only the PTs / codecs it accepts,
   and direction may flip (a `sendrecv` offer can be answered with
   `recvonly` if answerer can't send).
3. Each m-line in the answer must correspond positionally to the m-line in
   the offer. To "reject" a media stream, the answerer sets the port to 0.

```
OFFER:
m=audio 49170 RTP/AVP 0 8 18 101
a=rtpmap:0 PCMU/8000
a=rtpmap:8 PCMA/8000
a=rtpmap:18 G729/8000
a=rtpmap:101 telephone-event/8000
a=sendrecv

ANSWER:
m=audio 51020 RTP/AVP 8 101
a=rtpmap:8 PCMA/8000
a=rtpmap:101 telephone-event/8000
a=sendrecv
```

Rejection of a media stream:

```
OFFER:
m=audio 49170 RTP/AVP 0
m=video 49180 RTP/AVP 96

ANSWER:
m=audio 51020 RTP/AVP 0
m=video 0 RTP/AVP 96         # port=0 means "reject this m-line"
```

### Trickle ICE

```
INITIAL OFFER (no candidates yet):
m=audio 9 UDP/TLS/RTP/SAVPF 111
c=IN IP4 0.0.0.0
a=ice-ufrag:abc
a=ice-pwd:def123456789
a=ice-options:trickle
a=fingerprint:sha-256 ...
a=setup:actpass
a=mid:0
a=rtpmap:111 opus/48000/2
a=sendrecv
a=rtcp-mux

(trickle candidates arrive via UPDATE / INFO / signaling channel)
a=candidate:1 1 udp 2130706431 192.168.1.10 56789 typ host
a=candidate:2 1 udp 1677729535 198.51.100.1 56789 typ srflx \
              raddr 192.168.1.10 rport 56789

(end of trickle:)
a=end-of-candidates
```

### Re-INVITE / hold

To put a call on hold, send a re-INVITE with a new SDP where direction is
`sendonly` (you'll send hold music) or `inactive`.

```
m=audio 49170 RTP/AVP 0
a=sendonly                # I'll send music; expect nothing from you
```

To resume, re-INVITE with `sendrecv`.

## SDP Examples

### 1. Minimal G.711 audio

```
v=0
o=alice 1234567890 1234567890 IN IP4 192.0.2.1
s=-
c=IN IP4 192.0.2.1
t=0 0
m=audio 49170 RTP/AVP 0 8 101
a=rtpmap:0 PCMU/8000
a=rtpmap:8 PCMA/8000
a=rtpmap:101 telephone-event/8000
a=fmtp:101 0-15
a=ptime:20
a=sendrecv
```

### 2. Typical IP phone (Opus + PCMU + DTMF)

```
v=0
o=- 5234982 5234982 IN IP4 192.0.2.5
s=Yealink SIP-T54W
c=IN IP4 192.0.2.5
t=0 0
m=audio 11800 RTP/AVP 0 8 9 18 101
a=rtpmap:0 PCMU/8000
a=rtpmap:8 PCMA/8000
a=rtpmap:9 G722/8000
a=rtpmap:18 G729/8000
a=fmtp:18 annexb=no
a=rtpmap:101 telephone-event/8000
a=fmtp:101 0-15
a=ptime:20
a=sendrecv
```

### 3. WebRTC-style audio + video with BUNDLE + DTLS-SRTP

```
v=0
o=- 4858277 2 IN IP4 127.0.0.1
s=-
t=0 0
a=group:BUNDLE 0 1
a=extmap-allow-mixed
a=msid-semantic: WMS stream-id-1

m=audio 9 UDP/TLS/RTP/SAVPF 111 0 8 13 110 126
c=IN IP4 0.0.0.0
a=rtcp:9 IN IP4 0.0.0.0
a=ice-ufrag:F+A2
a=ice-pwd:Ye3kZcL6Q+5jdJYKPRwLw9rl
a=ice-options:trickle
a=fingerprint:sha-256 4A:AD:B9:B1:3F:82:18:3B:54:02:12:DF:3E:5D:49:6B:19:E5:7C:AB:34:97:36:34:97:97:36:34:97:36:34
a=setup:actpass
a=mid:0
a=extmap:1 urn:ietf:params:rtp-hdrext:ssrc-audio-level
a=extmap:2 http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time
a=extmap:3 http://www.ietf.org/id/draft-holmer-rmcat-transport-wide-cc-extensions-01
a=extmap:4 urn:ietf:params:rtp-hdrext:sdes:mid
a=sendrecv
a=msid:stream-id-1 audio-track-1
a=rtcp-mux
a=rtpmap:111 opus/48000/2
a=rtcp-fb:111 transport-cc
a=fmtp:111 minptime=10;useinbandfec=1
a=rtpmap:0 PCMU/8000
a=rtpmap:8 PCMA/8000
a=rtpmap:13 CN/8000
a=rtpmap:110 telephone-event/48000
a=rtpmap:126 telephone-event/8000
a=ssrc:1001 cname:b9bca7c66bdfa1e5
a=ssrc:1001 msid:stream-id-1 audio-track-1

m=video 9 UDP/TLS/RTP/SAVPF 96 97 98 99 100 101 122 123
c=IN IP4 0.0.0.0
a=rtcp:9 IN IP4 0.0.0.0
a=ice-ufrag:F+A2
a=ice-pwd:Ye3kZcL6Q+5jdJYKPRwLw9rl
a=ice-options:trickle
a=fingerprint:sha-256 4A:AD:B9:B1:3F:82:18:3B:54:02:12:DF:3E:5D:49:6B:19:E5:7C:AB:34:97:36:34:97:97:36:34:97:36:34
a=setup:actpass
a=mid:1
a=extmap:5 urn:ietf:params:rtp-hdrext:toffset
a=extmap:6 http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time
a=extmap:7 urn:3gpp:video-orientation
a=extmap:3 http://www.ietf.org/id/draft-holmer-rmcat-transport-wide-cc-extensions-01
a=extmap:4 urn:ietf:params:rtp-hdrext:sdes:mid
a=sendrecv
a=msid:stream-id-1 video-track-1
a=rtcp-mux
a=rtcp-rsize
a=rtpmap:96 VP8/90000
a=rtcp-fb:96 goog-remb
a=rtcp-fb:96 transport-cc
a=rtcp-fb:96 ccm fir
a=rtcp-fb:96 nack
a=rtcp-fb:96 nack pli
a=rtpmap:97 rtx/90000
a=fmtp:97 apt=96
a=rtpmap:98 VP9/90000
a=fmtp:98 profile-id=0
a=rtcp-fb:98 goog-remb
a=rtcp-fb:98 transport-cc
a=rtcp-fb:98 ccm fir
a=rtcp-fb:98 nack
a=rtcp-fb:98 nack pli
a=rtpmap:99 rtx/90000
a=fmtp:99 apt=98
a=rtpmap:100 H264/90000
a=fmtp:100 profile-level-id=42e01f;level-asymmetry-allowed=1;packetization-mode=1
a=rtcp-fb:100 goog-remb
a=rtcp-fb:100 transport-cc
a=rtcp-fb:100 ccm fir
a=rtcp-fb:100 nack
a=rtcp-fb:100 nack pli
a=rtpmap:101 rtx/90000
a=fmtp:101 apt=100
a=rtpmap:122 AV1/90000
a=rtpmap:123 rtx/90000
a=fmtp:123 apt=122
a=ssrc-group:FID 2001 2002
a=ssrc:2001 cname:b9bca7c66bdfa1e5
a=ssrc:2001 msid:stream-id-1 video-track-1
a=ssrc:2002 cname:b9bca7c66bdfa1e5
a=ssrc:2002 msid:stream-id-1 video-track-1
```

### 4. Conference participant with simulcast

```
m=video 9 UDP/TLS/RTP/SAVPF 96 97
c=IN IP4 0.0.0.0
a=mid:1
a=ice-ufrag:abc
a=ice-pwd:def
a=fingerprint:sha-256 ...
a=setup:actpass
a=rtpmap:96 VP8/90000
a=rtcp-fb:96 nack
a=rtcp-fb:96 nack pli
a=rtcp-fb:96 transport-cc
a=rtpmap:97 rtx/90000
a=fmtp:97 apt=96
a=rid:low send pt=96;max-width=320;max-height=180;max-fps=15
a=rid:mid send pt=96;max-width=640;max-height=360;max-fps=30
a=rid:high send pt=96;max-width=1280;max-height=720;max-fps=30
a=simulcast:send low;mid;high
a=sendonly
a=rtcp-mux
a=msid:streamid videotrack
```

## NAT Traversal for RTP

The fundamental NAT problem: SDP advertises an internal IP, but the actual
public IP is whatever the NAT mapped to. Three approaches:

### Symmetric RTP

Server-side learns the peer's actual `(src_ip, src_port)` from received
RTP packets and sends to *that* address, ignoring SDP. Most VoIP servers
(Asterisk, FreeSWITCH, Kamailio + rtpengine) do this by default.

```
[Asterisk]  rtp.conf   strictrtp=yes  nat=auto_force_rport,auto_comedia
[FreeSWITCH] sofia.conf  apply-nat-acl  rtp-ip=$${external_rtp_ip}
```

### ICE — Interactive Connectivity Establishment (RFC 8445)

Trickle through every possible candidate until one works. See ICE section.

### TURN — relay everything

When peer-to-peer fails, route media through a TURN server. Costs bandwidth
but always works.

### "One-way audio" — the classic NAT failure

Symptoms: A hears B but B doesn't hear A.

Causes (in order of likelihood):

1. NAT on B's side: A sent RTP to the IP B advertised (10.x.x.x), but
   that IP is unreachable. **Fix:** symmetric RTP — server replies to
   wherever B's RTP arrived from.
2. RTCP-mux mismatch: one side multiplexed RTCP onto the RTP port, the
   other didn't, so RTCP got dropped, breaking the codec's ability to
   sync. **Fix:** ensure both sides advertise `a=rtcp-mux`.
3. ICE asymmetry: one side resolved a relay candidate, the other only
   tried host. **Fix:** include both srflx and relay candidates on both sides.
4. SBC (Session Border Controller) only forwarding one direction: ACL or
   bandwidth limit. **Fix:** check SBC stats.

Diagnose:

```bash
# Capture and confirm direction:
tcpdump -ni any -nn 'udp portrange 10000-20000'
# Look for: RTP from peer to you (good), and from you to peer (good).
# If only one direction, you've found the leg that's blocked.
```

## ICE — The Algorithm (RFC 8445)

ICE harmonizes endpoints behind various NATs into a working media flow. Each
side runs the same state machine.

### Phase 1: Gather candidates

For each component (RTP=1, RTCP=2 if not muxed):

1. **host** — every local IP on every interface.
2. **srflx (server-reflexive)** — public IP/port discovered via STUN.
3. **prflx (peer-reflexive)** — discovered during connectivity checks
   (when NAT creates a binding mid-handshake).
4. **relay** — TURN relay address.

### Phase 2: Exchange (in SDP)

```
a=candidate:1 1 udp 2130706431 10.0.0.1 56789 typ host
a=candidate:2 1 udp 1677729535 198.51.100.1 56789 typ srflx \
            raddr 10.0.0.1 rport 56789
a=candidate:3 1 udp 16777215 203.0.113.1 56789 typ relay \
            raddr 10.0.0.1 rport 56789
```

### Phase 3: Connectivity checks

Each side computes `priority` for each pair. Form a check list:

```
priority = (2^32) * MIN(G,D) + 2 * MAX(G,D) + (G > D ? 1 : 0)
```

Where G = controlling agent's priority for the candidate, D = controlled.

Send STUN Binding requests to each pair. First success = a "valid" pair.

### Phase 4: Nominate

In aggressive nomination, the controlling side sets the USE-CANDIDATE
attribute on the highest-priority valid pair. In regular nomination, it
waits and re-runs checks before nominating.

### Phase 5: Media flows

Once nominated, both sides send media to that candidate pair until disconnection.

### Restart

```
a=ice-ufrag:newrandom
a=ice-pwd:newpasswordrandom
```

Updated ufrag/pwd triggers full re-gathering and re-checking. Used on
network change (cellular ↔ WiFi).

### ICE-Lite vs ICE-Full

- **Lite** — only host candidates; passive role. Used by servers with public IP.
- **Full** — gathers all candidate types, can be controlling or controlled.

A lite endpoint MUST NOT initiate; it must always be the answerer.

```
a=ice-lite           # I'm a lite agent
```

## STUN — Session Traversal Utilities for NAT (RFC 8489)

STUN's primary use: discover your public IP/port behind NAT.

```
Client → STUN Server   Binding Request
                       (transaction ID 12 bytes)

STUN Server → Client   Binding Response
                       XOR-MAPPED-ADDRESS = 198.51.100.1:33457
```

Stateless protocol. Default port: 3478 (UDP/TCP), 5349 (TLS/DTLS).

```bash
# Quickly check public IP via STUN:
stunclient stun.l.google.com 19302
# 198.51.100.1:33457
```

### Public STUN servers

```
stun.l.google.com:19302
stun1.l.google.com:19302
stun2.l.google.com:19302
stun3.l.google.com:19302
stun4.l.google.com:19302
stun.cloudflare.com:3478
stun.cloudflare.com:53            (port 53 to bypass UDP filters)
stun.ekiga.net:3478
global.stun.twilio.com:3478
```

### STUN message attributes (selected)

```
0x0001  MAPPED-ADDRESS         (deprecated, no XOR)
0x0006  USERNAME
0x0008  MESSAGE-INTEGRITY      (HMAC-SHA1 over packet)
0x0009  ERROR-CODE
0x000A  UNKNOWN-ATTRIBUTES
0x0014  REALM
0x0015  NONCE
0x0020  XOR-MAPPED-ADDRESS     (XOR with magic cookie + tid)
0x0024  PRIORITY               (ICE)
0x0025  USE-CANDIDATE          (ICE nomination)
0x8022  SOFTWARE
0x8023  ALTERNATE-SERVER
0x8028  FINGERPRINT            (CRC32)
0x8029  ICE-CONTROLLED
0x802A  ICE-CONTROLLING
```

## TURN — Traversal Using Relays around NAT (RFC 8656)

TURN extends STUN: client allocates a relay address on the TURN server,
peer sends to relay, server forwards to client, and vice versa. Costs
bandwidth (every byte of media goes through relay), so usually authenticated.

### Allocation flow

```
Client → TURN Server   Allocate Request (with credentials)
                       requested-transport=UDP

TURN Server → Client   Allocate Response
                       XOR-RELAYED-ADDRESS = 203.0.113.1:50001
                       XOR-MAPPED-ADDRESS = 198.51.100.1:33457
                       LIFETIME = 600
```

### Long-term vs short-term credentials

- **Long-term** — `username:realm:password` with HMAC over MESSAGE-INTEGRITY;
  realm and nonce echoed back; replay-resistant.
- **Short-term** — single-use credentials generated by the application
  server (e.g., REST API gives client a temporary username = `expiry:user`
  and password = HMAC of username with shared secret).

### TURN URL format (SDP not, but ICE config)

```
turn:turn.example.com:3478?transport=udp
turn:turn.example.com:3478?transport=tcp
turns:turn.example.com:5349
turns:turn.example.com:443?transport=tcp     # 443 to bypass firewalls
```

### coturn (most common open-source TURN)

```
# /etc/turnserver.conf
listening-port=3478
tls-listening-port=5349
external-ip=203.0.113.1
realm=turn.example.com
use-auth-secret
static-auth-secret=YourSharedSecretHere
total-quota=100
bps-capacity=0
no-loopback-peers
no-multicast-peers
cert=/etc/letsencrypt/live/turn.example.com/fullchain.pem
pkey=/etc/letsencrypt/live/turn.example.com/privkey.pem
```

```bash
# Run with REST-API auth:
turnserver -c /etc/turnserver.conf -v
```

## DTMF Over RTP

Three ways to signal a key press:

### 1. RFC 4733 telephone-event (RECOMMENDED)

In-band on RTP but as named events, not audio. PT typically 101.

```
a=rtpmap:101 telephone-event/8000
a=fmtp:101 0-16
```

Each key press is a sequence of RTP packets (every ~50ms during the press),
each carrying a 4-byte event payload:

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|     event     |E|R| volume    |          duration             |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

- `event` (8 bits) — 0–9, 10=*, 11=#, 12–15=A–D, 16=flash
- `E` (1 bit) — End of event (set on the LAST 3 packets to signal release)
- `R` (1 bit) — Reserved
- `volume` (6 bits) — power level (0 to -63 dBm0)
- `duration` (16 bits) — duration in timestamp units (8 kHz samples)

The marker bit is set on the FIRST packet of an event (talk-spurt rule).
For the last 3 packets, retransmit with E=1 in case of loss.

### 2. SIP INFO — out-of-band

DTMF as SIP INFO message body. Pros: travels through transcoders. Cons:
not real-time (signaling latency).

```
INFO sip:peer@example.com SIP/2.0
Content-Type: application/dtmf-relay
Content-Length: 22

Signal=5
Duration=160
```

### 3. In-band — legacy

DTMF tones encoded in audio. Survives any transcoding that preserves audio
fidelity, but G.729 / Opus may distort and break detection. AVOID.

### fmtp event range

```
a=fmtp:101 0-15           # digits 0–9, *, #, A–D
a=fmtp:101 0-16           # adds flash hook (16)
a=fmtp:101 0-15,32-49     # digits + DTMF tones for testing
```

## Comfort Noise (RFC 3389)

When a codec uses VAD (Voice Activity Detection) and stops sending during
silence, the receiver hears a hard cut. CN packets describe a noise spectrum
to synthesize so the line stays "alive".

```
a=rtpmap:13 CN/8000          # narrowband
a=rtpmap:118 CN/16000        # wideband
a=rtpmap:119 CN/32000        # super-wideband
```

CN packet payload: noise level + LPC coefficients describing spectrum shape.
Sent every ~200ms during silence (Discontinuous Transmission, DTX).

## RTP Header Extensions (RFC 8285)

Two formats:

### One-byte (legacy)

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|       0xBE    |    0xDE       |           length              |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|  ID   |  L=N  |       data    |  ID   |  L=N  |       data    |
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

ID = 1–14 (15 reserved). Length = 0–15 (data is L+1 bytes).

### Two-byte (RFC 8285 + RFC 5285 extension)

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|         0x100         |appbits|           length              |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|       ID      |     length    |       data...                 |
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

ID = 1–255. Length = 0–255.

### Common extension URIs

```
URI                                                                 Purpose
---                                                                 -------
urn:ietf:params:rtp-hdrext:ssrc-audio-level                        Audio level
urn:ietf:params:rtp-hdrext:csrc-audio-level                        Mixer-side audio
urn:ietf:params:rtp-hdrext:toffset                                 Timestamp offset
http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time         Absolute send time
http://www.ietf.org/id/draft-holmer-rmcat-transport-wide-cc-...    Transport-wide CC
urn:3gpp:video-orientation                                         CVO (orientation)
urn:ietf:params:rtp-hdrext:sdes:mid                                MID for BUNDLE
urn:ietf:params:rtp-hdrext:sdes:rtp-stream-id                      RID
urn:ietf:params:rtp-hdrext:sdes:repaired-rtp-stream-id             Repaired RID (RTX)
urn:ietf:params:rtp-hdrext:framemarking                            Frame marking
urn:ietf:params:rtp-hdrext:color-space                             Color space
http://www.webrtc.org/experiments/rtp-hdrext/playout-delay         Playout delay
http://www.webrtc.org/experiments/rtp-hdrext/video-content-type    Content type
http://www.webrtc.org/experiments/rtp-hdrext/video-timing          Video timing
urn:ietf:params:rtp-hdrext:encrypt:Y                               Encrypted ext
```

## Bandwidth Considerations

### Per-codec optimal ptime ranges

```
Codec      Frame        Optimal ptime    IP overhead %
-----      -----        -------------    -------------
PCMU/PCMA  10ms         20ms             ~25% (160B audio + 40B IP/UDP/RTP)
G.722      varies       20-30ms          ~25%
G.729      10ms         20-30ms          ~80% (10B audio + 40B header!)
GSM        20ms         20-40ms          ~60%
Opus       20ms          20-40ms          15-30%
iLBC       20/30ms      20/30ms          ~50%
```

### IP overhead

For every RTP packet:
- Ethernet header: 14 B
- IPv4 header: 20 B (40 for IPv6)
- UDP header: 8 B
- RTP header: 12 B (more with CSRC/extension)
- TOTAL pre-payload: 54 B IPv4, 74 B IPv6

A G.729 voice packet at 20ms = 20 bytes audio + 40 bytes header = 60 bytes
total = **24 kbps wire bandwidth** (vs 8 kbps codec rate, 3× overhead!).

A G.711 voice packet at 20ms = 160 bytes audio + 40 bytes header = 200 bytes
= 80 kbps wire bandwidth (vs 64 kbps codec, 1.25× overhead).

### Bandwidth budgeting

```
codec_kbps + (header_bytes * 8 / ptime_ms)
```

Examples (per direction, no SRTP):
- G.711 @ 20ms = 64 + (40*8/0.020) = 64 + 16 = **80 kbps**
- G.711 @ 30ms = 64 + (40*8/0.030) = 64 + 10.7 = **74.7 kbps**
- G.729 @ 20ms = 8 + 16 = **24 kbps**
- G.729 @ 40ms = 8 + 8 = **16 kbps**
- Opus @ 32 kbps + 20ms = 32 + 16 = **48 kbps**

### SRTP overhead

```
+10 bytes per packet (auth tag) ... +14 if using MKI
```

### Bandwidth advertising in SDP

```
b=AS:64           # 64 kbps Application-Specific (per-media)
b=TIAS:48000      # 48000 bps Transport-Independent (RFC 3890)
b=CT:1500         # 1500 kbps Conference Total
b=RR:0            # disable RTCP RR bandwidth
b=RS:0            # disable RTCP SR bandwidth
```

## Jitter & Loss Handling

### Jitter buffer

Stored packets reordered by sequence number; played back at constant rate
based on RTP timestamp. Two flavors:

- **Fixed** — playout delay is constant (e.g., 60ms). Simple, predictable.
- **Adaptive** — adjusts buffer depth based on observed jitter. Common in WebRTC.

```
typical buffer depth   30–200 ms
target packet loss     <1%
target end-to-end      <150 ms one-way for "good" voice (G.114 / R-factor)
```

### Loss detection

Sequence number gap detection. If `seq[N+2]` arrives but `seq[N+1]` doesn't,
NACK or wait for jitter buffer expiry.

### PLC — Packet Loss Concealment

Per-codec strategy to fill in lost packets:
- **G.711** — repeat last 8ms of audio with fade
- **G.729** — predict from previous LPC coefficients
- **iLBC** — built-in PLC algorithm
- **Opus** — built-in PLC + in-band FEC option

### NACK feedback (video)

```
RTPFB FMT=1 (Generic NACK)
  PID — packet ID (sequence number) of lost packet
  BLP — bitmask of subsequent lost packets
```

```
a=rtcp-fb:96 nack            # enable generic NACK
a=rtcp-fb:96 nack pli        # also Picture Loss Indication
```

NACK works well at low loss (<5%) and low RTT (<100ms). Beyond that,
retransmissions arrive too late.

### FEC — Forward Error Correction

Send redundant data so loss can be reconstructed without retransmission.

#### red (RFC 2198)

Generic redundancy: include previous packet's payload in current packet.
For audio:

```
a=rtpmap:112 red/8000
a=fmtp:112 96/96             # redundant primary PT 96 (Opus) for primary 96
```

#### ulpfec (RFC 5109)

Uneven Level Protection FEC. XOR of multiple packets.

```
a=rtpmap:113 ulpfec/90000
```

#### flexfec (RFC 8627)

Flexible 2D FEC. Better recovery for clustered losses.

```
a=rtpmap:114 flexfec-03/90000
a=fmtp:114 repair-window=10000000   # microseconds
```

## RTX — Retransmission Payload Format (RFC 4588)

Lost packets are retransmitted on a *separate* SSRC + payload type. The
retransmitted packet's first 2 bytes of payload = original sequence number.

```
a=rtpmap:96 VP8/90000
a=rtpmap:97 rtx/90000
a=fmtp:97 apt=96;rtx-time=200            # apt = associated payload type
a=ssrc-group:FID 12345 67890              # 12345=video, 67890=RTX
a=ssrc:12345 cname:abc
a=ssrc:67890 cname:abc
```

`rtx-time` is the maximum time (ms) the sender will wait before discarding
a retransmission request.

## Simulcast (RFC 8851 / 8853)

Send multiple resolutions of the same source as separate streams in ONE
m-line (modern) or multiple m-lines (legacy).

```
a=rtpmap:96 VP8/90000
a=rtcp-fb:96 nack
a=rid:low send pt=96;max-width=320;max-height=180;max-fps=15
a=rid:mid send pt=96;max-width=640;max-height=360;max-fps=30
a=rid:high send pt=96;max-width=1280;max-height=720;max-fps=30
a=simulcast:send low;mid;high
```

Receiver can select any subset (e.g., SFU forwards `low` to a mobile client,
`high` to a desktop). Each rid has its own SSRC; sender packages them with
the RID header extension.

### SVC — Scalable Video Coding

VP9 / AV1 / H.265 support multi-layer encoding within ONE stream:

- Temporal layers (frame rate)
- Spatial layers (resolution)
- Quality layers (fidelity)

Receiver selects which layers to decode. SFU can drop layers without the
sender re-encoding.

```
a=fmtp:98 profile-id=2;tier-flag=0;level-id=3
a=rtcp-fb:98 nack
a=rtcp-fb:98 nack pli
```

## BUNDLE — Multi-Media on One Transport (RFC 9143)

Multiplex audio + video + datachannel onto one ICE transport, one DTLS
context, one set of ports.

```
a=group:BUNDLE 0 1 2

m=audio 9 UDP/TLS/RTP/SAVPF 111
a=mid:0
...

m=video 0 UDP/TLS/RTP/SAVPF 96
a=mid:1
...

m=application 0 UDP/DTLS/SCTP webrtc-datachannel
a=mid:2
...
```

- Only the FIRST m-line gets a real port; subsequent bundled m-lines have port=0.
- All m-lines share the same `ice-ufrag`, `ice-pwd`, `fingerprint`.
- Demultiplexing: SSRC + MID header extension + PT.
- Falls back gracefully — older endpoints respond with port=0 for ALL m-lines
  if they don't support BUNDLE; the offerer must then either re-offer without
  BUNDLE or accept partial connectivity.

### BUNDLE policy hints

```
a=group:BUNDLE 0 1 2                 # ALL these mids must bundle
a=group:BUNDLE-ONLY 0 1 2            # signal "these MUST bundle or fail"
```

The webrtc API exposes `bundlePolicy: balanced | max-bundle | max-compat`:
- `balanced` — bundle within media types; one transport per type.
- `max-bundle` — everything on one transport.
- `max-compat` — separate transport per m-line (no bundle).

## SDP Lite vs Full

ICE roles:

- **Full** — gathers all candidate types (host, srflx, prflx, relay), can
  initiate connectivity checks, can be controlling or controlled.
- **Lite** — only host candidates, only passive (controlled) role, MUST NOT
  initiate. Used by infrastructure servers with stable public IPs (SFU,
  TURN-on-same-host, etc.).

```
a=ice-lite           # I'm a lite agent
```

Interop:

- Full + Full → both gather, full bidirectional checks. Works.
- Full + Lite → Full gathers and initiates; Lite responds. Works.
- Lite + Lite → both passive, **no one initiates**. **DOES NOT WORK.**

## Common Errors

```
SIP code   Cause                              Fix
--------   -----                              ---
488 Not Acceptable Here     SDP rejected      Inspect Warning header for hint;
                            (no codec match)  add common codec to offer
415 Unsupported Media Type  Wrong Content-    Set "Content-Type: application/sdp"
                            Type
603 Decline                 Callee rejected   N/A
500 Server Internal Error   Trans error       Check rtpengine / proxy logs
606 Not Acceptable          Reason hdr        Check Reason / Warning headers
```

```
"no codec match"       — m-line answer has empty fmt list
                         (every offered PT was rejected)

"DTLS handshake fails" — ICE worked but media is silent;
                         check fingerprint, setup:active/passive
                         pairing, certificate validity

"one-way audio"        — RTP getting from A to B but not B to A
                         (or vice versa); usually NAT or rtcp-mux
                         mismatch; symmetric RTP fix

"garbled audio"        — wrong sample rate (G.722 misadvertised),
                         clock skew, jitter buffer too shallow

"PT not recognized"    — missing a=rtpmap for a dynamic PT; receiver
                         falls back to PT-static-table guess

"ICE connectivity check failed" — host-only candidates behind NAT;
                                  TURN unavailable; firewall blocking
                                  STUN; ufrag mismatch
```

## Common Gotchas (Broken → Fixed)

### 1. G.722 misadvertised at 8 kHz

```
# BROKEN: implementation actually sends 16 kHz samples
a=rtpmap:9 G722/16000

# FIXED: per RFC 3551 §4.5.2, MUST advertise 8000
a=rtpmap:9 G722/8000
```

Some endpoints play back at half-speed if you put 16000 in the rtpmap.

### 2. Codec your software can't transcode

```
# BROKEN: upstream offers iSAC; my media server doesn't have iSAC
m=audio 49170 RTP/AVP 103       # iSAC dynamic PT
a=rtpmap:103 ISAC/16000

# FIXED: include something universal
m=audio 49170 RTP/AVP 103 0 8 111
a=rtpmap:103 ISAC/16000
a=rtpmap:0 PCMU/8000
a=rtpmap:8 PCMA/8000
a=rtpmap:111 opus/48000/2
```

Always include G.711 PCMU/PCMA in offers as a universal fallback.

### 3. rtcp-mux mismatch

```
# BROKEN: A advertises rtcp-mux, B doesn't
A:  a=rtcp-mux
B:  (no rtcp-mux)

# Result: B sends RTCP to RTP+1 port, A only listens on RTP port.
# Half the RTCP feedback goes nowhere.

# FIXED: both sides must agree
A:  a=rtcp-mux
B:  a=rtcp-mux
```

### 4. DTLS-SRTP setup mismatch

```
# BROKEN: both sides chose passive
A: a=setup:passive
B: a=setup:passive
# Neither side initiates handshake; deadlock.

# FIXED: offerer = actpass, answerer = active or passive
A (offer):  a=setup:actpass
B (answer): a=setup:active     # picks "I'll initiate"
```

### 5. Host-only candidates behind NAT

```
# BROKEN: behind NAT but only host candidates
a=candidate:1 1 udp 2130706431 10.0.0.1 56789 typ host

# FIXED: include srflx (and relay if available)
a=candidate:1 1 udp 2130706431 10.0.0.1 56789 typ host
a=candidate:2 1 udp 1677729535 198.51.100.1 33457 typ srflx \
              raddr 10.0.0.1 rport 56789
a=candidate:3 1 udp 16777215 203.0.113.1 50001 typ relay \
              raddr 10.0.0.1 rport 56789
```

### 6. SDP a-lines indented

```
# BROKEN: tab/space before a=
v=0
o=- 1 1 IN IP4 192.0.2.1
s=-
t=0 0
m=audio 49170 RTP/AVP 0
    a=rtpmap:0 PCMU/8000      # WRONG — leading whitespace breaks parsers

# FIXED: every line MUST start at column 0
v=0
o=- 1 1 IN IP4 192.0.2.1
s=-
t=0 0
m=audio 49170 RTP/AVP 0
a=rtpmap:0 PCMU/8000
```

### 7. Off-by-one in sequence-number wraparound

```c
// BROKEN: wraps at 65535
if (seq > expected_seq) drop_old(...);  // signed compare — negative after wrap

// FIXED: 16-bit modular comparison
if ((int16_t)(seq - expected_seq) > 0) drop_old(...);
```

### 8. SSRC collision

```
# BROKEN: two participants pick same SSRC
A: a=ssrc:1234 cname:userA
B: a=ssrc:1234 cname:userB

# Mixer can't tell who sent what; packet loss skewed.
# FIXED: detect collision via different CNAME on same SSRC,
#        send BYE on conflicting SSRC, allocate new random SSRC.
#
# RFC 3550 §8.2 mandates collision detection.
```

### 9. msid not parsed by older endpoints

```
# BROKEN: older Asterisk drops a=msid lines
a=msid:streamid trackid

# (some endpoints can't pair audio/video tracks for sync)
# FIXED: ensure ssrc-level msid attributes too
a=ssrc:1234 msid:streamid trackid
a=ssrc:1234 cname:abc
```

### 10. mid attribute missing → BUNDLE fails

```
# BROKEN: BUNDLE referenced mids 0,1 but m-lines lack a=mid
a=group:BUNDLE 0 1
m=audio 9 UDP/TLS/RTP/SAVPF 111
m=video 0 UDP/TLS/RTP/SAVPF 96

# Endpoint can't demultiplex incoming RTP across mids.
# FIXED:
a=group:BUNDLE 0 1
m=audio 9 UDP/TLS/RTP/SAVPF 111
a=mid:0
m=video 0 UDP/TLS/RTP/SAVPF 96
a=mid:1
```

### 11. Wrong fmtp params for codec

```
# BROKEN: fmtp specifies "annexb=yes" but software can't do VAD
a=rtpmap:18 G729/8000
a=fmtp:18 annexb=yes

# Result: peer expects DTX silence frames; software sends continuous frames.
# FIXED: match what your encoder actually does
a=fmtp:18 annexb=no
```

### 12. ICE ufrag/pwd not regenerated on restart

```
# BROKEN: re-INVITE with same ufrag/pwd → no ICE restart
re-INVITE  a=ice-ufrag:abc
                 a=ice-pwd:def

# FIXED: changing creds triggers full re-gather + re-check
re-INVITE  a=ice-ufrag:newrandom
                 a=ice-pwd:newpwdrandom
```

### 13. STUN/TURN over restrictive firewall

```
# BROKEN: corporate firewall blocks UDP/3478
turn:turn.example.com:3478?transport=udp

# FIXED: TURN over TCP/443 (TLS) — looks like HTTPS
turns:turn.example.com:443?transport=tcp
```

### 14. ptime mismatch

```
# BROKEN: A=20ms, B=10ms; B sends 2 packets per A's 1 expected.
A: a=ptime:20
B: a=ptime:10

# Some implementations adapt; others overflow jitter buffer.
# FIXED: match ptime on both sides; offer/answer should converge.
```

### 15. rtcp-fb advertised but not implemented

```
# BROKEN: SDP says we'll send NACK feedback, but our code doesn't
a=rtcp-fb:96 nack pli

# Sender expects PLI on heavy loss; we never send it; sender shows frozen video.
# FIXED: only advertise feedback your endpoint can actually generate.
```

### 16. SDES key reuse across re-INVITEs

```
# BROKEN: same crypto key in re-INVITE → replay risk
INVITE-1: a=crypto:1 AES_CM_128_HMAC_SHA1_80 inline:KEY1
re-INVITE: a=crypto:1 AES_CM_128_HMAC_SHA1_80 inline:KEY1

# FIXED: every offer/answer MUST regenerate crypto material
re-INVITE: a=crypto:1 AES_CM_128_HMAC_SHA1_80 inline:KEY2
```

## Diagnostic Tools

### Wireshark RTP analyzer

```
1. Capture pcap
2. Telephony → RTP → RTP Streams
3. Select stream → Analyze
4. Shows: lost packets, out-of-order, max delta, max jitter
5. Player: Telephony → RTP → RTP Player
   - decode actual audio for many static codecs (G.711)
   - export to .au / .wav for forensics
```

```bash
# Start a Wireshark capture from CLI:
tshark -i any -nn -f 'udp portrange 10000-20000' -w call.pcap
# Replay through analyzer offline:
wireshark call.pcap
```

### sngrep (TUI for SIP+RTP)

```bash
# Live SIP signaling:
sudo sngrep
# Press F1 / ? for help
# F2 = filter, F4 = settings, F8 = save
# Inside a call view: 'r' to show RTP streams
```

### tcpdump (raw capture)

```bash
sudo tcpdump -ni any -w call.pcap 'udp portrange 10000-20000'
# read back:
tcpdump -nn -r call.pcap | head -20
```

### rtpengine (Kamailio media proxy stats)

```bash
ngcp-rtpengine-ctl  list totals
ngcp-rtpengine-ctl  list sessions
ngcp-rtpengine-ctl  list session <call-id>
# Shows: bytes/packets/errors/jitter per call
```

### chrome://webrtc-internals/

In Chromium-based browsers, navigate to `chrome://webrtc-internals/`.
Shows live PeerConnection stats:
- `stats-graphs` — bitrate, jitter, packetLost, RTT
- ICE candidate pair list with state
- Selected pair (the "winning" candidate)
- DTLS transport state

### Firefox: about:webrtc

Same idea; click "Save Page" to dump a JSON of all stats.

### gst-launch / GStreamer for synthetic streams

```bash
# Send a 1kHz tone via RTP/PCMU:
gst-launch-1.0 audiotestsrc ! audioconvert ! audioresample ! \
   audio/x-raw,rate=8000 ! mulawenc ! rtppcmupay ! \
   udpsink host=192.0.2.1 port=49170

# Receive and play:
gst-launch-1.0 udpsrc port=49170 caps="application/x-rtp,media=audio,clock-rate=8000,encoding-name=PCMU,payload=0" \
   ! rtppcmudepay ! mulawdec ! audioconvert ! autoaudiosink
```

### sipp (load testing)

```bash
sipp -sf scenario.xml -m 1 -s 12345 192.0.2.1
```

Built-in scenarios: `uac` (user agent client), `uas` (server), `uac_pcap`
(replay RTP from pcap). Useful for bulk-call media testing.

## RTP Engine — Open-Source Media Proxy

[rtpengine](https://github.com/sipwise/rtpengine) is a kernel-bypass media
proxy used with Kamailio/OpenSIPS. Forwards RTP/SRTP through known media-server
IP, terminates DTLS-SRTP, transcodes if compiled with libavcodec.

### Control protocol (ng)

Bencoded UDP messages from SIP proxy to rtpengine:

```
d6:command5:offer7:call-id14:abc123@example8:from-tag5:fromt9:sdp-bodyN:<sdp>e

Response:
d6:result2:ok7:sdp-bodyN:<modified-sdp>e
```

Common commands:
- `offer` — register call, return modified SDP with rtpengine's media IP/port
- `answer` — receive answer, finalize media path
- `delete` — tear down
- `query` — current stats

### Kamailio configuration

```
loadmodule "rtpengine.so"
modparam("rtpengine", "rtpengine_sock", "udp:127.0.0.1:2223")

# In route():
rtpengine_offer("force trust-address replace-origin replace-session-connection ICE=force RTP/SAVPF");
rtpengine_answer("force trust-address replace-origin replace-session-connection");
```

### rtpengine flags

```
force                     always handle, even local-local
trust-address             trust SDP addresses (vs. learn from packets)
replace-origin            replace o= with proxy IP
replace-session-connection  replace c= at session level
ICE=force | remove        force ICE on or strip ICE
RTP/AVP | RTP/SAVP        force unencrypted or encrypted
TOS=0xb8                  set DSCP/TOS on forwarded packets
record-call=yes           record audio to disk
codec-mask-PCMU           mask out a codec
codec-strip-G729          remove a codec from offer
codec-transcode-Opus      enable transcoding to Opus
```

### Stats query

```bash
echo 'd7:command5:queryd7:call-id5:abc123ee' | nc -u 127.0.0.1 2223
```

## Idioms — Production Defaults

```
1.  Always use rtcp-mux on new deployments.
2.  Use DTLS-SRTP for any public-facing media; SDES only on
    fully-internal SIPS over TLS.
3.  Prefer Opus over G.711 when both sides support it (better quality
    at half the bandwidth).
4.  ptime=20 is the sweet spot for latency vs overhead.
5.  Use ICE for any media that crosses a NAT boundary (= almost everything).
6.  Run TURN with TCP/443 fallback (firewalls).
7.  rtpengine or FreeSWITCH for media anchoring; never trust RTP path
    to traverse NAT unaided.
8.  Test with both Wireshark RTP analyzer (packet-level) AND ear playback
    (decoded audio). One-way audio shows up in playback, not just stats.
9.  Always log SIP+SDP at signaling level (sngrep, Homer, sipcapture)
    for post-mortem.
10. CNAME consistency — same CNAME for every SSRC of one participant
    so the mixer can group audio + video.
11. Generate fresh SRTP keys on every offer/answer iteration.
12. Pin codec list to ~3-4 well-tested codecs; offering 15 codecs creates
    SDP edge cases nobody tests.
13. Profile-level-id=42e01f for H.264 unless you have a reason for higher;
    it's the most-compatible.
14. Set b=AS or b=TIAS to bound media; otherwise uncapped Opus can spike.
15. Disable in-band DTMF detection on transcoders; rely on telephone-event.
16. Use BUNDLE for WebRTC; separate ports only for legacy SIP interop.
17. ICE-Lite endpoints on servers with public IPs (SFU, MCU); ICE-Full
    for clients.
```

## See Also

- sip-protocol
- asterisk
- freeswitch
- ip-phone-provisioning
- sip-trunking
- tls

## References

- RFC 3550 — RTP: A Transport Protocol for Real-Time Applications
- RFC 3551 — RTP Profile for Audio and Video Conferences with Minimal Control
- RFC 3389 — RTP Payload for Comfort Noise (CN)
- RFC 3611 — RTCP Extended Reports (XR)
- RFC 3711 — Secure Real-time Transport Protocol (SRTP)
- RFC 3890 — TIAS bandwidth modifier for SDP
- RFC 3951 — Internet Low Bit Rate Codec (iLBC)
- RFC 4566 — SDP: Session Description Protocol (obsoleted by 8866)
- RFC 4568 — SDP Security Descriptions for Media Streams (SDES)
- RFC 4585 — Extended RTP Profile for RTCP-Based Feedback (RTP/AVPF)
- RFC 4588 — RTP Retransmission Payload Format (RTX)
- RFC 4733 — RTP Payload for DTMF Digits, Telephony Tones, and Telephony Signals
- RFC 5104 — Codec Control Messages in AVPF (FIR, TMMBR, TMMBN, TSTR, TSTN)
- RFC 5109 — RTP Payload Format for Generic Forward Error Correction (ulpfec)
- RFC 5124 — Extended Secure RTP Profile for AVPF (RTP/SAVPF)
- RFC 5285 — A General Mechanism for RTP Header Extensions (obsoleted by 8285)
- RFC 5506 — Reduced-Size RTCP
- RFC 5761 — Multiplexing RTP Data and Control Packets on a Single Port (rtcp-mux)
- RFC 5763 — Framework for Establishing a Secure RTP Security Context using DTLS
- RFC 5764 — DTLS Extension to Establish Keys for SRTP and SRTCP
- RFC 6184 — RTP Payload Format for H.264 Video
- RFC 6188 — The Use of AES-192 and AES-256 in Secure RTP
- RFC 6189 — ZRTP: Media Path Key Agreement for Unicast Secure RTP
- RFC 7714 — AES-GCM Authenticated Encryption in SRTP
- RFC 7798 — RTP Payload Format for High Efficiency Video Coding (HEVC, H.265)
- RFC 8285 — A General Mechanism for RTP Header Extensions
- RFC 8445 — Interactive Connectivity Establishment (ICE)
- RFC 8489 — Session Traversal Utilities for NAT (STUN)
- RFC 8627 — RTP Payload Format for Flexible Forward Error Correction (FlexFEC)
- RFC 8656 — Traversal Using Relays around NAT (TURN)
- RFC 8829 — JavaScript Session Establishment Protocol (JSEP)
- RFC 8839 — SDP Offer/Answer Procedures for ICE
- RFC 8841 — SCTP-Based Media Transport in WebRTC Data Channels
- RFC 8843 — Negotiating Media Multiplexing Using SDP (early BUNDLE)
- RFC 8851 — RTP Payload Format Restrictions
- RFC 8853 — Using Simulcast in SDP and RTP Sessions
- RFC 8866 — SDP: Session Description Protocol (replaces 4566)
- RFC 9143 — Negotiating Media Multiplexing Using SDP (BUNDLE)
- IANA RTP Payload Types registry — https://www.iana.org/assignments/rtp-parameters/
- IANA SDP Parameters registry — https://www.iana.org/assignments/sdp-parameters/
- IANA STUN Parameters registry — https://www.iana.org/assignments/stun-parameters/
- WebRTC W3C spec — https://www.w3.org/TR/webrtc/
- coturn TURN server — https://github.com/coturn/coturn
- rtpengine — https://github.com/sipwise/rtpengine
- sngrep — https://github.com/irontec/sngrep
