# RIP (Routing Information Protocol)

Distance-vector routing protocol using hop count as its metric, suitable for small or legacy networks with simple topologies.

## Concepts

### RIPv1 vs RIPv2 vs RIPng

| Feature          | RIPv1          | RIPv2          | RIPng          |
|------------------|----------------|----------------|----------------|
| Addressing       | Classful       | Classless/CIDR | IPv6           |
| Updates          | Broadcast      | Multicast 224.0.0.9 | Multicast ff02::9 |
| Authentication   | None           | Simple/MD5     | IPsec          |
| VLSM support     | No             | Yes            | Yes            |
| Max hop count    | 15             | 15             | 15             |

### Distance-Vector Mechanics

- **Hop count metric:** Each router hop = 1; max 15 hops, 16 = unreachable
- **Split horizon:** Don't advertise a route back out the interface it was learned on
- **Poison reverse:** Advertise learned routes back with metric 16 (unreachable)
- **Triggered updates:** Send immediate update on topology change instead of waiting for periodic timer
- **Route poisoning:** Set metric to 16 for a failed route to propagate the failure quickly

### Timers

| Timer    | Default  | Purpose                                           |
|----------|----------|---------------------------------------------------|
| Update   | 30s      | Periodic routing table broadcast interval          |
| Invalid  | 180s     | Route marked invalid if no update received         |
| Holddown | 180s     | Ignore inferior routes for this period after failure|
| Flush    | 240s     | Route removed from table entirely                  |

## FRRouting Configuration

### Basic RIPv2 Setup

```
router rip
 # Use RIPv2 (classless, multicast)
 version 2
 # Enable RIP on interfaces matching these networks
 network 10.0.0.0/8
 network 192.168.1.0/24
 # Prevent RIP updates on LAN-facing interfaces
 passive-interface eth2
 # Redistribute other routing sources into RIP
 redistribute connected
 redistribute static route-map STATIC-TO-RIP
 # Inject a default route
 default-information originate
 # Adjust timers (update, invalid, holddown, flush)
 timers basic 30 180 180 240
```

### Interface-Level Options

```
interface eth0
 # Force RIPv2 on a specific interface
 ip rip send version 2
 ip rip receive version 2
 # Set per-interface metric offset
 # Adds this value to the metric of learned routes
 ip rip metric-offset 2
```

### Authentication

```
interface eth0
 # MD5 authentication (recommended over simple)
 ip rip authentication mode md5
 ip rip authentication key-chain RIP-KEYS

key chain RIP-KEYS
 key 1
  key-string MyRIPSecret
```

### Route Filtering

```
# Distribute-list to filter routes
router rip
 # Filter inbound routes with access-list
 distribute-list prefix ALLOW-NETS in eth0
 # Filter outbound routes
 distribute-list prefix DENY-DEFAULT out eth0

ip prefix-list ALLOW-NETS seq 10 permit 10.0.0.0/8 le 24
ip prefix-list DENY-DEFAULT seq 10 deny 0.0.0.0/0
ip prefix-list DENY-DEFAULT seq 20 permit 0.0.0.0/0 le 32
```

### RIPng (IPv6)

```
router ripng
 # Enable RIPng on interfaces
 network eth0
 network eth1
 # Redistribute connected
 redistribute connected

interface eth0
 ipv6 router ripng
```

## Cisco IOS Equivalents

```
router rip
 version 2
 network 10.0.0.0
 network 192.168.1.0
 passive-interface GigabitEthernet0/2
 no auto-summary
 default-information originate
 redistribute static route-map STATIC-TO-RIP
 timers basic 30 180 180 240

interface GigabitEthernet0/0
 ip rip send version 2
 ip rip receive version 2
 ip rip authentication mode md5
 ip rip authentication key-chain RIP-KEYS
```

## Show Commands

```bash
# RIP routing table
vtysh -c "show ip rip"

# RIP process status (timers, version, networks)
vtysh -c "show ip rip status"

# RIPng routes (IPv6)
vtysh -c "show ipv6 ripng"
vtysh -c "show ipv6 ripng status"

# Debug RIP updates (use briefly, very verbose)
vtysh -c "debug rip events"
vtysh -c "debug rip packet"
```

## Troubleshooting

```bash
# Verify RIP is sending/receiving on the right interfaces
vtysh -c "show ip rip status" | grep -i "sending\|interface"

# Check version mismatch (v1 and v2 won't interoperate)
vtysh -c "show ip rip status" | grep -i "version"

# Check authentication mismatch
# Both sides must use the same mode and key

# Verify no distribute-list is blocking needed routes
vtysh -c "show ip rip" | grep -i "metric"

# Confirm routes are not at metric 16 (unreachable)
vtysh -c "show ip rip" | grep " 16 "
```

## Tips

- RIP is only appropriate for small, flat networks (under 15 hops); use OSPF or IS-IS for anything larger.
- Always use RIPv2 over RIPv1; classful routing with RIPv1 causes silent failures with VLSM/CIDR.
- Use `no auto-summary` on Cisco to prevent classful summarization from hiding subnets.
- The 15-hop limit is a hard ceiling; if your network could ever exceed it, migrate to a link-state protocol.
- RIP converges slowly compared to OSPF/IS-IS; expect 30-second update intervals plus holddown time before re-convergence.
- Use `passive-interface` on all interfaces that don't need to form RIP adjacencies to reduce unnecessary multicast traffic.
- MD5 authentication prevents rogue routers from injecting bad routes, even on internal networks.
- RIP is still found in legacy embedded systems, industrial networks, and some ISP CPE where simplicity outweighs performance.
- When migrating away from RIP, run OSPF alongside with higher administrative distance first, then remove RIP once validated.

## References

- [RFC 2453 — RIP Version 2](https://www.rfc-editor.org/rfc/rfc2453)
- [RFC 1058 — Routing Information Protocol (RIPv1)](https://www.rfc-editor.org/rfc/rfc1058)
- [RFC 2080 — RIPng for IPv6](https://www.rfc-editor.org/rfc/rfc2080)
- [RFC 4822 — RIPv2 Cryptographic Authentication](https://www.rfc-editor.org/rfc/rfc4822)
- [FRRouting RIP Documentation](https://docs.frrouting.org/en/latest/ripd.html)
- [FRRouting RIPng Documentation](https://docs.frrouting.org/en/latest/ripngd.html)
- [Cisco RIP Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/iproute_rip/configuration/xe-16/irr-xe-16-book.html)
- [Juniper RIP Configuration Guide](https://www.juniper.net/documentation/us/en/software/junos/rip/topics/topic-map/rip-overview.html)
