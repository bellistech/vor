# IP Phone Provisioning

Auto-configuration of SIP phones from a server — vendor-agnostic + per-vendor specifics.

## Setup

Provisioning is the auto-configuration of SIP/VoIP phones from a central server, eliminating manual per-handset configuration. A factory-fresh phone arrives knowing nothing about your PBX, your extensions, your SIP credentials, your codecs, your NTP server, your dial plan, your line keys, your ringtones, your firmware version. Provisioning solves this without anyone touching the device.

The bootstrap problem: phone arrives in a box, has no idea where the provisioning server is, what its extension number is, what credentials to use. Solutions, in order of operator effort:

- Plug into network → DHCP hands out provisioning URL → phone fetches per-MAC config → done. This is "zero-touch deploy."
- Plug into network → phone hits vendor redirector (Polycom RPRM, Yealink RPS) → redirector returns customer URL → phone fetches → done. Also zero-touch but requires pre-registering MAC with vendor.
- Manual: phone's web UI → enter provisioning URL → save → reboot → fetches → done. Not scalable past ~5 phones.
- Phone front-panel keypad: deeply painful, only for break-glass scenarios.

The goal of any sane deployment: zero-touch. Operator unboxes phone, plugs RJ45, walks away. By the time it stops booting, it's registered and ringing.

Provisioning covers: SIP credentials, registrar/proxy addresses, codec preference, line keys/programmable buttons, time zone, NTP, firmware, ringtones, contacts, BLF (busy lamp field) targets, web GUI password, admin password, network settings (VLAN, QoS), call forwarding, voicemail UI, dial plan, language, locale, screensaver, and dozens of vendor-specific knobs.

A "provisioning server" is just a webserver (or TFTP server) hosting per-MAC config files in a known directory layout. There's no magic — `nginx` + a directory of `.cfg` files + DHCP option pointing phones at it = working deployment.

```bash
# Conceptual flow:
# 1. Phone boots, no config
# 2. DHCP DISCOVER → DHCP OFFER includes Option 66 / 150 / 160
# 3. Phone parses option, builds URL: https://prov.example.com/{MAC}.cfg
# 4. HTTP GET /000000000000.cfg (or per-MAC)
# 5. Parse config, apply settings, reboot if firmware changed
# 6. Re-fetch config (now with applied admin pwd, etc)
# 7. SIP REGISTER to registrar
# 8. Phone is live
```

## Provisioning Protocols

### TFTP — port 69/UDP

Trivial File Transfer Protocol. No authentication. No encryption. Plaintext on the wire. Used historically because it's tiny — fits in phone bootloaders. Still supported by every vendor for backwards compatibility, but considered legacy. **Do not use TFTP for new deployments outside an isolated voice VLAN.** Configs may contain SIP passwords; broadcasting them in plaintext is unacceptable.

```bash
# TFTP server URL form (in DHCP option 66 or 150):
tftp://prov.example.com/{MAC}.cfg
# or just bare hostname:
prov.example.com
# Phone appends model-specific filename
```

### HTTP — port 80/TCP

Most modern phones default to HTTP for provisioning. Supports HTTP Basic auth (Authorization header), digest auth (some vendors), URL-embedded credentials. Common for LAN-only deployments. Configs in transit are still plaintext unless you wrap in TLS.

```bash
# HTTP URL forms:
http://prov.example.com/configs/{MAC}.cfg
http://user:pass@prov.example.com/configs/{MAC}.cfg
http://prov.example.com/?mac=$MA&model=$PN  # vendor variable substitution
```

### HTTPS — port 443/TCP

The "use TLS" rule: configs contain SIP passwords, so the channel must be encrypted. HTTPS with cert validation is the modern default. Phones must trust the CA — either public CA (Let's Encrypt) or pre-loaded private CA. Some phones disable cert validation by default; **enable it.** Mutual TLS (client cert) is the gold standard for high-security deployments.

```bash
# HTTPS:
https://prov.example.com/configs/{MAC}.cfg
# With cert pinning (vendor-specific config option)
# With mutual TLS — phone presents its MIC (manufacturer-installed cert)
```

### FTP — port 21/TCP (control), 20 or ephemeral (data)

Rare. A few legacy vendors support it. Plaintext credentials. Active vs passive mode complications. Avoid.

### FTPS — FTP over TLS

Even rarer. Some Audiocodes / Polycom legacy gear. Avoid; use HTTPS.

### MAC-address-based filename pattern

The universal convention: phone fetches a file named after its MAC address.

```bash
# MAC: 00:04:f2:3a:8b:6c
# Filename forms (vendor-dependent):
0004f23a8b6c.cfg            # lowercase, no separators (Polycom default)
0004F23A8B6C.cfg            # UPPERCASE (some Yealink)
00-04-f2-3a-8b-6c.cfg       # hyphenated
00:04:f2:3a:8b:6c.cfg       # colon (rare)
phone_0004f23a8b6c.xml      # vendor prefix
SEP0004F23A8B6C.cnf.xml     # Cisco SCCP / CUCM format

# Some phones lowercase, some uppercase. Test both. nginx case-insensitive
# matching can save you here:
location ~* "/[0-9a-f]{12}\.cfg$" { ... }
```

### Per-MAC + per-model config split

Best practice: split config into common (all phones), per-model (e.g., all VVX 411s), per-MAC (this specific phone). Phone reads in order, later overrides earlier.

```
common.cfg          → site-wide settings (registrar, NTP, codec)
vvx411.cfg          → model-specific (line key count, screen layout)
0004f23a8b6c.cfg    → per-phone (extension, SIP creds, BLF targets)
```

This lets you change one variable site-wide without regenerating 500 per-MAC files.

## DHCP Provisioning Pointers

DHCP options the phone reads to discover the provisioning server.

### Option 66 — TFTP server name (RFC 2132)

String. Usually a hostname or URL.

```bash
# ISC dhcpd:
option tftp-server-name "prov.example.com";
# Or full URL (most phones accept this):
option tftp-server-name "https://prov.example.com/configs/";

# dnsmasq:
dhcp-option=66,prov.example.com
dhcp-option=66,"https://prov.example.com/configs/"

# Cisco IOS:
ip dhcp pool VOICE
  option 66 ascii "https://prov.example.com/configs/"
```

### Option 67 — boot file name (RFC 2132)

The specific filename. Less common — most phones build the filename themselves from MAC.

```bash
option bootfile-name "phone.cfg";
dhcp-option=67,"{MAC}.cfg"
```

### Option 150 — TFTP server IP (Cisco)

Cisco-specific. List of IP addresses (not hostnames). Cisco phones prefer 150 over 66.

```bash
# ISC:
option space cisco;
option cisco.tftp-server code 150 = array of ip-address;
# or simpler:
option-150 code 150 = { array of ip-address };
option-150 192.0.2.10, 192.0.2.11;

# Cisco IOS:
ip dhcp pool VOICE
  option 150 ip 192.0.2.10 192.0.2.11
```

### Option 160 — Polycom/Yealink HTTP URL

Polycom historically. Yealink supports it too. URL-form.

```bash
option-160 code 160 = text;
option-160 "https://prov.example.com/configs/";

dhcp-option=160,"https://prov.example.com/configs/"
```

### Option 159 — Polycom HTTP URL alt

Polycom alternate. Some firmware versions look here first.

```bash
option-159 code 159 = text;
option-159 "https://prov.example.com/configs/";
```

### Option 43 — vendor-specific information (RFC 2132)

Encoded sub-option blob, vendor-specific. Cisco, Polycom, Yealink, Snom all use it differently. Often combined with Option 60 (vendor class identifier) so the DHCP server returns vendor-appropriate data.

```bash
# Polycom Option 43 sub-options:
# 1: client identifier (always 0x504C434D = "PLCM")
# 2: provisioning server URL
# 3: ... etc

# ISC syntax (raw hex):
option vendor-encapsulated-options 02:1e:68:74:74:70:73:3a:2f:2f:70:72:6f:76:2e:65:78:61:6d:70:6c:65:2e:63:6f:6d:2f;
# This is sub-option 2, length 30, then ASCII "https://prov.example.com/"
```

### Per-vendor option grab list

| Vendor | Primary Option | Backup |
|---|---|---|
| Polycom | 160 | 43, 66 |
| Yealink | 66 (URL form) | 160, 43 |
| Cisco IP Phone | 150 | 66 |
| Cisco SPA | 66 | 159, 160 |
| Snom | 66 | 43 |
| Grandstream | 66 | 43 |
| Mitel | 43 | 66, 125 |
| Audiocodes | 160 | 66, 43 |
| Avaya | 242 (custom) | 176 (custom), 66 |

Best practice: set Options 66, 150, AND 160 to the same URL. Phones will pick whichever they understand.

### Option 6 — DNS

If your provisioning URL uses a hostname, the phone needs to resolve it. Hand out DNS via Option 6.

```bash
option domain-name-servers 8.8.8.8, 8.8.4.4;
dhcp-option=6,8.8.8.8,8.8.4.4
```

If DNS is broken, Option 66 with a hostname fails silently and confusingly — the phone just sits there. Use IP addresses in Option 66 to bypass DNS during early bring-up, then switch to hostname.

### Option 42 — NTP servers (critical for TLS)

If TLS is in play, the phone needs an accurate clock to validate certificates. A factory-fresh phone has clock = epoch 0 = January 1970. Cert validation fails because `notBefore > current_time` is impossible. Always provide NTP.

```bash
option ntp-servers 192.0.2.5, time.cloudflare.com;
dhcp-option=42,192.0.2.5
```

Belt-and-braces: also bake NTP into the config file itself, so even if DHCP misses it, phone catches up on first config fetch.

## Manufacturer Redirector Services

Vendor-hosted services that auto-redirect new MACs to your provisioning URL. The "buy phone, pre-register MAC, ship to customer site, plug in, zero-touch deploy" workflow.

### Polycom RPRM / ZTP

Zero Touch Provisioning. Phone hits `https://obp.plcm.service-now.com` (or modern RPRM endpoint), checks if its MAC is registered, gets redirected to your URL.

```
Workflow:
1. Buy phone, get MAC sticker
2. Log into Polycom ZTP portal
3. Create profile pointing at https://prov.example.com/
4. Add MAC to profile
5. Ship phone to customer
6. Customer plugs in, phone hits ZTP, downloads YOUR config

Profile JSON (conceptual):
{
  "url": "https://prov.example.com/",
  "username": "phoneuser",
  "password": "rotating-secret"
}
```

### Yealink RPS — Redirection and Provisioning Service

`https://api-dm.yealink.com/`. Same model. Yealink RPS has a public CSV/Excel uploader for bulk MAC entry.

```
1. Yealink RPS portal login
2. Upload CSV of MACs + URL
3. Phone first-boot → asks RPS → gets your URL
```

### Grandstream GDMS — Device Management System

Modern Grandstream. `https://gdms.cloud.grandstream.com/`. Combines RPS-style redirection AND remote device management (config push, firmware update, status monitoring).

### Snom RPS

Snom Redirection. Less commonly used than Yealink/Polycom. Configure via Snom partner portal.

### Workflow & Hygiene

```
Buy phone with MAC → register MAC + URL with vendor RPS →
Phone arrives at site → plug in →
DHCP gives generic config (or no specific provisioning option) →
Phone has hardcoded vendor RPS URL in firmware →
Hits RPS → RPS replies "go to https://customer.example.com/" →
Phone fetches customer URL → loaded → registered.
```

Revocation: if phone is stolen / RMA'd, mark MAC as revoked in RPS portal. Phone now redirects to vendor default or fails to provision.

Audit: RPS logs each lookup. Useful to detect rogue phones or misconfigured deployments. Pull logs monthly.

## Configuration File Formats

### XML — Polycom, Yealink (older), Snom, Mitel, Cisco modern

```xml
<?xml version="1.0" encoding="UTF-8"?>
<polycomConfig xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <reg
    reg.1.address="alice@example.com"
    reg.1.auth.userId="alice"
    reg.1.auth.password="hunter2"
    reg.1.outboundProxy.address="sbc.example.com"
    reg.1.outboundProxy.port="5061"
    reg.1.server.1.address="registrar.example.com"
    reg.1.server.1.transport="TLSOnly"
  />
</polycomConfig>
```

### .cfg key=value — Cisco SPA, some Yealink, Grandstream

```ini
# Cisco SPA-style:
Display_Name_1_=Alice Smith
User_ID_1_=alice
Password_1_=hunter2
Proxy_1_=sbc.example.com:5061
Use_Outbound_Proxy_1_=Yes
Outbound_Proxy_1_=sbc.example.com:5061
Register_1_=Yes
Make_Call_Without_Reg_1_=No
```

### TXT — legacy Avaya 46xxsettings.txt, some legacy Cisco

```
## 46xxsettings.txt
SET MCIPADD 192.0.2.10
SET HTTPSRVR 192.0.2.10
SET TLSSRVR 192.0.2.11
SET ENABLE_AVAYA_ENVIRONMENT 0
SET SIP_CONTROLLER_LIST registrar.example.com:5061;transport=tls
```

### JSON — modern Yealink (T5/W7x), some Audiocodes

```json
{
  "account.1.enable": "1",
  "account.1.label": "Alice",
  "account.1.user_name": "alice",
  "account.1.auth_name": "alice",
  "account.1.password": "hunter2",
  "account.1.sip_server.1.address": "registrar.example.com",
  "account.1.sip_server.1.port": "5061",
  "account.1.sip_server.1.transport_type": "2"
}
```

### Per-vendor file naming cheatsheet

| Vendor | Master | Per-Model | Per-MAC | Notes |
|---|---|---|---|---|
| Polycom | 0000000000000.cfg | sip.cfg, phone1.cfg | 0004f23a8b6c.cfg | Master is "factory MAC" of zeros |
| Yealink | y0000000000XX.cfg | (model code) | 0015651234ab.cfg | `XX` = model code, e.g. `00` for T20P |
| Cisco SPA | spa.cfg | spa525g.cfg | spa$MA.cfg | `$MA` = MAC variable |
| Cisco IP Phone | XMLDefault.cnf.xml | (none) | SEP0004F23A8B6C.cnf.xml | "SEP" prefix mandatory |
| Grandstream | cfg.xml | cfgmodel.xml | cfg0004f23a8b6c.xml | Or `.bin` for encrypted |
| Snom | settings.xml | snom320.xml | 0004132ab8c.xml | |
| Mitel | aastra.cfg | model.cfg | mac.cfg | |
| Audiocodes | (variable) | (variable) | mp_<MAC>.ini | INI format |
| Avaya | 46xxsettings.txt | (none) | (none) | Single shared file w/ SET conditionals |

## Polycom (PolyEdge / VVX / SoundPoint / SoundStation)

### File structure

The master config is `0000000000000.cfg` (twelve zeros). Phone always fetches this first. It's a *pointer* file — its `CONFIG_FILES=` attribute lists which other files to load.

```xml
<?xml version="1.0" encoding="UTF-8"?>
<APPLICATION
  APP_FILE_PATH="firmware/sip.ld"
  CONFIG_FILES="features.cfg, sip.cfg, phoneapp.cfg, [PHONE_MAC_ADDRESS].cfg"
  MISC_FILES=""
  LOG_FILE_DIRECTORY=""
  OVERRIDES_DIRECTORY=""
  CONTACTS_DIRECTORY=""
  LICENSE_DIRECTORY=""
  USER_PROFILES_DIRECTORY=""
  CALL_LISTS_DIRECTORY=""
  COREFILE_DIRECTORY=""
/>
```

`[PHONE_MAC_ADDRESS]` is a literal Polycom variable; the phone substitutes its own MAC.

### Common files

- `0000000000000.cfg` — master pointer
- `sip.cfg` — SIP-related global settings (transport, codecs, registrar templates)
- `phoneapp.cfg` — phone application settings (line keys, dial plan)
- `features.cfg` — feature toggles (call park, presence, etc.)
- `contacts.xml` — global directory
- `<MAC>.cfg` — per-phone overrides (the actual SIP credential, line label)

### Common attributes

```xml
<reg
  reg.1.address="1001"
  reg.1.auth.userId="1001"
  reg.1.auth.password="hunter2"
  reg.1.label="Alice — Sales"
  reg.1.displayName="Alice Smith"
  reg.1.type="private"
  reg.1.outboundProxy.address="sbc.example.com"
  reg.1.outboundProxy.port="5061"
  reg.1.outboundProxy.transport="TLS"
  reg.1.server.1.address="registrar.example.com"
  reg.1.server.1.port="5061"
  reg.1.server.1.transport="TLSOnly"
  reg.1.server.1.expires="3600"
  reg.1.server.1.register="1"
/>
```

### Line keys (efk — enhanced feature keys)

```xml
<efk>
  efk.efklist.1.action.string="$FNoCol$Tinvite$Csip:5551234@sbc.example.com$Cwait$Tdigits$Csend$"
  efk.efklist.1.label="Speed: Bob"
  efk.efklist.1.mname="speedBob"
  efk.efklist.1.status="1"
</efk>
```

### Softkeys

```xml
<softkey
  softkey.1.label="Park"
  softkey.1.action="$FPark$"
  softkey.1.use.idle="1"
  softkey.1.use.connected="1"
/>
```

## Yealink (T2 / T3 / T4 / T5 / VP / W series DECT)

### File structure

Two-file model:

- `y0000000000XX.cfg` — "Common CFG" — applies to all phones of model `XX`
- `<MAC>.cfg` — "MAC-Oriented CFG" — per-phone overrides

Model codes (last two digits of `y0000000000XX.cfg`):

| Code | Model |
|---|---|
| 00 | T20P |
| 21 | T21P |
| 22 | T22P |
| 26 | T26P |
| 28 | T28P |
| 32 | T32G |
| 38 | T38G |
| 41 | T41P/T41S |
| 42 | T42G/T42S |
| 46 | T46G/T46S/T46U |
| 48 | T48G/T48S/T48U |
| 53 | T53/T53W |
| 54 | T54W |
| 57 | T57W |
| 73 | T73W |

### Common keys

```ini
# y000000000054.cfg (T54W common):
account.1.enable = 1
account.1.label = "Alice"
account.1.display_name = "Alice Smith"
account.1.user_name = 1001
account.1.auth_name = 1001
account.1.password = hunter2
account.1.sip_server.1.address = registrar.example.com
account.1.sip_server.1.port = 5061
account.1.sip_server.1.transport_type = 2  # 0=UDP, 1=TCP, 2=TLS
account.1.sip_server.1.expires = 3600
account.1.sip_server.1.register_on_enable = 1
account.1.outbound_proxy_enable = 1
account.1.outbound_host = sbc.example.com
account.1.outbound_port = 5061
account.1.subscribe_register = 1

# Codec preference (lower = preferred)
account.1.codec.1.enable = 1
account.1.codec.1.payload_type = PCMU
account.1.codec.1.priority = 1
account.1.codec.2.enable = 1
account.1.codec.2.payload_type = PCMA
account.1.codec.2.priority = 2
account.1.codec.3.enable = 1
account.1.codec.3.payload_type = G722
account.1.codec.3.priority = 3
account.1.codec.4.enable = 1
account.1.codec.4.payload_type = OPUS
account.1.codec.4.priority = 4

# Features
features.dnd.enable = 1
features.call_waiting.enable = 1
features.auto_answer.enable = 0
features.vq_rtpxr.enable = 1
features.headset_prior = 0
features.headset_training = 0
features.dtmf.type = 1  # 0=Inband, 1=RFC2833, 2=SIP INFO
features.dtmf.dtmf_payload = 101

# Network
network.dhcp_host_name = phone-1001
network.static_dns_enable = 0
network.vlan.enable = 1
network.vlan.vid = 100  # Voice VLAN
network.vlan.priority = 5
network.qos.signal_dscp = 40  # CS5
network.qos.voice_dscp = 46   # EF

# Time
local_time.ntp_server1 = pool.ntp.org
local_time.ntp_server2 = time.cloudflare.com
local_time.time_zone = -5
local_time.summer_time = 2  # auto

# Provisioning
auto_provision.url = https://prov.example.com/
auto_provision.user_name = phoneuser
auto_provision.password = rotating-secret
auto_provision.attempt_expired_time = 5
auto_provision.power_on = 1
auto_provision.repeat.enable = 1
auto_provision.repeat.minutes = 1440  # daily

# Programmable / DSS keys
linekey.1.line = 1
linekey.1.value = 1001
linekey.1.label = "Sales"
linekey.1.type = 15  # 15 = Line

linekey.2.line = 1
linekey.2.value = 1002
linekey.2.label = "Bob"
linekey.2.type = 16  # 16 = BLF
linekey.2.extension = 1002

# Web GUI
static.security.user_password = admin:newadminpwd
static.security.user_password = user:newuserpwd
```

### Web GUI

Default URL: `https://<phone-ip>` (port 443) or `http://<phone-ip>` (port 80, often redirects).
Default credentials: `admin / admin` — **change immediately** in provisioning.

### Auto-provisioning command (force a re-fetch)

Phone menu: Menu → Settings → Advanced (admin) → Auto Provision → Auto Provision Now.

## Cisco

### Three Cisco worlds

1. **SPA series** (Linksys lineage) — SPA122, SPA303, SPA504G, SPA525G, SPA112. Plaintext .cfg or XML profile. SIP. Easy to provision.
2. **7800/8800 series IP Phone** (modern enterprise) — 7821, 7841, 7861, 8841, 8851, 8861, 8865. Modern XML profile. Designed for CUCM but works "Multiplatform" (MPP) firmware standalone.
3. **7900 SCCP legacy** (7940/7960/7942/7945/7962) — Skinny Call Control Protocol, requires CUCM. Most have SIP firmware available; loading SIP firmware turns them into bog-standard SIP phones with `SEP<MAC>.cnf.xml` config.

### SPA config — .cfg key=value

```ini
# Per-line settings (line 1):
Display_Name_1_=Alice Smith
User_ID_1_=1001
Password_1_=hunter2
Auth_ID_1_=1001
Use_Auth_ID_1_=Yes
Proxy_1_=registrar.example.com
Outbound_Proxy_1_=sbc.example.com:5061
Use_Outbound_Proxy_1_=Yes
Register_1_=Yes
Make_Call_Without_Reg_1_=No
Ans_Call_Without_Reg_1_=No
Register_Expires_1_=3600
Ans_Call_Without_Reg_1_=No
SIP_Transport_1_=TLS

# Codec:
Preferred_Codec_1_=G711u
Use_Pref_Codec_Only_1_=No
Second_Preferred_Codec_1_=G722
Third_Preferred_Codec_1_=OPUS

# Provisioning:
Provision_Enable=Yes
Resync_On_Reset=Yes
Resync_Periodic=3600
Profile_Rule=https://prov.example.com/spa$MA.xml
Profile_Rule_B=
HTTPS_Cert_Verification=Yes
```

### 88xx Multiplatform (MPP)

Single XML profile. Modern. Supports HTTPS, mTLS, certificate pinning.

```xml
<flat-profile>
  <Profile_Rule ua="na">https://prov.example.com/$MA.xml</Profile_Rule>
  <Display_Name_1_ ua="na">Alice Smith</Display_Name_1_>
  <User_ID_1_ ua="na">1001</User_ID_1_>
  <Password_1_ ua="na">hunter2</Password_1_>
  <Proxy_1_ ua="na">registrar.example.com:5061</Proxy_1_>
  <SIP_Transport_1_ ua="na">TLS</SIP_Transport_1_>
</flat-profile>
```

### Dialplan dial-string

Cisco dialplan syntax — pattern matching on dialed digits:

```
(*xx|[3469]11|0|00|[2-9]xxxxxx|1xxx[2-9]xxxxxxS0|xxxxxxxxxxxx.)
# *xx           — star codes (e.g., *69)
# [3469]11      — 311, 411, 611, 911 (services)
# 0 / 00        — operator
# [2-9]xxxxxx   — 7-digit local
# 1xxx[2-9]xxxxxxS0 — 11-digit US LD, send immediately (S0)
# xxxxxxxxxxxx. — anything 12+, terminate on # or timeout
```

### CTL/ITL files (CUCM context)

For phones registered to Cisco Unified Communications Manager:

- **CTL** (Certificate Trust List) — list of CUCM trust anchors, signed by Cisco CTL Provider service.
- **ITL** (Identity Trust List) — phone's local trust list, derived from CTL on first boot.

If you swap CUCM clusters or change CallManager certs without CTL update, phones reject new server. Recovery: factory reset phone, or push new CTL out-of-band.

## Grandstream (GXP / GRP / DP / HT)

### File formats

- `config.txt` — global key=value (rare for new deploys)
- `cfg<MAC>.xml` — per-MAC XML
- `cfg<MAC>.bin` — encrypted/binary equivalent (vendor's "binary config tool" generates this)
- Master file: `cfg.xml` (uppercase HW: GXP / GRP) or `config.xml`

### XML format

```xml
<?xml version="1.0" encoding="UTF-8"?>
<gs_provision version="1">
  <config version="2">
    <P271>1</P271>             <!-- Account active -->
    <P270>Alice</P270>          <!-- Display Name -->
    <P47>registrar.example.com</P47>
    <P35>1001</P35>             <!-- SIP User ID -->
    <P36>1001</P36>             <!-- Auth ID -->
    <P34>hunter2</P34>          <!-- Password -->
    <P81>1</P81>                <!-- Use SIP user ID for auth -->
    <P52>3600</P52>             <!-- Register expiration sec -->
    <P25>1</P25>                <!-- Codec 1 preference (G.711u) -->
    <P26>2</P26>                <!-- Codec 2 (G.711a) -->
    <P27>9</P27>                <!-- Codec 3 (G.722) -->
  </config>
</gs_provision>
```

P-codes ("Pvalues") are Grandstream's per-setting numeric IDs. Reference table is in their tech doc per model.

### Binary config

Generated by the GS Configuration Tool (Windows). XML in, AES-encrypted .bin out. Phone has matching key built in via firmware. Used when you can't trust the transport (rare; just use HTTPS).

### Default passwords

- Admin GUI: `admin / admin` (older) or `admin / <random per device>` printed on box (newer GRP)
- End-user GUI: `user / 123` (older) or per-device sticker

**Always change in provisioning.**

```xml
<P2>newadminpwd</P2>      <!-- New admin password -->
<P196>newuserpwd</P196>   <!-- New end-user password -->
```

### GVS — Grandstream Video System

Grandstream's surveillance/video product line (GVR/GSC/GVS). Provisioning model is similar but uses different P-codes. Out of scope for IP phone but worth noting if you mix GS audio + video.

## Snom (3xx / 7xx / D series)

### Modern XML provisioning

Single master file: `settings.xml`. Minimal per-MAC overrides; most config is shared.

```xml
<?xml version="1.0" encoding="utf-8"?>
<settings>
  <phone-settings>
    <user_realname idx="1" perm="">Alice Smith</user_realname>
    <user_name idx="1" perm="">1001</user_name>
    <user_pname idx="1" perm="">1001</user_pname>
    <user_pass idx="1" perm="">hunter2</user_pass>
    <user_host idx="1" perm="">registrar.example.com:5061</user_host>
    <user_outbound idx="1" perm="">sip:sbc.example.com:5061;transport=tls</user_outbound>
    <user_active idx="1" perm="">on</user_active>
    <user_sipusername idx="1" perm="">1001</user_sipusername>
  </phone-settings>
</settings>
```

### Per-MAC override file

`<MAC>.xml` — Snom looks for this *after* settings.xml. Last write wins.

### Firmware download via provisioning

```xml
<phone-settings>
  <firmware_status perm="">https://prov.example.com/firmware/snom320.bin</firmware_status>
  <update_policy perm="">auto_update</update_policy>
</phone-settings>
```

### Default admin password

`0000` (four zeros). Change immediately:

```xml
<admin_mode_password perm="">newpwd</admin_mode_password>
```

## Mitel (6800/6900 series)

### XML provisioning

Mitel inherited the Aastra format after acquiring the company. File names: `aastra.cfg` (master), `<model>.cfg` (per-model), `<MAC>.cfg` (per-MAC).

```ini
# aastra.cfg style (key/value)
sip line1 number: 1001
sip line1 auth name: 1001
sip line1 password: hunter2
sip line1 display name: Alice Smith
sip line1 proxy ip: registrar.example.com
sip line1 proxy port: 5061
sip line1 registrar ip: registrar.example.com
sip line1 registrar port: 5061
sip line1 mode: 0
sip line1 transport: 4  # TLS
```

### CSP — Cloud-Link Service Platform

Mitel's hosted provisioning. Similar to Polycom RPRM. Used for hosted deployments at scale.

## Audiocodes (Mediant gateways, MP-1xx/2xx ATA, 4xx/5xx desk phones)

### INI-style provisioning

```ini
[ INI ]
;;; Configuration File
[BSP_PARAMS]
SyslogServerIP = 192.0.2.20
EnableSyslog = 1

[VOIP_PARAMS]
EnableMediaSecurity = 1
DTMFRelayMode = 0
EnableSilenceCompression = 1

[SIP_PARAMS]
ProxyName = registrar.example.com
ProxyIP = registrar.example.com
ProxyPort = 5061
EnableSIPSecure = 1  ; TLS
SIPTransportType = 2

[Account_Table]
FORMAT Account_Table_Index = Account_Table_AccountName, Account_Table_UserName, Account_Table_Password ...
Account_Table 0 = "Alice", "1001", "hunter2", ...
```

### URL placeholder substitution

Audiocodes URLs accept variables that the device substitutes:

```
https://prov.example.com/cfg/<MAC>.ini
https://prov.example.com/cfg/<MODEL>/<MAC>.ini
https://prov.example.com/cfg/<MAC>?ts=<TIMESTAMP>
```

| Variable | Substitution |
|---|---|
| `<MAC>` | Device MAC, no separators, lowercase |
| `<MODEL>` | Product name, e.g., `MP118` |
| `<TIMESTAMP>` | Unix epoch at fetch time |
| `<VER>` | Current firmware version |

## Avaya (J series, IX, 96xx legacy)

### 46xxsettings.txt

Single shared file. Per-phone differentiation via `IF $MACADDR SEQ XYZ GOTO foo` conditionals.

```
##########################################
# 46xxsettings.txt
##########################################

## Common
SET MCIPADD 192.0.2.10
SET HTTPSRVR 192.0.2.10
SET TLSSRVR 192.0.2.11
SET ENABLE_AVAYA_ENVIRONMENT 0
SET SIP_CONTROLLER_LIST registrar.example.com:5061;transport=tls
SET ENFORCE_SIPS_URI 1
SET SIPDOMAIN example.com
SET COUNTRY US
SET DSCPAUD 46
SET DSCPSIG 40
SET L2QVLAN 100

## Per-MAC overrides via conditionals:
IF $MACADDR SEQ 0004f23a8b6c GOTO ALICE
IF $MACADDR SEQ 0004f23a8b6d GOTO BOB
GOTO END

# ALICE
SET SIPUSERNAME 1001
SET SIPHA1 hunter2-md5-hash
GOTO END

# BOB
SET SIPUSERNAME 1002
SET SIPHA1 bobsecret-md5-hash
GOTO END

# END
```

### H.323 vs SIP firmware split

Avaya phones can run two firmwares. Same hardware, different protocol stack:

- **H.323 firmware** — talks to Avaya Aura Communication Manager via H.323. Legacy enterprise.
- **SIP firmware** — talks to Aura SM (Session Manager) or third-party SIP. Modern.

Switching requires loading the other firmware via provisioning. Once loaded, phone factory-reset, fetches config matching new protocol.

### Avaya Aura System Manager dependency

For H.323 / native Avaya SIP, phones expect to register against Aura Session Manager, which is configured by Aura System Manager. If you're running standalone SIP (third-party PBX), you skip this entirely; phone just registers to your registrar.

## Cookbook — Standard Provisioning Server Setup

### nginx + per-MAC configs

```bash
# /etc/nginx/sites-available/provisioning
server {
  listen 443 ssl http2;
  server_name prov.example.com;

  ssl_certificate /etc/letsencrypt/live/prov.example.com/fullchain.pem;
  ssl_certificate_key /etc/letsencrypt/live/prov.example.com/privkey.pem;
  ssl_protocols TLSv1.2 TLSv1.3;

  root /var/www/provisioning;
  index 0000000000000.cfg;

  # Strict MAC pattern; case-insensitive for vendor variations
  location ~* "^/(?<mac>[0-9a-f]{12})\.(cfg|xml)$" {
    auth_basic "Provisioning";
    auth_basic_user_file /etc/nginx/htpasswd-provisioning;
    try_files /$mac.cfg /$mac.xml =404;
  }

  # Common files
  location ~* "\.(cfg|xml|txt|json|ini)$" {
    auth_basic "Provisioning";
    auth_basic_user_file /etc/nginx/htpasswd-provisioning;
  }

  # Firmware downloads — separate auth or none
  location /firmware/ {
    autoindex off;
  }

  access_log /var/log/nginx/prov-access.log;
  error_log /var/log/nginx/prov-error.log;
}

# Force HTTP→HTTPS redirect
server {
  listen 80;
  server_name prov.example.com;
  return 301 https://$host$request_uri;
}
```

### Per-MAC config generator

```python
#!/usr/bin/env python3
# gen_phone_cfg.py
from jinja2 import Environment, FileSystemLoader
import csv, os, sys

env = Environment(loader=FileSystemLoader('templates'))
tpl = env.get_template('yealink_t54w.cfg.j2')

with open('phones.csv') as f:
  for row in csv.DictReader(f):
    mac = row['mac'].lower().replace(':', '').replace('-', '')
    rendered = tpl.render(
      mac=mac,
      ext=row['ext'],
      name=row['name'],
      sip_user=row['ext'],
      sip_password=row['password'],
      registrar='registrar.example.com',
      sbc='sbc.example.com',
    )
    out = f'/var/www/provisioning/{mac}.cfg'
    with open(out, 'w') as o:
      o.write(rendered)
    os.chmod(out, 0o644)
    print(f'wrote {out}')
```

### CSV format

```csv
mac,ext,name,password
0004f23a8b6c,1001,Alice Smith,hunter2
0004f23a8b6d,1002,Bob Jones,xKL93kf
```

### Jinja2 template (`templates/yealink_t54w.cfg.j2`)

```
account.1.enable = 1
account.1.label = "{{ name }}"
account.1.display_name = "{{ name }}"
account.1.user_name = {{ sip_user }}
account.1.auth_name = {{ sip_user }}
account.1.password = {{ sip_password }}
account.1.sip_server.1.address = {{ registrar }}
account.1.sip_server.1.port = 5061
account.1.sip_server.1.transport_type = 2
account.1.outbound_proxy_enable = 1
account.1.outbound_host = {{ sbc }}
account.1.outbound_port = 5061
local_time.ntp_server1 = pool.ntp.org
auto_provision.url = https://prov.example.com/
auto_provision.repeat.enable = 1
auto_provision.repeat.minutes = 1440
static.security.user_password = admin:{{ random_admin_pwd() }}
```

### Let's Encrypt for HTTPS

```bash
sudo apt install certbot python3-certbot-nginx
sudo certbot --nginx -d prov.example.com
sudo systemctl enable certbot.timer  # auto-renewal
```

### DHCP option 66 if all phones support it; per-vendor otherwise

```bash
# /etc/dhcp/dhcpd.conf (ISC dhcpd):
option tftp-server-name "https://prov.example.com/";

# Per-vendor:
option-150 192.0.2.10;
option-160 "https://prov.example.com/";
```

### HTTP basic auth with rotating creds

```bash
# Generate htpasswd:
htpasswd -B -c /etc/nginx/htpasswd-provisioning phoneuser
# Phones get this user/pass in their bootstrap config.
# Rotate quarterly: re-render configs, push, restart nginx.
```

### Cert-based mutual TLS (high security)

```nginx
ssl_client_certificate /etc/nginx/ca-trust.pem;
ssl_verify_client on;
ssl_verify_depth 2;

# Phone must present its MIC (manufacturer cert) signed by vendor's CA.
# Trust the vendor CA bundle in /etc/nginx/ca-trust.pem.
```

## Cookbook — Asterisk + chan_pjsip + provisioning

### `pjsip.conf`

```ini
[transport-tls]
type=transport
protocol=tls
bind=0.0.0.0:5061
external_media_address=203.0.113.10
external_signaling_address=203.0.113.10
cert_file=/etc/asterisk/keys/asterisk.crt
priv_key_file=/etc/asterisk/keys/asterisk.key
method=tlsv1_2
verify_client=no

[1001](!)
type=endpoint
context=internal
disallow=all
allow=opus,g722,ulaw,alaw
auth=auth1001
aors=1001
direct_media=no
rtp_symmetric=yes
force_rport=yes
rewrite_contact=yes

[auth1001]
type=auth
auth_type=userpass
username=1001
password=hunter2

[1001]
type=aor
max_contacts=1
remove_existing=yes
qualify_frequency=30
```

### Matching phone config

```ini
# Yealink:
account.1.user_name = 1001
account.1.auth_name = 1001
account.1.password = hunter2
account.1.sip_server.1.address = asterisk.example.com
account.1.sip_server.1.port = 5061
account.1.sip_server.1.transport_type = 2  # TLS
```

### Verify

```bash
asterisk -rx "pjsip show registrations"
# Expect:
#   <Registration/ServerURI..............................>  <Auth..........>  <Status.......>
#   1001/1001                                                auth1001          Registered

asterisk -rx "pjsip show contacts"
# Expect:
#   Contact:  1001/sip:1001@10.0.0.42:5061^3B               Avail        12.345

asterisk -rx "pjsip show endpoint 1001"
# Aor:  1001
#   Contact:  1001/sip:1001@10.0.0.42:5061^3B  Avail
```

## Cookbook — FreeSWITCH + Verto / SIP

### `directory/default/1001.xml`

```xml
<include>
  <user id="1001">
    <params>
      <param name="password" value="hunter2"/>
      <param name="vm-password" value="1001"/>
    </params>
    <variables>
      <variable name="toll_allow" value="domestic,international,local"/>
      <variable name="accountcode" value="1001"/>
      <variable name="user_context" value="default"/>
      <variable name="effective_caller_id_name" value="Alice Smith"/>
      <variable name="effective_caller_id_number" value="1001"/>
      <variable name="callgroup" value="techsupport"/>
    </variables>
  </user>
</include>
```

### Dial-string in `vars.xml`

```xml
<X-PRE-PROCESS cmd="set" data="default_password=hunter2"/>
<X-PRE-PROCESS cmd="set" data="domain=example.com"/>
<X-PRE-PROCESS cmd="set" data="domain_name=$${domain}"/>
```

### Phone config matches

```ini
account.1.user_name = 1001
account.1.auth_name = 1001
account.1.password = hunter2
account.1.sip_server.1.address = freeswitch.example.com
account.1.sip_server.1.port = 5060
```

### Verify

```bash
fs_cli -x "sofia status profile internal reg 1001"
# Expect rows showing the registered contact URI.
```

## Authentication Methods

### Plain HTTP (no auth) — bad

Anyone on the network can fetch any phone's config — including SIP password. Toll fraud waiting to happen.

### HTTP Basic auth — better, encrypt the channel

```
Authorization: Basic cGhvbmU6c2VjcmV0
```

Base64-encoded `phone:secret`. Trivially decoded; **must be over TLS**, otherwise it's just plaintext-with-extra-steps.

### HTTPS with mutual TLS — best

Both server and client present certificates. Server validates client's cert against trusted CA bundle. Phones with vendor-provisioned MIC (Manufacturer-Installed Certificate) authenticate to your server *as their physical hardware*. Server enforces "MAC must match cert CN."

### X.509 device cert + CSR

Some deployments generate per-device certs:

```bash
# Generate CSR for phone with MAC 0004f23a8b6c:
openssl req -new -newkey rsa:2048 -nodes \
  -keyout 0004f23a8b6c.key \
  -out 0004f23a8b6c.csr \
  -subj "/CN=0004f23a8b6c.phone.example.com/O=Example/C=US"

openssl x509 -req -in 0004f23a8b6c.csr \
  -CA ca.crt -CAkey ca.key -CAcreateserial \
  -out 0004f23a8b6c.crt -days 730 -sha256
```

Phone fetches its cert + key during initial provisioning (over a short-lived bootstrap creds), then uses them thereafter.

### MIC (Manufacturer-Installed Certificate)

Per-device cert burned in at the factory. Cisco, Polycom, Yealink, Snom all ship MICs in modern phones. Cert chain rooted at the vendor's CA.

```bash
# Verify a Cisco phone MIC:
openssl s_client -connect <phone-ip>:443 -showcerts
# Subject: CN=SEP0004F23A8B6C, ...
# Issuer: CN=Cisco Manufacturing CA, ...
```

Use cases:

- Phone authenticates to provisioning server with MIC, no shared password needed.
- Phone authenticates to PBX (CUCM) with MIC for secure SIP.
- Audit trail: MIC ties config fetch to physical hardware.

## Centralized Templating

### Off-the-shelf

| Tool | Notes |
|---|---|
| FreePBX endpoint manager | Free, ties to FreePBX/PBXact. Multi-vendor templates. |
| 3CX provisioning | Built into 3CX phone system. Web UI–driven. |
| Skyetel managed provisioning | Hosted carrier-managed provisioning. |
| Twilio Elastic SIP managed | Carrier provisioning for Twilio numbers. |
| AskoziaPBX (legacy) | OSS PBX with phone provisioning. Less active. |
| Sangoma PBXact / FreePBX | Commercial wrapper around FreePBX. |
| KazooUI | Hosted PBX with provisioning UI. |

### Custom Jinja2 + Python

```python
from jinja2 import Environment, FileSystemLoader
env = Environment(loader=FileSystemLoader('templates'))

template = env.get_template('polycom_vvx411.j2')
output = template.render(
  mac='0004f23a8b6c',
  ext='1001',
  name='Alice Smith',
  password='hunter2',
  server='registrar.example.com',
  domain='example.com',
)
print(output)
```

### Common template variables

```
{MAC}          — phone MAC (lowercase, no separators)
{MAC_UPPER}    — uppercase MAC
{EXT}          — extension number
{NAME}         — display name
{PASSWORD}     — SIP password
{SERVER}       — SIP registrar host
{SBC}          — outbound SBC host
{DOMAIN}       — SIP domain
{NTP}          — NTP server
{TZ}           — timezone string
{VOICE_VLAN}   — VLAN ID
{ADMIN_PWD}    — randomized admin password
{LINEKEY_N}    — line key N target
{BLF_N}        — busy lamp field target N
```

## Firmware Management

### Phone-side: firmware URL in config

```xml
<!-- Polycom -->
<APPLICATION
  APP_FILE_PATH="firmware/sip.ld"
  APP_FILE_PATH_VVX411="firmware/3111-65190-001.sip.ld"
/>
```

```ini
# Yealink:
static.firmware.url = https://prov.example.com/firmware/T54W-91.86.0.5.rom
static.auto_image_check.enable = 1
```

### Vendor: per-model firmware update channel

| Vendor | Firmware Source |
|---|---|
| Polycom | https://www.poly.com/us/en/support/downloads-licenses (IRP / Edge / VVX) |
| Yealink | https://www.yealink.com/en/product-list?type=firmware |
| Cisco | https://software.cisco.com/download/home (account required) |
| Grandstream | https://www.grandstream.com/support/firmware |
| Snom | https://service.snom.com/display/wiki/Snom+Firmware |
| Mitel | https://www.mitel.com/document-center/devices-and-accessories |

### Rollback strategy

Keep N–1 (and ideally N–2) firmware available on provisioning server. If a new firmware breaks something:

```bash
ls /var/www/provisioning/firmware/
T54W-91.86.0.5.rom         # current
T54W-91.85.0.30.rom        # previous (N-1)
T54W-91.84.0.125.rom       # 2 versions back (N-2)
```

To rollback: edit common config, change `static.firmware.url` to N-1 path, push.

### Test firmware in lab before mass deploy

The "always test firmware in lab" rule. Set up 2–3 spare phones. Push new firmware to lab. Run regression: register, place call, transfer, hold, BLF, voicemail, presence. **Then** push to production. Skipping this turns one vendor bug into 500 angry users.

## NAT Considerations

### STUN

Some phones support STUN natively; phone discovers public IP via STUN server, embeds in SDP/SIP.

```ini
# Yealink:
static.network.stun.enable = 1
static.network.stun.server = stun.l.google.com
static.network.stun.port = 19302
```

### Outbound proxy at SBC

Better than STUN for production. Phone sends all SIP/RTP through your Session Border Controller, which handles NAT for it. Phone config points to SBC as outbound proxy:

```ini
account.1.outbound_proxy_enable = 1
account.1.outbound_host = sbc.example.com
account.1.outbound_port = 5061
```

### Keepalive intervals

Phones send periodic keepalives to keep NAT pinholes open. Common intervals:

| Vendor | Default | Range |
|---|---|---|
| Polycom | 30s | 5–3600s |
| Yealink | 30s | 15–60s typically |
| Cisco SPA | 60s | configurable |
| Grandstream | 20s | 1–64s |

```ini
# Yealink:
account.1.nat.keep_alive_type = 1  # 0=disable, 1=default (CRLF), 2=double-CRLF
account.1.nat.keep_alive_interval = 30
```

### Prefer keepalive over symmetric NAT mode

Symmetric NAT (where the public port differs per destination) breaks STUN, hairpinning, and many SIP NAT helpers. The right answer is *not* to fight NAT with cleverness, but to keep a long-lived registration with frequent keepalives so the binding never expires. 30-second keepalives + outbound proxy at SBC = NAT solved.

## Quality of Service (QoS)

### DiffServ DSCP marking

```ini
# Yealink:
network.qos.signal_dscp = 40   # CS5 — call signaling
network.qos.voice_dscp = 46    # EF — voice media
network.qos.video_dscp = 34    # AF41 — video media
```

| DSCP | Decimal | Use |
|---|---|---|
| EF (Expedited Forwarding) | 46 | Voice RTP — strict-priority queue |
| CS5 | 40 | SIP signaling |
| AF41 | 34 | Video RTP |
| CS3 | 24 | Less common signaling |
| BE | 0 | Default; data |

### 802.1Q VLAN tagging

```ini
# Yealink:
network.vlan.enable = 1
network.vlan.vid = 100         # Voice VLAN
network.vlan.priority = 5      # 802.1p (CoS)
network.pc_port.enable = 1     # Pass-through port
network.pc_port.vlan.enable = 0  # Untag PC traffic
```

### Voice VLAN (LLDP-MED for auto-discovery)

Switch advertises voice VLAN via LLDP-MED. Phone auto-tags. No phone config needed if switch is configured properly:

```cisco
! Cisco IOS:
interface gigabitethernet0/1
  switchport access vlan 10        ! data VLAN
  switchport voice vlan 100        ! voice VLAN
  spanning-tree portfast
  lldp transmit
  lldp receive
  lldp med-tlv-select network-policy
  power inline auto
```

### Bandwidth tuning: codec selection drives this

| Codec | Bitrate | RTP payload (with G.711-style 20ms framing) | Bandwidth/call (incl. headers) |
|---|---|---|---|
| G.711 (PCMU/PCMA) | 64 kbps | 160 bytes/20ms | ~87 kbps |
| G.722 | 64 kbps | 160 bytes/20ms | ~87 kbps |
| G.729 | 8 kbps | 20 bytes/20ms | ~31 kbps |
| OPUS @ 24 kbps | 24 kbps | variable | ~47 kbps |
| OPUS @ 16 kbps | 16 kbps | variable | ~39 kbps |
| iLBC | 13.3 kbps | 38 bytes/30ms | ~31 kbps |

Pick OPUS where supported (best quality/bandwidth ratio), fall back to G.722 (HD voice on G.711 bandwidth) or G.711.

## LLDP-MED — Link-Layer Discovery Protocol Media Endpoint Discovery

Layer-2 protocol where switch tells phone, and phone tells switch:

- Voice VLAN ID
- 802.1p priority class
- DSCP marking
- PoE class request/response
- Civic location (E911)
- Phone capabilities

### Boot sequence with LLDP-MED

```
1. Phone boots, untagged on access VLAN (e.g., VLAN 1)
2. Phone sends LLDP advertisement (capabilities = phone)
3. Switch sees LLDP, replies with LLDP-MED Network Policy TLV
   advertising "voice VLAN 100, priority 5, DSCP 46"
4. Phone parses, retags itself onto VLAN 100
5. DHCP request on VLAN 100 → gets voice subnet IP
6. Phone now on voice VLAN, fetches provisioning, registers
```

### PoE classification via LLDP-MED

Phone advertises required power class. Switch budgets accordingly:

```cisco
power inline auto
power inline port priority high
```

### Civic location

Switch knows building/floor/room, advertises to phone. Phone sends location with E911 calls:

```cisco
voice-card 0
location civic-location identifier 1
  building "HQ"
  floor "3"
  room "302"
  city "Austin"
  state "TX"
  country US
!
interface gi0/1
  location civic-location-id 1
```

## PoE — Power over Ethernet

| Standard | Class | Max Watts (port) | Min Watts (device) |
|---|---|---|---|
| 802.3af (PoE) | 0 | 15.4 | 0.44–12.95 |
| 802.3af (PoE) | 1 | 4.0 | 0.44–3.84 |
| 802.3af (PoE) | 2 | 7.0 | 3.84–6.49 |
| 802.3af (PoE) | 3 | 15.4 | 6.49–12.95 |
| 802.3at (PoE+) | 4 | 30.0 | 12.95–25.50 |
| 802.3bt (PoE++/Type 3) | 5–6 | 60 | up to 51 |
| 802.3bt (PoE++/Type 4) | 7–8 | 100 | up to 73 |

### Phone class request

Most desk phones are Class 1 or 2 (under 7W). Conference phones (SoundStation, Yealink CP9xx) often Class 3 (12W+). Phones with HD video/large color displays and accessories climb to Class 4.

### Switch port budgeting

```cisco
show power inline
! Watch for "Reserved Power" approaching "Total Power" — over-budget
! Switch will deny PoE on additional ports if budget exhausted.

show power inline gi0/1
! Per-port detail.
```

## Provisioning Cycle

### Happy path

```
1. Phone boots
2. DHCP DISCOVER → DHCP OFFER (IP, gateway, DNS, NTP, Option 66/150/160)
3. Phone parses options → builds provisioning URL
4. (Optional) LLDP-MED → re-tag onto voice VLAN → re-DHCP
5. (Optional) NTP sync → clock now valid
6. HTTPS GET https://prov.example.com/000000000000.cfg
   → master config returned (or 404 → some phones try 0000000000000.cfg
                                    or vendor-specific master name)
7. Parse master, identify per-MAC file
8. HTTPS GET https://prov.example.com/0004f23a8b6c.cfg
9. Apply config
10. If firmware mismatch: download firmware, reboot, GOTO 1
11. If config-side reboot required: reboot, GOTO 1
12. SIP REGISTER (over TLS, port 5061) to registrar
13. 200 OK with registered contact
14. Phone shows extension, ready
```

### Failure modes

```
A. DHCP option missing
   → phone times out, displays "no configuration server"
   → cure: fix DHCP scope

B. Wrong server URL
   → phone times out, "cannot reach server"
   → cure: confirm DHCP option content

C. DNS issue
   → phone resolves provisioning hostname → NXDOMAIN
   → cure: fix DNS or use IP in DHCP option

D. Firmware not present
   → phone fetches config that says "use firmware sip.ld" but URL 404
   → cure: copy firmware to webroot

E. Password wrong
   → 401 Unauthorized on cfg fetch
   → cure: verify htpasswd matches phone's bootstrap creds
```

## Common Errors

### "Failed to connect to provisioning server"

DHCP option 66/150/160 missing or pointing at unreachable host. Verify `tcpdump -i eth0 'port 67 or port 68'` shows DHCP OFFER carries the option.

### "401 Unauthorized" on cfg fetch

HTTP Basic auth mismatch. Check `auto_provision.user_name` / `password` in phone bootstrap config matches `htpasswd -v <user>` test on server.

### "404 Not Found"

Filename pattern doesn't match server's content. Phone is requesting `0004F23A8B6C.cfg` (uppercase) but file on disk is `0004f23a8b6c.cfg` (lowercase). nginx is case-sensitive by default. Fix: `location ~* "..."` (case-insensitive regex) or rename file.

### "Certificate validation failed"

TLS cert untrusted. Causes:

- Self-signed cert, phone doesn't trust your CA
- Cert expired
- Cert CN doesn't match URL hostname
- NTP not set; phone thinks "now" is 1970, cert's `notBefore` is in the future from phone's perspective → "not yet valid"

Cure: install root CA on phone (vendor-specific provisioning option) or use a Let's Encrypt cert that's in the phone's default trust store.

### "Firmware download failed"

- Model mismatch: pushed VVX 411 firmware to a VVX 501. Cure: model-specific path.
- Firmware file missing at URL.
- Firmware file corrupted. Cure: re-download from vendor and re-checksum.

### "REGISTER 401 Unauthorized"

Phone got config but SIP credentials are wrong on PBX side. `pjsip show endpoints` to confirm extension exists and password matches.

### "REGISTER 408 Request Timeout"

NAT issue. Phone sends REGISTER, never sees response.

- Outbound proxy not configured
- SBC blocking
- ICMP "destination unreachable" being filtered

```bash
# On the PBX:
tcpdump -i any -n 'port 5060 or port 5061' -w sip.pcap
# Look for REGISTER from phone's public IP, then check what we send back.
# If our reply is fine, phone-side firewall is dropping.
```

### "Time not set"

NTP broken. DHCP option 42 missing, or NTP server unreachable from voice VLAN. Fix: confirm phone can reach NTP host on UDP 123.

## Common Gotchas — broken→fixed

### Wrong DHCP option (66 vs 160 vs 150)

Broken:
```
# DHCP scope:
option-66 "https://prov.example.com/"
# But phone is a Polycom that only reads 160.
```

Fixed: set 66, 150, AND 160 to same URL. Belt-and-braces.

```
option tftp-server-name "https://prov.example.com/";
option-150 192.0.2.10;
option-160 "https://prov.example.com/";
```

### URL trailing slash mismatch

Broken:
```
DHCP option 66: "https://prov.example.com"     (no trailing slash)
Phone tries: https://prov.example.com0004f23a8b6c.cfg
                                    ^^^ no separator → 404
```

Fixed: always include trailing slash in DHCP options. Phone appends filename to URL.

```
option-66 "https://prov.example.com/"
```

### File permission 600 on web server

Broken:
```bash
$ ls -l /var/www/provisioning/0004f23a8b6c.cfg
-rw------- 1 deploy deploy 4096 Apr 25 12:00 0004f23a8b6c.cfg
# nginx runs as www-data; can't read.
```

Fixed:
```bash
chmod 644 /var/www/provisioning/*.cfg
chown www-data:www-data /var/www/provisioning/*.cfg
# Or 640 + add www-data to deploy group.
```

### MAC case mismatch

Broken:
```bash
$ ls /var/www/provisioning/
0004F23A8B6C.cfg     # uppercase
# Phone fetches 0004f23a8b6c.cfg (lowercase); nginx case-sensitive → 404.
```

Fixed: case-insensitive nginx regex, or generate both, or rename:

```nginx
location ~* "^/(?<mac>[0-9a-f]{12})\.cfg$" {
  try_files $uri $uri.upper $uri.lower =404;
}
```

```bash
# Or rename:
for f in *.cfg; do mv "$f" "$(echo $f | tr A-Z a-z)"; done
```

### Mixed HTTP/HTTPS across vendors

Broken:
```
# Polycom and Yealink work fine over HTTPS.
# Old Cisco SPA122 firmware silently fails HTTPS handshake → falls
# back to HTTP → 403 (HTTPS-only nginx).
```

Fixed: dual-listen, redirect HTTP only for paths that don't matter. Or: upgrade legacy phones.

```nginx
server { listen 80; ... auth_basic ...; }
server { listen 443 ssl; ... }
```

### DNS not resolving inside phone VLAN

Broken:
```
# Office DNS is 10.0.0.1; voice VLAN is 192.168.20.0/24 with DHCP
# pointing at 8.8.8.8. Voice VLAN firewall blocks egress UDP 53.
# Phone can't resolve prov.example.com.
```

Fixed: voice VLAN DHCP option 6 = internal DNS reachable from voice subnet:

```
subnet 192.168.20.0 netmask 255.255.255.0 {
  option domain-name-servers 192.168.20.1;
  ...
}
```

### NTP not set; cert validation fails

Broken:
```
# Phone clock = 1970-01-01, cert valid 2025-2026. Phone says
# "cert not yet valid", aborts TLS.
```

Fixed: DHCP option 42 + bake NTP into bootstrap config:

```
option ntp-servers 192.168.20.1;
```

```ini
local_time.ntp_server1 = pool.ntp.org
```

### Vendor's "config encryption" enabled but no key

Broken:
```
# Yealink config-encrypt feature enabled in firmware, but
# provisioning server pushes plaintext. Phone rejects config.
```

Fixed: either disable encryption in firmware, or generate AES-CBC encrypted blob with vendor's tool and matching key:

```ini
# Disable:
static.auto_provision.aes_key_in_file = 0
static.auto_provision.aes_key_16.com = ""
```

### common.cfg loaded but per-MAC override missing

Broken:
```
# Phone fetches y0000000000XX.cfg (common) → registers to placeholder
# extension 9999 from common.cfg. Per-MAC file 404'd silently.
```

Fixed: monitor 404s on provisioning server, alert on them.

```nginx
error_log /var/log/nginx/prov-error.log;
# Plus a cron:
grep '404' /var/log/nginx/prov-access.log | mail -s "Phone 404s" ops@
```

### Default password not changed

Broken:
```
# Yealink phones still admin/admin on web GUI in production.
# Anyone on voice VLAN can reconfigure.
```

Fixed: every config must rotate web GUI password:

```ini
static.security.user_password = admin:$RANDOM_LONG_PASSWORD
static.security.user_password = user:$RANDOM_LONG_PASSWORD
```

### Old firmware can't parse new config schema

Broken:
```
# Pushed config with `account.1.sip_server.1.transport_type = 2`,
# but firmware on phone is 5 years old and doesn't recognize that key.
# Phone silently ignores TLS, falls back to UDP.
```

Fixed: bake firmware version into config, force upgrade:

```ini
static.firmware.url = https://prov.example.com/firmware/T54W-91.86.0.5.rom
static.auto_image_check.enable = 1
```

### Provisioning over WiFi without WPA-Enterprise

Broken:
```
# Phone has WPA-PSK with shared password, fetches config over HTTP
# (not HTTPS). SIP password leaks on every fetch via wireless sniffing.
```

Fixed: WPA2-Enterprise with EAP-TLS (per-device cert) OR HTTPS-only provisioning OR VPN tunnel for the WiFi voice SSID. Best: avoid WiFi for desk phones; use Ethernet.

## Diagnostic Tools

### tcpdump on provisioning server interface

```bash
sudo tcpdump -i eth0 -n -s 0 -w /tmp/prov.pcap \
  '(tcp port 80 or tcp port 443 or tcp port 5061 or udp port 67 or udp port 68 or udp port 69)'

# Then open in Wireshark:
wireshark /tmp/prov.pcap
# Filter: http.request or tls.handshake.type == 1
```

### Phone's web UI status page

| Vendor | URL | Default |
|---|---|---|
| Polycom | `https://<phone-ip>/` | `Polycom` / `456` (older) or per-config |
| Yealink | `https://<phone-ip>/` | `admin` / `admin` |
| Cisco SPA | `http://<phone-ip>/` | (from config) |
| Cisco 88xx MPP | `https://<phone-ip>/` | `admin` / (none, must be set) |
| Grandstream | `http://<phone-ip>/` | `admin` / `admin` |
| Snom | `http://<phone-ip>/` | (admin user) / `0000` |

Status page shows: registration state, last provisioning fetch, firmware version, IP/MAC, codec in use.

### Phone's debug log

- **Polycom**: `Settings → Status → Diagnostics → System logs` or push log via `log.render.level.5 = 1` in cfg.
- **Yealink**: `Settings → Status → System Status → Diagnostics` or web GUI Log Settings → set Module Log = 6, export.
- **Cisco SPA**: `http://<phone-ip>/admin/syslog.dump`
- **Grandstream**: `Settings → Maintenance → Upgrade and Provisioning → Reset Provisioning Path`; logs at `http://<phone-ip>/?log` (older) or web GUI logs section.
- **Snom**: `http://<phone-ip>/log.htm`

### Wireshark on phone-side mirror port

```cisco
! Cisco port mirror:
monitor session 1 source interface gi0/1 both
monitor session 1 destination interface gi0/24 encapsulation replicate
```

Plug a laptop into gi0/24, run Wireshark. See exactly what phone sends and receives.

### Factory reset then re-provision

| Vendor | Method |
|---|---|
| Polycom | Hold dialpad `*` while powering on, enter MAC password |
| Yealink | Press OK 10 sec, or web GUI Settings → Upgrade → Reset to Factory |
| Cisco SPA | Hold `*` then dial `73738#` from idle screen, confirm |
| Grandstream | Hold `*` 7 sec while booting, or web GUI Maintenance → Upgrade → Factory Reset |
| Snom | Boot, hold `Settings`, dial `\*0000` enter |

Use sparingly; each factory-reset re-fetches config and writes flash.

## Bulk Deployment

### Pre-import MAC list via vendor's web tool

Yealink RPS, Polycom ZTP, Grandstream GDMS — bulk CSV upload of MACs and target URL. Submit before phones arrive.

### Generate config files in CI

```yaml
# .github/workflows/phone-configs.yml
name: Generate phone configs
on: { push: { paths: [ 'phones.csv', 'templates/**' ] } }
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: pip install jinja2
      - run: python gen_phone_cfg.py
      - run: rsync -av output/ deploy@prov.example.com:/var/www/provisioning/
```

### Verify reachability before shipping

```bash
# In factory or warehouse, before boxing:
for ip in $(arp -a | grep yealink | awk '{print $2}' | tr -d '()'); do
  curl -s "http://$ip/" -u admin:admin | grep -q "Yealink" && echo "$ip OK" || echo "$ip FAIL"
done
```

### Burn-in rack

Provision lab rack of N phones. Run automated test:

```bash
# Place test calls between phones:
for src in 1001 1002 1003; do
  for dst in 1001 1002 1003; do
    [ "$src" = "$dst" ] && continue
    sipp -sn uac -m 1 -s $dst -i 192.168.20.$src ...
  done
done
```

If all pass, deploy. If any fail, fix template or config before mass deploy.

## Security

### HTTPS-only

```nginx
server {
  listen 80;
  return 301 https://$host$request_uri;
}
server {
  listen 443 ssl http2;
  ssl_protocols TLSv1.2 TLSv1.3;
  ssl_ciphers HIGH:!aNULL:!MD5:!3DES;
  ssl_prefer_server_ciphers on;
  ...
}
```

### Per-MAC credentials

Each phone's config contains its own SIP password. PBX has matching auth. If one phone is compromised, only one extension is at risk.

```ini
# Per-MAC config:
account.1.password = $UNIQUE_PER_MAC_PASSWORD
```

### Disable default admin/admin

```ini
static.security.user_password = admin:$LONG_RANDOM
```

### Restrict provisioning server to specific IP/network

```nginx
location / {
  allow 192.168.20.0/24;   # voice VLAN
  allow 10.10.0.0/16;      # remote workers VPN
  deny all;
}
```

Also pin firewall rules:

```
iptables -A INPUT -p tcp --dport 443 -s 192.168.20.0/24 -j ACCEPT
iptables -A INPUT -p tcp --dport 443 -j DROP
```

### Audit logs

```nginx
log_format provauditlog '$remote_addr - $remote_user [$time_local] "$request" '
                       '$status $body_bytes_sent "$http_user_agent"';
access_log /var/log/nginx/prov-audit.log provauditlog;
```

Aggregate to central log (Loki, Splunk, ELK). Alert on:

- 401/403 spikes (someone probing)
- Unknown User-Agent (not a known phone vendor)
- Source IP outside expected range

### Cert pinning

Some phones support pinning specific cert fingerprints:

```ini
# Yealink:
static.security.trust_certificates = 1
static.security.cn_validation = 1
static.security.ca_cert = /config/certs/our-ca.pem
```

## E911 / Emergency Call Handling

### Civic address per phone

Set during provisioning. Required for accurate dispatch on 911 calls.

```xml
<!-- Polycom -->
<device.set
  device.location.civic.country="US"
  device.location.civic.A1="TX"        <!-- State -->
  device.location.civic.A3="Austin"    <!-- City -->
  device.location.civic.PRD=""          <!-- Pre-direction -->
  device.location.civic.RD="Main"       <!-- Road -->
  device.location.civic.STS="St"        <!-- Street suffix -->
  device.location.civic.HNO="123"       <!-- House number -->
  device.location.civic.LOC="Suite 400" <!-- Location -->
  device.location.civic.PC="78701"      <!-- Postal code -->
/>
```

### LLDP-MED for location-aware

Switch port has the location, advertises to phone via LLDP-MED Location Identification TLV. Phone auto-picks up. No phone config needed.

```cisco
voice-card 0
location civic-location identifier 1
  building "HQ"
  floor "3"
  room "302"
  city "Austin"
  state "TX"
  country US
  postal-code "78701"
!
interface gi0/1
  location civic-location-id 1
  description "Alice desk"
```

### Kari's Law / RAY BAUM's Act compliance (US)

- **Kari's Law** (2018) — multi-line telephone systems must support direct 911 dialing without a prefix (no "9-911"); must signal a notification (email, SMS) to a designated party at the site.
- **RAY BAUM's Act** (Section 506, 2018) — 911 calls must include "dispatchable location" — building, floor, room — sufficient for first responders to find the caller.

Provisioning implications:

```
1. Dial plan includes 911 (no 9- prefix needed):
   - In Yealink:
     dialplan.dialnow.rule.1.value = 911
     dialplan.dialnow.rule.1.line = 0  # any line
   - In Polycom:
     dialplan.1.digitmap = "911|9911|..."
2. Phone has civic location set per the above.
3. PBX/SBC routes 911 to E911 service (Bandwidth, Intrado, etc.) with
   ELIN (Emergency Location Identification Number) or HELD (HTTP-Enabled
   Location Delivery) signaling.
4. PBX sends SMS/email to security@example.com on 911 dial.
```

### Location-via-MAC database

Maintain a CSV/database mapping MAC → physical location. Phone moves desks → location updated. Some PBXs (e.g., 3CX, FreePBX with E911 module) handle this automatically.

```csv
mac,bldg,floor,room,address1,city,state,zip
0004f23a8b6c,HQ,3,302,123 Main St Suite 400,Austin,TX,78701
0004f23a8b6d,HQ,3,302,123 Main St Suite 400,Austin,TX,78701
```

## Idioms

- **"Always HTTPS"** — Plain HTTP for provisioning is malpractice. Configs contain SIP passwords.
- **"DHCP option per-vendor cheat sheet"** — set 66, 150, AND 160 to the same URL; let each vendor pick.
- **"Bake NTP early"** — DHCP option 42 *and* config file. Without good time, TLS dies.
- **"Use vendor's redirector for global zero-touch"** — Polycom RPRM / Yealink RPS / GDMS. No need for VPN to reach customer site for first-boot.
- **"Audit fetched configs"** — log every fetch. Detect probing, missing files, stale clients.
- **"Lab before prod"** — every firmware, every template change. Always.
- **"One template per model, one config per phone"** — DRY at the model layer, unique at the device layer.
- **"Rotate creds quarterly"** — both SIP passwords and provisioning HTTP basic.
- **"Voice VLAN, always"** — separate broadcast domain, separate QoS, separate ACLs.
- **"Disable web GUI in production after provisioning"** — set admin password, then disable HTTP entirely if vendor allows.
- **"Test 911 every quarter"** — Kari's Law compliance. Place a test call to a non-emergency line that simulates 911 routing.
- **"Backups of the provisioning server are sacred"** — without them, mass redeploy is ugly.

## See Also

- [asterisk](../telephony/asterisk.md) — open-source PBX, common partner for self-hosted phone deployments.
- [freeswitch](../telephony/freeswitch.md) — alternative open-source softswitch.
- [sip-protocol](../telephony/sip-protocol.md) — SIP basics; provisioning targets registrar via SIP.
- [sip-trunking](../telephony/sip-trunking.md) — upstream carrier connectivity.
- [tls](../security/tls.md) — TLS configuration and cert handling for provisioning + SIP/TLS.

## References

- RFC 2131 — Dynamic Host Configuration Protocol
- RFC 2132 — DHCP Options and BOOTP Vendor Extensions (Options 66, 67)
- RFC 3925 — Vendor-Identifying Vendor Class Option for DHCPv4
- RFC 4242 — Information Refresh Time Option for DHCPv6
- RFC 1350 — TFTP Revision 2
- RFC 5246 — TLS 1.2 (with later RFC 8446 for TLS 1.3)
- RFC 3261 — SIP: Session Initiation Protocol
- RFC 5626 — Managing Client-Initiated Connections in SIP (NAT keepalive)
- RFC 4474 — STIR/SHAKEN authenticated identity (E911 context)
- ANSI/TIA-1057 — LLDP-MED Specification
- IEEE 802.3af / 802.3at / 802.3bt — Power over Ethernet standards
- Kari's Law (47 U.S.C. § 623) — Direct 911 dialing
- RAY BAUM's Act § 506 — Dispatchable location for 911
- Polycom UC Software Administrator's Guide (latest, vendor docs)
- Yealink Auto-Provisioning Guide (per-model PDF, vendor docs)
- Yealink IP Phone Family Configuration Guide
- Cisco SPA Series Provisioning Guide
- Cisco IP Phone 7800/8800 Multiplatform Phone Administration Guide
- Cisco Unified Communications Manager Administration Guide (CTL/ITL)
- Grandstream Networks Device Provisioning Guide
- Grandstream GDMS User Manual
- Snom Provisioning Guide and Settings.xml Reference
- Mitel/Aastra IP Phone Administration Guide
- Audiocodes Mediant SBC and Gateway User Manual (provisioning sections)
- Avaya Aura SIP Phone 46xxsettings.txt Configuration Reference
- FCC E911 Compliance Documentation
- NENA i3 Standard for Next Generation 9-1-1
