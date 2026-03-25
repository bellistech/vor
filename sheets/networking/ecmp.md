# ECMP (Equal-Cost Multi-Path)

Load-balancing technique that distributes traffic across multiple equal-cost routes, increasing bandwidth and providing redundancy without complex failover protocols.

## Concepts

### How ECMP Works

- When multiple routes to the same destination have identical metrics, they are all installed in the routing table
- A hash function determines which next-hop each packet or flow uses
- **Per-flow hashing:** All packets in a flow (same 5-tuple) take the same path, preserving packet ordering
- **Per-packet hashing:** Packets distributed round-robin regardless of flow (causes reordering, rarely used)

### Hash Inputs

| Policy | Hash Fields                     | Use Case                          |
|--------|---------------------------------|-----------------------------------|
| L3     | Src/Dst IP only                 | Simple, may polarize with NAT     |
| L3+L4  | Src/Dst IP + Src/Dst Port + Proto | Best distribution for most traffic|
| Inner  | Inner headers (for tunnels)     | Required for VXLAN/GRE/MPLS ECMP  |

### Polarization

- Occurs when multiple devices in the path use the same hash algorithm and inputs
- All traffic ends up on the same link at each hop, defeating ECMP
- Fix: use different hash seeds, or alternate between L3 and L3+L4 hashing at different layers

## Linux Configuration

### Hash Policy

```bash
# Show current hash policy
sysctl net.ipv4.fib_multipath_hash_policy

# 0 = L3 only (src/dst IP) — default
# 1 = L3+L4 (src/dst IP + ports + protocol) — recommended
# 2 = L3+L4 with inner headers (for encapsulated traffic)
# 3 = Custom (kernel 5.12+, uses fib_multipath_hash_fields)

# Set L3+L4 hashing (recommended for most environments)
sudo sysctl -w net.ipv4.fib_multipath_hash_policy=1

# For VXLAN/GRE/MPLS underlay: hash on inner headers
sudo sysctl -w net.ipv4.fib_multipath_hash_policy=2
```

```
# Persistent config in /etc/sysctl.d/ecmp.conf
net.ipv4.fib_multipath_hash_policy = 1
```

### Adding ECMP Routes

```bash
# Add a route with two equal-cost next-hops
ip route add 10.10.0.0/16 \
  nexthop via 10.1.1.1 dev eth0 \
  nexthop via 10.2.2.1 dev eth1

# Three-way ECMP
ip route add 10.10.0.0/16 \
  nexthop via 10.1.1.1 dev eth0 \
  nexthop via 10.2.2.1 dev eth1 \
  nexthop via 10.3.3.1 dev eth2

# Weighted ECMP (unequal load distribution)
ip route add 10.10.0.0/16 \
  nexthop via 10.1.1.1 dev eth0 weight 2 \
  nexthop via 10.2.2.1 dev eth1 weight 1
# Weight 2:1 means roughly 66%/33% traffic split
```

### Verifying ECMP Routes

```bash
# Show multipath routes (look for "nexthop" entries)
ip route show 10.10.0.0/16

# Show with detailed nexthop info
ip -d route show 10.10.0.0/16

# Count ECMP paths for a prefix
ip route show 10.10.0.0/16 | grep -c nexthop

# Trace which nexthop a specific flow uses (kernel 4.17+)
ip route get 10.10.5.5 from 192.168.1.100 sport 12345 dport 80 ipproto tcp
```

### Resilient Hashing (Kernel 5.17+)

```bash
# Resilient hashing: when a nexthop is removed, only flows on that
# nexthop are rehashed; other flows remain on their current path
# Requires nexthop objects

# Create nexthop group with resilient hashing
ip nexthop add id 1 via 10.1.1.1 dev eth0
ip nexthop add id 2 via 10.2.2.1 dev eth1
ip nexthop add id 3 via 10.3.3.1 dev eth2
ip nexthop add id 100 group 1/2/3 type resilient buckets 128 idle_timer 120

# Use the nexthop group in a route
ip route add 10.10.0.0/16 nhid 100
```

## FRRouting ECMP

### OSPF Equal-Cost Paths

```
router ospf
 # Maximum number of equal-cost paths (default varies, often 4)
 maximum-paths 16
```

### BGP Multipath

```
router bgp 65001
 address-family ipv4 unicast
  # eBGP multipath: use up to 8 equal AS-path-length routes
  maximum-paths 8
  # iBGP multipath: use up to 8 equal routes from iBGP peers
  maximum-paths ibgp 8
  # Compare MED across different neighbor ASes for multipath
  bgp bestpath as-path multipath-relax
```

### Show ECMP in FRRouting

```bash
# Routes with multiple nexthops show ECMP
vtysh -c "show ip route 10.10.0.0/16"

# BGP multipath routes show with "multipath" flag
vtysh -c "show bgp ipv4 unicast 10.10.0.0/16"

# OSPF equal-cost routes
vtysh -c "show ip ospf route" | grep "equal"
```

## Integration with Overlay Protocols

### ECMP with VXLAN

```bash
# Hash on inner headers so VXLAN-encapsulated flows are distributed
sudo sysctl -w net.ipv4.fib_multipath_hash_policy=2

# VXLAN source port is derived from inner header hash by default,
# which already provides entropy for underlay L3+L4 ECMP
```

### ECMP with MPLS

```bash
# For MPLS, use entropy label or hash on label stack
# Linux kernel uses the label stack for MPLS ECMP by default

# FRRouting: set maximum-paths in LDP or BGP for MPLS ECMP
router bgp 65001
 address-family vpnv4 unicast
  maximum-paths 4
  maximum-paths ibgp 4
```

## Troubleshooting

### Uneven Traffic Distribution

```bash
# Check hash policy — L3-only hashing causes poor distribution with few src/dst pairs
sysctl net.ipv4.fib_multipath_hash_policy

# Switch to L3+L4 if using L3-only
sudo sysctl -w net.ipv4.fib_multipath_hash_policy=1

# Verify actual flow distribution with per-interface counters
ip -s link show eth0
ip -s link show eth1
# Compare TX bytes/packets across ECMP interfaces
```

### Flow Pinning

```bash
# Single large flow will always hash to one path — this is by design
# Per-flow hashing preserves packet order but can't split a single flow
# Only solution: use bonding with per-packet distribution (causes reordering)
# or application-level load balancing
```

### Nexthop Failure

```bash
# When an ECMP nexthop goes down, all flows rehash (without resilient hashing)
# This causes brief disruption to ALL flows, not just those on the failed path

# Use resilient hashing (kernel 5.17+) to minimize rehash scope
# Or use BFD for fast failure detection to reduce disruption window
```

## Tips

- Always set `fib_multipath_hash_policy=1` (L3+L4) on Linux; the default L3-only causes severe polarization.
- Use `fib_multipath_hash_policy=2` on underlay routers carrying VXLAN or GRE to hash on inner headers.
- Weighted ECMP is useful for asymmetric link speeds, but most hardware switches only support equal-weight.
- BGP `bestpath as-path multipath-relax` allows ECMP across routes from different ASes with equal AS-path length.
- Resilient hashing (kernel 5.17+) is a major improvement; without it, any nexthop change causes a full rehash of all flows.
- Monitor per-interface byte counters to detect uneven distribution; `iftop` or `nload` make this easier to visualize.
- ECMP does not help with single elephant flows; those need application-level distribution or flowlet-based scheduling.
- In a Clos/leaf-spine fabric, ECMP across all spine switches is the standard design; use 4+ spines for good distribution.
- Test ECMP distribution with `iperf3` from multiple source ports to confirm flows spread across paths.
- Keep the number of ECMP paths as a power of 2 (2, 4, 8) for the most even hash distribution on hardware ASICs.

## References

- [RFC 2992 — Analysis of an Equal-Cost Multi-Path Algorithm](https://www.rfc-editor.org/rfc/rfc2992)
- [RFC 7690 — Close Encounters with the Internet's Edge: Avoiding ECMP Errors](https://www.rfc-editor.org/rfc/rfc7690)
- [Linux Kernel — IP Sysctl (Multipath Routing)](https://www.kernel.org/doc/html/latest/networking/ip-sysctl.html)
- [FRRouting Zebra — ECMP and Multipath](https://docs.frrouting.org/en/latest/zebra.html)
- [Cisco ECMP Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/iproute_pi/configuration/xe-16/iri-xe-16-book/iri-ecmp.html)
- [Juniper Understanding ECMP](https://www.juniper.net/documentation/us/en/software/junos/is-is/topics/topic-map/understanding-ecmp.html)
- [Arista EOS ECMP Configuration](https://www.arista.com/en/um-eos/eos-ecmp)
- [Cloudflare Blog — ECMP and Anycast](https://blog.cloudflare.com/unimog-cloudflares-edge-load-balancer/)
