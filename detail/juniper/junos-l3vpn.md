# JunOS L3VPN — Deep Dive Theory and Analysis

> In-depth exploration of JunOS L3VPN implementation: routing instance architecture, VRF table internals, RD/RT mechanics, PE-CE protocol interactions, inter-AS option analysis, scaling considerations, and JunOS-specific features. For JNCIE-SP level understanding.

## 1. JunOS Routing Instance Types

### 1.1 Instance Type Overview

JunOS supports multiple routing instance types, each with specific use cases:

| Instance Type      | Use Case                                  | Tables Created                  |
|-------------------|-------------------------------------------|----------------------------------|
| `vrf`             | L3VPN (standard BGP/MPLS VPN)             | `<name>.inet.0`, `<name>.inet6.0` |
| `virtual-router`  | Non-VPN routing isolation                 | `<name>.inet.0`                 |
| `vpls`            | Virtual Private LAN Service               | `<name>.vpls`                   |
| `evpn`            | Ethernet VPN                              | `<name>.evpn.0`                 |
| `l2vpn`           | Layer 2 VPN (circuit cross-connect)       | `<name>.l2vpn.0`               |
| `forwarding`      | Filter-based forwarding                   | `<name>.inet.0`                 |
| `no-forwarding`   | Route reception without forwarding        | `<name>.inet.0`                 |

### 1.2 VRF Instance Internals

When `instance-type vrf` is configured:

1. **Table creation:** JunOS creates `<instance>.inet.0` (and optionally `<instance>.inet6.0`). These are completely isolated from `inet.0`.

2. **Interface binding:** Interfaces assigned to the VRF are removed from `inet.0` and placed into the VRF table. The interface's connected route appears only in `<instance>.inet.0`.

3. **Label allocation:** `vrf-table-label` allocates a single MPLS label for the entire VRF table. Without it, JunOS allocates per-prefix labels (one label per VPN route), which is more granular but consumes more label space.

4. **Route leaking:** Routes from other VRFs or `inet.0` can be imported via RT matching or explicit `rib-group` / `instance-import` policies.

### 1.3 vrf-table-label vs Per-Prefix Labels

**Per-prefix labels (default without `vrf-table-label`):**
- Each VPN prefix gets a unique MPLS label
- Ingress PE can identify the exact destination prefix from the label alone
- More labels consumed, but enables per-prefix traffic engineering
- Egress PE performs a single label pop and directly forwards

**VRF table label (`vrf-table-label`):**
- A single label maps to the entire VRF
- Egress PE pops the VPN label, then performs an IP lookup in the VRF table
- Fewer labels consumed
- Required for certain features (e.g., when VRF has many prefixes)
- Slightly higher CPU on egress PE due to additional IP lookup

## 2. Route Distinguisher (RD) in JunOS

### 2.1 RD Purpose and Format

The RD makes VPN prefixes globally unique in the BGP VPNv4 address family. A VPNv4 route is: `RD:prefix` (e.g., `10.0.0.1:100:192.168.1.0/24`).

RD formats:

| Type | Format                    | Example           |
|------|---------------------------|-------------------|
| 0    | `ASN:number`              | `65000:100`       |
| 1    | `IP:number`               | `10.0.0.1:100`    |
| 2    | `4-byte-ASN:number`       | `65000.1:100`     |

### 2.2 RD Assignment Strategies

**Per-PE unique RD (recommended):**
- Each PE uses its loopback as the IP component: `10.0.0.1:100`, `10.0.0.2:100`
- Same VPN prefix from different PEs appears as different VPNv4 routes
- Route reflectors can carry both paths (they look like different prefixes in VPNv4)
- Enables optimal path selection and fast convergence (both paths visible)

**Per-VPN shared RD (not recommended):**
- All PEs use the same RD for the same VPN: `65000:100` everywhere
- Identical VPNv4 prefixes from different PEs: RR may suppress duplicates
- Breaks best-path diversity at the RR level
- Can cause suboptimal routing and slower convergence

### 2.3 RD and Best Path Selection

When a route reflector receives two VPNv4 routes with the same RD and prefix:
- They appear as the same VPNv4 prefix (same NLRI)
- RR runs best-path, selects one, may not advertise the other
- Client PEs only see one path

With unique RDs:
- They appear as different VPNv4 prefixes (different NLRI due to different RD)
- RR carries both routes
- Client PEs receive both and can make local best-path decisions

## 3. Route Target (RT) Mechanics in JunOS

### 3.1 RT as Extended Community

Route targets are BGP extended communities (type 0x0002) attached to VPNv4/VPNv6 NLRI:

- **Export RT:** Attached to routes when they are exported from the VRF into VPNv4
- **Import RT:** Checked against incoming VPNv4 routes to determine which VRF receives them

### 3.2 vrf-target vs vrf-import/vrf-export

JunOS provides two methods:

**`vrf-target` (simple):** Automatically generates import and export policies. `vrf-target target:65000:100` creates both import and export with the same RT.

**`vrf-import` / `vrf-export` (flexible):** Reference explicit routing policies. These policies can:
- Match on multiple RTs
- Set local-preference
- Set communities
- Filter specific prefixes
- Apply per-prefix actions

When both are configured, `vrf-import`/`vrf-export` override `vrf-target`.

### 3.3 RT Constrained Route Distribution (RFC 4684)

Without RT constraint, every PE receives every VPNv4 route from the RR, even for VPNs not present on that PE. This wastes bandwidth and memory.

With RT constraint:
1. Each PE advertises its import RTs to the RR via BGP (address family `route-target`)
2. RR only sends VPNv4 routes to PEs that have a matching import RT
3. Dramatically reduces unnecessary route distribution in large-scale deployments

```
family route-target;   /* enable in BGP group */
```

## 4. PE-CE Protocol Interactions in JunOS

### 4.1 OSPF as PE-CE Protocol

When OSPF runs between PE and CE, several complex interactions occur:

**Domain ID:** A 6-byte identifier (configured as `domain-id`) ensures OSPF routes redistributed between BGP and OSPF are treated correctly:
- Same domain-id on both PEs: redistributed routes appear as OSPF inter-area (Type 3 LSA)
- Different domain-id: redistributed routes appear as OSPF external (Type 5 LSA)
- This preserves OSPF route preference rules across the MPLS backbone

**Sham Link:** When a backdoor link exists between two CE sites running OSPF:
- OSPF prefers the intra-area backdoor path over the VPN path (which appears as inter-area)
- A sham link creates a virtual OSPF adjacency between PEs through the VPN
- Routes via the sham link appear as intra-area, competing fairly with the backdoor

```
routing-instances {
    CUSTOMER-A {
        protocols {
            ospf {
                sham-link {
                    local 10.100.0.1;     /* PE1 VRF address */
                    remote 10.100.0.2;    /* PE2 VRF address */
                    metric 10;
                }
            }
        }
    }
}
```

**DN bit and VPN Route Tag:** Prevent routing loops when OSPF routes transit multiple VPN sites:
- DN bit set on Type 3/5 LSAs generated from VPN routes
- An OSPF router receiving an LSA with DN bit set does not install it in the routing table
- VPN route tag (derived from route-distinguisher) provides additional loop prevention

### 4.2 BGP as PE-CE Protocol

When BGP runs between PE and CE:
- **AS override:** PE replaces the customer's AS number in the AS-path to prevent loop detection rejections when the same customer AS is used at multiple sites:
  ```
  neighbor 10.1.1.2 {
      as-override;
  }
  ```
- **allowas-in:** Alternative to AS override. CE accepts routes containing its own AS:
  ```
  neighbor 10.1.1.2 {
      accept-remote-nexthop;   /* for multihop */
  }
  ```
- **Site of Origin (SoO):** Prevents routing loops in hub-and-spoke or dual-homed scenarios by tagging routes with the originating site.

### 4.3 Route Redistribution Between PE-CE and MP-BGP

The flow of a VPN route through the network:

1. CE advertises `192.168.1.0/24` to PE via PE-CE protocol (e.g., OSPF)
2. PE installs route in `CUSTOMER-A.inet.0`
3. PE's VRF export policy:
   - Prepends RD to prefix: `10.0.0.1:100:192.168.1.0/24`
   - Attaches export RT: `target:65000:100`
   - Sets VPN label
   - Advertises via MP-BGP to remote PEs
4. Remote PE receives VPNv4 route
5. Remote PE's VRF import policy checks RT match
6. Route installed in remote PE's `CUSTOMER-A.inet.0` (RD stripped)
7. Remote PE redistributes to CE via PE-CE protocol

## 5. Inter-AS Options Analysis

### 5.1 Option A: Back-to-Back VRF

**Architecture:** ASBRs maintain per-VPN VRFs and run PE-CE protocols between them.

**Advantages:**
- Simplest to implement and troubleshoot
- Complete control plane isolation between ASes
- Independent VPN policies per AS
- No trust relationship required between AS operators

**Disadvantages:**
- Does not scale: each VPN requires a VRF, sub-interface, and routing session on each ASBR
- ASBR becomes a bottleneck for VPN scale
- Double IP lookup on each ASBR (egress from VRF, ingress to VRF)

**Scale limit:** Practical limit of ~100-500 VPNs per ASBR pair, depending on hardware.

### 5.2 Option B: ASBR-to-ASBR eBGP VPNv4

**Architecture:** ASBRs exchange VPNv4 routes directly via MP-eBGP. No per-VPN state on ASBR — just VPNv4 route forwarding.

**Key mechanics:**
- ASBR receives VPNv4 routes from internal iBGP
- Re-advertises to external ASBR peer with next-hop-self
- ASBR must allocate new labels (label swap at AS boundary)
- The `next-hop-self` is critical: both iBGP and eBGP sides need it

**Advantages:**
- Scales to thousands of VPNs (no per-VPN VRF on ASBR)
- Single ASBR peering session carries all VPNs
- Efficient label switching (no IP lookup)

**Disadvantages:**
- ASBR must hold all VPNv4 routes in BGP table (memory)
- Some trust between ASes (VPNv4 routes visible)
- ASBR label allocation can be significant

### 5.3 Option C: Multi-Hop eBGP Between RRs

**Architecture:** RRs in different ASes exchange VPNv4 routes. ASBRs only exchange labeled IPv4 for next-hop resolution.

**Key mechanics:**
1. RR in AS1 peers with RR in AS2 for VPNv4 (multi-hop eBGP)
2. VPNv4 routes carry the original PE next-hop (no next-hop-self on RR)
3. ASBRs exchange labeled IPv4 routes for PE loopbacks
4. End-to-end label stack: `[ASBR-label | VPN-label | payload]`
5. ASBR only sees labeled IPv4, never touches VPNv4 routes

**Advantages:**
- Best scalability (ASBR holds minimal state)
- End-to-end MPLS path (no VPN route processing on ASBR)
- Cleanest separation of concerns

**Disadvantages:**
- Most complex to configure and troubleshoot
- Requires labeled IPv4 exchange between ASBRs (BGP labeled-unicast)
- Higher trust requirement (PE loopbacks leaked between ASes)
- Multi-hop eBGP adds failure complexity

### 5.4 Comparison Matrix

| Aspect              | Option A         | Option B           | Option C            |
|--------------------|------------------|--------------------|---------------------|
| ASBR VPN state     | Per-VPN VRF      | VPNv4 routes       | Labeled IPv4 only   |
| Scalability        | Low (~500 VPNs)  | Medium (~10K VPNs) | High (~100K VPNs)   |
| Trust requirement  | None             | Medium             | High                |
| Label operations   | IP lookup x2     | Label swap         | Label swap          |
| Complexity         | Low              | Medium             | High                |
| Troubleshooting    | Easy             | Moderate           | Difficult           |

## 6. JunOS-Specific L3VPN Features

### 6.1 Auto-Export

When multiple VRFs on the same PE need to exchange routes (e.g., shared services), `auto-export` enables local route leaking without MP-BGP:

```
routing-options {
    auto-export;
}
```

JunOS examines all local VRF import/export RTs and directly leaks matching routes between local VRFs. This is more efficient than sending routes to a remote PE and back.

### 6.2 rib-group for Route Leaking

`rib-group` copies routes between routing tables, enabling route leaking between VRFs or between VRF and `inet.0`:

```
routing-options {
    rib-groups {
        LEAK-TO-INET {
            import-rib [ CUSTOMER-A.inet.0 inet.0 ];
            import-policy LEAK-FILTER;
        }
    }
}
```

### 6.3 Forwarding Table Chaining

JunOS can chain forwarding lookups: `next-table` in a VRF's static route points to another routing table for recursive lookup. This is used in hub-and-spoke to send spoke traffic through the hub VRF.

## 7. Scaling Considerations

### 7.1 Memory and Route Scale

Key scaling factors for L3VPN:
- **VPNv4 routes per PE:** Each VPNv4 route consumes ~200-400 bytes in BGP RIB
- **VRF table entries:** Each VRF route in FIB consumes PFE memory (ASIC dependent)
- **Labels:** Per-prefix labels consume more label space than `vrf-table-label`
- **RT constraint:** Without it, every PE stores routes for every VPN (O(n*m) problem)

### 7.2 Route Reflector Design

For large-scale L3VPN:
- Dedicate RRs for VPNv4 (separate from IPv4 unicast RRs)
- Use RT constraint to limit route distribution
- Consider hierarchical RR design for very large networks
- RR hardware must accommodate full VPNv4 table (potentially millions of routes)

### 7.3 PE Hardware Considerations

- **VRF count:** Platform-specific limits (MX series supports thousands)
- **VRF routes:** FIB capacity shared across all VRFs
- **PE-CE sessions:** Each routing adjacency consumes CPU for keepalives and updates
- **Label space:** Check platform label range (affects per-prefix vs vrf-table-label choice)

## See Also

- junos-mpls-advanced
- junos-l2vpn
- junos-evpn-vxlan
- junos-routing-fundamentals

## References

- RFC 4364 — BGP/MPLS IP Virtual Private Networks (VPNs)
- RFC 4659 — BGP-MPLS IP VPN Extension for IPv6 VPN
- RFC 4684 — Constrained Route Distribution for BGP/MPLS VPN
- RFC 4577 — OSPF as the PE/CE Protocol in BGP/MPLS VPNs
- RFC 6368 — Internal BGP as PE-CE Protocol
- Juniper TechLibrary: Layer 3 VPN Configuration Guide
- Juniper JNCIE-SP Study Guide: Advanced L3VPN
