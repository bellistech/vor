# Mitel IP Phones

Mitel SIP-firmware product family — 6800/6900 series provisioning, codecs, BLF, MBG, MiCollab, MiContact Center.

## Setup

Mitel (formerly Aastra Technologies, acquired 2014) sells a broad portfolio of IP and DECT desk phones, with overlapping product lines reflecting decades of mergers.

### Portfolio overview

- **6800 series — entry / mid SIP**
  - **6863i** — 2-line, monochrome 2-line LCD, 3 soft keys, no PoE class-1 power option, basic G.711/G.722.
  - **6865i** — 9-line, 3.5-inch monochrome LCD, 8 programmable keys, gigabit Ethernet, PoE class 2.
  - **6867i** — 9-line, 3.5-inch color LCD, 8 programmable keys, gigabit, PoE class 3.
  - **6869i** — 24-line, 4.3-inch color LCD, expansion-module support, dual gigabit, PoE class 3.

- **6900 series — modern SIP / IP**
  - **6905** — entry, 2-line monochrome LCD, single Ethernet, basic.
  - **6910** — 1-line LCD upgrade, single Ethernet.
  - **6920** — 9-line, 3.5-inch color LCD, gigabit, USB-A, PoE class 2.
  - **6930** — 24-line, 4.3-inch color LCD, USB-A, Bluetooth (Personal Wireless Pairing), gigabit, expansion-module support, PoE class 3.
  - **6940** — premium, 7-inch color touchscreen, dual gigabit, USB-A, Bluetooth, integrated DECT cordless option, PoE class 3.
  - **6970** — flagship, 7-inch color touchscreen, USB-C, Bluetooth, gigabit, expansion-module support, PoE class 3.

- **5300 series — legacy MiNet**
  - **5304 / 5312 / 5320 / 5324 / 5330 / 5340 / 5360** — proprietary MiNet protocol against MiVoice Business / 3300 ICP.
  - Re-flashable to **5320e / 5330e / 5340e / 5360e SIP firmware** for non-Mitel PBX use.

- **5600 series — cordless DECT**
  - **5614 / 5624 / 5634** — DECT handsets paired with **RFP / DECT IP base stations** (RFP-32, RFP-42, RFP-43, RFP-44).
  - Base stations register over SIP; handsets are MiNet-style payloads to the base.

- **7000 series — MiVoice Office hard phones**
  - **6735i / 6755i / 6757i** — older Aastra-branded SIP phones still in field.
  - **6739i** — touchscreen flagship of the 67xx series.
  - Predates 68xx; configuration framework is identical (XML config, aastra.cfg).

- **6800i / 6900i SIP firmware** — the **i** suffix marks "SIP firmware ready for non-Mitel PBX." Same hardware, different default firmware payload. Critical when ordering: 6867i and 6867 (MiNet) are different SKUs.

### Adjacent infrastructure

- **MiVoice Border Gateway (MBG)** — Mitel's SBC and remote-worker proxy. Phones register through MBG to reach an internal MiVoice Business / 5000 / 250 / Office 400 / Connect / MX-One.
- **MiCollab UC** — unified communications platform: presence, messaging, conferencing, mobile client. Provisions desk phones via CSP.
- **MiContact Center** — Mitel's contact center stack (formerly prairieFyre Solidus eCare); integrates with phones via ACD agent state, key/lamp control.
- **MiVoice Business / 5000 / 250 / Office 400 / Connect / MX-One** — the various PBX platforms across Mitel's mergers (3300 ICP, 5000, ShoreTel/Connect, Aastra MX-One). Each speaks SIP to phones.

The MiNet vs SIP firmware split for the 5300 series is the most common point of confusion: a 5320 in stock with MiNet firmware will not register to Asterisk/FreeSWITCH/3CX without re-imaging to 5320e SIP firmware. Mitel's tools (Mitel Reflash Utility) and the MiVoice Connect bootloader handle the conversion.

## Default Access

After power-up the phone obtains a DHCP lease and exposes its web UI at the lease address.

```
http://<phone-ip>/
```

Default credentials (vendor-shipped, identical across the 68xx and 69xx families):

| Account | Username | Password |
|---------|----------|----------|
| Admin   | admin    | 22222    |
| User    | user     | 1111     |

The admin password is famously "five 2s" — not 12345 like most vendors. Many auditors flag this on first walk-through because phones are routinely deployed with 22222 unchanged.

Other administrative entry points:

- **Web UI** — port 80 HTTP by default; can be moved to HTTPS-only via TLS configuration.
- **OIA Telnet** — legacy diagnostic console on TCP/23. Disabled by default in modern firmware. Some 67xx-era firmware still ships it for debug.
- **SSH** — disabled by default. Mitel rarely enables SSH on consumer phones; debug console is web-driven.
- **DHCP redirector / CSP** — zero-touch provisioning entry point (see CSP section).

If the admin password is forgotten, factory reset is the only recovery: hold the down-arrow on power-up for the 68xx, or use the recovery menu on 69xx via the **Settings → Advanced → Restore to Default** keypad sequence after entering the bootloader.

## Web UI Tour

The web admin interface is structured as a left-rail navigation with the following pages (terminology consistent across 68xx/69xx, slightly different on 67xx):

- **Status** — firmware version, MAC, IP, hardware revision, boot count, registration status per line, current call info.
- **System Information** — vendor, model, serial number, SIP user agent string.
- **Network Configuration** — DHCP/static, VLAN, LLDP, DNS, NTP, QoS markings.
- **Audio** — codecs, jitter buffer, AGC, sidetone, ringer settings.
- **Phone Lock** — PIN, emergency dial-allow, lock-on-boot.
- **Audio Diagnostics** — packet capture (PCAP), tone test, microphone test, audio loopback.
- **Call Forward** — per-line CFA / CFB / CFNR rules.
- **SIP Account Configuration** — per-line auth (see next section).
- **Operation** — DSS keys, BLF, key labels, ringtones, dial plan, DTMF mode, call waiting.
- **Programmable Keys** — soft / top / bottom / expansion module key assignments.
- **Time and Date** — NTP, DST, time zone, format, manual override.
- **Custom Branding** — wallpaper, screensaver, logo, idle-screen image.
- **TLS** — trusted certificates, identity certificate, mutual TLS.
- **Provisioning** — Configuration Server URL, transfer protocol, polling interval.
- **SNMP** — community strings, trap destinations, SNMPv3 user.
- **Reset** — factory reset, partial reset (network only / config only / clear logs).
- **Restart** — soft restart, scheduled restart.

Tabs that appear conditionally:

- **ACD** — only visible when SIP Account is configured for ACD.
- **XML Browser** — only when XML server is enabled.
- **Bluetooth** — only on 6930/6940/6970.

## SIP Account Configuration

Each phone supports up to **12 SIP lines** (Lines 1–12), with the line keys mapped to programmable keys (see Programmable Keys).

Per-line settings:

| Field                | Meaning                                                                                          |
|----------------------|--------------------------------------------------------------------------------------------------|
| Line Mode            | Generic / Asterisk / BroadSoft / MiVoice (per-line, alters feature behavior)                     |
| Display Name         | Caller name sent in From header (P-Asserted-Identity if enabled)                                 |
| Screen Name          | Label on the phone screen for this line                                                          |
| Screen Name 2        | Secondary line label (e.g., extension number under name)                                         |
| Phone Number         | E.164 / extension; populates From URI user part                                                  |
| Caller ID            | Override CID; if blank uses Phone Number                                                         |
| Auth Username        | SIP digest auth user (often same as Phone Number)                                                |
| Auth Password        | SIP digest password                                                                              |
| Server / SIP Proxy   | Primary SIP server (FQDN preferred)                                                              |
| Port                 | Primary server port (5060 UDP/TCP, 5061 TLS)                                                     |
| Outbound Proxy       | If using SBC / MBG separate from registrar                                                       |
| Outbound Proxy Port  | Usually same as registrar                                                                        |
| Backup Server        | Secondary registrar; phone fails over after Registration Period or X registration failures       |
| Backup Server Port   | Backup port                                                                                      |
| Registration Period  | REGISTER refresh interval; default 3600 sec; 600 typical for failover-sensitive deployments      |
| Voicemail Number     | MWI subscribed; per-line                                                                         |
| Voicemail Mailbox    | If voicemail account is keyed differently from Phone Number                                      |

Line Modes meaningfully change behavior:

- **Generic** — vanilla SIP/RFC 3261 + RFC 6665 (events). Use for Asterisk, FreeSWITCH, custom carriers.
- **Asterisk** — fixes a few quirks (e.g., the way Asterisk sends NOTIFY for MWI, the way reINVITEs are handled).
- **BroadSoft** — enables BroadSoft-specific features (XSI, BLF list as advertised by Broadworks).
- **MiVoice** — enables Mitel-proprietary features (MiVoice Business advanced presence, MBG indications).

## Codec Configuration

The codec list is configured under **Audio → Codec → Codec List** (and per-line under SIP Account if firmware supports per-line codec).

Supported codecs (varies by model; modern 6900 series carries the full list):

| Codec      | Bandwidth | Notes                                                            |
|------------|-----------|------------------------------------------------------------------|
| PCMU (G.711U) | 80 kbps | Default, ulaw — North America                                  |
| PCMA (G.711A) | 80 kbps | alaw — Europe / RoW                                            |
| G.722      | 80 kbps   | Wideband (HD voice), 50–7000 Hz                                  |
| G.722.1    | 24/32 kbps | Wideband, lower bandwidth than G.722                            |
| G.729a     | 8 kbps    | Low-bandwidth narrowband; license-bound on some firmware         |
| iLBC       | 13.3/15.2 kbps | Variable; useful over lossy links                            |
| Opus       | 6–510 kbps | 6900-series only; modern wideband codec                         |
| L16        | 256 kbps  | Linear PCM; rarely used                                          |

Order matters — the **first codec in the preferred list** appears first in SDP, and the upstream side typically picks it.

```
Codec 1 = OPUS
Codec 2 = G722
Codec 3 = PCMU
Codec 4 = PCMA
Codec 5 = G729
```

The **Audio Pre-Test** tool (under Audio Diagnostics) plays a tone through the speaker and records via the microphone, useful for verifying codec/loopback before deployment.

Per-codec parameters:

- **Packetization** — ptime in ms (10/20/30); 20 default. G.729 typically 20; Opus 20; G.722 20.
- **Silence suppression / VAD** — global toggle; off by default.
- **Echo cancellation** — on by default; G.168 compliant.
- **Comfort noise generation (CNG)** — on by default for narrowband codecs.

## Programmable Keys

Mitel phones expose four classes of programmable keys:

- **Soft Keys (S1–S20)** — context-sensitive keys below the LCD; assignment depends on phone state (idle / dialing / connected / ringing).
- **Top Keys** — the row of LEDs at the top of the phone (lines 1–12 typically).
- **Bottom Keys** — extra programmable buttons below the soft keys (varies by model).
- **Expansion Module Keys** — per-page programmable keys on attached M695/M680 modules (see Expansion Modules).

Key Types:

| Type             | Description                                                                  |
|------------------|------------------------------------------------------------------------------|
| Line             | Maps the key to a registered SIP line; press to seize                        |
| Speed Dial       | One-touch dial of arbitrary URI/digit string (NOT a registered line)         |
| BLF              | Subscribes to dialog event of a single URI; lamp shows monitored state       |
| BLF/List         | Subscribes to dialog event-list (RFC 4662); monitors many extensions         |
| Custom           | XML-based key, executes phone XML action                                     |
| Last Call Return | Dial the last incoming caller                                                |
| ACD              | ACD agent state toggle (Login/Logout/Auto/Wrap-Up)                           |
| Discreet Ringing | Visual ringing only, no audible                                              |
| Conference       | Initiate three-way                                                           |
| Transfer         | Blind/attended transfer trigger                                              |
| Hold             | Hold/resume                                                                  |
| Pickup           | Group pickup (via SIP REFER or feature code)                                 |
| Park / Pickup-Park | Call park slot                                                             |
| Voicemail        | Auto-dial voicemail box                                                      |
| XML              | Trigger XML browser app via URL                                              |
| Empty            | No assignment                                                                |
| None             | Disabled                                                                     |

Key labels are configurable per-key (text shown next to the LED). For BLF/List, labels come from the server-pushed list.

## Lines vs Speed Dial

Mitel explicitly distinguishes the two. **A "line" is a registered SIP account.** Pressing a line key seizes that account, gets a dial tone bound to that registration, and outgoing INVITEs use that From URI. Inbound calls to that account ring the line key.

A **Speed Dial** is just a one-touch outbound dial. It does not register, does not consume a SIP account slot, and on inbound calls cannot ring (because nothing is registered to that key). Pressing a speed dial seizes the **default outbound line** (Line 1 unless overridden) and dials the configured number/URI.

This matters because:

- A receptionist phone might have 1 registered line (their own DID) plus 24 speed-dial keys with one-touch internal numbers.
- Misconfiguring an external number as a "Line" causes failed REGISTER attempts to a number that isn't a SIP account.

The web UI labels the two distinctly; the underlying config parameters are different too:

```
sip line1 enabled: 1                    # Line — registered account
sip line1 user name: 5001
sip line1 auth name: 5001
sip line1 password: ...
sip line1 sip server: pbx.example.com

topsoftkey1 type: speeddial             # Speed Dial — not registered
topsoftkey1 value: 8005551212
topsoftkey1 label: External AnswerSvc
topsoftkey1 line: 1                     # Use Line 1's outbound proxy
```

## BLF + BLF List

Busy Lamp Field (BLF) lets a key's LED reflect the call state of another extension.

### Single-extension BLF

Per RFC 4235, the phone sends:

```
SUBSCRIBE sip:5001@pbx.example.com SIP/2.0
Event: dialog
Accept: application/dialog-info+xml
Expires: 3600
```

The PBX answers with NOTIFYs as `5001` transitions through `terminated → trying → confirmed → terminated`. The lamp goes green/red/flashing accordingly.

Each monitored extension consumes one SUBSCRIBE dialog. With 50+ keys this is heavy on the PBX.

### BLF List (RFC 4662)

The phone subscribes once to a "list URI" advertised by the PBX:

```
SUBSCRIBE sip:reception-blf@pbx.example.com SIP/2.0
Event: dialog;eventlist
Supported: eventlist
```

The PBX responds with a multipart NOTIFY containing the dialog state of all members of that list. One subscription, up to 50 monitored extensions in one body.

Mitel BLF List requires:

1. Server support for `Event: dialog;eventlist` (Asterisk: `res_pjsip_pubsub`+`hint` aggregation; FreeSWITCH: `mod_dialog`; BroadSoft: native).
2. The list URI configured on the phone under SIP Account → BLF/List URI.
3. Programmable keys set to **BLF/List** type — keys auto-populate with list members in order.

The labels on BLF/List keys come from the `<dialog-info>` body's display-name fields, so the PBX's user database controls what the receptionist sees.

## Expansion Modules

For receptionist and operator setups, Mitel sells two expansion modules that connect to the rear of compatible phones:

- **M695** — 60-button color LCD module, 3 pages × 20 keys per page, color icons, daisy-chains up to 3 modules per phone.
- **M680** — 36-button monochrome LCD, 3 pages × 12 keys, daisy-chains up to 3.

Compatibility: M680 with the 6800 series (6865/6867/6869 only); M695 with the 6800i (6867/6869) and 6900 series (6920/6930/6940/6970).

Connection is via **RJ-12** from the rear "EM" port — daisy-chain in series. The first module is powered by the phone (or by an external Mitel PSU if 3+ chained or if the phone is class-2 PoE).

Per-key, per-page programming follows the same key-type model as the phone itself (Line, Speed Dial, BLF, BLF/List, etc.). Page navigation is via the page-shift key on the module itself.

```
expmod1 page1 key1 type: blf
expmod1 page1 key1 value: 5001
expmod1 page1 key1 label: Reception
expmod1 page1 key1 line: 1
```

## XML Browser

Mitel phones include a built-in **XML Browser** that renders Mitel-XML pages, enabling phone-side applications:

- **Directory lookups** — pull contacts from LDAP or a CRM via an XML gateway.
- **Presence display** — show colleague status from MiCollab or Skype/Teams.
- **IVR menus** — phone-driven menus that drive call control.
- **External app integration** — service tickets, hotel checkout, factory floor operator tasks.

The schema is **MitelXML** (formerly AastraXML), defined by element types:

| Element             | Purpose                                                |
|---------------------|--------------------------------------------------------|
| `<MitelIPPhoneTextScreen>` | Plain text view                                |
| `<MitelIPPhoneTextMenu>`   | Menu of items, each maps to a URL or dial   |
| `<MitelIPPhoneInputScreen>` | Prompt user for input, POST result          |
| `<MitelIPPhoneDirectory>`   | Directory listing                            |
| `<MitelIPPhoneStatus>`      | Status line update                           |
| `<MitelIPPhoneFormattedTextScreen>` | Rich text                            |
| `<MitelIPPhoneImageScreen>` | Image display                                |
| `<MitelIPPhoneImageMenu>`   | Menu with images                             |
| `<MitelIPPhoneExecute>`     | Execute commands (Dial, Key, URL, etc.)      |

Example minimal XML page:

```xml
<MitelIPPhoneTextMenu>
  <Title>Quick Actions</Title>
  <Prompt>Pick one</Prompt>
  <MenuItem>
    <Name>Dial Reception</Name>
    <URL>http://app.example.com/dial?to=5000</URL>
  </MenuItem>
  <MenuItem>
    <Name>Page Group</Name>
    <URL>Dial:5500</URL>
  </MenuItem>
</MitelIPPhoneTextMenu>
```

Triggering: a programmable key set to **XML** type with the XML server URL. The phone fetches the URL with HTTP GET, including query parameters identifying the phone (MAC, ext) and the key pressed.

XML Push (server-initiated) is also supported: the phone listens on TCP/80 and accepts pushed XML from authorized origin IPs, replacing the current screen. Requires the XML Push Server List + XML Push Auth.

## Provisioning

Mitel phones support automatic provisioning from a configuration server. Settings live under **Provisioning** in the web UI.

### Configuration Server URL

```
Provisioning Server:    https://prov.example.com/mitel/
Configuration Server:   https://prov.example.com/mitel/
Download Protocol:      HTTPS
```

### Supported transfer protocols

| Protocol | Port | Auth                          | Encrypt-in-flight |
|----------|------|-------------------------------|-------------------|
| HTTP     | 80   | Basic                         | No                |
| HTTPS    | 443  | Basic + cert (mutual optional)| Yes               |
| FTP      | 21   | User/pass                     | No                |
| TFTP     | 69   | None                          | No                |
| FTPS     | 990  | User/pass + cert              | Yes               |

HTTPS is the standard for production. TFTP is used only for closed factory-floor LANs.

### Per-MAC and per-model files

When the phone boots and contacts the configuration server, it downloads files in this sequence:

1. **`aastra.cfg`** — global config (legacy company name preserved for backward compat).
2. **`<model>.cfg`** — per-model overrides (e.g., `6867i.cfg`, `6920.cfg`).
3. **`<MAC>.cfg`** — per-phone overrides (MAC in lowercase, no separators, e.g., `0008565a1b2c.cfg`).

Later files override earlier ones. The MAC file is most-specific.

Polling cadence: configurable, default daily at random offset; `auto resync` parameter.

## Configuration File Format

Mitel config files are flat, line-oriented text (NOT strictly XML — confusing because the XML Browser is XML, but config files are key-value text).

Comment marker: `#`.

Parameters use a "category subcategory key" naming convention. Examples:

```
sip line1 user name: 5001
sip line1 auth name: 5001
sip line1 password: secret
sip line1 sip server: pbx.example.com
sip line1 sip port: 5060
sip line1 outbound proxy: sbc.example.com
sip line1 registration period: 3600

sip line2 user name: 5002
...

audio codec 1: g722
audio codec 2: pcmu
audio codec 3: pcma

time server1: pool.ntp.org
time zone name: America/New_York
```

XML-style configs (newer firmware) are also accepted and use this structure:

```xml
<AastraIPPhoneConfiguration>
  <PhoneSetting name="sip line1 user name" value="5001"/>
  <PhoneSetting name="sip line1 auth name" value="5001"/>
  <PhoneSetting name="sip line1 password" value="secret"/>
  ...
</AastraIPPhoneConfiguration>
```

Most installations stick with the flat key-value format because it's easier to template (Jinja, ERB, etc.).

## Sample aastra.cfg

A minimal global configuration:

```
# aastra.cfg — global configuration for all Mitel phones in the org

# ------------------- SIP Server -------------------
sip line1 enabled: 1
sip line1 sip server: pbx.example.com
sip line1 sip port: 5060
sip line1 outbound proxy: pbx.example.com
sip line1 outbound proxy port: 5060
sip line1 registration period: 3600
sip line1 backup proxy: pbx-backup.example.com
sip line1 backup proxy port: 5060

# ------------------- Codecs -------------------
audio codec 1: g722
audio codec 2: pcmu
audio codec 3: pcma
audio codec 4: g729
audio codec 1 ptime: 20
audio codec 2 ptime: 20
audio codec 3 ptime: 20
audio codec 4 ptime: 20

# ------------------- DTMF -------------------
sip dtmf method: 2          # 2 = RFC2833, 1 = inband, 3 = SIP INFO
sip session timer: 1800

# ------------------- Line Keys (default) -------------------
topsoftkey1 type: line
topsoftkey1 value: 1
topsoftkey1 label: Line 1

topsoftkey2 type: line
topsoftkey2 value: 2
topsoftkey2 label: Line 2

# ------------------- Time / NTP -------------------
time server1: pool.ntp.org
time server2: time.cloudflare.com
time server disabled: 0
time zone name: America/New_York
dst auto adjust: 1
time format: 0              # 0 = 12-hour, 1 = 24-hour

# ------------------- Network -------------------
qos ethernet priority: 5
qos rtp dscp: 46            # EF for voice
qos sip dscp: 26            # AF31 for signaling
lldp enabled: 1
lldp med enabled: 1

# ------------------- Provisioning -------------------
configuration server protocol: HTTPS
download protocol: HTTPS
auto resync time: 02:00
auto resync mode: 1         # 1 = check daily, 2 = on-boot only

# ------------------- Security -------------------
sip transport protocol: 4   # 4 = TLS, 1 = UDP, 2 = TCP, 0 = auto
sip srtp mode: 1            # 0 = off, 1 = best-effort, 2 = strict
web access enabled: 1
admin password: REPLACE_ME
user password: REPLACE_ME
```

## Sample MAC-specific cfg

Per-phone overrides, e.g. `0008565a1b2c.cfg` for the receptionist phone with MAC `00:08:56:5A:1B:2C`:

```
# 0008565a1b2c.cfg — Reception phone, M695 expansion module

# ------------------- Identity -------------------
sip line1 user name: 5000
sip line1 auth name: 5000
sip line1 password: <fetched from secrets store>
sip line1 display name: Reception
sip line1 screen name: Reception
sip line1 screen name 2: 5000

# ------------------- BLF List -------------------
sip line1 blf list uri: sip:reception-blf@pbx.example.com

# Programmable keys: leave keys 1-2 as Line, the rest as BLF/List
topsoftkey3 type: blflist
topsoftkey4 type: blflist
topsoftkey5 type: blflist
topsoftkey6 type: blflist

# ------------------- Expansion Module M695 (60 keys) -------------------
expmod1 enabled: 1
expmod1 model: m695

# Page 1 keys 1-20 — auto-populated by BLF List
expmod1 page1 key1 type: blflist
expmod1 page1 key2 type: blflist
expmod1 page1 key3 type: blflist
# ...

# Page 2 — speed dials to external partners
expmod1 page2 key1 type: speeddial
expmod1 page2 key1 value: 18005551212
expmod1 page2 key1 label: Vendor A
expmod1 page2 key2 type: speeddial
expmod1 page2 key2 value: 14165550199
expmod1 page2 key2 label: Vendor B

# ------------------- Custom Branding -------------------
background image: https://prov.example.com/branding/reception-bg.jpg
screensaver enabled: 1
screensaver wait time: 300
```

## CSP

**CSP (Configuration Server Protocol)** is Mitel's vendor-specific protocol for the redirector / zero-touch deploy flow. The redirector tells a phone where to go for its real config server.

The flow:

1. Phone boots, gets DHCP, has no provisioning URL.
2. Phone contacts Mitel's hosted CSP redirector at a well-known FQDN (varies by region; US: `csp.mitel.com`).
3. Phone sends its MAC, model, firmware version, and a CSP-encoded auth token (the token is provisioned by the reseller into Mitel's portal during order entry).
4. CSP redirector replies with the actual configuration server URL for the customer.
5. Phone redirects to that URL, fetches `aastra.cfg`, `<model>.cfg`, `<MAC>.cfg`.

This eliminates the need to pre-stage phones with provisioning URLs at the depot.

**GBP (Global Bootstrap Protocol)** is the lower-level transport used by CSP — handles the redirector handshake, signed responses, and recovery when the redirector is unreachable.

Customer-side CSP setup:

- Reseller creates a "site" in the Mitel CSP portal.
- MAC addresses uploaded under that site.
- Customer's provisioning server URL configured.
- Phones ship with CSP enabled by default.

If CSP is unreachable, the phone falls back to DHCP option 66 / 43 / 159 in that order. If those fail, the phone boots to the empty default config and shows the "No Service" screen.

## DHCP Options

Mitel phones honor several DHCP options for provisioning discovery:

| Option | Purpose                                                                                    |
|--------|--------------------------------------------------------------------------------------------|
| 66     | TFTP server name (fallback URL); accepts FQDN or IP, accepts http://, https:// URLs in modern firmware |
| 43     | Vendor-specific; vendor sub-options inside (Mitel sub-options 1, 2, 3...)                  |
| 60     | Vendor class identifier — phone advertises `Mitel-6867i`, `Mitel-6920`, etc.               |
| 159    | URL option — some firmware accepts a full HTTP(S) URL for config server                    |
| 160    | Provisioning URL (legacy; partial support)                                                 |

### Option 43 vendor-encap

Phone identifies itself with Option 60 (`Mitel-<model>`). DHCP server matches and returns Option 43 with sub-options:

| Sub-option | Meaning                                  |
|------------|------------------------------------------|
| 1          | Configuration server URL                 |
| 2          | TFTP server                              |
| 3          | Reserved                                 |
| 4          | Country code                             |
| ...        | (model-specific)                          |

Example dnsmasq snippet:

```
dhcp-vendorclass=set:mitel,Mitel
dhcp-option=tag:mitel,66,prov.example.com
dhcp-option=tag:mitel,43,01:1d:68:74:74:70:73:3a:2f:2f:70:72:6f:76:2e:65:78:61:6d:70:6c:65:2e:63:6f:6d:2f
# Sub-option 1 (0x01), length 0x1d (29), value "https://prov.example.com/"
```

ISC dhcpd:

```
option mitel-cfg-server code 43 = string;
class "mitel" {
  match if substring(option vendor-class-identifier, 0, 5) = "Mitel";
  option mitel-cfg-server "https://prov.example.com/";
  option tftp-server-name "prov.example.com";
}
```

Modern Mitel firmware (5.x+ on 6900 series) prefers Option 159 with a full URL:

```
option voip-config-url code 159 = string;
option voip-config-url "https://prov.example.com/mitel/";
```

## NAT Settings

For phones behind NAT (remote workers, branch offices without SBC), several toggles compensate for NAT traversal:

### STUN

```
sip stun server: stun.example.com
sip stun port: 3478
sip stun keepalive: 30
```

The phone discovers its public IP via STUN bind requests and uses it in Contact / SDP `c=` lines. Works for full-cone NAT. Symmetric NAT requires TURN/SBC.

### Outbound Proxy

The cleaner option: send everything through an SBC (MBG, OpenSIPS, Kamailio):

```
sip line1 outbound proxy: sbc.example.com
sip line1 outbound proxy port: 5060
```

The SBC handles NAT discovery and rewrites Contact headers.

### Keep-Alive

Phones send keep-alive packets to maintain NAT pinholes. Configurable per-line:

```
sip line1 keep alive type: 1     # 1 = OPTIONS, 2 = STUN, 3 = CR/LF (rfc 5626)
sip line1 keep alive interval: 30
```

30-second interval is common; 60-second ok for symmetric NAT with longer-lived bindings; 15 if the firewall is aggressive.

### NAT IP override

For deployments where STUN is blocked but a static public IP is known:

```
sip nat ip: 203.0.113.10
sip nat enabled: 1
```

The phone hard-codes 203.0.113.10 in Contact / Via.

### Force RFC 3581 rport

```
sip rport: 1                     # 1 = always include rport, 0 = never, 2 = auto
```

`rport` (RFC 3581) lets the registrar tell the phone which UDP port the phone's NAT box used. Force-on improves reliability behind firewalls that randomize source ports.

## SIP-over-TLS

To encrypt signaling, configure under **Audio → TLS Configuration** (terminology overlap — the TLS page covers SIP signaling TLS):

```
sip transport protocol: 4        # 4 = TLS
sip line1 sip port: 5061
sip line1 sip server tls verify: 1
```

Trusted certificates are installed under **TLS → Trusted Certificates**:

- Upload the PBX's CA chain (PEM format).
- Phone verifies the server cert against this chain during TLS handshake.

For mutual TLS (mTLS), upload an identity certificate (the phone's own client cert) under **TLS → Identity Certificate**:

- Either generated per-phone by the CSP / provisioning server, or generated on-phone via the CSR flow.
- The PBX validates the phone's client cert during handshake.

```
sip tls mutual auth: 1
sip tls cert: <embedded PEM or URL to fetch>
sip tls key: <embedded PEM or URL>
```

mTLS is required for some BroadSoft and MiVoice Connect deployments.

## SRTP

For media-plane encryption, configure under **Audio → SRTP**:

| Mode | Behavior                                                                  |
|------|---------------------------------------------------------------------------|
| 0    | Off — RTP only                                                            |
| 1    | SRTP enabled — best-effort; offers SRTP in SDP, accepts plain RTP if peer doesn't support |
| 2    | SRTP enabled, encrypted only — refuses calls if peer doesn't support SRTP |

```
sip srtp mode: 2
```

Key exchange:

- **SDES** (Session Description Protocol Security Descriptions, RFC 4568) — keys carried in SDP `a=crypto` attributes. Requires SIP-over-TLS for signaling (otherwise keys travel in cleartext).
- **DTLS-SRTP** (RFC 5764) — keys negotiated in-band over a DTLS handshake on the media channel. More modern; preferred for WebRTC interop.

Mitel firmware support varies:

- 6800 series — SDES only.
- 6900 series — SDES + DTLS-SRTP (firmware 5.0+).
- 5300i / 5320e re-flashed series — SDES only.

If the PBX advertises both, Mitel selects SDES by default unless `sip srtp dtls preferred: 1` is set.

## PoE / VLAN

Most Mitel phones support 802.3af PoE (class 2 typical, class 3 for premium models).

| Model    | Class    | Watts   |
|----------|----------|---------|
| 6863i    | Class 1  | 4.0 W   |
| 6865i    | Class 2  | 6.5 W   |
| 6867i    | Class 2  | 7 W     |
| 6869i    | Class 3  | 8 W     |
| 6905     | Class 1  | 4 W     |
| 6920     | Class 2  | 6 W     |
| 6930     | Class 3  | 9 W     |
| 6940     | Class 3  | 11 W    |
| 6970     | Class 3  | 12 W    |

External PSU available for non-PoE deployments (Mitel PSU 48VDC 0.65A typical).

VLAN configuration:

```
vlan id: 100                      # Voice VLAN
vlan priority: 5
data port vlan id: 200            # PC port (passthrough)
data port priority: 0
lldp enabled: 1
lldp med enabled: 1
```

LLDP-MED is the canonical voice-VLAN auto-discovery mechanism. The switch advertises voice VLAN ID and DSCP; the phone tags accordingly. If LLDP-MED is disabled or unsupported, the phone falls back to:

1. CDP (if **CAS Compatibility** is enabled — see next section).
2. DHCP Option 132 (VLAN ID).
3. Static configuration.

## CAS Compatibility

**CAS (Cisco-Auto-Sensing)** lets a Mitel phone speak CDP for voice-VLAN discovery in Cisco environments where LLDP-MED is disabled.

```
cdp enabled: 1
cdp interval: 60
```

Mitel phones can advertise themselves via CDP TLVs (Device-ID, Capabilities, Voice-VLAN-Query). The Cisco switch responds with CDP Voice VLAN advertisement. The phone tags accordingly.

This is rarely the right choice — LLDP-MED is the standards-based path — but exists for legacy Cisco IOS deployments without LLDP. Disable CAS in mixed-vendor environments to avoid spurious CDP traffic.

## Time / NTP

Time configuration lives under **Time and Date**:

```
time server1: 0.pool.ntp.org
time server2: 1.pool.ntp.org
time server3: 2.pool.ntp.org
time server disabled: 0           # 0 = enabled
time zone name: America/Los_Angeles
time format: 1                    # 1 = 24h, 0 = 12h
date format: WWW MMM DD
dst auto adjust: 1                # 1 = follow time zone DST rules
dst start month: 3
dst start day: 8
dst start hour: 2
dst end month: 11
dst end day: 1
dst end hour: 2
```

Time zone selection uses Olson tz names (`America/New_York`, `Europe/London`, `Asia/Tokyo`).

DST rules are auto-derived from the tz name when `dst auto adjust: 1`. Manual override is possible by setting `dst auto adjust: 0` and filling start/end fields.

NTP polling interval: configurable via `time server poll interval` (default 3600 sec). On boot, an immediate sync attempt happens; failures retry with backoff.

Wrong time is a leading cause of TLS cert validation failures — see Common Gotchas.

## Web UI Authentication

Three account tiers:

| Tier   | Username | Default | Capabilities                                       |
|--------|----------|---------|---------------------------------------------------|
| Admin  | admin    | 22222   | Full config access                                 |
| User   | user     | 1111    | Limited (call forward, ringtones, time)            |
| Guest  | guest    | (none)  | Read-only Status pages                             |

Account lockout settings under **Operation → Web Access**:

```
web access enabled: 1
web access port: 80
web access port https: 443
admin password: <new>
user password: <new>
guest password: <new>
account lockout count: 5          # Failed attempts before lockout
account lockout duration: 600     # Seconds
account lockout reset time: 1800  # Seconds before lockout count resets
```

Disabling guest:

```
guest password: <empty>           # Empty disables guest
```

For HTTPS-only:

```
http enabled: 0
https enabled: 1
```

## Logging

Mitel phones support local and remote logging.

### Logging Levels

```
log level: 4                      # 0 = emerg, 7 = debug
syslog server: syslog.example.com
syslog port: 514
syslog protocol: 0                # 0 = UDP, 1 = TCP
```

Per-component log levels (firmware 5.x+):

```
log level sip: 4
log level audio: 3
log level provisioning: 5
log level network: 3
log level web: 3
log level xml: 4
```

### Web Diagnostics page

Under **Status → Diagnostic Information** (older firmware) or **Audio Diagnostics → Logs** (newer):

- Live tail of phone logs in browser.
- Download last 24h log archive.
- Search by component / level.

Useful for debugging registration, codec negotiation, NAT, TLS handshake without leaving the web UI.

## PCAP

The web UI exposes a built-in packet capture utility under **Audio Diagnostics → Pcap**:

1. Select interface (LAN, PC port, or both).
2. Select duration (10s, 30s, 60s, 5min, 10min).
3. Optional BPF filter (e.g., `port 5060 or port 5061`).
4. Click "Start Capture."
5. After completion, download the resulting `.pcap` file.

The capture is written to the phone's RAM filesystem; large captures (>10MB) may fail on memory-constrained 6800 series.

For longer captures, mirror the switch port and capture upstream — but the on-phone PCAP catches loopback / pre-encryption traffic that an external mirror misses.

## Phone Lock

The **Phone Lock** feature provides emergency-only call restriction. Useful for shared workspaces, hotelling, lobby phones.

```
phone lock enabled: 1
phone lock pin: 1234
phone lock emergency dial: 911,112,999
phone lock auto lock idle time: 600
```

When locked:

- Outgoing calls only to numbers in `emergency dial` list.
- Inbound calls still ring; can be answered.
- Settings menu inaccessible.
- Soft key shows "Unlock" prompt.

Unlock by entering PIN (4-digit configurable). PIN attempts are rate-limited.

Lock-on-idle: auto-lock after `auto lock idle time` seconds.

For hotelling: combine Phone Lock with the **Hot Desk** feature (a key sequence that logs the user in and pulls their per-user config from the PBX).

## Custom Branding

Visual customization under **Custom Branding**:

| Asset           | Format     | Resolution         | Models                       |
|-----------------|------------|--------------------|--------------------------------|
| Wallpaper       | JPG / PNG  | 360×218 (6867i), 480×272 (6869i, 6920), 800×480 (6940/6970) | All color models |
| Logo            | PNG (alpha)| 120×40 to 200×60   | Color models                   |
| Screensaver     | JPG/PNG    | Per-model res      | All models                     |
| Idle Background | JPG/PNG    | Per-model res      | 6900 series                    |
| Boot Logo       | BMP / PNG  | 192×64 (6863i)     | Monochrome models              |

Files are referenced via URL in the config:

```
background image: https://prov.example.com/branding/wallpaper.jpg
screensaver image: https://prov.example.com/branding/saver.png
boot logo: https://prov.example.com/branding/logo.png
```

The phone caches images locally; updates pulled on next provisioning sync.

Branding can be model-conditional via `<model>.cfg` files.

## ACD / Call Center

For contact-center deployments (MiContact Center or BroadWorks ACD), Mitel phones support agent state keys.

Programmable key types for ACD:

- **ACD Login** — log agent in.
- **ACD Logout** — log out.
- **Auto-In** — accept next call automatically.
- **Manual-In** — pause between calls.
- **Wrap-Up** — post-call work; not available for next call.
- **Available** — generic ready state.
- **Unavailable** — break / lunch / training.
- **Reason Codes** — sub-states for unavailable (configurable per code).

```
topsoftkey5 type: acd
topsoftkey5 value: login
topsoftkey5 label: Login

topsoftkey6 type: acd
topsoftkey6 value: logout
topsoftkey6 label: Logout

topsoftkey7 type: acd
topsoftkey7 value: wrap_up
topsoftkey7 label: Wrap

topsoftkey8 type: acd
topsoftkey8 value: unavail
topsoftkey8 label: Break
```

Integration with MiContact Center:

- Phone subscribes to `Event: as-feature-event` (BroadWorks) or `Event: x-mitel-acd-event` (MiContact Center).
- PBX pushes NOTIFYs as agent state changes (e.g., supervisor force-logs the agent).
- Phone updates lamp state on the ACD keys.

For BroadSoft, the **XSI** integration (Xtended Services Interface) exposes ACD via REST in addition to SIP NOTIFYs.

## MiVoice Border Gateway

**MBG** is Mitel's combined SBC + remote-worker proxy + reverse proxy. It sits in the DMZ and brokers SIP traffic between internal MiVoice Business / 5000 and remote endpoints (phones, MiCollab clients, soft phones).

Topology for remote workers:

```
[Mitel phone @ home] -- TLS+SRTP --> [MBG @ DMZ] -- internal SIP --> [MiVoice Business]
```

Phone configuration when registering through MBG:

```
sip line1 sip server: mbg.example.com
sip line1 sip port: 5061
sip transport protocol: 4         # TLS
sip srtp mode: 2                  # encrypted only
sip line1 outbound proxy: mbg.example.com
sip line1 outbound proxy port: 5061
```

MBG capabilities:

- TLS termination toward phones; TCP/UDP toward PBX.
- SRTP termination; plain RTP toward PBX (or pass-through if PBX speaks SRTP).
- Per-MAC allow-list (only registered phones permitted).
- Built-in proxy for HTTPS provisioning (phone fetches config through MBG's web layer).
- Active-active clustering for HA.

MBG also handles **Teleworker** (MiCollab Client / MiVoice Connect Mobility) — softphone clients use MBG as their auth + media relay endpoint.

## Encryption

For protecting the configuration files in transit and at rest on the provisioning server, Mitel supports **AES-256 config-file encryption**.

The flow:

1. Provisioning server encrypts `<MAC>.cfg` with AES-256 using a per-MAC key derived from a master key.
2. Phone fetches encrypted file (filename suffix `.tuz` or `.cfg.enc`).
3. Phone derives the AES key from its embedded master key + MAC.
4. Phone decrypts and applies.

The master key is delivered via vendor-specific channels:

- Pre-shared at the factory and identified by Mitel CSP portal.
- Or pre-installed by the integrator via an unencrypted "bootstrap" config that includes `aastra cfg encrypt key: <hex>`.

Mitel's **anatool** (provided to channel partners) generates encrypted configs from plain templates.

Use cases:

- Storing configs on a public S3 bucket.
- Provisioning over untrusted HTTP (avoid this anyway — use HTTPS).
- Multi-tenant providers where config files transit shared infrastructure.

If the master key is lost, the phone cannot boot the encrypted config and must be factory-reset. Always keep an unencrypted recovery path (e.g., a separate provisioning URL via DHCP that returns plain config).

## Common Errors

Verbatim error messages and their canonical causes:

### "Registration Failed: 401 Unauthorized"

PBX rejected REGISTER. Causes:
- Wrong `auth name` or `password`.
- PBX expects a different auth realm.
- For BroadSoft, the line ID URL doesn't match the phone's claimed AOR.

Fix: verify auth username/password against PBX directory. For MiVoice Business, check that the line/extension is bound to the phone's MAC.

### "Registration Failed: 403 Forbidden"

PBX is reachable and recognizes the credentials, but is blocking the registration.
- Phone not yet provisioned in PBX.
- IP allow-list rejects phone's source IP.
- Account locked or disabled.
- Trying to register over UDP when PBX requires TLS.

Fix: check PBX admin logs for the matching 403; usually a `Reason:` header explains.

### "Registration Failed: 408 Request Timeout"

PBX never responded within the timeout (default 32s).
- Wrong sip server address (DNS resolves to wrong host).
- Firewall drops outbound 5060/5061.
- PBX overloaded.

Fix: ping/traceroute the SIP server from another machine on the phone's VLAN; check firewall.

### "DNS Resolution Failed"

Phone cannot resolve `sip server` FQDN.
- DHCP didn't deliver DNS server address.
- DNS server unreachable.
- FQDN typo.

Fix: check Status → Network for resolved DNS servers; nslookup from another host.

### "Audio Codec Negotiation Failed"

INVITE/200 OK exchange completed but no common codec.
- Phone's codec list and PBX's codec list don't intersect.
- Phone offered SRTP but PBX wanted plain RTP (with SRTP strict mode).

Fix: enable G.711U on both sides as the universal fallback. Set SRTP to mode 1 (best-effort) during troubleshooting.

### "TLS Handshake Failed"

TLS connection to sip server failed.
- Server cert not trusted (CA chain not installed in TLS → Trusted Certificates).
- Server cert hostname doesn't match `sip server` value.
- Phone's clock is wrong (cert "not yet valid" or "expired").
- Server requires mTLS but phone has no identity cert.

Fix: install full CA chain; verify time/NTP; if mTLS, upload identity cert.

### "Provisioning: Auth Failed"

HTTP 401/403 from configuration server.
- Wrong provisioning user/password.
- Provisioning server requires per-MAC auth and the phone's auth doesn't match.

Fix: check provisioning server logs; for HTTPS Basic, set `configuration server username` and `configuration server password`.

### "Provisioning: 404 Not Found"

Configuration file not found at expected URL.
- aastra.cfg / `<model>.cfg` / `<MAC>.cfg` missing on server.
- URL path wrong.
- Per-MAC file casing mismatch (Mitel uses lowercase MAC; many provisioning systems use uppercase).

Fix: ls the provisioning directory; ensure both `aastra.cfg` exists and the per-MAC file with lowercase MAC.

### "Network Time Sync Failed"

NTP failed.
- NTP server unreachable.
- Firewall blocks UDP/123.
- Phone's clock is so far off the NTP server rejects (NTP `step threshold`).

Fix: verify NTP server reachable; pre-set time manually if drift is large.

### "Unknown Caller"

Inbound call has empty or unrecognized From display name.
- PBX strips P-Asserted-Identity / Privacy header.
- Caller anonymous (Privacy: id).
- LDAP/Directory lookup failed.

Fix: configure directory integration if using XML phone book; for anonymous calls, this is expected.

## Common Gotchas

### Default password 22222 not changed

```
admin password: 22222              # broken — default vendor pw
admin password: <strong unique>    # fixed
```

22222 is the most-attacked default password on Mitel phones. Phones reachable from corporate Wi-Fi or guest networks get scanned and pwned. Always rotate at provisioning time.

### HTTP not HTTPS provisioning URL — credentials in plain

```
configuration server protocol: HTTP                 # broken
download protocol: HTTP

configuration server protocol: HTTPS                # fixed
download protocol: HTTPS
```

Config files contain SIP passwords, voicemail PINs, sometimes AES master keys. Over HTTP these travel cleartext on the wire. Always HTTPS in production.

### Wrong DHCP option for vendor-encap

```
# broken — using sub-option 0x42 (66) inside Option 43 like Cisco
dhcp-option=tag:mitel,43,42:0a:70:72:6f:76...

# fixed — Mitel uses sub-option 0x01 inside Option 43
dhcp-option=tag:mitel,43,01:1d:68:74:74:70:73:3a:2f:2f:70:72:6f:76:2e:65:78:61:6d:70:6c:65:2e:63:6f:6d:2f
```

Mitel sub-option 1 carries the URL; Cisco uses sub-option 0x42 in vendor space. Mixing them up means the phone never reads the URL.

### Multiple lines registered to same upstream — duplicate REGISTER

```
sip line1 user name: 5001          # broken — both lines auth as 5001
sip line2 user name: 5001
sip line1 sip server: pbx.example.com
sip line2 sip server: pbx.example.com

sip line1 user name: 5001          # fixed — distinct accounts
sip line2 user name: 5002
```

If the PBX sees two REGISTERs for the same AOR from the same Contact, behavior is undefined: some PBXs accept (latest wins), some reject 403, some race.

### BLF subscribed but server doesn't publish (event=dialog)

```
# Phone log:
SUBSCRIBE sip:5001@pbx.example.com SIP/2.0
Event: dialog

# Server returns 489 Bad Event or never NOTIFYs.
```

Server doesn't support the `dialog` event package, or the user/extension isn't configured for presence publishing. On Asterisk, `hint exten => 5001,hint,SIP/5001` must be set in dialplan. On FreeSWITCH, `mod_dialog` enabled and the contact configured for presence.

Fix the server-side; the phone is doing the right thing.

### 6900 series uses different firmware family than 6800 — config not portable

```
# 6867i firmware: 4.2.x
# 6920 firmware:  5.x.x

# A config that uses 6800-series-only parameters fails on 6900:
sip line1 retransmission timer: 500   # 6800 only
```

The config schemas overlap mostly but diverge on advanced features (Bluetooth pairing, USB-C HID, touchscreen UI). Maintain separate `<model>.cfg` for 6800 vs 6900 series. Don't assume `aastra.cfg` covers both.

### DECT cordless 5614 needs base + handset firmware match

```
# Base RFP-32: firmware 6.x
# Handset 5614: firmware 5.x

# Result: handset registers but calls drop after 30 seconds, audio one-way.
```

DECT firmware is a tightly-coupled pair: handset and base must match (within a known compatibility matrix). After base firmware upgrade, all paired handsets need OTA upgrade before normal operation resumes.

### Time zone wrong — cert validation fails

```
time zone name: UTC                # phone in California: shows time 8h ahead
# TLS handshake to PBX cert valid 2024-01-01 to 2025-01-01:
# Phone thinks it's 2025-01-01 02:00 UTC = 2024-12-31 18:00 PST
# Borderline — sometimes works, sometimes "cert not yet valid"

time zone name: America/Los_Angeles  # fixed
dst auto adjust: 1
time server1: pool.ntp.org
```

Wrong tz combined with wrong NTP gives random-looking TLS failures. Always set tz before bringing up TLS.

### Codec list order matters; first match wins

```
# Phone offers codecs in this SDP order:
audio codec 1: g729                # broken — narrowband first
audio codec 2: pcmu
audio codec 3: g722

# PBX picks G.729 → low-quality calls despite G.722 supported on both ends.

audio codec 1: opus                # fixed — wideband first
audio codec 2: g722
audio codec 3: pcmu
audio codec 4: pcma
audio codec 5: g729
```

Most PBXs honor the offer order and pick the first match. Put HD codecs first for HD voice; put G.711 high enough to be the fallback.

### LLDP-MED disabled on switch — wrong VLAN

```
# Phone boots, gets DHCP on data VLAN (200), can't reach PBX (on voice VLAN 100).
# Phone log: "Network up; cannot reach SIP server"
```

LLDP-MED on the switch port is required for the phone to learn voice VLAN 100. If disabled, the phone stays on the native data VLAN. Either:

1. Enable LLDP-MED on the switch port (Cisco: `lldp run` + `network-policy profile`).
2. Statically configure the VLAN on the phone (`vlan id: 100`).
3. Use DHCP Option 132.

LLDP-MED is the canonical solution; static VLAN means re-touching every phone if the VLAN scheme changes.

### Vendor encrypted config without key — boot fails

```
# Phone fetches 0008565a1b2c.cfg.enc but has no master key.
# Boot loops; LCD shows "Provisioning Error: Decryption Failed"
```

If the config file is AES-encrypted but the phone never received the master key (lost during stock rotation, or phone was wiped and rejoined a different tenant), it cannot decrypt. Have a recovery URL that delivers an unencrypted bootstrap config containing the new key.

### Migration from MiNet 5300 to SIP firmware 5300i — admin password reset

```
# 5320 (MiNet) admin password: <whatever it was>
# Reflash to 5320e SIP firmware
# After reflash, admin password = 22222 (factory default)
```

Cross-firmware reflash factory-resets all settings including credentials. Plan for a re-provisioning sweep after MiNet→SIP migration. Don't rely on prior admin password persisting.

## Diagnostic Tools

### Web UI Audio Diagnostics

Tone test, microphone test, audio loopback, packet capture, audio statistics for current call (R-factor, MOS-LQO, jitter, packet loss).

### Status → Auto-discovery info

Shows which provisioning method discovered the config server: CSP / Option 66 / Option 43 / Option 159 / static.

### PCAP from web UI

See PCAP section. Download as `.pcap`, open in Wireshark.

### Syslog forwarding

Configure syslog server; correlate against PBX logs by Call-ID.

### Network ping/traceroute from phone

Under **Audio Diagnostics → Network Tools**:

```
Ping: pbx.example.com
Traceroute: pbx.example.com
DNS lookup: pbx.example.com
```

Useful for confirming reachability without logging into the phone OS shell (which doesn't exist publicly).

## Sample Cookbook

### Asterisk integration

```
; pjsip.conf
[5001]
type=endpoint
context=default
disallow=all
allow=opus
allow=g722
allow=ulaw
auth=5001-auth
aors=5001
direct_media=no
rtp_symmetric=yes
force_rport=yes
rewrite_contact=yes
trust_id_inbound=yes

[5001-auth]
type=auth
auth_type=userpass
username=5001
password=<secret>

[5001]
type=aor
max_contacts=2
remove_existing=yes
```

Phone:

```
sip line1 line mode: asterisk
sip line1 user name: 5001
sip line1 auth name: 5001
sip line1 password: <secret>
sip line1 sip server: asterisk.example.com
sip line1 sip port: 5060
sip transport protocol: 1            # UDP
audio codec 1: opus
audio codec 2: g722
audio codec 3: pcmu
```

For BLF on Asterisk:

```
; extensions.conf
exten => 5001,hint,PJSIP/5001
exten => 5002,hint,PJSIP/5002
```

Phone BLF key:

```
topsoftkey3 type: blf
topsoftkey3 value: 5002
topsoftkey3 line: 1
topsoftkey3 label: Bob
```

### FreeSWITCH integration

```
<!-- conf/directory/default/5001.xml -->
<include>
  <user id="5001">
    <params>
      <param name="password" value="<secret>"/>
    </params>
    <variables>
      <variable name="user_context" value="default"/>
      <variable name="effective_caller_id_number" value="5001"/>
    </variables>
  </user>
</include>
```

Phone:

```
sip line1 line mode: generic
sip line1 user name: 5001
sip line1 auth name: 5001
sip line1 password: <secret>
sip line1 sip server: freeswitch.example.com
sip line1 sip port: 5060
```

For BLF List on FreeSWITCH (mod_dialog):

```
<!-- mod_event_subscribe handles dialog event-list -->
sip line1 blf list uri: sip:reception-blf@freeswitch.example.com
```

### 3CX integration

3CX has native Mitel support. Add the phone by MAC in 3CX Admin → Phones; 3CX generates per-MAC config and serves it via its built-in provisioning server.

Phone CSP / DHCP discovers the 3CX provisioning URL. Zero on-phone config beyond the URL.

```
configuration server protocol: HTTPS
configuration server: https://3cx.example.com:5001/provisioning/
```

3CX-specific quirks:

- 3CX uses TCP for SIP by default; set `sip transport protocol: 2`.
- Set `sip line1 line mode: 3cx` (firmware 5.x+).
- BLF List works against 3CX's "Park Slot" extensions.

### BroadWorks integration

```
sip line1 line mode: broadsoft
sip line1 user name: 5001@example.com
sip line1 auth name: bw_user_5001
sip line1 password: <secret>
sip line1 sip server: bsft.example.com
sip line1 sip port: 5060
sip line1 outbound proxy: sbc.example.com

# BroadSoft XSI for ACD and directory
xsi server: xsi.example.com
xsi port: 443
xsi protocol: HTTPS
xsi user: bw_user_5001
xsi password: <secret>

# BroadSoft BLF List
sip line1 blf list uri: sip:5001-blf@example.com
```

BroadSoft features unlocked by `line mode: broadsoft`:

- Server-side call logs accessible from phone Directory.
- Shared Call Appearance (SCA) for executive/assistant.
- Hot Desk via XSI.
- Unified call history.

## Hardware Specifics

### 6900 series

- **Gigabit Ethernet** — 10/100/1000 PHY on LAN port. PC port also gigabit on 6920+; older models 100Mbit on PC port.
- **USB-C** — 6970 only; USB-C for headset, USB hub, charging accessories.
- **USB-A** — 6920/6930/6940/6970 have USB-A on the side (headsets, USB drives for PCAP export, Bluetooth dongle on older model).
- **Bluetooth** — built-in on 6930, 6940, 6970. Pair Bluetooth headsets natively.
- **Personal Wireless Pairing (PWP)** — Mitel's BLE-based pairing for 6940's integrated DECT cordless option (sold as 6940 + DECT handset bundle).

### Mi-USB headset port

Premium models (6940, 6970) have a dedicated **Mi-USB** port — USB-A with extra pins for the DHSG (Direct Hookswitch Group) signaling required by EHS-capable headsets (Plantronics APD, Jabra GN1000, Sennheiser CEHS).

```
Mi-USB pinout: USB-A pins + EHS DHSG +5V, ground, ring, hookswitch
```

This means a Plantronics CS540 wireless headset can answer/hang up calls from the headset button — the phone listens for the EHS signal on Mi-USB.

### 6900 vs 6800 hardware differences

- 6900 has higher-res LCDs, faster CPU (ARM Cortex-A53 vs older A9), more RAM (256–512MB vs 128MB).
- 6900 supports Opus codec; 6800 stops at G.722.
- 6900 has on-device DTLS-SRTP; 6800 stops at SDES-SRTP.
- 6900 has full HD voice everywhere (handset, headset, speaker); 6800 is HD on handset only.

## Firmware Channels

Mitel publishes firmware in three channels:

- **Release** — production-stable, GA. Passes Mitel's full QA matrix. ~quarterly cadence.
- **Beta** — feature-preview. Available to channel partners. Used for early-adopter MiVoice Business sites.
- **Patches** — security and critical-bug hotfixes. Off-cycle, pushed to release channel devices.

### Per-region firmware variants

Some firmware is region-specific due to:

- Tone plans (dial tone, busy, ringback) — UK, EU, NA, ANZ, JP.
- Language packs — UI translations.
- Regulatory codecs — China G.722.1C, India custom.

Region is set via `country code` parameter:

```
country code: US                  # or UK, DE, FR, JP, AU, ...
```

Wrong region → wrong dial tones, sometimes wrong ringback cadence, cosmetic but noticeable.

Firmware filename convention:

```
firmware_6867i_release_4.3.0.5099.tar.gz
firmware_6920_beta_5.1.0.42.tar.gz
firmware_6940_patch_5.0.2.114-cve-2024-12345.tar.gz
```

Hosted on `productdocuments.mitel.com` and `mitel.custhelp.com` for partners.

## Idioms

- **Always change default 22222 password** — first thing at provisioning. Embed in `aastra.cfg` template.
- **Use HTTPS for provisioning** — never HTTP in production. Configs contain credentials.
- **MiNet for legacy Mitel-only deployments** — if the only PBX is MiVoice Business / 3300 ICP and stays that way, keep MiNet. Otherwise re-flash to SIP.
- **SIP firmware for everything else** — Asterisk, FreeSWITCH, 3CX, BroadWorks, Cisco UCM (with limitations), MX-ONE.
- **MBG for remote workers** — don't expose MiVoice Business directly to the internet. MBG handles TLS/SRTP termination, NAT, and MAC allow-listing.
- **BLF List for receptionist setups** — single subscription scales to 50 monitored extensions; one SUBSCRIBE per phone.
- **Expansion modules chain via RJ-12** — daisy-chain up to 3, the last needs an external PSU.
- **Set time before TLS** — wrong clock kills cert validation. NTP first, TLS second.
- **Codec order = quality order** — put HD codec first; the offer order drives PBX selection.
- **Per-model `<model>.cfg`** — don't assume 6800 and 6900 share a config; some parameters diverge.
- **Lowercase MAC in filenames** — Mitel firmware fetches `<lowercase mac>.cfg`. Match exactly on Linux filesystems.
- **CSP for zero-touch, DHCP fallback for closed networks** — CSP is the easy path; closed factories use DHCP Option 43/66/159.
- **Mi-USB for EHS headsets** — wireless headsets answer/hang up only via the Mi-USB DHSG signaling, not plain USB.

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

## References

- productdocuments.mitel.com — per-model admin guides (6867i, 6869i, 6920, 6930, 6940, 6970, M695, M680)
- Mitel SIP Phone Configuration Guide — global parameter reference (covers aastra.cfg / `<MAC>.cfg` schemas)
- Mitel CSP Specification — Configuration Server Protocol; redirector, GBP, and zero-touch provisioning details
- Mitel MiVoice Border Gateway Engineering Guidelines — MBG topology, TLS/SRTP termination, teleworker setup
- Mitel MiContact Center Solutions Guide — ACD agent state, key/lamp protocol, XSI integration
- RFC 3261 — SIP base
- RFC 3581 — symmetric response routing (rport)
- RFC 4235 — dialog event package (BLF)
- RFC 4568 — SDES (SDP Security Descriptions for SRTP)
- RFC 4662 — event-list (BLF List)
- RFC 5764 — DTLS-SRTP
- RFC 6665 — SIP-specific event notification
- IEEE 802.3af — Power over Ethernet
- ANSI/TIA-1057 — LLDP-MED
