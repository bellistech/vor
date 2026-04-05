# Site-to-Site VPN (IPsec / IKE / GRE / VTI)

> Connect remote sites securely over untrusted networks using IPsec tunnels with IKEv1/IKEv2, crypto maps, VTI, GRE over IPsec, DMVPN, and FlexVPN on Cisco IOS-XE.

## IKEv1 Configuration

### Phase 1 — ISAKMP Policy (Main Mode / Aggressive Mode)

```
! Main mode: 6 messages, identity protection (recommended)
! Aggressive mode: 3 messages, faster, exposes identity (avoid if possible)

! Define ISAKMP policy (Phase 1)
crypto isakmp policy 10
  encryption aes 256
  hash sha256
  authentication pre-share
  group 14                          ! DH group 14 (2048-bit)
  lifetime 86400                    ! 24 hours

! Pre-shared key
crypto isakmp key COMPLEX_PSK_KEY address 203.0.113.1

! Aggressive mode (if required — e.g., dynamic peer IP)
crypto isakmp peer address 203.0.113.1
  set aggressive-mode client-endpoint ipv4-address 198.51.100.1
  set aggressive-mode password COMPLEX_PSK_KEY
```

### Phase 2 — IPsec Transform Set and Crypto Map

```
! Define transform set (Phase 2 encryption + integrity)
crypto ipsec transform-set MY_TSET esp-aes 256 esp-sha256-hmac
  mode tunnel                       ! tunnel mode (default)

! Define interesting traffic (ACL)
ip access-list extended VPN_TRAFFIC
  permit ip 10.1.0.0 0.0.255.255 10.2.0.0 0.0.255.255

! Define crypto map
crypto map MY_CMAP 10 ipsec-isakmp
  set peer 203.0.113.1
  set transform-set MY_TSET
  set pfs group14                   ! PFS with DH group 14
  set security-association lifetime seconds 3600
  set security-association lifetime kilobytes 4096000
  match address VPN_TRAFFIC

! Apply crypto map to WAN interface
interface GigabitEthernet0/0
  ip address 198.51.100.1 255.255.255.0
  crypto map MY_CMAP
```

## IKEv2 Configuration

### IKEv2 Proposal and Policy

```
! IKEv2 proposal (replaces ISAKMP policy)
crypto ikev2 proposal IKEv2_PROP
  encryption aes-cbc-256
  integrity sha256
  group 14 19 20                    ! DH groups: 14, 19(ECP-256), 20(ECP-384)

! IKEv2 policy (binds proposal to scope)
crypto ikev2 policy IKEv2_POL
  match fvrf any                    ! front-door VRF (any)
  proposal IKEv2_PROP

! IKEv2 keyring (PSK)
crypto ikev2 keyring MY_KEYRING
  peer SITE_B
    address 203.0.113.1
    pre-shared-key COMPLEX_PSK_KEY

! IKEv2 profile
crypto ikev2 profile IKEv2_PROF
  match identity remote address 203.0.113.1 255.255.255.255
  authentication remote pre-share
  authentication local pre-share
  keyring local MY_KEYRING
  lifetime 86400
  dpd 10 3 periodic                 ! DPD: 10s interval, 3 retries
```

### IKEv2 with Certificates

```
! Enroll with CA
crypto pki trustpoint MY_CA
  enrollment url http://ca-server.example.com/certsrv/mscep/mscep.dll
  serial-number
  ip-address 198.51.100.1
  subject-name CN=router1.example.com,O=Example
  revocation-check crl
  rsakeypair MY_RSA_KEY 2048

crypto pki authenticate MY_CA
crypto pki enroll MY_CA

! IKEv2 profile with certificate auth
crypto ikev2 profile IKEv2_CERT_PROF
  match identity remote dn
  authentication remote rsa-sig
  authentication local rsa-sig
  pki trustpoint MY_CA
  dpd 10 3 periodic
```

### IKEv2 IPsec Profile (for VTI)

```
! IPsec transform set
crypto ipsec transform-set IKEv2_TSET esp-aes 256 esp-sha256-hmac
  mode tunnel

! IPsec profile (used with VTI instead of crypto map)
crypto ipsec profile IKEv2_IPSEC_PROF
  set transform-set IKEv2_TSET
  set pfs group19                   ! ECP-256 for PFS
  set ikev2-profile IKEv2_PROF
  set security-association lifetime seconds 3600
```

## VTI (Virtual Tunnel Interface)

### Static VTI (sVTI)

```
! Static VTI — point-to-point, routable interface
interface Tunnel0
  ip address 172.16.0.1 255.255.255.252
  tunnel source GigabitEthernet0/0
  tunnel destination 203.0.113.1
  tunnel mode ipsec ipv4
  tunnel protection ipsec profile IKEv2_IPSEC_PROF
  ip mtu 1400
  ip tcp adjust-mss 1360

! Route traffic through VTI
ip route 10.2.0.0 255.255.0.0 Tunnel0

! Advantages over crypto map:
!   - Routable interface (supports routing protocols)
!   - Per-tunnel QoS, NetFlow, ACL
!   - No "interesting traffic" ACL needed
!   - Cleaner failover with routing protocol convergence
```

### Dynamic VTI (dVTI)

```
! Dynamic VTI — hub creates tunnel interfaces on demand
! Used for hub-and-spoke with many spokes

! Virtual-template (hub)
interface Virtual-Template1 type tunnel
  ip unnumbered Loopback0
  tunnel mode ipsec ipv4
  tunnel protection ipsec profile IKEv2_IPSEC_PROF

! IKEv2 profile referencing virtual-template
crypto ikev2 profile IKEv2_DVTI_PROF
  match identity remote address 0.0.0.0
  authentication remote pre-share
  authentication local pre-share
  keyring local MY_KEYRING
  virtual-template 1                ! creates Virtual-Access interfaces

! Spoke config — standard sVTI pointing to hub
interface Tunnel0
  ip address 172.16.0.2 255.255.255.252
  tunnel source GigabitEthernet0/0
  tunnel destination 198.51.100.1   ! hub address
  tunnel mode ipsec ipv4
  tunnel protection ipsec profile IKEv2_IPSEC_PROF
```

## GRE over IPsec

### Basic GRE over IPsec

```
! GRE tunnel (carries multicast, routing protocols)
interface Tunnel0
  ip address 172.16.0.1 255.255.255.252
  tunnel source GigabitEthernet0/0
  tunnel destination 203.0.113.1
  tunnel mode gre ip                ! standard GRE
  ip mtu 1400
  ip tcp adjust-mss 1360

! Protect GRE with IPsec (crypto map method)
ip access-list extended GRE_TRAFFIC
  permit gre host 198.51.100.1 host 203.0.113.1

crypto map MY_CMAP 10 ipsec-isakmp
  set peer 203.0.113.1
  set transform-set MY_TSET
  match address GRE_TRAFFIC

interface GigabitEthernet0/0
  crypto map MY_CMAP

! Alternative: GRE with tunnel protection (cleaner)
interface Tunnel0
  ip address 172.16.0.1 255.255.255.252
  tunnel source GigabitEthernet0/0
  tunnel destination 203.0.113.1
  tunnel mode gre ip
  tunnel protection ipsec profile IKEv2_IPSEC_PROF
```

### GRE over IPsec MTU and Fragmentation

```
! Overhead calculation:
!   Original packet:                          up to 1500 bytes
!   + GRE header:                              4 bytes (+ 4 optional key)
!   + New IP header (GRE outer):              20 bytes
!   + ESP header:                              8 bytes
!   + ESP IV (AES-CBC):                       16 bytes
!   + ESP trailer (padding + pad length + NH): 2-17 bytes
!   + ESP ICV (SHA-256):                      16 bytes
!   + New IP header (IPsec outer):            20 bytes
!   Total overhead:                           ~80-100 bytes

! Recommended MTU settings
interface Tunnel0
  ip mtu 1400                       ! GRE payload MTU
  ip tcp adjust-mss 1360            ! TCP MSS clamping (MTU - 40)

! Enable path MTU discovery
interface Tunnel0
  tunnel path-mtu-discovery
```

## DMVPN Reference

### DMVPN Phase 3 (Spoke-to-Spoke Direct)

```
! Hub configuration
interface Tunnel0
  ip address 172.16.0.1 255.255.255.0
  ip nhrp network-id 1
  ip nhrp authentication NHRP_KEY
  ip nhrp map multicast dynamic
  ip nhrp redirect                  ! Phase 3: tell spokes to go direct
  tunnel source GigabitEthernet0/0
  tunnel mode gre multipoint
  tunnel protection ipsec profile IKEv2_IPSEC_PROF
  ip mtu 1400
  ip tcp adjust-mss 1360

! Spoke configuration
interface Tunnel0
  ip address 172.16.0.2 255.255.255.0
  ip nhrp network-id 1
  ip nhrp authentication NHRP_KEY
  ip nhrp nhs 172.16.0.1 nbma 198.51.100.1 multicast
  ip nhrp shortcut                  ! Phase 3: install shortcut routes
  tunnel source GigabitEthernet0/0
  tunnel mode gre multipoint
  tunnel protection ipsec profile IKEv2_IPSEC_PROF
  ip mtu 1400
  ip tcp adjust-mss 1360

! Routing (EIGRP or BGP over DMVPN)
router eigrp 100
  network 172.16.0.0 0.0.0.255
  network 10.2.0.0 0.0.255.255
  no auto-summary
```

## FlexVPN Reference

### FlexVPN Hub-and-Spoke (IKEv2-Based)

```
! FlexVPN hub
crypto ikev2 authorization policy FLEX_AUTH
  route set interface
  route set access-list FLEX_ROUTES

crypto ikev2 profile FLEX_PROF
  match identity remote address 0.0.0.0
  authentication remote pre-share
  authentication local pre-share
  keyring local MY_KEYRING
  virtual-template 1
  aaa authorization group psk list default FLEX_AUTH

interface Virtual-Template1 type tunnel
  ip unnumbered Loopback0
  tunnel mode ipsec ipv4
  tunnel protection ipsec profile IKEv2_IPSEC_PROF

! FlexVPN spoke
interface Tunnel0
  ip address negotiated              ! IP assigned by hub
  tunnel source GigabitEthernet0/0
  tunnel destination 198.51.100.1
  tunnel mode ipsec ipv4
  tunnel protection ipsec profile IKEv2_IPSEC_PROF

! FlexVPN advantages over DMVPN:
!   - Pure IKEv2 (no GRE/NHRP dependency)
!   - Dynamic routing via IKEv2 config exchange
!   - Simpler configuration
!   - Better integration with certificate PKI
```

## IPsec Parameters

### DPD (Dead Peer Detection)

```
! IKEv2 DPD
crypto ikev2 profile MY_PROF
  dpd 10 3 periodic                 ! check every 10s, 3 retries, always send
  dpd 10 3 on-demand                ! check only when there's traffic to send

! IKEv1 DPD
crypto isakmp keepalive 10 3 periodic

! DPD behavior:
!   periodic: sends DPD probes at interval regardless of traffic
!   on-demand: sends DPD probe only when outbound traffic exists and
!              no inbound traffic has been received within the interval
```

### NAT Traversal (NAT-T)

```
! NAT-T automatically encapsulates ESP in UDP 4500 when NAT is detected
! Enabled by default in IKEv2; must be enabled for IKEv1

! IKEv1: enable NAT-T
crypto isakmp nat-traversal 30      ! keepalive interval in seconds

! IKEv2: enabled by default (disable with)
crypto ikev2 nat-keepalive 30       ! NAT keepalive interval
no crypto ikev2 nat-keepalive       ! disable NAT-T (not recommended)

! NAT-T detection:
!   IKE peers exchange NAT-D (NAT Discovery) payloads
!   Each peer hashes its IP:port and the peer's IP:port
!   If hash mismatch → NAT detected → switch to UDP 4500
!   Keepalives prevent NAT mapping timeout
```

### Anti-Replay Protection

```
! Anti-replay uses a sliding window of sequence numbers
! Default window: 64 packets

! Increase anti-replay window (high-bandwidth links)
crypto ipsec security-association replay window-size 512
crypto ipsec security-association replay window-size 1024

! Disable anti-replay (not recommended)
crypto ipsec security-association replay disable

! Anti-replay is per-SA:
!   Sender increments 64-bit sequence number per packet
!   Receiver maintains sliding window
!   Packets with sequence number outside/before window → dropped
!   Packets already seen within window → dropped (duplicate)
```

### PFS (Perfect Forward Secrecy)

```
! PFS performs new DH exchange for each Child SA (Phase 2)
! Compromise of IKE SA keys does not expose Child SA keys

! Enable PFS with DH group (IKEv1 crypto map)
crypto map MY_CMAP 10 ipsec-isakmp
  set pfs group14

! Enable PFS (IKEv2 IPsec profile)
crypto ipsec profile MY_PROF
  set pfs group19                   ! ECP-256 (NIST P-256)

! Common DH groups:
!   group 14:  2048-bit MODP (minimum recommended)
!   group 19:  256-bit ECP (NIST P-256, faster than MODP)
!   group 20:  384-bit ECP (NIST P-384)
!   group 21:  521-bit ECP (NIST P-521)
!   group 24:  2048-bit MODP with 256-bit POS (deprecated, Logjam-vulnerable)
```

## Troubleshooting

### Show Commands

```
! IKEv1 Phase 1
show crypto isakmp sa
  dst             src             state          conn-id  status
  203.0.113.1     198.51.100.1    QM_IDLE        1001     ACTIVE

! IKEv1 Phase 2
show crypto ipsec sa
  interface: GigabitEthernet0/0
    local  ident: 10.1.0.0/255.255.0.0
    remote ident: 10.2.0.0/255.255.0.0
    #pkts encaps: 15234   #pkts encrypt: 15234
    #pkts decaps: 14890   #pkts decrypt: 14890
    #pkts no sa : 0       #send errors:  0

! IKEv2
show crypto ikev2 sa
  Tunnel-id  Local           Remote          fvrf/ivrf   Status
  1          198.51.100.1/500 203.0.113.1/500 none/none   READY

show crypto ikev2 sa detailed
  ! Shows encryption, integrity, DH group, lifetime, bytes tx/rx

! IPsec SAs
show crypto ipsec sa detail
show crypto ipsec sa peer 203.0.113.1

! Tunnel interface status
show interface Tunnel0
show ip interface brief | include Tunnel

! Crypto engine
show crypto engine connections active
```

### Debug Commands

```
! IKEv2 debugging (most useful)
debug crypto ikev2
debug crypto ikev2 error
debug crypto ikev2 packet

! IKEv1 debugging
debug crypto isakmp
debug crypto ipsec

! IPsec debugging
debug crypto ipsec
debug crypto ipsec error

! Conditional debugging (recommended for production)
debug crypto condition peer ipv4 203.0.113.1
debug crypto ikev2
! ... run test ...
no debug all
```

### Common Issues

```
# IKE Phase 1 not establishing
#   - Crypto policy mismatch (encryption, hash, DH group, auth method)
#   - PSK mismatch (check for trailing spaces, encoding)
#   - ACL blocking UDP 500/4500 on transit path
#   - NAT without NAT-T enabled
#   - Identity mismatch (FQDN vs IP in aggressive mode)

# IPsec SA not creating (Phase 2 failure)
#   - Transform set mismatch (encryption + integrity)
#   - PFS group mismatch
#   - Proxy identity (ACL) mismatch — must be mirror image on both peers
#     Local: permit ip 10.1.0.0/16 10.2.0.0/16
#     Remote: permit ip 10.2.0.0/16 10.1.0.0/16

# Tunnel up but no traffic
#   - Routing: traffic not directed into tunnel interface
#   - Crypto map: "interesting traffic" ACL not matching
#   - MTU/fragmentation: oversized packets dropped
#   - Anti-replay drops: show crypto ipsec sa | include replay

# Tunnel flapping
#   - DPD too aggressive (increase interval/retries)
#   - Unstable underlying link
#   - Rekey failure: check SA lifetime alignment between peers
#   - NAT-T keepalive timeout

# Performance issues
#   - Software crypto (no hardware accelerator)
#   - show crypto engine configuration
#   - Consider AES-GCM (combined mode, single pass)
#   - Reduce overhead: use transport mode if appropriate
```

### Packet Capture for VPN Debugging

```
! Embedded packet capture (EPC)
monitor capture VPN_CAP interface GigabitEthernet0/0 both
monitor capture VPN_CAP match ipv4 protocol udp any any
monitor capture VPN_CAP start
! ... reproduce issue ...
monitor capture VPN_CAP stop
show monitor capture VPN_CAP buffer brief
show monitor capture VPN_CAP buffer dump

! ACL-based capture
ip access-list extended VPN_DEBUG
  permit udp host 198.51.100.1 host 203.0.113.1 eq 500
  permit udp host 198.51.100.1 host 203.0.113.1 eq 4500
  permit esp host 198.51.100.1 host 203.0.113.1
```

## See Also

- ipsec
- remote-access-vpn
- cryptography
- pki
- tls
- cisco-ftd

## References

- RFC 7296 — IKEv2 (Internet Key Exchange Protocol Version 2)
- RFC 2409 — IKEv1 (The Internet Key Exchange)
- RFC 4303 — ESP (IP Encapsulating Security Payload)
- RFC 4302 — AH (IP Authentication Header)
- RFC 3948 — UDP Encapsulation of IPsec ESP Packets (NAT-T)
- Cisco IOS-XE IPsec Configuration Guide
- Cisco DMVPN Design Guide
- Cisco FlexVPN Configuration Guide
