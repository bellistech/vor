# 802.1X (Port-Based Network Access Control with EAP)

> IEEE standard for port-based network access control using EAP authentication over LAN (EAPOL) with RADIUS backend.

## Architecture

### Three-Party Model

```
Supplicant              Authenticator            Authentication Server
(Client)                (Switch/AP)              (RADIUS — ISE, FreeRADIUS)
   |                        |                           |
   |--- EAPOL-Start ------->|                           |
   |<-- EAP-Request/ID -----|                           |
   |--- EAP-Response/ID --->|                           |
   |                        |--- RADIUS Access-Req ---->|
   |                        |<-- RADIUS Access-Chal ----|
   |<-- EAP-Request/Chal ---|                           |
   |--- EAP-Response/Chal ->|                           |
   |                        |--- RADIUS Access-Req ---->|
   |                        |<-- RADIUS Access-Accept --|  # with VLAN, ACL attributes
   |<-- EAP-Success --------|                           |
   |                        |                           |
   |=== PORT AUTHORIZED ====|                           |
```

### Port States

```
# Unauthorized — only EAPOL traffic allowed (default)
# Authorized   — all traffic forwarded after successful auth
# Force-Auth   — always authorized (bypass 802.1X)
# Force-Unauth — always blocked (admin lockout)

# Port control modes
interface GigabitEthernet1/0/1
  authentication port-control auto         # normal 802.1X
  authentication port-control force-authorized    # bypass
  authentication port-control force-unauthorized  # block
```

## EAP Types

### Comparison

```
EAP Type    | Server Cert | Client Cert | Inner Auth     | Security | Complexity
------------|-------------|-------------|----------------|----------|----------
EAP-TLS     | Yes         | Yes         | N/A (mutual)   | Highest  | High (PKI)
PEAP        | Yes         | No          | MSCHAPv2/EAP   | High     | Medium
EAP-TTLS    | Yes         | No          | PAP/CHAP/MSCHAP| High     | Medium
EAP-FAST    | Optional    | No          | PAC/EAP        | High     | Low
EAP-MD5     | No          | No          | MD5 hash       | Low      | Low
LEAP        | No          | No          | MS-CHAP        | Broken   | Low
```

### EAP-TLS (Most Secure)

```
# Mutual certificate authentication — both client and server present certs
# Requires PKI: CA, server cert, per-device client cert
# Immune to credential theft (no passwords transmitted)
# Used for: corporate laptops, managed devices

Supplicant                  RADIUS
   |--- ClientHello -------->|
   |<-- ServerHello ---------|
   |<-- Server Certificate --|
   |<-- Certificate Request -|
   |--- Client Certificate ->|
   |--- ClientKeyExchange -->|
   |=== TLS Tunnel ===========|
```

### PEAP (Most Common)

```
# Server-side TLS tunnel, then inner MSCHAPv2 authentication
# Only server needs a certificate
# Versions: PEAPv0 (MS-CHAPv2), PEAPv1 (EAP-GTC)
# Used for: Windows environments, Active Directory

# Phase 1: TLS tunnel established (server cert validates)
# Phase 2: MSCHAPv2 runs inside the tunnel (username/password)

# Windows registry — disable server cert validation (LAB ONLY)
# HKLM\SOFTWARE\Microsoft\EAPOL\Parameters\General
# AuthMode = 2
```

### EAP-FAST (Cisco)

```
# Flexible Authentication via Secure Tunneling
# Uses Protected Access Credentials (PAC) instead of certificates
# PAC provisioning: automatic (Phase 0) or manual
# Supports chaining (machine + user auth in one session)

# Phase 0: PAC provisioned via anonymous Diffie-Hellman (optional)
# Phase 1: TLS tunnel using PAC
# Phase 2: Inner EAP method (MSCHAPv2, GTC, TLS)
```

### EAP-TTLS

```
# Tunneled TLS — similar to PEAP but more flexible inner methods
# Supports PAP inside tunnel (useful for LDAP/token backends)
# Less common on Windows (native support added in Win 8+)
# Popular in eduroam deployments

# Inner methods: PAP, CHAP, MS-CHAPv1, MS-CHAPv2, EAP
```

## IOS-XE Configuration

### AAA Configuration

```
! Enable AAA
aaa new-model

! Authentication method list for 802.1X
aaa authentication dot1x default group radius

! Authorization for downloadable ACLs and VLAN assignment
aaa authorization network default group radius

! Accounting for session tracking
aaa accounting dot1x default start-stop group radius

! Enable CoA (Change of Authorization)
aaa server radius dynamic-author
  client 10.1.1.100 server-key Cisco123
  server-key Cisco123
  auth-type any

! Enable 802.1X globally
dot1x system-auth-control
```

### RADIUS Server Configuration

```
! RADIUS server definition (IOS-XE)
radius server ISE-PRIMARY
  address ipv4 10.1.1.100 auth-port 1812 acct-port 1813
  key 0 RadiusSharedSecret
  automate-tester username probe-user probe-on

radius server ISE-SECONDARY
  address ipv4 10.1.1.101 auth-port 1812 acct-port 1813
  key 0 RadiusSharedSecret

! Server group
aaa group server radius ISE-SERVERS
  server name ISE-PRIMARY
  server name ISE-SECONDARY
  deadtime 15
  load-balance method least-outstanding

! Global RADIUS source interface
ip radius source-interface Loopback0
```

### Interface Configuration

```
! Standard 802.1X port config
interface GigabitEthernet1/0/1
  description ACCESS-PORT
  switchport mode access
  switchport access vlan 100
  authentication port-control auto
  authentication periodic
  authentication timer reauthenticate server
  dot1x pae authenticator
  dot1x timeout tx-period 10
  dot1x max-reauth-req 2
  spanning-tree portfast
  spanning-tree bpduguard enable
```

### Multi-Auth / Multi-Domain / Multi-Host

```
! Host modes — how many clients per port
interface GigabitEthernet1/0/1

  ! Single-host: one MAC, one session (default)
  authentication host-mode single-host

  ! Multi-host: one auth, all MACs allowed after (hub/IP phone)
  authentication host-mode multi-host

  ! Multi-domain: one voice + one data (IP phone + PC)
  authentication host-mode multi-domain

  ! Multi-auth: each MAC authenticates independently
  authentication host-mode multi-auth

! Multi-domain example (IP phone + PC)
interface GigabitEthernet1/0/1
  switchport mode access
  switchport access vlan 100
  switchport voice vlan 200
  authentication host-mode multi-domain
  authentication port-control auto
  dot1x pae authenticator
  mab
  authentication order dot1x mab
  authentication priority dot1x mab
```

## VLAN Assignment

### Dynamic VLAN via RADIUS

```
# RADIUS attributes returned in Access-Accept:
#   Tunnel-Type            = VLAN (13)
#   Tunnel-Medium-Type     = IEEE-802 (6)
#   Tunnel-Private-Group-Id = <VLAN-ID or VLAN-Name>

# ISE authorization profile returns:
#   (64) Tunnel-Type              = VLAN
#   (65) Tunnel-Medium-Type       = 802
#   (81) Tunnel-Private-Group-Id  = 100

# Switch must have the assigned VLAN in its database
# Port moves from access VLAN to RADIUS-assigned VLAN
```

### Guest VLAN

```
! Assign clients that never attempt 802.1X
interface GigabitEthernet1/0/1
  authentication event no-response action authorize vlan 999
  ! or legacy syntax:
  dot1x guest-vlan 999

! Guest VLAN triggers when:
#   - Supplicant sends no EAPOL-Start
#   - tx-period expires (default 30s x 3 retries)
#   - Non-802.1X capable device connected
```

### Auth-Fail VLAN

```
! Assign clients that fail authentication
interface GigabitEthernet1/0/1
  authentication event fail action authorize vlan 998
  authentication event fail retry 2 action authorize vlan 998
  ! or legacy:
  dot1x auth-fail-vlan 998

! Auth-fail VLAN triggers when:
#   - RADIUS returns Access-Reject
#   - Credentials invalid
#   - Certificate expired or untrusted
```

### Critical VLAN (Server-Dead)

```
! Assign when RADIUS server is unreachable
interface GigabitEthernet1/0/1
  authentication event server dead action authorize vlan 997
  authentication event server dead action authorize voice
  authentication event server alive action reinitialize

! Critical VLAN triggers when:
#   - All RADIUS servers marked dead
#   - Network connectivity loss to auth servers
#   - Used for business continuity during outages
```

## MAB (MAC Authentication Bypass)

### Configuration

```
! MAB as fallback after 802.1X timeout
interface GigabitEthernet1/0/1
  authentication port-control auto
  dot1x pae authenticator
  mab                              ! enable MAB
  authentication order dot1x mab   ! try 802.1X first, then MAB
  authentication priority dot1x mab

! MAB sends MAC address as both username and password to RADIUS
! Format options:
mab request format attribute 1 groupsize 2 separator : lowercase
! Sends: aa:bb:cc:dd:ee:ff (default is aabbccddeeff)

! MAB-only port (printers, cameras, IoT)
interface GigabitEthernet1/0/2
  authentication port-control auto
  mab
  authentication order mab
  dot1x pae authenticator        ! still needed for EAPOL handling
```

### ISE MAB Policy

```
# ISE Profiling — identify device type by:
#   - MAC OUI (vendor prefix)
#   - DHCP fingerprint (options 55, 60)
#   - HTTP User-Agent
#   - CDP/LLDP attributes
#   - SNMP probe

# MAB authorization rule in ISE:
# IF: MAB AND Endpoint:BYODRegistration=Yes
# THEN: PermitAccess + VLAN 100
# ELSE: GuestRedirect + VLAN 999
```

## WebAuth Fallback

```
! Central Web Authentication (CWA) via ISE
interface GigabitEthernet1/0/1
  authentication port-control auto
  dot1x pae authenticator
  mab
  authentication order dot1x mab webauth
  authentication priority dot1x mab webauth
  ip admission name WEBAUTH_RULE

! Local Web Authentication (LWA)
ip admission name WEBAUTH_RULE proxy http
ip http server
ip http secure-server

! WebAuth flow:
# 1. Client connects, fails 802.1X and MAB
# 2. Switch assigns limited ACL (DNS + DHCP + redirect)
# 3. HTTP intercepted, redirected to ISE/local portal
# 4. User enters credentials on web portal
# 5. ISE sends CoA to authorize port with correct VLAN/ACL
```

## Deployment Modes

### Monitor Mode (Phase 1)

```
! All traffic allowed regardless of auth result — logging only
interface GigabitEthernet1/0/1
  authentication port-control auto
  authentication open               ! <-- key: allows all traffic
  dot1x pae authenticator
  mab
  authentication order dot1x mab

! Purpose:
#   - Baseline which devices authenticate successfully
#   - Identify devices needing MAB entries
#   - No disruption to production traffic
#   - Review logs: show authentication sessions
```

### Low-Impact Mode (Phase 2)

```
! Pre-auth ACL allows basic network access, post-auth gets full access
interface GigabitEthernet1/0/1
  authentication port-control auto
  authentication open
  dot1x pae authenticator
  mab
  authentication order dot1x mab
  ip access-group PRE-AUTH-ACL in    ! <-- pre-auth ACL limits access

! Pre-auth ACL example
ip access-list extended PRE-AUTH-ACL
  permit udp any any eq bootps       ! DHCP
  permit udp any any eq domain       ! DNS
  permit icmp any any                ! ping
  permit udp any host 10.1.1.100 eq 1812  ! RADIUS (if needed)
  deny   ip any any                  ! block everything else

! Post-auth: RADIUS returns dACL or removes pre-auth ACL
! Purpose:
#   - Gradual enforcement
#   - Basic services always available
#   - Full access only after authentication
```

### Closed Mode (Phase 3)

```
! Strict enforcement — no traffic until authenticated
interface GigabitEthernet1/0/1
  authentication port-control auto   ! no "authentication open"
  dot1x pae authenticator
  mab
  authentication order dot1x mab

! Closed mode behavior:
#   - Port blocks all traffic until Access-Accept
#   - Only EAPOL frames allowed pre-auth
#   - Failed auth = no network access (unless auth-fail VLAN)
#   - Most secure, but highest risk of disruption
```

## Change of Authorization (CoA / RFC 5176)

### Overview

```
# CoA allows RADIUS server to push policy changes mid-session
# Port: UDP 3799 (default)
# Messages:
#   CoA-Request  — change session attributes (VLAN, ACL)
#   CoA-ACK      — change applied successfully
#   CoA-NAK      — change failed
#   Disconnect-Request — terminate session
#   Disconnect-ACK     — session terminated

# Common triggers:
#   - Posture assessment complete (compliant/non-compliant)
#   - VLAN change after guest registration
#   - Quarantine after threat detection
#   - Admin-initiated session termination
```

### IOS-XE CoA Configuration

```
! CoA listener
aaa server radius dynamic-author
  client 10.1.1.100 server-key Cisco123
  auth-type any

! Verify CoA operations
show aaa server radius dynamic-author
debug radius dynamic-authorization
```

## Wireless 802.1X

### WLC Configuration (IOS-XE / C9800)

```
! WLAN with 802.1X
wlan CORPORATE 1 CORPORATE-SSID
  security dot1x authentication-list default
  security wpa wpa2
  security wpa wpa2 ciphers aes
  security wpa pairwise-cipher aes
  no shutdown

! Policy profile
wireless profile policy CORP-POLICY
  vlan 100
  aaa-override                    ! allow RADIUS to override VLAN
  nac                             ! enable posture (ISE)
  no shutdown

! Tag assignment
wireless tag policy SITE-A
  wlan CORPORATE policy CORP-POLICY
```

### Wireless EAP Timers

```
! Adjust for wireless latency
dot1x timeout tx-period 10
dot1x timeout supp-timeout 30
dot1x max-reauth-req 2
authentication timer reauthenticate 3600
```

## ISE Integration

### Network Device Setup

```
# ISE: Administration > Network Resources > Network Devices
#   Name: SW-ACCESS-01
#   IP: 10.1.1.10
#   RADIUS Shared Secret: RadiusSharedSecret
#   CoA Port: 1799
#   SNMP: v2c/v3 for profiling
#   Device Type: Cisco Switches
#   Location: Building-A

# ISE: Policy > Authentication
#   Rule: Wired-Dot1X
#   Condition: Wired_802.1X
#   Allowed Protocols: Default Network Access
#   Identity Source: Active Directory
```

### Authorization Policy

```
# ISE: Policy > Authorization
# Rule order matters — first match wins

# Rule 1: Corporate Full Access
#   Condition: AD:MemberOf = Domain Computers AND Compliant
#   Result: PermitAccess, VLAN=100, dACL=FULL-ACCESS

# Rule 2: BYOD Limited
#   Condition: BYOD_Registered = Yes
#   Result: PermitAccess, VLAN=200, dACL=LIMITED-ACCESS

# Rule 3: Non-Compliant
#   Condition: Posture = NonCompliant
#   Result: Quarantine, VLAN=998, dACL=REMEDIATION

# Rule 4: Guest
#   Condition: Guest_Flow = Yes
#   Result: GuestAccess, VLAN=999

# Default: DenyAccess
```

## Troubleshooting

### Show Commands

```
! Session status
show authentication sessions
show authentication sessions interface Gi1/0/1
show authentication sessions interface Gi1/0/1 details

! 802.1X status
show dot1x all
show dot1x interface Gi1/0/1 details
show dot1x statistics interface Gi1/0/1

! RADIUS
show aaa servers
show radius server-group all
show radius statistics
test aaa group ISE-SERVERS user testuser password testpass new-code

! MAB
show mab all
show mab interface Gi1/0/1

! Logs and events
show authentication history
show authentication sessions method dot1x
show authentication sessions method mab
```

### Debug Commands

```
! Use with caution in production
debug dot1x all
debug radius authentication
debug radius accounting
debug authentication all
debug mab all
debug epm all                    ! Endpoint Policy Manager

! Conditional debugging (safer)
debug condition interface Gi1/0/1
debug dot1x all
! Remember to remove:
no debug condition all
undebug all
```

### Common Issues

```
# RADIUS timeout — check connectivity, shared secret, source interface
# EAP timeout — tx-period too short, supplicant misconfigured
# VLAN not assigned — VLAN must exist on switch, check RADIUS attributes
# MAB failing — check MAC format (separator, case), verify in ISE
# CoA not working — verify CoA client IP/key in dynamic-author config
# Auth loop — check reauth timer, EAP retries, server dead detection
# IP phone issues — verify multi-domain mode, voice VLAN, CDP/LLDP
# Critical VLAN not activating — check server dead detection method
```

## Tips

- Deploy in phases: monitor mode first, then low-impact, then closed mode
- Always configure a critical VLAN for RADIUS server outages
- Use `authentication open` during initial rollout to prevent lockouts
- Set `dot1x timeout tx-period 10` and `max-reauth-req 2` to reduce auth delay for non-802.1X devices falling back to MAB
- Enable RADIUS server load balancing with `load-balance method least-outstanding`
- Configure RADIUS automate-tester to detect server recovery faster
- Use device tracking (`ip device tracking`) for CoA and IPSG compatibility
- Enable `authentication event server dead action authorize vlan` on every port
- Match the MAC format in MAB (separator, grouping, case) between switch and ISE
- For IP phones, always use multi-domain mode with a voice VLAN
- Configure `spanning-tree portfast` and `bpduguard` on all access ports with 802.1X
- Test RADIUS connectivity with `test aaa group` before enabling enforcement

## See Also

- RADIUS
- TLS
- PKI
- SSH
- Network Defense
- Zero Trust
- CoPP

## References

- IEEE 802.1X-2020 — Port-Based Network Access Control
- RFC 3748 — Extensible Authentication Protocol (EAP)
- RFC 5216 — EAP-TLS Authentication Protocol
- RFC 5281 — EAP-TTLS
- RFC 7170 — EAP-FAST (Tunnel EAP)
- PEAP — draft-josefsson-pppext-eap-tls-eap (Microsoft/Cisco/RSA)
- RFC 2865 — Remote Authentication Dial-In User Service (RADIUS)
- RFC 5176 — Dynamic Authorization Extensions to RADIUS (CoA)
- Cisco IOS-XE Identity-Based Networking Services Configuration Guide
- Cisco ISE Admin Guide — https://www.cisco.com/c/en/us/support/security/identity-services-engine/series.html
- FreeRADIUS Documentation — https://freeradius.org/documentation/
- wpa_supplicant — https://w1.fi/wpa_supplicant/
