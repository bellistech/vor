# Polycom / Poly / HP IP Phones

Polycom (now Poly, acquired by HP 2022) IP phone family — VVX, Edge, SoundPoint, SoundStation, Trio, CCX, Rove. UC Software (UCS) provisioning via XML cfg files, master+child architecture, EFK scripting, RPRM/Cloud management.

## Setup

Polycom portfolio (now Poly + HP-acquired):

```
Family             Models                                           Status
-----------------  ----------------------------------------------- ------------
VVX (legacy 1st)   VVX 200/300/400/500/600                          EOL
VVX (2nd gen)      VVX 150/250/350/450/501/601                      Sold via HP
VVX 1500           Touchscreen video phone                          EOL
Edge (modern)      Edge B series (basic), Edge E series (executive) Current
SoundPoint IP      320/321/331/335/450/550/650/670                  EOL legacy
SoundStation IP    4000/5000/6000/7000 (conference)                 EOL
Trio (conference)  8500, 8800, C60                                  Current/EOL
CCX (Teams)        CCX 400/500/600/700                              Current Teams
Rove (DECT)        Rove 30/40 handsets, B2/B4/B8 base stations      Current
OBi (ATA / DECT)   OBi200, OBi202, OBi302, OBi504, OBi1062          Different OS!
```

Key model facts:

- **VVX 150** — 2-line entry, monochrome, 2 line keys, Gigabit
- **VVX 250** — 4-line, color, 4 line keys, Gigabit, USB
- **VVX 350** — 6-line, color, 6 line keys, Gigabit, USB
- **VVX 450** — 12-line, color, 12 line keys, Gigabit, USB, BT optional
- **VVX 501/601** — 12/16-line color touch, BT, WiFi optional, USB recording
- **Edge B series** — basic, 2/4/8/12-line, color screen, Gigabit
- **Edge E series** — executive, color, larger screens, BT, WiFi, USB-C
- **Trio 8500** — small conference, Ethernet, BT, color touch
- **Trio 8800** — medium-large conference, expansion mics, HDMI, USB
- **CCX 400** — 5" color touch, Teams, Android-based
- **CCX 500** — 5" color touch, Teams or generic SIP, BT
- **CCX 600** — 7" color touch, Teams, BT, USB
- **CCX 700** — 7" color touch, Teams, BT, USB-C, side USB camera
- **Rove DECT** — multi-cell DECT, up to 1000 handsets across base stations

PoE classes:

```
VVX 150        802.3af class 1 (3.84W)
VVX 250/350    802.3af class 2 (6.49W)
VVX 450        802.3af class 2 (6.49W)
VVX 501/601    802.3af class 2 with BT/USB activity bumps to class 3
Trio 8500      802.3af class 3 (12.95W)
Trio 8800      802.3at PoE+ class 4 (25.5W)
CCX 600/700    802.3af with USB peripherals can need PoE+
```

Polycom-as-vendor history:

- 1990 — Polycom founded (audio conferencing)
- 1998 — SoundPoint IP debut
- 2007 — VVX line introduced
- 2018 — Plantronics acquires Polycom, becomes Poly
- 2022 — HP acquires Poly

## UC Software Lineage

Polycom UC Software (UCS) versions (the legacy/SoundPoint/VVX firmware):

```
Version       Released   Highlights
------------  ---------  ----------------------------------------------------
4.0.x         2011       Foundation rewrite, sip.cfg deprecated for cfg-files
4.1.x         2012       SfB/Lync support
5.0.x         2013       BroadWorks visual voicemail
5.4.x         2015       Major: H.264, USB headset, EFK improvements
5.5.x         2016       OpenSSL refresh, AS-SIP
5.7.x         2017       Better TLS, dial-plan engine
5.8.x         2018       SIP transport fallback
5.9.x         2019       Last 5.x; security fixes
6.0.x         2019       Renumbered from 5.9; web UI overhaul
6.1.x         2020       Acoustic fence, OAuth2
6.2.x         2021       TLS 1.3, larger contact directory
6.3.x         2022       BlueJeans integration removed (BJN sunset)
6.4.x         2023       Rebranded to Poly UC Software
7.0.x         2024       Edge B/E base, gradually replacing UCS for new models
```

UCS download:

```bash
# Legacy: support.polycom.com → Polycom UC Software → device family
# Modern: support.hp.com/us-en/drivers → search by model

# Each release ships:
# - sip-<MODEL>.ld (firmware blob)
# - cfg/ directory of template configs
# - splfeatures.cfg, region.cfg, etc.
```

OBi product reality (OBiHai acquisition 2018):

- OBi line is *not* on UCS — runs OBi Firmware (separate codebase)
- Different config schema (XML but completely different parameter names)
- Provisioned via OBiTalk cloud or XML files
- OBi200 ATA, OBi202 ATA dual-port, OBi302 enterprise ATA
- OBi504 / OBi1062 IP phones (also legacy)
- Treat as a separate vendor when planning provisioning

Edge OS for Edge B/E:

- New firmware family from 2022
- Schema-incompatible with UCS — *different* parameter names
- Auto-update default through Poly Cloud (Lens)
- Provisioning still works with XML but parameter set is the new one

## Default Access

Web UI:

```
URL:           http://<phone-ip>/
Modern HTTPS:  https://<phone-ip>/  (self-signed cert)
```

Default credentials:

```
admin password:  456    (yes, three digits, factory default)
user password:   123
```

Recovery / boot password is the same admin password (456) by default.

Web UI redirects:

```
http://<ip>/        → login.htm
After login:        → home.htm
```

Phone-side admin login (no web access):

```
1. While at idle: Settings → Advanced
2. Press: 4 5 6 (the password)
3. Press # to "make multi-key" (older phones) — type chars rapidly
4. OK to confirm

# Some legacy SoundPoint:
# Press *, *, *, MENU, then 4 5 6 OK
```

Reset web password from phone:

```
Menu → Settings → Advanced → 456 → Admin Settings → Change Admin Password
```

Reset to factory default (when locked out):

```
1. Power-cycle phone (unplug PoE / power)
2. During boot: hold 1, 3, 5 simultaneously
3. Phone enters "Boot Recovery" menu
4. Select "Reset to Factory" or enter password 0123 (boot menu)

# VVX 4xx/5xx/6xx:
# During boot, hold 4, 6, 8, *
```

## Web UI Tour

Top-level pages:

```
Home          Status snapshot, registration state, line summary
Status        Network, lines, system, hardware diagnostics
Settings      Most config knobs (when in Advanced view)
Diagnostics   Live PCAP, SIP message log, ping test, traceroute
Utilities     Reboot, factory reset, import/export config, software upgrade
```

Simple/Advanced View toggle:

```
Simple:    Stripped-down user-facing options (5–10 fields per page)
Advanced:  Full UCS parameter tree exposure
Toggle:    Top-right of any Settings page → "Switch to Advanced View"
```

Status sub-pages:

```
Status → Platform → Phone           (model, MAC, serial, board rev)
Status → Network → Ethernet         (link, speed, DHCP, IP)
Status → Network → SIP              (Lines registered, transport)
Status → Lines → Line N             (per-line statistics)
Status → Diagnostics → Audio        (jitter, packet loss, MOS estimate)
Status → Diagnostics → Memory       (heap, stack)
```

Diagnostics commonly used:

```
Diagnostics → Capture                (PCAP — start/stop, download .pcap)
Diagnostics → Logs → Application     (live UCS log)
Diagnostics → Logs → Boot            (boot log)
Diagnostics → Ping
Diagnostics → Traceroute
```

## Configuration File Architecture

Master config → list of additional cfg files → applied in order → per-MAC overrides last:

```
1. <MAC>.cfg                    (per-phone master, optional)
2. 000000000000.cfg             (global master, the "default")
3. Files listed in CONFIG_FILES (region, features, sip-basic, ...)
4. <MAC>-overrides.xml          (admin/user overrides, last writer wins)
5. user-side personalization saved on phone itself
```

Typical file split:

```
File              Purpose
----------------  --------------------------------------------------------------
000000000000.cfg  Master: lists CONFIG_FILES, server-wide policies
<MAC>.cfg         Per-phone master: overrides global master
sip-basic.cfg     SIP registrations: reg.1.address, auth, transport
phoneapp.cfg      Phone application UI: feature toggles, line keys
region.cfg        Region: timezone, time format, audio codecs by region
features.cfg      Per-feature toggles: voicemail, BLF, headset, BT, etc.
applications.cfg  Web/microbrowser apps; URL stuff
contacts.xml      Local phonebook (pure phonebook XML, not UCS)
site.cfg          Per-site customisations
overrides.cfg     Last-applied overlay
splfeatures.cfg   Service Provider features (Lync, BroadCloud)
```

Resolution order:

1. **Defaults baked in firmware** — every parameter has a default
2. **Master config** (000000000000.cfg or <MAC>.cfg)
3. **Files in CONFIG_FILES** — left to right
4. **Per-MAC overrides** — <MAC>-phone.cfg, <MAC>-overrides.xml
5. **Local user changes** stored on phone (optional)

Storage paths on provisioning server:

```
/tftpboot/                          (typical TFTP root)
├── 000000000000.cfg                (default master)
├── sip-basic.cfg
├── features.cfg
├── region.cfg
├── phoneapp.cfg
├── contacts.xml
├── 0004f2abcdef.cfg                (per-MAC master example)
├── 0004f2abcdef-phone.cfg          (per-MAC settings)
├── 0004f2abcdef-directory.xml      (per-MAC directory)
├── overrides/                      (auto-saved overrides)
│   └── 0004f2abcdef-phone.cfg
├── logs/
│   └── 0004f2abcdef-app.log
└── sip-vvx450.ld                   (firmware blob)
```

## Master Config Format

Master config is XML, root element `APPLICATION`:

```xml
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<APPLICATION
    APP_FILE_PATH="sip.ld"
    CONFIG_FILES="phone1.cfg, sip-basic.cfg, features.cfg, region.cfg"
    MISC_FILES=""
    LOG_FILE_DIRECTORY="logs"
    OVERRIDES_DIRECTORY="overrides"
    CONTACTS_DIRECTORY=""
    LICENSE_DIRECTORY="">
</APPLICATION>
```

Per-MAC master (e.g., `0004f2abcdef.cfg`):

```xml
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<APPLICATION
    APP_FILE_PATH="sip.ld"
    CONFIG_FILES="0004f2abcdef-phone.cfg, sip-basic.cfg, features.cfg, region.cfg"
    LOG_FILE_DIRECTORY="logs"
    OVERRIDES_DIRECTORY="overrides">
</APPLICATION>
```

Attribute summary:

```
APP_FILE_PATH        Path to firmware .ld blob (relative to prov root)
CONFIG_FILES         Comma-separated list of cfg files to load (in order)
MISC_FILES           Optional per-phone media (ringtones, images)
LOG_FILE_DIRECTORY   Where phone uploads logs
OVERRIDES_DIRECTORY  Where phone uploads override changes (web UI saves here)
CONTACTS_DIRECTORY   Where to fetch/upload contacts.xml
LICENSE_DIRECTORY    Where to fetch licenses (rare, mostly UCS pre-feature-pack)
```

Comments:

```xml
<!-- This is a config comment -->
<!-- Polycom only honours top-level XML comments, not inside attributes -->
```

## sip-basic.cfg

Canonical starting file for any deployment:

```xml
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<polycomConfig
    xsi:schemaLocation="urn:com:polycom:configuration polycom_config.xsd"
    xmlns="urn:com:polycom:configuration"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">

  <reg
    reg.1.address="2001@sip.example.com"
    reg.1.label="2001"
    reg.1.displayName="Reception"
    reg.1.auth.userId="2001"
    reg.1.auth.password="seCret123!"
    reg.1.outboundProxy.address="sbc.example.com"
    reg.1.outboundProxy.port="5060"
    reg.1.outboundProxy.transport="UDPOnly"
    reg.1.server.1.address="sip.example.com"
    reg.1.server.1.port="5060"
    reg.1.server.1.transport="UDPOnly"
    reg.1.server.1.expires="3600"
    reg.1.server.1.expires.overlap="120"
    reg.1.server.1.register="1"
    reg.1.server.1.retryMaxCount="3"
    reg.1.server.1.retryTimeOut="0"

    reg.1.server.2.address="sip-failover.example.com"
    reg.1.server.2.port="5060"
    reg.1.server.2.transport="UDPOnly"
    reg.1.server.2.expires="3600"
    reg.1.server.2.register="1" />

  <voIpProt
    voIpProt.SIP.local.port="5060"
    voIpProt.SIP.specialEvent.checkSync.alwaysReboot="0"
    voIpProt.SIP.outboundProxy.address=""
    voIpProt.SIP.outboundProxy.port="5060" />

</polycomConfig>
```

Key reg.X.* parameters:

```
reg.X.address                 SIP URI / username @ domain
reg.X.label                   Label shown next to line key
reg.X.displayName             Display name in From header
reg.X.auth.userId             Auth username (often == address user-part)
reg.X.auth.password           Auth password (plaintext in cfg!)
reg.X.outboundProxy.address   Outbound proxy IP/FQDN
reg.X.outboundProxy.port      Outbound proxy port
reg.X.outboundProxy.transport UDPOnly / TCPOnly / TLS / DNSnaptr
reg.X.server.Y.address        Per-registration server (Y=1..4 fallback)
reg.X.server.Y.port           Per-server port
reg.X.server.Y.transport      Per-server transport
reg.X.server.Y.expires        Registration interval (seconds)
reg.X.server.Y.expires.overlap Time before expiry to send re-REGISTER
reg.X.server.Y.register       1=register here, 0=use as proxy only
reg.X.server.Y.retryMaxCount  Retry attempts before next server
reg.X.server.Y.retryTimeOut   Initial retry delay (0=auto)
reg.X.server.Y.failOver.failBack.mode disabled/registration/duration/newRequests
reg.X.lineKeys                Number of line keys this reg consumes
reg.X.callsPerLineKey         Concurrent calls per line key
reg.X.type                    private/shared (BLA shared lines)
reg.X.thirdPartyName          Shared appearance group ID
```

## Configuration Parameter Names

Names are dot-separated, case-sensitive, lowercase generally except acronyms:

```
Category       Examples
-------------  --------------------------------------------------
reg.*          Registration / line config
call.*         Call handling: forwarding, transfer, hold, DND
tcpIpApp.*     Network: SNTP, DHCP, port ranges, RTP/RTCP
device.*       Hardware: provisioning, network mode, sleep
feature.*      Capabilities: voicemail, BLF, BT, headset, lync, etc.
softkey.*      Per-state softkey buttons (idle, ringing, in-call, etc.)
msg.*          Message Waiting Indicator, voicemail behavior
efk.*          Enhanced Feature Keys (programmable button scripts)
lineKey.*      Line key labels, types, mapping
voIpProt.*     SIP-protocol layer behaviour
sec.*          Security: SRTP, TLS, Hosting
log.*          Logging
bg.*           Background image / colour
nat.*          NAT keep-alive, STUN address
dialplan.*     Dial-plan rules
attendant.*    BLF / attendant console
dir.*          Directory: corporate (LDAP), personal
homeScreen.*   Home screen layout
qos.*          QoS DSCP marking for voice/video/SIP
applications.* Microbrowser / OpenH323 / Lync
```

Boolean: `0`/`1` (sometimes `true`/`false` in newer schemas).
String: quoted directly in XML attribute.
Numeric: bare number.
Enum: documented per-parameter.

Find any parameter:

```bash
# Pull the official "UC Software Configuration Reference" PDF
# Currently: poly_config_reference_<version>.xls/.pdf
# Search: param name → description → valid values → default
```

## Line Keys

Basic line-key mapping:

```xml
<lineKey
  lineKey.reassignment.enabled="1"

  lineKey.1.category="Line"
  lineKey.1.index="1"
  lineKey.1.label="2001 Reception"

  lineKey.2.category="BLF"
  lineKey.2.index="2"
  lineKey.2.label="Sales 2010"
  lineKey.2.value="2010"

  lineKey.3.category="EFK"
  lineKey.3.index="3"
  lineKey.3.label="Park 1"
  lineKey.3.value="1" />
```

Categories:

```
Line       Standard line / registration appearance
BLF        Busy Lamp Field for monitored extension
EFK        Enhanced Feature Key macro
SpeedDial  Speed dial only
Empty      Disabled
```

Softkey customisation per call state:

```xml
<softkey
  softkey.feature.basicCallManagement="0"
  softkey.feature.cancel="1"
  softkey.feature.directories="1"
  softkey.feature.endcall="1"
  softkey.feature.forward="1"
  softkey.feature.split="1"

  efk.softkey.alignLeft="0" />
```

Custom softkeys per state (UCS 5.4+):

```xml
<customSoftKey
  softkey.1.enable="1"
  softkey.1.label="Park"
  softkey.1.action="*7"
  softkey.1.use.idle="1"
  softkey.1.use.active="0"
  softkey.1.use.alerting="0"
  softkey.1.use.dialtone="0"
  softkey.1.use.proceeding="0"
  softkey.1.use.setup="0"
  softkey.1.use.hold="0" />
```

States exposed:

```
idle         Idle / no call
dialtone     Off-hook, no digits yet
proceeding   Outbound INVITE sent
setup        Inbound call ringing, before answer
alerting     Outbound ringing
active       Connected call
hold         Call on hold
remotehold   Far end has placed us on hold
```

## EFK — Enhanced Feature Keys

Programmable line keys / softkeys with a mini-language:

```xml
<efk
  efk.efklist.1.label="Park"
  efk.efklist.1.mname="park1"
  efk.efklist.1.status="1"
  efk.efklist.1.action.string="*7$Tinvite$"
  efk.efklist.1.type="invite" />

<efk
  efk.efklist.2.label="Login"
  efk.efklist.2.mname="login"
  efk.efklist.2.status="1"
  efk.efklist.2.action.string="$P1N4$$Cwc$$P2N6$$Cwc$$Tdtmf$"
  efk.efklist.2.type="dtmf" />
```

Action mini-language tokens:

```
$Tinvite$         Send INVITE with the preceding string as URI/digits
$Tdtmf$           Send the preceding as in-call DTMF
$Cwc$             Wait for next prompt (clears the working buffer)
$Pname$           Prompt user for input (name = label)
$PNn$             Prompt and limit to N digits
$Crelease$        Release current call
$Chold$           Hold current call
$Cresume$         Resume held call
$Cans$            Answer incoming
$Ctransfer$       Initiate blind transfer
$Cforward$        Forward
$FDigits$         Insert literal digits (with no prompt)
$L<duration>$     Wait <duration> ms before next action
```

Examples:

```xml
<!-- Speed-dial 5551212 -->
efk.efklist.3.action.string="5551212$Tinvite$"

<!-- Park current call to slot 1 -->
efk.efklist.4.action.string="$Cxfer$*701$Tdtmf$"

<!-- Conference: hold current, dial new, transfer-to-conference -->
efk.efklist.5.action.string="$Chold$$P1N7$$Tinvite$$L3000$$Cconf$"

<!-- Send DTMF *9 then wait 1 second then 1234# -->
efk.efklist.6.action.string="*9$L1000$1234#$Tdtmf$"

<!-- Conditional based on call state — UCS limited support -->
efk.efklist.7.action.string="$FN1$1234$Tinvite$"
```

Soft-key triggering EFK:

```xml
softkey.1.label="Park"
softkey.1.action="!park1"   <!-- ! prefix invokes EFK by mname -->
softkey.1.use.active="1"
```

## Provisioning Setup

End-to-end workflow:

```
1. DHCP gives phone an IP + Option 66/160 with provisioning URL
2. Phone fetches <MAC>.cfg or 000000000000.cfg (master)
3. Master config: APP_FILE_PATH=firmware blob, CONFIG_FILES=child files
4. Phone downloads firmware if version mismatch
5. Child files in CONFIG_FILES applied in order
6. Per-MAC overrides applied last (highest precedence)
7. Phone registers using reg.X.* parameters
```

Layered overrides:

```
firmware defaults
   ↓
000000000000.cfg                 (global master settings)
   ↓
sip-basic.cfg, features.cfg, ... (CONFIG_FILES from master)
   ↓
<MAC>.cfg                        (per-phone master overrides)
   ↓
<MAC>-phone.cfg                  (per-phone individual settings)
   ↓
<MAC>-overrides.xml              (web UI / phone UI changes)
   ↓
in-memory user changes (until next reload)
```

Per-MAC config naming:

```
0004f2abcdef.cfg              Per-MAC master (replaces 000000000000.cfg)
0004f2abcdef-phone.cfg        Per-MAC settings (overrides shared cfgs)
0004f2abcdef-directory.xml    Per-MAC contacts
0004f2abcdef-license.xml      Per-MAC license (rare)
```

Trigger phone to re-provision:

```
- Reboot
- Web UI → Utilities → Restart Phone
- "Resync" via NOTIFY check-sync (RFC 4235 if enabled)
- DHCP renewal won't trigger; phone polls per device.prov.* schedule
```

Polling settings:

```xml
<device
  device.prov.serverName="provisioning.example.com"
  device.prov.serverType="HTTPS"
  device.prov.user="phone"
  device.prov.password="provpass"
  device.prov.tagSerialNo="1"
  device.prov.zeroTouchURL="https://obp.plcm.com/profilez" />

<prov
  prov.polling.enabled="1"
  prov.polling.mode="abs"            <!-- abs / rel / random -->
  prov.polling.period="86400"        <!-- daily -->
  prov.polling.time="04:00"
  prov.polling.timeRandomEnd="" />
```

## Polycom RPRM (Resource Manager)

RPRM = Resource Manager (Polycom-Cloud or Self-hosted central management).

Capabilities:

- Bulk firmware management
- Global and per-phone config templates
- Provisioning service (HTTPS with optional client certs)
- Real-time status / call logs
- Inventory across thousands of devices
- Group-based config inheritance

Modes:

```
Cloud:        Polycom/Poly Cloud RPRM (now mostly Lens / Voice)
On-Premises:  RPRM appliance (Linux VM, requires license)
              - URL: https://rprm.example.com
              - Web UI on port 443
              - Phone provisioning port 80/443
```

Certificate-based device authentication:

```
Each phone has factory-installed Polycom Device Cert (PDC)
RPRM trusts Polycom Root CA → can verify phone's cert
TLS-mutual auth: phone presents PDC, RPRM accepts
Replaces / supplements username+password provisioning auth
```

Phone-side enable:

```xml
<device
  device.prov.serverName="rprm.example.com"
  device.prov.serverType="HTTPS"
  device.auth.useDeviceCertificate="1"
  device.prov.user=""
  device.prov.password="" />
```

## Polycom Cloud

Modern Poly cloud services (replacing RPRM for new deployments):

```
Poly Lens         Device management, analytics, firmware
                  https://lens.poly.com
Poly Voice        SIP-as-a-Service (PSTN trunk + UC platform)
                  https://voice.poly.com
Poly Studio       Video bar management
Poly Cloud Relay  Provisioning hand-off for ZTP
```

Device Certificate (PKI):

```
Every Polycom/Poly phone manufactured since ~2010 ships with:
  - Polycom Device Certificate (PDC) — issued by Polycom Root CA
  - Per-device unique private key (in TPM-like secure element)
  - Used for: ZTP, RPRM auth, Cloud auth, optional SIP-TLS mutual

Verify chain: support.poly.com / Polycom Root CA bundle (.pem)
Common Name: <model>-<serial>.polycom.com
Subject Alt: phone MAC
```

ZTP flow leveraging PDC:

```
1. Phone factory-fresh, no provisioning URL configured
2. DHCP → no Option 66/160 → fall back to ZTP
3. Phone makes TLS request to obp.plcm.com (Poly Cloud Redirector)
4. Redirector validates PDC, looks up MAC in customer database
5. Returns 302 redirect to customer's actual provisioning URL
6. Phone re-provisions from customer URL
```

## PolyOS / Edge OS

Newer firmware family for Edge B/E series:

- Linux-based (vs UCS proprietary RTOS)
- Different XML schema — parameter names *don't* match UCS
- Phone Web UI redesigned
- Auto-update through Lens default
- Backwards-incompatible config: cannot copy/paste UCS files

Edge OS schema sample:

```xml
<configuration>
  <accounts>
    <account id="1">
      <enabled>true</enabled>
      <username>2001</username>
      <password>seCret123!</password>
      <domain>sip.example.com</domain>
      <outboundProxy>sbc.example.com:5060;transport=tls</outboundProxy>
      <displayName>Reception</displayName>
    </account>
  </accounts>
  <network>
    <vlan>
      <enabled>true</enabled>
      <id>200</id>
    </vlan>
  </network>
</configuration>
```

Edge OS provisioning paths:

```
Default URL:        https://lens.poly.com/...     (cloud)
Self-hosted:        Configure via Lens or local web UI
Per-MAC file:       <MAC>.xml  (note: .xml, not .cfg)
```

## Codec Configuration

Codec preference list:

```xml
<voice
  voice.codecPref.G711_A="1"
  voice.codecPref.G711_Mu="2"
  voice.codecPref.G722="3"
  voice.codecPref.G7221_24kbps="4"
  voice.codecPref.G7221_32kbps="0"
  voice.codecPref.G7221C_24kbps="0"
  voice.codecPref.G7221C_32kbps="0"
  voice.codecPref.G7221C_48kbps="0"
  voice.codecPref.G729_AB="5"
  voice.codecPref.iLBC_13_33kbps="0"
  voice.codecPref.iLBC_15_2kbps="0"
  voice.codecPref.Opus="0"
  voice.codecPref.Siren14_24kbps="0"
  voice.codecPref.Siren14_32kbps="0"
  voice.codecPref.Siren14_48kbps="0"
  voice.codecPref.Siren22_32kbps="0"
  voice.codecPref.Siren22_48kbps="0"
  voice.codecPref.Siren22_64kbps="0"
  voice.codecPref.AAC_LD_64kbps="0"
  voice.codecPref.AAC_LD_96kbps="0" />
```

Numeric value = priority (1 = highest, 0 = disabled).

SDP advertised codecs (Limit what's offered, not just preferred):

```xml
<voice
  voice.codecPref.SDP.G711_A="1"
  voice.codecPref.SDP.G711_Mu="2"
  voice.codecPref.SDP.G722="3"
  voice.codecPref.SDP.G729_AB="0" />
```

Supported codec list (UCS 6.x):

```
PCMU (G.711 Mu)         Mandatory baseline
PCMA (G.711 A)          Mandatory baseline
G.722                   Wideband 7kHz, 64kbps
G.722.1 24kbps          Wideband
G.722.1 32kbps          Wideband
G.722.1C 24kbps         Super-wideband (14kHz)
G.722.1C 32kbps         Super-wideband
G.722.1C 48kbps         Super-wideband premium
G.729AB                 8kbps narrowband (license required on legacy)
Opus                    UCS 5.7+; wideband, dynamic bitrate
Siren14                 Polycom proprietary, 14kHz wideband
Siren22                 Polycom proprietary, 22kHz fullband
iLBC 13.33 / 15.2       Internet Low Bitrate, narrowband
AAC-LD 64/96kbps        Apple/Lync compatible
```

Codec license note:

```
G.729 historically required a license sticker activation
UCS 5.0+ usually bundled, but some VVX models need per-phone license
Check: Status → Platform → Licenses
Add: Utilities → License Configuration → upload license XML
```

## RTP Settings

Port ranges:

```xml
<tcpIpApp
  tcpIpApp.port.rtp.mediaPortRangeStart="2222"
  tcpIpApp.port.rtp.mediaPortRangeEnd="2269"
  tcpIpApp.port.rtp.lowBandwidthMaxValue="2269"
  tcpIpApp.port.rtp.filterByIp="1"
  tcpIpApp.port.rtp.filterByPort="1"
  tcpIpApp.port.rtcp.adjusted="1"     <!-- RTCP on RTP+1 by default -->
  tcpIpApp.port.rtcp.mediaPortRangeStart="0"
  tcpIpApp.port.rtcp.mediaPortRangeEnd="0" />
```

Port allocation rules:

- Default RTP range 2222–2269 (24 even ports = 24 concurrent calls)
- Each call uses one even port for RTP
- RTCP: even+1 if `tcpIpApp.port.rtcp.adjusted=1`
- Symmetric RTP: phone keeps source IP/port from far end
- DSCP marking via `qos.ethernet.rtp.user_priority` (CoS) and `qos.ip.rtp.dscp` (default EF=46)

QoS:

```xml
<qos
  qos.ethernet.rtp.user_priority="5"
  qos.ip.rtp.dscp="EF"
  qos.ip.callControl.dscp="CS3"
  qos.ip.video.dscp="AF41"
  qos.ip.rtcp.dscp="EF" />
```

## NAT

NAT keep-alive:

```xml
<nat
  nat.keepalive.interval="30"
  nat.signalPort="0"
  nat.mediaPortStart="0"
  nat.ip="" />
```

Keep-alive types:

```
voIpProt.SIP.keepalive.sessionTimers.refreshInDialog Re-INVITE in dialog
voIpProt.SIP.keepalive.sessionTimers.uacInactiveCalls Refresh idle calls
nat.keepalive.interval               OPTIONS-based keep-alive every N sec
```

STUN config:

```xml
<nat
  nat.ice.enabled="1"
  nat.ice.mode="Standard"             <!-- Standard / MSOC -->
  nat.ice.stun.server="stun.example.com"
  nat.ice.stun.port="3478"
  nat.ice.turn.server="turn.example.com"
  nat.ice.turn.port="3478"
  nat.ice.turn.username="phone"
  nat.ice.turn.password="turnpass"
  nat.ice.turn.transports="UDP,TCP" />
```

RFC 2543 hold (sometimes needed for legacy SBCs):

```xml
voIpProt.SIP.useRfc2543hold="1"  <!-- send c=0.0.0.0 instead of a=sendonly -->
```

Server-specific keep-alive:

```xml
voIpProt.server.1.specialEvent.checkSync.alwaysReboot="0"
voIpProt.server.1.specialEvent.lineSeize.nonINVITE="0"
```

## SIP-over-TLS

Enable TLS transport:

```xml
<reg
  reg.1.transport="TLS"            <!-- TLS / TCPpreferred / UDPOnly / DNSnaptr -->
  reg.1.outboundProxy.transport="TLS"
  reg.1.server.1.transport="TLS"
  reg.1.server.1.port="5061" />
```

Cert install — web UI:

```
Settings → Network → TLS → Configure TLS
  - Trusted CA certificates: Add CA cert (.pem)
  - Custom Device Certificate: Add per-device cert + key
  - Choose: TLS Application: SIP / Provisioning / Web / etc.
```

Cert install — config file:

```xml
<sec
  sec.TLS.profileSelection.SIP="ApplicationProfile1"
  sec.TLS.profile.ApplicationProfile1.caCertList="AppProfile1"
  sec.TLS.profile.ApplicationProfile1.deviceCert="Builtin" />

<device
  device.sec.TLS.customCaCert1.set="1"
  device.sec.TLS.customCaCert1="-----BEGIN CERTIFICATE-----
MIIDXT...base64...
-----END CERTIFICATE-----" />
```

Strict cert validation:

```xml
<sec
  sec.TLS.protocol="TLSv1_2"             <!-- TLSv1_0/1_1/1_2/1_3 -->
  sec.TLS.cipherSuiteDefault="0"
  sec.TLS.profile.ApplicationProfile1.cipherList="ECDHE-RSA-AES128-GCM-SHA256:..."
  sec.TLS.cipherSuiteOverride="OPENSSL"
  sec.TLS.customCipherList=""
  sec.TLS.profile.ApplicationProfile1.peerNameValidate="1"
  sec.TLS.profile.ApplicationProfile1.peerHostnameMatch.byEnvironment="1" />
```

Mutual TLS:

```xml
<sec
  sec.TLS.profile.ApplicationProfile1.mutualAuth="1"
  sec.TLS.profile.ApplicationProfile1.deviceCert="Builtin" />
```

## SRTP

SRTP/DTLS-SRTP enable + behavior:

```xml
<sec
  sec.srtp.enable="1"
  sec.srtp.offer="1"            <!-- Include SRTP in our SDP offer -->
  sec.srtp.require="0"          <!-- 1 = drop call if no SRTP -->
  sec.srtp.simplifiedBestEffort="1"
  sec.srtp.holdWithNewKey="1"
  sec.srtp.resumeWithNewKey="1"
  sec.srtp.mki.enabled="1"
  sec.srtp.mki.length="4"
  sec.srtp.key.lifetime="2147483648"   <!-- 2^31 = max -->
  sec.srtp.callTransfer.useCallerKey="1"
  sec.srtp.profile.AES_CM_128_HMAC_SHA1_80="1"
  sec.srtp.profile.AES_CM_128_HMAC_SHA1_32="1" />
```

Per-registration SRTP override:

```xml
reg.1.srtp.enable="1"
reg.1.srtp.offer="1"
reg.1.srtp.require="1"
```

SDP shows:

```
a=crypto:1 AES_CM_128_HMAC_SHA1_80 inline:fjslkdfjlksdj+lkjkjsdf|2^31|1:4
a=crypto:2 AES_CM_128_HMAC_SHA1_32 inline:askdjklsdfasdf+jklsdjkflsd|2^31|1:4
```

## Provisioning URL

Configuration:

```xml
<device
  device.prov.serverName="provisioning.example.com"
  device.prov.serverType="HTTPS"
  device.prov.user="phone"
  device.prov.password="provpass"
  device.prov.tagSerialNo="1" />
```

Server types:

```
TFTP            UDP/69, no auth, plaintext
FTP             TCP/21, basic auth, plaintext
FTPS            TCP/990 (implicit) or 21 (explicit), TLS
HTTP            TCP/80, basic auth in URL or via 401 challenge
HTTPS           TCP/443, optional client cert auth
```

URL forms:

```
device.prov.serverName="tftp.example.com"          → tftp://...
device.prov.serverName="http://prov.example.com/path"  → forces HTTP
device.prov.serverName="https://user:pass@prov.example.com/cfg" → with creds inline
```

The `device.prov.tagSerialNo="1"` substitutes:

```
{$mac}    Phone MAC (lowercase, no separators)
{$model}  Phone model string (vvx450, vvx601, etc.)
```

When set in path:

```
device.prov.serverName="https://prov.example.com/{$model}/{$mac}/"
→ phone fetches https://prov.example.com/vvx450/0004f2abcdef/000000000000.cfg
```

## DHCP Provisioning

DHCP option support:

```
Option 66       TFTP server name (string)        Standard PXE, used by many
Option 150      TFTP server IPs (Cisco)          Polycom: NO (Cisco-specific)
Option 160      Polycom-specific HTTP/HTTPS URL  Recommended for Polycom
Option 159      Alternative HTTP                  Older / fallback
Option 60       Vendor Class Identifier          "Polycom-VVX450" etc.
Option 43       Vendor-specific encapsulated     Used for ZTP override
```

Polycom phone DHCP option order:

```
1. Boot server option / device.prov.serverName (config)
2. Option 160 (Polycom HTTP/HTTPS)
3. Option 159
4. Option 66 (TFTP)
5. Static IP fallback / ZTP cloud
```

ISC DHCP example:

```
option polycom-160 code 160 = string;

subnet 10.20.30.0 netmask 255.255.255.0 {
  range 10.20.30.50 10.20.30.200;
  option routers 10.20.30.1;
  option domain-name-servers 10.20.30.1;

  # Standard TFTP fallback
  option tftp-server-name "tftp.example.com";

  # Polycom HTTPS provisioning
  option polycom-160 "https://prov.example.com/poly/";
}
```

dnsmasq:

```
dhcp-option=66,"tftp.example.com"
dhcp-option=160,"https://prov.example.com/poly/"
```

Cisco IOS:

```
ip dhcp pool VOICE
 network 10.20.30.0 255.255.255.0
 default-router 10.20.30.1
 option 66 ip 10.20.30.10
 option 160 ascii "https://prov.example.com/poly/"
```

Vendor encap (Option 43) for ZTP override:

```
# Polycom-specific suboptions inside Option 43
# Suboption 1: provisioning URL
# Suboption 2: server type
```

## Polycom Zero Touch Provisioning (ZTP)

ZTP bypasses DHCP options — uses Poly Cloud:

```
1. Factory-fresh phone boots, no DHCP option 66/160 received
2. After ~30 sec discovery period
3. Phone phones home: TLS to obp.plcm.com (Poly Cloud Redirector)
4. Authenticates with factory PDC (Polycom Device Cert)
5. Cloud looks up phone MAC → customer profile (set in Poly Lens)
6. Cloud returns redirect URL to customer provisioning server
7. Phone provisions from customer URL — pulls master cfg
```

ZTP profile setup (Poly Lens):

```
1. Customer logs in to Poly Lens
2. Adds devices by MAC range / serial / SKU
3. Defines "ZTP Profile":
     - Provisioning Server URL (HTTPS)
     - Provisioning username/password
     - First-boot password override
4. When phone first boots, Lens push profile via redirector
```

Disable ZTP per phone:

```xml
<device
  device.prov.zeroTouchURL="" />
```

Force ZTP retry:

```
1. Web UI → Utilities → Reboot
2. Or factory reset
3. Phone re-checks ZTP after DHCP options exhausted
```

## Time Settings

Time / NTP / Timezone:

```xml
<tcpIpApp
  tcpIpApp.sntp.address="pool.ntp.org"
  tcpIpApp.sntp.address.overrideDHCP="1"
  tcpIpApp.sntp.gmtOffset="0"
  tcpIpApp.sntp.gmtOffset.overrideDHCP="1"
  tcpIpApp.sntp.gmtOffsetCityID="000"
  tcpIpApp.sntp.resyncPeriod="86400"
  tcpIpApp.sntp.daylightSavings.enable="1"
  tcpIpApp.sntp.daylightSavings.fixedDayEnable="0"
  tcpIpApp.sntp.daylightSavings.start.month="3"
  tcpIpApp.sntp.daylightSavings.start.dayOfWeek="1"
  tcpIpApp.sntp.daylightSavings.start.dayOfWeek.lastInMonth="0"
  tcpIpApp.sntp.daylightSavings.start.date="0"
  tcpIpApp.sntp.daylightSavings.start.time="2"
  tcpIpApp.sntp.daylightSavings.stop.month="11"
  tcpIpApp.sntp.daylightSavings.stop.dayOfWeek="1"
  tcpIpApp.sntp.daylightSavings.stop.dayOfWeek.lastInMonth="0"
  tcpIpApp.sntp.daylightSavings.stop.date="0"
  tcpIpApp.sntp.daylightSavings.stop.time="2" />

<feature
  feature.timezone="GMT"
  feature.localUI.timeFormat="0"   <!-- 0=24h, 1=12h -->
  feature.localUI.dateFormat="0" />
```

Timezone strings (some examples):

```
gmtOffsetCityID    Description                  gmtOffset
-----------------  ---------------------------  ---------
006                Honolulu                     -36000
018                Los Angeles (Pacific)        -28800
020                Denver (Mountain)            -25200
024                Chicago (Central)            -21600
030                New York (Eastern)           -18000
035                Halifax (Atlantic)           -14400
050                London (GMT/BST)             0
055                Berlin / Paris               3600
060                Athens / Helsinki            7200
070                Moscow                       10800
085                Singapore                    28800
090                Tokyo                        32400
094                Sydney                       36000
```

(Full list in `region.cfg` template shipped with UCS.)

## Display Customization

Background:

```xml
<bg
  bg.color.bm.1.name="Default"
  bg.color.bm.1.em.name="Default"
  bg.color.bm.1.selection="0"
  bg.color.selection="0"
  bg.color.bm.1.em.selection="0"
  bg.bm.1.name="image1.png"
  bg.bm.1.adj="0"
  bg.bm.1.em.name="image1_em.png"
  bg.bm.1.em.adj="0"
  bg.color.bm.1.em.thumb="image1_thumb.png"
  bg.color.gradient.startpos="0,0"
  bg.color.gradient.endpos="100,100"
  bg.color.gradient.startcolor.red="255"
  bg.color.gradient.endcolor.red="0"
  bg.logo.name="logo.bmp"
  bg.logo.adj="0" />
```

Background image format/spec:

```
VVX 250/350      320x240 BMP / PNG / JPG (200KB max)
VVX 450          320x240 (some 480x272)
VVX 501/601      320x240 (color), background-display area
Edge B/E         Native screen res, larger for executive
Trio 8500/8800   1280x720 / 1280x800 with on-screen UI overlay
CCX 600/700      Android wallpaper standard sizes
```

Idle screen template / "phone display":

```xml
<homeScreen
  homeScreen.directory.enable="1"
  homeScreen.statusIndicator.enable="1"
  homeScreen.bridgeAttn.enable="1"
  homeScreen.callQuality.enable="0"
  homeScreen.dial.enable="1"
  homeScreen.forward.enable="1"
  homeScreen.dnd.enable="1"
  homeScreen.featureKeys.enable="1"
  homeScreen.messages.enable="1"
  homeScreen.recents.enable="1"
  homeScreen.settings.enable="1"
  homeScreen.statusIndicator.enable="1" />
```

Screensaver (UCS 5.x+):

```xml
<feature
  feature.screensaver.enabled="1"
  feature.screensaver.idleTimeout="180"      <!-- seconds -->
  feature.screensaver.imageFolder="screensaver"
  feature.screensaver.type="default"          <!-- default / logoOnly / pictureFrame -->
  feature.screensaver.waitTime="120"
  feature.screensaver.allowed="1" />
```

## BLF (Busy Lamp Field)

Enable BLF subscription:

```xml
<feature
  feature.busyLamp.enable="1"
  feature.busyLamp.alertingDuration="0"
  feature.busyLamp.callBack.enable="1" />

<attendant
  attendant.uri="blf-list@sip.example.com"
  attendant.behaviors.display.spontaneousCallAppearances.normal="1"
  attendant.behaviors.display.spontaneousCallAppearances.attendant="1"
  attendant.behaviors.display.remoteCalleeNumber="1"
  attendant.reg="1" />
```

Per-resource BLF:

```xml
attendant.resourceList.1.address="2010@sip.example.com"
attendant.resourceList.1.label="Sales 2010"
attendant.resourceList.1.type="normal"
attendant.resourceList.1.proceeding-prompt=""
attendant.resourceList.1.callAddress="2010@sip.example.com"
```

Behind the scenes:

```
Phone sends SUBSCRIBE for "dialog;sla" event package to attendant.uri
Server sends NOTIFY with multipart body listing each resource and state
Phone updates LED:
  - Off:    idle
  - Solid:  in call
  - Slow blink:  ringing
  - Fast blink:  hold
```

## Bluetooth

Enable Bluetooth (where supported):

```xml
<feature
  feature.bluetooth.enabled="1" />

<bluetooth
  bluetooth.connectable="1"
  bluetooth.discoverable="1"
  bluetooth.adapterEnabled="1"
  bluetooth.radioOn="1"
  bluetooth.pairedList.maxNumber="10" />
```

Model support matrix:

```
Model        BT?     Notes
-----------  ------  ------------------------------------------
VVX 150      No
VVX 250      No
VVX 350      No
VVX 401      Yes     (USB BT dongle required pre-built models)
VVX 450      Yes     (built-in BT)
VVX 501      Yes
VVX 601      Yes     (built-in BT, classic + BLE)
Edge B100    No
Edge B400    Yes     (top SKU)
Edge E series Yes     (all)
Trio 8500    Yes
Trio 8800    Yes     (BT 4.0)
CCX 500/600/700 Yes  (BT, USB-C/USB peripherals)
```

Pairing flow:

```
1. Settings → Basic → Bluetooth
2. Toggle ON
3. "Add Device" / make phone discoverable
4. From other device, search and pair
5. Phone shows pairing PIN; enter on other device
```

## EHS Headset

Electronic Hook Switch (remote answer/end via headset button):

```xml
<feature
  feature.ehs.enable="1"
  feature.headset.useDefaultRingerOnly="0" />

<up
  up.headsetMode="0"           <!-- 0=normal, 1=headset always on -->
  up.echoCancellation.enable="1" />
```

Supported brands (UCS support varies by version):

```
Plantronics (Poly)   Most common; APD-80 / APV-66 cables
Sennheiser           CEHS-PO 01 cable
Jabra                LINK 14201-19 cable
Yealink              EHS36 cable
GN Netcom            GN1000 RHL
```

Cable connects to phone's RJ-9 / 2.5mm "EHS" port (varies by model).

## Phone-Side Phonebook

Local contacts XML (`<MAC>-directory.xml` or `contacts.xml`):

```xml
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<directory>
  <item_list>
    <item>
      <fn>Alice</fn>
      <ln>Smith</ln>
      <ct>2001</ct>
      <sd>1</sd>
      <lb>Reception</lb>
      <pt>1</pt>
      <bw>0</bw>
      <bb>0</bb>
      <ad>0</ad>
      <ar>0</ar>
      <rt>2</rt>
      <dc></dc>
      <ik>0</ik>
      <fi>0</fi>
    </item>
    <item>
      <fn>Bob</fn>
      <ln>Jones</ln>
      <ct>2002</ct>
      <sd>2</sd>
      <lb>Support</lb>
    </item>
  </item_list>
</directory>
```

Element keys:

```
fn   First name
ln   Last name
ct   Contact (SIP URI / number)
sd   Speed dial index (1=press#1 from idle)
lb   Label / display name
pt   Protocol (1=SIP)
bw   Buddy watching
bb   Buddy block
ad   Auto-divert
ar   Auto-reject
rt   Ringtone (1..N from region.cfg)
dc   Divert contact
ik   Instant Key
fi   Favorite Index
```

## LDAP / Corporate Directory

LDAP corporate directory:

```xml
<dir
  dir.corp.address="ldap.example.com"
  dir.corp.port="389"
  dir.corp.transport="TCP"           <!-- TCP / TLS -->
  dir.corp.baseDN="ou=people,dc=example,dc=com"
  dir.corp.user="cn=phone,ou=service,dc=example,dc=com"
  dir.corp.password="ldappass"
  dir.corp.bindOnInit="0"
  dir.corp.scope="sub"               <!-- sub / one / base -->
  dir.corp.searchFilter="(objectclass=person)"
  dir.corp.viewPersistence="1"

  dir.corp.attribute.1.name="givenName"
  dir.corp.attribute.1.label="First"
  dir.corp.attribute.1.type="first_name"
  dir.corp.attribute.1.searchable="1"

  dir.corp.attribute.2.name="sn"
  dir.corp.attribute.2.label="Last"
  dir.corp.attribute.2.type="last_name"
  dir.corp.attribute.2.searchable="1"

  dir.corp.attribute.3.name="telephoneNumber"
  dir.corp.attribute.3.label="Phone"
  dir.corp.attribute.3.type="phone_number"
  dir.corp.attribute.3.searchable="0"

  dir.corp.attribute.4.name="mail"
  dir.corp.attribute.4.label="Email"
  dir.corp.attribute.4.type="email"
  dir.corp.attribute.4.searchable="1" />
```

Microsoft AD specifics:

```xml
dir.corp.address="dc.example.local"
dir.corp.port="3268"                       <!-- Global Catalog -->
dir.corp.user="phone@example.local"        <!-- UPN form -->
dir.corp.searchFilter="(&(objectCategory=person)(objectClass=user)(telephoneNumber=*))"
dir.corp.baseDN="DC=example,DC=local"
dir.corp.transport="TLS"
dir.corp.port="3269"                       <!-- LDAPS GC -->
```

## Web Login Levels

Two levels:

```
admin   Full access. Default password 456.
user    Limited; can change personal settings (label, ringtone, BG).
        Default password 123.
```

Change passwords:

```xml
<device
  device.user.password="user-password-here"
  device.user.password.set="1"
  device.auth.localAdminPassword="newadminpw"
  device.auth.localAdminPassword.set="1" />
```

Web UI access scope:

```xml
<httpd
  httpd.cfg.enabled="1"
  httpd.cfg.port="80"
  httpd.cfg.secureTunnelEnabled="1"
  httpd.cfg.secureTunnelPort="443"
  httpd.cfg.secureTunnelRequired="1"      <!-- redirect all to HTTPS -->
  httpd.cfg.AccessControl.LocalAdmin="0"  <!-- 0=full, 1=read-only -->
  httpd.cfg.AccessControl.User="2" />     <!-- 0=full, 2=hide some -->
```

## Logging

Log levels (per-module):

```xml
<log
  log.level.change.cipher="3"
  log.level.change.copy="3"
  log.level.change.dapp="3"
  log.level.change.drvr="3"
  log.level.change.ec="3"
  log.level.change.efk="3"
  log.level.change.h323="3"
  log.level.change.hset="3"
  log.level.change.httpd="3"
  log.level.change.kbrd="3"
  log.level.change.lcd="3"
  log.level.change.ldap="3"
  log.level.change.lic="3"
  log.level.change.linelist="3"
  log.level.change.log="3"
  log.level.change.mb="3"
  log.level.change.mobi="3"
  log.level.change.netmon="3"
  log.level.change.niche="3"
  log.level.change.ntp="3"
  log.level.change.os="3"
  log.level.change.osd="3"
  log.level.change.pcap="3"
  log.level.change.pcd="3"
  log.level.change.pkihs="3"
  log.level.change.pres="3"
  log.level.change.prov="3"
  log.level.change.rdisk="3"
  log.level.change.rtos="3"
  log.level.change.sec="3"
  log.level.change.sip="4"
  log.level.change.slog="3"
  log.level.change.snmp="3"
  log.level.change.so="3"
  log.level.change.sshc="3"
  log.level.change.tcpip="3"
  log.level.change.term="3"
  log.level.change.usb="3"
  log.level.change.utilm="3"
  log.level.change.wbm="3"
  log.level.change.wifi="3"
  log.level.change.xmpp="3" />
```

Levels:

```
0   Show everything (debug)
1   Verbose
2   Informational
3   Warning (default)
4   Error
5   Critical
6   Off
```

Render formatting:

```xml
log.render.realtime.suite="1"
log.render.realtime.severity="3"
log.render.file="1"
log.render.file.severity="3"
log.render.type="0"
log.render.suite.length="6"
log.render.format="0"           <!-- 0=long, 1=short -->
log.render.timeStamp="2"
log.render.usec="1"
log.render.uniqueId="1"
log.render.eol=""
log.render.zone="local"
```

Syslog:

```xml
<device
  device.syslog.serverName="syslog.example.com"
  device.syslog.transport="UDP"      <!-- UDP / TCP / TLS -->
  device.syslog.facility="16"         <!-- local0 -->
  device.syslog.prependMac="1"
  device.syslog.renderLevel="3"
  device.syslog.serverName.set="1"
  device.syslog.transport.set="1" />
```

Upload logs to provisioning server (ad-hoc):

```
Web UI → Utilities → Phone Backup & Restore → Backup
  - Logs uploaded to LOG_FILE_DIRECTORY on provisioning server
  - Filename: <MAC>-app.log, <MAC>-boot.log
```

## Sample Production Config

Minimal sip-basic.cfg + per-MAC cfg:

`000000000000.cfg`:

```xml
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<APPLICATION
    APP_FILE_PATH="sip.ld"
    CONFIG_FILES="features.cfg, region.cfg, sip-basic.cfg, security.cfg"
    LOG_FILE_DIRECTORY="logs"
    OVERRIDES_DIRECTORY="overrides"
    CONTACTS_DIRECTORY="">
</APPLICATION>
```

`features.cfg`:

```xml
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<polycomConfig
  xmlns="urn:com:polycom:configuration"
  xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">

  <feature
    feature.busyLamp.enable="1"
    feature.bluetooth.enabled="1"
    feature.ehs.enable="1"
    feature.directedCallPickup.enable="1"
    feature.groupCallPickup.enable="1"
    feature.callPark.enable="1"
    feature.urlDialing.enabled="1"
    feature.dndPersistent.enabled="1" />

  <up
    up.formatPhoneNumbers="0"
    up.cnameLookup="1"
    up.audioMode="0" />

</polycomConfig>
```

`region.cfg`:

```xml
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<polycomConfig
  xmlns="urn:com:polycom:configuration"
  xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">

  <tcpIpApp
    tcpIpApp.sntp.address="time.example.com"
    tcpIpApp.sntp.gmtOffset="0"
    tcpIpApp.sntp.gmtOffsetCityID="050"
    tcpIpApp.sntp.daylightSavings.enable="1"
    tcpIpApp.sntp.daylightSavings.fixedDayEnable="0"
    tcpIpApp.sntp.daylightSavings.start.month="3"
    tcpIpApp.sntp.daylightSavings.start.dayOfWeek="1"
    tcpIpApp.sntp.daylightSavings.start.dayOfWeek.lastInMonth="1"
    tcpIpApp.sntp.daylightSavings.start.time="1"
    tcpIpApp.sntp.daylightSavings.stop.month="10"
    tcpIpApp.sntp.daylightSavings.stop.dayOfWeek="1"
    tcpIpApp.sntp.daylightSavings.stop.dayOfWeek.lastInMonth="1"
    tcpIpApp.sntp.daylightSavings.stop.time="2" />

  <feature
    feature.timezone="GMT"
    feature.localUI.timeFormat="0"
    feature.localUI.dateFormat="0" />

  <voice
    voice.codecPref.G711_A="1"
    voice.codecPref.G711_Mu="2"
    voice.codecPref.G722="3"
    voice.codecPref.Opus="0"
    voice.codecPref.G729_AB="0" />

</polycomConfig>
```

`security.cfg`:

```xml
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<polycomConfig>
  <device
    device.auth.localAdminPassword="changeMe123!"
    device.auth.localAdminPassword.set="1"
    device.user.password="userPass!"
    device.user.password.set="1" />

  <httpd
    httpd.cfg.secureTunnelRequired="1"
    httpd.cfg.secureTunnelPort="443" />

  <sec
    sec.TLS.protocol="TLSv1_2"
    sec.srtp.enable="1"
    sec.srtp.offer="1"
    sec.srtp.require="0" />
</polycomConfig>
```

`0004f2abcdef-phone.cfg` (per-MAC):

```xml
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<polycomConfig
  xmlns="urn:com:polycom:configuration"
  xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">

  <reg
    reg.1.address="2001@sip.example.com"
    reg.1.label="2001 Reception"
    reg.1.displayName="Reception"
    reg.1.auth.userId="2001"
    reg.1.auth.password="seCret123!"
    reg.1.outboundProxy.address="sbc.example.com"
    reg.1.outboundProxy.port="5060"
    reg.1.outboundProxy.transport="UDPOnly"
    reg.1.server.1.address="sip.example.com"
    reg.1.server.1.port="5060"
    reg.1.server.1.transport="UDPOnly"
    reg.1.server.1.expires="3600" />

  <attendant
    attendant.uri="blf-reception@sip.example.com"
    attendant.reg="1" />

</polycomConfig>
```

## Common Errors

Verbatim error strings and fixes:

```
"Reg Failed: 401 Unauthorized"
  Cause:  Wrong reg.X.auth.password
  Check:  pcap from web UI → SIP REGISTER → 401 challenge → re-REGISTER fails
  Fix:    Verify reg.X.auth.userId AND reg.X.auth.password; reload config

"Reg Failed: 403 Forbidden"
  Cause:  Server rejects this user (account disabled, IP not allowed)
  Fix:    Check ACL on server; verify user account active

"No Service: Could Not Reach Server"
  Cause:  Network/DNS/firewall, can't reach reg.X.server.1.address
  Fix:    ping/dig from phone (Diagnostics → Ping); verify SIP port open

"Lost Service: Will Retry in N Sec"
  Cause:  Was registered, lost connectivity (server down or NAT timeout)
  Fix:    Check server, verify keep-alive (nat.keepalive.interval=30)

"Cert Verify Failed"
  Cause:  TLS server cert untrusted; CA cert missing OR clock skew
  Fix:    1. Add CA to trust list (Settings → TLS)
          2. Verify time (NTP) — TLS rejects expired-from-future certs
          3. Set sec.TLS.profile.X.peerNameValidate=0 only for testing

"Provisioning: Auth Failure"
  Cause:  Wrong device.prov.user / device.prov.password
  Fix:    Verify with: curl -u user:pass https://prov/path/000000000000.cfg

"Reset to Factory Default"
  Cause:  Boot recovery triggered (1+3+5 held during boot, or admin selected)
  Result: All config wiped; phone re-provisions from DHCP

"Web UI: Auth Required"
  Cause:  Logged out; default 456 was changed
  Fix:    1. If known: log in.
          2. If unknown: factory reset (1+3+5 boot + 0123 menu)

"DHCP Failed"
  Cause:  No DHCP response on boot
  Fix:    1. Check switch/cable
          2. Check VLAN tag (device.net.vlanId / DHCP option 132)
          3. Static fallback via Settings → Network

"Phone Configuration Update Failed"
  Cause:  XML parse error in cfg file OR file unreachable
  Fix:    1. Validate XML: xmllint --noout file.cfg
          2. Check provisioning server logs for HTTP 404/403
          3. Web UI → Diagnostics → Logs → Boot for line number

"Configuration File Error: file too large"
  Cause:  Single cfg > 1 MB on some VVX models
  Fix:    Split into multiple cfg files; reference via CONFIG_FILES

"Firmware version mismatch"
  Cause:  APP_FILE_PATH points at wrong firmware blob for model
  Fix:    Use $model$ substitution: APP_FILE_PATH="firmware/{$model}.ld"

"Insufficient memory"
  Cause:  Too many BLF subscriptions or contacts
  Fix:    Lower attendant.resourceList count; trim contacts.xml

"User Not Authorized"
  Cause:  Web UI user-level account; needs admin
  Fix:    Log out, log in with admin (456)
```

## Common Gotchas

### 1. reg.X.auth.userId vs reg.X.address

```xml
<!-- WRONG: assumes auth.userId == address part -->
reg.1.address="2001@example.com"
<!-- but auth.userId not set, defaults to empty -->

<!-- RIGHT: explicitly set both -->
reg.1.address="2001@example.com"
reg.1.auth.userId="2001"
reg.1.auth.password="pass123"
```

Many SIP servers require auth.userId. UCS does NOT auto-derive.

### 2. outboundProxy vs SRV-routed address

```xml
<!-- WRONG: phone resolves SRV for sip.example.com THEN tries outbound -->
reg.1.outboundProxy.address="sbc.example.com"
reg.1.address="2001@sip.example.com"
reg.1.server.1.address="sip.example.com"
reg.1.server.1.transport="DNSnaptr"   <!-- triggers SRV first! -->

<!-- RIGHT: force exact transport -->
reg.1.outboundProxy.address="sbc.example.com"
reg.1.server.1.transport="UDPOnly"
```

### 3. Codec preference ordering

```xml
<!-- BROKEN: phone offers G.722.1C, server can't transcode → silence -->
voice.codecPref.G7221C_48kbps="1"
voice.codecPref.G711_Mu="0"

<!-- FIXED: always include G.711 baseline -->
voice.codecPref.G711_Mu="1"
voice.codecPref.G711_A="2"
voice.codecPref.G7221C_48kbps="3"
```

### 4. Default admin password 456 not changed

```xml
<!-- BROKEN: production phone with factory default -->
<!-- Anyone on LAN can access http://phone/ login: Polycom / 456 -->

<!-- FIXED: -->
<device
  device.auth.localAdminPassword="$omeStr0ngPw!"
  device.auth.localAdminPassword.set="1" />
```

### 5. "Polycom expects XML, not text" provisioning

```
BROKEN: prov server returns text/plain or HTML 404 page
        Phone fetches the file but XML parse fails
        Boot log: "configuration file error"

FIXED:  Server MUST send Content-Type: application/xml or text/xml
        AND the file MUST be valid XML with <?xml ... ?> declaration
        Test:  curl -v https://prov/000000000000.cfg | xmllint --noout -
```

### 6. UCS 5.x vs 6.x parameter renames

```
5.x parameter                        6.x parameter
---------------------------------    ---------------------------------
device.sntp.serverName               tcpIpApp.sntp.address (already same)
voIpProt.SIP.serverFeatureControl.SCA call.shared.subscribe
feature.callRecording.enabled        feature.callRecording.type
sec.tls.cipherSuiteDefault           sec.TLS.cipherSuiteDefault (case)

Solution: read the UC Software Configuration Reference for your exact
firmware version. Don't copy old configs blindly.
```

### 7. Missing CONFIG_FILES list

```xml
<!-- BROKEN: master cfg doesn't list child files -->
<APPLICATION
  APP_FILE_PATH="sip.ld" />
<!-- Phone fetches firmware but no sip-basic.cfg, no features.cfg -->

<!-- FIXED: -->
<APPLICATION
  APP_FILE_PATH="sip.ld"
  CONFIG_FILES="sip-basic.cfg, features.cfg, region.cfg" />
```

### 8. Time skew breaks TLS

```
SYMPTOM: "Cert Verify Failed" but cert is valid
CAUSE:   Phone clock years behind/ahead → cert NotBefore/NotAfter check fails
FIX:     1. Set NTP: tcpIpApp.sntp.address="pool.ntp.org"
         2. Allow phone to bootstrap NTP before TLS reg attempts
         3. Quick fix: set time manually via web UI to test
```

### 9. Two registrations on different servers competing

```xml
<!-- BROKEN: caller-ID flips between registrations -->
reg.1.address="2001@sip-internal.example.com"
reg.2.address="2001@sbc-external.example.com"
<!-- Phone registers BOTH; outbound calls hash by reg index -->

<!-- FIXED: pick one; or use reg.X.lineKeys to separate UI -->
reg.1.address="2001@sip-internal.example.com"
reg.1.lineKeys="2"
reg.2.address="2002-emergency@sbc.example.com"
reg.2.lineKeys="1"
```

### 10. VVX 600/601 BT not enabled by default

```xml
<!-- VVX 601 has BT hardware but feature must be enabled -->
feature.bluetooth.enabled="1"
bluetooth.connectable="1"
bluetooth.discoverable="1"
bluetooth.radioOn="1"
```

### 11. Recovery boot menu password

```
ENTER:    Hold 1+3+5 (some models 4+6+8+*) during power-on
PROMPT:   "Boot Block recovery menu"
PASSWORD: 456 (factory default) — same as admin web password
          OR 0123 if 456 has been changed
ESCAPE:   Power-cycle if you can't remember
```

### 12. Edge B/E vs VVX schema confusion

```
BROKEN: Tried to copy sip-basic.cfg from VVX 450 to Edge E450
        Edge OS rejects: "unknown parameter reg.1.auth.userId"

FIXED:  Edge OS uses NEW XML schema (different element/attribute names)
        Pull Edge OS template from Poly support site
        OR provision via Lens cloud (handles schema for you)
```

### 13. CONTACTS_DIRECTORY blank loses contacts

```xml
<!-- BROKEN: -->
<APPLICATION
  CONTACTS_DIRECTORY="" />
<!-- Phone won't push contacts.xml back to server on save -->

<!-- FIXED: -->
<APPLICATION
  CONTACTS_DIRECTORY="contacts" />
<!-- Or set per-MAC: <MAC>-directory.xml in same dir as <MAC>.cfg -->
```

### 14. dialplan.X.* not honoured

```xml
<!-- BROKEN: dialplan only applies to outbound from KEYPAD, not headset/dial-via-redial -->
dialplan.applyToTelUriDial="1"
dialplan.applyToUserDial="1"
dialplan.applyToUserSend="1"
dialplan.applyToForward="0"

<!-- FIXED: enable for all paths -->
dialplan.applyToTelUriDial="1"
dialplan.applyToUserDial="1"
dialplan.applyToUserSend="1"
dialplan.applyToForward="1"
dialplan.applyToDirectoryDial="1"
dialplan.applyToCallListDial="1"
```

### 15. Multiple polycomConfig roots in one file

```xml
<!-- BROKEN: only first <polycomConfig> parsed -->
<polycomConfig>...</polycomConfig>
<polycomConfig>...</polycomConfig>

<!-- FIXED: single root, all settings inside -->
<polycomConfig>
  <reg ... />
  <feature ... />
</polycomConfig>
```

## Diagnostic Tools

Web UI Diagnostics:

```
Diagnostics → Capture
  Start, place call / observe issue, Stop
  Download as <MAC>.pcap
  Open in Wireshark — full SIP + RTP

Diagnostics → Ping
  Inputs hostname/IP; phone runs ping; shows RTT/loss

Diagnostics → Traceroute
  Phone runs traceroute to target

Diagnostics → Logs → Application
  Live tail of UCS app log

Diagnostics → Logs → Boot
  Boot-time log (provisioning, network init)
```

PCAP from web UI is the fastest debug tool. Captures traffic on phone's interface — bypasses any LAN tap requirement.

Admin debug shell (limited):

```
Some VVX models: ssh root@<phone-ip>
  - Mostly disabled in production firmware
  - When available: limited busybox-style env
  - Not officially supported

# Enable shell access (where supported):
device.shell.enabled="1"
device.shell.password="<set strong pw>"
```

Syslog to remote:

```xml
<device
  device.syslog.serverName="syslog.example.com"
  device.syslog.transport="UDP"
  device.syslog.facility="16"
  device.syslog.renderLevel="3" />

# On syslog server:
# Filter by phone MAC (added as syslog tag if device.syslog.prependMac="1")
```

SIP message tracing (verbose):

```xml
log.level.change.sip="0"   <!-- 0 = full debug -->
log.level.change.dapp="0"
```

Phone-side menu diagnostics:

```
Settings → Status → Diagnostics → Audio Diagnostics
  Live RTP stream stats: jitter, packet loss, MOS estimate

Settings → Status → Network → SIP
  Per-line: registered? proxy? transport?

Settings → Status → Platform → Phone
  Model, MAC, IP, firmware, serial, board rev, license
```

## Sample Cookbook

### Asterisk reg

```xml
<reg
  reg.1.address="2001@asterisk.example.com"
  reg.1.label="2001"
  reg.1.auth.userId="2001"
  reg.1.auth.password="asteriskPw!"
  reg.1.outboundProxy.address="asterisk.example.com"
  reg.1.outboundProxy.port="5060"
  reg.1.outboundProxy.transport="UDPOnly"
  reg.1.server.1.address="asterisk.example.com"
  reg.1.server.1.port="5060"
  reg.1.server.1.transport="UDPOnly"
  reg.1.server.1.expires="120" />

<voIpProt
  voIpProt.SIP.local.port="5060"
  voIpProt.server.1.specialEvent.checkSync.alwaysReboot="0" />

<voice
  voice.codecPref.G711_A="1"
  voice.codecPref.G711_Mu="2"
  voice.codecPref.G722="3" />
```

### FreeSWITCH reg

```xml
<reg
  reg.1.address="2001@freeswitch.example.com"
  reg.1.auth.userId="2001"
  reg.1.auth.password="fsPw!"
  reg.1.outboundProxy.address="freeswitch.example.com"
  reg.1.outboundProxy.port="5060"
  reg.1.outboundProxy.transport="UDPOnly"
  reg.1.server.1.address="freeswitch.example.com"
  reg.1.server.1.expires="600" />

<voIpProt
  voIpProt.SIP.useRfc2543hold="0" />

<feature
  feature.dndPersistent.enabled="1" />
```

### BroadWorks reg

```xml
<reg
  reg.1.address="2001@as.broadworks.example.com"
  reg.1.label="2001"
  reg.1.auth.userId="2001@example.com"
  reg.1.auth.password="bwPw!"
  reg.1.outboundProxy.address="sbc.broadworks.example.com"
  reg.1.outboundProxy.port="5060"
  reg.1.outboundProxy.transport="UDPOnly"
  reg.1.server.1.address="as.broadworks.example.com"
  reg.1.server.1.port="5060"
  reg.1.server.1.transport="UDPOnly"
  reg.1.server.1.expires="3600"
  reg.1.serverFeatureControl.cf="1"
  reg.1.serverFeatureControl.dnd="1"
  reg.1.serverFeatureControl.signalingMethod="serviceMsWithEvent" />

<voIpProt
  voIpProt.SIP.specialEvent.checkSync.alwaysReboot="1"
  voIpProt.server.dhcp.available="0" />

<feature
  feature.broadsoftcallpark.enabled="1"
  feature.broadsoftDirectCallPickup.enabled="1"
  feature.broadsoftGroupCallPickup.enabled="1"
  feature.broadsoftACD.enabled="1"
  feature.callRecording.enabled="1" />
```

### Microsoft Teams (CCX series only)

```
1. CCX comes pre-flashed for Teams (CCX 400/500/600/700)
2. First boot: device prompts for Teams sign-in
3. Sign in via Microsoft 365 account or device-code flow
4. Teams Admin Center adds device to tenant inventory
5. No UCS XML cfg — use Teams Admin Center policies instead
```

For UCS-mode CCX (some SKUs):

```xml
<reg
  reg.1.address="2001@sip.example.com"
  reg.1.transport="TLS"
  reg.1.outboundProxy.transport="TLS" />

<feature
  feature.lyncIntegration.enabled="1" />
```

### Yealink-style migration target reg

```xml
<reg
  reg.1.address="user@domain"
  reg.1.auth.userId="user"
  reg.1.auth.password="pass"
  reg.1.outboundProxy.address="sbc.example.com"
  reg.1.outboundProxy.port="5061"
  reg.1.outboundProxy.transport="TLS"
  reg.1.server.1.transport="TLS"
  reg.1.server.1.expires="3600" />

<sec
  sec.srtp.enable="1"
  sec.srtp.offer="1"
  sec.srtp.require="0" />
```

## Trio Conference Specifics

Polycom Trio 8500 / 8800 — conference room phones.

Differences vs VVX:

```
Form factor:  Three-spoke conference podium (vs handset/desk)
Speakerphone: Multi-mic array (8500 = 3 mics, 8800 = 4 mics + 2 expansion)
Display:      5" color touch (8500), 5" color touch (8800)
Mics range:   Up to 20 ft pickup (8800 with expansion mics: 50 ft)
Audio band:   Wideband (G.722 / G.722.1C / Siren14/22) by default
HDMI:         8800 has HDMI in/out for video conferencing pairing
USB:          8800 USB for laptops to use phone as speakerphone
BT:           Both have Bluetooth — phones can join via BT
Touchscreen:  Full Android-style touch UI (vs VVX softkeys)
```

Trio config differences:

```xml
<!-- Trio uses same UCS XML schema, but additional params for conf -->
<feature
  feature.audio.acousticFenceEnabled="1"   <!-- noise gating -->
  feature.audioInputMonitor.enabled="1"
  feature.bluetooth.enabled="1"
  feature.usbHeadset.enabled="1" />

<voice
  voice.acousticFence.aft.0="0"
  voice.acousticFence.fadeOut.duration="700"
  voice.audioProfile.aacLd.headphone="-30" />

<bluetooth
  bluetooth.connectable="1"
  bluetooth.discoverable="1" />
```

Trio expansion microphone (8800 only):

```
Wired expansion mic plugs into RJ-45 on Trio 8800 base
Up to 2 expansion mics
Auto-detected
```

Pair phone to Trio via Bluetooth:

```
1. Trio screen → Settings → Bluetooth → Add Device
2. Trio enters discoverable mode
3. From phone Bluetooth menu, select "Trio 8800-XXXX"
4. Confirm PIN on both
5. Calls from phone can route through Trio speakerphone
```

Trio Visual+:

```
Optional accessory paired with Trio 8800
Adds dual HDMI display + camera for video conferencing
USB connection to Trio
```

## CCX Series

CCX = Microsoft Teams native phones.

Architecture:

```
- Android-based (Android 10+)
- Native Teams app pre-installed
- Touch screen (5" CCX 400/500, 7" CCX 600/700)
- Sign-in via Microsoft 365 account
- Teams Admin Center for device management
- NO traditional UCS XML provisioning by default
```

Sign-in flow:

```
1. Device boots → Teams login screen
2. Two methods:
   a. Web sign-in: tap "Sign in" → enter email/password directly
   b. Code sign-in: device shows code, user enters at microsoft.com/devicelogin
3. Tenant policy applied
4. Device shown in Teams Admin Center → Voice → Devices
```

CCX has dual personalities (depending on SKU):

```
Teams-only CCX:    Microsoft Teams app only, locked down
Open SIP CCX:      Same hardware, runs UCS UC Software with Polycom SIP stack
```

Switch modes (where supported):

```
Settings → Device Management → Base Profile
  Options: Teams / Generic SIP / Lync
  Reboot required after change
```

In SIP mode CCX uses standard UCS XML schema but with CCX-specific limitations:

```xml
<!-- Touch-only — no DSS keys, all UI through touch -->
<feature
  feature.lyncIntegration.enabled="0"
  feature.urlDialing.enabled="1" />

<reg
  reg.1.address="2001@sip.example.com"
  reg.1.transport="TLS"
  reg.1.label="2001" />
```

Teams admin policies (no XML — Teams Admin Center):

```
Configuration policies:
  - Sign-in / sign-out
  - Display screensaver
  - Audio + video defaults
  - Call queues / auto-attendants
  - Firmware update window
```

## Idioms

- **Always change default 456 password.** Then change it again.
- **Set provisioning to HTTPS.** Plaintext HTTP/TFTP exposes auth credentials.
- **DHCP Option 160 for Polycom.** It's the "their" option — works without server-type guessing.
- **Use SRV records for HA.** `_sip._udp.example.com SRV 0 100 5060 sbc1.example.com` lets phones load-balance.
- **PCAP from web UI is the fastest debug tool.** Bypasses LAN tap; captures all phone traffic.
- **Polycom == open architecture.** Standards-based SIP. Edge OS == new family with cloud-first defaults.
- **Master file lists, child files configure.** Don't put reg.X.* in 000000000000.cfg unless you mean global.
- **Per-MAC overrides last.** Use `<MAC>-phone.cfg` for one-off settings, leave templates clean.
- **Validate XML before deploying.** `xmllint --noout file.cfg` saves a 1000-phone reboot loop.
- **Time first, then TLS.** NTP must work or every TLS cert validation fails.
- **One root <polycomConfig>.** Not multiple — only first is parsed.
- **Avoid `transport="DNSnaptr"` if you have a SBC.** Force `UDPOnly` / `TCPOnly` / `TLS`.
- **Disable G.729 unless licensed.** Otherwise calls fall back to G.711 with confusing renegotiation.
- **Enable `device.syslog.prependMac="1"`.** Makes correlating events trivial across hundreds of phones.
- **For BroadWorks, set `serverFeatureControl.signalingMethod="serviceMsWithEvent"`.** Matches BroadSoft's expectations.
- **Reboot after major schema changes.** Some params take effect only on full restart, not "Apply".

## See Also

- ip-phone-provisioning
- sip-protocol
- asterisk
- freeswitch
- yealink-phones
- cisco-phones
- grandstream-phones
- snom-phones

## References

- support.polycom.com (legacy)
- support.poly.com
- support.hp.com/poly
- Polycom UC Software Administrators' Guide (per-version PDF)
- Polycom UC Software Configuration Reference (parameter list, per-version)
- support.polycom.com/PolycomService/support/us/support/voice/uc_software/
- RFC 3261 — SIP
- RFC 3262 — Reliability of provisional responses (PRACK)
- RFC 3265 — SIP Specific Event Notification
- RFC 3711 — SRTP
- RFC 4566 — SDP
- RFC 5763 — DTLS-SRTP
- Poly Lens — https://lens.poly.com
- HP/Poly Voice — https://voice.poly.com
- Polycom Resource Manager (RPRM) Admin Guide
- Edge OS Configuration Guide (Poly internal portal)
