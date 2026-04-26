# AudioCodes IP Phones

AudioCodes 4xxHD-series desk phones, RX-series Teams Native devices, RXV video collaboration endpoints, and Mediant SBC/Gateway overlap — provisioning, SIP, codecs, TLS/SRTP, OVOC management, Direct Routing, and operational gotchas.

## Setup

AudioCodes ships an end-to-end voice stack: desk phones at the edge, Mediant SBCs and Gateways in the core, and the One Voice Operations Center (OVOC) plus Operations Platform Suite (OPS) for orchestration. The 4xxHD desk phones run a generic SIP firmware out of the box and can be re-flashed with a Microsoft Teams native firmware variant on supported models.

```
Portfolio map
├── 400HD-series IP phones (generic SIP / Skype for Business / Microsoft Teams firmware variants)
│   ├── 405HD     entry-level, monochrome 132x64, 2-line, dual-port 10/100, PoE class 1
│   ├── 420HD     mid-range, monochrome 132x64, 2-line, dual gigabit, PoE class 2
│   ├── 430HD     enhanced mid-range, color 320x240, 12 programmable keys, gigabit, PoE class 2
│   ├── 440HD     executive, color 320x240, 12 keys, gigabit, USB, PoE class 2
│   ├── 445HD     executive plus, color 480x272, Bluetooth, USB, gigabit, PoE class 3
│   ├── 450HD     premium, 5" color touch, Bluetooth, USB-A/USB-C, gigabit, PoE class 3
│   └── COSMO 470HD top-tier, large display, gigabit, advanced HD voice, PoE class 4
├── RX-series  Microsoft Teams Native dedicated endpoints
│   ├── RX50     compact desktop, Teams-native, no SIP path
│   └── RX-Pad   tablet-style controller for Teams meetings
├── RXV-series video collaboration
│   ├── RXV80    huddle-room camera/codec
│   └── RXV200   medium-room camera/codec, dual-display
├── Mediant SBC family
│   ├── Mediant 500   small branch, ~ 25 SIP sessions
│   ├── Mediant 800   mid-branch, ~ 250 sessions, voice-and-data appliance
│   ├── Mediant 1000  branch HQ, ~ 500 sessions, modular FXS/FXO/PRI
│   ├── Mediant 2600  enterprise core, ~ 600 sessions
│   ├── Mediant 4000  high-density data-center, ~ 4000 sessions
│   └── Mediant 9000  carrier-grade, ~ 16000 sessions
├── Mediant Gateways
│   ├── Mediant 1000  modular analog/digital
│   ├── Mediant 2000  digital E1/T1
│   └── Mediant 3000  high-density digital, SS7 capable
├── OVOC (One Voice Operations Center)
│   └── unified management, alarms, performance, voice-quality, mass-deployment
├── EMS (Element Management System)
│   └── legacy management for older Mediant fleets, predecessor to OVOC
└── OPI / OPS (Operations Platform Suite)
    └── multi-tenant, per-brand templated provisioning, used by service providers
```

The AudioCodes "One Voice" branding ties the device, the SBC, and the operations platform into a single ecosystem. In Microsoft-certified deployments AudioCodes is among the most-cited Direct Routing SBC vendors.

```
Where each piece fits
                    ┌─────────────────────────┐
                    │  OVOC / OPS / EMS       │
                    │  (management + provis.) │
                    └────────────┬────────────┘
                                 │ HTTPS, SNMP, syslog
        ┌────────────────────────┼────────────────────────┐
        ▼                        ▼                        ▼
   ┌────────┐              ┌──────────┐             ┌──────────┐
   │ 4xxHD  │              │ Mediant  │             │ Mediant  │
   │ phones │   SIP/SRTP   │   SBC    │  SIP/TDM    │ Gateway  │
   └────────┘─────────────►└──────────┘────────────►└──────────┘
                                 │
                                 ▼
                          PSTN / SIP trunk / Teams
```

## Default Access

Every freshly-unboxed AudioCodes phone defaults to predictable credentials. The very first thing to change after the phone provisions itself is the admin password, because the default is published in every public installation guide.

```
http://phone-ip                     web UI
admin / 1234                        administrator account, full access
user / 1234                         end-user account, restricted
sec-admin / 1234                    high-privilege security admin (newer firmware)
monitor / 1234                      read-only operator (newer firmware)
PIN 0000                            phone-side menu PIN, local LCD config
```

The admin account is also reachable as just "Admin" on some firmware versions — check the model. Phones that have OPS-provisioned themselves often have the local admin password rotated automatically.

```
# basic reachability check
ping <phone-ip>
curl -k -o /dev/null -w "%{http_code}\n" http://<phone-ip>/
```

If the phone refuses HTTP and only answers HTTPS, that is a sign that "Force HTTPS" is enabled in provisioning — the phone is then accessed at `https://<phone-ip>` and the self-signed cert needs to be trusted (or replaced).

## Web UI Tour

The AudioCodes web UI is divided into seven top-level menus that mirror the underlying configuration schema. The labels are mostly identical across the 4xxHD line, with a few extras on premium models.

```
Top-level menus
├── Status         registration state, line state, network state, system info
│   ├── Network Status      IPv4/IPv6 address, DNS, gateway, link speed/duplex
│   ├── Phone Status        firmware version, model, MAC, uptime
│   ├── Line Status         registration state per SIP account
│   └── Memory Status       free RAM, free flash
├── Configuration  the bulk of the configuration tree
│   ├── Voice               SIP, audio, codec, hold, DTMF
│   ├── Network             IP, VLAN, QoS/DSCP, 802.1x
│   ├── Phone               keys, ringer, idle screen
│   ├── Personal Settings   per-user prefs, language, time format
│   └── Security            web auth, TLS, SRTP, certificate
├── Network        IP, VLAN, DNS, time, TLS configuration
├── Provisioning   provisioning server URL, polling interval, authentication
├── Maintenance    firmware upload, config file upload/download, reboot, factory reset
│   ├── Firmware Upload     manual .img / .cmp upload
│   ├── Configuration File  upload .ini / download .ini
│   ├── Reset Configuration reset to factory defaults
│   ├── Reboot              soft reboot
│   └── Logs                syslog config, in-RAM log buffer
├── Diagnostics    PCAP capture, recording tool, ping, trace, DNS lookup
└── Logout
```

The "Status" page is the single most useful read-only view; it shows the registration state per SIP account in real-time, which is the fastest way to confirm a configuration change took effect.

## SIP Account Configuration

Each 4xxHD phone supports multiple SIP accounts, with the actual count depending on model — 405HD/420HD support 2, 430HD/440HD/445HD support 6, and 450HD/COSMO 470HD support up to 12. Each account is configured independently.

```
Configuration → Voice → SIP Account 1 ... N
├── Enable Account                 Yes / No
├── Display Name                   "Stevie Bellis"     CallerID, name shown on far-end
├── User ID                        2001                 SIP user (becomes left-side of From: URI)
├── Authentication User ID         2001                 SIP digest auth username
├── Authentication Password        ********             SIP digest auth password
├── Outbound Proxy                 sip.example.com      OR explicit IP
├── Outbound Proxy Port            5060
├── SIP Listening Port             5060                 local UDP/TCP port
├── Transport                      UDP | TCP | TLS      transport for SIP
├── Registration Period            3600                 seconds, REGISTER refresh interval
├── Re-registration on Failure     Yes / No             retry on REGISTER failure
├── SIP Realm                                            optional, for digest realm-binding
├── Server Mode                    Active / Active-Standby / Parallel
├── DNS Query Type                 A / SRV / NAPTR
└── Voicemail Number               *97                   user-dialed VM access
```

Worked example — a 450HD registered to an Asterisk PBX:

```ini
[ Account 1 ]
Enable             = Yes
DisplayName        = "Stevie Bellis"
UserId             = 2001
AuthName           = 2001
AuthPassword       = secret123
OutboundProxy      = pbx.example.com
OutboundProxyPort  = 5060
SipListenPort      = 5060
Transport          = TLS
RegInterval        = 300
ReregOnFailure     = Yes
DnsQueryType       = SRV
```

A short registration period (e.g. 60s) is useful for short NAT-binding lifetimes; 3600s is the legacy default but few NATs hold a binding that long.

## Codec Configuration

The 4xxHD phones offer a stable list of audio codecs; exactly which appear depends on model, firmware, and license.

```
Configuration → Voice → Audio Settings
├── Codec Priority      ordered list, top-to-bottom = preferred-to-fallback
│   ├── G.711 mu-law (PCMU)        always available, 64 kbps, narrowband
│   ├── G.711 A-law  (PCMA)        always available, 64 kbps, narrowband
│   ├── G.722                       wideband, 64 kbps
│   ├── G.722.1                     wideband, 24/32 kbps (Siren7)
│   ├── G.722.2 (AMR-WB)            wideband, 6.6-23.85 kbps (license)
│   ├── G.729                        narrowband, 8 kbps (license)
│   ├── iLBC                         narrowband, 13.3/15.2 kbps
│   └── Opus                         wide/super-wideband, 6-510 kbps (newer firmware)
├── Packetization Time (ptime)     20 / 30 / 40 ms
├── Silence Suppression / VAD      Yes / No
├── Comfort Noise (CN)             Yes / No
└── Echo Cancellation              Yes / No (G.168)
```

G.711 (both mu-law and A-law) is **always supported as the fallback**. If a codec negotiation collapses, a 4xxHD phone will fall back to G.711 unless explicitly disabled. Disabling G.711 entirely is virtually never the right answer — it breaks emergency-call interop with carriers that only support G.711.

```ini
[ Audio Settings ]
Codec1     = G.722
Codec2     = G.711U
Codec3     = G.711A
Codec4     = G.729
Ptime      = 20
VAD        = No
CN         = No
```

## Programmable Keys

The 4xxHD phones expose three classes of keys:

```
Programmable Soft Keys      labels at the bottom of the LCD, dynamic per-state
Function Keys / DSS keys    dedicated hardware buttons next to the LCD (BLF lamps)
Side-car keys               on optional expansion module (440HD/450HD/COSMO)
```

The supported key types:

```
Speed Dial                  dial a fixed number on press
BLF (Busy Lamp Field)       monitor another extension via SIP SUBSCRIBE/NOTIFY (dialog)
Park                        park-call on press, supervised by feature code
Pickup                      directed-pickup of another ringing extension
Conference                  invoke conference bridge
Transfer                    blind/attended transfer initiation
Hold                        local-hold toggle
DND (Do Not Disturb)        toggle DND mode
Voicemail                   call voicemail extension
Last Number Redial          redial last outbound call
Line                        bind to SIP account N
Call Pickup Group           group-pickup feature code
URL Action                  HTTP GET to a configurable URL on press
None                        unused
```

```ini
[ Programmable Keys ]
Key01_Type         = Speed Dial
Key01_Label        = "Reception"
Key01_Number       = 100
Key01_Account      = 1

Key02_Type         = BLF
Key02_Label        = "Stevie"
Key02_Number       = 2002
Key02_Account      = 1

Key03_Type         = Park
Key03_Label        = "Park 700"
Key03_Number       = 700
Key03_Account      = 1
```

## Provisioning

The phone polls a configured provisioning server for its `.ini` config and (optionally) a firmware `.img/.cmp` file. The poll happens at boot and at a periodic interval thereafter.

```
Configuration → Provisioning
├── Server URL                http(s)://prov.example.com/<MAC>.ini
├── Polling Interval          24 hours (default)
├── Authentication            None / Basic / Digest
├── Authentication User       provadmin
├── Authentication Password   ********
├── Verify Server Cert        Yes (HTTPS only)
└── Trusted Root Cert         uploaded PEM
```

Placeholder substitution applied by the phone before fetching:

```
<MAC>           the phone's MAC address, lowercase, no separators (e.g. 00908f1234ab)
<MODEL>         AudioCodes model string (e.g. 450HD)
<FIRMWARE>      currently-installed firmware version
<TIMESTAMP>     UTC seconds-since-epoch at the moment of fetch
<VENDOR>        vendor identifier
<HW>            hardware revision
```

Worked URLs:

```
http://prov.example.com/<MAC>.ini                    per-MAC config file
https://prov.example.com/<MODEL>/<MAC>.ini           per-model + per-MAC layout
https://prov.example.com/firmware/<MODEL>.img        per-model firmware blob
http://prov.example.com/cfg?mac=<MAC>&fw=<FIRMWARE>  query-string style
```

Supported transport protocols:

```
HTTP        port 80     unencrypted, fast, simplest
HTTPS       port 443    encrypted, optionally cert-validated
TFTP        port 69     legacy, no auth, no encryption — avoid in production
FTP         port 21     legacy, plaintext credentials — avoid
FTPS        port 990    FTP over TLS
```

Use HTTPS in production. TFTP is acceptable inside an entirely isolated provisioning VLAN, but the phone has no protection against an attacker injecting a malicious config.

## Configuration File Format

AudioCodes uses two file formats interchangeably:

```
.ini      INI-style text, key = value, sections in [Brackets]; human-readable;
          easiest to template and diff in Git.

.cmp      "Compiled" binary format; produced from a master .ini by the
          AudioCodes Configuration Compiler tool; smaller and faster to load,
          but opaque — model-specific and not human-editable.
```

You will encounter `.cmp` chiefly in service-provider environments where the OPS produces per-customer compiled blobs. For most enterprise self-managed fleets, plain `.ini` is preferred because it diffs well in version control and is amenable to templating with any text-based template engine.

A `.cmp` is *bound to a specific model*. Pushing a `.cmp` built for the 440HD onto a 450HD will fail to load and the phone will fall back to its previous config — see the gotcha below.

## Sample .ini

A near-minimal but realistic provisioning file for a 4xxHD phone, illustrating the four most common sections:

```ini
;====================================================================
; AudioCodes 4xxHD provisioning file
; Phone: 00908f1234ab (placeholder; will be replaced per-MAC)
;====================================================================

[ Voice ]
Account1_Enable        = Yes
Account1_DisplayName   = "Stevie Bellis"
Account1_UserID        = 2001
Account1_AuthName      = 2001
Account1_AuthPassword  = secret123
Account1_OutboundProxy = sbc.example.com
Account1_Transport     = TLS
Account1_RegInterval   = 300
Account1_DnsQueryType  = SRV

Account2_Enable        = Yes
Account2_DisplayName   = "Sales Hunt Group"
Account2_UserID        = 4000
Account2_AuthName      = 4000
Account2_AuthPassword  = secret456
Account2_OutboundProxy = sbc.example.com
Account2_Transport     = TLS

[ SIP ]
SipListenPort          = 5061
SipTLSListenPort       = 5061
RegisterRetryInterval  = 30
SessionTimerExpires    = 1800
RFC2833DTMF            = Yes
SipAlwaysSendThroughPrx = Yes

[ Audio ]
Codec1                 = G.722
Codec2                 = G.711U
Codec3                 = G.711A
Ptime                  = 20
VAD                    = No
CN                     = No
SrtpMode               = Required
SrtpAuth               = Mandatory

[ Network ]
DhcpEnable             = Yes
DnsPrimary             = 1.1.1.1
DnsSecondary           = 1.0.0.1
NtpServer              = ntp.example.com
TimeZone               = UTC+00:00
DSCP_Audio             = 46
DSCP_Signaling         = 24
VLAN_Voice             = 100
VLAN_PCPort            = 0
LLDP_MED_Enable        = Yes
```

The four sections you see — `[Voice]`, `[SIP]`, `[Audio]`, `[Network]` — cover roughly 80% of typical desk-phone configuration. Premium models add `[Phone]`, `[Bluetooth]`, `[Phonebook]`, `[Security]`, and `[ProgrammableKeys]`.

## The "OneVoice CONFIG" Service

OneVoice CONFIG (sometimes branded "Device Manager Cloud" or "RPS — Redirection and Provisioning Service") is AudioCodes' cloud-hosted redirector, equivalent in role to Yealink RPS or Polycom ZTP.

```
Workflow
1. Reseller registers each new MAC against the customer tenant in the OneVoice portal.
2. Phone leaves the factory pre-baked to phone-home to redirector.audiocodes.com.
3. End customer plugs the phone in; phone resolves DNS, contacts redirector.
4. Redirector returns the customer-tenant provisioning URL.
5. Phone fetches its real .ini from the customer's provisioning server (or OPS).
6. Phone reboots into customer config — zero-touch.
```

This is the "buy phone, register MAC, ship" pattern: the reseller does not need to pre-configure each unit, the customer does not need an on-site IT engineer, and the phone is in service the moment it has DHCP and outbound HTTPS.

The redirector is an HTTPS service; phones validate its certificate against a baked-in AudioCodes root CA. There is no way to disable that check from the field — a man-in-the-middle attacker cannot impersonate the redirector without compromising the AudioCodes PKI.

## DHCP Options

When a phone boots, it asks DHCP for a provisioning URL. AudioCodes phones consult several DHCP options in priority order:

```
Option 160      vendor-specific HTTP/HTTPS URL          PREFERRED for AudioCodes
                e.g. "https://prov.example.com/<MAC>.ini"

Option 66       TFTP/HTTP server name (legacy "tftp-server-name")
                e.g. "prov.example.com" — phone derives URL from default path

Option 43       Vendor-encapsulated options
                Sub-option 1 = provisioning URL (HTTP)
                Sub-option 2 = provisioning URL (HTTPS)

Option 6        DNS servers
Option 42       NTP servers
Option 2        Time offset (legacy; prefer NTP)
```

ISC dhcpd snippet for Option 160:

```conf
option provision-url code 160 = text;
subnet 10.10.10.0 netmask 255.255.255.0 {
    option provision-url "https://prov.example.com/<MAC>.ini";
    option routers 10.10.10.1;
    option domain-name-servers 1.1.1.1, 1.0.0.1;
}
```

Mismatched DHCP options between the phone and the DHCP server are the single most common cause of "phone boots but never provisions" tickets — see the gotcha section.

## NAT

Phones behind NAT face the same problems as any SIP endpoint: the Via/Contact headers and SDP advertise private addresses, the NAT binding times out, and incoming RTP arrives at a port the NAT no longer maps.

```
Configuration → Voice → SIP → NAT
├── NAT Mode             None | STUN | TURN | ICE | Force STUN
├── STUN Server          stun.example.com:3478
├── STUN Refresh         30 seconds
├── TURN Server          turn.example.com:3478
├── TURN User            turnuser
├── TURN Password        ********
├── ICE Enable           Yes / No
└── Keep-Alive (CRLF)    every 30 seconds, holds NAT binding
```

`Force STUN` always uses the STUN-discovered public address even when the phone thinks it has a public IP — useful in symmetric NAT scenarios where the local IP detection is misleading.

The cleanest topology behind NAT is to terminate phones on a Mediant SBC — the SBC owns both the SIP and RTP NAT-traversal logic, and the phone needs no NAT awareness at all.

```
Phone (private) ──► Mediant SBC (public) ──► ITSP / Teams
        SIP/SRTP                  SIP/SRTP
```

In this topology the phone's NAT mode can be left at `None`.

## SIP-over-TLS

```
Network → TLS Configuration
├── TLS Version             1.0 / 1.1 / 1.2 / 1.3   prefer 1.2 minimum
├── Cipher Suite            ECDHE-RSA-AES128-GCM-SHA256, ...
├── Server Cert             upload PEM
├── Server Cert Private Key upload PEM (encrypted)
├── Trusted Root CA         upload PEM (verifies the SBC/PBX server cert)
├── mTLS / Client Cert      Required / Optional      (premium models)
└── Cert OCSP / CRL Check   Yes / No
```

Premium models (445HD, 450HD, COSMO 470HD) support mutual TLS — the phone presents a client certificate that the SBC validates. This is required for some Microsoft Teams Direct Routing deployments where the SBC enforces client-cert authentication for SIP TLS.

```
# generate a CSR on the phone (web UI: Security → Certificates → Generate CSR)
# OR generate offline and upload the resulting cert + key
openssl req -new -newkey rsa:2048 -nodes -keyout phone.key -out phone.csr \
    -subj "/CN=00908f1234ab.phones.example.com"
```

Make sure the phone's clock is correct before any TLS handshake — see the "Time Sync Failed" error below.

## SRTP

```
Configuration → Voice → Audio Settings → SRTP
├── SRTP Mode               Disabled                  no SRTP, RTP only
│                           Optional                  offer SRTP, fall back to RTP
│                           Required                  refuse calls without SRTP
│                           Required + Mandatory Auth refuse without auth tag
├── Key Exchange            SDES                      keys in SDP (must be over TLS)
│                           DTLS-SRTP                 DTLS handshake on RTP path
├── Cipher Suite            AES_CM_128_HMAC_SHA1_80    default
│                           AES_CM_128_HMAC_SHA1_32    smaller auth tag
│                           AES_GCM_128 / AES_GCM_256  newer firmware
└── ZRTP                    not supported
```

If `SRTP Mode = Required`, the phone will return `488 Not Acceptable Here` on inbound calls without SRTP. This is desirable from a security standpoint but it cuts off any peer that does not negotiate SRTP — a classic source of "calls work to other phones but not to the carrier" tickets.

```ini
[ Audio Settings ]
SrtpMode    = Required
SrtpAuth    = Mandatory
SrtpCipher  = AES_CM_128_HMAC_SHA1_80
```

DTLS-SRTP is required for WebRTC interop and for some Teams Direct Routing topologies. SDES requires SIP-over-TLS to keep the SDES key out of plaintext.

## Audio Quality

DSCP marking and jitter buffer tuning are the two QoS knobs that actually move the needle on a wired-Ethernet 4xxHD deployment.

```
Configuration → Network → QoS
├── DSCP Audio              46 (Expedited Forwarding)        upstream RTP
├── DSCP Signaling          24 (Class Selector 3)            SIP signalling
└── 802.1p (VLAN priority)  5 (voice) / 3 (signaling)        when 802.1Q is enabled

Configuration → Voice → Audio Settings → Jitter Buffer
├── Type                    Static | Dynamic                  prefer Dynamic for WAN
├── Min Delay               20 ms
├── Max Delay               160 ms
├── Initial Delay           40 ms
└── Comfort Noise           Yes (with VAD)

Configuration → Voice → SIP
└── RFC 2833 DTMF           Yes      out-of-band DTMF in RTP payload (recommended)
                            No       sends in-band DTMF (audio tones, low fidelity)
```

A wrong DSCP value silently degrades call quality — the LAN happily forwards the packet but it is queued behind data traffic in any congested egress port. Verify with a packet capture.

## Bluetooth

Bluetooth is supported only on premium models (445HD, 450HD, COSMO 470HD) and on certain RX-series devices. The supported Bluetooth profiles are:

```
HFP   Hands-Free Profile        connects to wireless headsets
HSP   Headset Profile           legacy, narrowband
A2DP  Advanced Audio Profile    music streaming (RX-series only)
PBAP  Phone Book Access         contact sync from a paired mobile
```

```
Phone-side menu:
Menu → Settings → Bluetooth → Pair Device → ...

Web UI:
Configuration → Phone → Bluetooth
├── Enable               Yes / No
├── Discoverable         Yes / No
├── Auto-Connect         Yes / No
└── Paired Devices       list of MACs, with profile flags
```

Note that Bluetooth headset audio uses a narrowband CVSD/SBC codec by default, which downgrades a wideband G.722 RTP call. Higher-fidelity LC3/aptX support depends on firmware and headset.

## Phone-side Phonebook

Phonebook directory entries can be local (stored on the phone) or fetched from an LDAP server.

```
Configuration → Phone → Phonebook
├── Local Phonebook        per-phone storage, ~ 1000 entries
├── LDAP Phonebook
│   ├── Enable              Yes / No
│   ├── LDAP Server         ldap.example.com
│   ├── Port                389 (LDAP) | 636 (LDAPS)
│   ├── Use TLS             Yes / No
│   ├── Bind DN             cn=phonebook,ou=service,dc=example,dc=com
│   ├── Bind Password       ********
│   ├── Base DN             ou=people,dc=example,dc=com
│   ├── Search Filter       (&(objectClass=person)(|(cn=%s*)(sn=%s*)(telephoneNumber=%s*)))
│   ├── Name Attributes     cn,sn,givenName
│   ├── Number Attributes   telephoneNumber,mobile,ipPhone
│   └── Result Limit        50
└── Per-Line Phonebook      account-bound directory (per SIP account)
```

The LDAP search filter has to be properly XML-escaped if delivered via OPS templates — see the gotcha below about `&` vs `&amp;`.

## PoE

All 4xxHD phones are 802.3af-compliant. Premium models negotiate 802.3at (PoE+) for higher peak draw. Class advertisement matters because PoE switches budget power per-port based on advertised class.

```
Class    Power-out (W)     Phones in class
0        0.44–12.95        405HD, 420HD (legacy default)
1        0.44–3.84         (rare)
2        3.84–6.49         430HD, 440HD, 445HD
3        6.49–12.95        450HD, COSMO 470HD
4        12.95–25.5        COSMO 470HD with sidecar / extension module
```

If a PoE switch is set to a per-port class lower than the phone's actual class, the phone may boot, fail to bring up its display, or reboot mid-call. Set the switch port to either "auto" or the explicit class advertised by the phone.

```
# Cisco IOS — let auto class detection do its job
interface GigabitEthernet1/0/1
 power inline auto
 power inline port priority high
```

## Logging

```
Maintenance → Logs / Configuration → Logs
├── Syslog Enable          Yes / No
├── Syslog Server          syslog.example.com
├── Syslog Port            514 (UDP) | 6514 (TLS)
├── Syslog Severity        CRITICAL | ERROR | WARN | INFO | DEBUG
├── In-RAM Log Buffer      ~ 256 KB ring buffer, downloadable
└── Recording Tool         on-phone trace, capture SIP and audio events
```

Recommended baseline:

```ini
[ Logs ]
SyslogEnable      = Yes
SyslogServer      = syslog.example.com
SyslogPort        = 514
SyslogSeverity    = INFO
LogToRamEnable    = Yes
```

`DEBUG` is fine for troubleshooting but produces enough volume that a fleet of >100 phones at DEBUG can saturate a small syslog collector. Drop to `INFO` once an issue is resolved.

The "Recording Tool" is an in-phone troubleshooting facility that captures registration, call setup, and audio events into a downloadable archive — useful for support tickets where you cannot attach a full PCAP.

## PCAP

```
Maintenance → Diagnostic Recording → Capture Tool
├── Interface              LAN / WAN / Both
├── Filter                 BPF expression, e.g. "host 10.0.0.1 and port 5060"
├── Max File Size          5 MB (default)
├── Start                  begin capture
├── Stop                   end capture
└── Download               .pcap file via web UI
```

The capture is a real libpcap-format `.pcap` and opens directly in Wireshark. For SIP-over-TLS captures you also need the TLS pre-master-secret log (Wireshark's `(Pre)-Master-Secret log filename` setting) — AudioCodes phones do not export TLS keys, so SIP-over-TLS payload is unreadable except via SBC-side decryption.

```
# typical filter expressions
host 10.0.0.1                        all traffic to/from PBX
port 5060                            SIP traffic only
port 5061                            SIP-over-TLS only
udp portrange 16384-32767            RTP/SRTP
host 10.0.0.1 and not port 22        exclude SSH noise
```

## Time / NTP

The phone needs accurate clock for TLS, certificate validity windows, syslog timestamps, and SIP timestamps.

```
Network → Time Settings
├── NTP Server                     ntp.example.com (primary)
├── NTP Server 2                   pool.ntp.org   (backup)
├── NTP Update Interval            60 minutes
├── Time Zone                      UTC+00:00 / America/New_York / Asia/Tokyo / ...
├── Daylight Saving Time           Auto / Manual / Off
├── DST Start                      e.g. "Second Sunday of March, 02:00"
├── DST End                        e.g. "First Sunday of November, 02:00"
└── DST Offset                     +60 minutes
```

On clock drift, the most-visible symptom is "TLS Handshake Failed" — the phone presents the wrong system time and the server cert is "not yet valid" or "expired."

## Hot Desking

Hot desking lets a user log in to any phone in the pool with their personal SIP credentials. The phone clears the previous user's account, registers as the new user, and (optionally) pulls down a per-user provisioning fragment.

```
Configuration → Phone → Hot Desking
├── Enable                  Yes / No
├── Login Method            PIN | Username + Password
├── Idle Logout Timeout     30 minutes
├── Cache User Credentials  Yes / No   keep recent logins for fast re-login
└── Per-User Provisioning   URL with <USERID> placeholder
```

A typical kiosk-style desk:

```
1. Phone idle, generic shell registration (no SIP account).
2. User taps a soft-key, enters PIN.
3. Phone fetches https://prov.example.com/users/<USERID>.ini
4. Phone reloads the SIP account section, registers.
5. User uses phone normally.
6. User logs out (or 30-min idle), phone reverts to the generic shell.
```

## ACD Mode

ACD (Automatic Call Distribution) integrates a 4xxHD phone with a contact-center backend so the agent's status (Available / On-Call / Wrap-Up / Away) is signalled via SIP NOTIFY events.

```
Configuration → Voice → ACD
├── Enable                 Yes / No
├── Mode                   Manual | Auto-On-Login | Auto-On-Off-Hook
├── Wrap-Up Timer          30 seconds      auto-leave wrap-up after N seconds
├── States                 Available / Unavailable / Wrap-Up
├── Reason Codes           list, e.g. "Lunch", "Training", "Break"
└── Server                 ACD server, typically the PBX or a contact-center host
```

Soft-key bindings appear automatically when ACD is enabled — the bottom of the LCD shows `Login`, `Logout`, `Available`, and `Wrap-Up` while the agent is signed in.

## MS Teams Direct Routing

AudioCodes is the canonical Direct Routing SBC vendor and appears at the top of Microsoft's certified-vendor list. The 4xxHD also has a Microsoft Teams native firmware variant (Teams "compatible" devices), and the RX-series is fully Teams-native.

```
Direct Routing topology
                                          ┌────────────────┐
                                          │  MS Teams M365 │
                                          └────────┬───────┘
                                                   │ SIP-TLS, SRTP
                                          ┌────────┴───────┐
                                          │  Mediant SBC   │ ← AudioCodes
                                          └────────┬───────┘
                                                   │ SIP / SIP-TLS
                                          ┌────────┴───────┐
                                          │   PSTN / ITSP  │
                                          └────────────────┘

Endpoints
─────────
Teams clients (PC / mobile)         use Teams APIs, no SIP
RX-series Teams Native phones       Teams-firmware, Microsoft cloud auth, no SIP path
4xxHD with Teams firmware           "Teams compatible" path, Microsoft sign-in
4xxHD with Generic SIP firmware     register to PBX (Asterisk/FreeSWITCH/etc), SIP
```

Direct Routing requires:

- A public DNS name with a public TLS cert on the SBC.
- Trunk authentication via FQDN (no IP-based auth in Teams Direct Routing).
- TLS 1.2+, SRTP, and a specific cipher set published by Microsoft.
- The SBC FQDN added to the Teams admin centre under "Direct Routing → SBCs."
- A voice routing policy in Teams that maps user numbers to that SBC.

Day-to-day, AudioCodes ships a "Direct Routing Configuration Wizard" in the Mediant web UI that fills in the conformant defaults; deviating from the wizard requires careful reading of the AudioCodes Configuration Note for Microsoft Teams Direct Routing.

## Mediant SBC Brief

The Mediant SBC family ranges from a small-branch appliance (Mediant 500) to a carrier-grade chassis (Mediant 9000). They share a common configuration schema:

```
Mediant 500     ~ 25 SIP sessions, ~ 25 transcoded         small branch
Mediant 1000    ~ 500 sessions, modular FXS/FXO/PRI         branch HQ
Mediant 4000    ~ 4000 sessions                              data centre
Mediant 9000    ~ 16000 sessions, carrier-grade              service provider
```

Configuration is driven from a similar INI-style file (often loaded via the SBC's own web UI) plus a graphical "SIP Routing" matrix:

```ini
[ IP Group #0 ]
Name                  = ITSP
ProxyName             = sip.itsp.example.com
SIPGroupName          = itsp
ContactUser           = +12025550100

[ IP Group #1 ]
Name                  = Teams
ProxyName             = sip.pstnhub.microsoft.com
SIPGroupName          = teams
TLSContext            = teams-tls

[ Routing Rule #0 ]
Name        = "Teams to ITSP"
SrcIPGroup  = 1
DstIPGroup  = 0
Priority    = 100

[ Routing Rule #1 ]
Name        = "ITSP to Teams"
SrcIPGroup  = 0
DstIPGroup  = 1
Priority    = 100
```

Common SBC features:

```
SIP Routing rules           IP-Group → IP-Group with priority
Manipulation rules          rewrite headers, From/To, P-Asserted-Identity
Number normalisation        E.164 conversion, country-code prefixing
Transcoding                 codec interworking (G.722 ↔ G.711 ↔ Opus, ...)
Mediation                   T.38 fax, RFC 2833 DTMF transcoding
Encryption                  SIP-TLS termination, SRTP termination, mTLS
Topology hiding             rewrite Via, Contact, Record-Route headers
```

For the typical enterprise Direct Routing build, the SBC is the *only* component on the public Internet — the desk phones, PBX, and PSTN trunk all sit behind it.

## Mediant Gateway Brief

Mediant Gateways convert between TDM (analog or digital telephony) and SIP. They appear in three flavours:

```
Mediant 1000          modular: FXS, FXO, BRI, PRI cards          branch
Mediant 2000          digital E1/T1 PRI / SS7                    medium
Mediant 3000          high-density E1/T1, SS7, R2                carrier
```

The most common deployment:

```
PSTN PRI / E1  ──(TDM)──►  Mediant 2000  ──(SIP)──►  PBX / SBC
Analog (FXO)   ──────────►  Mediant 1000  ──────────►  PBX
Analog (FXS)   ──────────►  Mediant 1000  ──────────►  Analog endpoints (fax, paging)
```

Configuration concepts:

```
Trunk                 a TDM port (E1/T1)
Channel               a single 64 kbps DS0 inside a trunk
Hunt Group            a logical pool of trunks/channels
Trunk Group           a set of hunt groups (for routing)
ISDN signalling       Q.931 + Q.921 (PRI), or DSS1 (BRI)
SS7                   ISUP / TUP for carrier interconnect
```

Mediant Gateways speak the same SIP and the same INI schema as the Mediant SBC; in fact, the Mediant 1000 and Mediant 4000 chassis can be licensed as Gateway-only, SBC-only, or both at once.

## Web UI Authentication

```
admin         full read/write configuration access
sec-admin     security-only: certs, TLS, SRTP, web auth
monitor       read-only operational view (stats, alarms, status)
user          end-user limited menu (programmable keys, ringer)
```

Each role has its own password. After OPS deploys, the `admin` password is typically rotated to a per-MAC value tracked in the OPS inventory. The `monitor` account is useful for handing read-only access to a NOC team without giving them the keys to reconfigure the fleet.

```
Configuration → Security → Web Access
├── HTTP Enable           Yes / No
├── HTTPS Enable          Yes / No
├── HTTPS Cert            self-signed | uploaded
├── Brute-Force Lockout   5 failed attempts → 60 second lockout
└── Account Inactivity    auto-logout after 10 minutes
```

## Encryption

AudioCodes supports vendor-specific AES encryption of the `.ini` and `.cmp` provisioning blobs. This is distinct from HTTPS transport encryption — the file itself is encrypted at rest and in transit.

```
Provisioning encryption modes
├── None                   plaintext .ini
├── AES-128-CBC            symmetric, with a per-fleet shared key
├── Per-MAC Key            unique key per phone, derived from the MAC + a master secret
└── AES-256-GCM            newer firmware, authenticated encryption
```

The per-MAC mode is the cleanest: even if the provisioning server is breached and the entire `.ini` archive leaks, an attacker cannot decrypt any single file without the master secret.

```
# pseudo-recipe (actual tool: AudioCodes Configuration Encryption Tool)
$ ac_encrypt --mode per-mac --master-key master.key 00908f1234ab.ini
→ writes 00908f1234ab.ini.enc
```

The encrypted file is then served over HTTPS; the phone holds (in flash) the master key derivation parameters that let it decrypt the per-MAC blob on download.

## Multi-Tenant Provisioning

The Operations Platform Suite (OPS) supports tenant-isolated configuration:

```
Tenant Foo
├── Brand:       "Foo Communications"
├── Logo:        foo-logo.bmp                   shown on idle screen
├── Wallpaper:   foo-bg.jpg
├── Idle Text:   "Foo Communications — please dial 0 for reception"
├── Key Layout:  per-tenant programmable key template
├── Codec Plan:  G.722 + G.711U
└── Provisioning host:  prov.foo.example.com

Tenant Bar
├── Brand:       "Bar Holdings Plc"
├── Logo:        bar-logo.bmp
├── ...
```

OPS uses Jinja-like template variables in its master `.ini` templates:

```
DisplayName = "{{ user.first_name }} {{ user.last_name }}"
UserID      = {{ user.extension }}
AuthName    = {{ user.extension }}
AuthPassword = {{ user.sip_password | encrypt(per_mac_key) }}
OutboundProxy = {{ tenant.sbc_fqdn }}
```

A misrendered template — for instance an unescaped `&` — yields a phone that fails to load its config and falls back to defaults; see the gotcha section.

## Common Errors

The exact error strings the phone surfaces, and the canonical fix for each.

```
Error: "REGISTER failed: 401 Unauthorized"
Cause: SIP digest auth credentials wrong (User ID, Auth Name, Password mismatch with PBX).
Fix:   Verify Authentication User ID and Authentication Password match the PBX user record.
       Re-enter the password (does not show in UI for verification — re-type fresh).

Error: "REGISTER failed: 403 Forbidden"
Cause: PBX rejects the registration despite valid auth — typically IP-based ACL,
       extension out of license, or per-extension toggle disabled.
Fix:   Check the PBX-side allowlist and the user's "register from any IP" setting.
       For Asterisk: confirm `pjsip.conf` `endpoint` and `aor` permit the source IP.

Error: "Provisioning: HTTP 401"
Cause: Provisioning server demands HTTP Basic/Digest auth, phone has no creds.
Fix:   Configuration → Provisioning → Authentication User / Password.
       Also check if the URL needs to switch to HTTPS.

Error: "Cert Verification Failed"
Cause: Server cert does not chain to a CA the phone trusts, OR the Common Name /
       SAN does not match the URL hostname.
Fix:   Upload the issuing CA root to Network → TLS → Trusted Root CA.
       Reissue the server cert with a SAN that matches the FQDN the phone uses.

Error: "DNS Resolution Failed"
Cause: Phone's DHCP-assigned DNS server cannot resolve the SBC/PBX/provisioning FQDN.
Fix:   Network → DNS → Primary / Secondary, set explicit servers (1.1.1.1, 8.8.8.8).
       Verify with phone-side menu → Diagnostics → DNS Lookup.

Error: "Codec Negotiation Failed"
Cause: SDP offer/answer collapse — no codec common to phone and far-end.
Fix:   Configuration → Voice → Audio Settings → enable G.711U + G.711A as bottom
       of the codec priority list. They are the universal fallback.

Error: "Time Sync Failed"
Cause: NTP server unreachable, blocked by firewall, or wrong FQDN.
Fix:   Network → Time Settings → check NTP Server reachability (ICMP, port 123/UDP).
       If outbound NTP is blocked, use an internal NTP source.

Error: "PoE: Insufficient Power"
Cause: Switch port budget below the phone's required class.
Fix:   Bump the switch port to "auto" PoE class detection, or set explicitly to
       class 3 (PoE+) for 450HD/COSMO. Some switches need `power inline auto` then
       a port shut/no-shut to renegotiate.

Error: "TLS Handshake Failed"
Cause: Multi-source: clock skew, mismatched cipher suites, expired cert, or
       hard-disabled TLS version.
Fix:   1. Confirm phone clock (Network → Time Settings).
       2. Set TLS Version to "TLS 1.2 or higher" (avoid TLS 1.0/1.1).
       3. Verify server cert SAN/CN matches the URL.
       4. Check that the issuing CA is in the phone's Trusted Root list.

Error: "Firmware Mismatch"
Cause: A pushed .cmp / .img is built for a different model than the phone.
Fix:   Confirm the file is for the exact phone model. AudioCodes ships
       per-model firmware images — a 440HD build will not load on a 450HD.
       Re-upload the correct image, or fall back to .ini-based provisioning.
```

## Common Gotchas

Twelve broken→fixed pairs covering the failures that show up in real deployments.

### 1. Default 1234 password not changed

```
Broken
─────
admin / 1234        still active months after deployment.
```
Anyone with reachability to the management VLAN can log in, dump the SIP password from the config, and impersonate the user.

```
Fixed
─────
Configuration → Security → Web Access → admin → New Password: <strong>
And rotate it via OPS so every phone gets a unique per-MAC password.
```

### 2. HTTP not HTTPS in provisioning URL

```
Broken
─────
ProvisioningURL = http://prov.example.com/<MAC>.ini

The .ini is sent in plain over the wire. SIP password leaks to anyone
on-path.
```

```
Fixed
─────
ProvisioningURL = https://prov.example.com/<MAC>.ini
And install the issuing CA root on the phone for cert validation.
```

### 3. Multiple SIP accounts but DTMF mode different per account

```
Broken
─────
Account1.RFC2833DTMF = Yes
Account2.RFC2833DTMF = No

Account 1 uses RTP-event DTMF, Account 2 uses in-band DTMF.
IVR digit-collection works on Account 1, fails on Account 2.
```

```
Fixed
─────
Set RFC2833DTMF = Yes globally (in [SIP] section), or apply
identical DTMF settings to every account that talks to the same PBX.
```

### 4. SRTP required but server doesn't support

```
Broken
─────
SrtpMode = Required
PBX is plain RTP only.

Phone returns 488 Not Acceptable Here on every call.
```

```
Fixed
─────
SrtpMode = Optional         (negotiate, fall back to RTP)
or upgrade the PBX to SRTP, or terminate the phone on a Mediant SBC
that mediates SRTP-to-RTP for you.
```

### 5. DSCP marking wrong → degraded QoS

```
Broken
─────
DSCP_Audio = 0          (best-effort)
DSCP_Signaling = 0

Calls drop bits during peak data transfer through the same WAN link.
```

```
Fixed
─────
DSCP_Audio = 46         (EF, Expedited Forwarding)
DSCP_Signaling = 24     (CS3)
And confirm the LAN/WAN actually honours those DSCP values — the
markings are inert if the routers are configured to bleach them.
```

### 6. DHCP option mismatch (160 vs 66)

```
Broken
─────
DHCP server hands out option 66 (TFTP) only.
Phones expect option 160 (preferred).

Phones boot with no provisioning URL → use stale local config.
```

```
Fixed
─────
Set both option 160 and option 66 to the same URL. Phones will pick 160.
```

### 7. LDAP credentials wrong → no phonebook

```
Broken
─────
LDAP Bind DN:  cn=phonebook,ou=service,dc=example,dc=com
LDAP Bind Pwd: (typo)

Phonebook search returns 0 results, no error message on the LCD.
```

```
Fixed
─────
Test the bind from a workstation:
  ldapsearch -x -H ldap://ldap.example.com:389 \
    -D "cn=phonebook,ou=service,dc=example,dc=com" \
    -w '<password>' \
    -b "ou=people,dc=example,dc=com" "(cn=stevie*)"
If that works, copy the exact DN and password into the phone.
```

### 8. PoE class wrong → power negotiation fails

```
Broken
─────
Switch port forced to class 2 (max 6.49 W).
450HD advertises class 3 (max 12.95 W).

Phone reboots randomly when the LCD backlight is on full.
```

```
Fixed
─────
Cisco IOS:
  interface GigabitEthernet1/0/12
    power inline auto
And shut/no shut to renegotiate.
```

### 9. .CMP binary not for this model

```
Broken
─────
$ acconfigtool --build --model 440HD config.ini → 440HD.cmp
$ scp 440HD.cmp prov.example.com:/srv/prov/<MAC>.cmp   # 450HD's MAC

Phone fetches the .cmp, fails to parse, falls back to factory defaults,
nothing registers.
```

```
Fixed
─────
Either build a 450HD-specific .cmp, or stick to .ini files which are
model-portable (within reason — premium-only sections like [Bluetooth]
will be ignored on entry-level phones, not error out).
```

### 10. OPS template variable substitution wrong

```
Broken
─────
Template:  AuthPassword = {{ user.password }}
Rendered:  AuthPassword = {{ user.password }}     (literally, unrendered)

The OPS template engine never expanded the variable; the phone tries
to register with the literal string "{{ user.password }}" as its password.
```

```
Fixed
─────
1. Confirm the OPS template engine is enabled for that file.
2. Confirm the variable name matches the OPS user-record schema.
3. Render the template to a test file and inspect before deploying:
     ops render --tenant=foo --user=2001 user-template.ini > /tmp/out.ini
```

### 11. Microsoft Teams firmware vs Generic SIP firmware

```
Broken
─────
Phone bought as Teams-firmware (RX-series or 450HD-Teams).
Provisioning URL points at the generic SIP .ini — phone ignores it,
prompts for Teams sign-in instead.
```

```
Fixed
─────
Decide deployment model first:
  - Teams Native: provision via Teams admin centre, no SIP.
  - Generic SIP:  flash the phone with the generic SIP firmware (.cmp from
                   AudioCodes downloads area for that model + variant).
```

### 12. LDAP search filter syntax (`&` vs `&amp;`) escaping

```
Broken
─────
Inside an OPS XML/HTML template:
  <SearchFilter>(&(objectClass=person)(cn=%s*))</SearchFilter>

The `&` is interpreted by the XML parser as start-of-entity, breaks
the whole filter, the phone's LDAP tab silently fails.
```

```
Fixed
─────
<SearchFilter>(&amp;(objectClass=person)(cn=%s*))</SearchFilter>

The `&amp;` decodes back to a single `&` before being used by the
phone's LDAP client, which sees the correctly-formed
"(&(objectClass=person)(cn=*))" filter.
```

## Diagnostic Tools

```
Web UI Diagnostic Recording      .pcap capture from the phone itself
Syslog forwarding                continuous log stream to a central collector
In-Phone Recording Tool          SIP-and-audio event capture, downloadable
Phone-side menu Diagnostics      ping, traceroute, DNS lookup, network info
Mediant CLI                      SBC/Gateway side, full IOS-like CLI
OVOC dashboard                   fleet view, alarms, SLA, voice quality,
                                  per-call detail records (CDR), SBC stats
```

The OVOC dashboard pulls SNMP, syslog, and HTTP REST metrics from every device under management and rolls them up into a single SLA view. For a fleet of >100 phones this is essential — the per-phone web UIs do not scale.

```
# Mediant CLI quick reference (over SSH)
ssh admin@sbc.example.com
> enable
# show running-config
# show registered-users
# show calls active
# show ip-group
# show routing-rule
# debug sip on
# debug syslog level debug
```

## Sample Cookbook

Four end-to-end integration recipes.

### Asterisk integration (4xxHD as PJSIP endpoint)

`pjsip.conf` (Asterisk side):

```ini
[2001]
type=endpoint
context=internal
disallow=all
allow=g722
allow=ulaw
auth=2001-auth
aors=2001-aor
direct_media=no
rtp_symmetric=yes
rewrite_contact=yes
force_rport=yes
trust_id_inbound=yes
media_encryption=sdes
media_encryption_optimistic=yes

[2001-auth]
type=auth
auth_type=userpass
username=2001
password=secret123

[2001-aor]
type=aor
max_contacts=2
qualify_frequency=60
remove_existing=yes
```

Phone side (`.ini`):

```ini
Account1_Enable        = Yes
Account1_UserID        = 2001
Account1_AuthName      = 2001
Account1_AuthPassword  = secret123
Account1_OutboundProxy = asterisk.example.com
Account1_Transport     = TLS
Account1_RegInterval   = 60
Codec1                 = G.722
Codec2                 = G.711U
SrtpMode               = Optional
```

### FreeSWITCH integration

`directory/default/2001.xml` (FreeSWITCH):

```xml
<include>
  <user id="2001">
    <params>
      <param name="password" value="secret123"/>
    </params>
    <variables>
      <variable name="user_context" value="default"/>
      <variable name="effective_caller_id_name" value="Stevie Bellis"/>
      <variable name="effective_caller_id_number" value="2001"/>
    </variables>
  </user>
</include>
```

Phone side (same as Asterisk above, just point `OutboundProxy` at FreeSWITCH).

### Microsoft Teams Direct Routing (Mediant SBC + 4xxHD endpoints)

Conceptual flow:

```
4xxHD phones (LAN, SIP-TLS, SRTP)
        ↓
Mediant SBC (DMZ, public FQDN sbc.example.com, public TLS cert)
        ↓
Microsoft Teams sip.pstnhub.microsoft.com / sip2.pstnhub.microsoft.com
                  / sip3.pstnhub.microsoft.com
```

Mediant SBC `.ini` highlights:

```ini
[ TLS Context #1 ]
Name           = teams-tls
TLSVersion     = TLSv1.2
PrivateKeyName = sbc-key.pem
CertificateName = sbc-cert.pem
TrustedRoot    = baltimore-ca.pem,digicert-ca.pem

[ Proxy Set #1 ]
Name              = teams-proxy
ProxyAddress      = sip.pstnhub.microsoft.com
ProxyAddress2     = sip2.pstnhub.microsoft.com
ProxyAddress3     = sip3.pstnhub.microsoft.com
TransportType     = TLS
TLSContext        = teams-tls

[ IP Group #1 ]
Name              = teams
ProxySetId        = 1
ContactUser       = +12025550100
SIPGroupName      = teams
```

PowerShell (Teams admin side):

```powershell
New-CsOnlinePSTNGateway -Identity "sbc.example.com" `
    -Enabled $true -SipSignalingPort 5061 -ForwardCallHistory $true `
    -ForwardPai $true -MediaBypass $false

New-CsOnlineVoiceRoute -Identity "AllPSTN" `
    -NumberPattern "^\+\d+" -OnlinePstnGatewayList "sbc.example.com"
```

### SIP trunk to ITSP

```
Branch site: 4xxHD phones (Account1) → Mediant 500 SBC → ITSP SIP trunk

Phone side: register to Mediant 500 (SIP-TLS, SRTP)
Mediant 500: forward registered users out to the ITSP via a SIP trunk
            with E.164 manipulation:
              From-User on egress: rewrite to a +1XXXYYYNNNN E.164 number
              To-User on ingress:  strip leading + and country code, route
                                   to internal extension 2001..2099
```

Mediant 500 `.ini` (snippet):

```ini
[ IP Group #2 ]
Name        = itsp
ProxyName   = sip.itsp.example.com
SIPGroupName = itsp

[ Routing Rule #5 ]
Name        = "Internal to ITSP"
SrcIPGroup  = 0    ; phones
DstIPGroup  = 2    ; ITSP

[ Number Manipulation #5 ]
Name        = "Add E.164 Prefix"
Direction   = Egress
SourceIPGroup = 0
DestPrefix    = 9
ManipulatedDestPrefix = +1
```

## Hardware Specifics

```
405HD       monochrome, 132x64, 2-line, 10/100 dual-port, no USB, no Bluetooth, PoE class 1
420HD       monochrome, 132x64, 2-line, gigabit dual-port, no USB, no Bluetooth, PoE class 2
430HD       color 320x240, 12 keys, gigabit, no USB, no BT, PoE class 2
440HD       color 320x240, 12 keys, gigabit, USB-A, no BT, PoE class 2
445HD       color 480x272, 12 keys, gigabit, USB-A + Bluetooth, PoE class 3
450HD       color 5" touch, gigabit, USB-A + USB-C, Bluetooth, PoE class 3
COSMO 470HD top-tier, large color display, gigabit, USB, Bluetooth, PoE class 4

All 4xxHD: PoE 802.3af min, 802.3at on premium models, gigabit on 420HD+.
Sidecar / expansion module: 440HD, 450HD, COSMO — adds 24 BLF/DSS keys per panel.
```

The COSMO 470HD is positioned as the executive top-tier and ships with a wider screen and a dedicated programmable side-key panel suited to receptionist consoles.

## RX-series Teams Native

```
RX50        compact desktop Teams phone, no SIP path, sign in with M365 creds
RX-Pad      tablet-style controller for Teams Rooms
RXV80       huddle-room camera + codec, Teams Rooms-on-Android base
RXV200      medium-room camera + codec, dual-display, Teams Rooms
```

These devices boot Teams firmware natively; there is no `[Voice]` section, no SIP account UI, and no provisioning `.ini`. They are managed entirely from the Microsoft Teams admin centre via the same APIs as the Microsoft-branded Teams Rooms hardware.

If a customer wants both Teams Native phones and a generic-SIP PBX, the result is two parallel deployments: Teams users have RX-series, SIP users have generic-SIP-firmware 4xxHD, and a Mediant SBC bridges Teams ↔ PSTN.

## Idioms

```
"use HTTPS for provisioning"             never plain HTTP for production fleets
"OPS for fleet management"               OPS scales where per-phone web UIs do not
"Mediant SBC for SIP-trunk termination"  the SBC is the single public-Internet
                                          face; phones stay private
"Direct Routing requires AudioCodes-     Microsoft maintains a vendor cert list;
 certified setup"                         AudioCodes is on it, but the deployment
                                          must follow the cert'd config
"OneVoice CONFIG for zero-touch"         redirector eliminates per-unit
                                          pre-provisioning at the reseller
"SRTP Optional, not Required, until      avoid call-blocking surprises until
 the whole stack supports it"             every leg of every route does SRTP
"G.711 always at the bottom of the       universal fallback codec; do not
 codec list"                              disable
"DSCP 46 for audio, 24 for signalling"   industry-standard markings, what
                                          most managed LANs/WANs are tuned for
"NTP first, TLS later"                   no TLS handshake works without
                                          accurate clock
```

## See Also

- ip-phone-provisioning
- sip-protocol
- asterisk
- freeswitch
- yealink-phones
- polycom-phones
- cisco-phones
- grandstream-phones
- snom-phones
- mitel-phones

## References

- AudioCodes — official support portal: https://www.audiocodes.com/services-support
- AudioCodes 4xxHD User Guide (per-model: 405HD, 420HD, 430HD, 440HD, 445HD, 450HD, COSMO 470HD)
- AudioCodes 4xxHD Administrator's Manual
- AudioCodes Mediant SBC User Manual (Mediant 500 / 800 / 1000 / 2600 / 4000 / 9000)
- AudioCodes Mediant SBC Configuration Guide
- AudioCodes Mediant Gateway User Manual (Mediant 1000 / 2000 / 3000)
- AudioCodes One Voice Operations Center (OVOC) documentation
- AudioCodes Operations Platform Suite (OPS) documentation
- AudioCodes Element Management System (EMS) legacy documentation
- AudioCodes Configuration Note for Microsoft Teams Direct Routing
- AudioCodes Device Manager Cloud / OneVoice CONFIG (Redirection and Provisioning Service) documentation
- Microsoft Teams Direct Routing — Plan Direct Routing: https://learn.microsoft.com/microsoftteams/direct-routing-plan
- Microsoft Teams Direct Routing — Certified SBC list: https://learn.microsoft.com/microsoftteams/direct-routing-border-controllers
- RFC 3261 — SIP: Session Initiation Protocol
- RFC 3711 — The Secure Real-time Transport Protocol (SRTP)
- RFC 4568 — Session Description Protocol (SDP) Security Descriptions for Media Streams (SDES)
- RFC 5763 — Framework for Establishing a Secure Real-time Transport Protocol (DTLS-SRTP)
- RFC 5764 — DTLS Extension to Establish Keys for SRTP
- RFC 2833 — RTP Payload for DTMF Digits, Telephony Tones and Telephony Signals (and successor RFC 4733)
- RFC 3550 — RTP: A Transport Protocol for Real-Time Applications
- RFC 7826 — Real-Time Streaming Protocol (RTSP) v2 (background; not used by phones directly)
- IEEE 802.3af / 802.3at — Power over Ethernet standards
- IEEE 802.1Q — VLAN tagging
- IEEE 802.1p — Priority Code Point (LAN QoS)
- IETF DSCP — RFC 2474, RFC 3168 (DiffServ markings)
- ITU-T G.711, G.722, G.722.1, G.722.2 (AMR-WB), G.729, G.168 (echo cancellation)
- IETF Opus — RFC 6716
- LDAP — RFC 4511, RFC 4515 (LDAP search filters)
- LLDP-MED — ANSI/TIA-1057 (voice VLAN auto-discovery)
- DHCP options — RFC 2132 (option 66), RFC 3925 (vendor-encap option 43), AudioCodes-specific option 160
