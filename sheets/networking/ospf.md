# OSPF (Open Shortest Path First)

Link-state interior gateway protocol that uses SPF algorithm to compute shortest paths within an autonomous system.

## Concepts

### Areas and Hierarchy

- **Area 0 (backbone):** All other areas must connect to area 0
- **Regular areas:** Connected to backbone, receive all LSA types
- **Stub area:** No external LSAs (type 5), uses default route instead
- **Totally stubby area:** No external or inter-area LSAs (type 3/5), default only
- **NSSA:** Like stub but allows limited external routes via type 7 LSAs
- **Totally NSSA:** NSSA + no type 3 inter-area LSAs

### LSA Types

| Type | Name                  | Scope        |
|------|-----------------------|--------------|
| 1    | Router LSA            | Intra-area   |
| 2    | Network LSA           | Intra-area   |
| 3    | Summary LSA           | Inter-area   |
| 4    | ASBR Summary LSA      | Inter-area   |
| 5    | AS External LSA       | AS-wide      |
| 7    | NSSA External LSA     | NSSA area    |

### DR/BDR Election

- Elected on multi-access segments (Ethernet) to reduce adjacency count
- Highest priority wins (default 1, range 0-255, 0 = never DR)
- Ties broken by highest router-id
- Election is non-preemptive: won't change until DR/BDR goes down

### Cost Calculation

```
# OSPF cost = reference bandwidth / interface bandwidth
# Default reference: 100 Mbps
# 100 Mbps link = cost 1, 10 Mbps = cost 10, 1 Gbps = cost 1
# Increase reference for modern networks:
auto-cost reference-bandwidth 10000   # 10 Gbps reference
```

## FRRouting Configuration

### Basic Router Setup

```bash
vtysh -c "configure terminal"
```

```
router ospf
 ospf router-id 10.0.0.1
 # Advertise networks into OSPF areas
 network 10.0.0.0/24 area 0
 network 10.1.0.0/24 area 1
 # Prevent OSPF hellos on LAN-facing interfaces
 passive-interface eth2
 # Inject a default route into OSPF (if one exists in RIB)
 default-information originate
 # Always inject default route even if not in RIB
 default-information originate always
 # Redistribute connected/static routes
 redistribute connected
 redistribute static route-map STATIC-INTO-OSPF
 # Adjust reference bandwidth for 10G fabric
 auto-cost reference-bandwidth 10000
```

### Interface-Level Config

```
interface eth0
 # Override automatic cost
 ip ospf cost 100
 # Set OSPF area directly on interface (alternative to network statement)
 ip ospf area 0
 # Adjust timers (hello must match neighbors)
 ip ospf hello-interval 10
 ip ospf dead-interval 40
 # Set priority for DR election (0 = never become DR)
 ip ospf priority 200
 # Enable BFD for fast failure detection
 ip ospf bfd
```

### Area Configuration

```
router ospf
 # Stub area: blocks type 5 LSAs, injects default
 area 1 stub
 # Totally stubby: blocks type 3 and 5, only default route
 area 1 stub no-summary
 # NSSA: allows external routes via type 7 LSAs
 area 2 nssa
 # Totally NSSA
 area 2 nssa no-summary
 # Route summarization at ABR (inter-area)
 area 0 range 10.0.0.0/16
 # Summarize external routes at ASBR
 summary-address 172.16.0.0/16
```

### Virtual Links

```
router ospf
 # Connect a disconnected area through a transit area
 # Configured on both ABRs, use each other's router-id
 area 1 virtual-link 10.0.0.2
```

### Authentication

```
interface eth0
 # Simple password authentication
 ip ospf authentication
 ip ospf authentication-key MySecret
 # MD5 authentication (preferred)
 ip ospf authentication message-digest
 ip ospf message-digest-key 1 md5 MyMD5Secret

router ospf
 # Area-level authentication (all interfaces in the area)
 area 0 authentication message-digest
```

### SPF Throttle Timers

```
router ospf
 # timers throttle spf <initial-delay> <min-hold> <max-wait> (ms)
 # Start SPF 50ms after trigger, backoff to 5s max
 timers throttle spf 50 200 5000
 # LSA generation throttle
 timers throttle lsa all 0 200 5000
```

## Cisco IOS Equivalents

```
! Cisco uses wildcard masks instead of prefix lengths
router ospf 1
 router-id 10.0.0.1
 network 10.0.0.0 0.0.0.255 area 0
 passive-interface GigabitEthernet0/2
 default-information originate always
 area 1 stub no-summary
 area 1 virtual-link 10.0.0.2

interface GigabitEthernet0/0
 ip ospf 1 area 0
 ip ospf cost 100
 ip ospf authentication message-digest
 ip ospf message-digest-key 1 md5 MyMD5Secret
```

## Show Commands

```bash
# OSPF process overview (router-id, areas, timers, SPF stats)
vtysh -c "show ip ospf"

# Neighbor table (state, DR/BDR, dead timer)
vtysh -c "show ip ospf neighbor"

# Full LSDB (all LSA types)
vtysh -c "show ip ospf database"

# Specific LSA type
vtysh -c "show ip ospf database router"
vtysh -c "show ip ospf database network"
vtysh -c "show ip ospf database external"

# Interface details (area, cost, state, timers, neighbor count)
vtysh -c "show ip ospf interface"
vtysh -c "show ip ospf interface eth0"

# OSPF routes in the routing table
vtysh -c "show ip ospf route"

# Border routers (ABR/ASBR)
vtysh -c "show ip ospf border-routers"
```

## Troubleshooting

### Neighbor Stuck in Init/2-Way

```bash
# Check hello/dead timer mismatch (must match on both sides)
vtysh -c "show ip ospf interface eth0" | grep -i "timer"

# Check area mismatch
vtysh -c "show ip ospf interface eth0" | grep -i "area"

# Check authentication mismatch
vtysh -c "show ip ospf interface eth0" | grep -i "auth"
```

### Neighbor Stuck in ExStart/Exchange

```bash
# Usually MTU mismatch — both sides must agree
ip link show eth0 | grep mtu

# Workaround: disable MTU check (not recommended for production)
# interface eth0
#  ip ospf mtu-ignore
```

### Routes Not Appearing

```bash
# Verify network statement covers the interface
vtysh -c "show ip ospf interface" | grep -i "area"

# Check if interface is passive (passive won't form adjacencies)
vtysh -c "show ip ospf interface" | grep -i "passive"

# Verify LSAs are in the database
vtysh -c "show ip ospf database"

# Check for filtering or summarization hiding routes
vtysh -c "show ip ospf route"
```

## Tips

- Always set `router-id` explicitly; auto-selection from interfaces can change unexpectedly after reboots.
- Increase `auto-cost reference-bandwidth` on modern networks so 1G and 10G links get different costs.
- Use `passive-interface default` then selectively enable OSPF on point-to-point links to reduce attack surface.
- On point-to-point links (e.g., tunnel, direct cable), set `ip ospf network point-to-point` to skip DR election and speed convergence.
- MTU mismatch is the number one cause of stuck adjacencies in ExStart/Exchange state.
- OSPF hello/dead timers must match on both ends of a link; default is 10/40 on broadcast, 30/120 on NBMA.
- In multi-vendor environments, watch for differences in stub/NSSA default route metric behavior.
- Use BFD alongside OSPF for sub-second failure detection rather than lowering hello timers aggressively.
- Virtual links are a band-aid; redesign the network if possible to avoid them.
- When redistributing into OSPF, always use a route-map to prevent accidental route leaks.

## References

- [RFC 2328 — OSPF Version 2](https://www.rfc-editor.org/rfc/rfc2328)
- [RFC 5340 — OSPF for IPv6 (OSPFv3)](https://www.rfc-editor.org/rfc/rfc5340)
- [RFC 3630 — Traffic Engineering Extensions to OSPF](https://www.rfc-editor.org/rfc/rfc3630)
- [RFC 5243 — OSPF Database Exchange Summary List Optimization](https://www.rfc-editor.org/rfc/rfc5243)
- [RFC 6549 — OSPFv2 Multi-Instance Extensions](https://www.rfc-editor.org/rfc/rfc6549)
- [FRRouting OSPF Documentation](https://docs.frrouting.org/en/latest/ospfd.html)
- [FRRouting OSPFv3 Documentation](https://docs.frrouting.org/en/latest/ospf6d.html)
- [BIRD Internet Routing Daemon — OSPF](https://bird.network.cz/?get_doc&v=20&f=bird-6.html)
- [Cisco OSPF Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/iproute_ospf/configuration/xe-16/iro-xe-16-book.html)
- [Juniper OSPF Configuration Guide](https://www.juniper.net/documentation/us/en/software/junos/ospf/topics/topic-map/ospf-overview.html)
- [Arista EOS OSPF Configuration Guide](https://www.arista.com/en/um-eos/eos-open-shortest-path-first-version-2)
