# softphones

SIP softphone clients — Linphone, Bria, Zoiper, MicroSIP, Jami, Jitsi, Telephone, Acrobits, 3CX, pjsua, baresip — configuration, codecs, push notifications, ZRTP, audio quirks.

## Setup

A softphone is a SIP User Agent (UA) running as software on a general-purpose computing device — Linux/Mac/Windows desktop, iOS/Android phone, or a web browser. Functionally identical to a desk IP phone (Yealink, Polycom, Cisco, Grandstream): registers to a SIP registrar, places/receives calls, negotiates codecs via SDP, carries RTP media. Differs only in form-factor: no dedicated handset hardware, no PoE, no LCD, no fixed audio path.

**Deployment model differences from physical IP phones:**

- **No provisioning protocols.** A desk phone fetches its config from a TFTP/HTTP/HTTPS provisioning URL on boot — `cfg<MAC>.xml`, certificate enrollment, dial-plan push. A softphone has no MAC-anchored identity; it has a user account stored in app preferences. Mass deployment requires either (a) the vendor's central management (Bria Stretto, 3CX Welcome Email, Cloudsoftphone admin), or (b) MDM-pushed config profiles, or (c) manual user entry.
- **No PoE.** Powered by the host device. Battery life on mobile becomes a real concern — registration keep-alives drain the radio.
- **No fixed audio hardware.** A desk phone has a fixed handset, a fixed speakerphone, a fixed mic. A softphone is at the mercy of whatever the OS exposes — built-in mic, USB headset, Bluetooth, virtual audio cables. Audio quality varies wildly.
- **No vendor-tested DSP chain.** Desk phones ship with calibrated AEC/AGC/noise suppression for their hardware. Softphones rely on the host OS audio stack and software DSP, which can be surprisingly bad — feedback loops on laptops with built-in mics next to speakers, AEC tail-length wrong for the room.
- **OS-mediated everything.** The OS owns the network, the audio device, and (on mobile) the wake/sleep state of the radio. Push notifications, background-execution policies, and audio-routing rules all become softphone problems.

**Codec/network reality of softphones:**

- More capable in raw codec terms — typical softphones support Opus (wideband + super-wideband + fullband), G.722 (wideband), Opus FEC, video codecs (H.264, VP8, VP9, sometimes H.265). Desk phones typically cap at G.722 wideband.
- But at the mercy of the OS audio stack — sample rate switching, exclusive vs shared device modes, Bluetooth profile negotiation (HFP vs A2DP), USB audio class compliance.
- Network adaptation usually better — softphone CPU has cycles for adaptive jitter buffer, PLC, FEC, NetEQ-like algorithms. Desk-phone DSPs are often fixed.
- But also subject to OS-network policies — VPN split-tunneling, firewall rules, Wi-Fi power-save mode dropping packets, mobile carrier NAT timeouts.

## Decision Matrix

| Softphone | License | Platforms | SIP/SDP | Codecs | DTMF | TLS | SRTP | ICE/STUN/TURN | ZRTP | Push (mobile) | Recording | Video Codecs | Headless CLI | BroadSoft/3CX | Teams Compat |
|-----------|---------|-----------|---------|--------|------|-----|------|---------------|------|---------------|-----------|--------------|--------------|---------------|--------------|
| Linphone | GPLv3 (FOSS) / commercial SDK | Linux/Mac/Win/iOS/Android/Web | Full + extensions | G.711, G.722, G.729 (paid), Opus, iLBC, AMR, AMR-WB, SILK, Speex | RFC 4733 / SIP-INFO / in-band | Yes | SDES + DTLS-SRTP | Yes (ICE/STUN/TURN) | Yes | Yes (own server) | Yes | H.264, H.265, VP8, VP9, AV1 | Yes (linphonec/linphonecsh) | Partial | No |
| Bria (Counterpath) | Commercial subscription | Win/Mac/iOS/Android | Full + extensions | G.711, G.722, G.729, Opus, iLBC, SILK, AMR-WB | RFC 4733 / SIP-INFO / in-band | Yes | SDES | Yes | No | Yes (Counterpath PNS) | Yes (Pro/Ent) | H.264, VP8 | No | Yes (BroadSoft XSI) | Limited (via SIP gateway) |
| Zoiper 5 | Freemium / Pro / Premium | Win/Mac/Linux/iOS/Android | Full | Free: G.711; Pro: + G.722, iLBC, Opus, Speex; Premium: + G.729 (license), AMR-WB | RFC 4733 / SIP-INFO / in-band | Pro+ | SDES (Pro+) | Yes | ZRTP add-on | Yes (Zoiper PNS) | Pro+ | H.263, H.264, VP8 (Pro+) | Pro+ (zoiper.exe) | Yes | No |
| MicroSIP | GPL (FOSS) | Windows only | Full (via PJSIP) | G.711, G.722, G.729 (license), Opus, iLBC, AMR, Speex | RFC 4733 / SIP-INFO / in-band | Yes | SDES + DTLS-SRTP | Yes | Yes | N/A (desktop) | Yes | H.264, H.263 | No (GUI only) | Partial | No |
| Jami | GPLv3 (FOSS) | Linux/Mac/Win/iOS/Android | SIP + Jami P2P account | Opus, G.711, G.722, Speex | RFC 4733 / SIP-INFO | Yes | SDES + DTLS-SRTP | Yes | Yes (mandatory for Jami acct) | Yes (Jami DHT) | Yes | H.264, H.265, VP8, VP9 | Yes (jami-dbus, jamictrl) | No | No |
| Jitsi Desktop | LGPL/Apache (FOSS) | Linux/Mac/Win | Full | G.711, G.722, Opus, iLBC, SILK, Speex | RFC 4733 / SIP-INFO | Yes | SDES + DTLS-SRTP | Yes | Yes | N/A | Yes | H.264, H.263, VP8 | No | Partial | No |
| Telephone | MIT (FOSS) | macOS / iOS | Full (via PJSIP) | G.711, G.722, Opus, iLBC, Speex, GSM | RFC 4733 / SIP-INFO | Yes | SDES | Yes | No | Limited (iOS) | No | None (audio only) | No | No | No |
| Acrobits Softphone | Commercial | iOS/Android | Full | G.711, G.722, Opus, iLBC, G.729 (paid) | RFC 4733 / SIP-INFO | Yes | SDES + ZRTP | Yes | Yes | Yes (Acrobits PNS — APNS/FCM) | Yes | H.264 | No | Yes (Cloudsoftphone) | No |
| 3CX Softphone | Commercial (free with 3CX) | Win/Mac/iOS/Android/Web | Full + 3CX extensions | G.711, G.722, Opus, iLBC | RFC 4733 / SIP-INFO | Yes | SDES | Yes | No | Yes (3CX PNS) | Yes | H.264 | No | 3CX-only by design | Direct (3CX SBC) |
| pjsua | GPL+commercial | Linux/Mac/Win/iOS/Android | Full (reference) | G.711, G.722, G.729, Opus, iLBC, Speex, SILK, AMR | RFC 4733 / SIP-INFO / in-band | Yes | SDES + DTLS-SRTP | Yes | Yes | Yes (in SDK) | Yes | H.264, H.263, VP8 | Yes (pjsua CLI) | No | No |
| baresip | BSD (FOSS) | Linux/Mac/Win/iOS/Android/embedded | Full (modular) | G.711, G.722, Opus, AMR, iLBC, Speex, codec2 | RFC 4733 / SIP-INFO / in-band | Yes | SDES + DTLS-SRTP | Yes | Yes (zrtp module) | No (custom) | Yes (rec module) | H.264, H.265, VP8, VP9, AV1 | Yes (CLI by default) | No | No |
| Twinkle | GPL (FOSS) | Linux only | Full | G.711, G.722, Speex, GSM, iLBC | RFC 4733 / SIP-INFO / in-band | Yes (older) | SDES, ZRTP | Limited | Yes | N/A | No | None | Limited | No | No |
| sipML5 | BSD (deprecated) | Browser | WebRTC + SIP-over-WS | Opus, G.711, OPUS-WebRTC | RFC 4733 | Yes (WSS) | DTLS-SRTP (WebRTC) | Yes | No | N/A | No | VP8, H.264 | No | No | No |
| SIP.js / JsSIP | MIT (libraries) | Browser | WebRTC + SIP-over-WS | Opus, G.711 | RFC 4733 | Yes (WSS) | DTLS-SRTP (WebRTC) | Yes | No | N/A | App-defined | VP8, VP9, H.264 | N/A | App-defined | No |

## Linphone

Open-source SIP softphone from Belledonne Communications (Grenoble, France). Released under GPLv3 — the application binaries are free, but commercial use of the SDK (Liblinphone) for embedding into your own products requires a commercial license. The Belle-SIP stack underneath is also dual-licensed.

**Components:**

- **Linphone Desktop** — GTK app on Linux, Qt5/Qt6 app on macOS and Windows. Recent versions (5.x) standardised on Qt for cross-platform UI consistency.
- **Linphone iOS / Android** — full-featured mobile clients with push notifications via Belledonne's push gateway.
- **Linphone Web** — WebRTC-based browser client (newer; not feature-equivalent with desktop).
- **linphonec / linphonecsh** — CLI tools, useful for headless servers, scripting, kiosks, and embedded systems. linphonec is a fully interactive shell; linphonecsh is a thin daemon-control wrapper.
- **Liblinphone SDK** — C / C++ / Java / Swift / C# bindings; the same engine that drives the GUI. Used to embed SIP UA functionality inside other applications.
- **Belle-SIP** — the underlying SIP transaction stack (RFC 3261 + extensions). Independent C library; can be used standalone.

**Codec set:** Opus, G.711μ/A, G.722, G.729 (commercial license required for distribution), iLBC, AMR, AMR-WB, SILK, Speex, GSM. Video: H.264, H.265 (paid module), VP8, VP9, AV1.

**Encryption:** SDES, DTLS-SRTP, ZRTP — all supported, all configurable per-account.

**SDK development tier:** the FOSS app is free; embedding Liblinphone in a commercial product requires a Belledonne commercial license. The split is: anyone can compile and distribute the standalone Linphone app under GPLv3; you cannot link Liblinphone into closed-source software without buying a license.

## Linphone CLI Workflow

linphonec is the interactive command-line client. linphonecsh is the daemon controller — useful for cron jobs, scripts, and embedded systems where you want to launch a daemon, send commands, and exit.

```bash
# Start linphonec interactive
# -d <verbosity> for global; -d 6 = max debug; -d 0 = silent
# -c <file> for explicit config path
linphonec -d 0 -c ~/.linphonerc

# Daemon-mode: start a background daemon
linphonecsh init -c ~/.linphonerc -d 0
# Now send commands to the daemon
linphonecsh generic "register sip:alice@example.com sip:proxy.example.com password"
linphonecsh generic "call sip:bob@example.com"
linphonecsh generic "terminate"
linphonecsh exit

# Common interactive commands inside linphonec:
register sip:alice@example.com sip:proxy.example.com secret  # register account
unregister                                                    # unregister
proxy add                                                     # add proxy interactively
call sip:bob@example.com                                      # outgoing call
terminate                                                     # hang up current call
answer                                                        # accept incoming
calls                                                          # list active calls
transfer <call_id> sip:carol@example.com                      # blind transfer
dtmf 1234#                                                    # send DTMF digits
hold                                                          # put call on hold
resume                                                        # resume from hold
mute / unmute                                                 # mic mute toggle
codec list                                                    # list codecs + priority
codec disable G729                                            # disable a codec
codec enable opus                                             # enable a codec
audio-codec move opus 0                                       # move opus to top priority
status register                                               # show registration state
status hook                                                   # show call state
soundcard list                                                # list audio devices
soundcard use 1                                               # select audio device by index
firewall stun stun.example.com:3478                           # configure STUN
quit                                                          # exit linphonec
```

The `~/.linphonerc` config file is INI-style. Sections like `[net]`, `[sound]`, `[sip]`, `[proxy_0]`, `[auth_info_0]`, `[video]`, `[rtp]`. Editing it directly is supported and common for headless deployments.

```ini
# ~/.linphonerc minimal example
[sip]
sip_port=5060
sip_tcp_port=5060
sip_tls_port=5061
default_proxy=0

[proxy_0]
reg_proxy=<sip:proxy.example.com;transport=tls>
reg_route=<sip:proxy.example.com;transport=tls;lr>
reg_expires=600
reg_identity=sip:alice@example.com
reg_sendregister=1
publish=0

[auth_info_0]
username=alice
userid=alice
passwd=secret
realm=example.com

[net]
mtu=1300
firewall_policy=2  # 0=none, 1=NAT-from-config, 2=STUN, 3=ICE, 4=upnp
stun_server=stun.example.com:3478

[rtp]
audio_rtp_port=7078
video_rtp_port=9078
audio_jitt_comp=60
video_jitt_comp=60

[sound]
playback_dev_id=ALSA: default
capture_dev_id=ALSA: default
ec=1   # echo cancellation on
ec_tail_len=128
agc=0  # automatic gain control

[video]
enabled=1
size=vga

[audio_codec_0]
mime=opus
rate=48000
channels=1
enabled=1
[audio_codec_1]
mime=PCMU
rate=8000
channels=1
enabled=1
[audio_codec_2]
mime=PCMA
rate=8000
channels=1
enabled=1
```

## Linphone Account Setup

In the GUI, Settings → Account Settings → Add → SIP account. Fields:

- **SIP Address** — `sip:user@domain` form. The domain part is the SIP-domain (logical realm), not necessarily the server hostname.
- **Password** — auth password (becomes auth_info).
- **Server Address (Proxy)** — `<sip:proxy.example.com;transport=tls>`. Distinct from the SIP-domain. Many providers have `sip:user@example.com` but proxy is `sip:edge.provider.net:5061`.
- **Outbound Proxy** — separate "force route through" proxy if needed (the Route header).
- **Transport** — UDP / TCP / TLS / DTLS. TLS is over TCP; DTLS is rare. Default to TLS for new deployments.
- **Registration timeout** — seconds; the REGISTER expires value the UA wants. Servers may reduce it.
- **Publish presence** — toggle PUBLISH for SIMPLE presence (RFC 3856). Most enterprise deployments leave this off unless presence-aware.
- **Audio codec preferences** — per-account override of the global codec list and priority order.
- **Video codec preferences** — same for video.
- **AVPF** — toggle the use of RTP/AVPF profile (RFC 4585), which adds RTCP feedback for video congestion control. Required for some video-quality features.
- **Quality reporting** — RFC 6035 vq-rtcpxr + RTCP-XR; enable if your SIP provider tracks call quality.
- **Push notifications** — for mobile, the Linphone push gateway URL.

## Bria (Counterpath)

Counterpath's commercial softphone family. Counterpath was acquired by Alianza in 2020, but the Bria product line continues. Bria is the modern unified product — predecessors were eyeBeam (full-featured) and X-Lite (free, limited).

**Tiers:**

- **Bria Solo** — subscription for individuals/small business. Single-user license, hosted account management, push notifications via Counterpath PNS.
- **Bria Enterprise** — corporate deployment with central management (Bria Stretto / Bria Teams Pro), SSO (SAML/OIDC), MDM-deployable, BroadSoft XSI integration, contact-center features, presence federation.

**Platforms:** Windows, macOS, iOS, Android. Web SDK exists separately as Bria Mobile WebRTC.

**Codecs:** G.711, G.722, G.729, Opus, iLBC, SILK, AMR-WB. G.729 is included (Counterpath has the license).

**Predecessors (legacy, no longer sold):**

- **X-Lite** — free, limited (no video, no SRTP, no recording, no advanced codecs). Counterpath's gateway product to upsell to Bria.
- **eyeBeam** — paid, full-featured, predecessor to Bria. Discontinued circa 2010-2011.

## Bria Configuration

Account → SIP → fields:

- **Account Name** — display label only.
- **Display As (Display Name)** — the RFC 3261 display-name in the From header. Shown on the called party's caller ID.
- **User ID (Username)** — the SIP username (left-side of `sip:user@domain`).
- **Domain** — the SIP-domain (right-side of `sip:user@domain`). Distinct from the registrar host.
- **Password** — auth password.
- **Authorization Name** — separate from username for providers where the auth-user differs from the SIP-user (e.g., `auth=12345` but `from=alice@example.com`). If left blank, defaults to User ID.
- **Outbound Proxy / Domain Proxy** — explicit registrar/proxy host:port if not derivable from Domain via SRV/NAPTR.

Advanced → Audio:

- **Codec preference order** — drag to reorder. Default: G.722, Opus, G.711, G.729. Right side = enabled list, left side = available.
- **Jitter buffer** — adaptive (default) vs fixed. Min/max in ms.
- **Echo cancellation** — software AEC. Tail length configurable. Disable if using a hardware-AEC headset (double AEC degrades audio).
- **Noise reduction** — software NR; aggressive setting can clip speech.
- **AGC (automatic gain control)** — software AGC; can pump on quiet/loud transitions.
- **Voice Activity Detection (VAD)** — Silence Suppression / Comfort Noise.
- **DTMF mode** — RFC 4733 (telephone-event RTP), SIP-INFO, or In-band.

Advanced → Video:

- **Resolution** — QCIF / CIF / VGA / HD / FullHD.
- **Frame rate** — 15, 24, 30 fps.
- **Bitrate** — auto or fixed (kbps).
- **Codec list** — H.264 (with profile/level), VP8.

Network:

- **NAT keep-alive** — empty UDP packets every N seconds to keep NAT mappings open. Default 30s for UDP.
- **STUN server** — for ICE-Lite or NAT discovery.
- **ICE** — enable per-account.
- **Transport** — UDP / TCP / TLS. Auto-fallback (TCP if UDP > MTU, etc.).

## Bria Stretto Provisioning

Bria Stretto is Counterpath's central management server for Bria Enterprise. Lets administrators auto-config and deploy Bria across hundreds or thousands of corporate desktops without per-user manual entry.

**Workflow:**

1. Admin uploads a deployment XML to Stretto with account templates, branding, locked-down settings (e.g., "user cannot change codec order", "VPN required").
2. Bria client on first launch hits the Stretto auto-config URL (provided via MSI/PKG installer parameter or LDAP/AD discovery).
3. Bria authenticates the user (SSO, LDAP, or admin-issued temp credential), pulls the XML, applies the config.
4. Per-user identity (SIP username/password) is either: pulled from the LDAP record at provisioning time, generated by Stretto and pushed, or filled in by user on first run from a Welcome Email link.
5. Updates pushed centrally — change a setting in Stretto, all clients pick it up at next provisioning poll.

The "deploy 1000 desktops" workflow:

```bash
# Windows MSI silent install with provisioning URL
msiexec /i Bria5.msi /qn STRETTO_URL=https://stretto.corp.example.com/provision \
                          ENTERPRISE_USER=%USERNAME%

# macOS PKG with provisioning preferences
sudo defaults write /Library/Preferences/com.counterpath.bria5 StrettoURL https://stretto.corp.example.com/provision
sudo installer -pkg Bria5.pkg -target /
```

Stretto features include: branding override (logo, colors, app name "Acme Phone" instead of "Bria"), forced setting lockdown (user UI greyed out for locked fields), provisioning per AD-group (executives get HD video, call-center gets G.711-only), MDM integration on iOS/Android via per-device profiles.

## Zoiper

Zoiper is commercial freemium from Securax (Bulgaria). Long-running brand — Zoiper 3 (legacy, end-of-life) was Adobe-AIR-based; Zoiper 5 is the modern native cross-platform client.

**Platforms:** Windows, macOS, Linux, iOS, Android. Web Phone (paid).

**Generations:**

- **Zoiper 3** — legacy, AIR-based, very dated. Still seen in the wild; should be replaced by Zoiper 5. Configuration UI patterns are different and incompatible.
- **Zoiper 5** — current. Native per-platform. Fully compatible with modern SIP/SRTP/Opus.

**Tiers:**

- **Free** — basic SIP UA. G.711μ/A only. No SRTP. No video. No call recording. Ad banner.
- **Pro (one-time)** — adds wideband codecs (G.722, iLBC, Opus), TLS, SRTP, video, recording, call transfer, conferencing. Removes ads.
- **Premium** — all of Pro plus G.729 codec license (separate licensing fee), AMR-WB, additional Premium codecs.

## Zoiper Configuration

Settings → Accounts → Add Account.

**Wizard mode (Auto-Detect):**

1. Enter `username@domain` and password.
2. Zoiper queries known providers (Twilio, RingCentral, 8x8, Vonage, etc.) and pre-fills proxy/transport.
3. If domain matches a known template, proceeds. Otherwise falls back to manual.

**Manual mode:**

- **Hostname or provider** — SIP domain (the right-hand side of `sip:user@domain`).
- **Username** — SIP user.
- **Password** — SIP auth password.
- Click "Next" → Zoiper auto-tests UDP/TCP/TLS, picks the working transport, and pre-fills the proxy.

**Advanced (post-creation, Edit Account):**

- **SIP options** — outbound proxy, transport (UDP/TCP/TLS), local port, register timeout, NAT keep-alive interval.
- **Audio codecs** — drag-reorder enabled set; available codecs depend on tier (see below).
- **Video codecs** — Pro+ only; H.263, H.264, VP8.
- **DTMF** — RFC 4733 (default), SIP-INFO, in-band (legacy).
- **Encryption** — SRTP-SDES (Pro+), ZRTP (paid add-on).
- **Features** — voicemail dial code, BLF (Busy Lamp Field), MWI (Message Waiting Indicator).

## Zoiper Codec Set

| Tier | Available Codecs |
|------|------------------|
| Free | G.711μ-law, G.711A-law |
| Pro | + G.722, iLBC, Opus, Speex (narrowband + wideband + ultra-wideband), GSM |
| Premium | + G.729 (separate license fee), AMR-WB |

The Free-tier G.711-only restriction is the most common reason for "I configured Zoiper Free and the call sounds bad." If the server expects G.722 or Opus for HD voice and the client only offers G.711, both sides downgrade to narrowband. Pro tier removes this restriction.

G.729 in Premium tier is licensed separately because Sipro Lab Telecom (and historically the original patent pool) charge per-seat royalties. Premium covers your seat license for G.729; the Free/Pro tiers omit it to avoid the royalty.

## MicroSIP

Windows-native, lightweight, FOSS (GPL) softphone built on PJSIP. Single developer. Very small binary (~5 MB), very low memory and CPU footprint. Popular for kiosks, call-center seats on minimal Windows boxes, and any "I just need a SIP phone, no bells and whistles" scenario.

**Strengths:**

- Tiny installer, fast launch.
- Per-account codec preference.
- Full PJSIP-derived feature set: TLS, SRTP, ICE, video (H.264), DTMF (all modes).
- Tray-icon hide-on-close — no taskbar clutter.
- No telemetry, no ads, no nag.
- INI-based portable config — copy `Microsip.ini` and `accounts.ini` between machines.

**Limitations:**

- Windows only.
- No Linux, no macOS, no mobile.
- Single developer — feature pace is modest.
- No headless CLI mode (GUI only).

## MicroSIP Configuration

Menu → Add Account.

- **Account name** — display label.
- **SIP server** — SIP domain (right-hand side of address-of-record). Distinct from registrar host if SRV-resolved.
- **SIP proxy** — optional explicit proxy host:port.
- **Username** — SIP user.
- **Domain** — left blank by default; same as SIP server unless they differ.
- **Login** — Authorization Name (defaults to Username).
- **Password** — auth password.
- **Display name** — From header display-name.
- **Voicemail number** — dial-string for VM.
- **Dialing prefix** — prepended to all outgoing.
- **Dial plan** — regex transformations (e.g., `9XXXXXXX` for outside line).
- **Hide caller ID** — Privacy: id header.
- **Disable session timers** — RFC 4028 toggle.
- **Public address (NAT)** — auto / disabled / STUN / explicit.
- **STUN server** — host:port.
- **Transport** — UDP / TCP / TLS.
- **Public address discovery** — when behind NAT.
- **Allow rewrite contact / via** — for buggy registrars.

Per-account codec preferences in Edit Account → Codecs tab. Drag-reorder; click to enable/disable.

Hide on close: Settings → "Hide on close" — clicking X minimizes to tray instead of quitting; only File → Exit actually closes the daemon.

`accounts.ini` and `Microsip.ini` live in `%APPDATA%\MicroSIP\` (or alongside the EXE if portable mode). Copy to deploy.

## Jami

Open-source GPLv3, decentralized P2P SIP-and-more softphone. Originally **Ring** (2016-2018), renamed Jami in 2018 by Savoir-faire Linux when Cisco's "Ring Central" naming objection (and clarity) prompted the rename.

**Architectural distinction from traditional SIP:**

- No central SIP registrar. No SIP server at all (in P2P mode).
- Each Jami account = an X.509 certificate generated locally on first install.
- Account identifier (Jami ID) = SHA1 hash of the certificate's public key — a 40-hex-char string like `abc123...`.
- Peer discovery via OpenDHT, a Kademlia-style distributed hash table. Your Jami client publishes its current IP/port to the DHT against its hash; callers look it up.
- Optional username layer: register a human-readable name (`alice`) on the Jami namespace server (a centralized, optional convenience), which maps `alice` → `abc123...` hash. Without it, contacts have to share the 40-char hash.
- ZRTP encryption is mandatory for media — no plaintext mode. SAS verification on first call.
- TURN-relay fallback when direct P2P fails (corporate NAT, double-NAT, restrictive firewalls). Jami operates a public TURN, or you self-host.
- Also supports traditional SIP accounts for interop with regular PBXs — same client UI, both account types coexist.

**Use cases:**

- Privacy-sensitive comms — no central log, no metadata at a server.
- Federated/sovereign deployments — local network only, no internet dependency.
- SIP-equivalent functionality with end-to-end encryption guaranteed.
- Cross-platform (Linux/Mac/Win/iOS/Android).

## Jami Specifics

**No SIP traditional model.** A traditional SIP account requires a registrar; Jami in P2P mode replaces this with the OpenDHT lookup. There's no REGISTER, no INVITE-via-proxy. The SIP messages still exist on the wire (Jami uses SIP-over-TLS as the signaling protocol after DHT-discovery establishes the peer's IP), but they go peer-to-peer.

**Account = X.509 cert.** On account creation, Jami generates a 4096-bit RSA keypair and a self-signed X.509 cert. Cert + private key are stored on-device. **Loss of device = loss of account** unless the user exported a backup. Multi-device support involves linking a new device by transferring the cert via QR code or PIN-mediated handshake.

**Identifier = SHA1 of cert.** The Jami ID (40-char hex) is the SHA1 of the certificate. Used as the DHT key. Two devices linked to the same Jami account share the same cert and thus the same Jami ID — calls reach all linked devices.

**Optional username on Jami namespace server.** A central HTTP service (`ns.jami.net` by default; can be self-hosted) maps usernames to Jami IDs. Registering `alice@jami.net` is a courtesy layer for human-friendly identification. Without it, users share the 40-char hash via QR code, email, etc. The username is **not** essential to the protocol.

**TURN-relay fallback when P2P fails.** When NAT traversal fails (symmetric NAT both sides, or restrictive firewall), Jami falls back to a TURN relay. Public default: `turn.jami.net`. Self-hosted: configure your own coturn. The relay sees encrypted bytes (ZRTP-encrypted RTP + TLS-encrypted SIP) so it cannot eavesdrop, only forward.

## Jitsi (Jitsi Meet vs Jitsi Desktop)

Important disambiguation:

- **Jitsi Meet** — the modern WebRTC video conference application. Browser-based, plus iOS/Android apps. Not a SIP softphone — it's a WebRTC SFU client. Used at meet.jit.si, self-hosted as Jitsi Meet, integrated into Element/Slack/etc.
- **Jitsi Desktop** — the original Jitsi (formerly SIP Communicator). Java-based desktop SIP/XMPP softphone. Cross-platform via JVM. Still maintained, though attention has shifted to Jitsi Meet.

When someone says "Jitsi as a softphone," they almost always mean Jitsi Desktop, not Jitsi Meet.

**Jitsi Desktop specifics:**

- License: LGPL (libs) + Apache (app).
- Java/JNI; runs on Linux, macOS, Windows.
- Supports SIP, XMPP, IRC, ICQ (legacy), MSN (legacy), AIM (legacy). The IM protocols are largely deprecated; SIP and XMPP are the actively used ones.
- Integrated with **Jitsi Videobridge** for SIP-to-SFU bridging — you can join a Jitsi Meet conference from a SIP client, gatewayed by Jitsi Videobridge.
- ZRTP: yes (Phil Zimmermann's Z phone protocol; Jitsi was an early adopter).
- DTLS-SRTP: yes.
- Codec set: G.711, G.722, Opus, iLBC, SILK, Speex; H.264, H.263, VP8 for video.

## Jitsi Desktop Configuration

Tools → Options → Accounts → Add → SIP.

- **Service** — choose SIP.
- **SIP id** — `sip:user@domain` form.
- **Password** — auth password.
- **Display name** — From header display-name.
- **Server / Proxy** — explicit registrar/proxy hostname.
- **Authorization name** — if different from SIP user.
- **Default transport** — UDP / TCP / TLS.

Advanced (per-account):

- **Connection** — outbound proxy, transport, listening port.
- **Encryption** — toggle SDES-SRTP, DTLS-SRTP, ZRTP. Order of preference.
- **Presence** — XMPP-style presence over SIP (SIMPLE).
- **ICE** — enable, list STUN servers, list TURN servers (with credentials).
- **Audio codecs** — drag-reorder.
- **Video codecs** — drag-reorder, plus quality presets.

Tools → Options → Audio:

- **Capture device** — selects mic from the OS-exposed list.
- **Playback device** — selects speaker.
- **Notifications device** — separate output for ringtones.

Tools → Options → Video:

- **Camera device** — selects from list.
- **Resolution** — auto / per-resolution.

## Telephone (iOS / macOS)

Open-source MIT-licensed softphone for macOS and iOS. Built on PJSIP (PJSUA2 underneath). Famous for its minimalist UI — looks like the macOS Phone app would, if Apple had ever made one. Single developer (Alexey Kuznetsov).

**Strengths:**

- Native macOS / iOS look and feel.
- SIP-only — no extra protocols, no chat, no presence — keeps the UI clean.
- PJSIP under the hood means full codec/SRTP/ICE support.
- The "Mac mini call center" use case — set up a Mac mini, plug in a USB headset, run Telephone, hand it to the receptionist.
- Free, open source, no nag, no ads.

**Limitations:**

- macOS / iOS only (no Linux, no Windows, no Android).
- No video (audio only).
- No call recording.
- Limited push-notification support on iOS (PushKit integration is partial).
- Limited customization vs Bria/Linphone.

## Telephone Configuration

Telephone → Preferences → Accounts → + (add) → SIP Account.

- **Server** — registrar hostname (e.g., `sip.example.com`). The SIP-domain part of the AOR is derived from this, or you can override.
- **Username** — SIP user (left of `@`).
- **Password** — auth password.
- **Domain** — SIP-domain (right of `@`); if blank, same as Server.
- **Reregister every** — seconds.
- **SIP transport** — Auto / TCP / TLS.

Advanced tab:

- **SIP proxy** — explicit outbound proxy.
- **Authentication name** — if different from Username.
- **Update Contact header / Via** — NAT rewrite toggles.

That's it. The simplicity is the point — it's deliberately fewer fields than Bria or Linphone.

Sound preferences: Telephone → Preferences → Sound. Selects input/output device, ringtone, ringtone device.

## Acrobits

Commercial mobile-focused softphone vendor based in Prague. Two product lines:

- **Acrobits Softphone** (also marketed as **Groundwire** in some app stores) — branded retail iOS/Android SIP softphone.
- **Cloudsoftphone** — white-label SDK / app builder. Telecom carriers and PBX vendors use Cloudsoftphone to ship branded softphones to their customers — same engine, custom logo and account-provisioning flow.

**Mobile-first focus.** Acrobits historically specialized in iOS, where Apple's background-execution restrictions make persistent SIP registration impossible. Acrobits solved this with their **PNS (Push Notification Service)** architecture:

- Acrobits operates a SIP registrar/proxy in their cloud (or you self-host the Acrobits PNS).
- Your softphone registers to Acrobits PNS, **not** directly to your PBX.
- Acrobits PNS forwards SIP traffic to your real PBX/SIP-trunk.
- When an INVITE arrives at Acrobits PNS for a registered (sleeping) device, Acrobits sends an **APNS (Apple Push Notification Service) VoIP push** or **FCM data message** to wake the device.
- The device wakes, the app foregrounds (briefly), the SIP socket is re-established (or maintained via PushKit's pre-warm), and the call proceeds.

This "Acrobits PNS in the middle" architecture is what makes Acrobits the de facto choice for "I need a SIP softphone on iOS that actually rings." Without PNS, iOS will background-suspend any app holding a TCP socket, and incoming calls miss.

Cloudsoftphone provides the SDK + cloud PNS so any reseller can ship "MyVoIP App" on the App Store using Acrobits' engine, branded for their service.

## Acrobits Codec Set

| Codec | Tier |
|-------|------|
| G.711μ-law | Free / built-in |
| G.711A-law | Free / built-in |
| G.722 | Built-in |
| Opus | Built-in |
| iLBC | Built-in |
| G.729 | Paid in-app purchase add-on |

Acrobits ships G.729 as a paid IAP because of the per-seat licensing model — they pass through the royalty cost to users who specifically need G.729 (typically interop with legacy PBXs that don't speak Opus or G.722).

## 3CX Softphone

3CX is a commercial PBX vendor (3CX Phone System); their softphone is tightly tied to their server. Free for use with a 3CX PBX; not licensed standalone.

**Platforms:** Windows, macOS, iOS, Android, Web (browser-based via WebRTC).

**Provisioning model — 3CX Welcome Email:**

1. The 3CX PBX administrator creates an extension for the user.
2. 3CX generates a "Welcome Email" with a one-time provisioning URL or QR code.
3. User clicks the URL on their device (or scans the QR with the mobile app).
4. The softphone fetches the entire account config from the PBX — username, password (or token), SRTP keys, server hostname, port, transport — and self-configures.
5. Subsequent updates (extension changes, codec policy) are pushed automatically.

**Codecs:** G.711μ/A, G.722, Opus, iLBC. No G.729 (3CX deliberately steered away from licensed codecs).

**Push notifications:** 3CX runs their own PNS bridge for iOS/Android. Like Acrobits, the architecture is "phone registers to 3CX PBX which knows to send a push when call arrives."

**Teams compatibility:** 3CX has a direct SBC integration with Microsoft Teams Direct Routing — calls from a Teams user reach a 3CX-registered SIP phone via 3CX SBC. The 3CX softphone itself doesn't speak Teams, but the PBX it registers to can bridge.

3CX deliberately limits non-3CX use — their SIP credentials are scoped to a 3CX server. You can't take "3CX softphone" and point it at Asterisk or FreeSWITCH; while technically the SIP stack works, the provisioning model and licensing don't.

## pjsua

The reference SIP UA from PJSIP. CLI-only; useful for testing, scripting, automation, headless servers, embedded targets. Not a daily-driver softphone for end users — it's the developer's tool.

**Why pjsua matters:**

- The PJSIP library powers MicroSIP, Telephone, Acrobits, Sipgate's softphone, pjsua2 bindings into Python/Java/Swift, etc.
- pjsua is the canonical example app — anything PJSIP can do, pjsua exposes via CLI flags.
- "Headless test client" — register, place a call, verify audio, hang up, all scriptable.
- "Auto-answer" mode for honeypots, call-center test harnesses, IVR validation.
- Cross-compiles to embedded Linux/Android/iOS — same tool runs on a Raspberry Pi as on your laptop.

```bash
# Get the full flag list
pjsua --help | less

# Register and stay running
pjsua \
  --id "sip:alice@example.com" \
  --registrar "sip:proxy.example.com:5060" \
  --realm "*" \
  --username "alice" \
  --password "secret" \
  --transport "tcp" \
  --reg-timeout 600 \
  --no-vad \
  --auto-answer 200          # auto-answer every call with 200 OK

# Place an outbound call from inside pjsua's REPL
# After launch, type:
m
sip:bob@example.com
# 'm' = make call, then enter URI

# Other REPL commands:
# h           = hangup
# H           = hold
# v           = re-INVITE
# d           = duplicate / refresh registration
# q           = quit
# # <code>    = send DTMF (RFC 4733)

# Use a config file instead of long CLI
pjsua --config-file /path/to/pjsua.cfg

# Auto-redirect / auto-call test pattern
pjsua --id "sip:tester@example.com" \
      --registrar "sip:proxy.example.com" \
      --username "tester" --password "secret" \
      --auto-answer 200 \
      --auto-loop                # auto-call back any caller

# Codec management
pjsua --add-codec opus/48000/2 \
      --add-codec PCMA/8000 \
      --quality 8                # codec quality 0-10

# Audio device selection
pjsua --capture-dev 0 --playback-dev 0
pjsua --null-audio                # no actual audio I/O — for protocol testing only

# Recording
pjsua --rec-file /tmp/call.wav --auto-rec

# TLS with custom cert
pjsua --use-tls --tls-cert-file /etc/pki/cert.pem --tls-privkey-file /etc/pki/key.pem
```

## pjsua Configuration

Config file (`pjsua.cfg`) is whitespace-separated `--flag value` pairs, one per line; same as the CLI options. Example:

```ini
# pjsua.cfg
--id sip:alice@example.com
--registrar sip:proxy.example.com:5060
--realm *
--username alice
--password secret
--transport tls
--reg-timeout 600
--auto-answer 200
--auto-loop
--null-audio
--add-codec PCMU/8000
--add-codec PCMA/8000
--add-codec opus/48000/2
--use-srtp 2
--srtp-secure 1
--use-ice
--stun-srv stun.example.com:3478
--log-level 4
--app-log-level 4
--log-file /var/log/pjsua.log
```

Sections in larger configs are conceptual (account, proxy, codec, audio, log, network) — pjsua flags are flat but logically grouped.

**Auto-answer + auto-redirect-to-call test patterns** — `--auto-answer 200 --auto-loop` makes pjsua a poor man's echo/loopback test client. Combined with `--null-audio` it answers every call with 200 OK and echoes the audio back, useful for SIP-trunk smoke tests and registration-flow validation.

**Headless test client use case:**

```bash
# CI smoke test: can we register and place a call?
pjsua --config-file /test/pjsua.cfg --null-audio --no-tcp &
PJ_PID=$!
sleep 5
# Send a call command via stdin (pjsua reads stdin)
echo "m" >> /tmp/pjsua.in
echo "sip:test@target-pbx.example.com" >> /tmp/pjsua.in
sleep 10
echo "q" >> /tmp/pjsua.in
wait $PJ_PID
# Inspect /var/log/pjsua.log for INVITE → 200 OK sequence
```

## baresip

Modular CLI SIP UA from Creytiv (Alfred E. Heggestad). BSD-licensed (very permissive). Plugin/module architecture — you build the binary with exactly the modules you need, leaving everything else out. Lua scripting available. The de facto choice for embedded SIP, custom kiosks, and "I need a softphone but also a custom feature."

**Strengths:**

- Modular: include only what you need (small footprint).
- Lua scripting: automate calls, integrate with external systems.
- Cross-platform: Linux, macOS, Windows, iOS, Android, embedded ARM.
- Excellent codec support (audio + video).
- Active maintenance.

**Configuration files:**

- `~/.baresip/config` — main config (modules, audio, video, network, SIP).
- `~/.baresip/contacts` — contact list (one URI per line).
- `~/.baresip/accounts` — account list (one SIP URI with auth params per line).

```bash
# Install (Debian/Ubuntu)
apt install baresip

# Or build from source for custom module set
git clone https://github.com/baresip/baresip
cd baresip
make MODULES="opus alsa stun ice srtp dtls_srtp"
sudo make install

# Run interactively
baresip

# Common commands inside baresip CLI:
# d <uri>      = dial
# b            = hangup
# r            = list registered accounts
# l            = list contacts
# /            = type a SIP message / search command
# t            = transfer
# h            = answer
# m            = mute
# v            = video toggle
# q            = quit
# # <digit>    = DTMF

# Run headless with a script
baresip -e "/dial sip:bob@example.com" -e "/hangup"

# Daemon mode with HTTP control
baresip -d   # daemon
# Then control via the http_req module's API on localhost:8000
curl http://localhost:8000/raw/dial sip:bob@example.com
curl http://localhost:8000/raw/hangup
```

`~/.baresip/accounts`:

```
<sip:alice@example.com;auth_pass=secret;outbound1="sip:proxy.example.com;transport=tls";audio_codecs=opus,PCMU,PCMA;video_codecs=H264,VP8;mediaenc=srtp;ptime=20>
```

`~/.baresip/contacts`:

```
"Bob" <sip:bob@example.com>
"Carol" <sip:carol@example.com>
```

## baresip Modules

Modules are loaded in `~/.baresip/config` via `module <name>` directives. Common modules:

```
# Audio drivers (pick one for your platform)
module                  audio_alsa.so       # Linux ALSA
module                  audio_pulse.so      # Linux PulseAudio
module                  audio_pipewire.so   # Linux PipeWire (newer)
module                  audio_coreaudio.so  # macOS CoreAudio
module                  audio_jack.so       # JACK (pro-audio)
module                  audio_winwave.so    # Windows WaveOut
module                  audio_wasapi.so     # Windows WASAPI

# Video drivers
module                  video_v4l2.so       # Linux v4l2 (USB cams)
module                  video_avfoundation.so  # macOS AVFoundation
module                  video_dshow.so      # Windows DirectShow

# Audio codecs
module                  opus.so
module                  amr.so
module                  g711.so             # always-on; G.711μ/A
module                  g722.so
module                  ilbc.so
module                  speex.so
module                  codec2.so           # very-low-bitrate

# Video codecs
module                  vidcodec_x264.so    # H.264 (libx264)
module                  vidcodec_h265.so    # H.265
module                  vp8.so              # VP8
module                  vp9.so              # VP9
module                  av1.so              # AV1 (newer)

# Signaling / SIP extensions
module                  presence.so         # SIMPLE PUBLISH/SUBSCRIBE
module                  contact.so          # contact list
module                  account.so          # multi-account
module                  mwi.so              # message-waiting indicator
module                  natpmp.so           # NAT-PMP

# NAT / ICE
module                  ice.so
module                  stun.so
module                  turn.so

# Encryption
module                  srtp.so             # SRTP-SDES
module                  dtls_srtp.so        # DTLS-SRTP (WebRTC-style)
module                  zrtp.so             # ZRTP

# Recording / media
module                  rec.so              # call recording
module                  snapshot.so         # video snapshot
module                  vumeter.so          # audio level meter

# Control / scripting
module                  cons.so             # console UI
module                  httpd.so            # HTTP control API
module                  ctrl_tcp.so         # TCP control protocol
module                  ctrl_dbus.so        # DBus control
module                  lua.so              # Lua scripting
```

The modular architecture means: "you build the binary with modules you need." A kiosk with a custom audio path needs only audio_alsa, opus, g711, srtp, ice, stun, ctrl_tcp — perhaps 8 modules. The full module set has 60+. Embedded targets (busybox-based routers, OpenWRT) typically build a stripped baresip with under a megabyte of binary size.

## SIPFoundry sipXezPhone, SIPp, Twinkle, QuteCom / WengoPhone

Adjacent and legacy clients:

- **sipXezPhone** — part of the SIPfoundry sipXcom suite. Java-based desktop SIP softphone. Largely abandoned; sipXcom itself was archived. Useful only if you're already in sipXcom-land.
- **SIPp** — **not a softphone**, but adjacent. SIPp is a SIP traffic generator/load tester. Crafts SIP messages from XML scenarios, drives thousands of concurrent simulated UAs. Used to load-test PBXs, stress SBCs, validate SIP-trunk capacity. Standard tool in any SIP-engineer's bag. `sipp -sf scenario.xml -sn uac -d 5000 -r 100 target.example.com`.
- **Twinkle** — Linux Qt-based SIP softphone. FOSS. Strong on ZRTP (one of the early adopters), but development stalled; somewhat dated UI. Last release several years back. Still functional, still installed via Debian/Ubuntu repos, but not actively developed.
- **QuteCom / WengoPhone** — legacy. WengoPhone was an early-2000s open-source SIP softphone by the French ISP Wengo. Forked into QuteCom around 2008. Both projects are dormant. Mentioned mostly for historical context — if you're auditing old VoIP deployments, you may encounter them.

## Web-Based Softphones

Browser-resident SIP user agents — registration and signaling over WebSocket, media via WebRTC.

**sipML5** — first-generation HTML5+JavaScript SIP UA library. Worked, but development effectively stopped around 2015. Largely deprecated; modern browsers and security policies (mandatory DTLS-SRTP, dropped insecure WebRTC features) have outpaced it. Still seen in some legacy deployments.

**JsSIP** — modern, actively maintained JavaScript SIP library. WebRTC-native. Supports SIP-over-WebSocket (RFC 7118). Good for embedding SIP-call buttons into web apps. License: MIT.

**SIP.js** — competing modern JavaScript SIP library; arguably the most popular today. Same use case as JsSIP. License: MIT.

**Architecture:**

- Browser opens a WebSocket (WSS) to the SIP server on port 7443 or similar.
- SIP messages flow over the WebSocket — RFC 7118 specifies the SIP-over-WebSocket transport.
- Media (RTP) flows over WebRTC — peer-connection negotiates DTLS-SRTP, ICE handles NAT, codecs are Opus + G.711 (the WebRTC-mandatory set).
- Server side needs a WebSocket-capable SIP proxy: Asterisk PJSIP, FreeSWITCH (mod_sofia with ws/wss), Kamailio (websocket module), OpenSIPS.

**Codecs:** WebRTC mandates Opus and G.711 in the audio set. VP8, VP9, H.264 in video. No G.729, no G.722, no iLBC (or only via codec hacks).

```javascript
// SIP.js minimal example
import { Web } from "sip.js";

const simpleUser = new Web.SimpleUser("wss://sip.example.com:7443", {
  aor: "sip:alice@example.com",
  userAgentOptions: {
    authorizationUsername: "alice",
    authorizationPassword: "secret",
  },
});
await simpleUser.connect();
await simpleUser.register();
await simpleUser.call("sip:bob@example.com");
```

```javascript
// JsSIP minimal example
const socket = new JsSIP.WebSocketInterface("wss://sip.example.com:7443");
const config = {
  sockets: [socket],
  uri: "sip:alice@example.com",
  password: "secret",
};
const ua = new JsSIP.UA(config);
ua.start();
ua.call("sip:bob@example.com", { mediaConstraints: { audio: true, video: false } });
```

**The WebRTC requirement** — modern browsers require HTTPS for getUserMedia (mic access), so the calling page must be served via HTTPS, and the WebSocket must be WSS (TLS). HTTP/WS will be blocked.

## Mobile Softphone Push Notification

The "wake the device for incoming call" requirement is mobile-specific and central to softphone UX on iOS/Android. Without push notifications, an incoming call cannot ring a backgrounded/locked phone — the OS has suspended the SIP socket and stopped the radio.

**iOS — Apple PushKit (VoIP push):**

- Apple's APNS has a special "VoIP push" type, delivered via PushKit framework.
- PushKit pushes wake the app even from a fully-suspended state, bypassing normal background-execution restrictions.
- The app is given a brief execution window (10-20 seconds) to: re-establish SIP, send 100 Trying / 180 Ringing, present the CallKit ring UI (mandatory in iOS 13+).
- iOS 13+ enforces CallKit reporting — if the app receives a VoIP push but doesn't report a call to CallKit, iOS terminates the app and may revoke its VoIP-push entitlement.
- Apple grants VoIP-push entitlements only to genuine VoIP apps (App Store review).

**Android — FCM data message:**

- Android uses Firebase Cloud Messaging (FCM) data messages.
- Android's "Doze mode" and "App Standby" can delay or batch FCM pushes — high-priority FCM messages bypass Doze (priority="high"), but the OS is more aggressive than iOS about killing background apps.
- Recent Android versions require the app to start a foreground service within seconds of receiving a high-priority push.

**The mid-PJSIP work to integrate:**

PJSIP added VoIP-push and FCM hooks across several releases. The integration involves:

1. App registers for APNS/FCM at install time, receives a device token.
2. App sends the token to the SIP server (or PNS gateway like Acrobits PNS or 3CX PNS) via a custom SIP header (`X-PUSH-TOKEN`) or REST API at REGISTER time.
3. Server stores the token associated with the SIP AOR.
4. When INVITE arrives, server checks: is the device foreground (active SIP socket)? If yes, normal INVITE. If no, send VoIP push to APNS/FCM with the call metadata.
5. App receives push, wakes, presents incoming call UI, completes SIP handshake.

Code-wise, PJSIP-based apps (MicroSIP for desktop is fine — only mobile cares; iOS/Android Telephone, Acrobits, Linphone Mobile, etc.) have to manage:

- The custom REGISTER header carrying the push token.
- The early-media handling — sometimes the server starts ringing tone before the app has fully registered.
- The "stale token" problem — when iOS/Android rotates the token, the app must re-REGISTER to update the server.

## ZRTP-Enabled Softphones

ZRTP is Phil Zimmermann's protocol for end-to-end media encryption with no PKI dependence — Diffie-Hellman in the RTP stream itself, with a Short Authentication String (SAS) for human verification.

**Softphones with built-in ZRTP support:**

- **Linphone** — yes; per-account toggle.
- **Jitsi Desktop** — yes; one of the early ZRTP adopters.
- **Silent Phone** (Silent Circle) — yes; ZRTP-mandatory.
- **Twinkle** — yes; one of the original ZRTP-supporting Linux clients.
- **Telephone (macOS)** — depends on the build; PJSIP supports ZRTP via libzrtp, but Telephone's GUI may not expose it.
- **baresip** — yes, via the `zrtp.so` module.
- **MicroSIP** — yes, via PJSIP underlying.

**The SAS (Short Authentication String) verification UX:**

- After ZRTP key exchange, both endpoints derive a 4-character (typically) verbal string from the shared secret.
- Each party reads their string aloud; they should match. If they match, no man-in-the-middle.
- Once verified, both clients persist the shared secret cache — subsequent calls between the same parties don't require re-verification (Continuity).
- If the SAS suddenly changes between sessions, the cache is invalidated — strong indicator of a MITM attempt or key-rotation.

The UX challenge: training users to actually read the SAS rather than ignore it. Every well-implemented ZRTP softphone displays the SAS prominently and a "Verify" button.

## Codec Negotiation Surprises

The classic failure: softphone advertises a wide codec set including Opus, but the server expects narrowband G.711, and the SDP intersection lands on G.711 — call connects but quality is mediocre (narrowband 3.4 kHz audio instead of full-band 20 kHz Opus).

**Why it happens:**

- The softphone offers (in SDP `m=audio` line): `97 9 0 8 101` → meaning Opus, G.722, G.711μ, G.711A, telephone-event.
- The server prefers (its order): G.711μ first, then G.711A, then telephone-event, ignores Opus.
- Per SDP-offer/answer rules, the answerer chooses any of the offered codecs they support — typically the **first** they support in **their own** preference order.
- Result: call gets G.711μ (8 kHz narrowband), even though both sides could have done Opus 48 kHz.

**Codec-priority importance:**

- The softphone's local "codec preference order" controls **only** what it offers and accepts; it doesn't force the server to honor its priority.
- For best quality: ensure both sides agree on the preferred codec via configuration (e.g., set Opus as priority on the PBX too), or remove the lower-quality codecs from the softphone offer entirely (so the server is forced to either accept the better codec or fail).

**Other surprises:**

- **G.722 sample-rate confusion** — historical RFC 1890 bug had G.722 RTP timestamps run at 8 kHz despite 16 kHz audio. Some softphones implement the buggy historical behavior, some implement the corrected RFC 3551 behavior — interop between the two produces double-speed or half-speed audio. The fix is usually to force G.711 instead.
- **Opus FEC requires AVPF** — Opus's in-band FEC (Forward Error Correction) requires negotiating the AVPF profile. Without it, FEC bits are sent but not interpreted. SDP-level fmtp parameters must align (`useinbandfec=1; usedtx=1`).
- **Asymmetric codec negotiation** — RFC 3264 says you can negotiate different codecs in send vs receive direction. Most softphones don't handle this gracefully — quality issues until both directions match.

## Audio Hardware Quirks

The OS audio stack is a frequent source of softphone audio woes — independent of the SIP/RTP layer.

**macOS CoreAudio sample-rate switching:**

- CoreAudio devices have a "current sample rate" that can be changed by any app.
- If your softphone opens a device at 48 kHz and another app (Spotify, Zoom, Teams) is using it at 44.1 kHz, CoreAudio will resample — quality OK, but CPU spiked.
- Some USB headsets only support 16 kHz — the OS resamples up to whatever the app asked for.
- macOS Audio MIDI Setup can pin a device's sample rate; useful for studio-style determinism.

**Linux PulseAudio vs PipeWire:**

- PulseAudio is the legacy desktop audio server. Mature, stable, well-supported by softphones.
- PipeWire is the modern replacement (Fedora 35+, Ubuntu 22.10+). Wire-compatible API but underlying architecture rewritten — better for pro-audio/JACK use cases.
- Softphones built against PulseAudio API (libpulse) typically work on PipeWire via PipeWire's PulseAudio compatibility shim. But edge cases bite — exclusive mode, latency, device hot-plug all behave subtly differently.
- Linphone, Jitsi, baresip all support both. Twinkle is PulseAudio-only (last release predates PipeWire).

**Windows WASAPI exclusive vs shared:**

- WASAPI (Windows Audio Session API) has shared mode (Windows mixer mediates) and exclusive mode (app owns the device, lowest latency).
- A softphone in exclusive mode locks out other apps — if Teams is in a meeting and your SIP softphone starts in exclusive mode, conflict.
- Shared mode is the default and the right choice for most softphones.
- Some softphones default to MME or DirectSound (older, higher-latency Windows audio APIs) for compat — set explicitly to WASAPI shared in modern Windows.

**The "test call to echo service" before going live:**

Most SIP providers offer an echo-test number (Twilio: `+19568724577`; Vonage: `*43`; FreeSWITCH default: `9196`; SIPgate: `0123456789`). Dial it, speak, hear yourself echoed back. If echo is clean, your audio chain works. If echo is muffled, distorted, gappy, or absent — the problem is in the audio stack, not the SIP layer.

## Echo Cancellation

Most softphones offer software AEC (Acoustic Echo Cancellation) — algorithms like WebRTC's AECM, Speex AEC, or proprietary DSP. Quality varies enormously.

**Reality:**

- Software AEC quality depends on: tail length (how long the echo persists in the room — 50ms in a small office, 500ms in a big conference room), nonlinear distortion (cheap speakers/mics create distortion that defeats linear AEC), double-talk handling (when both parties speak simultaneously).
- Hardware AEC headsets are better — the headset's own DSP cancels echo before the audio reaches the softphone. Examples: Plantronics/Poly Voyager, Jabra Evolve, Logitech Zone, Apple AirPods Pro (for headset use).
- The "speakerphone causes echo without AEC" reality — laptop speakers and built-in mic, in close proximity, are an echo loop. Without good AEC, the called party hears their own voice 100-300ms delayed. With AEC, it's mostly cancelled but still present at low level.

**Recommendations:**

- Wired headset > Bluetooth headset > wired earbuds > laptop speaker+mic combo.
- Enable software AEC always for laptop scenarios.
- Disable software AEC if your headset has hardware AEC (double-AEC sounds worse than either alone).
- Test with the echo-service before assuming "the other side has bad audio."

## Network Adaptation

Modern softphones implement multiple layers of network adaptation:

**Adaptive jitter buffer:**

- Initial buffer size 30-60ms; grows under high-jitter conditions, shrinks under low-jitter.
- Trades latency for smoothness — bigger buffer hides jitter but adds end-to-end delay.
- Most softphones expose only a "min/max" ms range or "low / medium / high" preset.
- Voice quality at >150ms one-way delay starts to feel "walkie-talkie" — bigger jitter buffer increases delay.

**PLC (Packet Loss Concealment):**

- Synthetic audio generated from the previous packet's spectrum to fill in for a lost packet.
- Works well for losses up to ~5%.
- Simple PLC: repeat last packet (audible glitch).
- Better PLC: pitch-coherent synthesis (Opus's built-in PLC, G.711 Appendix I PLC).
- At sustained >5% loss, even good PLC sounds garbled.

**FEC (Forward Error Correction):**

- Opus has in-band FEC: each packet contains a low-bitrate copy of the previous packet, decodable if the previous was lost.
- AMR-WB has FEC modes too.
- Trades bandwidth for resilience — typically +20-30% bitrate for good FEC.
- Negotiated via SDP fmtp parameters (`useinbandfec=1` for Opus).

**Per-codec quality under loss:**

- Opus: graceful degradation, FEC helps significantly. Up to 30% loss can still be intelligible.
- G.711: no FEC, simple PLC. Above 3% loss, audible drop in quality. Above 10%, unintelligible.
- G.722: similar to G.711 but with worse PLC (the wideband side bands are harder to conceal).
- G.729: low bitrate, no FEC, sensitive to loss.
- iLBC: designed specifically for lossy networks; up to 10% loss tolerated reasonably.

**Network conditions for "good" softphone audio:**

- Latency one-way: <150ms ideal, <250ms acceptable, >400ms poor.
- Jitter: <30ms ideal, <60ms tolerable.
- Loss: <1% ideal, <3% tolerable, >5% problematic without FEC.

## Common Errors

**"Registration Failed: 401 Unauthorized"**
- Auth credentials wrong. Double-check username, password, authorization-name, realm. Recheck for trailing whitespace or caps.

**"Registration Failed: 403 Forbidden"**
- Server explicitly rejected. Could be: account disabled, IP-ACL blocking, max-registrations-reached. Check server logs.

**"Registration Failed: 408 Request Timeout"**
- Server didn't respond. Network issue: firewall blocking, wrong port, wrong transport, DNS resolving wrong IP. Try `nc -vu server.example.com 5060` to test reachability.

**"Registration Failed: 503 Service Unavailable"**
- Server overloaded or in maintenance. Retry with backoff. Persistent 503 = server-side problem.

**"No audio device found"**
- The OS exposed no devices to the softphone. Check device permissions. Linux: `pactl list short sources`. macOS: System Settings → Privacy → Microphone. Windows: Settings → Privacy → Microphone.

**"Microphone access denied"** (mobile permissions)
- iOS/Android privacy gate. Settings → App → Permissions → Microphone enable. iOS will only re-prompt if you've never explicitly denied; permanent denial requires Settings flip.

**"Codec not negotiated"**
- SDP intersection empty. Either softphone offered codecs server doesn't support, or vice-versa. Add G.711 to the offer set as fallback (G.711 is universal).

**"ICE connection failed"**
- ICE candidates exchanged but none reachable. Symmetric NAT both sides, or restrictive firewall. Solution: TURN-relay fallback (configure a TURN server with credentials).

**"Push notification not received"**
- Mobile-specific. Check: APNS/FCM registration token sent to server in REGISTER, server PNS-bridge configured, push app entitlements valid (iOS), foreground-service permission (Android), no battery-optimization restriction killing the app.

**"TLS certificate not trusted"**
- Server cert not in client trust store, or hostname mismatch, or expired. Check `openssl s_client -connect server.example.com:5061` to see the cert. Self-signed certs need explicit trust.

## Common Gotchas

**Wrong domain field — host vs SIP-domain distinction:**

```
# Broken — using server hostname as SIP domain
SIP URI: sip:alice@edge.provider.net  (hostname in domain field)
→ 403 Forbidden because account-of-record is alice@example.com

# Fixed — separate domain (logical) from server (physical)
SIP URI:    sip:alice@example.com
Server:     edge.provider.net:5061
Transport:  TLS
```

**DTMF mode mismatch (RFC 4733 vs SIP-INFO vs in-band):**

```
# Broken — softphone sends RFC 4733 telephone-event RTP
# but server expects SIP-INFO; IVR doesn't recognize digits
Softphone DTMF: RFC 4733
Server expects: SIP-INFO
Result: IVR menus don't respond to keypresses

# Fixed — match the server's expected mode
Softphone DTMF: SIP-INFO
Server expects: SIP-INFO
Result: IVR works
```

**Audio device sample-rate mismatch:**

```
# Broken — USB headset is 16 kHz capable; softphone opens at 48 kHz
# OS resamples on the fly; mic introduces aliasing
Headset native: 16 kHz
Softphone wants: 48 kHz
Result: scratchy mic audio

# Fixed — match the device's native rate or use a 48 kHz headset
Headset native: 48 kHz
Softphone wants: 48 kHz
Result: clean audio
```

**Firewall blocks outbound SIP/RTP:**

```
# Broken — corporate firewall blocks UDP 5060 + UDP 10000-20000
# REGISTER never reaches server
Firewall: blocks UDP 5060
Result: registration fails with 408 Timeout

# Fixed — switch to TLS (TCP 5061) for signaling
# RTP can flow over WebRTC media or via SBC TURN-relay on TCP 443
Transport: TLS (TCP 5061)
RTP relay: TURN over TCP 443
Result: registration succeeds, media flows
```

**Mobile push not configured — no incoming on screen-locked phone:**

```
# Broken — softphone registered to PBX directly with no push token
# Phone sleeps after 30s; OS suspends app; INVITE arrives, never delivered
Architecture: phone → SIP REGISTER → PBX
Result: incoming calls miss when phone is locked

# Fixed — register via push-aware proxy (Acrobits PNS, 3CX PNS, custom)
# X-PUSH-TOKEN header carries the device token to server
Architecture: phone → SIP REGISTER (with X-PUSH-TOKEN) → PNS proxy → PBX
Server: on INVITE, sends APNS/FCM push to wake phone
Result: phone wakes, rings, call connects
```

**VPN routing breaks SIP signaling but not media (or vice versa):**

```
# Broken — VPN routes corporate traffic; SIP is on corporate VLAN
# but RTP UDP goes to a public SBC outside the VPN
VPN route: 10.0.0.0/8 only
SIP signal: 10.1.2.3 (in VPN) → works
RTP media: 1.2.3.4 (public SBC) → goes via wrong interface, asymmetric
Result: one-way audio

# Fixed — split-tunnel correctly so both signaling and media
# follow the same path (both through VPN, or both direct)
VPN route: 10.0.0.0/8 + 1.2.3.0/24 (SBC subnet)
Result: symmetric routing, two-way audio
```

**Free-tier codec only allows G.711 → wideband server downgrades:**

```
# Broken — Zoiper Free offers only G.711μ/A
# Server prefers Opus, but G.711 is the only intersection
Softphone offers: PCMU, PCMA
Server offers: opus, G.722, PCMU, PCMA
Negotiated: PCMU (narrowband 8 kHz)
Result: HD-voice server downgraded to narrowband

# Fixed — upgrade to Zoiper Pro (or use Linphone/MicroSIP, both FOSS)
# Now the offer includes Opus and G.722
Softphone offers: opus, G.722, PCMU, PCMA
Server offers: opus, G.722, PCMU, PCMA
Negotiated: opus (super-wideband 48 kHz)
Result: HD voice
```

**Mic muted at OS level (not in app):**

```
# Broken — softphone shows unmuted, but no audio reaches RTP
# OS-level mic mute is independent of app-level mute
App mute: off
OS mute: ON (Fn key on laptop, hardware switch on headset)
Result: silence on far end

# Fixed — check OS audio panel and hardware mute button
App mute: off
OS mute: off
Hardware mute: off
Result: audio flows
```

**Softphone wants exclusive audio mode while another app holds it:**

```
# Broken — Bria configured for WASAPI exclusive
# Teams meeting open holds the device shared; Bria can't acquire
Bria mode: WASAPI exclusive
Other app: Teams (WASAPI shared)
Result: Bria can't open audio device — "Device busy"

# Fixed — switch Bria to WASAPI shared
Bria mode: WASAPI shared
Other app: Teams (WASAPI shared)
Result: both apps coexist (Windows mixer mediates)
```

**Headset Bluetooth profile (HFP narrowband vs A2DP wideband, but A2DP doesn't carry mic):**

```
# Broken — Bluetooth headset connected, audio sounds great in Spotify
# but on a SIP call, audio degrades to muffled/tinny
Bluetooth profile: A2DP (high-quality audio out, no mic input)
Softphone wants: bidirectional audio
Result: Bluetooth switches to HFP/HSP (narrowband 8 kHz both ways)
        Music quality drops, mic activates
        Mic audio is 8 kHz max regardless of softphone codec

# Fixed — accept the trade-off, or use a wired headset for HD calls
Wired headset: 48 kHz both ways
Softphone codec: Opus 48 kHz
Result: HD voice, no profile-switching glitch
```

The deeper issue: A2DP (Advanced Audio Distribution Profile) is one-way (sink-only) and high-quality (SBC/AAC/aptX/LDAC at 44.1/48 kHz). HFP (Hands-Free Profile) and HSP (Headset Profile) are bidirectional but narrowband (8 kHz CVSD or 16 kHz mSBC if both endpoints support it). When a softphone activates the mic, BT switches from A2DP to HFP, dropping audio quality drastically. mSBC ("Wideband Speech" / "HD Voice for Bluetooth") is a 16 kHz HFP variant that helps but isn't universally supported.

**Wireless headset latency causing perceived poor audio:**

```
# Broken — Bluetooth headset adds 150ms latency
# Combined with network 100ms RTT, total round-trip 250ms
# Both parties talk over each other; conversations feel laggy
Headset latency: 150ms (typical Bluetooth)
Network RTT: 100ms
Total round-trip: 250ms
Result: walkie-talkie effect, talk-overs

# Fixed — wired headset (1-2ms latency) or low-latency BT (aptX-LL, 40ms)
Headset latency: 2ms (wired)
Network RTT: 100ms
Total round-trip: 102ms
Result: natural conversation flow
```

**SRTP enforced but server doesn't support:**

```
# Broken — softphone configured "SRTP required"
# Server only supports plain RTP
Softphone: m=audio ... RTP/SAVP ... (SRTP-mandatory)
Server: m=audio ... RTP/AVP ... (plain RTP only)
Result: SDP intersection fails, no media flow

# Fixed — set "SRTP optional" or "SRTP preferred" mode
Softphone: m=audio ... RTP/SAVP and RTP/AVP both offered
Server: m=audio ... RTP/AVP
Result: falls back to plain RTP, call connects
```

## Decision Tree

**FOSS + Linux + minimal:**
- Linphone (full-featured GUI + CLI), or
- baresip (modular CLI, embedded-friendly).

**FOSS + cross-platform + GUI:**
- Linphone (Qt-based, broadest platform), or
- Jitsi Desktop (Java-based, mature, bridges to Jitsi Meet conferences).

**FOSS + decentralized + ZRTP-mandatory:**
- Jami (P2P, no central server, encrypted by default).

**Commercial + corporate Windows/Mac fleet:**
- Bria Solo (single-user subscription) for SMB, or
- Bria Enterprise + Stretto for centrally-managed large deployments.

**Lightweight Windows-only:**
- MicroSIP (PJSIP-based, sub-10MB binary, no nag).

**iOS/macOS native simple:**
- Telephone (PJSIP underneath, minimalist Apple-native UI).

**Mobile with reliable push:**
- Acrobits Softphone / Groundwire (best-in-class push), or
- Bria Mobile (Bria's iOS/Android edition with Counterpath PNS), or
- 3CX Softphone (if your PBX is 3CX).

**Tied to 3CX PBX:**
- 3CX Softphone (provisioned via 3CX Welcome Email; tightest integration).

**Browser-based (WebRTC):**
- SIP.js (modern, popular library), or
- JsSIP (alternative library, similar feature set).

**Headless CLI testing:**
- pjsua (PJSIP reference UA, scriptable), or
- baresip with a tight module set.

**Embedded Linux device (router, IoT, kiosk):**
- baresip with a custom module set (minimum modules to keep binary small).

## Idioms

**"Test call to echo service first."** Before debugging codec/network issues, dial the provider's echo-test (Twilio: `+19568724577`; FreeSWITCH default: `9196`; many providers: `*43`). Confirms full audio path works. If echo is clean, the bug is elsewhere.

**"Headset > speakerphone > laptop mic."** Wired USB headset gives best audio. Bluetooth headset is OK for casual use. Speakerphone (laptop speaker + mic) is the worst — feedback loops, AEC strain. Use it only when other options unavailable.

**"Wired Ethernet > Wi-Fi > 4G."** Wired ethernet gives consistent low jitter. Wi-Fi adds variable jitter (especially with Wi-Fi power-save mode active). 4G/5G adds variable latency and occasional bursts of loss.

**"Opus when both sides support, G.711 fallback."** Opus negotiated wins quality. G.711 always works as fallback. Disable G.722 unless you know both sides agree on the RTP timestamp interpretation.

**"RFC 4733 DTMF unless server forces in-band."** RFC 4733 is the modern standard — DTMF as RTP events. SIP-INFO is the alternative. In-band is the legacy that breaks under low-bitrate codecs (compresses the DTMF tones into garbage). Default to RFC 4733 unless server explicitly demands otherwise.

## See Also

- sip-protocol
- asterisk
- freeswitch
- ip-phone-provisioning
- sip-trunking
- yealink-phones
- polycom-phones
- cisco-phones
- grandstream-phones
- snom-phones
- mitel-phones
- audiocodes-phones
- avaya-phones

## References

- Linphone documentation — `https://www.linphone.org/technical-corner/liblinphone/documentation`
- Linphone CLI guide — `https://wiki.linphone.org/xwiki/wiki/public/view/Linphone/`
- Counterpath / Bria — `https://www.counterpath.com/` (now Alianza-owned)
- Bria Stretto admin guide — `https://docs.counterpath.com/`
- Zoiper documentation — `https://www.zoiper.com/en/support/home`
- MicroSIP — `https://www.microsip.org/`
- Jami documentation — `https://jami.net/` and `https://docs.jami.net/`
- Jitsi Desktop — `https://desktop.jitsi.org/` and `https://github.com/jitsi/jitsi`
- Jitsi Meet — `https://jitsi.org/jitsi-meet/`
- Telephone (macOS/iOS) — `https://github.com/64characters/Telephone`
- Acrobits Softphone — `https://www.acrobits.net/softphone/`
- Cloudsoftphone (Acrobits SDK) — `https://www.cloudsoftphone.com/`
- 3CX Softphone — `https://www.3cx.com/voip/voip-softphone/`
- PJSIP and pjsua — `https://www.pjsip.org/pjsua.htm`
- PJSIP API reference — `https://www.pjsip.org/docs.htm`
- baresip — `https://github.com/baresip/baresip` and `https://github.com/baresip/baresip/blob/master/README.md`
- baresip module list — `https://github.com/baresip/baresip/tree/master/modules`
- Twinkle — `http://twinkle.dolezel.info/`
- SIP.js — `https://sipjs.com/` and `https://github.com/onsip/SIP.js`
- JsSIP — `https://jssip.net/` and `https://github.com/versatica/JsSIP`
- sipML5 — `https://www.doubango.org/sipml5/`
- SIPp — `https://github.com/SIPp/sipp`
- RFC 3261 — SIP: Session Initiation Protocol
- RFC 3264 — An Offer/Answer Model with SDP
- RFC 3550 / RFC 3551 — RTP / RTP A/V Profile
- RFC 4733 — RTP Payload for DTMF Digits
- RFC 4585 — RTP/AVPF Feedback Profile
- RFC 4568 — SDES: SDP Security Descriptions
- RFC 5763 / RFC 5764 — DTLS-SRTP
- RFC 6189 — ZRTP: Media Path Key Agreement
- RFC 7118 — SIP-over-WebSocket Transport
- RFC 8866 — SDP: Session Description Protocol (current)
- Apple PushKit documentation — `https://developer.apple.com/documentation/pushkit`
- Firebase Cloud Messaging — `https://firebase.google.com/docs/cloud-messaging`
- WebRTC.org — `https://webrtc.org/`
