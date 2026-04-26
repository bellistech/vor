# Cisco IP Phones — CUCM, CUBE, SPA, Webex Reference

Cisco's IP phone portfolio: SCCP/SIP firmware, CUCM/CUBE provisioning, SPA flat-profile XML, 88xx TFTP+SEPMAC.cnf.xml, ITL/CTL security, SIP-OAuth, Webex Calling cloud, RoomOS endpoints.

## Setup — Cisco Portfolio

Cisco's IP phone catalog is enormous, fragmented across acquisitions, generations, and protocols. Identify which family before troubleshooting — provisioning, web UI, firmware naming, and protocol all differ.

### SPA (Linksys lineage) 5xx/3xx/AT — Small Business

```
SPA-112    — 2-port ATA (FXS only), small business analog->IP
SPA-122    — 2-port ATA + router, FXS gateway
SPA-2102   — 2-port ATA, the classic Linksys analog-IP gateway
SPA-3102   — 1 FXS + 1 FXO, hybrid PSTN bridge
SPA-303    — 3-line entry IP phone, monochrome display
SPA-501G   — 8-line monochrome, no display backlight
SPA-502G   — 1-line, basic display
SPA-504G   — 4-line, monochrome backlit
SPA-508G   — 8-line, monochrome backlit
SPA-509G   — 12-line, monochrome backlit
SPA-512G   — 1-line, monochrome (lower-end of "G" series)
SPA-514G   — 4-line, monochrome (gigabit PoE)
SPA-525G2  — 5-line color, 802.11g WiFi, Bluetooth, PoE
SPA-921    — 1-line legacy (predecessor of 50x)
SPA-922    — 1-line + sidecar support
SPA-941    — 4-line legacy
SPA-942    — 4-line legacy + sidecar
SPA-962    — 6-line color legacy
```

```
Inheritance: Linksys (acquired by Cisco 2003) -> Cisco SPA -> Cisco Small Business
Key trait: SIP-only (no SCCP), flat-profile XML config, multi-platform certified
                     (Asterisk, FreeSWITCH, BroadSoft, Metaswitch, etc.)
Web UI: http://phone-ip with admin/<blank> by default
```

### 7800 / 8800 Series — Modern Enterprise

```
7811       — 1-line entry, monochrome
7821       — 2-line, monochrome
7841       — 4-line, monochrome
7861       — 16-line, monochrome
8811       — 1-line, monochrome (modern)
8841       — 4-line, color
8845       — 4-line, color, integrated camera (video)
8851       — 5-line, color, USB
8851NR     — non-radio variant for sensitive environments
8861       — 5-line, color, WiFi+Bluetooth
8865       — 5-line, color, video, WiFi+Bluetooth
8865NR     — non-radio video variant
```

```
Generation: 78xx/88xx, the modern enterprise series (since ~2014)
Replaces: 79xx legacy line
Default protocol: SIP (firmware 10.x+ ships SIP-default)
Display: color OLED on 88xx, monochrome on 78xx
Provisioning: TFTP-based with SEP<MAC>.cnf.xml from CUCM
Encryption: TLS 1.2 + SRTP, supports SIP-OAuth (CUCM 12.5+)
PoE class: Class 3 (typical 7-12 watts)
```

### 7900 Series — Legacy SCCP and SIP Firmware

```
7902G      — 1-line entry, no display
7905G      — 1-line, monochrome
7906G      — 1-line, monochrome, 2 buttons
7910        — 1-line, monochrome, very early
7911G      — 1-line, monochrome, gigabit
7912G      — 1-line, monochrome, classic
7920       — Wireless 802.11b WiFi phone
7921G      — Wireless 802.11g WiFi phone (replacement for 7920)
7925G      — Wireless WiFi/Bluetooth (replaces 7921)
7931G      — 24-button compact desk
7935       — Conference station (older)
7936       — Conference station "Polycom-style" tri-mic
7937G      — Conference station with color display
7940G      — 2-line, monochrome (default workhorse)
7941G      — 2-line, monochrome (Gen2)
7942G      — 2-line, monochrome, gigabit
7945G      — 2-line, color
7960G      — 6-line, monochrome
7961G      — 6-line, monochrome (Gen2)
7962G      — 6-line, monochrome, gigabit
7965G      — 6-line, color
7970G      — 8-line, color, touchscreen
7971G-GE   — 8-line, color, gigabit
7975G      — 8-line, color, gigabit, USB
7985G      — Personal video phone (legacy)
```

```
Generation: 7900 series, legacy enterprise (early 2000s through ~2014)
Default protocol: SCCP (Skinny Client Control Protocol) — Cisco proprietary
Optional: SIP firmware can be loaded ("set sip" workflow via TFTP)
Provisioning: CUCM (CallManager) via SCCP, or any SIP server via SIP firmware
Display: monochrome on most, color on 7945/7965/7970/7971/7975
Note: end-of-life, but still common in production
```

### DX 80 / 650 — Video Desk Units

```
DX-80      — 23" video desk endpoint, integrated HD camera
DX-650     — 7" Android tablet phone, video, color touchscreen
```

```
Distinct from 88xx: runs Android (Cisco-customized)
Provisioning: CUCM-registered, or directly to Webex
Replaced by: Webex Desk Pro (since ~2019)
```

### Webex Desk Pro / Webex Board — Modern Webex Devices

```
Webex Desk Pro      — 27" 4K touchscreen all-in-one collaboration desk endpoint
Webex Desk Mini     — 15.6" smaller variant
Webex Desk          — 24" 1080p
Webex Board 55      — 55" 4K touchscreen (whiteboard-capable)
Webex Board 70      — 70" 4K
Webex Board 85      — 85" 4K
Webex Room Kit      — meeting room codec + camera bundle
Webex Room Kit Mini — smaller meeting room solution
```

```
OS: RoomOS (Cisco's collaboration OS, derived from TC OS)
Provisioning: Webex Control Hub (cloud), with optional CUCM coexistence
Protocol: SIP + Webex (proprietary collaboration protocol)
Setup: QR code activation or 16-digit activation code
```

### Cisco IP Phone (CIP) 6800 / 7800 — Newer Multiplatform

```
CIP-6821   — 2-line monochrome, multiplatform firmware
CIP-6841   — 4-line, multiplatform
CIP-6851   — 4-line + USB, multiplatform
CIP-6861   — WiFi multiplatform
CIP-6871   — 12-line color, multiplatform
CIP-7811   — 1-line entry multiplatform variant
CIP-7821   — 2-line multiplatform
CIP-7841   — 4-line multiplatform
CIP-7861   — 16-line multiplatform
CIP-8811   — 1-line multiplatform
CIP-8841   — 4-line color multiplatform
CIP-8851   — 5-line + USB multiplatform
CIP-8861   — 5-line WiFi multiplatform
CIP-8865   — 5-line color video multiplatform
```

```
Multiplatform = MPP firmware (works with non-CUCM third-party SIP)
Differentiator from CUCM versions: provisioning via XML profile, not CTL/ITL
Default web UI: http://phone-ip
Designed for BroadSoft, Webex Calling, ITSP integration
```

### Webex Calling vs CUCM vs SPA Distinctions

```
Webex Calling: cloud SaaS, account-bound, activation-code provisioning
CUCM:          on-prem (CallManager) cluster, TFTP+CTL/ITL provisioning
SPA:           multi-platform SIP phones, flat-profile XML provisioning
MPP firmware:  hybrid — designed for multi-platform but Cisco-formatted XML
RoomOS:        only on Webex Desk/Board/Room Kit
```

## Protocol — SCCP vs SIP

Two main signaling protocols. Choosing the right firmware is the first step in cisco-phone provisioning.

### SCCP (Skinny Client Control Protocol)

```
Cisco proprietary, also called "Skinny"
TCP port 2000 (non-secure) or 2443 (secure)
Tightly coupled to CUCM (CallManager)
Many Cisco-only features — XSI, Phone Services, native CUCM integration
Stateful, request-response, message types like RegisterMessage, OffhookMessage
Originally Selsius (acquired by Cisco 1998)
Used on: 7900 series, some 8800 series with optional SCCP firmware
End-of-life direction: Cisco moving to SIP across all phones
```

### SIP (Session Initiation Protocol)

```
Standards-based (RFC 3261), interoperates with non-Cisco PBX
UDP/TCP port 5060, TLS 5061, RTP for media (any 16384-32767 range)
INVITE / 200 OK / ACK signaling pattern
Default on: 8800 series, all SPA phones, all MPP firmware, RoomOS endpoints
Optional on: 7900 series via "loaded" SIP firmware
Less Cisco-specific feature support — typical SIP feature set
```

### "Load with SIP" Firmware Swap on 79xx

```
Default 79xx ships with SCCP firmware (e.g., SCCP41.9-3-1SR4-1S.loads)
Can flip to SIP firmware (e.g., SIP41.9-3-1SR4-1S.loads)
Workflow:
  1. Place SIP firmware in TFTP root directory
  2. Edit phone's config XML to point to SIP firmware instead of SCCP
  3. Phone reboots; downloads SIP firmware; converts itself
  4. Re-register against SIP server (Asterisk, FreeSWITCH, etc.)
Caveat: cannot easily revert if SIP fails — need physical access + factory reset
Why: enables 79xx use with non-Cisco PBX deployments
```

### 88xx is SIP-Default

```
88xx phones boot with SIP firmware by default
Older 88xx firmware can be flipped to SCCP for legacy CUCM 6.x/7.x clusters
Modern CUCM 11+ supports SIP-only mode for new deployments
SCCP support deprecated in CUCM 14
```

### SPA is Always SIP

```
SPA family was designed by Linksys/Sipura for SIP-only operation
No SCCP variant exists
Multi-platform: BroadSoft, Asterisk, FreeSWITCH, Metaswitch, ITSP
Configured exclusively via flat-profile XML
```

## Default Access

### Web UI URLs

```bash
# Legacy 79xx and many SPA phones
http://<phone-ip>

# Modern 78xx/88xx and MPP
https://<phone-ip>     # often self-signed cert
http://<phone-ip>      # may fall back to HTTP if HTTPS disabled

# Webex Desk/Board
# Web UI not exposed; managed via Control Hub or USB pairing
```

### Default Admin Credentials

```bash
# SPA phones (all generations)
Username: admin
Password: <blank>

# Legacy 79xx (Cisco)
# No web UI password by default; settings menu unlock code is "**#"

# Modern 78xx/88xx
Username: admin
Password: cisco               # often default
                              # may be set via CUCM or factory default

# MPP firmware (CIP-xxxx)
Username: admin
Password: cisco               # default; should be changed per CUCM/per-account

# After provisioning from CUCM, admin credentials may change
# CUCM Phone Configuration -> Phone Personalization -> Admin Password
```

### Settings Menu Unlock

```bash
# Legacy 79xx
**#                          # unlock code on Settings menu
                            # toggles read/write mode for editing

# 7800/8800 series
# Press Settings (gear icon) -> Status -> Network Configuration
# To edit network: Settings -> Admin Settings -> Network Setup
# Admin password required if set

# SPA phones
# Settings (Menu) -> Network -> view-only by default
# To enable write: Web UI Provisioning tab, set User_Login = admin
```

### Factory Reset

```bash
# 79xx
# At boot, hold "#" then press 1, 2, 3, 4, 5, 6, 7, 8, 9, *, 0, #

# 78xx/88xx
# Power off; hold "#" while plugging in power; release; press 1-2-3-4-5-6-7-8-9-*-0-#

# SPA phones
# Web UI -> Voice -> System -> Factory Reset = Yes
# Or DTMF: pickup -> dial * * * * (four asterisks) -> 73738# -> 1#

# Webex Desk/Board
# Settings -> About this Device -> Factory Reset (sign-out first)
```

## Web UI Tour

### Legacy 79xx Web UI

```
http://<phone-ip>
  Device Information
    - Model number, MAC, serial, firmware version
    - Hardware revision
    - Time since last restart
  Network Configuration
    - DHCP enabled/IP/mask/gateway
    - DNS servers, domain name
    - TFTP server addresses (Option 150 source)
    - VLAN ID (voice + data)
    - LLDP-MED received TLVs (newer firmware)
    - CDP cached neighbor info
  Network Statistics
    - Received/transmitted packets, errors
    - Auto-negotiate result, full/half duplex
  Phone Configuration
    - Read-only display of CUCM-pushed config
    - SCCP/SIP server list, port
    - Locale, time zone
  Streaming Statistics
    - Per-stream: codec, packet count, jitter, packet loss, latency
    - Use during a call to verify QoS
  Status Messages
    - Last 10-20 boot/registration messages
    - "TFTP Failed", "ITL Mismatch", "Phone Load ID"
```

### Modern 78xx/88xx Web UI

```
https://<phone-ip>  (admin/cisco)
  Device Information
    - Model, MAC, serial
    - Firmware version
    - Active firmware vs inactive (for rollback)
    - Hardware version
    - Last upgrade date
  Network Setup
    - DHCP / Static IP
    - PoE status (Class 1, 2, 3, 3+)
    - LLDP-MED voice VLAN, priority
    - 802.1X EAP status
    - VPN settings (built-in client for some models)
  Status -> Statistics
    - Stream stats
    - Cumulative call counters
    - Active call media (codec, bandwidth, jitter, loss)
  Phone Configuration
    - Phone Profile (read-only after CUCM push)
    - SIP profile, transport (UDP/TCP/TLS)
    - Authentication info (username, register URI)
  Status -> Diagnostics
    - PRT (Problem Report Tool) trigger
    - Trace level
    - System logs
  Personalization
    - Custom ringtones
    - Background image
    - Brightness
```

### SPA Web UI

```
http://<phone-ip>  (admin/blank)
  Voice (main tab)
    Info     - Model, firmware, IP, MAC, status
    System   - System config: NTP, time zone, GMT offset, syslog server
    SIP      - Per-line/per-extension SIP settings
    Provisioning - Profile rule, resync settings, profile auth
    Regional - Locale, dial tone, ring tone, dial plan
    Phone    - Web UI display, station name, line key configuration
    Ext 1-4  - Per-extension SIP/Provisioning/Audio/Video/Subscriber
    User     - User-tier settings (call forward, do not disturb)
  Status (top tab)
    System status, Network status, Voice status
  Provisioning (top tab)
    XML profile fetched, last resync, next resync
  Quality (top tab)
    Per-call MOS, jitter, packet loss
  Logging (top tab)
    Syslog stream + view
```

## CUCM (Cisco Unified Communications Manager)

The canonical enterprise call control. Phones register; CUCM mediates all signaling.

### Architecture

```
CUCM Cluster (typical)
  Publisher (CCMPUB) — primary database, only one active publisher
  Subscriber 1 (CCMSUB1) — secondary signaling, real-time call processing
  Subscriber 2 (CCMSUB2) — additional signaling node
  TFTP Server — dedicated TFTP node for firmware + configs
  CDR Repository — call detail records archive

Each phone registers to a list of subscribers (via CUCM Group)
On failover, phone re-registers to next subscriber in the group
```

### Registration Flow

```
1. Phone boots — DHCP request
2. DHCP response includes Option 150 with CUCM TFTP server IPs
3. Phone connects to TFTP, requests CTL.tlv (security trust list, if applicable)
4. Phone connects to TFTP, requests ITL.tlv (initial trust, if applicable)
5. Phone requests SEP<MAC>.cnf.xml from TFTP
6. SEP<MAC>.cnf.xml lists firmware load + CUCM IPs + dial settings
7. Phone fetches firmware load (cmterm-x.x.x.zip / .tar.zip)
8. Phone registers with CUCM (SCCP or SIP) using credentials in config
9. CUCM responds with capability + assigns lines/services
10. Phone is operational
```

### Device Pool

```
Logical grouping of phones with shared:
  - CUCM Group (which subscribers to register with)
  - Region (codec preferences for inter-region calls)
  - Date/Time Group (locale, time zone)
  - Media Resource Group (transcoders, conference bridges, MoH)
  - Calling Search Space (which partitions phone can dial)
  - Location (call admission control)
  - Network Locale (language/country)
  - User Locale (UI language for the phone)
  - SRST Reference (Survivable Remote Site Telephony fallback)
  - Roaming Sensitive Settings (bandwidth, location)
```

### Calling Search Space (CSS)

```
Ordered list of Partitions
Determines which dial-string-patterns the phone can match
Example:
  Internal_CSS = [Internal_PT, Local_PT, LongDistance_PT]
  Restricted_CSS = [Internal_PT]
Apply CSS to:
  Phone (line-level CSS) — per-DN dialing rights
  Translation Patterns (override CSS for specific patterns)
  Route Patterns (PSTN egress control)
```

### Device Profile (Extension Mobility)

```
"Logical phone" — a profile that overlays on physical phone
User logs in -> their lines + buttons appear on the phone
Used for hot-desking
Configured in: User Management -> Device Profile
Activated via: Extension Mobility service URL on phone soft key
```

## CUCM Provisioning — CTL, ITL, LSC

Security trust model. Phones MUST verify CUCM identity before registering.

### CTL (Certificate Trust List)

```
File: CTLFile.tlv
Contents: signed list of CUCM cluster certificates + ServerProxyFunction
Phone fetches via TFTP at boot
Phone uses CTL to validate ITL signatures
Generated when "Mixed Mode" cluster security is enabled
Signed with: USB eToken (Cisco) or Tokenless TFTP (CUCM 11+)
```

### ITL (Initial Trust List)

```
File: ITLFile.tlv
Contents: signed list of trusted certs (TFTP, CCM, CAPF, TVS)
Generated by every CUCM cluster (mixed-mode and non-secure)
Phone uses ITL to validate certs presented during registration
Replaces the older "trust on first use" (TOFU) model
Default since CUCM 8.x
Used for: secure phone services, firmware download verification
```

### LSC (Locally Significant Certificate)

```
Per-phone certificate, issued by CAPF (Certificate Authority Proxy Function)
Used for: 802.1X EAP-TLS authentication, SIP-TLS to CUCM
Workflow:
  1. CAPF Profile created in CUCM with operation "Install/Upgrade"
  2. Authentication mode: Null String, By Existing Certificate, By Authentication String
  3. Phone provisioned with profile; CAPF generates and pushes LSC
  4. LSC stored in phone's secure storage; replaces MIC for some operations
```

### Security Modes

```
Non-Secure
  - Plain SCCP/SIP signaling
  - RTP media unencrypted
  - Phone trust ITL (validates cert chain)
  - No CTL needed

Authenticated
  - Signaling: TLS-encrypted SCCP or SIP-TLS
  - Media: RTP unencrypted (still in clear)
  - Both ends mutually authenticated
  - Mixed Mode cluster required, CTL signed

Encrypted
  - Signaling: TLS-encrypted
  - Media: SRTP (encrypted RTP)
  - Both ends mutually authenticated
  - Mixed Mode cluster required, CTL signed
```

### Switching Modes

```bash
# CUCM Admin
System -> Enterprise Parameters -> Cluster Security Mode
  0 = Non-Secure
  1 = Mixed-Mode (some authenticated/encrypted, some non-secure)

# Then per-Phone Security Profile
Device -> Phone -> Security Profile = "Standard SIP Secure Profile"
                                    or "Standard SCCP Secure Profile"
                                    or "Encrypted Profile"
```

## SPA Configuration

Linksys-lineage SPA phones use flat XML profiles.

### Provisioning Server Configuration

```
SPA -> Voice -> Provisioning
  Provisioning Enable: Yes
  Resync On Reset: Yes
  Resync Random Delay: 600 sec  (avoid thundering herd)
  Resync At (HHmm): 0300
  Resync Periodic: 86400 sec (daily)
  Resync Error Retry Delay: 3600 sec
  Forced Resync Delay: 14400 sec
  Resync From SIP: Yes  (allow Notify-Resync on SIP NOTIFY)
  Resync After Upgrade Attempt: Yes
  Profile Rule: <URL with $MA>
```

### Profile Rule Format

```bash
# Static URL
http://provserver.example.com/spa/spa.cfg

# Per-MAC dynamic
http://provserver.example.com/spa/$MA.cfg

# Multi-protocol
tftp://10.1.1.1:69/spa/$MA.cfg
ftp://user:pass@ftpserver/spa/$MA.cfg
https://provserver.example.com/spa/$MA.cfg

# Multiple profiles (cascading)
[--key spasystemkey] http://server1/$MA.cfg
http://server2/$MA.cfg

# Conditional — only download if file changed
[--diffieHellman] http://server/$MA.cfg

# Macro variables
$MA   = MAC address (lowercase, no separator)
$MAU  = MAC address (uppercase, no separator)
$MAC  = MAC address (uppercase, colon-separated)
$MD   = Domain name
$PSN  = Product Serial Number
$SN   = Serial Number
$IP   = IP address
$EXT  = Extension number
$USER = User ID
```

### Supported Protocols

```
TFTP — UDP port 69; insecure but common
HTTP — TCP port 80; supports auth
HTTPS — TCP port 443; recommended for production
FTP — TCP port 21; legacy
DHCP Option 66 — server name (string)
DHCP Option 159 — alternative TFTP server
DHCP Option 160 — Cisco-specific (used by some SPA models)
```

### SPCONFIGURATION_FILE (Configuration File Variable)

```bash
# Set on the SPA web UI under Provisioning
SPCONFIGURATION_FILE = /spa/$MA.cfg
                     # The file path requested from Profile Rule URL

# Example mapping
Profile Rule:        https://prov.example.com/spa/cfg/$MA.xml
Phone fetches:       https://prov.example.com/spa/cfg/aabbccddeeff.xml
                     (where aabbccddeeff = MAC of this phone)
```

## SPA Profile Format

XML, root element <flat-profile>. All settings as XML elements with element name = SPA setting name.

### Schema Basics

```xml
<flat-profile>
  <!-- All settings live as direct children -->
  <Setting_Name>value</Setting_Name>

  <!-- Per-line settings have line index in [N] -->
  <Setting_Name_1_>value</Setting_Name_1_>  <!-- line 1 -->
  <Setting_Name_2_>value</Setting_Name_2_>  <!-- line 2 -->

  <!-- Some settings use bracket notation -->
  <Setting>value[1]</Setting>  <!-- alternate -->
</flat-profile>
```

### Common Top-Level Settings

```
SIP_Transport_1_                = UDP | TCP | TLS
Proxy_1_                        = sip-server.example.com:5060
Outbound_Proxy_1_               = sbc.example.com:5060
Use_Outbound_Proxy_1_           = Yes
Display_Name_1_                 = "John Doe"
User_ID_1_                      = 1001
Password_1_                     = secret
Auth_ID_1_                      = 1001
Use_Auth_ID_1_                  = No
Register_1_                     = Yes
Register_Expires_1_             = 3600
SIP_Port_1_                     = 5060
RTP_Port_Min_1_                 = 16384
RTP_Port_Max_1_                 = 16482
Preferred_Codec_1_              = G711u | G711a | G726-32 | G729a | G722
Codec_Negotiation_1_            = Default | List All
DTMF_Tx_Method_1_               = Auto | InBand | AVT | INFO | Auto+INFO
Voice_Quality_Report_Address_1_ = report-server.example.com
Dial_Plan_1_                    = (S0 <:1408>xxxxxxx | [2-9]xxxxxxx | 1[2-9]xxxxxxxxx | etc.)
Send_Anonymous_CID_Block_1_     = No
```

### SPA-2102 (ATA — Analog Telephone Adapter)

```
2 FXS ports (analog phones connect)
Schema: similar flat-profile but with FXS-specific lines
Common settings: Tip_Voltage, Ringback_Tone, Dial_Tone, Caller_ID_Method, FXS_Port_Impedance
```

### SPA-3102 (Hybrid: 1 FXS + 1 FXO)

```
1 FXS for analog phone
1 FXO for PSTN line bridge
Schema includes PSTN-side settings: PSTN_Caller_ID_Method, PSTN_Disconnect_Tone
Common deployment: home/SOHO bridging cell phone -> PSTN -> IP
```

### SPA-9xx (IP Phone)

```
Direct IP phone, no analog ports
Schema includes per-line button assignments + soft key configurations
Most extensive multi-line variant
```

## SPA Sample Profile

Asterisk-style provisioning for SPA-504G:

```xml
<flat-profile>
  <!-- System -->
  <Phone_DSCP>EF</Phone_DSCP>
  <Phone_VLAN_Enable>Yes</Phone_VLAN_Enable>
  <Phone_VLAN_ID>10</Phone_VLAN_ID>
  <NTP_Server_1>pool.ntp.org</NTP_Server_1>
  <Time_Zone>GMT-08:00</Time_Zone>
  <Daylight_Saving_Time_Enable>Yes</Daylight_Saving_Time_Enable>

  <!-- Provisioning -->
  <Provision_Enable>Yes</Provision_Enable>
  <Resync_On_Reset>Yes</Resync_On_Reset>
  <Resync_Periodic>86400</Resync_Periodic>
  <Profile_Rule>https://prov.example.com/spa/$MA.xml</Profile_Rule>

  <!-- Line 1 -->
  <Line_Enable_1_>Yes</Line_Enable_1_>
  <SIP_Transport_1_>UDP</SIP_Transport_1_>
  <Proxy_1_>asterisk.example.com</Proxy_1_>
  <Outbound_Proxy_1_></Outbound_Proxy_1_>
  <Use_Outbound_Proxy_1_>No</Use_Outbound_Proxy_1_>
  <SIP_Port_1_>5060</SIP_Port_1_>
  <Display_Name_1_>John Doe</Display_Name_1_>
  <User_ID_1_>1001</User_ID_1_>
  <Password_1_>topsecret</Password_1_>
  <Auth_ID_1_>1001</Auth_ID_1_>
  <Use_Auth_ID_1_>No</Use_Auth_ID_1_>
  <Register_1_>Yes</Register_1_>
  <Register_Expires_1_>3600</Register_Expires_1_>
  <Preferred_Codec_1_>G711u</Preferred_Codec_1_>
  <Use_Pref_Codec_Only_1_>No</Use_Pref_Codec_Only_1_>
  <Codec_Negotiation_1_>List All</Codec_Negotiation_1_>
  <DTMF_Tx_Method_1_>AVT</DTMF_Tx_Method_1_>
  <Dial_Plan_1_>(S0 &lt;:9008&gt;xxx.|[3469]11|[2-9]xxxxxxxxx)</Dial_Plan_1_>

  <!-- Line 2 (different extension) -->
  <Line_Enable_2_>Yes</Line_Enable_2_>
  <Proxy_2_>asterisk.example.com</Proxy_2_>
  <User_ID_2_>1002</User_ID_2_>
  <Password_2_>secret2</Password_2_>
  <Display_Name_2_>Jane Doe</Display_Name_2_>

  <!-- Line 3 disabled -->
  <Line_Enable_3_>No</Line_Enable_3_>

  <!-- Line 4 disabled -->
  <Line_Enable_4_>No</Line_Enable_4_>

  <!-- Phone settings -->
  <Backlight_Default>10</Backlight_Default>
  <Screen_Saver_Enable>Yes</Screen_Saver_Enable>
  <Screen_Saver_Trigger_Time>120</Screen_Saver_Trigger_Time>

  <!-- Soft Keys (line keys) -->
  <Line_Key_1_>fnc=sd;ext=1001;nme=John Doe</Line_Key_1_>
  <Line_Key_2_>fnc=sd;ext=1002;nme=Jane Doe</Line_Key_2_>
  <Line_Key_3_>fnc=blf;sub=2000@asterisk.example.com;nme=Reception</Line_Key_3_>
  <Line_Key_4_>fnc=sd;ext=1100;nme=Voicemail</Line_Key_4_>
</flat-profile>
```

## CUBE (Cisco Unified Border Element)

CUBE is Cisco's SBC running on IOS-XE routers (ISR, ASR, CSR1000v, CAT8000).

### CUBE Architecture

```
CUBE = router with voice features enabled
Sits at network edge between internal voice network and external SIP trunk
Functions:
  - SIP trunking to ITSP (Internet Telephony Service Provider)
  - Protocol interworking (SCCP <-> SIP, H.323 <-> SIP)
  - Topology hiding
  - Encryption boundary (SRTP <-> RTP)
  - Codec transcoding (transcode resources required)
```

### CUBE Configuration Skeleton

```cisco
! Global voice services
voice service voip
 ip address trusted list
  ipv4 10.0.0.0 255.0.0.0       ! permit internal range
  ipv4 172.16.0.0 255.240.0.0
  ipv4 1.2.3.4 255.255.255.252  ! permit ITSP range
 allow-connections sip to sip
 sip
  bind control source-interface GigabitEthernet0/0
  bind media source-interface GigabitEthernet0/0
  early-offer forced
  registrar server expires max 3600 min 60

! SIP UA configuration
sip-ua
 retry invite 3
 retry register 5
 timers connect 1000
 timers retry 200 200
 timers register 60
 transport tcp tls v1.2

! Voice class for ITSP (parameters reused across dial-peers)
voice class tenant 1
 sip-server ipv4:1.2.3.4:5060
 transport udp
 outbound-proxy ipv4:1.2.3.4:5060
 srtp-crypto 1
 session transport udp

! Voice class codec
voice class codec 1
 codec preference 1 g711ulaw
 codec preference 2 g729r8

! Inbound dial-peer (from ITSP)
dial-peer voice 100 voip
 description "Inbound from ITSP"
 destination-pattern .T
 session protocol sipv2
 session target sip-server
 voice-class codec 1
 voice-class tenant 1
 incoming called-number .T
 dtmf-relay rtp-nte sip-kpml
 no vad

! Outbound dial-peer (to CUCM)
dial-peer voice 200 voip
 description "Outbound to CUCM"
 destination-pattern 5...
 session protocol sipv2
 session target ipv4:10.1.1.10:5060
 voice-class codec 1
 dtmf-relay rtp-nte sip-kpml
 no vad
```

### Dial-Peer Match Order

```
Inbound:  destination-pattern matches called number
          OR incoming called-number matches inbound called number
          OR answer-address matches calling number
          OR session target matches source IP

Outbound: destination-pattern matches dialed number
          dial-peer matched on most specific pattern
          load-balance among ties
```

### Voice-Class Tenant

```
Profile of SIP settings reused across dial-peers
Reduces config redundancy when multiple ITSP trunks
Tenant 1: ITSP A (different SBC IP, codec, transport)
Tenant 2: ITSP B
Apply per dial-peer: voice-class tenant N
```

## 88xx Generation Configuration

### TFTP-Based Provisioning

```
Phone DHCP -> Option 150 -> TFTP IP
Phone fetches:
  CTL/ITL files (security trust)
  SEPMACADDR.cnf.xml (per-phone config)
  Default device load file (firmware)
  ConfigFiles/ringtones.xml (custom ringtones)
  ConfigFiles/dialplan.xml (per-cluster dial plan)
```

### XML Configuration Files Hierarchy

```
TFTP root /
  CTLFile.tlv          — security trust list (mixed-mode)
  ITLFile.tlv          — initial trust list (always)
  SEPAABBCCDDEEFF.cnf.xml   — per-phone (MAC-based filename)
  CTLSEPAABBCCDDEEFF.tlv    — per-phone TLV trust
  XMLDefault.cnf.xml   — default per-device-type config
  Cisco-CCM-CompatVersion.xml — CUCM version sync
  CallManagerGroup_*.cnf.xml — CallManager group definitions
  DialNumber/*.cnf.xml — DN-related configs
  ConfigFiles/
    GeoLocation/*.cnf.xml
    Ringlist.xml       — custom ringtones list
    DefaultPhoneRingList.xml — default ringtones
    Distinctive.cnf.xml — distinctive ring config
  Locales/
    en_US.cnf.xml      — English (US) translation
    es_ES.cnf.xml      — Spanish (Spain) translation
  PhoneCRRDevices/*.cnf.xml — per-device configs
  ConfigFiles/Distinctive/dialtone.cnf.xml — dial tone

# Per-phone XML name format
SEP<MAC_uppercase_no_separator>.cnf.xml
e.g., SEPAABBCCDDEEFF.cnf.xml
```

### Per-Phone Tags Structure

```xml
<device>
  <devicePool>
    <name>Default</name>
  </devicePool>
  <ipAddressMode>0</ipAddressMode>
  <deviceProtocol>SIP</deviceProtocol>
  ...
</device>
```

## SEPMAC.cnf.xml

The canonical per-phone XML config file for 88xx phones.

### Top-Level Structure

```xml
<device>
  <fullConfig>true</fullConfig>
  <deviceProtocol>SIP</deviceProtocol>
  <deviceProtocolName>SIP</deviceProtocolName>
  <devicePool>
    <revision>1</revision>
    <name>Default</name>
    <dateTimeSetting>
      <name>CMLocal</name>
      <dateFormat>D/M/YA</dateFormat>
      <timeFormat>24-hour</timeFormat>
      <timeZone>America/New_York</timeZone>
    </dateTimeSetting>
    <callManagerGroup>
      <members>
        <member>
          <priority>0</priority>
          <callManager>
            <name>CCMSUB1</name>
            <ports>
              <ethernetPhonePort>2000</ethernetPhonePort>
              <sipPort>5060</sipPort>
              <securedSipPort>5061</securedSipPort>
            </ports>
            <processNodeName>10.1.1.10</processNodeName>
          </callManager>
        </member>
      </members>
    </callManagerGroup>
  </devicePool>
  <commonProfile>
    <phonePassword>cisco</phonePassword>
    <backgroundImageAccess>true</backgroundImageAccess>
    <callLogBlfEnabled>2</callLogBlfEnabled>
  </commonProfile>
  <ipDtmfMode>1</ipDtmfMode>
  <preferredCodec>g711ulaw</preferredCodec>
  <vad>false</vad>
  <ringSettingBusyStationPolicy>0</ringSettingBusyStationPolicy>
  <ringSettingIdleStationPolicy>0</ringSettingIdleStationPolicy>
  <sshUserId>cisco</sshUserId>
  <sshPassword>cisco</sshPassword>
  <httpAccessEnabled>true</httpAccessEnabled>
  <webAccess>1</webAccess>
  <sipDirectoryNumberConfig>
    <line>
      <featureID>9</featureID>
      <featureLabel>1001</featureLabel>
      <subscribeCallingSearchSpaceName>Standard_CSS</subscribeCallingSearchSpaceName>
      <featureKey>1</featureKey>
      <name>1001</name>
      <displayName>John Doe</displayName>
      <contact>1001@cucm.example.com</contact>
      <authName>1001</authName>
      <authPassword>secretkey</authPassword>
    </line>
    <line>
      <featureID>9</featureID>
      <featureLabel>1002</featureLabel>
      <featureKey>2</featureKey>
      <name>1002</name>
    </line>
  </sipDirectoryNumberConfig>
  <advancedSecurity>
    <signalingEncryption>0</signalingEncryption>
    <transportProtocol>0</transportProtocol>
  </advancedSecurity>
  <encryption>
    <serverCertSourceFile>1</serverCertSourceFile>
    <ekuExtensionEnabled>1</ekuExtensionEnabled>
  </encryption>
  <networkParms>
    <ldpwExpiry>3600</ldpwExpiry>
    <ldpFreq>3600</ldpFreq>
    <vlan>10</vlan>
    <dscpForCalls>46</dscpForCalls>
    <dscpForSig>26</dscpForSig>
    <ipv6CfgMode>0</ipv6CfgMode>
  </networkParms>
  <loadInformation>SIP78XX.14-1-1SR3-1</loadInformation>
  <vendorConfig>
    <recoveryURL>https://10.1.1.10/cmm</recoveryURL>
  </vendorConfig>
</device>
```

### Key Elements Explained

```
<deviceProtocol>           SIP | SCCP
<callManagerGroup>          Ordered list of CCM nodes (failover)
<sipDirectoryNumberConfig>  Per-line registration credentials
<commonProfile>             Phone Personalization (password, web access, etc.)
<advancedSecurity>          Signaling encryption mode
<networkParms>              VLAN, QoS, DSCP, IPv6
<loadInformation>           Firmware load file name
<vendorConfig>              Cisco-specific extensions
<webAccess>                 1 = enabled, 0 = disabled
```

## Phone Load (Firmware) Naming

### Naming Conventions

```
Cisco firmware filenames follow pattern: cmterm-<series>-<version>.zip
Example: cmterm-7942-9-3-1SR4-1S.zip

Decoded
  cmterm    = Cisco multi-terminal (firmware archive)
  7942      = phone model series
  9-3-1SR4-1S = version (Service Release / Special)
                Major.Minor.Patch SR=Service Release N=Number S=Special
                Modern: x.y.z.w (e.g., 14-1-1SR3-1)
  .zip      = legacy archive format

Modern format
  cmterm-7800.7811.7821.7841.7861-12-9-1-3-K9-V300.zip
  Multi-model bundle for 7800 series
```

### File Types

```
.loads     — legacy individual firmware module (79xx era)
             SCCP41.9-3-1SR4-1S.loads
             SIP41.9-3-1SR4-1S.loads
             dsp41.9-3-1SR4-1S.loads (DSP firmware)
             term41.default.loads (terminal config defaults)
.tar.zip   — modern bundled archive (78xx/88xx era)
             cmterm-78xx.88xx-12-9-1-3.tar.zip
.cop.sgn   — CUCM Operations Patch (signed)
             cmterm-locale-installer-<version>.cop.sgn (locales)
.sgn       — signed firmware archive
```

### Firmware Distribution

```bash
# CUCM admin UI
Cisco Unified OS Administration -> Software Upgrades -> Install/Upgrade
  Source: SFTP / Local
  Path: /path/to/cmterm-X.X.X.zip
  -> Install

# CUCM admin UI Phone Firmware Load
Bulk Administration -> Phones -> Update Phones -> Query
  Phone Load Name = (specific load) or empty (use device default)

# Default load per model
Device -> Device Settings -> Device Defaults
  e.g., Cisco 8845: Default Load = SIP8845-12-9-1-3-K9
```

### Enabling a New Load

```
1. Upload .cop.sgn or .zip to CUCM (Software Upgrades)
2. Restart TFTP service (Cisco Tomcat or Cisco TFTP)
3. New firmware appears in TFTP root
4. Update Device Defaults, OR
5. Update individual phone's Phone Load Name
6. Phone resets, downloads new firmware
```

## SCCP vs SIP Firmware

### Operational Differences

```
SCCP                                    SIP
====                                    ===
Cisco-only feature support               Standards-based RFC 3261
Native CUCM integration                  Works with any SIP server
Stateful (TCP)                           Stateless (UDP/TCP/TLS)
Many proprietary features                Less Cisco-specific
Phone services via XML push             Phone services via SUBSCRIBE/NOTIFY
Internal-only XSI extensions            Standard NOTIFY for events
KPML for digit collection (default)     KPML or INFO for digit collection
Default SCCP digit collection           Tone-based + INVITE-based
                                        (per-INVITE digit-by-digit)
Cisco BLF via XML push                  SIP-PRESENCE / SUBSCRIBE
G.711 G.722 G.729 codecs                G.711 G.722 G.729 + Opus + iLBC
```

### Flipping 79xx to SIP

```bash
# 1. Place SIP firmware in TFTP
   SIP41.9-3-1SR4-1S.loads
   SIP41.9-3-1SR4-1Sdsp.loads
   SIP41.9-3-1SR4-1Sapps.loads
   SIP41.9-3-1SR4-1Scnu.loads
   SIP41.9-3-1SR4-1Sjar.loads
   term41.default.loads

# 2. Edit XMLDefault.cnf.xml or SEP<MAC>.cnf.xml
   <loadInformation>SIP41.9-3-1SR4-1S</loadInformation>
   (was previously SCCP41.9-3-1SR4-1S)

# 3. Phone factory-reset or reload
   Settings -> Reset Settings -> Reset

# 4. Verify firmware swap on phone
   Settings -> Status -> Firmware Versions
   App Load ID = SIP41.x.x.x (not SCCP41.x.x.x)
```

## Lines / Extensions

### Multiple Lines per Phone

```
A "line" is a registered SIP/SCCP endpoint
A "DN" (Directory Number) is the dial-able extension assigned to that line

Example: 6-line 8861 phone
  Line 1: ext 1001 (primary)
  Line 2: ext 1002 (secondary, shared with manager's office)
  Line 3: BLF for 1003
  Line 4: speed-dial Voicemail
  Line 5: service URL
  Line 6: empty
```

### Line Group / Hunt Group Routing in CUCM

```
Line Group: ordered list of DNs
  Line Group "Sales_LG" = [1001, 1002, 1003] (top-down)
                       = [1001, 1002, 1003] (round-robin)
                       = [1001 only] (broadcast)
  
Hunt List: ordered list of Line Groups
  Hunt List "Sales_HL" = [Sales_LG, Sales_Backup_LG]

Hunt Pilot: a "shadow DN" that routes calls to the Hunt List
  Hunt Pilot "5000" -> Hunt List "Sales_HL"
  Caller dials 5000 -> CUCM rings DNs in Sales_LG order
```

### Line Label vs DN

```
Line Label: text shown on phone display next to button
            "John Doe" or "Sales Reception"

DN:         actual dialable extension
            "1001" or "5000"

Configured separately in CUCM Device -> Phone -> Line config
```

## Speed Dials / Programmable Buttons

### Per-Button-Type Catalog

```
Line                — primary call appearance for a DN
Speed Dial          — quick dial number, plays through line
Speed Dial BLF      — speed dial + Busy Lamp Field (presence indicator)
Service URL         — XML-based phone service (HTTP-fetched)
IP Phone Service    — XML-based phone service (Cisco proprietary)
Native Phone Service — Cisco-specific phone services (e.g., Extension Mobility)
Call Pickup         — pick up calls in your call pickup group
Mobility            — toggle mobile remote destination
Privacy             — privacy lock indicator
Group Pickup        — pick up calls from any phone in pickup group
Other               — service URL parameter pass-through
None                — empty/disabled
```

### CUCM Configuration

```
Device -> Phone -> Phone Configuration
  Per Phone:
    Phone Button Template — defines button layout (1=Line, 2=Speed Dial, etc.)
    Each button has index 1..N
  
  Per-Button override:
    Button 1: Line "1001"
    Button 2: Speed Dial "1002 - Reception"
    Button 3: BLF "1003 - Manager"
    Button 4: Service URL "http://server/forward.xml"
```

### Speed Dial Configuration

```
Device -> Phone -> Phone Button Template -> Add SD button
  Index: 5
  Type: Speed Dial
  Label: "Operator"
  Number: 0
  Display Name: optional
```

### Programmable Line Key (PLK) Templates

```
Templates are model-specific
SCCP-Standard-7942-PLK = template for 7942 SCCP phones
SIP-Default-8841-PLK = template for 8841 SIP phones
```

## BLF / Presence

### Busy Lamp Field

```
BLF subscribes to another DN's call state
LED color indicates:
  Green/off = idle
  Red/lit   = on a call (busy)
  Amber     = ringing

Press to: pickup ringing call (with Pickup feature) or speed-dial
```

### CUCM Subscribe Calling Search Space

```
CUCM Phone Configuration -> Subscribe Calling Search Space
  CSS used to evaluate which DN this phone can subscribe to (BLF)
  
Without subscribe CSS: BLF will fail to authorize
Best practice: separate CSS for subscribe vs dial calls
```

### BLF SD Button Mapping

```
CUCM Phone Configuration -> Phone Button -> Type = Speed Dial BLF
  Number: 1003
  Label: "Manager"
  
Phone shows light next to Manager button = busy/idle/ringing
```

### SIP-Based BLF (88xx, MPP, SPA)

```
Phone sends: SUBSCRIBE sip:1003@cucm.example.com Event: dialog
CUCM/Asterisk responds: NOTIFY with dialog state XML
Phone parses dialog state, updates LED
```

## CUCM Class of Service

### Calling Search Space + Partition Model

```
Partition: a "bucket" of dialable numbers (DN, route patterns)
CSS:       ordered list of partitions

Phone's CSS evaluated against destination:
  - Outbound dial -> CUCM iterates phone's CSS partitions in order
  - First match wins
  - If phone's CSS doesn't include partition holding destination -> blocked
```

### Outbound Restrictions Example

```
Internal_PT     — extensions 1xxx, 2xxx
Local_PT        — local PSTN routes
LongDistance_PT — long distance PSTN routes
International_PT — international PSTN routes

User CSS = [Internal_PT, Local_PT]            — internal + local only
Manager CSS = [Internal_PT, Local_PT, LongDistance_PT]  — also long distance
Executive CSS = [Internal_PT, Local_PT, LongDistance_PT, International_PT]
```

### Dial-Pattern Matching

```
Route Pattern: dial-string with wildcards
  91xxx......  = 9 + 1 + xxx (NPA) + 7 digits
  9011T        = 9 + 011 + T (transparent — anything after)
  9!#          = 9 + ! (any digits) + # (terminator)
  Wildcards: . = any digit, X = any digit, [2-9] = range, ! = any, T = anything
```

## CUCM Translations

### Translation Pattern

```
Pre-CSS dial transformation
  Pattern: 9.@PSTN
  Calling Party Mask: 555XXXX (mask CLID for outbound)
  Called Party Mask:  9X (strip leading 9)
  
Use case:
  - Strip access digits (9, 8) before sending to PSTN
  - Add area code for local numbers
  - Block specific destinations
```

### Calling-Party-Mask

```
Replaces calling number on outbound calls
  Mask: XXXX1234 + Calling Number: 5551001 = 5551234 (last 4 replaced)
  
Use cases:
  - DID assignment per user
  - Privacy: hide internal extensions to PSTN
  - Brand: present a single Caller ID for all outbound from a department
```

### Called-Party-Mask

```
Replaces called number on inbound calls
  Mask: XXXX1001 + Inbound DID: 5551001 -> internal DN 1001
  
Use case: route inbound DIDs to specific internal extensions
```

## Codec Preferences

### Region in CUCM

```
A "Region" = group of devices that share inter-region codec policies
Typical setup:
  Default_Region: G.722 within, G.729 between
  Branch1: G.711 within, G.729 to Default_Region
  Branch2: G.711 within, G.729 to Default_Region

Codec preference per Region:
  Within Region: G.722 (highest quality, low compression)
  Between Regions: G.729 (compressed, lower bandwidth)
```

### Inter-Region Transcoding

```
If two regions have incompatible codec preferences:
  Region A prefers G.711, Region B prefers G.729
  Phones in A and B cannot agree on codec without help
  CUCM allocates a transcoder resource (DSP) to bridge

Configure transcoder in CUCM:
  Device -> Media Resources -> Transcoder
  Add SIP Profile transcoder (DSP-based)
```

## Bandwidth / CAC

### Location-Based Call Admission Control

```
Each Location has:
  - Audio Bandwidth Limit (kbps)
  - Video Bandwidth Limit (kbps)
  - Inter-region (between locations) bandwidth budget

CUCM tracks active calls
If a call would exceed budget -> reject (with reorder tone)
```

### LBM (Location Bandwidth Manager)

```
Service that tracks call admission across cluster
  Replicates location states across CUCM nodes
  Performs admission control decisions
  Configure: Cisco Unified Serviceability -> Service Activation
```

### RSVP (Resource Reservation Protocol)

```
End-to-end QoS reservation (rare in modern deployments)
RSVP-aware routers reserve bandwidth along path
CUCM coordinates RSVP signaling between phones
Used in: ATM/Frame Relay legacy networks (largely deprecated)
```

### Enhanced Locations CAC (ELCAC)

```
Modern CAC model (CUCM 9+)
Hub-and-spoke or multi-tier topology
  Locations with parent-child relationship
  Bandwidth shared/reserved at each level
Features:
  - Inter-cluster CAC (between CUCM clusters)
  - Better failover handling during WAN outages
  - More complex but accurate for multi-site
```

## Codec Catalog

### Standard Codecs

```
G.711μ (PCMU)      — North America/Japan, 64 kbps, 8 kHz
G.711A (PCMA)      — Europe/RoW, 64 kbps, 8 kHz
G.722              — wideband, 64 kbps, 16 kHz
G.722.1            — wideband, 24/32 kbps
G.722.2 (AMR-WB)   — wideband, 6.6-23.85 kbps
G.726-32           — ADPCM, 32 kbps
G.729a             — narrowband, 8 kbps
G.729ab            — same with VAD/silence suppression
iLBC                — narrowband, 13.33/15.2 kbps, packet-loss-tolerant
Opus               — wideband, variable bit rate, 6-510 kbps
                    (88xx late firmware 14.x+)
```

### Default 88xx Codec Preferences

```
Within region:    G.722 (preferred), then G.711μ
Between regions:  G.729 (with transcoder), then G.711μ
Mobile/cellular:  iLBC (loss-tolerant)
Peer-to-peer:     Opus (modern, variable bitrate)
```

### "No Transcoding Allowed" for Inter-Region

```
If Region A advertises G.711 only and Region B advertises G.729 only
  AND no transcoder is configured -> call fails
Workaround: configure overlapping codec, or add transcoder DSP resource
```

## SIP Profile (CUCM)

Per-phone SIP behavior controlled by SIP Profile.

### Common SIP Profile Settings

```
Use Fully Qualified Domain in SIP Requests: true (use dn@cucm.example.com)
SIP Rel1XX Options: Send PRACK if 1XX contains SDP
Call HOLD Ring Back Option: 0 (none) | 1 (provide ring) | 2 (cancel after CC)
Stutter Message Waiting: enabled (causes phone to play stutter dial-tone for VM)
Allow Lines Removable: false (admin can remove from CUCM only)
Resource Priority Namespace: dsr | drsn (priority calling)
Customer-specified MWI digit length: blank
Reroute Outbound Calls Before SIP Reinvite: false
Send Direct Calls Without Reinvite: false
Use Call-Info Header for Push: false (phone services delivery method)
Auto Subscribe MOH Sources: false
Send Quality of Service in SIP Body: false
DTMF DB Level: -16
Telephony Interface DTMF Tx: RFC 2833 (out-of-band, RTP-NTE)
Negotiate audio (early offer): on | off (force early offer for SDP)
Refer-To URI: blank or specific URI
Allow Line State to Reset MWI: false
SDP Inactive Behavior: AddReceive (modern)
SIP Notify Method: Application/SDP
Disconnect Procedure: Reverse RTP (close RTP after BYE/200)
Refer Method: Refer to Refer-To
Subscribe Method: SUBSCRIBE
Notify Method: NOTIFY
```

### KPML for Digit Collection

```
KPML = Key Press Markup Language
Phone subscribes to KPML during call setup
CUCM sends NOTIFY events for each digit press
Used for: 
  - Mid-call DTMF transmission
  - Real-time digit collection during long-extension dialing
  - Auto-attendant interactions
  
Default on SCCP, common on SIP
Alternative: SIP INFO method (less common, more compatible)
```

### Registration Interval

```
Phone registers with: Expires: 3600
                    (default 1 hour)
Phone refreshes registration every 80% of expires
  3600 sec * 0.8 = 2880 sec (~48 minutes)
On registration failure, phone retries with exponential backoff
```

## SIP-OAuth

Modern (CUCM 12.5+) authentication mechanism.

### Mechanism

```
Old: Phone has fixed cert (LSC) -> CUCM validates -> registers
New: Phone authenticates user via OAuth -> CUCM issues token -> phone uses token

Components:
  - Cisco Identity Service (Tomcat/IdS) running on CUCM
  - SAML SP for IdP (Microsoft Entra/AD/Azure AD/Okta/Ping/etc.)
  - JWT (JSON Web Token) returned to phone

Workflow:
  1. Phone presents SAML assertion to IdS
  2. IdS validates assertion via IdP
  3. IdS issues JWT token to phone
  4. Phone presents JWT to CUCM via SIP
  5. CUCM validates JWT and registers phone

Replaces: CTL/ITL trust model (removed for SIP-OAuth phones)
Benefit: simpler cert management, integrates with corporate identity
Requirement: CUCM 12.5+, IdP integration, phones support SIP-OAuth
```

### Configuring SIP-OAuth

```
1. Activate IdS on CUCM (Tomcat-based)
2. Configure SAML SSO with corporate IdP
3. Enable SIP-OAuth on phones via Phone Profile
4. Configure End User to use SAML
5. Phone re-registers using OAuth
```

## Provisioning Workflow

### Boot Sequence

```
1. Phone boots
2. Phone requests DHCP IP address
   - DHCP request sent
   - DHCP response includes:
     * IP address, subnet mask, gateway
     * Option 150 (Cisco TFTP) — preferred
     * Option 66 (generic TFTP) — fallback
     * Option 159 (alternate)
     * Option 160 (some Cisco models)
3. Phone connects to TFTP (UDP 69)
4. Phone requests SEP<MAC>.cnf.xml
5. Phone requests CTL.tlv (security trust)
6. Phone requests ITL.tlv (initial trust)
7. Phone parses config: which firmware to load
8. Phone requests cmterm-X.X.X.zip (firmware)
9. Phone applies firmware, reboots if needed
10. Phone connects to CUCM via SCCP/SIP
11. Phone authenticates (SCCP register, SIP REGISTER)
12. CUCM responds: capabilities, lines, services
13. Phone is in "Registered" state
14. Phone displays: line label + DN
```

### Detail: Option 150 vs 66

```
Option 150 (Cisco-specific): list of TFTP server IPs (binary IPv4 array)
Option 66  (Cisco/generic):  TFTP server hostname (string, FQDN)

Cisco prefers 150 for clusters with multiple TFTP servers (load distribution)
Generic SIP phones use 66 for single TFTP server
Cisco phones will FAIL with Option 66 on some models — must be 150
```

## Webex Calling Provisioning

Different model: cloud-based, no on-prem TFTP/CUCM.

### Account-Bound Provisioning

```
Phone factory-default
User in Webex admin portal: assign device to user
User receives email/SMS with: 
  - QR code (16-character activation code in QR)
  - Direct activation code
  - Pre-configured device IDs

Phone steps:
1. Power on
2. Phone shows "Activate Device" screen
3. Enter 16-char activation code OR scan QR
4. Phone connects to: webexcalling.cisco.com (cloud)
5. Cloud validates code -> downloads config + firmware
6. Phone registers with Webex Calling cloud
7. Phone is associated with user account
```

### QR Code Activation

```
QR contains: https://webexcalling.cisco.com/activate?code=XXXXXXXXXXXXXXXX
Phone has QR scanner (camera + image processing)
Faster than typing 16 characters
Available on: Webex Desk Pro, Webex Board, MPP firmware phones
```

### Activation Code

```
16-char code: XXXXXXXXXXXXXXXX (alphanumeric)
Generated per device assignment in Webex Control Hub
Type into phone keypad on activation screen
Code expires after 30 days
```

## Webex Calling vs CUCM

### Comparison

```
Aspect              Webex Calling                  CUCM
======              =============                  ====
Deployment          SaaS (Cisco cloud)             On-premises (cluster)
Hardware            Cisco-managed                  Customer-managed
Provisioning        Cloud activation               TFTP + CTL/ITL
Phone protocol      SIP (over TLS, secure-by-default) SIP/SCCP
Provisioning model  Cloud-bound                    On-prem cluster-bound
Firmware            Webex Calling firmware         CUCM firmware (separate)
Cost model          Per-user subscription          Hardware + license capex
Scaling             Elastic (cloud)                Cluster sizing
Network             Internet-based                 LAN-based + WAN
Update model        Continuous (cloud-updated)     Manual upgrades
Security            Webex cloud secured            Mixed-mode + LSC + ITL
Compliance          Cisco compliance attestations  Customer compliance
PSTN                Bring-your-own ITSP            CUBE / SIP gateway
Integration         Microsoft/Slack/Webex-native   Customer Java/REST APIs
Voicemail           Webex Voicemail (cloud)        CUC (Unity Connection)
Roaming             User profile follows           Extension Mobility
```

### Hybrid Coexistence

```
Webex Calling integration with CUCM possible (Webex Hybrid Calling)
Webex Edge Connect (private connection to Webex cloud)
Webex Edge for Devices (cloud-managed CUCM-registered phones)
SIP trunking between CUCM and Webex Calling for failover
```

## Webex Desk Pro / Board

### RoomOS Operating System

```
Cisco's collaboration OS
Derived from TC OS (Tandberg Collaboration)
Runs on: Webex Desk Pro, Webex Board, Webex Room Kit, etc.
Linux-based with Cisco-customized middleware
Updates: continuous via cloud
Local config: limited; mostly Control Hub-managed
```

### Device Capabilities

```
Voice: SIP (CUCM, Webex Calling, third-party)
Video: H.264, H.265 (HEVC), VP8, VP9, AV1 (newer)
Audio: G.722, Opus, AAC-LD
Display: 23"-85" 4K touchscreen
Camera: built-in HD/4K
Whiteboard: native digital whiteboarding
Calendar: integrates Microsoft 365, Google Calendar, etc.
Application: Cisco Webex collaboration suite
```

### Provisioning via Webex Control Hub

```
Webex admin -> Devices -> Add Device
  Device Type: Webex Desk Pro / Board / Room Kit
  Account assignment: per-user or per-room
  Provisioning method: 
    - QR code (activation)
    - Token/code (activation)
    - Direct cloud claim (existing fleet)
    - CUCM coexistence (registered to CUCM)
```

### Phone-side Setup

```
1. Power on Desk Pro / Board
2. RoomOS boot screen
3. "Sign in to Webex" or "Enter activation code"
4. Enter 16-char code OR scan QR
5. Device claims itself in Webex Control Hub
6. Cloud pushes RoomOS profile + apps
7. Device is operational
```

## Phone-side Diagnostics

### Settings → Status → Statistics

```
Phone navigation
  Settings (gear icon)
    Status
      System Status — uptime, time since last call
      Network Statistics — DHCP, DNS, TFTP, CUCM IPs
      Network Performance — latency, jitter, packet loss
      Active Call Statistics — current call: codec, bandwidth, MOS
      System Statistics — CPU, memory usage
      Battery Status — for wireless models
      Hardware Information — model, serial, MAC
```

### Web UI → Streaming Statistics

```
http(s)://<phone-ip>/StreamingStatistics
  Per-Stream:
    Codec (G.711μ, G.722, G.729, etc.)
    Sample Size, RTP Type
    Tx Frames, Rx Frames
    Jitter (worst case, average)
    Packet Loss %
    Round-trip latency
    DSCP/CoS values
    SRTP active/inactive
  Useful during call to validate QoS
```

### Settings → Status → Network Configuration

```
DHCP Enabled: true / false
IPv4 Address: 192.168.1.100
Subnet Mask: 255.255.255.0
Default Router: 192.168.1.1
Domain Name: corp.example.com
DNS Server 1: 192.168.1.10
TFTP Server 1: 192.168.1.20  (Option 150 result)
TFTP Server 2: 192.168.1.21  (failover)
VLAN ID: 10 (voice VLAN, from LLDP-MED)
Voice VLAN: 10
Operational VLAN: 10
LLDP Voice MED VLAN: 10
PoE Class: Class 3 (7-12W)
```

### Reset to Defaults

```bash
# 79xx series
# Power off; hold "#" while plugging in PoE; release; press 1-2-3-4-5-6-7-8-9-*-0-#

# 78xx/88xx series  
# Settings -> Admin Settings -> Reset Settings -> Factory Reset
# Or boot keypad: hold "#" + power, then 1-2-3-4-5-6-7-8-9-*-0-#

# SPA phones
# Web UI: Voice -> System -> Factory Reset = Yes -> Apply
# Or: pickup -> dial **** (4 stars) -> 73738# -> 1#

# Webex Desk Pro / Board
# Sign out first; then Settings -> About -> Factory Reset (long press)
```

## Common Errors

### "Phone is not registered"

```
Cause:    CUCM unreachable, signaling blocked
Diagnosis:
  - Check phone's TFTP server entry: Settings -> Status -> Network Config
  - Ping CUCM from phone (some firmware): Settings -> Network -> Diagnostics
  - Check CUCM Tomcat/CCM service status: CUCM admin
  - Check firewall: TCP 2000 (SCCP), UDP/TCP/TLS 5060/5061 (SIP)
Fix:      Restore CUCM connectivity; verify CUCM Group has reachable subscribers
```

### "TFTP Failed"

```
Cause:    DHCP Option 150 wrong, TFTP server unreachable, TFTP service down
Diagnosis:
  - Phone Settings -> Status -> Network Config: TFTP Server entry
  - Verify DHCP scope serves correct Option 150
  - From a workstation, tftp <tftp-ip> get SEP000000000001.cnf.xml
Fix:      Correct DHCP Option 150; restart TFTP service on CUCM
```

### "Connection Failed: 401 Unauthorized" (SIP firmware)

```
Cause:    Authentication failed at SIP server
Diagnosis:
  - SIP server logs (Asterisk asterisk -rvvv)
  - Verify Auth_ID, Password, User_ID match
  - Check SIP profile in CUCM matches phone protocol
Fix:      Correct credentials in CUCM/Asterisk; reset phone
```

### "ITL/CTL Mismatch"

```
Cause:    Phone has stale ITL after CUCM cert rotation
Diagnosis:
  - Phone shows: "Trust List Update" then "ITL Mismatch"
  - CUCM cert rotation event
Fix:      Delete ITL on phone (see ITL Recovery section)
```

### "Failed to verify license"

```
Cause:    CUCM PLM (Prime License Manager) unreachable, license expired
Diagnosis:
  - CUCM admin -> System -> Licensing -> verify license status
  - PLM connectivity from CUCM
Fix:      Renew license; restart CCMServer
```

### "Phone Load ID Not Found"

```
Cause:    Firmware archive not in TFTP root, or Phone Load Name in CUCM is wrong
Diagnosis:
  - Verify firmware exists: ls /var/tftp/ on CUCM TFTP node
  - Check Phone Configuration -> Phone Load Name
Fix:      Upload correct firmware; or clear Phone Load Name to use device default
```

### "Unable to Authenticate Phone with Server"

```
Cause:    Phone credentials don't match CUCM phone definition
Diagnosis:
  - Phone MAC matches CUCM Phone -> Device Description?
  - SCCP/SIP firmware matches CUCM Phone Configuration -> Device Protocol?
Fix:      Correct phone DN/profile; resync phone
```

### "License: Insufficient Privileges"

```
Cause:    User account permission issue
Diagnosis:
  - User -> End User -> Roles
  - Phone is associated with user
Fix:      Grant Standard CCM End User role
```

### "DHCP Failed"

```
Cause:    No DHCP response, link down, port misconfigured
Diagnosis:
  - Verify cable connectivity, port LED on phone
  - Verify switch port is enabled, in correct VLAN
  - Switch DHCP relay agent / forwarding
Fix:      Restore network connectivity; check switch port config
```

### "ENABLE Authentication Failed"

```
Cause:    802.1X authentication failed (sometimes shown on EAP-fail)
Diagnosis:
  - Check 802.1X EAP method: PEAP / EAP-TLS
  - LSC certificate valid (CAPF profile pushed?)
  - Switch RADIUS configuration
Fix:      Verify cert; check RADIUS server logs
```

## ITL/CTL Recovery

### When CUCM Cluster Security Changes

```
Common scenarios:
  - CUCM CallManager certificate rotated (annual cert renewal)
  - Cluster joined to new CA (Cert Authority)
  - Mixed Mode toggled
  - Security cert tree changed (RootCA changed)

After change, phones cannot validate new ITL signature
Phone shows: "ITL Mismatch" or "Trust List Update Failed"
Phone cannot register
```

### Manual Reset Pattern

```
1. Connect phone to laptop via Ethernet (or boot console)
2. Power on phone
3. During boot, tap Settings (gear) -> Admin Settings -> Reset Settings
4. Choose "Reset Network Settings" + "Reset Settings" both
5. Confirm reset
6. Phone rebuilds from DHCP/TFTP -> downloads new ITL
7. Re-registers with CUCM

Alternative: factory-reset code at boot (see Reset section)
```

### Automated Reset Pattern

```bash
# CUCM admin: Bulk Administration tool
Bulk Administration -> Phones -> Update Phones -> Query
  Filter: Reset Required = true OR Phone status = Unknown
  Action: Apply Config / Reset
  -> Apply to all selected phones

# CUCM admin: per-phone
Device -> Phone -> select phone -> Reset/Restart
  Reset: phone factory-default
  Restart: phone re-registers (no factory reset)
```

### Bulk ITL Distribution (CUCM 11+)

```
CUCM admin: Service Activation
  Cisco Trust Verification Service: Activate
  Cisco TFTP: Restart

Then phones automatically receive new ITL via TFTP refresh
Manual reset still needed for phones already in "ITL Mismatch" state
```

### Trust Verification Service (TVS)

```
Service running on CUCM (default port 2445)
Phone subscribes to TVS for cert change notifications
On cert rotation, TVS pushes new ITL to subscribed phones
Phone validates and accepts new ITL automatically
Avoids manual reset on each cert rotation
```

## Common Gotchas

### Phone has SIP firmware but CUCM expects SCCP (or vice versa)

```
Problem: 7942 phone shows "TFTP Configuration Error" or fails to register
Cause:   SCCP41.x.x.loads on phone, but CUCM Phone Configuration has Device Protocol = SIP
Symptom: Phone reads SEP<MAC>.cnf.xml -> sees deviceProtocol=SIP -> firmware mismatch -> error
Fix:     EITHER: change CUCM Phone Configuration to Device Protocol = SCCP
         OR: load SIP firmware on phone (SIP41.x.x.loads + manually edit XML)
```

### DHCP Option 150 (Cisco) vs Option 66 (generic) — Cisco wants 150

```
Problem: Phone gets IP via DHCP but no TFTP IP, then "TFTP Failed"
Cause:   DHCP scope serves Option 66 (DHCP server name) but not Option 150 (Cisco TFTP)
Symptom: Phone Status -> Network Config: TFTP Server = empty
Fix:     Add Option 150 to DHCP scope:
           ms-dhcp: Server Options -> 150 = TFTP server IP (binary array)
           cisco-dhcp: ip dhcp pool VOICE -> option 150 ip 192.168.1.20
           isc-dhcp: option vendor-encapsulated-options 8.13.4.192.168.1.20.255;
                     (or option-150 = 192.168.1.20)
         Or both options 150 + 66 for compatibility
```

### CTL/ITL files cached on phone don't match CUCM after CA rotation

```
Problem: After CA cert renewal, phones show "ITL Mismatch"
Cause:   Phone has old ITL cached; new ITL signed by new CA
Symptom: Phone fails registration; "Trust List Update Failed"
Fix:     1. Delete ITL on phone via factory reset OR
         2. Use Bulk Admin -> Reset Required filter -> bulk reset
         3. After reboot, phone fetches new ITL from TFTP
```

### Phone load not in TFTP directory at exact filename

```
Problem: Phone fails to download firmware
Cause:   CUCM Phone Configuration -> Phone Load Name = "SIP78XX-12-9-1-3-K9"
         but TFTP directory has "cmterm-78xx.88xx-12-9-1-3.tar.zip"
Symptom: Phone status: "Phone Load ID Not Found"
Fix:     Verify exact filename matches Phone Load Name field
         Reupload firmware archive if corrupted
         If using Device Default, ensure default points to actual file
```

### TFTP server only on UDP/69 — firewall must allow

```
Problem: Phone in DMZ can't reach CUCM TFTP behind firewall
Cause:   Firewall blocks UDP 69
Symptom: TFTP Failed, no firmware download
Fix:     Allow UDP/69 from phone subnet to CUCM TFTP IP
         Permit response back (stateful firewall handles return)
         If using TFTP failover, allow to all TFTP IPs
```

### Multi-site phones boot but voice partition (VLAN) wrong via LLDP-MED

```
Problem: Phone boots in default VLAN, can't reach CUCM (different VLAN)
Cause:   LLDP-MED voice VLAN TLV not advertised by switch
Symptom: Phone Network Config: VLAN = 1 (data VLAN); should be 10 (voice VLAN)
Fix:     Switch port config:
           interface Gi0/1
            switchport voice vlan 10
            switchport trunk native vlan 1
            lldp transmit
            lldp receive
         Or via CDP voice VLAN tagging
```

### 88xx wants Cisco-formatted XML, generic SIP profile won't work

```
Problem: 88xx phone fails to register on Asterisk/FreeSWITCH
Cause:   Phone expects SEPMAC.cnf.xml in Cisco format, not Asterisk-style flat-profile
Symptom: Phone parses XML but rejects as malformed
Fix:     Use Cisco-formatted SEPMAC.cnf.xml (see SEPMAC.cnf.xml section)
         Or load multi-platform (MPP) firmware that accepts simpler XML
         Or use chan_sccp / chan_sip with SCCP firmware on phone
```

### SPA expects flat-profile XML, won't accept Polycom/Yealink-style config

```
Problem: SPA-504G fails on cfg.cfg or sip.xml file
Cause:   SPA expects <flat-profile> root XML, not Polycom <CONFIGURATION> XML
Symptom: Provisioning shows "Resync Failed - Bad XML"
Fix:     Convert config to flat-profile XML format
         Use SPA-specific config generator (see SPA Sample Configs)
```

### "Press # to mute" but actually muted — ergonomic

```
Problem: User presses Mute button (#) thinking it ends call; line is now muted
Cause:   Mute button location varies per model
Symptom: Caller can't hear user, user thinks call ended
Fix:     User education; consider button-template change in CUCM
```

### PoE class wrong — phone fails to boot on inadequate switch port

```
Problem: 88xx phone boots, freezes, reboots loop
Cause:   PoE class 3 phone (7-12W) on switch port providing only PoE class 2 (3.84-6.49W)
Symptom: Phone boots, then reboots; LED dim
Fix:     Switch port: power inline auto class 3
         Or use external power adapter
         Newer phones (8865, video models) need PoE+ (Class 4, 30W)
```

### "Cisco Unity Connection" voicemail integration broken when MWI doesn't propagate

```
Problem: User has voicemail but no message-waiting indicator on phone
Cause:   CUCM not subscribing to MWI events from CUC
Symptom: User can dial *97 to retrieve VM but no light/icon
Fix:     CUC -> Mailbox -> MWI Notification: enable
         CUCM -> Voice Mail -> Voice Mail Pilot: configured
         Verify MWI On extension correctly mapped
```

### SRST (Survivable Remote Site Telephony) failover not configured

```
Problem: WAN outage to remote site; phones lose CUCM but local calls fail too
Cause:   No SRST router or SRST DialPeer not configured
Symptom: Phones show "Connection Failed"; can't call neighbors
Fix:     Configure SRST router at remote site (IOS-XE):
           sccp ccm 192.168.1.1 identifier 1 priority 1 version 7.0
           sccp local Loopback0
           sccp ccm group 1
           call-manager-fallback
            ip source-address 192.168.1.1 port 2000
            max-dn 24
            dialplan-pattern 1 5...... extension-pattern 1...
         CUCM Phone Configuration: Device Pool -> SRST Reference = (SRST router)
```

## SPA Sample Configs

### SPA-504G with Asterisk

```xml
<flat-profile>
  <Provision_Enable>Yes</Provision_Enable>
  <Resync_On_Reset>Yes</Resync_On_Reset>
  <Resync_Periodic>86400</Resync_Periodic>
  <Profile_Rule>https://prov.example.com/spa/$MA.xml</Profile_Rule>
  
  <Line_Enable_1_>Yes</Line_Enable_1_>
  <SIP_Transport_1_>UDP</SIP_Transport_1_>
  <Proxy_1_>asterisk.example.com</Proxy_1_>
  <Display_Name_1_>John Doe</Display_Name_1_>
  <User_ID_1_>1001</User_ID_1_>
  <Password_1_>secret</Password_1_>
  <Auth_ID_1_>1001</Auth_ID_1_>
  <Use_Auth_ID_1_>Yes</Use_Auth_ID_1_>
  <Register_1_>Yes</Register_1_>
  <Preferred_Codec_1_>G711u</Preferred_Codec_1_>
  <Codec_Negotiation_1_>List All</Codec_Negotiation_1_>
  <DTMF_Tx_Method_1_>RTP-NTE</DTMF_Tx_Method_1_>
  
  <Line_Enable_2_>Yes</Line_Enable_2_>
  <Proxy_2_>asterisk.example.com</Proxy_2_>
  <User_ID_2_>1002</User_ID_2_>
  <Password_2_>secret2</Password_2_>
  <Auth_ID_2_>1002</Auth_ID_2_>
  
  <Line_Enable_3_>No</Line_Enable_3_>
  <Line_Enable_4_>No</Line_Enable_4_>
  
  <Line_Key_1_>fnc=sd;ext=1003;nme=Reception</Line_Key_1_>
  <Line_Key_2_>fnc=blf;sub=1003@asterisk.example.com;nme=Reception BLF</Line_Key_2_>
</flat-profile>
```

### SPA-504G with FreeSWITCH

```xml
<flat-profile>
  <Provision_Enable>Yes</Provision_Enable>
  <Resync_Periodic>86400</Resync_Periodic>
  <Profile_Rule>https://prov.example.com/spa/$MA.xml</Profile_Rule>
  
  <Line_Enable_1_>Yes</Line_Enable_1_>
  <SIP_Transport_1_>UDP</SIP_Transport_1_>
  <Proxy_1_>fs.example.com</Proxy_1_>
  <Outbound_Proxy_1_>edge.example.com</Outbound_Proxy_1_>
  <Use_Outbound_Proxy_1_>Yes</Use_Outbound_Proxy_1_>
  <SIP_Port_1_>5060</SIP_Port_1_>
  <Display_Name_1_>John Doe</Display_Name_1_>
  <User_ID_1_>1001</User_ID_1_>
  <Password_1_>secretkey</Password_1_>
  <Auth_ID_1_>1001</Auth_ID_1_>
  <Use_Auth_ID_1_>Yes</Use_Auth_ID_1_>
  <Register_1_>Yes</Register_1_>
  <Register_Expires_1_>3600</Register_Expires_1_>
  <Preferred_Codec_1_>G711u</Preferred_Codec_1_>
  <Use_Pref_Codec_Only_1_>No</Use_Pref_Codec_Only_1_>
  <Codec_Negotiation_1_>List All</Codec_Negotiation_1_>
  <Codec_1_Enable>Yes</Codec_1_Enable>
  <DTMF_Tx_Method_1_>RTP-NTE</DTMF_Tx_Method_1_>
  <Voice_Quality_Report_Address_1_>vqr.example.com</Voice_Quality_Report_Address_1_>
  
  <Line_Enable_2_>Yes</Line_Enable_2_>
  <Proxy_2_>fs.example.com</Proxy_2_>
  <User_ID_2_>1001</User_ID_2_>
  <Password_2_>secretkey</Password_2_>
  
  <NAT_Mapping_Enable>Yes</NAT_Mapping_Enable>
  <NAT_Keepalive_Enable>Yes</NAT_Keepalive_Enable>
  <NAT_Keepalive_Msg>$NOTIFY</NAT_Keepalive_Msg>
  <Substitute_VIA_Addr>Yes</Substitute_VIA_Addr>
  <STUN_Enable>Yes</STUN_Enable>
  <STUN_Test_Enable>Yes</STUN_Test_Enable>
  <STUN_Server>stun.example.com</STUN_Server>
</flat-profile>
```

## 88xx Sample Configs

### CUCM Phone Configuration (8845)

```xml
<!-- SEP00112233445566.cnf.xml for an 8845 -->
<device>
  <fullConfig>true</fullConfig>
  <deviceProtocol>SIP</deviceProtocol>
  <devicePool>
    <name>Default</name>
    <dateTimeSetting>
      <name>CMLocal</name>
      <timeZone>America/New_York</timeZone>
    </dateTimeSetting>
    <callManagerGroup>
      <members>
        <member priority="0">
          <callManager>
            <name>CCMSUB1</name>
            <ports>
              <ethernetPhonePort>2000</ethernetPhonePort>
              <sipPort>5060</sipPort>
              <securedSipPort>5061</securedSipPort>
            </ports>
            <processNodeName>10.1.1.10</processNodeName>
          </callManager>
        </member>
        <member priority="1">
          <callManager>
            <name>CCMSUB2</name>
            <ports>
              <ethernetPhonePort>2000</ethernetPhonePort>
              <sipPort>5060</sipPort>
              <securedSipPort>5061</securedSipPort>
            </ports>
            <processNodeName>10.1.1.11</processNodeName>
          </callManager>
        </member>
      </members>
    </callManagerGroup>
  </devicePool>
  <commonProfile>
    <phonePassword>cisco</phonePassword>
    <backgroundImageAccess>true</backgroundImageAccess>
  </commonProfile>
  <ipDtmfMode>1</ipDtmfMode>
  <preferredCodec>g722</preferredCodec>
  <vad>false</vad>
  <sshUserId>cisco</sshUserId>
  <sshPassword>cisco</sshPassword>
  <httpAccessEnabled>true</httpAccessEnabled>
  <webAccess>1</webAccess>
  <sipDirectoryNumberConfig>
    <line>
      <featureID>9</featureID>
      <featureLabel>1001</featureLabel>
      <featureKey>1</featureKey>
      <name>1001</name>
      <displayName>John Doe</displayName>
      <contact>1001@cucm.example.com</contact>
      <subscribeCallingSearchSpaceName>Standard_CSS</subscribeCallingSearchSpaceName>
      <authName>1001</authName>
      <authPassword>secretkey</authPassword>
      <ringSettingIdle>0</ringSettingIdle>
      <ringSettingActive>0</ringSettingActive>
    </line>
    <line>
      <featureID>9</featureID>
      <featureLabel>1002</featureLabel>
      <featureKey>2</featureKey>
      <name>1002</name>
      <contact>1002@cucm.example.com</contact>
      <authName>1002</authName>
      <authPassword>secretkey2</authPassword>
    </line>
    <line>
      <featureID>21</featureID>  <!-- BLF Speed Dial -->
      <featureLabel>Reception</featureLabel>
      <featureKey>3</featureKey>
      <directoryNumber>1003</directoryNumber>
    </line>
  </sipDirectoryNumberConfig>
  <advancedSecurity>
    <signalingEncryption>0</signalingEncryption>
    <transportProtocol>0</transportProtocol>
  </advancedSecurity>
  <encryption>
    <serverCertSourceFile>1</serverCertSourceFile>
  </encryption>
  <networkParms>
    <vlan>10</vlan>
    <dscpForCalls>46</dscpForCalls>
    <dscpForSig>26</dscpForSig>
    <ipv6CfgMode>0</ipv6CfgMode>
  </networkParms>
  <loadInformation>SIP8845-14-1-1SR3-1</loadInformation>
  <vendorConfig>
    <recoveryURL>https://10.1.1.10/cmm</recoveryURL>
  </vendorConfig>
</device>
```

## CUBE Configuration Snippet

### SIP Trunk to ITSP

```cisco
! Global voice settings
voice service voip
 ip address trusted list
  ipv4 10.0.0.0 255.0.0.0
  ipv4 1.2.3.0 255.255.255.0
 allow-connections sip to sip
 sip
  bind control source-interface GigabitEthernet0/0
  bind media source-interface GigabitEthernet0/0
  early-offer forced
  registrar server expires max 3600 min 60

! SIP UA
sip-ua
 retry invite 3
 retry register 5
 timers connect 1000
 timers retry 200 200
 transport tcp tls v1.2
 sip-server ipv4:1.2.3.4:5060

! Voice class for ITSP
voice class tenant 1
 sip-server ipv4:1.2.3.4:5060
 transport udp
 outbound-proxy ipv4:1.2.3.4:5060
 srtp-crypto 1
 session transport udp

! Codec list
voice class codec 1
 codec preference 1 g711ulaw
 codec preference 2 g729r8

! Inbound from ITSP
dial-peer voice 100 voip
 description "ITSP Inbound"
 destination-pattern .T
 session protocol sipv2
 session target sip-server
 voice-class codec 1
 voice-class tenant 1
 incoming called-number .T
 dtmf-relay rtp-nte sip-kpml
 no vad

! Outbound to CUCM
dial-peer voice 200 voip
 description "CUCM"
 destination-pattern 5...
 session protocol sipv2
 session target ipv4:10.1.1.10:5060
 voice-class codec 1
 dtmf-relay rtp-nte sip-kpml
 no vad

! Translation rules (e.g., add 1 prefix to outbound 10-digit)
voice translation-rule 100
 rule 1 /^([2-9]\d\d)([2-9]\d\d)(\d\d\d\d)$/ /1\1\2\3/

voice translation-profile NA-OUT
 translate calling 100

! Apply translation to dial-peer
dial-peer voice 200 voip
 translation-profile outgoing NA-OUT
```

## Diagnostic Tools

### CUCM RTMT (Real-Time Monitoring Tool)

```
Java application bundled with CUCM
Connects to: CUCM Publisher + Subscribers
Capabilities:
  - Real-time call monitoring
  - Performance counters (CallManager, IPVM, CTI)
  - Trace log collection (per-service, per-server)
  - Alerts (threshold-based: CPU, memory, calls)
  - Event reports (call admission, registration)
  - SoapNotify trap collection
Download: CUCM admin -> Application -> Plugins -> Cisco Unified Real-Time Monitoring Tool

Common workflows:
  - View all phones, filter by registration status
  - Trace SIP signaling from phone to CUCM
  - Pull CCM trace logs for failed call
```

### Phone Web UI Stream Statistics

```
http(s)://<phone-ip>/StreamingStatistics?STREAM=1
Real-time:
  Codec, sample size
  Tx/Rx packets, bytes
  Jitter (max, avg)
  Packet loss
  Round-trip latency
Per active call only
```

### CUCM Trace Logs

```
File-based trace per service
  Cisco CallManager: /var/log/active/cm/trace/ccm/sdl
  Cisco TFTP: /var/log/active/cm/trace/tftp
  Cisco Tomcat: /var/log/active/tomcat/log
  Cisco TVS: /var/log/active/cm/trace/tvs
  
Trace levels:
  Detailed: maximum verbosity
  State Transition: state changes
  Significant: errors and warnings only
  Error: errors only
  Special: rare events

Adjust via: CUCM Serviceability -> Trace -> Configuration
```

### CTL Client

```
Tool for managing CTL files in CUCM mixed-mode
Cisco CTL Client (CCTL): software used by admin to:
  - Generate CTL with USB eToken
  - Update CTL after cluster cert rotation
  - Migrate to Tokenless CTL (CUCM 11+)
  
Replaced by Tokenless CTL (CTL Client not needed in CUCM 11+)
```

### PRT (Problem Report Tool)

```
Modern phones (78xx/88xx/MPP) have built-in problem report tool
Workflow:
  1. Phone Settings -> Problem Report -> Submit
  2. Phone collects: call logs, system logs, screen captures
  3. Phone uploads to: PRT URL (configured in CUCM Phone Configuration)
  4. Admin downloads PRT archive from PRT server (typically a TFTP/HTTPS server)

Configure PRT URL in CUCM:
  Device -> Phone Configuration -> 
    Problem Report Tool URL: http://prt-server.example.com/prt/upload
  
PRT contains: phone state, network state, call statistics, screen captures
```

## Idioms

### "DHCP Option 150 not 66 for Cisco"

```
For Cisco IP phones, use DHCP Option 150 (Cisco-specific TFTP IP array)
Generic SIP phones use Option 66 (TFTP server hostname string)
Cisco phones can fail with Option 66 only — must serve Option 150
Best practice: serve both (150 + 66) for compatibility
```

### "Load SIP firmware on 79xx for non-Cisco PBX"

```
Default 79xx firmware: SCCP (Cisco proprietary, only with CUCM)
To use 79xx with Asterisk/FreeSWITCH/Yealink/Polycom mixed env:
  Place SIP41.x.loads in TFTP root
  Edit SEPMAC.cnf.xml: <loadInformation>SIP41.x</loadInformation>
  Phone reboots, downloads SIP firmware, swaps in
```

### "ITL deletion is the cure for the cert-rotation cliff"

```
After CUCM cert rotation, phones cached old ITL fail to validate
Manual cure: factory-reset phone -> rebuilds ITL from new cert chain
Best practice: enable Trust Verification Service (TVS) -> auto-update ITL
```

### "Use Cisco IP Communicator for soft client"

```
Cisco IP Communicator (CIPC) — software phone for Windows/Mac
Registers with CUCM same as hardware phone
SCCP or SIP firmware (newer is SIP)
Useful for: remote workers, soft phones, testing
End-of-life since CUCM 12 (replaced by Webex)
```

### "Always allow TFTP UDP 69 in firewall paths"

```
Phone -> TFTP communication uses UDP/69
Stateful firewall must allow:
  - Outbound from phone subnet to TFTP IP, port 69 UDP
  - Return traffic from TFTP IP back to phone (handled stateful)
  - For multi-CUCM cluster, allow to all TFTP node IPs
Common firewall mistake: blocking UDP/69 -> phone "TFTP Failed"
```

## See Also

- ip-phone-provisioning
- sip-protocol
- asterisk
- freeswitch
- yealink-phones
- polycom-phones
- grandstream-phones
- snom-phones

## References

- Cisco IP Phone 8800 Series Administration Guide for Cisco Unified Communications Manager — https://www.cisco.com/c/en/us/td/docs/voice_ip_comm/cuipph/8800-series/english/admin/12_8/8800_AdminGuide_CUCM.html
- Cisco IP Phone 7800 Series Administration Guide — https://www.cisco.com/c/en/us/td/docs/voice_ip_comm/cuipph/7800-series/english/admin/12_8/7800_AdminGuide_CUCM.html
- Cisco SPA9xx Configuration Guide — https://www.cisco.com/c/en/us/td/docs/voice_ip_comm/csbpvga/SPA9x/admin/spa9x_AG_book.html
- Cisco SPA Provisioning Guide (XML profile format) — https://www.cisco.com/c/en/us/td/docs/voice_ip_comm/csbpvga/spa-provguide.html
- Cisco Unified Communications Manager (CUCM) Solution Reference Network Design (SRND) — https://www.cisco.com/c/en/us/td/docs/voice_ip_comm/cucm/srnd/collab12/collab12.html
- Cisco Unified Border Element (CUBE) Configuration Guide — https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/voice/cube/configuration/cube-book.html
- Cisco Webex Calling Administration Guide — https://help.webex.com/en-US/article/n2ebzxe/Cisco-Webex-Calling-Administration-Guide
- Cisco Webex Desk Pro Administration Guide — https://www.cisco.com/c/en/us/support/collaboration-endpoints/webex-desk-pro/products-installation-and-configuration-guides-list.html
- Cisco Webex Board Administration Guide — https://www.cisco.com/c/en/us/support/collaboration-endpoints/webex-board-series/products-installation-and-configuration-guides-list.html
- Cisco Multiplatform (MPP) Phone Administration Guide — https://www.cisco.com/c/en/us/td/docs/voice_ip_comm/cuipph/MPP/MPP_Admin_Guide.html
- RFC 3261 (SIP) — https://www.rfc-editor.org/rfc/rfc3261
- RFC 4566 (SDP) — https://www.rfc-editor.org/rfc/rfc4566
- RFC 3550 (RTP) — https://www.rfc-editor.org/rfc/rfc3550
- RFC 3711 (SRTP) — https://www.rfc-editor.org/rfc/rfc3711
- RFC 4733 (RTP DTMF / RTP-NTE) — https://www.rfc-editor.org/rfc/rfc4733
- Cisco SCCP Protocol Reference — https://developer.cisco.com/docs/sccp
- Cisco RoomOS Administrator Guide — https://www.cisco.com/c/en/us/support/collaboration-endpoints/roomos/products-maintenance-guides-list.html
- Cisco CUCM Security Guide (CTL/ITL/LSC) — https://www.cisco.com/c/en/us/td/docs/voice_ip_comm/cucm/security/12_5_1/cucm_b_security-guide-1251.html
- Cisco SIP-OAuth Configuration Guide — https://www.cisco.com/c/en/us/td/docs/voice_ip_comm/cucm/admin/12_5_1/cucm_b_administration-guide-1251.html
- Cisco SRST Configuration Guide — https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/voice/srst/configuration/srst-book.html
- Cisco Real-Time Monitoring Tool (RTMT) Administration Guide — https://www.cisco.com/c/en/us/td/docs/voice_ip_comm/cucm/service/12_5_1/rtmt/cucm_b_cisco-unified-rtmt-administration-1251.html
- Cisco IP Phone Provisioning Models (CUCM, MPP, Webex) — https://www.cisco.com/c/en/us/products/collaboration-endpoints/index.html
