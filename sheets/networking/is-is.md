# IS-IS (Intermediate System to Intermediate System)

Link-state IGP originally designed for OSI networks, widely used in service provider and large enterprise networks for its scalability and fast convergence.

## Concepts

### Levels and Hierarchy

- **Level 1 (L1):** Intra-area routing (like OSPF intra-area)
- **Level 2 (L2):** Inter-area / backbone routing (like OSPF area 0)
- **L1/L2 router:** Participates in both levels, connects L1 areas to the L2 backbone
- **L1-only router:** Reaches other areas via the nearest L1/L2 router (default route)
- No requirement for a backbone "area 0"; the L2 subdomain IS the backbone

### NET Addressing (Network Entity Title)

```
# Format: AFI.AreaID.SystemID.SEL
# Example: 49.0001.0010.0000.0001.00
#   49       = AFI (private)
#   0001     = Area ID
#   0010.0000.0001 = System ID (often derived from loopback IP)
#   00       = SEL (always 00 for IS-IS)
```

### DIS (Designated Intermediate System)

- Equivalent of OSPF DR on multi-access networks
- Elected by highest interface priority (default 64), then highest SNPA
- Unlike OSPF DR, DIS election IS preemptive
- DIS creates the pseudonode LSP for the multi-access segment

### Metrics

- **Narrow metrics:** Original, 6-bit (0-63), limits network diameter
- **Wide metrics:** Extended, 24-bit (0-16777215), required for TE and modern networks
- **Transition:** Run both during migration with `metric-style transition`

### TLVs (Type-Length-Value)

- IS-IS uses TLVs for extensibility (like BGP path attributes)
- Key TLVs: IS Neighbors (2/22), IP Reachability (128/135), Hostname (137)
- Wide metrics use "extended" TLV numbers (22, 135)

## FRRouting Configuration

### Basic Setup

```
router isis CORE
 # NET address (must be unique per router)
 net 49.0001.0010.0000.0001.00
 # Router participates in both L1 and L2
 is-type level-1-2
 # Use wide metrics (required for modern deployments)
 metric-style wide
 # Log adjacency changes
 log-adjacency-changes
```

### Interface Configuration

```
interface eth0
 # Enable IS-IS on this interface
 ip router isis CORE
 # Set IS-IS metric (wide metric range: 0-16777215)
 isis metric 100
 # Circuit type (level-1, level-2-only, level-1-2)
 isis circuit-type level-2-only
 # Passive: advertise prefix but don't form adjacencies
 isis passive
 # Set priority for DIS election (0-127, default 64)
 isis priority 200
 # Point-to-point (use on P2P links to skip DIS election)
 isis network point-to-point
 # BFD integration
 isis bfd

interface lo
 ip router isis CORE
 isis passive
```

### Authentication

```
router isis CORE
 # Area-level authentication (L1)
 area-password clear MyL1Secret
 # Domain-level authentication (L2)
 domain-password clear MyL2Secret

# Interface-level MD5 authentication
interface eth0
 isis password md5 MyInterfaceSecret
```

### Route Leaking Between Levels

```
router isis CORE
 # Redistribute L2 routes into L1 (allows L1-only routers to see external routes)
 redistribute isis level-2 into level-1
 # With route-map for selective leaking
 redistribute isis level-2 into level-1 route-map L2-TO-L1
```

### Overload Bit

```
router isis CORE
 # Set overload bit: router is in the SPF tree but not used for transit
 # Useful during maintenance or boot-up
 set-overload-bit
 # Auto-clear after timeout (seconds)
 set-overload-bit on-startup 300
```

### Multi-Topology and IPv6

```
router isis CORE
 # Enable multi-topology for IPv6
 topology ipv6-unicast

interface eth0
 # Enable IPv6 IS-IS on this interface
 ipv6 router isis CORE
```

## Cisco IOS Equivalents

```
router isis CORE
 net 49.0001.0010.0000.0001.00
 is-type level-1-2
 metric-style wide
 log-adjacency-changes
 !
 address-family ipv6 unicast
  multi-topology
 exit-address-family

interface GigabitEthernet0/0
 ip router isis CORE
 isis metric 100
 isis circuit-type level-2-only
 isis network point-to-point
 isis authentication mode md5
 isis authentication key-chain ISIS-KEYS

interface Loopback0
 ip router isis CORE
 isis passive-interface
```

## Show Commands

```bash
# IS-IS neighbor table (state, circuit type, holdtime)
vtysh -c "show isis neighbor"
vtysh -c "show isis neighbor detail"

# Link-state database (all LSPs)
vtysh -c "show isis database"
vtysh -c "show isis database detail"

# IS-IS topology (SPF tree)
vtysh -c "show isis topology"

# IS-IS routes
vtysh -c "show isis route"

# Interface IS-IS status
vtysh -c "show isis interface"
vtysh -c "show isis interface detail"

# IS-IS summary info
vtysh -c "show isis summary"

# SPF computation log
vtysh -c "show isis spf-delay-ietf"
```

## IS-IS vs OSPF Comparison

| Feature              | IS-IS                        | OSPF                        |
|----------------------|------------------------------|-----------------------------|
| Protocol base        | OSI (CLNS)                   | IP                          |
| Encapsulation        | Directly on L2               | IP protocol 89              |
| Area boundaries      | On links (between routers)   | On routers (ABR)            |
| Hierarchy            | L1/L2                        | Area 0 + other areas        |
| Backbone requirement | L2 subdomain                 | Area 0 required             |
| Extensibility        | TLVs (easy to extend)        | New LSA types (harder)      |
| IPv6 support         | Multi-topology or single     | OSPFv3 (separate process)   |
| Metric range         | Wide: 0-16M                  | 0-65535                     |
| SP network usage     | Very common                  | Less common                 |

## Troubleshooting

```bash
# Adjacency not forming: check NET area mismatch (L1 requires same area)
vtysh -c "show isis interface" | grep -i "area\|net"

# Check circuit type mismatch (L1-only won't peer with L2-only)
vtysh -c "show isis interface" | grep -i "circuit"

# Verify authentication matches on both sides
vtysh -c "show isis interface detail" | grep -i "auth"

# MTU issues: IS-IS PDUs can be large, especially with many TLVs
# Check originatingLSPBufferSize matches on both sides

# Verify metric-style matches (narrow vs wide must be consistent)
vtysh -c "show isis summary" | grep -i "metric"

# Check for overload bit preventing transit
vtysh -c "show isis database detail" | grep -i "overload"
```

## Tips

- IS-IS runs directly on L2 (not over IP), making it immune to IP routing problems during convergence.
- Always use `metric-style wide`; narrow metrics (0-63) are insufficient for any real network.
- Use `isis network point-to-point` on all point-to-point links to avoid unnecessary DIS election.
- Area boundaries in IS-IS are on links, not routers; an L1/L2 router belongs to exactly one area.
- IS-IS is preferred in SP networks because of TLV extensibility (easy to add new features like TE, SR).
- Set the overload bit during router boot to allow IGP/LDP to converge before the router starts forwarding transit traffic.
- Unlike OSPF, IS-IS DIS election is preemptive; a higher-priority router joining will take over DIS immediately.
- Use route leaking carefully; uncontrolled L2-to-L1 leaking can defeat the purpose of hierarchical routing.
- System ID in the NET is typically encoded from the loopback IP for easy identification (e.g., 10.0.0.1 becomes 0100.0000.0001).
- IS-IS hello padding can be disabled to save bandwidth, but keep it enabled initially to detect MTU mismatches.

## See Also

- ospf, bgp, bfd, mpls, ecmp

## References

- [RFC 1195 — Use of OSI IS-IS for Routing in TCP/IP and Dual Environments](https://www.rfc-editor.org/rfc/rfc1195)
- [RFC 5305 — IS-IS Extensions for Traffic Engineering](https://www.rfc-editor.org/rfc/rfc5305)
- [RFC 5308 — Routing IPv6 with IS-IS](https://www.rfc-editor.org/rfc/rfc5308)
- [RFC 5120 — M-ISIS: Multi Topology Routing in IS-IS](https://www.rfc-editor.org/rfc/rfc5120)
- [RFC 8202 — IS-IS Multi-Instance](https://www.rfc-editor.org/rfc/rfc8202)
- [FRRouting IS-IS Documentation](https://docs.frrouting.org/en/latest/isisd.html)
- [Cisco IS-IS Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/iproute_isis/configuration/xe-16/irs-xe-16-book.html)
- [Juniper IS-IS Configuration Guide](https://www.juniper.net/documentation/us/en/software/junos/is-is/topics/topic-map/is-is-overview.html)
- [BIRD Internet Routing Daemon Documentation](https://bird.network.cz/?get_doc&v=20&f=bird.html)
