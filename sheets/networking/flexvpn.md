# FlexVPN (IKEv2-Based VPN Framework)

Cisco's FlexVPN unifies site-to-site, remote-access, and spoke-to-spoke VPN deployments under a single IKEv2-based architecture using tunnel interfaces, smart defaults, AAA integration, and route injection to replace legacy DMVPN, EzVPN, and crypto-map VPNs with a coherent, standards-based framework.

## FlexVPN vs DMVPN

```
Feature                  DMVPN                           FlexVPN
─────────────────────────────────────────────────────────────────────────────────────
Keying Protocol          IKEv1 or IKEv2                  IKEv2 only
Tunnel Mechanism         mGRE + NHRP                     SVTI / DVTI (IKEv2-negotiated)
Spoke Resolution         NHRP Resolution/Redirect        IKEv2 redirect + shortcut
Multicast                Native (mGRE multicast map)     Requires MDT or replication
Config Complexity        Moderate (4 components)         Lower (smart defaults)
Remote Access            Separate (EzVPN / AnyConnect)   Unified (same IKEv2 profile)
EAP Support              IKEv2 only                      Native
Certificate Auth         Supported                       Preferred, deeply integrated
Dual-Stack               Retrofit                        Native (IKEv2 TS)
Standards Compliance     Cisco proprietary (NHRP=RFC)    RFC 7296 (IKEv2) throughout
AAA Integration          Limited                         Full (per-user policy push)
Route Injection          Routing protocol only           RRI + routing protocol
```

### When to Choose FlexVPN Over DMVPN

```
Choose FlexVPN when:
  - You need unified site-to-site + remote-access on the same headend
  - Certificate-based or EAP authentication is required
  - You want IKEv2-native features (redirect, MOBIKE, narrowing)
  - Dual-stack (IPv4 + IPv6) overlay is needed from day one
  - Per-user policy via AAA/RADIUS is a design requirement

Choose DMVPN when:
  - You need native multicast across the overlay
  - Existing DMVPN deployment is stable and feature-complete
  - Spoke-to-spoke traffic dominates and NHRP shortcuts are well-understood
  - IKEv1 support is required for legacy devices
```

## IKEv2 Protocol Fundamentals

### IKEv2 Exchange Overview

```
Initiator (Spoke)                    Responder (Hub)
      |                                    |
      |--- IKE_SA_INIT (HDR, SAi1, KEi, Ni) -->|     Exchange 1: IKE_SA_INIT
      |<-- IKE_SA_INIT (HDR, SAr1, KEr, Nr) ---|     (negotiate crypto, DH exchange)
      |                                    |
      |    [IKE SA established, all subsequent messages encrypted]
      |                                    |
      |--- IKE_AUTH (HDR, SK{IDi, AUTH,    |     Exchange 2: IKE_AUTH
      |     SAi2, TSi, TSr, [CP]}) ------->|     (authenticate, create first CHILD_SA)
      |<-- IKE_AUTH (HDR, SK{IDr, AUTH,    |
      |     SAr2, TSi, TSr, [CP]}) --------|
      |                                    |
      |    [CHILD_SA #1 up, IPsec traffic flows]
      |                                    |
      |--- CREATE_CHILD_SA (HDR, SK{SA,    |     Exchange 3: CREATE_CHILD_SA
      |     Ni, [KEi], TSi, TSr}) -------->|     (rekey or additional SA)
      |<-- CREATE_CHILD_SA (HDR, SK{SA,    |
      |     Nr, [KEr], TSi, TSr}) ---------|
```

### IKEv2 Payloads Relevant to FlexVPN

```
Payload   Code   Purpose in FlexVPN
──────────────────────────────────────────────────────────────────
SA        33     Propose/select crypto suites (encr, prf, integ, dh)
KE        34     Diffie-Hellman public value
Nonce     40     Anti-replay, key derivation input
IDi/IDr   35/36  Peer identity (FQDN, IP, email, DN)
AUTH      39     Authentication data (PSK, RSA, ECDSA)
CERT      37     X.509 certificate
CERTREQ   38     Certificate request (CA hash)
TS        44/45  Traffic selectors (proxy IDs in IKEv1 terms)
CP        47     Configuration payload (IP, DNS, subnet push)
N         41     Notify (REDIRECT, ADDITIONAL_TS_POSSIBLE, etc.)
D         42     Delete (tear down SA)
```

### IKEv2 Proposal Negotiation

```
crypto ikev2 proposal FLEX-PROPOSAL
 encryption aes-cbc-256 aes-cbc-128
 integrity sha512 sha256
 group 20 19 14

! Responder selects first matching (in its preference order)
! IKEv2 negotiates a single suite, not mix-and-match like IKEv1
```

## FlexVPN Hub-and-Spoke

### Hub Configuration (Complete)

```
! --- PKI ---
crypto pki trustpoint FLEX-CA
 enrollment url http://ca.example.com/certsrv/mscep/mscep.dll
 revocation-check crl
 rsakeypair FLEX-RSA 2048

! --- IKEv2 Proposal ---
crypto ikev2 proposal FLEX-PROP
 encryption aes-cbc-256
 integrity sha256
 group 19

! --- IKEv2 Policy ---
crypto ikev2 policy FLEX-POLICY
 match fvrf any
 proposal FLEX-PROP

! --- IKEv2 Keyring ---
crypto ikev2 keyring FLEX-KEYRING
 peer SPOKES
  address 0.0.0.0 0.0.0.0
  pre-shared-key FLEX-PSK-KEY

! --- IKEv2 Profile ---
crypto ikev2 profile FLEX-PROFILE
 match identity remote address 0.0.0.0
 identity local fqdn hub.example.com
 authentication remote pre-share
 authentication local pre-share
 keyring local FLEX-KEYRING
 dpd 30 5 on-demand
 aaa authorization group psk list FLEX-AAA FLEX-AUTHOR
 virtual-template 1

! --- IPsec Transform Set ---
crypto ipsec transform-set FLEX-TS esp-aes 256 esp-sha256-hmac
 mode tunnel

! --- IPsec Profile ---
crypto ipsec profile FLEX-IPSEC
 set transform-set FLEX-TS
 set ikev2-profile FLEX-PROFILE

! --- Loopback for SVTI source ---
interface Loopback0
 ip address 10.0.0.1 255.255.255.255

! --- Virtual-Template (DVTI) ---
interface Virtual-Template1 type tunnel
 ip unnumbered Loopback0
 ip mtu 1400
 ip tcp adjust-mss 1360
 tunnel source GigabitEthernet0/0
 tunnel mode ipsec ipv4
 tunnel protection ipsec profile FLEX-IPSEC
```

### Spoke Configuration (Complete)

```
! --- IKEv2 Proposal ---
crypto ikev2 proposal FLEX-PROP
 encryption aes-cbc-256
 integrity sha256
 group 19

! --- IKEv2 Policy ---
crypto ikev2 policy FLEX-POLICY
 proposal FLEX-PROP

! --- IKEv2 Keyring ---
crypto ikev2 keyring FLEX-KEYRING
 peer HUB
  address 198.51.100.1
  pre-shared-key FLEX-PSK-KEY

! --- IKEv2 Profile ---
crypto ikev2 profile FLEX-PROFILE
 match identity remote fqdn hub.example.com
 identity local fqdn spoke1.example.com
 authentication remote pre-share
 authentication local pre-share
 keyring local FLEX-KEYRING
 dpd 30 5 on-demand

! --- IPsec Profile ---
crypto ipsec transform-set FLEX-TS esp-aes 256 esp-sha256-hmac
 mode tunnel

crypto ipsec profile FLEX-IPSEC
 set transform-set FLEX-TS
 set ikev2-profile FLEX-PROFILE

! --- Static VTI (SVTI) ---
interface Tunnel0
 ip address 10.255.0.2 255.255.255.0
 ip mtu 1400
 ip tcp adjust-mss 1360
 tunnel source GigabitEthernet0/0
 tunnel destination 198.51.100.1
 tunnel mode ipsec ipv4
 tunnel protection ipsec profile FLEX-IPSEC
```

## FlexVPN Spoke-to-Spoke (Shortcut Switching)

```
Spoke A                        Hub                         Spoke B
   |                            |                            |
   |--- IKE_SA + CHILD_SA ---->|                            |
   |                            |<--- IKE_SA + CHILD_SA ----|
   |                            |                            |
   |    (Spoke A sends packet to Spoke B subnet)            |
   |--- encrypted traffic ---->|                            |
   |                            |--- REDIRECT notify ------>|   (hub tells Spoke A
   |<-- REDIRECT notify -------|                            |    about Spoke B)
   |                            |                            |
   |--- IKE_SA_INIT directly --|--------------------------->|   (direct tunnel)
   |<-- IKE_SA_INIT response --|----------------------------|
   |--- IKE_AUTH --------------|--------------------------->|
   |<-- IKE_AUTH --------------|----------------------------|
   |                            |                            |
   |--- direct encrypted traffic (bypasses hub) ----------->|
```

### Hub Shortcut Configuration

```
crypto ikev2 profile FLEX-PROFILE
 match identity remote address 0.0.0.0
 authentication remote pre-share
 authentication local pre-share
 keyring local FLEX-KEYRING
 dpd 30 5 on-demand
 aaa authorization group psk list FLEX-AAA FLEX-AUTHOR
 virtual-template 1

! Enable IKEv2 redirect for shortcut switching
crypto ikev2 redirect gateway init
```

### Spoke Shortcut Configuration

```
crypto ikev2 profile FLEX-PROFILE
 match identity remote fqdn hub.example.com
 match identity remote address 0.0.0.0      ! Accept other spokes
 authentication remote pre-share
 authentication local pre-share
 keyring local FLEX-KEYRING
 dpd 30 5 on-demand
 virtual-template 1                          ! DVTI for shortcut tunnels

crypto ikev2 redirect client
```

## Smart Defaults

```
! IKEv2 smart defaults reduce boilerplate configuration
! When no explicit proposal/policy is configured, IOS uses:

Default IKEv2 Proposal:
  encryption: aes-cbc-256, aes-cbc-192, aes-cbc-128
  integrity:  sha512, sha384, sha256, sha1
  group:      19, 20, 21, 14, 5, 2

Default IPsec Transform:
  esp-aes-256 + esp-sha-hmac (tunnel mode)

! Smart defaults allow minimal configuration:
crypto ikev2 profile MINIMAL
 match identity remote address 0.0.0.0
 authentication remote pre-share
 authentication local pre-share
 keyring local MY-KEYRING

! No proposal, policy, transform-set, or ipsec profile needed
! Smart defaults automatically apply
```

### Overriding Smart Defaults

```
! To lock down negotiation, always define explicit proposals:
crypto ikev2 proposal HARDENED
 encryption aes-gcm-256            ! AEAD cipher (no separate integrity)
 group 19                          ! ECP-256 only

crypto ikev2 policy HARDENED-POLICY
 match fvrf any
 proposal HARDENED

! AES-GCM eliminates the integrity algorithm (combined mode)
```

## Static VTI (SVTI) vs Dynamic VTI (DVTI)

```
Feature          SVTI (Tunnel interface)        DVTI (Virtual-Template/VA)
──────────────────────────────────────────────────────────────────────────────
Direction        Point-to-point (one peer)      Point-to-multipoint (any peer)
Interface        Tunnel0, Tunnel1, ...          Virtual-Access (cloned from VT)
Use Case         Spoke -> Hub (known dest)      Hub accepting N spokes
IP Addressing    Static per tunnel              Unnumbered or per-peer via AAA
Routing          Static or dynamic per tunnel   Dynamic, route injection (RRI)
Scale            One interface per peer          Unlimited (virtual-access)
Config           tunnel destination <ip>         virtual-template <n> type tunnel
```

### DVTI Virtual-Template Details

```
! The Virtual-Template is cloned into Virtual-Access interfaces on demand
! Each spoke connection creates a new Virtual-Access interface

show interfaces virtual-access 1 configuration
! Output:
!  Interface Virtual-Access1
!   ip unnumbered Loopback0
!   tunnel source GigabitEthernet0/0
!   tunnel mode ipsec ipv4
!   tunnel destination <dynamically set by IKEv2>
!   tunnel protection ipsec profile FLEX-IPSEC
```

## FlexVPN Server / Client Model

### FlexVPN Server (Hub as Server)

```
! --- AAA Configuration ---
aaa new-model
aaa authorization network FLEX-AAA local
aaa accounting network FLEX-ACCT start-stop group radius

! --- Authorization Policy ---
crypto ikev2 authorization policy FLEX-AUTHOR
 pool FLEX-POOL
 route set interface
 route set access-list FLEX-ROUTES
 dns 10.0.0.53
 def-domain example.com

! --- IP Pool ---
ip local pool FLEX-POOL 10.255.1.1 10.255.1.254

! --- ACL for Route Push ---
ip access-list standard FLEX-ROUTES
 permit 10.0.0.0 0.255.255.255
 permit 172.16.0.0 0.15.255.255

! --- IKEv2 Profile with Server Role ---
crypto ikev2 profile FLEX-SERVER
 match identity remote key-id FLEX-CLIENT
 identity local fqdn hub.example.com
 authentication remote pre-share
 authentication local rsa-sig
 keyring local FLEX-KEYRING
 pki trustpoint FLEX-CA
 dpd 30 5 on-demand
 aaa authorization group psk list FLEX-AAA FLEX-AUTHOR
 aaa authorization user psk list FLEX-AAA
 virtual-template 1
```

### FlexVPN Client (Spoke as Client)

```
crypto ikev2 profile FLEX-CLIENT
 match identity remote fqdn hub.example.com
 identity local key-id FLEX-CLIENT
 authentication remote rsa-sig
 authentication local pre-share
 keyring local FLEX-KEYRING
 pki trustpoint FLEX-CA
 dpd 30 5 on-demand

interface Tunnel0
 ip address negotiated                    ! Receive IP from server pool
 ip mtu 1400
 ip tcp adjust-mss 1360
 tunnel source GigabitEthernet0/0
 tunnel destination 198.51.100.1
 tunnel mode ipsec ipv4
 tunnel protection ipsec profile FLEX-IPSEC
```

## AAA Authorization for FlexVPN

### Local Authorization

```
crypto ikev2 authorization policy SITE-A-POLICY
 pool SITE-A-POOL
 route set interface
 route set access-list SITE-A-ROUTES
 banner ^C Welcome to FlexVPN ^C

crypto ikev2 profile FLEX-PROFILE
 aaa authorization group psk list default SITE-A-POLICY
 aaa authorization user psk list default
```

### RADIUS-Based Authorization

```
! Router configuration
aaa new-model
aaa authorization network FLEX-AAA group radius
radius server FLEX-RAD
 address ipv4 10.0.0.100 auth-port 1812 acct-port 1813
 key RADIUS-SECRET

crypto ikev2 profile FLEX-PROFILE
 aaa authorization group psk list FLEX-AAA
 aaa authorization user psk list FLEX-AAA

! RADIUS server returns attributes:
!   Cisco-AV-Pair = "ip:interface-config=ip unnumbered Loopback0"
!   Cisco-AV-Pair = "ip:route=10.1.1.0 255.255.255.0"
!   Cisco-AV-Pair = "ip:addr-pool=FLEX-POOL"
!   Framed-IP-Address = 10.255.1.50
```

### Per-User vs Per-Group Authorization

```
! Group authorization: applied to all peers matching the IKEv2 profile
aaa authorization group psk list FLEX-AAA FLEX-GROUP-POLICY

! User authorization: applied per individual peer identity
aaa authorization user psk list FLEX-AAA
! Looks up the peer's IKEv2 identity in RADIUS/local database
! User attributes override group attributes where they conflict
```

## Dual-Stack (IPv4 + IPv6) FlexVPN

### IKEv2 Traffic Selectors for Dual-Stack

```
! Hub: Virtual-Template with dual-stack
interface Virtual-Template1 type tunnel
 ip unnumbered Loopback0
 ipv6 unnumbered Loopback1
 tunnel source GigabitEthernet0/0
 tunnel mode ipsec ipv4           ! Underlay is IPv4
 tunnel protection ipsec profile FLEX-IPSEC

! IKEv2 negotiates separate traffic selectors for v4 and v6
! TSi = [0.0.0.0/0, ::/0]   TSr = [0.0.0.0/0, ::/0]
! Creates CHILD_SAs covering both address families

! Spoke: SVTI with dual-stack
interface Tunnel0
 ip address 10.255.0.2 255.255.255.0
 ipv6 address 2001:db8:ff::2/64
 tunnel source GigabitEthernet0/0
 tunnel destination 198.51.100.1
 tunnel mode ipsec ipv4
 tunnel protection ipsec profile FLEX-IPSEC

! Verify traffic selectors
show crypto ikev2 sa detail
! Look for:
!   Local selectors:  0.0.0.0/0 - 255.255.255.255/255
!                     ::/0 - ffff:ffff:..../128
!   Remote selectors: 0.0.0.0/0 - 255.255.255.255/255
!                     ::/0 - ffff:ffff:..../128
```

### IPv6-over-IPv6 FlexVPN

```
! When underlay is also IPv6:
interface Tunnel0
 ipv6 address 2001:db8:ff::2/64
 tunnel source GigabitEthernet0/0
 tunnel destination 2001:db8:1::1
 tunnel mode ipsec ipv6           ! IPv6 underlay
 tunnel protection ipsec profile FLEX-IPSEC
```

## Route Injection (RRI --- Reverse Route Injection)

```
! RRI automatically creates static routes for remote subnets
! announced by the spoke via IKEv2 traffic selectors

crypto ipsec profile FLEX-IPSEC
 set transform-set FLEX-TS
 set ikev2-profile FLEX-PROFILE
 set reverse-route                         ! Enable RRI

! When Spoke1 connects with TSi = 10.1.1.0/24:
! Hub installs: ip route 10.1.1.0 255.255.255.0 Virtual-Access1

! Verify RRI routes
show ip route | include %
! 10.1.1.0/24 [1/0] via 0.0.0.0, Virtual-Access1  (RRI)

! RRI with distance and tag
crypto ipsec profile FLEX-IPSEC
 set reverse-route distance 100 tag 999

! Redistribute RRI routes into OSPF/BGP
router ospf 1
 redistribute static subnets route-map RRI-TO-OSPF

route-map RRI-TO-OSPF permit 10
 match tag 999
 set metric-type 1
```

### RRI with AAA Route Push

```
! AAA can push specific routes (overrides traffic selector-based RRI)
crypto ikev2 authorization policy SPOKE1-POLICY
 route set access-list SPOKE1-ROUTES

ip access-list standard SPOKE1-ROUTES
 permit 10.1.1.0 0.0.0.255
 permit 10.1.2.0 0.0.0.255
```

## FlexVPN with Routing Protocols

### FlexVPN + OSPF

```
! Hub
router ospf 1
 router-id 10.0.0.1
 network 10.0.0.0 0.0.0.255 area 0
 network 10.255.0.0 0.0.0.255 area 0

interface Virtual-Template1 type tunnel
 ip ospf 1 area 0
 ip ospf network point-to-point
 ip ospf mtu-ignore

! Spoke
router ospf 1
 router-id 10.1.1.1
 network 10.1.1.0 0.0.0.255 area 0
 network 10.255.0.0 0.0.0.255 area 0

interface Tunnel0
 ip ospf 1 area 0
 ip ospf network point-to-point
 ip ospf mtu-ignore
```

### FlexVPN + BGP

```
! Hub (Route Reflector)
router bgp 65000
 bgp router-id 10.0.0.1
 bgp log-neighbor-changes
 neighbor FLEX-SPOKES peer-group
 neighbor FLEX-SPOKES remote-as 65000
 neighbor FLEX-SPOKES update-source Loopback0
 neighbor FLEX-SPOKES route-reflector-client
 neighbor FLEX-SPOKES next-hop-self
 !
 address-family ipv4
  neighbor FLEX-SPOKES activate

! Spoke
router bgp 65000
 bgp router-id 10.1.1.1
 neighbor 10.0.0.1 remote-as 65000
 neighbor 10.0.0.1 update-source Tunnel0
 !
 address-family ipv4
  network 10.1.1.0 mask 255.255.255.0
  neighbor 10.0.0.1 activate
```

### FlexVPN + EIGRP

```
! Hub
router eigrp FLEX
 address-family ipv4 unicast autonomous-system 100
  topology base
  exit-af-topology
  network 10.0.0.0 0.0.0.255
  network 10.255.0.0 0.0.0.255
  af-interface Virtual-Template1
   no split-horizon                        ! Allow spoke routes via hub
  exit-af-interface

! Spoke
router eigrp FLEX
 address-family ipv4 unicast autonomous-system 100
  topology base
  exit-af-topology
  network 10.1.1.0 0.0.0.255
  network 10.255.0.0 0.0.0.255
  af-interface Tunnel0
   stub-site
  exit-af-interface
```

## Configuration Templates

### Minimal Hub (Smart Defaults + PSK)

```
crypto ikev2 keyring KR
 peer ANY
  address 0.0.0.0 0.0.0.0
  pre-shared-key SIMPLE-KEY

crypto ikev2 profile SIMPLE
 match identity remote address 0.0.0.0
 authentication remote pre-share
 authentication local pre-share
 keyring local KR
 virtual-template 1

interface Loopback0
 ip address 10.0.0.1 255.255.255.255

interface Virtual-Template1 type tunnel
 ip unnumbered Loopback0
 tunnel source GigabitEthernet0/0
 tunnel mode ipsec ipv4
 tunnel protection ipsec profile default
```

### Minimal Spoke (Smart Defaults + PSK)

```
crypto ikev2 keyring KR
 peer HUB
  address 198.51.100.1
  pre-shared-key SIMPLE-KEY

crypto ikev2 profile SIMPLE
 match identity remote address 198.51.100.1
 authentication remote pre-share
 authentication local pre-share
 keyring local KR

interface Tunnel0
 ip address 10.255.0.2 255.255.255.0
 tunnel source GigabitEthernet0/0
 tunnel destination 198.51.100.1
 tunnel mode ipsec ipv4
 tunnel protection ipsec profile default
```

### Certificate-Based Hub

```
crypto pki trustpoint FLEX-CA
 enrollment url http://ca.example.com:80
 revocation-check crl
 rsakeypair FLEX-RSA 4096

crypto ikev2 profile CERT-PROFILE
 match identity remote any
 identity local dn
 authentication remote rsa-sig
 authentication local rsa-sig
 pki trustpoint FLEX-CA
 dpd 30 5 on-demand
 virtual-template 1
```

## Troubleshooting

### show crypto ikev2 sa

```
show crypto ikev2 sa
! Output:
! Tunnel-id  Local       Remote        fvrf/ivrf  Status
! 1          198.51.100.1/500 203.0.113.10/500 none/none READY
!     Encr: AES-CBC, keysize: 256, PRF: SHA256, Hash: SHA256, DH Grp:19
!     Auth sign: PSK, Auth verify: PSK
!     Life/Active Time: 86400/3521 sec

show crypto ikev2 sa detail
! Shows traffic selectors, child SA count, identity, rekey info, etc.
```

### show crypto ikev2 session

```
show crypto ikev2 session
! Displays active IKEv2 sessions with child SA mapping

show crypto ikev2 session detail
! Includes:
!   - Session ID, tunnel ID, IKE count, child count
!   - Remote identity (FQDN, IP, email)
!   - Local/remote traffic selectors
!   - Packets encrypted/decrypted
```

### show crypto ikev2 stats

```
show crypto ikev2 stats
! Aggregate statistics:
!   IKE_SA_INIT sent/received
!   IKE_AUTH sent/received
!   CREATE_CHILD_SA sent/received
!   INFORMATIONAL sent/received
!   Invalid IKE SPI / invalid SPI on CHILD
!   Auth failures, proposal mismatches
```

### show crypto ikev2 profile

```
show crypto ikev2 profile
! Lists all profiles with match criteria, authentication methods, etc.

show crypto ikev2 profile detail FLEX-PROFILE
! Full configuration dump of a specific profile
```

### show crypto ipsec sa

```
show crypto ipsec sa
! Per-SA detail: SPI, transform, packets encr/decr, errors
! Check for incrementing counters on both encaps and decaps

show crypto ipsec sa interface Tunnel0
! Filter to a specific interface
```

### Debug Commands

```
! Structured debug for IKEv2
debug crypto ikev2                         ! All IKEv2 events
debug crypto ikev2 error                   ! Errors only
debug crypto ikev2 packet                  ! Packet-level detail (verbose)
debug crypto ikev2 internal                ! State machine transitions

! Conditional debugging (production-safe)
debug crypto condition peer ipv4 203.0.113.10
debug crypto ikev2

! IPsec debug
debug crypto ipsec

! Clear SAs for testing
clear crypto ikev2 sa                      ! All IKEv2 SAs
clear crypto ikev2 sa <tunnel-id>          ! Specific tunnel
clear crypto sa                            ! All IPsec SAs
```

### Common Failure Scenarios

```
Problem: IKE_SA_INIT fails
  Check: Proposals match (encr, integ, group)
  Check: UDP 500/4500 not blocked by firewall
  Check: Correct peer address and reachability
  Verify: show crypto ikev2 proposal

Problem: IKE_AUTH fails
  Check: PSK mismatch (case-sensitive)
  Check: Certificate trust chain (show crypto pki cert, verify)
  Check: Identity mismatch (match identity remote vs actual)
  Verify: show crypto ikev2 profile, debug crypto ikev2

Problem: Tunnel up but no traffic
  Check: Traffic selectors (TSi/TSr) match desired subnets
  Check: Routing --- spoke has route to hub LAN and vice versa
  Check: ip mtu / ip tcp adjust-mss on tunnel interfaces
  Verify: show crypto ipsec sa (look for encaps/decaps counters)

Problem: Spoke-to-spoke shortcut fails
  Check: crypto ikev2 redirect gateway init (hub)
  Check: crypto ikev2 redirect client (spoke)
  Check: Spoke accepts connections from other spokes (match identity)
  Check: Spoke has virtual-template for inbound shortcut tunnels
  Verify: show crypto ikev2 redirect

Problem: RRI routes not appearing
  Check: set reverse-route in ipsec profile
  Check: Traffic selectors (TSr from spoke must include subnets)
  Verify: show ip route static, show crypto ipsec sa (remote ident)

Problem: AAA policy not applied
  Check: aaa authorization group/user lines in ikev2 profile
  Check: crypto ikev2 authorization policy exists and is correct
  Check: RADIUS reachability if using external AAA
  Verify: debug crypto ikev2, test aaa group <method> <user> <pass>
```

## Tips

- Start with smart defaults for proof-of-concept, then add explicit proposals and policies for production hardening.
- Use `tunnel mode ipsec ipv4` (not GRE/IPsec) for FlexVPN --- this avoids the GRE overhead and uses native IKEv2 tunnel negotiation.
- Always set `ip mtu 1400` and `ip tcp adjust-mss 1360` on tunnel interfaces to prevent fragmentation issues.
- For hub redundancy, deploy two hubs with IKEv2 redirect --- the primary can redirect spokes to the secondary during maintenance.
- Use certificate authentication in production --- PSK does not scale and cannot be rotated per-peer without reconfiguration.
- Enable `dpd 30 5 on-demand` for dead peer detection --- `periodic` mode generates unnecessary keepalive traffic.
- Tag RRI routes (`set reverse-route tag 999`) and use route-maps to control redistribution into IGP/BGP.
- When migrating from DMVPN, run both in parallel (different tunnel interfaces) and migrate spokes incrementally.
- Use `show crypto ikev2 sa detail` as your primary verification command --- it shows everything (proposals, identities, traffic selectors, child SAs).
- For troubleshooting, always start with `debug crypto condition peer ipv4 <ip>` before enabling debugs to avoid overwhelming the console.

## See Also

- `ipsec` --- IPsec fundamentals, ESP/AH, transport vs tunnel mode
- `dmvpn` --- DMVPN phases, mGRE, NHRP, migration considerations
- `eigrp` --- EIGRP named mode, stub routing, split-horizon
- `ospf` --- OSPF network types, area design, point-to-point links
- `bgp` --- BGP route reflection, iBGP, next-hop-self
- `radius` --- RADIUS authentication, authorization attributes
- `tacacs` --- TACACS+ for device management AAA

## References

- RFC 7296 --- Internet Key Exchange Protocol Version 2 (IKEv2)
- RFC 4303 --- IP Encapsulating Security Payload (ESP)
- RFC 4306 --- IKEv2 (obsoleted by RFC 7296, useful for historical context)
- RFC 5685 --- Redirect Mechanism for IKEv2
- RFC 7383 --- IKEv2 Message Fragmentation
- RFC 5998 --- An Extension for EAP-Only Authentication in IKEv2
- RFC 4555 --- IKEv2 Mobility and Multihoming Protocol (MOBIKE)
- Cisco FlexVPN Configuration Guide --- https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/sec_conn_ike2vpn/configuration/xe-16/sec-flex-vpn-xe-16-book.html
- Cisco IKEv2 Configuration Guide --- https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/sec_conn_ikevpn/configuration/xe-16/sec-sec-for-vpns-w-ipsec-xe-16-book.html
- Cisco FlexVPN Design Guide --- https://www.cisco.com/c/en/us/support/docs/security/flexvpn/200555-FlexVPN-Between-a-Hub-and-a-Remote-Spoke.html
