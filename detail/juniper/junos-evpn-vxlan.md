# JunOS EVPN-VXLAN — Deep Dive Theory and Analysis

> In-depth exploration of JunOS EVPN-VXLAN implementation: ERB vs CRB architecture trade-offs, ESI multihoming mechanics (DF election, split-horizon, aliasing), MAC/IP learning and advertising flow, ARP suppression, EVPN-VXLAN scaling, and JunOS-specific optimizations. For JNCIE-SP level understanding.

## 1. EVPN-VXLAN Architecture in JunOS

### 1.1 Overlay/Underlay Separation

EVPN-VXLAN separates the network into two independent planes:

**Underlay (IP Fabric):**
- Pure L3 routed network between leaf and spine switches
- Provides reachability between VTEP loopbacks
- Typically eBGP (one AS per leaf), OSPF, or IS-IS
- ECMP load balancing across multiple spine paths
- No MPLS required — standard IP forwarding

**Overlay (EVPN Control Plane + VXLAN Data Plane):**
- EVPN provides MAC/IP learning, route advertisement, and multihoming via BGP
- VXLAN provides L2 frame encapsulation over the IP underlay
- VNI (VXLAN Network Identifier) is the 24-bit tenant identifier
- VTEP (VXLAN Tunnel Endpoint) is the leaf switch loopback

### 1.2 VXLAN Encapsulation Details

VXLAN adds 50 bytes of overhead to the original Ethernet frame:

```
Outer Ethernet Header (14 bytes)
Outer IP Header (20 bytes)
Outer UDP Header (8 bytes, dst port 4789)
VXLAN Header (8 bytes):
  - Flags (8 bits): bit 3 = VNI present
  - Reserved (24 bits)
  - VNI (24 bits): tenant identifier (0 - 16,777,215)
  - Reserved (8 bits)
Inner Ethernet Frame (original frame)
```

**MTU consideration:** The underlay must support at least 1550-byte frames (original 1500 + 50 VXLAN overhead). Jumbo frames (9000+) are recommended to avoid fragmentation.

**UDP source port:** Derived from hash of inner frame fields (src/dst MAC, VLAN, IP, etc.). This provides entropy for underlay ECMP across spine paths.

### 1.3 Routing Tables in EVPN-VXLAN

| Table                          | Purpose                                        |
|-------------------------------|-------------------------------------------------|
| `default-switch.evpn.0`       | EVPN routes for the default switch instance     |
| `<instance>.evpn.0`           | EVPN routes for named instance                  |
| `<vrf>.inet.0`                | L3 routes in tenant VRF                         |
| `<vrf>.evpn.0`                | Type 5 IP prefix routes for VRF                |

## 2. ERB vs CRB Architecture Comparison

### 2.1 Edge-Routed Bridging (ERB)

**Architecture:** Every leaf switch performs both L2 switching and L3 routing. Each leaf has IRB interfaces and a VRF for every tenant.

**Traffic flow (inter-subnet):**
1. Host A (192.168.100.10) on Leaf-1 sends to Host B (192.168.200.20) on Leaf-2
2. Leaf-1 (ingress): Routes from VLAN 100 to VLAN 200 locally via IRB
3. Leaf-1 encapsulates in VXLAN with L3 VNI (symmetric IRB) or destination L2 VNI (asymmetric)
4. Leaf-2 (egress): Decapsulates and delivers to Host B

**Advantages:**
- Optimal traffic path (routing at first hop, no tromboning)
- Distributed workload across all leaves
- No single point of failure for routing
- Scales horizontally with leaf count

**Disadvantages:**
- Every leaf must have every VNI/subnet configured (asymmetric IRB) or L3 VNI (symmetric IRB)
- Configuration consistency required across all leaves
- More complex leaf configuration
- Higher memory usage per leaf (full routing table)

### 2.2 Centrally-Routed Bridging (CRB)

**Architecture:** Only designated gateway nodes (border leaves or spines) perform L3 routing. Regular leaves perform L2 switching only.

**Traffic flow (inter-subnet):**
1. Host A (192.168.100.10) on Leaf-1 sends to Host B (192.168.200.20) on Leaf-2
2. Leaf-1 (L2 only): Does not route. Bridges to gateway via VXLAN L2 VNI 1000
3. Gateway: Receives frame, routes from VLAN 100 to VLAN 200 via IRB
4. Gateway: Sends to Leaf-2 via VXLAN L2 VNI 2000
5. Leaf-2: Delivers to Host B

**Advantages:**
- Simpler leaf configuration (L2 only)
- Fewer subnets/VNIs on leaf switches
- Easier policy enforcement at centralized gateway
- Smaller leaf forwarding tables

**Disadvantages:**
- Suboptimal traffic path (trombone through gateway for inter-subnet)
- Gateway becomes bottleneck and single point of failure (mitigated by gateway redundancy)
- Higher east-west latency for inter-subnet traffic
- Gateway must handle all inter-subnet bandwidth

### 2.3 Comparison Matrix

| Aspect                | ERB                          | CRB                          |
|-----------------------|------------------------------|------------------------------|
| Routing location      | Every leaf                   | Gateway only                 |
| Traffic efficiency    | Optimal (first-hop routing)  | Suboptimal (trombone)        |
| Leaf complexity       | Higher                       | Lower (L2 only)              |
| Scalability           | Better east-west             | Better for simple topologies |
| Failure domain        | Distributed                  | Concentrated at gateway      |
| Configuration burden  | All leaves need VRF/IRB      | Only gateways need VRF/IRB   |
| Use case              | General purpose DC           | Small/medium DC, legacy      |

### 2.4 Symmetric vs Asymmetric IRB in ERB

**Asymmetric IRB:**
- Ingress leaf routes AND bridges (both L2 VNIs must exist on ingress)
- Egress leaf only bridges (destination L2 VNI receives frame)
- All VNIs must exist on all leaves
- Simpler conceptually but does not scale well

**Symmetric IRB:**
- Ingress leaf routes into L3 VNI
- Egress leaf routes from L3 VNI into destination L2 VNI
- Each leaf only needs its local L2 VNIs plus the shared L3 VNI
- Scales much better — recommended for production

**Label/VNI usage comparison:**

| Model      | Ingress VXLAN VNI      | Egress VXLAN VNI     | Leaf VNI requirement       |
|------------|------------------------|----------------------|----------------------------|
| Asymmetric | Destination L2 VNI     | N/A (local bridge)   | ALL L2 VNIs everywhere     |
| Symmetric  | L3 VNI (tenant VRF)    | Destination L2 VNI   | Local L2 VNIs + L3 VNI     |

## 3. ESI Multihoming in JunOS

### 3.1 Ethernet Segment Identifier (ESI)

The ESI is a 10-byte identifier that uniquely identifies a multihomed Ethernet segment (the LAG connecting a CE to multiple PEs):

```
ESI format: XX:XX:XX:XX:XX:XX:XX:XX:XX:XX

Type 0: Manually configured (most common)
Type 1: Auto-derived from LACP (system-id + port-key)
Type 3: Auto-derived from system MAC + local discriminator
```

All PEs connected to the same multihomed CE must configure the same ESI value. This is how EVPN knows they share the same Ethernet segment.

### 3.2 EVPN Route Types for Multihoming

**Type 1 — Ethernet Auto-Discovery (per-ES and per-EVI):**

*Per-ES route (ES=ESI, ETag=MAX):*
- Advertised by each PE for each local ESI
- Used for fast mass withdrawal: when a PE loses connectivity to an ES, withdrawing this single route causes all remote PEs to immediately update forwarding for all MACs on that ES
- Convergence: sub-second (single BGP withdrawal triggers bulk update)

*Per-EVI route (ES=ESI, ETag=EVI-specific):*
- Advertised per EVPN instance per ESI
- Used for aliasing: tells remote PEs that this PE can reach the ES for this EVI
- Enables load balancing across all-active PEs

**Type 4 — Ethernet Segment:**
- Exchanged between PEs sharing the same ESI
- Contains the PE's IP address and ESI
- Used for Designated Forwarder (DF) election
- Only exchanged between PEs with matching ESI

### 3.3 Designated Forwarder (DF) Election

The DF is the PE responsible for forwarding BUM traffic to a multihomed CE on a given VLAN/VNI. Without DF election, all PEs would forward BUM, causing duplicates.

**Default algorithm (mod-based):**
1. All PEs sharing an ESI exchange Type 4 routes
2. PEs are sorted by IP address in ascending order
3. For each VLAN/VNI: `DF_index = VLAN_ID mod number_of_PEs`
4. The PE at that index becomes the DF for that VLAN

Example with 3 PEs (10.0.0.1, 10.0.0.2, 10.0.0.3):
```
VLAN 100: 100 mod 3 = 1 → PE 10.0.0.2 is DF
VLAN 101: 101 mod 3 = 2 → PE 10.0.0.3 is DF
VLAN 102: 102 mod 3 = 0 → PE 10.0.0.1 is DF
```

This distributes the BUM forwarding load across PEs.

**Preference-based DF election:** JunOS supports `preference-based` DF election where explicit preference values determine the DF (RFC 8584).

### 3.4 Split-Horizon for ESI

When a BUM frame is received from a remote PE via VXLAN, the local PE must determine whether to forward it to a local multihomed interface:

- If the remote PE is also connected to the same ESI, the frame was already delivered via the remote PE's direct connection
- The local PE's ESI split-horizon label prevents duplicate delivery
- EVPN carries an ESI label in Type 1 routes for this purpose

**Flow:**
1. PE1 receives BUM from CE on ESI-1
2. PE1 floods via VXLAN with ESI label to all remote PEs
3. PE2 (also connected to ESI-1) receives the VXLAN frame
4. PE2 sees the ESI label matches its local ESI-1
5. PE2 does NOT forward the frame to its local ESI-1 interface (split-horizon)
6. PE3 (not connected to ESI-1) receives and forwards normally

### 3.5 Aliasing and Load Balancing

**Problem:** Remote PE4 has learned MAC-A from PE1 (via Type 2 route). But MAC-A's CE is all-active multihomed to PE1 and PE2. Traffic to MAC-A always goes to PE1, wasting PE2's link.

**Solution — Aliasing:**
1. Both PE1 and PE2 advertise Type 1 per-EVI routes for the shared ESI
2. Remote PE4 sees Type 2 (MAC) from PE1 AND Type 1 (EAD) from both PE1 and PE2
3. PE4 installs ECMP next-hops: traffic to MAC-A can go to either PE1 or PE2
4. PE2 receives traffic for MAC-A, recognizes it belongs to local ESI, forwards to CE

This achieves true all-active load balancing without requiring MAC advertisement from every PE.

## 4. MAC/IP Learning and Advertising

### 4.1 Local Learning → Type 2 Advertisement

1. CE sends frame to leaf switch (source MAC: `aa:bb:cc:dd:ee:ff`, IP: `192.168.100.10`)
2. Leaf learns MAC on local interface (data-plane learning)
3. Leaf generates EVPN Type 2 route:
   - MAC address: `aa:bb:cc:dd:ee:ff`
   - IP address: `192.168.100.10` (MAC+IP binding)
   - Route distinguisher: per-instance RD
   - Route target: per-instance RT
   - VXLAN VNI: included in PMSI or extended community
4. Type 2 route advertised via BGP to all EVPN peers

### 4.2 Remote Learning (Control-Plane)

1. Remote leaf receives Type 2 route via BGP
2. Installs MAC entry in bridge domain MAC table
3. Next-hop is the originating VTEP IP
4. VXLAN VNI from the route determines the bridge domain
5. No flooding required — MAC is learned via control plane

**Advantage over VPLS:** In VPLS, unknown unicast is flooded and MAC is learned via data-plane only. EVPN pre-populates MAC tables via BGP, reducing flooding significantly.

### 4.3 MAC Mobility

When a host moves from one leaf to another:

1. New leaf learns the MAC on its local interface
2. New leaf advertises Type 2 route with incremented **MAC Mobility extended community** sequence number
3. Old leaf receives the updated Type 2 and withdraws its own
4. All remote leaves update their MAC tables to point to the new leaf
5. Sequence number prevents oscillation (highest sequence wins)

**Sticky MAC:** JunOS supports `static-mac` or `mac-pinning` to prevent MAC mobility for specific addresses (security measure against MAC spoofing).

### 4.4 MAC Mass Withdrawal via Type 1

When a PE loses all connectivity to a multihomed CE:

1. PE withdraws its Type 1 per-ES route (single BGP withdrawal)
2. All remote PEs immediately know that PE can no longer reach any MAC on that ESI
3. Remote PEs update forwarding: all MACs on that ESI now point only to the remaining PE(s)
4. Convergence is sub-second (one BGP withdrawal triggers bulk update)

Compare to VPLS: would require individual MAC flush for every MAC, much slower convergence.

## 5. ARP Suppression

### 5.1 Mechanism

ARP suppression reduces BUM traffic by answering ARP requests locally at the leaf:

1. Host A sends ARP request for Host B's MAC (broadcast)
2. Leaf receives ARP request
3. Leaf checks its EVPN MAC/IP database (populated via Type 2 routes)
4. If Host B's MAC is known: leaf generates ARP reply directly (proxy ARP)
5. ARP request is NOT flooded to the fabric

### 5.2 JunOS Configuration

```
routing-instances {
    EVPN-VS {
        protocols {
            evpn {
                arp-suppression;
            }
        }
    }
}
```

### 5.3 Impact Analysis

Without ARP suppression in a 1000-host fabric:
- Each host ARPs for ~50 destinations = 50,000 ARP broadcasts
- Each broadcast is replicated to every VTEP via ingress replication
- With 20 VTEPs: 50,000 x 20 = 1,000,000 encapsulated BUM frames

With ARP suppression:
- Most ARP requests answered locally (proxy ARP)
- Only the first ARP (before Type 2 is received) is flooded
- Steady-state BUM reduction: 90%+ in typical environments

## 6. EVPN-VXLAN Scaling

### 6.1 Control Plane Scaling

Key scaling dimensions:
- **BGP sessions:** With route reflectors, each leaf has 2 BGP sessions (to redundant RRs). Scales linearly with leaf count.
- **EVPN routes:** Each MAC generates one Type 2 route. 100K MACs = 100K Type 2 routes system-wide.
- **Type 3 routes:** One per VNI per VTEP. 100 VNIs x 100 VTEPs = 10,000 Type 3 routes.
- **RR memory:** Must hold all EVPN routes. Size proportional to total MACs x VTEPs.

### 6.2 Data Plane Scaling

Key scaling dimensions:
- **VTEP tunnels:** Each leaf maintains a VXLAN tunnel to every other leaf with shared VNIs. Max: N*(N-1)/2 tunnels in the fabric.
- **MAC table entries:** Platform-dependent ASIC limits (QFX5100: 288K, QFX10K: 1M+).
- **VNI count:** Platform-dependent (QFX5100: 4K VNIs, QFX10K: 16K VNIs).
- **Ingress replication:** Each BUM frame is replicated to N-1 VTEPs. With 100 VTEPs, replication fanout = 99.

### 6.3 Scaling Best Practices

1. **Use symmetric IRB:** Avoids the need for all VNIs on all leaves
2. **Deploy route reflectors:** Eliminates full-mesh BGP in the overlay
3. **Enable ARP suppression:** Reduces BUM traffic by 90%+
4. **Use assisted replication:** For large-scale fabrics, offload BUM replication to spine
5. **Implement MAC limiting:** Prevent MAC table exhaustion from misbehaving hosts
6. **RT constraint:** Only distribute EVPN routes to interested VTEPs

## 7. JunOS-Specific Optimizations

### 7.1 MAC-VRF Instance Type

JunOS introduced `mac-vrf` as a simplified instance type for EVPN-VXLAN:
- Replaces the more complex `virtual-switch` configuration
- Automatically manages bridge domains and VNI mappings
- Reduces configuration lines by ~40% compared to `virtual-switch`
- Supported on QFX and MX platforms

### 7.2 EVPN Proxy ARP/NDP

Beyond basic ARP suppression, JunOS supports proxy ARP/NDP that:
- Learns from ARP/ND snooping on local traffic
- Cross-references with EVPN Type 2 MAC+IP bindings
- Suppresses redundant ARP/ND across VNI boundaries
- Reduces inter-VNI BUM when hosts in different subnets resolve each other

### 7.3 Virtual Gateway Address (VGA)

JunOS virtual-gateway-address provides anycast gateway without requiring VRRP:
- All leaves share the same virtual IP and virtual MAC
- No primary/backup election needed
- Host always ARPs and receives the same gateway MAC regardless of leaf
- Seamless mobility — host does not need to re-ARP after moving

### 7.4 EVPN-VXLAN with MPLS Underlay

JunOS supports EVPN with MPLS encapsulation instead of VXLAN:
- Used in service provider networks with existing MPLS infrastructure
- Same EVPN control plane (BGP route types 1-5)
- MPLS labels instead of VNIs
- Can coexist with L3VPN and VPLS on the same PE

## See Also

- junos-l2vpn
- junos-l3vpn
- junos-mpls-advanced

## References

- RFC 7432 — BGP MPLS-Based Ethernet VPN
- RFC 9135 — Integrated Routing and Bridging in EVPN
- RFC 7348 — VXLAN: Virtual Extensible LAN
- RFC 8365 — A Framework for Ethernet-Tree Service over EVPN
- RFC 8584 — Framework for EVPN Designated Forwarder Election Extensibility
- RFC 9136 — IP Prefix Advertisement in EVPN
- Juniper TechLibrary: EVPN User Guide
- Juniper Validated Design: EVPN-VXLAN Fabric Architecture
