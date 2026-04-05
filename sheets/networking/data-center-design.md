# Data Center Design (Spine-Leaf, Clos, and DC Network Architecture)

Comprehensive guide to data center network topologies, from legacy 3-tier to modern spine-leaf Clos fabrics, covering traffic patterns, oversubscription, overlay design, physical layout, and reliability tiers.

## Traditional 3-Tier Architecture

### Layers

| Layer        | Role                                   | Typical Gear                        |
|--------------|----------------------------------------|-------------------------------------|
| Core         | High-speed backbone between aggregation blocks | Chassis switches (Nexus 7K, Cat 6500) |
| Aggregation  | Policy enforcement, L2/L3 boundary, STP root | Modular switches with large tables  |
| Access       | Server connectivity, VLAN assignment   | Top-of-rack or end-of-row switches  |

### Characteristics

- **Spanning Tree dependent:** STP blocks redundant links, wasting 50% of available bandwidth
- **North-south optimized:** Designed for client-server traffic (users to servers)
- **Oversubscription stacked:** Each layer adds oversubscription (access 3:1, aggregation 4:1, core 2:1)
- **L2/L3 boundary at aggregation:** VLANs confined within a pod, inter-pod traffic routed

### Limitations

```
Problem: STP blocks redundant paths
  Core1 --- Core2
    |   \X/   |       X = STP blocked link
    |   /X\   |       Only 50% of links active
  Agg1 --- Agg2
   |         |
 Access    Access
```

- Maximum ~4,096 VLANs (12-bit VLAN ID)
- VM mobility limited to L2 domain within a pod
- Convergence time: 30-50s (STP) or 1-3s (RSTP)
- Difficult to scale horizontally; requires chassis upgrades

## Clos Topology

### Multi-Stage Switching

- Invented by Charles Clos (1953) at Bell Labs for telephone networks
- **Non-blocking** when the middle stage has enough crosspoints
- A 3-stage Clos with parameters (m, n, r): m input links per ingress, n ingress switches, r middle-stage switches
- **Strictly non-blocking** when r >= 2m - 1

### Folded Clos (Fat Tree)

```
         Spine Switches (Stage 2)
        /    |    |    |    \
       /     |    |    |     \
    Leaf1  Leaf2 Leaf3 Leaf4 Leaf5
     |       |     |     |     |
   Servers Servers ...  ...  Servers

Every leaf connects to every spine
Every spine connects to every leaf
```

- Modern DC networks are **folded 3-stage Clos** (leaf = ingress/egress, spine = middle)
- Provides equal bandwidth between any pair of leaf switches
- Scales by adding more spines (bandwidth) or more leaves (ports)
- Typically uses fixed-form-factor switches (cheaper than chassis)

### 5-Stage Clos (Super-Spine)

```
          Super-Spine (Stage 3)
         /    |    |    |    \
      Pod1-Spine  Pod2-Spine  Pod3-Spine
       |  |  |     |  |  |     |  |  |
      Leaves      Leaves      Leaves
```

- Used when a single spine tier cannot provide enough ports for all leaves
- Each pod has its own spine layer; super-spines interconnect pods
- Common in hyperscale DCs with 100,000+ servers

## Spine-Leaf Architecture

### Design Rules

```bash
# Key design parameters
LEAF_DOWNLINKS=48          # Server-facing ports (e.g., 25G or 10G)
LEAF_UPLINKS=8             # Spine-facing ports (e.g., 100G)
SPINE_COUNT=8              # Number of spine switches
LEAF_COUNT=64              # Number of leaf switches (limited by spine port count)

# Maximum servers
MAX_SERVERS=$((LEAF_COUNT * LEAF_DOWNLINKS))
echo "Max servers: $MAX_SERVERS"       # 3,072

# Oversubscription ratio
DOWNLINK_BW=$((LEAF_DOWNLINKS * 25))   # 1,200 Gbps southbound
UPLINK_BW=$((LEAF_UPLINKS * 100))      # 800 Gbps northbound
echo "Oversubscription: ${DOWNLINK_BW}:${UPLINK_BW}"  # 1200:800 = 1.5:1
```

### Port Density Planning

| Leaf Model        | Downlinks     | Uplinks      | Max Spines | Max Leaves per Spine |
|-------------------|---------------|--------------|------------|----------------------|
| 48x25G + 8x100G  | 48 x 25 GbE  | 8 x 100 GbE | 8          | 48-64 per spine      |
| 48x10G + 6x40G   | 48 x 10 GbE  | 6 x 40 GbE  | 6          | 32-48 per spine      |
| 32x100G + 8x400G | 32 x 100 GbE | 8 x 400 GbE | 8          | 32-64 per spine      |
| 36x400G (split)   | 24 x 400 GbE | 12 x 400 GbE| 12         | 64+ per spine        |

### Oversubscription Ratios

| Ratio  | Meaning                          | Use Case                              |
|--------|----------------------------------|---------------------------------------|
| 1:1    | Non-blocking; full bisection BW  | HPC, storage clusters, financial      |
| 1.5:1  | Mild oversubscription            | General compute, web serving          |
| 2:1    | Moderate                         | Dev/test, office workloads            |
| 3:1    | High oversubscription            | Legacy, low-bandwidth workloads       |
| 4:1+   | Aggressive                       | Archival storage, backup targets      |

```bash
# Calculate oversubscription
leaf_down_bw=1200     # Total southbound bandwidth (Gbps)
leaf_up_bw=800        # Total northbound bandwidth (Gbps)
ratio=$(echo "scale=2; $leaf_down_bw / $leaf_up_bw" | bc)
echo "Oversubscription ratio: ${ratio}:1"
```

## East-West vs North-South Traffic

### Traffic Patterns

```
          Internet / WAN
              |
         [ Firewall ]        North-South:
              |               Client <-> Server
          [ Border ]          (vertical, in/out of DC)
              |
  +-----------+-----------+
  |           |           |
Leaf1      Leaf2       Leaf3
  |           |           |
Server A   Server B   Server C

Server A <-------> Server B    East-West:
Server B <-------> Server C    Server <-> Server
                               (horizontal, within DC)
```

### Key Metrics

| Pattern      | Direction    | Typical Share | Trend       |
|--------------|-------------|---------------|-------------|
| North-South  | In/out of DC | 20-30%       | Decreasing  |
| East-West    | Within DC    | 70-80%       | Increasing  |

- **Microservices** generate massive east-west traffic (service-to-service calls)
- **Storage replication** (3-way replica, erasure coding) is pure east-west
- **ML/AI training** workloads are 95%+ east-west (GPU-to-GPU, parameter servers)
- Spine-leaf is optimized for east-west; 3-tier was optimized for north-south

### Implications for Design

- East-west dominance requires **full mesh** between leaves via spines
- **Firewalls** must handle inline east-west inspection (or use micro-segmentation)
- **Load balancers** shift from north-south perimeter to east-west service mesh
- Monitoring must capture lateral traffic, not just border flows

## ECMP in Spine-Leaf

### How ECMP Distributes Traffic

```
Server A (on Leaf1) -> Server B (on Leaf3)

Leaf1 has equal-cost paths to Leaf3 via:
  Spine1 (cost 2)
  Spine2 (cost 2)
  Spine3 (cost 2)
  Spine4 (cost 2)

Hash(src_ip, dst_ip, src_port, dst_port, proto) mod 4 -> select spine
```

### Configuration (FRRouting / Linux)

```bash
# Enable ECMP in the underlay routing protocol (BGP example)
vtysh -c "configure terminal" \
      -c "router bgp 65001" \
      -c " maximum-paths 64" \
      -c " bestpath as-path multipath-relax"

# Linux kernel: use L3+L4 hash for better distribution
sudo sysctl -w net.ipv4.fib_multipath_hash_policy=1

# For encapsulated traffic (VXLAN underlay), hash on inner headers
sudo sysctl -w net.ipv4.fib_multipath_hash_policy=2

# Enable resilient hashing (kernel 5.17+) to minimize rehash on link failure
sudo sysctl -w net.ipv4.fib_multipath_hash_policy=1
# Plus nexthop group configuration with resilient buckets
```

### Resilient Hashing

```bash
# Without resilient hashing: any spine failure rehashes ALL flows
# With resilient hashing: only flows on the failed spine are redistributed

# Create resilient nexthop group (iproute2)
ip nexthop add id 1 via 10.0.0.1 dev eth0
ip nexthop add id 2 via 10.0.0.2 dev eth1
ip nexthop add id 3 via 10.0.0.3 dev eth2
ip nexthop add id 4 via 10.0.0.4 dev eth3

ip nexthop add id 100 group 1/2/3/4 type resilient buckets 128 idle_timer 60
ip route add 10.1.0.0/16 nhid 100
```

## VXLAN EVPN Overlay

### Architecture

```
  Overlay (VXLAN + EVPN control plane)
  ┌──────────────────────────────────────────┐
  │  VNI 10001    VNI 10002    VNI 10003     │
  │  (Tenant A)  (Tenant B)  (Tenant C)     │
  └──────────────────────────────────────────┘
         Encapsulated in UDP:4789
  ┌──────────────────────────────────────────┐
  │  Underlay: BGP + ECMP on spine-leaf      │
  │  Pure L3, unnumbered or /31 P2P links    │
  └──────────────────────────────────────────┘
```

### EVPN Route Types

| Type | Name                  | Purpose                                    |
|------|-----------------------|--------------------------------------------|
| 1    | Ethernet Auto-Discovery | Multi-homing, fast convergence             |
| 2    | MAC/IP Advertisement  | MAC and ARP learning (replaces flooding)   |
| 3    | Inclusive Multicast   | BUM traffic handling per VNI               |
| 4    | Ethernet Segment      | DF election for multi-homed hosts          |
| 5    | IP Prefix             | Inter-subnet routing (L3 VNI)             |

### FRRouting EVPN Configuration

```bash
# BGP EVPN on a leaf switch (FRRouting)
vtysh -c "configure terminal" \
  -c "router bgp 65001" \
  -c " bgp router-id 10.255.0.1" \
  -c " neighbor SPINE peer-group" \
  -c " neighbor SPINE remote-as external" \
  -c " neighbor eth0 interface peer-group SPINE" \
  -c " neighbor eth1 interface peer-group SPINE" \
  -c " address-family l2vpn evpn" \
  -c "  neighbor SPINE activate" \
  -c "  advertise-all-vni"

# VXLAN interface for VNI 10001
ip link add vxlan10001 type vxlan id 10001 local 10.255.0.1 dstport 4789 nolearning
ip link set vxlan10001 master br10001
ip link set vxlan10001 up
```

### Symmetric vs Asymmetric IRB

| Mode         | Routing            | MAC Table           | Scalability        |
|--------------|--------------------|---------------------|--------------------|
| Asymmetric   | Ingress leaf only  | All VNIs on all leaves | Poor for many VNIs |
| Symmetric    | Ingress + egress   | Only local VNIs     | Scales well        |

- Symmetric IRB uses a **transit L3 VNI** for inter-subnet traffic
- Preferred for large-scale deployments (100+ VNIs)

## DC Interconnect (DCI)

### Options

| Method             | Distance  | Bandwidth  | Latency     | Use Case             |
|--------------------|-----------|------------|-------------|----------------------|
| Dark fiber         | < 80 km   | 100G-400G+ | < 1 ms      | Campus/metro DCI     |
| DWDM               | < 3000 km | 100G-800G  | ~5 us/km    | Metro/long-haul      |
| VXLAN over WAN     | Any       | Variable   | Variable    | L2 stretch (caution) |
| EVPN multi-site    | Any       | Variable   | Variable    | Preferred for L2+L3  |
| OTV (Cisco)        | Any       | Variable   | Variable    | Legacy L2 extension  |
| SD-WAN             | Any       | Variable   | Variable    | Branch-to-DC         |

### EVPN Multi-Site DCI

```bash
# Border leaf / DCI gateway configuration concept
# Each site has its own spine-leaf fabric
# Border leaves peer via eBGP EVPN across the DCI link

# Site 1 border leaf (AS 65001)
vtysh -c "configure terminal" \
  -c "router bgp 65001" \
  -c " neighbor 192.168.100.2 remote-as 65002" \
  -c " address-family l2vpn evpn" \
  -c "  neighbor 192.168.100.2 activate" \
  -c "  advertise-all-vni"
```

## Micro-Segmentation

### Approaches

- **Network-based:** ACLs on leaf switches, applied per-port or per-VLAN
- **Host-based:** eBPF/iptables/nftables on hypervisor or bare-metal host
- **Overlay-based:** Security groups in VXLAN EVPN with policy enforcement at VTEP
- **Service mesh:** Mutual TLS between microservices (Istio, Linkerd)

```bash
# eBPF-based micro-segmentation example (Cilium)
# Enforce L3/L4/L7 policies without traditional firewall rules

# Allow only HTTP traffic between frontend and backend
cat <<'POLICY'
apiVersion: cilium.io/v2
kind: CiliumNetworkPolicy
metadata:
  name: frontend-to-backend
spec:
  endpointSelector:
    matchLabels:
      app: backend
  ingress:
  - fromEndpoints:
    - matchLabels:
        app: frontend
    toPorts:
    - ports:
      - port: "8080"
        protocol: TCP
POLICY
```

## Multi-Pod and Multi-Site Design

### Pod Design

```
              Super-Spine
          /       |        \
    Pod A Spines  Pod B Spines  Pod C Spines
     |  |  |      |  |  |       |  |  |
    Leaves       Leaves        Leaves
```

- A **pod** is a self-contained spine-leaf fabric (typically 1 spine tier + N leaves)
- Pods connect via super-spines for inter-pod traffic
- Each pod can be a separate failure domain
- Typical pod size: 16-64 leaves per pod

### Scaling Limits

```bash
# Pod sizing calculation
SPINES_PER_POD=4
SPINE_PORTS=64                 # 64-port 100G spine switch
LEAVES_PER_POD=$((SPINE_PORTS))  # One leaf per spine port = 64
SERVERS_PER_LEAF=48

echo "Servers per pod: $((LEAVES_PER_POD * SERVERS_PER_LEAF))"   # 3,072
echo "With 8 pods: $((8 * LEAVES_PER_POD * SERVERS_PER_LEAF))"   # 24,576
```

### Multi-Site Considerations

- **Stretched VLANs** across sites: avoid if possible (split-brain risk)
- **Active-active DCI:** Both sites serve traffic; requires anycast gateways
- **Active-passive DCI:** One site primary, second for DR; simpler but wastes capacity
- **DNS-based GSLB** preferred over L2 stretching for multi-site load balancing

## Physical Layout

### Switch Placement Strategies

| Strategy         | Description                              | Pros                          | Cons                          |
|------------------|------------------------------------------|-------------------------------|-------------------------------|
| Top-of-Rack (ToR)| One leaf switch per rack (most common)   | Short server cables, simple   | Many switches to manage       |
| End-of-Row (EoR) | Leaf switch at end of row, serves multiple racks | Fewer switches      | Longer cables, patch panels   |
| Middle-of-Row    | Switch in center rack of the row         | Balanced cable lengths        | Rare, inflexible              |

### Cabling

```bash
# Cable type selection by distance
# DAC (Direct Attach Copper): 1-5m, cheapest, lowest power
# AOC (Active Optical Cable): 5-30m, moderate cost, low power
# SR fiber (multimode OM3/OM4): 70-100m for 100G, standard in DC
# LR fiber (singlemode OS2): 2-10 km, for DCI or large campuses
# ER/ZR fiber: 40-80+ km, long-haul DCI

# Typical spine-leaf cabling
# ToR leaf to spine: 10-30m (DAC or AOC within a row, SR fiber across rows)
# Spine to super-spine: 30-100m (SR fiber)
# DCI: 1-80 km (LR/ER/ZR fiber or DWDM)
```

### Structured Cabling Rules

- **Label everything:** Both ends of every cable, both patch panel and switch port
- **Color code:** Use different colors for server, management, spine uplinks, DCI
- **Bend radius:** Respect minimum bend radius (fiber: 10x cable diameter, DAC: per spec)
- **Cable management:** Overhead trays or under-floor; never obstruct hot/cold aisle airflow
- **Spare capacity:** Plan 20-30% spare ports on leaves for growth

## Power and Cooling

### Power Budget

```bash
# Per-rack power calculation
SERVERS_PER_RACK=40
SERVER_WATTS=500              # Average server power draw
SWITCH_WATTS=350              # Leaf switch power
TOTAL_RACK_WATTS=$(( (SERVERS_PER_RACK * SERVER_WATTS) + SWITCH_WATTS ))
echo "Per-rack power: ${TOTAL_RACK_WATTS}W"    # 20,350W (~20.4 kW)

# Facility-level
RACKS=200
TOTAL_FACILITY_KW=$(( (TOTAL_RACK_WATTS * RACKS) / 1000 ))
echo "Facility power: ${TOTAL_FACILITY_KW} kW"  # 4,070 kW
```

### PUE (Power Usage Effectiveness)

| PUE   | Rating            | Cooling Method                     |
|-------|-------------------|------------------------------------|
| 2.0+  | Poor (legacy)     | Raised-floor CRAC, no containment  |
| 1.5   | Average           | Hot/cold aisle containment          |
| 1.2   | Good              | In-row cooling, efficient UPS       |
| 1.1   | Excellent         | Free cooling, liquid cooling        |
| 1.05  | Hyperscale best   | Evaporative + immersion cooling     |

### Cooling Strategies

- **Hot/cold aisle containment:** Physical barriers to prevent air mixing; standard best practice
- **In-row cooling:** Cooling units between racks; reduces distance from source to load
- **Rear-door heat exchangers:** Water-cooled doors on rack backs; removes heat at source
- **Direct liquid cooling:** Cold plates on CPUs/GPUs; essential for high-density AI/ML racks (30-100+ kW/rack)
- **Immersion cooling:** Servers submerged in dielectric fluid; emerging for extreme density

## Uptime Institute Tiers

### Tier Definitions

| Tier   | Availability  | Downtime/Year | Key Requirement                         |
|--------|--------------|---------------|------------------------------------------|
| Tier I | 99.671%      | 28.8 hours    | Single path, no redundancy               |
| Tier II| 99.741%      | 22.0 hours    | Redundant components (N+1 UPS, cooling)  |
| Tier III| 99.982%     | 1.6 hours     | Concurrently maintainable (dual paths)   |
| Tier IV| 99.995%      | 26.3 minutes  | Fault tolerant (2N or 2N+1 everything)   |

### Power Redundancy

```
Tier I:   Utility -> UPS -> PDU -> Rack
          Single path, single UPS

Tier III: Utility A -> UPS A -> PDU A -> Rack (Path A)
          Utility B -> UPS B -> PDU B -> Rack (Path B)
          Either path can be maintained without downtime

Tier IV:  Same as Tier III but fault-tolerant:
          Any single component failure causes no service impact
          Includes automatic transfer switches (ATS)
```

### Network Redundancy per Tier

- **Tier I:** Single network path, single ISP
- **Tier II:** Redundant switches/routers, but still single path
- **Tier III:** Dual network paths, dual ISPs, concurrent maintainability
- **Tier IV:** Full fault-tolerant networking, automated failover, no single point of failure

## Design Checklist

```bash
# Pre-deployment verification
# 1. Topology
# [ ] Spine-leaf fabric validated (correct port count and oversubscription)
# [ ] ECMP verified across all spines (check with traceroute from each leaf)
# [ ] Underlay BGP/OSPF adjacencies all UP

# 2. Overlay
# [ ] VXLAN EVPN control plane operational
# [ ] All VNIs advertised and learned (show bgp l2vpn evpn summary)
# [ ] Symmetric IRB for inter-VNI routing if needed

# 3. Physical
# [ ] All cables labeled and tested (OTDR for fiber)
# [ ] Optics matched to cable type (SR/LR/DAC)
# [ ] Power redundancy matches target tier

# 4. Monitoring
# [ ] Interface counters and errors polled (SNMP/Streaming Telemetry)
# [ ] BFD enabled on all underlay links
# [ ] Syslog forwarding configured
```

## Tips

- Start with a 2-spine design and scale to 4+ spines as needed; spine-leaf scales horizontally by adding switches, not upgrading chassis.
- Use **eBGP** on the underlay (one ASN per leaf, one per spine) for simplicity; this avoids OSPF area design and STP entirely.
- Set `maximum-paths 64` and `bestpath as-path multipath-relax` to enable ECMP across all spines.
- Keep oversubscription at 3:1 or better for general compute; 1:1 for storage and HPC clusters.
- Use **unnumbered interfaces** (IPv6 LLA + RFC 5549) on spine-leaf links to eliminate IP address management.
- For VXLAN EVPN, prefer **symmetric IRB** over asymmetric when you have more than 20-30 VNIs.
- Avoid stretching L2 across data centers; use EVPN multi-site with anycast gateways instead.
- DAC cables are cheaper and lower-power than optics for distances under 5m; use them for ToR-to-server connections.
- Plan for 25G server links minimum in new builds; 10G is end-of-life for new deployments.
- Size your management network separately from the data fabric; out-of-band management prevents lockouts during fabric outages.
- Test failover by pulling spine links under load and verifying ECMP reconvergence time (should be < 1s with BFD).
- Document every port mapping, VLAN/VNI assignment, and IP allocation in a source-controlled YAML or TOML file.

## See Also

- bgp, ospf, ecmp, vxlan, bfd, iptables, nftables, ethernet, subnetting

## References

- [RFC 7348 — VXLAN: A Framework for Overlaying Virtualized Layer 2 Networks over Layer 3 Networks](https://www.rfc-editor.org/rfc/rfc7348)
- [RFC 7432 — BGP MPLS-Based Ethernet VPN (EVPN)](https://www.rfc-editor.org/rfc/rfc7432)
- [RFC 8365 — A Network Virtualization Overlay Solution Using EVPN](https://www.rfc-editor.org/rfc/rfc8365)
- [Charles Clos — A Study of Non-Blocking Switching Networks (1953)](https://ieeexplore.ieee.org/document/6770468)
- [Uptime Institute — Tier Standard: Topology](https://uptimeinstitute.com/tiers)
- [Facebook/Meta — Building Express Backbone (Clos Fabric at Scale)](https://engineering.fb.com/2014/11/14/production-engineering/introducing-data-center-fabric-the-next-generation-facebook-data-center-network/)
- [Google Jupiter Rising — A Decade of Clos Topologies in Google Data Centers](https://research.google/pubs/pub43837/)
- [Arista Design Guide — Leaf-Spine Architecture](https://www.arista.com/en/solutions/design-guides)
- [Cisco VXLAN EVPN Multi-Site Design Guide](https://www.cisco.com/c/en/us/products/collateral/switches/nexus-9000-series-switches/guide-c07-734107.html)
- [Cumulus/NVIDIA EVPN Configuration Guide](https://docs.nvidia.com/networking-ethernet-software/)
- [ASHRAE TC 9.9 — Thermal Guidelines for Data Centers](https://www.ashrae.org/)
