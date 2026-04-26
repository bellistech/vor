# rtpengine

In-kernel + userspace media proxy from Sipwise; the deferred-decision-on-media model; alternative to rtpproxy and mediaproxy. Handles RTP/RTCP forwarding, NAT traversal, DTLS-SRTP, ICE termination, transcoding, and recording for SIP/WebRTC stacks (Kamailio, OpenSIPS, FreeSWITCH, Asterisk).

## Setup

rtpengine is the de facto media proxy in the open-source SIP world. It sits between two RTP endpoints and relays media packets, performing NAT keep-alive, address rewriting, transcoding, encryption negotiation, and recording. The signaling stack (Kamailio/OpenSIPS) controls it via the **ng control protocol** — a bencoded UDP request/response wire.

```
       SIP signaling                       SIP signaling
caller ────────────► [Kamailio + rtpengine ctrl] ──────► callee
                              │ ng-protocol over UDP
                              ▼
        RTP ────────────► [rtpengine] ────────────► RTP
```

Key idea: rtpengine receives an `offer` from the proxy carrying the caller's SDP, rewrites the SDP to point media at rtpengine's own IP/port, and returns the rewritten SDP. The proxy forwards the rewritten SDP to the callee. When the answer comes back, rtpengine rewrites it the same way for the caller. Both endpoints now send media to rtpengine, which relays it.

This is the **deferred-decision-on-media model**: rtpengine doesn't pick the source/destination port pair until the first RTP packet arrives ("port latching"), so it tolerates NAT'd endpoints whose SDP-advertised port doesn't match the actually-sourced port.

Install (Debian/Ubuntu):

```bash
sudo apt install ngcp-rtpengine-daemon ngcp-rtpengine-iptables ngcp-rtpengine-kernel-dkms
```

Sipwise apt repo (stable):

```bash
echo "deb https://deb.sipwise.com/spce/bookworm/ bookworm main" \
  | sudo tee /etc/apt/sources.list.d/sipwise.list
curl -fsSL https://deb.sipwise.com/spce/sipwise.gpg | sudo apt-key add -
sudo apt update
sudo apt install ngcp-rtpengine-daemon
```

Build from source:

```bash
git clone https://github.com/sipwise/rtpengine.git
cd rtpengine
sudo apt install build-essential debhelper iptables-dev libavcodec-dev \
  libavfilter-dev libavformat-dev libavutil-dev libbencode-perl libcrypt-openssl-rsa-perl \
  libcrypt-rijndael-perl libcurl4-openssl-dev libdigest-crc-perl libdigest-hmac-perl \
  libevent-dev libglib2.0-dev libhiredis-dev libio-multiplex-perl libio-socket-inet6-perl \
  libiptc-dev libjson-glib-dev libnet-interface-perl libpcap0.8-dev libpcre3-dev \
  libsocket6-perl libspandsp-dev libssl-dev libsystemd-dev libxmlrpc-core-c3-dev \
  markdown perl pkg-config zlib1g-dev gperf default-libmysqlclient-dev libmosquitto-dev
make with_iptables_option=yes
sudo make install
```

After install, verify the userspace daemon and (optionally) the kernel module:

```bash
rtpengine --version
sudo modprobe xt_RTPENGINE
sudo lsmod | grep RTPENGINE
```

If the kernel module isn't loaded rtpengine still works — it just relays every packet through userspace, which is fine for tens to a few hundred concurrent calls but burns CPU at scale.

## Architecture

rtpengine is two pieces:

- **Userspace daemon** (`/usr/sbin/rtpengine`) — owns the ng control protocol, parses SDP, manages per-call state, terminates DTLS-SRTP, runs the transcoding pipeline, programs the kernel module.
- **Kernel module** (`xt_RTPENGINE`) — an iptables `mangle/forward` target that intercepts packets matching a kernel session table and forwards them with rewritten src/dst directly inside the kernel, bypassing the network stack.

The kernel module is the key to scale. Once both endpoints have started sending RTP, the daemon "promotes" the session into the kernel: the daemon writes a tuple of `<src-ip, src-port, dst-ip, dst-port, encryption-context>` into `/proc/rtpengine/<table>/control`, and from that point on, packets matching that tuple are forwarded entirely in the kernel. The userspace daemon never sees the packets.

```
            ┌──────────────────────────────┐
            │ rtpengine userspace daemon   │
            │  - ng-protocol               │
            │  - SDP rewriting             │
            │  - DTLS-SRTP                 │
            │  - transcoding               │
            │  - state mgmt                │
            └────────────┬─────────────────┘
                         │ writes session entries
                         ▼
            ┌──────────────────────────────┐
            │ /proc/rtpengine/<table>/...  │
            └────────────┬─────────────────┘
                         │ kernel session table
                         ▼
   RTP in ─────► [iptables -j RTPENGINE] ─────► RTP out
                  (kernel fast path)
```

Per-call session table holds:

- Local socket (rtpengine's own IP + port pair, one per direction per stream).
- Two remote endpoints (caller + callee).
- Encryption context (SRTP keys, SSRC, ROC, sequence-number window).
- Stat counters (packets, bytes, jitter, loss).
- Recording state.

Userspace falls back to handling the packets itself when:

- The kernel module is not loaded.
- The session needs SRTP (older versions; modern versions can do SRTP in-kernel).
- Transcoding is enabled — every packet must transit userspace through ffmpeg/spandsp.
- Recording is enabled in "decoded" mode.
- The packet is RTCP, since RTCP munging happens in userspace.

## Kernel Module

The kernel module is named `xt_RTPENGINE` (the `xt_` prefix follows the Netfilter `xtables` extension naming). Once loaded it provides:

- An iptables target `-j RTPENGINE` (with an `--id <table-id>` parameter selecting which session table to consult).
- A `/proc/rtpengine/<table-id>/` filesystem hierarchy for control and introspection.

Module parameters:

```bash
modinfo xt_RTPENGINE
# parm: proc_mask:proc filesystem permission mask (uint)
# parm: proc_uid:proc filesystem owner uid (uint)
# parm: proc_gid:proc filesystem owner gid (uint)
```

Load with permissions for an unprivileged daemon user:

```bash
sudo modprobe xt_RTPENGINE proc_uid=$(id -u rtpengine) proc_gid=$(id -g rtpengine)
```

The iptables rule that activates the kernel-bypass path:

```bash
sudo iptables -I INPUT  -p udp -m udp --dport 30000:40000 -j RTPENGINE --id 0
sudo iptables -I OUTPUT -p udp -m udp --sport 30000:40000 -j RTPENGINE --id 0
```

(Use `ip6tables` for IPv6.) The port range here must match the rtpengine daemon's `port-min`/`port-max` configuration.

`/proc/rtpengine/<table>/info` shows live stats:

```bash
cat /proc/rtpengine/0/info
# Version: 11.5.1.6
# Number of streams: 1284
# Number of forwarded packets: 12849392184
# Number of forwarded bytes: 2154872834930
```

`/proc/rtpengine/<table>/list` enumerates active sessions; `/proc/rtpengine/<table>/blist` is a binary-format equivalent for high-volume scraping. `/proc/rtpengine/<table>/control` is the write-only channel rtpengine uses to push session entries.

The bypass-Linux-routing optimization: when a packet matches the RTPENGINE target, the module rewrites the IP/UDP headers and pushes the packet directly out the matching network device, skipping conntrack, the routing decision, and most of the netfilter pipeline. On modern kernels this can sustain several gigabit of mid-call media on a single core.

## Userspace Daemon

The userspace daemon is written in C and uses GLib + libevent + OpenSSL + ffmpeg (libavcodec/libavformat) + libspandsp for narrow-band codecs. It:

- Listens on the **ng control port** (default UDP/22222) for offer/answer/delete/query/list commands.
- Parses incoming SDP, allocates UDP socket pairs from `port-min..port-max`, mints fresh transport-layer attributes, returns rewritten SDP.
- Pushes session entries into the kernel module when the session "stabilizes" (both sides sending).
- Falls back to userspace forwarding when the kernel module is unavailable or when SRTP/transcoding/recording is in play.
- Maintains call-id/from-tag/to-tag indexed session table.
- Emits stats to Graphite/Prometheus/Homer-HEP if configured.

Key threads:

```
main thread       — control protocol, signal handling
poller thread     — epoll over media sockets
timer thread      — janitor: timeouts, recordings, ICE re-checks
transcoding pool  — N threads, codec encode/decode pipeline
DTLS thread       — TLS handshake state machine
```

## Control Protocol (NG)

The "ng" (next-gen) protocol is rtpengine's primary control wire. Successor to the older "rtpproxy" line-protocol.

- Transport: UDP by default, but TCP and UDP-TLS are supported.
- Default port: **22222**.
- Encoding: **bencode** (the BitTorrent encoding) — like JSON but with simpler grammar and exact length-prefixed strings, which avoids escaping headaches when SDP contains binary or special chars.
- Each request is a bencoded dict; responses likewise.
- Each message is prefixed with a unique cookie (a 1–32 character ASCII string) so the daemon can correlate the response with the request.

Wire format, schematically:

```
<cookie> SP <bencoded-dict>
```

The cookie is just a token (often a Kamailio internal incrementing counter); rtpengine echoes it in the response so the proxy can match.

Configure the daemon listen socket in `/etc/rtpengine/rtpengine.conf`:

```ini
[rtpengine]
listen-ng       = 127.0.0.1:22222
listen-tcp-ng   = 127.0.0.1:22223     # optional TCP variant
listen-udp-ng   = 127.0.0.1:22222     # alias for listen-ng
listen-tls-ng   = 0.0.0.0:22224       # TLS-protected ng (rare)
```

In Kamailio:

```cfg
modparam("rtpengine", "rtpengine_sock", "udp:127.0.0.1:22222")
```

Multiple rtpengine instances can be load-balanced by listing several `rtpengine_sock` lines; Kamailio picks one per call-id.

## Bencode Format

bencode (rhymes with "encode") is the encoding used by BitTorrent and rtpengine. Compared to JSON it's:

- Length-prefixed strings — `4:spam` is the string `spam`. No escaping.
- Integers — `i42e` is `42`.
- Lists — `l<elements>e` — e.g. `l4:spami42ee` is `["spam", 42]`.
- Dicts — `d<key1><val1>...e` with keys lexicographically ordered — e.g. `d3:cow3:moo4:spam4:eggse` is `{cow: "moo", spam: "eggs"}`.

Why bencode for rtpengine: SDP can carry arbitrary text (passwords in SDES `crypto:` lines, base64 fingerprints, ICE candidates) and embedding it inside JSON-with-escaping is fragile. With bencode, an SDP blob is just `<len>:<sdp-bytes>` and you copy bytes verbatim.

A worked example. The following dict:

```
{
  "command": "offer",
  "call-id": "abc",
  "from-tag": "x"
}
```

bencodes to (with keys sorted):

```
d7:call-id3:abc7:command5:offer8:from-tag1:xe
```

And the wire-level message becomes:

```
1234_1 d7:call-id3:abc7:command5:offer8:from-tag1:xe
```

where `1234_1` is the cookie chosen by the caller.

## Offer Command

Sent by the proxy (Kamailio) when it sees an INVITE carrying SDP. rtpengine allocates media sockets and rewrites the SDP.

Request dict shape:

```json
{
  "command":   "offer",
  "call-id":   "8B0EBDF8-B3A1-44B6@10.0.0.1",
  "from-tag":  "as3f9d2c4b",
  "sdp":       "v=0\r\no=- ...\r\nm=audio 5004 RTP/AVP 0 8 101\r\n...",
  "flags":     ["trust-address", "replace-origin", "replace-session-connection"],
  "transport-protocol": "RTP/AVP",
  "ICE":       "remove",
  "DTLS":      "off",
  "SDES":      "off"
}
```

Response dict:

```json
{
  "result": "ok",
  "sdp":    "v=0\r\no=- ...\r\nm=audio 30142 RTP/AVP 0 8 101\r\nc=IN IP4 203.0.113.5\r\n..."
}
```

The returned SDP has:

- `o=` and `c=` lines rewritten to rtpengine's public IP (because `replace-origin` and `replace-session-connection` were requested).
- `m=` ports replaced with newly allocated rtpengine ports.
- All `a=candidate:` lines rewritten or stripped (per ICE flag).
- DTLS/SDES attributes added or removed per flags.

Kamailio uses the returned SDP as the body of the outgoing INVITE.

## Answer Command

Sent by the proxy when the 200 OK with answer SDP arrives from the callee. rtpengine pairs it with the prior offer (matched on call-id + from-tag) and rewrites the answer.

```json
{
  "command":  "answer",
  "call-id":  "8B0EBDF8-B3A1-44B6@10.0.0.1",
  "from-tag": "as3f9d2c4b",
  "to-tag":   "B72A1F89D",
  "sdp":      "v=0\r\no=- ...\r\nm=audio 6042 RTP/AVP 0 8 101\r\n..."
}
```

Response carries rewritten SDP that the proxy forwards back to the caller. After this exchange both endpoints have rtpengine's IP/port pair on their side and start sending media.

## Delete Command

Tear down a session.

```json
{
  "command":  "delete",
  "call-id":  "8B0EBDF8-B3A1-44B6@10.0.0.1",
  "from-tag": "as3f9d2c4b",
  "flags":    ["delete-delay-30"]
}
```

`delete-delay-N` keeps the session around for N seconds before removal, useful when a stray RE-INVITE may still arrive. Optionally returns aggregated stats:

```json
{
  "result":   "ok",
  "totals": {
    "RTP":   { "packets": 84132, "bytes": 13461120, "errors": 0 },
    "RTCP":  { "packets": 412,   "bytes": 47800,   "errors": 0 }
  }
}
```

Kamailio convention: always include `from-tag`. Not including it deletes _all_ sessions for that call-id (an accident waiting to happen).

## Query Command

Snapshot of an active call.

```json
{ "command": "query", "call-id": "8B0EBDF8-B3A1-44B6@10.0.0.1" }
```

Response:

```json
{
  "result":      "ok",
  "created":     1714045200,
  "last_signal": 1714045287,
  "tags": {
    "as3f9d2c4b": {
      "created":   1714045200,
      "tag":       "as3f9d2c4b",
      "in_dialog": true,
      "medias": [
        {
          "index":  1,
          "type":   "audio",
          "protocol": "RTP/AVP",
          "streams": [
            {
              "local port":    30142,
              "endpoint":      "192.0.2.10:5004",
              "advertised":    "192.0.2.10:5004",
              "RTP":  { "packets": 84132, "bytes": 13461120 },
              "RTCP": { "packets": 412,   "bytes": 47800   },
              "stats": { "loss": 0, "jitter": 4, "rtt": 38 }
            }
          ]
        }
      ]
    }
  }
}
```

## List Command

Enumerate active calls.

```json
{ "command": "list", "limit": 100 }
```

Response:

```json
{
  "result": "ok",
  "calls":  [
    "8B0EBDF8-B3A1-44B6@10.0.0.1",
    "F2A12C00-CE38-4ABE@10.0.0.1",
    "6D4118AA-2241-4FFF@10.0.0.1"
  ]
}
```

Pair with `query` to deep-inspect each. There's also `statistics` which returns daemon-wide totals (calls, bytes, packets), used by Prometheus exporters.

## ng Protocol Options

A request dict can carry these top-level keys:

- `flags` — list of behavior toggles (see "flags" section).
- `replace` — list of SDP fields to rewrite (`origin`, `session-connection`, `SDP-version`, `username`, `session-name`, `zero-address`).
- `transport-protocol` — coerce m= line protocol: `RTP/AVP`, `RTP/SAVP`, `RTP/AVPF`, `RTP/SAVPF`, `UDP/TLS/RTP/SAVPF`.
- `ICE` — `remove`, `force`, `force-relay`, `default`.
- `DTLS` — `passive`, `active`, `off`.
- `SDES` — `off`, `unencrypted_srtp`, `unencrypted_srtcp`, `unauthenticated_srtp`, `encrypted_srtp`, `encrypted_srtcp`, `authenticated_srtp`.
- `transcode-codec` — list of codecs to add to the offer when the peer doesn't advertise them, e.g. `["PCMA", "PCMU", "opus/48000/2"]`.
- `codec` — sub-dict with `strip`, `offer`, `mask`, `accept`, `consume`, `transcode`, `set` lists.
- `record-call` — `on`/`off`/`yes`/`no`.
- `metadata` — opaque string stored alongside the recording.
- `received-from` — `["IP4", "192.0.2.10"]` to override the `c=` source detection.
- `media-address` — force rtpengine to use this address for the media advert.
- `direction` — `[interface-A, interface-B]` for multi-NIC bridging.
- `T.38` — fax handling: `decode`, `force`, `stop`, `no-ECM`, `no-V.17`...
- `delay-buffer` — packet jitter buffer in ms.
- `volume`, `repacketize`, `ptime` — audio shaping.

## SDES SRTP Integration

SDES is the older SRTP-key-exchange-in-SDP scheme: keys appear in the SDP as `a=crypto:` lines.

rtpengine can:

- Receive SDES SRTP from one side and emit plain RTP toward the other.
- The inverse — receive plain RTP and add SDES toward the encrypted side.
- Re-encrypt with fresh keys both sides for "key isolation" — useful when you don't trust either endpoint.

Configure via the `SDES` key + `transport-protocol` change:

```json
{
  "command": "offer",
  "...":     "...",
  "transport-protocol": "RTP/SAVP",
  "SDES":    "encrypted_srtp"
}
```

A common bridging pattern: WebRTC client speaks `UDP/TLS/RTP/SAVPF` (DTLS-SRTP), and an Asterisk backend speaks plain `RTP/AVP`. rtpengine terminates DTLS-SRTP on the WebRTC leg and emits plain RTP on the Asterisk leg. From Asterisk's perspective there's no encryption at all.

## DTLS-SRTP Termination

DTLS-SRTP is the WebRTC-mandated key exchange: the two RTP endpoints run a DTLS handshake over the same UDP 5-tuple they will use for RTP, derive SRTP keys from the DTLS master secret, and switch to SRTP. The DTLS fingerprint is exchanged in SDP `a=fingerprint:` lines and validated by SDP-signed identity (in WebRTC, the `a=identity:` extension or simply by trusting the signaling channel).

rtpengine's role:

- Hold a long-lived self-signed DTLS certificate (configured via `dtls-cert-file`/`dtls-key-file` or auto-generated in memory).
- On `offer`, advertise its own fingerprint.
- On the wire, complete the DTLS handshake against the WebRTC client (passive role by default; the client always initiates).
- Derive SRTP keys; encrypt/decrypt RTP at line speed.

```ini
[rtpengine]
dtls-cert-file = /etc/rtpengine/cert.pem
dtls-key-file  = /etc/rtpengine/key.pem
dtls-mtu       = 1200
```

Mutual auth is **not required** — rtpengine accepts whatever client cert the peer presents and matches the fingerprint advertised in SDP. (The signaling layer is what authenticates the user; the DTLS fingerprint binds that user to the encrypted media stream.)

DTLS handshakes happen in userspace; once the SRTP context is derived, packet forwarding can be promoted into the kernel module on modern rtpengine versions (with the new in-kernel SRTP support).

## ICE Termination

ICE (Interactive Connectivity Establishment, RFC 8445) is how WebRTC traverses NATs: each side gathers candidate transport addresses (host, srflx via STUN, relay via TURN), exchanges them in SDP, and probes pairs with STUN binding requests until one succeeds.

rtpengine acts as **ICE-Lite**: it does not gather candidates, it only responds to peer probes. Its public IP/port pair is exactly one candidate (host type, since rtpengine is on the public Internet by design). The WebRTC client picks rtpengine's candidate, sends a STUN binding request, rtpengine replies with the binding response, the client switches to media, done.

```
WebRTC client (full ICE)  ←─ STUN check ─→  rtpengine (ICE-Lite)
                                                  │
                                                  │ plain RTP
                                                  ▼
                                              Asterisk
```

Configure in the offer:

```json
{ "command": "offer", "...": "...", "ICE": "force" }
```

`force` makes rtpengine add `a=ice-lite` and `a=ice-ufrag`/`a=ice-pwd` and `a=candidate:` lines. `remove` strips ICE from outgoing SDP for legacy peers. `default` only reflects what the offerer sent.

## Codec Transcoding

rtpengine can transcode between any pair of codecs that ffmpeg/spandsp build with. Common pairs:

- G.711 (PCMA/PCMU) ↔ G.722 (HD, but legacy)
- G.711 ↔ G.729 (low-bitrate, license-encumbered)
- G.711 ↔ Opus (WebRTC default)
- G.711 ↔ iLBC (older mobile)
- G.711 ↔ AMR-NB / AMR-WB (carrier-side mobile)
- T.38 fax pass-through and decode

CPU cost per stream (rough, on a modern x86-64 core):

- G.711 ↔ G.722: ~2% per call (cheap; cosine transform).
- G.711 ↔ Opus: ~5–8% per call.
- G.711 ↔ G.729: ~10–15% (license required for unrestricted use).
- G.711 ↔ AMR-WB: ~10–20%.

So a single rtpengine box with 8 cores can transcode roughly 100–300 concurrent calls depending on the codec mix. **Size your transcoding pool accordingly.** Pacemaker integration: fail over the rtpengine VIP between two transcoding boxes, with Redis as the shared call-state store, so a crash doesn't drop calls.

Enable transcoding in the offer:

```json
{
  "command": "offer",
  "...":     "...",
  "codec": {
    "transcode": ["PCMA", "PCMU"],
    "strip":     ["G729", "G723"]
  }
}
```

This says: regardless of what the offerer advertises, present PCMA/PCMU to the answerer; strip G.729/G.723 entirely.

## Recording

Record per-call media. Sent during or after offer/answer:

```json
{ "command": "start recording", "call-id": "8B0EBDF8-B3A1-44B6@10.0.0.1" }
{ "command": "stop recording",  "call-id": "8B0EBDF8-B3A1-44B6@10.0.0.1" }
```

Or set the `record-call: on` flag on the offer to record from the start.

Two recording modes (configured in `rtpengine.conf`):

- **PCAP** — write raw RTP packets into a pcap file. Lossless, replayable, but you need a separate tool to decode for listening.
- **WAV** — decode and mix into a stereo WAV (caller-left, callee-right). Larger storage, instantly playable.

```ini
[rtpengine]
recording-dir    = /var/spool/rtpengine
recording-method = pcap          ; or wav
recording-format = eth           ; pcap encapsulation
```

The "decoded streams" feature: in WAV mode, transcoding is forced for any non-PCM codec so the WAV captures the audio (not the encoded bitstream). Costs CPU.

## Replace SDP Address

```json
{ "replace": ["origin", "session-connection"] }
```

- `origin` — rewrite the `o=` line's source address to rtpengine's public IP. If you don't, the answerer sees the offerer's private IP in `o=` and some endpoints will reject.
- `session-connection` — rewrite the session-level `c=` line.
- `SDP-version` — bump the version field (some hard-phones get confused if it stays).
- `username` — replace the o= username.
- `session-name` — replace the s= line.
- `zero-address` — replace `c=IN IP4 0.0.0.0` (a "hold" address) with rtpengine's IP.

In Kamailio:

```cfg
rtpengine_offer("replace-origin replace-session-connection trust-address");
```

## flags

The most common bencoded flags list. Each is just a string in the `flags` list of the request dict:

- `trust-address` — believe the SDP-advertised address; don't override based on the SIP source IP.
- `symmetric` — assume the peer sends from the same port it advertises (false for symmetric NAT, but standard for endpoints behind cone NAT).
- `asymmetric` — opposite; allow source port to differ from advertised port. Required for many NAT'd endpoints.
- `strict-source` — drop packets from any IP/port other than the one negotiated. Tightens against spoofing; can break port-shifting NATs.
- `port-latching` — lock the remote endpoint to the first source IP/port seen, regardless of SDP. The deferred-decision-on-media model.
- `SDES-no-encryption` / `SDES-encrypted-srtp` — toggle SDES SRTP behavior.
- `no-rtcp-attribute` — strip the `a=rtcp:` line from the SDP.
- `RTCP-mux-offer` / `RTCP-mux-require` / `RTCP-mux-demux` / `RTCP-mux-accept` / `RTCP-mux-reject` — control multiplexing of RTP and RTCP on a single port (WebRTC requires this).
- `loop-protect` — drop packets that look like rtpengine's own forwarding loop.
- `record-call` — flag-form of the record-call option.
- `all` — apply per-flag option to all media sections.
- `force-encryption` — fail the offer if the peer can't encrypt.
- `reset` — clear and re-negotiate transport on a re-INVITE.
- `pad-crypto` — pad SDES key blocks to the codec's block size.
- `original-sendrecv` — preserve the offerer's a=sendrecv direction attribute.
- `inactive` — set the new media stream a=inactive.
- `early-media` — accept media before final answer.

In Kamailio, flags appear as a space-separated string and the module bencodes them:

```cfg
rtpengine_offer("trust-address replace-origin replace-session-connection ICE=remove DTLS=off");
```

## transport

The `transport-protocol` key forces the SDP `m=` line proto. Useful when you bridge an offer and answer with different security expectations.

| Value                    | Meaning                                      |
|--------------------------|----------------------------------------------|
| `RTP/AVP`                | plain RTP, no SRTP                           |
| `RTP/SAVP`               | SRTP (SDES)                                  |
| `RTP/AVPF`               | RTP with feedback (RFC 4585) — RTCP feedback |
| `RTP/SAVPF`              | SRTP + feedback                              |
| `UDP/TLS/RTP/SAVPF`      | DTLS-SRTP + feedback (WebRTC)                |

Default ng-protocol transport: **UDP** on port 22222. TCP and TLS variants are available for cross-host control where UDP packet loss matters.

## iptables Setup

Sample full setup for kernel-fast-path, IPv4:

```bash
TABLE=0
PORTMIN=30000
PORTMAX=40000

# load module with proper proc ownership
sudo modprobe xt_RTPENGINE \
    proc_uid=$(id -u rtpengine) \
    proc_gid=$(id -g rtpengine)

# allow inbound + outbound on the media port range and hand to RTPENGINE target
sudo iptables -I INPUT  -p udp -m udp --dport ${PORTMIN}:${PORTMAX} -j RTPENGINE --id ${TABLE}
sudo iptables -I OUTPUT -p udp -m udp --sport ${PORTMIN}:${PORTMAX} -j RTPENGINE --id ${TABLE}

# IPv6
sudo ip6tables -I INPUT  -p udp -m udp --dport ${PORTMIN}:${PORTMAX} -j RTPENGINE --id ${TABLE}
sudo ip6tables -I OUTPUT -p udp -m udp --sport ${PORTMIN}:${PORTMAX} -j RTPENGINE --id ${TABLE}
```

Persist via `iptables-save`/`netfilter-persistent`. The `--id` selects which kernel session table to consult; multiple rtpengine instances on one host use distinct table IDs.

Verify hits:

```bash
sudo iptables -L INPUT -nv  | grep RTPENGINE
sudo iptables -L OUTPUT -nv | grep RTPENGINE
# pkts bytes target     prot opt in out source  destination
# 1284 207K  RTPENGINE  udp  --  *  *   0.0.0.0/0 0.0.0.0/0  udp dpts:30000:40000 id 0
```

## Configuration File

`/etc/rtpengine/rtpengine.conf` (INI-style):

```ini
[rtpengine]
table              = 0
interface          = internal/10.0.0.5;external/203.0.113.5
listen-ng          = 127.0.0.1:22222
listen-tcp-ng      = 127.0.0.1:22223
port-min           = 30000
port-max           = 40000
recording-dir      = /var/spool/rtpengine
recording-method   = pcap
num-threads        = 8
poller-pool        = 4
log-level          = 6
log-stderr         = false
foreground         = false
pidfile            = /run/rtpengine/rtpengine.pid
no-fallback        = false
delete-delay       = 30
final-timeout      = 0
timeout            = 60
silent-timeout     = 3600

; redis
redis              = 127.0.0.1:6379/2
redis-num-threads  = 4
redis-expires      = 86400
redis-write        = 127.0.0.1:6379/2
redis-allowed-errors = 5
redis-disable-time   = 10
redis-cmd-timeout    = 5000
redis-connect-timeout = 1000

; tls
dtls-cert-file     = /etc/rtpengine/cert.pem
dtls-key-file      = /etc/rtpengine/key.pem
dtls-passive       = true

; transcoding
log-format         = json
codecs             = PCMA,PCMU,G722,opus
sip-source         = false
```

Per-deployment knobs you'll touch most:

- `interface` — multi-NIC bridging. `internal/10.0.0.5;external/203.0.113.5` defines two named interfaces; offers can pick `direction=[internal,external]` to bridge them.
- `port-min`/`port-max` — must match the iptables rule range.
- `num-threads` — scale to core count, leave one or two for the kernel.
- `recording-dir` — fast local disk; periodically prune.
- `redis` — the failover store (see "Redis Backend").
- `log-format=json` — structured logging into journald.

## Multi-Instance Setup

You can run multiple rtpengine processes on the same host. Two motivations:

1. **Workload isolation** — one instance for kernel-fast-path forwarding (cheap), another with transcoding enabled (CPU-heavy). The signaling layer routes calls to the appropriate instance based on whether transcoding is needed.
2. **Bigger transcoding pool than one process can handle** — multiple parallel daemons consuming distinct port ranges and ng ports.

Configuration: copy `/etc/rtpengine/rtpengine.conf` per instance, ensure each has:

- A distinct `listen-ng` port (22222, 22223, ...).
- A distinct `port-min`/`port-max` range — and a matching iptables rule with a distinct `--id <table>`.
- A distinct `pidfile`.
- A distinct systemd unit (`rtpengine@<name>.service`).

In Kamailio, register both instances and route by `set_rtpengine_set()` based on call attributes.

The "transcode-only-process" pattern: one rtpengine fronts all calls and forwards in-kernel; if transcoding is needed (call-leg codec mismatch), Kamailio re-routes the call's media to a dedicated transcoder process on the same box.

## Redis Backend

Persisting call state across rtpengine restarts and node failures:

```ini
[rtpengine]
redis      = 192.0.2.20:6379/2
redis-write = 192.0.2.20:6379/2
redis-num-threads = 4
```

On every offer/answer/delete and every state change, rtpengine writes the per-call entry into Redis. On restart, rtpengine reads back all keys and rebuilds its in-memory call table.

The failover pattern:

```
   ┌──────────┐         ┌──────────┐
   │ rtpe-A   │◄───────►│  redis   │◄───────►│ rtpe-B │
   │ (active) │  state   └──────────┘  state  │ (warm) │
   └──────────┘                                └────────┘
       ▲                                          ▲
       │ active VIP                               │ promoted on failover
       └──────────────── pacemaker ───────────────┘
```

When rtpe-A crashes, pacemaker brings up the VIP on rtpe-B. rtpe-B already has the call state from Redis. ICE/DTLS contexts may need re-negotiation depending on the call phase, but most established calls survive.

Latency budget: every state-mutating event waits for Redis ack on the write path. Modest (sub-ms) on a co-located Redis; **don't** put Redis behind a high-latency link.

## ng-protocol Helper Tools

- **`rtpengine-ng-client`** — a Perl CLI that bencodes a request, sends it on UDP, receives the response. Useful for manual `query`, `list`, and ad-hoc test offers.

```bash
rtpengine-ng-client --proxy=127.0.0.1:22222 \
    --command=list --limit=20
```

- **`ngcp-rtpengine-client`** — Sipwise's variant.
- **`ngcp-utils`** — Sipwise toolbox: ngcp-rtpengine-statistics, ngcp-rtpengine-recording-prune, etc.
- **rtpengine-recording**: a Perl daemon that watches `recording-dir` and converts pcaps to mp3/wav.
- **homer-rtpengine-stats**: pushes stats to Homer/HEP for centralized monitoring.

## Kamailio rtpengine_module

Load and configure:

```cfg
loadmodule "rtpengine.so"
modparam("rtpengine", "rtpengine_sock",   "udp:127.0.0.1:22222")
modparam("rtpengine", "rtpengine_disable_tout",   60)
modparam("rtpengine", "rtpengine_tout_ms",        1000)
modparam("rtpengine", "queried_nodes_limit",      1)
modparam("rtpengine", "extra_id_pv",              "$avp(extra_id)")
modparam("rtpengine", "setid_avp",                "$avp(setid)")
```

Per-route invocation:

```cfg
route[RELAY] {
    if (is_method("INVITE")) {
        if (has_body("application/sdp")) {
            rtpengine_offer("replace-origin replace-session-connection ICE=remove DTLS=off");
        }
        t_on_reply("REPLY_HANDLER");
    }
    if (is_method("ACK") && has_body("application/sdp")) {
        rtpengine_answer();
    }
    if (is_method("BYE")) {
        rtpengine_delete();
    }
    t_relay();
}

onreply_route[REPLY_HANDLER] {
    if (status =~ "(183|2[0-9][0-9])" && has_body("application/sdp")) {
        rtpengine_answer("replace-origin replace-session-connection");
    }
}
```

Module functions:

- `rtpengine_offer([flags])` — sends `offer`. Pulls SDP from the current SIP message, replaces it with rewritten SDP.
- `rtpengine_answer([flags])` — sends `answer`. Pulls SDP from the current message, replaces it.
- `rtpengine_delete([flags])` — sends `delete`.
- `rtpengine_manage([flags])` — heuristic auto-pick based on method/direction.
- `rtpengine_query()` — emits a `query` and exposes results in `$rtpstat` PV.
- `start_recording([flags])` / `stop_recording([flags])` — record control.
- `set_rtpengine_set(N)` — pick a particular rtpengine set for this call (multi-instance).

The flags parameter accepts a space-separated list of flag tokens; the module bencodes them into the wire request.

## OpenSIPS rtpproxy_module Compatibility

OpenSIPS historically used the older `rtpproxy_module` speaking the line-protocol of the original `rtpproxy` from Maxim Sobolev. This protocol is partially compatible with rtpengine — rtpengine listens on `listen-cli` (port 22221 by default) for legacy line-protocol commands. So OpenSIPS scripts that use `rtpproxy_offer()` / `rtpproxy_answer()` work against rtpengine, but only the original feature set (no flags, no transcoding, no recording).

For full functionality OpenSIPS has its own **rtpengine module** since OpenSIPS 2.x. Configuration is similar to Kamailio:

```cfg
loadmodule "rtpengine.so"
modparam("rtpengine", "rtpengine_sock", "udp:127.0.0.1:22222")
```

## FreeSWITCH Integration

FreeSWITCH has a built-in media server: it terminates RTP, transcodes, plays prompts, etc. So rtpengine isn't usually needed in front of FreeSWITCH for its own calls.

The exception is the WebRTC-edge pattern: FreeSWITCH as registrar/IVR/voicemail backend, with rtpengine bridging WebRTC clients to FreeSWITCH's plain RTP. FreeSWITCH does support DTLS-SRTP natively (via `mod_sofia` and `mod_verto`) but offloading to rtpengine has two advantages:

- rtpengine handles thousands of WebRTC sessions per box; FreeSWITCH's per-call overhead is higher.
- Concentrate all DTLS-SRTP termination in one place for auditing/recording.

## Asterisk Integration

Asterisk supports DTLS-SRTP and ICE in `res_rtp_asterisk` natively. rtpengine is sometimes deployed in front of Asterisk for:

- **IPv4 ↔ IPv6 bridging** — Asterisk's IPv6 support is solid, but bridging an IPv4 SIP trunk to an IPv6 Asterisk pjsip endpoint is cleaner via rtpengine.
- **NAT traversal under heavy load** — rtpengine's port-latching is more robust than Asterisk's.
- **WebRTC DTLS-SRTP at scale** — moving the DTLS handshake out of Asterisk frees Asterisk for dialplan work.
- **Centralized recording** — rtpengine records media without Asterisk's MixMonitor overhead.

When using rtpengine with Asterisk, configure Asterisk to advertise its IP and trust the source port from rtpengine, with NAT/STUN disabled inside Asterisk.

## WebRTC Pipeline

The canonical SIP-to-WebRTC pipeline:

```
Browser (SIP.js / JsSIP) ─── WSS ───► Kamailio ──── ng ────► rtpengine
                            (signaling)              (control)
                                │                        │
                                │                        │ DTLS-SRTP terminated
                                │                        │ ICE-Lite
                                │                        │ codec transcoded if needed
                                │                        ▼
                                │                  plain RTP
                                │                        │
                                ▼                        ▼
                       SIP-over-UDP/TCP ────► FreeSWITCH / Asterisk / SIP trunk
```

Kamailio:

- Accepts WSS from the browser (loadmodule "websocket.so" + tls.so).
- Routes signaling to FreeSWITCH/Asterisk over UDP/TCP.
- Calls `rtpengine_offer("ICE=force RTCP-mux-offer DTLS=passive trust-address replace-origin replace-session-connection transcode-PCMU transcode-PCMA strip-extmap strip-msid")` on outgoing INVITE.

rtpengine:

- Speaks ICE-Lite + DTLS-SRTP toward the browser.
- Emits plain RTP (PCMU/PCMA) toward the backend.
- Transcodes Opus ↔ PCMU/PCMA where needed.

This is the most common production WebRTC-to-PSTN topology and rtpengine is essentially the only widely-deployed open-source piece that does it well.

## IPv4 ↔ IPv6 Bridge

Configure two interfaces in `rtpengine.conf`:

```ini
[rtpengine]
interface = pub4/203.0.113.5;pub6/2001:db8:42::5
```

Or short form `--interface=external/203.0.113.5!2001:db8:42::5` to bind one logical interface to both address families.

In the offer, request the bridge:

```json
{
  "command":   "offer",
  "...":       "...",
  "direction": ["pub4", "pub6"]
}
```

rtpengine binds one socket per side from the right address family and bridges. The kernel module supports IPv4↔IPv6 forwarding natively.

## Performance Tuning

- **Kernel module is the big win.** Without it you're shoving every packet through userspace via recvmsg/sendmsg. With it, only the first few packets traverse userspace; the rest forward in the kernel.
- **Userspace falls back when transcoding/SRTP/recording is needed.** Plan CPU around the transcoding workload, not the relay workload.
- **CPU pinning.** `taskset` rtpengine threads to dedicated cores; isolate them from the kernel softirq cores via `isolcpus` or systemd CPUAffinity.
- **NUMA-aware allocation.** On a multi-socket box, pin all rtpengine threads + the NIC's interrupt handlers to the same NUMA node to avoid cross-socket memory traffic.
- **Tune the NIC.** Increase rx ring buffers (`ethtool -G eth0 rx 4096`), enable RSS, ensure RPS distributes interrupts.
- **Disable `conntrack`** for the media port range — it doubles per-packet overhead and burns connection-table entries.

```bash
iptables -t raw -I PREROUTING -p udp --dport 30000:40000 -j NOTRACK
iptables -t raw -I OUTPUT     -p udp --sport 30000:40000 -j NOTRACK
```

- **Increase the per-process file descriptor limit** to twice the expected concurrent socket count (one per direction per call, plus headroom).
- **Watch port-range exhaustion.** If `port-min..port-max` is N ports and each call uses 2 RTP + 2 RTCP sockets in non-rtcp-mux mode, you cap at N/4 calls.

## Stats

The `query` command per-call. The `statistics` command daemon-wide:

```json
{ "command": "statistics" }
```

Response includes:

- `currentstatistics` — calls active, sessions active, packets/sec, bytes/sec, transcoded sessions.
- `totalstatistics` — lifetime totals.
- `controlstatistics` — ng-protocol command counts.
- `mosstatistics` — MOS distribution if computed.

Integration with **Homer/HEP** (Homer is the open-source SIP capture/insight platform): rtpengine emits HEP packets carrying RTCP-XR-style stats, plus call-event metadata. Homer ingests, indexes by call-id, and renders quality timelines.

```ini
[rtpengine]
homer = 192.0.2.30:9060
homer-protocol = udp
homer-id = 2001
```

There's also a Prometheus exporter (`ngcp-rtpengine-prometheus-exporter`) that scrapes `statistics` and reformats as Prometheus metrics.

## Common Errors

Verbatim error strings (in rtpengine.log or echoed back to Kamailio) and the canonical fix:

- **"ng failed"** (Kamailio side) — Kamailio's rtpengine module timed out waiting for a response. Check rtpengine is running and listening on the configured port: `ss -lun | grep 22222`. Check the firewall isn't blocking 127.0.0.1:22222 (yes, this happens with overzealous host firewalls).
- **"Mismatched offer-answer"** — the answer arrived with a from-tag/to-tag combination that doesn't match a known offer. Usually caused by a dialog mismatch upstream — Kamailio is sending the answer with a different from-tag than the offer. Check the SIP trace; ensure your dialog module isn't rewriting tags.
- **"no codec match"** — both legs advertised disjoint codec sets and you didn't ask rtpengine to transcode. Fix: add `transcode-PCMU transcode-PCMA` (or whichever) to flags.
- **"ICE connectivity check failed"** — no candidate pair completed STUN connectivity. Common when rtpengine's advertised IP is unreachable from the browser, or when ICE-Lite is set but the browser is in `trickle ICE` mode and fails to send a candidate. Check rtpengine logs at log-level 7; check `a=candidate:` lines in the SDP.
- **"DTLS handshake failed"** — usually fingerprint mismatch (`a=fingerprint:` in SDP doesn't match the cert presented over the wire), expired cert, or peer using TLSv1.0 against an OpenSSL configured for TLSv1.2+. Check `dtls-cert-file`/`dtls-key-file` paths.
- **"SRTP key derivation failed"** — SDES `a=crypto:` keys malformed or wrong cipher requested. Check the offer SDP carries valid `AES_CM_128_HMAC_SHA1_80` or `AES_CM_128_HMAC_SHA1_32` lines.
- **"rtpengine: kernel module not loaded"** (warning at startup) — daemon couldn't open `/proc/rtpengine/<table>/control`. Run `modprobe xt_RTPENGINE` and confirm `/proc/rtpengine/0/info` exists. Match the `table` in rtpengine.conf to the iptables `--id`.
- **"address already in use"** — `port-min..port-max` overlaps another listener. Pick a different range.
- **"can't allocate port pair"** — port range exhausted. Increase `port-max - port-min` or shorten `delete-delay` so freed ports return faster.
- **"redis disconnected"** — Redis is unreachable. Calls keep working in-memory but state isn't persisted; restart loses state.
- **"transcoding context creation failed"** — ffmpeg couldn't open a codec. Usually because rtpengine was built without that codec, or the codec library is missing.

## Common Gotchas

Twelve+ broken→fixed pairs that catch people repeatedly:

- **Kernel module loaded but iptables rule not added → no kernel-fast-path.** rtpengine logs nothing wrong, calls work, but every packet hits userspace and CPU usage is 10× expected. Fix: add the `-j RTPENGINE --id <table>` rule for both INPUT and OUTPUT chains, then verify `iptables -L -nv | grep RTPENGINE` shows packet counts incrementing.
- **Port range too narrow under load → connection failures.** Symptom: random call setup failures during peak. Fix: widen `port-min..port-max` to ≥ 8 × peak concurrent calls, and ensure the firewall ACLs match.
- **Forgetting `replace=session-connection` → SDP advertises private IP.** rtpengine rewrites `m=` ports but not `c=` lines unless asked. The callee tries to send media to the offerer's private 10.0.0.x. Fix: always include `replace-session-connection` (and `replace-origin`) for NAT'd offers.
- **`flags=symmetric` on asymmetric NAT → media stuck.** Symmetric NAT changes the source port between the SIP/SDP advertisement and the actual RTP source. With `symmetric` set, rtpengine drops the actual RTP because it doesn't match the advertised port. Fix: use `asymmetric` (or `port-latching`).
- **`delete` called without `from-tag` → leak.** rtpengine deletes only the matching tag-pair; without `from-tag` it may delete _all_ sessions for the call-id, or refuse and leak the unrelated direction. Fix: always pass `from-tag` and (where known) `to-tag`.
- **SDES + `force-encryption=accept` on plain-RTP backend → handshake fails.** The backend doesn't speak SRTP; rtpengine refuses to fall back. Fix: configure two separate offers — encrypted toward the WebRTC client, plain toward the backend, with rtpengine bridging.
- **Recording enabled but disk fills → service degrades.** Once `recording-dir` is full, rtpengine still accepts calls but recordings silently drop. Fix: aggressive cron-based pruning, or use `recording-dir` on a dedicated ZFS dataset with quota.
- **Redis backend down → state-failover broken.** If Redis is down rtpengine keeps relaying calls but writes nothing; failover to the warm node loses state. Fix: monitor Redis health; alert on `redis disconnected` log lines.
- **Transcoding capacity exhausted → call quality degrades.** When the transcoding pool is saturated, packets queue up in worker threads and you see jitter/loss spikes. Fix: cap concurrent transcoded calls (Kamailio admission control) or scale out to more rtpengine instances.
- **DTLS-SRTP cert renewed but rtpengine not reloaded.** The new cert is on disk but rtpengine still serves the old one because it was loaded at startup. Calls succeed (SRTP works either way) but fingerprint validation against signaling fails for new clients. Fix: reload rtpengine on cert renewal; consider `systemctl reload-or-restart rtpengine` in the renewal hook.
- **ICE-Lite mode but client expects full-ICE.** Symptom: ICE check fails with WebRTC clients that don't fall back to ICE-Lite gracefully. Fix: make sure rtpengine's candidate is reachable from the client (public IP, no firewall blocking the media port range).
- **Multiple rtpengine instances + Redis: thundering herd on failover.** When the active rtpengine dies, all standby instances pull state from Redis at once and retry handshakes simultaneously. Fix: stagger startup with `redis-num-threads` low and use jittered retry delays.
- **`trust-address` left off in cloud → wrong IP advertised.** Without `trust-address`, rtpengine uses the SIP source IP (often a load-balancer's private IP) as the media destination. Fix: always include `trust-address` when running behind a load balancer.
- **RTCP-mux mismatch.** WebRTC requires RTCP-mux; legacy SIP doesn't. Without `RTCP-mux-demux` on the legacy side, rtpengine can't bridge. Fix: include `RTCP-mux-demux` toward the legacy leg.
- **`final-timeout=0` plus stuck call.** Default `final-timeout=0` means calls live forever even after media stops. Fix: set `final-timeout` to a sane value (e.g. 7200) so abandoned calls eventually GC.

## Diagnostic Tools

- **`/var/log/rtpengine.log`** — primary log. Bump `log-level` to 7 for debug-level (very noisy; only short windows).
- **`ngcp-rtpengine-ctl statistics`** / `query <call-id>` — runtime introspection.
- **tcpdump on the RTP port range:**

```bash
sudo tcpdump -i any -nn -vvv 'udp portrange 30000-40000' -w /tmp/rtp.pcap
```

Then open in Wireshark; use Telephony → RTP Streams.

- **`/proc/rtpengine/0/info`** — kernel-side counters. If "Number of forwarded packets" stays at 0 while userspace logs show calls, the iptables rule is missing.
- **`/proc/rtpengine/0/list`** — active kernel sessions. Each line is one stream tuple.
- **`iptables -L -nv | grep RTPENGINE`** — confirm the target is hitting packets.
- **`ss -lup | grep rtpengine`** — confirm the daemon is listening on `port-min..port-max`.
- **`ngcp-rtpengine-recording-prune`** — cleanup utility for old recordings.
- **Homer** — SIP/RTCP capture, indexed by call-id; the production-grade troubleshooting tool for VoIP.

## Sample Cookbook

### Kamailio + rtpengine for WebRTC ↔ SIP

`/etc/rtpengine/rtpengine.conf`:

```ini
[rtpengine]
interface     = pub/203.0.113.5
listen-ng     = 127.0.0.1:22222
port-min      = 30000
port-max      = 40000
recording-dir = /var/spool/rtpengine
num-threads   = 8
dtls-cert-file = /etc/rtpengine/cert.pem
dtls-key-file  = /etc/rtpengine/key.pem
log-level     = 6
table         = 0
```

iptables:

```bash
sudo iptables -I INPUT  -p udp --dport 30000:40000 -j RTPENGINE --id 0
sudo iptables -I OUTPUT -p udp --sport 30000:40000 -j RTPENGINE --id 0
```

Kamailio `kamailio.cfg` (relevant snippet):

```cfg
loadmodule "rtpengine.so"
modparam("rtpengine", "rtpengine_sock", "udp:127.0.0.1:22222")

route[WEBRTC] {
    if (proto == WSS && is_method("INVITE")) {
        rtpengine_offer("RTCP-mux-demux ICE=force DTLS=passive trust-address replace-origin replace-session-connection transcode-PCMU transcode-PCMA");
        t_on_reply("WEBRTC_REPLY");
    }
}

onreply_route[WEBRTC_REPLY] {
    if (status =~ "(183|2[0-9][0-9])" && has_body("application/sdp")) {
        rtpengine_answer("RTCP-mux-demux trust-address replace-origin replace-session-connection");
    }
}

route[BYE] {
    if (is_method("BYE")) {
        rtpengine_delete();
    }
}
```

### rtpengine for SIP-trunk transcoding

Two SIP carriers, one offers G.729 only, the other PCMU only. rtpengine in the middle transcodes:

```cfg
rtpengine_offer("transcode-PCMU strip-G729 trust-address replace-origin");
rtpengine_answer("transcode-PCMU strip-G729 trust-address replace-origin");
```

### rtpengine for recording

Record every call destined to a particular trunk:

```cfg
if ($rd =~ "trunk-record\.example\.com") {
    rtpengine_offer("record-call=yes replace-origin replace-session-connection");
} else {
    rtpengine_offer("replace-origin replace-session-connection");
}
```

The recordings land in `recording-dir`. A cron job converts pcaps to WAV nightly via `tshark` + `sox`.

### rtpengine for IPv4 ↔ IPv6 bridging

```ini
[rtpengine]
interface = v4/203.0.113.5;v6/2001:db8::5
```

```cfg
rtpengine_offer("direction=v4=v6 replace-origin replace-session-connection");
```

## Idioms

- **"Kernel module + iptables rule for every deploy."** Without the iptables rule the kernel module is dead weight. Every install playbook must drop both.
- **"Transcoding userspace-only — size CPUs accordingly."** Plan core count around peak transcoded calls, not peak total calls.
- **"Use Redis for HA failover."** Without Redis you cannot survive a daemon restart without dropping every active call. With Redis, properly tested, you can.
- **"Monitor port-range exhaustion."** Alert at 70% utilization of `(port-max - port-min)/4`. Calls don't fail until exhaustion, then they all fail at once.
- **"Always `trust-address` behind a load balancer."** Without it, rtpengine learns the wrong IP and media flows nowhere.
- **"Always `replace-origin` and `replace-session-connection`."** It's cheap, it's idempotent, and it prevents the canonical "media one-way" bug.
- **"Match `port-min..port-max` to the iptables rule."** Drift between the daemon config and the firewall rule is the single most common operational mistake.
- **"Run rtpengine on dedicated cores."** Isolate from kernel softirqs, pin threads, NUMA-bind to the NIC's node.
- **"Test RTCP-mux explicitly per leg."** WebRTC requires it, legacy SIP often doesn't, and the bridge needs `RTCP-mux-demux`.
- **"Renew DTLS certs and reload rtpengine in the same hook."** Otherwise the new cert sits unused and silently breaks fingerprint validation eventually.

## Worked ng-Protocol Examples

### Bencoded offer command (full request)

```text
d10:call-idi  20:abc123-call-id-99...e
8:from-tag20:tag-from-A...
4:flagsl4:SDES7:trust-address9:replace=eOf 8:replace-origin18:replace-session-connection7:rtcp-muxe
3:sdpL...full SDP body as bencoded string...
7:commande5:offere
```

Decoded:

```text
{
  "command": "offer",
  "call-id": "abc123-call-id-99",
  "from-tag": "tag-from-A",
  "flags": ["SDES", "trust-address", "replace-origin", "replace-session-connection", "rtcp-mux"],
  "sdp": "v=0\r\no=alice 1 1 IN IP4 192.0.2.1\r\ns=-\r\nc=IN IP4 192.0.2.1\r\n..."
}
```

### Answer command

```text
{
  "command": "answer",
  "call-id": "abc123-call-id-99",
  "from-tag": "tag-from-A",
  "to-tag": "tag-from-B",
  "flags": ["SDES", "rtcp-mux"],
  "sdp": "v=0\r\no=bob 2 2 IN IP4 198.51.100.5\r\n..."
}
```

### Delete command

```text
{
  "command": "delete",
  "call-id": "abc123-call-id-99",
  "from-tag": "tag-from-A",
  "flags": []
}

# Response:
{
  "result": "ok",
  "totals": {
    "RTP": {"input": {"packets": 1234, "bytes": 197440}, "output": {"packets": 1234, "bytes": 197440}},
    "RTCP": {"input": {"packets": 28, "bytes": 1456}, "output": {"packets": 28, "bytes": 1456}}
  }
}
```

### Query command

```text
{ "command": "query", "call-id": "abc123-call-id-99" }

# Response:
{
  "result": "ok",
  "created": 1714048823,
  "last signal": 1714049124,
  "tags": {
    "tag-from-A": {
      "created": 1714048823,
      "media": [
        {
          "index": 1,
          "type": "audio",
          "protocol": "RTP/AVP",
          "streams": [
            {
              "local port": 30002,
              "endpoint": {"family": "IP4", "address": "192.0.2.1", "port": 50000},
              "stats": {"packets": 1234, "bytes": 197440, "errors": 0}
            }
          ]
        }
      ]
    },
    "tag-from-B": { ... }
  }
}
```

### Start recording

```text
{
  "command": "start recording",
  "call-id": "abc123-call-id-99",
  "from-tag": "tag-from-A",
  "output-destination": "wav",
  "output-name": "/var/spool/rtpengine/abc123.wav"
}
```

### List active calls

```text
{ "command": "list", "limit": 10 }

# Response:
{
  "result": "ok",
  "calls": ["abc123-call-id-99", "def456-call-id-87", ...]
}
```

## iptables Rule Recipes

### Standard single-instance

```bash
# Insert at top of FORWARD; rtpengine table 0
iptables -I FORWARD -p udp --dport 30000:40000 -j RTPENGINE --id 0
ip6tables -I FORWARD -p udp --dport 30000:40000 -j RTPENGINE --id 0
```

### Multi-instance (separate kernel tables)

```bash
# Instance 1 on port range 30000-40000, table 0
iptables -I FORWARD -p udp --dport 30000:40000 -j RTPENGINE --id 0

# Instance 2 on port range 40001-50000, table 1
iptables -I FORWARD -p udp --dport 40001:50000 -j RTPENGINE --id 1
```

Each rtpengine daemon must specify `table = 0` or `table = 1` in its config.

### Mark-based selection (advanced multi-tenant)

```bash
# Match calls coming from tenant1 (marked upstream by Kamailio with -j MARK --set-mark 0x1)
iptables -I FORWARD -p udp -m mark --mark 0x1 -j RTPENGINE --id 0

# Tenant2
iptables -I FORWARD -p udp -m mark --mark 0x2 -j RTPENGINE --id 1
```

### IPv6 + dual-stack

```bash
iptables  -I FORWARD -p udp --dport 30000:40000 -j RTPENGINE --id 0
ip6tables -I FORWARD -p udp --dport 30000:40000 -j RTPENGINE --id 0
```

The kernel module bridges across address families when configured with `--interface=public/PUBLIC_V4!PUBLIC_V6`.

## DTLS-SRTP Cert Configuration

```ini
# /etc/rtpengine/rtpengine.conf
[rtpengine]
listen-ng = 127.0.0.1:22222
interface = public/198.51.100.10
recording-dir = /var/spool/rtpengine
table = 0
foreground = false

# DTLS-SRTP for WebRTC termination
dtls-cert-cipher = ec
dtls-mtu = 1200
dtls-signature = sha-256
```

Generate the DTLS cert (rtpengine auto-generates per-startup if missing, but pinning a long-lived one across restarts is useful for fingerprint pinning by clients):

```bash
openssl req -x509 -newkey ec:<(openssl ecparam -name prime256v1) \
  -keyout /etc/rtpengine/dtls.key \
  -out    /etc/rtpengine/dtls.crt \
  -days 365 -nodes -subj '/CN=rtpengine.example.com'
```

Reference in config:

```ini
dtls-cert-file = /etc/rtpengine/dtls.crt
dtls-key-file  = /etc/rtpengine/dtls.key
```

## Transcoding Combinations

```text
# G.711 PCMU (caller) ↔ Opus (callee)
flags: ["transcode-PCMU", "transcode-opus"]
# rtpengine accepts whichever is offered, emits both, transcodes per leg

# G.711 PCMA ↔ G.722 (wideband upgrade)
flags: ["transcode-PCMA", "transcode-G722"]

# G.729 ↔ Opus (full transcoding through G.711 bridge in some impls)
flags: ["transcode-G729", "transcode-opus"]

# AMR-WB ↔ Opus (mobile interconnect)
flags: ["transcode-AMR-WB", "transcode-opus"]
codec-options: { "codec-options-AMR-WB": "mode-set=8", "codec-options-opus": "useinbandfec=1" }
```

CPU-cost rule of thumb:
- G.711 ↔ Opus: ~3% CPU per call on a modern x86 core
- G.729 ↔ anything: ~5% per call (G.729 itself is the expensive end)
- AMR-WB ↔ anything: ~6% per call
- Plain RTP relay (no transcoding): kernel-fastpath, ~0.05% per call

Size your transcoding pool: 100 concurrent G.729↔Opus calls ≈ 5 cores at 100% utilization. Always reserve 50% headroom.

## Kamailio rtpengine_module Integration

```text
# kamailio.cfg
loadmodule "rtpengine.so"

modparam("rtpengine", "rtpengine_sock", "udp:127.0.0.1:22222")
modparam("rtpengine", "rtpengine_tout_ms", 1000)
modparam("rtpengine", "extra_id_pv", "$avp(extra_id)")

# Inside the route block handling INVITE
route[NAT_OFFER] {
    if (is_method("INVITE")) {
        rtpengine_offer("trust-address replace-origin replace-session-connection ICE=force-relay RTP/SAVPF SDES");
    }
}

# In branch_route or onreply_route on 200 OK
onreply_route[REPLY_FROM_LEG] {
    if (status =~ "(183)|(200)") {
        rtpengine_answer("trust-address replace-origin replace-session-connection RTP/AVP");
    }
}

# In failure_route or BYE handling
route[NAT_DELETE] {
    if (is_method("BYE")) {
        rtpengine_delete();
    }
}
```

The flag set per call is the differential: rtpengine offer can specify "I want WebRTC fingerprint here," answer can specify "but the answer leg is plain SIP/AVP." rtpengine bridges both transparently.

### Kamailio + rtpengine + multi-tenant

```text
# Kamailio sets rtpengine instance via marker
modparam("rtpengine", "setid_avp", "$avp(rtpengine_set)")

route[ROUTE_BY_TENANT] {
    if ($au =~ "^.*@tenant1\.example$") {
        $avp(rtpengine_set) = 0;
    } else if ($au =~ "^.*@tenant2\.example$") {
        $avp(rtpengine_set) = 1;
    }
    rtpengine_manage();
}
```

Two rtpengine instances on different ports + different kernel tables; Kamailio routes calls between them.

## Flag/Option Combinations

| Goal | flags |
|---|---|
| Plain RTP relay | `[]` (defaults; just relay) |
| WebRTC ↔ SIP bridge | `["SDES", "DTLS=passive", "ICE=force-relay", "RTP/AVP", "RTP/SAVPF", "rtcp-mux", "rtcp-mux-demux"]` |
| Force SDES (server-side termination) | `["SDES", "RTP/SAVP"]` |
| Force DTLS-SRTP | `["DTLS=passive", "RTP/SAVPF"]` |
| Recording on | `["start-recording"]` (or send separate "start recording" command) |
| Replace SDP IPs | `["replace-origin", "replace-session-connection"]` |
| Trust source address (behind LB) | `["trust-address"]` |
| Symmetric RTP (one-way LBs) | `["symmetric"]` |
| Strict source port enforcement | `["strict-source"]` |
| Asymmetric (different ports for read/write) | `["asymmetric"]` |
| Disable RTCP entirely | `["no-rtcp"]` |
| Add ICE candidates | `["ICE=force"]` (rtpengine offers as if it had been negotiated) |
| Drop ICE candidates | `["ICE=remove"]` (strip ICE attributes from SDP) |

## See Also

- kamailio
- opensips
- drachtio
- asterisk
- freeswitch
- sip-protocol
- rtp-sdp

## References

- github.com/sipwise/rtpengine — source, README, ng-protocol documentation (`docs/ng_client.md`, `docs/control-protocol.md`).
- github.com/sipwise/rtpengine/blob/master/README.md — install, config, full flag/option reference.
- github.com/sipwise/rtpengine/blob/master/daemon/control_ng.c — the wire-format parser.
- github.com/sipwise/rtpengine/blob/master/kernel-module/xt_RTPENGINE.c — the kernel module source.
- sipwise.com/doc/ — Sipwise C5 documentation, integration guides.
- kamailio.org/docs/modules/devel/modules/rtpengine.html — Kamailio rtpengine module reference.
- opensips.org/Documentation/Interface-CoreFunctions-3-0 — OpenSIPS rtpengine module.
- RFC 3711 — SRTP.
- RFC 5763 / RFC 5764 — DTLS-SRTP.
- RFC 8445 — ICE.
- RFC 5761 — RTP/RTCP multiplexing.
- RFC 7635 — STUN extensions for OAuth (optional).
- BEP 3 (BitTorrent Enhancement Proposal 3) — bencode specification.
