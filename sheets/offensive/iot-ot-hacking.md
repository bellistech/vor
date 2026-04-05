> For authorized security testing, red team exercises, and educational study only.

# IoT & OT Hacking (CEH v13 Module 18)

Attacking and defending Internet of Things devices, Industrial Control Systems, and Operational Technology environments.

---

## IoT Architecture Layers

```
Application Layer    Web dashboards, mobile apps, cloud APIs, analytics
                     Vulnerabilities: insecure APIs, weak auth, data leakage

Network Layer        Wi-Fi, Ethernet, MQTT, CoAP, ZigBee, BLE, LoRaWAN, 6LoWPAN
                     Vulnerabilities: MITM, replay, packet injection, no encryption

Perception Layer     Sensors, actuators, embedded MCUs, cameras, RFID tags
                     Vulnerabilities: firmware extraction, JTAG/UART debug, tampering
```

---

## IoT Protocols

```
Protocol    Transport    Port(s)     Notes
--------    ---------    -------     -----
MQTT        TCP          1883/8883   Pub/sub, QoS 0-2, TLS on 8883
CoAP        UDP          5683/5684   REST-like, DTLS on 5684, observe pattern
ZigBee      802.15.4     --          Mesh, 128-bit AES, touchlink vuln
Z-Wave      Sub-GHz      --          Mesh, S0 (weak) vs S2 (strong) security
BLE         2.4 GHz      --          GATT profiles, pairing modes, sniffing with Ubertooth
LoRaWAN     Sub-GHz      --          Long range, ABP vs OTAA join, AES-128 keys
6LoWPAN     802.15.4     --          IPv6 adaptation layer, header compression
```

---

## IoT Attack Surface

### Device Level
```bash
# Firmware extraction via SPI flash
flashrom -p ch341a_spi -r firmware.bin

# UART console access (find TX/RX with multimeter or logic analyzer)
screen /dev/ttyUSB0 115200

# JTAG debugging with OpenOCD
openocd -f interface/ftdi/jlink.cfg -f target/stm32f4x.cfg
# then: telnet localhost 4444

# Find debug pads: look for 4-pin (UART) or 10/20-pin (JTAG) headers on PCB
```

### Network Level
```bash
# BLE scanning and enumeration
sudo hcitool lescan
gatttool -b AA:BB:CC:DD:EE:FF --primary
gatttool -b AA:BB:CC:DD:EE:FF --char-read -a 0x0025

# ZigBee sniffing with KillerBee
zbstumbler                          # discover networks
zbdump -c 15 -w capture.pcap       # capture on channel 15
zbreplay -c 15 -r capture.pcap     # replay attack
```

### Cloud / API Level
```bash
# Test IoT cloud API endpoints
curl -s https://api.iot-device.example/v1/devices | jq .
# Check for: no auth, IDOR, verbose errors, default creds

# Mobile app analysis: decompile APK, look for hardcoded keys
apktool d iot-app.apk
grep -rn "api_key\|secret\|password\|token" iot-app/
```

---

## Firmware Analysis

```bash
# Identify firmware contents
binwalk firmware.bin
# Extract filesystem
binwalk -e firmware.bin
cd _firmware.bin.extracted/

# Alternative: firmware-mod-kit
./extract-firmware.sh firmware.bin
# Explore squashfs/cramfs root filesystem

# Find hardcoded credentials
grep -rn "password\|passwd\|secret\|key=" .
strings firmware.bin | grep -i "pass\|admin\|root\|login"

# Identify crypto keys
find . -name "*.pem" -o -name "*.key" -o -name "*.crt"

# Analyze binaries with Ghidra (headless mode)
analyzeHeadless /tmp/ghidra_project proj -import ./usr/bin/target_binary \
  -postScript FindCrypto.java

# Check for known vulnerable libraries
find . -name "*.so" | xargs strings | grep -i "openssl\|busybox\|dropbear"
```

---

## Hardware Interfaces: UART / JTAG / SPI / I2C

```
Interface   Pins              Use Case
---------   ----              --------
UART        TX, RX, GND      Serial console, bootloader access, debug shells
JTAG        TDI,TDO,TCK,     Full debug: read/write memory, halt CPU,
            TMS, TRST, GND   extract firmware, set breakpoints
SPI         MOSI,MISO,CLK,   Read/write flash chips (firmware extraction)
            CS, GND
I2C         SDA, SCL, GND    Read EEPROM, sensor data, config
```

```bash
# Identify UART pins with JTAGulator or multimeter
# TX: fluctuating voltage when device boots
# RX: steady high voltage
# GND: 0V to chassis ground

# SPI flash dump
flashrom -p ch341a_spi -r dump.bin

# I2C EEPROM read (Bus Pirate)
i2cdump -y 1 0x50

# JTAG boundary scan with JTAGulator
# Auto-detect pinout: connect all test points, run IDCODE scan
```

---

## IoT Device Discovery: Shodan & Censys

```bash
# Shodan searches
shodan search "mqtt" --fields ip_str,port,org
shodan search "port:1883"                          # MQTT brokers
shodan search "port:47808 product:BACnet"          # building automation
shodan search "Server: GoAhead-Webs" country:US    # IoT webcams
shodan search "port:5683"                          # CoAP devices
shodan search "mikrotik" country:US                # routers

# Shodan API usage
shodan init YOUR_API_KEY
shodan host 1.2.3.4
shodan stats --facets country "port:1883 mqtt"

# Censys searches
censys search "services.mqtt" --index-type hosts
censys search "services.port=502"                  # Modbus devices
```

---

## MQTT Attacks

```bash
# Connect to unauthenticated broker
mosquitto_sub -h target -t '#' -v          # subscribe to ALL topics
mosquitto_sub -h target -t '$SYS/#' -v     # broker system info

# Topic enumeration
mosquitto_sub -h target -t '+/+' -v        # single-level wildcard
mosquitto_sub -h target -t '+/+/+/#' -v    # multi-level discovery

# Message injection
mosquitto_pub -h target -t "home/door/lock" -m "UNLOCK"
mosquitto_pub -h target -t "factory/plc/cmd" -m '{"action":"stop"}'

# Brute-force MQTT credentials
ncrack -p 1883 --user admin -P wordlist.txt mqtt://target

# MQTT fuzzing
# Use mqtt-pwn or custom scripts to fuzz topic names & payloads
pip install mqtt-pwn
mqtt-pwn
```

---

## OT / ICS / SCADA

### Purdue Model (ISA-95)

```
Level 5    Enterprise Network       ERP, email, internet
Level 4    Business Planning        MES, historian, DMZ
           ── IT/OT Boundary (Demilitarized Zone) ──
Level 3    Site Operations          SCADA servers, historians
Level 2    Area Supervisory         HMI, engineering workstations
Level 1    Basic Control            PLCs, RTUs, DCS controllers
Level 0    Physical Process         Sensors, actuators, valves, motors
```

### ICS Protocols

```
Protocol       Port     Transport   Auth?   Encryption?   Notes
--------       ----     ---------   -----   -----------   -----
Modbus TCP     502      TCP         No      No            Read/write coils & registers
DNP3           20000    TCP/serial  Optional Optional     SCADA, outstation polling
OPC UA         4840     TCP         Yes     TLS           Modern, replaces OPC DA
S7comm         102      TCP         No      No            Siemens S7 PLCs
EtherNet/IP    44818    TCP/UDP     No      No            CIP over Ethernet
BACnet/IP      47808    UDP         No      No            Building automation
PROFINET       ---      Ethernet    No      No            Siemens industrial Ethernet
```

---

## OT Vulnerability Scanning

```bash
# Nmap ICS/SCADA scripts
nmap -sU -p 47808 --script bacnet-info target         # BACnet discovery
nmap -p 502 --script modbus-discover target            # Modbus device info
nmap -p 102 --script s7-info target                    # Siemens S7 info
nmap -p 44818 --script enip-info target                # EtherNet/IP info
nmap -p 20000 --script dnp3-info target                # DNP3 info
nmap -p 4840 --script opcua-endpoints target           # OPC UA endpoints

# Redpoint (ICS Nmap scripts collection)
git clone https://github.com/digitalbond/Redpoint
nmap --script-path Redpoint/ -p 502 --script modicon-info target

# plcscan - PLC scanner
python plcscan.py -t target -p 502                     # Modbus scan
python plcscan.py -t target -p 102 --s7                # S7comm scan

# Modbus interaction
modbus-cli read target 0 10                            # Read 10 holding registers
modbus-cli write target 0 1234                         # Write to register 0
pip install pymodbus
# Use pymodbus for custom scripts
```

---

## Notable SCADA/ICS Attacks

```
Attack              Year    Target                   Technique
------              ----    ------                   ---------
Stuxnet             2010    Iran uranium centrifuges  USB propagation, 4 zero-days,
                                                     modified S7-315/417 PLC code
TRITON/TRISIS       2017    Saudi petrochemical SIS   Targeted Triconex safety
                                                     controllers, RAT on EWS
Ukraine Power Grid  2015    Ukrainian power companies BlackEnergy trojan, KillDisk,
                    2016                              Industroyer/CrashOverride
Havex/Dragonfly     2014    ICS vendors               Trojanized ICS software,
                                                     OPC DA scanning
Oldsmar Water       2021    Florida water treatment   Remote access to HMI,
                                                     NaOH level manipulation
```

---

## ICS Network Segmentation (ISA/IEC 62443)

```
Concept              Description
-------              -----------
Zones                Groups of assets with same security requirements
Conduits             Communication paths between zones (firewalled, monitored)
Security Levels      SL 0-4 (0=none, 4=state-level threat resistance)
IDMZ                 Industrial DMZ between IT and OT networks
Unidirectional       Data diodes: OT -> IT only (Waterfall, Owl)
Gateways
```

```
Recommended Architecture:
  Internet <-> Corporate FW <-> IT Network <-> IDMZ <-> OT FW <-> SCADA/DCS
                                                                    |
                                                              PLCs / RTUs
  - No direct internet access to OT
  - Data diodes for historian replication
  - Jump servers for remote OT access (MFA required)
  - Separate AD forest or local accounts for OT
```

---

## OWASP IoT Top 10

```
#    Vulnerability                        Example
--   -------------                        -------
I1   Weak/Guessable/Hardcoded Passwords   admin:admin, root:root
I2   Insecure Network Services            Open Telnet, UPnP, debug ports
I3   Insecure Ecosystem Interfaces        API without auth, IDOR in cloud
I4   Lack of Secure Update Mechanism      No signed firmware, HTTP OTA
I5   Use of Insecure/Outdated Components  Old OpenSSL, BusyBox CVEs
I6   Insufficient Privacy Protection      PII in plaintext, no consent
I7   Insecure Data Transfer/Storage       No TLS, plaintext credentials
I8   Lack of Device Management            No inventory, no patching plan
I9   Insecure Default Settings            Debug enabled, default creds
I10  Lack of Physical Hardening           Exposed UART/JTAG, no tamper detect
```

---

## Countermeasures

```
Area                Countermeasure
----                --------------
Firmware            Secure boot chain, code signing, encrypted storage
OTA Updates         Signed packages, TLS transport, rollback protection
Network             Segment IoT/OT from IT, VLAN isolation, micro-segmentation
Protocols           Use TLS/DTLS, MQTT over TLS (8883), CoAPS (5684)
Authentication      Unique per-device credentials, certificate-based auth, no defaults
ICS Monitoring      Dragos Platform, Claroty CTD, Nozomi Guardian, SecurityBridge
Physical            Disable JTAG in production, epoxy debug ports, tamper-evident seals
Cloud               API rate limiting, OAuth/JWT, input validation, WAF
```

---

## Tips

- Always check for default credentials first; most IoT compromises start there
- Firmware analysis often reveals more than network testing; extract and grep before scanning
- MQTT brokers without auth are shockingly common; `mosquitto_sub -t '#'` is the first test
- ICS/SCADA scanning must be done with extreme caution; active probing can crash PLCs
- Use passive monitoring (Zeek, Suricata with ICS rulesets) in OT environments when possible
- Shodan `has_screenshot:true` filter finds exposed HMIs and webcams quickly
- The Purdue model is heavily tested on CEH; know which devices live at each level
- BLE and ZigBee attacks require specialized hardware (Ubertooth, ApiMote, HackRF)
- For CEH exam: know the OWASP IoT Top 10 categories and at least one example each
- ISA/IEC 62443 zones and conduits are the gold standard for ICS network segmentation

---

## See Also

- `sheets/offensive/wireless-hacking.md` -- BLE/ZigBee overlap
- `sheets/offensive/network-attacks.md` -- MITM and replay fundamentals
- `sheets/offensive/cloud-security.md` -- IoT cloud API testing
- `sheets/defensive/network-defense.md` -- segmentation and monitoring

---

## References

- CEH v13 Module 18: IoT and OT Hacking
- OWASP IoT Top 10 (2018): https://owasp.org/www-project-internet-of-things/
- ISA/IEC 62443: https://www.isa.org/standards-and-publications/isa-standards/isa-iec-62443-series-of-standards
- Shodan: https://www.shodan.io/
- NIST SP 800-82 Rev 3: Guide to OT Security
- MITRE ATT&CK for ICS: https://attack.mitre.org/techniques/ics/
- Redpoint ICS Scripts: https://github.com/digitalbond/Redpoint
- Eclipse Mosquitto: https://mosquitto.org/
- Binwalk: https://github.com/ReFirmLabs/binwalk
