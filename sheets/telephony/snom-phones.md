# Snom IP Phones

German-engineered SIP phones — terse reference for D-series desk, M-series DECT, C-series conference, PA1 paging.

## Setup — Snom Portfolio

Snom AG (Berlin, Germany) — founded 1996. Acquired by VTech Holdings (Hong Kong) in 2016 — separate brand from Yealink (Yealink is independent Chinese vendor; sometimes confused). VTech-OEM history: VTech also owns Snom and Gigaset DECT business; Snom remains "German-engineered" branding under VTech parent.

### Series matrix

| Series | Era | Examples | Notes |
|--------|-----|----------|-------|
| 3xx | Legacy 2005-2012 | 320, 360, 370 | Discontinued; firmware 8.7 final |
| 7xx | 2012-2016 | 710, 715, 720, 725, 760, 765 | Color/mono; replaced by D-series |
| D series | Current 2016+ | D305, D315, D335, D345, D375, D385, D712, D715, D717, D725, D735, D745, D785, D862, D862T | Modern desk phones |
| D8xx | Premium | D862, D862T | Touch, Bluetooth, dual gigabit |
| M series | DECT | M325, M425, M58, M65, M70, M85, M90 | Multi-cell DECT |
| A series | DECT | A170, A190 | Single-cell SOHO DECT |
| C series | Conference | C520, C620 | Speakerphone w/mic array |
| PA1 | Paging | PA1 | SIP-controlled overhead PA amp |
| K series | Expansion | K15, K30, K35 | Sidecar key modules |

### 3xx legacy

- snom 320 — entry mono, 12 SIP keys, GbE on 320 Pro (PoE optional)
- snom 360 — 12 dual-color LEDs, 47 keys, headset port, expansion bus
- snom 370 — 240x128 color LCD, 12 keys, USB host, top of legacy

### 7xx mid-gen (2012-2016)

- snom 710 — 4 lines, mono, GbE, PoE — entry
- snom 715 — same hw, more keys
- snom 720 — 18 keys, mono — power user
- snom 725 — color version of 720
- snom 760 — 4.3" color, 12 self-labeling keys (paper inserts replaced by LCD)
- snom 765 — 760 + WiFi/Bluetooth

### D series (current desk)

| Model | LCD | Keys | GbE | BT | WiFi | USB | Notes |
|-------|-----|------|-----|----|----|-----|-------|
| D120 | mono 2.7" | 2 | 100M | no | no | no | Entry |
| D305 | mono | 5 | yes | no | no | no | Single line |
| D315 | mono | 5+8 | yes | no | no | no | 8 self-label |
| D335 | mono | 8 | yes | no | no | no | 12 BLFs |
| D345 | color | 12 self | yes | no | no | no | Color |
| D375 | color 4.3" | 12 self | yes | yes | yes | yes | Premium color |
| D385 | color 4.3" | 12 self | yes | yes | yes | yes | Successor to D375 |
| D712 | mono | 4 | 100M | no | no | no | Budget exec |
| D715 | mono | 5+5 | yes | no | no | no | 5 self-label |
| D717 | mono | 6 | yes | no | no | no | |
| D725 | mono | 18 | yes | no | no | yes | Power user mono |
| D735 | color sensor 2.7" | 5+5 | yes | yes | opt | yes | Sensor display |
| D745 | color | 5 | yes | yes | yes | yes | |
| D785 | color 4.3" | 12 self | yes | yes | yes | yes | Flagship |
| D862 | color 5" touch | dynamic | dual GbE | yes | yes | yes | Latest premium |
| D862T | as D862 | + IP66 | dual GbE | yes | yes | yes | Ruggedized |

### M-series DECT multicell

- M58 — color screen handset
- M65 — color, IP65 rugged
- M70 — workhorse handset
- M85 — IP65 rugged industrial
- M90 — premium business
- M325 — single base + 4 handsets
- M425 — multi-cell base, up to 30 handsets / 40 calls
- M900 — IP DECT base for enterprise (up to 1000 users with multiple bases)

### A-series single-cell DECT

- A170 — entry single-cell base (up to 4 handsets, 4 calls)
- A190 — color screen, replaces M65 in pairs

### C-series conference

- C520 WiMi — modular, base + 2 wireless mics, daisy-chain
- C620 — Bluetooth, HD voice, 12-foot pickup

### PA1 paging amplifier

- 1-watt internal speaker + line-out for external 600-ohm horn/ceiling speakers
- Ethernet PoE
- SIP — register as identity, accept inbound calls = page

### K-series expansion (sidecar)

- K15, K30, K35 — paper-insert vs LCD-self-label key modules
- Connects via USB or proprietary expansion bus depending on phone
- Model compatibility: D7xx + K15 (paper), D8xx supports K35 (LCD)

## Default Access

```bash
# Browser
http://<phone-ip>/

# Default credentials (factory)
Username: admin
Password: 0000        # YES — literally four zeros

# User-mode (limited)
Username: user
Password: 0000

# IP discovery (factory state)
# Press: Settings (cogwheel) → 6 (Information) → 2 (System Info) → IP
# Or on D-series: Menu → Information → System Info
```

Default password 0000 is well-known — change at first config or via provisioning.

## Web UI Tour

```
Status
├── System Information
├── Log
├── SIP Trace                  ← live SIP trace toggle
├── DNS Cache
├── Memory
└── Settings

Setup
├── Preferences                ← time, language, DND, busy lamp, headset
├── Speed Dial                 ← 0..30 number list
├── Function Keys              ← P1..P12, Fkey, navigation softkeys
├── Identity 1                 ← SIP account 1 (12 total identities)
│   ├── Login
│   ├── SIP Settings
│   ├── NAT
│   ├── RTP
│   └── More options
├── Identity 2..12
├── Action URL Settings        ← outbound HTTP webhooks on events
├── Advanced                   ← provisioning, audio, codec, security, QoS
└── Network                    ← IP, DHCP, VLAN, VPN, 802.1X

Directories
├── Phone Directory (local)
├── External directories
└── LDAP

Maintenance
├── Diagnostics
├── Reboot
├── Reset Values
└── Software Update
```

## Identity Configuration

Snom uses "Identity" for what other vendors call "Account" or "Line". Up to 12 identities per phone (model-dependent — D862 = 12, D305 = 4).

Setup → Identity 1 → Login tab:

| Field | Description |
|-------|-------------|
| Identity active | on/off |
| Display name | name shown on outbound caller-ID |
| Account | SIP user-part (e.g. `1001`) |
| Password | SIP auth password |
| Registrar | server FQDN/IP (e.g. `pbx.example.com`) |
| Outbound Proxy | optional proxy `proxy.example.com:5060;transport=udp` |
| Failover Identity | secondary registrar |
| Authentication Username | if different from Account |
| Mailbox | voicemail extension (e.g. `*97`) — used by Voicemail key |
| Ringtone | per-identity ring tone selection |
| Display text for idle screen | label shown bottom of idle screen |
| Dial-Plan String | digit-map regex |
| Country | for tones (dial, busy) |

Setup → Identity 1 → SIP tab:

```
Music on hold server   sip:moh@pbx.example.com
User picture           http://provserver/photo/1001.png
Dial-Plan              [x*]+#|[0-9]{2,11}#?
Network Identity       LAN | WAN
Outgoing Identity Type UDP | TCP | TLS
Q-Value                0.0..1.0  (for parallel multi-identity)
Subscription Expiry    3600 sec
Re-registration time   60 sec
Failed registration retry 30 sec
```

Setup → Identity 1 → NAT tab:

```
NAT Identity      LAN | WAN
STUN identity     yes | no
STUN server       stun.example.com:3478
STUN refresh time 30
Long SIP Contact  (NAT keepalive) yes
```

Setup → Identity 1 → RTP tab:

```
codec1: PCMA (alaw)
codec2: PCMU (ulaw)
codec3: G722
codec4: opus
codec5: iLBC
codec6: G729
codec7: telephone-event
RTP packet size      20ms
Symmetric RTP        yes
RTP Encryption       off | optional | mandatory   (= SRTP)
Media security       SDES
```

## Codec Configuration

Per-identity in Setup → Identity X → RTP, OR globally via setting `codec_priority`. Phone walks codecs in order during SDP offer.

Supported codecs (model-dependent):

| Codec | Bitrate | Bandwidth | Notes |
|-------|---------|-----------|-------|
| PCMU (g711u) | 64 kbps | 87 kbps | Universal |
| PCMA (g711a) | 64 kbps | 87 kbps | EU default |
| G.722 | 64 kbps | 87 kbps | Wideband HD |
| G.729 | 8 kbps | 31 kbps | License-free since 2017 |
| iLBC | 13.3/15.2 kbps | 28 kbps | Lossy-network friendly |
| Opus | 6-510 kbps | adaptive | D7xx and later only |
| L16 | 256 kbps | 280 kbps | Uncompressed (rare) |

Set codec list:

```bash
# Provisioning XML
<setting_name>codec1</setting_name>: PCMA
<setting_name>codec2</setting_name>: PCMU
<setting_name>codec3</setting_name>: G722
```

## Function Keys

Programmable keys (P1..P12 on D-series, expandable via K-modules). Each key has a Type and Number.

Setup → Function Keys:

| Type | Description | Number field |
|------|-------------|--------------|
| Line | identity selector | 1..12 |
| Extension | BLF for monitored extension | `1002` (subscribes presence) |
| Destination | Speed Dial | `1002` |
| Speed Dial (without BLF) | dial only | `1002` |
| Configuration Type | quick toggle (DND, redirect) | `dnd` / `redirect` |
| Action URL | HTTP GET on press | `http://srv/cmd?id=$mac` |
| DTMF | send DTMF mid-call | `*99` |
| Conference | bridge in extension | `1003` |
| Voice Recorder | record current call | (no value) |
| Voicemail | dial mailbox | `*97` |
| Multicast Listen | join multicast group | `224.0.1.50:5555` |
| Park Orbit | call park | `*68101` |
| Pickup | directed pickup | `**1002` |
| Group Pickup | answer group call | `*8` |
| Intercom | auto-answer extension | `1002` |
| Transfer | blind transfer hot key | `1002` |
| URL | open built-in browser | `http://internal/menu.xml` |
| Push 2 Talk | hold-to-talk multicast | group `224.0.1.50:5555` |
| Forward | redirect target toggle | `1010` |
| Conf Init | start conference | (none) |
| Help | display help | (none) |
| Headset | toggle headset | (none) |

LED indication mode:

```
P1_led: on-off       — light when feature active
P1_led: on-blink     — blink when active (e.g. ringing BLF)
```

## BLF — Busy Lamp Field

Function Key Type = **Extension**; Number = monitored extension. Phone subscribes via SIP `SUBSCRIBE` to `dialog` event package on the registrar.

```
Setup → Function Keys → P2:
  Type:        Extension
  Number:      1002
  Short Text:  Bob
```

Activity indicator states:

| State | LED | Display |
|-------|-----|---------|
| idle | off | name only |
| ringing | fast blink | name + ring icon |
| busy | solid red | name + busy icon |
| dnd | solid yellow | name + DND icon |

Press idle BLF = speed dial monitored ext.
Press ringing BLF = pickup that call (directed pickup via configured prefix).

```
Activity Indicator settings (per-key):
  Mode: idle | ringing-only | busy-only | all
  Subscribe via: <identity>      ← which identity carries the SUBSCRIBE
```

## Action URL — Outbound Webhooks

HTTP GET callbacks fired on phone events. Setup → Advanced → QoS/Security → Action URL Settings (or for some firmwares Setup → Action URL Settings).

| Event | Setting key |
|-------|-------------|
| Phone setup (boot) | `action_setup_url` |
| Incoming ringing | `action_incoming_url` |
| Outgoing call | `action_outgoing_url` |
| Connected | `action_connected_url` |
| Disconnected | `action_disconnected_url` |
| Missed call | `action_missed_url` |
| DND on | `action_dnd_on_url` |
| DND off | `action_dnd_off_url` |
| Register success | `action_registered_url` |
| Register failed | `action_register_failed_url` |
| Off-hook | `action_offhook_url` |
| On-hook | `action_onhook_url` |

URL variables (substituted at fire time):

| Var | Value |
|-----|-------|
| `$mac` | phone MAC |
| `$ip` | phone IP |
| `$model` | phone model (e.g. `D785`) |
| `$firmware` | firmware version |
| `$active_url` | current SIP URI |
| `$active_user` | identity user |
| `$active_host` | identity registrar |
| `$local` | local SIP URI |
| `$remote` | remote SIP URI |
| `$display_local` | local display name |
| `$display_remote` | remote display name |
| `$call-id` | SIP Call-ID |
| `$cseq` | SIP CSeq |
| `$expansion_module` | attached EM model |
| `$csta_id` | CSTA call ID |

Example:

```
action_incoming_url: http://crm.example.com/popup?phone=$mac&from=$remote&called=$local
```

## Action URI — Inbound Commands

Phone exposes HTTP server accepting commands at `/command.htm?key=<NAME>`. Used by external apps to push DTMF, dial numbers, answer.

```bash
# Dial
curl http://10.0.0.50/command.htm?number=1002

# Press function key
curl http://10.0.0.50/command.htm?key=F1

# Send DTMF
curl http://10.0.0.50/command.htm?key=DTMF1

# Hangup
curl http://10.0.0.50/command.htm?key=CANCEL

# Answer
curl http://10.0.0.50/command.htm?key=ENTER

# Hold
curl http://10.0.0.50/command.htm?key=R

# Reboot
curl http://10.0.0.50/command.htm?key=keyboot
```

Security — Setup → Advanced → QoS/Security:

```
http_user / http_pass         ← Basic auth on web UI
filter_registrar              ← only accept SIP requests from registered server IP
admin_mode                    ← user vs admin login
allowed_ips                   ← whitelist (comma-sep)
```

## Provisioning

Setup → Advanced → Update tab:

```
Setting Server                http://provserver.example.com/{mac}.xml
Update Policy                 Update automatically | Ask for update | Never update
Settings refresh timer        86400 sec (= once per day)
Trusted Cert. Auth.           upload CA bundle for HTTPS provisioning
SBC                            (Session Border Controller URL)
```

URL macros:

| Macro | Substituted with |
|-------|------------------|
| `{mac}` | MAC w/o colons, lowercase (`000413abcdef`) |
| `{model}` | model name (`snomD785`) |
| `{firmware}` | firmware version |

Provisioning protocols supported:

- HTTP (port 80) — most common
- HTTPS (port 443) — recommended; requires CA in trust store
- TFTP (port 69) — legacy DHCP option 66
- FTP (port 21) — anonymous or auth

Trigger reload:

```bash
# Settings refresh timer in XML
<setting_name>settings_refresh_timer</setting_name>3600
# Or send SIP NOTIFY check-sync
# Or manual: Maintenance → Reboot
# Or: curl http://phone/command.htm?key=keyboot
```

## Configuration File Format

Snom uses XML. File can be flat (single file with all settings) or hierarchy (`{model}.xml` + per-MAC override).

Top-level structure:

```xml
<?xml version="1.0" encoding="utf-8"?>
<settings>
  <phone-settings>
    <!-- generic phone-wide settings -->
    <setting_name>setting_value</setting_name>
  </phone-settings>
  <functionKeys>
    <!-- function key definitions -->
  </functionKeys>
  <programmableKeys>
    <!-- on D series additional bank -->
  </programmableKeys>
</settings>
```

Per-identity settings indexed `[1..12]`:

```xml
<user_realname idx="1" perm="">Alice</user_realname>
<user_name idx="1" perm="">1001</user_name>
<user_pname idx="1" perm="">password123</user_pname>
<user_host idx="1" perm="">pbx.example.com</user_host>
```

Permission attribute (`perm`):

- `""` (empty) — user can change
- `"R"` — read-only (locked)
- `"!"` — change requires admin re-auth

## Sample Snom XML

Minimal `000413aabbcc.xml` for a Snom D785 with one identity + 4 function keys:

```xml
<?xml version="1.0" encoding="utf-8"?>
<settings>
  <phone-settings>
    <!-- network -->
    <dhcp e="2">on</dhcp>
    <vlan_id perm="R">100</vlan_id>

    <!-- web UI -->
    <http_user perm="">admin</http_user>
    <http_pass perm="">S3cret!</http_pass>
    <admin_mode_password perm="">S3cret!</admin_mode_password>

    <!-- time -->
    <ntp_server perm="">pool.ntp.org</ntp_server>
    <timezone perm="">+1</timezone>
    <utc_offset perm="">3600</utc_offset>

    <!-- locale -->
    <language perm="">English</language>
    <web_language perm="">English</web_language>
    <tone_scheme perm="">USA</tone_scheme>

    <!-- identity 1 -->
    <user_active idx="1" perm="">on</user_active>
    <user_realname idx="1" perm="">Alice Adams</user_realname>
    <user_name idx="1" perm="">1001</user_name>
    <user_pname idx="1" perm="">SuperSecret</user_pname>
    <user_host idx="1" perm="">pbx.example.com</user_host>
    <user_mailbox idx="1" perm="">*97</user_mailbox>
    <user_outbound idx="1" perm=""></user_outbound>
    <user_dp_str idx="1" perm="">[x*]+#|[0-9]{2,11}#?</user_dp_str>
    <user_idle_text idx="1" perm="">Alice 1001</user_idle_text>

    <!-- codec preferences identity 1 -->
    <codec1 idx="1" perm="">pcma</codec1>
    <codec2 idx="1" perm="">pcmu</codec2>
    <codec3 idx="1" perm="">g722</codec3>
    <codec4 idx="1" perm="">opus</codec4>

    <!-- SRTP -->
    <user_srtp idx="1" perm="">optional</user_srtp>
    <user_srtp_auth idx="1" perm="">on</user_srtp_auth>

    <!-- action URL -->
    <action_incoming_url perm="">http://crm.example.com/popup?from=$remote&amp;to=$local</action_incoming_url>
    <action_connected_url perm="">http://crm.example.com/connected?call=$call-id</action_connected_url>
  </phone-settings>

  <functionKeys>
    <fkey idx="0" context="active" label="Line 1" perm="">line 1</fkey>
    <fkey idx="1" context="1" label="Bob" perm="">blf sip:1002@pbx.example.com</fkey>
    <fkey idx="2" context="1" label="Carol" perm="">blf sip:1003@pbx.example.com</fkey>
    <fkey idx="3" context="1" label="VM" perm="">speed *97</fkey>
    <fkey idx="4" context="1" label="Park" perm="">speed *68</fkey>
    <fkey idx="5" context="1" label="Conf" perm="">keyevent F_CONFERENCE</fkey>
  </functionKeys>
</settings>
```

Per-model fallback file `snomD785.xml` (lower priority — overlaid by per-MAC):

```xml
<?xml version="1.0" encoding="utf-8"?>
<settings>
  <phone-settings>
    <ntp_server perm="">pool.ntp.org</ntp_server>
    <timezone perm="">+1</timezone>
    <language perm="">English</language>
  </phone-settings>
</settings>
```

Lookup order on boot:

1. `<provserver>/snomD785-<MAC>.xml`
2. `<provserver>/<MAC>.xml`
3. `<provserver>/snomD785.xml`
4. `<provserver>/snom.xml` (very generic)

## Snom Redirection Service (RPS / SRS)

Snom-managed cloud service: every Snom phone ships with hardcoded URL `provisioning.snom.com`. On factory boot, phone asks SRS — "where do I provision from?" SRS responds with redirect URL configured by reseller.

Workflow:

1. Reseller buys phone with MAC `000413aabbcc`
2. Reseller logs into https://service.snom.com → assigns MAC to provisioning URL (e.g. `https://prov.example.com/{mac}.xml`)
3. Customer powers on phone — DHCP assigns IP — phone queries SRS over HTTPS
4. SRS replies: "your settings are at `https://prov.example.com/000413aabbcc.xml`"
5. Phone fetches that URL, applies config, registers with PBX

Account binding:

```
service.snom.com → MAC list
  MAC: 000413aabbcc
  Provisioning URL: https://prov.example.com/{mac}.xml
  Auth: HTTP Basic (optional username/password)
  Locked: yes (prevents change without password)
```

To remove from SRS — reseller deletes assignment OR phone admin sets `setting_server` locally to a different URL (only if SRS not "locked").

## DHCP Options

Phone honors:

- **Option 6** — DNS servers
- **Option 42** — NTP servers
- **Option 43** — vendor-specific
- **Option 66** — TFTP server name (legacy provisioning)
- **Option 12** — hostname

Vendor-class string: `snom<model>` (e.g. `snom320`, `snomD785`).

Option 43 sub-options (Snom-specific encoding):

```
Sub-option 1: Setting URL          (e.g. "http://prov.example.com/{mac}.xml")
Sub-option 2: HTTP user
Sub-option 3: HTTP password
Sub-option 4: VLAN ID
Sub-option 5: Update Policy
```

Example ISC dhcpd:

```
option vendor-class-identifier "snomD785";
option vendor-encapsulated-options
    01:23:"http://prov.example.com/{mac}.xml";
option tftp-server-name "tftp.example.com";  # Option 66
```

ISC dhcpd snom class:

```
class "snom-phones" {
    match if substring(option vendor-class-identifier, 0, 4) = "snom";
    option tftp-server-name "tftp.example.com";
}
```

## NAT

Setup → Identity X → NAT:

```
NAT Identity            LAN | WAN
STUN Identity           yes | no
STUN server             stun.example.com:3478
STUN refresh interval   30 sec
Symmetric RTP           on
Long SIP-Contact (rport)  on   ← RFC 3581 rport
Persistent TCP          on    ← keep TCP socket open
NAT keepalive           crlf | options | none
NAT keepalive interval  20 sec
```

Outbound proxy goes in Identity → Login → Outbound proxy:

```
proxy.example.com:5060;transport=udp
proxy.example.com:5061;transport=tls
```

## SIP-over-TLS

Setup → Identity X → SIP:

```
Outgoing Identity Type:    TLS
SIP Port:                  5061   (typical)
```

Cert install — Setup → Advanced → QoS/Security → Trusted Certificate Authorities:

```
Upload PEM bundle of CAs phone must trust.
Or set:
  user_certificate idx="1": URL to client cert
  user_private_key idx="1": URL to client key
```

Provisioning XML:

```xml
<user_outgoing idx="1" perm="">tls</user_outgoing>
<user_host idx="1" perm="">pbx.example.com:5061;transport=tls</user_host>
<trusted_certificates perm="">http://prov/ca-bundle.pem</trusted_certificates>
```

## SRTP

Per-identity:

```
RTP Encryption:       off | optional | mandatory
SRTP Auth:            on | off
```

Config XML:

```xml
<user_srtp idx="1" perm="">optional</user_srtp>     <!-- offers SDES if peer does -->
<user_srtp idx="1" perm="">mandatory</user_srtp>    <!-- refuses plain RTP -->
<user_srtp_auth idx="1" perm="">on</user_srtp_auth>
```

SRTP key exchange = SDES (a=crypto in SDP). Phone generates key, sends in SDP offer, peer echoes accepted suite.

## Phone Languages

Snom ships extensive language packs. Set globally + per-user.

Available (D series):

- English (US/UK)
- German
- French
- Italian
- Spanish
- Portuguese
- Russian
- Polish
- Czech
- Slovak
- Hungarian
- Romanian
- Bulgarian
- Greek
- Turkish
- Dutch
- Swedish
- Norwegian
- Danish
- Finnish
- Hebrew (RTL)
- Arabic (RTL)
- Chinese (Simplified)
- Japanese
- Korean

```xml
<language perm="">German</language>
<web_language perm="">German</web_language>
<tone_scheme perm="">DEU</tone_scheme>
```

## Phone Customization

Ringtones — built-in 10 melodies; custom WAV files via URL:

```xml
<ringer1 perm="">http://prov/sounds/marimba.wav</ringer1>
<user_ringer idx="1" perm="">1</user_ringer>
```

Idle screen layout — D series with color LCD:

```xml
<idle_clock_size perm="">large</idle_clock_size>     <!-- large | medium -->
<idle_show_date perm="">on</idle_show_date>
<idle_show_weather perm="">off</idle_show_weather>
<idle_logo_url perm="">http://prov/logo-150x60.png</idle_logo_url>
<idle_logo_position perm="">center</idle_logo_position>
```

Color schemes:

```xml
<theme perm="">dark</theme>      <!-- dark | light | classic -->
<accent_color perm="">blue</accent_color>
```

Screen saver:

```xml
<screensaver_enabled perm="">on</screensaver_enabled>
<screensaver_timeout perm="">300</screensaver_timeout>     <!-- sec -->
<screensaver_url perm="">http://prov/screensaver.png</screensaver_url>
```

## M-Series DECT Multicell

Architecture:

```
M425/M900 IP base  (DECT radio + SIP front-end)
  ├── M325 (single-cell secondary base)
  ├── repeater
  └── handsets (M58, M65, M70, M85, M90)

multi-cell M900: up to 40 base stations, 1000 handsets, 80 simultaneous calls
single-cell M325: 1 base, 4 handsets, 4 simultaneous calls
single-cell M425: 1 base, 30 handsets, 8 simultaneous calls
```

Provisioning:

```xml
<!-- M-series base phone-settings -->
<dect_handset_count perm="">8</dect_handset_count>
<handset_user idx="1" perm="">1001</handset_user>      <!-- handset 1 → identity for 1001 -->
<handset_password idx="1" perm="">SuperSecret</handset_password>
<handset_realname idx="1" perm="">Alice</handset_realname>
```

Handset registration (manual, per-handset):

```
Base web UI:
  Setup → Identity → enable identity for HS index
  Press [REGISTER] in web UI within 60s
On handset:
  Menu → Settings → Connectivity → Register → Pair with base
  Enter PIN (default 0000)
```

Multi-cell handover — handset roams; ongoing call hands off between bases without drop. Requires:

- Bases on same LAN (multicast for sync)
- DECT IPEI registered to "data sync" master base
- Audio path goes through registered base (MGW), so latency tolerable LAN-wide

## C-Series Conference

C520 modular:

- Base unit with speaker, mics, daisy chain port
- 2 wireless mics (charge in base) — stage on table
- Up to 2 base units cascaded for large rooms

C620:

- Bluetooth pairing (mobile bridge)
- 360° mic array
- 4-watt speaker
- 12-foot voice pickup

Both register as standard SIP identity (1 identity each, plus Bluetooth bridge).

## PA1 — Paging Amplifier

SIP-controlled overhead PA. Internal 1W amp + 600Ω line-out for external horn/ceiling speaker.

Provisioning:

```xml
<!-- register as identity, auto-answer + speaker on -->
<user_active idx="1" perm="">on</user_active>
<user_name idx="1" perm="">page1</user_name>
<user_pname idx="1" perm="">pagepw</user_pname>
<user_host idx="1" perm="">pbx.example.com</user_host>
<user_auto_answer idx="1" perm="">on</user_auto_answer>
<user_dp_str idx="1" perm="">[x*]+#</user_dp_str>

<!-- multicast listening to overhead group -->
<multicast_codec perm="">PCMU</multicast_codec>
<multicast_address1 perm="">224.0.1.50:5555</multicast_address1>
```

Use case — Asterisk dialplan dials PA1 identity → call auto-answers → audio plays out PA. Or PA1 joins multicast group; sender phones do "Push 2 Talk" to group.

## Logs

Setup → Status → SIP Trace:

- Live SIP capture in browser; refresh button; "Save" to download `.txt`
- Buffer ~ last 50 messages

Setup → Status → Log:

- System log (kern, sip, audio, cert, dhcp) — last ~500 lines

Remote syslog:

```xml
<logging_servers perm="">syslog.example.com:514</logging_servers>
<log_level_console perm="">7</log_level_console>     <!-- 0=emerg .. 7=debug -->
<log_level_remote perm="">5</log_level_remote>       <!-- 5=notice typical -->
```

## PCAP

D-series & later — Setup → Status → PCAP recorder (firmware ≥ 8.9 / 10.x). Record button → start, Stop → download `.pcap`.

Older firmwares — only via SSH:

```bash
ssh admin@phone-ip
$ tcpdump -i eth0 -w /tmp/cap.pcap port 5060 or portrange 10000-20000
```

## Common Errors

```
Identity 1: Registration Failed (401 Unauthorized)
  → Wrong SIP password (user_pname)
  → Wrong auth-username (if PBX requires distinct auth-user)

Identity 1: Registration Failed (403 Forbidden)
  → Account disabled on PBX
  → IP blocked by PBX firewall (fail2ban triggered)
  → User reached registration limit

Identity 1: Registration Failed (408 Request Timeout)
  → No reply from registrar — wrong host, firewall, NAT
  → Check DNS resolution of user_host
  → STUN problem behind NAT

DHCP Failed
  → No DHCP server on VLAN
  → DHCP scope exhausted
  → 802.1X required and not configured

Provisioning: HTTP 401
  → Server requires auth — set http_user/http_pass for provisioning

Provisioning: HTTP 404 / File not found
  → URL macro not substituted (server expecting bare {mac} string?)
  → Filename case (Linux servers care about D785 vs d785)
  → MAC formatting — colons or no colons? Snom = no colons, lowercase

Provisioning: HTTPS - TLS Handshake Failed
  → CA not in phone trust store
  → Cert SAN mismatch
  → Time wrong → cert "not yet valid"

STUN Server Unreachable
  → Wrong STUN host
  → Firewall blocking 3478/udp

Codec Negotiation Failed
  → Phone offers only G729; PBX wants only PCMU
  → Add PCMU/PCMA to codec list

Time not set / NTP failed
  → ntp_server unreachable, DNS or firewall
  → Time wrong → SRTP fails, TLS fails, certs invalid
```

## Common Gotchas

### 1. Default password 0000 not changed

```
Broken — phone deployed; admin password still 0000; on shared LAN any user
        can browse to http://phone/, change SIP creds, dial $$$$.
Fix —   In provisioning XML:
        <admin_mode_password perm="R">StrongPass!</admin_mode_password>
        <http_pass perm="R">StrongPass!</http_pass>
        # perm="R" makes it read-only (cannot be changed via web UI)
```

### 2. Provisioning URL with {mac} but server expects hard-coded MAC

```
Broken — Setting URL: http://prov/{mac}.xml
        Phone replaces {mac} → "000413aabbcc" → http://prov/000413aabbcc.xml
        But server is serving fixed http://prov/myphone.xml — no match → 404
Fix —   Either:
        (a) Configure server to serve per-MAC files, OR
        (b) Setting URL: http://prov/myphone.xml  (no macro)
```

### 3. Multiple identities but function key not assigned to specific identity

```
Broken — Identities 1 & 2 both registered; function key BLF for ext 1002.
        Phone auto-uses identity 1 to subscribe; identity 1 has no presence
        permission on PBX. BLF stays grey.
Fix —   Function Keys → key context = "2"  (forces identity 2 for SUBSCRIBE)
        XML: <fkey idx="1" context="2" ...>blf sip:1002@...</fkey>
```

### 4. SRTP mandatory but server only does plain RTP

```
Broken — user_srtp = mandatory; PBX sends SDP without a=crypto; phone
        rejects with "488 Not Acceptable Here"; calls fail.
Fix —   user_srtp = optional   ← phone offers SRTP, falls back to RTP
        Or enable SRTP at PBX side
```

### 5. LDAP search filter wrong → no directory entries

```
Broken — LDAP filter: (cn=*$$NAME$$*)
        Phone substitutes search input "smith" but ldap field is "sn".
        Search returns 0 results.
Fix —   Filter: (|(cn=*$$NAME$$*)(sn=*$$NAME$$*)(givenName=*$$NAME$$*))
        Test with ldapsearch first.
```

### 6. Action URL using HTTPS without cert in trust store

```
Broken — action_incoming_url = https://crm.example.com/popup?from=$remote
        crm.example.com uses Let's Encrypt; phone trust store empty;
        webhook silently fails; no CRM popup.
Fix —   Upload LE root CA OR full chain to:
        Setup → Advanced → Certificate
        Or via XML: <trusted_certificates>http://prov/lets-encrypt-r3.pem</trusted_certificates>
```

### 7. DECT handset registration timeout

```
Broken — Press REGISTER on M325 base web UI → start handset registration
        on handset → take coffee break → 60s expired → handset shows
        "no base found".
Fix —   Have handset already in pairing menu BEFORE pressing base REGISTER.
        Within 60s click REGISTER on base, then press OK on handset.
```

### 8. Time wrong → cert validation fails

```
Broken — Phone deployed, NTP not configured, time = 2010-01-01.
        TLS provisioning (HTTPS) fails: "cert not yet valid".
        Phone falls back, registers w/o cert validation = security risk.
Fix —   Set ntp_server before HTTPS provisioning, OR pre-set time via
        DHCP option 42, OR use HTTP first boot then HTTPS subsequent.
```

### 9. VPN (built-in OpenVPN client) not enabled

```
Broken — Remote-worker phone deployed, no VPN; PBX only accepts SIP from
        corp net; registration 408 timeouts.
Fix —   Setup → Network → VPN → upload .ovpn config + ca.crt + client.crt
        + client.key. Enable. PBX-side firewall sees VPN-source IP.
        XML:
        <vpn_active perm="">on</vpn_active>
        <vpn_config perm="">http://prov/site.ovpn</vpn_config>
```

### 10. Function key BLF type vs "Speed Dial with BLF" confusion

```
Broken — Picked Type = "Speed Dial" with number "1002". Light never on.
        Speed Dial does not subscribe to dialog event package.
Fix —   Type = "Extension" (synonyms: BLF). This is the type that issues
        SUBSCRIBE for presence.
```

### 11. Codec priority excludes server's preferred

```
Broken — Phone codec list: opus, g722.
        PBX trunk codec list: pcmu, g729.
        SDP intersection empty → 488 Not Acceptable.
Fix —   Always include PCMU/PCMA at minimum:
        codec1=PCMA, codec2=PCMU, codec3=G722, codec4=opus
```

### 12. Old vs new firmware default config-file naming

```
Broken — Old firmware (8.x) looks for SIP_<MAC>.htm + snom320.htm.
        New firmware (10.x+) looks for {MAC}.xml + snomD785.xml.
        Mixed-fleet provisioning server needs both naming conventions.
Fix —   Run provserver behind URL rewrite:
        SIP_(.*)\.htm → /\1.xml
        snom([0-9]+)\.htm → /snom\1.xml
        Or generate both formats per phone.
```

### 13. Function key fkey idx=0 reserved for line key

```
Broken — XML: <fkey idx="0" context="1">blf sip:1002@host</fkey>
        BLF appears not to work; idx=0 is the dedicated line key on D-series.
Fix —   Start BLFs from idx="1" or higher; reserve 0 for line.
```

### 14. Dial plan eats # before sending

```
Broken — Want to send "1002#" (dial-now hash).
        Default dial-plan: [x*]+#  → strips trailing # before sending.
        Server sees "1002" only, no transfer-out.
Fix —   Adjust user_dp_str:  [x*]+#?  or per-key dial-as-is.
```

### 15. Duplicate Identity registrations

```
Broken — Same identity assigned to two phones (e.g. desk + softphone via
        same SIP user). PBX may allow parallel registrations only if
        forking enabled; otherwise newest wins, oldest 408s.
Fix —   Use distinct identities per device, OR enable PBX parallel
        registration (Asterisk: max_contacts = 5).
```

## Diagnostic Tools

### SIP trace from web UI

```
Setup → Status → SIP Trace
  [Reset]   [Refresh]   [Save]
  ─────────────────────────────
  REGISTER sip:pbx.example.com SIP/2.0 ...
  SIP/2.0 401 Unauthorized
  REGISTER sip:pbx.example.com SIP/2.0 ...
  SIP/2.0 200 OK
```

### Syslog forwarding

```xml
<logging_servers perm="">10.0.0.99:514</logging_servers>
<log_filter perm="">5</log_filter>          <!-- noise filter level -->
```

Receive at server:

```bash
nc -ul 514
# or rsyslog
tail -f /var/log/syslog | grep snom
```

### SSH access (older firmware)

```bash
ssh admin@phone-ip
# password 0000

# inside:
$ cat /var/log/messages
$ sip-trace
$ tcpdump -i eth0 -w /tmp/cap.pcap
$ scp /tmp/cap.pcap user@host:/tmp/   # copy out
```

Modern firmware (10.x+) SSH disabled by default — re-enable:

```xml
<sshd_active perm="">on</sshd_active>     <!-- and set strong admin pwd -->
```

### Phone debug menu

```
Settings (cogwheel) on idle screen
  → Information → System Info
    IP, MAC, Firmware, Identity status
  → Maintenance → Reboot
  → Maintenance → Reset Settings
```

### `command.htm` reboot

```bash
curl -u admin:S3cret! "http://10.0.0.50/command.htm?key=keyboot"
```

## SSH Access

| Firmware era | SSH default | Notes |
|--------------|-------------|-------|
| 7.x (3xx series) | enabled | user `root`, pass `0000` |
| 8.x | enabled | user `admin`, pass = admin web pwd |
| 8.7+ | enabled but warned | recommend password change |
| 10.x+ (D series) | disabled by default | enable via `sshd_active` |

When enabled — busybox shell, limited tools:

```bash
ssh admin@phone-ip
> ls /
> cat /tmp/sip-trace.log
> tcpdump -i eth0 -nn port 5060
> ps
> top
> exit
```

## Multicast Paging

Group of phones tunes to a UDP multicast address; one phone or amp pushes RTP-encoded PCMU/PCMA to that group; receivers play out speaker.

Listener config — Function Key:

```
Type:   Multicast Listen
Number: 224.0.1.50:5555
Codec:  PCMU
Label:  Page-All
```

Or via XML phone-settings:

```xml
<multicast_address1 perm="">224.0.1.50:5555</multicast_address1>
<multicast_address1_label perm="">All Page</multicast_address1_label>
<multicast_codec1 perm="">PCMU</multicast_codec1>
<multicast_priority perm="">2</multicast_priority>     <!-- 1..10 -->
```

Sender — Push-to-Talk function key:

```
Type:   Push 2 Talk
Number: 224.0.1.50:5555
Codec:  PCMU
```

Phone holds key → encodes mic to RTP → sends to group → all listeners speaker out.

Multicast routing — by default IP multicast does NOT cross VLANs/routers; ensure all phones on same L2, OR enable IGMP/PIM on routers.

## EHS Headset

Electronic Hook Switch — wireless headset (Plantronics/Jabra/Sennheiser) "answers" via base-station signal carrying the off-hook message back to the phone over a third connector.

Snom EHS adapter — Plantronics APC-43 or Jabra LINK 14201-35; plugs into:

```
Phone:           Headset jack (RJ-9 / 3.5mm)
EHS adapter:     EHS port (standard Snom EHS pinout)
Headset base:    PC USB or DECT base
```

Function Key for headset:

```
Type:   Headset
Number: (none)
Press: toggle headset on/off (or use EHS lift-to-answer)
```

XML:

```xml
<headset_device perm="">EHS</headset_device>     <!-- vs USB | BT | none -->
<ehs_type perm="">Plantronics</ehs_type>
```

## Snom Mini-Browser

XML browser app — phone displays XML-defined screens. Define menu, list, input, tone screens.

Function Key:

```
Type:   URL
Number: http://internal.example.com/menu.xml
Label:  IT Menu
```

Sample XML response:

```xml
<?xml version="1.0" encoding="utf-8"?>
<SnomIPPhoneText>
  <Title>IT Menu</Title>
  <Text>1=Reboot 2=Status 3=Tickets</Text>
  <SoftKeyItem>
    <Label>Back</Label>
    <Action>SoftKey:Exit</Action>
  </SoftKeyItem>
</SnomIPPhoneText>
```

```xml
<SnomIPPhoneMenu>
  <Title>Tickets</Title>
  <MenuItem>
    <Prompt>Open ticket</Prompt>
    <URL>http://internal/open.xml</URL>
  </MenuItem>
  <MenuItem>
    <Prompt>Status</Prompt>
    <URL>http://internal/status.xml</URL>
  </MenuItem>
</SnomIPPhoneMenu>
```

Useful for status displays, room booking, helpdesk launch, queue stats.

## Idioms

- **Always change default 0000 password** — first config, every device. Prefer perm="R" lock so user web UI cannot revert.
- **Use HTTPS for provisioning** — and verify CA cert in trust store. NTP first, HTTPS second.
- **DECT M-series for medium businesses, A-series for SOHO** — M425/M900 scales to thousands of handsets across multiple bases; A170 is 4-handset cap.
- **Snom + 3CX is a popular pair** — 3CX includes "Snom Plug-and-Play" template provisioning out of the box.
- **Use Snom Redirection Service for zero-touch global deploy** — register MAC + URL at service.snom.com; ship phone to remote site; first boot = autoconfig.
- **PCMA before PCMU in EU; PCMU before PCMA in US** — match local PSTN.
- **Backup `setting_server`-rendered XML before firmware upgrade** — old firmware's per-MAC keys may be renamed in new firmware (e.g. legacy `dect_handset_pin` → modern `handset_password`).
- **Never deploy with `user_srtp = mandatory` until SRTP confirmed end-to-end** — start `optional`, raise once RTPs are encrypted in capture.
- **Use `perm="R"` on critical settings** — locks user web-UI override; provisioning server is source of truth.
- **For multicast paging, force same VLAN** — avoid IGMP-snooping headaches.
- **Function key context attr = identity index** — drives which identity sends SUBSCRIBE for BLF.
- **Snom dial plan accepts regex-ish syntax** — test in Setup → Identity → Dial-Plan; `[0-9]{2,11}#?` allows 2-11 digit numbers with optional #.

## See Also

- ip-phone-provisioning
- sip-protocol
- asterisk
- freeswitch
- yealink-phones
- polycom-phones
- cisco-phones
- grandstream-phones

## References

- Snom Service Hub — https://service.snom.com (knowledge base, documentation)
- Snom Support — https://www.snom.com/en/support/
- Snom XML Provisioning Guide — https://service.snom.com/display/wiki/XML+Provisioning
- Snom Configuration Reference — https://service.snom.com/display/wiki/Configuration+Settings
- Snom Action URLs — https://service.snom.com/display/wiki/Action+URL
- Snom Mini-Browser XML — https://service.snom.com/display/wiki/Mini+Browser
- Snom Redirection Service — https://service.snom.com (login → "Redirection")
- VTech (parent) — https://www.vtech.com
- RFC 3261 — SIP
- RFC 3711 — SRTP
- RFC 4568 — SDES
- RFC 3581 — rport
- RFC 5389 — STUN
