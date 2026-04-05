# Cisco ISE (Identity Services Engine)

Centralized policy platform for network access control, providing RADIUS/TACACS+ authentication, profiling, posture assessment, guest access, BYOD, and TrustSec (SGT) policy enforcement across wired, wireless, and VPN.

## Concepts

### ISE Deployment Models

| Model | Description |
|-------|-------------|
| Standalone | Single ISE node (all personas, lab/small) |
| Distributed | Multiple ISE nodes with different personas |
| Two-node HA | PAN primary + PAN secondary (automatic failover) |
| Full distributed | Multiple PSNs behind load balancer, PAN HA, MnT nodes |

### Node Personas

| Persona | Role | Count |
|---------|------|-------|
| PAN (Policy Administration Node) | Configuration, policy management UI | 2 (primary + secondary) |
| MnT (Monitoring and Troubleshooting) | Logging, reporting, alerting | 1-2 |
| PSN (Policy Service Node) | RADIUS/TACACS+ processing, profiling, posture | 2+ (behind load balancer) |
| pxGrid | Context sharing with ecosystem partners | Runs on PSN or dedicated |

- A single node can run multiple personas (standalone mode)
- Maximum deployment: 50 nodes (ISE 3.x)

## RADIUS Authentication Flows

### 802.1X (Wired/Wireless)

```
Supplicant          Authenticator (Switch/WLC)          ISE (PSN)
    |                        |                              |
    |<-- EAP-Request/ID -----|                              |
    |--- EAP-Response/ID --->|                              |
    |                        |--- RADIUS Access-Request --->|
    |                        |    (EAP-Message attribute)   |
    |                        |                              |
    |                        |<-- RADIUS Access-Challenge --|
    |<-- EAP-Request --------|    (EAP method negotiation) |
    |--- EAP-Response ------>|                              |
    |                        |--- RADIUS Access-Request --->|
    |                        |                              |
    |          ... (EAP method exchange) ...                |
    |                        |                              |
    |                        |<-- RADIUS Access-Accept -----|
    |<-- EAP-Success --------|    + VLAN, SGT, dACL, etc.  |
    |                        |                              |
    | [Port authorized]      | [Apply policy]              |
```

### MAB (MAC Authentication Bypass)

```
Endpoint            Switch                    ISE (PSN)
    |                  |                          |
    | [No 802.1X]      |                          |
    | [MAB timeout]    |                          |
    |-- Any frame ---->|                          |
    |                  |--- Access-Request ------->|
    |                  |    (MAC as username)      |
    |                  |                          |
    |                  |<-- Access-Accept ---------|
    |                  |    (or Reject)            |
    |                  |    + VLAN, SGT, dACL     |
```

### Common EAP Methods

| Method | Auth Type | Credential | Certificate Needed |
|--------|-----------|------------|-------------------|
| EAP-TLS | Mutual TLS | Client + server cert | Both |
| PEAP (MSCHAPv2) | TLS tunnel + password | Server cert + AD password | Server only |
| EAP-FAST | TLS tunnel + PAC | Server cert + PAC or password | Server only |
| EAP-TEAP | TLS tunnel (RFC 7170) | Cert + password (chained) | Server + optional client |

## Policy Sets

### Structure

```
Policy Set: "Wired_Access"
  ├── Conditions: RADIUS:NAS-Port-Type == Ethernet
  │
  ├── Authentication Policy:
  │   ├── Rule 1: If Dot1X → use AD identity source
  │   ├── Rule 2: If MAB → use Internal Endpoints DB
  │   └── Default: DenyAccess
  │
  └── Authorization Policy:
      ├── Rule 1: If AD-Group=IT-Admins AND Posture=Compliant → PermitAccess + VLAN 10 + SGT=IT
      ├── Rule 2: If AD-Group=Employees AND Posture=Compliant → PermitAccess + VLAN 20 + SGT=Employee
      ├── Rule 3: If AD-Group=Employees AND Posture=NonCompliant → Quarantine_VLAN + SGT=Quarantine
      ├── Rule 4: If EndpointProfile=IP-Phone → PermitAccess + Voice_VLAN + SGT=Voice
      └── Default: DenyAccess
```

### Authorization Profiles

Authorization profiles define what happens when a rule matches:

| Attribute | Description |
|-----------|-------------|
| VLAN | Assign endpoint to a specific VLAN |
| dACL (Downloadable ACL) | Push ACL to the switch for the port |
| SGT (Security Group Tag) | Assign TrustSec tag |
| URL Redirect | Redirect HTTP to ISE portal (guest, posture) |
| Voice Domain Permission | Allow phone onto voice VLAN |
| Auto Smart Port | Apply Cisco smart port macro |
| RADIUS Attributes | Any standard or vendor-specific attribute |

## Identity Sources

### Supported Sources

| Source | Protocol | Use Case |
|--------|----------|----------|
| Active Directory | LDAP/Kerberos | Primary enterprise identity |
| LDAP | LDAP | Generic directory services |
| Internal Users | Local DB | Fallback, guest sponsors |
| Internal Endpoints | Local DB | MAC addresses for MAB |
| Certificate Auth | EAP-TLS | Certificate-based auth |
| SAML | SAML 2.0 | Federated identity (guest portals) |
| ODBC | SQL | External database authentication |
| RSA SecurID | RADIUS proxy | MFA integration |
| Social Login | OAuth | Guest self-registration |

### Identity Source Sequence

```
! Example: Try AD first, then LDAP, then internal
Identity Source Sequence: CORP_SEQUENCE
  1. Active Directory (corp.example.com)
     - If user not found: Continue to next
     - If auth failed: Reject
     - If server unreachable: Continue to next
  2. LDAP (ldap.example.com)
     - If user not found: Continue to next
  3. Internal Users
     - If user not found: Reject
```

## Profiling

### Profiling Probes

| Probe | Data Collected | Method |
|-------|---------------|--------|
| RADIUS | RADIUS attributes (NAS-Port-Type, Calling-Station-Id) | Passive |
| DHCP | DHCP options (hostname, vendor class, fingerprint) | Passive (span/helper) |
| HTTP | User-Agent string, HTTP headers | Passive (span) |
| SNMP | CDP/LLDP neighbors, device info | Active (queries switch) |
| NetFlow | Traffic patterns, port usage | Passive |
| DNS | Reverse DNS lookup | Active |
| NMAP | OS fingerprint, open ports | Active (intrusive) |
| Active Directory | AD machine attributes | Active (WMI) |

### Profiling Configuration on Switch

```
! Send DHCP info to ISE via RADIUS accounting
interface GigabitEthernet1/0/1
 ip dhcp snooping
 ip helper-address 10.1.1.50

! Enable device sensor for CDP/LLDP/DHCP
device-sensor filter-list dhcp list DHCP-LIST
 option name host-name
 option name class-identifier
 option name client-identifier
device-sensor filter-list lldp list LLDP-LIST
 tlv name system-name
 tlv name system-description
device-sensor filter-list cdp list CDP-LIST
 tlv name device-name
 tlv name platform-type

device-sensor notify all-changes

! SNMP for ISE profiling queries
snmp-server community ISE-PROFILE ro
snmp-server host 10.1.1.50 version 2c ISE-PROFILE
```

### Endpoint Profile Matching

ISE uses a certainty factor system:

$$Certainty = \sum_{i=1}^{n} Weight_i \times Match_i$$

Where $Match_i$ is 1 if the condition matches, 0 otherwise. An endpoint is assigned the profile with the highest certainty that exceeds the minimum certainty factor (default 10).

Example:
| Condition | Weight | Match |
|-----------|--------|-------|
| DHCP class-id contains "Cisco" | 10 | Yes |
| CDP platform = "IP Phone" | 20 | Yes |
| OUI = Cisco (MAC prefix) | 10 | Yes |
| **Total certainty** | **40** | **Cisco-IP-Phone** |

## Posture Assessment

### Posture Flow

```
Endpoint                Switch              ISE                Posture Agent
    |                      |                  |                      |
    |--- 802.1X auth ----->|--- RADIUS ------>|                      |
    |                      |<- Accept + URL ->|                      |
    |                      |   redirect       |                      |
    |                      |                  |                      |
    |--- HTTP request ---->|--- Redirect ---->|                      |
    |<-- Posture agent download / redirect ---|                      |
    |                      |                  |                      |
    |                      |                  |<-- Posture check ----|
    |                      |                  |    (AV version,      |
    |                      |                  |     patches, etc.)   |
    |                      |                  |                      |
    |                      |                  |--- Compliant ------->|
    |                      |<-- CoA (new authz)--|                   |
    |                      |   (full access)  |                      |
```

### Posture Conditions

| Type | Checks |
|------|--------|
| Anti-Virus | AV installed, definition date, engine version |
| Anti-Malware | AM product, version, definition date |
| Patch Management | OS patches, specific KB articles |
| Disk Encryption | Full-disk encryption enabled |
| USB | USB storage blocked |
| Firewall | Host firewall enabled |
| Application | Required/prohibited applications |
| File | Specific file presence/absence |
| Registry | Registry key values (Windows) |
| Service | Required Windows/Linux services running |

## Guest Access

### Guest Portal Types

| Portal | Purpose |
|--------|---------|
| Hotspot | Click-through acceptance (no credentials) |
| Self-Registered | Guest creates own account |
| Sponsored | Employee sponsor creates guest account |
| Credentialed | Pre-created guest accounts |

### Guest Flow

```
Guest                    Switch/WLC              ISE
  |                          |                     |
  |-- Associate/connect ---->|                     |
  |                          |--- MAB/WebAuth ----->|
  |                          |<-- Redirect to -----|
  |                          |    guest portal     |
  |<--- HTTP redirect ------|                     |
  |--- Guest portal -------->|                     |
  |    (login/register)      |                     |
  |                          |                     |
  |                          |<-- CoA + authz -----|
  |                          |    (guest VLAN/ACL) |
  | [Internet access]       |                     |
```

## BYOD (Bring Your Own Device)

### BYOD Flow

1. User connects with personal device, gets redirected to BYOD portal
2. User authenticates with corporate credentials
3. ISE pushes certificate and/or Wi-Fi profile via SCEP/native supplicant provisioning
4. Device re-authenticates with certificate (EAP-TLS)
5. ISE applies BYOD-specific authorization policy (limited access)

### My Devices Portal

- End users can register/deregister personal devices
- View registered devices and their status
- Self-service: mark device as lost/stolen (blacklist MAC)

## TrustSec

### SGT Assignment

```
! ISE assigns SGT via RADIUS (cisco-av-pair)
! Example RADIUS attribute returned:
!   cisco-av-pair = "cts:security-group-tag=0005-00"
!   (SGT 5, assigned to the endpoint session)
```

### SGT Assignment Methods

| Method | Description |
|--------|-------------|
| Dynamic (RADIUS) | ISE assigns SGT during authentication |
| Static (switch CLI) | Manually assign SGT to VLAN or port |
| SXP (SGT Exchange Protocol) | Propagate IP-to-SGT bindings over TCP |
| Inline tagging (CMD) | SGT embedded in Ethernet frame (802.1AE MACsec header) |

### SXP (SGT Exchange Protocol)

```
! Switch: SXP to propagate IP-SGT bindings
cts sxp enable
cts sxp default source-ip 10.1.1.1
cts sxp default password cisco123
cts sxp connection peer 10.1.1.2 password default mode local speaker

! Show SXP connections
show cts sxp connections
show cts sxp sgt-map
```

### SGACL (Security Group ACL)

```
! ISE defines SGACL policies:
! Source SGT: Employees (5)
! Destination SGT: Servers (10)
! SGACL: Permit HTTP, HTTPS, DNS; Deny all else

! Switch: download and apply SGACLs
cts role-based enforcement
cts role-based enforcement vlan-list all

! Show SGACL policy
show cts role-based permissions
show cts role-based counters
```

## pxGrid

### Architecture

- Publish/subscribe messaging framework for context sharing
- Partners can subscribe to session data, TrustSec topology, profiling data
- WebSocket-based (pxGrid 2.0) or XMPP-based (pxGrid 1.0)
- Common integrations: Cisco FMC, Stealthwatch, DNA Center, third-party SIEM

### pxGrid Topics

| Topic | Data |
|-------|------|
| Session | User-IP-MAC-SGT bindings |
| TrustSec | SGT-name mappings, SGACL policies |
| Profiling | Endpoint profile data |
| MDM | Mobile device compliance status |
| Adaptive Network Control | ANC actions (quarantine, shutdown) |
| Threat | IOC (Indicators of Compromise) |

## ISE REST API (ERS)

### API Basics

```bash
# Base URL
# https://<ISE-PAN>:9060/ers/config/

# List internal users
curl -k -X GET \
  -H "Accept: application/json" \
  -H "Content-Type: application/json" \
  -u admin:password \
  "https://ise.example.com:9060/ers/config/internaluser"

# Get specific endpoint by MAC
curl -k -X GET \
  -H "Accept: application/json" \
  -u admin:password \
  "https://ise.example.com:9060/ers/config/endpoint?filter=mac.EQ.AA:BB:CC:DD:EE:FF"

# Create a guest user
curl -k -X POST \
  -H "Accept: application/json" \
  -H "Content-Type: application/json" \
  -u sponsor:password \
  "https://ise.example.com:9060/ers/config/guestuser" \
  -d '{
    "GuestUser": {
      "guestType": "Daily (default)",
      "portalId": "portal-id-here",
      "guestInfo": {
        "userName": "guest1",
        "password": "TempPass123"
      },
      "guestAccessInfo": {
        "validDays": 1,
        "fromDate": "04/05/2026 08:00",
        "toDate": "04/05/2026 18:00"
      }
    }
  }'

# List network devices
curl -k -X GET \
  -H "Accept: application/json" \
  -u admin:password \
  "https://ise.example.com:9060/ers/config/networkdevice"

# ANC (Adaptive Network Control) — quarantine an endpoint
curl -k -X PUT \
  -H "Accept: application/json" \
  -H "Content-Type: application/json" \
  -u admin:password \
  "https://ise.example.com:9060/ers/config/ancendpoint/apply" \
  -d '{
    "OperationAdditionalData": {
      "additionalData": [
        {"name": "macAddress", "value": "AA:BB:CC:DD:EE:FF"},
        {"name": "policyName", "value": "Quarantine"}
      ]
    }
  }'
```

### Open API (ISE 3.1+)

```bash
# ISE 3.1+ also exposes OpenAPI on port 443
# https://<ISE>/api/v1/

# List endpoints
curl -k -X GET \
  -H "Accept: application/json" \
  -u admin:password \
  "https://ise.example.com/api/v1/endpoint"
```

## TACACS+ Device Administration

### TACACS+ Policy Set

```
Policy Set: "Network_Device_Admin"
  ├── Conditions: Device Type = Switches
  │
  ├── Authentication:
  │   └── Use AD for authentication
  │
  ├── Authorization:
  │   ├── Rule 1: AD-Group=NetAdmins → Shell Profile: privilege 15
  │   ├── Rule 2: AD-Group=NetOps → Shell Profile: privilege 7
  │   └── Default: DenyAccess
  │
  └── Command Authorization:
      ├── Rule 1: NetAdmins → Permit All
      ├── Rule 2: NetOps → Permit show *, Permit config term, Deny write *
      └── Default: Deny All
```

### Switch TACACS+ Configuration

```
! Point to ISE for TACACS+
tacacs server ISE-PRIMARY
 address ipv4 10.1.1.50
 key 0 TacacsSharedSecret
 timeout 5

aaa new-model
aaa authentication login default group tacacs+ local
aaa authorization exec default group tacacs+ local
aaa authorization commands 15 default group tacacs+ local
aaa accounting exec default start-stop group tacacs+
aaa accounting commands 15 default start-stop group tacacs+
```

## ISE Certificates

### Certificate Types

| Certificate | Purpose |
|-------------|---------|
| Admin | ISE admin portal HTTPS |
| EAP Authentication | Server certificate for EAP methods |
| pxGrid | pxGrid controller/client authentication |
| Portal | Guest/sponsor/BYOD portal HTTPS |
| SAML | SAML IdP certificate |
| System | Internal ISE communication |

### Certificate Chain

```
! For EAP-TLS or PEAP, the ISE server certificate must be trusted by endpoints:
! Root CA → Intermediate CA → ISE EAP Certificate

! For 802.1X client certificates (EAP-TLS):
! Root CA → Intermediate CA → Client Certificate
! ISE must trust the client's CA chain

! Import certificates via ISE GUI:
! Administration > System > Certificates
```

## Switch Configuration for ISE

### Full 802.1X + MAB + Guest

```
! AAA configuration
aaa new-model
aaa authentication dot1x default group radius
aaa authorization network default group radius
aaa accounting dot1x default start-stop group radius

! RADIUS server
radius server ISE-PSN1
 address ipv4 10.1.1.50 auth-port 1812 acct-port 1813
 key RadiusSecret123
 automate-tester username probe-user probe-on

radius server ISE-PSN2
 address ipv4 10.1.1.51 auth-port 1812 acct-port 1813
 key RadiusSecret123
 automate-tester username probe-user probe-on

aaa group server radius ISE
 server name ISE-PSN1
 server name ISE-PSN2
 deadtime 15

! Global 802.1X settings
dot1x system-auth-control

! Enable CoA (Change of Authorization)
aaa server radius dynamic-author
 client 10.1.1.50 server-key RadiusSecret123
 client 10.1.1.51 server-key RadiusSecret123

! Access port configuration
interface GigabitEthernet1/0/1
 description User Access Port
 switchport mode access
 switchport access vlan 20
 authentication host-mode multi-auth
 authentication order dot1x mab
 authentication priority dot1x mab
 authentication port-control auto
 authentication periodic
 authentication timer reauthenticate server
 mab
 dot1x pae authenticator
 dot1x timeout tx-period 10
 spanning-tree portfast
 spanning-tree bpduguard enable

! Enable RADIUS accounting for profiling
interface GigabitEthernet1/0/1
 ip device tracking
```

## Troubleshooting

### ISE GUI

```
! Operations > RADIUS > Live Logs
! - Real-time authentication/authorization results
! - Click detail icon for full RADIUS exchange

! Operations > RADIUS > Live Sessions
! - Currently active sessions
! - CoA actions (re-auth, port bounce, shutdown)

! Operations > Reports > Endpoints and Users
! - Historical authentication data
! - Failed authentication analysis
```

### ISE CLI

```
! Show ISE application status
show application status ise

! Show ISE node configuration
show running-config

! Test RADIUS connectivity
test aaa group ISE user@corp.example.com password123 new-code

! Debug RADIUS on switch
debug radius authentication
debug radius accounting
debug dot1x all
debug authentication all

! Show active 802.1X sessions on switch
show authentication sessions
show authentication sessions interface GigabitEthernet1/0/1 details
show dot1x all
```

## Tips

- Always deploy at least two PSNs behind a load balancer for RADIUS high availability.
- Use identity source sequences to handle AD unreachability gracefully (fallback to local or LDAP).
- Enable RADIUS accounting on all access devices; profiling depends on it for accuracy.
- Start with Monitor Mode (open authentication) before enforcing closed mode to identify policy gaps.
- Use the "Monitor" action in authorization rules to log without blocking during migration.
- RADIUS probe is always on and free; enable DHCP and HTTP probes next for best profiling accuracy.
- CoA requires the switch to have the ISE PSN configured as a dynamic-author client.
- TrustSec SXP is a fallback for devices that do not support inline SGT tagging; prefer inline tagging where possible.
- ISE EAP certificate must be trusted by all supplicants; deploy the CA to endpoints via GPO or MDM.
- Always test posture policy in audit mode before enforcing to avoid locking out legitimate users.

## See Also

- radius, cisco-ftd, ipsec, snmp

## References

- [Cisco ISE Administrator Guide](https://www.cisco.com/c/en/us/td/docs/security/ise/3-2/admin_guide/b_ise_admin_3_2.html)
- [Cisco ISE REST API (ERS) Guide](https://developer.cisco.com/docs/identity-services-engine/latest/)
- [Cisco TrustSec Design Guide](https://www.cisco.com/c/en/us/solutions/enterprise-networks/trustsec/design-guide-series.html)
- [Cisco ISE Prescriptive Deployment Guide](https://community.cisco.com/t5/security-knowledge-base/ise-secure-wired-access-prescriptive-deployment-guide/ta-p/3641515)
- [RFC 2865 — RADIUS](https://www.rfc-editor.org/rfc/rfc2865)
- [RFC 3579 — RADIUS Support for EAP](https://www.rfc-editor.org/rfc/rfc3579)
- [RFC 5176 — Dynamic Authorization Extensions to RADIUS (CoA)](https://www.rfc-editor.org/rfc/rfc5176)
- [RFC 8907 — TACACS+ Protocol](https://www.rfc-editor.org/rfc/rfc8907)
