# BNG (Broadband Network Gateway)

Subscriber-aware edge router in service provider access networks that terminates subscriber sessions, enforces per-subscriber policy, and bridges the access and core networks.

## Concepts

### BNG Architecture

- **Access layer:** DSLAMs, OLTs, access switches connecting CPE to the aggregation network
- **Aggregation layer:** Pre-aggregation and aggregation switches/routers; BNG sits here or at the edge of core
- **Core layer:** P/PE routers running MPLS/SR; BNG hands off authenticated, policy-applied traffic to the core
- **BNG role:** Session termination, authentication, authorization, accounting, per-subscriber QoS/ACL/NAT

### Subscriber Session Types

| Type | Trigger | Use Case |
|------|---------|----------|
| PPPoE | PADI discovery | DSL, FTTH legacy, enterprise |
| IPoE | DHCP Discover | FTTH modern, cable (DOCSIS), wireless backhaul |
| L2TP (LAC) | PPPoE forwarded via tunnel | Wholesale, multi-ISP |

### PPPoE Protocol State Machine

```
CPE                          BNG (AC)
 |--- PADI (broadcast) ------->|   # Discovery Initiation
 |<-- PADO (unicast) ----------|   # Discovery Offer
 |--- PADR (unicast) --------->|   # Discovery Request
 |<-- PADS (unicast) ----------|   # Discovery Session-Confirmation (session_id assigned)
 |                              |
 |=== PPP LCP Negotiation =====>|   # Link Control Protocol
 |=== PPP Authentication ======>|   # PAP or CHAP (RADIUS proxy)
 |=== PPP IPCP/IPv6CP =========>|   # IP address assignment
 |                              |
 |--- PADT --------------------->|   # Discovery Terminate (either side)
```

### IPoE Session Lifecycle

```
CPE                          BNG
 |--- DHCP Discover ----------->|   # Subscriber initiates
 |    (Option 82 inserted       |   # Access-node adds circuit-id/remote-id
 |     by access node)          |
 |<-- DHCP Offer ---------------|   # BNG/DHCP server responds
 |--- DHCP Request ------------->|
 |<-- DHCP Ack -----------------|   # Session created, RADIUS accounting starts
 |                              |
 |--- DHCP Release ------------->|   # Session teardown
```

## AAA Integration

### RADIUS Authentication Flow

```
BNG                         RADIUS Server
 |--- Access-Request ---------->|   # Username, password, NAS-Port-Id
 |    NAS-IP-Address            |   # Circuit-id, remote-id (Option 82)
 |    NAS-Port-Id               |
 |    Calling-Station-Id        |
 |<-- Access-Accept ------------|   # Framed-IP-Address, Framed-Pool
 |    (or Access-Reject)        |   # QoS attributes, ACL, service template
```

### RADIUS Accounting

```
# Accounting-Request types
Acct-Status-Type = Start          # Session established
Acct-Status-Type = Interim-Update # Periodic (every 15-30 min typically)
Acct-Status-Type = Stop           # Session terminated

# Key attributes in accounting
Acct-Session-Id                   # Unique session identifier
Acct-Input-Octets / Acct-Output-Octets
Acct-Input-Packets / Acct-Output-Packets
Acct-Session-Time                 # Duration in seconds
Acct-Terminate-Cause              # Why session ended
```

### Change of Authorization (CoA)

```
RADIUS Server                BNG
 |--- CoA-Request ------------->|   # Modify active session
 |    Acct-Session-Id           |   # Target specific session
 |    Cisco-AVPair="sub-qos-policy-in=NEW-POLICY"
 |<-- CoA-ACK ------------------|   # Applied successfully
 |    (or CoA-NAK)              |   # Failed to apply

# Disconnect-Request (Pod)
 |--- Disconnect-Request ------>|   # Terminate session
 |    Acct-Session-Id           |
 |<-- Disconnect-ACK -----------|
```

### Common Cisco AVPairs

```
# QoS policy application
Cisco-AVPair = "sub-qos-policy-in=SUBSCRIBER-IN"
Cisco-AVPair = "sub-qos-policy-out=SUBSCRIBER-OUT"

# ACL application
Cisco-AVPair = "ip:inacl=SUBSCRIBER-ACL-IN"
Cisco-AVPair = "ip:outacl=SUBSCRIBER-ACL-OUT"

# VRF assignment
Cisco-AVPair = "lcp:interface-config=ip vrf forwarding CUSTOMER-VRF"

# IP address assignment
Framed-IP-Address = 100.64.1.10
Framed-IP-Netmask = 255.255.255.255
Framed-Pool = "RESIDENTIAL-POOL"

# Delegated-IPv6-Prefix (IA_PD)
Delegated-IPv6-Prefix = 2001:db8:c000::/48

# Service activation
Cisco-AVPair = "subscriber:sa=INTERNET"
Cisco-AVPair = "subscriber:sa=VOIP"
```

## IOS-XR BNG Configuration

### Interface and BBA Group

```
! Physical or bundle interface toward access network
interface Bundle-Ether100
 description *** Aggregation Uplink ***
 no shutdown

! Sub-interface with VLAN encapsulation
interface Bundle-Ether100.100
 ipv4 point-to-point
 ipv4 unnumbered Loopback100
 encapsulation dot1q 100
 pppoe enable bba-group RESIDENTIAL
 !

! BBA Group defines PPPoE behavior
bba-group pppoe RESIDENTIAL
 sessions per-vlan throttle 4000
 sessions per-vlan limit 8000
 sessions per-mac limit 4
 service name ISP-BROADBAND
 !
```

### PPPoE BNG Configuration

```
! Dynamic template for PPPoE subscribers
dynamic-template type ppp PPP-DEFAULT
 ppp authentication chap
 ppp ipcp peer-address pool RESIDENTIAL-POOL
 ipv4 unnumbered Loopback100
 accounting aaa list default type session periodic-interval 15
 !

! Address pool
pool vrf default ipv4 RESIDENTIAL-POOL
 address-range 100.64.0.1 100.64.255.254
 !

! PPP interface
interface Bundle-Ether100.100
 pppoe enable bba-group RESIDENTIAL
 !
```

### IPoE BNG Configuration

```
! Dynamic template for IPoE (DHCP-triggered) subscribers
dynamic-template type ipsubscriber IPOE-DEFAULT
 ipv4 unnumbered Loopback100
 accounting aaa list default type session periodic-interval 15
 !

! DHCP profile
dhcp ipv4
 profile RESIDENTIAL-DHCP server
  lease 0 8 0    ! 8 hours
  pool RESIDENTIAL-POOL
  dns-server 8.8.8.8 8.8.4.4
  default-router 100.64.0.1
  !
 !
 interface Bundle-Ether100.200
  server profile RESIDENTIAL-DHCP
  !
 !
```

### AAA Configuration

```
! RADIUS server group
aaa group server radius ISP-RADIUS
 server-private 10.10.10.1 auth-port 1812 acct-port 1813
  key 7 <encrypted-key>
  !
 server-private 10.10.10.2 auth-port 1812 acct-port 1813
  key 7 <encrypted-key>
  !
 !

! AAA method lists
aaa authentication subscriber default group ISP-RADIUS
aaa authorization subscriber default group ISP-RADIUS
aaa accounting subscriber default group ISP-RADIUS

! CoA server (for dynamic policy changes)
aaa server radius dynamic-author
 client 10.10.10.1 server-key <encrypted-key>
 client 10.10.10.2 server-key <encrypted-key>
 !
```

### Dynamic Templates and Service Activation

```
! Service template applied via RADIUS
dynamic-template type service INTERNET-100M
 qos output service-policy SHAPE-100M
 qos input service-policy POLICE-100M
 ipv4 access-group ACL-INTERNET-IN ingress
 ipv4 access-group ACL-INTERNET-OUT egress
 !

dynamic-template type service VOIP-SERVICE
 qos output service-policy VOIP-PRIORITY
 !

! Activate via RADIUS AVPair:
! Cisco-AVPair = "subscriber:sa=INTERNET-100M"
! Cisco-AVPair = "subscriber:sa=VOIP-SERVICE"

! Multiple services can be stacked on one subscriber session
```

### Per-Subscriber QoS

```
! Shaping policy (egress toward subscriber)
policy-map SHAPE-100M
 class class-default
  service-policy CHILD-QUEUING
  shape average 100000000   ! 100 Mbps
  !
 !

! Child policy for prioritization within the shaped rate
policy-map CHILD-QUEUING
 class VOIP
  priority level 1
  police rate 2000000       ! 2 Mbps reserved for VoIP
  !
 class VIDEO
  bandwidth remaining percent 40
  !
 class class-default
  bandwidth remaining percent 60
  !
 !

! Policing policy (ingress from subscriber)
policy-map POLICE-100M
 class class-default
  police rate 100000000 burst 1250000
   conform-action transmit
   exceed-action drop
   !
  !
 !
```

### Per-Subscriber NAT (CGN Integration)

```
! NAT inside interface (subscriber-facing)
dynamic-template type ppp PPP-WITH-NAT
 service-policy type pbr NAT-REDIRECT
 !

! NAT service on BNG or redirect to external CGN
! See cgnat sheet for detailed CGN configuration
```

## Show Commands

```bash
# Subscriber session summary
show subscriber session all summary

# Detailed subscriber session (by IP, MAC, or session-id)
show subscriber session filter ipv4-address 100.64.1.10 detail
show subscriber session filter mac-address 00aa.bbcc.ddee detail

# PPPoE session info
show pppoe session

# BBA group statistics
show pppoe bba-group RESIDENTIAL

# AAA statistics
show aaa accounting subscriber
show radius accounting

# Dynamic template application
show subscriber session filter ipv4-address 100.64.1.10 internal

# DHCP bindings (IPoE)
show dhcp ipv4 server binding

# Address pool utilization
show pool vrf default ipv4 RESIDENTIAL-POOL

# Subscriber feature counters (QoS, ACL drops)
show subscriber session filter ipv4-address 100.64.1.10 | include QoS
show policy-map interface Bundle-Ether100.100
```

## Troubleshooting

### PPPoE Session Not Establishing

```bash
# Verify PADI/PADO on interface
monitor interface Bundle-Ether100.100 | include PPPoE

# Check BBA group is applied
show running-config interface Bundle-Ether100.100 | include pppoe

# Debug PPPoE (caution in production)
debug pppoe events
debug pppoe packets

# Verify RADIUS reachability
ping 10.10.10.1 source Loopback0
show radius server-group ISP-RADIUS

# Check for session limits hit
show pppoe bba-group RESIDENTIAL | include limit
```

### Subscriber Cannot Get IP Address

```bash
# PPPoE: Check IPCP negotiation
show subscriber session filter mac-address 00aa.bbcc.ddee detail | include IPCP

# Verify pool has available addresses
show pool vrf default ipv4 RESIDENTIAL-POOL | include Utiliz

# IPoE: Check DHCP server state
show dhcp ipv4 server statistics

# Verify RADIUS is returning Framed-IP-Address or Framed-Pool
# Check RADIUS server logs for Access-Accept content
```

### Session Drops / Flapping

```bash
# Check session terminate causes
show subscriber session filter ipv4-address 100.64.1.10 history

# RADIUS accounting Stop reason
# Acct-Terminate-Cause common values:
#   1 = User-Request        (normal logout)
#   2 = Lost-Carrier        (link down)
#   4 = Idle-Timeout
#   5 = Session-Timeout
#   6 = Admin-Reset         (CoA disconnect)
#   9 = NAS-Reboot

# Check keepalive/LCP echo failures
show ppp interface Bundle-Ether100.100 | include echo

# Verify access node is stable (check for interface flaps)
show interface Bundle-Ether100 | include "rate|error|flap"
```

### CoA Not Working

```bash
# Verify CoA client is configured
show running-config aaa server radius dynamic-author

# Check CoA statistics
show radius dynamic-author statistics

# Ensure Acct-Session-Id in CoA matches active session
show subscriber session all | include <session-id>

# Verify UDP 3799 (CoA port) is reachable from RADIUS server
```

## Tips

- Always set per-VLAN and per-MAC session limits in BBA groups to prevent resource exhaustion from misbehaving CPE.
- Use `accounting periodic-interval 15` (minutes) to ensure billing accuracy; shorter intervals increase RADIUS load.
- Apply QoS via dynamic templates activated through RADIUS, not statically on interfaces, for per-subscriber granularity.
- For IPoE, rely on DHCP Option 82 (circuit-id/remote-id) for subscriber identification rather than MAC address alone.
- Use RADIUS CoA (Change of Authorization) for real-time service upgrades/downgrades without disconnecting the subscriber.
- Test RADIUS failover by shutting down the primary server; the BNG should seamlessly fall over to the secondary.
- Monitor address pool utilization closely; pool exhaustion silently blocks new subscribers.
- Keep PPPoE MTU at 1492 (1500 minus 8-byte PPPoE header) unless baby jumbo frames are supported end-to-end.
- Use loopback unnumbered for subscriber interfaces to conserve IPv4 address space on the BNG itself.
- Enable subscriber session history (`subscriber manager session history`) for post-mortem troubleshooting.

## See Also

- radius, mpls, bgp, cgnat, qos, ipv4, ipv6, ethernet

## References

- [RFC 2516 — A Method for Transmitting PPP Over Ethernet (PPPoE)](https://www.rfc-editor.org/rfc/rfc2516)
- [RFC 2865 — Remote Authentication Dial In User Service (RADIUS)](https://www.rfc-editor.org/rfc/rfc2865)
- [RFC 5176 — Dynamic Authorization Extensions to RADIUS](https://www.rfc-editor.org/rfc/rfc5176)
- [RFC 3046 — DHCP Relay Agent Information Option (Option 82)](https://www.rfc-editor.org/rfc/rfc3046)
- [Cisco IOS-XR BNG Configuration Guide](https://www.cisco.com/c/en/us/td/docs/routers/asr9000/software/bng/configuration/guide/b-bng-cg-asr9000.html)
- [Juniper MX Series BNG Configuration](https://www.juniper.net/documentation/us/en/software/junos/subscriber-mgmt/topics/concept/subscriber-management-overview.html)
