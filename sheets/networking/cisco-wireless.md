# Cisco Wireless (WLC, CAPWAP, and Enterprise Wi-Fi)

Enterprise wireless LAN architecture centered on Cisco Wireless LAN Controllers (WLC) managing lightweight access points via CAPWAP, providing centralized RF management, client roaming, security policy enforcement, and integration with Cisco DNA Center for intent-based wireless networking across 2.4 GHz, 5 GHz, and 6 GHz bands.

## 802.11 Standards

### Standards Evolution

| Standard   | Wi-Fi Name | Band(s)          | Max Rate     | Channel Width   | MIMO          | Key Technology                     |
|------------|------------|------------------|--------------|-----------------|---------------|------------------------------------|
| 802.11b    | —          | 2.4 GHz          | 11 Mbps      | 22 MHz          | None          | DSSS/CCK                           |
| 802.11a    | —          | 5 GHz            | 54 Mbps      | 20 MHz          | None          | OFDM                               |
| 802.11g    | —          | 2.4 GHz          | 54 Mbps      | 20 MHz          | None          | OFDM (backward compat with b)      |
| 802.11n    | Wi-Fi 4    | 2.4 / 5 GHz     | 600 Mbps     | 20/40 MHz       | 4x4 MIMO      | HT (High Throughput), A-MPDU       |
| 802.11ac   | Wi-Fi 5    | 5 GHz            | 6.93 Gbps    | 20/40/80/160 MHz| 8x8 MU-MIMO   | VHT, beamforming, 256-QAM          |
| 802.11ax   | Wi-Fi 6/6E | 2.4/5/6 GHz     | 9.6 Gbps     | 20/40/80/160 MHz| 8x8 MU-MIMO   | HE, OFDMA, TWT, BSS Coloring       |
| 802.11be   | Wi-Fi 7    | 2.4/5/6 GHz     | 46 Gbps      | Up to 320 MHz   | 16x16 MU-MIMO | EHT, MLO, 4K-QAM, preamble punct.  |

### Key Technology Concepts

```
OFDM (Orthogonal Frequency Division Multiplexing):
  - Divides channel into multiple narrow subcarriers
  - Each subcarrier carries part of the data
  - Resistant to multipath interference
  - Used in 802.11a/g/n/ac

OFDMA (Orthogonal Frequency Division Multiple Access):
  - 802.11ax/be: subdivides channel into Resource Units (RUs)
  - Multiple clients transmit/receive simultaneously on different RUs
  - Dramatically improves dense environment performance
  - Uplink and downlink OFDMA

MU-MIMO (Multi-User Multiple Input Multiple Output):
  - 802.11ac Wave 2: downlink MU-MIMO (up to 4 clients)
  - 802.11ax: uplink + downlink MU-MIMO (up to 8 clients)
  - Beamforming steers signals toward specific clients

TWT (Target Wake Time) — 802.11ax:
  - AP schedules wake times for each client
  - Reduces contention and saves battery (IoT devices)
  - Clients sleep until their scheduled TWT window

BSS Coloring — 802.11ax:
  - Tags frames with a 6-bit color identifier per BSS
  - Clients ignore frames from different-colored BSSes
  - Increases spatial reuse in dense deployments

MLO (Multi-Link Operation) — 802.11be:
  - Single client aggregates traffic across multiple bands simultaneously
  - Example: 2.4 + 5 + 6 GHz used concurrently for one connection
  - Reduces latency, increases throughput, improves reliability
```

## WLC Architecture

### Deployment Models

```
Centralized WLC (traditional):
  - Dedicated hardware appliance or Catalyst 9800 (physical)
  - All CAPWAP tunnels terminate at WLC
  - Centralized data plane: all client data traverses WLC
  - Best for: campus with WLC in data center or MDF
  - Platforms: 9800-40, 9800-80, 9800-L, 5520, 8540

Catalyst 9800 Embedded Wireless (EWC):
  - WLC software runs on a Catalyst 9000 switch
  - Switch acts as both access switch and WLC
  - Supports up to 200 APs (model dependent)
  - Best for: small/medium sites without dedicated WLC hardware
  - Platform: Catalyst 9300/9400/9500 with EWC license

Mobility Express (ME):
  - WLC software runs directly on an AP
  - One AP elected as master (WLC), rest are subordinate
  - Supports up to 100 APs
  - Best for: small branch offices, retail locations
  - Platform: Aironet 1815, 1840, 2800, 3800, Catalyst 9100

Cloud-Managed (Meraki / Catalyst Center):
  - WLC function hosted in Cisco cloud (Meraki Dashboard)
  - APs connect to cloud for management, policy, monitoring
  - Data plane remains local (no data tunneled to cloud)
  - Best for: distributed sites, MSP-managed networks

FlexConnect (deployment mode, not WLC type):
  - APs can locally switch traffic even if WLC link fails
  - CAPWAP control to WLC, data switched locally at AP
  - Best for: remote offices with unreliable WAN to WLC
  - Can operate in standalone mode during WLC outage
```

### Catalyst 9800 WLC Platform

```
Platform comparison:
  Model         APs    Clients   Throughput   Form Factor
  9800-L        250    5,000     5 Gbps       1RU appliance
  9800-40       2,000  32,000    40 Gbps      1RU appliance
  9800-80       6,000  64,000    80 Gbps      2RU appliance
  9800-CL       6,000  64,000    Varies       VM (ESXi/KVM/cloud)
  EWC on 9300   200    4,000     N/A          Embedded on switch

Operating system: IOS-XE (same as Catalyst switches)
Management: CLI, Web UI, NETCONF/YANG, RESTCONF, DNA Center
HA: Stateful Switchover (SSO) — sub-second failover
  - Active/Standby pair with redundancy port or network-based HA
  - Client sessions, AP joins, RRM state all synchronized
  - N+1 HA also supported (one standby for multiple active WLCs)
```

## CAPWAP Protocol

### Architecture

```
CAPWAP (Control And Provisioning of Wireless Access Points):
  RFC 5415 / 5416
  Runs over UDP
  Two tunnels between AP and WLC:

  Control tunnel:
    UDP port 5246
    DTLS encrypted (mandatory)
    Carries: AP join, configuration, firmware, client auth events
    Keepalive: every 30 seconds (default)
    Dead peer detection: 5 missed keepalives = AP failover

  Data tunnel:
    UDP port 5247
    DTLS encryption optional (disabled by default for performance)
    Carries: 802.11 frames encapsulated in CAPWAP
    Only used in centralized (local) mode — not FlexConnect local switching

  MTU requirement: 1485 bytes minimum (CAPWAP overhead ~58-72 bytes)
  Fragmentation: CAPWAP fragments if path MTU < frame size
```

### AP Discovery and Join Process

```
AP boot sequence:
  1. AP powers on (PoE or local power)
  2. Obtains IP via DHCP (or static config)
  3. Discovers WLC using (in order):
     a. Previously saved (primed) WLC IP
     b. DHCP Option 43 (vendor-specific WLC IP list)
     c. DNS resolution of CISCO-CAPWAP-CONTROLLER.<domain>
     d. Local subnet broadcast (255.255.255.255:5246)
     e. Over-the-air provisioning (from neighboring APs)

  4. AP sends Discovery Request to each discovered WLC
  5. WLCs respond with Discovery Response (capacity, load, type)
  6. AP selects WLC (primary > secondary > tertiary > least loaded)
  7. DTLS handshake establishes encrypted control channel
  8. AP sends Join Request (model, serial, capabilities, certificates)
  9. WLC validates AP certificate (MIC or LSC)
  10. WLC sends Join Response (accepted or rejected)
  11. AP downloads configuration from WLC
  12. AP downloads firmware if version mismatch (auto image download)
  13. AP reboots with new firmware if needed
  14. AP enters RUN state — begins serving clients

Certificate types:
  MIC (Manufacturing Installed Certificate): factory-installed, default
  LSC (Locally Significant Certificate): customer CA-issued, more secure
```

## AP Modes

### Operating Modes

```
Local mode (default):
  - Normal client-serving operation
  - All data tunneled to WLC via CAPWAP (centralized switching)
  - AP performs off-channel scanning (brief 60ms scans) for RRM
  - Best for: campus deployments with reliable WLC connectivity

FlexConnect mode:
  - AP can locally switch data traffic (bypass WLC for data plane)
  - Control plane still via CAPWAP to WLC
  - Two sub-modes per WLAN:
    Central switching: data tunneled to WLC (like local mode)
    Local switching:  data bridged directly at AP to local VLAN
  - Standalone mode: serves clients even if WLC link is lost
  - FlexConnect Group: share CCKM/OKC cache for fast roaming
  - Best for: remote/branch offices

Monitor mode:
  - AP does not serve clients
  - Dedicated full-time scanning on all channels
  - Used for wIPS (wireless intrusion prevention)
  - Detects rogue APs, ad-hoc networks, DoS attacks
  - Reports to WLC/DNA Center for correlation and alerting

Sniffer mode:
  - AP captures raw 802.11 frames on a specified channel
  - Forwards captures to a remote machine running Wireshark
  - Encapsulated in TZSP or Peekremote format
  - Used for deep packet-level wireless troubleshooting

Bridge mode (mesh):
  - AP acts as a wireless bridge between wired segments
  - Root AP (RAP): connected to wired network
  - Mesh AP (MAP): connects wirelessly to RAP
  - AWPP (Adaptive Wireless Path Protocol) for mesh path selection
  - Best for: outdoor deployments, warehouse, campus interconnects

SE-Connect (Spectrum Expert Connect):
  - Dedicated spectrum analysis mode
  - AP becomes a spectrum sensor
  - Streams raw FFT data to Cisco Spectrum Expert or DNA Center
  - Used for detailed RF interference analysis
  - Cannot serve clients in this mode

Flex+Bridge mode:
  - Combined FlexConnect and bridge functionality
  - Mesh AP with local switching capability
  - For outdoor/mesh deployments at remote sites
```

## RF Fundamentals

### Frequency Bands and Channels

```
2.4 GHz band (802.11b/g/n/ax):
  Range: 2.400 — 2.4835 GHz
  Channels: 1-14 (varies by regulatory domain)
  Non-overlapping channels: 1, 6, 11 (20 MHz width)
  Channel width: 20 MHz typical (40 MHz possible but not recommended)
  Characteristics:
    - Better range/penetration through walls
    - More interference (Bluetooth, microwave, Zigbee, cordless phones)
    - Only 3 non-overlapping channels = limited capacity
    - Avoid 40 MHz channels (consumes 2/3 of available spectrum)

5 GHz band (802.11a/n/ac/ax):
  Range: 5.150 — 5.825 GHz
  UNII bands:
    UNII-1: 36, 40, 44, 48            (indoor only, no DFS)
    UNII-2: 52, 56, 60, 64            (DFS required)
    UNII-2e: 100-144 (varies)          (DFS required)
    UNII-3: 149, 153, 157, 161, 165   (no DFS, higher power)
  Non-overlapping 20 MHz channels: up to 25 (regulatory dependent)
  Channel widths: 20, 40, 80, 160 MHz
  Characteristics:
    - More channels = more capacity
    - Shorter range than 2.4 GHz
    - DFS channels require radar detection (AP must vacate on detection)
    - DFS CAC (Channel Availability Check): 60-second scan before use

6 GHz band (802.11ax Wi-Fi 6E / 802.11be Wi-Fi 7):
  Range: 5.925 — 7.125 GHz
  Channels: up to 59 channels (20 MHz), 14 channels (80 MHz)
  Channel widths: 20, 40, 80, 160, 320 MHz (Wi-Fi 7)
  Characteristics:
    - Greenfield spectrum: no legacy devices (clean start)
    - AFC (Automated Frequency Coordination) for standard power outdoor
    - LPI (Low Power Indoor) for indoor without AFC
    - No DFS requirement
    - WPA3 required (OWE for open networks)

DFS (Dynamic Frequency Selection):
  - Required on UNII-2 and UNII-2e channels
  - AP must detect radar signals (weather, military, airport)
  - On detection: AP vacates channel within 200ms
  - Non-occupancy period: 30 minutes before retrying that channel
  - CAC: AP scans channel for 60s before transmitting
  - Impact: brief client disruption during channel change
```

### Signal and Power Concepts

```
Key RF metrics:
  RSSI (Received Signal Strength Indicator):
    Measured in dBm (decibels relative to 1 milliwatt)
    Typical ranges:
      -30 dBm:  Excellent (very close to AP)
      -50 dBm:  Strong
      -65 dBm:  Good (minimum for voice/video)
      -70 dBm:  Adequate (minimum for data)
      -80 dBm:  Weak (connection issues likely)
      -90 dBm:  Near unusable

  SNR (Signal-to-Noise Ratio):
    SNR = Signal (dBm) - Noise Floor (dBm)
    Example: -60 dBm signal, -90 dBm noise = 30 dB SNR
    Requirements:
      > 25 dB:  Excellent (supports highest data rates)
      20-25 dB: Good (reliable for most applications)
      15-20 dB: Marginal (reduced throughput)
      < 15 dB:  Poor (high retransmissions, disconnections)

  Noise floor:
    Typical: -90 to -95 dBm (clean environment)
    Elevated: -85 to -80 dBm (interference present)
    Sources: co-channel APs, Bluetooth, microwave ovens, radar

  Tx Power:
    Measured in dBm or mW
    Common AP power levels: 1-23 dBm (1 mW to 200 mW)
    Regulatory maximums vary by country and band
    EIRP = Tx Power + Antenna Gain - Cable Loss

Free Space Path Loss:
  FSPL(dB) = 20*log10(d) + 20*log10(f) - 27.55
    d = distance in meters, f = frequency in MHz
  Every doubling of distance = 6 dB loss
  5 GHz loses approximately 8 dB more than 2.4 GHz at same distance
```

### Antenna Types

```
Omnidirectional:
  - Radiates signal equally in all horizontal directions (360 degrees)
  - Donut-shaped pattern (weak above and below)
  - Typical gain: 2-5 dBi
  - Used for: general indoor coverage, ceiling-mount APs

Directional (patch/panel):
  - Focuses signal in a specific direction
  - Typical gain: 6-14 dBi
  - Beamwidth: 30-120 degrees (horizontal)
  - Used for: corridors, long hallways, outdoor point-to-multipoint

High-gain directional (Yagi, parabolic):
  - Highly focused beam
  - Typical gain: 12-28 dBi
  - Very narrow beamwidth: 5-30 degrees
  - Used for: point-to-point bridges, long-distance outdoor links

Internal antennas (integrated):
  - Built into AP housing
  - Omnidirectional pattern
  - Typical gain: 2-4 dBi
  - Most indoor enterprise APs use internal antennas
  - Catalyst 9100 series: internal dual-band antennas

Beamforming (smart antennas):
  - 802.11ac/ax: transmit beamforming via NDP sounding
  - AP shapes beam toward individual client
  - Improves SNR at client without increasing total power
  - Requires client feedback (explicit beamforming)
```

## RRM (Radio Resource Management)

### Overview

```
RRM components (all run automatically on WLC):

  TPC (Transmit Power Control):
    - Adjusts AP transmit power to optimal level
    - Goal: sufficient coverage with minimum co-channel interference
    - Runs every 600 seconds (10 minutes) by default
    - Calculates power based on neighbor AP signal strength
    - Third-party AP neighbor detected = increase power
    - Target: neighbors hear each other at -65 to -70 dBm
    - Hysteresis: 3 dB (avoids power oscillation)

  DCA (Dynamic Channel Assignment):
    - Assigns optimal channel to each AP radio
    - Considers: co-channel interference, noise, load, radar
    - Runs every 600 seconds by default
    - EDRRM (Event-Driven RRM): immediate re-channel on high interference
    - Anchor time: configurable window when DCA is allowed to run
    - Channel width selection: auto (prefers 20 MHz in dense environments)

  Coverage Hole Detection:
    - Monitors client RSSI reported by APs
    - If clients persistently report RSSI below threshold:
      Triggers TPC to increase power on neighboring APs
    - Default threshold: -80 dBm
    - Minimum client count: 3 clients below threshold
    - Helps identify areas where clients have weak signal

  Load balancing:
    - Distributes clients across APs with overlapping coverage
    - AP denies association if overloaded (client retries on less-loaded AP)
    - Window: configurable client count difference threshold
    - Band steering: encourages dual-band clients to use 5 GHz

  Band steering (Band Select):
    - Detects dual-band capable clients (via probe requests)
    - Delays probe response on 2.4 GHz, responds immediately on 5 GHz
    - Client preferentially associates on 5 GHz
    - Configurable: enable/disable per WLAN
    - Does NOT force clients (only influences association decision)
```

### RRM Configuration

```bash
# View RRM status
show ap dot11 5ghz summary
show ap dot11 24ghz summary
show advanced 802.11a channel
show advanced 802.11a txpower

# Configure TPC
config 802.11a txPower global auto
config 802.11a txPower global min 7
config 802.11a txPower global max 17
config 802.11a txPower global threshold -65

# Configure DCA
config 802.11a channel global auto
config 802.11a channel add 36
config 802.11a channel add 40
config 802.11a channel add 44
config 802.11a channel add 48

# IOS-XE (9800 WLC) RRM configuration
ap dot11 5ghz rrm channel dca
ap dot11 5ghz rrm txpower auto
ap dot11 5ghz rrm txpower min 7
ap dot11 5ghz rrm txpower max 17
ap dot11 5ghz rrm channel dca channel-width 40

# EDRRM (Event-Driven RRM)
config advanced 802.11a channel update-interval 600
config advanced 802.11a channel edrrm enable

# Coverage hole detection
config advanced 802.11a coverage data rssi-threshold -80
config advanced 802.11a coverage data fail-percentage 25
config advanced 802.11a coverage level global 3
```

## Client Authentication and Security

### Authentication Methods

```
PSK (Pre-Shared Key):
  - Static passphrase configured on WLAN and clients
  - WPA2-PSK: CCMP-AES encryption, 4-way handshake
  - WPA3-SAE: Simultaneous Authentication of Equals
    - Replaces PSK 4-way handshake with SAE (Dragonfly protocol)
    - Resistant to offline dictionary attacks
    - Forward secrecy per session
  - Best for: small offices, guest networks (simple)

802.1X / EAP (Enterprise authentication):
  - Per-user credentials via RADIUS (ISE, FreeRADIUS)
  - EAP types:
    EAP-TLS:      Mutual certificate auth (most secure, complex PKI)
    PEAP-MSCHAPv2: Server cert + username/password (most common)
    EAP-FAST:     Cisco proprietary, PAC-based (legacy)
    EAP-TTLS:     Tunneled TLS (flexible inner method)
  - Flow: Client -> AP -> WLC -> RADIUS -> response
  - PMK caching: master key cached after first auth for fast re-auth
  - Best for: enterprise, managed devices, per-user audit trail

Web Authentication (WebAuth):
  - Client connects with open or PSK association
  - HTTP traffic intercepted and redirected to portal
  - Types:
    Local Web Auth (LWA):    Portal hosted on WLC
    Central Web Auth (CWA):  Portal hosted on ISE (redirect via RADIUS CoA)
    External Web Auth (EWA): Portal on third-party server
  - Best for: guest access, BYOD onboarding, hotspot

MAC Authentication Bypass (MAB):
  - AP/WLC sends client MAC to RADIUS as username/password
  - ISE checks MAC against known endpoint database
  - Used for: printers, cameras, IoT devices that cannot do 802.1X
  - Combine with ISE profiling for device identification
  - Weakest method (MACs are spoofable)
```

### WLAN Security Standards

```
WPA2 (Wi-Fi Protected Access 2):
  - Personal (PSK): AES-CCMP, 4-way handshake
  - Enterprise (802.1X): AES-CCMP, per-user keys via RADIUS
  - Still widely deployed, minimum acceptable security
  - Vulnerability: KRACK attack (mitigated with patches)

WPA3 (Wi-Fi Protected Access 3):
  - Personal (SAE): Dragonfly key exchange, forward secrecy
  - Enterprise (802.1X): 192-bit security suite (CNSA), PMF mandatory
  - Transition mode: WPA2+WPA3 simultaneously (migration path)
  - Protected Management Frames (PMF/802.11w): mandatory
  - Resistant to: offline dictionary, KRACK, deauthentication attacks

OWE (Opportunistic Wireless Encryption):
  - Encrypts open (no-passphrase) networks
  - Diffie-Hellman key exchange during association
  - No authentication — still an open network
  - Protects against passive eavesdropping (coffee shop scenario)
  - Enhanced Open: marketing name for OWE
  - Transition mode: Open + OWE simultaneously

802.11w (Protected Management Frames — PMF):
  - Protects management frames from spoofing/injection
  - Prevents deauthentication attacks (deauth flood)
  - Required by WPA3, optional for WPA2
  - SA Query mechanism: validates management frame source

Fast Transition (802.11r / FT):
  - Reduces roaming time from ~50ms to <10ms
  - Pre-negotiates PMK with target AP before roaming
  - Over-the-Air (OTA): client directly with target AP
  - Over-the-DS (ODS): client through current AP to target AP
  - Critical for voice/video roaming (avoids call drops)
```

## Roaming

### Roaming Types

```
L2 roaming (same subnet, same WLC):
  - Client moves between APs on the same controller
  - Same VLAN and IP address
  - WLC updates internal tables (AP association, BSSID)
  - Seamless: sub-50ms with PMK caching
  - No DHCP renewal needed

L2 roaming (same subnet, different WLC — intra-mobility group):
  - Client moves between APs managed by different WLCs
  - WLCs are in the same mobility group
  - Anchor WLC maintains client context
  - Foreign WLC tunnels traffic to anchor WLC
  - IP address preserved, seamless from client perspective

L3 roaming (different subnet):
  - Client moves to AP on a different VLAN/subnet
  - Anchor WLC maintains original IP and DHCP binding
  - Foreign WLC tunnels client traffic back to anchor WLC
  - Client keeps original IP (anchor WLC is default gateway proxy)
  - Higher latency (traffic hairpins through anchor)
  - Timeout: session lifetime or L3 roaming timeout expires

Inter-controller roaming:
  - Mobility messaging between WLCs (UDP 16666, CAPWAP 16667)
  - Mobility group: up to 24 WLCs sharing roaming context
  - Mobility domain: up to 72 WLCs with mobility tunnels
  - PMKID caching shared between mobility group members
  - Requires matching mobility group name and DTLS configuration

Fast Secure Roaming methods:
  CCKM (Cisco Centralized Key Management):
    - Cisco proprietary, fastest roam (<5ms)
    - WLC caches and distributes session keys
    - Client reuses cached key during roam
    - Requires Cisco-compatible supplicant

  OKC (Opportunistic Key Caching):
    - PMKSA caching with key derivation
    - Works with most supplicants
    - WLC distributes PMK-R1 to neighboring APs
    - Roam time: ~20ms

  802.11r (Fast BSS Transition):
    - IEEE standard for fast roaming
    - FT Initial Mobility Domain association
    - FT Action frames pre-authenticate to target AP
    - Over-the-Air or Over-the-DS variants
    - Most compatible across vendors
    - Roam time: <10ms

  802.11k (Radio Resource Measurement):
    - AP provides neighbor reports to clients
    - Client learns available APs without full scanning
    - Reduces roaming scan time significantly
    - Complements 802.11r for optimal roaming

  802.11v (BSS Transition Management):
    - AP can suggest client move to a better AP (steer)
    - Client can report preferred transition candidates
    - Helps offload clients from overloaded APs
    - Works with load balancing and band steering
```

## Channel Planning

### Design Guidelines

```
2.4 GHz channel plan:
  +------+    +------+    +------+
  | Ch 1 |    | Ch 6 |    | Ch 11|
  +------+    +------+    +------+
  Only 3 non-overlapping channels — plan in a honeycomb pattern
  AP placement:
    - Adjacent APs on different channels (1-6-11 rotation)
    - Target: -67 dBm at cell edge for data
    - Target: -65 dBm at cell edge for voice
    - Overlap: 15-20% cell overlap for seamless roaming
    - Avoid channel 14 (Japan only, restricted use)

5 GHz channel plan:
  UNII-1:     36  40  44  48         (4 channels, indoor, no DFS)
  UNII-2:     52  56  60  64         (4 channels, DFS required)
  UNII-2e:    100-144                (up to 12 channels, DFS)
  UNII-3:     149 153 157 161 165    (5 channels, higher power)
  Total: up to 25 non-overlapping 20 MHz channels
  With 40 MHz: up to 12 channels
  With 80 MHz: up to 6 channels
  With 160 MHz: up to 2-3 channels

  Design rules:
    - Prefer 20 MHz in high-density (more channels, less CCI)
    - Use 40 MHz for moderate density with throughput needs
    - Reserve 80/160 MHz for point-to-point or low-density areas
    - Enable DFS channels (more spectrum, fewer co-channel conflicts)
    - Plan for DFS evacuation: clients need fallback channels

6 GHz channel plan:
  59 channels at 20 MHz, 29 at 40, 14 at 80, 7 at 160, 3 at 320 MHz
  Greenfield band — no legacy device interference
  PSC (Preferred Scanning Channels): subset for discovery
    20 channels at 20 MHz for efficient scanning
  Use wider channels freely (no legacy coexistence issues)
  AFC required for standard power outdoor deployments
```

### High-Density Design

```
High-density environments (stadiums, lecture halls, conference rooms):

  AP placement:
    - Mount APs underneath seats or below crowd level
    - Use directional antennas pointed at the client area
    - Reduce cell size: lower power, tighter spacing
    - Target 25-50 clients per radio (maximum)

  Configuration:
    - Disable lower data rates (disable 1, 2, 5.5, 6, 9 Mbps)
    - Set minimum data rate to 12 or 18 Mbps (5 GHz)
    - Disable 802.11b rates entirely on 2.4 GHz
    - Enable OFDMA and BSS Coloring (802.11ax)
    - Enable Airtime Fairness
    - Set RTS/CTS threshold to 200-500 bytes
    - Reduce beacon interval (increases overhead but helps roaming)
    - Use 20 MHz channels exclusively (maximize channel reuse)

  Capacity planning:
    Rule of thumb: 1 AP per 20-30 active users (enterprise)
    High-density: 1 AP per 15-20 users
    Very high density: 1 AP per 10-15 users
    Bandwidth per user: 2-5 Mbps (web), 10-20 Mbps (video), 0.1 Mbps (voice)
```

## Advanced Features

### wIPS (Wireless Intrusion Prevention System)

```
wIPS capabilities:
  - Rogue AP detection and classification
    Managed, friendly, malicious, unclassified
    Auto-containment: AP sends deauth to rogue clients
  - Ad-hoc network detection
  - Denial of Service detection:
    Deauth flood, association flood, EAPOL flood, beacon flood
  - Man-in-the-middle detection (honeypot APs)
  - Client exclusion (blacklisting) based on signature match
  - Integration: DNA Center, ISE, SIEM

wIPS modes:
  Local mode APs: part-time scanning (time-sliced, less effective)
  Monitor mode APs: full-time dedicated scanning (recommended)
  Enhanced Local Mode (ELM): improved part-time scanning
```

### CleanAir

```
CleanAir (Cisco Spectrum Intelligence):
  - Silicon-level spectrum analysis in AP hardware
  - Detects and classifies non-Wi-Fi interference:
    Bluetooth, microwave oven, cordless phone, baby monitor,
    wireless video camera, motion sensor, Zigbee, radar, jammer
  - Interference device type, severity, duty cycle reported
  - Event-Driven RRM: automatic channel change on interference
  - CleanAir Advisor: DNA Center shows interference on floor maps
  - Persistent Device Avoidance (PDA): marks channels with chronic interference
  - Air Quality Index (AQI): 1-100 score per channel (100 = clean)
  - Requires CleanAir-capable APs (Catalyst 9100, Aironet 2800/3800/4800)
```

### DNA Spaces (Cisco Spaces)

```
DNA Spaces capabilities:
  - Location analytics: heatmaps, dwell time, visitor flow
  - Presence detection: MAC-based or probe-request-based
  - Proximity-based engagement: push notifications, wayfinding
  - Asset tracking: BLE tags tracked via AP infrastructure
  - IoT services: environmental sensors, BLE beacons
  - APIs: REST API for custom integrations
  - Integration with CMX (Connected Mobile Experiences)
  - Cloud-based: SaaS platform connected to on-prem WLC/DNA Center
  - Privacy: MAC randomization handling, opt-in/opt-out policies
```

## Verification and Troubleshooting

```bash
# AP status and count
show ap summary
show ap join stats summary
show ap uptime

# Client status
show wireless client summary
show wireless client mac-address <mac> detail

# RF status
show ap dot11 5ghz summary
show ap dot11 24ghz summary
show ap name <ap-name> dot11 5ghz
show ap auto-rf dot11 5ghz

# RRM status
show advanced 802.11a channel
show advanced 802.11a txpower
show advanced 802.11a coverage
show advanced 802.11a rrm

# CAPWAP status
show ap capwap summary
show capwap client rcb

# Channel utilization
show ap dot11 5ghz load-info
show ap name <ap-name> channel-utilization

# Security
show wlan summary
show wlan id <id>
show wireless client mac <mac> security

# Mobility/roaming
show mobility summary
show mobility ap-list
show wireless mobility statistics

# CleanAir
show ap dot11 5ghz cleanair device type all
show ap dot11 5ghz cleanair air-quality summary

# IOS-XE 9800 specific
show wireless stats ap join summary
show wireless tag site summary
show wireless profile policy summary
show ap rf-profile summary

# Debug (use carefully — performance impact)
debug capwap events
debug dot11 client mac <mac>
debug mobility handoff
debug dot1x events
```

## Tips

- Always set a minimum mandatory data rate of 12 Mbps on 5 GHz and disable 802.11b rates on 2.4 GHz; low rates consume excessive airtime and reduce capacity for all clients.
- Use 5 GHz as the primary band and 2.4 GHz only for legacy devices or IoT; enable band steering to push dual-band clients to 5 GHz automatically.
- Set WLC HA in SSO mode with a dedicated redundancy port; active/standby failover should be sub-second and transparent to clients and APs.
- Deploy FlexConnect for any remote site where losing WLC connectivity should not take down local wireless; test standalone mode during deployment.
- Keep CAPWAP path MTU at 1500 minimum on all intermediate devices; CAPWAP fragmentation causes hidden performance degradation.
- Enable 802.11r (Fast Transition) and 802.11k (neighbor reports) together for optimal roaming; 802.11r alone may cause issues with older clients so use FT+PMK caching as fallback.
- In high-density venues, reduce AP power to shrink cell size and use 20 MHz channels exclusively; more channels means more spatial reuse.
- Always enable PMF (802.11w) — it prevents deauthentication attacks and is mandatory for WPA3.
- Use CleanAir data to identify non-Wi-Fi interference before blaming the wireless network for poor performance; a microwave oven can destroy a 2.4 GHz channel.
- Deploy at least one dedicated monitor mode AP per floor for wIPS; part-time scanning in local mode misses fast rogue activity.
- Configure DCA anchor time to run channel changes during low-usage hours (e.g., 2:00-5:00 AM) to minimize client disruption.
- For 6 GHz deployments, require WPA3 and plan for AFC registration; OWE is mandatory for open networks on 6 GHz.
- Match AP antenna gain with expected coverage area; higher gain is not always better because it creates co-channel interference with distant APs.

## See Also

- sd-access, radius, tacacs, vlan, eigrp, ospf, ipsec, ntp, dhcp, snmp, 802.1X

## References

- [IEEE 802.11 Standards — IEEE SA](https://standards.ieee.org/ieee/802.11/7028/)
- [RFC 5415 — CAPWAP Protocol Specification](https://www.rfc-editor.org/rfc/rfc5415)
- [RFC 5416 — CAPWAP Binding for IEEE 802.11](https://www.rfc-editor.org/rfc/rfc5416)
- [Cisco Catalyst 9800 WLC Configuration Guide](https://www.cisco.com/c/en/us/td/docs/wireless/controller/9800/config-guide.html)
- [Cisco Wireless LAN Controller Best Practices](https://www.cisco.com/c/en/us/td/docs/wireless/controller/technotes/lwapp-best-practices.html)
- [Cisco High-Density Wi-Fi Design Guide (CVD)](https://www.cisco.com/c/en/us/td/docs/wireless/controller/technotes/8-4/High_Density_design_guide.html)
- [Cisco RRM White Paper](https://www.cisco.com/c/en/us/td/docs/wireless/controller/technotes/8-3/b_RRM_White_Paper.html)
- [Cisco CleanAir Technology](https://www.cisco.com/c/en/us/products/wireless/cleanair-technology.html)
- [Wi-Fi Alliance — WPA3 Specification](https://www.wi-fi.org/discover-wi-fi/security)
- [Wi-Fi Alliance — Wi-Fi 6/6E/7](https://www.wi-fi.org/discover-wi-fi/wi-fi-certified-6)
