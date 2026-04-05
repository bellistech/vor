# JunOS Segment Routing

Source-routing paradigm replacing LDP/RSVP with IGP-distributed label/SID allocation. Covers SR-MPLS (SPRING), SRv6, SR-TE, TI-LFA, and flex-algo on JunOS platforms.

## SR-MPLS (SPRING) Fundamentals

### Segment types
```
Prefix-SID:    Globally unique, identifies a prefix (node or anycast)
               Allocated from SRGB, advertised via IGP
               Label = SRGB base + SID index

Adjacency-SID: Locally significant, identifies a link/adjacency
               Allocated from SRLB or dynamic range
               Used for strict path specification (TE)

Node-SID:      A prefix-SID associated with a loopback (node identity)
               Most common prefix-SID usage
```

### SRGB and SRLB
```
# SRGB (Segment Routing Global Block) — global label range for prefix-SIDs
# Default SRGB: 16000-23999 (8000 labels)
set protocols isis source-packet-routing srgb start-label 16000 index-range 8000

# SRLB (Segment Routing Local Block) — local label range for adjacency-SIDs
set protocols isis source-packet-routing srlb start-label 100000 index-range 1000

# Verify
show isis overview | match "SRGB|SRLB"
show mpls label usage
```

## IS-IS with Segment Routing

### Enable SR on IS-IS
```
# Enable SPRING on IS-IS globally
set protocols isis source-packet-routing

# Node-SID for loopback (must be globally unique index)
set protocols isis source-packet-routing node-segment ipv4-index 100
set protocols isis source-packet-routing node-segment ipv6-index 200

# Enable SR on specific interfaces
set protocols isis interface ge-0/0/0.0 level 2 post-convergence-lfa node-protection
set protocols isis interface ge-0/0/1.0 level 2 post-convergence-lfa node-protection
set protocols isis interface lo0.0 passive
```

### Adjacency-SID
```
# Static adjacency-SID (manually assigned)
set protocols isis interface ge-0/0/0.0 level 2 ipv4-adjacency-segment protected label 100001
set protocols isis interface ge-0/0/1.0 level 2 ipv4-adjacency-segment protected label 100002

# Verify adjacency-SIDs
show isis adjacency detail | match "Adjacency-SID"
show isis database detail | match "Adj-SID"
```

### IS-IS SR verification
```
show isis overview | match "Segment"           # SR capability
show isis database detail | match "SID|SRGB"   # SID advertisements in LSDB
show isis adjacency detail                      # adjacency-SIDs
show route table inet.3 protocol isis           # SR-learned labels
show route table mpls.0 protocol isis           # MPLS label table
show isis spring-te overview                    # SPRING-TE state
```

## OSPF with Segment Routing

### Enable SR on OSPF
```
set protocols ospf source-packet-routing
set protocols ospf source-packet-routing node-segment ipv4-index 100
set protocols ospf source-packet-routing srgb start-label 16000 index-range 8000

# SR-enabled interfaces
set protocols ospf area 0.0.0.0 interface ge-0/0/0.0
set protocols ospf area 0.0.0.0 interface lo0.0 passive

# Verify
show ospf overview | match "Segment"
show ospf database detail | match "SID"
```

## SR-TE (Traffic Engineering)

### Colored tunnels
```
# SR-TE uses segment lists (label stacks) to steer traffic along explicit paths
# "Colors" associate tunnels with routing policies

# Define SR-TE LSP
set protocols source-packet-routing segment-list SEG-LIST-1 hop1 label 16001
set protocols source-packet-routing segment-list SEG-LIST-1 hop2 label 16002
set protocols source-packet-routing segment-list SEG-LIST-1 hop3 label 16003

# Create SR-TE path
set protocols source-packet-routing source-routing-path SR-PATH-1 to 10.255.0.3
set protocols source-packet-routing source-routing-path SR-PATH-1 primary SEG-LIST-1
set protocols source-packet-routing source-routing-path SR-PATH-1 color 100

# Backup segment list
set protocols source-packet-routing segment-list SEG-LIST-1-BACKUP hop1 label 16004
set protocols source-packet-routing segment-list SEG-LIST-1-BACKUP hop2 label 16003
set protocols source-packet-routing source-routing-path SR-PATH-1 secondary SEG-LIST-1-BACKUP
```

### Color-based steering with BGP
```
# Map BGP next-hop color community to SR-TE tunnel
set policy-options community COLOR-100 members color:0:100

set policy-options policy-statement SR-STEERING term COLOR from community COLOR-100
set policy-options policy-statement SR-STEERING term COLOR then install-nexthop lsp SR-PATH-1
set policy-options policy-statement SR-STEERING term DEFAULT then accept

set protocols bgp group IBGP import SR-STEERING
```

### Verify SR-TE
```
show source-packet-routing overview             # SR-TE global state
show source-packet-routing segment-list          # configured segment lists
show source-packet-routing source-routing-path   # SR-TE paths
show route table inet.3                          # SR tunnels in inet.3
show route 10.255.0.3 table inet.3 detail       # specific SR-TE path details
```

## TI-LFA (Topology-Independent Loop-Free Alternate)

### Enable TI-LFA with IS-IS
```
# TI-LFA provides sub-50ms protection using pre-computed backup paths
set protocols isis backup-spf-options use-post-convergence-lfa
set protocols isis interface ge-0/0/0.0 level 2 post-convergence-lfa node-protection
set protocols isis interface ge-0/0/1.0 level 2 post-convergence-lfa node-protection

# Optional: enable for all interfaces
set protocols isis level 2 post-convergence-lfa node-protection
```

### TI-LFA protection types
```
# Link protection: protects against link failure
set protocols isis interface ge-0/0/0.0 level 2 post-convergence-lfa

# Node protection: protects against node failure (preferred in SP)
set protocols isis interface ge-0/0/0.0 level 2 post-convergence-lfa node-protection

# SRLG protection: protects against shared-risk link group failure
set protocols isis interface ge-0/0/0.0 level 2 post-convergence-lfa srlg-protection
```

### Verify TI-LFA
```
show isis backup-spf results                    # backup path computation results
show isis backup coverage                       # protection coverage percentage
show route 10.255.0.3 detail | match "backup"   # backup next-hop for specific prefix
show isis interface detail | match "LFA|backup"  # per-interface LFA state
show route forwarding-table destination 10.255.0.3  # PFE forwarding with backup
```

## Flex-Algo (Flexible Algorithm)

### Define flex-algo
```
# Flex-algo allows multiple IGP topologies with different constraints
# Algorithm IDs 128-255 are available for flex-algo

# Define algorithm with metric type
set protocols isis source-packet-routing flex-algorithm 128
set protocols isis source-packet-routing flex-algorithm 128 metric-type igp
set protocols isis source-packet-routing flex-algorithm 128 admin-group include-any GROUP-LOW-LATENCY

# Define algorithm for TE metric
set protocols isis source-packet-routing flex-algorithm 129
set protocols isis source-packet-routing flex-algorithm 129 metric-type te-metric

# Node-SID for flex-algo
set protocols isis source-packet-routing node-segment ipv4-index 100 algorithm 128
set protocols isis source-packet-routing node-segment ipv4-index 200 algorithm 129
```

### Admin groups (colors) for flex-algo
```
# Define admin groups on interfaces
set protocols isis interface ge-0/0/0.0 level 2 te-metric 10
set protocols isis interface ge-0/0/0.0 level 2 admin-group GROUP-LOW-LATENCY

set protocols isis interface ge-0/0/1.0 level 2 te-metric 100
set protocols isis interface ge-0/0/1.0 level 2 admin-group GROUP-HIGH-BW
```

### Verify flex-algo
```
show isis flex-algorithm                         # defined algorithms
show isis database detail | match "Flex-Algo"    # flex-algo advertisements
show route table inet.3 protocol isis            # per-algo routes
```

## SRv6 (Segment Routing over IPv6)

### SRv6 locator and SID
```
# SRv6 uses IPv6 addresses as segment identifiers
# Locator: IPv6 prefix that identifies the node
# Function: suffix within locator that specifies behavior

# Enable SRv6
set routing-options source-packet-routing srv6

# Define locator
set routing-options source-packet-routing srv6 locator LOC1 prefix 2001:db8:1::/48
set routing-options source-packet-routing srv6 locator LOC1 static-function end-dt4-sid 2001:db8:1::1

# Enable SRv6 with IS-IS
set protocols isis source-packet-routing srv6 locator LOC1
```

### SRv6 SID behaviors (END functions)
```
# End:       Basic endpoint — node SID, pop SRH and forward
# End.X:     Endpoint with cross-connect — adjacency SID
# End.DT4:   Decapsulate and lookup in IPv4 table — L3VPN
# End.DT6:   Decapsulate and lookup in IPv6 table — L3VPN
# End.DT46:  Decapsulate and lookup in IPv4/IPv6 table
# End.DX4:   Decapsulate and cross-connect IPv4
# End.DX6:   Decapsulate and cross-connect IPv6
# End.B6:    Endpoint bound to SRv6 policy (encapsulation)
```

### SRv6 TE policy
```
# SRv6 TE uses SID lists with IPv6 SIDs
set protocols source-packet-routing srv6 segment-list SRV6-SEG-1 hop1 sid 2001:db8:1::1
set protocols source-packet-routing srv6 segment-list SRV6-SEG-1 hop2 sid 2001:db8:2::1
set protocols source-packet-routing srv6 segment-list SRV6-SEG-1 hop3 sid 2001:db8:3::1

set protocols source-packet-routing srv6 source-routing-path SRV6-PATH to 2001:db8:3::1
set protocols source-packet-routing srv6 source-routing-path SRV6-PATH primary SRV6-SEG-1
```

### Verify SRv6
```
show route table inet6.0 protocol isis | match "SRv6"
show isis database detail | match "SRv6|Locator|SID"
show source-packet-routing srv6 locator
show source-packet-routing srv6 sid
show route table inet6.3                        # SRv6 tunnels
```

## SR-MPLS to SRv6 Interworking

### Gateway function
```
# Transit node that converts between SR-MPLS and SRv6 domains
# Ingress: receives SR-MPLS label stack, translates to SRv6 SID list
# Egress: receives SRv6, translates to SR-MPLS labels

# This is typically handled via SR policies that bind
# MPLS-domain segment lists to SRv6-domain segment lists
```

## SR Policies

### Policy with binding SID
```
# Binding SID represents an SR policy — any packet with this label
# enters the policy's segment list
set protocols source-packet-routing source-routing-path SR-POLICY-1 binding-sid 1000100
set protocols source-packet-routing source-routing-path SR-POLICY-1 to 10.255.0.5
set protocols source-packet-routing source-routing-path SR-POLICY-1 color 200
set protocols source-packet-routing source-routing-path SR-POLICY-1 primary SEG-LIST-GOLD
set protocols source-packet-routing source-routing-path SR-POLICY-1 secondary SEG-LIST-SILVER
```

### Verify SR policies
```
show source-packet-routing source-routing-path detail
show route label 1000100                         # binding SID route
```

## Complete SR-MPLS Deployment Example

### Topology: 3-node IS-IS SR
```
# Node R1: lo0 = 10.255.0.1/32, SID index 1
set protocols isis interface lo0.0 passive
set protocols isis interface ge-0/0/0.0 point-to-point
set protocols isis interface ge-0/0/1.0 point-to-point
set protocols isis source-packet-routing node-segment ipv4-index 1
set protocols isis source-packet-routing srgb start-label 16000 index-range 8000
set protocols isis level 2 post-convergence-lfa node-protection

# R1 label for reaching R2 (SID index 2): 16000 + 2 = 16002
# R1 label for reaching R3 (SID index 3): 16000 + 3 = 16003

# Node R2: lo0 = 10.255.0.2/32, SID index 2
set protocols isis source-packet-routing node-segment ipv4-index 2

# Node R3: lo0 = 10.255.0.3/32, SID index 3
set protocols isis source-packet-routing node-segment ipv4-index 3
```

## Verification Commands Summary

```
# SR-MPLS
show isis overview | match "Segment|SRGB"        # IS-IS SR state
show isis database detail | match "SID|SRGB"     # LSDB SID info
show isis adjacency detail                        # adjacency-SIDs
show route table inet.3 protocol isis             # SR label routes
show route table mpls.0 protocol isis             # MPLS forwarding
show mpls label usage                             # label range usage

# SR-TE
show source-packet-routing overview               # global SR-TE
show source-packet-routing segment-list            # segment lists
show source-packet-routing source-routing-path     # SR-TE paths

# TI-LFA
show isis backup-spf results                      # backup paths
show isis backup coverage                          # coverage stats

# SRv6
show source-packet-routing srv6 locator            # SRv6 locators
show source-packet-routing srv6 sid                 # SRv6 SIDs

# Flex-algo
show isis flex-algorithm                           # defined algorithms
```

## Tips

- SID indexes must be globally unique across the SR domain — plan a SID allocation scheme before deployment
- SRGB should be consistent across all routers for operational simplicity (same base + range)
- TI-LFA with node-protection provides better coverage than link-protection alone
- Adjacency-SIDs are essential for SR-TE strict paths — prefix-SIDs alone only provide ECMP shortest path
- SRv6 requires IPv6 reachability — ensure IPv6 is enabled on all interfaces before enabling SRv6
- Flex-algo requires all participating nodes to advertise and participate in the same algorithm
- SR-MPLS and LDP can coexist during migration — use `set protocols isis source-packet-routing preference` to control priority
- Binding SIDs simplify policy abstraction — remote nodes reference the policy via a single label
- Label stack depth matters — deep stacks may require ECMP hash rebalancing or imposition limit checks

## See Also

- junos-isis-advanced, junos-routing-fundamentals, junos-bgp-advanced, mpls, isis

## References

- [Juniper TechLibrary — Segment Routing](https://www.juniper.net/documentation/us/en/software/junos/segment-routing/index.html)
- [Juniper TechLibrary — TI-LFA](https://www.juniper.net/documentation/us/en/software/junos/segment-routing/topics/concept/ti-lfa-overview.html)
- [Juniper TechLibrary — SRv6](https://www.juniper.net/documentation/us/en/software/junos/segment-routing/topics/concept/srv6-overview.html)
- [Juniper TechLibrary — Flex-Algorithm](https://www.juniper.net/documentation/us/en/software/junos/segment-routing/topics/concept/flex-algorithm-overview.html)
- [RFC 8402 — Segment Routing Architecture](https://www.rfc-editor.org/rfc/rfc8402)
- [RFC 8667 — IS-IS Extensions for Segment Routing](https://www.rfc-editor.org/rfc/rfc8667)
- [RFC 8665 — OSPF Extensions for Segment Routing](https://www.rfc-editor.org/rfc/rfc8665)
- [RFC 8986 — SRv6 Network Programming](https://www.rfc-editor.org/rfc/rfc8986)
- [RFC 9256 — Segment Routing Policy Architecture](https://www.rfc-editor.org/rfc/rfc9256)
- [RFC 8355 — Resiliency Use Cases in SR](https://www.rfc-editor.org/rfc/rfc8355)
