# Network Security Infrastructure (Defense-in-Depth for the Network Plane)

Hardening routers, switches, and firewalls across control, management, and data planes using infrastructure ACLs, uRPF, routing protocol authentication, CoPP, NTP/SNMP security, and device hardening checklists to reduce attack surface and protect network availability.

## Infrastructure ACLs (iACL)

### Concept

```
iACLs are applied at WAN/peering edges to filter traffic destined TO network
infrastructure addresses (loopbacks, link-nets, management subnets).

Goal: block the public Internet from reaching your router control/management plane
while allowing transit traffic to pass through.

                    ┌──────────────┐
  Internet ────────►│  Edge Router │────► Internal network
                    │  iACL here   │
                    └──────────────┘
                          │
             Permit: BGP peers (explicit IPs)
             Permit: ICMP unreachable/TTL-exceeded (for traceroute)
             Deny:   all other traffic TO infrastructure space
             Permit: all transit traffic (implicit, at end)
```

### IOS-XE iACL Template

```
ip access-list extended IACL-EDGE
 ! --- Permit BGP from known peers ---
 permit tcp host 203.0.113.1 host 198.51.100.1 eq bgp
 permit tcp host 203.0.113.1 eq bgp host 198.51.100.1
 ! --- Permit ICMP for traceroute/PMTUD ---
 permit icmp any any ttl-exceeded
 permit icmp any any port-unreachable
 permit icmp any any packet-too-big
 ! --- Permit OSPF/IS-IS from directly connected ---
 permit ospf any host 224.0.0.5
 permit ospf any host 224.0.0.6
 ! --- Deny all other traffic to infrastructure space ---
 deny ip any 198.51.100.0 0.0.0.255 log
 deny ip any 10.255.0.0 0.0.255.255 log
 ! --- Permit all transit traffic ---
 permit ip any any

interface GigabitEthernet0/0/0
 description UPSTREAM-ISP
 ip access-group IACL-EDGE in
```

### NX-OS iACL

```
ip access-list IACL-EDGE
 10 permit tcp host 203.0.113.1 host 198.51.100.1 eq bgp
 20 permit tcp host 203.0.113.1 eq bgp host 198.51.100.1
 30 permit icmp any any ttl-exceeded
 40 permit icmp any any port-unreachable
 50 deny ip any 198.51.100.0/24 log
 60 deny ip any 10.255.0.0/16 log
 70 permit ip any any

interface Ethernet1/1
 ip access-group IACL-EDGE in
```

## Management Plane Protection (MPP)

```
IOS-XE: Restrict which interfaces accept management protocols.
Only the dedicated management interface should accept SSH/SNMP/HTTP.

! Enable management-plane protection
management-plane
 host
  management-interface GigabitEthernet0/0 allow ssh snmp
  ! All other interfaces: management protocols silently dropped

! IOS-XR equivalent
control-plane
 management-plane
  inband
   interface all
    allow SSH peer
     address ipv4 10.0.0.0/8
```

## Control Plane Policing (CoPP)

```
CoPP rate-limits traffic destined to the router CPU.
See the dedicated CoPP cheatsheet for full configuration.

Quick reference — recommended rate limits:

Traffic Class         Rate (pps)    Burst       Action on Exceed
────────────────────────────────────────────────────────────────
BGP                   2000          500         drop
OSPF/IS-IS            5000          1000        drop
BFD                   10000         2000        drop
SSH/SNMP/NTP          500           200         drop
ICMP                  1000          250         drop
ARP                   2000          500         drop
All other             500           200         drop
```

## uRPF (Unicast Reverse Path Forwarding)

### Mode Comparison

```
Mode        Check                               Use Case                  Drops
──────────────────────────────────────────────────────────────────────────────────
Strict      Source IP must be reachable via       Single-homed links       Spoofed + asymmetric
            the SAME interface packet arrived
Loose       Source IP must exist in FIB           Multi-homed / asymmetric Spoofed (bogons only)
            (any interface)
Feasible    Source IP reachable via arriving      Multi-homed with         Spoofed, preserves
            interface OR any ECMP/backup path     backup paths             asymmetric
```

### IOS-XE Configuration

```
! Strict mode — single-homed customer-facing interface
interface GigabitEthernet0/0/1
 ip verify unicast source reachable-via rx
 ! 'rx' = strict mode

! Loose mode — multi-homed upstream interface
interface GigabitEthernet0/0/0
 ip verify unicast source reachable-via any
 ! 'any' = loose mode

! Feasible mode (IOS-XE 16.x+)
interface GigabitEthernet0/0/2
 ip verify unicast source reachable-via rx allow-self-ping

! Allow default route to satisfy uRPF check (loose mode common tweak)
ip verify unicast source reachable-via any allow-default

! uRPF with ACL for exceptions
ip access-list extended URPF-EXCEPTIONS
 permit ip host 0.0.0.0 any  ! DHCP discover
 permit ip host 169.254.0.0 0.0.255.255 any  ! link-local

interface GigabitEthernet0/0/1
 ip verify unicast source reachable-via rx URPF-EXCEPTIONS
```

### NX-OS uRPF

```
! Enable uRPF feature
feature urpf

interface Ethernet1/1
 ip verify unicast source reachable-via rx
```

### JunOS uRPF

```
set interfaces ge-0/0/0 unit 0 family inet rpf-check mode strict
set interfaces ge-0/0/0 unit 0 family inet rpf-check fail-filter URPF-FAIL

! Feasible-path mode
set interfaces ge-0/0/1 unit 0 family inet rpf-check mode loose
set routing-options forwarding-table unicast-reverse-path feasible-paths
```

## Infrastructure Anti-Spoofing

```
Defense-in-depth anti-spoofing stack:
  1. iACLs at edge:          deny packets FROM your own address space inbound
  2. uRPF on all interfaces: source address validation
  3. BCP38/BCP84:            RFC 2827 / RFC 3704 ingress filtering

! Edge anti-spoofing ACL (deny your own prefixes from outside)
ip access-list extended ANTI-SPOOF
 deny ip 198.51.100.0 0.0.0.255 any log      ! your prefix
 deny ip 10.0.0.0 0.255.255.255 any log       ! RFC 1918
 deny ip 172.16.0.0 0.15.255.255 any log      ! RFC 1918
 deny ip 192.168.0.0 0.0.255.255 any log      ! RFC 1918
 deny ip 127.0.0.0 0.255.255.255 any log      ! loopback
 deny ip 0.0.0.0 0.255.255.255 any log        ! unspecified
 deny ip 224.0.0.0 31.255.255.255 any log     ! multicast as source
 permit ip any any
```

## Routing Protocol Authentication

### BGP MD5 Authentication

```
! IOS-XE: Per-neighbor MD5
router bgp 65001
 neighbor 203.0.113.1 password 7 <encrypted-password>

! IOS-XR: Per-neighbor MD5
router bgp 65001
 neighbor 203.0.113.1
  password encrypted <hash>

! Verification
show tcp brief | include .179
show ip bgp neighbors 203.0.113.1 | include password
```

### BGP TCP-AO (RFC 5925)

```
! IOS-XE (17.6+): TCP-AO replaces MD5 — supports key rollover
key chain BGP-AO-KEY
 key 1
  key-string SuperSecret1
  send-lifetime 00:00:00 Jan 1 2025 00:00:00 Jul 1 2025
  accept-lifetime 00:00:00 Jan 1 2025 00:00:00 Aug 1 2025
  cryptographic-algorithm hmac-sha-256
 key 2
  key-string SuperSecret2
  send-lifetime 00:00:00 Jun 1 2025 infinite
  accept-lifetime 00:00:00 Jun 1 2025 infinite
  cryptographic-algorithm hmac-sha-256

tcp ao BGP-AO-POLICY
 key-chain BGP-AO-KEY
  mode send-receive

router bgp 65001
 neighbor 203.0.113.1 ao BGP-AO-POLICY

! IOS-XR: TCP-AO
key chain BGP-AO-KEY
 key 1
  key-string password SuperSecret1
  lifetime 00:00:00 january 1 2025 00:00:00 july 1 2025
  cryptographic-algorithm HMAC-SHA-256

router bgp 65001
 neighbor 203.0.113.1
  ao BGP-AO-POLICY include-tcp-options
```

### OSPF Authentication

```
! OSPF MD5 authentication (per-area)
router ospf 1
 area 0 authentication message-digest

interface GigabitEthernet0/0/0
 ip ospf message-digest-key 1 md5 OspfSecret123
 ip ospf authentication message-digest

! OSPFv3 IPsec authentication (IOS-XE)
interface GigabitEthernet0/0/0
 ospfv3 authentication ipsec spi 256 sha1 <40-hex-chars>

! OSPF authentication — JunOS
set protocols ospf area 0.0.0.0 interface ge-0/0/0.0 authentication md5 1 key OspfSecret123

! Verification
show ip ospf interface GigabitEthernet0/0/0 | include authentication
show ip ospf neighbor
```

### IS-IS Authentication

```
! IS-IS HMAC-MD5 authentication
key chain ISIS-KEY
 key 1
  key-string IsisSecret123

router isis CORE
 authentication mode md5 level-2
 authentication key-chain ISIS-KEY level-2

! Per-interface IS-IS authentication
interface GigabitEthernet0/0/0
 isis authentication mode md5 level-2
 isis authentication key-chain ISIS-KEY level-2

! JunOS IS-IS authentication
set protocols isis level 2 authentication-key IsisSecret123
set protocols isis level 2 authentication-type md5
set protocols isis interface ge-0/0/0.0 level 2 hello-authentication-key IsisSecret123
set protocols isis interface ge-0/0/0.0 level 2 hello-authentication-type md5
```

### EIGRP Authentication

```
! EIGRP SHA-256 authentication (named mode)
router eigrp SITE
 address-family ipv4 unicast autonomous-system 100
  af-interface GigabitEthernet0/0/0
   authentication mode hmac-sha-256 EigrpSecret123

! EIGRP MD5 authentication (classic mode)
key chain EIGRP-KEY
 key 1
  key-string EigrpSecret123

interface GigabitEthernet0/0/0
 ip authentication mode eigrp 100 md5
 ip authentication key-chain eigrp 100 EIGRP-KEY
```

## NTP Authentication

```
! IOS-XE: NTP MD5 authentication
ntp authenticate
ntp authentication-key 1 md5 NtpSecret123
ntp trusted-key 1
ntp server 10.0.0.100 key 1

! NTP access restrictions
ntp access-group peer 10          ! ACL 10 = trusted peers
ntp access-group serve-only 20    ! ACL 20 = allowed clients
ntp access-group query-only 30    ! ACL 30 = allowed queries
access-list 10 permit 10.0.0.100
access-list 20 permit 10.0.0.0 0.0.255.255
access-list 30 deny any

! JunOS NTP authentication
set system ntp authentication-key 1 type md5 value NtpSecret123
set system ntp trusted-key 1
set system ntp server 10.0.0.100 key 1

! Disable NTP monlist (amplification vector)
ntp disable               ! on interfaces not needing NTP
no ntp                    ! or globally if not used
```

## SNMP v3 Security

```
! Disable SNMP v1/v2c — use v3 only
no snmp-server community public
no snmp-server community private

! SNMP v3 with auth + encryption (authPriv)
snmp-server group SNMPV3-RO v3 priv read SNMP-VIEW
snmp-server group SNMPV3-RW v3 priv read SNMP-VIEW write SNMP-VIEW
snmp-server view SNMP-VIEW iso included

snmp-server user admin SNMPV3-RW v3 auth sha AuthPass123 priv aes 256 PrivPass456

! Restrict SNMP to management subnet
snmp-server host 10.0.0.50 version 3 priv admin
access-list 99 permit 10.0.0.0 0.0.0.255
snmp-server community RESTRICT RO 99  ! if v2c must coexist

! SNMP v3 security levels
Level         Auth    Encryption   Visibility
──────────────────────────────────────────────
noAuthNoPriv  None    None         Plaintext (avoid!)
authNoPriv    SHA/MD5 None         Authenticated but readable
authPriv      SHA/MD5 AES-256     Authenticated + encrypted (use this)

! JunOS SNMP v3
set snmp v3 usm local-engine user admin authentication-sha authentication-key AuthPass123
set snmp v3 usm local-engine user admin privacy-aes128 privacy-key PrivPass456
set snmp v3 vacm security-to-group security-model usm security-name admin group SNMPV3-RO
```

## Management Access Hardening

### VTY ACLs

```
! Restrict VTY (SSH) access to management subnet
ip access-list standard VTY-ACCESS
 permit 10.0.0.0 0.0.0.255
 deny any log

line vty 0 15
 access-class VTY-ACCESS in
 transport input ssh
 transport output none
 exec-timeout 15 0
 logging synchronous

! IPv6 VTY ACL
ipv6 access-list VTY-ACCESS-V6
 permit ipv6 2001:db8:a::/48 any
 deny ipv6 any any log

line vty 0 15
 ipv6 access-class VTY-ACCESS-V6 in
```

### SSH Hardening

```
! SSH v2 only, strong crypto
ip ssh version 2
ip ssh time-out 60
ip ssh authentication-retries 3
ip ssh source-interface Loopback0

! SSH RSA key (2048+ bits)
crypto key generate rsa modulus 4096 label SSH-KEY

! Disable Telnet globally
no service telnet

! SSH rate limiting (CoPP complements this)
ip ssh maxstartups 5

! JunOS SSH hardening
set system services ssh protocol-version v2
set system services ssh root-login deny
set system services ssh rate-limit 5
set system services ssh connection-limit 10
set system services ssh no-tcp-forwarding
set system login retry-options tries-before-disconnect 3
```

### Disable Unnecessary Services

```
! IOS-XE — disable unused services
no service finger
no service pad
no service udp-small-servers
no service tcp-small-servers
no ip http server          ! disable HTTP — use HTTPS only
ip http secure-server
no ip http secure-active-session-modules none
no cdp run                 ! disable CDP on untrusted interfaces
no ip source-route
no ip gratuitous-arps
no service dhcp            ! unless DHCP relay is needed
no ip bootp server
no mop enabled
no ip domain-lookup        ! on VTY to prevent DNS-delay on typos

! Per-interface hardening
interface GigabitEthernet0/0/0
 no ip redirects
 no ip unreachables
 no ip proxy-arp
 no ip directed-broadcast
 no cdp enable
 no mop enabled
 no mop sysid
```

## Logging Security

```
! Secure syslog configuration
logging buffered 64000 informational
logging console critical
logging monitor warnings

! Syslog to central server (encrypted if possible)
logging host 10.0.0.200 transport tcp port 6514
logging source-interface Loopback0
logging trap informational
logging origin-id hostname

! Timestamps for forensics
service timestamps log datetime msec localtime show-timezone year
service timestamps debug datetime msec localtime show-timezone year

! Log failed login attempts
login on-failure log
login on-success log

! AAA accounting (who did what)
aaa accounting exec default start-stop group tacacs+
aaa accounting commands 15 default start-stop group tacacs+

! Archive configuration changes
archive
 log config
  logging enable
  logging size 500
  notify syslog contenttype plaintext
  hidekeys
```

## Infrastructure Device Hardening Checklist

```
Category                    Item                                          Status
─────────────────────────────────────────────────────────────────────────────────
Management Plane
  [ ] SSH v2 only, RSA 4096-bit key
  [ ] Telnet disabled (no service telnet)
  [ ] VTY ACLs restrict access to management subnet
  [ ] SNMP v3 authPriv (SHA + AES-256)
  [ ] SNMP v1/v2c communities removed
  [ ] NTP authenticated, restricted access groups
  [ ] HTTP disabled, HTTPS enabled with valid cert
  [ ] TACACS+/RADIUS for AAA (no local-only auth)
  [ ] Exec timeout on VTY/console (10-15 min)
  [ ] Encrypted passwords (service password-encryption)
  [ ] Enable secret (not enable password)
  [ ] Management-plane protection enabled (MPP)
  [ ] Banner with legal notice (no hostname disclosure)

Control Plane
  [ ] CoPP deployed with per-protocol policers
  [ ] Routing protocol authentication (MD5/SHA/TCP-AO)
  [ ] BGP TTL security (GTSM) for eBGP peers
  [ ] BGP max-prefix limits per neighbor
  [ ] RPKI/ROV enabled for BGP prefix validation
  [ ] BFD timers tuned (not sub-second unless needed)
  [ ] IS-IS authentication on all interfaces
  [ ] OSPF passive-interface default + selective activation

Data Plane
  [ ] iACLs on all external-facing interfaces
  [ ] uRPF strict on single-homed, loose on multi-homed
  [ ] Anti-spoofing ACLs at edge (BCP38/BCP84)
  [ ] No IP source-route
  [ ] No IP directed-broadcast
  [ ] No IP proxy-arp on untrusted interfaces
  [ ] No IP redirects on untrusted interfaces
  [ ] Storm control on access ports

Logging & Monitoring
  [ ] Centralized syslog (TLS encrypted)
  [ ] Logging source = Loopback0
  [ ] Timestamps with msec precision
  [ ] Configuration change logging (archive)
  [ ] AAA accounting for exec and commands
  [ ] Login failure logging enabled
  [ ] SNMP traps to NMS for interface/BGP/env

Physical & Software
  [ ] Latest stable firmware / IOS version
  [ ] Unused interfaces shut down
  [ ] Console password set with exec-timeout
  [ ] Auxiliary port disabled (no exec)
  [ ] Boot image verification (secure boot)
```

## BGP TTL Security (GTSM)

```
! Generalized TTL Security Mechanism (RFC 5082)
! Drops BGP packets with TTL < 254 — attacker must be directly connected
router bgp 65001
 neighbor 203.0.113.1 ttl-security hops 1

! JunOS GTSM
set protocols bgp group EBGP-PEERS neighbor 203.0.113.1 ttl 255
set protocols bgp group EBGP-PEERS neighbor 203.0.113.1 multihop ttl 1

! Cannot use ttl-security and ebgp-multihop simultaneously
! GTSM is strongly recommended for all eBGP sessions
```

## BGP Max-Prefix Protection

```
! Prevent route table explosion from misconfigured peer
router bgp 65001
 address-family ipv4 unicast
  neighbor 203.0.113.1 maximum-prefix 10000 80 restart 30
  ! 10000 = max prefixes, 80 = warning at 80%, restart = retry in 30 min

! JunOS max-prefix
set protocols bgp group EBGP neighbor 203.0.113.1 family inet unicast prefix-limit maximum 10000
set protocols bgp group EBGP neighbor 203.0.113.1 family inet unicast prefix-limit teardown 80 idle-timeout 30
```

## Verification Commands

```
! --- iACL / ACL verification ---
show access-lists IACL-EDGE
show ip access-lists interface GigabitEthernet0/0/0

! --- uRPF verification ---
show ip interface GigabitEthernet0/0/1 | include verify
show ip traffic | include RPF
show cef interface GigabitEthernet0/0/1 | include RPF

! --- Routing protocol auth verification ---
show ip ospf interface | include authentication
show ip bgp neighbors | include password
show isis adjacency detail | include Authentication

! --- NTP verification ---
show ntp associations detail
show ntp status
show ntp packets

! --- SNMP verification ---
show snmp user
show snmp group
show snmp engineID

! --- SSH/VTY verification ---
show ip ssh
show line vty 0 15 | include access-class
show users

! --- CoPP verification ---
show policy-map control-plane
show platform software infrastructure punt statistics

! --- Logging verification ---
show logging | include config
show archive log config all
```

## Tips

- Deploy infrastructure ACLs before CoPP. iACLs drop unwanted traffic in hardware at line rate, while CoPP policers consume CPU cycles for classification. The two are complementary: iACLs are the outer wall, CoPP is the inner gate.
- Use TCP-AO instead of MD5 for BGP authentication wherever supported. MD5 has no key rollover mechanism, meaning key changes require simultaneous configuration on both peers and cause session flaps. TCP-AO supports overlapping key lifetimes for hitless rotation.
- Always enable GTSM (TTL security) on eBGP sessions. Without it, an attacker anywhere on the Internet can attempt TCP RST attacks against your BGP sessions. GTSM ensures only directly-connected peers (TTL=255) can establish sessions.
- Apply uRPF strict mode on single-homed customer-facing interfaces and loose mode on upstream/peering interfaces. Strict mode on asymmetric paths drops legitimate traffic because the return path uses a different interface.
- Remove all SNMP v1/v2c community strings before enabling v3. A forgotten "public" community string is the single most common infrastructure vulnerability found in penetration tests.
- Set NTP authentication on all peers and restrict NTP service with access groups. NTP amplification attacks (monlist) have generated 400+ Gbps DDoS floods from misconfigured NTP servers.
- Log all configuration changes with AAA accounting and archive logging. In incident response, the first question is always "what changed?" and without logging the answer is "we do not know."
- Disable IP source-route, IP redirects, IP proxy-arp, and IP directed-broadcast on every external-facing interface. These legacy features are used in MITM, ICMP redirect, and smurf amplification attacks.
- Use Loopback0 as the source interface for all management traffic (syslog, SNMP, NTP, TACACS+). This ensures management sessions survive individual link failures and simplifies ACL management.
- Test hardening changes in a maintenance window. A typo in a VTY ACL can lock you out of the device permanently if console access is not available.

## See Also

- copp, acl, bgp, ospf, is-is, eigrp, snmp, ntp, ssh, network-defense, cis-benchmarks, zero-trust, rpki

## References

- [RFC 2827 — Network Ingress Filtering (BCP38)](https://datatracker.ietf.org/doc/html/rfc2827)
- [RFC 3704 — Ingress Filtering for Multihomed Networks (BCP84)](https://datatracker.ietf.org/doc/html/rfc3704)
- [RFC 5082 — GTSM (TTL Security)](https://datatracker.ietf.org/doc/html/rfc5082)
- [RFC 5925 — TCP Authentication Option (TCP-AO)](https://datatracker.ietf.org/doc/html/rfc5925)
- [RFC 6192 — Protecting the Router Control Plane](https://datatracker.ietf.org/doc/html/rfc6192)
- [NIST SP 800-189 — Resilient Interdomain Traffic Exchange](https://csrc.nist.gov/publications/detail/sp/800-189/final)
- [NSA Network Infrastructure Security Guide](https://media.defense.gov/2022/Jun/15/2003018261/-1/-1/0/CTR_NSA_NETWORK_INFRASTRUCTURE_SECURITY_GUIDE_20220615.PDF)
- [CIS Cisco IOS Benchmarks](https://www.cisecurity.org/benchmark/cisco)
- [Cisco Infrastructure Protection ACLs Guide](https://www.cisco.com/c/en/us/support/docs/ip/access-lists/43920-iacl.html)
