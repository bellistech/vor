# Network Access Control (NAC)

Framework for controlling endpoint access to the network based on identity, device health, and compliance posture. Enforces pre-admission (before network access) and post-admission (continuous monitoring) policies across wired, wireless, and VPN connections.

## Architecture

### NAC Components

```
Endpoint              Network Device           NAC Server           Policy/Directory
(Supplicant/Agent)    (Authenticator)          (RADIUS/Policy)      (AD, LDAP, MDM)
       |                    |                       |                      |
       |--- Credentials --->|                       |                      |
       |                    |--- RADIUS ----------->|                      |
       |                    |                       |--- Identity query --->|
       |                    |                       |<-- User/group info ---|
       |                    |                       |                      |
       |<-- Posture req ----|<-- Challenge ---------|                      |
       |--- Posture data -->|--- Posture data ----->|                      |
       |                    |                       | [Evaluate policy]    |
       |                    |<-- Access-Accept -----|                      |
       |                    |    + VLAN/ACL/SGT     |                      |
       | [Network access]  |                       |                      |
```

### Pre-Admission vs Post-Admission

| Phase | Timing | Purpose | Enforcement |
|-------|--------|---------|-------------|
| Pre-admission | Before network access granted | Verify identity and compliance | Block/quarantine until compliant |
| Post-admission | After access granted, ongoing | Monitor for compliance changes | CoA to re-evaluate, quarantine, or disconnect |

### NAC Deployment Phases

```
Phase 1: Visibility (Monitor Mode)
  - Deploy 802.1X / MAB in open mode
  - Authenticate but do not enforce (permit all)
  - Log all authentication events
  - Profile all endpoints
  - Duration: 4-8 weeks

Phase 2: Low-Impact (Selective Enforcement)
  - Apply pre-auth ACL (permit DHCP, DNS, AD)
  - Enforce for specific groups (IT pilot)
  - Quarantine VLAN for non-compliant
  - Monitor false-positive rate
  - Duration: 4-8 weeks

Phase 3: Closed Mode (Full Enforcement)
  - Default deny without authentication
  - Full policy enforcement (VLAN, ACL, SGT)
  - Posture assessment required
  - Guest and BYOD policies active
  - Ongoing operations
```

## 802.1X-Based NAC

See the `dot1x` sheet for detailed 802.1X protocol coverage.

### 802.1X with NAC Integration

```
! Switch port for full NAC (802.1X + MAB + WebAuth)

interface GigabitEthernet1/0/1
 description NAC-Enabled Access Port
 switchport mode access
 switchport access vlan 20

 ! Authentication configuration
 authentication host-mode multi-auth
 authentication order dot1x mab
 authentication priority dot1x mab
 authentication port-control auto
 authentication periodic
 authentication timer reauthenticate server

 ! 802.1X
 dot1x pae authenticator
 dot1x timeout tx-period 10

 ! MAB fallback
 mab

 ! Pre-auth ACL (limited access before full auth)
 ip access-group PRE-AUTH-ACL in

 ! Spanning tree for access ports
 spanning-tree portfast
 spanning-tree bpduguard enable

! Pre-auth ACL — minimum access before authentication
ip access-list extended PRE-AUTH-ACL
 permit udp any any eq bootps        ! DHCP
 permit udp any any eq bootpc
 permit udp any any eq domain        ! DNS
 permit udp any host 10.1.1.50 eq 1812  ! RADIUS (for WebAuth)
 deny   ip any any
```

### Authentication Order and Priority

```
! Authentication order — which methods to try
authentication order dot1x mab webauth

! Order determines the sequence:
! 1. dot1x — try 802.1X first (wait for EAP)
! 2. mab — if 802.1X times out, try MAB
! 3. webauth — if MAB fails, redirect to web portal

! Authentication priority — which method can preempt
authentication priority dot1x mab

! If a higher-priority method starts while a lower-priority
! method has already authenticated, the higher-priority
! method takes over (e.g., dot1x preempts mab)

! Host modes:
! single-host    — one device per port
! multi-host     — one auth, all devices on port get access
! multi-domain   — one data device + one voice device
! multi-auth     — each device authenticates independently
```

## MAC Authentication Bypass (MAB)

### MAB Overview

```
! MAB authenticates devices that cannot run 802.1X supplicant
! (printers, IP phones, cameras, IoT sensors, badge readers)

! MAB flow:
! 1. Switch detects Ethernet frame from endpoint
! 2. 802.1X times out (no EAP response from endpoint)
! 3. Switch sends RADIUS Access-Request with MAC as username/password
! 4. RADIUS server looks up MAC in endpoint database
! 5. If found → Access-Accept with policy
! 6. If not found → Access-Reject (or guest policy)

! RADIUS attributes for MAB:
! User-Name: AA:BB:CC:DD:EE:FF (or aabbccddeeff, varies by format)
! User-Password: AA:BB:CC:DD:EE:FF (same as username)
! Service-Type: Call-Check (10)
! NAS-Port-Type: Ethernet (15)
```

### MAB Configuration

```
! Switch MAB configuration
interface GigabitEthernet1/0/10
 description Printer - MAB Only
 switchport mode access
 switchport access vlan 30
 authentication port-control auto
 mab
 dot1x pae authenticator
 dot1x timeout tx-period 3      ! shorter timeout for faster MAB fallback
 spanning-tree portfast

! MAB format options (global)
mab request format attribute 1 groupsize 2 separator :  lowercase
! Sends MAC as aa:bb:cc:dd:ee:ff

! Alternative: Send MAC as aabbccddeeff (no separator)
mab request format attribute 1 groupsize 12 separator "" lowercase
```

### ISE MAB Policy

```
! ISE policy for MAB authentication
! (See cisco-ise sheet for full ISE policy configuration)

Policy Set: "Wired_MAB"
  ├── Condition: Radius:Service-Type == Call-Check
  │
  ├── Authentication:
  │   └── Identity Source: Internal Endpoints DB
  │
  └── Authorization:
      ├── Rule 1: EndpointProfile=Cisco-IP-Phone → Voice_VLAN + SGT=Voice
      ├── Rule 2: EndpointProfile=Printer → Printer_VLAN + SGT=Printer
      ├── Rule 3: EndpointProfile=Camera → Camera_VLAN + SGT=IoT
      ├── Rule 4: EndpointIdentityGroup=RegisteredDevices → Data_VLAN
      └── Default: Guest_VLAN (limited access)
```

## Posture Assessment

### Posture Architecture

```
Endpoint                  NAC Agent              ISE                  Remediation
    |                        |                    |                       |
    |-- Connect to network ->|                    |                       |
    |                        |-- 802.1X auth ---->|                       |
    |                        |<- Accept + posture |                       |
    |                        |   redirect         |                       |
    |                        |                    |                       |
    |                        |-- Posture check -->|                       |
    |                        |   (AV version,     |                       |
    |                        |    patches, FW,    |                       |
    |                        |    encryption)     |                       |
    |                        |                    |                       |
    |                        |<- Non-compliant ---|                       |
    |                        |   (missing patch)  |                       |
    |                        |                    |                       |
    |<- Remediation prompt --|                    |                       |
    |-- Install patch ------>|                    |                       |
    |                        |                    |                       |
    |                        |-- Re-check ------->|                       |
    |                        |<- Compliant -------|                       |
    |                        |                    |-- CoA (full access) ->| Switch
    |                        |                    |                       |
    | [Full network access]  |                    |                       |
```

### Posture Conditions

| Category | Checks | Platform |
|----------|--------|----------|
| Anti-Virus | AV installed, version, definition date | Windows, macOS |
| Anti-Malware | AM product active, up to date | Windows, macOS |
| Patch Management | OS patch level, specific KB updates | Windows |
| Disk Encryption | BitLocker, FileVault enabled | Windows, macOS |
| Firewall | Host firewall enabled and running | Windows, macOS, Linux |
| Application | Required apps present, prohibited apps absent | All |
| File | Specific file exists/absent, file hash match | All |
| Registry | Registry key values (Windows) | Windows |
| Service | Required services running | Windows, Linux |
| USB | USB mass storage blocked | Windows |

### Agent-Based vs Agentless

| Feature | Agent-Based (AnyConnect ISE Posture) | Agentless (Temporal Agent) |
|---------|--------------------------------------|---------------------------|
| Installation | Persistent agent installed | Downloaded on-demand, runs once |
| Checks | Full posture (AV, patches, encryption, registry) | Limited (OS, AV, patches) |
| Remediation | Agent can auto-remediate | Manual remediation only |
| Platforms | Windows, macOS, Linux | Windows, macOS |
| User experience | Transparent after install | Requires download each session |
| Best for | Managed corporate devices | BYOD, contractor laptops |
| Continuous monitoring | Yes (periodic re-assessment) | No (one-time check) |

### Posture Policy Configuration

```
! ISE posture policy example
! (Administration > System > Settings > Posture)

! Requirement: Windows Patch Compliance
! Condition: Windows patches current within 30 days
! Remediation: Redirect to WSUS for patch installation
! Grace period: 4 hours (allow temporary access while patching)

! Requirement: Antivirus Active
! Condition: AV definition date within 3 days
! Remediation: Link to AV update server
! Grace period: none (must remediate before full access)

! Posture policy (per OS):
! Windows:
!   - AV: CrowdStrike, Defender, or Symantec installed + definitions < 3 days
!   - Patches: All critical patches installed
!   - Firewall: Windows Firewall enabled
!   - Encryption: BitLocker on C: drive
! macOS:
!   - AV: CrowdStrike or XProtect active
!   - Patches: macOS version >= 14.0
!   - Firewall: macOS Firewall enabled
!   - Encryption: FileVault enabled
```

## Remediation

### Quarantine Strategies

| Strategy | Mechanism | Access During Quarantine |
|----------|-----------|------------------------|
| Quarantine VLAN | Assign non-compliant endpoint to isolated VLAN | Remediation servers only |
| Downloadable ACL (dACL) | Push restrictive ACL to switch port | DNS, DHCP, remediation servers |
| URL redirect | Redirect HTTP to ISE remediation portal | Web portal with instructions |
| SGT quarantine | Assign quarantine SGT, enforce with SGACL | Limited by SGACL policy |

### Quarantine VLAN Design

```
! Quarantine VLAN network design

! VLAN 999 — Quarantine VLAN
interface Vlan999
 ip address 10.99.0.1 255.255.0.0
 ip helper-address 10.1.1.10      ! DHCP server
 ip access-group QUARANTINE-ACL in

! Quarantine ACL — permit only remediation traffic
ip access-list extended QUARANTINE-ACL
 ! DNS (required for name resolution)
 permit udp any any eq domain
 ! DHCP (required for IP assignment)
 permit udp any any eq bootps
 permit udp any any eq bootpc
 ! WSUS / SCCM (patch remediation)
 permit tcp any host 10.1.1.20 eq 8530
 permit tcp any host 10.1.1.20 eq 8531
 ! AV update server
 permit tcp any host 10.1.1.21 eq 443
 ! ISE portals (posture re-check)
 permit tcp any host 10.1.1.50 eq 8443
 permit tcp any host 10.1.1.51 eq 8443
 ! Block everything else
 deny ip any any log
```

### Change of Authorization (CoA)

```
! CoA enables dynamic policy changes after initial authentication
! (See cisco-ise sheet for detailed CoA coverage)

! CoA flow for posture state change:
! 1. Endpoint authenticates → limited access (posture unknown)
! 2. Posture agent checks endpoint → reports compliant
! 3. ISE sends CoA to switch → reauthenticate port
! 4. Switch re-auths → ISE grants full access
!
! 5. Later: AV definitions expire → posture reports non-compliant
! 6. ISE sends CoA → reauthenticate port
! 7. Switch re-auths → ISE assigns quarantine VLAN

! Switch CoA configuration
aaa server radius dynamic-author
 client 10.1.1.50 server-key RadiusSecret
 client 10.1.1.51 server-key RadiusSecret

! CoA types:
! - Reauthenticate: re-run authentication (apply new policy)
! - Port bounce: link down/up (force DHCP renewal for VLAN change)
! - Terminate: disconnect session immediately
```

## Profiling-Based NAC

```
! NAC uses profiling to identify device types and apply appropriate policy
! (See cisco-ise sheet for detailed profiling coverage)

! Profiling enables:
! 1. Automatic policy assignment based on device type
! 2. IoT device identification without 802.1X
! 3. Rogue device detection (unknown profiles)
! 4. Asset inventory and visibility

! Common profile-based policies:
! - IP Phone → Voice VLAN, QoS marking
! - Printer → Printer VLAN, restricted ACL
! - Security Camera → IoT VLAN, strict ACL
! - Medical Device → Clinical VLAN, regulatory ACL
! - Unknown → Guest VLAN or quarantine

! Profiling + MAB = NAC for unmanaged devices:
! 1. Device connects → MAB authenticates MAC
! 2. ISE profiles device (DHCP fingerprint, CDP/LLDP, HTTP UA)
! 3. ISE assigns policy based on profile match
! 4. CoA updates policy as profile certainty increases
```

## Guest NAC

### Guest Access Flow

```
Guest                Switch/WLC              ISE                  Sponsor
  |                      |                    |                      |
  |-- Connect (no creds)->|                   |                      |
  |                      |--- MAB or open --->|                      |
  |                      |<-- Accept + URL -->|                      |
  |                      |    redirect        |                      |
  |                      |                    |                      |
  |<-- Redirect to ------|                    |                      |
  |    guest portal      |                    |                      |
  |                      |                    |                      |
  | Option A: Self-registration              |                      |
  |--- Register on portal ----------------->|                      |
  |<-- Credentials (via email/SMS) ----------|                      |
  |--- Login with credentials -------------->|                      |
  |                      |<-- CoA (guest) ----|                      |
  |                      |                    |                      |
  | Option B: Sponsored access              |                      |
  |                      |                    |<-- Create guest -----|
  |                      |                    |    account           |
  |--- Login with sponsored credentials --->|                      |
  |                      |<-- CoA (guest) ----|                      |
  |                      |                    |                      |
  | [Internet access, no internal access]    |                      |
```

### Guest Policy

```
! Guest VLAN / ACL — internet access only
ip access-list extended GUEST-ACL
 permit udp any any eq domain        ! DNS
 permit udp any any eq bootps        ! DHCP
 permit tcp any any eq 80            ! HTTP
 permit tcp any any eq 443           ! HTTPS
 deny   ip any 10.0.0.0 0.255.255.255  ! Block internal RFC1918
 deny   ip any 172.16.0.0 0.15.255.255
 deny   ip any 192.168.0.0 0.0.255.255
 permit ip any any                   ! Allow internet

! Guest access time limits:
! - Duration: 1 day, 3 days, 1 week (configurable)
! - Auto-expire: account disabled after duration
! - Rate limit: bandwidth cap (e.g., 10 Mbps)
! - Acceptable Use Policy: user must accept before access
```

## IoT NAC

### IoT Challenges

```
! IoT devices present unique NAC challenges:
! 1. No 802.1X supplicant capability
! 2. Diverse protocols (MQTT, CoAP, BACnet, Modbus)
! 3. No ability to run posture agent
! 4. Long lifespans (10+ years, rarely patched)
! 5. Vendor-specific behavior (non-standard DHCP, no HTTP)

! IoT NAC strategy:
! 1. Identify: Profiling (DHCP fingerprint, MAC OUI, traffic patterns)
! 2. Authenticate: MAB with known MAC database
! 3. Segment: Dedicated IoT VLANs per function
! 4. Restrict: Strict ACLs (only required protocols/destinations)
! 5. Monitor: Behavioral anomaly detection

! IoT segmentation example:
! VLAN 100: Building Management (HVAC, lighting)
! VLAN 101: Security Systems (cameras, badge readers)
! VLAN 102: Medical Devices (pumps, monitors)
! VLAN 103: Industrial IoT (PLCs, SCADA)
```

### IoT Policy Example

```
! ISE policy for IoT devices
Policy Set: "IoT_Devices"
  ├── Condition: Radius:Service-Type == Call-Check
  │              AND EndpointProfile starts-with "IoT-"
  │
  └── Authorization:
      ├── Rule 1: Profile=IoT-Camera
      │   → VLAN 101, dACL: permit tcp any host <NVR> eq 554 (RTSP)
      │     permit udp any host <NVR> (RTP), deny ip any any
      │
      ├── Rule 2: Profile=IoT-HVAC
      │   → VLAN 100, dACL: permit tcp any host <BMS> eq 47808 (BACnet)
      │     deny ip any any
      │
      ├── Rule 3: Profile=IoT-BadgeReader
      │   → VLAN 101, dACL: permit tcp any host <AccessCtrl> eq 3001
      │     deny ip any any
      │
      └── Default: Quarantine (unknown IoT)
```

## NAC for Wired, Wireless, and VPN

### Deployment Differences

| Aspect | Wired | Wireless | VPN |
|--------|-------|----------|-----|
| Authenticator | Switch (dot1x pae) | WLC (WLAN settings) | VPN headend (ASA/FTD) |
| Primary auth | 802.1X / MAB | 802.1X / MAC filter | Certificate / MFA |
| Enforcement | Port VLAN, dACL, SGT | WLAN VLAN, ACL, SGT | VPN group policy, ACL |
| Posture | AnyConnect ISE module | AnyConnect ISE module | AnyConnect ISE module |
| CoA support | Yes (RFC 5176) | Yes (via WLC) | Yes (via ASA/FTD) |
| Profiling | DHCP, CDP/LLDP, HTTP | DHCP, HTTP, RF fingerprint | Limited (IP, user-agent) |

### Wireless NAC Configuration

```
! WLC 802.1X WLAN for NAC
! (Cisco WLC 9800 IOS-XE)

wlan CORP-SECURE 1 CORP-SECURE
 security wpa wpa2
 security wpa wpa2 ciphers aes
 security dot1x authentication-list ISE-RADIUS
 aaa-override
 nac
 session-timeout 28800
 no shutdown

! AAA for wireless
aaa new-model
aaa authentication dot1x ISE-RADIUS group radius
aaa authorization network ISE-RADIUS group radius
aaa accounting dot1x default start-stop group radius

! RADIUS server for WLC
radius server ISE-PSN1
 address ipv4 10.1.1.50 auth-port 1812 acct-port 1813
 key RadiusSecret123

! CoA support
aaa server radius dynamic-author
 client 10.1.1.50 server-key RadiusSecret123
```

### VPN NAC Configuration

```
! ASA VPN with ISE posture assessment

! RADIUS pointing to ISE
aaa-server ISE-RADIUS protocol radius
 aaa-server ISE-RADIUS (inside) host 10.1.1.50
  key RadiusSecret123
  radius-common-pw RadiusSecret123

! VPN tunnel group with ISE authentication
tunnel-group CORP-VPN general-attributes
 authentication-server-group ISE-RADIUS
 default-group-policy CORP-VPN-POLICY
 address-pool VPN-POOL

! CoA for VPN sessions
aaa-server ISE-RADIUS protocol radius
 dynamic-authorization

! VPN posture flow:
! 1. User connects VPN with AnyConnect
! 2. ASA authenticates user via RADIUS to ISE
! 3. ISE returns limited ACL (posture-required)
! 4. AnyConnect ISE posture module checks endpoint
! 5. ISE receives posture report
! 6. ISE sends CoA to ASA with full-access ACL
! 7. ASA applies new ACL to VPN session
```

## Tips

- Always start NAC deployment in monitor mode (Phase 1) for at least 4 weeks to understand your endpoint population before enforcing.
- Use profiling combined with MAB for IoT devices; do not attempt to install 802.1X supplicants on printers and cameras.
- Design quarantine VLANs with access to DNS, DHCP, and remediation servers; without these, endpoints cannot self-remediate.
- Deploy posture assessment in audit mode before enforcement to identify non-compliant endpoints without blocking users.
- Use multi-auth host-mode on access ports to support IP phones and computers on the same port with independent authentication.
- MAB is inherently weaker than 802.1X (MAC addresses can be spoofed); layer it with profiling for additional confidence.
- CoA requires RADIUS dynamic authorization configured on every switch; automate this with configuration templates.
- Guest NAC should enforce acceptable use policy acceptance and time-limited access with automatic expiration.
- For IoT, apply the principle of least privilege: each device type gets an ACL allowing only the protocols and destinations it needs.
- Test NAC failover by shutting down a RADIUS server; verify that switches fall back to the secondary server without dropping authenticated sessions.

## See Also

- dot1x, cisco-ise, radius, zero-trust, cisco-ftd

## References

- [Cisco ISE NAC Deployment Guide](https://www.cisco.com/c/en/us/td/docs/security/ise/3-2/admin_guide/b_ise_admin_3_2/b_ise_admin_3_2_chapter_010110.html)
- [Cisco TrustSec Design Guide](https://www.cisco.com/c/en/us/solutions/enterprise-networks/trustsec/design-guide-series.html)
- [Cisco ISE Prescriptive Deployment Guide](https://community.cisco.com/t5/security-knowledge-base/ise-secure-wired-access-prescriptive-deployment-guide/ta-p/3641515)
- [RFC 3748 — Extensible Authentication Protocol (EAP)](https://www.rfc-editor.org/rfc/rfc3748)
- [RFC 5176 — Dynamic Authorization Extensions to RADIUS (CoA)](https://www.rfc-editor.org/rfc/rfc5176)
- [IEEE 802.1X-2020 — Port-Based Network Access Control](https://standards.ieee.org/standard/802_1X-2020.html)
- [NIST SP 800-207 — Zero Trust Architecture](https://csrc.nist.gov/publications/detail/sp/800-207/final)
