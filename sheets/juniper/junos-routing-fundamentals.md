# JunOS Routing Fundamentals (JNCIA-Junos Exam Prep)

Juniper routing architecture covering packet forwarding, routing tables, route selection, routing instances, static routing, and dynamic protocol overview for JNCIA certification.

## Traffic Forwarding Concepts

### How a Packet Traverses the Device

- **Ingress PFE:** Packet arrives on interface, Layer 2 header stripped, lookup begins
- **Forwarding table lookup:** PFE performs longest-match lookup against forwarding table (copy of active routes from RE)
- **Next-hop resolution:** Egress interface and next-hop MAC determined
- **Egress PFE:** Layer 2 header rewritten, packet queued and transmitted
- **Exception traffic:** Packets destined to the RE (SSH, OSPF hellos, SNMP) sent up via internal link to Routing Engine
- **Control plane vs data plane:** RE (control) builds routes; PFE (data) forwards packets at hardware speed

```
Ingress Interface --> PFE (forwarding table lookup)
  |                         |
  |  exception traffic      |  transit traffic
  v                         v
Routing Engine (RE)    Egress PFE --> Egress Interface
  |
  v
Process locally (BGP, OSPF, SSH, etc.)
```

## Routing Tables

### Standard Routing Tables

| Table            | Purpose                                              |
|------------------|------------------------------------------------------|
| `inet.0`         | IPv4 unicast routes (default table for IPv4)         |
| `inet6.0`        | IPv6 unicast routes                                  |
| `inet.2`         | IPv4 multicast RPF lookup table                      |
| `inet.1`         | IPv4 multicast forwarding cache                      |
| `inet.3`         | MPLS path info (used for BGP next-hop resolution)    |
| `mpls.0`         | MPLS label-switched paths                            |
| `bgp.l3vpn.0`   | BGP Layer 3 VPN routes (VPNv4/VPNv6)                |
| `instance.inet.0`| Per-routing-instance IPv4 unicast table              |

```bash
# List all routing tables on the device
show route summary

# View specific table
show route table inet.0
show route table inet6.0
show route table mpls.0
show route table bgp.l3vpn.0
```

## Routing Table vs Forwarding Table

| Aspect           | Routing Table (RE)                         | Forwarding Table (PFE)              |
|------------------|--------------------------------------------|--------------------------------------|
| **Location**     | Routing Engine (control plane)             | Packet Forwarding Engine (data plane)|
| **Contents**     | All learned routes (active + inactive)     | Only active/best routes              |
| **Populated by** | Routing protocols, static config, direct   | Copied from RE routing table         |
| **Purpose**      | Route selection, policy evaluation         | Actual packet forwarding             |
| **Lookup**       | Software-based                             | Hardware-accelerated (ASIC/Memory)   |

```bash
# View routing table (all routes, on RE)
show route

# View forwarding table (active routes, on PFE)
show route forwarding-table

# Compare: routing table shows inactive alternatives
show route 10.0.0.0/24 detail
# Forwarding table shows only the installed next-hop
show route forwarding-table destination 10.0.0.0/24
```

## Route Preference (Administrative Distance)

### Default Route Preference Values

| Protocol / Source            | Preference |
|------------------------------|------------|
| Direct (connected)           | 0          |
| Local                        | 0          |
| Static                       | 5          |
| OSPF internal                | 10         |
| IS-IS Level 1 internal       | 15         |
| IS-IS Level 2 internal       | 18         |
| RIP / RIPng                  | 100        |
| PIM                          | 105        |
| Aggregate                    | 130        |
| OSPF AS external             | 150        |
| IS-IS Level 1 external       | 160        |
| IS-IS Level 2 external       | 165        |
| BGP (iBGP and eBGP)          | 170        |

- Lower preference = more preferred
- Unlike Cisco, Junos uses the same default preference (170) for both iBGP and eBGP
- Preference can be overridden with routing policy: `set routing-options static route ... preference 200`

```bash
# View route with preference value shown
show route 10.0.0.0/24 detail | match preference

# Override static route preference
set routing-options static route 10.0.0.0/24 preference 200
```

## Route Selection Process

### Active Route Selection Algorithm

1. **Lowest route preference** (administrative distance)
2. **Lowest metric** (protocol-specific: OSPF cost, BGP MED, IS-IS metric)
3. **Protocol-specific tiebreakers** (BGP: local-pref > AS-path > origin > MED > eBGP > IGP cost > router-id)
4. **Next-hop resolution** — route must have a valid, resolvable next-hop
5. **Longest match wins** for actual forwarding decisions (independent of preference)

```bash
# See active (*) and inactive routes with selection reason
show route 10.0.0.0/24 extensive

# Active route marked with asterisk (*)
show route protocol ospf
```

## Routing Instances

### Instance Types

| Type              | Use Case                                              |
|-------------------|-------------------------------------------------------|
| `default`         | Main instance; inet.0, inet6.0 live here              |
| `virtual-router`  | Separate routing table, no VPN/MPLS signaling         |
| `vrf`             | Full L3VPN: route-distinguisher, route-target, MPLS   |
| `forwarding`      | Filter-based forwarding (no routing protocols)         |

```bash
# Create a virtual-router instance
set routing-instances CUST-A instance-type virtual-router
set routing-instances CUST-A interface ge-0/0/1.100
set routing-instances CUST-A routing-options static route 0.0.0.0/0 next-hop 192.168.1.1

# Create a VRF instance
set routing-instances VPN-B instance-type vrf
set routing-instances VPN-B interface ge-0/0/2.200
set routing-instances VPN-B route-distinguisher 65000:100
set routing-instances VPN-B vrf-target target:65000:100

# View instance routing table
show route table CUST-A.inet.0
show route instance CUST-A

# List all routing instances
show route instance summary
```

## Static Routing

### Basic Static Routes

```bash
# Simple static route with next-hop IP
set routing-options static route 10.10.0.0/16 next-hop 192.168.1.1

# Static route with next-hop interface (point-to-point links)
set routing-options static route 10.20.0.0/16 next-hop ge-0/0/0.0

# Discard route (silently drop — used for aggregation/null route)
set routing-options static route 10.0.0.0/8 discard

# Reject route (drop and send ICMP unreachable)
set routing-options static route 192.168.99.0/24 reject

# Prevent static route from being redistributed
set routing-options static route 10.10.0.0/16 no-readvertise

# Resolve: allow next-hop resolution via routing table (indirect next-hop)
set routing-options static route 172.16.0.0/12 next-hop 10.0.0.1 resolve
```

### Qualified Next-Hop (Floating Static)

```bash
# Primary path: preference 5 (default for static)
set routing-options static route 10.10.0.0/16 next-hop 192.168.1.1

# Backup path: higher preference = less preferred (floating static)
set routing-options static route 10.10.0.0/16 qualified-next-hop 172.16.1.1 preference 200

# Alternative: set preference on the entire route for floating static
set routing-options static route 10.10.0.0/16 next-hop 172.16.1.1 preference 210
```

### Floating Static Route Example

```bash
# OSPF-learned route (preference 10) is preferred while available
# Static backup activates only when OSPF route disappears
set routing-options static route 10.10.0.0/16 next-hop 172.16.1.1 preference 200

# Verify: OSPF active, static inactive
show route 10.10.0.0/16
# 10.10.0.0/16  *[OSPF/10] via 192.168.1.1, ge-0/0/0.0
#                [Static/200] via 172.16.1.1, ge-0/0/1.0
```

## Show Commands

```bash
# Full routing table
show route

# Summary: route count per table and protocol
show route summary

# Specific table
show route table inet.0
show route table inet6.0
show route table CUST-A.inet.0

# Filter by protocol
show route protocol static
show route protocol ospf
show route protocol bgp

# Specific prefix with detail (shows preference, metric, next-hop, age)
show route 10.0.0.0/24 detail
show route 10.0.0.0/24 extensive    # maximum detail

# Forwarding table (what PFE actually uses)
show route forwarding-table
show route forwarding-table destination 10.0.0.0/24
show route forwarding-table family inet

# Route instance tables
show route instance
show route instance summary
```

## Dynamic Routing Protocols Overview

### IGP vs EGP

| Aspect      | IGP (Interior Gateway Protocol)       | EGP (Exterior Gateway Protocol)    |
|-------------|---------------------------------------|------------------------------------|
| **Scope**   | Within a single AS                    | Between autonomous systems         |
| **Examples**| OSPF, IS-IS, RIP                      | BGP                               |
| **Goal**    | Fast convergence, optimal paths       | Policy control, scalability        |

### When to Use Each Protocol

- **OSPF:** Enterprise/campus networks, link-state, fast convergence, well-understood, areas for scaling
- **IS-IS:** Large ISP/service provider networks, protocol-agnostic (IPv4/IPv6 natively), simpler TLV extensibility
- **BGP:** Internet peering, WAN/multi-homed connections, L3VPN, policy-heavy environments
- **RIP:** Legacy or very simple networks only; limited to 15 hops, slow convergence

```bash
# View configured protocols
show protocols

# Protocol-specific route views
show route protocol ospf
show route protocol bgp
show route protocol isis

# Protocol neighbor/adjacency status
show ospf neighbor
show bgp summary
show isis adjacency
```

## Tips

- Route preference in Junos is equivalent to administrative distance in Cisco; lower is better.
- The forwarding table is a subset of the routing table containing only active best-path routes pushed to the PFE.
- Use `qualified-next-hop` with a higher preference value to create backup static routes that activate on primary failure.
- A `discard` route silently drops packets; a `reject` route drops and sends ICMP unreachable back to the sender.
- Always verify the forwarding table with `show route forwarding-table` when debugging forwarding issues; the routing table may show a route that has not yet been installed.
- The `resolve` option on static routes allows indirect (recursive) next-hop resolution, required when the next-hop is not directly connected.
- When mixing static and dynamic routing, remember that static (preference 5) beats OSPF internal (10) by default; adjust preference if you want dynamic routes to win.
- Use `no-readvertise` on static routes that should remain local and never leak into dynamic protocol advertisements.
- Routing instance tables are named `<instance-name>.inet.0`; forgetting this is a common troubleshooting mistake.
- BGP has the same default preference (170) for both iBGP and eBGP in Junos, unlike Cisco where eBGP (20) beats iBGP (200).

## See Also

- bgp, ospf, is-is, rip, mpls, vxlan, ecmp, bfd, ipv4, ipv6, subnetting

## References

- [Juniper JNCIA-Junos Study Guide](https://www.juniper.net/documentation/us/en/software/junos/junos-getting-started/topics/topic-map/junos-getting-started.html)
- [Juniper TechLibrary — Routing Tables Overview](https://www.juniper.net/documentation/us/en/software/junos/routing-overview/topics/concept/routing-tables-overview.html)
- [Juniper TechLibrary — Route Preference](https://www.juniper.net/documentation/us/en/software/junos/routing-overview/topics/ref/general/routing-protocols-default-route-preference-values.html)
- [Juniper TechLibrary — Routing Instances](https://www.juniper.net/documentation/us/en/software/junos/routing-overview/topics/concept/routing-instances-overview.html)
- [Juniper TechLibrary — Static Routing](https://www.juniper.net/documentation/us/en/software/junos/static-routing/topics/concept/static-routing-overview.html)
- [Juniper Day One: Exploring the Junos CLI (2nd Edition)](https://www.juniper.net/documentation/en_US/day-one-books/DO_CLI2.pdf)
- [RFC 4271 — A Border Gateway Protocol 4 (BGP-4)](https://www.rfc-editor.org/rfc/rfc4271)
- [RFC 2328 — OSPF Version 2](https://www.rfc-editor.org/rfc/rfc2328)
