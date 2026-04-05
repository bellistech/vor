# JunOS Routing Fundamentals — Deep Dive Theory and Analysis

> In-depth exploration of the Junos routing architecture: the complete route selection algorithm, routing table internals, VRF implementation models, route leaking mechanics, and convergence characteristics. Builds on the cheatsheet with theory, math, and design reasoning for JNCIA-Junos and beyond.

## 1. Junos Route Selection Algorithm in Depth

The Junos route selection process determines which route becomes **active** (marked with `*`) and is installed into the forwarding table. When multiple routes exist to the same prefix, the following tiebreaker sequence applies in strict order.

### 1.1 Complete Tiebreaker Sequence

1. **Route preference (administrative distance):** Lowest value wins. Direct=0, Static=5, OSPF-internal=10, IS-IS-L1=15, IS-IS-L2=18, RIP=100, OSPF-external=150, BGP=170.

2. **Next-hop validity:** The route must have a resolvable next-hop. A route with an unresolvable next-hop is immediately eliminated regardless of preference.

3. **Protocol-specific tiebreakers:** When multiple routes from the same protocol share the same preference, protocol-specific rules apply:

   **OSPF tiebreakers:**
   - Intra-area routes preferred over inter-area
   - Inter-area preferred over external type 1
   - External type 1 preferred over external type 2
   - For same type: lowest metric wins
   - Final tie: lowest router-id of the advertising router

   **IS-IS tiebreakers:**
   - Level 1 preferred over Level 2 (by default preference: 15 vs 18)
   - Internal routes preferred over external
   - Lowest metric wins
   - Lowest system-id breaks final tie

   **BGP tiebreakers (full best-path algorithm):**
   1. Highest `local-preference` (default 100)
   2. Shortest `AS-path` length
   3. Lowest `origin` code (IGP < EGP < Incomplete)
   4. Lowest `MED` (multi-exit discriminator), compared only between paths from the same neighboring AS by default
   5. eBGP preferred over iBGP
   6. Lowest IGP metric to the BGP next-hop
   7. Oldest route (most stable path)
   8. Lowest `router-id` of the BGP peer
   9. Lowest peer IP address (final tiebreaker)

4. **Equal-cost multipath (ECMP):** If multiple routes survive all tiebreakers with identical cost, Junos can install all of them as active routes for load balancing. Requires explicit policy configuration:

   ```
   policy-options {
       policy-statement ECMP {
           then accept;
       }
   }
   routing-options {
       forwarding-table {
           export ECMP;
       }
   }
   ```

### 1.2 Active vs Inactive Routes

- **Active route (`*`):** Best route for a prefix; installed into the forwarding table and pushed to PFE.
- **Inactive route:** Valid but not best; retained in routing table as backup. Becomes active if the current active route is withdrawn.
- **Hidden route:** Failed import policy or has unresolvable next-hop. Not eligible for selection. Visible only with `show route hidden`.

### 1.3 Route Selection Math Example

Consider three routes to `10.0.0.0/24`:

| Source   | Preference | Metric | Status     |
|----------|------------|--------|------------|
| Static   | 5          | N/A    | Active `*` |
| OSPF     | 10         | 100    | Inactive   |
| BGP      | 170        | N/A    | Inactive   |

Static wins because `preference 5 < 10 < 170`. If the static route is removed:

| Source   | Preference | Metric | Status     |
|----------|------------|--------|------------|
| OSPF     | 10         | 100    | Active `*` |
| BGP      | 170        | N/A    | Inactive   |

OSPF becomes active. The routing table retains both for instant failover without waiting for protocol re-convergence.

## 2. Routing Table Implementation

### 2.1 Radix Tree (Patricia Trie) Structure

Junos routing tables are implemented as **radix trees** (also called Patricia tries), a compressed trie data structure optimized for IP prefix lookup.

**Properties of the radix tree:**
- Each node represents a bit position in the prefix
- Branches are compressed: single-child nodes are merged with their parent
- Lookup time is `O(W)` where `W` is the address width (32 for IPv4, 128 for IPv6)
- Memory usage is proportional to the number of prefixes, not the address space size
- Insertion and deletion are `O(W)` operations

**Longest-match lookup procedure:**
1. Start at the root of the radix tree
2. At each node, examine the bit at the node's bit position in the destination address
3. Branch left (0) or right (1)
4. Record the most recent node that has an associated route (the "best match so far")
5. Continue until a leaf is reached or no further branches match
6. Return the last recorded matching route (longest prefix match)

### 2.2 Routing Table to Forwarding Table Synchronization

```
Routing Protocols (OSPF, BGP, IS-IS, Static)
         |
         v
   Routing Table (RE)         <-- All routes, all preferences
   [radix tree, software]
         |
         | (active route selection)
         v
   Active Route Set            <-- Best route per prefix
         |
         | (kernel copy / PFE download)
         v
   Forwarding Table (PFE)     <-- Hardware lookup structure
   [ASIC-optimized trie]
```

The RE continuously synchronizes the forwarding table. When a route change occurs:
1. RE re-runs route selection for affected prefixes
2. New active routes are computed
3. Delta is pushed to the PFE via the internal RE-PFE link
4. PFE updates its forwarding ASIC entries
5. Forwarding converges once the PFE update completes (typically < 1ms for the PFE update itself)

### 2.3 Table Size and Scale

| Platform Tier  | Typical FIB Capacity     | RIB Capacity         |
|----------------|--------------------------|----------------------|
| Branch (SRX)   | 256K-512K IPv4 prefixes  | Memory-limited       |
| Edge (MX204)   | 2M IPv4 prefixes         | 16M+ routes in RIB  |
| Core (MX960)   | 4M+ IPv4 prefixes        | 32M+ routes in RIB  |

The RIB (routing table) is limited only by RE memory. The FIB (forwarding table) is limited by PFE ASIC memory (TCAM or algorithmic LPM engines).

## 3. VRF-Lite vs Full VRF with MPLS

### 3.1 VRF-Lite (virtual-router Instance Type)

VRF-Lite provides routing table isolation without MPLS or VPN signaling. Each `virtual-router` instance has its own `<instance>.inet.0` table, its own routing processes, and its own forwarding entries.

**Characteristics:**
- No route-distinguisher or route-target required
- No MPLS label allocation
- Traffic stays on the local device or uses IP forwarding between routers
- Requires a physical or logical interface in each VRF on every router along the path
- Suitable for simple multi-tenancy on a single device or small-scale segmentation

```
set routing-instances CUST-A instance-type virtual-router
set routing-instances CUST-A interface ge-0/0/1.100
set routing-instances CUST-A routing-options static route 0.0.0.0/0 next-hop 10.1.1.1
```

**Scaling limitation:** Every router in the path must have per-customer interfaces and per-customer routing table entries. For `N` customers across `M` routers, you need `N x M` interface/table configurations.

### 3.2 Full VRF with MPLS (vrf Instance Type)

Full VRF uses MPLS L3VPN (RFC 4364) to provide scalable multi-tenant routing. PE routers maintain per-customer VRFs; P routers in the core only switch MPLS labels with no customer awareness.

**Characteristics:**
- Route-distinguisher (RD) makes overlapping prefixes unique in BGP
- Route-target (RT) controls route import/export between VRFs
- MPLS labels provide data-plane isolation in the core
- MP-BGP (VPNv4/VPNv6) distributes VPN routes between PE routers
- P routers only need MPLS forwarding — no per-customer state
- Scales to thousands of VRFs across a provider network

```
set routing-instances VPN-B instance-type vrf
set routing-instances VPN-B interface ge-0/0/2.200
set routing-instances VPN-B route-distinguisher 65000:100
set routing-instances VPN-B vrf-target target:65000:100
set routing-instances VPN-B vrf-table-label
```

### 3.3 Comparison Matrix

| Aspect                  | VRF-Lite (virtual-router)        | Full VRF (MPLS L3VPN)            |
|-------------------------|----------------------------------|----------------------------------|
| MPLS required           | No                               | Yes                              |
| BGP signaling           | Not required                     | MP-BGP (VPNv4) between PEs      |
| Core router state       | Per-customer tables on every hop | Label switching only (no VRF)    |
| Overlapping addresses   | Isolated per instance            | RD makes unique in BGP           |
| Scalability             | Limited (N x M problem)          | Scales to thousands of VRFs      |
| Complexity              | Low                              | High (MPLS + MP-BGP + RT/RD)    |
| Typical use case        | Single-device segmentation       | Service provider, enterprise WAN |

## 4. Route Leaking Between Instances

Route leaking allows controlled sharing of routes between routing instances that are otherwise isolated. Junos provides several mechanisms.

### 4.1 RIB Groups (Routing Table Groups)

RIB groups copy routes from one routing table into another during the route installation phase. This is the most common route leaking method in Junos.

```
# Define a RIB group that copies inet.0 routes into CUST-A.inet.0
routing-options {
    rib-groups {
        LEAK-TO-CUST-A {
            import-rib [ inet.0 CUST-A.inet.0 ];
            import-policy LEAK-FILTER;
        }
    }
    interface-routes {
        rib-group inet LEAK-TO-CUST-A;
    }
}
```

**How it works:**
1. When a route is installed into the primary RIB (first in the `import-rib` list), it is also copied into secondary RIBs
2. The `import-policy` filters which routes are leaked
3. The primary RIB is always the first table in the list
4. Routes are copied at installation time, not at protocol receipt time

### 4.2 Logical Tunnel (lt-) Interfaces

A logical tunnel creates a virtual point-to-point link between two routing instances on the same device. Each instance sees the tunnel as a regular interface and can run routing protocols across it.

```
set interfaces lt-0/0/0 unit 0 encapsulation ethernet
set interfaces lt-0/0/0 unit 0 peer-unit 1
set interfaces lt-0/0/0 unit 0 family inet address 10.255.0.0/31
set interfaces lt-0/0/0 unit 1 encapsulation ethernet
set interfaces lt-0/0/0 unit 1 peer-unit 0
set interfaces lt-0/0/0 unit 1 family inet address 10.255.0.1/31

set routing-instances CUST-A interface lt-0/0/0.1
```

**Trade-off:** More flexible than RIB groups (supports dynamic routing across the tunnel) but consumes tunnel resources and adds forwarding overhead.

### 4.3 Instance Import/Export Policies (VRF Route Targets)

In MPLS VRF environments, route leaking between VRFs is controlled by route-target (RT) communities. A VRF imports routes tagged with matching RTs.

```
# VRF-A exports with target:65000:100
# VRF-B imports target:65000:100 to receive VRF-A's routes
set routing-instances VRF-B vrf-import IMPORT-FROM-A
policy-options {
    policy-statement IMPORT-FROM-A {
        term accept-A {
            from community RT-A;
            then accept;
        }
        term default {
            then reject;
        }
    }
    community RT-A members target:65000:100;
}
```

### 4.4 Route Leaking Decision Tree

1. **Same device, simple prefix sharing** --> RIB groups
2. **Same device, need routing protocol adjacency between instances** --> Logical tunnel (lt-)
3. **Across MPLS network, VRF to VRF** --> Route-target import/export
4. **Selective leaking with policy control** --> Any method + import-policy filtering

## 5. Convergence Analysis: Static vs Dynamic Routing

### 5.1 Failure Detection Time

| Method                          | Detection Time         | Mechanism                          |
|---------------------------------|------------------------|------------------------------------|
| Interface down (physical)       | < 50ms                 | Hardware/PHY signal loss           |
| Interface down (logical/VLAN)   | Depends on underlying  | May require protocol timeout       |
| Static route (no BFD)           | Infinite (no detection)| Requires manual intervention or interface down |
| OSPF (default timers)           | 40 seconds             | Dead interval (4x hello)          |
| OSPF (tuned timers)             | 1-4 seconds            | Reduced hello/dead intervals       |
| IS-IS (default timers)          | 30 seconds             | Hold time (3x hello)              |
| BGP (default hold timer)        | 90 seconds             | 3x keepalive (30s)                |
| BFD                             | 50-300ms               | Hardware-assisted bidirectional    |

### 5.2 Convergence Components

Total convergence time = Detection + Propagation + Computation + FIB Update

```
T_converge = T_detect + T_propagate + T_compute + T_fib_update

Static routing (interface up, remote failure):
  T_detect    = infinity (no mechanism)
  T_propagate = N/A
  T_compute   = N/A
  T_fib_update = N/A
  T_converge  = never (traffic blackholed)

OSPF (default timers, single area, 1000 prefixes):
  T_detect    = 40s (dead interval)
  T_propagate = < 100ms (LSA flooding)
  T_compute   = < 10ms (SPF on modern RE)
  T_fib_update = < 50ms (RE to PFE sync)
  T_converge  ~ 40.2s (dominated by detection)

OSPF + BFD (1000 prefixes):
  T_detect    = 150ms (BFD, 50ms x 3)
  T_propagate = < 100ms
  T_compute   = < 10ms
  T_fib_update = < 50ms
  T_converge  ~ 310ms
```

### 5.3 Static Routing Convergence Limitations

Static routes have no inherent failure detection for remote failures:

- **Interface failure (local):** Converges when the interface state changes; the route is withdrawn when the outgoing interface goes down. If using `qualified-next-hop`, the backup activates within seconds.
- **Remote failure (upstream link down):** The static route remains active because the local next-hop interface is still up. Traffic is blackholed until manual intervention or an out-of-band mechanism detects the failure.
- **Mitigation with BFD:** Static routes can reference a BFD session to detect remote path failures:
  ```
  set routing-options static route 10.0.0.0/16 next-hop 192.168.1.1 bfd-liveness-detection minimum-interval 300
  ```

### 5.4 Dynamic Routing Recovery Characteristics

**OSPF/IS-IS recovery profile:**
- Sub-second with BFD + SPF fast timers
- Prefix-independent convergence (PIC): pre-computed backup next-hops in PFE allow sub-50ms data-plane switchover independent of table size
- Loop-free alternate (LFA) provides immediate backup without waiting for SPF recomputation

**BGP recovery profile:**
- Slower by nature: 90s default hold timer, complex best-path recomputation
- Mitigated by BFD (150-300ms detection) and add-path/best-external for pre-computed alternatives
- Graceful restart allows a restarting router to preserve forwarding state while the control plane reconverges

### 5.5 Convergence Design Recommendations

| Network Scale     | Recommended Approach                                           |
|-------------------|----------------------------------------------------------------|
| Small/branch      | Static routes with BFD + qualified-next-hop backup             |
| Campus/enterprise | OSPF with BFD, tuned SPF timers (50/200/5000ms)               |
| Service provider  | IS-IS with BFD, TI-LFA for sub-50ms, BGP PIC edge             |
| Internet edge     | BGP with BFD, graceful restart, prefix-independent convergence |

## Prerequisites

- bgp
- ospf
- is-is
- mpls
- ipv4
- subnetting
- ecmp
