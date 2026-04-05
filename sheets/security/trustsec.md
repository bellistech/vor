# Cisco TrustSec (SGT-Based Network Segmentation)

> Software-defined segmentation using Security Group Tags (SGTs) to classify and enforce policy based on identity/role rather than IP addresses, integrated with ISE for centralized policy.

## Architecture

### TrustSec Components

```
                  ISE (Policy Server)
                  ┌─────────────────┐
                  │ SGT Assignment   │
                  │ SGACL Policy     │
                  │ Environment Data │
                  └────────┬────────┘
                           │ RADIUS / CoA / PAC
            ┌──────────────┼──────────────┐
            │              │              │
     ┌──────▼──────┐ ┌────▼─────┐ ┌─────▼─────┐
     │ Ingress     │ │ Transit  │ │ Egress    │
     │ Device      │ │ Device   │ │ Device    │
     │ (SGT assign)│ │ (propagate)│ │ (enforce) │
     └─────────────┘ └──────────┘ └───────────┘
```

### Key Terminology

| Term | Description |
|:-----|:------------|
| SGT | Security Group Tag — 16-bit value (0-65535) assigned to traffic |
| SGT assignment | Classification of endpoint into a security group |
| SGACL | Security Group ACL — policy enforced based on src/dst SGT pair |
| SXP | SGT Exchange Protocol — propagates IP-SGT bindings over TCP |
| CMD | Cisco Meta Data — inline SGT in Ethernet frame (EtherType 0x8909) |
| PAC | Protected Access Credential — used for TrustSec device auth |
| Environment Data | SGACL policy + SGT name table downloaded from ISE |
| IP-SGT Binding | Mapping of IP address to SGT value |

## SGT Assignment Methods

### Static SGT Assignment (Manual)

```
! Assign SGT to a specific IP address (IOS-XE)
cts role-based sgt-map 10.10.10.0/24 sgt 100

! Assign SGT to a VLAN (VLAN-SGT mapping)
cts role-based sgt-map vlan-list 100 sgt 10

! Assign SGT to an interface (port-level)
interface GigabitEthernet1/0/1
  cts manual
  policy static sgt 15
```

### Dynamic SGT via 802.1X / RADIUS

```
! ISE returns SGT as a RADIUS attribute in Access-Accept
! cisco-av-pair = "cts:security-group-tag=0064-00"   (SGT 100, hex)

! Switch config — enable TrustSec + 802.1X on port
interface GigabitEthernet1/0/1
  switchport mode access
  authentication port-control auto
  dot1x pae authenticator
  cts dot1x
```

### MAB with SGT

```
! ISE authorization policy assigns SGT for MAB-authenticated devices
! Same RADIUS attribute returned: cts:security-group-tag=XXXX-XX

interface GigabitEthernet1/0/5
  switchport mode access
  authentication port-control auto
  mab
  cts dot1x
```

### VLAN-SGT Mapping (Layer 3 Boundary)

```
! Used at L3 interfaces where inline tagging is not available
! Maps all traffic from a VLAN to a specific SGT

cts role-based sgt-map vlan-list 10 sgt 5
cts role-based sgt-map vlan-list 20 sgt 10
cts role-based sgt-map vlan-list 30 sgt 15
```

### Subnet-SGT Mapping

```
! Map IP subnets to SGTs (useful for non-TrustSec capable segments)
cts role-based sgt-map 192.168.1.0/24 sgt 100
cts role-based sgt-map 10.0.0.0/8 sgt 200
```

## SGT Propagation

### Inline Tagging (CMD — Cisco Meta Data)

```
! Preferred method — SGT embedded directly in Ethernet frame
! Requires hardware support on both ends of the link

! Enable inline tagging on a trunk link
interface TenGigabitEthernet1/0/1
  switchport mode trunk
  cts manual
  policy static sgt 2 trusted
  propagate sgt

! Verify inline tagging
show cts interface GigabitEthernet1/0/1
  CTS Information for Interface GigabitEthernet1/0/1:
    Propagate SGT:    Enabled
    Tag Mode:         Inline
```

### CMD Ethernet Frame Format

```
+----------+---------+-----------+----------+--------+----------+
| Dst MAC  | Src MAC | CMD EType | CMD HDR  | VLAN   | Payload  |
| (6B)     | (6B)    | 0x8909   | SGT+Ver  | (opt)  |          |
+----------+---------+-----------+----------+--------+----------+

CMD Header (16 bytes):
  Version (1B) | Length (1B) | Option Type (2B) | SGT Value (2B) | ...
```

### SXP (SGT Exchange Protocol)

```
! TCP-based protocol (port 64999) for propagating IP-SGT bindings
! Used when inline tagging is not supported on all devices

! SXP Speaker (sends IP-SGT bindings)
cts sxp enable
cts sxp default source-ip 10.1.1.1
cts sxp default password TRUSTSEC_KEY
cts sxp connection peer 10.2.2.2 password default mode local speaker hold-time 120 120

! SXP Listener (receives IP-SGT bindings)
cts sxp enable
cts sxp default source-ip 10.2.2.2
cts sxp default password TRUSTSEC_KEY
cts sxp connection peer 10.1.1.1 password default mode local listener hold-time 120 120

! SXP on NX-OS
cts sxp enable
cts sxp node-id interface loopback0
cts sxp connection peer 10.1.1.1 password required TRUSTSEC_KEY mode listener
```

### SXP Topology

```
# Speaker → Listener direction (unidirectional by default)
# SXPv4 supports bidirectional mode

Access Switch (Speaker)  ---SXP--->  Distribution/FW (Listener)
  IP-SGT: 10.1.1.5 → SGT 10         Learns bindings
  IP-SGT: 10.1.1.8 → SGT 20         Applies SGACL enforcement

# SXP loop prevention: peer-sequence attribute tracks path
```

## SGACL (Security Group ACL)

### SGACL Configuration on ISE

```
! SGACLs are defined on ISE and pushed to enforcement devices
! Policy matrix: Source SGT (rows) × Destination SGT (columns)

! Example SGACL policy (ISE TrustSec policy matrix):
!   Source: Employees (SGT 5) → Destination: Servers (SGT 100)
!   SGACL: permit tcp dst eq 443; permit tcp dst eq 80; deny ip

! Example SGACL policy:
!   Source: Guests (SGT 15) → Destination: Servers (SGT 100)
!   SGACL: deny ip
```

### Local SGACL (Fallback / Override)

```
! Define SGACL locally on enforcement device (IOS-XE)
ip access-list role-based PERMIT_WEB
  permit tcp dst eq 80
  permit tcp dst eq 443
  deny ip

! Map SGACL to source-destination SGT pair
cts role-based permissions from 5 to 100 PERMIT_WEB

! Default permission (unknown SGT pairs)
cts role-based permissions default DENY_ALL
```

### Enable SGACL Enforcement

```
! Global enforcement (IOS-XE)
cts role-based enforcement
cts role-based enforcement vlan-list all

! Per-VLAN enforcement
cts role-based enforcement vlan-list 10,20,30

! NX-OS enforcement
cts role-based enforcement
cts role-based enforcement vlan 10-30
```

## ISE TrustSec Configuration

### Device Registration

```
! ISE: Administration > Network Resources > Network Devices
!   Add device with TrustSec settings:
!   - Device ID (hostname)
!   - TrustSec notification and updates: Use PAC / password
!   - Include device in SGA type (TrustSec)

! ISE: Work Centers > TrustSec > Components > Security Groups
!   Create SGTs:
!     Name: Employees    SGT: 5
!     Name: Servers      SGT: 100
!     Name: Guests       SGT: 15
!     Name: IOT_Devices  SGT: 50
!     Name: Unknown      SGT: 0 (reserved)
```

### Authorization Policy (SGT Assignment)

```
! ISE: Policy > Authorization
! Rule: IF identity-group = Employees AND posture = Compliant
!         THEN assign SGT = Employees (5)
!
! Rule: IF endpoint-group = IOT
!         THEN assign SGT = IOT_Devices (50)
```

### Policy Matrix (SGACL Mapping)

```
! ISE: Work Centers > TrustSec > TrustSec Policy > Egress Policy > Matrix

! Source SGT     Dest SGT     SGACL
! Employees(5)  Servers(100) Permit_Web (permit 80,443)
! Guests(15)    Servers(100) Deny_All
! IOT(50)       Servers(100) Permit_MQTT (permit 8883)
! Unknown(0)    Any          Deny_All
```

## NX-OS TrustSec Configuration

### Basic TrustSec Setup

```
! Enable TrustSec on Nexus
feature cts
feature dot1x

! CTS credentials
cts device-id NEXUS-9K password 0 TrustSecPAC

! AAA for TrustSec
radius-server host 10.10.10.50 key RADIUS_KEY pac
aaa group server radius ISE_GROUP
  server 10.10.10.50
aaa authentication cts default group ISE_GROUP
aaa authorization cts default group ISE_GROUP

! Download environment data from ISE
cts refresh environment-data
cts refresh policy
```

### NX-OS SGT Assignment and Enforcement

```
! Static IP-SGT on NX-OS
cts role-based sgt-map 10.10.0.0/16 sgt 100

! VLAN-SGT on NX-OS
cts role-based sgt-map vlan 100 sgt 10

! Enable enforcement on NX-OS
cts role-based enforcement
cts role-based enforcement vlan 10-50

! Local SGACL on NX-OS
ip access-list SGACL_PERMIT_WEB role-based
  permit tcp dst eq 443
  permit tcp dst eq 80
  deny ip
cts role-based permissions from 5 to 100 SGACL_PERMIT_WEB
```

## TrustSec with VXLAN (SGT in GPO)

```
! VXLAN Group Policy Option (GPO) carries SGT in VXLAN header
! Used in SD-Access (Cisco DNA Center / Catalyst Center)

! VXLAN GPO header extension:
!   Standard VXLAN header (8 bytes)
!   + Group Policy ID field (16 bits) = SGT value
!   Flag: G-bit set indicates GPO is present

! NX-OS VXLAN with SGT (fabric edge)
interface nve1
  member vni 50001
    ingress-replication protocol bgp
    suppress-arp

! SD-Access: SGT is assigned at fabric edge, carried in VXLAN-GPO
! across fabric, enforced at egress fabric edge
```

## TrustSec with SD-Access

```
! SD-Access uses TrustSec SGTs as the segmentation mechanism
! Catalyst Center (DNA Center) automates SGT assignment and policy

! Flow:
! 1. Endpoint authenticates → ISE assigns SGT
! 2. Fabric edge encapsulates in VXLAN with SGT in GPO
! 3. Traffic traverses fabric with SGT intact
! 4. Egress fabric edge enforces SGACL based on src/dst SGT

! Catalyst Center: Policy > Group-Based Access Control
!   - Create scalable groups (mapped to SGTs)
!   - Define access contracts (mapped to SGACLs)
!   - Policy matrix deployed via ISE to fabric devices

! Macro-segmentation: VNs (Virtual Networks) = VRFs
! Micro-segmentation: SGTs within a VN
```

## IP-SGT Binding Table

```
! View IP-SGT bindings (IOS-XE)
show cts role-based sgt-map all
  Active IPv4-SGT Bindings Information
  IP Address        SGT     Source
  10.1.1.5          5       INTERNAL (learned via 802.1X)
  10.1.1.10         15      SXP (learned from SXP peer)
  10.10.0.0/16      100     CLI (configured manually)

! View IP-SGT bindings (NX-OS)
show cts role-based sgt-map
```

## Troubleshooting

### Verify TrustSec Status

```
! Check TrustSec global status
show cts

! Check PAC (Protected Access Credential)
show cts pac
  PAC-Info:
    PAC-type: Cisco TrustSec
    AID:  a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4
    I-ID: SWITCH-01
    A-ID-Info: Identity Services Engine
    PAC-Opaque: ...
    PAC Lifetime: ... (Valid)
    PAC Refresh: ...

! Verify environment data download
show cts environment-data
  CTS Environment Data
  ====================
  Current state = COMPLETE
  Last status = Successful
  Server List:  10.10.10.50 (Installed)
  SGT Table:
    SGT 0  : Unknown
    SGT 5  : Employees
    SGT 15 : Guests
    SGT 100: Servers
```

### Verify SGT Assignment

```
! Check SGT on interfaces
show cts interface brief
  Interface    Mode     IFC-state  dot1x-role  peer-id    SGT
  Gi1/0/1      MANUAL   OPEN       unknown     unknown    [5]
  Gi1/0/5      DOT1X    OPEN       both        SWITCH-02  [10]

! Check SGT on specific interface
show cts interface GigabitEthernet1/0/1

! Check role-based permissions (SGACL)
show cts role-based permissions
  IPv4 Role-based permissions:
    From  To    Policy
    5     100   PERMIT_WEB-05
    15    100   DENY_ALL-01
```

### Verify SXP

```
! Check SXP connections
show cts sxp connections brief
  Peer IP          Source IP        Conn Status  Duration    Mode
  10.2.2.2         10.1.1.1         On           2:15:30:00  Speaker
  10.3.3.3         10.1.1.1         On           1:05:12:00  Listener

! Check SXP bindings
show cts sxp sgt-map brief
  IP Address       SGT   Peer IP       Ins Num  Status
  10.1.1.5         5     10.2.2.2      1        Active
  10.1.1.8         20    10.2.2.2      1        Active
```

### Verify SGACL Enforcement

```
! Check role-based counters
show cts role-based counters
  From   To     SW-Denied  HW-Denied  SW-Permit  HW-Permit
  5      100    0          0          15234      98432
  15     100    345        2100       0          0

! Clear counters for troubleshooting
clear cts role-based counters

! Check SGACL policy
show cts role-based permissions from 5 to 100
  IPv4 Role-based permissions from group 5:Employees to group 100:Servers:
    PERMIT_WEB-05
      permit tcp dst eq 80
      permit tcp dst eq 443
      deny ip
```

### Debug Commands

```
! Enable TrustSec debugging (use with caution)
debug cts all
debug cts authorization
debug cts sxp message
debug cts environment-data

! Clear and refresh
cts refresh environment-data
cts refresh policy

! Force re-authentication to get new SGT
clear authentication sessions interface Gi1/0/1
```

### Common Issues

```
# PAC provisioning failure
#   - Verify device credentials on ISE match switch config
#   - Check RADIUS reachability: test aaa group ISE_GROUP admin Cisco123 legacy
#   - Ensure ISE TrustSec settings have correct device ID

# SGT not assigned via RADIUS
#   - Verify ISE authorization policy returns cts:security-group-tag
#   - Check: show authentication sessions interface Gi1/0/1 detail
#   - Look for "SGT Value" in session output

# SGACL not enforcing
#   - Ensure "cts role-based enforcement" is enabled globally
#   - Verify SGACL downloaded: show cts role-based permissions
#   - Check TCAM: show platform hardware fed switch active sgacl

# SXP connection down
#   - Verify TCP 64999 reachable between peers
#   - Check password match on both ends
#   - Verify source-ip matches reachable address
#   - show cts sxp connections — check "Conn Status"
```

## See Also

- dot1x
- cisco-ise
- macsec
- acl
- zero-trust
- vxlan

## References

- Cisco TrustSec Configuration Guide (IOS-XE)
- Cisco TrustSec Configuration Guide (NX-OS)
- Cisco ISE TrustSec Administration Guide
- RFC 7343 — SGT Exchange Protocol (SXP)
- Cisco SD-Access Solution Design Guide
