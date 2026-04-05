# Physical Security

> Layered protection of facilities, equipment, and personnel through access control, surveillance, environmental controls, fire suppression, and power management.

## Defense in Depth — Physical Layers

```
Layer          Controls                        Examples
─────          ────────                        ────────
Perimeter      Fencing, gates, bollards,       8ft fence + barbed wire,
               lighting, vehicle barriers      anti-ram bollards
Building       Locks, doors, windows,          Reinforced doors, mantraps,
               reception, visitor mgmt         badge readers
Floor/Zone     Badge access, turnstiles,       Zoned access cards,
               interior walls                  glass break sensors
Room           Biometrics, dual control,       Server room: badge + PIN,
               CCTV, motion sensors            two-person integrity
Rack/Device    Cable locks, rack locks,        Locked cabinets, port locks,
               tamper seals, port blockers     chassis intrusion detection
```

## Access Control Systems

### Badge and Card Systems

```
Type            Technology          Range    Security Level
────            ──────────          ─────    ──────────────
Proximity       125 kHz RFID        10cm     Low (easily cloned)
Smart Card      13.56 MHz (MIFARE)  5cm      Medium
iCLASS SE       13.56 MHz + crypto  5cm      High
SEOS            NFC + BLE           5cm      Very High
Mobile          BLE / NFC           varies   High (with MDM)

# Multi-factor physical access
# Something you HAVE (badge) + Something you KNOW (PIN)
# Something you HAVE (badge) + Something you ARE (biometric)

# Anti-passback
# - Prevents using same badge to enter twice without exiting
# - Hard anti-passback: denies entry
# - Soft anti-passback: allows entry but alerts
```

### Biometric Systems

```
Type              FAR        FRR        CER     Speed
────              ───        ───        ───     ─────
Fingerprint       0.001%     0.1%       ~0.05%  1-2 sec
Iris scan         0.0001%    0.2%       ~0.01%  2-4 sec
Retina scan       0.00001%   0.1%       ~0.001% 3-5 sec
Facial recognition 0.1%      1%         ~0.5%   1-3 sec
Hand geometry     0.1%       0.1%       ~0.1%   1-2 sec
Voice recognition 0.5%       2%         ~1%     3-5 sec

FAR = False Acceptance Rate (Type II error — impostor accepted)
FRR = False Rejection Rate (Type I error — legit user rejected)
CER = Crossover Error Rate (where FAR = FRR — lower is better)

# Higher security = lower FAR (fewer impostors accepted)
#   → Increases FRR (more legit rejections) — inconvenience tradeoff
# Throughput requirement: airports need speed, vaults need accuracy
```

### Mantraps / Security Vestibules

```
# Two interlocking doors — only one opens at a time
# Prevents tailgating/piggybacking

┌─────────────────────────────────┐
│  Public          ┌────────┐    │
│  Side     Door 1 │        │ Door 2   Secure
│  ────────>  ═══  │ Vestib │  ═══  ─> Side
│           (badge)│  ule   │(badge+
│                  │        │ bio)
│                  └────────┘
└─────────────────────────────────┘

Features:
- Weight sensors (detect multiple people)
- CCTV inside vestibule
- Intercom for security guard
- Automatic lock if both doors triggered
- Anti-passback integrated
```

## Surveillance

### CCTV Systems

```
Camera Type    Use Case                Resolution
───────────    ────────                ──────────
Fixed dome     Indoor, general         2-8 MP
PTZ            Perimeter, parking      2-4 MP (optical zoom)
Bullet         Outdoor, long range     4-12 MP
Fisheye/360    Lobbies, open areas     12 MP (dewarped)
Thermal        Perimeter, night        VGA-HD (heat detection)
LPR            Vehicle gates           2+ MP (license plate)

# Storage calculations
# 1080p @ 15fps ≈ 1.5 GB/day/camera (H.264)
# 4K   @ 15fps ≈ 6 GB/day/camera (H.265)
# 100 cameras × 30 days × 1.5 GB = 4.5 TB (1080p)
# 100 cameras × 30 days × 6 GB   = 18 TB (4K)

# Retention requirements
# General:    30-90 days
# Financial:  1-7 years (PCI DSS, SOX)
# Government: varies by classification
# Legal hold: indefinite during litigation
```

### Additional Surveillance

```
# Motion detection
- PIR (Passive Infrared): detects body heat, indoor
- Microwave: detects movement through walls
- Dual-tech: PIR + microwave (reduces false alarms)
- Video analytics: AI-based motion/object detection

# Guard force
- Fixed posts: reception, control room, server room
- Patrol: randomized routes (avoid predictability)
- Response: dedicated to alarm response
- K-9 units: explosive/narcotic detection

# Intrusion Detection Systems (Physical)
- Door contacts: magnetic reed switches
- Glass break sensors: acoustic + shock
- Vibration sensors: vault/safe protection
- Photoelectric beams: perimeter invisible fence
```

## Environmental Controls

### HVAC

```
# Data Center Temperature and Humidity (ASHRAE guidelines)

Parameter              Recommended    Allowable
─────────              ───────────    ─────────
Temperature (inlet)    18-27°C        15-32°C
                       (64-80°F)      (59-90°F)
Humidity (RH)          20-80%         8-80%
Dew point              5.5-15°C       -12-24°C

# Hot aisle / cold aisle containment
# ┌─────┐  ┌─────┐  ┌─────┐
# │Rack │  │Rack │  │Rack │    ← HOT AISLE (exhaust)
# │ ▲▲▲ │  │ ▲▲▲ │  │ ▲▲▲ │
# │     │  │     │  │     │
# │ ▼▼▼ │  │ ▼▼▼ │  │ ▼▼▼ │
# │Rack │  │Rack │  │Rack │    ← COLD AISLE (intake)
# └─────┘  └─────┘  └─────┘
# CRAC/CRAH units push cold air into raised floor → cold aisle

# Monitoring
- Temperature sensors: every rack (top, middle, bottom)
- Humidity sensors: per zone
- Water leak detection: under raised floor, near CRAC units
- Airflow sensors: confirm proper circulation
- SNMP/IPMI alerts: threshold-based notifications
```

## Fire Suppression

### Fire Classes

```
Class    Fuel Type              Suppression Method
─────    ─────────              ──────────────────
A        Ordinary (wood,paper)  Water, foam, dry chemical
B        Flammable liquid       Foam, CO2, dry chemical
C        Electrical equipment   CO2, clean agent, dry chemical
D        Combustible metal      Special dry powder
K        Cooking oils/fats      Wet chemical
```

### Suppression Systems

```
System          Mechanism                    Use Case
──────          ─────────                    ────────
Wet Pipe        Water in pipes; sprinkler    Offices, warehouses
                heads activate by heat       (NOT data centers)
                (fastest response)

Dry Pipe        Pressurized air in pipes;    Cold environments
                water released when air      (prevents frozen pipes)
                pressure drops

Pre-Action       Dry pipe + detection         Data centers, museums,
                system must trigger          archives (double
                before water flows           interlock = 2 triggers)

Deluge          All heads open               High-hazard areas
                simultaneously when          (aircraft hangars,
                detection activates          chemical storage)

FM-200          Halocarbon clean agent       Data centers, telecom
(HFC-227ea)     (no water, safe for          rooms (10 sec discharge,
                electronics, safe for        safe for occupied spaces)
                people at design conc.)

Novec 1230      Fluoroketone clean agent     Data centers (zero ODP,
                (no water, safe for          lowest GWP of clean
                electronics)                 agents, 10 sec discharge)

CO2             Displaces oxygen             Unoccupied spaces only
                (suffocation risk to         (engine rooms, vaults)
                humans — LETHAL)             REQUIRES evacuation alarm

Inergen         Blend of N2, Ar, CO2         Occupied spaces
                reduces O2 to 12.5%          (safe for people,
                (fire cannot sustain)         safe for equipment)
```

### Detection Systems

```
Type              Detects              Response Time
────              ───────              ─────────────
Ionization        Fast-flaming fires   Seconds
Photoelectric     Slow-smoldering      Seconds
Aspirating (VESDA) Very early smoke    Minutes before visible
Heat (rate-of-rise) Rapid temp increase Seconds
Heat (fixed temp)  Threshold temp      Seconds-minutes
Flame (UV/IR)     Open flame           Milliseconds
```

## Power

### UPS (Uninterruptible Power Supply)

```
Type               How It Works              Use Case
────               ────────────              ────────
Standby/Offline    Switches to battery       Home/small office
                   on outage (5-12ms gap)    (cheapest)

Line-Interactive   AVR regulates voltage;    SMB, network closets
                   battery on outage         (handles brownouts)
                   (2-4ms switchover)

Online/            Constant double           Data centers, critical
Double-Conversion  conversion (AC→DC→AC)     systems (zero transfer
                   Battery always in         time, cleanest power)
                   circuit (0ms gap)

# Power calculations
# Load (Watts) = Volts × Amps × Power Factor
# Runtime = Battery capacity (Wh) / Load (W)
# Typical PF: 0.9 for servers
# N+1 redundancy: one extra UPS module
# 2N redundancy: fully duplicated power path
```

### Power Distribution

```
# Utility power → ATS → UPS → PDU → Rack → Server

ATS     Automatic Transfer Switch — switches between
        utility and generator power

PDU     Power Distribution Unit
        Basic: power strip with monitoring
        Metered: per-outlet power monitoring
        Switched: remote outlet control
        Managed: metered + switched + environmental sensors

Generator  Diesel/natural gas backup
           Startup time: 10-30 seconds (UPS bridges the gap)
           Fuel capacity: 24-72 hours typical
           Testing: monthly under load (30 min minimum)
           Maintenance: weekly inspection, annual full service

# Power redundancy levels
# Tier I:   Single path, no redundancy (99.671% uptime)
# Tier II:  Single path, redundant components (99.741%)
# Tier III: Multiple paths, one active (99.982%)
# Tier IV:  Multiple paths, all active (99.995%)
```

## Cable Management

```
# Physical security for cabling

Structured Cabling Standards:
- TIA-568: Commercial building telecommunications
- TIA-942: Data center telecommunications infrastructure
- TIA-606: Labeling and documentation

Cable Security:
- Plenum-rated cables for air handling spaces (fire safety)
- Armored fiber for exposed runs (physical protection)
- Conduit for cable runs between buildings
- Locked cable trays and pathways
- Cable seals at room penetrations (fire stopping)
- Fiber optic tapping detection (OTDR monitoring)

# Labeling
- Both ends of every cable labeled
- Patch panel ports labeled
- Color coding by function (data, voice, management)
- Documentation in cable management database
```

## Visitor Management

```
# Process
1. Pre-registration: host registers visitor in advance
2. Arrival: visitor presents ID at reception
3. Verification: ID checked against pre-registration
4. Badge issuance: temporary badge (different color from employee)
5. Escort: visitor escorted at all times in secure areas
6. Sign-in/out log: name, company, host, time in/out, areas visited
7. Badge return: collected at departure (alarm if not returned)

# Badge Types
Employee:       Photo, permanent, full access per role
Contractor:     Photo, expiring, limited access
Visitor:        No photo, single-day, escort required
Temporary:      Photo, 1-90 days, defined access

# NDA requirement: visitors accessing sensitive areas
# Photography policy: no cameras in secure areas
# Device policy: no personal electronics past security boundary
```

## Evidence Storage (Physical)

```
# Secure evidence storage for digital forensics

Requirements:
- Dedicated, access-controlled room
- Environmental monitoring (temperature, humidity)
- Fire suppression (clean agent, not water)
- Tamper-evident seals on evidence containers
- Chain of custody log at room entrance
- CCTV recording of all access
- Dual-person access for high-sensitivity evidence
- Faraday cage/bags for mobile devices (prevent remote wipe)
- Write-blocker storage rack
- Evidence safe (fireproof, combination lock)
```

## Data Center Standards — TIA-942

```
Tier    Availability    Downtime/Year    Key Requirements
────    ────────────    ─────────────    ────────────────
I       99.671%         28.8 hours       Single path, no redundancy
II      99.741%         22.0 hours       Redundant components, single path
III     99.982%         1.6 hours        Multiple paths (one active),
                                         concurrent maintainability
IV      99.995%         0.4 hours        Multiple active paths,
                                         fault tolerant, 2N+1

# Site Selection Criteria
- Not in flood plain (100-year + 500-year flood maps)
- Not on major flight path
- Not adjacent to chemical/industrial facilities
- Not on geological fault line
- Access to diverse utility feeds
- Proximity to emergency services
- Distance from major highways (vehicle bomb radius)
- Low crime area
```

## See Also

- hardening-linux
- fire-suppression
- environmental-controls
- cis-benchmarks
- incident-response

## References

- TIA-942: Data Center Telecommunications Infrastructure Standard
- ASHRAE TC 9.9: Thermal Guidelines for Data Processing Environments
- NFPA 75: Standard for Protection of IT Equipment
- NFPA 76: Standard for Fire Protection of Telecommunications Facilities
- NIST SP 800-116: Guidelines for PIV Card Authentication
- ASIS Physical Security Professional (PSP) Body of Knowledge
- CPTED: Crime Prevention Through Environmental Design (Jeffery, 1971)
