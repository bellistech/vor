# Collision & Broadcast Domains (Layer 1/2/3 Segmentation)

How network devices partition the physical and logical boundaries where frame collisions and broadcast traffic propagate, and why segmentation at each layer matters for performance, scalability, and security.

## Collision Domains

```
# A collision domain is a set of devices sharing the same physical medium
# where simultaneous transmissions cause frame collisions.
#
# CSMA/CD (Carrier Sense Multiple Access / Collision Detection):
# 1. Listen before transmit (carrier sense)
# 2. If medium is idle, transmit
# 3. If collision detected, send jam signal (48 bits)
# 4. Wait random backoff time (exponential: 0 to 2^c - 1 slot times)
# 5. Retry (up to 16 attempts, then drop frame)
#
# Only applies to HALF-DUPLEX Ethernet (hubs, coax, shared media)
# Full-duplex links have NO collisions — CSMA/CD is disabled

# Slot time (minimum frame transmission time):
#   10 Mbps:   51.2 us  (512 bits / 10 Mbps)
#   100 Mbps:  5.12 us  (512 bits / 100 Mbps)
#   1 Gbps:    4.096 us (4096 bits / 1 Gbps — extended slot time)
```

## Broadcast Domains

```
# A broadcast domain is a set of devices that all receive a Layer 2
# broadcast frame (destination MAC ff:ff:ff:ff:ff:ff).
#
# Common broadcast traffic:
#   ARP requests     — "Who has 192.168.1.1? Tell 192.168.1.50"
#   DHCP Discover    — "I need an IP address" (0.0.0.0 → 255.255.255.255)
#   DHCP Offer       — server response (may be broadcast or unicast)
#   NetBIOS          — Windows name resolution (legacy)
#   mDNS             — multicast DNS (224.0.0.251, not strictly broadcast)
#   OSPF Hello       — 224.0.0.5 (multicast, but flooded within segment)
#
# Every device in the broadcast domain must process every broadcast frame
# Even if the frame is irrelevant — NIC interrupts CPU, kernel processes it
```

## Hub vs Switch vs Router

### Hub (Layer 1 — Physical)

```
# Hub = multiport repeater
# ALL ports share ONE collision domain
# ALL ports share ONE broadcast domain
# Every frame is flooded to every port (no MAC learning)

         Hub (4 ports)
  ┌─────────────────────────┐
  │                         │
  │  ┌───┐ ┌───┐ ┌───┐ ┌───┐
  │  │ 1 │ │ 2 │ │ 3 │ │ 4 │
  └──┴───┴─┴───┴─┴───┴─┴───┘
     │       │       │     │
    PC-A    PC-B   PC-C  PC-D

  Collision domains:  1 (all ports)
  Broadcast domains:  1 (all ports)

  If PC-A and PC-C transmit simultaneously → COLLISION
  If PC-A sends broadcast → PC-B, PC-C, PC-D all receive it
```

### Switch (Layer 2 — Data Link)

```
# Switch = multiport bridge with MAC address table
# Each port is its own collision domain (microsegmentation)
# All ports in the same VLAN share one broadcast domain
# Unicast: forwarded only to the destination port (after MAC learning)
# Broadcast/unknown unicast: flooded to all ports in the VLAN

         Switch (4 ports, default VLAN 1)
  ┌─────────────────────────────────┐
  │     MAC Address Table           │
  │  ┌───┐  ┌───┐  ┌───┐  ┌───┐   │
  │  │ 1 │  │ 2 │  │ 3 │  │ 4 │   │
  └──┴───┴──┴───┴──┴───┴──┴───┘   │
     │        │       │      │
    PC-A     PC-B   PC-C   PC-D

  Collision domains:  4 (one per port)
  Broadcast domains:  1 (all ports, same VLAN)

  PC-A and PC-C can transmit simultaneously → NO collision
  PC-A sends broadcast → PC-B, PC-C, PC-D all receive it
```

### Switch with VLANs

```
# VLANs segment the switch into multiple broadcast domains
# Ports in different VLANs cannot communicate at Layer 2
# Inter-VLAN traffic requires a router or Layer 3 switch

         Switch (8 ports, 2 VLANs)
  ┌─────────────────────────────────────────────┐
  │  VLAN 10              │  VLAN 20            │
  │  ┌───┐ ┌───┐ ┌───┐   │  ┌───┐ ┌───┐ ┌───┐ │
  │  │ 1 │ │ 2 │ │ 3 │   │  │ 4 │ │ 5 │ │ 6 │ │
  └──┴───┴─┴───┴─┴───┴───┴──┴───┴─┴───┴─┴───┘ │
     │       │       │         │       │     │
    PC-A    PC-B   PC-C      PC-D    PC-E  PC-F

  Collision domains:  6 (one per port)
  Broadcast domains:  2 (one per VLAN)

  PC-A sends broadcast → PC-B, PC-C receive it
  PC-D, PC-E, PC-F do NOT receive VLAN 10 broadcast
```

### Router (Layer 3 — Network)

```
# Router breaks BOTH collision domains AND broadcast domains
# Each interface is a separate broadcast domain
# Broadcasts are NEVER forwarded between router interfaces
# (Exception: DHCP relay / ip helper-address forwards specific broadcasts)

  VLAN 10                Router              VLAN 20
  (Broadcast Domain 1)   ┌─────┐   (Broadcast Domain 2)
                         │     │
  ┌────────┐   ge-0/0/0 │     │ ge-0/0/1  ┌────────┐
  │ Switch ├────────────┤  R  ├───────────┤ Switch │
  │  (4PC) │            │     │           │  (4PC) │
  └────────┘            └─────┘           └────────┘

  Collision domains:  8 (switch ports) + 2 (router interfaces) = 10
  Broadcast domains:  2 (one per router interface)

  Broadcast from VLAN 10 → stays in VLAN 10
  Router does NOT forward ff:ff:ff:ff:ff:ff to VLAN 20
```

## Microsegmentation

```
# Full-duplex switched Ethernet = microsegmentation
# Each switch port is a dedicated collision domain with ONE device
# CSMA/CD is disabled on full-duplex links
# Collisions are impossible — TX and RX use separate wire pairs

# Half-duplex (hub era):
#   - Shared medium, CSMA/CD active
#   - Typical utilization: 30-40% before excessive collisions

# Full-duplex (modern switches):
#   - Dedicated TX and RX paths per port
#   - 100% utilization in both directions simultaneously
#   - No collisions, no backoff, no wasted bandwidth
#   - 1 Gbps full-duplex = 2 Gbps aggregate per port

# Duplex mismatch:
#   - One side full-duplex, other half-duplex
#   - Full-duplex side ignores collisions (doesn't back off)
#   - Half-duplex side sees "late collisions" and drops frames
#   - Result: intermittent packet loss, poor performance
#   - Common cause: auto-negotiation failure
```

## Broadcast Storms

```
# A broadcast storm occurs when broadcast frames loop endlessly
# through a Layer 2 network, consuming all available bandwidth

# Cause: physical loop without Spanning Tree Protocol (STP)
#
#   Switch A ──────── Switch B
#      │                  │
#      └──────────────────┘   ← redundant link = LOOP
#
# 1. PC sends broadcast frame
# 2. Switch A floods to all ports, including both links to Switch B
# 3. Switch B floods the copy back to Switch A
# 4. Switch A floods it again → infinite loop
# 5. Each iteration DOUBLES the traffic (geometric explosion)
# 6. Network saturates within seconds

# Prevention:
#   STP / RSTP / MSTP    — blocks redundant links (primary defense)
#   Storm control         — rate-limit broadcast/multicast per port
#   BPDU Guard            — shut down port if STP BPDU received on edge port
#   Loop Guard            — prevent blocked ports from going to forwarding

# Juniper storm control:
set interfaces ge-0/0/0 unit 0 family ethernet-switching storm-control default

# Cisco storm control:
storm-control broadcast level 10        # limit broadcast to 10% of port bandwidth
storm-control action shutdown           # err-disable port on violation
```

## ARP and DHCP — Broadcast in Practice

```
# ARP (Address Resolution Protocol):
#   "I need the MAC address for 192.168.1.1"
#   Src MAC: aa:bb:cc:dd:ee:01   Dst MAC: ff:ff:ff:ff:ff:ff
#   Every host in the broadcast domain receives and processes this
#   Only the owner of 192.168.1.1 sends a unicast reply

# In a /24 subnet with 200 hosts:
#   Each host ARPs periodically → ~200 broadcasts/minute baseline
#   Host cache miss or reboot → burst of ARP requests
#   ARP storm: IP scanning or misconfigured host → thousands of ARPs/sec

# DHCP:
#   DHCPDISCOVER: client → broadcast (no IP yet)
#   DHCPOFFER:    server → broadcast or unicast
#   DHCPREQUEST:  client → broadcast (inform all servers)
#   DHCPACK:      server → broadcast or unicast
#   4 broadcasts minimum per DHCP transaction
#   Monday morning: 500 PCs boot → 2,000+ DHCP broadcasts in minutes

# Gratuitous ARP:
#   Host announces its own IP/MAC mapping (broadcast)
#   Sent on boot, IP change, VRRP/HSRP failover
#   Every device in the broadcast domain processes it
```

## Why Smaller Broadcast Domains

```
# Performance:
#   Fewer broadcasts = less CPU interruption on every host
#   500 hosts × 5 bcast/sec = 2,500 interrupts/sec per host
#   50 hosts × 5 bcast/sec = 250 interrupts/sec per host

# Security:
#   Broadcasts reveal network topology (ARP → who's here)
#   Smaller domain = smaller blast radius for ARP spoofing
#   VLAN segmentation limits lateral movement

# Stability:
#   Broadcast storm in VLAN 10 does not affect VLAN 20
#   Fault isolation: misbehaving host affects fewer neighbors

# Scalability:
#   MAC address tables stay smaller per VLAN
#   STP topology is simpler per VLAN
#   DHCP pools and ARP tables are manageable

# Rule of thumb:
#   < 250 hosts per broadcast domain — ideal
#   250-500 — acceptable with modern hardware
#   > 500 — split into multiple VLANs
#   > 1,000 — guaranteed performance problems
```

## Device Summary

```
Device    OSI Layer   Collision Domains    Broadcast Domains
──────────────────────────────────────────────────────────────
Hub       1           1 (shared)           1 (all ports)
Bridge    2           1 per port           1 (all ports)
Switch    2           1 per port           1 per VLAN
Router    3           1 per interface      1 per interface

# Hub:    extends both collision and broadcast domains
# Switch: breaks collision domains, passes broadcasts within VLAN
# Router: breaks both collision and broadcast domains
```

## Juniper Configuration

```bash
# VLAN configuration (Junos) — create broadcast domain boundaries
set vlans SALES vlan-id 10
set vlans ENGINEERING vlan-id 20

# Assign access ports
set interfaces ge-0/0/0 unit 0 family ethernet-switching vlan members SALES
set interfaces ge-0/0/1 unit 0 family ethernet-switching vlan members ENGINEERING

# Trunk between switches (carries multiple broadcast domains)
set interfaces ge-0/0/23 unit 0 family ethernet-switching interface-mode trunk
set interfaces ge-0/0/23 unit 0 family ethernet-switching vlan members [SALES ENGINEERING]

# Storm control — protect against broadcast storms
set interfaces ge-0/0/0 unit 0 family ethernet-switching storm-control default

# Inter-VLAN routing (L3 switch / IRB)
set interfaces irb unit 10 family inet address 192.168.10.1/24
set interfaces irb unit 20 family inet address 192.168.20.1/24
set vlans SALES l3-interface irb.10
set vlans ENGINEERING l3-interface irb.20
```

## Tips

- Full-duplex point-to-point links between a host and a switch port eliminate collisions entirely. CSMA/CD is disabled and both sides transmit simultaneously. Every modern Ethernet connection operates this way — collisions only matter on shared media (hubs, coax).
- A VLAN is a broadcast domain. Creating VLAN 10 and VLAN 20 on a switch is functionally equivalent to using two separate physical switches. Broadcasts in VLAN 10 never reach VLAN 20 without a router.
- Duplex mismatch is the most common cause of "collisions" on modern networks. If one side negotiates full-duplex and the other half-duplex, the half-duplex side reports late collisions and CRC errors. Force both sides to the same setting or verify auto-negotiation.
- Routers are the ultimate broadcast domain boundary. Even if two interfaces are on the same physical switch (Layer 3 switch with SVIs), each SVI is a separate broadcast domain. This is why inter-VLAN routing requires Layer 3.
- Broadcast storms can saturate a 10 Gbps link in under a second. Always run STP/RSTP and enable storm control on access ports. A single cable loop in a wiring closet without STP will take down the entire VLAN.
- On the JNCIA-Junos exam, remember: hubs extend collision domains, switches break collision domains but extend broadcast domains within a VLAN, and routers break both. Count collision domains by counting switch ports (plus one per hub group) and broadcast domains by counting VLANs or router interfaces.
- ARP broadcasts scale linearly with the number of hosts. A /16 subnet with 65,000 hosts generates enough ARP traffic to visibly degrade performance. Keep subnets at /24 or smaller for access networks.
- DHCP relay (`ip helper-address` on Cisco, `forwarding-options helpers` on Junos) is the controlled exception to "routers don't forward broadcasts." It converts specific broadcast types to unicast for cross-VLAN delivery.

## See Also

- ethernet, vlan, stp, arp, dhcp, subnetting

## References

- [IEEE 802.3-2022 — Ethernet Standard (CSMA/CD)](https://standards.ieee.org/standard/802_3-2022.html)
- [IEEE 802.1Q-2022 — VLANs and Bridges](https://standards.ieee.org/standard/802_1Q-2022.html)
- [Juniper — Understanding Broadcast and Collision Domains](https://www.juniper.net/documentation/us/en/software/junos/switching/topics/concept/collision-broadcast-domains.html)
- [Juniper — JNCIA-Junos Study Guide](https://www.juniper.net/us/en/training/certification/tracks/junos/jncia-junos.html)
- [RFC 826 — ARP](https://www.rfc-editor.org/rfc/rfc826)
- [RFC 2131 — DHCP](https://www.rfc-editor.org/rfc/rfc2131)
