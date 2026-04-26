# Avaya IP Phones

Avaya J100/J200 series desk phones, Vantage K-series Android units, IX Workplace softphone, and the H.323 vs SIP firmware family split — the operator-facing reference for Aura/IP Office/Cloud Office deployments and third-party PBX interop.

## Setup

Avaya's enterprise telephony portfolio splits across hardware endpoints, software clients, and back-end call-control platforms. The desk phones share a common firmware codebase but ship in two mutually exclusive flavours: H.323 (works only with Avaya Aura Communication Manager acting as gatekeeper) and SIP (works with Avaya Aura Session Manager or any third-party SIP PBX such as Asterisk, FreeSWITCH, 3CX, Cisco CUCM, Mitel MiVoice). The firmware is loaded onto the device at boot from a TFTP/HTTP server, so swapping firmware family is a provisioning task — not a hardware change.

### Endpoints

- **J100 series** — entry- to mid-range business desk phones, Avaya's current generation (replaces 96xx and 1600 series)
  - **J139** — 4-line entry phone, monochrome 2.3" display, 4 line keys, no expansion module support, gigabit pass-through optional
  - **J159** — 4 primary line keys + 4 secondary, dual color displays (primary + secondary for line keys), USB port, 802.3az EEE
  - **J169** — 12-line monochrome, soft keys, USB-A, dual gigabit, supports JBM24 button module
  - **J179** — 12-line color (4.3"), soft keys, USB-A, dual gigabit, supports JBM24, Bluetooth/wireless via dongle
  - **J189** — 12-line color (5.0"), 8 primary + 4 secondary line keys, dual gigabit, USB-A + USB-C, supports JBM24, integrated wifi/Bluetooth, the new flagship
- **J200 series** — newer generation announced 2023+, hardware refresh of J100 with USB-C, improved color displays
  - **J259** — color screen, mid-range
  - **J279** — color screen, executive class
- **Vantage K-series** — Android-based touchscreen phones, can run Avaya Workplace plus arbitrary Android apps
  - **K155** — entry-level 5" touchscreen
  - **K165** — 8" tablet-style with handset
  - **K175** — flagship 8" touchscreen, full HD camera, the executive Android device
- **DECT 3700 series** — wireless DECT handsets, paired with DECT base station for in-building roaming (3725/3735/3745/3749)
- **Conference phones** — B179/B189 wired conference phones (legacy Konftel-derived hardware)
- **IX Workplace softphone** — Avaya's unified communications client for Windows, macOS, iOS, Android. SIP only (no H.323). Same provisioning back-end as J series SIP firmware.

### Back-end Platforms

- **Avaya IP Office** — small/medium business platform (up to ~3000 users on IP Office Server Edition), supports both H.323 and SIP, simpler licensing, single-box appliance or VM
- **Avaya Aura** — enterprise platform (10K+ users), modular: Communication Manager (CM, the call control engine) + Session Manager (SM, the SIP routing) + System Manager (SMGR, the central provisioning + admin console) + Application Enablement Services (AES, CTI/API gateway)
- **Avaya Cloud Office** — UC SaaS, white-labeled RingCentral, replaces on-premises Aura/IP Office for cloud-only deployments
- **Avaya Spaces** — cloud collaboration / video meetings (separate from voice)

### H.323 vs SIP Firmware Split

Every J-series phone ships with one of two firmware families. The firmware itself is in `.tar` packages on the provisioning server.

| Aspect | H.323 firmware | SIP firmware |
|--------|----------------|--------------|
| Filename pattern | `S96x6_H323_*.tar` (legacy 96xx), `J100_H323_*.tar` | `J100_SIP_*.tar` |
| Works with | Avaya Aura CM only (CM acts as H.323 gatekeeper) | Avaya Aura SM, IP Office, third-party SIP PBX |
| Registration | H.323 RAS to gatekeeper, "Extension Number" identity | SIP REGISTER, AOR + Auth Username |
| Survivability | LSP (Local Survivable Processor) — H.323 fallback | ESS (Enterprise Survivable Server) or SIP backup proxy |
| Feature parity | Older but battle-tested; full Aura feature set | Now feature-equivalent on Aura, plus third-party interop |
| Recommended for new deployments | No (Avaya is migrating everything to SIP) | Yes |

You cannot swap "live" — switching a phone from H.323 to SIP firmware requires factory-resetting the phone, ensuring the provisioning server delivers the SIP `.tar`, and re-registering. The phone reads `46xxsettings.txt` and re-evaluates its mode at boot.

### Initial Hardware Setup

```
1. Connect ethernet to PoE switch (802.3af class 2 typical, J189 may need 802.3at)
   or use 48V external PSU
2. Daisy-chain PC through phone's PC port if desired (most J series support gigabit pass-through)
3. Phone boots, displays "Avaya" splash, attempts DHCP
4. Watches for DHCP Option 242 (or Option 176 legacy, or Option 66 TFTP)
5. Pulls 46xxsettings.txt from indicated HTTP/TFTP server
6. Pulls firmware .tar if version mismatch
7. Reboots into target firmware (H.323 or SIP)
8. Registers with configured server
```

If no DHCP option is present, the phone boots to a manual config screen: press `Mute` then `2 7 2 3 8 #` (the "27238" admin code spells AVAYA on a phone keypad) and configure manually.

## Default Access

```bash
# Web UI
http://<phone-ip>/
https://<phone-ip>/   # newer firmware

# Admin password
admin / 27238

# 27238 spelt on a phone keypad
2 = A,B,C
7 = P,Q,R,S
2 = A,B,C
3 = D,E,F
8 = T,U,V
# 27238 = AVAYA  (Avaya brand-numeric default)
```

The "27238 = AVAYA" password is hardcoded into the device-side admin menu and the web UI of every untouched Avaya phone shipped in the last decade. You **must** change it. Search engines turn up thousands of internet-exposed Avaya phones with this default still set.

```bash
# Settings menu access on the phone itself
# Press "Mute" then 27238# to enter admin menu
# Press "Mute" then 7387# (= "RESET") to factory reset (firmware stays)
# Hold "*" during boot to clear all settings (some models)

# User PIN (different from admin password)
# Used to lock/unlock user-facing settings (ringtones, brightness, etc.)
# Default user PIN: typically blank or "0000"; site-configured via 46xxsettings.txt
# PROCPSWD parameter controls user PIN
# PROCPSWD=123456   # six-digit PIN to enter user options menu

# Web UI default port
80   # HTTP, default
443  # HTTPS if cert provisioned
```

## Web UI Tour

Login → left-nav menu. The exact tabs vary slightly by firmware version (J100 H.323 R6.x vs SIP R4.x).

- **Status** — registration state per line, IP/MAC, firmware version, uptime, peer status
- **Network** — IPv4/IPv6 addressing, VLAN, LLDP-MED, DHCP option viewer, DNS, NTP
- **Audio** — codec list and per-line preference, DSCP, jitter buffer, comfort noise, SRTP mode
- **SIP** — domain, primary/backup proxy, registration period, transport (UDP/TCP/TLS), SBC mode
- **H.323** (H.323 firmware only) — gatekeeper address, extension number, password
- **Programmable Buttons** — line appearance, BLF, BLA, speed dial, feature button mapping
- **Phonebook** — local entries + LDAP/AD config
- **Logs** — component log level (SIP, audio, network), syslog server config, download log archive
- **Diagnostics** — ping, traceroute, packet capture (PCAP), audio loopback test, DHCP renew, factory reset
- **Security** — TLS version, server cert validation, 802.1X EAP-TLS/PEAP, web UI HTTPS toggle, admin password change

## SIP Account Configuration

For SIP firmware, the phone uses standard RFC 3261 REGISTER. Configuration via web UI (`SIP` tab) or via `46xxsettings.txt` parameters.

### Per-line SIP Parameters

```bash
# Web UI fields:
SIP Domain               # e.g. "voice.example.com" — appears in From header
SIP Server (Primary)     # e.g. sm.example.com or 10.1.1.20 — Session Manager / SBCE
SIP Server (Backup)      # secondary proxy for failover
SIP Server Port          # 5060 UDP/TCP, 5061 TLS
SIP Transport            # UDP / TCP / TLS — TLS recommended in production
SIP User ID              # the AOR localpart, e.g. "5301" — extension number
Auth Username            # the digest auth username (often same as User ID)
Auth Password            # the digest auth password
Display Name             # caller ID name shown to far end
Registration Period      # default 3600 seconds; phone re-REGISTERs every period
Subscribe Period         # for BLF/BLA; default 3600s
```

### 46xxsettings.txt Parameters (SIP)

```bash
# Per-line SIP parameters (line index appended _N for line 2, 3, etc.)
SET SIPDOMAIN voice.example.com
SET SIP_CONTROLLER_LIST sm-pri.example.com:5061;transport=tls,sm-sec.example.com:5061;transport=tls
SET ENABLE_PRESENCE 1
SET SIP_REG_PROXY_POLICY simultaneous
SET SIP_PORT_SECURE 5061
SET SIP_PORT 5060
SET DISCOVER_AVAYA_ENVIRONMENT 1
SET ENFORCE_SIPS_URI 1                # require sip TLS
SET REGISTERWAIT 3600                 # registration period
SET SUBSCRIBEWAIT 3600
```

### Backup Server Fallback Timing

```bash
# Failover behaviour
SET FAILBACK_POLICY auto              # auto / admin
SET RECOVERYREGISTERWAIT 60           # how often to retry primary after failover
SET FAST_RESPONSE_TIMEOUT 3           # primary response window before retry
```

### Register-Through-MBP/SBC Pattern

Mobile workers register through Avaya SBCE (Session Border Controller for Enterprise) which sits in the DMZ. The phone is configured with the SBCE's public address/FQDN as its SIP server; SBCE proxies REGISTER and INVITE to internal Aura Session Manager.

```bash
# Remote-worker phone config
SET SIPDOMAIN voice.corp.example.com
SET SIP_CONTROLLER_LIST sbce-public.example.com:5061;transport=tls
SET ENFORCE_SIPS_URI 1
SET TLSSRVRID 1                       # validate TLS server identity
SET TRUSTCERTS sbce-ca.crt            # bundle in trust store
```

## H.323 Configuration

For H.323 firmware (CM-only deployments).

```bash
# Web UI / on-phone admin menu
H.323 Aura Communication Manager Server   # CM IP/FQDN, e.g. 10.1.1.10
                                          # gatekeeper address, RAS port 1719
Extension Number                          # e.g. 5301 — NOT a SIP user ID, this is the H.323 endpoint identifier
H.323 Password                            # the registration password (set in CM SAT)

# 46xxsettings.txt (H.323)
SET MCIPADD 10.1.1.10,10.1.1.11    # primary,LSP — comma-separated list of CMs
SET MCPORT 1719                    # gatekeeper RAS port
SET TLSSRVR 10.1.1.10              # TLS gatekeeper
```

CM-side: in the SAT (System Administration Terminal), `add station 5301`, set type to `J179`, set Security Code (the password), then on the phone enter Extension `5301` and matching password.

## Codec Configuration

```bash
# Supported codecs (J series, SIP firmware R4.x+)
G.711 mu-law       # toll-quality narrowband, US/JP standard
G.711 A-law        # toll-quality narrowband, EU/world standard
G.722              # wideband, 7 kHz audio at 64 kbps
G.729a             # narrowband, 8 kbps, low-bandwidth WAN
G.726-32           # legacy ADPCM, rare
Opus               # SIP firmware 4.0.7+, wideband variable-rate, recommended for internet
iLBC               # legacy, rare in modern Avaya

# 46xxsettings.txt
SET AUDIO_CODEC_LIST G.722,G.711MU,G.711A,G.729A
SET AUDIO_CODEC_PRIORITY G722,PCMU,PCMA,G729A,OPUS

# Web UI: Audio → Codecs → drag to reorder priority list
# First codec in list is offered first in SDP; far end picks from intersection
```

Per-line codec preference is supported on multi-line phones — useful when line 1 is the corporate Aura PBX (G.722 wideband) and line 2 is a remote-worker SIP service (Opus / G.711).

## Programmable Buttons

Avaya terminology: physical line keys are **Line Appearance** buttons; soft / app keys are **Feature Buttons** or **Application Buttons**.

### Button Types

```bash
# Line / call handling
Line Appearance              # a primary line on this phone
Bridged Line Appearance      # appearance of someone else's line (BLA / SCA)
Call Pickup                  # pick up another phone in same pickup group
Group Call Pickup            # pick up any ringing phone in directed group

# Speed / contact
Speed Dial                   # one-press dial to a number
Auto-Dial                    # immediate dial (no pause)
BLF                          # Busy Lamp Field — green/red status of monitored extension

# Call control
Hold
Conference                   # build N-way conference
Transfer                     # blind/attended transfer
Drop                         # drop last party from active call
Forward                      # call forward toggle
Send All Calls (SAC)         # Avaya's DND — routes to coverage path
Voicemail                    # MWI indicator + speed-dial to mailbox

# Telephony
Group Page                   # multicast page to a paging group (intercom)
Park                         # park to a park orbit
Unpark                       # retrieve from park orbit

# Mobility
EC500                        # Extension to Cellular — toggle simul-ring on cell phone

# Other
Headset                      # toggle handset/headset/speaker
Mute
Redial
```

### Configuring Buttons via 46xxsettings.txt

```bash
# Button definitions per line (button index 1-12 typical)
SET BUTTON_LIST line=1,user_id=5301
SET BUTTON_LIST line=2,user_id=5302   # second appearance of own number for transfer
SET BUTTON_LIST bla=4,user_id=5400    # bridged appearance of 5400
SET BUTTON_LIST sd=5,number=5500,label="Help Desk"
SET BUTTON_LIST blf=6,user_id=5305,label="Boss"
SET BUTTON_LIST sac=7                 # Send All Calls toggle
SET BUTTON_LIST ec500=8               # EC500 toggle
```

### Configuring via SMGR

For Aura deployments, programmable buttons are typically configured **per user template** in System Manager → Users → Communication Profile → Endpoint Profile, then pushed to phones automatically when the user logs in. Direct phone-side button editing is disabled in the SMGR-managed default.

## Bridged Line Appearance (BLA)

Avaya's name for shared call appearance (SCA in other vendors). Multiple phones share the same SIP/H.323 line — typically a manager + assistant scenario.

```
                        ┌── 5301 (principal)
                        │
   Line 5301 (primary)──┼── 5301-bridge on assistant phone (BLA)
                        │
                        └── 5301-bridge on second assistant phone (BLA)
```

Concepts:

- **Principal** — the user the line "belongs to" (usually appears on their primary phone)
- **Bridge** — appearance of the same line on another phone
- Each bridge sees:
  - Idle (steady)
  - Ringing (slow flash + optional audible ring)
  - Active (steady on, color)
  - Held (fast flash) — anyone with bridge can pick up the held call
- Any party can answer the inbound call; first off-hook wins
- Active call shows on all bridges with its current state (so assistant can see boss is on the phone)

```bash
# 46xxsettings.txt — assistant phone configured with bridge of 5301
SET BUTTON_LIST line=1,user_id=5400          # assistant's own line
SET BUTTON_LIST bla=2,user_id=5301           # boss's line as bridge
SET BUTTON_LIST bla=3,user_id=5302           # boss's second line

# Ringing options
SET BLA_RING_TYPE silent    # bridge alerts silently (lamp only) — default for assistant
SET BLA_RING_TYPE delayed   # ring after N seconds if principal hasn't answered
SET BLA_RING_TYPE audible   # ring immediately
```

In CM SAT, BLA is provisioned by adding a button of type `brdg-appr` on station 5400 pointing at station 5301 button 1.

## Send All Calls (SAC)

Avaya's specific name for DND. Pressing the SAC button sets a status on the call server that immediately routes inbound calls to the user's coverage path (typically voicemail). Distinguished from "Forward" which routes to a specific number.

```bash
# Behaviour
- Inbound call arrives at extension 5301
- Extension has SAC active
- CM/SM evaluates coverage path
- Coverage path step 1 = "send to voicemail"
- Caller never hears the principal's phone ring

# Setting/clearing via feature access code (FAC)
*3   # SAC on (default Aura FAC)
#3   # SAC off

# Or programmable button "Send All Calls" — toggles, with on/off lamp
```

SAC vs DND distinction: DND on third-party phones often blocks inbound at the device. Avaya SAC is a server-side state — the call never reaches the phone, so the phone doesn't even briefly ring. This is critical for assistant/boss BLA setups: SAC on the principal does not stop the bridge from ringing if BLA is configured for audible ring.

## Coverage Path

Avaya's call-forwarding-on-no-answer mechanism. A coverage path is a numbered sequence of "points" (steps), each defining a destination if the previous step doesn't pick up.

```
Coverage Path 1
  Step 1: Coverage to extension 5302 (assistant), ring 4 times
  Step 2: Coverage to extension 5500 (department hunt), ring 3 times
  Step 3: Coverage to voicemail (hunt group h1)
```

Each step has criteria — e.g. step only applies if "Don't Answer" or "Busy" or "All". So one path can route differently for busy vs unanswered.

```bash
# CM SAT
add coverage-path 1
# fields:
#   Number of Rings: 4
#   Coverage Criteria: Active=n, Don't Answer=y, Busy=y, All=n
#   Point 1: 5302
#   Point 2: 5500
#   Point 3: h1   (hunt group = voicemail)

# Assign to station
change station 5301
# Coverage Path 1: 1
# Coverage Path 2: 2     (alternate, used when SAC active)
```

If a station has no coverage path and SAC is on, calls get dropped — a common misconfiguration.

## EC500 (Extension to Cellular)

Avaya's mobile twinning — when an inbound call hits the desk extension, it simul-rings on a configured cell phone. Whichever device answers wins; the other stops ringing.

```bash
# Configured per user in SMGR or CM SAT
add off-pbx-telephone station-mapping 5301
#   Application: EC500
#   Dial Prefix: 9          # outbound prefix for cell route
#   Phone Number: 19175551234
#   Trunk Selection: 1      # cellular gateway trunk group

# User-side toggle
*72   # EC500 on (FAC, default)
*73   # EC500 off

# Or programmable button "EC500" with on/off lamp
```

Considerations:

- Cellular trunk needs PSTN access → trunk group routing must be configured
- Caller ID on the cell may show the original caller, the corporate main number, or the desk extension depending on outbound trunk and CLI rules
- Hand-off: pick up on cell, then later transfer to desk by pressing a "hand-off" button on the desk phone (extends the call back without disconnect)
- Counts as a concurrent license seat in some Aura tiers

## Provisioning

Avaya phones boot, DHCP, then fetch a configuration file from a TFTP/HTTP server. The filename is **always** `46xxsettings.txt` — a legacy name from the original Avaya 4600 series. It's still the standard filename across J100, J200, K-series, and IX Workplace.

### Boot Sequence

```
1. DHCP DISCOVER → DHCP OFFER (IP, gateway, DNS, +Option 242 vendor data)
2. Phone parses Option 242: extracts MCIPADD, HTTPSRVR, L2QVLAN, etc.
3. Phone requests http(s)://<HTTPSRVR>/46xxsettings.txt
4. Phone parses settings file: SET ... lines
5. Phone requests firmware .tar if version differs (GETSET parameter check)
6. If firmware updated, phone reboots back to step 1
7. Phone registers with configured CM/SM server
8. Phone subscribes for BLF/BLA, fetches phonebook, etc.
```

### Provisioning Sources

- **Avaya Aura System Manager (SMGR)** — generates per-device 46xxsettings.txt dynamically based on user profile and endpoint template; phones request file with MAC-address suffix (e.g. `46xx-AABBCCDDEEFF.txt`) and SMGR generates user-specific config
- **IP Office Manager** — IP Office's web interface for editing phone profiles, generates settings file via internal HTTP
- **Generic HTTP/TFTP server** — for third-party PBX deployments (Asterisk, FreeSWITCH); admin maintains a static `46xxsettings.txt` plus per-MAC override files (`<MAC>.txt`)

### Per-Device Override

```bash
# Phone first requests:
http://server/46xxsettings.txt          # global settings
# Then:
http://server/<MAC>.txt                 # per-device override (MAC lowercase, no separators)

# Example:
http://server/46xxsettings.txt
http://server/aabbccddeeff.txt          # MAC AA:BB:CC:DD:EE:FF
```

Per-device file overrides any setting from the global file. Common pattern for assigning different extensions/users per phone.

## 46xxsettings.txt Format

Plain text, line-based, comments start with `##`. Parameters set with `SET <NAME> <VALUE>`. Long lines may continue with `\` at line end (some firmware versions only — quote carefully).

```bash
## comments are ##  (double hash)
## blank lines OK

## syntax
SET PARAMETER value
SET PARAMETER "value with spaces"
SET PARAMETER value1,value2,value3       # comma-separated lists

## conditional with IF (legacy)
IF $MACADDR SEQ 1234567890ab GOTO model_J189
GOTO end
# model_J189
SET PARAMETER specific_value
# end
```

Parameter naming conventions:

- `MCIPADD` — H.323 call-server addresses
- `SIPDOMAIN`, `SIP_CONTROLLER_LIST` — SIP back-end
- `HTTPSRVR` — HTTP provisioning server (parsed from DHCP Option 242 normally)
- `L2QVLAN`, `L2Q` — VLAN ID and 802.1p priority
- `PROCPSWD` — user PIN for on-phone settings menu
- `ADMIN_PSWD` — *not* settable here for security (must use SMGR or per-device cert)
- `LANG_LARGE_FONT` — large-font menu language
- `COUNTRY` — ringtone/dial-tone region
- `TIMEZONE_OFFSET` — local TZ in minutes from UTC
- `NTPSRVR` — NTP server list
- `SES_SRTP` — SRTP mode (0/1/2)
- `TRUSTCERTS` — additional CA cert files to trust

## Sample 46xxsettings.txt

A minimal but production-shaped configuration for a SIP-firmware deployment registering to Aura Session Manager:

```bash
## 46xxsettings.txt — example for Aura SIP deployment
## Tested with J179 SIP firmware 4.0.13

## Network / VLAN
SET L2QVLAN 200                          ## voice VLAN ID
SET L2Q 1                                ## enable 802.1Q tagging
SET L2QAUD 5                             ## 802.1p priority for audio (CoS 5)
SET L2QSIG 4                             ## 802.1p priority for signalling (CoS 4)
SET DSCPAUD 46                           ## DSCP EF for RTP
SET DSCPSIG 26                           ## DSCP AF31 for SIP
SET LLDP_ENABLED 2                       ## prefer LLDP over DHCP for VLAN

## NTP / TZ
SET NTPSRVR 10.1.1.5,10.1.1.6
SET DATETIMEFORMAT 1                     ## 1 = 24h "hh:mm"
SET TIMEZONE_OFFSET -300                 ## US/Eastern winter
SET DSTOFFSET 60                         ## DST shift in minutes
SET DSTSTART 2-Sun-Mar:02:00             ## DST rules (US 2007+)
SET DSTSTOP 1-Sun-Nov:02:00

## Provisioning
SET HTTPSRVR provisioning.example.com
SET TRUSTCERTS corp-ca.crt,corp-issuing-ca.crt
SET TLSSRVRID 1                          ## validate server cert CN/SAN

## SIP — Session Manager primary + secondary
SET ENABLE_AVAYA_ENVIRONMENT 1
SET SIPDOMAIN voice.example.com
SET SIP_CONTROLLER_LIST sm-pri.example.com:5061;transport=tls,sm-sec.example.com:5061;transport=tls
SET REGISTERWAIT 3600
SET SUBSCRIBEWAIT 3600
SET ENFORCE_SIPS_URI 1                   ## require TLS

## Audio / codecs
SET AUDIO_CODEC_LIST G722,PCMU,PCMA,G729A
SET ENABLE_OPUS 1
SET ENABLE_G722 1

## SRTP
SET SES_SRTP 1                           ## 0=off 1=best-effort 2=enforce

## Security / 802.1X
SET DOT1X 0                              ## 0=disabled 1=mode unicast 2=mode multicast
SET EAPID                                ## EAP-TLS identity
SET EAPMD5 0
SET LOCK_PSWD                            ## phone-side password lock disabled

## User PIN
SET PROCPSWD 27238                       ## settings-menu PIN — CHANGE IN PROD

## Coverage / EC500 — server-side, no settings here

## Phonebook
SET PHNCC 1                              ## country code US
SET PHNDPLENGTH 10                       ## standard 10-digit
SET LDAPENABLED 1
SET LDAPCERT corp-ca.crt
SET SES_LDAP_USERNAME ldap-svc@example.com
SET SES_LDAP_PASSWORD lookup-shared-secret
SET LDAPSRVR ldap.example.com
SET LDAPPORT 636

## Logging
SET SYSLOG_SRVR syslog.example.com
SET SYSLOG_LEVEL 4                       ## warn

## Display
SET LANG_LARGE_FONT 1
SET LANGUAGE Mlf_English
SET COUNTRY US
```

For a third-party PBX (Asterisk/FreeSWITCH/3CX) configuration:

```bash
## 46xxsettings.txt — example for Asterisk SIP
SET ENABLE_AVAYA_ENVIRONMENT 0          ## CRITICAL — disable Avaya-specific extensions
SET SIPDOMAIN pbx.example.com
SET SIP_CONTROLLER_LIST pbx.example.com:5060;transport=udp
SET DISCOVER_AVAYA_ENVIRONMENT 0
SET ENFORCE_SIPS_URI 0
SET ENABLE_PRESENCE 0
SET ENABLE_3PCC 0                       ## third-party call control off (no AES)
SET REGISTERWAIT 600
SET AUDIO_CODEC_LIST PCMU,PCMA,G722
SET SES_SRTP 0
```

The `ENABLE_AVAYA_ENVIRONMENT 0` is essential for third-party PBX — without it the phone enables AES (Application Enablement Services), AST (Avaya Session Travelling), and other proprietary protocols that confuse Asterisk dialplan and cause registration loops.

## Aura System Manager

SMGR — the central management console for Aura. Web app at `https://smgr.example.com`. Default admin: `admin` / `admin123` (CHANGE IMMEDIATELY).

Functions:

- **Users** — provision SIP user accounts, assign communication profiles, station templates, button maps
- **Endpoints** — register physical/virtual stations, assign phone hardware to users
- **Routing** — dial plans, regular-expression-based number transformations, adaptations (header rewrites)
- **Session Manager** — SM cluster nodes, monitoring, traffic stats
- **Communication Manager** — CM elements, station list, signalling group config
- **Inventory** — phone inventory by MAC, firmware version, registration state
- **Events** — alarm log, audit log of admin changes
- **Backup / Restore** — scheduled backups of all platform config

The "LSP/CMS/SMGR triad" in operator parlance:

- **CMS** — Call Management System, the historical reporting platform (Genesys-derived)
- **SMGR** — System Manager, modern config + provisioning
- **LSP** — Local Survivable Processor, branch-office mini-CM that takes over when WAN to main CM fails

```bash
# SMGR CLI access
ssh admin@smgr.example.com
# Sub-shell prompts
SMGR> show services           # list running components
SMGR> restart jboss           # restart web tier
SMGR> backup now              # adhoc backup
SMGR> show users              # SIP users summary

# Common files
/var/log/Avaya/mgmt/jboss/server.log     # SMGR app logs
/var/lib/Avaya/mgmt/data                 # database tablespace
```

## Aura Communication Manager

CM — the call-control engine. Originally based on Definity G3R. Manages stations, trunks, dial plan, routing, voice mail integration. Configured primarily through SAT (System Administration Terminal), a curses-style screen-based admin interface accessed via SSH.

```bash
# Access SAT
ssh init@cm.example.com
# Or web SMI: https://cm.example.com → System Maintenance Interface

# Common SAT commands
display system-parameters customer-options    # show licensed feature set
display station 5301                          # show extension 5301 config
change station 5301                           # edit station
add station 5302                              # provision new station
list configuration station                    # list all stations
status station 5301                           # current call state
list trace station 5301                       # live trace events
busyout station 5301                          # take out of service
release station 5301                          # restore service

# Dial plan
display dialplan analysis
change dialplan analysis                      # edit
display ars analysis
change ars analysis                           # AAR/ARS digit analysis

# Trunks
list trunk-group                              # all trunks
status trunk 1                                # trunk group 1 status
display trunk-group 1
change trunk-group 1
```

### ESS / LSP Survivability

Aura survivability when WAN to main CM fails:

- **ESS (Enterprise Survivable Server)** — full CM standby in a remote data center, registered phones fail over to ESS within ~60 seconds when primary CM unreachable
- **LSP (Local Survivable Processor)** — a small CM appliance at a branch office; phones at that branch fail over to LSP for local survivability (intra-branch calls keep working even with WAN down)

```bash
# CM SAT — set ESS/LSP precedence list
display ip-network-region 1
# Failover order: primary CM → ESS → LSP
# Phones get this list via Aura environment discovery and fall back when primary stops responding
```

Phones honour the failover list using the `MCIPADD` (H.323) or `SIP_CONTROLLER_LIST` (SIP) parameter. For phones that should fail over to LSP in branch B:

```bash
SET MCIPADD cm-main.example.com,cm-ess.example.com,lsp-branch-b.example.com
```

## Avaya Aura Session Manager

SM — the SIP routing engine. Replaces the legacy "G3" gatekeeper paradigm with a modern SIP proxy + registrar. Each SM instance handles SIP REGISTER, INVITE routing, SIP-trunk normalization, B2BUA functions, presence and IM signalling.

Key concepts:

- **SIP Entities** — defined in SMGR; each is a SIP node SM talks to (CM, SBCE, third-party PBX, voicemail server). Each has IP/FQDN, transport, port.
- **Entity Links** — TCP/TLS associations between SM and SIP entities; persistent connections monitored with OPTIONS pings.
- **Routing Policies** — match incoming dialed digits against regex, choose Entity, apply Adaptations.
- **Adaptations** — header rewriting rules (e.g. strip `+1`, replace `From` URI domain, normalize PAI).
- **Communication Profiles** — per-user SIP profile assigned in SMGR Users → links the user's AOR to home Session Manager and CM.

```
[Phone] ──REGISTER──> [SM] ──Entity Link──> [CM]  for call control
                       │
                       ├──Entity Link──> [Voicemail]
                       └──Entity Link──> [PSTN SBCE]
```

### ESM-via-SBC Pattern for Remote Workers

Remote workers register through the SBCE (in DMZ), which proxies to the internal SM. This is the canonical "remote worker" deployment.

```
[Remote J179 phone] ──TLS:5061── [SBCE on public IP] ──TLS:5061── [SM internal] ── [CM]
                          (SRTP across)              (SRTP)
```

Critical: the phone treats the SBCE as its SIP server. SBCE handles NAT traversal, far-end mucking with media, and topology hiding from Internet to internal network.

## Avaya SBCE

Session Border Controller for Enterprise. The DMZ box that:

- Terminates TLS / SRTP from remote phones and SIP trunks
- Hides internal Aura topology from external SIP entities
- Performs NAT traversal for media (latching, ICE, STUN-on-behalf)
- Enforces SIP-level security (anti-fraud, malformed-message blocking, rate limiting)
- Provides the public-facing FQDN remote workers use as their SIP server

```bash
# SBCE management
https://sbce.example.com/      # web UI
ssh ipcs@sbce.example.com      # CLI

# Common SBCE concepts
Application Server             # the internal SM SIP entity SBCE proxies to
SIP Server Profile             # connection params to internal SM
Routing Profile                # how to route outbound SIP based on URI/host
Server Configuration           # public-facing TLS termination interface
Topology Hiding                # rewrite Via, Contact, Record-Route, From/To
Endpoint Flow                  # match remote-worker traffic and apply policy
Server Flow                    # match SIP-trunk traffic and apply policy

# Diagnostic
tracesbc -t      # tail-style live SIP trace, filterable by URI/IP/method
tracesbc -p      # pcap capture mode
```

## DHCP Options

Avaya phones can be told their provisioning server through three DHCP options. Use Option 242 unless legacy 96xx series support is required.

### Option 242 — Vendor-Specific Concatenated TLVs

Modern preferred option. Sub-options encoded as a comma-separated key=value string.

```bash
# Common keys
MCIPADD     # H.323 call server list
HTTPSRVR    # HTTP/TFTP provisioning server
HTTPDIR     # subdirectory under HTTPSRVR
L2QVLAN     # VLAN ID
L2Q         # 802.1Q enable
L2QAUD      # 802.1p priority audio
L2QSIG      # 802.1p priority signalling
VLANTEST    # seconds to attempt tagged VLAN before falling back

# Encoding
"MCIPADD=10.1.1.10,10.1.1.11,HTTPSRVR=10.1.1.20,L2QVLAN=200,L2Q=1,L2QAUD=5,L2QSIG=4"

# ISC dhcpd config
option space avaya;
option avaya.option-242 code 242 = string;
class "avaya-phones" {
  match if substring(option vendor-class-identifier, 0, 4) = "ccp.";
  option avaya.option-242 "MCIPADD=10.1.1.10,HTTPSRVR=10.1.1.20,L2QVLAN=200,L2Q=1";
}
```

### Option 176 — Legacy

The original Avaya DHCP option, used by 4600/9600 series with H.323 firmware. Same TLV format as Option 242 but on a different DHCP code. Modern J series support both for backward-compat.

```bash
option avaya.option-176 code 176 = string;
option avaya.option-176 "MCIPADD=10.1.1.10,HTTPSRVR=10.1.1.20";
```

### Option 66 — TFTP Server

Standard DHCP option for TFTP server name (RFC 2132). Avaya phones honour it as a last-resort if Options 242/176 are absent. Less flexible — only specifies the server, no other params.

```bash
option tftp-server-name "10.1.1.20";
```

### Option 67 — Boot File

```bash
option bootfile-name "46xxsettings.txt";
```

## NAT

Avaya phones support multiple NAT-traversal strategies depending on deployment:

- **STUN** — phone discovers its public IP and uses it in SIP Contact / SDP. Works behind one layer of NAT, fails with symmetric NAT or carrier-grade NAT.
- **SBCE remote-worker** — far-end NAT traversal, the canonical Aura solution. Phone connects outbound TLS to SBCE, SBCE handles all NAT.
- **TURN** — relay-based, used by IX Workplace softphone for media when SBCE not in path.
- **ICE** — for IX Workplace softphone in WebRTC mode (modern variants).

```bash
# SIP firmware STUN config
SET STUN_ENABLE 1
SET STUN_SERVER stun.example.com:3478
SET STUN_USERNAME stunuser
SET STUN_PASSWORD shared-secret

# Symmetric RTP recommended even without STUN
SET ENABLE_SYMMETRIC_RTP 1
```

## SIP-over-TLS

```bash
# Web UI: Network → Security → Server Certificates → Add CA Cert
# Or 46xxsettings.txt:
SET TRUSTCERTS corp-ca.crt,corp-issuing-ca.crt
SET TLSSRVRID 1                          ## verify server identity (CN or SAN match)
SET TLSPORT 5061
SET ENFORCE_SIPS_URI 1                   ## require sips: scheme
SET SIP_CONTROLLER_LIST sm.example.com:5061;transport=tls
```

Cert pinning option (newer firmware):

```bash
SET PINNING_ENABLE 1
SET PINNING_FINGERPRINT sha256/<base64-sha256-of-cert>
```

If `TLSSRVRID 1`, the phone validates the server's TLS cert CN/SAN against `SIP_CONTROLLER_LIST` value. Common gotcha: SAN must include both FQDN and IP if phones are configured by IP.

## SRTP

SES_SRTP modes — the most-used SRTP control parameter.

```bash
SET SES_SRTP 0    # disabled — RTP only, no encryption
SET SES_SRTP 1    # best-effort — offer SRTP, fall back to RTP if peer doesn't support
SET SES_SRTP 2    # enforce — require SRTP, hang up call if peer can't support
```

Crypto suites supported (firmware-dependent):

```
AES_CM_128_HMAC_SHA1_80   # default suite
AES_CM_128_HMAC_SHA1_32
AES_CM_256_HMAC_SHA1_80   # newer firmware
AES_GCM_128
```

```bash
SET SRTPCRYPTOSUITE_LIST AES_CM_128_HMAC_SHA1_80,AES_CM_256_HMAC_SHA1_80
```

For mixed third-party PBX deployments, use `SES_SRTP 1` (best-effort). Setting 2 against a PBX that doesn't support SRTP will cause every call to fail with no audio.

## PoE / VLAN

Avaya phones use LLDP-MED for VLAN/QoS discovery, with DHCP Option 242 as fallback. The L2Q parameter family controls 802.1Q tagging.

```bash
SET LLDP_ENABLED 2          # 0=off, 1=enabled, 2=prefer LLDP over DHCP
SET L2QVLAN 200             # voice VLAN ID
SET L2Q 1                   # 1=enable tagging, 0=disable, 2=auto
SET L2QAUD 5                # 802.1p priority for audio (CoS 5 = expedited forwarding)
SET L2QSIG 4                # 802.1p priority for signalling (CoS 4)
SET PHY1STAT 1              # PHY1 (network port) status: 1=auto-neg
SET PHY2STAT 1              # PHY2 (PC port) status: 1=auto-neg
SET PHY2VLAN 0              # PC port VLAN (0=untagged data)
SET PHY2PRIO 0              # PC port 802.1p priority
```

Switch-side:

```cisco
! Cisco IOS access port for Avaya phone
interface GigabitEthernet1/0/10
 switchport mode access
 switchport access vlan 100               ! data VLAN (PC traffic, untagged)
 switchport voice vlan 200                ! voice VLAN (phone traffic, tagged)
 spanning-tree portfast
 lldp transmit
 lldp receive
 power inline auto
```

Avaya ERS / Extreme switches:

```text
config ports 1/10
  vlan add 100 untagged
  vlan add 200 tagged
  qos 802.1p ingress-tag-priority 5
```

## Audio Quality

```bash
# DSCP marking
SET DSCPAUD 46              # EF (Expedited Forwarding) for RTP — typical
SET DSCPSIG 26              # AF31 (Assured Forwarding) for SIP/H.323

# Jitter buffer
SET AUDIO_JB_NOM 60         # nominal jitter buffer in ms
SET AUDIO_JB_MAX 240        # max in ms
SET AUDIO_JB_MIN 10         # min in ms
SET AUDIO_JB_TYPE 0         # 0=adaptive, 1=fixed

# Comfort noise / VAD
SET ENABLE_COMFORT_NOISE 1
SET ENABLE_VAD 1            # voice activity detection
SET ENABLE_AEC 1            # acoustic echo cancellation

# Codec specifics
SET ENABLE_G722 1           # wideband 64kbps; CCITT/ITU-T spec
SET ENABLE_OPUS 1           # adaptive 6-510 kbps; RFC 6716
```

G.722 wideband is supported on all J100 series and is preferred for internal calls. It reverts to G.711 narrowband when traversing PSTN.

## Phonebook

Local entries plus LDAP / Active Directory lookup.

```bash
# Local phonebook
SET PHNDPLENGTH 10                    # default phonebook dial-plan digit length
SET LOCALDIAL 1                       # local number expansion

# LDAP / AD
SET LDAPENABLED 1
SET LDAPSRVR dc.example.com
SET LDAPPORT 636                      # 389 plain, 636 TLS
SET LDAPCERT corp-ca.crt
SET LDAPSEARCHBASE "OU=Users,DC=example,DC=com"
SET SES_LDAP_USERNAME ldap-svc@example.com
SET SES_LDAP_PASSWORD lookup-shared-secret
SET LDAPGRPATTR memberOf
SET LDAPNUMATTR telephoneNumber
SET LDAPNAMEATTR displayName
SET LDAPMAILATTR mail
```

LDAP queries are issued live when the user types into the phonebook search box. Results show name + matched number; pressing dial sends INVITE.

## Microsoft Teams Compatibility

Avaya does **not** ship Teams firmware on its phones (unlike Yealink/Poly which have officially-certified Teams variants). Teams interop is via:

1. **Direct Routing through SBCE** — SBCE configured as a Direct Routing SBC, Teams routes to SBCE via Microsoft 365, SBCE bridges to Aura.
2. **Operator Connect** — Microsoft-certified carrier integration; some Avaya cloud offerings support this.
3. **Avaya IX Workplace + Teams alongside** — separate apps, no shared call control; presence federation possible via SIP/SIMPLE.

```
[Teams user] ──Teams cloud──> [Direct Routing SBC = SBCE] ──> [SM] ──> [CM]
                                                                       │
                                                                       └── [J179 phones]
```

Direct Routing tariff uses TLS:5061 inbound from Microsoft to SBCE public IP, Microsoft requires SAN certificate matching SBCE FQDN, signed by trusted public CA, and certain SIP-OPTIONS keepalive behaviour.

## Vantage K-Series Specifics

K155/K165/K175 are Android phones (Android Open Source variants), not running the J-series firmware. They run **Avaya Workplace** (the Android UC client) plus, on K175, additional admin-approved Android apps.

Differences from J series:

- Touchscreen primary input; physical keypad on K155/K165 only
- Provisioned via SMGR but uses the Avaya Workplace app for SIP, not the native phone OS
- Camera (K175) for video calls
- Additional apps: web browser, custom HTML5 apps via Vantage Web Apps SDK
- 46xxsettings.txt is consumed by the Workplace app, not the underlying Android firmware
- Firmware updates: Android OTA + Workplace app updates separately

```bash
# K-series-specific 46xxsettings.txt
SET ENABLE_VANTAGE_WEB_APPS 1
SET VANTAGE_HOMEPAGE_URL https://intranet.example.com/vantage/home
SET VANTAGE_BROWSER_ENABLE 1
```

## IX Workplace Softphone

Avaya's softphone (renamed from Avaya Communicator over the years; rebranded again as Avaya Workplace Client). Available for:

- Windows 10/11
- macOS 11+
- iOS 14+ (App Store)
- Android 8+ (Play Store / direct APK)

SIP only — no H.323. Provisioned by:

- **SMGR/SM** — recommended; per-user profile pushed
- **Manual config** — user enters SIP URI, password, server FQDN
- **Auto-discovery** — DNS SRV `_sip._tls.<domain>` plus username

```bash
# Manual config equivalent (settings file pulled by Workplace at first login)
Server URL:  https://smgr.example.com:443
Username:    user@example.com
Password:    <SIP digest password>
```

Workplace supports:

- Voice + video calls (H.264/VP8 video)
- Screen sharing during calls
- Avaya Spaces meeting integration
- Presence (with corporate AD/Aura presence server)
- IM (Avaya Multimedia Messaging)
- Voicemail visual

## Logging

```bash
# Web UI: Logs → set per-component levels
sip                    # SIP signalling
audio                  # codec / RTP
network                # IP, DHCP, LLDP
ui                     # UI events
provisioning           # 46xxsettings parsing

# Levels
0 = emergency
1 = alert
2 = critical
3 = error
4 = warning   (typical default)
5 = notice
6 = info      (verbose)
7 = debug     (very verbose)

# Syslog forwarding
SET SYSLOG_SRVR syslog.example.com
SET SYSLOG_PORT 514
SET SYSLOG_LEVEL 4
```

Download captured logs as a tar archive from Diagnostics → Download Log Bundle.

## PCAP

Built-in capture from web UI:

```
Diagnostics → Capture
  Interface: PHY1 (network) / PHY2 (PC port) / both
  Filter: BPF expression (e.g. "host 10.1.1.20 or port 5060")
  Duration: max 5 minutes typically
  Buffer size: ~10 MB on most J series

Start → wait → Stop → Download .pcap
```

Open in Wireshark — captures RTP, SIP, DHCP, LLDP, ARP. Useful for diagnosing one-way audio (RTP arriving but SIP signalling broken or vice versa).

## License Tier Considerations

Avaya licensing is famously complex and a frequent cause of "phone rings but call drops" or "phone won't fully register" surprises.

- **Per-device CAL** — Client Access License per registered endpoint
- **Per-user CAL** — License per named user (covers all that user's devices)
- **Feature add-on** — extras like EC500, video, IX Workplace, presence
- **Standard / Power / Suite** tiers — bundled features at increasing price points
- **License starvation** — if license server runs out of seats, new registrations succeed but with restricted feature set; users see "Limited functionality" or calls fail at INVITE

```bash
# Aura License Server
ssh admin@wlm.example.com
WLM> show licenses                    # current licensed counts
WLM> show inuse                       # current consumption
WLM> show alarms                      # license alarms (e.g. "30-day grace expiring")
```

Common license-related surprises:

- Phone registers but EC500 button does nothing → EC500 license pool exhausted
- Outbound to PSTN fails 30 days after deployment → license server certificate expired
- IX Workplace login fails with "License unavailable" → softphone tier exhausted

## Common Errors

Verbatim error messages and fixes.

### "Registration failed (401)"

```
SIP REGISTER returned 401 Unauthorized. The phone's Auth Username or Password is wrong.

Fix:
- Verify SIP User ID, Auth Username, Auth Password in web UI
- On Aura: confirm the user's Communication Profile Password in SMGR matches the phone's configured Auth Password
- Restart phone after correcting
```

### "Login failed: incorrect password"

```
Web UI admin login failed.

Fix:
- Default admin password is 27238 (CHANGE if not already changed)
- If forgotten, factory-reset by holding * during boot (some models) or
  press Mute then 73738# (RESET) on phone keypad
```

### "Cannot connect to call server"

```
Phone reaches network but can't reach SIP/H.323 server.

Fix:
- Ping server from phone (Diagnostics → Ping)
- Verify SIP_CONTROLLER_LIST or MCIPADD value
- Verify firewall permits SIP (5060/UDP/TCP, 5061/TCP for TLS) and RTP UDP 16384-32767
- Check SIP transport (UDP/TCP/TLS) matches server listen port
```

### "License unavailable"

```
Aura license pool exhausted or license server unreachable.

Fix:
- Check WLM (Web License Manager) UI for current usage / available
- If exhausted: increase license seats or de-provision unused users
- If unreachable: check network connectivity to WLM, certificate validity (license server cert often expires)
```

### "Network unreachable"

```
DHCP failed, no IP assigned, or default gateway unreachable.

Fix:
- Cable check, switch port LED, PoE (does phone power on at all?)
- Verify VLAN: if phone is in voice VLAN but DHCP is on data VLAN, re-tag the port
- Check L2Q parameters in 46xxsettings.txt
- Static-IP fallback: Mute → 27238# → Network → IP Configuration → set static
```

### "TLS Connection failed"

```
SIP-over-TLS handshake failed; certificate or trust issue.

Fix:
- Check the server cert's SAN includes the FQDN/IP in SIP_CONTROLLER_LIST
- Add the server's CA to TRUSTCERTS in 46xxsettings.txt
- Disable strict identity check temporarily: SET TLSSRVRID 0 (debug only)
- Verify NTP — TLS requires accurate time; phone with wrong clock rejects valid certs
- Check firewall allows TCP 5061 in both directions
```

### "Provisioning: 404 Not Found for 46xxsettings.txt"

```
Phone cannot fetch the settings file from HTTPSRVR.

Fix:
- Verify HTTPSRVR value (in DHCP Option 242 or static config)
- HTTP server document root must contain 46xxsettings.txt at root or in HTTPDIR path
- Permissions: file readable by webserver user
- HTTPS: ensure cert is trusted (or use HTTP for dev)
- Test: curl -v http://<HTTPSRVR>/<HTTPDIR>/46xxsettings.txt from same network
```

### "DHCP Option 242 not present"

```
DHCP server isn't sending Option 242, phone falls back to manual config.

Fix:
- Configure DHCP server with vendor-specific Option 242
- Verify phone's vendor-class-identifier matches your dhcpd class match
- Some DHCP servers default to Option 176 (legacy) — change to 242
- Test: tcpdump on DHCP server, check OFFER packet contains option 242
```

### "Coverage path not configured"

```
SAC button pressed but no coverage path defined → calls drop.

Fix:
- CM SAT: change station 5301 → set Coverage Path 1 / Coverage Path 2 fields
- Verify coverage path itself exists: display coverage-path 1
- Test by calling extension with SAC on
```

### "EC500 unavailable"

```
EC500 button pressed but feature disabled or unlicensed.

Fix:
- CM SAT: display off-pbx-telephone station-mapping 5301 → confirm row exists
- Verify cellular trunk group reachable: status trunk N
- Check license: WLM EC500 pool not exhausted
- Confirm dial prefix and outbound CLI rules
```

## Common Gotchas

Broken → fixed pairs.

### H.323 firmware on phone but PBX expects SIP (or vice versa)

```
Broken:
  Phone boots, attempts H.323 RAS, gets nothing — "Cannot connect to call server"
  PBX is Asterisk (SIP only)

Fixed:
  - Replace firmware: in 46xxsettings.txt set the SIP firmware tar:
      SET FW_FILE_NAME J100_SIP_R4_0_13_0_0.tar
      SET FW_FILE_NAME_BACKUP J100_SIP_R4_0_13_0_0.tar
  - Or place SIP firmware tar in TFTP/HTTP root with the right name
  - Reboot phone; it will pull SIP firmware and switch family
  - First boot: clear settings (factory reset menu) so legacy H.323 params don't confuse SIP boot
```

### Default admin password 27238 not changed

```
Broken:
  Anyone on the network (or worse, internet) can browse to phone and reconfigure
  Common in tens of thousands of internet-exposed Avaya phones

Fixed:
  - Web UI: Security → Change Admin Password — set strong password
  - Or in 46xxsettings.txt push site-wide via SMGR (admin password is hashed)
  - Block phone web UI from internet — never expose 80/443 from a phone WAN-side
  - Track via inventory: every phone audited for non-default admin password
```

### DHCP Option 242 syntax wrong (TLV format)

```
Broken:
  Option 242 string formatted wrong (e.g. semicolons instead of commas)
  Phone parses partial values, skips HTTPSRVR, falls back to manual config

Fixed:
  - Strict format: comma-separated KEY=VALUE pairs
    "MCIPADD=10.1.1.10,HTTPSRVR=10.1.1.20,L2QVLAN=200,L2Q=1"
  - No spaces around =
  - Quotes around full string in dhcpd.conf
  - Verify with tcpdump: tcpdump -nv -i any port 67 or port 68 -X
```

### LSP not configured → Survivability fails when CM goes down

```
Broken:
  Branch office WAN to main CM drops; phones go dead — no internal calls work
  No LSP configured at branch

Fixed:
  - Deploy LSP appliance (or VM) at branch
  - In SMGR/CM: register LSP as ESS, assign to relevant ip-network-region
  - In 46xxsettings.txt:
      SET MCIPADD cm-main.example.com,lsp-branch-b.example.com
  - Test failover: simulate WAN drop, verify phones re-register to LSP within ~60s
```

### SRTP enforced but third-party PBX doesn't support

```
Broken:
  SET SES_SRTP 2 (enforce)
  PBX is Asterisk with chan_pjsip default media_encryption=none
  Every call fails with no audio or 488 Not Acceptable

Fixed:
  - Either: set Asterisk dialplan media_encryption=sdes for the relevant endpoint
  - Or: change phones to SET SES_SRTP 1 (best-effort) so they fall back to plain RTP
  - Generally SES_SRTP 1 for mixed deployments unless full TLS+SRTP enforced site-wide
```

### License not assigned in SMGR → phone won't fully register

```
Broken:
  Phone registers (200 OK on REGISTER) but immediately deregisters
  Or: phone shows "Limited service" / "License unavailable"

Fixed:
  - SMGR → Users → user → Communication Profile → Endpoint Profile → assign template
  - Verify license tier covers the chosen profile (e.g. Power Suite for J189)
  - WLM → check available seats; increase if exhausted
```

### Coverage path missing → calls go unanswered with no fallback

```
Broken:
  Inbound call rings extension; nobody answers; caller hears beeping forever, then hangs up
  No coverage path → no voicemail fallback

Fixed:
  - CM SAT: change station 5301 → Coverage Path 1: 1
  - Verify coverage-path 1 routes to voicemail hunt group h1 in step 3
  - If SAC active, ensure Coverage Path 2 also set (separate path used during SAC)
```

### Wrong country/language code → menu in wrong language

```
Broken:
  Phone menu in Spanish, dial tone is Italian — confused users
  COUNTRY parameter wrong

Fixed:
  - 46xxsettings.txt: SET COUNTRY US (or DE, FR, GB, etc.)
  - SET LANGUAGE Mlf_English (or Mlf_French, Mlf_German, etc.)
  - SET LANG_LARGE_FONT 1 if elderly users need bigger text
  - Reboot phone to apply
```

### VLAN tag set on phone but switch port untagged

```
Broken:
  SET L2QVLAN 200, SET L2Q 1
  Switch port: switchport access vlan 100 (no voice vlan defined)
  Phone tags packets VLAN 200; switch drops them; phone goes offline

Fixed:
  - Switch port: add voice vlan
      switchport voice vlan 200
  - Or remove tag from phone:
      SET L2Q 0
      SET L2QVLAN 0
  - Confirm with LLDP-MED: lldp transmit/receive on switch
```

### Aura Session Manager PSTN gateway dial-rules wrong → outbound fails

```
Broken:
  User dials 9-1-555-1234, Aura returns 404 Not Found
  Routing Policy regex doesn't match dialed digits

Fixed:
  - SMGR → Routing → Dial Patterns: confirm pattern "9XXXXXXXXXX" matches
  - Routing Policy: bind dial pattern to PSTN SIP Entity
  - Adaptation: strip leading "9" before sending to ITSP (most ITSPs want bare E.164)
  - Use SIP trace (tracesbc) to see exact request sent and ITSP response
```

### SBCE certificate not in trust store → remote-worker fails

```
Broken:
  Remote J179 fails with "TLS Connection failed"
  SBCE uses public CA cert; phone doesn't have public CA in TRUSTCERTS

Fixed:
  - Add public CA chain to TRUSTCERTS in 46xxsettings.txt:
      SET TRUSTCERTS DigiCertGlobalRoot.crt,DigiCertSHA2.crt
  - Or use SET TLSSRVRID 0 temporarily to disable validation (debug only)
  - Verify cert chain: openssl s_client -connect sbce.example.com:5061 -showcerts
```

### 46xxsettings.txt parsing error (line continuation, quoting)

```
Broken:
  Long SIP_CONTROLLER_LIST broken across two lines with backslash, but the firmware
  doesn't honour line-continuation; second line is parsed as new SET command, errors
  out, half the settings missing

Fixed:
  - Keep SET lines on a single physical line, even if very long
  - Quote values with spaces: SET FOO "value with spaces"
  - Avoid trailing whitespace and CRLF (use Unix LF only)
  - Validate with: avaya-46xx-validator (community tool) or test on a single phone first
```

## Diagnostic Tools

```bash
# Phone web UI
Diagnostics → Ping
Diagnostics → Traceroute
Diagnostics → Capture (PCAP download)
Diagnostics → Audio Loopback
Diagnostics → DHCP Renew
Diagnostics → Reset Network Stack

# Aura Trace Viewer (CM)
ssh init@cm.example.com
> list trace station 5301
> list trace tac 1
> list trace ras 5301              # H.323 RAS trace
> list trace sigsm 5301            # SIP signalling trace via SM

# SMGR System Manager Audit Log
SMGR → System Manager → Audit Log → filter by user/element/timeframe

# SBCE tracesbc
ssh ipcs@sbce.example.com
$ tracesbc -t                       # interactive live trace
$ tracesbc -uri sip:5301@           # filter by URI
$ tracesbc -m INVITE                # only INVITE methods
$ tracesbc -p -o /tmp/trace.pcap    # pcap output

# CM trace summarizer
$ traceroute -i sip 5301            # CM-side trace tool
```

## Sample Cookbook

### Asterisk + Avaya J series via SIP firmware

```ini
; pjsip.conf — Asterisk endpoint for Avaya J179
[5301]
type=endpoint
context=internal
disallow=all
allow=g722
allow=ulaw
allow=alaw
auth=5301-auth
aors=5301
direct_media=no
rtp_symmetric=yes
force_rport=yes
rewrite_contact=yes
media_encryption=no                 ; or sdes if SES_SRTP=1 on phone

[5301-auth]
type=auth
auth_type=userpass
username=5301
password=Sup3rS3cret!

[5301]
type=aor
max_contacts=1
qualify_frequency=30
```

```bash
## 46xxsettings.txt for Asterisk
SET ENABLE_AVAYA_ENVIRONMENT 0
SET DISCOVER_AVAYA_ENVIRONMENT 0
SET SIPDOMAIN pbx.example.com
SET SIP_CONTROLLER_LIST pbx.example.com:5060;transport=udp
SET REGISTERWAIT 600
SET ENABLE_PRESENCE 0
SET ENABLE_3PCC 0
SET AUDIO_CODEC_LIST G722,PCMU,PCMA
SET SES_SRTP 0
```

Phone-side: enter SIP User ID `5301`, Auth Username `5301`, Auth Password `Sup3rS3cret!`, Domain `pbx.example.com`.

### FreeSWITCH + Avaya

```xml
<!-- directory/default/5301.xml -->
<include>
  <user id="5301">
    <params>
      <param name="password" value="Sup3rS3cret!"/>
      <param name="vm-password" value="5301"/>
    </params>
    <variables>
      <variable name="user_context" value="default"/>
      <variable name="effective_caller_id_name" value="Alice"/>
      <variable name="effective_caller_id_number" value="5301"/>
    </variables>
  </user>
</include>
```

Phone configured the same way as Asterisk example, just with FreeSWITCH IP/FQDN as SIP server.

### IP Office Manager Front-End

For small/medium deployments, IP Office Manager replaces SMGR. The IP Office appliance runs both call control and provisioning. Phones boot, DHCP delivers Option 242 with HTTPSRVR=IP-Office-IP, IP Office Manager generates the per-MAC settings file based on the user-extension assignment in its database.

```
[IP Office (Server Edition or IP500v2)]
   ├── DHCP server (built-in) → Option 242
   ├── HTTP provisioning → 46xxsettings.txt + per-MAC files
   ├── SIP registrar (port 5060/5061)
   └── Phone admin in IP Office Manager (Windows GUI)
```

## Hardware Specifics

| Model | Display | Lines | Gigabit | USB | Bluetooth | Expansion | PoE Class |
|-------|---------|-------|---------|-----|-----------|-----------|-----------|
| J139 | 2.3" mono | 4 | optional | no | no | no | 802.3af class 1 |
| J159 | 2.8" color (dual) | 8 | yes | no | no | no | 802.3af class 1 |
| J169 | 3.5" mono | 12 | yes | USB-A | dongle | JBM24 (up to 3) | 802.3af class 2 |
| J179 | 4.3" color | 12 | yes | USB-A | dongle/built-in | JBM24 (up to 3) | 802.3af class 2 |
| J189 | 5.0" color (dual) | 12 (8+4) | yes | USB-A + USB-C | built-in | JBM24 (up to 3) | 802.3at class 4 |
| J259 | 4.3" color | 12 | yes | USB-C | built-in | JBM24 | 802.3at |
| J279 | 5.0" color | 12 | yes | USB-C | built-in | JBM24 | 802.3at |
| K155 | 5" touch | virtual | yes | USB-C | built-in | no | 802.3at |
| K165 | 8" touch | virtual | yes | USB-C | built-in | no | 802.3at |
| K175 | 8" touch + camera | virtual | yes | USB-C | built-in | no | 802.3at class 4 |

JBM24 expansion module: 24 additional programmable buttons with monochrome label display. Daisy-chains via USB; up to 3 modules per phone for 72 extra buttons.

## Migration Patterns

### H.323 → SIP firmware swap

Avaya is migrating customers from H.323 to SIP. Process:

```
1. Inventory all phones currently on H.323 firmware
2. Verify Aura Session Manager is configured and capacity-sized
3. In SMGR: create user Communication Profiles for all migrating users
4. Stage SIP firmware .tar files on provisioning server
5. Update 46xxsettings.txt: change FW_FILE_NAME from H323 to SIP variant
6. Reboot phones in waves (overnight maintenance windows)
7. Each phone pulls SIP firmware, switches family, registers via SM instead of CM RAS
8. Validate dial-tone, BLA buttons, voicemail integration
9. Decommission CM gatekeeper RAS function once empty
```

### Legacy 96xx series → modern J series

```
Legacy 9608/9611/9620/9650/9670 → J169/J179/J189
- Same physical wiring (Cat5e, PoE)
- Configuration migration via SMGR template
- 46xxsettings.txt mostly compatible (J series adds new params)
- Button maps need re-verification (button counts differ by model)
- Plug new phone into wall jack, it pulls J-series firmware on first boot
```

### Legacy 1100/1200 series Nortel-acquired lineage

```
Avaya inherited the 1100/1200 series from Nortel Networks acquisition (2009).
These run UNIStim or SIP firmware against:
  - Nortel CS1000 / Avaya CS1K (legacy gatekeeper)
  - Avaya CS2100
  - Avaya Aura with adaptation

Migration target: replace 1100/1200 with J series + Aura SIP.
- 46xxsettings.txt does NOT apply to 1100/1200 (different format)
- Network access cards still on Cat5e PoE — wiring unchanged
- Buttons may not map 1:1; review user feedback after cutover
```

## Idioms

- "Always change default 27238 password" — first task on any new phone deployment.
- "DHCP Option 242 is critical" — without it, phones can't auto-provision.
- "Use SMGR not direct file editing" — for Aura, the central console keeps phones consistent.
- "LSP for survivability" — branch sites without LSP go dark when WAN drops.
- "SBCE for remote workers" — never expose Aura SM directly to internet.
- "License tier matters" — feature you tested in lab might not be licensed in prod.
- "ENABLE_AVAYA_ENVIRONMENT 0 for third-party PBX" — required when registering to Asterisk/FreeSWITCH/3CX/Mitel.
- "G.722 internally, G.711 to PSTN" — wideband on-net, narrowband through carriers.
- "SES_SRTP 1 for mixed deployments" — best-effort lets phones interop with non-SRTP peers.
- "Test the firmware swap on one phone first" — H.323→SIP can surface unexpected SBC/dial-plan issues.
- "Keep 46xxsettings.txt under source control" — it's the source of truth for site policy.

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
- audiocodes-phones

## References

- support.avaya.com — Avaya Support, per product family (J100, J200, K-series, IX Workplace)
- Avaya 46xxsettings.txt File Format Reference — full parameter list, downloaded from support site per firmware release
- Avaya Aura System Manager Administration Guide — current revision per release (10.x as of 2024)
- Avaya Aura Communication Manager Feature Description and Implementation — feature reference for CM
- Avaya Aura Session Manager Overview and Specification — SM architecture
- Avaya Session Border Controller for Enterprise Administration Guide — SBCE config
- Avaya Aura Web License Manager — licensing administration
- RFC 3261 — SIP: Session Initiation Protocol
- RFC 3711 — SRTP
- RFC 6716 — Opus Audio Codec
- ITU-T G.722 — 7 kHz audio coding within 64 kbit/s
- IEEE 802.1Q — VLAN tagging
- IEEE 802.3af / 802.3at — Power over Ethernet
- ANSI/TIA-1057 (LLDP-MED) — Link Layer Discovery Protocol Media Endpoint Discovery
