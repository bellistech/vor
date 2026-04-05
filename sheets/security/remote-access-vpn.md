# Remote Access VPN (AnyConnect / SSL VPN / IKEv2 RA)

> Secure remote user connectivity using Cisco AnyConnect with SSL/TLS or IKEv2, clientless SSL VPN, split/full tunneling, group policies, DAP, posture, and MFA on ASA, FTD, and IOS-XE.

## AnyConnect SSL VPN — ASA Configuration

### Basic SSL VPN Setup

```
! Enable SSL on outside interface
webvpn
  enable outside
  anyconnect image disk0:/anyconnect-win-4.10.06079-webdeploy-k9.pkg 1
  anyconnect image disk0:/anyconnect-macos-4.10.06079-webdeploy-k9.pkg 2
  anyconnect enable
  tunnel-group-list enable         ! show group selector on login page

! IP address pool for VPN clients
ip local pool VPN_POOL 10.100.0.10-10.100.0.254 mask 255.255.255.0

! Group policy (defines session parameters)
group-policy GP_EMPLOYEES internal
group-policy GP_EMPLOYEES attributes
  vpn-tunnel-protocol ssl-client ssl-clientless
  split-tunnel-policy tunnelspecified
  split-tunnel-network-list value SPLIT_ACL
  dns-server value 10.1.1.53 10.1.2.53
  default-domain value example.com
  address-pools value VPN_POOL
  webvpn
    anyconnect keep-installer installed
    anyconnect modules value dart
    anyconnect ask none default anyconnect

! Split tunnel ACL (only listed networks go through VPN)
access-list SPLIT_ACL standard permit 10.0.0.0 255.0.0.0
access-list SPLIT_ACL standard permit 172.16.0.0 255.240.0.0
access-list SPLIT_ACL standard permit 192.168.0.0 255.255.0.0

! Tunnel group (connection profile)
tunnel-group EMPLOYEES type remote-access
tunnel-group EMPLOYEES general-attributes
  address-pool VPN_POOL
  default-group-policy GP_EMPLOYEES
  authentication-server-group ISE_RADIUS
  accounting-server-group ISE_RADIUS
tunnel-group EMPLOYEES webvpn-attributes
  group-alias "Employee VPN" enable
  group-url https://vpn.example.com/employees enable
```

### Full Tunnel Configuration

```
! Full tunnel — all traffic through VPN (no split tunnel)
group-policy GP_FULL_TUNNEL internal
group-policy GP_FULL_TUNNEL attributes
  vpn-tunnel-protocol ssl-client
  split-tunnel-policy tunnelall            ! ALL traffic through VPN
  dns-server value 10.1.1.53
  default-domain value example.com
  address-pools value VPN_POOL

! Full tunnel with split-DNS (DNS queries for specific domains via VPN)
group-policy GP_FULL_TUNNEL attributes
  split-dns value example.com internal.example.com
```

### VPN Filter (ACL Push)

```
! Apply ACL to VPN session (restrict what VPN users can access)
access-list VPN_FILTER extended permit tcp any 10.1.0.0 255.255.0.0 eq 443
access-list VPN_FILTER extended permit tcp any 10.1.0.0 255.255.0.0 eq 22
access-list VPN_FILTER extended permit icmp any 10.1.0.0 255.255.0.0
access-list VPN_FILTER extended deny ip any any

group-policy GP_EMPLOYEES attributes
  vpn-filter value VPN_FILTER

! Per-user ACL via RADIUS (ISE pushes ACL in Access-Accept)
! cisco-av-pair = "ip:inacl#1=permit tcp any 10.1.0.0 255.255.0.0 eq 443"
! Or use downloadable ACL (dACL) name:
! cisco-av-pair = "ACS:CiscoSecure-Defined-ACL=#ACSACL#-IP-DACL_NAME"
```

## AnyConnect IKEv2 VPN — ASA

### IKEv2 Remote Access Configuration

```
! Enable IKEv2 on outside interface
crypto ikev2 enable outside client-services port 443

! IKEv2 policy
crypto ikev2 policy 10
  encryption aes-256
  integrity sha256
  group 19 20 14
  prf sha256
  lifetime seconds 86400

! IPsec proposal
crypto ipsec ikev2 ipsec-proposal IKEv2_PROP
  protocol esp encryption aes-256-gcm
  protocol esp integrity null       ! GCM provides integrity

! Group policy for IKEv2
group-policy GP_IKEV2 internal
group-policy GP_IKEV2 attributes
  vpn-tunnel-protocol ikev2
  split-tunnel-policy tunnelspecified
  split-tunnel-network-list value SPLIT_ACL
  address-pools value VPN_POOL

! Tunnel group for IKEv2
tunnel-group IKEV2_RA type remote-access
tunnel-group IKEV2_RA general-attributes
  address-pool VPN_POOL
  default-group-policy GP_IKEV2
  authentication-server-group ISE_RADIUS
tunnel-group IKEV2_RA ipsec-attributes
  ikev2 remote-authentication eap
  ikev2 local-authentication certificate
```

## AnyConnect SSL VPN — IOS-XE

### IOS-XE SSL VPN Configuration

```
! Trustpoint for SSL certificate
crypto pki trustpoint VPN_CERT
  enrollment selfsigned
  subject-name CN=vpn.example.com
  rsakeypair VPN_RSA_KEY 2048
  revocation-check none

crypto pki enroll VPN_CERT

! AAA configuration
aaa new-model
aaa authentication login VPN_AUTH group radius local
aaa authorization network VPN_AUTHZ local

! SSL VPN gateway
webvpn gateway VPN_GW
  ip address 198.51.100.1 port 443
  ssl trustpoint VPN_CERT
  inservice

! WebVPN context
webvpn context VPN_CTX
  gateway VPN_GW
  inservice

! Virtual template for VPN sessions
interface Virtual-Template1
  ip unnumbered Loopback0
  ip mtu 1400

! SSL VPN policy
crypto vpn anyconnect profile VPN_PROFILE bootflash:vpn_profile.xml
```

## AnyConnect SSL VPN — FTD (via FMC)

### FMC Configuration Steps

```
! FTD Remote Access VPN is configured through FMC GUI:

! 1. Devices > VPN > Remote Access > Add
!    Name: RA_VPN
!    Target Devices: FTD-01
!
! 2. Connection Profile:
!    Name: Employees
!    Authentication: AAA (ISE RADIUS)
!    Client Address Assignment: VPN_POOL (10.100.0.0/24)
!    Group Policy: GP_EMPLOYEES
!
! 3. AnyConnect Settings:
!    AnyConnect packages: upload .pkg files
!    AnyConnect profiles: upload .xml profile
!
! 4. Access Control:
!    - sysopt connection permit-vpn (bypass ACL for VPN traffic)
!    OR
!    - Create ACP rule for VPN zone traffic
!
! 5. NAT Exemption:
!    NAT rule: source=inside, dest=VPN_POOL → no-nat
!    ! Prevents VPN traffic from being NATted

! FTD FlexConfig (for features not in FMC GUI):
! Devices > FlexConfig > Add FlexConfig Object
! Example: add crypto settings not exposed in GUI
```

## Clientless SSL VPN (WebVPN)

### ASA Clientless VPN

```
! Clientless SSL VPN — browser-based, no client software
webvpn
  enable outside
  anyconnect enable

! Group policy for clientless access
group-policy GP_CLIENTLESS internal
group-policy GP_CLIENTLESS attributes
  vpn-tunnel-protocol ssl-clientless
  webvpn
    url-list value BOOKMARKS
    port-forward enable
    file-access enable
    file-browse enable

! Bookmark list (web applications accessible via portal)
webvpn
  url-list BOOKMARKS "Internal Wiki" https://wiki.internal.example.com
  url-list BOOKMARKS "Ticketing" https://tickets.internal.example.com
  url-list BOOKMARKS "Mail" https://mail.internal.example.com

! Smart tunnel (run thick-client apps through SSL tunnel)
webvpn
  smart-tunnel list SMART_APPS
    smart-tunnel list SMART_APPS "RDP" mstsc.exe platform windows
    smart-tunnel list SMART_APPS "SSH" ssh platform mac-intel

! Port forwarding (legacy — map local port to remote service)
webvpn
  port-forward PORTFWD 3389 server1.internal.example.com 3389
  port-forward PORTFWD 2222 server2.internal.example.com 22
```

## Group Policies

### Group Policy Attributes

```
! Comprehensive group policy configuration
group-policy GP_EMPLOYEES internal
group-policy GP_EMPLOYEES attributes
  ! Tunnel protocol
  vpn-tunnel-protocol ssl-client ssl-clientless ikev2

  ! Address assignment
  address-pools value VPN_POOL

  ! DNS and domain
  dns-server value 10.1.1.53 10.1.2.53
  default-domain value example.com

  ! Split tunneling
  split-tunnel-policy tunnelspecified     ! or tunnelall, excludespecified
  split-tunnel-network-list value SPLIT_ACL
  split-tunnel-all-dns disable

  ! Idle and session timeouts
  vpn-idle-timeout 30                    ! minutes
  vpn-session-timeout 480               ! minutes (8 hours)

  ! Simultaneous logins
  vpn-simultaneous-logins 3

  ! Banner
  banner value "Authorized access only. All activity monitored."

  ! ACL filter
  vpn-filter value VPN_FILTER_ACL

  ! AnyConnect modules
  webvpn
    anyconnect modules value dart,feedback,posture,websecurity
    anyconnect profiles value VPN_PROFILE type user
    anyconnect ask none default anyconnect
    homepage value https://intranet.example.com
```

### Group Policy Inheritance

```
! Group policies inherit from DfltGrpPolicy (built-in)
! Override specific attributes; unset inherits from parent

! Check effective policy
show vpn-sessiondb anyconnect
show running-config group-policy GP_EMPLOYEES

! Inheritance chain:
!   User-specific attributes (RADIUS)
!     ↓ overrides
!   Tunnel-group group-policy
!     ↓ overrides
!   DfltGrpPolicy (system default)

! Force attribute, don't inherit
group-policy GP_LOCKED attributes
  vpn-simultaneous-logins 1             ! explicit value, not inherited
  vpn-idle-timeout none                 ! disable idle timeout
```

## DAP (Dynamic Access Policies)

### DAP Configuration

```
! DAP evaluated after authentication — applies additional restrictions
! based on endpoint attributes and AAA attributes

! DAP selection criteria:
!   - AAA attributes (LDAP groups, RADIUS class)
!   - Endpoint attributes (OS, posture status, AnyConnect version)
!   - Connection type (AnyConnect, clientless, IKEv2)

! DAP example: restrict non-compliant endpoints
! Configuration via ASDM:
!   1. Configuration > Remote Access VPN > Network (Client) Access >
!      Dynamic Access Policies
!   2. Add DAP record:
!      Name: DAP_NONCOMPLIANT
!      Priority: 10
!      Criteria:
!        Endpoint attribute: posture(assessment) = noncompliant
!      Action:
!        Network ACL: RESTRICTED_ACL (limited access)
!        Banner: "Your device is non-compliant. Limited access granted."

! DAP via CLI (ASA 9.x)
dynamic-access-policy-record DAP_NONCOMPLIANT
  priority 10
  description "Non-compliant device — restricted access"
  network-acl RESTRICTED_ACL
  action terminate                      ! or continue

! DAP with LDAP group matching
dynamic-access-policy-record DAP_CONTRACTORS
  priority 20
  description "Contractor access"
  network-acl CONTRACTOR_ACL
  ! AAA attribute: memberOf = CN=Contractors,OU=Groups,DC=example,DC=com
```

### DAP Evaluation Logic

```
! DAP evaluation order:
!   1. All DAP records evaluated against session attributes
!   2. Records with ALL criteria matching are selected
!   3. Multiple matching DAPs are combined (logical AND of ACLs)
!   4. Higher priority number = higher precedence for conflicts
!   5. DfltAccessPolicy applies if no DAP matches

! Precedence:
!   DAP attributes override group-policy and user attributes
!   DAP > User attributes > Group-policy > DfltGrpPolicy

! Action types:
!   continue  — apply DAP attributes and continue session
!   terminate — disconnect the session immediately
!   quarantine — apply quarantine restrictions
```

## Posture Assessment

### AnyConnect Posture Module

```
! Posture assessment checks endpoint compliance before granting access
! Components: AnyConnect Posture Module + ISE Posture Service

! ISE posture policy (configured on ISE):
!   1. Policy > Policy Elements > Conditions > Posture
!      - Antivirus: check if installed + updated
!      - Anti-malware: check definitions age
!      - Disk encryption: check if BitLocker/FileVault enabled
!      - Firewall: check if Windows Firewall enabled
!      - Patch management: check OS patch level
!      - USB device: restrict removable media
!
!   2. Policy > Posture Policy
!      Rule: IF Windows AND CorpAsset
!        THEN require: AV-installed AND FW-enabled AND Encrypted
!
!   3. Results:
!      Compliant     → full access (assign full-access SGT/ACL)
!      Non-compliant → restricted (assign quarantine ACL, remediation URL)
!      Unknown       → limited (posture not assessed yet)

! ASA configuration for posture redirect
access-list POSTURE_REDIRECT extended deny ip any host 10.1.1.100
access-list POSTURE_REDIRECT extended deny udp any any eq 53
access-list POSTURE_REDIRECT extended permit tcp any any eq 80
access-list POSTURE_REDIRECT extended permit tcp any any eq 443

! Group policy — redirect non-compliant to ISE
group-policy GP_POSTURE attributes
  webvpn
    anyconnect modules value posture
```

## Authentication Methods

### RADIUS Authentication (ISE)

```
! AAA server group for ISE
aaa-server ISE_RADIUS protocol radius
aaa-server ISE_RADIUS (management) host 10.10.10.50
  key RADIUS_SECRET
  authentication-port 1812
  accounting-port 1813
  timeout 10
  retry-interval 5

! Tunnel group using RADIUS
tunnel-group EMPLOYEES general-attributes
  authentication-server-group ISE_RADIUS
  accounting-server-group ISE_RADIUS

! RADIUS attributes returned by ISE:
!   Class = group-policy name
!   Filter-Id = ACL name
!   cisco-av-pair = various (dACL, SGT, etc.)
!   Framed-IP-Address = specific IP assignment
!   Session-Timeout = maximum session duration
```

### Certificate Authentication

```
! Authenticate users via client certificate (X.509)
! Requires CA infrastructure + client certs on endpoints

! Import CA certificate
crypto ca trustpoint INTERNAL_CA
  enrollment terminal
  crl configure

crypto ca authenticate INTERNAL_CA
! (paste CA certificate)

! Tunnel group with certificate auth
tunnel-group CERT_VPN type remote-access
tunnel-group CERT_VPN general-attributes
  default-group-policy GP_CERT
tunnel-group CERT_VPN webvpn-attributes
  authentication certificate
  group-url https://vpn.example.com/cert enable

! Certificate mapping (match specific cert fields)
crypto ca certificate map CERT_MAP 10
  subject-name attr cn co example.com
  subject-name attr ou eq engineering

tunnel-group CERT_VPN webvpn-attributes
  group-url https://vpn.example.com/cert enable
```

### MFA Integration

```
! MFA typically via RADIUS to ISE or third-party MFA provider

! Option 1: ISE with MFA (Duo, RSA SecurID, etc.)
!   ISE: Administration > External Identity Sources > RADIUS Token
!   ISE acts as RADIUS proxy to MFA provider
!   ASA → ISE (RADIUS) → Duo/RSA (RADIUS/API)

! Option 2: SAML SSO with MFA
tunnel-group SAML_VPN type remote-access
tunnel-group SAML_VPN webvpn-attributes
  authentication saml
  group-url https://vpn.example.com/saml enable

webvpn
  saml idp https://idp.example.com/saml/metadata
    url sign-in https://idp.example.com/saml/sso
    url sign-out https://idp.example.com/saml/slo
    trustpoint idp IDP_CERT
    trustpoint sp VPN_CERT

! Option 3: Duo LDAP proxy (secondary auth)
!   Primary: AD/LDAP for username/password
!   Secondary: Duo proxy for push/token
aaa-server DUO_LDAP protocol ldap
aaa-server DUO_LDAP (management) host 10.10.10.60
  ldap-base-dn DC=example,DC=com
  ldap-scope subtree
  ldap-login-dn CN=duo_svc,OU=Service,DC=example,DC=com
  ldap-login-password DUO_LDAP_PASS
  server-type microsoft

tunnel-group MFA_VPN general-attributes
  authentication-server-group ISE_RADIUS
  secondary-authentication-server-group DUO_LDAP use-primary-username
```

## Always-On VPN

### Always-On Configuration

```
! AnyConnect always-on VPN — auto-connects at all times
! Prevents users from disconnecting or using network without VPN

! AnyConnect XML profile settings:
! <AnyConnectProfile>
!   <ClientInitialization>
!     <AutoConnectOnStart UserControllable="false">true</AutoConnectOnStart>
!     <AutoReconnect UserControllable="false">true</AutoReconnect>
!     <AutoReconnectBehavior>ReconnectAfterResume</AutoReconnectBehavior>
!   </ClientInitialization>
!   <AlwaysOn UserControllable="false">true</AlwaysOn>
!   <ConnectFailurePolicy>Closed</ConnectFailurePolicy>
!   <!-- Closed = no network access if VPN fails -->
!   <!-- Open = allow network access if VPN fails -->
! </AnyConnectProfile>

! Trusted Network Detection (TND):
! Disable VPN when inside corporate network
! <TrustedNetworkDetection>
!   <TrustedDNSDomains>internal.example.com</TrustedDNSDomains>
!   <TrustedDNSServers>10.1.1.53,10.1.2.53</TrustedDNSServers>
! </TrustedNetworkDetection>
```

## Banner and Portal Customization

```
! Login banner
tunnel-group EMPLOYEES webvpn-attributes
  customization value CORP_THEME

! Customization object
webvpn
  customization CORP_THEME
    title text "Corporate VPN Portal"
    title style "background-color:#003366;color:white;font-size:24px"
    secondary-text text "Authorized Users Only"
    login-message text "Enter your corporate credentials"

! Pre-login banner (displayed before authentication)
group-policy GP_EMPLOYEES attributes
  banner value "WARNING: Unauthorized access prohibited. Activity is monitored."

! Post-login banner
group-policy GP_EMPLOYEES attributes
  webvpn
    homepage value https://intranet.example.com
    customization value CORP_THEME
```

## Troubleshooting

### Show Commands

```
! VPN session summary
show vpn-sessiondb summary
  Active Sessions:
    AnyConnect Client   : 145
    Clientless           : 12
    IKEv2 IPsec         : 8

! Detailed AnyConnect sessions
show vpn-sessiondb anyconnect
  Username   : jdoe
  Index      : 42
  Assigned IP: 10.100.0.25
  Public IP  : 203.0.113.50
  Protocol   : AnyConnect-Parent SSL-Tunnel DTLS-Tunnel
  Encryption : AES-GCM-256     Hashing: SHA256
  Group Policy: GP_EMPLOYEES
  Tunnel Group: EMPLOYEES
  Bytes Tx   : 15234567       Bytes Rx: 98765432
  Duration   : 2h:15m:30s

! Specific session detail
show vpn-sessiondb detail anyconnect filter name jdoe

! SSL VPN status
show webvpn anyconnect

! Crypto status
show crypto protocol statistics ssl

! DAP debugging
show dynamic-access-policy-record

! Group policy details
show running-config group-policy GP_EMPLOYEES
```

### Debug Commands

```
! AnyConnect debugging
debug webvpn anyconnect 255

! SSL VPN debugging
debug webvpn 255

! Authentication debugging
debug aaa authentication
debug aaa authorization
debug radius all

! DAP debugging
debug dap trace
debug dap errors

! IKEv2 RA debugging
debug crypto ikev2 platform 255
debug crypto ikev2 protocol 255

! DTLS debugging
debug webvpn dtls

! Conditional debug (recommended)
debug webvpn condition user jdoe
debug webvpn anyconnect 127
```

### Common Issues

```
# AnyConnect cannot connect
#   - Verify SSL certificate validity (expired cert = connection refused)
#   - Check: show crypto ca certificates
#   - Verify AnyConnect image on flash: dir disk0: | include anyconnect
#   - Check webvpn: show webvpn anyconnect
#   - Verify IP pool not exhausted: show ip local pool VPN_POOL

# Connected but no access to resources
#   - Check split tunnel ACL: show access-list SPLIT_ACL
#   - Check VPN filter: show access-list VPN_FILTER
#   - Check NAT exemption: show nat detail | include VPN
#   - Verify routing: ping from VPN pool to destination
#   - Check DAP: show dynamic-access-policy-record (unexpected DAP match)

# DTLS not establishing (fallback to TLS only)
#   - DTLS requires UDP 443 (firewall may block)
#   - Check: show vpn-sessiondb detail anyconnect (look for DTLS-Tunnel)
#   - MTU issues: reduce DTLS MTU in group policy

# Disconnections / session drops
#   - Idle timeout: increase vpn-idle-timeout
#   - Session timeout: increase vpn-session-timeout
#   - DPD failure: check keepalive settings
#   - SSL renegotiation failure: check certificate expiry

# Authentication failures
#   - test aaa-server ISE_RADIUS host 10.10.10.50 username jdoe password test
#   - Check RADIUS logs on ISE: Operations > RADIUS > Live Log
#   - Verify shared secret match between ASA and ISE
#   - Check certificate chain: show crypto ca certificates

# IP pool exhaustion
#   - show ip local pool VPN_POOL
#   - Increase pool range or add secondary pool
#   - Check for stale sessions: clear vpn-sessiondb anyconnect
```

### Client-Side Diagnostics

```
# AnyConnect DART (Diagnostic And Reporting Tool)
#   Bundled with AnyConnect — collects logs, system info, configs
#   Launch from AnyConnect UI: gear icon → Diagnostics
#   Generates a .zip bundle for support analysis

# AnyConnect log locations:
#   Windows: C:\ProgramData\Cisco\Cisco AnyConnect\Logs\
#   macOS: /opt/cisco/anyconnect/log/
#   Linux: /opt/cisco/anyconnect/log/

# Key log files:
#   vpnagentd.log — VPN agent daemon (connection lifecycle)
#   aciseposture.log — posture assessment
#   csc_ui.log — UI interactions
```

## See Also

- site-to-site-vpn
- ipsec
- tls
- cisco-ftd
- cisco-ise
- pki
- oauth
- zero-trust

## References

- Cisco AnyConnect Secure Mobility Client Administrator Guide
- Cisco ASA VPN Configuration Guide (9.x)
- Cisco FTD Remote Access VPN Configuration (FMC)
- Cisco IOS-XE SSL VPN Configuration Guide
- RFC 7296 — IKEv2 for Remote Access
- RFC 8446 — TLS 1.3
- Cisco ISE Posture Services Configuration Guide
