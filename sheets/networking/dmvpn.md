# DMVPN (Dynamic Multipoint VPN)

Cisco's Dynamic Multipoint VPN combines mGRE tunnels, NHRP address resolution, IPsec encryption, and a routing protocol to build scalable hub-and-spoke or spoke-to-spoke overlay networks over any IP transport without requiring a full mesh of static tunnels.

## Core Components

```
Component        Role                                    Protocol/Standard
─────────────────────────────────────────────────────────────────────────────
mGRE             Multipoint GRE tunnel interface          RFC 2784 / RFC 7676
NHRP             Maps tunnel IPs to NBMA (public) IPs     RFC 2332
IPsec            Encrypts GRE encapsulated traffic        ESP (RFC 4303)
Routing Protocol Distributes overlay routes (EIGRP/OSPF/BGP) Various
```

### How They Fit Together

```
Spoke A (192.168.1.0/24)         Hub (10.0.0.0/24)         Spoke B (172.16.1.0/24)
  Tunnel0: 10.255.0.2            Tunnel0: 10.255.0.1        Tunnel0: 10.255.0.3
  NBMA:    203.0.113.10          NBMA:    198.51.100.1      NBMA:    203.0.113.20
     |                              |                          |
     +--- mGRE ---> NHRP reg ------>|                          |
     |              IPsec SA ------>|                          |
     |                              |<------ NHRP reg --------+
     |                              |<------ IPsec SA --------+
     |                              |                          |
     +--- EIGRP neighbor ---------->|<------- EIGRP neighbor --+
     |         (routing exchange)   |    (routing exchange)    |
```

## DMVPN Phases

### Phase 1 --- Hub-and-Spoke Only

```
! Hub configuration
interface Tunnel0
 ip address 10.255.0.1 255.255.255.0
 tunnel source GigabitEthernet0/0
 tunnel mode gre multipoint
 ip nhrp network-id 1
 ip nhrp map multicast dynamic
 ip nhrp authentication DMVPNKEY
 ip nhrp redirect                         ! Not used in Phase 1

! Spoke configuration
interface Tunnel0
 ip address 10.255.0.2 255.255.255.0
 tunnel source GigabitEthernet0/0
 tunnel mode gre multipoint
 tunnel destination 198.51.100.1          ! Phase 1: static tunnel dest to hub
 ip nhrp network-id 1
 ip nhrp nhs 10.255.0.1
 ip nhrp map 10.255.0.1 198.51.100.1
 ip nhrp map multicast 198.51.100.1
 ip nhrp authentication DMVPNKEY
```

**Behavior:** All traffic between spokes transits through the hub. Spokes have a static tunnel destination (or p2p GRE) pointing at the hub. No spoke-to-spoke tunnels.

**Use case:** Small deployments, regulatory requirements mandating traffic inspection at a central site.

### Phase 2 --- Spoke-to-Spoke (Partial Mesh)

```
! Hub configuration (same as Phase 1 except no tunnel destination)
interface Tunnel0
 ip address 10.255.0.1 255.255.255.0
 tunnel source GigabitEthernet0/0
 tunnel mode gre multipoint
 ip nhrp network-id 1
 ip nhrp map multicast dynamic
 ip nhrp authentication DMVPNKEY
 ! Routing: must NOT change next-hop
 ! EIGRP: no ip next-hop-self eigrp <AS>
 ! OSPF:  use network type broadcast, hub is DR

! Spoke configuration
interface Tunnel0
 ip address 10.255.0.2 255.255.255.0
 tunnel source GigabitEthernet0/0
 tunnel mode gre multipoint              ! mGRE on spoke (required for Phase 2+)
 ip nhrp network-id 1
 ip nhrp nhs 10.255.0.1
 ip nhrp map 10.255.0.1 198.51.100.1
 ip nhrp map multicast 198.51.100.1
 ip nhrp authentication DMVPNKEY
```

**Behavior:** Spoke A sends traffic to Spoke B via hub initially. NHRP resolution request is triggered because the routing next-hop is the remote spoke (not the hub). After resolution, a direct spoke-to-spoke tunnel forms.

**Key requirement:** Routing must preserve the original spoke next-hop address. If the hub rewrites next-hop to itself, NHRP resolution never triggers and Phase 2 collapses back to Phase 1.

### Phase 3 --- Spoke-to-Spoke with NHRP Shortcuts

```
! Hub configuration
interface Tunnel0
 ip address 10.255.0.1 255.255.255.0
 tunnel source GigabitEthernet0/0
 tunnel mode gre multipoint
 ip nhrp network-id 1
 ip nhrp map multicast dynamic
 ip nhrp authentication DMVPNKEY
 ip nhrp redirect                         ! Phase 3: tell spoke a better path exists

! Spoke configuration
interface Tunnel0
 ip address 10.255.0.2 255.255.255.0
 tunnel source GigabitEthernet0/0
 tunnel mode gre multipoint
 ip nhrp network-id 1
 ip nhrp nhs 10.255.0.1
 ip nhrp map 10.255.0.1 198.51.100.1
 ip nhrp map multicast 198.51.100.1
 ip nhrp authentication DMVPNKEY
 ip nhrp shortcut                         ! Phase 3: install NHRP shortcut routes
```

**Behavior:** Hub can set next-hop to itself (EIGRP default). When the hub forwards spoke-to-spoke traffic, it sends an NHRP Redirect to the source spoke. The source spoke sends an NHRP Resolution Request to the destination spoke. On resolution, a shortcut route is installed in the spoke's CEF table, bypassing the hub for subsequent packets.

**Advantages over Phase 2:**
- No routing constraints (hub can be next-hop-self)
- Summarization at the hub is supported
- Default routes on spokes work for spoke-to-spoke
- More scalable routing tables on spokes

## Phase Comparison

```
Feature                  Phase 1           Phase 2           Phase 3
──────────────────────────────────────────────────────────────────────────
Spoke-to-spoke direct    No                Yes               Yes
Spoke tunnel mode        p2p GRE or mGRE   mGRE              mGRE
Hub next-hop-self        Yes (default)     No (breaks it)    Yes (fine)
Summarization at hub     Yes               No (breaks it)    Yes
Default route on spoke   Yes               No (breaks it)    Yes
NHRP redirect on hub     N/A               N/A               Required
NHRP shortcut on spoke   N/A               N/A               Required
Spoke routing table      Full/summary      Full (required)   Summary/default OK
Trigger for direct path  N/A               Routing next-hop  NHRP redirect
Scalability              High              Medium            High
```

## NHRP (Next Hop Resolution Protocol)

### NHRP Message Types

```
Type                 Code   Direction         Purpose
────────────────────────────────────────────────────────────────
Registration Request   1    Spoke -> Hub      Register NBMA mapping
Registration Reply     2    Hub -> Spoke      Confirm registration
Resolution Request     3    Spoke -> Hub/Spoke Resolve tunnel-to-NBMA mapping
Resolution Reply       4    Hub/Spoke -> Spoke Return NBMA address
Purge Request          5    Any -> Any        Remove stale mapping
Purge Reply            6    Any -> Any        Confirm purge
Error Indication       7    Any -> Any        Report error
Redirect               -    Hub -> Spoke      Phase 3: signal shortcut available
```

### NHRP Registration Process

```
Spoke boots up
  |
  v
Spoke sends NHRP Registration Request to NHS
  - Tunnel IP: 10.255.0.2
  - NBMA IP:   203.0.113.10
  - Hold time:  600 seconds (default)
  |
  v
Hub (NHS) receives, adds to NHRP cache
  |
  v
Hub sends Registration Reply (success)
  |
  v
Spoke sets re-registration timer (hold_time / 3 = 200s)
```

### NHRP Resolution (Phase 2)

```
Spoke A wants to reach 172.16.1.0/24 (behind Spoke B)
  |
  v
Routing table says next-hop = 10.255.0.3 (Spoke B tunnel IP)
  |
  v
Spoke A checks NHRP cache for 10.255.0.3 -> no entry
  |
  v
Spoke A sends NHRP Resolution Request to NHS (hub)
  |
  v
Hub looks up 10.255.0.3 in NHRP cache -> NBMA 203.0.113.20
  |
  v
Hub forwards Resolution Request to Spoke B (or replies directly)
  |
  v
Spoke B replies with Resolution Reply
  - 10.255.0.3 -> 203.0.113.20
  |
  v
Spoke A installs NHRP mapping, IPsec SA builds spoke-to-spoke
  |
  v
Direct tunnel established (bypasses hub)
```

### NHRP Shortcut (Phase 3)

```
Spoke A sends packet to Spoke B (via hub, next-hop = hub)
  |
  v
Hub forwards packet AND sends NHRP Redirect to Spoke A
  "There's a shorter path to 10.255.0.3"
  |
  v
Spoke A sends NHRP Resolution Request to Spoke B (via hub)
  |
  v
Spoke B sends Resolution Reply directly to Spoke A
  |
  v
Spoke A installs shortcut route in CEF:
  172.16.1.0/24 -> next-hop 10.255.0.3 via Tunnel0 (NBMA 203.0.113.20)
  |
  v
Subsequent packets go directly spoke-to-spoke
  (shortcut expires when NHRP hold timer runs out if no traffic)
```

## IPsec Integration

### IPsec Profile (Preferred for DMVPN)

```
! Crypto ISAKMP policy (IKEv1) or IKEv2 proposal
crypto isakmp policy 10
 encryption aes 256
 hash sha256
 authentication pre-share
 group 14
 lifetime 86400

crypto isakmp key SUPERSECRETKEY address 0.0.0.0

! IPsec transform set
crypto ipsec transform-set DMVPN-TS esp-aes 256 esp-sha256-hmac
 mode transport                           ! Transport mode (GRE adds outer header)

! IPsec profile (no ACL needed -- triggered by tunnel)
crypto ipsec profile DMVPN-PROFILE
 set transform-set DMVPN-TS

! Apply to tunnel interface
interface Tunnel0
 tunnel protection ipsec profile DMVPN-PROFILE
```

### IKEv2 Configuration (Recommended)

```
crypto ikev2 proposal DMVPN-PROP
 encryption aes-cbc-256
 integrity sha256
 group 14

crypto ikev2 policy DMVPN-POL
 proposal DMVPN-PROP

crypto ikev2 keyring DMVPN-KR
 peer ANY
  address 0.0.0.0 0.0.0.0
  pre-shared-key SUPERSECRETKEY

crypto ikev2 profile DMVPN-IKEV2
 match identity remote address 0.0.0.0
 authentication remote pre-share
 authentication local pre-share
 keyring local DMVPN-KR

crypto ipsec transform-set DMVPN-TS esp-gcm 256
 mode transport

crypto ipsec profile DMVPN-PROFILE
 set transform-set DMVPN-TS
 set ikev2-profile DMVPN-IKEV2
```

### Transport vs Tunnel Mode

```
Mode        Overhead    Use Case                      Header Structure
──────────────────────────────────────────────────────────────────────────────
Transport   Lower       DMVPN (GRE already adds IP)   [IP][ESP][GRE][IP][Payload]
Tunnel      Higher      Standalone IPsec VPNs         [IP][ESP][IP][Payload]

DMVPN uses transport mode because mGRE already encapsulates with an
outer IP header. Using tunnel mode would double-encapsulate unnecessarily.
```

## Routing Over DMVPN

### EIGRP

```
! Hub
router eigrp 100
 network 10.255.0.0 0.0.0.255
 network 10.0.0.0 0.0.255.255

interface Tunnel0
 ! Phase 2: preserve spoke next-hop
 no ip next-hop-self eigrp 100
 no ip split-horizon eigrp 100           ! Allow spoke routes to be re-advertised

 ! Phase 3: next-hop-self is fine (default)
 ! ip summary-address eigrp 100 0.0.0.0 0.0.0.0  ! Optional: default to spokes
```

**EIGRP Split-Horizon:** Disabled on hub tunnel interface. Without this, routes learned from one spoke are not advertised to other spokes (all on the same interface).

**EIGRP Next-Hop-Self:** Phase 2 must disable this. Phase 3 can leave it enabled.

### OSPF

```
! Hub
router ospf 1
 network 10.255.0.0 0.0.0.255 area 0
 network 10.0.0.0 0.0.255.255 area 0

interface Tunnel0
 ip ospf network broadcast                ! Hub is DR (highest priority)
 ip ospf priority 100

! Spoke
interface Tunnel0
 ip ospf network broadcast
 ip ospf priority 0                       ! Never become DR/BDR
```

**OSPF Network Type Considerations:**

```
Network Type        DR/BDR    Next-Hop Behavior     Phase 2 OK    Phase 3 OK
──────────────────────────────────────────────────────────────────────────────
point-to-multipoint No        Preserves next-hop    Yes           Yes
broadcast           Yes       DR rewrites next-hop  Needs tuning  Yes
point-to-point      No        N/A (single neighbor) Phase 1 only  No
```

- **broadcast:** Most common for Phase 3. Hub must be DR. Spokes set priority 0.
- **point-to-multipoint:** Works for Phase 2 (preserves next-hop). Higher LSA overhead.

### BGP

```
! Hub (route reflector)
router bgp 65000
 neighbor DMVPN-SPOKES peer-group
 neighbor DMVPN-SPOKES remote-as 65000
 neighbor DMVPN-SPOKES route-reflector-client
 neighbor DMVPN-SPOKES next-hop-self        ! Phase 3 only
 neighbor 10.255.0.2 peer-group DMVPN-SPOKES
 neighbor 10.255.0.3 peer-group DMVPN-SPOKES

! Spoke
router bgp 65000
 neighbor 10.255.0.1 remote-as 65000
```

**BGP Considerations:**
- iBGP between hub and spokes (hub as route reflector)
- Phase 2: hub must NOT set next-hop-self
- Phase 3: hub CAN set next-hop-self (NHRP shortcuts handle direct paths)
- eBGP possible between autonomous spokes but less common
- BGP scales better than EIGRP/OSPF for very large DMVPN (hundreds of spokes)

## Front-Door VRF (FVRF)

FVRF separates the tunnel transport (underlay) from the tunnel overlay by placing the physical interface in a VRF while the tunnel interface remains in the global routing table (or another VRF).

```
! Define the transport VRF
ip vrf TRANSPORT
 rd 100:1

! Physical interface in FVRF
interface GigabitEthernet0/0
 ip vrf forwarding TRANSPORT
 ip address 203.0.113.10 255.255.255.0

! Default route in transport VRF
ip route vrf TRANSPORT 0.0.0.0 0.0.0.0 203.0.113.1

! Tunnel interface references FVRF for transport
interface Tunnel0
 ip address 10.255.0.2 255.255.255.0
 tunnel source GigabitEthernet0/0
 tunnel mode gre multipoint
 tunnel vrf TRANSPORT                     ! Key: tunnel transport uses FVRF
 ip nhrp network-id 1
 ip nhrp nhs 10.255.0.1 nbma 198.51.100.1 multicast
 ip nhrp authentication DMVPNKEY
 tunnel protection ipsec profile DMVPN-PROFILE

! NHRP mapping with FVRF
 ip nhrp map 10.255.0.1 198.51.100.1      ! NBMA addr resolved in TRANSPORT VRF
```

**Why FVRF:**
- Prevents recursive routing (tunnel source resolved through the tunnel)
- Isolates transport routing from overlay routing
- Mandatory when the ISP-facing route and the overlay default route would conflict
- Eliminates the need for static routes to NHS NBMA addresses in the global table

## Dual-Hub Redundancy

### Dual-Hub Single-Cloud

```
                    Hub-1 (10.255.0.1)
                   / NBMA: 198.51.100.1
Spoke A --------- +
  10.255.0.10      \
                    Hub-2 (10.255.0.2)
                    NBMA: 198.51.100.2

! Spoke configuration (two NHS entries, one tunnel)
interface Tunnel0
 ip address 10.255.0.10 255.255.255.0
 tunnel source GigabitEthernet0/0
 tunnel mode gre multipoint
 ip nhrp nhs 10.255.0.1 nbma 198.51.100.1 multicast
 ip nhrp nhs 10.255.0.2 nbma 198.51.100.2 multicast
 ip nhrp authentication DMVPNKEY
 tunnel protection ipsec profile DMVPN-PROFILE
```

### Dual-Hub Dual-Cloud

```
                    Hub-1 (Tunnel0: 10.255.0.1)
                    NBMA: 198.51.100.1
Spoke -----Tunnel0-+
  Tunnel0: 10.255.0.10
  Tunnel1: 10.255.1.10
Spoke -----Tunnel1-+
                    Hub-2 (Tunnel1: 10.255.1.1)
                    NBMA: 198.51.100.2

! Spoke configuration (two tunnels, two clouds)
interface Tunnel0
 ip address 10.255.0.10 255.255.255.0
 tunnel source GigabitEthernet0/0
 tunnel mode gre multipoint
 tunnel key 100
 ip nhrp network-id 1
 ip nhrp nhs 10.255.0.1 nbma 198.51.100.1 multicast
 ip nhrp authentication DMVPNKEY1
 tunnel protection ipsec profile DMVPN-PROFILE1

interface Tunnel1
 ip address 10.255.1.10 255.255.255.0
 tunnel source GigabitEthernet0/0
 tunnel mode gre multipoint
 tunnel key 200
 ip nhrp network-id 2
 ip nhrp nhs 10.255.1.1 nbma 198.51.100.2 multicast
 ip nhrp authentication DMVPNKEY2
 tunnel protection ipsec profile DMVPN-PROFILE2
```

**Tunnel Key:** Required when multiple tunnels share the same source interface. The key differentiates which tunnel receives inbound GRE packets.

**Failover:** Routing protocol metrics determine primary/backup path. EIGRP delay or OSPF cost tuning on tunnel interfaces controls preference.

## Per-Tunnel QoS

Per-tunnel QoS allows the hub to apply individual QoS policies to each spoke tunnel, enforcing bandwidth limits and traffic prioritization per site.

```
! Define QoS policy
policy-map SPOKE-QOS
 class VOICE
  priority 256
 class VIDEO
  bandwidth 512
 class CRITICAL
  bandwidth 256
 class class-default
  fair-queue

! NHRP group on hub tunnel
interface Tunnel0
 ip nhrp group SPOKE-BRANCH

! Per-tunnel QoS template
ip nhrp map group SPOKE-BRANCH service-policy output SPOKE-QOS

! Spoke registers with NHRP group
interface Tunnel0
 ip nhrp group SPOKE-BRANCH
```

**How it works:**
- Hub creates a virtual interface per spoke when the NHRP registration arrives
- The service-policy is instantiated per spoke tunnel, not once for the whole Tunnel0
- Each spoke gets its own set of queues and bandwidth allocations
- The hub enforces the rate based on the spoke's NBMA mapping

## Troubleshooting

### Essential Show Commands

```
! DMVPN tunnel status
show dmvpn
show dmvpn detail

! NHRP mappings and registrations
show ip nhrp
show ip nhrp brief
show ip nhrp multicast
show ip nhrp nhs detail
show ip nhrp traffic

! IPsec tunnel status
show crypto session
show crypto session detail
show crypto isakmp sa
show crypto ipsec sa
show crypto ikev2 sa

! Tunnel interface
show interface Tunnel0
show ip interface brief | include Tunnel

! Routing protocol neighbors over DMVPN
show ip eigrp neighbors
show ip ospf neighbor
show ip bgp summary
```

### Common Issues and Fixes

```
Symptom                           Likely Cause                    Fix
──────────────────────────────────────────────────────────────────────────────────────
NHRP registration fails           Wrong NHS IP or NBMA mapping    Verify ip nhrp nhs / map
                                  Authentication mismatch         Check ip nhrp authentication
                                  Firewall blocking GRE/NHRP     Allow IP protocol 47, UDP 4500

Tunnel up, no NHRP entries        network-id mismatch             Same network-id on all nodes
                                  MTU/fragmentation issues        Lower tunnel MTU (ip mtu 1400)

Phase 2 no spoke-to-spoke         Hub using next-hop-self         Disable: no ip next-hop-self
                                  Split-horizon on hub            Disable: no ip split-horizon
                                  Spoke not using mGRE            Change to tunnel mode gre multipoint

Phase 3 no shortcuts              Missing ip nhrp redirect (hub)  Add to hub tunnel interface
                                  Missing ip nhrp shortcut (spoke) Add to spoke tunnel interface
                                  CEF not installed shortcut      Check: show ip cef <prefix>

IPsec SA not forming              Pre-shared key mismatch         Verify keys on both ends
                                  Transform set mismatch          Match encryption/integrity algos
                                  NAT between peers               Enable NAT-T (UDP 4500)
                                  Clock skew (certificates)       Sync NTP

Routing neighbors not forming     Tunnel interface down           Check NHRP registration first
                                  ACL blocking routing protocol   Permit protocol traffic on tunnel
                                  OSPF network type mismatch      Match on all routers
                                  EIGRP AS mismatch               Same AS number everywhere

Intermittent spoke-to-spoke       NHRP hold timer too short       Increase: ip nhrp holdtime 600
                                  IPsec SA lifetime mismatch      Match lifetime on all peers
                                  Underlying path flapping        Check ISP link stability
```

### Debug Commands (Use with Caution)

```
debug dmvpn condition peer nbma <ip>      ! Filter debug to one peer
debug dmvpn all
debug nhrp
debug nhrp cache
debug nhrp packet
debug crypto isakmp
debug crypto ipsec
```

### MTU and Fragmentation

```
! Calculate tunnel MTU
Physical MTU:                1500 bytes
- GRE header:                  24 bytes (4 base + 4 key + 16 if using sequence)
- IPsec ESP (AES-256/SHA-256): ~73 bytes (SPI:4 + Seq:4 + IV:16 + Pad:~14 + Auth:32 + ESP-Hdr:2)
                              ≈ 1400 bytes safe tunnel MTU

! Set on tunnel interface
interface Tunnel0
 ip mtu 1400
 ip tcp adjust-mss 1360                   ! MSS = MTU - 40 (IP+TCP headers)

! Verify
ping 10.255.0.3 size 1400 df-bit source Tunnel0
```

## Quick Reference --- Full Hub Configuration (Phase 3)

```
! === ISAKMP / IKEv2 ===
crypto ikev2 proposal DMVPN-PROP
 encryption aes-cbc-256
 integrity sha256
 group 14
crypto ikev2 policy DMVPN-POL
 proposal DMVPN-PROP
crypto ikev2 keyring DMVPN-KR
 peer ANY
  address 0.0.0.0 0.0.0.0
  pre-shared-key YOURPSK
crypto ikev2 profile DMVPN-IKEV2
 match identity remote address 0.0.0.0
 authentication remote pre-share
 authentication local pre-share
 keyring local DMVPN-KR

! === IPsec ===
crypto ipsec transform-set DMVPN-TS esp-gcm 256
 mode transport
crypto ipsec profile DMVPN-PROFILE
 set transform-set DMVPN-TS
 set ikev2-profile DMVPN-IKEV2

! === Tunnel Interface ===
interface Tunnel0
 ip address 10.255.0.1 255.255.255.0
 tunnel source GigabitEthernet0/0
 tunnel mode gre multipoint
 ip nhrp network-id 1
 ip nhrp map multicast dynamic
 ip nhrp authentication DMVPNKEY
 ip nhrp redirect
 ip mtu 1400
 ip tcp adjust-mss 1360
 tunnel protection ipsec profile DMVPN-PROFILE

! === Routing (EIGRP) ===
router eigrp 100
 network 10.255.0.0 0.0.0.255
 network 10.0.0.0 0.0.255.255

! === Routing (OSPF alternative) ===
! router ospf 1
!  network 10.255.0.0 0.0.0.255 area 0
! interface Tunnel0
!  ip ospf network broadcast
!  ip ospf priority 100
```

## Quick Reference --- Full Spoke Configuration (Phase 3)

```
! === IKEv2 (same as hub) ===
crypto ikev2 proposal DMVPN-PROP
 encryption aes-cbc-256
 integrity sha256
 group 14
crypto ikev2 policy DMVPN-POL
 proposal DMVPN-PROP
crypto ikev2 keyring DMVPN-KR
 peer ANY
  address 0.0.0.0 0.0.0.0
  pre-shared-key YOURPSK
crypto ikev2 profile DMVPN-IKEV2
 match identity remote address 0.0.0.0
 authentication remote pre-share
 authentication local pre-share
 keyring local DMVPN-KR

! === IPsec ===
crypto ipsec transform-set DMVPN-TS esp-gcm 256
 mode transport
crypto ipsec profile DMVPN-PROFILE
 set transform-set DMVPN-TS
 set ikev2-profile DMVPN-IKEV2

! === Tunnel Interface ===
interface Tunnel0
 ip address 10.255.0.2 255.255.255.0
 tunnel source GigabitEthernet0/0
 tunnel mode gre multipoint
 ip nhrp network-id 1
 ip nhrp nhs 10.255.0.1 nbma 198.51.100.1 multicast
 ip nhrp authentication DMVPNKEY
 ip nhrp shortcut
 ip mtu 1400
 ip tcp adjust-mss 1360
 tunnel protection ipsec profile DMVPN-PROFILE

! === Routing (EIGRP) ===
router eigrp 100
 network 10.255.0.0 0.0.0.255
 network 192.168.1.0 0.0.0.255

! === Routing (OSPF alternative) ===
! router ospf 1
!  network 10.255.0.0 0.0.0.255 area 0
! interface Tunnel0
!  ip ospf network broadcast
!  ip ospf priority 0
```

## Tips

- Always start with Phase 3 for new deployments unless you have a specific reason not to --- it is the most flexible and scalable.
- Use IKEv2 over IKEv1 for better performance, EAP support, and asymmetric authentication.
- Set `ip mtu 1400` and `ip tcp adjust-mss 1360` on all tunnel interfaces to avoid fragmentation issues that silently break applications.
- Use `tunnel key` when multiple DMVPN clouds share the same tunnel source interface.
- Always configure `ip nhrp authentication` --- even though it is not encrypted, it prevents accidental misconfiguration from rogue devices.
- For FVRF, remember that `tunnel vrf <name>` refers to the transport VRF, not the overlay VRF. The tunnel interface itself stays in the global table or an `ip vrf forwarding` VRF.
- Test spoke-to-spoke connectivity with `traceroute` to verify traffic is not still hair-pinning through the hub.
- Monitor NHRP hold timers --- if they expire faster than IPsec rekey, tunnels flap. Align `ip nhrp holdtime` with IPsec SA lifetime.
- Use `debug dmvpn condition peer nbma <ip>` to filter debugs to a single spoke when troubleshooting in production.
- For large-scale DMVPN (100+ spokes), prefer BGP with the hub as a route reflector over EIGRP or OSPF to reduce control-plane overhead.

## See Also

- `ipsec` --- IPsec fundamentals, IKEv1/IKEv2 negotiation, ESP/AH
- `gre` --- GRE tunneling, mGRE, GRE keepalives, key usage
- `eigrp` --- EIGRP configuration, split-horizon, stub routing
- `ospf` --- OSPF network types, DR/BDR election, area design
- `bgp` --- BGP route reflection, next-hop-self, peer groups
- `vrf` --- VRF-Lite, FVRF, IVRF, route leaking
- `qos` --- QoS policy maps, classification, queuing, shaping
- `ipsec-ikev2` --- IKEv2 profiles, certificate auth, EAP

## References

- RFC 2332 --- NBMA Next Hop Resolution Protocol (NHRP)
- RFC 2784 --- Generic Routing Encapsulation (GRE)
- RFC 7676 --- IPv6 Support for GRE
- RFC 4303 --- IP Encapsulating Security Payload (ESP)
- RFC 7296 --- Internet Key Exchange Protocol Version 2 (IKEv2)
- Cisco DMVPN Design Guide --- https://www.cisco.com/c/en/us/td/docs/solutions/Enterprise/WAN_and_MAN/DMVPN_Design_Guide.html
- Cisco DMVPN Configuration Guide --- https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/sec_conn_dmvpn/configuration/xe-16/sec-conn-dmvpn-xe-16-book.html
