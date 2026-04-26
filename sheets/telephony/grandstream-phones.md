# Grandstream Phones

Reference for Grandstream IP phones, ATAs, DECT, gateways, and UCM PBX — provisioning, SIP accounts, codecs, BLF/MPK, NAT, security, troubleshooting.

## Setup — Grandstream Portfolio

Grandstream Networks (Boston, MA, founded 2002) ships a wide telephony stack — desk phones, cordless DECT, ATAs, gateways, video, and PBX. All models speak SIP; all share a common web UI lineage; all support remote provisioning via TFTP/FTP/HTTP/HTTPS or the GDMS cloud.

```
GXP (older line, EOL drift)        — desk phones, basic to executive
GRP (current 6th gen)              — carrier-grade, replaces GXP
DP (DECT cordless)                 — base + handsets
WP (DECT IP phone, integrated)     — wireless deskphone-style
HT (Handytone ATA)                 — FXS/FXO analog adapter
GVC / GVS (video)                  — videoconferencing endpoints
GXW (gateway)                      — bulk FXS/FXO ports
HA series                          — wired/wireless headsets
UCM                                — Asterisk-based PBX (hardware + software)
```

### GXP — Older Desk Phone Line

Pre-GRP generation. Still in service worldwide; firmware updates ongoing for some models, EOL for others. Mostly monochrome screens, RJ45 100Base-TX (some Gigabit on higher SKUs).

```
GXP1610   — 2-line entry (no PoE, 132×48 LCD)
GXP1620   — 2-line + PoE
GXP1625   — 2-line + PoE + extra line key
GXP1628   — 2-line + Gigabit
GXP1630   — 3-line + Gigabit + PoE
GXP2130   — 3-line, color 2.8" LCD, BT (some rev)
GXP2140   — 4-line, color, 8 BLF, BT
GXP2160   — 6-line, color, 24 BLF
GXP2170   — 12-line, color 4.3" LCD, 48 multi-purpose keys
```

Common: 2 SIP accounts (entry) up to 6 SIP accounts (executive); HD voice (G.722); 5-way conference.

### GRP — Current Carrier-Grade Line

6th-generation. Replaces GXP. Color screens, Gigabit standard, USB, Wi-Fi/BT optional via dongle, more multi-purpose keys, GDMS-ready.

```
GRP2601   — 2-line entry (no PoE)
GRP2602   — 2-line entry + PoE
GRP2612   — 4-line, color 2.4" LCD, PoE
GRP2614   — 4-line, color, 4 BLF, BT, USB
GRP2615   — 10-line, color 2.8" LCD, BT, USB, dual-color BLF
GRP2616   — 16-line, dual color screens (3.5" + 2.4" sidecar)
GRP2624   — 8-line + BT/Wi-Fi, 24 BLF, color 4.3"
GRP2634   — 8-line + dual screens, hi-end
GRP2636   — 16-line + larger screen, executive
```

Up to 16 SIP accounts (top SKU). Codecs include Opus on most GRP. Native VPN client (OpenVPN). LDAP. EHS headset support.

### DP — DECT Cordless

DECT base station + handset. Single base supports multiple handsets and SIP accounts.

```
DP750    — base, 5 handsets, 5 SIP accounts, 4 concurrent calls, range ~300m
DP752    — base, 10 handsets, 10 SIP accounts, 5 concurrent calls (newer)
DP720    — handset for DP750/752; mono screen, 1.8" LCD
DP722    — handset; color 1.8", BT, headset jack
DP730    — premium handset; 2.4" color, vibrate, longer battery
DP732    — newer premium handset
```

Repeaters: DP760 — extends DECT range. Up to 10 repeaters per DP752.

### WP — DECT IP Phone (Integrated)

Self-contained DECT phone (no separate base required for some setups; can also register to DP base). Looks like a cordless deskphone with a screen.

```
WP810    — 2 SIP accounts, 1.8" color, BT, ~30hrs talktime
WP820    — 2 SIP accounts, 2.4" color, BT, Wi-Fi, push-to-talk
WP822    — 2 lines, color 2.4", Wi-Fi, BT, USB-C (newer)
WP825    — 2 lines, larger screen, hardened, IP67 (rugged)
```

### HT — Handytone Analog Telephone Adapter

Converts SIP to analog (FXS for analog phones; FXO for PSTN line).

```
HT801    — 1× FXS, 1× LAN
HT802    — 2× FXS, 1× LAN
HT812    — 2× FXS, 2× LAN (router built-in), GbE
HT813    — 1× FXS + 1× FXO (PSTN failover), 1× LAN
HT814    — 4× FXS, 2× LAN (router), GbE
HT818    — 8× FXS, 2× LAN (router), GbE
```

Each FXS port = 1 SIP account. HT813 FXO port allows analog PSTN as fallback or as inbound trunk.

### GVS / GVC — Video

Video conferencing endpoints with H.264/H.265, multi-party, integrated camera.

```
GVC3210  — Android-based, 1080p, BT, MIRACAST, IPVT cloud
GVC3220  — newer, 4K camera, H.265
GVC3230  — high-end, 4K, multi-screen, AI features
```

### WP / HA Headsets

Wireless DECT/Bluetooth headsets, EHS hookswitch integration.

### GXW — Analog Gateway

Bulk FXO/FXS gateway for connecting many analog phones or PSTN trunks.

```
GXW4216    — 16× FXS
GXW4224    — 24× FXS
GXW4232    — 32× FXS
GXW4248    — 48× FXS
GXW4504    — 4× FXO
GXW4508    — 8× FXO
GXW4516    — 16× FXO
GXW4524    — 24× FXO
```

Also FXO models for PSTN trunk concentration.

### UCM PBX

Grandstream's own PBX — hardware appliance running an Asterisk-based stack with a Grandstream web UI.

```
UCM6202    — small, 2× FXO + 2× FXS
UCM6204    — 4× FXO + 2× FXS
UCM6208    — 8× FXO + 2× FXS
UCM6301    — newer entry, 1× FXO + 1× FXS
UCM6302    — 2/2
UCM6304    — 4/4
UCM6308    — 8/8
UCM6510    — E1/T1 + analog, larger
UCM6510A   — clustered/HA variant
```

UCM ships with auto-provisioning for Grandstream phones (zero-config) — pair a GRP/GXP/DP with a UCM and the UCM will discover and configure the phone over LAN.

## Default Access

```
URL              http://<phone-ip>/
admin password   admin / 123456
user password    user / 123
```

Yes, the default admin password is literally `123456` — six digits. (Some newer firmware images now ship with a unique password printed on a sticker; check the box.)

Toggle web access:

```
Phone keypad   → MENU
Settings       → Maintenance → Web Access
Options        : HTTP / HTTPS / Both / Disabled
Default port   : 80 (HTTP) / 443 (HTTPS)
```

After first login the UI typically prompts to change the default password — strongly recommended.

User-level credentials (`user / 123`) only see Status and a subset of Account/Settings; admin sees everything including Maintenance.

### Default Credential Reference (Fleet)

```
Component               Default Username   Default Password   Notes
GXP/GRP web UI admin    admin              123456             newer SKUs unique-per-device
GXP/GRP web UI user     user               123                limited Status-only view
DP750/DP752 base        admin              admin              admin/admin not admin/123456
HT8xx ATA               admin              admin              also unique sticker on newer
WP8xx                   admin              admin
GVC/GVS3xxx Android     admin              admin              also has Google account login
UCM63xx PBX (web)       admin              <serial>           random 8-char serial-derived
UCM61xx PBX (web)       admin              admin              older default
UCM SSH                 root               admin              hardened off in 1.0.20+
GDMS portal             user-defined       user-defined       cloud account; MFA optional
DECT subscription PIN   <none>             0000               4-digit DECT pairing
Phone screen unlock PIN <none>             123                user-level menu lock
```

P-id mapping (web access toggle):

```
P196   web access mode      0=HTTP & HTTPS, 1=HTTPS only, 2=HTTP only, 3=Disabled
P3637  HTTP web port        default 80
P3638  HTTPS web port       default 443
P22    user-level password  default "123"
P2     admin-level password default "admin" (legacy) / "123456" (modern GXP/GRP)
P1     admin-account name   default "admin"
P3406  IP whitelist for web access (CIDR list, comma-separated)
P273   web access expiry    auto-logout idle session after N seconds (default 600)
```

### Forgotten Password Recovery

```
GXP/GRP    Hold * + 9 during boot          → factory reset (10s power-cycle)
DP base    Press subscribe button >5s      → factory reset
HT8xx      Hold reset pinhole >7s          → factory reset (20s power-cycle)
WP8xx      Menu → Factory Reset (PIN req)  OR  power+volume_down combo
UCM        Console serial 115200 8N1, "RESET" boot prompt → admin/admin
```

After factory reset, the phone re-fetches provisioning if DHCP options 66/43/160 are set — so reset alone may not actually wipe a managed device long-term unless GDMS/provisioning binding is also cleared.

## Web UI Tour

Six top-level sections (some models add Phonebook):

```
Status          Account/Network/System status, line state, RTP stats
Account         Account 1..N — SIP server, creds, codecs, advanced
Settings        General, Call Features, Programmable Keys, MPK, Web Service, Audio
Network         Basic (DHCP/static), Wi-Fi, OpenVPN, Advanced (QoS, VLAN)
Maintenance     Web Access, Upgrade, Syslog, TR-069, Tools (Ping/Traceroute), Capture
Phonebook       XML phonebook, LDAP, Group, BroadSoft directory
```

Status → Account Status shows registration state per account. Status → Network shows live IP, gateway, DNS, MAC. Maintenance → Tools includes Ping, Traceroute, NSLookup — handy when no shell is available.

## Account Configuration

Each account is configured under `Account` → `Account 1`/`Account 2`/etc.

Key fields:

```
Account Active                : Yes / No
Account Name                  : Display name (UI label)
SIP Server                    : sip.example.com or IP
Secondary SIP Server          : failover registrar
Outbound Proxy                : proxy.example.com (optional)
Backup Outbound Proxy         : 
SIP User ID                   : 1001
Authenticate ID               : 1001 (often same as user ID)
Authenticate Password         : ********
Name                          : Display name in From header
Voice Mail Access Number      : *97 / *98
SIP Registration              : Yes
Unregister On Reboot          : Yes
Register Expiration           : 60 (minutes; per-firmware unit varies — some are seconds)
Re-register before Expiration : 0 means default behavior
Wait Time Retry Reg Failure   : 20 (seconds)
Local SIP Port                : 5060 (Account 1), 5062 (Account 2), etc.
Local RTP Port                : 5004 base
Use Random Port               : No (recommended — predictable for firewalls)
```

Watch the unit on `Register Expiration` — newer firmware uses minutes, older uses seconds. Check the tooltip. Setting it to `0` means "use default" (typically 60 minutes / 3600 seconds).

## Multiple SIP Accounts

Each model has a fixed account count:

```
GXP1610          2
GXP2160         24 (paginated multi-account; effective concurrent ~6)
GRP2602          2
GRP2614          4
GRP2615         10
GRP2616         16
GRP2636         16
DP750            5
DP752           10
WP820            2
HT802            2 (one per FXS)
HT818            8
```

Each account = independent registrar, independent creds, independent codec settings. Calls can ring on any account; outgoing call routing chooses account by line key, dial-prefix MPK, or default account.

```
Default Account            : Account 1
Use # as Send Key          : Yes
Auto Dial Account          : Account 1 (when using long-press digit, etc.)
```

Account selection idioms:

- Dial out from Account 2: press the Account 2 line key first, then dial.
- BLF subscribed via the account hosting the monitored extension.
- Call waiting/hold per-account.

## Codec Configuration

Per-account: `Account N` → `Codec Settings`.

```
Preferred Vocoder
  Choice 1   : Opus
  Choice 2   : G.722
  Choice 3   : PCMU
  Choice 4   : PCMA
  Choice 5   : G.729A
  Choice 6   : G.726-32
  Choice 7   : iLBC
  Choice 8   : G.722.1
```

Codec list per model:

```
GXP older       PCMU PCMA G.722 G.726-32 G.729A iLBC
GRP series      PCMU PCMA G.722 G.722.1 G.726-32 iLBC G.729A Opus
DP series       PCMU PCMA G.722 G.726-32 G.729A
HT series       PCMU PCMA G.722 G.726-32 G.729A iLBC G.723.1
GVC series      PCMU PCMA G.722 Opus + H.264/H.265 video
```

Other knobs:

```
Voice Frames per TX        : 2 (per RTP packet; higher = lower bandwidth, higher latency)
G.726-32 Packing Mode       : ITU / IETF
iLBC Frame Size             : 20 ms / 30 ms
Opus Payload Type           : 123 (default)
Use First Matching Vocoder  : Yes (use call originator's first preference)
DTMF                        : RFC2833 / SIP INFO / In-audio
DTMF Payload Type           : 101
SRTP Mode                   : No / Optional / Required
SRTP Key Length             : AES 128/192/256
```

`Voice Frames per TX = 2` for G.711 → 40 ms packets (saves bandwidth). For latency-sensitive, set to 1 → 20 ms. iLBC and Opus have their own framing.

## Programmable Keys

Multi-Purpose Keys (MPK) — physical keys flanking the screen, plus virtual keys via expansion module (GBX20, GXP2200EXT, etc.). Configurable types:

```
None
Speed Dial                       — calls a number on press
Busy Lamp Field (BLF)            — monitors extension; LED + press to call/pickup
Presence Watcher                 — RFC3856 presence (busy/away/idle)
Eventlist BLF                    — single SUBSCRIBE for many BLFs
Speed Dial via Active Account    — dial via the account currently selected
Speed Dial via Specific Account  — bind to a chosen account
Dial DTMF                        — sends DTMF mid-call
Voicemail                        — dials VM access for chosen account
Call Return                      — last caller redial
Transfer                         — blind transfer to preset
Call Park                        — park call to preset park slot
Intercom                         — auto-answer intercom to extension
LDAP Search                      — opens LDAP query
Conference                       — adds party to conference
Multicast Paging (Listen)        — listen on multicast group
Multicast Paging (Send)          — page to multicast group
Record                           — call recording trigger
Call Log                         — opens call history
Menu                             — opens phone menu
XML Application                  — custom XML app URL
Information                      — phone info
Headset                          — toggle headset mode
DND                              — Do Not Disturb toggle
Redial                           — last dialed
Pickup                           — directed call pickup
Pickup by Account                — bound to specific account
Forward                          — call forward toggle
Dial Prefix                      — prepends digits before dialing
Hold                             — hold/unhold
Mute                             — mic mute
Speaker                          — speakerphone
Account                          — switch active account
```

Configuration path: `Settings` → `Programmable Keys` → `MPK Settings` (or `Line Keys` / `Soft Keys` / `Side Keys` depending on model).

Each key has:

```
Key Mode      : (one of the above)
Account       : Account N (for line/BLF/speed-dial)
Description   : label shown next to LED
Value         : the number / extension / multicast IP / preset
```

## BLF — Busy Lamp Field

Subscribes via `SUBSCRIBE` (Event: `dialog`) to the registrar; shows the monitored extension's call state via LED + soft-label.

```
Key Mode      : Busy Lamp Field (BLF)
Account       : Account 1
Description   : Reception
Value         : 1100        ← extension to monitor
```

LED state mapping:

```
Off              — extension idle
Solid (red/green model dependent)  — extension on a call
Slow blink       — extension ringing (incoming) — pickup-able
Fast blink       — call alerting / setup
```

Press a BLF key:

- Idle: places call to monitored ext.
- Ringing: directed pickup (`*8` style or BLF pickup feature, depending on PBX).
- On call: barge / monitor / call (PBX feature dependent).

Per-PBX behavior varies — Asterisk/FreePBX needs `Hint` extensions; UCM auto-handles BLF when paired.

## Eventlist BLF

Standard BLF subscribes once per monitored ext — 50 BLFs = 50 SUBSCRIBE dialogs. Eventlist (RFC4662) lets the server publish a *list* and the phone subscribes to it once.

```
Settings → Call Features → Eventlist BLF URI : sip:my-blf-list@pbx.example.com
Programmable Keys → set MPK type to "Eventlist BLF" (NOT plain BLF)
Description     : Reception
Value           : 1100  (extension included in eventlist)
```

PBX side: a resource list named `my-blf-list` with members `1100, 1101, 1102, ...`. Server replies to one SUBSCRIBE with NOTIFYs aggregating state for all listed extensions.

Saves load on registrar; some PBXes (BroadWorks, FreePBX with mod_eventlist, UCM) support it natively.

## Multicast Paging

One-way audio to a group of phones via IP multicast (no SIP signaling). Useful for overhead paging, mass announcements.

```
Settings → Multicast Paging
Paging Barge        : 0 (disabled) or 1-10 (priority threshold)
Paging Priority     : Active

Listen Address 1    : 237.11.10.11:6767       Label: All Page
Listen Address 2    : 237.11.10.12:6767       Label: Sales
Listen Address 3    : 237.11.10.13:6767       Label: Floor 2
```

Send via MPK:

```
Key Mode    : Multicast Paging (Send)
Value       : 237.11.10.11:6767
Codec       : PCMU (must match listeners)
```

Listen via MPK:

```
Key Mode    : Multicast Paging (Listen) / "Listen to multicast IP"
Value       : 237.11.10.11:6767
```

Multicast group must be reachable across the L2/L3 path — IGMP snooping enabled on switches; PIM if crossing routers; or keep all paging-enabled phones on one VLAN.

`Paging Barge` priority: lower number = higher priority. Active call with priority 5 will be barged by an incoming page on priority 3, but not by priority 7.

Combine with intercom: intercom = SIP-signaled auto-answer to single ext; multicast paging = best-effort one-way to many.

## Provisioning

Two flavors:

1. Manual — log into web UI, configure account, save.
2. Auto — phone fetches a config file from a server on boot.

Auto-provisioning settings (`Maintenance` → `Upgrade and Provisioning`):

```
Firmware Server Path           : fw.example.com
Config Server Path             : provision.example.com/cs/$mac
Config Upgrade via             : HTTP / HTTPS / TFTP / FTP / FTPS
Firmware Upgrade via           : (same)
HTTP/HTTPS Username            : 
HTTP/HTTPS Password            : 
Always Authenticate Before Challenge : Yes / No
Validate Certificate Chain     : Yes (HTTPS)
Validate Hostname              : Yes
Custom Config File Name        : (optional override; default: cfg<MAC>.xml or cfg<MAC>)
Download Mode                  : Always Check / When Different / Skip
Allow DHCP Option 43/66        : Yes
Allow DHCP Option 120          : Yes
3CX Auto Provision             : No
Automatic Upgrade              : Yes (every N minutes / hour-of-day)
Upgrade Check Interval         : 10080 (minutes; weekly default)
```

Boot sequence (typical):

```
1. DHCP request               → IP, gateway, DNS, plus options 66/43/160
2. NTP sync
3. Fetch config:
   a. Resolve "Config Server Path"
   b. Try cfg<MAC>.xml         (per-device XML)
   c. Fall back to cfg<MAC>    (per-device text)
   d. Fall back to cfg.xml     (model/global)
4. Apply config; re-register accounts
5. Optional firmware upgrade
```

File request URL pattern:

```
http://provision.example.com/cs/cfg<MAC>.xml
http://provision.example.com/cs/cfg<MAC>
http://provision.example.com/cs/cfgaabbccddeeff.xml   (lowercase MAC)
```

Some firmware variants use `cfg<MAC>` (no extension) for binary, `cfg<MAC>.xml` for XML. Check `Maintenance` → `Tools` → `Trace Configuration File Download` to see which URL is fetched.

## Configuration File Format

Three formats over time:

```
P-tag text     : P1=hello\nP2=world\n        (older firmware; key = P-id integer)
XML            : <gs_provision> ... </gs_provision>   (newer; current default)
Binary CFG     : compiled cfg.bin (key obfuscation; built via Grandstream Configuration Tool)
```

Older GXP models (GXP21xx pre-1.0.10.x) prefer text. Modern GRP and HT firmware default to XML. The binary `.bin` wraps either format with light obfuscation — historically used to hide credentials in transit before HTTPS provisioning was common.

## Sample Text-Format Config

P-id mapping documented in each model's "Administration Guide" appendix.

```
# cfg<MAC> for GXP2170

# Network
P8=0                  # DHCP enabled
P30=10                # WAN port mode

# Account 1
P271=1                # Account 1 active
P47=sip.example.com   # SIP server
P35=1001              # SIP user ID
P36=1001              # Authenticate ID
P34=secret-pw         # Authenticate password
P3=Reception          # Display name
P33=*97               # Voicemail access number

# Account 1 - Codec
P57=18                # Codec choice 1: 18 = G.729A
P58=0                 # Codec choice 2:  0 = PCMU
P59=8                 # Codec choice 3:  8 = PCMA
P60=9                 # Codec choice 4:  9 = G.722

# NAT
P52=0                 # NAT Traversal: 0=No, 1=STUN, 2=KeepAlive, 3=UPnP

# Provisioning
P192=fw.example.com
P237=provision.example.com/cs
P212=1                # Config: 0=TFTP, 1=HTTP, 2=HTTPS

# Time zone / NTP
P64=US/Eastern
P30=time.cloudflare.com
```

P-id 1 = admin password (don't ship it in plaintext — encrypt the channel via HTTPS, or use binary cfg.bin).

## XML Config Format

Modern firmware uses XML. Schema slightly varies per model; the structure is consistent.

```xml
<?xml version="1.0" encoding="UTF-8"?>
<gs_provision version="1">
  <config version="2">
    <!-- Account 1 -->
    <P271>1</P271>
    <P47>sip.example.com</P47>
    <P35>1001</P35>
    <P36>1001</P36>
    <P34>secret-pw</P34>
    <P3>Reception</P3>

    <!-- Codec -->
    <P57>9</P57>     <!-- G.722 -->
    <P58>0</P58>     <!-- PCMU -->
    <P59>8</P59>     <!-- PCMA -->

    <!-- NAT -->
    <P52>1</P52>
    <P77>stun.example.com</P77>

    <!-- Provisioning -->
    <P192>fw.example.com</P192>
    <P237>provision.example.com/cs</P237>
    <P212>2</P212>

    <!-- Programmable keys -->
    <P301>0</P301>   <!-- MPK1 mode: BLF -->
    <P302>1</P302>   <!-- MPK1 account -->
    <P303>1100</P303><!-- MPK1 value -->
    <P304>Reception</P304><!-- MPK1 description -->
  </config>
</gs_provision>
```

Newer GRP and UCM-paired phones expose a more verbose schema (named tags instead of P-ids), but P-ids remain the canonical underlying representation.

Per-line-key arrays (`pvalue` / `account` arrays) appear in some firmware:

```xml
<lineKey>
  <key index="1">
    <mode>0</mode>          <!-- 0=Line -->
    <account>1</account>
    <description>Line 1</description>
    <value></value>
  </key>
  <key index="2">
    <mode>3</mode>          <!-- 3=BLF -->
    <account>1</account>
    <description>Reception</description>
    <value>1100</value>
  </key>
</lineKey>
```

## GDMS

Grandstream Device Management System. Cloud-based zero-touch provisioning + monitoring + alerting + remote control.

```
URL                    https://www.gdms.cloud/
Account                free tier (up to N devices)
Bound by               MAC address
Reseller channels      separate "GDMS Reseller" portal
```

Workflow:

1. Buy phone with GDMS-eligible firmware (most current GRP/DP/HT/WP).
2. In GDMS, register your account; add device by MAC + serial (or scan QR on box).
3. In GDMS, define a Site/Group with template (SIP server, codec list, MPK, etc.).
4. Plug phone into any Internet-connected LAN.
5. Phone boots → contacts Grandstream redirector → redirector points to GDMS → phone fetches config from GDMS HTTPS provisioning.
6. Phone auto-registers; GDMS shows online.

GDMS-as-redirector flow:

```
Phone boot → DNS/DHCP → no Option 66/160 → contact fm.grandstream.com (built-in default)
fm.grandstream.com → consult GDMS DB → return GDMS config URL
Phone → fetch GDMS config → register → send heartbeat to GDMS
```

GDMS features:

```
Site/group templates           — bulk apply config
Per-device override            — exception per phone
Firmware management            — staged rollout
Remote reboot / factory reset
SIP account auto-provisioning  — push creds without manual web UI touch
Live status / heartbeat
Push log/PCAP capture          — pull diagnostic from device
Voice quality reporting        — per-call MOS/jitter
Alert rules                    — offline > N minutes, etc.
```

GDMS-controlled phones still allow local web UI by default; toggle "Lock to GDMS" if you want config drift prevented.

## UCM Integration

UCM = Grandstream's PBX. Sold standalone or paired with phones.

When a UCM is on the same LAN as Grandstream phones, the UCM auto-detects them via:

```
PnP SUBSCRIBE (multicast)        — phone sends multicast SUBSCRIBE on boot
LLDP-MED                         — UCM advertises voice VLAN, phone tags
DHCP Option 66                    — DHCP server points to UCM provision URL
ARP scan                          — UCM scans LAN for Grandstream MAC OUI
```

UCM zero-config:

```
1. UCM scans LAN, finds Grandstream MAC OUIs
2. UCM offers to assign extension to discovered phones
3. Click "Create Extension" → UCM generates SIP creds + provision file
4. UCM posts config to phone via TR-069-style push (or phone-pull on boot)
5. Phone registers; UCM shows online status in Phone List
```

UCM also handles:

```
BLF hints                       — auto-generated for all extensions
Eventlist BLF                   — auto-list per group
Park slots                      — *6X codes
Pickup groups                   — *8 / **1 codes
Voicemail                       — *97 / *98
Call recording                  — server-side
SIP/IAX2 trunk                  — to other PBX
PSTN trunk                      — via on-board FXO
```

Combined with GDMS, UCM offers full zero-touch even for off-site phones — phone provisions from GDMS, registers to UCM via Internet (typically via OpenVPN tunnel from UCM-deployed certs, also auto-provisioned).

## NAT Settings

Per-account: `Account N` → `Network Settings`.

```
NAT Traversal
  No                  — no NAT logic; for LAN phones with no NAT
  STUN                — use STUN server for public IP discovery
  Keep-alive          — periodic packet to keep NAT pinhole open
  UPnP                — request port forward from UPnP-capable router
  Auto                — combine STUN + Keep-alive
  VPN                 — assume VPN tunnel; no NAT logic on tunnel side

STUN Server                : stun.example.com:3478
Keep-alive Interval        : 20 (seconds)
Use NAT IP                 : (override; manual public IP)
SIP T1 Timeout             : 0.5s default
SIP T2 Interval            : 4s
SIP Timer D Interval       : 0
SIP REGISTER Behind NAT    : Yes (rport, contact rewriting)
Use Symmetric RTP          : Yes (recommended behind NAT)
```

`Use NAT IP`: phone embeds this IP in SIP `Contact` and SDP — used when behind a 1:1 NAT (DMZ host). Otherwise STUN does the same dynamically.

`Symmetric RTP` = phone listens for RTP from wherever the remote sent it from (not from the SDP-advertised port). Critical for NAT — most servers send RTP to the source address of the first received packet.

## SIP-over-TLS

Encrypted SIP transport.

```
Account → SIP Settings
SIP Transport         : UDP / TCP / TLS
TLS Version           : TLS 1.0 / 1.1 / 1.2 / 1.3 (recommend 1.2+)
Mutual TLS (mTLS)     : Yes / No
Validate Certificate  : Yes (verify server cert)
TLS Cipher Suites     : (default list / custom)
Local SIP Port (TLS)  : 5061
```

Cert install:

```
Maintenance → Upload Cert → Trusted CA Cert
Maintenance → Upload Cert → Custom Cert (mTLS device cert)
Maintenance → Upload Cert → Private Key (matching device cert)
```

Format: PEM. Multiple CA certs concatenated in one file for chains.

After upload, reboot — TLS contexts re-init on boot for some firmware versions.

## SRTP

Encrypted media. SDES-based: keys exchanged in SDP `a=crypto:` lines. Requires SDP exchange to be itself encrypted (via SIP-over-TLS) for end-to-end secrecy.

```
Account → SIP Settings
SRTP Mode
  Disabled              — RTP only; reject any SRTP offer
  Optional / Negotiated  — accept SRTP if offered, else fall back to RTP
  Required               — only SRTP; reject RTP-only

SRTP Key Length         : AES-128 / AES-192 / AES-256
Crypto Suite            : AES_CM_128_HMAC_SHA1_80 (default)
                          AES_CM_128_HMAC_SHA1_32
                          AES_256_CM_HMAC_SHA1_80
```

ZRTP (key exchange in media) supported on some models — toggle in same panel.

`Required` + a server that doesn't speak SRTP = call setup `488 Not Acceptable Here` or `415 Unsupported Media`.

## DHCP Options

Phone can take provisioning hints from DHCP:

```
Option 66        TFTP server hostname/IP        (older convention)
Option 67        Boot file name
Option 43        Vendor-specific (encapsulated; opcode + length + data)
Option 120       SIP servers (RFC3361)
Option 132       VLAN ID (voice VLAN; not std)
Option 160       HTTP/HTTPS provisioning URL    (Grandstream/some other vendors)
```

Per-firmware preference order (typical):

```
1. Option 66    (if present, use as TFTP server)
2. Option 43    (if Grandstream sub-option present, use HTTP URL)
3. Option 160   (HTTP/HTTPS URL — preferred when present)
4. Manual configuration ("Config Server Path" in web UI)
5. GDMS redirector
```

Newer firmware lifts Option 160 above Option 66 — Option 160 carries a full URL (`http://prov.example.com/cs/`) vs Option 66 just an IP/hostname for TFTP.

Option 43 vendor-encap example (server config):

```
# ISC dhcpd
option space gs;
option gs.url code 1 = text;
class "grandstream" {
  match if substring(option vendor-class-identifier, 0, 11) = "Grandstream";
  vendor-option-space gs;
  option gs.url "http://prov.example.com/cs/";
}
```

## PoE / VLAN

Most desk phones (GXP1620+, GRP series) accept 802.3af PoE (15.4W class). Some draw class 0 by default — explicit `Class` setting in `Network` → `Advanced` to negotiate Class 1/2/3 if switch supports it.

VLAN:

```
Network → Advanced
Layer 2 QoS — 802.1Q/VLAN Tag        : 100   (voice VLAN ID)
Layer 2 QoS — 802.1p Priority Value  : 5     (voice priority)

PC Port Layer 2 QoS — 802.1Q VLAN     : 0    (untagged for PC port pass-through)
PC Port Mode                          : Bridge / Mirror / NAT
```

LLDP-MED (auto-VLAN):

```
LLDP                : Yes
LLDP-MED Network Policy : Yes
```

If switch advertises voice VLAN via LLDP-MED, phone auto-tags. Otherwise set VLAN manually.

PC port: most desk phones have a passthrough switchport. Set PC port VLAN = 0 (untagged) to plumb a workstation through the phone on the data VLAN while phone tags voice VLAN.

## DECT (DP / WP)

DP base station + DP/WP handsets.

Pairing:

1. On base: hold the registration button until LED blinks (or web UI → "Subscribe" mode).
2. On handset: Menu → Settings → Registration → Register → select base → enter PIN (default `0000`).
3. Wait for handset display "Registered" / handset number assigned.

```
DP750 Web UI → DECT → Base Station Settings
Base Station Name       : MainOffice
DECT PIN Code           : 0000   (CHANGE THIS)
Country / Frequency      : US (1880-1900) / EU (1880-1900) / etc.

DP750 Web UI → DECT → Handset Line Settings
HS1   Account 1, 2, 3    (which SIP accounts can be used outgoing/incoming)
HS2   Account 2, 3
...
```

Each handset can have multiple lines. Multiple handsets can share a line (last-pickup-wins or simultaneous ring depending on settings).

Concurrent calls: DP750 = 4, DP752 = 5. Beyond that, additional incoming calls busy out.

Range: line of sight ~300m; in-building ~50m typical (steel/concrete reduces). Repeaters extend.

WP series: built-in DECT base + handset combo. Pair to existing DP base for site-wide roaming (some firmware).

## ATA (HT series)

Analog telephone adapter. FXS port = phone-side; analog phone (cordless base, fax, alarm panel, doorphone) plugs in.

```
Web UI → FXS Port 1
Account Active             : Yes
SIP Server                 : sip.example.com
SIP User ID                : 2001
Authenticate Password      : ********

Audio / Voice
Caller ID Scheme           : Bellcore (US) / ETSI (UK/EU FSK) / DTMF (UK/Cable)
Distinctive Ringtone        : (per-account ring cadence)

Telephony / Tones
Country / Region            : United States / United Kingdom / Germany / France / ...
                              (sets dial tone, ring cadence, busy tone freq/duration)
Dial Plan                   : { x. | *xx | *xx*x. | [2-9]xxxxxx | 1[2-9]xxxxxxxxx }

FXS Impedance
  US                         : 600Ω
  UK                         : 370Ω + (620Ω || 310nF)
  EU (most)                  : 270Ω + (750Ω || 150nF)
  AU                         : 220Ω + (820Ω || 120nF)
  ...

Hook Flash Timing            : 100-1000 ms (window for hookflash detection)
Onhook Voltage               : 48V (some regions 24V)
Ring Voltage                 : 60V / 75V / 90V (selectable)
Ring Frequency               : 20Hz / 25Hz / 50Hz
```

HT813 FXO port (PSTN line):

```
FXO Port → Channel Settings
PSTN Disconnect Tone       : (auto-detect / manual: freq pair + duration)
PSTN Ring Threshold        : (voltage / duration)
Use PSTN as Failover       : Yes (when SIP unregistered, route to PSTN)

Dial Plan FXO              : { ... | 9xxxxxxx | ... }   (digits to forward to PSTN)
```

Calling from analog phone via HT:

```
Pick up handset → dial tone (region-specific)
Dial digits per dial plan → matches → INVITE sent
```

Inbound from SIP → HT rings the FXS port → analog phone rings.

## GVS Video

Video conferencing. Android-based (GVC32xx). Camera + screen + Bluetooth speakers.

```
Settings → Account → SIP/IPVT/H.323 (multi-protocol)
Settings → Video → Resolution: 720p / 1080p / 4K (model)
Settings → Video → Bitrate: 1Mbps / 2Mbps / 4Mbps / 8Mbps
Settings → Video → Frame Rate: 15/30 fps

Codec
  Audio        : Opus, G.722, G.722.1C, G.711
  Video        : H.264 (Baseline/Main/High), H.265 (HEVC), VP8 (newer)
  Telepresence : H.239 (content sharing channel)
```

Negotiation: SDP `m=video` with profile-level-id for H.264. H.265 advertised as `payload type / encoding name`. Mismatched profiles → negotiated down to H.264 baseline as fallback.

GVS3220 supports up to 9-way video MCU on-device; larger calls via IPVideoTalk cloud.

## Phone-side Customization

Per-phone visual customization via web UI:

```
Settings → Programmable Keys → Soft Keys
Settings → LCD Display
  Idle Screen Layout       : (XML upload)
  Wallpaper                : upload .jpg/.png; per-model resolution requirements
  Screensaver              : timer + clock/photos
  Backlight Timeout        : 5 / 30 / 60 seconds / Always On
  Date/Time Format         : 12h / 24h, M/D/Y vs D/M/Y

Settings → Audio Control
Custom Ringtone            : upload .wav (16kHz mono recommended)
Distinctive Rings          : per-account assignment
Headset / Speaker / Handset volume

Settings → Web Service / XML Application
XML Idle Screen URL        : http://intranet/idle.xml   (custom widget)
LDAP                       : LDAP server for directory lookup
```

Custom XML idle apps display weather, queue stats, in-house dashboards on the screen. Schema documented per model in the "XML Application Programming Guide".

## Logging

Syslog + local log buffer.

```
Maintenance → Syslog
Syslog Server          : 192.0.2.50
Syslog Level           : NONE / DEBUG / INFO / WARNING / ERROR / FATAL
Syslog Tag             : phone-1010
Syslog Protocol        : UDP / TCP / TLS / Hybrid
Syslog Keyword Filtering : (regex; only forward matching lines)

Per-Component Log Level
  SIP                  : DEBUG
  Audio                : INFO
  Provisioning         : DEBUG
  Web UI               : INFO
  System               : WARNING
```

Set SIP=DEBUG when troubleshooting registration/INVITE issues; otherwise leave at INFO to reduce log volume on busy fleets.

Local log: `Maintenance → Tools → Download Log` exports recent log lines as `.tar.gz`.

## PCAP

Packet capture from web UI (no SSH access on most models).

```
Maintenance → Tools → Capture Trace
Trace Type            : SIP / Media / All
Capture Duration      : 60 seconds (default; configurable)
Capture Buffer        : 4 MB (per-model)
Filter (optional)     : tcpdump-style (e.g. host 192.0.2.10 and port 5060)

Action: Start → reproduce issue → Stop → Download trace.pcap
```

Open in Wireshark — `sip` filter for signaling, `rtp.streams` for media.

## Common Errors

Codes seen on the LCD or in syslog:

```
Account 1 Registration Failed: 401     — auth challenge but creds wrong
                                          → fix: verify Authenticate ID + password

Account 1 Registration Failed: 403     — server rejects (account disabled, IP blocked)
                                          → fix: PBX account enabled? IP whitelist?

Account 1 Registration Failed: 408     — request timeout; no reply from server
                                          → fix: connectivity? wrong port? wrong transport?

Account 1 Registration Failed: 503     — service unavailable
                                          → fix: server overloaded; retry; check secondary

Account 1 Registration Failed: 480     — temporarily unavailable
                                          → fix: account state on PBX (e.g. user not provisioned)

Network: DHCP Failed                   — no DHCP reply
                                          → fix: cable, switch port, DHCP server, VLAN

TFTP Failed                            — provisioning fetch via TFTP
                                          → fix: server reachable? path correct? UDP/69 open?

HTTP Failed                            — provisioning fetch via HTTP
                                          → fix: URL correct? auth? cert chain (HTTPS)?

Failed to download config              — exhausted all attempts
                                          → fix: combine TFTP/HTTP logs; check provision URL;
                                                 verify file naming (cfg<MAC>.xml)

Codec Negotiation Failed               — 488 Not Acceptable Here returned/received
                                          → fix: codec list overlap with peer

STUN Server Unreachable                — STUN UDP/3478 blocked or wrong server
                                          → fix: dig SRV; firewall; STUN test

NTP Sync Failed                        — UDP/123 blocked or wrong NTP server
                                          → fix: pool.ntp.org reachable; UDP/123 outbound

Cert Verification Failed               — TLS cert chain or hostname mismatch
                                          → fix: upload correct CA; SNI; system clock current
```

## Common Gotchas

### 1. Default password 123456 not changed

```
[broken]
Phone deployed → web UI exposed on LAN → admin/123456 still default
→ rogue device on network logs in → exfiltrates SIP creds in plaintext

[fixed]
First boot: change admin password
GDMS: enforce password policy in template
GRP series: disable HTTP (HTTPS only); restrict web access by IP allowlist
```

### 2. Web UI on port 80 instead of 443

```
[broken]
Web Access: HTTP only → admin login over plaintext → creds visible to anyone
on the LAN segment running tcpdump

[fixed]
Maintenance → Web Access → HTTPS only
Upload internal CA cert; phone serves UI cert signed by it
Restrict source IPs (Maintenance → Web Access Control List)
```

### 3. "Use Random Port" enabled but firewall expects specific port

```
[broken]
Account → Use Random Port: Yes → SIP source port randomized each registration
Firewall has rule: permit 192.0.2.10 from src 5060 only → drops
→ registration intermittent / drops after firewall NAT timeout

[fixed]
Use Random Port: No → predictable Local SIP Port = 5060 (Account 1) / 5062 (Account 2) / ...
Firewall rule matches; symmetric NAT also less brittle
```

### 4. Multiple BLF subscriptions but server doesn't support eventlist

```
[broken]
50 MPK keys configured as BLF → 50 SUBSCRIBE dialogs every refresh interval
→ PBX CPU spike / SUBSCRIBE backlog / BLFs flap

[fixed]
Switch all 50 keys to MPK type "Eventlist BLF"
Configure Eventlist BLF URI in Settings → Call Features
PBX: configure resource list with all 50 monitored extensions
→ 1 SUBSCRIBE per phone, server NOTIFY aggregates
```

### 5. DECT handset not paired before configuring SIP account

```
[broken]
Configure DP750 Account 1 → assign HS1 to Account 1
HS1 not yet paired → calls fail "Handset Unavailable"

[fixed]
Pair HS1 first: base subscribe mode → handset registration menu → enter PIN
Verify HS1 status = "Registered"
THEN assign Account 1 → HS1 in HS Line Settings
```

### 6. HT analog impedance setting wrong for region

```
[broken]
HT812 deployed in EU but FXS Impedance = 600Ω (US default)
Analog phone connected → echo, distortion, low volume, intermittent dial tone
Some phones won't even ring (impedance mismatch reflects ring voltage)

[fixed]
FXS Settings → Impedance → set to "270Ω + 750Ω||150nF" (most EU)
                                  "370Ω + 620Ω||310nF" (UK)
                                  "220Ω + 820Ω||120nF" (AU)
Reboot; test dial tone, ring, voice quality
```

### 7. HT FXS Caller ID — ETSI vs Bellcore

```
[broken]
HT in UK; phone displays no caller ID despite SIP From containing it
→ HT defaulting to Bellcore FSK (US standard), UK phone expects ETSI FSK before-ring

[fixed]
FXS → Caller ID Scheme → ETSI FSK During Ringing (or ETSI DTMF, depending on phone)
Reboot; verify CID displays on inbound call
```

### 8. Codec priority list excludes server's preferred → 488

```
[broken]
Phone codec list: PCMU, PCMA only
Server SDP offer: G.722, Opus, G.729A
→ no overlap → phone replies 488 Not Acceptable Here → call fails

[fixed]
Add G.722, G.729A (and Opus on GRP) to codec preference list
Or: PBX-side dialplan to transcode (taxes PBX CPU)
Verify with PCAP — look for matching m=audio entries in 200 OK SDP
```

### 9. Force RTP encryption but server can't do SRTP

```
[broken]
Account → SRTP Mode = Required
Server (legacy Asterisk, plain RTP only) → no a=crypto in SDP offer
→ phone rejects with 488 / 415 → call fails silently

[fixed]
SRTP Mode = Optional / Negotiated → falls back to RTP
OR upgrade server to support SRTP (chan_pjsip + dtls/srtp)
Verify with SIP trace — look for a=crypto: lines and 200 OK acceptance
```

### 10. GXP older firmware uses txt config, GRP newer uses XML — mixing breaks

```
[broken]
Provision server has cfg<MAC> (text) for fleet
GRP2614 fetches → expects cfg<MAC>.xml → 404 → falls back? Maybe.
Different firmware versions parse text vs XML differently → silent partial config

[fixed]
Maintain BOTH cfg<MAC>.xml AND cfg<MAC> (text) for mixed-firmware fleet
OR migrate fully to XML (newer firmware all support XML)
Use GDMS — abstracts the format per-model
Verify with "Trace Configuration File Download" tool to see which URL fetched
```

### 11. GDMS account assignment but device offline → never provisions

```
[broken]
Add MAC to GDMS, define template
Phone never appears online; "last heartbeat: never"

[fixed]
Verify: phone has Internet egress on TCP/443
Verify: phone firmware version supports GDMS (older GXP may not)
Verify: phone hasn't been factory-defaulted into GDMS-disabled state
Toggle: Maintenance → "Allow GDMS / Cloud" = Yes
Check: redirector reachable — phone log shows "Contacting fm.grandstream.com"
```

### 12. VLAN tagging set but switch port untagged → boot fails

```
[broken]
Network → 802.1Q VLAN = 100 set
Switch port configured access (untagged) on VLAN 1
→ phone tags VLAN 100 → switch drops → no DHCP → no boot

[fixed]
Either:
  (a) Switch port → trunk with native VLAN 1 (data) and tagged VLAN 100 (voice)
      Phone PC port = untagged VLAN 1
  (b) Phone VLAN = 0 (untagged), switch port access VLAN 100
  (c) LLDP-MED on switch advertises VLAN 100; phone auto-tags
→ Easiest: factory reset phone (long-press OK on boot), let LLDP-MED handle it
```

### 13. Register Expiration in seconds vs minutes mismatch

```
[broken]
GXP firmware 1.0.x — Register Expiration = 60 (interpreted as 60 seconds)
→ 60-second re-REGISTER → server overloaded / FreePBX rate-limits → 503

[fixed]
Check tooltip — confirm unit (minutes vs seconds)
Set 3600 (seconds) or 60 (minutes) → 1 hour expiry
For NAT keep-alive, use SIP OPTIONS keep-alive at 30-45s instead of short re-REGISTER
```

### 14. Phone clock skewed → TLS cert verification fails

```
[broken]
NTP unreachable (UDP/123 blocked) → phone clock at default 2009-01-01
TLS cert "not yet valid" → registration fails over TLS

[fixed]
Ensure NTP reachable: pool.ntp.org or local NTP server
Network → Time → NTP Server: time.cloudflare.com
Set timezone explicitly
For air-gapped deployments: use pre-shared cert with long validity, or self-signed CA
```

## Diagnostic Tools

```
Web UI: Status → Account Status              live registration state
Web UI: Status → Network                      live IP, gateway, DNS
Web UI: Maintenance → Tools → Ping            ICMP test
Web UI: Maintenance → Tools → Traceroute      hop trace
Web UI: Maintenance → Tools → NSLookup        DNS resolve
Web UI: Maintenance → Tools → Capture Trace   PCAP download
Web UI: Maintenance → Syslog                  remote log forward
Web UI: Maintenance → Tools → Download Log    local log dump

Phone keypad: MENU → Status → Network         IP, MAC, FW version
Phone keypad: MENU → Status → Account         per-account state

Boot recovery: hold "*" + "9" during boot     factory reset (some models)
              or:  long-press OK on boot     enter recovery mode

LED color codes (per-model):
  Green slow blink           — boot
  Solid green                — registered, idle
  Red slow blink             — incoming call
  Red solid                  — error / unregistered
  Amber                      — DND / forward active

Side-by-side debug:
  syslog server (rsyslog, splunk, papertrail)
  phone PCAP from web UI   (parallel to syslog)
  PBX SIP trace             (asterisk: pjsip set logger on)
  → triangulate signaling vs media issues
```

## Sample Cookbook — Asterisk + GRP2614

Minimal provisioning of a GRP2614 to register against Asterisk.

PBX side (`pjsip.conf`):

```ini
[transport-udp]
type=transport
protocol=udp
bind=0.0.0.0:5060

[1001]
type=endpoint
context=internal
disallow=all
allow=ulaw
allow=alaw
allow=g722
auth=1001
aors=1001

[1001]
type=auth
auth_type=userpass
username=1001
password=secret-pw

[1001]
type=aor
max_contacts=2
qualify_frequency=60
```

Provisioning file `cfg002b67aabbcc.xml` (replace MAC):

```xml
<?xml version="1.0" encoding="UTF-8"?>
<gs_provision version="1">
  <config version="2">
    <P271>1</P271>
    <P47>asterisk.example.com</P47>
    <P35>1001</P35>
    <P36>1001</P36>
    <P34>secret-pw</P34>
    <P3>1001 Reception</P3>

    <P57>9</P57>     <!-- G.722 -->
    <P58>0</P58>     <!-- PCMU -->
    <P59>8</P59>     <!-- PCMA -->

    <P52>1</P52>     <!-- NAT: STUN -->
    <P77>stun.l.google.com:19302</P77>

    <P192>fw.grandstream.com</P192>
    <P237>https://prov.example.com/cs/</P237>
    <P212>2</P212>   <!-- HTTPS -->

    <!-- MPK 1: BLF on extension 1100 -->
    <P301>0</P301>
    <P302>1</P302>
    <P303>1100</P303>
    <P304>Reception</P304>
  </config>
</gs_provision>
```

Serve via nginx with HTTPS:

```nginx
server {
  listen 443 ssl http2;
  server_name prov.example.com;
  ssl_certificate /etc/ssl/prov.crt;
  ssl_certificate_key /etc/ssl/prov.key;
  root /var/www/cs;
  location / {
    autoindex off;
    try_files $uri =404;
  }
}
```

Phone factory default → set Config Server Path = `https://prov.example.com/cs/` → reboot → fetches `cfgaabbccddeeff.xml` (lowercase MAC) → registers.

Verify on PBX:

```
asterisk -rvvv
pjsip show endpoint 1001
pjsip show contacts
pjsip set logger on
```

Make a test call. Watch INVITE/200 OK exchange; verify codec negotiation in SDP.

## Hardware Specifics

```
GRP series      better build, more keys, color screen, Gigabit, USB, BT optional, GDMS-native
GXP older       cheaper, monochrome on entry models, 100Mbps on 1610/1620, basic codec list
DP series       cordless DECT — for warehouse, hotel, hospital, retail floor
WP series       integrated DECT — wireless deskphone use case
HT series       analog reuse — fax machines, alarms, doorphones, cordless base stations
GVC/GVS         videoconferencing — meeting rooms, executive offices
GXW             bulk gateways — connect 16/24/48 analog phones to a SIP PBX
UCM             vertically-integrated PBX — auto-provisions Grandstream phones, GDMS-aware
                Asterisk underneath but with Grandstream UI/UX wrap
```

Choose by user count + use case:

```
SOHO < 10 users          UCM6202 + GRP2602 (entry desk) + maybe HT802 (fax)
SMB 10-50 users          UCM6304 + GRP2614 + DP752 (warehouse handsets)
Larger 50-200 users      UCM6308 + GRP2615/2616 (execs) + GRP2614 (general)
                          + DP752 + repeaters + WP820 (mobile floor)
Enterprise 200+          UCM6510A clustered + mixed GRP fleet + GDMS for management
                          + GVS for meeting rooms
```

## Idioms

```
"Use HTTPS for provisioning."
  Plain HTTP/TFTP leaks credentials. Set up Let's Encrypt or internal CA on
  prov server; set Config Upgrade Via = HTTPS; Validate Certificate = Yes.

"GDMS for zero-touch."
  Don't waste hours on per-phone manual config. Pre-register MACs in GDMS;
  ship phones to remote site; user plugs in; phone auto-provisions. Done.

"DP series for warehouse / medical / hotel."
  Cordless DECT, long battery, robust handsets, multiple-handsets-per-base.
  Pair with UCM for full auto-provisioning of handsets.

"HT for analog door phones / fax / alarms."
  Don't try to make a SIP-aware door entry station. Use existing analog
  intercom with HT802 → SIP → PBX.

"UCM if you want vertically-integrated."
  Asterisk + Grandstream UI + zero-config phone provisioning. One-vendor support.
  Trade-off: less flexibility than raw FreePBX/Asterisk.

"Syslog all phones to central server for fleet diagnostics."
  Set Syslog Server = central rsyslog; tag = $MAC or extension.
  Grep across fleet for "Registration Failed" patterns; build dashboards.

"Lock to GDMS for compliance fleets."
  GDMS template with "Lock Web UI" prevents end-user tampering.
  Audit trail of config changes; staged firmware rollout.

"Eventlist BLF for any deployment > 20 BLF buttons."
  Per-extension SUBSCRIBE doesn't scale. UCM/BroadWorks/FreePBX all support
  eventlist; configure once at PBX, point all phones to the URI.

"Test codec list with PCAP before deploying."
  488 errors are silent; users blame "the phones." PCAP one inbound + one
  outbound; verify SDP overlap; fix at fleet level via XML config.

"Disable HTTP web access in production."
  Maintenance → Web Access → HTTPS only. Or Disabled if GDMS-managed.
  Default 123456 + plain HTTP = inevitable compromise.

"Reboot after major config changes via XML."
  Some P-tags require reboot to apply (network-level, transport, codec).
  Push config + reboot via GDMS in maintenance window.

"NAT: STUN + symmetric RTP for road-warrior phones."
  Set NAT Traversal = STUN, Symmetric RTP = Yes, Keep-Alive Interval = 20s.
  Solves 90% of remote-phone audio issues.
```

## Grandstream P-ID Reference

Grandstream's text-format config files use numbered parameters (P-IDs). Knowing the major ones lets you read/diff configs without the web UI.

| P-ID | Setting | Notes |
|---|---|---|
| P2 | Admin password | default 123456 |
| P3 | User password | default 123 |
| P4 | Web access port | default 80 (or 443 for HTTPS) |
| P8 | Time zone | TZ database name (America/Los_Angeles) |
| P30 | Status web access mode | 0=disabled, 1=user, 2=admin |
| P63 | DTMF mode | 1=in-band, 2=RFC2833, 3=SIP-INFO |
| P64 | Session expiration | seconds (default 180) |
| P65 | Min session expiration | seconds (default 90) |
| P67 | Random Port (boolean) | randomize local SIP port |
| P75 | Use Random Port | similar to P67 (model-dependent) |
| P83 | Local SIP port | default 5060 |
| P135 | Layer 3 QoS DSCP | DSCP value for SIP signaling |
| P137 | Layer 3 QoS DSCP | DSCP value for RTP media |
| P191 | Account 1 active | 1=enable, 0=disable |
| P192 | Account 1 SIP server | hostname or IP |
| P193 | Account 1 outbound proxy | optional |
| P196 | Account 1 SIP user ID | the extension number / username |
| P197 | Account 1 auth ID | usually same as user ID |
| P34 | Account 1 password | the SIP password |
| P3 | Account 1 display name | what shows on caller ID |
| P40 | Account 1 voicemail | feature code (e.g., *97) |
| P47 | Account 1 SUBSCRIBE for MWI | 1=enable |
| P50 | Account 1 voice mail userID | usually same as account |
| P57 | Account 1 NAT traversal | 0=No NAT, 1=STUN, 2=Keep-alive, 3=UPnP, 4=Auto, 5=VPN |
| P58 | Account 1 use NAT IP | manual public IP override |
| P59 | Account 1 dial plan | regex-flavored pattern |
| P78 | Account 1 SUBSCRIBE expiration | seconds |
| P85 | Account 1 codec preference | comma-separated codec IDs |
| P102 | Account 1 SRTP mode | 0=disabled, 1=enabled, 2=enabled-if-supported |
| P130 | Layer 3 QoS DSCP for video | DSCP value |
| P156 | Account 1 BLF list URI | URL of BLF event list |
| P181 | Time zone display in idle | DST handling |
| P212 | LDAP server address | hostname |
| P213 | LDAP server port | default 389 |
| P214 | LDAP server bind user | DN |
| P215 | LDAP server bind password | password |
| P217 | LDAP search base | DN |
| P244 | NTP server | hostname |
| P246 | DHCP option 132 (VLAN) | 1=enable |
| P246 | LLDP-MED enable | 1=enable |
| P273 | Web HTTPS only | 1=enforce |
| P276 | LDAP version | 2 or 3 |
| P301 | Phonebook download mode | 0=disabled, 1=manual, 2=auto |
| P304 | Phonebook XML server | URL |
| P305 | Phonebook download interval | seconds |
| P327 | SIP transport | 0=UDP, 1=TCP, 2=TLS |
| P341 | TLS verification mode | 0=none, 1=verify-once, 2=full |
| P391 | SIP T1 timer | ms (default 500) |
| P392 | SIP T2 timer | ms (default 4000) |
| P398 | TLS minimum version | 0=any, 1=TLS1.0, 2=TLS1.1, 3=TLS1.2, 4=TLS1.3 |
| P847 | Configuration server URL | http(s):// for provisioning |
| P848 | Config server username | for HTTP auth |
| P849 | Config server password | for HTTP auth |
| P850 | Config server method | 0=TFTP, 1=HTTP, 2=HTTPS, 3=FTP, 4=FTPS |
| P1359 | Use AES encryption for config file | 1=enable |
| P1360 | AES key for config encryption | hex string |

The full list is in the Grandstream "Administrators Guide" XML reference per model. Modern firmware supports both numbered (P-id) and named XML config formats.

## Sample Config Snippets — txt vs XML format

### Text format (legacy, all models):

```text
P2 = NewSecurePassword123!
P192 = sip.example.com
P196 = 1001
P34 = SIPpassword!
P3 = Alice Anderson
P57 = 1
P327 = 2
P847 = https://provision.example.com/{$mac}.xml
P850 = 2
```

### XML format (newer firmware, GRP series):

```xml
<?xml version="1.0" encoding="UTF-8"?>
<gs_provision version="1">
  <config>
    <P2>NewSecurePassword123!</P2>
    <P192>sip.example.com</P192>
    <P196>1001</P196>
    <P34>SIPpassword!</P34>
    <P3>Alice Anderson</P3>
    <P57>1</P57>
    <P327>2</P327>
    <P847>https://provision.example.com/cfg{$mac}.xml</P847>
    <P850>2</P850>
  </config>
</gs_provision>
```

The advantage of XML format: per-line readability, easier diffing, structured parsers can validate.

## HT Analog Adapter Regional Impedance Settings

The HT-series analog adapters connect to PSTN-style FXO ports. Regional impedance settings ensure the analog signaling matches the local telephony spec:

| Region | Impedance | Caller ID Standard | Ring Cadence |
|---|---|---|---|
| United States | 600Ω | Bellcore (FSK) | 2s on / 4s off |
| United Kingdom | 270Ω + 750Ω + 150nF | DTMF or FSK | 0.4s on / 0.2s off / 0.4s on / 2s off |
| Germany | 220Ω + 820Ω + 115nF | DTMF | 1s on / 4s off |
| France | 215Ω + 750Ω + 200nF | FSK | 1.5s on / 3.5s off |
| Italy | 600Ω | FSK | 1s on / 4s off |
| Netherlands | 600Ω | FSK | 1s on / 3s off |
| Spain | 220Ω + 820Ω + 120nF | FSK | 1.5s on / 3s off |
| Sweden | 200Ω + 600Ω + 1µF | DTMF | 1s on / 5s off |
| Australia | 200Ω + 780Ω + 150nF | FSK | 0.4s on / 0.2s off / 0.4s on / 2s off |
| Japan | 600Ω | DTMF | 1s on / 2s off |
| Brazil | 900Ω | DTMF | 1s on / 4s off |
| China | 200Ω + 680Ω + 100nF | FSK | 1s on / 4s off |
| India | 600Ω | DTMF | 0.4s on / 0.2s off / 0.4s on / 2s off |
| Russia | 220Ω + 820Ω + 115nF | DTMF | 1s on / 4s off |

The HT813 has both FXS (toward analog phone) and FXO (toward PSTN line) ports — each port can be set to a different regional impedance if you're doing hybrid setups.

## See Also

- ip-phone-provisioning
- sip-protocol
- asterisk
- freeswitch
- yealink-phones
- polycom-phones
- cisco-phones
- snom-phones

## References

- grandstream.com/support — main support portal
- documentation.grandstream.com — model-specific admin guides + datasheets
- GRP26xx Series Administration Guide (PDF)
- GXP21xx Series Administration Guide (PDF)
- DP750/DP752 User Guide and Administration Guide (PDF)
- HT8xx Administration Guide (PDF)
- UCM63xx/65xx Administration Guide (PDF)
- WP8xx Administration Guide (PDF)
- GVC/GVS3xxx Administration Guide (PDF)
- Grandstream Configuration Tool (binary cfg.bin generator) — gs_config_tool
- GDMS Cloud — www.gdms.cloud (user/reseller portal)
- Grandstream XML Application Programming Guide (per-model)
- Grandstream Provisioning Guide (cross-model)
- RFC 3261 — SIP: Session Initiation Protocol
- RFC 3262 — Reliability of Provisional Responses (PRACK)
- RFC 3263 — SIP locating SIP servers (DNS SRV/NAPTR)
- RFC 3361 — DHCP Option for SIP Servers (Option 120)
- RFC 3711 — Secure RTP (SRTP)
- RFC 3856 — A Presence Event Package for SIP
- RFC 4566 — SDP: Session Description Protocol
- RFC 4662 — Resource List Subscriptions / Eventlist
- RFC 5285 — RTP Header Extensions
- RFC 5763 — DTLS-SRTP
- RFC 7064 — STUN URI scheme
- RFC 7215 — SIP RFC 3261 Errata
- RFC 8866 — SDP latest
- ETSI EN 300 175 — DECT standards
- ITU-T G.711 / G.722 / G.722.1 / G.726 / G.729 — codecs
- IEEE 802.1Q — VLAN tagging
- IEEE 802.3af / 802.3at — PoE
- ANSI/TIA-1057 — LLDP-MED
