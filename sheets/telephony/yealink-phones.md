# Yealink IP Phones

Configuration, provisioning, and operation reference for Yealink T/W/VP/MP series SIP endpoints — web UI, config keys, RPS, DHCP options, BLF, dial plans, troubleshooting.

## Setup

Yealink portfolio breakdown:

- **T-series** — desk phones, the bulk of deployments.
  - **T2x** — entry tier, 100Mbps Ethernet, monochrome LCD. T21P, T23G, T27G.
  - **T3x** — popular SMB tier. T31G, T33G, T34W, T31P (no PoE on -P? PoE on -P, no PoE on plain). 1Gbps + PoE on G/W.
  - **T4x** — mid-range business, color screens. T42G, T42S, T42U, T43U, T44W, T46U, T46S, T46G, T48U, T48S, T48G.
  - **T5x** — top-tier exec. T53W, T54W, T57W, T58W, T58A. Bluetooth, WiFi, USB camera headers.
- **W-series** — DECT cordless and WiFi.
  - **W56** — handset (W56H) + base (W60B/W56P). Older 6-line DECT.
  - **W60** — base only (W60B), pairs with W56H/W53H.
  - **W70** — newer multi-cell DECT base, replaces W60.
  - **W80** — multi-cell DECT system (W80B base + W80DM manager).
  - **W90** — large-scale multi-cell DECT for hotels/warehouses (W90B/W90DM).
  - **WiFi handsets** — W76P (base), W79P, W57R (rugged).
- **VP-series** — video desk phones with built-in camera. VP59 is the surviving SKU.
- **MeetingBoard** — Android-based all-in-one conference appliances. MeetingBoard 65/86, MeetingBar A20/A30 (camera bars).
- **MP-series** — Microsoft Teams (MP54, MP56, MP58) and audio gateway.
- **CP-series** — conference IP phones (CP920, CP925, CP935W, CP965).
- **EXP** — sidecar expansion modules: EXP43, EXP50.
- **EHS** — electronic hook switch adapters: EHS35, EHS36, EHS40, EHS60, EHS61.

Firmware naming convention `X.X.X.X.rom`:

- Format: `<HardwareID>.<MajorBranch>.<Minor>.<Build>.rom`
- Example: `54.85.0.5.rom` — hardware 54 (T54W), branch 85, version 0.5.
- Example: `66.86.0.30.rom` — hardware 66 (T46U), branch 86, version 0.30.
- The first number identifies hardware family — never load a `54` ROM onto a `66` device or it will brick.
- `.cfg` files are configuration; `.rom` is firmware; `.boot` is bootloader (rare).
- Yealink hardware IDs (selected): T31G=124, T33G=147, T42G=29, T46G=28, T46U=108, T48G=35, T48U=109, T54W=96, T57W=97, T58W=86, W60B=77, W70B=146, W80B=103, MP54=132, MP56=133, VP59=91.

## Default Access

- **Web UI** — `http://<phone-ip>` (port 80) or `https://<phone-ip>` (port 443).
  - Find IP from handset: **Menu → Status → IPv4** (or press the OK button on idle for some models).
- **Default credentials** — `admin / admin`. **Always change in production.**
- **Var-tier login** — `var / var` (variable level, restricted UI; available when the manufacturer enables it).
- **User-tier login** — `user / user` on some models.
- **SSH** — disabled by default; not in standard firmware. Requires unlocked / debug firmware (rare; only for support cases). Port 22 once enabled.
- **Telnet** — disabled. Use SSH if available.
- **Factory reset** — three options:
  1. **Web UI** — *Settings → Upgrade → Reset to Factory*.
  2. **Handset menu** — *Menu → Advanced (admin/admin) → Reset to Factory Settings*.
  3. **Key combo** — long-press `*` and `#` on idle for ~10 seconds while booting (T-series). On W-series, the combo is on the base or via handset menu.
- **Boot key combos** — hold `*` during power-on to enter recovery; `#` for network setup on some W-series.

## Web UI Tour

Top-level page tabs (varies subtly by model/firmware, but the canonical layout is):

- **Status** — registration, network, MAC, firmware version, uptime, account status.
- **Account** — per-account SIP registration: Register, Codec, Advanced.
  - Sub-tabs: Register, Basic, Codec, Advanced.
- **Network** — IP, VLAN, QoS, VPN, Web Server, 802.1X, NAT.
  - Sub-tabs: Basic, Wi-Fi, VPN, Web Server, Port, NAT, Advanced, PC Port.
- **Features** — DSS keys, transfer, conference, intercom, call waiting, DND, BLF, MWI.
  - Sub-tabs: Forward & DND, General, Audio, Intercom, Transfer, Call Pickup, Remote Control, Phone Lock, ACD, SMS, Action URL, Power LED, Notification Popups.
- **Settings** — date & time, ringtone, autoprovisioning, configuration, upgrade, tones.
  - Sub-tabs: Preference, Time & Date, Call Display, Upgrade, Auto Provision, Configuration, Dial Plan, Voice, Ring, Tones, SoftKey Layout, TR069, Voice Monitoring, FWD International.
- **Directory** — local, remote phonebook, LDAP, BroadSoft, blacklist.
- **Security** — passwords, certificates, trusted certificates, server certificates.
  - Sub-tabs: Password, Trusted Certificates, Server Certificates.
- **Phone (UI)** / **DSSKey** — line key, programmable key, ext key, soft key layout.
- **Programs** / **Apps** — for Android-based MeetingBoard / Teams models.
- **Maintenance** — PCAP, System Log, Configuration export, Diagnostic.

## Account Configuration

Yealink supports `account.1` through `account.16` (model-dependent — T2x = 2 lines, T3x = 4-12, T46U = 16, T48U/T57W = 16). Each account is a SIP registration.

Key fields (config-file syntax, also reflected in the web UI under *Account → Register*):

```ini
account.1.enable = 1
account.1.label = "Reception"
account.1.display_name = "Reception"
account.1.auth_name = "100"
account.1.user_name = "100"
account.1.password = "secret"
account.1.sip_server.1.address = "pbx.example.com"
account.1.sip_server.1.port = 5060
account.1.sip_server.1.transport_type = 0   # 0=UDP, 1=TCP, 2=TLS, 3=DNS-NAPTR
account.1.sip_server.1.expires = 3600
account.1.sip_server.1.retry_counter = 3
account.1.outbound_proxy_enable = 1
account.1.outbound_host = "sbc.example.com"
account.1.outbound_port = 5060
account.1.transport = 0
account.1.expires = 3600
account.1.register_mac = 0
account.1.anonymous_call = 0
account.1.anonymous_call_oban = 0
account.1.cid_source = 0
account.1.dnd.enable = 0
account.1.missed_calllog = 1
account.1.dialoginfo_callpickup = 0
account.1.shared_line = 0
account.1.shared_line_callpull_code = "*11"
account.1.cp_source = 0
```

Backup SIP server (failover):

```ini
account.1.sip_server.2.address = "pbx-backup.example.com"
account.1.sip_server.2.port = 5060
account.1.sip_server.2.transport_type = 0
account.1.sip_server.2.expires = 3600
account.1.sip_server.2.retry_counter = 3
account.1.fallback.redundancy_type = 1   # 0=concurrent, 1=failover
account.1.fallback.timeout = 32
```

Useful registration tuning:

```ini
account.1.reregister_enable = 1
account.1.subscribe_register = 0
account.1.subscribe_mwi = 1
account.1.subscribe_mwi_to_vm = 1
account.1.subscribe_mwi_expires = 3600
account.1.subscribe_acd = 0
account.1.dial_without_reg = 0
account.1.timer_t1 = 0.5
account.1.timer_t2 = 4
```

`account.1.transport` legacy alias maps to `sip_server.1.transport_type`. Stick with the new key on firmware 84+.

## Multiple Lines / Multiple Accounts

Each line key on the phone can be bound to an account. With multiple SIP accounts, you assign which account each line key uses:

```ini
account.1.enable = 1
account.1.label = "Sales"
account.2.enable = 1
account.2.label = "Support"
account.3.enable = 1
account.3.label = "Reception"

linekey.1.line = 1
linekey.1.type = 15            # Line
linekey.1.label = "Sales"
linekey.2.line = 2
linekey.2.type = 15
linekey.2.label = "Support"
linekey.3.line = 3
linekey.3.type = 15
linekey.3.label = "Reception"
```

Per-account dial plan (overrides the global one for that account):

```ini
account.1.dial_plan = "{[*#0-9]+}"
account.1.dialplan_replace.1.prefix = "9"
account.1.dialplan_replace.1.replace = ""
account.1.dialplan_replace.1.line = 1
```

`account.X.outgoing_call_logs_enable`, `account.X.missed_calllog`, `account.X.recv_message_enable` per-account log toggles.

## Dial Plan

Yealink uses a regex-flavored dial-plan grammar with two mechanisms:

1. **Replace rule** — strip/prefix digits before sending.
2. **Match rule** — accept-as-dialed.

Replace-rule keys:

```ini
dialplan.replace.prefix.1 = "9"
dialplan.replace.replace.1 = ""
dialplan.replace.line.1 = 0       # 0 = all accounts; 1..N = specific
dialplan.replace.account.1 = ""

dialplan.replace.prefix.2 = "00"
dialplan.replace.replace.2 = "+"
dialplan.replace.line.2 = 0
```

Or the older `dial_plan.X.*` style (still accepted on most firmware):

```ini
dial_plan.1.prefix = "9"
dial_plan.1.replace = ""
dial_plan.1.line = 1
dial_plan.1.type = 0    # 0 = replace, 1 = matchonly
```

Block-rule (deny pattern):

```ini
dialplan.block.prefix.1 = "1900"
dialplan.block.line.1 = 0
```

Area code / country code:

```ini
dialplan.area_code.code = "212"
dialplan.area_code.min_len = 7
dialplan.area_code.max_len = 7
dialplan.area_code.line.1 = 1
```

Pattern syntax (under *Settings → Dial Plan* in the web UI):

```text
[0-9]    digit class
.        any character
*        0+ repetitions of preceding
+        1+ repetitions of preceding
?        0 or 1
{n,m}    range of repetitions
|        alternation
xxx      literal digits
\        escape
```

E.164 outbound conversion example (strip 9, prepend country code):

```ini
dialplan.replace.prefix.1 = "9011"
dialplan.replace.replace.1 = "+"
dialplan.replace.prefix.2 = "9"
dialplan.replace.replace.2 = "+1"
```

## Codec Configuration

Codec list with priority is per-account. Higher number = lower priority (sometimes inverted by firmware; check before deploying).

```ini
account.1.codec.1.enable = 1
account.1.codec.1.payload_type = "PCMU"
account.1.codec.1.priority = 1
account.1.codec.1.rtpmap = 0

account.1.codec.2.enable = 1
account.1.codec.2.payload_type = "PCMA"
account.1.codec.2.priority = 2
account.1.codec.2.rtpmap = 8

account.1.codec.3.enable = 1
account.1.codec.3.payload_type = "G722"
account.1.codec.3.priority = 3
account.1.codec.3.rtpmap = 9

account.1.codec.4.enable = 0
account.1.codec.4.payload_type = "G729"

account.1.codec.5.enable = 0
account.1.codec.5.payload_type = "G723_53"

account.1.codec.6.enable = 0
account.1.codec.6.payload_type = "iLBC"

account.1.codec.7.enable = 1
account.1.codec.7.payload_type = "Opus"
account.1.codec.7.priority = 4
```

Supported codec strings (model-dependent):

- `PCMU` — G.711µ-law, 64 kbps, NA standard.
- `PCMA` — G.711a-law, 64 kbps, EU standard.
- `G722` — wideband HD, 64 kbps.
- `G723_53` / `G723_63` — G.723.1 5.3/6.3 kbps.
- `G729` — 8 kbps narrowband (license needed on some hardware revs).
- `iLBC` — 13.3/15.2 kbps internet low-bit codec.
- `Opus` — 6-510 kbps adaptive (firmware 84+ on most T5x/T4x-U).
- `AMR`, `AMR-WB` — mobile codec (rare).
- `L16` — 16-bit linear (test).
- `H264`, `H265`, `H263` — video codecs on VP and MeetingBoard.

Global codec disable (kill an codec across all accounts):

```ini
features.codec.G722.enable = 0
```

DTMF mode:

```ini
account.1.dtmf.type = 1                 # 0=inband, 1=RFC2833, 2=SIP-INFO, 3=Auto+SIP-INFO
account.1.dtmf.dtmf_payload = 101
account.1.dtmf.info_type = 1            # 1=DTMF-Relay, 2=DTMF, 3=Telephone-Event
```

## Programmable Keys

Three key categories on most T-series:

1. **Line keys** (`linekey.X.*`) — large keys next to display, can be re-purposed.
2. **Memory / DSS keys** (`memorykey.X.*` on older models).
3. **Programmable keys** (`programablekey.X.*`) — top-of-screen, navigation arrows, OK, X.
4. **Function/Soft keys** (`softkey.X.*`) — bottom row, context-sensitive.
5. **Ext keys** (`expansion_module.X.linekey.Y.*`) — sidecar EXP43/50 module.

Universal triplet — `type`, `value`, `label` (plus `line` for SIP-bound keys):

```ini
linekey.1.type = 15           # Line
linekey.1.value = ""
linekey.1.line = 1
linekey.1.label = "Reception"
```

Common `linekey.X.type` values:

```text
0   N/A (disabled)
1   Conference
2   Forward
3   Transfer
4   Hold
5   DND
6   Redial
7   Call Return
8   Pickup
9   Call Park
10  DTMF
11  Voice Mail
12  Speed Dial
13  Intercom
14  Line
15  Line (alt index, varies by FW)
16  BLF
17  URL
18  Group Listening
20  Private Hold
22  XML Group
23  Group Pickup
24  Multicast Paging
25  Record
27  XML Browser
30  Hot Desking
38  ACD
39  Zero Touch
45  Local Group
49  BLF List
50  URL Record
55  Meet-me Conference
56  Retrieve Park
59  Hoteling
60  ACD Grace
61  SISP code
62  Emergency
63  Directory
73  Paging List
80  Network Favorites
81  XML Phonebook
82  XML Browser
83  Forward All Lines
84  GPickup
85  XML Browser
```

Speed dial:

```ini
linekey.5.type = 13
linekey.5.value = "1001"
linekey.5.line = 1
linekey.5.label = "Helpdesk"
```

DTMF key (sends digit string mid-call):

```ini
linekey.6.type = 10
linekey.6.value = "*98"
linekey.6.label = "VM"
```

XML browser:

```ini
linekey.10.type = 27
linekey.10.value = "http://intranet/xml/menu.xml"
linekey.10.label = "Apps"
```

Group paging / multicast paging:

```ini
linekey.7.type = 24
linekey.7.value = "224.5.6.20:10000"
linekey.7.label = "All Page"
multicast.codec = "PCMU"
multicast.listen_address.1.ip_address = "224.5.6.20:10000"
multicast.listen_address.1.label = "All Page"
multicast.listen_address.1.priority = 1
```

Recording:

```ini
linekey.11.type = 25
linekey.11.value = "*99"        # PBX feature code
linekey.11.label = "Record"
features.config_call_record_button.enable = 1
```

Programmable keys (above-display):

```ini
programablekey.1.type = 28      # History
programablekey.1.label = "History"
programablekey.2.type = 30      # Directory
programablekey.3.type = 22      # DND
programablekey.4.type = 28      # Soft key reposition
```

## BLF (Busy Lamp Field)

BLF lets a key show another extension's hook state and one-press dial/pick-up.

Server side: PBX must publish `dialog` event package (Asterisk: `hint`; FreeSWITCH: `presence_in/out`; FreePBX: BLF auto-on hint mod).

Phone side:

```ini
linekey.20.type = 16
linekey.20.value = "1001"
linekey.20.line = 1
linekey.20.label = "Alice"
linekey.20.extension = "1001@pbx.example.com"
linekey.20.pickup_value = "*8"
```

The phone sends a SIP `SUBSCRIBE` for `Event: dialog` (or `presence` if configured) to `1001@pbx.example.com`.

Subscription tuning:

```ini
account.1.blf.subscribe_period = 1800
account.1.blf_list_uri = "blf-list@pbx.example.com"
account.1.blf_list_pickup_code = "**"
account.1.blf_list_barge_in_code = "*81"
phone_setting.blf_list_pickup_code = "**"
features.blf.led_mode = 1
features.blf.dialog_idle_blink = 0
```

Indicator color/pattern table (typical T-series RGB LED):

```text
State          | LED Color | Pattern
---------------|-----------|----------------
Idle           | Off       | (off)
Ringing        | Red       | Fast blink
Talking        | Red       | Solid
DND            | Red       | Slow blink
Hold           | Red       | Slow blink (alt firmware)
Failed/sub-err | Yellow    | Slow blink
Subscribed     | Green     | Solid (some FW)
```

`features.blf.led_mode` values change which states light up; mode 1 = standard, 2 = ringing-only, 3 = busy-only.

BLF List (server-side hint list, fewer SUBSCRIBEs):

```ini
account.1.blf_list_uri = "buddies@pbx"
account.1.blf_list_code = "*8"
linekey.21.type = 49     # BLF List
linekey.21.line = 1
```

## BLA / SCA — Bridged / Shared Line Appearance

Multiple phones share one extension; pick-up on one shows others as in-use.

```ini
account.1.shared_line = 2          # 0=disabled, 1=BLA, 2=SCA
account.1.shared_call_appearance = 1
account.1.number_of_calls_per_line_key = 2
account.1.shared_line_callpull_code = "*11"
sip.shared_appearance.barge_in_enable = 1

linekey.1.line = 1
linekey.1.type = 15
```

Server (BroadWorks / Asterisk app_sla / FreeSWITCH mod_sla) must understand `Event: dialog;sla` and the `BLA` AOR group. SCA is the BroadSoft variant; BLA is the older RFC.

Full SCA / BLA parameter set:

```ini
account.1.shared_line = 2
account.1.shared_call_appearance = 1
account.1.number_of_calls_per_line_key = 2
account.1.shared_line_callpull_code = "*11"
account.1.bla_number = "100"
account.1.bla_subscribe_period = 300
account.1.outbound_proxy_enable = 1
account.1.bla_subscribe_event = "dialog"
account.1.shared_line_alert_tone_enable = 1
account.1.shared_line.public_hold_priv_listen.enable = 0
account.1.shared_line.barge_in_enable = 1
sip.shared_appearance.barge_in_enable = 1
sip.shared_appearance.silent_barge_in.enable = 0
sip.shared_appearance.privacy.enable = 1
sip.shared_appearance.public_hold.led_enable = 1
sip.shared_appearance.private_hold.led_enable = 1
features.line_seize.timeout = 15
features.shared_line.led_idle_color = 0
features.shared_line.led_seize_color = 1
features.shared_line.led_progress_color = 1
features.shared_line.led_active_color = 1
features.shared_line.led_held_color = 1
features.shared_line.led_held_private_color = 1
```

Server-side examples:

```text
# BroadWorks SCA: configure "Shared Call Appearance" service and add phone device
# as a "Shared Call Appearance Location" with a unique linePort suffix.

# Asterisk app_sla (chan_pjsip example):
[100-bridge]
type=sla_bridge
device=100

[100-sla]
type=sla_station
device=PJSIP/100
trunk=100-bridge

# FreeSWITCH mod_sla:
<settings>
  <param name="bridge-domain" value="100@example.com"/>
</settings>
```

Useful behaviour matrix:

```text
Action          Local LED        Remote LED       Notes
----------------|-----------------|-----------------|-----------------
Idle            Off              Off              Both phones available
Seize line      Green-fast       Red-solid        Local user got dial tone
Active call     Green-solid      Red-solid        Cannot barge unless allowed
Public hold     Green-blink      Red-blink        Anyone can pick up
Private hold    Green-slow       Red-solid        Only originator resumes
Bridge in       Green-solid      Green-solid      Both parties on call
```

Capacity limits — BroadWorks default is 35 SCA endpoints per AOR; Asterisk app_sla recommends ≤8; FreeSWITCH mod_sla scales to ~16 cleanly. Beyond that, NOTIFY storms become noticeable.

Sample full per-MAC.cfg for a 5-line SCA reception:

```ini
#!version:1.0.0.1
account.1.enable = 1
account.1.label = "Reception"
account.1.user_name = "100"
account.1.auth_name = "100"
account.1.password = "redacted"
account.1.shared_line = 2
account.1.shared_call_appearance = 1
account.1.number_of_calls_per_line_key = 5
account.1.bla_subscribe_event = "dialog"
linekey.1.type = 15
linekey.1.line = 1
linekey.1.label = "100 (1)"
linekey.2.type = 15
linekey.2.line = 1
linekey.2.label = "100 (2)"
linekey.3.type = 15
linekey.3.line = 1
linekey.3.label = "100 (3)"
linekey.4.type = 15
linekey.4.line = 1
linekey.4.label = "100 (4)"
linekey.5.type = 15
linekey.5.line = 1
linekey.5.label = "100 (5)"
```

## Expansion Modules

EXP43 (color, 20 keys × 2 pages) and EXP50 (color, 20 keys × 3 pages, T5x compatible).

```ini
expansion_module.1.linekey.1.type = 16
expansion_module.1.linekey.1.line = 1
expansion_module.1.linekey.1.value = "1001"
expansion_module.1.linekey.1.label = "Alice"
expansion_module.1.linekey.1.extension = "1001@pbx"

expansion_module.2.linekey.1.type = 13
expansion_module.2.linekey.1.value = "1002"
expansion_module.2.linekey.1.label = "Bob"
```

Up to 6 modules can chain (model-dependent). The first module powered from phone; modules 2+ need their own PSU.

EHS adapter (for wireless headsets) — `EHS35` (Plantronics-style), `EHS36` (Sennheiser DHSG), `EHS40` (USB→DECT), `EHS60/61` (USB-A direct headset adapters).

Full EXP key parameter set:

```ini
expansion_module.X.linekey.Y.type = 16
expansion_module.X.linekey.Y.line = 1
expansion_module.X.linekey.Y.value = "1001"
expansion_module.X.linekey.Y.extension = "1001@pbx"
expansion_module.X.linekey.Y.label = "Alice"
expansion_module.X.linekey.Y.pickup_value = "*8"
expansion_module.X.linekey.Y.short_label = "Ali"
expansion_module.X.linekey.Y.icon = ""
expansion_module.X.linekey.Y.xml_phonebook = 0
expansion_module.X.linekey.Y.attendant.barge_in_code = "*81"
expansion_module.X.linekey.Y.attendant.transfer_mode_via_dsskey = 0
expansion_module.X.lcd.backlight.power_saving_enable = 1
expansion_module.X.lcd.brightness = 6
expansion_module.X.lcd.contrast = 5
```

Compatibility & cabling:

```text
Module  Connector      Compatible phones                    Pages × keys  PSU
EXP20   RJ-12 daisy    T27/T29/T46/T48 (mono LCD module)    1 × 40        from phone (1st only)
EXP38   RJ-12 daisy    T26/T28 (legacy)                     1 × 38        from phone
EXP40   RJ-12 daisy    T46G/T48G                            1 × 40        ext PSU after 2nd
EXP43   USB-A          T43U/T46U/T48U/T53/T54/T57           2 × 20        bus-powered + ext PSU 2+
EXP50   USB-A          T5x series, T58W                     3 × 20 LCD    ext PSU recommended
```

Key icons (3 letters max for compact display):

```ini
expansion_module.1.linekey.1.icon = "user"
expansion_module.1.linekey.1.icon = "voicemail"
expansion_module.1.linekey.1.icon = "park"
expansion_module.1.linekey.1.icon = "transfer"
```

Daisy-chain limits — Yealink rates the bus at 6 modules max but real-world clean operation is typically 3 (USB) / 4 (RJ-12) before LED refresh stalls. Past chain length 3, run a dedicated EXP power supply on every module.

Attendant console (turret) usage — set `linekey.X.type = 16` (BLF) on every key; reception of a 60-extension SMB drops cleanly into 3 EXP43 pages. Use `expansion_module.X.linekey.Y.attendant.transfer_mode_via_dsskey = 1` to make a tap = blind transfer when on a call, normal pickup when idle.

## Bluetooth / Wireless

T48U, T54W, T57W, T58W have native Bluetooth. T46U/T48U via USB-BT dongle.

```ini
features.bluetooth.enable = 1
features.bluetooth.discoverable = 1
features.bluetooth.cfg_changed = 0
features.bluetooth.headset.audio_route = 1
```

Pair via *Menu → Basic → Bluetooth → Add Device*. Paired devices stored in NVRAM across reboots.

WiFi (T54W, T53W, T57W, T58W, W76P):

```ini
network.wifi.enable = 1
network.wifi.X.label = "officewifi"
network.wifi.X.ssid = "Office"
network.wifi.X.security_mode = "WPA2 PSK"
network.wifi.X.cipher_type = "AES"
network.wifi.X.password = "redacted"
```

USB headset profiles (HID + audio):

- *Settings → Audio → USB Headset* — claim USB device.
- Many models honour HID hook/answer/end keys on the headset.

## Auto-Provisioning

Two-file model:

1. **Common config** — `y0000000000XX.cfg` (where `XX` is the hardware ID — e.g., `y000000000054.cfg` for T54W). Applied to every phone of that model.
2. **Per-MAC config** — `<MAC>.cfg` lowercase, no separators. E.g., `805e0c123456.cfg`. Override common with per-phone settings (extension, password, line keys).

Filename rules:

```text
y000000000028.cfg   -> applies to all T46G (HW ID 28)
y000000000035.cfg   -> applies to all T48G (HW ID 35)
y000000000096.cfg   -> applies to all T54W
y000000000147.cfg   -> applies to all T33G
<lowercase-mac>.cfg -> per-phone (must be MAC of phone, no colons/dashes)
```

Phone fetch order: common file first, then MAC file (MAC overrides common).

Static provisioning URL (set on phone or via DHCP/RPS):

```ini
static.auto_provision.server.url = "https://prov.example.com/yealink/"
static.auto_provision.server.username = "yealink"
static.auto_provision.server.password = "secret"
static.auto_provision.power_on = 1
static.auto_provision.repeat.enable = 1
static.auto_provision.repeat.minutes = 1440
static.auto_provision.weekly.enable = 0
static.auto_provision.weekly.begin_time = "00:00"
static.auto_provision.weekly.end_time = "01:00"
static.auto_provision.weekly.dayofweek = "0123456"
static.auto_provision.attempt_expired_time = 5
```

(Older firmware uses `static_provision.url`/`.user`/`.password` — both keys still parsed on most.)

URL schemes supported:

```text
http://    plain HTTP
https://   HTTPS (cert verify; toggle static.security.trust_certificates = 0 to disable)
ftp://     FTP
tftp://    TFTP
ftp://user:pass@host/path
```

DHCP-driven provisioning URL — see *DHCP Option 160* below.

Encrypted config support (AES-256):

```ini
static.auto_provision.aes_key_in_file = 0
static.auto_provision.aes_key_16.com = "deadbeefcafef00d"
static.auto_provision.aes_key_16.mac = "deadbeefcafef00d"
```

`.cfg` encryption uses AES-128 / AES-256 with the YCCT-encrypted output. The phone decrypts using either an embedded factory key (per-MAC unique key) or your set keys.

## RPS — Redirect Provisioning Service

Yealink's zero-touch redirector at `rps.yealink.com`. The phone, on factory boot, contacts RPS over HTTPS. RPS replies with the dealer's provisioning URL.

Workflow:

1. Buy phone from authorized Yealink reseller (RPS-eligible MAC range).
2. Reseller / partner / customer logs into the **Yealink Device Management Cloud Service (YDMP)** — `https://dm.yealink.com` — and tags the MAC to a target provisioning URL + credentials.
3. Ship phone to customer; phone pulls IP via DHCP, contacts `rps.yealink.com`.
4. RPS replies `301 Moved Permanently` (or HTTPS 200 with config redirect) pointing to the customer's provisioning URL.
5. Phone fetches `y0000000000XX.cfg` and `<MAC>.cfg` from that URL, registers SIP, ready to ring.

Good for warehouse-shipping or remote workers — you never touch the phone before deploying.

Limits:

- RPS only fires once per factory-reset boot. Reset to factory if you need to re-trigger.
- RPS is account-bound to the reseller; resold phones may need RPS-transfer through Yealink support.
- RPS won't override an existing static provisioning URL set in the phone (only fires when the phone has no URL).

```ini
static.security.trust_certificates = 1
static.auto_provision.server.url = ""    # leave empty so RPS can populate
```

## DHCP Option 160

Yealink-preferred DHCP option for provisioning URL.

DHCP server snippets:

```text
# ISC dhcpd
option yealink-prov code 160 = string;
subnet 10.0.0.0 netmask 255.255.255.0 {
  ...
  option yealink-prov "https://prov.example.com/yealink/";
}

# Cisco IOS DHCP
ip dhcp pool VOICE
  network 10.0.0.0 255.255.255.0
  option 160 ascii https://prov.example.com/yealink/
```

Yealink option preference order:

```text
1. Option 66 (TFTP server name) — TFTP/HTTP if Yealink-DHCP-option = 66
2. Option 67 (boot file)
3. Option 159
4. Option 160 (default Yealink provisioning URL) - PREFERRED
5. Option 43 (vendor-specific)
6. RPS
```

Configurable phone-side preference:

```ini
static.auto_provision.dhcp_option.option = "160,66"
static.auto_provision.dhcp_option.enable = 1
```

DHCP Option 66 historical fallback for SPA/Polycom-mixed environments. Use 160 when you only have Yealink in the network.

## Common Configuration File

Sample `y000000000096.cfg` (all T54W):

```ini
#!version:1.0.0.1
# T54W common config

# --- Network / NTP ---
local_time.ntp_server1 = "time.cloudflare.com"
local_time.ntp_server2 = "pool.ntp.org"
local_time.summer_time = 1
local_time.time_format = 0
local_time.date_format = 0

# --- SIP servers (template; per-MAC overrides creds) ---
account.1.sip_server.1.address = "pbx.example.com"
account.1.sip_server.1.port = 5060
account.1.sip_server.1.transport_type = 2
account.1.outbound_proxy_enable = 1
account.1.outbound_host = "sbc.example.com"
account.1.outbound_port = 5061

# --- Codec policy ---
account.1.codec.1.payload_type = "PCMU"
account.1.codec.1.enable = 1
account.1.codec.1.priority = 1
account.1.codec.2.payload_type = "PCMA"
account.1.codec.2.enable = 1
account.1.codec.2.priority = 2
account.1.codec.3.payload_type = "G722"
account.1.codec.3.enable = 1
account.1.codec.3.priority = 3
account.1.codec.4.payload_type = "Opus"
account.1.codec.4.enable = 1
account.1.codec.4.priority = 4

# --- Generic Line key layout ---
linekey.1.type = 15
linekey.1.line = 1
linekey.1.label = "Line 1"

# --- Global features ---
features.dnd.enable = 1
features.headset_prior = 0
features.auto_answer_delay = 5
phone_setting.lcd_logo.mode = 0
features.config_call_record_button.enable = 1

# --- Web UI security ---
static.security.user_password = "admin:strong-replace-me"
static.security.user_password = "user:user"
static.web_item_level.enable = 1

# --- Provisioning loop ---
static.auto_provision.repeat.enable = 1
static.auto_provision.repeat.minutes = 1440
```

Sample `<MAC>.cfg`:

```ini
#!version:1.0.0.1
# Per-phone overrides
account.1.enable = 1
account.1.label = "Reception"
account.1.display_name = "Reception"
account.1.user_name = "100"
account.1.auth_name = "100"
account.1.password = "redacted-secret"
linekey.1.label = "Reception"
features.dnd.enable = 0
```

## Firmware Upgrade

```ini
static.firmware.url = "https://prov.example.com/firmware/96.86.0.30.rom"
static.firmware.upgrade.check = 1
static.firmware.upgrade.weekly.enable = 0
static.firmware.upgrade.weekly.dayofweek = "0123456"
static.firmware.upgrade.weekly.begin_time = "00:00"
static.firmware.upgrade.weekly.end_time = "06:00"
static.auto_provision.firmware_upgrade_only_when_idle = 1
```

Manual upgrade — Web UI → *Settings → Upgrade → Select File → Upgrade*. Phone reboots after flash.

Important: hardware ID in firmware filename **must** match phone hardware. Mismatch yields `Firmware Upgrade Failed: Wrong Model` and the phone simply rejects the file (won't brick).

Gradual rollout — use `static.firmware.upgrade.weekly.*` to push during off-hours.

Two-bank firmware (newer T-series): if upgrade fails mid-flash, phone reverts to last-known-good bank automatically.

## Phone GUI Customization

Wallpaper sizes (reference, vary per model):

```text
T31G/T33G          240x120  (mono)
T42G/T42S/T42U     192x64   (mono backlit)
T46G/T46U/T46S     480x272  color
T48G/T48U/T48S     800x480  color touch
T53W               360x162  color
T54W               480x272  color
T57W               800x480  color touch
T58W               1024x600 color touch (Android)
VP59               1024x600 + camera
W56H/W53H/W73H     240x320  color handset
```

```ini
static.wallpaper_upload.url = "https://prov.example.com/wallpapers/corp.jpg"
phone_setting.lcd_wallpaper = "corp.jpg"
phone_setting.backgrounds = "corp.jpg"
phone_setting.screensaver.type = 1
phone_setting.screensaver.wait_time = 600
phone_setting.lcd_logo.mode = 1
static.lcd_logo.url = "https://prov.example.com/logos/logo.dob"
```

`.dob` (Yealink Distributed Object Bitmap) — produced by the Yealink BMP-to-DOB converter for monochrome screens.

Idle clock skin and screensaver — *Settings → Preference*. Soft key layout via `softkey.<state>.<position>.label/value/type`.

## SIP-over-TLS

```ini
account.1.sip_server.1.transport_type = 2     # 0=UDP, 1=TCP, 2=TLS, 3=DNS-NAPTR
account.1.sip_server.1.port = 5061
account.1.tls.mode = 1                        # 0=v1.0, 1=All, 2=v1.1, 3=v1.2
account.1.transport = 2
```

Trusted CA — *Security → Trusted Certificates → Upload*:

```ini
static.security.ca_cert = "https://prov.example.com/ca/ca-bundle.pem"
static.security.trust_certificates = 1
static.security.cn_validation = 1
```

Phone provides server cert for mutual auth (rare, BroadWorks/SBC):

```ini
static.security.dev_cert.url = "https://prov.example.com/certs/<mac>.pem"
static.security.dev_cert.private_key.url = "https://prov.example.com/certs/<mac>.key"
```

NTP must be working **before** TLS — cert validity check fails when the clock is wrong (year 1970 / 2000 default).

## Encryption

SRTP for media:

```ini
account.1.srtp_encryption = 1     # 0=Disabled, 1=Optional (try-then-rtp), 2=Compulsory
account.1.srtp_signaling_encryption = 0
```

ZRTP (where supported):

```ini
features.zrtp_enable = 1
```

Key exchange — Yealink supports SDES (RFC 4568) widely; DTLS-SRTP only on firmware 84+ T5x/T4x-U. Confirm both ends speak the same scheme.

`a=crypto:` line in SDP — phone advertises `AES_CM_128_HMAC_SHA1_80`/`32`.

```ini
account.1.tls.tls_version = 3      # TLSv1.2 only
account.1.cipher_suite = 1         # 0=Default, 1=High, 2=Custom
```

## NAT

```ini
account.1.nat.nat_traversal = 0    # 0=STUN, 1=Disabled, 2=Active Keep-alive, 3=ICE
account.1.nat.stun_enable = 1
account.1.nat.stun_server = "stun.example.com"
account.1.nat.stun_port = 3478
account.1.nat.rport = 1
account.1.nat.udp_update_enable = 1
account.1.nat.udp_update_time = 30
account.1.nat.tcp_update_enable = 0

account.1.outbound_proxy_enable = 1
account.1.outbound_host = "sbc.example.com"
account.1.outbound_port = 5060
```

Remote-NAT recommendation: outbound proxy pointing at the SBC is more reliable than STUN for asymmetric NAT.

`account.X.nat.received_in_via = 1` enables `Via: rport` parsing.

## Action URL

Programmable HTTP-callbacks fired on phone events. Variables: `$mac`, `$ip`, `$model`, `$firmware`, `$active_user`, `$active_host`, `$local`, `$remote`, `$display_local`, `$display_remote`, `$call_id`, `$call_type`.

```ini
action_url.setup_completed = "https://crm.example.com/yealink?ev=setup&mac=$mac"
action_url.registered = "https://crm.example.com/yealink?ev=reg&u=$active_user&mac=$mac"
action_url.unregistered = "https://crm.example.com/yealink?ev=unreg&u=$active_user&mac=$mac"
action_url.register_failed = "https://crm.example.com/yealink?ev=regfail&u=$active_user&mac=$mac"
action_url.off_hook = "https://crm.example.com/yealink?ev=offhook"
action_url.on_hook = "https://crm.example.com/yealink?ev=onhook"
action_url.outgoing_call = "https://crm.example.com/yealink?ev=dial&num=$remote"
action_url.incoming_call = "https://crm.example.com/yealink?ev=ring&from=$remote"
action_url.call_established = "https://crm.example.com/yealink?ev=answer&id=$call_id"
action_url.call_terminated = "https://crm.example.com/yealink?ev=hangup&id=$call_id"
action_url.missed_call = "https://crm.example.com/yealink?ev=miss&from=$remote"
action_url.dnd_on = "https://crm.example.com/yealink?ev=dnd_on"
action_url.dnd_off = "https://crm.example.com/yealink?ev=dnd_off"
action_url.forward_on = "https://crm.example.com/yealink?ev=fwd_on&to=$forward"
action_url.forward_off = "https://crm.example.com/yealink?ev=fwd_off"
action_url.transfer_call = "https://crm.example.com/yealink?ev=xfer&to=$remote"
action_url.blind_transfer_call = "https://crm.example.com/yealink?ev=bxfer&to=$remote"
action_url.hold = "https://crm.example.com/yealink?ev=hold"
action_url.unhold = "https://crm.example.com/yealink?ev=unhold"
action_url.mute = "https://crm.example.com/yealink?ev=mute"
action_url.unmute = "https://crm.example.com/yealink?ev=unmute"
action_url.idle_to_busy = "https://crm.example.com/yealink?ev=busy"
action_url.busy_to_idle = "https://crm.example.com/yealink?ev=idle"
action_url.ip_changed = "https://crm.example.com/yealink?ev=ip&ip=$ip"
```

CRM popup integration — couple `incoming_call` to a thin server that opens the matching customer record; works with Salesforce, HubSpot, Zoho, custom.

## Action URI

Inverse of Action URL — incoming HTTP commands the phone accepts. Used for click-to-dial, remote-answer, remote-hangup, screen-pop.

```ini
features.action_uri_limit_ip = "10.0.0.0/8,172.16.0.0/12,192.168.0.0/16"
push_xml.server = "10.0.0.5"
push_xml.sip_notify = 1
```

Send commands:

```bash
# Click to dial
curl -u admin:admin "http://10.0.0.10/cgi-bin/ConfigManApp.com?key=Number=18005551212&outgoing_uri=account1"

# Send DTMF
curl -u admin:admin "http://10.0.0.10/cgi-bin/ConfigManApp.com?key=DTMF1"

# Hold/Unhold
curl -u admin:admin "http://10.0.0.10/cgi-bin/ConfigManApp.com?key=HOLD"

# Push XML to phone screen
curl -u admin:admin -X POST "http://10.0.0.10/servlet?key=push" \
  -d 'XML=<YealinkIPPhoneText><Title>Alert</Title><Text>Reboot at 18:00</Text></YealinkIPPhoneText>'
```

Allow IP-list and trusted-server config:

```ini
features.action_uri_limit_ip = "10.0.0.0/8"
push_xml.server = "10.0.0.5;10.0.0.6"
features.action_uri.allow.ip_list = "10.0.0.0/8"
```

## XML Browser / XML Phonebook

XML Browser fetches Yealink-flavored XML pages and renders interactive menus. Used for hotel directories, weather, BroadWorks call logs, custom directories.

XML object types:

```text
YealinkIPPhoneText        -- text page
YealinkIPPhoneInputScreen -- input prompt
YealinkIPPhoneStatus      -- status page
YealinkIPPhoneMenu        -- multi-choice menu
YealinkIPPhoneDirectory   -- contact list
YealinkIPPhoneFormattedTextScreen
YealinkIPPhoneIconList
YealinkIPPhoneIconMenu
YealinkIPPhoneImageScreen
YealinkIPPhoneExecute     -- execute URI / make call
```

Sample `directory.xml`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<YealinkIPPhoneDirectory>
  <Title>Company</Title>
  <Prompt>Select a contact</Prompt>
  <DirectoryEntry>
    <Name>Alice Smith</Name>
    <Telephone>1001</Telephone>
  </DirectoryEntry>
  <DirectoryEntry>
    <Name>Bob Jones</Name>
    <Telephone>1002</Telephone>
  </DirectoryEntry>
  <SoftKey index="1">
    <Label>Dial</Label>
    <URI>SK_DIAL</URI>
  </SoftKey>
</YealinkIPPhoneDirectory>
```

Remote phonebook:

```ini
remote_phonebook.data.1.url = "https://prov.example.com/phonebook/staff.xml"
remote_phonebook.data.1.name = "Staff"
remote_phonebook.data.2.url = "https://prov.example.com/phonebook/clients.xml"
remote_phonebook.data.2.name = "Clients"
remote_phonebook.search.enable = 1
remote_phonebook.callout_enable = 1
remote_phonebook.update_time = 21600
```

Phonebook XML (remote_phonebook):

```xml
<?xml version="1.0" encoding="UTF-8"?>
<YealinkIPPhoneBook>
  <Title>Staff</Title>
  <Menu Name="Sales">
    <Unit Name="Alice Smith" Phone1="1001" Phone2="14155551001" Phone3=""/>
    <Unit Name="Bob Jones" Phone1="1002"/>
  </Menu>
  <Menu Name="Engineering">
    <Unit Name="Carol Liu" Phone1="2001"/>
  </Menu>
</YealinkIPPhoneBook>
```

BroadSoft, 3CX, Asterisk, FreePBX integrations expose Yealink-format phonebook XML at known URLs — paste them into `remote_phonebook.data.X.url` and they appear under *Directory*.

## LDAP

```ini
ldap.enable = 1
ldap.name_filter = "(|(cn=%)(sn=%))"
ldap.number_filter = "(|(telephoneNumber=%)(mobile=%)(homePhone=%))"
ldap.host = "ldap.example.com"
ldap.port = 389
ldap.base = "ou=Users,dc=example,dc=com"
ldap.user = "cn=phone,ou=Service,dc=example,dc=com"
ldap.password = "redacted"
ldap.max_hits = 50
ldap.name_attr = "cn sn"
ldap.numb_attr = "telephoneNumber mobile homePhone"
ldap.display_name = "%cn"
ldap.version = 3
ldap.tls_mode = 0
ldap.call_in_lookup = 1
ldap.ldap_sort = 1
ldap.lookup_incoming = 1
ldap.search_delay = 0
```

LDAP-over-TLS:

```ini
ldap.port = 636
ldap.tls_mode = 2          # 0=disabled, 1=StartTLS, 2=LDAPS
```

`ldap.name_filter` substitutes `%` with the search string. Test against your AD/389 schema with `ldapsearch` first.

## Web UI Authentication

Three permission tiers — admin, var, user:

```ini
static.security.user_password = "admin:newadminpass"
static.security.user_password = "user:newuserpass"
static.security.user_password = "var:newvarpass"
static.security.var_enable = 1
static.web_item_level.enable = 1
features.web_logon.enable = 1
features.disable_user_account_settings = 1
features.disable_user_phone_settings = 0
```

Restrict the web UI port:

```ini
network.port.http_port_enable = 1
network.port.https_port_enable = 1
network.port.http_port = 80
network.port.https_port = 443
network.port.web_server_type = 2     # 0=disabled, 1=HTTP, 2=HTTPS, 3=both
```

`web_item.user.X` / `web_item.var.X` keys hide individual UI elements per role.

## Password Policy

```ini
static.security.password_complex_enable = 1
static.security.password_min_length = 10
static.security.password_max_age = 90
static.security.password_history = 5
static.security.lock.enable = 1
static.security.lock.attempts = 5
static.security.lock.duration = 300
```

Minimum-length check applies on every password change (Web UI, handset menu, provisioning). 0 disables.

## Logs

System log levels:

```text
0 - emergency
1 - alert
2 - critical
3 - error
4 - warning
5 - notice
6 - info / debug (most verbose useful)
7 - trace (firmware-only; rarely usable)
```

```ini
static.syslog.mode = 1                       # 0=local only, 1=remote (also keeps local)
static.syslog.server = "10.0.0.20"
static.syslog.server_port = 514
static.syslog.transport_type = 0             # 0=UDP, 1=TCP, 2=TLS
static.syslog.facility = 16                  # local0
static.syslog.log_level = 6
static.syslog.app_module_log_level = 6
static.syslog.prepend_mac_address.enable = 1
static.local_log.enable = 1
static.local_log.level = 6
static.local_log.max_file_size = 1024
static.local_log.contain_year.enable = 1
```

Pull local logs from web UI: *Maintenance → System Log → Export*. They land as a `.tar.gz` with `system.log`, `boot.log`, `syslog`, `dmesg`.

## Auto Answer / Intercom

Auto-answer when a call comes in (great for front-desk):

```ini
features.auto_answer = 1                     # global
account.1.auto_answer = 1                    # per-account
features.auto_answer_delay = 0
features.auto_answer.tone.enable = 1
```

Intercom (push-to-talk-style — answers headset/speaker without ringing):

```ini
intercom.allow.code_enable = 1
features.intercom.allow = 1
features.intercom.mute = 0
features.intercom.tone = 1
features.intercom.barge = 0
linekey.5.type = 13
linekey.5.value = "1001"
linekey.5.line = 1
linekey.5.label = "PageAlice"
```

PBX side must send `Alert-Info: <intercom>` (or `Call-Info` per BroadWorks) for the phone to honour intercom.

## Hot Desking

```ini
features.hot_desking.enable = 1
linekey.10.type = 30
linekey.10.label = "Login"
features.hot_desking.password.enable = 0
```

User taps the Hot Desk key, types extension + auth password, the phone reconfigures `account.1.user_name` etc. on the fly. On logout, phone reverts to a base-template config (provided via provisioning) so the next user starts fresh.

## Common Errors

Verbatim where possible — these are the strings that show on the LCD or in the system log.

- **"Register Failed: Authentication Failed"** — wrong `account.X.password` or `auth_name`. Check PBX user, regenerate password.
- **"Register Failed: 403 Forbidden"** — server rejected the IP or transport. Check ACLs (`permit=` in Asterisk, ACL list in FreePBX, IP whitelist on SBC).
- **"Register Failed: 408 Request Timeout"** — phone got no response. Network/firewall blocking SIP; outbound proxy unreachable; wrong port.
- **"No SIP Server"** — `account.X.sip_server.1.address` typo or DNS resolves wrong. Try IP literal to bypass DNS.
- **"SDP Negotiation Failed"** — codec mismatch. Phone offers PCMU/G722; PBX wants G729. Adjust `account.X.codec.X.enable` to overlap.
- **"Call Failed: 480 Temporarily Unavailable"** — callee not registered, DND on, or no available channel.
- **"DHCP Discover Failed"** — phone couldn't get IP. Check DHCP scope, exhausted leases, untagged port (VLAN), cable/PoE.
- **"TLS Handshake Failed"** — wrong CA, wrong TLS version, wrong cipher, time wrong (cert not yet valid). Check `static.security.trust_certificates` and `account.X.tls.mode`.
- **"Provisioning Failed: 401 Unauthorized"** — wrong `static.auto_provision.server.username/password`.
- **"Firmware Upgrade Failed: Wrong Model"** — hardware ID prefix in `.rom` filename does not match phone family.
- **"Network Unavailable"** — link down, no IPv4 lease, or DNS unresolvable.
- **"Connecting To Server"** — registration still pending; can stall on TCP handshake at SBC.
- **"Server Connection Lost"** — outbound proxy dropped after registration.
- **"Configuration File Format Error"** — malformed `.cfg` (mismatched quotes, unsupported encoding, BOM at start of file). Re-export from YCCT.
- **"Decryption Failed"** — encrypted `.cfg` AES key mismatch.
- **"DECT not registered"** — handset on W-series lost base; press base sync key for 5s.
- **"Battery Low"** — DECT/Bluetooth handset.

## Common Gotchas

Each entry: broken → fixed.

1. **Default password**:
   - Broken — `static.security.user_password = "admin:admin"` shipped to production. Anyone on the LAN can re-flash.
   - Fixed — set `static.security.user_password = "admin:<long-random>"` in the common config; rotate on every provisioning cycle.
2. **HTTP vs HTTPS provisioning URL**:
   - Broken — `static.auto_provision.server.url = "http://prov.example.com/yealink/"` but server only listens on 443.
   - Fixed — match the URL scheme exactly to the listener; verify with `curl -I` from the phone's network.
3. **Codec list excludes the required codec**:
   - Broken — `account.1.codec.1.payload_type = "Opus"` only; PBX is Asterisk on G.711 only — calls fail SDP.
   - Fixed — keep PCMU/PCMA enabled at minimum; promote Opus/G722 above them, but never remove the lowest-common-denominator.
4. **Multiple accounts but `linekey.X.line` not set**:
   - Broken — `account.2.enable=1` registers, but `linekey.2.line` defaults to `1` so dial-out from line 2 button uses account 1.
   - Fixed — explicitly `linekey.2.line = 2` and `linekey.2.label = "Account2"`.
5. **BLF subscribed but server doesn't publish**:
   - Broken — Phone sends `SUBSCRIBE Event: dialog`, FreeSWITCH publishes `presence` instead — LED never lights.
   - Fixed — On Yealink, force `account.1.blf.list_event_type = "dialog"`. On FreeSWITCH, enable `mod_dialog`.
6. **Static provision URL wrong path**:
   - Broken — `static.auto_provision.server.url = "https://prov.example.com/configs"` (missing trailing slash) → some HTTP servers 404 the directory listing.
   - Fixed — always end the URL with `/` so phone appends `y0000000000XX.cfg` cleanly.
7. **DHCP Option 66 and 160 conflicting**:
   - Broken — Both options set on DHCP scope to different URLs; phone picks 66, ignores 160.
   - Fixed — Set `static.auto_provision.dhcp_option.option = "160"` on the phone, or remove option 66 from DHCP for Yealink-only nets.
8. **Time wrong (NTP)**:
   - Broken — Phone clock is 1970-01-01. TLS cert validation fails because cert `notBefore` is in the future.
   - Fixed — Set NTP at the network level (DHCP option 42) and `local_time.ntp_server1`. Verify via *Status* page before enabling TLS.
9. **Outbound proxy set per-account but not enabled**:
   - Broken — `account.1.outbound_host = "sbc"` but `account.1.outbound_proxy_enable = 0`. Phone bypasses SBC, fails over symmetric NAT.
   - Fixed — flip the enable bit; check *Status* shows `via SBC`.
10. **W-series DECT pairing fails — base/handset firmware mismatch**:
    - Broken — W60B base on `77.x.x.x` firmware; handset W56H on `61.x.x.x` from a 3-yr-old box. Pairing aborts.
    - Fixed — upgrade the base first, then upgrade each handset over-the-air via base. Yealink publishes paired firmware sets.
11. **VP video not supported by upstream codec set**:
    - Broken — VP59 advertises H.264; SBC strips video. Calls connect audio-only.
    - Fixed — ensure SBC config passes `m=video` and that `H264` is enabled on the upstream account profile.
12. **Action URL via HTTPS without cert in trust store**:
    - Broken — `action_url.registered = "https://crm.example.com/..."` but the phone hasn't loaded the CA. POST silently fails.
    - Fixed — upload the CA to *Security → Trusted Certificates* or via `static.security.ca_cert`.
13. **MAC config filename case**:
    - Broken — uploaded `805E0C123456.cfg` (uppercase). Phone fetches `805e0c123456.cfg` (lowercase) — 404, falls through to common only.
    - Fixed — always lowercase MAC filenames; validate with `nginx` access log.
14. **Provisioning loop without rate-limit**:
    - Broken — `static.auto_provision.repeat.minutes = 5` (every 5 min) on 200 phones DDoS's the prov server.
    - Fixed — set 1440 (daily) and rely on action-URL re-provisioning when needed.
15. **Web UI exposed to WAN**:
    - Broken — port-forward 80→phone, default creds, or even rotated creds — RCE-class CVEs land yearly.
    - Fixed — keep web UI on management VLAN, use `network.port.web_server_type = 0` to disable on field phones, or restrict by ACL.
16. **Wrong factory-reset combo**:
    - Broken — pressing OK key on T57W during boot does nothing.
    - Fixed — long-press `*` and `#` simultaneously for 10 seconds while phone is booting; wait for "Reset to factory?" prompt.
17. **Wallpaper wrong size**:
    - Broken — pushed 800x600 wallpaper to T54W (480x272); phone scales badly, looks awful.
    - Fixed — match resolution exactly per the wallpaper-size table.
18. **EHS adapter wrong model**:
    - Broken — Plantronics-EHS35 paired with Sennheiser headset → no answer/end events.
    - Fixed — use EHS36 for Sennheiser DHSG. Check headset brand → adapter compatibility chart.

## Diagnostic Tools

Web UI → *Maintenance*:

- **PCAP** — *Diagnostic → Packet Capture* — start, reproduce issue, stop, download `.pcap`. Usable in Wireshark with the `sip` filter; `rtp` for media.
- **System Log** — *System Log → Export Log Files* → `.tar.gz` with full /var/log.
- **SIP Trace** — *Diagnostic → SIP Trace* — live SIP message stream in browser; also captured to log.
- **Configuration Export** — *Configuration → Export CFG* — local + non-static settings → `.cfg`.
- **TR-069 Export** — *Settings → TR069* — for ACS-managed deployments.
- **Diagnostic Tools** — ping, traceroute, ARP via Web UI.

CLI when SSH unlocked:

```bash
# diagnostic shell on factory-debug firmware
ssh admin@<phone-ip>
> show network
> show sip account 1
> show registration
> tcpdump -i eth0 -w /tmp/capture.pcap
```

`ConfigManApp.com` URI is also the basis for headless Action-URI scripting.

YCCT (Yealink Configuration Conversion Tool) — Windows desktop tool that reads/writes `.cfg`/`.boot`, encrypts AES, validates schema. Free download from `support.yealink.com`.

## Hardware Specifics

- **T2 series** — `T21P_E2`, `T23G`, `T27G`. Low-end. 100Mbps LAN/PC, monochrome LCD, PoE on -G models. T27G has 21 line keys via 8 physical + paged context.
- **T3 series** — `T31G`, `T33G`, `T34W`. Mid-range. 1Gbps LAN/PC + PoE; T34W adds Wi-Fi/BT. T33G has color LCD. Yealink's "value" sweet spot.
- **T4 series** — `T42U`, `T43U`, `T44W`, `T46U`, `T48U`. Color (T46U/T48U are flagship). T46U mid-tier exec; T48U color touchscreen. -U variants support USB headset; -G is older 1Gbps; -S is intermediate.
- **T5 series** — `T53W`, `T54W`, `T57W`, `T58W`. Top tier. BT, Wi-Fi, USB. T57W is touchscreen; T58W runs Android (apps, Skype for Business). Camera CAM50 plug-in for video.
- **W-series** — `W56P` (single-cell base + 1 handset), `W76P` (W70B base, premium handset), `W60P` (legacy base + W56H), `W80B/W80DM` (multi-cell), `W90B/W90DM` (large enterprise). DECT 1.92 GHz / 1.88 GHz / 1.78 GHz region-coded.
- **VP-series** — `VP59` only currently. 8" 1024x600 capacitive touchscreen, integrated 1080p camera, BT/Wi-Fi/USB.
- **MeetingBoard** — `MeetingBoard 65/86` Android conferencing display. `MeetingBar A20/A30` USB camera bar for Zoom/Teams.
- **MP-series** — `MP54`, `MP56`, `MP58` Microsoft Teams certified phones (Android). Plus `MP54-WH` for Webex.
- **CP-series** — `CP920`, `CP925`, `CP935W`, `CP965` conference room phones. Wireless options on -W variants.
- **ATA / Audio Gateway** — `MP54` analogue ATA isn't in this line; the older `RT10/RT20/RT30` and Yealink VC-series SBC handle gateway use cases.

## Models Most-Used

- **T54W** — most-deployed exec phone. 4.3" color, BT/WiFi.
- **T46U** — staff workhorse. 4.3" color, dual 1Gbps + PoE.
- **T48U** — touchscreen exec. 7" capacitive, dual Gbps + PoE + USB.
- **T57W** — premium touchscreen. 7", BT/WiFi.
- **T31G** — entry phone. 1Gbps LAN. 132x64 monochrome.
- **T33G** — entry+ phone. Color. 1Gbps + PoE.
- **T42G** — small office classic.
- **T44W** — mid-tier with WiFi.
- **T48G** — older flagship 7" touch.
- **W56P** — base + 1 cordless handset.
- **W76P** — premium DECT (W70B + W76H handset).
- **MP54** — Teams-certified small office.
- **MP56** — Teams 7" color.
- **VP59** — video desk, the surviving VP SKU.

## Idioms

- "Use HTTPS for provisioning." — never plain HTTP; phone caches creds in flash and a tcpdump on a hub leaks every account password.
- "Always set static_provision.url + .user + .password." — incomplete triplet defeats RPS / DHCP-160; populate all three or none.
- "Match codec list to upstream PBX." — the lowest-common-denominator codec must always be enabled (PCMU or PCMA).
- "BLF requires server-side dialog event package." — Asterisk needs `hint=` lines; FreeSWITCH `mod_dialog`; BroadWorks publishes natively.
- "Use RPS for global zero-touch deploy." — one-time MAC-tag, ship anywhere, boot anywhere.
- "T-series for desk; W-series for cordless; MP for ATA." — pick the line by user pattern, not by feature checklist.
- "Lowercase MAC, no separators, `.cfg`." — `805e0c123456.cfg` not `80-5E-0C-12-34-56.cfg`.
- "Common.cfg, then MAC.cfg." — common applies first; per-MAC overrides last write.
- "NTP before TLS, always." — TLS dies silently on wrong clock.
- "Restrict web UI to management VLAN." — the web UI is the soft underbelly of every IP phone family.
- "Tag firmware ROM by hardware ID." — wrong prefix → upgrade refused, no brick, but no upgrade either.

## See Also

- ip-phone-provisioning
- sip-protocol
- asterisk
- freeswitch
- polycom-phones
- cisco-phones
- grandstream-phones
- snom-phones

## References

- Yealink Support — `https://support.yealink.com/` (model-specific admin guides, datasheets, RPS portal).
- Yealink Documentation — `https://docs.yealink.com/` (auto-provisioning guides, REST API, Action URI/URL spec).
- Yealink Configuration Conversion Tool (YCCT) — `https://support.yealink.com/en/portal/docList?archiveType=software&productCode=cd2c6a4cd9f6d6b9` (cfg authoring, encryption, validation).
- Yealink Device Management Cloud Service (YDMP) — `https://dm.yealink.com/`.
- Yealink RPS — `https://rps.yealink.com/` (account-bound zero-touch redirector).
- RFC 3261 — SIP.
- RFC 3265 / 6665 — SIP-Specific Event Notification (SUBSCRIBE/NOTIFY).
- RFC 4235 — Dialog Event Package (BLF substrate).
- RFC 4568 — SDES (SDP Security Descriptions).
- RFC 5763 / 5764 — DTLS-SRTP framework.
- RFC 5359 — Session Initiation Protocol Service Examples (BLA / SLA).
- RFC 6228 — Session Initiation Protocol (SIP) Response Code for Indication of Terminated Dialog.
- BroadWorks Device Management — `https://www.cisco.com/c/en/us/products/unified-communications/broadworks.html` (BLA/SCA reference).
- Yealink Forum — `https://forum.yealink.com/` (community Q&A, bug reports).
