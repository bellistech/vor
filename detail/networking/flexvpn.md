# FlexVPN --- IKEv2 VPN Framework Architecture

> *FlexVPN represents Cisco's convergence of disparate VPN technologies --- EzVPN, DMVPN, GET VPN, and crypto-map VPNs --- into a single IKEv2-based framework. Where DMVPN bolted IKEv1/IKEv2 onto a multipoint GRE substrate with NHRP resolution, FlexVPN eliminates the GRE and NHRP layers entirely, using IKEv2's native capabilities (traffic selectors, configuration payloads, redirect, and MOBIKE) to negotiate tunnel parameters, push routes, and enable spoke-to-spoke shortcuts. Understanding FlexVPN requires a deep grasp of IKEv2's exchange model, the relationship between IKE SAs and Child SAs, the role of virtual tunnel interfaces in Cisco's forwarding plane, and how AAA integration transforms a static VPN topology into a policy-driven, per-user architecture.*

---

## 1. The Evolution from Crypto Maps to FlexVPN

### 1.1 Crypto Map Era (1990s--2000s)

The original Cisco IPsec implementation used crypto maps: numbered, ordered entries that matched traffic via ACLs and applied transforms and peer definitions. Crypto maps are fundamentally tied to physical interfaces and operate in the CEF exception path.

```
Packet flow with crypto maps:

1. Packet arrives at egress physical interface
2. CEF lookup determines outbound interface
3. Crypto map ACL is evaluated (classify)
4. If match: punt to crypto engine (process switching or hardware)
5. ESP encapsulation applied
6. New outer IP header added
7. Encrypted packet forwarded via physical interface

Problems:
- ACL-based classification is fragile and order-dependent
- No tunnel interface = no routing protocol over the VPN
- Each new remote network requires ACL modification on both sides
- No per-tunnel QoS (no interface to attach service-policy)
- IPsec SAs are not associated with any interface in the RIB/FIB
```

### 1.2 VTI Era (2000s)

Virtual Tunnel Interfaces (VTIs) solved the routing problem by creating a routable interface for each IPsec tunnel. Traffic forwarded into the VTI is automatically encrypted without ACL classification.

```
Packet flow with VTI:

1. CEF lookup determines destination is reachable via Tunnel0
2. Packet forwarded into Tunnel0 (tunnel interface in FIB)
3. Tunnel interface triggers IPsec encapsulation
4. ESP header applied
5. Outer IP header uses tunnel source/destination
6. Encrypted packet forwarded via physical interface

Advantages over crypto maps:
- Routing protocols run natively over the tunnel
- Per-tunnel QoS via service-policy on tunnel interface
- Dynamic routing eliminates ACL maintenance
- Tunnel interface appears in RIB/FIB (CEF-switched, not process-switched)
```

### 1.3 FlexVPN: VTI + IKEv2 Native Features

FlexVPN takes VTIs and adds IKEv2's full feature set: configuration payloads for IP assignment, traffic selector narrowing for proxy ID negotiation, redirect for spoke-to-spoke shortcuts, and AAA integration for per-user policy. The result is a VPN framework that handles site-to-site, remote-access, and shortcut topologies with one configuration paradigm.

---

## 2. IKEv2 Protocol Deep Dive

### 2.1 IKEv2 vs IKEv1: Structural Differences

IKEv1 uses two phases: Phase 1 (Main Mode or Aggressive Mode, 6 or 3 messages) establishes an ISAKMP SA, then Phase 2 (Quick Mode, 3 messages) establishes IPsec SAs. IKEv2 consolidates this into a minimum of 4 messages (2 request/response pairs).

```
IKEv1 Main Mode + Quick Mode:
  Messages: 6 (MM) + 3 (QM) = 9 messages minimum
  Round trips: 3 (MM) + 1.5 (QM) = 4.5 RTTs
  SAs created: 1 ISAKMP SA + 1 IPsec SA pair

IKEv2:
  Messages: 2 (IKE_SA_INIT) + 2 (IKE_AUTH) = 4 messages minimum
  Round trips: 2 RTTs
  SAs created: 1 IKE SA + 1 Child SA pair (IPsec)

Efficiency gain: 55% fewer messages, 55% fewer RTTs
```

### 2.2 IKE_SA_INIT Exchange

The first exchange establishes the IKE SA's cryptographic parameters and performs the Diffie-Hellman key exchange. All subsequent messages are encrypted under this IKE SA.

```
Initiator                              Responder
    |                                      |
    |  HDR(SPIi=X, SPIr=0, IKE_SA_INIT)   |
    |  SAi1: proposals [AES-256, SHA-256, Group 19]
    |  KEi: DH public value (ECP-256)      |
    |  Ni: initiator nonce (128-256 bits)  |
    |  [N(NAT_DETECTION_SOURCE_IP)]        |
    |  [N(NAT_DETECTION_DESTINATION_IP)]   |
    |------------------------------------->|
    |                                      |
    |  HDR(SPIi=X, SPIr=Y, IKE_SA_INIT)   |
    |  SAr1: selected [AES-256, SHA-256, Group 19]
    |  KEr: DH public value (ECP-256)      |
    |  Nr: responder nonce (128-256 bits)  |
    |  [N(NAT_DETECTION_SOURCE_IP)]        |
    |  [N(NAT_DETECTION_DESTINATION_IP)]   |
    |  [CERTREQ]                           |
    |<-------------------------------------|
    |                                      |

After this exchange:
  SKEYSEED = prf(Ni | Nr, g^ir)
  {SK_d, SK_ai, SK_ar, SK_ei, SK_er, SK_pi, SK_pr} = prf+(SKEYSEED, Ni | Nr | SPIi | SPIr)

Where:
  SK_d   = key for deriving child SA keys
  SK_ai  = IKE SA integrity key (initiator to responder)
  SK_ar  = IKE SA integrity key (responder to initiator)
  SK_ei  = IKE SA encryption key (initiator to responder)
  SK_er  = IKE SA encryption key (responder to initiator)
  SK_pi  = key for generating initiator AUTH payload
  SK_pr  = key for generating responder AUTH payload
```

### 2.3 IKE_AUTH Exchange

The second exchange authenticates both peers, establishes identity, and creates the first Child SA (IPsec SA pair). Everything in this exchange is encrypted under the IKE SA.

```
Initiator                              Responder
    |                                      |
    |  HDR(SPIi=X, SPIr=Y, IKE_AUTH)      |
    |  SK{                                 |
    |    IDi: identity (FQDN, DN, IP)      |
    |    [CERT: certificate chain]         |
    |    [CERTREQ]                         |
    |    AUTH: proof of identity            |
    |    [CP(CFG_REQUEST): request IP/DNS] |
    |    SAi2: child SA proposals          |
    |    TSi: initiator traffic selectors  |
    |    TSr: responder traffic selectors  |
    |  }                                   |
    |------------------------------------->|
    |                                      |
    |  HDR(SPIi=X, SPIr=Y, IKE_AUTH)      |
    |  SK{                                 |
    |    IDr: identity                     |
    |    [CERT: certificate chain]         |
    |    AUTH: proof of identity            |
    |    [CP(CFG_REPLY): assigned IP/DNS]  |
    |    SAr2: selected child SA           |
    |    TSi: narrowed traffic selectors   |
    |    TSr: narrowed traffic selectors   |
    |  }                                   |
    |<-------------------------------------|

AUTH payload computation (PSK):
  AUTH = prf(prf(Shared_Secret, "Key Pad for IKEv2"),
             <InitiatorSignedOctets>)

AUTH payload computation (RSA/ECDSA):
  AUTH = Sign(PrivateKey, <InitiatorSignedOctets>)

InitiatorSignedOctets = IKE_SA_INIT_Request | Nr | prf(SK_pi, IDi')
ResponderSignedOctets = IKE_SA_INIT_Response | Ni | prf(SK_pr, IDr')
```

### 2.4 CREATE_CHILD_SA Exchange

Used for rekeying the IKE SA, rekeying Child SAs, or creating additional Child SAs. This is the only exchange that can include an optional Diffie-Hellman exchange for Perfect Forward Secrecy (PFS).

```
Use cases:
  1. Rekey IKE SA:       SA payload contains IKE SA proposals
  2. Rekey Child SA:     SA + TSi/TSr + N(REKEY_SA, <SPI of old SA>)
  3. New Child SA:       SA + TSi/TSr (no REKEY_SA notify)

For PFS:
  Optional KEi/KEr payloads trigger a new DH exchange
  New Child SA keys = prf+(SK_d, Ni | Nr | g^ir_new)

Without PFS:
  New Child SA keys = prf+(SK_d, Ni | Nr)
```

### 2.5 NAT Traversal

IKEv2 includes native NAT traversal (no separate NAT-T negotiation like IKEv1). NAT detection occurs in IKE_SA_INIT via two notify payloads.

```
NAT_DETECTION_SOURCE_IP = SHA-1(SPIi | SPIr | Source_IP | Source_Port)
NAT_DETECTION_DESTINATION_IP = SHA-1(SPIi | SPIr | Dest_IP | Dest_Port)

If either hash mismatches:
  - NAT detected
  - All subsequent IKE messages use UDP 4500
  - ESP packets encapsulated in UDP 4500 (NAT-T encapsulation)
  - Non-ESP marker (4 bytes of 0x00) prepended to distinguish from IKE

UDP encapsulation format:
  [IP Header][UDP 4500][0x00000000][ESP Header][Encrypted Payload]
```

---

## 3. Virtual Tunnel Interface Architecture

### 3.1 SVTI (Static Virtual Tunnel Interface)

An SVTI is a point-to-point tunnel interface with a fixed tunnel destination. It maps one-to-one with an IKEv2 SA.

```
SVTI characteristics:
  - Configured as "interface Tunnel<N>"
  - Has explicit tunnel destination
  - One IKEv2 SA per SVTI
  - Appears in routing table as a connected interface
  - CEF entry points to tunnel interface
  - Supports all interface features (QoS, ACL, PBR, etc.)

Forwarding path:
  CEF lookup -> Tunnel0 -> IPsec encapsulation -> physical egress

SVTI is appropriate for:
  - Spoke-to-hub connections (known, fixed hub IP)
  - Small number of peers (< 10)
  - Scenarios requiring per-tunnel QoS or ACLs
```

### 3.2 DVTI (Dynamic Virtual Tunnel Interface)

A DVTI uses a Virtual-Template interface that is cloned into Virtual-Access interfaces on demand. Each incoming IKEv2 connection creates a new Virtual-Access interface.

```
DVTI operation:

1. Hub has Virtual-Template1 configured with tunnel parameters
2. Spoke initiates IKEv2 to hub
3. IKEv2 profile on hub references "virtual-template 1"
4. IOS clones Virtual-Template1 into Virtual-Access<N>
5. Virtual-Access<N> inherits all template configuration
6. IKEv2 dynamically sets tunnel destination to spoke's NBMA address
7. IPsec profile applied, Child SA created
8. Virtual-Access<N> appears in routing table

Virtual-Template cloning:
  Virtual-Template1:
    ip unnumbered Loopback0
    tunnel source GigabitEthernet0/0
    tunnel mode ipsec ipv4
    tunnel protection ipsec profile FLEX-IPSEC
    ip mtu 1400

  Cloned to Virtual-Access1:
    ip unnumbered Loopback0
    tunnel source GigabitEthernet0/0
    tunnel destination 203.0.113.10       <-- dynamically set
    tunnel mode ipsec ipv4
    tunnel protection ipsec profile FLEX-IPSEC
    ip mtu 1400

DVTI is appropriate for:
  - Hub accepting connections from many spokes
  - Number of spokes is large or unknown
  - Per-spoke policy via AAA (each Virtual-Access can have unique config)
```

### 3.3 Interface Numbering and Limits

```
Virtual-Access interface allocation:
  - Numbered sequentially from 1
  - Released when IKEv2 SA is deleted
  - Can be reused after release
  - Maximum depends on platform memory

Platform limits (approximate):
  ISR 4331:     ~500 Virtual-Access interfaces
  ISR 4451:     ~2000 Virtual-Access interfaces
  CSR 1000v:    ~4000 Virtual-Access interfaces (memory-dependent)
  ASR 1001-X:   ~8000 Virtual-Access interfaces
  ASR 1002-HX:  ~16000 Virtual-Access interfaces
```

---

## 4. AAA Integration Architecture

### 4.1 Authorization Flow

FlexVPN's AAA integration allows per-user and per-group policy to be applied dynamically during IKE_AUTH. The authorization flow determines IP addressing, routing, interface configuration, and access policies.

```
IKE_AUTH received at hub:
    |
    v
IKEv2 profile matched (match identity remote ...)
    |
    v
Group authorization (aaa authorization group ...)
    |--- Local: crypto ikev2 authorization policy <name>
    |--- RADIUS: Access-Request with group identity
    |
    v
User authorization (aaa authorization user ...)
    |--- Local: lookup by peer IKEv2 identity
    |--- RADIUS: Access-Request with peer identity
    |
    v
Merge group + user attributes (user overrides group)
    |
    v
Apply to Virtual-Access interface:
    - IP address (pool or specific)
    - Routes (RRI or pushed)
    - Interface config (bandwidth, QoS)
    - ACLs (per-user)
    - DNS, domain, banner
```

### 4.2 RADIUS Attribute Mapping

```
Standard RADIUS attributes for FlexVPN:
  Framed-IP-Address (8)       -> Assigned tunnel IP
  Framed-IP-Netmask (9)       -> Tunnel IP netmask
  Framed-Route (22)           -> Static routes to install
  Filter-Id (11)              -> ACL name to apply
  Session-Timeout (27)        -> IKE SA lifetime override
  Idle-Timeout (28)           -> DPD idle timeout

Cisco AV-Pairs for FlexVPN:
  ip:interface-config=<cmd>   -> Applied to Virtual-Access interface
  ip:route=<net> <mask>       -> Route installed pointing to VA
  ip:addr-pool=<name>         -> Address pool to use
  ip:dns-servers=<ip>         -> DNS server pushed via CP
  ip:wins-servers=<ip>        -> WINS server pushed via CP
  ipsec:tunnel-password=<key> -> Per-user PSK (override keyring)
  ipsec:route-set=prefix      -> RRI route set method
```

### 4.3 Per-User PSK via RADIUS

```
! On RADIUS server, for user "spoke1@example.com":
User-Name = "spoke1@example.com"
  Cisco-AV-Pair = "ipsec:tunnel-password=UniqueKeyForSpoke1"
  Framed-IP-Address = 10.255.1.10
  Cisco-AV-Pair = "ip:route=10.1.1.0 255.255.255.0"

! On hub:
crypto ikev2 profile FLEX-PROFILE
 match identity remote any
 authentication remote pre-share
 authentication local rsa-sig
 keyring local FLEX-KR
 aaa authorization user psk list FLEX-AAA

! The keyring provides a fallback PSK
! RADIUS "ipsec:tunnel-password" overrides it per-user
! This enables unique PSKs without per-spoke configuration on the hub
```

---

## 5. Traffic Selector Negotiation

### 5.1 Traffic Selector Theory

Traffic selectors (TS) in IKEv2 replace the proxy identities / ACLs of IKEv1's Quick Mode. They define which traffic the Child SA protects.

```
Traffic Selector format (RFC 7296 Section 3.13.1):

 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| TS Type       |  IP Protocol  |          Selector Length      |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|          Start Port           |           End Port            |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                   Starting Address                            |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                   Ending Address                              |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

TS Types:
  7 = TS_IPV4_ADDR_RANGE (16 bytes)
  8 = TS_IPV6_ADDR_RANGE (40 bytes)

For FlexVPN with VTI, typical TS:
  TSi = 0.0.0.0 - 255.255.255.255 (any IPv4)
  TSr = 0.0.0.0 - 255.255.255.255 (any IPv4)
  Protocol = 0 (any), Ports = 0-65535 (any)

This "any-any" selector is correct for VTI because the tunnel interface
handles route-based forwarding, not selector-based classification.
```

### 5.2 Traffic Selector Narrowing

The responder may narrow the initiator's proposed traffic selectors. This is how the hub can restrict which subnets a spoke can access.

```
Narrowing example:

Initiator proposes:
  TSi = 10.1.1.0 - 10.1.1.255     (spoke's subnet)
  TSr = 0.0.0.0 - 255.255.255.255 (wants access to everything)

Responder narrows:
  TSi = 10.1.1.0 - 10.1.1.255     (accepted)
  TSr = 10.0.0.0 - 10.0.0.255     (restricted to hub's LAN only)

The narrowed TS defines what traffic the Child SA will encrypt.
Packets outside the TS range are dropped by IPsec.

In FlexVPN, narrowing is typically driven by:
  - AAA authorization policy (route set access-list)
  - IKEv2 authorization policy definitions
  - Per-user RADIUS attributes
```

### 5.3 Multiple Traffic Selectors

```
IKEv2 allows multiple TS entries in a single payload:

TSi = [10.1.1.0/24, 10.1.2.0/24, 10.1.3.0/24]
TSr = [10.0.0.0/24, 172.16.0.0/16]

Each combination creates a separate IPsec SA pair:
  SA1: 10.1.1.0/24 <-> 10.0.0.0/24
  SA2: 10.1.1.0/24 <-> 172.16.0.0/16
  SA3: 10.1.2.0/24 <-> 10.0.0.0/24
  ...

With VTI mode, this complexity is avoided because the VTI uses
a single any-any selector and relies on routing for traffic steering.
```

---

## 6. IKEv2 Redirect and Spoke-to-Spoke Shortcuts

### 6.1 Redirect Mechanism (RFC 5685)

IKEv2 redirect allows a responder to tell an initiator to connect to a different gateway. FlexVPN uses this for two purposes: hub load balancing and spoke-to-spoke shortcut switching.

```
Redirect during IKE_SA_INIT (load balancing):

Spoke -> Hub1:  IKE_SA_INIT request
Hub1 -> Spoke:  IKE_SA_INIT response + N(REDIRECT, Hub2_IP)
Spoke -> Hub2:  IKE_SA_INIT request (new attempt)
Hub2 -> Spoke:  IKE_SA_INIT response (normal)
...continue IKE_AUTH with Hub2...

Redirect after IKE SA established (spoke-to-spoke):

Spoke_A -> Hub:   Encrypted traffic to Spoke_B subnet
Hub:              Detects traffic should go to Spoke_B
Hub -> Spoke_A:   INFORMATIONAL{N(REDIRECT, Spoke_B_NBMA_IP)}
Spoke_A:          Initiates IKE_SA_INIT directly to Spoke_B
Spoke_A <-> Spoke_B: Direct IKE SA + Child SA established
Spoke_A -> Spoke_B:  Direct encrypted traffic (bypasses hub)
```

### 6.2 Shortcut Switching Internals

```
Shortcut tunnel lifecycle:

1. Spoke_A sends packet to Spoke_B's subnet via hub tunnel
2. Hub receives decrypted packet, forwards to Spoke_B (via its VA)
3. Hub sends INFORMATIONAL with N(REDIRECT) to Spoke_A
   - Contains Spoke_B's NBMA address and identity
4. Spoke_A initiates IKE_SA_INIT to Spoke_B directly
5. Spoke_B must have:
   - IKEv2 profile accepting connections from other spokes
   - Virtual-Template for creating dynamic shortcut tunnels
6. Full IKEv2 exchange completes (IKE_SA_INIT + IKE_AUTH)
7. New Virtual-Access created on both Spoke_A and Spoke_B
8. Routes installed (static or RRI) pointing to shortcut VA
9. Traffic flows directly Spoke_A <-> Spoke_B

Shortcut teardown:
  - DPD timeout: if no traffic flows, DPD detects dead peer
  - Idle timeout: configured via IKEv2 profile or AAA
  - Explicit delete: INFORMATIONAL{D(IKE_SA_SPI)}
```

### 6.3 Comparison with DMVPN Spoke-to-Spoke

```
                    DMVPN Phase 3              FlexVPN Shortcut
Resolution          NHRP Resolution/Redirect   IKEv2 REDIRECT notify
Resolution Time     ~2-5 sec (NHRP + IKEv2)   ~2-4 sec (IKEv2 only)
Tunnel Setup        mGRE + IPsec              VTI + IPsec (no GRE)
Overhead per pkt    GRE (24B) + ESP            ESP only (no GRE)
Multicast           Supported (mGRE)           Not supported (unicast only)
Spoke Config        NHRP maps + mGRE           IKEv2 profile + VT
Hub Config          NHRP server + mGRE         IKEv2 redirect + VT
Standards           NHRP (RFC 2332)            IKEv2 redirect (RFC 5685)
```

---

## 7. Route Injection Mechanisms

### 7.1 Reverse Route Injection (RRI)

RRI creates static routes on the hub for each spoke's subnets, derived from the IKEv2 traffic selectors negotiated during IKE_AUTH.

```
RRI operation:

1. Spoke connects with TSi = 10.1.1.0/24
2. Hub's ipsec profile has "set reverse-route"
3. Hub installs: ip route 10.1.1.0 255.255.255.0 Virtual-Access1
4. Route is tagged (if configured) and has configurable AD

RRI route lifecycle:
  - Created: when Child SA is established
  - Removed: when Child SA is deleted (rekey creates new, then deletes old)
  - Redistributed: via route-map matching tag into OSPF/BGP/EIGRP

RRI + routing protocol interaction:
  RRI route (AD 1) -> redistributed into OSPF (E1/E2)
  RRI route (AD 1) -> redistributed into BGP (origin incomplete)
  RRI route (AD 1) -> redistributed into EIGRP (external)

  If spoke also runs routing protocol over tunnel:
    IGP route (AD varies) competes with RRI route (AD 1)
    RRI wins by default (AD 1 < OSPF 110, BGP 200, EIGRP 170)
    Use "set reverse-route distance 200" to let IGP win
```

### 7.2 AAA-Based Route Push

```
Routes pushed via AAA are installed as static routes pointing to the
spoke's Virtual-Access interface, similar to RRI but explicitly defined.

AAA route sources:
  1. Local: crypto ikev2 authorization policy -> route set access-list
  2. RADIUS: Cisco-AV-Pair "ip:route=<network> <mask>"
  3. RADIUS: Framed-Route attribute

AAA routes vs RRI:
  - RRI is derived from traffic selectors (what the spoke claims)
  - AAA routes are defined by policy (what the hub allows)
  - In conflict, AAA routes typically take precedence
  - Use AAA routes for security (hub controls routing)
  - Use RRI for simplicity (spoke self-declares subnets)
```

---

## 8. Dual-Stack Architecture

### 8.1 IKEv2 Dual-Stack Negotiation

```
IKEv2 supports multiple traffic selectors of different address families
in a single IKE_AUTH exchange:

TSi payload:
  TS[0]: type=TS_IPV4_ADDR_RANGE, 0.0.0.0 - 255.255.255.255
  TS[1]: type=TS_IPV6_ADDR_RANGE, :: - ffff:...:ffff

TSr payload:
  TS[0]: type=TS_IPV4_ADDR_RANGE, 0.0.0.0 - 255.255.255.255
  TS[1]: type=TS_IPV6_ADDR_RANGE, :: - ffff:...:ffff

The responder can accept or narrow each TS independently.
A single Child SA can protect both IPv4 and IPv6 traffic.
```

### 8.2 Dual-Stack VTI Configuration

```
! Single tunnel carrying both IPv4 and IPv6 overlay traffic
! over an IPv4 underlay

interface Tunnel0
 ip address 10.255.0.2 255.255.255.0
 ipv6 address 2001:db8:ff::2/64
 tunnel source GigabitEthernet0/0
 tunnel destination 198.51.100.1
 tunnel mode ipsec ipv4               ! Underlay: IPv4
 tunnel protection ipsec profile FLEX-IPSEC

! Routing: both address families over the same tunnel
router ospfv3 1
 address-family ipv4 unicast
  router-id 10.1.1.1
  exit-address-family
 address-family ipv6 unicast
  router-id 10.1.1.1
  exit-address-family

interface Tunnel0
 ospfv3 1 area 0 ipv4
 ospfv3 1 area 0 ipv6
```

### 8.3 IPv6-Only Underlay

```
! When the transport network is IPv6-only:

interface Tunnel0
 ip address 10.255.0.2 255.255.255.0   ! IPv4 overlay
 ipv6 address 2001:db8:ff::2/64        ! IPv6 overlay
 tunnel source GigabitEthernet0/0
 tunnel destination 2001:db8:1::1       ! IPv6 underlay
 tunnel mode ipsec ipv6                 ! Underlay: IPv6

! IKEv2 runs over IPv6 (UDP 500/4500 on IPv6 transport)
! ESP is encapsulated in IPv6 outer headers
! Inner traffic can be IPv4, IPv6, or both
```

---

## 9. FlexVPN Redundancy and High Availability

### 9.1 Dual-Hub with IKEv2 Redirect

```
Architecture:
  Hub1 (primary):   198.51.100.1, crypto ikev2 redirect gateway init
  Hub2 (backup):    198.51.100.2, crypto ikev2 redirect gateway init
  Spoke:            crypto ikev2 redirect client

Failover flow:
  1. Spoke connects to Hub1 (primary)
  2. Hub1 fails (or enters maintenance)
  3. Spoke's DPD detects Hub1 is dead
  4. Spoke reconnects to Hub2 (configured as backup peer)
  5. When Hub1 recovers, Hub2 can redirect spoke back to Hub1

Active-active load balancing:
  1. Spoke connects to Hub1
  2. Hub1 decides Spoke should use Hub2 (load balancing algorithm)
  3. Hub1 responds to IKE_SA_INIT with N(REDIRECT, Hub2)
  4. Spoke reconnects to Hub2
  5. Hub2 accepts the connection normally
```

### 9.2 Stateful Failover Limitations

```
FlexVPN does not support stateful HA (IKE SA sync between hubs).
On failover:
  - IKEv2 SAs are re-established from scratch
  - IPsec SAs are re-negotiated (new keys)
  - Application sessions may break (TCP sessions)
  - Typical reconvergence: 30-60 seconds (DPD + IKEv2 + routing)

Mitigation strategies:
  - Aggressive DPD: dpd 10 3 periodic (detect failure in ~30 sec)
  - BGP BFD: rapid routing reconvergence over new tunnel
  - Application-layer keepalives for session continuity
  - TCP keepalives to detect broken connections
```

---

## 10. Performance Considerations

### 10.1 IKEv2 SA Scaling

```
Per-IKE SA memory:
  IKE SA state:        ~2-4 KB
  Child SA state:      ~1-2 KB per pair
  Virtual-Access:      ~8-16 KB (interface structures)
  Routing state:       variable (per protocol)

Platform SA limits (IKEv2 + IPsec):
  ISR 4221:     ~1,000 IKE SAs, ~2,000 IPsec SAs
  ISR 4331:     ~2,000 IKE SAs, ~4,000 IPsec SAs
  ISR 4451:     ~5,000 IKE SAs, ~10,000 IPsec SAs
  CSR 1000v:    ~10,000 IKE SAs (memory-dependent)
  ASR 1001-X:   ~20,000 IKE SAs, ~40,000 IPsec SAs (with EPA)
```

### 10.2 Throughput and MTU

```
Overhead calculation:

Original packet:              1500 bytes (Ethernet MTU)
ESP header:                      8 bytes
ESP IV (AES-CBC):               16 bytes
ESP padding:                   1-15 bytes (to block boundary)
ESP pad length + next header:    2 bytes
ESP auth (SHA-256-128):         16 bytes (truncated to 128 bits)
New IP header:                  20 bytes (IPv4) or 40 bytes (IPv6)
                               ────────
Total overhead (IPv4):          ~62-76 bytes
Total overhead (IPv6):          ~82-96 bytes

Recommended tunnel MTU settings:
  IPv4 underlay: ip mtu 1400 (conservative), ip mtu 1438 (tight)
  IPv6 underlay: ip mtu 1380 (conservative), ip mtu 1418 (tight)
  Always set: ip tcp adjust-mss = ip mtu - 40

With AES-GCM (combined mode, no separate integrity):
  ESP header:       8 bytes
  ESP IV:           8 bytes (GCM uses 8-byte IV)
  ESP ICV:         16 bytes (GCM authentication tag)
  New IP header:   20 bytes (IPv4)
                  ────────
  Total overhead:  ~52-60 bytes (less than CBC + HMAC)
```

---

## 11. Migration from DMVPN to FlexVPN

### 11.1 Parallel Deployment Strategy

```
Phase 1: Deploy FlexVPN hub alongside existing DMVPN hub
  - Same physical router, different tunnel interfaces
  - FlexVPN: Virtual-Template1, Loopback1
  - DMVPN: Tunnel0 (existing)
  - Both advertise the same hub networks

Phase 2: Migrate spokes incrementally
  - Configure FlexVPN spoke tunnel (Tunnel1) alongside DMVPN (Tunnel0)
  - Verify FlexVPN connectivity
  - Adjust routing to prefer FlexVPN (lower AD or metric)
  - Shut down DMVPN tunnel on spoke

Phase 3: Decommission DMVPN
  - Once all spokes migrated, remove DMVPN configuration from hub
  - Clean up NHRP, mGRE, and IKEv1 configurations

Key differences for spoke administrators:
  DMVPN:    tunnel mode gre multipoint + NHRP + IPsec profile
  FlexVPN:  tunnel mode ipsec ipv4 + IKEv2 profile
  DMVPN:    NHRP registration to hub (ip nhrp nhs)
  FlexVPN:  IKEv2 SA to hub (tunnel destination or DVTI)
  DMVPN:    NHRP shortcuts (Phase 3)
  FlexVPN:  IKEv2 redirect shortcuts
```

---

## 12. Security Hardening

### 12.1 Proposal Hardening

```
! Minimum recommended for 2024+:
crypto ikev2 proposal HARDENED
 encryption aes-gcm-256               ! AEAD (no separate integrity needed)
 group 19                             ! ECP-256 (NIST P-256)

! For post-quantum considerations:
crypto ikev2 proposal FUTURE
 encryption aes-gcm-256
 group 21                             ! ECP-521 (largest standard curve)

! Disable smart defaults to prevent negotiation of weak suites:
no crypto ikev2 proposal default
```

### 12.2 Certificate Authentication Best Practices

```
! Use ECDSA certificates for performance:
crypto pki trustpoint FLEX-CA
 enrollment url http://ca.example.com
 revocation-check crl ocsp            ! Check both CRL and OCSP
 eckeypair FLEX-EC 384                ! ECDSA P-384
 hash sha384                         ! Match key strength

! Certificate validation:
  - Always enable revocation checking (never "none" in production)
  - Use OCSP for near-real-time revocation
  - Set CRL cache lifetime: crl cache delete-after <minutes>
  - Validate certificate chain up to root CA
  - Use identity matching: match identity remote dn (most specific)
```

### 12.3 Anti-Replay and DPD

```
! Anti-replay:
  IKEv2 includes built-in anti-replay via message IDs
  IPsec uses sequence number window (default 64 packets)
  For high-throughput links: crypto ipsec security-association replay window-size 1024

! Dead Peer Detection:
crypto ikev2 profile FLEX-PROFILE
 dpd 30 5 on-demand
 ! 30 = interval (seconds) between DPD probes when idle
 ! 5 = retry count before declaring peer dead
 ! on-demand = only send DPD when there is traffic to send
 !   (vs periodic = always send DPD regardless of traffic)

 ! Total detection time (worst case):
 !   on-demand: 30 * 5 = 150 seconds
 !   periodic:  30 * 5 = 150 seconds (but continuous overhead)
```

---

## Prerequisites

Before deploying FlexVPN, ensure familiarity with:

- **IKEv2 protocol:** RFC 7296, exchanges, payloads, SA lifecycle, and rekeying
- **IPsec fundamentals:** ESP encapsulation, tunnel vs transport mode, anti-replay
- **X.509 certificates:** PKI enrollment, certificate chains, CRL/OCSP, trust anchors
- **Cisco VTI:** Static and dynamic tunnel interfaces, tunnel modes
- **AAA framework:** RADIUS attribute-value pairs, authorization policies, local vs server-based
- **IP routing:** At least one of OSPF, BGP, or EIGRP for overlay routing
- **Cisco IOS-XE CLI:** Interface configuration, crypto subsystem, show/debug commands

---

## References

- RFC 7296 --- Internet Key Exchange Protocol Version 2 (IKEv2)
- RFC 4303 --- IP Encapsulating Security Payload (ESP)
- RFC 5685 --- Redirect Mechanism for the Internet Key Exchange Protocol Version 2 (IKEv2)
- RFC 7383 --- Internet Key Exchange Protocol Version 2 (IKEv2) Message Fragmentation
- RFC 4555 --- IKEv2 Mobility and Multihoming Protocol (MOBIKE)
- RFC 5998 --- An Extension for EAP-Only Authentication in IKEv2
- RFC 5996 --- Internet Key Exchange Protocol Version 2 (IKEv2) -- Clarifications and Implementation Guidelines
- RFC 6023 --- A Childless Initiation of the Internet Key Exchange Version 2 (IKEv2) Security Association
- Cisco FlexVPN Configuration Guide (IOS XE 16/17) --- https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/sec_conn_ike2vpn/configuration/xe-16/sec-flex-vpn-xe-16-book.html
- Cisco IKEv2 Configuration Guide --- https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/sec_conn_ikevpn/configuration/xe-16/sec-sec-for-vpns-w-ipsec-xe-16-book.html
- Cisco FlexVPN Between Hub and Spoke --- https://www.cisco.com/c/en/us/support/docs/security/flexvpn/200555-FlexVPN-Between-a-Hub-and-a-Remote-Spoke.html
- "IKEv2 IPsec Virtual Private Networks" by Graham Bartlett, Amjad Inamdar (Cisco Press)
- NIST SP 800-77 Rev. 1 --- Guide to IPsec VPNs
