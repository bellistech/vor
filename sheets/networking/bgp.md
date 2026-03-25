# BGP (Border Gateway Protocol)

Path-vector exterior gateway protocol used to exchange routing information between autonomous systems on the internet and within large-scale networks.

## Concepts

### iBGP vs eBGP

- **eBGP:** Between different autonomous systems; TTL=1 by default; next-hop changes
- **iBGP:** Within the same AS; TTL=255; next-hop preserved (need next-hop-self); requires full mesh or route reflectors
- iBGP does not modify AS-path, so loop prevention relies on not accepting routes with own AS in path (for eBGP) and split-horizon (for iBGP)

### Path Attributes and Best Path Selection

Order of preference (top wins first):

1. **Weight** (Cisco/local, higher better, default 0)
2. **Local Preference** (iBGP, higher better, default 100)
3. **Locally originated** (network/aggregate/redistribute)
4. **AS-path length** (shorter wins)
5. **Origin** (IGP < EGP < Incomplete)
6. **MED** (lower better, compared within same neighbor AS)
7. **eBGP over iBGP**
8. **Lowest IGP metric to next-hop**
9. **Oldest eBGP route**
10. **Lowest router-id**

### Route Reflectors

```
# Eliminates iBGP full-mesh requirement
# RR reflects routes from clients to other clients and non-clients
# Cluster-id identifies the RR cluster for loop prevention
router bgp 65000
 neighbor 10.0.0.2 remote-as 65000
 address-family ipv4 unicast
  neighbor 10.0.0.2 route-reflector-client
```

### Confederations

```
# Split a large AS into sub-ASes that appear as one AS externally
router bgp 65501
 bgp confederation identifier 65000
 bgp confederation peers 65502 65503
```

## FRRouting Configuration

### Basic eBGP Setup

```
router bgp 65001
 bgp router-id 10.0.0.1
 # eBGP neighbor in a different AS
 neighbor 192.168.1.2 remote-as 65002
 # Description for clarity
 neighbor 192.168.1.2 description upstream-provider
 # Announce networks (must exist in RIB or use static route)
 address-family ipv4 unicast
  network 10.10.0.0/16
  # Redistribute connected routes with a route-map filter
  redistribute connected route-map CONNECTED-OUT
 exit-address-family
```

### iBGP Setup with Next-Hop-Self

```
router bgp 65001
 # iBGP neighbor (same AS)
 neighbor 10.0.0.2 remote-as 65001
 # Update source for iBGP (use loopback for stability)
 neighbor 10.0.0.2 update-source lo
 address-family ipv4 unicast
  # Rewrite next-hop to self so iBGP peers can reach external prefixes
  neighbor 10.0.0.2 next-hop-self
 exit-address-family
```

### Prefix Lists

```
# Define prefix filters
ip prefix-list ALLOW-DEFAULT seq 5 permit 0.0.0.0/0
ip prefix-list DENY-RFC1918 seq 10 deny 10.0.0.0/8 le 32
ip prefix-list DENY-RFC1918 seq 20 deny 172.16.0.0/12 le 32
ip prefix-list DENY-RFC1918 seq 30 deny 192.168.0.0/16 le 32
ip prefix-list DENY-RFC1918 seq 100 permit 0.0.0.0/0 le 32

# Apply to neighbor
router bgp 65001
 address-family ipv4 unicast
  neighbor 192.168.1.2 prefix-list DENY-RFC1918 in
  neighbor 192.168.1.2 prefix-list ALLOW-DEFAULT out
```

### Route Maps

```
# Match and modify attributes
route-map SET-LOCAL-PREF permit 10
 match ip address prefix-list FROM-PROVIDER
 set local-preference 200

route-map SET-LOCAL-PREF permit 20
 # Default permit for everything else with lower preference
 set local-preference 100

# AS-path prepending to make a path less preferred
route-map PREPEND-OUT permit 10
 set as-path prepend 65001 65001 65001

# Apply route-map to neighbor
router bgp 65001
 address-family ipv4 unicast
  neighbor 192.168.1.2 route-map SET-LOCAL-PREF in
  neighbor 192.168.2.2 route-map PREPEND-OUT out
```

### Communities

```
# Define community-list
bgp community-list standard BLACKHOLE permit 65001:666
bgp community-list standard CUSTOMER permit 65001:100

# Route-map using communities
route-map TAG-CUSTOMER permit 10
 set community 65001:100

route-map APPLY-COMMUNITY permit 10
 match community BLACKHOLE
 set local-preference 0

# Send communities to neighbor
router bgp 65001
 address-family ipv4 unicast
  neighbor 192.168.1.2 send-community both
```

### AS-Path Access Lists

```
# Match specific AS paths
bgp as-path access-list ORIGIN-65002 permit ^65002$
bgp as-path access-list TRANSIT-65002 permit _65002_

route-map FILTER-ASPATH permit 10
 match as-path ORIGIN-65002
```

### Local Preference, MED, Weight

```
# Local preference: higher = more preferred (iBGP scope)
route-map PREFER-PRIMARY permit 10
 set local-preference 200

# MED: lower = more preferred (sent to eBGP neighbor, compared within same AS)
route-map SET-MED permit 10
 set metric 50

# Weight: Cisco-style local preference (local to router, highest wins)
router bgp 65001
 neighbor 192.168.1.2 weight 200
```

### MP-BGP Address Families

```
router bgp 65001
 # IPv6 unicast
 address-family ipv6 unicast
  neighbor 2001:db8::2 activate
  network 2001:db8:1::/48
 exit-address-family

 # VPNv4 for MPLS L3VPN
 address-family vpnv4 unicast
  neighbor 10.0.0.2 activate
  neighbor 10.0.0.2 next-hop-self
 exit-address-family

 # EVPN for VXLAN
 address-family l2vpn evpn
  neighbor 10.0.0.2 activate
  advertise-all-vni
 exit-address-family
```

### Graceful Restart

```
router bgp 65001
 bgp graceful-restart
 # Restart timer (how long to wait for peer to re-establish)
 bgp graceful-restart restart-time 120
 # Stalepath timer (how long to keep stale routes)
 bgp graceful-restart stalepath-time 360
```

### BFD Integration

```
router bgp 65001
 neighbor 192.168.1.2 bfd
 # With specific BFD profile
 neighbor 192.168.1.2 bfd profile FAST-DETECT
```

## Cisco IOS Equivalents

```
router bgp 65001
 bgp router-id 10.0.0.1
 neighbor 192.168.1.2 remote-as 65002
 neighbor 10.0.0.2 remote-as 65001
 neighbor 10.0.0.2 update-source Loopback0
 neighbor 10.0.0.2 next-hop-self
 !
 address-family ipv4 unicast
  network 10.10.0.0 mask 255.255.0.0
  neighbor 192.168.1.2 activate
  neighbor 192.168.1.2 prefix-list DENY-RFC1918 in
  neighbor 192.168.1.2 route-map SET-LOCAL-PREF in
  neighbor 192.168.1.2 send-community both
  neighbor 10.0.0.2 route-reflector-client
 exit-address-family
```

## Show Commands

```bash
# BGP summary (neighbor states, prefixes received, uptime)
vtysh -c "show bgp summary"

# Detailed neighbor info (timers, capabilities, AFI/SAFI)
vtysh -c "show bgp neighbors"
vtysh -c "show bgp neighbors 192.168.1.2"

# All received BGP routes
vtysh -c "show bgp ipv4 unicast"

# Routes from a specific neighbor
vtysh -c "show bgp ipv4 unicast neighbors 192.168.1.2 received-routes"
vtysh -c "show bgp ipv4 unicast neighbors 192.168.1.2 advertised-routes"

# Specific prefix lookup
vtysh -c "show bgp ipv4 unicast 10.10.0.0/16"

# BGP paths with all attributes
vtysh -c "show bgp ipv4 unicast 10.10.0.0/16 bestpath"

# Community-based lookup
vtysh -c "show bgp community 65001:100"

# BGP RIB failure (routes not installed)
vtysh -c "show bgp ipv4 unicast rib-failure"

# Soft reset (re-apply policy without dropping session)
vtysh -c "clear bgp ipv4 unicast 192.168.1.2 soft in"
vtysh -c "clear bgp ipv4 unicast 192.168.1.2 soft out"
```

## Troubleshooting

### Session Not Establishing

```bash
# Check TCP connectivity on port 179
nc -zv 192.168.1.2 179

# Verify neighbor config (remote-as, IP address)
vtysh -c "show bgp neighbors 192.168.1.2" | grep -i "bgp state"

# Check for ACL/firewall blocking TCP 179
iptables -L -n | grep 179

# For eBGP multihop (peers not directly connected)
# neighbor 192.168.1.2 ebgp-multihop 2
```

### Routes Not Being Advertised

```bash
# Verify the route exists in the RIB
vtysh -c "show ip route 10.10.0.0/16"

# Check if network statement matches exactly (mask must match)
vtysh -c "show running-config" | grep "network.*10.10"

# Confirm route-map/prefix-list is not filtering
vtysh -c "show bgp ipv4 unicast neighbors 192.168.1.2 advertised-routes"

# Check if neighbor has reached max-prefix limit
vtysh -c "show bgp neighbors 192.168.1.2" | grep -i "prefix"
```

### Next-Hop Unreachable

```bash
# iBGP routes may have eBGP next-hop that internal routers cannot reach
# Fix: use next-hop-self on iBGP peerings
vtysh -c "show bgp ipv4 unicast" | grep -i "inaccessible"

# Verify next-hop is in the IGP
vtysh -c "show ip route <next-hop-ip>"
```

## Tips

- Always filter inbound and outbound on eBGP sessions; never accept or send the full table without intent.
- Use `maximum-prefix` on all eBGP sessions to protect against route leaks flooding your RIB.
- Set `next-hop-self` on all iBGP peerings unless you have a specific reason not to (e.g., route server at IXP).
- For iBGP, either use route reflectors or full mesh; partial mesh causes route black holes.
- The `network` statement requires an exact RIB match including mask; a /24 in RIB won't match a `network x.x.x.x/16`.
- Use loopback addresses for iBGP peering with `update-source` for resilience against single link failures.
- When prepending, 3 prepends is usually sufficient; more than that has diminishing returns and clutters AS-paths.
- After changing policy (route-map, prefix-list), apply with `clear bgp soft in/out` rather than hard reset.
- Keep BGP timers at defaults (60s keepalive, 180s hold) and use BFD for fast failure detection instead.
- MED is non-transitive and only compared between paths from the same neighbor AS by default; use `bgp always-compare-med` to compare across ASes.
- Document every route-map and prefix-list with comments; BGP policy is the most common source of outages.
- Use `bgp graceful-restart` to minimize traffic loss during planned maintenance or software upgrades.

## References

- [RFC 4271 — A Border Gateway Protocol 4 (BGP-4)](https://www.rfc-editor.org/rfc/rfc4271)
- [RFC 4456 — BGP Route Reflection](https://www.rfc-editor.org/rfc/rfc4456)
- [RFC 4760 — Multiprotocol Extensions for BGP-4](https://www.rfc-editor.org/rfc/rfc4760)
- [RFC 7911 — Advertisement of Multiple Paths in BGP (ADD-PATH)](https://www.rfc-editor.org/rfc/rfc7911)
- [RFC 8092 — BGP Large Communities](https://www.rfc-editor.org/rfc/rfc8092)
- [RFC 9234 — Route Leak Prevention and Detection Using Roles](https://www.rfc-editor.org/rfc/rfc9234)
- [FRRouting BGP Documentation](https://docs.frrouting.org/en/latest/bgp.html)
- [BIRD Internet Routing Daemon — BGP](https://bird.network.cz/?get_doc&v=20&f=bird-6.html)
- [Cisco BGP Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/iproute_bgp/configuration/xe-16/irg-xe-16-book.html)
- [Juniper BGP Configuration Guide](https://www.juniper.net/documentation/us/en/software/junos/bgp/topics/topic-map/bgp-overview.html)
- [Arista EOS BGP Configuration Guide](https://www.arista.com/en/um-eos/eos-border-gateway-protocol-bgp)
